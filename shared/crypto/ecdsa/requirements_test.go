package ecdsa_test

import (
	"testing"

	"local/james-orcales/shared/crypto/ecdsa"
	"local/james-orcales/shared/crypto/prng"
	"local/james-orcales/shared/crypto/sha256"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// Test_Keys binds the minimum private scalar to the SEC 2 P-256 generator.
func Test_Keys(t *testing.T) {
	private_encoding := private_scalar_minimum()
	var private_key ecdsa.Private_Key
	status := ecdsa.Private_Key_Set_Bytes(
		&private_key, ecdsa.Private_Key_Unvalidated(private_encoding),
	)
	testify.Equal(t, ecdsa.KEY_STATUS_OK, status)

	var public_key ecdsa.Public_Key
	ecdsa.Public_Key_From_Private(&public_key, private_key)
	var public_storage [ecdsa.PUBLIC_KEY_SIZE]byte
	public_encoding := ecdsa.Public_Key_Encoding(public_storage[:])
	ecdsa.Public_Key_Bytes_Into(public_encoding, public_key)
	want := private_public_key_encoding()
	testify.Equal(t, want, []byte(public_encoding))

	var parsed ecdsa.Public_Key
	status = ecdsa.Public_Key_Set_Bytes(
		&parsed, ecdsa.Public_Key_Unvalidated(public_encoding),
	)
	testify.Equal(t, ecdsa.KEY_STATUS_OK, status)
	var parsed_storage [ecdsa.PUBLIC_KEY_SIZE]byte
	parsed_encoding := ecdsa.Public_Key_Encoding(parsed_storage[:])
	ecdsa.Public_Key_Bytes_Into(parsed_encoding, parsed)
	testify.Equal(t, public_encoding, parsed_encoding)
}

// Test_Signatures checks owned verification and tamper refusal.
func Test_Signatures(t *testing.T) {
	private_key := private_key_minimum(t)
	var public_key ecdsa.Public_Key
	ecdsa.Public_Key_From_Private(&public_key, private_key)
	digest := digest_sha_256([]byte("bounded ECDSA"))
	generator := test_generator()
	source := prng.Chacha_To_Source(&generator)
	var signature_storage [ecdsa.SIGNATURE_SIZE]byte
	signature := ecdsa.Signature(signature_storage[:])
	status := ecdsa.Sign(
		signature, source, private_key, ecdsa.Digest(digest[:]),
	)
	testify.Equal(t, ecdsa.SIGN_STATUS_OK, status)

	testify.True(t, bool(ecdsa.Verify(
		public_key, ecdsa.Digest(digest[:]), ecdsa.Signature_Unvalidated(signature),
	)))

	signature[len(signature)-binary.UINT_8_SIZE] ^= byte(binary.UINT_8_SIZE)
	testify.False(t, bool(ecdsa.Verify(
		public_key, ecdsa.Digest(digest[:]), ecdsa.Signature_Unvalidated(signature),
	)))
}

// Test_Bounds preserves key destinations and rejects bounded malformed signatures.
func Test_Bounds(t *testing.T) {
	private_key := private_key_minimum(t)
	private_before := private_key
	var zero [ecdsa.PRIVATE_KEY_SIZE]byte
	status := ecdsa.Private_Key_Set_Bytes(&private_key, zero[:])
	testify.Equal(t, ecdsa.KEY_STATUS_INPUT_INVALID, status)
	testify.Equal(t, private_before, private_key)

	var public_key ecdsa.Public_Key
	ecdsa.Public_Key_From_Private(&public_key, private_key)
	public_before := public_key
	status = ecdsa.Public_Key_Set_Bytes(&public_key, []byte{bits.WORD_8_MINIMUM})
	testify.Equal(t, ecdsa.KEY_STATUS_INPUT_INVALID, status)
	testify.Equal(t, public_before, public_key)

	digest := digest_sha_256(nil)
	testify.False(t, bool(ecdsa.Verify(public_key, ecdsa.Digest(digest[:]), nil)))
	var zero_signature [ecdsa.SIGNATURE_SIZE]byte
	testify.False(t, bool(ecdsa.Verify(
		public_key, ecdsa.Digest(digest[:]), zero_signature[:],
	)))
	var empty_generator prng.Chacha
	empty_source := prng.Chacha_To_Source(&empty_generator)
	var signature_storage [ecdsa.SIGNATURE_SIZE]byte
	signature := ecdsa.Signature(signature_storage[:])
	signature[bits.BIT_COUNT_MINIMUM] = bits.WORD_8_MAXIMUM
	var signature_before [ecdsa.SIGNATURE_SIZE]byte
	copy(signature_before[:], signature)
	sign_status := ecdsa.Sign(
		signature, empty_source, private_key, ecdsa.Digest(digest[:]),
	)
	testify.Equal(t, ecdsa.SIGN_STATUS_ENTROPY_EXHAUSTED, sign_status)
	testify.Equal(t, signature_before[:], []byte(signature))

	var private_oversized [ecdsa.PRIVATE_KEY_UNVALIDATED_SIZE_MAXIMUM +
		binary.UINT_8_SIZE]byte
	testify.Panics(t, func() {
		ecdsa.Private_Key_Set_Bytes(&private_key, private_oversized[:])
	})
	var public_oversized [ecdsa.PUBLIC_KEY_UNVALIDATED_SIZE_MAXIMUM +
		binary.UINT_8_SIZE]byte
	testify.Panics(t, func() {
		ecdsa.Public_Key_Set_Bytes(&public_key, public_oversized[:])
	})
	var signature_oversized [ecdsa.SIGNATURE_UNVALIDATED_SIZE_MAXIMUM +
		binary.UINT_8_SIZE]byte
	testify.Panics(t, func() {
		ecdsa.Verify(public_key, ecdsa.Digest(digest[:]), signature_oversized[:])
	})
}

// Test_Allocation measures every exported successful runtime operation.
func Test_Allocation(t *testing.T) {
	private_encoding := private_scalar_minimum()
	var private_key ecdsa.Private_Key
	var public_key, parsed ecdsa.Public_Key
	var public_storage [ecdsa.PUBLIC_KEY_SIZE]byte
	public_encoding := ecdsa.Public_Key_Encoding(public_storage[:])
	var signature_storage [ecdsa.SIGNATURE_SIZE]byte
	signature := ecdsa.Signature(signature_storage[:])
	digest := digest_sha_256([]byte("allocation"))
	owned_digest := ecdsa.Digest(digest[:])
	generator := test_generator()
	source := prng.Chacha_To_Source(&generator)
	var key_status ecdsa.Key_Status
	var sign_status ecdsa.Sign_Status
	var verified ecdsa.Verification
	testify.Zero_Allocation(t, func() {
		key_status = ecdsa.Private_Key_Set_Bytes(
			&private_key, ecdsa.Private_Key_Unvalidated(private_encoding),
		)
	})
	testify.Zero_Allocation(t, func() {
		ecdsa.Public_Key_From_Private(&public_key, private_key)
	})
	testify.Zero_Allocation(t, func() {
		ecdsa.Public_Key_Bytes_Into(public_encoding, public_key)
	})
	testify.Zero_Allocation(t, func() {
		key_status = ecdsa.Public_Key_Set_Bytes(
			&parsed, ecdsa.Public_Key_Unvalidated(public_encoding),
		)
	})
	testify.Zero_Allocation(t, func() {
		sign_status = ecdsa.Sign(signature, source, private_key, owned_digest)
	})
	testify.Zero_Allocation(t, func() {
		verified = ecdsa.Verify(
			public_key, owned_digest, ecdsa.Signature_Unvalidated(signature),
		)
	})
	testify.Equal(t, ecdsa.KEY_STATUS_OK, key_status)
	testify.Equal(t, ecdsa.SIGN_STATUS_OK, sign_status)
	testify.True(t, bool(verified))
}

// Test_Invariant_Domains reaches every bounded hostile input length.
func Test_Invariant_Domains(t *testing.T) {
	var private_key ecdsa.Private_Key
	var private_source [ecdsa.PRIVATE_KEY_UNVALIDATED_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		ecdsa.PRIVATE_KEY_UNVALIDATED_SIZE_MINIMUM,
		ecdsa.PRIVATE_KEY_UNVALIDATED_SIZE_MINIMUM + binary.UINT_8_SIZE,
		ecdsa.PRIVATE_KEY_UNVALIDATED_SIZE_MINIMUM + binary.UINT_16_SIZE,
		ecdsa.PRIVATE_KEY_UNVALIDATED_SIZE_MAXIMUM - binary.UINT_8_SIZE,
		ecdsa.PRIVATE_KEY_UNVALIDATED_SIZE_MAXIMUM,
	} {
		ecdsa.Private_Key_Set_Bytes(&private_key, private_source[:size])
	}

	var public_key ecdsa.Public_Key
	var public_source [ecdsa.PUBLIC_KEY_UNVALIDATED_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		ecdsa.PUBLIC_KEY_UNVALIDATED_SIZE_MINIMUM,
		ecdsa.PUBLIC_KEY_UNVALIDATED_SIZE_MINIMUM + binary.UINT_8_SIZE,
		ecdsa.PUBLIC_KEY_UNVALIDATED_SIZE_MINIMUM + binary.UINT_16_SIZE,
		ecdsa.PUBLIC_KEY_UNVALIDATED_SIZE_MAXIMUM - binary.UINT_8_SIZE,
		ecdsa.PUBLIC_KEY_UNVALIDATED_SIZE_MAXIMUM,
	} {
		ecdsa.Public_Key_Set_Bytes(&public_key, public_source[:size])
	}

	ecdsa.Public_Key_From_Private(&public_key, private_key_minimum(t))
	digest_storage := digest_sha_256(nil)
	digest := ecdsa.Digest(digest_storage[:])
	var signature [ecdsa.SIGNATURE_UNVALIDATED_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		ecdsa.SIGNATURE_UNVALIDATED_SIZE_MINIMUM,
		ecdsa.SIGNATURE_UNVALIDATED_SIZE_MINIMUM + binary.UINT_8_SIZE,
		ecdsa.SIGNATURE_UNVALIDATED_SIZE_MINIMUM + binary.UINT_16_SIZE,
		ecdsa.SIGNATURE_UNVALIDATED_SIZE_MAXIMUM - binary.UINT_8_SIZE,
		ecdsa.SIGNATURE_UNVALIDATED_SIZE_MAXIMUM,
	} {
		ecdsa.Verify(public_key, digest, signature[:size])
	}

	var output_storage [ecdsa.SIGNATURE_SIZE]byte
	output := ecdsa.Signature(output_storage[:])
	for _, position := range [...]prng.Cursor{
		prng.CURSOR_MIN,
		prng.CURSOR_MIN + binary.UINT_8_SIZE,
		prng.CURSOR_MIN + binary.UINT_16_SIZE,
		prng.CURSOR_MAX,
	} {
		generator := prng.Chacha{Position: position}
		source := prng.Chacha_To_Source(&generator)
		testify.Panics(t, func() {
			ecdsa.Sign(output, source, ecdsa.Private_Key{}, digest)
		})
	}
	test_key_state_and_scalar_decisions(t, digest, output)
}

func test_key_state_and_scalar_decisions(
	t *testing.T, digest ecdsa.Digest, output ecdsa.Signature,
) {
	var public_output_storage [ecdsa.PUBLIC_KEY_SIZE]byte
	public_output := ecdsa.Public_Key_Encoding(public_output_storage[:])
	testify.Panics(t, func() {
		ecdsa.Public_Key_Bytes_Into(public_output, ecdsa.Public_Key{})
	})
	testify.Panics(t, func() {
		var public_key ecdsa.Public_Key
		ecdsa.Public_Key_From_Private(&public_key, ecdsa.Private_Key{})
	})
	testify.Panics(t, func() {
		ecdsa.Verify(ecdsa.Public_Key{}, digest, nil)
	})

	private_key := private_key_minimum(t)
	for _, seed_value := range [...]byte{
		byte(binary.UINT_8_SIZE),
		byte(binary.UINT_16_SIZE),
		byte(binary.UINT_16_SIZE + binary.UINT_8_SIZE),
		byte(binary.UINT_32_SIZE),
	} {
		generator := test_generator_seed(seed_value)
		source := prng.Chacha_To_Source(&generator)
		status := ecdsa.Sign(output, source, private_key, digest)
		testify.Equal(t, ecdsa.SIGN_STATUS_OK, status)
	}
}

func private_scalar_minimum() (encoding []byte) {
	encoding = make([]byte, ecdsa.PRIVATE_KEY_SIZE)
	encoding[len(encoding)-binary.UINT_8_SIZE] = byte(binary.UINT_8_SIZE)
	return encoding
}

func private_key_minimum(t *testing.T) (private_key ecdsa.Private_Key) {
	t.Helper()
	encoding := private_scalar_minimum()
	status := ecdsa.Private_Key_Set_Bytes(
		&private_key, ecdsa.Private_Key_Unvalidated(encoding),
	)
	testify.Equal(t, ecdsa.KEY_STATUS_OK, status)
	return private_key
}

func test_generator() (generator prng.Chacha) {
	return test_generator_seed(bits.WORD_8_MINIMUM)
}

func test_generator_seed(seed_value byte) (generator prng.Chacha) {
	var seed [prng.KEY_BYTES]byte
	seed[bits.BIT_COUNT_MINIMUM] = seed_value
	prng.Chacha_Init(&generator, seed[:], prng.CURSOR_MIN)
	return generator
}

func digest_sha_256(source sha256.Source) (digest sha256.Destination) {
	digest = make(sha256.Destination, sha256.DIGEST_256_SIZE)
	sha256.Checksum_Into(digest, sha256.KIND_SHA_256, source)
	return digest
}

func private_public_key_encoding() (encoding []byte) {
	return []byte{
		0x04,
		0x6b, 0x17, 0xd1, 0xf2, 0xe1, 0x2c, 0x42, 0x47,
		0xf8, 0xbc, 0xe6, 0xe5, 0x63, 0xa4, 0x40, 0xf2,
		0x77, 0x03, 0x7d, 0x81, 0x2d, 0xeb, 0x33, 0xa0,
		0xf4, 0xa1, 0x39, 0x45, 0xd8, 0x98, 0xc2, 0x96,
		0x4f, 0xe3, 0x42, 0xe2, 0xfe, 0x1a, 0x7f, 0x9b,
		0x8e, 0xe7, 0xeb, 0x4a, 0x7c, 0x0f, 0x9e, 0x16,
		0x2b, 0xce, 0x33, 0x57, 0x6b, 0x31, 0x5e, 0xce,
		0xcb, 0xb6, 0x40, 0x68, 0x37, 0xbf, 0x51, 0xf5,
	}
}
