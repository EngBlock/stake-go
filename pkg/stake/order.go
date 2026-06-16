package stake

import (
	"context"
	"net/http"
)

// OrderType is a NYSE pending-order type.
type OrderType int

const (
	OrderTypeMarket OrderType = 1
	OrderTypeLimit  OrderType = 2
	OrderTypeStop   OrderType = 3
)

// NYSEOrder is a US-market pending order.
type NYSEOrder struct {
	OrderNumber      string       `json:"orderNo"`
	OrderID          string       `json:"orderID"`
	OrderCashAmount  int          `json:"orderCashAmt"`
	Symbol           string       `json:"symbol"`
	StopPrice        float64      `json:"stopPrice"`
	Side             Side         `json:"side"`
	OrderType        OrderType    `json:"orderType"`
	CumulativeQty    string       `json:"cumQty"`
	LimitPrice       float64      `json:"limitPrice"`
	CreatedWhen      FlexibleTime `json:"createdWhen"`
	OrderStatus      int          `json:"orderStatus"`
	OrderQty         float64      `json:"orderQty"`
	Description      string       `json:"description"`
	InstrumentID     string       `json:"instrumentID"`
	ImageURL         string       `json:"imageUrl"`
	InstrumentSymbol string       `json:"instrumentSymbol"`
	InstrumentName   string       `json:"instrumentName"`
	EncodedName      string       `json:"encodedName"`
}

// Brokerage is a brokerage estimate.
type Brokerage struct {
	BrokerageFee          *FlexibleFloat64 `json:"brokerageFee,omitempty"`
	BrokerageDiscount     *FlexibleFloat64 `json:"brokerageDiscount,omitempty"`
	FixedFee              *FlexibleFloat64 `json:"fixedFee,omitempty"`
	VariableFeePercentage *FlexibleFloat64 `json:"variableFeePercentage,omitempty"`
	VariableLimit         *FlexibleInt     `json:"variableLimit,omitempty"`
}

// CancelOrderRequest identifies an order to cancel.
type CancelOrderRequest struct {
	OrderID string `json:"orderId"`
}

// NYSEOrdersService manages US-market pending orders.
type NYSEOrdersService struct {
	client *Client
}

// List returns US-market pending orders.
func (s *NYSEOrdersService) List(ctx context.Context) ([]NYSEOrder, error) {
	var orders []NYSEOrder
	if err := s.client.do(ctx, http.MethodGet, NYSE.Orders, nil, &orders); err != nil {
		return nil, err
	}
	return orders, nil
}

// Cancel cancels a US-market pending order.
func (s *NYSEOrdersService) Cancel(ctx context.Context, request CancelOrderRequest) error {
	endpoint, err := FormatEndpoint(NYSE.CancelOrder, map[string]string{"orderId": request.OrderID})
	if err != nil {
		return err
	}
	return s.client.do(ctx, http.MethodDelete, endpoint, nil, nil)
}

// Brokerage returns the US-market brokerage estimate for an order amount.
func (s *NYSEOrdersService) Brokerage(ctx context.Context, orderAmount float64) (*Brokerage, error) {
	endpoint, err := FormatEndpoint(NYSE.Brokerage, map[string]string{"orderAmount": formatFloat(orderAmount)})
	if err != nil {
		return nil, err
	}

	var brokerage Brokerage
	if err := s.client.do(ctx, http.MethodGet, endpoint, nil, &brokerage); err != nil {
		return nil, err
	}
	return &brokerage, nil
}

// ASXOrder is an Australian-market order.
type ASXOrder struct {
	AveragePrice          *FlexibleFloat64 `json:"averagePrice,omitempty"`
	Broker                string           `json:"broker,omitempty"`
	CompletedTimestamp    *FlexibleTime    `json:"completedTimestamp,omitempty"`
	EstimatedBrokerage    *FlexibleFloat64 `json:"estimatedBrokerage,omitempty"`
	EstimatedExchangeFees *FlexibleFloat64 `json:"estimatedExchangeFees,omitempty"`
	ExpiresAt             *FlexibleTime    `json:"expiresAt,omitempty"`
	FilledUnits           *FlexibleFloat64 `json:"filledUnits,omitempty"`
	InstrumentCode        string           `json:"instrumentCode"`
	InstrumentID          string           `json:"instrumentId,omitempty"`
	LimitPrice            *FlexibleFloat64 `json:"limitPrice,omitempty"`
	OrderCompletionType   string           `json:"orderCompletionType,omitempty"`
	OrderID               string           `json:"id"`
	OrderStatus           string           `json:"orderStatus,omitempty"`
	PlacedTimestamp       FlexibleTime     `json:"placedTimestamp"`
	Side                  ASXSide          `json:"side"`
	Type                  ASXTradeType     `json:"type"`
	UnitsRemaining        *FlexibleInt     `json:"unitsRemaining,omitempty"`
	ValidityDate          string           `json:"validityDate,omitempty"`
	Validity              string           `json:"validity,omitempty"`
}

// ASXOrdersService manages Australian-market pending orders.
type ASXOrdersService struct {
	client *Client
}

// List returns Australian-market pending orders.
func (s *ASXOrdersService) List(ctx context.Context) ([]ASXOrder, error) {
	var orders []ASXOrder
	if err := s.client.do(ctx, http.MethodGet, ASX.Orders, nil, &orders); err != nil {
		return nil, err
	}
	return orders, nil
}

// Cancel cancels an Australian-market pending order.
func (s *ASXOrdersService) Cancel(ctx context.Context, request CancelOrderRequest) error {
	endpoint, err := FormatEndpoint(ASX.CancelOrder, map[string]string{"orderId": request.OrderID})
	if err != nil {
		return err
	}
	return s.client.do(ctx, http.MethodPost, endpoint, struct{}{}, nil)
}

// Brokerage returns the Australian-market brokerage estimate for an order amount.
func (s *ASXOrdersService) Brokerage(ctx context.Context, orderAmount float64) (*Brokerage, error) {
	endpoint, err := FormatEndpoint(ASX.Brokerage, map[string]string{"orderAmount": formatFloat(orderAmount)})
	if err != nil {
		return nil, err
	}

	var brokerage Brokerage
	if err := s.client.do(ctx, http.MethodGet, endpoint, nil, &brokerage); err != nil {
		return nil, err
	}
	return &brokerage, nil
}
