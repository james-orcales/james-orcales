// Package rsa implements bounded caller-owned RSA-2048 operations.
package rsa

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/crypto/prng"
	"local/james-orcales/shared/crypto/sha256"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
)

// MODULUS_LIMB_COUNT fixes RSA-2048 storage to 32 machine words.
const MODULUS_LIMB_COUNT = bits.BIT_COUNT_32_MAXIMUM

// MODULUS_BIT_COUNT derives the exact supported modulus width.
const MODULUS_BIT_COUNT = MODULUS_LIMB_COUNT * bits.BIT_COUNT_64_MAXIMUM

// MODULUS_SIZE derives the RSA-2048 wire width.
const MODULUS_SIZE = MODULUS_BIT_COUNT / bits.BIT_COUNT_8_MAXIMUM

// PUBLIC_EXPONENT is the fixed interoperable Fermat exponent 65537.
const PUBLIC_EXPONENT = uint32(binary.UINT_8_SIZE<<bits.BIT_COUNT_16_MAXIMUM) +
	binary.UINT_8_SIZE

// PUBLIC_EXPONENT_BIT_COUNT scans the complete fixed exponent storage.
const PUBLIC_EXPONENT_BIT_COUNT = bits.BIT_COUNT_32_MAXIMUM

// HASH_SIZE is the OAEP, PSS, and PKCS1 v1.5 SHA-256 width.
const HASH_SIZE = sha256.DIGEST_256_SIZE

// OAEP_OVERHEAD_SIZE holds two hashes and two framing octets.
const OAEP_OVERHEAD_SIZE = HASH_SIZE + HASH_SIZE + binary.UINT_16_SIZE

// MESSAGE_SIZE_MINIMUM admits an empty OAEP plaintext.
const MESSAGE_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// MESSAGE_SIZE_MAXIMUM is the RFC 8017 OAEP capacity for RSA-2048 and SHA-256.
const MESSAGE_SIZE_MAXIMUM = MODULUS_SIZE - OAEP_OVERHEAD_SIZE

// OAEP_DATABASE_SIZE excludes the zero prefix and masked seed.
const OAEP_DATABASE_SIZE = MODULUS_SIZE - binary.UINT_8_SIZE - HASH_SIZE

// PSS_DATABASE_SIZE excludes the digest and trailer.
const PSS_DATABASE_SIZE = MODULUS_SIZE - HASH_SIZE - binary.UINT_8_SIZE

// PSS_SALT_SIZE uses one digest-width random salt.
const PSS_SALT_SIZE = HASH_SIZE

// PSS_PREFIX_SIZE is the RFC 8017 eight-zero hash prefix.
const PSS_PREFIX_SIZE = binary.UINT_64_SIZE

// PSS_PADDING_SIZE fills the database before its delimiter and salt.
const PSS_PADDING_SIZE = PSS_DATABASE_SIZE - PSS_SALT_SIZE - binary.UINT_8_SIZE

// MASK_COUNTER_SIZE is MGF1's 32-bit counter encoding.
const MASK_COUNTER_SIZE = binary.UINT_32_SIZE

// MASK_BLOCK_COUNT covers the longest database mask.
const MASK_BLOCK_COUNT = (PSS_DATABASE_SIZE + HASH_SIZE - binary.UINT_8_SIZE) / HASH_SIZE

// MODULUS_UNVALIDATED_SIZE_MINIMUM admits empty hostile input.
const MODULUS_UNVALIDATED_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// MODULUS_UNVALIDATED_SIZE_MAXIMUM admits one invalid byte past exact width.
const MODULUS_UNVALIDATED_SIZE_MAXIMUM = MODULUS_SIZE + binary.UINT_8_SIZE

// PRIVATE_EXPONENT_UNVALIDATED_SIZE_MINIMUM admits empty hostile input.
const PRIVATE_EXPONENT_UNVALIDATED_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// PRIVATE_EXPONENT_UNVALIDATED_SIZE_MAXIMUM admits one invalid byte past exact width.
const PRIVATE_EXPONENT_UNVALIDATED_SIZE_MAXIMUM = MODULUS_SIZE + binary.UINT_8_SIZE

// CIPHERTEXT_UNVALIDATED_SIZE_MINIMUM admits empty hostile input.
const CIPHERTEXT_UNVALIDATED_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// CIPHERTEXT_UNVALIDATED_SIZE_MAXIMUM admits one invalid byte past exact width.
const CIPHERTEXT_UNVALIDATED_SIZE_MAXIMUM = MODULUS_SIZE + binary.UINT_8_SIZE

// SIGNATURE_UNVALIDATED_SIZE_MINIMUM admits empty hostile input.
const SIGNATURE_UNVALIDATED_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// SIGNATURE_UNVALIDATED_SIZE_MAXIMUM admits one invalid byte past exact width.
const SIGNATURE_UNVALIDATED_SIZE_MAXIMUM = MODULUS_SIZE + binary.UINT_8_SIZE

// DESTINATION_SIZE_MINIMUM admits caller output too short for plaintext.
const DESTINATION_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// DESTINATION_SIZE_MAXIMUM holds the longest OAEP plaintext.
const DESTINATION_SIZE_MAXIMUM = MESSAGE_SIZE_MAXIMUM

// COUNT_MINIMUM reports empty plaintext.
const COUNT_MINIMUM Count = MESSAGE_SIZE_MINIMUM

// COUNT_MAXIMUM reports the longest plaintext.
const COUNT_MAXIMUM Count = MESSAGE_SIZE_MAXIMUM

// KEY_STATUS_OK means validated key material committed.
const KEY_STATUS_OK Key_Status = Key_Status(bits.WORD_8_MINIMUM)

// KEY_STATUS_INPUT_INVALID leaves key storage unchanged.
const KEY_STATUS_INPUT_INVALID Key_Status = KEY_STATUS_OK + binary.UINT_8_SIZE

// DECRYPT_STATUS_OK means complete plaintext committed.
const DECRYPT_STATUS_OK Decrypt_Status = Decrypt_Status(bits.WORD_8_MINIMUM)

// DECRYPT_STATUS_INPUT_INVALID leaves plaintext storage unchanged.
const DECRYPT_STATUS_INPUT_INVALID Decrypt_Status = DECRYPT_STATUS_OK + binary.UINT_8_SIZE

// DECRYPT_STATUS_SMALL is the derived short-output status value.
const DECRYPT_STATUS_SMALL Decrypt_Status = DECRYPT_STATUS_INPUT_INVALID + DECRYPT_STATUS_STEP

// DECRYPT_STATUS_DESTINATION_TOO_SMALL leaves plaintext storage unchanged.
const DECRYPT_STATUS_DESTINATION_TOO_SMALL = DECRYPT_STATUS_SMALL

// DECRYPT_STATUS_STEP separates each contiguous outcome.
const DECRYPT_STATUS_STEP = binary.UINT_8_SIZE

// READY_INDEX stores caller key-state identity.
const READY_INDEX = bits.BIT_COUNT_MINIMUM

// READY_WORD_COUNT holds one key-state octet.
const READY_WORD_COUNT = READY_INDEX + binary.UINT_8_SIZE

// READY_EMPTY marks caller storage without validated key material.
const READY_EMPTY byte = bits.WORD_8_MINIMUM

// READY_COMPLETE marks validated key material.
const READY_COMPLETE byte = READY_EMPTY + binary.UINT_8_SIZE

// CONDITION_LIMB_COUNT gives branchless decisions fixed array identity.
const CONDITION_LIMB_COUNT = binary.UINT_8_SIZE

// MONTGOMERY_TEMPORARY_LIMB_COUNT holds one double carry above the modulus.
const MONTGOMERY_TEMPORARY_LIMB_COUNT = MODULUS_LIMB_COUNT + binary.UINT_16_SIZE

// MONTGOMERY_INVERSE_COUNT doubles one inverse bit six times to 64 bits.
const MONTGOMERY_INVERSE_COUNT = binary.UINT_64_SIZE - binary.UINT_16_SIZE

// PSS_TRAILER is the RFC 8017 trailer field.
const PSS_TRAILER byte = 0xbc

// BLOCK_TYPE_SIGNATURE is PKCS1 v1.5's signature block marker.
const BLOCK_TYPE_SIGNATURE byte = binary.UINT_8_SIZE

// PADDING_BYTE_SIGNATURE is PKCS1 v1.5's required all-ones padding.
const PADDING_BYTE_SIGNATURE byte = bits.WORD_8_MAXIMUM

// PKCS1_SHA_256_PREFIX is the DER DigestInfo prefix before 32 digest bytes.
const PKCS1_SHA_256_PREFIX = "\x30\x31\x30\x0d\x06\x09\x60\x86\x48\x01" +
	"\x65\x03\x04\x02\x01\x05\x00\x04\x20"

// PKCS1_SHA_256_PREFIX_SIZE follows the authoritative DER octets.
const PKCS1_SHA_256_PREFIX_SIZE = len(PKCS1_SHA_256_PREFIX)

// Modulus is one exact RSA-2048 big-endian modulus.
type Modulus [MODULUS_SIZE]byte

// Modulus_Invariants fixes modulus storage width.
func Modulus_Invariants(value Modulus, _ aver.Namespace) {
	aver.Always(len(value) == MODULUS_SIZE, "An RSA modulus has exact 2048-bit storage.")
}

// Private_Exponent is one exact-width private exponent.
type Private_Exponent [MODULUS_SIZE]byte

// Private_Exponent_Invariants fixes private exponent storage width.
func Private_Exponent_Invariants(value Private_Exponent, _ aver.Namespace) {
	aver.Always(
		len(value) == MODULUS_SIZE,
		"An RSA private exponent has modulus width.",
	)
}

// Ready stores caller key-state identity.
type Ready [READY_WORD_COUNT]byte

// Ready_Invariants fixes key-state storage width.
func Ready_Invariants(value Ready, _ aver.Namespace) {
	aver.Always(len(value) == READY_WORD_COUNT, "RSA key state has fixed width.")
}

// Public_Key stores validated modulus authority for fixed exponent 65537.
type Public_Key struct {
	// Modulus is the exact-width odd RSA-2048 modulus.
	Modulus Modulus
	// Ready proves modulus validation completed.
	Ready Ready
}

// Public_Key_Invariants relates complete state to a valid supported modulus.
func Public_Key_Invariants(value Public_Key, namespace aver.Namespace) {
	Modulus_Invariants(value.Modulus, namespace)
	Ready_Invariants(value.Ready, namespace)
	aver.Always(
		value.Ready[READY_INDEX] <= READY_COMPLETE,
		"An RSA public key has empty or complete state.",
	)
	valid := modulus_encoding_valid(&value.Modulus)
	aver.Always(
		uint64(value.Ready[READY_INDEX])&valid[bits.BIT_COUNT_MINIMUM] ==
			uint64(value.Ready[READY_INDEX]),
		"A complete RSA public key has an odd 2048-bit modulus.",
	)
}

// Private_Key stores validated modulus and private exponent authority.
type Private_Key struct {
	// Modulus is the exact-width odd RSA-2048 modulus.
	Modulus Modulus
	// Exponent is the exact-width canonical private exponent.
	Exponent Private_Exponent
	// Ready proves both integers passed validation.
	Ready Ready
}

// Private_Key_Invariants relates complete state to valid bounded integers.
func Private_Key_Invariants(value Private_Key, namespace aver.Namespace) {
	Modulus_Invariants(value.Modulus, namespace)
	Private_Exponent_Invariants(value.Exponent, namespace)
	Ready_Invariants(value.Ready, namespace)
	aver.Always(
		value.Ready[READY_INDEX] <= READY_COMPLETE,
		"An RSA private key has empty or complete state.",
	)
	modulus_valid := modulus_encoding_valid(&value.Modulus)
	exponent_valid := private_exponent_encoding_valid(&value.Exponent, &value.Modulus)
	valid := modulus_valid[bits.BIT_COUNT_MINIMUM] &
		exponent_valid[bits.BIT_COUNT_MINIMUM]
	aver.Always(
		uint64(value.Ready[READY_INDEX])&valid ==
			uint64(value.Ready[READY_INDEX]),
		"A complete RSA private key has valid bounded integers.",
	)
}

// Public_Key_Destination is nonnil caller-owned public key storage.
type Public_Key_Destination *Public_Key

// Public_Key_Destination_Invariants proves caller storage exists.
func Public_Key_Destination_Invariants(
	value Public_Key_Destination, _ aver.Namespace,
) {
	aver.Always(value != nil, "An RSA public key destination exists.")
	aver.Always(
		len(value.Modulus) == MODULUS_SIZE,
		"An RSA public key destination has modulus storage.",
	)
	aver.Always(
		len(value.Ready) == READY_WORD_COUNT,
		"An RSA public key destination has state storage.",
	)
}

// Private_Key_Destination is nonnil caller-owned private key storage.
type Private_Key_Destination *Private_Key

// Private_Key_Destination_Invariants proves caller storage exists.
func Private_Key_Destination_Invariants(
	value Private_Key_Destination, _ aver.Namespace,
) {
	aver.Always(value != nil, "An RSA private key destination exists.")
	aver.Always(
		len(value.Modulus) == MODULUS_SIZE,
		"An RSA private key destination has modulus storage.",
	)
	aver.Always(
		len(value.Exponent) == MODULUS_SIZE,
		"An RSA private key destination has exponent storage.",
	)
	aver.Always(
		len(value.Ready) == READY_WORD_COUNT,
		"An RSA private key destination has state storage.",
	)
}

// Modulus_Destination is nonnil caller-owned modulus output storage.
type Modulus_Destination *Modulus

// Modulus_Destination_Invariants proves caller storage exists.
func Modulus_Destination_Invariants(
	value Modulus_Destination, _ aver.Namespace,
) {
	aver.Always(value != nil, "An RSA modulus destination exists.")
}

// Modulus_Unvalidated is one bounded hostile modulus encoding.
type Modulus_Unvalidated []byte

// Modulus_Unvalidated_Invariants bounds public integer parsing.
func Modulus_Unvalidated_Invariants(
	value Modulus_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), MODULUS_UNVALIDATED_SIZE_MINIMUM,
			MODULUS_UNVALIDATED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Private_Exponent_Unvalidated is one bounded hostile exponent encoding.
type Private_Exponent_Unvalidated []byte

// Private_Exponent_Unvalidated_Invariants bounds private integer parsing.
func Private_Exponent_Unvalidated_Invariants(
	value Private_Exponent_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), PRIVATE_EXPONENT_UNVALIDATED_SIZE_MINIMUM,
			PRIVATE_EXPONENT_UNVALIDATED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Ciphertext is one exact-width RSA-2048 encrypted message.
type Ciphertext [MODULUS_SIZE]byte

// Ciphertext_Invariants fixes caller-owned ciphertext width.
func Ciphertext_Invariants(value Ciphertext, _ aver.Namespace) {
	aver.Always(len(value) == MODULUS_SIZE, "An RSA ciphertext has modulus width.")
}

// Ciphertext_Destination is nonnil caller-owned ciphertext storage.
type Ciphertext_Destination *Ciphertext

// Ciphertext_Destination_Invariants proves caller storage exists.
func Ciphertext_Destination_Invariants(
	value Ciphertext_Destination, _ aver.Namespace,
) {
	aver.Always(value != nil, "An RSA ciphertext destination exists.")
}

// Ciphertext_Unvalidated is one bounded hostile ciphertext.
type Ciphertext_Unvalidated []byte

// Ciphertext_Unvalidated_Invariants bounds private-operation parsing.
func Ciphertext_Unvalidated_Invariants(
	value Ciphertext_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), CIPHERTEXT_UNVALIDATED_SIZE_MINIMUM,
			CIPHERTEXT_UNVALIDATED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Signature is one exact-width RSA-2048 signature.
type Signature [MODULUS_SIZE]byte

// Signature_Invariants fixes caller-owned signature width.
func Signature_Invariants(value Signature, _ aver.Namespace) {
	aver.Always(len(value) == MODULUS_SIZE, "An RSA signature has modulus width.")
}

// Signature_Destination is nonnil caller-owned signature storage.
type Signature_Destination *Signature

// Signature_Destination_Invariants proves caller storage exists.
func Signature_Destination_Invariants(
	value Signature_Destination, _ aver.Namespace,
) {
	aver.Always(value != nil, "An RSA signature destination exists.")
}

// Signature_Unvalidated is one bounded hostile signature.
type Signature_Unvalidated []byte

// Signature_Unvalidated_Invariants bounds public-operation parsing.
func Signature_Unvalidated_Invariants(
	value Signature_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), SIGNATURE_UNVALIDATED_SIZE_MINIMUM,
			SIGNATURE_UNVALIDATED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Message is one OAEP-bounded plaintext.
type Message []byte

// Message_Invariants bounds OAEP encoding work.
func Message_Invariants(value Message, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), MESSAGE_SIZE_MINIMUM, MESSAGE_SIZE_MAXIMUM).
		Ensure()
}

// Destination is bounded caller-owned plaintext storage.
type Destination []byte

// Destination_Invariants bounds plaintext commit storage.
func Destination_Invariants(value Destination, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), DESTINATION_SIZE_MINIMUM, DESTINATION_SIZE_MAXIMUM).
		Ensure()
}

// Digest is one SHA-256 digest.
type Digest [HASH_SIZE]byte

// Digest_Invariants fixes signature digest width.
func Digest_Invariants(value Digest, _ aver.Namespace) {
	aver.Always(len(value) == HASH_SIZE, "An RSA digest has SHA-256 width.")
}

// Count is one complete plaintext byte count.
type Count int

// Count_Invariants spans empty through maximum OAEP plaintext.
func Count_Invariants(value Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), int(COUNT_MINIMUM), int(COUNT_MAXIMUM)).
		Ensure()
}

// Key_Status reports bounded key parsing.
type Key_Status uint8

// Key_Status_Invariants covers committed and refused key input.
func Key_Status_Invariants(value Key_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(KEY_STATUS_OK), uint8(KEY_STATUS_INPUT_INVALID)).
		Ensure()
}

// Decrypt_Status reports padding validity and caller capacity.
type Decrypt_Status uint8

// Decrypt_Status_Invariants covers every transactional decryption outcome.
func Decrypt_Status_Invariants(value Decrypt_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(DECRYPT_STATUS_OK),
			uint8(DECRYPT_STATUS_INPUT_INVALID),
			uint8(DECRYPT_STATUS_DESTINATION_TOO_SMALL),
		).
		Ensure()
}

// Verification reports signature validity.
type Verification bool

// Verification_Invariants covers accepted and refused signatures.
func Verification_Invariants(value Verification, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "An RSA signature verifies.").
		Ensure()
}

// Public_Key_Set_Bytes validates a supported modulus before replacement.
func Public_Key_Set_Bytes(
	destination Public_Key_Destination, modulus Modulus_Unvalidated,
) (status Key_Status) {
	defer func() {
		Key_Status_Invariants(status, "Public_Key_Set_Bytes.status")
		Public_Key_Invariants(*destination, "Public_Key_Set_Bytes.destination.output")
	}()
	Public_Key_Destination_Invariants(destination, "Public_Key_Set_Bytes.destination")
	Modulus_Unvalidated_Invariants(modulus, "Public_Key_Set_Bytes.modulus")
	if len(modulus) > MODULUS_UNVALIDATED_SIZE_MAXIMUM {
		panic("rsa: modulus exceeds bound")
	}
	if len(modulus) != MODULUS_SIZE {
		return KEY_STATUS_INPUT_INVALID
	}
	var encoding Modulus
	copy(encoding[:], modulus)
	if modulus_encoding_valid(&encoding)[bits.BIT_COUNT_MINIMUM] != binary.UINT_8_SIZE {
		return KEY_STATUS_INPUT_INVALID
	}
	*destination = Public_Key{
		Modulus: encoding, Ready: Ready{READY_COMPLETE},
	}
	return KEY_STATUS_OK
}

// Private_Key_Set_Bytes validates both private integers before replacement.
func Private_Key_Set_Bytes(
	destination Private_Key_Destination,
	modulus Modulus_Unvalidated,
	exponent Private_Exponent_Unvalidated,
) (status Key_Status) {
	defer func() {
		Key_Status_Invariants(status, "Private_Key_Set_Bytes.status")
		Private_Key_Invariants(*destination, "Private_Key_Set_Bytes.destination.output")
	}()
	Private_Key_Destination_Invariants(destination, "Private_Key_Set_Bytes.destination")
	Modulus_Unvalidated_Invariants(modulus, "Private_Key_Set_Bytes.modulus")
	Private_Exponent_Unvalidated_Invariants(exponent, "Private_Key_Set_Bytes.exponent")
	if len(modulus) > MODULUS_UNVALIDATED_SIZE_MAXIMUM {
		panic("rsa: private modulus exceeds bound")
	}
	if len(exponent) > PRIVATE_EXPONENT_UNVALIDATED_SIZE_MAXIMUM {
		panic("rsa: private exponent exceeds bound")
	}
	if len(modulus) != MODULUS_SIZE {
		return KEY_STATUS_INPUT_INVALID
	}
	if len(exponent) != MODULUS_SIZE {
		return KEY_STATUS_INPUT_INVALID
	}
	var modulus_encoding Modulus
	var exponent_encoding Private_Exponent
	copy(modulus_encoding[:], modulus)
	copy(exponent_encoding[:], exponent)
	modulus_valid := modulus_encoding_valid(&modulus_encoding)
	exponent_valid := private_exponent_encoding_valid(
		&exponent_encoding, &modulus_encoding,
	)
	if modulus_valid[bits.BIT_COUNT_MINIMUM]&
		exponent_valid[bits.BIT_COUNT_MINIMUM] != binary.UINT_8_SIZE {
		return KEY_STATUS_INPUT_INVALID
	}
	*destination = Private_Key{
		Modulus: modulus_encoding, Exponent: exponent_encoding,
		Ready: Ready{READY_COMPLETE},
	}
	return KEY_STATUS_OK
}

// Public_Key_From_Private retains only shared modulus authority.
func Public_Key_From_Private(private_key Private_Key) (public_key Public_Key) {
	defer func() { Public_Key_Invariants(public_key, "Public_Key_From_Private.public_key") }()
	Private_Key_Invariants(private_key, "Public_Key_From_Private.private_key")
	private_key_require(private_key)
	return Public_Key{
		Modulus: private_key.Modulus, Ready: Ready{READY_COMPLETE},
	}
}

// Public_Key_Bytes_Into copies the fixed modulus into caller storage.
func Public_Key_Bytes_Into(destination Modulus_Destination, public_key Public_Key) {
	defer func() {
		Modulus_Invariants(*destination, "Public_Key_Bytes_Into.destination.output")
	}()
	Modulus_Destination_Invariants(destination, "Public_Key_Bytes_Into.destination")
	Public_Key_Invariants(public_key, "Public_Key_Bytes_Into.public_key")
	public_key_require(public_key)
	*destination = public_key.Modulus
}

// Encrypt_OAEP_SHA_256 commits one fixed ciphertext after complete padding construction.
func Encrypt_OAEP_SHA_256(
	destination Ciphertext_Destination,
	generator prng.Source,
	public_key Public_Key,
	message Message,
) {
	defer func() { Ciphertext_Invariants(*destination, "Encrypt.destination.output") }()
	Ciphertext_Destination_Invariants(destination, "Encrypt.destination")
	Public_Key_Invariants(public_key, "Encrypt.public_key")
	Message_Invariants(message, "Encrypt.message")
	prng.Source_Invariants(generator, "Encrypt.generator")
	if generator.State == nil {
		panic("rsa: generator is nil")
	}
	public_key_require(public_key)
	if len(message) > MESSAGE_SIZE_MAXIMUM {
		panic("rsa: OAEP message exceeds bound")
	}
	var encoded [MODULUS_SIZE]byte
	var seed Digest
	prng.Source_Read(generator, seed[:])
	database := (*[OAEP_DATABASE_SIZE]byte)(
		encoded[binary.UINT_8_SIZE+HASH_SIZE:],
	)
	empty_hash := sha256.Checksum_256(nil)
	copy(database[:HASH_SIZE], empty_hash[:])
	delimiter_index := OAEP_DATABASE_SIZE - len(message) - binary.UINT_8_SIZE
	database[delimiter_index] = binary.UINT_8_SIZE
	copy(database[delimiter_index+binary.UINT_8_SIZE:], message)
	var database_mask [OAEP_DATABASE_SIZE]byte
	mask_database(&database_mask, seed)
	for index := range database {
		database[index] ^= database_mask[index]
	}
	var seed_mask Digest
	mask_digest(&seed_mask, database)
	for index := range seed {
		encoded[binary.UINT_8_SIZE+index] = seed[index] ^ seed_mask[index]
	}
	var result [MODULUS_SIZE]byte
	rsa_public_operation(&result, &encoded, public_key)
	copy(destination[:], result[:])
}

// Decrypt_OAEP_SHA_256 validates padding before transactional plaintext commit.
func Decrypt_OAEP_SHA_256(
	destination Destination,
	private_key Private_Key,
	ciphertext Ciphertext_Unvalidated,
) (count Count, status Decrypt_Status) {
	defer func() {
		Count_Invariants(count, "Decrypt.count")
		Decrypt_Status_Invariants(status, "Decrypt.status")
	}()
	Destination_Invariants(destination, "Decrypt.destination")
	Private_Key_Invariants(private_key, "Decrypt.private_key")
	Ciphertext_Unvalidated_Invariants(ciphertext, "Decrypt.ciphertext")
	private_key_require(private_key)
	if len(destination) > DESTINATION_SIZE_MAXIMUM {
		panic("rsa: plaintext destination exceeds bound")
	}
	if len(ciphertext) > CIPHERTEXT_UNVALIDATED_SIZE_MAXIMUM {
		panic("rsa: ciphertext exceeds bound")
	}
	if len(ciphertext) != MODULUS_SIZE {
		return COUNT_MINIMUM, DECRYPT_STATUS_INPUT_INVALID
	}
	var ciphertext_encoding, encoded [MODULUS_SIZE]byte
	copy(ciphertext_encoding[:], ciphertext)
	if integer_encoding_canonical(
		&ciphertext_encoding, &private_key.Modulus,
	)[bits.BIT_COUNT_MINIMUM] != binary.UINT_8_SIZE {
		return COUNT_MINIMUM, DECRYPT_STATUS_INPUT_INVALID
	}
	rsa_private_operation(&encoded, &ciphertext_encoding, private_key)
	var seed Digest
	copy(seed[:], encoded[binary.UINT_8_SIZE:binary.UINT_8_SIZE+HASH_SIZE])
	database := (*[OAEP_DATABASE_SIZE]byte)(
		encoded[binary.UINT_8_SIZE+HASH_SIZE:],
	)
	var seed_mask Digest
	mask_digest(&seed_mask, database)
	for index := range seed {
		seed[index] ^= seed_mask[index]
	}
	var database_mask [OAEP_DATABASE_SIZE]byte
	mask_database(&database_mask, seed)
	for index := range database {
		database[index] ^= database_mask[index]
	}
	message_count, valid := oaep_message_count(&encoded, database)
	if valid[bits.BIT_COUNT_MINIMUM] != binary.UINT_8_SIZE {
		return COUNT_MINIMUM, DECRYPT_STATUS_INPUT_INVALID
	}
	count = message_count
	if len(destination) < int(count) {
		return count, DECRYPT_STATUS_DESTINATION_TOO_SMALL
	}
	message_start := OAEP_DATABASE_SIZE - int(count)
	var plaintext [MESSAGE_SIZE_MAXIMUM]byte
	copy(plaintext[:count], database[message_start:])
	copy(destination[:count], plaintext[:count])
	return count, DECRYPT_STATUS_OK
}

// Sign_PSS_SHA_256 uses one digest-width injected salt and fixed private work.
func Sign_PSS_SHA_256(
	destination Signature_Destination,
	generator prng.Source,
	private_key Private_Key,
	digest Digest,
) {
	defer func() { Signature_Invariants(*destination, "Sign_PSS.destination.output") }()
	Signature_Destination_Invariants(destination, "Sign_PSS.destination")
	Private_Key_Invariants(private_key, "Sign_PSS.private_key")
	Digest_Invariants(digest, "Sign_PSS.digest")
	prng.Source_Invariants(generator, "Sign_PSS.generator")
	if generator.State == nil {
		panic("rsa: generator is nil")
	}
	private_key_require(private_key)
	var salt Digest
	prng.Source_Read(generator, salt[:])
	encoded := pss_encoding(digest, salt)
	var signature [MODULUS_SIZE]byte
	rsa_private_operation(&signature, &encoded, private_key)
	copy(destination[:], signature[:])
}

// Verify_PSS_SHA_256 rejects malformed signatures before fixed public work.
func Verify_PSS_SHA_256(
	public_key Public_Key, digest Digest, signature Signature_Unvalidated,
) (verified Verification) {
	defer func() { Verification_Invariants(verified, "Verify_PSS.verified") }()
	Public_Key_Invariants(public_key, "Verify_PSS.public_key")
	Digest_Invariants(digest, "Verify_PSS.digest")
	Signature_Unvalidated_Invariants(signature, "Verify_PSS.signature")
	public_key_require(public_key)
	if len(signature) > SIGNATURE_UNVALIDATED_SIZE_MAXIMUM {
		panic("rsa: PSS signature exceeds bound")
	}
	if len(signature) != MODULUS_SIZE {
		return false
	}
	var signature_encoding, encoded [MODULUS_SIZE]byte
	copy(signature_encoding[:], signature)
	if integer_encoding_canonical(
		&signature_encoding, &public_key.Modulus,
	)[bits.BIT_COUNT_MINIMUM] != binary.UINT_8_SIZE {
		return false
	}
	rsa_public_operation(&encoded, &signature_encoding, public_key)
	return Verification(
		pss_encoding_valid(&encoded, digest)[bits.BIT_COUNT_MINIMUM] ==
			binary.UINT_8_SIZE,
	)
}

// Sign_PKCS1_V1_5_SHA_256 constructs the exact SHA-256 DigestInfo block.
func Sign_PKCS1_V1_5_SHA_256(
	destination Signature_Destination, private_key Private_Key, digest Digest,
) {
	defer func() { Signature_Invariants(*destination, "Sign_PKCS1.destination.output") }()
	Signature_Destination_Invariants(destination, "Sign_PKCS1.destination")
	Private_Key_Invariants(private_key, "Sign_PKCS1.private_key")
	Digest_Invariants(digest, "Sign_PKCS1.digest")
	private_key_require(private_key)
	encoded := pkcs1_v1_5_encoding(digest)
	var signature [MODULUS_SIZE]byte
	rsa_private_operation(&signature, &encoded, private_key)
	copy(destination[:], signature[:])
}

// Verify_PKCS1_V1_5_SHA_256 compares the complete recovered encoding.
func Verify_PKCS1_V1_5_SHA_256(
	public_key Public_Key, digest Digest, signature Signature_Unvalidated,
) (verified Verification) {
	defer func() { Verification_Invariants(verified, "Verify_PKCS1.verified") }()
	Public_Key_Invariants(public_key, "Verify_PKCS1.public_key")
	Digest_Invariants(digest, "Verify_PKCS1.digest")
	Signature_Unvalidated_Invariants(signature, "Verify_PKCS1.signature")
	public_key_require(public_key)
	if len(signature) > SIGNATURE_UNVALIDATED_SIZE_MAXIMUM {
		panic("rsa: PKCS1 signature exceeds bound")
	}
	if len(signature) != MODULUS_SIZE {
		return false
	}
	var signature_encoding, encoded [MODULUS_SIZE]byte
	copy(signature_encoding[:], signature)
	if integer_encoding_canonical(
		&signature_encoding, &public_key.Modulus,
	)[bits.BIT_COUNT_MINIMUM] != binary.UINT_8_SIZE {
		return false
	}
	rsa_public_operation(&encoded, &signature_encoding, public_key)
	want := pkcs1_v1_5_encoding(digest)
	difference := bits.WORD_8_MINIMUM
	for index := range encoded {
		difference |= encoded[index] ^ want[index]
	}
	return Verification(difference == bits.WORD_8_MINIMUM)
}

func oaep_message_count(
	encoded *[MODULUS_SIZE]byte, database *[OAEP_DATABASE_SIZE]byte,
) (message_count Count, valid [CONDITION_LIMB_COUNT]uint64) {
	defer func() { Count_Invariants(message_count, "oaep_message_count.message_count") }()
	empty_hash := sha256.Checksum_256(nil)
	difference := encoded[bits.BIT_COUNT_MINIMUM]
	for index := bits.BIT_COUNT_MINIMUM; index < HASH_SIZE; index++ {
		difference |= database[index] ^ empty_hash[index]
	}
	delimiter_search := uint64(binary.UINT_8_SIZE)
	invalid := uint64(bits.WORD_64_MINIMUM)
	selected_index := uint64(bits.WORD_64_MINIMUM)
	for index := HASH_SIZE; index < OAEP_DATABASE_SIZE; index++ {
		zero_difference := uint64(database[index])
		zero := (zero_difference|-zero_difference)>>
			(bits.BIT_COUNT_64_MAXIMUM-binary.UINT_8_SIZE) ^ binary.UINT_8_SIZE
		one_difference := uint64(database[index] ^ binary.UINT_8_SIZE)
		one := (one_difference|-one_difference)>>
			(bits.BIT_COUNT_64_MAXIMUM-binary.UINT_8_SIZE) ^ binary.UINT_8_SIZE
		invalid |= delimiter_search & (zero ^ binary.UINT_8_SIZE) &
			(one ^ binary.UINT_8_SIZE)
		select_delimiter := delimiter_search & one
		mask := uint64(bits.WORD_64_MINIMUM) - select_delimiter
		selected_index ^= mask & (selected_index ^ uint64(index))
		delimiter_search &= one ^ binary.UINT_8_SIZE
	}
	invalid |= delimiter_search
	difference_word := uint64(difference) | invalid
	valid[bits.BIT_COUNT_MINIMUM] = (difference_word|-difference_word)>>
		(bits.BIT_COUNT_64_MAXIMUM-binary.UINT_8_SIZE) ^ binary.UINT_8_SIZE
	message_start := int(selected_index) + binary.UINT_8_SIZE
	return Count(OAEP_DATABASE_SIZE - message_start), valid
}

func pss_encoding(digest Digest, salt Digest) (encoded [MODULUS_SIZE]byte) {
	Digest_Invariants(digest, "pss_encoding.digest")
	Digest_Invariants(salt, "pss_encoding.salt")
	hash := pss_hash(digest, salt)
	database := (*[PSS_DATABASE_SIZE]byte)(encoded[:PSS_DATABASE_SIZE])
	database[PSS_PADDING_SIZE] = binary.UINT_8_SIZE
	copy(database[PSS_PADDING_SIZE+binary.UINT_8_SIZE:], salt[:])
	var mask [PSS_DATABASE_SIZE]byte
	mask_pss_database(&mask, hash)
	for index := range database {
		database[index] ^= mask[index]
	}
	database[bits.BIT_COUNT_MINIMUM] &=
		byte(bits.WORD_8_MAXIMUM >> binary.UINT_8_SIZE)
	copy(encoded[PSS_DATABASE_SIZE:PSS_DATABASE_SIZE+HASH_SIZE], hash[:])
	encoded[MODULUS_SIZE-binary.UINT_8_SIZE] = PSS_TRAILER
	return encoded
}

func pss_encoding_valid(
	encoded *[MODULUS_SIZE]byte, digest Digest,
) (valid [CONDITION_LIMB_COUNT]uint64) {
	Digest_Invariants(digest, "pss_encoding_valid.digest")
	difference := encoded[MODULUS_SIZE-binary.UINT_8_SIZE] ^ PSS_TRAILER
	difference |= encoded[bits.BIT_COUNT_MINIMUM] & byte(binary.UINT_8_SIZE<<
		(bits.BIT_COUNT_8_MAXIMUM-binary.UINT_8_SIZE))
	var hash Digest
	copy(hash[:], encoded[PSS_DATABASE_SIZE:PSS_DATABASE_SIZE+HASH_SIZE])
	var database [PSS_DATABASE_SIZE]byte
	copy(database[:], encoded[:PSS_DATABASE_SIZE])
	var mask [PSS_DATABASE_SIZE]byte
	mask_pss_database(&mask, hash)
	for index := range database {
		database[index] ^= mask[index]
	}
	database[bits.BIT_COUNT_MINIMUM] &=
		byte(bits.WORD_8_MAXIMUM >> binary.UINT_8_SIZE)
	for index := bits.BIT_COUNT_MINIMUM; index < PSS_PADDING_SIZE; index++ {
		difference |= database[index]
	}
	difference |= database[PSS_PADDING_SIZE] ^ binary.UINT_8_SIZE
	var salt Digest
	copy(salt[:], database[PSS_PADDING_SIZE+binary.UINT_8_SIZE:])
	want := pss_hash(digest, salt)
	for index := range hash {
		difference |= hash[index] ^ want[index]
	}
	word := uint64(difference)
	valid[bits.BIT_COUNT_MINIMUM] = (word|-word)>>
		(bits.BIT_COUNT_64_MAXIMUM-binary.UINT_8_SIZE) ^ binary.UINT_8_SIZE
	return valid
}

func pss_hash(digest Digest, salt Digest) (value Digest) {
	defer func() { Digest_Invariants(value, "pss_hash.value") }()
	Digest_Invariants(digest, "pss_hash.digest")
	Digest_Invariants(salt, "pss_hash.salt")
	var hash sha256.Digest
	sha256.Digest_Init(&hash, sha256.KIND_SHA_256)
	var prefix [PSS_PREFIX_SIZE]byte
	sha256.Digest_Write(&hash, prefix[:])
	sha256.Digest_Write(&hash, digest[:])
	sha256.Digest_Write(&hash, salt[:])
	result := sha256.Digest_Sum_256(&hash)
	copy(value[:], result[:])
	return value
}

func pkcs1_v1_5_encoding(digest Digest) (encoded [MODULUS_SIZE]byte) {
	Digest_Invariants(digest, "pkcs1_v1_5_encoding.digest")
	prefix := pkcs1_sha_256_prefix()
	encoded[binary.UINT_8_SIZE] = BLOCK_TYPE_SIGNATURE
	delimiter_index := MODULUS_SIZE - len(prefix) - HASH_SIZE - binary.UINT_8_SIZE
	for index := binary.UINT_16_SIZE; index < delimiter_index; index++ {
		encoded[index] = PADDING_BYTE_SIGNATURE
	}
	copy(encoded[delimiter_index+binary.UINT_8_SIZE:], prefix[:])
	copy(encoded[MODULUS_SIZE-HASH_SIZE:], digest[:])
	return encoded
}

// SHA-256 DigestInfo is fixed by PKCS1 and the NIST algorithm identifier.
func pkcs1_sha_256_prefix() (prefix [PKCS1_SHA_256_PREFIX_SIZE]byte) {
	copy(prefix[:], PKCS1_SHA_256_PREFIX)
	return prefix
}

func mask_database(destination *[OAEP_DATABASE_SIZE]byte, seed Digest) {
	Digest_Invariants(seed, "mask_database.seed")
	mask_sha_256_blocks(destination, &seed)
}

func mask_pss_database(destination *[PSS_DATABASE_SIZE]byte, seed Digest) {
	Digest_Invariants(seed, "mask_pss_database.seed")
	mask_sha_256_blocks(destination, &seed)
}

func mask_digest(destination *Digest, seed *[OAEP_DATABASE_SIZE]byte) {
	Digest_Invariants(*destination, "mask_digest.destination.input")
	var hash sha256.Digest
	sha256.Digest_Init(&hash, sha256.KIND_SHA_256)
	sha256.Digest_Write(&hash, seed[:])
	var counter [MASK_COUNTER_SIZE]byte
	sha256.Digest_Write(&hash, counter[:])
	value := sha256.Digest_Sum_256(&hash)
	copy(destination[:], value[:])
}

func mask_sha_256_blocks(
	destination *[OAEP_DATABASE_SIZE]byte, seed *Digest,
) {
	Digest_Invariants(*seed, "mask_sha_256_blocks.seed")
	var counter [MASK_COUNTER_SIZE]byte
	for block_index := bits.BIT_COUNT_MINIMUM; block_index < MASK_BLOCK_COUNT; block_index++ {
		binary.Put_Uint_32(
			counter[:], binary.Word_32(block_index), binary.BIG_ENDIAN,
		)
		var hash sha256.Digest
		sha256.Digest_Init(&hash, sha256.KIND_SHA_256)
		sha256.Digest_Write(&hash, seed[:])
		sha256.Digest_Write(&hash, counter[:])
		value := sha256.Digest_Sum_256(&hash)
		start := block_index * HASH_SIZE
		end := start + HASH_SIZE
		if end > len(destination) {
			end = len(destination)
		}
		copy(destination[start:end], value[:end-start])
	}
}

func rsa_public_operation(
	destination *[MODULUS_SIZE]byte,
	source *[MODULUS_SIZE]byte,
	public_key Public_Key,
) {
	Public_Key_Invariants(public_key, "rsa_public_operation.public_key")
	var exponent [MODULUS_LIMB_COUNT]uint64
	exponent[bits.BIT_COUNT_MINIMUM] = uint64(PUBLIC_EXPONENT)
	rsa_operation(
		destination, source, &public_key.Modulus, &exponent,
	)
}

func rsa_private_operation(
	destination *[MODULUS_SIZE]byte,
	source *[MODULUS_SIZE]byte,
	private_key Private_Key,
) {
	Private_Key_Invariants(private_key, "rsa_private_operation.private_key")
	var exponent [MODULUS_LIMB_COUNT]uint64
	integer_decode(&exponent, (*[MODULUS_SIZE]byte)(&private_key.Exponent))
	rsa_operation(
		destination, source, &private_key.Modulus, &exponent,
	)
}

func rsa_operation(
	destination *[MODULUS_SIZE]byte,
	source *[MODULUS_SIZE]byte,
	modulus_encoding *Modulus,
	exponent *[MODULUS_LIMB_COUNT]uint64,
) {
	Modulus_Invariants(*modulus_encoding, "rsa_operation.modulus_encoding")
	var modulus, source_integer [MODULUS_LIMB_COUNT]uint64
	integer_decode(&modulus, (*[MODULUS_SIZE]byte)(modulus_encoding))
	integer_decode(&source_integer, source)
	modulus_low := [CONDITION_LIMB_COUNT]uint64{modulus[bits.BIT_COUNT_MINIMUM]}
	n0_inverse := montgomery_inverse(&modulus_low)
	r_squared := montgomery_r_squared(&modulus)
	var one [MODULUS_LIMB_COUNT]uint64
	one[bits.BIT_COUNT_MINIMUM] = binary.UINT_8_SIZE
	var base, result [MODULUS_LIMB_COUNT]uint64
	montgomery_multiply(&base, &source_integer, &r_squared, &modulus, &n0_inverse)
	montgomery_multiply(&result, &one, &r_squared, &modulus, &n0_inverse)
	for bit_count := MODULUS_BIT_COUNT; bit_count > bits.BIT_COUNT_MINIMUM; bit_count-- {
		bit_index := bit_count - binary.UINT_8_SIZE
		var square, product [MODULUS_LIMB_COUNT]uint64
		montgomery_multiply(&square, &result, &result, &modulus, &n0_inverse)
		montgomery_multiply(&product, &square, &base, &modulus, &n0_inverse)
		word_index := bit_index / bits.BIT_COUNT_64_MAXIMUM
		word_shift := uint(bit_index % bits.BIT_COUNT_64_MAXIMUM)
		bit := exponent[word_index] >> word_shift & binary.UINT_8_SIZE
		limbs_select(
			&result, &product, &square,
			[CONDITION_LIMB_COUNT]uint64{bit},
		)
	}
	var decoded [MODULUS_LIMB_COUNT]uint64
	montgomery_multiply(&decoded, &result, &one, &modulus, &n0_inverse)
	integer_encode(destination, &decoded)
}

func montgomery_multiply(
	destination *[MODULUS_LIMB_COUNT]uint64,
	left *[MODULUS_LIMB_COUNT]uint64,
	right *[MODULUS_LIMB_COUNT]uint64,
	modulus *[MODULUS_LIMB_COUNT]uint64,
	n0_inverse *[CONDITION_LIMB_COUNT]uint64,
) {
	var temporary [MONTGOMERY_TEMPORARY_LIMB_COUNT]uint64
	for word_index := range MODULUS_LIMB_COUNT {
		multiplier := [CONDITION_LIMB_COUNT]uint64{right[word_index]}
		montgomery_add_multiply(&temporary, left, &multiplier)
		factor := [CONDITION_LIMB_COUNT]uint64{
			temporary[bits.BIT_COUNT_MINIMUM] * n0_inverse[bits.BIT_COUNT_MINIMUM],
		}
		montgomery_add_multiply(&temporary, modulus, &factor)
		for shift_index := range MODULUS_LIMB_COUNT + binary.UINT_8_SIZE {
			temporary[shift_index] = temporary[shift_index+binary.UINT_8_SIZE]
		}
		temporary[MODULUS_LIMB_COUNT+binary.UINT_8_SIZE] = bits.WORD_64_MINIMUM
	}
	var value, reduced [MODULUS_LIMB_COUNT]uint64
	copy(value[:], temporary[:MODULUS_LIMB_COUNT])
	borrow := limbs_subtract(&reduced, &value, modulus)
	top := temporary[MODULUS_LIMB_COUNT]
	top_nonzero := (top | -top) >> (bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
	limbs_select(
		destination, &reduced, &value,
		[CONDITION_LIMB_COUNT]uint64{
			top_nonzero | (borrow[bits.BIT_COUNT_MINIMUM] ^ binary.UINT_8_SIZE),
		},
	)
}

func montgomery_add_multiply(
	temporary *[MONTGOMERY_TEMPORARY_LIMB_COUNT]uint64,
	multiplicand *[MODULUS_LIMB_COUNT]uint64,
	multiplier *[CONDITION_LIMB_COUNT]uint64,
) {
	carry := uint64(bits.WORD_64_MINIMUM)
	for limb_index := bits.BIT_COUNT_MINIMUM; limb_index < MODULUS_LIMB_COUNT; limb_index++ {
		left := multiplicand[limb_index]
		right := multiplier[bits.BIT_COUNT_MINIMUM]
		const HALF_MASK uint64 = binary.UINT_8_SIZE<<bits.BIT_COUNT_32_MAXIMUM -
			binary.UINT_8_SIZE
		left_low := left & HALF_MASK
		left_high := left >> bits.BIT_COUNT_32_MAXIMUM
		right_low := right & HALF_MASK
		right_high := right >> bits.BIT_COUNT_32_MAXIMUM
		partial := left_low * right_low
		middle_first := left_high*right_low +
			partial>>bits.BIT_COUNT_32_MAXIMUM
		middle_second := left_low*right_high + middle_first&HALF_MASK
		high := left_high*right_high +
			middle_first>>bits.BIT_COUNT_32_MAXIMUM +
			middle_second>>bits.BIT_COUNT_32_MAXIMUM
		low := left * right
		first_sum := low + temporary[limb_index]
		first_carry := ((low & temporary[limb_index]) |
			((low | temporary[limb_index]) & ^first_sum)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		low = first_sum + carry
		second_carry := ((first_sum & carry) |
			((first_sum | carry) & ^low)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		temporary[limb_index] = low
		carry = high + first_carry + second_carry
	}
	top := temporary[MODULUS_LIMB_COUNT]
	sum := top + carry
	top_carry := ((top & carry) | ((top | carry) & ^sum)) >>
		(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
	temporary[MODULUS_LIMB_COUNT] = sum
	temporary[MODULUS_LIMB_COUNT+binary.UINT_8_SIZE] += top_carry
}

func montgomery_r_squared(
	modulus *[MODULUS_LIMB_COUNT]uint64,
) (value [MODULUS_LIMB_COUNT]uint64) {
	value[bits.BIT_COUNT_MINIMUM] = binary.UINT_8_SIZE
	for range MODULUS_BIT_COUNT + MODULUS_BIT_COUNT {
		modular_add(&value, &value, &value, modulus)
	}
	return value
}

func montgomery_inverse(
	modulus_low *[CONDITION_LIMB_COUNT]uint64,
) (inverse [CONDITION_LIMB_COUNT]uint64) {
	inverse[bits.BIT_COUNT_MINIMUM] = binary.UINT_8_SIZE
	for range MONTGOMERY_INVERSE_COUNT {
		inverse[bits.BIT_COUNT_MINIMUM] *= binary.UINT_16_SIZE -
			modulus_low[bits.BIT_COUNT_MINIMUM]*inverse[bits.BIT_COUNT_MINIMUM]
	}
	inverse[bits.BIT_COUNT_MINIMUM] = bits.WORD_64_MINIMUM -
		inverse[bits.BIT_COUNT_MINIMUM]
	return inverse
}

func modular_add(
	destination *[MODULUS_LIMB_COUNT]uint64,
	left *[MODULUS_LIMB_COUNT]uint64,
	right *[MODULUS_LIMB_COUNT]uint64,
	modulus *[MODULUS_LIMB_COUNT]uint64,
) {
	var sum, reduced [MODULUS_LIMB_COUNT]uint64
	carry := limbs_add(&sum, left, right)
	borrow := limbs_subtract(&reduced, &sum, modulus)
	limbs_select(
		destination, &reduced, &sum,
		[CONDITION_LIMB_COUNT]uint64{
			carry[bits.BIT_COUNT_MINIMUM] |
				(borrow[bits.BIT_COUNT_MINIMUM] ^ binary.UINT_8_SIZE),
		},
	)
}

func integer_decode(
	destination *[MODULUS_LIMB_COUNT]uint64, source *[MODULUS_SIZE]byte,
) {
	for limb_index := bits.BIT_COUNT_MINIMUM; limb_index < MODULUS_LIMB_COUNT; limb_index++ {
		source_index := MODULUS_SIZE - (limb_index+binary.UINT_8_SIZE)*binary.UINT_64_SIZE
		destination[limb_index] = uint64(binary.Uint_64(
			source[source_index:source_index+binary.UINT_64_SIZE], binary.BIG_ENDIAN,
		))
	}
}

func integer_encode(
	destination *[MODULUS_SIZE]byte, source *[MODULUS_LIMB_COUNT]uint64,
) {
	for limb_index := bits.BIT_COUNT_MINIMUM; limb_index < MODULUS_LIMB_COUNT; limb_index++ {
		destination_index := MODULUS_SIZE -
			(limb_index+binary.UINT_8_SIZE)*binary.UINT_64_SIZE
		binary.Put_Uint_64(
			destination[destination_index:destination_index+binary.UINT_64_SIZE],
			binary.Word_64(source[limb_index]), binary.BIG_ENDIAN,
		)
	}
}

func modulus_encoding_valid(
	modulus *Modulus,
) (valid [CONDITION_LIMB_COUNT]uint64) {
	Modulus_Invariants(*modulus, "modulus_encoding_valid.modulus")
	top_bit := uint64(
		modulus[bits.BIT_COUNT_MINIMUM] >>
			(bits.BIT_COUNT_8_MAXIMUM - binary.UINT_8_SIZE),
	)
	odd := uint64(modulus[MODULUS_SIZE-binary.UINT_8_SIZE] & byte(binary.UINT_8_SIZE))
	valid[bits.BIT_COUNT_MINIMUM] = top_bit & odd
	return valid
}

func private_exponent_encoding_valid(
	exponent *Private_Exponent, modulus *Modulus,
) (valid [CONDITION_LIMB_COUNT]uint64) {
	Private_Exponent_Invariants(
		*exponent, "private_exponent_encoding_valid.exponent",
	)
	Modulus_Invariants(*modulus, "private_exponent_encoding_valid.modulus")
	var exponent_integer, modulus_integer [MODULUS_LIMB_COUNT]uint64
	integer_decode(&exponent_integer, (*[MODULUS_SIZE]byte)(exponent))
	integer_decode(&modulus_integer, (*[MODULUS_SIZE]byte)(modulus))
	var difference [MODULUS_LIMB_COUNT]uint64
	canonical := limbs_subtract(&difference, &exponent_integer, &modulus_integer)
	nonzero := limbs_is_zero(&exponent_integer)
	valid[bits.BIT_COUNT_MINIMUM] = canonical[bits.BIT_COUNT_MINIMUM] &
		(nonzero[bits.BIT_COUNT_MINIMUM] ^ binary.UINT_8_SIZE)
	return valid
}

func integer_encoding_canonical(
	encoding *[MODULUS_SIZE]byte, modulus *Modulus,
) (canonical [CONDITION_LIMB_COUNT]uint64) {
	Modulus_Invariants(*modulus, "integer_encoding_canonical.modulus")
	var value, modulus_integer [MODULUS_LIMB_COUNT]uint64
	integer_decode(&value, encoding)
	integer_decode(&modulus_integer, (*[MODULUS_SIZE]byte)(modulus))
	var difference [MODULUS_LIMB_COUNT]uint64
	return limbs_subtract(&difference, &value, &modulus_integer)
}

func limbs_add(
	destination *[MODULUS_LIMB_COUNT]uint64,
	left *[MODULUS_LIMB_COUNT]uint64,
	right *[MODULUS_LIMB_COUNT]uint64,
) (carry [CONDITION_LIMB_COUNT]uint64) {
	carry_value := uint64(bits.WORD_64_MINIMUM)
	for limb_index := range destination {
		partial := left[limb_index] + right[limb_index]
		partial_carry := ((left[limb_index] & right[limb_index]) |
			((left[limb_index] | right[limb_index]) & ^partial)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		value := partial + carry_value
		carry_carry := ((partial & carry_value) |
			((partial | carry_value) & ^value)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		destination[limb_index] = value
		carry_value = partial_carry | carry_carry
	}
	carry[bits.BIT_COUNT_MINIMUM] = carry_value
	return carry
}

func limbs_subtract(
	destination *[MODULUS_LIMB_COUNT]uint64,
	left *[MODULUS_LIMB_COUNT]uint64,
	right *[MODULUS_LIMB_COUNT]uint64,
) (borrow [CONDITION_LIMB_COUNT]uint64) {
	borrow_value := uint64(bits.WORD_64_MINIMUM)
	for limb_index := range destination {
		partial := left[limb_index] - right[limb_index]
		partial_borrow := ((^left[limb_index] & right[limb_index]) |
			(^(left[limb_index] ^ right[limb_index]) & partial)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		value := partial - borrow_value
		borrow_borrow := ((^partial & borrow_value) |
			(^(partial ^ borrow_value) & value)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		destination[limb_index] = value
		borrow_value = partial_borrow | borrow_borrow
	}
	borrow[bits.BIT_COUNT_MINIMUM] = borrow_value
	return borrow
}

func limbs_select(
	destination *[MODULUS_LIMB_COUNT]uint64,
	first *[MODULUS_LIMB_COUNT]uint64,
	second *[MODULUS_LIMB_COUNT]uint64,
	condition [CONDITION_LIMB_COUNT]uint64,
) {
	mask := uint64(bits.WORD_64_MINIMUM) - condition[bits.BIT_COUNT_MINIMUM]
	for limb_index := range destination {
		destination[limb_index] = second[limb_index] ^
			mask&(first[limb_index]^second[limb_index])
	}
}

func limbs_is_zero(
	value *[MODULUS_LIMB_COUNT]uint64,
) (zero [CONDITION_LIMB_COUNT]uint64) {
	difference := uint64(bits.WORD_64_MINIMUM)
	for limb_index := range value {
		difference |= value[limb_index]
	}
	zero[bits.BIT_COUNT_MINIMUM] = (difference|-difference)>>
		(bits.BIT_COUNT_64_MAXIMUM-binary.UINT_8_SIZE) ^ binary.UINT_8_SIZE
	return zero
}

func public_key_require(public_key Public_Key) {
	Public_Key_Invariants(public_key, "public_key_require.public_key")
	if public_key.Ready[READY_INDEX] != READY_COMPLETE {
		panic("rsa: public key is not ready")
	}
}

func private_key_require(private_key Private_Key) {
	Private_Key_Invariants(private_key, "private_key_require.private_key")
	if private_key.Ready[READY_INDEX] != READY_COMPLETE {
		panic("rsa: private key is not ready")
	}
}
