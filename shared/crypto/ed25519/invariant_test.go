package ed25519

import (
	"testing"

	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
)

// Test_Internal_Point_Invariant_Domains reaches raw points forbidden by public decoding.
func Test_Internal_Point_Invariant_Domains(_ *testing.T) {
	var encoding_storage [PUBLIC_KEY_SIZE]byte
	encoding := Point_Encoding(encoding_storage[:])
	for _, word := range [...]uint64{
		bits.WORD_64_MINIMUM,
		binary.UINT_8_SIZE,
		binary.UINT_16_SIZE,
		bits.WORD_64_MAXIMUM,
	} {
		point := point_filled(word)
		scalar := scalar_filled(word)
		destination := point_filled(word)

		point_scalar_base_multiply(&destination, &scalar)
		destination = point_filled(word)
		point_scalar_multiply(&destination, &point, &scalar)
		destination = point_filled(word)
		point_add(&destination, &point, &point)
		destination = point_filled(word)
		point_double(&destination, &point)
		destination = point_filled(word)
		point_negate(&destination, &point)
		destination = point_filled(word)
		point_select(&destination, &point, &point, Decision(word&binary.UINT_8_SIZE))
		point_encode(encoding, &point)
		destination = point_filled(word)
		point_decode(&destination, encoding)
		point_has_small_order(&point)
		point_equal(&point, &point)
		destination = point_filled(word)
		point_identity(&destination)
		destination = point_filled(word)
		point_base(&destination)
	}
}

// Test_Internal_Field_Invariant_Domains reaches every opaque machine-word edge.
func Test_Internal_Field_Invariant_Domains(_ *testing.T) {
	var point_encoding_storage [PUBLIC_KEY_SIZE]byte
	point_encoding := Point_Encoding(point_encoding_storage[:])
	for _, word := range [...]uint64{
		bits.WORD_64_MINIMUM,
		binary.UINT_8_SIZE,
		binary.UINT_16_SIZE,
		bits.WORD_64_MAXIMUM,
	} {
		field := field_filled(word)
		field_result := field_filled(word)
		field_limbs(&field)
		field_multiply(&field_result, &field, &field)
		field_result = field_filled(word)
		field_square(&field_result, &field)
		field_result = field_filled(word)
		field_add(&field_result, &field, &field)
		field_result = field_filled(word)
		field_subtract(&field_result, &field, &field)
		field_result = field_filled(word)
		field_negate(&field_result, &field)
		field_result = field_filled(word)
		field_inverse(&field_result, &field)
		field_result = field_filled(word)
		field_square_root(&field_result, &field)
		field_result = field_filled(word)
		field_power(&field_result, &field, &field)
		field_result = field_filled(word)
		field_select(
			&field_result, &field, &field, Decision(word&binary.UINT_8_SIZE),
		)
		field_equal(&field, &field)
		field_is_zero(&field)
		field_result = field_filled(word)
		field_decode(&field_result, point_encoding)
		field_encode(point_encoding, &field)

		field_result = field_filled(word)
		field_modulus(&field_result)
		field_result = field_filled(word)
		field_one(&field_result)
		field_result = field_filled(word)
		field_d(&field_result)
		field_result = field_filled(word)
		field_twice_d(&field_result)
		field_result = field_filled(word)
		field_inverse_exponent(&field_result)
		field_result = field_filled(word)
		field_square_root_exponent(&field_result)
		field_result = field_filled(word)
		field_square_root_minus_one(&field_result)
		test_scalar_invariant_domains(word)
	}
}

func test_scalar_invariant_domains(word uint64) {
	var scalar_encoding_storage [SEED_SIZE]byte
	scalar_encoding := Scalar_Encoding(scalar_encoding_storage[:])
	var wide_storage [WIDE_SCALAR_SIZE]byte
	wide := Wide_Source(wide_storage[:])
	scalar := scalar_filled(word)
	scalar_result := scalar_filled(word)
	scalar_limbs(&scalar)
	scalar_reduce_wide(&scalar_result, wide)
	scalar_result = scalar_filled(word)
	scalar_decode_canonical(&scalar_result, scalar_encoding)
	scalar_encode(scalar_encoding, &scalar)
	scalar_result = scalar_filled(word)
	scalar_add(&scalar_result, &scalar, &scalar)
	scalar_result = scalar_filled(word)
	scalar_multiply(&scalar_result, &scalar, &scalar)

	limbs := Limbs(field_filled(word))
	limbs_result := limbs
	limbs_add(&limbs_result, &limbs, &limbs)
	limbs_result = limbs
	limbs_subtract(&limbs_result, &limbs, &limbs)
	limbs_result = limbs
	limbs_select(
		&limbs_result, &limbs, &limbs, Decision(word&binary.UINT_8_SIZE),
	)

	scalar_result = scalar_filled(word)
	scalar_order(&scalar_result)
	scalar_result = scalar_filled(word)
	scalar_one(&scalar_result)
}

// Test_Internal_Key_Invariant_Domains reaches every owned seed word through public operations.
func Test_Internal_Key_Invariant_Domains(_ *testing.T) {
	var public_storage [PUBLIC_KEY_SIZE]byte
	public_key := Public_Key(public_storage[:])
	var signature_storage [SIGNATURE_SIZE]byte
	signature := Signature(signature_storage[:])
	for _, word := range [...]uint64{
		bits.WORD_64_MINIMUM,
		binary.UINT_8_SIZE,
		binary.UINT_16_SIZE,
		bits.WORD_64_MAXIMUM,
	} {
		private_key := private_key_filled(word)
		seed := seed_filled(word)
		Private_Key_From_Seed(&private_key, seed)
		scalar := scalar_filled(word)
		var prefix_storage [SEED_SIZE]byte
		private_key_expand(&scalar, Prefix(prefix_storage[:]), &private_key)
		Public_Key_From_Private(public_key, &private_key)
		Sign(signature, &private_key, nil)
	}
}

func field_filled(word uint64) (result Field) {
	result = Field{
		Limb_0: Limb_0(word),
		Limb_1: Limb_1(word),
		Limb_2: Limb_2(word),
		Limb_3: Limb_3(word),
	}
	return result
}

func scalar_filled(word uint64) (result Scalar) {
	return Scalar(field_filled(word))
}

func point_filled(word uint64) (result Point) {
	limbs := Limb_Storage(field_filled(word))
	result = Point{
		X: X_Field(limbs),
		Y: Y_Field(limbs),
		Z: Z_Field(limbs),
		T: T_Field(limbs),
	}
	return result
}

func private_key_filled(word uint64) (result Private_Key) {
	return Private_Key(field_filled(word))
}

func seed_filled(word uint64) (seed Seed) {
	var storage [SEED_SIZE]byte
	for index := range storage {
		shift := uint(index%binary.UINT_64_SIZE) * bits.BIT_COUNT_8_MAXIMUM
		storage[index] = byte(word >> shift)
	}
	return Seed(storage[:])
}
