package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/nathanbeddoewebdev/stake-go/pkg/stake"
)

type noInput struct{}

type fxConvertInput struct {
	FromCurrency string  `json:"fromCurrency" jsonschema:"Source currency code: AUD or USD"`
	ToCurrency   string  `json:"toCurrency" jsonschema:"Destination currency code: AUD or USD"`
	FromAmount   float64 `json:"fromAmount" jsonschema:"Amount to convert from the source currency"`
}

type symbolInput struct {
	Symbol string `json:"symbol" jsonschema:"Ticker symbol, for example AAPL or CBA"`
}

type keywordInput struct {
	Keyword string `json:"keyword" jsonschema:"Search keyword, company name, or ticker-like text"`
}

type brokerageInput struct {
	OrderAmount float64 `json:"orderAmount" jsonschema:"Estimated order value"`
}

type nyseTransactionsInput struct {
	From      string `json:"from,omitempty" jsonschema:"Start timestamp or date, RFC3339 or YYYY-MM-DD. Defaults to one year ago."`
	To        string `json:"to,omitempty" jsonschema:"End timestamp or date, RFC3339 or YYYY-MM-DD. Defaults to now."`
	Limit     int    `json:"limit,omitempty" jsonschema:"Maximum number of records. Defaults to 1000."`
	Offset    string `json:"offset,omitempty" jsonschema:"Pagination timestamp, RFC3339 or YYYY-MM-DD."`
	Direction string `json:"direction,omitempty" jsonschema:"Pagination direction: prev or next. Defaults to prev."`
}

type sortInput struct {
	Attribute string `json:"attribute" jsonschema:"Field to sort by"`
	Direction string `json:"direction" jsonschema:"Sort direction: asc or desc"`
}

type asxTransactionsInput struct {
	Sort   []sortInput `json:"sort,omitempty" jsonschema:"Sort expressions"`
	Limit  int         `json:"limit,omitempty" jsonschema:"Page size. Defaults to 100."`
	Offset int         `json:"offset,omitempty" jsonschema:"Page offset"`
}

type asxFundingsInput struct {
	Statuses []string    `json:"statuses,omitempty" jsonschema:"Funding statuses such as RECONCILED, PENDING, or AWAITING_APPROVAL"`
	Actions  []string    `json:"actions,omitempty" jsonschema:"Funding actions such as DEPOSIT, WITHDRAWAL, TRANSFER, SETTLEMENT, or ADJUSTMENT"`
	Sort     []sortInput `json:"sort,omitempty" jsonschema:"Sort expressions"`
	Limit    int         `json:"limit,omitempty" jsonschema:"Page size. Defaults to 100."`
	Offset   int         `json:"offset,omitempty" jsonschema:"Page offset"`
}

type ratingsInput struct {
	Symbols []string `json:"symbols" jsonschema:"Ticker symbols to request ratings for"`
	Limit   int      `json:"limit,omitempty" jsonschema:"Maximum ratings to return. Defaults to 50."`
}

type statementInput struct {
	Symbol    string `json:"symbol" jsonschema:"NYSE ticker symbol"`
	StartDate string `json:"startDate,omitempty" jsonschema:"Statement start date, RFC3339 or YYYY-MM-DD. Defaults to one year ago."`
}

type watchlistIDInput struct {
	ID string `json:"id" jsonschema:"Stake watchlist ID"`
}

type watchlistCreateInput struct {
	Name    string   `json:"name" jsonschema:"Watchlist name"`
	Tickers []string `json:"tickers,omitempty" jsonschema:"Optional initial ticker symbols"`
}

type watchlistUpdateInput struct {
	ID      string   `json:"id" jsonschema:"Stake watchlist ID"`
	Tickers []string `json:"tickers" jsonschema:"Ticker symbols to add or remove"`
}

type orderCancelInput struct {
	OrderID string `json:"orderId" jsonschema:"Stake order ID to cancel"`
}

func (in fxConvertInput) toStake() (stake.FXConversionRequest, error) {
	from, err := parseCurrency(in.FromCurrency, "fromCurrency")
	if err != nil {
		return stake.FXConversionRequest{}, err
	}
	to, err := parseCurrency(in.ToCurrency, "toCurrency")
	if err != nil {
		return stake.FXConversionRequest{}, err
	}
	if in.FromAmount <= 0 {
		return stake.FXConversionRequest{}, fmt.Errorf("fromAmount must be greater than zero")
	}

	return stake.FXConversionRequest{
		FromCurrency: from,
		ToCurrency:   to,
		FromAmount:   in.FromAmount,
	}, nil
}

func (in nyseTransactionsInput) toStake() (stake.NYSETransactionRecordRequest, error) {
	var request stake.NYSETransactionRecordRequest

	from, err := parseOptionalTime(in.From, "from")
	if err != nil {
		return request, err
	}
	to, err := parseOptionalTime(in.To, "to")
	if err != nil {
		return request, err
	}
	offset, err := parseOptionalTime(in.Offset, "offset")
	if err != nil {
		return request, err
	}
	direction, err := parseNYSEDirection(in.Direction)
	if err != nil {
		return request, err
	}

	request.From = from
	request.To = to
	request.Limit = in.Limit
	request.Direction = direction
	if !offset.IsZero() {
		request.Offset = &offset
	}
	return request, nil
}

func (in asxTransactionsInput) toStake() (stake.ASXTransactionRecordRequest, error) {
	sort, err := parseASXSorts(in.Sort)
	if err != nil {
		return stake.ASXTransactionRecordRequest{}, err
	}
	return stake.ASXTransactionRecordRequest{
		Sort:   sort,
		Limit:  in.Limit,
		Offset: in.Offset,
	}, nil
}

func (in asxFundingsInput) toStake() (stake.ASXFundingRequest, error) {
	sort, err := parseASXSorts(in.Sort)
	if err != nil {
		return stake.ASXFundingRequest{}, err
	}

	statuses := make([]stake.ASXFundingStatus, 0, len(in.Statuses))
	for _, status := range in.Statuses {
		status = strings.TrimSpace(status)
		if status == "" {
			continue
		}
		statuses = append(statuses, stake.ASXFundingStatus(strings.ToUpper(status)))
	}

	actions := make([]stake.ASXFundingAction, 0, len(in.Actions))
	for _, action := range in.Actions {
		action = strings.TrimSpace(action)
		if action == "" {
			continue
		}
		actions = append(actions, stake.ASXFundingAction(strings.ToUpper(action)))
	}

	return stake.ASXFundingRequest{
		Statuses: statuses,
		Sort:     sort,
		Actions:  actions,
		Limit:    in.Limit,
		Offset:   in.Offset,
	}, nil
}

func parseCurrency(value, field string) (stake.Currency, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case string(stake.CurrencyAUD):
		return stake.CurrencyAUD, nil
	case string(stake.CurrencyUSD):
		return stake.CurrencyUSD, nil
	default:
		return "", fmt.Errorf("%s must be AUD or USD", field)
	}
}

func parseNYSEDirection(value string) (stake.TransactionDirection, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	switch trimmed {
	case "":
		return "", nil
	case string(stake.TransactionDirectionPrevious):
		return stake.TransactionDirectionPrevious, nil
	case string(stake.TransactionDirectionNext):
		return stake.TransactionDirectionNext, nil
	default:
		return "", fmt.Errorf("direction must be prev or next")
	}
}

func parseASXSorts(input []sortInput) ([]stake.ASXSort, error) {
	sorts := make([]stake.ASXSort, 0, len(input))
	for _, item := range input {
		attribute := strings.TrimSpace(item.Attribute)
		direction := strings.ToLower(strings.TrimSpace(item.Direction))
		if attribute == "" && direction == "" {
			continue
		}
		if attribute == "" || direction == "" {
			return nil, fmt.Errorf("sort attribute and direction are both required")
		}

		switch direction {
		case string(stake.ASXSortAscending), string(stake.ASXSortDescending):
			sorts = append(sorts, stake.ASXSort{
				Attribute: attribute,
				Direction: stake.ASXSortDirection(direction),
			})
		default:
			return nil, fmt.Errorf("sort direction must be asc or desc")
		}
	}
	return sorts, nil
}

func parseOptionalTime(value, field string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, nil
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("%s must be RFC3339 or YYYY-MM-DD", field)
}

func requireNonEmpty(value, field string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	return trimmed, nil
}

func cleanStrings(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return cleaned
}

func output(key string, value any) any {
	return map[string]any{key: value}
}
