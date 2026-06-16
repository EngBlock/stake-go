package stake

import (
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Side is a NYSE trade/order side.
type Side string

const (
	SideBuy  Side = "B"
	SideSell Side = "S"
)

// ASXSide is an ASX trade/order side.
type ASXSide string

const (
	ASXSideBuy  ASXSide = "BUY"
	ASXSideSell ASXSide = "SELL"
)

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func formatInt(value int) string {
	return strconv.Itoa(value)
}

func appendQuery(endpoint string, values url.Values) string {
	encoded := values.Encode()
	if encoded == "" {
		return endpoint
	}

	separator := "?"
	if strings.Contains(endpoint, "?") {
		separator = "&"
	}

	return endpoint + separator + encoded
}

func dateString(t time.Time) string {
	return t.Format("2006-01-02")
}
