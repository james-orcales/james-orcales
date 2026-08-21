// Package hkdf derives bounded caller-owned keys through RFC 5869 HMAC expansion.
package hkdf

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/crypto/hmac"
	"local/james-orcales/shared/crypto/md5"
	"local/james-orcales/shared/crypto/sha1"
	"local/james-orcales/shared/crypto/sha256"
	"local/james-orcales/shared/crypto/sha512"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
)

// INPUT_SIZE_MINIMUM admits absent optional salt and info.
const INPUT_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// INPUT_SIZE_MAXIMUM follows repository byte-slice bound.
const INPUT_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// BLOCK_COUNT_MAXIMUM is RFC 5869 single-octet counter capacity.
const BLOCK_COUNT_MAXIMUM = int(bits.WORD_8_MAXIMUM)

// COUNTER_SIZE is RFC 5869 counter encoding width.
const COUNTER_SIZE = binary.UINT_8_SIZE

// OUTPUT_SIZE_MINIMUM admits derivation of no bytes.
const OUTPUT_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// OUTPUT_SIZE_MAXIMUM is 255 blocks of widest supported digest.
const OUTPUT_SIZE_MAXIMUM = hmac.DIGEST_SIZE_MAXIMUM * BLOCK_COUNT_MAXIMUM

// SELECTED_OUTPUT_SIZE_MINIMUM is 255 blocks of shortest supported digest.
const SELECTED_OUTPUT_SIZE_MINIMUM = hmac.DIGEST_SIZE_MINIMUM * BLOCK_COUNT_MAXIMUM

// EXTRACT_DESTINATION_SIZE_MINIMUM admits short-output status paths.
const EXTRACT_DESTINATION_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// EXTRACT_DESTINATION_SIZE_MAXIMUM holds widest pseudorandom key.
const EXTRACT_DESTINATION_SIZE_MAXIMUM = hmac.DIGEST_SIZE_MAXIMUM

// COUNT_EMPTY reports refused expansion before any write.
const COUNT_EMPTY Count = OUTPUT_SIZE_MINIMUM

// COUNT_MAXIMUM is full RFC capacity for widest supported digest.
const COUNT_MAXIMUM Count = Count(OUTPUT_SIZE_MAXIMUM)

// EXTRACT_COUNT_MINIMUM is shortest supported pseudorandom key.
const EXTRACT_COUNT_MINIMUM Extract_Count = hmac.DIGEST_SIZE_MINIMUM

// EXTRACT_COUNT_MAXIMUM is longest supported pseudorandom key.
const EXTRACT_COUNT_MAXIMUM Extract_Count = hmac.DIGEST_SIZE_MAXIMUM

// EXTRACT_STATUS_OK means complete pseudorandom key reached caller storage.
const EXTRACT_STATUS_OK Extract_Status = Extract_Status(bits.WORD_8_MINIMUM)

// EXTRACT_STATUS_TOO_SMALL leaves short caller storage untouched.
const EXTRACT_STATUS_TOO_SMALL Extract_Status = EXTRACT_STATUS_OK + binary.UINT_8_SIZE

// EXPAND_STATUS_OK means complete requested key reached caller storage.
const EXPAND_STATUS_OK Expand_Status = Expand_Status(bits.WORD_8_MINIMUM)

// EXPAND_STATUS_TOO_LARGE rejects more than 255 selected-hash blocks.
const EXPAND_STATUS_TOO_LARGE Expand_Status = EXPAND_STATUS_OK + binary.UINT_8_SIZE

// Secret is bounded input keying material.
type Secret []byte

// Secret_Invariants bounds extraction input without inspecting secret bytes.
func Secret_Invariants(value Secret, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), INPUT_SIZE_MINIMUM, INPUT_SIZE_MAXIMUM).
		Ensure()
}

// Salt is bounded independent extraction salt.
type Salt []byte

// Salt_Invariants bounds extraction work without inspecting salt bytes.
func Salt_Invariants(value Salt, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), INPUT_SIZE_MINIMUM, INPUT_SIZE_MAXIMUM).
		Ensure()
}

// Pseudorandom_Key is bounded expansion key material.
type Pseudorandom_Key []byte

// Pseudorandom_Key_Invariants bounds expansion key work without inspecting bytes.
func Pseudorandom_Key_Invariants(value Pseudorandom_Key, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), INPUT_SIZE_MINIMUM, INPUT_SIZE_MAXIMUM).
		Ensure()
}

// Info is bounded application context.
type Info []byte

// Info_Invariants bounds expansion work without inspecting context bytes.
func Info_Invariants(value Info, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), INPUT_SIZE_MINIMUM, INPUT_SIZE_MAXIMUM).
		Ensure()
}

// Destination is caller-owned expanded-key storage.
type Destination []byte

// Destination_Invariants bounds expansion to full RFC counter capacity.
func Destination_Invariants(value Destination, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), OUTPUT_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM).
		Ensure()
}

// Extract_Destination is caller-owned pseudorandom-key storage.
type Extract_Destination []byte

// Extract_Destination_Invariants bounds storage to widest digest.
func Extract_Destination_Invariants(
	value Extract_Destination,
	namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), EXTRACT_DESTINATION_SIZE_MINIMUM,
			EXTRACT_DESTINATION_SIZE_MAXIMUM,
		).
		Ensure()
}

// Count is expanded bytes written to caller storage.
type Count int

// Count_Invariants spans empty refusal through widest RFC output.
func Count_Invariants(value Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), int(COUNT_EMPTY), int(COUNT_MAXIMUM)).
		Ensure()
}

// Extract_Count is selected pseudorandom-key width.
type Extract_Count uint8

// Extract_Count_Invariants spans supported digest widths.
func Extract_Count_Invariants(value Extract_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(EXTRACT_COUNT_MINIMUM), uint8(EXTRACT_COUNT_MAXIMUM),
		).
		Ensure()
}

// Extract_Status reports pseudorandom-key output capacity.
type Extract_Status uint8

// Extract_Status_Invariants covers complete and short extraction output.
func Extract_Status_Invariants(value Extract_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(EXTRACT_STATUS_OK), uint8(EXTRACT_STATUS_TOO_SMALL),
		).
		Ensure()
}

// Expand_Status reports selected-hash length-limit validation.
type Expand_Status uint8

// Expand_Status_Invariants covers complete and refused expansion.
func Expand_Status_Invariants(value Expand_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(EXPAND_STATUS_OK), uint8(EXPAND_STATUS_TOO_LARGE),
		).
		Ensure()
}

// Size is formula maximum for one selected hash.
type Size int

// Size_Invariants spans 255 blocks of shortest through longest digest.
func Size_Invariants(value Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), SELECTED_OUTPUT_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM,
		).
		Ensure()
}

// Extract_Into derives selected-width pseudorandom key into caller storage.
func Extract_Into(
	destination Extract_Destination,
	kind hmac.Kind,
	secret Secret,
	salt Salt,
) (count Extract_Count, status Extract_Status) {
	defer func() {
		Extract_Count_Invariants(count, "Extract_Into.count")
		Extract_Status_Invariants(status, "Extract_Into.status")
	}()
	Extract_Destination_Invariants(destination, "Extract_Into.destination")
	hmac.Kind_Invariants(kind, "Extract_Into.kind")
	Secret_Invariants(secret, "Extract_Into.secret")
	Salt_Invariants(salt, "Extract_Into.salt")
	require_kind(kind)
	require_extract_destination(destination)
	require_secret(secret)
	require_salt(salt)
	count = Pseudorandom_Key_Size(kind)
	if len(destination) < int(count) {
		return count, EXTRACT_STATUS_TOO_SMALL
	}
	var digest hmac.Digest
	hmac.Digest_Init(&digest, kind, hmac.Key(salt))
	hmac.Digest_Write(&digest, hmac.Source(secret))
	hmac.Digest_Sum_Into(&digest, hmac.Destination(destination[:count]))
	return count, EXTRACT_STATUS_OK
}

// Expand_Into fills caller storage with at most 255 selected-hash blocks.
func Expand_Into(
	destination Destination,
	kind hmac.Kind,
	pseudorandom_key Pseudorandom_Key,
	info Info,
) (count Count, status Expand_Status) {
	defer func() {
		Count_Invariants(count, "Expand_Into.count")
		Expand_Status_Invariants(status, "Expand_Into.status")
	}()
	Destination_Invariants(destination, "Expand_Into.destination")
	hmac.Kind_Invariants(kind, "Expand_Into.kind")
	Pseudorandom_Key_Invariants(pseudorandom_key, "Expand_Into.pseudorandom_key")
	Info_Invariants(info, "Expand_Into.info")
	require_kind(kind)
	require_destination(destination)
	require_pseudorandom_key(pseudorandom_key)
	require_info(info)
	if len(destination) > int(Output_Size_Maximum(kind)) {
		return COUNT_EMPTY, EXPAND_STATUS_TOO_LARGE
	}
	expand(destination, kind, pseudorandom_key, info)
	return Count(len(destination)), EXPAND_STATUS_OK
}

// Key_Into performs extraction and expansion without exposing intermediate key material.
func Key_Into(
	destination Destination,
	kind hmac.Kind,
	secret Secret,
	salt Salt,
	info Info,
) (count Count, status Expand_Status) {
	defer func() {
		Count_Invariants(count, "Key_Into.count")
		Expand_Status_Invariants(status, "Key_Into.status")
	}()
	Destination_Invariants(destination, "Key_Into.destination")
	hmac.Kind_Invariants(kind, "Key_Into.kind")
	Secret_Invariants(secret, "Key_Into.secret")
	Salt_Invariants(salt, "Key_Into.salt")
	Info_Invariants(info, "Key_Into.info")
	require_kind(kind)
	require_destination(destination)
	require_secret(secret)
	require_salt(salt)
	require_info(info)
	if len(destination) > int(Output_Size_Maximum(kind)) {
		return COUNT_EMPTY, EXPAND_STATUS_TOO_LARGE
	}
	var pseudorandom_key [hmac.DIGEST_SIZE_MAXIMUM]byte
	key_count, _ := Extract_Into(pseudorandom_key[:], kind, secret, salt)
	expand(destination, kind, pseudorandom_key[:key_count], info)
	return Count(len(destination)), EXPAND_STATUS_OK
}

// Pseudorandom_Key_Size reports selected extraction width.
func Pseudorandom_Key_Size(kind hmac.Kind) (size Extract_Count) {
	defer func() { Extract_Count_Invariants(size, "Pseudorandom_Key_Size.size") }()
	hmac.Kind_Invariants(kind, "Pseudorandom_Key_Size.kind")
	require_kind(kind)
	switch kind {
	case hmac.KIND_MD5:
		return md5.DIGEST_SIZE
	case hmac.KIND_SHA_1:
		return sha1.DIGEST_SIZE
	case hmac.KIND_SHA_224, hmac.KIND_SHA_512_224:
		return sha256.DIGEST_224_SIZE
	case hmac.KIND_SHA_256, hmac.KIND_SHA_512_256:
		return sha256.DIGEST_256_SIZE
	case hmac.KIND_SHA_384:
		return sha512.DIGEST_384_SIZE
	default:
		return sha512.DIGEST_512_SIZE
	}
}

// Output_Size_Maximum reports 255 selected-hash blocks.
func Output_Size_Maximum(kind hmac.Kind) (size Size) {
	defer func() { Size_Invariants(size, "Output_Size_Maximum.size") }()
	hmac.Kind_Invariants(kind, "Output_Size_Maximum.kind")
	require_kind(kind)
	return Size(int(Pseudorandom_Key_Size(kind)) * BLOCK_COUNT_MAXIMUM)
}

func expand(
	destination Destination,
	kind hmac.Kind,
	pseudorandom_key Pseudorandom_Key,
	info Info,
) {
	Destination_Invariants(destination, "expand.destination")
	hmac.Kind_Invariants(kind, "expand.kind")
	Pseudorandom_Key_Invariants(pseudorandom_key, "expand.pseudorandom_key")
	Info_Invariants(info, "expand.info")
	digest_size := int(Pseudorandom_Key_Size(kind))
	var previous hmac.Value
	previous_count := OUTPUT_SIZE_MINIMUM
	written := OUTPUT_SIZE_MINIMUM
	counter := byte(COUNTER_SIZE)
	for written < len(destination) {
		var digest hmac.Digest
		hmac.Digest_Init(&digest, kind, hmac.Key(pseudorandom_key))
		if previous_count != OUTPUT_SIZE_MINIMUM {
			hmac.Digest_Write(&digest, previous[:previous_count])
		}
		hmac.Digest_Write(&digest, hmac.Source(info))
		counter_source := [COUNTER_SIZE]byte{counter}
		hmac.Digest_Write(&digest, counter_source[:])
		previous, _ = hmac.Digest_Sum(&digest)
		previous_count = digest_size
		copy_count := min(digest_size, len(destination)-written)
		copy(destination[written:written+copy_count], previous[:copy_count])
		written += copy_count
		counter++
	}
}

func require_kind(kind hmac.Kind) {
	hmac.Kind_Invariants(kind, "require_kind.kind")
	if kind > hmac.KIND_SHA_512 {
		panic("hkdf: kind is invalid")
	}
}

func require_extract_destination(destination Extract_Destination) {
	Extract_Destination_Invariants(destination, "require_extract_destination.destination")
	if len(destination) > EXTRACT_DESTINATION_SIZE_MAXIMUM {
		panic("hkdf: extract destination exceeds bound")
	}
}

func require_destination(destination Destination) {
	Destination_Invariants(destination, "require_destination.destination")
	if len(destination) > OUTPUT_SIZE_MAXIMUM {
		panic("hkdf: destination exceeds bound")
	}
}

func require_secret(secret Secret) {
	Secret_Invariants(secret, "require_secret.secret")
	if len(secret) > INPUT_SIZE_MAXIMUM {
		panic("hkdf: secret exceeds bound")
	}
}

func require_salt(salt Salt) {
	Salt_Invariants(salt, "require_salt.salt")
	if len(salt) > INPUT_SIZE_MAXIMUM {
		panic("hkdf: salt exceeds bound")
	}
}

func require_pseudorandom_key(pseudorandom_key Pseudorandom_Key) {
	Pseudorandom_Key_Invariants(pseudorandom_key, "require_pseudorandom_key.key")
	if len(pseudorandom_key) > INPUT_SIZE_MAXIMUM {
		panic("hkdf: pseudorandom key exceeds bound")
	}
}

func require_info(info Info) {
	Info_Invariants(info, "require_info.info")
	if len(info) > INPUT_SIZE_MAXIMUM {
		panic("hkdf: info exceeds bound")
	}
}
