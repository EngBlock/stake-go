package stake

import (
	"fmt"
	"regexp"
)

const StakeURL = "https://api2.prd.hellostake.com/"

// NYSEEndpoints contains all known Stake NYSE API endpoints.
type NYSEEndpoints struct {
	StakeURL            string
	AccountBalance      string
	AccountTransactions string
	Brokerage           string
	CancelOrder         string
	CashAvailable       string
	CreateSession       string
	EquityPositions     string
	FundDetails         string
	MarketDataQuote     string
	MarketStatus        string
	Orders              string
	ProductsSuggestions string
	QuickBuy            string
	Quotes              string
	Rate                string
	Ratings             string
	SellOrders          string
	Symbol              string
	TransactionHistory  string
	TransactionDetails  string
	Transactions        string
	Users               string
	Watchlists          string
	CreateWatchlist     string
	ReadWatchlist       string
	UpdateWatchlist     string
	Statement           string
}

// ASXEndpoints contains all known Stake ASX API endpoints.
type ASXEndpoints struct {
	StakeURL             string
	Brokerage            string
	CashAvailable        string
	CancelOrder          string
	EquityPositions      string
	MarketStatus         string
	AggregatedDepth      string
	CourseOfSales        string
	Orders               string
	ProductsSuggestions  string
	Symbol               string
	TradeActivity        string
	Watchlists           string
	CreateWatchlist      string
	ReadWatchlist        string
	UpdateWatchlist      string
	InstrumentFromSymbol string
	Transactions         string
	Users                string
}

var (
	NYSE = NYSEEndpoints{
		StakeURL:            StakeURL,
		AccountBalance:      "https://api2.prd.hellostake.com/api/cma/getAccountBalance",
		AccountTransactions: "https://api2.prd.hellostake.com/api/users/accounts/accountTransactions",
		Brokerage:           "https://api2.prd.hellostake.com/api/orders/brokerage?orderAmount={orderAmount}",
		CancelOrder:         "https://api2.prd.hellostake.com/api/orders/cancelOrder/{orderId}",
		CashAvailable:       "https://api2.prd.hellostake.com/api/users/accounts/cashAvailableForWithdrawal",
		CreateSession:       "https://api2.prd.hellostake.com/api/sessions/v2/createSession",
		EquityPositions:     "https://api2.prd.hellostake.com/api/users/accounts/v2/equityPositions",
		FundDetails:         "https://api2.prd.hellostake.com/api/fund/details",
		MarketDataQuote:     "https://api.prd.hellostake.com/us/pricing/quotes/marketData",
		MarketStatus:        "https://api2.prd.hellostake.com/api/utils/marketStatus",
		Orders:              "https://api2.prd.hellostake.com/api/users/accounts/v2/orders",
		ProductsSuggestions: "https://api2.prd.hellostake.com/api/products/getProductSuggestions/{keyword}",
		QuickBuy:            "https://api2.prd.hellostake.com/api/purchaseorders/v2/quickBuy",
		Quotes:              "https://api2.prd.hellostake.com/api/quotes/marketData/{symbols}",
		Rate:                "https://api2.prd.hellostake.com/api/wallet/rate",
		Ratings:             "https://api2.prd.hellostake.com/api/data/calendar/ratings?tickers={symbols}&pageSize={limit}",
		SellOrders:          "https://api2.prd.hellostake.com/api/sellorders",
		Symbol:              "https://api2.prd.hellostake.com/api/products/searchProduct?symbol={symbol}&page=1&max=1",
		TransactionHistory:  "https://api2.prd.hellostake.com/api/users/accounts/transactionHistory",
		TransactionDetails:  "https://api2.prd.hellostake.com/api/users/accounts/transactionDetails?reference={reference}&referenceType={reference_type}",
		Transactions:        "https://api2.prd.hellostake.com/api/users/accounts/transactions",
		Users:               "https://api2.prd.hellostake.com/api/user",
		Watchlists:          "https://api2.prd.hellostake.com/us/instrument/watchlists",
		CreateWatchlist:     "https://api2.prd.hellostake.com/us/instrument/watchlist",
		ReadWatchlist:       "https://api2.prd.hellostake.com/us/instrument/watchlist/{watchlist_id}",
		UpdateWatchlist:     "https://api2.prd.hellostake.com/us/instrument/watchlist/{watchlist_id}/items",
		Statement:           "https://api2.prd.hellostake.com/api/data/fundamentals/{symbol}/statements?startDate={date}",
	}

	ASX = ASXEndpoints{
		StakeURL:             StakeURL,
		Brokerage:            "https://api2.prd.hellostake.com/api/asx/orders/brokerage?orderAmount={orderAmount}",
		CashAvailable:        "https://api2.prd.hellostake.com/api/asx/cash",
		CancelOrder:          "https://api2.prd.hellostake.com/api/asx/orders/{orderId}/cancel",
		EquityPositions:      "https://api2.prd.hellostake.com/api/asx/instrument/equityPositions",
		MarketStatus:         "https://api2.prd.hellostake.com/api/asx/instrument/quoteTwo/ASX",
		AggregatedDepth:      "https://api2.prd.hellostake.com/api/asx/instrument/aggregatedDepth/{symbol}?type=EQUITY",
		CourseOfSales:        "https://api2.prd.hellostake.com/api/asx/instrument/courseOfSales/{symbol}",
		Orders:               "https://api2.prd.hellostake.com/api/asx/orders",
		ProductsSuggestions:  "https://api2.prd.hellostake.com/api/asx/instrument/search?searchKey={keyword}",
		Symbol:               "https://api2.prd.hellostake.com/api/asx/instrument/singleQuote/{symbol}",
		TradeActivity:        "https://api2.prd.hellostake.com/api/asx/orders/tradeActivity",
		Watchlists:           "https://api2.prd.hellostake.com/api/asx/instrument/v2/watchlists",
		CreateWatchlist:      "https://api2.prd.hellostake.com/api/asx/instrument/v2/watchlist",
		ReadWatchlist:        "https://api2.prd.hellostake.com/api/asx/instrument/v2/watchlist/{watchlist_id}",
		UpdateWatchlist:      "https://api2.prd.hellostake.com/api/asx/instrument/v2/watchlist/{watchlist_id}/items",
		InstrumentFromSymbol: "https://api2.prd.hellostake.com/api/asx/instrument/view/{symbol}",
		Transactions:         "https://api2.prd.hellostake.com/api/asx/transactions",
		Users:                "https://api2.prd.hellostake.com/api/user",
	}
)

var endpointPlaceholderPattern = regexp.MustCompile(`\{([A-Za-z0-9_]+)\}`)

// FormatEndpoint replaces Python-style endpoint placeholders without URL encoding.
func FormatEndpoint(template string, values map[string]string) (string, error) {
	var missing string

	formatted := endpointPlaceholderPattern.ReplaceAllStringFunc(template, func(match string) string {
		name := match[1 : len(match)-1]
		value, ok := values[name]
		if !ok {
			if missing == "" {
				missing = name
			}
			return match
		}

		return value
	})

	if missing != "" {
		return "", fmt.Errorf("missing endpoint placeholder %q", missing)
	}

	return formatted, nil
}
