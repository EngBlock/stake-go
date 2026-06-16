# Plan 003 — Guard the hand-written equity `UnmarshalJSON` against field drift

- **Written against commit:** `22636da`
- **Category:** Tech debt / maintainability
- **Effort:** S (≈1 hour)
- **Risk of the change:** None (adds a test only; no production code change)
- **Status:** TODO

> Verify the cited lines still match before editing. If the excerpts have drifted,
> STOP and report.

## Context (you have no prior knowledge of this repo)

`github.com/nathanbeddoewebdev/stake-go` is a Go client for the Stake brokerage
API. Stake's JSON is inconsistent — numbers sometimes arrive as JSON numbers and
sometimes as quoted strings (e.g. `"101.25"`). To cope, `pkg/stake/flex.go`
defines `FlexibleFloat64`, `FlexibleInt`, `FlexibleString`, and `FlexibleTime`,
each with a custom `UnmarshalJSON`.

For `NYSEEquityPosition` and `NYSEEquityPositions` (`pkg/stake/equity.go`), the
public struct keeps plain `float64`/`*float64` fields (so callers get clean
types), and a **hand-written `UnmarshalJSON`** decodes into a private `aux` struct
that uses the `Flexible*` types, then copies each field across:

```go
// pkg/stake/equity.go:46-110 (abridged)
func (p *NYSEEquityPosition) UnmarshalJSON(data []byte) error {
	var aux struct {
		AskPrice               *FlexibleFloat64 `json:"askPrice,omitempty"`
		AvailableForTradingQty FlexibleFloat64  `json:"availableForTradingQty"`
		// ... 22 more fields, one per public field ...
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	p.AskPrice = float64Ptr(aux.AskPrice)
	p.AvailableForTradingQty = aux.AvailableForTradingQty.Float64()
	// ... one assignment per field ...
	return nil
}
```

### The problem

This `aux` struct must mirror the public struct **field for field, by hand**. If
someone adds a field to `NYSEEquityPosition` (e.g. Stake adds a new attribute) and
forgets to add it to `aux` + the copy block, the field will **silently fail to
decode** — it will be the zero value at runtime, with **no compile error and no
test failure**. For portfolio/financial data, silently-zero fields are a nasty,
hard-to-spot bug.

The Go alias trick that normally avoids this (`type alias T; json.Unmarshal(data,
(*alias)(p))`) does **not** work here, because the public fields are `float64` and
cannot accept Stake's quoted-string numbers. So the parallel `aux` struct is
necessary — which means we need a **test that fails when the two drift apart**.

### The fix

Add a table-driven decode test that exercises **every field** of
`NYSEEquityPosition` (and the wrapper `NYSEEquityPositions`) using **string-encoded
numbers** for the numeric fields. If a future field is added to the struct but not
wired through `UnmarshalJSON`, this test will show the field decoding as zero and
fail. Pair it with a reflection-based assertion that counts decoded non-zero fields
so the guard is robust even if someone forgets to extend the literal assertions.

This is non-breaking: no production code changes, test-only.

## Repo conventions to follow

- Tests use `net/http/httptest` + `testing` and the helper `writeJSON` already
  present in `pkg/stake/client_test.go`. There is already a very similar test you
  should model on: `TestNYSEEquityPositionsDecodeFlexibleNumbers` in
  `client_test.go` (added in the same uncommitted change set). Read it first.
- Keep new tests in `pkg/stake` package (same package as existing client tests —
  check the package clause at the top of `client_test.go`).

## Files in scope

- `pkg/stake/equity_test.go` — **new file** (there is currently no `equity_test.go`;
  the equity decode test lives in `client_test.go`). Putting the drift guard in its
  own file keeps it discoverable. If you prefer, append to `client_test.go` instead
  — either is acceptable, but do not duplicate the existing
  `TestNYSEEquityPositionsDecodeFlexibleNumbers`.

## Files explicitly OUT of scope — do not touch

- `pkg/stake/equity.go` (no production change in this plan).
- `pkg/stake/flex.go`.
- `cmd/mcp-server/**`.

## Implementation steps

### Step 1 — Add a full-coverage decode test using `reflect`

Create `pkg/stake/equity_test.go`. The test:

1. Builds a JSON object where **every numeric field is a quoted string** and every
   string field is non-empty, then unmarshals it directly with
   `json.Unmarshal([]byte(...), &position)`.
2. Asserts a representative set of fields decoded to their expected non-zero values.
3. Uses `reflect` to assert that **no exported field of the decoded struct is its
   zero value** — this is the drift guard: a newly-added-but-unwired field will be
   zero and fail here.

```go
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

	// Spot-check a few values (including a string-encoded number and a pointer).
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
			continue // unexported
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
```

> **Important caveats to verify against the real struct** (`pkg/stake/equity.go:17-43`):
> - The drift guard relies on **every** exported field being non-zero given the
>   payload. Cross-check the payload JSON keys against the struct's `json:"..."`
>   tags. If any exported field is intentionally always-absent or hard to make
>   non-zero, add it to a small skip set with a comment rather than weakening the
>   guard. As of `22636da` all 25 fields map cleanly to the payload above.
> - `Category` is type `EquityCategory` (a string); `"Stock"` is the known value
>   `EquityCategoryStock` (`equity.go:14`). `Side` is type `Side` with value `"B"`.
>   Confirm both are non-empty after decode (they are, given the payload).

### Step 2 — Confirm it actually catches drift (one-time manual check, do NOT commit the edit)

To prove the guard works, *temporarily* add a throwaway exported field to
`NYSEEquityPosition` (e.g. `TestDriftField float64 \`json:"testDriftField"\``)
**without** wiring it into `UnmarshalJSON`, run the test, confirm it FAILS with the
"decoded to zero value" message, then **revert the throwaway field**. Do not leave
this field in `equity.go`. This step is verification only — `equity.go` must be
unchanged when you finish.

## Verification gates (run from repo root)

```sh
go build ./...
go vet ./...
go test ./pkg/stake/ -run 'TestNYSEEquityPosition.*CoversAllFields' -v
go test ./...
git diff --name-only pkg/stake/equity.go
```

Expected:
- Build/vet clean.
- Both new tests `PASS`.
- `go test ./...`: all `ok`.
- `git diff --name-only pkg/stake/equity.go` prints **nothing** (proves the
  production file is untouched — only the throwaway from Step 2, fully reverted).

## Done criteria (machine-checkable)

- [ ] `go build ./...` and `go vet ./...` exit 0.
- [ ] `pkg/stake/equity_test.go` exists with the two coverage tests.
- [ ] Both tests pass; `go test ./...` passes.
- [ ] `pkg/stake/equity.go` has **no** diff vs `22636da`.
- [ ] You confirmed (Step 2) the guard fails when a field is left unwired, then reverted.

## Test plan

Two new tests in `pkg/stake/equity_test.go`, modeled on the existing
`TestNYSEEquityPositionsDecodeFlexibleNumbers` in `client_test.go`. The
reflection-based assertion is the durable guard; the literal spot-checks document
intent. No existing tests change.

## Maintenance note

When Stake adds a field and you extend `NYSEEquityPosition`, also add it to the
test payload JSON above (and to the `aux` struct + copy block in `equity.go`). The
reflection guard will remind you if you forget the latter. Consider applying the
same drift-guard pattern to any other struct that grows a hand-written
`UnmarshalJSON` with a parallel `aux` struct (a quick grep for `var aux struct` in
`pkg/stake` finds candidates). Reviewers: any new hand-written `UnmarshalJSON`
mirroring a public struct should ship with a coverage test like this.
