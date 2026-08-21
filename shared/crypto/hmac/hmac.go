// Package hmac computes keyed message authentication codes with bounded caller-owned state.
package hmac

import (
	"unsafe"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/crypto/md5"
	"local/james-orcales/shared/crypto/sha1"
	"local/james-orcales/shared/crypto/sha256"
	"local/james-orcales/shared/crypto/sha512"
	"local/james-orcales/shared/crypto/subtle"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/simulation/aver/default"
)

// KIND_MD5 selects RFC 1321 compression.
const KIND_MD5 Kind = Kind(bits.WORD_8_MINIMUM)

// KIND_SHA_1 selects FIPS 180-4 SHA-1 compression.
const KIND_SHA_1 Kind = KIND_MD5 + binary.UINT_8_SIZE

// KIND_SHA_224 selects FIPS 180-4 SHA-224 compression.
const KIND_SHA_224 Kind = KIND_SHA_1 + binary.UINT_8_SIZE

// KIND_SHA_256 selects FIPS 180-4 SHA-256 compression.
const KIND_SHA_256 Kind = KIND_SHA_224 + binary.UINT_8_SIZE

// KIND_SHA_384 selects FIPS 180-4 SHA-384 compression.
const KIND_SHA_384 Kind = KIND_SHA_256 + binary.UINT_8_SIZE

// KIND_SHA_512_224 selects FIPS 180-4 SHA-512/224 compression.
const KIND_SHA_512_224 Kind = KIND_SHA_384 + binary.UINT_8_SIZE

// KIND_SHA_512_256 selects FIPS 180-4 SHA-512/256 compression.
const KIND_SHA_512_256 Kind = KIND_SHA_512_224 + binary.UINT_8_SIZE

// KIND_SHA_512 selects FIPS 180-4 SHA-512 compression.
const KIND_SHA_512 Kind = KIND_SHA_512_256 + binary.UINT_8_SIZE

// KIND_COUNT follows contiguous supported kind values.
const KIND_COUNT = int(KIND_SHA_512-KIND_MD5) + binary.UINT_8_SIZE

// DIGEST_SIZE_MINIMUM is shortest supported tag.
const DIGEST_SIZE_MINIMUM = md5.DIGEST_SIZE

// DIGEST_SIZE_MAXIMUM is longest supported tag.
const DIGEST_SIZE_MAXIMUM = sha512.DIGEST_512_SIZE

// BLOCK_SIZE_MINIMUM is narrowest supported compression block.
const BLOCK_SIZE_MINIMUM = md5.BLOCK_SIZE

// BLOCK_SIZE_MAXIMUM is widest supported compression block.
const BLOCK_SIZE_MAXIMUM = sha512.BLOCK_SIZE

// KEY_SIZE_MINIMUM admits empty protocol keys.
const KEY_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// KEY_SIZE_MAXIMUM follows repository byte-slice bound.
const KEY_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// REDUCED_KEY_SIZE_MINIMUM exceeds narrowest supported compression block.
const REDUCED_KEY_SIZE_MINIMUM = BLOCK_SIZE_MINIMUM + binary.UINT_8_SIZE

// SOURCE_SIZE_MINIMUM admits empty message chunks.
const SOURCE_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// SOURCE_SIZE_MAXIMUM follows repository byte-slice bound.
const SOURCE_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// DESTINATION_SIZE_MINIMUM admits short-output status paths.
const DESTINATION_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// DESTINATION_SIZE_MAXIMUM follows repository byte-slice bound.
const DESTINATION_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// TAG_SIZE_MINIMUM admits empty comparison input.
const TAG_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// TAG_SIZE_MAXIMUM follows repository byte-slice bound.
const TAG_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// COUNT_MINIMUM is empty input.
const COUNT_MINIMUM = SOURCE_SIZE_MINIMUM

// COUNT_MAXIMUM is one complete bounded source.
const COUNT_MAXIMUM = SOURCE_SIZE_MAXIMUM

// OUTPUT_COUNT_MINIMUM is shortest complete supported tag.
const OUTPUT_COUNT_MINIMUM Output_Count = DIGEST_SIZE_MINIMUM

// OUTPUT_COUNT_MAXIMUM is longest complete supported tag.
const OUTPUT_COUNT_MAXIMUM Output_Count = DIGEST_SIZE_MAXIMUM

// OUTPUT_STATUS_OK means complete tag reached caller storage.
const OUTPUT_STATUS_OK Output_Status = Output_Status(bits.WORD_8_MINIMUM)

// OUTPUT_STATUS_TOO_SMALL leaves short caller storage untouched.
const OUTPUT_STATUS_TOO_SMALL Output_Status = OUTPUT_STATUS_OK + binary.UINT_8_SIZE

// READY_INDEX stores caller-state identity.
const READY_INDEX = bits.BIT_COUNT_MINIMUM

// READY_WORD_COUNT holds one caller-state identity octet.
const READY_WORD_COUNT = READY_INDEX + binary.UINT_8_SIZE

// READY_EMPTY_MARKER is zero caller storage.
const READY_EMPTY_MARKER byte = bits.WORD_8_MINIMUM

// READY_COMPLETE_MARKER distinguishes keyed state from zero caller storage.
const READY_COMPLETE_MARKER byte = bits.WORD_8_MAXIMUM

// INNER_PAD_BYTE is RFC 2104 inner pad octet.
const INNER_PAD_BYTE byte = 0x36

// OUTER_PAD_BYTE is RFC 2104 outer pad octet.
const OUTER_PAD_BYTE byte = 0x5c

// HASH_STATE_WORD_COUNT rounds widest digest state into aligned 64-bit caller storage.
const HASH_STATE_WORD_COUNT = (int(unsafe.Sizeof(sha512.Digest{})) +
	binary.UINT_64_SIZE - binary.UINT_8_SIZE) / binary.UINT_64_SIZE

// Kind selects one supported compression function.
type Kind uint8

// Kind_Invariants covers contiguous supported algorithms.
func Kind_Invariants(value Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(KIND_MD5), uint8(KIND_SHA_512)).
		Ensure()
}

// Key is one bounded secret key.
type Key []byte

// Key_Invariants bounds initialization work without inspecting key bytes.
func Key_Invariants(value Key, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), KEY_SIZE_MINIMUM, KEY_SIZE_MAXIMUM).
		Ensure()
}

// Reduced_Key is large enough to require selected-hash reduction.
type Reduced_Key []byte

// Reduced_Key_Invariants narrows hash_key to reachable key lengths.
func Reduced_Key_Invariants(value Reduced_Key, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), REDUCED_KEY_SIZE_MINIMUM, KEY_SIZE_MAXIMUM).
		Ensure()
}

// Source is one bounded message chunk.
type Source []byte

// Source_Invariants bounds write work without inspecting message bytes.
func Source_Invariants(value Source, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SOURCE_SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Destination is bounded caller-owned output storage.
type Destination []byte

// Destination_Invariants binds output storage to repository byte limits.
func Destination_Invariants(value Destination, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), DESTINATION_SIZE_MINIMUM, DESTINATION_SIZE_MAXIMUM).
		Ensure()
}

// Tag is one bounded authentication value supplied for comparison.
type Tag []byte

// Tag_Invariants bounds equality work without inspecting tag bytes.
func Tag_Invariants(value Tag, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TAG_SIZE_MINIMUM, TAG_SIZE_MAXIMUM).
		Ensure()
}

// Count is bytes accepted by one bounded write.
type Count int

// Count_Invariants covers complete source consumption.
func Count_Invariants(value Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Output_Count is selected complete tag width.
type Output_Count uint8

// Output_Count_Invariants spans supported tag widths.
func Output_Count_Invariants(value Output_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(OUTPUT_COUNT_MINIMUM), uint8(OUTPUT_COUNT_MAXIMUM),
		).
		Ensure()
}

// Output_Status reports caller output capacity.
type Output_Status uint8

// Output_Status_Invariants covers complete and short output.
func Output_Status_Invariants(value Output_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(OUTPUT_STATUS_OK), uint8(OUTPUT_STATUS_TOO_SMALL)).
		Ensure()
}

// Size is selected tag width.
type Size uint8

// Size_Invariants spans supported tag widths.
func Size_Invariants(value Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), DIGEST_SIZE_MINIMUM, DIGEST_SIZE_MAXIMUM).
		Ensure()
}

// Block_Size is selected compression-block width.
type Block_Size uint8

// Block_Size_Invariants spans supported block widths.
func Block_Size_Invariants(value Block_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), BLOCK_SIZE_MINIMUM, BLOCK_SIZE_MAXIMUM).
		Ensure()
}

// Ready stores caller-state identity without adding an enum branch to every keyed operation.
type Ready [READY_WORD_COUNT]byte

// Ready_Invariants fixes caller-state identity storage width.
func Ready_Invariants(value Ready, _ aver.Namespace) {
	aver.Always(len(value) == READY_WORD_COUNT, "HMAC identity has fixed width.")
}

// Equality exposes constant-time comparison result.
type Equality bool

// Equality_Invariants covers equal and unequal tags.
func Equality_Invariants(value Equality, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "HMAC tags are equal.").
		Ensure()
}

// Value holds longest tag so one fixed result covers every kind.
type Value [DIGEST_SIZE_MAXIMUM]byte

// Value_Invariants fixes caller-independent return width.
func Value_Invariants(value Value, _ aver.Namespace) {
	aver.Always(len(value) == DIGEST_SIZE_MAXIMUM, "HMAC value has fixed width.")
}

// Pad holds widest RFC 2104 compression pad.
type Pad [BLOCK_SIZE_MAXIMUM]byte

// Pad_Invariants fixes stack-owned pad capacity.
func Pad_Invariants(value Pad, _ aver.Namespace) {
	aver.Always(len(value) == BLOCK_SIZE_MAXIMUM, "HMAC pad has fixed width.")
}

// Initial_State stores selected keyed inner baseline in aligned caller storage.
type Initial_State [HASH_STATE_WORD_COUNT]uint64

// Initial_State_Invariants proves widest digest fits without heap storage.
func Initial_State_Invariants(value Initial_State, _ aver.Namespace) {
	aver.Always(
		len(value) == HASH_STATE_WORD_COUNT,
		"HMAC initial state has fixed width.",
	)
}

// Inner_State stores selected live inner digest in aligned caller storage.
type Inner_State [HASH_STATE_WORD_COUNT]uint64

// Inner_State_Invariants proves widest digest fits without heap storage.
func Inner_State_Invariants(value Inner_State, _ aver.Namespace) {
	aver.Always(len(value) == HASH_STATE_WORD_COUNT, "HMAC inner state has fixed width.")
}

// Outer_State stores selected keyed outer digest in aligned caller storage.
type Outer_State [HASH_STATE_WORD_COUNT]uint64

// Outer_State_Invariants proves widest digest fits without heap storage.
func Outer_State_Invariants(value Outer_State, _ aver.Namespace) {
	aver.Always(len(value) == HASH_STATE_WORD_COUNT, "HMAC outer state has fixed width.")
}

// Digest keeps selected inner, initial, and outer states in caller storage.
type Digest struct {
	// Kind preserves selected hash across reset and sum.
	Kind Kind
	// Ready rejects zero caller storage before keyed operations.
	Ready Ready
	// Initial restores keyed inner state.
	Initial Initial_State
	// Inner accepts message chunks.
	Inner Inner_State
	// Outer completes authentication.
	Outer Outer_State
}

// Digest_Invariants composes selector, identity, and aligned fixed-capacity hash states.
func Digest_Invariants(value Digest, namespace aver.Namespace) {
	Kind_Invariants(value.Kind, namespace)
	Ready_Invariants(value.Ready, namespace)
	Initial_State_Invariants(value.Initial, namespace)
	Inner_State_Invariants(value.Inner, namespace)
	Outer_State_Invariants(value.Outer, namespace)
}

// Digest_Init clears prior key material before constructing selected pads.
func Digest_Init(digest *Digest, kind Kind, key Key) {
	Digest_Invariants(*digest, "Digest_Init.digest.input")
	Kind_Invariants(kind, "Digest_Init.kind")
	Key_Invariants(key, "Digest_Init.key")
	if kind > KIND_SHA_512 {
		panic("hmac: kind is invalid")
	}
	if len(key) > KEY_SIZE_MAXIMUM {
		panic("hmac: key exceeds bound")
	}
	*digest = Digest{Kind: kind}
	inner := inner_pad(kind, key)
	outer := outer_pad(kind, inner)
	digest_state_init(digest, kind, &inner, &outer)
	digest.Ready[READY_INDEX] = READY_COMPLETE_MARKER
	Digest_Invariants(*digest, "Digest_Init.digest.output")
}

// Digest_Reset restores keyed inner state without retaining message state.
func Digest_Reset(digest *Digest) {
	Digest_Invariants(*digest, "Digest_Reset.digest.input")
	digest_require(digest)
	digest.Inner = Inner_State(digest.Initial)
	Digest_Invariants(*digest, "Digest_Reset.digest.output")
}

// Digest_Write consumes one complete bounded source chunk.
func Digest_Write(digest *Digest, source Source) (count Count) {
	defer func() { Count_Invariants(count, "Digest_Write.count") }()
	Digest_Invariants(*digest, "Digest_Write.digest.input")
	Source_Invariants(source, "Digest_Write.source")
	digest_require(digest)
	if len(source) > SOURCE_SIZE_MAXIMUM {
		panic("hmac: source exceeds bound")
	}
	digest_write(digest, source)
	Digest_Invariants(*digest, "Digest_Write.digest.output")
	return Count(len(source))
}

// Digest_Sum returns fixed storage and selected live prefix width.
func Digest_Sum(digest *Digest) (value Value, count Output_Count) {
	defer func() {
		Value_Invariants(value, "Digest_Sum.value")
		Output_Count_Invariants(count, "Digest_Sum.count")
	}()
	Digest_Invariants(*digest, "Digest_Sum.digest")
	digest_require(digest)
	count = Output_Count(kind_size(digest.Kind))
	switch digest.Kind {
	case KIND_MD5:
		return digest_sum_md5(digest.Inner, digest.Outer), count
	case KIND_SHA_1:
		return digest_sum_sha_1(digest.Inner, digest.Outer), count
	case KIND_SHA_224, KIND_SHA_256:
		return digest_sum_sha_256(digest.Inner, digest.Outer), count
	default:
		return digest_sum_sha_512(digest.Inner, digest.Outer), count
	}
}

// Digest_Sum_Into leaves short destination untouched.
func Digest_Sum_Into(
	digest *Digest,
	destination Destination,
) (count Output_Count, status Output_Status) {
	defer func() {
		Output_Count_Invariants(count, "Digest_Sum_Into.count")
		Output_Status_Invariants(status, "Digest_Sum_Into.status")
	}()
	Digest_Invariants(*digest, "Digest_Sum_Into.digest")
	Destination_Invariants(destination, "Digest_Sum_Into.destination")
	digest_require(digest)
	if len(destination) > DESTINATION_SIZE_MAXIMUM {
		panic("hmac: destination exceeds bound")
	}
	count = Output_Count(Digest_Size(digest))
	if len(destination) < int(count) {
		return count, OUTPUT_STATUS_TOO_SMALL
	}
	value, _ := Digest_Sum(digest)
	copy(destination[:count], value[:count])
	return count, OUTPUT_STATUS_OK
}

// Digest_Clone_Into copies live keyed state without aliasing caller storage.
func Digest_Clone_Into(destination *Digest, source *Digest) {
	Digest_Invariants(*destination, "Digest_Clone_Into.destination.input")
	Digest_Invariants(*source, "Digest_Clone_Into.source")
	digest_require(source)
	*destination = *source
	Digest_Invariants(*destination, "Digest_Clone_Into.destination.output")
}

// Digest_Size reports selected tag width.
func Digest_Size(digest *Digest) (size Size) {
	defer func() { Size_Invariants(size, "Digest_Size.size") }()
	Digest_Invariants(*digest, "Digest_Size.digest")
	digest_require(digest)
	return kind_size(digest.Kind)
}

// Digest_Block_Size reports selected compression-block width.
func Digest_Block_Size(digest *Digest) (size Block_Size) {
	defer func() { Block_Size_Invariants(size, "Digest_Block_Size.size") }()
	Digest_Invariants(*digest, "Digest_Block_Size.digest")
	digest_require(digest)
	return kind_block_size(digest.Kind)
}

// Equal compares bounded tags without content-dependent exit.
func Equal(left Tag, right Tag) (equal Equality) {
	defer func() { Equality_Invariants(equal, "Equal.equal") }()
	Tag_Invariants(left, "Equal.left")
	Tag_Invariants(right, "Equal.right")
	if len(left) > TAG_SIZE_MAXIMUM {
		panic("hmac: tag exceeds bound")
	}
	if len(right) > TAG_SIZE_MAXIMUM {
		panic("hmac: tag exceeds bound")
	}
	decision := subtle.Constant_Time_Compare(subtle.Source(left), subtle.Source(right))
	return Equality(decision == subtle.DECISION_TRUE)
}

func inner_pad(kind Kind, key Key) (pad Pad) {
	defer func() { Pad_Invariants(pad, "inner_pad.pad") }()
	Kind_Invariants(kind, "inner_pad.kind")
	Key_Invariants(key, "inner_pad.key")
	block_size := int(kind_block_size(kind))
	if len(key) > block_size {
		hash_key(kind, Reduced_Key(key), &pad)
	} else {
		copy(pad[:block_size], key)
	}
	for index := range block_size {
		pad[index] ^= INNER_PAD_BYTE
	}
	return pad
}

func outer_pad(kind Kind, inner Pad) (outer Pad) {
	defer func() { Pad_Invariants(outer, "outer_pad.outer") }()
	Kind_Invariants(kind, "outer_pad.kind")
	Pad_Invariants(inner, "outer_pad.inner")
	outer = inner
	block_size := int(kind_block_size(kind))
	for index := range block_size {
		outer[index] ^= INNER_PAD_BYTE ^ OUTER_PAD_BYTE
	}
	return outer
}

func hash_key(kind Kind, key Reduced_Key, destination *Pad) {
	Kind_Invariants(kind, "hash_key.kind")
	Reduced_Key_Invariants(key, "hash_key.key")
	Pad_Invariants(*destination, "hash_key.destination.input")
	switch kind {
	case KIND_MD5:
		value := md5.Checksum(md5.Source(key))
		copy(destination[:], value[:])
	case KIND_SHA_1:
		value := sha1.Checksum(sha1.Source(key))
		copy(destination[:], value[:])
	case KIND_SHA_224:
		value := sha256.Checksum_224(sha256.Source(key))
		copy(destination[:], value[:])
	case KIND_SHA_256:
		value := sha256.Checksum_256(sha256.Source(key))
		copy(destination[:], value[:])
	case KIND_SHA_384:
		value := sha512.Checksum_384(sha512.Source(key))
		copy(destination[:], value[:])
	case KIND_SHA_512_224:
		value := sha512.Checksum_512_224(sha512.Source(key))
		copy(destination[:], value[:])
	case KIND_SHA_512_256:
		value := sha512.Checksum_512_256(sha512.Source(key))
		copy(destination[:], value[:])
	default:
		value := sha512.Checksum_512(sha512.Source(key))
		copy(destination[:], value[:])
	}
	Pad_Invariants(*destination, "hash_key.destination.output")
}

func digest_state_init(digest *Digest, kind Kind, inner *Pad, outer *Pad) {
	Digest_Invariants(*digest, "digest_state_init.digest.input")
	Kind_Invariants(kind, "digest_state_init.kind")
	Pad_Invariants(*inner, "digest_state_init.inner")
	Pad_Invariants(*outer, "digest_state_init.outer")
	switch kind {
	case KIND_MD5:
		digest_state_init_md5(&digest.Initial, &digest.Inner, &digest.Outer, inner, outer)
	case KIND_SHA_1:
		digest_state_init_sha_1(&digest.Initial, &digest.Inner, &digest.Outer, inner, outer)
	case KIND_SHA_224:
		digest_state_init_sha_256(
			&digest.Initial, &digest.Inner, &digest.Outer,
			inner, outer, sha256.KIND_SHA_224,
		)
	case KIND_SHA_256:
		digest_state_init_sha_256(
			&digest.Initial, &digest.Inner, &digest.Outer,
			inner, outer, sha256.KIND_SHA_256,
		)
	case KIND_SHA_384:
		digest_state_init_sha_512(
			&digest.Initial, &digest.Inner, &digest.Outer,
			inner, outer, sha512.KIND_SHA_384,
		)
	case KIND_SHA_512_224:
		digest_state_init_sha_512(
			&digest.Initial, &digest.Inner, &digest.Outer,
			inner, outer, sha512.KIND_SHA_512_224,
		)
	case KIND_SHA_512_256:
		digest_state_init_sha_512(
			&digest.Initial, &digest.Inner, &digest.Outer,
			inner, outer, sha512.KIND_SHA_512_256,
		)
	default:
		digest_state_init_sha_512(
			&digest.Initial, &digest.Inner, &digest.Outer,
			inner, outer, sha512.KIND_SHA_512,
		)
	}
	Digest_Invariants(*digest, "digest_state_init.digest.output")
}

func digest_state_init_md5(
	initial *Initial_State,
	live *Inner_State,
	keyed_outer *Outer_State,
	inner *Pad,
	outer *Pad,
) {
	Initial_State_Invariants(*initial, "digest_state_init_md5.initial.input")
	Inner_State_Invariants(*live, "digest_state_init_md5.live.input")
	Outer_State_Invariants(*keyed_outer, "digest_state_init_md5.keyed_outer.input")
	Pad_Invariants(*inner, "digest_state_init_md5.inner")
	Pad_Invariants(*outer, "digest_state_init_md5.outer")
	initial_digest := (*md5.Digest)(unsafe.Pointer(&initial[bits.BIT_COUNT_MINIMUM]))
	live_digest := (*md5.Digest)(unsafe.Pointer(&live[bits.BIT_COUNT_MINIMUM]))
	outer_digest := (*md5.Digest)(unsafe.Pointer(&keyed_outer[bits.BIT_COUNT_MINIMUM]))
	md5.Digest_Init(initial_digest)
	md5.Digest_Write(initial_digest, inner[:md5.BLOCK_SIZE])
	*live_digest = *initial_digest
	md5.Digest_Init(outer_digest)
	md5.Digest_Write(outer_digest, outer[:md5.BLOCK_SIZE])
	Initial_State_Invariants(*initial, "digest_state_init_md5.initial.output")
	Inner_State_Invariants(*live, "digest_state_init_md5.live.output")
	Outer_State_Invariants(*keyed_outer, "digest_state_init_md5.keyed_outer.output")
}

func digest_state_init_sha_1(
	initial *Initial_State,
	live *Inner_State,
	keyed_outer *Outer_State,
	inner *Pad,
	outer *Pad,
) {
	Initial_State_Invariants(*initial, "digest_state_init_sha_1.initial.input")
	Inner_State_Invariants(*live, "digest_state_init_sha_1.live.input")
	Outer_State_Invariants(*keyed_outer, "digest_state_init_sha_1.keyed_outer.input")
	Pad_Invariants(*inner, "digest_state_init_sha_1.inner")
	Pad_Invariants(*outer, "digest_state_init_sha_1.outer")
	initial_digest := (*sha1.Digest)(unsafe.Pointer(&initial[bits.BIT_COUNT_MINIMUM]))
	live_digest := (*sha1.Digest)(unsafe.Pointer(&live[bits.BIT_COUNT_MINIMUM]))
	outer_digest := (*sha1.Digest)(unsafe.Pointer(&keyed_outer[bits.BIT_COUNT_MINIMUM]))
	sha1.Digest_Init(initial_digest)
	sha1.Digest_Write(initial_digest, inner[:sha1.BLOCK_SIZE])
	*live_digest = *initial_digest
	sha1.Digest_Init(outer_digest)
	sha1.Digest_Write(outer_digest, outer[:sha1.BLOCK_SIZE])
	Initial_State_Invariants(*initial, "digest_state_init_sha_1.initial.output")
	Inner_State_Invariants(*live, "digest_state_init_sha_1.live.output")
	Outer_State_Invariants(*keyed_outer, "digest_state_init_sha_1.keyed_outer.output")
}

func digest_state_init_sha_256(
	initial *Initial_State,
	live *Inner_State,
	keyed_outer *Outer_State,
	inner *Pad,
	outer *Pad,
	kind sha256.Kind,
) {
	Initial_State_Invariants(*initial, "digest_state_init_sha_256.initial.input")
	Inner_State_Invariants(*live, "digest_state_init_sha_256.live.input")
	Outer_State_Invariants(*keyed_outer, "digest_state_init_sha_256.keyed_outer.input")
	Pad_Invariants(*inner, "digest_state_init_sha_256.inner")
	Pad_Invariants(*outer, "digest_state_init_sha_256.outer")
	sha256.Kind_Invariants(kind, "digest_state_init_sha_256.kind")
	initial_digest := (*sha256.Digest)(unsafe.Pointer(&initial[bits.BIT_COUNT_MINIMUM]))
	live_digest := (*sha256.Digest)(unsafe.Pointer(&live[bits.BIT_COUNT_MINIMUM]))
	outer_digest := (*sha256.Digest)(unsafe.Pointer(&keyed_outer[bits.BIT_COUNT_MINIMUM]))
	sha256.Digest_Init(initial_digest, kind)
	sha256.Digest_Write(initial_digest, inner[:sha256.BLOCK_SIZE])
	*live_digest = *initial_digest
	sha256.Digest_Init(outer_digest, kind)
	sha256.Digest_Write(outer_digest, outer[:sha256.BLOCK_SIZE])
	Initial_State_Invariants(*initial, "digest_state_init_sha_256.initial.output")
	Inner_State_Invariants(*live, "digest_state_init_sha_256.live.output")
	Outer_State_Invariants(*keyed_outer, "digest_state_init_sha_256.keyed_outer.output")
}

func digest_state_init_sha_512(
	initial *Initial_State,
	live *Inner_State,
	keyed_outer *Outer_State,
	inner *Pad,
	outer *Pad,
	kind sha512.Kind,
) {
	Initial_State_Invariants(*initial, "digest_state_init_sha_512.initial.input")
	Inner_State_Invariants(*live, "digest_state_init_sha_512.live.input")
	Outer_State_Invariants(*keyed_outer, "digest_state_init_sha_512.keyed_outer.input")
	Pad_Invariants(*inner, "digest_state_init_sha_512.inner")
	Pad_Invariants(*outer, "digest_state_init_sha_512.outer")
	sha512.Kind_Invariants(kind, "digest_state_init_sha_512.kind")
	initial_digest := (*sha512.Digest)(unsafe.Pointer(&initial[bits.BIT_COUNT_MINIMUM]))
	live_digest := (*sha512.Digest)(unsafe.Pointer(&live[bits.BIT_COUNT_MINIMUM]))
	outer_digest := (*sha512.Digest)(unsafe.Pointer(&keyed_outer[bits.BIT_COUNT_MINIMUM]))
	sha512.Digest_Init(initial_digest, kind)
	sha512.Digest_Write(initial_digest, inner[:sha512.BLOCK_SIZE])
	*live_digest = *initial_digest
	sha512.Digest_Init(outer_digest, kind)
	sha512.Digest_Write(outer_digest, outer[:sha512.BLOCK_SIZE])
	Initial_State_Invariants(*initial, "digest_state_init_sha_512.initial.output")
	Inner_State_Invariants(*live, "digest_state_init_sha_512.live.output")
	Outer_State_Invariants(*keyed_outer, "digest_state_init_sha_512.keyed_outer.output")
}

func digest_write(digest *Digest, source Source) {
	Digest_Invariants(*digest, "digest_write.digest.input")
	Source_Invariants(source, "digest_write.source")
	switch digest.Kind {
	case KIND_MD5:
		inner := (*md5.Digest)(unsafe.Pointer(&digest.Inner[bits.BIT_COUNT_MINIMUM]))
		md5.Digest_Write(inner, md5.Source(source))
	case KIND_SHA_1:
		inner := (*sha1.Digest)(unsafe.Pointer(&digest.Inner[bits.BIT_COUNT_MINIMUM]))
		sha1.Digest_Write(inner, sha1.Source(source))
	case KIND_SHA_224, KIND_SHA_256:
		inner := (*sha256.Digest)(unsafe.Pointer(&digest.Inner[bits.BIT_COUNT_MINIMUM]))
		sha256.Digest_Write(inner, sha256.Source(source))
	default:
		inner := (*sha512.Digest)(unsafe.Pointer(&digest.Inner[bits.BIT_COUNT_MINIMUM]))
		sha512.Digest_Write(inner, sha512.Source(source))
	}
	Digest_Invariants(*digest, "digest_write.digest.output")
}

func digest_sum_md5(inner Inner_State, outer Outer_State) (value Value) {
	defer func() { Value_Invariants(value, "digest_sum_md5.value") }()
	Inner_State_Invariants(inner, "digest_sum_md5.inner")
	Outer_State_Invariants(outer, "digest_sum_md5.outer")
	inner_digest := (*md5.Digest)(unsafe.Pointer(&inner[bits.BIT_COUNT_MINIMUM]))
	outer_digest := (*md5.Digest)(unsafe.Pointer(&outer[bits.BIT_COUNT_MINIMUM]))
	inner_value := md5.Digest_Sum(inner_digest)
	md5.Digest_Write(outer_digest, inner_value[:])
	tag := md5.Digest_Sum(outer_digest)
	copy(value[:], tag[:])
	return value
}

func digest_sum_sha_1(inner Inner_State, outer Outer_State) (value Value) {
	defer func() { Value_Invariants(value, "digest_sum_sha_1.value") }()
	Inner_State_Invariants(inner, "digest_sum_sha_1.inner")
	Outer_State_Invariants(outer, "digest_sum_sha_1.outer")
	inner_digest := (*sha1.Digest)(unsafe.Pointer(&inner[bits.BIT_COUNT_MINIMUM]))
	outer_digest := (*sha1.Digest)(unsafe.Pointer(&outer[bits.BIT_COUNT_MINIMUM]))
	inner_value := sha1.Digest_Sum(inner_digest)
	sha1.Digest_Write(outer_digest, inner_value[:])
	tag := sha1.Digest_Sum(outer_digest)
	copy(value[:], tag[:])
	return value
}

func digest_sum_sha_256(inner Inner_State, outer Outer_State) (value Value) {
	defer func() { Value_Invariants(value, "digest_sum_sha_256.value") }()
	Inner_State_Invariants(inner, "digest_sum_sha_256.inner")
	Outer_State_Invariants(outer, "digest_sum_sha_256.outer")
	inner_digest := (*sha256.Digest)(unsafe.Pointer(&inner[bits.BIT_COUNT_MINIMUM]))
	outer_digest := (*sha256.Digest)(unsafe.Pointer(&outer[bits.BIT_COUNT_MINIMUM]))
	var inner_value [sha256.DIGEST_256_SIZE]byte
	inner_count, _ := sha256.Digest_Sum_Into(inner_digest, inner_value[:])
	sha256.Digest_Write(outer_digest, inner_value[:inner_count])
	sha256.Digest_Sum_Into(outer_digest, value[:])
	return value
}

func digest_sum_sha_512(inner Inner_State, outer Outer_State) (value Value) {
	defer func() { Value_Invariants(value, "digest_sum_sha_512.value") }()
	Inner_State_Invariants(inner, "digest_sum_sha_512.inner")
	Outer_State_Invariants(outer, "digest_sum_sha_512.outer")
	inner_digest := (*sha512.Digest)(unsafe.Pointer(&inner[bits.BIT_COUNT_MINIMUM]))
	outer_digest := (*sha512.Digest)(unsafe.Pointer(&outer[bits.BIT_COUNT_MINIMUM]))
	var inner_value [sha512.DIGEST_512_SIZE]byte
	inner_count, _ := sha512.Digest_Sum_Into(inner_digest, inner_value[:])
	sha512.Digest_Write(outer_digest, inner_value[:inner_count])
	sha512.Digest_Sum_Into(outer_digest, value[:])
	return value
}

func kind_size(kind Kind) (size Size) {
	defer func() { Size_Invariants(size, "kind_size.size") }()
	Kind_Invariants(kind, "kind_size.kind")
	switch kind {
	case KIND_MD5:
		return md5.DIGEST_SIZE
	case KIND_SHA_1:
		return sha1.DIGEST_SIZE
	case KIND_SHA_224, KIND_SHA_512_224:
		return sha256.DIGEST_224_SIZE
	case KIND_SHA_256, KIND_SHA_512_256:
		return sha256.DIGEST_256_SIZE
	case KIND_SHA_384:
		return sha512.DIGEST_384_SIZE
	default:
		return sha512.DIGEST_512_SIZE
	}
}

func kind_block_size(kind Kind) (size Block_Size) {
	defer func() { Block_Size_Invariants(size, "kind_block_size.size") }()
	Kind_Invariants(kind, "kind_block_size.kind")
	if kind <= KIND_SHA_256 {
		return md5.BLOCK_SIZE
	}
	return sha512.BLOCK_SIZE
}

func digest_require(digest *Digest) {
	Digest_Invariants(*digest, "digest_require.digest")
	Kind_Invariants(digest.Kind, "digest_require.kind")
	if digest.Kind > KIND_SHA_512 {
		panic("hmac: kind is invalid")
	}
	aver.Always(
		digest.Ready[READY_INDEX] == READY_COMPLETE_MARKER,
		"HMAC operations require Digest_Init.",
	)
	if digest.Ready[READY_INDEX] != READY_COMPLETE_MARKER {
		panic("hmac: digest is not initialized")
	}
}
