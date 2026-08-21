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

// BOUNDARY_SUFFIX_SIZE derives both boundary widths from their shared token.
const BOUNDARY_SUFFIX_SIZE = len(BOUNDARY_SUFFIX)

// BEGIN_LINE_FIXED_SIZE excludes only the caller block type.
const BEGIN_LINE_FIXED_SIZE = len(BEGIN_PREFIX) + BOUNDARY_SUFFIX_SIZE + LINE_FEED_SIZE

// END_LINE_FIXED_SIZE excludes only the caller block type.
const END_LINE_FIXED_SIZE = len(END_PREFIX) + BOUNDARY_SUFFIX_SIZE + LINE_FEED_SIZE

// BLOCK_SIZE_MINIMUM is an empty block with an empty type.
const BLOCK_SIZE_MINIMUM = BEGIN_LINE_FIXED_SIZE + END_LINE_FIXED_SIZE

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
	encode_unchecked(destination, block.Type, block.Headers, block.Data, required)
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
	search_start := bytes.SLICE_SIZE_MINIMUM
	malformed_seen := false
	for search_start <= len(source) {
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
			search_start = int(content_start)
			continue
		}
		body_start, body_end, end_next, header_count, sections_valid := parse_sections(
			source, content_start, type_start, type_end,
		)
		if !bool(sections_valid) {
			malformed_seen = true
			search_start = int(content_start)
			continue
		}
		decoded_size, body_shape_valid := body_decoded_size(source, body_start, body_end)
		if !bool(body_shape_valid) {
			malformed_seen = true
			search_start = int(content_start)
			continue
		}
		if !bool(body_encoding_valid(source, body_start, body_end)) {
			malformed_seen = true
			search_start = int(content_start)
			continue
		}
		if header_count > len(headers) {
			return Block{}, 0, STATUS_HEADERS_TOO_SMALL
		}
		if decoded_size > len(destination) {
			return Block{}, 0, STATUS_OUTPUT_TOO_SMALL
		}
		fill_headers(headers, source, content_start, header_count)
		decode_body(destination, source, body_start, body_end, decoded_size)
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
	if !metadata_valid(block.Type, true) {
		return false
	}
	for _, header := range block.Headers {
		Header_Invariants(header, "block_valid.header")
		if !metadata_valid(header.Key, true) {
			return false
		}
		if !metadata_valid(header.Value, false) {
			return false
		}
	}
	return true
}

func metadata_valid[Text ~[]byte](
	value Text, colon_invalid Boolean,
) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "metadata_valid.valid") }()
	Boolean_Invariants(colon_invalid, "metadata_valid.colon_invalid")
	if len(value) > bytes.SLICE_SIZE_MINIMUM {
		if horizontal_space(byte(value[0])) {
			return false
		}
		if horizontal_space(byte(value[len(value)-1])) {
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

func horizontal_space[Value ~byte](value Value) (space Boolean) {
	defer func() { Boolean_Invariants(space, "horizontal_space.space") }()
	if byte(value) == SPACE {
		return true
	}
	return byte(value) == HORIZONTAL_TAB
}

func encode_unchecked[
	Destination ~[]byte, Type_Bytes ~[]byte, Header_Values ~[]Header,
	Data_Bytes ~[]byte, Required ~int,
](
	destination Destination, block_type Type_Bytes, headers Header_Values,
	data Data_Bytes, required Required,
) {
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
	position = encode_body(destination, position, data)
	position += copy(destination[position:], END_PREFIX)
	position += copy(destination[position:], block_type)
	position += copy(destination[position:], BOUNDARY_SUFFIX)
	destination[position] = LINE_FEED
	position++
	aver.Always(position == int(required), "PEM encoding writes its exact reported size.")
}

func encode_body[
	Destination ~[]byte, Position ~int, Data_Bytes ~[]byte,
](
	destination Destination, destination_position Position, data Data_Bytes,
) (next_position Position) {
	next := int(destination_position)
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
		line_end := next + int(encoded_count)
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
	return Position(next)
}

func find_begin[Source_Bytes ~[]byte, Position ~int](
	source Source_Bytes, start Position,
) (
	type_start Position, type_end Position, content_start Position,
	found Boolean, valid Boolean,
) {
	defer func() {
		Boolean_Invariants(found, "find_begin.found")
		Boolean_Invariants(valid, "find_begin.valid")
	}()
	for position := int(start); position+len(BEGIN_PREFIX) <= len(source); position++ {
		if position > bytes.SLICE_SIZE_MINIMUM {
			if byte(source[position-1]) != LINE_FEED {
				continue
			}
		}
		if !text_at(source, position, BEGIN_PREFIX) {
			continue
		}
		resume := Position(position + len(BEGIN_PREFIX))
		line_end, next := line_bounds(source, position)
		trimmed_end := trim_right(source, line_end)
		label_start := position + len(BEGIN_PREFIX)
		if trimmed_end-label_start < BOUNDARY_SUFFIX_SIZE {
			return 0, 0, resume, true, false
		}
		label_end := trimmed_end - BOUNDARY_SUFFIX_SIZE
		if !text_at(source, label_end, BOUNDARY_SUFFIX) {
			return 0, 0, resume, true, false
		}
		if label_end-label_start > TYPE_SIZE_MAXIMUM {
			return 0, 0, resume, true, false
		}
		if !metadata_valid(source[label_start:label_end], true) {
			return 0, 0, resume, true, false
		}
		if int(next) == len(source) {
			return 0, 0, resume, true, false
		}
		return Position(label_start), Position(label_end), Position(next), true, true
	}
	return 0, 0, 0, false, false
}

func parse_sections[Source_Bytes ~[]byte, Position ~int](
	source Source_Bytes, content_start Position, type_start Position, type_end Position,
) (
	body_start Position, body_end Position, end_next Position,
	header_count Position, valid Boolean,
) {
	defer func() { Boolean_Invariants(valid, "parse_sections.valid") }()
	position := int(content_start)
	body_position := position
	headers_active := true
	count := bytes.SLICE_SIZE_MINIMUM
	for position < len(source) {
		line_end, next := line_bounds(source, position)
		if end_line_valid(source, position, int(line_end), int(type_start), int(type_end)) {
			return Position(body_position), Position(position),
				Position(next), Position(count), true
		}
		if headers_active {
			_, colon_found := line_colon(source, position, int(line_end))
			if colon_found {
				count++
				body_position = int(next)
				position = int(next)
				continue
			}
			headers_active = false
			if trim_right(source, line_end) == position {
				body_position = int(next)
				position = int(next)
				continue
			}
			body_position = position
		}
		if int(next) == position {
			break
		}
		position = int(next)
	}
	return 0, 0, 0, 0, false
}

func fill_headers[
	Header_Storage ~[]Header, Source_Bytes ~[]byte, Position ~int,
](
	headers Header_Storage, source Source_Bytes, content_start Position,
	header_count Position,
) {
	position := int(content_start)
	for index := bytes.SLICE_SIZE_MINIMUM; index < int(header_count); index++ {
		line_end, next := line_bounds(source, position)
		colon, found := line_colon(source, position, int(line_end))
		aver.Always(found, "A counted PEM header retains its separator.")
		key := bytes.Trim_Space(bytes.Slice(source[position:int(colon)]))
		value := bytes.Trim_Space(bytes.Slice(source[int(colon)+1 : int(line_end)]))
		headers[index] = Header{
			Key: Header_Key(key), Value: Header_Value(value),
		}
		position = int(next)
	}
}

func body_decoded_size[Source_Bytes ~[]byte, Position ~int](
	source Source_Bytes, body_start Position, body_end Position,
) (decoded_size Position, valid Boolean) {
	defer func() { Boolean_Invariants(valid, "body_decoded_size.valid") }()
	encoded_count := bytes.SLICE_SIZE_MINIMUM
	var previous byte
	var final byte
	for position := int(body_start); position < int(body_end); position++ {
		value := byte(source[position])
		if body_space(value) {
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
	return Position(decoded), true
}

func body_encoding_valid[Source_Bytes ~[]byte, Position ~int](
	source Source_Bytes, body_start Position, body_end Position,
) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "body_encoding_valid.valid") }()
	encoding := base64.Standard_Encoding()
	var quantum [base64.ENCODED_GROUP_SIZE]byte
	var decoded [base64.DECODED_GROUP_SIZE]byte
	quantum_count := bytes.SLICE_SIZE_MINIMUM
	padding_seen := false
	for position := int(body_start); position < int(body_end); position++ {
		value := byte(source[position])
		if bool(body_space(value)) {
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

func decode_body[
	Destination ~[]byte, Source_Bytes ~[]byte, Position ~int,
](
	destination Destination, source Source_Bytes,
	body_start Position, body_end Position, decoded_size Position,
) {
	encoding := base64.Standard_Encoding()
	var quantum [base64.ENCODED_GROUP_SIZE]byte
	quantum_count := bytes.SLICE_SIZE_MINIMUM
	destination_position := bytes.SLICE_SIZE_MINIMUM
	for position := int(body_start); position < int(body_end); position++ {
		value := byte(source[position])
		if body_space(value) {
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

func line_bounds[Source_Bytes ~[]byte, Position ~int](
	source Source_Bytes, start Position,
) (line_end Position, next Position) {
	for position := int(start); position < len(source); position++ {
		if byte(source[position]) == LINE_FEED {
			return Position(position), Position(position + LINE_FEED_SIZE)
		}
	}
	return Position(len(source)), Position(len(source))
}

func trim_right[Source_Bytes ~[]byte, Position ~int](
	source Source_Bytes, end Position,
) (trimmed Position) {
	position := int(end)
	for position > bytes.SLICE_SIZE_MINIMUM {
		value := byte(source[position-1])
		if value == CARRIAGE_RETURN {
			position--
			continue
		}
		if horizontal_space(value) {
			position--
			continue
		}
		break
	}
	return Position(position)
}

func line_colon[Source_Bytes ~[]byte, Position ~int](
	source Source_Bytes, start Position, end Position,
) (colon Position, found Boolean) {
	defer func() { Boolean_Invariants(found, "line_colon.found") }()
	for position := int(start); position < int(end); position++ {
		if byte(source[position]) == HEADER_KEY_SEPARATOR {
			return Position(position), true
		}
	}
	return 0, false
}

func end_line_valid[Source_Bytes ~[]byte, Position ~int](
	source Source_Bytes, line_start Position, line_end Position,
	type_start Position, type_end Position,
) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "end_line_valid.valid") }()
	trimmed_end := int(trim_right(source, line_end))
	type_size := int(type_end) - int(type_start)
	expected_size := len(END_PREFIX) + type_size + BOUNDARY_SUFFIX_SIZE
	if trimmed_end-int(line_start) != expected_size {
		return false
	}
	if !text_at(source, line_start, END_PREFIX) {
		return false
	}
	line_type_start := int(line_start) + len(END_PREFIX)
	for index := bytes.SLICE_SIZE_MINIMUM; index < type_size; index++ {
		if byte(source[line_type_start+index]) != byte(source[int(type_start)+index]) {
			return false
		}
	}
	return text_at(source, line_type_start+type_size, BOUNDARY_SUFFIX)
}

func text_at[Source_Bytes ~[]byte, Position ~int, Text ~string](
	source Source_Bytes, start Position, text Text,
) (present Boolean) {
	defer func() { Boolean_Invariants(present, "text_at.present") }()
	if int(start) < bytes.SLICE_SIZE_MINIMUM {
		return false
	}
	if len(source)-int(start) < len(text) {
		return false
	}
	for index := range len(text) {
		if byte(source[int(start)+index]) != text[index] {
			return false
		}
	}
	return true
}

func body_space[Value ~byte](value Value) (space Boolean) {
	defer func() { Boolean_Invariants(space, "body_space.space") }()
	if byte(value) == LINE_FEED {
		return true
	}
	if byte(value) == CARRIAGE_RETURN {
		return true
	}
	return horizontal_space(value)
}
