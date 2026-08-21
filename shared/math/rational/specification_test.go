package rational_test

import (
	"testing"

	"local/james-orcales/shared/math/integer"
	"local/james-orcales/shared/math/rational"
	"local/james-orcales/shared/testify"
)

// Test_Representation binds the Representation leaf before fixture declarations.
func Test_Representation(t *testing.T) {
	test_representation(t)
}

// Test_Normalisation binds the Normalisation leaf before fixture declarations.
func Test_Normalisation(t *testing.T) {
	test_normalisation(t)
}

// Test_Conversion binds the Conversion specification leaf before fixture declarations.
func Test_Conversion(t *testing.T) {
	test_conversion(t)
}

// Test_Arithmetic binds the Arithmetic specification leaf before fixture declarations.
func Test_Arithmetic(t *testing.T) {
	test_arithmetic(t)
}

// Test_Comparison binds the Comparison specification leaf before fixture declarations.
func Test_Comparison(t *testing.T) {
	test_comparison(t)
}

// Test_Text binds the Text specification leaf before fixture declarations.
func Test_Text(t *testing.T) {
	test_text(t)
}

// Test_Bounds binds the Bounds specification leaf before fixture declarations.
func Test_Bounds(t *testing.T) {
	test_bounds(t)
}

// Test_Allocation binds the Allocation specification leaf before fixture declarations.
func Test_Allocation(t *testing.T) {
	test_allocation(t)
}

const TEST_STORAGE_SIZE = rational.TEXT_SIZE_MAXIMUM

type allocation_fixture struct {
	Value rational.Rational
	Ok    rational.Boolean
	Order integer.Order
	Count rational.Text_Count
}

// Caller-owned wrappers keep behavior assertions compact while allocation checks call production
// mutation directly.
func integer_from_text(source integer.Text) (value integer.Integer, ok integer.Boolean) {
	ok = integer.From_Text(&value, source)
	return value, ok
}

func integer_negate(source integer.Integer) (value integer.Integer, ok integer.Boolean) {
	ok = integer.Negate(&value, source)
	return value, ok
}

func integer_one() (value integer.Integer) {
	integer.One(&value)
	return value
}

func integer_zero() (value integer.Integer) {
	integer.Zero(&value)
	return value
}

func integer_shift_left(source integer.Integer, count integer.Shift_Count) (
	value integer.Integer,
	ok integer.Boolean,
) {
	ok = integer.Shift_Left(&value, source, count)
	return value, ok
}

func integer_not(source integer.Integer) (value integer.Integer) {
	integer.Not(&value, source)
	return value
}

func zero_ratio() (value rational.Rational) {
	rational.Zero(&value)
	return value
}

func unit_ratio() (value rational.Rational) {
	rational.One(&value)
	return value
}

func from_integer(source rational.Numerator) (value rational.Rational) {
	rational.From_Integer(&value, source)
	return value
}

func from_ratio(above rational.Numerator, below rational.Denominator) (
	value rational.Rational,
	ok rational.Boolean,
) {
	ok = rational.From_Ratio(&value, above, below)
	return value, ok
}

func whole(source rational.Rational) (value rational.Numerator, ok rational.Boolean) {
	ok = rational.Whole(&value, source)
	return value, ok
}

func add(left rational.Rational, right rational.Rational) (
	value rational.Rational,
	ok rational.Boolean,
) {
	ok = rational.Add(&value, left, right)
	return value, ok
}

func subtract(left rational.Rational, right rational.Rational) (
	value rational.Rational,
	ok rational.Boolean,
) {
	ok = rational.Subtract(&value, left, right)
	return value, ok
}

func multiply(left rational.Rational, right rational.Rational) (
	value rational.Rational,
	ok rational.Boolean,
) {
	ok = rational.Multiply(&value, left, right)
	return value, ok
}

func divide(left rational.Rational, right rational.Rational) (
	value rational.Rational,
	ok rational.Boolean,
) {
	ok = rational.Divide(&value, left, right)
	return value, ok
}

func negate(source rational.Rational) (value rational.Rational, ok rational.Boolean) {
	ok = rational.Negate(&value, source)
	return value, ok
}

func absolute(source rational.Rational) (value rational.Rational, ok rational.Boolean) {
	ok = rational.Absolute(&value, source)
	return value, ok
}

// Reads one ratio back as text, which is how every case states what it expects.
func text_of(value rational.Rational) (text string) {
	var storage [TEST_STORAGE_SIZE]byte
	count := rational.Into_Text(storage[:], value)
	return string(storage[:count])
}

// Reads one whole value from a decimal literal, refusing to hide a bad fixture.
func whole_of(t *testing.T, text string) (value integer.Integer) {
	body := text
	negative := false
	if len(body) > 0 {
		if body[0] == '-' {
			negative = true
			body = body[1:]
		}
	}
	value, ok := integer_from_text(integer.Text(body))
	testify.True(t, bool(ok), "the fixture %q reads as a value", text)
	if !negative {
		return value
	}
	value, ok = integer_negate(value)
	testify.True(t, bool(ok), "the fixture %q negates", text)
	return value
}

// Builds one ratio from two decimal literals, refusing to hide a bad fixture.
func ratio_of(t *testing.T, above string, below string) (value rational.Rational) {
	value, ok := from_ratio(
		rational.Numerator(whole_of(t, above).Limbs),
		rational.Denominator(whole_of(t, below).Limbs))
	testify.True(t, bool(ok), "the fixture %s over %s reads as a ratio", above, below)
	return value
}

func test_representation(t *testing.T) {
	testify.Equal(t, "0", text_of(zero_ratio()), "zero reads as zero")
	testify.Equal(t, "1", text_of(unit_ratio()), "one reads as one")
	testify.True(t, bool(rational.Is_Zero(zero_ratio())), "zero names zero")
	testify.False(t, bool(rational.Is_Zero(unit_ratio())), "one names no zero")
	testify.True(t, bool(rational.Is_Whole(unit_ratio())), "one names a whole value")
	testify.False(t, bool(rational.Is_Whole(ratio_of(t, "1", "2"))),
		"a half names no whole value")
	testify.Equal(t, integer.ORDER_SAME, rational.Sign(zero_ratio()), "zero has no sign")
	testify.Equal(t, integer.ORDER_AFTER, rational.Sign(unit_ratio()), "one signs above")
	testify.Equal(t, integer.ORDER_BEFORE, rational.Sign(ratio_of(t, "-1", "2")),
		"a negative half signs below")
}

func test_normalisation(t *testing.T) {
	testify.Equal(t, "1/2", text_of(ratio_of(t, "2", "4")), "two over four is one over two")
	testify.Equal(t, "3", text_of(ratio_of(t, "12", "4")), "twelve over four is three")
	testify.Equal(t, "-1/2", text_of(ratio_of(t, "1", "-2")),
		"a negative denominator moves its sign above the line")
	testify.Equal(t, "-1/2", text_of(ratio_of(t, "-1", "2")),
		"a negative numerator keeps its sign")
	testify.Equal(t, "1/2", text_of(ratio_of(t, "-1", "-2")),
		"two negatives leave a positive ratio")
	testify.Equal(t, "0", text_of(ratio_of(t, "0", "5")), "zero over anything is zero")
	_, ok := from_ratio(
		rational.Numerator(integer_one().Limbs),
		rational.Denominator(integer_zero().Limbs))
	testify.False(t, bool(ok), "a zero denominator names no ratio")
}

func test_conversion(t *testing.T) {
	lifted := from_integer(rational.Numerator(whole_of(t, "42").Limbs))
	testify.Equal(t, "42", text_of(lifted), "a lifted whole value reads itself")
	testify.True(t, bool(rational.Is_Whole(lifted)), "a lifted value is whole")
	for _, one := range []struct{ Above, Below, Want string }{
		{"7", "2", "3"}, {"-7", "2", "-3"}, {"7", "-2", "-3"}, {"1", "2", "0"},
		{"8", "2", "4"},
	} {
		result, ok := whole(ratio_of(t, one.Above, one.Below))
		testify.True(t, bool(ok), "%s over %s truncates", one.Above, one.Below)
		testify.Equal(t, one.Want,
			whole_text(integer.Integer{Limbs: integer.Limbs(result)}),
			"%s over %s truncates toward zero", one.Above, one.Below)
	}
}

// Reads one whole value back as text.
func whole_text(value integer.Integer) (text string) {
	var storage [TEST_STORAGE_SIZE]byte
	count := integer.Into_Text(storage[:], value)
	return string(storage[:count])
}

func test_arithmetic(t *testing.T) {
	for _, one := range []struct {
		Left, Right, Sum, Difference, Product, Quotient string
	}{
		{"1/2", "1/3", "5/6", "1/6", "1/6", "3/2"},
		{"1/2", "1/2", "1", "0", "1/4", "1"},
		{"-1/2", "1/3", "-1/6", "-5/6", "-1/6", "-3/2"},
		{"3/1", "2/1", "5", "1", "6", "3/2"},
	} {
		left := parse_ratio(t, one.Left)
		right := parse_ratio(t, one.Right)
		sum, ok := add(left, right)
		testify.True(t, bool(ok), "%s and %s sum", one.Left, one.Right)
		testify.Equal(t, one.Sum, text_of(sum), "%s and %s sum to", one.Left, one.Right)
		difference, ok := subtract(left, right)
		testify.True(t, bool(ok), "%s less %s holds", one.Left, one.Right)
		testify.Equal(t, one.Difference, text_of(difference),
			"%s less %s is", one.Left, one.Right)
		product, ok := multiply(left, right)
		testify.True(t, bool(ok), "%s by %s holds", one.Left, one.Right)
		testify.Equal(t, one.Product, text_of(product), "%s by %s is", one.Left, one.Right)
		quotient, ok := divide(left, right)
		testify.True(t, bool(ok), "%s over %s holds", one.Left, one.Right)
		testify.Equal(t, one.Quotient, text_of(quotient),
			"%s over %s is", one.Left, one.Right)
	}
	_, ok := divide(unit_ratio(), zero_ratio())
	testify.False(t, bool(ok), "a zero divisor names no quotient")
	negated, ok := negate(parse_ratio(t, "2/3"))
	testify.True(t, bool(ok), "a small ratio negates")
	testify.Equal(t, "-2/3", text_of(negated), "two thirds negates")
	magnitude, ok := absolute(parse_ratio(t, "-2/3"))
	testify.True(t, bool(ok), "a small ratio reads its magnitude")
	testify.Equal(t, "2/3", text_of(magnitude), "minus two thirds reads two thirds")
	magnitude, ok = absolute(parse_ratio(t, "2/3"))
	testify.True(t, bool(ok), "a positive ratio reads its magnitude")
	testify.Equal(t, "2/3", text_of(magnitude), "a positive magnitude is itself")
}

// Reads a fixture written as a numerator, a slash, and a denominator.
func parse_ratio(t *testing.T, text string) (value rational.Rational) {
	for index := range len(text) {
		if text[index] != '/' {
			continue
		}
		return ratio_of(t, text[:index], text[index+1:])
	}
	return ratio_of(t, text, "1")
}

func test_comparison(t *testing.T) {
	for _, one := range []struct {
		Left, Right string
		Order       integer.Order
	}{
		{"1/2", "1/3", integer.ORDER_AFTER},
		{"1/3", "1/2", integer.ORDER_BEFORE},
		{"2/4", "1/2", integer.ORDER_SAME},
		{"-1/2", "1/2", integer.ORDER_BEFORE},
		{"0", "0", integer.ORDER_SAME},
	} {
		order, ok := rational.Compare(parse_ratio(t, one.Left), parse_ratio(t, one.Right))
		testify.True(t, bool(ok), "%s stands against %s", one.Left, one.Right)
		testify.Equal(t, one.Order, order, "%s against %s", one.Left, one.Right)
	}
}

func test_text(t *testing.T) {
	testify.Equal(t, "5/6", text_of(parse_ratio(t, "5/6")), "a ratio writes its two halves")
	testify.Equal(t, "-5/6", text_of(parse_ratio(t, "-5/6")), "a ratio writes its sign")
	testify.Equal(t, "7", text_of(parse_ratio(t, "7")), "a whole ratio writes one half")
	var narrow [TEST_NARROW_SIZE]byte
	testify.Equal(t, rational.Text_Count(0),
		rational.Into_Text(narrow[:], parse_ratio(t, "123/456")),
		"a destination that cannot hold the form writes nothing")
	var storage [TEST_STORAGE_SIZE]byte
	testify.Equal(t, rational.Text_Count(1),
		rational.Into_Text(storage[:], zero_ratio()), "zero writes one byte")
	testify.Equal(t, rational.Text_Count(0),
		rational.Into_Text(storage[:0], unit_ratio()), "no storage writes nothing")
	testify.Equal(t, rational.Text_Count(1),
		rational.Into_Text(storage[:1], unit_ratio()),
		"one byte of storage holds one digit")
	testify.Equal(t, rational.Text_Count(rational.TEXT_SIZE_MAXIMUM),
		rational.Into_Text(storage[:], widest_ratio(t)),
		"the longest form spells both halves and the slash")
}

const TEST_NARROW_SIZE = 2

// Reads the value whose magnitude leaves the width, which no normalised ratio holds.
func no_magnitude(t *testing.T) (value integer.Integer) {
	value, ok := integer_shift_left(integer_one(), integer.BIT_COUNT_MAXIMUM-1)
	testify.False(t, bool(ok), "the sign bit alone leaves the positive range")
	return value
}

// Builds the ratio whose written form spans the whole width.
func widest_ratio(t *testing.T) (value rational.Rational) {
	largest := integer_not(no_magnitude(t))
	lowest, ok := integer_negate(largest)
	testify.True(t, bool(ok), "the largest magnitude negates")
	below, below_ok := integer_shift_left(integer_one(), integer.BIT_COUNT_MAXIMUM-2)
	testify.True(t, bool(below_ok), "the widest denominator fits")
	value, ratio_ok := from_ratio(
		rational.Numerator(lowest.Limbs), rational.Denominator(below.Limbs))
	testify.True(t, bool(ratio_ok), "the widest halves form a ratio")
	return value
}

func test_bounds(t *testing.T) {
	widest, ok := integer_shift_left(integer_one(), integer.BIT_COUNT_MAXIMUM-2)
	testify.True(t, bool(ok), "the widest whole value fits")
	wide := from_integer(rational.Numerator(widest.Limbs))
	_, over := add(wide, wide)
	testify.False(t, bool(over), "a sum past the width says so")
	_, product := multiply(wide, wide)
	testify.False(t, bool(product), "a product past the width says so")
	_, quotient := divide(wide, ratio_of(t, "1", "3"))
	testify.False(t, bool(quotient), "a quotient past the width says so")
	_, order := rational.Compare(wide, ratio_of(t, "1", "3"))
	testify.False(t, bool(order), "a comparison past the width says so")
	// A ratio lifted whole skips the normalisation that would have refused it, thus the value
	// with no magnitude reaches the steps that owe a refusal.
	edge := from_integer(rational.Numerator(no_magnitude(t).Limbs))
	_, flip := negate(edge)
	testify.False(t, bool(flip), "the value with no magnitude does not negate")
	_, magnitude := absolute(edge)
	testify.False(t, bool(magnitude), "the value with no magnitude has none")
	_, truncated := whole(edge)
	testify.False(t, bool(truncated), "the value with no magnitude does not truncate")
	lowest, flipped_ok := negate(wide)
	testify.True(t, bool(flipped_ok), "the widest whole value negates")
	_, difference := subtract(lowest, wide)
	testify.False(t, bool(difference), "a difference past the width says so")
}

func test_allocation(t *testing.T) {
	fixture := allocation_fixture{}
	left := ratio_of(t, "355", "113")
	right := ratio_of(t, "22", "7")
	var storage [TEST_STORAGE_SIZE]byte
	checks := []struct {
		Name string
		Call func()
	}{
		{Name: "Add", Call: func() {
			fixture.Ok = rational.Add(&fixture.Value, left, right)
		}},
		{Name: "Subtract", Call: func() {
			fixture.Ok = rational.Subtract(&fixture.Value, left, right)
		}},
		{Name: "Multiply", Call: func() {
			fixture.Ok = rational.Multiply(&fixture.Value, left, right)
		}},
		{Name: "Divide", Call: func() {
			fixture.Ok = rational.Divide(&fixture.Value, left, right)
		}},
		{Name: "Compare", Call: func() {
			fixture.Order, fixture.Ok = rational.Compare(left, right)
		}},
		{Name: "Into_Text", Call: func() {
			fixture.Count = rational.Into_Text(storage[:], left)
		}},
	}
	for _, check := range checks {
		t.Run(check.Name, func(t *testing.T) { testify.Zero_Allocation(t, check.Call) })
	}
	testify.True(t, bool(fixture.Ok), "the allocation fixture holds a ratio")
}
