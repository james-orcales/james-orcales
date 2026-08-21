// Package bytes manipulates byte slices. It ports the Go standard library bytes package with
// repository names and bounded collection domains.
package bytes

import (
	"unsafe"

	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
)

// SLICE_SIZE_MINIMUM is the size of an empty Slice.
const SLICE_SIZE_MINIMUM = 0

// SLICE_SIZE_MAXIMUM bounds each Slice that the package owns or reads.
const SLICE_SIZE_MAXIMUM = 4 * bits.KIBIBYTE_BYTES

// TEXT_SIZE_MINIMUM is the size of empty Text.
const TEXT_SIZE_MINIMUM = SLICE_SIZE_MINIMUM

// TEXT_SIZE_MAXIMUM keeps string-backed bytes inside the Slice domain.
const TEXT_SIZE_MAXIMUM = SLICE_SIZE_MAXIMUM

// SLICES_COUNT_MINIMUM is the count in an absent split result.
const SLICES_COUNT_MINIMUM = SLICE_SIZE_MINIMUM

// SLICES_COUNT_MAXIMUM includes both sides of a separator at every input byte.
const SLICES_COUNT_MAXIMUM = SLICE_SIZE_MAXIMUM + 1

// INDEX_ABSENT reports that a search found no match.
const INDEX_ABSENT = -1

// INDEX_MAXIMUM is the final byte index in the largest Slice.
const INDEX_MAXIMUM = SLICE_SIZE_MAXIMUM - 1

// BOUNDARY_INDEX_MAXIMUM is the boundary after the final byte.
const BOUNDARY_INDEX_MAXIMUM = SLICE_SIZE_MAXIMUM

// COUNT_VALUE_MINIMUM is the smallest produced item count.
const COUNT_VALUE_MINIMUM = SLICES_COUNT_MINIMUM

// COUNT_VALUE_MAXIMUM occurs when every source byte produces one item.
const COUNT_VALUE_MAXIMUM = SLICE_SIZE_MAXIMUM

// OCCURRENCE_COUNT_MINIMUM is the smallest match count.
const OCCURRENCE_COUNT_MINIMUM = SLICES_COUNT_MINIMUM

// OCCURRENCE_COUNT_MAXIMUM includes both boundaries around every byte.
const OCCURRENCE_COUNT_MAXIMUM = SLICES_COUNT_MAXIMUM

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

// READ_OPERATION_OTHER records a read that cannot restore its full width.
const READ_OPERATION_OTHER = -1

// READ_OPERATION_ABSENT records that no read can be reversed.
const READ_OPERATION_ABSENT = 0

// READ_OPERATION_SINGLE records one reversible byte.
const READ_OPERATION_SINGLE = 1

// READ_OPERATION_MINIMUM records a read whose full width is unavailable.
const READ_OPERATION_MINIMUM = READ_OPERATION_OTHER

// READ_OPERATION_MAXIMUM bounds extension-owned grouped reads.
const READ_OPERATION_MAXIMUM = 4

// GROWTH_COUNT_MINIMUM requests no additional Buffer space.
const GROWTH_COUNT_MINIMUM = SLICE_SIZE_MINIMUM

// GROWTH_COUNT_MAXIMUM fills the largest empty Buffer.
const GROWTH_COUNT_MAXIMUM = SLICE_SIZE_MAXIMUM

// READER_POSITION_MINIMUM is the first Reader position.
const READER_POSITION_MINIMUM int64 = SLICE_SIZE_MINIMUM

// READER_POSITION_MAXIMUM is the boundary after the largest Reader source.
const READER_POSITION_MAXIMUM int64 = SLICE_SIZE_MAXIMUM

// READER_OFFSET_MINIMUM reaches one complete source before an origin.
const READER_OFFSET_MINIMUM int64 = -READER_POSITION_MAXIMUM

// READER_OFFSET_MAXIMUM reaches one complete source after an origin.
const READER_OFFSET_MAXIMUM int64 = READER_POSITION_MAXIMUM

// SIZE_VALUE_MINIMUM is the size of an empty Reader.
const SIZE_VALUE_MINIMUM int64 = READER_POSITION_MINIMUM

// SIZE_VALUE_MAXIMUM is the size of the largest Reader source.
const SIZE_VALUE_MAXIMUM int64 = READER_POSITION_MAXIMUM

// SEEK_FROM_START measures Reader offset from source start.
const SEEK_FROM_START Seek_From = 0

// SEEK_FROM_CURRENT measures Reader offset from current position.
const SEEK_FROM_CURRENT Seek_From = 1

// SEEK_FROM_END measures Reader offset from source end.
const SEEK_FROM_END Seek_From = 2

// Boundary is a byte boundary from the start through the end of a Slice.
type Boundary int

// Boundary_Invariants bounds a byte boundary through the final Slice end.
func Boundary_Invariants(value Boundary, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), SLICE_SIZE_MINIMUM, BOUNDARY_INDEX_MAXIMUM).
		Ensure()
}

// Separator_Size is the part of a separator that a split result retains.
type Separator_Size int

// Separator_Size_Invariants bounds retained separator bytes to one Slice.
func Separator_Size_Invariants(value Separator_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), SLICE_SIZE_MINIMUM, SLICE_SIZE_MAXIMUM).
		Ensure()
}

// Empty_Slices is a split result for an empty separator.
type Empty_Slices []Slice

// Empty_Slices_Invariants bounds results to one item for each source byte.
func Empty_Slices_Invariants(value Empty_Slices, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SLICES_COUNT_MINIMUM, SLICE_SIZE_MAXIMUM).
		Ensure()
}

// Slice is a bounded byte slice.
type Slice []byte

// Slice_Invariants bounds a Slice that the package reads or owns.
func Slice_Invariants(value Slice, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SLICE_SIZE_MINIMUM, SLICE_SIZE_MAXIMUM).
		Ensure()
}

// Text carries bounded string storage without changing its byte meaning.
type Text string

// Text_Invariants bounds Text to the Slice size domain.
func Text_Invariants(value Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Slices is a split result or a collection that Join reads.
type Slices []Slice

// Slices_Invariants bounds the number of Slices in a split or join collection.
func Slices_Invariants(value Slices, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SLICES_COUNT_MINIMUM, SLICES_COUNT_MAXIMUM).
		Ensure()
}

// Index_Value is a byte position or INDEX_ABSENT.
type Index_Value int

// Index_Value_Invariants bounds an index to absence or the final byte.
func Index_Value_Invariants(value Index_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), INDEX_ABSENT, INDEX_MAXIMUM).
		Ensure()
}

// Boundary_Index is a byte position, an end boundary, or INDEX_ABSENT.
type Boundary_Index int

// Boundary_Index_Invariants includes the boundary after the largest Slice.
func Boundary_Index_Invariants(value Boundary_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), INDEX_ABSENT, BOUNDARY_INDEX_MAXIMUM).
		Ensure()
}

// Count_Value is the number of produced split or line items.
type Count_Value int

// Count_Value_Invariants bounds one output item per source byte.
func Count_Value_Invariants(value Count_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_VALUE_MINIMUM, COUNT_VALUE_MAXIMUM).
		Ensure()
}

// Occurrence_Count counts matches, including every empty byte boundary.
type Occurrence_Count int

// Occurrence_Count_Invariants bounds matches through every empty boundary.
func Occurrence_Count_Invariants(value Occurrence_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), OCCURRENCE_COUNT_MINIMUM, OCCURRENCE_COUNT_MAXIMUM,
		).
		Ensure()
}

// Limit is a requested split count. Negative one requests all results.
type Limit int

// Limit_Invariants bounds a split limit.
func Limit_Invariants(value Limit, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), LIMIT_MINIMUM, LIMIT_MAXIMUM).
		Ensure()
}

// Repeat_Count is the number of copies that Repeat writes.
type Repeat_Count int

// Repeat_Count_Invariants bounds the copy count.
func Repeat_Count_Invariants(value Repeat_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), REPEAT_COUNT_MINIMUM, REPEAT_COUNT_MAXIMUM).
		Ensure()
}

// Replacement_Count is a replacement limit. Negative one requests every replacement.
type Replacement_Count int

// Replacement_Count_Invariants bounds a replacement count.
func Replacement_Count_Invariants(value Replacement_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), REPLACEMENT_COUNT_MINIMUM, REPLACEMENT_COUNT_MAXIMUM,
		).
		Ensure()
}

// Order is a lexical comparison result.
type Order int

// Order_Invariants lists the three comparison results.
func Order_Invariants(value Order, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Int(int(value), ORDER_BEFORE, ORDER_EQUAL, ORDER_AFTER).
		Ensure()
}

// Boolean is one true or false report.
type Boolean bool

// Boolean_Invariants records both Boolean states.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A Boolean report is true.").
		Ensure()
}

// Byte is one byte that a search reads.
type Byte byte

// Byte_Invariants states the complete byte domain.
func Byte_Invariants(value Byte, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), bits.WORD_8_MINIMUM, bits.WORD_8_MAXIMUM).
		Ensure()
}

// Available_Slice is writable Buffer storage beyond initialized content. Storage uses full length
// because empty slice would falsely describe zero members while retaining large capacity.
type Available_Slice struct {
	// Storage aliases each writable byte after initialized content.
	Storage Slice
}

// Available_Slice_Invariants composes writable storage range.
func Available_Slice_Invariants(value Available_Slice, namespace aver.Namespace) {
	Slice_Invariants(value.Storage, namespace)
}

// Growth_Count is the capacity that Buffer_Grow reserves.
type Growth_Count int

// Growth_Count_Invariants bounds reserved capacity to one Buffer.
func Growth_Count_Invariants(value Growth_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), GROWTH_COUNT_MINIMUM, GROWTH_COUNT_MAXIMUM).
		Ensure()
}

// Read_Operation records the read that Buffer can reverse.
type Read_Operation int8

// Read_Operation_Invariants bounds reversible widths and other reads.
func Read_Operation_Invariants(value Read_Operation, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int8(int8(value), READ_OPERATION_MINIMUM, READ_OPERATION_MAXIMUM).
		Ensure()
}

// Reader_Position is a nonnegative byte position.
type Reader_Position int64

// Reader_Position_Invariants bounds a Reader position to signed integer storage.
func Reader_Position_Invariants(value Reader_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value), READER_POSITION_MINIMUM, READER_POSITION_MAXIMUM,
		).
		Ensure()
}

// Reader_Offset is a byte offset that Reader_Read_At validates.
type Reader_Offset int64

// Reader_Offset_Invariants bounds an offset to one Reader source in either direction.
func Reader_Offset_Invariants(value Reader_Offset, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(
			int64(value), READER_OFFSET_MINIMUM, READER_OFFSET_MAXIMUM,
		).
		Ensure()
}

// Size_Value is the original byte count in a Reader.
type Size_Value int64

// Size_Value_Invariants bounds a Reader size to one Slice.
func Size_Value_Invariants(value Size_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), SIZE_VALUE_MINIMUM, SIZE_VALUE_MAXIMUM).
		Ensure()
}

// Seek_From selects one Reader offset origin.
type Seek_From uint8

// Seek_From_Invariants states each accepted Reader offset origin.
func Seek_From_Invariants(value Seek_From, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(SEEK_FROM_START), uint8(SEEK_FROM_CURRENT),
			uint8(SEEK_FROM_END),
		).
		Ensure()
}

// Buffer borrows caller storage for sequential reads and writes.
type Buffer struct {
	// Content views initialized caller storage and retains its fixed capacity.
	Content Slice
	// Position selects unread content start.
	Position Boundary
	// Operation records read width that Unread can reverse.
	Operation Read_Operation
}

// Buffer_Invariants keeps cursor state inside caller-owned storage.
func Buffer_Invariants(value Buffer, namespace aver.Namespace) {
	Slice_Invariants(value.Content, namespace)
	Boundary_Invariants(value.Position, namespace)
	Read_Operation_Invariants(value.Operation, namespace)
	aver.Always(
		int(value.Position) <= len(value.Content),
		"Buffer position does not exceed content size.",
	)
	aver.Always(
		cap(value.Content) <= SLICE_SIZE_MAXIMUM,
		"Buffer capacity does not exceed Slice limit.",
	)
}

// Buffer_Handle keeps borrowed buffer state nonnil at every mutation boundary.
type Buffer_Handle *Buffer

// Buffer_Handle_Invariants composes the buffer behind a required handle.
func Buffer_Handle_Invariants(value Buffer_Handle, namespace aver.Namespace) {
	aver.Always(value != nil, "Byte Buffer handle exists.")
	aver.Tree(value, namespace).
		Range_Int(len(value.Content), SLICE_SIZE_MINIMUM, SLICE_SIZE_MAXIMUM).
		Range_Int(int(value.Position), SLICE_SIZE_MINIMUM, BOUNDARY_INDEX_MAXIMUM).
		Range_Int8(
			int8(value.Operation), READ_OPERATION_MINIMUM, READ_OPERATION_MAXIMUM,
		).
		Ensure()
	aver.Always(
		int(value.Position) <= len(value.Content),
		"Buffer handle position does not exceed content size.",
	)
	aver.Always(
		cap(value.Content) <= SLICE_SIZE_MAXIMUM,
		"Buffer handle capacity does not exceed Slice limit.",
	)
	Buffer_Invariants(*value, namespace)
}

// Reader borrows one bounded Slice and tracks its byte cursor.
type Reader struct {
	// Source remains caller-owned.
	Source Slice
	// Position selects unread suffix start.
	Position Reader_Position
	// Previous lets an extension record one grouped read boundary.
	Previous Index_Value
}

// Reader_Invariants keeps the cursor inside caller-owned source.
func Reader_Invariants(value Reader, namespace aver.Namespace) {
	Slice_Invariants(value.Source, namespace)
	Reader_Position_Invariants(value.Position, namespace)
	Index_Value_Invariants(value.Previous, namespace)
	aver.Always(
		int(value.Position) <= len(value.Source),
		"Reader position does not exceed source size.",
	)
}

// Reader_Handle keeps borrowed reader state nonnil at every cursor boundary.
type Reader_Handle *Reader

// Reader_Handle_Invariants composes the reader behind a required handle.
func Reader_Handle_Invariants(value Reader_Handle, namespace aver.Namespace) {
	aver.Always(value != nil, "Byte Reader handle exists.")
	aver.Tree(value, namespace).
		Range_Int(len(value.Source), SLICE_SIZE_MINIMUM, SLICE_SIZE_MAXIMUM).
		Range_Int64(
			int64(value.Position), READER_POSITION_MINIMUM, READER_POSITION_MAXIMUM,
		).
		Range_Int(int(value.Previous), INDEX_ABSENT, INDEX_MAXIMUM).
		Ensure()
	aver.Always(
		int(value.Position) <= len(value.Source),
		"Reader handle position does not exceed source size.",
	)
	Reader_Invariants(*value, namespace)
}

// Buffer_Init borrows storage and copies initial content into it.
func Buffer_Init(buffer Buffer_Handle, storage Slice, content Slice) (count Boundary) {
	defer func() { Boundary_Invariants(count, "buffer_init.count") }()
	Buffer_Handle_Invariants(buffer, "buffer_init.buffer")
	Slice_Invariants(storage, "buffer_init.storage")
	Slice_Invariants(content, "buffer_init.content")
	aver.Always(
		len(content) <= len(storage),
		"Buffer storage holds initial content.",
	)
	buffer.Content = storage[:len(content):len(storage)]
	copy(buffer.Content, content)
	buffer.Position = 0
	buffer.Operation = READ_OPERATION_ABSENT
	return Boundary(len(content))
}

// Buffer_Init_Text borrows storage and copies initial text into it.
func Buffer_Init_Text(buffer Buffer_Handle, storage Slice, content Text) (count Boundary) {
	defer func() { Boundary_Invariants(count, "buffer_init_text.count") }()
	Buffer_Handle_Invariants(buffer, "buffer_init_text.buffer")
	Slice_Invariants(storage, "buffer_init_text.storage")
	Text_Invariants(content, "buffer_init_text.content")
	aver.Always(
		len(content) <= len(storage),
		"Buffer storage holds initial text.",
	)
	buffer.Content = storage[:len(content):len(storage)]
	copy(buffer.Content, content)
	buffer.Position = 0
	buffer.Operation = READ_OPERATION_ABSENT
	return Boundary(len(content))
}

// Buffer_Bytes returns the unread content and aliases Buffer storage.
func Buffer_Bytes(buffer Buffer_Handle) (content Slice) {
	defer func() { Slice_Invariants(content, "buffer_bytes.content") }()
	Buffer_Handle_Invariants(buffer, "buffer_bytes.buffer")
	return buffer.Content[buffer.Position:len(buffer.Content):len(buffer.Content)]
}

// Buffer_Available_Slice returns full writable storage beyond initialized Buffer content.
func Buffer_Available_Slice(buffer Buffer_Handle) (available Available_Slice) {
	defer func() {
		Available_Slice_Invariants(available, "buffer_available_slice.available")
	}()
	Buffer_Handle_Invariants(buffer, "buffer_available_slice.buffer")
	capacity_count := cap(buffer.Content)
	return Available_Slice{
		Storage: buffer.Content[len(buffer.Content):capacity_count:capacity_count],
	}
}

// Buffer_Peek returns up to size unread bytes without state change.
func Buffer_Peek(buffer Buffer_Handle, size Boundary) (content Slice) {
	defer func() { Slice_Invariants(content, "buffer_peek.content") }()
	Buffer_Handle_Invariants(buffer, "buffer_peek.buffer")
	Boundary_Invariants(size, "buffer_peek.size")
	if size > Buffer_Size(buffer) {
		size = Buffer_Size(buffer)
	}
	end := buffer.Position + size
	return buffer.Content[buffer.Position:end:end]
}

// Buffer_Size returns the unread byte count in Buffer.
func Buffer_Size(buffer Buffer_Handle) (count Boundary) {
	defer func() { Boundary_Invariants(count, "buffer_length.count") }()
	Buffer_Handle_Invariants(buffer, "buffer_len.buffer")
	return Boundary(len(buffer.Content)) - buffer.Position
}

// Buffer_Capacity returns the storage capacity of Buffer.
func Buffer_Capacity(buffer Buffer_Handle) (capacity Boundary) {
	defer func() { Boundary_Invariants(capacity, "buffer_capacity.capacity") }()
	Buffer_Handle_Invariants(buffer, "buffer_capacity.buffer")
	return Boundary(cap(buffer.Content))
}

// Buffer_Available returns the unused Buffer capacity.
func Buffer_Available(buffer Buffer_Handle) (available Boundary) {
	defer func() { Boundary_Invariants(available, "buffer_available.available") }()
	Buffer_Handle_Invariants(buffer, "buffer_available.buffer")
	return Boundary(cap(buffer.Content) - len(buffer.Content))
}

// Buffer_Truncate keeps the first size unread bytes.
func Buffer_Truncate(buffer Buffer_Handle, size Boundary) {
	Buffer_Handle_Invariants(buffer, "buffer_truncate.buffer")
	Boundary_Invariants(size, "buffer_truncate.size")
	if size == 0 {
		Buffer_Reset(buffer)
		return
	}
	buffer.Operation = READ_OPERATION_ABSENT
	if size > Buffer_Size(buffer) {
		panic("bytes: truncation out of range")
	}
	buffer.Content = buffer.Content[:buffer.Position+size]
}

// Buffer_Reset makes Buffer empty and keeps its storage.
func Buffer_Reset(buffer Buffer_Handle) {
	Buffer_Handle_Invariants(buffer, "buffer_reset.buffer")
	buffer.Content = buffer.Content[:0]
	buffer.Position = 0
	buffer.Operation = READ_OPERATION_ABSENT
}

// Buffer_Grow reserves count bytes after the current content.
func Buffer_Grow(buffer Buffer_Handle, count Growth_Count) {
	Buffer_Handle_Invariants(buffer, "buffer_grow_public.buffer")
	Growth_Count_Invariants(count, "buffer_grow_public.count")
	buffer_reserve(buffer, count)
}

// Buffer_Write appends source to Buffer.
func Buffer_Write(buffer Buffer_Handle, source Slice) (count Boundary) {
	defer func() { Boundary_Invariants(count, "buffer_write.count") }()
	Buffer_Handle_Invariants(buffer, "buffer_write.buffer")
	Slice_Invariants(source, "buffer_write.source")
	buffer.Operation = READ_OPERATION_ABSENT
	buffer_reserve(buffer, Growth_Count(len(source)))
	content_count := len(buffer.Content)
	buffer.Content = buffer.Content[:content_count+len(source)]
	return Boundary(copy(buffer.Content[content_count:], source))
}

// Buffer_Write_Text appends source to Buffer.
func Buffer_Write_Text(buffer Buffer_Handle, source Text) (count Boundary) {
	defer func() { Boundary_Invariants(count, "buffer_write_text.count") }()
	Buffer_Handle_Invariants(buffer, "buffer_write_string.buffer")
	Text_Invariants(source, "buffer_write_string.source")
	buffer.Operation = READ_OPERATION_ABSENT
	buffer_reserve(buffer, Growth_Count(len(source)))
	content_count := len(buffer.Content)
	buffer.Content = buffer.Content[:content_count+len(source)]
	return Boundary(copy(buffer.Content[content_count:], source))
}

// Buffer_Write_Byte appends value to Buffer.
func Buffer_Write_Byte(buffer Buffer_Handle, value Byte) {
	Buffer_Handle_Invariants(buffer, "buffer_write_byte.buffer")
	Byte_Invariants(value, "buffer_write_byte.value")
	buffer.Operation = READ_OPERATION_ABSENT
	buffer_reserve(buffer, 1)
	content_count := len(buffer.Content)
	buffer.Content = buffer.Content[:content_count+1]
	buffer.Content[content_count] = byte(value)
}

// Buffer_Read copies unread Buffer content into destination.
func Buffer_Read_Into(buffer Buffer_Handle, destination Slice) (count Boundary) {
	defer func() { Boundary_Invariants(count, "buffer_read_into.count") }()
	Buffer_Handle_Invariants(buffer, "buffer_read.buffer")
	Slice_Invariants(destination, "buffer_read.destination")
	buffer.Operation = READ_OPERATION_ABSENT
	if Buffer_Size(buffer) == 0 {
		Buffer_Reset(buffer)
		return 0
	}
	count = Boundary(copy(destination, buffer.Content[buffer.Position:]))
	buffer.Position += count
	if count > 0 {
		buffer.Operation = READ_OPERATION_OTHER
	}
	return count
}

// Buffer_Next returns and consumes up to count bytes.
func Buffer_Next(buffer Buffer_Handle, count Boundary) (content Slice) {
	defer func() { Slice_Invariants(content, "buffer_next.content") }()
	Buffer_Handle_Invariants(buffer, "buffer_next.buffer")
	Boundary_Invariants(count, "buffer_next.count")
	buffer.Operation = READ_OPERATION_ABSENT
	read_size := count
	if read_size > Buffer_Size(buffer) {
		read_size = Buffer_Size(buffer)
	}
	end := buffer.Position + read_size
	content = buffer.Content[buffer.Position:end:end]
	buffer.Position = end
	if read_size > 0 {
		buffer.Operation = READ_OPERATION_OTHER
	}
	return content
}

// Buffer_Read_Byte returns and consumes next Buffer byte.
func Buffer_Read_Byte(buffer Buffer_Handle) (value Byte, found Boolean) {
	defer func() {
		Byte_Invariants(value, "buffer_read_byte.value")
		Boolean_Invariants(found, "buffer_read_byte.found")
	}()
	Buffer_Handle_Invariants(buffer, "buffer_read_byte.buffer")
	if Buffer_Size(buffer) == 0 {
		Buffer_Reset(buffer)
		return 0, false
	}
	value = Byte(buffer.Content[buffer.Position])
	buffer.Position++
	buffer.Operation = READ_OPERATION_OTHER
	return value, true
}

// Buffer_Unread_Byte moves before the last byte from a successful read.
func Buffer_Unread_Byte(buffer Buffer_Handle) {
	Buffer_Handle_Invariants(buffer, "buffer_unread_byte.buffer")
	if buffer.Operation == READ_OPERATION_ABSENT {
		panic("bytes: no byte to unread")
	}
	buffer.Operation = READ_OPERATION_ABSENT
	if buffer.Position > 0 {
		buffer.Position--
	}
}

// Buffer_Read_Until returns aliased content through first delimiter.
func Buffer_Read_Until(
	buffer Buffer_Handle, delimiter Byte,
) (content Slice, found Boolean) {
	defer func() {
		Slice_Invariants(content, "buffer_read_until.content")
		Boolean_Invariants(found, "buffer_read_until.found")
	}()
	Buffer_Handle_Invariants(buffer, "buffer_read_bytes.buffer")
	Byte_Invariants(delimiter, "buffer_read_bytes.delimiter")
	index := Index_Byte(buffer.Content[buffer.Position:], delimiter)
	end := Boundary(len(buffer.Content))
	if index >= 0 {
		end = buffer.Position + Boundary(index) + 1
		found = true
	}
	content = buffer.Content[buffer.Position:end:end]
	buffer.Position = end
	buffer.Operation = READ_OPERATION_ABSENT
	if len(content) > 0 {
		buffer.Operation = READ_OPERATION_OTHER
	}
	return content, found
}

// Reader_Unread_Size returns the unread byte count in Reader.
func Reader_Unread_Size(reader Reader_Handle) (count Boundary) {
	defer func() { Boundary_Invariants(count, "reader_length.count") }()
	Reader_Handle_Invariants(reader, "reader_len.reader")
	if reader.Position >= Reader_Position(len(reader.Source)) {
		return 0
	}
	return Boundary(len(reader.Source) - int(reader.Position))
}

// Reader_Size returns the original source size.
func Reader_Size(reader Reader_Handle) (size Size_Value) {
	defer func() { Size_Value_Invariants(size, "reader_size.size") }()
	Reader_Handle_Invariants(reader, "reader_size.reader")
	return Size_Value(len(reader.Source))
}

// Reader_Read_Into copies unread Reader content into destination.
func Reader_Read_Into(reader Reader_Handle, destination Slice) (count Boundary) {
	defer func() { Boundary_Invariants(count, "reader_read_into.count") }()
	Reader_Handle_Invariants(reader, "reader_read.reader")
	Slice_Invariants(destination, "reader_read.destination")
	if reader.Position >= Reader_Position(len(reader.Source)) {
		return 0
	}
	reader.Previous = INDEX_ABSENT
	count = Boundary(copy(destination, reader.Source[reader.Position:]))
	reader.Position += Reader_Position(count)
	return count
}

// Reader_Read_At copies Reader content at position without a state change.
func Reader_Read_At_Into(
	reader Reader_Handle,
	destination Slice,
	offset Reader_Offset,
) (count Boundary) {
	defer func() { Boundary_Invariants(count, "reader_read_at_into.count") }()
	Reader_Handle_Invariants(reader, "reader_read_at.reader")
	Slice_Invariants(destination, "reader_read_at.destination")
	Reader_Offset_Invariants(offset, "reader_read_at.offset")
	if offset < 0 {
		panic("bytes: invalid Reader offset")
	}
	position := Reader_Position(offset)
	if position > Reader_Position(len(reader.Source)) {
		panic("bytes: invalid Reader offset")
	}
	if position == Reader_Position(len(reader.Source)) {
		return 0
	}
	return Boundary(copy(destination, reader.Source[position:]))
}

// Reader_Read_Byte returns and consumes the next Reader byte.
func Reader_Read_Byte(reader Reader_Handle) (value Byte, found Boolean) {
	defer func() {
		Byte_Invariants(value, "reader_read_byte.value")
		Boolean_Invariants(found, "reader_read_byte.found")
	}()
	Reader_Handle_Invariants(reader, "reader_read_byte.reader")
	reader.Previous = INDEX_ABSENT
	if reader.Position >= Reader_Position(len(reader.Source)) {
		return 0, false
	}
	value = Byte(reader.Source[reader.Position])
	reader.Position++
	return value, true
}

// Reader_Unread_Byte moves Reader back by one byte.
func Reader_Unread_Byte(reader Reader_Handle) {
	Reader_Handle_Invariants(reader, "reader_unread_byte.reader")
	if reader.Position <= 0 {
		panic("bytes: no Reader byte to unread")
	}
	reader.Previous = INDEX_ABSENT
	reader.Position--
}

// Reader_Seek sets next Reader position from one origin.
func Reader_Seek(
	reader Reader_Handle,
	offset Reader_Offset,
	origin Seek_From,
) (position Reader_Position) {
	defer func() { Reader_Position_Invariants(position, "reader_seek.position") }()
	Reader_Handle_Invariants(reader, "reader_seek.reader")
	Reader_Offset_Invariants(offset, "reader_seek.offset")
	Seek_From_Invariants(origin, "reader_seek.origin")
	reader.Previous = INDEX_ABSENT
	target := int64(offset)
	if origin == SEEK_FROM_CURRENT {
		target = int64(reader.Position) + int64(offset)
	} else if origin == SEEK_FROM_END {
		target = int64(len(reader.Source)) + int64(offset)
	}
	if target < 0 {
		panic("bytes: invalid Reader offset")
	}
	if target > int64(len(reader.Source)) {
		panic("bytes: invalid Reader offset")
	}
	position = Reader_Position(target)
	reader.Position = position
	return position
}

// Reader_Reset replaces Reader source and returns to its first byte.
func Reader_Reset(reader Reader_Handle, source Slice) {
	Reader_Handle_Invariants(reader, "reader_reset.reader")
	Slice_Invariants(source, "reader_reset.source")
	reader.Source = source
	reader.Position = 0
	reader.Previous = INDEX_ABSENT
}

func buffer_reserve(buffer Buffer_Handle, count Growth_Count) {
	Buffer_Handle_Invariants(buffer, "buffer_grow.buffer")
	Growth_Count_Invariants(count, "buffer_grow.count")
	unread_size := int(Buffer_Size(buffer))
	if int(count) > SLICE_SIZE_MAXIMUM-unread_size {
		panic("bytes: result too large")
	}
	if unread_size == 0 {
		Buffer_Reset(buffer)
	}
	if int(count) <= cap(buffer.Content)-len(buffer.Content) {
		return
	}
	if unread_size+int(count) <= cap(buffer.Content) {
		copy(buffer.Content, buffer.Content[buffer.Position:])
		buffer.Position = 0
		buffer.Content = buffer.Content[:unread_size]
		return
	}
	panic("bytes: destination too small")
}

// Equal reports whether two Slices have the same bytes. Nil and empty are equal.
func Equal(left Slice, right Slice) (equal Boolean) {
	defer func() { Boolean_Invariants(equal, "equal.equal") }()
	Slice_Invariants(left, "equal.left")
	Slice_Invariants(right, "equal.right")
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
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
func Count(source Slice, separator Slice) (count Occurrence_Count) {
	defer func() { Occurrence_Count_Invariants(count, "count.count") }()
	Slice_Invariants(source, "count.source")
	Slice_Invariants(separator, "count.separator")
	if len(separator) == 0 {
		return Occurrence_Count(len(source) + 1)
	}
	tail := source
	for len(tail) >= len(separator) {
		separator_index := Index(tail, separator)
		if separator_index == INDEX_ABSENT {
			break
		}
		count++
		tail = tail[int(separator_index)+len(separator):]
	}
	return count
}

// Contains reports whether a separator occurs in source.
func Contains(source Slice, separator Slice) (contained Boolean) {
	defer func() { Boolean_Invariants(contained, "contains.contained") }()
	Slice_Invariants(source, "contains.source")
	Slice_Invariants(separator, "contains.separator")
	return Boolean(Index(source, separator) >= 0)
}

// Index_Byte returns the first index of value or INDEX_ABSENT.
func Index_Byte(source Slice, value Byte) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "index_byte.index") }()
	Slice_Invariants(source, "index_byte.source")
	Byte_Invariants(value, "index_byte.value")
	for source_index, source_value := range source {
		if Byte(source_value) == value {
			return Index_Value(source_index)
		}
	}
	return INDEX_ABSENT
}

// Last_Index returns the last separator index, including the end boundary for empty separator.
func Last_Index(source Slice, separator Slice) (index Boundary_Index) {
	defer func() { Boundary_Index_Invariants(index, "last_index.index") }()
	Slice_Invariants(source, "last_index.source")
	Slice_Invariants(separator, "last_index.separator")
	if len(separator) == 0 {
		return Boundary_Index(len(source))
	}
	if len(separator) > len(source) {
		return INDEX_ABSENT
	}
	for source_index := len(source) - len(separator); source_index >= 0; source_index-- {
		if Equal(source[source_index:source_index+len(separator)], separator) {
			return Boundary_Index(source_index)
		}
	}
	return INDEX_ABSENT
}

// Last_Index_Byte returns the last index of value or INDEX_ABSENT.
func Last_Index_Byte(source Slice, value Byte) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "last_index_byte.index") }()
	Slice_Invariants(source, "last_index_byte.source")
	Byte_Invariants(value, "last_index_byte.value")
	for source_index := len(source) - 1; source_index >= 0; source_index-- {
		if Byte(source[source_index]) == value {
			return Index_Value(source_index)
		}
	}
	return INDEX_ABSENT
}

// Split_Into fills caller slots at every non-overlapping separator.
func Split_Into(
	destination Slices, source Slice, separator Slice,
) (count Count_Value) {
	defer func() { Count_Value_Invariants(count, "split_into.count") }()
	Slices_Invariants(destination, "split_into.destination")
	Slice_Invariants(source, "split_into.source")
	Slice_Invariants(separator, "split_into.separator")
	return Split_N_Into(destination, source, separator, LIMIT_MINIMUM)
}

// Split_N_Into fills at most limit caller slots.
func Split_N_Into(
	destination Slices, source Slice, separator Slice, limit Limit,
) (count Count_Value) {
	defer func() { Count_Value_Invariants(count, "split_n_into.count") }()
	Slices_Invariants(destination, "split_n_into.destination")
	Slice_Invariants(source, "split_n_into.source")
	Slice_Invariants(separator, "split_n_into.separator")
	Limit_Invariants(limit, "split_n_into.limit")
	return split_into(destination, source, separator, limit, false)
}

// Split_After_Into retains separator inside each preceding source view.
func Split_After_Into(
	destination Slices, source Slice, separator Slice,
) (count Count_Value) {
	defer func() { Count_Value_Invariants(count, "split_after_into.count") }()
	Slices_Invariants(destination, "split_after_into.destination")
	Slice_Invariants(source, "split_after_into.source")
	Slice_Invariants(separator, "split_after_into.separator")
	return Split_After_N_Into(destination, source, separator, LIMIT_MINIMUM)
}

// Split_After_N_Into retains separator while applying result limit.
func Split_After_N_Into(
	destination Slices, source Slice, separator Slice, limit Limit,
) (count Count_Value) {
	defer func() { Count_Value_Invariants(count, "split_after_n_into.count") }()
	Slices_Invariants(destination, "split_after_n_into.destination")
	Slice_Invariants(source, "split_after_n_into.source")
	Slice_Invariants(separator, "split_after_n_into.separator")
	Limit_Invariants(limit, "split_after_n_into.limit")
	return split_into(destination, source, separator, limit, true)
}

func split_into(
	destination Slices, source Slice, separator Slice, limit Limit, after Boolean,
) (count Count_Value) {
	defer func() { Count_Value_Invariants(count, "split_internal.count") }()
	Slices_Invariants(destination, "split_internal.destination")
	Slice_Invariants(source, "split_internal.source")
	Slice_Invariants(separator, "split_internal.separator")
	Limit_Invariants(limit, "split_internal.limit")
	Boolean_Invariants(after, "split_internal.after")
	if len(separator) == 0 {
		return split_empty_into(destination, source, limit)
	}
	if limit == 0 {
		return 0
	}
	tail := source
	for limit < 0 || int(count) < int(limit)-1 {
		separator_index := Index(tail, separator)
		if separator_index == INDEX_ABSENT {
			break
		}
		end := int(separator_index)
		if after {
			end += len(separator)
		}
		if int(count) == len(destination) {
			panic("bytes: destination too small")
		}
		destination[int(count)] = tail[:end:end]
		count++
		tail = tail[int(separator_index)+len(separator):]
	}
	if int(count) == len(destination) {
		panic("bytes: destination too small")
	}
	destination[int(count)] = tail
	return count + 1
}

func split_empty_into(
	destination Slices, source Slice, limit Limit,
) (count Count_Value) {
	defer func() { Count_Value_Invariants(count, "split_empty_into.count") }()
	Slices_Invariants(destination, "split_empty_into.destination")
	Slice_Invariants(source, "split_empty_into.source")
	Limit_Invariants(limit, "split_empty_into.limit")
	if limit == 0 {
		return 0
	}
	if len(source) == 0 {
		return 0
	}
	tail := source
	for len(tail) > 0 {
		if limit > 0 {
			if int(count)+1 == int(limit) {
				break
			}
		}
		if int(count) == len(destination) {
			panic("bytes: destination too small")
		}
		destination[int(count)] = tail[:1:1]
		count++
		tail = tail[1:]
	}
	if len(tail) == 0 {
		return count
	}
	if int(count) == len(destination) {
		panic("bytes: destination too small")
	}
	destination[int(count)] = tail
	return count + 1
}

// Join_Into writes parts and separators into caller storage.
func Join_Into(
	destination Slice, parts Slices, separator Slice,
) (count Boundary) {
	defer func() { Boundary_Invariants(count, "join_into.count") }()
	Slice_Invariants(destination, "join_into.destination")
	Slices_Invariants(parts, "join_into.parts")
	Slice_Invariants(separator, "join_into.separator")
	result_size := 0
	for _, part := range parts {
		Slice_Invariants(part, "join_into.part")
		if len(part) > SLICE_SIZE_MAXIMUM-result_size {
			panic("bytes: result too large")
		}
		result_size += len(part)
	}
	if len(parts) > 1 {
		if len(separator) > 0 {
			separator_count := len(parts) - 1
			if separator_count > (SLICE_SIZE_MAXIMUM-result_size)/len(separator) {
				panic("bytes: result too large")
			}
			result_size += len(separator) * separator_count
		}
	}
	aver.Always(
		result_size <= len(destination),
		"Join destination holds complete result.",
	)
	for _, part := range parts {
		aver.Always(
			!Overlap(destination[:result_size], part),
			"Join destination does not overlap a part.",
		)
	}
	aver.Always(
		!Overlap(destination[:result_size], separator),
		"Join destination does not overlap separator.",
	)
	written := 0
	for part_index, part := range parts {
		if part_index > 0 {
			written += copy(destination[written:], separator)
		}
		written += copy(destination[written:], part)
	}
	return Boundary(written)
}

// Has_Prefix reports whether source starts with prefix.
func Has_Prefix(source Slice, prefix Slice) (present Boolean) {
	defer func() { Boolean_Invariants(present, "has_prefix.present") }()
	Slice_Invariants(source, "has_prefix.source")
	Slice_Invariants(prefix, "has_prefix.prefix")
	if len(prefix) > len(source) {
		return false
	}
	return Equal(source[:len(prefix)], prefix)
}

// Has_Suffix reports whether source ends with suffix.
func Has_Suffix(source Slice, suffix Slice) (present Boolean) {
	defer func() { Boolean_Invariants(present, "has_suffix.present") }()
	Slice_Invariants(source, "has_suffix.source")
	Slice_Invariants(suffix, "has_suffix.suffix")
	if len(suffix) > len(source) {
		return false
	}
	return Equal(source[len(source)-len(suffix):], suffix)
}

// Has_Text_Suffix avoids allocating a byte copy when a caller already owns string input.
func Has_Text_Suffix(source Slice, suffix Text) (present Boolean) {
	defer func() { Boolean_Invariants(present, "has_text_suffix.present") }()
	Slice_Invariants(source, "has_text_suffix.source")
	Text_Invariants(suffix, "has_text_suffix.suffix")
	if len(suffix) > len(source) {
		return false
	}
	start := len(source) - len(suffix)
	for index := range suffix {
		if source[start+index] != suffix[index] {
			return false
		}
	}
	return true
}

// Repeat_Into writes count consecutive source copies into caller storage.
func Repeat_Into(
	destination Slice, source Slice, count Repeat_Count,
) (result_count Boundary) {
	defer func() { Boundary_Invariants(result_count, "repeat_into.result_count") }()
	Slice_Invariants(destination, "repeat_into.destination")
	Slice_Invariants(source, "repeat_into.source")
	Repeat_Count_Invariants(count, "repeat_into.count")
	if len(source) > 0 {
		if int(count) > SLICE_SIZE_MAXIMUM/len(source) {
			panic("bytes: result too large")
		}
		if int(count) > len(destination)/len(source) {
			panic("bytes: destination too small")
		}
	}
	written := len(source) * int(count)
	if written == 0 {
		return 0
	}
	copied := copy(destination[:written], source)
	for copied < written {
		copied += copy(destination[copied:written], destination[:copied])
	}
	return Boundary(written)
}

// Trim_Prefix removes prefix when source starts with it.
func Trim_Prefix(source Slice, prefix Slice) (trimmed Slice) {
	defer func() { Slice_Invariants(trimmed, "trim_prefix.trimmed") }()
	Slice_Invariants(source, "trim_prefix.source")
	Slice_Invariants(prefix, "trim_prefix.prefix")
	if Has_Prefix(source, prefix) {
		return source[len(prefix):]
	}
	return source
}

// Trim_Suffix removes suffix when source ends with it.
func Trim_Suffix(source Slice, suffix Slice) (trimmed Slice) {
	defer func() { Slice_Invariants(trimmed, "trim_suffix.trimmed") }()
	Slice_Invariants(source, "trim_suffix.source")
	Slice_Invariants(suffix, "trim_suffix.suffix")
	if Has_Suffix(source, suffix) {
		return source[:len(source)-len(suffix)]
	}
	return source
}

// Replace_Into writes bounded non-overlapping substitutions into caller storage.
func Replace_Into(
	destination Slice, source Slice, old Slice, replacement Slice,
	count Replacement_Count,
) (result_count Boundary) {
	defer func() { Boundary_Invariants(result_count, "replace_into.result_count") }()
	Slice_Invariants(destination, "replace_into.destination")
	Slice_Invariants(source, "replace_into.source")
	Slice_Invariants(old, "replace_into.old")
	Slice_Invariants(replacement, "replace_into.replacement")
	Replacement_Count_Invariants(count, "replace_into.count")
	aver.Always(
		!Overlap(destination, source),
		"Replace destination does not overlap source.",
	)
	aver.Always(
		!Overlap(destination, old),
		"Replace destination does not overlap old value.",
	)
	aver.Always(
		!Overlap(destination, replacement),
		"Replace destination does not overlap replacement.",
	)
	match_count := Count(source, old)
	if count < 0 {
		count = Replacement_Count(match_count)
	} else if Occurrence_Count(count) > match_count {
		count = Replacement_Count(match_count)
	}
	written := 0
	source_count := 0
	for replacement_index := Replacement_Count(0); replacement_index < count; {
		prefix_count := 0
		if len(old) == 0 {
			if replacement_index > 0 {
				if source_count < len(source) {
					prefix_count = 1
				}
			}
		} else {
			prefix_count = int(Index(source[source_count:], old))
		}
		if prefix_count > len(destination)-written {
			panic("bytes: destination too small")
		}
		boundary_count := source_count + prefix_count
		written += copy(destination[written:], source[source_count:boundary_count])
		if len(replacement) > len(destination)-written {
			panic("bytes: destination too small")
		}
		written += copy(destination[written:], replacement)
		source_count = boundary_count + len(old)
		replacement_index++
	}
	if len(source)-source_count > len(destination)-written {
		panic("bytes: destination too small")
	}
	written += copy(destination[written:], source[source_count:])
	return Boundary(written)
}

// Replace_All_Into writes every non-overlapping substitution into caller storage.
func Replace_All_Into(
	destination Slice, source Slice, old Slice, replacement Slice,
) (count Boundary) {
	defer func() { Boundary_Invariants(count, "replace_all_into.count") }()
	Slice_Invariants(destination, "replace_all_into.destination")
	Slice_Invariants(source, "replace_all_into.source")
	Slice_Invariants(old, "replace_all_into.old")
	Slice_Invariants(replacement, "replace_all_into.replacement")
	return Replace_Into(
		destination, source, old, replacement, REPLACEMENT_COUNT_MINIMUM,
	)
}

// Index returns the first separator index or INDEX_ABSENT.
func Index(source Slice, separator Slice) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "index.index") }()
	Slice_Invariants(source, "index.source")
	Slice_Invariants(separator, "index.separator")
	if len(separator) == 0 {
		return 0
	}
	if len(separator) > len(source) {
		return INDEX_ABSENT
	}
	final_index := len(source) - len(separator)
	for source_index := 0; source_index <= final_index; source_index++ {
		if Equal(source[source_index:source_index+len(separator)], separator) {
			return Index_Value(source_index)
		}
	}
	return INDEX_ABSENT
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
	offset := int(Index(source, separator))
	if offset < 0 {
		return source, nil, false
	}
	return source[:offset], source[offset+len(separator):], true
}

// Clone_Into copies source into caller storage.
func Clone_Into(destination Slice, source Slice) (count Boundary) {
	defer func() { Boundary_Invariants(count, "clone_into.count") }()
	Slice_Invariants(destination, "clone_into.destination")
	Slice_Invariants(source, "clone_into.source")
	aver.Always(
		len(source) <= len(destination),
		"Clone destination holds source.",
	)
	copy(destination, source)
	return Boundary(len(source))
}

// Cut_Prefix removes prefix and reports whether it was present.
func Cut_Prefix(source Slice, prefix Slice) (after Slice, found Boolean) {
	defer func() {
		Slice_Invariants(after, "cut_prefix.after")
		Boolean_Invariants(found, "cut_prefix.found")
	}()
	Slice_Invariants(source, "cut_prefix.source")
	Slice_Invariants(prefix, "cut_prefix.prefix")
	if Has_Prefix(source, prefix) {
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
	if Has_Suffix(source, suffix) {
		return source[:len(source)-len(suffix)], true
	}
	return source, false
}

// Yield_Function receives one aliased source view until it rejects continuation.
type Yield_Function func(content Slice) (continued Boolean)

// Lines synchronously yields newline-terminated source views.
func Lines(source Slice, yield Yield_Function) (count Count_Value) {
	defer func() { Count_Value_Invariants(count, "lines.count") }()
	Slice_Invariants(source, "lines.source")
	tail := source
	for len(tail) > 0 {
		offset := Index_Byte(tail, '\n')
		if offset == INDEX_ABSENT {
			yield_content(yield, tail[:len(tail):len(tail)])
			return count + 1
		}
		end := int(offset) + 1
		continued := yield_content(yield, tail[:end:end])
		count++
		if !continued {
			return count
		}
		tail = tail[end:]
	}
	return count
}

// Split_Sequence synchronously yields split source views.
func Split_Sequence(
	source Slice, separator Slice, yield Yield_Function,
) (count Count_Value) {
	defer func() { Count_Value_Invariants(count, "split_sequence.count") }()
	Slice_Invariants(source, "split_sequence.source")
	Slice_Invariants(separator, "split_sequence.separator")
	return split_sequence(source, separator, 0, yield)
}

// Split_After_Sequence retains separator inside preceding yielded view.
func Split_After_Sequence(
	source Slice, separator Slice, yield Yield_Function,
) (count Count_Value) {
	defer func() { Count_Value_Invariants(count, "split_after_sequence.count") }()
	Slice_Invariants(source, "split_after_sequence.source")
	Slice_Invariants(separator, "split_after_sequence.separator")
	return split_sequence(source, separator, Separator_Size(len(separator)), yield)
}

func split_sequence(
	source Slice, separator Slice, saved Separator_Size, yield Yield_Function,
) (count Count_Value) {
	defer func() { Count_Value_Invariants(count, "split_sequence_internal.count") }()
	Slice_Invariants(source, "split_sequence_internal.source")
	Slice_Invariants(separator, "split_sequence_internal.separator")
	Separator_Size_Invariants(saved, "split_sequence_internal.saved")
	if len(separator) == 0 {
		tail := source
		for len(tail) > 0 {
			continued := yield_content(yield, tail[:1:1])
			count++
			if !continued {
				return count
			}
			tail = tail[1:]
		}
		return count
	}
	tail := source
	separator_offset := Index(tail, separator)
	for separator_offset >= 0 {
		end := int(separator_offset) + int(saved)
		continued := yield_content(yield, tail[:end:end])
		count++
		if !continued {
			return count
		}
		tail = tail[int(separator_offset)+len(separator):]
		separator_offset = Index(tail, separator)
	}
	yield_content(yield, tail[:len(tail):len(tail)])
	return count + 1
}

func yield_content(yield Yield_Function, content Slice) (continued Boolean) {
	defer func() { Boolean_Invariants(continued, "yield_content.continued") }()
	Slice_Invariants(content, "yield_content.content")
	return yield(content)
}

// Overlap uses address distance because nested scans let hostile maximum slices multiply work.
func Overlap(left Slice, right Slice) (overlap Boolean) {
	defer func() { Boolean_Invariants(overlap, "overlap.overlap") }()
	Slice_Invariants(left, "overlap.left")
	Slice_Invariants(right, "overlap.right")
	if len(left) == SLICE_SIZE_MINIMUM {
		return false
	}
	if len(right) == SLICE_SIZE_MINIMUM {
		return false
	}
	left_address := uintptr(unsafe.Pointer(unsafe.SliceData(left)))
	right_address := uintptr(unsafe.Pointer(unsafe.SliceData(right)))
	if left_address < right_address {
		return Boolean(right_address-left_address < uintptr(len(left)))
	}
	return Boolean(left_address-right_address < uintptr(len(right)))
}
