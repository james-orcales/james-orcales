// Package uuid generates and parses RFC 9562 (formerly RFC 4122) and DCE 1.1
// UUIDs. It is a dependency-injected port of github.com/google/uuid: the upstream
// package hides its entropy (crypto/rand), its clock (time.Now), and its host node
// (net.Interfaces) behind package globals, all of which the house linter forbids
// and deterministic simulation cannot replay. Here they are fields of a Generator,
// injected at construction — production wires the real sources through uuid/default,
// a simulation wires a seeded reader and a virtual clock, and the same seed
// reproduces the same UUIDs bit-for-bit.
//
// The linter also bans methods that do not satisfy a stdlib interface, so an
// accessor like upstream uuid.Version() is free function UUID_Version(u).
// Encoding interface methods remain. Database conversion uses shared closed driver
// union instead of reflection-backed standard SQL values.
package uuid

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/crypto/md5"
	"local/james-orcales/shared/crypto/prng"
	"local/james-orcales/shared/crypto/sha1"
	"local/james-orcales/shared/database/driver"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/encoding/hex"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/simulation/aver/default"
	"local/james-orcales/shared/simulation/time"
	"local/james-orcales/shared/strings"
)

// UUID_BIT_COUNT is RFC 9562 binary width.
const UUID_BIT_COUNT = bits.BIT_COUNT_64_MAXIMUM * 2

// UUID_BYTE_COUNT converts RFC 9562 width to bytes.
const UUID_BYTE_COUNT = UUID_BIT_COUNT / bits.BIT_COUNT_8_MAXIMUM

// NODE_BIT_COUNT is RFC 9562 node-field width.
const NODE_BIT_COUNT = bits.BIT_COUNT_16_MAXIMUM * 3

// NODE_BYTE_COUNT converts RFC 9562 node-field width to bytes.
const NODE_BYTE_COUNT = NODE_BIT_COUNT / bits.BIT_COUNT_8_MAXIMUM

// CLOCK_SEQUENCE_BIT_COUNT is RFC 9562 clock-sequence width after variant bits.
const CLOCK_SEQUENCE_BIT_COUNT = bits.BIT_COUNT_16_MAXIMUM - 2

// CLOCK_SEQUENCE_VALUE_COUNT is count of distinct clock sequences.
const CLOCK_SEQUENCE_VALUE_COUNT = 1 << CLOCK_SEQUENCE_BIT_COUNT

// CLOCK_SEQUENCE_MAXIMUM is final clock-sequence value.
const CLOCK_SEQUENCE_MAXIMUM = CLOCK_SEQUENCE_VALUE_COUNT - 1

// CLOCK_SEQUENCE_BYTE_COUNT stores the complete RFC 9562 clock sequence.
const CLOCK_SEQUENCE_BYTE_COUNT = bits.BIT_COUNT_16_MAXIMUM / bits.BIT_COUNT_8_MAXIMUM

// VERSION_BIT_COUNT is one hexadecimal version nibble.
const VERSION_BIT_COUNT = bits.BIT_COUNT_8_MAXIMUM / 2

// VERSION_VALUE_COUNT is count of values one version nibble can represent.
const VERSION_VALUE_COUNT = 1 << VERSION_BIT_COUNT

// VERSION_MAXIMUM is final version nibble value.
const VERSION_MAXIMUM = VERSION_VALUE_COUNT - 1

// UUID_HEXADECIMAL_BYTE_COUNT is unhyphenated textual width.
const UUID_HEXADECIMAL_BYTE_COUNT = UUID_BYTE_COUNT * hex.ENCODED_BYTE_SIZE

// UUID_TEXT_HYPHEN_COUNT is separator count in canonical UUID text.
const UUID_TEXT_HYPHEN_COUNT = 4

// UUID_TEXT_LARGE_GROUP_BYTE_COUNT is leading 32-bit field width.
const UUID_TEXT_LARGE_GROUP_BYTE_COUNT = bits.BIT_COUNT_32_MAXIMUM /
	bits.BIT_COUNT_8_MAXIMUM

// UUID_TEXT_SMALL_GROUP_BYTE_COUNT is each middle 16-bit field width.
const UUID_TEXT_SMALL_GROUP_BYTE_COUNT = bits.BIT_COUNT_16_MAXIMUM /
	bits.BIT_COUNT_8_MAXIMUM

// UUID_TEXT_FIRST_HYPHEN_INDEX follows leading 32-bit field.
const UUID_TEXT_FIRST_HYPHEN_INDEX = UUID_TEXT_LARGE_GROUP_BYTE_COUNT * hex.ENCODED_BYTE_SIZE

// UUID_TEXT_SECOND_HYPHEN_INDEX follows first 16-bit field.
const UUID_TEXT_SECOND_HYPHEN_INDEX = UUID_TEXT_FIRST_HYPHEN_INDEX + 1 +
	UUID_TEXT_SMALL_GROUP_BYTE_COUNT*hex.ENCODED_BYTE_SIZE

// UUID_TEXT_THIRD_HYPHEN_INDEX follows second 16-bit field.
const UUID_TEXT_THIRD_HYPHEN_INDEX = UUID_TEXT_SECOND_HYPHEN_INDEX + 1 +
	UUID_TEXT_SMALL_GROUP_BYTE_COUNT*hex.ENCODED_BYTE_SIZE

// UUID_TEXT_FINAL_HYPHEN_INDEX follows third 16-bit field.
const UUID_TEXT_FINAL_HYPHEN_INDEX = UUID_TEXT_THIRD_HYPHEN_INDEX + 1 +
	UUID_TEXT_SMALL_GROUP_BYTE_COUNT*hex.ENCODED_BYTE_SIZE

// UUID_TEXT_BYTE_COUNT is the canonical hyphenated text width.
const UUID_TEXT_BYTE_COUNT = UUID_HEXADECIMAL_BYTE_COUNT + UUID_TEXT_HYPHEN_COUNT

// UUID_BRACE_BYTE_COUNT is wrapper width around braced UUID text.
const UUID_BRACE_BYTE_COUNT = 2

// UUID_BRACED_BYTE_COUNT is complete braced UUID text width.
const UUID_BRACED_BYTE_COUNT = UUID_TEXT_BYTE_COUNT + UUID_BRACE_BYTE_COUNT

// UUID_URN_PREFIX starts RFC 2141 UUID URNs.
const UUID_URN_PREFIX = "urn:uuid:"

// UUID_URN_PREFIX_BYTE_COUNT accounts for the complete "urn:uuid:" prefix.
const UUID_URN_PREFIX_BYTE_COUNT = len(UUID_URN_PREFIX)

// UUID_URN_BYTE_COUNT prevents a URN formatter from allocating a larger buffer.
const UUID_URN_BYTE_COUNT = UUID_URN_PREFIX_BYTE_COUNT + UUID_TEXT_BYTE_COUNT

// UUID is a 128-bit RFC 9562 Universally Unique IDentifier.
type UUID [UUID_BYTE_COUNT]byte

// UUID_Invariants fixes RFC 9562 storage width.
func UUID_Invariants(value UUID, _ aver.Namespace) {
	aver.Always(len(value) == UUID_BYTE_COUNT, "UUID storage has RFC 9562 width.")
}

// UUIDs is a slice of UUID, given a name so UUIDs_Strings can hang off it.
type UUIDs []UUID

// UUIDs_Invariants bounds UUID collections by shared byte-storage capacity.
func UUIDs_Invariants(value UUIDs, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, UUIDS_COUNT_MAXIMUM).
		Ensure()
}

// UUIDS_COUNT_MAXIMUM is largest UUID collection fitting shared byte capacity.
const UUIDS_COUNT_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM / UUID_BYTE_COUNT

// Version is the version nibble of a UUID: which generation algorithm produced it.
type Version byte

// Version_Invariants bounds one hexadecimal version nibble.
func Version_Invariants(value Version, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), bits.WORD_8_MINIMUM, VERSION_MAXIMUM).
		Ensure()
}

// Variant is the layout variant of a UUID: which bit interpretation its fields follow.
type Variant byte

// Variant_Invariants admits every recognized UUID layout variant.
func Variant_Invariants(value Variant, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(VARIANT_RFC_4122), uint8(VARIANT_RESERVED),
			uint8(VARIANT_MICROSOFT), uint8(VARIANT_FUTURE),
		).
		Ensure()
}

// Domain is a DCE 1.1 (Version 2) security domain.
type Domain byte

// Domain_Invariants admits three DCE Security domains.
func Domain_Invariants(value Domain, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(DOMAIN_PERSON), uint8(DOMAIN_GROUP),
			uint8(DOMAIN_ORGANIZATION),
		).
		Ensure()
}

// Time is an RFC 9562 timestamp: 100-nanosecond ticks since 15 Oct 1582.
type Time int64

// Time_Invariants bounds timestamp to RFC 9562 60-bit field.
func Time_Invariants(value Time, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), TIME_MINIMUM, TIME_MAXIMUM).
		Ensure()
}

// Generated_Time is timestamp subset reachable from signed nanosecond Clock.
type Generated_Time int64

// Generated_Time_Invariants matches arithmetic image of complete Clock domain.
func Generated_Time_Invariants(value Generated_Time, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), GENERATED_TIME_MINIMUM, GENERATED_TIME_MAXIMUM).
		Ensure()
}

// Node is RFC 9562 node identifier.
type Node [NODE_BYTE_COUNT]byte

// Node_Invariants fixes RFC 9562 node storage width.
func Node_Invariants(value Node, _ aver.Namespace) {
	aver.Always(len(value) == NODE_BYTE_COUNT, "UUID node has RFC 9562 width.")
}

// Clock_Sequence separates timestamp collisions within one generator.
type Clock_Sequence uint16

// Clock_Sequence_Invariants bounds RFC 9562 sequence field before variant stamping.
func Clock_Sequence_Invariants(value Clock_Sequence, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), bits.WORD_16_MINIMUM, uint16(CLOCK_SEQUENCE_MAXIMUM),
		).
		Ensure()
}

// Valid records whether nullable UUID contains value.
type Valid bool

// Valid_Invariants requires valid and invalid nullable values.
func Valid_Invariants(value Valid, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A nullable UUID contains a value.").
		Ensure()
}

// Identifier is DCE Security identifier embedded in UUID low field.
type Identifier uint32

// Identifier_Invariants admits complete DCE identifier storage domain.
func Identifier_Invariants(value Identifier, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, bits.WORD_32_MAXIMUM).
		Ensure()
}

// Name is bounded byte sequence hashed by name-based generators.
type Name []byte

// Name_Invariants bounds malicious names before hashing work.
func Name_Invariants(value Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, bytes.SLICE_SIZE_MAXIMUM).
		Ensure()
}

// Text_Unvalidated is caller text before UUID syntax validation.
type Text_Unvalidated string

// Text_Unvalidated_Invariants bounds malicious input before syntax work.
func Text_Unvalidated_Invariants(value Text_Unvalidated, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// URN is complete RFC 2141 UUID name.
type URN string

// URN_Invariants fixes prefix and UUID text width.
func URN_Invariants(value URN, _ aver.Namespace) {
	aver.Always(len(value) == UUID_URN_BYTE_COUNT, "UUID URN has RFC 2141 width.")
}

// Strings is caller-independent rendering of bounded UUID collection.
type Strings []string

// Strings_Invariants preserves collection count bound.
func Strings_Invariants(value Strings, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, UUIDS_COUNT_MAXIMUM).
		Ensure()
}

// Unix_Second is complete seconds relative to Unix epoch.
type Unix_Second int64

// Unix_Second_Invariants bounds conversion from RFC 9562 timestamp.
func Unix_Second_Invariants(value Unix_Second, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), UNIX_SECOND_MINIMUM, UNIX_SECOND_MAXIMUM).
		Ensure()
}

// NANOSECOND_STORAGE_COUNT keeps exact remainder in one scalar slot.
const NANOSECOND_STORAGE_COUNT = UUID_BIT_COUNT / UUID_BIT_COUNT

// Nanosecond is subsecond remainder from RFC timestamp conversion. Exact UUID
// precision makes only multiples of one 100-nanosecond tick representable.
type Nanosecond [NANOSECOND_STORAGE_COUNT]int64

// Nanosecond_Invariants states bounds and resolution without claiming impossible values.
func Nanosecond_Invariants(value Nanosecond, _ aver.Namespace) {
	aver.Always(
		value[0] >= NANOSECOND_MINIMUM,
		"UUID Unix nanoseconds stay above signed subsecond minimum.",
	)
	aver.Always(
		value[0] <= NANOSECOND_MAXIMUM,
		"UUID Unix nanoseconds stay below signed subsecond maximum.",
	)
	aver.Always(
		value[0]%UUID_TICK_NANOSECOND_COUNT == 0,
		"UUID Unix nanoseconds retain exact RFC tick resolution.",
	)
}

// Order is normalized lexical UUID comparison.
type Order int

// Order_Invariants admits before, equal, and after.
func Order_Invariants(value Order, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Int(int(value), ORDER_BEFORE, ORDER_EQUAL, ORDER_AFTER).
		Ensure()
}

// Unix_Millisecond is Version 7 wall-clock field.
type Unix_Millisecond uint64

// Unix_Millisecond_Invariants bounds Version 7 48-bit millisecond field.
func Unix_Millisecond_Invariants(value Unix_Millisecond, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), UNIX_MILLISECOND_MINIMUM, UNIX_MILLISECOND_MAXIMUM).
		Ensure()
}

// Generated_Unix_Millisecond is Version 7 subset reachable from signed Clock.
type Generated_Unix_Millisecond uint64

// Generated_Unix_Millisecond_Invariants matches nonnegative Clock image.
func Generated_Unix_Millisecond_Invariants(
	value Generated_Unix_Millisecond, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(
			uint64(value), GENERATED_UNIX_MILLISECOND_MINIMUM,
			GENERATED_UNIX_MILLISECOND_MAXIMUM,
		).
		Ensure()
}

// V7_Sequence orders draws within one Version 7 millisecond.
type V7_Sequence uint16

// V7_Sequence_Invariants bounds Version 7 12-bit sequence field.
func V7_Sequence_Invariants(value V7_Sequence, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), V7_SEQUENCE_MINIMUM, V7_SEQUENCE_MAXIMUM).
		Ensure()
}

// Clock_Sequence_State stores zero before seed and seeded sequence plus one afterward.
type Clock_Sequence_State uint16

// Clock_Sequence_State_Invariants bounds internal optional sequence encoding.
func Clock_Sequence_State_Invariants(
	value Clock_Sequence_State, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint16(
			uint16(value), bits.WORD_16_MINIMUM, CLOCK_SEQUENCE_VALUE_COUNT,
		).
		Ensure()
}

// Timestamp_State is last timestamp retained for clock-regression detection.
type Timestamp_State int64

// Timestamp_State_Invariants bounds internal timestamp storage.
func Timestamp_State_Invariants(value Timestamp_State, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), TIME_MINIMUM, TIME_MAXIMUM).
		Ensure()
}

// V7_State is last combined Version 7 ordering value.
type V7_State uint64

// V7_State_Invariants bounds internal Version 7 ordering storage.
func V7_State_Invariants(value V7_State, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), V7_VALUE_MINIMUM, V7_VALUE_MAXIMUM).
		Ensure()
}

// Generator_State holds caller-selected ordering state, zero for fresh generator.
type Generator_State struct {
	// Clock_Sequence stores zero before seed and seeded sequence plus one afterward.
	Clock_Sequence Clock_Sequence_State
	// Last_Time retains last Version 1 or Version 6 timestamp.
	Last_Time Timestamp_State
	// Last_V7 retains last combined Version 7 ordering value.
	Last_V7 V7_State
}

// Generator_State_Invariants composes caller-selected ordering state.
func Generator_State_Invariants(value Generator_State, namespace aver.Namespace) {
	Clock_Sequence_State_Invariants(value.Clock_Sequence, namespace)
	Timestamp_State_Invariants(value.Last_Time, namespace)
	V7_State_Invariants(value.Last_V7, namespace)
}

// VARIANT_INVALID marks a variant that matches no known scheme.
const VARIANT_INVALID Variant = 0

// VARIANT_RFC_4122 is the RFC 9562 (formerly RFC 4122) variant every version here produces.
const VARIANT_RFC_4122 Variant = 1

// VARIANT_RESERVED is the reserved NCS-backward-compatibility variant.
const VARIANT_RESERVED Variant = 2

// VARIANT_MICROSOFT is the reserved Microsoft-backward-compatibility variant.
const VARIANT_MICROSOFT Variant = 3

// VARIANT_FUTURE is reserved for future definition.
const VARIANT_FUTURE Variant = 4

// DOMAIN_PERSON is the person domain; on POSIX the identifier is a user id.
const DOMAIN_PERSON Domain = 0

// DOMAIN_GROUP is the group domain; on POSIX the identifier is a group id.
const DOMAIN_GROUP Domain = 1

// DOMAIN_ORGANIZATION is the organization domain; the identifier's meaning is site-defined.
const DOMAIN_ORGANIZATION Domain = 2

// TIME_MINIMUM begins RFC 9562 timestamp field.
const TIME_MINIMUM int64 = bits.INTEGER_64_MINIMUM - bits.INTEGER_64_MINIMUM

// TIME_VALUE_COUNT is count of values representable by 60-bit timestamp field.
const TIME_VALUE_COUNT = int64(1) << (UUID_BIT_COUNT -
	bits.BIT_COUNT_64_MAXIMUM - VERSION_BIT_COUNT)

// TIME_MAXIMUM closes RFC 9562 timestamp field.
const TIME_MAXIMUM = TIME_VALUE_COUNT - 1

// GENERATED_TIME_MINIMUM is earliest RFC tick reachable from signed Clock.
const GENERATED_TIME_MINIMUM = bits.INTEGER_64_MINIMUM/
	UUID_TICK_NANOSECOND_COUNT + EPOCH_100NS

// GENERATED_TIME_MAXIMUM is latest RFC tick reachable from signed Clock.
const GENERATED_TIME_MAXIMUM = bits.INTEGER_64_MAXIMUM/
	UUID_TICK_NANOSECOND_COUNT + EPOCH_100NS

// V7_SEQUENCE_BIT_COUNT is random-a field width used for monotonic sequence.
const V7_SEQUENCE_BIT_COUNT = bits.BIT_COUNT_16_MAXIMUM - VERSION_BIT_COUNT

// V7_SEQUENCE_VALUE_COUNT is count of values representable by Version 7 sequence.
const V7_SEQUENCE_VALUE_COUNT = 1 << V7_SEQUENCE_BIT_COUNT

// V7_SEQUENCE_MINIMUM begins Version 7 sequence field.
const V7_SEQUENCE_MINIMUM uint16 = bits.WORD_16_MINIMUM

// V7_SEQUENCE_MAXIMUM closes Version 7 sequence field.
const V7_SEQUENCE_MAXIMUM = V7_SEQUENCE_VALUE_COUNT - 1

// UNIX_MILLISECOND_BIT_COUNT is Version 7 leading timestamp field width.
const UNIX_MILLISECOND_BIT_COUNT = NODE_BIT_COUNT

// UNIX_MILLISECOND_VALUE_COUNT is count of Version 7 millisecond values.
const UNIX_MILLISECOND_VALUE_COUNT = uint64(1) << UNIX_MILLISECOND_BIT_COUNT

// UNIX_MILLISECOND_MINIMUM begins Version 7 millisecond field.
const UNIX_MILLISECOND_MINIMUM uint64 = bits.WORD_64_MINIMUM

// UNIX_MILLISECOND_MAXIMUM closes Version 7 millisecond field.
const UNIX_MILLISECOND_MAXIMUM = UNIX_MILLISECOND_VALUE_COUNT - 1

// GENERATED_UNIX_MILLISECOND_MINIMUM is zero because Version 7 rejects pre-Unix clocks.
const GENERATED_UNIX_MILLISECOND_MINIMUM uint64 = bits.WORD_64_MINIMUM

// GENERATED_UNIX_MILLISECOND_MAXIMUM is final millisecond reachable from signed Clock.
const GENERATED_UNIX_MILLISECOND_MAXIMUM = uint64(bits.INTEGER_64_MAXIMUM) /
	uint64(NANO_PER_MILLI)

// V7_VALUE_MINIMUM begins combined ordering state.
const V7_VALUE_MINIMUM uint64 = bits.WORD_64_MINIMUM

// V7_VALUE_MAXIMUM combines maximum milliseconds and sub-millisecond sequence.
const V7_VALUE_MAXIMUM = UNIX_MILLISECOND_MAXIMUM<<V7_SEQUENCE_BIT_COUNT |
	uint64(V7_SEQUENCE_MAXIMUM)

// ORDER_BEFORE reports first UUID sorts before second.
const ORDER_BEFORE = -1

// ORDER_EQUAL reports equal UUIDs.
const ORDER_EQUAL = 0

// ORDER_AFTER reports first UUID sorts after second.
const ORDER_AFTER = 1

// VERSION_MD5 is name-based UUID version using MD5.
const VERSION_MD5 = 3

// VERSION_SHA1 is name-based UUID version using SHA-1.
const VERSION_SHA1 = 5

// JULIAN_1582 is the Julian day number of 15 Oct 1582, the RFC 9562 epoch.
const JULIAN_1582 = 2299160

// JULIAN_1970 is the Julian day number of 1 Jan 1970, the Unix epoch.
const JULIAN_1970 = 2440587

// EPOCH_DAYS is the day count between the RFC 9562 and Unix epochs.
const EPOCH_DAYS = JULIAN_1970 - JULIAN_1582

// EPOCH_SECONDS is second count between two epochs.
const EPOCH_SECONDS = EPOCH_DAYS * int64(time.DAY/time.SECOND)

// UUID_TICK_NANOSECOND_COUNT is RFC timestamp resolution.
const UUID_TICK_NANOSECOND_COUNT = 100

// UUID_TICK_COUNT_PER_SECOND converts seconds to RFC timestamp ticks.
const UUID_TICK_COUNT_PER_SECOND = int64(time.SECOND/time.NANOSECOND) /
	UUID_TICK_NANOSECOND_COUNT

// EPOCH_100NS is the 100-nanosecond-tick count between the two epochs, the constant
// that rebases a Unix time onto the RFC 9562 timeline.
const EPOCH_100NS = EPOCH_SECONDS * UUID_TICK_COUNT_PER_SECOND

// NANO_PER_MILLI is nanoseconds per millisecond, the Version 7 timestamp unit.
const NANO_PER_MILLI = int64(time.MILLISECOND / time.NANOSECOND)

// UNIX_SECOND_MINIMUM is earliest UUID time converted to Unix epoch.
const UNIX_SECOND_MINIMUM = -EPOCH_SECONDS

// UNIX_SECOND_MAXIMUM is latest UUID time converted to Unix epoch.
const UNIX_SECOND_MAXIMUM = (TIME_MAXIMUM - EPOCH_100NS) / UUID_TICK_COUNT_PER_SECOND

// NANOSECOND_MAXIMUM is final complete RFC tick within one second.
const NANOSECOND_MAXIMUM = (UUID_TICK_COUNT_PER_SECOND - 1) * UUID_TICK_NANOSECOND_COUNT

// NANOSECOND_MINIMUM mirrors negative Go remainder before Unix epoch.
const NANOSECOND_MINIMUM = -NANOSECOND_MAXIMUM

// JSON_NULL is JSON spelling for absent nullable UUID.
const JSON_NULL = "null"

// Error is bounded UUID failure kind. Static text needs no allocated diagnostic wrapper.
type Error uint8

// Error_Invariants closes every UUID failure kind.
func Error_Invariants(value Error, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(ERROR_INVALID_FORMAT),
			uint8(ERROR_V7_TIME_OUTPUT_OF_RANGE),
		).
		Ensure()
}

// ERROR_INVALID_FORMAT reports text that is not any accepted UUID form.
const ERROR_INVALID_FORMAT Error = Error(bits.WORD_8_MINIMUM)

// ERROR_INVALID_BRACKETED_FORMAT reports brace text missing one required brace.
const ERROR_INVALID_BRACKETED_FORMAT Error = ERROR_INVALID_FORMAT + 1

// ERROR_INVALID_SIZE reports input whose byte count matches no accepted form.
const ERROR_INVALID_SIZE Error = ERROR_INVALID_BRACKETED_FORMAT + 1

// ERROR_INVALID_URN_PREFIX reports 45-byte text without required UUID URN prefix.
const ERROR_INVALID_URN_PREFIX Error = ERROR_INVALID_SIZE + 1

// ERROR_V7_STATE_EXHAUSTED reports no greater Version 7 value remains.
const ERROR_V7_STATE_EXHAUSTED Error = ERROR_INVALID_URN_PREFIX + 1

// ERROR_V7_TIME_OUTPUT_OF_RANGE reports clock time RFC Version 7 cannot encode.
const ERROR_V7_TIME_OUTPUT_OF_RANGE Error = ERROR_V7_STATE_EXHAUSTED + 1

// Error implements error with static bounded diagnostics.
func (value Error) Error() (text string) {
	Error_Invariants(value, "error.value")
	switch value {
	case ERROR_INVALID_FORMAT:
		return "uuid: invalid format"
	case ERROR_INVALID_BRACKETED_FORMAT:
		return "uuid: invalid bracketed format"
	case ERROR_INVALID_SIZE:
		return "uuid: invalid length"
	case ERROR_INVALID_URN_PREFIX:
		return "uuid: invalid urn prefix"
	case ERROR_V7_STATE_EXHAUSTED:
		return "uuid: version 7 state exhausted"
	case ERROR_V7_TIME_OUTPUT_OF_RANGE:
		return "uuid: version 7 time out of range"
	}
	return "uuid: invalid error"
}

// Generator holds the injected sources the upstream package kept as mutable package
// globals. A zero Generator is unusable; construct one with New. It is not safe for
// concurrent use — the clock-sequence and monotonic-time fields are mutated on each
// timestamped draw, so give each goroutine its own.
type Generator struct {
	// Source supplies entropy for random fields and a random node. Caller owns state.
	Source prng.Source
	// Clock supplies wall-clock time for the timestamp fields of V1, V6, and V7.
	Clock time.Clock
	// Node is the 6-byte node identifier embedded in V1 and V6. A zero value draws a
	// random node from Source on first use.
	Node Node
	// Clock_Sequence is the V1/V6 sequence counter, bumped when the clock repeats or
	// regresses so same-instant UUIDs still differ.
	Clock_Sequence Clock_Sequence_State
	// Last_Time is the last V1/V6 timestamp (100ns units) handed out, for regression detection.
	Last_Time Timestamp_State
	// Last_V7 is the last V7 value (milliseconds<<12 | sub-millisecond sequence) handed out.
	Last_V7 V7_State
}

// Generator_Invariants composes every injected dependency and mutable ordering field.
func Generator_Invariants(value Generator, namespace aver.Namespace) {
	prng.Source_Invariants(value.Source, namespace)
	time.Clock_Invariants(value.Clock, namespace)
	Node_Invariants(value.Node, namespace)
	Clock_Sequence_State_Invariants(value.Clock_Sequence, namespace)
	Timestamp_State_Invariants(value.Last_Time, namespace)
	V7_State_Invariants(value.Last_V7, namespace)
}

// Null_UUID is a UUID that may be SQL NULL, the scan destination for a nullable column.
type Null_UUID struct {
	// UUID is the value, meaningful only when Valid.
	UUID UUID
	// Valid is true when UUID holds a non-NULL value.
	Valid Valid
}

// Null_UUID_Invariants composes nullable value and presence state.
func Null_UUID_Invariants(value Null_UUID, namespace aver.Namespace) {
	UUID_Invariants(value.UUID, namespace)
	Valid_Invariants(value.Valid, namespace)
}

// New builds Generator from injected dependencies and explicit replay state.
func New(
	source prng.Source, host time.Clock, node Node, state Generator_State,
) (generator Generator) {
	defer func() { Generator_Invariants(generator, "new.generator") }()
	prng.Source_Invariants(source, "new.source")
	time.Clock_Invariants(host, "new.host")
	Node_Invariants(node, "new.node")
	Generator_State_Invariants(state, "new.state")
	generator.Source = source
	generator.Clock = host
	generator.Node = node
	generator.Clock_Sequence = state.Clock_Sequence
	generator.Last_Time = state.Last_Time
	generator.Last_V7 = state.Last_V7
	return generator
}

// Nil is the zero UUID, all 128 bits clear. It is a function, not a package var,
// because a UUID is an array literal and the house linter bans mutable package state.
func Nil() (uuid UUID) {
	defer func() { UUID_Invariants(uuid, "nil.uuid") }()
	return UUID{}
}

// Max is the RFC 9562 maximum UUID, all 128 bits set.
func Max() (uuid UUID) {
	defer func() { UUID_Invariants(uuid, "max.uuid") }()
	return UUID{
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	}
}

// Name_Space_DNS is the RFC 9562 namespace for domain names, used as the V3/V5 space.
func Name_Space_DNS() (uuid UUID) {
	defer func() { UUID_Invariants(uuid, "name_space_dns.uuid") }()
	return Must_Parse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
}

// Name_Space_URL is the RFC 9562 namespace for URLs, used as the V3/V5 space.
func Name_Space_URL() (uuid UUID) {
	defer func() { UUID_Invariants(uuid, "name_space_url.uuid") }()
	return Must_Parse("6ba7b811-9dad-11d1-80b4-00c04fd430c8")
}

// Name_Space_OID is the RFC 9562 namespace for ISO OIDs, used as the V3/V5 space.
func Name_Space_OID() (uuid UUID) {
	defer func() { UUID_Invariants(uuid, "name_space_oid.uuid") }()
	return Must_Parse("6ba7b812-9dad-11d1-80b4-00c04fd430c8")
}

// Name_Space_X500 is the RFC 9562 namespace for X.500 DNs, used as the V3/V5 space.
func Name_Space_X500() (uuid UUID) {
	defer func() { UUID_Invariants(uuid, "name_space_x500.uuid") }()
	return Must_Parse("6ba7b814-9dad-11d1-80b4-00c04fd430c8")
}

// Generator_V4 returns a random (Version 4) UUID, 122 bits drawn from Source.
func Generator_V4(generator *Generator) (uuid UUID, err error) {
	defer func() { UUID_Invariants(uuid, "generator_v4.uuid") }()
	Generator_Invariants(*generator, "generator_v4.generator")
	prng.Source_Read(generator.Source, prng.Sink(uuid[:]))
	uuid[6] = uuid[6]&0x0f | 0x40 // Version 4.
	uuid[8] = uuid[8]&0x3f | 0x80 // RFC 9562 variant.
	return uuid, nil
}

// Generator_V7 returns a time-ordered (Version 7) UUID: 48 bits of Unix
// milliseconds, then a sub-millisecond sequence, then random bits. The value is
// strictly greater than any previous V7 from this Generator, so a burst within one
// millisecond still sorts in creation order.
func Generator_V7(generator *Generator) (uuid UUID, err error) {
	defer func() { UUID_Invariants(uuid, "generator_v7.uuid") }()
	Generator_Invariants(*generator, "generator_v7.generator")
	uuid, read_err := Generator_V4(generator)
	if read_err != nil {
		return Nil(), read_err
	}
	milliseconds, sequence, time_err := generator_v7_time(
		generator.Clock, &generator.Last_V7,
	)
	if time_err != nil {
		return Nil(), time_err
	}
	uuid[0] = byte(milliseconds >> 40)
	uuid[1] = byte(milliseconds >> 32)
	uuid[2] = byte(milliseconds >> 24)
	uuid[3] = byte(milliseconds >> 16)
	uuid[4] = byte(milliseconds >> 8)
	uuid[5] = byte(milliseconds)
	uuid[6] = 0x70 | 0x0f&byte(sequence>>8) // Version 7 over the high sequence nibble.
	uuid[7] = byte(sequence)
	return uuid, nil
}

// Generator_V1 returns a time-and-node (Version 1) UUID.
func Generator_V1(generator *Generator) (uuid UUID, err error) {
	defer func() { UUID_Invariants(uuid, "generator_v1.uuid") }()
	Generator_Invariants(*generator, "generator_v1.generator")
	timestamp, sequence := generator_time(
		generator.Clock, generator.Source, &generator.Clock_Sequence, &generator.Last_Time,
	)
	node := generator_node(generator.Source, generator.Node)
	generator.Node = node
	time_low := uint32(timestamp & 0xffffffff)
	time_middle := uint16((timestamp >> 32) & 0xffff)
	time_high := uint16((timestamp >> 48) & 0x0fff)
	time_high = time_high | 0x1000 // Version 1.
	binary.Put_Uint_32(binary.Bytes(uuid[0:]), binary.Word_32(time_low), binary.BIG_ENDIAN)
	binary.Put_Uint_16(binary.Bytes(uuid[4:]), binary.Word_16(time_middle), binary.BIG_ENDIAN)
	binary.Put_Uint_16(binary.Bytes(uuid[6:]), binary.Word_16(time_high), binary.BIG_ENDIAN)
	binary.Put_Uint_16(
		binary.Bytes(uuid[8:]), binary.Word_16(sequence)|0x8000, binary.BIG_ENDIAN,
	)
	copy(uuid[10:], node[:])
	return uuid, nil
}

// Generator_V6 returns a Version 6 UUID: the V1 fields reordered most-significant
// first so that lexical order tracks time order, better for database locality.
func Generator_V6(generator *Generator) (uuid UUID, err error) {
	defer func() { UUID_Invariants(uuid, "generator_v6.uuid") }()
	Generator_Invariants(*generator, "generator_v6.generator")
	timestamp, sequence := generator_time(
		generator.Clock, generator.Source, &generator.Clock_Sequence, &generator.Last_Time,
	)
	node := generator_node(generator.Source, generator.Node)
	generator.Node = node
	time_high := uint32((timestamp >> 28) & 0xffffffff)
	time_middle := uint16((timestamp >> 12) & 0xffff)
	time_low := uint16(timestamp & 0x0fff)
	time_low = time_low | 0x6000 // Version 6.
	binary.Put_Uint_32(binary.Bytes(uuid[0:]), binary.Word_32(time_high), binary.BIG_ENDIAN)
	binary.Put_Uint_16(binary.Bytes(uuid[4:]), binary.Word_16(time_middle), binary.BIG_ENDIAN)
	binary.Put_Uint_16(binary.Bytes(uuid[6:]), binary.Word_16(time_low), binary.BIG_ENDIAN)
	binary.Put_Uint_16(
		binary.Bytes(uuid[8:]), binary.Word_16(sequence)|0x8000, binary.BIG_ENDIAN,
	)
	copy(uuid[10:], node[:])
	return uuid, nil
}

// Generator_DCE_Security returns a DCE Security (Version 2) UUID that embeds domain
// and identifier in the low fields of a Version 1 layout. The identifier is a user
// or group id the caller supplies (the host's os.Getuid/os.Getgid), kept out of this
// pure package so it imports no operating-system state.
func Generator_DCE_Security(
	generator *Generator, domain Domain, identifier Identifier,
) (uuid UUID, err error) {
	defer func() { UUID_Invariants(uuid, "generator_dce_security.uuid") }()
	Generator_Invariants(*generator, "generator_dce_security.generator")
	Domain_Invariants(domain, "generator_dce_security.domain")
	Identifier_Invariants(identifier, "generator_dce_security.identifier")
	uuid, v1_err := Generator_V1(generator)
	if v1_err != nil {
		return Nil(), v1_err
	}
	uuid[6] = uuid[6]&0x0f | 0x20 // Version 2.
	uuid[9] = byte(domain)
	binary.Put_Uint_32(binary.Bytes(uuid[0:]), binary.Word_32(identifier), binary.BIG_ENDIAN)
	return uuid, nil
}

// V3 returns a name-based MD5 (Version 3) UUID: the deterministic hash of namespace
// concatenated with data. The same inputs always yield the same UUID, so V3 needs
// no Generator.
func V3(namespace UUID, name Name) (uuid UUID) {
	defer func() { UUID_Invariants(uuid, "v3.uuid") }()
	UUID_Invariants(namespace, "v3.namespace")
	Name_Invariants(name, "v3.name")
	var digest md5.Digest
	md5.Digest_Init(&digest)
	md5.Digest_Write(&digest, md5.Source(namespace[:]))
	md5.Digest_Write(&digest, md5.Source(name))
	value := md5.Digest_Sum(&digest)
	copy(uuid[:], value[:])
	uuid[6] = uuid[6]&0x0f | VERSION_MD5<<4
	uuid[8] = uuid[8]&0x3f | 0x80 // RFC 9562 variant.
	return uuid
}

// V5 returns a name-based SHA-1 (Version 5) UUID, the SHA-1 counterpart of V3 and
// the RFC-preferred of the two name-based versions.
func V5(namespace UUID, name Name) (uuid UUID) {
	defer func() { UUID_Invariants(uuid, "v5.uuid") }()
	UUID_Invariants(namespace, "v5.namespace")
	Name_Invariants(name, "v5.name")
	var digest sha1.Digest
	sha1.Digest_Init(&digest)
	sha1.Digest_Write(&digest, sha1.Source(namespace[:]))
	sha1.Digest_Write(&digest, sha1.Source(name))
	value := sha1.Digest_Sum(&digest)
	copy(uuid[:], value[:UUID_BYTE_COUNT])
	uuid[6] = uuid[6]&0x0f | VERSION_SHA1<<4
	uuid[8] = uuid[8]&0x3f | 0x80 // RFC 9562 variant.
	return uuid
}

// Returns the current time as RFC 9562 100-nanosecond ticks since 15 Oct 1582, with
// the clock sequence, advancing the sequence if the clock did not move forward so
// successive UUIDs stay distinct and ordered.
func generator_time(
	host time.Clock, source prng.Source, clock_sequence *Clock_Sequence_State,
	last_time *Timestamp_State,
) (timestamp Generated_Time, sequence Clock_Sequence) {
	defer func() {
		Generated_Time_Invariants(timestamp, "generator_time.timestamp")
		Clock_Sequence_Invariants(sequence, "generator_time.sequence")
	}()
	time.Clock_Invariants(host, "generator_time.host")
	prng.Source_Invariants(source, "generator_time.source")
	Clock_Sequence_State_Invariants(*clock_sequence, "generator_time.clock_sequence")
	Timestamp_State_Invariants(*last_time, "generator_time.last_time")
	if *clock_sequence == 0 {
		var raw [CLOCK_SEQUENCE_BYTE_COUNT]byte
		prng.Source_Read(source, prng.Sink(raw[:]))
		sequence_bits := uint16(raw[0])<<bits.BIT_COUNT_8_MAXIMUM | uint16(raw[1])
		*clock_sequence = Clock_Sequence_State(
			Clock_Sequence(sequence_bits&CLOCK_SEQUENCE_MAXIMUM) + 1,
		)
	}
	now := Generated_Time(int64(time.Clock_Now_Realtime(host))/
		UUID_TICK_NANOSECOND_COUNT + EPOCH_100NS)
	if Time(now) <= Time(*last_time) {
		sequence = Clock_Sequence(*clock_sequence - 1)
		sequence = (sequence + 1) & CLOCK_SEQUENCE_MAXIMUM
		*clock_sequence = Clock_Sequence_State(sequence + 1)
	}
	*last_time = Timestamp_State(now)
	return now, Clock_Sequence(*clock_sequence - 1)
}

// Returns Unix milliseconds and a sub-millisecond sequence, forced strictly upward
// past the last V7 draw so a same-millisecond burst still orders by creation.
func generator_v7_time(
	host time.Clock, last_v7 *V7_State,
) (milliseconds Generated_Unix_Millisecond, sequence V7_Sequence, err error) {
	defer func() {
		Generated_Unix_Millisecond_Invariants(
			milliseconds, "generator_v7_time.milliseconds",
		)
		V7_Sequence_Invariants(sequence, "generator_v7_time.sequence")
	}()
	time.Clock_Invariants(host, "generator_v7_time.host")
	V7_State_Invariants(*last_v7, "generator_v7_time.last_v7")
	nanoseconds := int64(time.Clock_Now_Realtime(host))
	if nanoseconds < 0 {
		return 0, 0, ERROR_V7_TIME_OUTPUT_OF_RANGE
	}
	milliseconds = Generated_Unix_Millisecond(nanoseconds / NANO_PER_MILLI)
	sequence = V7_Sequence(
		(nanoseconds - int64(milliseconds)*NANO_PER_MILLI) >> bits.BIT_COUNT_8_MAXIMUM,
	)
	combined := V7_State(uint64(milliseconds)<<V7_SEQUENCE_BIT_COUNT | uint64(sequence))
	if combined <= *last_v7 {
		if uint64(*last_v7) == V7_VALUE_MAXIMUM {
			return 0, 0, ERROR_V7_STATE_EXHAUSTED
		}
		combined = *last_v7 + 1
		milliseconds = Generated_Unix_Millisecond(combined >> V7_SEQUENCE_BIT_COUNT)
		sequence = V7_Sequence(combined & V7_SEQUENCE_MAXIMUM)
	}
	*last_v7 = combined
	return milliseconds, sequence, nil
}

// Resolves the node used for V1 and V6: the injected Node when set, otherwise a
// one-time random draw from Source with the multicast bit set to mark it as not a
// real hardware address.
func generator_node(source prng.Source, node Node) (resolved Node) {
	defer func() { Node_Invariants(resolved, "generator_node.resolved") }()
	prng.Source_Invariants(source, "generator_node.source")
	Node_Invariants(node, "generator_node.node")
	if node != (Node{}) {
		return node
	}
	prng.Source_Read(source, prng.Sink(node[:]))
	node[0] = node[0] | 0x01 // Multicast bit: not a real MAC.
	return node
}

// Parse decodes s into a UUID. It accepts the canonical hyphenated form, the
// urn:uuid: prefixed form, a single-brace-wrapped form, and the 32-character
// unhyphenated form; any other length is an error. Parse is lenient by design — use
// Validate to reject non-canonical encodings.
func Parse(input Text_Unvalidated) (uuid UUID, err error) {
	defer func() { UUID_Invariants(uuid, "parse.uuid") }()
	Text_Unvalidated_Invariants(input, "parse.input")
	body := input
	canonical := false
	switch len(input) {
	case UUID_TEXT_BYTE_COUNT:
		canonical = true
	case UUID_URN_BYTE_COUNT:
		prefix := strings.Text(input[:UUID_URN_PREFIX_BYTE_COUNT])
		if !bool(strings.Equal_Fold(prefix, UUID_URN_PREFIX)) {
			return uuid, ERROR_INVALID_URN_PREFIX
		}
		body = input[UUID_URN_PREFIX_BYTE_COUNT:]
		canonical = true
	case UUID_BRACED_BYTE_COUNT:
		if input[bytes.SLICE_SIZE_MINIMUM] != '{' {
			return uuid, ERROR_INVALID_BRACKETED_FORMAT
		}
		if input[UUID_BRACED_BYTE_COUNT-1] != '}' {
			return uuid, ERROR_INVALID_BRACKETED_FORMAT
		}
		body = input[UUID_BRACE_BYTE_COUNT/2 : UUID_BRACED_BYTE_COUNT-1]
		canonical = true
	case UUID_HEXADECIMAL_BYTE_COUNT:
	default:
		return uuid, ERROR_INVALID_SIZE
	}
	if canonical {
		if body[UUID_TEXT_FIRST_HYPHEN_INDEX] != '-' {
			return uuid, ERROR_INVALID_FORMAT
		}
		if body[UUID_TEXT_SECOND_HYPHEN_INDEX] != '-' {
			return uuid, ERROR_INVALID_FORMAT
		}
		if body[UUID_TEXT_THIRD_HYPHEN_INDEX] != '-' {
			return uuid, ERROR_INVALID_FORMAT
		}
		if body[UUID_TEXT_FINAL_HYPHEN_INDEX] != '-' {
			return uuid, ERROR_INVALID_FORMAT
		}
	}
	var encoded [UUID_HEXADECIMAL_BYTE_COUNT]byte
	encoded_position := bytes.SLICE_SIZE_MINIMUM
	source_position := bytes.SLICE_SIZE_MINIMUM
	for source_position < len(body) {
		if canonical {
			if body[source_position] == '-' {
				source_position++
				continue
			}
		}
		encoded[encoded_position] = body[source_position]
		encoded_position++
		source_position++
	}
	if encoded_position != len(encoded) {
		return uuid, ERROR_INVALID_FORMAT
	}
	count, status := hex.Decode_Into(hex.Decoded(uuid[:]), hex.Encoded(encoded[:]))
	var status_ok hex.Decode_Status
	if status != status_ok {
		return UUID{}, ERROR_INVALID_FORMAT
	}
	if int(count) != UUID_BYTE_COUNT {
		return UUID{}, ERROR_INVALID_FORMAT
	}
	return uuid, nil
}

// Parse_Bytes is Parse over a byte slice. It delegates through a string conversion
// rather than duplicating the format logic; a UUID is 45 bytes at most, so the copy
// is negligible.
func Parse_Bytes(input bytes.Slice) (uuid UUID, err error) {
	defer func() { UUID_Invariants(uuid, "parse_bytes.uuid") }()
	bytes.Slice_Invariants(input, "parse_bytes.input")
	return Parse(Text_Unvalidated(string(input)))
}

// Validate reports whether s is a UUID Parse would accept, discarding the value.
func Validate(input Text_Unvalidated) (err error) {
	Text_Unvalidated_Invariants(input, "validate.input")
	_, parse_err := Parse(input)
	return parse_err
}

// Must_Parse is Parse but panics on error, for compile-time-constant UUIDs where a
// parse failure is a programming error, not a runtime condition.
func Must_Parse(input Text_Unvalidated) (uuid UUID) {
	defer func() { UUID_Invariants(uuid, "must_parse.uuid") }()
	Text_Unvalidated_Invariants(input, "must_parse.input")
	parsed, err := Parse(input)
	if err != nil {
		panic(err)
	}
	return parsed
}

// Must unwraps a (UUID, error) pair, panicking on error. It wraps a generator call
// whose only error is a starved entropy reader — a condition production treats as fatal.
func Must(uuid UUID, err error) (result UUID) {
	defer func() { UUID_Invariants(result, "must.result") }()
	UUID_Invariants(uuid, "must.uuid")
	if err != nil {
		panic(err)
	}
	return uuid
}

// From_Bytes builds a UUID from exactly 16 raw bytes, which it copies.
func From_Bytes(input bytes.Slice) (uuid UUID, err error) {
	defer func() { UUID_Invariants(uuid, "from_bytes.uuid") }()
	bytes.Slice_Invariants(input, "from_bytes.input")
	if len(input) != UUID_BYTE_COUNT {
		return uuid, ERROR_INVALID_SIZE
	}
	copy(uuid[:], input)
	return uuid, nil
}

// String returns the canonical 36-character form xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx,
// satisfying fmt.Stringer.
func (uuid UUID) String() (text string) {
	var buffer [UUID_TEXT_BYTE_COUNT]byte
	encode_hexadecimal(&buffer, uuid)
	return string(buffer[:])
}

// UUID_URN returns the RFC 2141 URN form urn:uuid:xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.
func UUID_URN(uuid UUID) (urn URN) {
	defer func() { URN_Invariants(urn, "uuid_urn.urn") }()
	UUID_Invariants(uuid, "uuid_urn.uuid")
	var buffer [UUID_URN_BYTE_COUNT]byte
	var encoded [UUID_TEXT_BYTE_COUNT]byte
	copy(buffer[:], UUID_URN_PREFIX)
	encode_hexadecimal(&encoded, uuid)
	copy(buffer[UUID_URN_PREFIX_BYTE_COUNT:], encoded[:])
	return URN(buffer[:])
}

// Exact storage keeps fixed UUID text from claiming the shared encoder's broader
// size domain, whose boundaries can never occur here.
func encode_hexadecimal(destination *[UUID_TEXT_BYTE_COUNT]byte, uuid UUID) {
	UUID_Invariants(uuid, "encode_hexadecimal.uuid")
	destination_position := 0
	for _, source_byte := range uuid {
		switch destination_position {
		case UUID_TEXT_FIRST_HYPHEN_INDEX, UUID_TEXT_SECOND_HYPHEN_INDEX,
			UUID_TEXT_THIRD_HYPHEN_INDEX, UUID_TEXT_FINAL_HYPHEN_INDEX:
			destination[destination_position] = '-'
			destination_position++
		}
		first_nibble := source_byte >> hex.NIBBLE_BIT_COUNT
		destination[destination_position] = hex.ENCODE_ALPHABET[first_nibble]
		final_position := destination_position + hex.ENCODED_BYTE_FINAL_OFFSET
		destination[final_position] =
			hex.ENCODE_ALPHABET[source_byte&hex.ALPHABET_FINAL_INDEX]
		destination_position += hex.ENCODED_BYTE_SIZE
	}
}

// UUID_Version returns the version encoded in uuid.
func UUID_Version(uuid UUID) (version Version) {
	defer func() { Version_Invariants(version, "uuid_version.version") }()
	UUID_Invariants(uuid, "uuid_version.uuid")
	return Version(uuid[6] >> 4)
}

// UUID_Variant returns the layout variant encoded in uuid.
func UUID_Variant(uuid UUID) (variant Variant) {
	defer func() { Variant_Invariants(variant, "uuid_variant.variant") }()
	UUID_Invariants(uuid, "uuid_variant.uuid")
	if uuid[8]&0xc0 == 0x80 {
		return VARIANT_RFC_4122
	}
	if uuid[8]&0xe0 == 0xc0 {
		return VARIANT_MICROSOFT
	}
	if uuid[8]&0xe0 == 0xe0 {
		return VARIANT_FUTURE
	}
	return VARIANT_RESERVED
}

// UUID_Node_Identifier returns a copy of the 6-byte node field, well defined only for V1 and V2.
func UUID_Node_Identifier(uuid UUID) (node Node) {
	defer func() { Node_Invariants(node, "uuid_node_identifier.node") }()
	UUID_Invariants(uuid, "uuid_node_identifier.uuid")
	var copied Node
	copy(copied[:], uuid[10:])
	return copied
}

// UUID_Domain returns the domain of a Version 2 UUID.
func UUID_Domain(uuid UUID) (domain Domain) {
	defer func() { Domain_Invariants(domain, "uuid_domain.domain") }()
	UUID_Invariants(uuid, "uuid_domain.uuid")
	return Domain(uuid[9])
}

// UUID_Identifier returns the embedded identifier of a Version 2 UUID.
func UUID_Identifier(uuid UUID) (identifier Identifier) {
	defer func() { Identifier_Invariants(identifier, "uuid_identifier.identifier") }()
	UUID_Invariants(uuid, "uuid_identifier.uuid")
	return Identifier(binary.Uint_32(binary.Bytes(uuid[0:]), binary.BIG_ENDIAN))
}

// UUID_Time returns the timestamp embedded in uuid, well defined for versions 1, 2,
// 6, and 7. Each version packs the ticks differently, so the layout is selected by version.
func UUID_Time(uuid UUID) (timestamp Time) {
	defer func() { Time_Invariants(timestamp, "uuid_time.timestamp") }()
	UUID_Invariants(uuid, "uuid_time.uuid")
	switch UUID_Version(uuid) {
	case 6:
		high := int64(binary.Uint_32(binary.Bytes(uuid[0:]), binary.BIG_ENDIAN)) <<
			(V7_SEQUENCE_BIT_COUNT + bits.BIT_COUNT_16_MAXIMUM)
		middle := int64(binary.Uint_16(binary.Bytes(uuid[4:]), binary.BIG_ENDIAN)) <<
			V7_SEQUENCE_BIT_COUNT
		low := int64(binary.Uint_16(binary.Bytes(uuid[6:]), binary.BIG_ENDIAN) &
			binary.Word_16(V7_SEQUENCE_MAXIMUM))
		return Time(high | middle | low)
	case 7:
		milliseconds := int64(binary.Uint_64(binary.Bytes(uuid[:]), binary.BIG_ENDIAN) >>
			bits.BIT_COUNT_16_MAXIMUM)
		return Time(milliseconds*(NANO_PER_MILLI/UUID_TICK_NANOSECOND_COUNT) + EPOCH_100NS)
	}
	low := int64(binary.Uint_32(binary.Bytes(uuid[0:]), binary.BIG_ENDIAN))
	middle := int64(binary.Uint_16(binary.Bytes(uuid[4:]), binary.BIG_ENDIAN)) <<
		bits.BIT_COUNT_32_MAXIMUM
	high := int64(binary.Uint_16(binary.Bytes(uuid[6:]), binary.BIG_ENDIAN)&
		binary.Word_16(V7_SEQUENCE_MAXIMUM)) << NODE_BIT_COUNT
	return Time(low | middle | high)
}

// UUID_Clock_Sequence returns the clock sequence embedded in uuid, well defined for
// versions 1 and 2.
func UUID_Clock_Sequence(uuid UUID) (sequence Clock_Sequence) {
	defer func() { Clock_Sequence_Invariants(sequence, "uuid_clock_sequence.sequence") }()
	UUID_Invariants(uuid, "uuid_clock_sequence.uuid")
	return Clock_Sequence(binary.Uint_16(binary.Bytes(uuid[8:]), binary.BIG_ENDIAN)) &
		CLOCK_SEQUENCE_MAXIMUM
}

// UUIDs_Strings returns the canonical string form of each UUID in uuids.
func UUIDs_Strings(uuids UUIDs) (rendered Strings) {
	defer func() { Strings_Invariants(rendered, "uuids_strings.rendered") }()
	UUIDs_Invariants(uuids, "uuids_strings.uuids")
	rendered = make(Strings, len(uuids))
	for index, uuid := range uuids {
		rendered[index] = uuid.String()
	}
	return rendered
}

// Time_Unix converts an RFC 9562 timestamp to Unix seconds and nanoseconds.
func Time_Unix(timestamp Time) (seconds Unix_Second, nanoseconds Nanosecond) {
	defer func() {
		Unix_Second_Invariants(seconds, "time_unix.seconds")
		Nanosecond_Invariants(nanoseconds, "time_unix.nanoseconds")
	}()
	Time_Invariants(timestamp, "time_unix.timestamp")
	ticks := int64(timestamp) - EPOCH_100NS
	nanoseconds[0] = (ticks % UUID_TICK_COUNT_PER_SECOND) *
		UUID_TICK_NANOSECOND_COUNT
	seconds = Unix_Second(ticks / UUID_TICK_COUNT_PER_SECOND)
	return seconds, nanoseconds
}

// Compare_Input names the two operands Compare orders; a repeated field type requires
// the named input struct the house linter mandates.
type Compare_Input struct {
	// A is the left operand.
	A [UUID_BYTE_COUNT]byte
	// B is the right operand.
	B [UUID_BYTE_COUNT]byte
}

// Compare_Input_Invariants fixes both RFC 9562 operand widths without repeating UUID type.
func Compare_Input_Invariants(value Compare_Input, _ aver.Namespace) {
	aver.Always(
		len(value.A) == UUID_BYTE_COUNT,
		"First UUID comparison operand has RFC 9562 width.",
	)
	aver.Always(
		len(value.B) == UUID_BYTE_COUNT,
		"Second UUID comparison operand has RFC 9562 width.",
	)
}

// Compare orders two UUIDs lexicographically by their bytes: -1 if A<B, 0 if equal, +1 if A>B.
func Compare(input *Compare_Input) (order Order) {
	defer func() { Order_Invariants(order, "compare.order") }()
	Compare_Input_Invariants(*input, "compare.input")
	return Order(bytes.Compare(bytes.Slice(input.A[:]), bytes.Slice(input.B[:])))
}

// String returns static version name without allocating formatted text.
func (version Version) String() (name string) {
	switch version {
	case 0:
		return "VERSION_0"
	case 1:
		return "VERSION_1"
	case 2:
		return "VERSION_2"
	case 3:
		return "VERSION_3"
	case 4:
		return "VERSION_4"
	case 5:
		return "VERSION_5"
	case 6:
		return "VERSION_6"
	case 7:
		return "VERSION_7"
	case 8:
		return "VERSION_8"
	case 9:
		return "VERSION_9"
	case 10:
		return "VERSION_10"
	case 11:
		return "VERSION_11"
	case 12:
		return "VERSION_12"
	case 13:
		return "VERSION_13"
	case 14:
		return "VERSION_14"
	case 15:
		return "VERSION_15"
	}
	return "BAD_VERSION"
}

// String returns a readable name for the variant, satisfying fmt.Stringer.
func (variant Variant) String() (name string) {
	switch variant {
	case VARIANT_RFC_4122:
		return "RFC4122"
	case VARIANT_RESERVED:
		return "Reserved"
	case VARIANT_MICROSOFT:
		return "Microsoft"
	case VARIANT_FUTURE:
		return "Future"
	case VARIANT_INVALID:
		return "Invalid"
	}
	return "BadVariant"
}

// String returns a readable name for the domain, satisfying fmt.Stringer.
func (domain Domain) String() (name string) {
	switch domain {
	case DOMAIN_PERSON:
		return "Person"
	case DOMAIN_GROUP:
		return "Group"
	case DOMAIN_ORGANIZATION:
		return "Org"
	}
	return "DomainInvalid"
}

// MarshalText implements encoding.TextMarshaler, the canonical hyphenated form.
func (uuid UUID) MarshalText() (text []byte, err error) {
	var buffer [UUID_TEXT_BYTE_COUNT]byte
	encode_hexadecimal(&buffer, uuid)
	return buffer[:], nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (uuid *UUID) UnmarshalText(text []byte) (err error) {
	parsed, parse_err := Parse_Bytes(bytes.Slice(text))
	if parse_err != nil {
		return parse_err
	}
	*uuid = parsed
	return nil
}

// MarshalBinary implements encoding.BinaryMarshaler, the 16 raw bytes.
func (uuid UUID) MarshalBinary() (data []byte, err error) {
	return uuid[:], nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (uuid *UUID) UnmarshalBinary(data []byte) (err error) {
	if len(data) != UUID_BYTE_COUNT {
		return ERROR_INVALID_SIZE
	}
	copy(uuid[:], data)
	return nil
}

// UUID_Scan reads shared driver text or bytes without reflection or open unions.
func UUID_Scan(uuid *UUID, source driver.Value) (status driver.Validation_Status) {
	defer func() {
		driver.Validation_Status_Invariants(status, "uuid_scan.status")
	}()
	UUID_Invariants(*uuid, "uuid_scan.uuid")
	driver.Value_Invariants(source, "uuid_scan.source")
	status_ok := driver.Validation_Status(driver.STATUS_OK)
	status_invalid := driver.Validation_Status(driver.STATUS_INPUT_INVALID)
	switch source.Kinds[driver.VALUE_SLOT] {
	case driver.VALUE_NULL:
		return status_ok
	case driver.VALUE_TEXT:
		text, text_status := driver.Value_As_Text(source)
		if text_status != status_ok {
			return status_invalid
		}
		if len(text) == bytes.SLICE_SIZE_MINIMUM {
			return status_ok
		}
		parsed, parse_err := Parse(Text_Unvalidated(text))
		if parse_err != nil {
			return status_invalid
		}
		*uuid = parsed
		return status_ok
	case driver.VALUE_BYTES:
		data, data_status := driver.Value_As_Bytes(source)
		if data_status != status_ok {
			return status_invalid
		}
		if len(data) == bytes.SLICE_SIZE_MINIMUM {
			return status_ok
		}
		if len(data) == UUID_BYTE_COUNT {
			copy(uuid[:], data)
			return status_ok
		}
		parsed, parse_err := Parse_Bytes(bytes.Slice(data))
		if parse_err != nil {
			return status_invalid
		}
		*uuid = parsed
		return status_ok
	}
	return status_invalid
}

// UUID_Value returns canonical text through shared closed driver union.
func UUID_Value(uuid UUID) (value driver.Value) {
	defer func() { driver.Value_Invariants(value, "uuid_value.value") }()
	UUID_Invariants(uuid, "uuid_value.uuid")
	var status driver.Validation_Status
	value, status = driver.Value_Of_Text(driver.Text_Unvalidated(uuid.String()))
	aver.Always(
		status == driver.Validation_Status(driver.STATUS_OK),
		"Canonical UUID text always fits shared driver text bound.",
	)
	return value
}

// Null_UUID_Scan reads shared driver value and clears validity on NULL or rejection.
func Null_UUID_Scan(
	null_uuid *Null_UUID, source driver.Value,
) (status driver.Validation_Status) {
	defer func() {
		driver.Validation_Status_Invariants(status, "null_uuid_scan.status")
		Null_UUID_Invariants(*null_uuid, "null_uuid_scan.null_uuid.output")
	}()
	Null_UUID_Invariants(*null_uuid, "null_uuid_scan.null_uuid.input")
	driver.Value_Invariants(source, "null_uuid_scan.source")
	if source.Kinds[driver.VALUE_SLOT] == driver.VALUE_NULL {
		null_uuid.UUID = Nil()
		null_uuid.Valid = false
		return driver.Validation_Status(driver.STATUS_OK)
	}
	status = UUID_Scan(&null_uuid.UUID, source)
	if status != driver.Validation_Status(driver.STATUS_OK) {
		null_uuid.Valid = false
		return status
	}
	null_uuid.Valid = true
	return status
}

// Null_UUID_Value returns shared SQL NULL or canonical UUID text.
func Null_UUID_Value(null_uuid Null_UUID) (value driver.Value) {
	defer func() { driver.Value_Invariants(value, "null_uuid_value.value") }()
	Null_UUID_Invariants(null_uuid, "null_uuid_value.null_uuid")
	if !null_uuid.Valid {
		return driver.Value_Null()
	}
	return UUID_Value(null_uuid.UUID)
}

// MarshalBinary implements encoding.BinaryMarshaler; an invalid value marshals empty.
func (null_uuid Null_UUID) MarshalBinary() (data []byte, err error) {
	if bool(null_uuid.Valid) {
		return null_uuid.UUID[:], nil
	}
	return []byte(nil), nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (null_uuid *Null_UUID) UnmarshalBinary(data []byte) (err error) {
	if len(data) != UUID_BYTE_COUNT {
		return ERROR_INVALID_SIZE
	}
	copy(null_uuid.UUID[:], data)
	null_uuid.Valid = true
	return nil
}

// MarshalText implements encoding.TextMarshaler; an invalid value marshals as "null".
func (null_uuid Null_UUID) MarshalText() (text []byte, err error) {
	if bool(null_uuid.Valid) {
		return null_uuid.UUID.MarshalText()
	}
	return []byte("null"), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (null_uuid *Null_UUID) UnmarshalText(text []byte) (err error) {
	parsed, parse_err := Parse_Bytes(bytes.Slice(text))
	if parse_err != nil {
		null_uuid.Valid = false
		return parse_err
	}
	null_uuid.UUID = parsed
	null_uuid.Valid = true
	return nil
}

// MarshalJSON implements json.Marshaler; an invalid value marshals as JSON null.
func (null_uuid Null_UUID) MarshalJSON() (data []byte, err error) {
	if bool(null_uuid.Valid) {
		data = make([]byte, UUID_BRACED_BYTE_COUNT)
		var encoded [UUID_TEXT_BYTE_COUNT]byte
		data[bytes.SLICE_SIZE_MINIMUM] = '"'
		encode_hexadecimal(&encoded, null_uuid.UUID)
		copy(data[UUID_BRACE_BYTE_COUNT/2:UUID_BRACED_BYTE_COUNT-1], encoded[:])
		data[UUID_BRACED_BYTE_COUNT-1] = '"'
		return data, nil
	}
	return []byte("null"), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (null_uuid *Null_UUID) UnmarshalJSON(data []byte) (err error) {
	if string(data) == JSON_NULL {
		null_uuid.UUID = Nil()
		null_uuid.Valid = false
		return nil
	}
	if len(data) != UUID_BRACED_BYTE_COUNT {
		null_uuid.Valid = false
		return ERROR_INVALID_FORMAT
	}
	if data[bytes.SLICE_SIZE_MINIMUM] != '"' {
		null_uuid.Valid = false
		return ERROR_INVALID_FORMAT
	}
	if data[UUID_BRACED_BYTE_COUNT-1] != '"' {
		null_uuid.Valid = false
		return ERROR_INVALID_FORMAT
	}
	parsed, unmarshal_err := Parse_Bytes(bytes.Slice(
		data[UUID_BRACE_BYTE_COUNT/2 : UUID_BRACED_BYTE_COUNT-1],
	))
	if unmarshal_err != nil {
		null_uuid.Valid = false
		return unmarshal_err
	}
	null_uuid.UUID = parsed
	null_uuid.Valid = true
	return nil
}
