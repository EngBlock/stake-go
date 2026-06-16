package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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
}

func TestServerRegistersGatedMutationTools(t *testing.T) {
	client, err := stake.NewClient(stake.WithSessionToken("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	session := connectMCP(t, newMCPServer(staticAuth(client), serverConfig{
		EnableWatchlistMutations: true,
		EnableOrderCancel:        true,
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

func connectMCP(t *testing.T, server *mcp.Server) *mcp.ClientSession {
	t.Helper()

	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server Connect: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "stake-go-test", Version: "test"}, nil)
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
