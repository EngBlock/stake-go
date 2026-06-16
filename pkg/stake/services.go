package stake

// NYSEServices groups services backed by Stake's US market endpoints.
type NYSEServices struct {
	Equities     *NYSEEquitiesService
	Fundings     *NYSEFundingsService
	FX           *FXService
	Market       *NYSEMarketService
	Orders       *NYSEOrdersService
	Products     *NYSEProductsService
	Ratings      *RatingsService
	Statements   *StatementService
	Trades       *NYSETradesService
	Transactions *NYSETransactionsService
	Watchlists   *WatchlistService
}

func newNYSEServices(client *Client) *NYSEServices {
	return &NYSEServices{
		Equities:     &NYSEEquitiesService{client: client},
		Fundings:     &NYSEFundingsService{client: client},
		FX:           &FXService{client: client},
		Market:       &NYSEMarketService{client: client},
		Orders:       &NYSEOrdersService{client: client},
		Products:     &NYSEProductsService{client: client},
		Ratings:      &RatingsService{client: client},
		Statements:   &StatementService{client: client},
		Trades:       &NYSETradesService{client: client},
		Transactions: &NYSETransactionsService{client: client},
		Watchlists:   &WatchlistService{client: client, exchange: ExchangeNYSE},
	}
}

// ASXServices groups services backed by Stake's Australian market endpoints.
type ASXServices struct {
	Equities     *ASXEquitiesService
	Fundings     *ASXFundingsService
	Market       *ASXMarketService
	Orders       *ASXOrdersService
	Products     *ASXProductsService
	Trades       *ASXTradesService
	Transactions *ASXTransactionsService
	Watchlists   *WatchlistService
}

func newASXServices(client *Client) *ASXServices {
	return &ASXServices{
		Equities:     &ASXEquitiesService{client: client},
		Fundings:     &ASXFundingsService{client: client},
		Market:       &ASXMarketService{client: client},
		Orders:       &ASXOrdersService{client: client},
		Products:     &ASXProductsService{client: client},
		Trades:       &ASXTradesService{client: client},
		Transactions: &ASXTransactionsService{client: client},
		Watchlists:   &WatchlistService{client: client, exchange: ExchangeASX},
	}
}
