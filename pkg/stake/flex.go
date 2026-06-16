package stake

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"
)

// FlexibleString decodes JSON strings or primitive values into a string.
type FlexibleString string

// String returns the value as a string.
func (s FlexibleString) String() string {
	return string(s)
}

// UnmarshalJSON decodes strings, numbers, booleans, empty strings, and null.
func (s *FlexibleString) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) || bytes.Equal(trimmed, []byte(`""`)) {
		*s = ""
		return nil
	}

	var text string
	if err := json.Unmarshal(trimmed, &text); err == nil {
		*s = FlexibleString(text)
		return nil
	}

	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("stake: parse flexible string %q", string(trimmed))
	}

	switch value := value.(type) {
	case json.Number:
		*s = FlexibleString(value.String())
	case bool:
		*s = FlexibleString(strconv.FormatBool(value))
	default:
		return fmt.Errorf("stake: parse flexible string %q", string(trimmed))
	}
	return nil
}

// FlexibleFloat64 decodes JSON numbers or numeric strings.
type FlexibleFloat64 float64

// Float64 returns the value as a float64.
func (f FlexibleFloat64) Float64() float64 {
	return float64(f)
}

// UnmarshalJSON decodes numbers, quoted numbers, empty strings, and null.
func (f *FlexibleFloat64) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) || bytes.Equal(trimmed, []byte(`""`)) {
		*f = 0
		return nil
	}

	var number float64
	if err := json.Unmarshal(trimmed, &number); err == nil {
		*f = FlexibleFloat64(number)
		return nil
	}

	var text string
	if err := json.Unmarshal(trimmed, &text); err != nil {
		return err
	}
	number, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return err
	}
	*f = FlexibleFloat64(number)
	return nil
}

// MarshalJSON encodes the value as a JSON number.
func (f FlexibleFloat64) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatFloat(float64(f), 'f', -1, 64)), nil
}

// FlexibleInt decodes JSON integers, integer-like floats, or numeric strings.
type FlexibleInt int

// Int returns the value as an int.
func (i FlexibleInt) Int() int {
	return int(i)
}

// UnmarshalJSON decodes numbers, quoted numbers, empty strings, and null.
func (i *FlexibleInt) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) || bytes.Equal(trimmed, []byte(`""`)) {
		*i = 0
		return nil
	}

	var number float64
	if err := json.Unmarshal(trimmed, &number); err == nil {
		*i = FlexibleInt(number)
		return nil
	}

	var text string
	if err := json.Unmarshal(trimmed, &text); err != nil {
		return err
	}
	number, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return err
	}
	*i = FlexibleInt(number)
	return nil
}

// MarshalJSON encodes the value as a JSON integer.
func (i FlexibleInt) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Itoa(int(i))), nil
}

// FlexibleTime decodes RFC3339 timestamps, timezone-less Stake timestamps,
// date-only strings, and Unix second/millisecond timestamps.
type FlexibleTime struct {
	time.Time
}

// UnmarshalJSON decodes common Stake timestamp encodings.
func (t *FlexibleTime) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) || bytes.Equal(trimmed, []byte(`""`)) {
		t.Time = time.Time{}
		return nil
	}

	var number float64
	if err := json.Unmarshal(trimmed, &number); err == nil {
		if number > 1_000_000_000_000 {
			t.Time = time.UnixMilli(int64(number)).UTC()
		} else {
			t.Time = time.Unix(int64(number), 0).UTC()
		}
		return nil
	}

	var text string
	if err := json.Unmarshal(trimmed, &text); err != nil {
		return err
	}

	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05.999999",
		"2006-01-02T15:04:05.999",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, text)
		if err == nil {
			t.Time = parsed
			return nil
		}
	}

	return fmt.Errorf("stake: parse time %q", text)
}

// MarshalJSON encodes the value as RFC3339Nano or null when zero.
func (t FlexibleTime) MarshalJSON() ([]byte, error) {
	if t.Time.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.Time.Format(time.RFC3339Nano))
}
