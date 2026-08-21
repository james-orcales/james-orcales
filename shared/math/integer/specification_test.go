package integer_test

import (
	"testing"

	"local/james-orcales/shared/math/integer"
	"local/james-orcales/shared/testify"
)

// Test_Representation binds the Representation leaf before fixture declarations.
func Test_Representation(t *testing.T) {
	test_representation(t)
}

// Test_Conversion binds the Conversion specification leaf before fixture declarations.
func Test_Conversion(t *testing.T) {
	test_conversion(t)
}

// Test_Addition binds the Addition specification leaf before fixture declarations.
func Test_Addition(t *testing.T) {
	test_addition(t)
}

// Test_Multiplication binds the Multiplication leaf before fixture declarations.
func Test_Multiplication(t *testing.T) {
	test_multiplication(t)
}

// Test_Division binds the Division specification leaf before fixture declarations.
func Test_Division(t *testing.T) {
	test_division(t)
}

// Test_Comparison binds the Comparison specification leaf before fixture declarations.
func Test_Comparison(t *testing.T) {
	test_comparison(t)
}

// Test_Shifts binds the Shifts specification leaf before fixture declarations.
func Test_Shifts(t *testing.T) {
	test_shifts(t)
}

// Test_Bitwise binds the Bitwise specification leaf before fixture declarations.
func Test_Bitwise(t *testing.T) {
	test_bitwise(t)
}

// Test_Common_Divisor binds the Common Divisor leaf before fixture declarations.
func Test_Common_Divisor(t *testing.T) {
	test_common_divisor(t)
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

const TEST_DIGITS_SIZE = 256

const TEST_NARROW_SIZE = 2

type allocation_fixture struct {
	Value integer.Integer
	Ok    integer.Boolean
	Order integer.Order
	Count integer.Digit_Count
}

// Reads one value back as decimal text, which is how every case states what it expects.
func text_of(value integer.Integer) (text string) {
	var storage [TEST_DIGITS_SIZE]byte
	count := integer.Into_Text(storage[:], value)
	return string(storage[:count])
}

// Reads one literal into a value. A Go literal carries no sign, thus a leading minus is the
// unary operator a caller would apply and the fixture applies it here.
func value_of(t *testing.T, text string) (value integer.Integer) {
	body := text
	negative := false
	if len(body) > 0 {
		if body[0] == '-' {
			negative = true
			body = body[1:]
		}
	}
	value, ok := integer.From_Text(integer.Text(body))
	testify.True(t, bool(ok), "the fixture %q reads as a value", text)
	if !negative {
		return value
	}
	value, ok = integer.Negate(value)
	testify.True(t, bool(ok), "the fixture %q negates", text)
	return value
}

func test_representation(t *testing.T) {
	testify.Equal(t, "0", text_of(integer.Zero()), "zero reads as zero")
	testify.Equal(t, "1", text_of(integer.One()), "one reads as one")
	testify.True(t, bool(integer.Is_Zero(integer.Zero())), "zero holds no bit")
	testify.False(t, bool(integer.Is_Zero(integer.One())), "one holds a bit")
	testify.False(t, bool(integer.Is_Negative(integer.One())), "one stands above zero")
	testify.True(t, bool(integer.Is_Negative(integer.From_Int_64(-1))),
		"a lifted negative stands below zero")
	testify.Equal(t, integer.ORDER_SAME, integer.Sign(integer.Zero()), "zero has no sign")
	testify.Equal(t, integer.ORDER_AFTER, integer.Sign(integer.One()), "one signs above")
	testify.Equal(t, integer.ORDER_BEFORE, integer.Sign(integer.From_Int_64(-5)),
		"a negative signs below")
}

func test_conversion(t *testing.T) {
	for _, one := range []int64{0, 1, -1, 2, -2, 1 << 62, -1 << 62,
		integer.INT_64_MAXIMUM, integer.INT_64_MINIMUM} {
		value := integer.From_Int_64(integer.Int_64(one))
		back, ok := integer.To_Int_64(value)
		testify.True(t, bool(ok), "the machine integer %d fits", one)
		testify.Equal(t, one, int64(back), "the machine integer %d survives", one)
	}
	wide, _ := integer.Shift_Left(integer.One(), 100)
	_, ok := integer.To_Int_64(wide)
	testify.False(t, bool(ok), "a value past the machine width does not fit")
}

func test_addition(t *testing.T) {
	sum, ok := integer.Add(integer.From_Int_64(2), integer.From_Int_64(3))
	testify.True(t, bool(ok), "a small sum fits")
	testify.Equal(t, "5", text_of(sum), "two and three sum to five")
	difference, ok := integer.Subtract(integer.From_Int_64(2), integer.From_Int_64(5))
	testify.True(t, bool(ok), "a small difference fits")
	testify.Equal(t, "-3", text_of(difference), "two less five is minus three")
	negated, ok := integer.Negate(integer.From_Int_64(7))
	testify.True(t, bool(ok), "a small negation fits")
	testify.Equal(t, "-7", text_of(negated), "seven negates to minus seven")
	absolute, ok := integer.Absolute(integer.From_Int_64(-7))
	testify.True(t, bool(ok), "a small magnitude fits")
	testify.Equal(t, "7", text_of(absolute), "minus seven reads seven")
	high := value_of(t, "0x7fffffffffffffffffffffffffffffff"+
		"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"+
		"ffffffffffffffffffffffffffffffff")
	_, over := integer.Add(high, integer.One())
	testify.False(t, bool(over), "a sum past the width says so")
	_, under := integer.Subtract(text_minimum(t), integer.One())
	testify.False(t, bool(under), "a difference past the width says so")
	_, flip := integer.Negate(text_minimum(t))
	testify.False(t, bool(flip), "the most negative value has no positive twin")
	_, magnitude := integer.Absolute(text_minimum(t))
	testify.False(t, bool(magnitude), "the most negative value has no magnitude")
}

// Reads the most negative value the width holds.
func text_minimum(t *testing.T) (value integer.Integer) {
	shifted, ok := integer.Shift_Left(integer.One(), integer.BIT_COUNT_MAXIMUM-1)
	testify.False(t, bool(ok), "the sign bit alone leaves the positive range")
	return shifted
}

func test_multiplication(t *testing.T) {
	product, ok := integer.Multiply(integer.From_Int_64(6), integer.From_Int_64(7))
	testify.True(t, bool(ok), "a small product fits")
	testify.Equal(t, "42", text_of(product), "six by seven is forty two")
	product, ok = integer.Multiply(integer.From_Int_64(-6), integer.From_Int_64(7))
	testify.True(t, bool(ok), "a signed product fits")
	testify.Equal(t, "-42", text_of(product), "a lone negative signs the product")
	product, ok = integer.Multiply(integer.From_Int_64(-6), integer.From_Int_64(-7))
	testify.True(t, bool(ok), "two negatives fit")
	testify.Equal(t, "42", text_of(product), "two negatives sign the product above")
	product, ok = integer.Multiply(integer.From_Int_64(0), integer.From_Int_64(-7))
	testify.True(t, bool(ok), "a zero product fits")
	testify.Equal(t, "0", text_of(product), "zero by anything is zero")
	wide := value_of(t, "18446744073709551616")
	product, ok = integer.Multiply(wide, wide)
	testify.True(t, bool(ok), "a product past one limb fits the width")
	testify.Equal(t, "340282366920938463463374607431768211456", text_of(product),
		"two to the sixty four squares to two to the one hundred twenty eight")
	half, _ := integer.Shift_Left(integer.One(), integer.BIT_COUNT_MAXIMUM-2)
	_, over := integer.Multiply(half, integer.From_Int_64(4))
	testify.False(t, bool(over), "a product past the width says so")
}

func test_division(t *testing.T) {
	for _, one := range []struct {
		Dividend, Divisor, Quotient, Remainder string
	}{
		{"42", "7", "6", "0"},
		{"43", "7", "6", "1"},
		{"-43", "7", "-6", "-1"},
		{"43", "-7", "-6", "1"},
		{"-43", "-7", "6", "-1"},
		{"5", "7", "0", "5"},
		{"0", "7", "0", "0"},
		{"340282366920938463463374607431768211456", "18446744073709551616",
			"18446744073709551616", "0"},
	} {
		quotient, remainder, ok := integer.Divide(
			value_of(t, one.Dividend), value_of(t, one.Divisor))
		testify.True(t, bool(ok), "%s over %s divides", one.Dividend, one.Divisor)
		testify.Equal(t, one.Quotient, text_of(quotient),
			"%s over %s quotients", one.Dividend, one.Divisor)
		testify.Equal(t, one.Remainder, text_of(remainder),
			"%s over %s remains", one.Dividend, one.Divisor)
	}
	_, _, ok := integer.Divide(integer.One(), integer.Zero())
	testify.False(t, bool(ok), "a zero divisor is refused")
}

func test_comparison(t *testing.T) {
	for _, one := range []struct {
		Left, Right string
		Order       integer.Order
	}{
		{"0", "0", integer.ORDER_SAME},
		{"1", "0", integer.ORDER_AFTER},
		{"0", "1", integer.ORDER_BEFORE},
		{"-1", "1", integer.ORDER_BEFORE},
		{"1", "-1", integer.ORDER_AFTER},
		{"-2", "-1", integer.ORDER_BEFORE},
		{"18446744073709551616", "18446744073709551615", integer.ORDER_AFTER},
		{"18446744073709551616", "18446744073709551616", integer.ORDER_SAME},
	} {
		testify.Equal(t, one.Order,
			integer.Compare(value_of(t, one.Left), value_of(t, one.Right)),
			"%s stands against %s", one.Left, one.Right)
	}
}

func test_shifts(t *testing.T) {
	still, ok := integer.Shift_Left(integer.One(), 0)
	testify.True(t, bool(ok), "a shift of nothing fits")
	testify.Equal(t, "1", text_of(still), "a shift of nothing moves nothing")
	testify.Equal(t, "2", text_of(must_shift(t, integer.One(), 1)), "one shifted once is two")
	testify.Equal(t, "4", text_of(must_shift(t, integer.One(), 2)), "one shifted twice is four")
	testify.Equal(t, "4", text_of(integer.Shift_Right(value_of(t, "16"), 2)),
		"sixteen shifted down twice is four")
	testify.Equal(t, "16", text_of(integer.Shift_Right(value_of(t, "16"), 0)),
		"a shift down of nothing moves nothing")
	shifted, ok := integer.Shift_Left(integer.One(), 10)
	testify.True(t, bool(ok), "a small shift fits")
	testify.Equal(t, "1024", text_of(shifted), "one shifted ten is a thousand and twenty four")
	shifted, ok = integer.Shift_Left(integer.One(), 200)
	testify.True(t, bool(ok), "a wide shift fits the width")
	testify.Equal(t, "1606938044258990275541962092341162602522202993782792835301376",
		text_of(shifted), "one shifted two hundred is exact")
	testify.Equal(t, "1", text_of(integer.Shift_Right(shifted, 200)),
		"a shift back reads the value again")
	testify.Equal(t, "0", text_of(integer.Shift_Right(integer.One(), 1)),
		"one shifted down is zero")
	testify.Equal(t, "-1", text_of(integer.Shift_Right(integer.From_Int_64(-1), 10)),
		"a negative shift carries the sign down")
	testify.Equal(t, "-3", text_of(integer.Shift_Right(integer.From_Int_64(-5), 1)),
		"a negative shift rounds away from zero")
	_, over := integer.Shift_Left(integer.One(), integer.BIT_COUNT_MAXIMUM)
	testify.False(t, bool(over), "a shift past the width says so")
	_, wide := integer.Shift_Left(integer.One(), integer.SHIFT_COUNT_MAXIMUM)
	testify.False(t, bool(wide), "the largest shift says so")
	empty, ok := integer.Shift_Left(integer.Zero(), integer.SHIFT_COUNT_MAXIMUM)
	testify.True(t, bool(ok), "zero shifts to zero however far")
	testify.Equal(t, "0", text_of(empty), "zero shifted is zero")
	testify.Equal(t, "0", text_of(integer.Shift_Right(integer.One(),
		integer.SHIFT_COUNT_MAXIMUM)), "a shift down past the width empties a positive")
}

func test_bitwise(t *testing.T) {
	left := value_of(t, "12")
	right := value_of(t, "10")
	testify.Equal(t, "8", text_of(integer.And(left, right)), "twelve and ten is eight")
	testify.Equal(t, "14", text_of(integer.Or(left, right)), "twelve or ten is fourteen")
	testify.Equal(t, "6", text_of(integer.Exclusive_Or(left, right)),
		"twelve against ten is six")
	testify.Equal(t, "4", text_of(integer.And_Not(left, right)),
		"twelve without ten is four")
	testify.Equal(t, "-13", text_of(integer.Not(left)), "twelve flips to minus thirteen")
	testify.Equal(t, "-2", text_of(integer.And(integer.From_Int_64(-2),
		integer.From_Int_64(-1))), "two negatives meet in two's complement")
	testify.Equal(t, integer.Bit_Count(1), integer.Bit_Size(integer.One()),
		"one spans one bit")
	testify.Equal(t, integer.Bit_Count(2), integer.Bit_Size(value_of(t, "3")),
		"three spans two bits")
	testify.Equal(t, integer.Bit_Count(integer.BIT_COUNT_MAXIMUM),
		integer.Bit_Size(text_minimum(t)),
		"the value with no magnitude spans the whole width")
	testify.Equal(t, integer.Bit_Count(4), integer.Bit_Size(left), "twelve spans four bits")
	testify.Equal(t, integer.Bit_Count(0), integer.Bit_Size(integer.Zero()),
		"zero spans no bit")
	testify.Equal(t, integer.Bit_Count(201), integer.Bit_Size(
		value_of(t, "1606938044258990275541962092341162602522202993782792835301376")),
		"one shifted two hundred spans two hundred and one bits")
	testify.True(t, bool(integer.Bit(left, 2)), "twelve holds its third bit")
	testify.False(t, bool(integer.Bit(left, 0)), "twelve holds no first bit")
	testify.True(t, bool(integer.Bit(integer.From_Int_64(-1),
		integer.SHIFT_COUNT_MAXIMUM)), "a bit past the width reads the sign")
}

func test_common_divisor(t *testing.T) {
	for _, one := range []struct{ Left, Right, Want string }{
		{"12", "18", "6"},
		{"18", "12", "6"},
		{"-12", "18", "6"},
		{"12", "0", "12"},
		{"0", "0", "0"},
		{"17", "5", "1"},
		{"340282366920938463463374607431768211456", "18446744073709551616",
			"18446744073709551616"},
	} {
		result, ok := integer.Greatest_Common_Divisor(
			value_of(t, one.Left), value_of(t, one.Right))
		testify.True(t, bool(ok), "%s and %s share a divisor", one.Left, one.Right)
		testify.Equal(t, one.Want, text_of(result),
			"%s and %s divide by", one.Left, one.Right)
	}
	_, ok := integer.Greatest_Common_Divisor(text_minimum(t), integer.One())
	testify.False(t, bool(ok), "the most negative value has no magnitude to divide")
}

func test_text(t *testing.T) {
	for _, one := range []struct{ Literal, Want string }{
		{"0", "0"}, {"1", "1"}, {"42", "42"}, {"1_000", "1000"},
		{"0b1010", "10"}, {"0B1010", "10"}, {"0o17", "15"}, {"0O17", "15"},
		{"017", "15"}, {"0xff", "255"}, {"0XFF", "255"}, {"0x_ff", "255"},
		{"18446744073709551615", "18446744073709551615"},
		{"18446744073709551616", "18446744073709551616"},
	} {
		testify.Equal(t, one.Want, text_of(value_of(t, one.Literal)),
			"the literal %q reads its value", one.Literal)
	}
	// A literal carries any byte, thus every byte reaches the digit reader and every byte that
	// spells no digit refuses the literal.
	for _, one := range []string{"", "_", "0x", "0b", "0o", "abc", "0b2", "0o8", "12a",
		"-1", "1\x00", "1\x01", "1\x02", "1\xff"} {
		_, ok := integer.From_Text(integer.Text(one))
		testify.False(t, bool(ok), "the literal %q reads as no value", one)
	}
	var narrow [TEST_NARROW_SIZE]byte
	testify.Equal(t, integer.Digit_Count(0), integer.Into_Text(narrow[:], value_of(t, "1000")),
		"a destination that cannot hold the form writes nothing")
	testify.Equal(t, "-42", text_of(value_of(t, "-42")), "a negative reads its sign")
}

func test_bounds(t *testing.T) {
	testify.Equal(t, 512, integer.BIT_COUNT_MAXIMUM,
		"a value spans five hundred and twelve bits")
	widest, ok := integer.Shift_Left(integer.One(), integer.BIT_COUNT_MAXIMUM-2)
	testify.True(t, bool(ok), "the widest positive fits")
	testify.Equal(t, integer.Bit_Count(integer.BIT_COUNT_MAXIMUM-1),
		integer.Bit_Size(widest), "the widest positive spans the width but its sign")
	doubled, over := integer.Add(widest, widest)
	testify.False(t, bool(over), "a sum that reaches the sign bit says so")
	testify.True(t, bool(integer.Is_Negative(doubled)),
		"a sum past the width wears the wrapped sign it reports")
	var room [TEST_DIGITS_SIZE]byte
	largest := integer.Not(text_minimum(t))
	testify.Equal(t, integer.Digit_Count(154), integer.Into_Text(room[:], largest),
		"the largest value spells one hundred and fifty four digits")
	smallest, ok := integer.Negate(largest)
	testify.True(t, bool(ok), "the largest value negates")
	testify.Equal(t, integer.Digit_Count(integer.DIGIT_COUNT_MAXIMUM),
		integer.Into_Text(room[:], smallest),
		"the longest form spells a sign and every digit")
	var storage [integer.TEXT_SIZE_MAXIMUM]byte
	testify.Equal(t, integer.Digit_Count(0), integer.Into_Text(storage[:0], integer.One()),
		"no storage writes nothing")
	testify.Equal(t, integer.Digit_Count(1), integer.Into_Text(storage[:1], integer.One()),
		"one byte of storage holds one digit")
	testify.Equal(t, integer.Digit_Count(1), integer.Into_Text(storage[:], integer.One()),
		"the widest storage holds one digit")
	for index := range storage {
		storage[index] = '9'
	}
	_, wide := integer.From_Text(integer.Text(storage[:]))
	testify.False(t, bool(wide), "a literal past the width is refused")
	_, empty := integer.From_Text(integer.Text(storage[:0]))
	testify.False(t, bool(empty), "an empty literal is refused")
}

func test_allocation(t *testing.T) {
	fixture := allocation_fixture{}
	left := value_of(t, "340282366920938463463374607431768211456")
	right := value_of(t, "18446744073709551616")
	var storage [TEST_DIGITS_SIZE]byte
	checks := []struct {
		Name string
		Call func()
	}{
		{Name: "Add", Call: func() {
			fixture.Value, fixture.Ok = integer.Add(left, right)
		}},
		{Name: "Subtract", Call: func() {
			fixture.Value, fixture.Ok = integer.Subtract(left, right)
		}},
		{Name: "Multiply", Call: func() {
			fixture.Value, fixture.Ok = integer.Multiply(right, right)
		}},
		{Name: "Divide", Call: func() {
			fixture.Value, _, fixture.Ok = integer.Divide(left, right)
		}},
		{Name: "Compare", Call: func() { fixture.Order = integer.Compare(left, right) }},
		{Name: "Shift_Left", Call: func() {
			fixture.Value, fixture.Ok = integer.Shift_Left(right, 3)
		}},
		{Name: "Shift_Right", Call: func() {
			fixture.Value = integer.Shift_Right(left, 3)
		}},
		{Name: "And", Call: func() { fixture.Value = integer.And(left, right) }},
		{Name: "Greatest_Common_Divisor", Call: func() {
			fixture.Value, fixture.Ok = integer.Greatest_Common_Divisor(left, right)
		}},
		{Name: "From_Text", Call: func() {
			fixture.Value, fixture.Ok = integer.From_Text("123456789")
		}},
		{Name: "Into_Text", Call: func() {
			fixture.Count = integer.Into_Text(storage[:], left)
		}},
	}
	for _, check := range checks {
		t.Run(check.Name, func(t *testing.T) { testify.Zero_Allocation(t, check.Call) })
	}
	testify.True(t, bool(fixture.Ok), "the allocation fixture holds a value")
}

// Shifts one value up and refuses to hide a fixture that leaves the width.
func must_shift(
	t *testing.T, value integer.Integer, count integer.Shift_Count,
) (result integer.Integer) {
	result, ok := integer.Shift_Left(value, count)
	testify.True(t, bool(ok), "the fixture shifts by %d", int(count))
	return result
}
