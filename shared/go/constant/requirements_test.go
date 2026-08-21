package constant_test

import (
	"testing"

	"local/james-orcales/shared/go/constant"
	"local/james-orcales/shared/math/integer"
	"local/james-orcales/shared/testify"
)

// Test_Values binds the Values specification leaf before fixture declarations.
func Test_Values(t *testing.T) {
	test_values(t)
}

// Test_Kinds binds the Kinds specification leaf before fixture declarations.
func Test_Kinds(t *testing.T) {
	test_kinds(t)
}

// Test_Construction binds the Construction specification leaf before fixture declarations.
func Test_Construction(t *testing.T) {
	test_construction(t)
}

// Test_Reading binds the Reading specification leaf before fixture declarations.
func Test_Reading(t *testing.T) {
	test_reading(t)
}

// Test_Unary_Operations binds the Unary Operations leaf before fixture declarations.
func Test_Unary_Operations(t *testing.T) {
	test_unary_operations(t)
}

// Test_Binary_Operations binds the Binary Operations leaf before fixture declarations.
func Test_Binary_Operations(t *testing.T) {
	test_binary_operations(t)
}

// Test_Comparison binds the Comparison specification leaf before fixture declarations.
func Test_Comparison(t *testing.T) {
	test_comparison(t)
}

// Test_Shifts binds the Shifts specification leaf before fixture declarations.
func Test_Shifts(t *testing.T) {
	test_shifts(t)
}

// Test_Bounds binds the Bounds specification leaf before fixture declarations.
func Test_Bounds(t *testing.T) {
	test_bounds(t)
}

// Test_Allocation binds the Allocation specification leaf before fixture declarations.
func Test_Allocation(t *testing.T) {
	test_allocation(t)
}

const TEST_STORAGE_SIZE = constant.FORM_SIZE_MAXIMUM

// TEST_WIDEST_WHOLE is the largest whole number the limbs hold, two to the five hundred eleventh
// less one. Its complement is the most negative whole number, which has no positive of its own
// width.
const TEST_WIDEST_WHOLE = "670390396497129854978701249910292306373968291029619668886178072186" +
	"0882015036773488400937149083451713845015929093243025426876941405973284973216824503042047"

type allocation_fixture struct {
	Value  constant.Value
	Ok     constant.Read
	Order  integer.Order
	Count  constant.Form_Count
	Kind   constant.Kind
	Truth  constant.Boolean
	Text   constant.Text
	Number constant.Int_64
}

// Widens one ratio fold to a value, which is what a case states its expectation against.
func ratio_constant(numerator constant.Int_64, denominator constant.Int_64) (value constant.Value) {
	folded := constant.Make_Ratio(numerator, denominator)
	if !bool(folded.OK) {
		return constant.Make(constant.KIND_UNKNOWN, "", constant.Zero_Pair())
	}
	return constant.Make(constant.Kind(folded.Kind), "", folded.Numbers)
}

// Widens one truth to a value, which is what a case states its expectation against.
func truth_constant(value constant.Boolean) (folded constant.Value) {
	return constant.Make(constant.KIND_BOOLEAN, "", constant.Truth_Pair(value))
}

// Widens one machine integer to a value.
func whole_constant(value constant.Int_64) (folded constant.Value) {
	return constant.Make(constant.KIND_INT, "", constant.Int_64_Pair(value))
}

// Widens one view of the source to a value.
func view_constant(value constant.Text) (folded constant.Value) {
	return constant.Make(constant.KIND_TEXT, value, constant.Zero_Pair())
}

// Widens one pair of machine integers to a value.
func pair_constant(real_part constant.Int_64, imaginary_part constant.Int_64) (
	folded constant.Value,
) {
	return constant.Make(constant.KIND_COMPLEX, "", constant.Make_Complex(
		constant.Int_64_Pair(real_part).Real.Value,
		constant.Int_64_Pair(imaginary_part).Real.Value))
}

// Widens one shift to a value. Only a whole number admits a shift, thus the kind stands here and
// the package step takes the numbers alone.
func shift_constant(
	value constant.Value, operation constant.Shift_Operation, count integer.Shift_Count,
) (folded constant.Value) {
	if constant.Kind_Of(value) != constant.KIND_INT {
		return constant.Make(constant.KIND_UNKNOWN, "", constant.Zero_Pair())
	}
	moved := constant.Shift(constant.Pair_Of(value), operation, count)
	if !bool(moved.OK) {
		return constant.Make(constant.KIND_UNKNOWN, "", constant.Zero_Pair())
	}
	return constant.Make(constant.KIND_INT, "", moved.Numbers)
}

// Widens one unary fold to a value.
func unary_constant(
	operation constant.Unary, value constant.Value,
) (folded constant.Value) {
	one := constant.Unary_Operation(operation, value)
	if !bool(one.OK) {
		return constant.Make(constant.KIND_UNKNOWN, "", constant.Zero_Pair())
	}
	return constant.Make(constant.Kind(one.Kind), "", one.Numbers)
}

// Widens one literal fold to a value.
func literal_constant(text constant.Text) (folded constant.Value) {
	spelled := constant.Make_From_Literal(text)
	if !bool(spelled.OK) {
		return constant.Make(constant.KIND_UNKNOWN, "", constant.Zero_Pair())
	}
	return constant.Make(constant.Kind(spelled.Kind), "", spelled.Numbers)
}

// Widens one binary fold to a value.
func binary_constant(
	left constant.Value, operation constant.Binary, right constant.Value,
) (folded constant.Value) {
	one := constant.Binary_Operation(left, operation, right)
	if !bool(one.OK) {
		return constant.Make(constant.KIND_UNKNOWN, "", constant.Zero_Pair())
	}
	return constant.Make(constant.Kind(one.Kind), one.View, one.Numbers)
}

// Reads one value back as text, which is how every case states what it expects.
func text_of(value constant.Value) (text string) {
	var storage [TEST_STORAGE_SIZE]byte
	count := constant.Into_Text(storage[:], value)
	return string(storage[:count])
}

func test_values(t *testing.T) {
	unknown := constant.Make(constant.KIND_UNKNOWN, "", constant.Zero_Pair())
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(unknown),
		"an unknown names no kind")
	testify.Equal(t, "", text_of(unknown), "an unknown writes nothing")
	testify.Equal(t, constant.KIND_UNKNOWN,
		constant.Kind_Of(unary_constant(constant.UNARY_MINUS, unknown)),
		"an operation on an unknown stays unknown")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(binary_constant(
		unknown, constant.BINARY_ADD, whole_constant(1))),
		"one unknown operand keeps the sum unknown")
	// Every entry point takes a value of any kind and a view of any width. A corpus that
	// hands one of them a whole number alone states nothing about the kinds it refuses.
	wide := constant.Text(make([]byte, constant.TEXT_SIZE_MAXIMUM))
	every := []constant.Value{
		unknown, truth_constant(true), view_constant(""), view_constant("a"),
		view_constant("ab"), view_constant(wide), whole_constant(1),
		ratio_constant(1, 2), pair_constant(3, 4),
	}
	numbers := 0
	for _, one := range every {
		if !bool(constant.Compare(one, one).OK) {
			continue
		}
		numbers++
	}
	testify.Equal(t, 8, numbers, "an unknown alone admits no comparison with itself")
	var storage [TEST_STORAGE_SIZE]byte
	written := 0
	for _, left := range every {
		for _, right := range every {
			binary_constant(left, constant.BINARY_ADD, right)
		}
		unary_constant(constant.UNARY_MINUS, left)
		constant.Pair_Of(left)
		if constant.Into_Text(storage[:], left) == 0 {
			continue
		}
		written++
	}
	testify.Equal(t, 7, written, "an unknown and an empty view alone write nothing")
	value_width_cases(t)
}

// A pair whose parts stand at the width the limbs hold refuses every operation that needs one
// more bit. Nothing short of that width reaches the refusal.
func value_width_cases(t *testing.T) {
	// Five followed by nines is the widest literal whose own sum leaves the limbs. One digit
	// more and the literal itself no longer folds.
	digits := make([]byte, 154)
	digits[0] = '5'
	for index := 1; index < len(digits); index++ {
		digits[index] = '9'
	}
	brink := constant.Pair_Of(literal_constant(constant.Text(digits))).Real.Value
	edge := constant.Make(constant.KIND_COMPLEX, "", constant.Make_Complex(brink, brink))
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(binary_constant(
		edge, constant.BINARY_ADD, edge)), "a sum past the width folds to no value")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(binary_constant(
		edge, constant.BINARY_MULTIPLY, edge)),
		"a product past the width folds to no value")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(binary_constant(
		edge, constant.BINARY_QUOTIENT, edge)),
		"a quotient whose divisor squares past the width folds to no value")
	// The most negative whole number has no positive of its own width, thus reversing it is
	// the one reversal the limbs refuse, and a divisor that carries it refuses its conjugate.
	widest := literal_constant(constant.Text(TEST_WIDEST_WHOLE))
	testify.Equal(t, constant.KIND_INT, constant.Kind_Of(widest),
		"the widest whole number folds")
	least := unary_constant(constant.UNARY_COMPLEMENT, widest)
	testify.Equal(t, constant.KIND_INT, constant.Kind_Of(least),
		"the complement of the widest whole number folds")
	turned := constant.Make(constant.KIND_COMPLEX, "", constant.Make_Complex(
		constant.Int_64_Pair(1).Real.Value, constant.Pair_Of(least).Real.Value))
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(binary_constant(
		turned, constant.BINARY_QUOTIENT, turned)),
		"a divisor the conjugate cannot reverse folds to no value")
}

func test_kinds(t *testing.T) {
	testify.Equal(t, constant.KIND_BOOLEAN, constant.Kind_Of(truth_constant(true)),
		"a truth names a truth")
	testify.Equal(t, constant.KIND_TEXT, constant.Kind_Of(view_constant("a")),
		"a view names text")
	testify.Equal(t, constant.KIND_INT, constant.Kind_Of(whole_constant(1)),
		"a machine integer names a whole number")
	testify.Equal(t, constant.KIND_INT, constant.Kind_Of(ratio_constant(4, 2)),
		"a ratio over one names a whole number")
	testify.Equal(t, constant.KIND_FLOAT, constant.Kind_Of(ratio_constant(1, 2)),
		"a ratio that is no whole one names a number")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(ratio_constant(1, 0)),
		"a zero denominator names no value")
}

func test_construction(t *testing.T) {
	testify.Equal(t, "42", text_of(whole_constant(42)), "a machine integer folds")
	testify.Equal(t, "-42", text_of(whole_constant(-42)),
		"a negative machine integer folds")
	testify.Equal(t, "1/2", text_of(ratio_constant(2, 4)), "a ratio folds in lowest terms")
	testify.Equal(t, "true", text_of(truth_constant(true)), "a truth writes its word")
	testify.Equal(t, "false", text_of(truth_constant(false)),
		"a falsehood writes its word")
	testify.Equal(t, "text", text_of(view_constant("text")), "a view writes itself")
	for _, one := range []struct{ Literal, Want string }{
		{"0", "0"}, {"42", "42"}, {"0xff", "255"}, {"0b1010", "10"}, {"1_000", "1000"},
		{"18446744073709551616", "18446744073709551616"},
	} {
		testify.Equal(t, one.Want, text_of(literal_constant(
			constant.Text(one.Literal))), "the literal %q folds", one.Literal)
	}
	testify.Equal(t, constant.KIND_COMPLEX, constant.Kind_Of(literal_constant("3i")),
		"the imaginary mark makes a pair of a literal")
	testify.Equal(t, "(0 + 3i)", text_of(literal_constant("3i")),
		"an imaginary literal folds to a pair whose real part is zero")
	testify.Equal(t, constant.KIND_UNKNOWN,
		constant.Kind_Of(literal_constant("abc")),
		"a literal the grammar refuses folds unknown")
}

func test_reading(t *testing.T) {
	result := constant.Boolean_Value(truth_constant(true))
	truth, ok := result.Value, result.OK
	testify.True(t, bool(ok), "a truth reads back")
	testify.True(t, bool(truth), "a truth reads its own value")
	wrong := constant.Boolean_Value(whole_constant(1)).OK
	testify.False(t, bool(wrong), "a whole number reads as no truth")
	text_result := constant.Text_Value(view_constant("abc"))
	text, ok := text_result.Value, text_result.OK
	testify.True(t, bool(ok), "a view reads back")
	testify.Equal(t, constant.Text("abc"), text, "a view reads its own bytes")
	absent := constant.Text_Value(truth_constant(true)).OK
	testify.False(t, bool(absent), "a truth reads as no view")
	number_result := constant.Int_64_Value(whole_constant(-7))
	number, ok := number_result.Value, number_result.OK
	testify.True(t, bool(ok), "a whole number reads back")
	testify.Equal(t, constant.Int_64(-7), number, "a whole number reads its own value")
	fractional := constant.Int_64_Value(ratio_constant(1, 2)).OK
	testify.False(t, bool(fractional), "a number that is no whole one reads as none")
	// Every reader takes a value of any kind, thus every kind and a text of every width
	// reaches each of them here. A reader the corpus never hands an unknown states nothing
	// about the kind it refuses.
	wide := constant.Text(make([]byte, constant.TEXT_SIZE_MAXIMUM))
	reads := 0
	for _, one := range []constant.Value{
		constant.Make(constant.KIND_UNKNOWN, "", constant.Zero_Pair()),
		truth_constant(false),
		view_constant("a"),
		view_constant("ab"),
		view_constant(wide),
		whole_constant(1),
		ratio_constant(1, 2),
		pair_constant(3, 4),
	} {
		if bool(constant.Int_64_Value(one).OK) {
			reads++
		}
		if bool(constant.Boolean_Value(one).OK) {
			reads++
		}
		if bool(constant.Text_Value(one).OK) {
			reads++
		}
	}
	testify.Equal(t, 5, reads, "one kind alone reads back through each reader")
}

func test_unary_operations(t *testing.T) {
	testify.Equal(t, "42", text_of(unary_constant(
		constant.UNARY_PLUS, whole_constant(42))), "a plus leaves a number")
	testify.Equal(t, "-42", text_of(unary_constant(
		constant.UNARY_MINUS, whole_constant(42))), "a minus reverses a number")
	testify.Equal(t, "-1/2", text_of(unary_constant(
		constant.UNARY_MINUS, ratio_constant(1, 2))), "a minus reverses a ratio")
	testify.Equal(t, "-43", text_of(unary_constant(
		constant.UNARY_COMPLEMENT, whole_constant(42))),
		"a complement flips every bit")
	testify.Equal(t, "false", text_of(unary_constant(
		constant.UNARY_NOT, truth_constant(true))), "a not reverses a truth")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(unary_constant(
		constant.UNARY_NOT, whole_constant(1))), "a not of a number is unknown")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(unary_constant(
		constant.UNARY_COMPLEMENT, ratio_constant(1, 2))),
		"a complement of a ratio is unknown")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(unary_constant(
		constant.UNARY_PLUS, truth_constant(true))), "a plus of a truth is unknown")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(unary_constant(
		constant.UNARY_MINUS, view_constant("a"))), "a minus of a view is unknown")
}

func test_binary_operations(t *testing.T) {
	binary_number_cases(t)
	binary_pair_cases(t)
	binary_bit_cases(t)
	binary_text_cases(t)
}

func binary_number_cases(t *testing.T) {
	half := ratio_constant(1, 2)
	third := ratio_constant(1, 3)
	for _, one := range []struct {
		Left      constant.Value
		Operation constant.Binary
		Right     constant.Value
		Want      string
	}{
		{whole_constant(6), constant.BINARY_ADD, whole_constant(7), "13"},
		{whole_constant(6), constant.BINARY_SUBTRACT, whole_constant(7), "-1"},
		{whole_constant(6), constant.BINARY_MULTIPLY, whole_constant(7), "42"},
		{whole_constant(43), constant.BINARY_QUOTIENT, whole_constant(7), "6"},
		{whole_constant(-43), constant.BINARY_QUOTIENT,
			whole_constant(7), "-6"},
		{half, constant.BINARY_ADD, third, "5/6"},
		{half, constant.BINARY_QUOTIENT, third, "3/2"},
		{half, constant.BINARY_MULTIPLY, whole_constant(4), "2"},
	} {
		testify.Equal(t, one.Want, text_of(binary_constant(
			one.Left, one.Operation, one.Right)),
			"the operation %d folds", one.Operation)
	}
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(binary_constant(
		whole_constant(1), constant.BINARY_QUOTIENT, whole_constant(0))),
		"a quotient over zero folds unknown")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(binary_constant(
		truth_constant(true), constant.BINARY_ADD, whole_constant(1))),
		"a sum of a truth and a number folds unknown")
}

func binary_pair_cases(t *testing.T) {
	pair := pair_constant(3, 4)
	other := pair_constant(1, 2)
	for _, one := range []struct {
		Operation constant.Binary
		Want      string
	}{
		{constant.BINARY_ADD, "(4 + 6i)"},
		{constant.BINARY_SUBTRACT, "(2 + 2i)"},
		{constant.BINARY_MULTIPLY, "(-5 + 10i)"},
		{constant.BINARY_QUOTIENT, "(11/5 + -2/5i)"},
	} {
		testify.Equal(t, one.Want, text_of(binary_constant(
			pair, one.Operation, other)), "the pair operation %d folds", one.Operation)
	}
	testify.Equal(t, "(4 + 4i)", text_of(binary_constant(
		pair, constant.BINARY_ADD, whole_constant(1))),
		"a number adds to the real part of a pair alone")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(binary_constant(
		pair, constant.BINARY_REMAINDER, other)), "a remainder of two pairs folds unknown")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(binary_constant(
		pair, constant.BINARY_QUOTIENT, pair_constant(0, 0))),
		"a quotient over the pair of zeroes folds unknown")
}

func binary_bit_cases(t *testing.T) {
	for _, one := range []struct {
		Left      constant.Int_64
		Operation constant.Binary
		Right     constant.Int_64
		Want      string
	}{
		{12, constant.BINARY_AND, 10, "8"},
		{12, constant.BINARY_OR, 10, "14"},
		{12, constant.BINARY_EXCLUSIVE_OR, 10, "6"},
		{12, constant.BINARY_AND_NOT, 10, "4"},
		{43, constant.BINARY_REMAINDER, 7, "1"},
		{-43, constant.BINARY_REMAINDER, 7, "-1"},
	} {
		testify.Equal(t, one.Want, text_of(binary_constant(
			whole_constant(one.Left), one.Operation,
			whole_constant(one.Right))),
			"the bit operation %d folds", one.Operation)
	}
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(binary_constant(
		ratio_constant(1, 2), constant.BINARY_AND, whole_constant(1))),
		"a bit operation on a ratio folds unknown")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(binary_constant(
		whole_constant(1), constant.BINARY_AND, ratio_constant(1, 2))),
		"a bit operation over a ratio folds unknown")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(binary_constant(
		whole_constant(1), constant.BINARY_REMAINDER, whole_constant(0))),
		"a remainder over zero folds unknown")
}

func binary_text_cases(t *testing.T) {
	testify.Equal(t, "abc", text_of(binary_constant(
		view_constant(""), constant.BINARY_ADD, view_constant("abc"))),
		"an empty view joins to the view behind it")
	testify.Equal(t, "abc", text_of(binary_constant(
		view_constant("abc"), constant.BINARY_ADD, view_constant(""))),
		"a view joins to the empty view behind it")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(binary_constant(
		view_constant("a"), constant.BINARY_ADD, view_constant("b"))),
		"two views join to a text no source states")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(binary_constant(
		view_constant("a"), constant.BINARY_ADD, whole_constant(1))),
		"a view joins to no number")
}

func test_comparison(t *testing.T) {
	for _, one := range []struct {
		Left  constant.Value
		Right constant.Value
		Order integer.Order
	}{
		{whole_constant(1), whole_constant(1), integer.ORDER_SAME},
		{whole_constant(2), whole_constant(1), integer.ORDER_AFTER},
		{whole_constant(1), whole_constant(2), integer.ORDER_BEFORE},
		{ratio_constant(1, 3), ratio_constant(1, 2), integer.ORDER_BEFORE},
		{truth_constant(false), truth_constant(true), integer.ORDER_BEFORE},
		{truth_constant(true), truth_constant(false), integer.ORDER_AFTER},
		{truth_constant(true), truth_constant(true), integer.ORDER_SAME},
		{view_constant("abc"), view_constant("abc"), integer.ORDER_SAME},
		{view_constant("abc"), view_constant("abd"), integer.ORDER_BEFORE},
		{view_constant("abd"), view_constant("abc"), integer.ORDER_AFTER},
		{view_constant("ab"), view_constant("abc"), integer.ORDER_BEFORE},
		{view_constant("abc"), view_constant("ab"), integer.ORDER_AFTER},
	} {
		result := constant.Compare(one.Left, one.Right)
		order, ok := result.Order, result.OK
		testify.True(t, bool(ok), "a pair of one kind compares")
		testify.Equal(t, one.Order, order, "the order reads as the values state")
	}
	compare_pair_cases(t)
}

func compare_pair_cases(t *testing.T) {
	pair := pair_constant(3, 4)
	same := pair_constant(3, 4)
	wider := pair_constant(3, 5)
	result := constant.Compare(pair, same)
	order, ok := result.Order, result.OK
	testify.True(t, bool(ok), "two pairs compare")
	testify.Equal(t, integer.ORDER_SAME, order, "two equal pairs read the same order")
	result = constant.Compare(pair, wider)
	order, ok = result.Order, result.OK
	testify.True(t, bool(ok), "two pairs of one real part compare")
	testify.Equal(t, integer.ORDER_BEFORE, order, "a smaller imaginary part orders first")
	mixed := constant.Compare(whole_constant(1), view_constant("a")).OK
	testify.False(t, bool(mixed), "two kinds compare as no order")
	none := constant.Make(constant.KIND_UNKNOWN, "", constant.Zero_Pair())
	unknown := constant.Compare(none, none).OK
	testify.False(t, bool(unknown), "two unknowns compare as no order")
}

func test_shifts(t *testing.T) {
	for _, one := range []struct {
		Value     constant.Int_64
		Operation constant.Shift_Operation
		Count     integer.Shift_Count
		Want      string
	}{
		{1, constant.SHIFT_UP, 0, "1"},
		{1, constant.SHIFT_UP, 1, "2"},
		{1, constant.SHIFT_UP, 10, "1024"},
		{-1, constant.SHIFT_UP, 2, "-4"},
		{1024, constant.SHIFT_DOWN, 10, "1"},
		{1, constant.SHIFT_DOWN, 1, "0"},
		{1, constant.SHIFT_DOWN, 600, "0"},
		{1, constant.SHIFT_DOWN, integer.SHIFT_COUNT_MAXIMUM, "0"},
	} {
		testify.Equal(t, one.Want, text_of(shift_constant(
			whole_constant(one.Value), one.Operation, one.Count)),
			"the shift of %d by %d folds", one.Value, one.Count)
	}
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(shift_constant(
		ratio_constant(1, 2), constant.SHIFT_UP, 1)),
		"a shift of a ratio folds unknown")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(shift_constant(
		view_constant("a"), constant.SHIFT_DOWN, 1)),
		"a shift of a view folds unknown")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(shift_constant(
		whole_constant(1), constant.SHIFT_UP, 600)),
		"a shift past the width folds unknown")
}

func test_bounds(t *testing.T) {
	for _, one := range []constant.Int_64{
		integer.INT_64_MINIMUM, integer.INT_64_MAXIMUM, -1, 0, 1, 2,
	} {
		result := constant.Int_64_Value(whole_constant(one))
		read, ok := result.Value, result.OK
		testify.True(t, bool(ok), "a machine integer reads back")
		testify.Equal(t, one, read, "a machine integer reads its own value")
	}
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(
		literal_constant(constant.Text(widest_literal()))),
		"a literal past the width folds unknown")
	result := constant.Int_64_Value(binary_constant(
		whole_constant(integer.INT_64_MAXIMUM), constant.BINARY_ADD,
		whole_constant(1)))
	over, held := result.Value, result.OK
	testify.False(t, bool(held), "a whole number past a machine integer reads as none")
	testify.Equal(t, constant.Int_64(0), over, "a value that does not fit reads as zero")
	bounds_ratio_cases(t)
	bounds_text_cases(t)
	bounds_form_cases(t)
}

func bounds_text_cases(t *testing.T) {
	widest := constant.Text(widest_text())
	for _, one := range []constant.Text{"", "a", "ab", widest} {
		result := constant.Text_Value(view_constant(one))
		read, ok := result.Value, result.OK
		testify.True(t, bool(ok), "a view of %d bytes reads back", len(one))
		testify.Equal(t, one, read, "a view reads its own bytes")
	}
	for _, one := range []struct {
		Left  constant.Text
		Right constant.Text
		Order integer.Order
	}{
		{"", "", integer.ORDER_SAME},
		{"a", "", integer.ORDER_AFTER},
		{"", "a", integer.ORDER_BEFORE},
		{widest, widest, integer.ORDER_SAME},
	} {
		result := constant.Compare(
			view_constant(one.Left), view_constant(one.Right))
		order, ok := result.Order, result.OK
		testify.True(t, bool(ok), "two views compare")
		testify.Equal(t, one.Order, order, "the order reads as the views state")
	}
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(literal_constant("")),
		"a literal of no bytes folds unknown")
	testify.Equal(t, constant.KIND_UNKNOWN,
		constant.Kind_Of(literal_constant(widest)),
		"a literal that spans the widest source folds unknown")
}

func bounds_ratio_cases(t *testing.T) {
	for _, one := range []struct {
		Numerator   constant.Int_64
		Denominator constant.Int_64
		Want        string
	}{
		{1, 1, "1"},
		{1, -1, "-1"},
		{0, 3, "0"},
		{-1, 3, "-1/3"},
		{2, 3, "2/3"},
		{1, integer.INT_64_MINIMUM, "-1/9223372036854775808"},
		{1, integer.INT_64_MAXIMUM, "1/9223372036854775807"},
		{integer.INT_64_MINIMUM, 3, "-9223372036854775808/3"},
		{integer.INT_64_MAXIMUM, 3, "9223372036854775807/3"},
	} {
		testify.Equal(t, one.Want, text_of(ratio_constant(
			one.Numerator, one.Denominator)), "the ratio %d over %d folds",
			one.Numerator, one.Denominator)
	}
	wide := shift_constant(whole_constant(1), constant.SHIFT_UP, 300)
	left := binary_constant(wide, constant.BINARY_MULTIPLY,
		ratio_constant(1, 3))
	right := binary_constant(ratio_constant(1, 2),
		constant.BINARY_QUOTIENT, wide)
	ok := constant.Compare(left, right).OK
	testify.False(t, bool(ok), "two ratios that cross past the width compare as no order")
}

func bounds_form_cases(t *testing.T) {
	pair := pair_constant(3, 4)
	for _, size := range []int{0, 1, 2} {
		storage := make([]byte, size)
		for _, one := range []constant.Value{
			truth_constant(true), whole_constant(1000), pair,
			view_constant("abc"),
			constant.Make(constant.KIND_UNKNOWN, "", constant.Zero_Pair()),
		} {
			testify.Equal(t, constant.Form_Count(0), constant.Into_Text(storage, one),
				"storage of %d bytes writes nothing", size)
		}
	}
	widest := widest_text()
	storage := make([]byte, constant.FORM_SIZE_MAXIMUM)
	testify.Equal(t, constant.Form_Count(constant.FORM_SIZE_MAXIMUM),
		constant.Into_Text(storage, view_constant(constant.Text(widest))),
		"the widest view fills the widest storage")
	testify.Equal(t, constant.Form_Count(4),
		constant.Into_Text(storage, truth_constant(true)),
		"a truth writes four bytes")
	testify.Equal(t, constant.Form_Count(1),
		constant.Into_Text(storage[:1], whole_constant(7)),
		"one digit writes into one byte")
	testify.Equal(t, constant.Form_Count(2),
		constant.Into_Text(storage[:2], whole_constant(42)),
		"two digits write into two bytes")
}

// Builds a decimal literal that spends more bits than one whole number holds.
func widest_literal() (text string) {
	buffer := make([]byte, integer.DIGIT_COUNT_MAXIMUM+1)
	for index := range buffer {
		buffer[index] = '9'
	}
	return string(buffer)
}

// Builds a string value that spans the widest source a scanner admits.
func widest_text() (text string) {
	buffer := make([]byte, constant.TEXT_SIZE_MAXIMUM)
	for index := range buffer {
		buffer[index] = 'a'
	}
	return string(buffer)
}

func test_allocation(t *testing.T) {
	allocation_fold_checks(t)
	allocation_read_checks(t)
}

func allocation_read_checks(t *testing.T) {
	fixture := allocation_fixture{}
	pair := pair_constant(3, 4)
	view := view_constant("abc")
	var storage [TEST_STORAGE_SIZE]byte
	checks := []struct {
		Name string
		Call func()
	}{
		{Name: "Make_Unknown", Call: func() {
			fixture.Value = constant.Make(
				constant.KIND_UNKNOWN, "", constant.Zero_Pair())
		}},
		{Name: "Make_Boolean", Call: func() {
			fixture.Value = truth_constant(true)
		}},
		{Name: "Make_Text", Call: func() {
			fixture.Value = view_constant("abc")
		}},
		{Name: "Make_Ratio", Call: func() {
			fixture.Value = ratio_constant(22, 7)
		}},
		{Name: "Make_Complex", Call: func() {
			fixture.Value = pair_constant(3, 4)
		}},
		{Name: "Kind_Of", Call: func() {
			fixture.Kind = constant.Kind_Of(pair)
		}},
		{Name: "Boolean_Value", Call: func() {
			result := constant.Boolean_Value(truth_constant(true))
			fixture.Truth, fixture.Ok = result.Value, result.OK
		}},
		{Name: "Text_Value", Call: func() {
			result := constant.Text_Value(view)
			fixture.Text, fixture.Ok = result.Value, result.OK
		}},
		{Name: "Int_64_Value", Call: func() {
			result := constant.Int_64_Value(whole_constant(42))
			fixture.Number, fixture.Ok = result.Value, result.OK
		}},
		{Name: "Into_Text_Truth", Call: func() {
			fixture.Count = constant.Into_Text(
				storage[:], truth_constant(true))
		}},
		{Name: "Into_Text_View", Call: func() {
			fixture.Count = constant.Into_Text(storage[:], view)
		}},
		{Name: "Into_Text_Pair", Call: func() {
			fixture.Count = constant.Into_Text(storage[:], pair)
		}},
	}
	for _, check := range checks {
		t.Run(check.Name, func(t *testing.T) { testify.Zero_Allocation(t, check.Call) })
	}
	testify.True(t, bool(fixture.Ok), "the allocation fixture holds a read")
}

func allocation_fold_checks(t *testing.T) {
	fixture := allocation_fixture{}
	left := ratio_constant(355, 113)
	right := ratio_constant(22, 7)
	pair := pair_constant(3, 4)
	var storage [TEST_STORAGE_SIZE]byte
	checks := []struct {
		Name string
		Call func()
	}{
		{Name: "Make_Int_64", Call: func() {
			fixture.Value = whole_constant(42)
		}},
		{Name: "Make_From_Literal", Call: func() {
			fixture.Value = literal_constant("42")
		}},
		{Name: "Unary_Operation", Call: func() {
			fixture.Value = unary_constant(constant.UNARY_MINUS, left)
		}},
		{Name: "Binary_Operation", Call: func() {
			fixture.Value = binary_constant(left, constant.BINARY_ADD, right)
		}},
		{Name: "Pair_Operation", Call: func() {
			fixture.Value = binary_constant(
				pair, constant.BINARY_MULTIPLY, pair)
		}},
		{Name: "Compare", Call: func() {
			result := constant.Compare(left, right)
			fixture.Order, fixture.Ok = result.Order, result.OK
		}},
		{Name: "Shift", Call: func() {
			fixture.Value = shift_constant(
				whole_constant(1), constant.SHIFT_UP, 8)
		}},
		{Name: "Into_Text", Call: func() {
			fixture.Count = constant.Into_Text(storage[:], left)
		}},
	}
	for _, check := range checks {
		t.Run(check.Name, func(t *testing.T) { testify.Zero_Allocation(t, check.Call) })
	}
	testify.True(t, bool(fixture.Ok), "the allocation fixture holds an order")
}

// Decimal fixtures independently pin each high-limb shift and its inverse.
func Test_Shifts_High_Limbs(t *testing.T) {
	for _, one := range []struct {
		Number constant.Int_64
		Shift  integer.Shift_Count
		Text   string
	}{
		{Number: 1, Shift: 64,
			Text: "18446744073709551616"},
		{Number: 2, Shift: 64,
			Text: "36893488147419103232"},
		{Number: 1, Shift: 128,
			Text: "340282366920938463463374607431768211456"},
		{Number: 2, Shift: 128,
			Text: "680564733841876926926749214863536422912"},
		{Number: 1, Shift: 192,
			Text: "6277101735386680763835789423207666416102355444464034512896"},
		{Number: 2, Shift: 192,
			Text: "12554203470773361527671578846415332832204710888928069025792"},
		{Number: 1, Shift: 256,
			Text: "115792089237316195423570985008687907853269984665640564039457" +
				"584007913129639" +
				"936"},
		{Number: 2, Shift: 256,
			Text: "231584178474632390847141970017375815706539969331281128078915" +
				"168015826259279" +
				"872"},
		{Number: 1, Shift: 320,
			Text: "213598703592091008239502170616955211460270452235665276994704" +
				"160782221972578" +
				"0640550022962086936576"},
		{Number: 2, Shift: 320,
			Text: "427197407184182016479004341233910422920540904471330553989408" +
				"321564443945156" +
				"1281100045924173873152"},
		{Number: 1, Shift: 384,
			Text: "394020061963944792122790401001436138050797392704654466679482" +
				"934042457217714" +
				"97210611414266254884915640806627990306816"},
		{Number: 2, Shift: 384,
			Text: "788040123927889584245580802002872276101594785409308933358965" +
				"868084914435429" +
				"94421222828532509769831281613255980613632"},
		{Number: 1, Shift: 448,
			Text: "726838724295606890549323807888004534353641360687318060281490" +
				"199180639288113" +
				"397923326191050713763565560762521606266177933534601628614656"},
		{Number: 2, Shift: 448,
			Text: "145367744859121378109864761577600906870728272137463612056298" +
				"039836127857622" +
				"6795846652382101427527131121525043212532355867069203257229312"},
	} {
		for _, sign := range []constant.Int_64{1, -1} {
			number := one.Number * sign
			expected := one.Text
			if sign < 0 {
				expected = "-" + expected
			}
			shifted := shift_constant(
				whole_constant(number), constant.SHIFT_UP, one.Shift,
			)
			testify.Equal(t, expected, text_of(shifted))
			restored := constant.Int_64_Value(
				shift_constant(shifted, constant.SHIFT_DOWN, one.Shift),
			)
			testify.Equal(t, number, restored.Value)
			testify.True(t, bool(restored.OK))
		}
	}
}
