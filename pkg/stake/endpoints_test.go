package stake

import "testing"

func TestNYSEEndpointConstants(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		got  string
		want string
	}{
		"stake_url": {
			got:  NYSE.StakeURL,
			want: "https://api2.prd.hellostake.com/",
		},
		"account_balance": {
			got:  NYSE.AccountBalance,
			want: "https://api2.prd.hellostake.com/api/cma/getAccountBalance",
		},
		"brokerage": {
			got:  NYSE.Brokerage,
			want: "https://api2.prd.hellostake.com/api/orders/brokerage?orderAmount={orderAmount}",
		},
		"market_data_quote": {
			got:  NYSE.MarketDataQuote,
			want: "https://api.prd.hellostake.com/us/pricing/quotes/marketData",
		},
		"ratings": {
			got:  NYSE.Ratings,
			want: "https://api2.prd.hellostake.com/api/data/calendar/ratings?tickers={symbols}&pageSize={limit}",
		},
		"transaction_details": {
			got:  NYSE.TransactionDetails,
			want: "https://api2.prd.hellostake.com/api/users/accounts/transactionDetails?reference={reference}&referenceType={reference_type}",
		},
		"statement": {
			got:  NYSE.Statement,
			want: "https://api2.prd.hellostake.com/api/data/fundamentals/{symbol}/statements?startDate={date}",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tt.got != tt.want {
				t.Fatalf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestASXEndpointConstants(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		got  string
		want string
	}{
		"stake_url": {
			got:  ASX.StakeURL,
			want: "https://api2.prd.hellostake.com/",
		},
		"brokerage": {
			got:  ASX.Brokerage,
			want: "https://api2.prd.hellostake.com/api/asx/orders/brokerage?orderAmount={orderAmount}",
		},
		"cancel_order": {
			got:  ASX.CancelOrder,
			want: "https://api2.prd.hellostake.com/api/asx/orders/{orderId}/cancel",
		},
		"aggregated_depth": {
			got:  ASX.AggregatedDepth,
			want: "https://api2.prd.hellostake.com/api/asx/instrument/aggregatedDepth/{symbol}?type=EQUITY",
		},
		"products_suggestions": {
			got:  ASX.ProductsSuggestions,
			want: "https://api2.prd.hellostake.com/api/asx/instrument/search?searchKey={keyword}",
		},
		"instrument_from_symbol": {
			got:  ASX.InstrumentFromSymbol,
			want: "https://api2.prd.hellostake.com/api/asx/instrument/view/{symbol}",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tt.got != tt.want {
				t.Fatalf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestFormatEndpoint(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		template string
		values   map[string]string
		want     string
	}{
		"single_placeholder": {
			template: NYSE.CancelOrder,
			values: map[string]string{
				"orderId": "order-123",
			},
			want: "https://api2.prd.hellostake.com/api/orders/cancelOrder/order-123",
		},
		"multiple_placeholders": {
			template: NYSE.Ratings,
			values: map[string]string{
				"symbols": "AAPL,MSFT",
				"limit":   "10",
			},
			want: "https://api2.prd.hellostake.com/api/data/calendar/ratings?tickers=AAPL,MSFT&pageSize=10",
		},
		"query_string_placeholder": {
			template: ASX.Brokerage,
			values: map[string]string{
				"orderAmount": "1000.5",
			},
			want: "https://api2.prd.hellostake.com/api/asx/orders/brokerage?orderAmount=1000.5",
		},
		"no_url_encoding": {
			template: NYSE.ProductsSuggestions,
			values: map[string]string{
				"keyword": "hello stake",
			},
			want: "https://api2.prd.hellostake.com/api/products/getProductSuggestions/hello stake",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := FormatEndpoint(tt.template, tt.values)
			if err != nil {
				t.Fatalf("FormatEndpoint returned error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatEndpointMissingPlaceholder(t *testing.T) {
	t.Parallel()

	_, err := FormatEndpoint(NYSE.TransactionDetails, map[string]string{
		"reference": "ref-123",
	})
	if err == nil {
		t.Fatal("expected missing placeholder error")
	}
}
