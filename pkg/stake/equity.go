package stake

import (
	"context"
	"encoding/json"
	"net/http"
)

// EquityCategory is a NYSE equity category.
type EquityCategory string

const (
	EquityCategoryETF   EquityCategory = "ETF"
	EquityCategoryStock EquityCategory = "Stock"
)

// NYSEEquityPosition is one US-market portfolio position.
type NYSEEquityPosition struct {
	AskPrice               *float64       `json:"askPrice,omitempty"`
	AvailableForTradingQty float64        `json:"availableForTradingQty"`
	AveragePrice           float64        `json:"avgPrice"`
	BidPrice               *float64       `json:"bidPrice,omitempty"`
	Category               EquityCategory `json:"category,omitempty"`
	CostBasis              float64        `json:"costBasis"`
	DailyReturnValue       float64        `json:"dailyReturnValue"`
	EncodedName            string         `json:"encodedName"`
	InstrumentID           string         `json:"instrumentID"`
	LastTrade              float64        `json:"lastTrade"`
	MarketPrice            float64        `json:"mktPrice"`
	MarketValue            float64        `json:"marketValue"`
	Name                   string         `json:"name"`
	OpenQty                float64        `json:"openQty"`
	Period                 string         `json:"period"`
	PriorClose             float64        `json:"priorClose"`
	ReturnOnStock          *float64       `json:"returnOnStock,omitempty"`
	Side                   Side           `json:"side"`
	Symbol                 string         `json:"symbol"`
	UnrealizedDayPLPercent float64        `json:"unrealizedDayPLPercent"`
	UnrealizedDayPL        float64        `json:"unrealizedDayPL"`
	UnrealizedPL           float64        `json:"unrealizedPL"`
	URLImage               string         `json:"urlImage"`
	YearlyReturnPercentage *float64       `json:"yearlyReturnPercentage,omitempty"`
	YearlyReturnValue      *float64       `json:"yearlyReturnValue,omitempty"`
}

// UnmarshalJSON accepts Stake's mixed numeric encodings while preserving the public float64 fields.
func (p *NYSEEquityPosition) UnmarshalJSON(data []byte) error {
	var aux struct {
		AskPrice               *FlexibleFloat64 `json:"askPrice,omitempty"`
		AvailableForTradingQty FlexibleFloat64  `json:"availableForTradingQty"`
		AveragePrice           FlexibleFloat64  `json:"avgPrice"`
		BidPrice               *FlexibleFloat64 `json:"bidPrice,omitempty"`
		Category               EquityCategory   `json:"category,omitempty"`
		CostBasis              FlexibleFloat64  `json:"costBasis"`
		DailyReturnValue       FlexibleFloat64  `json:"dailyReturnValue"`
		EncodedName            string           `json:"encodedName"`
		InstrumentID           string           `json:"instrumentID"`
		LastTrade              FlexibleFloat64  `json:"lastTrade"`
		MarketPrice            FlexibleFloat64  `json:"mktPrice"`
		MarketValue            FlexibleFloat64  `json:"marketValue"`
		Name                   string           `json:"name"`
		OpenQty                FlexibleFloat64  `json:"openQty"`
		Period                 string           `json:"period"`
		PriorClose             FlexibleFloat64  `json:"priorClose"`
		ReturnOnStock          *FlexibleFloat64 `json:"returnOnStock,omitempty"`
		Side                   Side             `json:"side"`
		Symbol                 string           `json:"symbol"`
		UnrealizedDayPLPercent FlexibleFloat64  `json:"unrealizedDayPLPercent"`
		UnrealizedDayPL        FlexibleFloat64  `json:"unrealizedDayPL"`
		UnrealizedPL           FlexibleFloat64  `json:"unrealizedPL"`
		URLImage               string           `json:"urlImage"`
		YearlyReturnPercentage *FlexibleFloat64 `json:"yearlyReturnPercentage,omitempty"`
		YearlyReturnValue      *FlexibleFloat64 `json:"yearlyReturnValue,omitempty"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	p.AskPrice = float64Ptr(aux.AskPrice)
	p.AvailableForTradingQty = aux.AvailableForTradingQty.Float64()
	p.AveragePrice = aux.AveragePrice.Float64()
	p.BidPrice = float64Ptr(aux.BidPrice)
	p.Category = aux.Category
	p.CostBasis = aux.CostBasis.Float64()
	p.DailyReturnValue = aux.DailyReturnValue.Float64()
	p.EncodedName = aux.EncodedName
	p.InstrumentID = aux.InstrumentID
	p.LastTrade = aux.LastTrade.Float64()
	p.MarketPrice = aux.MarketPrice.Float64()
	p.MarketValue = aux.MarketValue.Float64()
	p.Name = aux.Name
	p.OpenQty = aux.OpenQty.Float64()
	p.Period = aux.Period
	p.PriorClose = aux.PriorClose.Float64()
	p.ReturnOnStock = float64Ptr(aux.ReturnOnStock)
	p.Side = aux.Side
	p.Symbol = aux.Symbol
	p.UnrealizedDayPLPercent = aux.UnrealizedDayPLPercent.Float64()
	p.UnrealizedDayPL = aux.UnrealizedDayPL.Float64()
	p.UnrealizedPL = aux.UnrealizedPL.Float64()
	p.URLImage = aux.URLImage
	p.YearlyReturnPercentage = float64Ptr(aux.YearlyReturnPercentage)
	p.YearlyReturnValue = float64Ptr(aux.YearlyReturnValue)
	return nil
}

// NYSEEquityPositions is the user's US-market portfolio.
type NYSEEquityPositions struct {
	EquityPositions []NYSEEquityPosition `json:"equityPositions"`
	EquityValue     float64              `json:"equityValue"`
	PricesOnly      bool                 `json:"pricesOnly"`
}

// UnmarshalJSON accepts Stake's mixed numeric encodings while preserving the public float64 fields.
func (p *NYSEEquityPositions) UnmarshalJSON(data []byte) error {
	var aux struct {
		EquityPositions []NYSEEquityPosition `json:"equityPositions"`
		EquityValue     FlexibleFloat64      `json:"equityValue"`
		PricesOnly      bool                 `json:"pricesOnly"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	p.EquityPositions = aux.EquityPositions
	p.EquityValue = aux.EquityValue.Float64()
	p.PricesOnly = aux.PricesOnly
	return nil
}

func float64Ptr(value *FlexibleFloat64) *float64 {
	if value == nil {
		return nil
	}
	float := value.Float64()
	return &float
}

// NYSEEquitiesService reads US-market equity positions.
type NYSEEquitiesService struct {
	client *Client
}

// List returns the user's US-market portfolio positions.
func (s *NYSEEquitiesService) List(ctx context.Context) (*NYSEEquityPositions, error) {
	var positions NYSEEquityPositions
	if err := s.client.do(ctx, http.MethodGet, NYSE.EquityPositions, nil, &positions); err != nil {
		return nil, err
	}
	return &positions, nil
}

// ASXEquityPosition is one Australian-market portfolio position.
type ASXEquityPosition struct {
	AvailableForTradingQty *int     `json:"availableForTradingQty,omitempty"`
	AveragePrice           string   `json:"averagePrice,omitempty"`
	InstrumentID           string   `json:"instrumentId,omitempty"`
	MarketValue            string   `json:"marketValue,omitempty"`
	MarketPrice            string   `json:"mktPrice,omitempty"`
	Name                   string   `json:"name,omitempty"`
	OpenQty                *int     `json:"openQty,omitempty"`
	PriorClose             string   `json:"priorClose,omitempty"`
	RecentAnnouncement     *bool    `json:"recentAnnouncement,omitempty"`
	Sensitive              *bool    `json:"sensitive,omitempty"`
	Symbol                 string   `json:"symbol,omitempty"`
	UnrealizedDayPLPercent *float64 `json:"unrealizedDayPLPercent,omitempty"`
	UnrealizedDayPL        *float64 `json:"unrealizedDayPL,omitempty"`
	UnrealizedPLPercent    *float64 `json:"unrealizedPLPercent,omitempty"`
	UnrealizedPL           *float64 `json:"unrealizedPL,omitempty"`
}

// ASXEquityPositions is the user's Australian-market portfolio.
type ASXEquityPositions struct {
	PageNum         *int                `json:"pageNum,omitempty"`
	HasNext         *bool               `json:"hasNext,omitempty"`
	EquityPositions []ASXEquityPosition `json:"equityPositions,omitempty"`
}

// ASXEquitiesService reads Australian-market equity positions.
type ASXEquitiesService struct {
	client *Client
}

// List returns the user's Australian-market portfolio positions.
func (s *ASXEquitiesService) List(ctx context.Context) (*ASXEquityPositions, error) {
	var positions ASXEquityPositions
	if err := s.client.do(ctx, http.MethodGet, ASX.EquityPositions, nil, &positions); err != nil {
		return nil, err
	}
	return &positions, nil
}
