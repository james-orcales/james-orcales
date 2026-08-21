// Package integer holds a two's complement integer of fixed width. It exists because the Go
// constant grammar needs exact arithmetic far past a machine word, and this repository admits
// neither the standard big-number package nor a float. Every value is a fixed array the caller
// holds, thus no operation allocates.
package integer

import (
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
)

// LIMB_BIT_COUNT is the bit width of one limb.
const LIMB_BIT_COUNT = 64

// LIMB_COUNT is the limb count of one value. Eight limbs span 512 bits, which is twice the
// mantissa the Go specification asks a constant to carry.
const LIMB_COUNT = 8

// LIMB_INDEX_MINIMUM is the least significant limb.
const LIMB_INDEX_MINIMUM = 0

// LIMB_INDEX_MAXIMUM is the most significant limb.
const LIMB_INDEX_MAXIMUM = LIMB_COUNT - 1

// LIMB_MINIMUM is the smallest limb value.
const LIMB_MINIMUM = 0

// LIMB_MAXIMUM is the largest limb value.
const LIMB_MAXIMUM = 1<<LIMB_BIT_COUNT - 1

// BIT_COUNT_MINIMUM is the bit length of zero.
const BIT_COUNT_MINIMUM = 0

// BIT_COUNT_MAXIMUM is the bit width of one value.
const BIT_COUNT_MAXIMUM = LIMB_COUNT * LIMB_BIT_COUNT

// SIGN_BIT_INDEX is the bit that carries the sign of a two's complement value.
const SIGN_BIT_INDEX = LIMB_BIT_COUNT - 1

// SHIFT_COUNT_MINIMUM is a shift that moves nothing.
const SHIFT_COUNT_MINIMUM = 0

// SHIFT_COUNT_MAXIMUM caps a shift. A count past the width empties the value, thus a larger cap
// would state nothing a caller could not read from the width itself.
const SHIFT_COUNT_MAXIMUM = 4096

// ORDER_BEFORE marks a left value smaller than the right one.
const ORDER_BEFORE Order = -1

// ORDER_SAME marks two equal values.
const ORDER_SAME Order = 0

// ORDER_AFTER marks a left value larger than the right one.
const ORDER_AFTER Order = 1

// INT_64_MINIMUM is the smallest machine integer.
const INT_64_MINIMUM = -1 << 63

// INT_64_MAXIMUM is the largest machine integer.
const INT_64_MAXIMUM = 1<<63 - 1

// TEXT_SIZE_MINIMUM admits the empty literal, which reads as no value.
const TEXT_SIZE_MINIMUM = 0

// TEXT_SIZE_MAXIMUM caps a literal. A 512-bit value spells 155 decimal digits, and the rest of
// the room holds a base prefix, a sign, and the underscores a reader groups digits with.
const TEXT_SIZE_MAXIMUM = 1024

// DIGIT_COUNT_MINIMUM is the byte count of no output.
const DIGIT_COUNT_MINIMUM = 0

// DIGIT_COUNT_MAXIMUM is the byte count of the longest decimal form. The largest magnitude
// the width holds spells 154 digits, and the sign of its negative takes one byte more.
const DIGIT_COUNT_MAXIMUM = 155

// DECIMAL_BASE is the base Into_Text writes.
const DECIMAL_BASE = 10

// BASE_BINARY is the base a zero-b prefix names.
const BASE_BINARY = 2

// BASE_OCTAL is the base a zero-o prefix names.
const BASE_OCTAL = 8

// BASE_HEXADECIMAL is the base a zero-x prefix names.
const BASE_HEXADECIMAL = 16

// BASE_MINIMUM is the smallest base a literal names.
const BASE_MINIMUM = BASE_BINARY

// BASE_MAXIMUM is the largest base a literal names.
const BASE_MAXIMUM = BASE_HEXADECIMAL

// DIGIT_BYTE_MINIMUM is the smallest byte a literal can carry.
const DIGIT_BYTE_MINIMUM = 0

// DIGIT_BYTE_MAXIMUM is the largest byte a literal can carry.
const DIGIT_BYTE_MAXIMUM = 255

// DIGIT_VALUE_MINIMUM is the value of the digit zero.
const DIGIT_VALUE_MINIMUM = 0

// DIGIT_VALUE_MAXIMUM is the value of the digit f.
const DIGIT_VALUE_MAXIMUM = BASE_HEXADECIMAL - 1

// DIGIT_VALUE_ABSENT marks a byte that spells no digit.
const DIGIT_VALUE_ABSENT = -1

// Limb is one 64-bit piece of a value, least significant first.
type Limb uint64

// Limb_Invariants states the complete limb domain.
func Limb_Invariants(value Limb, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), LIMB_MINIMUM, LIMB_MAXIMUM).
		Ensure()
}

// Limb_0 gives least-significant storage independent invariant identity.
type Limb_0 Limb

// Limb_0_Invariants states complete word domain.
func Limb_0_Invariants(value Limb_0, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), LIMB_MINIMUM, LIMB_MAXIMUM).
		Ensure()
}

// Limb_1 gives second storage word independent invariant identity.
type Limb_1 Limb

// Limb_1_Invariants states complete word domain.
func Limb_1_Invariants(value Limb_1, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), LIMB_MINIMUM, LIMB_MAXIMUM).
		Ensure()
}

// Limb_2 gives third storage word independent invariant identity.
type Limb_2 Limb

// Limb_2_Invariants states complete word domain.
func Limb_2_Invariants(value Limb_2, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), LIMB_MINIMUM, LIMB_MAXIMUM).
		Ensure()
}

// Limb_3 gives fourth storage word independent invariant identity.
type Limb_3 Limb

// Limb_3_Invariants states complete word domain.
func Limb_3_Invariants(value Limb_3, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), LIMB_MINIMUM, LIMB_MAXIMUM).
		Ensure()
}

// Limb_4 gives fifth storage word independent invariant identity.
type Limb_4 Limb

// Limb_4_Invariants states complete word domain.
func Limb_4_Invariants(value Limb_4, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), LIMB_MINIMUM, LIMB_MAXIMUM).
		Ensure()
}

// Limb_5 gives sixth storage word independent invariant identity.
type Limb_5 Limb

// Limb_5_Invariants states complete word domain.
func Limb_5_Invariants(value Limb_5, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), LIMB_MINIMUM, LIMB_MAXIMUM).
		Ensure()
}

// Limb_6 gives seventh storage word independent invariant identity.
type Limb_6 Limb

// Limb_6_Invariants states complete word domain.
func Limb_6_Invariants(value Limb_6, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), LIMB_MINIMUM, LIMB_MAXIMUM).
		Ensure()
}

// Limb_7 gives sign-bearing storage word independent invariant identity.
type Limb_7 Limb

// Limb_7_Invariants states complete word domain.
func Limb_7_Invariants(value Limb_7, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), LIMB_MINIMUM, LIMB_MAXIMUM).
		Ensure()
}

// Limb_Index names one limb of a value.
type Limb_Index int

// Limb_Index_Invariants states the complete limb position domain.
func Limb_Index_Invariants(value Limb_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), LIMB_INDEX_MINIMUM, LIMB_INDEX_MAXIMUM).
		Ensure()
}

// Bit_Count is a bit length of one magnitude.
type Bit_Count int

// Bit_Count_Invariants admits the zero length of zero itself.
func Bit_Count_Invariants(value Bit_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), BIT_COUNT_MINIMUM, BIT_COUNT_MAXIMUM).
		Ensure()
}

// Shift_Count is how far a shift moves a value.
type Shift_Count int

// Shift_Count_Invariants admits a count past the width, which empties the value.
func Shift_Count_Invariants(value Shift_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), SHIFT_COUNT_MINIMUM, SHIFT_COUNT_MAXIMUM).
		Ensure()
}

// Order reports where one value stands against another.
type Order int

// Order_Invariants states the three standings two values can hold.
func Order_Invariants(value Order, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Int(int(value), int(ORDER_BEFORE), int(ORDER_SAME), int(ORDER_AFTER)).
		Ensure()
}

// Int_64 is a machine integer a value lifts from or lowers to.
type Int_64 int64

// Int_64_Invariants states the complete machine integer domain.
func Int_64_Invariants(value Int_64, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), INT_64_MINIMUM, INT_64_MAXIMUM).
		Ensure()
}

// Boolean is a true or false report about one value.
type Boolean bool

// Boolean_Invariants states both value reports as obligations.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The integer report is true.").
		Ensure()
}

// Base is the base one literal spells its digits in.
type Base int

// Base_Invariants states the four bases a Go literal names.
func Base_Invariants(value Base, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Int(int(value), BASE_BINARY, BASE_OCTAL, DECIMAL_BASE, BASE_HEXADECIMAL).
		Ensure()
}

// Digit_Byte is one byte of a literal, whatever it spells.
type Digit_Byte byte

// Digit_Byte_Invariants states the complete byte domain, because a literal carries any byte.
func Digit_Byte_Invariants(value Digit_Byte, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), DIGIT_BYTE_MINIMUM, DIGIT_BYTE_MAXIMUM).
		Ensure()
}

// Digit_Value is the value one digit byte spells, or DIGIT_VALUE_ABSENT for no digit.
type Digit_Value int

// Digit_Value_Invariants admits the absent value, which marks a byte that spells no digit.
func Digit_Value_Invariants(value Digit_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), DIGIT_VALUE_ABSENT, DIGIT_VALUE_MAXIMUM).
		Ensure()
}

// Text is a literal a caller reads a value from.
type Text string

// Text_Invariants caps a literal so a hostile one is refused before the first digit.
func Text_Invariants(value Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Digits is caller storage one value writes its decimal form into.
type Digits []byte

// Digits_Invariants states the storage the longest decimal form needs.
func Digits_Invariants(value Digits, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), DIGIT_COUNT_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Digit_Count is the byte count one decimal form spans.
type Digit_Count int

// Digit_Count_Invariants states the byte count of the longest decimal form.
func Digit_Count_Invariants(value Digit_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), DIGIT_COUNT_MINIMUM, DIGIT_COUNT_MAXIMUM).
		Ensure()
}

// Limbs is fixed-width storage. Separate fields keep value copies caller-owned without array
// parameters or slice backing storage.
type Limbs struct {
	// Limb_0 stays inline so copies own least-significant storage.
	Limb_0 Limb_0
	// Limb_1 stays inline so copies own second storage word.
	Limb_1 Limb_1
	// Limb_2 stays inline so copies own third storage word.
	Limb_2 Limb_2
	// Limb_3 stays inline so copies own fourth storage word.
	Limb_3 Limb_3
	// Limb_4 stays inline so copies own fifth storage word.
	Limb_4 Limb_4
	// Limb_5 stays inline so copies own sixth storage word.
	Limb_5 Limb_5
	// Limb_6 stays inline so copies own seventh storage word.
	Limb_6 Limb_6
	// Limb_7 stays inline so copies own sign-bearing storage word.
	Limb_7 Limb_7
}

// Limbs_Invariants states the width every value holds.
func Limbs_Invariants(value Limbs, namespace aver.Namespace) {
	Limb_0_Invariants(value.Limb_0, namespace)
	Limb_1_Invariants(value.Limb_1, namespace)
	Limb_2_Invariants(value.Limb_2, namespace)
	Limb_3_Invariants(value.Limb_3, namespace)
	Limb_4_Invariants(value.Limb_4, namespace)
	Limb_5_Invariants(value.Limb_5, namespace)
	Limb_6_Invariants(value.Limb_6, namespace)
	Limb_7_Invariants(value.Limb_7, namespace)
}

func limb_at(value Limbs, index Limb_Index) (limb Limb) {
	defer func() { Limb_Invariants(limb, "limb_at.limb") }()
	Limbs_Invariants(value, "limb_at.value")
	Limb_Index_Invariants(index, "limb_at.index")
	switch index {
	case 0:
		return Limb(value.Limb_0)
	case 1:
		return Limb(value.Limb_1)
	case 2:
		return Limb(value.Limb_2)
	case 3:
		return Limb(value.Limb_3)
	case 4:
		return Limb(value.Limb_4)
	case 5:
		return Limb(value.Limb_5)
	case 6:
		return Limb(value.Limb_6)
	case 7:
		return Limb(value.Limb_7)
	}
	return 0
}

func limbs_with(value Limbs, index Limb_Index, limb Limb) (result Limbs) {
	defer func() { Limbs_Invariants(result, "limbs_with.result") }()
	Limbs_Invariants(value, "limbs_with.value")
	Limb_Index_Invariants(index, "limbs_with.index")
	Limb_Invariants(limb, "limbs_with.limb")
	result = value
	switch index {
	case 0:
		result.Limb_0 = Limb_0(limb)
	case 1:
		result.Limb_1 = Limb_1(limb)
	case 2:
		result.Limb_2 = Limb_2(limb)
	case 3:
		result.Limb_3 = Limb_3(limb)
	case 4:
		result.Limb_4 = Limb_4(limb)
	case 5:
		result.Limb_5 = Limb_5(limb)
	case 6:
		result.Limb_6 = Limb_6(limb)
	case 7:
		result.Limb_7 = Limb_7(limb)
	}
	return result
}

// Integer is a two's complement value of fixed width, least significant limb first.
type Integer struct {
	// Limbs holds the value, least significant limb first. The top bit of the final limb
	// carries the sign, thus a negative value needs no field of its own.
	Limbs Limbs
}

// Integer_Invariants states the width every value holds.
func Integer_Invariants(value Integer, namespace aver.Namespace) {
	Limbs_Invariants(value.Limbs, namespace)
}

// Integer_Handle keeps caller-owned storage explicit across mutation boundaries.
type Integer_Handle *Integer

// Integer_Handle_Invariants composes present integer storage.
func Integer_Handle_Invariants(value Integer_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Integer_Invariants(*value, namespace)
}

// Zero is the value every magnitude starts from.
func Zero(destination Integer_Handle) {
	Integer_Handle_Invariants(destination, "zero.destination")
	*destination = Integer{Limbs: Limbs{}}
}

// One is the unit every count steps by.
func One(destination Integer_Handle) {
	Integer_Handle_Invariants(destination, "one.destination")
	*destination = Integer{}
	destination.Limbs = limbs_with(destination.Limbs, LIMB_INDEX_MINIMUM, 1)
}

// From_Int_64 lifts a machine integer, carrying its sign into every limb above it.
func From_Int_64(destination Integer_Handle, value Int_64) {
	Integer_Handle_Invariants(destination, "from_int_64.destination")
	Int_64_Invariants(value, "from_int_64.value")
	result := Integer{}
	fill := Limb(0)
	if value < 0 {
		fill = LIMB_MAXIMUM
	}
	for index := range LIMB_COUNT {
		result.Limbs = limbs_with(result.Limbs, Limb_Index(index), fill)
	}
	result.Limbs = limbs_with(result.Limbs, LIMB_INDEX_MINIMUM, Limb(uint64(value)))
	*destination = result
}

// To_Int_64 lowers a value and reports whether it fits a machine integer.
func To_Int_64(value Integer) (result Int_64, ok Boolean) {
	defer func() {
		Int_64_Invariants(result, "to_int_64.result")
		Boolean_Invariants(ok, "to_int_64.ok")
	}()
	Integer_Invariants(value, "to_int_64.value")
	fill := Limb(0)
	if bool(Is_Negative(value)) {
		fill = LIMB_MAXIMUM
	}
	for index := LIMB_INDEX_MINIMUM + 1; index <= LIMB_INDEX_MAXIMUM; index++ {
		if limb_at(value.Limbs, Limb_Index(index)) != fill {
			return 0, false
		}
	}
	low := limb_at(value.Limbs, LIMB_INDEX_MINIMUM)
	if bool(Is_Negative(value)) != (low>>SIGN_BIT_INDEX == 1) {
		return 0, false
	}
	return Int_64(int64(low)), true
}

// Is_Negative reports whether the sign bit of a value stands.
func Is_Negative(value Integer) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_negative.yes") }()
	Integer_Invariants(value, "is_negative.value")
	return limb_at(value.Limbs, LIMB_INDEX_MAXIMUM)>>SIGN_BIT_INDEX == 1
}

// Is_Zero reports whether every limb of a value is empty.
func Is_Zero(value Integer) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_zero.yes") }()
	Integer_Invariants(value, "is_zero.value")
	for index := range LIMB_COUNT {
		if limb_at(value.Limbs, Limb_Index(index)) != 0 {
			return false
		}
	}
	return true
}

// Sign reports whether a value stands below, at, or above zero.
func Sign(value Integer) (order Order) {
	defer func() { Order_Invariants(order, "sign.order") }()
	Integer_Invariants(value, "sign.value")
	if bool(Is_Zero(value)) {
		return ORDER_SAME
	}
	if bool(Is_Negative(value)) {
		return ORDER_BEFORE
	}
	return ORDER_AFTER
}

// Add sums two values and reports whether the true sum fits the width. Two operands of one sign
// whose sum carries the other sign left the width, which is the whole overflow test.
func Add(destination Integer_Handle, augend Integer, addend Integer) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "add.ok") }()
	Integer_Handle_Invariants(destination, "add.destination")
	Integer_Invariants(augend, "add.augend")
	Integer_Invariants(addend, "add.addend")
	sum := Integer{}
	carry := bits.Carry_In(0)
	for index := range LIMB_COUNT {
		total, next := bits.Add_64(
			bits.Word_64(limb_at(augend.Limbs, Limb_Index(index))),
			bits.Addend_64(limb_at(addend.Limbs, Limb_Index(index))),
			carry,
		)
		sum.Limbs = limbs_with(sum.Limbs, Limb_Index(index), Limb(total))
		carry = bits.Carry_In(next)
	}
	*destination = sum
	if Is_Negative(augend) != Is_Negative(addend) {
		return true
	}
	return Is_Negative(augend) == Is_Negative(sum)
}

// Not flips every bit of a value.
func Not(destination Integer_Handle, value Integer) {
	Integer_Handle_Invariants(destination, "not.destination")
	Integer_Invariants(value, "not.value")
	result := Integer{}
	for index := range LIMB_COUNT {
		result.Limbs = limbs_with(
			result.Limbs, Limb_Index(index), ^limb_at(value.Limbs, Limb_Index(index)))
	}
	*destination = result
}

// Negate reverses the sign of a value. The most negative value has no positive twin inside the
// width, thus it alone reports an overflow.
func Negate(destination Integer_Handle, value Integer) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "negate.ok") }()
	Integer_Handle_Invariants(destination, "negate.destination")
	Integer_Invariants(value, "negate.value")
	inverted := Integer{}
	Not(&inverted, value)
	one := Integer{}
	One(&one)
	ok = Add(destination, inverted, one)
	if bool(Is_Zero(value)) {
		return true
	}
	return Is_Negative(value) != Is_Negative(*destination)
}

// Subtract takes one value from another and reports whether the difference fits the width.
func Subtract(
	destination Integer_Handle, minuend Integer, subtrahend Integer,
) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "subtract.ok") }()
	Integer_Handle_Invariants(destination, "subtract.destination")
	Integer_Invariants(minuend, "subtract.minuend")
	Integer_Invariants(subtrahend, "subtract.subtrahend")
	difference := Integer{}
	borrow := bits.Borrow_In(0)
	for index := range LIMB_COUNT {
		total, next := bits.Subtract_64(
			bits.Word_64(limb_at(minuend.Limbs, Limb_Index(index))),
			bits.Subtrahend_64(limb_at(subtrahend.Limbs, Limb_Index(index))),
			borrow,
		)
		difference.Limbs = limbs_with(
			difference.Limbs, Limb_Index(index), Limb(total))
		borrow = bits.Borrow_In(next)
	}
	*destination = difference
	if Is_Negative(minuend) == Is_Negative(subtrahend) {
		return true
	}
	return Is_Negative(minuend) == Is_Negative(difference)
}

// Absolute reads the magnitude of a value. The most negative value has no magnitude inside the
// width, thus it alone reports an overflow.
func Absolute(destination Integer_Handle, value Integer) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "absolute.ok") }()
	Integer_Handle_Invariants(destination, "absolute.destination")
	Integer_Invariants(value, "absolute.value")
	if !bool(Is_Negative(value)) {
		*destination = value
		return true
	}
	return Negate(destination, value)
}

// Compare reports where the left value stands against the right one.
func Compare(left Integer, right Integer) (order Order) {
	defer func() { Order_Invariants(order, "compare.order") }()
	Integer_Invariants(left, "compare.left")
	Integer_Invariants(right, "compare.right")
	if Is_Negative(left) != Is_Negative(right) {
		if bool(Is_Negative(left)) {
			return ORDER_BEFORE
		}
		return ORDER_AFTER
	}
	for index := LIMB_INDEX_MAXIMUM; index >= LIMB_INDEX_MINIMUM; index-- {
		left_limb := limb_at(left.Limbs, Limb_Index(index))
		right_limb := limb_at(right.Limbs, Limb_Index(index))
		if left_limb == right_limb {
			continue
		}
		if left_limb < right_limb {
			return ORDER_BEFORE
		}
		return ORDER_AFTER
	}
	return ORDER_SAME
}

// And, Or, Exclusive_Or, and And_Not run limb by limb. Two's complement makes each exact for a
// negative operand without a case of its own.
func And(destination Integer_Handle, left Integer, right Integer) {
	Integer_Handle_Invariants(destination, "and.destination")
	Integer_Invariants(left, "and.left")
	Integer_Invariants(right, "and.right")
	result := Integer{}
	for index := range LIMB_COUNT {
		left_limb := limb_at(left.Limbs, Limb_Index(index))
		right_limb := limb_at(right.Limbs, Limb_Index(index))
		result.Limbs = limbs_with(result.Limbs, Limb_Index(index),
			left_limb&right_limb)
	}
	*destination = result
}

// Or joins the bits of two values.
func Or(destination Integer_Handle, left Integer, right Integer) {
	Integer_Handle_Invariants(destination, "or.destination")
	Integer_Invariants(left, "or.left")
	Integer_Invariants(right, "or.right")
	result := Integer{}
	for index := range LIMB_COUNT {
		left_limb := limb_at(left.Limbs, Limb_Index(index))
		right_limb := limb_at(right.Limbs, Limb_Index(index))
		result.Limbs = limbs_with(result.Limbs, Limb_Index(index),
			left_limb|right_limb)
	}
	*destination = result
}

// Exclusive_Or keeps the bits that stand in one value alone.
func Exclusive_Or(destination Integer_Handle, left Integer, right Integer) {
	Integer_Handle_Invariants(destination, "exclusive_or.destination")
	Integer_Invariants(left, "exclusive_or.left")
	Integer_Invariants(right, "exclusive_or.right")
	result := Integer{}
	for index := range LIMB_COUNT {
		left_limb := limb_at(left.Limbs, Limb_Index(index))
		right_limb := limb_at(right.Limbs, Limb_Index(index))
		result.Limbs = limbs_with(result.Limbs, Limb_Index(index),
			left_limb^right_limb)
	}
	*destination = result
}

// And_Not clears the bits of the left value that stand in the right one.
func And_Not(destination Integer_Handle, left Integer, right Integer) {
	Integer_Handle_Invariants(destination, "and_not.destination")
	Integer_Invariants(left, "and_not.left")
	Integer_Invariants(right, "and_not.right")
	result := Integer{}
	for index := range LIMB_COUNT {
		left_limb := limb_at(left.Limbs, Limb_Index(index))
		right_limb := limb_at(right.Limbs, Limb_Index(index))
		result.Limbs = limbs_with(result.Limbs, Limb_Index(index),
			left_limb&^right_limb)
	}
	*destination = result
}

// Bit reports whether the bit at one position stands. A position past the width reads the sign,
// which is what a two's complement value holds above its written bits.
func Bit(value Integer, position Shift_Count) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "bit.yes") }()
	Integer_Invariants(value, "bit.value")
	Shift_Count_Invariants(position, "bit.position")
	if int(position) >= BIT_COUNT_MAXIMUM {
		return Is_Negative(value)
	}
	limb := int(position) / LIMB_BIT_COUNT
	offset := int(position) % LIMB_BIT_COUNT
	return limb_at(value.Limbs, Limb_Index(limb))>>offset&1 == 1
}

// Bit_Size reads how many bits the magnitude of a value spans. Zero spans none.
func Bit_Size(value Integer) (count Bit_Count) {
	defer func() { Bit_Count_Invariants(count, "bit_size.count") }()
	Integer_Invariants(value, "bit_size.value")
	magnitude := Integer{}
	ok := Absolute(&magnitude, value)
	if !bool(ok) {
		return BIT_COUNT_MAXIMUM
	}
	for index := LIMB_INDEX_MAXIMUM; index >= LIMB_INDEX_MINIMUM; index-- {
		magnitude_limb := limb_at(magnitude.Limbs, Limb_Index(index))
		if magnitude_limb == 0 {
			continue
		}
		return Bit_Count(index*LIMB_BIT_COUNT) +
			Bit_Count(bits.Bit_Size_64(bits.Word_64(magnitude_limb)))
	}
	return BIT_COUNT_MINIMUM
}

// Shift_Left moves every bit up and reports whether a bit left the width.
func Shift_Left(
	destination Integer_Handle, value Integer, count Shift_Count,
) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "shift_left.ok") }()
	Integer_Handle_Invariants(destination, "shift_left.destination")
	Integer_Invariants(value, "shift_left.value")
	Shift_Count_Invariants(count, "shift_left.count")
	if bool(Is_Zero(value)) {
		*destination = value
		return true
	}
	if int(count) >= BIT_COUNT_MAXIMUM {
		Zero(destination)
		return false
	}
	result := value
	for range count {
		carry := Limb(0)
		for index := range LIMB_COUNT {
			limb := limb_at(result.Limbs, Limb_Index(index))
			next := limb >> SIGN_BIT_INDEX
			result.Limbs = limbs_with(
				result.Limbs, Limb_Index(index), limb<<1|carry)
			carry = next
		}
	}
	back := Integer{}
	Shift_Right(&back, result, count)
	*destination = result
	return Compare(back, value) == ORDER_SAME
}

// Shift_Right moves every bit down and carries the sign into the vacated bits, which is the
// arithmetic shift Go states for a signed value.
func Shift_Right(destination Integer_Handle, value Integer, count Shift_Count) {
	Integer_Handle_Invariants(destination, "shift_right.destination")
	Integer_Invariants(value, "shift_right.value")
	Shift_Count_Invariants(count, "shift_right.count")
	result := Integer{}
	fill := Limb(0)
	if bool(Is_Negative(value)) {
		fill = LIMB_MAXIMUM
	}
	if int(count) >= BIT_COUNT_MAXIMUM {
		for index := range LIMB_COUNT {
			result.Limbs = limbs_with(result.Limbs, Limb_Index(index), fill)
		}
		*destination = result
		return
	}
	result = value
	for range count {
		carry := fill << SIGN_BIT_INDEX
		for index := LIMB_INDEX_MAXIMUM; index >= LIMB_INDEX_MINIMUM; index-- {
			limb := limb_at(result.Limbs, Limb_Index(index))
			next := limb & 1
			result.Limbs = limbs_with(
				result.Limbs, Limb_Index(index), limb>>1|carry)
			carry = next << SIGN_BIT_INDEX
		}
	}
	*destination = result
}

// Multiply forms the product of two values and reports whether it fits the width. The magnitudes
// multiply and the sign follows, because a two's complement product of the written limbs would
// wrap without saying so.
func Multiply(
	destination Integer_Handle, multiplicand Integer, multiplier Integer,
) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "multiply.ok") }()
	Integer_Handle_Invariants(destination, "multiply.destination")
	Integer_Invariants(multiplicand, "multiply.multiplicand")
	Integer_Invariants(multiplier, "multiply.multiplier")
	negative := Is_Negative(multiplicand) != Is_Negative(multiplier)
	left := Integer{}
	left_ok := Absolute(&left, multiplicand)
	right := Integer{}
	right_ok := Absolute(&right, multiplier)
	if !bool(left_ok) {
		Zero(destination)
		return false
	}
	if !bool(right_ok) {
		Zero(destination)
		return false
	}
	ok = multiply_magnitude(destination, left, right)
	if !bool(ok) {
		Zero(destination)
		return false
	}
	if !bool(negative) {
		return !Is_Negative(*destination)
	}
	product := *destination
	return Negate(destination, product)
}

// Multiplies two magnitudes and reports whether the product fits the width. A carry out of the
// final limb, or a product that reaches the sign bit, has left the room a signed value has.
func multiply_magnitude(
	destination Integer_Handle, left Integer, right Integer,
) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "multiply_magnitude.ok") }()
	Integer_Handle_Invariants(destination, "multiply_magnitude.destination")
	Integer_Invariants(left, "multiply_magnitude.left")
	Integer_Invariants(right, "multiply_magnitude.right")
	product := Integer{}
	spill := Limb(0)
	for outer := range LIMB_COUNT {
		carry := Limb(0)
		for inner := range LIMB_COUNT {
			high, low := bits.Multiply_64(
				bits.Word_64(limb_at(left.Limbs, Limb_Index(outer))),
				bits.Multiplier_64(limb_at(right.Limbs, Limb_Index(inner))),
			)
			if outer+inner >= LIMB_COUNT {
				// Both words of the partial product fall outside the width, thus
				// either one standing is a product the width cannot hold.
				if low != 0 {
					Zero(destination)
					return false
				}
				if high != 0 {
					Zero(destination)
					return false
				}
				continue
			}
			total, first := bits.Add_64(
				bits.Word_64(limb_at(product.Limbs, Limb_Index(outer+inner))),
				bits.Addend_64(low), bits.Carry_In(0))
			rolled, second := bits.Add_64(
				bits.Word_64(total), bits.Addend_64(carry), bits.Carry_In(0))
			product.Limbs = limbs_with(
				product.Limbs, Limb_Index(outer+inner), Limb(rolled))
			carry = Limb(high) + Limb(first) + Limb(second)
		}
		spill = spill | carry
	}
	if spill != 0 {
		Zero(destination)
		return false
	}
	*destination = product
	return !Is_Negative(product)
}

// Divide truncates its quotient toward zero and gives the remainder the sign of the dividend,
// which is what Go states. A zero divisor is refused rather than trapped.
func Divide(
	quotient Integer_Handle,
	remainder Integer_Handle,
	dividend Integer,
	divisor Integer,
) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "divide.ok") }()
	Integer_Handle_Invariants(quotient, "divide.quotient")
	Integer_Handle_Invariants(remainder, "divide.remainder")
	Integer_Invariants(dividend, "divide.dividend")
	Integer_Invariants(divisor, "divide.divisor")
	if bool(Is_Zero(divisor)) {
		Zero(quotient)
		Zero(remainder)
		return false
	}
	left := Integer{}
	left_ok := Absolute(&left, dividend)
	right := Integer{}
	right_ok := Absolute(&right, divisor)
	if !bool(left_ok) {
		Zero(quotient)
		Zero(remainder)
		return false
	}
	if !bool(right_ok) {
		Zero(quotient)
		Zero(remainder)
		return false
	}
	divide_magnitude(quotient, remainder, left, right)
	if Is_Negative(dividend) != Is_Negative(divisor) {
		value := *quotient
		ok = Negate(quotient, value)
		if !bool(ok) {
			Zero(quotient)
			Zero(remainder)
			return false
		}
	}
	if bool(Is_Negative(dividend)) {
		value := *remainder
		ok = Negate(remainder, value)
		if !bool(ok) {
			Zero(quotient)
			Zero(remainder)
			return false
		}
	}
	return true
}

// Divides two magnitudes one bit at a time. The walk is a loop over the width rather than a
// word-at-a-time estimate, because a bit walk needs no correction step and reads as what it is.
func divide_magnitude(
	quotient Integer_Handle,
	remainder Integer_Handle,
	dividend Integer,
	divisor Integer,
) {
	Integer_Handle_Invariants(quotient, "divide_magnitude.quotient")
	Integer_Handle_Invariants(remainder, "divide_magnitude.remainder")
	Integer_Invariants(dividend, "divide_magnitude.dividend")
	Integer_Invariants(divisor, "divide_magnitude.divisor")
	source := dividend.Limbs
	dividend_limbs := [LIMB_COUNT]Limb{
		Limb(source.Limb_0), Limb(source.Limb_1), Limb(source.Limb_2), Limb(source.Limb_3),
		Limb(source.Limb_4), Limb(source.Limb_5), Limb(source.Limb_6), Limb(source.Limb_7),
	}
	source = divisor.Limbs
	divisor_limbs := [LIMB_COUNT]Limb{
		Limb(source.Limb_0), Limb(source.Limb_1), Limb(source.Limb_2), Limb(source.Limb_3),
		Limb(source.Limb_4), Limb(source.Limb_5), Limb(source.Limb_6), Limb(source.Limb_7),
	}
	var quotient_limbs, remainder_limbs [LIMB_COUNT]Limb
	for step := BIT_COUNT_MAXIMUM - 1; step >= 0; step-- {
		carry := Limb(0)
		for index := range LIMB_COUNT {
			next := remainder_limbs[index] >> SIGN_BIT_INDEX
			remainder_limbs[index] = remainder_limbs[index]<<1 | carry
			carry = next
		}
		limb := step / LIMB_BIT_COUNT
		if dividend_limbs[limb]>>uint(step%LIMB_BIT_COUNT)&1 == 1 {
			remainder_limbs[LIMB_INDEX_MINIMUM] |= 1
		}
		order := ORDER_SAME
		for index := LIMB_INDEX_MAXIMUM; index >= LIMB_INDEX_MINIMUM; index-- {
			if remainder_limbs[index] == divisor_limbs[index] {
				continue
			}
			if remainder_limbs[index] < divisor_limbs[index] {
				order = ORDER_BEFORE
			} else {
				order = ORDER_AFTER
			}
			break
		}
		if order == ORDER_BEFORE {
			continue
		}
		borrow := bits.Borrow_In(0)
		for index := range LIMB_COUNT {
			total, next := bits.Subtract_64(
				bits.Word_64(remainder_limbs[index]),
				bits.Subtrahend_64(divisor_limbs[index]),
				borrow,
			)
			remainder_limbs[index] = Limb(total)
			borrow = bits.Borrow_In(next)
		}
		quotient_limbs[limb] |= 1 << uint(step%LIMB_BIT_COUNT)
	}
	quotient.Limbs = Limbs{
		Limb_0: Limb_0(quotient_limbs[0]), Limb_1: Limb_1(quotient_limbs[1]),
		Limb_2: Limb_2(quotient_limbs[2]), Limb_3: Limb_3(quotient_limbs[3]),
		Limb_4: Limb_4(quotient_limbs[4]), Limb_5: Limb_5(quotient_limbs[5]),
		Limb_6: Limb_6(quotient_limbs[6]), Limb_7: Limb_7(quotient_limbs[7]),
	}
	remainder.Limbs = Limbs{
		Limb_0: Limb_0(remainder_limbs[0]), Limb_1: Limb_1(remainder_limbs[1]),
		Limb_2: Limb_2(remainder_limbs[2]), Limb_3: Limb_3(remainder_limbs[3]),
		Limb_4: Limb_4(remainder_limbs[4]), Limb_5: Limb_5(remainder_limbs[5]),
		Limb_6: Limb_6(remainder_limbs[6]), Limb_7: Limb_7(remainder_limbs[7]),
	}
}

// Greatest_Common_Divisor runs Euclid over the magnitudes. The result is never negative, and the
// divisor of zero and zero is zero.
func Greatest_Common_Divisor(
	destination Integer_Handle, left Integer, right Integer,
) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "greatest_common_divisor.ok") }()
	Integer_Handle_Invariants(destination, "greatest_common_divisor.destination")
	Integer_Invariants(left, "greatest_common_divisor.left")
	Integer_Invariants(right, "greatest_common_divisor.right")
	first := Integer{}
	first_ok := Absolute(&first, left)
	second := Integer{}
	second_ok := Absolute(&second, right)
	if !bool(first_ok) {
		Zero(destination)
		return false
	}
	if !bool(second_ok) {
		Zero(destination)
		return false
	}
	for range BIT_COUNT_MAXIMUM * 2 {
		if bool(Is_Zero(second)) {
			*destination = first
			return true
		}
		quotient := Integer{}
		rest := Integer{}
		divide_magnitude(&quotient, &rest, first, second)
		first = second
		second = rest
	}
	Zero(destination)
	return false
}

// From_Text reads a Go integer literal in base two, eight, ten, or sixteen. An underscore groups
// digits and carries no value. A literal that spells no digit, or one past the width, is refused.
func From_Text(destination Integer_Handle, text Text) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "from_text.ok") }()
	Integer_Handle_Invariants(destination, "from_text.destination")
	Text_Invariants(text, "from_text.text")
	base, rest := read_base(text)
	if len(rest) == 0 {
		Zero(destination)
		return false
	}
	step := Integer{}
	From_Int_64(&step, Int_64(base))
	result := Integer{}
	digits := 0
	for index := range len(rest) {
		if rest[index] == '_' {
			continue
		}
		value := digit_value(Digit_Byte(rest[index]))
		if int(value) == DIGIT_VALUE_ABSENT {
			Zero(destination)
			return false
		}
		if int(value) >= int(base) {
			Zero(destination)
			return false
		}
		digits++
		scaled := Integer{}
		scaled_ok := Multiply(&scaled, result, step)
		if !bool(scaled_ok) {
			Zero(destination)
			return false
		}
		digit := Integer{}
		From_Int_64(&digit, Int_64(value))
		ok = Add(&result, scaled, digit)
		if !bool(ok) {
			Zero(destination)
			return false
		}
	}
	if digits == 0 {
		Zero(destination)
		return false
	}
	*destination = result
	return true
}

// Reads the base a literal names and returns the digits behind the prefix.
func read_base(text Text) (base Base, rest Text) {
	defer func() {
		Base_Invariants(base, "read_base.base")
		Text_Invariants(rest, "read_base.rest")
	}()
	Text_Invariants(text, "read_base.text")
	if len(text) < 2 {
		return DECIMAL_BASE, text
	}
	if text[0] != '0' {
		return DECIMAL_BASE, text
	}
	switch text[1] {
	case 'b', 'B':
		return BASE_BINARY, text[2:]
	case 'o', 'O':
		return BASE_OCTAL, text[2:]
	case 'x', 'X':
		return BASE_HEXADECIMAL, text[2:]
	}
	return BASE_OCTAL, text[1:]
}

// Reads the value one digit byte spells, or DIGIT_VALUE_ABSENT when it spells none.
func digit_value(value Digit_Byte) (digit Digit_Value) {
	defer func() { Digit_Value_Invariants(digit, "digit_value.digit") }()
	Digit_Byte_Invariants(value, "digit_value.value")
	switch {
	case value >= '0' && value <= '9':
		return Digit_Value(value - '0')
	case value >= 'a' && value <= 'f':
		return Digit_Value(value-'a') + DECIMAL_BASE
	case value >= 'A' && value <= 'F':
		return Digit_Value(value-'A') + DECIMAL_BASE
	}
	return DIGIT_VALUE_ABSENT
}

// Into_Text writes the decimal form of a value into caller storage and returns the byte count. A
// destination that cannot hold the form is refused rather than filled part way.
func Into_Text(destination Digits, value Integer) (count Digit_Count) {
	defer func() { Digit_Count_Invariants(count, "into_text.count") }()
	Digits_Invariants(destination, "into_text.destination")
	Integer_Invariants(value, "into_text.value")
	var storage [DIGIT_COUNT_MAXIMUM]byte
	magnitude := Integer{}
	ok := Absolute(&magnitude, value)
	if !bool(ok) {
		return DIGIT_COUNT_MINIMUM
	}
	step := Integer{}
	From_Int_64(&step, DECIMAL_BASE)
	written := 0
	for range DIGIT_COUNT_MAXIMUM {
		if bool(Is_Zero(magnitude)) {
			break
		}
		quotient := Integer{}
		rest := Integer{}
		divide_magnitude(&quotient, &rest, magnitude, step)
		magnitude = quotient
		storage[written] = byte(rest.Limbs.Limb_0) + '0'
		written++
	}
	if written == 0 {
		storage[0] = '0'
		written = 1
	}
	if bool(Is_Negative(value)) {
		storage[written] = '-'
		written++
	}
	if len(destination) < written {
		return DIGIT_COUNT_MINIMUM
	}
	for index := range written {
		destination[index] = storage[written-1-index]
	}
	return Digit_Count(written)
}
