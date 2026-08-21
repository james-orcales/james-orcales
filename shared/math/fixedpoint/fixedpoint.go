// Package fixedpoint is a base-two fixed-point number backed by int64, the deterministic
// stand-in for float64 in packages the linter holds reproducible. Float arithmetic
// diverges across platforms; an integer scaled by a fixed power of two does not. The
// scale is a power of two so the rescale every multiply and divide needs is a bit shift,
// not the hardware divide a base-ten scale would force. The linter bans methods, so every
// operation is a free function named for what it does, with the value it acts on first.
package fixedpoint

import (
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/simulation/aver/default"
)

// FRACTION_DIGITS_MAXIMUM is how many fraction digits a parse reads. Ten digits scaled by
// the fraction width would leave the signed range, thus the parse stops at nine.
const FRACTION_DIGITS_MAXIMUM = 9

// DECIMAL_BASE is the base this package reads and writes.
const DECIMAL_BASE = 10

// DECIMAL_TEXT_SIZE_MAXIMUM is how many digits the largest unsigned value needs.
const DECIMAL_TEXT_SIZE_MAXIMUM = 20

// DECIMAL_TEXT_SIZE_MINIMUM is the size of the empty text, which a parse rejects rather
// than refuses to accept.
const DECIMAL_TEXT_SIZE_MINIMUM = 0

// DECIMAL_DIGITS_SIZE_MINIMUM is the size of the shortest rendered value, a single zero.
const DECIMAL_DIGITS_SIZE_MINIMUM = 1

// DECIMAL_VALUE_MINIMUM is the smallest value a decimal parse gives.
const DECIMAL_VALUE_MINIMUM int64 = 0

// Unsigned_Value is a value this package renders as decimal text.
type Unsigned_Value uint64

// Unsigned_Value_Invariants states the complete unsigned 64-bit domain.
func Unsigned_Value_Invariants(value Unsigned_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Decimal_Text is the digits a parse reads. The empty text carries no digit, thus the
// domain starts at no bytes rather than at one.
type Decimal_Text string

// Decimal_Text_Invariants bounds the digits a parse accepts. Text longer than the largest
// unsigned value needs cannot name an integer this storage holds, thus the caller answers
// it before the parse runs.
func Decimal_Text_Invariants(value Decimal_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), DECIMAL_TEXT_SIZE_MINIMUM, DECIMAL_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Decimal_Value is the value a decimal parse gives. The parse reads digits only and the
// caller strips any sign, thus the value is never negative.
type Decimal_Value int64

// Decimal_Value_Invariants bounds a parsed value to the nonnegative integers.
func Decimal_Value_Invariants(value Decimal_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), DECIMAL_VALUE_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Decimal_Digit_Count keeps internal rendering within widest unsigned value.
type Decimal_Digit_Count int

// Decimal_Digit_Count_Invariants makes each digit walk bounded by unsigned storage width.
func Decimal_Digit_Count_Invariants(value Decimal_Digit_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), DECIMAL_DIGITS_SIZE_MINIMUM, DECIMAL_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Decimal_Digits keeps internal output separate from broader public caller storage.
type Decimal_Digits []byte

// Decimal_Digits_Invariants prevents internal writer from exceeding unsigned decimal width.
func Decimal_Digits_Invariants(value Decimal_Digits, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), DECIMAL_DIGITS_SIZE_MINIMUM, DECIMAL_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Counts decimal digits before caller output is touched.
func decimal_digit_count(value Unsigned_Value) (count Decimal_Digit_Count) {
	defer func() {
		Decimal_Digit_Count_Invariants(count, "decimal_digit_count.count")
	}()
	Unsigned_Value_Invariants(value, "decimal_digit_count.value")
	if value == 0 {
		return DECIMAL_DIGITS_SIZE_MINIMUM
	}
	residue := uint64(value)
	for residue > 0 {
		count++
		residue /= DECIMAL_BASE
	}
	return count
}

// Writes one unsigned value after caller proved exact room exists.
func decimal_into(destination Decimal_Digits, value Unsigned_Value) {
	Decimal_Digits_Invariants(destination, "decimal_into.destination")
	Unsigned_Value_Invariants(value, "decimal_into.value")
	if value == 0 {
		destination[0] = '0'
		return
	}
	residue := uint64(value)
	for index := len(destination) - 1; index >= 0; index-- {
		destination[index] = byte('0' + residue%DECIMAL_BASE)
		residue /= DECIMAL_BASE
	}
}

// Reads decimal text into a signed value, reporting whether every byte was a digit. A
// leading sign is the caller's to strip, thus this reads digits only.
func decimal_value(text Decimal_Text) (value Decimal_Value, ok Boolean) {
	defer func() {
		Decimal_Value_Invariants(value, "decimal_value.value")
		Boolean_Invariants(ok, "decimal_value.ok")
	}()
	Decimal_Text_Invariants(text, "decimal_value.text")
	if text == "" {
		return 0, false
	}
	for index := 0; index < len(text); index++ {
		symbol := text[index]
		if symbol < '0' {
			return 0, false
		}
		if symbol > '9' {
			return 0, false
		}
		digit := Decimal_Value(symbol - '0')
		// The next step overflows when the value already passes the largest signed value
		// less that digit, divided by the base. Testing before the step keeps the value
		// inside the range at every point, so no wrapped value ever escapes.
		if int64(value) > (bits.INTEGER_64_MAXIMUM-int64(digit))/DECIMAL_BASE {
			return 0, false
		}
		value = value*DECIMAL_BASE + digit
	}
	return value, true
}

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

// SIGN_BYTE_COUNT_MAXIMUM is room for one negative sign.
const SIGN_BYTE_COUNT_MAXIMUM = 1

// DECIMAL_POINT_BYTE_COUNT is room for one decimal separator.
const DECIMAL_POINT_BYTE_COUNT = 1

// WHOLE_BIT_COUNT_MAXIMUM removes fixed fraction from signed storage magnitude width.
const WHOLE_BIT_COUNT_MAXIMUM = bits.BIT_COUNT_64_MAXIMUM - FRACTIONAL_BITS

// WHOLE_DECIMAL_DIGIT_COUNT_MAXIMUM converts highest whole bit position to decimal width.
const WHOLE_DECIMAL_DIGIT_COUNT_MAXIMUM = (WHOLE_BIT_COUNT_MAXIMUM-1)*
	bits.DECIMAL_DIGIT_BINARY_LOGARITHM_CEILING/
	bits.DECIMAL_DIGIT_BINARY_LOGARITHM_SCALE + 1

// TEXT_SIZE_MINIMUM admits no output when caller storage is short.
const TEXT_SIZE_MINIMUM = 0

// TEXT_SIZE_MAXIMUM holds each decimal integer digit, sign, point, and requested fraction.
const TEXT_SIZE_MAXIMUM = SIGN_BYTE_COUNT_MAXIMUM + WHOLE_DECIMAL_DIGIT_COUNT_MAXIMUM +
	DECIMAL_POINT_BYTE_COUNT + DIGIT_COUNT_MAXIMUM

// JSON_TEXT_SIZE_MINIMUM is shortest bare JSON decimal.
const JSON_TEXT_SIZE_MINIMUM = DECIMAL_DIGITS_SIZE_MINIMUM

// JSON_TEXT_SIZE_MAXIMUM holds sign, integer digits, point, and parsed fraction digits.
const JSON_TEXT_SIZE_MAXIMUM = SIGN_BYTE_COUNT_MAXIMUM + DECIMAL_TEXT_SIZE_MAXIMUM +
	DECIMAL_POINT_BYTE_COUNT + FRACTION_DIGITS_MAXIMUM

// JSON_TEXT_UNVALIDATED_SIZE_MINIMUM admits empty hostile input.
const JSON_TEXT_UNVALIDATED_SIZE_MINIMUM = 0

// JSON_TEXT_UNVALIDATED_SIZE_MAXIMUM admits one oversized hostile input for rejection.
const JSON_TEXT_UNVALIDATED_SIZE_MAXIMUM = JSON_TEXT_SIZE_MAXIMUM + 1

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
func Number_Invariants(value Number, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Ratio is a dimensionless fixed-point multiplier denoting ratio divided by SCALE. It
// is a type distinct from Number so Apply takes one of each — sidestepping the
// input-struct rule two Numbers would trip — and so a ratio reads as a plain constant.
type Ratio int64

// Ratio_Invariants states the complete ratio storage domain.
func Ratio_Invariants(value Ratio, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Numerator is the dividend of a ratio.
type Numerator int64

// Numerator_Invariants states the complete signed 64-bit domain.
func Numerator_Invariants(value Numerator, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Denominator is the divisor of a ratio.
type Denominator int64

// Denominator_Invariants states the complete signed 64-bit domain.
func Denominator_Invariants(value Denominator, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Multiplicand is the first fixed-point factor.
type Multiplicand Number

// Multiplicand_Invariants states the complete fixed-point storage domain.
func Multiplicand_Invariants(value Multiplicand, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Multiplier is the second fixed-point factor.
type Multiplier Number

// Multiplier_Invariants states the complete fixed-point storage domain.
func Multiplier_Invariants(value Multiplier, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Dividend is the fixed-point value that a divisor divides.
type Dividend Number

// Dividend_Invariants states the complete fixed-point storage domain.
func Dividend_Invariants(value Dividend, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Divisor is the fixed-point value that divides a dividend.
type Divisor Number

// Divisor_Invariants states the complete fixed-point storage domain.
func Divisor_Invariants(value Divisor, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Whole_Integer is an integer that fixed-point storage can lift without overflow.
type Whole_Integer int64

// Whole_Integer_Invariants bounds an integer to the fixed-point whole-number domain.
func Whole_Integer_Invariants(value Whole_Integer, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), WHOLE_INTEGER_MINIMUM, WHOLE_INTEGER_MAXIMUM).
		Ensure()
}

// Integer_Number is a fixed-point Number with no fractional units.
type Integer_Number Number

// Integer_Number_Invariants bounds the fixed-point whole-number domain.
func Integer_Number_Invariants(value Integer_Number, namespace aver.Namespace) {
	aver.Always(
		int64(value)%SCALE == 0,
		"An Integer_Number has no fractional units.",
	)
	aver.Tree(value, namespace).
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
func Radicand_Invariants(value Radicand, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Number_Root is a nonnegative square root of a Number.
type Number_Root Number

// Number_Root_Invariants bounds a root to the Number radicand domain.
func Number_Root_Invariants(value Number_Root, namespace aver.Namespace) {
	aver.Tree(value, namespace).
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
func Scaled_Root_Invariants(value Scaled_Root, namespace aver.Namespace) {
	aver.Tree(value, namespace).
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
func Root_Integer_Invariants(value Root_Integer, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), ROOT_INTEGER_MINIMUM, ROOT_INTEGER_MAXIMUM).
		Ensure()
}

// Sine is a fixed-point sine in the closed interval from negative one to one.
type Sine Number

// Sine_Invariants bounds a sine to the unit interval.
func Sine_Invariants(value Sine, namespace aver.Namespace) {
	aver.Tree(value, namespace).
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
func High_Word_Invariants(value High_Word, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, HIGH_WORD_MAXIMUM).
		Ensure()
}

// Low_Word is the lower word of a 128-bit integer.
type Low_Word uint64

// Low_Word_Invariants states the complete unsigned 64-bit domain.
func Low_Word_Invariants(value Low_Word, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Digit_Count is a count of decimal fraction digits.
type Digit_Count int

// Digit_Count_Invariants keeps decimal scaling in the signed 64-bit domain.
func Digit_Count_Invariants(value Digit_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), DIGIT_COUNT_MINIMUM, DIGIT_COUNT_MAXIMUM).
		Ensure()
}

// Boolean is one true or false value.
type Boolean bool

// Boolean_Invariants records both Boolean states.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A Boolean value is true.").
		Ensure()
}

// Text_Unvalidated admits empty and one-byte-oversized hostile input for graceful rejection.
type Text_Unvalidated string

// Text_Unvalidated_Invariants caps work before syntax scanning.
func Text_Unvalidated_Invariants(value Text_Unvalidated, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), JSON_TEXT_UNVALIDATED_SIZE_MINIMUM,
			JSON_TEXT_UNVALIDATED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Text is bounded bare JSON text.
type Text string

// Text_Invariants makes syntax work proportional only to package bound.
func Text_Invariants(value Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), JSON_TEXT_SIZE_MINIMUM, JSON_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Digits lets caller placement own variable-length decimal output.
type Digits []byte

// Digits_Invariants prevents caller storage from expanding package work bound.
func Digits_Invariants(value Digits, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Text_Count keeps output result scalar instead of returning owned text.
type Text_Count int

// Text_Count_Invariants binds output count to caller storage bound.
func Text_Count_Invariants(value Text_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Text_Validate centralizes hostile size rejection before syntax scanning.
func Text_Validate(value Text_Unvalidated) (text Text, ok Boolean) {
	defer func() {
		Text_Invariants(text, "text_validate.text")
		Boolean_Invariants(ok, "text_validate.ok")
	}()
	Text_Unvalidated_Invariants(value, "text_validate.value")
	if len(value) < JSON_TEXT_SIZE_MINIMUM {
		return "0", false
	}
	if len(value) > JSON_TEXT_SIZE_MAXIMUM {
		return "0", false
	}
	return Text(value), true
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

// Into_Text checks exact room before write so short caller storage stays unchanged.
func Into_Text(destination Digits, value Number, digits Digit_Count) (count Text_Count) {
	defer func() { Text_Count_Invariants(count, "into_text.count") }()
	Digits_Invariants(destination, "into_text.destination")
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
	whole := Unsigned_Value(scaled / unsigned_power)
	whole_count := decimal_digit_count(whole)
	required := int(whole_count)
	if digits > 0 {
		required += DECIMAL_POINT_BYTE_COUNT + int(digits)
	}
	if negative {
		required += SIGN_BYTE_COUNT_MAXIMUM
	}
	if len(destination) < required {
		return TEXT_SIZE_MINIMUM
	}
	index := 0
	if negative {
		destination[index] = '-'
		index += SIGN_BYTE_COUNT_MAXIMUM
	}
	decimal_into(Decimal_Digits(destination[index:index+int(whole_count)]), whole)
	index += int(whole_count)
	if digits > 0 {
		destination[index] = '.'
		index += DECIMAL_POINT_BYTE_COUNT
		fraction := scaled % unsigned_power
		fraction_destination := destination[index : index+int(digits)]
		fraction_index := len(fraction_destination) - 1
		for ; fraction_index >= 0; fraction_index-- {
			fraction_destination[fraction_index] = byte('0' + fraction%DECIMAL_BASE)
			fraction /= DECIMAL_BASE
		}
	}
	return Text_Count(required)
}

// Into_JSON stages bounded text locally so trimming never exposes partial caller output.
func Into_JSON(destination Digits, value Number) (count Text_Count) {
	defer func() { Text_Count_Invariants(count, "into_json.count") }()
	Digits_Invariants(destination, "into_json.destination")
	Number_Invariants(value, "into_json.value")
	var storage [TEXT_SIZE_MAXIMUM]byte
	digits := Digit_Count(DIGIT_COUNT_MAXIMUM)
	if bool(Is_Integer(value)) {
		digits = DIGIT_COUNT_MINIMUM
	}
	written := Into_Text(storage[:], value, digits)
	end_count := int(written)
	if digits > DIGIT_COUNT_MINIMUM {
		for end_count > TEXT_SIZE_MINIMUM {
			if storage[end_count-1] != '0' {
				break
			}
			end_count--
		}
		if end_count > TEXT_SIZE_MINIMUM {
			if storage[end_count-1] == '.' {
				end_count--
			}
		}
	}
	if len(destination) < end_count {
		return TEXT_SIZE_MINIMUM
	}
	copy(destination, storage[:end_count])
	return Text_Count(end_count)
}

// From_JSON returns scalar validity so malformed text owns no diagnostic allocation.
func From_JSON(value Text_Unvalidated) (number Number, ok Boolean) {
	defer func() {
		Number_Invariants(number, "from_json.number")
		Boolean_Invariants(ok, "from_json.ok")
	}()
	Text_Unvalidated_Invariants(value, "from_json.value")
	validated, bounded := Text_Validate(value)
	if !bool(bounded) {
		return 0, false
	}
	return from_json_text(validated)
}

// Keeps size validation outside syntax cases so hostile rejection stays one bounded step.
func from_json_text(value Text) (number Number, ok Boolean) {
	defer func() {
		Number_Invariants(number, "from_json_text.number")
		Boolean_Invariants(ok, "from_json_text.ok")
	}()
	Text_Invariants(value, "from_json_text.value")
	text := string(value)
	negative := false
	if text[0] == '-' {
		negative = true
		text = text[SIGN_BYTE_COUNT_MAXIMUM:]
		if len(text) == 0 {
			return 0, false
		}
	}
	whole_text := text
	fraction_text := ""
	for index := 0; index < len(text); index++ {
		if text[index] == '.' {
			whole_text = text[:index]
			fraction_text = text[index+DECIMAL_POINT_BYTE_COUNT:]
			break
		}
	}
	if len(whole_text) > DECIMAL_TEXT_SIZE_MAXIMUM {
		return 0, false
	}
	whole_part, whole_ok := decimal_value(Decimal_Text(whole_text))
	if !bool(whole_ok) {
		return 0, false
	}
	point_present := len(whole_text) != len(text)
	if point_present {
		if fraction_text == "" {
			return 0, false
		}
		if len(fraction_text) > FRACTION_DIGITS_MAXIMUM {
			return 0, false
		}
	}
	fraction_units := uint64(0)
	if fraction_text != "" {
		parsed, parsed_ok := decimal_value(Decimal_Text(fraction_text))
		if !bool(parsed_ok) {
			return 0, false
		}
		power := uint64(1)
		for index := 0; index < len(fraction_text); index++ {
			power *= DECIMAL_BASE
		}
		fraction_units = (uint64(parsed)*SCALE + power/2) / power
	}
	magnitude_maximum := uint64(bits.INTEGER_64_MAXIMUM)
	if negative {
		magnitude_maximum++
	}
	if uint64(whole_part) > magnitude_maximum/SCALE {
		return 0, false
	}
	whole_units := uint64(whole_part) * SCALE
	if fraction_units > magnitude_maximum-whole_units {
		return 0, false
	}
	units := whole_units + fraction_units
	if negative {
		return Number(-int64(units)), true
	}
	return Number(units), true
}
