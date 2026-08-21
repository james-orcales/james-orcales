package fixedpoint

import (
	"testing"
)

// LARGEST_UNSIGNED_TEXT is the decimal text of the largest unsigned value, twenty digits.
const LARGEST_UNSIGNED_TEXT = "18446744073709551615"

// LARGEST_SIGNED_TEXT is the decimal text of the largest signed value, nineteen digits.
const LARGEST_SIGNED_TEXT = "9223372036854775807"

// Test_Decimal_Text drives the digit writer over the ends of its domain. This package sits
// beneath the shared strconv, which reaches back here, thus it writes its own digits and
// this test stands in for that package's coverage.
func Test_Decimal_Text(t *testing.T) {
	t.Parallel()
	for _, one := range []struct {
		Value Unsigned_Value
		Want  string
	}{
		{Value: 0, Want: "0"},
		{Value: 1, Want: "1"},
		{Value: 2, Want: "2"},
		{Value: 9, Want: "9"},
		{Value: 10, Want: "10"},
		{Value: 18446744073709551615, Want: LARGEST_UNSIGNED_TEXT},
	} {
		var storage [DECIMAL_TEXT_SIZE_MAXIMUM]byte
		count := decimal_digit_count(one.Value)
		decimal_into(storage[:count], one.Value)
		got := string(storage[:count])
		if got != one.Want {
			t.Fatalf("decimal_into(%d) = %q, want %q", one.Value, got, one.Want)
		}
	}
}

// Test_Decimal_Value drives the digit reader over the ends of its domain, including the
// empty text and a text that carries a byte which is not a digit.
func Test_Decimal_Value(t *testing.T) {
	t.Parallel()
	for _, one := range []struct {
		Text  Decimal_Text
		Want  Decimal_Value
		Valid bool
	}{
		{Text: "", Want: 0, Valid: false},
		{Text: "x", Want: 0, Valid: false},
		{Text: "1x", Want: 0, Valid: false},
		{Text: "-1", Want: 0, Valid: false},
		{Text: "0", Want: 0, Valid: true},
		{Text: "1", Want: 1, Valid: true},
		{Text: "2", Want: 2, Valid: true},
		{Text: "10", Want: 10, Valid: true},
		{Text: LARGEST_SIGNED_TEXT, Want: 9223372036854775807, Valid: true},
		// Twenty digits is the widest text the parse reads, and it passes the signed
		// range, thus the parse rejects it rather than returning a wrapped value.
		{Text: LARGEST_UNSIGNED_TEXT, Want: 0, Valid: false},
		{Text: "9999999999999999999", Want: 0, Valid: false},
	} {
		got, valid := decimal_value(one.Text)
		if bool(valid) != one.Valid {
			t.Fatalf("decimal_value(%q) reports %v", one.Text, valid)
		}
		if got != one.Want {
			t.Fatalf("decimal_value(%q) = %d, want %d", one.Text, got, one.Want)
		}
	}
}

// Test_Decimal_Round_Trip requires the reader to return what the writer produced, over the
// values that bound the two domains.
func Test_Decimal_Round_Trip(t *testing.T) {
	t.Parallel()
	for _, value := range []Unsigned_Value{0, 1, 2, 9, 10, 9223372036854775807} {
		var storage [DECIMAL_TEXT_SIZE_MAXIMUM]byte
		count := decimal_digit_count(value)
		decimal_into(storage[:count], value)
		text := string(storage[:count])
		returned, valid := decimal_value(Decimal_Text(text))
		if !valid {
			t.Fatalf("decimal_value rejected the text decimal_into wrote for %d", value)
		}
		if Unsigned_Value(returned) != value {
			t.Fatalf("the round trip of %d gave %d", value, returned)
		}
	}
}
