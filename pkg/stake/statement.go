package stake

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// StatementRequest requests fundamentals statements for a symbol.
type StatementRequest struct {
	Symbol    string
	StartDate time.Time
}

// StatementValue is one fundamentals data point.
type StatementValue struct {
	DataCode string  `json:"dataCode"`
	Value    float64 `json:"value"`
}

// StatementData groups fundamentals data categories.
type StatementData struct {
	BalanceSheet    []StatementValue `json:"balanceSheet"`
	IncomeStatement []StatementValue `json:"incomeStatement"`
	CashFlow        []StatementValue `json:"cashFlow"`
	Overview        []StatementValue `json:"overview"`
}

// Statement is one fundamentals statement period.
type Statement struct {
	Date          string        `json:"date"`
	Quarter       int           `json:"quarter"`
	Year          int           `json:"year"`
	StatementData StatementData `json:"statementData"`
}

// StatementService lists US-market fundamentals statements.
type StatementService struct {
	client *Client
}

// List returns fundamentals statements for a symbol.
func (s *StatementService) List(ctx context.Context, request StatementRequest) ([]Statement, error) {
	startDate := request.StartDate
	if startDate.IsZero() {
		startDate = time.Now().AddDate(-1, 0, 0)
	}

	endpoint, err := FormatEndpoint(NYSE.Statement, map[string]string{
		"symbol": request.Symbol,
		"date":   dateString(startDate),
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
		return []Statement{}, nil
	}

	var statements []Statement
	if err := json.Unmarshal(raw, &statements); err != nil {
		return nil, err
	}
	return statements, nil
}
