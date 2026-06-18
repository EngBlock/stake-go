package stake

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestLoginWithCredentialsSetsSessionTokenAndUser(t *testing.T) {
	var sawLogin bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/sessions/v2/createSession":
			if r.Method != http.MethodPost {
				t.Fatalf("login method = %s, want POST", r.Method)
			}
			var payload CredentialsLoginRequest
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode login payload: %v", err)
			}
			if payload.Username != "user@example.com" || payload.Password != "secret" {
				t.Fatalf("unexpected credentials payload: %+v", payload)
			}
			if payload.RememberMeDays != 30 || payload.PlatformType != defaultPlatformType {
				t.Fatalf("defaults were not applied: %+v", payload)
			}
			sawLogin = true
			writeJSON(t, w, map[string]string{"sessionKey": "session-123"})
		case "/api/user":
			if !sawLogin {
				t.Fatal("user endpoint called before login endpoint")
			}
			if got := r.Header.Get("Stake-Session-Token"); got != "session-123" {
				t.Fatalf("Stake-Session-Token = %q, want session-123", got)
			}
			writeJSON(t, w, map[string]any{
				"userId":           "user-1",
				"firstName":        "Ada",
				"lastName":         "Lovelace",
				"emailAddress":     "ada@example.com",
				"macStatus":        "OK",
				"accountType":      "INDIVIDUAL",
				"regionIdentifier": "AU",
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(
		WithBaseURL(server.URL),
		WithCredentials("user@example.com", "secret"),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	user, err := client.Login(context.Background())
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if user.ID != "user-1" || client.SessionToken() != "session-123" {
		t.Fatalf("unexpected login result: user=%+v token=%q", user, client.SessionToken())
	}
}

func TestAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte(`{"error":"short and stout"}`))
	}))
	defer server.Close()

	client, err := NewClient(WithBaseURL(server.URL), WithSessionToken("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	err = client.Get(context.Background(), NYSE.Users, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want APIError", err)
	}
	if apiErr.StatusCode != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", apiErr.StatusCode, http.StatusTeapot)
	}
}

func TestNYSEMarketIsOpenCaseInsensitive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/utils/marketStatus" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]any{
			"response": map[string]any{
				"status": map[string]any{"current": "OPEN"},
			},
		})
	}))
	defer server.Close()

	client, err := NewClient(WithBaseURL(server.URL), WithSessionToken("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	open, err := client.NYSE.Market.IsOpen(context.Background())
	if err != nil {
		t.Fatalf("IsOpen: %v", err)
	}
	if !open {
		t.Fatal("market should be open")
	}
}

func TestRatingsNoDataAndBlankFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/data/calendar/ratings" {
			t.Fatalf("path = %s", r.URL.Path)
		}

		if r.URL.Query().Get("tickers") == "UNKNOWN" {
			writeJSON(t, w, map[string]string{"message": "No data returned"})
			return
		}

		writeJSON(t, w, map[string]any{
			"ratings": []map[string]any{
				{
					"ticker":         "AAPL",
					"rating_current": "",
					"rating_prior":   "Buy",
					"pt_current":     "",
					"pt_prior":       "123.45",
				},
			},
		})
	}))
	defer server.Close()

	client, err := NewClient(WithBaseURL(server.URL), WithSessionToken("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	empty, err := client.NYSE.Ratings.List(context.Background(), RatingsRequest{Symbols: []string{"UNKNOWN"}})
	if err != nil {
		t.Fatalf("List unknown: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("unknown ratings len = %d, want 0", len(empty))
	}

	ratings, err := client.NYSE.Ratings.List(context.Background(), RatingsRequest{Symbols: []string{"AAPL"}})
	if err != nil {
		t.Fatalf("List AAPL: %v", err)
	}
	if len(ratings) != 1 {
		t.Fatalf("ratings len = %d, want 1", len(ratings))
	}
	if ratings[0].RatingCurrent != nil || ratings[0].PTCurrent != nil {
		t.Fatalf("blank current fields should be nil: %+v", ratings[0])
	}
	if ratings[0].RatingPrior == nil || *ratings[0].RatingPrior != "Buy" {
		t.Fatalf("rating_prior not decoded: %+v", ratings[0])
	}
	if ratings[0].PTPrior == nil || *ratings[0].PTPrior != 123.45 {
		t.Fatalf("pt_prior not decoded: %+v", ratings[0])
	}
}

func TestNYSEEquityPositionsDecodeFlexibleNumbers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/users/accounts/v2/equityPositions" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]any{
			"equityValue": "1234.56",
			"pricesOnly":  false,
			"equityPositions": []map[string]any{{
				"askPrice":               "101.25",
				"availableForTradingQty": "2",
				"avgPrice":               "95.50",
				"bidPrice":               100.75,
				"category":               "Stock",
				"costBasis":              "191.00",
				"dailyReturnValue":       "1.23",
				"encodedName":            "apple-inc-aapl",
				"instrumentID":           "instrument-1",
				"lastTrade":              "101.00",
				"mktPrice":               "101.00",
				"marketValue":            "202.00",
				"name":                   "Apple",
				"openQty":                "2",
				"period":                 "1D",
				"priorClose":             "99.77",
				"returnOnStock":          "11.00",
				"side":                   "B",
				"symbol":                 "AAPL",
				"unrealizedDayPLPercent": "0.61",
				"unrealizedDayPL":        "1.23",
				"unrealizedPL":           "11.00",
				"urlImage":               "",
				"yearlyReturnPercentage": "12.34",
				"yearlyReturnValue":      "22.00",
			}},
		})
	}))
	defer server.Close()

	client, err := NewClient(WithBaseURL(server.URL), WithSessionToken("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	positions, err := client.NYSE.Equities.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if positions.EquityValue != 1234.56 {
		t.Fatalf("equity value = %v, want 1234.56", positions.EquityValue)
	}
	if len(positions.EquityPositions) != 1 {
		t.Fatalf("position count = %d, want 1", len(positions.EquityPositions))
	}
	position := positions.EquityPositions[0]
	if position.Symbol != "AAPL" || position.UnrealizedDayPLPercent != 0.61 || position.MarketValue != 202.00 || position.AskPrice == nil || *position.AskPrice != 101.25 {
		t.Fatalf("unexpected position: %+v", position)
	}
}

func TestASXTransactionsQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/asx/orders/tradeActivity" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("size"); got != "25" {
			t.Fatalf("size = %q, want 25", got)
		}
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Fatalf("page = %q, want 2", got)
		}
		if got := r.URL.Query()["sort"]; len(got) != 1 || got[0] != "insertedAt,asc" {
			t.Fatalf("sort = %#v, want insertedAt,asc", got)
		}

		writeJSON(t, w, map[string]any{
			"items":      []map[string]any{{"instrumentCode": "CBA", "side": "BUY", "type": "LIMIT", "units": 1}},
			"hasNext":    false,
			"page":       2,
			"totalItems": 1,
		})
	}))
	defer server.Close()

	client, err := NewClient(WithBaseURL(server.URL), WithSessionToken("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	transactions, err := client.ASX.Transactions.List(context.Background(), ASXTransactionRecordRequest{
		Limit:  25,
		Offset: 2,
		Sort:   []ASXSort{{Attribute: "insertedAt", Direction: ASXSortAscending}},
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(transactions.Transactions) != 1 || transactions.Transactions[0].InstrumentID != "CBA" {
		t.Fatalf("unexpected transactions: %+v", transactions)
	}
}

func TestNYSEProductDecodesDecimalIntegerCounters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/products/searchProduct" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("symbol"); got != "AAPL" {
			t.Fatalf("symbol = %q, want AAPL", got)
		}

		writeJSON(t, w, map[string]any{
			"products": []map[string]any{{
				"id":                    "product-1",
				"symbol":                "AAPL",
				"description":           "Apple Inc.",
				"urlImage":              "",
				"name":                  "Apple",
				"dailyReturn":           0,
				"dailyReturnPercentage": 0,
				"lastTraded":            0,
				"monthlyReturn":         0,
				"popularity":            1967.0,
				"watched":               12.0,
				"news":                  3.0,
				"bought":                4.0,
				"viewed":                5.0,
				"productType":           "stocks",
				"encodedName":           "apple",
				"period":                "1D",
				"inceptionDate":         315532800000.0,
				"instrumentTags":        []any{},
				"childInstruments":      []any{},
			}},
		})
	}))
	defer server.Close()

	client, err := NewClient(WithBaseURL(server.URL), WithSessionToken("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	product, err := client.NYSE.Products.Get(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if product.Popularity.Int() != 1967 || product.Watched.Int() != 12 || product.News.Int() != 3 || product.Bought.Int() != 4 || product.Viewed.Int() != 5 {
		t.Fatalf("unexpected counters: %+v", product)
	}
	if product.InceptionDate.String() != "315532800000" {
		t.Fatalf("inception date = %q, want 315532800000", product.InceptionDate.String())
	}
}

func TestWatchlistAddFiltersExistingTickers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/us/instrument/watchlist/watchlist-1":
			writeJSON(t, w, map[string]any{
				"watchlistId": "watchlist-1",
				"name":        "Tech",
				"instruments": []map[string]any{{"symbol": "TSLA"}},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/us/instrument/watchlist/watchlist-1/items":
			var payload struct {
				Tickers []string `json:"tickers"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode update payload: %v", err)
			}
			if len(payload.Tickers) != 1 || payload.Tickers[0] != "MSFT" {
				t.Fatalf("tickers = %#v, want [MSFT]", payload.Tickers)
			}
			writeJSON(t, w, map[string]any{
				"watchlistId": "watchlist-1",
				"name":        "Tech",
				"instruments": []map[string]any{{"symbol": "TSLA"}, {"symbol": "MSFT"}},
			})
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(WithBaseURL(server.URL), WithSessionToken("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	watchlist, err := client.NYSE.Watchlists.Add(context.Background(), UpdateWatchlistRequest{
		ID:      "watchlist-1",
		Tickers: []string{"TSLA", "MSFT"},
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if len(watchlist.Instruments) != 2 {
		t.Fatalf("instrument len = %d, want 2", len(watchlist.Instruments))
	}
}

func TestASXMarketBuyFillsAskAndInstrumentCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/asx/instrument/singleQuote/CBA":
			writeJSON(t, w, map[string]any{"symbol": "CBA", "ask": "101.5000", "bid": "100.5000"})
		case r.Method == http.MethodPost && r.URL.Path == "/api/asx/instrument/view/CBA":
			writeJSON(t, w, map[string]string{"instrumentId": "I-CBA"})
		case r.Method == http.MethodPost && r.URL.Path == "/api/asx/orders":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode order payload: %v", err)
			}
			if payload["instrumentCode"] != "I-CBA" || payload["side"] != "BUY" || payload["type"] != "MARKET_TO_LIMIT" || payload["price"] != 101.5 {
				t.Fatalf("unexpected ASX order payload: %+v", payload)
			}
			writeJSON(t, w, map[string]any{
				"order": map[string]any{
					"id":              "order-1",
					"instrumentCode":  "I-CBA",
					"placedTimestamp": "2022-07-27T22:22:45.164059",
					"side":            "BUY",
					"type":            "MARKET_TO_LIMIT",
				},
			})
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(WithBaseURL(server.URL), WithSessionToken("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	order, err := client.ASX.Trades.Buy(context.Background(), ASXMarketBuyRequest{Symbol: "CBA", Units: 10})
	if err != nil {
		t.Fatalf("Buy: %v", err)
	}
	if order.OrderID != "order-1" {
		t.Fatalf("order id = %q, want order-1", order.OrderID)
	}
}

func TestNYSETradeAddsUserAndItemAndVerifies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/products/searchProduct":
			if got := r.URL.Query().Get("symbol"); got != "AAPL" {
				t.Fatalf("symbol query = %q, want AAPL", got)
			}
			writeJSON(t, w, map[string]any{"products": []map[string]any{{"id": "product-1", "symbol": "AAPL"}}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/purchaseorders/v2/quickBuy":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode trade payload: %v", err)
			}
			if _, ok := payload["symbol"]; ok {
				t.Fatalf("trade payload should not include symbol: %+v", payload)
			}
			if payload["userId"] != "user-1" || payload["itemId"] != "product-1" || payload["orderType"] != "market" || payload["amountCash"] != float64(100) {
				t.Fatalf("unexpected trade payload: %+v", payload)
			}
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

	client, err := NewClient(WithBaseURL(server.URL), WithSessionToken("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	client.User = &User{ID: "user-1"}

	trade, err := client.NYSE.Trades.Buy(context.Background(), NYSEMarketBuyRequest{Symbol: "AAPL", AmountCash: 100})
	if err != nil {
		t.Fatalf("Buy: %v", err)
	}
	if trade.DWOrderID != "order-1" {
		t.Fatalf("dw order id = %q, want order-1", trade.DWOrderID)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func TestResolveEndpointPreservesQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "orderAmount=100.5") {
			t.Fatalf("query = %q, want orderAmount=100.5", r.URL.RawQuery)
		}
		writeJSON(t, w, map[string]float64{"brokerageFee": 1})
	}))
	defer server.Close()

	client, err := NewClient(WithBaseURL(server.URL), WithSessionToken("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	brokerage, err := client.NYSE.Orders.Brokerage(context.Background(), 100.5)
	if err != nil {
		t.Fatalf("Brokerage: %v", err)
	}
	if brokerage.BrokerageFee == nil || *brokerage.BrokerageFee != 1 {
		t.Fatalf("unexpected brokerage: %+v", brokerage)
	}
}

// TestClientLoginIsRaceFreeUnderConcurrency fires concurrent Login and
// SessionToken calls against a shared *Client. It is meaningful only under
// the race detector (-race); a data race on the sessionToken/User fields
// would be reported as a WARNING: DATA RACE.
func TestClientLoginIsRaceFreeUnderConcurrency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/sessions/v2/createSession":
			writeJSON(t, w, map[string]string{"sessionKey": "tok"})
		case "/api/user":
			writeJSON(t, w, map[string]any{
				"userId":           "u1",
				"firstName":        "Ada",
				"lastName":         "L",
				"emailAddress":     "a@example.com",
				"macStatus":        "OK",
				"accountType":      "INDIVIDUAL",
				"regionIdentifier": "AU",
			})
		default:
			writeJSON(t, w, map[string]any{})
		}
	}))
	defer server.Close()

	client, err := NewClient(
		WithBaseURL(server.URL),
		WithCredentials("user@example.com", "secret"),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); _, _ = client.Login(context.Background()) }()
		go func() { defer wg.Done(); _ = client.SessionToken() }()
	}
	wg.Wait()
}
