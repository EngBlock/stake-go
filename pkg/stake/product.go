package stake

import (
	"context"
	"fmt"
	"net/http"
	"sync"
)

// ProductSearchByName searches products by name, description, or symbol-like keyword.
type ProductSearchByName struct {
	Keyword string `json:"keyword"`
}

// NYSEInstrument is a US-market instrument suggestion.
type NYSEInstrument struct {
	EncodedName  string `json:"encodedName,omitempty"`
	ImageURL     string `json:"imageUrl,omitempty"`
	InstrumentID string `json:"instrumentId"`
	Name         string `json:"name"`
	Symbol       string `json:"symbol"`
}

// NYSEProduct is a US-market product.
type NYSEProduct struct {
	ID                     string           `json:"id"`
	InstrumentTypeID       string           `json:"instrumentTypeID,omitempty"`
	Symbol                 string           `json:"symbol"`
	Description            string           `json:"description"`
	Category               string           `json:"category,omitempty"`
	CurrencyID             string           `json:"currencyID,omitempty"`
	URLImage               string           `json:"urlImage"`
	Sector                 string           `json:"sector,omitempty"`
	ParentID               string           `json:"parentID,omitempty"`
	Name                   string           `json:"name"`
	DailyReturn            float64          `json:"dailyReturn"`
	DailyReturnPercentage  float64          `json:"dailyReturnPercentage"`
	LastTraded             float64          `json:"lastTraded"`
	MonthlyReturn          float64          `json:"monthlyReturn"`
	YearlyReturnPercentage *FlexibleFloat64 `json:"yearlyReturnPercentage,omitempty"`
	YearlyReturnValue      *FlexibleFloat64 `json:"yearlyReturnValue,omitempty"`
	Popularity             FlexibleInt      `json:"popularity"`
	Watched                FlexibleInt      `json:"watched"`
	News                   FlexibleInt      `json:"news"`
	Bought                 FlexibleInt      `json:"bought"`
	Viewed                 FlexibleInt      `json:"viewed"`
	ProductType            string           `json:"productType"`
	TradeStatus            *FlexibleInt     `json:"tradeStatus,omitempty"`
	EncodedName            string           `json:"encodedName"`
	Period                 string           `json:"period"`
	InceptionDate          FlexibleString   `json:"inceptionDate,omitempty"`
	InstrumentTags         []any            `json:"instrumentTags"`
	ChildInstruments       []NYSEInstrument `json:"childInstruments"`
}

// NYSEMarketDataQuote is a real-time US-market price snapshot.
type NYSEMarketDataQuote struct {
	Open                               *FlexibleFloat64 `json:"open,omitempty"`
	High                               *FlexibleFloat64 `json:"high,omitempty"`
	Low                                *FlexibleFloat64 `json:"low,omitempty"`
	PriorClose                         *FlexibleFloat64 `json:"priorClose,omitempty"`
	Close                              *FlexibleFloat64 `json:"close,omitempty"`
	Bid                                *FlexibleFloat64 `json:"bid,omitempty"`
	Ask                                *FlexibleFloat64 `json:"ask,omitempty"`
	CloseBid                           *FlexibleFloat64 `json:"closeBid,omitempty"`
	CloseAsk                           *FlexibleFloat64 `json:"closeAsk,omitempty"`
	LastTrade                          *FlexibleFloat64 `json:"lastTrade,omitempty"`
	PrePostMarketLastTrade             *FlexibleFloat64 `json:"prePostMarketLastTrade,omitempty"`
	DailyReturnQuote                   *FlexibleFloat64 `json:"dailyReturnQuote,omitempty"`
	DailyReturnPercentageQuote         *FlexibleFloat64 `json:"dailyReturnPercentageQuote,omitempty"`
	PrePostMarketDailyReturn           *FlexibleFloat64 `json:"prePostMarketDailyReturn,omitempty"`
	PrePostMarketDailyReturnPercentage *FlexibleFloat64 `json:"prePostMarketDailyReturnPercentage,omitempty"`
	TradingStatus                      string           `json:"tradingStatus,omitempty"`
	MarketStatus                       string           `json:"marketStatus,omitempty"`
	TradeTimestamp                     *FlexibleTime    `json:"tradeTimestamp,omitempty"`
	Volume                             *FlexibleInt     `json:"volume,omitempty"`
	StakeInstrumentID                  string           `json:"stakeInstrumentId,omitempty"`
	Symbol                             string           `json:"symbol,omitempty"`
}

// NYSEProductWithQuote merges product metadata with live market data.
type NYSEProductWithQuote struct {
	NYSEProduct
	MarketDataQuote NYSEMarketDataQuote `json:"marketDataQuote"`
}

// NYSEProductsService reads US-market products and instruments.
type NYSEProductsService struct {
	client *Client
}

// Get returns a US-market product for a symbol. A nil product means Stake returned no match.
func (s *NYSEProductsService) Get(ctx context.Context, symbol string) (*NYSEProduct, error) {
	endpoint, err := FormatEndpoint(NYSE.Symbol, map[string]string{"symbol": symbol})
	if err != nil {
		return nil, err
	}

	var response struct {
		Products []NYSEProduct `json:"products"`
	}
	if err := s.client.do(ctx, http.MethodGet, endpoint, nil, &response); err != nil {
		return nil, err
	}
	if len(response.Products) == 0 {
		return nil, nil
	}
	return &response.Products[0], nil
}

// GetMarketDataQuote returns a real-time US-market price snapshot for a symbol.
func (s *NYSEProductsService) GetMarketDataQuote(ctx context.Context, symbol string) (*NYSEMarketDataQuote, error) {
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

// GetWithQuote returns a US-market product enriched with live market data.
func (s *NYSEProductsService) GetWithQuote(ctx context.Context, symbol string) (*NYSEProductWithQuote, error) {
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
		quote, quoteErr = s.GetMarketDataQuote(ctx, symbol)
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

// Search returns US-market instrument suggestions for a keyword.
func (s *NYSEProductsService) Search(ctx context.Context, request ProductSearchByName) ([]NYSEInstrument, error) {
	endpoint, err := FormatEndpoint(NYSE.ProductsSuggestions, map[string]string{"keyword": request.Keyword})
	if err != nil {
		return nil, err
	}

	var response struct {
		Instruments []NYSEInstrument `json:"instruments"`
	}
	if err := s.client.do(ctx, http.MethodGet, endpoint, nil, &response); err != nil {
		return nil, err
	}
	return response.Instruments, nil
}

// ProductFromInstrument returns the product for a US-market instrument suggestion.
func (s *NYSEProductsService) ProductFromInstrument(ctx context.Context, instrument NYSEInstrument) (*NYSEProduct, error) {
	return s.Get(ctx, instrument.Symbol)
}

// ASXInstrument is an Australian-market instrument suggestion.
type ASXInstrument struct {
	InstrumentID       string `json:"instrumentId"`
	Symbol             string `json:"symbol"`
	Name               string `json:"name,omitempty"`
	Type               string `json:"type"`
	RecentAnnouncement *bool  `json:"recentAnnouncement,omitempty"`
	Sensitive          *bool  `json:"sensitive,omitempty"`
}

// ASXProduct is an Australian-market quote/product response.
type ASXProduct struct {
	Symbol              string           `json:"symbol,omitempty"`
	OutOfMarketQuantity *FlexibleInt     `json:"outOfMarketQuantity,omitempty"`
	OutOfMarketSurplus  *FlexibleInt     `json:"outOfMarketSurplus,omitempty"`
	MarketStatus        string           `json:"marketStatus,omitempty"`
	LastTradedExchange  string           `json:"lastTradedExchange,omitempty"`
	LastTradedTimestamp *int64           `json:"lastTradedTimestamp,omitempty"`
	LastTrade           string           `json:"lastTrade,omitempty"`
	Bid                 *FlexibleFloat64 `json:"bid,omitempty"`
	Ask                 *FlexibleFloat64 `json:"ask,omitempty"`
	PriorClose          *FlexibleFloat64 `json:"priorClose,omitempty"`
	Open                *FlexibleFloat64 `json:"open,omitempty"`
	High                *FlexibleFloat64 `json:"high,omitempty"`
	Low                 *FlexibleFloat64 `json:"low,omitempty"`
	PointsChange        *FlexibleFloat64 `json:"pointsChange,omitempty"`
	PercentageChange    *FlexibleFloat64 `json:"percentageChange,omitempty"`
	OutOfMarketPrice    *FlexibleFloat64 `json:"outOfMarketPrice,omitempty"`
}

// ASXDepthOrder is an individual order inside an ASX depth level.
type ASXDepthOrder struct {
	ID          string           `json:"id,omitempty"`
	Exchange    string           `json:"exchange,omitempty"`
	Volume      *FlexibleInt     `json:"volume,omitempty"`
	Value       *FlexibleFloat64 `json:"value,omitempty"`
	Undisclosed *bool            `json:"undisclosed,omitempty"`
}

// ASXDepthLevel is an ASX market-depth level.
type ASXDepthLevel struct {
	ID             string           `json:"id,omitempty"`
	Price          *FlexibleFloat64 `json:"price,omitempty"`
	Volume         *FlexibleInt     `json:"volume,omitempty"`
	NumberOfOrders *FlexibleInt     `json:"numberOfOrders,omitempty"`
	Value          *FlexibleFloat64 `json:"value,omitempty"`
	Orders         []ASXDepthOrder  `json:"orders,omitempty"`
}

// ASXProductAggregatedDepth is ASX aggregated market depth.
type ASXProductAggregatedDepth struct {
	ID              string          `json:"id,omitempty"`
	Ticker          string          `json:"ticker,omitempty"`
	TotalBuyCount   *FlexibleInt    `json:"totalBuyCount,omitempty"`
	TotalSellCount  *FlexibleInt    `json:"totalSellCount,omitempty"`
	TotalBuyVolume  *FlexibleInt    `json:"totalBuyVolume,omitempty"`
	TotalSellVolume *FlexibleInt    `json:"totalSellVolume,omitempty"`
	BuyOrders       []ASXDepthLevel `json:"buyOrders,omitempty"`
	SellOrders      []ASXDepthLevel `json:"sellOrders,omitempty"`
}

// ASXCourseOfSale is an ASX course-of-sales record.
type ASXCourseOfSale struct {
	ID                  string           `json:"id,omitempty"`
	InstrumentCodeID    string           `json:"instrumentCodeId,omitempty"`
	ExchangeMarket      string           `json:"exchangeMarket,omitempty"`
	Price               *FlexibleFloat64 `json:"price,omitempty"`
	Volume              *FlexibleInt     `json:"volume,omitempty"`
	Value               *FlexibleFloat64 `json:"value,omitempty"`
	TradeTimeMillis     *int64           `json:"tradeTimeMillis,omitempty"`
	CancelledTimeMillis *int64           `json:"cancelledTimeMillis,omitempty"`
	BuyOrderNumber      string           `json:"buyOrderNumber,omitempty"`
	SellOrderNumber     string           `json:"sellOrderNumber,omitempty"`
}

// ASXProductCourseOfSales is ASX course-of-sales data.
type ASXProductCourseOfSales struct {
	Ticker        string            `json:"ticker,omitempty"`
	TotalVolume   *FlexibleInt      `json:"totalVolume,omitempty"`
	TotalTrades   *FlexibleInt      `json:"totalTrades,omitempty"`
	TotalValue    *FlexibleFloat64  `json:"totalValue,omitempty"`
	CourseOfSales []ASXCourseOfSale `json:"courseOfSales,omitempty"`
}

// ASXProductsService reads Australian-market products and instruments.
type ASXProductsService struct {
	client *Client
}

// Get returns an Australian-market product/quote for a symbol.
func (s *ASXProductsService) Get(ctx context.Context, symbol string) (*ASXProduct, error) {
	endpoint, err := FormatEndpoint(ASX.Symbol, map[string]string{"symbol": symbol})
	if err != nil {
		return nil, err
	}

	var product ASXProduct
	if err := s.client.do(ctx, http.MethodGet, endpoint, nil, &product); err != nil {
		return nil, err
	}
	return &product, nil
}

// Depth returns ASX aggregated market depth for a symbol.
func (s *ASXProductsService) Depth(ctx context.Context, symbol string) (*ASXProductAggregatedDepth, error) {
	endpoint, err := FormatEndpoint(ASX.AggregatedDepth, map[string]string{"symbol": symbol})
	if err != nil {
		return nil, err
	}

	var depth ASXProductAggregatedDepth
	if err := s.client.do(ctx, http.MethodGet, endpoint, nil, &depth); err != nil {
		return nil, err
	}
	return &depth, nil
}

// CourseOfSales returns ASX course-of-sales data for a symbol.
func (s *ASXProductsService) CourseOfSales(ctx context.Context, symbol string) (*ASXProductCourseOfSales, error) {
	endpoint, err := FormatEndpoint(ASX.CourseOfSales, map[string]string{"symbol": symbol})
	if err != nil {
		return nil, err
	}

	var sales ASXProductCourseOfSales
	if err := s.client.do(ctx, http.MethodGet, endpoint, nil, &sales); err != nil {
		return nil, err
	}
	return &sales, nil
}

// Search returns Australian-market instrument suggestions for a keyword.
func (s *ASXProductsService) Search(ctx context.Context, request ProductSearchByName) ([]ASXInstrument, error) {
	endpoint, err := FormatEndpoint(ASX.ProductsSuggestions, map[string]string{"keyword": request.Keyword})
	if err != nil {
		return nil, err
	}

	var response struct {
		Instruments []ASXInstrument `json:"instruments"`
	}
	if err := s.client.do(ctx, http.MethodGet, endpoint, nil, &response); err != nil {
		return nil, err
	}
	return response.Instruments, nil
}

// ProductFromInstrument returns the product for an Australian-market instrument suggestion.
func (s *ASXProductsService) ProductFromInstrument(ctx context.Context, instrument ASXInstrument) (*ASXProduct, error) {
	return s.Get(ctx, instrument.Symbol)
}
