// Package bytes manipulates byte slices. It ports the Go standard library bytes package with
// repository names and bounded collection domains.
package bytes

import (
	"errors"
	"iter"

	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/io"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/strings"
	"local/james-orcales/shared/unicode/ucd"
	"local/james-orcales/shared/unicode/utf8"
)

// Error_Too_Large is the panic value for a result above SLICE_SIZE_MAXIMUM.
var Error_Too_Large = errors.New("bytes: result too large")

// SLICE_SIZE_MINIMUM is the size of an empty Slice.
const SLICE_SIZE_MINIMUM = utf8.SEQUENCE_SIZE_MINIMUM

// SLICE_SIZE_MAXIMUM bounds each Slice that the package owns or reads.
const SLICE_SIZE_MAXIMUM = utf8.SEQUENCE_SIZE_MAXIMUM

// TEXT_SIZE_MINIMUM is the size of empty Text.
const TEXT_SIZE_MINIMUM = SLICE_SIZE_MINIMUM

// TEXT_SIZE_MAXIMUM bounds text that defines a character or cut set.
const TEXT_SIZE_MAXIMUM = SLICE_SIZE_MAXIMUM

// SLICES_COUNT_MINIMUM is the count in an absent split result.
const SLICES_COUNT_MINIMUM = SLICE_SIZE_MINIMUM

// SLICES_COUNT_MAXIMUM includes both sides of a separator at every input byte.
const SLICES_COUNT_MAXIMUM = SLICE_SIZE_MAXIMUM + 1

// FIELDS_COUNT_MINIMUM is the field count of empty input.
const FIELDS_COUNT_MINIMUM = SLICES_COUNT_MINIMUM

// FIELDS_COUNT_MAXIMUM alternates one-byte fields and one-byte separators.
const FIELDS_COUNT_MAXIMUM = SLICES_COUNT_MAXIMUM / 2

// CHARACTERS_COUNT_MINIMUM is the character count of empty input.
const CHARACTERS_COUNT_MINIMUM = SLICE_SIZE_MINIMUM

// CHARACTERS_COUNT_MAXIMUM occurs when each input byte names one character.
const CHARACTERS_COUNT_MAXIMUM = SLICE_SIZE_MAXIMUM

// INDEX_ABSENT reports that a search found no match.
const INDEX_ABSENT = -1

// INDEX_MAXIMUM is the final byte index in the largest Slice.
const INDEX_MAXIMUM = SLICE_SIZE_MAXIMUM - 1

// BOUNDARY_INDEX_MAXIMUM is the boundary after the final byte.
const BOUNDARY_INDEX_MAXIMUM = SLICE_SIZE_MAXIMUM

// COUNT_VALUE_MINIMUM is the smallest match count.
const COUNT_VALUE_MINIMUM = SLICES_COUNT_MINIMUM

// COUNT_VALUE_MAXIMUM includes both sides of every one-byte character.
const COUNT_VALUE_MAXIMUM = SLICES_COUNT_MAXIMUM

// LIMIT_MINIMUM requests all split results.
const LIMIT_MINIMUM = -1

// LIMIT_MAXIMUM bounds a requested result count.
const LIMIT_MAXIMUM = SLICE_SIZE_MAXIMUM

// REPEAT_COUNT_MINIMUM requests no copies.
const REPEAT_COUNT_MINIMUM = SLICE_SIZE_MINIMUM

// REPEAT_COUNT_MAXIMUM permits one-byte input to fill the largest Slice.
const REPEAT_COUNT_MAXIMUM = SLICE_SIZE_MAXIMUM

// REPLACEMENT_COUNT_MINIMUM requests all replacements.
const REPLACEMENT_COUNT_MINIMUM = LIMIT_MINIMUM

// REPLACEMENT_COUNT_MAXIMUM bounds a requested replacement count.
const REPLACEMENT_COUNT_MAXIMUM = LIMIT_MAXIMUM

// ORDER_BEFORE reports that the left Slice precedes the right Slice.
const ORDER_BEFORE = -1

// ORDER_EQUAL reports equal Slices.
const ORDER_EQUAL = 0

// ORDER_AFTER reports that the left Slice follows the right Slice.
const ORDER_AFTER = 1

// DECODED_CHARACTER_MINIMUM is the smallest decoded UTF-8 character.
const DECODED_CHARACTER_MINIMUM int32 = utf8.DECODED_CHARACTER_MINIMUM

// DECODED_CHARACTER_MAXIMUM is the largest decoded UTF-8 character.
const DECODED_CHARACTER_MAXIMUM int32 = utf8.DECODED_CHARACTER_MAXIMUM

// DECODED_SIZE_MINIMUM reports that no character was read.
const DECODED_SIZE_MINIMUM = utf8.DECODED_SIZE_MINIMUM

// DECODED_SIZE_MAXIMUM is the largest UTF-8 encoding.
const DECODED_SIZE_MAXIMUM = utf8.DECODED_SIZE_MAXIMUM

// ENCODED_SIZE_MINIMUM is the smallest UTF-8 encoding.
const ENCODED_SIZE_MINIMUM = utf8.CHARACTER_SIZE_MINIMUM

// ENCODED_SIZE_TWO is a two-byte UTF-8 encoding.
const ENCODED_SIZE_TWO = utf8.CHARACTER_SIZE_TWO

// ENCODED_SIZE_THREE is a three-byte UTF-8 encoding.
const ENCODED_SIZE_THREE = utf8.CHARACTER_SIZE_THREE

// ENCODED_SIZE_MAXIMUM is the largest UTF-8 encoding.
const ENCODED_SIZE_MAXIMUM = DECODED_SIZE_MAXIMUM

// MINIMUM_READ_SIZE is the standard Buffer.ReadFrom allocation step.
const MINIMUM_READ_SIZE = 512

// BUFFER_STATE_SIZE holds a Buffer offset and its last read operation.
const BUFFER_STATE_SIZE = 3

// READER_STATE_SIZE holds a Reader position and its previous rune position.
const READER_STATE_SIZE = 12

// READ_OPERATION_OTHER records a read that was not ReadRune.
const READ_OPERATION_OTHER = -1

// READ_OPERATION_ABSENT records that no read can be reversed.
const READ_OPERATION_ABSENT = DECODED_SIZE_MINIMUM

// READ_OPERATION_RUNE_1 records a one-byte rune.
const READ_OPERATION_RUNE_1 = ENCODED_SIZE_MINIMUM

// READ_OPERATION_MINIMUM records a non-rune read.
const READ_OPERATION_MINIMUM = READ_OPERATION_OTHER

// READ_OPERATION_MAXIMUM records the largest UTF-8 rune.
const READ_OPERATION_MAXIMUM = DECODED_SIZE_MAXIMUM

// GROWTH_COUNT_MINIMUM requests no additional Buffer space.
const GROWTH_COUNT_MINIMUM = SLICE_SIZE_MINIMUM

// GROWTH_COUNT_MAXIMUM fills the largest empty Buffer.
const GROWTH_COUNT_MAXIMUM = SLICE_SIZE_MAXIMUM

// READER_POSITION_MINIMUM is the first Reader position.
const READER_POSITION_MINIMUM int64 = SLICE_SIZE_MINIMUM

// READER_POSITION_MAXIMUM is the boundary after the largest Reader source.
const READER_POSITION_MAXIMUM int64 = SLICE_SIZE_MAXIMUM

// SIZE_VALUE_MINIMUM is the size of an empty Reader.
const SIZE_VALUE_MINIMUM int64 = READER_POSITION_MINIMUM

// SIZE_VALUE_MAXIMUM is the size of the largest Reader source.
const SIZE_VALUE_MAXIMUM int64 = READER_POSITION_MAXIMUM

// BUFFER_STREAM_MODES names each operation that a Buffer stream answers.
const BUFFER_STREAM_MODES io.Stream_Mode_Set = 1<<io.STREAM_MODE_QUERY |
	1<<io.STREAM_MODE_READ | 1<<io.STREAM_MODE_WRITE

// READER_STREAM_MODES names each operation that a Reader stream answers.
const READER_STREAM_MODES io.Stream_Mode_Set = 1<<io.STREAM_MODE_QUERY |
	1<<io.STREAM_MODE_READ | 1<<io.STREAM_MODE_READ_AT |
	1<<io.STREAM_MODE_SEEK | 1<<io.STREAM_MODE_SIZE

// Boundary is a byte boundary from the start through the end of a Slice.
type Boundary int

// Boundary_Invariants bounds a byte boundary through the final Slice end.
func Boundary_Invariants(value Boundary, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), SLICE_SIZE_MINIMUM, BOUNDARY_INDEX_MAXIMUM).
		Ensure()
}

// Separator_Size is the part of a separator that a split result retains.
type Separator_Size int

// Separator_Size_Invariants bounds retained separator bytes to one Slice.
func Separator_Size_Invariants(value Separator_Size, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), SLICE_SIZE_MINIMUM, SLICE_SIZE_MAXIMUM).
		Ensure()
}

// Empty_Slices is a split result for an empty separator.
type Empty_Slices [][]byte

// Empty_Slices_Invariants bounds results to one item for each source character.
func Empty_Slices_Invariants(value Empty_Slices, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SLICES_COUNT_MINIMUM, SLICE_SIZE_MAXIMUM).
		Ensure()
}

// Slice is a bounded byte slice.
type Slice []byte

// Slice_Invariants bounds a Slice that the package reads or owns.
func Slice_Invariants(value Slice, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SLICE_SIZE_MINIMUM, SLICE_SIZE_MAXIMUM).
		Ensure()
}

// Text is bounded text that supplies a character or cut set.
type Text string

// Text_Invariants bounds Text to the Slice size domain.
func Text_Invariants(value Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Slices is a split result or a collection that Join reads.
type Slices [][]byte

// Slices_Invariants bounds the number of Slices in a split or join collection.
func Slices_Invariants(value Slices, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SLICES_COUNT_MINIMUM, SLICES_COUNT_MAXIMUM).
		Ensure()
}

// Field_Slices is the set of fields that one bounded Slice can hold.
type Field_Slices [][]byte

// Field_Slices_Invariants bounds the field count that alternating bytes can make.
func Field_Slices_Invariants(value Field_Slices, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), FIELDS_COUNT_MINIMUM, FIELDS_COUNT_MAXIMUM).
		Ensure()
}

// Characters is the decoded characters of a Slice.
type Characters []rune

// Characters_Invariants bounds the number of decoded characters.
func Characters_Invariants(value Characters, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), CHARACTERS_COUNT_MINIMUM, CHARACTERS_COUNT_MAXIMUM).
		Ensure()
}

// Index_Value is a byte position or INDEX_ABSENT.
type Index_Value int

// Index_Value_Invariants bounds an index to absence or the final byte.
func Index_Value_Invariants(value Index_Value, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), INDEX_ABSENT, INDEX_MAXIMUM).
		Ensure()
}

// Boundary_Index is a byte position, an end boundary, or INDEX_ABSENT.
type Boundary_Index int

// Boundary_Index_Invariants includes the boundary after the largest Slice.
func Boundary_Index_Invariants(value Boundary_Index, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), INDEX_ABSENT, BOUNDARY_INDEX_MAXIMUM).
		Ensure()
}

// Count_Value is a non-overlapping match count.
type Count_Value int

// Count_Value_Invariants bounds matches through every empty boundary.
func Count_Value_Invariants(value Count_Value, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), COUNT_VALUE_MINIMUM, COUNT_VALUE_MAXIMUM).
		Ensure()
}

// Limit is a requested split count. Negative one requests all results.
type Limit int

// Limit_Invariants bounds a split limit.
func Limit_Invariants(value Limit, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), LIMIT_MINIMUM, LIMIT_MAXIMUM).
		Ensure()
}

// Repeat_Count is the number of copies that Repeat writes.
type Repeat_Count int

// Repeat_Count_Invariants bounds the copy count.
func Repeat_Count_Invariants(value Repeat_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), REPEAT_COUNT_MINIMUM, REPEAT_COUNT_MAXIMUM).
		Ensure()
}

// Replacement_Count is a replacement limit. Negative one requests every replacement.
type Replacement_Count int

// Replacement_Count_Invariants bounds a replacement count.
func Replacement_Count_Invariants(value Replacement_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), REPLACEMENT_COUNT_MINIMUM, REPLACEMENT_COUNT_MAXIMUM,
		).
		Ensure()
}

// Order is a lexical comparison result.
type Order int

// Order_Invariants lists the three comparison results.
func Order_Invariants(value Order, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Int(int(value), ORDER_BEFORE, ORDER_EQUAL, ORDER_AFTER).
		Ensure()
}

// Boolean is one true or false report.
type Boolean bool

// Boolean_Invariants records both Boolean states.
func Boolean_Invariants(value Boolean, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "A Boolean report is true.").
		Ensure()
}

// Character is one rune that a search or mapping reads.
type Character rune

// Character_Invariants states the complete rune storage domain.
func Character_Invariants(value Character, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int32(int32(value), bits.INTEGER_32_MINIMUM, bits.INTEGER_32_MAXIMUM).
		Ensure()
}

// Byte is one byte that a search reads.
type Byte byte

// Byte_Invariants states the complete byte domain.
func Byte_Invariants(value Byte, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), bits.WORD_8_MINIMUM, bits.WORD_8_MAXIMUM).
		Ensure()
}

// Decoded_Character is a character returned by a UTF-8 decoder.
type Decoded_Character rune

// Decoded_Character_Invariants bounds a decoded character to valid Unicode values.
func Decoded_Character_Invariants(value Decoded_Character, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int32(
			int32(value), DECODED_CHARACTER_MINIMUM, DECODED_CHARACTER_MAXIMUM,
		).
		Ensure()
}

// Decoded_Size is the byte count consumed by a character read.
type Decoded_Size int

// Decoded_Size_Invariants bounds a character read to one UTF-8 encoding.
func Decoded_Size_Invariants(value Decoded_Size, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), DECODED_SIZE_MINIMUM, DECODED_SIZE_MAXIMUM).
		Ensure()
}

// Encoded_Size is the byte count written for one character.
type Encoded_Size int

// Encoded_Size_Invariants bounds a character write to one UTF-8 encoding.
func Encoded_Size_Invariants(value Encoded_Size, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_4_Int(
			int(value),
			ENCODED_SIZE_MINIMUM,
			ENCODED_SIZE_TWO,
			ENCODED_SIZE_THREE,
			ENCODED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Available_Slice is the empty capacity that Buffer exposes for append.
type Available_Slice []byte

// Available_Slice_Invariants bounds the available Buffer capacity.
func Available_Slice_Invariants(value Available_Slice, namespace invariant.Namespace) {
	invariant.Always(
		len(value) == SLICE_SIZE_MINIMUM,
		"Available Buffer storage has no readable bytes",
	)
	invariant.Tree(value, namespace).
		Range_Int(cap(value), SLICE_SIZE_MINIMUM, SLICE_SIZE_MAXIMUM).
		Ensure()
}

// Growth_Count is the capacity that Buffer_Grow reserves.
type Growth_Count int

// Growth_Count_Invariants bounds reserved capacity to one Buffer.
func Growth_Count_Invariants(value Growth_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), GROWTH_COUNT_MINIMUM, GROWTH_COUNT_MAXIMUM).
		Ensure()
}

// Read_Operation records the read that Buffer can reverse.
type Read_Operation int8

// Read_Operation_Invariants bounds the read operation to UTF-8 sizes and non-rune reads.
func Read_Operation_Invariants(value Read_Operation, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int8(int8(value), READ_OPERATION_MINIMUM, READ_OPERATION_MAXIMUM).
		Ensure()
}

// Reader_Position is a nonnegative byte position.
type Reader_Position int64

// Reader_Position_Invariants bounds a Reader position to signed integer storage.
func Reader_Position_Invariants(value Reader_Position, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(
			int64(value), READER_POSITION_MINIMUM, READER_POSITION_MAXIMUM,
		).
		Ensure()
}

// Reader_Offset is a byte offset that Reader_Read_At validates.
type Reader_Offset int64

// Reader_Offset_Invariants bounds a Reader_Read_At input to signed integer storage.
func Reader_Offset_Invariants(value Reader_Offset, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(
			int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM,
		).
		Ensure()
}

// Size_Value is the original byte count in a Reader.
type Size_Value int64

// Size_Value_Invariants bounds a Reader size to one Slice.
func Size_Value_Invariants(value Size_Value, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), SIZE_VALUE_MINIMUM, SIZE_VALUE_MAXIMUM).
		Ensure()
}

// Buffer_State stores mutable Buffer state without another invariant branch.
type Buffer_State [BUFFER_STATE_SIZE]byte

// Buffer_State_Invariants states the fixed Buffer state storage.
func Buffer_State_Invariants(value Buffer_State, _ invariant.Namespace) {
	invariant.Always(
		len(value) == BUFFER_STATE_SIZE,
		"Buffer state has its fixed storage size",
	)
}

// Buffer holds bounded bytes for sequential reads and writes.
type Buffer struct {
	// Content owns the bytes before and after the read position.
	Content Slice
	// State stores the read position and the operation that Unread can reverse.
	State Buffer_State
}

// Buffer_Invariants composes Buffer content and fixed state storage.
func Buffer_Invariants(value Buffer, namespace invariant.Namespace) {
	Slice_Invariants(value.Content, namespace)
	Buffer_State_Invariants(value.State, namespace)
}

// Reader_State stores mutable Reader state without another invariant branch.
type Reader_State [READER_STATE_SIZE]byte

// Reader_State_Invariants states the fixed Reader state storage.
func Reader_State_Invariants(value Reader_State, _ invariant.Namespace) {
	invariant.Always(
		len(value) == READER_STATE_SIZE,
		"Reader state has its fixed storage size",
	)
}

// Reader reads and seeks through one bounded Slice.
type Reader struct {
	// Source is the Slice that Reader reads.
	Source Slice
	// State stores the current and previous Reader positions.
	State Reader_State
}

// Reader_Invariants composes Reader source and fixed state storage.
func Reader_Invariants(value Reader, namespace invariant.Namespace) {
	Slice_Invariants(value.Source, namespace)
	Reader_State_Invariants(value.State, namespace)
}

var error_unread_byte = errors.New(
	"bytes.Buffer: UnreadByte: previous operation was not a successful read",
)

// New_Buffer gives content to a new Buffer.
func New_Buffer(content Slice) (buffer *Buffer) {
	defer func() { Buffer_Invariants(*buffer, "new_buffer.buffer") }()
	Slice_Invariants(content, "new_buffer.content")
	if cap(content) > SLICE_SIZE_MAXIMUM {
		content = content[:len(content):SLICE_SIZE_MAXIMUM]
	}
	return &Buffer{Content: content}
}

// New_Buffer_Text copies content into a new Buffer.
func New_Buffer_Text(content Text) (buffer *Buffer) {
	defer func() { Buffer_Invariants(*buffer, "new_buffer_text.buffer") }()
	Text_Invariants(content, "new_buffer_text.content")
	return &Buffer{Content: Slice(content)}
}

// Buffer_Bytes returns the unread content and aliases Buffer storage.
func Buffer_Bytes(buffer *Buffer) (content Slice) {
	defer func() { Slice_Invariants(content, "buffer_bytes.content") }()
	Buffer_Invariants(*buffer, "buffer_bytes.buffer")
	offset := buffer_offset(buffer)
	return buffer.Content[offset:]
}

// Buffer_Available_Slice returns empty writable capacity from Buffer storage.
func Buffer_Available_Slice(buffer *Buffer) (available Available_Slice) {
	defer func() {
		Available_Slice_Invariants(available, "buffer_available_slice.available")
	}()
	Buffer_Invariants(*buffer, "buffer_available_slice.buffer")
	return Available_Slice(buffer.Content[len(buffer.Content):])
}

// Buffer_To_Stream returns a Stream that reads from and writes to Buffer.
func Buffer_To_Stream(buffer *Buffer) (stream io.Stream) {
	Buffer_Invariants(*buffer, "buffer_to_stream.buffer")
	return io.Stream{
		Procedure: func(
			data any,
			mode io.Stream_Mode,
			content []byte,
			_ int64,
			_ io.Seek_From,
		) (count int64, err error) {
			stream_count, stream_err := buffer_stream_procedure(
				data, mode, Slice(content),
			)
			return int64(stream_count), stream_err
		},
		Data: buffer,
	}
}

// Buffer_String returns the unread Buffer content as text.
func Buffer_String(buffer *Buffer) (content Text) {
	defer func() { Text_Invariants(content, "buffer_string.content") }()
	Buffer_Invariants(*buffer, "buffer_string.buffer")
	return Text(buffer.Content[buffer_offset(buffer):])
}

// Buffer_Peek returns up to size unread bytes without a state change.
func Buffer_Peek(buffer *Buffer, size Boundary) (content Slice, err error) {
	defer func() { Slice_Invariants(content, "buffer_peek.content") }()
	Buffer_Invariants(*buffer, "buffer_peek.buffer")
	Boundary_Invariants(size, "buffer_peek.size")
	offset := buffer_offset(buffer)
	if Buffer_Size(buffer) < size {
		return buffer.Content[offset:], io.Stream_EOF
	}
	return buffer.Content[offset : offset+size], nil
}

// Buffer_Size returns the unread byte count in Buffer.
func Buffer_Size(buffer *Buffer) (count Boundary) {
	defer func() { Boundary_Invariants(count, "buffer_length.count") }()
	Buffer_Invariants(*buffer, "buffer_len.buffer")
	return Boundary(len(buffer.Content)) - buffer_offset(buffer)
}

// Buffer_Capacity returns the storage capacity of Buffer.
func Buffer_Capacity(buffer *Buffer) (capacity Boundary) {
	defer func() { Boundary_Invariants(capacity, "buffer_capacity.capacity") }()
	Buffer_Invariants(*buffer, "buffer_capacity.buffer")
	return Boundary(cap(buffer.Content))
}

// Buffer_Available returns the unused Buffer capacity.
func Buffer_Available(buffer *Buffer) (available Boundary) {
	defer func() { Boundary_Invariants(available, "buffer_available.available") }()
	Buffer_Invariants(*buffer, "buffer_available.buffer")
	return Boundary(cap(buffer.Content) - len(buffer.Content))
}

// Buffer_Truncate keeps the first size unread bytes.
func Buffer_Truncate(buffer *Buffer, size Boundary) {
	Buffer_Invariants(*buffer, "buffer_truncate.buffer")
	Boundary_Invariants(size, "buffer_truncate.size")
	if size == 0 {
		Buffer_Reset(buffer)
		return
	}
	buffer_set_read_operation(buffer, READ_OPERATION_ABSENT)
	if size > Buffer_Size(buffer) {
		panic("bytes.Buffer: truncation out of range")
	}
	offset := buffer_offset(buffer)
	buffer.Content = buffer.Content[:offset+size]
}

// Buffer_Reset makes Buffer empty and keeps its storage.
func Buffer_Reset(buffer *Buffer) {
	Buffer_Invariants(*buffer, "buffer_reset.buffer")
	buffer.Content = buffer.Content[:0]
	buffer_set_offset(buffer, 0)
	buffer_set_read_operation(buffer, READ_OPERATION_ABSENT)
}

// Buffer_Grow reserves count bytes after the current content.
func Buffer_Grow(buffer *Buffer, count Growth_Count) {
	Buffer_Invariants(*buffer, "buffer_grow_public.buffer")
	Growth_Count_Invariants(count, "buffer_grow_public.count")
	index := buffer_grow(buffer, count)
	buffer.Content = buffer.Content[:index]
}

// Buffer_Write appends source to Buffer.
func Buffer_Write(buffer *Buffer, source Slice) (count Boundary, err error) {
	defer func() { Boundary_Invariants(count, "buffer_write.count") }()
	Buffer_Invariants(*buffer, "buffer_write.buffer")
	Slice_Invariants(source, "buffer_write.source")
	buffer_set_read_operation(buffer, READ_OPERATION_ABSENT)
	index := buffer_grow(buffer, Growth_Count(len(source)))
	return Boundary(copy(buffer.Content[index:], source)), nil
}

// Buffer_Write_Text appends source to Buffer.
func Buffer_Write_Text(buffer *Buffer, source Text) (count Boundary, err error) {
	defer func() { Boundary_Invariants(count, "buffer_write_text.count") }()
	Buffer_Invariants(*buffer, "buffer_write_string.buffer")
	Text_Invariants(source, "buffer_write_string.source")
	buffer_set_read_operation(buffer, READ_OPERATION_ABSENT)
	index := buffer_grow(buffer, Growth_Count(len(source)))
	return Boundary(copy(buffer.Content[index:], source)), nil
}

// Buffer_Read_From appends source to Buffer until an error or end of input.
func Buffer_Read_From(
	buffer *Buffer,
	source io.Stream,
) (count Boundary, err error) {
	defer func() { Boundary_Invariants(count, "buffer_read_from.count") }()
	Buffer_Invariants(*buffer, "buffer_read_from.buffer")
	buffer_set_read_operation(buffer, READ_OPERATION_ABSENT)
	var block [MINIMUM_READ_SIZE]byte
	for err == nil {
		available := SLICE_SIZE_MAXIMUM - len(buffer.Content)
		if available == 0 {
			read_count, read_err := io.Read(source, block[:1])
			if read_count > 0 {
				panic(Error_Too_Large)
			}
			if read_err == io.Stream_EOF {
				return count, nil
			}
			if read_err != nil {
				return count, read_err
			}
			continue
		}
		read_size := MINIMUM_READ_SIZE
		if available < read_size {
			read_size = available
		}
		read_count, read_err := io.Read(source, block[:read_size])
		if read_count > int64(read_size) {
			panic("bytes.Buffer: reader returned an invalid count")
		}
		if read_count > 0 {
			written, _ := Buffer_Write(buffer, Slice(block[:read_count]))
			count += written
		}
		if read_err == io.Stream_EOF {
			return count, nil
		}
		if read_err != nil {
			return count, read_err
		}
	}
	return count, err
}

// Buffer_Write_To writes unread Buffer content and consumes the stored bytes.
func Buffer_Write_To(
	buffer *Buffer,
	destination io.Stream,
) (count Boundary, err error) {
	defer func() { Boundary_Invariants(count, "buffer_write_to.count") }()
	Buffer_Invariants(*buffer, "buffer_write_to.buffer")
	buffer_set_read_operation(buffer, READ_OPERATION_ABSENT)
	unread_size := int(Buffer_Size(buffer))
	if unread_size > 0 {
		offset := buffer_offset(buffer)
		written, write_err := io.Write(destination, buffer.Content[offset:])
		buffer_set_offset(buffer, offset+Boundary(written))
		count = Boundary(written)
		if write_err != nil {
			return count, write_err
		}
	}
	Buffer_Reset(buffer)
	return count, nil
}

// Buffer_Write_Byte appends value to Buffer.
func Buffer_Write_Byte(buffer *Buffer, value Byte) (err error) {
	Buffer_Invariants(*buffer, "buffer_write_byte.buffer")
	Byte_Invariants(value, "buffer_write_byte.value")
	buffer_set_read_operation(buffer, READ_OPERATION_ABSENT)
	index := buffer_grow(buffer, 1)
	buffer.Content[index] = byte(value)
	return nil
}

// Buffer_Write_Character appends the UTF-8 form of character to Buffer.
func Buffer_Write_Character(
	buffer *Buffer,
	character Character,
) (count Encoded_Size, err error) {
	defer func() { Encoded_Size_Invariants(count, "buffer_write_character.count") }()
	Buffer_Invariants(*buffer, "buffer_write_rune.buffer")
	Character_Invariants(character, "buffer_write_rune.character")
	if uint32(character) < uint32(utf8.CHARACTER_SELF) {
		write_err := Buffer_Write_Byte(buffer, Byte(character))
		return 1, write_err
	}
	buffer_set_read_operation(buffer, READ_OPERATION_ABSENT)
	encoded_size := int(utf8.Character_Size(utf8.Character(character)))
	if encoded_size < 0 {
		encoded_size = int(utf8.Character_Size(utf8.Character(utf8.REPLACEMENT_CHARACTER)))
	}
	index := buffer_grow(buffer, Growth_Count(encoded_size))
	buffer.Content = Slice(utf8.Append_Character(
		utf8.Bytes(buffer.Content[:index]), utf8.Character(character),
	))
	return Encoded_Size(Boundary(len(buffer.Content)) - index), nil
}

// Buffer_Read copies unread Buffer content into destination.
func Buffer_Read(buffer *Buffer, destination Slice) (count Boundary, err error) {
	defer func() { Boundary_Invariants(count, "buffer_read.count") }()
	Buffer_Invariants(*buffer, "buffer_read.buffer")
	Slice_Invariants(destination, "buffer_read.destination")
	buffer_set_read_operation(buffer, READ_OPERATION_ABSENT)
	if Buffer_Size(buffer) == 0 {
		Buffer_Reset(buffer)
		return 0, io.Stream_EOF
	}
	offset := buffer_offset(buffer)
	count = Boundary(copy(destination, buffer.Content[offset:]))
	buffer_set_offset(buffer, Boundary(int(offset)+int(count)))
	if count > 0 {
		buffer_set_read_operation(buffer, READ_OPERATION_OTHER)
	}
	return count, nil
}

// Buffer_Next returns and consumes up to count bytes.
func Buffer_Next(buffer *Buffer, count Boundary) (content Slice) {
	defer func() { Slice_Invariants(content, "buffer_next.content") }()
	Buffer_Invariants(*buffer, "buffer_next.buffer")
	Boundary_Invariants(count, "buffer_next.count")
	buffer_set_read_operation(buffer, READ_OPERATION_ABSENT)
	read_size := count
	if read_size > Buffer_Size(buffer) {
		read_size = Buffer_Size(buffer)
	}
	offset := buffer_offset(buffer)
	content = buffer.Content[offset : offset+read_size]
	buffer_set_offset(buffer, offset+read_size)
	if read_size > 0 {
		buffer_set_read_operation(buffer, READ_OPERATION_OTHER)
	}
	return content
}

// Buffer_Read_Byte returns and consumes the next Buffer byte.
func Buffer_Read_Byte(buffer *Buffer) (value Byte, err error) {
	defer func() { Byte_Invariants(value, "buffer_read_byte.value") }()
	Buffer_Invariants(*buffer, "buffer_read_byte.buffer")
	if Buffer_Size(buffer) == 0 {
		Buffer_Reset(buffer)
		return 0, io.Stream_EOF
	}
	offset := buffer_offset(buffer)
	value = Byte(buffer.Content[offset])
	buffer_set_offset(buffer, offset+1)
	buffer_set_read_operation(buffer, READ_OPERATION_OTHER)
	return value, nil
}

// Buffer_Read_Character returns and consumes the next UTF-8 character from Buffer.
func Buffer_Read_Character(
	buffer *Buffer,
) (character Decoded_Character, size Decoded_Size, err error) {
	defer func() {
		Decoded_Character_Invariants(character, "buffer_read_character.character")
		Decoded_Size_Invariants(size, "buffer_read_character.size")
	}()
	Buffer_Invariants(*buffer, "buffer_read_rune.buffer")
	if Buffer_Size(buffer) == 0 {
		Buffer_Reset(buffer)
		return 0, 0, io.Stream_EOF
	}
	offset := buffer_offset(buffer)
	value := buffer.Content[offset]
	if value < byte(utf8.CHARACTER_SELF) {
		buffer_set_offset(buffer, offset+1)
		buffer_set_read_operation(buffer, READ_OPERATION_RUNE_1)
		return Decoded_Character(value), 1, nil
	}
	decoded, decoded_size := utf8.Decode_Character(utf8.Bytes(buffer.Content[offset:]))
	character = Decoded_Character(decoded)
	size = Decoded_Size(decoded_size)
	buffer_set_offset(buffer, offset+Boundary(size))
	buffer_set_read_operation(buffer, Read_Operation(size))
	return character, size, nil
}

// Buffer_Unread_Character moves before the character from the last character read.
func Buffer_Unread_Character(buffer *Buffer) (err error) {
	Buffer_Invariants(*buffer, "buffer_unread_rune.buffer")
	operation := buffer_read_operation(buffer)
	if operation <= READ_OPERATION_ABSENT {
		return errors.New(
			"bytes.Buffer: UnreadRune: previous operation was not a " +
				"successful ReadRune",
		)
	}
	offset := buffer_offset(buffer)
	buffer_set_offset(buffer, offset-Boundary(operation))
	buffer_set_read_operation(buffer, READ_OPERATION_ABSENT)
	return nil
}

// Buffer_Unread_Byte moves before the last byte from a successful read.
func Buffer_Unread_Byte(buffer *Buffer) (err error) {
	Buffer_Invariants(*buffer, "buffer_unread_byte.buffer")
	if buffer_read_operation(buffer) == READ_OPERATION_ABSENT {
		return error_unread_byte
	}
	buffer_set_read_operation(buffer, READ_OPERATION_ABSENT)
	offset := buffer_offset(buffer)
	if offset > 0 {
		buffer_set_offset(buffer, offset-1)
	}
	return nil
}

// Buffer_Read_Bytes returns a copy through the first delimiter.
func Buffer_Read_Bytes(buffer *Buffer, delimiter Byte) (line Slice, err error) {
	defer func() { Slice_Invariants(line, "buffer_read_bytes.line") }()
	Buffer_Invariants(*buffer, "buffer_read_bytes.buffer")
	Byte_Invariants(delimiter, "buffer_read_bytes.delimiter")
	content, read_err := buffer_read_slice(buffer, delimiter)
	line = append(line, content...)
	return line, read_err
}

// Buffer_Read_Text returns text through the first delimiter.
func Buffer_Read_Text(buffer *Buffer, delimiter Byte) (line Text, err error) {
	defer func() { Text_Invariants(line, "buffer_read_text.line") }()
	Buffer_Invariants(*buffer, "buffer_read_text.buffer")
	Byte_Invariants(delimiter, "buffer_read_text.delimiter")
	content, read_err := buffer_read_slice(buffer, delimiter)
	return Text(content), read_err
}

// New_Reader starts a Reader at the first byte of source.
func New_Reader(source Slice) (reader *Reader) {
	defer func() { Reader_Invariants(*reader, "new_reader.reader") }()
	Slice_Invariants(source, "new_reader.source")
	reader = &Reader{Source: source}
	reader_set_previous_rune(reader, INDEX_ABSENT)
	return reader
}

// Reader_To_Stream returns a read-only Stream that uses Reader state.
func Reader_To_Stream(reader *Reader) (stream io.Stream) {
	Reader_Invariants(*reader, "reader_to_stream.reader")
	return io.Stream{
		Procedure: func(
			data any,
			mode io.Stream_Mode,
			content []byte,
			offset int64,
			origin io.Seek_From,
		) (count int64, err error) {
			stream_count, stream_err := reader_stream_procedure(
				data, mode, Slice(content), Reader_Offset(offset), origin,
			)
			return int64(stream_count), stream_err
		},
		Data: reader,
	}
}

// Reader_Unread_Size returns the unread byte count in Reader.
func Reader_Unread_Size(reader *Reader) (count Boundary) {
	defer func() { Boundary_Invariants(count, "reader_length.count") }()
	Reader_Invariants(*reader, "reader_len.reader")
	position := reader_position(reader)
	if position >= Reader_Position(len(reader.Source)) {
		return 0
	}
	return Boundary(len(reader.Source) - int(position))
}

// Reader_Size returns the original source size.
func Reader_Size(reader *Reader) (size Size_Value) {
	defer func() { Size_Value_Invariants(size, "reader_size.size") }()
	Reader_Invariants(*reader, "reader_size.reader")
	return Size_Value(len(reader.Source))
}

// Reader_Read copies unread Reader content into destination.
func Reader_Read(reader *Reader, destination Slice) (count Boundary, err error) {
	defer func() { Boundary_Invariants(count, "reader_read.count") }()
	Reader_Invariants(*reader, "reader_read.reader")
	Slice_Invariants(destination, "reader_read.destination")
	position := reader_position(reader)
	if position >= Reader_Position(len(reader.Source)) {
		return 0, io.Stream_EOF
	}
	reader_set_previous_rune(reader, INDEX_ABSENT)
	count = Boundary(copy(destination, reader.Source[position:]))
	reader_set_position(reader, position+Reader_Position(count))
	return count, nil
}

// Reader_Read_At copies Reader content at position without a state change.
func Reader_Read_At(
	reader *Reader,
	destination Slice,
	offset Reader_Offset,
) (count Boundary, err error) {
	defer func() { Boundary_Invariants(count, "reader_read_at.count") }()
	Reader_Invariants(*reader, "reader_read_at.reader")
	Slice_Invariants(destination, "reader_read_at.destination")
	Reader_Offset_Invariants(offset, "reader_read_at.offset")
	if offset < 0 {
		return 0, io.Stream_Invalid_Offset
	}
	position := Reader_Position(offset)
	if position > Reader_Position(len(reader.Source)) {
		return 0, io.Stream_Invalid_Offset
	}
	if position == Reader_Position(len(reader.Source)) {
		return 0, io.Stream_EOF
	}
	count = Boundary(copy(destination, reader.Source[position:]))
	return count, nil
}

// Reader_Read_Byte returns and consumes the next Reader byte.
func Reader_Read_Byte(reader *Reader) (value Byte, err error) {
	defer func() { Byte_Invariants(value, "reader_read_byte.value") }()
	Reader_Invariants(*reader, "reader_read_byte.reader")
	reader_set_previous_rune(reader, INDEX_ABSENT)
	position := reader_position(reader)
	if position >= Reader_Position(len(reader.Source)) {
		return 0, io.Stream_EOF
	}
	value = Byte(reader.Source[position])
	reader_set_position(reader, position+1)
	return value, nil
}

// Reader_Unread_Byte moves Reader back by one byte.
func Reader_Unread_Byte(reader *Reader) (err error) {
	Reader_Invariants(*reader, "reader_unread_byte.reader")
	position := reader_position(reader)
	if position <= 0 {
		return errors.New("bytes.Reader.UnreadByte: at beginning of slice")
	}
	reader_set_previous_rune(reader, INDEX_ABSENT)
	reader_set_position(reader, position-1)
	return nil
}

// Reader_Read_Character returns and consumes the next UTF-8 character from Reader.
func Reader_Read_Character(
	reader *Reader,
) (character Decoded_Character, size Decoded_Size, err error) {
	defer func() {
		Decoded_Character_Invariants(character, "reader_read_character.character")
		Decoded_Size_Invariants(size, "reader_read_character.size")
	}()
	Reader_Invariants(*reader, "reader_read_rune.reader")
	position := reader_position(reader)
	if position >= Reader_Position(len(reader.Source)) {
		reader_set_previous_rune(reader, INDEX_ABSENT)
		return 0, 0, io.Stream_EOF
	}
	reader_set_previous_rune(reader, Index_Value(position))
	value := reader.Source[position]
	if value < byte(utf8.CHARACTER_SELF) {
		reader_set_position(reader, position+1)
		return Decoded_Character(value), 1, nil
	}
	decoded, decoded_size := utf8.Decode_Character(utf8.Bytes(reader.Source[position:]))
	character = Decoded_Character(decoded)
	size = Decoded_Size(decoded_size)
	reader_set_position(reader, position+Reader_Position(size))
	return character, size, nil
}

// Reader_Unread_Character returns Reader to the start of its last character read.
func Reader_Unread_Character(reader *Reader) (err error) {
	Reader_Invariants(*reader, "reader_unread_rune.reader")
	position := reader_position(reader)
	if position <= 0 {
		return errors.New("bytes.Reader.UnreadRune: at beginning of slice")
	}
	previous := reader_previous_rune(reader.State)
	if previous < 0 {
		return errors.New("bytes.Reader.UnreadRune: previous operation was not ReadRune")
	}
	reader_set_position(reader, Reader_Position(previous))
	reader_set_previous_rune(reader, INDEX_ABSENT)
	return nil
}

// Reader_Seek sets the next Reader position from one Stream origin.
func Reader_Seek(
	reader *Reader,
	offset Reader_Offset,
	origin io.Seek_From,
) (position Reader_Position, err error) {
	defer func() { Reader_Position_Invariants(position, "reader_seek.position") }()
	Reader_Invariants(*reader, "reader_seek.reader")
	Reader_Offset_Invariants(offset, "reader_seek.offset")
	reader_set_previous_rune(reader, INDEX_ABSENT)
	target := int64(offset)
	if origin == io.SEEK_FROM_CURRENT {
		target = int64(reader_position(reader)) + int64(offset)
	} else if origin == io.SEEK_FROM_END {
		target = int64(len(reader.Source)) + int64(offset)
	} else if origin != io.SEEK_FROM_START {
		return 0, io.Stream_Invalid_Whence
	}
	if target < 0 {
		return 0, io.Stream_Invalid_Offset
	}
	if target > int64(len(reader.Source)) {
		return 0, io.Stream_Invalid_Offset
	}
	position = Reader_Position(target)
	reader_set_position(reader, Reader_Position(position))
	return position, nil
}

// Reader_Write_To writes unread Reader content and consumes the stored bytes.
func Reader_Write_To(
	reader *Reader,
	destination io.Stream,
) (count Boundary, err error) {
	defer func() { Boundary_Invariants(count, "reader_write_to.count") }()
	Reader_Invariants(*reader, "reader_write_to.reader")
	reader_set_previous_rune(reader, INDEX_ABSENT)
	position := reader_position(reader)
	if position >= Reader_Position(len(reader.Source)) {
		return 0, nil
	}
	content := reader.Source[position:]
	written, write_err := io.Write(destination, content)
	reader_set_position(reader, position+Reader_Position(written))
	count = Boundary(written)
	return count, write_err
}

// Reader_Reset replaces Reader source and returns to its first byte.
func Reader_Reset(reader *Reader, source Slice) {
	Reader_Invariants(*reader, "reader_reset.reader")
	Slice_Invariants(source, "reader_reset.source")
	reader.Source = source
	reader_set_position(reader, 0)
	reader_set_previous_rune(reader, INDEX_ABSENT)
}

// Runs one Stream mode against Buffer.
func buffer_stream_procedure(
	data any,
	mode io.Stream_Mode,
	content Slice,
) (count Boundary, err error) {
	defer func() { Boundary_Invariants(count, "buffer_stream.count") }()
	Slice_Invariants(content, "buffer_stream.content")
	buffer, held := data.(*Buffer)
	invariant.Always(held, "A Buffer stream procedure receives its Buffer")
	invariant.Always(buffer != nil, "A Buffer stream procedure receives a Buffer value")
	Buffer_Invariants(*buffer, "buffer_stream.buffer")
	if mode == io.STREAM_MODE_QUERY {
		return Boundary(BUFFER_STREAM_MODES), nil
	}
	if mode == io.STREAM_MODE_READ {
		return Buffer_Read(buffer, content)
	}
	if mode == io.STREAM_MODE_WRITE {
		return Buffer_Write(buffer, content)
	}
	return 0, io.Stream_Empty
}

// Runs one Stream mode against Reader.
func reader_stream_procedure(
	data any,
	mode io.Stream_Mode,
	content Slice,
	offset Reader_Offset,
	origin io.Seek_From,
) (count Boundary, err error) {
	defer func() { Boundary_Invariants(count, "reader_stream.count") }()
	Slice_Invariants(content, "reader_stream.content")
	Reader_Offset_Invariants(offset, "reader_stream.offset")
	reader, held := data.(*Reader)
	invariant.Always(held, "A Reader stream procedure receives its Reader")
	invariant.Always(reader != nil, "A Reader stream procedure receives a Reader value")
	Reader_Invariants(*reader, "reader_stream.reader")
	if mode == io.STREAM_MODE_QUERY {
		return Boundary(READER_STREAM_MODES), nil
	}
	if mode == io.STREAM_MODE_READ {
		return Reader_Read(reader, content)
	}
	if mode == io.STREAM_MODE_READ_AT {
		return Reader_Read_At(reader, content, offset)
	}
	if mode == io.STREAM_MODE_SEEK {
		position, seek_err := Reader_Seek(reader, offset, origin)
		return Boundary(position), seek_err
	}
	if mode == io.STREAM_MODE_SIZE {
		return Boundary(Reader_Size(reader)), nil
	}
	return 0, io.Stream_Empty
}

func buffer_grow(buffer *Buffer, count Growth_Count) (index Boundary) {
	defer func() { Boundary_Invariants(index, "buffer_grow.index") }()
	Buffer_Invariants(*buffer, "buffer_grow.buffer")
	Growth_Count_Invariants(count, "buffer_grow.count")
	unread_size := int(Buffer_Size(buffer))
	if int(count) > SLICE_SIZE_MAXIMUM-unread_size {
		panic(Error_Too_Large)
	}
	if unread_size == 0 {
		Buffer_Reset(buffer)
	}
	index = Boundary(len(buffer.Content))
	if int(count) <= cap(buffer.Content)-int(index) {
		buffer.Content = buffer.Content[:index+Boundary(count)]
		return index
	}
	if unread_size+int(count) <= cap(buffer.Content) {
		offset := buffer_offset(buffer)
		copy(buffer.Content, buffer.Content[offset:])
		buffer_set_offset(buffer, 0)
		buffer.Content = buffer.Content[:unread_size+int(count)]
		return Boundary(unread_size)
	}
	capacity := 2 * cap(buffer.Content)
	if capacity < unread_size+int(count) {
		capacity = unread_size + int(count)
	}
	if capacity < 64 {
		capacity = 64
	}
	if capacity > SLICE_SIZE_MAXIMUM {
		capacity = SLICE_SIZE_MAXIMUM
	}
	content := make(Slice, unread_size+int(count), capacity)
	offset := buffer_offset(buffer)
	copy(content, buffer.Content[offset:])
	buffer.Content = content
	buffer_set_offset(buffer, 0)
	return Boundary(unread_size)
}

func buffer_read_slice(buffer *Buffer, delimiter Byte) (line Slice, err error) {
	defer func() { Slice_Invariants(line, "buffer_read_slice.line") }()
	Buffer_Invariants(*buffer, "buffer_read_slice.buffer")
	Byte_Invariants(delimiter, "buffer_read_slice.delimiter")
	offset := buffer_offset(buffer)
	index := Index_Byte(buffer.Content[offset:], delimiter)
	end := offset + Boundary(index) + 1
	if index < 0 {
		end = Boundary(len(buffer.Content))
		err = io.Stream_EOF
	}
	line = buffer.Content[offset:end]
	buffer_set_offset(buffer, end)
	buffer_set_read_operation(buffer, READ_OPERATION_OTHER)
	return line, err
}

func buffer_offset(buffer *Buffer) (offset Boundary) {
	defer func() { Boundary_Invariants(offset, "buffer_offset.offset") }()
	Buffer_Invariants(*buffer, "buffer_offset.buffer")
	return Boundary(uint16(buffer.State[0]) | uint16(buffer.State[1])<<8)
}

func buffer_set_offset(buffer *Buffer, offset Boundary) {
	Buffer_Invariants(*buffer, "buffer_set_offset.buffer")
	Boundary_Invariants(offset, "buffer_set_offset.offset")
	buffer.State[0] = byte(offset)
	buffer.State[1] = byte(uint16(offset) >> 8)
}

func buffer_read_operation(buffer *Buffer) (operation Read_Operation) {
	defer func() {
		Read_Operation_Invariants(operation, "buffer_read_operation.operation")
	}()
	Buffer_Invariants(*buffer, "buffer_read_operation.buffer")
	return Read_Operation(int8(buffer.State[2]))
}

func buffer_set_read_operation(buffer *Buffer, operation Read_Operation) {
	Buffer_Invariants(*buffer, "buffer_set_read_operation.buffer")
	Read_Operation_Invariants(operation, "buffer_set_read_operation.operation")
	buffer.State[2] = byte(operation)
}

func reader_position(reader *Reader) (position Reader_Position) {
	defer func() { Reader_Position_Invariants(position, "reader_position.position") }()
	Reader_Invariants(*reader, "reader_position.reader")
	var encoded uint64
	for index := 0; index < 8; index++ {
		encoded |= uint64(reader.State[index]) << (index * 8)
	}
	return Reader_Position(encoded)
}

func reader_set_position(reader *Reader, position Reader_Position) {
	Reader_Invariants(*reader, "reader_set_position.reader")
	Reader_Position_Invariants(position, "reader_set_position.position")
	encoded := uint64(position)
	for index := 0; index < 8; index++ {
		reader.State[index] = byte(encoded >> (index * 8))
	}
}

func reader_previous_rune(state Reader_State) (position Index_Value) {
	defer func() { Index_Value_Invariants(position, "reader_previous_rune.position") }()
	Reader_State_Invariants(state, "reader_previous_rune.state")
	encoded := uint32(state[8]) |
		uint32(state[9])<<8 |
		uint32(state[10])<<16 |
		uint32(state[11])<<24
	return Index_Value(int32(encoded))
}

func reader_set_previous_rune(reader *Reader, position Index_Value) {
	Reader_Invariants(*reader, "reader_set_previous_rune.reader")
	Index_Value_Invariants(position, "reader_set_previous_rune.position")
	encoded := uint32(int32(position))
	reader.State[8] = byte(encoded)
	reader.State[9] = byte(encoded >> 8)
	reader.State[10] = byte(encoded >> 16)
	reader.State[11] = byte(encoded >> 24)
}

// Equal reports whether two Slices have the same bytes. Nil and empty are equal.
func Equal(left Slice, right Slice) (equal Boolean) {
	defer func() { Boolean_Invariants(equal, "equal.equal") }()
	Slice_Invariants(left, "equal.left")
	Slice_Invariants(right, "equal.right")
	return Boolean(string(left) == string(right))
}

// Compare orders two Slices lexically.
func Compare(left Slice, right Slice) (order Order) {
	defer func() { Order_Invariants(order, "compare.order") }()
	Slice_Invariants(left, "compare.left")
	Slice_Invariants(right, "compare.right")
	minimum_count := len(left)
	if len(right) < minimum_count {
		minimum_count = len(right)
	}
	for index := 0; index < minimum_count; index++ {
		if left[index] < right[index] {
			return ORDER_BEFORE
		}
		if left[index] > right[index] {
			return ORDER_AFTER
		}
	}
	if len(left) < len(right) {
		return ORDER_BEFORE
	}
	if len(left) > len(right) {
		return ORDER_AFTER
	}
	return ORDER_EQUAL
}

// Count returns the number of non-overlapping separator occurrences.
func Count(source Slice, separator Slice) (count Count_Value) {
	defer func() { Count_Value_Invariants(count, "count.count") }()
	Slice_Invariants(source, "count.source")
	Slice_Invariants(separator, "count.separator")
	if len(separator) == 1 {
		for _, value := range source {
			if value == separator[0] {
				count++
			}
		}
		return count
	}
	return Count_Value(strings.Count(strings.Text(source), strings.Text(separator)))
}

// Contains reports whether a separator occurs in source.
func Contains(source Slice, separator Slice) (contained Boolean) {
	defer func() { Boolean_Invariants(contained, "contains.contained") }()
	Slice_Invariants(source, "contains.source")
	Slice_Invariants(separator, "contains.separator")
	return Boolean(strings.Contains(strings.Text(source), strings.Text(separator)))
}

// Contains_Any reports whether a character from Text occurs in source.
func Contains_Any(source Slice, characters Text) (contained Boolean) {
	defer func() { Boolean_Invariants(contained, "contains_any.contained") }()
	Slice_Invariants(source, "contains_any.source")
	Text_Invariants(characters, "contains_any.characters")
	return Boolean(strings.Contains_Any(strings.Text(source), strings.Text(characters)))
}

// Contains_Rune reports whether a character occurs in source.
func Contains_Rune(source Slice, character Character) (contained Boolean) {
	defer func() { Boolean_Invariants(contained, "contains_rune.contained") }()
	Slice_Invariants(source, "contains_rune.source")
	Character_Invariants(character, "contains_rune.character")
	return Boolean(strings.Contains_Rune(strings.Text(source), strings.Character(character)))
}

// Contains_Function reports whether a source character satisfies predicate.
func Contains_Function(
	source Slice, predicate func(rune) (matches bool),
) (contained Boolean) {
	defer func() { Boolean_Invariants(contained, "contains_func.contained") }()
	Slice_Invariants(source, "contains_func.source")
	return Boolean(strings.Contains_Function(strings.Text(source), predicate))
}

// Index_Byte returns the first index of value or INDEX_ABSENT.
func Index_Byte(source Slice, value Byte) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "index_byte.index") }()
	Slice_Invariants(source, "index_byte.source")
	Byte_Invariants(value, "index_byte.value")
	return Index_Value(strings.Index_Byte(strings.Text(source), strings.Byte(value)))
}

// Last_Index returns the last separator index, including the end boundary for empty separator.
func Last_Index(source Slice, separator Slice) (index Boundary_Index) {
	defer func() { Boundary_Index_Invariants(index, "last_index.index") }()
	Slice_Invariants(source, "last_index.source")
	Slice_Invariants(separator, "last_index.separator")
	return Boundary_Index(strings.Last_Index(strings.Text(source), strings.Text(separator)))
}

// Last_Index_Byte returns the last index of value or INDEX_ABSENT.
func Last_Index_Byte(source Slice, value Byte) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "last_index_byte.index") }()
	Slice_Invariants(source, "last_index_byte.source")
	Byte_Invariants(value, "last_index_byte.value")
	return Index_Value(strings.Last_Index_Byte(strings.Text(source), strings.Byte(value)))
}

// Index_Rune returns the first byte index of character or INDEX_ABSENT.
func Index_Rune(source Slice, character Character) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "index_rune.index") }()
	Slice_Invariants(source, "index_rune.source")
	Character_Invariants(character, "index_rune.character")
	return Index_Value(strings.Index_Rune(strings.Text(source), strings.Character(character)))
}

// Index_Any returns the first byte index of a character from Text.
func Index_Any(source Slice, characters Text) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "index_any.index") }()
	Slice_Invariants(source, "index_any.source")
	Text_Invariants(characters, "index_any.characters")
	return Index_Value(strings.Index_Any(strings.Text(source), strings.Text(characters)))
}

// Last_Index_Any returns the last byte index of a character from Text.
func Last_Index_Any(source Slice, characters Text) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "last_index_any.index") }()
	Slice_Invariants(source, "last_index_any.source")
	Text_Invariants(characters, "last_index_any.characters")
	return Index_Value(strings.Last_Index_Any(strings.Text(source), strings.Text(characters)))
}

// Split_N divides source after at most limit minus one separators.
func Split_N(source Slice, separator Slice, limit Limit) (parts Slices) {
	defer func() { Slices_Invariants(parts, "split_n.parts") }()
	Slice_Invariants(source, "split_n.source")
	Slice_Invariants(separator, "split_n.separator")
	Limit_Invariants(limit, "split_n.limit")
	return split(source, separator, 0, limit)
}

// Split_After_N divides source after separators and keeps each separator.
func Split_After_N(source Slice, separator Slice, limit Limit) (parts Slices) {
	defer func() { Slices_Invariants(parts, "split_after_n.parts") }()
	Slice_Invariants(source, "split_after_n.source")
	Slice_Invariants(separator, "split_after_n.separator")
	Limit_Invariants(limit, "split_after_n.limit")
	return split(source, separator, Separator_Size(len(separator)), limit)
}

// Split divides source at every non-overlapping separator.
func Split(source Slice, separator Slice) (parts Slices) {
	defer func() { Slices_Invariants(parts, "split.parts") }()
	Slice_Invariants(source, "split.source")
	Slice_Invariants(separator, "split.separator")
	return split(source, separator, 0, LIMIT_MINIMUM)
}

// Split_After divides source at every separator and keeps each separator.
func Split_After(source Slice, separator Slice) (parts Slices) {
	defer func() { Slices_Invariants(parts, "split_after.parts") }()
	Slice_Invariants(source, "split_after.source")
	Slice_Invariants(separator, "split_after.separator")
	return split(
		source, separator, Separator_Size(len(separator)), LIMIT_MINIMUM,
	)
}

// Fields divides source around consecutive Unicode white-space characters.
func Fields(source Slice) (fields Field_Slices) {
	defer func() { Field_Slices_Invariants(fields, "fields.fields") }()
	Slice_Invariants(source, "fields.source")
	space := func(character rune) (yes bool) {
		return bool(ucd.Is_Space(ucd.Character(character)))
	}
	return fields_function(source, space)
}

// Fields_Function divides source around runs of characters that satisfy predicate.
func Fields_Function(
	source Slice, predicate func(rune) (matches bool),
) (fields Field_Slices) {
	defer func() { Field_Slices_Invariants(fields, "fields_func.fields") }()
	Slice_Invariants(source, "fields_func.source")
	return fields_function(source, predicate)
}

// Join joins each part through separator.
func Join(parts Slices, separator Slice) (joined Slice) {
	defer func() { Slice_Invariants(joined, "join.joined") }()
	Slices_Invariants(parts, "join.parts")
	Slice_Invariants(separator, "join.separator")
	joined_size := 0
	for _, part := range parts {
		if len(part) > SLICE_SIZE_MAXIMUM-joined_size {
			panic(Error_Too_Large)
		}
		joined_size += len(part)
	}
	if len(parts) > 1 {
		separator_count := len(parts) - 1
		if len(separator) > 0 {
			if separator_count > (SLICE_SIZE_MAXIMUM-joined_size)/len(separator) {
				panic(Error_Too_Large)
			}
		}
		joined_size += len(separator) * separator_count
	}
	joined = make(Slice, 0, joined_size)
	for index, part := range parts {
		if index > 0 {
			joined = append(joined, separator...)
		}
		joined = append(joined, part...)
	}
	return joined
}

// Has_Prefix reports whether source starts with prefix.
func Has_Prefix(source Slice, prefix Slice) (present Boolean) {
	defer func() { Boolean_Invariants(present, "has_prefix.present") }()
	Slice_Invariants(source, "has_prefix.source")
	Slice_Invariants(prefix, "has_prefix.prefix")
	return Boolean(strings.Has_Prefix(strings.Text(source), strings.Text(prefix)))
}

// Has_Suffix reports whether source ends with suffix.
func Has_Suffix(source Slice, suffix Slice) (present Boolean) {
	defer func() { Boolean_Invariants(present, "has_suffix.present") }()
	Slice_Invariants(source, "has_suffix.source")
	Slice_Invariants(suffix, "has_suffix.suffix")
	return Boolean(strings.Has_Suffix(strings.Text(source), strings.Text(suffix)))
}

// Map applies mapping to each decoded character and drops a negative result.
func Map(mapping func(rune) (mapped_character rune), source Slice) (mapped Slice) {
	defer func() { Slice_Invariants(mapped, "map.mapped") }()
	Slice_Invariants(source, "map.source")
	result := strings.Map(mapping, strings.Text(source))
	if len(result) > SLICE_SIZE_MAXIMUM {
		panic(Error_Too_Large)
	}
	return Slice([]byte(result))
}

// Repeat returns count consecutive copies of source.
func Repeat(source Slice, count Repeat_Count) (repeated Slice) {
	defer func() { Slice_Invariants(repeated, "repeat.repeated") }()
	Slice_Invariants(source, "repeat.source")
	Repeat_Count_Invariants(count, "repeat.count")
	repeated_size := 0
	if len(source) > 0 {
		if int(count) > SLICE_SIZE_MAXIMUM/len(source) {
			panic(Error_Too_Large)
		}
		repeated_size = len(source) * int(count)
	}
	if repeated_size == 0 {
		return Slice{}
	}
	repeated = make(Slice, repeated_size)
	copied := copy(repeated, source)
	for copied < repeated_size {
		copied += copy(repeated[copied:], repeated[:copied])
	}
	return repeated
}

// To_Upper returns source with each Unicode letter mapped to uppercase.
func To_Upper(source Slice) (upper Slice) {
	defer func() { Slice_Invariants(upper, "to_upper.upper") }()
	Slice_Invariants(source, "to_upper.source")
	text := strings.To_Upper(strings.Text(source))
	if len(text) > SLICE_SIZE_MAXIMUM {
		panic(Error_Too_Large)
	}
	return Slice([]byte(text))
}

// To_Lower returns source with each Unicode letter mapped to lowercase.
func To_Lower(source Slice) (lower Slice) {
	defer func() { Slice_Invariants(lower, "to_lower.lower") }()
	Slice_Invariants(source, "to_lower.source")
	text := strings.To_Lower(strings.Text(source))
	if len(text) > SLICE_SIZE_MAXIMUM {
		panic(Error_Too_Large)
	}
	return Slice([]byte(text))
}

// To_Title returns source with each Unicode letter mapped to title case.
func To_Title(source Slice) (title Slice) {
	defer func() { Slice_Invariants(title, "to_title.title") }()
	Slice_Invariants(source, "to_title.source")
	text := strings.To_Title(strings.Text(source))
	if len(text) > SLICE_SIZE_MAXIMUM {
		panic(Error_Too_Large)
	}
	return Slice([]byte(text))
}

// To_Upper_Special applies a special-case uppercase mapping.
func To_Upper_Special(special ucd.Special_Case, source Slice) (upper Slice) {
	defer func() { Slice_Invariants(upper, "to_upper_special.upper") }()
	ucd.Special_Case_Invariants(special, "to_upper_special.special")
	Slice_Invariants(source, "to_upper_special.source")
	text := strings.To_Upper_Special(special, strings.Text(source))
	if len(text) > SLICE_SIZE_MAXIMUM {
		panic(Error_Too_Large)
	}
	return Slice([]byte(text))
}

// To_Lower_Special applies a special-case lowercase mapping.
func To_Lower_Special(special ucd.Special_Case, source Slice) (lower Slice) {
	defer func() { Slice_Invariants(lower, "to_lower_special.lower") }()
	ucd.Special_Case_Invariants(special, "to_lower_special.special")
	Slice_Invariants(source, "to_lower_special.source")
	text := strings.To_Lower_Special(special, strings.Text(source))
	if len(text) > SLICE_SIZE_MAXIMUM {
		panic(Error_Too_Large)
	}
	return Slice([]byte(text))
}

// To_Title_Special applies a special-case title mapping.
func To_Title_Special(special ucd.Special_Case, source Slice) (title Slice) {
	defer func() { Slice_Invariants(title, "to_title_special.title") }()
	ucd.Special_Case_Invariants(special, "to_title_special.special")
	Slice_Invariants(source, "to_title_special.source")
	text := strings.To_Title_Special(special, strings.Text(source))
	if len(text) > SLICE_SIZE_MAXIMUM {
		panic(Error_Too_Large)
	}
	return Slice([]byte(text))
}

// To_Valid_UTF8 replaces each run of invalid UTF-8 bytes.
func To_Valid_UTF8(source Slice, replacement Slice) (valid Slice) {
	defer func() { Slice_Invariants(valid, "to_valid_utf8.valid") }()
	Slice_Invariants(source, "to_valid_utf8.source")
	Slice_Invariants(replacement, "to_valid_utf8.replacement")
	text := strings.To_Valid_UTF8(strings.Text(source), strings.Text(replacement))
	if len(text) > SLICE_SIZE_MAXIMUM {
		panic(Error_Too_Large)
	}
	return Slice([]byte(text))
}

// Title returns source with each word start mapped to title case.
func Title(source Slice) (title Slice) {
	defer func() { Slice_Invariants(title, "title.title") }()
	Slice_Invariants(source, "title.source")
	text := strings.Title(strings.Text(source))
	if len(text) > SLICE_SIZE_MAXIMUM {
		panic(Error_Too_Large)
	}
	return Slice([]byte(text))
}

// Trim_Left_Function removes the leading characters that satisfy predicate.
func Trim_Left_Function(
	source Slice, predicate func(rune) (matches bool),
) (trimmed Slice) {
	defer func() { Slice_Invariants(trimmed, "trim_left_func.trimmed") }()
	Slice_Invariants(source, "trim_left_func.source")
	end := trim_left_boundary(source, predicate)
	if int(end) == len(source) {
		return nil
	}
	return source[end:]
}

// Trim_Right_Function removes the trailing characters that satisfy predicate.
func Trim_Right_Function(
	source Slice, predicate func(rune) (matches bool),
) (trimmed Slice) {
	defer func() { Slice_Invariants(trimmed, "trim_right_func.trimmed") }()
	Slice_Invariants(source, "trim_right_func.source")
	return source[:int(trim_right_boundary(source, predicate))]
}

// Trim_Function removes leading and trailing characters that satisfy predicate.
func Trim_Function(
	source Slice, predicate func(rune) (matches bool),
) (trimmed Slice) {
	defer func() { Slice_Invariants(trimmed, "trim_func.trimmed") }()
	Slice_Invariants(source, "trim_func.source")
	left := int(trim_left_boundary(source, predicate))
	if left == len(source) {
		return nil
	}
	right := int(trim_right_boundary(source[left:], predicate))
	return source[left : left+right]
}

// Trim_Prefix removes prefix when source starts with it.
func Trim_Prefix(source Slice, prefix Slice) (trimmed Slice) {
	defer func() { Slice_Invariants(trimmed, "trim_prefix.trimmed") }()
	Slice_Invariants(source, "trim_prefix.source")
	Slice_Invariants(prefix, "trim_prefix.prefix")
	if strings.Has_Prefix(strings.Text(source), strings.Text(prefix)) {
		return source[len(prefix):]
	}
	return source
}

// Trim_Suffix removes suffix when source ends with it.
func Trim_Suffix(source Slice, suffix Slice) (trimmed Slice) {
	defer func() { Slice_Invariants(trimmed, "trim_suffix.trimmed") }()
	Slice_Invariants(source, "trim_suffix.source")
	Slice_Invariants(suffix, "trim_suffix.suffix")
	if strings.Has_Suffix(strings.Text(source), strings.Text(suffix)) {
		return source[:len(source)-len(suffix)]
	}
	return source
}

// Index_Function returns the first byte index of a predicate match.
func Index_Function(
	source Slice, predicate func(rune) (matches bool),
) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "index_func.index") }()
	Slice_Invariants(source, "index_func.source")
	return Index_Value(strings.Index_Function(strings.Text(source), predicate))
}

// Last_Index_Function returns the last byte index of a predicate match.
func Last_Index_Function(
	source Slice, predicate func(rune) (matches bool),
) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "last_index_func.index") }()
	Slice_Invariants(source, "last_index_func.source")
	return Index_Value(strings.Last_Index_Function(strings.Text(source), predicate))
}

// Trim removes leading and trailing characters in cutset.
func Trim(source Slice, cutset Text) (trimmed Slice) {
	defer func() { Slice_Invariants(trimmed, "trim.trimmed") }()
	Slice_Invariants(source, "trim.source")
	Text_Invariants(cutset, "trim.cutset")
	cutset_text := strings.Text(cutset)
	predicate := func(character rune) (matches bool) {
		return bool(strings.Contains_Rune(cutset_text, strings.Character(character)))
	}
	left := int(trim_left_boundary(source, predicate))
	if left == len(source) {
		return nil
	}
	right := int(trim_right_boundary(source[left:], predicate))
	return source[left : left+right]
}

// Trim_Left removes leading characters in cutset.
func Trim_Left(source Slice, cutset Text) (trimmed Slice) {
	defer func() { Slice_Invariants(trimmed, "trim_left.trimmed") }()
	Slice_Invariants(source, "trim_left.source")
	Text_Invariants(cutset, "trim_left.cutset")
	cutset_text := strings.Text(cutset)
	predicate := func(character rune) (matches bool) {
		return bool(strings.Contains_Rune(cutset_text, strings.Character(character)))
	}
	end := trim_left_boundary(source, predicate)
	if int(end) == len(source) {
		return nil
	}
	return source[end:]
}

// Trim_Right removes trailing characters in cutset.
func Trim_Right(source Slice, cutset Text) (trimmed Slice) {
	defer func() { Slice_Invariants(trimmed, "trim_right.trimmed") }()
	Slice_Invariants(source, "trim_right.source")
	Text_Invariants(cutset, "trim_right.cutset")
	cutset_text := strings.Text(cutset)
	predicate := func(character rune) (matches bool) {
		return bool(strings.Contains_Rune(cutset_text, strings.Character(character)))
	}
	return source[:int(trim_right_boundary(source, predicate))]
}

// Trim_Space removes leading and trailing Unicode white space.
func Trim_Space(source Slice) (trimmed Slice) {
	defer func() { Slice_Invariants(trimmed, "trim_space.trimmed") }()
	Slice_Invariants(source, "trim_space.source")
	space := func(character rune) (yes bool) {
		return bool(ucd.Is_Space(ucd.Character(character)))
	}
	left := int(trim_left_boundary(source, space))
	if left == len(source) {
		return nil
	}
	right := int(trim_right_boundary(source[left:], space))
	return source[left : left+right]
}

// Runes decodes source into Unicode characters and replaces invalid encodings.
func Runes(source Slice) (characters Characters) {
	defer func() { Characters_Invariants(characters, "runes.characters") }()
	Slice_Invariants(source, "runes.source")
	return Characters([]rune(string(source)))
}

// Replace substitutes at most count non-overlapping old Slices.
func Replace(
	source Slice, old Slice, replacement Slice, count Replacement_Count,
) (replaced Slice) {
	defer func() { Slice_Invariants(replaced, "replace.replaced") }()
	Slice_Invariants(source, "replace.source")
	Slice_Invariants(old, "replace.old")
	Slice_Invariants(replacement, "replace.replacement")
	Replacement_Count_Invariants(count, "replace.count")
	result := strings.Replace(
		strings.Text(source), strings.Text(old), strings.Text(replacement),
		strings.Replacement_Count(count),
	)
	if len(result) > SLICE_SIZE_MAXIMUM {
		panic(Error_Too_Large)
	}
	return Slice([]byte(result))
}

// Replace_All substitutes every non-overlapping old Slice.
func Replace_All(source Slice, old Slice, replacement Slice) (replaced Slice) {
	defer func() { Slice_Invariants(replaced, "replace_all.replaced") }()
	Slice_Invariants(source, "replace_all.source")
	Slice_Invariants(old, "replace_all.old")
	Slice_Invariants(replacement, "replace_all.replacement")
	result := strings.Replace_All(
		strings.Text(source), strings.Text(old), strings.Text(replacement),
	)
	if len(result) > SLICE_SIZE_MAXIMUM {
		panic(Error_Too_Large)
	}
	return Slice([]byte(result))
}

// Equal_Fold reports whether two UTF-8 Slices are equal under Unicode simple folding.
func Equal_Fold(left Slice, right Slice) (equal Boolean) {
	defer func() { Boolean_Invariants(equal, "equal_fold.equal") }()
	Slice_Invariants(left, "equal_fold.left")
	Slice_Invariants(right, "equal_fold.right")
	return Boolean(strings.Equal_Fold(strings.Text(left), strings.Text(right)))
}

// Index returns the first separator index or INDEX_ABSENT.
func Index(source Slice, separator Slice) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "index.index") }()
	Slice_Invariants(source, "index.source")
	Slice_Invariants(separator, "index.separator")
	return Index_Value(strings.Index(strings.Text(source), strings.Text(separator)))
}

// Cut divides source around the first separator.
func Cut(source Slice, separator Slice) (before Slice, after Slice, found Boolean) {
	defer func() {
		Slice_Invariants(before, "cut.before")
		Slice_Invariants(after, "cut.after")
		Boolean_Invariants(found, "cut.found")
	}()
	Slice_Invariants(source, "cut.source")
	Slice_Invariants(separator, "cut.separator")
	offset := int(strings.Index(strings.Text(source), strings.Text(separator)))
	if offset < 0 {
		return source, nil, false
	}
	return source[:offset], source[offset+len(separator):], true
}

// Clone copies source and preserves a nil Slice.
func Clone(source Slice) (clone Slice) {
	defer func() { Slice_Invariants(clone, "clone.clone") }()
	Slice_Invariants(source, "clone.source")
	if source == nil {
		return nil
	}
	return append(Slice{}, source...)
}

// Cut_Prefix removes prefix and reports whether it was present.
func Cut_Prefix(source Slice, prefix Slice) (after Slice, found Boolean) {
	defer func() {
		Slice_Invariants(after, "cut_prefix.after")
		Boolean_Invariants(found, "cut_prefix.found")
	}()
	Slice_Invariants(source, "cut_prefix.source")
	Slice_Invariants(prefix, "cut_prefix.prefix")
	if strings.Has_Prefix(strings.Text(source), strings.Text(prefix)) {
		return source[len(prefix):], true
	}
	return source, false
}

// Cut_Suffix removes suffix and reports whether it was present.
func Cut_Suffix(source Slice, suffix Slice) (before Slice, found Boolean) {
	defer func() {
		Slice_Invariants(before, "cut_suffix.before")
		Boolean_Invariants(found, "cut_suffix.found")
	}()
	Slice_Invariants(source, "cut_suffix.source")
	Slice_Invariants(suffix, "cut_suffix.suffix")
	if strings.Has_Suffix(strings.Text(source), strings.Text(suffix)) {
		return source[:len(source)-len(suffix)], true
	}
	return source, false
}

// Lines yields newline-terminated lines and a final unterminated line.
func Lines(source Slice) (sequence iter.Seq[Slice]) {
	Slice_Invariants(source, "lines.source")
	return func(yield func(Slice) (continue_iteration bool)) {
		tail := source
		for len(tail) > 0 {
			offset := int(strings.Index_Byte(strings.Text(tail), strings.Byte('\n')))
			if offset < 0 {
				yield(tail)
				return
			}
			if !yield(tail[:offset+1]) {
				return
			}
			tail = tail[offset+1:]
		}
	}
}

// Split_Sequence yields the same values as Split without a result collection.
func Split_Sequence(source Slice, separator Slice) (sequence iter.Seq[Slice]) {
	Slice_Invariants(source, "split_seq.source")
	Slice_Invariants(separator, "split_seq.separator")
	return split_sequence(source, separator, 0)
}

// Split_After_Sequence yields the Split_After values without a result collection.
func Split_After_Sequence(
	source Slice, separator Slice,
) (sequence iter.Seq[Slice]) {
	Slice_Invariants(source, "split_after_seq.source")
	Slice_Invariants(separator, "split_after_seq.separator")
	return split_sequence(source, separator, Separator_Size(len(separator)))
}

// Fields_Sequence yields the same values as Fields without a result collection.
func Fields_Sequence(source Slice) (sequence iter.Seq[Slice]) {
	Slice_Invariants(source, "fields_seq.source")
	space := func(character rune) (yes bool) {
		return bool(ucd.Is_Space(ucd.Character(character)))
	}
	return fields_sequence(source, space)
}

// Fields_Function_Sequence yields the Fields_Function values without a collection.
func Fields_Function_Sequence(
	source Slice, predicate func(rune) (matches bool),
) (sequence iter.Seq[Slice]) {
	Slice_Invariants(source, "fields_func_seq.source")
	return fields_sequence(source, predicate)
}

func split(
	source Slice, separator Slice, saved Separator_Size, limit Limit,
) (parts Slices) {
	defer func() { Slices_Invariants(parts, "split_internal.parts") }()
	Slice_Invariants(source, "split_internal.source")
	Slice_Invariants(separator, "split_internal.separator")
	Separator_Size_Invariants(saved, "split_internal.saved")
	Limit_Invariants(limit, "split_internal.limit")
	if len(separator) == 0 {
		return Slices(split_empty(source, limit))
	}
	if limit == 0 {
		return nil
	}
	separator_count := int(strings.Count(strings.Text(source), strings.Text(separator)))
	if separator_count == 0 {
		return Slices{source}
	}
	result_count := separator_count + 1
	if limit > 0 {
		if result_count > int(limit) {
			result_count = int(limit)
		}
	}
	parts = make(Slices, result_count)
	tail := source
	for index := 0; index < result_count-1; index++ {
		separator_offset := int(strings.Index(strings.Text(tail), strings.Text(separator)))
		end := separator_offset + int(saved)
		parts[index] = tail[:end:end]
		tail = tail[separator_offset+len(separator):]
	}
	parts[result_count-1] = tail
	return parts
}

func split_empty(source Slice, limit Limit) (parts Empty_Slices) {
	defer func() { Empty_Slices_Invariants(parts, "split_empty.parts") }()
	Slice_Invariants(source, "split_empty.source")
	Limit_Invariants(limit, "split_empty.limit")
	if limit == 0 {
		return nil
	}
	if len(source) == 0 {
		return Empty_Slices{}
	}
	count := int(utf8.Character_Count(utf8.Bytes(source)))
	if limit > 0 {
		if count > int(limit) {
			count = int(limit)
		}
	}
	parts = make(Empty_Slices, 0, count)
	tail := source
	for len(tail) > 0 {
		if limit > 0 {
			if len(parts)+1 == int(limit) {
				parts = append(parts, tail)
				return parts
			}
		}
		_, character_size := utf8.Decode_Character(utf8.Bytes(tail))
		parts = append(parts, tail[:character_size:character_size])
		tail = tail[character_size:]
	}
	return parts
}

func fields_function(
	source Slice, predicate func(rune) (matches bool),
) (fields Field_Slices) {
	defer func() { Field_Slices_Invariants(fields, "fields_function_internal.fields") }()
	Slice_Invariants(source, "fields_function_internal.source")
	start := -1
	for index := 0; index < len(source); {
		character, size := utf8.Decode_Character(utf8.Bytes(source[index:]))
		if predicate(rune(character)) {
			if start >= 0 {
				fields = append(fields, source[start:index:index])
				start = -1
			}
		} else if start < 0 {
			start = index
		}
		index += int(size)
	}
	if start >= 0 {
		fields = append(fields, source[start:len(source):len(source)])
	}
	return fields
}

func trim_left_boundary(
	source Slice, predicate func(rune) (matches bool),
) (end Boundary) {
	defer func() { Boundary_Invariants(end, "trim_left_boundary.end") }()
	Slice_Invariants(source, "trim_left_boundary.source")
	for int(end) < len(source) {
		character, character_size := utf8.Decode_Character(utf8.Bytes(source[int(end):]))
		if !predicate(rune(character)) {
			return end
		}
		end += Boundary(character_size)
	}
	return end
}

func trim_right_boundary(
	source Slice, predicate func(rune) (matches bool),
) (end Boundary) {
	defer func() { Boundary_Invariants(end, "trim_right_boundary.end") }()
	Slice_Invariants(source, "trim_right_boundary.source")
	end = Boundary(len(source))
	for end > 0 {
		character, character_size := utf8.Decode_Final_Character(
			utf8.Bytes(source[:int(end)]),
		)
		if !predicate(rune(character)) {
			return end
		}
		end -= Boundary(character_size)
	}
	return end
}

func split_sequence(
	source Slice, separator Slice, saved Separator_Size,
) (sequence iter.Seq[Slice]) {
	Slice_Invariants(source, "split_sequence_internal.source")
	Slice_Invariants(separator, "split_sequence_internal.separator")
	Separator_Size_Invariants(saved, "split_sequence_internal.saved")
	return func(yield func(Slice) (continue_iteration bool)) {
		if len(separator) == 0 {
			tail := source
			for len(tail) > 0 {
				_, character_size := utf8.Decode_Character(utf8.Bytes(tail))
				if !yield(tail[:character_size:character_size]) {
					return
				}
				tail = tail[character_size:]
			}
			return
		}
		tail := source
		separator_offset := int(strings.Index(strings.Text(tail), strings.Text(separator)))
		for separator_offset >= 0 {
			end := separator_offset + int(saved)
			if !yield(tail[:end:end]) {
				return
			}
			tail = tail[separator_offset+len(separator):]
			separator_offset = int(strings.Index(
				strings.Text(tail), strings.Text(separator),
			))
		}
		yield(tail)
	}
}

func fields_sequence(
	source Slice, predicate func(rune) (matches bool),
) (sequence iter.Seq[Slice]) {
	Slice_Invariants(source, "fields_sequence_internal.source")
	return func(yield func(Slice) (continue_iteration bool)) {
		start := -1
		for index := 0; index < len(source); {
			character, size := utf8.Decode_Character(utf8.Bytes(source[index:]))
			if predicate(rune(character)) {
				if start >= 0 {
					if !yield(source[start:index:index]) {
						return
					}
					start = -1
				}
			} else if start < 0 {
				start = index
			}
			index += int(size)
		}
		if start >= 0 {
			yield(source[start:len(source):len(source)])
		}
	}
}
