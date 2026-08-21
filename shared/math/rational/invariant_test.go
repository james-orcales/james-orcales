package rational

import (
	"testing"

	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/math/integer"
)

// Test_Internal_Invariant_Domains feeds storage edges through production entry points, because
// normalised ratios cannot preserve every machine word in every limb.
func Test_Internal_Invariant_Domains(_ *testing.T) {
	for _, word := range [...]uint64{
		bits.WORD_64_MINIMUM,
		1,
		2,
		bits.WORD_64_MAXIMUM,
	} {
		value := rational_filled(word)
		destination := value

		Zero(&destination)
		destination = value
		One(&destination)
		destination = value
		From_Integer(&destination, value.Numerator)
		destination = value
		From_Ratio(&destination, value.Numerator, value.Denominator)

		numerator := value.Numerator
		Whole(&numerator, value)
		Is_Whole(value)
		Is_Zero(value)
		Sign(value)

		destination = value
		Add(&destination, value, value)
		destination = value
		Negate(&destination, value)
		destination = value
		Subtract(&destination, value, value)
		destination = value
		Absolute(&destination, value)
		destination = value
		Multiply(&destination, value, value)
		destination = value
		Divide(&destination, value, value)
		Compare(value, value)

		var text [TEXT_SIZE_MAXIMUM]byte
		Into_Text(text[:], value)
	}
}

func rational_filled(word uint64) (value Rational) {
	limbs := integer.Limbs{
		Limb_0: integer.Limb_0(word), Limb_1: integer.Limb_1(word),
		Limb_2: integer.Limb_2(word), Limb_3: integer.Limb_3(word),
		Limb_4: integer.Limb_4(word), Limb_5: integer.Limb_5(word),
		Limb_6: integer.Limb_6(word), Limb_7: integer.Limb_7(word),
	}
	value.Numerator = Numerator(limbs)
	value.Denominator = Denominator(limbs)
	return value
}
