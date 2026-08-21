package integer

import (
	"testing"

	"local/james-orcales/shared/math/bits"
)

// Test_Internal_Invariant_Domains feeds opaque storage edges through production entry points,
// because ordinary arithmetic cannot make constant constructors return every machine word.
func Test_Internal_Invariant_Domains(_ *testing.T) {
	for _, word := range [...]uint64{
		bits.WORD_64_MINIMUM,
		1,
		2,
		bits.WORD_64_MAXIMUM,
	} {
		value := integer_filled(word)
		destination := value

		Zero(&destination)
		destination = value
		One(&destination)
		destination = value
		From_Int_64(&destination, Int_64(word))
		To_Int_64(value)
		Is_Negative(value)
		Is_Zero(value)
		Sign(value)

		destination = value
		Add(&destination, value, value)
		destination = value
		Not(&destination, value)
		destination = value
		Negate(&destination, value)
		destination = value
		Subtract(&destination, value, value)
		destination = value
		Absolute(&destination, value)
		Compare(value, value)

		destination = value
		And(&destination, value, value)
		destination = value
		Or(&destination, value, value)
		destination = value
		Exclusive_Or(&destination, value, value)
		destination = value
		And_Not(&destination, value, value)
		Bit(value, 0)
		Bit_Size(value)

		destination = value
		Shift_Left(&destination, value, 0)
		destination = value
		Shift_Right(&destination, value, 0)
		destination = value
		Multiply(&destination, value, value)
		destination = value
		multiply_magnitude(&destination, value, value)

		quotient := value
		remainder := value
		Divide(&quotient, &remainder, value, value)
		quotient = value
		remainder = value
		divide_magnitude(&quotient, &remainder, value, value)
		destination = value
		Greatest_Common_Divisor(&destination, value, value)
		destination = value
		From_Text(&destination, "1")
		var text [DIGIT_COUNT_MAXIMUM]byte
		Into_Text(text[:], value)
	}
}

func integer_filled(word uint64) (value Integer) {
	value.Limbs = Limbs{
		Limb_0: Limb_0(word), Limb_1: Limb_1(word),
		Limb_2: Limb_2(word), Limb_3: Limb_3(word),
		Limb_4: Limb_4(word), Limb_5: Limb_5(word),
		Limb_6: Limb_6(word), Limb_7: Limb_7(word),
	}
	return value
}
