// Package pem implements bounded Privacy-Enhanced Mail encoding on caller-owned storage.
package pem

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/encoding/base64"
	"local/james-orcales/shared/sim/aver/default"
)

// BEGIN_PREFIX opens one line-bound block.
const BEGIN_PREFIX = "-----BEGIN "

// END_PREFIX closes one line-bound block.
const END_PREFIX = "-----END "

// BOUNDARY_SUFFIX terminates both boundary labels.
const BOUNDARY_SUFFIX = "-----"

// HEADER_SEPARATOR keeps key and value separation canonical.
const HEADER_SEPARATOR = ": "

// LINE_FEED is the canonical encoded line ending.
const LINE_FEED byte = '\n'

// CARRIAGE_RETURN is accepted only as part of decoded line endings.
const CARRIAGE_RETURN byte = '\r'

// HORIZONTAL_TAB is ignored in base64 and trimmed from metadata fields.
const HORIZONTAL_TAB byte = '\t'

// SPACE is ignored in base64 and trimmed from metadata fields.
const SPACE byte = ' '

// HEADER_KEY_SEPARATOR distinguishes metadata from base64 body lines.
const HEADER_KEY_SEPARATOR byte = ':'

// BASE64_PADDING closes a partial standard base64 quantum.
const BASE64_PADDING byte = '='

// LINE_FEED_SIZE derives every line contribution from the wire token.
const LINE_FEED_SIZE = len("\n")

// BEGIN_PREFIX_SIZE derives the searchable boundary prefix width.
const BEGIN_PREFIX_SIZE = len(BEGIN_PREFIX)

// BOUNDARY_SUFFIX_SIZE derives both boundary widths from their shared token.
const BOUNDARY_SUFFIX_SIZE = len(BOUNDARY_SUFFIX)

// BEGIN_LINE_FIXED_SIZE excludes only the caller block type.
const BEGIN_LINE_FIXED_SIZE = len(BEGIN_PREFIX) + BOUNDARY_SUFFIX_SIZE + LINE_FEED_SIZE

// END_LINE_FIXED_SIZE excludes only the caller block type.
const END_LINE_FIXED_SIZE = len(END_PREFIX) + BOUNDARY_SUFFIX_SIZE + LINE_FEED_SIZE

// BLOCK_SIZE_MINIMUM is an empty block with an empty type.
const BLOCK_SIZE_MINIMUM = BEGIN_LINE_FIXED_SIZE + END_LINE_FIXED_SIZE

// BEGIN_SOURCE_SIZE_MINIMUM can contain one complete empty BEGIN line.
const BEGIN_SOURCE_SIZE_MINIMUM = BEGIN_LINE_FIXED_SIZE

// SECTION_SOURCE_SIZE_MINIMUM leaves one byte after a complete BEGIN line.
const SECTION_SOURCE_SIZE_MINIMUM = BEGIN_SOURCE_SIZE_MINIMUM + 1

// ENCODED_SIZE_MAXIMUM follows the repository byte-slice boundary.
const ENCODED_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// TYPE_SIZE_MAXIMUM leaves room for two appearances and fixed block framing.
const TYPE_SIZE_MAXIMUM = (ENCODED_SIZE_MAXIMUM - BLOCK_SIZE_MINIMUM) / 2

// HEADER_LINE_SIZE_MINIMUM is an empty key, empty value, separator, and line ending.
const HEADER_LINE_SIZE_MINIMUM = len(HEADER_SEPARATOR) + LINE_FEED_SIZE

// HEADER_SECTION_SEPARATOR_SIZE is the blank line emitted after nonempty headers.
const HEADER_SECTION_SEPARATOR_SIZE = LINE_FEED_SIZE

// HEADER_COUNT_MAXIMUM fills bounded output with minimum header lines and framing.
const HEADER_COUNT_MAXIMUM = (ENCODED_SIZE_MAXIMUM - BLOCK_SIZE_MINIMUM -
	HEADER_SECTION_SEPARATOR_SIZE) / HEADER_LINE_SIZE_MINIMUM

// BASE64_LINE_ENCODED_SIZE follows the conventional 64-column PEM body.
const BASE64_LINE_ENCODED_SIZE = base64.ALPHABET_SIZE

// BASE64_LINE_SOURCE_SIZE is the complete source quanta represented by one encoded line.
const BASE64_LINE_SOURCE_SIZE = BASE64_LINE_ENCODED_SIZE /
	base64.ENCODED_GROUP_SIZE * base64.DECODED_GROUP_SIZE

// BASE64_LINE_STORAGE_SIZE includes its canonical line ending.
const BASE64_LINE_STORAGE_SIZE = BASE64_LINE_ENCODED_SIZE + LINE_FEED_SIZE

// BODY_STORAGE_MAXIMUM is output left after minimum framing.
const BODY_STORAGE_MAXIMUM = ENCODED_SIZE_MAXIMUM - BLOCK_SIZE_MINIMUM

// BASE64_FULL_LINE_COUNT_MAXIMUM is every complete wrapped line fitting the body.
const BASE64_FULL_LINE_COUNT_MAXIMUM = BODY_STORAGE_MAXIMUM / BASE64_LINE_STORAGE_SIZE

// BASE64_BODY_REMAINDER_SIZE is storage after all fitting complete lines.
const BASE64_BODY_REMAINDER_SIZE = BODY_STORAGE_MAXIMUM % BASE64_LINE_STORAGE_SIZE

// BASE64_TAIL_ENCODED_SIZE_MAXIMUM leaves one byte for its canonical line ending.
const BASE64_TAIL_ENCODED_SIZE_MAXIMUM = BASE64_BODY_REMAINDER_SIZE - LINE_FEED_SIZE

// BASE64_TAIL_GROUP_COUNT_MAXIMUM admits only complete padded quanta.
const BASE64_TAIL_GROUP_COUNT_MAXIMUM = BASE64_TAIL_ENCODED_SIZE_MAXIMUM /
	base64.ENCODED_GROUP_SIZE

// BASE64_TAIL_SOURCE_SIZE_MAXIMUM is source represented by the fitting tail quanta.
const BASE64_TAIL_SOURCE_SIZE_MAXIMUM = BASE64_TAIL_GROUP_COUNT_MAXIMUM *
	base64.DECODED_GROUP_SIZE

// ENCODE_SOURCE_SIZE_MAXIMUM is the largest payload whose wrapped body stays bounded.
const ENCODE_SOURCE_SIZE_MAXIMUM = BASE64_FULL_LINE_COUNT_MAXIMUM*BASE64_LINE_SOURCE_SIZE +
	BASE64_TAIL_SOURCE_SIZE_MAXIMUM

// DECODE_BODY_ENCODED_SIZE_MAXIMUM leaves one line ending before the END boundary.
const DECODE_BODY_ENCODED_SIZE_MAXIMUM = (BODY_STORAGE_MAXIMUM - LINE_FEED_SIZE) /
	base64.ENCODED_GROUP_SIZE * base64.ENCODED_GROUP_SIZE

// DECODED_SIZE_MAXIMUM is the largest unwrapped body fitting bounded encoded input.
const DECODED_SIZE_MAXIMUM = DECODE_BODY_ENCODED_SIZE_MAXIMUM /
	base64.ENCODED_GROUP_SIZE * base64.DECODED_GROUP_SIZE

// COUNT_HOLE_FIRST excludes an impossible one-byte block result.
const COUNT_HOLE_FIRST = bytes.SLICE_SIZE_MINIMUM + 1

// COUNT_HOLE_SECOND excludes an impossible two-byte block result.
const COUNT_HOLE_SECOND = COUNT_HOLE_FIRST + 1

// STATUS_OK means the operation completed.
const STATUS_OK = 0

// STATUS_NOT_FOUND means input contains no line-bound BEGIN marker.
const STATUS_NOT_FOUND = STATUS_OK + 1

// STATUS_INPUT_INVALID means a found block violates PEM or standard base64 grammar.
const STATUS_INPUT_INVALID = STATUS_NOT_FOUND + 1

// STATUS_OUTPUT_TOO_SMALL means caller byte storage cannot hold the result.
const STATUS_OUTPUT_TOO_SMALL = STATUS_INPUT_INVALID + 1

// STATUS_HEADERS_TOO_SMALL means caller header slots cannot hold parsed metadata.
const STATUS_HEADERS_TOO_SMALL = STATUS_OUTPUT_TOO_SMALL + 1

// STATUS_STORAGE_INVALID means encoded and decoded byte storage overlap.
const STATUS_STORAGE_INVALID = STATUS_HEADERS_TOO_SMALL + 1

// STATUS_BLOCK_INVALID means caller metadata cannot form canonical PEM.
const STATUS_BLOCK_INVALID = STATUS_STORAGE_INVALID + 1

// STATUS_BLOCK_TOO_LARGE means bounded fields combine into oversized encoded output.
const STATUS_BLOCK_TOO_LARGE = STATUS_BLOCK_INVALID + 1

// BYTE_MINIMUM is the first PEM octet.
const BYTE_MINIMUM = 0

// BYTE_MAXIMUM is the final PEM octet.
const BYTE_MAXIMUM = 255

// PATTERN_SIZE_MINIMUM is the shared boundary suffix size.
const PATTERN_SIZE_MINIMUM = len(BOUNDARY_SUFFIX)

// PATTERN_SIZE_MIDDLE is the END prefix size.
const PATTERN_SIZE_MIDDLE = len(END_PREFIX)

// PATTERN_SIZE_MAXIMUM is the BEGIN prefix size.
const PATTERN_SIZE_MAXIMUM = len(BEGIN_PREFIX)

// LINE_NEXT_MINIMUM is first position after one line-feed byte.
const LINE_NEXT_MINIMUM = bytes.SLICE_SIZE_MINIMUM + LINE_FEED_SIZE

// COLON_POSITION_MAXIMUM is final addressable encoded byte.
const COLON_POSITION_MAXIMUM = ENCODED_SIZE_MAXIMUM - 1

// Type is a borrowed PEM boundary label.
type Type []byte

// Type_Invariants leaves room for both boundary appearances.
func Type_Invariants(value Type, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, TYPE_SIZE_MAXIMUM).
		Ensure()
}

// Header_Key is one borrowed metadata name.
type Header_Key []byte

// Header_Key_Invariants bounds an independently supplied key before aggregate sizing.
func Header_Key_Invariants(value Header_Key, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Header_Value is one borrowed metadata value.
type Header_Value []byte

// Header_Value_Invariants bounds an independently supplied value before aggregate sizing.
func Header_Value_Invariants(value Header_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Header retains caller order because a bounded slice already supplies deterministic order.
type Header struct {
	// Key borrows caller or encoded-input storage.
	Key Header_Key
	// Value borrows caller or encoded-input storage.
	Value Header_Value
}

// Header_Invariants composes independently bounded metadata fields.
func Header_Invariants(value Header, namespace aver.Namespace) {
	Header_Key_Invariants(value.Key, namespace)
	Header_Value_Invariants(value.Value, namespace)
}

// Headers is caller-owned metadata input or decode output storage.
type Headers []Header

// Headers_Invariants derives slot count from the smallest encoded header line.
func Headers_Invariants(value Headers, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, HEADER_COUNT_MAXIMUM).
		Ensure()
}

// Data is decoded payload borrowing caller storage.
type Data []byte

// Data_Invariants follows the largest unwrapped body admitted by bounded input.
func Data_Invariants(value Data, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, DECODED_SIZE_MAXIMUM).
		Ensure()
}

// Block is one bounded PEM value without maps, strings, or owned storage.
type Block struct {
	// Type is repeated in both boundary lines.
	Type Type
	// Headers retains explicit caller order.
	Headers Headers
	// Data holds decoded body bytes.
	Data Data
}

// Block_Invariants composes every caller-owned collection boundary.
func Block_Invariants(value Block, namespace aver.Namespace) {
	Type_Invariants(value.Type, namespace)
	Headers_Invariants(value.Headers, namespace)
	Data_Invariants(value.Data, namespace)
}

// Encoded is bounded PEM input or writable output.
type Encoded []byte

// Encoded_Invariants follows the repository byte-slice boundary.
func Encoded_Invariants(value Encoded, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Begin_Source contains enough bytes to match one BEGIN prefix.
type Begin_Source []byte

// Begin_Source_Invariants excludes input too short to enter boundary parsing.
func Begin_Source_Invariants(value Begin_Source, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), BEGIN_PREFIX_SIZE, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Section_Source contains a complete BEGIN line and one following byte.
type Section_Source []byte

// Section_Source_Invariants excludes input that cannot enter section parsing.
func Section_Source_Invariants(value Section_Source, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SECTION_SOURCE_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Block_Source contains enough bytes for complete minimum PEM framing.
type Block_Source []byte

// Block_Source_Invariants excludes input shorter than one complete block.
func Block_Source_Invariants(value Block_Source, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), BLOCK_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Block_Output contains storage proven sufficient for minimum PEM framing.
type Block_Output []byte

// Block_Output_Invariants excludes output too short for a complete block.
func Block_Output_Invariants(value Block_Output, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), BLOCK_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Encode_Data is payload proven representable by bounded wrapped output.
type Encode_Data []byte

// Encode_Data_Invariants follows the encoder-specific payload maximum.
func Encode_Data_Invariants(value Encode_Data, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, ENCODE_SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Block_Count is exact successful output size.
type Block_Count int

// Block_Count_Invariants excludes refusal and partial framing counts.
func Block_Count_Invariants(value Block_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), BLOCK_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Decoded is caller-owned payload destination.
type Decoded []byte

// Decoded_Invariants follows the largest body possible in bounded encoded input.
func Decoded_Invariants(value Decoded, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, DECODED_SIZE_MAXIMUM).
		Ensure()
}

// Encoded_Count is exact required or written output bytes, or zero on refusal.
type Encoded_Count int

// Encoded_Count_Invariants excludes results shorter than complete empty framing.
func Encoded_Count_Invariants(value Encoded_Count, namespace aver.Namespace) {
	valid := value == 0 || int(value) >= BLOCK_SIZE_MINIMUM
	aver.Always(valid, "A nonempty encoded count contains complete block framing.")
	aver.Tree(value, namespace).
		Range_Holed_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM,
			COUNT_HOLE_FIRST, COUNT_HOLE_SECOND, COUNT_HOLE_SECOND, COUNT_HOLE_SECOND,
		).
		Ensure()
}

// Consumed_Count is input through the decoded END line, or zero on refusal.
type Consumed_Count int

// Consumed_Count_Invariants excludes successful counts shorter than empty framing.
func Consumed_Count_Invariants(value Consumed_Count, namespace aver.Namespace) {
	valid := value == 0 || int(value) >= BLOCK_SIZE_MINIMUM
	aver.Always(valid, "A nonzero consumed count contains complete block framing.")
	aver.Tree(value, namespace).
		Range_Holed_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM,
			COUNT_HOLE_FIRST, COUNT_HOLE_SECOND, COUNT_HOLE_SECOND, COUNT_HOLE_SECOND,
		).
		Ensure()
}

// Block_Valid reports canonical caller metadata without an allocating error.
type Block_Valid bool

// Block_Valid_Invariants reaches accepted and rejected caller blocks.
func Block_Valid_Invariants(value Block_Valid, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Caller metadata forms canonical PEM.").
		Ensure()
}

// Boolean is an internal parser decision with both outcomes tracked.
type Boolean bool

// Boolean_Invariants reaches positive and negative parser decisions.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A PEM parser decision is positive.").
		Ensure()
}

// Internal_Position is one zero-based encoded-storage boundary.
type Internal_Position int

// Internal_Position_Invariants keeps parser and writer positions bounded.
func Internal_Position_Invariants(value Internal_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Line_Next is first byte after line ending or encoded source end.
type Line_Next int

// Line_Next_Invariants excludes impossible zero continuation.
func Line_Next_Invariants(value Line_Next, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), LINE_NEXT_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Colon_Position is absent zero or addressable header separator byte.
type Colon_Position int

// Colon_Position_Invariants excludes position after encoded storage.
func Colon_Position_Invariants(value Colon_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, COLON_POSITION_MAXIMUM,
		).
		Ensure()
}

// Header_Count is parsed metadata slot count.
type Header_Count int

// Header_Count_Invariants keeps parsed metadata inside caller slot boundary.
func Header_Count_Invariants(value Header_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, HEADER_COUNT_MAXIMUM).
		Ensure()
}

// Decoded_Count is exact decoded body bytes.
type Decoded_Count int

// Decoded_Count_Invariants keeps body size inside decoded storage boundary.
func Decoded_Count_Invariants(value Decoded_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, DECODED_SIZE_MAXIMUM).
		Ensure()
}

// PEM_Byte is one untrusted encoded octet.
type PEM_Byte byte

// PEM_Byte_Invariants covers the complete octet domain.
func PEM_Byte_Invariants(value PEM_Byte, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), BYTE_MINIMUM, BYTE_MAXIMUM).
		Ensure()
}

// Pattern is one fixed PEM framing token.
type Pattern string

// Pattern_Invariants lists every token size accepted by text matching.
func Pattern_Invariants(value Pattern, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Int(
			len(value), PATTERN_SIZE_MINIMUM, PATTERN_SIZE_MIDDLE,
			PATTERN_SIZE_MAXIMUM,
		).
		Ensure()
}

// Size_Status reports caller block validation and aggregate bound failures.
type Size_Status uint8

// Size_Status_Invariants lists every size outcome.
func Size_Status_Invariants(value Size_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_BLOCK_INVALID),
			uint8(STATUS_BLOCK_TOO_LARGE),
		).
		Ensure()
}

// Encode_Status adds caller output and overlap failures to size outcomes.
type Encode_Status uint8

// Encode_Status_Invariants excludes decode-only statuses from the shared status range.
func Encode_Status_Invariants(value Encode_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_BLOCK_TOO_LARGE),
			uint8(STATUS_NOT_FOUND), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_HEADERS_TOO_SMALL),
		).
		Ensure()
}

// Decode_Status reports every bounded parser result.
type Decode_Status uint8

// Decode_Status_Invariants covers the contiguous decoder status domain.
func Decode_Status_Invariants(value Decode_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_STORAGE_INVALID),
		).
		Ensure()
}

// Encoded_Size reports exact caller storage after complete block validation.
func Encoded_Size(block Block) (count Encoded_Count, status Size_Status) {
	defer func() {
		Encoded_Count_Invariants(count, "Encoded_Size.count")
		Size_Status_Invariants(status, "Encoded_Size.status")
	}()
	Block_Invariants(block, "Encoded_Size.block")
	if !bool(block_valid(block)) {
		return 0, STATUS_BLOCK_INVALID
	}
	encoded_body_size := (len(block.Data) + base64.DECODED_GROUP_SIZE - 1) /
		base64.DECODED_GROUP_SIZE * base64.ENCODED_GROUP_SIZE
	body_line_count := bytes.SLICE_SIZE_MINIMUM
	if encoded_body_size > bytes.SLICE_SIZE_MINIMUM {
		body_line_count = (encoded_body_size + BASE64_LINE_ENCODED_SIZE - 1) /
			BASE64_LINE_ENCODED_SIZE
	}
	required := BLOCK_SIZE_MINIMUM + len(block.Type)*2 + encoded_body_size
	required += body_line_count
	if len(block.Headers) > bytes.SLICE_SIZE_MINIMUM {
		required += HEADER_SECTION_SEPARATOR_SIZE
	}
	for _, header := range block.Headers {
		header_size := len(header.Key) + len(HEADER_SEPARATOR) + len(header.Value)
		required += header_size + LINE_FEED_SIZE
	}
	if required > ENCODED_SIZE_MAXIMUM {
		return 0, STATUS_BLOCK_TOO_LARGE
	}
	return Encoded_Count(required), STATUS_OK
}

// Encode_Into validates every refusal before changing caller output.
func Encode_Into(
	destination Encoded, block Block,
) (count Encoded_Count, status Encode_Status) {
	defer func() {
		Encoded_Count_Invariants(count, "Encode_Into.count")
		Encode_Status_Invariants(status, "Encode_Into.status")
	}()
	Encoded_Invariants(destination, "Encode_Into.destination")
	Block_Invariants(block, "Encode_Into.block")
	required, size_status := Encoded_Size(block)
	if size_status == STATUS_BLOCK_INVALID {
		return 0, STATUS_BLOCK_INVALID
	}
	if size_status == STATUS_BLOCK_TOO_LARGE {
		return 0, STATUS_BLOCK_TOO_LARGE
	}
	if bytes.Overlap(bytes.Slice(destination), bytes.Slice(block.Type)) {
		return 0, STATUS_STORAGE_INVALID
	}
	if bytes.Overlap(bytes.Slice(destination), bytes.Slice(block.Data)) {
		return 0, STATUS_STORAGE_INVALID
	}
	for _, header := range block.Headers {
		if bytes.Overlap(bytes.Slice(destination), bytes.Slice(header.Key)) {
			return 0, STATUS_STORAGE_INVALID
		}
		if bytes.Overlap(bytes.Slice(destination), bytes.Slice(header.Value)) {
			return 0, STATUS_STORAGE_INVALID
		}
	}
	if len(destination) < int(required) {
		return 0, STATUS_OUTPUT_TOO_SMALL
	}
	encode_unchecked(
		Block_Output(destination), block.Type, block.Headers,
		Encode_Data(block.Data), Block_Count(required),
	)
	return required, STATUS_OK
}

// Decode_Into returns one complete block and the input consumed through its END line.
func Decode_Into(
	destination Decoded, headers Headers, source Encoded,
) (block Block, consumed Consumed_Count, status Decode_Status) {
	defer func() {
		Block_Invariants(block, "Decode_Into.block")
		Consumed_Count_Invariants(consumed, "Decode_Into.consumed")
		Decode_Status_Invariants(status, "Decode_Into.status")
	}()
	Decoded_Invariants(destination, "Decode_Into.destination")
	Headers_Invariants(headers, "Decode_Into.headers")
	Encoded_Invariants(source, "Decode_Into.source")
	if bytes.Overlap(bytes.Slice(destination), bytes.Slice(source)) {
		return Block{}, 0, STATUS_STORAGE_INVALID
	}
	search_start := Internal_Position(bytes.SLICE_SIZE_MINIMUM)
	malformed_seen := false
	for int(search_start) <= len(source) {
		type_start, type_end, content_start, found, begin_valid := find_begin(
			source, search_start,
		)
		if !bool(found) {
			if malformed_seen {
				return Block{}, 0, STATUS_INPUT_INVALID
			}
			return Block{}, 0, STATUS_NOT_FOUND
		}
		if !bool(begin_valid) {
			malformed_seen = true
			search_start = content_start
			continue
		}
		body_start, body_end, end_next, header_count, sections_valid := parse_sections(
			Section_Source(source), content_start, type_start, type_end,
		)
		if !bool(sections_valid) {
			malformed_seen = true
			search_start = content_start
			continue
		}
		block_source := Block_Source(source)
		decoded_size, body_shape_valid := body_decoded_size(
			block_source, body_start, body_end,
		)
		if !bool(body_shape_valid) {
			malformed_seen = true
			search_start = content_start
			continue
		}
		if !bool(body_encoding_valid(block_source, body_start, body_end)) {
			malformed_seen = true
			search_start = content_start
			continue
		}
		if int(header_count) > len(headers) {
			return Block{}, 0, STATUS_HEADERS_TOO_SMALL
		}
		if int(decoded_size) > len(destination) {
			return Block{}, 0, STATUS_OUTPUT_TOO_SMALL
		}
		fill_headers(headers, block_source, content_start, header_count)
		decode_body(destination, block_source, body_start, body_end, decoded_size)
		block.Type = Type(source[type_start:type_end])
		block.Headers = headers[:header_count]
		block.Data = Data(destination[:decoded_size])
		return block, Consumed_Count(end_next), STATUS_OK
	}
	if malformed_seen {
		return Block{}, 0, STATUS_INPUT_INVALID
	}
	return Block{}, 0, STATUS_NOT_FOUND
}

func block_valid(block Block) (valid Block_Valid) {
	defer func() { Block_Valid_Invariants(valid, "block_valid.valid") }()
	Block_Invariants(block, "block_valid.block")
	if !metadata_valid(Encoded(block.Type), true) {
		return false
	}
	for _, header := range block.Headers {
		Header_Invariants(header, "block_valid.header")
		if !metadata_valid(Encoded(header.Key), true) {
			return false
		}
		if !metadata_valid(Encoded(header.Value), false) {
			return false
		}
	}
	return true
}

func metadata_valid(value Encoded, colon_invalid Boolean) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "metadata_valid.valid") }()
	Encoded_Invariants(value, "metadata_valid.value")
	Boolean_Invariants(colon_invalid, "metadata_valid.colon_invalid")
	if len(value) > bytes.SLICE_SIZE_MINIMUM {
		if horizontal_space(PEM_Byte(value[0])) {
			return false
		}
		if horizontal_space(PEM_Byte(value[len(value)-1])) {
			return false
		}
	}
	for _, value_byte := range value {
		if value_byte == CARRIAGE_RETURN {
			return false
		}
		if value_byte == LINE_FEED {
			return false
		}
		if colon_invalid {
			if value_byte == HEADER_KEY_SEPARATOR {
				return false
			}
		}
	}
	return true
}

func horizontal_space(value PEM_Byte) (space Boolean) {
	defer func() { Boolean_Invariants(space, "horizontal_space.space") }()
	PEM_Byte_Invariants(value, "horizontal_space.value")
	if value == PEM_Byte(SPACE) {
		return true
	}
	return value == PEM_Byte(HORIZONTAL_TAB)
}

func encode_unchecked(
	destination Block_Output, block_type Type, headers Headers,
	data Encode_Data, required Block_Count,
) {
	Block_Output_Invariants(destination, "encode_unchecked.destination")
	Type_Invariants(block_type, "encode_unchecked.block_type")
	Headers_Invariants(headers, "encode_unchecked.headers")
	Encode_Data_Invariants(data, "encode_unchecked.data")
	Block_Count_Invariants(required, "encode_unchecked.required")
	position := copy(destination, BEGIN_PREFIX)
	position += copy(destination[position:], block_type)
	position += copy(destination[position:], BOUNDARY_SUFFIX)
	destination[position] = LINE_FEED
	position++
	for _, header := range headers {
		position += copy(destination[position:], header.Key)
		position += copy(destination[position:], HEADER_SEPARATOR)
		position += copy(destination[position:], header.Value)
		destination[position] = LINE_FEED
		position++
	}
	if len(headers) > bytes.SLICE_SIZE_MINIMUM {
		destination[position] = LINE_FEED
		position++
	}
	position = int(encode_body(destination, Internal_Position(position), data))
	position += copy(destination[position:], END_PREFIX)
	position += copy(destination[position:], block_type)
	position += copy(destination[position:], BOUNDARY_SUFFIX)
	destination[position] = LINE_FEED
	position++
	aver.Always(position == int(required), "PEM encoding writes its exact reported size.")
}

func encode_body(
	destination Block_Output, destination_position Internal_Position, data Encode_Data,
) (next_position Internal_Position) {
	defer func() {
		Internal_Position_Invariants(next_position, "encode_body.next_position")
	}()
	Block_Output_Invariants(destination, "encode_body.destination")
	Internal_Position_Invariants(destination_position, "encode_body.destination_position")
	Encode_Data_Invariants(data, "encode_body.data")
	next := destination_position
	encoding := base64.Standard_Encoding()
	for source_position := bytes.SLICE_SIZE_MINIMUM; source_position < len(data); {
		source_size := len(data) - source_position
		if source_size > BASE64_LINE_SOURCE_SIZE {
			source_size = BASE64_LINE_SOURCE_SIZE
		}
		encoded_count, size_status := base64.Encoded_Size(
			encoding, base64.Source_Count(source_size),
		)
		aver.Always(
			size_status == base64.STATUS_OK,
			"Standard base64 sizing succeeds.",
		)
		line_end := next + Internal_Position(encoded_count)
		count, encode_status := base64.Encode_Into(
			base64.Encoded(destination[next:line_end]),
			base64.Source(data[source_position:source_position+source_size]),
			encoding,
		)
		aver.Always(
			encode_status == base64.STATUS_OK,
			"Bounded base64 line encoding succeeds.",
		)
		aver.Always(
			count == encoded_count,
			"Base64 line encoding writes its exact size.",
		)
		next = line_end
		destination[next] = LINE_FEED
		next++
		source_position += source_size
	}
	return next
}

func find_begin(
	source Encoded, start Internal_Position,
) (
	type_start Internal_Position, type_end Internal_Position,
	content_start Internal_Position,
	found Boolean, valid Boolean,
) {
	defer func() {
		Internal_Position_Invariants(type_start, "find_begin.type_start")
		Internal_Position_Invariants(type_end, "find_begin.type_end")
		Internal_Position_Invariants(content_start, "find_begin.content_start")
		Boolean_Invariants(found, "find_begin.found")
		Boolean_Invariants(valid, "find_begin.valid")
	}()
	Encoded_Invariants(source, "find_begin.source")
	Internal_Position_Invariants(start, "find_begin.start")
	for position := start; int(position)+len(BEGIN_PREFIX) <= len(source); position++ {
		if position > bytes.SLICE_SIZE_MINIMUM {
			if byte(source[position-1]) != LINE_FEED {
				continue
			}
		}
		begin_source := Begin_Source(source)
		if !text_at(begin_source, position, BEGIN_PREFIX) {
			continue
		}
		resume := position + Internal_Position(len(BEGIN_PREFIX))
		line_end, next := line_bounds(begin_source, position)
		trimmed_end := trim_right(begin_source, line_end)
		label_start := position + Internal_Position(len(BEGIN_PREFIX))
		if int(trimmed_end-label_start) < BOUNDARY_SUFFIX_SIZE {
			return 0, 0, resume, true, false
		}
		label_end := trimmed_end - Internal_Position(BOUNDARY_SUFFIX_SIZE)
		if !text_at(begin_source, label_end, BOUNDARY_SUFFIX) {
			return 0, 0, resume, true, false
		}
		if int(label_end-label_start) > TYPE_SIZE_MAXIMUM {
			return 0, 0, resume, true, false
		}
		if !metadata_valid(Encoded(source[label_start:label_end]), true) {
			return 0, 0, resume, true, false
		}
		if int(next) == len(source) {
			return 0, 0, resume, true, false
		}
		return label_start, label_end, Internal_Position(next), true, true
	}
	return start, start, start, false, false
}

func parse_sections(
	source Section_Source, content_start Internal_Position,
	type_start Internal_Position, type_end Internal_Position,
) (
	body_start Internal_Position, body_end Internal_Position,
	end_next Internal_Position, header_count Header_Count, valid Boolean,
) {
	defer func() {
		Internal_Position_Invariants(body_start, "parse_sections.body_start")
		Internal_Position_Invariants(body_end, "parse_sections.body_end")
		Internal_Position_Invariants(end_next, "parse_sections.end_next")
		Header_Count_Invariants(header_count, "parse_sections.header_count")
		Boolean_Invariants(valid, "parse_sections.valid")
	}()
	Section_Source_Invariants(source, "parse_sections.source")
	Internal_Position_Invariants(content_start, "parse_sections.content_start")
	Internal_Position_Invariants(type_start, "parse_sections.type_start")
	Internal_Position_Invariants(type_end, "parse_sections.type_end")
	position := content_start
	body_position := position
	headers_active := true
	count := Header_Count(bytes.SLICE_SIZE_MINIMUM)
	for int(position) < len(source) {
		begin_source := Begin_Source(source)
		line_end, next := line_bounds(begin_source, position)
		if end_line_valid(source, position, line_end, type_start, type_end) {
			return body_position, position, Internal_Position(next), count, true
		}
		if headers_active {
			_, colon_found := line_colon(source, position, line_end)
			if colon_found {
				count++
				body_position = Internal_Position(next)
				position = Internal_Position(next)
				continue
			}
			headers_active = false
			if trim_right(begin_source, line_end) == position {
				body_position = Internal_Position(next)
				position = Internal_Position(next)
				continue
			}
			body_position = position
		}
		if Internal_Position(next) == position {
			break
		}
		position = Internal_Position(next)
	}
	return content_start, content_start, content_start, 0, false
}

func fill_headers(
	headers Headers, source Block_Source, content_start Internal_Position,
	header_count Header_Count,
) {
	Headers_Invariants(headers, "fill_headers.headers")
	Block_Source_Invariants(source, "fill_headers.source")
	Internal_Position_Invariants(content_start, "fill_headers.content_start")
	Header_Count_Invariants(header_count, "fill_headers.header_count")
	position := content_start
	for index := bytes.SLICE_SIZE_MINIMUM; index < int(header_count); index++ {
		line_end, next := line_bounds(Begin_Source(source), position)
		colon, found := line_colon(Section_Source(source), position, line_end)
		aver.Always(found, "A counted PEM header retains its separator.")
		key_start := position
		key_end := Internal_Position(colon)
		for key_start < key_end && header_space(PEM_Byte(source[key_start])) {
			key_start++
		}
		for key_end > key_start && header_space(PEM_Byte(source[key_end-1])) {
			key_end--
		}
		value_start := Internal_Position(colon) + 1
		value_end := line_end
		for value_start < value_end && header_space(PEM_Byte(source[value_start])) {
			value_start++
		}
		for value_end > value_start && header_space(PEM_Byte(source[value_end-1])) {
			value_end--
		}
		headers[index] = Header{
			Key:   Header_Key(source[key_start:key_end]),
			Value: Header_Value(source[value_start:value_end]),
		}
		position = Internal_Position(next)
	}
}

func header_space(value PEM_Byte) (space Boolean) {
	defer func() { Boolean_Invariants(space, "header_space.space") }()
	PEM_Byte_Invariants(value, "header_space.value")
	return value == PEM_Byte(SPACE) ||
		value >= PEM_Byte(HORIZONTAL_TAB) && value <= PEM_Byte(CARRIAGE_RETURN)
}

func body_decoded_size(
	source Block_Source, body_start Internal_Position, body_end Internal_Position,
) (decoded_size Decoded_Count, valid Boolean) {
	defer func() {
		Decoded_Count_Invariants(decoded_size, "body_decoded_size.decoded_size")
		Boolean_Invariants(valid, "body_decoded_size.valid")
	}()
	Block_Source_Invariants(source, "body_decoded_size.source")
	Internal_Position_Invariants(body_start, "body_decoded_size.body_start")
	Internal_Position_Invariants(body_end, "body_decoded_size.body_end")
	encoded_count := bytes.SLICE_SIZE_MINIMUM
	var previous byte
	var final byte
	for position := body_start; position < body_end; position++ {
		value := source[position]
		if body_space(PEM_Byte(value)) {
			continue
		}
		previous = final
		final = value
		encoded_count++
	}
	if encoded_count%base64.ENCODED_GROUP_SIZE != bytes.SLICE_SIZE_MINIMUM {
		return 0, false
	}
	padding_count := bytes.SLICE_SIZE_MINIMUM
	if encoded_count > bytes.SLICE_SIZE_MINIMUM {
		if final == BASE64_PADDING {
			padding_count++
			if previous == BASE64_PADDING {
				padding_count++
			}
		}
	}
	decoded := encoded_count/base64.ENCODED_GROUP_SIZE*base64.DECODED_GROUP_SIZE -
		padding_count
	if decoded > DECODED_SIZE_MAXIMUM {
		return 0, false
	}
	return Decoded_Count(decoded), true
}

func body_encoding_valid(
	source Block_Source, body_start Internal_Position, body_end Internal_Position,
) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "body_encoding_valid.valid") }()
	Block_Source_Invariants(source, "body_encoding_valid.source")
	Internal_Position_Invariants(body_start, "body_encoding_valid.body_start")
	Internal_Position_Invariants(body_end, "body_encoding_valid.body_end")
	encoding := base64.Standard_Encoding()
	var quantum [base64.ENCODED_GROUP_SIZE]byte
	var decoded [base64.DECODED_GROUP_SIZE]byte
	quantum_count := bytes.SLICE_SIZE_MINIMUM
	padding_seen := false
	for position := body_start; position < body_end; position++ {
		value := source[position]
		if bool(body_space(PEM_Byte(value))) {
			continue
		}
		if padding_seen {
			return false
		}
		quantum[quantum_count] = value
		quantum_count++
		if quantum_count != len(quantum) {
			continue
		}
		_, status := base64.Decode_Into(
			base64.Decoded(decoded[:]), base64.Encoded(quantum[:]), encoding,
		)
		if status != base64.STATUS_OK {
			return false
		}
		for _, symbol := range quantum {
			if symbol == BASE64_PADDING {
				padding_seen = true
			}
		}
		quantum_count = bytes.SLICE_SIZE_MINIMUM
	}
	return true
}

func decode_body(
	destination Decoded, source Block_Source,
	body_start Internal_Position, body_end Internal_Position,
	decoded_size Decoded_Count,
) {
	Decoded_Invariants(destination, "decode_body.destination")
	Block_Source_Invariants(source, "decode_body.source")
	Internal_Position_Invariants(body_start, "decode_body.body_start")
	Internal_Position_Invariants(body_end, "decode_body.body_end")
	Decoded_Count_Invariants(decoded_size, "decode_body.decoded_size")
	encoding := base64.Standard_Encoding()
	var quantum [base64.ENCODED_GROUP_SIZE]byte
	quantum_count := bytes.SLICE_SIZE_MINIMUM
	destination_position := bytes.SLICE_SIZE_MINIMUM
	for position := body_start; position < body_end; position++ {
		value := source[position]
		if body_space(PEM_Byte(value)) {
			continue
		}
		quantum[quantum_count] = value
		quantum_count++
		if quantum_count != len(quantum) {
			continue
		}
		count, status := base64.Decode_Into(
			base64.Decoded(destination[destination_position:]),
			base64.Encoded(quantum[:]), encoding,
		)
		aver.Always(
			status == base64.STATUS_OK,
			"Prevalidated PEM body quantum decodes.",
		)
		destination_position += int(count)
		quantum_count = bytes.SLICE_SIZE_MINIMUM
	}
	aver.Always(
		quantum_count == bytes.SLICE_SIZE_MINIMUM,
		"Prevalidated PEM body ends on complete quantum.",
	)
	aver.Always(
		destination_position == int(decoded_size),
		"PEM body decoding writes exact validated size.",
	)
}

func line_bounds(
	source Begin_Source, start Internal_Position,
) (line_end Internal_Position, next Line_Next) {
	defer func() {
		Internal_Position_Invariants(line_end, "line_bounds.line_end")
		Line_Next_Invariants(next, "line_bounds.next")
	}()
	Begin_Source_Invariants(source, "line_bounds.source")
	Internal_Position_Invariants(start, "line_bounds.start")
	for position := start; int(position) < len(source); position++ {
		if source[position] == LINE_FEED {
			return position, Line_Next(position + Internal_Position(LINE_FEED_SIZE))
		}
	}
	return Internal_Position(len(source)), Line_Next(len(source))
}

func trim_right(source Begin_Source, end Internal_Position) (trimmed Internal_Position) {
	defer func() { Internal_Position_Invariants(trimmed, "trim_right.trimmed") }()
	Begin_Source_Invariants(source, "trim_right.source")
	Internal_Position_Invariants(end, "trim_right.end")
	position := end
	for position > bytes.SLICE_SIZE_MINIMUM {
		value := byte(source[position-1])
		if value == CARRIAGE_RETURN {
			position--
			continue
		}
		if horizontal_space(PEM_Byte(value)) {
			position--
			continue
		}
		break
	}
	return position
}

func line_colon(
	source Section_Source, start Internal_Position, end Internal_Position,
) (colon Colon_Position, found Boolean) {
	defer func() {
		Colon_Position_Invariants(colon, "line_colon.colon")
		Boolean_Invariants(found, "line_colon.found")
	}()
	Section_Source_Invariants(source, "line_colon.source")
	Internal_Position_Invariants(start, "line_colon.start")
	Internal_Position_Invariants(end, "line_colon.end")
	for position := start; position < end; position++ {
		if source[position] == HEADER_KEY_SEPARATOR {
			return Colon_Position(position), true
		}
	}
	return 0, false
}

func end_line_valid(
	source Section_Source, line_start Internal_Position, line_end Internal_Position,
	type_start Internal_Position, type_end Internal_Position,
) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "end_line_valid.valid") }()
	Section_Source_Invariants(source, "end_line_valid.source")
	Internal_Position_Invariants(line_start, "end_line_valid.line_start")
	Internal_Position_Invariants(line_end, "end_line_valid.line_end")
	Internal_Position_Invariants(type_start, "end_line_valid.type_start")
	Internal_Position_Invariants(type_end, "end_line_valid.type_end")
	begin_source := Begin_Source(source)
	trimmed_end := trim_right(begin_source, line_end)
	type_size := int(type_end - type_start)
	expected_size := len(END_PREFIX) + type_size + BOUNDARY_SUFFIX_SIZE
	if int(trimmed_end-line_start) != expected_size {
		return false
	}
	if !text_at(begin_source, line_start, END_PREFIX) {
		return false
	}
	line_type_start := line_start + Internal_Position(len(END_PREFIX))
	for index := bytes.SLICE_SIZE_MINIMUM; index < type_size; index++ {
		if source[line_type_start+Internal_Position(index)] !=
			source[type_start+Internal_Position(index)] {
			return false
		}
	}
	return text_at(
		begin_source, line_type_start+Internal_Position(type_size), BOUNDARY_SUFFIX,
	)
}

func text_at(source Begin_Source, start Internal_Position, text Pattern) (present Boolean) {
	defer func() { Boolean_Invariants(present, "text_at.present") }()
	Begin_Source_Invariants(source, "text_at.source")
	Internal_Position_Invariants(start, "text_at.start")
	Pattern_Invariants(text, "text_at.text")
	if start < bytes.SLICE_SIZE_MINIMUM {
		return false
	}
	if len(source)-int(start) < len(text) {
		return false
	}
	for index := range len(text) {
		if source[start+Internal_Position(index)] != text[index] {
			return false
		}
	}
	return true
}

func body_space(value PEM_Byte) (space Boolean) {
	defer func() { Boolean_Invariants(space, "body_space.space") }()
	PEM_Byte_Invariants(value, "body_space.value")
	if value == PEM_Byte(LINE_FEED) {
		return true
	}
	if value == PEM_Byte(CARRIAGE_RETURN) {
		return true
	}
	return horizontal_space(value)
}
