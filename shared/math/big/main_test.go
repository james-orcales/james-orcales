package big_test

import (
	"testing"

	"local/james-orcales/shared/math/big"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/testify"
)

func TestMain(m *testing.M) {
	aver.Run_Test_Main(m)
}

func test_float_text_fraction_retained_count(t *testing.T) {
	var value big.Float
	test_float_initialize(&value)
	big.Float_Set_Int_64(&value, 3)
	value.Exponent = 1
	var workspace big.Float_Text_Workspace
	test_float_text_workspace_initialize(&workspace)
	output := make(big.Text, big.FLOAT_TEXT_SIZE_MAXIMUM)
	for _, retained := range []big.Float_Text_Count{
		1, 2, big.Float_Text_Count(big.FLOAT_TEXT_SIZE_MAXIMUM),
	} {
		for _, one := range []struct {
			Format   big.Float_Text_Format_Unvalidated
			Expected string
		}{
			{'b', "13835058055282163712p-63"},
			{'p', "0x.cp+1"},
			{'x', "0x1.8p+00"},
			{'e', "1.5e+00"},
			{'f', "1.5"},
			{'g', "1.5"},
		} {
			workspace.Control.Output_Count = retained
			count, status := big.Float_Text_Into(
				output, &value, one.Format, -1, &workspace)
			testify.Equal_Values(t, big.STATUS_OK, status)
			testify.Equal(t, one.Expected, string(output[:count]))
		}
	}
}

func test_float_text_zero_precision_maximum(t *testing.T) {
	var value big.Float
	expected := make([]byte, big.FLOAT_TEXT_PRECISION_MAXIMUM+8)
	copy(expected, "0x0.")
	for index := 4; index < len(expected)-4; index++ {
		expected[index] = '0'
	}
	copy(expected[len(expected)-4:], "p+00")
	var workspace big.Float_Text_Workspace
	test_float_text_workspace_initialize(&workspace)
	output := make(big.Text, big.FLOAT_TEXT_SIZE_MAXIMUM)
	count, status := big.Float_Text_Into(
		output, &value, 'x', big.FLOAT_TEXT_PRECISION_MAXIMUM, &workspace,
	)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, string(expected), string(output[:count]))
	big.Float_Negate(&value, &value)
	count, status = big.Float_Text_Into(
		output, &value, 'x', big.FLOAT_TEXT_PRECISION_MAXIMUM, &workspace,
	)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, "-"+string(expected), string(output[:count]))
	value.Mantissa.Words = make(big.Words, 1)
	big.Float_Set_Int_64(&value, -1)
	expected[2] = '1'
	count, status = big.Float_Text_Into(
		output, &value, 'x', big.FLOAT_TEXT_PRECISION_MAXIMUM, &workspace,
	)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, "-"+string(expected), string(output[:count]))
}

func test_square_root_double_word_destinations(
	t *testing.T, source *big.Int, expected *big.Int,
	workspace *big.Int_Square_Root_Workspace,
) {
	for _, initial := range int_domain_values(t) {
		result := initial
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Square_Root(&result, source, workspace))
		testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&result, expected))
	}
}

func test_square_root_storage(t *testing.T) {
	test_square_root_double_word_digits(t)
	var workspace big.Int_Square_Root_Workspace
	test_square_root_workspace_initialize(&workspace)
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		workspace.Integers.Current = big.Int{Words: make(big.Words, size)}
		source := big.Int{Words: make(big.Words, size)}
		result := big.Int{Words: make(big.Words, size)}
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Square_Root(&result, &source, &workspace))
		test_bitwise_result(t, &result, 0)
		if size != 0 {
			big.Int_Set_Int_64(&source, 17)
			testify.Equal_Values(t, big.STATUS_OK,
				big.Int_Square_Root(&result, &source, &workspace))
			test_bitwise_result(t, &result, 4)
			wide := big.Int{Words: big.Words{0, 1}, Count: 2}
			testify.Equal_Values(t, big.STATUS_OK,
				big.Int_Square_Root(&result, &wide, &workspace))
			test_bitwise_result(t, &result, 4294967296)
		}
	}
}

func text_expect(
	t *testing.T, value int64, base_value big.Base_Unvalidated, want string,
	workspace *big.Int_Text_Workspace,
) {
	base, status := big.Base_Validate(base_value)
	testify.Equal_Values(t, big.STATUS_OK, status)
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		if size == 0 {
			if value != 0 {
				continue
			}
		}
		integer := big.Int{Words: make(big.Words, size)}
		big.Int_Set_Int_64(&integer, big.Int_64(value))
		var destination [big.WORD_BIT_COUNT]byte
		count, destination_status := big.Int_Text_Into(
			destination[:], &integer, base, workspace,
		)
		testify.Equal_Values(t, big.STATUS_OK, destination_status)
		testify.Equal(t, want, string(destination[:count]))
	}
}

func test_rat_quotient_retained(t *testing.T, value *big.Rat, workspace *big.Rat_Workspace) {
	test_rat_set_retained(t, value, workspace)
	test_rat_arithmetic_retained(t, value, workspace)
	var result big.Rat
	test_rat_initialize(&result)
	expected_sign := big.Int_Sign((*big.Int)(&value.Integers.Numerator))
	testify.Equal(t, expected_sign, big.Rat_Sign(value))
	numerator := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	big.Rat_Numerator_Into(&numerator, value)
	testify.Equal(t, big.ORDER_SAME,
		big.Int_Compare(&numerator, (*big.Int)(&value.Integers.Numerator)))
	big.Rat_Set(&result, value)
	testify.Equal_Values(t, big.STATUS_OK, big.Rat_Quotient(&result, value, value, workspace))
	rat_expect(t, &result, 1, 1)
	testify.Equal_Values(t, big.STATUS_OK, big.Rat_Inverse(&result, value, workspace))
	testify.Equal(t, expected_sign, big.Rat_Sign(&result))
	big.Rat_Numerator_Into(&numerator, &result)
	testify.Equal(t, big.ORDER_SAME,
		big.Int_Compare(&numerator, (*big.Int)(&result.Integers.Numerator)))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Rat_Quotient(&result, &result, &result, workspace))
	rat_expect(t, &result, 1, 1)
}

func test_rat_arithmetic_retained(t *testing.T, value *big.Rat, workspace *big.Rat_Workspace) {
	var zero big.Rat
	testify.Equal_Values(t, big.STATUS_DIVISOR_ZERO, big.Rat_Inverse(&zero, &zero, workspace))
	testify.Equal(t, big.SIGN_ZERO, big.Rat_Sign(&zero))
	var result, one big.Rat
	test_rat_initialize(&result)
	test_rat_initialize(&one)
	big.Rat_Set_Int_64(&one, 1)
	workspace.Integers.Result_Denominator.Count = big.Word_Count(big.WORD_COUNT_MAXIMUM)
	workspace.Integers.Result_Denominator.Words[big.WORD_COUNT_MAXIMUM-1] = 1
	testify.Equal(t, big.ORDER_SAME, big.Rat_Compare(&one, &one, workspace))
	big.Rat_Absolute(&result, value)
	test_bitwise_result(t, (*big.Int)(&result.Integers.Numerator), 1)
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(
		(*big.Int)(&result.Integers.Denominator), (*big.Int)(&value.Integers.Denominator)))
	testify.Equal_Values(t, big.STATUS_OK, big.Rat_Inverse(&result, value, workspace))
	big.Rat_Absolute(&result, &result)
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(
		(*big.Int)(&result.Integers.Numerator), (*big.Int)(&value.Integers.Denominator)))
	testify.Equal(t, big.Word_Count(0), result.Integers.Denominator.Count)
	var precision_workspace big.Rat_Float_Precision_Workspace
	precision_workspace.Integers.Denominator.Words = make(big.Words, 1)
	places, exact := big.Rat_Float_Precision(&result, &precision_workspace)
	testify.Equal(t, big.Decimal_Place_Count(0), places)
	testify.True(t, bool(exact))
	testify.Equal_Values(t, big.STATUS_OK, big.Rat_Add(&result, &one, &one, workspace))
	rat_expect(t, &result, 2, 1)
	big.Rat_Set(&result, value)
	testify.Equal_Values(t, big.STATUS_OK, big.Rat_Multiply(&result, &one, &one, workspace))
	rat_expect(t, &result, 1, 1)
}

func test_rat_set_retained(t *testing.T, value *big.Rat, workspace *big.Rat_Workspace) {
	var result big.Rat
	test_rat_initialize(&result)
	integer := big.Int{Words: big.Words{7}, Count: 1}
	big.Rat_Set(&result, value)
	testify.Equal_Values(t, big.STATUS_OK, big.Rat_Set_Int(&result, &integer))
	rat_expect(t, &result, 7, 1)
	big.Rat_Set(&result, value)
	big.Rat_Set_Int_64(&result, -7)
	rat_expect(t, &result, -7, 1)
	testify.Equal_Values(t, big.STATUS_OK, big.Rat_Inverse(&result, value, workspace))
	big.Rat_Set_Int_64(&result, 7)
	rat_expect(t, &result, 7, 1)
}

func test_rat_float_64_double_word(t *testing.T) {
	var value big.Rat
	test_rat_initialize(&value)
	value.Integers.Numerator.Words[0], value.Integers.Numerator.Words[1] = 1, 1
	value.Integers.Numerator.Count = 2
	value.Integers.Denominator.Words[1] = 1
	value.Integers.Denominator.Count = 2
	var workspace big.Rat_Float_64_Workspace
	test_rat_float_64_workspace_initialize(&workspace)
	// 2^-64 is below half an ulp at one, but exactness must remain false.
	encoding, exact := big.Rat_Float_64_Bits(&value, &workspace)
	testify.Equal(t, big.Float_64_Value_Bits(FLOAT_64_ONE_BITS), encoding)
	testify.False(t, bool(exact))
}

func test_rat_serialization_double_word(t *testing.T) {
	var value big.Rat
	test_rat_initialize(&value)
	value.Integers.Numerator.Words[0], value.Integers.Numerator.Words[1] = 1, 1
	value.Integers.Numerator.Count = 2
	value.Integers.Denominator.Words[1] = 1
	value.Integers.Denominator.Count = 2
	expected := []byte{
		2, 0, 0, 0, 9,
		1, 0, 0, 0, 0, 0, 0, 0, 1,
		1, 0, 0, 0, 0, 0, 0, 0, 0,
	}
	var storage [big.RAT_GOB_SIZE_MAXIMUM]byte
	count, status := big.Rat_Gob_Encode_Into(storage[:], &value)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, expected, storage[:count])
	big.Rat_Negate(&value, &value)
	expected[0] = 3
	count, status = big.Rat_Gob_Encode_Into(storage[:], &value)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, expected, storage[:count])
}

func test_rat_text_double_word(t *testing.T) {
	var value big.Rat
	test_rat_initialize(&value)
	value.Integers.Numerator.Words[0], value.Integers.Numerator.Words[1] = 1, 1
	value.Integers.Numerator.Count = 2
	value.Integers.Denominator.Words[1] = 1
	value.Integers.Denominator.Count = 2
	const TEXT = "18446744073709551617/18446744073709551616"
	rat_text_expect(t, &value, TEXT, TEXT)
	big.Rat_Negate(&value, &value)
	rat_text_expect(t, &value, "-"+TEXT, "-"+TEXT)
}

func test_greatest_common_divisor_coprime_words(
	t *testing.T, workspace *big.Int_Greatest_Common_Divisor_Workspace,
) {
	test_greatest_common_divisor_maximum_difference(t, workspace)
	left := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM), Count: 2}
	right := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM), Count: 2}
	result := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	left.Words[0], left.Words[1] = 1, 3
	right.Words[0], right.Words[1] = 1, 2
	// Difference is a power of two; both operands are odd, so their GCD is one.
	big.Int_Greatest_Common_Divisor(&result, &left, &right, workspace)
	test_bitwise_result(t, &result, 1)
	// Consecutive values straddle the one-word boundary before Euclidean reduction.
	left.Words[0], left.Words[1] = 0, 1
	right.Words[0], right.Words[1], right.Count = big.Word(bits.WORD_64_MAXIMUM), 0, 1
	big.Int_Greatest_Common_Divisor(&result, &left, &right, workspace)
	test_bitwise_result(t, &result, 1)
	left.Words[0], left.Words[1], left.Words[2], left.Count = 0, 0, 1, 3
	right.Words[0], right.Words[1] = 0, 1
	right.Count = 2
	big.Int_Greatest_Common_Divisor(&result, &left, &right, workspace)
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&result, &right))
}

func test_greatest_common_divisor_maximum_difference(
	t *testing.T, workspace *big.Int_Greatest_Common_Divisor_Workspace,
) {
	left := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM),
		Count: big.WORD_COUNT_MAXIMUM}
	right := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM), Count: left.Count}
	result := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	left.Words[0], right.Words[0] = 1, 1
	left.Words[big.WORD_COUNT_MAXIMUM-1] = 3
	right.Words[big.WORD_COUNT_MAXIMUM-1] = 2
	// Odd operands differ by a power of two, so their only common divisor is one.
	big.Int_Greatest_Common_Divisor(&result, &left, &right, workspace)
	test_bitwise_result(t, &result, 1)
	// 2^64 is one modulo three, so 4*B^511+1 is coprime to three.
	left.Words[big.WORD_COUNT_MAXIMUM-1] = 4
	right.Count, right.Words[0] = 1, 3
	big.Int_Greatest_Common_Divisor(&result, &left, &right, workspace)
	test_bitwise_result(t, &result, 1)
	// L-B*R=1-B and R is one modulo B-1, hence gcd(L,R)=1.
	left.Count, right.Count = 3, 2
	left.Words[2] = big.Word(bits.WORD_64_MAXIMUM)
	right.Words[0], right.Words[1] = 1, big.Word(bits.WORD_64_MAXIMUM)
	big.Int_Greatest_Common_Divisor(&result, &left, &right, workspace)
	test_bitwise_result(t, &result, 1)
}

func test_rat_unit_denominator_storage(t *testing.T) {
	var workspace big.Rat_Workspace
	test_rat_workspace_initialize(&workspace)
	half := rat_set_pair(t, 1, 2, &workspace)
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		var result big.Rat
		result.Integers.Numerator.Words = make(big.Words, 1)
		result.Integers.Denominator.Words = make(big.Words, size)
		testify.Equal_Values(t, big.STATUS_OK,
			big.Rat_Add(&result, &half, &half, &workspace))
		rat_expect(t, &result, 1, 1)
		testify.Equal_Values(t, big.STATUS_OK,
			big.Rat_Subtract(&result, &half, &half, &workspace))
		rat_expect(t, &result, 0, 1)
	}
}

func test_words_storage(t *testing.T) {
	var zero big.Int
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Words(&zero, big.Words_Unvalidated{0}))
	test_bitwise_result(t, &zero, 0)
	testify.Equal(t, big.Trailing_Zero_Bit_Count(0), big.Int_Trailing_Zero_Bit_Count(&zero))
	big.Int_Negate(&zero, &zero)
	testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&zero))
	filled := big.Bytes{1, 2}
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Fill_Bytes(filled, &zero))
	testify.Equal(t, big.Bytes{0, 0}, filled)
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		value := big.Int{Words: make(big.Words, size)}
		output := make(big.Words, size)
		count, status := big.Int_Words_Into(output, &value)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal_Values(t, 0, count)
		if size != 0 {
			big.Int_Set_Int_64(&value, -7)
			count, status = big.Int_Words_Into(output, &value)
			testify.Equal_Values(t, big.STATUS_OK, status)
			testify.Equal_Values(t, 1, count)
			testify.Equal_Values(t, 7, output[0])
		}
	}
}

func test_multiply_range_storage(t *testing.T, workspace *big.Int_Product_Workspace) {
	var zero big.Int
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Multiply_Range(&zero, -1, 1, workspace))
	testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&zero))
	result := big.Int{Words: make(big.Words, 2)}
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Multiply_Range(&result, 2, 5, workspace))
	test_bitwise_result(t, &result, 120)
}

func test_random_storage(t *testing.T) {
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		maximum := big.Int{Words: make(big.Words, size)}
		result := big.Int{Words: make(big.Words, size)}
		var workspace big.Int_Random_Workspace
		workspace.Integers.Candidate.Words = make(big.Words, size)
		consumed, status := big.Int_Random(&result, &maximum, nil, &workspace)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal_Values(t, 0, consumed)
		test_bitwise_result(t, &result, 0)
		if size != 0 {
			big.Int_Set_Int_64(&maximum, 10)
			consumed, status = big.Int_Random(
				&result, &maximum, []big.Word{7}, &workspace)
			testify.Equal_Values(t, big.STATUS_OK, status)
			testify.Equal_Values(t, 1, consumed)
			test_bitwise_result(t, &result, 7)
		}
	}
}

func test_primality_storage(t *testing.T, workspace *big.Int_Primality_Workspace) {
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		value := big.Int{Words: make(big.Words, size)}
		prime, consumed, status := big.Int_Probably_Prime(
			&value, test_primality_options(0), nil, workspace)
		testify.False(t, bool(prime))
		testify.Equal_Values(t, 0, consumed)
		testify.Equal_Values(t, big.STATUS_OK, status)
		if size != 0 {
			big.Int_Set_Int_64(&value, 97)
			prime, consumed, status = big.Int_Probably_Prime(
				&value, test_primality_options(0), nil, workspace)
			testify.True(t, bool(prime))
			testify.Equal_Values(t, 0, consumed)
			testify.Equal_Values(t, big.STATUS_OK, status)
		}
	}
}

func test_not_storage(t *testing.T, workspace *big.Int_Bitwise_Workspace) {
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		value := big.Int{Words: make(big.Words, size)}
		result := big.Int{Words: make(big.Words, max(1, size))}
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Not(&result, &value, workspace))
		test_bitwise_result(t, &result, -1)
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Not(&value, &result, workspace))
		test_bitwise_result(t, &value, 0)
		if size != 0 {
			for _, source := range []int64{-7, -1, 0, 1, 7} {
				big.Int_Set_Int_64(&value, big.Int_64(source))
				testify.Equal_Values(t, big.STATUS_OK,
					big.Int_Not(&result, &value, workspace))
				test_bitwise_result(t, &result, ^source)
			}
		}
	}
}

func test_multiplication_storage(t *testing.T) {
	workspace := big.Int_Multiplication_Workspace{
		Product: make(big.Int_Product_Words, big.WORD_COUNT_MAXIMUM),
	}
	factor := big.Int{Words: big.Words{0, 1}, Count: 2}
	product := big.Int{Words: make(big.Words, 3)}
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Multiply(&product, &factor, &factor, &workspace))
	testify.Equal_Values(t, 3, product.Count)
	testify.Equal_Values(t, 0, product.Words[0])
	testify.Equal_Values(t, 0, product.Words[1])
	testify.Equal_Values(t, 1, product.Words[2])
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		left := big.Int{Words: make(big.Words, size)}
		right := big.Int{Words: make(big.Words, size)}
		result := big.Int{Words: make(big.Words, size)}
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Multiply(&result, &left, &right, &workspace))
		testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&result))
		if size != 0 {
			big.Int_Set_Int_64(&left, 6)
			big.Int_Set_Int_64(&right, -7)
			testify.Equal_Values(t, big.STATUS_OK,
				big.Int_Multiply(&result, &left, &right, &workspace))
			test_bitwise_result(t, &result, -42)
		}
	}
}

func test_binomial_storage(t *testing.T, workspace *big.Int_Product_Workspace) {
	maximum := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	// Central coefficient reaches the largest divisor before bounded accumulation overflows.
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Binomial(&maximum, 32774, 16387, workspace))
	testify.Equal_Values(t, 32767, big.Int_Bit_Count(&maximum))
	testify.Equal_Values(t, 3, big.Int_Trailing_Zero_Bit_Count(&maximum))
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW,
		big.Int_Binomial(&maximum, 32776, 16387, workspace))
	testify.Equal_Values(t, 32767, big.Int_Bit_Count(&maximum))
	wide := big.Int{Words: make(big.Words, 2)}
	// C(2^33, 2) = 2^65 - 2^32, so division by two survives scalar reduction.
	expected := big.Int{Words: big.Words{big.Word(bits.WORD_64_MAXIMUM -
		uint64(bits.WORD_32_MAXIMUM)), 1}, Count: 2}
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Binomial(&wide, 8589934592, 2, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&wide, &expected))
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		result := big.Int{Words: make(big.Words, size)}
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Binomial(&result, 3, 4, workspace))
		testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&result))
		if size != 0 {
			testify.Equal_Values(t, big.STATUS_OK,
				big.Int_Binomial(&result, 5, 2, workspace))
			converted, status := big.Int_Int_64(&result)
			testify.Equal_Values(t, big.STATUS_OK, status)
			testify.Equal(t, big.Int_64(10), converted)
		}
	}
}

func test_modular_square_root_storage(t *testing.T) {
	var workspace big.Int_Modular_Square_Root_Workspace
	test_modular_square_root_workspace_initialize(&workspace)
	test_modular_square_root_negative_nonresidue(t, &workspace)
	test_modular_square_root_double_word(t, &workspace)
	test_modular_square_root_order_one_storage(t, &workspace)
	test_modular_square_root_maximum_factor(t, &workspace)
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		value := big.Int{Words: make(big.Words, size)}
		modulus := big.Int{Words: make(big.Words, size)}
		nonresidue := big.Int{Words: make(big.Words, size)}
		result := big.Int{Words: make(big.Words, size)}
		testify.Equal_Values(t, big.STATUS_INPUT_INVALID,
			big.Int_Modular_Square_Root(
				&result, &value, &modulus, &nonresidue, &workspace))
		if size == 0 {
			modulus.Words, nonresidue.Words = make(big.Words, 1), make(big.Words, 1)
		}
		big.Int_Set_Int_64(&modulus, 17)
		big.Int_Set_Int_64(&nonresidue, 3)
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Modular_Square_Root(
				&result, &value, &modulus, &nonresidue, &workspace))
		testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&result))
		if size != 0 {
			big.Int_Set_Int_64(&value, 4)
			testify.Equal_Values(t, big.STATUS_OK,
				big.Int_Modular_Square_Root(
					&result, &value, &modulus, &nonresidue, &workspace))
			root, status := big.Int_Int_64(&result)
			testify.Equal_Values(t, big.STATUS_OK, status)
			testify.True(t, root == 2 || root == 15)
		}
	}
}

func test_modular_square_root_negative_nonresidue(
	t *testing.T, workspace *big.Int_Modular_Square_Root_Workspace,
) {
	value := big.Int{Words: big.Words{1}, Count: 1}
	modulus := big.Int{Words: big.Words{3}, Count: 1}
	nonresidue := big.Int{Words: big.Words{1, 3}, Count: 2,
		Negative: big.POLARITY_NEGATIVE}
	result := big.Int{Words: make(big.Words, 1)}
	// -(3*2^64+1) is two modulo three, hence a quadratic nonresidue.
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Modular_Square_Root(&result, &value, &modulus, &nonresidue, workspace))
	root, status := big.Int_Int_64(&result)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.True(t, root == 1 || root == 2)
}

func test_modular_square_root_maximum_factor(
	t *testing.T, workspace *big.Int_Modular_Square_Root_Workspace,
) {
	modulus := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM),
		Count: big.WORD_COUNT_MAXIMUM}
	value := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM),
		Count: big.WORD_COUNT_MAXIMUM}
	for index := range modulus.Words {
		modulus.Words[index] = big.Word(bits.WORD_64_MAXIMUM)
		value.Words[index] = big.Word(bits.WORD_64_MAXIMUM)
	}
	modulus.Words[0] -= 2
	value.Words[0] -= 3
	modulus.Words[big.WORD_COUNT_MAXIMUM-1] = 5
	value.Words[big.WORD_COUNT_MAXIMUM-1] = 5
	nonresidue := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM),
		Count: big.WORD_COUNT_MAXIMUM}
	nonresidue.Words[big.WORD_COUNT_MAXIMUM-1] = 2
	result := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	// -1 has Jacobi symbol one here but cannot be a square modulo the factor three.
	testify.Equal_Values(t, big.STATUS_RESULT_ABSENT,
		big.Int_Modular_Square_Root(&result, &value, &modulus, &nonresidue, workspace))
	testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&result))
}

func test_modular_square_root_double_word(
	t *testing.T, workspace *big.Int_Modular_Square_Root_Workspace,
) {
	value := big.Int{Words: big.Words{4}, Count: 1}
	modulus := big.Int{Words: big.Words{13, 1}, Count: 2}
	nonresidue := big.Int{Words: big.Words{2}, Count: 1}
	result := big.Int{Words: make(big.Words, 2)}
	positive := big.Int{Words: big.Words{2}, Count: 1}
	negative := big.Int{Words: big.Words{11, 1}, Count: 2}
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Modular_Square_Root(&result, &value, &modulus, &nonresidue, workspace))
	testify.True(t, big.Int_Compare(&result, &positive) == big.ORDER_SAME ||
		big.Int_Compare(&result, &negative) == big.ORDER_SAME)
}

func test_modular_square_root_order_one_storage(
	t *testing.T, workspace *big.Int_Modular_Square_Root_Workspace,
) {
	value := big.Int{Words: big.Words{3}, Count: 1}
	modulus := big.Int{Words: big.Words{7}, Count: 1}
	nonresidue := big.Int{Words: big.Words{3}, Count: 1}
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		result := big.Int{Words: make(big.Words, size)}
		value.Words[0] = 3
		modulus.Words[0], nonresidue.Words[0] = 7, 3
		testify.Equal_Values(t, big.STATUS_RESULT_ABSENT,
			big.Int_Modular_Square_Root(
				&result, &value, &modulus, &nonresidue, workspace))
		testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&result))
		// Jacobi symbol alone cannot prove a root exists for composite modulus.
		value.Words[0], modulus.Words[0], nonresidue.Words[0] = 2, 15, 7
		testify.Equal_Values(t, big.STATUS_RESULT_ABSENT,
			big.Int_Modular_Square_Root(
				&result, &value, &modulus, &nonresidue, workspace))
		testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&result))
		value.Words[0], modulus.Words[0], nonresidue.Words[0] = 5, 21, 2
		testify.Equal_Values(t, big.STATUS_RESULT_ABSENT,
			big.Int_Modular_Square_Root(
				&result, &value, &modulus, &nonresidue, workspace))
		testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&result))
		if size != 0 {
			value.Words[0] = 4
			modulus.Words[0], nonresidue.Words[0] = 7, 3
			testify.Equal_Values(t, big.STATUS_OK,
				big.Int_Modular_Square_Root(
					&result, &value, &modulus, &nonresidue, workspace))
			root, status := big.Int_Int_64(&result)
			testify.Equal_Values(t, big.STATUS_OK, status)
			testify.True(t, root == 2 || root == 5)
		}
	}
}

func test_rat_float_text_storage(
	t *testing.T, value *big.Rat, workspace *big.Rat_Float_Text_Workspace,
) {
	output := make(big.Text, 4)
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		workspace.Exponent.Integers.Result = big.Int{Words: make(big.Words, size)}
		count, status := big.Rat_Float_Text_Into(output, value, 2, workspace)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, "0.33", string(output[:count]))
	}
}

func test_rat_float_text_wide(t *testing.T) {
	var workspace big.Rat_Float_Text_Workspace
	test_rat_float_text_workspace_initialize(&workspace)
	var value big.Rat
	test_rat_initialize(&value)
	value.Integers.Numerator.Count = 1
	value.Integers.Numerator.Words[0] = 1
	output := make(big.Text, 32)
	for _, size := range []int{2, big.RAT_WORD_COUNT_MAXIMUM} {
		clear(value.Integers.Denominator.Words)
		value.Integers.Denominator.Count = big.Word_Count(size)
		value.Integers.Denominator.Words[size-1] = 1
		count, status := big.Rat_Float_Text_Into(output, &value, 2, &workspace)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, "0.00", string(output[:count]))
	}
	clear(value.Integers.Denominator.Words)
	value.Integers.Denominator.Count = 1
	value.Integers.Denominator.Words[0] = 3
	value.Integers.Numerator.Count = 2
	value.Integers.Numerator.Words[0], value.Integers.Numerator.Words[1] = 0, 1
	count, status := big.Rat_Float_Text_Into(output, &value, 2, &workspace)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, "6148914691236517205.33", string(output[:count]))
}

func test_rat_integer_storage(t *testing.T, workspace *big.Rat_Workspace) {
	var value big.Rat
	test_rat_initialize(&value)
	for _, size := range []int{1, 2, big.WORD_COUNT_MAXIMUM} {
		numerator := big.Int{Words: make(big.Words, size)}
		denominator := big.Int{Words: make(big.Words, size)}
		result := big.Int{Words: make(big.Words, size)}
		big.Int_Set_Int_64(&numerator, -6)
		testify.Equal_Values(t, big.STATUS_OK, big.Rat_Set_Int(&value, &numerator))
		big.Rat_Numerator_Into(&result, &value)
		testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&result, &numerator))
		big.Rat_Denominator_Into((*big.Nonempty_Int)(&result), &value)
		converted, status := big.Int_Int_64(&result)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, big.Int_64(1), converted)
		big.Int_Set_Int_64(&denominator, 4)
		testify.Equal_Values(t, big.STATUS_OK,
			big.Rat_Set_Fraction(&value, &numerator, &denominator, workspace))
		big.Rat_Numerator_Into(&result, &value)
		converted, status = big.Int_Int_64(&result)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, big.Int_64(-3), converted)
		big.Rat_Denominator_Into((*big.Nonempty_Int)(&result), &value)
		converted, status = big.Int_Int_64(&result)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, big.Int_64(2), converted)
	}
}

func test_modular_multiply_maximum_product(
	t *testing.T, workspace *big.Int_Modular_Workspace,
) {
	left := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM),
		Count: big.WORD_COUNT_MAXIMUM - 1}
	left.Words[big.WORD_COUNT_MAXIMUM-2] = big.Word(1) << big.WORD_BIT_INDEX_MAXIMUM
	right := big.Int{Words: big.Words{2}, Count: 1}
	modulus := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM),
		Count: big.WORD_COUNT_MAXIMUM}
	for index := range modulus.Words {
		modulus.Words[index] = big.Word(bits.WORD_64_MAXIMUM)
	}
	expected := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM), Count: modulus.Count}
	expected.Words[big.WORD_COUNT_MAXIMUM-1] = 1
	result := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Modular_Multiply(&result, &left, &right, &modulus, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&result, &expected))
	left.Count = big.WORD_COUNT_MAXIMUM
	left.Words[big.WORD_COUNT_MAXIMUM-1] = 1
	big.Int_Set_Int_64(&right, 0)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Modular_Multiply(&result, &left, &right, &modulus, workspace))
	test_bitwise_result(t, &result, 0)
}

func test_modular_multiply_mersenne_zero(
	t *testing.T, workspace *big.Int_Modular_Workspace,
) {
	test_modular_multiply_maximum_product(t, workspace)
	zero := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	two := big.Int{Words: big.Words{2}, Count: 1}
	modulus := big.Int{Words: big.Words{0xffffffffffffffff, 0x7fffffffffffffff}, Count: 2}
	result := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Modular_Multiply(&result, &zero, &two, &modulus, workspace))
	test_bitwise_result(t, &result, 0)
}

func test_modular_multiply_storage(
	t *testing.T, workspace *big.Int_Modular_Workspace,
	left_value, right_value, modulus_value, expected int64,
) {
	for _, size := range []int{1, 2, big.WORD_COUNT_MAXIMUM} {
		left := big.Int{Words: make(big.Words, size)}
		right := big.Int{Words: make(big.Words, size)}
		modulus := big.Int{Words: make(big.Words, size)}
		result := big.Int{Words: make(big.Words, size)}
		big.Int_Set_Int_64(&left, big.Int_64(left_value))
		big.Int_Set_Int_64(&right, big.Int_64(right_value))
		big.Int_Set_Int_64(&modulus, big.Int_64(modulus_value))
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Modular_Multiply(
			&result, &left, &right, &modulus, workspace,
		))
		test_bitwise_result(t, &result, expected)
	}
}

func test_modular_inverse_storage(t *testing.T, workspace *big.Int_Modular_Workspace) {
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		modulus := big.Int{Words: make(big.Words, size)}
		value := big.Int{Words: make(big.Words, size)}
		result := big.Int{Words: make(big.Words, size)}
		testify.Equal_Values(t, big.STATUS_DIVISOR_ZERO,
			big.Int_Modular_Inverse(&result, &value, &modulus, workspace))
		testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&result))
		if size == 0 {
			modulus.Words = make(big.Words, 1)
		}
		big.Int_Set_Int_64(&modulus, 1)
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Modular_Inverse(&result, &value, &modulus, workspace))
		testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&result))
		if size != 0 {
			big.Int_Set_Int_64(&value, 3)
			big.Int_Set_Int_64(&modulus, 11)
			testify.Equal_Values(t, big.STATUS_OK,
				big.Int_Modular_Inverse(&result, &value, &modulus, workspace))
			converted, status := big.Int_Int_64(&result)
			testify.Equal_Values(t, big.STATUS_OK, status)
			testify.Equal(t, big.Int_64(4), converted)
		}
		modulus = big.Int{Words: big.Words{1, 0, 1}, Count: 3}
		expected := big.Modular_Status(big.STATUS_RESULT_ABSENT)
		if size != 0 {
			big.Int_Set_Int_64(&value, 1)
			expected = big.STATUS_OK
		}
		testify.Equal(t, expected,
			big.Int_Modular_Inverse(&result, &value, &modulus, workspace))
		testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&result, &value))
	}
}

func test_modular_exponent_word_bounds(t *testing.T, workspace *big.Int_Modular_Workspace) {
	base := big.Int{Words: big.Words{2}, Count: 1}
	exponent := big.Int{Words: big.Words{10}, Count: 1}
	result := big.Int{Words: make(big.Words, 1)}
	for _, word := range []big.Word{2, big.Word(bits.WORD_64_MAXIMUM)} {
		modulus := big.Int{Words: big.Words{word}, Count: 1}
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Modular_Exponent(&result, &base, &exponent, &modulus, workspace))
		test_bitwise_result(t, &result, int64(1024%uint64(word)))
	}
}

func test_modular_exponent_storage(t *testing.T, workspace *big.Int_Modular_Workspace) {
	test_modular_exponent_word_bounds(t, workspace)
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		modulus := big.Int{Words: make(big.Words, size)}
		base := big.Int{Words: make(big.Words, size)}
		exponent := big.Int{Words: make(big.Words, size)}
		result := big.Int{Words: make(big.Words, size)}
		testify.Equal_Values(t, big.STATUS_DIVISOR_ZERO,
			big.Int_Modular_Exponent(&result, &base, &exponent, &modulus, workspace))
		testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&result))
		if size == 0 {
			modulus.Words = make(big.Words, 1)
		}
		big.Int_Set_Int_64(&modulus, 1)
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Modular_Exponent(&result, &base, &exponent, &modulus, workspace))
		testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&result))
		if size != 0 {
			big.Int_Set_Int_64(&base, 2)
			big.Int_Set_Int_64(&exponent, 10)
			big.Int_Set_Int_64(&modulus, 1000)
			testify.Equal_Values(t, big.STATUS_OK,
				big.Int_Modular_Exponent(
					&result, &base, &exponent, &modulus, workspace))
			converted, status := big.Int_Int_64(&result)
			testify.Equal_Values(t, big.STATUS_OK, status)
			testify.Equal(t, big.Int_64(24), converted)
		}
		modulus = big.Int{Words: big.Words{0, 1}, Count: 2}
		expected := big.Int_64(1024)
		if size == 0 {
			exponent = big.Int{Words: big.Words{1}, Count: 1}
			expected = 0
		}
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Modular_Exponent(&result, &base, &exponent, &modulus, workspace))
		converted, status := big.Int_Int_64(&result)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, expected, converted)
	}
}

func test_jacobi_storage(t *testing.T, workspace *big.Int_Jacobi_Workspace) {
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		numerator := big.Int{Words: make(big.Words, size)}
		denominator := big.Int{Words: make(big.Words, size)}
		symbol, status := big.Int_Jacobi(&numerator, &denominator, workspace)
		testify.Equal_Values(t, big.STATUS_INPUT_INVALID, status)
		testify.Equal(t, big.Jacobi_Symbol(0), symbol)
		if size != 0 {
			big.Int_Set_Int_64(&numerator, 2)
			big.Int_Set_Int_64(&denominator, 5)
			symbol, status = big.Int_Jacobi(&numerator, &denominator, workspace)
			testify.Equal_Values(t, big.STATUS_OK, status)
			testify.Equal(t, big.Jacobi_Symbol(-1), symbol)
		}
	}
}

func test_int_machine_storage(t *testing.T) {
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		value := big.Int{Words: make(big.Words, size)}
		testify.True(t, bool(big.Int_Is_Int_64(&value)))
		testify.True(t, bool(big.Int_Is_Uint_64(&value)))
		if size != 0 {
			big.Int_Set_Int_64(&value, -1)
			testify.True(t, bool(big.Int_Is_Int_64(&value)))
			testify.False(t, bool(big.Int_Is_Uint_64(&value)))
		}
	}
}

func test_int_gob_storage(t *testing.T, source []byte, expected *big.Int) {
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		if size < int(expected.Count) {
			continue
		}
		decoded := big.Int{Words: make(big.Words, size)}
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Gob_Decode(&decoded, source))
		testify.Equal(t, big.ORDER_SAME, big.Int_Compare(expected, &decoded))
		encoded := make(big.Int_Encoding, big.INT_GOB_SIZE_MAXIMUM)
		count, status := big.Int_Gob_Encode_Into(encoded, &decoded)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, source, []byte(encoded[:count]))
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Gob_Decode(&decoded, nil))
		testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&decoded))
	}
}

func test_int_float_64_output_storage(
	t *testing.T, number int64, expected big.Int_Float_64_Value_Bits,
) {
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		if size == 0 {
			if number != 0 {
				continue
			}
		}
		value := big.Int{Words: make(big.Words, size)}
		big.Int_Set_Int_64(&value, big.Int_64(number))
		encoding, accuracy := big.Int_Float_64_Bits(&value)
		testify.Equal(t, expected, encoding)
		testify.Equal(t, big.ACCURACY_EXACT, accuracy)
		decoded := big.Int{Words: make(big.Words, size)}
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Set_Float_64_Bits(&decoded, big.Float_64_Bits(encoding)))
		testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&decoded, &value))
	}
}

func test_int_compare_storage(t *testing.T) {
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		left := big.Int{Words: make(big.Words, size)}
		right := big.Int{Words: make(big.Words, size)}
		testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&left, &right))
		testify.Equal(t, big.ORDER_SAME, big.Int_Compare_Absolute(&left, &right))
		if size != 0 {
			big.Int_Set_Int_64(&left, -7)
			big.Int_Set_Int_64(&right, 5)
			testify.Equal(t, big.ORDER_BEFORE, big.Int_Compare(&left, &right))
			testify.Equal(t, big.ORDER_AFTER, big.Int_Compare(&right, &left))
			testify.Equal(t, big.ORDER_AFTER, big.Int_Compare_Absolute(&left, &right))
			testify.Equal(t, big.ORDER_BEFORE, big.Int_Compare_Absolute(&right, &left))
		}
	}
}

func test_int_bit_storage(t *testing.T) {
	test_int_set_bit_storage(t)
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		value := big.Int{Words: make(big.Words, size)}
		for _, number := range []big.Int_64{0, 7, -1} {
			if size == 0 {
				if number != 0 {
					continue
				}
			}
			big.Int_Set_Int_64(&value, number)
			for _, index := range []big.Bit_Index{0, 1, 63, 64, big.BIT_INDEX_MAXIMUM} {
				expected := big.Bit_Value((int64(number) >> uint(index)) & 1)
				testify.Equal(t, expected, big.Int_Bit(&value, index))
			}
		}
	}
}

func test_int_set_bit_storage(t *testing.T) {
	workspace := big.Int_Bitwise_Workspace{
		Left:   make(big.Int_Bitwise_Left_Words, big.BITWISE_WORD_COUNT_MAXIMUM),
		Right:  make(big.Int_Bitwise_Right_Words, big.BITWISE_WORD_COUNT_MAXIMUM),
		Result: make(big.Int_Bitwise_Result_Words, big.BITWISE_WORD_COUNT_MAXIMUM),
	}
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		value := big.Int{Words: make(big.Words, size)}
		result := big.Int{Words: make(big.Words, size)}
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Set_Bit(&result, &value, 0, big.BIT_CLEAR, &workspace))
		test_bitwise_result(t, &result, 0)
		if size != 0 {
			big.Int_Set_Int_64(&value, 7)
			testify.Equal_Values(t, big.STATUS_OK,
				big.Int_Set_Bit(&result, &value, 1, big.BIT_CLEAR, &workspace))
			test_bitwise_result(t, &result, 5)
			testify.Equal_Values(t, big.STATUS_OK,
				big.Int_Set_Bit(&result, &value, big.BIT_INDEX_MAXIMUM,
					big.BIT_CLEAR, &workspace))
			test_bitwise_result(t, &result, 7)
		}
	}
}

func test_int_shift_storage(t *testing.T) {
	var zero, empty big.Int
	for _, shift := range []big.Shift_Count{0, 1, 64, big.Shift_Count(big.BIT_COUNT_MAXIMUM)} {
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Shift_Left(&empty, &zero, shift))
		test_bitwise_result(t, &empty, 0)
		big.Int_Shift_Right(&empty, &zero, shift)
		test_bitwise_result(t, &empty, 0)
	}
	source := big.Int{Words: make(big.Words, 2)}
	result := big.Int{Words: make(big.Words, 2)}
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Words(&source,
		big.Words_Unvalidated{big.Word(bits.WORD_64_MAXIMUM), 1}))
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Shift_Left(&result, &source, 1))
	testify.Equal(t, big.Word_Count(2), result.Count)
	testify.Equal(t, big.Word(bits.WORD_64_MAXIMUM-1), result.Words[0])
	testify.Equal(t, big.Word(3), result.Words[1])
}

func test_int_add_storage(t *testing.T) {
	test_int_subtract_double_word_storage(t)
	test_int_sum_operand_storage(t)
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		left := big.Int{Words: make(big.Words, size)}
		right := big.Int{Words: make(big.Words, size)}
		result := big.Int{Words: make(big.Words, size)}
		big.Int_Set_Int_64(&left, 0)
		big.Int_Set_Int_64(&right, 0)
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Add(&result, &left, &right))
		testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&result))
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Subtract(&result, &left, &right))
		test_bitwise_result(t, &result, 0)
		if size == 0 {
			continue
		}
		big.Int_Set_Int_64(&left, 5)
		big.Int_Set_Int_64(&right, 7)
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Add(&result, &left, &right))
		converted, status := big.Int_Int_64(&result)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, big.Int_64(12), converted)
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Subtract(&result, &left, &right))
		test_bitwise_result(t, &result, -2)
		if size >= 2 {
			testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Words(&left,
				big.Words_Unvalidated{big.Word(bits.WORD_64_MAXIMUM), 1}))
			testify.Equal_Values(t, big.STATUS_OK,
				big.Int_Set_Words(&right, big.Words_Unvalidated{1, 1}))
			testify.Equal_Values(t, big.STATUS_OK, big.Int_Add(&result, &left, &right))
			testify.Equal(t, big.Word_Count(2), result.Count)
			testify.Equal(t, big.Word(0), result.Words[0])
			testify.Equal(t, big.Word(3), result.Words[1])
		}
	}
}

func test_int_subtract_double_word_storage(t *testing.T) {
	left := big.Int{Words: big.Words{0, 1}, Count: 2}
	right := big.Int{Words: big.Words{0, 1}, Count: 2}
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		result := big.Int{Words: make(big.Words, size)}
		left.Words[0] = 0
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Subtract(&result, &left, &right))
		test_bitwise_result(t, &result, 0)
		if size != 0 {
			left.Words[0] = 1
			testify.Equal_Values(t, big.STATUS_OK,
				big.Int_Subtract(&result, &left, &right))
			test_bitwise_result(t, &result, 1)
			testify.Equal_Values(t, big.STATUS_OK,
				big.Int_Subtract(&result, &right, &left))
			test_bitwise_result(t, &result, -1)
		}
	}
}

func test_int_sum_operand_storage(t *testing.T) {
	left := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	result := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	big.Int_Set_Int_64(&left, -7)
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		right := big.Int{Words: make(big.Words, size)}
		for _, number := range []big.Int_64{0, 5} {
			if size == 0 {
				if number != 0 {
					continue
				}
			}
			big.Int_Set_Int_64(&right, number)
			testify.Equal_Values(t, big.STATUS_OK, big.Int_Add(&result, &left, &right))
			converted, status := big.Int_Int_64(&result)
			testify.Equal_Values(t, big.STATUS_OK, status)
			testify.Equal(t, number-7, converted)
		}
	}
}

func test_int_absolute_storage(t *testing.T) {
	source := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		result := big.Int{Words: make(big.Words, size)}
		big.Int_Set_Int_64(&source, 0)
		big.Int_Absolute(&result, &source)
		testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&result))
		if size != 0 {
			big.Int_Set_Int_64(&source, -42)
			big.Int_Absolute(&result, &source)
			converted, status := big.Int_Int_64(&result)
			testify.Equal_Values(t, big.STATUS_OK, status)
			testify.Equal(t, big.Int_64(42), converted)
			big.Int_Absolute(&result, &result)
			converted, status = big.Int_Int_64(&result)
			testify.Equal_Values(t, big.STATUS_OK, status)
			testify.Equal(t, big.Int_64(42), converted)
		}
	}
}

func test_float_square_root_digits(
	t *testing.T, workspace *big.Float_Square_Root_Workspace,
) {
	test_float_square_root_two_word_pair(t, workspace)
	root := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	square := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	multiplication := big.Int_Multiplication_Workspace{
		Product: make(big.Int_Product_Words, big.WORD_COUNT_MAXIMUM),
	}
	var source, expected, result big.Float
	for _, value := range []*big.Float{&source, &expected, &result} {
		test_float_initialize(value)
		value.Precision = 10 * big.WORD_BIT_COUNT
	}
	for _, word := range []big.Word{
		0x5555555555555555, 0xaaaaaaaaaaaaaaaa, 0xffffffffffffffff,
	} {
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Words(&root,
			big.Words_Unvalidated{word, word, word, word, word}))
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Multiply(&square, &root, &root, &multiplication))
		big.Float_Set_Int(&source, &square)
		big.Float_Set_Int(&expected, &root)
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Square_Root(&result, &source, workspace))
		testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&result, &expected))
		testify.Equal(t, big.ACCURACY_EXACT, result.Accuracy)
	}
}

func test_float_square_root_two_word_pair(
	t *testing.T, workspace *big.Float_Square_Root_Workspace,
) {
	// (2^64 + 3)^2 leaves binary pair 10 once two workspace words are active.
	source := big.Float{Precision: 129, Form: big.FLOAT_FORM_FINITE, Exponent: 129,
		Mantissa: big.Float_Mantissa{Words: big.Words{9, 6, 1}, Count: 3}}
	expected := big.Float{Precision: 65, Form: big.FLOAT_FORM_FINITE, Exponent: 65,
		Mantissa: big.Float_Mantissa{Words: big.Words{3, 1}, Count: 2}}
	result := big.Float{Precision: 128,
		Mantissa: big.Float_Mantissa{Words: make(big.Words, 2)}}
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Square_Root(&result, &source, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&result, &expected))
	testify.Equal(t, big.ACCURACY_EXACT, result.Accuracy)
}

func test_float_binary_storage(t *testing.T) {
	test_float_binary_maximum(t)
	wide := big.Float{Precision: 65, Form: big.FLOAT_FORM_FINITE, Exponent: 65,
		Mantissa: big.Float_Mantissa{Words: big.Words{1, 1}, Count: 2}}
	float_text_expect(t, &wide, 'b', -1, "18446744073709551617p+0")
	var expected big.Float
	test_float_initialize(&expected)
	big.Float_Set_Int_64(&expected, 3)
	expected.Exponent = 1
	for _, size := range []int{1, 2, big.WORD_COUNT_MAXIMUM} {
		words := make(big.Words, size)
		value := big.Float{Mantissa: big.Float_Mantissa{Words: words}}
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Set_Float_32_Bits(&value, FLOAT_32_THREE_HALVES_BITS))
		testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&value, &expected))
		testify.Equal(t, big.ACCURACY_EXACT, value.Accuracy)
		value = big.Float{Mantissa: big.Float_Mantissa{Words: words}}
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Set_Float_64_Bits(&value, FLOAT_64_THREE_HALVES_BITS))
		testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&value, &expected))
		testify.Equal(t, big.ACCURACY_EXACT, value.Accuracy)
		encoded_32, accuracy := big.Float_Float_32_Bits(&value)
		testify.Equal_Values(t, FLOAT_32_THREE_HALVES_BITS, encoded_32)
		testify.Equal(t, big.ACCURACY_EXACT, accuracy)
		encoded_64, accuracy := big.Float_Float_64_Bits(&value)
		testify.Equal_Values(t, FLOAT_64_THREE_HALVES_BITS, encoded_64)
		testify.Equal(t, big.ACCURACY_EXACT, accuracy)
		value.Precision = 64
		value.Mantissa.Words[0] = 0xffffffffffffffff
		encoded_32, accuracy = big.Float_Float_32_Bits(&value)
		testify.Equal_Values(t, 0x40000000, encoded_32)
		testify.Equal(t, big.ACCURACY_ABOVE, accuracy)
		encoded_64, accuracy = big.Float_Float_64_Bits(&value)
		testify.Equal_Values(t, 0x4000000000000000, encoded_64)
		testify.Equal(t, big.ACCURACY_ABOVE, accuracy)
	}
}

func test_float_binary_maximum(t *testing.T) {
	integers := int_domain_values(t)
	var value big.Float
	test_float_initialize(&value)
	big.Float_Set_Int(&value, &integers[POSITIVE_MAXIMUM_WORD_INDEX])
	var workspace big.Float_Text_Workspace
	test_float_text_workspace_initialize(&workspace)
	output := make(big.Text, big.FLOAT_TEXT_SIZE_MAXIMUM)
	count, status := big.Float_Text_Into(output, &value, 'f', 0, &workspace)
	testify.Equal_Values(t, big.STATUS_OK, status)
	expected := string(output[:count]) + "p+0"
	count, status = big.Float_Text_Into(output, &value, 'b', -1, &workspace)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, expected, string(output[:count]))
}

func test_float_division_bound_storage(
	t *testing.T, workspace *big.Float_Division_Workspace,
) {
	test_float_division_reciprocal(t, workspace)
	test_float_division_two_word_quotient(t, workspace)
	test_float_division_empty_overflow(t, workspace)
	test_float_division_word_scale_maximum(t, workspace)
	var low, high big.Float
	test_float_initialize(&low)
	test_float_initialize(&high)
	big.Float_Set_Int_64(&low, 1)
	big.Float_Set_Int_64(&high, 1)
	low.Exponent, high.Exponent = big.FLOAT_EXPONENT_MINIMUM, big.FLOAT_EXPONENT_MAXIMUM
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		result := big.Float{Mantissa: big.Float_Mantissa{Words: make(big.Words, size)}}
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Quotient(&result, &low, &high, workspace))
		testify.Equal(t, big.FLOAT_FORM_ZERO, result.Form)
		testify.Equal(t, big.ACCURACY_BELOW, result.Accuracy)
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Quotient(&result, &high, &low, workspace))
		testify.Equal(t, big.FLOAT_FORM_INFINITY, result.Form)
		testify.Equal(t, big.ACCURACY_ABOVE, result.Accuracy)
	}
}

func test_float_division_reciprocal(t *testing.T, workspace *big.Float_Division_Workspace) {
	var one, divisor, result big.Float
	test_float_initialize(&one)
	test_float_initialize(&divisor)
	test_float_initialize(&result)
	big.Float_Set_Int_64(&one, 1)
	divisor.Precision, divisor.Form, divisor.Exponent = 65, big.FLOAT_FORM_FINITE, 65
	divisor.Mantissa.Count = 2
	divisor.Mantissa.Words[0], divisor.Mantissa.Words[1] = 1, 1
	result.Precision = 128
	// B^3/(B+1) = B^2-B+1-1/(B+1), so nearest rounding raises the reciprocal.
	expected := big.Float{Precision: 128, Form: big.FLOAT_FORM_FINITE, Exponent: -64,
		Mantissa: big.Float_Mantissa{
			Words: big.Words{1, big.Word(bits.WORD_64_MAXIMUM)}, Count: 2}}
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Quotient(&result, &one, &divisor, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&result, &expected))
	testify.Equal(t, big.ACCURACY_ABOVE, result.Accuracy)
	testify.Equal_Values(t, big.STATUS_OK, big.Float_Set_Precision(&result, 127))
	expected.Precision = 127
	expected.Mantissa.Words[0] = 1 << 63
	expected.Mantissa.Words[1] = (1 << 63) - 1
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Quotient(&result, &one, &divisor, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&result, &expected))
	testify.Equal(t, big.ACCURACY_BELOW, result.Accuracy)
}

func test_float_division_two_word_quotient(t *testing.T, workspace *big.Float_Division_Workspace) {
	left := big.Float{Precision: 67, Form: big.FLOAT_FORM_FINITE, Exponent: 67,
		Mantissa: big.Float_Mantissa{
			Words: make(big.Words, big.WORD_COUNT_MAXIMUM), Count: 2}}
	right := big.Float{Precision: 67, Form: big.FLOAT_FORM_FINITE, Exponent: 67,
		Mantissa: big.Float_Mantissa{
			Words: make(big.Words, big.WORD_COUNT_MAXIMUM), Count: 2}}
	left.Mantissa.Words[0], left.Mantissa.Words[1] = 5, 5
	right.Mantissa.Words[0], right.Mantissa.Words[1] = 4, 4
	var result, expected big.Float
	test_float_initialize(&result)
	test_float_initialize(&expected)
	result.Precision = 1
	big.Float_Set_Int_64(&expected, 1)
	// Common factor B+1 leaves 5/4, which rounds down at one-bit precision.
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Quotient(&result, &left, &right, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&result, &expected))
	testify.Equal(t, big.ACCURACY_BELOW, result.Accuracy)
}

func test_float_division_empty_overflow(t *testing.T, workspace *big.Float_Division_Workspace) {
	left := big.Float{Precision: 131, Form: big.FLOAT_FORM_FINITE,
		Exponent: big.FLOAT_EXPONENT_MAXIMUM,
		Mantissa: big.Float_Mantissa{
			Words: make(big.Words, big.WORD_COUNT_MAXIMUM), Count: 3}}
	right := big.Float{Precision: 131, Form: big.FLOAT_FORM_FINITE, Exponent: 1,
		Mantissa: big.Float_Mantissa{
			Words: make(big.Words, big.WORD_COUNT_MAXIMUM), Count: 3}}
	left.Mantissa.Words[0], left.Mantissa.Words[2] = 7, 7
	right.Mantissa.Words[0], right.Mantissa.Words[2] = 4, 4
	// Common factor B^2+1 leaves 7/4; one-bit rounding overflows the final exponent.
	result := big.Float{Precision: 1}
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Quotient(&result, &left, &right, workspace))
	testify.Equal(t, big.FLOAT_FORM_INFINITY, result.Form)
	testify.Equal(t, big.ACCURACY_ABOVE, result.Accuracy)
	testify.Equal(t, big.POLARITY_NONNEGATIVE, result.Negative)
}

func test_float_division_word_scale_maximum(
	t *testing.T, workspace *big.Float_Division_Workspace,
) {
	var one, three, result big.Float
	test_float_initialize(&one)
	test_float_initialize(&three)
	test_float_initialize(&result)
	big.Float_Set_Int_64(&one, 1)
	big.Float_Set_Int_64(&three, 3)
	result.Precision = big.FLOAT_PRECISION_MAXIMUM - big.WORD_BIT_COUNT - 1
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Quotient(&result, &one, &three, workspace))
	// Odd precision truncates repeating 10 with a one; the omitted tail is below half.
	testify.Equal_Values(t, big.WORD_COUNT_MAXIMUM-1, result.Mantissa.Count)
	for _, word := range result.Mantissa.Words[:result.Mantissa.Count] {
		testify.Equal_Values(t, 0x5555555555555555, word)
	}
	testify.Equal_Values(t, -1, result.Exponent)
	testify.Equal(t, big.FLOAT_FORM_FINITE, result.Form)
	testify.Equal(t, big.ACCURACY_BELOW, result.Accuracy)
	one.Mantissa.Words[0], three.Mantissa.Words[0] = 1, 3
	one.Precision, three.Precision = 1, 2
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Set_Precision(&result, big.FLOAT_PRECISION_MAXIMUM-3))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Quotient(&result, &one, &three, workspace))
	testify.Equal_Values(t, big.WORD_COUNT_MAXIMUM, result.Mantissa.Count)
	for _, word := range result.Mantissa.Words[:big.WORD_COUNT_MAXIMUM-1] {
		testify.Equal_Values(t, 0x5555555555555555, word)
	}
	testify.Equal_Values(t, 0x1555555555555555,
		result.Mantissa.Words[big.WORD_COUNT_MAXIMUM-1])
	testify.Equal_Values(t, -1, result.Exponent)
	testify.Equal(t, big.ACCURACY_BELOW, result.Accuracy)
	one.Precision, one.Exponent, one.Mantissa.Words[0] = 2, 2, 3
	three.Mantissa.Words[0] = 2
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Set_Precision(&result, big.FLOAT_PRECISION_MAXIMUM-2))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Quotient(&result, &one, &three, workspace))
	testify.Equal_Values(t, big.WORD_COUNT_MAXIMUM, result.Mantissa.Count)
	for _, word := range result.Mantissa.Words[:big.WORD_COUNT_MAXIMUM-1] {
		testify.Equal_Values(t, 0, word)
	}
	testify.Equal_Values(t, 0x3000000000000000,
		result.Mantissa.Words[big.WORD_COUNT_MAXIMUM-1])
	testify.Equal_Values(t, 1, result.Exponent)
	testify.Equal(t, big.ACCURACY_EXACT, result.Accuracy)
}

func test_float_parse_storage(t *testing.T) {
	var workspace big.Float_Parse_Workspace
	test_float_parse_workspace_initialize(&workspace)
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		value := big.Float{Mantissa: big.Float_Mantissa{Words: make(big.Words, size)}}
		for _, source := range []string{"-0", "+Inf", "-Inf"} {
			_, status := big.Float_Parse(
				&value, big.Float_Parse_Text_Unvalidated(source), 0, &workspace)
			testify.Equal_Values(t, big.STATUS_OK, status)
			float_text_expect(t, &value, 'g', -1, source)
			testify.Equal_Values(t, big.STATUS_OK, big.Float_Set_Precision(&value, 2))
			testify.Equal(t, big.Float_Precision(2), value.Precision)
			testify.Equal(t, big.ACCURACY_EXACT, value.Accuracy)
		}
		if size != 0 {
			_, status := big.Float_Parse(&value, []byte("1.5"), 0, &workspace)
			testify.Equal_Values(t, big.STATUS_OK, status)
			float_text_expect(t, &value, 'g', -1, "1.5")
		}
	}
}

func test_float_integer_storage(t *testing.T) {
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		value := big.Float{Mantissa: big.Float_Mantissa{Words: make(big.Words, size)}}
		integer := big.Int{Words: make(big.Words, size)}
		for _, magnitude := range []big.Word{0, 7} {
			if magnitude != 0 {
				if size == 0 {
					continue
				}
				value.Precision, value.Form, value.Exponent =
					64, big.FLOAT_FORM_FINITE, 3
				value.Mantissa.Count, value.Mantissa.Words[0] = 1, magnitude
			}
			if size != 0 {
				big.Int_Set_Int_64(&integer, -1)
			}
			accuracy, status := big.Float_Int_Into(&integer, &value)
			testify.Equal_Values(t, big.STATUS_OK, status)
			testify.Equal(t, big.ACCURACY_EXACT, accuracy)
			converted, conversion := big.Int_Uint_64(&integer)
			testify.Equal_Values(t, big.STATUS_OK, conversion)
			testify.Equal(t, big.Word_64(magnitude), converted)
			signed, signed_accuracy, signed_status := big.Float_Int_64(&value)
			testify.Equal_Values(t, big.STATUS_OK, signed_status)
			testify.Equal(t, big.ACCURACY_EXACT, signed_accuracy)
			testify.Equal(t, big.Int_64(magnitude), signed)
			unsigned, unsigned_accuracy, unsigned_status := big.Float_Uint_64(&value)
			testify.Equal_Values(t, big.STATUS_OK, unsigned_status)
			testify.Equal(t, big.ACCURACY_EXACT, unsigned_accuracy)
			testify.Equal(t, big.Word_64(magnitude), unsigned)
			decoded := big.Float{
				Mantissa: big.Float_Mantissa{Words: make(big.Words, size)},
			}
			big.Float_Set_Int(&decoded, &integer)
			testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&decoded, &value))
			testify.Equal(t, big.ACCURACY_EXACT, decoded.Accuracy)
		}
	}
}

func test_float_add_storage(t *testing.T, workspace *big.Float_Addition_Workspace) {
	test_float_add_double_word_storage(t, workspace)
	var left, right, expected big.Float
	for _, value := range []*big.Float{&left, &right, &expected} {
		test_float_initialize(value)
		big.Float_Set_Int_64(value, 1)
	}
	left.Mantissa.Words[0], right.Mantissa.Words[0] = 1, 1
	left.Exponent, right.Exponent = 0, -2
	for _, count := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		result := big.Float{
			Precision: 64,
			Mantissa:  big.Float_Mantissa{Words: make(big.Words, count)},
		}
		zero := big.Float{Mantissa: big.Float_Mantissa{Words: make(big.Words, count)}}
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Subtract(&result, &zero, &zero, workspace))
		testify.Equal(t, big.FLOAT_FORM_ZERO, result.Form)
		testify.Equal(t, big.ACCURACY_EXACT, result.Accuracy)
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Subtract(&result, &left, &left, workspace))
		testify.Equal(t, big.FLOAT_FORM_ZERO, result.Form)
		testify.Equal(t, big.ACCURACY_EXACT, result.Accuracy)
		if count == 0 {
			continue
		}
		left.Mantissa.Words = make(big.Words, count)
		right.Mantissa.Words = make(big.Words, count)
		left.Mantissa.Words[0], right.Mantissa.Words[0] = 1, 1
		big.Float_Set_Int_64(&expected, 5)
		expected.Exponent = 0
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Add(&result, &left, &right, workspace))
		testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&result, &expected))
		testify.Equal(t, big.ACCURACY_EXACT, result.Accuracy)
		big.Float_Set_Int_64(&expected, 3)
		expected.Exponent = -1
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Subtract(&result, &left, &right, workspace))
		testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&result, &expected))
		testify.Equal(t, big.ACCURACY_EXACT, result.Accuracy)
	}
}

func test_float_add_empty_storage(t *testing.T, workspace *big.Float_Addition_Workspace) {
	value := big.Float{Precision: 128, Form: big.FLOAT_FORM_FINITE,
		Exponent: big.FLOAT_EXPONENT_MAXIMUM,
		Mantissa: big.Float_Mantissa{Words: big.Words{0, 1}, Count: 2}}
	result := big.Float{Precision: 128}
	testify.Equal_Values(t, big.STATUS_OK, big.Float_Add(&result, &value, &value, workspace))
	testify.Equal(t, big.FLOAT_FORM_INFINITY, result.Form)
	testify.Equal(t, big.ACCURACY_ABOVE, result.Accuracy)
	value.Exponent = 65
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Subtract(&result, &value, &value, workspace))
	testify.Equal(t, big.FLOAT_FORM_ZERO, result.Form)
	testify.Equal(t, big.ACCURACY_EXACT, result.Accuracy)
}

func test_float_add_double_word_storage(t *testing.T, workspace *big.Float_Addition_Workspace) {
	test_float_add_empty_storage(t, workspace)
	var left, right, expected big.Float
	test_float_initialize(&left)
	test_float_initialize(&right)
	test_float_initialize(&expected)
	integer := big.Int{Words: big.Words{0, 1}, Count: 2}
	left.Precision, right.Precision = 128, 128
	big.Float_Set_Int(&right, &integer)
	integer.Words[1] = 2
	big.Float_Set_Int(&left, &integer)
	result := big.Float{Precision: 64,
		Mantissa: big.Float_Mantissa{Words: make(big.Words, 1)}}
	big.Float_Set_Int_64(&expected, 3)
	expected.Exponent = 66
	testify.Equal_Values(t, big.STATUS_OK, big.Float_Add(&result, &left, &right, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&result, &expected))
	testify.Equal(t, big.ACCURACY_EXACT, result.Accuracy)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Subtract(&result, &left, &right, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&result, &right))
	testify.Equal(t, big.ACCURACY_EXACT, result.Accuracy)
}

func test_square_root_double_word_digits(t *testing.T) {
	value := big.Int{Words: big.Words{2, 2}, Count: 2}
	result := big.Int{Words: make(big.Words, 1)}
	var workspace big.Int_Square_Root_Workspace
	test_square_root_workspace_initialize(&workspace)
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Square_Root(&result, &value, &workspace))
	test_bitwise_result(t, &result, 6074000999)
}

func test_float_square_root_operand_storage(
	t *testing.T, workspace *big.Float_Square_Root_Workspace,
) {
	var result, expected big.Float
	test_float_initialize(&result)
	test_float_initialize(&expected)
	result.Precision = 1
	big.Float_Set_Int_64(&expected, 1)
	for _, size := range []int{1, 2} {
		source := big.Float{Precision: 64,
			Mantissa: big.Float_Mantissa{Words: make(big.Words, size)}}
		big.Float_Set_Int_64(&source, 2)
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Square_Root(&result, &source, workspace))
		testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&result, &expected))
		testify.Equal(t, big.ACCURACY_BELOW, result.Accuracy)
	}
}

func test_float_square_root_storage(
	t *testing.T, workspace *big.Float_Square_Root_Workspace,
) {
	test_float_square_root_digits(t, workspace)
	test_float_square_root_operand_storage(t, workspace)
	var source, expected big.Float
	test_float_initialize(&source)
	test_float_initialize(&expected)
	big.Float_Set_Int_64(&source, 1)
	big.Float_Set_Int_64(&expected, 1)
	source.Exponent, expected.Exponent = -1, 0
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		result := big.Float{Mantissa: big.Float_Mantissa{Words: make(big.Words, size)}}
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Square_Root(&result, &source, workspace))
		testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&result, &expected))
		testify.Equal(t, big.ACCURACY_EXACT, result.Accuracy)
	}
}

func test_float_quotient_storage(
	t *testing.T, dividend, divisor *big.Float, workspace *big.Float_Division_Workspace,
) {
	test_float_quotient_operand_storage(t, workspace)
	test_float_quotient_negative_accuracy(t, workspace)
	var expected big.Float
	test_float_initialize(&expected)
	big.Float_Set_Int_64(&expected, 3)
	for _, count := range []int{1, 2, big.WORD_COUNT_MAXIMUM} {
		result := big.Float{
			Precision: 64,
			Mantissa:  big.Float_Mantissa{Words: make(big.Words, count)},
		}
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Quotient(&result, dividend, divisor, workspace))
		testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&result, &expected))
		testify.Equal(t, big.ACCURACY_EXACT, result.Accuracy)
	}
}

func test_float_quotient_negative_accuracy(t *testing.T, workspace *big.Float_Division_Workspace) {
	var left, right, result, expected big.Float
	test_float_initialize(&left)
	test_float_initialize(&right)
	test_float_initialize(&result)
	test_float_initialize(&expected)
	big.Float_Set_Int_64(&left, -1)
	right.Precision = 192
	integer := big.Int{Words: big.Words{1, 0, 1}, Count: 3}
	big.Float_Set_Int(&right, &integer)
	result.Precision = 1
	big.Float_Set_Int_64(&expected, -1)
	expected.Exponent = -127
	// -1/(2^128+1) lies just above -2^-128, its nearest one-bit value.
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Quotient(&result, &left, &right, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&result, &expected))
	testify.Equal(t, big.ACCURACY_BELOW, result.Accuracy)
}

func test_float_quotient_operand_storage(t *testing.T, workspace *big.Float_Division_Workspace) {
	for _, size := range []int{1, 2, big.WORD_COUNT_MAXIMUM} {
		left := big.Float{
			Precision: 64, Form: big.FLOAT_FORM_FINITE, Exponent: 2,
			Mantissa: big.Float_Mantissa{Words: make(big.Words, size), Count: 1},
		}
		right := big.Float{
			Precision: 64, Form: big.FLOAT_FORM_FINITE, Exponent: 1,
			Mantissa: big.Float_Mantissa{Words: make(big.Words, size), Count: 1},
		}
		left.Mantissa.Words[0], right.Mantissa.Words[0] = 3, 1
		result := big.Float{
			Precision: 64, Mantissa: big.Float_Mantissa{Words: make(big.Words, size)},
		}
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Quotient(&result, &left, &right, workspace))
		testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&result, &left))
		testify.Equal(t, big.ACCURACY_EXACT, result.Accuracy)
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Quotient(&result, &right, &left, workspace))
		testify.Equal(t, big.FLOAT_FORM_FINITE, result.Form)
		testify.Equal(t, big.Float_Exponent(-1), result.Exponent)
		testify.Equal(t, big.Word_Count(1), result.Mantissa.Count)
		testify.Equal(t, big.Word(0xaaaaaaaaaaaaaaab), result.Mantissa.Words[0])
		testify.Equal(t, big.ACCURACY_ABOVE, result.Accuracy)
	}
}

func test_float_multiply_aligned_negative(
	t *testing.T, value, expected *big.Float, workspace *big.Float_Multiplication_Workspace,
) {
	var negative, result big.Float
	test_float_initialize(&negative)
	test_float_initialize(&result)
	big.Float_Negate(&negative, value)
	result.Precision = 64
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Multiply(&result, value, &negative, workspace))
	testify.Equal(t, big.POLARITY_NEGATIVE, result.Negative)
	testify.Equal(t, big.ACCURACY_EXACT, result.Accuracy)
	big.Float_Negate(&result, &result)
	testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&result, expected))
}

func test_float_multiply_aligned_storage(t *testing.T) {
	var value, expected big.Float
	test_float_initialize(&value)
	test_float_initialize(&expected)
	value.Precision = 128
	value.Form = big.FLOAT_FORM_FINITE
	value.Mantissa.Count = 2
	value.Mantissa.Words[1] = 1
	value.Exponent = 65
	big.Float_Set_Int_64(&expected, 1)
	expected.Exponent = 129
	var workspace big.Float_Multiplication_Workspace
	workspace.Result = make(big.Float_Arithmetic_Words, big.FLOAT_ADDITION_WORD_COUNT_MAXIMUM)
	test_float_multiply_aligned_negative(t, &value, &expected, &workspace)
	short_result := big.Float{Precision: 64,
		Mantissa: big.Float_Mantissa{Words: make(big.Words, 1)}}
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Multiply(&short_result, &value, &value, &workspace))
	testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&short_result, &expected))
	testify.Equal(t, big.ACCURACY_EXACT, short_result.Accuracy)
	for _, count := range []int{1, 2, big.WORD_COUNT_MAXIMUM} {
		value.Mantissa.Words = make(big.Words, count)
		value.Exponent = 65
		value.Mantissa.Count = 2
		if count == 1 {
			value.Mantissa.Count = 1
		}
		value.Mantissa.Words[value.Mantissa.Count-1] = 1
		result := big.Float{
			Precision: 64,
			Mantissa:  big.Float_Mantissa{Words: make(big.Words, count)},
		}
		zero := big.Float{Mantissa: big.Float_Mantissa{Words: make(big.Words, count)}}
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Multiply(&result, &zero, &zero, &workspace))
		testify.Equal(t, big.FLOAT_FORM_ZERO, result.Form)
		testify.Equal(t, big.ACCURACY_EXACT, result.Accuracy)
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Multiply(&result, &value, &value, &workspace))
		testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&result, &expected))
		testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&expected, &result))
		testify.Equal(t, big.ACCURACY_EXACT, big.Float_Accuracy(&result))
		result = big.Float{
			Precision: 63, Mantissa: big.Float_Mantissa{Words: result.Mantissa.Words},
		}
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Multiply(&result, &value, &value, &workspace))
		testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&result, &expected))
		testify.Equal(t, big.ACCURACY_EXACT, result.Accuracy)
		big.Float_Set_Infinity(&zero, true)
		value.Mantissa.Count, value.Mantissa.Words[0] = 1, 0xffffffffffffffff
		value.Exponent = 1
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Multiply(&result, &value, &value, &workspace))
		testify.Equal(t, big.Float_Exponent(2), result.Exponent)
		testify.Equal(t, big.Word_Count(1), result.Mantissa.Count)
		testify.Equal(t, big.Word(0x7fffffffffffffff), result.Mantissa.Words[0])
		testify.Equal(t, big.ACCURACY_BELOW, result.Accuracy)
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Multiply(&result, &zero, &value, &workspace))
		testify.Equal(t, big.FLOAT_FORM_INFINITY, result.Form)
		testify.Equal(t, big.POLARITY_NEGATIVE, result.Negative)
		testify.Equal(t, big.ACCURACY_EXACT, result.Accuracy)
	}
}

func test_float_accuracy_storage(t *testing.T) {
	for _, count := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		value := big.Float{Mantissa: big.Float_Mantissa{Words: make(big.Words, count)}}
		testify.Equal(t, value.Precision, big.Float_Precision_Of(&value))
		for _, mode := range []big.Rounding_Mode_Unvalidated{
			big.ROUND_TO_NEAREST_EVEN, big.ROUND_TO_POSITIVE_INFINITY,
		} {
			testify.Equal_Values(t, big.STATUS_OK,
				big.Float_Set_Rounding_Mode(&value, mode))
			testify.Equal(t, big.Rounding_Mode(mode), big.Float_Rounding_Mode(&value))
		}
		for _, accuracy := range []big.Accuracy{
			big.ACCURACY_BELOW, big.ACCURACY_EXACT, big.ACCURACY_ABOVE,
		} {
			value.Accuracy = accuracy
			testify.Equal(t, accuracy, big.Float_Accuracy(&value))
		}
	}
}

func test_float_text_decimal_carry(t *testing.T) {
	var fraction big.Float
	test_float_initialize(&fraction)
	big.Float_Set_Int_64(&fraction, 7)
	fraction.Exponent = 1
	float_text_expect(t, &fraction, 'g', 2, "1.8")
	var value big.Float
	test_float_initialize(&value)
	for _, test := range []struct {
		Mantissa  big.Int_64
		Exponent  big.Float_Exponent
		Precision int
		Expected  string
	}{
		{3, 0, 0, "1"},
		{13, 0, 4, "0.8125"},
		{5, -3, 2, "0.08"},
		{199, 7, 0, "100"},
	} {
		big.Float_Set_Int_64(&value, test.Mantissa)
		value.Exponent = test.Exponent
		float_text_expect(t, &value, 'f', test.Precision, test.Expected)
	}
	for _, mantissa := range []big.Int_64{1<<59 + 1, 1<<56 + 1} {
		big.Float_Set_Int_64(&value, mantissa)
		value.Exponent = -61
		float_text_expect(t, &value, 'e', 0, "2e-19")
	}
}

func test_float_text_empty_significand(t *testing.T) {
	var workspace big.Float_Text_Workspace
	test_float_text_workspace_initialize(&workspace)
	workspace.Integers.Significand = big.Int{}
	var value big.Float
	test_float_initialize(&value)
	output := make(big.Text, big.FLOAT_TEXT_SIZE_MAXIMUM)
	for _, format := range []big.Float_Text_Format_Unvalidated{
		'b', 'p', 'x', 'e', 'E', 'f', 'g', 'G',
	} {
		big.Float_Set_Int_64(&value, 0)
		count, status := big.Float_Text_Into(output, &value, format, 0, &workspace)
		testify.Equal_Values(t, big.STATUS_OK, status)
		expected := "0"
		if format == 'e' {
			expected = "0e+00"
		}
		if format == 'E' {
			expected = "0E+00"
		}
		if format == 'x' {
			expected = "0x0p+00"
		}
		testify.Equal(t, expected, string(output[:count]))
		big.Float_Set_Infinity(&value, false)
		count, status = big.Float_Text_Into(output, &value, format, 0, &workspace)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, "+Inf", string(output[:count]))
	}
}

func test_division_trial_words(t *testing.T, workspace *big.Int_Division_Workspace) {
	test_division_storage(t, workspace)
	test_division_word_quotient_maximum(t, workspace)
	maximum := big.Word(bits.WORD_64_MAXIMUM)
	dividend := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	divisor := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	quotient := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	remainder := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	reconstructed := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	multiplication := big.Int_Multiplication_Workspace{
		Product: make(big.Int_Product_Words, big.WORD_COUNT_MAXIMUM),
	}
	for _, divisor_low := range []big.Word{1, 2, maximum} {
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Set_Words(&divisor, big.Words_Unvalidated{divisor_low, maximum}))
		for _, high := range []big.Word{1, 2, maximum} {
			for _, next := range []big.Word{0, 1, 2, maximum} {
				for _, low := range []big.Word{0, 1, 2, maximum} {
					testify.Equal_Values(t, big.STATUS_OK,
						big.Int_Set_Words(&dividend,
							big.Words_Unvalidated{low, next, high}))
					testify.Equal_Values(t, big.STATUS_OK,
						big.Int_Quotient_Remainder(&quotient, &remainder,
							&dividend, &divisor, workspace))
					testify.Equal(t, big.POLARITY_NONNEGATIVE,
						remainder.Negative)
					testify.Equal(t, big.ORDER_BEFORE,
						big.Int_Compare(&remainder, &divisor))
					testify.Equal_Values(t, big.STATUS_OK, big.Int_Multiply(
						&reconstructed, &quotient, &divisor,
						&multiplication))
					testify.Equal_Values(t, big.STATUS_OK,
						big.Int_Add(&reconstructed, &reconstructed,
							&remainder))
					testify.Equal(t, big.ORDER_SAME,
						big.Int_Compare(&dividend, &reconstructed))
				}
			}
		}
	}
}

func test_division_word_quotient_maximum(t *testing.T, workspace *big.Int_Division_Workspace) {
	maximum := big.Word(bits.WORD_64_MAXIMUM)
	// ((B-2)*B^2+B) / ((B-1)*B) is exactly B-1 for B=2^64.
	dividend := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM), Count: 3}
	dividend.Words[1], dividend.Words[2] = 1, maximum-1
	divisor := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM), Count: 2}
	divisor.Words[1] = maximum
	quotient := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	remainder := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Quotient_Remainder(&quotient, &remainder, &dividend, &divisor, workspace))
	testify.Equal_Values(t, 1, quotient.Count)
	testify.Equal(t, maximum, quotient.Words[0])
	testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&remainder))
}

func test_division_storage(t *testing.T, workspace *big.Int_Division_Workspace) {
	test_quotient_storage(t, workspace)
	test_divisor_storage(t, workspace)
	test_division_result_storage(t, workspace)
	divisor := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	quotient := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	remainder := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Set_Words(&divisor, big.Words_Unvalidated{0, 0, 1}))
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		dividend := big.Int{Words: make(big.Words, size)}
		if size != 0 {
			big.Int_Set_Int_64(&dividend, 5)
		}
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Quotient_Remainder(
				&quotient, &remainder, &dividend, &divisor, workspace))
		testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&quotient))
		testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&remainder, &dividend))
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Divide_Modulus(
				&quotient, &remainder, &dividend, &divisor, workspace))
		testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&quotient))
		testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&remainder, &dividend))
	}
}

func test_quotient_storage(t *testing.T, workspace *big.Int_Division_Workspace) {
	test_quotient_wide_storage(t, workspace)
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		dividend := big.Int{Words: make(big.Words, size)}
		divisor := big.Int{Words: make(big.Words, max(1, size))}
		result := big.Int{Words: make(big.Words, size)}
		big.Int_Set_Int_64(&divisor, 3)
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Quotient(&result, &dividend, &divisor, workspace))
		test_bitwise_result(t, &result, 0)
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Remainder(&result, &dividend, &divisor, workspace))
		test_bitwise_result(t, &result, 0)
		if size != 0 {
			big.Int_Set_Int_64(&dividend, -7)
			testify.Equal_Values(t, big.STATUS_OK,
				big.Int_Quotient(&result, &dividend, &divisor, workspace))
			test_bitwise_result(t, &result, -2)
			testify.Equal_Values(t, big.STATUS_OK,
				big.Int_Remainder(&result, &dividend, &divisor, workspace))
			test_bitwise_result(t, &result, -1)
			remainder := big.Int{Words: make(big.Words, size)}
			testify.Equal_Values(t, big.STATUS_OK, big.Int_Quotient_Remainder(
				&result, &remainder, &dividend, &divisor, workspace))
			test_bitwise_result(t, &result, -2)
			test_bitwise_result(t, &remainder, -1)
		}
	}
}

func test_quotient_wide_storage(t *testing.T, workspace *big.Int_Division_Workspace) {
	divisor := big.Int{Words: big.Words{0, 1}, Count: 2}
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		dividend := big.Int{Words: make(big.Words, size)}
		result := big.Int{Words: make(big.Words, size)}
		if size != 0 {
			big.Int_Set_Int_64(&dividend, 7)
		}
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Quotient(&result, &dividend, &divisor, workspace))
		test_bitwise_result(t, &result, 0)
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Remainder(&result, &dividend, &divisor, workspace))
		testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&result, &dividend))
	}
}

func test_division_result_storage(t *testing.T, workspace *big.Int_Division_Workspace) {
	dividend := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	divisor := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	big.Int_Set_Int_64(&divisor, 3)
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		quotient := big.Int{Words: make(big.Words, size)}
		remainder := big.Int{Words: make(big.Words, size)}
		big.Int_Set_Int_64(&dividend, 0)
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Divide_Modulus(
				&quotient, &remainder, &dividend, &divisor, workspace))
		testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&quotient))
		testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&remainder))
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Quotient_Remainder(
				&quotient, &remainder, &dividend, &divisor, workspace))
		testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&quotient))
		testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&remainder))
		if size != 0 {
			big.Int_Set_Int_64(&dividend, -7)
			testify.Equal_Values(t, big.STATUS_OK,
				big.Int_Divide_Modulus(
					&quotient, &remainder, &dividend, &divisor, workspace))
			converted, status := big.Int_Int_64(&quotient)
			testify.Equal_Values(t, big.STATUS_OK, status)
			testify.Equal(t, big.Int_64(-3), converted)
			converted, status = big.Int_Int_64(&remainder)
			testify.Equal_Values(t, big.STATUS_OK, status)
			testify.Equal(t, big.Int_64(2), converted)
		}
	}
}

func test_divisor_storage(t *testing.T, workspace *big.Int_Division_Workspace) {
	dividend := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	quotient := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	remainder := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	big.Int_Set_Int_64(&dividend, -7)
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		divisor := big.Int{Words: make(big.Words, size)}
		big.Int_Set_Int_64(&quotient, 11)
		big.Int_Set_Int_64(&remainder, 13)
		testify.Equal_Values(t, big.STATUS_DIVISOR_ZERO,
			big.Int_Divide_Modulus(
				&quotient, &remainder, &dividend, &divisor, workspace))
		testify.Equal(t, big.Word(11), quotient.Words[0])
		testify.Equal(t, big.Word(13), remainder.Words[0])
		if size == 0 {
			continue
		}
		big.Int_Set_Int_64(&divisor, 3)
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Divide_Modulus(
				&quotient, &remainder, &dividend, &divisor, workspace))
		converted, status := big.Int_Int_64(&quotient)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, big.Int_64(-3), converted)
		converted, status = big.Int_Int_64(&remainder)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, big.Int_64(2), converted)
	}
}

func test_rat_copy_minimum(
	t *testing.T, source, expected *big.Rat, workspace *big.Rat_Workspace,
) {
	var destination big.Rat
	destination.Integers.Numerator.Words = make(big.Words, 2)
	destination.Integers.Denominator.Words = make(big.Words, 2)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Rat_Set_Fraction_64(&destination, 1, 2, workspace))
	big.Rat_Absolute(&destination, source)
	testify.Equal(t, big.ORDER_SAME, big.Rat_Compare(&destination, expected, workspace))
	testify.Equal(t, big.Word_Count(2), destination.Integers.Numerator.Count)
	testify.Equal(t, big.Word_Count(2), destination.Integers.Denominator.Count)
}

func test_modular_inverse_fibonacci(t *testing.T) {
	const FIBONACCI_STEPS = 190
	first := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	second := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	scratch := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	big.Int_Set_Int_64(&first, 1)
	big.Int_Set_Int_64(&second, 1)
	previous, current, next := &first, &second, &scratch
	for step_index := 0; step_index < FIBONACCI_STEPS; step_index++ {
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Add(next, previous, current))
		previous, current, next = current, next, previous
	}
	var workspace big.Int_Modular_Workspace
	test_modular_workspace_initialize(&workspace)
	result := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Modular_Inverse(&result, previous, current, &workspace))
	// Cassini's identity gives F_191 squared congruent to one modulo F_192.
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&result, previous))
}

func test_rat_zero_fraction(t *testing.T, destination *big.Rat) {
	var workspace big.Rat_Workspace
	test_rat_workspace_initialize(&workspace)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Rat_Set_Fraction_64(destination, 0, -1, &workspace))
	testify.Equal(t, big.SIGN_ZERO, big.Rat_Sign(destination))
	testify.True(t, bool(big.Rat_Is_Integer(destination)))
	testify.Equal_Values(t, big.STATUS_DIVISOR_ZERO,
		big.Rat_Set_Fraction_64(destination, 0, 0, &workspace))
	testify.Equal(t, big.SIGN_ZERO, big.Rat_Sign(destination))
}

func test_rat_parse_retained_storage(t *testing.T) {
	var value big.Rat
	test_rat_initialize(&value)
	var workspace big.Rat_Parse_Workspace
	test_rat_parse_workspace_initialize(&workspace)
	for _, count := range []int{1, 2, big.WORD_COUNT_MAXIMUM} {
		denominator := (*big.Int)(&workspace.Integers.Denominator)
		words := make(big.Words_Unvalidated, count)
		words[count-1] = 1
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Words(denominator, words))
		big.Int_Negate(denominator, denominator)
		testify.Equal_Values(t, big.STATUS_OK,
			big.Rat_Parse(&value, []byte("1.5"), &workspace))
		rat_expect(t, &value, 3, 2)
	}
	for _, size := range []int{0, 1, 2} {
		workspace.Integers.Numerator = big.Int{Words: make(big.Words, size)}
		testify.Equal_Values(t, big.STATUS_INPUT_INVALID,
			big.Rat_Parse(&value, []byte("."), &workspace))
		if size == 0 {
			rat_expect(t, &value, 3, 2)
		}
		testify.Equal_Values(t, big.STATUS_OK,
			big.Rat_Parse(&value, []byte("0"), &workspace))
		rat_expect(t, &value, 0, 1)
	}
}

func test_primality_random_storage(t *testing.T) {
	prime := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	big.Int_Set_Int_64(&prime, 71)
	var workspace big.Int_Primality_Workspace
	test_primality_workspace_initialize(&workspace)
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		candidate := &workspace.Random.Integers.Candidate
		*candidate = big.Int{Words: make(big.Words, size), Count: big.Word_Count(size)}
		repetitions := big.Primality_Repetition_Count_Unvalidated(0)
		if size > 0 {
			candidate.Words[size-1] = 1
			candidate.Negative = big.POLARITY_NEGATIVE
			repetitions = 1
		}
		result, used, status := big.Int_Probably_Prime(
			&prime, test_primality_options(repetitions), []big.Word{0}, &workspace)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.True(t, bool(result))
		testify.Equal(t, big.Random_Word_Count(repetitions), used)
	}
}

func test_exponent_wide_input(t *testing.T, workspace *big.Int_Exponent_Workspace) {
	test_exponent_base_storage(t, workspace)
	test_exponent_accumulator_bound(t, workspace)
	base := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Set_Words(&base, big.Words_Unvalidated{1, 1}))
	result := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	big.Int_Set_Int_64(&result, 7)
	for _, size := range []int{2, big.WORD_COUNT_MAXIMUM} {
		exponent := big.Int{Words: make(big.Words, size), Count: big.Word_Count(size)}
		exponent.Words[size-1] = 1
		testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW,
			big.Int_Exponent(&result, &base, &exponent, workspace))
		test_bitwise_result(t, &result, 7)
	}
}

func test_exponent_base_storage(t *testing.T, workspace *big.Int_Exponent_Workspace) {
	exponent := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	result := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		result = big.Int{Words: make(big.Words, size)}
		base := big.Int{Words: make(big.Words, size)}
		expected := big.Int_64(0)
		if size != 0 {
			exponent = big.Int{Words: make(big.Words, size)}
			big.Int_Set_Int_64(&base, -5)
			expected = 25
		}
		big.Int_Set_Int_64(&exponent, 2)
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Exponent(&result, &base, &exponent, workspace))
		converted, status := big.Int_Int_64(&result)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, expected, converted)
	}
	var zero big.Int
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Exponent(&result, &zero, &zero, workspace))
	converted, status := big.Int_Int_64(&result)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, big.Int_64(1), converted)
}

func test_rat_text_scratch(t *testing.T, value *big.Rat) {
	var workspace big.Rat_Text_Workspace
	test_rat_text_workspace_initialize(&workspace)
	var output [big.RAT_TEXT_SIZE_MAXIMUM]byte
	for _, size := range []int{1, 2, big.WORD_COUNT_MAXIMUM} {
		workspace.Integers.Denominator = big.Rat_Text_Denominator{
			Words: make(big.Words, size),
		}
		for _, count := range []int{0, size} {
			words := make(big.Words_Unvalidated, count)
			if count > 0 {
				words[count-1] = 1
			}
			integer := (*big.Int)(&workspace.Integers.Denominator)
			testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Words(integer, words))
			big.Int_Negate(integer, integer)
			written, status := big.Rat_Rational_Text_Into(output[:], value, &workspace)
			testify.Equal_Values(t, big.STATUS_OK, status)
			testify.Equal(t, "-3/4", string(output[:written]))
			testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Words(integer, words))
			big.Int_Negate(integer, integer)
			fraction_count, fraction_status := big.Rat_Text_Into(
				output[:], value, &workspace)
			testify.Equal_Values(t, big.STATUS_OK, fraction_status)
			testify.Equal(t, "-3/4", string(output[:fraction_count]))
		}
	}
}

func test_greatest_common_divisor_storage(
	t *testing.T, workspace *big.Int_Greatest_Common_Divisor_Workspace,
) {
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		left := big.Int{Words: make(big.Words, size)}
		right := big.Int{Words: make(big.Words, size)}
		result := big.Int{Words: make(big.Words, size)}
		big.Int_Greatest_Common_Divisor(&result, &left, &right, workspace)
		testify.Equal(t, big.SIGN_ZERO, big.Int_Sign(&result))
		if size != 0 {
			big.Int_Set_Int_64(&left, -42)
			big.Int_Set_Int_64(&right, 30)
			big.Int_Greatest_Common_Divisor(&result, &left, &right, workspace)
			converted, status := big.Int_Int_64(&result)
			testify.Equal_Values(t, big.STATUS_OK, status)
			testify.Equal(t, big.Int_64(6), converted)
		}
	}
}

func test_greatest_common_divisor_odd_words(
	t *testing.T, workspace *big.Int_Greatest_Common_Divisor_Workspace,
) {
	test_greatest_common_divisor_storage(t, workspace)
	value := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	result := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	for _, high := range []big.Word{1, 2, big.Word(bits.WORD_64_MAXIMUM)} {
		for _, low := range []big.Word{1, 3, 5, big.Word(bits.WORD_64_MAXIMUM)} {
			testify.Equal_Values(t, big.STATUS_OK,
				big.Int_Set_Words(&value, big.Words_Unvalidated{low, high}))
			big.Int_Greatest_Common_Divisor(&result, &value, &value, workspace)
			testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&result, &value))
		}
	}
}

func test_modular_inverse_word_divisor(t *testing.T) {
	var workspace big.Int_Modular_Workspace
	test_modular_workspace_initialize(&workspace)
	value := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	modulus := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	result := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	product := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	maximum := big.Word(bits.WORD_64_MAXIMUM)
	big.Int_Set_Uint_64(&value, big.Word_64(maximum))
	for _, high := range []big.Word{2, maximum - 1} {
		for _, low := range []big.Word{2, maximum} {
			testify.Equal_Values(t, big.STATUS_OK,
				big.Int_Set_Words(&modulus, big.Words_Unvalidated{low, high}))
			testify.Equal_Values(t, big.STATUS_OK,
				big.Int_Modular_Inverse(&result, &value, &modulus, &workspace))
			testify.Equal_Values(t, big.STATUS_OK,
				big.Int_Modular_Multiply(
					&product, &result, &value, &modulus, &workspace))
			test_bitwise_result(t, &product, 1)
		}
	}
}

func test_modular_inverse_retained_quotient(t *testing.T) {
	var workspace big.Int_Modular_Workspace
	test_modular_workspace_initialize(&workspace)
	test_modular_inverse_quotient_bound(t, &workspace)
	test_modular_inverse_double_word_scratch(t, &workspace)
	value := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	modulus := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	result := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	for _, words := range []big.Words_Unvalidated{
		{0, 1}, {1 << 32}, {0, 0, 1}, {0xfffffffffffffffe, 0xffffffffffffffff},
	} {
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Words(&value, words))
		big.Int_Set(&modulus, &value)
		modulus.Words[0]++
		for _, count := range []big.Word_Count{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
			for _, index := range []int{
				big.MODULAR_QUOTIENT_INDEX, big.MODULAR_GCD_REMAINDER_INDEX,
				big.MODULAR_COEFFICIENT_REMAINDER_INDEX,
			} {
				integer := &workspace.Integers[index]
				big.Int_Set_Int_64(integer, 0)
				if count > 0 {
					integer.Count = count
					integer.Words[count-1] = 1
					integer.Negative = big.POLARITY_NEGATIVE
				}
			}
			testify.Equal_Values(t, big.STATUS_OK,
				big.Int_Modular_Inverse(&result, &value, &modulus, &workspace))
			testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&result, &value))
		}
	}
}

func test_modular_inverse_double_word_scratch(t *testing.T, workspace *big.Int_Modular_Workspace) {
	test_modular_inverse_double_word_digits(t, workspace)
	value := big.Int{Words: big.Words{2}, Count: 1}
	modulus := big.Int{Words: big.Words{1, 1}, Count: 2}
	result := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	for _, count := range []big.Word_Count{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		for _, index := range []int{
			big.MODULAR_QUOTIENT_INDEX, big.MODULAR_GCD_REMAINDER_INDEX,
			big.MODULAR_COEFFICIENT_REMAINDER_INDEX,
		} {
			integer := &workspace.Integers[index]
			big.Int_Set_Int_64(integer, 0)
			if count > 0 {
				integer.Count = count
				integer.Words[count-1] = 1
				integer.Negative = big.POLARITY_NEGATIVE
			}
		}
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Modular_Inverse(&result, &value, &modulus, workspace))
		converted, status := big.Int_Uint_64(&result)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, big.Word_64(0x8000000000000001), converted)
	}
}

func test_modular_inverse_double_word_digits(t *testing.T, workspace *big.Int_Modular_Workspace) {
	value := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM), Count: 2}
	modulus := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	result := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	one := big.Int{Words: big.Words{1}, Count: 1}
	value.Words[1] = 1
	for _, low := range []big.Word{1, 2, big.Word(bits.WORD_64_MAXIMUM)} {
		value.Words[0] = low
		// Modulus 2*value-1 makes two the exact modular inverse.
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Shift_Left(&modulus, &value, 1))
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Subtract(&modulus, &modulus, &one))
		testify.Equal_Values(t, big.STATUS_OK,
			big.Int_Modular_Inverse(&result, &value, &modulus, workspace))
		test_bitwise_result(t, &result, 2)
	}
	value.Count = 1
	value.Words[0] = big.Word(bits.WORD_64_MAXIMUM)
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Shift_Left(&modulus, &value, 1))
	testify.Equal_Values(t, big.STATUS_OK, big.Int_Subtract(&modulus, &modulus, &one))
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Modular_Inverse(&result, &value, &modulus, workspace))
	test_bitwise_result(t, &result, 2)
}

func test_modular_inverse_quotient_bound(t *testing.T, workspace *big.Int_Modular_Workspace) {
	test_modular_inverse_coefficient_bound(t, workspace)
	test_modular_inverse_word_quotient_maximum(t, workspace)
	test_modular_inverse_full_coefficient(t, workspace)
	value := big.Int{Words: big.Words{1}, Count: 1}
	modulus := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM), Count: 255}
	modulus.Words[254] = 1
	result := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	result.Count = big.WORD_COUNT_MAXIMUM
	result.Words[big.WORD_COUNT_MAXIMUM-1] = 1
	result.Negative = big.POLARITY_NEGATIVE
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Modular_Inverse(&result, &value, &modulus, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&result, &value))
}

func test_modular_inverse_word_quotient_maximum(
	t *testing.T, workspace *big.Int_Modular_Workspace,
) {
	value := big.Int{Words: big.Words{1, 0, 1}, Count: 3}
	modulus := big.Int{Words: big.Words{0, 1, big.Word(bits.WORD_64_MAXIMUM)}, Count: 3}
	expected := big.Int{Words: big.Words{1, 0, big.Word(bits.WORD_64_MAXIMUM)}, Count: 3}
	result := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	// Modulus m = value*(2^64-1)+1, so inverse(value) = m-(2^64-1).
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Modular_Inverse(&result, &value, &modulus, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&result, &expected))
}

func test_modular_inverse_full_coefficient(t *testing.T, workspace *big.Int_Modular_Workspace) {
	value := big.Int{Words: big.Words{2}, Count: 1}
	modulus := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM),
		Count: big.WORD_COUNT_MAXIMUM}
	expected := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM), Count: modulus.Count}
	modulus.Words[0], modulus.Words[big.WORD_COUNT_MAXIMUM-1] = 1, 2
	expected.Words[0], expected.Words[big.WORD_COUNT_MAXIMUM-1] = 1, 1
	result := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	// For odd m, inverse(2)=(m+1)/2; this coefficient occupies every word.
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Modular_Inverse(&result, &value, &modulus, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&result, &expected))
}

func test_modular_inverse_coefficient_bound(t *testing.T, workspace *big.Int_Modular_Workspace) {
	value := big.Int{Words: big.Words{5}, Count: 1}
	modulus := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM),
		Count: big.Word_Count(big.COEFFICIENT_WORD_COUNT_MAXIMUM)}
	expected := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM), Count: modulus.Count}
	result := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	for index := 0; index < int(modulus.Count); index++ {
		modulus.Words[index] = big.Word(bits.WORD_64_MAXIMUM)
		expected.Words[index] = 0x9999999999999999
	}
	// For m=2^N-3 with N divisible by four, inverse(5)=(3*m+1)/5.
	modulus.Words[0] -= 2
	expected.Words[0]--
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Modular_Inverse(&result, &value, &modulus, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&result, &expected))
}

func test_modular_inverse_wide_divisor(t *testing.T) {
	var workspace big.Int_Modular_Workspace
	test_modular_workspace_initialize(&workspace)
	value := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	modulus := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	result := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	value.Count, modulus.Count = big.WORD_COUNT_MAXIMUM, big.WORD_COUNT_MAXIMUM
	value.Words[big.WORD_COUNT_MAXIMUM-1] = 1
	modulus.Words[big.WORD_COUNT_MAXIMUM-1] = 2
	big.Int_Set_Int_64(&result, 7)
	testify.Equal_Values(t, big.STATUS_RESULT_ABSENT,
		big.Int_Modular_Inverse(&result, &value, &modulus, &workspace))
	test_bitwise_result(t, &result, 7)
}

func test_float_text_hexadecimal_width(t *testing.T) {
	var value big.Float
	test_float_initialize(&value)
	big.Float_Set_Int_64(&value, 1)
	big.Float_Negate(&value, &value)
	float_text_expect(t, &value, 'x', 1, "-0x1.0p+00")
	big.Float_Negate(&value, &value)
	var workspace big.Float_Text_Workspace
	test_float_text_workspace_initialize(&workspace)
	output := make(big.Text, big.FLOAT_TEXT_SIZE_MAXIMUM)
	for _, digits := range []int{
		1, big.WORD_BIT_COUNT / big.BASE_HEXADECIMAL_DIGIT_BIT_COUNT,
		(big.FLOAT_PRECISION_MAXIMUM - 1) / big.BASE_HEXADECIMAL_DIGIT_BIT_COUNT,
	} {
		written, status := big.Float_Text_Into(
			output, &value, 'x', big.Float_Text_Precision_Unvalidated(digits),
			&workspace)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, "0x1.", string(output[:4]))
		testify.Equal(t, digits+8, int(written))
		for _, digit := range output[4 : 4+digits] {
			testify.Equal(t, byte('0'), digit)
		}
		testify.Equal(t, "p+00", string(output[4+digits:written]))
	}
}

func test_float_parse_retained_storage(t *testing.T) {
	var value big.Float
	test_float_initialize(&value)
	var workspace big.Float_Parse_Workspace
	test_float_parse_workspace_initialize(&workspace)
	big.Int_Set_Int_64((*big.Int)(&workspace.Integers.Denominator), -7)
	_, parse_status := big.Float_Parse(
		&value, big.Float_Parse_Text_Unvalidated("1.5"), 0, &workspace)
	testify.Equal_Values(t, big.STATUS_OK, parse_status)
	parsed_bits, parsed_accuracy := big.Float_Float_64_Bits(&value)
	testify.Equal(t, big.Float_64_Value_Bits(FLOAT_64_THREE_HALVES_BITS), parsed_bits)
	testify.Equal(t, big.ACCURACY_EXACT, parsed_accuracy)
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		workspace.Integers.Numerator = big.Int{Words: make(big.Words, size)}
		big.Int_Set_Int_64((*big.Int)(&workspace.Integers.Denominator), -7)
		big.Float_Set_Int_64(&value, 1)
		_, status := big.Float_Parse(&value, big.Float_Parse_Text_Unvalidated("."), 0,
			&workspace)
		testify.Equal_Values(t, big.STATUS_INPUT_INVALID, status)
		encoding, accuracy := big.Float_Float_64_Bits(&value)
		testify.Equal(t, big.Float_64_Value_Bits(0x3ff0000000000000), encoding)
		testify.Equal(t, big.ACCURACY_EXACT, accuracy)
		_, status = big.Float_Parse(&value, big.Float_Parse_Text_Unvalidated("0"), 0,
			&workspace)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, big.SIGN_ZERO, big.Float_Sign(&value))
		if size > 0 {
			_, status = big.Float_Parse(
				&value, big.Float_Parse_Text_Unvalidated("1"), 0, &workspace)
			testify.Equal_Values(t, big.STATUS_OK, status)
			encoding, accuracy = big.Float_Float_64_Bits(&value)
			testify.Equal(t, big.Float_64_Value_Bits(0x3ff0000000000000), encoding)
			testify.Equal(t, big.ACCURACY_EXACT, accuracy)
		}
	}
}

func test_float_encode_retained_storage(t *testing.T, expected []byte, value *big.Float) {
	var workspace big.Float_Gob_Workspace
	var output [big.FLOAT_GOB_SIZE_MAXIMUM]byte
	for _, size := range []int{1, 2, big.WORD_COUNT_MAXIMUM} {
		workspace.Mantissas.Value = big.Int{
			Words: make(big.Words, size), Count: big.Word_Count(size),
			Negative: big.POLARITY_NEGATIVE,
		}
		workspace.Mantissas.Value.Words[size-1] = 1
		source := *value
		source.Mantissa.Words = make(big.Words, size)
		copy(source.Mantissa.Words, value.Mantissa.Words[:value.Mantissa.Count])
		count, status := big.Float_Gob_Encode_Into(output[:], &source, &workspace)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, big.Word_Count(1), count)
		testify.Equal(t, expected, output[:len(expected)])
	}
	var zero big.Float
	workspace.Mantissas.Value = big.Int{}
	count, status := big.Float_Gob_Encode_Into(output[:], &zero, &workspace)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, big.Word_Count(0), count)
	testify.Equal(t, []byte{1, 8, 0, 0, 0, 0}, output[:big.FLOAT_GOB_HEADER_SIZE])
}

func test_float_decode_retained_storage(t *testing.T, source []byte, expected *big.Float) {
	test_float_decode_storage(t, source, expected)
	var workspace big.Float_Gob_Workspace
	workspace.Mantissas.Value.Words = make(big.Words, big.WORD_COUNT_MAXIMUM)
	for _, count := range []big.Word_Count{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		for _, precision := range []big.Float_Precision{0, expected.Precision} {
			mantissa := &workspace.Mantissas.Value
			big.Int_Set_Int_64(mantissa, 0)
			if count != 0 {
				mantissa.Count = count
				mantissa.Words[count-1] = 1
				mantissa.Negative = big.POLARITY_NEGATIVE
			}
			var result big.Float
			test_float_initialize(&result)
			result.Precision = precision
			testify.Equal_Values(t, big.STATUS_OK,
				big.Float_Gob_Decode(&result, source, &workspace))
			testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&result, expected))
			testify.Equal(t, expected.Precision, result.Precision)
		}
	}
}

func test_float_decode_storage(t *testing.T, source []byte, expected *big.Float) {
	for _, size := range []int{0, 1, 2, big.WORD_COUNT_MAXIMUM} {
		var workspace big.Float_Gob_Workspace
		workspace.Mantissas.Value.Words = make(big.Words, size)
		result := big.Float{Mantissa: big.Float_Mantissa{Words: make(big.Words, size)}}
		big.Float_Set_Infinity(&result, true)
		testify.Equal_Values(t, big.STATUS_INPUT_INVALID,
			big.Float_Gob_Decode(&result, []byte{1}, &workspace))
		testify.Equal(t, big.FLOAT_FORM_INFINITY, result.Form)
		testify.Equal(t, big.POLARITY_NEGATIVE, result.Negative)
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Gob_Decode(&result, nil, &workspace))
		testify.Equal(t, big.FLOAT_FORM_ZERO, result.Form)
		testify.Equal(t, big.POLARITY_NONNEGATIVE, result.Negative)
		testify.Equal(t, size, len(result.Mantissa.Words))
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Gob_Decode(&result, []byte{1, 8, 0, 0, 0, 0}, &workspace))
		testify.Equal(t, big.FLOAT_FORM_ZERO, result.Form)
		if size != 0 {
			result.Precision = expected.Precision
			testify.Equal_Values(t, big.STATUS_OK,
				big.Float_Gob_Decode(&result, source, &workspace))
			testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&result, expected))
			big.Float_Gob_Decode(&result, nil, &workspace)
			testify.Equal_Values(t, big.STATUS_OK,
				big.Float_Gob_Decode(&result, source, &workspace))
			testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&result, expected))
		}
	}
}

func test_float_text_shortest_exponents(t *testing.T) {
	test_float_text_smallest_midpoint(t)
	test_float_fixed_precision_bounds(t)
	var value, parsed big.Float
	test_float_initialize(&value)
	test_float_initialize(&parsed)
	big.Float_Set_Int_64(&value, 1)
	parsed.Precision = value.Precision
	var text_workspace big.Float_Text_Workspace
	test_float_text_workspace_initialize(&text_workspace)
	var parse_workspace big.Float_Parse_Workspace
	test_float_parse_workspace_initialize(&parse_workspace)
	output := make(big.Text, big.FLOAT_TEXT_SIZE_MAXIMUM)
	for _, one := range []struct {
		Exponent  big.Float_Exponent
		Precision big.Float_Precision
	}{
		{big.FLOAT_EXPONENT_MINIMUM, 64}, {big.FLOAT_EXPONENT_MAXIMUM, 64},
		{5, 64}, {-5, 64}, {big.FLOAT_EXPONENT_MAXIMUM, 1}, {2, 1},
	} {
		exponent := one.Exponent
		value.Precision = one.Precision
		parsed = big.Float{Precision: one.Precision,
			Mantissa: big.Float_Mantissa{Words: parsed.Mantissa.Words}}
		value.Exponent = exponent
		count, status := big.Float_Text_Into(output, &value, 'g', -1, &text_workspace)
		testify.Equal_Values(t, big.STATUS_OK, status)
		_, parse_status := big.Float_Parse(
			&parsed, big.Float_Parse_Text_Unvalidated(output[:count]), 0,
			&parse_workspace)
		testify.Equal_Values(t, big.STATUS_OK, parse_status,
			"exponent=%d precision=%d", exponent, one.Precision)
		testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&value, &parsed))
		if exponent == 5 {
			testify.Equal(t, "16", string(output[:count]))
		}
		if exponent == -5 {
			testify.Equal(t, "0.015625", string(output[:count]))
		}
	}
}

func test_float_text_smallest_midpoint(t *testing.T) {
	value := big.Float{Precision: big.FLOAT_PRECISION_MAXIMUM,
		Form: big.FLOAT_FORM_FINITE, Exponent: big.FLOAT_EXPONENT_MINIMUM,
		Mantissa: big.Float_Mantissa{Words: big.Words{1}, Count: 1}}
	var workspace big.Float_Text_Workspace
	test_float_text_workspace_initialize(&workspace)
	output := make(big.Text, big.FLOAT_TEXT_SIZE_MAXIMUM)
	count, status := big.Float_Text_Into(output, &value, 'e', -1, &workspace)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.True(t, count > big.FLOAT_TEXT_PRECISION_MAXIMUM)
	testify.Equal(t, "3.53", string(output[:4]))
	testify.Equal(t, "e-9865", string(output[count-6:count]))
	positive := string(output[:count])
	// Fixed notation retains every leading fractional zero before the same shortest digits.
	count, status = big.Float_Text_Into(output, &value, 'f', -1, &workspace)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, "0.", string(output[:2]))
	for _, digit := range output[2:9866] {
		testify.Equal_Values(t, '0', digit)
	}
	testify.Equal(t, positive[:1]+positive[2:len(positive)-6], string(output[9866:count]))
	fixed := string(output[:count])
	big.Float_Negate(&value, &value)
	count, status = big.Float_Text_Into(output, &value, 'f', -1, &workspace)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, "-"+fixed, string(output[:count]))
	testify.Equal_Values(t, big.FLOAT_TEXT_SIZE_MAXIMUM, count)
	count, status = big.Float_Text_Into(output, &value, 'e', -1, &workspace)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, "-"+positive, string(output[:count]))
	value.Negative = big.POLARITY_NONNEGATIVE
	// Midpoint scale -32821 leaves one bit after 547 sixty-bit shifts.
	value.Precision = 52
	count, status = big.Float_Text_Into(output, &value, 'e', -1, &workspace)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, positive[:10], string(output[:10]))
	testify.Equal(t, "e-9865", string(output[count-6:count]))
	value.Precision = big.FLOAT_PRECISION_MAXIMUM - 3
	value.Exponent = big.FLOAT_EXPONENT_MAXIMUM
	test_float_text_maximum_round_trip(t, &value)
	// A small leading decimal digit can need one more digit to distinguish adjacent floats.
	value.Precision = big.FLOAT_PRECISION_MAXIMUM
	value.Exponent = big.FLOAT_EXPONENT_MINIMUM + 4
	value.Mantissa.Words[0] = big.Word(bits.WORD_64_MAXIMUM)
	value.Negative = big.POLARITY_NEGATIVE
	count, status = big.Float_Text_Into(output, &value, 'e', -1, &workspace)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, "-1.13", string(output[:5]))
	testify.Equal(t, "e-9863", string(output[count-6:count]))
	testify.Equal_Values(t, big.FLOAT_TEXT_SCIENTIFIC_COUNT_MAXIMUM, count)
	scientific := string(output[:count])
	count, status = big.Float_Text_Into(output, &value, 'g', -1, &workspace)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, scientific, string(output[:count]))
}

func test_float_fixed_precision_bounds(t *testing.T) {
	var value big.Float
	test_float_initialize(&value)
	big.Float_Set_Int_64(&value, 1)
	var workspace big.Float_Text_Workspace
	test_float_text_workspace_initialize(&workspace)
	output := make(big.Text, big.FLOAT_TEXT_SIZE_MAXIMUM)
	for _, exponent := range []big.Float_Exponent{-4, big.FLOAT_EXPONENT_MINIMUM} {
		value.Exponent = exponent
		count, status := big.Float_Text_Into(output, &value, 'f', 0, &workspace)
		testify.Equal_Values(t, big.STATUS_OK, status)
		testify.Equal(t, "0", string(output[:count]))
	}
	float_text_expect(t, &value, 'e', 2, "3.53e-9865")
	big.Float_Set_Uint_64(&value, big.Word_64(bits.WORD_64_MAXIMUM))
	value.Exponent = big.FLOAT_EXPONENT_MAXIMUM
	float_text_expect(t, &value, 'e', 2, "1.42e+9864")
	float_text_expect(t, &value, 'g', 3, "1.42e+9864")
	test_float_text_maximum_round_trip(t, &value)
	big.Float_Set_Int_64(&value, 1)
	value.Exponent = big.FLOAT_EXPONENT_MAXIMUM
	integer := make(big.Text, big.FLOAT_TEXT_SIZE_MAXIMUM)
	count, status := big.Float_Text_Into(integer, &value, 'f', 0, &workspace)
	testify.Equal_Values(t, big.STATUS_OK, status)
	written, status := big.Float_Text_Into(
		output, &value, 'f', big.FLOAT_TEXT_PRECISION_MAXIMUM, &workspace)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal_Values(t, int(count)+1+big.FLOAT_TEXT_PRECISION_MAXIMUM, written)
	testify.Equal(t, string(integer[:count]), string(output[:count]))
	testify.Equal_Values(t, '.', output[count])
	for _, digit := range output[count+1 : written] {
		testify.Equal_Values(t, '0', digit)
	}
	// One discarded fractional digit maximizes retained fixed digits before a carry.
	value.Precision = big.FLOAT_PRECISION_MAXIMUM
	value.Exponent = big.Float_Exponent(big.FLOAT_PRECISION_MAXIMUM -
		big.FLOAT_TEXT_PRECISION_MAXIMUM - 1)
	value.Mantissa.Count = big.WORD_COUNT_MAXIMUM
	for index := range value.Mantissa.Words {
		value.Mantissa.Words[index] = big.Word(bits.WORD_64_MAXIMUM)
	}
	count, status = big.Float_Text_Into(
		output, &value, 'f', big.FLOAT_TEXT_PRECISION_MAXIMUM, &workspace)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal_Values(t, 15591, count)
	// 10^8192 - 5^8192 ends in 9375; nearest-even rounding carries its odd retained digit.
	testify.Equal(t, "938", string(output[count-3:count]))
	// Minimum exponent and odd full-width mantissa maximize the exact decimal expansion.
	value.Exponent = big.FLOAT_EXPONENT_MINIMUM
	float_text_expect(t, &value, 'f', 0, "0")
	count, status = big.Float_Text_Into(output, &value, 'e', -1, &workspace)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, "7.06", string(output[:4]))
	testify.Equal(t, "e-9865", string(output[count-6:count]))
}

func test_float_text_maximum_round_trip(t *testing.T, value *big.Float) {
	var text_workspace big.Float_Text_Workspace
	test_float_text_workspace_initialize(&text_workspace)
	output := make(big.Text, big.FLOAT_TEXT_SIZE_MAXIMUM)
	count, status := big.Float_Text_Into(output, value, 'g', -1, &text_workspace)
	testify.Equal_Values(t, big.STATUS_OK, status)
	var parsed big.Float
	test_float_initialize(&parsed)
	parsed.Precision = value.Precision
	var parse_workspace big.Float_Parse_Workspace
	test_float_parse_workspace_initialize(&parse_workspace)
	_, parse_status := big.Float_Parse(
		&parsed, big.Float_Parse_Text_Unvalidated(output[:count]), 10, &parse_workspace)
	testify.Equal_Values(t, big.STATUS_OK, parse_status)
	testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&parsed, value))
}

func test_float_add_exponent_bounds(t *testing.T, workspace *big.Float_Addition_Workspace) {
	var value, expected, result big.Float
	test_float_initialize(&value)
	test_float_initialize(&expected)
	test_float_initialize(&result)
	value.Precision = 128
	value.Form = big.FLOAT_FORM_FINITE
	value.Mantissa.Count = 2
	value.Mantissa.Words[1] = 1
	for _, exponent := range []big.Float_Exponent{
		big.FLOAT_EXPONENT_MINIMUM, -1, 0, 2, big.FLOAT_EXPONENT_MAXIMUM,
	} {
		value.Exponent = exponent
		big.Float_Copy(&expected, &value)
		accuracy := big.ACCURACY_EXACT
		if exponent == big.FLOAT_EXPONENT_MAXIMUM {
			big.Float_Set_Infinity(&expected, false)
			accuracy = big.ACCURACY_ABOVE
		} else {
			expected.Exponent++
		}
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Add(&result, &value, &value, workspace))
		testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&result, &expected))
		testify.Equal(t, accuracy, result.Accuracy)
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Subtract(&result, &value, &value, workspace))
		testify.Equal(t, big.SIGN_ZERO, big.Float_Sign(&result))
		testify.Equal(t, big.ACCURACY_EXACT, result.Accuracy)
	}
}

func test_float_add_retained_mantissa(t *testing.T, workspace *big.Float_Addition_Workspace) {
	var value big.Float
	test_float_initialize(&value)
	value.Precision = big.FLOAT_PRECISION_MAXIMUM
	value.Form = big.FLOAT_FORM_FINITE
	value.Exponent = big.WORD_BIT_COUNT + 1
	value.Mantissa.Count = 2
	value.Mantissa.Words[1] = 1
	for _, size := range []int{2, big.WORD_COUNT_MAXIMUM} {
		var result big.Float
		result.Precision = big.FLOAT_PRECISION_MAXIMUM
		result.Form = big.FLOAT_FORM_FINITE
		result.Exponent = big.FLOAT_EXPONENT_MINIMUM
		result.Mantissa.Words = make(big.Words, size)
		result.Mantissa.Count = big.Word_Count(size)
		result.Mantissa.Words[size-1] = 1
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Add(&result, &value, &value, workspace))
		testify.Equal(t, big.Word_Count(2), result.Mantissa.Count)
		testify.Equal(t, big.Words{0, 2}, result.Mantissa.Words[:2])
		testify.Equal(t, big.Float_Exponent(big.WORD_BIT_COUNT+2), result.Exponent)
		testify.Equal(t, big.ACCURACY_EXACT, result.Accuracy)
		for _, word := range result.Mantissa.Words[2:] {
			testify.Equal(t, big.Word(0), word)
		}
		var sum big.Float
		test_float_initialize(&sum)
		big.Float_Copy(&sum, &result)
		result.Mantissa.Count = big.Word_Count(size)
		result.Mantissa.Words[size-1] = 1
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Subtract(&result, &sum, &value, workspace))
		testify.Equal(t, big.Word_Count(2), result.Mantissa.Count)
		testify.Equal(t, big.Words{0, 1}, result.Mantissa.Words[:2])
		testify.Equal(t, value.Exponent, result.Exponent)
		testify.Equal(t, big.ACCURACY_EXACT, result.Accuracy)
		for _, word := range result.Mantissa.Words[2:] {
			testify.Equal(t, big.Word(0), word)
		}
	}
}

func test_float_multiply_exponent_bounds(t *testing.T) {
	var value, one, result big.Float
	for _, number := range []*big.Float{&value, &one, &result} {
		test_float_initialize(number)
		number.Precision = 128
	}
	value.Form = big.FLOAT_FORM_FINITE
	value.Exponent = 1
	value.Mantissa.Count = 2
	value.Mantissa.Words[1] = 1
	big.Float_Copy(&one, &value)
	var workspace big.Float_Multiplication_Workspace
	workspace.Result = make(big.Float_Arithmetic_Words, big.FLOAT_ADDITION_WORD_COUNT_MAXIMUM)
	for _, exponent := range []big.Float_Exponent{
		big.FLOAT_EXPONENT_MINIMUM, -1, 0, 2, big.FLOAT_EXPONENT_MAXIMUM,
	} {
		value.Exponent = exponent
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Multiply(&result, &value, &one, &workspace))
		testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&result, &value))
		testify.Equal(t, big.ACCURACY_EXACT, result.Accuracy)
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Multiply(&result, &one, &value, &workspace))
		testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&result, &value))
		testify.Equal(t, big.ACCURACY_EXACT, result.Accuracy)
	}
}

func test_float_multiply_retained_mantissa(t *testing.T) {
	var value, result big.Float
	test_float_initialize(&value)
	test_float_initialize(&result)
	value.Precision = big.FLOAT_PRECISION_MAXIMUM
	value.Form = big.FLOAT_FORM_FINITE
	value.Exponent = big.WORD_BIT_COUNT + 1
	value.Mantissa.Count = 2
	value.Mantissa.Words[1] = 1
	result.Precision = big.FLOAT_PRECISION_MAXIMUM
	result.Form = big.FLOAT_FORM_FINITE
	result.Exponent = big.FLOAT_EXPONENT_MINIMUM
	result.Mantissa.Count = big.WORD_COUNT_MAXIMUM
	result.Mantissa.Words[big.WORD_COUNT_MAXIMUM-1] = 1
	var workspace big.Float_Multiplication_Workspace
	workspace.Result = make(big.Float_Arithmetic_Words, big.FLOAT_ADDITION_WORD_COUNT_MAXIMUM)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Multiply(&result, &value, &value, &workspace))
	testify.Equal(t, big.Word_Count(3), result.Mantissa.Count)
	testify.Equal(t, big.Words{0, 0, 1}, result.Mantissa.Words[:3])
	testify.Equal(t, big.Float_Exponent(2*big.WORD_BIT_COUNT+1), result.Exponent)
	testify.Equal(t, big.ACCURACY_EXACT, result.Accuracy)
	for _, word := range result.Mantissa.Words[3:] {
		testify.Equal(t, big.Word(0), word)
	}
}

func test_float_multiply_empty_underflow(t *testing.T) {
	var workspace big.Float_Multiplication_Workspace
	workspace.Result = make(big.Float_Arithmetic_Words, big.FLOAT_ADDITION_WORD_COUNT_MAXIMUM)
	for _, count := range []int{1, 2} {
		value := big.Float{Precision: 128, Form: big.FLOAT_FORM_FINITE,
			Exponent: big.FLOAT_EXPONENT_MINIMUM,
			Mantissa: big.Float_Mantissa{Words: make(big.Words, count),
				Count: big.Word_Count(count)}}
		value.Mantissa.Words[count-1] = 1
		result := big.Float{Precision: 128}
		testify.Equal_Values(t, big.STATUS_OK,
			big.Float_Multiply(&result, &value, &value, &workspace))
		testify.Equal(t, big.FLOAT_FORM_ZERO, result.Form)
		testify.Equal(t, big.ACCURACY_BELOW, result.Accuracy)
		testify.Equal(t, big.POLARITY_NONNEGATIVE, result.Negative)
	}
}

func test_float_multiply_retained_storage(t *testing.T) {
	test_float_multiply_empty_underflow(t)
	var left, right big.Float
	test_float_initialize(&left)
	test_float_initialize(&right)
	big.Float_Set_Int_64(&left, 2)
	big.Float_Set_Int_64(&right, 3)
	var workspace big.Float_Multiplication_Workspace
	workspace.Result = make(big.Float_Arithmetic_Words, big.FLOAT_ADDITION_WORD_COUNT_MAXIMUM)
	for _, size := range []int{1, 2, big.WORD_COUNT_MAXIMUM} {
		for _, exponent := range []big.Float_Exponent{big.FLOAT_EXPONENT_MINIMUM, 2} {
			var result big.Float
			result.Mantissa.Words = make(big.Words, size)
			result.Mantissa.Count = big.Word_Count(size)
			result.Mantissa.Words[size-1] = 1
			result.Precision = big.FLOAT_PRECISION_MAXIMUM
			result.Form = big.FLOAT_FORM_FINITE
			result.Exponent = exponent
			testify.Equal_Values(t, big.STATUS_OK,
				big.Float_Multiply(&result, &left, &right, &workspace))
			testify.Equal(t, big.Word_Count(1), result.Mantissa.Count)
			testify.Equal(t, big.Word(6), result.Mantissa.Words[0])
			testify.Equal(t, big.Float_Exponent(3), result.Exponent)
			testify.Equal(t, big.ACCURACY_EXACT, result.Accuracy)
			for _, word := range result.Mantissa.Words[1:] {
				testify.Equal(t, big.Word(0), word)
			}
		}
	}
}

func test_exponent_accumulator_bound(t *testing.T, workspace *big.Int_Exponent_Workspace) {
	base := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	exponent := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	result := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	big.Int_Set_Uint_64(&base, big.Word_64(bits.WORD_64_MAXIMUM))
	big.Int_Set_Int_64(&exponent, 1)
	testify.Equal_Values(t, big.STATUS_OK,
		big.Int_Exponent(&result, &base, &exponent, workspace))
	testify.Equal(t, big.ORDER_SAME, big.Int_Compare(&result, &base))
	big.Int_Set_Int_64(&base, 65535)
	big.Int_Set_Int_64(&exponent, 4095)
	big.Int_Set_Int_64(&result, 7)
	testify.Equal_Values(t, big.STATUS_VALUE_OVERFLOW,
		big.Int_Exponent(&result, &base, &exponent, workspace))
	test_bitwise_result(t, &result, 7)
	testify.Equal(t, big.Word_Count(big.WORD_COUNT_MAXIMUM), workspace.Integers.Result.Count)
}

func test_float_serialization_empty_overflow(t *testing.T) {
	value := big.Float{Precision: 64, Form: big.FLOAT_FORM_FINITE,
		Exponent: big.FLOAT_EXPONENT_MAXIMUM,
		Mantissa: big.Float_Mantissa{
			Words: big.Words{big.Word(bits.WORD_64_MAXIMUM)}, Count: 1}}
	var workspace big.Float_Gob_Workspace
	workspace.Mantissas.Value.Words = make(big.Words, big.WORD_COUNT_MAXIMUM)
	var encoded [big.FLOAT_GOB_FINITE_PREFIX_SIZE + big.WORD_BYTE_COUNT]byte
	count, status := big.Float_Gob_Encode_Into(encoded[:], &value, &workspace)
	testify.Equal_Values(t, big.STATUS_OK, status)
	testify.Equal(t, big.Word_Count(1), count)
	result := big.Float{Precision: 1}
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Gob_Decode(&result, encoded[:], &workspace))
	testify.Equal(t, big.FLOAT_FORM_INFINITY, result.Form)
	testify.Equal(t, big.ACCURACY_ABOVE, result.Accuracy)
	workspace = big.Float_Gob_Workspace{}
	result = big.Float{Precision: 64,
		Mantissa: big.Float_Mantissa{Words: make(big.Words, 1)}}
	testify.Equal_Values(t, big.STATUS_OK,
		big.Float_Gob_Decode(&result, encoded[:], &workspace))
	testify.Equal(t, big.ORDER_SAME, big.Float_Compare(&result, &value))
}

func test_float_serialization_double_words(t *testing.T) {
	test_float_serialization_empty_overflow(t)
	const DOUBLE_WORD_COUNT = 2
	const DOUBLE_WORD_BYTE_COUNT = DOUBLE_WORD_COUNT * big.WORD_BYTE_COUNT
	var value big.Float
	test_float_initialize(&value)
	testify.Equal_Values(t, big.STATUS_OK, big.Float_Set_Precision(&value, 128))
	integer := big.Int{Words: make(big.Words, big.WORD_COUNT_MAXIMUM)}
	var workspace big.Float_Gob_Workspace
	workspace.Mantissas.Value.Words = make(big.Words, big.WORD_COUNT_MAXIMUM)
	var storage [big.FLOAT_GOB_FINITE_PREFIX_SIZE + DOUBLE_WORD_BYTE_COUNT]byte
	for _, test := range []struct {
		Words    [DOUBLE_WORD_COUNT]big.Word
		Expected [DOUBLE_WORD_BYTE_COUNT]byte
	}{
		{[DOUBLE_WORD_COUNT]big.Word{0, 1}, [DOUBLE_WORD_BYTE_COUNT]byte{0x80}},
		{[DOUBLE_WORD_COUNT]big.Word{1, 2},
			[DOUBLE_WORD_BYTE_COUNT]byte{0x80, 0, 0, 0, 0, 0, 0, 0, 0x40}},
		{[DOUBLE_WORD_COUNT]big.Word{2, 1},
			[DOUBLE_WORD_BYTE_COUNT]byte{0x80, 0, 0, 0, 0, 0, 0, 1}},
		{[DOUBLE_WORD_COUNT]big.Word{
			big.Word(bits.WORD_64_MAXIMUM), big.Word(bits.WORD_64_MAXIMUM)},
			[DOUBLE_WORD_BYTE_COUNT]byte{255, 255, 255, 255, 255, 255, 255, 255,
				255, 255, 255, 255, 255, 255, 255, 255}},
	} {
		testify.Equal_Values(t, big.STATUS_OK, big.Int_Set_Words(&integer, test.Words[:]))
		big.Float_Set_Int(&value, &integer)
		for _, exponent := range []big.Float_Exponent{
			value.Exponent, big.FLOAT_EXPONENT_MINIMUM, big.FLOAT_EXPONENT_MAXIMUM,
			-1, 0, 1, 2,
		} {
			value.Exponent = exponent
			count, status := big.Float_Gob_Encode_Into(storage[:], &value, &workspace)
			testify.Equal_Values(t, big.STATUS_OK, status)
			testify.Equal(t, big.Word_Count(2), count)
			testify.Equal(t, test.Expected[:], storage[big.FLOAT_GOB_MANTISSA_OFFSET:])
		}
	}
}
