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

type nyseMarketBuyInput struct {
	Symbol     string  `json:"symbol" jsonschema:"NYSE ticker symbol, for example AAPL"`
	AmountCash float64 `json:"amountCash" jsonschema:"Cash amount in USD to buy at market price"`
	Comments   string  `json:"comments,omitempty" jsonschema:"Optional order comments"`
}

type nyseLimitBuyInput struct {
	Symbol     string  `json:"symbol" jsonschema:"NYSE ticker symbol, for example AAPL"`
	Quantity   int     `json:"quantity" jsonschema:"Whole-share quantity to buy"`
	LimitPrice float64 `json:"limitPrice" jsonschema:"Maximum price per share"`
	Comments   string  `json:"comments,omitempty" jsonschema:"Optional order comments"`
}

type nyseStopBuyInput struct {
	Symbol     string  `json:"symbol" jsonschema:"NYSE ticker symbol, for example AAPL"`
	AmountCash float64 `json:"amountCash" jsonschema:"Cash amount in USD to buy when the stop price is reached"`
	Price      float64 `json:"price" jsonschema:"Stop trigger price"`
	Comments   string  `json:"comments,omitempty" jsonschema:"Optional order comments"`
}

type nyseMarketSellInput struct {
	Symbol   string  `json:"symbol" jsonschema:"NYSE ticker symbol, for example AAPL"`
	Quantity float64 `json:"quantity" jsonschema:"Share quantity to sell"`
	Comments string  `json:"comments,omitempty" jsonschema:"Optional order comments"`
}

type nyseLimitSellInput struct {
	Symbol     string  `json:"symbol" jsonschema:"NYSE ticker symbol, for example AAPL"`
	Quantity   int     `json:"quantity" jsonschema:"Whole-share quantity to sell"`
	LimitPrice float64 `json:"limitPrice" jsonschema:"Minimum price per share"`
	Comments   string  `json:"comments,omitempty" jsonschema:"Optional order comments"`
}

type nyseStopSellInput struct {
	Symbol    string  `json:"symbol" jsonschema:"NYSE ticker symbol, for example AAPL"`
	Quantity  float64 `json:"quantity" jsonschema:"Share quantity to sell"`
	StopPrice float64 `json:"stopPrice" jsonschema:"Stop trigger price"`
	Comments  string  `json:"comments,omitempty" jsonschema:"Optional order comments"`
}

type asxMarketTradeInput struct {
	Symbol         string  `json:"symbol,omitempty" jsonschema:"ASX ticker symbol, for example CBA. Required when instrumentCode is omitted or price is omitted."`
	InstrumentCode string  `json:"instrumentCode,omitempty" jsonschema:"Stake ASX instrument code. Required when symbol is omitted. If price is omitted, symbol is also required."`
	Units          int     `json:"units" jsonschema:"Whole unit quantity to trade"`
	Validity       string  `json:"validity,omitempty" jsonschema:"Order validity: GFD for one day or GTC for thirty days. Defaults to GTC."`
	ValidityDate   string  `json:"validityDate,omitempty" jsonschema:"Optional validity date, RFC3339 or YYYY-MM-DD"`
	Price          float64 `json:"price,omitempty" jsonschema:"Optional market-to-limit price. When omitted, buy uses current ask and sell uses current bid."`
}

type asxLimitTradeInput struct {
	Symbol         string  `json:"symbol,omitempty" jsonschema:"ASX ticker symbol, for example CBA. Required when instrumentCode is omitted."`
	InstrumentCode string  `json:"instrumentCode,omitempty" jsonschema:"Stake ASX instrument code. Required when symbol is omitted."`
	Units          int     `json:"units" jsonschema:"Whole unit quantity to trade"`
	Validity       string  `json:"validity,omitempty" jsonschema:"Order validity: GFD for one day or GTC for thirty days. Defaults to GTC."`
	ValidityDate   string  `json:"validityDate,omitempty" jsonschema:"Optional validity date, RFC3339 or YYYY-MM-DD"`
	Price          float64 `json:"price" jsonschema:"Limit price per unit"`
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

func (in nyseMarketBuyInput) toStake() (stake.NYSEMarketBuyRequest, error) {
	symbol, err := requireNonEmpty(in.Symbol, "symbol")
	if err != nil {
		return stake.NYSEMarketBuyRequest{}, err
	}
	if in.AmountCash <= 0 {
		return stake.NYSEMarketBuyRequest{}, fmt.Errorf("amountCash must be greater than zero")
	}
	return stake.NYSEMarketBuyRequest{Symbol: symbol, AmountCash: in.AmountCash, Comments: strings.TrimSpace(in.Comments)}, nil
}

func (in nyseLimitBuyInput) toStake() (stake.NYSELimitBuyRequest, error) {
	symbol, err := requireNonEmpty(in.Symbol, "symbol")
	if err != nil {
		return stake.NYSELimitBuyRequest{}, err
	}
	if in.Quantity <= 0 {
		return stake.NYSELimitBuyRequest{}, fmt.Errorf("quantity must be greater than zero")
	}
	if in.LimitPrice <= 0 {
		return stake.NYSELimitBuyRequest{}, fmt.Errorf("limitPrice must be greater than zero")
	}
	return stake.NYSELimitBuyRequest{Symbol: symbol, Quantity: in.Quantity, LimitPrice: in.LimitPrice, Comments: strings.TrimSpace(in.Comments)}, nil
}

func (in nyseStopBuyInput) toStake() (stake.NYSEStopBuyRequest, error) {
	symbol, err := requireNonEmpty(in.Symbol, "symbol")
	if err != nil {
		return stake.NYSEStopBuyRequest{}, err
	}
	if in.AmountCash < 10 {
		return stake.NYSEStopBuyRequest{}, fmt.Errorf("amountCash must be at least 10")
	}
	if in.Price <= 0 {
		return stake.NYSEStopBuyRequest{}, fmt.Errorf("price must be greater than zero")
	}
	return stake.NYSEStopBuyRequest{Symbol: symbol, AmountCash: in.AmountCash, Price: in.Price, Comments: strings.TrimSpace(in.Comments)}, nil
}

func (in nyseMarketSellInput) toStake() (stake.NYSEMarketSellRequest, error) {
	symbol, err := requireNonEmpty(in.Symbol, "symbol")
	if err != nil {
		return stake.NYSEMarketSellRequest{}, err
	}
	if in.Quantity <= 0 {
		return stake.NYSEMarketSellRequest{}, fmt.Errorf("quantity must be greater than zero")
	}
	return stake.NYSEMarketSellRequest{Symbol: symbol, Quantity: in.Quantity, Comments: strings.TrimSpace(in.Comments)}, nil
}

func (in nyseLimitSellInput) toStake() (stake.NYSELimitSellRequest, error) {
	symbol, err := requireNonEmpty(in.Symbol, "symbol")
	if err != nil {
		return stake.NYSELimitSellRequest{}, err
	}
	if in.Quantity <= 0 {
		return stake.NYSELimitSellRequest{}, fmt.Errorf("quantity must be greater than zero")
	}
	if in.LimitPrice <= 0 {
		return stake.NYSELimitSellRequest{}, fmt.Errorf("limitPrice must be greater than zero")
	}
	return stake.NYSELimitSellRequest{Symbol: symbol, Quantity: in.Quantity, LimitPrice: in.LimitPrice, Comments: strings.TrimSpace(in.Comments)}, nil
}

func (in nyseStopSellInput) toStake() (stake.NYSEStopSellRequest, error) {
	symbol, err := requireNonEmpty(in.Symbol, "symbol")
	if err != nil {
		return stake.NYSEStopSellRequest{}, err
	}
	if in.Quantity <= 0 {
		return stake.NYSEStopSellRequest{}, fmt.Errorf("quantity must be greater than zero")
	}
	if in.StopPrice <= 0 {
		return stake.NYSEStopSellRequest{}, fmt.Errorf("stopPrice must be greater than zero")
	}
	return stake.NYSEStopSellRequest{Symbol: symbol, Quantity: in.Quantity, StopPrice: in.StopPrice, Comments: strings.TrimSpace(in.Comments)}, nil
}

func (in asxMarketTradeInput) toMarketBuy() (stake.ASXMarketBuyRequest, error) {
	base, err := asxTradeBase(in.Symbol, in.InstrumentCode, in.Units, in.Validity, in.ValidityDate)
	if err != nil {
		return stake.ASXMarketBuyRequest{}, err
	}
	price, err := optionalPositivePrice(in.Price, "price")
	if err != nil {
		return stake.ASXMarketBuyRequest{}, err
	}
	if price == nil && base.symbol == "" {
		return stake.ASXMarketBuyRequest{}, fmt.Errorf("symbol is required when price is omitted")
	}
	return stake.ASXMarketBuyRequest{Symbol: base.symbol, InstrumentCode: base.instrumentCode, Units: base.units, Validity: base.validity, ValidityDate: base.validityDate, Price: price}, nil
}

func (in asxMarketTradeInput) toMarketSell() (stake.ASXMarketSellRequest, error) {
	base, err := asxTradeBase(in.Symbol, in.InstrumentCode, in.Units, in.Validity, in.ValidityDate)
	if err != nil {
		return stake.ASXMarketSellRequest{}, err
	}
	price, err := optionalPositivePrice(in.Price, "price")
	if err != nil {
		return stake.ASXMarketSellRequest{}, err
	}
	if price == nil && base.symbol == "" {
		return stake.ASXMarketSellRequest{}, fmt.Errorf("symbol is required when price is omitted")
	}
	return stake.ASXMarketSellRequest{Symbol: base.symbol, InstrumentCode: base.instrumentCode, Units: base.units, Validity: base.validity, ValidityDate: base.validityDate, Price: price}, nil
}

func (in asxLimitTradeInput) toLimitBuy() (stake.ASXLimitBuyRequest, error) {
	base, err := asxTradeBase(in.Symbol, in.InstrumentCode, in.Units, in.Validity, in.ValidityDate)
	if err != nil {
		return stake.ASXLimitBuyRequest{}, err
	}
	if in.Price <= 0 {
		return stake.ASXLimitBuyRequest{}, fmt.Errorf("price must be greater than zero")
	}
	return stake.ASXLimitBuyRequest{Symbol: base.symbol, InstrumentCode: base.instrumentCode, Units: base.units, Validity: base.validity, ValidityDate: base.validityDate, Price: in.Price}, nil
}

func (in asxLimitTradeInput) toLimitSell() (stake.ASXLimitSellRequest, error) {
	base, err := asxTradeBase(in.Symbol, in.InstrumentCode, in.Units, in.Validity, in.ValidityDate)
	if err != nil {
		return stake.ASXLimitSellRequest{}, err
	}
	if in.Price <= 0 {
		return stake.ASXLimitSellRequest{}, fmt.Errorf("price must be greater than zero")
	}
	return stake.ASXLimitSellRequest{Symbol: base.symbol, InstrumentCode: base.instrumentCode, Units: base.units, Validity: base.validity, ValidityDate: base.validityDate, Price: in.Price}, nil
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

type parsedASXTradeBase struct {
	symbol         string
	instrumentCode string
	units          int
	validity       stake.ASXExpiryDate
	validityDate   *time.Time
}

func asxTradeBase(symbolValue, instrumentCodeValue string, units int, validityValue, validityDateValue string) (parsedASXTradeBase, error) {
	symbol := strings.TrimSpace(symbolValue)
	instrumentCode := strings.TrimSpace(instrumentCodeValue)
	if symbol == "" && instrumentCode == "" {
		return parsedASXTradeBase{}, fmt.Errorf("symbol or instrumentCode is required")
	}
	if units <= 0 {
		return parsedASXTradeBase{}, fmt.Errorf("units must be greater than zero")
	}
	validity, err := parseASXValidity(validityValue)
	if err != nil {
		return parsedASXTradeBase{}, err
	}
	validityDate, err := parseOptionalTime(validityDateValue, "validityDate")
	if err != nil {
		return parsedASXTradeBase{}, err
	}
	var validityDatePtr *time.Time
	if !validityDate.IsZero() {
		validityDatePtr = &validityDate
	}

	return parsedASXTradeBase{
		symbol:         symbol,
		instrumentCode: instrumentCode,
		units:          units,
		validity:       validity,
		validityDate:   validityDatePtr,
	}, nil
}

func parseASXValidity(value string) (stake.ASXExpiryDate, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "":
		return "", nil
	case string(stake.ASXExpiryOneDay):
		return stake.ASXExpiryOneDay, nil
	case string(stake.ASXExpiryThirtyDays):
		return stake.ASXExpiryThirtyDays, nil
	default:
		return "", fmt.Errorf("validity must be GFD or GTC")
	}
}

func optionalPositivePrice(value float64, field string) (*float64, error) {
	if value < 0 {
		return nil, fmt.Errorf("%s must be greater than zero when provided", field)
	}
	if value == 0 {
		return nil, nil
	}
	return &value, nil
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
