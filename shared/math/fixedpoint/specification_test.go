package fixedpoint_test

import (
	"encoding/json"
	"testing"

	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/math/fixedpoint"
)

// Test_Conversion verifies the lift to fixed-point, the truncating round trip, and the
// whole-number predicate.
func Test_Conversion(t *testing.T) {
	t.Parallel()
	integer := number_from_integer
	if integer(5) != 5*fixedpoint.SCALE {
		t.Fatalf("From_Integer(5) = %d, want %d", integer(5), 5*fixedpoint.SCALE)
	}
	if fixedpoint.Whole(integer(5)) != 5 {
		t.Fatalf("Whole(5.0) = %d, want 5", fixedpoint.Whole(integer(5)))
	}
	// 7/2 = 3.5 truncates toward zero to 3; the negative truncates toward zero to -3.
	if fixedpoint.Whole(integer(7)/2) != 3 {
		t.Fatalf("Whole(3.5) = %d, want 3", fixedpoint.Whole(integer(7)/2))
	}
	if fixedpoint.Whole(integer(-7)/2) != -3 {
		t.Fatalf("Whole(-3.5) = %d, want -3", fixedpoint.Whole(integer(-7)/2))
	}
	if !fixedpoint.Is_Integer(integer(5)) {
		t.Fatal("Is_Integer(5.0) = false, want true")
	}
	if fixedpoint.Is_Integer(integer(7) / 2) {
		t.Fatal("Is_Integer(3.5) = true, want false")
	}
	// 7/2 as a ratio is 3.5; a zero denominator yields zero, not a panic.
	ratio := fixedpoint.From_Ratio(7, 2)
	if ratio != integer(7)/2 {
		t.Fatalf("From_Ratio(7,2) = %d, want %d", ratio, integer(7)/2)
	}
	if fixedpoint.From_Ratio(7, 0) != 0 {
		t.Fatal("From_Ratio(7,0) must be 0")
	}
}

// Test_Arithmetic verifies fixed-point multiply and divide, including the sign, and the
// native add and subtract.
func Test_Arithmetic(t *testing.T) {
	t.Parallel()
	integer := number_from_integer
	product := fixedpoint.Multiply(
		fixedpoint.Multiplicand(integer(3)), fixedpoint.Multiplier(integer(4)),
	)
	if product != integer(12) {
		t.Fatalf("3*4 = %d, want %d", product, integer(12))
	}
	half_times_two := fixedpoint.Multiply(
		fixedpoint.Multiplicand(integer(7)/2), fixedpoint.Multiplier(integer(2)),
	)
	if half_times_two != integer(7) {
		t.Fatalf("3.5*2 = %d, want %d", half_times_two, integer(7))
	}
	negative := fixedpoint.Multiply(
		fixedpoint.Multiplicand(integer(-3)), fixedpoint.Multiplier(integer(4)),
	)
	if negative != integer(-12) {
		t.Fatalf("-3*4 = %d, want %d", negative, integer(-12))
	}
	quotient := fixedpoint.Divide(
		fixedpoint.Dividend(integer(1)), fixedpoint.Divisor(integer(2)),
	)
	if quotient != integer(1)/2 {
		t.Fatalf("1/2 = %d, want %d", quotient, integer(1)/2)
	}
	if integer(2)+integer(3) != integer(5) {
		t.Fatal("2+3 != 5 under native addition")
	}
}

// Test_Ratio verifies that Apply scales a value by a dimensionless fixed-point ratio.
func Test_Ratio(t *testing.T) {
	t.Parallel()
	integer := number_from_integer
	// 1.3 is not exact in base two, so 10*1.3 lands near 13 within a unit or two.
	thirteen_tenths := fixedpoint.Ratio(13 * fixedpoint.SCALE / 10)
	near(t, &near_input{
		Got: fixedpoint.Apply(integer(10), thirteen_tenths), Want: integer(13), Slack: 16,
	})
	// One half is exact, so 10*0.5 is exactly 5.
	halved := fixedpoint.Apply(integer(10), fixedpoint.Ratio(fixedpoint.SCALE/2))
	if halved != integer(5) {
		t.Fatalf("10*0.5 = %d, want %d", halved, integer(5))
	}
}

// Test_Square_Root verifies exact and floored roots, and the scaled root of a plain
// integer sum of squares.
func Test_Square_Root(t *testing.T) {
	t.Parallel()
	integer := number_from_integer
	root_of := integer_root
	if fixedpoint.Number(fixedpoint.Square_Root(integer(4))) != integer(2) {
		t.Fatalf("sqrt(4) = %d, want %d", fixedpoint.Square_Root(integer(4)), integer(2))
	}
	// The root of 2 is irrational; squaring the result returns to 2 within rounding.
	root_two := fixedpoint.Number(fixedpoint.Square_Root(integer(2)))
	near(t, &near_input{
		Got: fixedpoint.Multiply(
			fixedpoint.Multiplicand(root_two), fixedpoint.Multiplier(root_two),
		),
		Want:  integer(2),
		Slack: 16,
	})
	// The sample variance of {10,20,30,40,50} is 250; squaring its root returns to 250.
	root_variance := fixedpoint.Number(fixedpoint.Square_Root_Scaled(250))
	squared := fixedpoint.Multiply(
		fixedpoint.Multiplicand(root_variance), fixedpoint.Multiplier(root_variance),
	)
	near(t, &near_input{Got: squared, Want: integer(250), Slack: 16})
	// Integer_Root floors the root of a 128-bit radicand: a perfect square exactly.
	if root_of(144) != 12 {
		t.Fatalf("Integer_Root(144) = %d, want 12", root_of(144))
	}
	// The fast 64-bit path must floor exactly at every perfect-square boundary.
	for index := 1; index < 200_000; index++ {
		root := uint64(index)
		square := root * root
		if root_of(square) != fixedpoint.Root_Integer(root) {
			t.Fatalf("Integer_Root(%d^2) != %d", root, root)
		}
		if root_of(square-1) != fixedpoint.Root_Integer(root-1) {
			t.Fatalf("Integer_Root(%d^2 - 1) != %d", root, root-1)
		}
		if root_of(square+2*root) != fixedpoint.Root_Integer(root) {
			t.Fatalf("Integer_Root(just below (%d+1)^2) != %d", root, root)
		}
	}
	verify_root_property(t)
}

// Test_Sine verifies the quarter-turn extremes exactly, range reduction past one turn,
// and a midpoint within the approximation's tolerance.
func Test_Sine(t *testing.T) {
	t.Parallel()
	integer := number_from_integer
	quarter := integer(1) / 4
	if fixedpoint.Number(fixedpoint.Sine_Turns(0)) != 0 {
		t.Fatalf("sine(0) = %d, want 0", fixedpoint.Sine_Turns(0))
	}
	if fixedpoint.Number(fixedpoint.Sine_Turns(quarter)) != integer(1) {
		t.Fatalf("sine(0.25) = %d, want %d", fixedpoint.Sine_Turns(quarter), integer(1))
	}
	if fixedpoint.Number(fixedpoint.Sine_Turns(3*quarter)) != integer(-1) {
		t.Fatalf("sine(0.75) = %d, want %d", fixedpoint.Sine_Turns(3*quarter), integer(-1))
	}
	// A 1.25 turn reduces to 0.25, and -0.25 reduces to 0.75.
	full_turn := fixedpoint.Number(fixedpoint.Sine_Turns(integer(1) + quarter))
	if full_turn != integer(1) {
		t.Fatalf("sine(1.25) = %d, want %d", full_turn, integer(1))
	}
	if fixedpoint.Number(fixedpoint.Sine_Turns(-quarter)) != integer(-1) {
		t.Fatalf("sine(-0.25) = %d, want %d", fixedpoint.Sine_Turns(-quarter), integer(-1))
	}
	// Sine of 1/12 turn equals sine of 30 degrees, 0.5, held within a small slack.
	midpoint := fixedpoint.Number(fixedpoint.Sine_Turns(integer(1) / 12))
	near(t, &near_input{Got: midpoint, Want: integer(1) / 2, Slack: 2000})
}

// Test_Format verifies decimal rendering, the sign, half-away rounding, and carry.
func Test_Format(t *testing.T) {
	t.Parallel()
	integer := number_from_integer
	cases := []format_case{
		{Value: integer(5), Digits: 2, Want: "5.00"},
		{Value: integer(1) / 2, Digits: 2, Want: "0.50"},
		{Value: integer(3) / 2, Digits: 2, Want: "1.50"},
		{Value: -integer(3) / 2, Digits: 2, Want: "-1.50"},
		{Value: integer(1) - integer(1)/1000, Digits: 2, Want: "1.00"},
		{Value: integer(3) / 2, Digits: 0, Want: "2"},
	}
	for _, one := range cases {
		got := fixedpoint.Format(one.Value, one.Digits)
		if got != one.Want {
			t.Errorf("Format %d/%d = %q want %q", one.Value, one.Digits, got, one.Want)
		}
	}
}

// Test_Serialization verifies a Number round-trips through JSON as a bare decimal.
func Test_Serialization(t *testing.T) {
	t.Parallel()
	integer := number_from_integer
	encoded, err := json.Marshal(holder{Value: integer(5) / 2})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(encoded) != `{"Value":2.5}` {
		t.Fatalf("Marshal(2.5) = %s, want {\"Value\":2.5}", encoded)
	}
	whole, err := json.Marshal(holder{Value: integer(3)})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(whole) != `{"Value":3}` {
		t.Fatalf("Marshal(3.0) = %s, want {\"Value\":3}", whole)
	}
	var decoded holder
	err = json.Unmarshal([]byte(`{"Value":2.5}`), &decoded)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.Value != integer(5)/2 {
		t.Fatalf("Unmarshal(2.5) = %d, want %d", decoded.Value, integer(5)/2)
	}
}

// Test_Invariant_Domains verifies the special values of each admitted scalar domain.
func Test_Invariant_Domains(t *testing.T) {
	t.Parallel()
	verify_number_domains()
	verify_integer_domains()
	verify_format_domains()
}

// Wraps a Number so the JSON test exercises the marshaler through a struct field.
type holder struct {
	// Value is the wrapped fixed-point number.
	Value fixedpoint.Number
}

// One Format expectation for the table in Test_Format.
type format_case struct {
	// Value is the fixed-point input.
	Value fixedpoint.Number
	// Digits is the requested count of fractional digits.
	Digits fixedpoint.Digit_Count
	// Want is the expected rendering.
	Want fixedpoint.Text
}

// Bundles the operands of near, which repeat a type.
type near_input struct {
	// Got is the produced value.
	Got fixedpoint.Number
	// Want is the expected value.
	Want fixedpoint.Number
	// Slack is the tolerated absolute difference.
	Slack fixedpoint.Number
}

// Fails when Got and Want differ by more than Slack fixed-point units, for the
// approximate results sine produces away from the exact quarter turns.
func near(t *testing.T, input *near_input) {
	t.Helper()
	difference := input.Got - input.Want
	if difference < 0 {
		difference = -difference
	}
	if difference > input.Slack {
		t.Errorf("got %d, want %d within %d", input.Got, input.Want, input.Slack)
	}
}

// Returns the fixedpoint root of one 64-bit radicand.
func integer_root(radicand uint64) (floored fixedpoint.Root_Integer) {
	return fixedpoint.Integer_Root(0, fixedpoint.Low_Word(radicand))
}

// Converts one whole integer to the general fixed-point arithmetic type.
func number_from_integer(value fixedpoint.Whole_Integer) (number fixedpoint.Number) {
	return fixedpoint.Number(fixedpoint.From_Integer(value))
}

// Exercises each signed special value at the fixed-point arithmetic boundaries.
func verify_number_domains() {
	values := [...]int64{
		bits.INTEGER_64_MINIMUM,
		bits.INTEGER_64_MAXIMUM,
		0,
		1,
		2,
		-1,
	}
	for _, value := range values {
		number := fixedpoint.Number(value)
		fixedpoint.Apply(number, fixedpoint.Ratio(fixedpoint.SCALE))
		fixedpoint.Apply(0, fixedpoint.Ratio(value))
		fixedpoint.Multiply(
			fixedpoint.Multiplicand(number), fixedpoint.Multiplier(fixedpoint.SCALE),
		)
		fixedpoint.Multiply(0, fixedpoint.Multiplier(number))
		fixedpoint.Divide(fixedpoint.Dividend(number), fixedpoint.Divisor(fixedpoint.SCALE))
		fixedpoint.Divide(0, fixedpoint.Divisor(number))
		fixedpoint.Format(number, 0)
		fixedpoint.Is_Integer(number)
		fixedpoint.Sine_Turns(number)
		fixedpoint.Square_Root(number)
		fixedpoint.Whole(number)
	}
	for _, value := range values[2:] {
		fixedpoint.Whole(fixedpoint.Number(value * fixedpoint.SCALE))
	}
}

// Exercises each integer special value at the conversion and root boundaries.
func verify_integer_domains() {
	whole_values := [...]fixedpoint.Whole_Integer{
		fixedpoint.Whole_Integer(fixedpoint.WHOLE_INTEGER_MINIMUM),
		fixedpoint.Whole_Integer(fixedpoint.WHOLE_INTEGER_MAXIMUM),
		0,
		1,
		2,
		-1,
	}
	for _, value := range whole_values {
		fixedpoint.From_Integer(value)
	}
	values := [...]int64{
		bits.INTEGER_64_MINIMUM,
		bits.INTEGER_64_MAXIMUM,
		0,
		1,
		2,
		-1,
	}
	for _, value := range values {
		fixedpoint.From_Ratio(fixedpoint.Numerator(value), fixedpoint.SCALE)
		fixedpoint.From_Ratio(0, fixedpoint.Denominator(value))
		fixedpoint.Square_Root_Scaled(fixedpoint.Radicand(value))
	}
	fixedpoint.Integer_Root(0, 0)
	fixedpoint.Integer_Root(0, 1)
	fixedpoint.Integer_Root(0, 2)
	fixedpoint.Integer_Root(0, 4)
	fixedpoint.Integer_Root(0, fixedpoint.Low_Word(bits.WORD_64_MAXIMUM))
	fixedpoint.Integer_Root(1, 0)
	fixedpoint.Integer_Root(2, 0)
	fixedpoint.Integer_Root(fixedpoint.High_Word(fixedpoint.HIGH_WORD_MAXIMUM), 0)
	fixedpoint.Integer_Root(
		fixedpoint.High_Word(fixedpoint.HIGH_WORD_MAXIMUM),
		fixedpoint.Low_Word(bits.WORD_64_MAXIMUM),
	)
}

// Exercises each precision boundary and each special formatted text size.
func verify_format_domains() {
	fixedpoint.Format(0, 0)
	fixedpoint.Format(number_from_integer(10), 0)
	fixedpoint.Format(0, 1)
	fixedpoint.Format(0, 2)
	fixedpoint.Format(fixedpoint.Number(bits.INTEGER_64_MINIMUM), 6)
}

// Verifies that the divide-free root satisfies its defining property across large,
// scattered radicands: r^2 <= n < (r+1)^2, checked in 128 bits.
func verify_root_property(t *testing.T) {
	t.Helper()
	root_of := integer_root
	scatter := uint64(0x9E3779B97F4A7C15)
	probe := uint64(1)
	for index := 0; index < 500_000; index++ {
		probe += scatter
		root := uint64(root_of(probe))
		low_high, low_low := bits.Multiply_64(bits.Word_64(root), bits.Multiplier_64(root))
		if low_high != 0 {
			t.Fatalf("Integer_Root(%d) = %d, but r^2 overflows past n", probe, root)
		}
		if uint64(low_low) > probe {
			t.Fatalf("Integer_Root(%d) = %d, but r^2 > n", probe, root)
		}
		next_root := root + 1
		next_high, next_low := bits.Multiply_64(
			bits.Word_64(next_root), bits.Multiplier_64(next_root))
		if next_high == 0 {
			if uint64(next_low) <= probe {
				t.Fatalf("Integer_Root(%d) = %d, but (r+1)^2 <= n", probe, root)
			}
		}
	}
	// A large 128-bit radicand floors exactly too: (3<<40)^2 roots back to 3<<40.
	big_root := uint64(3) << 40
	big_high, big_low := bits.Multiply_64(
		bits.Word_64(big_root), bits.Multiplier_64(big_root))
	big := fixedpoint.Integer_Root(fixedpoint.High_Word(big_high), fixedpoint.Low_Word(big_low))
	if big != fixedpoint.Root_Integer(big_root) {
		t.Fatalf("Integer_Root((3<<40)^2) != 3<<40")
	}
}
