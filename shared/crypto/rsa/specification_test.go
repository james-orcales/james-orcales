package rsa_test

import (
	"testing"

	"local/james-orcales/shared/crypto/prng"
	"local/james-orcales/shared/crypto/rsa"
	"local/james-orcales/shared/crypto/sha256"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/encoding/hex"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// Test_Keys validates pinned RSA-2048 key material.
func Test_Keys(t *testing.T) {
	private_key, public_key := owned_keys(t)
	var modulus_storage [rsa.MODULUS_SIZE]byte
	modulus := rsa.Modulus_Encoding(modulus_storage[:])
	rsa.Public_Key_Bytes_Into(modulus, public_key)
	expected := key_component(t, MODULUS_HEX)
	testify.Equal(t, []byte(expected), []byte(modulus))
	var derived rsa.Public_Key
	rsa.Public_Key_From_Private(&derived, private_key)
	testify.Equal(t, public_key, derived)
}

// Test_Encryption proves owned OAEP round-trip interoperability.
func Test_Encryption(t *testing.T) {
	private_key, public_key := owned_keys(t)
	message := []byte("bounded RSA OAEP")
	generator := test_generator()
	source := prng.Chacha_To_Source(&generator)
	var ciphertext_storage [rsa.MODULUS_SIZE]byte
	ciphertext := rsa.Ciphertext(ciphertext_storage[:])
	rsa.Encrypt_OAEP_SHA_256(ciphertext, source, public_key, message)
	var output [rsa.MESSAGE_SIZE_MAXIMUM]byte
	count, status := rsa.Decrypt_OAEP_SHA_256(
		output[:], private_key, rsa.Ciphertext_Unvalidated(ciphertext),
	)
	testify.Equal(t, rsa.DECRYPT_STATUS_OK, status)
	testify.Equal(t, message, output[:count])
}

// Test_Signatures proves owned PSS and PKCS1 v1.5 round trips.
func Test_Signatures(t *testing.T) {
	private_key, public_key := owned_keys(t)
	digest := digest_sha_256([]byte("bounded RSA signatures"))
	generator := test_generator()
	source := prng.Chacha_To_Source(&generator)
	var signature_storage [rsa.MODULUS_SIZE]byte
	signature := rsa.Signature(signature_storage[:])
	rsa.Sign_PSS_SHA_256(signature, source, private_key, digest[:])
	testify.True(t, bool(rsa.Verify_PSS_SHA_256(
		public_key, digest[:], rsa.Signature_Unvalidated(signature),
	)))

	rsa.Sign_PKCS1_V1_5_SHA_256(signature, private_key, digest[:])
	testify.True(t, bool(rsa.Verify_PKCS1_V1_5_SHA_256(
		public_key, digest[:], rsa.Signature_Unvalidated(signature),
	)))
	signature[len(signature)-binary.UINT_8_SIZE] ^= byte(binary.UINT_8_SIZE)
	testify.False(t, bool(rsa.Verify_PKCS1_V1_5_SHA_256(
		public_key, digest[:], rsa.Signature_Unvalidated(signature),
	)))
}

// Test_Bounds rejects hostile sizes and preserves output on refusal.
func Test_Bounds(t *testing.T) {
	private_key, public_key := owned_keys(t)
	generator := test_generator()
	source := prng.Chacha_To_Source(&generator)
	test_empty_keys(t)
	var ciphertext_storage [rsa.MODULUS_SIZE]byte
	ciphertext := rsa.Ciphertext(ciphertext_storage[:])
	var message_oversized [rsa.MESSAGE_SIZE_MAXIMUM + binary.UINT_8_SIZE]byte
	testify.Panics(t, func() {
		rsa.Encrypt_OAEP_SHA_256(
			ciphertext, source, public_key, message_oversized[:],
		)
	})
	var ciphertext_oversized [rsa.CIPHERTEXT_UNVALIDATED_SIZE_MAXIMUM +
		binary.UINT_8_SIZE]byte
	var output [rsa.MESSAGE_SIZE_MAXIMUM]byte
	testify.Panics(t, func() {
		rsa.Decrypt_OAEP_SHA_256(
			output[:], private_key, ciphertext_oversized[:],
		)
	})
	output[bits.BIT_COUNT_MINIMUM] = bits.WORD_8_MAXIMUM
	count, status := rsa.Decrypt_OAEP_SHA_256(output[:], private_key, nil)
	testify.Equal(t, rsa.COUNT_MINIMUM, count)
	testify.Equal(t, rsa.DECRYPT_STATUS_INPUT_INVALID, status)
	testify.Equal(t, byte(bits.WORD_8_MAXIMUM), output[bits.BIT_COUNT_MINIMUM])

	rsa.Encrypt_OAEP_SHA_256(ciphertext, source, public_key, []byte("short"))
	var short [rsa.DESTINATION_SIZE_MINIMUM + binary.UINT_8_SIZE]byte
	short[bits.BIT_COUNT_MINIMUM] = bits.WORD_8_MAXIMUM
	count, status = rsa.Decrypt_OAEP_SHA_256(
		short[:], private_key, rsa.Ciphertext_Unvalidated(ciphertext),
	)
	testify.Equal(t, rsa.Count(len("short")), count)
	testify.Equal(t, rsa.DECRYPT_STATUS_DESTINATION_TOO_SMALL, status)
	testify.Equal(t, byte(bits.WORD_8_MAXIMUM), short[bits.BIT_COUNT_MINIMUM])

	digest := digest_sha_256(nil)
	var signature_oversized [rsa.SIGNATURE_UNVALIDATED_SIZE_MAXIMUM +
		binary.UINT_8_SIZE]byte
	testify.Panics(t, func() {
		rsa.Verify_PSS_SHA_256(public_key, digest[:], signature_oversized[:])
	})
	testify.False(t, bool(rsa.Verify_PSS_SHA_256(public_key, digest[:], nil)))
	var noncanonical [rsa.MODULUS_SIZE]byte
	for index := range noncanonical {
		noncanonical[index] = bits.WORD_8_MAXIMUM
	}
	testify.False(t, bool(rsa.Verify_PSS_SHA_256(
		public_key, digest[:], noncanonical[:],
	)))
}

// Test_Allocation measures every exported successful runtime operation.
func Test_Allocation(t *testing.T) {
	modulus_encoding := key_component(t, MODULUS_HEX)
	private_exponent := key_component(t, PRIVATE_EXPONENT_HEX)
	var private_key rsa.Private_Key
	var public_key, derived_public_key rsa.Public_Key
	var modulus_storage [rsa.MODULUS_SIZE]byte
	modulus := rsa.Modulus_Encoding(modulus_storage[:])
	var private_key_status, public_key_status rsa.Key_Status
	testify.Zero_Allocation(t, func() {
		public_key_status = rsa.Public_Key_Set_Bytes(
			&public_key, rsa.Modulus_Unvalidated(modulus_encoding),
		)
	})
	testify.Zero_Allocation(t, func() {
		private_key_status = rsa.Private_Key_Set_Bytes(
			&private_key, rsa.Modulus_Unvalidated(modulus_encoding),
			rsa.Private_Exponent_Unvalidated(private_exponent),
		)
	})
	testify.Zero_Allocation(t, func() {
		rsa.Public_Key_From_Private(&derived_public_key, private_key)
	})
	testify.Zero_Allocation(t, func() {
		rsa.Public_Key_Bytes_Into(modulus, public_key)
	})
	message := []byte("allocation")
	digest := digest_sha_256(message)
	generator := test_generator()
	source := prng.Chacha_To_Source(&generator)
	var ciphertext_storage, signature_storage [rsa.MODULUS_SIZE]byte
	ciphertext := rsa.Ciphertext(ciphertext_storage[:])
	signature := rsa.Signature(signature_storage[:])
	var output [rsa.MESSAGE_SIZE_MAXIMUM]byte
	var count rsa.Count
	var decrypt_status rsa.Decrypt_Status
	var verified rsa.Verification
	testify.Zero_Allocation(t, func() {
		rsa.Encrypt_OAEP_SHA_256(ciphertext, source, public_key, message)
	})
	testify.Zero_Allocation(t, func() {
		count, decrypt_status = rsa.Decrypt_OAEP_SHA_256(
			output[:], private_key, rsa.Ciphertext_Unvalidated(ciphertext),
		)
	})
	testify.Zero_Allocation(t, func() {
		rsa.Sign_PSS_SHA_256(signature, source, private_key, digest[:])
	})
	testify.Zero_Allocation(t, func() {
		verified = rsa.Verify_PSS_SHA_256(
			public_key, digest[:], rsa.Signature_Unvalidated(signature),
		)
	})
	testify.Zero_Allocation(t, func() {
		rsa.Sign_PKCS1_V1_5_SHA_256(signature, private_key, digest[:])
	})
	testify.Zero_Allocation(t, func() {
		verified = rsa.Verify_PKCS1_V1_5_SHA_256(
			public_key, digest[:], rsa.Signature_Unvalidated(signature),
		)
	})
	testify.Equal(t, rsa.Count(len(message)), count)
	testify.Equal(t, rsa.KEY_STATUS_OK, private_key_status)
	testify.Equal(t, rsa.KEY_STATUS_OK, public_key_status)
	testify.Equal(t, public_key, derived_public_key)
	testify.Equal(t, []byte(modulus_encoding), []byte(modulus))
	testify.Equal(t, rsa.DECRYPT_STATUS_OK, decrypt_status)
	testify.True(t, bool(verified))
}

// Test_Invariant_Domains reaches every bounded input, output, count, and generator value.
func Test_Invariant_Domains(t *testing.T) {
	test_key_domains()
	private_key, public_key := owned_keys(t)
	test_encryption_domains(private_key, public_key)
	test_signature_domains(public_key)
	test_generator_domains(t)
}

func test_empty_keys(t *testing.T) {
	t.Helper()
	var modulus_storage [rsa.MODULUS_SIZE]byte
	modulus := rsa.Modulus_Encoding(modulus_storage[:])
	testify.Panics(t, func() {
		rsa.Public_Key_Bytes_Into(modulus, rsa.Public_Key{})
	})
	testify.Panics(t, func() {
		var public_key rsa.Public_Key
		rsa.Public_Key_From_Private(&public_key, rsa.Private_Key{})
	})
	var output [rsa.MESSAGE_SIZE_MAXIMUM]byte
	testify.Panics(t, func() {
		rsa.Decrypt_OAEP_SHA_256(output[:], rsa.Private_Key{}, nil)
	})
	digest := digest_sha_256(nil)
	var signature_storage [rsa.MODULUS_SIZE]byte
	signature := rsa.Signature(signature_storage[:])
	testify.Panics(t, func() {
		rsa.Sign_PKCS1_V1_5_SHA_256(signature, rsa.Private_Key{}, digest[:])
	})
	testify.Panics(t, func() {
		rsa.Verify_PSS_SHA_256(rsa.Public_Key{}, digest[:], nil)
	})
	testify.Panics(t, func() {
		rsa.Verify_PKCS1_V1_5_SHA_256(rsa.Public_Key{}, digest[:], nil)
	})
}

func test_key_domains() {
	var public_key rsa.Public_Key
	var private_key rsa.Private_Key
	var modulus [rsa.MODULUS_UNVALIDATED_SIZE_MAXIMUM]byte
	var exponent [rsa.PRIVATE_EXPONENT_UNVALIDATED_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		rsa.MODULUS_UNVALIDATED_SIZE_MINIMUM,
		rsa.MODULUS_UNVALIDATED_SIZE_MINIMUM + binary.UINT_8_SIZE,
		rsa.MODULUS_UNVALIDATED_SIZE_MINIMUM + binary.UINT_16_SIZE,
		rsa.MODULUS_UNVALIDATED_SIZE_MAXIMUM - binary.UINT_8_SIZE,
		rsa.MODULUS_UNVALIDATED_SIZE_MAXIMUM,
	} {
		rsa.Public_Key_Set_Bytes(&public_key, modulus[:size])
		rsa.Private_Key_Set_Bytes(&private_key, modulus[:size], exponent[:size])
	}
}

func test_encryption_domains(private_key rsa.Private_Key, public_key rsa.Public_Key) {
	generator := test_generator()
	source := prng.Chacha_To_Source(&generator)
	var ciphertext_storage [rsa.MODULUS_SIZE]byte
	ciphertext := rsa.Ciphertext(ciphertext_storage[:])
	var message [rsa.MESSAGE_SIZE_MAXIMUM]byte
	var output [rsa.MESSAGE_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		rsa.MESSAGE_SIZE_MINIMUM,
		rsa.MESSAGE_SIZE_MINIMUM + binary.UINT_8_SIZE,
		rsa.MESSAGE_SIZE_MINIMUM + binary.UINT_16_SIZE,
		rsa.MESSAGE_SIZE_MAXIMUM - binary.UINT_8_SIZE,
		rsa.MESSAGE_SIZE_MAXIMUM,
	} {
		rsa.Encrypt_OAEP_SHA_256(ciphertext, source, public_key, message[:size])
		rsa.Decrypt_OAEP_SHA_256(
			output[:], private_key, rsa.Ciphertext_Unvalidated(ciphertext),
		)
	}
	for _, size := range [...]int{
		rsa.DESTINATION_SIZE_MINIMUM,
		rsa.DESTINATION_SIZE_MINIMUM + binary.UINT_8_SIZE,
		rsa.DESTINATION_SIZE_MINIMUM + binary.UINT_16_SIZE,
		rsa.DESTINATION_SIZE_MAXIMUM - binary.UINT_8_SIZE,
		rsa.DESTINATION_SIZE_MAXIMUM,
	} {
		rsa.Decrypt_OAEP_SHA_256(
			output[:size], private_key, rsa.Ciphertext_Unvalidated(ciphertext),
		)
	}
	var hostile_ciphertext [rsa.CIPHERTEXT_UNVALIDATED_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		rsa.CIPHERTEXT_UNVALIDATED_SIZE_MINIMUM,
		rsa.CIPHERTEXT_UNVALIDATED_SIZE_MINIMUM + binary.UINT_8_SIZE,
		rsa.CIPHERTEXT_UNVALIDATED_SIZE_MINIMUM + binary.UINT_16_SIZE,
		rsa.CIPHERTEXT_UNVALIDATED_SIZE_MAXIMUM - binary.UINT_8_SIZE,
		rsa.CIPHERTEXT_UNVALIDATED_SIZE_MAXIMUM,
	} {
		rsa.Decrypt_OAEP_SHA_256(output[:], private_key, hostile_ciphertext[:size])
	}
}

func test_signature_domains(public_key rsa.Public_Key) {
	digest := digest_sha_256(nil)
	var hostile_signature [rsa.SIGNATURE_UNVALIDATED_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		rsa.SIGNATURE_UNVALIDATED_SIZE_MINIMUM,
		rsa.SIGNATURE_UNVALIDATED_SIZE_MINIMUM + binary.UINT_8_SIZE,
		rsa.SIGNATURE_UNVALIDATED_SIZE_MINIMUM + binary.UINT_16_SIZE,
		rsa.SIGNATURE_UNVALIDATED_SIZE_MAXIMUM - binary.UINT_8_SIZE,
		rsa.SIGNATURE_UNVALIDATED_SIZE_MAXIMUM,
	} {
		rsa.Verify_PSS_SHA_256(public_key, digest[:], hostile_signature[:size])
		rsa.Verify_PKCS1_V1_5_SHA_256(public_key, digest[:], hostile_signature[:size])
	}
}

func test_generator_domains(t *testing.T) {
	t.Helper()
	digest := digest_sha_256(nil)
	var ciphertext_storage, signature_storage [rsa.MODULUS_SIZE]byte
	ciphertext := rsa.Ciphertext(ciphertext_storage[:])
	signature := rsa.Signature(signature_storage[:])
	for _, position := range [...]prng.Cursor{
		prng.CURSOR_MIN,
		prng.CURSOR_MIN + binary.UINT_8_SIZE,
		prng.CURSOR_MIN + binary.UINT_16_SIZE,
		prng.CURSOR_MAX,
	} {
		empty_generator := prng.Chacha{Position: position}
		empty_source := prng.Chacha_To_Source(&empty_generator)
		testify.Panics(t, func() {
			rsa.Encrypt_OAEP_SHA_256(
				ciphertext, empty_source, rsa.Public_Key{}, nil,
			)
		})
		testify.Panics(t, func() {
			rsa.Sign_PSS_SHA_256(
				signature, empty_source, rsa.Private_Key{}, digest[:],
			)
		})
	}
}

// Fixed components keep key tests deterministic and independent from ambient entropy.
const MODULUS_HEX = "cda3215461e0eeab0fa48b2bf8ee2072fee75b5a641da5b519f09a1b1eff2e42" +
	"bbcbeec40dea5f37b5e1e0349b17aad88b177302f8af9723a0ea8715167d0c0b" +
	"8a9f16127bfb271ffee774bcca9fc24b8270f3a4356d01db663143a5ca2a4aec" +
	"00c657a0c35f073590987e98a96270359580a6f8c4e3b605c941b8c0d1dc1418" +
	"27f804332ab7eefa264dd7645e7def9adfb5b906a13f5ebbc26d5ece45e57930" +
	"77cab77e4433f9fa4b5b8bbd8eb7c5c90eb545116462b0a3961b468d82330471" +
	"a040ce27fec357862cb9874b40ed76ba37262b37f3977bf6352bc473fedce5bd" +
	"b328c60660b41b8aa353779bc724162ee15152b678dbfb1a801bb5e4d33eb83f"

const PRIVATE_EXPONENT_HEX = "0277ec5699ac2f7474061e205d1a4f78ceb71fa7b56bc07c93257088ef68d1c5" +
	"d780aa7df8fdfd8b37c1d0ab7a9720a3fd683ec0e413eeb839ab2d72230edf04" +
	"71a6723aaec61dee374b37619f957329ba4806397f4be1fcec8937daf752ff3e" +
	"3dce23b5bed2a3b403cdd59e66eaecb019940746af00c45d18d1463f94b220b7" +
	"33977439f614f14387334472a3674e8bd7fb00fad1556ced63dfd14df2730e53" +
	"fa2f2c5e25026653910e803fa9fa5cf1cf07c36aee3498cfa4433b6d348f3a7a" +
	"52c76b9e38e42516f33b137f7e7f9c9023fe598baafb2fddd39113f4f7556b8a" +
	"88272afcd03f5080799fced71e72f630a2c9069e46f61784e6a745a7023ff155"

func owned_keys(t *testing.T) (private_key rsa.Private_Key, public_key rsa.Public_Key) {
	t.Helper()
	modulus := key_component(t, MODULUS_HEX)
	private_exponent := key_component(t, PRIVATE_EXPONENT_HEX)
	status := rsa.Private_Key_Set_Bytes(
		&private_key, rsa.Modulus_Unvalidated(modulus),
		rsa.Private_Exponent_Unvalidated(private_exponent),
	)
	testify.Equal(t, rsa.KEY_STATUS_OK, status)
	status = rsa.Public_Key_Set_Bytes(&public_key, rsa.Modulus_Unvalidated(modulus))
	testify.Equal(t, rsa.KEY_STATUS_OK, status)
	return private_key, public_key
}

type test_key_component []byte

func key_component(t *testing.T, encoded string) (component test_key_component) {
	t.Helper()
	component = make(test_key_component, rsa.MODULUS_SIZE)
	count, status := hex.Decode_Into(hex.Decoded(component), []byte(encoded))
	testify.Equal(t, hex.Decode_Status(hex.STATUS_OK), status)
	testify.Equal(t, hex.Decoded_Count(len(component)), count)
	return component
}

func digest_sha_256(source sha256.Source) (digest rsa.Digest) {
	digest = make(rsa.Digest, sha256.DIGEST_256_SIZE)
	sha256.Checksum_Into(sha256.Destination(digest), sha256.KIND_SHA_256, source)
	return digest
}

func test_generator() (generator prng.Chacha) {
	var seed [prng.KEY_BYTES]byte
	prng.Chacha_Init(&generator, seed[:], prng.CURSOR_MIN)
	return generator
}
