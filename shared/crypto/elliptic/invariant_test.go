package elliptic

import (
	"testing"

	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
)

// Test_Internal_Point_Invariant_Domains reaches raw points forbidden by public decoding.
func Test_Internal_Point_Invariant_Domains(_ *testing.T) {
	for _, word := range [...]uint64{
		bits.WORD_64_MINIMUM,
		binary.UINT_8_SIZE,
		binary.UINT_16_SIZE,
		bits.WORD_64_MAXIMUM,
	} {
		edge := limbs_filled(word)
		point := point_filled(word)

		x_limbs(&point.X)
		y_limbs(&point.Y)
		z_limbs(&point.Z)
		x_coordinate(&point.X, edge)
		y_coordinate(&point.Y, edge)
		z_coordinate(&point.Z, edge)

		destination := point_filled(word)
		Point_Identity(&destination)
		destination = point_filled(word)
		Point_Generator(&destination)
		destination = point_filled(word)
		ignore_panic(func() { Point_Set_Bytes(&destination, nil) })
		destination = point_filled(word)
		ignore_panic(func() { Point_Bytes_Into(nil, &destination, ENCODING_COMPRESSED) })
		destination = point_filled(word)
		ignore_panic(func() { Point_Add(&destination, &point, &point) })
		destination = point_filled(word)
		ignore_panic(func() { Point_Double(&destination, &point) })
		destination = point_filled(word)
		ignore_panic(func() { Point_Scalar_Multiply(&destination, &point, nil) })
		destination = point_filled(word)
		ignore_panic(func() { Point_Scalar_Base_Multiply(&destination, nil) })
		ignore_panic(func() { Point_Equal(&point, &point) })

		destination = point_filled(word)
		point_add(&destination, &point, &point)
		destination = point_filled(word)
		point_double(&destination, &point)
		destination = point_filled(word)
		point_scalar_multiply(&destination, &point, &edge)
		destination = point_filled(word)
		point_canonicalize_identity(&destination, &point)
		destination = point_filled(word)
		point_select(&destination, &point, &point, Decision(word&binary.UINT_8_SIZE))
	}
}

// Test_Internal_Field_Invariant_Domains reaches raw limbs forbidden by public decoding.
func Test_Internal_Field_Invariant_Domains(_ *testing.T) {
	var field_storage [FIELD_ELEMENT_SIZE]byte
	field_encoding := Field_Encoding(field_storage[:])
	var scalar_storage [SCALAR_SIZE]byte
	scalar_encoding := Scalar_Encoding(scalar_storage[:])
	for _, word := range [...]uint64{
		bits.WORD_64_MINIMUM,
		binary.UINT_8_SIZE,
		binary.UINT_16_SIZE,
		bits.WORD_64_MAXIMUM,
	} {
		value := limbs_filled(word)
		result := limbs_filled(word)
		point_affine_on_curve(&value, &value)
		point_projective_on_curve(&value, &value, &value)
		result = limbs_filled(word)
		field_polynomial(&result, &value)
		result = limbs_filled(word)
		field_multiply(&result, &value, &value)
		result = limbs_filled(word)
		field_square(&result, &value)
		result = limbs_filled(word)
		field_add(&result, &value, &value)
		result = limbs_filled(word)
		field_subtract(&result, &value, &value)
		result = limbs_filled(word)
		field_negate(&result, &value)
		result = limbs_filled(word)
		field_select(&result, &value, &value, Decision(word&binary.UINT_8_SIZE))
		field_equal(&value, &value)
		field_is_zero(&value)
		field_canonical(&value)
		result = limbs_filled(word)
		field_inverse(&result, &value)
		result = limbs_filled(word)
		field_square_root(&result, &value)
		result = limbs_filled(word)
		field_power(&result, &value, &value)
		result = limbs_filled(word)
		field_decode(&result, field_encoding)
		field_encode(field_encoding, &value)
		result = limbs_filled(word)
		scalar_decode(&result, scalar_encoding)
		result = limbs_filled(word)
		limbs_add(&result, &value, &value)
		result = limbs_filled(word)
		limbs_subtract(&result, &value, &value)
		result = limbs_filled(word)
		limbs_select(&result, &value, &value, Decision(word&binary.UINT_8_SIZE))

		result = limbs_filled(word)
		field_modulus(&result)
		result = limbs_filled(word)
		field_b(&result)
		result = limbs_filled(word)
		field_one(&result)
		result = limbs_filled(word)
		field_generator_x(&result)
		result = limbs_filled(word)
		field_generator_y(&result)
		result = limbs_filled(word)
		field_inverse_exponent(&result)
		result = limbs_filled(word)
		field_square_root_exponent(&result)
		result = limbs_filled(word)
		scalar_order(&result)
	}
}

func limbs_filled(word uint64) (result Limbs) {
	result = Limbs{
		Limb_0: Storage_Limb_0(word),
		Limb_1: Storage_Limb_1(word),
		Limb_2: Storage_Limb_2(word),
		Limb_3: Storage_Limb_3(word),
	}
	return result
}

func point_filled(word uint64) (result Point) {
	result = Point{
		X: X_Coordinate{
			Limb_0: X_Limb_0(word),
			Limb_1: X_Limb_1(word),
			Limb_2: X_Limb_2(word),
			Limb_3: X_Limb_3(word),
		},
		Y: Y_Coordinate{
			Limb_0: Y_Limb_0(word),
			Limb_1: Y_Limb_1(word),
			Limb_2: Y_Limb_2(word),
			Limb_3: Y_Limb_3(word),
		},
		Z: Z_Coordinate{
			Limb_0: Z_Limb_0(word),
			Limb_1: Z_Limb_1(word),
			Limb_2: Z_Limb_2(word),
			Limb_3: Z_Limb_3(word),
		},
	}
	return result
}

func ignore_panic(operation func()) {
	defer func() { recover() }()
	operation()
}
