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
type Numerator integer.Integer

// Numerator_Invariants states the width a numerator holds.
func Numerator_Invariants(value Numerator, namespace aver.Namespace) {
	aver.Always(
		len(value.Limbs) == integer.LIMB_COUNT,
		"A numerator holds one limb for every piece of its width.",
	)
}

// Denominator is the value below the line of a ratio. It is never zero and never negative.
type Denominator integer.Integer

// Denominator_Invariants states the width a denominator holds.
func Denominator_Invariants(value Denominator, namespace aver.Namespace) {
	aver.Always(
		len(value.Limbs) == integer.LIMB_COUNT,
		"A denominator holds one limb for every piece of its width.",
	)
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

// Zero is the ratio every sum starts from.
func Zero() (result Rational) {
	defer func() { Rational_Invariants(result, "zero.result") }()
	return Rational{
		Numerator:   Numerator(integer.Zero()),
		Denominator: Denominator(integer.One()),
	}
}

// One is the ratio every product starts from.
func One() (result Rational) {
	defer func() { Rational_Invariants(result, "one.result") }()
	return Rational{
		Numerator:   Numerator(integer.One()),
		Denominator: Denominator(integer.One()),
	}
}

// From_Integer lifts a whole value.
func From_Integer(value Numerator) (result Rational) {
	defer func() { Rational_Invariants(result, "from_integer.result") }()
	Numerator_Invariants(value, "from_integer.value")
	return Rational{Numerator: value, Denominator: Denominator(integer.One())}
}

// From_Ratio takes a numerator and a denominator and returns the ratio in lowest terms. A zero
// denominator names no value and is refused.
func From_Ratio(
	numerator Numerator, denominator Denominator,
) (result Rational, ok Boolean) {
	defer func() {
		Rational_Invariants(result, "from_ratio.result")
		Boolean_Invariants(ok, "from_ratio.ok")
	}()
	Numerator_Invariants(numerator, "from_ratio.numerator")
	Denominator_Invariants(denominator, "from_ratio.denominator")
	above := integer.Integer(numerator)
	below := integer.Integer(denominator)
	if bool(integer.Is_Zero(below)) {
		return Zero(), false
	}
	if bool(integer.Is_Negative(below)) {
		flipped_above, above_ok := integer.Negate(above)
		flipped_below, below_ok := integer.Negate(below)
		if !bool(above_ok) {
			return Zero(), false
		}
		if !bool(below_ok) {
			return Zero(), false
		}
		above = flipped_above
		below = flipped_below
	}
	divisor, divisor_ok := integer.Greatest_Common_Divisor(above, below)
	if !bool(divisor_ok) {
		return Zero(), false
	}
	if bool(integer.Is_Zero(divisor)) {
		return Zero(), true
	}
	above, _, _ = integer.Divide(above, divisor)
	below, _, _ = integer.Divide(below, divisor)
	return Rational{Numerator: Numerator(above), Denominator: Denominator(below)}, true
}

// Whole truncates a ratio toward zero.
func Whole(value Rational) (result Numerator, ok Boolean) {
	defer func() {
		Numerator_Invariants(result, "whole.result")
		Boolean_Invariants(ok, "whole.ok")
	}()
	Rational_Invariants(value, "whole.value")
	quotient, _, divided := integer.Divide(
		integer.Integer(value.Numerator), integer.Integer(value.Denominator))
	return Numerator(quotient), Boolean(divided)
}

// Is_Whole reports whether a ratio names an integer.
func Is_Whole(value Rational) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_whole.yes") }()
	Rational_Invariants(value, "is_whole.value")
	return integer.Compare(integer.Integer(value.Denominator), integer.One()) ==
		integer.ORDER_SAME
}

// Is_Zero reports whether a ratio names zero.
func Is_Zero(value Rational) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_zero.yes") }()
	Rational_Invariants(value, "is_zero.value")
	return Boolean(integer.Is_Zero(integer.Integer(value.Numerator)))
}

// Sign reports whether a ratio stands below, at, or above zero.
func Sign(value Rational) (order integer.Order) {
	defer func() { integer.Order_Invariants(order, "sign.order") }()
	Rational_Invariants(value, "sign.value")
	return integer.Sign(integer.Integer(value.Numerator))
}

// Add sums two ratios and normalises the result. Each step reports whether it held the width,
// thus a caller learns of an overflow rather than reading a wrapped ratio.
func Add(augend Rational, addend Rational) (sum Rational, ok Boolean) {
	defer func() {
		Rational_Invariants(sum, "add.sum")
		Boolean_Invariants(ok, "add.ok")
	}()
	Rational_Invariants(augend, "add.augend")
	Rational_Invariants(addend, "add.addend")
	left, left_ok := integer.Multiply(
		integer.Integer(augend.Numerator), integer.Integer(addend.Denominator))
	right, right_ok := integer.Multiply(
		integer.Integer(addend.Numerator), integer.Integer(augend.Denominator))
	below, below_ok := integer.Multiply(
		integer.Integer(augend.Denominator), integer.Integer(addend.Denominator))
	above, above_ok := integer.Add(left, right)
	if !bool(left_ok) {
		return Zero(), false
	}
	if !bool(right_ok) {
		return Zero(), false
	}
	if !bool(below_ok) {
		return Zero(), false
	}
	if !bool(above_ok) {
		return Zero(), false
	}
	return From_Ratio(Numerator(above), Denominator(below))
}

// Negate reverses the sign of a ratio.
func Negate(value Rational) (result Rational, ok Boolean) {
	defer func() {
		Rational_Invariants(result, "negate.result")
		Boolean_Invariants(ok, "negate.ok")
	}()
	Rational_Invariants(value, "negate.value")
	above, above_ok := integer.Negate(integer.Integer(value.Numerator))
	if !bool(above_ok) {
		return Zero(), false
	}
	return Rational{Numerator: Numerator(above), Denominator: value.Denominator}, true
}

// Subtract takes one ratio from another.
func Subtract(minuend Rational, subtrahend Rational) (difference Rational, ok Boolean) {
	defer func() {
		Rational_Invariants(difference, "subtract.difference")
		Boolean_Invariants(ok, "subtract.ok")
	}()
	Rational_Invariants(minuend, "subtract.minuend")
	Rational_Invariants(subtrahend, "subtract.subtrahend")
	flipped, flipped_ok := Negate(subtrahend)
	if !bool(flipped_ok) {
		return Zero(), false
	}
	return Add(minuend, flipped)
}

// Absolute reads the magnitude of a ratio.
func Absolute(value Rational) (result Rational, ok Boolean) {
	defer func() {
		Rational_Invariants(result, "absolute.result")
		Boolean_Invariants(ok, "absolute.ok")
	}()
	Rational_Invariants(value, "absolute.value")
	if Sign(value) != integer.ORDER_BEFORE {
		return value, true
	}
	return Negate(value)
}

// Multiply forms the product of two ratios and normalises it.
func Multiply(
	multiplicand Rational, multiplier Rational,
) (product Rational, ok Boolean) {
	defer func() {
		Rational_Invariants(product, "multiply.product")
		Boolean_Invariants(ok, "multiply.ok")
	}()
	Rational_Invariants(multiplicand, "multiply.multiplicand")
	Rational_Invariants(multiplier, "multiply.multiplier")
	above, above_ok := integer.Multiply(
		integer.Integer(multiplicand.Numerator), integer.Integer(multiplier.Numerator))
	below, below_ok := integer.Multiply(
		integer.Integer(multiplicand.Denominator), integer.Integer(multiplier.Denominator))
	if !bool(above_ok) {
		return Zero(), false
	}
	if !bool(below_ok) {
		return Zero(), false
	}
	return From_Ratio(Numerator(above), Denominator(below))
}

// Divide forms the quotient of two ratios. A zero divisor names no quotient and is refused.
func Divide(dividend Rational, divisor Rational) (quotient Rational, ok Boolean) {
	defer func() {
		Rational_Invariants(quotient, "divide.quotient")
		Boolean_Invariants(ok, "divide.ok")
	}()
	Rational_Invariants(dividend, "divide.dividend")
	Rational_Invariants(divisor, "divide.divisor")
	if bool(Is_Zero(divisor)) {
		return Zero(), false
	}
	above, above_ok := integer.Multiply(
		integer.Integer(dividend.Numerator), integer.Integer(divisor.Denominator))
	below, below_ok := integer.Multiply(
		integer.Integer(dividend.Denominator), integer.Integer(divisor.Numerator))
	if !bool(above_ok) {
		return Zero(), false
	}
	if !bool(below_ok) {
		return Zero(), false
	}
	return From_Ratio(Numerator(above), Denominator(below))
}

// Compare cross-multiplies and reads the order of the products. Both denominators stand above
// zero, thus the comparison needs no sign case of its own.
func Compare(left Rational, right Rational) (order integer.Order, ok Boolean) {
	defer func() {
		integer.Order_Invariants(order, "compare.order")
		Boolean_Invariants(ok, "compare.ok")
	}()
	Rational_Invariants(left, "compare.left")
	Rational_Invariants(right, "compare.right")
	first, first_ok := integer.Multiply(
		integer.Integer(left.Numerator), integer.Integer(right.Denominator))
	second, second_ok := integer.Multiply(
		integer.Integer(right.Numerator), integer.Integer(left.Denominator))
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
	Rational_Invariants(value, "into_text.value")
	var storage [TEXT_SIZE_MAXIMUM]byte
	above := integer.Into_Text(storage[:], integer.Integer(value.Numerator))
	written := int(above)
	if written == 0 {
		return TEXT_SIZE_MINIMUM
	}
	if !bool(Is_Whole(value)) {
		storage[written] = SEPARATOR
		written++
		below := integer.Into_Text(
			storage[written:], integer.Integer(value.Denominator))
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
