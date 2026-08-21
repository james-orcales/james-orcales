package ed25519_test

import (
	"testing"

	"local/james-orcales/shared/crypto/ed25519"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/testify"
)

// Test_Keys matches RFC 8032 test vector one.
func Test_Keys(t *testing.T) {
	seed := vector_seed()
	var private_key ed25519.Private_Key
	ed25519.Private_Key_From_Seed(&private_key, seed)
	var public_storage [ed25519.PUBLIC_KEY_SIZE]byte
	public_key := ed25519.Public_Key(public_storage[:])
	ed25519.Public_Key_From_Private(public_key, &private_key)
	testify.Equal(t, []byte(vector_public_key()), []byte(public_key))
}

// Test_Signatures matches RFC 8032 and checks owned message verification.
func Test_Signatures(t *testing.T) {
	seed := vector_seed()
	var private_key ed25519.Private_Key
	ed25519.Private_Key_From_Seed(&private_key, seed)
	var public_storage [ed25519.PUBLIC_KEY_SIZE]byte
	public_key := ed25519.Public_Key(public_storage[:])
	ed25519.Public_Key_From_Private(public_key, &private_key)
	var signature_storage [ed25519.SIGNATURE_SIZE]byte
	signature := ed25519.Signature(signature_storage[:])
	ed25519.Sign(signature, &private_key, nil)
	testify.Equal(t, []byte(vector_signature()), []byte(signature))
	testify.True(t, bool(ed25519.Verify(
		public_key, nil, ed25519.Signature_Unvalidated(signature),
	)))
	signature[len(signature)-binary.UINT_8_SIZE] ^= byte(binary.UINT_8_SIZE)
	testify.False(t, bool(ed25519.Verify(
		public_key, nil, ed25519.Signature_Unvalidated(signature),
	)))

	message := []byte("bounded Ed25519")
	ed25519.Sign(signature, &private_key, message)
	testify.True(t, bool(ed25519.Verify(
		public_key, message, ed25519.Signature_Unvalidated(signature),
	)))
	signature[len(signature)-binary.UINT_8_SIZE] ^= byte(binary.UINT_8_SIZE)
	testify.False(t, bool(ed25519.Verify(
		public_key, message, ed25519.Signature_Unvalidated(signature),
	)))
}

// Test_Bounds rejects oversized input before cryptographic work.
func Test_Bounds(t *testing.T) {
	seed := vector_seed()
	var private_key ed25519.Private_Key
	ed25519.Private_Key_From_Seed(&private_key, seed)
	var public_storage [ed25519.PUBLIC_KEY_SIZE]byte
	public_key := ed25519.Public_Key(public_storage[:])
	ed25519.Public_Key_From_Private(public_key, &private_key)
	var signature_storage [ed25519.SIGNATURE_SIZE]byte
	signature := ed25519.Signature(signature_storage[:])
	var message_oversized [ed25519.MESSAGE_SIZE_MAXIMUM + binary.UINT_8_SIZE]byte
	testify.Panics(t, func() {
		ed25519.Sign(signature, &private_key, message_oversized[:])
	})
	testify.Panics(t, func() {
		ed25519.Verify(
			public_key, message_oversized[:], ed25519.Signature_Unvalidated(signature),
		)
	})
	var signature_oversized [ed25519.SIGNATURE_UNVALIDATED_SIZE_MAXIMUM +
		binary.UINT_8_SIZE]byte
	testify.Panics(t, func() {
		ed25519.Verify(public_key, nil, signature_oversized[:])
	})
	testify.False(t, bool(ed25519.Verify(public_key, nil, nil)))
}

// Test_Allocation measures every exported successful runtime operation.
func Test_Allocation(t *testing.T) {
	seed := vector_seed()
	var private_key ed25519.Private_Key
	var public_storage [ed25519.PUBLIC_KEY_SIZE]byte
	public_key := ed25519.Public_Key(public_storage[:])
	var signature_storage [ed25519.SIGNATURE_SIZE]byte
	signature := ed25519.Signature(signature_storage[:])
	var verified ed25519.Verification
	testify.Zero_Allocation(t, func() {
		ed25519.Private_Key_From_Seed(&private_key, seed)
	})
	testify.Zero_Allocation(t, func() {
		ed25519.Public_Key_From_Private(public_key, &private_key)
	})
	testify.Zero_Allocation(t, func() {
		ed25519.Sign(signature, &private_key, []byte("allocation"))
	})
	testify.Zero_Allocation(t, func() {
		verified = ed25519.Verify(
			public_key, []byte("allocation"), ed25519.Signature_Unvalidated(signature),
		)
	})
	testify.True(t, bool(verified))
}

// Test_Invariant_Domains reaches every bounded message and signature length.
func Test_Invariant_Domains(t *testing.T) {
	seed := vector_seed()
	var private_key ed25519.Private_Key
	ed25519.Private_Key_From_Seed(&private_key, seed)
	var public_storage [ed25519.PUBLIC_KEY_SIZE]byte
	public_key := ed25519.Public_Key(public_storage[:])
	ed25519.Public_Key_From_Private(public_key, &private_key)
	var signature_storage [ed25519.SIGNATURE_SIZE]byte
	signature := ed25519.Signature(signature_storage[:])
	var message [ed25519.MESSAGE_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		ed25519.MESSAGE_SIZE_MINIMUM,
		ed25519.MESSAGE_SIZE_MINIMUM + binary.UINT_8_SIZE,
		ed25519.MESSAGE_SIZE_MINIMUM + binary.UINT_16_SIZE,
		ed25519.MESSAGE_SIZE_MAXIMUM - binary.UINT_8_SIZE,
		ed25519.MESSAGE_SIZE_MAXIMUM,
	} {
		ed25519.Sign(signature, &private_key, message[:size])
		ed25519.Verify(
			public_key, message[:size], ed25519.Signature_Unvalidated(signature),
		)
	}
	var hostile [ed25519.SIGNATURE_UNVALIDATED_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		ed25519.SIGNATURE_UNVALIDATED_SIZE_MINIMUM,
		ed25519.SIGNATURE_UNVALIDATED_SIZE_MINIMUM + binary.UINT_8_SIZE,
		ed25519.SIGNATURE_UNVALIDATED_SIZE_MINIMUM + binary.UINT_16_SIZE,
		ed25519.SIGNATURE_UNVALIDATED_SIZE_MAXIMUM - binary.UINT_8_SIZE,
		ed25519.SIGNATURE_UNVALIDATED_SIZE_MAXIMUM,
	} {
		ed25519.Verify(public_key, nil, hostile[:size])
	}
	test_decision_domains(t, public_key)
}

func test_decision_domains(t *testing.T, public_key ed25519.Public_Key) {
	var identity_storage [ed25519.PUBLIC_KEY_SIZE]byte
	identity_storage[0] = byte(binary.UINT_8_SIZE)
	identity := ed25519.Public_Key(identity_storage[:])
	testify.False(t, bool(ed25519.Verify(
		identity, nil, ed25519.Signature_Unvalidated(vector_signature()),
	)))

	var invalid_public_storage [ed25519.PUBLIC_KEY_SIZE]byte
	for index := range invalid_public_storage {
		invalid_public_storage[index] = 0xff
	}
	invalid_public := ed25519.Public_Key(invalid_public_storage[:])
	testify.False(t, bool(ed25519.Verify(
		invalid_public, nil, ed25519.Signature_Unvalidated(vector_signature()),
	)))

	invalid_signature := vector_signature()
	for index := ed25519.PUBLIC_KEY_SIZE; index < len(invalid_signature); index++ {
		invalid_signature[index] = 0xff
	}
	testify.False(t, bool(ed25519.Verify(
		public_key, nil, ed25519.Signature_Unvalidated(invalid_signature),
	)))
}

func vector_seed() (seed ed25519.Seed) {
	return ed25519.Seed([]byte{
		0x9d, 0x61, 0xb1, 0x9d, 0xef, 0xfd, 0x5a, 0x60,
		0xba, 0x84, 0x4a, 0xf4, 0x92, 0xec, 0x2c, 0xc4,
		0x44, 0x49, 0xc5, 0x69, 0x7b, 0x32, 0x69, 0x19,
		0x70, 0x3b, 0xac, 0x03, 0x1c, 0xae, 0x7f, 0x60,
	})
}

func vector_public_key() (public_key ed25519.Public_Key) {
	return ed25519.Public_Key([]byte{
		0xd7, 0x5a, 0x98, 0x01, 0x82, 0xb1, 0x0a, 0xb7,
		0xd5, 0x4b, 0xfe, 0xd3, 0xc9, 0x64, 0x07, 0x3a,
		0x0e, 0xe1, 0x72, 0xf3, 0xda, 0xa6, 0x23, 0x25,
		0xaf, 0x02, 0x1a, 0x68, 0xf7, 0x07, 0x51, 0x1a,
	})
}

func vector_signature() (signature ed25519.Signature) {
	return ed25519.Signature([]byte{
		0xe5, 0x56, 0x43, 0x00, 0xc3, 0x60, 0xac, 0x72,
		0x90, 0x86, 0xe2, 0xcc, 0x80, 0x6e, 0x82, 0x8a,
		0x84, 0x87, 0x7f, 0x1e, 0xb8, 0xe5, 0xd9, 0x74,
		0xd8, 0x73, 0xe0, 0x65, 0x22, 0x49, 0x01, 0x55,
		0x5f, 0xb8, 0x82, 0x15, 0x90, 0xa3, 0x3b, 0xac,
		0xc6, 0x1e, 0x39, 0x70, 0x1c, 0xf9, 0xb4, 0x6b,
		0xd2, 0x5b, 0xf5, 0xf0, 0x59, 0x5b, 0xbe, 0x24,
		0x65, 0x51, 0x41, 0x43, 0x8e, 0x7a, 0x10, 0x0b,
	})
}
