package stake

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RatingsRequest requests analyst ratings for one or more symbols.
type RatingsRequest struct {
	Symbols []string
	Limit   int
}

// Rating is an analyst rating entry.
type Rating struct {
	ID            string     `json:"id,omitempty"`
	Symbol        string     `json:"ticker,omitempty"`
	Exchange      string     `json:"exchange,omitempty"`
	Name          string     `json:"name,omitempty"`
	Analyst       string     `json:"analyst,omitempty"`
	Currency      string     `json:"currency,omitempty"`
	URL           string     `json:"url,omitempty"`
	Importance    *int       `json:"importance,omitempty"`
	Notes         string     `json:"notes,omitempty"`
	Updated       *time.Time `json:"updated,omitempty"`
	ActionPT      string     `json:"action_pt,omitempty"`
	ActionCompany string     `json:"action_company,omitempty"`
	RatingCurrent *string    `json:"rating_current,omitempty"`
	PTCurrent     *float64   `json:"pt_current,omitempty"`
	RatingPrior   *string    `json:"rating_prior,omitempty"`
	PTPrior       *float64   `json:"pt_prior,omitempty"`
	URLCalendar   string     `json:"url_calendar,omitempty"`
	URLNews       string     `json:"url_news,omitempty"`
	AnalystName   string     `json:"analyst_name,omitempty"`
}

// UnmarshalJSON maps blank rating/price-target fields to nil, matching stake-python.
func (r *Rating) UnmarshalJSON(data []byte) error {
	type ratingAlias Rating
	var aux struct {
		ratingAlias
		RatingCurrent *string         `json:"rating_current"`
		RatingPrior   *string         `json:"rating_prior"`
		PTCurrent     json.RawMessage `json:"pt_current"`
		PTPrior       json.RawMessage `json:"pt_prior"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	*r = Rating(aux.ratingAlias)
	r.RatingCurrent = blankStringAsNil(aux.RatingCurrent)
	r.RatingPrior = blankStringAsNil(aux.RatingPrior)

	ptCurrent, err := rawOptionalFloat(aux.PTCurrent)
	if err != nil {
		return err
	}
	ptPrior, err := rawOptionalFloat(aux.PTPrior)
	if err != nil {
		return err
	}
	r.PTCurrent = ptCurrent
	r.PTPrior = ptPrior
	return nil
}

func blankStringAsNil(value *string) *string {
	if value == nil || *value == "" {
		return nil
	}
	return value
}

func rawOptionalFloat(raw json.RawMessage) (*float64, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) || bytes.Equal(trimmed, []byte(`""`)) {
		return nil, nil
	}

	var number float64
	if err := json.Unmarshal(trimmed, &number); err == nil {
		return &number, nil
	}

	var text string
	if err := json.Unmarshal(trimmed, &text); err != nil {
		return nil, err
	}
	if text == "" {
		return nil, nil
	}
	number, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return nil, err
	}
	return &number, nil
}

// RatingsService lists US-market analyst ratings.
type RatingsService struct {
	client *Client
}

// List returns analyst ratings for the requested symbols.
func (s *RatingsService) List(ctx context.Context, request RatingsRequest) ([]Rating, error) {
	limit := request.Limit
	if limit == 0 {
		limit = 50
	}
	endpoint, err := FormatEndpoint(NYSE.Ratings, map[string]string{
		"symbols": strings.Join(request.Symbols, ","),
		"limit":   formatInt(limit),
	})
	if err != nil {
		return nil, err
	}

	var noData struct {
		Message string `json:"message"`
	}
	var raw json.RawMessage
	if err := s.client.do(ctx, http.MethodGet, endpoint, nil, &raw); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &noData); err == nil && noData.Message == "No data returned" {
		return []Rating{}, nil
	}

	var response struct {
		Ratings []Rating `json:"ratings"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, err
	}
	return response.Ratings, nil
}
