// Package scalar gives the operations of the Go standard library math package over a
// fixedpoint Number rather than a float64. The deterministic tier bans the float because
// IEEE-754 results differ across platforms, so a scaled integer carries the real values
// here and one seed gives one result everywhere. The name says what the package holds:
// arithmetic on one number at a time, beside the bit operations of a sibling package. The
// linter bans methods, so every operation is a free function named for what it does.
package scalar

import (
	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/math/fixedpoint"
)

// SCALE is the count of fixed-point units in one whole, repeated from fixedpoint so the
// bounds below stay literal constant expressions the assertion analyzer can resolve.
const SCALE = fixedpoint.SCALE

// FRACTION_UNIT_NEGATIVE_ONE is negative one fixed-point fraction unit. A result that the
// grid quantizes cannot land on it, thus it is excluded from those result domains.
const FRACTION_UNIT_NEGATIVE_ONE int64 = -1

// FRACTION_UNIT_ONE is one fixed-point fraction unit.
const FRACTION_UNIT_ONE int64 = 1

// FRACTION_UNIT_TWO is two fixed-point fraction units.
const FRACTION_UNIT_TWO int64 = 2

// MAGNITUDE_MINIMUM is the smallest magnitude of a signed integer.
const MAGNITUDE_MINIMUM int64 = 0

// POSITIVE_MINIMUM is the smallest Number above zero, one fixed-point unit. A logarithm
// argument, a power base, and a cube-root magnitude all start here.
const POSITIVE_MINIMUM int64 = 1

// NEGATABLE_MINIMUM is the most negative value this package can negate. Negating the
// storage floor overflows back to the floor, so that one value has no magnitude, and every
// operation that takes a magnitude first starts one above it.
const NEGATABLE_MINIMUM int64 = bits.INTEGER_64_MINIMUM + 1

// HALF is one half on the grid, the point at which Round moves away from zero.
const HALF int64 = SCALE / 2

// QUARTER_TURN is a quarter turn measured in turns. A cosine is a sine this far ahead.
const QUARTER_TURN = SCALE / 4

// UNIT_MINIMUM is negative one, the lower end of a sine, a cosine, and an arcsine
// argument.
const UNIT_MINIMUM int64 = -SCALE

// UNIT_MAXIMUM is one, the upper end of a sine, a cosine, and an arcsine argument.
const UNIT_MAXIMUM int64 = SCALE

// PI is the ratio of a circumference to a diameter, on the fixed-point grid.
const PI fixedpoint.Number = 3294199

// PI_HALF is a quarter turn in radians.
const PI_HALF fixedpoint.Number = 1647099

// PI_QUARTER is an eighth turn in radians.
const PI_QUARTER fixedpoint.Number = 823550

// E is the base of the natural logarithm, on the fixed-point grid.
const E fixedpoint.Number = 2850325

// TURNS_PER_RADIAN converts an angle in radians to the turns the fixedpoint sine accepts.
const TURNS_PER_RADIAN fixedpoint.Ratio = 166886

// NATURAL_LOGARITHM_OF_TWO scales a base-two logarithm into a natural logarithm.
const NATURAL_LOGARITHM_OF_TWO fixedpoint.Ratio = 726817

// BINARY_LOGARITHM_OF_E scales a natural exponent into a base-two exponent.
const BINARY_LOGARITHM_OF_E fixedpoint.Ratio = 1512775

// BINARY_LOGARITHM_OF_TEN divides a base-two logarithm into a base-ten logarithm.
const BINARY_LOGARITHM_OF_TEN fixedpoint.Number = 3483294

// ARCTANGENT_COEFFICIENT_A is the first coefficient of the arctangent approximation.
const ARCTANGENT_COEFFICIENT_A fixedpoint.Number = 256587

// ARCTANGENT_COEFFICIENT_B is the second coefficient of the arctangent approximation.
const ARCTANGENT_COEFFICIENT_B fixedpoint.Number = 69521

// FRACTION_BIT_COUNT is how many fraction bits a logarithm resolves, one for each squaring
// of the mantissa. It is the fraction width of the grid, so a logarithm resolves the whole
// fraction and no further.
const FRACTION_BIT_COUNT = fixedpoint.FRACTIONAL_BITS

// ROOT_TABLE_SIZE is how many entries the table of the roots of two holds, one for each
// fraction bit and one unused entry at the head so the index matches the bit.
const ROOT_TABLE_SIZE = FRACTION_BIT_COUNT + 1

// LOGARITHM_2_MINIMUM is the base-two logarithm of the smallest positive Number, which is
// one fixed-point unit and so two raised to the negative fraction width.
const LOGARITHM_2_MINIMUM int64 = -20971520

// LOGARITHM_2_MAXIMUM is the base-two logarithm of the largest Number.
const LOGARITHM_2_MAXIMUM int64 = 45088767

// LOGARITHM_MINIMUM is the natural logarithm of the smallest positive Number.
const LOGARITHM_MINIMUM int64 = -14536340

// LOGARITHM_MAXIMUM is the natural logarithm of the largest Number.
const LOGARITHM_MAXIMUM int64 = 31253130

// LOGARITHM_10_MINIMUM is the base-ten logarithm of the smallest positive Number.
const LOGARITHM_10_MINIMUM int64 = -6313056

// LOGARITHM_10_MAXIMUM is the base-ten logarithm of the largest Number.
const LOGARITHM_10_MAXIMUM int64 = 13573071

// EXPONENT_2_MINIMUM is the smallest base-two exponent. Every exponent below it gives
// zero, thus the domain stops where the result stops changing.
const EXPONENT_2_MINIMUM int64 = -64 * SCALE

// EXPONENT_2_MAXIMUM is the largest base-two exponent the Number range holds.
const EXPONENT_2_MAXIMUM int64 = 42 * SCALE

// EXPONENT_MINIMUM is the smallest natural exponent.
const EXPONENT_MINIMUM int64 = -44 * SCALE

// EXPONENT_MAXIMUM is the largest natural exponent the Number range holds.
const EXPONENT_MAXIMUM int64 = 29 * SCALE

// POWER_MINIMUM is the smallest result of an exponential, which underflow gives.
const POWER_MINIMUM int64 = 0

// POWER_MAXIMUM is the largest result of an exponential, two raised to EXPONENT_2_MAXIMUM.
const POWER_MAXIMUM int64 = 4611686018427387904

// RADICAND_MINIMUM is the smallest radicand of a square root. A negative value has no real
// root, thus the domain starts at zero.
const RADICAND_MINIMUM int64 = 0

// CUBE_ROOT_MINIMUM is the cube root of the smallest Number.
const CUBE_ROOT_MINIMUM int64 = -CUBE_ROOT_MAXIMUM

// CUBE_ROOT_MAXIMUM is the cube root of the largest Number.
const CUBE_ROOT_MAXIMUM int64 = 21645278819

// ANGLE_MINIMUM is the smallest angle an inverse function returns, negative PI.
const ANGLE_MINIMUM int64 = -ANGLE_MAXIMUM

// ANGLE_MAXIMUM is the largest angle an inverse function returns, PI.
const ANGLE_MAXIMUM int64 = int64(PI)

// HYPERBOLIC_ANGLE_MINIMUM is the smallest hyperbolic angle the Number range holds.
const HYPERBOLIC_ANGLE_MINIMUM int64 = -HYPERBOLIC_ANGLE_MAXIMUM

// HYPERBOLIC_ANGLE_MAXIMUM is the largest hyperbolic angle the Number range holds. Each
// hyperbolic function is built from the natural exponential, thus the exponent bound is
// the angle bound.
const HYPERBOLIC_ANGLE_MAXIMUM int64 = EXPONENT_MAXIMUM

// HYPERBOLIC_MINIMUM is the smallest hyperbolic sine, which the most negative angle gives.
const HYPERBOLIC_MINIMUM int64 = -HYPERBOLIC_MAXIMUM

// HYPERBOLIC_MAXIMUM is the largest hyperbolic sine or cosine.
const HYPERBOLIC_MAXIMUM int64 = 2061129104266100736

// HYPERBOLIC_COSINE_MINIMUM is the smallest hyperbolic cosine. A hyperbolic cosine is
// never below one, thus its domain starts at one and not at the negative end.
const HYPERBOLIC_COSINE_MINIMUM int64 = 1048572

// Negatable_Integer is an integer this package can negate, and so one that has a
// magnitude.
type Negatable_Integer int64

// Negatable_Integer_Invariants excludes the storage floor, which negates to itself.
func Negatable_Integer_Invariants(value Negatable_Integer, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), NEGATABLE_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Magnitude_Integer is the magnitude of a signed integer.
type Magnitude_Integer int64

// Magnitude_Integer_Invariants bounds a magnitude to the nonnegative integers.
func Magnitude_Integer_Invariants(value Magnitude_Integer, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), MAGNITUDE_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// First_Integer is the first operand of an ordering comparison.
type First_Integer int64

// First_Integer_Invariants states the complete signed 64-bit domain.
func First_Integer_Invariants(value First_Integer, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Second_Integer is the second operand of an ordering comparison.
type Second_Integer int64

// Second_Integer_Invariants states the complete signed 64-bit domain.
func Second_Integer_Invariants(value Second_Integer, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Ordered_Integer is the integer an ordering comparison selects.
type Ordered_Integer int64

// Ordered_Integer_Invariants states the complete signed 64-bit domain.
func Ordered_Integer_Invariants(value Ordered_Integer, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Value is a Number that rounds to a whole Number without overflow.
type Value fixedpoint.Number

// Value_Invariants bounds a value to the range whose rounding stays in storage.
func Value_Invariants(value Value, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), fixedpoint.INTEGER_NUMBER_MINIMUM,
			fixedpoint.INTEGER_NUMBER_MAXIMUM).
		Ensure()
}

// Whole_Number is a Number with no fractional units.
type Whole_Number fixedpoint.Number

// Whole_Number_Invariants states that a whole Number carries no fraction.
func Whole_Number_Invariants(value Whole_Number, namespace invariant.Namespace) {
	invariant.Always(
		int64(value)%SCALE == 0,
		"A Whole_Number has no fractional units.",
	)
	invariant.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value), fixedpoint.INTEGER_NUMBER_MINIMUM,
			fixedpoint.INTEGER_NUMBER_MAXIMUM,
			FRACTION_UNIT_NEGATIVE_ONE, FRACTION_UNIT_ONE,
			FRACTION_UNIT_TWO, FRACTION_UNIT_TWO,
		).
		Ensure()
}

// Dividend is the Number a divisor divides.
type Dividend fixedpoint.Number

// Dividend_Invariants states the complete fixed-point storage domain.
func Dividend_Invariants(value Dividend, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Divisor is the Number that divides a dividend.
type Divisor fixedpoint.Number

// Divisor_Invariants states the complete fixed-point storage domain. A zero divisor gives
// a zero remainder rather than a panic, which matches fixedpoint division.
func Divisor_Invariants(value Divisor, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Remainder is the remainder of a truncated division.
type Remainder fixedpoint.Number

// Remainder_Invariants bounds a remainder. A remainder stays below its divisor in
// magnitude, and no divisor exceeds the most negative Number, thus the remainder stops one
// above it.
func Remainder_Invariants(value Remainder, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), NEGATABLE_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Radicand is a Number a square root accepts.
type Radicand fixedpoint.Number

// Radicand_Invariants excludes the negative values, which have no real root.
func Radicand_Invariants(value Radicand, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), RADICAND_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Root is a square root of a Number.
type Root fixedpoint.Number

// Root_Invariants bounds a root to the square roots the Number range holds. The root of
// the smallest positive radicand is already far above one unit, thus the grid leaves the
// first units empty.
func Root_Invariants(value Root, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value), fixedpoint.NUMBER_ROOT_MINIMUM,
			fixedpoint.NUMBER_ROOT_MAXIMUM,
			FRACTION_UNIT_ONE, FRACTION_UNIT_TWO,
			FRACTION_UNIT_TWO, FRACTION_UNIT_TWO,
		).
		Ensure()
}

// Cube_Radicand is a Number a cube root accepts, of either sign.
type Cube_Radicand fixedpoint.Number

// Cube_Radicand_Invariants states the complete fixed-point storage domain.
func Cube_Radicand_Invariants(value Cube_Radicand, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Cube_Root_Value is a cube root of a Number.
type Cube_Root_Value fixedpoint.Number

// Cube_Root_Value_Invariants bounds a cube root to the Number range. The root of the
// smallest positive radicand is already far above one unit, thus the grid leaves the units
// nearest zero empty.
func Cube_Root_Value_Invariants(value Cube_Root_Value, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value), CUBE_ROOT_MINIMUM, CUBE_ROOT_MAXIMUM,
			FRACTION_UNIT_NEGATIVE_ONE, FRACTION_UNIT_ONE,
			FRACTION_UNIT_TWO, FRACTION_UNIT_TWO,
		).
		Ensure()
}

// LEG_MAXIMUM is the largest leg whose hypotenuse the storage still holds. Two equal legs
// give a hypotenuse the square root of two times as long, thus the leg stops that far
// below the largest Number.
const LEG_MAXIMUM int64 = 6521908912666391106

// LEG_MINIMUM is the most negative leg whose hypotenuse the storage still holds.
const LEG_MINIMUM int64 = -LEG_MAXIMUM

// Leg_Opposite is the leg of a right triangle across from the angle, in a distance.
type Leg_Opposite fixedpoint.Number

// Leg_Opposite_Invariants bounds a leg so its hypotenuse stays in the storage range.
func Leg_Opposite_Invariants(value Leg_Opposite, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), LEG_MINIMUM, LEG_MAXIMUM).
		Ensure()
}

// Leg_Adjacent is the leg of a right triangle beside the angle, in a distance.
type Leg_Adjacent fixedpoint.Number

// Leg_Adjacent_Invariants bounds a leg so its hypotenuse stays in the storage range.
func Leg_Adjacent_Invariants(value Leg_Adjacent, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), LEG_MINIMUM, LEG_MAXIMUM).
		Ensure()
}

// CUBE_ESTIMATE_MINIMUM is the smallest starting estimate of a cube root. The smallest
// radicand is one unit, whose exponent is the negative fraction width, thus one third of
// that exponent still leaves thirteen bits.
const CUBE_ESTIMATE_MINIMUM int64 = 8192

// CUBE_ESTIMATE_MAXIMUM is the largest starting estimate of a cube root.
const CUBE_ESTIMATE_MAXIMUM int64 = 17179869184

// Cube_Magnitude is the magnitude of a radicand before a cube root.
type Cube_Magnitude fixedpoint.Number

// Cube_Magnitude_Invariants excludes zero, which the caller returns before it estimates.
func Cube_Magnitude_Invariants(value Cube_Magnitude, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), POSITIVE_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Cube_Estimate is a starting estimate of a cube root, always a power of two.
type Cube_Estimate fixedpoint.Number

// Cube_Estimate_Invariants bounds an estimate to the powers of two one third of the
// radicand exponent reaches.
func Cube_Estimate_Invariants(value Cube_Estimate, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), CUBE_ESTIMATE_MINIMUM, CUBE_ESTIMATE_MAXIMUM).
		Ensure()
}

// UNIT_ANGLE_MAXIMUM is the largest angle the unit arctangent returns, an eighth turn.
const UNIT_ANGLE_MAXIMUM int64 = int64(PI_QUARTER)

// Unit_Magnitude is the magnitude of a slope in the closed unit interval.
type Unit_Magnitude fixedpoint.Number

// Unit_Magnitude_Invariants bounds a magnitude to the closed unit interval.
func Unit_Magnitude_Invariants(value Unit_Magnitude, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), RADICAND_MINIMUM, UNIT_MAXIMUM).
		Ensure()
}

// Unit_Angle is the angle the unit arctangent returns.
type Unit_Angle fixedpoint.Number

// Unit_Angle_Invariants bounds an angle to the eighth turn the unit interval spans. The
// approximation steps from one unit to three, thus two is excluded.
func Unit_Angle_Invariants(value Unit_Angle, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value), RADICAND_MINIMUM, UNIT_ANGLE_MAXIMUM,
			FRACTION_UNIT_TWO, FRACTION_UNIT_TWO,
			FRACTION_UNIT_TWO, FRACTION_UNIT_TWO,
		).
		Ensure()
}

// Quotient_Angle is the principal angle of one leg divided by another. The quotient of two
// legs lands on a coarser set than a slope does, thus the units nearest zero stay empty.
type Quotient_Angle fixedpoint.Number

// Quotient_Angle_Invariants bounds an angle to a quarter turn on each side of zero.
func Quotient_Angle_Invariants(value Quotient_Angle, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value), PRINCIPAL_ANGLE_MINIMUM, PRINCIPAL_ANGLE_MAXIMUM,
			FRACTION_UNIT_NEGATIVE_ONE, FRACTION_UNIT_ONE,
			FRACTION_UNIT_TWO, FRACTION_UNIT_TWO,
		).
		Ensure()
}

// ADJACENT_ZERO is the leg a quotient rejects. A zero adjacent leg has no quotient, thus
// the caller answers it before the principal arctangent runs.
const ADJACENT_ZERO int64 = 0

// Nonzero_Adjacent is an adjacent leg that a quotient can divide by.
type Nonzero_Adjacent fixedpoint.Number

// Nonzero_Adjacent_Invariants excludes zero, which the caller answers on its own.
func Nonzero_Adjacent_Invariants(value Nonzero_Adjacent, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM,
			ADJACENT_ZERO, ADJACENT_ZERO, ADJACENT_ZERO, ADJACENT_ZERO,
		).
		Ensure()
}

// Opposite is the leg of a right triangle across from the angle.
type Opposite fixedpoint.Number

// Opposite_Invariants states the complete fixed-point storage domain.
func Opposite_Invariants(value Opposite, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Adjacent is the leg of a right triangle beside the angle.
type Adjacent fixedpoint.Number

// Adjacent_Invariants states the complete fixed-point storage domain.
func Adjacent_Invariants(value Adjacent, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// DISTANCE_MAXIMUM is the largest hypotenuse two admitted legs give.
const DISTANCE_MAXIMUM int64 = 9223369546587102923

// Distance is the hypotenuse of a right triangle.
type Distance fixedpoint.Number

// Distance_Invariants bounds a distance to the hypotenuses the admitted legs reach.
func Distance_Invariants(value Distance, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), RADICAND_MINIMUM, DISTANCE_MAXIMUM).
		Ensure()
}

// Exponent_2 is a base-two exponent.
type Exponent_2 fixedpoint.Number

// Exponent_2_Invariants bounds an exponent to the range the Number storage holds.
func Exponent_2_Invariants(value Exponent_2, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), EXPONENT_2_MINIMUM, EXPONENT_2_MAXIMUM).
		Ensure()
}

// Exponent is a natural exponent.
type Exponent fixedpoint.Number

// Exponent_Invariants bounds an exponent to the range the Number storage holds.
func Exponent_Invariants(value Exponent, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), EXPONENT_MINIMUM, EXPONENT_MAXIMUM).
		Ensure()
}

// Power_Value is the result of an exponential.
type Power_Value fixedpoint.Number

// Power_Value_Invariants bounds a power to the nonnegative Numbers an exponential reaches.
// A power leaves the units nearest zero empty, because the smallest power above zero is
// already the smallest whole shift.
func Power_Value_Invariants(value Power_Value, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value), POWER_MINIMUM, POWER_MAXIMUM,
			FRACTION_UNIT_ONE, FRACTION_UNIT_TWO,
			FRACTION_UNIT_TWO, FRACTION_UNIT_TWO,
		).
		Ensure()
}

// NATURAL_POWER_MAXIMUM is the largest result of the natural exponential. The natural
// exponent stops below the base-two exponent, thus this power stops below the other.
const NATURAL_POWER_MAXIMUM int64 = 4122258208532201472

// Natural_Power_Value is the result of the natural exponential.
type Natural_Power_Value fixedpoint.Number

// Natural_Power_Value_Invariants bounds the natural exponential to the powers it reaches.
func Natural_Power_Value_Invariants(
	value Natural_Power_Value, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value), POWER_MINIMUM, NATURAL_POWER_MAXIMUM,
			FRACTION_UNIT_ONE, FRACTION_UNIT_TWO,
			FRACTION_UNIT_TWO, FRACTION_UNIT_TWO,
		).
		Ensure()
}

// Argument is a positive Number a logarithm accepts.
type Argument fixedpoint.Number

// Argument_Invariants excludes zero and the negative values, which have no logarithm.
func Argument_Invariants(value Argument, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), POSITIVE_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Logarithm_2_Value is a base-two logarithm.
type Logarithm_2_Value fixedpoint.Number

// Logarithm_2_Value_Invariants bounds a base-two logarithm to the Number range.
func Logarithm_2_Value_Invariants(
	value Logarithm_2_Value, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value), LOGARITHM_2_MINIMUM, LOGARITHM_2_MAXIMUM,
			FRACTION_UNIT_NEGATIVE_ONE, FRACTION_UNIT_ONE,
			FRACTION_UNIT_TWO, FRACTION_UNIT_TWO,
		).
		Ensure()
}

// Logarithm_Value is a natural logarithm.
type Logarithm_Value fixedpoint.Number

// Logarithm_Value_Invariants bounds a natural logarithm to the Number range.
func Logarithm_Value_Invariants(value Logarithm_Value, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value), LOGARITHM_MINIMUM, LOGARITHM_MAXIMUM,
			FRACTION_UNIT_NEGATIVE_ONE, FRACTION_UNIT_ONE,
			FRACTION_UNIT_TWO, FRACTION_UNIT_TWO,
		).
		Ensure()
}

// Logarithm_10_Value is a base-ten logarithm.
type Logarithm_10_Value fixedpoint.Number

// Logarithm_10_Value_Invariants bounds a base-ten logarithm to the Number range.
func Logarithm_10_Value_Invariants(
	value Logarithm_10_Value, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value), LOGARITHM_10_MINIMUM, LOGARITHM_10_MAXIMUM,
			FRACTION_UNIT_NEGATIVE_ONE, FRACTION_UNIT_ONE,
			FRACTION_UNIT_TWO, FRACTION_UNIT_TWO,
		).
		Ensure()
}

// Base is the positive Number a power raises.
type Base fixedpoint.Number

// Base_Invariants excludes zero and the negative values, which have no logarithm.
func Base_Invariants(value Base, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), POSITIVE_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Power_Exponent is the exponent a power raises a base to.
type Power_Exponent fixedpoint.Number

// Power_Exponent_Invariants states the complete fixed-point storage domain. The product of
// the exponent and the logarithm carries its own bound, thus this one stays wide.
func Power_Exponent_Invariants(value Power_Exponent, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Angle is an angle in radians.
type Angle fixedpoint.Number

// Angle_Invariants bounds an angle to the range whose turn conversion stays in storage.
func Angle_Invariants(value Angle, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), fixedpoint.INTEGER_NUMBER_MINIMUM,
			fixedpoint.INTEGER_NUMBER_MAXIMUM).
		Ensure()
}

// Unit_Value is a Number in the closed interval from negative one to one.
type Unit_Value fixedpoint.Number

// Unit_Value_Invariants bounds a value to the closed unit interval.
func Unit_Value_Invariants(value Unit_Value, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value), UNIT_MINIMUM, UNIT_MAXIMUM,
			FRACTION_UNIT_NEGATIVE_ONE, FRACTION_UNIT_ONE,
			FRACTION_UNIT_TWO, FRACTION_UNIT_TWO,
		).
		Ensure()
}

// Unit_Argument is a Number in the closed unit interval an arcsine accepts.
type Unit_Argument fixedpoint.Number

// Unit_Argument_Invariants bounds an argument to the closed unit interval. Every value in
// that interval is a legal argument, thus this domain has no hole.
func Unit_Argument_Invariants(value Unit_Argument, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), UNIT_MINIMUM, UNIT_MAXIMUM).
		Ensure()
}

// Slope is the ratio an arctangent accepts.
type Slope fixedpoint.Number

// Slope_Invariants states the complete fixed-point storage domain.
func Slope_Invariants(value Slope, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Angle_Value is the angle an inverse function returns.
type Angle_Value fixedpoint.Number

// Angle_Value_Invariants bounds an angle to one half turn on each side of zero.
func Angle_Value_Invariants(value Angle_Value, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value), ANGLE_MINIMUM, ANGLE_MAXIMUM,
			FRACTION_UNIT_NEGATIVE_ONE, FRACTION_UNIT_ONE,
			FRACTION_UNIT_TWO, FRACTION_UNIT_TWO,
		).
		Ensure()
}

// PRINCIPAL_ANGLE_MINIMUM is the smallest angle an arcsine or an arctangent returns.
const PRINCIPAL_ANGLE_MINIMUM int64 = -PRINCIPAL_ANGLE_MAXIMUM

// PRINCIPAL_ANGLE_MAXIMUM is the largest angle an arcsine or an arctangent returns. Each
// stays within a quarter turn of zero, thus neither reaches the half turn a quotient does.
const PRINCIPAL_ANGLE_MAXIMUM int64 = int64(PI_HALF)

// Principal_Angle is the angle an arcsine or an arctangent returns.
type Principal_Angle fixedpoint.Number

// Principal_Angle_Invariants bounds an angle to a quarter turn on each side of zero. The
// approximation steps over two units near zero, thus that one value is excluded.
func Principal_Angle_Invariants(value Principal_Angle, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value), PRINCIPAL_ANGLE_MINIMUM, PRINCIPAL_ANGLE_MAXIMUM,
			FRACTION_UNIT_TWO, FRACTION_UNIT_TWO,
			FRACTION_UNIT_TWO, FRACTION_UNIT_TWO,
		).
		Ensure()
}

// ARCCOSINE_ANGLE_MINIMUM is the smallest angle an arccosine returns.
const ARCCOSINE_ANGLE_MINIMUM int64 = 0

// ARCCOSINE_ANGLE_MAXIMUM is the largest angle an arccosine returns. An arccosine is a
// quarter turn less the arcsine, thus the smallest arcsine gives twice the quarter turn.
// That doubling misses PI by a unit, because the quarter turn rounds down on its own.
const ARCCOSINE_ANGLE_MAXIMUM int64 = 2 * int64(PI_HALF)

// Arccosine_Angle is the angle an arccosine returns.
type Arccosine_Angle fixedpoint.Number

// Arccosine_Angle_Invariants bounds an angle to the half turn above zero.
func Arccosine_Angle_Invariants(value Arccosine_Angle, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value), ARCCOSINE_ANGLE_MINIMUM, ARCCOSINE_ANGLE_MAXIMUM,
			FRACTION_UNIT_ONE, FRACTION_UNIT_TWO,
			FRACTION_UNIT_TWO, FRACTION_UNIT_TWO,
		).
		Ensure()
}

// TANGENT_MINIMUM is the smallest tangent this package reports.
const TANGENT_MINIMUM int64 = -TANGENT_MAXIMUM

// TANGENT_MAXIMUM is the largest tangent this package reports. A true tangent grows
// without bound at a quarter turn, thus the value saturates here, which is the largest
// quotient the angle grid produces on its own.
const TANGENT_MAXIMUM int64 = 366502128298

// Tangent_Value is the tangent of an angle.
type Tangent_Value fixedpoint.Number

// Tangent_Value_Invariants bounds a tangent to the quotient the division can produce. The
// smallest sine above zero is three units, thus a tangent of one or two units would need a
// cosine above one and cannot occur.
func Tangent_Value_Invariants(value Tangent_Value, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value), TANGENT_MINIMUM, TANGENT_MAXIMUM,
			FRACTION_UNIT_NEGATIVE_ONE, FRACTION_UNIT_ONE,
			FRACTION_UNIT_TWO, FRACTION_UNIT_TWO,
		).
		Ensure()
}

// Hyperbolic_Angle is an angle a hyperbolic function accepts.
type Hyperbolic_Angle fixedpoint.Number

// Hyperbolic_Angle_Invariants bounds an angle to the range the Number storage holds.
func Hyperbolic_Angle_Invariants(value Hyperbolic_Angle, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), HYPERBOLIC_ANGLE_MINIMUM, HYPERBOLIC_ANGLE_MAXIMUM).
		Ensure()
}

// Hyperbolic_Value is a hyperbolic sine or cosine.
type Hyperbolic_Value fixedpoint.Number

// Hyperbolic_Value_Invariants bounds a hyperbolic sine to the values it reaches.
func Hyperbolic_Value_Invariants(value Hyperbolic_Value, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Holed_Int64(
			int64(value), HYPERBOLIC_MINIMUM, HYPERBOLIC_MAXIMUM,
			FRACTION_UNIT_NEGATIVE_ONE, FRACTION_UNIT_ONE,
			FRACTION_UNIT_TWO, FRACTION_UNIT_TWO,
		).
		Ensure()
}

// Hyperbolic_Cosine_Value is a hyperbolic cosine, which is never below one.
type Hyperbolic_Cosine_Value fixedpoint.Number

// Hyperbolic_Cosine_Value_Invariants bounds a hyperbolic cosine, whose domain starts at
// one rather than at the negative end a hyperbolic sine reaches.
func Hyperbolic_Cosine_Value_Invariants(
	value Hyperbolic_Cosine_Value, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), HYPERBOLIC_COSINE_MINIMUM, HYPERBOLIC_MAXIMUM).
		Ensure()
}

// Absolute_Integer returns the magnitude of a signed integer.
func Absolute_Integer(value Negatable_Integer) (magnitude Magnitude_Integer) {
	defer func() { Magnitude_Integer_Invariants(magnitude, "absolute_integer.magnitude") }()
	Negatable_Integer_Invariants(value, "absolute_integer.value")
	if value < 0 {
		return Magnitude_Integer(-int64(value))
	}
	return Magnitude_Integer(value)
}

// Minimum_Integer returns the smaller of two integers.
func Minimum_Integer(first First_Integer, second Second_Integer) (smallest Ordered_Integer) {
	defer func() { Ordered_Integer_Invariants(smallest, "minimum_integer.smallest") }()
	First_Integer_Invariants(first, "minimum_integer.first")
	Second_Integer_Invariants(second, "minimum_integer.second")
	if int64(first) < int64(second) {
		return Ordered_Integer(first)
	}
	return Ordered_Integer(second)
}

// Maximum_Integer returns the larger of two integers.
func Maximum_Integer(first First_Integer, second Second_Integer) (largest Ordered_Integer) {
	defer func() { Ordered_Integer_Invariants(largest, "maximum_integer.largest") }()
	First_Integer_Invariants(first, "maximum_integer.first")
	Second_Integer_Invariants(second, "maximum_integer.second")
	if int64(first) > int64(second) {
		return Ordered_Integer(first)
	}
	return Ordered_Integer(second)
}

// Truncate discards the fraction toward zero.
func Truncate(value Value) (whole Whole_Number) {
	defer func() { Whole_Number_Invariants(whole, "truncate.whole") }()
	Value_Invariants(value, "truncate.value")
	return Whole_Number(int64(value) / SCALE * SCALE)
}

// Floor returns the largest whole Number that is not larger than the value. Integer
// division truncates toward zero, thus a negative value with a fraction needs one step
// down.
func Floor(value Value) (whole Whole_Number) {
	defer func() {
		Whole_Number_Invariants(whole, "floor.whole")
		// The result brackets the value from below by less than one whole. A shift that
		// leaves the storage range breaks this even when the result still looks whole.
		invariant.Always(
			int64(whole) <= int64(value),
			"A floor never exceeds its own value.",
		)
		invariant.Always(
			int64(value)-int64(whole) < SCALE,
			"A floor is within one whole of its own value.",
		)
	}()
	Value_Invariants(value, "floor.value")
	truncated := int64(value) / SCALE * SCALE
	if int64(value) < 0 {
		if truncated != int64(value) {
			return Whole_Number(truncated - SCALE)
		}
	}
	return Whole_Number(truncated)
}

// Ceiling returns the smallest whole Number that is not smaller than the value.
func Ceiling(value Value) (whole Whole_Number) {
	defer func() {
		Whole_Number_Invariants(whole, "ceiling.whole")
		invariant.Always(
			int64(whole) >= int64(value),
			"A ceiling is never below its own value.",
		)
		invariant.Always(
			int64(whole)-int64(value) < SCALE,
			"A ceiling is within one whole of its own value.",
		)
	}()
	Value_Invariants(value, "ceiling.value")
	truncated := int64(value) / SCALE * SCALE
	if int64(value) > 0 {
		if truncated != int64(value) {
			return Whole_Number(truncated + SCALE)
		}
	}
	return Whole_Number(truncated)
}

// Round returns the nearest whole Number and moves a half away from zero. It reads the
// fraction rather than shifting the value by a half first, because a shift at either end
// of the domain would leave the storage range.
func Round(value Value) (whole Whole_Number) {
	defer func() {
		Whole_Number_Invariants(whole, "round.whole")
		// The nearest whole is never more than a half away. A shift past the end of the
		// storage lands far outside this, whatever the sign it wraps to.
		distance := int64(value) - int64(whole)
		if distance < 0 {
			distance = -distance
		}
		invariant.Always(
			distance <= HALF,
			"A rounded value is within one half of its own value.",
		)
	}()
	Value_Invariants(value, "round.value")
	truncated := int64(value) / SCALE * SCALE
	fraction := int64(value) - truncated
	if fraction >= HALF {
		return Whole_Number(truncated + SCALE)
	}
	if fraction <= -HALF {
		return Whole_Number(truncated - SCALE)
	}
	return Whole_Number(truncated)
}

// Modulo returns the remainder of a truncated division and keeps the sign of the dividend.
// A zero divisor gives zero, which matches fixedpoint division.
func Modulo(dividend Dividend, divisor Divisor) (remainder Remainder) {
	defer func() { Remainder_Invariants(remainder, "modulo.remainder") }()
	Dividend_Invariants(dividend, "modulo.dividend")
	Divisor_Invariants(divisor, "modulo.divisor")
	if divisor == 0 {
		return 0
	}
	return Remainder(int64(dividend) % int64(divisor))
}

// Square_Root returns the square root of a Number that is not negative.
func Square_Root(value Radicand) (root Root) {
	defer func() {
		Root_Invariants(root, "square_root.root")
		// The square of a floored root never passes the radicand. The square stays in
		// range because the root is the floor, thus this needs no wider arithmetic.
		square := fixedpoint.Multiply(
			fixedpoint.Multiplicand(root), fixedpoint.Multiplier(root))
		invariant.Always(
			int64(square) <= int64(value),
			"The square of a root never exceeds its radicand.",
		)
	}()
	Radicand_Invariants(value, "square_root.value")
	return Root(fixedpoint.Square_Root(fixedpoint.Number(value)))
}

// Cube_Root returns the cube root of a Number of either sign. Newton's iteration converges
// on the root, and the magnitude carries the sign back at the end.
func Cube_Root(value Cube_Radicand) (root Cube_Root_Value) {
	defer func() {
		Cube_Root_Value_Invariants(root, "cube_root.root")
		// A cube keeps the sign of its root, thus the two always agree. An estimate that
		// overflows lands on a value of the wrong sign while still looking like a root.
		invariant.Always(
			(int64(root) < 0) == (int64(value) < 0),
			"A cube root carries the sign of its radicand.",
		)
	}()
	Cube_Radicand_Invariants(value, "cube_root.value")
	if value == 0 {
		return 0
	}
	magnitude := fixedpoint.Number(value)
	negative := false
	if value < 0 {
		negative = true
		magnitude = -magnitude
	}
	// The most negative Number negates to itself, so its magnitude is one unit beyond the
	// storage. The two radicands differ by one part in nine quintillion, thus their cube
	// roots land on the same grid point and the largest magnitude stands in for it.
	if magnitude < 0 {
		magnitude = fixedpoint.Number(bits.INTEGER_64_MAXIMUM)
	}
	estimate := fixedpoint.Number(cube_root_estimate(Cube_Magnitude(magnitude)))
	for index := 0; index < 64; index++ {
		square := fixedpoint.Multiply(
			fixedpoint.Multiplicand(estimate), fixedpoint.Multiplier(estimate))
		if square == 0 {
			break
		}
		quotient := fixedpoint.Divide(
			fixedpoint.Dividend(magnitude), fixedpoint.Divisor(square))
		next := (2*estimate + quotient) / 3
		difference := next - estimate
		if difference < 0 {
			difference = -difference
		}
		estimate = next
		if difference <= 1 {
			break
		}
	}
	if negative {
		return Cube_Root_Value(-estimate)
	}
	return Cube_Root_Value(estimate)
}

// Gives the starting estimate of a cube root. One third of the exponent puts the estimate
// within a factor of two of the root, which keeps the square of the estimate inside the
// storage range. Squaring the radicand itself would leave that range at once.
func cube_root_estimate(magnitude Cube_Magnitude) (estimate Cube_Estimate) {
	defer func() { Cube_Estimate_Invariants(estimate, "cube_root_estimate.estimate") }()
	Cube_Magnitude_Invariants(magnitude, "cube_root_estimate.magnitude")
	residue := uint64(magnitude)
	position := int64(-1)
	for _, step := range []uint{32, 16, 8, 4, 2, 1} {
		if residue>>step != 0 {
			residue >>= step
			position += int64(step)
		}
	}
	if residue >= 1 {
		position++
	}
	exponent := position - FRACTION_BIT_COUNT
	third := exponent / 3
	if exponent < 0 {
		if exponent%3 != 0 {
			third--
		}
	}
	return Cube_Estimate(int64(1) << uint(third+FRACTION_BIT_COUNT))
}

// Hypotenuse returns the root of the sum of two squares. Dividing the smaller leg by the
// larger keeps the squared ratio inside the unit interval, thus a large leg cannot
// overflow the sum.
func Hypotenuse(opposite Leg_Opposite, adjacent Leg_Adjacent) (distance Distance) {
	defer func() {
		Distance_Invariants(distance, "hypotenuse.distance")
		// The hypotenuse is the longest side, thus it reaches at least as far as either
		// leg. A product that overflows falls short of this, or turns negative.
		opposite_reach := int64(opposite)
		if opposite_reach < 0 {
			opposite_reach = -opposite_reach
		}
		adjacent_reach := int64(adjacent)
		if adjacent_reach < 0 {
			adjacent_reach = -adjacent_reach
		}
		invariant.Always(
			int64(distance) >= opposite_reach,
			"A hypotenuse reaches at least as far as the opposite leg.",
		)
		invariant.Always(
			int64(distance) >= adjacent_reach,
			"A hypotenuse reaches at least as far as the adjacent leg.",
		)
	}()
	Leg_Opposite_Invariants(opposite, "hypotenuse.opposite")
	Leg_Adjacent_Invariants(adjacent, "hypotenuse.adjacent")
	larger := fixedpoint.Number(opposite)
	if larger < 0 {
		larger = -larger
	}
	smaller := fixedpoint.Number(adjacent)
	if smaller < 0 {
		smaller = -smaller
	}
	if smaller > larger {
		larger, smaller = smaller, larger
	}
	if larger == 0 {
		return 0
	}
	ratio := fixedpoint.Divide(
		fixedpoint.Dividend(smaller), fixedpoint.Divisor(larger))
	square := fixedpoint.Multiply(
		fixedpoint.Multiplicand(ratio), fixedpoint.Multiplier(ratio))
	root := fixedpoint.Square_Root(fixedpoint.Number(SCALE) + square)
	return Distance(fixedpoint.Multiply(
		fixedpoint.Multiplicand(larger), fixedpoint.Multiplier(fixedpoint.Number(root))))
}

// Exponential_2 raises two to a power. The whole part of the exponent is a shift, and each
// set fraction bit selects one root of two from the table, so no series is necessary.
func Exponential_2(exponent Exponent_2) (power Power_Value) {
	defer func() { Power_Value_Invariants(power, "exponential_2.power") }()
	Exponent_2_Invariants(exponent, "exponential_2.exponent")
	whole := int64(exponent) >> FRACTION_BIT_COUNT
	fraction := int64(exponent) - whole*SCALE
	// Each entry is two raised to the reciprocal of two raised to its index, the factor
	// one fraction bit of the exponent contributes. The head entry is never read, so the
	// index matches the bit it stands for.
	table := [ROOT_TABLE_SIZE]fixedpoint.Number{
		0, 1482910, 1246974, 1143480, 1095000, 1071537, 1059994, 1054270,
		1051419, 1049997, 1049286, 1048931, 1048753, 1048665, 1048620,
		1048598, 1048587, 1048582, 1048579, 1048577, 1048577,
	}
	result := fixedpoint.Number(SCALE)
	for index := 1; index <= FRACTION_BIT_COUNT; index++ {
		if fraction&(SCALE>>uint(index)) != 0 {
			result = fixedpoint.Multiply(
				fixedpoint.Multiplicand(result),
				fixedpoint.Multiplier(table[index]))
		}
	}
	if whole > 0 {
		return Power_Value(result << uint(whole))
	}
	if whole < -63 {
		return 0
	}
	if whole < 0 {
		return Power_Value(result >> uint(-whole))
	}
	return Power_Value(result)
}

// Exponential raises E to a power.
func Exponential(exponent Exponent) (power Natural_Power_Value) {
	defer func() { Natural_Power_Value_Invariants(power, "exponential.power") }()
	Exponent_Invariants(exponent, "exponential.exponent")
	binary := fixedpoint.Apply(fixedpoint.Number(exponent), BINARY_LOGARITHM_OF_E)
	return Natural_Power_Value(Exponential_2(Exponent_2(binary)))
}

// Logarithm_2 returns the base-two logarithm of a positive Number. The position of the
// highest set bit gives the whole part, then each squaring of the mantissa resolves one
// more fraction bit.
func Logarithm_2(value Argument) (logarithm Logarithm_2_Value) {
	defer func() { Logarithm_2_Value_Invariants(logarithm, "logarithm_2.logarithm") }()
	Argument_Invariants(value, "logarithm_2.value")
	// A binary search over the halves finds the highest set bit, whose position above the
	// fraction width is the whole part of the logarithm.
	residue := uint64(value)
	position := int64(-1)
	for _, step := range []uint{32, 16, 8, 4, 2, 1} {
		if residue>>step != 0 {
			residue >>= step
			position += int64(step)
		}
	}
	if residue >= 1 {
		position++
	}
	whole := position - FRACTION_BIT_COUNT
	mantissa := fixedpoint.Number(value)
	if whole > 0 {
		mantissa = mantissa >> uint(whole)
	}
	if whole < 0 {
		mantissa = mantissa << uint(-whole)
	}
	result := whole * SCALE
	for index := 0; index < FRACTION_BIT_COUNT; index++ {
		mantissa = fixedpoint.Multiply(
			fixedpoint.Multiplicand(mantissa), fixedpoint.Multiplier(mantissa))
		if mantissa >= 2*SCALE {
			mantissa >>= 1
			result += SCALE >> uint(index+1)
		}
	}
	return Logarithm_2_Value(result)
}

// Logarithm returns the natural logarithm of a positive Number.
func Logarithm(value Argument) (logarithm Logarithm_Value) {
	defer func() { Logarithm_Value_Invariants(logarithm, "logarithm.logarithm") }()
	Argument_Invariants(value, "logarithm.value")
	binary := Logarithm_2(value)
	return Logarithm_Value(fixedpoint.Apply(
		fixedpoint.Number(binary), NATURAL_LOGARITHM_OF_TWO))
}

// Logarithm_10 returns the base-ten logarithm of a positive Number.
func Logarithm_10(value Argument) (logarithm Logarithm_10_Value) {
	defer func() { Logarithm_10_Value_Invariants(logarithm, "logarithm_10.logarithm") }()
	Argument_Invariants(value, "logarithm_10.value")
	binary := Logarithm_2(value)
	return Logarithm_10_Value(fixedpoint.Divide(
		fixedpoint.Dividend(binary), fixedpoint.Divisor(BINARY_LOGARITHM_OF_TEN)))
}

// Power raises a positive base to an exponent. The logarithm and the exponential each
// carry an error, thus the result is less exact than either operation alone.
func Power(base Base, exponent Power_Exponent) (result Power_Value) {
	defer func() { Power_Value_Invariants(result, "power.result") }()
	Base_Invariants(base, "power.base")
	Power_Exponent_Invariants(exponent, "power.exponent")
	binary := Logarithm_2(Argument(base))
	scaled := fixedpoint.Multiply(
		fixedpoint.Multiplicand(binary), fixedpoint.Multiplier(exponent))
	if int64(scaled) < EXPONENT_2_MINIMUM {
		return 0
	}
	return Exponential_2(Exponent_2(scaled))
}

// Sine returns the sine of an angle in radians. The angle converts to turns and the
// fixedpoint sine approximates the result, thus the error reaches about 0.002.
func Sine(angle Angle) (sine Unit_Value) {
	defer func() { Unit_Value_Invariants(sine, "sine.sine") }()
	Angle_Invariants(angle, "sine.angle")
	turns := fixedpoint.Apply(fixedpoint.Number(angle), TURNS_PER_RADIAN)
	return Unit_Value(fixedpoint.Sine_Turns(turns))
}

// Cosine returns the cosine of an angle in radians. A cosine is a sine one quarter turn
// ahead, thus the quarter turn is added in turns and no second approximation is needed.
func Cosine(angle Angle) (cosine Unit_Value) {
	defer func() { Unit_Value_Invariants(cosine, "cosine.cosine") }()
	Angle_Invariants(angle, "cosine.angle")
	turns := fixedpoint.Apply(fixedpoint.Number(angle), TURNS_PER_RADIAN)
	return Unit_Value(fixedpoint.Sine_Turns(turns + QUARTER_TURN))
}

// Tangent returns the tangent of an angle in radians. A zero cosine gives zero, which
// matches fixedpoint division and replaces the infinity a float would give.
func Tangent(angle Angle) (tangent Tangent_Value) {
	defer func() { Tangent_Value_Invariants(tangent, "tangent.tangent") }()
	Angle_Invariants(angle, "tangent.angle")
	sine := Sine(angle)
	cosine := Cosine(angle)
	quotient := fixedpoint.Divide(
		fixedpoint.Dividend(sine), fixedpoint.Divisor(cosine))
	if int64(quotient) > TANGENT_MAXIMUM {
		return Tangent_Value(TANGENT_MAXIMUM)
	}
	if int64(quotient) < TANGENT_MINIMUM {
		return Tangent_Value(TANGENT_MINIMUM)
	}
	return Tangent_Value(quotient)
}

// Arctangent returns the angle whose tangent is the slope. A rational approximation covers
// the unit interval, and the reciprocal identity covers the rest, thus the error reaches
// about 0.0015 radians over the whole domain. The standard library test vectors measure
// that bound, and it is far wider than the grid.
func Arctangent(value Slope) (angle Principal_Angle) {
	defer func() { Principal_Angle_Invariants(angle, "arctangent.angle") }()
	Slope_Invariants(value, "arctangent.value")
	// The most negative slope negates to itself, so its magnitude is no Number. Its
	// reciprocal is below one unit either way, thus the angle is a quarter turn already.
	if int64(value) == bits.INTEGER_64_MINIMUM {
		return Principal_Angle(-PI_HALF)
	}
	magnitude := fixedpoint.Number(value)
	negative := false
	if magnitude < 0 {
		negative = true
		magnitude = -magnitude
	}
	result := fixedpoint.Number(0)
	if magnitude <= fixedpoint.Number(SCALE) {
		result = fixedpoint.Number(arctangent_unit(Unit_Magnitude(magnitude)))
	}
	if magnitude > fixedpoint.Number(SCALE) {
		reciprocal := fixedpoint.Divide(
			fixedpoint.Dividend(SCALE), fixedpoint.Divisor(magnitude))
		result = fixedpoint.Number(PI_HALF) -
			fixedpoint.Number(arctangent_unit(Unit_Magnitude(reciprocal)))
	}
	if negative {
		return Principal_Angle(-result)
	}
	return Principal_Angle(result)
}

// Approximates the arctangent over the closed unit interval by the rational form
// x*PI/4 - x*(x-1)*(A + B*x), which stays within about 0.0015 of the true angle.
func arctangent_unit(value Unit_Magnitude) (angle Unit_Angle) {
	defer func() { Unit_Angle_Invariants(angle, "arctangent_unit.angle") }()
	Unit_Magnitude_Invariants(value, "arctangent_unit.value")
	first := fixedpoint.Multiply(
		fixedpoint.Multiplicand(value), fixedpoint.Multiplier(PI_QUARTER))
	inner := ARCTANGENT_COEFFICIENT_A + fixedpoint.Multiply(
		fixedpoint.Multiplicand(ARCTANGENT_COEFFICIENT_B), fixedpoint.Multiplier(value))
	offset := fixedpoint.Multiply(
		fixedpoint.Multiplicand(value), fixedpoint.Multiplier(value-SCALE))
	return Unit_Angle(first - fixedpoint.Multiply(
		fixedpoint.Multiplicand(offset), fixedpoint.Multiplier(inner)))
}

// Arcsine returns the angle whose sine is the value. The identity through the arctangent
// needs the cosine, and that cosine is zero at each end, thus the ends return the quarter
// turn directly.
func Arcsine(value Unit_Argument) (angle Principal_Angle) {
	defer func() { Principal_Angle_Invariants(angle, "arcsine.angle") }()
	Unit_Argument_Invariants(value, "arcsine.value")
	if int64(value) == UNIT_MAXIMUM {
		return Principal_Angle(PI_HALF)
	}
	if int64(value) == UNIT_MINIMUM {
		return Principal_Angle(-PI_HALF)
	}
	square := fixedpoint.Multiply(
		fixedpoint.Multiplicand(value), fixedpoint.Multiplier(value))
	cosine := fixedpoint.Square_Root(fixedpoint.Number(SCALE) - square)
	slope := fixedpoint.Divide(
		fixedpoint.Dividend(value), fixedpoint.Divisor(cosine))
	return Arctangent(Slope(slope))
}

// Arccosine returns the angle whose cosine is the value.
func Arccosine(value Unit_Argument) (angle Arccosine_Angle) {
	defer func() { Arccosine_Angle_Invariants(angle, "arccosine.angle") }()
	Unit_Argument_Invariants(value, "arccosine.value")
	return Arccosine_Angle(fixedpoint.Number(PI_HALF) -
		fixedpoint.Number(Arcsine(value)))
}

// Arctangent_Quotient returns the angle of the point whose legs are the two operands, in
// the quadrant those signs select.
func Arctangent_Quotient(
	opposite Opposite, adjacent Adjacent,
) (angle Angle_Value) {
	defer func() { Angle_Value_Invariants(angle, "arctangent_quotient.angle") }()
	Opposite_Invariants(opposite, "arctangent_quotient.opposite")
	Adjacent_Invariants(adjacent, "arctangent_quotient.adjacent")
	if adjacent == 0 {
		if opposite > 0 {
			return Angle_Value(PI_HALF)
		}
		if opposite < 0 {
			return Angle_Value(-PI_HALF)
		}
		return 0
	}
	base := fixedpoint.Number(
		principal_arctangent(opposite, Nonzero_Adjacent(adjacent)))
	if adjacent > 0 {
		return Angle_Value(base)
	}
	if opposite < 0 {
		return Angle_Value(base - fixedpoint.Number(PI))
	}
	return Angle_Value(base + fixedpoint.Number(PI))
}

// Gives the arctangent of one leg divided by the other, in the open interval from negative
// a quarter turn to a quarter turn. It always divides the smaller magnitude by the larger,
// because the other order overflows once one leg dwarfs the other.
func principal_arctangent(
	opposite Opposite, adjacent Nonzero_Adjacent,
) (angle Quotient_Angle) {
	defer func() { Quotient_Angle_Invariants(angle, "principal_arctangent.angle") }()
	Opposite_Invariants(opposite, "principal_arctangent.opposite")
	Nonzero_Adjacent_Invariants(adjacent, "principal_arctangent.adjacent")
	// The magnitudes are unsigned, because the most negative Number negates to itself and
	// would compare as the smallest rather than the largest.
	opposite_magnitude := uint64(opposite)
	if opposite < 0 {
		opposite_magnitude = uint64(-int64(opposite))
	}
	adjacent_magnitude := uint64(adjacent)
	if adjacent < 0 {
		adjacent_magnitude = uint64(-int64(adjacent))
	}
	if opposite_magnitude <= adjacent_magnitude {
		slope := fixedpoint.Divide(
			fixedpoint.Dividend(opposite), fixedpoint.Divisor(adjacent))
		return Quotient_Angle(Arctangent(Slope(slope)))
	}
	// The reciprocal identity turns the steep ratio into a shallow one: the arctangent of
	// a quotient is a quarter turn of the quotient's sign, less the arctangent of its
	// reciprocal.
	ratio := fixedpoint.Divide(
		fixedpoint.Dividend(adjacent), fixedpoint.Divisor(opposite))
	inner := fixedpoint.Number(Arctangent(Slope(ratio)))
	if (opposite < 0) != (adjacent < 0) {
		return Quotient_Angle(-fixedpoint.Number(PI_HALF) - inner)
	}
	return Quotient_Angle(fixedpoint.Number(PI_HALF) - inner)
}

// Hyperbolic_Sine returns half the difference of the two exponentials.
func Hyperbolic_Sine(angle Hyperbolic_Angle) (sine Hyperbolic_Value) {
	defer func() { Hyperbolic_Value_Invariants(sine, "hyperbolic_sine.sine") }()
	Hyperbolic_Angle_Invariants(angle, "hyperbolic_sine.angle")
	forward := Exponential(Exponent(angle))
	backward := Exponential(Exponent(-angle))
	return Hyperbolic_Value((fixedpoint.Number(forward) -
		fixedpoint.Number(backward)) / 2)
}

// Hyperbolic_Cosine returns half the sum of the two exponentials.
func Hyperbolic_Cosine(angle Hyperbolic_Angle) (cosine Hyperbolic_Cosine_Value) {
	defer func() {
		Hyperbolic_Cosine_Value_Invariants(cosine, "hyperbolic_cosine.cosine")
	}()
	Hyperbolic_Angle_Invariants(angle, "hyperbolic_cosine.angle")
	forward := Exponential(Exponent(angle))
	backward := Exponential(Exponent(-angle))
	return Hyperbolic_Cosine_Value((fixedpoint.Number(forward) +
		fixedpoint.Number(backward)) / 2)
}

// Hyperbolic_Tangent returns the ratio of the hyperbolic sine to the hyperbolic cosine.
func Hyperbolic_Tangent(angle Hyperbolic_Angle) (tangent Unit_Value) {
	defer func() { Unit_Value_Invariants(tangent, "hyperbolic_tangent.tangent") }()
	Hyperbolic_Angle_Invariants(angle, "hyperbolic_tangent.angle")
	sine := Hyperbolic_Sine(angle)
	cosine := Hyperbolic_Cosine(angle)
	return Unit_Value(fixedpoint.Divide(
		fixedpoint.Dividend(sine), fixedpoint.Divisor(cosine)))
}
