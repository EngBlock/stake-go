package stake

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// TransactionDirection controls NYSE transaction pagination direction.
type TransactionDirection string

const (
	TransactionDirectionPrevious TransactionDirection = "prev"
	TransactionDirectionNext     TransactionDirection = "next"
)

// NYSETransactionRecordRequest filters NYSE account transactions.
type NYSETransactionRecordRequest struct {
	To        time.Time            `json:"to"`
	From      time.Time            `json:"from"`
	Limit     int                  `json:"limit"`
	Offset    *time.Time           `json:"offset,omitempty"`
	Direction TransactionDirection `json:"direction"`
}

// NewNYSETransactionRecordRequest returns the same defaults used by stake-python.
func NewNYSETransactionRecordRequest() NYSETransactionRecordRequest {
	now := time.Now().UTC()
	return NYSETransactionRecordRequest{
		To:        now,
		From:      now.AddDate(-1, 0, 0),
		Limit:     1000,
		Direction: TransactionDirectionPrevious,
	}
}

func (r NYSETransactionRecordRequest) withDefaults() NYSETransactionRecordRequest {
	defaults := NewNYSETransactionRecordRequest()
	if r.To.IsZero() {
		r.To = defaults.To
	}
	if r.From.IsZero() {
		r.From = defaults.From
	}
	if r.Limit == 0 {
		r.Limit = defaults.Limit
	}
	if r.Direction == "" {
		r.Direction = defaults.Direction
	}
	return r
}

// NYSETransactionHistoryType is Stake's NYSE transaction-history category.
type NYSETransactionHistoryType string

const (
	NYSETransactionHistoryBuy             NYSETransactionHistoryType = "Buy"
	NYSETransactionHistoryCorporateAction NYSETransactionHistoryType = "Corporate Action"
	NYSETransactionHistoryDividend        NYSETransactionHistoryType = "Dividend"
	NYSETransactionHistoryDividendTax     NYSETransactionHistoryType = "Dividend Tax"
	NYSETransactionHistoryFunding         NYSETransactionHistoryType = "Funding"
	NYSETransactionHistorySell            NYSETransactionHistoryType = "Sell"
)

// NYSETransactionInstrument is the instrument nested on NYSE transactions.
type NYSETransactionInstrument struct {
	ID     string `json:"id"`
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}

// NYSETransaction is an account transaction from the US market.
type NYSETransaction struct {
	AccountAmount                 float64                    `json:"accountAmount"`
	AccountBalance                float64                    `json:"accountBalance"`
	AccountType                   string                     `json:"accountType"`
	Comment                       string                     `json:"comment"`
	DividendTax                   map[string]any             `json:"dividendTax,omitempty"`
	Dividend                      map[string]any             `json:"dividend,omitempty"`
	DNB                           bool                       `json:"dnb"`
	FeeBase                       int                        `json:"feeBase"`
	FeeExchange                   int                        `json:"feeExchange"`
	FeeSEC                        float64                    `json:"feeSec"`
	FeeTAF                        float64                    `json:"feeTaf"`
	FeeXtraShares                 int                        `json:"feeXtraShares"`
	FillPrice                     float64                    `json:"fillPx"`
	FillQuantity                  float64                    `json:"fillQty"`
	FinancialTransactionID        string                     `json:"finTranID"`
	FinancialTransactionTypeID    string                     `json:"finTranTypeID"`
	Instrument                    *NYSETransactionInstrument `json:"instrument,omitempty"`
	MergerAcquisition             map[string]any             `json:"mergerAcquisition,omitempty"`
	OrderID                       string                     `json:"orderID,omitempty"`
	OrderNumber                   string                     `json:"orderNo,omitempty"`
	PositionDelta                 *FlexibleFloat64           `json:"positionDelta,omitempty"`
	SendCommissionToInteliclear   bool                       `json:"sendCommissionToInteliclear"`
	Symbol                        string                     `json:"symbol,omitempty"`
	SystemAmount                  int                        `json:"systemAmount"`
	TransactionAmount             float64                    `json:"tranAmount"`
	TransactionSource             string                     `json:"tranSource"`
	TransactionWhen               FlexibleTime               `json:"tranWhen"`
	UpdatedReason                 string                     `json:"updatedReason,omitempty"`
	WLPAmount                     int                        `json:"wlpAmount"`
	WLPFinancialTransactionTypeID string                     `json:"wlpFinTranTypeID,omitempty"`
}

// NYSETransactionsService lists NYSE account transactions.
type NYSETransactionsService struct {
	client *Client
}

// List returns transactions matching the request.
func (s *NYSETransactionsService) List(ctx context.Context, request NYSETransactionRecordRequest) ([]NYSETransaction, error) {
	request = request.withDefaults()

	var transactions []NYSETransaction
	if err := s.client.do(ctx, http.MethodPost, NYSE.AccountTransactions, request, &transactions); err != nil {
		return nil, err
	}

	for i := range transactions {
		if transactions[i].Instrument != nil && transactions[i].Symbol == "" {
			transactions[i].Symbol = transactions[i].Instrument.Symbol
		}
	}

	return transactions, nil
}

// ASXSortDirection controls ASX pagination sort direction.
type ASXSortDirection string

const (
	ASXSortAscending  ASXSortDirection = "asc"
	ASXSortDescending ASXSortDirection = "desc"
)

// ASXSort is a sort expression for ASX endpoints.
type ASXSort struct {
	Attribute string           `json:"attribute"`
	Direction ASXSortDirection `json:"direction"`
}

// ASXTransactionRecordRequest filters ASX trade activity.
type ASXTransactionRecordRequest struct {
	Sort   []ASXSort
	Limit  int
	Offset int
}

func (r ASXTransactionRecordRequest) query() url.Values {
	values := url.Values{}
	limit := r.Limit
	if limit == 0 {
		limit = 100
	}
	values.Set("size", formatInt(limit))
	values.Set("page", formatInt(r.Offset))

	for _, sort := range r.Sort {
		if sort.Attribute == "" || sort.Direction == "" {
			continue
		}
		values.Add("sort", sort.Attribute+","+string(sort.Direction))
	}

	return values
}

// ASXTransaction is a trade activity record from the Australian market.
type ASXTransaction struct {
	AveragePrice         *FlexibleFloat64 `json:"averagePrice,omitempty"`
	BrokerOrderID        *FlexibleInt     `json:"brokerOrderId,omitempty"`
	CompletedTimestamp   *FlexibleTime    `json:"completedTimestamp,omitempty"`
	Consideration        *FlexibleFloat64 `json:"consideration,omitempty"`
	ContractNoteNumber   *FlexibleInt     `json:"contractNoteNumber,omitempty"`
	ContractNoteNumbers  []int            `json:"contractNoteNumbers,omitempty"`
	ContractNoteReceived *bool            `json:"contractNoteReceived,omitempty"`
	EffectivePrice       *FlexibleFloat64 `json:"effectivePrice,omitempty"`
	ExecutionDate        *FlexibleTime    `json:"executionDate,omitempty"`
	InstrumentID         string           `json:"instrumentCode,omitempty"`
	LimitPrice           *FlexibleFloat64 `json:"limitPrice,omitempty"`
	OrderCompletionType  string           `json:"orderCompletionType,omitempty"`
	OrderStatus          string           `json:"orderStatus,omitempty"`
	PlacedTimestamp      *FlexibleTime    `json:"placedTimestamp,omitempty"`
	Side                 ASXSide          `json:"side,omitempty"`
	Type                 string           `json:"type,omitempty"`
	Units                *FlexibleFloat64 `json:"units,omitempty"`
	UserBrokerageFees    *FlexibleFloat64 `json:"userBrokerageFees,omitempty"`
}

// ASXTransactions is a paginated ASX transaction response.
type ASXTransactions struct {
	Transactions []ASXTransaction `json:"items,omitempty"`
	HasNext      *bool            `json:"hasNext,omitempty"`
	Page         *FlexibleInt     `json:"page,omitempty"`
	TotalItems   *FlexibleInt     `json:"totalItems,omitempty"`
}

// ASXTransactionsService lists ASX trade activity.
type ASXTransactionsService struct {
	client *Client
}

// List returns ASX trade activity matching the request.
func (s *ASXTransactionsService) List(ctx context.Context, request ASXTransactionRecordRequest) (*ASXTransactions, error) {
	endpoint := appendQuery(ASX.TradeActivity, request.query())
	var transactions ASXTransactions
	if err := s.client.do(ctx, http.MethodGet, endpoint, nil, &transactions); err != nil {
		return nil, err
	}
	return &transactions, nil
}
