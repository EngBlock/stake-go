package stake

import (
	"context"
	"net/http"
	"strings"
)

// NYSEMarketStatusValue is a US-market status payload.
type NYSEMarketStatusValue struct {
	ChangeAt string `json:"change_at,omitempty"`
	Next     string `json:"next,omitempty"`
	Current  string `json:"current"`
}

// NYSEMarketStatus is Stake's US-market status response.
type NYSEMarketStatus struct {
	Status NYSEMarketStatusValue `json:"status"`
}

// NYSEMarketService reads US-market status.
type NYSEMarketService struct {
	client *Client
}

// Get returns the current US-market status.
func (s *NYSEMarketService) Get(ctx context.Context) (*NYSEMarketStatus, error) {
	var response struct {
		Response NYSEMarketStatus `json:"response"`
	}
	if err := s.client.do(ctx, http.MethodGet, NYSE.MarketStatus, nil, &response); err != nil {
		return nil, err
	}
	return &response.Response, nil
}

// IsOpen reports whether the US market is open.
func (s *NYSEMarketService) IsOpen(ctx context.Context) (bool, error) {
	status, err := s.Get(ctx)
	if err != nil {
		return false, err
	}
	return strings.EqualFold(status.Status.Current, "open"), nil
}

// ASXMarketStatusValue is an Australian-market status payload.
type ASXMarketStatusValue struct {
	Current string `json:"current"`
}

// ASXMarketStatus is Stake's Australian-market status response.
type ASXMarketStatus struct {
	LastTradingDate *FlexibleTime        `json:"lastTradingDate,omitempty"`
	Status          ASXMarketStatusValue `json:"status"`
}

// ASXMarketService reads Australian-market status.
type ASXMarketService struct {
	client *Client
}

// Get returns the current Australian-market status.
func (s *ASXMarketService) Get(ctx context.Context) (*ASXMarketStatus, error) {
	var response []struct {
		LastTradedTimestamp *FlexibleTime `json:"lastTradedTimestamp"`
		MarketStatus        string        `json:"marketStatus"`
	}
	if err := s.client.do(ctx, http.MethodGet, ASX.MarketStatus, nil, &response); err != nil {
		return nil, err
	}
	if len(response) == 0 {
		return nil, ErrNotFound
	}
	return &ASXMarketStatus{
		LastTradingDate: response[0].LastTradedTimestamp,
		Status:          ASXMarketStatusValue{Current: response[0].MarketStatus},
	}, nil
}

// IsOpen reports whether the Australian market is open.
func (s *ASXMarketService) IsOpen(ctx context.Context) (bool, error) {
	status, err := s.Get(ctx)
	if err != nil {
		return false, err
	}
	return strings.EqualFold(status.Status.Current, "open"), nil
}
