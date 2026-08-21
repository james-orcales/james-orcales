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

// INTEGER_CHUNK_LANE_COUNT is the visible word count in one storage chunk.
const INTEGER_CHUNK_LANE_COUNT = binary.UINT_32_SIZE

// INTEGER_CHUNK_SIZE derives one chunk's encoded width.
const INTEGER_CHUNK_SIZE = INTEGER_CHUNK_LANE_COUNT * binary.UINT_64_SIZE

// INTEGER_CHUNK_COUNT derives the exact number of chunks in one RSA integer.
const INTEGER_CHUNK_COUNT = MODULUS_SIZE / INTEGER_CHUNK_SIZE

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

// READY_EMPTY marks caller storage without validated key material.
const READY_EMPTY Ready = Ready(bits.WORD_8_MINIMUM)

// READY_COMPLETE marks validated key material.
const READY_COMPLETE Ready = READY_EMPTY + binary.UINT_8_SIZE

// DECISION_FALSE marks a rejected constant-time predicate.
const DECISION_FALSE Decision = Decision(bits.WORD_64_MINIMUM)

// DECISION_TRUE marks an accepted constant-time predicate.
const DECISION_TRUE Decision = DECISION_FALSE + binary.UINT_8_SIZE

// MONTGOMERY_TEMPORARY_LIMB_COUNT holds one double carry above the modulus.
const MONTGOMERY_TEMPORARY_LIMB_COUNT = MODULUS_LIMB_COUNT + binary.UINT_16_SIZE

// MONTGOMERY_INVERSE_COUNT doubles one inverse bit six times to 64 bits.
const MONTGOMERY_INVERSE_COUNT = binary.UINT_64_SIZE - binary.UINT_16_SIZE

// MONTGOMERY_HALF_BASE separates each word into two multiplication halves.
const MONTGOMERY_HALF_BASE = uint64(binary.UINT_8_SIZE) << bits.BIT_COUNT_32_MAXIMUM

// MONTGOMERY_HALF_MASK selects the low half without a machine-width shift.
const MONTGOMERY_HALF_MASK = MONTGOMERY_HALF_BASE - binary.UINT_8_SIZE

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

// Integer_Lane_0 gives the first word in each opaque chunk a stable type.
type Integer_Lane_0 uint64

// Integer_Lane_0_Invariants preserves every possible stored bit pattern.
func Integer_Lane_0_Invariants(value Integer_Lane_0, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Integer_Lane_1 gives the second word in each opaque chunk a stable type.
type Integer_Lane_1 uint64

// Integer_Lane_1_Invariants preserves every possible stored bit pattern.
func Integer_Lane_1_Invariants(value Integer_Lane_1, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Integer_Lane_2 gives the third word in each opaque chunk a stable type.
type Integer_Lane_2 uint64

// Integer_Lane_2_Invariants preserves every possible stored bit pattern.
func Integer_Lane_2_Invariants(value Integer_Lane_2, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Integer_Lane_3 gives the fourth word in each opaque chunk a stable type.
type Integer_Lane_3 uint64

// Integer_Lane_3_Invariants preserves every possible stored bit pattern.
func Integer_Lane_3_Invariants(value Integer_Lane_3, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Integer_Chunk_Storage gives four words a padding-free common layout.
type Integer_Chunk_Storage struct {
	// Lane_0 keeps the first word visible to the layout invariant.
	Lane_0 Integer_Lane_0
	// Lane_1 keeps the second word visible to the layout invariant.
	Lane_1 Integer_Lane_1
	// Lane_2 keeps the third word visible to the layout invariant.
	Lane_2 Integer_Lane_2
	// Lane_3 keeps the fourth word visible to the layout invariant.
	Lane_3 Integer_Lane_3
}

// Integer_Chunk_Storage_Invariants composes every word position once.
func Integer_Chunk_Storage_Invariants(
	value Integer_Chunk_Storage, namespace aver.Namespace,
) {
	Integer_Lane_0_Invariants(value.Lane_0, namespace)
	Integer_Lane_1_Invariants(value.Lane_1, namespace)
	Integer_Lane_2_Invariants(value.Lane_2, namespace)
	Integer_Lane_3_Invariants(value.Lane_3, namespace)
}

// Integer_Chunk_0 gives the least-addressed chunk independent identity.
type Integer_Chunk_0 Integer_Chunk_Storage

// Integer_Chunk_0_Invariants guards the first encoded segment.
func Integer_Chunk_0_Invariants(value Integer_Chunk_0, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Lane_0), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_1), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_2), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_3), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Integer_Chunk_1 gives the second chunk independent identity.
type Integer_Chunk_1 Integer_Chunk_Storage

// Integer_Chunk_1_Invariants guards the second encoded segment.
func Integer_Chunk_1_Invariants(value Integer_Chunk_1, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Lane_0), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_1), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_2), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_3), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Integer_Chunk_2 gives the third chunk independent identity.
type Integer_Chunk_2 Integer_Chunk_Storage

// Integer_Chunk_2_Invariants guards the third encoded segment.
func Integer_Chunk_2_Invariants(value Integer_Chunk_2, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Lane_0), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_1), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_2), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_3), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Integer_Chunk_3 gives the fourth chunk independent identity.
type Integer_Chunk_3 Integer_Chunk_Storage

// Integer_Chunk_3_Invariants guards the fourth encoded segment.
func Integer_Chunk_3_Invariants(value Integer_Chunk_3, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Lane_0), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_1), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_2), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_3), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Integer_Chunk_4 gives the fifth chunk independent identity.
type Integer_Chunk_4 Integer_Chunk_Storage

// Integer_Chunk_4_Invariants guards the fifth encoded segment.
func Integer_Chunk_4_Invariants(value Integer_Chunk_4, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Lane_0), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_1), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_2), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_3), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Integer_Chunk_5 gives the sixth chunk independent identity.
type Integer_Chunk_5 Integer_Chunk_Storage

// Integer_Chunk_5_Invariants guards the sixth encoded segment.
func Integer_Chunk_5_Invariants(value Integer_Chunk_5, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Lane_0), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_1), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_2), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_3), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Integer_Chunk_6 gives the seventh chunk independent identity.
type Integer_Chunk_6 Integer_Chunk_Storage

// Integer_Chunk_6_Invariants guards the seventh encoded segment.
func Integer_Chunk_6_Invariants(value Integer_Chunk_6, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Lane_0), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_1), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_2), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_3), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Integer_Chunk_7 gives the most-addressed chunk independent identity.
type Integer_Chunk_7 Integer_Chunk_Storage

// Integer_Chunk_7_Invariants guards the last encoded segment.
func Integer_Chunk_7_Invariants(value Integer_Chunk_7, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Lane_0), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_1), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_2), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_3), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Private_Exponent_Chunk_0 keeps modulus and exponent identities distinct in one key chain.
type Private_Exponent_Chunk_0 Integer_Chunk_Storage

// Private_Exponent_Chunk_0_Invariants spans every first exponent-chunk word.
func Private_Exponent_Chunk_0_Invariants(
	value Private_Exponent_Chunk_0, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Lane_0), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_1), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_2), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_3), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Private_Exponent_Chunk_1 keeps the second exponent segment independently observable.
type Private_Exponent_Chunk_1 Integer_Chunk_Storage

// Private_Exponent_Chunk_1_Invariants spans every second exponent-chunk word.
func Private_Exponent_Chunk_1_Invariants(
	value Private_Exponent_Chunk_1, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Lane_0), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_1), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_2), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_3), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Private_Exponent_Chunk_2 keeps the third exponent segment independently observable.
type Private_Exponent_Chunk_2 Integer_Chunk_Storage

// Private_Exponent_Chunk_2_Invariants spans every third exponent-chunk word.
func Private_Exponent_Chunk_2_Invariants(
	value Private_Exponent_Chunk_2, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Lane_0), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_1), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_2), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_3), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Private_Exponent_Chunk_3 keeps the fourth exponent segment independently observable.
type Private_Exponent_Chunk_3 Integer_Chunk_Storage

// Private_Exponent_Chunk_3_Invariants spans every fourth exponent-chunk word.
func Private_Exponent_Chunk_3_Invariants(
	value Private_Exponent_Chunk_3, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Lane_0), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_1), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_2), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_3), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Private_Exponent_Chunk_4 keeps the fifth exponent segment independently observable.
type Private_Exponent_Chunk_4 Integer_Chunk_Storage

// Private_Exponent_Chunk_4_Invariants spans every fifth exponent-chunk word.
func Private_Exponent_Chunk_4_Invariants(
	value Private_Exponent_Chunk_4, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Lane_0), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_1), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_2), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_3), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Private_Exponent_Chunk_5 keeps the sixth exponent segment independently observable.
type Private_Exponent_Chunk_5 Integer_Chunk_Storage

// Private_Exponent_Chunk_5_Invariants spans every sixth exponent-chunk word.
func Private_Exponent_Chunk_5_Invariants(
	value Private_Exponent_Chunk_5, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Lane_0), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_1), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_2), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_3), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Private_Exponent_Chunk_6 keeps the seventh exponent segment independently observable.
type Private_Exponent_Chunk_6 Integer_Chunk_Storage

// Private_Exponent_Chunk_6_Invariants spans every seventh exponent-chunk word.
func Private_Exponent_Chunk_6_Invariants(
	value Private_Exponent_Chunk_6, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Lane_0), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_1), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_2), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_3), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Private_Exponent_Chunk_7 keeps the final exponent segment independently observable.
type Private_Exponent_Chunk_7 Integer_Chunk_Storage

// Private_Exponent_Chunk_7_Invariants spans every final exponent-chunk word.
func Private_Exponent_Chunk_7_Invariants(
	value Private_Exponent_Chunk_7, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Lane_0), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_1), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_2), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Lane_3), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Integer_Storage keeps RSA-2048 ownership inline without array boundaries.
type Integer_Storage struct {
	// Chunk_0 keeps the first 32 encoded bytes copy-safe.
	Chunk_0 Integer_Chunk_0
	// Chunk_1 keeps the next 32 encoded bytes copy-safe.
	Chunk_1 Integer_Chunk_1
	// Chunk_2 keeps the next 32 encoded bytes copy-safe.
	Chunk_2 Integer_Chunk_2
	// Chunk_3 keeps the next 32 encoded bytes copy-safe.
	Chunk_3 Integer_Chunk_3
	// Chunk_4 keeps the next 32 encoded bytes copy-safe.
	Chunk_4 Integer_Chunk_4
	// Chunk_5 keeps the next 32 encoded bytes copy-safe.
	Chunk_5 Integer_Chunk_5
	// Chunk_6 keeps the next 32 encoded bytes copy-safe.
	Chunk_6 Integer_Chunk_6
	// Chunk_7 keeps the final 32 encoded bytes copy-safe.
	Chunk_7 Integer_Chunk_7
}

// Integer_Storage_Invariants exposes each chunk and therefore any layout drift.
func Integer_Storage_Invariants(value Integer_Storage, namespace aver.Namespace) {
	Integer_Chunk_0_Invariants(value.Chunk_0, namespace)
	Integer_Chunk_1_Invariants(value.Chunk_1, namespace)
	Integer_Chunk_2_Invariants(value.Chunk_2, namespace)
	Integer_Chunk_3_Invariants(value.Chunk_3, namespace)
	Integer_Chunk_4_Invariants(value.Chunk_4, namespace)
	Integer_Chunk_5_Invariants(value.Chunk_5, namespace)
	Integer_Chunk_6_Invariants(value.Chunk_6, namespace)
	Integer_Chunk_7_Invariants(value.Chunk_7, namespace)
}

// Modulus is opaque copy-safe RSA-2048 modulus ownership.
type Modulus Integer_Storage

// Modulus_Invariants composes every exact-width storage chunk.
func Modulus_Invariants(value Modulus, namespace aver.Namespace) {
	Integer_Chunk_0_Invariants(value.Chunk_0, namespace)
	Integer_Chunk_1_Invariants(value.Chunk_1, namespace)
	Integer_Chunk_2_Invariants(value.Chunk_2, namespace)
	Integer_Chunk_3_Invariants(value.Chunk_3, namespace)
	Integer_Chunk_4_Invariants(value.Chunk_4, namespace)
	Integer_Chunk_5_Invariants(value.Chunk_5, namespace)
	Integer_Chunk_6_Invariants(value.Chunk_6, namespace)
	Integer_Chunk_7_Invariants(value.Chunk_7, namespace)
}

// Private_Exponent owns one RSA-2048 exponent without sharing modulus assertion identities.
type Private_Exponent struct {
	// Chunk_0 keeps the first 32 encoded exponent bytes copy-safe.
	Chunk_0 Private_Exponent_Chunk_0
	// Chunk_1 keeps the next 32 encoded exponent bytes copy-safe.
	Chunk_1 Private_Exponent_Chunk_1
	// Chunk_2 keeps the next 32 encoded exponent bytes copy-safe.
	Chunk_2 Private_Exponent_Chunk_2
	// Chunk_3 keeps the next 32 encoded exponent bytes copy-safe.
	Chunk_3 Private_Exponent_Chunk_3
	// Chunk_4 keeps the next 32 encoded exponent bytes copy-safe.
	Chunk_4 Private_Exponent_Chunk_4
	// Chunk_5 keeps the next 32 encoded exponent bytes copy-safe.
	Chunk_5 Private_Exponent_Chunk_5
	// Chunk_6 keeps the next 32 encoded exponent bytes copy-safe.
	Chunk_6 Private_Exponent_Chunk_6
	// Chunk_7 keeps the final 32 encoded exponent bytes copy-safe.
	Chunk_7 Private_Exponent_Chunk_7
}

// Private_Exponent_Invariants composes every exact-width storage chunk.
func Private_Exponent_Invariants(value Private_Exponent, namespace aver.Namespace) {
	Private_Exponent_Chunk_0_Invariants(value.Chunk_0, namespace)
	Private_Exponent_Chunk_1_Invariants(value.Chunk_1, namespace)
	Private_Exponent_Chunk_2_Invariants(value.Chunk_2, namespace)
	Private_Exponent_Chunk_3_Invariants(value.Chunk_3, namespace)
	Private_Exponent_Chunk_4_Invariants(value.Chunk_4, namespace)
	Private_Exponent_Chunk_5_Invariants(value.Chunk_5, namespace)
	Private_Exponent_Chunk_6_Invariants(value.Chunk_6, namespace)
	Private_Exponent_Chunk_7_Invariants(value.Chunk_7, namespace)
}

// Ready stores caller key-state identity.
type Ready uint8

// Ready_Invariants admits empty and validated key storage.
func Ready_Invariants(value Ready, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(READY_EMPTY), uint8(READY_COMPLETE)).
		Ensure()
}

// Decision is one constant-time RSA predicate.
type Decision uint64

// Decision_Invariants admits only rejected and accepted predicates.
func Decision_Invariants(value Decision, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint64(uint64(value), uint64(DECISION_FALSE), uint64(DECISION_TRUE)).
		Ensure()
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
	valid := modulus_encoding_valid(&value.Modulus)
	aver.Always(
		uint64(value.Ready)&uint64(valid) == uint64(value.Ready),
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
	modulus_valid := modulus_encoding_valid(&value.Modulus)
	exponent_valid := private_exponent_encoding_valid(&value.Exponent, &value.Modulus)
	valid := modulus_valid & exponent_valid
	aver.Always(
		uint64(value.Ready)&uint64(valid) == uint64(value.Ready),
		"A complete RSA private key has valid bounded integers.",
	)
}

// Public_Key_Destination is nonnil caller-owned public key storage.
type Public_Key_Destination *Public_Key

// Public_Key_Destination_Invariants proves caller storage exists.
func Public_Key_Destination_Invariants(
	value Public_Key_Destination, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Public_Key_Invariants(*value, namespace)
}

// Private_Key_Destination is nonnil caller-owned private key storage.
type Private_Key_Destination *Private_Key

// Private_Key_Destination_Invariants proves caller storage exists.
func Private_Key_Destination_Invariants(
	value Private_Key_Destination, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Private_Key_Invariants(*value, namespace)
}

// Modulus_Destination names mutable opaque modulus ownership.
type Modulus_Destination *Modulus

// Modulus_Destination_Invariants composes present modulus storage.
func Modulus_Destination_Invariants(
	value Modulus_Destination, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Modulus_Invariants(*value, namespace)
}

// Private_Exponent_Destination names mutable opaque exponent ownership.
type Private_Exponent_Destination *Private_Exponent

// Private_Exponent_Destination_Invariants composes present exponent storage.
func Private_Exponent_Destination_Invariants(
	value Private_Exponent_Destination, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Private_Exponent_Invariants(*value, namespace)
}

// Modulus_Encoding is exact caller-owned big-endian modulus output.
type Modulus_Encoding []byte

// Modulus_Encoding_Invariants fixes the public wire width.
func Modulus_Encoding_Invariants(value Modulus_Encoding, _ aver.Namespace) {
	aver.Always(len(value) == MODULUS_SIZE, "RSA modulus output has exact width.")
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

// Ciphertext is one exact-width caller-owned RSA-2048 encrypted message.
type Ciphertext []byte

// Ciphertext_Invariants fixes caller-owned ciphertext width.
func Ciphertext_Invariants(value Ciphertext, _ aver.Namespace) {
	aver.Always(len(value) == MODULUS_SIZE, "An RSA ciphertext has modulus width.")
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

// Signature is one exact-width caller-owned RSA-2048 signature.
type Signature []byte

// Signature_Invariants fixes caller-owned signature width.
func Signature_Invariants(value Signature, _ aver.Namespace) {
	aver.Always(len(value) == MODULUS_SIZE, "An RSA signature has modulus width.")
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

// Digest is one exact caller-owned SHA-256 digest.
type Digest []byte

// Digest_Invariants fixes signature digest width.
func Digest_Invariants(value Digest, _ aver.Namespace) {
	aver.Always(len(value) == HASH_SIZE, "An RSA digest has SHA-256 width.")
}

// Encoded is one exact internal RSA integer encoding.
type Encoded []byte

// Encoded_Invariants fixes the private arithmetic wire width.
func Encoded_Invariants(value Encoded, _ aver.Namespace) {
	aver.Always(len(value) == MODULUS_SIZE, "RSA arithmetic encoding has modulus width.")
}

// Database is the shared exact OAEP and PSS database width.
type Database []byte

// Database_Invariants fixes the padding database width.
func Database_Invariants(value Database, _ aver.Namespace) {
	aver.Always(len(value) == OAEP_DATABASE_SIZE, "RSA padding database has derived width.")
}

// Limbs is one exact little-endian Montgomery integer.
type Limbs []uint64

// Limbs_Invariants fixes RSA-2048 arithmetic width.
func Limbs_Invariants(value Limbs, _ aver.Namespace) {
	aver.Always(len(value) == MODULUS_LIMB_COUNT, "RSA integer has 32 machine words.")
}

// Temporary is one exact Montgomery product with two carry words.
type Temporary []uint64

// Temporary_Invariants fixes reduction scratch width.
func Temporary_Invariants(value Temporary, _ aver.Namespace) {
	aver.Always(
		len(value) == MONTGOMERY_TEMPORARY_LIMB_COUNT,
		"Montgomery scratch has two carry words.",
	)
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
	defer func() { Key_Status_Invariants(status, "Public_Key_Set_Bytes.status") }()
	Public_Key_Destination_Invariants(destination, "Public_Key_Set_Bytes.destination")
	Modulus_Unvalidated_Invariants(modulus, "Public_Key_Set_Bytes.modulus")
	defer func() {
		Public_Key_Invariants(*destination, "Public_Key_Set_Bytes.destination.output")
	}()
	if len(modulus) != MODULUS_SIZE {
		return KEY_STATUS_INPUT_INVALID
	}
	var encoding Modulus
	modulus_replace(&encoding, Encoded(modulus))
	if modulus_encoding_valid(&encoding) != DECISION_TRUE {
		return KEY_STATUS_INPUT_INVALID
	}
	*destination = Public_Key{
		Modulus: encoding, Ready: READY_COMPLETE,
	}
	return KEY_STATUS_OK
}

// Private_Key_Set_Bytes validates both private integers before replacement.
func Private_Key_Set_Bytes(
	destination Private_Key_Destination,
	modulus Modulus_Unvalidated,
	exponent Private_Exponent_Unvalidated,
) (status Key_Status) {
	defer func() { Key_Status_Invariants(status, "Private_Key_Set_Bytes.status") }()
	Private_Key_Destination_Invariants(destination, "Private_Key_Set_Bytes.destination")
	Modulus_Unvalidated_Invariants(modulus, "Private_Key_Set_Bytes.modulus")
	Private_Exponent_Unvalidated_Invariants(exponent, "Private_Key_Set_Bytes.exponent")
	defer func() {
		Private_Key_Invariants(*destination, "Private_Key_Set_Bytes.destination.output")
	}()
	if len(modulus) != MODULUS_SIZE {
		return KEY_STATUS_INPUT_INVALID
	}
	if len(exponent) != MODULUS_SIZE {
		return KEY_STATUS_INPUT_INVALID
	}
	var modulus_encoding Modulus
	var exponent_encoding Private_Exponent
	modulus_replace(&modulus_encoding, Encoded(modulus))
	private_exponent_replace(&exponent_encoding, Encoded(exponent))
	modulus_valid := modulus_encoding_valid(&modulus_encoding)
	exponent_valid := private_exponent_encoding_valid(
		&exponent_encoding, &modulus_encoding,
	)
	if modulus_valid&exponent_valid != DECISION_TRUE {
		return KEY_STATUS_INPUT_INVALID
	}
	*destination = Private_Key{
		Modulus: modulus_encoding, Exponent: exponent_encoding,
		Ready: READY_COMPLETE,
	}
	return KEY_STATUS_OK
}

// Public_Key_From_Private retains only shared modulus authority in caller storage.
func Public_Key_From_Private(
	destination Public_Key_Destination, private_key Private_Key,
) {
	defer func() {
		Public_Key_Invariants(*destination, "Public_Key_From_Private.destination.output")
	}()
	Public_Key_Destination_Invariants(destination, "Public_Key_From_Private.destination")
	Private_Key_Invariants(private_key, "Public_Key_From_Private.private_key")
	private_key_require(private_key)
	*destination = Public_Key{
		Modulus: private_key.Modulus, Ready: READY_COMPLETE,
	}
}

// Public_Key_Bytes_Into copies the fixed modulus into caller storage.
func Public_Key_Bytes_Into(destination Modulus_Encoding, public_key Public_Key) {
	defer func() {
		Modulus_Encoding_Invariants(destination, "Public_Key_Bytes_Into.destination.output")
	}()
	Modulus_Encoding_Invariants(destination, "Public_Key_Bytes_Into.destination")
	Public_Key_Invariants(public_key, "Public_Key_Bytes_Into.public_key")
	public_key_require(public_key)
	modulus_copy(Encoded(destination), &public_key.Modulus)
}

// Encrypt_OAEP_SHA_256 commits one fixed ciphertext after complete padding construction.
func Encrypt_OAEP_SHA_256(
	destination Ciphertext,
	generator prng.Source,
	public_key Public_Key,
	message Message,
) {
	defer func() { Ciphertext_Invariants(destination, "Encrypt.destination.output") }()
	Ciphertext_Invariants(destination, "Encrypt.destination")
	Public_Key_Invariants(public_key, "Encrypt.public_key")
	Message_Invariants(message, "Encrypt.message")
	prng.Source_Invariants(generator, "Encrypt.generator")
	public_key_require(public_key)
	var encoded_storage [MODULUS_SIZE]byte
	encoded := Encoded(encoded_storage[:])
	var seed_storage [HASH_SIZE]byte
	seed := Digest(seed_storage[:])
	prng.Source_Read(generator, prng.Sink(seed))
	database := Database(encoded[binary.UINT_8_SIZE+HASH_SIZE:])
	var empty_hash_storage [HASH_SIZE]byte
	empty_hash := Digest(empty_hash_storage[:])
	sha256.Checksum_Into(sha256.Destination(empty_hash), sha256.KIND_SHA_256, nil)
	copy(database[:HASH_SIZE], empty_hash[:])
	delimiter_index := OAEP_DATABASE_SIZE - len(message) - binary.UINT_8_SIZE
	database[delimiter_index] = binary.UINT_8_SIZE
	copy(database[delimiter_index+binary.UINT_8_SIZE:], message)
	var database_mask_storage [OAEP_DATABASE_SIZE]byte
	database_mask := Database(database_mask_storage[:])
	mask_database(database_mask, seed)
	for index := range database {
		database[index] ^= database_mask[index]
	}
	var seed_mask_storage [HASH_SIZE]byte
	seed_mask := Digest(seed_mask_storage[:])
	mask_digest(seed_mask, database)
	for index := range seed {
		encoded[binary.UINT_8_SIZE+index] = seed[index] ^ seed_mask[index]
	}
	var result_storage [MODULUS_SIZE]byte
	result := Encoded(result_storage[:])
	rsa_public_operation(result, encoded, &public_key.Modulus)
	copy(destination, result)
}

// Decrypt_OAEP_SHA_256 validates padding before transactional plaintext commit.
func Decrypt_OAEP_SHA_256(
	destination Destination,
	private_key Private_Key,
	ciphertext Ciphertext_Unvalidated,
) (count Count, _ Decrypt_Status) {
	defer func() { Count_Invariants(count, "Decrypt.count") }()
	Destination_Invariants(destination, "Decrypt.destination")
	Private_Key_Invariants(private_key, "Decrypt.private_key")
	Ciphertext_Unvalidated_Invariants(ciphertext, "Decrypt.ciphertext")
	status := DECRYPT_STATUS_INPUT_INVALID
	defer func() { Decrypt_Status_Invariants(status, "Decrypt.status") }()
	private_key_require(private_key)
	if len(ciphertext) != MODULUS_SIZE {
		return COUNT_MINIMUM, status
	}
	var ciphertext_storage, encoded_storage [MODULUS_SIZE]byte
	ciphertext_encoding := Encoded(ciphertext_storage[:])
	encoded := Encoded(encoded_storage[:])
	copy(ciphertext_encoding, ciphertext)
	if integer_encoding_canonical(
		ciphertext_encoding, &private_key.Modulus,
	) != DECISION_TRUE {
		return COUNT_MINIMUM, status
	}
	rsa_private_operation(
		encoded, ciphertext_encoding, &private_key.Modulus, &private_key.Exponent,
	)
	var seed_storage [HASH_SIZE]byte
	seed := Digest(seed_storage[:])
	copy(seed, encoded[binary.UINT_8_SIZE:binary.UINT_8_SIZE+HASH_SIZE])
	database := Database(encoded[binary.UINT_8_SIZE+HASH_SIZE:])
	var seed_mask_storage [HASH_SIZE]byte
	seed_mask := Digest(seed_mask_storage[:])
	mask_digest(seed_mask, database)
	for index := range seed {
		seed[index] ^= seed_mask[index]
	}
	var database_mask_storage [OAEP_DATABASE_SIZE]byte
	database_mask := Database(database_mask_storage[:])
	mask_database(database_mask, seed)
	for index := range database {
		database[index] ^= database_mask[index]
	}
	message_count, valid := oaep_message_count(encoded, database)
	if valid != DECISION_TRUE {
		return COUNT_MINIMUM, status
	}
	count = message_count
	if len(destination) < int(count) {
		status = DECRYPT_STATUS_DESTINATION_TOO_SMALL
		return count, status
	}
	message_start := OAEP_DATABASE_SIZE - int(count)
	var plaintext [MESSAGE_SIZE_MAXIMUM]byte
	copy(plaintext[:count], database[message_start:])
	copy(destination[:count], plaintext[:count])
	status = DECRYPT_STATUS_OK
	return count, status
}

// Sign_PSS_SHA_256 uses one digest-width injected salt and fixed private work.
func Sign_PSS_SHA_256(
	destination Signature,
	generator prng.Source,
	private_key Private_Key,
	digest Digest,
) {
	defer func() { Signature_Invariants(destination, "Sign_PSS.destination.output") }()
	Signature_Invariants(destination, "Sign_PSS.destination")
	Private_Key_Invariants(private_key, "Sign_PSS.private_key")
	Digest_Invariants(digest, "Sign_PSS.digest")
	prng.Source_Invariants(generator, "Sign_PSS.generator")
	private_key_require(private_key)
	var salt_storage [HASH_SIZE]byte
	salt := Digest(salt_storage[:])
	prng.Source_Read(generator, prng.Sink(salt))
	var encoded_storage, signature_storage [MODULUS_SIZE]byte
	encoded := Encoded(encoded_storage[:])
	signature := Encoded(signature_storage[:])
	pss_encoding(encoded, digest, salt)
	rsa_private_operation(
		signature, encoded, &private_key.Modulus, &private_key.Exponent,
	)
	copy(destination, signature)
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
	if len(signature) != MODULUS_SIZE {
		return false
	}
	var signature_storage, encoded_storage [MODULUS_SIZE]byte
	signature_encoding := Encoded(signature_storage[:])
	encoded := Encoded(encoded_storage[:])
	copy(signature_encoding, signature)
	if integer_encoding_canonical(
		signature_encoding, &public_key.Modulus,
	) != DECISION_TRUE {
		return false
	}
	rsa_public_operation(encoded, signature_encoding, &public_key.Modulus)
	return Verification(
		pss_encoding_valid(encoded, digest) == DECISION_TRUE,
	)
}

// Sign_PKCS1_V1_5_SHA_256 constructs the exact SHA-256 DigestInfo block.
func Sign_PKCS1_V1_5_SHA_256(
	destination Signature, private_key Private_Key, digest Digest,
) {
	defer func() { Signature_Invariants(destination, "Sign_PKCS1.destination.output") }()
	Signature_Invariants(destination, "Sign_PKCS1.destination")
	Private_Key_Invariants(private_key, "Sign_PKCS1.private_key")
	Digest_Invariants(digest, "Sign_PKCS1.digest")
	private_key_require(private_key)
	var encoded_storage, signature_storage [MODULUS_SIZE]byte
	encoded := Encoded(encoded_storage[:])
	signature := Encoded(signature_storage[:])
	pkcs1_v1_5_encoding(encoded, digest)
	rsa_private_operation(
		signature, encoded, &private_key.Modulus, &private_key.Exponent,
	)
	copy(destination, signature)
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
	if len(signature) != MODULUS_SIZE {
		return false
	}
	var signature_storage, encoded_storage, want_storage [MODULUS_SIZE]byte
	signature_encoding := Encoded(signature_storage[:])
	encoded := Encoded(encoded_storage[:])
	want := Encoded(want_storage[:])
	copy(signature_encoding, signature)
	if integer_encoding_canonical(
		signature_encoding, &public_key.Modulus,
	) != DECISION_TRUE {
		return false
	}
	rsa_public_operation(encoded, signature_encoding, &public_key.Modulus)
	pkcs1_v1_5_encoding(want, digest)
	difference := bits.WORD_8_MINIMUM
	for index := range encoded {
		difference |= encoded[index] ^ want[index]
	}
	return Verification(difference == bits.WORD_8_MINIMUM)
}

func oaep_message_count(
	encoded Encoded, database Database,
) (message_count Count, _ Decision) {
	defer func() { Count_Invariants(message_count, "oaep_message_count.message_count") }()
	Encoded_Invariants(encoded, "oaep_message_count.encoded")
	Database_Invariants(database, "oaep_message_count.database")
	var valid Decision
	defer func() { Decision_Invariants(valid, "oaep_message_count.valid") }()
	var empty_hash_storage [HASH_SIZE]byte
	empty_hash := Digest(empty_hash_storage[:])
	sha256.Checksum_Into(sha256.Destination(empty_hash), sha256.KIND_SHA_256, nil)
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
	valid = Decision((difference_word|-difference_word)>>
		(bits.BIT_COUNT_64_MAXIMUM-binary.UINT_8_SIZE) ^ binary.UINT_8_SIZE)
	message_start := int(selected_index) + binary.UINT_8_SIZE
	return Count(OAEP_DATABASE_SIZE - message_start), valid
}

func pss_encoding(encoded Encoded, digest Digest, salt Digest) {
	Encoded_Invariants(encoded, "pss_encoding.encoded")
	Digest_Invariants(digest, "pss_encoding.digest")
	Digest_Invariants(salt, "pss_encoding.salt")
	var hash_storage [HASH_SIZE]byte
	hash := Digest(hash_storage[:])
	pss_hash(hash, digest, salt)
	database := Database(encoded[:PSS_DATABASE_SIZE])
	database[PSS_PADDING_SIZE] = binary.UINT_8_SIZE
	copy(database[PSS_PADDING_SIZE+binary.UINT_8_SIZE:], salt)
	var mask_storage [PSS_DATABASE_SIZE]byte
	mask := Database(mask_storage[:])
	mask_pss_database(mask, hash)
	for index := range database {
		database[index] ^= mask[index]
	}
	database[bits.BIT_COUNT_MINIMUM] &=
		byte(bits.WORD_8_MAXIMUM >> binary.UINT_8_SIZE)
	copy(encoded[PSS_DATABASE_SIZE:PSS_DATABASE_SIZE+HASH_SIZE], hash)
	encoded[MODULUS_SIZE-binary.UINT_8_SIZE] = PSS_TRAILER
}

func pss_encoding_valid(
	encoded Encoded, digest Digest,
) (valid Decision) {
	defer func() { Decision_Invariants(valid, "pss_encoding_valid.valid") }()
	Encoded_Invariants(encoded, "pss_encoding_valid.encoded")
	Digest_Invariants(digest, "pss_encoding_valid.digest")
	difference := encoded[MODULUS_SIZE-binary.UINT_8_SIZE] ^ PSS_TRAILER
	difference |= encoded[bits.BIT_COUNT_MINIMUM] & byte(binary.UINT_8_SIZE<<
		(bits.BIT_COUNT_8_MAXIMUM-binary.UINT_8_SIZE))
	var hash_storage [HASH_SIZE]byte
	hash := Digest(hash_storage[:])
	copy(hash, encoded[PSS_DATABASE_SIZE:PSS_DATABASE_SIZE+HASH_SIZE])
	var database_storage [PSS_DATABASE_SIZE]byte
	database := Database(database_storage[:])
	copy(database, encoded[:PSS_DATABASE_SIZE])
	var mask_storage [PSS_DATABASE_SIZE]byte
	mask := Database(mask_storage[:])
	mask_pss_database(mask, hash)
	for index := range database {
		database[index] ^= mask[index]
	}
	database[bits.BIT_COUNT_MINIMUM] &=
		byte(bits.WORD_8_MAXIMUM >> binary.UINT_8_SIZE)
	for index := bits.BIT_COUNT_MINIMUM; index < PSS_PADDING_SIZE; index++ {
		difference |= database[index]
	}
	difference |= database[PSS_PADDING_SIZE] ^ binary.UINT_8_SIZE
	var salt_storage, want_storage [HASH_SIZE]byte
	salt := Digest(salt_storage[:])
	want := Digest(want_storage[:])
	copy(salt, database[PSS_PADDING_SIZE+binary.UINT_8_SIZE:])
	pss_hash(want, digest, salt)
	for index := range hash {
		difference |= hash[index] ^ want[index]
	}
	word := uint64(difference)
	valid = Decision((word|-word)>>
		(bits.BIT_COUNT_64_MAXIMUM-binary.UINT_8_SIZE) ^ binary.UINT_8_SIZE)
	return valid
}

func pss_hash(destination Digest, digest Digest, salt Digest) {
	Digest_Invariants(destination, "pss_hash.destination")
	Digest_Invariants(digest, "pss_hash.digest")
	Digest_Invariants(salt, "pss_hash.salt")
	var hash sha256.Digest
	sha256.Digest_Init(&hash, sha256.KIND_SHA_256)
	var prefix [PSS_PREFIX_SIZE]byte
	sha256.Digest_Write(&hash, prefix[:])
	sha256.Digest_Write(&hash, sha256.Source(digest))
	sha256.Digest_Write(&hash, sha256.Source(salt))
	sha256.Digest_Sum_Into(&hash, sha256.Destination(destination))
}

func pkcs1_v1_5_encoding(encoded Encoded, digest Digest) {
	Encoded_Invariants(encoded, "pkcs1_v1_5_encoding.encoded")
	Digest_Invariants(digest, "pkcs1_v1_5_encoding.digest")
	encoded[binary.UINT_8_SIZE] = BLOCK_TYPE_SIGNATURE
	delimiter_index := MODULUS_SIZE - PKCS1_SHA_256_PREFIX_SIZE - HASH_SIZE -
		binary.UINT_8_SIZE
	for index := binary.UINT_16_SIZE; index < delimiter_index; index++ {
		encoded[index] = PADDING_BYTE_SIGNATURE
	}
	copy(encoded[delimiter_index+binary.UINT_8_SIZE:], PKCS1_SHA_256_PREFIX)
	copy(encoded[MODULUS_SIZE-HASH_SIZE:], digest)
}

func mask_database(destination Database, seed Digest) {
	Database_Invariants(destination, "mask_database.destination")
	Digest_Invariants(seed, "mask_database.seed")
	mask_sha_256_blocks(destination, seed)
}

func mask_pss_database(destination Database, seed Digest) {
	Database_Invariants(destination, "mask_pss_database.destination")
	Digest_Invariants(seed, "mask_pss_database.seed")
	mask_sha_256_blocks(destination, seed)
}

func mask_digest(destination Digest, seed Database) {
	Digest_Invariants(destination, "mask_digest.destination")
	Database_Invariants(seed, "mask_digest.seed")
	var hash sha256.Digest
	sha256.Digest_Init(&hash, sha256.KIND_SHA_256)
	sha256.Digest_Write(&hash, sha256.Source(seed))
	var counter [MASK_COUNTER_SIZE]byte
	sha256.Digest_Write(&hash, counter[:])
	sha256.Digest_Sum_Into(&hash, sha256.Destination(destination))
}

func mask_sha_256_blocks(
	destination Database, seed Digest,
) {
	Database_Invariants(destination, "mask_sha_256_blocks.destination")
	Digest_Invariants(seed, "mask_sha_256_blocks.seed")
	var counter [MASK_COUNTER_SIZE]byte
	for block_index := bits.BIT_COUNT_MINIMUM; block_index < MASK_BLOCK_COUNT; block_index++ {
		binary.Put_Uint_32(
			counter[:], binary.Word_32(block_index), binary.BIG_ENDIAN,
		)
		var hash sha256.Digest
		sha256.Digest_Init(&hash, sha256.KIND_SHA_256)
		sha256.Digest_Write(&hash, sha256.Source(seed))
		sha256.Digest_Write(&hash, counter[:])
		var value_storage [HASH_SIZE]byte
		value := Digest(value_storage[:])
		sha256.Digest_Sum_Into(&hash, sha256.Destination(value))
		start := block_index * HASH_SIZE
		end := start + HASH_SIZE
		if end > len(destination) {
			end = len(destination)
		}
		copy(destination[start:end], value[:end-start])
	}
}

func rsa_public_operation(
	destination Encoded,
	source Encoded,
	modulus Modulus_Destination,
) {
	Encoded_Invariants(destination, "rsa_public_operation.destination")
	Encoded_Invariants(source, "rsa_public_operation.source")
	Modulus_Destination_Invariants(modulus, "rsa_public_operation.modulus")
	var exponent_storage [MODULUS_LIMB_COUNT]uint64
	exponent := Limbs(exponent_storage[:])
	exponent[bits.BIT_COUNT_MINIMUM] = uint64(PUBLIC_EXPONENT)
	rsa_operation(destination, source, modulus, exponent)
}

func rsa_private_operation(
	destination Encoded,
	source Encoded,
	modulus Modulus_Destination,
	exponent_encoding Private_Exponent_Destination,
) {
	Encoded_Invariants(destination, "rsa_private_operation.destination")
	Encoded_Invariants(source, "rsa_private_operation.source")
	Modulus_Destination_Invariants(modulus, "rsa_private_operation.modulus")
	Private_Exponent_Destination_Invariants(
		exponent_encoding, "rsa_private_operation.exponent_encoding",
	)
	var exponent_storage [MODULUS_LIMB_COUNT]uint64
	exponent := Limbs(exponent_storage[:])
	var exponent_bytes [MODULUS_SIZE]byte
	private_exponent_copy(Encoded(exponent_bytes[:]), exponent_encoding)
	integer_decode(exponent, Encoded(exponent_bytes[:]))
	rsa_operation(destination, source, modulus, exponent)
}

func rsa_operation(
	destination Encoded,
	source Encoded,
	modulus_encoding Modulus_Destination,
	exponent Limbs,
) {
	Encoded_Invariants(destination, "rsa_operation.destination")
	Encoded_Invariants(source, "rsa_operation.source")
	Modulus_Destination_Invariants(modulus_encoding, "rsa_operation.modulus_encoding")
	Limbs_Invariants(exponent, "rsa_operation.exponent")
	var modulus_storage, source_storage [MODULUS_LIMB_COUNT]uint64
	modulus := Limbs(modulus_storage[:])
	source_integer := Limbs(source_storage[:])
	var modulus_encoding_storage [MODULUS_SIZE]byte
	modulus_copy(Encoded(modulus_encoding_storage[:]), modulus_encoding)
	integer_decode(modulus, Encoded(modulus_encoding_storage[:]))
	integer_decode(source_integer, source)
	var r_squared_storage [MODULUS_LIMB_COUNT]uint64
	r_squared := Limbs(r_squared_storage[:])
	montgomery_r_squared(r_squared, modulus)
	rsa_exponentiate(destination, source_integer, exponent, r_squared, modulus)
}

func rsa_exponentiate(
	destination Encoded,
	source Limbs,
	exponent Limbs,
	r_squared Limbs,
	modulus Limbs,
) {
	Encoded_Invariants(destination, "rsa_exponentiate.destination")
	Limbs_Invariants(source, "rsa_exponentiate.source")
	Limbs_Invariants(exponent, "rsa_exponentiate.exponent")
	Limbs_Invariants(r_squared, "rsa_exponentiate.r_squared")
	Limbs_Invariants(modulus, "rsa_exponentiate.modulus")
	var one, base, result [MODULUS_LIMB_COUNT]uint64
	one[bits.BIT_COUNT_MINIMUM] = binary.UINT_8_SIZE
	montgomery_multiply(Limbs(base[:]), source, r_squared, modulus)
	montgomery_multiply(Limbs(result[:]), Limbs(one[:]), r_squared, modulus)
	for bit_count := MODULUS_BIT_COUNT; bit_count > bits.BIT_COUNT_MINIMUM; bit_count-- {
		bit_index := bit_count - binary.UINT_8_SIZE
		var square, product [MODULUS_LIMB_COUNT]uint64
		montgomery_multiply(
			Limbs(square[:]), Limbs(result[:]), Limbs(result[:]), modulus,
		)
		montgomery_multiply(
			Limbs(product[:]), Limbs(square[:]), Limbs(base[:]), modulus,
		)
		word_index := bit_index / bits.BIT_COUNT_64_MAXIMUM
		word_shift := uint(bit_index % bits.BIT_COUNT_64_MAXIMUM)
		bit := exponent[word_index] >> word_shift & binary.UINT_8_SIZE
		mask := uint64(bits.WORD_64_MINIMUM) - bit
		for limb_index := range result {
			result[limb_index] = square[limb_index] ^
				mask&(product[limb_index]^square[limb_index])
		}
	}
	var decoded [MODULUS_LIMB_COUNT]uint64
	montgomery_multiply(
		Limbs(decoded[:]), Limbs(result[:]), Limbs(one[:]), modulus,
	)
	integer_encode(destination, Limbs(decoded[:]))
}

func montgomery_multiply(destination Limbs, left Limbs, right Limbs, modulus Limbs) {
	Limbs_Invariants(destination, "montgomery_multiply.destination")
	Limbs_Invariants(left, "montgomery_multiply.left")
	Limbs_Invariants(right, "montgomery_multiply.right")
	Limbs_Invariants(modulus, "montgomery_multiply.modulus")
	n0_inverse := uint64(binary.UINT_8_SIZE)
	for range MONTGOMERY_INVERSE_COUNT {
		n0_inverse *= binary.UINT_16_SIZE - modulus[bits.BIT_COUNT_MINIMUM]*n0_inverse
	}
	n0_inverse = bits.WORD_64_MINIMUM - n0_inverse
	var temporary_storage [MONTGOMERY_TEMPORARY_LIMB_COUNT]uint64
	temporary := Temporary(temporary_storage[:])
	for word_index := range MODULUS_LIMB_COUNT {
		multiplicands := [...]Limbs{left, modulus}
		for operation_index, multiplicand := range multiplicands {
			multiplier := right[word_index]
			if operation_index == binary.UINT_8_SIZE {
				multiplier = temporary[bits.BIT_COUNT_MINIMUM] * n0_inverse
			}
			carry := uint64(bits.WORD_64_MINIMUM)
			for limb_index := range MODULUS_LIMB_COUNT {
				left_word := multiplicand[limb_index]
				left_low := left_word & MONTGOMERY_HALF_MASK
				left_high := left_word >> bits.BIT_COUNT_32_MAXIMUM
				right_low := multiplier & MONTGOMERY_HALF_MASK
				right_high := multiplier >> bits.BIT_COUNT_32_MAXIMUM
				partial := left_low * right_low
				middle_first := left_high*right_low +
					partial>>bits.BIT_COUNT_32_MAXIMUM
				middle_second := left_low*right_high +
					middle_first&MONTGOMERY_HALF_MASK
				high := left_high*right_high +
					middle_first>>bits.BIT_COUNT_32_MAXIMUM +
					middle_second>>bits.BIT_COUNT_32_MAXIMUM
				low := left_word * multiplier
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
		for shift_index := range MODULUS_LIMB_COUNT + binary.UINT_8_SIZE {
			temporary[shift_index] = temporary[shift_index+binary.UINT_8_SIZE]
		}
		temporary[MODULUS_LIMB_COUNT+binary.UINT_8_SIZE] = bits.WORD_64_MINIMUM
	}
	montgomery_reduce(destination, temporary, modulus)
}

func montgomery_reduce(destination Limbs, temporary Temporary, modulus Limbs) {
	Limbs_Invariants(destination, "montgomery_reduce.destination")
	Temporary_Invariants(temporary, "montgomery_reduce.temporary")
	Limbs_Invariants(modulus, "montgomery_reduce.modulus")
	var value, reduced [MODULUS_LIMB_COUNT]uint64
	copy(value[:], temporary[:MODULUS_LIMB_COUNT])
	borrow := uint64(bits.WORD_64_MINIMUM)
	for limb_index := range reduced {
		partial := value[limb_index] - modulus[limb_index]
		partial_borrow := ((^value[limb_index] & modulus[limb_index]) |
			(^(value[limb_index] ^ modulus[limb_index]) & partial)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		word := partial - borrow
		borrow_borrow := ((^partial & borrow) | (^(partial ^ borrow) & word)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		reduced[limb_index] = word
		borrow = partial_borrow | borrow_borrow
	}
	top := temporary[MODULUS_LIMB_COUNT]
	top_nonzero := (top | -top) >> (bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
	mask := uint64(bits.WORD_64_MINIMUM) -
		(top_nonzero | (borrow ^ binary.UINT_8_SIZE))
	for limb_index := range destination {
		destination[limb_index] = value[limb_index] ^
			mask&(reduced[limb_index]^value[limb_index])
	}
}

func montgomery_r_squared(destination Limbs, modulus Limbs) {
	Limbs_Invariants(destination, "montgomery_r_squared.destination")
	Limbs_Invariants(modulus, "montgomery_r_squared.modulus")
	destination[bits.BIT_COUNT_MINIMUM] = binary.UINT_8_SIZE
	for range MODULUS_BIT_COUNT + MODULUS_BIT_COUNT {
		var sum, reduced [MODULUS_LIMB_COUNT]uint64
		carry := uint64(bits.WORD_64_MINIMUM)
		for limb_index := range sum {
			partial := destination[limb_index] + destination[limb_index]
			partial_carry := ((destination[limb_index] & destination[limb_index]) |
				((destination[limb_index] | destination[limb_index]) & ^partial)) >>
				(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
			word := partial + carry
			carry_carry := ((partial & carry) | ((partial | carry) & ^word)) >>
				(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
			sum[limb_index] = word
			carry = partial_carry | carry_carry
		}
		borrow := uint64(bits.WORD_64_MINIMUM)
		for limb_index := range reduced {
			partial := sum[limb_index] - modulus[limb_index]
			partial_borrow := ((^sum[limb_index] & modulus[limb_index]) |
				(^(sum[limb_index] ^ modulus[limb_index]) & partial)) >>
				(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
			word := partial - borrow
			borrow_borrow := ((^partial & borrow) | (^(partial ^ borrow) & word)) >>
				(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
			reduced[limb_index] = word
			borrow = partial_borrow | borrow_borrow
		}
		mask := uint64(bits.WORD_64_MINIMUM) -
			(carry | (borrow ^ binary.UINT_8_SIZE))
		for limb_index := range destination {
			destination[limb_index] = sum[limb_index] ^
				mask&(reduced[limb_index]^sum[limb_index])
		}
	}
}

func integer_decode(
	destination Limbs, source Encoded,
) {
	Limbs_Invariants(destination, "integer_decode.destination")
	Encoded_Invariants(source, "integer_decode.source")
	for limb_index := bits.BIT_COUNT_MINIMUM; limb_index < MODULUS_LIMB_COUNT; limb_index++ {
		source_index := MODULUS_SIZE - (limb_index+binary.UINT_8_SIZE)*binary.UINT_64_SIZE
		destination[limb_index] = uint64(binary.Uint_64(
			binary.Bytes(source[source_index:source_index+binary.UINT_64_SIZE]),
			binary.BIG_ENDIAN,
		))
	}
}

func integer_encode(
	destination Encoded, source Limbs,
) {
	Encoded_Invariants(destination, "integer_encode.destination")
	Limbs_Invariants(source, "integer_encode.source")
	for limb_index := bits.BIT_COUNT_MINIMUM; limb_index < MODULUS_LIMB_COUNT; limb_index++ {
		destination_index := MODULUS_SIZE -
			(limb_index+binary.UINT_8_SIZE)*binary.UINT_64_SIZE
		destination_end := destination_index + binary.UINT_64_SIZE
		binary.Put_Uint_64(
			binary.Bytes(destination[destination_index:destination_end]),
			binary.Word_64(source[limb_index]), binary.BIG_ENDIAN,
		)
	}
}

func modulus_encoding_valid(
	modulus Modulus_Destination,
) (valid Decision) {
	defer func() { Decision_Invariants(valid, "modulus_encoding_valid.valid") }()
	Modulus_Destination_Invariants(modulus, "modulus_encoding_valid.modulus")
	var encoding_storage [MODULUS_SIZE]byte
	encoding := Encoded(encoding_storage[:])
	modulus_copy(encoding, modulus)
	top_bit := uint64(
		encoding[bits.BIT_COUNT_MINIMUM] >>
			(bits.BIT_COUNT_8_MAXIMUM - binary.UINT_8_SIZE),
	)
	odd := uint64(encoding[MODULUS_SIZE-binary.UINT_8_SIZE] & byte(binary.UINT_8_SIZE))
	valid = Decision(top_bit & odd)
	return valid
}

func private_exponent_encoding_valid(
	exponent Private_Exponent_Destination, modulus Modulus_Destination,
) (valid Decision) {
	defer func() {
		Decision_Invariants(valid, "private_exponent_encoding_valid.valid")
	}()
	Private_Exponent_Destination_Invariants(
		exponent, "private_exponent_encoding_valid.exponent",
	)
	Modulus_Destination_Invariants(modulus, "private_exponent_encoding_valid.modulus")
	var exponent_integer, modulus_integer [MODULUS_LIMB_COUNT]uint64
	var exponent_bytes, modulus_encoding [MODULUS_SIZE]byte
	private_exponent_copy(Encoded(exponent_bytes[:]), exponent)
	modulus_copy(Encoded(modulus_encoding[:]), modulus)
	integer_decode(Limbs(exponent_integer[:]), Encoded(exponent_bytes[:]))
	integer_decode(Limbs(modulus_integer[:]), Encoded(modulus_encoding[:]))
	var difference [MODULUS_LIMB_COUNT]uint64
	canonical := limbs_subtract(
		Limbs(difference[:]), Limbs(exponent_integer[:]), Limbs(modulus_integer[:]),
	)
	nonzero := limbs_is_zero(Limbs(exponent_integer[:]))
	valid = canonical & (nonzero ^ DECISION_TRUE)
	return valid
}

func integer_encoding_canonical(
	encoding Encoded, modulus Modulus_Destination,
) (canonical Decision) {
	defer func() {
		Decision_Invariants(canonical, "integer_encoding_canonical.canonical")
	}()
	Encoded_Invariants(encoding, "integer_encoding_canonical.encoding")
	Modulus_Destination_Invariants(modulus, "integer_encoding_canonical.modulus")
	var value, modulus_integer [MODULUS_LIMB_COUNT]uint64
	integer_decode(Limbs(value[:]), encoding)
	var modulus_encoding [MODULUS_SIZE]byte
	modulus_copy(Encoded(modulus_encoding[:]), modulus)
	integer_decode(Limbs(modulus_integer[:]), Encoded(modulus_encoding[:]))
	var difference [MODULUS_LIMB_COUNT]uint64
	return limbs_subtract(
		Limbs(difference[:]), Limbs(value[:]), Limbs(modulus_integer[:]),
	)
}

func limbs_subtract(
	destination Limbs, left Limbs, right Limbs,
) (borrow Decision) {
	defer func() { Decision_Invariants(borrow, "limbs_subtract.borrow") }()
	Limbs_Invariants(destination, "limbs_subtract.destination")
	Limbs_Invariants(left, "limbs_subtract.left")
	Limbs_Invariants(right, "limbs_subtract.right")
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
	return Decision(borrow_value)
}

func limbs_is_zero(
	value Limbs,
) (zero Decision) {
	defer func() { Decision_Invariants(zero, "limbs_is_zero.zero") }()
	Limbs_Invariants(value, "limbs_is_zero.value")
	difference := uint64(bits.WORD_64_MINIMUM)
	for limb_index := range value {
		difference |= value[limb_index]
	}
	zero = Decision((difference|-difference)>>
		(bits.BIT_COUNT_64_MAXIMUM-binary.UINT_8_SIZE) ^ binary.UINT_8_SIZE)
	return zero
}

func modulus_replace(destination Modulus_Destination, source Encoded) {
	Modulus_Destination_Invariants(destination, "modulus_replace.destination")
	Encoded_Invariants(source, "modulus_replace.source")
	for chunk_index := range INTEGER_CHUNK_COUNT {
		var chunk Integer_Chunk_Storage
		for lane_index := range INTEGER_CHUNK_LANE_COUNT {
			start := chunk_index*INTEGER_CHUNK_SIZE + lane_index*binary.UINT_64_SIZE
			end := start + binary.UINT_64_SIZE
			word := binary.Uint_64(
				binary.Bytes(source[start:end]), binary.LITTLE_ENDIAN,
			)
			switch lane_index {
			case 0:
				chunk.Lane_0 = Integer_Lane_0(word)
			case 1:
				chunk.Lane_1 = Integer_Lane_1(word)
			case 2:
				chunk.Lane_2 = Integer_Lane_2(word)
			default:
				chunk.Lane_3 = Integer_Lane_3(word)
			}
		}
		switch chunk_index {
		case 0:
			destination.Chunk_0 = Integer_Chunk_0(chunk)
		case 1:
			destination.Chunk_1 = Integer_Chunk_1(chunk)
		case 2:
			destination.Chunk_2 = Integer_Chunk_2(chunk)
		case 3:
			destination.Chunk_3 = Integer_Chunk_3(chunk)
		case 4:
			destination.Chunk_4 = Integer_Chunk_4(chunk)
		case 5:
			destination.Chunk_5 = Integer_Chunk_5(chunk)
		case 6:
			destination.Chunk_6 = Integer_Chunk_6(chunk)
		default:
			destination.Chunk_7 = Integer_Chunk_7(chunk)
		}
	}
}

func modulus_copy(destination Encoded, source Modulus_Destination) {
	Encoded_Invariants(destination, "modulus_copy.destination")
	Modulus_Destination_Invariants(source, "modulus_copy.source")
	for chunk_index := range INTEGER_CHUNK_COUNT {
		var chunk Integer_Chunk_Storage
		switch chunk_index {
		case 0:
			chunk = Integer_Chunk_Storage(source.Chunk_0)
		case 1:
			chunk = Integer_Chunk_Storage(source.Chunk_1)
		case 2:
			chunk = Integer_Chunk_Storage(source.Chunk_2)
		case 3:
			chunk = Integer_Chunk_Storage(source.Chunk_3)
		case 4:
			chunk = Integer_Chunk_Storage(source.Chunk_4)
		case 5:
			chunk = Integer_Chunk_Storage(source.Chunk_5)
		case 6:
			chunk = Integer_Chunk_Storage(source.Chunk_6)
		default:
			chunk = Integer_Chunk_Storage(source.Chunk_7)
		}
		lanes := [INTEGER_CHUNK_LANE_COUNT]uint64{
			uint64(chunk.Lane_0), uint64(chunk.Lane_1),
			uint64(chunk.Lane_2), uint64(chunk.Lane_3),
		}
		for lane_index, word := range lanes {
			start := chunk_index*INTEGER_CHUNK_SIZE + lane_index*binary.UINT_64_SIZE
			binary.Put_Uint_64(
				binary.Bytes(destination[start:start+binary.UINT_64_SIZE]),
				binary.Word_64(word), binary.LITTLE_ENDIAN,
			)
		}
	}
}

func modulus_bytes(value Modulus_Destination) {
	Modulus_Destination_Invariants(value, "modulus_bytes.value")
	var destination [MODULUS_SIZE]byte
	modulus_copy(Encoded(destination[:]), value)
	modulus_replace(value, Encoded(destination[:]))
}

func private_exponent_replace(
	destination Private_Exponent_Destination, source Encoded,
) {
	Private_Exponent_Destination_Invariants(
		destination, "private_exponent_replace.destination",
	)
	Encoded_Invariants(source, "private_exponent_replace.source")
	for chunk_index := range INTEGER_CHUNK_COUNT {
		var chunk Integer_Chunk_Storage
		for lane_index := range INTEGER_CHUNK_LANE_COUNT {
			start := chunk_index*INTEGER_CHUNK_SIZE + lane_index*binary.UINT_64_SIZE
			end := start + binary.UINT_64_SIZE
			word := binary.Uint_64(
				binary.Bytes(source[start:end]), binary.LITTLE_ENDIAN,
			)
			switch lane_index {
			case 0:
				chunk.Lane_0 = Integer_Lane_0(word)
			case 1:
				chunk.Lane_1 = Integer_Lane_1(word)
			case 2:
				chunk.Lane_2 = Integer_Lane_2(word)
			default:
				chunk.Lane_3 = Integer_Lane_3(word)
			}
		}
		switch chunk_index {
		case 0:
			destination.Chunk_0 = Private_Exponent_Chunk_0(chunk)
		case 1:
			destination.Chunk_1 = Private_Exponent_Chunk_1(chunk)
		case 2:
			destination.Chunk_2 = Private_Exponent_Chunk_2(chunk)
		case 3:
			destination.Chunk_3 = Private_Exponent_Chunk_3(chunk)
		case 4:
			destination.Chunk_4 = Private_Exponent_Chunk_4(chunk)
		case 5:
			destination.Chunk_5 = Private_Exponent_Chunk_5(chunk)
		case 6:
			destination.Chunk_6 = Private_Exponent_Chunk_6(chunk)
		default:
			destination.Chunk_7 = Private_Exponent_Chunk_7(chunk)
		}
	}
}

func private_exponent_copy(
	destination Encoded, source Private_Exponent_Destination,
) {
	Encoded_Invariants(destination, "private_exponent_copy.destination")
	Private_Exponent_Destination_Invariants(source, "private_exponent_copy.source")
	for chunk_index := range INTEGER_CHUNK_COUNT {
		var chunk Integer_Chunk_Storage
		switch chunk_index {
		case 0:
			chunk = Integer_Chunk_Storage(source.Chunk_0)
		case 1:
			chunk = Integer_Chunk_Storage(source.Chunk_1)
		case 2:
			chunk = Integer_Chunk_Storage(source.Chunk_2)
		case 3:
			chunk = Integer_Chunk_Storage(source.Chunk_3)
		case 4:
			chunk = Integer_Chunk_Storage(source.Chunk_4)
		case 5:
			chunk = Integer_Chunk_Storage(source.Chunk_5)
		case 6:
			chunk = Integer_Chunk_Storage(source.Chunk_6)
		default:
			chunk = Integer_Chunk_Storage(source.Chunk_7)
		}
		lanes := [INTEGER_CHUNK_LANE_COUNT]uint64{
			uint64(chunk.Lane_0), uint64(chunk.Lane_1),
			uint64(chunk.Lane_2), uint64(chunk.Lane_3),
		}
		for lane_index, word := range lanes {
			start := chunk_index*INTEGER_CHUNK_SIZE + lane_index*binary.UINT_64_SIZE
			binary.Put_Uint_64(
				binary.Bytes(destination[start:start+binary.UINT_64_SIZE]),
				binary.Word_64(word), binary.LITTLE_ENDIAN,
			)
		}
	}
}

func private_exponent_bytes(value Private_Exponent_Destination) {
	Private_Exponent_Destination_Invariants(value, "private_exponent_bytes.value")
	var destination [MODULUS_SIZE]byte
	private_exponent_copy(Encoded(destination[:]), value)
	private_exponent_replace(value, Encoded(destination[:]))
}

func public_key_require(public_key Public_Key) {
	Public_Key_Invariants(public_key, "public_key_require.public_key")
	aver.Always(
		public_key.Ready == READY_COMPLETE,
		"RSA operations require a ready public key.",
	)
}

func private_key_require(private_key Private_Key) {
	Private_Key_Invariants(private_key, "private_key_require.private_key")
	aver.Always(
		private_key.Ready == READY_COMPLETE,
		"RSA operations require a ready private key.",
	)
}
