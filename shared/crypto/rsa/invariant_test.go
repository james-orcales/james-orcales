package rsa

import (
	"testing"

	"local/james-orcales/shared/crypto/prng"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
)

// Test_Internal_Key_Invariant_Domains reaches raw key words refused by key parsing.
func Test_Internal_Key_Invariant_Domains(_ *testing.T) {
	var modulus_output_storage [MODULUS_SIZE]byte
	modulus_output := Modulus_Encoding(modulus_output_storage[:])
	var ciphertext_storage [MODULUS_SIZE]byte
	ciphertext := Ciphertext(ciphertext_storage[:])
	var signature_storage [MODULUS_SIZE]byte
	signature := Signature(signature_storage[:])
	var digest_storage [HASH_SIZE]byte
	digest := Digest(digest_storage[:])
	var plaintext_storage [MESSAGE_SIZE_MAXIMUM]byte
	plaintext := Destination(plaintext_storage[:])
	for _, word := range [...]uint64{
		bits.WORD_64_MINIMUM,
		binary.UINT_8_SIZE,
		binary.UINT_16_SIZE,
		bits.WORD_64_MAXIMUM,
	} {
		public_key := Public_Key{Modulus: modulus_filled(word)}
		private_key := Private_Key{
			Modulus:  modulus_filled(word),
			Exponent: private_exponent_filled(word),
		}
		Public_Key_Set_Bytes(&public_key, nil)
		Private_Key_Set_Bytes(&private_key, nil, nil)

		public_key = Public_Key{Modulus: modulus_filled(word)}
		private_key = Private_Key{
			Modulus:  modulus_filled(word),
			Exponent: private_exponent_filled(word),
		}
		ignore_panic(func() { Public_Key_From_Private(&public_key, private_key) })
		ignore_panic(func() { Public_Key_Bytes_Into(modulus_output, public_key) })
		ignore_panic(func() {
			Encrypt_OAEP_SHA_256(ciphertext, prng.Source{}, public_key, nil)
		})
		ignore_panic(func() {
			Decrypt_OAEP_SHA_256(plaintext, private_key, nil)
		})
		ignore_panic(func() {
			Sign_PSS_SHA_256(signature, prng.Source{}, private_key, digest)
		})
		ignore_panic(func() {
			Sign_PKCS1_V1_5_SHA_256(signature, private_key, digest)
		})
		ignore_panic(func() { Verify_PSS_SHA_256(public_key, digest, nil) })
		ignore_panic(func() { Verify_PKCS1_V1_5_SHA_256(public_key, digest, nil) })
	}
}

// Test_Internal_Integer_Invariant_Domains reaches each opaque encoding path.
func Test_Internal_Integer_Invariant_Domains(_ *testing.T) {
	var source_storage, destination_storage [MODULUS_SIZE]byte
	source := Encoded(source_storage[:])
	destination := Encoded(destination_storage[:])
	for _, word := range [...]uint64{
		bits.WORD_64_MINIMUM,
		binary.UINT_8_SIZE,
		binary.UINT_16_SIZE,
		bits.WORD_64_MAXIMUM,
	} {
		modulus := modulus_filled(word)
		exponent := private_exponent_filled(word)
		modulus_bytes(&modulus)
		private_exponent_bytes(&exponent)
		modulus_encoding_valid(&modulus)
		private_exponent_encoding_valid(&exponent, &modulus)
		integer_encoding_canonical(source, &modulus)
		rsa_public_operation(destination, source, &modulus)
		rsa_private_operation(destination, source, &modulus, &exponent)
	}
}

func modulus_filled(word uint64) (result Modulus) {
	chunk := integer_chunk_storage_filled(word)
	result = Modulus{
		Chunk_0: Integer_Chunk_0(chunk),
		Chunk_1: Integer_Chunk_1(chunk),
		Chunk_2: Integer_Chunk_2(chunk),
		Chunk_3: Integer_Chunk_3(chunk),
		Chunk_4: Integer_Chunk_4(chunk),
		Chunk_5: Integer_Chunk_5(chunk),
		Chunk_6: Integer_Chunk_6(chunk),
		Chunk_7: Integer_Chunk_7(chunk),
	}
	return result
}

func private_exponent_filled(word uint64) (result Private_Exponent) {
	chunk := integer_chunk_storage_filled(word)
	result = Private_Exponent{
		Chunk_0: Private_Exponent_Chunk_0(chunk),
		Chunk_1: Private_Exponent_Chunk_1(chunk),
		Chunk_2: Private_Exponent_Chunk_2(chunk),
		Chunk_3: Private_Exponent_Chunk_3(chunk),
		Chunk_4: Private_Exponent_Chunk_4(chunk),
		Chunk_5: Private_Exponent_Chunk_5(chunk),
		Chunk_6: Private_Exponent_Chunk_6(chunk),
		Chunk_7: Private_Exponent_Chunk_7(chunk),
	}
	return result
}

func integer_chunk_storage_filled(word uint64) (result Integer_Chunk_Storage) {
	result = Integer_Chunk_Storage{
		Lane_0: Integer_Lane_0(word),
		Lane_1: Integer_Lane_1(word),
		Lane_2: Integer_Lane_2(word),
		Lane_3: Integer_Lane_3(word),
	}
	return result
}

func ignore_panic(operation func()) {
	defer func() { recover() }()
	operation()
}
