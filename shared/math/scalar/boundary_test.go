package scalar

import (
	"testing"

	"local/james-orcales/shared/math/fixedpoint"
	"local/james-orcales/shared/testify"
)

// Sweeps the ends of the fixed-point storage domain and the small values around zero.
func sweep_storage() (values []int64) {
	return []int64{INTEGER_64_MINIMUM, -2, -1, 0, 1, 2, INTEGER_64_MAXIMUM}
}

// Sweeps the ends of the rounding domain and the small values around zero.
func sweep_value() (values []int64) {
	return []int64{VALUE_MINIMUM, -2, -1, 0, 1, 2, VALUE_MAXIMUM}
}

// Sweeps the ends of the closed unit interval and the small values around zero.
func sweep_unit() (values []int64) {
	return []int64{UNIT_MINIMUM, -2, -1, 0, 1, 2, UNIT_MAXIMUM}
}

// Sweeps the values that have a magnitude. The most negative Number negates to itself,
// thus every operation that takes a magnitude first excludes it.
func sweep_magnitude() (values []int64) {
	return []int64{SIGNED_INTEGER_MINIMUM, -2, -1, 0, 1, 2, INTEGER_64_MAXIMUM}
}

// Sweeps the legs whose hypotenuse the storage holds.
func sweep_leg() (values []int64) {
	return []int64{LEG_MINIMUM, -2, -1, 0, 1, 2, LEG_MAXIMUM}
}

// Sweeps the values a logarithm accepts.
func sweep_argument() (values []int64) {
	return []int64{1, 2, SCALE, INTEGER_64_MAXIMUM}
}

// Sweeps the values a square root accepts.
func sweep_radicand() (values []int64) {
	return []int64{0, 1, 2, INTEGER_64_MAXIMUM}
}

// Test_Rounding_Holds_At_The_Domain_Ends drives each rounding function over the ends of
// its domain and over the small values around zero, and requires one result for one input.
func Test_Rounding_Holds_At_The_Domain_Ends(t *testing.T) {
	t.Parallel()
	for _, raw := range sweep_value() {
		value := Value(raw)
		testify.Equal_Values(t, Floor(value), Floor(value),
			"Floor(%d) is not deterministic", raw)
		testify.Equal_Values(t, Ceiling(value), Ceiling(value),
			"Ceiling(%d) is not deterministic", raw)
		testify.Equal_Values(t, Truncate(value), Truncate(value),
			"Truncate(%d) is not deterministic", raw)
		testify.Equal_Values(t, Round(value), Round(value),
			"Round(%d) is not deterministic", raw)
		// Floor never exceeds the value, and Ceiling is never below it.
		testify.False(t, int64(Floor(value)) > raw,
			"Floor(%d) exceeds its own value", raw)
		testify.False(t, int64(Ceiling(value)) < raw,
			"Ceiling(%d) is below its own value", raw)

	}
	dividends := append(sweep_storage(), SIGNED_INTEGER_MINIMUM)
	for _, dividend := range dividends {
		for _, divisor := range sweep_storage() {
			remainder := Modulo(Dividend(dividend), Divisor(divisor))
			testify.Equal_Values(t,
				Modulo(Dividend(dividend), Divisor(divisor)),
				remainder,
				"Modulo(%d,%d) is not deterministic", dividend, divisor)

		}
	}
}

// Test_Roots_Hold_At_The_Domain_Ends drives each root over the ends of its domain.
func Test_Roots_Hold_At_The_Domain_Ends(t *testing.T) {
	t.Parallel()
	for _, raw := range sweep_radicand() {
		testify.Equal_Values(t, Square_Root(Radicand(raw)), Square_Root(Radicand(raw)),
			"Square_Root(%d) is not deterministic", raw)

	}
	for _, raw := range sweep_storage() {
		testify.Equal_Values(t,
			Cube_Root(Cube_Radicand(raw)),
			Cube_Root(Cube_Radicand(raw)),
			"Cube_Root(%d) is not deterministic", raw)

	}
	// The two most negative radicands share a cube root, because they differ by far less
	// than one grid unit of that root.
	testify.Equal_Values(t,
		Cube_Root(Cube_Radicand(SIGNED_INTEGER_MINIMUM)),
		Cube_Root(Cube_Radicand(INTEGER_64_MINIMUM)),

		"the two most negative radicands must share a cube root")

	for _, opposite := range sweep_leg() {
		for _, adjacent := range sweep_leg() {
			distance := Hypotenuse(Leg_Opposite(opposite), Leg_Adjacent(adjacent))
			testify.Equal_Values(t,
				Hypotenuse(Leg_Opposite(opposite), Leg_Adjacent(adjacent)),
				distance,
				"Hypotenuse(%d,%d) varies", opposite, adjacent)
			testify.False(

				// A distance is never negative, whatever the signs of the legs.
				t, int64(distance) < 0,
				"Hypotenuse(%d,%d) is negative", opposite, adjacent)

		}
	}
}

// Test_Exponential_Holds_At_The_Domain_Ends drives each exponential over the ends of its
// domain.
func Test_Exponential_Holds_At_The_Domain_Ends(t *testing.T) {
	t.Parallel()
	binary := []int64{EXPONENT_2_MINIMUM, -2, -1, 0, 1, 2, EXPONENT_2_MAXIMUM}
	for _, raw := range binary {
		testify.Equal_Values(t,
			Exponential_2(Exponent_2(raw)),
			Exponential_2(Exponent_2(raw)),
			"Exponential_2(%d) is not deterministic", raw)

	}
	natural := []int64{EXPONENT_MINIMUM, -2, -1, 0, 1, 2, EXPONENT_MAXIMUM}
	for _, raw := range natural {
		testify.Equal_Values(t, Exponential(Exponent(raw)), Exponential(Exponent(raw)),
			"Exponential(%d) is not deterministic", raw)

	}
}

// Test_Logarithm_Holds_At_The_Domain_Ends drives each logarithm over the ends of its
// domain and requires the base-two result to grow with its argument.
func Test_Logarithm_Holds_At_The_Domain_Ends(t *testing.T) {
	t.Parallel()
	previous := int64(LOGARITHM_2_MINIMUM) - 1
	for _, raw := range sweep_argument() {
		binary := Logarithm_2(Argument(raw))
		testify.Equal_Values(t, Logarithm_2(Argument(raw)), binary,
			"Logarithm_2(%d) is not deterministic", raw)
		testify.False(t, int64(binary) < previous,
			"Logarithm_2(%d) fell below the previous argument", raw)

		previous = int64(binary)
		testify.Equal_Values(t, Logarithm(Argument(raw)), Logarithm(Argument(raw)),
			"Logarithm(%d) is not deterministic", raw)
		testify.Equal_Values(t, Logarithm_10(Argument(raw)), Logarithm_10(Argument(raw)),
			"Logarithm_10(%d) is not deterministic", raw)

	}
}

// Test_Power_Holds_At_The_Domain_Ends drives the power over the ends of each operand. A
// base of one has a zero logarithm, thus every exponent stays in range with it.
func Test_Power_Holds_At_The_Domain_Ends(t *testing.T) {
	t.Parallel()
	for _, raw := range sweep_storage() {
		result := Power(Base(SCALE), Power_Exponent(raw))
		testify.Equal_Values(t, Power(Base(SCALE), Power_Exponent(raw)), result,
			"Power(1,%d) is not deterministic", raw)

	}
	// A tiny exponent keeps the product in range for every base.
	for _, raw := range sweep_argument() {
		result := Power(Base(raw), Power_Exponent(1))
		testify.Equal_Values(t, Power(Base(raw), Power_Exponent(1)), result,
			"Power(%d,1) is not deterministic", raw)

	}
	// A base of two raised to the largest base-two exponent gives the largest power.
	largest := Power(Base(2*SCALE), Power_Exponent(EXPONENT_2_MAXIMUM))
	testify.Equal_Values(t, POWER_MAXIMUM, int64(largest),
		"the largest power is %d, want %d", largest, POWER_MAXIMUM)

}

// Test_Trigonometry_Holds_At_The_Domain_Ends drives each circular function over the ends
// of its domain and requires the sine and the cosine to stay in the unit interval.
func Test_Trigonometry_Holds_At_The_Domain_Ends(t *testing.T) {
	t.Parallel()
	for _, raw := range sweep_value() {
		angle := Angle(raw)
		sine := Sine(angle)
		testify.Equal_Values(t, Sine(angle), sine,
			"Sine(%d) is not deterministic", raw)
		testify.False(t, int64(sine) < UNIT_MINIMUM,
			"Sine(%d) fell below negative one", raw)
		testify.False(t, int64(sine) > UNIT_MAXIMUM,
			"Sine(%d) rose above one", raw)

		cosine := Cosine(angle)
		testify.Equal_Values(t, Cosine(angle), cosine,
			"Cosine(%d) is not deterministic", raw)
		testify.Equal_Values(t, Tangent(angle), Tangent(angle),
			"Tangent(%d) is not deterministic", raw)

	}
}

// Test_Trigonometry_Reaches_Its_Extremes scans the angles around a quarter turn and a half
// turn, where the sine and the cosine reach the ends of the unit interval and the tangent
// grows without bound. A sweep of separated values steps over those points.
func Test_Trigonometry_Reaches_Its_Extremes(t *testing.T) {
	t.Parallel()
	smallest_sine := int64(UNIT_MAXIMUM)
	largest_sine := int64(UNIT_MINIMUM)
	smallest_tangent := int64(0)
	largest_tangent := int64(0)
	for _, center := range []int64{
		int64(PI_HALF), -int64(PI_HALF), int64(PI), -int64(PI), int64(PI) + int64(PI_HALF),
	} {
		for offset := int64(-3000); offset <= 3000; offset++ {
			angle := Angle(center + offset)
			sine := int64(Sine(angle))
			if sine < smallest_sine {
				smallest_sine = sine
			}
			if sine > largest_sine {
				largest_sine = sine
			}
			cosine := int64(Cosine(angle))
			if cosine < smallest_sine {
				smallest_sine = cosine
			}
			if cosine > largest_sine {
				largest_sine = cosine
			}
			tangent := int64(Tangent(angle))
			if tangent < smallest_tangent {
				smallest_tangent = tangent
			}
			if tangent > largest_tangent {
				largest_tangent = tangent
			}
		}
	}
	testify.Equal_Values(t, UNIT_MINIMUM, smallest_sine,
		"the scan reached %d, want the unit minimum %d",
		smallest_sine, UNIT_MINIMUM)
	testify.Equal_Values(t, UNIT_MAXIMUM, largest_sine,
		"the scan reached %d, want the unit maximum %d",
		largest_sine, UNIT_MAXIMUM)
	testify.Equal_Values(t, TANGENT_MINIMUM, smallest_tangent,
		"the scan reached tangents %d..%d, want %d..%d",
		smallest_tangent, largest_tangent, TANGENT_MINIMUM, TANGENT_MAXIMUM)
	testify.Equal_Values(t, TANGENT_MAXIMUM, largest_tangent,
		"the scan reached tangents %d..%d, want %d..%d",
		smallest_tangent, largest_tangent, TANGENT_MINIMUM, TANGENT_MAXIMUM)

}

// Test_Inverse_Trigonometry_Holds_At_The_Domain_Ends drives each inverse function over the
// ends of its domain.
func Test_Inverse_Trigonometry_Holds_At_The_Domain_Ends(t *testing.T) {
	t.Parallel()
	for _, raw := range sweep_unit() {
		testify.Equal_Values(t, Arcsine(Unit_Argument(raw)), Arcsine(Unit_Argument(raw)),
			"Arcsine(%d) is not deterministic", raw)
		testify.Equal_Values(t,
			Arccosine(Unit_Argument(raw)),
			Arccosine(Unit_Argument(raw)),
			"Arccosine(%d) is not deterministic", raw)

	}
	for _, raw := range sweep_storage() {
		testify.Equal_Values(t, Arctangent(Slope(raw)), Arctangent(Slope(raw)),
			"Arctangent(%d) is not deterministic", raw)

	}
	for _, opposite := range sweep_storage() {
		for _, adjacent := range sweep_storage() {
			angle := Arctangent_Quotient(Opposite(opposite), Adjacent(adjacent))
			testify.Equal_Values(t,
				Arctangent_Quotient(Opposite(opposite), Adjacent(adjacent)),
				angle,
				"Arctangent_Quotient(%d,%d) is not deterministic",
				opposite, adjacent)

		}
	}
}

// Test_Hyperbolic_Holds_At_The_Domain_Ends drives each hyperbolic function over the ends
// of its domain.
func Test_Hyperbolic_Holds_At_The_Domain_Ends(t *testing.T) {
	t.Parallel()
	sweep := []int64{
		HYPERBOLIC_ANGLE_MINIMUM, -2, -1, 0, 1, 2, HYPERBOLIC_ANGLE_MAXIMUM,
	}
	for _, raw := range sweep {
		angle := Hyperbolic_Angle(raw)
		testify.Equal_Values(t, Hyperbolic_Sine(angle), Hyperbolic_Sine(angle),
			"Hyperbolic_Sine(%d) is not deterministic", raw)
		testify.Equal_Values(t, Hyperbolic_Cosine(angle), Hyperbolic_Cosine(angle),
			"Hyperbolic_Cosine(%d) is not deterministic", raw)
		testify.Equal_Values(t, Hyperbolic_Tangent(angle), Hyperbolic_Tangent(angle),
			"Hyperbolic_Tangent(%d) is not deterministic", raw)

	}
}

// Test_Integer_Arithmetic_Holds_At_The_Domain_Ends drives each integer operation over the
// ends of its domain.
func Test_Integer_Arithmetic_Holds_At_The_Domain_Ends(t *testing.T) {
	t.Parallel()
	signed := []int64{SIGNED_INTEGER_MINIMUM, -2, -1, 0, 1, 2, INTEGER_64_MAXIMUM}
	for _, raw := range signed {
		testify.Equal_Values(t,
			Absolute_Integer(Signed_Integer(raw)),
			Absolute_Integer(Signed_Integer(raw)),
			"Absolute_Integer(%d) is not deterministic", raw)

	}
	for _, first := range sweep_storage() {
		for _, second := range sweep_storage() {
			smallest := Minimum_Integer(First_Integer(first), Second_Integer(second))
			largest := Maximum_Integer(First_Integer(first), Second_Integer(second))
			testify.False(t, int64(smallest) > int64(largest),
				"min of %d and %d exceeds the max", first, second)

		}
	}
}

// Test_A_Whole_Number_Carries_No_Fraction requires each rounding result to sit on the
// whole-number grid, which the fixedpoint scale defines.
func Test_A_Whole_Number_Carries_No_Fraction(t *testing.T) {
	t.Parallel()
	for _, raw := range sweep_value() {
		value := Value(raw)
		for _, whole := range []Whole_Number{
			Floor(value), Ceiling(value), Truncate(value), Round(value),
		} {
			testify.Equal_Values(t, 0, int64(whole)%fixedpoint.SCALE,
				"a rounding of %d left a fraction", raw)

		}
	}
}

// The tables below are the input vector and the expected results of the Go standard
// library math test suite, quantized onto the fixed-point grid. The standard library
// computed each expected value to twenty-six digits with a high precision calculator,
// thus they are independent of any implementation.

// TABLE_SCALE is the scale the tables below were quantized against. Every entry is a real
// value multiplied by it, thus a different scale makes each entry wrong.
const TABLE_SCALE = 1 << 20

// Test_The_Tables_Match_The_Grid requires the fixed-point scale to be the one the tables
// were quantized against. The entries are frozen numbers, so a change of the fraction
// width would leave them silently wrong rather than merely stale.
func Test_The_Tables_Match_The_Grid(t *testing.T) {
	t.Parallel()
	testify.Equal_Values(t, TABLE_SCALE, SCALE,
		"the grid scale is %d, but the tables hold values scaled by %d",
		SCALE, TABLE_SCALE)

}

// The standard library input vector, on the grid.
func vector_input() (values []int64) {
	return []int64{
		5220872,
		8114796,
		-290330,
		-5253999,
		10104386,
		3068529,
		5483091,
		2860452,
		1913974,
		-9107852,
	}
}

// The magnitudes of the input vector.
func vector_absolute() (values []int64) {
	return []int64{
		5220872,
		8114796,
		290330,
		5253999,
		10104386,
		3068529,
		5483091,
		2860452,
		1913974,
		9107852,
	}
}

// The input vector divided by ten.
func vector_tenth() (values []int64) {
	return []int64{
		522087,
		811480,
		-29033,
		-525400,
		1010439,
		306853,
		548309,
		286045,
		191397,
		-910785,
	}
}

// The values the standard library test pins for sqrt.
func expected_square_root() (values []int64) {
	return []int64{
		2339761,
		2917016,
		551754,
		2347172,
		3255029,
		1793763,
		2397799,
		1731878,
		1416668,
		3090352,
	}
}

// The values the standard library test pins for cbrt.
func expected_cbrt() (values []int64) {
	return []int64{
		1790527,
		2074081,
		-683435,
		-1794306,
		2231361,
		1499835,
		1820016,
		1465137,
		1281480,
		-2155453,
	}
}

// The values the standard library test pins for exp.
func expected_exponential() (values []int64) {
	return []int64{
		152390298,
		2407407034,
		794973,
		6991,
		16054215795,
		19566331,
		195687008,
		16044610,
		6506106,
		177,
	}
}

// The values the standard library test pins for log.
func expected_log() (values []int64) {
	return []int64{
		1683207,
		2145655,
		-1346551,
		1689839,
		2375587,
		1125924,
		1734592,
		1052295,
		630979,
		2266711,
	}
}

// The values the standard library test pins for log10.
func expected_log10() (values []int64) {
	return []int64{
		731008,
		931846,
		-584800,
		733888,
		1031704,
		488983,
		753324,
		457006,
		274031,
		984420,
	}
}

// The values the standard library test pins for log2.
func expected_log2() (values []int64) {
	return []int64{
		2428355,
		3095526,
		-1942662,
		2437923,
		3427248,
		1624366,
		2502488,
		1518141,
		910311,
		3270172,
	}
}

// The values the standard library test pins for pow.
func expected_pow() (values []int64) {
	return []int64{
		99910662364,
		57474127603171,
		554268,
		10,
		4538304382208358,
		885069047,
		177698329342,
		560453807,
		70130673,
		0,
	}
}

// The values the standard library test pins for sin.
func expected_sine() (values []int64) {
	return []int64{
		-1011526,
		1041637,
		-286634,
		1002295,
		-220140,
		223932,
		-911692,
		421482,
		1014798,
		-706154,
	}
}

// The values the standard library test pins for cos.
func expected_cosine() (values []int64) {
	return []int64{
		276274,
		120434,
		1008639,
		308086,
		-1025207,
		-1024386,
		518006,
		-960138,
		-264003,
		-775151,
	}
}

// The values the standard library test pins for tan.
func expected_tangent() (values []int64) {
	return []int64{
		-3839169,
		9069136,
		-297984,
		3411322,
		225158,
		-229220,
		-1845497,
		-460305,
		-4030606,
		955241,
	}
}

// The values the standard library test pins for asin.
func expected_arcsine() (values []int64) {
	return []int64{
		546494,
		927944,
		-29037,
		-550317,
		1363428,
		311410,
		576989,
		289717,
		192476,
		-1103474,
	}
}

// The values the standard library test pins for acos.
func expected_arccosine() (values []int64) {
	return []int64{
		1100606,
		719155,
		1676136,
		2197417,
		283671,
		1335689,
		1070111,
		1357382,
		1454623,
		2750573,
	}
}

// The values the standard library test pins for atan.
func expected_arctangent() (values []int64) {
	return []int64{
		1439265,
		1512351,
		-283234,
		-1440542,
		1538672,
		1301825,
		1448964,
		1278666,
		1121560,
		-1526907,
	}
}

// The values the standard library test pins for atan2.
func expected_arctangent_quotient() (values []int64) {
	return []int64{
		1162692,
		956492,
		1676125,
		2134158,
		842969,
		1348581,
		1141889,
		1367849,
		1457786,
		2397030,
	}
}

// The values the standard library test pins for sinh.
func expected_sinh() (values []int64) {
	return []int64{
		76191541,
		1203703289,
		-294054,
		-78637214,
		8027107863,
		9755069,
		97840695,
		7988041,
		3168555,
		-3103266709,
	}
}

// The values the standard library test pins for cosh.
func expected_cosh() (values []int64) {
	return []int64{
		76198756,
		1203703745,
		1089027,
		78644204,
		8027107932,
		9811263,
		97846313,
		8056569,
		3337552,
		3103266886,
	}
}

// The values the standard library test pins for tanh.
func expected_tanh() (values []int64) {
	return []int64{
		1048477,
		1048576,
		-283131,
		-1048483,
		1048576,
		1042570,
		1048516,
		1039657,
		995481,
		-1048576,
	}
}

// The values the standard library test pins for floor.
func expected_floor() (values []int64) {
	return []int64{
		4194304,
		7340032,
		-1048576,
		-6291456,
		9437184,
		2097152,
		5242880,
		2097152,
		1048576,
		-9437184,
	}
}

// The values the standard library test pins for ceil.
func expected_ceil() (values []int64) {
	return []int64{
		5242880,
		8388608,
		0,
		-5242880,
		10485760,
		3145728,
		6291456,
		3145728,
		2097152,
		-8388608,
	}
}

// The values the standard library test pins for trunc.
func expected_trunc() (values []int64) {
	return []int64{
		4194304,
		7340032,
		0,
		-5242880,
		9437184,
		2097152,
		5242880,
		2097152,
		1048576,
		-8388608,
	}
}

// PARTS_PER_MILLION is the divisor of a relative tolerance.
const PARTS_PER_MILLION = 1000000

// Gives the tolerance for one expected value: the wider of a fixed number of grid units
// and a share of the expected magnitude. The grid bounds the small results and the
// relative error bounds the large ones, thus neither alone covers a whole vector.
func tolerance(want int64, units int64, share int64) (delta int64) {
	magnitude := want
	if magnitude < 0 {
		magnitude = -magnitude
	}
	relative := magnitude / PARTS_PER_MILLION * share
	if relative > units {
		return relative
	}
	return units
}

// Drives one function over the standard library input vector and compares each result
// against the value that suite pins.
func compare_vector(
	t *testing.T, name string, inputs []int64, expected []int64,
	units int64, share int64, apply func(input int64) (result int64),
) {
	t.Helper()
	testify.Count(t, inputs, len(expected),
		"%s: the input and the expected vector differ in size", name)
	for index := range inputs {
		testify.In_Delta(t, &testify.In_Delta_Input{
			Expected: fixedpoint.Number(expected[index]),
			Actual:   fixedpoint.Number(apply(inputs[index])),
			Delta:    fixedpoint.Number(tolerance(expected[index], units, share)),
		}, "%s(%d)", name, inputs[index])
	}
}

// Test_Standard_Roots compares the square root and the cube root against the standard
// library vectors.
func Test_Standard_Roots(t *testing.T) {
	t.Parallel()
	compare_vector(t, "Square_Root", vector_absolute(), expected_square_root(), 8, 0,
		func(input int64) (result int64) {
			return int64(Square_Root(Radicand(input)))
		})
	compare_vector(t, "Cube_Root", vector_input(), expected_cbrt(), 16, 0,
		func(input int64) (result int64) {
			return int64(Cube_Root(Cube_Radicand(input)))
		})
}

// Test_Standard_Exponential compares the natural exponential and the power against the
// standard library vectors.
func Test_Standard_Exponential(t *testing.T) {
	t.Parallel()
	compare_vector(t, "Exponential", vector_input(), expected_exponential(), 8, 200,
		func(input int64) (result int64) {
			return int64(Exponential(Exponent(input)))
		})
	compare_vector(t, "Power", vector_input(), expected_pow(), 8, 2000,
		func(input int64) (result int64) {
			return int64(Power(Base(10*SCALE), Power_Exponent(input)))
		})
}

// Test_Standard_Logarithm compares the three logarithms against the standard library
// vectors.
func Test_Standard_Logarithm(t *testing.T) {
	t.Parallel()
	compare_vector(t, "Logarithm", vector_absolute(), expected_log(), 32, 0,
		func(input int64) (result int64) {
			return int64(Logarithm(Argument(input)))
		})
	compare_vector(t, "Logarithm_10", vector_absolute(), expected_log10(), 32, 0,
		func(input int64) (result int64) {
			return int64(Logarithm_10(Argument(input)))
		})
	compare_vector(t, "Logarithm_2", vector_absolute(), expected_log2(), 32, 0,
		func(input int64) (result int64) {
			return int64(Logarithm_2(Argument(input)))
		})
}

// Test_Standard_Trigonometry compares the sine, the cosine, and the tangent against the
// standard library vectors. Bhaskara's approximation carries an error near two thousandths,
// thus the tolerance is far wider than the grid.
func Test_Standard_Trigonometry(t *testing.T) {
	t.Parallel()
	compare_vector(t, "Sine", vector_input(), expected_sine(), 3000, 0,
		func(input int64) (result int64) {
			return int64(Sine(Angle(input)))
		})
	compare_vector(t, "Cosine", vector_input(), expected_cosine(), 3000, 0,
		func(input int64) (result int64) {
			return int64(Cosine(Angle(input)))
		})
	compare_vector(t, "Tangent", vector_input(), expected_tangent(), 6000, 40000,
		func(input int64) (result int64) {
			return int64(Tangent(Angle(input)))
		})
}

// Test_Standard_Inverse_Trigonometry compares the inverse functions against the standard
// library vectors.
func Test_Standard_Inverse_Trigonometry(t *testing.T) {
	t.Parallel()
	compare_vector(t, "Arcsine", vector_tenth(), expected_arcsine(), 2000, 0,
		func(input int64) (result int64) {
			return int64(Arcsine(Unit_Argument(input)))
		})
	compare_vector(t, "Arccosine", vector_tenth(), expected_arccosine(), 2000, 0,
		func(input int64) (result int64) {
			return int64(Arccosine(Unit_Argument(input)))
		})
	compare_vector(t, "Arctangent", vector_input(), expected_arctangent(), 2000, 0,
		func(input int64) (result int64) {
			return int64(Arctangent(Slope(input)))
		})
	compare_vector(t, "Arctangent_Quotient", vector_input(),
		expected_arctangent_quotient(), 2000, 0,
		func(input int64) (result int64) {
			return int64(Arctangent_Quotient(
				Opposite(10*SCALE), Adjacent(input)))
		})
}

// Test_Standard_Hyperbolic compares the three hyperbolic functions against the standard
// library vectors.
func Test_Standard_Hyperbolic(t *testing.T) {
	t.Parallel()
	compare_vector(t, "Hyperbolic_Sine", vector_input(), expected_sinh(), 16, 500,
		func(input int64) (result int64) {
			return int64(Hyperbolic_Sine(Hyperbolic_Angle(input)))
		})
	compare_vector(t, "Hyperbolic_Cosine", vector_input(), expected_cosh(), 16, 500,
		func(input int64) (result int64) {
			return int64(Hyperbolic_Cosine(Hyperbolic_Angle(input)))
		})
	compare_vector(t, "Hyperbolic_Tangent", vector_input(), expected_tanh(), 2000, 0,
		func(input int64) (result int64) {
			return int64(Hyperbolic_Tangent(Hyperbolic_Angle(input)))
		})
}

// Test_Standard_Whole_Numbers compares the three whole-number selections against the
// standard library vectors. Each selection is exact, thus the tolerance is zero.
func Test_Standard_Whole_Numbers(t *testing.T) {
	t.Parallel()
	compare_vector(t, "Floor", vector_input(), expected_floor(), 0, 0,
		func(input int64) (result int64) {
			return int64(Floor(Value(input)))
		})
	compare_vector(t, "Ceiling", vector_input(), expected_ceil(), 0, 0,
		func(input int64) (result int64) {
			return int64(Ceiling(Value(input)))
		})
	compare_vector(t, "Truncate", vector_input(), expected_trunc(), 0, 0,
		func(input int64) (result int64) {
			return int64(Truncate(Value(input)))
		})
}

// IDENTITY_TOLERANCE bounds a relation that carries the sine error twice, once through
// each operand it squares.
const IDENTITY_TOLERANCE = 6000

// Test_The_Circular_Identity_Holds requires the squared sine and cosine to sum to one over
// the whole angle sweep. No range states this, because each value alone is a legal sine.
func Test_The_Circular_Identity_Holds(t *testing.T) {
	t.Parallel()
	for _, raw := range sweep_value() {
		angle := Angle(raw)
		sine := fixedpoint.Number(Sine(angle))
		cosine := fixedpoint.Number(Cosine(angle))
		total := fixedpoint.Multiply(
			fixedpoint.Multiplicand(sine), fixedpoint.Multiplier(sine)) +
			fixedpoint.Multiply(
				fixedpoint.Multiplicand(cosine), fixedpoint.Multiplier(cosine))
		testify.In_Delta(t, &testify.In_Delta_Input{
			Expected: fixedpoint.Number(SCALE),
			Actual:   total,
			Delta:    IDENTITY_TOLERANCE,
		}, "the squared sine and cosine of %d do not sum to one", raw)
	}
}

// Test_The_Hyperbolic_Identity_Holds requires the squared hyperbolic cosine less the
// squared hyperbolic sine to equal one. The squares grow fast, thus the sweep stays at the
// small angles where the difference still fits the storage.
func Test_The_Hyperbolic_Identity_Holds(t *testing.T) {
	t.Parallel()
	for _, whole_angle := range []int64{-3, -2, -1, 0, 1, 2, 3} {
		angle := Hyperbolic_Angle(whole_angle * SCALE)
		sine := fixedpoint.Number(Hyperbolic_Sine(angle))
		cosine := fixedpoint.Number(Hyperbolic_Cosine(angle))
		difference := fixedpoint.Multiply(
			fixedpoint.Multiplicand(cosine), fixedpoint.Multiplier(cosine)) -
			fixedpoint.Multiply(
				fixedpoint.Multiplicand(sine), fixedpoint.Multiplier(sine))
		testify.In_Delta(t, &testify.In_Delta_Input{
			Expected: fixedpoint.Number(SCALE),
			Actual:   difference,
			Delta:    IDENTITY_TOLERANCE,
		}, "the hyperbolic identity fails at %d", whole_angle)
	}
}

// Test_The_Exponential_Undoes_The_Logarithm requires raising two to the base-two logarithm
// of a value to return that value. The two operations carry independent errors, thus the
// round trip states something neither range does.
func Test_The_Exponential_Undoes_The_Logarithm(t *testing.T) {
	t.Parallel()
	for _, whole_value := range []int64{1, 2, 3, 5, 10, 100, 1000, 100000} {
		value := whole_value * SCALE
		binary := Logarithm_2(Argument(value))
		returned := Exponential_2(Exponent_2(binary))
		// The logarithm resolves twenty fraction bits, thus the round trip keeps about
		// six digits and the tolerance is a share of the value rather than a fixed step.
		testify.In_Delta(t, &testify.In_Delta_Input{
			Expected: fixedpoint.Number(value),
			Actual:   fixedpoint.Number(returned),
			Delta:    fixedpoint.Number(tolerance(value, 8, 2000)),
		}, "raising two to the logarithm of %d does not return it", whole_value)
	}
}

// Test_The_Sine_Undoes_The_Arcsine requires the sine of an arcsine to return the argument.
// The sine carries Bhaskara's error, thus the tolerance is that error and not the grid.
func Test_The_Sine_Undoes_The_Arcsine(t *testing.T) {
	t.Parallel()
	for _, raw := range []int64{-SCALE, -SCALE / 2, 0, SCALE / 4, SCALE / 2, SCALE} {
		angle := Arcsine(Unit_Argument(raw))
		returned := Sine(Angle(angle))
		testify.In_Delta(t, &testify.In_Delta_Input{
			Expected: fixedpoint.Number(raw),
			Actual:   fixedpoint.Number(returned),
			Delta:    IDENTITY_TOLERANCE,
		}, "the sine of the arcsine of %d does not return it", raw)
	}
}
