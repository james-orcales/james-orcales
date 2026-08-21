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

// Integer is a two's complement value of fixed width, least significant limb first.
type Integer struct {
	// Limbs holds the value, least significant limb first. The top bit of the final limb
	// carries the sign, thus a negative value needs no field of its own.
	Limbs [LIMB_COUNT]Limb
}

// Integer_Invariants states the width every value holds.
func Integer_Invariants(value Integer, namespace aver.Namespace) {
	aver.Always(
		len(value.Limbs) == LIMB_COUNT,
		"An integer holds one limb for every piece of its width.",
	)
}

// Zero is the value every magnitude starts from.
func Zero() (result Integer) {
	defer func() { Integer_Invariants(result, "zero.result") }()
	return Integer{Limbs: [LIMB_COUNT]Limb{}}
}

// One is the unit every count steps by.
func One() (result Integer) {
	defer func() { Integer_Invariants(result, "one.result") }()
	result.Limbs[LIMB_INDEX_MINIMUM] = 1
	return result
}

// From_Int_64 lifts a machine integer, carrying its sign into every limb above it.
func From_Int_64(value Int_64) (result Integer) {
	defer func() { Integer_Invariants(result, "from_int_64.result") }()
	Int_64_Invariants(value, "from_int_64.value")
	fill := Limb(0)
	if value < 0 {
		fill = LIMB_MAXIMUM
	}
	for index := range LIMB_COUNT {
		result.Limbs[index] = fill
	}
	result.Limbs[LIMB_INDEX_MINIMUM] = Limb(uint64(value))
	return result
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
		if value.Limbs[index] != fill {
			return 0, false
		}
	}
	low := value.Limbs[LIMB_INDEX_MINIMUM]
	if bool(Is_Negative(value)) != (low>>SIGN_BIT_INDEX == 1) {
		return 0, false
	}
	return Int_64(int64(low)), true
}

// Is_Negative reports whether the sign bit of a value stands.
func Is_Negative(value Integer) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_negative.yes") }()
	Integer_Invariants(value, "is_negative.value")
	return value.Limbs[LIMB_INDEX_MAXIMUM]>>SIGN_BIT_INDEX == 1
}

// Is_Zero reports whether every limb of a value is empty.
func Is_Zero(value Integer) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_zero.yes") }()
	Integer_Invariants(value, "is_zero.value")
	for index := range LIMB_COUNT {
		if value.Limbs[index] != 0 {
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
func Add(augend Integer, addend Integer) (sum Integer, ok Boolean) {
	defer func() {
		Integer_Invariants(sum, "add.sum")
		Boolean_Invariants(ok, "add.ok")
	}()
	Integer_Invariants(augend, "add.augend")
	Integer_Invariants(addend, "add.addend")
	carry := bits.Carry_In(0)
	for index := range LIMB_COUNT {
		total, next := bits.Add_64(
			bits.Word_64(augend.Limbs[index]),
			bits.Addend_64(addend.Limbs[index]),
			carry,
		)
		sum.Limbs[index] = Limb(total)
		carry = bits.Carry_In(next)
	}
	if Is_Negative(augend) != Is_Negative(addend) {
		return sum, true
	}
	return sum, Is_Negative(augend) == Is_Negative(sum)
}

// Not flips every bit of a value.
func Not(value Integer) (result Integer) {
	defer func() { Integer_Invariants(result, "not.result") }()
	Integer_Invariants(value, "not.value")
	for index := range LIMB_COUNT {
		result.Limbs[index] = ^value.Limbs[index]
	}
	return result
}

// Negate reverses the sign of a value. The most negative value has no positive twin inside the
// width, thus it alone reports an overflow.
func Negate(value Integer) (result Integer, ok Boolean) {
	defer func() {
		Integer_Invariants(result, "negate.result")
		Boolean_Invariants(ok, "negate.ok")
	}()
	Integer_Invariants(value, "negate.value")
	result, ok = Add(Not(value), One())
	if bool(Is_Zero(value)) {
		return result, true
	}
	return result, Is_Negative(value) != Is_Negative(result)
}

// Subtract takes one value from another and reports whether the difference fits the width.
func Subtract(minuend Integer, subtrahend Integer) (difference Integer, ok Boolean) {
	defer func() {
		Integer_Invariants(difference, "subtract.difference")
		Boolean_Invariants(ok, "subtract.ok")
	}()
	Integer_Invariants(minuend, "subtract.minuend")
	Integer_Invariants(subtrahend, "subtract.subtrahend")
	borrow := bits.Borrow_In(0)
	for index := range LIMB_COUNT {
		total, next := bits.Subtract_64(
			bits.Word_64(minuend.Limbs[index]),
			bits.Subtrahend_64(subtrahend.Limbs[index]),
			borrow,
		)
		difference.Limbs[index] = Limb(total)
		borrow = bits.Borrow_In(next)
	}
	if Is_Negative(minuend) == Is_Negative(subtrahend) {
		return difference, true
	}
	return difference, Is_Negative(minuend) == Is_Negative(difference)
}

// Absolute reads the magnitude of a value. The most negative value has no magnitude inside the
// width, thus it alone reports an overflow.
func Absolute(value Integer) (result Integer, ok Boolean) {
	defer func() {
		Integer_Invariants(result, "absolute.result")
		Boolean_Invariants(ok, "absolute.ok")
	}()
	Integer_Invariants(value, "absolute.value")
	if !bool(Is_Negative(value)) {
		return value, true
	}
	return Negate(value)
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
		if left.Limbs[index] == right.Limbs[index] {
			continue
		}
		if left.Limbs[index] < right.Limbs[index] {
			return ORDER_BEFORE
		}
		return ORDER_AFTER
	}
	return ORDER_SAME
}

// And, Or, Exclusive_Or, and And_Not run limb by limb. Two's complement makes each exact for a
// negative operand without a case of its own.
func And(left Integer, right Integer) (result Integer) {
	defer func() { Integer_Invariants(result, "and.result") }()
	Integer_Invariants(left, "and.left")
	Integer_Invariants(right, "and.right")
	for index := range LIMB_COUNT {
		result.Limbs[index] = left.Limbs[index] & right.Limbs[index]
	}
	return result
}

// Or joins the bits of two values.
func Or(left Integer, right Integer) (result Integer) {
	defer func() { Integer_Invariants(result, "or.result") }()
	Integer_Invariants(left, "or.left")
	Integer_Invariants(right, "or.right")
	for index := range LIMB_COUNT {
		result.Limbs[index] = left.Limbs[index] | right.Limbs[index]
	}
	return result
}

// Exclusive_Or keeps the bits that stand in one value alone.
func Exclusive_Or(left Integer, right Integer) (result Integer) {
	defer func() { Integer_Invariants(result, "exclusive_or.result") }()
	Integer_Invariants(left, "exclusive_or.left")
	Integer_Invariants(right, "exclusive_or.right")
	for index := range LIMB_COUNT {
		result.Limbs[index] = left.Limbs[index] ^ right.Limbs[index]
	}
	return result
}

// And_Not clears the bits of the left value that stand in the right one.
func And_Not(left Integer, right Integer) (result Integer) {
	defer func() { Integer_Invariants(result, "and_not.result") }()
	Integer_Invariants(left, "and_not.left")
	Integer_Invariants(right, "and_not.right")
	for index := range LIMB_COUNT {
		result.Limbs[index] = left.Limbs[index] &^ right.Limbs[index]
	}
	return result
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
	return value.Limbs[limb]>>offset&1 == 1
}

// Bit_Size reads how many bits the magnitude of a value spans. Zero spans none.
func Bit_Size(value Integer) (count Bit_Count) {
	defer func() { Bit_Count_Invariants(count, "bit_size.count") }()
	Integer_Invariants(value, "bit_size.value")
	magnitude, ok := Absolute(value)
	if !bool(ok) {
		return BIT_COUNT_MAXIMUM
	}
	for index := LIMB_INDEX_MAXIMUM; index >= LIMB_INDEX_MINIMUM; index-- {
		if magnitude.Limbs[index] == 0 {
			continue
		}
		return Bit_Count(index*LIMB_BIT_COUNT) +
			Bit_Count(bits.Bit_Size_64(bits.Word_64(magnitude.Limbs[index])))
	}
	return BIT_COUNT_MINIMUM
}

// Shift_Left moves every bit up and reports whether a bit left the width.
func Shift_Left(value Integer, count Shift_Count) (result Integer, ok Boolean) {
	defer func() {
		Integer_Invariants(result, "shift_left.result")
		Boolean_Invariants(ok, "shift_left.ok")
	}()
	Integer_Invariants(value, "shift_left.value")
	Shift_Count_Invariants(count, "shift_left.count")
	if bool(Is_Zero(value)) {
		return value, true
	}
	if int(count) >= BIT_COUNT_MAXIMUM {
		return Zero(), false
	}
	result = value
	for range count {
		carry := Limb(0)
		for index := range LIMB_COUNT {
			next := result.Limbs[index] >> SIGN_BIT_INDEX
			result.Limbs[index] = result.Limbs[index]<<1 | carry
			carry = next
		}
	}
	back := Shift_Right(result, count)
	return result, Compare(back, value) == ORDER_SAME
}

// Shift_Right moves every bit down and carries the sign into the vacated bits, which is the
// arithmetic shift Go states for a signed value.
func Shift_Right(value Integer, count Shift_Count) (result Integer) {
	defer func() { Integer_Invariants(result, "shift_right.result") }()
	Integer_Invariants(value, "shift_right.value")
	Shift_Count_Invariants(count, "shift_right.count")
	fill := Limb(0)
	if bool(Is_Negative(value)) {
		fill = LIMB_MAXIMUM
	}
	if int(count) >= BIT_COUNT_MAXIMUM {
		for index := range LIMB_COUNT {
			result.Limbs[index] = fill
		}
		return result
	}
	result = value
	for range count {
		carry := fill << SIGN_BIT_INDEX
		for index := LIMB_INDEX_MAXIMUM; index >= LIMB_INDEX_MINIMUM; index-- {
			next := result.Limbs[index] & 1
			result.Limbs[index] = result.Limbs[index]>>1 | carry
			carry = next << SIGN_BIT_INDEX
		}
	}
	return result
}

// Multiply forms the product of two values and reports whether it fits the width. The magnitudes
// multiply and the sign follows, because a two's complement product of the written limbs would
// wrap without saying so.
func Multiply(multiplicand Integer, multiplier Integer) (product Integer, ok Boolean) {
	defer func() {
		Integer_Invariants(product, "multiply.product")
		Boolean_Invariants(ok, "multiply.ok")
	}()
	Integer_Invariants(multiplicand, "multiply.multiplicand")
	Integer_Invariants(multiplier, "multiply.multiplier")
	negative := Is_Negative(multiplicand) != Is_Negative(multiplier)
	left, left_ok := Absolute(multiplicand)
	right, right_ok := Absolute(multiplier)
	if !bool(left_ok) {
		return Zero(), false
	}
	if !bool(right_ok) {
		return Zero(), false
	}
	product, ok = multiply_magnitude(left, right)
	if !bool(ok) {
		return Zero(), false
	}
	if !bool(negative) {
		return product, !Is_Negative(product)
	}
	product, ok = Negate(product)
	return product, ok
}

// Multiplies two magnitudes and reports whether the product fits the width. A carry out of the
// final limb, or a product that reaches the sign bit, has left the room a signed value has.
func multiply_magnitude(left Integer, right Integer) (product Integer, ok Boolean) {
	defer func() {
		Integer_Invariants(product, "multiply_magnitude.product")
		Boolean_Invariants(ok, "multiply_magnitude.ok")
	}()
	Integer_Invariants(left, "multiply_magnitude.left")
	Integer_Invariants(right, "multiply_magnitude.right")
	spill := Limb(0)
	for outer := range LIMB_COUNT {
		carry := Limb(0)
		for inner := range LIMB_COUNT {
			high, low := bits.Multiply_64(
				bits.Word_64(left.Limbs[outer]),
				bits.Multiplier_64(right.Limbs[inner]),
			)
			if outer+inner >= LIMB_COUNT {
				// Both words of the partial product fall outside the width, thus
				// either one standing is a product the width cannot hold.
				if low != 0 {
					return Zero(), false
				}
				if high != 0 {
					return Zero(), false
				}
				continue
			}
			total, first := bits.Add_64(
				bits.Word_64(product.Limbs[outer+inner]),
				bits.Addend_64(low), bits.Carry_In(0))
			rolled, second := bits.Add_64(
				bits.Word_64(total), bits.Addend_64(carry), bits.Carry_In(0))
			product.Limbs[outer+inner] = Limb(rolled)
			carry = Limb(high) + Limb(first) + Limb(second)
		}
		spill = spill | carry
	}
	if spill != 0 {
		return Zero(), false
	}
	return product, !Is_Negative(product)
}

// Divide truncates its quotient toward zero and gives the remainder the sign of the dividend,
// which is what Go states. A zero divisor is refused rather than trapped.
func Divide(
	dividend Integer, divisor Integer,
) (quotient Integer, remainder Integer, ok Boolean) {
	defer func() {
		Integer_Invariants(quotient, "divide.quotient")
		Integer_Invariants(remainder, "divide.remainder")
		Boolean_Invariants(ok, "divide.ok")
	}()
	Integer_Invariants(dividend, "divide.dividend")
	Integer_Invariants(divisor, "divide.divisor")
	if bool(Is_Zero(divisor)) {
		return Zero(), Zero(), false
	}
	left, left_ok := Absolute(dividend)
	right, right_ok := Absolute(divisor)
	if !bool(left_ok) {
		return Zero(), Zero(), false
	}
	if !bool(right_ok) {
		return Zero(), Zero(), false
	}
	quotient, remainder = divide_magnitude(left, right)
	if Is_Negative(dividend) != Is_Negative(divisor) {
		quotient, ok = Negate(quotient)
		if !bool(ok) {
			return Zero(), Zero(), false
		}
	}
	if bool(Is_Negative(dividend)) {
		remainder, ok = Negate(remainder)
		if !bool(ok) {
			return Zero(), Zero(), false
		}
	}
	return quotient, remainder, true
}

// Divides two magnitudes one bit at a time. The walk is a loop over the width rather than a
// word-at-a-time estimate, because a bit walk needs no correction step and reads as what it is.
func divide_magnitude(
	dividend Integer, divisor Integer,
) (quotient Integer, remainder Integer) {
	defer func() {
		Integer_Invariants(quotient, "divide_magnitude.quotient")
		Integer_Invariants(remainder, "divide_magnitude.remainder")
	}()
	Integer_Invariants(dividend, "divide_magnitude.dividend")
	Integer_Invariants(divisor, "divide_magnitude.divisor")
	for step := BIT_COUNT_MAXIMUM - 1; step >= 0; step-- {
		carry := Limb(0)
		for index := range LIMB_COUNT {
			next := remainder.Limbs[index] >> SIGN_BIT_INDEX
			remainder.Limbs[index] = remainder.Limbs[index]<<1 | carry
			carry = next
		}
		if bool(Bit(dividend, Shift_Count(step))) {
			low := remainder.Limbs[LIMB_INDEX_MINIMUM]
			remainder.Limbs[LIMB_INDEX_MINIMUM] = low | 1
		}
		if Compare(remainder, divisor) == ORDER_BEFORE {
			continue
		}
		remainder, _ = Subtract(remainder, divisor)
		limb := step / LIMB_BIT_COUNT
		quotient.Limbs[limb] = quotient.Limbs[limb] | 1<<(step%LIMB_BIT_COUNT)
	}
	return quotient, remainder
}

// Greatest_Common_Divisor runs Euclid over the magnitudes. The result is never negative, and the
// divisor of zero and zero is zero.
func Greatest_Common_Divisor(left Integer, right Integer) (result Integer, ok Boolean) {
	defer func() {
		Integer_Invariants(result, "greatest_common_divisor.result")
		Boolean_Invariants(ok, "greatest_common_divisor.ok")
	}()
	Integer_Invariants(left, "greatest_common_divisor.left")
	Integer_Invariants(right, "greatest_common_divisor.right")
	first, first_ok := Absolute(left)
	second, second_ok := Absolute(right)
	if !bool(first_ok) {
		return Zero(), false
	}
	if !bool(second_ok) {
		return Zero(), false
	}
	for range BIT_COUNT_MAXIMUM * 2 {
		if bool(Is_Zero(second)) {
			return first, true
		}
		_, rest := divide_magnitude(first, second)
		first = second
		second = rest
	}
	return Zero(), false
}

// From_Text reads a Go integer literal in base two, eight, ten, or sixteen. An underscore groups
// digits and carries no value. A literal that spells no digit, or one past the width, is refused.
func From_Text(text Text) (result Integer, ok Boolean) {
	defer func() {
		Integer_Invariants(result, "from_text.result")
		Boolean_Invariants(ok, "from_text.ok")
	}()
	Text_Invariants(text, "from_text.text")
	base, rest := read_base(text)
	if len(rest) == 0 {
		return Zero(), false
	}
	step := From_Int_64(Int_64(base))
	digits := 0
	for index := range len(rest) {
		if rest[index] == '_' {
			continue
		}
		value := digit_value(Digit_Byte(rest[index]))
		if int(value) == DIGIT_VALUE_ABSENT {
			return Zero(), false
		}
		if int(value) >= int(base) {
			return Zero(), false
		}
		digits++
		scaled, scaled_ok := Multiply(result, step)
		if !bool(scaled_ok) {
			return Zero(), false
		}
		result, ok = Add(scaled, From_Int_64(Int_64(value)))
		if !bool(ok) {
			return Zero(), false
		}
	}
	if digits == 0 {
		return Zero(), false
	}
	return result, true
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
	magnitude, ok := Absolute(value)
	if !bool(ok) {
		return DIGIT_COUNT_MINIMUM
	}
	step := From_Int_64(DECIMAL_BASE)
	written := 0
	for range DIGIT_COUNT_MAXIMUM {
		if bool(Is_Zero(magnitude)) {
			break
		}
		rest := Integer{}
		magnitude, rest = divide_magnitude(magnitude, step)
		storage[written] = byte(rest.Limbs[LIMB_INDEX_MINIMUM]) + '0'
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
