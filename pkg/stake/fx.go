package stake

import (
	"context"
	"net/http"
)

// Currency is a currency code used by Stake's FX endpoint.
type Currency string

const (
	CurrencyAUD Currency = "AUD"
	CurrencyUSD Currency = "USD"
)

// FXConversionRequest requests a currency conversion quote.
type FXConversionRequest struct {
	FromCurrency Currency `json:"fromCurrency"`
	ToCurrency   Currency `json:"toCurrency"`
	FromAmount   float64  `json:"fromAmount"`
}

// FXConversion is a Stake currency conversion quote.
type FXConversion struct {
	FromCurrency Currency `json:"fromCurrency"`
	ToCurrency   Currency `json:"toCurrency"`
	FromAmount   float64  `json:"fromAmount"`
	ToAmount     float64  `json:"toAmount"`
	Rate         float64  `json:"rate"`
	Quote        string   `json:"quote"`
}

// FXService converts currencies through Stake's US-market wallet endpoint.
type FXService struct {
	client *Client
}

// Convert returns an FX conversion quote.
func (s *FXService) Convert(ctx context.Context, request FXConversionRequest) (*FXConversion, error) {
	var conversion FXConversion
	if err := s.client.do(ctx, http.MethodPost, NYSE.Rate, request, &conversion); err != nil {
		return nil, err
	}
	return &conversion, nil
}
