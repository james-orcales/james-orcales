// Package fixedpoint is a base-two fixed-point number backed by int64, the deterministic
// stand-in for float64 in packages the linter holds reproducible. Float arithmetic
// diverges across platforms; an integer scaled by a fixed power of two does not. The
// scale is a power of two so the rescale every multiply and divide needs is a bit shift,
// not the hardware divide a base-ten scale would force. The linter bans methods, so every
// operation is a free function named for what it does, with the value it acts on first.
package fixedpoint

import (
	"strconv"
	"strings"

	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
)

// FRACTIONAL_BITS is how many of a Number's low bits hold the fraction; the rest hold the
// integer part. Twenty bits gives ~9.5e-7 precision over a ±8.8e12 range, matching the
// magnitudes the deterministic callers work in.
const FRACTIONAL_BITS = 20

// SCALE is the count of fixed-point units in one whole — two raised to FRACTIONAL_BITS.
// A Number's real value is its stored integer divided by SCALE.
const SCALE = 1 << FRACTIONAL_BITS

// HIGH_WORD_MAXIMUM keeps the 128-bit root in the signed 64-bit result domain.
const HIGH_WORD_MAXIMUM uint64 = 1<<56 - 1

// DIGIT_COUNT_MINIMUM is the smallest decimal precision.
const DIGIT_COUNT_MINIMUM = 0

// DIGIT_COUNT_MAXIMUM prevents an int64 power of ten from overflowing.
const DIGIT_COUNT_MAXIMUM = 6

// TEXT_SIZE_MINIMUM is one decimal digit.
const TEXT_SIZE_MINIMUM = 1

// TEXT_SIZE_MAXIMUM holds a sign, an integer part, a point, and six fraction digits.
const TEXT_SIZE_MAXIMUM = 21

// WHOLE_INTEGER_MINIMUM is the smallest integer that fixed-point storage can lift.
const WHOLE_INTEGER_MINIMUM int64 = -8796093022208

// WHOLE_INTEGER_MAXIMUM is the largest integer that fixed-point storage can lift.
const WHOLE_INTEGER_MAXIMUM int64 = 8796093022207

// INTEGER_NUMBER_MINIMUM is the smallest fixed-point integer.
const INTEGER_NUMBER_MINIMUM int64 = bits.INTEGER_64_MINIMUM

// INTEGER_NUMBER_MAXIMUM is the largest fixed-point integer.
const INTEGER_NUMBER_MAXIMUM int64 = 9223372036853727232

// NUMBER_ROOT_MINIMUM is the smallest root of a Number.
const NUMBER_ROOT_MINIMUM int64 = 0

// NUMBER_ROOT_MAXIMUM is the largest root of a Number.
const NUMBER_ROOT_MAXIMUM int64 = 3109888511975

// SCALED_ROOT_MINIMUM is the smallest scaled root of an integer.
const SCALED_ROOT_MINIMUM int64 = 0

// SCALED_ROOT_MAXIMUM is the largest scaled root of an integer.
const SCALED_ROOT_MAXIMUM int64 = 3184525836262886

// ROOT_INTEGER_MINIMUM is the smallest integer root.
const ROOT_INTEGER_MINIMUM int64 = 0

// ROOT_INTEGER_MAXIMUM is the largest root in the admitted 128-bit domain.
const ROOT_INTEGER_MAXIMUM int64 = 1152921504606846975

// SINE_MINIMUM is the lower bound of a sine.
const SINE_MINIMUM int64 = -SCALE

// SINE_MAXIMUM is the upper bound of a sine.
const SINE_MAXIMUM int64 = SCALE

// FRACTION_UNIT_ONE is one fixed-point fraction unit.
const FRACTION_UNIT_ONE int64 = 1

// FRACTION_UNIT_TWO is two fixed-point fraction units.
const FRACTION_UNIT_TWO int64 = 2

// FRACTION_UNIT_NEGATIVE_ONE is negative one fixed-point fraction unit.
const FRACTION_UNIT_NEGATIVE_ONE int64 = -1

// Number is a base-two fixed-point value. Addition, subtraction, negation, and the
// ordering comparisons are the native int64 operators, the shared SCALE aligning them;
// multiplication and division need the functions below because the scale must cancel.
type Number int64

// Number_Invariants states the complete fixed-point storage domain.
func Number_Invariants(value Number, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Ratio is a dimensionless fixed-point multiplier denoting ratio divided by SCALE. It
// is a type distinct from Number so Apply takes one of each — sidestepping the
// input-struct rule two Numbers would trip — and so a ratio reads as a plain constant.
type Ratio int64

// Ratio_Invariants states the complete ratio storage domain.
func Ratio_Invariants(value Ratio, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Numerator is the dividend of a ratio.
type Numerator int64

// Numerator_Invariants states the complete signed 64-bit domain.
func Numerator_Invariants(value Numerator, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Denominator is the divisor of a ratio.
type Denominator int64

// Denominator_Invariants states the complete signed 64-bit domain.
func Denominator_Invariants(value Denominator, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Multiplicand is the first fixed-point factor.
type Multiplicand Number

// Multiplicand_Invariants states the complete fixed-point storage domain.
func Multiplicand_Invariants(value Multiplicand, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Multiplier is the second fixed-point factor.
type Multiplier Number

// Multiplier_Invariants states the complete fixed-point storage domain.
func Multiplier_Invariants(value Multiplier, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Dividend is the fixed-point value that a divisor divides.
type Dividend Number

// Dividend_Invariants states the complete fixed-point storage domain.
func Dividend_Invariants(value Dividend, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Divisor is the fixed-point value that divides a dividend.
type Divisor Number

// Divisor_Invariants states the complete fixed-point storage domain.
func Divisor_Invariants(value Divisor, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Whole_Integer is an integer that fixed-point storage can lift without overflow.
type Whole_Integer int64

// Whole_Integer_Invariants bounds an integer to the fixed-point whole-number domain.
func Whole_Integer_Invariants(value Whole_Integer, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), WHOLE_INTEGER_MINIMUM, WHOLE_INTEGER_MAXIMUM).
		Ensure()
}

// Integer_Number is a fixed-point Number with no fractional units.
type Integer_Number Number

// Integer_Number_Invariants bounds the fixed-point whole-number domain.
func Integer_Number_Invariants(value Integer_Number, namespace invariant.Namespace) {
	invariant.Always(
		int64(value)%SCALE == 0,
		"An Integer_Number has no fractional units.",
	)
	invariant.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value), INTEGER_NUMBER_MINIMUM, INTEGER_NUMBER_MAXIMUM,
			FRACTION_UNIT_NEGATIVE_ONE, FRACTION_UNIT_ONE,
			FRACTION_UNIT_TWO, FRACTION_UNIT_TWO,
		).
		Ensure()
}

// Radicand is a signed 64-bit integer before a scaled square-root operation.
type Radicand int64

// Radicand_Invariants states the complete signed 64-bit domain.
func Radicand_Invariants(value Radicand, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Number_Root is a nonnegative square root of a Number.
type Number_Root Number

// Number_Root_Invariants bounds a root to the Number radicand domain.
func Number_Root_Invariants(value Number_Root, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value), NUMBER_ROOT_MINIMUM, NUMBER_ROOT_MAXIMUM,
			FRACTION_UNIT_ONE, FRACTION_UNIT_TWO,
			FRACTION_UNIT_TWO, FRACTION_UNIT_TWO,
		).
		Ensure()
}

// Scaled_Root is a fixed-point square root of an integer radicand.
type Scaled_Root Number

// Scaled_Root_Invariants bounds a root to the signed integer radicand domain.
func Scaled_Root_Invariants(value Scaled_Root, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value), SCALED_ROOT_MINIMUM, SCALED_ROOT_MAXIMUM,
			FRACTION_UNIT_ONE, FRACTION_UNIT_TWO,
			FRACTION_UNIT_TWO, FRACTION_UNIT_TWO,
		).
		Ensure()
}

// Root_Integer is a nonnegative integer root of a 128-bit radicand.
type Root_Integer int64

// Root_Integer_Invariants bounds a root to the admitted 128-bit radicand domain.
func Root_Integer_Invariants(value Root_Integer, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), ROOT_INTEGER_MINIMUM, ROOT_INTEGER_MAXIMUM).
		Ensure()
}

// Sine is a fixed-point sine in the closed interval from negative one to one.
type Sine Number

// Sine_Invariants bounds a sine to the unit interval.
func Sine_Invariants(value Sine, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value), SINE_MINIMUM, SINE_MAXIMUM,
			FRACTION_UNIT_NEGATIVE_ONE, FRACTION_UNIT_ONE,
			FRACTION_UNIT_TWO, FRACTION_UNIT_TWO,
		).
		Ensure()
}

// High_Word is the upper word of a 128-bit integer.
type High_Word uint64

// High_Word_Invariants keeps the root in the signed 64-bit result domain.
func High_Word_Invariants(value High_Word, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, HIGH_WORD_MAXIMUM).
		Ensure()
}

// Low_Word is the lower word of a 128-bit integer.
type Low_Word uint64

// Low_Word_Invariants states the complete unsigned 64-bit domain.
func Low_Word_Invariants(value Low_Word, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Digit_Count is a count of decimal fraction digits.
type Digit_Count int

// Digit_Count_Invariants keeps decimal scaling in the signed 64-bit domain.
func Digit_Count_Invariants(value Digit_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), DIGIT_COUNT_MINIMUM, DIGIT_COUNT_MAXIMUM).
		Ensure()
}

// Boolean is one true or false value.
type Boolean bool

// Boolean_Invariants records both Boolean states.
func Boolean_Invariants(value Boolean, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "A Boolean value is true.").
		Ensure()
}

// Text is decimal text.
type Text string

// Text_Invariants bounds a formatted fixed-point number.
func Text_Invariants(value Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// From_Integer lifts a whole number into fixed-point.
func From_Integer(value Whole_Integer) (number Integer_Number) {
	defer func() { Integer_Number_Invariants(number, "from_integer.number") }()
	Whole_Integer_Invariants(value, "from_integer.value")
	return Integer_Number(value * SCALE)
}

// From_Ratio lifts the quotient numerator/denominator into fixed-point, the scaled-up
// numerator taken through a 128-bit intermediate so it cannot overflow and a large
// integer mean keeps its fraction. A zero denominator yields zero, not a divide by zero.
func From_Ratio(numerator Numerator, denominator Denominator) (number Number) {
	defer func() { Number_Invariants(number, "from_ratio.number") }()
	Numerator_Invariants(numerator, "from_ratio.numerator")
	Denominator_Invariants(denominator, "from_ratio.denominator")
	if denominator == 0 {
		return 0
	}
	negative := (numerator < 0) != (denominator < 0)
	numerator_magnitude := uint64(numerator)
	if numerator < 0 {
		numerator_magnitude = uint64(-int64(numerator))
	}
	denominator_magnitude := uint64(denominator)
	if denominator < 0 {
		denominator_magnitude = uint64(-int64(denominator))
	}
	high := numerator_magnitude >> (64 - FRACTIONAL_BITS)
	low := numerator_magnitude << FRACTIONAL_BITS
	magnitude, _ := bits.Divide_64(bits.Dividend_High_64(high),
		bits.Dividend_Low_64(low), bits.Divisor_64(denominator_magnitude))
	if negative {
		return Number(-int64(magnitude))
	}
	return Number(magnitude)
}

// Whole truncates a fixed-point value toward zero to the integer it contains.
func Whole(value Number) (whole Whole_Integer) {
	defer func() { Whole_Integer_Invariants(whole, "whole.whole") }()
	Number_Invariants(value, "whole.value")
	return Whole_Integer(int64(value) / SCALE)
}

// Is_Integer reports whether a value carries no fractional part.
func Is_Integer(value Number) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_integer.yes") }()
	Number_Invariants(value, "is_integer.value")
	return Boolean(int64(value)%SCALE == 0)
}

// Multiply returns the fixed-point product: the 128-bit product of the factors shifted
// back down by the scale, so no divide and no premature overflow.
func Multiply(multiplicand Multiplicand, multiplier Multiplier) (product Number) {
	defer func() { Number_Invariants(product, "multiply.product") }()
	Multiplicand_Invariants(multiplicand, "multiply.multiplicand")
	Multiplier_Invariants(multiplier, "multiply.multiplier")
	negative := (multiplicand < 0) != (multiplier < 0)
	left := uint64(multiplicand)
	if multiplicand < 0 {
		left = uint64(-int64(multiplicand))
	}
	right := uint64(multiplier)
	if multiplier < 0 {
		right = uint64(-int64(multiplier))
	}
	product_high, product_low := bits.Multiply_64(
		bits.Word_64(left), bits.Multiplier_64(right))
	high := uint64(product_high)
	low := uint64(product_low)
	magnitude := (high << (64 - FRACTIONAL_BITS)) | (low >> FRACTIONAL_BITS)
	if negative {
		return Number(-int64(magnitude))
	}
	return Number(magnitude)
}

// Divide returns the fixed-point quotient: the dividend scaled up by a shift, then one
// hardware divide by the runtime divisor. A zero divisor yields zero.
func Divide(dividend Dividend, divisor Divisor) (quotient Number) {
	defer func() { Number_Invariants(quotient, "divide.quotient") }()
	Dividend_Invariants(dividend, "divide.dividend")
	Divisor_Invariants(divisor, "divide.divisor")
	if divisor == 0 {
		return 0
	}
	negative := (dividend < 0) != (divisor < 0)
	numerator := uint64(dividend)
	if dividend < 0 {
		numerator = uint64(-int64(dividend))
	}
	denominator := uint64(divisor)
	if divisor < 0 {
		denominator = uint64(-int64(divisor))
	}
	high := numerator >> (64 - FRACTIONAL_BITS)
	low := numerator << FRACTIONAL_BITS
	magnitude, _ := bits.Divide_64(bits.Dividend_High_64(high),
		bits.Dividend_Low_64(low), bits.Divisor_64(denominator))
	if negative {
		return Number(-int64(magnitude))
	}
	return Number(magnitude)
}

// Apply scales a value by a dimensionless ratio.
func Apply(value Number, ratio Ratio) (scaled Number) {
	defer func() { Number_Invariants(scaled, "apply.scaled") }()
	Number_Invariants(value, "apply.value")
	Ratio_Invariants(ratio, "apply.ratio")
	negative := (value < 0) != (ratio < 0)
	left := uint64(value)
	if value < 0 {
		left = uint64(-int64(value))
	}
	right := uint64(ratio)
	if ratio < 0 {
		right = uint64(-int64(ratio))
	}
	product_high, product_low := bits.Multiply_64(
		bits.Word_64(left), bits.Multiplier_64(right))
	high := uint64(product_high)
	low := uint64(product_low)
	magnitude := (high << (64 - FRACTIONAL_BITS)) | (low >> FRACTIONAL_BITS)
	if negative {
		return Number(-int64(magnitude))
	}
	return Number(magnitude)
}

// Square_Root returns the fixed-point square root of a fixed-point value. A negative
// input has no real root and yields zero.
func Square_Root(value Number) (root Number_Root) {
	defer func() { Number_Root_Invariants(root, "square_root.root") }()
	Number_Invariants(value, "square_root.value")
	if value < 0 {
		return 0
	}
	// The root of value/SCALE, scaled back up, is the root of value*SCALE; the product
	// can exceed int64, so it is taken as a 128-bit radicand.
	high, low := bits.Multiply_64(bits.Word_64(value), SCALE)
	return Number_Root(Integer_Root(High_Word(high), Low_Word(low)))
}

// Square_Root_Scaled returns the fixed-point square root of a plain integer, for a sum
// of squares whose magnitude would overflow if first lifted into fixed-point. A
// negative input yields zero.
func Square_Root_Scaled(value Radicand) (root Scaled_Root) {
	defer func() { Scaled_Root_Invariants(root, "square_root_scaled.root") }()
	Radicand_Invariants(value, "square_root_scaled.value")
	if value < 0 {
		return 0
	}
	// The root of value, scaled up by SCALE, is the root of value*SCALE*SCALE.
	high, low := bits.Multiply_64(bits.Word_64(value), SCALE*SCALE)
	return Scaled_Root(Integer_Root(High_Word(high), Low_Word(low)))
}

// Integer_Root returns the floor of the square root of a 128-bit radicand — the integer
// primitive the fixed-point roots build on, and the escape hatch for a sum of squares too
// large to lift into fixed-point first.
func Integer_Root(high High_Word, low Low_Word) (root Root_Integer) {
	defer func() { Root_Integer_Invariants(root, "integer_root.root") }()
	High_Word_Invariants(high, "integer_root.high")
	Low_Word_Invariants(low, "integer_root.low")
	if high == 0 {
		if low == 0 {
			return 0
		}
		size := int(bits.Bit_Size_64(bits.Word_64(low)))
		estimate := uint64(1) << uint((size+1)/2)
		for index := 0; index < 64; index++ {
			next := (estimate + uint64(low)/estimate) / 2
			if next >= estimate {
				return Root_Integer(estimate)
			}
			estimate = next
		}
		return Root_Integer(estimate)
	}
	leading_zeros := int(bits.Leading_Zeros_64(bits.Word_64(high)))
	estimate := uint64(1) << uint((128-leading_zeros+1)/2)
	for index := 0; index < 64; index++ {
		quotient, _ := bits.Divide_64(bits.Dividend_High_64(high),
			bits.Dividend_Low_64(low), bits.Divisor_64(estimate))
		next := (estimate + uint64(quotient)) / 2
		if next >= estimate {
			return Root_Integer(estimate)
		}
		estimate = next
	}
	return Root_Integer(estimate)
}

// Sine_Turns returns the sine of an angle measured in whole turns. The angle is reduced
// to one period and approximated by Bhaskara's rational formula for sin(pi*theta), so no
// irrational pi enters and the quarter-turn extremes land exactly on plus or minus one.
func Sine_Turns(turns Number) (sine Sine) {
	defer func() { Sine_Invariants(sine, "sine_turns.sine") }()
	Number_Invariants(turns, "sine_turns.turns")
	fraction := turns % Number(SCALE)
	if fraction < 0 {
		fraction += Number(SCALE)
	}
	negative := false
	if fraction >= Number(SCALE)/2 {
		negative = true
		fraction -= Number(SCALE) / 2
	}
	// Theta in [0,1] is twice the half-period fraction; the product theta*(1-theta)
	// drives Bhaskara's 16p / (5 - 4p) approximation of sin(pi*theta).
	theta := fraction * 2
	product := Multiply(Multiplicand(theta), Multiplier(Number(SCALE)-theta))
	numerator := 16 * product
	magnitude := Divide(
		Dividend(numerator), Divisor(Number(From_Integer(5))-4*product),
	)
	if negative {
		return Sine(-magnitude)
	}
	return Sine(magnitude)
}

// Format renders a value as decimal text with a set number of fractional digits, the
// dropped remainder rounded half away from zero.
func Format(value Number, digits Digit_Count) (text Text) {
	defer func() { Text_Invariants(text, "format.text") }()
	Number_Invariants(value, "format.value")
	Digit_Count_Invariants(digits, "format.digits")
	negative := value < 0
	magnitude := uint64(value)
	if value < 0 {
		magnitude = uint64(-int64(value))
	}
	power := int64(1)
	for index := 0; index < int(digits); index++ {
		power *= 10
	}
	product_high, product_low := bits.Multiply_64(
		bits.Word_64(magnitude), bits.Multiplier_64(power))
	rounded, carry := bits.Add_64(
		bits.Word_64(product_low), 1<<(FRACTIONAL_BITS-1), 0)
	high := uint64(product_high) + uint64(carry)
	low := uint64(rounded)
	scaled := (high << (64 - FRACTIONAL_BITS)) | (low >> FRACTIONAL_BITS)
	if scaled == 0 {
		negative = false
	}
	unsigned_power := uint64(power)
	text = Text(strconv.FormatUint(scaled/unsigned_power, 10))
	if digits > 0 {
		fraction := strconv.FormatUint(scaled%unsigned_power, 10)
		for len(fraction) < int(digits) {
			fraction = "0" + fraction
		}
		text += Text("." + fraction)
	}
	if negative {
		return Text("-" + text)
	}
	return text
}

// MarshalJSON renders a Number as a bare JSON decimal — an integer when whole, else the
// fraction to six places with trailing zeros trimmed. Rounding at six places hides the
// sub-microscale binary remainder, so ordinary values still read as clean decimals.
func (number Number) MarshalJSON() (data []byte, err error) {
	if Is_Integer(number) {
		return []byte(strconv.FormatInt(int64(Whole(number)), 10)), nil
	}
	text := Format(number, 6)
	end_count := len(text)
	for end_count > 0 {
		if text[end_count-1] != '0' {
			break
		}
		end_count--
	}
	if end_count > 0 {
		if text[end_count-1] == '.' {
			end_count--
		}
	}
	return []byte(text[:end_count]), nil
}

// UnmarshalJSON parses a JSON decimal number into a Number, rounding the fraction onto the
// fixed-point grid.
func (number *Number) UnmarshalJSON(data []byte) (err error) {
	text := string(data)
	negative := false
	if strings.HasPrefix(text, "-") {
		negative = true
		text = text[1:]
	}
	whole_text := text
	fraction_text := ""
	point_offset := strings.IndexByte(text, '.')
	if point_offset >= 0 {
		whole_text = text[:point_offset]
		fraction_text = text[point_offset+1:]
	}
	whole_part, err := strconv.ParseInt(whole_text, 10, 64)
	if err != nil {
		return err
	}
	digits := fraction_text
	if len(digits) > 9 {
		digits = digits[:9]
	}
	units := int64(0)
	if digits != "" {
		parsed, parse_error := strconv.ParseInt(digits, 10, 64)
		if parse_error == nil {
			power := int64(1)
			for index := 0; index < len(digits); index++ {
				power *= 10
			}
			units = (parsed<<FRACTIONAL_BITS + power/2) / power
		}
	}
	value := whole_part*SCALE + units
	if negative {
		value = -value
	}
	*number = Number(value)
	return nil
}
