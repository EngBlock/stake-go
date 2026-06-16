package stake

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
)

// GetWatchlistRequest retrieves a watchlist by ID.
type GetWatchlistRequest struct {
	ID string `json:"id"`
}

// DeleteWatchlistRequest deletes a watchlist by ID.
type DeleteWatchlistRequest struct {
	ID string `json:"id"`
}

// CreateWatchlistRequest creates a watchlist, optionally with initial tickers.
type CreateWatchlistRequest struct {
	Name    string   `json:"name"`
	Tickers []string `json:"tickers,omitempty"`
}

// UpdateWatchlistRequest adds or removes tickers from a watchlist.
type UpdateWatchlistRequest struct {
	ID      string   `json:"id"`
	Tickers []string `json:"tickers"`
}

// WatchlistInstrument is the common instrument shape returned inside watchlists.
type WatchlistInstrument struct {
	EncodedName        string `json:"encodedName,omitempty"`
	ImageURL           string `json:"imageUrl,omitempty"`
	InstrumentID       string `json:"instrumentId,omitempty"`
	Name               string `json:"name,omitempty"`
	Symbol             string `json:"symbol,omitempty"`
	Type               string `json:"type,omitempty"`
	RecentAnnouncement *bool  `json:"recentAnnouncement,omitempty"`
	Sensitive          *bool  `json:"sensitive,omitempty"`
}

// Watchlist is a Stake watchlist.
type Watchlist struct {
	WatchlistID string                `json:"watchlistId"`
	Name        string                `json:"name,omitempty"`
	Count       *FlexibleInt          `json:"count,omitempty"`
	TimeCreated *FlexibleTime         `json:"timeCreated,omitempty"`
	Instruments []WatchlistInstrument `json:"instruments,omitempty"`
}

// WatchlistService manages watchlists for one exchange.
type WatchlistService struct {
	client   *Client
	exchange Exchange
}

type watchlistEndpoints struct {
	List   string
	Create string
	Read   string
	Update string
}

func (s *WatchlistService) endpoints() (watchlistEndpoints, error) {
	switch s.exchange {
	case ExchangeNYSE:
		return watchlistEndpoints{
			List:   NYSE.Watchlists,
			Create: NYSE.CreateWatchlist,
			Read:   NYSE.ReadWatchlist,
			Update: NYSE.UpdateWatchlist,
		}, nil
	case ExchangeASX:
		return watchlistEndpoints{
			List:   ASX.Watchlists,
			Create: ASX.CreateWatchlist,
			Read:   ASX.ReadWatchlist,
			Update: ASX.UpdateWatchlist,
		}, nil
	default:
		return watchlistEndpoints{}, fmt.Errorf("%w: %s", ErrUnsupportedExchange, s.exchange)
	}
}

// List returns all watchlists for the service exchange.
func (s *WatchlistService) List(ctx context.Context) ([]Watchlist, error) {
	endpoints, err := s.endpoints()
	if err != nil {
		return nil, err
	}

	var response struct {
		Watchlists []Watchlist `json:"watchlists"`
	}
	if err := s.client.do(ctx, http.MethodGet, endpoints.List, nil, &response); err != nil {
		return nil, err
	}
	return response.Watchlists, nil
}

// Get returns a watchlist by ID.
func (s *WatchlistService) Get(ctx context.Context, request GetWatchlistRequest) (*Watchlist, error) {
	endpoints, err := s.endpoints()
	if err != nil {
		return nil, err
	}

	endpoint, err := FormatEndpoint(endpoints.Read, map[string]string{"watchlist_id": request.ID})
	if err != nil {
		return nil, err
	}

	var watchlist Watchlist
	if err := s.client.do(ctx, http.MethodGet, endpoint, nil, &watchlist); err != nil {
		return nil, err
	}
	return &watchlist, nil
}

// Create creates a watchlist. Duplicate local names return an error before the POST.
func (s *WatchlistService) Create(ctx context.Context, request CreateWatchlistRequest) (*Watchlist, error) {
	endpoints, err := s.endpoints()
	if err != nil {
		return nil, err
	}

	existing, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, watchlist := range existing {
		if watchlist.Name == request.Name {
			return nil, fmt.Errorf("stake: watchlist named %q already exists", request.Name)
		}
	}

	var response struct {
		NewWatchlistID string `json:"newWatchlistId"`
	}
	if err := s.client.do(ctx, http.MethodPost, endpoints.Create, request, &response); err != nil {
		return nil, err
	}
	if response.NewWatchlistID == "" {
		return nil, errors.New("stake: create watchlist response did not include newWatchlistId")
	}

	if len(request.Tickers) > 0 {
		return s.Add(ctx, UpdateWatchlistRequest{ID: response.NewWatchlistID, Tickers: request.Tickers})
	}
	return s.Get(ctx, GetWatchlistRequest{ID: response.NewWatchlistID})
}

// Add adds tickers to a watchlist, ignoring tickers already present.
func (s *WatchlistService) Add(ctx context.Context, request UpdateWatchlistRequest) (*Watchlist, error) {
	return s.update(ctx, request, true)
}

// Remove removes tickers from a watchlist, ignoring tickers not present.
func (s *WatchlistService) Remove(ctx context.Context, request UpdateWatchlistRequest) (*Watchlist, error) {
	return s.update(ctx, request, false)
}

func (s *WatchlistService) update(ctx context.Context, request UpdateWatchlistRequest, add bool) (*Watchlist, error) {
	endpoints, err := s.endpoints()
	if err != nil {
		return nil, err
	}

	existing, err := s.Get(ctx, GetWatchlistRequest{ID: request.ID})
	if err != nil {
		return nil, err
	}

	existingTickers := make([]string, 0, len(existing.Instruments))
	for _, instrument := range existing.Instruments {
		existingTickers = append(existingTickers, instrument.Symbol)
	}

	filtered := make([]string, 0, len(request.Tickers))
	for _, ticker := range request.Tickers {
		contains := slices.Contains(existingTickers, ticker)
		if add && !contains {
			filtered = append(filtered, ticker)
		}
		if !add && contains {
			filtered = append(filtered, ticker)
		}
	}

	if len(filtered) == 0 {
		return existing, nil
	}

	endpoint, err := FormatEndpoint(endpoints.Update, map[string]string{"watchlist_id": request.ID})
	if err != nil {
		return nil, err
	}

	payload := struct {
		Tickers []string `json:"tickers"`
	}{Tickers: filtered}

	var watchlist Watchlist
	method := http.MethodPost
	if !add {
		method = http.MethodDelete
	}
	if err := s.client.do(ctx, method, endpoint, payload, &watchlist); err != nil {
		return nil, err
	}
	return &watchlist, nil
}

// Delete deletes a watchlist. It returns true when the deleted ID is absent from the response.
func (s *WatchlistService) Delete(ctx context.Context, request DeleteWatchlistRequest) (bool, error) {
	endpoints, err := s.endpoints()
	if err != nil {
		return false, err
	}

	endpoint, err := FormatEndpoint(endpoints.Read, map[string]string{"watchlist_id": request.ID})
	if err != nil {
		return false, err
	}

	var remaining []Watchlist
	if err := s.client.do(ctx, http.MethodDelete, endpoint, nil, &remaining); err != nil {
		return false, err
	}

	for _, watchlist := range remaining {
		if watchlist.WatchlistID == request.ID {
			return false, nil
		}
	}
	return true, nil
}
