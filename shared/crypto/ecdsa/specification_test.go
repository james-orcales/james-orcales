package ecdsa_test

import (
	standard_ecdsa "crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"testing"

	"local/james-orcales/shared/crypto/ecdsa"
	"local/james-orcales/shared/crypto/prng"
	"local/james-orcales/shared/encoding/asn1"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// Test_Keys binds the minimum private scalar and its public key to standard P-256.
func Test_Keys(t *testing.T) {
	private_encoding := private_scalar_minimum()
	var private_key ecdsa.Private_Key
	status := ecdsa.Private_Key_Set_Bytes(&private_key, private_encoding[:])
	testify.Equal(t, ecdsa.KEY_STATUS_OK, status)

	public_key := ecdsa.Public_Key_From_Private(private_key)
	var public_encoding ecdsa.Public_Key_Encoding
	ecdsa.Public_Key_Bytes_Into(&public_encoding, public_key)
	parameters := elliptic.P256().Params()
	want := elliptic.Marshal(
		elliptic.P256(), parameters.Gx, parameters.Gy,
	)
	testify.Equal(t, want, public_encoding[:])

	var parsed ecdsa.Public_Key
	status = ecdsa.Public_Key_Set_Bytes(&parsed, public_encoding[:])
	testify.Equal(t, ecdsa.KEY_STATUS_OK, status)
	var parsed_encoding ecdsa.Public_Key_Encoding
	ecdsa.Public_Key_Bytes_Into(&parsed_encoding, parsed)
	testify.Equal(t, public_encoding, parsed_encoding)
}

// Test_Signatures checks standard verification and owned tamper refusal.
func Test_Signatures(t *testing.T) {
	private_key := private_key_minimum(t)
	public_key := ecdsa.Public_Key_From_Private(private_key)
	digest := sha256.Sum256([]byte("bounded ECDSA"))
	generator := prng.New([prng.KEY_BYTES]byte{}, prng.CURSOR_MIN)
	source := prng.Chacha_To_Source(&generator)
	var signature ecdsa.Signature
	status := ecdsa.Sign(&signature, source, private_key, digest)
	testify.Equal(t, ecdsa.SIGN_STATUS_OK, status)

	parameters := elliptic.P256().Params()
	standard_public := standard_ecdsa.PublicKey{
		Curve: elliptic.P256(), X: parameters.Gx, Y: parameters.Gy,
	}
	standard_signature, standard_signature_size := signature_asn_1(signature)
	testify.True(t, standard_ecdsa.VerifyASN1(
		&standard_public, digest[:], standard_signature[:standard_signature_size],
	))

	signature[len(signature)-binary.UINT_8_SIZE] ^= byte(binary.UINT_8_SIZE)
	testify.False(t, bool(ecdsa.Verify(public_key, digest, signature[:])))
}

// Test_Bounds preserves key destinations and rejects bounded malformed signatures.
func Test_Bounds(t *testing.T) {
	private_key := private_key_minimum(t)
	private_before := private_key
	var zero [ecdsa.PRIVATE_KEY_SIZE]byte
	status := ecdsa.Private_Key_Set_Bytes(&private_key, zero[:])
	testify.Equal(t, ecdsa.KEY_STATUS_INPUT_INVALID, status)
	testify.Equal(t, private_before, private_key)

	public_key := ecdsa.Public_Key_From_Private(private_key)
	public_before := public_key
	status = ecdsa.Public_Key_Set_Bytes(&public_key, []byte{bits.WORD_8_MINIMUM})
	testify.Equal(t, ecdsa.KEY_STATUS_INPUT_INVALID, status)
	testify.Equal(t, public_before, public_key)

	digest := sha256.Sum256(nil)
	testify.False(t, bool(ecdsa.Verify(public_key, digest, nil)))
	var zero_signature ecdsa.Signature
	testify.False(t, bool(ecdsa.Verify(public_key, digest, zero_signature[:])))
	var empty_generator prng.Chacha
	empty_source := prng.Chacha_To_Source(&empty_generator)
	var signature ecdsa.Signature
	signature[bits.BIT_COUNT_MINIMUM] = bits.WORD_8_MAXIMUM
	signature_before := signature
	sign_status := ecdsa.Sign(
		&signature, empty_source, private_key, digest,
	)
	testify.Equal(t, ecdsa.SIGN_STATUS_ENTROPY_EXHAUSTED, sign_status)
	testify.Equal(t, signature_before, signature)

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
		ecdsa.Verify(public_key, digest, signature_oversized[:])
	})
}

// Test_Allocation measures every exported successful runtime operation.
func Test_Allocation(t *testing.T) {
	private_encoding := private_scalar_minimum()
	var private_key ecdsa.Private_Key
	var public_key, parsed ecdsa.Public_Key
	var public_encoding ecdsa.Public_Key_Encoding
	var signature ecdsa.Signature
	digest := sha256.Sum256([]byte("allocation"))
	generator := prng.New([prng.KEY_BYTES]byte{}, prng.CURSOR_MIN)
	source := prng.Chacha_To_Source(&generator)
	var key_status ecdsa.Key_Status
	var sign_status ecdsa.Sign_Status
	var verified ecdsa.Verification
	testify.Zero_Allocation(t, func() {
		key_status = ecdsa.Private_Key_Set_Bytes(&private_key, private_encoding[:])
	})
	testify.Zero_Allocation(t, func() {
		public_key = ecdsa.Public_Key_From_Private(private_key)
	})
	testify.Zero_Allocation(t, func() {
		ecdsa.Public_Key_Bytes_Into(&public_encoding, public_key)
	})
	testify.Zero_Allocation(t, func() {
		key_status = ecdsa.Public_Key_Set_Bytes(&parsed, public_encoding[:])
	})
	testify.Zero_Allocation(t, func() {
		sign_status = ecdsa.Sign(&signature, source, private_key, digest)
	})
	testify.Zero_Allocation(t, func() {
		verified = ecdsa.Verify(public_key, digest, signature[:])
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

	public_key = ecdsa.Public_Key_From_Private(private_key_minimum(t))
	digest := sha256.Sum256(nil)
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

	var output ecdsa.Signature
	for _, position := range [...]prng.Cursor{
		prng.CURSOR_MIN,
		prng.CURSOR_MIN + binary.UINT_8_SIZE,
		prng.CURSOR_MIN + binary.UINT_16_SIZE,
		prng.CURSOR_MAX,
	} {
		generator := prng.Chacha{Position: position}
		source := prng.Chacha_To_Source(&generator)
		testify.Panics(t, func() {
			ecdsa.Sign(&output, source, ecdsa.Private_Key{}, digest)
		})
	}
}

func private_scalar_minimum() (encoding [ecdsa.PRIVATE_KEY_SIZE]byte) {
	encoding[len(encoding)-binary.UINT_8_SIZE] = byte(binary.UINT_8_SIZE)
	return encoding
}

func private_key_minimum(t *testing.T) (private_key ecdsa.Private_Key) {
	t.Helper()
	encoding := private_scalar_minimum()
	status := ecdsa.Private_Key_Set_Bytes(&private_key, encoding[:])
	testify.Equal(t, ecdsa.KEY_STATUS_OK, status)
	return private_key
}

const ASN_1_ELEMENT_HEADER_SIZE = asn1.IDENTIFIER_SIZE_MINIMUM +
	asn1.CONTENT_SIZE_FIELD_SIZE_MINIMUM

const ASN_1_INTEGER_PADDING_SIZE_MAXIMUM = binary.UINT_8_SIZE

const ASN_1_SIGNATURE_OVERHEAD_SIZE_MAXIMUM = ASN_1_ELEMENT_HEADER_SIZE +
	binary.UINT_16_SIZE*(ASN_1_ELEMENT_HEADER_SIZE+ASN_1_INTEGER_PADDING_SIZE_MAXIMUM)

const ASN_1_INTEGER_IDENTIFIER byte = byte(binary.UINT_16_SIZE)

const ASN_1_SEQUENCE_IDENTIFIER byte = byte(asn1.CONSTRUCTED_MASK) |
	byte(bits.BIT_COUNT_16_MAXIMUM)

func signature_asn_1(
	signature ecdsa.Signature,
) (
	encoding [ecdsa.SIGNATURE_SIZE + ASN_1_SIGNATURE_OVERHEAD_SIZE_MAXIMUM]byte,
	encoding_size int,
) {
	encoding_size = ASN_1_ELEMENT_HEADER_SIZE
	encoding_size += integer_asn_1_into(
		encoding[encoding_size:], signature[:ecdsa.SCALAR_SIZE],
	)
	encoding_size += integer_asn_1_into(
		encoding[encoding_size:], signature[ecdsa.SCALAR_SIZE:],
	)
	encoding[bits.BIT_COUNT_MINIMUM] = ASN_1_SEQUENCE_IDENTIFIER
	encoding[asn1.IDENTIFIER_SIZE_MINIMUM] = byte(
		encoding_size - ASN_1_ELEMENT_HEADER_SIZE,
	)
	return encoding, encoding_size
}

func integer_asn_1_into(destination []byte, scalar []byte) (encoding_size int) {
	for len(scalar) > binary.UINT_8_SIZE &&
		scalar[bits.BIT_COUNT_MINIMUM] == bits.WORD_8_MINIMUM {
		scalar = scalar[binary.UINT_8_SIZE:]
	}
	leading_zero := scalar[bits.BIT_COUNT_MINIMUM]>>
		(bits.BIT_COUNT_8_MAXIMUM-binary.UINT_8_SIZE) == byte(binary.UINT_8_SIZE)
	destination[bits.BIT_COUNT_MINIMUM] = ASN_1_INTEGER_IDENTIFIER
	destination[asn1.IDENTIFIER_SIZE_MINIMUM] = byte(len(scalar))
	encoding_size = ASN_1_ELEMENT_HEADER_SIZE
	if leading_zero {
		destination[asn1.IDENTIFIER_SIZE_MINIMUM] +=
			byte(ASN_1_INTEGER_PADDING_SIZE_MAXIMUM)
		destination[encoding_size] = bits.WORD_8_MINIMUM
		encoding_size += ASN_1_INTEGER_PADDING_SIZE_MAXIMUM
	}
	copy(destination[encoding_size:], scalar)
	return encoding_size + len(scalar)
}
