package stake

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestNYSEEquityPositionUnmarshalCoversAllFields(t *testing.T) {
	const payload = `{
		"askPrice": "101.25",
		"availableForTradingQty": "2",
		"avgPrice": "95.50",
		"bidPrice": "100.75",
		"category": "Stock",
		"costBasis": "191.00",
		"dailyReturnValue": "1.23",
		"encodedName": "apple-inc-aapl",
		"instrumentID": "instrument-1",
		"lastTrade": "101.00",
		"mktPrice": "101.00",
		"marketValue": "202.00",
		"name": "Apple",
		"openQty": "2",
		"period": "1D",
		"priorClose": "99.77",
		"returnOnStock": "11.00",
		"side": "B",
		"symbol": "AAPL",
		"unrealizedDayPLPercent": "0.61",
		"unrealizedDayPL": "1.23",
		"unrealizedPL": "11.00",
		"urlImage": "https://example.com/aapl.png",
		"yearlyReturnPercentage": "12.34",
		"yearlyReturnValue": "22.00"
	}`

	var pos NYSEEquityPosition
	if err := json.Unmarshal([]byte(payload), &pos); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if pos.Symbol != "AAPL" {
		t.Fatalf("Symbol = %q, want AAPL", pos.Symbol)
	}
	if pos.MarketValue != 202.00 {
		t.Fatalf("MarketValue = %v, want 202.00", pos.MarketValue)
	}
	if pos.AskPrice == nil || *pos.AskPrice != 101.25 {
		t.Fatalf("AskPrice = %v, want 101.25", pos.AskPrice)
	}

	// Drift guard: every exported field in the payload above is non-empty, so
	// every exported field of the decoded struct must be non-zero. A field added
	// to the struct but not wired through UnmarshalJSON will fail here.
	v := reflect.ValueOf(pos)
	tp := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := tp.Field(i)
		if field.PkgPath != "" {
			continue
		}
		fv := v.Field(i)
		if fv.Kind() == reflect.Ptr {
			if fv.IsNil() {
				t.Errorf("field %s decoded to nil; add it to NYSEEquityPosition.UnmarshalJSON aux struct + copy block", field.Name)
			}
			continue
		}
		if fv.IsZero() {
			t.Errorf("field %s decoded to zero value; add it to NYSEEquityPosition.UnmarshalJSON aux struct + copy block", field.Name)
		}
	}
}

func TestNYSEEquityPositionsWrapperUnmarshalCoversAllFields(t *testing.T) {
	const payload = `{
		"equityValue": "1234.56",
		"pricesOnly": true,
		"equityPositions": [{"symbol": "AAPL", "marketValue": "10"}]
	}`

	var wrapper NYSEEquityPositions
	if err := json.Unmarshal([]byte(payload), &wrapper); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if wrapper.EquityValue != 1234.56 {
		t.Fatalf("EquityValue = %v, want 1234.56", wrapper.EquityValue)
	}
	if !wrapper.PricesOnly {
		t.Fatal("PricesOnly = false, want true")
	}
	if len(wrapper.EquityPositions) != 1 || wrapper.EquityPositions[0].Symbol != "AAPL" {
		t.Fatalf("EquityPositions = %+v, want one AAPL position", wrapper.EquityPositions)
	}
}
