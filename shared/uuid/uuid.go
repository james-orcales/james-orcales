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
// accessor like the upstream uuid.Version() is a free function UUID_Version(u).
// Methods that satisfy fmt.Stringer, sql.Scanner, or the encoding marshalers are
// kept, since those are the seams the standard library reaches through.
package uuid

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"database/sql/driver"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"strings"

	"local/james-orcales/shared/time"
)

// UUID_BYTE_COUNT is the RFC 9562 binary width.
const UUID_BYTE_COUNT = 16

// NODE_BYTE_COUNT is the RFC 9562 node-field width.
const NODE_BYTE_COUNT = 6

// CLOCK_SEQUENCE_BYTE_COUNT stores the complete RFC 9562 clock sequence.
const CLOCK_SEQUENCE_BYTE_COUNT = 2

// UUID_TEXT_BYTE_COUNT is the canonical hyphenated text width.
const UUID_TEXT_BYTE_COUNT = 36

// UUID_URN_PREFIX_BYTE_COUNT accounts for the complete "urn:uuid:" prefix.
const UUID_URN_PREFIX_BYTE_COUNT = 9

// UUID_URN_BYTE_COUNT prevents a URN formatter from allocating a larger buffer.
const UUID_URN_BYTE_COUNT = UUID_URN_PREFIX_BYTE_COUNT + UUID_TEXT_BYTE_COUNT

// UUID is a 128-bit RFC 9562 Universally Unique IDentifier.
type UUID [UUID_BYTE_COUNT]byte

// UUIDs is a slice of UUID, given a name so UUIDs_Strings can hang off it.
type UUIDs []UUID

// Version is the version nibble of a UUID: which generation algorithm produced it.
type Version byte

// Variant is the layout variant of a UUID: which bit interpretation its fields follow.
type Variant byte

// Domain is a DCE 1.1 (Version 2) security domain.
type Domain byte

// Time is an RFC 9562 timestamp: 100-nanosecond ticks since 15 Oct 1582.
type Time int64

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

// JULIAN_1582 is the Julian day number of 15 Oct 1582, the RFC 9562 epoch.
const JULIAN_1582 = 2299160

// JULIAN_1970 is the Julian day number of 1 Jan 1970, the Unix epoch.
const JULIAN_1970 = 2440587

// EPOCH_DAYS is the day count between the RFC 9562 and Unix epochs.
const EPOCH_DAYS = JULIAN_1970 - JULIAN_1582

// EPOCH_SECONDS is the second count between the two epochs.
const EPOCH_SECONDS = EPOCH_DAYS * 86400

// EPOCH_100NS is the 100-nanosecond-tick count between the two epochs, the constant
// that rebases a Unix time onto the RFC 9562 timeline.
const EPOCH_100NS = EPOCH_SECONDS * 10000000

// NANO_PER_MILLI is nanoseconds per millisecond, the Version 7 timestamp unit.
const NANO_PER_MILLI = 1000000

// Error_Invalid_Format reports a string that is not a UUID in any accepted form.
var Error_Invalid_Format = errors.New("uuid: invalid format")

// Error_Invalid_Bracketed_Format reports a brace-wrapped string missing a brace.
var Error_Invalid_Bracketed_Format = errors.New("uuid: invalid bracketed format")

// Error_Invalid_Size reports a string whose character count matches no accepted form.
var Error_Invalid_Size = errors.New("uuid: invalid length")

// Error_Invalid_URN_Prefix reports a 45-byte string whose prefix is not "urn:uuid:".
var Error_Invalid_URN_Prefix = errors.New("uuid: invalid urn prefix")

// Generator holds the injected sources the upstream package kept as mutable package
// globals. A zero Generator is unusable; construct one with New. It is not safe for
// concurrent use — the clock-sequence and monotonic-time fields are mutated on each
// timestamped draw, so give each goroutine its own.
type Generator struct {
	// Source supplies entropy for the random fields of V4 and V7 and for a random node
	// when Node is zero. Production wires crypto/rand; a simulation wires a seeded reader.
	Source io.Reader
	// Clock supplies wall-clock time for the timestamp fields of V1, V6, and V7.
	Clock time.Clock
	// Node is the 6-byte node identifier embedded in V1 and V6. A zero value draws a
	// random node from Source on first use.
	Node [NODE_BYTE_COUNT]byte
	// Clock_Sequence is the V1/V6 sequence counter, bumped when the clock repeats or
	// regresses so same-instant UUIDs still differ.
	Clock_Sequence uint16
	// Clock_Sequence_Set records whether Clock_Sequence has been seeded from Source.
	Clock_Sequence_Set bool
	// Last_Time is the last V1/V6 timestamp (100ns units) handed out, for regression detection.
	Last_Time uint64
	// Last_V7 is the last V7 value (milliseconds<<12 | sub-millisecond sequence) handed out.
	Last_V7 int64
	// Node_Resolved records whether a zero Node has been replaced by a random draw.
	Node_Resolved bool
}

// Null_UUID is a UUID that may be SQL NULL, the scan destination for a nullable column.
type Null_UUID struct {
	// UUID is the value, meaningful only when Valid.
	UUID UUID
	// Valid is true when UUID holds a non-NULL value.
	Valid bool
}

// New builds a Generator from its three injected sources. The parameter types are
// distinct, so they stay positional rather than folding into an input struct.
func New(source io.Reader, clock time.Clock, node [NODE_BYTE_COUNT]byte) (generator Generator) {
	generator.Source = source
	generator.Clock = clock
	generator.Node = node
	return generator
}

// Nil is the zero UUID, all 128 bits clear. It is a function, not a package var,
// because a UUID is an array literal and the house linter bans mutable package state.
func Nil() (uuid UUID) {
	return UUID{}
}

// Max is the RFC 9562 maximum UUID, all 128 bits set.
func Max() (uuid UUID) {
	return UUID{
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	}
}

// Name_Space_DNS is the RFC 9562 namespace for domain names, used as the V3/V5 space.
func Name_Space_DNS() (uuid UUID) {
	return Must_Parse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
}

// Name_Space_URL is the RFC 9562 namespace for URLs, used as the V3/V5 space.
func Name_Space_URL() (uuid UUID) {
	return Must_Parse("6ba7b811-9dad-11d1-80b4-00c04fd430c8")
}

// Name_Space_OID is the RFC 9562 namespace for ISO OIDs, used as the V3/V5 space.
func Name_Space_OID() (uuid UUID) {
	return Must_Parse("6ba7b812-9dad-11d1-80b4-00c04fd430c8")
}

// Name_Space_X500 is the RFC 9562 namespace for X.500 DNs, used as the V3/V5 space.
func Name_Space_X500() (uuid UUID) {
	return Must_Parse("6ba7b814-9dad-11d1-80b4-00c04fd430c8")
}

// Generator_V4 returns a random (Version 4) UUID, 122 bits drawn from Source.
func Generator_V4(generator *Generator) (uuid UUID, err error) {
	_, read_err := io.ReadFull(generator.Source, uuid[:])
	if read_err != nil {
		return Nil(), read_err
	}
	uuid[6] = uuid[6]&0x0f | 0x40 // Version 4.
	uuid[8] = uuid[8]&0x3f | 0x80 // RFC 9562 variant.
	return uuid, nil
}

// Generator_V7 returns a time-ordered (Version 7) UUID: 48 bits of Unix
// milliseconds, then a sub-millisecond sequence, then random bits. The value is
// strictly greater than any previous V7 from this Generator, so a burst within one
// millisecond still sorts in creation order.
func Generator_V7(generator *Generator) (uuid UUID, err error) {
	uuid, read_err := Generator_V4(generator)
	if read_err != nil {
		return Nil(), read_err
	}
	milliseconds, sequence := generator_v7_time(generator)
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
	timestamp, sequence, time_err := generator_time(generator)
	if time_err != nil {
		return Nil(), time_err
	}
	node, node_err := generator_node(generator)
	if node_err != nil {
		return Nil(), node_err
	}
	time_low := uint32(timestamp & 0xffffffff)
	time_middle := uint16((timestamp >> 32) & 0xffff)
	time_high := uint16((timestamp >> 48) & 0x0fff)
	time_high = time_high | 0x1000 // Version 1.
	binary.BigEndian.PutUint32(uuid[0:], time_low)
	binary.BigEndian.PutUint16(uuid[4:], time_middle)
	binary.BigEndian.PutUint16(uuid[6:], time_high)
	binary.BigEndian.PutUint16(uuid[8:], sequence)
	copy(uuid[10:], node[:])
	return uuid, nil
}

// Generator_V6 returns a Version 6 UUID: the V1 fields reordered most-significant
// first so that lexical order tracks time order, better for database locality.
func Generator_V6(generator *Generator) (uuid UUID, err error) {
	timestamp, sequence, time_err := generator_time(generator)
	if time_err != nil {
		return Nil(), time_err
	}
	node, node_err := generator_node(generator)
	if node_err != nil {
		return Nil(), node_err
	}
	time_high := uint32((timestamp >> 28) & 0xffffffff)
	time_middle := uint16((timestamp >> 12) & 0xffff)
	time_low := uint16(timestamp & 0x0fff)
	time_low = time_low | 0x6000 // Version 6.
	binary.BigEndian.PutUint32(uuid[0:], time_high)
	binary.BigEndian.PutUint16(uuid[4:], time_middle)
	binary.BigEndian.PutUint16(uuid[6:], time_low)
	binary.BigEndian.PutUint16(uuid[8:], sequence)
	copy(uuid[10:], node[:])
	return uuid, nil
}

// Generator_DCE_Security returns a DCE Security (Version 2) UUID that embeds domain
// and identifier in the low fields of a Version 1 layout. The identifier is a user
// or group id the caller supplies (the host's os.Getuid/os.Getgid), kept out of this
// pure package so it imports no operating-system state.
func Generator_DCE_Security(
	generator *Generator, domain Domain, identifier uint32,
) (uuid UUID, err error) {
	uuid, v1_err := Generator_V1(generator)
	if v1_err != nil {
		return Nil(), v1_err
	}
	uuid[6] = uuid[6]&0x0f | 0x20 // Version 2.
	uuid[9] = byte(domain)
	binary.BigEndian.PutUint32(uuid[0:], identifier)
	return uuid, nil
}

// V3 returns a name-based MD5 (Version 3) UUID: the deterministic hash of namespace
// concatenated with data. The same inputs always yield the same UUID, so V3 needs
// no Generator.
func V3(namespace UUID, data []byte) (uuid UUID) {
	return new_hash(md5.New(), namespace, data, 3)
}

// V5 returns a name-based SHA-1 (Version 5) UUID, the SHA-1 counterpart of V3 and
// the RFC-preferred of the two name-based versions.
func V5(namespace UUID, data []byte) (uuid UUID) {
	return new_hash(sha1.New(), namespace, data, 5)
}

// Returns the current time as RFC 9562 100-nanosecond ticks since 15 Oct 1582, with
// the clock sequence, advancing the sequence if the clock did not move forward so
// successive UUIDs stay distinct and ordered.
func generator_time(generator *Generator) (timestamp uint64, sequence uint16, err error) {
	if !generator.Clock_Sequence_Set {
		seed_err := generator_seed_clock_sequence(generator)
		if seed_err != nil {
			return 0, 0, seed_err
		}
	}
	now := uint64(int64(generator.Clock.Now_Realtime())/100) + EPOCH_100NS
	if now <= generator.Last_Time {
		generator.Clock_Sequence = (generator.Clock_Sequence+1)&0x3fff | 0x8000
	}
	generator.Last_Time = now
	return now, generator.Clock_Sequence, nil
}

// Returns Unix milliseconds and a sub-millisecond sequence, forced strictly upward
// past the last V7 draw so a same-millisecond burst still orders by creation.
func generator_v7_time(generator *Generator) (milliseconds int64, sequence int64) {
	nanoseconds := int64(generator.Clock.Now_Realtime())
	milliseconds = nanoseconds / NANO_PER_MILLI
	sequence = (nanoseconds - milliseconds*NANO_PER_MILLI) >> 8
	combined := milliseconds<<12 + sequence
	if combined <= generator.Last_V7 {
		combined = generator.Last_V7 + 1
		milliseconds = combined >> 12
		sequence = combined & 0xfff
	}
	generator.Last_V7 = combined
	return milliseconds, sequence
}

// Draws the initial 14-bit clock sequence from Source, matching the upstream
// random-clock-sequence-on-first-use behavior.
func generator_seed_clock_sequence(generator *Generator) (err error) {
	var raw [CLOCK_SEQUENCE_BYTE_COUNT]byte
	_, read_err := io.ReadFull(generator.Source, raw[:])
	if read_err != nil {
		return read_err
	}
	sequence := uint16(raw[0])<<8 | uint16(raw[1])
	generator.Clock_Sequence = sequence&0x3fff | 0x8000 // RFC 9562 variant bits.
	generator.Clock_Sequence_Set = true
	return nil
}

// Resolves the node used for V1 and V6: the injected Node when set, otherwise a
// one-time random draw from Source with the multicast bit set to mark it as not a
// real hardware address.
func generator_node(generator *Generator) (node [NODE_BYTE_COUNT]byte, err error) {
	if generator.Node_Resolved {
		return generator.Node, nil
	}
	if generator.Node != ([NODE_BYTE_COUNT]byte{}) {
		generator.Node_Resolved = true
		return generator.Node, nil
	}
	_, read_err := io.ReadFull(generator.Source, generator.Node[:])
	if read_err != nil {
		return generator.Node, read_err
	}
	generator.Node[0] = generator.Node[0] | 0x01 // Multicast bit: not a real MAC.
	generator.Node_Resolved = true
	return generator.Node, nil
}

// Forms a UUID from the first 16 bytes of hasher over namespace then data, stamping
// the given version and the RFC 9562 variant.
func new_hash(hasher hash.Hash, namespace UUID, data []byte, version byte) (uuid UUID) {
	hasher.Reset()
	// The Write calls never return an error by contract (hash.Hash), so their results
	// are intentionally unused.
	hasher.Write(namespace[:])
	hasher.Write(data)
	sum := hasher.Sum(nil)
	copy(uuid[:], sum)
	uuid[6] = uuid[6]&0x0f | (version&0x0f)<<4
	uuid[8] = uuid[8]&0x3f | 0x80 // RFC 9562 variant.
	return uuid
}

// Parse decodes s into a UUID. It accepts the canonical hyphenated form, the
// urn:uuid: prefixed form, a single-brace-wrapped form, and the 32-character
// unhyphenated form; any other length is an error. Parse is lenient by design — use
// Validate to reject non-canonical encodings.
func Parse(s string) (uuid UUID, err error) {
	switch len(s) {
	case 36:
		return parse_canonical(s)
	case 9 + 36:
		if !strings.EqualFold(s[:9], "urn:uuid:") {
			return uuid, Error_Invalid_URN_Prefix
		}
		return parse_canonical(s[9:])
	case 2 + 36:
		// The parse_canonical function reads only the leading 36 bytes. Thus the closing
		// byte is checked here or never, and any two wrapper characters pass.
		if s[0] != '{' {
			return uuid, Error_Invalid_Bracketed_Format
		}
		if s[37] != '}' {
			return uuid, Error_Invalid_Bracketed_Format
		}
		return parse_canonical(s[1:37])
	case 32:
		return parse_compact(s)
	}
	return uuid, fmt.Errorf("%w: %d", Error_Invalid_Size, len(s))
}

// Parse_Bytes is Parse over a byte slice. It delegates through a string conversion
// rather than duplicating the format logic; a UUID is 45 bytes at most, so the copy
// is negligible.
func Parse_Bytes(b []byte) (uuid UUID, err error) {
	return Parse(string(b))
}

// Validate reports whether s is a UUID Parse would accept, discarding the value.
func Validate(s string) (err error) {
	_, parse_err := Parse(s)
	return parse_err
}

// Must_Parse is Parse but panics on error, for compile-time-constant UUIDs where a
// parse failure is a programming error, not a runtime condition.
func Must_Parse(s string) (uuid UUID) {
	parsed, err := Parse(s)
	if err != nil {
		panic(fmt.Sprintf("uuid: Must_Parse(%q): %v", s, err))
	}
	return parsed
}

// Must unwraps a (UUID, error) pair, panicking on error. It wraps a generator call
// whose only error is a starved entropy reader — a condition production treats as fatal.
func Must(uuid UUID, err error) (result UUID) {
	if err != nil {
		panic(err)
	}
	return uuid
}

// From_Bytes builds a UUID from exactly 16 raw bytes, which it copies.
func From_Bytes(b []byte) (uuid UUID, err error) {
	if len(b) != 16 {
		return uuid, fmt.Errorf("uuid: from bytes got %d, want 16", len(b))
	}
	copy(uuid[:], b)
	return uuid, nil
}

// Decodes the hyphenated form from the first 36 bytes of s.
func parse_canonical(s string) (uuid UUID, err error) {
	dash_positions := []int{8, 13, 18, 23}
	for _, position := range dash_positions {
		if s[position] != '-' {
			return uuid, Error_Invalid_Format
		}
	}
	source_offsets := []int{0, 2, 4, 6, 9, 11, 14, 16, 19, 21, 24, 26, 28, 30, 32, 34}
	for index, offset := range source_offsets {
		value, ok := hexadecimal_pair(s, offset)
		if !ok {
			return uuid, Error_Invalid_Format
		}
		uuid[index] = value
	}
	return uuid, nil
}

// Decodes the 32-character unhyphenated form.
func parse_compact(s string) (uuid UUID, err error) {
	for index := range uuid {
		value, ok := hexadecimal_pair(s, index*2)
		if !ok {
			return uuid, Error_Invalid_Format
		}
		uuid[index] = value
	}
	return uuid, nil
}

// Decodes the two hex digits at s[index:index+2] into one byte.
func hexadecimal_pair(s string, index int) (value byte, ok bool) {
	high, high_ok := hexadecimal_value(s[index])
	if !high_ok {
		return 0, false
	}
	low, low_ok := hexadecimal_value(s[index+1])
	if !low_ok {
		return 0, false
	}
	return high<<4 | low, true
}

// Decodes one hexadecimal digit, reporting ok=false for a non-hex byte.
func hexadecimal_value(character byte) (value byte, ok bool) {
	if character >= '0' {
		if character <= '9' {
			return character - '0', true
		}
	}
	if character >= 'a' {
		if character <= 'f' {
			return character - 'a' + 10, true
		}
	}
	if character >= 'A' {
		if character <= 'F' {
			return character - 'A' + 10, true
		}
	}
	return 0, false
}

// String returns the canonical 36-character form xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx,
// satisfying fmt.Stringer.
func (uuid UUID) String() (text string) {
	var buffer [UUID_TEXT_BYTE_COUNT]byte
	encode_hexadecimal(buffer[:], uuid)
	return string(buffer[:])
}

// UUID_URN returns the RFC 2141 URN form urn:uuid:xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.
func UUID_URN(uuid UUID) (urn string) {
	var buffer [UUID_URN_BYTE_COUNT]byte
	copy(buffer[:], "urn:uuid:")
	encode_hexadecimal(buffer[9:], uuid)
	return string(buffer[:])
}

// Writes the canonical hyphenated hex of uuid into destination, which must be at
// least 36 bytes.
func encode_hexadecimal(destination []byte, uuid UUID) {
	hex.Encode(destination, uuid[:4])
	destination[8] = '-'
	hex.Encode(destination[9:13], uuid[4:6])
	destination[13] = '-'
	hex.Encode(destination[14:18], uuid[6:8])
	destination[18] = '-'
	hex.Encode(destination[19:23], uuid[8:10])
	destination[23] = '-'
	hex.Encode(destination[24:], uuid[10:])
}

// UUID_Version returns the version encoded in uuid.
func UUID_Version(uuid UUID) (version Version) {
	return Version(uuid[6] >> 4)
}

// UUID_Variant returns the layout variant encoded in uuid.
func UUID_Variant(uuid UUID) (variant Variant) {
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
func UUID_Node_Identifier(uuid UUID) (node []byte) {
	var copied [NODE_BYTE_COUNT]byte
	copy(copied[:], uuid[10:])
	return copied[:]
}

// UUID_Domain returns the domain of a Version 2 UUID.
func UUID_Domain(uuid UUID) (domain Domain) {
	return Domain(uuid[9])
}

// UUID_Identifier returns the embedded identifier of a Version 2 UUID.
func UUID_Identifier(uuid UUID) (identifier uint32) {
	return binary.BigEndian.Uint32(uuid[0:4])
}

// UUID_Time returns the timestamp embedded in uuid, well defined for versions 1, 2,
// 6, and 7. Each version packs the ticks differently, so the layout is selected by version.
func UUID_Time(uuid UUID) (timestamp Time) {
	switch UUID_Version(uuid) {
	case 6:
		high := int64(binary.BigEndian.Uint32(uuid[0:4])) << 28
		middle := int64(binary.BigEndian.Uint16(uuid[4:6])) << 12
		low := int64(binary.BigEndian.Uint16(uuid[6:8]) & 0x0fff)
		return Time(high | middle | low)
	case 7:
		milliseconds := int64(binary.BigEndian.Uint64(uuid[:8]) >> 16)
		return Time(milliseconds*10000 + EPOCH_100NS)
	}
	low := int64(binary.BigEndian.Uint32(uuid[0:4]))
	middle := int64(binary.BigEndian.Uint16(uuid[4:6])) << 32
	high := int64(binary.BigEndian.Uint16(uuid[6:8])&0x0fff) << 48
	return Time(low | middle | high)
}

// UUID_Clock_Sequence returns the clock sequence embedded in uuid, well defined for
// versions 1 and 2.
func UUID_Clock_Sequence(uuid UUID) (sequence int) {
	return int(binary.BigEndian.Uint16(uuid[8:10])) & 0x3fff
}

// UUIDs_Strings returns the canonical string form of each UUID in uuids.
func UUIDs_Strings(uuids UUIDs) (rendered []string) {
	rendered = make([]string, len(uuids))
	for index, uuid := range uuids {
		rendered[index] = uuid.String()
	}
	return rendered
}

// Time_Unix converts an RFC 9562 timestamp to Unix seconds and nanoseconds.
func Time_Unix(timestamp Time) (seconds int64, nanoseconds int64) {
	ticks := int64(timestamp) - EPOCH_100NS
	nanoseconds = (ticks % 10000000) * 100
	seconds = ticks / 10000000
	return seconds, nanoseconds
}

// Compare_Input names the two operands Compare orders; a repeated field type requires
// the named input struct the house linter mandates.
type Compare_Input struct {
	// A is the left operand.
	A UUID
	// B is the right operand.
	B UUID
}

// Compare orders two UUIDs lexicographically by their bytes: -1 if A<B, 0 if equal, +1 if A>B.
func Compare(input *Compare_Input) (order int) {
	return bytes.Compare(input.A[:], input.B[:])
}

// String returns a readable name for the version, satisfying fmt.Stringer.
func (version Version) String() (name string) {
	if version > 15 {
		return fmt.Sprintf("BAD_VERSION_%d", byte(version))
	}
	return fmt.Sprintf("VERSION_%d", byte(version))
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
	return fmt.Sprintf("BadVariant%d", byte(variant))
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
	return fmt.Sprintf("Domain%d", byte(domain))
}

// MarshalText implements encoding.TextMarshaler, the canonical hyphenated form.
func (uuid UUID) MarshalText() (text []byte, err error) {
	var buffer [UUID_TEXT_BYTE_COUNT]byte
	encode_hexadecimal(buffer[:], uuid)
	return buffer[:], nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (uuid *UUID) UnmarshalText(text []byte) (err error) {
	parsed, parse_err := Parse_Bytes(text)
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
	if len(data) != 16 {
		return fmt.Errorf("uuid: unmarshal binary got %d, want 16", len(data))
	}
	copy(uuid[:], data)
	return nil
}

// Scan implements sql.Scanner, reading a UUID from a string or 16-byte column.
func (uuid *UUID) Scan(source any) (err error) {
	switch typed := source.(type) {
	case nil:
		return nil
	case string:
		return scan_text(uuid, typed)
	case []byte:
		if len(typed) == 0 {
			return nil
		}
		// A 16-byte column is the raw form; anything else is text, delegated to
		// scan_text rather than a recursive Scan, which the linter bans.
		if len(typed) != 16 {
			return scan_text(uuid, string(typed))
		}
		copy(uuid[:], typed)
		return nil
	}
	return fmt.Errorf("uuid: cannot scan %T", source)
}

// UUID_Value returns the database driver value for uuid — its canonical string — the
// free-function replacement for the driver.Valuer method the linter bans.
func UUID_Value(uuid UUID) (value driver.Value, err error) {
	return uuid.String(), nil
}

// Parses a textual column into uuid, treating the empty string as a no-op so a
// NULL-like empty cell leaves uuid at its zero value.
func scan_text(uuid *UUID, text string) (err error) {
	if text == "" {
		return nil
	}
	parsed, parse_err := Parse(text)
	if parse_err != nil {
		return fmt.Errorf("uuid: scan: %w", parse_err)
	}
	*uuid = parsed
	return nil
}

// Scan implements sql.Scanner over a nullable column, clearing Valid on NULL.
func (null_uuid *Null_UUID) Scan(source any) (err error) {
	if source == nil {
		null_uuid.UUID = Nil()
		null_uuid.Valid = false
		return nil
	}
	scan_err := null_uuid.UUID.Scan(source)
	if scan_err != nil {
		null_uuid.Valid = false
		return scan_err
	}
	null_uuid.Valid = true
	return nil
}

// Null_UUID_Value returns the database driver value, SQL NULL when not Valid.
func Null_UUID_Value(null_uuid Null_UUID) (value driver.Value, err error) {
	if !null_uuid.Valid {
		return nil, nil
	}
	return UUID_Value(null_uuid.UUID)
}

// MarshalBinary implements encoding.BinaryMarshaler; an invalid value marshals empty.
func (null_uuid Null_UUID) MarshalBinary() (data []byte, err error) {
	if null_uuid.Valid {
		return null_uuid.UUID[:], nil
	}
	return []byte(nil), nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (null_uuid *Null_UUID) UnmarshalBinary(data []byte) (err error) {
	if len(data) != 16 {
		return fmt.Errorf("uuid: unmarshal binary got %d, want 16", len(data))
	}
	copy(null_uuid.UUID[:], data)
	null_uuid.Valid = true
	return nil
}

// MarshalText implements encoding.TextMarshaler; an invalid value marshals as "null".
func (null_uuid Null_UUID) MarshalText() (text []byte, err error) {
	if null_uuid.Valid {
		return null_uuid.UUID.MarshalText()
	}
	return []byte("null"), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (null_uuid *Null_UUID) UnmarshalText(text []byte) (err error) {
	parsed, parse_err := Parse_Bytes(text)
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
	if null_uuid.Valid {
		return json.Marshal(null_uuid.UUID)
	}
	return []byte("null"), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (null_uuid *Null_UUID) UnmarshalJSON(data []byte) (err error) {
	if bytes.Equal(data, []byte("null")) {
		null_uuid.UUID = Nil()
		null_uuid.Valid = false
		return nil
	}
	unmarshal_err := json.Unmarshal(data, &null_uuid.UUID)
	if unmarshal_err != nil {
		null_uuid.Valid = false
		return unmarshal_err
	}
	null_uuid.Valid = true
	return nil
}
