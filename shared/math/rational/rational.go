// Package rational holds an exact ratio of two fixed-width integers. It exists because the Go
// constant grammar states a floating value exactly, and an exact value is a ratio and never a
// float. Every value is two fixed arrays the caller holds, thus no operation allocates.
package rational

import (
	"local/james-orcales/shared/math/integer"
	"local/james-orcales/shared/sim/aver/default"
)

// TEXT_SIZE_MINIMUM admits no output.
const TEXT_SIZE_MINIMUM = 0

// TEXT_SIZE_MAXIMUM holds the longest form: a signed numerator, a slash, and a denominator.
// A denominator never carries a sign, thus it spells one byte fewer than a numerator.
const TEXT_SIZE_MAXIMUM = integer.DIGIT_COUNT_MAXIMUM * 2

// SEPARATOR is the byte that stands between a numerator and a denominator.
const SEPARATOR = '/'

// Numerator is the value above the line of a ratio.
type Numerator integer.Limbs

// Numerator_Invariants states the width a numerator holds.
func Numerator_Invariants(value Numerator, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Limb_0), integer.LIMB_MINIMUM, integer.LIMB_MAXIMUM).
		Range_Uint64(uint64(value.Limb_1), integer.LIMB_MINIMUM, integer.LIMB_MAXIMUM).
		Range_Uint64(uint64(value.Limb_2), integer.LIMB_MINIMUM, integer.LIMB_MAXIMUM).
		Range_Uint64(uint64(value.Limb_3), integer.LIMB_MINIMUM, integer.LIMB_MAXIMUM).
		Range_Uint64(uint64(value.Limb_4), integer.LIMB_MINIMUM, integer.LIMB_MAXIMUM).
		Range_Uint64(uint64(value.Limb_5), integer.LIMB_MINIMUM, integer.LIMB_MAXIMUM).
		Range_Uint64(uint64(value.Limb_6), integer.LIMB_MINIMUM, integer.LIMB_MAXIMUM).
		Range_Uint64(uint64(value.Limb_7), integer.LIMB_MINIMUM, integer.LIMB_MAXIMUM).
		Ensure()
}

func numerator_integer(value Numerator) (result integer.Integer) {
	defer func() {
		integer.Integer_Invariants(result, "rational.numerator_integer.result")
	}()
	Numerator_Invariants(value, "rational.numerator_integer.value")
	return integer.Integer{Limbs: integer.Limbs(value)}
}

// Denominator is the value below the line of a ratio. It is never zero and never negative.
type Denominator integer.Limbs

// Denominator_Invariants states the width a denominator holds.
func Denominator_Invariants(value Denominator, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Limb_0), integer.LIMB_MINIMUM, integer.LIMB_MAXIMUM).
		Range_Uint64(uint64(value.Limb_1), integer.LIMB_MINIMUM, integer.LIMB_MAXIMUM).
		Range_Uint64(uint64(value.Limb_2), integer.LIMB_MINIMUM, integer.LIMB_MAXIMUM).
		Range_Uint64(uint64(value.Limb_3), integer.LIMB_MINIMUM, integer.LIMB_MAXIMUM).
		Range_Uint64(uint64(value.Limb_4), integer.LIMB_MINIMUM, integer.LIMB_MAXIMUM).
		Range_Uint64(uint64(value.Limb_5), integer.LIMB_MINIMUM, integer.LIMB_MAXIMUM).
		Range_Uint64(uint64(value.Limb_6), integer.LIMB_MINIMUM, integer.LIMB_MAXIMUM).
		Range_Uint64(uint64(value.Limb_7), integer.LIMB_MINIMUM, integer.LIMB_MAXIMUM).
		Ensure()
}

func denominator_integer(value Denominator) (result integer.Integer) {
	defer func() {
		integer.Integer_Invariants(result, "rational.denominator_integer.result")
	}()
	Denominator_Invariants(value, "rational.denominator_integer.value")
	return integer.Integer{Limbs: integer.Limbs(value)}
}

// Boolean is a true or false report about one ratio.
type Boolean bool

// Boolean_Invariants states both ratio reports as obligations.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The rational report is true.").
		Ensure()
}

// Text_Count is the byte count one written form spans.
type Text_Count int

// Text_Count_Invariants states the byte count of the longest written form.
func Text_Count_Invariants(value Text_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Digits is caller storage one ratio writes its form into.
type Digits []byte

// Digits_Invariants states the storage the longest written form needs.
func Digits_Invariants(value Digits, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Rational is an exact ratio in lowest terms.
type Rational struct {
	// Numerator carries the value and the sign of the ratio.
	Numerator Numerator
	// Denominator is never zero and never negative, thus one value wears one shape.
	Denominator Denominator
}

// Rational_Invariants composes both halves of one ratio.
func Rational_Invariants(value Rational, namespace aver.Namespace) {
	Numerator_Invariants(value.Numerator, namespace)
	Denominator_Invariants(value.Denominator, namespace)
}

// Rational_Handle keeps caller-owned ratio storage explicit.
type Rational_Handle *Rational

// Rational_Handle_Invariants composes present ratio storage.
func Rational_Handle_Invariants(value Rational_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Rational_Invariants(*value, namespace)
}

// Numerator_Handle keeps caller-owned whole-number output explicit.
type Numerator_Handle *Numerator

// Numerator_Handle_Invariants composes present numerator storage.
func Numerator_Handle_Invariants(value Numerator_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Numerator_Invariants(*value, namespace)
}

// Zero is the ratio every sum starts from.
func Zero(destination Rational_Handle) {
	Rational_Handle_Invariants(destination, "rational.zero.destination")
	above := integer.Integer{}
	integer.Zero(&above)
	below := integer.Integer{}
	integer.One(&below)
	*destination = Rational{
		Numerator:   Numerator(above.Limbs),
		Denominator: Denominator(below.Limbs),
	}
}

// One is the ratio every product starts from.
func One(destination Rational_Handle) {
	Rational_Handle_Invariants(destination, "rational.one.destination")
	unit := integer.Integer{}
	integer.One(&unit)
	*destination = Rational{
		Numerator:   Numerator(unit.Limbs),
		Denominator: Denominator(unit.Limbs),
	}
}

// From_Integer lifts a whole value.
func From_Integer(destination Rational_Handle, value Numerator) {
	Rational_Handle_Invariants(destination, "rational.from_integer.destination")
	Numerator_Invariants(value, "rational.from_integer.value")
	unit := integer.Integer{}
	integer.One(&unit)
	*destination = Rational{Numerator: value, Denominator: Denominator(unit.Limbs)}
}

// From_Ratio takes a numerator and a denominator and returns the ratio in lowest terms. A zero
// denominator names no value and is refused.
func From_Ratio(
	destination Rational_Handle,
	numerator Numerator,
	denominator Denominator,
) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "rational.from_ratio.ok") }()
	Rational_Handle_Invariants(destination, "rational.from_ratio.destination")
	Numerator_Invariants(numerator, "rational.from_ratio.numerator")
	Denominator_Invariants(denominator, "rational.from_ratio.denominator")
	above := numerator_integer(numerator)
	below := denominator_integer(denominator)
	if bool(integer.Is_Zero(below)) {
		Zero(destination)
		return false
	}
	if bool(integer.Is_Negative(below)) {
		flipped_above := integer.Integer{}
		above_ok := integer.Negate(&flipped_above, above)
		flipped_below := integer.Integer{}
		below_ok := integer.Negate(&flipped_below, below)
		if !bool(above_ok) {
			Zero(destination)
			return false
		}
		if !bool(below_ok) {
			Zero(destination)
			return false
		}
		above = flipped_above
		below = flipped_below
	}
	divisor := integer.Integer{}
	divisor_ok := integer.Greatest_Common_Divisor(&divisor, above, below)
	if !bool(divisor_ok) {
		Zero(destination)
		return false
	}
	if bool(integer.Is_Zero(divisor)) {
		Zero(destination)
		return true
	}
	quotient := integer.Integer{}
	remainder := integer.Integer{}
	integer.Divide(&quotient, &remainder, above, divisor)
	above = quotient
	integer.Divide(&quotient, &remainder, below, divisor)
	below = quotient
	*destination = Rational{
		Numerator:   Numerator(above.Limbs),
		Denominator: Denominator(below.Limbs),
	}
	return true
}

// Whole truncates a ratio toward zero.
func Whole(destination Numerator_Handle, value Rational) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "rational.whole.ok") }()
	Numerator_Handle_Invariants(destination, "rational.whole.destination")
	Rational_Invariants(value, "rational.whole.value")
	quotient := integer.Integer{}
	remainder := integer.Integer{}
	divided := integer.Divide(
		&quotient, &remainder,
		numerator_integer(value.Numerator), denominator_integer(value.Denominator))
	*destination = Numerator(quotient.Limbs)
	return Boolean(divided)
}

// Is_Whole reports whether a ratio names an integer.
func Is_Whole(value Rational) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_whole.yes") }()
	Rational_Invariants(value, "rational.is_whole.value")
	unit := integer.Integer{}
	integer.One(&unit)
	return integer.Compare(denominator_integer(value.Denominator), unit) ==
		integer.ORDER_SAME
}

// Is_Zero reports whether a ratio names zero.
func Is_Zero(value Rational) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_zero.yes") }()
	Rational_Invariants(value, "rational.is_zero.value")
	return Boolean(integer.Is_Zero(numerator_integer(value.Numerator)))
}

// Sign reports whether a ratio stands below, at, or above zero.
func Sign(value Rational) (order integer.Order) {
	defer func() { integer.Order_Invariants(order, "sign.order") }()
	Rational_Invariants(value, "rational.sign.value")
	return integer.Sign(numerator_integer(value.Numerator))
}

// Add sums two ratios and normalises the result. Each step reports whether it held the width,
// thus a caller learns of an overflow rather than reading a wrapped ratio.
func Add(destination Rational_Handle, augend Rational, addend Rational) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "rational.add.ok") }()
	Rational_Handle_Invariants(destination, "rational.add.destination")
	Rational_Invariants(augend, "rational.add.augend")
	Rational_Invariants(addend, "rational.add.addend")
	left := integer.Integer{}
	left_ok := integer.Multiply(
		&left, numerator_integer(augend.Numerator),
		denominator_integer(addend.Denominator))
	right := integer.Integer{}
	right_ok := integer.Multiply(
		&right, numerator_integer(addend.Numerator),
		denominator_integer(augend.Denominator))
	below := integer.Integer{}
	below_ok := integer.Multiply(
		&below, denominator_integer(augend.Denominator),
		denominator_integer(addend.Denominator))
	above := integer.Integer{}
	above_ok := integer.Add(&above, left, right)
	if !bool(left_ok) {
		Zero(destination)
		return false
	}
	if !bool(right_ok) {
		Zero(destination)
		return false
	}
	if !bool(below_ok) {
		Zero(destination)
		return false
	}
	if !bool(above_ok) {
		Zero(destination)
		return false
	}
	return From_Ratio(
		destination, Numerator(above.Limbs), Denominator(below.Limbs))
}

// Negate reverses the sign of a ratio.
func Negate(destination Rational_Handle, value Rational) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "rational.negate.ok") }()
	Rational_Handle_Invariants(destination, "rational.negate.destination")
	Rational_Invariants(value, "rational.negate.value")
	above := integer.Integer{}
	above_ok := integer.Negate(&above, numerator_integer(value.Numerator))
	if !bool(above_ok) {
		Zero(destination)
		return false
	}
	*destination = Rational{
		Numerator: Numerator(above.Limbs), Denominator: value.Denominator,
	}
	return true
}

// Subtract takes one ratio from another.
func Subtract(
	destination Rational_Handle, minuend Rational, subtrahend Rational,
) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "rational.subtract.ok") }()
	Rational_Handle_Invariants(destination, "rational.subtract.destination")
	Rational_Invariants(minuend, "rational.subtract.minuend")
	Rational_Invariants(subtrahend, "rational.subtract.subtrahend")
	flipped := Rational{}
	flipped_ok := Negate(&flipped, subtrahend)
	if !bool(flipped_ok) {
		Zero(destination)
		return false
	}
	return Add(destination, minuend, flipped)
}

// Absolute reads the magnitude of a ratio.
func Absolute(destination Rational_Handle, value Rational) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "rational.absolute.ok") }()
	Rational_Handle_Invariants(destination, "rational.absolute.destination")
	Rational_Invariants(value, "rational.absolute.value")
	if Sign(value) != integer.ORDER_BEFORE {
		*destination = value
		return true
	}
	return Negate(destination, value)
}

// Multiply forms the product of two ratios and normalises it.
func Multiply(
	destination Rational_Handle, multiplicand Rational, multiplier Rational,
) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "rational.multiply.ok") }()
	Rational_Handle_Invariants(destination, "rational.multiply.destination")
	Rational_Invariants(multiplicand, "rational.multiply.multiplicand")
	Rational_Invariants(multiplier, "rational.multiply.multiplier")
	above := integer.Integer{}
	above_ok := integer.Multiply(
		&above, numerator_integer(multiplicand.Numerator),
		numerator_integer(multiplier.Numerator))
	below := integer.Integer{}
	below_ok := integer.Multiply(
		&below, denominator_integer(multiplicand.Denominator),
		denominator_integer(multiplier.Denominator))
	if !bool(above_ok) {
		Zero(destination)
		return false
	}
	if !bool(below_ok) {
		Zero(destination)
		return false
	}
	return From_Ratio(
		destination, Numerator(above.Limbs), Denominator(below.Limbs))
}

// Divide forms the quotient of two ratios. A zero divisor names no quotient and is refused.
func Divide(destination Rational_Handle, dividend Rational, divisor Rational) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "rational.divide.ok") }()
	Rational_Handle_Invariants(destination, "rational.divide.destination")
	Rational_Invariants(dividend, "rational.divide.dividend")
	Rational_Invariants(divisor, "rational.divide.divisor")
	if bool(Is_Zero(divisor)) {
		Zero(destination)
		return false
	}
	above := integer.Integer{}
	above_ok := integer.Multiply(
		&above, numerator_integer(dividend.Numerator),
		denominator_integer(divisor.Denominator))
	below := integer.Integer{}
	below_ok := integer.Multiply(
		&below, denominator_integer(dividend.Denominator),
		numerator_integer(divisor.Numerator))
	if !bool(above_ok) {
		Zero(destination)
		return false
	}
	if !bool(below_ok) {
		Zero(destination)
		return false
	}
	return From_Ratio(
		destination, Numerator(above.Limbs), Denominator(below.Limbs))
}

// Compare cross-multiplies and reads the order of the products. Both denominators stand above
// zero, thus the comparison needs no sign case of its own.
func Compare(left Rational, right Rational) (order integer.Order, ok Boolean) {
	defer func() {
		integer.Order_Invariants(order, "compare.order")
		Boolean_Invariants(ok, "compare.ok")
	}()
	Rational_Invariants(left, "rational.compare.left")
	Rational_Invariants(right, "rational.compare.right")
	first := integer.Integer{}
	first_ok := integer.Multiply(
		&first, numerator_integer(left.Numerator),
		denominator_integer(right.Denominator))
	second := integer.Integer{}
	second_ok := integer.Multiply(
		&second, numerator_integer(right.Numerator),
		denominator_integer(left.Denominator))
	if !bool(first_ok) {
		return integer.ORDER_SAME, false
	}
	if !bool(second_ok) {
		return integer.ORDER_SAME, false
	}
	return integer.Compare(first, second), true
}

// Into_Text writes the form of a ratio into caller storage and returns the byte count. A whole
// ratio writes its numerator alone, thus the form reads back as the value it names.
func Into_Text(destination Digits, value Rational) (count Text_Count) {
	defer func() { Text_Count_Invariants(count, "into_text.count") }()
	Digits_Invariants(destination, "into_text.destination")
	Rational_Invariants(value, "rational.into_text.value")
	var storage [TEXT_SIZE_MAXIMUM]byte
	above := integer.Into_Text(storage[:], numerator_integer(value.Numerator))
	written := int(above)
	if written == 0 {
		return TEXT_SIZE_MINIMUM
	}
	if !bool(Is_Whole(value)) {
		storage[written] = SEPARATOR
		written++
		below := integer.Into_Text(
			storage[written:], denominator_integer(value.Denominator))
		if below == 0 {
			return TEXT_SIZE_MINIMUM
		}
		written = written + int(below)
	}
	if len(destination) < written {
		return TEXT_SIZE_MINIMUM
	}
	copy(destination, storage[:written])
	return Text_Count(written)
}
