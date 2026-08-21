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

// Reads one value back as text, which is how every case states what it expects.
func text_of(value constant.Value) (text string) {
	var storage [TEST_STORAGE_SIZE]byte
	count := constant.Into_Text(storage[:], value)
	return string(storage[:count])
}

func test_values(t *testing.T) {
	unknown := constant.Make_Unknown()
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(unknown),
		"an unknown names no kind")
	testify.Equal(t, "", text_of(unknown), "an unknown writes nothing")
	testify.Equal(t, constant.KIND_UNKNOWN,
		constant.Kind_Of(constant.Unary_Operation(constant.UNARY_MINUS, unknown)),
		"an operation on an unknown stays unknown")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(constant.Binary_Operation(
		unknown, constant.BINARY_ADD, constant.Make_Int_64(1))),
		"one unknown operand keeps the sum unknown")
}

func test_kinds(t *testing.T) {
	testify.Equal(t, constant.KIND_BOOLEAN, constant.Kind_Of(constant.Make_Boolean(true)),
		"a truth names a truth")
	testify.Equal(t, constant.KIND_TEXT, constant.Kind_Of(constant.Make_Text("a")),
		"a view names text")
	testify.Equal(t, constant.KIND_INT, constant.Kind_Of(constant.Make_Int_64(1)),
		"a machine integer names a whole number")
	testify.Equal(t, constant.KIND_INT, constant.Kind_Of(constant.Make_Ratio(4, 2)),
		"a ratio over one names a whole number")
	testify.Equal(t, constant.KIND_FLOAT, constant.Kind_Of(constant.Make_Ratio(1, 2)),
		"a ratio that is no whole one names a number")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(constant.Make_Ratio(1, 0)),
		"a zero denominator names no value")
}

func test_construction(t *testing.T) {
	testify.Equal(t, "42", text_of(constant.Make_Int_64(42)), "a machine integer folds")
	testify.Equal(t, "-42", text_of(constant.Make_Int_64(-42)),
		"a negative machine integer folds")
	testify.Equal(t, "1/2", text_of(constant.Make_Ratio(2, 4)), "a ratio folds in lowest terms")
	testify.Equal(t, "true", text_of(constant.Make_Boolean(true)), "a truth writes its word")
	testify.Equal(t, "false", text_of(constant.Make_Boolean(false)),
		"a falsehood writes its word")
	testify.Equal(t, "text", text_of(constant.Make_Text("text")), "a view writes itself")
	for _, one := range []struct{ Literal, Want string }{
		{"0", "0"}, {"42", "42"}, {"0xff", "255"}, {"0b1010", "10"}, {"1_000", "1000"},
		{"18446744073709551616", "18446744073709551616"},
	} {
		testify.Equal(t, one.Want, text_of(constant.Make_From_Literal(
			constant.Text(one.Literal))), "the literal %q folds", one.Literal)
	}
	testify.Equal(t, constant.KIND_UNKNOWN,
		constant.Kind_Of(constant.Make_From_Literal("abc")),
		"a literal the grammar refuses folds unknown")
}

func test_reading(t *testing.T) {
	result := constant.Boolean_Value(constant.Make_Boolean(true))
	truth, ok := result.Value, result.OK
	testify.True(t, bool(ok), "a truth reads back")
	testify.True(t, bool(truth), "a truth reads its own value")
	wrong := constant.Boolean_Value(constant.Make_Int_64(1)).OK
	testify.False(t, bool(wrong), "a whole number reads as no truth")
	text_result := constant.Text_Value(constant.Make_Text("abc"))
	text, ok := text_result.Value, text_result.OK
	testify.True(t, bool(ok), "a view reads back")
	testify.Equal(t, constant.Text("abc"), text, "a view reads its own bytes")
	absent := constant.Text_Value(constant.Make_Boolean(true)).OK
	testify.False(t, bool(absent), "a truth reads as no view")
	number_result := constant.Int_64_Value(constant.Make_Int_64(-7))
	number, ok := number_result.Value, number_result.OK
	testify.True(t, bool(ok), "a whole number reads back")
	testify.Equal(t, constant.Int_64(-7), number, "a whole number reads its own value")
	fractional := constant.Int_64_Value(constant.Make_Ratio(1, 2)).OK
	testify.False(t, bool(fractional), "a number that is no whole one reads as none")
}

func test_unary_operations(t *testing.T) {
	testify.Equal(t, "42", text_of(constant.Unary_Operation(
		constant.UNARY_PLUS, constant.Make_Int_64(42))), "a plus leaves a number")
	testify.Equal(t, "-42", text_of(constant.Unary_Operation(
		constant.UNARY_MINUS, constant.Make_Int_64(42))), "a minus reverses a number")
	testify.Equal(t, "-1/2", text_of(constant.Unary_Operation(
		constant.UNARY_MINUS, constant.Make_Ratio(1, 2))), "a minus reverses a ratio")
	testify.Equal(t, "-43", text_of(constant.Unary_Operation(
		constant.UNARY_COMPLEMENT, constant.Make_Int_64(42))),
		"a complement flips every bit")
	testify.Equal(t, "false", text_of(constant.Unary_Operation(
		constant.UNARY_NOT, constant.Make_Boolean(true))), "a not reverses a truth")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(constant.Unary_Operation(
		constant.UNARY_NOT, constant.Make_Int_64(1))), "a not of a number is unknown")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(constant.Unary_Operation(
		constant.UNARY_COMPLEMENT, constant.Make_Ratio(1, 2))),
		"a complement of a ratio is unknown")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(constant.Unary_Operation(
		constant.UNARY_PLUS, constant.Make_Boolean(true))), "a plus of a truth is unknown")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(constant.Unary_Operation(
		constant.UNARY_MINUS, constant.Make_Text("a"))), "a minus of a view is unknown")
}

func test_binary_operations(t *testing.T) {
	binary_number_cases(t)
	binary_pair_cases(t)
	binary_bit_cases(t)
	binary_text_cases(t)
}

func binary_number_cases(t *testing.T) {
	half := constant.Make_Ratio(1, 2)
	third := constant.Make_Ratio(1, 3)
	for _, one := range []struct {
		Left      constant.Value
		Operation constant.Binary
		Right     constant.Value
		Want      string
	}{
		{constant.Make_Int_64(6), constant.BINARY_ADD, constant.Make_Int_64(7), "13"},
		{constant.Make_Int_64(6), constant.BINARY_SUBTRACT, constant.Make_Int_64(7), "-1"},
		{constant.Make_Int_64(6), constant.BINARY_MULTIPLY, constant.Make_Int_64(7), "42"},
		{constant.Make_Int_64(43), constant.BINARY_QUOTIENT, constant.Make_Int_64(7), "6"},
		{constant.Make_Int_64(-43), constant.BINARY_QUOTIENT,
			constant.Make_Int_64(7), "-6"},
		{half, constant.BINARY_ADD, third, "5/6"},
		{half, constant.BINARY_QUOTIENT, third, "3/2"},
		{half, constant.BINARY_MULTIPLY, constant.Make_Int_64(4), "2"},
	} {
		testify.Equal(t, one.Want, text_of(constant.Binary_Operation(
			one.Left, one.Operation, one.Right)),
			"the operation %d folds", one.Operation)
	}
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(constant.Binary_Operation(
		constant.Make_Int_64(1), constant.BINARY_QUOTIENT, constant.Make_Int_64(0))),
		"a quotient over zero folds unknown")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(constant.Binary_Operation(
		constant.Make_Boolean(true), constant.BINARY_ADD, constant.Make_Int_64(1))),
		"a sum of a truth and a number folds unknown")
}

func binary_pair_cases(t *testing.T) {
	pair := constant.Make_Complex(constant.Make_Int_64(3), constant.Make_Int_64(4))
	other := constant.Make_Complex(constant.Make_Int_64(1), constant.Make_Int_64(2))
	for _, one := range []struct {
		Operation constant.Binary
		Want      string
	}{
		{constant.BINARY_ADD, "(4 + 6i)"},
		{constant.BINARY_SUBTRACT, "(2 + 2i)"},
		{constant.BINARY_MULTIPLY, "(-5 + 10i)"},
		{constant.BINARY_QUOTIENT, "(11/5 + -2/5i)"},
	} {
		testify.Equal(t, one.Want, text_of(constant.Binary_Operation(
			pair, one.Operation, other)), "the pair operation %d folds", one.Operation)
	}
	testify.Equal(t, "(4 + 4i)", text_of(constant.Binary_Operation(
		pair, constant.BINARY_ADD, constant.Make_Int_64(1))),
		"a number adds to the real part of a pair alone")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(constant.Binary_Operation(
		pair, constant.BINARY_REMAINDER, other)), "a remainder of two pairs folds unknown")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(constant.Binary_Operation(
		pair, constant.BINARY_QUOTIENT, constant.Make_Complex(
			constant.Make_Int_64(0), constant.Make_Int_64(0)))),
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
		testify.Equal(t, one.Want, text_of(constant.Binary_Operation(
			constant.Make_Int_64(one.Left), one.Operation,
			constant.Make_Int_64(one.Right))),
			"the bit operation %d folds", one.Operation)
	}
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(constant.Binary_Operation(
		constant.Make_Ratio(1, 2), constant.BINARY_AND, constant.Make_Int_64(1))),
		"a bit operation on a ratio folds unknown")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(constant.Binary_Operation(
		constant.Make_Int_64(1), constant.BINARY_AND, constant.Make_Ratio(1, 2))),
		"a bit operation over a ratio folds unknown")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(constant.Binary_Operation(
		constant.Make_Int_64(1), constant.BINARY_REMAINDER, constant.Make_Int_64(0))),
		"a remainder over zero folds unknown")
}

func binary_text_cases(t *testing.T) {
	testify.Equal(t, "abc", text_of(constant.Binary_Operation(
		constant.Make_Text(""), constant.BINARY_ADD, constant.Make_Text("abc"))),
		"an empty view joins to the view behind it")
	testify.Equal(t, "abc", text_of(constant.Binary_Operation(
		constant.Make_Text("abc"), constant.BINARY_ADD, constant.Make_Text(""))),
		"a view joins to the empty view behind it")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(constant.Binary_Operation(
		constant.Make_Text("a"), constant.BINARY_ADD, constant.Make_Text("b"))),
		"two views join to a text no source states")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(constant.Binary_Operation(
		constant.Make_Text("a"), constant.BINARY_ADD, constant.Make_Int_64(1))),
		"a view joins to no number")
}

func test_comparison(t *testing.T) {
	for _, one := range []struct {
		Left  constant.Value
		Right constant.Value
		Order integer.Order
	}{
		{constant.Make_Int_64(1), constant.Make_Int_64(1), integer.ORDER_SAME},
		{constant.Make_Int_64(2), constant.Make_Int_64(1), integer.ORDER_AFTER},
		{constant.Make_Int_64(1), constant.Make_Int_64(2), integer.ORDER_BEFORE},
		{constant.Make_Ratio(1, 3), constant.Make_Ratio(1, 2), integer.ORDER_BEFORE},
		{constant.Make_Boolean(false), constant.Make_Boolean(true), integer.ORDER_BEFORE},
		{constant.Make_Boolean(true), constant.Make_Boolean(false), integer.ORDER_AFTER},
		{constant.Make_Boolean(true), constant.Make_Boolean(true), integer.ORDER_SAME},
		{constant.Make_Text("abc"), constant.Make_Text("abc"), integer.ORDER_SAME},
		{constant.Make_Text("abc"), constant.Make_Text("abd"), integer.ORDER_BEFORE},
		{constant.Make_Text("abd"), constant.Make_Text("abc"), integer.ORDER_AFTER},
		{constant.Make_Text("ab"), constant.Make_Text("abc"), integer.ORDER_BEFORE},
		{constant.Make_Text("abc"), constant.Make_Text("ab"), integer.ORDER_AFTER},
	} {
		result := constant.Compare(one.Left, one.Right)
		order, ok := result.Order, result.OK
		testify.True(t, bool(ok), "a pair of one kind compares")
		testify.Equal(t, one.Order, order, "the order reads as the values state")
	}
	compare_pair_cases(t)
}

func compare_pair_cases(t *testing.T) {
	pair := constant.Make_Complex(constant.Make_Int_64(3), constant.Make_Int_64(4))
	same := constant.Make_Complex(constant.Make_Int_64(3), constant.Make_Int_64(4))
	wider := constant.Make_Complex(constant.Make_Int_64(3), constant.Make_Int_64(5))
	result := constant.Compare(pair, same)
	order, ok := result.Order, result.OK
	testify.True(t, bool(ok), "two pairs compare")
	testify.Equal(t, integer.ORDER_SAME, order, "two equal pairs read the same order")
	result = constant.Compare(pair, wider)
	order, ok = result.Order, result.OK
	testify.True(t, bool(ok), "two pairs of one real part compare")
	testify.Equal(t, integer.ORDER_BEFORE, order, "a smaller imaginary part orders first")
	mixed := constant.Compare(constant.Make_Int_64(1), constant.Make_Text("a")).OK
	testify.False(t, bool(mixed), "two kinds compare as no order")
	unknown := constant.Compare(constant.Make_Unknown(), constant.Make_Unknown()).OK
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
		testify.Equal(t, one.Want, text_of(constant.Shift(
			constant.Make_Int_64(one.Value), one.Operation, one.Count)),
			"the shift of %d by %d folds", one.Value, one.Count)
	}
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(constant.Shift(
		constant.Make_Ratio(1, 2), constant.SHIFT_UP, 1)),
		"a shift of a ratio folds unknown")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(constant.Shift(
		constant.Make_Text("a"), constant.SHIFT_DOWN, 1)),
		"a shift of a view folds unknown")
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(constant.Shift(
		constant.Make_Int_64(1), constant.SHIFT_UP, 600)),
		"a shift past the width folds unknown")
}

func test_bounds(t *testing.T) {
	for _, one := range []constant.Int_64{
		integer.INT_64_MINIMUM, integer.INT_64_MAXIMUM, -1, 0, 1, 2,
	} {
		result := constant.Int_64_Value(constant.Make_Int_64(one))
		read, ok := result.Value, result.OK
		testify.True(t, bool(ok), "a machine integer reads back")
		testify.Equal(t, one, read, "a machine integer reads its own value")
	}
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(
		constant.Make_From_Literal(constant.Text(widest_literal()))),
		"a literal past the width folds unknown")
	result := constant.Int_64_Value(constant.Binary_Operation(
		constant.Make_Int_64(integer.INT_64_MAXIMUM), constant.BINARY_ADD,
		constant.Make_Int_64(1)))
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
		result := constant.Text_Value(constant.Make_Text(one))
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
			constant.Make_Text(one.Left), constant.Make_Text(one.Right))
		order, ok := result.Order, result.OK
		testify.True(t, bool(ok), "two views compare")
		testify.Equal(t, one.Order, order, "the order reads as the views state")
	}
	testify.Equal(t, constant.KIND_UNKNOWN, constant.Kind_Of(constant.Make_From_Literal("")),
		"a literal of no bytes folds unknown")
	testify.Equal(t, constant.KIND_UNKNOWN,
		constant.Kind_Of(constant.Make_From_Literal(widest)),
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
		testify.Equal(t, one.Want, text_of(constant.Make_Ratio(
			one.Numerator, one.Denominator)), "the ratio %d over %d folds",
			one.Numerator, one.Denominator)
	}
	wide := constant.Shift(constant.Make_Int_64(1), constant.SHIFT_UP, 300)
	left := constant.Binary_Operation(wide, constant.BINARY_MULTIPLY,
		constant.Make_Ratio(1, 3))
	right := constant.Binary_Operation(constant.Make_Ratio(1, 2),
		constant.BINARY_QUOTIENT, wide)
	ok := constant.Compare(left, right).OK
	testify.False(t, bool(ok), "two ratios that cross past the width compare as no order")
}

func bounds_form_cases(t *testing.T) {
	pair := constant.Make_Complex(constant.Make_Int_64(3), constant.Make_Int_64(4))
	for _, size := range []int{0, 1, 2} {
		storage := make([]byte, size)
		for _, one := range []constant.Value{
			constant.Make_Boolean(true), constant.Make_Int_64(1000), pair,
			constant.Make_Text("abc"), constant.Make_Unknown(),
		} {
			testify.Equal(t, constant.Form_Count(0), constant.Into_Text(storage, one),
				"storage of %d bytes writes nothing", size)
		}
	}
	widest := widest_text()
	storage := make([]byte, constant.FORM_SIZE_MAXIMUM)
	testify.Equal(t, constant.Form_Count(constant.FORM_SIZE_MAXIMUM),
		constant.Into_Text(storage, constant.Make_Text(constant.Text(widest))),
		"the widest view fills the widest storage")
	testify.Equal(t, constant.Form_Count(4),
		constant.Into_Text(storage, constant.Make_Boolean(true)),
		"a truth writes four bytes")
	testify.Equal(t, constant.Form_Count(1),
		constant.Into_Text(storage[:1], constant.Make_Int_64(7)),
		"one digit writes into one byte")
	testify.Equal(t, constant.Form_Count(2),
		constant.Into_Text(storage[:2], constant.Make_Int_64(42)),
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
	pair := constant.Make_Complex(constant.Make_Int_64(3), constant.Make_Int_64(4))
	view := constant.Make_Text("abc")
	var storage [TEST_STORAGE_SIZE]byte
	checks := []struct {
		Name string
		Call func()
	}{
		{Name: "Make_Unknown", Call: func() {
			fixture.Value = constant.Make_Unknown()
		}},
		{Name: "Make_Boolean", Call: func() {
			fixture.Value = constant.Make_Boolean(true)
		}},
		{Name: "Make_Text", Call: func() {
			fixture.Value = constant.Make_Text("abc")
		}},
		{Name: "Make_Ratio", Call: func() {
			fixture.Value = constant.Make_Ratio(22, 7)
		}},
		{Name: "Make_Complex", Call: func() {
			fixture.Value = constant.Make_Complex(
				constant.Make_Int_64(3), constant.Make_Int_64(4))
		}},
		{Name: "Kind_Of", Call: func() {
			fixture.Kind = constant.Kind_Of(pair)
		}},
		{Name: "Boolean_Value", Call: func() {
			result := constant.Boolean_Value(constant.Make_Boolean(true))
			fixture.Truth, fixture.Ok = result.Value, result.OK
		}},
		{Name: "Text_Value", Call: func() {
			result := constant.Text_Value(view)
			fixture.Text, fixture.Ok = result.Value, result.OK
		}},
		{Name: "Int_64_Value", Call: func() {
			result := constant.Int_64_Value(constant.Make_Int_64(42))
			fixture.Number, fixture.Ok = result.Value, result.OK
		}},
		{Name: "Into_Text_Truth", Call: func() {
			fixture.Count = constant.Into_Text(
				storage[:], constant.Make_Boolean(true))
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
	left := constant.Make_Ratio(355, 113)
	right := constant.Make_Ratio(22, 7)
	pair := constant.Make_Complex(constant.Make_Int_64(3), constant.Make_Int_64(4))
	var storage [TEST_STORAGE_SIZE]byte
	checks := []struct {
		Name string
		Call func()
	}{
		{Name: "Make_Int_64", Call: func() {
			fixture.Value = constant.Make_Int_64(42)
		}},
		{Name: "Make_From_Literal", Call: func() {
			fixture.Value = constant.Make_From_Literal("42")
		}},
		{Name: "Unary_Operation", Call: func() {
			fixture.Value = constant.Unary_Operation(constant.UNARY_MINUS, left)
		}},
		{Name: "Binary_Operation", Call: func() {
			fixture.Value = constant.Binary_Operation(left, constant.BINARY_ADD, right)
		}},
		{Name: "Pair_Operation", Call: func() {
			fixture.Value = constant.Binary_Operation(
				pair, constant.BINARY_MULTIPLY, pair)
		}},
		{Name: "Compare", Call: func() {
			result := constant.Compare(left, right)
			fixture.Order, fixture.Ok = result.Order, result.OK
		}},
		{Name: "Shift", Call: func() {
			fixture.Value = constant.Shift(
				constant.Make_Int_64(1), constant.SHIFT_UP, 8)
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
