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
	"local/james-orcales/shared/sim/aver/default"
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
type Ready bool

// Ready_Invariants covers uninitialized and keyed caller state.
func Ready_Invariants(value Ready, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "HMAC state is initialized.").
		Ensure()
}

// Equality exposes constant-time comparison result.
type Equality bool

// Equality_Invariants covers equal and unequal tags.
func Equality_Invariants(value Equality, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "HMAC tags are equal.").
		Ensure()
}

// Pad names exact scratch supplied by Digest_Init.
type Pad []byte

// Pad_Invariants fixes stack-owned pad capacity.
func Pad_Invariants(value Pad, _ aver.Namespace) {
	aver.Always(len(value) == BLOCK_SIZE_MAXIMUM, "HMAC pad has fixed width.")
}

// Initial_State stores selected keyed inner baseline in widest aligned storage.
type Initial_State sha512.Digest

// Initial_State_Invariants proves widest digest fits without heap storage.
func Initial_State_Invariants(value Initial_State, _ aver.Namespace) {
	aver.Always(
		unsafe.Sizeof(value) == unsafe.Sizeof(sha512.Digest{}),
		"HMAC initial state has fixed width.",
	)
}

// Inner_State stores selected live inner digest in aligned caller storage.
type Inner_State sha512.Digest

// Inner_State_Invariants proves widest digest fits without heap storage.
func Inner_State_Invariants(value Inner_State, _ aver.Namespace) {
	aver.Always(
		unsafe.Sizeof(value) == unsafe.Sizeof(sha512.Digest{}),
		"HMAC inner state has fixed width.",
	)
}

// Outer_State stores selected keyed outer digest in aligned caller storage.
type Outer_State sha512.Digest

// Outer_State_Invariants proves widest digest fits without heap storage.
func Outer_State_Invariants(value Outer_State, _ aver.Namespace) {
	aver.Always(
		unsafe.Sizeof(value) == unsafe.Sizeof(sha512.Digest{}),
		"HMAC outer state has fixed width.",
	)
}

// Digest_Handle names mutable keyed state.
type Digest_Handle *Digest

// Digest_Handle_Invariants composes state when storage exists.
func Digest_Handle_Invariants(value Digest_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Digest_Invariants(*value, namespace)
}

// Initial_State_Destination names mutable keyed-inner baseline.
type Initial_State_Destination *Initial_State

// Initial_State_Destination_Invariants composes storage when present.
func Initial_State_Destination_Invariants(
	value Initial_State_Destination, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Initial_State_Invariants(*value, namespace)
}

// Inner_State_Destination names mutable live inner state.
type Inner_State_Destination *Inner_State

// Inner_State_Destination_Invariants composes storage when present.
func Inner_State_Destination_Invariants(
	value Inner_State_Destination, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Inner_State_Invariants(*value, namespace)
}

// Outer_State_Destination names mutable keyed outer state.
type Outer_State_Destination *Outer_State

// Outer_State_Destination_Invariants composes storage when present.
func Outer_State_Destination_Invariants(
	value Outer_State_Destination, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Outer_State_Invariants(*value, namespace)
}

// MD5_Destination names exact MD5 tag storage.
type MD5_Destination []byte

// MD5_Destination_Invariants prevents partial MD5 tags.
func MD5_Destination_Invariants(value MD5_Destination, _ aver.Namespace) {
	aver.Always(len(value) == md5.DIGEST_SIZE, "HMAC MD5 destination has exact width.")
}

// SHA_1_Destination names exact SHA-1 tag storage.
type SHA_1_Destination []byte

// SHA_1_Destination_Invariants prevents partial SHA-1 tags.
func SHA_1_Destination_Invariants(value SHA_1_Destination, _ aver.Namespace) {
	aver.Always(len(value) == sha1.DIGEST_SIZE, "HMAC SHA-1 destination has exact width.")
}

// SHA_256_Destination names SHA-224 or SHA-256 tag storage.
type SHA_256_Destination []byte

// SHA_256_Destination_Invariants permits both family widths.
func SHA_256_Destination_Invariants(value SHA_256_Destination, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Int(len(value), sha256.DIGEST_224_SIZE, sha256.DIGEST_256_SIZE).
		Ensure()
}

// SHA_512_Destination names a SHA-512 family tag storage.
type SHA_512_Destination []byte

// SHA_512_Destination_Invariants spans truncated and complete family widths.
func SHA_512_Destination_Invariants(value SHA_512_Destination, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), sha512.DIGEST_224_SIZE, sha512.DIGEST_512_SIZE).
		Ensure()
}

// Storage keeps selected inner, initial, and outer states in caller storage.
type Storage struct {
	// Kind preserves selected hash across reset and sum.
	Kind Kind
	// Initial restores keyed inner state.
	Initial Initial_State
	// Inner accepts message chunks.
	Inner Inner_State
	// Outer completes authentication.
	Outer Outer_State
}

// Storage_Invariants composes selector and aligned fixed-capacity hash states.
func Storage_Invariants(value Storage, namespace aver.Namespace) {
	Kind_Invariants(value.Kind, namespace)
	Initial_State_Invariants(value.Initial, namespace)
	Inner_State_Invariants(value.Inner, namespace)
	Outer_State_Invariants(value.Outer, namespace)
}

// Digest keeps lifecycle identity beside its mutable keyed state.
type Digest struct {
	// Storage remains embedded so existing field access stays direct.
	Storage
	// Ready rejects zero caller storage before keyed operations.
	Ready Ready
}

// Digest_Invariants composes keyed state and lifecycle identity.
func Digest_Invariants(value Digest, namespace aver.Namespace) {
	Storage_Invariants(value.Storage, namespace)
	Ready_Invariants(value.Ready, namespace)
}

// Storage_Destination names mutable keyed state after lifecycle validation.
type Storage_Destination *Storage

// Storage_Destination_Invariants composes storage when present.
func Storage_Destination_Invariants(value Storage_Destination, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Storage_Invariants(*value, namespace)
}

// Digest_Init clears prior key material before constructing selected pads.
func Digest_Init(digest Digest_Handle, kind Kind, key Key) {
	Digest_Handle_Invariants(digest, "Digest_Init.digest.input")
	Kind_Invariants(kind, "Digest_Init.kind")
	Key_Invariants(key, "Digest_Init.key")
	*digest = Digest{Storage: Storage{Kind: kind}}
	var inner [BLOCK_SIZE_MAXIMUM]byte
	var outer [BLOCK_SIZE_MAXIMUM]byte
	inner_pad(Pad(inner[:]), kind, key)
	outer_pad(Pad(outer[:]), Pad(inner[:]), kind)
	digest_state_init(&digest.Storage, kind, Pad(inner[:]), Pad(outer[:]))
	digest.Ready = true
	Storage_Invariants(digest.Storage, "Digest_Init.digest.output")
}

// Digest_Reset restores keyed inner state without retaining message state.
func Digest_Reset(digest Digest_Handle) {
	Digest_Handle_Invariants(digest, "Digest_Reset.digest.input")
	digest_require(digest)
	digest.Inner = Inner_State(digest.Initial)
	Storage_Invariants(digest.Storage, "Digest_Reset.digest.output")
}

// Digest_Write consumes one complete bounded source chunk.
func Digest_Write(digest Digest_Handle, source Source) (count Count) {
	defer func() { Count_Invariants(count, "Digest_Write.count") }()
	Digest_Handle_Invariants(digest, "Digest_Write.digest.input")
	Source_Invariants(source, "Digest_Write.source")
	digest_require(digest)
	digest_write(&digest.Storage, source)
	Storage_Invariants(digest.Storage, "Digest_Write.digest.output")
	return Count(len(source))
}

// Digest_Sum_Into leaves short destination untouched.
func Digest_Sum_Into(
	digest Digest_Handle,
	destination Destination,
) (count Output_Count, status Output_Status) {
	defer func() {
		Output_Count_Invariants(count, "Digest_Sum_Into.count")
		Output_Status_Invariants(status, "Digest_Sum_Into.status")
	}()
	Digest_Handle_Invariants(digest, "Digest_Sum_Into.digest")
	Destination_Invariants(destination, "Digest_Sum_Into.destination")
	digest_require(digest)
	count = Output_Count(Digest_Size(digest))
	if len(destination) < int(count) {
		return count, OUTPUT_STATUS_TOO_SMALL
	}
	switch digest.Kind {
	case KIND_MD5:
		digest_sum_md5(
			MD5_Destination(destination[:count]), digest.Inner, digest.Outer,
		)
	case KIND_SHA_1:
		digest_sum_sha_1(
			SHA_1_Destination(destination[:count]), digest.Inner, digest.Outer,
		)
	case KIND_SHA_224, KIND_SHA_256:
		digest_sum_sha_256(
			SHA_256_Destination(destination[:count]), digest.Inner, digest.Outer,
		)
	default:
		digest_sum_sha_512(
			SHA_512_Destination(destination[:count]), digest.Inner, digest.Outer,
		)
	}
	return count, OUTPUT_STATUS_OK
}

// Digest_Clone_Into copies live keyed state without aliasing caller storage.
func Digest_Clone_Into(destination Digest_Handle, source Digest_Handle) {
	Digest_Handle_Invariants(destination, "Digest_Clone_Into.destination.input")
	Digest_Handle_Invariants(source, "Digest_Clone_Into.source")
	digest_require(source)
	*destination = *source
	Storage_Invariants(destination.Storage, "Digest_Clone_Into.destination.output")
}

// Digest_Size reports selected tag width.
func Digest_Size(digest Digest_Handle) (size Size) {
	defer func() { Size_Invariants(size, "Digest_Size.size") }()
	Digest_Handle_Invariants(digest, "Digest_Size.digest")
	digest_require(digest)
	return kind_size(digest.Kind)
}

// Digest_Block_Size reports selected compression-block width.
func Digest_Block_Size(digest Digest_Handle) (size Block_Size) {
	defer func() { Block_Size_Invariants(size, "Digest_Block_Size.size") }()
	Digest_Handle_Invariants(digest, "Digest_Block_Size.digest")
	digest_require(digest)
	return kind_block_size(digest.Kind)
}

// Equal compares bounded tags without content-dependent exit.
func Equal(left Tag, right Tag) (equal Equality) {
	defer func() { Equality_Invariants(equal, "Equal.equal") }()
	Tag_Invariants(left, "Equal.left")
	Tag_Invariants(right, "Equal.right")
	decision := subtle.Constant_Time_Compare(subtle.Source(left), subtle.Source(right))
	return Equality(decision == subtle.DECISION_TRUE)
}

func inner_pad(pad Pad, kind Kind, key Key) {
	Pad_Invariants(pad, "inner_pad.pad")
	Kind_Invariants(kind, "inner_pad.kind")
	Key_Invariants(key, "inner_pad.key")
	block_size := int(kind_block_size(kind))
	if len(key) > block_size {
		hash_key(kind, Reduced_Key(key), pad)
	} else {
		copy(pad[:block_size], key)
	}
	for index := range block_size {
		pad[index] ^= INNER_PAD_BYTE
	}
}

func outer_pad(outer Pad, inner Pad, kind Kind) {
	Pad_Invariants(outer, "outer_pad.outer")
	Kind_Invariants(kind, "outer_pad.kind")
	Pad_Invariants(inner, "outer_pad.inner")
	copy(outer, inner)
	block_size := int(kind_block_size(kind))
	for index := range block_size {
		outer[index] ^= INNER_PAD_BYTE ^ OUTER_PAD_BYTE
	}
}

func hash_key(kind Kind, key Reduced_Key, destination Pad) {
	Kind_Invariants(kind, "hash_key.kind")
	Reduced_Key_Invariants(key, "hash_key.key")
	Pad_Invariants(destination, "hash_key.destination.input")
	switch kind {
	case KIND_MD5:
		md5.Checksum_Into(md5.Destination(destination[:md5.DIGEST_SIZE]), md5.Source(key))
	case KIND_SHA_1:
		sha1.Checksum_Into(
			sha1.Destination(destination[:sha1.DIGEST_SIZE]), sha1.Source(key),
		)
	case KIND_SHA_224:
		sha256.Checksum_Into(
			sha256.Destination(destination[:sha256.DIGEST_224_SIZE]),
			sha256.KIND_SHA_224, sha256.Source(key),
		)
	case KIND_SHA_256:
		sha256.Checksum_Into(
			sha256.Destination(destination[:sha256.DIGEST_256_SIZE]),
			sha256.KIND_SHA_256, sha256.Source(key),
		)
	case KIND_SHA_384:
		sha512.Checksum_Into(
			sha512.Destination(destination[:sha512.DIGEST_384_SIZE]),
			sha512.KIND_SHA_384, sha512.Source(key),
		)
	case KIND_SHA_512_224:
		sha512.Checksum_Into(
			sha512.Destination(destination[:sha512.DIGEST_224_SIZE]),
			sha512.KIND_SHA_512_224, sha512.Source(key),
		)
	case KIND_SHA_512_256:
		sha512.Checksum_Into(
			sha512.Destination(destination[:sha512.DIGEST_256_SIZE]),
			sha512.KIND_SHA_512_256, sha512.Source(key),
		)
	default:
		sha512.Checksum_Into(
			sha512.Destination(destination[:sha512.DIGEST_512_SIZE]),
			sha512.KIND_SHA_512, sha512.Source(key),
		)
	}
	Pad_Invariants(destination, "hash_key.destination.output")
}

func digest_state_init(digest Storage_Destination, kind Kind, inner Pad, outer Pad) {
	Storage_Destination_Invariants(digest, "digest_state_init.digest.input")
	Kind_Invariants(kind, "digest_state_init.kind")
	Pad_Invariants(inner, "digest_state_init.inner")
	Pad_Invariants(outer, "digest_state_init.outer")
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
	Storage_Invariants(*digest, "digest_state_init.digest.output")
}

func digest_state_init_md5(
	initial Initial_State_Destination,
	live Inner_State_Destination,
	keyed_outer Outer_State_Destination,
	inner Pad,
	outer Pad,
) {
	Initial_State_Destination_Invariants(initial, "digest_state_init_md5.initial.input")
	Inner_State_Destination_Invariants(live, "digest_state_init_md5.live.input")
	Outer_State_Destination_Invariants(keyed_outer, "digest_state_init_md5.keyed_outer.input")
	Pad_Invariants(inner, "digest_state_init_md5.inner")
	Pad_Invariants(outer, "digest_state_init_md5.outer")
	initial_digest := (*md5.Digest)(unsafe.Pointer(initial))
	live_digest := (*md5.Digest)(unsafe.Pointer(live))
	outer_digest := (*md5.Digest)(unsafe.Pointer(keyed_outer))
	md5.Digest_Init(initial_digest)
	md5.Digest_Write(initial_digest, md5.Source(inner[:md5.BLOCK_SIZE]))
	*live_digest = *initial_digest
	md5.Digest_Init(outer_digest)
	md5.Digest_Write(outer_digest, md5.Source(outer[:md5.BLOCK_SIZE]))
	Initial_State_Invariants(*initial, "digest_state_init_md5.initial.output")
	Inner_State_Invariants(*live, "digest_state_init_md5.live.output")
	Outer_State_Invariants(*keyed_outer, "digest_state_init_md5.keyed_outer.output")
}

func digest_state_init_sha_1(
	initial Initial_State_Destination,
	live Inner_State_Destination,
	keyed_outer Outer_State_Destination,
	inner Pad,
	outer Pad,
) {
	Initial_State_Destination_Invariants(initial, "digest_state_init_sha_1.initial.input")
	Inner_State_Destination_Invariants(live, "digest_state_init_sha_1.live.input")
	Outer_State_Destination_Invariants(keyed_outer, "digest_state_init_sha_1.keyed_outer.input")
	Pad_Invariants(inner, "digest_state_init_sha_1.inner")
	Pad_Invariants(outer, "digest_state_init_sha_1.outer")
	initial_digest := (*sha1.Digest)(unsafe.Pointer(initial))
	live_digest := (*sha1.Digest)(unsafe.Pointer(live))
	outer_digest := (*sha1.Digest)(unsafe.Pointer(keyed_outer))
	sha1.Digest_Init(initial_digest)
	sha1.Digest_Write(initial_digest, sha1.Source(inner[:sha1.BLOCK_SIZE]))
	*live_digest = *initial_digest
	sha1.Digest_Init(outer_digest)
	sha1.Digest_Write(outer_digest, sha1.Source(outer[:sha1.BLOCK_SIZE]))
	Initial_State_Invariants(*initial, "digest_state_init_sha_1.initial.output")
	Inner_State_Invariants(*live, "digest_state_init_sha_1.live.output")
	Outer_State_Invariants(*keyed_outer, "digest_state_init_sha_1.keyed_outer.output")
}

func digest_state_init_sha_256(
	initial Initial_State_Destination,
	live Inner_State_Destination,
	keyed_outer Outer_State_Destination,
	inner Pad,
	outer Pad,
	kind sha256.Kind,
) {
	Initial_State_Destination_Invariants(initial, "digest_state_init_sha_256.initial.input")
	Inner_State_Destination_Invariants(live, "digest_state_init_sha_256.live.input")
	Outer_State_Destination_Invariants(
		keyed_outer, "digest_state_init_sha_256.keyed_outer.input",
	)
	Pad_Invariants(inner, "digest_state_init_sha_256.inner")
	Pad_Invariants(outer, "digest_state_init_sha_256.outer")
	sha256.Kind_Invariants(kind, "digest_state_init_sha_256.kind")
	initial_digest := (*sha256.Digest)(unsafe.Pointer(initial))
	live_digest := (*sha256.Digest)(unsafe.Pointer(live))
	outer_digest := (*sha256.Digest)(unsafe.Pointer(keyed_outer))
	sha256.Digest_Init(initial_digest, kind)
	sha256.Digest_Write(initial_digest, sha256.Source(inner[:sha256.BLOCK_SIZE]))
	*live_digest = *initial_digest
	sha256.Digest_Init(outer_digest, kind)
	sha256.Digest_Write(outer_digest, sha256.Source(outer[:sha256.BLOCK_SIZE]))
	Initial_State_Invariants(*initial, "digest_state_init_sha_256.initial.output")
	Inner_State_Invariants(*live, "digest_state_init_sha_256.live.output")
	Outer_State_Invariants(*keyed_outer, "digest_state_init_sha_256.keyed_outer.output")
}

func digest_state_init_sha_512(
	initial Initial_State_Destination,
	live Inner_State_Destination,
	keyed_outer Outer_State_Destination,
	inner Pad,
	outer Pad,
	kind sha512.Kind,
) {
	Initial_State_Destination_Invariants(initial, "digest_state_init_sha_512.initial.input")
	Inner_State_Destination_Invariants(live, "digest_state_init_sha_512.live.input")
	Outer_State_Destination_Invariants(
		keyed_outer, "digest_state_init_sha_512.keyed_outer.input",
	)
	Pad_Invariants(inner, "digest_state_init_sha_512.inner")
	Pad_Invariants(outer, "digest_state_init_sha_512.outer")
	sha512.Kind_Invariants(kind, "digest_state_init_sha_512.kind")
	initial_digest := (*sha512.Digest)(unsafe.Pointer(initial))
	live_digest := (*sha512.Digest)(unsafe.Pointer(live))
	outer_digest := (*sha512.Digest)(unsafe.Pointer(keyed_outer))
	sha512.Digest_Init(initial_digest, kind)
	sha512.Digest_Write(initial_digest, sha512.Source(inner[:sha512.BLOCK_SIZE]))
	*live_digest = *initial_digest
	sha512.Digest_Init(outer_digest, kind)
	sha512.Digest_Write(outer_digest, sha512.Source(outer[:sha512.BLOCK_SIZE]))
	Initial_State_Invariants(*initial, "digest_state_init_sha_512.initial.output")
	Inner_State_Invariants(*live, "digest_state_init_sha_512.live.output")
	Outer_State_Invariants(*keyed_outer, "digest_state_init_sha_512.keyed_outer.output")
}

func digest_write(digest Storage_Destination, source Source) {
	Storage_Destination_Invariants(digest, "digest_write.digest.input")
	Source_Invariants(source, "digest_write.source")
	switch digest.Kind {
	case KIND_MD5:
		inner := (*md5.Digest)(unsafe.Pointer(&digest.Inner))
		md5.Digest_Write(inner, md5.Source(source))
	case KIND_SHA_1:
		inner := (*sha1.Digest)(unsafe.Pointer(&digest.Inner))
		sha1.Digest_Write(inner, sha1.Source(source))
	case KIND_SHA_224, KIND_SHA_256:
		inner := (*sha256.Digest)(unsafe.Pointer(&digest.Inner))
		sha256.Digest_Write(inner, sha256.Source(source))
	default:
		inner := (*sha512.Digest)(unsafe.Pointer(&digest.Inner))
		sha512.Digest_Write(inner, sha512.Source(source))
	}
	Storage_Invariants(*digest, "digest_write.digest.output")
}

func digest_sum_md5(destination MD5_Destination, inner Inner_State, outer Outer_State) {
	MD5_Destination_Invariants(destination, "digest_sum_md5.destination")
	Inner_State_Invariants(inner, "digest_sum_md5.inner")
	Outer_State_Invariants(outer, "digest_sum_md5.outer")
	inner_digest := (*md5.Digest)(unsafe.Pointer(&inner))
	outer_digest := (*md5.Digest)(unsafe.Pointer(&outer))
	var inner_value [md5.DIGEST_SIZE]byte
	md5.Digest_Sum_Into(inner_digest, md5.Destination(inner_value[:]))
	md5.Digest_Write(outer_digest, inner_value[:])
	md5.Digest_Sum_Into(outer_digest, md5.Destination(destination))
}

func digest_sum_sha_1(destination SHA_1_Destination, inner Inner_State, outer Outer_State) {
	SHA_1_Destination_Invariants(destination, "digest_sum_sha_1.destination")
	Inner_State_Invariants(inner, "digest_sum_sha_1.inner")
	Outer_State_Invariants(outer, "digest_sum_sha_1.outer")
	inner_digest := (*sha1.Digest)(unsafe.Pointer(&inner))
	outer_digest := (*sha1.Digest)(unsafe.Pointer(&outer))
	var inner_value [sha1.DIGEST_SIZE]byte
	sha1.Digest_Sum_Into(inner_digest, sha1.Destination(inner_value[:]))
	sha1.Digest_Write(outer_digest, sha1.Source(inner_value[:]))
	sha1.Digest_Sum_Into(outer_digest, sha1.Destination(destination))
}

func digest_sum_sha_256(
	destination SHA_256_Destination, inner Inner_State, outer Outer_State,
) {
	SHA_256_Destination_Invariants(destination, "digest_sum_sha_256.destination")
	Inner_State_Invariants(inner, "digest_sum_sha_256.inner")
	Outer_State_Invariants(outer, "digest_sum_sha_256.outer")
	inner_digest := (*sha256.Digest)(unsafe.Pointer(&inner))
	outer_digest := (*sha256.Digest)(unsafe.Pointer(&outer))
	var inner_value [sha256.DIGEST_256_SIZE]byte
	inner_count, _ := sha256.Digest_Sum_Into(inner_digest, inner_value[:])
	sha256.Digest_Write(outer_digest, inner_value[:inner_count])
	sha256.Digest_Sum_Into(outer_digest, sha256.Destination(destination))
}

func digest_sum_sha_512(
	destination SHA_512_Destination, inner Inner_State, outer Outer_State,
) {
	SHA_512_Destination_Invariants(destination, "digest_sum_sha_512.destination")
	Inner_State_Invariants(inner, "digest_sum_sha_512.inner")
	Outer_State_Invariants(outer, "digest_sum_sha_512.outer")
	inner_digest := (*sha512.Digest)(unsafe.Pointer(&inner))
	outer_digest := (*sha512.Digest)(unsafe.Pointer(&outer))
	var inner_value [sha512.DIGEST_512_SIZE]byte
	inner_count, _ := sha512.Digest_Sum_Into(inner_digest, inner_value[:])
	sha512.Digest_Write(outer_digest, inner_value[:inner_count])
	sha512.Digest_Sum_Into(outer_digest, sha512.Destination(destination))
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

func digest_require(digest Digest_Handle) {
	Digest_Handle_Invariants(digest, "digest_require.digest")
	Kind_Invariants(digest.Kind, "digest_require.kind")
	aver.Always(
		bool(digest.Ready),
		"HMAC operations require Digest_Init.",
	)
}
