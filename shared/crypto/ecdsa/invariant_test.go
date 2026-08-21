package ecdsa

import (
	"testing"

	"local/james-orcales/shared/crypto/elliptic"
	"local/james-orcales/shared/crypto/prng"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// Test_Internal_Invariant_Domains sends storage edges through production boundaries that ordinary
// valid signatures cannot reach.
func Test_Internal_Invariant_Domains(t *testing.T) {
	var encoding_storage [SCALAR_SIZE]byte
	encoding := Scalar_Encoding(encoding_storage[:])
	var signature_storage [SIGNATURE_SIZE]byte
	signature := Signature(signature_storage[:])
	var public_key_storage [PUBLIC_KEY_SIZE]byte
	for _, word := range [...]uint64{
		bits.WORD_64_MINIMUM,
		binary.UINT_8_SIZE,
		binary.UINT_16_SIZE,
		bits.WORD_64_MAXIMUM,
	} {
		edge := scalar_filled(word)
		destination := edge
		scalar_decode(&destination, encoding)
		destination = edge
		scalar_decode_reduce(&destination, encoding)
		destination = edge
		scalar_decode_raw(&destination, encoding)
		scalar_encode(encoding, &edge)
		destination = edge
		scalar_add(&destination, &edge, &edge)
		destination = edge
		scalar_multiply(&destination, &edge, &edge)
		destination = edge
		scalar_inverse(&destination, &edge)
		destination = edge
		scalar_negate(&destination, &edge)
		scalar_equal(&edge, &edge)
		scalar_is_zero(&edge)
		scalar_greater(&edge, &edge)
		destination = edge
		scalar_select(&destination, &edge, &edge, Decision(word&binary.UINT_8_SIZE))
		destination = edge
		scalar_limbs_add(&destination, &edge, &edge)
		destination = edge
		scalar_limbs_subtract(&destination, &edge, &edge)

		destination = edge
		scalar_order(&destination)
		destination = edge
		scalar_order_minus_two(&destination)
		destination = edge
		scalar_one(&destination)
		destination = edge
		scalar_half_order(&destination)

		sign_candidate(signature, &edge, &edge, encoding)
		private_key := Private_Key{Scalar: edge}
		Private_Key_Set_Bytes(&private_key, nil)
		testify.Panics(t, func() {
			var source prng.Chacha
			random := prng.Chacha_To_Source(&source)
			Sign(
				signature, random, private_key, Digest(encoding),
			)
		})

		point := point_filled(word)
		public_key := Public_Key{Point: point}
		Public_Key_Set_Bytes(&public_key, nil)
		testify.Panics(t, func() {
			Public_Key_Bytes_Into(
				Public_Key_Encoding(public_key_storage[:]), public_key,
			)
		})
		testify.Panics(t, func() { Verify(public_key, Digest(encoding), nil) })
		testify.Panics(t, func() { Public_Key_From_Private(&public_key, private_key) })
	}
}

func scalar_filled(word uint64) (result Scalar) {
	result = Scalar{
		Limb_0: Scalar_Limb_0(word),
		Limb_1: Scalar_Limb_1(word),
		Limb_2: Scalar_Limb_2(word),
		Limb_3: Scalar_Limb_3(word),
	}
	return result
}

func point_filled(word uint64) (result elliptic.Point) {
	result = elliptic.Point{
		X: elliptic.X_Coordinate{
			Limb_0: elliptic.X_Limb_0(word),
			Limb_1: elliptic.X_Limb_1(word),
			Limb_2: elliptic.X_Limb_2(word),
			Limb_3: elliptic.X_Limb_3(word),
		},
		Y: elliptic.Y_Coordinate{
			Limb_0: elliptic.Y_Limb_0(word),
			Limb_1: elliptic.Y_Limb_1(word),
			Limb_2: elliptic.Y_Limb_2(word),
			Limb_3: elliptic.Y_Limb_3(word),
		},
		Z: elliptic.Z_Coordinate{
			Limb_0: elliptic.Z_Limb_0(word),
			Limb_1: elliptic.Z_Limb_1(word),
			Limb_2: elliptic.Z_Limb_2(word),
			Limb_3: elliptic.Z_Limb_3(word),
		},
	}
	return result
}
