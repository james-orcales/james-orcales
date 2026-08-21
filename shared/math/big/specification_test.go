package big_test

import (
	"testing"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/big"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/random/prng"
	"local/james-orcales/shared/testify"
)

// Test_Representation keeps zero-value behavior tied to Representation specification.
func Test_Representation(t *testing.T) {
	test_representation(t)
}

// Test_Conversion keeps every bounded conversion under one specification leaf.
func Test_Conversion(t *testing.T) {
	test_conversion(t)
}

// Test_Magnitude keeps caller-owned byte output and bit counts under one leaf.
func Test_Magnitude(t *testing.T) {
	test_magnitude(t)
}

// Test_Sign keeps unique zero sign and aliasing under Sign specification.
func Test_Sign(t *testing.T) {
	test_sign(t)
}

// Test_Comparison keeps signed and magnitude ordering under Comparison specification.
func Test_Comparison(t *testing.T) {
	test_comparison(t)
}

// Test_Addition keeps transactional overflow and aliasing under Addition specification.
func Test_Addition(t *testing.T) {
	test_addition(t)
}

// Test_Multiplication keeps wide products and transactional overflow under one leaf.
func Test_Multiplication(t *testing.T) {
	test_multiplication(t)
}

// Test_Division keeps signed quotient, remainder, aliasing, and failure atomicity under one leaf.
func Test_Division(t *testing.T) {
	test_division(t)
}

// Test_Shift keeps bounded counts and signed right-shift rounding under one leaf.
func Test_Shift(t *testing.T) {
	test_shift(t)
}

// Test_Bitwise keeps infinite signed-bit semantics and boundary overflow under one leaf.
func Test_Bitwise(t *testing.T) {
	test_bitwise(t)
}

// Test_Number_Theory keeps exact integer algorithms under one specification leaf.
func Test_Number_Theory(t *testing.T) {
	test_number_theory(t)
}

// Test_Rational keeps normalized fractions and transactional failures under one leaf.
func Test_Rational(t *testing.T) {
	test_rational(t)
}

// Test_Float keeps bounded precision, rounding, and binary64 parity under one leaf.
func Test_Float(t *testing.T) {
	test_float(t)
}

// Test_Text keeps every output base and destination failure under one leaf.
func Test_Text(t *testing.T) {
	test_text(t)
}

// Test_Serialization keeps bounded gob compatibility under one specification leaf.
func Test_Serialization(t *testing.T) {
	test_serialization(t)
}

// Test_Bounds keeps formulas and hostile input rejection under Bounds specification.
func Test_Bounds(t *testing.T) {
	test_bounds(t)
}

// Test_Allocation keeps every current public operation under Allocation specification.
func Test_Allocation(t *testing.T) {
	test_allocation(t)
}

func test_representation(t *testing.T) {
	var zero big.Int
	testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&zero))
	testify.Equal(t, big.Bit_Count(0), big.Int_Bit_Count(&zero))
}

func test_float(t *testing.T) {
	var zero big.Float
	testify.Equal(t, big.Float_Precision(0), big.Float_Precision_Of(&zero))
	testify.Equal(t, big.SIGN_ZERO, big.Float_Sign(&zero))
	testify.False(t, bool(big.Float_Sign_Bit(&zero)))
	testify.False(t, bool(big.Float_Is_Infinite(&zero)))
	testify.True(t, bool(big.Float_Is_Integer(&zero)))
	test_float_sign_comparison(t)
	test_float_binary_32(t)
	test_float_binary_64(t)
	test_float_integer_conversion(t)
	test_float_round_policy(t)
	test_float_arithmetic(t)
}

func test_float_binary_32(t *testing.T) {
	for _, encoding := range []big.Float_32_Bits{
		FLOAT_32_POSITIVE_ZERO_BITS,
		FLOAT_32_NEGATIVE_ZERO_BITS,
		FLOAT_32_HALF_BITS,
		FLOAT_32_THREE_HALVES_BITS,
		FLOAT_32_MAXIMUM_FINITE_BITS,
		FLOAT_32_POSITIVE_INFINITY_BITS,
		FLOAT_32_NEGATIVE_INFINITY_BITS,
		1,
		2,
		big.Float_32_Bits(uint32(1) << big.FLOAT_32_EXPONENT_SHIFT),
		big.Float_32_Bits(uint32(2) << big.FLOAT_32_EXPONENT_SHIFT),
	} {
		var value big.Float
		status := big.Float_Set_Float_32_Bits(&value, encoding)
		testify.Equal_Values(t, big.STATUS_OK, status)
		actual, accuracy := big.Float_Float_32_Bits(&value)
		testify.Equal(t, big.Float_32_Value_Bits(encoding), actual)
		testify.Equal(t, big.ACCURACY_EXACT, accuracy)
	}
	var value big.Float
	value_before := value
	nan := big.Float_32_Bits(big.FLOAT_32_POSITIVE_INFINITY_BITS | INDEX_STEP)
	testify.Equal_Values(t, big.STATUS_INPUT_INVALID,
		big.Float_Set_Float_32_Bits(&value, nan))
	testify.Equal(t, value_before, value)
	big.Float_Set_Uint_64(
		&value, big.Word_64(uint64(1)<<big.FLOAT_32_VALUE_MANTISSA_BIT_COUNT+1),
	)
	_, accuracy := big.Float_Float_32_Bits(&value)
	testify.Equal(t, big.ACCURACY_BELOW, accuracy)
}

func test_float_integer_conversion(t *testing.T) {
	var value big.Float
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Set_Float_64_Bits(&value, FLOAT_64_THREE_HALVES_BITS))
	var integer big.Int
	accuracy, status := big.Float_Int_Into(&integer, &value)
	testify.Equal(t, big.ACCURACY_BELOW, accuracy)
	testify.Equal_Values(t, big.STATUS_OK, status)
	converted, conversion := big.Int_Int_64(&integer)
	testify.Equal(t, big.Int_64(1), converted)
	testify.Equal_Values(t, big.STATUS_OK, conversion)
	machine, machine_accuracy, machine_status := big.Float_Int_64(&value)
	testify.Equal(t, big.Int_64(1), machine)
	testify.Equal(t, big.ACCURACY_BELOW, machine_accuracy)
	testify.Equal_Values(t, big.STATUS_OK, machine_status)
	unsigned, unsigned_accuracy, unsigned_status := big.Float_Uint_64(&value)
	testify.Equal(t, big.Word_64(1), unsigned)
	testify.Equal(t, big.ACCURACY_BELOW, unsigned_accuracy)
	testify.Equal_Values(t, big.STATUS_OK, unsigned_status)
	big.Float_Negate(&value, &value)
	accuracy, status = big.Float_Int_Into(&integer, &value)
	testify.Equal(t, big.ACCURACY_ABOVE, accuracy)
	testify.Equal_Values(t, big.STATUS_OK, status)
	converted, conversion = big.Int_Int_64(&integer)
	testify.Equal(t, big.Int_64(-1), converted)
	testify.Equal_Values(t, big.STATUS_OK, conversion)
	big.Float_Set_Infinity(&value, false)
	before := integer
	_, status = big.Float_Int_Into(&integer, &value)
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW, status)
	testify.Equal(t, before, integer)
	var rational big.Rat
	var rational_workspace big.Rat_Workspace
	testify.Equal_Values(t, big.STATUS_OK,
		big.Rat_Set_Fraction_64(&rational, 3, 2, &rational_workspace))
	var float_rational_workspace big.Float_Rat_Workspace
	value = big.Float{}
	big.Float_Set_Rat(&value, &rational, &float_rational_workspace)
	encoding, rational_accuracy := big.Float_Float_64_Bits(&value)
	testify.Equal(t, big.Float_64_Value_Bits(FLOAT_64_THREE_HALVES_BITS), encoding)
	testify.Equal(t, big.ACCURACY_EXACT, rational_accuracy)
	var converted_rational big.Rat
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Rat_Into(&converted_rational, &value, &float_rational_workspace))
	testify.Equal(t, big.ORDER_SAME,
		big.Rat_Compare(&rational, &converted_rational, &rational_workspace))
	big.Float_Set_Infinity(&value, false)
	converted_before := converted_rational
	testify.Equal_Values(t, big.STATUS_INPUT_INVALID,
		big.Float_Rat_Into(&converted_rational, &value, &float_rational_workspace))
	testify.Equal(t, converted_before, converted_rational)
	for _, expected := range []int64{
		bits.INTEGER_64_MINIMUM, -1, 0, 1, 2, bits.INTEGER_64_MAXIMUM,
	} {
		value = big.Float{}
		big.Float_Set_Int_64(&value, big.Int_64(expected))
		actual, exact, conversion_status := big.Float_Int_64(&value)
		testify.Equal(t, big.Int_64(expected), actual)
		testify.Equal(t, big.ACCURACY_EXACT, exact)
		testify.Equal_Values(t, big.STATUS_OK, conversion_status)
	}
	for _, expected := range []uint64{0, 1, 2, bits.WORD_64_MAXIMUM} {
		value = big.Float{}
		big.Float_Set_Uint_64(&value, big.Word_64(expected))
		actual, exact, conversion_status := big.Float_Uint_64(&value)
		testify.Equal(t, big.Word_64(expected), actual)
		testify.Equal(t, big.ACCURACY_EXACT, exact)
		testify.Equal_Values(t, big.STATUS_OK, conversion_status)
	}
}

func test_float_sign_comparison(t *testing.T) {
	var source big.Float
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Set_Float_64_Bits(&source, FLOAT_64_NEGATIVE_ZERO_BITS))
	var destination big.Float
	big.Float_Absolute(&destination, &source)
	encoding, _ := big.Float_Float_64_Bits(&destination)
	testify.Equal(t, big.Float_64_Value_Bits(FLOAT_64_POSITIVE_ZERO_BITS), encoding)
	big.Float_Negate(&destination, &destination)
	encoding, _ = big.Float_Float_64_Bits(&destination)
	testify.Equal(t, big.Float_64_Value_Bits(FLOAT_64_NEGATIVE_ZERO_BITS), encoding)

	var one big.Float
	var two big.Float
	big.Float_Set_Int_64(&one, 1)
	big.Float_Set_Int_64(&two, 2)
	testify.Equal(t, big.ORDER_BEFORE, big.Float_Compare(&one, &two))
	testify.Equal(t, big.ORDER_AFTER, big.Float_Compare(&two, &one))
	testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&one, &one))
	two.Exponent = one.Exponent
	testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&one, &two))
	testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&two, &one))
	big.Float_Set_Infinity(&one, false)
	big.Float_Set_Infinity(&two, true)
	testify.Equal(t, big.ORDER_AFTER, big.Float_Compare(&one, &two))
}

func test_float_binary_64(t *testing.T) {
	for _, encoding := range []big.Float_64_Bits{
		FLOAT_64_POSITIVE_ZERO_BITS,
		FLOAT_64_NEGATIVE_ZERO_BITS,
		FLOAT_64_HALF_BITS,
		FLOAT_64_THREE_HALVES_BITS,
		FLOAT_64_MAXIMUM_FINITE_BITS,
		FLOAT_64_POSITIVE_INFINITY_BITS,
		FLOAT_64_NEGATIVE_INFINITY_BITS,
		1,
		2,
		big.Float_64_Bits(uint64(1) << big.FLOAT_64_EXPONENT_SHIFT),
		big.Float_64_Bits(uint64(2) << big.FLOAT_64_EXPONENT_SHIFT),
	} {
		var value big.Float
		status := big.Float_Set_Float_64_Bits(&value, encoding)
		testify.Equal_Values(t, big.STATUS_OK, status)
		actual, accuracy := big.Float_Float_64_Bits(&value)
		testify.Equal(t, big.Float_64_Value_Bits(encoding), actual)
		testify.Equal(t, big.ACCURACY_EXACT, accuracy)
	}
	var value big.Float
	value_before := value
	nan := big.Float_64_Bits(big.FLOAT_64_POSITIVE_INFINITY_BITS | 1)
	testify.Equal_Values(t, big.STATUS_INPUT_INVALID,
		big.Float_Set_Float_64_Bits(&value, nan))
	testify.Equal(t, value_before, value)
	big.Float_Set_Uint_64(
		&value, big.Word_64(uint64(1)<<big.FLOAT_64_VALUE_MANTISSA_BIT_COUNT+1),
	)
	_, accuracy := big.Float_Float_64_Bits(&value)
	testify.Equal(t, big.ACCURACY_BELOW, accuracy)
}

func test_float_round_policy(t *testing.T) {
	for _, test := range []struct {
		Mode     big.Rounding_Mode_Unvalidated
		Value    big.Int_64
		Encoding big.Float_64_Value_Bits
		Accuracy big.Accuracy
	}{
		{big.ROUND_TO_NEAREST_EVEN, 15, 0x4030000000000000, big.ACCURACY_ABOVE},
		{big.ROUND_TO_NEAREST_AWAY, 15, 0x4030000000000000, big.ACCURACY_ABOVE},
		{big.ROUND_TO_ZERO, 15, 0x402c000000000000, big.ACCURACY_BELOW},
		{big.ROUND_AWAY_FROM_ZERO, 15, 0x4030000000000000, big.ACCURACY_ABOVE},
		{big.ROUND_TO_NEGATIVE_INFINITY, 15, 0x402c000000000000, big.ACCURACY_BELOW},
		{big.ROUND_TO_POSITIVE_INFINITY, 15, 0x4030000000000000, big.ACCURACY_ABOVE},
		{big.ROUND_TO_NEAREST_EVEN, -15, 0xc030000000000000, big.ACCURACY_BELOW},
		{big.ROUND_TO_ZERO, -15, 0xc02c000000000000, big.ACCURACY_ABOVE},
	} {
		var value big.Float
		testify.Equal_Values(t, big.STATUS_OK, big.Float_Set_Precision(&value, 3))
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Set_Rounding_Mode(&value, test.Mode))
		big.Float_Set_Int_64(&value, test.Value)
		encoding, _ := big.Float_Float_64_Bits(&value)
		testify.Equal(t, test.Encoding, encoding)
		testify.Equal(t, test.Accuracy, big.Float_Accuracy(&value))
	}
	for _, mode := range []big.Rounding_Mode_Unvalidated{
		big.ROUND_TO_NEAREST_EVEN, big.ROUND_TO_NEAREST_AWAY, big.ROUND_TO_ZERO,
		big.ROUND_AWAY_FROM_ZERO, big.ROUND_TO_NEGATIVE_INFINITY,
		big.ROUND_TO_POSITIVE_INFINITY,
	} {
		var value big.Float
		testify.Equal_Values(t, big.STATUS_OK, big.Float_Set_Rounding_Mode(&value, mode))
	}
}

func test_float_arithmetic(t *testing.T) {
	var workspace big.Float_Addition_Workspace
	var multiplication_workspace big.Float_Multiplication_Workspace
	var division_workspace big.Float_Division_Workspace
	var square_root_workspace big.Float_Square_Root_Workspace
	var one big.Float
	var half big.Float
	big.Float_Set_Int_64(&one, 1)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Set_Float_64_Bits(&half, FLOAT_64_HALF_BITS))
	var result big.Float
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Add(&result, &one, &half, &workspace))
	encoding, accuracy := big.Float_Float_64_Bits(&result)
	testify.Equal(t, big.Float_64_Value_Bits(FLOAT_64_THREE_HALVES_BITS), encoding)
	testify.Equal(t, big.ACCURACY_EXACT, accuracy)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Subtract(&result, &one, &half, &workspace))
	encoding, accuracy = big.Float_Float_64_Bits(&result)
	testify.Equal(t, big.Float_64_Value_Bits(FLOAT_64_HALF_BITS), encoding)
	testify.Equal(t, big.ACCURACY_EXACT, accuracy)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Subtract(&result, &one, &one, &workspace))
	testify.Equal(t, big.SIGN_ZERO, big.Float_Sign(&result))
	big.Float_Set_Infinity(&one, false)
	big.Float_Set_Infinity(&half, true)
	result_before := result
	testify.Equal_Values(t, big.STATUS_INPUT_INVALID,
		big.Float_Add(&result, &one, &half, &workspace))
	testify.Equal(t, result_before, result)
	big.Float_Set_Int_64(&one, 3)
	big.Float_Set_Int_64(&half, 3)
	testify.Equal_Values(t, big.STATUS_OK, big.Float_Set_Precision(&result, 3))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Multiply(&result, &one, &half, &multiplication_workspace))
	encoding, _ = big.Float_Float_64_Bits(&result)
	testify.Equal(t, big.Float_64_Value_Bits(0x4020000000000000), encoding)
	testify.Equal(t, big.ACCURACY_BELOW, big.Float_Accuracy(&result))
	big.Float_Set_Int_64(&one, 1)
	big.Float_Set_Int_64(&half, 3)
	testify.Equal_Values(t, big.STATUS_OK, big.Float_Set_Precision(&result, 3))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Quotient(&result, &one, &half, &division_workspace))
	encoding, _ = big.Float_Float_64_Bits(&result)
	testify.Equal(t, big.Float_64_Value_Bits(0x3fd4000000000000), encoding)
	testify.Equal(t, big.ACCURACY_BELOW, big.Float_Accuracy(&result))
	big.Float_Set_Int_64(&one, 4)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Square_Root(&result, &one, &square_root_workspace))
	encoding, _ = big.Float_Float_64_Bits(&result)
	testify.Equal(t, big.Float_64_Value_Bits(FLOAT_64_TWO_BITS), encoding)
	big.Float_Set_Int_64(&one, 2)
	testify.Equal_Values(t, big.STATUS_OK, big.Float_Set_Precision(&result, 3))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Square_Root(&result, &one, &square_root_workspace))
	encoding, _ = big.Float_Float_64_Bits(&result)
	testify.Equal(t, big.Float_64_Value_Bits(FLOAT_64_THREE_HALVES_BITS), encoding)
	testify.Equal(t, big.ACCURACY_ABOVE, big.Float_Accuracy(&result))
	big.Float_Set_Int_64(&one, 4)
	testify.Equal_Values(t, big.STATUS_OK, big.Float_Set_Precision(&one, 3))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Square_Root(&one, &one, &square_root_workspace))
	encoding, _ = big.Float_Float_64_Bits(&one)
	testify.Equal(t, big.Float_64_Value_Bits(FLOAT_64_TWO_BITS), encoding)
	big.Float_Set_Int_64(&one, -1)
	result_before = result
	testify.Equal_Values(t, big.STATUS_INPUT_INVALID,
		big.Float_Square_Root(&result, &one, &square_root_workspace))
	testify.Equal(t, result_before, result)
}

func test_conversion(t *testing.T) {
	for _, value := range []int64{
		bits.INTEGER_64_MINIMUM, -1, 0, 1, 2, bits.INTEGER_64_MAXIMUM,
	} {
		var integer big.Int
		big.Int_Set_Int_64(&integer, big.Int_64(value))
		testify.True(t, bool(big.Int_Is_Int_64(&integer)))
		converted, status := big.Int_Int_64(&integer)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, value, int64(converted))
	}

	var unsigned big.Int
	big.Int_Set_Uint_64(&unsigned, big.Word_64(bits.WORD_64_MAXIMUM))
	testify.False(t, bool(big.Int_Is_Int_64(&unsigned)))
	_, status := big.Int_Int_64(&unsigned)
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW, status)
	converted, status := big.Int_Uint_64(&unsigned)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, uint64(bits.WORD_64_MAXIMUM), uint64(converted))
	testify.True(t, bool(big.Int_Is_Uint_64(&unsigned)))
	for _, one := range []struct {
		Bytes big.Bytes_Unvalidated
		Bits  big.Bit_Count
	}{
		{Bytes: nil, Bits: 0},
		{Bytes: big.Bytes_Unvalidated{1}, Bits: 1},
		{Bytes: big.Bytes_Unvalidated{2}, Bits: 2},
		{Bytes: big.Bytes_Unvalidated{0, 1}, Bits: 1},
	} {
		var integer big.Int
		parse_status := big.Int_Set_Bytes(&integer, one.Bytes)
		testify.Equal_Values(t, big.STATUS_OK, parse_status)
		testify.Equal(t, one.Bits, big.Int_Bit_Count(&integer))
	}
	test_words(t)
	test_int_float_64_bits(t)
}

func test_int_float_64_bits(t *testing.T) {
	for _, test := range []struct {
		Encoding big.Float_64_Bits
		Value    int64
	}{
		{FLOAT_64_POSITIVE_ZERO_BITS, 0},
		{FLOAT_64_NEGATIVE_ZERO_BITS, 0},
		{FLOAT_64_HALF_BITS, 0},
		{FLOAT_64_NEGATIVE_HALF_BITS, 0},
		{FLOAT_64_THREE_HALVES_BITS, 1},
		{FLOAT_64_NEGATIVE_THREE_HALVES_BITS, -1},
		{FLOAT_64_THREE_BITS, 3},
		{1, 0},
		{2, 0},
	} {
		var value big.Int
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Set_Float_64_Bits(&value, test.Encoding))
		test_bitwise_result(t, &value, test.Value)
	}
	test_int_set_float_64_boundaries(t)
	test_int_float_64_output(t)
}

func test_int_set_float_64_boundaries(t *testing.T) {
	var value big.Int
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Set_Float_64_Bits(&value, FLOAT_64_MAXIMUM_FINITE_BITS))
	testify.Equal(t, big.Bit_Count(big.FLOAT_64_VALUE_BIT_COUNT_MAXIMUM),
		big.Int_Bit_Count(&value))
	big.Int_Set_Int_64(&value, 7)
	for _, encoding := range []big.Float_64_Bits{
		FLOAT_64_POSITIVE_INFINITY_BITS, FLOAT_64_NEGATIVE_INFINITY_BITS,
		FLOAT_64_NOT_A_NUMBER_BITS, big.Float_64_Bits(bits.WORD_64_MAXIMUM),
	} {
		unchanged := value
		testify.Equal_Values(t, big.STATUS_INPUT_INVALID,
			big.Int_Set_Float_64_Bits(&value, encoding))
		testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&value, &unchanged))
	}
}

func test_int_float_64_output(t *testing.T) {
	for _, test := range []struct {
		Value    int64
		Encoding big.Int_Float_64_Value_Bits
	}{
		{0, big.Int_Float_64_Value_Bits(FLOAT_64_POSITIVE_ZERO_BITS)},
		{1, big.Int_Float_64_Value_Bits(FLOAT_64_ONE_BITS)},
		{-1, big.Int_Float_64_Value_Bits(
			uint64(FLOAT_64_ONE_BITS) | uint64(FLOAT_64_SIGN_BITS),
		)},
		{2, big.Int_Float_64_Value_Bits(FLOAT_64_TWO_BITS)},
		{3, big.Int_Float_64_Value_Bits(FLOAT_64_THREE_BITS)},
		{4, big.Int_Float_64_Value_Bits(FLOAT_64_FOUR_BITS)},
	} {
		var value big.Int
		big.Int_Set_Int_64(&value, big.Int_64(test.Value))
		encoding, accuracy := big.Int_Float_64_Bits(&value)
		testify.Equal(t, test.Encoding, encoding)
		testify.Equal(t, big.ACCURACY_EXACT, accuracy)
	}
	test_int_float_64_output_fields(t)
	test_int_float_64_output_round(t)
}

func test_int_float_64_output_fields(t *testing.T) {
	for _, mantissa := range []uint64{1, 2, big.FLOAT_64_MANTISSA_MASK} {
		integer := uint64(1)<<big.FLOAT_64_MANTISSA_BIT_COUNT | mantissa
		var value big.Int
		big.Int_Set_Uint_64(&value, big.Word_64(integer))
		encoding, accuracy := big.Int_Float_64_Bits(&value)
		exponent := big.FLOAT_64_EXPONENT_BIAS + big.FLOAT_64_MANTISSA_BIT_COUNT
		want := uint64(exponent)<<big.FLOAT_64_EXPONENT_SHIFT | mantissa
		testify.Equal(t, big.Int_Float_64_Value_Bits(want), encoding)
		testify.Equal(t, big.ACCURACY_EXACT, accuracy)
	}
}

func test_int_float_64_output_round(t *testing.T) {
	base := uint64(1) << big.FLOAT_64_VALUE_MANTISSA_BIT_COUNT
	for _, test := range []struct {
		Value    uint64
		Mantissa uint64
		Accuracy big.Accuracy
	}{
		{base + 1, 0, big.ACCURACY_BELOW},
		{base + 3, 2, big.ACCURACY_ABOVE},
	} {
		var value big.Int
		big.Int_Set_Uint_64(&value, big.Word_64(test.Value))
		encoding, accuracy := big.Int_Float_64_Bits(&value)
		exponent := big.FLOAT_64_EXPONENT_BIAS + big.FLOAT_64_VALUE_MANTISSA_BIT_COUNT
		want := uint64(exponent)<<big.FLOAT_64_EXPONENT_SHIFT | test.Mantissa
		testify.Equal(t, big.Int_Float_64_Value_Bits(want), encoding)
		testify.Equal(t, test.Accuracy, accuracy)
	}
	maximum := int_domain_values(t)[POSITIVE_MAXIMUM_WORD_INDEX]
	encoding, accuracy := big.Int_Float_64_Bits(&maximum)
	testify.Equal(t, big.Int_Float_64_Value_Bits(FLOAT_64_POSITIVE_INFINITY_BITS), encoding)
	testify.Equal(t, big.ACCURACY_ABOVE, accuracy)
	big.Int_Negate(&maximum, &maximum)
	encoding, accuracy = big.Int_Float_64_Bits(&maximum)
	testify.Equal(t, big.Int_Float_64_Value_Bits(FLOAT_64_NEGATIVE_INFINITY_BITS), encoding)
	testify.Equal(t, big.ACCURACY_BELOW, accuracy)
}

func test_words(t *testing.T) {
	var value big.Int
	var storage [big.WORD_COUNT_MAXIMUM]big.Word
	count, status := big.Int_Words_Into(storage[:], &value)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, big.Word_Count(0), count)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Set_Words(&value, big.Words_Unvalidated{1, 2}))
	count, status = big.Int_Words_Into(storage[:], &value)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, big.Word_Count(2), count)
	testify.Equal(t, []big.Word{1, 2}, []big.Word(storage[:count]))
	short := [...]big.Word{9}
	unchanged_short := short
	count, status = big.Int_Words_Into(short[:], &value)
	testify.Equal_Values(t, big.STATUS_DESTINATION_TOO_SMALL, status)
	testify.Equal(t, big.Word_Count(0), count)
	testify.Equal(t, unchanged_short, short)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Set_Words(&value, big.Words_Unvalidated{1, 0}))
	testify.Equal(t, big.Bit_Count(1), big.Int_Bit_Count(&value))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Set_Words(&value, big.Words_Unvalidated{0, 0}))
	testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&value))
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Words(&value, nil))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Set_Words(&value, big.Words_Unvalidated{1}))
	for _, size := range []int{0, 1, 2} {
		count, status = big.Int_Words_Into(storage[:size], &value)
		if size == 0 {
			testify.Equal_Values(t, big.STATUS_DESTINATION_TOO_SMALL, status)
			testify.Equal(t, big.Word_Count(0), count)
		} else {
			testify.Equal_Values(t, big.STATUS_OK, status)
			testify.Equal(t, big.Word_Count(1), count)
		}
	}

	var maximum_words [big.WORD_COUNT_MAXIMUM]big.Word
	for index := range maximum_words {
		maximum_words[index] = big.Word(bits.WORD_64_MAXIMUM)
	}
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Set_Words(&value, maximum_words[:]))
	big.Int_Negate(&value, &value)
	count, status = big.Int_Words_Into(storage[:], &value)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, big.Word_Count(big.WORD_COUNT_MAXIMUM), count)
	testify.Equal(t, maximum_words, storage)
	unchanged := value
	var oversized [big.WORD_COUNT_MAXIMUM + INDEX_STEP]big.Word
	testify.Equal_Values(t, big.STATUS_INPUT_INVALID,
		big.Int_Set_Words(&value, oversized[:]))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&value, &unchanged))
}

func test_magnitude(t *testing.T) {
	var integer big.Int
	var storage [bytes.SLICE_SIZE_MAXIMUM]byte
	destination, validation_status := big.Bytes_Validate(storage[:])
	testify.Equal_Values(t, big.STATUS_OK, validation_status)
	count, status := big.Int_Bytes_Into(destination, &integer)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, big.Byte_Count(0), count)

	source := big.Bytes_Unvalidated{1, 2}
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Bytes(&integer, source))
	count, status = big.Int_Bytes_Into(destination, &integer)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, big.Byte_Count(len(source)), count)
	testify.Equal(t, []byte(source), []byte(destination[:count]))

	short, validation_status := big.Bytes_Validate(storage[:len(source)-1])
	testify.Equal_Values(t, big.STATUS_OK, validation_status)
	short_before := [...]byte{storage[0]}
	count, status = big.Int_Bytes_Into(short, &integer)
	testify.Equal_Values(t, big.STATUS_DESTINATION_TOO_SMALL, status)
	testify.Equal(t, big.Byte_Count(0), count)
	testify.Equal(t, short_before[:], []byte(short))
	status = big.Int_Fill_Bytes(short, &integer)
	testify.Equal_Values(t, big.STATUS_DESTINATION_TOO_SMALL, status)
	testify.Equal(t, short_before[:], []byte(short))

	for index := range storage {
		storage[index] = byte(bits.WORD_8_MAXIMUM)
	}
	filled, validation_status := big.Bytes_Validate(storage[:len(source)+2])
	testify.Equal_Values(t, big.STATUS_OK, validation_status)
	status = big.Int_Fill_Bytes(filled, &integer)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, []byte{0, 0, 1, 2}, []byte(filled))

	big.Int_Negate(&integer, &integer)
	count, status = big.Int_Bytes_Into(destination, &integer)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, []byte(source), []byte(destination[:count]))

	for _, one := range []struct {
		Value int64
		Count big.Trailing_Zero_Bit_Count
	}{
		{Value: 0, Count: 0},
		{Value: 1, Count: 0},
		{Value: 2, Count: 1},
		{Value: 4, Count: 2},
		{Value: 8, Count: 3},
	} {
		big.Int_Set_Int_64(&integer, big.Int_64(one.Value))
		testify.Equal(t, one.Count, big.Int_Trailing_Zero_Bit_Count(&integer))
	}
	var wide_bytes [big.WORD_BYTE_COUNT + 1]byte
	wide_bytes[0] = 1
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Bytes(&integer, wide_bytes[:]))
	testify.Equal(t, big.Trailing_Zero_Bit_Count(big.WORD_BIT_COUNT),
		big.Int_Trailing_Zero_Bit_Count(&integer))
	var maximum_trailing_bytes [bytes.SLICE_SIZE_MAXIMUM]byte
	maximum_trailing_bytes[0] = byte(
		bits.CARRY_MAXIMUM << (bits.BIT_COUNT_8_MAXIMUM - 1),
	)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Set_Bytes(&integer, maximum_trailing_bytes[:]))
	testify.Equal(t,
		big.Trailing_Zero_Bit_Count(big.TRAILING_ZERO_BIT_COUNT_MAXIMUM),
		big.Int_Trailing_Zero_Bit_Count(&integer))
}

func test_sign(t *testing.T) {
	var source big.Int
	var result big.Int
	big.Int_Set_Int_64(&source, -42)
	big.Int_Absolute(&result, &source)
	converted, status := big.Int_Int_64(&result)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, int64(42), int64(converted))
	big.Int_Negate(&result, &result)
	converted, status = big.Int_Int_64(&result)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, int64(-42), int64(converted))

	var zero big.Int
	big.Int_Negate(&result, &zero)
	testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&result))
	big.Int_Set_Int_64(&result, 1)
	testify.Equal(t, big.SIGN_POSITIVE, big.Int_Sign(&result))
}

func test_comparison(t *testing.T) {
	var negative big.Int
	var positive big.Int
	big.Int_Set_Int_64(&negative, -7)
	big.Int_Set_Int_64(&positive, 5)
	testify.Equal(t, big.ORDER_BEFORE, big.Int_Compare(&negative, &positive))
	testify.Equal(t, big.ORDER_AFTER, big.Int_Compare(&positive, &negative))
	testify.Equal(t, big.ORDER_AFTER, big.Int_Compare_Absolute(&negative, &positive))
	testify.Equal(t, big.ORDER_BEFORE, big.Int_Compare_Absolute(&positive, &negative))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare_Absolute(&negative, &negative))
}

func test_addition(t *testing.T) {
	for _, one := range []struct {
		Left       int64
		Right      int64
		Sum        int64
		Difference int64
	}{
		{Left: 2, Right: 3, Sum: 5, Difference: -1},
		{Left: -2, Right: -3, Sum: -5, Difference: 1},
		{Left: 7, Right: -2, Sum: 5, Difference: 9},
		{Left: -7, Right: 2, Sum: -5, Difference: -9},
		{Left: 7, Right: -7, Sum: 0, Difference: 14},
	} {
		var left big.Int
		var right big.Int
		var result big.Int
		big.Int_Set_Int_64(&left, big.Int_64(one.Left))
		big.Int_Set_Int_64(&right, big.Int_64(one.Right))
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Add(&result, &left, &right))
		converted, status := big.Int_Int_64(&result)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, one.Sum, int64(converted))
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Subtract(&result, &left, &right))
		converted, status = big.Int_Int_64(&result)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, one.Difference, int64(converted))
	}

	var left big.Int
	var right big.Int
	big.Int_Set_Uint_64(&left, big.Word_64(bits.WORD_64_MAXIMUM))
	big.Int_Set_Uint_64(&right, 1)
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Add(&left, &left, &right))
	testify.Equal(t, big.Bit_Count(bits.BIT_COUNT_64_MAXIMUM+1),
		big.Int_Bit_Count(&left))
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Subtract(&left, &left, &right))
	converted, status := big.Int_Uint_64(&left)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, uint64(bits.WORD_64_MAXIMUM), uint64(converted))
}

func test_multiplication(t *testing.T) {
	var workspace big.Int_Multiplication_Workspace
	for _, one := range []struct {
		Left    int64
		Right   int64
		Product int64
	}{
		{Left: 0, Right: 7, Product: 0},
		{Left: 6, Right: 7, Product: 42},
		{Left: -6, Right: 7, Product: -42},
		{Left: 6, Right: -7, Product: -42},
		{Left: -6, Right: -7, Product: 42},
	} {
		var left big.Int
		var right big.Int
		var product big.Int
		big.Int_Set_Int_64(&left, big.Int_64(one.Left))
		big.Int_Set_Int_64(&right, big.Int_64(one.Right))
		status := big.Int_Multiply(&product, &left, &right, &workspace)
		testify.Equal_Values(t, big.STATUS_OK, status)
		converted, conversion_status := big.Int_Int_64(&product)
		testify.Equal_Values(t, big.STATUS_OK, conversion_status)
		testify.Equal(t, one.Product, int64(converted))
	}

	var factor_bytes [big.WORD_BYTE_COUNT + 1]byte
	factor_bytes[0] = 1
	factor_bytes[len(factor_bytes)-1] = 1
	var factor big.Int
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Bytes(&factor, factor_bytes[:]))
	var expected_bytes [2*big.WORD_BYTE_COUNT + 1]byte
	expected_bytes[0] = 1
	expected_bytes[big.WORD_BYTE_COUNT] = 2
	expected_bytes[len(expected_bytes)-1] = 1
	var expected big.Int
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Bytes(&expected, expected_bytes[:]))
	var product big.Int
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Multiply(&product, &factor, &factor, &workspace))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&product, &expected))

	var left big.Int
	var right big.Int
	big.Int_Set_Int_64(&left, 6)
	big.Int_Set_Int_64(&right, 7)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Multiply(&left, &left, &right, &workspace))
	converted, conversion_status := big.Int_Int_64(&left)
	testify.Equal_Values(t, big.STATUS_OK, conversion_status)
	testify.Equal(t, int64(42), int64(converted))
	big.Int_Set_Int_64(&left, 6)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Multiply(&right, &left, &right, &workspace))
	converted, conversion_status = big.Int_Int_64(&right)
	testify.Equal_Values(t, big.STATUS_OK, conversion_status)
	testify.Equal(t, int64(42), int64(converted))

	var maximum_bytes [bytes.SLICE_SIZE_MAXIMUM]byte
	for index := range maximum_bytes {
		maximum_bytes[index] = byte(bits.WORD_8_MAXIMUM)
	}
	var maximum big.Int
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Bytes(&maximum, maximum_bytes[:]))
	unchanged := maximum
	big.Int_Set_Uint_64(&right, 2)
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW,
		big.Int_Multiply(&maximum, &maximum, &right, &workspace))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&maximum, &unchanged))
}

func test_division(t *testing.T) {
	var workspace big.Int_Division_Workspace
	test_division_machine(t, &workspace)
	test_division_single(t, &workspace)
	test_euclidean_division(t, &workspace)
	test_division_wide(t, &workspace)
	test_division_failure(t, &workspace)
}

func test_division_machine(t *testing.T, workspace *big.Int_Division_Workspace) {
	for _, one := range []struct {
		Dividend  int64
		Divisor   int64
		Quotient  int64
		Remainder int64
	}{
		{Dividend: 7, Divisor: 3, Quotient: 2, Remainder: 1},
		{Dividend: -7, Divisor: 3, Quotient: -2, Remainder: -1},
		{Dividend: 7, Divisor: -3, Quotient: -2, Remainder: 1},
		{Dividend: -7, Divisor: -3, Quotient: 2, Remainder: -1},
		{Dividend: 2, Divisor: 3, Quotient: 0, Remainder: 2},
		{Dividend: 6, Divisor: 3, Quotient: 2, Remainder: 0},
	} {
		var dividend big.Int
		var divisor big.Int
		var quotient big.Int
		var remainder big.Int
		big.Int_Set_Int_64(&dividend, big.Int_64(one.Dividend))
		big.Int_Set_Int_64(&divisor, big.Int_64(one.Divisor))
		status := big.Int_Quotient_Remainder(
			&quotient, &remainder, &dividend, &divisor, workspace,
		)
		testify.Equal_Values(t, big.STATUS_OK, status)
		converted, conversion_status := big.Int_Int_64(&quotient)
		testify.Equal_Values(t, big.STATUS_OK, conversion_status)
		testify.Equal(t, one.Quotient, int64(converted))
		converted, conversion_status = big.Int_Int_64(&remainder)
		testify.Equal_Values(t, big.STATUS_OK, conversion_status)
		testify.Equal(t, one.Remainder, int64(converted))
	}
}

func test_division_single(t *testing.T, workspace *big.Int_Division_Workspace) {
	var dividend big.Int
	var divisor big.Int
	var result big.Int
	big.Int_Set_Int_64(&dividend, -7)
	big.Int_Set_Int_64(&divisor, 3)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Quotient(&result, &dividend, &divisor, workspace))
	converted, conversion_status := big.Int_Int_64(&result)
	testify.Equal_Values(t, big.STATUS_OK, conversion_status)
	testify.Equal(t, int64(-2), int64(converted))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Remainder(&result, &dividend, &divisor, workspace))
	converted, conversion_status = big.Int_Int_64(&result)
	testify.Equal_Values(t, big.STATUS_OK, conversion_status)
	testify.Equal(t, int64(-1), int64(converted))
	var zero big.Int
	unchanged := result
	testify.Equal_Values(t, big.STATUS_DIVISOR_ZERO,
		big.Int_Quotient(&result, &dividend, &zero, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&result, &unchanged))
	testify.Equal_Values(t, big.STATUS_DIVISOR_ZERO,
		big.Int_Remainder(&result, &dividend, &zero, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&result, &unchanged))
}

func test_euclidean_division(t *testing.T, workspace *big.Int_Division_Workspace) {
	for _, one := range []struct {
		Dividend int64
		Divisor  int64
		Quotient int64
		Modulus  int64
	}{
		{Dividend: 7, Divisor: 3, Quotient: 2, Modulus: 1},
		{Dividend: -7, Divisor: 3, Quotient: -3, Modulus: 2},
		{Dividend: -2, Divisor: 3, Quotient: -1, Modulus: 1},
		{Dividend: 7, Divisor: -3, Quotient: -2, Modulus: 1},
		{Dividend: -7, Divisor: -3, Quotient: 3, Modulus: 2},
	} {
		var dividend big.Int
		var divisor big.Int
		var quotient big.Int
		var modulus big.Int
		big.Int_Set_Int_64(&dividend, big.Int_64(one.Dividend))
		big.Int_Set_Int_64(&divisor, big.Int_64(one.Divisor))
		status := big.Int_Divide_Modulus(
			&quotient, &modulus, &dividend, &divisor, workspace,
		)
		testify.Equal_Values(t, big.STATUS_OK, status)
		converted, conversion_status := big.Int_Int_64(&quotient)
		testify.Equal_Values(t, big.STATUS_OK, conversion_status)
		testify.Equal(t, one.Quotient, int64(converted))
		converted, conversion_status = big.Int_Int_64(&modulus)
		testify.Equal_Values(t, big.STATUS_OK, conversion_status)
		testify.Equal(t, one.Modulus, int64(converted))
	}
}

func test_division_wide(t *testing.T, workspace *big.Int_Division_Workspace) {
	var dividend_bytes [big.WORD_BYTE_COUNT + 1]byte
	dividend_bytes[0] = 1
	dividend_bytes[len(dividend_bytes)-1] = 5
	var divisor_bytes [big.WORD_BYTE_COUNT + 1]byte
	divisor_bytes[0] = 1
	divisor_bytes[len(divisor_bytes)-1] = 1
	var dividend big.Int
	var divisor big.Int
	var quotient big.Int
	var remainder big.Int
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Set_Bytes(&dividend, dividend_bytes[:]))
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Bytes(&divisor, divisor_bytes[:]))
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Quotient_Remainder(
		&quotient, &remainder, &dividend, &divisor, workspace,
	))
	converted, conversion_status := big.Int_Int_64(&quotient)
	testify.Equal_Values(t, big.STATUS_OK, conversion_status)
	testify.Equal(t, int64(1), int64(converted))
	converted, conversion_status = big.Int_Int_64(&remainder)
	testify.Equal_Values(t, big.STATUS_OK, conversion_status)
	testify.Equal(t, int64(4), int64(converted))

	big.Int_Set_Int_64(&dividend, 17)
	big.Int_Set_Int_64(&divisor, 5)
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Quotient_Remainder(
		&dividend, &divisor, &dividend, &divisor, workspace,
	))
	converted, conversion_status = big.Int_Int_64(&dividend)
	testify.Equal_Values(t, big.STATUS_OK, conversion_status)
	testify.Equal(t, int64(3), int64(converted))
	converted, conversion_status = big.Int_Int_64(&divisor)
	testify.Equal_Values(t, big.STATUS_OK, conversion_status)
	testify.Equal(t, int64(2), int64(converted))
}

func test_division_failure(t *testing.T, workspace *big.Int_Division_Workspace) {
	var dividend big.Int
	var divisor big.Int
	var quotient big.Int
	var remainder big.Int
	big.Int_Set_Int_64(&dividend, 17)
	big.Int_Set_Int_64(&divisor, 5)
	big.Int_Set_Int_64(&quotient, 11)
	big.Int_Set_Int_64(&remainder, 13)
	quotient_before := quotient
	remainder_before := remainder
	var zero big.Int
	testify.Equal_Values(t, big.STATUS_DIVISOR_ZERO, big.Int_Quotient_Remainder(
		&quotient, &remainder, &dividend, &zero, workspace,
	))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&quotient, &quotient_before))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&remainder, &remainder_before))
	testify.Equal_Values(t, big.STATUS_DESTINATIONS_OVERLAP,
		big.Int_Quotient_Remainder(
			&quotient, &quotient, &dividend, &divisor, workspace,
		))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&quotient, &quotient_before))
	testify.Equal_Values(t, big.STATUS_DESTINATIONS_OVERLAP,
		big.Int_Divide_Modulus(
			&quotient, &quotient, &dividend, &divisor, workspace,
		))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&quotient, &quotient_before))
}

func test_shift(t *testing.T) {
	zero_count, status := big.Shift_Count_Validate(0)
	testify.Equal_Values(t, big.STATUS_OK, status)
	one_count, status := big.Shift_Count_Validate(1)
	testify.Equal_Values(t, big.STATUS_OK, status)
	two_count, status := big.Shift_Count_Validate(2)
	testify.Equal_Values(t, big.STATUS_OK, status)
	wide_count, status := big.Shift_Count_Validate(big.WORD_BIT_COUNT + 1)
	testify.Equal_Values(t, big.STATUS_OK, status)
	maximum_count, status := big.Shift_Count_Validate(big.BIT_COUNT_MAXIMUM)
	testify.Equal_Values(t, big.STATUS_OK, status)
	_, status = big.Shift_Count_Validate(big.BIT_COUNT_MAXIMUM + 1)
	testify.Equal_Values(t, big.STATUS_INPUT_INVALID, status)
	_, status = big.Shift_Count_Validate(big.Shift_Count_Unvalidated(bits.WORD_MAXIMUM))
	testify.Equal_Values(t, big.STATUS_INPUT_INVALID, status)

	var source big.Int
	var result big.Int
	big.Int_Set_Int_64(&source, 3)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Shift_Left(&result, &source, wide_count))
	var expected_bytes [big.WORD_BYTE_COUNT + 1]byte
	expected_bytes[0] = 6
	var expected big.Int
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Bytes(&expected, expected_bytes[:]))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&result, &expected))
	big.Int_Shift_Right(&result, &result, wide_count)
	converted, conversion_status := big.Int_Int_64(&result)
	testify.Equal_Values(t, big.STATUS_OK, conversion_status)
	testify.Equal(t, int64(3), int64(converted))

	for _, one := range []struct {
		Value int64
		Count big.Shift_Count
		Want  int64
	}{
		{Value: -3, Count: one_count, Want: -2},
		{Value: -4, Count: one_count, Want: -2},
		{Value: -1, Count: maximum_count, Want: -1},
		{Value: 7, Count: two_count, Want: 1},
	} {
		big.Int_Set_Int_64(&source, big.Int_64(one.Value))
		big.Int_Shift_Right(&result, &source, one.Count)
		converted, conversion_status = big.Int_Int_64(&result)
		testify.Equal_Values(t, big.STATUS_OK, conversion_status)
		testify.Equal(t, one.Want, int64(converted))
	}

	big.Int_Set_Int_64(&source, -3)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Shift_Left(&source, &source, one_count))
	converted, conversion_status = big.Int_Int_64(&source)
	testify.Equal_Values(t, big.STATUS_OK, conversion_status)
	testify.Equal(t, int64(-6), int64(converted))
	big.Int_Shift_Right(&source, &source, zero_count)
	converted, conversion_status = big.Int_Int_64(&source)
	testify.Equal_Values(t, big.STATUS_OK, conversion_status)
	testify.Equal(t, int64(-6), int64(converted))
	test_shift_word_carry(t)
	test_shift_overflow(t, one_count)
}

func test_shift_overflow(t *testing.T, count big.Shift_Count) {
	var maximum_bytes [bytes.SLICE_SIZE_MAXIMUM]byte
	for index := range maximum_bytes {
		maximum_bytes[index] = byte(bits.WORD_8_MAXIMUM)
	}
	var maximum big.Int
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Bytes(&maximum, maximum_bytes[:]))
	unchanged := maximum
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW,
		big.Int_Shift_Left(&maximum, &maximum, count))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&maximum, &unchanged))
}

func test_shift_word_carry(t *testing.T) {
	carry_count, status := big.Shift_Count_Validate(big.WORD_BIT_COUNT - INDEX_STEP)
	testify.Equal_Values(t, big.STATUS_OK, status)
	var value big.Int
	big.Int_Set_Uint_64(&value, 3)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Shift_Left(&value, &value, carry_count))
	testify.Equal(t, big.Bit_Count(big.WORD_BIT_COUNT+INDEX_STEP),
		big.Int_Bit_Count(&value))
}

func test_bitwise(t *testing.T) {
	test_bit_access(t)
	var workspace big.Int_Bitwise_Workspace
	for _, one := range []struct {
		Left  int64
		Right int64
	}{
		{Left: 0, Right: 0},
		{Left: 5, Right: 3},
		{Left: -6, Right: 3},
		{Left: 6, Right: -3},
		{Left: -6, Right: -3},
	} {
		test_bitwise_pair(t, one.Left, one.Right, &workspace)
	}

	var value big.Int
	var result big.Int
	for _, source := range []int64{-7, -1, 0, 1, 7} {
		big.Int_Set_Int_64(&value, big.Int_64(source))
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Not(&result, &value, &workspace))
		converted, status := big.Int_Int_64(&result)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, ^source, int64(converted))
	}

	big.Int_Set_Int_64(&value, -6)
	big.Int_Set_Int_64(&result, 3)
	big.Int_And(&value, &value, &result, &workspace)
	converted, status := big.Int_Int_64(&value)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, int64(-6)&int64(3), int64(converted))

	var maximum_bytes [bytes.SLICE_SIZE_MAXIMUM]byte
	for index := range maximum_bytes {
		maximum_bytes[index] = byte(bits.WORD_8_MAXIMUM)
	}
	var maximum big.Int
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Bytes(&maximum, maximum_bytes[:]))
	unchanged := maximum
	var negative_one big.Int
	big.Int_Set_Int_64(&negative_one, -1)
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW,
		big.Int_Xor(&maximum, &maximum, &negative_one, &workspace))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&maximum, &unchanged))
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW,
		big.Int_And_Not(&maximum, &negative_one, &maximum, &workspace))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&maximum, &unchanged))
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW,
		big.Int_Not(&maximum, &maximum, &workspace))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&maximum, &unchanged))
}

func test_bit_access(t *testing.T) {
	for _, one := range []struct {
		Input  big.Bit_Index_Unvalidated
		Want   big.Bit_Index
		Status big.Validation_Status
	}{
		{Input: -1, Want: 0, Status: big.STATUS_INPUT_INVALID},
		{Input: 0, Want: 0, Status: big.STATUS_OK},
		{Input: 1, Want: 1, Status: big.STATUS_OK},
		{Input: 2, Want: 2, Status: big.STATUS_OK},
		{Input: big.BIT_INDEX_MAXIMUM, Want: big.BIT_INDEX_MAXIMUM, Status: big.STATUS_OK},
		{Input: big.BIT_COUNT_MAXIMUM, Want: 0, Status: big.STATUS_INPUT_INVALID},
	} {
		index, status := big.Bit_Index_Validate(one.Input)
		testify.Equal_Values(t, one.Status, status)
		testify.Equal(t, one.Want, index)
	}

	zero_index, status := big.Bit_Index_Validate(0)
	testify.Equal_Values(t, big.STATUS_OK, status)
	one_index, status := big.Bit_Index_Validate(1)
	testify.Equal_Values(t, big.STATUS_OK, status)
	two_index, status := big.Bit_Index_Validate(2)
	testify.Equal_Values(t, big.STATUS_OK, status)
	var value big.Int
	big.Int_Set_Int_64(&value, 5)
	testify.Equal(t, big.BIT_SET, big.Int_Bit(&value, zero_index))
	testify.Equal(t, big.BIT_CLEAR, big.Int_Bit(&value, one_index))
	testify.Equal(t, big.BIT_SET, big.Int_Bit(&value, two_index))
	big.Int_Set_Int_64(&value, -5)
	testify.Equal(t, big.BIT_SET, big.Int_Bit(&value, zero_index))
	testify.Equal(t, big.BIT_SET, big.Int_Bit(&value, one_index))
	testify.Equal(t, big.BIT_CLEAR, big.Int_Bit(&value, two_index))

	var workspace big.Int_Bitwise_Workspace
	var result big.Int
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Set_Bit(&result, &value, two_index, big.BIT_SET, &workspace))
	test_bitwise_result(t, &result, -1)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Set_Bit(&value, &value, zero_index, big.BIT_CLEAR, &workspace))
	test_bitwise_result(t, &value, -6)

	maximum_index, status := big.Bit_Index_Validate(big.BIT_INDEX_MAXIMUM)
	testify.Equal_Values(t, big.STATUS_OK, status)
	var zero big.Int
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Set_Bit(&result, &zero, maximum_index, big.BIT_SET, &workspace))
	testify.Equal(t, big.Bit_Count(big.BIT_COUNT_MAXIMUM), big.Int_Bit_Count(&result))
	testify.Equal(t, big.BIT_SET, big.Int_Bit(&result, maximum_index))
	test_bit_set_boundary(t, zero_index, &workspace)
}

func test_bit_set_boundary(
	t *testing.T, zero_index big.Bit_Index, workspace *big.Int_Bitwise_Workspace,
) {
	var maximum_bytes [bytes.SLICE_SIZE_MAXIMUM]byte
	for index := range maximum_bytes {
		maximum_bytes[index] = byte(bits.WORD_8_MAXIMUM)
	}
	var maximum big.Int
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Bytes(&maximum, maximum_bytes[:]))
	big.Int_Negate(&maximum, &maximum)
	unchanged := maximum
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW,
		big.Int_Set_Bit(&maximum, &maximum, zero_index, big.BIT_CLEAR, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&maximum, &unchanged))
}

func test_bitwise_pair(
	t *testing.T, left_value int64, right_value int64, workspace *big.Int_Bitwise_Workspace,
) {
	var left big.Int
	var right big.Int
	var result big.Int
	big.Int_Set_Int_64(&left, big.Int_64(left_value))
	big.Int_Set_Int_64(&right, big.Int_64(right_value))
	big.Int_And(&result, &left, &right, workspace)
	test_bitwise_result(t, &result, left_value&right_value)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_And_Not(&result, &left, &right, workspace))
	test_bitwise_result(t, &result, left_value&^right_value)
	big.Int_Or(&result, &left, &right, workspace)
	test_bitwise_result(t, &result, left_value|right_value)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Xor(&result, &left, &right, workspace))
	test_bitwise_result(t, &result, left_value^right_value)
}

func test_bitwise_result(t *testing.T, value *big.Int, expected int64) {
	converted, status := big.Int_Int_64(value)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, expected, int64(converted))
}

func test_number_theory(t *testing.T) {
	test_binomial(t)
	test_exponent(t)
	test_modular_arithmetic(t)
	test_multiply_range(t)
	test_square_root(t)
	test_random(t)
	test_jacobi(t)
	test_primality(t)
	var workspace big.Int_Greatest_Common_Divisor_Workspace
	for _, one := range []struct {
		Left  int64
		Right int64
		Want  int64
	}{
		{Left: 0, Right: 0, Want: 0},
		{Left: 48, Right: 18, Want: 6},
		{Left: -48, Right: 18, Want: 6},
		{Left: 48, Right: -18, Want: 6},
		{Left: -48, Right: -18, Want: 6},
	} {
		var left big.Int
		var right big.Int
		var result big.Int
		big.Int_Set_Int_64(&left, big.Int_64(one.Left))
		big.Int_Set_Int_64(&right, big.Int_64(one.Right))
		big.Int_Greatest_Common_Divisor(&result, &left, &right, &workspace)
		test_bitwise_result(t, &result, one.Want)
	}
	var left big.Int
	var right big.Int
	big.Int_Set_Int_64(&left, 21)
	big.Int_Set_Int_64(&right, 14)
	big.Int_Greatest_Common_Divisor(&left, &left, &right, &workspace)
	test_bitwise_result(t, &left, 7)

	values := int_domain_values(t)
	for index := range values {
		left = values[index]
		big.Int_Greatest_Common_Divisor(
			&left, &values[index], &values[ZERO_INDEX], &workspace,
		)
		left = values[index]
		big.Int_Greatest_Common_Divisor(
			&left, &values[ZERO_INDEX], &values[index], &workspace,
		)
	}
	exercise_greatest_common_workspace_domains(
		&left, &values[ZERO_INDEX], &workspace,
	)
}

func test_jacobi(t *testing.T) {
	for _, test := range []struct {
		Numerator   int64
		Denominator int64
		Symbol      big.Jacobi_Symbol
	}{
		{0, 1, 1}, {0, -1, 1}, {1, 1, 1}, {1, -1, 1},
		{0, 5, 0}, {1, 5, 1}, {2, 5, -1}, {-2, 5, -1},
		{2, -5, -1}, {-2, -5, 1}, {3, 5, -1}, {5, 5, 0},
		{-5, 5, 0}, {6, 5, 1}, {6, -5, 1}, {-6, 5, 1}, {-6, -5, -1},
	} {
		var numerator big.Int
		var denominator big.Int
		big.Int_Set_Int_64(&numerator, big.Int_64(test.Numerator))
		big.Int_Set_Int_64(&denominator, big.Int_64(test.Denominator))
		var workspace big.Int_Jacobi_Workspace
		symbol, status := big.Int_Jacobi(&numerator, &denominator, &workspace)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, test.Symbol, symbol)
	}
	for _, denominator_value := range []big.Int_64{0, 2, -2} {
		var numerator big.Int
		var denominator big.Int
		big.Int_Set_Int_64(&numerator, 1)
		big.Int_Set_Int_64(&denominator, denominator_value)
		var workspace big.Int_Jacobi_Workspace
		symbol, status := big.Int_Jacobi(&numerator, &denominator, &workspace)
		testify.Equal(t, big.Jacobi_Symbol(0), symbol)
		testify.Equal_Values(t, big.STATUS_INPUT_INVALID, status)
	}
	var numerator big.Int
	var denominator big.Int
	big.Int_Set_Int_64(&numerator, 2)
	big.Int_Set_Int_64(&denominator, 5)
	var workspace big.Int_Jacobi_Workspace
	for _, count := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		workspace.Division.Quotient_Count = big.Quotient_Count(count)
		workspace.Division.Remainder_Count = big.Remainder_Count(count)
		symbol, status := big.Int_Jacobi(&numerator, &denominator, &workspace)
		testify.Equal(t, big.Jacobi_Symbol(-1), symbol)
		testify.Equal_Values(t, big.STATUS_OK, status)
	}
	values := int_domain_values(t)
	for _, large_numerator := range []struct {
		Index  int
		Symbol big.Jacobi_Symbol
	}{
		{Index: POSITIVE_TWO_WORD_INDEX, Symbol: 1},
		{Index: POSITIVE_MAXIMUM_WORD_INDEX, Symbol: 0},
	} {
		symbol, status := big.Int_Jacobi(
			&values[large_numerator.Index], &denominator, &workspace,
		)
		testify.Equal(t, large_numerator.Symbol, symbol)
		testify.Equal_Values(t, big.STATUS_OK, status)
	}
	var odd_two_word big.Int
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Add(
		&odd_two_word, &values[POSITIVE_TWO_WORD_INDEX],
		&values[POSITIVE_ONE_WORD_INDEX],
	))
	for _, large_denominator := range []*big.Int{
		&odd_two_word, &values[POSITIVE_MAXIMUM_WORD_INDEX],
	} {
		symbol, status := big.Int_Jacobi(&numerator, large_denominator, &workspace)
		testify.Equal(t, big.Jacobi_Symbol(1), symbol)
		testify.Equal_Values(t, big.STATUS_OK, status)
	}
}

func test_primality(t *testing.T) {
	for _, test := range []struct {
		Value int64
		Prime big.Boolean
	}{
		{-7, false}, {0, false}, {1, false}, {2, true}, {3, true}, {4, false},
		{5, true}, {61, true}, {63, false}, {67, true}, {73, true}, {79, true},
		{83, true}, {89, true}, {97, true}, {101, true}, {103, true}, {107, true},
		{109, true}, {127, true}, {989, false}, {2047, false}, {4757, false},
		{42799, false},
	} {
		var value big.Int
		big.Int_Set_Int_64(&value, big.Int_64(test.Value))
		var workspace big.Int_Primality_Workspace
		prime, consumed, status := big.Int_Probably_Prime(
			&value, test_primality_options(0), nil, &workspace,
		)
		testify.Equal(t, test.Prime, prime)
		testify.Equal(t, big.Random_Word_Count(0), consumed)
		testify.Equal_Values(t, big.STATUS_OK, status)
	}
	test_primality_entropy(t)
	test_primality_limits(t)
	test_primality_domains(t)
	test_primality_standard_vectors(t)
}

func test_primality_standard_vectors(t *testing.T) {
	for _, text := range []string{
		"13756265695458089029",
		"13496181268022124907",
		"10953742525620032441",
		"17908251027575790097",
		"18699199384836356663",
	} {
		test_primality_text(t, text, true)
	}
	for _, text := range []string{
		"82793403787388584738507275144194252681",
		"1195068768795265792518361315725116351898245581",
		"989", "3239", "5777", "10877", "27971", "29681", "30739",
		"31631", "39059", "72389", "73919", "75077", "100127", "113573",
		"125249", "137549", "137801", "153931", "155819", "161027", "162133",
		"189419", "218321", "231703", "249331", "370229", "429479", "430127",
		"459191", "473891", "480689", "600059", "621781", "632249", "635627",
		"2047", "3277", "4033", "4681", "8321", "15841", "29341", "42799",
		"49141", "52633", "65281", "74665", "80581", "85489", "88357", "90751",
		"3673744903", "3281593591", "2385076987", "2738053141", "2009621503",
		"1502682721", "255866131", "117987841", "587861", "6368689", "8725753",
		"80579735209", "105919633",
	} {
		test_primality_text(t, text, false)
	}
}

func test_primality_text(t *testing.T, text string, expected big.Boolean) {
	var value big.Int
	var parse_workspace big.Int_Parse_Workspace
	base, parse_status := big.Int_Parse(&value, []byte(text), 10, &parse_workspace)
	testify.Equal(t, big.Base(10), base)
	testify.Equal_Values(t, big.STATUS_OK, parse_status)
	var primality_workspace big.Int_Primality_Workspace
	prime, consumed, status := big.Int_Probably_Prime(
		&value, test_primality_options(0), nil, &primality_workspace,
	)
	testify.Equal(t, expected, prime)
	testify.Equal(t, big.Random_Word_Count(0), consumed)
	testify.Equal_Values(t, big.STATUS_OK, status)
}

func test_primality_entropy(t *testing.T) {
	var prime big.Int
	big.Int_Set_Uint_64(&prime, 71)
	var workspace big.Int_Primality_Workspace
	for _, test := range []struct {
		Source      []big.Word
		Repetitions big.Primality_Repetition_Count_Unvalidated
		Consumed    big.Random_Word_Count
		Status      int
	}{
		{Source: []big.Word{0}, Repetitions: 1, Consumed: 1, Status: big.STATUS_OK},
		{Source: []big.Word{0, 1}, Repetitions: 2, Consumed: 2, Status: big.STATUS_OK},
		{Source: nil, Repetitions: 1, Consumed: 0, Status: big.STATUS_SOURCE_EXHAUSTED},
		{Source: []big.Word{big.Word(bits.WORD_64_MAXIMUM)}, Repetitions: 1,
			Consumed: 1, Status: big.STATUS_SOURCE_EXHAUSTED},
	} {
		result, used, result_status := big.Int_Probably_Prime(
			&prime, test_primality_options(test.Repetitions), test.Source, &workspace,
		)
		testify.Equal(t, big.Boolean(test.Status == big.STATUS_OK), result)
		testify.Equal(t, test.Consumed, used)
		testify.Equal_Values(t, test.Status, result_status)
	}
	var maximum_source [big.RANDOM_WORD_SIZE_MAXIMUM]big.Word
	result, used, result_status := big.Int_Probably_Prime(
		&prime,
		test_primality_options(big.PRIMALITY_REPETITION_COUNT_MAXIMUM),
		maximum_source[:], &workspace,
	)
	testify.True(t, bool(result))
	testify.Equal(t, big.Random_Word_Count(big.RANDOM_WORD_SIZE_MAXIMUM), used)
	testify.Equal_Values(t, big.STATUS_OK, result_status)
}

func test_primality_limits(t *testing.T) {
	var prime big.Int
	big.Int_Set_Uint_64(&prime, 71)
	var workspace big.Int_Primality_Workspace
	for _, repetitions := range []big.Primality_Repetition_Count_Unvalidated{
		big.PRIMALITY_REPETITION_COUNT_UNVALIDATED_MINIMUM,
		big.PRIMALITY_REPETITION_COUNT_UNVALIDATED_MAXIMUM,
	} {
		result, used, result_status := big.Int_Probably_Prime(
			&prime, test_primality_options(repetitions), nil, &workspace,
		)
		testify.False(t, bool(result))
		testify.Equal(t, big.Random_Word_Count(0), used)
		testify.Equal_Values(t, big.STATUS_INPUT_INVALID, result_status)
	}
	var oversized [big.RANDOM_WORD_SIZE_UNVALIDATED_MAXIMUM]big.Word
	result, used, result_status := big.Int_Probably_Prime(
		&prime, test_primality_options(0), oversized[:], &workspace,
	)
	testify.False(t, bool(result))
	testify.Equal(t, big.Random_Word_Count(0), used)
	testify.Equal_Values(t, big.STATUS_INPUT_INVALID, result_status)
	for _, parameter_count := range []big.Primality_Parameter_Count_Unvalidated{
		big.PRIMALITY_PARAMETER_COUNT_UNVALIDATED_MINIMUM,
		big.PRIMALITY_PARAMETER_COUNT_UNVALIDATED_MAXIMUM,
	} {
		options := test_primality_options(0)
		options.Parameter_Count = parameter_count
		result, used, result_status = big.Int_Probably_Prime(
			&prime, options, nil, &workspace,
		)
		testify.False(t, bool(result))
		testify.Equal(t, big.Random_Word_Count(0), used)
		testify.Equal_Values(t, big.STATUS_INPUT_INVALID, result_status)
	}
	options := test_primality_options(0)
	options.Parameter_Count = big.PRIMALITY_PARAMETER_COUNT_MINIMUM
	result, used, result_status = big.Int_Probably_Prime(
		&prime, options, nil, &workspace,
	)
	testify.False(t, bool(result))
	testify.Equal(t, big.Random_Word_Count(0), used)
	testify.Equal_Values(t, big.STATUS_SEARCH_EXHAUSTED, result_status)
	options.Parameter_Count = big.PRIMALITY_PARAMETER_COUNT_MINIMUM + INDEX_STEP
	result, used, result_status = big.Int_Probably_Prime(
		&prime, options, nil, &workspace,
	)
	testify.False(t, bool(result))
	testify.Equal(t, big.Random_Word_Count(0), used)
	testify.Equal_Values(t, big.STATUS_SEARCH_EXHAUSTED, result_status)
}

func test_primality_domains(t *testing.T) {
	var workspace big.Int_Primality_Workspace
	values := int_domain_values(t)
	for _, index := range []int{
		POSITIVE_TWO_WORD_INDEX, NEGATIVE_TWO_WORD_INDEX,
		POSITIVE_MAXIMUM_WORD_INDEX, NEGATIVE_MAXIMUM_WORD_INDEX,
	} {
		_, _, status := big.Int_Probably_Prime(
			&values[index], test_primality_options(0), nil, &workspace,
		)
		testify.Equal_Values(t, big.STATUS_OK, status)
	}
	for _, count := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		for _, prime_value := range []big.Word_64{67, 71, 73, 79, 83, 89, 97, 101} {
			var prime big.Int
			big.Int_Set_Uint_64(&prime, prime_value)
			workspace.Modular.Division.Quotient_Count = big.Quotient_Count(count)
			workspace.Modular.Division.Remainder_Count = big.Remainder_Count(count)
			result, _, status := big.Int_Probably_Prime(
				&prime, test_primality_options(0), nil, &workspace,
			)
			testify.True(t, bool(result))
			testify.Equal_Values(t, big.STATUS_OK, status)
		}
	}
}

func test_primality_options(
	repetitions big.Primality_Repetition_Count_Unvalidated,
) (options big.Primality_Options_Unvalidated) {
	return big.Primality_Options_Unvalidated{
		Repetitions:     repetitions,
		Parameter_Count: big.PRIMALITY_PARAMETER_COUNT_MAXIMUM,
	}
}

func test_random(t *testing.T) {
	const RANDOM_BOUND = 10
	var maximum big.Int
	big.Int_Set_Uint_64(&maximum, RANDOM_BOUND)
	generator := prng.New(7)
	reference := prng.New(7)
	var workspace big.Int_Random_Workspace
	var reference_workspace big.Int_Random_Workspace
	var value big.Int
	var reference_value big.Int
	var source [big.RANDOM_WORD_SIZE_MAXIMUM]big.Word
	var reference_source [big.RANDOM_WORD_SIZE_MAXIMUM]big.Word
	var seen [RANDOM_BOUND]bool
	for index := 0; index < len(seen)*len(seen); index++ {
		test_random_words(&source, &generator)
		test_random_words(&reference_source, &reference)
		consumed, random_status := big.Int_Random(
			&value, &maximum, source[:], &workspace,
		)
		reference_consumed, reference_status := big.Int_Random(
			&reference_value, &maximum, reference_source[:], &reference_workspace,
		)
		testify.Equal_Values(t, big.STATUS_OK, random_status)
		testify.Equal(t, consumed, reference_consumed)
		testify.Equal(t, random_status, reference_status)
		testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&value, &reference_value))
		testify.Equal(t, big.ORDER_BEFORE, big.Int_Compare(&value, &maximum))
		word, status := big.Int_Uint_64(&value)
		testify.Equal_Values(t, big.STATUS_OK, status)
		seen[word] = true
	}
	for _, observed := range seen {
		testify.True(t, observed)
	}
	original := maximum
	test_random_words(&source, &generator)
	_, status := big.Int_Random(&maximum, &maximum, source[:], &workspace)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, big.ORDER_BEFORE, big.Int_Compare(&maximum, &original))
	test_random_statuses(t, &workspace)
	test_random_int_domains(t, &workspace)
}

func test_random_int_domains(t *testing.T, workspace *big.Int_Random_Workspace) {
	values := int_domain_values(t)
	var source [big.RANDOM_WORD_SIZE_MAXIMUM]big.Word
	for _, value_index := range []int{
		POSITIVE_TWO_WORD_INDEX, POSITIVE_MAXIMUM_WORD_INDEX,
	} {
		maximum := values[value_index]
		destination := maximum
		consumed, status := big.Int_Random(
			&destination, &maximum, source[:maximum.Count], workspace,
		)
		testify.Equal(t, big.Random_Word_Count(maximum.Count), consumed)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&destination))
	}
}

func test_random_words(
	destination *[big.RANDOM_WORD_SIZE_MAXIMUM]big.Word, generator *prng.Generator,
) {
	for index := range destination {
		destination[index] = big.Word(prng.Generator_Next(generator))
	}
}

func test_random_statuses(t *testing.T, workspace *big.Int_Random_Workspace) {
	for _, maximum_value := range []big.Int_64{0, -1} {
		var maximum big.Int
		big.Int_Set_Int_64(&maximum, maximum_value)
		var value big.Int
		big.Int_Set_Int_64(&value, 7)
		consumed, status := big.Int_Random(&value, &maximum, nil, workspace)
		testify.Equal(t, big.Random_Word_Count(0), consumed)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&value))
	}
	var maximum big.Int
	big.Int_Set_Uint_64(&maximum, 10)
	var value big.Int
	big.Int_Set_Int_64(&value, 7)
	unchanged := value
	one_rejection := []big.Word{15}
	consumed, status := big.Int_Random(&value, &maximum, one_rejection, workspace)
	testify.Equal(t, big.Random_Word_Count(1), consumed)
	testify.Equal_Values(t, big.STATUS_SOURCE_EXHAUSTED, status)
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&value, &unchanged))
	two_attempts := []big.Word{15, 1}
	consumed, status = big.Int_Random(&value, &maximum, two_attempts, workspace)
	testify.Equal(t, big.Random_Word_Count(2), consumed)
	testify.Equal_Values(t, big.STATUS_OK, status)
	var rejected [big.RANDOM_WORD_SIZE_MAXIMUM]big.Word
	for index := range rejected {
		rejected[index] = 15
	}
	consumed, status = big.Int_Random(&value, &maximum, rejected[:], workspace)
	testify.Equal(t, big.Random_Word_Count(big.RANDOM_WORD_SIZE_MAXIMUM), consumed)
	testify.Equal_Values(t, big.STATUS_SOURCE_EXHAUSTED, status)
	var oversized [big.RANDOM_WORD_SIZE_UNVALIDATED_MAXIMUM]big.Word
	consumed, status = big.Int_Random(&value, &maximum, oversized[:], workspace)
	testify.Equal(t, big.Random_Word_Count(0), consumed)
	testify.Equal_Values(t, big.STATUS_INPUT_INVALID, status)
}

func test_modular_arithmetic(t *testing.T) {
	test_modular_multiply(t)
	test_modular_inverse(t)
	test_modular_exponent(t)
	test_modular_square_root(t)
	test_modular_domains(t)
}

func test_modular_multiply(t *testing.T) {
	var workspace big.Int_Modular_Workspace
	for _, one := range []struct {
		Left    int64
		Right   int64
		Modulus int64
		Want    int64
	}{
		{Left: 3, Right: 4, Modulus: 5, Want: 2},
		{Left: -3, Right: 4, Modulus: 5, Want: 3},
		{Left: 3, Right: -4, Modulus: 5, Want: 3},
		{Left: 3, Right: 4, Modulus: -5, Want: 2},
	} {
		var left big.Int
		var right big.Int
		var modulus big.Int
		var result big.Int
		big.Int_Set_Int_64(&left, big.Int_64(one.Left))
		big.Int_Set_Int_64(&right, big.Int_64(one.Right))
		big.Int_Set_Int_64(&modulus, big.Int_64(one.Modulus))
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Modular_Multiply(
			&result, &left, &right, &modulus, &workspace,
		))
		test_bitwise_result(t, &result, one.Want)
	}
	test_modular_multiply_bounds(t, &workspace)
}

func test_modular_multiply_bounds(
	t *testing.T, workspace *big.Int_Modular_Workspace,
) {
	values := int_domain_values(t)
	maximum := values[POSITIVE_MAXIMUM_WORD_INDEX]
	var one big.Int
	big.Int_Set_Uint_64(&one, 1)
	var modulus big.Int
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Subtract(&modulus, &maximum, &one))
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Modular_Multiply(
		&maximum, &maximum, &maximum, &modulus, workspace,
	))
	test_bitwise_result(t, &maximum, 1)
	var zero big.Int
	unchanged := modulus
	testify.Equal_Values(t, big.STATUS_DIVISOR_ZERO, big.Int_Modular_Multiply(
		&modulus, &maximum, &maximum, &zero, workspace,
	))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&modulus, &unchanged))
}

func test_modular_inverse(t *testing.T) {
	var workspace big.Int_Modular_Workspace
	for _, one := range []struct {
		Value   int64
		Modulus int64
		Want    int64
	}{
		{Value: 3, Modulus: 11, Want: 4},
		{Value: -3, Modulus: 11, Want: 7},
		{Value: 3, Modulus: -11, Want: 4},
		{Value: 7, Modulus: 1, Want: 0},
	} {
		var value big.Int
		var modulus big.Int
		var result big.Int
		big.Int_Set_Int_64(&value, big.Int_64(one.Value))
		big.Int_Set_Int_64(&modulus, big.Int_64(one.Modulus))
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Modular_Inverse(&result, &value, &modulus, &workspace))
		test_bitwise_result(t, &result, one.Want)
	}
	test_modular_inverse_failures(t, &workspace)
}

func test_modular_inverse_failures(
	t *testing.T, workspace *big.Int_Modular_Workspace,
) {
	var value big.Int
	var modulus big.Int
	var result big.Int
	big.Int_Set_Int_64(&value, 2)
	big.Int_Set_Int_64(&modulus, 4)
	big.Int_Set_Int_64(&result, 7)
	unchanged := result
	testify.Equal_Values(t, big.STATUS_RESULT_ABSENT,
		big.Int_Modular_Inverse(&result, &value, &modulus, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&result, &unchanged))
	modulus = big.Int{}
	testify.Equal_Values(t, big.STATUS_DIVISOR_ZERO,
		big.Int_Modular_Inverse(&result, &value, &modulus, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&result, &unchanged))
}

func test_modular_exponent(t *testing.T) {
	var workspace big.Int_Modular_Workspace
	for _, one := range []struct {
		Base     int64
		Exponent int64
		Modulus  int64
		Want     int64
	}{
		{Base: 2, Exponent: 10, Modulus: 1000, Want: 24},
		{Base: -2, Exponent: 3, Modulus: 5, Want: 2},
		{Base: 2, Exponent: 10, Modulus: -1000, Want: 24},
		{Base: 2, Exponent: 0, Modulus: 1, Want: 0},
		{Base: 3, Exponent: -1, Modulus: 11, Want: 4},
	} {
		var base big.Int
		var exponent big.Int
		var modulus big.Int
		var result big.Int
		big.Int_Set_Int_64(&base, big.Int_64(one.Base))
		big.Int_Set_Int_64(&exponent, big.Int_64(one.Exponent))
		big.Int_Set_Int_64(&modulus, big.Int_64(one.Modulus))
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Modular_Exponent(
			&result, &base, &exponent, &modulus, &workspace,
		))
		test_bitwise_result(t, &result, one.Want)
	}
	test_modular_exponent_failures(t, &workspace)
}

func test_modular_exponent_failures(
	t *testing.T, workspace *big.Int_Modular_Workspace,
) {
	var base big.Int
	var exponent big.Int
	var modulus big.Int
	big.Int_Set_Int_64(&base, 2)
	big.Int_Set_Int_64(&exponent, -1)
	big.Int_Set_Int_64(&modulus, 4)
	unchanged := base
	testify.Equal_Values(t, big.STATUS_RESULT_ABSENT,
		big.Int_Modular_Exponent(&base, &base, &exponent, &modulus, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&base, &unchanged))
	modulus = big.Int{}
	testify.Equal_Values(t, big.STATUS_DIVISOR_ZERO,
		big.Int_Modular_Exponent(&base, &base, &exponent, &modulus, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&base, &unchanged))
}

func test_modular_square_root(t *testing.T) {
	for _, test := range []struct {
		Value      int64
		Modulus    int64
		Nonresidue int64
		Root_1     int64
		Root_2     int64
	}{
		{Value: 10, Modulus: 13, Nonresidue: 2, Root_1: 6, Root_2: 7},
		{Value: -3, Modulus: 13, Nonresidue: 2, Root_1: 6, Root_2: 7},
		{Value: 4, Modulus: 17, Nonresidue: 3, Root_1: 2, Root_2: 15},
		{Value: 56, Modulus: 101, Nonresidue: 2, Root_1: 37, Root_2: 64},
	} {
		var value big.Int
		var modulus big.Int
		var nonresidue big.Int
		var result big.Int
		big.Int_Set_Int_64(&value, big.Int_64(test.Value))
		big.Int_Set_Int_64(&modulus, big.Int_64(test.Modulus))
		big.Int_Set_Int_64(&nonresidue, big.Int_64(test.Nonresidue))
		var workspace big.Int_Modular_Square_Root_Workspace
		status := big.Int_Modular_Square_Root(
			&result, &value, &modulus, &nonresidue, &workspace,
		)
		testify.Equal_Values(t, big.STATUS_OK, status)
		root, conversion := big.Int_Int_64(&result)
		testify.Equal_Values(t, big.STATUS_OK, conversion)
		testify.True(t, int64(root) == test.Root_1 || int64(root) == test.Root_2)
	}
	test_modular_square_root_failures(t)
	test_modular_square_root_alias(t)
}

func test_modular_square_root_failures(t *testing.T) {
	var value big.Int
	var modulus big.Int
	var nonresidue big.Int
	var result big.Int
	big.Int_Set_Int_64(&value, 3)
	big.Int_Set_Int_64(&modulus, 7)
	big.Int_Set_Int_64(&nonresidue, 3)
	big.Int_Set_Int_64(&result, 11)
	unchanged := result
	var workspace big.Int_Modular_Square_Root_Workspace
	status := big.Int_Modular_Square_Root(
		&result, &value, &modulus, &nonresidue, &workspace,
	)
	testify.Equal_Values(t, big.STATUS_RESULT_ABSENT, status)
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&result, &unchanged))
	big.Int_Set_Int_64(&value, 2)
	big.Int_Set_Int_64(&modulus, 15)
	big.Int_Set_Int_64(&nonresidue, 7)
	status = big.Int_Modular_Square_Root(
		&result, &value, &modulus, &nonresidue, &workspace,
	)
	testify.Equal_Values(t, big.STATUS_RESULT_ABSENT, status)
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&result, &unchanged))
	big.Int_Set_Int_64(&value, 4)
	big.Int_Set_Int_64(&modulus, 17)
	big.Int_Set_Int_64(&nonresidue, 1)
	status = big.Int_Modular_Square_Root(
		&result, &value, &modulus, &nonresidue, &workspace,
	)
	testify.Equal_Values(t, big.STATUS_INPUT_INVALID, status)
	for _, invalid := range []int64{0, 2, -7} {
		big.Int_Set_Int_64(&modulus, big.Int_64(invalid))
		status = big.Int_Modular_Square_Root(
			&result, &value, &modulus, &nonresidue, &workspace,
		)
		testify.Equal_Values(t, big.STATUS_INPUT_INVALID, status)
	}
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&result, &unchanged))
}

func test_modular_square_root_alias(t *testing.T) {
	var value big.Int
	var modulus big.Int
	var nonresidue big.Int
	big.Int_Set_Int_64(&value, 10)
	big.Int_Set_Int_64(&modulus, 13)
	big.Int_Set_Int_64(&nonresidue, 2)
	var workspace big.Int_Modular_Square_Root_Workspace
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Modular_Square_Root(
		&value, &value, &modulus, &nonresidue, &workspace,
	))
	root, status := big.Int_Int_64(&value)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.True(t, int64(root) == 6 || int64(root) == 7)
	var zero big.Int
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Modular_Square_Root(
		&value, &zero, &modulus, &nonresidue, &workspace,
	))
	test_bitwise_result(t, &value, 0)
}

func test_modular_domains(t *testing.T) {
	values := int_domain_values(t)
	var modulus big.Int
	big.Int_Set_Uint_64(&modulus, 1)
	var workspace big.Int_Modular_Workspace
	for index := range values {
		destination := values[(index+INDEX_STEP)%len(values)]
		right := values[(index+TWO_WORD_COUNT)%len(values)]
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Modular_Multiply(
			&destination, &values[index], &right, &modulus, &workspace,
		))
		destination = values[(index+INDEX_STEP)%len(values)]
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Modular_Inverse(
			&destination, &values[index], &modulus, &workspace,
		))
		destination = values[(index+INDEX_STEP)%len(values)]
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Modular_Exponent(
			&destination, &values[index], &values[index], &modulus, &workspace,
		))
	}
	test_modular_multiply_domains(t, &values, &workspace)
	test_modular_bound_domains(t, &values, &workspace)
	test_modular_square_root_domains(t, &values)
}

func test_modular_square_root_domains(
	t *testing.T, values *[INT_DOMAIN_COUNT]big.Int,
) {
	var modulus big.Int
	var nonresidue big.Int
	var source big.Int
	big.Int_Set_Uint_64(&modulus, 13)
	big.Int_Set_Uint_64(&nonresidue, 2)
	big.Int_Set_Uint_64(&source, 10)
	var workspace big.Int_Modular_Square_Root_Workspace
	for index := range values {
		destination := values[index]
		big.Int_Modular_Square_Root(
			&destination, &source, &modulus, &nonresidue, &workspace,
		)
		destination = values[(index+INDEX_STEP)%len(values)]
		big.Int_Modular_Square_Root(
			&destination, &values[index], &modulus, &nonresidue, &workspace,
		)
		destination = values[(index+INDEX_STEP)%len(values)]
		big.Int_Modular_Square_Root(
			&destination, &source, &modulus, &values[index], &workspace,
		)
		destination = values[(index+INDEX_STEP)%len(values)]
		big.Int_Modular_Square_Root(
			&destination, &source, &values[index], &nonresidue, &workspace,
		)
	}
	for _, count := range []int{
		big.WORD_COUNT_MINIMUM, big.WORD_COUNT_INCREMENT,
		TWO_WORD_COUNT, big.WORD_COUNT_MAXIMUM,
	} {
		workspace.Modular.Division.Quotient_Count = big.Quotient_Count(count)
		workspace.Modular.Division.Remainder_Count = big.Remainder_Count(count)
		var destination big.Int
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Modular_Square_Root(
			&destination, &source, &modulus, &nonresidue, &workspace,
		))
	}
}

func test_modular_multiply_domains(
	t *testing.T,
	values *[INT_DOMAIN_COUNT]big.Int,
	workspace *big.Int_Modular_Workspace,
) {
	var zero big.Int
	for _, count := range []int{
		big.WORD_COUNT_MINIMUM, big.WORD_COUNT_INCREMENT,
		TWO_WORD_COUNT, big.WORD_COUNT_MAXIMUM,
	} {
		workspace.Division.Quotient_Count = big.Quotient_Count(count)
		workspace.Division.Remainder_Count = big.Remainder_Count(count)
		var result big.Int
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Modular_Multiply(
			&result, &zero, &zero, &values[POSITIVE_TWO_WORD_INDEX], workspace,
		))
	}
}

func test_modular_bound_domains(
	t *testing.T,
	values *[INT_DOMAIN_COUNT]big.Int,
	workspace *big.Int_Modular_Workspace,
) {
	var one big.Int
	big.Int_Set_Uint_64(&one, 1)
	var zero big.Int
	for _, index := range []int{POSITIVE_TWO_WORD_INDEX, POSITIVE_MAXIMUM_WORD_INDEX} {
		count := values[index].Count
		workspace.Division.Quotient_Count = big.Quotient_Count(count)
		workspace.Division.Remainder_Count = big.Remainder_Count(count)
		var result big.Int
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Modular_Inverse(
			&result, &one, &values[index], workspace,
		))
		test_bitwise_result(t, &result, 1)
		workspace.Division.Remainder_Count = big.Remainder_Count(count)
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Modular_Exponent(
			&result, &zero, &zero, &values[index], workspace,
		))
		test_bitwise_result(t, &result, 1)
	}
	test_modular_remainder_domains(t, values, workspace, &one)
}

func test_modular_remainder_domains(
	t *testing.T,
	values *[INT_DOMAIN_COUNT]big.Int,
	workspace *big.Int_Modular_Workspace,
	one *big.Int,
) {
	for _, index := range []int{POSITIVE_TWO_WORD_INDEX, POSITIVE_MAXIMUM_WORD_INDEX} {
		base := values[index]
		modulus := values[index]
		if index == POSITIVE_TWO_WORD_INDEX {
			testify.Equal_Values(t, big.STATUS_OK,
				big.Int_Add(&base, &base, one))
			testify.Equal_Values(t, big.STATUS_OK,
				big.Int_Add(&modulus, &base, one))
		} else {
			testify.Equal_Values(t, big.STATUS_OK,
				big.Int_Subtract(&base, &base, one))
		}
		var result big.Int
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Modular_Exponent(
			&result, &base, one, &modulus, workspace,
		))
		testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&result, &base))
	}
}

func test_binomial(t *testing.T) {
	var workspace big.Int_Product_Workspace
	for _, one := range []struct {
		N    int64
		K    int64
		Want int64
	}{
		{N: 0, K: 0, Want: 1},
		{N: 5, K: 2, Want: 10},
		{N: 10, K: 3, Want: 120},
		{N: 5, K: 5, Want: 1},
		{N: 3, K: 5, Want: 0},
		{N: 5, K: -1, Want: 1},
		{N: -1, K: 0, Want: 0},
		{N: -1, K: -1, Want: 1},
		{N: bits.INTEGER_64_MINIMUM, K: bits.INTEGER_64_MINIMUM, Want: 1},
		{N: bits.INTEGER_64_MAXIMUM, K: 1, Want: bits.INTEGER_64_MAXIMUM},
		{N: 1, K: bits.INTEGER_64_MAXIMUM, Want: 0},
		{N: 2, K: 0, Want: 1},
	} {
		var result big.Int
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Binomial(
			&result, big.Int_64(one.N), big.Int_64(one.K), &workspace,
		))
		test_bitwise_result(t, &result, one.Want)
	}
	var result big.Int
	big.Int_Set_Int_64(&result, 7)
	unchanged := result
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW, big.Int_Binomial(
		&result, big.Int_64(bits.INTEGER_64_MAXIMUM),
		big.Int_64(big.WORD_COUNT_MAXIMUM*SQUARE_ROOT_TEST_DEGREE),
		&workspace,
	))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&result, &unchanged))
	values := int_domain_values(t)
	for index := range values {
		result = values[index]
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Binomial(&result, 0, 0, &workspace))
	}
}

func test_multiply_range(t *testing.T) {
	var workspace big.Int_Product_Workspace
	for _, one := range []struct {
		Minimum int64
		Maximum int64
		Want    int64
	}{
		{Minimum: 2, Maximum: 1, Want: 1},
		{Minimum: 1, Maximum: 0, Want: 1},
		{Minimum: 0, Maximum: 2, Want: 0},
		{Minimum: 1, Maximum: 1, Want: 1},
		{Minimum: 2, Maximum: 5, Want: 120},
		{Minimum: -1, Maximum: -1, Want: -1},
		{Minimum: -5, Maximum: -2, Want: 120},
		{Minimum: -5, Maximum: -3, Want: -60},
		{Minimum: bits.INTEGER_64_MINIMUM, Maximum: bits.INTEGER_64_MINIMUM,
			Want: bits.INTEGER_64_MINIMUM},
		{Minimum: bits.INTEGER_64_MAXIMUM, Maximum: bits.INTEGER_64_MAXIMUM,
			Want: bits.INTEGER_64_MAXIMUM},
	} {
		var result big.Int
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Multiply_Range(
			&result, big.Int_64(one.Minimum), big.Int_64(one.Maximum), &workspace,
		))
		test_bitwise_result(t, &result, one.Want)
	}
	var result big.Int
	big.Int_Set_Int_64(&result, 7)
	unchanged := result
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW, big.Int_Multiply_Range(
		&result, 2, big.BIT_COUNT_MAXIMUM, &workspace,
	))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&result, &unchanged))
	values := int_domain_values(t)
	for index := range values {
		result = values[index]
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Multiply_Range(&result, 1, 0, &workspace))
	}
}

func test_exponent(t *testing.T) {
	var workspace big.Int_Exponent_Workspace
	for _, one := range []struct {
		Base     int64
		Exponent int64
		Want     int64
	}{
		{Base: 0, Exponent: 0, Want: 1},
		{Base: 0, Exponent: 2, Want: 0},
		{Base: 2, Exponent: 10, Want: 1024},
		{Base: -2, Exponent: 3, Want: -8},
		{Base: -2, Exponent: 4, Want: 16},
		{Base: 2, Exponent: -3, Want: 1},
	} {
		var base big.Int
		var exponent big.Int
		var result big.Int
		big.Int_Set_Int_64(&base, big.Int_64(one.Base))
		big.Int_Set_Int_64(&exponent, big.Int_64(one.Exponent))
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Exponent(&result, &base, &exponent, &workspace))
		test_bitwise_result(t, &result, one.Want)
	}
	var base big.Int
	var exponent big.Int
	big.Int_Set_Int_64(&base, -2)
	big.Int_Set_Int_64(&exponent, 3)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Exponent(&base, &base, &exponent, &workspace))
	test_bitwise_result(t, &base, -8)
	big.Int_Set_Int_64(&base, -2)
	big.Int_Set_Int_64(&exponent, 3)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Exponent(&exponent, &base, &exponent, &workspace))
	test_bitwise_result(t, &exponent, -8)
	test_exponent_bounds(t, &workspace)
}

func test_exponent_bounds(t *testing.T, workspace *big.Int_Exponent_Workspace) {
	var base big.Int
	var exponent big.Int
	var result big.Int
	big.Int_Set_Uint_64(&base, 2)
	big.Int_Set_Uint_64(&exponent, big.BIT_INDEX_MAXIMUM)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Exponent(&result, &base, &exponent, workspace))
	testify.Equal(t, big.Bit_Count(big.BIT_COUNT_MAXIMUM), big.Int_Bit_Count(&result))
	testify.Equal(t, big.BIT_SET,
		big.Int_Bit(&result, big.Bit_Index(big.BIT_INDEX_MAXIMUM)))
	big.Int_Set_Uint_64(&exponent, big.BIT_COUNT_MAXIMUM)
	unchanged := result
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW,
		big.Int_Exponent(&result, &base, &exponent, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&result, &unchanged))

	values := int_domain_values(t)
	var one big.Int
	big.Int_Set_Uint_64(&one, 1)
	for index := range values {
		result = values[index]
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Exponent(&result, &values[index], &one, workspace))
		result = values[index]
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Exponent(&result, &one, &values[index], workspace))
	}
}

const SQUARE_ROOT_TEST_DEGREE = 2

func test_square_root(t *testing.T) {
	var workspace big.Int_Square_Root_Workspace
	for _, one := range []struct {
		Source int64
		Want   int64
	}{
		{Source: 0, Want: 0},
		{Source: 1, Want: 1},
		{Source: 2, Want: 1},
		{Source: 3, Want: 1},
		{Source: 4, Want: 2},
		{Source: 15, Want: 3},
		{Source: 16, Want: 4},
		{Source: 17, Want: 4},
	} {
		var source big.Int
		var result big.Int
		big.Int_Set_Int_64(&source, big.Int_64(one.Source))
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Square_Root(&result, &source, &workspace))
		test_bitwise_result(t, &result, one.Want)
	}
	var two_word_bytes [big.WORD_BYTE_COUNT + 1]byte
	two_word_bytes[0] = 1
	var two_word big.Int
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Bytes(&two_word, two_word_bytes[:]))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Square_Root(&two_word, &two_word, &workspace))
	test_bitwise_result(
		t, &two_word, int64(1)<<(big.WORD_BIT_COUNT/SQUARE_ROOT_TEST_DEGREE),
	)

	var maximum_bytes [bytes.SLICE_SIZE_MAXIMUM]byte
	for index := range maximum_bytes {
		maximum_bytes[index] = byte(bits.WORD_8_MAXIMUM)
	}
	var negative big.Int
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Bytes(&negative, maximum_bytes[:]))
	big.Int_Negate(&negative, &negative)
	unchanged := negative
	counts := [...]int{0, 1, 2, big.WORD_COUNT_MAXIMUM}
	for _, count := range counts {
		workspace.Division.Quotient_Count = big.Quotient_Count(count)
		workspace.Division.Remainder_Count = big.Remainder_Count(count)
		testify.Equal_Values(t, big.STATUS_INPUT_INVALID,
			big.Int_Square_Root(&negative, &negative, &workspace))
	}
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&negative, &unchanged))
}

func test_rational(t *testing.T) {
	var zero big.Rat
	testify.Equal(t, big.SIGN_ZERO, big.Rat_Sign(&zero))
	testify.True(t, bool(big.Rat_Is_Integer(&zero)))
	var workspace big.Rat_Workspace
	var numerator big.Int
	var denominator big.Int
	big.Int_Set_Int_64(&numerator, 6)
	big.Int_Set_Int_64(&denominator, -8)
	var value big.Rat
	testify.Equal_Values(t, big.STATUS_OK,
		big.Rat_Set_Fraction(&value, &numerator, &denominator, &workspace))
	rat_expect(t, &value, -3, 4)
	testify.Equal(t, big.SIGN_NEGATIVE, big.Rat_Sign(&value))
	testify.False(t, bool(big.Rat_Is_Integer(&value)))
	var copy big.Rat
	big.Rat_Set(&copy, &value)
	big.Rat_Absolute(&copy, &copy)
	rat_expect(t, &copy, 3, 4)
	testify.Equal(t, big.SIGN_POSITIVE, big.Rat_Sign(&copy))
	big.Rat_Negate(&copy, &copy)
	rat_expect(t, &copy, -3, 4)
	big.Rat_Set_Int_64(&copy, -5)
	rat_expect(t, &copy, -5, 1)
	big.Rat_Set_Uint_64(&copy, 7)
	rat_expect(t, &copy, 7, 1)
	test_rat_fraction_64_inverse(t, &workspace)
	test_rat_arithmetic(t, &workspace)
	test_rat_text(t, &workspace)
	test_rat_float_text(t, &workspace)
	test_rat_float_precision(t, &workspace)
	test_rat_set_float_32_bits(t, &workspace)
	test_rat_float_32_bits(t, &workspace)
	test_rat_set_float_64_bits(t, &workspace)
	test_rat_float_64_bits(t, &workspace)
	test_rat_parse_fraction(t, &workspace)
	test_rat_parse(t, &workspace)
	test_rat_bounds(t, &workspace)
	test_rat_overflow(t, &workspace)
	exercise_rat_workspace_domains(t, &workspace)
	exercise_rat_int_domains(t, &workspace)
}

func test_rat_text(t *testing.T, rational_workspace *big.Rat_Workspace) {
	zero := rat_set_pair(t, 0, 1, rational_workspace)
	rat_text_expect(t, &zero, "0/1", "0")
	integer := rat_set_pair(t, 5, 1, rational_workspace)
	rat_text_expect(t, &integer, "5/1", "5")
	negative_one := rat_set_pair(t, -1, 1, rational_workspace)
	rat_text_expect(t, &negative_one, "-1/1", "-1")
	fraction := rat_set_pair(t, -3, 4, rational_workspace)
	rat_text_expect(t, &fraction, "-3/4", "-3/4")
	var workspace big.Rat_Text_Workspace
	for _, size := range []int{0, 1, 2, big.INT_TEXT_SIZE_MAXIMUM} {
		var storage [big.INT_TEXT_SIZE_MAXIMUM]byte
		for index := range storage {
			storage[index] = 'x'
		}
		unchanged := storage
		count, status := big.Rat_Text_Into(storage[:size], &fraction, &workspace)
		compact_count, compact_status := big.Rat_Rational_Text_Into(
			storage[:size], &fraction, &workspace,
		)
		if size < len("-3/4") {
			testify.Equal_Values(t, big.STATUS_DESTINATION_TOO_SMALL, status)
			testify.Equal_Values(t, big.STATUS_DESTINATION_TOO_SMALL, compact_status)
			testify.Equal(t, big.Rat_Fraction_Text_Count(len("-3/4")), count)
			testify.Equal(t, big.Rat_Text_Count(len("-3/4")), compact_count)
			testify.Equal(t, unchanged, storage)
		} else {
			testify.Equal_Values(t, big.STATUS_OK, status)
			testify.Equal_Values(t, big.STATUS_OK, compact_status)
		}
	}
	test_rat_text_maximum(t, rational_workspace, &workspace)
}

func test_rat_text_maximum(
	t *testing.T, rational_workspace *big.Rat_Workspace, workspace *big.Rat_Text_Workspace,
) {
	var numerator_words [big.RAT_WORD_COUNT_MAXIMUM]big.Word
	for index := range numerator_words {
		numerator_words[index] = big.Word(bits.WORD_64_MAXIMUM)
	}
	denominator_words := numerator_words
	denominator_words[ZERO_INDEX]--
	var numerator big.Int
	var denominator big.Int
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Set_Words(&numerator, numerator_words[:]))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Set_Words(&denominator, denominator_words[:]))
	big.Int_Negate(&numerator, &numerator)
	var value big.Rat
	testify.Equal_Values(t, big.STATUS_OK,
		big.Rat_Set_Fraction(&value, &numerator, &denominator, rational_workspace))
	var storage [big.RAT_TEXT_SIZE_MAXIMUM]byte
	count, status := big.Rat_Text_Into(storage[:], &value, workspace)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, big.Rat_Fraction_Text_Count(big.RAT_TEXT_SIZE_MAXIMUM), count)
	compact_count, status := big.Rat_Rational_Text_Into(storage[:], &value, workspace)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, big.Rat_Text_Count(big.RAT_TEXT_SIZE_MAXIMUM), compact_count)
}

func rat_text_expect(t *testing.T, value *big.Rat, fraction string, compact string) {
	var workspace big.Rat_Text_Workspace
	var storage [big.RAT_TEXT_SIZE_MAXIMUM]byte
	count, status := big.Rat_Text_Into(storage[:], value, &workspace)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, fraction, string(storage[:count]))
	compact_count, compact_status := big.Rat_Rational_Text_Into(
		storage[:], value, &workspace,
	)
	testify.Equal_Values(t, big.STATUS_OK, compact_status)
	testify.Equal(t, compact, string(storage[:compact_count]))
}

func test_rat_float_text(t *testing.T, rational_workspace *big.Rat_Workspace) {
	for _, test := range []struct {
		Numerator   int64
		Denominator int64
		Precision   big.Rat_Precision
		Expected    string
	}{
		{0, 1, 0, "0"},
		{-1, 1, 0, "-1"},
		{1, 2, 0, "1"},
		{-1, 2, 0, "-1"},
		{1, 3, 2, "0.33"},
		{1, 6, 1, "0.2"},
		{-1, 6, 1, "-0.2"},
		{999, 1000, 2, "1.00"},
		{1, 4, 3, "0.250"},
		{2, 1, 2, "2.00"},
		{12, 1, 0, "12"},
	} {
		value := rat_set_pair(t, test.Numerator, test.Denominator, rational_workspace)
		rat_float_text_expect(t, &value, test.Precision, test.Expected)
	}
	test_rat_float_text_validation(t)
	test_rat_float_text_destination(t, rational_workspace)
	test_rat_float_text_workspace_domains(t, rational_workspace)
	test_rat_float_text_maximum(t)
}

func rat_float_text_expect(
	t *testing.T, value *big.Rat, precision big.Rat_Precision, expected string,
) {
	var workspace big.Rat_Float_Text_Workspace
	var storage [big.RAT_TEXT_SIZE_MAXIMUM]byte
	count, status := big.Rat_Float_Text_Into(
		storage[:len(expected)], value, precision, &workspace,
	)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, big.Rat_Text_Count(len(expected)), count)
	testify.Equal(t, expected, string(storage[:count]))
}

func test_rat_float_text_validation(t *testing.T) {
	for _, test := range []struct {
		Value  big.Rat_Precision_Unvalidated
		Status big.Validation_Status
	}{
		{big.RAT_PRECISION_UNVALIDATED_MINIMUM, big.STATUS_INPUT_INVALID},
		{big.RAT_PRECISION_MINIMUM, big.STATUS_OK},
		{big.Rat_Precision_Unvalidated(1), big.STATUS_OK},
		{big.RAT_PRECISION_MAXIMUM, big.STATUS_OK},
		{big.RAT_PRECISION_UNVALIDATED_MAXIMUM, big.STATUS_INPUT_INVALID},
	} {
		precision, status := big.Rat_Precision_Validate(test.Value)
		testify.Equal_Values(t, test.Status, status)
		if status == big.Validation_Status(big.STATUS_OK) {
			testify.Equal(t, big.Rat_Precision(test.Value), precision)
		}
	}
}

func test_rat_float_text_destination(
	t *testing.T, rational_workspace *big.Rat_Workspace,
) {
	value := rat_set_pair(t, -3, 4, rational_workspace)
	const EXPECTED = "-0.75"
	var workspace big.Rat_Float_Text_Workspace
	for _, size := range []int{
		0, 1, 2, len(EXPECTED) - INDEX_STEP, len(EXPECTED), big.INT_TEXT_SIZE_MAXIMUM,
	} {
		var storage [big.INT_TEXT_SIZE_MAXIMUM]byte
		for index := range storage {
			storage[index] = 'x'
		}
		unchanged := storage
		count, status := big.Rat_Float_Text_Into(
			storage[:size], &value, 2, &workspace,
		)
		testify.Equal(t, big.Rat_Text_Count(len(EXPECTED)), count)
		if size < len(EXPECTED) {
			testify.Equal_Values(t, big.STATUS_DESTINATION_TOO_SMALL, status)
			testify.Equal(t, unchanged, storage)
		} else {
			testify.Equal_Values(t, big.STATUS_OK, status)
			testify.Equal(t, EXPECTED, string(storage[:count]))
		}
	}
}

func test_rat_float_text_workspace_domains(
	t *testing.T, rational_workspace *big.Rat_Workspace,
) {
	value := rat_set_pair(t, 1, 3, rational_workspace)
	var workspace big.Rat_Float_Text_Workspace
	var storage [len("0.33")]byte
	for _, count := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		workspace.Division.Quotient_Count = big.Quotient_Count(count)
		workspace.Division.Remainder_Count = big.Remainder_Count(count)
		_, status := big.Rat_Float_Text_Into(storage[:], &value, 2, &workspace)
		testify.Equal_Values(t, big.STATUS_OK, status)
	}
}

func test_rat_float_text_maximum(t *testing.T) {
	var words [big.RAT_WORD_COUNT_MAXIMUM]big.Word
	for index := range words {
		words[index] = big.Word(bits.WORD_64_MAXIMUM)
	}
	var integer big.Int
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Words(&integer, words[:]))
	big.Int_Negate(&integer, &integer)
	var denominator big.Int
	big.Int_Set_Uint_64(&denominator, 2)
	var value big.Rat
	var rational_workspace big.Rat_Workspace
	testify.Equal_Values(t, big.STATUS_OK,
		big.Rat_Set_Fraction(&value, &integer, &denominator, &rational_workspace))
	var workspace big.Rat_Float_Text_Workspace
	var storage [big.RAT_TEXT_SIZE_MAXIMUM]byte
	count, status := big.Rat_Float_Text_Into(
		storage[:], &value, big.RAT_PRECISION_MAXIMUM, &workspace,
	)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, big.Rat_Text_Count(big.RAT_TEXT_SIZE_MAXIMUM), count)
}

func test_rat_float_precision(t *testing.T, rational_workspace *big.Rat_Workspace) {
	for _, test := range []struct {
		Numerator   int64
		Denominator int64
		Precision   big.Decimal_Place_Count
		Exact       bool
	}{
		{0, 1, 0, true},
		{1, 1, 0, true},
		{1, 2, 1, true},
		{1, 3, 0, false},
		{1, 4, 2, true},
		{1, 5, 1, true},
		{1, 6, 1, false},
		{1, 7, 0, false},
		{1, 9, 0, false},
		{1, 11, 0, false},
	} {
		value := rat_set_pair(t, test.Numerator, test.Denominator, rational_workspace)
		var workspace big.Rat_Float_Precision_Workspace
		precision, exact := big.Rat_Float_Precision(&value, &workspace)
		testify.Equal(t, test.Precision, precision)
		testify.Equal(t, test.Exact, bool(exact))
	}
	test_rat_float_precision_maximum(t, rational_workspace)
}

func test_rat_float_precision_maximum(
	t *testing.T, rational_workspace *big.Rat_Workspace,
) {
	var denominator big.Int
	big.Int_Set_Uint_64(&denominator, 1)
	shift, status := big.Shift_Count_Validate(big.RAT_FLOAT_PRECISION_COUNT_MAXIMUM)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Shift_Left(&denominator, &denominator, shift))
	var numerator big.Int
	big.Int_Set_Uint_64(&numerator, 1)
	var value big.Rat
	testify.Equal_Values(t, big.STATUS_OK,
		big.Rat_Set_Fraction(&value, &numerator, &denominator, rational_workspace))
	var workspace big.Rat_Float_Precision_Workspace
	precision, exact := big.Rat_Float_Precision(&value, &workspace)
	testify.Equal(t,
		big.Decimal_Place_Count(big.RAT_FLOAT_PRECISION_COUNT_MAXIMUM), precision)
	testify.True(t, bool(exact))
}

func test_rat_set_float_32_bits(t *testing.T, workspace *big.Rat_Workspace) {
	for _, test := range []struct {
		Encoding    big.Float_32_Bits
		Numerator   int64
		Denominator int64
	}{
		{FLOAT_32_POSITIVE_ZERO_BITS, 0, 1},
		{FLOAT_32_NEGATIVE_ZERO_BITS, 0, 1},
		{FLOAT_32_HALF_BITS, 1, 2},
		{FLOAT_32_NEGATIVE_HALF_BITS, -1, 2},
		{FLOAT_32_THREE_HALVES_BITS, 3, 2},
	} {
		var value big.Rat
		testify.Equal_Values(t, big.STATUS_OK,
			big.Rat_Set_Float_32_Bits(&value, test.Encoding))
		rat_expect(t, &value, test.Numerator, test.Denominator)
	}
	var value big.Rat
	big.Rat_Set_Int_64(&value, 7)
	for _, encoding := range []big.Float_32_Bits{
		FLOAT_32_POSITIVE_INFINITY_BITS, FLOAT_32_NEGATIVE_INFINITY_BITS,
		FLOAT_32_NOT_A_NUMBER_BITS, big.Float_32_Bits(bits.WORD_32_MAXIMUM),
	} {
		unchanged := value
		testify.Equal_Values(t, big.STATUS_INPUT_INVALID,
			big.Rat_Set_Float_32_Bits(&value, encoding))
		testify.Equal(t, big.ORDER_SAME, big.Rat_Compare(&value, &unchanged, workspace))
	}
	test_rat_set_float_32_boundaries(t, workspace)
}

func test_rat_set_float_32_boundaries(t *testing.T, workspace *big.Rat_Workspace) {
	for _, encoding := range []big.Float_32_Bits{1, 2} {
		var value big.Rat
		testify.Equal_Values(t, big.STATUS_OK,
			big.Rat_Set_Float_32_Bits(&value, encoding))
		var numerator big.Int
		var denominator big.Int
		big.Rat_Numerator_Into(&numerator, &value)
		big.Rat_Denominator_Into(&denominator, &value)
		numerator_word, status := big.Int_Uint_64(&numerator)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, big.Word_64(1), numerator_word)
		expected_shift := big.FLOAT_32_SUBNORMAL_DENOMINATOR_SHIFT_MAXIMUM -
			int(encoding) + INDEX_STEP
		testify.Equal(t, big.Bit_Count(expected_shift+INDEX_STEP),
			big.Int_Bit_Count(&denominator))
	}
	var maximum big.Rat
	testify.Equal_Values(t, big.STATUS_OK,
		big.Rat_Set_Float_32_Bits(&maximum, FLOAT_32_MAXIMUM_FINITE_BITS))
	testify.True(t, bool(big.Rat_Is_Integer(&maximum)))
	var numerator big.Int
	big.Rat_Numerator_Into(&numerator, &maximum)
	testify.Equal(t, big.Bit_Count(big.FLOAT_32_VALUE_BIT_COUNT_MAXIMUM),
		big.Int_Bit_Count(&numerator))
	var one big.Rat
	big.Rat_Set_Int_64(&one, 1)
	testify.Equal(t, big.ORDER_AFTER,
		big.Rat_Compare(&maximum, &one, workspace))
}

func test_rat_float_32_bits(t *testing.T, rational_workspace *big.Rat_Workspace) {
	for _, test := range []struct {
		Numerator   int64
		Denominator int64
		Encoding    big.Float_32_Value_Bits
	}{
		{0, 1, big.Float_32_Value_Bits(FLOAT_32_POSITIVE_ZERO_BITS)},
		{1, 2, big.Float_32_Value_Bits(FLOAT_32_HALF_BITS)},
		{-1, 2, big.Float_32_Value_Bits(FLOAT_32_NEGATIVE_HALF_BITS)},
		{1, 4, FLOAT_32_QUARTER_BITS},
		{3, 2, big.Float_32_Value_Bits(FLOAT_32_THREE_HALVES_BITS)},
		{3, 1, FLOAT_32_THREE_BITS},
	} {
		value := rat_set_pair(t, test.Numerator, test.Denominator, rational_workspace)
		var workspace big.Rat_Float_32_Workspace
		encoding, exact := big.Rat_Float_32_Bits(&value, &workspace)
		testify.Equal(t, test.Encoding, encoding)
		testify.True(t, bool(exact))
	}
	test_rat_float_32_midpoints(t, rational_workspace)
	test_rat_float_32_boundaries(t, rational_workspace)
	test_rat_float_32_workspace_domains(t, rational_workspace)
}

func test_rat_float_32_midpoints(t *testing.T, rational_workspace *big.Rat_Workspace) {
	const DENOMINATOR = int64(INDEX_STEP) << (big.FLOAT_32_MANTISSA_BIT_COUNT + INDEX_STEP)
	for _, test := range []struct {
		Numerator int64
		Encoding  big.Float_32_Value_Bits
	}{
		{DENOMINATOR + INDEX_STEP, FLOAT_32_ONE_BITS},
		{DENOMINATOR + INDEX_STEP + INDEX_STEP + INDEX_STEP, FLOAT_32_ONE_PLUS_TWO_BITS},
	} {
		value := rat_set_pair(t, test.Numerator, DENOMINATOR, rational_workspace)
		var workspace big.Rat_Float_32_Workspace
		encoding, exact := big.Rat_Float_32_Bits(&value, &workspace)
		testify.Equal(t, test.Encoding, encoding)
		testify.False(t, bool(exact))
	}
}

func test_rat_float_32_boundaries(
	t *testing.T, rational_workspace *big.Rat_Workspace,
) {
	var maximum big.Rat
	testify.Equal_Values(t, big.STATUS_OK,
		big.Rat_Set_Float_32_Bits(&maximum, FLOAT_32_MAXIMUM_FINITE_BITS))
	var workspace big.Rat_Float_32_Workspace
	encoding, exact := big.Rat_Float_32_Bits(&maximum, &workspace)
	testify.Equal(t, big.Float_32_Value_Bits(FLOAT_32_MAXIMUM_FINITE_BITS), encoding)
	testify.True(t, bool(exact))
	huge := maximum_rat_component(t)
	testify.Equal_Values(t, big.STATUS_OK, big.Rat_Set_Int(&maximum, &huge))
	encoding, exact = big.Rat_Float_32_Bits(&maximum, &workspace)
	testify.Equal(t, big.Float_32_Value_Bits(FLOAT_32_POSITIVE_INFINITY_BITS), encoding)
	testify.False(t, bool(exact))
	big.Rat_Negate(&maximum, &maximum)
	encoding, exact = big.Rat_Float_32_Bits(&maximum, &workspace)
	testify.Equal(t, big.Float_32_Value_Bits(FLOAT_32_NEGATIVE_INFINITY_BITS), encoding)
	testify.False(t, bool(exact))
	test_rat_float_32_small(t, rational_workspace)
}

func test_rat_float_32_small(t *testing.T, rational_workspace *big.Rat_Workspace) {
	var numerator big.Int
	var denominator big.Int
	big.Int_Set_Uint_64(&numerator, 1)
	for index := 0; index < 2; index++ {
		big.Int_Set_Uint_64(&denominator, 1)
		shift := big.Shift_Count(big.FLOAT_32_SUBNORMAL_DENOMINATOR_SHIFT_MAXIMUM - index)
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Shift_Left(&denominator, &denominator, shift))
		var value big.Rat
		testify.Equal_Values(t, big.STATUS_OK,
			big.Rat_Set_Fraction(&value, &numerator, &denominator, rational_workspace))
		var workspace big.Rat_Float_32_Workspace
		encoding, exact := big.Rat_Float_32_Bits(&value, &workspace)
		testify.Equal(t, big.Float_32_Value_Bits(index+INDEX_STEP), encoding)
		testify.True(t, bool(exact))
	}
	big.Int_Set_Uint_64(&denominator, 1)
	shift := big.Shift_Count(
		big.FLOAT_32_SUBNORMAL_DENOMINATOR_SHIFT_MAXIMUM + INDEX_STEP,
	)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Shift_Left(&denominator, &denominator, shift))
	var value big.Rat
	testify.Equal_Values(t, big.STATUS_OK,
		big.Rat_Set_Fraction(&value, &numerator, &denominator, rational_workspace))
	var workspace big.Rat_Float_32_Workspace
	encoding, exact := big.Rat_Float_32_Bits(&value, &workspace)
	testify.Equal(t, big.Float_32_Value_Bits(FLOAT_32_POSITIVE_ZERO_BITS), encoding)
	testify.False(t, bool(exact))
	denominator = maximum_rat_component(t)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Rat_Set_Fraction(&value, &numerator, &denominator, rational_workspace))
	encoding, exact = big.Rat_Float_32_Bits(&value, &workspace)
	testify.Equal(t, big.Float_32_Value_Bits(FLOAT_32_POSITIVE_ZERO_BITS), encoding)
	testify.False(t, bool(exact))
}

func test_rat_float_32_workspace_domains(
	t *testing.T, rational_workspace *big.Rat_Workspace,
) {
	value := rat_set_pair(t, 1, 3, rational_workspace)
	var workspace big.Rat_Float_32_Workspace
	for _, count := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		workspace.Division.Quotient_Count = big.Quotient_Count(count)
		workspace.Division.Remainder_Count = big.Remainder_Count(count)
		big.Rat_Float_32_Bits(&value, &workspace)
	}
}

func test_rat_set_float_64_bits(t *testing.T, workspace *big.Rat_Workspace) {
	for _, test := range []struct {
		Encoding    big.Float_64_Bits
		Numerator   int64
		Denominator int64
	}{
		{FLOAT_64_POSITIVE_ZERO_BITS, 0, 1},
		{FLOAT_64_NEGATIVE_ZERO_BITS, 0, 1},
		{FLOAT_64_HALF_BITS, 1, 2},
		{FLOAT_64_NEGATIVE_HALF_BITS, -1, 2},
		{FLOAT_64_THREE_HALVES_BITS, 3, 2},
		{FLOAT_64_FIVE_QUARTERS_BITS, 5, 4},
	} {
		var value big.Rat
		testify.Equal_Values(t, big.STATUS_OK,
			big.Rat_Set_Float_64_Bits(&value, test.Encoding))
		rat_expect(t, &value, test.Numerator, test.Denominator)
	}
	test_rat_set_float_64_bits_boundaries(t, workspace)
	test_rat_set_float_64_bits_invalid(t, workspace)
}

func test_rat_set_float_64_bits_boundaries(t *testing.T, workspace *big.Rat_Workspace) {
	for _, float_bits := range []uint64{1, 2} {
		var value big.Rat
		testify.Equal_Values(t, big.STATUS_OK,
			big.Rat_Set_Float_64_Bits(&value, big.Float_64_Bits(float_bits)))
		var numerator big.Int
		var denominator big.Int
		big.Rat_Numerator_Into(&numerator, &value)
		big.Rat_Denominator_Into(&denominator, &value)
		numerator_word, status := big.Int_Uint_64(&numerator)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, big.Word_64(1), numerator_word)
		expected_shift := big.FLOAT_64_SUBNORMAL_DENOMINATOR_SHIFT_MAXIMUM -
			int(float_bits) + INDEX_STEP
		testify.Equal(t, big.Bit_Count(expected_shift+INDEX_STEP),
			big.Int_Bit_Count(&denominator))
	}
	var maximum big.Rat
	testify.Equal_Values(t, big.STATUS_OK,
		big.Rat_Set_Float_64_Bits(&maximum, FLOAT_64_MAXIMUM_FINITE_BITS))
	testify.True(t, bool(big.Rat_Is_Integer(&maximum)))
	var numerator big.Int
	big.Rat_Numerator_Into(&numerator, &maximum)
	testify.Equal(t, big.Bit_Count(big.FLOAT_64_VALUE_BIT_COUNT_MAXIMUM),
		big.Int_Bit_Count(&numerator))
	var one big.Rat
	big.Rat_Set_Int_64(&one, 1)
	testify.Equal(t, big.ORDER_AFTER, big.Rat_Compare(&maximum, &one, workspace))
}

func test_rat_set_float_64_bits_invalid(t *testing.T, workspace *big.Rat_Workspace) {
	var value big.Rat
	big.Rat_Set_Int_64(&value, 7)
	for _, float_bits := range []big.Float_64_Bits{
		FLOAT_64_POSITIVE_INFINITY_BITS,
		FLOAT_64_NEGATIVE_INFINITY_BITS,
		FLOAT_64_NOT_A_NUMBER_BITS,
		big.Float_64_Bits(bits.WORD_64_MAXIMUM),
	} {
		unchanged := value
		testify.Equal_Values(t, big.STATUS_INPUT_INVALID,
			big.Rat_Set_Float_64_Bits(&value, float_bits))
		testify.Equal(t, big.ORDER_SAME, big.Rat_Compare(&value, &unchanged, workspace))
	}
}

func test_rat_float_64_bits(t *testing.T, rational_workspace *big.Rat_Workspace) {
	for _, test := range []struct {
		Numerator   int64
		Denominator int64
		Encoding    big.Float_64_Value_Bits
	}{
		{0, 1, big.Float_64_Value_Bits(FLOAT_64_POSITIVE_ZERO_BITS)},
		{1, 2, big.Float_64_Value_Bits(FLOAT_64_HALF_BITS)},
		{-1, 2, big.Float_64_Value_Bits(FLOAT_64_NEGATIVE_HALF_BITS)},
		{3, 2, big.Float_64_Value_Bits(FLOAT_64_THREE_HALVES_BITS)},
		{5, 4, big.Float_64_Value_Bits(FLOAT_64_FIVE_QUARTERS_BITS)},
		{3, 1, FLOAT_64_THREE_BITS},
	} {
		value := rat_set_pair(t, test.Numerator, test.Denominator, rational_workspace)
		var workspace big.Rat_Float_64_Workspace
		encoding, exact := big.Rat_Float_64_Bits(&value, &workspace)
		testify.Equal(t, test.Encoding, encoding)
		testify.True(t, bool(exact))
	}
	test_rat_float_64_bits_midpoints(t, rational_workspace)
	test_rat_float_64_bits_boundaries(t, rational_workspace)
	test_rat_float_64_workspace_domains(t, rational_workspace)
}

func test_rat_float_64_bits_midpoints(
	t *testing.T, rational_workspace *big.Rat_Workspace,
) {
	const DENOMINATOR = int64(INDEX_STEP) << (big.FLOAT_64_MANTISSA_BIT_COUNT + INDEX_STEP)
	for _, test := range []struct {
		Numerator int64
		Encoding  big.Float_64_Value_Bits
	}{
		{DENOMINATOR + INDEX_STEP, FLOAT_64_ONE_BITS},
		{DENOMINATOR + INDEX_STEP + INDEX_STEP + INDEX_STEP, FLOAT_64_ONE_PLUS_TWO_BITS},
	} {
		value := rat_set_pair(t, test.Numerator, DENOMINATOR, rational_workspace)
		var workspace big.Rat_Float_64_Workspace
		encoding, exact := big.Rat_Float_64_Bits(&value, &workspace)
		testify.Equal(t, test.Encoding, encoding)
		testify.False(t, bool(exact))
	}
}

func test_rat_float_64_bits_boundaries(
	t *testing.T, rational_workspace *big.Rat_Workspace,
) {
	test_rat_float_64_subnormal_boundaries(t, rational_workspace)
	test_rat_float_64_maximum_finite(t)
	test_rat_float_64_overflow(t)
	test_rat_float_64_underflow(t, rational_workspace)
}

func test_rat_float_64_subnormal_boundaries(
	t *testing.T, rational_workspace *big.Rat_Workspace,
) {
	for index := 0; index < 2; index++ {
		var numerator big.Int
		var denominator big.Int
		big.Int_Set_Uint_64(&numerator, 1)
		big.Int_Set_Uint_64(&denominator, 1)
		shift, status := big.Shift_Count_Validate(
			big.Shift_Count_Unvalidated(
				big.FLOAT_64_SUBNORMAL_DENOMINATOR_SHIFT_MAXIMUM - index,
			),
		)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Shift_Left(&denominator, &denominator, shift))
		var value big.Rat
		testify.Equal_Values(t, big.STATUS_OK,
			big.Rat_Set_Fraction(&value, &numerator, &denominator, rational_workspace))
		var workspace big.Rat_Float_64_Workspace
		encoding, exact := big.Rat_Float_64_Bits(&value, &workspace)
		testify.Equal(t, big.Float_64_Value_Bits(index+INDEX_STEP), encoding)
		testify.True(t, bool(exact))
	}
	var numerator big.Int
	var denominator big.Int
	big.Int_Set_Uint_64(&numerator, 1)
	big.Int_Set_Uint_64(&denominator, 1)
	shift, status := big.Shift_Count_Validate(
		big.FLOAT_64_SUBNORMAL_DENOMINATOR_SHIFT_MAXIMUM + INDEX_STEP,
	)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Shift_Left(&denominator, &denominator, shift))
	var value big.Rat
	testify.Equal_Values(t, big.STATUS_OK,
		big.Rat_Set_Fraction(&value, &numerator, &denominator, rational_workspace))
	var workspace big.Rat_Float_64_Workspace
	encoding, exact := big.Rat_Float_64_Bits(&value, &workspace)
	testify.Equal(t, big.Float_64_Value_Bits(FLOAT_64_POSITIVE_ZERO_BITS), encoding)
	testify.False(t, bool(exact))
}

func test_rat_float_64_maximum_finite(t *testing.T) {
	var value big.Rat
	testify.Equal_Values(t, big.STATUS_OK,
		big.Rat_Set_Float_64_Bits(&value, FLOAT_64_MAXIMUM_FINITE_BITS))
	var workspace big.Rat_Float_64_Workspace
	encoding, exact := big.Rat_Float_64_Bits(&value, &workspace)
	testify.Equal(t, big.Float_64_Value_Bits(FLOAT_64_MAXIMUM_FINITE_BITS), encoding)
	testify.True(t, bool(exact))
}

func test_rat_float_64_overflow(t *testing.T) {
	maximum := maximum_rat_component(t)
	var value big.Rat
	testify.Equal_Values(t, big.STATUS_OK, big.Rat_Set_Int(&value, &maximum))
	var workspace big.Rat_Float_64_Workspace
	encoding, exact := big.Rat_Float_64_Bits(&value, &workspace)
	testify.Equal(t, big.Float_64_Value_Bits(FLOAT_64_POSITIVE_INFINITY_BITS), encoding)
	testify.False(t, bool(exact))
	big.Rat_Negate(&value, &value)
	encoding, exact = big.Rat_Float_64_Bits(&value, &workspace)
	testify.Equal(t, big.Float_64_Value_Bits(FLOAT_64_NEGATIVE_INFINITY_BITS), encoding)
	testify.False(t, bool(exact))
}

func test_rat_float_64_underflow(
	t *testing.T, rational_workspace *big.Rat_Workspace,
) {
	denominator := maximum_rat_component(t)
	var numerator big.Int
	big.Int_Set_Uint_64(&numerator, 1)
	var value big.Rat
	testify.Equal_Values(t, big.STATUS_OK,
		big.Rat_Set_Fraction(&value, &numerator, &denominator, rational_workspace))
	var workspace big.Rat_Float_64_Workspace
	encoding, exact := big.Rat_Float_64_Bits(&value, &workspace)
	testify.Equal(t, big.Float_64_Value_Bits(FLOAT_64_POSITIVE_ZERO_BITS), encoding)
	testify.False(t, bool(exact))
	big.Rat_Negate(&value, &value)
	encoding, exact = big.Rat_Float_64_Bits(&value, &workspace)
	testify.Equal(t, big.Float_64_Value_Bits(FLOAT_64_NEGATIVE_ZERO_BITS), encoding)
	testify.False(t, bool(exact))
}

func test_rat_float_64_workspace_domains(
	t *testing.T, rational_workspace *big.Rat_Workspace,
) {
	value := rat_set_pair(t, 1, 3, rational_workspace)
	var workspace big.Rat_Float_64_Workspace
	for _, count := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		workspace.Division.Quotient_Count = big.Quotient_Count(count)
		workspace.Division.Remainder_Count = big.Remainder_Count(count)
		big.Rat_Float_64_Bits(&value, &workspace)
	}
}

func maximum_rat_component(t *testing.T) (value big.Int) {
	var words [big.RAT_WORD_COUNT_MAXIMUM]big.Word
	for index := range words {
		words[index] = big.Word(bits.WORD_64_MAXIMUM)
	}
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Words(&value, words[:]))
	return value
}

func test_rat_parse_fraction(t *testing.T, rational_workspace *big.Rat_Workspace) {
	for _, test := range []struct {
		Text        string
		Numerator   int64
		Denominator int64
	}{
		{"0/1", 0, 1},
		{"3/4", 3, 4},
		{"-6/8", -3, 4},
		{"+0x10/0b10", 8, 1},
		{"010/010", 1, 1},
		{"1_0/2", 5, 1},
	} {
		var value big.Rat
		var workspace big.Rat_Parse_Fraction_Workspace
		status := big.Rat_Parse_Fraction(&value, []byte(test.Text), &workspace)
		testify.Equal_Values(t, big.STATUS_OK, status)
		rat_expect(t, &value, test.Numerator, test.Denominator)
	}
	test_rat_parse_fraction_invalid(t, rational_workspace)
	test_rat_parse_fraction_bounds(t, rational_workspace)
	test_rat_parse_fraction_workspace_domains(t)
}

func test_rat_parse_fraction_invalid(t *testing.T, rational_workspace *big.Rat_Workspace) {
	for _, source := range [][]byte{
		nil, {}, {'1'}, {'/', '1'}, {'1', '/'}, {'1', '/', '0'},
		{'1', '/', '-', '2'}, {'1', '/', '+', '2'}, {'1', '/', '2', '/', '3'},
		{'1', '.', '0', '/', '2'}, {'1', '/', '_', '2'},
	} {
		var value big.Rat
		big.Rat_Set_Int_64(&value, 7)
		unchanged := value
		var workspace big.Rat_Parse_Fraction_Workspace
		status := big.Rat_Parse_Fraction(&value, source, &workspace)
		testify.Equal_Values(t, big.STATUS_INPUT_INVALID, status)
		testify.Equal(t, big.ORDER_SAME, big.Rat_Compare(
			&value, &unchanged, rational_workspace,
		))
	}
}

func test_rat_parse_fraction_bounds(t *testing.T, rational_workspace *big.Rat_Workspace) {
	var overflow [big.RAT_COMPONENT_TEXT_DIGIT_COUNT_MAXIMUM + len("0b1/1")]byte
	copy(overflow[:], "0b1")
	for count := len("0b1"); count < len(overflow)-len("/1"); count++ {
		overflow[count] = '0'
	}
	copy(overflow[len(overflow)-len("/1"):], "/1")
	var value big.Rat
	big.Rat_Set_Int_64(&value, 7)
	unchanged := value
	var workspace big.Rat_Parse_Fraction_Workspace
	status := big.Rat_Parse_Fraction(&value, overflow[:], &workspace)
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW, status)
	testify.Equal(t, big.ORDER_SAME, big.Rat_Compare(
		&value, &unchanged, rational_workspace,
	))
	for _, size := range []int{
		0, 1, 2, big.RAT_PARSE_FRACTION_TEXT_SIZE_MAXIMUM,
		big.RAT_PARSE_FRACTION_TEXT_UNVALIDATED_SIZE_MAXIMUM,
	} {
		var source [big.RAT_PARSE_FRACTION_TEXT_UNVALIDATED_SIZE_MAXIMUM]byte
		status = big.Rat_Parse_Fraction(&value, source[:size], &workspace)
		testify.Equal_Values(t, big.STATUS_INPUT_INVALID, status)
	}
}

func test_rat_parse_fraction_workspace_domains(t *testing.T) {
	var value big.Rat
	var workspace big.Rat_Parse_Fraction_Workspace
	for _, count := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		workspace.Rational.Greatest_Common.Division.Quotient_Count =
			big.Quotient_Count(count)
		workspace.Rational.Greatest_Common.Division.Remainder_Count =
			big.Remainder_Count(count)
		status := big.Rat_Parse_Fraction(&value, []byte("3/4"), &workspace)
		testify.Equal_Values(t, big.STATUS_OK, status)
	}
}

func test_rat_parse(t *testing.T, rational_workspace *big.Rat_Workspace) {
	for _, test := range []struct {
		Text        string
		Numerator   int64
		Denominator int64
	}{
		{"0", 0, 1},
		{"-0", 0, 1},
		{"1", 1, 1},
		{".5", 1, 2},
		{"1.", 1, 1},
		{"1.25", 5, 4},
		{"1e2", 100, 1},
		{"1e-2", 1, 100},
		{"1p3", 8, 1},
		{"0b1.1", 3, 2},
		{"0o1.4", 3, 2},
		{"0x1.8", 3, 2},
		{"0x1p4", 16, 1},
		{"0x1e2", 482, 1},
		{"-0x1", -1, 1},
		{"0x10000000000000000p-64", 1, 1},
		{"1_0.0_0e-1", 1, 1},
		{"0x_1.8p0", 3, 2},
		{"3/4", 3, 4},
		{"0e99999", 0, 1},
		{"0e9223372036854775807", 0, 1},
		{"0e-9223372036854775808", 0, 1},
		{"0.0e-9223372036854775808", 0, 1},
	} {
		var value big.Rat
		var workspace big.Rat_Parse_Workspace
		status := big.Rat_Parse(&value, []byte(test.Text), &workspace)
		testify.Equal_Values(t, big.STATUS_OK, status)
		rat_expect(t, &value, test.Numerator, test.Denominator)
	}
	test_rat_parse_invalid(t, rational_workspace)
	test_rat_parse_bounds(t, rational_workspace)
	test_rat_parse_workspace_domains(t)
}

func test_rat_parse_invalid(t *testing.T, rational_workspace *big.Rat_Workspace) {
	for _, source := range [][]byte{
		nil, {}, {'+'}, {'-'}, {'.'}, {'1', '.', '.'}, {'_', '1'}, {'1', '_'},
		{'1', '_', '.', '0'}, {'1', '.', '_', '0'}, {'0', 'b', '2'},
		{'0', 'x', '.', 'p', '1'}, {'1', 'e'}, {'1', 'e', '+'}, {'1', 'e', '_', '1'},
		{'1', 'e', '1', '_'}, {'1', 'e', '1', 'x'},
	} {
		var value big.Rat
		big.Rat_Set_Int_64(&value, 7)
		unchanged := value
		var workspace big.Rat_Parse_Workspace
		status := big.Rat_Parse(&value, source, &workspace)
		testify.Equal_Values(t, big.STATUS_INPUT_INVALID, status)
		testify.Equal(t, big.ORDER_SAME, big.Rat_Compare(
			&value, &unchanged, rational_workspace,
		))
	}
}

func test_rat_parse_bounds(t *testing.T, rational_workspace *big.Rat_Workspace) {
	var overflow [big.RAT_COMPONENT_TEXT_DIGIT_COUNT_MAXIMUM + len("0b1")]byte
	copy(overflow[:], "0b1")
	for count := len("0b1"); count < len(overflow); count++ {
		overflow[count] = '0'
	}
	var value big.Rat
	big.Rat_Set_Int_64(&value, 7)
	unchanged := value
	var workspace big.Rat_Parse_Workspace
	status := big.Rat_Parse(&value, overflow[:], &workspace)
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW, status)
	testify.Equal(t, big.ORDER_SAME, big.Rat_Compare(
		&value, &unchanged, rational_workspace,
	))
	status = big.Rat_Parse(&value, []byte("1e99999"), &workspace)
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW, status)
	testify.Equal(t, big.ORDER_SAME, big.Rat_Compare(
		&value, &unchanged, rational_workspace,
	))
	status = big.Rat_Parse(&value, []byte("1e32768"), &workspace)
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW, status)
	var maximum_words [big.BIT_COUNT_MAXIMUM/big.BASE_HEXADECIMAL_DIGIT_BIT_COUNT +
		len("0x")]byte
	copy(maximum_words[:], "0x8")
	for count := len("0x8"); count < len(maximum_words); count++ {
		maximum_words[count] = '0'
	}
	status = big.Rat_Parse(&value, maximum_words[:], &workspace)
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW, status)
	var mantissa_overflow [len("0x1") +
		big.BIT_COUNT_MAXIMUM/big.BASE_HEXADECIMAL_DIGIT_BIT_COUNT]byte
	copy(mantissa_overflow[:], "0x1")
	for count := len("0x1"); count < len(mantissa_overflow); count++ {
		mantissa_overflow[count] = '0'
	}
	status = big.Rat_Parse(&value, mantissa_overflow[:], &workspace)
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW, status)
	var fractional [big.RAT_PARSE_TEXT_SIZE_MAXIMUM]byte
	fractional[0] = '.'
	for count := len("."); count < len(fractional); count++ {
		fractional[count] = '0'
	}
	status = big.Rat_Parse(&value, fractional[:], &workspace)
	testify.Equal_Values(t, big.STATUS_OK, status)
	rat_expect(t, &value, 0, 1)
	var exponent [big.RAT_PARSE_TEXT_SIZE_MAXIMUM]byte
	copy(exponent[:], "0e")
	for count := len("0e"); count < len(exponent); count++ {
		exponent[count] = '0'
	}
	status = big.Rat_Parse(&value, exponent[:], &workspace)
	testify.Equal_Values(t, big.STATUS_OK, status)
	rat_expect(t, &value, 0, 1)
	test_rat_parse_input_sizes(t, &value, &workspace)
}

func test_rat_parse_input_sizes(
	t *testing.T, value *big.Rat, workspace *big.Rat_Parse_Workspace,
) {
	var source [big.RAT_PARSE_TEXT_UNVALIDATED_SIZE_MAXIMUM]byte
	for count := range source {
		source[count] = '0'
	}
	for _, size := range []int{
		0, 1, 2, big.RAT_PARSE_TEXT_SIZE_MAXIMUM,
		big.RAT_PARSE_TEXT_UNVALIDATED_SIZE_MAXIMUM,
	} {
		status := big.Rat_Parse(value, source[:size], workspace)
		if size == 0 {
			testify.Equal_Values(t, big.STATUS_INPUT_INVALID, status)
			continue
		}
		if size > big.RAT_PARSE_TEXT_SIZE_MAXIMUM {
			testify.Equal_Values(t, big.STATUS_INPUT_INVALID, status)
			continue
		}
		testify.Equal_Values(t, big.STATUS_OK, status)
		rat_expect(t, value, 0, 1)
	}
}

func test_rat_parse_workspace_domains(t *testing.T) {
	var value big.Rat
	var workspace big.Rat_Parse_Workspace
	for _, count := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		workspace.Rational.Greatest_Common.Division.Quotient_Count =
			big.Quotient_Count(count)
		workspace.Rational.Greatest_Common.Division.Remainder_Count =
			big.Remainder_Count(count)
		status := big.Rat_Parse(&value, []byte("1.5"), &workspace)
		testify.Equal_Values(t, big.STATUS_OK, status)
	}
}

func test_rat_fraction_64_inverse(t *testing.T, workspace *big.Rat_Workspace) {
	var value big.Rat
	testify.Equal_Values(t, big.STATUS_OK,
		big.Rat_Set_Fraction_64(&value, 6, -8, workspace))
	rat_expect(t, &value, -3, 4)
	unchanged := value
	testify.Equal_Values(t, big.STATUS_DIVISOR_ZERO,
		big.Rat_Set_Fraction_64(&value, 1, 0, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Rat_Compare(&value, &unchanged, workspace))
	testify.Equal_Values(t, big.STATUS_OK, big.Rat_Inverse(&value, &value, workspace))
	rat_expect(t, &value, -4, 3)
	var zero big.Rat
	unchanged = value
	testify.Equal_Values(t, big.STATUS_DIVISOR_ZERO,
		big.Rat_Inverse(&value, &zero, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Rat_Compare(&value, &unchanged, workspace))

	numbers := [...]big.Int_64{
		big.Int_64(bits.INTEGER_64_MINIMUM), -1, 0, 1, 2,
		big.Int_64(bits.INTEGER_64_MAXIMUM),
	}
	for index := range numbers {
		denominator := numbers[(index+INDEX_STEP)%len(numbers)]
		big.Rat_Set_Fraction_64(&value, numbers[index], denominator, workspace)
	}
	source := rat_set_pair(t, 2, 3, workspace)
	for _, count := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		rat_workspace_count_set(workspace, count)
		big.Rat_Set_Fraction_64(&value, 1, 2, workspace)
		rat_workspace_count_set(workspace, count)
		big.Rat_Inverse(&value, &source, workspace)
	}
}

func test_rat_arithmetic(t *testing.T, workspace *big.Rat_Workspace) {
	left := rat_set_pair(t, 1, 2, workspace)
	right := rat_set_pair(t, 1, 3, workspace)
	var result big.Rat
	testify.Equal_Values(t, big.STATUS_OK, big.Rat_Add(&result, &left, &right, workspace))
	rat_expect(t, &result, 5, 6)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Rat_Subtract(&result, &left, &right, workspace))
	rat_expect(t, &result, 1, 6)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Rat_Multiply(&result, &left, &right, workspace))
	rat_expect(t, &result, 1, 6)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Rat_Quotient(&left, &left, &right, workspace))
	rat_expect(t, &left, 3, 2)
	testify.Equal(t, big.ORDER_AFTER, big.Rat_Compare(&left, &right, workspace))
	zero := rat_set_pair(t, 0, 1, workspace)
	unchanged := result
	testify.Equal_Values(t, big.STATUS_DIVISOR_ZERO,
		big.Rat_Quotient(&result, &left, &zero, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Rat_Compare(&result, &unchanged, workspace))
}

func rat_set_pair(
	t *testing.T, numerator_value int64, denominator_value int64, workspace *big.Rat_Workspace,
) (value big.Rat) {
	var numerator big.Int
	var denominator big.Int
	big.Int_Set_Int_64(&numerator, big.Int_64(numerator_value))
	big.Int_Set_Int_64(&denominator, big.Int_64(denominator_value))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Rat_Set_Fraction(&value, &numerator, &denominator, workspace))
	return value
}

func rat_expect(t *testing.T, value *big.Rat, numerator_want int64, denominator_want int64) {
	var numerator big.Int
	var denominator big.Int
	big.Rat_Numerator_Into(&numerator, value)
	big.Rat_Denominator_Into(&denominator, value)
	test_bitwise_result(t, &numerator, numerator_want)
	test_bitwise_result(t, &denominator, denominator_want)
}

func test_rat_bounds(t *testing.T, workspace *big.Rat_Workspace) {
	testify.Equal(t, big.WORD_COUNT_MAXIMUM/RAT_TEST_COMPONENT_COUNT-INDEX_STEP,
		big.RAT_WORD_COUNT_MAXIMUM)
	var maximum_bytes [bytes.SLICE_SIZE_MAXIMUM]byte
	for index := range maximum_bytes {
		maximum_bytes[index] = byte(bits.WORD_8_MAXIMUM)
	}
	var maximum big.Int
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Bytes(&maximum, maximum_bytes[:]))
	var zero big.Int
	value := rat_set_pair(t, 1, 2, workspace)
	unchanged := value
	testify.Equal_Values(t, big.STATUS_DIVISOR_ZERO,
		big.Rat_Set_Fraction(&value, &maximum, &zero, workspace))
	counts := [...]int{0, 1, 2, big.WORD_COUNT_MAXIMUM}
	for _, count := range counts {
		workspace.Greatest_Common.Division.Quotient_Count = big.Quotient_Count(count)
		workspace.Greatest_Common.Division.Remainder_Count = big.Remainder_Count(count)
		testify.Equal_Values(t, big.STATUS_DIVISOR_ZERO,
			big.Rat_Set_Fraction(&value, &maximum, &zero, workspace))
	}
	testify.Equal_Values(t, big.STATUS_OK,
		big.Rat_Set_Fraction(&value, &zero, &maximum, workspace))
	rat_expect(t, &value, 0, 1)
	value = unchanged
	var one big.Int
	big.Int_Set_Uint_64(&one, 1)
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW,
		big.Rat_Set_Fraction(&value, &maximum, &one, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Rat_Compare(&value, &unchanged, workspace))
}

const RAT_TEST_COMPONENT_COUNT = 2

func test_rat_overflow(t *testing.T, workspace *big.Rat_Workspace) {
	maximum, reciprocal := rat_maximum_component(t, workspace)
	negative := maximum
	big.Rat_Negate(&negative, &negative)
	result := rat_set_pair(t, 1, 2, workspace)
	unchanged := result
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW,
		big.Rat_Add(&result, &maximum, &maximum, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Rat_Compare(&result, &unchanged, workspace))
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW,
		big.Rat_Subtract(&result, &maximum, &negative, workspace))
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW,
		big.Rat_Multiply(&result, &maximum, &maximum, workspace))
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW,
		big.Rat_Quotient(&result, &maximum, &reciprocal, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Rat_Compare(&result, &unchanged, workspace))
}

func rat_maximum_component(
	t *testing.T, workspace *big.Rat_Workspace,
) (maximum big.Rat, reciprocal big.Rat) {
	var component_bytes [big.RAT_WORD_COUNT_MAXIMUM * big.WORD_BYTE_COUNT]byte
	for index := range component_bytes {
		component_bytes[index] = byte(bits.WORD_8_MAXIMUM)
	}
	var component big.Int
	var one big.Int
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Bytes(&component, component_bytes[:]))
	big.Int_Set_Uint_64(&one, 1)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Rat_Set_Fraction(&maximum, &component, &one, workspace))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Rat_Set_Fraction(&reciprocal, &one, &component, workspace))
	return maximum, reciprocal
}

func exercise_rat_workspace_domains(t *testing.T, workspace *big.Rat_Workspace) {
	left := rat_set_pair(t, 2, 3, workspace)
	right := rat_set_pair(t, 3, 5, workspace)
	var result big.Rat
	counts := [...]int{0, 1, 2, big.WORD_COUNT_MAXIMUM}
	for _, count := range counts {
		rat_workspace_count_set(workspace, count)
		big.Rat_Compare(&left, &right, workspace)
		rat_workspace_count_set(workspace, count)
		big.Rat_Add(&result, &left, &right, workspace)
		rat_workspace_count_set(workspace, count)
		big.Rat_Subtract(&result, &left, &right, workspace)
		rat_workspace_count_set(workspace, count)
		big.Rat_Multiply(&result, &left, &right, workspace)
		rat_workspace_count_set(workspace, count)
		big.Rat_Quotient(&result, &left, &right, workspace)
	}
}

func rat_workspace_count_set(workspace *big.Rat_Workspace, count int) {
	workspace.Greatest_Common.Division.Quotient_Count = big.Quotient_Count(count)
	workspace.Greatest_Common.Division.Remainder_Count = big.Remainder_Count(count)
}

func exercise_rat_int_domains(t *testing.T, workspace *big.Rat_Workspace) {
	values := int_domain_values(t)
	var value big.Rat
	var zero big.Int
	for index := range values {
		big.Rat_Set_Int(&value, &values[index])
		big.Rat_Set_Fraction(&value, &values[index], &zero, workspace)
		big.Rat_Set_Fraction(&value, &zero, &values[index], workspace)
		destination := values[index]
		big.Rat_Numerator_Into(&destination, &value)
		destination = values[index]
		big.Rat_Denominator_Into(&destination, &value)
	}
	for _, number := range []big.Int_64{
		big.Int_64(bits.INTEGER_64_MINIMUM), -1, 0, 1, 2,
		big.Int_64(bits.INTEGER_64_MAXIMUM),
	} {
		big.Rat_Set_Int_64(&value, number)
	}
	for _, number := range []big.Word_64{0, 1, 2, big.Word_64(bits.WORD_64_MAXIMUM)} {
		big.Rat_Set_Uint_64(&value, number)
	}
}

func test_text(t *testing.T) {
	for _, one := range []struct {
		Input  big.Base_Unvalidated
		Want   big.Base
		Status big.Validation_Status
	}{
		{Input: -1, Want: big.BASE_MINIMUM, Status: big.STATUS_INPUT_INVALID},
		{Input: 0, Want: big.BASE_MINIMUM, Status: big.STATUS_INPUT_INVALID},
		{Input: 1, Want: big.BASE_MINIMUM, Status: big.STATUS_INPUT_INVALID},
		{Input: 2, Want: 2, Status: big.STATUS_OK},
		{Input: 10, Want: 10, Status: big.STATUS_OK},
		{Input: 62, Want: 62, Status: big.STATUS_OK},
		{Input: 63, Want: big.BASE_MINIMUM, Status: big.STATUS_INPUT_INVALID},
	} {
		base, status := big.Base_Validate(one.Input)
		testify.Equal_Values(t, one.Status, status)
		testify.Equal(t, one.Want, base)
	}
	var workspace big.Int_Text_Workspace
	text_expect(t, 0, 10, "0", &workspace)
	text_expect(t, 0, 2, "0", &workspace)
	text_expect(t, 2, 2, "10", &workspace)
	text_expect(t, 255, 2, "11111111", &workspace)
	text_expect(t, 255, 8, "377", &workspace)
	text_expect(t, 255, 10, "255", &workspace)
	text_expect(t, -255, 16, "-ff", &workspace)
	text_expect(t, 255, 36, "73", &workspace)
	text_expect(t, 61, 62, "Z", &workspace)
	test_text_destination_failure(t, &workspace)
	test_text_maximum(t, &workspace)
	test_float_text_power_formats(t)
	test_float_text_validation(t)
	test_float_text_maximum(t)
	test_parse(t)
}

func test_float_text_power_formats(t *testing.T) {
	var value big.Float
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Set_Float_64_Bits(&value, FLOAT_64_THREE_HALVES_BITS))
	float_text_expect(t, &value, 'b', 0, "6755399441055744p-52")
	float_text_expect(t, &value, 'p', 0, "0x.cp+1")
	float_text_expect(t, &value, 'x', 2, "0x1.80p+00")
	float_text_expect(t, &value, 'x', -1, "0x1.8p+00")
	float_text_expect(t, &value, 'x', 0, "0x1p+01")
	big.Float_Negate(&value, &value)
	float_text_expect(t, &value, 'p', 0, "-0x.cp+1")
	big.Float_Set_Infinity(&value, false)
	float_text_expect(t, &value, 'x', 2, "+Inf")
	big.Float_Set_Infinity(&value, true)
	float_text_expect(t, &value, 'b', 0, "-Inf")
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Set_Float_64_Bits(&value, FLOAT_64_NEGATIVE_ZERO_BITS))
	float_text_expect(t, &value, 'b', 0, "-0")
	float_text_expect(t, &value, 'p', 0, "-0")
	float_text_expect(t, &value, 'x', 2, "-0x0.00p+00")
	test_float_text_failure(t, &value)
	test_float_text_decimal_formats(t)
}

func test_float_text_decimal_formats(t *testing.T) {
	var value big.Float
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Set_Float_64_Bits(&value, FLOAT_64_THREE_HALVES_BITS))
	float_text_expect(t, &value, 'e', 2, "1.50e+00")
	float_text_expect(t, &value, 'E', 2, "1.50E+00")
	float_text_expect(t, &value, 'f', 2, "1.50")
	float_text_expect(t, &value, 'g', 2, "1.5")
	float_text_expect(t, &value, 'G', 2, "1.5")
	float_text_expect(t, &value, 'f', 0, "2")
	big.Float_Set_Int_64(&value, 123456)
	float_text_expect(t, &value, 'g', 4, "1.235e+05")
	float_text_expect(t, &value, 'G', 4, "1.235E+05")
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Set_Float_64_Bits(&value, FLOAT_64_FIVE_QUARTERS_BITS))
	float_text_expect(t, &value, 'f', 1, "1.2")
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Set_Precision(&value, big.FLOAT_64_VALUE_MANTISSA_BIT_COUNT))
	var numerator big.Float
	var denominator big.Float
	big.Float_Set_Uint_64(&numerator, 1)
	big.Float_Set_Uint_64(&denominator, 10)
	var division big.Float_Division_Workspace
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Quotient(&value, &numerator, &denominator, &division))
	float_text_expect(t, &value, 'e', -1, "1e-01")
	float_text_expect(t, &value, 'f', -1, "0.1")
	float_text_expect(t, &value, 'g', -1, "0.1")
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Set_Float_64_Bits(&value, FLOAT_64_NEGATIVE_ZERO_BITS))
	float_text_expect(t, &value, 'f', 2, "-0.00")
	test_float_text_smallest_subnormal(t)
	test_float_text_large_power(t)
}

func test_float_text_smallest_subnormal(t *testing.T) {
	var value big.Float
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Set_Precision(&value, big.Float_Precision_Unvalidated(INDEX_STEP)))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Set_Float_64_Bits(&value, FLOAT_64_SMALLEST_POSITIVE_BITS))
	float_text_expect(t, &value, 'g', -INDEX_STEP, "5e-324")
}

func test_float_text_large_power(t *testing.T) {
	var mantissa big.Float
	big.Float_Set_Uint_64(&mantissa, 1)
	var value big.Float
	testify.Equal_Values(t, big.STATUS_OK, big.Float_Set_Mantissa_Exponent(
		&value, &mantissa,
		big.Float_Exponent_Unvalidated(big.BASE_DECIMAL*big.BASE_DECIMAL),
	))
	float_text_expect(t, &value, 'f', 0, "1267650600228229401496703205376")
}

func float_text_expect(
	t *testing.T, value *big.Float, format byte, precision int, expected string,
) {
	var storage [bytes.SLICE_SIZE_MAXIMUM]byte
	var workspace big.Float_Text_Workspace
	count, status := big.Float_Text_Into(
		storage[:], value, big.Float_Text_Format_Unvalidated(format),
		big.Float_Text_Precision_Unvalidated(precision), &workspace,
	)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, big.Float_Text_Count(len(expected)), count)
	testify.Equal(t, expected, string(storage[:count]))
}

func test_float_text_failure(t *testing.T, value *big.Float) {
	var storage [bytes.SLICE_SIZE_MAXIMUM]byte
	for index := range storage {
		storage[index] = 'x'
	}
	unchanged := storage
	var workspace big.Float_Text_Workspace
	count, status := big.Float_Text_Into(
		storage[:1], value, 'x', 2, &workspace,
	)
	testify.Equal_Values(t, big.STATUS_DESTINATION_TOO_SMALL, status)
	testify.True(t, count > big.Float_Text_Count(len(storage[:1])))
	testify.Equal(t, unchanged, storage)
	count, status = big.Float_Text_Into(storage[:], value, '?', 2, &workspace)
	testify.Equal_Values(t, big.STATUS_INPUT_INVALID, status)
	testify.Equal(t, big.Float_Text_Count(0), count)
	testify.Equal(t, unchanged, storage)
	count, status = big.Float_Text_Into(storage[:], value, 'x',
		big.FLOAT_TEXT_PRECISION_UNVALIDATED_MAXIMUM, &workspace)
	testify.Equal_Values(t, big.STATUS_INPUT_INVALID, status)
	testify.Equal(t, big.Float_Text_Count(0), count)
	testify.Equal(t, unchanged, storage)
}

func test_float_text_validation(t *testing.T) {
	for _, format := range []big.Float_Text_Format_Unvalidated{
		0, 1, 2, big.Float_Text_Format_Unvalidated(bits.WORD_8_MAXIMUM),
	} {
		_, status := big.Float_Text_Format_Validate(format)
		testify.Equal_Values(t, big.STATUS_INPUT_INVALID, status)
	}
	for _, one := range []struct {
		Input  big.Float_Text_Precision_Unvalidated
		Want   big.Float_Text_Precision
		Status big.Validation_Status
	}{
		{Input: big.FLOAT_TEXT_PRECISION_UNVALIDATED_MINIMUM,
			Want: big.FLOAT_TEXT_PRECISION_MINIMUM, Status: big.STATUS_INPUT_INVALID},
		{Input: big.FLOAT_TEXT_PRECISION_MINIMUM,
			Want: big.FLOAT_TEXT_PRECISION_MINIMUM, Status: big.STATUS_OK},
		{Input: 0, Want: 0, Status: big.STATUS_OK},
		{Input: 1, Want: 1, Status: big.STATUS_OK},
		{Input: 2, Want: 2, Status: big.STATUS_OK},
		{Input: big.FLOAT_TEXT_PRECISION_MAXIMUM,
			Want: big.FLOAT_TEXT_PRECISION_MAXIMUM, Status: big.STATUS_OK},
		{Input: big.FLOAT_TEXT_PRECISION_UNVALIDATED_MAXIMUM,
			Want: big.FLOAT_TEXT_PRECISION_MINIMUM, Status: big.STATUS_INPUT_INVALID},
	} {
		precision, status := big.Float_Text_Precision_Validate(one.Input)
		testify.Equal(t, one.Want, precision)
		testify.Equal_Values(t, one.Status, status)
	}
}

func test_float_text_maximum(t *testing.T) {
	integers := int_domain_values(t)
	var value big.Float
	big.Float_Set_Int(&value, &integers[POSITIVE_MAXIMUM_WORD_INDEX])
	big.Float_Negate(&value, &value)
	var destination [big.INT_TEXT_SIZE_MAXIMUM]byte
	var workspace big.Float_Text_Workspace
	count, status := big.Float_Text_Into(
		destination[:], &value, 'f', big.FLOAT_TEXT_PRECISION_MAXIMUM,
		&workspace,
	)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, big.Float_Text_Count(big.FLOAT_TEXT_SIZE_MAXIMUM), count)
	testify.Equal(t, byte('-'), destination[0])
	testify.Equal(t, byte('.'),
		destination[big.SIGN_BYTE_COUNT_MAXIMUM+
			big.FLOAT_TEXT_INTEGER_DIGIT_COUNT_MAXIMUM])
	testify.Equal(t, byte('0'), destination[int(count)-INDEX_STEP])
}

func test_serialization(t *testing.T) {
	test_float_serialization(t)
	test_int_serialization(t)
	test_rat_serialization(t)
}

func test_float_serialization(t *testing.T) {
	var value big.Float
	big.Float_Set_Int_64(&value, 1)
	var workspace big.Float_Gob_Workspace
	var storage [big.FLOAT_GOB_SIZE_MAXIMUM]byte
	mantissa_count, status := big.Float_Gob_Encode_Into(
		storage[:], &value, &workspace,
	)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, big.Word_Count(1), mantissa_count)
	size := big.FLOAT_GOB_FINITE_PREFIX_SIZE + int(mantissa_count)*big.WORD_BYTE_COUNT
	var expected [big.FLOAT_GOB_FINITE_PREFIX_SIZE + big.WORD_BYTE_COUNT]byte
	expected[big.FLOAT_GOB_VERSION_OFFSET] = byte(big.FLOAT_GOB_VERSION)
	expected[big.FLOAT_GOB_ATTRIBUTES_OFFSET] = byte(
		uint8(value.Mode)<<big.FLOAT_GOB_MODE_SHIFT |
			uint8(int8(value.Accuracy)+1)<<big.FLOAT_GOB_ACCURACY_SHIFT |
			uint8(value.Form)<<big.FLOAT_GOB_FORM_SHIFT,
	)
	binary.Put_Uint_32(
		binary.Bytes(expected[big.FLOAT_GOB_PRECISION_OFFSET:]),
		binary.Word_32(value.Precision), binary.BIG_ENDIAN,
	)
	binary.Put_Uint_32(
		binary.Bytes(expected[big.FLOAT_GOB_EXPONENT_OFFSET:]),
		binary.Word_32(value.Exponent), binary.BIG_ENDIAN,
	)
	expected[big.FLOAT_GOB_MANTISSA_OFFSET] = byte(
		bits.CARRY_MAXIMUM << (bits.BIT_COUNT_8_MAXIMUM - INDEX_STEP),
	)
	testify.Equal(t, expected[:], storage[:size])
	var decoded big.Float
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Gob_Decode(&decoded, storage[:size], &workspace))
	testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&value, &decoded))
	testify.Equal(t, value.Precision, decoded.Precision)
	testify.Equal(t, value.Mode, decoded.Mode)
	testify.Equal(t, value.Accuracy, decoded.Accuracy)
	test_float_serialization_bounds(t, &value, &workspace, storage[:size])
}

func test_float_serialization_bounds(
	t *testing.T,
	value *big.Float,
	workspace *big.Float_Gob_Workspace,
	encoding []byte,
) {
	for _, size := range []int{0, 1, 2, len(encoding) - INDEX_STEP, len(encoding)} {
		var destination [big.FLOAT_GOB_SIZE_MAXIMUM]byte
		for index := range destination {
			destination[index] = 'x'
		}
		unchanged := destination
		count, status := big.Float_Gob_Encode_Into(
			destination[:size], value, workspace,
		)
		testify.Equal(t, big.Word_Count(1), count)
		if size < len(encoding) {
			testify.Equal_Values(t, big.STATUS_DESTINATION_TOO_SMALL, status)
			testify.Equal(t, unchanged, destination)
		} else {
			testify.Equal_Values(t, big.STATUS_OK, status)
		}
	}
	test_float_serialization_decode_invalid(t, value, workspace, encoding)
}

func test_float_serialization_decode_invalid(
	t *testing.T,
	value *big.Float,
	workspace *big.Float_Gob_Workspace,
	encoding []byte,
) {
	unchanged := *value
	for _, source := range [][]byte{
		{big.FLOAT_GOB_VERSION},
		{big.FLOAT_GOB_VERSION, 0},
		{big.FLOAT_GOB_VERSION + INDEX_STEP, 0, 0, 0, 0, 0},
	} {
		testify.Equal_Values(t, big.STATUS_INPUT_INVALID,
			big.Float_Gob_Decode(value, source, workspace))
		testify.Equal(t, unchanged, *value)
	}
	for _, size := range []int{
		big.FLOAT_GOB_HEADER_SIZE,
		big.FLOAT_GOB_MANTISSA_OFFSET - INDEX_STEP,
		big.FLOAT_GOB_MANTISSA_OFFSET,
		big.FLOAT_GOB_MANTISSA_OFFSET + INDEX_STEP,
		big.FLOAT_GOB_MANTISSA_OFFSET + INDEX_STEP + INDEX_STEP,
	} {
		test_float_gob_decode_invalid_case(
			t, value, workspace, encoding[:size], unchanged,
		)
	}
	var invalid [big.FLOAT_GOB_FINITE_PREFIX_SIZE + big.WORD_BYTE_COUNT]byte
	copy(invalid[:], encoding)
	invalid[big.FLOAT_GOB_MANTISSA_OFFSET] = 0
	test_float_gob_decode_invalid_case(t, value, workspace, invalid[:], unchanged)
	copy(invalid[:], encoding)
	test_float_gob_decode_invalid_case(
		t, value, workspace, invalid[:len(invalid)-INDEX_STEP], unchanged,
	)
	test_float_gob_decode_invalid_header(t, value, workspace, encoding, unchanged)
	copy(invalid[:], encoding)
	binary.Put_Uint_32(
		binary.Bytes(invalid[big.FLOAT_GOB_EXPONENT_OFFSET:]),
		binary.Word_32(big.FLOAT_EXPONENT_UNVALIDATED_MAXIMUM), binary.BIG_ENDIAN,
	)
	test_float_gob_decode_invalid_case(t, value, workspace, invalid[:], unchanged)
	copy(invalid[:], encoding)
	binary.Put_Uint_32(
		binary.Bytes(invalid[big.FLOAT_GOB_PRECISION_OFFSET:]),
		binary.Word_32(INDEX_STEP), binary.BIG_ENDIAN,
	)
	high_bit := bits.BIT_COUNT_8_MAXIMUM - INDEX_STEP
	invalid[big.FLOAT_GOB_MANTISSA_OFFSET] = byte(bits.CARRY_MAXIMUM)<<high_bit |
		byte(bits.CARRY_MAXIMUM)<<(high_bit-INDEX_STEP)
	test_float_gob_decode_invalid_case(t, value, workspace, invalid[:], unchanged)
	var oversized [big.FLOAT_GOB_UNVALIDATED_SIZE_MAXIMUM]byte
	testify.Equal_Values(t, big.STATUS_INPUT_INVALID,
		big.Float_Gob_Decode(value, oversized[:], workspace))
	testify.Equal(t, unchanged, *value)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Gob_Decode(value, nil, workspace))
	testify.Equal(t, big.SIGN_ZERO, big.Float_Sign(value))
}

func test_float_gob_decode_invalid_header(
	t *testing.T,
	value *big.Float,
	workspace *big.Float_Gob_Workspace,
	encoding []byte,
	unchanged big.Float,
) {
	var invalid [big.FLOAT_GOB_FINITE_PREFIX_SIZE + big.WORD_BYTE_COUNT]byte
	copy(invalid[:], encoding)
	mode_mask := big.FLOAT_GOB_MODE_MASK << big.FLOAT_GOB_MODE_SHIFT
	invalid[big.FLOAT_GOB_ATTRIBUTES_OFFSET] &^= mode_mask
	invalid[big.FLOAT_GOB_ATTRIBUTES_OFFSET] |=
		byte(big.ROUNDING_MODE_UNVALIDATED_MAXIMUM) << big.FLOAT_GOB_MODE_SHIFT
	test_float_gob_decode_invalid_case(t, value, workspace, invalid[:], unchanged)
	copy(invalid[:], encoding)
	accuracy_mask := big.FLOAT_GOB_ACCURACY_MASK << big.FLOAT_GOB_ACCURACY_SHIFT
	invalid[big.FLOAT_GOB_ATTRIBUTES_OFFSET] &^= accuracy_mask
	invalid[big.FLOAT_GOB_ATTRIBUTES_OFFSET] |= accuracy_mask
	test_float_gob_decode_invalid_case(t, value, workspace, invalid[:], unchanged)
	copy(invalid[:], encoding)
	form_mask := big.FLOAT_GOB_FORM_MASK << big.FLOAT_GOB_FORM_SHIFT
	invalid[big.FLOAT_GOB_ATTRIBUTES_OFFSET] &^= form_mask
	invalid[big.FLOAT_GOB_ATTRIBUTES_OFFSET] |= form_mask
	test_float_gob_decode_invalid_case(t, value, workspace, invalid[:], unchanged)
	copy(invalid[:], encoding)
	binary.Put_Uint_32(
		binary.Bytes(invalid[big.FLOAT_GOB_PRECISION_OFFSET:]),
		binary.Word_32(big.FLOAT_PRECISION_UNVALIDATED_MAXIMUM), binary.BIG_ENDIAN,
	)
	test_float_gob_decode_invalid_case(t, value, workspace, invalid[:], unchanged)
	copy(invalid[:], encoding)
	binary.Put_Uint_32(
		binary.Bytes(invalid[big.FLOAT_GOB_PRECISION_OFFSET:]),
		binary.Word_32(big.FLOAT_PRECISION_MINIMUM), binary.BIG_ENDIAN,
	)
	test_float_gob_decode_invalid_case(t, value, workspace, invalid[:], unchanged)
}

func test_float_gob_decode_invalid_case(
	t *testing.T,
	value *big.Float,
	workspace *big.Float_Gob_Workspace,
	source []byte,
	unchanged big.Float,
) {
	testify.Equal_Values(t, big.STATUS_INPUT_INVALID,
		big.Float_Gob_Decode(value, source, workspace))
	testify.Equal(t, unchanged, *value)
}

func test_int_serialization(t *testing.T) {
	for _, one := range []struct {
		Value int64
		Want  []byte
	}{
		{Value: 0, Want: []byte{2}},
		{Value: 1, Want: []byte{2, 1}},
		{Value: -255, Want: []byte{3, 255}},
	} {
		var value big.Int
		big.Int_Set_Int_64(&value, big.Int_64(one.Value))
		var storage [big.INT_GOB_SIZE_MAXIMUM]byte
		count, status := big.Int_Gob_Encode_Into(storage[:], &value)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, one.Want, storage[:count])
		var decoded big.Int
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Gob_Decode(&decoded, storage[:count]))
		testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&value, &decoded))
	}
	test_serialization_bounds(t)
}

func test_rat_serialization(t *testing.T) {
	for _, one := range []struct {
		Numerator   int64
		Denominator int64
		Want        []byte
	}{
		{Numerator: 0, Denominator: 1, Want: []byte{2, 0, 0, 0, 0}},
		{Numerator: 1, Denominator: 1, Want: []byte{2, 0, 0, 0, 1, 1}},
		{Numerator: 256, Denominator: 1, Want: []byte{2, 0, 0, 0, 2, 1, 0}},
		{Numerator: -3, Denominator: 4, Want: []byte{3, 0, 0, 0, 1, 3, 4}},
	} {
		var value big.Rat
		var rat_workspace big.Rat_Workspace
		testify.Equal_Values(t, big.STATUS_OK, big.Rat_Set_Fraction_64(
			&value, big.Int_64(one.Numerator), big.Int_64(one.Denominator),
			&rat_workspace,
		))
		var storage [big.RAT_GOB_SIZE_MAXIMUM]byte
		count, status := big.Rat_Gob_Encode_Into(storage[:], &value)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, one.Want, storage[:count])
		var decoded big.Rat
		var gob_workspace big.Rat_Gob_Workspace
		testify.Equal_Values(t, big.STATUS_OK, big.Rat_Gob_Decode(
			&decoded, storage[:count], &gob_workspace,
		))
		testify.Equal(t, big.ORDER_SAME,
			big.Rat_Compare(&value, &decoded, &rat_workspace))
	}
	test_rat_serialization_bounds(t)
}

func test_rat_serialization_bounds(t *testing.T) {
	value := maximum_rat(t)
	var storage [big.RAT_GOB_SIZE_MAXIMUM]byte
	count, status := big.Rat_Gob_Encode_Into(storage[:], &value)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, big.Rat_Encoding_Count(big.RAT_GOB_SIZE_MAXIMUM), count)
	for _, size := range []int{0, 1, 2, big.RAT_GOB_PREFIX_SIZE, big.RAT_GOB_SIZE_MAXIMUM} {
		var destination [big.RAT_GOB_SIZE_MAXIMUM]byte
		for index := range destination {
			destination[index] = 'x'
		}
		unchanged := destination
		required, destination_status := big.Rat_Gob_Encode_Into(
			destination[:size], &value,
		)
		testify.Equal(t, count, required)
		if size < big.RAT_GOB_SIZE_MAXIMUM {
			testify.Equal_Values(
				t, big.STATUS_DESTINATION_TOO_SMALL, destination_status,
			)
			testify.Equal(t, unchanged, destination)
		} else {
			testify.Equal_Values(t, big.STATUS_OK, destination_status)
		}
	}
	test_rat_serialization_decode_bounds(t, storage[:count])
}

func test_rat_serialization_decode_bounds(t *testing.T, maximum []byte) {
	var value big.Rat
	big.Rat_Set_Int_64(&value, 7)
	unchanged := value
	var workspace big.Rat_Gob_Workspace
	var compare_workspace big.Rat_Workspace
	for _, source := range [][]byte{{2}, {2, 0}, {4, 0, 0, 0, 0}, {2, 0, 0, 0, 1}} {
		testify.Equal_Values(t, big.STATUS_INPUT_INVALID,
			big.Rat_Gob_Decode(&value, source, &workspace))
		testify.Equal(t, big.ORDER_SAME, big.Rat_Compare(
			&value, &unchanged, &compare_workspace,
		))
	}
	var oversized [big.RAT_GOB_UNVALIDATED_SIZE_MAXIMUM]byte
	testify.Equal_Values(t, big.STATUS_INPUT_INVALID,
		big.Rat_Gob_Decode(&value, oversized[:], &workspace))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Rat_Gob_Decode(&value, nil, &workspace))
	testify.Equal(t, big.SIGN_ZERO, big.Rat_Sign(&value))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Rat_Gob_Decode(&value, maximum, &workspace))
}

func maximum_rat(t *testing.T) (value big.Rat) {
	var numerator_bytes [big.RAT_COMPONENT_BYTE_SIZE_MAXIMUM]byte
	for index := range numerator_bytes {
		numerator_bytes[index] = byte(bits.WORD_8_MAXIMUM)
	}
	denominator_bytes := numerator_bytes
	denominator_bytes[len(denominator_bytes)-INDEX_STEP]--
	var numerator big.Int
	var denominator big.Int
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Set_Bytes(&numerator, numerator_bytes[:]))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Set_Bytes(&denominator, denominator_bytes[:]))
	var workspace big.Rat_Workspace
	testify.Equal_Values(t, big.STATUS_OK,
		big.Rat_Set_Fraction(&value, &numerator, &denominator, &workspace))
	return value
}

func test_serialization_bounds(t *testing.T) {
	var maximum_words [big.WORD_COUNT_MAXIMUM]big.Word
	for index := range maximum_words {
		maximum_words[index] = big.Word(bits.WORD_64_MAXIMUM)
	}
	var maximum big.Int
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Set_Words(&maximum, maximum_words[:]))
	var storage [big.INT_GOB_SIZE_MAXIMUM]byte
	count, status := big.Int_Gob_Encode_Into(storage[:], &maximum)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, big.Int_Encoding_Count(big.INT_GOB_SIZE_MAXIMUM), count)
	for _, size := range []int{0, 1, 2, big.INT_GOB_SIZE_MAXIMUM} {
		var destination [big.INT_GOB_SIZE_MAXIMUM]byte
		for index := range destination {
			destination[index] = 'x'
		}
		unchanged := destination
		written, destination_status := big.Int_Gob_Encode_Into(
			destination[:size], &maximum,
		)
		if size < big.INT_GOB_SIZE_MAXIMUM {
			testify.Equal_Values(
				t, big.STATUS_DESTINATION_TOO_SMALL, destination_status,
			)
			testify.Equal(t, big.Int_Encoding_Count(0), written)
			testify.Equal(t, unchanged, destination)
		} else {
			testify.Equal_Values(t, big.STATUS_OK, destination_status)
		}
	}
	var value big.Int
	big.Int_Set_Int_64(&value, 7)
	unchanged := value
	testify.Equal_Values(t, big.STATUS_INPUT_INVALID,
		big.Int_Gob_Decode(&value, []byte{4}))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&value, &unchanged))
	var oversized [big.INT_GOB_SIZE_MAXIMUM + INDEX_STEP]byte
	testify.Equal_Values(t, big.STATUS_INPUT_INVALID,
		big.Int_Gob_Decode(&value, oversized[:]))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&value, &unchanged))
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Gob_Decode(&value, nil))
	testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&value))
	for _, source := range [][]byte{{2}, {2, 1}, storage[:count]} {
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Gob_Decode(&value, source))
	}
	exercise_serialization_domains(t)
}

func exercise_serialization_domains(t *testing.T) {
	values := int_domain_values(t)
	var storage [big.INT_GOB_SIZE_MAXIMUM]byte
	for index := range values {
		_, status := big.Int_Gob_Encode_Into(storage[:], &values[index])
		testify.Equal_Values(t, big.STATUS_OK, status)
		destination := values[index]
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Gob_Decode(&destination, []byte{2}))
	}
}

func text_expect(
	t *testing.T, value int64, base_value big.Base_Unvalidated, want string,
	workspace *big.Int_Text_Workspace,
) {
	base, status := big.Base_Validate(base_value)
	testify.Equal_Values(t, big.STATUS_OK, status)
	var integer big.Int
	big.Int_Set_Int_64(&integer, big.Int_64(value))
	var destination [big.WORD_BIT_COUNT]byte
	count, destination_status := big.Int_Text_Into(
		destination[:], &integer, base, workspace,
	)
	testify.Equal_Values(t, big.STATUS_OK, destination_status)
	testify.Equal(t, want, string(destination[:count]))
}

func test_text_destination_failure(t *testing.T, workspace *big.Int_Text_Workspace) {
	base, status := big.Base_Validate(10)
	testify.Equal_Values(t, big.STATUS_OK, status)
	var integer big.Int
	big.Int_Set_Int_64(&integer, -255)
	for _, size := range []int{0, 1, 2, 3} {
		destination := [...]byte{'x', 'x', 'x'}
		unchanged := destination
		count, destination_status := big.Int_Text_Into(
			destination[:size], &integer, base, workspace,
		)
		testify.Equal(t, big.Text_Count(0), count)
		testify.Equal_Values(t, big.STATUS_DESTINATION_TOO_SMALL, destination_status)
		testify.Equal(t, unchanged, destination)
	}
}

func test_text_maximum(t *testing.T, workspace *big.Int_Text_Workspace) {
	var two_word_bytes [big.WORD_BYTE_COUNT + 1]byte
	two_word_bytes[0] = 1
	var two_word big.Int
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Bytes(&two_word, two_word_bytes[:]))
	base, status := big.Base_Validate(2)
	testify.Equal_Values(t, big.STATUS_OK, status)
	var destination [big.INT_TEXT_SIZE_MAXIMUM]byte
	count, destination_status := big.Int_Text_Into(
		destination[:], &two_word, base, workspace,
	)
	testify.Equal_Values(t, big.STATUS_OK, destination_status)
	testify.Equal(t, big.Text_Count(big.WORD_BIT_COUNT+INDEX_STEP), count)
	testify.Equal(t, byte('1'), destination[0])
	testify.Equal(t, byte('0'), destination[int(count)-1])

	var maximum_bytes [bytes.SLICE_SIZE_MAXIMUM]byte
	for index := range maximum_bytes {
		maximum_bytes[index] = byte(bits.WORD_8_MAXIMUM)
	}
	var maximum big.Int
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Bytes(&maximum, maximum_bytes[:]))
	big.Int_Negate(&maximum, &maximum)
	count, destination_status = big.Int_Text_Into(
		destination[:], &maximum, base, workspace,
	)
	testify.Equal_Values(t, big.STATUS_OK, destination_status)
	testify.Equal(t, big.Text_Count(big.INT_TEXT_SIZE_MAXIMUM), count)
	testify.Equal(t, byte('-'), destination[0])
	testify.Equal(t, byte('1'), destination[1])
	testify.Equal(t, byte('1'), destination[int(count)-1])
}

func test_parse(t *testing.T) {
	var workspace big.Int_Parse_Workspace
	for _, one := range []struct {
		Text string
		Base big.Base_Unvalidated
		Used big.Base
		Want int64
	}{
		{Text: "0", Base: 0, Used: 8, Want: 0},
		{Text: "255", Base: 10, Used: 10, Want: 255},
		{Text: "-ff", Base: 16, Used: 16, Want: -255},
		{Text: "+Z", Base: 62, Used: 62, Want: 61},
		{Text: "FF", Base: 16, Used: 16, Want: 255},
		{Text: "0xff", Base: 0, Used: 16, Want: 255},
		{Text: "0b101", Base: 0, Used: 2, Want: 5},
		{Text: "0755", Base: 0, Used: 8, Want: 493},
		{Text: "1_000", Base: 0, Used: 10, Want: 1000},
		{Text: "0x_ff", Base: 0, Used: 16, Want: 255},
	} {
		var value big.Int
		used, status := big.Int_Parse(&value, []byte(one.Text), one.Base, &workspace)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, one.Used, used)
		test_bitwise_result(t, &value, one.Want)
	}
	var two_word_bytes [big.WORD_BYTE_COUNT + INDEX_STEP]byte
	two_word_bytes[ZERO_INDEX] = INDEX_STEP
	var two_word_destination big.Int
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Set_Bytes(&two_word_destination, two_word_bytes[:]))
	big.Int_Negate(&two_word_destination, &two_word_destination)
	used, status := big.Int_Parse(
		&two_word_destination, []byte("-0x1"), 0, &workspace,
	)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, big.Base(16), used)
	test_bitwise_result(t, &two_word_destination, -1)
	test_parse_invalid(t, &workspace)
	test_parse_bounds(t, &workspace)
	test_float_parse(t)
}

func test_float_parse(t *testing.T) {
	for _, one := range []struct {
		Text string
		Base big.Base_Unvalidated
		Used big.Float_Parse_Base
		Want string
	}{
		{Text: "0", Base: 0, Used: big.FLOAT_PARSE_BASE_DECIMAL, Want: "0"},
		{Text: "-0", Base: 0, Used: big.FLOAT_PARSE_BASE_DECIMAL, Want: "-0"},
		{Text: ".5", Base: 0, Used: big.FLOAT_PARSE_BASE_DECIMAL, Want: "0.5"},
		{Text: "1.5", Base: 0, Used: big.FLOAT_PARSE_BASE_DECIMAL, Want: "1.5"},
		{Text: "1e2", Base: 0, Used: big.FLOAT_PARSE_BASE_DECIMAL, Want: "100"},
		{Text: "1e2", Base: 10, Used: big.FLOAT_PARSE_BASE_DECIMAL, Want: "100"},
		{Text: "0755", Base: 0, Used: big.FLOAT_PARSE_BASE_DECIMAL, Want: "755"},
		{Text: "0b1.1p+1", Base: 0, Used: big.FLOAT_PARSE_BASE_BINARY, Want: "3"},
		{Text: "0o1.4p+1", Base: 0, Used: big.FLOAT_PARSE_BASE_OCTAL, Want: "3"},
		{Text: "0x1.8p+1", Base: 0, Used: big.FLOAT_PARSE_BASE_HEXADECIMAL, Want: "3"},
		{Text: "0x1e2", Base: 0, Used: big.FLOAT_PARSE_BASE_HEXADECIMAL, Want: "482"},
		{Text: "1.1p+1", Base: 2, Used: big.FLOAT_PARSE_BASE_BINARY, Want: "3"},
		{Text: "1.4p+1", Base: 8, Used: big.FLOAT_PARSE_BASE_OCTAL, Want: "3"},
		{Text: "0x_1.8p+1_0", Base: 0, Used: big.FLOAT_PARSE_BASE_HEXADECIMAL,
			Want: "1536"},
		{Text: "0e100000", Base: 0, Used: big.FLOAT_PARSE_BASE_DECIMAL, Want: "0"},
		{Text: "Inf", Base: 0, Used: big.FLOAT_PARSE_BASE_AUTOMATIC, Want: "+Inf"},
		{Text: "-inf", Base: 0, Used: big.FLOAT_PARSE_BASE_AUTOMATIC, Want: "-Inf"},
	} {
		var value big.Float
		var workspace big.Float_Parse_Workspace
		used, status := big.Float_Parse(
			&value, big.Float_Parse_Text_Unvalidated(one.Text), one.Base,
			&workspace,
		)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, one.Used, used)
		float_text_expect(t, &value, 'g', -1, one.Want)
	}
	test_float_parse_policy(t)
	test_float_parse_failure(t)
	test_float_parse_bounds(t)
	test_float_parse_exponent_bounds(t)
}

func test_float_parse_policy(t *testing.T) {
	var value big.Float
	testify.Equal_Values(t, big.STATUS_OK, big.Float_Set_Precision(&value, 2))
	var workspace big.Float_Parse_Workspace
	_, status := big.Float_Parse(
		&value, big.Float_Parse_Text_Unvalidated("1.25"), 0, &workspace,
	)
	testify.Equal_Values(t, big.STATUS_OK, status)
	encoding, _ := big.Float_Float_64_Bits(&value)
	testify.Equal(t, big.Float_64_Value_Bits(FLOAT_64_ONE_BITS), encoding)
	testify.Equal(t, big.ACCURACY_BELOW, big.Float_Accuracy(&value))

	value = big.Float{}
	testify.Equal_Values(t, big.STATUS_OK, big.Float_Set_Precision(&value, 2))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Set_Rounding_Mode(&value, big.ROUND_TO_NEAREST_AWAY))
	_, status = big.Float_Parse(
		&value, big.Float_Parse_Text_Unvalidated("1.25"), 0, &workspace,
	)
	testify.Equal_Values(t, big.STATUS_OK, status)
	encoding, _ = big.Float_Float_64_Bits(&value)
	testify.Equal(t, big.Float_64_Value_Bits(FLOAT_64_THREE_HALVES_BITS), encoding)
	testify.Equal(t, big.ACCURACY_ABOVE, big.Float_Accuracy(&value))
}

func test_float_parse_failure(t *testing.T) {
	for _, one := range []struct {
		Text   string
		Base   big.Base_Unvalidated
		Status big.Parse_Status
	}{
		{Text: "", Base: 0, Status: big.STATUS_INPUT_INVALID},
		{Text: "+", Base: 0, Status: big.STATUS_INPUT_INVALID},
		{Text: ".", Base: 0, Status: big.STATUS_INPUT_INVALID},
		{Text: "1_", Base: 0, Status: big.STATUS_INPUT_INVALID},
		{Text: "1_0", Base: 10, Status: big.STATUS_INPUT_INVALID},
		{Text: "0x1", Base: 16, Status: big.STATUS_INPUT_INVALID},
		{Text: "1e", Base: 0, Status: big.STATUS_INPUT_INVALID},
		{Text: "1e+", Base: 0, Status: big.STATUS_INPUT_INVALID},
		{Text: "1e100000", Base: 0, Status: big.STATUS_VALUE_OVERFLOW},
		{Text: "1", Base: -1, Status: big.STATUS_INPUT_INVALID},
		{Text: "1", Base: 1, Status: big.STATUS_INPUT_INVALID},
		{Text: "1", Base: 3, Status: big.STATUS_INPUT_INVALID},
		{Text: "1", Base: 63, Status: big.STATUS_INPUT_INVALID},
	} {
		var value big.Float
		big.Float_Set_Int_64(&value, 7)
		unchanged := value
		var workspace big.Float_Parse_Workspace
		_, status := big.Float_Parse(
			&value, big.Float_Parse_Text_Unvalidated(one.Text), one.Base,
			&workspace,
		)
		testify.Equal_Values(t, one.Status, status)
		testify.Equal(t, unchanged, value)
	}
}

func test_float_parse_bounds(t *testing.T) {
	var maximum [big.FLOAT_PARSE_TEXT_SIZE_MAXIMUM]byte
	for index := range maximum {
		maximum[index] = '0'
	}
	maximum[0] = '1'
	var value big.Float
	big.Float_Set_Int_64(&value, 7)
	unchanged := value
	var workspace big.Float_Parse_Workspace
	_, status := big.Float_Parse(&value, maximum[:], 2, &workspace)
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW, status)
	testify.Equal(t, unchanged, value)

	var oversized [big.FLOAT_PARSE_TEXT_UNVALIDATED_SIZE_MAXIMUM]byte
	_, status = big.Float_Parse(&value, oversized[:], 0, &workspace)
	testify.Equal_Values(t, big.STATUS_INPUT_INVALID, status)
	testify.Equal(t, unchanged, value)
}

func test_float_parse_exponent_bounds(t *testing.T) {
	for _, one := range []struct {
		Text string
		Want big.Float_Exponent
	}{
		// Parsed one has normalized exponent one, so source shifts differ by one.
		{Text: "1p32767", Want: big.FLOAT_EXPONENT_MAXIMUM},
		{Text: "1p-32769", Want: big.FLOAT_EXPONENT_MINIMUM},
	} {
		var value, mantissa big.Float
		var workspace big.Float_Parse_Workspace
		used, status := big.Float_Parse(
			&value, big.Float_Parse_Text_Unvalidated(one.Text), 0, &workspace,
		)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, big.FLOAT_PARSE_BASE_DECIMAL, used)
		testify.Equal(t, one.Want, big.Float_Mantissa_Exponent(&value, &mantissa))
	}
	for _, text := range []string{"1p32768", "1p-32770"} {
		var value big.Float
		big.Float_Set_Int_64(&value, 7)
		unchanged := value
		var workspace big.Float_Parse_Workspace
		_, status := big.Float_Parse(
			&value, big.Float_Parse_Text_Unvalidated(text), 0, &workspace,
		)
		testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW, status)
		testify.Equal(t, unchanged, value)
	}
}

func test_parse_invalid(t *testing.T, workspace *big.Int_Parse_Workspace) {
	for _, one := range []struct {
		Text string
		Base big.Base_Unvalidated
	}{
		{Text: "", Base: 10},
		{Text: "+", Base: 10},
		{Text: "-", Base: 10},
		{Text: "1", Base: -1},
		{Text: "1", Base: 1},
		{Text: "1", Base: 63},
		{Text: "_1", Base: 0},
		{Text: "1_", Base: 0},
		{Text: "1__0", Base: 0},
		{Text: "0x", Base: 0},
		{Text: "0x_", Base: 0},
		{Text: "1_0", Base: 10},
		{Text: "2", Base: 2},
		{Text: "\x00", Base: 10},
		{Text: "\x01", Base: 10},
		{Text: "\x02", Base: 10},
		{Text: "\xff", Base: 10},
	} {
		var value big.Int
		big.Int_Set_Int_64(&value, 7)
		unchanged := value
		_, status := big.Int_Parse(&value, []byte(one.Text), one.Base, workspace)
		testify.Equal_Values(t, big.STATUS_INPUT_INVALID, status)
		testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&value, &unchanged))
	}
}

func test_parse_bounds(t *testing.T, workspace *big.Int_Parse_Workspace) {
	var maximum_text [big.BIT_COUNT_MAXIMUM]byte
	for index := range maximum_text {
		maximum_text[index] = '1'
	}
	var maximum big.Int
	used, status := big.Int_Parse(&maximum, maximum_text[:], 2, workspace)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, big.Base(2), used)
	testify.Equal(t, big.Bit_Count(big.BIT_COUNT_MAXIMUM), big.Int_Bit_Count(&maximum))

	var overflow_text [big.INT_TEXT_SIZE_MAXIMUM]byte
	overflow_text[0] = '1'
	for index := 1; index < len(overflow_text); index++ {
		overflow_text[index] = '0'
	}
	unchanged := maximum
	_, status = big.Int_Parse(&maximum, overflow_text[:], 2, workspace)
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW, status)
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&maximum, &unchanged))

	var oversized [big.INT_TEXT_SIZE_MAXIMUM + 1]byte
	_, status = big.Int_Parse(&maximum, oversized[:], 10, workspace)
	testify.Equal_Values(t, big.STATUS_INPUT_INVALID, status)
}

func exercise_greatest_common_workspace_domains(
	destination *big.Int,
	zero *big.Int,
	workspace *big.Int_Greatest_Common_Divisor_Workspace,
) {
	counts := [...]int{0, 1, 2, big.WORD_COUNT_MAXIMUM}
	for _, count := range counts {
		workspace.Division.Quotient_Count = big.Quotient_Count(count)
		workspace.Division.Remainder_Count = big.Remainder_Count(count)
		big.Int_Greatest_Common_Divisor(destination, zero, zero, workspace)
	}
}

func test_bounds(t *testing.T) {
	testify.Equal(t, bytes.SLICE_SIZE_MAXIMUM/big.WORD_BYTE_COUNT,
		big.WORD_COUNT_MAXIMUM)
	testify.Equal(t, big.WORD_COUNT_MAXIMUM*big.WORD_BIT_COUNT,
		big.BIT_COUNT_MAXIMUM)

	var maximum_bytes [bytes.SLICE_SIZE_MAXIMUM]byte
	for index := range maximum_bytes {
		maximum_bytes[index] = byte(bits.WORD_8_MAXIMUM)
	}
	var maximum big.Int
	status := big.Int_Set_Bytes(&maximum, maximum_bytes[:])
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, big.Bit_Count(big.BIT_COUNT_MAXIMUM),
		big.Int_Bit_Count(&maximum))
	unchanged := maximum
	var one big.Int
	big.Int_Set_Uint_64(&one, 1)
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW,
		big.Int_Add(&maximum, &maximum, &one))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&maximum, &unchanged))

	var oversized [bytes.SLICE_SIZE_MAXIMUM + 1]byte
	status = big.Int_Set_Bytes(&maximum, oversized[:])
	testify.Equal_Values(t, big.STATUS_INPUT_INVALID, status)
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&maximum, &unchanged))
	_, status = big.Bytes_Validate(oversized[:])
	testify.Equal_Values(t, big.STATUS_INPUT_INVALID, status)

	var negative_one big.Int
	big.Int_Set_Int_64(&negative_one, -1)
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW,
		big.Int_Subtract(&maximum, &maximum, &negative_one))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&maximum, &unchanged))

	values := int_domain_values(t)
	exercise_int_domains(t, &values)
	exercise_float_domains(t, &values)
}

const INDEX_STEP = 1

const FLOAT_64_EXPONENT_SHIFT = big.FLOAT_64_EXPONENT_SHIFT

const FLOAT_32_EXPONENT_SHIFT = big.FLOAT_32_EXPONENT_SHIFT

const FLOAT_32_EXPONENT_BIAS = big.FLOAT_32_EXPONENT_BIAS

const FLOAT_32_EXPONENT_MASK = big.FLOAT_32_EXPONENT_MASK

const FLOAT_32_MANTISSA_BIT_COUNT = big.FLOAT_32_MANTISSA_BIT_COUNT

const FLOAT_32_SIGN_BITS = big.Float_32_Bits(big.FLOAT_32_SIGN_MASK)

const FLOAT_32_EXPONENT_HALF = FLOAT_32_EXPONENT_BIAS - INDEX_STEP

const FLOAT_32_HALF_EXPONENT_BITS = FLOAT_32_EXPONENT_HALF << FLOAT_32_EXPONENT_SHIFT

const FLOAT_32_ONE_EXPONENT_BITS = FLOAT_32_EXPONENT_BIAS << FLOAT_32_EXPONENT_SHIFT

const FLOAT_32_ONE_BITS big.Float_32_Value_Bits = FLOAT_32_ONE_EXPONENT_BITS

const FLOAT_32_ONE_PLUS_TWO_BITS = FLOAT_32_ONE_EXPONENT_BITS | INDEX_STEP<<INDEX_STEP

const FLOAT_32_QUARTER_EXPONENT = FLOAT_32_EXPONENT_BIAS - INDEX_STEP - INDEX_STEP

const FLOAT_32_QUARTER_EXPONENT_BITS = FLOAT_32_QUARTER_EXPONENT << FLOAT_32_EXPONENT_SHIFT

const FLOAT_32_QUARTER_BITS big.Float_32_Value_Bits = FLOAT_32_QUARTER_EXPONENT_BITS

const FLOAT_32_TWO_EXPONENT = FLOAT_32_EXPONENT_BIAS + INDEX_STEP

const FLOAT_32_TWO_EXPONENT_BITS = FLOAT_32_TWO_EXPONENT << FLOAT_32_EXPONENT_SHIFT

const FLOAT_32_THREE_BITS big.Float_32_Value_Bits = FLOAT_32_TWO_EXPONENT_BITS |
	FLOAT_32_HALF_MANTISSA_BITS

const FLOAT_32_HALF_MANTISSA_BITS = INDEX_STEP << (FLOAT_32_MANTISSA_BIT_COUNT - INDEX_STEP)

const FLOAT_32_INFINITY_EXPONENT_BITS = FLOAT_32_EXPONENT_MASK << FLOAT_32_EXPONENT_SHIFT

const FLOAT_32_POSITIVE_ZERO_BITS big.Float_32_Bits = 0

const FLOAT_32_NEGATIVE_ZERO_BITS = FLOAT_32_SIGN_BITS

const FLOAT_32_HALF_BITS big.Float_32_Bits = FLOAT_32_HALF_EXPONENT_BITS

const FLOAT_32_NEGATIVE_HALF_BITS = FLOAT_32_HALF_BITS | FLOAT_32_SIGN_BITS

const FLOAT_32_THREE_HALVES_BITS = FLOAT_32_ONE_EXPONENT_BITS | FLOAT_32_HALF_MANTISSA_BITS

const FLOAT_32_MAXIMUM_FINITE_BITS = big.Float_32_Bits(
	(FLOAT_32_EXPONENT_MASK-INDEX_STEP)<<FLOAT_32_EXPONENT_SHIFT |
		big.FLOAT_32_MANTISSA_MASK,
)

const FLOAT_32_POSITIVE_INFINITY_BITS big.Float_32_Bits = FLOAT_32_INFINITY_EXPONENT_BITS

const FLOAT_32_NEGATIVE_INFINITY_BITS = FLOAT_32_POSITIVE_INFINITY_BITS | FLOAT_32_SIGN_BITS

const FLOAT_32_NOT_A_NUMBER_BITS = FLOAT_32_POSITIVE_INFINITY_BITS | INDEX_STEP

const FLOAT_64_EXPONENT_BIAS = big.FLOAT_64_EXPONENT_BIAS

const FLOAT_64_EXPONENT_MASK = big.FLOAT_64_EXPONENT_MASK

const FLOAT_64_MANTISSA_BIT_COUNT = big.FLOAT_64_MANTISSA_BIT_COUNT

const FLOAT_64_SIGN_BITS = big.Float_64_Bits(big.FLOAT_64_SIGN_MASK)

const FLOAT_64_EXPONENT_HALF = FLOAT_64_EXPONENT_BIAS - INDEX_STEP

const FLOAT_64_HALF_EXPONENT_BITS = FLOAT_64_EXPONENT_HALF << FLOAT_64_EXPONENT_SHIFT

const FLOAT_64_ONE_EXPONENT_BITS = FLOAT_64_EXPONENT_BIAS << FLOAT_64_EXPONENT_SHIFT

const FLOAT_64_ONE_BITS big.Float_64_Value_Bits = FLOAT_64_ONE_EXPONENT_BITS

const FLOAT_64_ONE_PLUS_TWO_BITS = FLOAT_64_ONE_EXPONENT_BITS | INDEX_STEP<<INDEX_STEP

const FLOAT_64_TWO_EXPONENT = FLOAT_64_EXPONENT_BIAS + INDEX_STEP

const FLOAT_64_TWO_EXPONENT_BITS = FLOAT_64_TWO_EXPONENT << FLOAT_64_EXPONENT_SHIFT

const FLOAT_64_TWO_BITS = FLOAT_64_TWO_EXPONENT_BITS

const FLOAT_64_THREE_BITS = FLOAT_64_TWO_EXPONENT_BITS | FLOAT_64_HALF_MANTISSA_BITS

const FLOAT_64_HALF_MANTISSA_BITS = INDEX_STEP << (FLOAT_64_MANTISSA_BIT_COUNT - INDEX_STEP)

const FLOAT_64_QUARTER_SHIFT = FLOAT_64_MANTISSA_BIT_COUNT - INDEX_STEP - INDEX_STEP

const FLOAT_64_QUARTER_MANTISSA_BITS = INDEX_STEP << FLOAT_64_QUARTER_SHIFT

const FLOAT_64_INFINITY_EXPONENT_BITS = FLOAT_64_EXPONENT_MASK << FLOAT_64_EXPONENT_SHIFT

const FLOAT_64_POSITIVE_ZERO_BITS big.Float_64_Bits = 0

const FLOAT_64_SMALLEST_POSITIVE_BITS = big.Float_64_Bits(bits.CARRY_MAXIMUM)

const FLOAT_64_NEGATIVE_ZERO_BITS = FLOAT_64_SIGN_BITS

const FLOAT_64_HALF_BITS big.Float_64_Bits = FLOAT_64_HALF_EXPONENT_BITS

const FLOAT_64_NEGATIVE_HALF_BITS = FLOAT_64_HALF_BITS | FLOAT_64_SIGN_BITS

const FLOAT_64_THREE_HALVES_BITS = FLOAT_64_ONE_EXPONENT_BITS | FLOAT_64_HALF_MANTISSA_BITS

const FLOAT_64_NEGATIVE_THREE_HALVES_BITS = FLOAT_64_THREE_HALVES_BITS | FLOAT_64_SIGN_BITS

const FLOAT_64_FIVE_QUARTERS_BITS = FLOAT_64_ONE_EXPONENT_BITS | FLOAT_64_QUARTER_MANTISSA_BITS

const FLOAT_64_FOUR_BITS = (FLOAT_64_TWO_EXPONENT + INDEX_STEP) << FLOAT_64_EXPONENT_SHIFT

const FLOAT_64_MAXIMUM_FINITE_BITS = big.Float_64_Bits(
	(FLOAT_64_EXPONENT_MASK-INDEX_STEP)<<FLOAT_64_EXPONENT_SHIFT |
		big.FLOAT_64_MANTISSA_MASK,
)

const FLOAT_64_POSITIVE_INFINITY_BITS big.Float_64_Bits = FLOAT_64_INFINITY_EXPONENT_BITS

const FLOAT_64_NEGATIVE_INFINITY_BITS = FLOAT_64_POSITIVE_INFINITY_BITS | FLOAT_64_SIGN_BITS

const FLOAT_64_NOT_A_NUMBER_BITS = FLOAT_64_POSITIVE_INFINITY_BITS | INDEX_STEP

const TWO_WORD_COUNT = big.WORD_COUNT_INCREMENT + big.WORD_COUNT_INCREMENT

const ZERO_INDEX = big.WORD_COUNT_MINIMUM

const POSITIVE_ONE_WORD_INDEX = ZERO_INDEX + INDEX_STEP

const NEGATIVE_ONE_WORD_INDEX = POSITIVE_ONE_WORD_INDEX + INDEX_STEP

const POSITIVE_TWO_WORD_INDEX = NEGATIVE_ONE_WORD_INDEX + INDEX_STEP

const NEGATIVE_TWO_WORD_INDEX = POSITIVE_TWO_WORD_INDEX + INDEX_STEP

const POSITIVE_THREE_WORD_INDEX = NEGATIVE_TWO_WORD_INDEX + INDEX_STEP

const NEGATIVE_THREE_WORD_INDEX = POSITIVE_THREE_WORD_INDEX + INDEX_STEP

const POSITIVE_MAXIMUM_WORD_INDEX = NEGATIVE_THREE_WORD_INDEX + INDEX_STEP

const NEGATIVE_MAXIMUM_WORD_INDEX = POSITIVE_MAXIMUM_WORD_INDEX + INDEX_STEP

const INT_DOMAIN_COUNT = NEGATIVE_MAXIMUM_WORD_INDEX + INDEX_STEP

const FLOAT_DOMAIN_ZERO_INDEX = 0

const FLOAT_DOMAIN_NEGATIVE_ZERO_INDEX = FLOAT_DOMAIN_ZERO_INDEX + INDEX_STEP

const FLOAT_DOMAIN_INFINITY_INDEX = FLOAT_DOMAIN_NEGATIVE_ZERO_INDEX + INDEX_STEP

const FLOAT_DOMAIN_NEGATIVE_INFINITY_INDEX = FLOAT_DOMAIN_INFINITY_INDEX + INDEX_STEP

const FLOAT_DOMAIN_MINIMUM_EXPONENT_INDEX = FLOAT_DOMAIN_NEGATIVE_INFINITY_INDEX + INDEX_STEP

const FLOAT_DOMAIN_MAXIMUM_EXPONENT_INDEX = FLOAT_DOMAIN_MINIMUM_EXPONENT_INDEX + INDEX_STEP

const FLOAT_DOMAIN_TWO_BIT_INDEX = FLOAT_DOMAIN_MAXIMUM_EXPONENT_INDEX + INDEX_STEP

const FLOAT_DOMAIN_NEGATIVE_ONE_EXPONENT_INDEX = FLOAT_DOMAIN_TWO_BIT_INDEX + INDEX_STEP

const FLOAT_DOMAIN_TWO_WORD_INDEX = FLOAT_DOMAIN_NEGATIVE_ONE_EXPONENT_INDEX + INDEX_STEP

const FLOAT_DOMAIN_THREE_WORD_INDEX = FLOAT_DOMAIN_TWO_WORD_INDEX + INDEX_STEP

const FLOAT_DOMAIN_MAXIMUM_WORD_INDEX = FLOAT_DOMAIN_THREE_WORD_INDEX + INDEX_STEP

const FLOAT_DOMAIN_NEGATIVE_MINIMUM_EXPONENT_INDEX = FLOAT_DOMAIN_MAXIMUM_WORD_INDEX + INDEX_STEP

const FLOAT_DOMAIN_COUNT = FLOAT_DOMAIN_NEGATIVE_MINIMUM_EXPONENT_INDEX + INDEX_STEP

func int_domain_values(t *testing.T) (values [INT_DOMAIN_COUNT]big.Int) {
	big.Int_Set_Uint_64(&values[POSITIVE_ONE_WORD_INDEX], 1)
	big.Int_Negate(
		&values[NEGATIVE_ONE_WORD_INDEX], &values[POSITIVE_ONE_WORD_INDEX],
	)

	var two_word_bytes [big.WORD_BYTE_COUNT + 1]byte
	two_word_bytes[0] = 1
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Bytes(
		&values[POSITIVE_TWO_WORD_INDEX], two_word_bytes[:],
	))
	big.Int_Negate(
		&values[NEGATIVE_TWO_WORD_INDEX], &values[POSITIVE_TWO_WORD_INDEX],
	)

	var three_word_bytes [2*big.WORD_BYTE_COUNT + 1]byte
	three_word_bytes[0] = 1
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Bytes(
		&values[POSITIVE_THREE_WORD_INDEX], three_word_bytes[:],
	))
	big.Int_Negate(
		&values[NEGATIVE_THREE_WORD_INDEX], &values[POSITIVE_THREE_WORD_INDEX],
	)

	var maximum_bytes [bytes.SLICE_SIZE_MAXIMUM]byte
	for index := range maximum_bytes {
		maximum_bytes[index] = byte(bits.WORD_8_MAXIMUM)
	}
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Bytes(
		&values[POSITIVE_MAXIMUM_WORD_INDEX], maximum_bytes[:],
	))
	big.Int_Negate(
		&values[NEGATIVE_MAXIMUM_WORD_INDEX], &values[POSITIVE_MAXIMUM_WORD_INDEX],
	)
	return values
}

func exercise_int_domains(t *testing.T, values *[INT_DOMAIN_COUNT]big.Int) {
	words := [...]big.Word_64{0, 1, 2, big.Word_64(bits.WORD_64_MAXIMUM)}
	shift_counts := [...]big.Shift_Count{0, 1, 2, 3, big.BIT_COUNT_MAXIMUM}
	bit_indexes := [...]big.Bit_Index{0, 1, 2, 3, big.BIT_INDEX_MAXIMUM}
	bit_values := [...]big.Bit_Value{big.BIT_CLEAR, big.BIT_SET}
	var byte_storage [bytes.SLICE_SIZE_MAXIMUM]byte
	byte_destinations := [...]big.Bytes{
		byte_storage[:0],
		byte_storage[:1],
		byte_storage[:2],
		byte_storage[:3],
		byte_storage[:],
	}
	var multiplication_workspace big.Int_Multiplication_Workspace
	var division_workspace big.Int_Division_Workspace
	var bitwise_workspace big.Int_Bitwise_Workspace
	for index := range values {
		value := &values[index]
		right := &values[(index+1)%len(values)]
		word := words[index%len(words)]
		shift_count := shift_counts[index%len(shift_counts)]
		bit_index := bit_indexes[index%len(bit_indexes)]
		bit_value := bit_values[index%len(bit_values)]
		byte_destination := byte_destinations[index%len(byte_destinations)]

		destination := *value
		big.Int_Set_Int_64(&destination, big.Int_64(index))
		destination = *value
		big.Int_Set_Uint_64(&destination, word)
		big.Int_Uint_64(&destination)
		destination = *value
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Bytes(&destination, nil))
		destination = *right
		big.Int_Set(&destination, value)

		big.Int_Sign(value)
		big.Int_Bit_Count(value)
		big.Int_Is_Int_64(value)
		big.Int_Int_64(value)
		big.Int_Is_Uint_64(value)
		big.Int_Uint_64(value)
		destination = *right
		big.Int_Absolute(&destination, value)
		destination = *right
		big.Int_Negate(&destination, value)
		big.Int_Compare(value, right)
		big.Int_Compare_Absolute(value, right)

		exercise_int_arithmetic_domains(
			value, right, &values[POSITIVE_ONE_WORD_INDEX],
			&multiplication_workspace, &division_workspace,
		)
		exercise_int_bitwise_domains(
			value, right, bit_index, bit_value, &bitwise_workspace,
		)
		destination = *right
		big.Int_Shift_Left(&destination, value, shift_count)
		destination = *right
		big.Int_Shift_Right(&destination, value, shift_count)
		exercise_int_byte_domains(
			value, byte_destination, byte_destinations[len(byte_destinations)-1],
		)
	}
	exercise_int_float_domains(values)
	exercise_int_division_workspace_domains(t, values, &division_workspace)
	destination := values[ZERO_INDEX]
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Add(
		&destination, &values[ZERO_INDEX], &values[POSITIVE_ONE_WORD_INDEX],
	))
}

func exercise_int_float_domains(values *[INT_DOMAIN_COUNT]big.Int) {
	encodings := [...]big.Float_64_Bits{
		FLOAT_64_POSITIVE_ZERO_BITS, 1, 2, big.Float_64_Bits(bits.WORD_64_MAXIMUM),
	}
	for index := range values {
		destination := values[(index+INDEX_STEP)%len(values)]
		big.Int_Set_Float_64_Bits(
			&destination, encodings[index%len(encodings)],
		)
		big.Int_Float_64_Bits(&values[index])
	}
}

func float_domain_values(
	t *testing.T, integers *[INT_DOMAIN_COUNT]big.Int,
) (values [FLOAT_DOMAIN_COUNT]big.Float) {
	big.Float_Set_Float_64_Bits(
		&values[FLOAT_DOMAIN_NEGATIVE_ZERO_INDEX], FLOAT_64_NEGATIVE_ZERO_BITS,
	)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Set_Precision(&values[FLOAT_DOMAIN_NEGATIVE_ZERO_INDEX], 1))
	big.Float_Set_Infinity(&values[FLOAT_DOMAIN_INFINITY_INDEX], false)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Set_Precision(&values[FLOAT_DOMAIN_INFINITY_INDEX], 2))
	big.Float_Set_Infinity(&values[FLOAT_DOMAIN_NEGATIVE_INFINITY_INDEX], true)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Set_Precision(
			&values[FLOAT_DOMAIN_NEGATIVE_INFINITY_INDEX], big.FLOAT_PRECISION_MAXIMUM,
		))
	for index := FLOAT_DOMAIN_MINIMUM_EXPONENT_INDEX; index < len(values); index++ {
		integer_index := [...]int{
			POSITIVE_ONE_WORD_INDEX, NEGATIVE_ONE_WORD_INDEX,
			POSITIVE_ONE_WORD_INDEX, POSITIVE_ONE_WORD_INDEX,
			POSITIVE_TWO_WORD_INDEX, POSITIVE_THREE_WORD_INDEX,
			POSITIVE_MAXIMUM_WORD_INDEX, NEGATIVE_ONE_WORD_INDEX,
		}[index-FLOAT_DOMAIN_MINIMUM_EXPONENT_INDEX]
		big.Float_Set_Int(&values[index], &integers[integer_index])
	}
	values[FLOAT_DOMAIN_MINIMUM_EXPONENT_INDEX].Precision = 1
	values[FLOAT_DOMAIN_MAXIMUM_EXPONENT_INDEX].Precision = 1
	values[FLOAT_DOMAIN_TWO_BIT_INDEX].Precision = 1
	values[FLOAT_DOMAIN_TWO_BIT_INDEX].Mantissa.Words[ZERO_INDEX] = 3
	values[FLOAT_DOMAIN_TWO_BIT_INDEX].Precision = 2
	values[FLOAT_DOMAIN_NEGATIVE_ONE_EXPONENT_INDEX].Precision = 1
	values[FLOAT_DOMAIN_NEGATIVE_MINIMUM_EXPONENT_INDEX].Precision = 1
	values[FLOAT_DOMAIN_MINIMUM_EXPONENT_INDEX].Exponent = big.FLOAT_EXPONENT_MINIMUM
	values[FLOAT_DOMAIN_MAXIMUM_EXPONENT_INDEX].Exponent = big.FLOAT_EXPONENT_MAXIMUM
	values[FLOAT_DOMAIN_TWO_BIT_INDEX].Exponent = 0
	values[FLOAT_DOMAIN_NEGATIVE_ONE_EXPONENT_INDEX].Exponent = -1
	values[FLOAT_DOMAIN_TWO_WORD_INDEX].Mantissa.Words[ZERO_INDEX] = 1
	values[FLOAT_DOMAIN_TWO_WORD_INDEX].Exponent = 1
	values[FLOAT_DOMAIN_THREE_WORD_INDEX].Exponent = 2
	values[FLOAT_DOMAIN_MAXIMUM_WORD_INDEX].Exponent = 0
	values[FLOAT_DOMAIN_NEGATIVE_MINIMUM_EXPONENT_INDEX].Exponent = big.FLOAT_EXPONENT_MINIMUM
	for index := range values {
		values[index].Mode = big.Rounding_Mode(index %
			(int(big.ROUND_TO_POSITIVE_INFINITY) + INDEX_STEP))
	}
	values[FLOAT_DOMAIN_MINIMUM_EXPONENT_INDEX].Accuracy = big.ACCURACY_BELOW
	values[FLOAT_DOMAIN_MAXIMUM_EXPONENT_INDEX].Accuracy = big.ACCURACY_ABOVE
	return values
}

func exercise_float_domains(t *testing.T, integers *[INT_DOMAIN_COUNT]big.Int) {
	values := float_domain_values(t, integers)
	var addition_workspace big.Float_Addition_Workspace
	var multiplication_workspace big.Float_Multiplication_Workspace
	var division_workspace big.Float_Division_Workspace
	var float_rat_workspace big.Float_Rat_Workspace
	var square_root_workspace big.Float_Square_Root_Workspace
	var rational big.Rat
	big.Rat_Set_Int_64(&rational, 1)
	subtraction_larger := values[FLOAT_DOMAIN_MAXIMUM_WORD_INDEX]
	subtraction_larger.Exponent = big.FLOAT_EXPONENT_MAXIMUM
	precisions := [...]big.Float_Precision_Unvalidated{
		0, 1, 2, big.FLOAT_PRECISION_MAXIMUM,
	}
	modes := [...]big.Rounding_Mode_Unvalidated{
		big.ROUND_TO_NEAREST_EVEN, big.ROUND_TO_NEAREST_AWAY, big.ROUND_TO_ZERO,
		big.ROUND_TO_POSITIVE_INFINITY,
	}
	encodings_32 := [...]big.Float_32_Bits{
		FLOAT_32_POSITIVE_ZERO_BITS, FLOAT_32_NEGATIVE_ZERO_BITS,
		1, 2, FLOAT_32_HALF_BITS, FLOAT_32_MAXIMUM_FINITE_BITS,
		FLOAT_32_POSITIVE_INFINITY_BITS, big.Float_32_Bits(bits.WORD_32_MAXIMUM),
	}
	encodings := [...]big.Float_64_Bits{
		FLOAT_64_POSITIVE_ZERO_BITS, FLOAT_64_NEGATIVE_ZERO_BITS,
		1, 2, FLOAT_64_HALF_BITS, FLOAT_64_NEGATIVE_HALF_BITS,
		FLOAT_64_MAXIMUM_FINITE_BITS, FLOAT_64_POSITIVE_INFINITY_BITS,
		big.Float_64_Bits(bits.WORD_64_MAXIMUM),
	}
	for index := range values {
		value := &values[index]
		destination := values[(index+INDEX_STEP)%len(values)]
		big.Float_Set_Precision(&destination, precisions[index%len(precisions)])
		destination = values[(index+INDEX_STEP)%len(values)]
		big.Float_Set_Rounding_Mode(&destination, modes[index%len(modes)])
		next := &values[(index+INDEX_STEP)%len(values)]
		exercise_float_observation_domain(value, &destination, next)
		exercise_float_binary_32_domain(
			value, &destination, next, encodings_32[index%len(encodings_32)],
		)
		exercise_float_arithmetic_domain(
			value, &destination, &values[FLOAT_DOMAIN_ZERO_INDEX], next,
			&subtraction_larger, &addition_workspace,
		)
		exercise_float_multiplication_domain(
			value, &destination, &values[FLOAT_DOMAIN_ZERO_INDEX], next,
			&multiplication_workspace,
		)
		exercise_float_division_domain(
			value, &destination, &values[FLOAT_DOMAIN_ZERO_INDEX], next,
			&division_workspace,
		)
		exercise_float_square_root_domain(
			value, &destination, next, &square_root_workspace,
		)
		exercise_float_conversion_domain(
			value, &destination, next, &integers[index%len(integers)], &rational,
			&float_rat_workspace, big.Word_64(index),
			big.Int_64(index-INDEX_STEP), encodings[index%len(encodings)],
			big.Boolean(index%2 == 0),
		)
	}
	exercise_float_representation_domains(&values)
	exercise_float_round_boundaries(
		t, &values, &addition_workspace, &multiplication_workspace,
	)
	exercise_float_division_boundaries(t, &values, &division_workspace)
	exercise_float_square_root_boundaries(t, &values, &square_root_workspace)
	exercise_float_validation_domains(t, &values[0])
}

func exercise_float_representation_domains(values *[FLOAT_DOMAIN_COUNT]big.Float) {
	exercise_float_text_domains(values)
	exercise_float_parse_domains(values)
	exercise_float_gob_domains(values)
}

func exercise_float_parse_domains(values *[FLOAT_DOMAIN_COUNT]big.Float) {
	var workspace big.Float_Parse_Workspace
	for index := range values {
		destination := values[index]
		big.Float_Parse(&destination, nil, 63, &workspace)
	}
}

func exercise_float_text_domains(values *[FLOAT_DOMAIN_COUNT]big.Float) {
	var storage [big.INT_TEXT_SIZE_MAXIMUM]byte
	destinations := [...]big.Text{
		storage[:0], storage[:1], storage[:2], storage[:],
	}
	formats := [...]big.Float_Text_Format_Unvalidated{
		0, 1, 2, big.Float_Text_Format_Unvalidated(bits.WORD_8_MAXIMUM),
	}
	var workspace big.Float_Text_Workspace
	for index := range values {
		big.Float_Text_Into(
			destinations[index%len(destinations)], &values[index],
			formats[index%len(formats)],
			big.FLOAT_TEXT_PRECISION_UNVALIDATED_MINIMUM, &workspace,
		)
	}
}

func exercise_float_gob_domains(values *[FLOAT_DOMAIN_COUNT]big.Float) {
	var workspace big.Float_Gob_Workspace
	var storage [big.FLOAT_GOB_SIZE_MAXIMUM]byte
	for index := range values {
		value := &values[index]
		next := &values[(index+INDEX_STEP)%len(values)]
		destination := *next
		exercise_float_gob_domain(value, &destination, next, &workspace, storage[:])
	}
}

func exercise_float_gob_domain(
	value *big.Float,
	destination *big.Float,
	next *big.Float,
	workspace *big.Float_Gob_Workspace,
	storage big.Float_Gob_Encoding,
) {
	count, _ := big.Float_Gob_Encode_Into(storage, value, workspace)
	size := big.FLOAT_GOB_HEADER_SIZE
	if value.Form == big.FLOAT_FORM_FINITE {
		size = big.FLOAT_GOB_FINITE_PREFIX_SIZE + int(count)*big.WORD_BYTE_COUNT
	}
	*destination = *next
	big.Float_Gob_Decode(
		destination, big.Float_Gob_Encoding_Unvalidated(storage[:size]), workspace,
	)
}

func exercise_float_binary_32_domain(
	value *big.Float,
	destination *big.Float,
	next *big.Float,
	encoding big.Float_32_Bits,
) {
	*destination = *next
	big.Float_Set_Float_32_Bits(destination, encoding)
	big.Float_Float_32_Bits(destination)
	*destination = *next
	big.Float_Set_Float_32_Bits(destination, FLOAT_32_HALF_BITS)
	big.Float_Set_Float_32_Bits(destination, FLOAT_32_NEGATIVE_HALF_BITS)
	big.Float_Float_32_Bits(value)
}

func exercise_float_conversion_domain(
	value *big.Float,
	destination *big.Float,
	next *big.Float,
	integer *big.Int,
	rational *big.Rat,
	workspace *big.Float_Rat_Workspace,
	word big.Word_64,
	signed big.Int_64,
	encoding big.Float_64_Bits,
	negative big.Boolean,
) {
	*destination = *next
	big.Float_Set_Uint_64(destination, word)
	big.Float_Set_Uint_64(destination, big.Word_64(bits.WORD_64_MAXIMUM))
	*destination = *next
	big.Float_Set_Int_64(destination, signed)
	big.Float_Set_Int_64(destination, big.Int_64(bits.INTEGER_64_MINIMUM))
	big.Float_Set_Int_64(destination, big.Int_64(bits.INTEGER_64_MAXIMUM))
	*destination = *next
	big.Float_Set_Int(destination, integer)
	*destination = *next
	big.Float_Set_Float_64_Bits(destination, encoding)
	*destination = *next
	big.Float_Set_Float_64_Bits(destination, FLOAT_64_HALF_BITS)
	big.Float_Set_Float_64_Bits(destination, FLOAT_64_NEGATIVE_HALF_BITS)
	*destination = *next
	big.Float_Set_Infinity(destination, negative)
	*destination = *next
	big.Float_Set(destination, value)
	*destination = *next
	big.Float_Copy(destination, value)
	mantissa := *next
	exponent := big.Float_Mantissa_Exponent(value, &mantissa)
	*destination = *next
	big.Float_Set_Mantissa_Exponent(
		destination, &mantissa, big.Float_Exponent_Unvalidated(exponent),
	)
	*destination = *next
	big.Float_Set_Mantissa_Exponent(destination, value, 0)
	big.Float_Float_64_Bits(value)
	integer_destination := *integer
	big.Float_Int_Into(&integer_destination, value)
	big.Float_Int_64(value)
	big.Float_Uint_64(value)
	*destination = *next
	big.Float_Set_Rat(destination, rational, workspace)
	converted_rational := *rational
	big.Float_Rat_Into(&converted_rational, value, workspace)
}

func exercise_float_observation_domain(
	value *big.Float, destination *big.Float, next *big.Float,
) {
	big.Float_Precision_Of(value)
	big.Float_Minimum_Precision(value)
	big.Float_Rounding_Mode(value)
	big.Float_Accuracy(value)
	big.Float_Sign(value)
	big.Float_Sign_Bit(value)
	big.Float_Is_Infinite(value)
	big.Float_Is_Integer(value)
	*destination = *next
	big.Float_Absolute(destination, value)
	*destination = *next
	big.Float_Negate(destination, value)
	big.Float_Compare(value, value)
	big.Float_Compare(value, destination)
}

func exercise_float_arithmetic_domain(
	value *big.Float,
	destination *big.Float,
	zero *big.Float,
	next *big.Float,
	larger *big.Float,
	workspace *big.Float_Addition_Workspace,
) {
	*destination = *next
	big.Float_Add(destination, value, value, workspace)
	*destination = *next
	big.Float_Add(destination, zero, value, workspace)
	*destination = *next
	big.Float_Add(destination, value, zero, workspace)
	*destination = *next
	big.Float_Subtract(destination, value, value, workspace)
	*destination = *next
	big.Float_Subtract(destination, value, next, workspace)
	*destination = *next
	smaller := *value
	smaller.Negative = big.POLARITY_NONNEGATIVE
	big.Float_Subtract(destination, larger, &smaller, workspace)
}

func exercise_float_multiplication_domain(
	value *big.Float,
	destination *big.Float,
	zero *big.Float,
	next *big.Float,
	workspace *big.Float_Multiplication_Workspace,
) {
	*destination = *next
	big.Float_Multiply(destination, value, value, workspace)
	*destination = *next
	big.Float_Multiply(destination, zero, value, workspace)
	*destination = *next
	big.Float_Multiply(destination, value, zero, workspace)
	var infinity big.Float
	big.Float_Set_Infinity(&infinity, false)
	var one big.Float
	big.Float_Set_Int_64(&one, 1)
	*destination = *value
	big.Float_Multiply(destination, &infinity, &one, workspace)
	big.Float_Negate(&one, &one)
	*destination = *value
	big.Float_Multiply(destination, &infinity, &one, workspace)
	if value.Form == big.FLOAT_FORM_FINITE {
		negative := *value
		big.Float_Negate(&negative, &negative)
		*destination = *next
		big.Float_Multiply(destination, value, &negative, workspace)
	}
	var zero_precision big.Float
	big.Float_Multiply(&zero_precision, &infinity, &infinity, workspace)
}

func exercise_float_division_domain(
	value *big.Float,
	destination *big.Float,
	zero *big.Float,
	next *big.Float,
	workspace *big.Float_Division_Workspace,
) {
	*destination = *next
	big.Float_Quotient(destination, value, value, workspace)
	*destination = *next
	big.Float_Quotient(destination, zero, value, workspace)
	*destination = *next
	big.Float_Quotient(destination, value, zero, workspace)
	var infinity big.Float
	big.Float_Set_Infinity(&infinity, false)
	var one big.Float
	big.Float_Set_Int_64(&one, 1)
	*destination = *next
	big.Float_Quotient(destination, &infinity, &one, workspace)
	*destination = *next
	big.Float_Quotient(destination, &one, &infinity, workspace)
	var three big.Float
	big.Float_Set_Int_64(&three, 3)
	if big.Float_Sign_Bit(value) {
		big.Float_Negate(&three, &three)
	}
	*destination = *next
	big.Float_Quotient(destination, &one, &three, workspace)
	underflow_dividend := one
	underflow_dividend.Exponent = big.FLOAT_EXPONENT_MINIMUM
	underflow_dividend.Negative = three.Negative
	underflow_divisor := three
	underflow_divisor.Negative = big.POLARITY_NONNEGATIVE
	underflow_divisor.Exponent = big.FLOAT_EXPONENT_MAXIMUM
	*destination = *next
	big.Float_Quotient(
		destination, &underflow_dividend, &underflow_divisor, workspace,
	)
	overflow_dividend := three
	overflow_dividend.Exponent = big.FLOAT_EXPONENT_MAXIMUM
	overflow_divisor := one
	overflow_divisor.Exponent = big.FLOAT_EXPONENT_MINIMUM
	*destination = *next
	big.Float_Quotient(
		destination, &overflow_dividend, &overflow_divisor, workspace,
	)
}

func exercise_float_square_root_domain(
	value *big.Float,
	destination *big.Float,
	next *big.Float,
	workspace *big.Float_Square_Root_Workspace,
) {
	*destination = *next
	big.Float_Square_Root(destination, value, workspace)
	if value.Form == big.FLOAT_FORM_FINITE {
		positive := *value
		positive.Negative = big.POLARITY_NONNEGATIVE
		*destination = *next
		big.Float_Square_Root(destination, &positive, workspace)
	}
	var one big.Float
	big.Float_Set_Int_64(&one, 1)
	*destination = *next
	big.Float_Square_Root(destination, &one, workspace)
}

func exercise_float_square_root_boundaries(
	t *testing.T,
	values *[FLOAT_DOMAIN_COUNT]big.Float,
	workspace *big.Float_Square_Root_Workspace,
) {
	var two big.Float
	big.Float_Set_Int_64(&two, 2)
	var result big.Float
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Set_Precision(&result, big.WORD_BIT_COUNT+INDEX_STEP))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Square_Root(&result, &two, workspace))
	maximum := values[FLOAT_DOMAIN_MAXIMUM_WORD_INDEX]
	maximum.Negative = big.POLARITY_NONNEGATIVE
	maximum.Exponent = 1
	testify.Equal_Values(t, big.STATUS_OK, big.Float_Set_Precision(&result, 1))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Square_Root(&result, &maximum, workspace))
	for _, source_index := range []int{0, 1, 2} {
		shift := source_index + big.FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT*
			(INDEX_STEP+big.FLOAT_SQUARE_ROOT_GUARD_BIT_COUNT)
		var source big.Float
		big.Float_Set_Uint_64(&source, big.Word_64(INDEX_STEP<<uint(shift)))
		source.Exponent = 0
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Square_Root(&result, &source, workspace))
	}
	two.Exponent = -big.Float_Exponent(big.BASE_BINARY)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Square_Root(&result, &two, workspace))
}

func exercise_float_division_boundaries(
	t *testing.T,
	values *[FLOAT_DOMAIN_COUNT]big.Float,
	workspace *big.Float_Division_Workspace,
) {
	var one big.Float
	big.Float_Set_Int_64(&one, 1)
	var two big.Float
	big.Float_Set_Int_64(&two, 2)
	var three big.Float
	big.Float_Set_Int_64(&three, 3)
	var result big.Float
	testify.Equal_Values(t, big.STATUS_OK, big.Float_Set_Precision(&result, 3))
	for _, exponent := range []big.Float_Exponent{
		big.FLOAT_EXPONENT_MINIMUM, -1, 0, 1, 2, big.FLOAT_EXPONENT_MAXIMUM,
	} {
		dividend := one
		dividend.Exponent = exponent
		divisor := three
		divisor.Exponent = 0
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Quotient(&result, &dividend, &divisor, workspace))
	}
	testify.Equal_Values(t, big.STATUS_OK, big.Float_Set_Precision(&result, 2))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Quotient(&result, &three, &two, workspace))
	maximum := values[FLOAT_DOMAIN_MAXIMUM_WORD_INDEX]
	smaller := maximum
	smaller.Mantissa.Words[ZERO_INDEX]--
	testify.Equal_Values(t, big.STATUS_OK, big.Float_Set_Precision(&result, 1))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Quotient(&result, &maximum, &smaller, workspace))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Quotient(&result, &smaller, &maximum, workspace))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Quotient(&result, &one, &one, workspace))
}

func exercise_float_round_boundaries(
	t *testing.T,
	values *[FLOAT_DOMAIN_COUNT]big.Float,
	workspace *big.Float_Addition_Workspace,
	multiplication_workspace *big.Float_Multiplication_Workspace,
) {
	maximum := values[FLOAT_DOMAIN_MAXIMUM_WORD_INDEX]
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Set_Precision(&maximum, big.BIT_INDEX_MAXIMUM))
	maximum = values[FLOAT_DOMAIN_MAXIMUM_WORD_INDEX]
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Set_Precision(&maximum, 1))
	for _, exponent := range []big.Float_Exponent{
		big.FLOAT_EXPONENT_MINIMUM, -1, big.FLOAT_EXPONENT_MAXIMUM,
	} {
		value := values[FLOAT_DOMAIN_TWO_BIT_INDEX]
		value.Exponent = exponent
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Set_Precision(&value, 1))
	}
	low := values[FLOAT_DOMAIN_MAXIMUM_WORD_INDEX]
	low.Exponent = big.FLOAT_EXPONENT_MINIMUM
	high := values[FLOAT_DOMAIN_MINIMUM_EXPONENT_INDEX]
	high.Exponent = big.FLOAT_EXPONENT_MAXIMUM
	var result big.Float
	testify.Equal_Values(t, big.STATUS_OK, big.Float_Set_Precision(&result, 1))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Add(&result, &low, &high, workspace))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Subtract(&result, &high, &low, workspace))
	exercise_float_addition_origins(t, values, workspace)
	exercise_float_addition_round_domains(t, values, workspace)
	low_product := low
	low_product.Exponent = big.FLOAT_EXPONENT_MINIMUM
	high_product := values[FLOAT_DOMAIN_MINIMUM_EXPONENT_INDEX]
	high_product.Exponent = big.FLOAT_EXPONENT_MAXIMUM
	big.Float_Multiply(
		&result, &low_product, &low_product, multiplication_workspace,
	)
	big.Float_Multiply(
		&result, &high_product, &high_product, multiplication_workspace,
	)
	result = values[FLOAT_DOMAIN_INFINITY_INDEX]
	big.Float_Multiply(
		&result, &high_product, &high_product, multiplication_workspace,
	)
}

func exercise_float_addition_origins(
	t *testing.T,
	values *[FLOAT_DOMAIN_COUNT]big.Float,
	workspace *big.Float_Addition_Workspace,
) {
	for _, exponent := range []big.Float_Exponent{0, 1, 2, 3} {
		one := values[FLOAT_DOMAIN_MINIMUM_EXPONENT_INDEX]
		one.Exponent = exponent
		result := values[FLOAT_DOMAIN_INFINITY_INDEX]
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Add(&result, &one, &one, workspace))
	}
	one := values[FLOAT_DOMAIN_MINIMUM_EXPONENT_INDEX]
	one.Exponent = 1
	var zero big.Float
	big.Float_Add(&zero, &zero, &zero, workspace)
	infinity := values[FLOAT_DOMAIN_INFINITY_INDEX]
	big.Float_Subtract(&infinity, &one, &one, workspace)
	for _, exponent := range []big.Float_Exponent{2, 3} {
		shifted := one
		shifted.Exponent = exponent
		var result big.Float
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Add(&result, &one, &shifted, workspace))
		larger := values[FLOAT_DOMAIN_MAXIMUM_WORD_INDEX]
		larger.Exponent = big.FLOAT_EXPONENT_MAXIMUM
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Subtract(&result, &larger, &shifted, workspace))
	}
}

func exercise_float_addition_round_domains(
	t *testing.T,
	values *[FLOAT_DOMAIN_COUNT]big.Float,
	workspace *big.Float_Addition_Workspace,
) {
	one := values[FLOAT_DOMAIN_MINIMUM_EXPONENT_INDEX]
	one.Exponent = 1
	for _, domain := range []struct {
		Exponent big.Float_Exponent
		Mode     big.Rounding_Mode_Unvalidated
	}{
		{-1, big.ROUND_AWAY_FROM_ZERO},
		{2, big.ROUND_AWAY_FROM_ZERO},
		{2, big.ROUND_TO_ZERO},
	} {
		left := values[FLOAT_DOMAIN_MINIMUM_EXPONENT_INDEX]
		left.Exponent = domain.Exponent - 1
		right := left
		right.Exponent = domain.Exponent
		var result big.Float
		big.Float_Set_Precision(&result, 1)
		big.Float_Set_Rounding_Mode(&result, domain.Mode)
		big.Float_Add(&result, &left, &right, workspace)
	}
	exercise_float_addition_minimum_exponent(values, workspace)
	exercise_float_addition_wide_precision(values, workspace)
	var result big.Float
	result = values[FLOAT_DOMAIN_MINIMUM_EXPONENT_INDEX]
	big.Float_Subtract(&result, &one, &one, workspace)
	eight := one
	eight.Exponent = 4
	big.Float_Set_Precision(&result, 1)
	big.Float_Set_Rounding_Mode(&result, big.ROUND_AWAY_FROM_ZERO)
	big.Float_Add(&result, &one, &eight, workspace)
}

func exercise_float_addition_minimum_exponent(
	values *[FLOAT_DOMAIN_COUNT]big.Float,
	workspace *big.Float_Addition_Workspace,
) {
	left := values[FLOAT_DOMAIN_MAXIMUM_WORD_INDEX]
	left.Exponent = -2
	right := left
	right.Mantissa.Words[ZERO_INDEX] -= big.Word(
		INDEX_STEP + INDEX_STEP + INDEX_STEP,
	)
	var result big.Float
	big.Float_Set_Precision(&result, 1)
	big.Float_Set_Rounding_Mode(&result, big.ROUND_AWAY_FROM_ZERO)
	big.Float_Subtract(&result, &left, &right, workspace)
}

func exercise_float_addition_wide_precision(
	values *[FLOAT_DOMAIN_COUNT]big.Float,
	workspace *big.Float_Addition_Workspace,
) {
	one := values[FLOAT_DOMAIN_MINIMUM_EXPONENT_INDEX]
	one.Exponent = 1
	wide := values[FLOAT_DOMAIN_THREE_WORD_INDEX]
	wide.Exponent = big.Float_Exponent(big.WORD_BIT_COUNT + INDEX_STEP + INDEX_STEP)
	var result big.Float
	big.Float_Set_Precision(&result, big.WORD_BIT_COUNT+INDEX_STEP)
	big.Float_Set_Rounding_Mode(&result, big.ROUND_AWAY_FROM_ZERO)
	big.Float_Add(&result, &one, &wide, workspace)
	maximum := values[FLOAT_DOMAIN_MAXIMUM_WORD_INDEX]
	below := maximum
	below.Mantissa.Words[ZERO_INDEX] -= big.Word(INDEX_STEP)
	big.Float_Set_Precision(&result, big.FLOAT_PRECISION_MAXIMUM)
	big.Float_Set_Rounding_Mode(&result, big.ROUND_AWAY_FROM_ZERO)
	big.Float_Add(&result, &maximum, &below, workspace)
	two := one
	two.Exponent = 2
	four := one
	four.Exponent = 3
	big.Float_Set_Precision(&result, 2)
	big.Float_Add(&result, &one, &four, workspace)
	big.Float_Subtract(&result, &two, &one, workspace)
}

func exercise_float_validation_domains(t *testing.T, destination *big.Float) {
	for _, precision := range []big.Float_Precision_Unvalidated{
		0, 1, 2, big.FLOAT_PRECISION_MAXIMUM,
		big.Float_Precision_Unvalidated(bits.WORD_MAXIMUM),
	} {
		big.Float_Set_Precision(destination, precision)
	}
	for _, mode := range []big.Rounding_Mode_Unvalidated{
		big.ROUND_TO_NEAREST_EVEN, big.ROUND_TO_NEAREST_AWAY, big.ROUND_TO_ZERO,
		big.ROUND_TO_POSITIVE_INFINITY, big.ROUNDING_MODE_UNVALIDATED_MAXIMUM,
	} {
		big.Float_Set_Rounding_Mode(destination, mode)
	}
	for _, exponent := range []big.Float_Exponent_Unvalidated{
		big.FLOAT_EXPONENT_UNVALIDATED_MINIMUM, big.FLOAT_EXPONENT_MINIMUM,
		0, 1, 2, big.FLOAT_EXPONENT_MAXIMUM,
		big.FLOAT_EXPONENT_UNVALIDATED_MAXIMUM,
	} {
		var mantissa big.Float
		big.Float_Set_Uint_64(&mantissa, 1)
		big.Float_Set_Mantissa_Exponent(destination, &mantissa, exponent)
	}
}

func exercise_int_bitwise_domains(
	left *big.Int,
	right *big.Int,
	index big.Bit_Index,
	bit big.Bit_Value,
	workspace *big.Int_Bitwise_Workspace,
) {
	destination := *right
	big.Int_And(&destination, left, right, workspace)
	destination = *right
	big.Int_And_Not(&destination, left, right, workspace)
	destination = *right
	big.Int_Or(&destination, left, right, workspace)
	destination = *right
	big.Int_Xor(&destination, left, right, workspace)
	destination = *right
	big.Int_Not(&destination, left, workspace)
	big.Int_Bit(left, index)
	destination = *right
	big.Int_Set_Bit(&destination, left, index, bit, workspace)
}

func exercise_int_division_workspace_domains(
	t *testing.T,
	values *[INT_DOMAIN_COUNT]big.Int,
	workspace *big.Int_Division_Workspace,
) {
	one := &values[POSITIVE_ONE_WORD_INDEX]
	maximum := &values[POSITIVE_MAXIMUM_WORD_INDEX]
	var quotient big.Int
	var remainder big.Int
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Quotient_Remainder(&quotient, &remainder, maximum, one, workspace))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Quotient_Remainder(&quotient, &remainder, one, one, workspace))
	var below_maximum big.Int
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Subtract(&below_maximum, maximum, one))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Quotient_Remainder(
			&quotient, &remainder, &below_maximum, maximum, workspace,
		))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Quotient_Remainder(&quotient, &remainder, one, one, workspace))
	two_words := &values[POSITIVE_TWO_WORD_INDEX]
	three_words := &values[POSITIVE_THREE_WORD_INDEX]
	exercise_division_remainder_count(
		t, &below_maximum, maximum, one, workspace,
	)
	exercise_division_remainder_count(t, two_words, three_words, one, workspace)
}

func exercise_division_remainder_count(
	t *testing.T,
	dividend *big.Int,
	divisor *big.Int,
	one *big.Int,
	workspace *big.Int_Division_Workspace,
) {
	var quotient big.Int
	var remainder big.Int
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Quotient_Remainder(
			&quotient, &remainder, dividend, divisor, workspace,
		))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Quotient_Remainder(&quotient, &remainder, one, one, workspace))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Quotient_Remainder(
			&quotient, &remainder, dividend, divisor, workspace,
		))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Quotient(&quotient, one, one, workspace))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Quotient_Remainder(
			&quotient, &remainder, dividend, divisor, workspace,
		))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Remainder(&remainder, one, one, workspace))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Quotient_Remainder(
			&quotient, &remainder, dividend, divisor, workspace,
		))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Divide_Modulus(&quotient, &remainder, one, one, workspace))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Divide_Modulus(
			&quotient, &remainder, dividend, divisor, workspace,
		))
}

func exercise_int_arithmetic_domains(
	value *big.Int,
	right *big.Int,
	one *big.Int,
	multiplication_workspace *big.Int_Multiplication_Workspace,
	division_workspace *big.Int_Division_Workspace,
) {
	destination := *right
	big.Int_Add(&destination, value, value)
	destination = *right
	big.Int_Subtract(&destination, value, value)
	destination = *right
	big.Int_Multiply(&destination, value, one, multiplication_workspace)
	destination = *right
	big.Int_Multiply(&destination, one, value, multiplication_workspace)
	quotient := *value
	remainder := *right
	big.Int_Quotient_Remainder(
		&quotient, &remainder, value, one, division_workspace,
	)
	quotient = *right
	remainder = *value
	big.Int_Quotient_Remainder(
		&quotient, &remainder, one, value, division_workspace,
	)
	destination = *right
	big.Int_Quotient(&destination, value, one, division_workspace)
	destination = *value
	big.Int_Quotient(&destination, one, value, division_workspace)
	destination = *right
	big.Int_Remainder(&destination, value, one, division_workspace)
	destination = *value
	big.Int_Remainder(&destination, one, value, division_workspace)
	quotient = *value
	remainder = *right
	big.Int_Divide_Modulus(
		&quotient, &remainder, value, one, division_workspace,
	)
	quotient = *right
	remainder = *value
	big.Int_Divide_Modulus(
		&quotient, &remainder, one, value, division_workspace,
	)
}

func exercise_int_byte_domains(value *big.Int, destination big.Bytes, full big.Bytes) {
	big.Int_Bytes_Into(destination, value)
	big.Int_Fill_Bytes(destination, value)
	big.Int_Bytes_Into(full, value)
	big.Int_Fill_Bytes(full, value)
	big.Int_Trailing_Zero_Bit_Count(value)
}

type allocation_fixture struct {
	Left                  big.Int
	Right                 big.Int
	Result                big.Int
	Validated             big.Bytes
	Validation            big.Validation_Status
	Conversion            big.Conversion_Status
	Arithmetic            big.Arithmetic_Status
	Boolean               big.Boolean
	Sign                  big.Sign
	Order                 big.Order
	Count                 big.Bit_Count
	Bit_Index             big.Bit_Index
	Bit_Value             big.Bit_Value
	Int_64                big.Int_64
	Word_64               big.Word_64
	Bytes                 [bytes.SLICE_SIZE_MAXIMUM]byte
	Int_Encoding          [big.INT_GOB_SIZE_MAXIMUM]byte
	Encoding_Count        big.Int_Encoding_Count
	Words                 [big.WORD_COUNT_MAXIMUM]big.Word
	Word_Count            big.Word_Count
	Multiply              big.Int_Multiplication_Workspace
	Division              big.Int_Division_Workspace
	Bitwise               big.Int_Bitwise_Workspace
	Greatest_Common       big.Int_Greatest_Common_Divisor_Workspace
	Exponent              big.Int_Exponent_Workspace
	Modular               big.Int_Modular_Workspace
	Modular_Status        big.Modular_Status
	Modular_Square_Root   big.Int_Modular_Square_Root_Workspace
	Square_Root_Candidate big.Int
	Square_Root_Status    big.Modular_Square_Root_Status
	Product               big.Int_Product_Workspace
	Square_Root           big.Int_Square_Root_Workspace
	Random_Maximum        big.Int
	Random_Source         [big.RANDOM_WORD_SIZE_MAXIMUM]big.Word
	Random_Consumed       big.Random_Word_Count
	Random_Status         big.Random_Status
	Random_Memory         big.Int_Random_Workspace
	Jacobi                big.Jacobi_Symbol
	Jacobi_Status         big.Validation_Status
	Jacobi_Memory         big.Int_Jacobi_Workspace
	Primality_Value       big.Int
	Primality             big.Boolean
	Primality_Consumed    big.Random_Word_Count
	Primality_Status      big.Primality_Status
	Primality_Options     big.Primality_Options
	Primality_Source      [big.RANDOM_WORD_SIZE_MAXIMUM]big.Word
	Primality_Memory      big.Int_Primality_Workspace
	Rat_Left              big.Rat
	Rat_Right             big.Rat
	Rat_Result            big.Rat
	Rat_Arithmetic        big.Rat_Arithmetic_Status
	Rat_Division          big.Rat_Division_Status
	Rat_Workspace         big.Rat_Workspace
	Rat_Text              [big.RAT_TEXT_SIZE_MAXIMUM]byte
	Rat_Text_Count        big.Rat_Text_Count
	Rat_Fraction          big.Rat_Fraction_Text_Count
	Rat_Text_Memory       big.Rat_Text_Workspace
	Rat_Precision         big.Rat_Precision
	Rat_Float_Memory      big.Rat_Float_Text_Workspace
	Rat_Places            big.Decimal_Place_Count
	Rat_Exact             big.Boolean
	Rat_Place_Memory      big.Rat_Float_Precision_Workspace
	Float_32_Bits         big.Float_32_Bits
	Float_32_Encoding     big.Float_32_Value_Bits
	Float_64_Bits         big.Float_64_Bits
	Int_Float_64          big.Int_Float_64_Value_Bits
	Rat_Float_32          big.Float_32_Value_Bits
	Rat_Float_64          big.Float_64_Value_Bits
	Rat_Float_Exact       big.Boolean
	Rat_Float_32_Memory   big.Rat_Float_32_Workspace
	Rat_Float_64_Memory   big.Rat_Float_64_Workspace
	Rat_Parse_Text        [len("-3/4")]byte
	Rat_Parse_Status      big.Parse_Status
	Rat_Parse_Memory      big.Rat_Parse_Fraction_Workspace
	Rat_Number_Text       [len("-1.5e1")]byte
	Rat_Number_Status     big.Parse_Status
	Rat_Number_Memory     big.Rat_Parse_Workspace
	Rat_Gob               [big.RAT_GOB_SIZE_MAXIMUM]byte
	Rat_Encoding          big.Rat_Encoding_Count
	Rat_Gob_Memory        big.Rat_Gob_Workspace
	Float_Left            big.Float
	Float_Right           big.Float
	Float_Result          big.Float
	Float_Addition        big.Float_Addition_Workspace
	Float_Multiplication  big.Float_Multiplication_Workspace
	Float_Division        big.Float_Division_Workspace
	Float_Rat             big.Float_Rat_Workspace
	Float_Rat_Status      big.Float_Rat_Status
	Float_Square_Root     big.Float_Square_Root_Workspace
	Float_Gob             [big.FLOAT_GOB_SIZE_MAXIMUM]byte
	Float_Gob_Count       big.Word_Count
	Float_Gob_Workspace   big.Float_Gob_Workspace
	Float_Text_Count      big.Float_Text_Count
	Float_Text_Status     big.Float_Text_Status
	Float_Text_Format     big.Float_Text_Format
	Float_Text_Precision  big.Float_Text_Precision
	Float_Text_Memory     big.Float_Text_Workspace
	Float_Parse_Text      [len("1.5")]byte
	Float_Parse_Base      big.Float_Parse_Base
	Float_Parse_Memory    big.Float_Parse_Workspace
	Float_Encoding        big.Float_64_Value_Bits
	Float_Accuracy        big.Accuracy
	Float_Precision       big.Float_Precision
	Float_Exponent        big.Float_Exponent
	Float_Mode            big.Rounding_Mode
	Text                  [big.INT_TEXT_SIZE_MAXIMUM]byte
	Text_Count            big.Text_Count
	Base                  big.Base
	Text_Workspace        big.Int_Text_Workspace
	Parse_Status          big.Parse_Status
	Parse_Workspace       big.Int_Parse_Workspace
	Shift                 big.Shift_Count
	Byte_Count            big.Byte_Count
	Destination           big.Destination_Status
	Division_Status       big.Division_Status
	Divisor_Status        big.Divisor_Status
	Low_Zero_Count        big.Trailing_Zero_Bit_Count
}

func test_allocation(t *testing.T) {
	var fixture allocation_fixture
	big.Int_Set_Uint_64(&fixture.Random_Maximum, 10)
	fixture.Random_Source[0] = 1
	big.Int_Set_Uint_64(&fixture.Primality_Value, 71)
	copy(fixture.Rat_Parse_Text[:], "-3/4")
	copy(fixture.Rat_Number_Text[:], "-1.5e1")
	copy(fixture.Float_Parse_Text[:], "1.5")
	testify.Zero_Allocation(t, func() {
		allocation_validation(&fixture)
		allocation_conversion(&fixture)
		fixture.Arithmetic = big.Int_Add(&fixture.Result, &fixture.Left, &fixture.Right)
		fixture.Arithmetic = big.Int_Subtract(
			&fixture.Result, &fixture.Left, &fixture.Right,
		)
		fixture.Arithmetic = big.Int_Multiply(
			&fixture.Result, &fixture.Left, &fixture.Right, &fixture.Multiply,
		)
		fixture.Division_Status = big.Int_Quotient_Remainder(
			&fixture.Result, &fixture.Right, &fixture.Left, &fixture.Right,
			&fixture.Division,
		)
		big.Int_Set_Uint_64(&fixture.Right, 7)
		fixture.Divisor_Status = big.Int_Quotient(
			&fixture.Result, &fixture.Left, &fixture.Right, &fixture.Division,
		)
		fixture.Divisor_Status = big.Int_Remainder(
			&fixture.Result, &fixture.Left, &fixture.Right, &fixture.Division,
		)
		fixture.Division_Status = big.Int_Divide_Modulus(
			&fixture.Result, &fixture.Right, &fixture.Left, &fixture.Right,
			&fixture.Division,
		)
		fixture.Shift, fixture.Validation = big.Shift_Count_Validate(1)
		fixture.Bit_Index, fixture.Validation = big.Bit_Index_Validate(1)
		fixture.Arithmetic = big.Int_Shift_Left(
			&fixture.Result, &fixture.Left, fixture.Shift,
		)
		big.Int_Shift_Right(&fixture.Result, &fixture.Left, fixture.Shift)
		big.Int_And(&fixture.Result, &fixture.Left, &fixture.Right, &fixture.Bitwise)
		fixture.Arithmetic = big.Int_And_Not(
			&fixture.Result, &fixture.Left, &fixture.Right, &fixture.Bitwise,
		)
		big.Int_Or(&fixture.Result, &fixture.Left, &fixture.Right, &fixture.Bitwise)
		fixture.Arithmetic = big.Int_Xor(
			&fixture.Result, &fixture.Left, &fixture.Right, &fixture.Bitwise,
		)
		fixture.Arithmetic = big.Int_Not(
			&fixture.Result, &fixture.Left, &fixture.Bitwise,
		)
		fixture.Bit_Value = big.Int_Bit(&fixture.Left, fixture.Bit_Index)
		fixture.Arithmetic = big.Int_Set_Bit(
			&fixture.Result, &fixture.Left, fixture.Bit_Index, big.BIT_SET,
			&fixture.Bitwise,
		)
		allocation_composite(&fixture)
		allocation_float(&fixture)
		fixture.Byte_Count, fixture.Destination = big.Int_Bytes_Into(
			fixture.Validated, &fixture.Left,
		)
		fixture.Destination = big.Int_Fill_Bytes(fixture.Validated, &fixture.Left)
		fixture.Low_Zero_Count = big.Int_Trailing_Zero_Bit_Count(&fixture.Left)
		fixture.Primality, fixture.Primality_Consumed, fixture.Primality_Status =
			big.Int_Probably_Prime(
				&fixture.Primality_Value, test_primality_options(1),
				fixture.Primality_Source[:1],
				&fixture.Primality_Memory,
			)
	})
}

func allocation_validation(fixture *allocation_fixture) {
	fixture.Primality_Options, fixture.Validation = big.Primality_Options_Validate(
		test_primality_options(big.Primality_Repetition_Count_Unvalidated(
			big.PRIMALITY_REPETITION_COUNT_MINIMUM,
		)),
	)
	fixture.Float_Precision, fixture.Validation = big.Float_Precision_Validate(
		big.FLOAT_64_VALUE_MANTISSA_BIT_COUNT,
	)
	fixture.Float_Mode, fixture.Validation = big.Rounding_Mode_Validate(
		big.ROUND_TO_NEAREST_EVEN,
	)
	fixture.Float_Exponent, fixture.Validation = big.Float_Exponent_Validate(
		big.Float_Exponent_Unvalidated(big.FLOAT_EXPONENT_ZERO),
	)
	fixture.Float_Text_Format, fixture.Validation = big.Float_Text_Format_Validate('g')
	fixture.Float_Text_Precision, fixture.Validation = big.Float_Text_Precision_Validate(
		big.FLOAT_TEXT_PRECISION_MINIMUM,
	)
	fixture.Float_Parse_Base, fixture.Validation = big.Float_Parse_Base_Validate(
		big.BASE_AUTOMATIC,
	)
}

func allocation_float(fixture *allocation_fixture) {
	allocation_float_setters(fixture)
	fixture.Validation = big.Float_Set_Float_32_Bits(
		&fixture.Float_Result, fixture.Float_32_Bits,
	)
	fixture.Float_32_Encoding, fixture.Float_Accuracy =
		big.Float_Float_32_Bits(&fixture.Float_Result)
	fixture.Validation = big.Float_Set_Precision(
		&fixture.Float_Result, big.FLOAT_64_VALUE_MANTISSA_BIT_COUNT,
	)
	fixture.Validation = big.Float_Set_Rounding_Mode(
		&fixture.Float_Result, big.ROUND_TO_NEAREST_EVEN,
	)
	fixture.Validation = big.Float_Add(
		&fixture.Float_Result, &fixture.Float_Left, &fixture.Float_Right,
		&fixture.Float_Addition,
	)
	fixture.Validation = big.Float_Subtract(
		&fixture.Float_Result, &fixture.Float_Right, &fixture.Float_Left,
		&fixture.Float_Addition,
	)
	fixture.Validation = big.Float_Multiply(
		&fixture.Float_Result, &fixture.Float_Left, &fixture.Float_Right,
		&fixture.Float_Multiplication,
	)
	fixture.Validation = big.Float_Quotient(
		&fixture.Float_Result, &fixture.Float_Left, &fixture.Float_Right,
		&fixture.Float_Division,
	)
	fixture.Float_Accuracy, fixture.Conversion = big.Float_Int_Into(
		&fixture.Result, &fixture.Float_Result,
	)
	fixture.Int_64, fixture.Float_Accuracy, fixture.Conversion =
		big.Float_Int_64(&fixture.Float_Result)
	fixture.Word_64, fixture.Float_Accuracy, fixture.Conversion =
		big.Float_Uint_64(&fixture.Float_Result)
	big.Rat_Set_Int_64(&fixture.Rat_Left, 1)
	big.Float_Set_Rat(&fixture.Float_Result, &fixture.Rat_Left, &fixture.Float_Rat)
	fixture.Float_Rat_Status = big.Float_Rat_Into(
		&fixture.Rat_Result, &fixture.Float_Result, &fixture.Float_Rat,
	)
	fixture.Validation = big.Float_Square_Root(
		&fixture.Float_Result, &fixture.Float_Left, &fixture.Float_Square_Root,
	)
	big.Float_Absolute(&fixture.Float_Result, &fixture.Float_Left)
	big.Float_Negate(&fixture.Float_Result, &fixture.Float_Result)
	fixture.Order = big.Float_Compare(&fixture.Float_Left, &fixture.Float_Right)
	fixture.Float_Precision = big.Float_Precision_Of(&fixture.Float_Result)
	fixture.Float_Precision = big.Float_Minimum_Precision(&fixture.Float_Result)
	fixture.Float_Mode = big.Float_Rounding_Mode(&fixture.Float_Result)
	fixture.Float_Accuracy = big.Float_Accuracy(&fixture.Float_Result)
	fixture.Sign = big.Float_Sign(&fixture.Float_Result)
	fixture.Boolean = big.Float_Sign_Bit(&fixture.Float_Result)
	fixture.Boolean = big.Float_Is_Infinite(&fixture.Float_Result)
	fixture.Boolean = big.Float_Is_Integer(&fixture.Float_Result)
	fixture.Float_Encoding, fixture.Float_Accuracy =
		big.Float_Float_64_Bits(&fixture.Float_Result)
	allocation_float_representation(fixture)
}

func allocation_float_setters(fixture *allocation_fixture) {
	big.Float_Set_Int_64(&fixture.Float_Left, 1)
	big.Float_Set_Int_64(&fixture.Float_Right, 2)
	big.Float_Set_Uint_64(&fixture.Float_Result, 3)
	big.Float_Set_Int(&fixture.Float_Result, &fixture.Left)
	fixture.Validation = big.Float_Set_Float_64_Bits(
		&fixture.Float_Result, FLOAT_64_THREE_HALVES_BITS,
	)
	big.Float_Set_Infinity(&fixture.Float_Result, false)
	big.Float_Set(&fixture.Float_Result, &fixture.Float_Left)
	big.Float_Copy(&fixture.Float_Result, &fixture.Float_Left)
	fixture.Float_Exponent = big.Float_Mantissa_Exponent(
		&fixture.Float_Left, &fixture.Float_Result,
	)
	fixture.Validation = big.Float_Set_Mantissa_Exponent(
		&fixture.Float_Result, &fixture.Float_Left,
		big.Float_Exponent_Unvalidated(fixture.Float_Exponent),
	)
}

func allocation_float_representation(fixture *allocation_fixture) {
	fixture.Float_Gob_Count, fixture.Destination = big.Float_Gob_Encode_Into(
		fixture.Float_Gob[:], &fixture.Float_Result, &fixture.Float_Gob_Workspace,
	)
	fixture.Float_Text_Count, fixture.Float_Text_Status = big.Float_Text_Into(
		fixture.Text[:], &fixture.Float_Result, 'g',
		big.FLOAT_TEXT_PRECISION_MINIMUM, &fixture.Float_Text_Memory,
	)
	fixture.Float_Parse_Base, fixture.Parse_Status = big.Float_Parse(
		&fixture.Float_Result, fixture.Float_Parse_Text[:], 0,
		&fixture.Float_Parse_Memory,
	)
	float_gob_size := big.FLOAT_GOB_FINITE_PREFIX_SIZE +
		int(fixture.Float_Gob_Count)*big.WORD_BYTE_COUNT
	fixture.Validation = big.Float_Gob_Decode(
		&fixture.Float_Result, fixture.Float_Gob[:float_gob_size],
		&fixture.Float_Gob_Workspace,
	)
}

func allocation_conversion(fixture *allocation_fixture) {
	fixture.Validated, fixture.Validation = big.Bytes_Validate(fixture.Bytes[:])
	big.Int_Set_Int_64(&fixture.Left, -13)
	big.Int_Set_Uint_64(&fixture.Right, 7)
	fixture.Validation = big.Int_Set_Bytes(&fixture.Result, fixture.Bytes[:])
	fixture.Validation = big.Int_Set_Words(&fixture.Result, fixture.Words[:])
	fixture.Encoding_Count, fixture.Destination = big.Int_Gob_Encode_Into(
		fixture.Int_Encoding[:], &fixture.Left,
	)
	fixture.Validation = big.Int_Gob_Decode(
		&fixture.Result, fixture.Int_Encoding[:fixture.Encoding_Count],
	)
	fixture.Word_Count, fixture.Destination = big.Int_Words_Into(
		fixture.Words[:], &fixture.Left,
	)
	big.Int_Set(&fixture.Result, &fixture.Left)
	fixture.Sign = big.Int_Sign(&fixture.Result)
	fixture.Count = big.Int_Bit_Count(&fixture.Result)
	fixture.Boolean = big.Int_Is_Int_64(&fixture.Result)
	fixture.Int_64, fixture.Conversion = big.Int_Int_64(&fixture.Result)
	fixture.Boolean = big.Int_Is_Uint_64(&fixture.Result)
	fixture.Word_64, fixture.Conversion = big.Int_Uint_64(&fixture.Result)
	fixture.Float_64_Bits = FLOAT_64_THREE_HALVES_BITS
	fixture.Validation = big.Int_Set_Float_64_Bits(
		&fixture.Result, fixture.Float_64_Bits,
	)
	fixture.Int_Float_64, fixture.Float_Accuracy =
		big.Int_Float_64_Bits(&fixture.Result)
	big.Int_Absolute(&fixture.Result, &fixture.Left)
	big.Int_Negate(&fixture.Result, &fixture.Result)
	fixture.Order = big.Int_Compare(&fixture.Left, &fixture.Right)
	fixture.Order = big.Int_Compare_Absolute(&fixture.Left, &fixture.Right)
}

func allocation_composite(fixture *allocation_fixture) {
	fixture.Random_Consumed, fixture.Random_Status = big.Int_Random(
		&fixture.Result, &fixture.Random_Maximum, fixture.Random_Source[:1],
		&fixture.Random_Memory,
	)
	fixture.Jacobi, fixture.Jacobi_Status = big.Int_Jacobi(
		&fixture.Left, &fixture.Right, &fixture.Jacobi_Memory,
	)
	fixture.Arithmetic = big.Int_Exponent(
		&fixture.Result, &fixture.Left, &fixture.Right, &fixture.Exponent,
	)
	big.Int_Set_Uint_64(&fixture.Right, 7)
	fixture.Divisor_Status = big.Int_Modular_Multiply(
		&fixture.Result, &fixture.Left, &fixture.Right, &fixture.Right,
		&fixture.Modular,
	)
	fixture.Modular_Status = big.Int_Modular_Inverse(
		&fixture.Result, &fixture.Left, &fixture.Right, &fixture.Modular,
	)
	fixture.Modular_Status = big.Int_Modular_Exponent(
		&fixture.Result, &fixture.Left, &fixture.Right, &fixture.Right,
		&fixture.Modular,
	)
	big.Int_Set_Uint_64(&fixture.Left, 10)
	big.Int_Set_Uint_64(&fixture.Right, 13)
	big.Int_Set_Uint_64(&fixture.Square_Root_Candidate, 2)
	fixture.Square_Root_Status = big.Int_Modular_Square_Root(
		&fixture.Result, &fixture.Left, &fixture.Right,
		&fixture.Square_Root_Candidate, &fixture.Modular_Square_Root,
	)
	fixture.Arithmetic = big.Int_Multiply_Range(
		&fixture.Result, 2, 5, &fixture.Product,
	)
	fixture.Arithmetic = big.Int_Binomial(
		&fixture.Result, 10, 3, &fixture.Product,
	)
	big.Int_Greatest_Common_Divisor(
		&fixture.Result, &fixture.Left, &fixture.Right, &fixture.Greatest_Common,
	)
	fixture.Validation = big.Int_Square_Root(
		&fixture.Result, &fixture.Right, &fixture.Square_Root,
	)
	allocation_rational(fixture)
	allocation_text(fixture)
}

func allocation_text(fixture *allocation_fixture) {
	fixture.Base, fixture.Validation = big.Base_Validate(10)
	fixture.Text_Count, fixture.Destination = big.Int_Text_Into(
		fixture.Text[:], &fixture.Left, fixture.Base, &fixture.Text_Workspace,
	)
	fixture.Text[ZERO_INDEX] = '7'
	fixture.Base, fixture.Parse_Status = big.Int_Parse(
		&fixture.Result, fixture.Text[:INDEX_STEP], 10,
		&fixture.Parse_Workspace,
	)
}

func allocation_rational(fixture *allocation_fixture) {
	big.Rat_Set_Int_64(&fixture.Rat_Left, -3)
	big.Rat_Set_Uint_64(&fixture.Rat_Right, 4)
	fixture.Arithmetic = big.Rat_Set_Int(&fixture.Rat_Result, &fixture.Left)
	big.Rat_Set(&fixture.Rat_Result, &fixture.Rat_Left)
	fixture.Rat_Division = big.Rat_Set_Fraction(
		&fixture.Rat_Result, &fixture.Left, &fixture.Right, &fixture.Rat_Workspace,
	)
	fixture.Divisor_Status = big.Rat_Set_Fraction_64(
		&fixture.Rat_Result, 3, 4, &fixture.Rat_Workspace,
	)
	fixture.Divisor_Status = big.Rat_Inverse(
		&fixture.Rat_Result, &fixture.Rat_Left, &fixture.Rat_Workspace,
	)
	fixture.Sign = big.Rat_Sign(&fixture.Rat_Result)
	fixture.Boolean = big.Rat_Is_Integer(&fixture.Rat_Result)
	big.Rat_Numerator_Into(&fixture.Result, &fixture.Rat_Result)
	big.Rat_Denominator_Into(&fixture.Result, &fixture.Rat_Result)
	big.Rat_Absolute(&fixture.Rat_Result, &fixture.Rat_Left)
	big.Rat_Negate(&fixture.Rat_Result, &fixture.Rat_Result)
	fixture.Order = big.Rat_Compare(
		&fixture.Rat_Left, &fixture.Rat_Right, &fixture.Rat_Workspace,
	)
	fixture.Rat_Arithmetic = big.Rat_Add(
		&fixture.Rat_Result, &fixture.Rat_Left, &fixture.Rat_Right,
		&fixture.Rat_Workspace,
	)
	fixture.Rat_Arithmetic = big.Rat_Subtract(
		&fixture.Rat_Result, &fixture.Rat_Left, &fixture.Rat_Right,
		&fixture.Rat_Workspace,
	)
	fixture.Rat_Arithmetic = big.Rat_Multiply(
		&fixture.Rat_Result, &fixture.Rat_Left, &fixture.Rat_Right,
		&fixture.Rat_Workspace,
	)
	fixture.Rat_Division = big.Rat_Quotient(
		&fixture.Rat_Result, &fixture.Rat_Left, &fixture.Rat_Right,
		&fixture.Rat_Workspace,
	)
	fixture.Rat_Fraction, fixture.Destination = big.Rat_Text_Into(
		fixture.Rat_Text[:], &fixture.Rat_Result, &fixture.Rat_Text_Memory,
	)
	fixture.Rat_Text_Count, fixture.Destination = big.Rat_Rational_Text_Into(
		fixture.Rat_Text[:], &fixture.Rat_Result, &fixture.Rat_Text_Memory,
	)
	fixture.Rat_Precision, fixture.Validation = big.Rat_Precision_Validate(2)
	fixture.Rat_Text_Count, fixture.Destination = big.Rat_Float_Text_Into(
		fixture.Rat_Text[:], &fixture.Rat_Result, fixture.Rat_Precision,
		&fixture.Rat_Float_Memory,
	)
	fixture.Rat_Places, fixture.Rat_Exact = big.Rat_Float_Precision(
		&fixture.Rat_Result, &fixture.Rat_Place_Memory,
	)
	allocation_rational_binary(fixture)
	fixture.Rat_Parse_Status = big.Rat_Parse_Fraction(
		&fixture.Rat_Result, fixture.Rat_Parse_Text[:], &fixture.Rat_Parse_Memory,
	)
	fixture.Rat_Number_Status = big.Rat_Parse(
		&fixture.Rat_Result, fixture.Rat_Number_Text[:], &fixture.Rat_Number_Memory,
	)
	allocation_rational_serialization(fixture)
}

func allocation_rational_binary(fixture *allocation_fixture) {
	fixture.Float_64_Bits = FLOAT_64_THREE_HALVES_BITS
	fixture.Validation = big.Rat_Set_Float_64_Bits(
		&fixture.Rat_Result, fixture.Float_64_Bits,
	)
	fixture.Rat_Float_64, fixture.Rat_Float_Exact = big.Rat_Float_64_Bits(
		&fixture.Rat_Result, &fixture.Rat_Float_64_Memory,
	)
	fixture.Float_32_Bits = FLOAT_32_THREE_HALVES_BITS
	fixture.Validation = big.Rat_Set_Float_32_Bits(
		&fixture.Rat_Result, fixture.Float_32_Bits,
	)
	fixture.Rat_Float_32, fixture.Rat_Float_Exact = big.Rat_Float_32_Bits(
		&fixture.Rat_Result, &fixture.Rat_Float_32_Memory,
	)
}

func allocation_rational_serialization(fixture *allocation_fixture) {
	fixture.Rat_Encoding, fixture.Destination = big.Rat_Gob_Encode_Into(
		fixture.Rat_Gob[:], &fixture.Rat_Result,
	)
	fixture.Validation = big.Rat_Gob_Decode(
		&fixture.Rat_Result, fixture.Rat_Gob[:fixture.Rat_Encoding],
		&fixture.Rat_Gob_Memory,
	)
}
