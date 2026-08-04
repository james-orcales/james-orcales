package scalar_test

import (
	"testing"

	"local/james-orcales/shared/math/fixedpoint"
	"local/james-orcales/shared/math/scalar"
	"local/james-orcales/shared/testify"
)

// Test_Integer_Limits verifies that each width limit holds the value the width admits and
// that one more step would leave the width.
func Test_Integer_Limits(t *testing.T) {
	t.Parallel()
	// A constant cannot hold one step past its own width, thus the step happens at run
	// time where the value wraps instead.
	signed_8 := scalar.INTEGER_8_MAXIMUM
	signed_8++
	testify.Equal_Values(t, scalar.INTEGER_8_MINIMUM, signed_8,
		"the 8-bit limits must wrap into each other")

	signed_16 := scalar.INTEGER_16_MAXIMUM
	signed_16++
	testify.Equal_Values(t, scalar.INTEGER_16_MINIMUM, signed_16,
		"the 16-bit limits must wrap into each other")

	signed_32 := scalar.INTEGER_32_MAXIMUM
	signed_32++
	testify.Equal_Values(t, scalar.INTEGER_32_MINIMUM, signed_32,
		"the 32-bit limits must wrap into each other")

	signed_64 := scalar.INTEGER_64_MAXIMUM
	signed_64++
	testify.Equal_Values(t, scalar.INTEGER_64_MINIMUM, signed_64,
		"the 64-bit limits must wrap into each other")

	unsigned_8 := scalar.UNSIGNED_8_MAXIMUM
	unsigned_8++
	testify.Equal_Values(t, scalar.UNSIGNED_MINIMUM, unsigned_8,
		"the unsigned 8-bit limit must wrap to zero")

	unsigned_64 := scalar.UNSIGNED_64_MAXIMUM
	unsigned_64++
	testify.Equal_Values(t, scalar.UNSIGNED_MINIMUM, unsigned_64,
		"the unsigned 64-bit limit must wrap to zero")

	machine := scalar.INTEGER_MAXIMUM
	machine++
	testify.Equal_Values(t, scalar.INTEGER_MINIMUM, machine,
		"the machine limits must wrap into each other")

	unsigned_machine := scalar.UNSIGNED_MAXIMUM
	unsigned_machine++
	testify.Equal_Values(t, scalar.UNSIGNED_MINIMUM, unsigned_machine,
		"the unsigned machine limit must wrap to zero")
	testify.Equal_Values(t, 65535, scalar.UNSIGNED_16_MAXIMUM,
		"UNSIGNED_16_MAXIMUM = %d, want 65535", scalar.UNSIGNED_16_MAXIMUM)
	testify.Equal_Values(t, 4294967295, scalar.UNSIGNED_32_MAXIMUM,
		"UNSIGNED_32_MAXIMUM = %d, want 4294967295", scalar.UNSIGNED_32_MAXIMUM)

}

// Test_Integer_Arithmetic verifies the magnitude of a signed integer and the two ordering
// selections, at the ends of the domain and around zero.
func Test_Integer_Arithmetic(t *testing.T) {
	t.Parallel()
	testify.Equal_Values(t, 5, scalar.Absolute_Integer(-5),
		"Absolute_Integer(-5) = %d, want 5", scalar.Absolute_Integer(-5))
	testify.Equal_Values(t, 5, scalar.Absolute_Integer(5),
		"Absolute_Integer(5) = %d, want 5", scalar.Absolute_Integer(5))
	testify.Equal_Values(t, 0, scalar.Absolute_Integer(0),
		"Absolute_Integer(0) = %d, want 0", scalar.Absolute_Integer(0))

	largest := scalar.Signed_Integer(scalar.INTEGER_64_MAXIMUM)
	testify.Equal_Values(t, scalar.INTEGER_64_MAXIMUM, int64(scalar.Absolute_Integer(largest)),
		"the largest integer is its own magnitude")
	testify.Equal_Values(t, scalar.INTEGER_64_MAXIMUM, int64(scalar.Absolute_Integer(-largest)),
		"the negation of the largest integer has the same magnitude")
	testify.Equal_Values(t, 2, scalar.Minimum_Integer(2, 3),
		"Minimum_Integer(2,3) = %d, want 2", scalar.Minimum_Integer(2, 3))
	testify.Equal_Values(t, 2, scalar.Minimum_Integer(3, 2),
		"Minimum_Integer(3,2) = %d, want 2", scalar.Minimum_Integer(3, 2))
	testify.Equal_Values(t, 3, scalar.Maximum_Integer(2, 3),
		"Maximum_Integer(2,3) = %d, want 3", scalar.Maximum_Integer(2, 3))
	testify.Equal_Values(t, 3, scalar.Maximum_Integer(3, 2),
		"Maximum_Integer(3,2) = %d, want 3", scalar.Maximum_Integer(3, 2))

	smallest := scalar.First_Integer(scalar.INTEGER_64_MINIMUM)
	testify.Equal_Values(t,
		scalar.INTEGER_64_MINIMUM,
		int64(scalar.Minimum_Integer(smallest, 0)),
		"the smallest integer must win a minimum")
	testify.Equal_Values(t, 0, int64(scalar.Maximum_Integer(smallest, 0)),
		"the smallest integer must lose a maximum")

}

// Test_Constants verifies that each constant sits within one grid unit of its true value,
// which the grid of about six decimal digits allows.
func Test_Constants(t *testing.T) {
	t.Parallel()
	// PI is 3.14159265, thus 3.14159265 * SCALE rounds to 3294199.
	testify.Equal_Values(t, 3294199, scalar.PI,
		"PI = %d, want 3294199", scalar.PI)
	// Each part rounds on its own, thus the sum of the parts can miss PI by a unit.
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: scalar.PI, Actual: scalar.PI_HALF * 2, Delta: 2},
		"PI_HALF doubled = %d, want PI",
		scalar.PI_HALF*2)
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: scalar.PI,
			Actual: scalar.PI_QUARTER * 4,
			Delta:  4},
		"PI_QUARTER quadrupled = %d, want PI",
		scalar.PI_QUARTER*4)
	// E is 2.71828182, thus E * SCALE rounds to 2850325.
	testify.Equal_Values(t, 2850325, scalar.E,
		"E = %d, want 2850325", scalar.E)
	// The natural logarithm of two is 0.69314718.
	testify.Equal_Values(t, 726817, scalar.NATURAL_LOGARITHM_OF_TWO,
		"NATURAL_LOGARITHM_OF_TWO = %d", scalar.NATURAL_LOGARITHM_OF_TWO)
	// The base-two logarithm of E is 1.44269504.
	testify.Equal_Values(t, 1512775, scalar.BINARY_LOGARITHM_OF_E,
		"BINARY_LOGARITHM_OF_E = %d", scalar.BINARY_LOGARITHM_OF_E)
	// The base-two logarithm of ten is 3.32192809.
	testify.Equal_Values(t, 3483294, scalar.BINARY_LOGARITHM_OF_TEN,
		"BINARY_LOGARITHM_OF_TEN = %d", scalar.BINARY_LOGARITHM_OF_TEN)

}

// Test_Whole_Numbers verifies the four whole-number selections and the remainder, on each
// side of zero and at a value that already carries no fraction.
func Test_Whole_Numbers(t *testing.T) {
	t.Parallel()
	half := whole(7) / 2
	testify.Equal_Values(t, whole(3), fixedpoint.Number(scalar.Floor(scalar.Value(half))),
		"Floor(3.5) = %d, want 3", scalar.Floor(scalar.Value(half)))
	testify.Equal_Values(t, whole(4), fixedpoint.Number(scalar.Ceiling(scalar.Value(half))),
		"Ceiling(3.5) = %d, want 4", scalar.Ceiling(scalar.Value(half)))
	testify.Equal_Values(t, whole(3), fixedpoint.Number(scalar.Truncate(scalar.Value(half))),
		"Truncate(3.5) = %d, want 3", scalar.Truncate(scalar.Value(half)))
	testify.Equal_Values(t, whole(4), fixedpoint.Number(scalar.Round(scalar.Value(half))),
		"Round(3.5) = %d, want 4", scalar.Round(scalar.Value(half)))

	negative := -half
	testify.Equal_Values(t, whole(-4), fixedpoint.Number(scalar.Floor(scalar.Value(negative))),
		"Floor(-3.5) = %d, want -4", scalar.Floor(scalar.Value(negative)))
	testify.Equal_Values(t,
		whole(-3),
		fixedpoint.Number(scalar.Ceiling(scalar.Value(negative))),
		"Ceiling(-3.5) = %d, want -3", scalar.Ceiling(scalar.Value(negative)))
	testify.Equal_Values(t,
		whole(-3),
		fixedpoint.Number(scalar.Truncate(scalar.Value(negative))),
		"Truncate(-3.5) = %d, want -3", scalar.Truncate(scalar.Value(negative)))
	// Round moves a half away from zero, thus negative three and a half gives minus four.
	testify.Equal_Values(t, whole(-4), fixedpoint.Number(scalar.Round(scalar.Value(negative))),
		"Round(-3.5) = %d, want -4", scalar.Round(scalar.Value(negative)))
	testify.Equal_Values(t, whole(3), fixedpoint.Number(scalar.Floor(scalar.Value(whole(3)))),
		"Floor leaves a whole number unchanged")
	testify.Equal_Values(t, whole(3), fixedpoint.Number(scalar.Ceiling(scalar.Value(whole(3)))),
		"Ceiling leaves a whole number unchanged")

	remainder := scalar.Modulo(scalar.Dividend(whole(7)), scalar.Divisor(whole(2)))
	testify.Equal_Values(t, whole(1), fixedpoint.Number(remainder),
		"Modulo(7,2) = %d, want 1", remainder)

	negative_remainder := scalar.Modulo(scalar.Dividend(whole(-7)), scalar.Divisor(whole(2)))
	testify.Equal_Values(t, whole(-1), fixedpoint.Number(negative_remainder),
		"Modulo(-7,2) = %d, want -1", negative_remainder)
	testify.Equal_Values(t, 0, scalar.Modulo(scalar.Dividend(whole(7)), scalar.Divisor(0)),
		"a zero divisor gives a zero remainder")

}

// Test_Roots verifies the square root, the cube root of each sign, and the hypotenuse of
// the triangle whose sides are three, four, and five.
func Test_Roots(t *testing.T) {
	t.Parallel()
	root := scalar.Square_Root(scalar.Radicand(whole(9)))
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: whole(3),
			Actual: fixedpoint.Number(root),
			Delta:  2},
		"Square_Root(9) = %d, want 3",
		root)
	testify.Equal_Values(t, 0, scalar.Square_Root(scalar.Radicand(0)),
		"Square_Root(0) must be 0")

	two := scalar.Square_Root(scalar.Radicand(whole(2)))
	// The square root of two is 1.41421356, thus the grid holds 1482910.
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: 1482910,
			Actual: fixedpoint.Number(two),
			Delta:  2},
		"Square_Root(2) = %d, want about 1482910",
		two)

	cube := scalar.Cube_Root(scalar.Cube_Radicand(whole(27)))
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: whole(3),
			Actual: fixedpoint.Number(cube),
			Delta:  4},
		"Cube_Root(27) = %d, want 3",
		cube)

	negative_cube := scalar.Cube_Root(scalar.Cube_Radicand(whole(-27)))
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: whole(-3),
			Actual: fixedpoint.Number(negative_cube),
			Delta:  4},
		"Cube_Root(-27) = %d, want -3",
		negative_cube)
	testify.Equal_Values(t, 0, scalar.Cube_Root(scalar.Cube_Radicand(0)),
		"Cube_Root(0) must be 0")

	distance := scalar.Hypotenuse(scalar.Leg_Opposite(whole(3)), scalar.Leg_Adjacent(whole(4)))
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: whole(5),
			Actual: fixedpoint.Number(distance),
			Delta:  8},
		"Hypotenuse(3,4) = %d, want 5",
		distance)

	// The sign of a leg does not change the distance.
	mirrored := scalar.Hypotenuse(
		scalar.Leg_Opposite(whole(-3)), scalar.Leg_Adjacent(whole(-4)))
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: whole(5),
			Actual: fixedpoint.Number(mirrored),
			Delta:  8},
		"Hypotenuse(-3,-4) = %d, want 5",
		mirrored)
	testify.Equal_Values(t,
		0,
		scalar.Hypotenuse(scalar.Leg_Opposite(0), scalar.Leg_Adjacent(0)),
		"Hypotenuse(0,0) must be 0")

}

// Test_Exponential verifies the two exponentials at a whole power, at a fraction, at zero,
// and at the ends of the domain.
func Test_Exponential(t *testing.T) {
	t.Parallel()
	testify.Equal_Values(t, whole(1), fixedpoint.Number(scalar.Exponential_2(0)),
		"Exponential_2(0) = %d, want 1", scalar.Exponential_2(0))
	testify.Equal_Values(t,
		whole(1024),
		fixedpoint.Number(scalar.Exponential_2(scalar.Exponent_2(whole(10)))),
		"Exponential_2(10) = %d, want 1024",
		scalar.Exponential_2(scalar.Exponent_2(whole(10))))
	testify.Equal_Values(t,
		whole(1)/2,
		fixedpoint.Number(scalar.Exponential_2(scalar.Exponent_2(whole(-1)))),
		"Exponential_2(-1) = %d, want one half",
		scalar.Exponential_2(scalar.Exponent_2(whole(-1))))

	// Two raised to one half is the square root of two.
	root := scalar.Exponential_2(scalar.Exponent_2(whole(1) / 2))
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: 1482910,
			Actual: fixedpoint.Number(root),
			Delta:  8},
		"Exponential_2(0.5) = %d, want about 1482910",
		root)

	largest := scalar.Exponential_2(scalar.Exponent_2(scalar.EXPONENT_2_MAXIMUM))
	testify.Equal_Values(t, scalar.POWER_MAXIMUM, int64(largest),
		"Exponential_2 at the maximum = %d, want %d",
		largest, scalar.POWER_MAXIMUM)
	testify.Equal_Values(t,
		0,
		scalar.Exponential_2(scalar.Exponent_2(scalar.EXPONENT_2_MINIMUM)),
		"an exponent at the minimum underflows to zero")
	testify.Equal_Values(t, whole(1), fixedpoint.Number(scalar.Exponential(0)),
		"Exponential(0) = %d, want 1", scalar.Exponential(0))

	// E raised to one is E. The exponent passes through a base-two conversion first, thus
	// two round steps separate the result from the constant.
	natural := scalar.Exponential(scalar.Exponent(whole(1)))
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: fixedpoint.Number(scalar.E),
			Actual: fixedpoint.Number(natural),
			Delta:  32},
		"Exponential(1) = %d, want %d",
		natural,
		scalar.E)

}

// Test_Logarithm verifies the three logarithms at their exact points and at the ends of
// the domain.
func Test_Logarithm(t *testing.T) {
	t.Parallel()
	testify.Equal_Values(t, 0, scalar.Logarithm_2(scalar.Argument(whole(1))),
		"Logarithm_2(1) = %d, want 0",
		scalar.Logarithm_2(scalar.Argument(whole(1))))
	testify.Equal_Values(t,
		whole(1),
		fixedpoint.Number(scalar.Logarithm_2(scalar.Argument(whole(2)))),
		"Logarithm_2(2) = %d, want 1",
		scalar.Logarithm_2(scalar.Argument(whole(2))))
	testify.Equal_Values(t,
		whole(10),
		fixedpoint.Number(scalar.Logarithm_2(scalar.Argument(whole(1024)))),
		"Logarithm_2(1024) = %d, want 10",
		scalar.Logarithm_2(scalar.Argument(whole(1024))))
	testify.Equal_Values(t,
		whole(-1),
		fixedpoint.Number(scalar.Logarithm_2(scalar.Argument(whole(1)/2))),
		"Logarithm_2(0.5) = %d, want -1",
		scalar.Logarithm_2(scalar.Argument(whole(1)/2)))

	smallest := scalar.Logarithm_2(scalar.Argument(1))
	testify.Equal_Values(t, scalar.LOGARITHM_2_MINIMUM, int64(smallest),
		"Logarithm_2 of one unit = %d, want %d",
		smallest, scalar.LOGARITHM_2_MINIMUM)

	largest := scalar.Logarithm_2(scalar.Argument(scalar.INTEGER_64_MAXIMUM))
	testify.Equal_Values(t, scalar.LOGARITHM_2_MAXIMUM, int64(largest),
		"Logarithm_2 of the largest Number = %d, want %d",
		largest, scalar.LOGARITHM_2_MAXIMUM)
	testify.Equal_Values(t, 0, scalar.Logarithm(scalar.Argument(whole(1))),
		"Logarithm(1) must be 0")

	// The natural logarithm of E is one.
	natural := scalar.Logarithm(scalar.Argument(fixedpoint.Number(scalar.E)))
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: whole(1),
			Actual: fixedpoint.Number(natural),
			Delta:  8},
		"Logarithm(E) = %d, want 1",
		natural)
	testify.Equal_Values(t, 0, scalar.Logarithm_10(scalar.Argument(whole(1))),
		"Logarithm_10(1) must be 0")

	decade := scalar.Logarithm_10(scalar.Argument(whole(1000)))
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: whole(3),
			Actual: fixedpoint.Number(decade),
			Delta:  8},
		"Logarithm_10(1000) = %d, want 3",
		decade)

}

// Test_Power verifies the power at a whole exponent, at zero, at one, and at a root.
func Test_Power(t *testing.T) {
	t.Parallel()
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: whole(1),
			Actual: fixedpoint.Number(scalar.Power(scalar.Base(whole(2)), 0)),
			Delta:  2},
		"Power(2,0) = %d, want 1",
		scalar.Power(scalar.Base(whole(2)), 0))

	square := scalar.Power(scalar.Base(whole(3)), scalar.Power_Exponent(whole(2)))
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: whole(9),
			Actual: fixedpoint.Number(square),
			Delta:  32},
		"Power(3,2) = %d, want 9",
		square)

	cube := scalar.Power(scalar.Base(whole(2)), scalar.Power_Exponent(whole(10)))
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: whole(1024),
			Actual: fixedpoint.Number(cube),
			Delta:  4096},
		"Power(2,10) = %d, want 1024",
		cube)

	// A half exponent is a square root.
	root := scalar.Power(scalar.Base(whole(9)), scalar.Power_Exponent(whole(1)/2))
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: whole(3),
			Actual: fixedpoint.Number(root),
			Delta:  32},
		"Power(9,0.5) = %d, want 3",
		root)

	// A negative exponent is a reciprocal.
	reciprocal := scalar.Power(scalar.Base(whole(2)), scalar.Power_Exponent(whole(-2)))
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: whole(1) / 4,
			Actual: fixedpoint.Number(reciprocal),
			Delta:  16},
		"Power(2,-2) = %d, want one quarter",
		reciprocal)
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: whole(1),
			Actual: fixedpoint.Number(scalar.Power(scalar.Base(whole(5)), 0)),
			Delta:  2},
		"any base raised to zero is one")

}

// Test_Trigonometry verifies the sine, the cosine, and the tangent at the quarter turns
// and at zero. The underlying approximation carries an error near 0.002, thus the
// tolerance is wider than the grid.
func Test_Trigonometry(t *testing.T) {
	t.Parallel()
	tolerance := fixedpoint.Number(3000)
	testify.Equal_Values(t, 0, scalar.Sine(0),
		"Sine(0) = %d, want 0", scalar.Sine(0))
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: whole(1),
			Actual: fixedpoint.Number(scalar.Cosine(0)),
			Delta:  tolerance},
		"Cosine(0) = %d, want 1",
		scalar.Cosine(0))

	quarter := scalar.Angle(scalar.PI_HALF)
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: whole(1),
			Actual: fixedpoint.Number(scalar.Sine(quarter)),
			Delta:  tolerance},
		"Sine(PI/2) = %d, want 1",
		scalar.Sine(quarter))
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: 0,
			Actual: fixedpoint.Number(scalar.Cosine(quarter)),
			Delta:  tolerance},
		"Cosine(PI/2) = %d, want 0",
		scalar.Cosine(quarter))

	half := scalar.Angle(scalar.PI)
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: 0,
			Actual: fixedpoint.Number(scalar.Sine(half)),
			Delta:  tolerance},
		"Sine(PI) = %d, want 0",
		scalar.Sine(half))
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: whole(-1),
			Actual: fixedpoint.Number(scalar.Cosine(half)),
			Delta:  tolerance},
		"Cosine(PI) = %d, want -1",
		scalar.Cosine(half))
	// The sine is odd, thus a negative angle mirrors the value.
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: whole(-1),
			Actual: fixedpoint.Number(scalar.Sine(-quarter)),
			Delta:  tolerance},
		"Sine(-PI/2) = %d, want -1",
		scalar.Sine(-quarter))
	testify.Equal_Values(t, 0, scalar.Tangent(0),
		"Tangent(0) = %d, want 0", scalar.Tangent(0))

	eighth := scalar.Angle(scalar.PI_QUARTER)
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: whole(1),
			Actual: fixedpoint.Number(scalar.Tangent(eighth)),
			Delta:  tolerance * 4},
		"Tangent(PI/4) = %d, want 1",
		scalar.Tangent(eighth))

}

// Test_Inverse_Trigonometry verifies the three inverse functions and the quadrant that
// Arctangent_Quotient selects.
func Test_Inverse_Trigonometry(t *testing.T) {
	t.Parallel()
	tolerance := fixedpoint.Number(600)
	testify.Equal_Values(t, 0, scalar.Arcsine(0),
		"Arcsine(0) = %d, want 0", scalar.Arcsine(0))
	testify.Equal_Values(t,
		fixedpoint.Number(scalar.PI_HALF),
		fixedpoint.Number(scalar.Arcsine(scalar.Unit_Argument(SCALE))),

		"Arcsine(1) must be a quarter turn")
	testify.Equal_Values(t,
		-fixedpoint.Number(scalar.PI_HALF),
		fixedpoint.Number(scalar.Arcsine(scalar.Unit_Argument(-SCALE))),

		"Arcsine(-1) must be a negative quarter turn")
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: fixedpoint.Number(scalar.PI_HALF),
			Actual: fixedpoint.Number(scalar.Arccosine(0)),
			Delta:  tolerance},
		"Arccosine(0) = %d, want a quarter turn",
		scalar.Arccosine(0))
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: 0,
			Actual: fixedpoint.Number(scalar.Arccosine(scalar.Unit_Argument(SCALE))),
			Delta:  tolerance},
		"Arccosine(1) must be zero")
	testify.Equal_Values(t, 0, scalar.Arctangent(0),
		"Arctangent(0) = %d, want 0", scalar.Arctangent(0))
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: fixedpoint.Number(scalar.PI_QUARTER),
			Actual: fixedpoint.Number(scalar.Arctangent(scalar.Slope(SCALE))),
			Delta:  tolerance},
		"Arctangent(1) = %d, want an eighth turn",
		scalar.Arctangent(scalar.Slope(SCALE)))

	// A large slope approaches a quarter turn but does not reach it. The arctangent of a
	// thousand is 1.56979632, thus the grid holds about 1646051.
	steep := scalar.Arctangent(scalar.Slope(whole(1000)))
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: 1646051,
			Actual: fixedpoint.Number(steep),
			Delta:  tolerance},
		"Arctangent(1000) = %d, want about 1646051",
		steep)

	compare_quadrants(t, tolerance)
}

// Test_Hyperbolic verifies the three hyperbolic functions at zero, at one, and at the
// saturation the domain end gives.
func Test_Hyperbolic(t *testing.T) {
	t.Parallel()
	testify.Equal_Values(t, 0, scalar.Hyperbolic_Sine(0),
		"Hyperbolic_Sine(0) = %d, want 0", scalar.Hyperbolic_Sine(0))
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: whole(1),
			Actual: fixedpoint.Number(scalar.Hyperbolic_Cosine(0)),
			Delta:  4},
		"Hyperbolic_Cosine(0) = %d, want 1",
		scalar.Hyperbolic_Cosine(0))
	testify.Equal_Values(t, 0, scalar.Hyperbolic_Tangent(0),
		"Hyperbolic_Tangent(0) = %d, want 0", scalar.Hyperbolic_Tangent(0))

	// The hyperbolic sine of one is 1.17520119, thus the grid holds about 1232219.
	one := scalar.Hyperbolic_Sine(scalar.Hyperbolic_Angle(whole(1)))
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: 1232219,
			Actual: fixedpoint.Number(one),
			Delta:  64},
		"Hyperbolic_Sine(1) = %d, want about 1232219",
		one)

	// The hyperbolic cosine of one is 1.54308063, thus the grid holds about 1617963.
	cosine := scalar.Hyperbolic_Cosine(scalar.Hyperbolic_Angle(whole(1)))
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: 1617963,
			Actual: fixedpoint.Number(cosine),
			Delta:  128},
		"Hyperbolic_Cosine(1) = %d, want about 1617963",
		cosine)

	// The hyperbolic tangent of one is 0.76159416, thus the grid holds about 798592.
	tangent := scalar.Hyperbolic_Tangent(scalar.Hyperbolic_Angle(whole(1)))
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: 798592,
			Actual: fixedpoint.Number(tangent),
			Delta:  64},
		"Hyperbolic_Tangent(1) = %d, want about 798592",
		tangent)

	// The hyperbolic sine is odd, thus a negative angle mirrors the value.
	mirrored := scalar.Hyperbolic_Sine(scalar.Hyperbolic_Angle(whole(-1)))
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: -1232219,
			Actual: fixedpoint.Number(mirrored),
			Delta:  64},
		"Hyperbolic_Sine(-1) = %d, want about -1232219",
		mirrored)

	// The hyperbolic tangent saturates at one for a large angle.
	saturated := scalar.Hyperbolic_Tangent(
		scalar.Hyperbolic_Angle(scalar.HYPERBOLIC_ANGLE_MAXIMUM))
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: whole(1),
			Actual: fixedpoint.Number(saturated),
			Delta:  4},
		"Hyperbolic_Tangent at the maximum = %d, want 1",
		saturated)

}

// Test_Domain_Errors verifies that each value outside a domain panics instead of returning
// an error value.
func Test_Domain_Errors(t *testing.T) {
	t.Parallel()
	testify.Panics(t, func() {
		scalar.Absolute_Integer(scalar.Signed_Integer(scalar.INTEGER_64_MINIMUM))
	},
		"the most negative integer has no magnitude and must panic")
	testify.Panics(t, func() { scalar.Square_Root(scalar.Radicand(-1)) },
		"a negative radicand must panic")
	testify.Panics(t, func() { scalar.Logarithm_2(scalar.Argument(0)) },
		"a zero logarithm argument must panic")
	testify.Panics(t, func() { scalar.Logarithm(scalar.Argument(-1)) },
		"a negative logarithm argument must panic")
	testify.Panics(t, func() {
		scalar.Exponential_2(scalar.Exponent_2(scalar.EXPONENT_2_MAXIMUM + 1))
	},
		"an exponent above the maximum must panic")
	testify.Panics(t, func() {
		scalar.Exponential(scalar.Exponent(scalar.EXPONENT_MAXIMUM + 1))
	},
		"a natural exponent above the maximum must panic")
	testify.Panics(t, func() { scalar.Arcsine(scalar.Unit_Argument(SCALE + 1)) },
		"an arcsine argument above one must panic")
	testify.Panics(t, func() { scalar.Arcsine(scalar.Unit_Argument(-SCALE - 1)) },
		"an arcsine argument below negative one must panic")
	testify.Panics(t, func() { scalar.Power(scalar.Base(0), 0) },
		"a zero base must panic")
	testify.Panics(t, func() {
		scalar.Hyperbolic_Sine(scalar.Hyperbolic_Angle(scalar.HYPERBOLIC_ANGLE_MAXIMUM + 1))
	},
		"a hyperbolic angle above the maximum must panic")

}

// Lifts a whole number onto the fixed-point grid.
func whole(value int64) (lifted fixedpoint.Number) {
	return fixedpoint.Number(value * SCALE)
}

// SCALE is the count of fixed-point units in one whole.
const SCALE = fixedpoint.SCALE

// Compares the angle Arctangent_Quotient gives in each quadrant, and at the two legs that
// have no quotient.
func compare_quadrants(t *testing.T, tolerance fixedpoint.Number) {
	t.Helper()
	first := scalar.Arctangent_Quotient(scalar.Opposite(whole(1)), scalar.Adjacent(whole(1)))
	testify.In_Delta(t,
		&testify.In_Delta_Input{Expected: fixedpoint.Number(scalar.PI_QUARTER),
			Actual: fixedpoint.Number(first),
			Delta:  tolerance},
		"the first quadrant angle = %d",
		first)

	second := scalar.Arctangent_Quotient(scalar.Opposite(whole(1)), scalar.Adjacent(whole(-1)))
	testify.In_Delta(t,
		&testify.In_Delta_Input{
			Expected: fixedpoint.Number(scalar.PI - scalar.PI_QUARTER),
			Actual:   fixedpoint.Number(second),
			Delta:    tolerance},
		"the second quadrant angle = %d",
		second)

	third := scalar.Arctangent_Quotient(scalar.Opposite(whole(-1)), scalar.Adjacent(whole(-1)))
	testify.In_Delta(t,
		&testify.In_Delta_Input{
			Expected: fixedpoint.Number(scalar.PI_QUARTER - scalar.PI),
			Actual:   fixedpoint.Number(third),
			Delta:    tolerance},
		"the third quadrant angle = %d",
		third)
	testify.Equal_Values(t,
		fixedpoint.Number(scalar.PI_HALF),
		fixedpoint.Number(scalar.Arctangent_Quotient(scalar.Opposite(whole(1)),
			scalar.Adjacent(0))),
		"a zero adjacent leg with a positive opposite gives a quarter turn")
	testify.Equal_Values(t,
		0,
		scalar.Arctangent_Quotient(scalar.Opposite(0), scalar.Adjacent(0)),
		"two zero legs give a zero angle")

}
