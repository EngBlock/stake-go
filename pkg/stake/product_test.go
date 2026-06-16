package stake

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestNYSEProductGetWithQuoteMergesProductAndMarketData(t *testing.T) {
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
					"id":                    "product-1",
					"symbol":                "TQQQ",
					"description":           "ProShares UltraPro QQQ ETF",
					"category":              "ETF",
					"urlImage":              "",
					"name":                  "ProShares UltraPro QQQ ETF",
					"dailyReturn":           1.54,
					"dailyReturnPercentage": 2.03,
					"lastTraded":            77.55,
					"monthlyReturn":         0,
					"popularity":            1967,
					"watched":               12,
					"news":                  3,
					"bought":                4,
					"viewed":                5,
					"productType":           "etfs",
					"encodedName":           "proshares-ultrapro-qqq-etf",
					"period":                "1D",
					"instrumentTags":        []any{},
					"childInstruments":      []any{},
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
				"symbol":                             "TQQQ",
				"open":                               76.34,
				"high":                               78.36,
				"low":                                74.29,
				"priorClose":                         76.01,
				"close":                              77.52,
				"bid":                                82.59,
				"ask":                                82.61,
				"closeBid":                           77.49,
				"closeAsk":                           77.5,
				"lastTrade":                          77.52,
				"prePostMarketLastTrade":             82.61,
				"dailyReturnQuote":                   1.51,
				"dailyReturnPercentageQuote":         1.99,
				"prePostMarketDailyReturn":           5.09,
				"prePostMarketDailyReturnPercentage": 6.566,
				"tradingStatus":                      "TRADING_STATUS_V1_NORMAL",
				"marketStatus":                       "PREMARKET",
				"tradeTimestamp":                     "2026-06-13T12:01:49.247Z",
				"volume":                             4398852,
				"stakeInstrumentId":                  "instrument-1",
			}})
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client, err := NewClient(WithBaseURL(server.URL), WithSessionToken("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	product, err := client.NYSE.Products.GetWithQuote(context.Background(), "TQQQ")
	if err != nil {
		t.Fatalf("GetWithQuote: %v", err)
	}
	if !sawProduct.Load() || !sawQuote.Load() {
		t.Fatalf("called product=%v quote=%v, want both", sawProduct.Load(), sawQuote.Load())
	}
	if product.Symbol != "TQQQ" || product.Name != "ProShares UltraPro QQQ ETF" {
		t.Fatalf("unexpected product: %+v", product)
	}
	quote := product.MarketDataQuote
	if quote.Symbol != "TQQQ" || quote.MarketStatus != "PREMARKET" || quote.TradingStatus != "TRADING_STATUS_V1_NORMAL" {
		t.Fatalf("unexpected quote metadata: %+v", quote)
	}
	if quote.Bid == nil || quote.Bid.Float64() != 82.59 || quote.PrePostMarketLastTrade == nil || quote.PrePostMarketLastTrade.Float64() != 82.61 {
		t.Fatalf("unexpected quote prices: %+v", quote)
	}
	if quote.Volume == nil || quote.Volume.Int() != 4398852 || quote.TradeTimestamp == nil || quote.TradeTimestamp.IsZero() {
		t.Fatalf("unexpected quote timing/volume: %+v", quote)
	}
}

func TestNYSEProductGetWithQuoteReturnsProductWhenQuoteFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/products/searchProduct":
			writeJSON(t, w, map[string]any{
				"products": []map[string]any{{
					"id":               "product-1",
					"symbol":           "TQQQ",
					"description":      "ProShares UltraPro QQQ ETF",
					"urlImage":         "",
					"name":             "ProShares UltraPro QQQ ETF",
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
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":"quote unavailable"}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client, err := NewClient(WithBaseURL(server.URL), WithSessionToken("token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	product, err := client.NYSE.Products.GetWithQuote(context.Background(), "TQQQ")
	if err != nil {
		t.Fatalf("GetWithQuote: %v", err)
	}
	if product.Symbol != "TQQQ" {
		t.Fatalf("symbol = %q, want TQQQ", product.Symbol)
	}
	if product.MarketDataQuote.MarketStatus != "" {
		t.Fatalf("quote should be empty after quote failure: %+v", product.MarketDataQuote)
	}
}
