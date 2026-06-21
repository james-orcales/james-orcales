// Package fixedpoint is a base-two fixed-point number backed by int64, the deterministic
// stand-in for float64 in packages the linter holds reproducible. Float arithmetic
// diverges across platforms; an integer scaled by a fixed power of two does not. The
// scale is a power of two so the rescale every multiply and divide needs is a bit shift,
// not the hardware divide a base-ten scale would force. The linter bans methods, so every
// operation is a free function named for what it does, with the value it acts on first.
package fixedpoint

import (
	"math/bits"
	"strconv"
	"strings"
)

// Fractional_bits is how many of a Number's low bits hold the fraction; the rest hold the
// integer part. Twenty bits gives ~9.5e-7 precision over a ±8.8e12 range, matching the
// magnitudes the deterministic callers work in.
const fractional_bits = 20

// Scale is the count of fixed-point units in one whole — two raised to fractional_bits.
// A Number's real value is its stored integer divided by Scale.
const Scale = 1 << fractional_bits

// Number is a base-two fixed-point value. Addition, subtraction, negation, and the
// ordering comparisons are the native int64 operators, the shared Scale aligning them;
// multiplication and division need the functions below because the scale must cancel.
type Number int64

// Ratio is a dimensionless fixed-point multiplier denoting ratio divided by Scale. It
// is a type distinct from Number so Apply takes one of each — sidestepping the
// input-struct rule two Numbers would trip — and so a ratio reads as a plain constant.
type Ratio int64

// From_Integer lifts a whole number into fixed-point.
func From_Integer(value int64) (number Number) {
	return Number(value * Scale)
}

// From_Ratio_Input holds the operands of From_Ratio, which repeat a type.
type From_Ratio_Input struct {
	// Numerator is the dividend of the ratio.
	Numerator int64
	// Denominator is the divisor of the ratio.
	Denominator int64
}

// From_Ratio lifts the quotient numerator/denominator into fixed-point, the scaled-up
// numerator taken through a 128-bit intermediate so it cannot overflow and a large
// integer mean keeps its fraction. A zero denominator yields zero, not a divide by zero.
func From_Ratio(input *From_Ratio_Input) (number Number) {
	if input.Denominator == 0 {
		return 0
	}
	negative := (input.Numerator < 0) != (input.Denominator < 0)
	magnitude := shift_divide(&shift_divide_input{
		Numerator:   absolute(Number(input.Numerator)),
		Denominator: absolute(Number(input.Denominator)),
	})
	return signed(magnitude, negative)
}

// Whole truncates a fixed-point value toward zero to the integer it contains.
func Whole(value Number) (whole int64) {
	return int64(value) / Scale
}

// Is_Integer reports whether a value carries no fractional part.
func Is_Integer(value Number) (yes bool) {
	return int64(value)%Scale == 0
}

// Multiply_Input pairs the two factors of Multiply, which repeat a type.
type Multiply_Input struct {
	// A is the first factor.
	A Number
	// B is the second factor.
	B Number
}

// Multiply returns the fixed-point product: the 128-bit product of the factors shifted
// back down by the scale, so no divide and no premature overflow.
func Multiply(input *Multiply_Input) (product Number) {
	negative := (input.A < 0) != (input.B < 0)
	magnitude := multiply_shifted(&multiply_shifted_input{
		Left: absolute(input.A), Right: absolute(input.B),
	})
	return signed(magnitude, negative)
}

// Divide_Input pairs the operands of Divide, which repeat a type.
type Divide_Input struct {
	// Dividend is the value being divided.
	Dividend Number
	// Divisor is the value it is divided by.
	Divisor Number
}

// Divide returns the fixed-point quotient: the dividend scaled up by a shift, then one
// hardware divide by the runtime divisor. A zero divisor yields zero.
func Divide(input *Divide_Input) (quotient Number) {
	if input.Divisor == 0 {
		return 0
	}
	negative := (input.Dividend < 0) != (input.Divisor < 0)
	magnitude := shift_divide(&shift_divide_input{
		Numerator:   absolute(input.Dividend),
		Denominator: absolute(input.Divisor),
	})
	return signed(magnitude, negative)
}

// Apply scales a value by a dimensionless ratio.
func Apply(value Number, ratio Ratio) (scaled Number) {
	negative := (value < 0) != (ratio < 0)
	magnitude := multiply_shifted(&multiply_shifted_input{
		Left: absolute(value), Right: absolute(Number(ratio)),
	})
	return signed(magnitude, negative)
}

// Square_Root returns the fixed-point square root of a fixed-point value. A negative
// input has no real root and yields zero.
func Square_Root(value Number) (root Number) {
	if value < 0 {
		return 0
	}
	// The root of value/Scale, scaled back up, is the root of value*Scale; the product
	// can exceed int64, so it is taken as a 128-bit radicand.
	high, low := bits.Mul64(uint64(value), Scale)
	return Number(square_root_uint128(&Integer_Root_Input{High: high, Low: low}))
}

// Square_Root_Scaled returns the fixed-point square root of a plain integer, for a sum
// of squares whose magnitude would overflow if first lifted into fixed-point. A
// negative input yields zero.
func Square_Root_Scaled(value int64) (root Number) {
	if value < 0 {
		return 0
	}
	// The root of value, scaled up by Scale, is the root of value*Scale*Scale.
	high, low := bits.Mul64(uint64(value), Scale*Scale)
	return Number(square_root_uint128(&Integer_Root_Input{High: high, Low: low}))
}

// Integer_Root_Input holds the high and low words of a 128-bit radicand.
type Integer_Root_Input struct {
	// High is the upper 64 bits of the radicand.
	High uint64
	// Low is the lower 64 bits of the radicand.
	Low uint64
}

// Integer_Root returns the floor of the square root of a 128-bit radicand — the integer
// primitive the fixed-point roots build on, and the escape hatch for a sum of squares too
// large to lift into fixed-point first.
func Integer_Root(input *Integer_Root_Input) (root int64) {
	return int64(square_root_uint128(input))
}

// Sine_Turns returns the sine of an angle measured in whole turns. The angle is reduced
// to one period and approximated by Bhaskara's rational formula for sin(pi*theta), so no
// irrational pi enters and the quarter-turn extremes land exactly on plus or minus one.
func Sine_Turns(turns Number) (sine Number) {
	fraction := turns % Number(Scale)
	if fraction < 0 {
		fraction += Number(Scale)
	}
	negative := false
	if fraction >= Number(Scale)/2 {
		negative = true
		fraction -= Number(Scale) / 2
	}
	// Theta in [0,1] is twice the half-period fraction; the product theta*(1-theta)
	// drives Bhaskara's 16p / (5 - 4p) approximation of sin(pi*theta).
	theta := fraction * 2
	product := Multiply(&Multiply_Input{A: theta, B: Number(Scale) - theta})
	numerator := 16 * product
	magnitude := Divide(&Divide_Input{
		Dividend: numerator, Divisor: From_Integer(5) - 4*product,
	})
	if negative {
		return -magnitude
	}
	return magnitude
}

// Format renders a value as decimal text with a set number of fractional digits, the
// dropped remainder rounded half away from zero.
func Format(value Number, digits int) (text string) {
	negative := value < 0
	scaled := decimal_scaled(absolute(value), digits)
	if scaled == 0 {
		negative = false
	}
	power := uint64(power_of_ten(digits))
	text = strconv.FormatUint(scaled/power, 10)
	if digits > 0 {
		fraction := strconv.FormatUint(scaled%power, 10)
		text += "." + left_pad_zeros(fraction, digits)
	}
	if negative {
		return "-" + text
	}
	return text
}

// MarshalJSON renders a Number as a bare JSON decimal — an integer when whole, else the
// fraction to six places with trailing zeros trimmed. Rounding at six places hides the
// sub-microscale binary remainder, so ordinary values still read as clean decimals.
func (number Number) MarshalJSON() (data []byte, err error) {
	if Is_Integer(number) {
		return []byte(strconv.FormatInt(Whole(number), 10)), nil
	}
	return []byte(trim_trailing_zeros(Format(number, 6))), nil
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
	value := whole_part*Scale + fraction_to_units(fraction_text)
	if negative {
		value = -value
	}
	*number = Number(value)
	return nil
}

// Holds the unsigned factors of multiply_shifted.
type multiply_shifted_input struct {
	// Left is the first factor.
	Left uint64
	// Right is the second factor.
	Right uint64
}

// Returns left*right rescaled by the scale: the high and low words of the 128-bit product
// shifted down by fractional_bits — a shift where a base-ten scale would need a divide.
// Callers pass factors whose true product fits int64, so the shifted-away high bits are
// zero.
func multiply_shifted(input *multiply_shifted_input) (magnitude uint64) {
	high, low := bits.Mul64(input.Left, input.Right)
	return (high << (64 - fractional_bits)) | (low >> fractional_bits)
}

// Holds the unsigned operands of shift_divide.
type shift_divide_input struct {
	// Numerator is the value being divided, before the scale-up shift.
	Numerator uint64
	// Denominator is the value it is divided by.
	Denominator uint64
}

// Returns (numerator << fractional_bits) / denominator through a 128-bit intermediate, so
// the scaled-up numerator keeps full precision without overflowing. The divide is the one
// the runtime divisor forces; only the scale-up is a shift. Callers pass a nonzero
// denominator and a result that fits int64, so the high word stays below the denominator.
func shift_divide(input *shift_divide_input) (quotient uint64) {
	high := input.Numerator >> (64 - fractional_bits)
	low := input.Numerator << fractional_bits
	quotient, _ = bits.Div64(high, low, input.Denominator)
	return quotient
}

// Returns the magnitude of a value as an unsigned integer, so the unsigned 128-bit
// primitives can run on a signed Number.
func absolute(value Number) (magnitude uint64) {
	if value < 0 {
		return uint64(-int64(value))
	}
	return uint64(value)
}

// Reapplies a sign that was stripped for the unsigned 128-bit math.
func signed(magnitude uint64, negative bool) (value Number) {
	if negative {
		return Number(-int64(magnitude))
	}
	return Number(magnitude)
}

// Returns the floor of the square root of a 128-bit radicand by Newton's method, seeded
// at two raised to half the radicand's bit length — an overestimate the iteration drives
// down to the floor in a handful of steps. The seed keeps the running estimate above the
// radicand's high word, so the 128-bit divide cannot overflow. A radicand that fits 64
// bits — every Square_Root and all but a large variance's Square_Root_Scaled — takes the
// faster path, whose per-step divide is a plain 64-bit one, not the wide 128-bit divide.
// Callers keep the radicand below 2^120, a bound any benchmark variance respects.
func square_root_uint128(input *Integer_Root_Input) (root uint64) {
	if input.High == 0 {
		return square_root_uint64(input.Low)
	}
	leading_zeros := bits.LeadingZeros64(input.High)
	estimate := uint64(1) << uint((128-leading_zeros+1)/2)
	for index := 0; index < 64; index++ {
		quotient, _ := bits.Div64(input.High, input.Low, estimate)
		next := (estimate + quotient) / 2
		if next >= estimate {
			return estimate
		}
		estimate = next
	}
	return estimate
}

// Returns the floor of the square root of a 64-bit radicand by Newton's method from a
// power-of-two seed. A reciprocal-square-root variant (table seed plus multiply-only
// refinement) was tried to avoid the per-step divide; it measured no faster, because the
// refinement is a serial dependency chain whatever each step costs, so the plain divide
// stays — simpler and lint-clean.
func square_root_uint64(value uint64) (root uint64) {
	if value == 0 {
		return 0
	}
	estimate := uint64(1) << uint((bits.Len64(value)+1)/2)
	for index := 0; index < 64; index++ {
		next := (estimate + value/estimate) / 2
		if next >= estimate {
			return estimate
		}
		estimate = next
	}
	return estimate
}

// Returns round(magnitude * 10^digits / Scale): the magnitude expressed as an integer
// with digits decimal places, the divide by the power-of-two Scale done as a rounding add
// and a shift.
func decimal_scaled(magnitude uint64, digits int) (scaled uint64) {
	high, low := bits.Mul64(magnitude, uint64(power_of_ten(digits)))
	low, carry := bits.Add64(low, 1<<(fractional_bits-1), 0)
	high += carry
	return (high << (64 - fractional_bits)) | (low >> fractional_bits)
}

// Returns ten raised to a small non-negative exponent.
func power_of_ten(exponent int) (result int64) {
	result = 1
	for index := 0; index < exponent; index++ {
		result *= 10
	}
	return result
}

// Widens text to width by prepending zeros, so a fraction keeps its leading zeros once
// the integer part has been stripped.
func left_pad_zeros(text string, width int) (padded string) {
	for len(text) < width {
		text = "0" + text
	}
	return text
}

// Drops trailing zeros, and a now-trailing point, from decimal text, leaving the minimal
// exact decimal.
func trim_trailing_zeros(text string) (trimmed string) {
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
	return text[:end_count]
}

// Reads decimal fraction digits as a count of fixed-point units, rounding fraction*Scale.
// Digits past the ninth are dropped, far beyond the grid's resolution.
func fraction_to_units(fraction_text string) (units int64) {
	digits := fraction_text
	if len(digits) > 9 {
		digits = digits[:9]
	}
	if digits == "" {
		return 0
	}
	parsed, err := strconv.ParseInt(digits, 10, 64)
	if err != nil {
		return 0
	}
	power := power_of_ten(len(digits))
	return (parsed<<fractional_bits + power/2) / power
}
