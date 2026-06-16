package stake

import (
	"context"
	"net/http"
	"net/url"
)

// NYSEFunding is a detailed US-market funding transaction.
type NYSEFunding struct {
	IOF           string           `json:"iof,omitempty"`
	VET           string           `json:"vet,omitempty"`
	BSB           string           `json:"bsb,omitempty"`
	AccountNumber string           `json:"accountNumber,omitempty"`
	InsertDate    *FlexibleTime    `json:"insertDate,omitempty"`
	Channel       string           `json:"channel,omitempty"`
	AmountTo      *FlexibleFloat64 `json:"amountTo,omitempty"`
	AmountFrom    *FlexibleFloat64 `json:"amountFrom,omitempty"`
	Status        string           `json:"status,omitempty"`
	Speed         string           `json:"speed,omitempty"`
	FXFee         *FlexibleFloat64 `json:"fxFee,omitempty"`
	ExpressFee    *FlexibleInt     `json:"expressFee,omitempty"`
	TotalFee      *FlexibleFloat64 `json:"totalFee,omitempty"`
	SpotRate      *FlexibleFloat64 `json:"spotRate,omitempty"`
	Reference     string           `json:"reference,omitempty"`
	W8Fee         *FlexibleInt     `json:"w8Fee,omitempty"`
	CurrencyFrom  string           `json:"currencyFrom,omitempty"`
	CurrencyTo    string           `json:"currencyTo,omitempty"`
}

// NYSECashSettlement is a pending cash settlement line.
type NYSECashSettlement struct {
	UTCTime FlexibleTime `json:"utcTime"`
	Cash    float64      `json:"cash"`
}

// NYSECashAvailable holds US-market cash availability.
type NYSECashAvailable struct {
	CardHoldAmount               float64              `json:"cardHoldAmount"`
	CashAvailableForTrade        float64              `json:"cashAvailableForTrade"`
	CashAvailableForWithdrawal   float64              `json:"cashAvailableForWithdrawal"`
	CashBalance                  float64              `json:"cashBalance"`
	CashSettlement               []NYSECashSettlement `json:"cashSettlement"`
	DWCashAvailableForWithdrawal float64              `json:"dwCashAvailableForWithdrawal"`
	PendingOrdersAmount          float64              `json:"pendingOrdersAmount"`
	PendingPOLIAmount            float64              `json:"pendingPoliAmount"`
	PendingWithdrawals           float64              `json:"pendingWithdrawals"`
	ReservedCash                 float64              `json:"reservedCash"`
}

// NYSEFundsInFlight is a US-market pending funding transfer.
type NYSEFundsInFlight struct {
	Type                   string  `json:"type"`
	InsertDateTime         string  `json:"insertDateTime"`
	EstimatedArrivalTime   string  `json:"estimatedArrivalTime"`
	EstimatedArrivalTimeUS string  `json:"estimatedArrivalTimeUS"`
	TransactionType        string  `json:"transactionType"`
	ToAmount               float64 `json:"toAmount"`
	FromAmount             float64 `json:"fromAmount"`
}

// NYSEFundingsService reads US-market funding data.
type NYSEFundingsService struct {
	client *Client
}

// List returns US-market funding transactions by expanding transaction-history details.
func (s *NYSEFundingsService) List(ctx context.Context, request NYSETransactionRecordRequest) ([]NYSEFunding, error) {
	request = request.withDefaults()

	var history []struct {
		Reference     string `json:"reference"`
		ReferenceType string `json:"referenceType"`
	}
	if err := s.client.do(ctx, http.MethodPost, NYSE.TransactionHistory, request, &history); err != nil {
		return nil, err
	}

	fundings := make([]NYSEFunding, 0)
	for _, item := range history {
		if item.ReferenceType != string(NYSETransactionHistoryFunding) {
			continue
		}

		endpoint, err := FormatEndpoint(NYSE.TransactionDetails, map[string]string{
			"reference":      item.Reference,
			"reference_type": item.ReferenceType,
		})
		if err != nil {
			return nil, err
		}

		var funding NYSEFunding
		if err := s.client.do(ctx, http.MethodGet, endpoint, nil, &funding); err != nil {
			return nil, err
		}
		fundings = append(fundings, funding)
	}

	return fundings, nil
}

// InFlight returns US-market funds currently in flight.
func (s *NYSEFundingsService) InFlight(ctx context.Context) ([]NYSEFundsInFlight, error) {
	var response struct {
		FundsInFlight []NYSEFundsInFlight `json:"fundsInFlight"`
	}
	if err := s.client.do(ctx, http.MethodGet, NYSE.FundDetails, nil, &response); err != nil {
		return nil, err
	}
	return response.FundsInFlight, nil
}

// CashAvailable returns US-market cash availability.
func (s *NYSEFundingsService) CashAvailable(ctx context.Context) (*NYSECashAvailable, error) {
	var cash NYSECashAvailable
	if err := s.client.do(ctx, http.MethodGet, NYSE.CashAvailable, nil, &cash); err != nil {
		return nil, err
	}
	return &cash, nil
}

// ASXFundingAction is an ASX funding action filter/value.
type ASXFundingAction string

const (
	ASXFundingActionDeposit         ASXFundingAction = "DEPOSIT"
	ASXFundingActionDividendDeposit ASXFundingAction = "DIVIDEND_DEPOSIT"
	ASXFundingActionSettlement      ASXFundingAction = "SETTLEMENT"
	ASXFundingActionTransfer        ASXFundingAction = "TRANSFER"
	ASXFundingActionWithdrawal      ASXFundingAction = "WITHDRAWAL"
	ASXFundingActionAdjustment      ASXFundingAction = "ADJUSTMENT"
)

// ASXFundingCurrency is an ASX funding currency.
type ASXFundingCurrency string

const ASXFundingCurrencyAUD ASXFundingCurrency = "AUD"

// ASXFundingSide is an ASX funding ledger side.
type ASXFundingSide string

const (
	ASXFundingSideCredit ASXFundingSide = "CREDIT"
	ASXFundingSideDebit  ASXFundingSide = "DEBIT"
)

// ASXFundingStatus is an ASX funding status.
type ASXFundingStatus string

const (
	ASXFundingStatusAwaitingApproval ASXFundingStatus = "AWAITING_APPROVAL"
	ASXFundingStatusPending          ASXFundingStatus = "PENDING"
	ASXFundingStatusReconciled       ASXFundingStatus = "RECONCILED"
)

// ASXFundingRequest filters ASX funding transactions.
type ASXFundingRequest struct {
	Statuses []ASXFundingStatus
	Sort     []ASXSort
	Actions  []ASXFundingAction
	Limit    int
	Offset   int
}

func (r ASXFundingRequest) query() url.Values {
	values := url.Values{}
	statuses := r.Statuses
	if len(statuses) == 0 {
		statuses = []ASXFundingStatus{ASXFundingStatusReconciled}
	}
	for _, status := range statuses {
		values.Add("status", string(status))
	}
	for _, action := range r.Actions {
		values.Add("action", string(action))
	}
	for _, sort := range r.Sort {
		if sort.Attribute == "" || sort.Direction == "" {
			continue
		}
		values.Add("sort", sort.Attribute+","+string(sort.Direction))
	}
	limit := r.Limit
	if limit == 0 {
		limit = 100
	}
	values.Set("size", formatInt(limit))
	values.Set("page", formatInt(r.Offset))
	return values
}

// ASXFundingRecord is an Australian-market funding transaction.
type ASXFundingRecord struct {
	Action      ASXFundingAction   `json:"action,omitempty"`
	Amount      *FlexibleFloat64   `json:"amount,omitempty"`
	ApprovedBy  string             `json:"approvedBy,omitempty"`
	Currency    ASXFundingCurrency `json:"currency,omitempty"`
	CustomerFee *FlexibleFloat64   `json:"customerFee,omitempty"`
	ID          string             `json:"id,omitempty"`
	InsertedAt  *FlexibleTime      `json:"insertedAt,omitempty"`
	Reference   string             `json:"reference,omitempty"`
	Side        ASXFundingSide     `json:"side,omitempty"`
	Status      ASXFundingStatus   `json:"status,omitempty"`
	UpdatedAt   *FlexibleTime      `json:"updatedAt,omitempty"`
	UserID      string             `json:"userId,omitempty"`
}

// ASXFundings is a paginated ASX funding response.
type ASXFundings struct {
	Fundings   []ASXFundingRecord `json:"items,omitempty"`
	HasNext    *bool              `json:"hasNext,omitempty"`
	Page       *FlexibleInt       `json:"page,omitempty"`
	TotalItems *FlexibleInt       `json:"totalItems,omitempty"`
}

// ASXCashAvailable holds Australian-market cash availability.
type ASXCashAvailable struct {
	BuyingPower                    *FlexibleFloat64 `json:"buyingPower,omitempty"`
	CashAvailableForTransfer       *FlexibleFloat64 `json:"cashAvailableForTransfer,omitempty"`
	CashAvailableForWithdrawalHold *FlexibleFloat64 `json:"cashAvailableForWithdrawalHold,omitempty"`
	CashAvailableForWithdrawal     *FlexibleFloat64 `json:"cashAvailableForWithdrawal,omitempty"`
	ClearingCash                   *FlexibleFloat64 `json:"clearingCash,omitempty"`
	PendingBuys                    *FlexibleInt     `json:"pendingBuys,omitempty"`
	PendingWithdrawals             *FlexibleInt     `json:"pendingWithdrawals,omitempty"`
	SettledCash                    *FlexibleFloat64 `json:"settledCash,omitempty"`
	SettlementHold                 *FlexibleInt     `json:"settlementHold,omitempty"`
	TradeSettlement                *FlexibleInt     `json:"tradeSettlement,omitempty"`
}

// ASXFundingsService reads Australian-market funding data.
type ASXFundingsService struct {
	client *Client
}

// List returns ASX funding transactions matching the request.
func (s *ASXFundingsService) List(ctx context.Context, request ASXFundingRequest) (*ASXFundings, error) {
	endpoint := appendQuery(ASX.Transactions, request.query())
	var fundings ASXFundings
	if err := s.client.do(ctx, http.MethodGet, endpoint, nil, &fundings); err != nil {
		return nil, err
	}
	return &fundings, nil
}

// InFlight returns ASX pending or awaiting-approval funding transactions.
func (s *ASXFundingsService) InFlight(ctx context.Context) (*ASXFundings, error) {
	return s.List(ctx, ASXFundingRequest{
		Statuses: []ASXFundingStatus{ASXFundingStatusPending, ASXFundingStatusAwaitingApproval},
	})
}

// CashAvailable returns Australian-market cash availability.
func (s *ASXFundingsService) CashAvailable(ctx context.Context) (*ASXCashAvailable, error) {
	var cash ASXCashAvailable
	if err := s.client.do(ctx, http.MethodGet, ASX.CashAvailable, nil, &cash); err != nil {
		return nil, err
	}
	return &cash, nil
}
