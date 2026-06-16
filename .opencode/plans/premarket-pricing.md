# Plan: Integrate NYSE Pre-Market Pricing into stake-go

## Objective

Merge the `NYSEProduct` (metadata) and `NYSEMarketDataQuote` (live pricing) endpoints into a single, unified response. Fire both requests in parallel, merge the results, and expose the combined data through the existing MCP tool.

## Background

Currently, `nyse.product.get` returns product metadata (description, category, yearly return, popularity) but lacks live market data. The Stake web UI calls an additional endpoint for pre-market/post-market data, bid/ask spreads, volume, and OHLC values.

### Existing Endpoint
```
GET https://api2.prd.hellostake.com/api/products/searchProduct?symbol=TQQQ&page=1&max=1
```

### New Endpoint (from HAR analysis)
```
POST https://api.prd.hellostake.com/us/pricing/quotes/marketData
Body: {"symbols": ["TQQQ"]}
```

### Response Fields Available (new endpoint)
- `open`, `high`, `low`, `close`, `priorClose`
- `bid`, `ask`, `closeBid`, `closeAsk`
- `lastTrade`, `prePostMarketLastTrade`
- `dailyReturnQuote`, `dailyReturnPercentageQuote`
- `prePostMarketDailyReturn`, `prePostMarketDailyReturnPercentage`
- `marketStatus` (e.g., "PREMARKET")
- `tradingStatus` (e.g., "TRADING_STATUS_V1_NORMAL")
- `tradeTimestamp` (ISO 8601)
- `volume` (int64)
- `stakeInstrumentId`

## Proposed Changes

### 1. Add Market Data Endpoint Constant

**File:** `pkg/stake/endpoints.go`

Add a new field to `NYSEEndpoints`:

```go
type NYSEEndpoints struct {
    // ... existing fields ...
    MarketDataQuote string
}
```

Add to `NYSE` variable initialization:
```go
NYSE = NYSEEndpoints{
    // ... existing fields ...
    MarketDataQuote: "https://api.prd.hellostake.com/us/pricing/quotes/marketData",
}
```

### 2. Define Market Data Struct

**File:** `pkg/stake/product.go` (or new `pkg/stake/pricing.go`)

Create a new struct for the market data response (from `us/pricing/quotes/marketData`):

```go
// NYSEMarketDataQuote is a real-time market data snapshot from the US pricing endpoint.
type NYSEMarketDataQuote struct {
    Open                           float64 `json:"open"`
    High                           float64 `json:"high"`
    Low                            float64 `json:"low"`
    PriorClose                     float64 `json:"priorClose"`
    Close                          float64 `json:"close"`
    Bid                            float64 `json:"bid"`
    Ask                            float64 `json:"ask"`
    CloseBid                       float64 `json:"closeBid"`
    CloseAsk                       float64 `json:"closeAsk"`
    LastTrade                      float64 `json:"lastTrade"`
    PrePostMarketLastTrade         float64 `json:"prePostMarketLastTrade"`
    DailyReturnQuote               float64 `json:"dailyReturnQuote"`
    DailyReturnPercentageQuote     float64 `json:"dailyReturnPercentageQuote"`
    PrePostMarketDailyReturn       float64 `json:"prePostMarketDailyReturn"`
    PrePostMarketDailyReturnPercentage float64 `json:"prePostMarketDailyReturnPercentage"`
    TradingStatus                  string  `json:"tradingStatus"`
    MarketStatus                   string  `json:"marketStatus"`
    TradeTimestamp                 time.Time `json:"tradeTimestamp"`
    Volume                         int64   `json:"volume"`
    StakeInstrumentID              string  `json:"stakeInstrumentId"`
    Symbol                         string  `json:"symbol"`
}
```

### 3. Define Merged Product Struct

**File:** `pkg/stake/product.go`

Create a new struct that combines `NYSEProduct` with market data:

```go
// NYSEProductWithQuote merges product metadata with live market data.
type NYSEProductWithQuote struct {
    // All existing NYSEProduct fields (embedded)
    NYSEProduct

    // Market data fields
    MarketDataQuote NYSEMarketDataQuote `json:"marketDataQuote"`
}
```

**Decision:** Use embedding rather than duplicating `NYSEProduct` fields to keep the struct flat and JSON-compatible.

### 4. Implement Parallel Fetching

**File:** `pkg/stake/product.go`

Add a new method to `NYSEProductsService`:

```go
// GetWithQuote returns a US-market product enriched with live market data.
// Both requests are fired in parallel. If either fails, the error is returned
// with the successful partial result still populated (unless both fail).
func (s *NYSEProductsService) GetWithQuote(ctx context.Context, symbol string) (*NYSEProductWithQuote, error) {
    type result struct {
        product *NYSEProduct
        quote   *NYSEMarketDataQuote
    }

    var wg sync.WaitGroup
    var productErr, quoteErr error
    var product *NYSEProduct
    var quote *NYSEMarketDataQuote

    wg.Add(2)

    go func() {
        defer wg.Done()
        product, productErr = s.Get(ctx, symbol)
    }()

    go func() {
        defer wg.Done()
        quote, quoteErr = s.getMarketDataQuote(ctx, symbol)
    }()

    wg.Wait()

    if productErr != nil && quoteErr != nil {
        return nil, fmt.Errorf("failed to fetch product and quote: product_err=%w, quote_err=%w", productErr, quoteErr)
    }

    merged := &NYSEProductWithQuote{}
    if product != nil {
        merged.NYSEProduct = *product
    }
    if quote != nil {
        merged.MarketDataQuote = *quote
    }

    return merged, nil
}

// getMarketDataQuote calls the market data endpoint for a single symbol.
func (s *NYSEProductsService) getMarketDataQuote(ctx context.Context, symbol string) (*NYSEMarketDataQuote, error) {
    body := struct {
        Symbols []string `json:"symbols"`
    }{
        Symbols: []string{symbol},
    }

    var response []NYSEMarketDataQuote
    if err := s.client.Post(ctx, NYSE.MarketDataQuote, body, &response); err != nil {
        return nil, err
    }
    if len(response) == 0 {
        return nil, nil
    }
    return &response[0], nil
}
```

### 5. Update MCP Tool

**File:** `cmd/mcp-server/tools.go`

Replace the existing `nyse.product.get` handler to use `GetWithQuote`:

```go
addStakeTool(server, auth, "nyse.product.get", "Get a US-market product by ticker symbol, including live market data and pre-market pricing.", func(ctx context.Context, client *stake.Client, args symbolInput) (any, error) {
    symbol, err := requireNonEmpty(args.Symbol, "symbol")
    if err != nil {
        return nil, err
    }
    product, err := client.NYSE.Products.GetWithQuote(ctx, symbol)
    return output("product", product), err
})
```

### 6. Add Endpoint Test

**File:** `pkg/stake/endpoints_test.go`

Add test for the new endpoint constant:
```go
"market_data_quote": {
    got:  NYSE.MarketDataQuote,
    want: "https://api.prd.hellostake.com/us/pricing/quotes/marketData",
},
```

### 7. Add MCP Tool Test

**File:** `cmd/mcp-server/tools_test.go`

Add a test that verifies the tool calls both endpoints and returns merged data:

```go
func TestProductGetToolCallsBothEndpoints(t *testing.T) {
    // Mock server that handles both:
    // GET /api/products/searchProduct?symbol=TQQQ&page=1&max=1
    // POST /us/pricing/quotes/marketData
    // Verify both are called and merged response is returned
}
```

## Error Handling Strategy

| Scenario | Behavior |
|----------|----------|
| Both succeed | Return full `NYSEProductWithQuote` |
| Product fails, quote succeeds | Return partial with market data only (product fields zero) |
| Product succeeds, quote fails | Return partial with product metadata only (market data fields zero) |
| Both fail | Return error |

**Rationale:** The tool description says "Get a US-market product by ticker symbol, including live market data." If market data is unavailable, the user still gets the product metadata. This is the least-surprising behavior.

## Files to Modify

1. `pkg/stake/endpoints.go` — Add `MarketDataQuote` constant
2. `pkg/stake/endpoints_test.go` — Test new constant
3. `pkg/stake/product.go` — Add `NYSEMarketDataQuote`, `NYSEProductWithQuote`, `GetWithQuote`, `getMarketDataQuote`
4. `pkg/stake/product_test.go` — *(new, if tests exist)* — Add tests for parallel fetch and merge logic
5. `cmd/mcp-server/tools.go` — Update `nyse.product.get` to use `GetWithQuote`
6. `cmd/mcp-server/tools_test.go` — Add test for merged endpoint

## Backward Compatibility

- **Client library users:** `GetWithQuote` is a new method; `Get` is unchanged.
- **MCP tool users:** The tool returns a richer JSON structure. Old consumers that only read `NYSEProduct` fields will still work. New consumers can read the `marketDataQuote` sub-object.
- **No breaking changes** to existing endpoints or tool signatures.

## Open Questions

1. **Should the MCP tool name change?** 
   - No — keep `nyse.product.get` to avoid breaking existing clients.
2. **Should we also add `getMarketDataQuote` to the `NYSEProductsService` as a public method?**
   - Yes — expose it as `GetMarketDataQuote` for callers who only want the quote.
3. **Should `NYSEMarketDataQuote` fields be optional (`omitempty`) or required?**
   - Optional with `omitempty` — some fields may be null in the API response.
4. **Should we also add this to the ASX product endpoint?**
   - ASX already has `outOfMarketPrice` in the `ASXProduct` struct. If pre-market data is available via a separate endpoint, investigate that separately. For now, scope is NYSE only.

## Implementation Order

1. Add endpoint constant and test
2. Add `NYSEMarketDataQuote` struct and `getMarketDataQuote` method
3. Add `NYSEProductWithQuote` and `GetWithQuote` parallel fetch logic
4. Update MCP tool
5. Add MCP tool test
6. Run tests: `go test ./pkg/stake/... ./cmd/mcp-server/...`

## Expected Result

After implementation, `nyse.product.get` for `TQQQ` returns:

```json
{
  "product": {
    "id": "63698af7-e71e-4424-9ce7-d34b70fbe260",
    "symbol": "TQQQ",
    "name": "ProShares UltraPro QQQ ETF",
    "description": "...",
    "category": "ETF",
    "dailyReturn": 1.54,
    "dailyReturnPercentage": 2.03,
    "lastTraded": 77.55,
    "yearlyReturnPercentage": 142,
    "yearlyReturnValue": 50.24,
    "marketDataQuote": {
      "open": 76.34,
      "high": 78.36,
      "low": 74.29,
      "priorClose": 76.01,
      "close": 77.52,
      "bid": 82.59,
      "ask": 82.61,
      "closeBid": 77.49,
      "closeAsk": 77.5,
      "lastTrade": 77.52,
      "prePostMarketLastTrade": 82.61,
      "dailyReturnQuote": 1.51,
      "dailyReturnPercentageQuote": 1.99,
      "prePostMarketDailyReturn": 5.09,
      "prePostMarketDailyReturnPercentage": 6.566,
      "tradingStatus": "TRADING_STATUS_V1_NORMAL",
      "marketStatus": "PREMARKET",
      "tradeTimestamp": "2026-06-13T12:01:49.247Z",
      "volume": 4398852,
      "stakeInstrumentId": "0e7de5d1-df74-482b-a297-c3c7dcbf036f",
      "symbol": "TQQQ"
    }
  }
}
```
