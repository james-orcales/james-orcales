// Package pbkdf2 derives bounded caller-owned keys through RFC 8018 PBKDF2.
package pbkdf2

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/crypto/hmac"
	"local/james-orcales/shared/crypto/subtle"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
)

// INPUT_SIZE_MINIMUM admits empty password and salt values defined by PBKDF2.
const INPUT_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// INPUT_SIZE_MAXIMUM follows repository byte-slice bound.
const INPUT_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// OUTPUT_SIZE_MINIMUM admits callers that require no derived bytes.
const OUTPUT_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// OUTPUT_SIZE_MAXIMUM follows repository byte-slice bound.
const OUTPUT_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// BLOCK_COUNT_MAXIMUM derives from largest output and shortest supported digest.
const BLOCK_COUNT_MAXIMUM = (OUTPUT_SIZE_MAXIMUM + hmac.DIGEST_SIZE_MINIMUM -
	binary.UINT_8_SIZE) /
	hmac.DIGEST_SIZE_MINIMUM

// COUNTER_SIZE is RFC 8018's unsigned 32-bit block index width.
const COUNTER_SIZE = binary.UINT_32_SIZE

// BLOCK_INDEX_MINIMUM starts RFC 8018 output block numbering at one.
const BLOCK_INDEX_MINIMUM = binary.UINT_8_SIZE

// PRF_EVALUATION_COUNT_MAXIMUM caps synchronous attacker-controlled CPU work at one mebi calls.
const PRF_EVALUATION_COUNT_MAXIMUM = bits.MEBIBYTE_BYTES

// ITERATION_COUNT_MINIMUM requires the first pseudorandom-function evaluation.
const ITERATION_COUNT_MINIMUM Iteration_Count = Iteration_Count(BLOCK_INDEX_MINIMUM)

// ITERATION_COUNT_MAXIMUM spends the complete work limit on one output block.
const ITERATION_COUNT_MAXIMUM Iteration_Count = Iteration_Count(
	PRF_EVALUATION_COUNT_MAXIMUM,
)

// COUNT_EMPTY reports no caller bytes changed.
const COUNT_EMPTY Count = Count(OUTPUT_SIZE_MINIMUM)

// COUNT_MAXIMUM reports a complete largest caller output.
const COUNT_MAXIMUM Count = Count(OUTPUT_SIZE_MAXIMUM)

// STATUS_OK means all requested output reached caller storage.
const STATUS_OK Status = Status(bits.WORD_8_MINIMUM)

// STATUS_WORK_TOO_LARGE refuses excessive CPU work before output changes.
const STATUS_WORK_TOO_LARGE Status = STATUS_OK + binary.UINT_8_SIZE

// Password is bounded PBKDF2 secret input.
type Password []byte

// Password_Invariants bounds key setup without inspecting secret bytes.
func Password_Invariants(value Password, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), INPUT_SIZE_MINIMUM, INPUT_SIZE_MAXIMUM).
		Ensure()
}

// Salt is bounded PBKDF2 independent input.
type Salt []byte

// Salt_Invariants bounds first-block work without inspecting salt bytes.
func Salt_Invariants(value Salt, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), INPUT_SIZE_MINIMUM, INPUT_SIZE_MAXIMUM).
		Ensure()
}

// Destination is bounded caller-owned derived-key storage.
type Destination []byte

// Destination_Invariants binds output and block count to repository limits.
func Destination_Invariants(value Destination, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), OUTPUT_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM).
		Ensure()
}

// Iteration_Count is the number of pseudorandom-function evaluations per output block.
type Iteration_Count uint32

// Iteration_Count_Invariants keeps one-block synchronous work inside its cap.
func Iteration_Count_Invariants(value Iteration_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(
			uint32(value), uint32(ITERATION_COUNT_MINIMUM),
			uint32(ITERATION_COUNT_MAXIMUM),
		).
		Ensure()
}

// Count is derived bytes written to caller storage.
type Count int

// Count_Invariants spans empty through largest bounded output.
func Count_Invariants(value Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), int(COUNT_EMPTY), int(COUNT_MAXIMUM)).
		Ensure()
}

// Status reports whether total requested work fits the synchronous bound.
type Status uint8

// Status_Invariants covers completed and refused derivation.
func Status_Invariants(value Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_WORK_TOO_LARGE),
		).
		Ensure()
}

// Key_Into derives output only after proving output-block work fits one bounded call.
func Key_Into(
	destination Destination,
	kind hmac.Kind,
	password Password,
	salt Salt,
	iterations Iteration_Count,
) (count Count, status Status) {
	defer func() {
		Count_Invariants(count, "Key_Into.count")
		Status_Invariants(status, "Key_Into.status")
	}()
	Destination_Invariants(destination, "Key_Into.destination")
	hmac.Kind_Invariants(kind, "Key_Into.kind")
	Password_Invariants(password, "Key_Into.password")
	Salt_Invariants(salt, "Key_Into.salt")
	Iteration_Count_Invariants(iterations, "Key_Into.iterations")
	require_destination(destination)
	require_password(password)
	require_salt(salt)
	require_iterations(iterations)

	var digest hmac.Digest
	hmac.Digest_Init(&digest, kind, hmac.Key(password))
	digest_size := int(hmac.Digest_Size(&digest))
	block_count := (len(destination) + digest_size - binary.UINT_8_SIZE) / digest_size
	if block_count > PRF_EVALUATION_COUNT_MAXIMUM/int(iterations) {
		return COUNT_EMPTY, STATUS_WORK_TOO_LARGE
	}
	derive(destination, &digest, salt, iterations)
	return Count(len(destination)), STATUS_OK
}

// PBKDF2 chains U values from the same keyed HMAC baseline; resetting avoids rebuilding key pads.
func derive(
	destination Destination,
	digest *hmac.Digest,
	salt Salt,
	iterations Iteration_Count,
) {
	Destination_Invariants(destination, "derive.destination")
	hmac.Digest_Invariants(*digest, "derive.digest")
	Salt_Invariants(salt, "derive.salt")
	Iteration_Count_Invariants(iterations, "derive.iterations")
	digest_size := int(hmac.Digest_Size(digest))
	written := OUTPUT_SIZE_MINIMUM
	block_index := BLOCK_INDEX_MINIMUM
	for written < len(destination) {
		var counter [COUNTER_SIZE]byte
		binary.Put_Uint_32(
			binary.Bytes(counter[:]), binary.Word_32(block_index), binary.BIG_ENDIAN,
		)
		hmac.Digest_Reset(digest)
		hmac.Digest_Write(digest, hmac.Source(salt))
		hmac.Digest_Write(digest, hmac.Source(counter[:]))
		value, _ := hmac.Digest_Sum(digest)
		combined := value
		iteration_index := ITERATION_COUNT_MINIMUM
		for iteration_index < iterations {
			hmac.Digest_Reset(digest)
			hmac.Digest_Write(digest, hmac.Source(value[:digest_size]))
			value, _ = hmac.Digest_Sum(digest)
			subtle.XOR_Bytes(
				combined[:digest_size], combined[:digest_size], value[:digest_size],
			)
			iteration_index++
		}
		copy_count := min(digest_size, len(destination)-written)
		copy(destination[written:written+copy_count], combined[:copy_count])
		written += copy_count
		block_index++
	}
}

func require_destination(destination Destination) {
	Destination_Invariants(destination, "require_destination.destination")
	if len(destination) > OUTPUT_SIZE_MAXIMUM {
		panic("pbkdf2: destination exceeds bound")
	}
}

func require_password(password Password) {
	Password_Invariants(password, "require_password.password")
	if len(password) > INPUT_SIZE_MAXIMUM {
		panic("pbkdf2: password exceeds bound")
	}
}

func require_salt(salt Salt) {
	Salt_Invariants(salt, "require_salt.salt")
	if len(salt) > INPUT_SIZE_MAXIMUM {
		panic("pbkdf2: salt exceeds bound")
	}
}

func require_iterations(iterations Iteration_Count) {
	Iteration_Count_Invariants(iterations, "require_iterations.iterations")
	if iterations < ITERATION_COUNT_MINIMUM {
		panic("pbkdf2: iteration count exceeds bound")
	}
	if iterations > ITERATION_COUNT_MAXIMUM {
		panic("pbkdf2: iteration count exceeds bound")
	}
}
