package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nathanbeddoewebdev/stake-go/pkg/secretsauce"
	"github.com/nathanbeddoewebdev/stake-go/pkg/stake"
)

func TestDefaultServerOmitsMutationTools(t *testing.T) {
	client, err := stake.NewClient(stake.WithSessionToken("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	session := connectMCP(t, newMCPServer(staticAuth(client), serverConfig{}))
	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	names := toolNames(tools.Tools)
	if !names["nyse.market.is_open"] {
		t.Fatal("read-only market tool was not registered")
	}
	if names["nyse.watchlists.create"] {
		t.Fatal("watchlist mutation tool should not be registered by default")
	}
	if names["nyse.orders.cancel"] {
		t.Fatal("order cancellation tool should not be registered by default")
	}
	if names["nyse.trades.market_buy"] {
		t.Fatal("trading tool should not be registered by default")
	}
}

func TestServerRegistersGatedMutationTools(t *testing.T) {
	client, err := stake.NewClient(stake.WithSessionToken("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	session := connectMCP(t, newMCPServer(staticAuth(client), serverConfig{
		EnableWatchlistMutations: true,
		EnableOrderCancel:        true,
		EnableTrading:            true,
	}))
	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	names := toolNames(tools.Tools)
	if !names["nyse.watchlists.create"] {
		t.Fatal("watchlist mutation tool was not registered")
	}
	if !names["nyse.orders.cancel"] {
		t.Fatal("order cancellation tool was not registered")
	}
	if !names["nyse.trades.market_buy"] {
		t.Fatal("NYSE trading tool was not registered")
	}
	if !names["asx.trades.limit_sell"] {
		t.Fatal("ASX trading tool was not registered")
	}
}

func TestTradingToolsHaveDestructiveAnnotations(t *testing.T) {
	client, err := stake.NewClient(stake.WithSessionToken("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	session := connectMCP(t, newMCPServer(staticAuth(client), serverConfig{EnableTrading: true}))
	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	tool := toolByName(tools.Tools, "nyse.trades.market_buy")
	if tool == nil {
		t.Fatal("trading tool was not registered")
	}
	if tool.Annotations == nil {
		t.Fatal("trading tool annotations were not set")
	}
	if tool.Annotations.ReadOnlyHint {
		t.Fatal("trading tool should not be marked read-only")
	}
	if tool.Annotations.DestructiveHint == nil || !*tool.Annotations.DestructiveHint {
		t.Fatal("trading tool should be marked destructive")
	}
	if tool.Annotations.IdempotentHint {
		t.Fatal("trading tool should not be marked idempotent")
	}
	if tool.Annotations.OpenWorldHint == nil || !*tool.Annotations.OpenWorldHint {
		t.Fatal("trading tool should be marked open-world")
	}
}

func TestCommandTransportStartsWithoutStakeLogin(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "run", ".")
	session, err := mcp.NewClient(&mcp.Implementation{Name: "stake-go-test", Version: "test"}, nil).Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() {
		_ = session.Close()
	})

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if !toolNames(tools.Tools)["me"] {
		t.Fatal("me tool was not registered")
	}
}

func TestMarketIsOpenToolCallsStakeClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/user":
			writeJSON(t, w, map[string]any{
				"userId":           "user-1",
				"firstName":        "Ada",
				"lastName":         "Lovelace",
				"emailAddress":     "ada@example.com",
				"macStatus":        "OK",
				"accountType":      "INDIVIDUAL",
				"regionIdentifier": "AU",
			})
		case "/api/utils/marketStatus":
			if got := r.Header.Get("Stake-Session-Token"); got != "token" {
				t.Fatalf("Stake-Session-Token = %q, want token", got)
			}
			writeJSON(t, w, map[string]any{
				"response": map[string]any{
					"status": map[string]any{"current": "OPEN"},
				},
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := stake.NewClient(stake.WithBaseURL(server.URL), stake.WithSessionToken("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := client.Login(context.Background()); err != nil {
		t.Fatalf("Login: %v", err)
	}

	session := connectMCP(t, newMCPServer(staticAuth(client), serverConfig{}))
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "nyse.market.is_open"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error content: %+v", result.Content)
	}

	var output struct {
		Open bool `json:"open"`
	}
	decodeStructuredContent(t, result.StructuredContent, &output)
	if !output.Open {
		t.Fatal("open = false, want true")
	}
}

func TestProductGetToolCallsBothNYSEEndpoints(t *testing.T) {
	var sawProduct atomic.Bool
	var sawQuote atomic.Bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/products/searchProduct":
			if got := r.URL.Query().Get("symbol"); got != "TQQQ" {
				t.Fatalf("symbol = %q, want TQQQ", got)
			}
			sawProduct.Store(true)
			writeJSON(t, w, map[string]any{
				"products": []map[string]any{{
					"id":               "product-1",
					"symbol":           "TQQQ",
					"description":      "ProShares UltraPro QQQ ETF",
					"category":         "ETF",
					"urlImage":         "",
					"name":             "ProShares UltraPro QQQ ETF",
					"lastTraded":       77.55,
					"popularity":       0,
					"watched":          0,
					"news":             0,
					"bought":           0,
					"viewed":           0,
					"productType":      "etfs",
					"encodedName":      "proshares-ultrapro-qqq-etf",
					"period":           "1D",
					"instrumentTags":   []any{},
					"childInstruments": []any{},
				}},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/us/pricing/quotes/marketData":
			var payload struct {
				Symbols []string `json:"symbols"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode quote payload: %v", err)
			}
			if len(payload.Symbols) != 1 || payload.Symbols[0] != "TQQQ" {
				t.Fatalf("symbols = %#v, want [TQQQ]", payload.Symbols)
			}
			sawQuote.Store(true)
			writeJSON(t, w, []map[string]any{{
				"symbol":                 "TQQQ",
				"bid":                    82.59,
				"ask":                    82.61,
				"prePostMarketLastTrade": 82.61,
				"marketStatus":           "PREMARKET",
				"tradingStatus":          "TRADING_STATUS_V1_NORMAL",
				"tradeTimestamp":         "2026-06-13T12:01:49.247Z",
				"volume":                 4398852,
				"stakeInstrumentId":      "instrument-1",
			}})
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client, err := stake.NewClient(stake.WithBaseURL(server.URL), stake.WithSessionToken("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	session := connectMCP(t, newMCPServer(staticAuth(client), serverConfig{}))
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "nyse.product.get",
		Arguments: map[string]any{"symbol": "TQQQ"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error content: %+v", result.Content)
	}
	if !sawProduct.Load() || !sawQuote.Load() {
		t.Fatalf("called product=%v quote=%v, want both", sawProduct.Load(), sawQuote.Load())
	}

	var output struct {
		Product struct {
			Symbol          string  `json:"symbol"`
			Name            string  `json:"name"`
			LastTraded      float64 `json:"lastTraded"`
			MarketDataQuote struct {
				Symbol                 string  `json:"symbol"`
				Bid                    float64 `json:"bid"`
				Ask                    float64 `json:"ask"`
				PrePostMarketLastTrade float64 `json:"prePostMarketLastTrade"`
				MarketStatus           string  `json:"marketStatus"`
				Volume                 int     `json:"volume"`
			} `json:"marketDataQuote"`
		} `json:"product"`
	}
	decodeStructuredContent(t, result.StructuredContent, &output)
	if output.Product.Symbol != "TQQQ" || output.Product.Name != "ProShares UltraPro QQQ ETF" || output.Product.LastTraded != 77.55 {
		t.Fatalf("unexpected product output: %+v", output.Product)
	}
	quote := output.Product.MarketDataQuote
	if quote.Symbol != "TQQQ" || quote.Bid != 82.59 || quote.Ask != 82.61 || quote.PrePostMarketLastTrade != 82.61 || quote.MarketStatus != "PREMARKET" || quote.Volume != 4398852 {
		t.Fatalf("unexpected quote output: %+v", quote)
	}
}

func TestOrderCancelToolCallsStakeClientWhenEnabled(t *testing.T) {
	var cancelled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/orders/cancelOrder/order-123" {
			t.Fatalf("path = %s, want /api/orders/cancelOrder/order-123", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %s, want DELETE", r.Method)
		}
		if got := r.Header.Get("Stake-Session-Token"); got != "token" {
			t.Fatalf("Stake-Session-Token = %q, want token", got)
		}
		cancelled = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := stake.NewClient(stake.WithBaseURL(server.URL), stake.WithSessionToken("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	session := connectMCP(t, newMCPServer(staticAuth(client), serverConfig{EnableOrderCancel: true}))
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "nyse.orders.cancel",
		Arguments: map[string]any{"orderId": "order-123"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error content: %+v", result.Content)
	}
	if !cancelled {
		t.Fatal("cancel endpoint was not called")
	}

	var output struct {
		Cancelled bool `json:"cancelled"`
	}
	decodeStructuredContent(t, result.StructuredContent, &output)
	if !output.Cancelled {
		t.Fatal("cancelled = false, want true")
	}
}

func TestNYSETradingToolConfirmsAndCallsStakeClient(t *testing.T) {
	var placed atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/user":
			writeJSON(t, w, map[string]any{
				"userId":           "user-1",
				"firstName":        "Ada",
				"lastName":         "Lovelace",
				"emailAddress":     "ada@example.com",
				"macStatus":        "OK",
				"accountType":      "INDIVIDUAL",
				"regionIdentifier": "AU",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/products/searchProduct":
			if got := r.URL.Query().Get("symbol"); got != "AAPL" {
				t.Fatalf("symbol = %q, want AAPL", got)
			}
			writeJSON(t, w, map[string]any{"products": []map[string]any{{"id": "product-1", "symbol": "AAPL"}}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/purchaseorders/v2/quickBuy":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode trade payload: %v", err)
			}
			if payload["userId"] != "user-1" || payload["itemId"] != "product-1" || payload["orderType"] != "market" || payload["amountCash"] != float64(100) {
				t.Fatalf("unexpected trade payload: %+v", payload)
			}
			placed.Store(true)
			writeJSON(t, w, []map[string]any{{
				"category":     "Stock",
				"dwOrderId":    "order-1",
				"encodedName":  "apple",
				"id":           "trade-1",
				"imageURL":     "",
				"insertedDate": "2024-01-02T03:04:05Z",
				"itemId":       "product-1",
				"name":         "Apple",
				"side":         "B",
				"symbol":       "AAPL",
				"updatedDate":  "2024-01-02T03:04:05Z",
			}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/users/accounts/transactions":
			writeJSON(t, w, map[string]any{"transactions": []map[string]any{{"orderId": "order-1", "updatedReason": "OK"}}})
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client, err := stake.NewClient(stake.WithBaseURL(server.URL), stake.WithSessionToken("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	var confirmationMessage string
	session := connectMCPWithClientOptions(t, newMCPServer(staticAuth(client), serverConfig{EnableTrading: true}), &mcp.ClientOptions{
		ElicitationHandler: func(ctx context.Context, req *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
			confirmationMessage = req.Params.Message
			return &mcp.ElicitResult{Action: "accept"}, nil
		},
	})
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "nyse.trades.market_buy",
		Arguments: map[string]any{"symbol": "AAPL", "amountCash": 100},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error content: %+v", result.Content)
	}
	if !placed.Load() {
		t.Fatal("trade endpoint was not called")
	}
	if !strings.Contains(confirmationMessage, "real-money Stake order") || !strings.Contains(confirmationMessage, "NYSE market buy AAPL") {
		t.Fatalf("unexpected confirmation message: %q", confirmationMessage)
	}

	var output struct {
		Trade struct {
			DWOrderID string `json:"dwOrderId"`
			Symbol    string `json:"symbol"`
		} `json:"trade"`
	}
	decodeStructuredContent(t, result.StructuredContent, &output)
	if output.Trade.DWOrderID != "order-1" || output.Trade.Symbol != "AAPL" {
		t.Fatalf("unexpected trade output: %+v", output.Trade)
	}
}

func TestTradingToolDeclineDoesNotSubmitOrder(t *testing.T) {
	var placed atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/user":
			writeJSON(t, w, map[string]any{"userId": "user-1"})
		case r.Method == http.MethodPost && r.URL.Path == "/api/sellorders":
			placed.Store(true)
			w.WriteHeader(http.StatusInternalServerError)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client, err := stake.NewClient(stake.WithBaseURL(server.URL), stake.WithSessionToken("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	session := connectMCPWithClientOptions(t, newMCPServer(staticAuth(client), serverConfig{EnableTrading: true}), &mcp.ClientOptions{
		ElicitationHandler: func(ctx context.Context, req *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
			return &mcp.ElicitResult{Action: "decline"}, nil
		},
	})
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "nyse.trades.market_sell",
		Arguments: map[string]any{"symbol": "AAPL", "quantity": 1},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("declined trade should return tool error")
	}
	if placed.Load() {
		t.Fatal("trade endpoint was called after decline")
	}
}

func TestTradingToolWithoutElicitationFailsClosed(t *testing.T) {
	var placed atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/user":
			writeJSON(t, w, map[string]any{"userId": "user-1"})
		case r.Method == http.MethodPost && r.URL.Path == "/api/asx/orders":
			placed.Store(true)
			w.WriteHeader(http.StatusInternalServerError)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client, err := stake.NewClient(stake.WithBaseURL(server.URL), stake.WithSessionToken("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	session := connectMCP(t, newMCPServer(staticAuth(client), serverConfig{EnableTrading: true}))
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "asx.trades.limit_sell",
		Arguments: map[string]any{"instrumentCode": "I-CBA", "units": 10, "price": 101.5},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("trade without client elicitation should return tool error")
	}
	if placed.Load() {
		t.Fatal("trade endpoint was called without confirmation")
	}
}

func TestTradingToolAutoSkipsConfirmation(t *testing.T) {
	var placed atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/user":
			writeJSON(t, w, map[string]any{"userId": "user-1"})
		case r.Method == http.MethodPost && r.URL.Path == "/api/asx/orders":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode order payload: %v", err)
			}
			if payload["instrumentCode"] != "I-CBA" || payload["side"] != "BUY" || payload["type"] != "LIMIT" || payload["price"] != 101.5 {
				t.Fatalf("unexpected ASX order payload: %+v", payload)
			}
			placed.Store(true)
			writeJSON(t, w, map[string]any{
				"order": map[string]any{
					"id":              "order-1",
					"instrumentCode":  "I-CBA",
					"placedTimestamp": "2022-07-27T22:22:45.164059",
					"side":            "BUY",
					"type":            "LIMIT",
				},
			})
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client, err := stake.NewClient(stake.WithBaseURL(server.URL), stake.WithSessionToken("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	session := connectMCP(t, newMCPServer(staticAuth(client), serverConfig{EnableTrading: true, AutoConfirmWrites: true}))
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "asx.trades.limit_buy",
		Arguments: map[string]any{"instrumentCode": "I-CBA", "units": 10, "price": 101.5},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error content: %+v", result.Content)
	}
	if !placed.Load() {
		t.Fatal("trade endpoint was not called")
	}

	var output struct {
		Order struct {
			OrderID        string `json:"id"`
			InstrumentCode string `json:"instrumentCode"`
		} `json:"order"`
	}
	decodeStructuredContent(t, result.StructuredContent, &output)
	if output.Order.OrderID != "order-1" || output.Order.InstrumentCode != "I-CBA" {
		t.Fatalf("unexpected order output: %+v", output.Order)
	}
}

func TestASXMarketTradeInstrumentCodeOnlyRequiresPrice(t *testing.T) {
	client, err := stake.NewClient(stake.WithSessionToken("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	session := connectMCP(t, newMCPServer(staticAuth(client), serverConfig{EnableTrading: true, AutoConfirmWrites: true}))
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "asx.trades.market_buy",
		Arguments: map[string]any{"instrumentCode": "I-CBA", "units": 10},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("instrumentCode-only ASX market trade without price should return tool error")
	}
}

func TestTradingToolRefreshesStaleTokenBeforeConfirmation(t *testing.T) {
	var refreshed atomic.Bool
	var placed atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/user":
			switch r.Header.Get("Stake-Session-Token") {
			case "stale":
				w.WriteHeader(http.StatusUnauthorized)
			case "fresh":
				writeJSON(t, w, map[string]any{"userId": "user-1"})
			default:
				t.Fatalf("unexpected token %q", r.Header.Get("Stake-Session-Token"))
			}
		case r.Method == http.MethodPost && r.URL.Path == "/api/sessions/v2/createSession":
			refreshed.Store(true)
			writeJSON(t, w, map[string]string{"sessionKey": "fresh"})
		case r.Method == http.MethodPost && r.URL.Path == "/api/asx/orders":
			if got := r.Header.Get("Stake-Session-Token"); got != "fresh" {
				t.Fatalf("Stake-Session-Token = %q, want fresh", got)
			}
			placed.Store(true)
			writeJSON(t, w, map[string]any{
				"order": map[string]any{
					"id":              "order-1",
					"instrumentCode":  "I-CBA",
					"placedTimestamp": "2022-07-27T22:22:45.164059",
					"side":            "BUY",
					"type":            "LIMIT",
				},
			})
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	auth, err := newStakeAuth(stakeAuthConfig{
		BaseURL:           server.URL,
		DisableTokenCache: true,
		Token:             "stale",
		Username:          secretsauce.Source{Value: "ada@example.com"},
		Password:          secretsauce.Source{Value: "password"},
	})
	if err != nil {
		t.Fatalf("newStakeAuth: %v", err)
	}

	session := connectMCPWithClientOptions(t, newMCPServer(auth, serverConfig{EnableTrading: true}), &mcp.ClientOptions{
		ElicitationHandler: func(ctx context.Context, req *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
			return &mcp.ElicitResult{Action: "accept"}, nil
		},
	})
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "asx.trades.limit_buy",
		Arguments: map[string]any{"instrumentCode": "I-CBA", "units": 10, "price": 101.5},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error content: %+v", result.Content)
	}
	if !refreshed.Load() {
		t.Fatal("stale token was not refreshed")
	}
	if !placed.Load() {
		t.Fatal("trade endpoint was not called")
	}
}

func TestTradingToolDoesNotRetryOrderPostAfterUnauthorized(t *testing.T) {
	var orderPosts atomic.Int32
	var refreshed atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/user":
			writeJSON(t, w, map[string]any{"userId": "user-1"})
		case r.Method == http.MethodPost && r.URL.Path == "/api/sessions/v2/createSession":
			refreshed.Store(true)
			writeJSON(t, w, map[string]string{"sessionKey": "fresh"})
		case r.Method == http.MethodPost && r.URL.Path == "/api/asx/orders":
			orderPosts.Add(1)
			w.WriteHeader(http.StatusUnauthorized)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	auth, err := newStakeAuth(stakeAuthConfig{
		BaseURL:           server.URL,
		DisableTokenCache: true,
		Token:             "token",
		Username:          secretsauce.Source{Value: "ada@example.com"},
		Password:          secretsauce.Source{Value: "password"},
	})
	if err != nil {
		t.Fatalf("newStakeAuth: %v", err)
	}

	session := connectMCP(t, newMCPServer(auth, serverConfig{EnableTrading: true, AutoConfirmWrites: true}))
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "asx.trades.limit_buy",
		Arguments: map[string]any{"instrumentCode": "I-CBA", "units": 10, "price": 101.5},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatal("unauthorized order POST should return tool error")
	}
	if got := orderPosts.Load(); got != 1 {
		t.Fatalf("order POST count = %d, want 1", got)
	}
	if refreshed.Load() {
		t.Fatal("order POST 401 should not refresh and retry the trading tool")
	}
}

func connectMCP(t *testing.T, server *mcp.Server) *mcp.ClientSession {
	return connectMCPWithClientOptions(t, server, nil)
}

func connectMCPWithClientOptions(t *testing.T, server *mcp.Server, options *mcp.ClientOptions) *mcp.ClientSession {
	t.Helper()

	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server Connect: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "stake-go-test", Version: "test"}, options)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client Connect: %v", err)
	}

	t.Cleanup(func() {
		_ = clientSession.Close()
		_ = serverSession.Wait()
	})
	return clientSession
}

func staticAuth(client *stake.Client) *stakeAuth {
	return &stakeAuth{client: client, disableTokenCache: true}
}

func toolNames(tools []*mcp.Tool) map[string]bool {
	names := make(map[string]bool, len(tools))
	for _, tool := range tools {
		names[tool.Name] = true
	}
	return names
}

func toolByName(tools []*mcp.Tool, name string) *mcp.Tool {
	for _, tool := range tools {
		if tool.Name == name {
			return tool
		}
	}
	return nil
}

func decodeStructuredContent(t *testing.T, content any, out any) {
	t.Helper()

	data, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("decode structured content: %v", err)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("write JSON: %v", err)
	}
}
