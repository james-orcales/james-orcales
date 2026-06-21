package fixedpoint_test

import (
	"encoding/json"
	"math/bits"
	"testing"

	"github.com/james-orcales/james-orcales/shared/fixedpoint"
)

// Test_Conversion verifies the lift to fixed-point, the truncating round trip, and the
// whole-number predicate.
func Test_Conversion(t *testing.T) {
	t.Parallel()
	integer := fixedpoint.From_Integer
	if integer(5) != 5*fixedpoint.Scale {
		t.Fatalf("From_Integer(5) = %d, want %d", integer(5), 5*fixedpoint.Scale)
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
	ratio := fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{Numerator: 7, Denominator: 2})
	if ratio != integer(7)/2 {
		t.Fatalf("From_Ratio(7,2) = %d, want %d", ratio, integer(7)/2)
	}
	if fixedpoint.From_Ratio(&fixedpoint.From_Ratio_Input{Numerator: 7, Denominator: 0}) != 0 {
		t.Fatal("From_Ratio(7,0) must be 0")
	}
}

// Test_Arithmetic verifies fixed-point multiply and divide, including the sign, and the
// native add and subtract.
func Test_Arithmetic(t *testing.T) {
	t.Parallel()
	integer := fixedpoint.From_Integer
	product := fixedpoint.Multiply(&fixedpoint.Multiply_Input{A: integer(3), B: integer(4)})
	if product != integer(12) {
		t.Fatalf("3*4 = %d, want %d", product, integer(12))
	}
	half_times_two := fixedpoint.Multiply(&fixedpoint.Multiply_Input{
		A: integer(7) / 2, B: integer(2),
	})
	if half_times_two != integer(7) {
		t.Fatalf("3.5*2 = %d, want %d", half_times_two, integer(7))
	}
	negative := fixedpoint.Multiply(&fixedpoint.Multiply_Input{A: integer(-3), B: integer(4)})
	if negative != integer(-12) {
		t.Fatalf("-3*4 = %d, want %d", negative, integer(-12))
	}
	quotient := fixedpoint.Divide(&fixedpoint.Divide_Input{
		Dividend: integer(1), Divisor: integer(2),
	})
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
	integer := fixedpoint.From_Integer
	// 1.3 is not exact in base two, so 10*1.3 lands near 13 within a unit or two.
	thirteen_tenths := fixedpoint.Ratio(13 * fixedpoint.Scale / 10)
	near(t, &near_input{
		Got: fixedpoint.Apply(integer(10), thirteen_tenths), Want: integer(13), Slack: 16,
	})
	// One half is exact, so 10*0.5 is exactly 5.
	halved := fixedpoint.Apply(integer(10), fixedpoint.Ratio(fixedpoint.Scale/2))
	if halved != integer(5) {
		t.Fatalf("10*0.5 = %d, want %d", halved, integer(5))
	}
}

// Test_Square_Root verifies exact and floored roots, and the scaled root of a plain
// integer sum of squares.
func Test_Square_Root(t *testing.T) {
	t.Parallel()
	integer := fixedpoint.From_Integer
	root_of := func(radicand uint64) (floored int64) {
		return fixedpoint.Integer_Root(&fixedpoint.Integer_Root_Input{Low: radicand})
	}
	if fixedpoint.Square_Root(integer(4)) != integer(2) {
		t.Fatalf("sqrt(4) = %d, want %d", fixedpoint.Square_Root(integer(4)), integer(2))
	}
	// The root of 2 is irrational; squaring the result returns to 2 within rounding.
	root_two := fixedpoint.Square_Root(integer(2))
	near(t, &near_input{
		Got:   fixedpoint.Multiply(&fixedpoint.Multiply_Input{A: root_two, B: root_two}),
		Want:  integer(2),
		Slack: 16,
	})
	// The sample variance of {10,20,30,40,50} is 250; squaring its root returns to 250.
	root_variance := fixedpoint.Square_Root_Scaled(250)
	squared := fixedpoint.Multiply(&fixedpoint.Multiply_Input{
		A: root_variance, B: root_variance,
	})
	near(t, &near_input{Got: squared, Want: integer(250), Slack: 16})
	// Integer_Root floors the root of a 128-bit radicand: a perfect square exactly.
	if root_of(144) != 12 {
		t.Fatalf("Integer_Root(144) = %d, want 12", root_of(144))
	}
	// The fast 64-bit path must floor exactly at every perfect-square boundary.
	for index := 1; index < 200_000; index++ {
		root := uint64(index)
		square := root * root
		if root_of(square) != int64(root) {
			t.Fatalf("Integer_Root(%d^2) != %d", root, root)
		}
		if root_of(square-1) != int64(root-1) {
			t.Fatalf("Integer_Root(%d^2 - 1) != %d", root, root-1)
		}
		if root_of(square+2*root) != int64(root) {
			t.Fatalf("Integer_Root(just below (%d+1)^2) != %d", root, root)
		}
	}
	// Across large, scattered 64-bit radicands the divide-free root must satisfy the
	// defining property exactly: r^2 <= n < (r+1)^2, checked in 128 bits.
	scatter := uint64(0x9E3779B97F4A7C15)
	probe := uint64(1)
	for index := 0; index < 500_000; index++ {
		probe += scatter
		root := uint64(root_of(probe))
		low_high, low_low := bits.Mul64(root, root)
		if low_high != 0 {
			t.Fatalf("Integer_Root(%d) = %d, but r^2 overflows past n", probe, root)
		}
		if low_low > probe {
			t.Fatalf("Integer_Root(%d) = %d, but r^2 > n", probe, root)
		}
		next_high, next_low := bits.Mul64(root+1, root+1)
		if next_high == 0 {
			if next_low <= probe {
				t.Fatalf("Integer_Root(%d) = %d, but (r+1)^2 <= n", probe, root)
			}
		}
	}
	// A large 128-bit radicand floors exactly too: (3<<40)^2 roots back to 3<<40.
	big_root := uint64(3) << 40
	big_high, big_low := bits.Mul64(big_root, big_root)
	big := fixedpoint.Integer_Root(&fixedpoint.Integer_Root_Input{High: big_high, Low: big_low})
	if big != int64(big_root) {
		t.Fatalf("Integer_Root((3<<40)^2) != 3<<40")
	}
}

// Test_Sine verifies the quarter-turn extremes exactly, range reduction past one turn,
// and a midpoint within the approximation's tolerance.
func Test_Sine(t *testing.T) {
	t.Parallel()
	integer := fixedpoint.From_Integer
	quarter := integer(1) / 4
	if fixedpoint.Sine_Turns(0) != 0 {
		t.Fatalf("sine(0) = %d, want 0", fixedpoint.Sine_Turns(0))
	}
	if fixedpoint.Sine_Turns(quarter) != integer(1) {
		t.Fatalf("sine(0.25) = %d, want %d", fixedpoint.Sine_Turns(quarter), integer(1))
	}
	if fixedpoint.Sine_Turns(3*quarter) != integer(-1) {
		t.Fatalf("sine(0.75) = %d, want %d", fixedpoint.Sine_Turns(3*quarter), integer(-1))
	}
	// A 1.25 turn reduces to 0.25, and -0.25 reduces to 0.75.
	full_turn := fixedpoint.Sine_Turns(integer(1) + quarter)
	if full_turn != integer(1) {
		t.Fatalf("sine(1.25) = %d, want %d", full_turn, integer(1))
	}
	if fixedpoint.Sine_Turns(-quarter) != integer(-1) {
		t.Fatalf("sine(-0.25) = %d, want %d", fixedpoint.Sine_Turns(-quarter), integer(-1))
	}
	// Sine of 1/12 turn equals sine of 30 degrees, 0.5, held within a small slack.
	midpoint := fixedpoint.Sine_Turns(integer(1) / 12)
	near(t, &near_input{Got: midpoint, Want: integer(1) / 2, Slack: 2000})
}

// Test_Format verifies decimal rendering, the sign, half-away rounding, and carry.
func Test_Format(t *testing.T) {
	t.Parallel()
	integer := fixedpoint.From_Integer
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
	integer := fixedpoint.From_Integer
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
	Digits int
	// Want is the expected rendering.
	Want string
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
