package rsa_test

import (
	"crypto"
	standard_rsa "crypto/rsa"
	"crypto/sha256"
	"testing"

	"local/james-orcales/shared/crypto/rsa"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/random/csprng"
	"local/james-orcales/shared/testify"
)

// Test_Keys validates standard RSA-2048 key material.
func Test_Keys(t *testing.T) {
	standard_private := standard_private_key(t)
	private_key, public_key := owned_keys(t, standard_private)
	var modulus rsa.Modulus
	rsa.Public_Key_Bytes_Into(&modulus, public_key)
	testify.Equal(t, rsa.Modulus(padded_integer(standard_private.N.Bytes())), modulus)
	derived := rsa.Public_Key_From_Private(private_key)
	testify.Equal(t, public_key, derived)
}

// Test_Encryption proves OAEP interoperability in both directions.
func Test_Encryption(t *testing.T) {
	standard_private := standard_private_key(t)
	private_key, public_key := owned_keys(t, standard_private)
	message := []byte("bounded RSA OAEP")
	generator := csprng.New([csprng.KEY_BYTES]byte{}, csprng.CURSOR_MIN)
	var ciphertext rsa.Ciphertext
	rsa.Encrypt_OAEP_SHA_256(&ciphertext, &generator, public_key, message)
	plaintext, err := standard_rsa.DecryptOAEP(
		sha256.New(), nil, standard_private, ciphertext[:], nil,
	)
	testify.No_Error(t, err)
	testify.Equal(t, message, plaintext)

	standard_generator := csprng.New([csprng.KEY_BYTES]byte{}, csprng.CURSOR_MIN)
	standard_ciphertext, err := standard_rsa.EncryptOAEP(
		sha256.New(), &standard_generator, &standard_private.PublicKey, message, nil,
	)
	testify.No_Error(t, err)
	var output [rsa.MESSAGE_SIZE_MAXIMUM]byte
	count, status := rsa.Decrypt_OAEP_SHA_256(
		output[:], private_key, standard_ciphertext,
	)
	testify.Equal(t, rsa.DECRYPT_STATUS_OK, status)
	testify.Equal(t, message, output[:count])
}

// Test_Signatures proves PSS and PKCS1 v1.5 interoperability.
func Test_Signatures(t *testing.T) {
	standard_private := standard_private_key(t)
	private_key, public_key := owned_keys(t, standard_private)
	digest := sha256.Sum256([]byte("bounded RSA signatures"))
	generator := csprng.New([csprng.KEY_BYTES]byte{}, csprng.CURSOR_MIN)
	var signature rsa.Signature
	rsa.Sign_PSS_SHA_256(&signature, &generator, private_key, digest)
	err := standard_rsa.VerifyPSS(
		&standard_private.PublicKey, crypto.SHA256, digest[:], signature[:], nil,
	)
	testify.No_Error(t, err)
	testify.True(t, bool(rsa.Verify_PSS_SHA_256(public_key, digest, signature[:])))

	rsa.Sign_PKCS1_V1_5_SHA_256(&signature, private_key, digest)
	err = standard_rsa.VerifyPKCS1v15(
		&standard_private.PublicKey, crypto.SHA256, digest[:], signature[:],
	)
	testify.No_Error(t, err)
	testify.True(t, bool(rsa.Verify_PKCS1_V1_5_SHA_256(
		public_key, digest, signature[:],
	)))
	signature[len(signature)-binary.UINT_8_SIZE] ^= byte(binary.UINT_8_SIZE)
	testify.False(t, bool(rsa.Verify_PKCS1_V1_5_SHA_256(
		public_key, digest, signature[:],
	)))
}

// Test_Bounds rejects hostile sizes and preserves output on refusal.
func Test_Bounds(t *testing.T) {
	standard_private := standard_private_key(t)
	private_key, public_key := owned_keys(t, standard_private)
	generator := csprng.New([csprng.KEY_BYTES]byte{}, csprng.CURSOR_MIN)
	var ciphertext rsa.Ciphertext
	var message_oversized [rsa.MESSAGE_SIZE_MAXIMUM + binary.UINT_8_SIZE]byte
	testify.Panics(t, func() {
		rsa.Encrypt_OAEP_SHA_256(
			&ciphertext, &generator, public_key, message_oversized[:],
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

	rsa.Encrypt_OAEP_SHA_256(&ciphertext, &generator, public_key, []byte("short"))
	var short [rsa.DESTINATION_SIZE_MINIMUM + binary.UINT_8_SIZE]byte
	short[bits.BIT_COUNT_MINIMUM] = bits.WORD_8_MAXIMUM
	count, status = rsa.Decrypt_OAEP_SHA_256(short[:], private_key, ciphertext[:])
	testify.Equal(t, rsa.Count(len("short")), count)
	testify.Equal(t, rsa.DECRYPT_STATUS_DESTINATION_TOO_SMALL, status)
	testify.Equal(t, byte(bits.WORD_8_MAXIMUM), short[bits.BIT_COUNT_MINIMUM])

	digest := sha256.Sum256(nil)
	var signature_oversized [rsa.SIGNATURE_UNVALIDATED_SIZE_MAXIMUM +
		binary.UINT_8_SIZE]byte
	testify.Panics(t, func() {
		rsa.Verify_PSS_SHA_256(public_key, digest, signature_oversized[:])
	})
	testify.False(t, bool(rsa.Verify_PSS_SHA_256(public_key, digest, nil)))
}

// Test_Allocation measures every exported successful runtime operation.
func Test_Allocation(t *testing.T) {
	standard_private := standard_private_key(t)
	modulus_encoding := padded_integer(standard_private.N.Bytes())
	private_exponent := padded_integer(standard_private.D.Bytes())
	var private_key rsa.Private_Key
	var public_key, derived_public_key rsa.Public_Key
	var modulus rsa.Modulus
	var private_key_status, public_key_status rsa.Key_Status
	testify.Zero_Allocation(t, func() {
		public_key_status = rsa.Public_Key_Set_Bytes(&public_key, modulus_encoding[:])
	})
	testify.Zero_Allocation(t, func() {
		private_key_status = rsa.Private_Key_Set_Bytes(
			&private_key, modulus_encoding[:], private_exponent[:],
		)
	})
	testify.Zero_Allocation(t, func() {
		derived_public_key = rsa.Public_Key_From_Private(private_key)
	})
	testify.Zero_Allocation(t, func() {
		rsa.Public_Key_Bytes_Into(&modulus, public_key)
	})
	message := []byte("allocation")
	digest := sha256.Sum256(message)
	generator := csprng.New([csprng.KEY_BYTES]byte{}, csprng.CURSOR_MIN)
	var ciphertext rsa.Ciphertext
	var signature rsa.Signature
	var output [rsa.MESSAGE_SIZE_MAXIMUM]byte
	var count rsa.Count
	var decrypt_status rsa.Decrypt_Status
	var verified rsa.Verification
	testify.Zero_Allocation(t, func() {
		rsa.Encrypt_OAEP_SHA_256(&ciphertext, &generator, public_key, message)
	})
	testify.Zero_Allocation(t, func() {
		count, decrypt_status = rsa.Decrypt_OAEP_SHA_256(
			output[:], private_key, ciphertext[:],
		)
	})
	testify.Zero_Allocation(t, func() {
		rsa.Sign_PSS_SHA_256(&signature, &generator, private_key, digest)
	})
	testify.Zero_Allocation(t, func() {
		verified = rsa.Verify_PSS_SHA_256(public_key, digest, signature[:])
	})
	testify.Zero_Allocation(t, func() {
		rsa.Sign_PKCS1_V1_5_SHA_256(&signature, private_key, digest)
	})
	testify.Zero_Allocation(t, func() {
		verified = rsa.Verify_PKCS1_V1_5_SHA_256(public_key, digest, signature[:])
	})
	testify.Equal(t, rsa.Count(len(message)), count)
	testify.Equal(t, rsa.KEY_STATUS_OK, private_key_status)
	testify.Equal(t, rsa.KEY_STATUS_OK, public_key_status)
	testify.Equal(t, public_key, derived_public_key)
	testify.Equal(t, rsa.Modulus(modulus_encoding), modulus)
	testify.Equal(t, rsa.DECRYPT_STATUS_OK, decrypt_status)
	testify.True(t, bool(verified))
}

// Test_Invariant_Domains reaches every bounded input, output, count, and generator value.
func Test_Invariant_Domains(t *testing.T) {
	test_key_domains()
	standard_private := standard_private_key(t)
	private_key, public_key := owned_keys(t, standard_private)
	test_encryption_domains(private_key, public_key)
	test_signature_domains(public_key)
	test_generator_domains(t)
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
	generator := csprng.New([csprng.KEY_BYTES]byte{}, csprng.CURSOR_MIN)
	var ciphertext rsa.Ciphertext
	var message [rsa.MESSAGE_SIZE_MAXIMUM]byte
	var output [rsa.MESSAGE_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		rsa.MESSAGE_SIZE_MINIMUM,
		rsa.MESSAGE_SIZE_MINIMUM + binary.UINT_8_SIZE,
		rsa.MESSAGE_SIZE_MINIMUM + binary.UINT_16_SIZE,
		rsa.MESSAGE_SIZE_MAXIMUM - binary.UINT_8_SIZE,
		rsa.MESSAGE_SIZE_MAXIMUM,
	} {
		rsa.Encrypt_OAEP_SHA_256(&ciphertext, &generator, public_key, message[:size])
		rsa.Decrypt_OAEP_SHA_256(output[:], private_key, ciphertext[:])
	}
	for _, size := range [...]int{
		rsa.DESTINATION_SIZE_MINIMUM,
		rsa.DESTINATION_SIZE_MINIMUM + binary.UINT_8_SIZE,
		rsa.DESTINATION_SIZE_MINIMUM + binary.UINT_16_SIZE,
		rsa.DESTINATION_SIZE_MAXIMUM - binary.UINT_8_SIZE,
		rsa.DESTINATION_SIZE_MAXIMUM,
	} {
		rsa.Decrypt_OAEP_SHA_256(output[:size], private_key, ciphertext[:])
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
	digest := sha256.Sum256(nil)
	var hostile_signature [rsa.SIGNATURE_UNVALIDATED_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		rsa.SIGNATURE_UNVALIDATED_SIZE_MINIMUM,
		rsa.SIGNATURE_UNVALIDATED_SIZE_MINIMUM + binary.UINT_8_SIZE,
		rsa.SIGNATURE_UNVALIDATED_SIZE_MINIMUM + binary.UINT_16_SIZE,
		rsa.SIGNATURE_UNVALIDATED_SIZE_MAXIMUM - binary.UINT_8_SIZE,
		rsa.SIGNATURE_UNVALIDATED_SIZE_MAXIMUM,
	} {
		rsa.Verify_PSS_SHA_256(public_key, digest, hostile_signature[:size])
		rsa.Verify_PKCS1_V1_5_SHA_256(public_key, digest, hostile_signature[:size])
	}
}

func test_generator_domains(t *testing.T) {
	t.Helper()
	digest := sha256.Sum256(nil)
	var ciphertext rsa.Ciphertext
	var signature rsa.Signature
	for _, position := range [...]csprng.Cursor{
		csprng.CURSOR_MIN,
		csprng.CURSOR_MIN + binary.UINT_8_SIZE,
		csprng.CURSOR_MIN + binary.UINT_16_SIZE,
		csprng.CURSOR_MAX,
	} {
		empty_generator := csprng.Generator{Position: position}
		testify.Panics(t, func() {
			rsa.Encrypt_OAEP_SHA_256(
				&ciphertext, &empty_generator, rsa.Public_Key{}, nil,
			)
		})
		testify.Panics(t, func() {
			rsa.Sign_PSS_SHA_256(
				&signature, &empty_generator, rsa.Private_Key{}, digest,
			)
		})
	}
}

func standard_private_key(t *testing.T) (private_key *standard_rsa.PrivateKey) {
	t.Helper()
	generator := csprng.New([csprng.KEY_BYTES]byte{}, csprng.CURSOR_MIN)
	private_key, err := standard_rsa.GenerateKey(&generator, rsa.MODULUS_BIT_COUNT)
	testify.No_Error(t, err)
	return private_key
}

func owned_keys(
	t *testing.T, standard_private *standard_rsa.PrivateKey,
) (private_key rsa.Private_Key, public_key rsa.Public_Key) {
	t.Helper()
	modulus := padded_integer(standard_private.N.Bytes())
	private_exponent := padded_integer(standard_private.D.Bytes())
	status := rsa.Private_Key_Set_Bytes(
		&private_key, modulus[:], private_exponent[:],
	)
	testify.Equal(t, rsa.KEY_STATUS_OK, status)
	status = rsa.Public_Key_Set_Bytes(&public_key, modulus[:])
	testify.Equal(t, rsa.KEY_STATUS_OK, status)
	return private_key, public_key
}

func padded_integer(source []byte) (encoding [rsa.MODULUS_SIZE]byte) {
	copy(encoding[len(encoding)-len(source):], source)
	return encoding
}
