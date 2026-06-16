package stake

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"
)

var failedTransactionPattern = regexp.MustCompile(`^[0-9]{4}`)

// NYSETradeType is the US-market order type used by trade requests.
type NYSETradeType string

const (
	NYSETradeTypeMarket NYSETradeType = "market"
	NYSETradeTypeLimit  NYSETradeType = "limit"
	NYSETradeTypeStop   NYSETradeType = "stop"
)

// NYSEMarketBuyRequest buys by cash amount at market price.
type NYSEMarketBuyRequest struct {
	Symbol     string
	AmountCash float64
	Comments   string
}

// NYSELimitBuyRequest buys a quantity at a limit price.
type NYSELimitBuyRequest struct {
	Symbol     string
	LimitPrice float64
	Quantity   int
	Comments   string
}

// NYSEStopBuyRequest buys by cash amount when the stop price is reached.
type NYSEStopBuyRequest struct {
	Symbol     string
	AmountCash float64
	Price      float64
	Comments   string
}

// NYSELimitSellRequest sells a quantity at a limit price.
type NYSELimitSellRequest struct {
	Symbol     string
	LimitPrice float64
	Quantity   int
	Comments   string
}

// NYSEStopSellRequest sells a quantity when the stop price is reached.
type NYSEStopSellRequest struct {
	Symbol    string
	Quantity  float64
	StopPrice float64
	Comments  string
}

// NYSEMarketSellRequest sells a quantity at market price.
type NYSEMarketSellRequest struct {
	Symbol   string
	Quantity float64
	Comments string
}

type nyseTradeRequest interface {
	symbol() string
	payload(userID, itemID string) (map[string]any, error)
}

// NYSEBuyRequest is implemented by US-market buy requests.
type NYSEBuyRequest interface {
	nyseTradeRequest
	nyseBuyRequest()
}

// NYSESellRequest is implemented by US-market sell requests.
type NYSESellRequest interface {
	nyseTradeRequest
	nyseSellRequest()
}

func (r NYSEMarketBuyRequest) symbol() string  { return r.Symbol }
func (r NYSELimitBuyRequest) symbol() string   { return r.Symbol }
func (r NYSEStopBuyRequest) symbol() string    { return r.Symbol }
func (r NYSELimitSellRequest) symbol() string  { return r.Symbol }
func (r NYSEStopSellRequest) symbol() string   { return r.Symbol }
func (r NYSEMarketSellRequest) symbol() string { return r.Symbol }

func (r NYSEMarketBuyRequest) nyseBuyRequest()   {}
func (r NYSELimitBuyRequest) nyseBuyRequest()    {}
func (r NYSEStopBuyRequest) nyseBuyRequest()     {}
func (r NYSELimitSellRequest) nyseSellRequest()  {}
func (r NYSEStopSellRequest) nyseSellRequest()   {}
func (r NYSEMarketSellRequest) nyseSellRequest() {}

func baseNYSETradePayload(userID, itemID string, orderType NYSETradeType, comments string) map[string]any {
	payload := map[string]any{
		"itemType":  "instrument",
		"orderType": string(orderType),
		"userId":    userID,
		"itemId":    itemID,
	}
	if comments != "" {
		payload["comments"] = comments
	}
	return payload
}

func (r NYSEMarketBuyRequest) payload(userID, itemID string) (map[string]any, error) {
	payload := baseNYSETradePayload(userID, itemID, NYSETradeTypeMarket, r.Comments)
	payload["amountCash"] = r.AmountCash
	return payload, nil
}

func (r NYSELimitBuyRequest) payload(userID, itemID string) (map[string]any, error) {
	payload := baseNYSETradePayload(userID, itemID, NYSETradeTypeLimit, r.Comments)
	payload["limitPrice"] = r.LimitPrice
	payload["quantity"] = r.Quantity
	return payload, nil
}

func (r NYSEStopBuyRequest) payload(userID, itemID string) (map[string]any, error) {
	if r.AmountCash < 10 {
		return nil, errors.New("stake: amount cash must be at least 10")
	}
	payload := baseNYSETradePayload(userID, itemID, NYSETradeTypeStop, r.Comments)
	payload["amountCash"] = r.AmountCash
	payload["price"] = r.Price
	return payload, nil
}

func (r NYSELimitSellRequest) payload(userID, itemID string) (map[string]any, error) {
	payload := baseNYSETradePayload(userID, itemID, NYSETradeTypeLimit, r.Comments)
	payload["limitPrice"] = r.LimitPrice
	payload["quantity"] = r.Quantity
	return payload, nil
}

func (r NYSEStopSellRequest) payload(userID, itemID string) (map[string]any, error) {
	payload := baseNYSETradePayload(userID, itemID, NYSETradeTypeStop, r.Comments)
	payload["quantity"] = r.Quantity
	payload["stopPrice"] = r.StopPrice
	return payload, nil
}

func (r NYSEMarketSellRequest) payload(userID, itemID string) (map[string]any, error) {
	payload := baseNYSETradePayload(userID, itemID, NYSETradeTypeMarket, r.Comments)
	payload["quantity"] = r.Quantity
	return payload, nil
}

// NYSETradeResponse is Stake's US-market trade response.
type NYSETradeResponse struct {
	AmountCash        *FlexibleFloat64 `json:"amountCash,omitempty"`
	Category          string           `json:"category"`
	Commission        *FlexibleFloat64 `json:"commission,omitempty"`
	Description       string           `json:"description,omitempty"`
	DWOrderID         string           `json:"dwOrderId"`
	EffectivePrice    *FlexibleFloat64 `json:"effectivePrice,omitempty"`
	EncodedName       string           `json:"encodedName"`
	ID                string           `json:"id"`
	ImageURL          string           `json:"imageURL"`
	InsertedDate      FlexibleTime     `json:"insertedDate"`
	ItemID            string           `json:"itemId"`
	LimitPrice        *FlexibleFloat64 `json:"limitPrice,omitempty"`
	Name              string           `json:"name"`
	OrderRejectReason string           `json:"orderRejectReason,omitempty"`
	Quantity          *FlexibleFloat64 `json:"quantity,omitempty"`
	Side              string           `json:"side"`
	Status            *FlexibleInt     `json:"status,omitempty"`
	StopPrice         *FlexibleFloat64 `json:"stopPrice,omitempty"`
	Symbol            string           `json:"symbol"`
	UpdatedDate       FlexibleTime     `json:"updatedDate"`
}

// NYSETradesService submits US-market trades.
type NYSETradesService struct {
	client *Client
}

// Buy submits a US-market buy request.
func (s *NYSETradesService) Buy(ctx context.Context, request NYSEBuyRequest) (*NYSETradeResponse, error) {
	return s.trade(ctx, NYSE.QuickBuy, request)
}

// Sell submits a US-market sell request.
func (s *NYSETradesService) Sell(ctx context.Context, request NYSESellRequest) (*NYSETradeResponse, error) {
	return s.trade(ctx, NYSE.SellOrders, request)
}

func (s *NYSETradesService) trade(ctx context.Context, endpoint string, request nyseTradeRequest) (*NYSETradeResponse, error) {
	if s.client.User == nil {
		return nil, fmt.Errorf("%w: login required", ErrInvalidLogin)
	}

	product, err := s.client.NYSE.Products.Get(ctx, request.symbol())
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, fmt.Errorf("%w: product %q", ErrNotFound, request.symbol())
	}

	payload, err := request.payload(s.client.User.ID, product.ID)
	if err != nil {
		return nil, err
	}

	var response []NYSETradeResponse
	if err := s.client.do(ctx, http.MethodPost, endpoint, payload, &response); err != nil {
		return nil, err
	}
	if len(response) == 0 {
		return nil, ErrNotFound
	}

	trade := response[0]
	if err := s.verifySuccessfulTrade(ctx, trade); err != nil {
		return nil, err
	}
	return &trade, nil
}

func (s *NYSETradesService) verifySuccessfulTrade(ctx context.Context, trade NYSETradeResponse) error {
	var response struct {
		Transactions []struct {
			OrderID       string `json:"orderId"`
			OrderIDAlt    string `json:"orderID"`
			UpdatedReason string `json:"updatedReason"`
		} `json:"transactions"`
	}
	if err := s.client.do(ctx, http.MethodGet, NYSE.Transactions, nil, &response); err != nil {
		return err
	}
	if len(response.Transactions) == 0 {
		return fmt.Errorf("%w: no transaction found", ErrTradeFailed)
	}

	for _, transaction := range response.Transactions {
		orderID := transaction.OrderID
		if orderID == "" {
			orderID = transaction.OrderIDAlt
		}
		if orderID != trade.DWOrderID {
			continue
		}
		if failedTransactionPattern.MatchString(transaction.UpdatedReason) {
			return fmt.Errorf("%w: %s", ErrTradeFailed, transaction.UpdatedReason)
		}
		return nil
	}

	return fmt.Errorf("%w: matching transaction not found", ErrTradeFailed)
}

// ASXTradeType is the Australian-market order type used by trade requests.
type ASXTradeType string

const (
	ASXTradeTypeMarket ASXTradeType = "MARKET_TO_LIMIT"
	ASXTradeTypeLimit  ASXTradeType = "LIMIT"
	ASXTradeTypeStop   ASXTradeType = "STOP"
)

// ASXExpiryDate controls ASX order validity.
type ASXExpiryDate string

const (
	ASXExpiryOneDay     ASXExpiryDate = "GFD"
	ASXExpiryThirtyDays ASXExpiryDate = "GTC"
)

// ASXLimitBuyRequest buys ASX units at a limit price.
type ASXLimitBuyRequest struct {
	Symbol         string
	InstrumentCode string
	Units          int
	Validity       ASXExpiryDate
	ValidityDate   *time.Time
	Price          float64
}

// ASXLimitSellRequest sells ASX units at a limit price.
type ASXLimitSellRequest struct {
	Symbol         string
	InstrumentCode string
	Units          int
	Validity       ASXExpiryDate
	ValidityDate   *time.Time
	Price          float64
}

// ASXMarketBuyRequest buys ASX units using market-to-limit pricing.
type ASXMarketBuyRequest struct {
	Symbol         string
	InstrumentCode string
	Units          int
	Validity       ASXExpiryDate
	ValidityDate   *time.Time
	Price          *float64
}

// ASXMarketSellRequest sells ASX units using market-to-limit pricing.
type ASXMarketSellRequest struct {
	Symbol         string
	InstrumentCode string
	Units          int
	Validity       ASXExpiryDate
	ValidityDate   *time.Time
	Price          *float64
}

type asxTradePayload struct {
	InstrumentCode string        `json:"instrumentCode"`
	Units          int           `json:"units"`
	Validity       ASXExpiryDate `json:"validity"`
	ValidityDate   *time.Time    `json:"validityDate,omitempty"`
	Side           ASXSide       `json:"side"`
	Type           ASXTradeType  `json:"type"`
	Price          float64       `json:"price"`
}

type asxTradeRequest interface {
	prepareASXTrade(context.Context, *Client) (asxTradePayload, error)
}

// ASXBuyRequest is implemented by Australian-market buy requests.
type ASXBuyRequest interface {
	asxTradeRequest
	asxBuyRequest()
}

// ASXSellRequest is implemented by Australian-market sell requests.
type ASXSellRequest interface {
	asxTradeRequest
	asxSellRequest()
}

func (r ASXLimitBuyRequest) asxBuyRequest()    {}
func (r ASXMarketBuyRequest) asxBuyRequest()   {}
func (r ASXLimitSellRequest) asxSellRequest()  {}
func (r ASXMarketSellRequest) asxSellRequest() {}

func (r ASXLimitBuyRequest) prepareASXTrade(ctx context.Context, client *Client) (asxTradePayload, error) {
	return prepareASXTrade(ctx, client, r.Symbol, r.InstrumentCode, r.Units, r.Validity, r.ValidityDate, ASXSideBuy, ASXTradeTypeLimit, &r.Price)
}

func (r ASXLimitSellRequest) prepareASXTrade(ctx context.Context, client *Client) (asxTradePayload, error) {
	return prepareASXTrade(ctx, client, r.Symbol, r.InstrumentCode, r.Units, r.Validity, r.ValidityDate, ASXSideSell, ASXTradeTypeLimit, &r.Price)
}

func (r ASXMarketBuyRequest) prepareASXTrade(ctx context.Context, client *Client) (asxTradePayload, error) {
	price := r.Price
	if price == nil {
		product, err := client.ASX.Products.Get(ctx, r.Symbol)
		if err != nil {
			return asxTradePayload{}, err
		}
		if product == nil || product.Ask == nil {
			return asxTradePayload{}, fmt.Errorf("%w: missing ASX ask price for %q", ErrNotFound, r.Symbol)
		}
		ask := product.Ask.Float64()
		price = &ask
	}
	return prepareASXTrade(ctx, client, r.Symbol, r.InstrumentCode, r.Units, r.Validity, r.ValidityDate, ASXSideBuy, ASXTradeTypeMarket, price)
}

func (r ASXMarketSellRequest) prepareASXTrade(ctx context.Context, client *Client) (asxTradePayload, error) {
	price := r.Price
	if price == nil {
		product, err := client.ASX.Products.Get(ctx, r.Symbol)
		if err != nil {
			return asxTradePayload{}, err
		}
		if product == nil || product.Bid == nil {
			return asxTradePayload{}, fmt.Errorf("%w: missing ASX bid price for %q", ErrNotFound, r.Symbol)
		}
		bid := product.Bid.Float64()
		price = &bid
	}
	return prepareASXTrade(ctx, client, r.Symbol, r.InstrumentCode, r.Units, r.Validity, r.ValidityDate, ASXSideSell, ASXTradeTypeMarket, price)
}

func prepareASXTrade(ctx context.Context, client *Client, symbol, instrumentCode string, units int, validity ASXExpiryDate, validityDate *time.Time, side ASXSide, tradeType ASXTradeType, price *float64) (asxTradePayload, error) {
	if instrumentCode == "" {
		if symbol == "" {
			return asxTradePayload{}, errors.New("stake: either symbol or instrument code is required")
		}
		resolved, err := instrumentIDFromASXSymbol(ctx, client, symbol)
		if err != nil {
			return asxTradePayload{}, err
		}
		instrumentCode = resolved
	}
	if validity == "" {
		validity = ASXExpiryThirtyDays
	}
	if price == nil {
		return asxTradePayload{}, errors.New("stake: ASX trade price is required")
	}

	return asxTradePayload{
		InstrumentCode: instrumentCode,
		Units:          units,
		Validity:       validity,
		ValidityDate:   validityDate,
		Side:           side,
		Type:           tradeType,
		Price:          *price,
	}, nil
}

func instrumentIDFromASXSymbol(ctx context.Context, client *Client, symbol string) (string, error) {
	endpoint, err := FormatEndpoint(ASX.InstrumentFromSymbol, map[string]string{"symbol": symbol})
	if err != nil {
		return "", err
	}

	var response struct {
		InstrumentID string `json:"instrumentId"`
	}
	if err := client.do(ctx, http.MethodPost, endpoint, struct{}{}, &response); err != nil {
		return "", err
	}
	if response.InstrumentID == "" {
		return "", fmt.Errorf("%w: instrument %q", ErrNotFound, symbol)
	}
	return response.InstrumentID, nil
}

// ASXTradesService submits Australian-market trades.
type ASXTradesService struct {
	client *Client
}

// Buy submits an Australian-market buy request.
func (s *ASXTradesService) Buy(ctx context.Context, request ASXBuyRequest) (*ASXOrder, error) {
	return s.trade(ctx, request)
}

// Sell submits an Australian-market sell request.
func (s *ASXTradesService) Sell(ctx context.Context, request ASXSellRequest) (*ASXOrder, error) {
	return s.trade(ctx, request)
}

func (s *ASXTradesService) trade(ctx context.Context, request asxTradeRequest) (*ASXOrder, error) {
	payload, err := request.prepareASXTrade(ctx, s.client)
	if err != nil {
		return nil, err
	}

	var response struct {
		Order ASXOrder `json:"order"`
	}
	if err := s.client.do(ctx, http.MethodPost, ASX.Orders, payload, &response); err != nil {
		return nil, err
	}
	return &response.Order, nil
}
