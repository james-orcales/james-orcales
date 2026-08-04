// Package strings supplies bounded text operations through repository names.
// Stateful values use shared I/O streams, so they implement no standard-library interface.
package strings

import (
	standard_binary "encoding/binary"
	"errors"
	"iter"

	invariant "local/james-orcales/shared/invariant/default"
	shared_io "local/james-orcales/shared/io"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/unicode/ucd"
	"local/james-orcales/shared/unicode/utf8"
)

// Error_Too_Large identifies an owned result that exceeds the package memory budget.
var Error_Too_Large = errors.New("strings: result too large")

// TEXT_SIZE_MINIMUM keeps empty text in the text domain.
const TEXT_SIZE_MINIMUM = utf8.SEQUENCE_SIZE_MINIMUM

// TEXT_SIZE_MAXIMUM gives each text boundary a deterministic memory budget.
const TEXT_SIZE_MAXIMUM = utf8.SEQUENCE_SIZE_MAXIMUM

// SLICE_SIZE_MINIMUM keeps an empty stream buffer in the slice domain.
const SLICE_SIZE_MINIMUM = TEXT_SIZE_MINIMUM

// SLICE_SIZE_MAXIMUM matches the text budget at stream boundaries.
const SLICE_SIZE_MAXIMUM = TEXT_SIZE_MAXIMUM

// TEXTS_COUNT_MINIMUM permits an absent split result.
const TEXTS_COUNT_MINIMUM = 0

// TEXTS_COUNT_MAXIMUM includes both sides of every one-byte separator.
const TEXTS_COUNT_MAXIMUM = TEXT_SIZE_MAXIMUM + 1

// FIELDS_COUNT_MINIMUM permits an empty field result.
const FIELDS_COUNT_MINIMUM = 0

// FIELDS_COUNT_MAXIMUM permits alternating one-byte fields and separators.
const FIELDS_COUNT_MAXIMUM = TEXTS_COUNT_MAXIMUM / 2

// INDEX_ABSENT distinguishes an absent match from the first byte.
const INDEX_ABSENT = -1

// INDEX_MAXIMUM is the final byte index in the largest Text.
const INDEX_MAXIMUM = TEXT_SIZE_MAXIMUM - 1

// BOUNDARY_INDEX_MAXIMUM includes the boundary after the final byte.
const BOUNDARY_INDEX_MAXIMUM = TEXT_SIZE_MAXIMUM

// COUNT_VALUE_MINIMUM permits no match.
const COUNT_VALUE_MINIMUM = 0

// COUNT_VALUE_MAXIMUM includes every empty character boundary.
const COUNT_VALUE_MAXIMUM = TEXTS_COUNT_MAXIMUM

// LIMIT_MINIMUM uses the standard negative-one value for all split results.
const LIMIT_MINIMUM = -1

// LIMIT_MAXIMUM keeps a requested collection inside the package budget.
const LIMIT_MAXIMUM = TEXT_SIZE_MAXIMUM

// REPEAT_COUNT_MINIMUM permits no copy.
const REPEAT_COUNT_MINIMUM = 0

// REPEAT_COUNT_MAXIMUM lets one-byte text fill the complete budget.
const REPEAT_COUNT_MAXIMUM = TEXT_SIZE_MAXIMUM

// REPLACEMENT_COUNT_MINIMUM uses negative one for all replacements.
const REPLACEMENT_COUNT_MINIMUM = LIMIT_MINIMUM

// REPLACEMENT_COUNT_MAXIMUM keeps a replacement request inside the text budget.
const REPLACEMENT_COUNT_MAXIMUM = TEXT_SIZE_MAXIMUM

// REPLACEMENT_PAIRS_COUNT_MINIMUM permits a Replacer with no rules.
const REPLACEMENT_PAIRS_COUNT_MINIMUM = 0

// REPLACEMENT_PAIRS_COUNT_MAXIMUM permits one rule for each input byte.
const REPLACEMENT_PAIRS_COUNT_MAXIMUM = TEXT_SIZE_MAXIMUM

// REPLACEMENT_PAIR_SIZE keeps the match and substitute together.
const REPLACEMENT_PAIR_SIZE = 2

// ORDER_BEFORE normalizes every negative lexical result.
const ORDER_BEFORE = -1

// ORDER_EQUAL identifies equal text.
const ORDER_EQUAL = 0

// ORDER_AFTER normalizes every positive lexical result.
const ORDER_AFTER = 1

// CHARACTER_MINIMUM includes every value that rune storage can hold.
const CHARACTER_MINIMUM int32 = bits.INTEGER_32_MINIMUM

// CHARACTER_MAXIMUM includes every value that rune storage can hold.
const CHARACTER_MAXIMUM int32 = bits.INTEGER_32_MAXIMUM

// DECODED_CHARACTER_MINIMUM is the first Unicode code point.
const DECODED_CHARACTER_MINIMUM int32 = utf8.DECODED_CHARACTER_MINIMUM

// DECODED_CHARACTER_MAXIMUM is the final Unicode code point.
const DECODED_CHARACTER_MAXIMUM int32 = utf8.DECODED_CHARACTER_MAXIMUM

// CHARACTER_SIZE_MINIMUM reports no character at the end of a Reader.
const CHARACTER_SIZE_MINIMUM = utf8.DECODED_SIZE_MINIMUM

// CHARACTER_SIZE_MAXIMUM is the longest UTF-8 encoding.
const CHARACTER_SIZE_MAXIMUM = utf8.DECODED_SIZE_MAXIMUM

// BYTE_MINIMUM is the first byte value.
const BYTE_MINIMUM uint8 = bits.WORD_8_MINIMUM

// BYTE_MAXIMUM is the final byte value.
const BYTE_MAXIMUM uint8 = bits.WORD_8_MAXIMUM

// WORD_BYTE_COUNT lets a byte search test one 64-bit word at each step.
const WORD_BYTE_COUNT = bits.BIT_COUNT_64_MAXIMUM / bits.BIT_COUNT_8_MAXIMUM

// BYTE_LOW_BITS puts the low bit in each byte of a machine word.
const BYTE_LOW_BITS uint64 = 0x0101010101010101

// BYTE_HIGH_BITS puts the high bit in each byte of a machine word.
const BYTE_HIGH_BITS uint64 = 0x8080808080808080

// GROWTH_COUNT_MINIMUM permits a capacity check without new storage.
const GROWTH_COUNT_MINIMUM = 0

// BUILDER_CAPACITY_MAXIMUM keeps Builder allocation inside the text budget.
const BUILDER_CAPACITY_MAXIMUM = SLICE_SIZE_MAXIMUM

// GROWTH_COUNT_MAXIMUM permits one empty Builder to reserve the complete budget.
const GROWTH_COUNT_MAXIMUM = BUILDER_CAPACITY_MAXIMUM

// READER_POSITION_MINIMUM is the first Reader byte.
const READER_POSITION_MINIMUM int64 = 0

// READER_POSITION_MAXIMUM is the boundary after the largest Reader source.
const READER_POSITION_MAXIMUM int64 = BOUNDARY_INDEX_MAXIMUM

// STREAM_OFFSET_MINIMUM admits every negative shared stream offset.
const STREAM_OFFSET_MINIMUM int64 = bits.INTEGER_64_MINIMUM

// STREAM_OFFSET_MAXIMUM admits every positive shared stream offset.
const STREAM_OFFSET_MAXIMUM int64 = bits.INTEGER_64_MAXIMUM

// STREAM_COUNT_MINIMUM is the smallest successful stream byte count.
const STREAM_COUNT_MINIMUM int64 = 0

// STREAM_COUNT_MAXIMUM is the largest bounded stream byte count.
const STREAM_COUNT_MAXIMUM int64 = READER_POSITION_MAXIMUM

// STREAM_STATE_OPEN permits stream operations.
const STREAM_STATE_OPEN uint8 = 0

// STREAM_STATE_CLOSED rejects stream operations after Close or Destroy.
const STREAM_STATE_CLOSED uint8 = 1

// STREAM_STATE_SIZE stores one lifecycle marker.
const STREAM_STATE_SIZE = shared_io.STREAM_BYTE_SIZE

// BUILDER_STATE_SIZE stores the stream lifecycle marker.
const BUILDER_STATE_SIZE = STREAM_STATE_SIZE

// READER_POSITION_OFFSET is the first byte of the encoded cursor.
const READER_POSITION_OFFSET = 0

// READER_POSITION_SIZE stores the encoded cursor.
const READER_POSITION_SIZE = bits.BIT_COUNT_64_MAXIMUM / bits.BIT_COUNT_8_MAXIMUM

// READER_PREVIOUS_OFFSET follows the encoded cursor.
const READER_PREVIOUS_OFFSET = READER_POSITION_OFFSET + READER_POSITION_SIZE

// READER_PREVIOUS_SIZE stores the previous-character index.
const READER_PREVIOUS_SIZE = bits.BIT_COUNT_32_MAXIMUM / bits.BIT_COUNT_8_MAXIMUM

// READER_STREAM_STATE_OFFSET follows the previous-character index.
const READER_STREAM_STATE_OFFSET = READER_PREVIOUS_OFFSET + READER_PREVIOUS_SIZE

// READER_STATE_SIZE stores the cursor, previous character, and lifecycle marker.
const READER_STATE_SIZE = READER_STREAM_STATE_OFFSET + STREAM_STATE_SIZE

// BUILDER_STREAM_STATE_OFFSET is the Builder lifecycle marker.
const BUILDER_STREAM_STATE_OFFSET = 0

// STREAM_LIFECYCLE_MODES names the operations that both stateful text streams supply.
const STREAM_LIFECYCLE_MODES shared_io.Stream_Mode_Set = 1<<shared_io.STREAM_MODE_CLOSE |
	1<<shared_io.STREAM_MODE_FLUSH | 1<<shared_io.STREAM_MODE_DESTROY |
	1<<shared_io.STREAM_MODE_QUERY

// BUILDER_STREAM_MODES declares the shared stream operations that Builder supplies.
const BUILDER_STREAM_MODES shared_io.Stream_Mode_Set = STREAM_LIFECYCLE_MODES |
	1<<shared_io.STREAM_MODE_WRITE

// READER_STREAM_MODES declares the shared stream operations that Reader supplies.
const READER_STREAM_MODES shared_io.Stream_Mode_Set = STREAM_LIFECYCLE_MODES |
	1<<shared_io.STREAM_MODE_READ | 1<<shared_io.STREAM_MODE_READ_AT |
	1<<shared_io.STREAM_MODE_SEEK | 1<<shared_io.STREAM_MODE_SIZE

// Text gives each string boundary a bounded domain identity.
type Text string

// Text_Invariants applies the shared text budget at each named boundary.
func Text_Invariants(value Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Slice gives each stream buffer the same budget as Text.
type Slice []byte

// Slice_Invariants prevents a stream buffer from bypassing the text budget.
func Slice_Invariants(value Slice, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SLICE_SIZE_MINIMUM, SLICE_SIZE_MAXIMUM).
		Ensure()
}

// Texts keeps a split or join collection in one bounded domain.
type Texts []Text

// Texts_Invariants includes the extra item that an all-separator split returns.
func Texts_Invariants(value Texts, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), TEXTS_COUNT_MINIMUM, TEXTS_COUNT_MAXIMUM).
		Ensure()
}

// Field_Texts uses the more narrow count that character separators permit.
type Field_Texts []Text

// Field_Texts_Invariants preserves the alternating-byte field maximum.
func Field_Texts_Invariants(value Field_Texts, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), FIELDS_COUNT_MINIMUM, FIELDS_COUNT_MAXIMUM).
		Ensure()
}

// Replacement_Pair keeps one old-new rule complete by construction.
type Replacement_Pair [REPLACEMENT_PAIR_SIZE]Text

// Replacement_Pair_Invariants fixes the pair storage without sharing Text identities.
func Replacement_Pair_Invariants(value Replacement_Pair, _ invariant.Namespace) {
	invariant.Always(
		len(value) == REPLACEMENT_PAIR_SIZE,
		"A replacement pair contains old and new text.",
	)
}

// Replacement_Pairs gives the rule collection a bounded identity.
type Replacement_Pairs []Replacement_Pair

// Replacement_Pairs_Invariants permits one replacement rule for each input byte.
func Replacement_Pairs_Invariants(value Replacement_Pairs, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value),
			REPLACEMENT_PAIRS_COUNT_MINIMUM,
			REPLACEMENT_PAIRS_COUNT_MAXIMUM,
		).
		Ensure()
}

// Index_Value preserves the absent value beside every valid byte index.
type Index_Value int

// Index_Value_Invariants prevents a search from returning an end boundary.
func Index_Value_Invariants(value Index_Value, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), INDEX_ABSENT, INDEX_MAXIMUM).
		Ensure()
}

// Boundary_Index admits the end boundary that an empty final match returns.
type Boundary_Index int

// Boundary_Index_Invariants includes absence and the final end boundary.
func Boundary_Index_Invariants(value Boundary_Index, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), INDEX_ABSENT, BOUNDARY_INDEX_MAXIMUM).
		Ensure()
}

// Count_Value identifies a non-overlapping match count.
type Count_Value int

// Count_Value_Invariants includes every empty boundary in the largest Text.
func Count_Value_Invariants(value Count_Value, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), COUNT_VALUE_MINIMUM, COUNT_VALUE_MAXIMUM).
		Ensure()
}

// Limit distinguishes all results from a bounded result count.
type Limit int

// Limit_Invariants admits only the standard all-results sentinel and bounded counts.
func Limit_Invariants(value Limit, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), LIMIT_MINIMUM, LIMIT_MAXIMUM).
		Ensure()
}

// Repeat_Count gives repetition a nonnegative bounded domain.
type Repeat_Count int

// Repeat_Count_Invariants prevents an unbounded repeat request.
func Repeat_Count_Invariants(value Repeat_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), REPEAT_COUNT_MINIMUM, REPEAT_COUNT_MAXIMUM).
		Ensure()
}

// Replacement_Count distinguishes all replacements from a bounded count.
type Replacement_Count int

// Replacement_Count_Invariants preserves the one supported negative sentinel.
func Replacement_Count_Invariants(value Replacement_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), REPLACEMENT_COUNT_MINIMUM, REPLACEMENT_COUNT_MAXIMUM,
		).
		Ensure()
}

// Order gives lexical results three stable values.
type Order int

// Order_Invariants rejects an implementation-specific comparison magnitude.
func Order_Invariants(value Order, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Int(int(value), ORDER_BEFORE, ORDER_EQUAL, ORDER_AFTER).
		Ensure()
}

// Boolean gives each query result its own coverage identity.
type Boolean bool

// Boolean_Invariants requires evidence for both query results.
func Boolean_Invariants(value Boolean, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "A Boolean report is true.").
		Ensure()
}

// Character keeps a rune argument distinct from an integer argument.
type Character rune

// Character_Invariants covers the complete rune storage domain.
func Character_Invariants(value Character, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int32(int32(value), CHARACTER_MINIMUM, CHARACTER_MAXIMUM).
		Ensure()
}

// Decoded_Character excludes rune values that UTF-8 cannot produce.
type Decoded_Character rune

// Decoded_Character_Invariants keeps Reader output in the Unicode code-point domain.
func Decoded_Character_Invariants(
	value Decoded_Character, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int32(
			int32(value), DECODED_CHARACTER_MINIMUM, DECODED_CHARACTER_MAXIMUM,
		).
		Ensure()
}

// Character_Size keeps a decoded size separate from a general text boundary.
type Character_Size int

// Character_Size_Invariants includes the end result and all UTF-8 encoding sizes.
func Character_Size_Invariants(value Character_Size, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), CHARACTER_SIZE_MINIMUM, CHARACTER_SIZE_MAXIMUM).
		Ensure()
}

// Byte keeps a byte argument distinct from a character argument.
type Byte byte

// Byte_Invariants covers all byte values.
func Byte_Invariants(value Byte, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), BYTE_MINIMUM, BYTE_MAXIMUM).
		Ensure()
}

// Boundary identifies a byte count or an in-text boundary.
type Boundary int

// Boundary_Invariants includes the boundary after the largest Text.
func Boundary_Invariants(value Boundary, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Growth_Count keeps a Builder reservation inside one text budget.
type Growth_Count int

// Growth_Count_Invariants prevents negative or unbounded reservations.
func Growth_Count_Invariants(value Growth_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), GROWTH_COUNT_MINIMUM, GROWTH_COUNT_MAXIMUM).
		Ensure()
}

// Capacity admits the standard growth factor without increasing the content budget.
type Capacity int

// Capacity_Invariants keeps allocation inside one text budget.
func Capacity_Invariants(value Capacity, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), TEXT_SIZE_MINIMUM, BUILDER_CAPACITY_MAXIMUM).
		Ensure()
}

// Stream_Offset keeps the shared stream ABI outside raw scalar boundaries.
type Stream_Offset int64

// Stream_Offset_Invariants admits every offset so a Stream can report invalid values.
func Stream_Offset_Invariants(value Stream_Offset, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), STREAM_OFFSET_MINIMUM, STREAM_OFFSET_MAXIMUM).
		Ensure()
}

// Stream_Count keeps stream counts inside one bounded text operation.
type Stream_Count int64

// Stream_Count_Invariants includes query mode sets and all text byte counts.
func Stream_Count_Invariants(value Stream_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), STREAM_COUNT_MINIMUM, STREAM_COUNT_MAXIMUM).
		Ensure()
}

// Reader_Position keeps each stream cursor inside its source.
type Reader_Position int64

// Reader_Position_Invariants rejects a cursor outside the largest Reader source.
func Reader_Position_Invariants(value Reader_Position, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(
			int64(value), READER_POSITION_MINIMUM, READER_POSITION_MAXIMUM,
		).
		Ensure()
}

// Builder_State stores stream state without another coverage branch.
type Builder_State [BUILDER_STATE_SIZE]byte

// Builder_State_Invariants fixes the state storage size.
func Builder_State_Invariants(value Builder_State, _ invariant.Namespace) {
	invariant.Always(
		len(value) == BUILDER_STATE_SIZE,
		"A Builder stream state has fixed storage.",
	)
}

// Builder keeps bounded owned text behind a shared stream.
type Builder struct {
	// Content stays visible because the assertion tree must compose its bounded identity.
	Content Slice
	// State keeps lifecycle storage fixed without a Boolean coverage identity.
	State Builder_State
}

// Builder_Invariants composes the content and stream state.
func Builder_Invariants(value Builder, namespace invariant.Namespace) {
	Slice_Invariants(value.Content, namespace)
	Capacity_Invariants(Capacity(cap(value.Content)), namespace)
	Builder_State_Invariants(value.State, namespace)
}

// Reader_State stores cursor state without constructor-only coverage gaps.
type Reader_State [READER_STATE_SIZE]byte

// Reader_State_Invariants fixes the cursor storage size.
func Reader_State_Invariants(value Reader_State, _ invariant.Namespace) {
	invariant.Always(
		len(value) == READER_STATE_SIZE,
		"A Reader stream state has fixed storage.",
	)
}

// Reader keeps a bounded immutable source and explicit cursor state.
type Reader struct {
	// Source stays visible because the assertion tree must compose its bounded identity.
	Source Text
	// State keeps each cursor value private from callers without hiding its storage.
	State Reader_State
}

// Reader_Invariants composes the source and fixed cursor storage.
func Reader_Invariants(value Reader, namespace invariant.Namespace) {
	Text_Invariants(value.Source, namespace)
	Reader_State_Invariants(value.State, namespace)
}

// Replacer keeps an immutable rule copy for each replacement operation.
type Replacer struct {
	// Pairs stays visible because the assertion tree must compose its collection identity.
	Pairs Replacement_Pairs
}

// Replacer_Invariants keeps the copied rule collection in its domain.
func Replacer_Invariants(value Replacer, namespace invariant.Namespace) {
	Replacement_Pairs_Invariants(value.Pairs, namespace)
}

// Builder_To_Stream keeps byte output on the repository stream boundary.
func Builder_To_Stream(builder *Builder) (stream shared_io.Stream) {
	Builder_Invariants(*builder, "builder_to_stream.builder")
	builder.State[BUILDER_STREAM_STATE_OFFSET] = STREAM_STATE_OPEN
	return shared_io.Stream{
		Procedure: func(
			data any,
			mode shared_io.Stream_Mode,
			buffer []byte,
			_ int64,
			_ shared_io.Seek_From,
		) (count int64, err error) {
			stream_count, stream_err := builder_stream_procedure(
				data, mode, Slice(buffer),
			)
			return int64(stream_count), stream_err
		},
		Data: builder,
	}
}

// Builder_Text returns the bounded content through the Text domain.
func Builder_Text(builder *Builder) (content Text) {
	defer func() { Text_Invariants(content, "builder_text.content") }()
	Builder_Invariants(*builder, "builder_text.builder")
	return Text(string(builder.Content))
}

// Builder_Size keeps the byte count in the text boundary domain.
func Builder_Size(builder *Builder) (size Boundary) {
	defer func() { Boundary_Invariants(size, "builder_size.size") }()
	Builder_Invariants(*builder, "builder_size.builder")
	return Boundary(len(builder.Content))
}

// Builder_Capacity exposes allocation only through its bounded capacity domain.
func Builder_Capacity(builder *Builder) (capacity Capacity) {
	defer func() { Capacity_Invariants(capacity, "builder_capacity.capacity") }()
	Builder_Invariants(*builder, "builder_capacity.builder")
	return Capacity(cap(builder.Content))
}

// Builder_Grow rejects a reservation that cannot fit the final text budget.
func Builder_Grow(builder *Builder, count Growth_Count) {
	Builder_Invariants(*builder, "builder_grow.builder")
	Growth_Count_Invariants(count, "builder_grow.count")
	if int(count) > TEXT_SIZE_MAXIMUM-len(builder.Content) {
		panic(Error_Too_Large)
	}
	required := len(builder.Content) + int(count)
	if cap(builder.Content) < required {
		capacity := 2*cap(builder.Content) + int(count)
		if capacity < required {
			capacity = required
		}
		if capacity > BUILDER_CAPACITY_MAXIMUM {
			capacity = BUILDER_CAPACITY_MAXIMUM
		}
		grown := make(Slice, len(builder.Content), capacity)
		copy(grown, builder.Content)
		builder.Content = grown
	}
	Builder_Invariants(*builder, "builder_grow.result")
}

// Builder_Reset releases the copy identity and reopens the stream state.
func Builder_Reset(builder *Builder) {
	Builder_Invariants(*builder, "builder_reset.builder")
	builder.Content = nil
	builder.State[BUILDER_STREAM_STATE_OFFSET] = STREAM_STATE_OPEN
}

func builder_stream_procedure(
	data any,
	mode shared_io.Stream_Mode,
	buffer Slice,
) (count Stream_Count, err error) {
	defer func() { Stream_Count_Invariants(count, "builder_stream.count") }()
	Slice_Invariants(buffer, "builder_stream.buffer")
	builder, held := data.(*Builder)
	invariant.Always(held, "A Builder stream receives Builder state.")
	Builder_Invariants(*builder, "builder_stream.builder")
	if mode == shared_io.STREAM_MODE_QUERY {
		return Stream_Count(BUILDER_STREAM_MODES), nil
	}
	if mode == shared_io.STREAM_MODE_CLOSE {
		builder.State[BUILDER_STREAM_STATE_OFFSET] = STREAM_STATE_CLOSED
		return 0, nil
	}
	if mode == shared_io.STREAM_MODE_DESTROY {
		builder.State[BUILDER_STREAM_STATE_OFFSET] = STREAM_STATE_CLOSED
		return 0, nil
	}
	if builder.State[BUILDER_STREAM_STATE_OFFSET] == STREAM_STATE_CLOSED {
		return 0, shared_io.Stream_Empty
	}
	if mode == shared_io.STREAM_MODE_FLUSH {
		return 0, nil
	}
	if mode != shared_io.STREAM_MODE_WRITE {
		return 0, shared_io.Stream_Empty
	}
	if len(buffer) > TEXT_SIZE_MAXIMUM-len(builder.Content) {
		panic(Error_Too_Large)
	}
	Builder_Grow(builder, Growth_Count(len(buffer)))
	builder.Content = append(builder.Content, buffer...)
	Builder_Invariants(*builder, "builder_stream.result")
	return Stream_Count(len(buffer)), nil
}

// New_Reader fixes the source before a stream can read it.
func New_Reader(source Text) (reader *Reader) {
	defer func() { Reader_Invariants(*reader, "new_reader.reader") }()
	Text_Invariants(source, "new_reader.source")
	reader = &Reader{Source: source}
	reader_set_previous(reader, INDEX_ABSENT)
	return reader
}

// Reader_To_Stream keeps sequential reads on the repository stream boundary.
func Reader_To_Stream(reader *Reader) (stream shared_io.Stream) {
	Reader_Invariants(*reader, "reader_to_stream.reader")
	reader.State[READER_STREAM_STATE_OFFSET] = STREAM_STATE_OPEN
	return shared_io.Stream{
		Procedure: func(
			data any,
			mode shared_io.Stream_Mode,
			buffer []byte,
			offset int64,
			whence shared_io.Seek_From,
		) (count int64, err error) {
			stream_count, stream_err := reader_stream_procedure(
				data, mode, Slice(buffer), Stream_Offset(offset), whence,
			)
			return int64(stream_count), stream_err
		},
		Data: reader,
	}
}

// Reader_Unread_Size gives the unread byte count a bounded identity.
func Reader_Unread_Size(reader *Reader) (size Boundary) {
	defer func() { Boundary_Invariants(size, "reader_unread_size.size") }()
	Reader_Invariants(*reader, "reader_unread_size.reader")
	return Boundary(len(reader.Source) - int(reader_position(reader)))
}

// Reader_Read_Byte preserves the byte domain that a generic stream cannot return.
func Reader_Read_Byte(reader *Reader) (value Byte, err error) {
	defer func() { Byte_Invariants(value, "reader_read_byte.value") }()
	Reader_Invariants(*reader, "reader_read_byte.reader")
	if reader.State[READER_STREAM_STATE_OFFSET] == STREAM_STATE_CLOSED {
		return 0, shared_io.Stream_Empty
	}
	reader_set_previous(reader, INDEX_ABSENT)
	position := reader_position(reader)
	if position == Reader_Position(len(reader.Source)) {
		return 0, shared_io.Stream_EOF
	}
	value = Byte(reader.Source[position])
	reader_set_position(reader, position+1)
	return value, nil
}

// Reader_Unread_Byte keeps the standard one-byte reversal outside the Stream mode set.
func Reader_Unread_Byte(reader *Reader) (err error) {
	Reader_Invariants(*reader, "reader_unread_byte.reader")
	if reader.State[READER_STREAM_STATE_OFFSET] == STREAM_STATE_CLOSED {
		return shared_io.Stream_Empty
	}
	position := reader_position(reader)
	if position == 0 {
		return shared_io.Stream_Invalid_Unread
	}
	reader_set_position(reader, position-1)
	reader_set_previous(reader, INDEX_ABSENT)
	return nil
}

// Reader_Read_Character preserves one decoded character for the unread operation.
func Reader_Read_Character(
	reader *Reader,
) (character Decoded_Character, size Character_Size, err error) {
	defer func() {
		Decoded_Character_Invariants(character, "reader_read_character.character")
		Character_Size_Invariants(size, "reader_read_character.size")
	}()
	Reader_Invariants(*reader, "reader_read_character.reader")
	if reader.State[READER_STREAM_STATE_OFFSET] == STREAM_STATE_CLOSED {
		return 0, 0, shared_io.Stream_Empty
	}
	position := reader_position(reader)
	if position == Reader_Position(len(reader.Source)) {
		reader_set_previous(reader, INDEX_ABSENT)
		return 0, 0, shared_io.Stream_EOF
	}
	reader_set_previous(reader, Index_Value(position))
	decoded, decoded_size := utf8.Decode_Character_Text(
		utf8.Text(reader.Source[position:]),
	)
	reader_set_position(reader, position+Reader_Position(decoded_size))
	return Decoded_Character(decoded), Character_Size(decoded_size), nil
}

// Reader_Unread_Character rejects a reversal after any operation except a character read.
func Reader_Unread_Character(reader *Reader) (err error) {
	Reader_Invariants(*reader, "reader_unread_character.reader")
	if reader.State[READER_STREAM_STATE_OFFSET] == STREAM_STATE_CLOSED {
		return shared_io.Stream_Empty
	}
	previous := reader_previous(reader)
	if previous == INDEX_ABSENT {
		return shared_io.Stream_Invalid_Unread
	}
	reader_set_position(reader, Reader_Position(previous))
	reader_set_previous(reader, INDEX_ABSENT)
	return nil
}

// Reader_Write_To keeps the destination behind a shared stream.
func Reader_Write_To(
	reader *Reader, destination shared_io.Stream,
) (count Stream_Count, err error) {
	defer func() { Stream_Count_Invariants(count, "reader_write_to.count") }()
	Reader_Invariants(*reader, "reader_write_to.reader")
	if reader.State[READER_STREAM_STATE_OFFSET] == STREAM_STATE_CLOSED {
		return 0, shared_io.Stream_Empty
	}
	reader_set_previous(reader, INDEX_ABSENT)
	position := reader_position(reader)
	if position == Reader_Position(len(reader.Source)) {
		return 0, nil
	}
	content := []byte(reader.Source[position:])
	written, write_err := shared_io.Write(destination, content)
	reader_set_position(reader, position+Reader_Position(written))
	return Stream_Count(written), write_err
}

// Reader_Reset replaces the source and reopens the Reader stream state.
func Reader_Reset(reader *Reader, source Text) {
	Reader_Invariants(*reader, "reader_reset.reader")
	Text_Invariants(source, "reader_reset.source")
	reader.Source = source
	reader_set_position(reader, 0)
	reader_set_previous(reader, INDEX_ABSENT)
	reader.State[READER_STREAM_STATE_OFFSET] = STREAM_STATE_OPEN
}

func reader_position(reader *Reader) (position Reader_Position) {
	defer func() { Reader_Position_Invariants(position, "reader_position.position") }()
	Reader_Invariants(*reader, "reader_position.reader")
	position_bits := standard_binary.LittleEndian.Uint64(
		reader.State[READER_POSITION_OFFSET:READER_PREVIOUS_OFFSET],
	)
	return Reader_Position(position_bits)
}

func reader_set_position(reader *Reader, position Reader_Position) {
	Reader_Invariants(*reader, "reader_set_position.reader")
	Reader_Position_Invariants(position, "reader_set_position.position")
	invariant.Always(
		position <= Reader_Position(len(reader.Source)),
		"A Reader position does not pass its source.",
	)
	standard_binary.LittleEndian.PutUint64(
		reader.State[READER_POSITION_OFFSET:READER_PREVIOUS_OFFSET],
		uint64(position),
	)
}

func reader_previous(reader *Reader) (previous Index_Value) {
	defer func() { Index_Value_Invariants(previous, "reader_previous.previous") }()
	Reader_Invariants(*reader, "reader_previous.reader")
	previous_bits := standard_binary.LittleEndian.Uint32(
		reader.State[READER_PREVIOUS_OFFSET:READER_STREAM_STATE_OFFSET],
	)
	return Index_Value(int32(previous_bits))
}

func reader_set_previous(reader *Reader, previous Index_Value) {
	Reader_Invariants(*reader, "reader_set_previous.reader")
	Index_Value_Invariants(previous, "reader_set_previous.previous")
	standard_binary.LittleEndian.PutUint32(
		reader.State[READER_PREVIOUS_OFFSET:READER_STREAM_STATE_OFFSET],
		uint32(int32(previous)),
	)
}

func reader_stream_procedure(
	data any,
	mode shared_io.Stream_Mode,
	buffer Slice,
	offset Stream_Offset,
	whence shared_io.Seek_From,
) (count Stream_Count, err error) {
	defer func() { Stream_Count_Invariants(count, "reader_stream.count") }()
	Slice_Invariants(buffer, "reader_stream.buffer")
	Stream_Offset_Invariants(offset, "reader_stream.offset")
	reader, held := data.(*Reader)
	invariant.Always(held, "A Reader stream receives Reader state.")
	Reader_Invariants(*reader, "reader_stream.reader")
	if mode == shared_io.STREAM_MODE_QUERY {
		return Stream_Count(READER_STREAM_MODES), nil
	}
	if mode == shared_io.STREAM_MODE_CLOSE {
		reader.State[READER_STREAM_STATE_OFFSET] = STREAM_STATE_CLOSED
		return 0, nil
	}
	if mode == shared_io.STREAM_MODE_DESTROY {
		reader.State[READER_STREAM_STATE_OFFSET] = STREAM_STATE_CLOSED
		return 0, nil
	}
	if reader.State[READER_STREAM_STATE_OFFSET] == STREAM_STATE_CLOSED {
		return 0, shared_io.Stream_Empty
	}
	if mode == shared_io.STREAM_MODE_FLUSH {
		return 0, nil
	}
	if mode == shared_io.STREAM_MODE_SIZE {
		return Stream_Count(len(reader.Source)), nil
	}
	if mode == shared_io.STREAM_MODE_SEEK {
		return reader_stream_seek(reader, offset, whence)
	}
	if mode == shared_io.STREAM_MODE_READ_AT {
		return reader_stream_read_at(reader, buffer, offset)
	}
	if mode == shared_io.STREAM_MODE_READ {
		reader_set_previous(reader, INDEX_ABSENT)
		position := reader_position(reader)
		moved, read_err := reader_stream_read_at(
			reader, buffer, Stream_Offset(position),
		)
		reader_set_position(reader, position+Reader_Position(moved))
		return moved, read_err
	}
	return 0, shared_io.Stream_Empty
}

func reader_stream_read_at(
	reader *Reader, buffer Slice, offset Stream_Offset,
) (count Stream_Count, err error) {
	defer func() { Stream_Count_Invariants(count, "reader_stream_read_at.count") }()
	Reader_Invariants(*reader, "reader_stream_read_at.reader")
	Slice_Invariants(buffer, "reader_stream_read_at.buffer")
	Stream_Offset_Invariants(offset, "reader_stream_read_at.offset")
	if offset < 0 {
		return 0, shared_io.Stream_Invalid_Offset
	}
	if offset > Stream_Offset(len(reader.Source)) {
		return 0, shared_io.Stream_Invalid_Offset
	}
	if offset == Stream_Offset(len(reader.Source)) {
		return 0, shared_io.Stream_EOF
	}
	return Stream_Count(copy(buffer, reader.Source[offset:])), nil
}

func reader_stream_seek(
	reader *Reader, offset Stream_Offset, whence shared_io.Seek_From,
) (position Stream_Count, err error) {
	defer func() { Stream_Count_Invariants(position, "reader_stream_seek.position") }()
	Reader_Invariants(*reader, "reader_stream_seek.reader")
	Stream_Offset_Invariants(offset, "reader_stream_seek.offset")
	base := Stream_Offset(0)
	if whence == shared_io.SEEK_FROM_CURRENT {
		base = Stream_Offset(reader_position(reader))
	} else if whence == shared_io.SEEK_FROM_END {
		base = Stream_Offset(len(reader.Source))
	} else if whence != shared_io.SEEK_FROM_START {
		return 0, shared_io.Stream_Invalid_Whence
	}
	if offset > Stream_Offset(STREAM_OFFSET_MAXIMUM)-base {
		return 0, shared_io.Stream_Invalid_Offset
	}
	target := offset + base
	if target < 0 {
		return 0, shared_io.Stream_Invalid_Offset
	}
	if target > Stream_Offset(len(reader.Source)) {
		return 0, shared_io.Stream_Invalid_Offset
	}
	position = Stream_Count(target)
	reader_set_position(reader, Reader_Position(position))
	reader_set_previous(reader, INDEX_ABSENT)
	return position, nil
}

// New_Replacer copies its rules so later caller changes cannot change replacement order.
func New_Replacer(pairs Replacement_Pairs) (replacer *Replacer) {
	defer func() { Replacer_Invariants(*replacer, "new_replacer.replacer") }()
	Replacement_Pairs_Invariants(pairs, "new_replacer.pairs")
	copied := append(Replacement_Pairs(nil), pairs...)
	for _, pair := range copied {
		Replacement_Pair_Invariants(pair, "new_replacer.pair")
		Text_Invariants(pair[0], "new_replacer.match")
		Text_Invariants(pair[1], "new_replacer.substitute")
	}
	return &Replacer{Pairs: copied}
}

// Replacer_Replace applies the immutable rules before it admits the owned result.
func Replacer_Replace(replacer *Replacer, source Text) (replaced Text) {
	defer func() { Text_Invariants(replaced, "replacer_replace.replaced") }()
	Replacer_Invariants(*replacer, "replacer_replace.replacer")
	Text_Invariants(source, "replacer_replace.source")
	for _, pair := range replacer.Pairs {
		Replacement_Pair_Invariants(pair, "replacer_replace.pair")
		Text_Invariants(pair[0], "replacer_replace.match")
		Text_Invariants(pair[1], "replacer_replace.substitute")
	}
	result := make([]byte, 0, len(source))
	position := 0
	previous_match_empty := false
	for position <= len(source) {
		matched := false
		for _, pair := range replacer.Pairs {
			match := pair[0]
			if len(match) == 0 {
				if previous_match_empty {
					continue
				}
			} else if !Has_Prefix(source[position:], match) {
				continue
			}
			if len(pair[1]) > TEXT_SIZE_MAXIMUM-len(result) {
				panic(Error_Too_Large)
			}
			result = append(result, pair[1]...)
			position += len(match)
			previous_match_empty = len(match) == 0
			matched = true
			break
		}
		if matched {
			continue
		}
		if position == len(source) {
			break
		}
		_, size := utf8.Decode_Character_Text(utf8.Text(source[position:]))
		if int(size) > TEXT_SIZE_MAXIMUM-len(result) {
			panic(Error_Too_Large)
		}
		result = append(result, source[position:position+int(size)]...)
		position += int(size)
		previous_match_empty = false
	}
	return Text(string(result))
}

// Replacer_Write_Text keeps replacement output on the shared stream boundary.
func Replacer_Write_Text(
	replacer *Replacer, destination shared_io.Stream, source Text,
) (count Stream_Count, err error) {
	defer func() { Stream_Count_Invariants(count, "replacer_write_text.count") }()
	Replacer_Invariants(*replacer, "replacer_write_text.replacer")
	Text_Invariants(source, "replacer_write_text.source")
	replaced := Replacer_Replace(replacer, source)
	written, write_err := shared_io.Write_String(destination, string(replaced))
	return Stream_Count(written), write_err
}

// Compare normalizes lexical order to three repository values.
func Compare(left Text, right Text) (order Order) {
	defer func() { Order_Invariants(order, "compare.order") }()
	Text_Invariants(left, "compare.left")
	Text_Invariants(right, "compare.right")
	if left < right {
		return ORDER_BEFORE
	}
	if left > right {
		return ORDER_AFTER
	}
	return ORDER_EQUAL
}

// Clone forces a distinct text allocation when the source is not empty.
func Clone(source Text) (clone Text) {
	defer func() { Text_Invariants(clone, "clone.clone") }()
	Text_Invariants(source, "clone.source")
	if len(source) == 0 {
		return ""
	}
	content := append([]byte(nil), source...)
	return Text(string(content))
}

// Contains keeps a substring query in the Boolean coverage domain.
func Contains(source Text, separator Text) (contained Boolean) {
	defer func() { Boolean_Invariants(contained, "contains.contained") }()
	Text_Invariants(source, "contains.source")
	Text_Invariants(separator, "contains.separator")
	return Boolean(Index(source, separator) >= 0)
}

// Contains_Any treats characters as a set instead of a substring.
func Contains_Any(source Text, characters Text) (contained Boolean) {
	defer func() { Boolean_Invariants(contained, "contains_any.contained") }()
	Text_Invariants(source, "contains_any.source")
	Text_Invariants(characters, "contains_any.characters")
	return Boolean(Index_Any(source, characters) >= 0)
}

// Contains_Rune keeps a character query distinct from a byte query.
func Contains_Rune(source Text, character Character) (contained Boolean) {
	defer func() { Boolean_Invariants(contained, "contains_rune.contained") }()
	Text_Invariants(source, "contains_rune.source")
	Character_Invariants(character, "contains_rune.character")
	return Boolean(Index_Rune(source, character) >= 0)
}

// Contains_Function keeps injected character selection explicit.
func Contains_Function(
	source Text, predicate func(rune) (matches bool),
) (contained Boolean) {
	defer func() { Boolean_Invariants(contained, "contains_function.contained") }()
	Text_Invariants(source, "contains_function.source")
	return Boolean(Index_Function(source, predicate) >= 0)
}

// Count includes empty matches at each UTF-8 character boundary.
func Count(source Text, separator Text) (count Count_Value) {
	defer func() { Count_Value_Invariants(count, "count.count") }()
	Text_Invariants(source, "count.source")
	Text_Invariants(separator, "count.separator")
	if len(separator) == 0 {
		return Count_Value(utf8.Character_Count_Text(utf8.Text(source)) + 1)
	}
	tail := source
	position := Index(tail, separator)
	for position != INDEX_ABSENT {
		count++
		tail = tail[int(position)+len(separator):]
		position = Index(tail, separator)
	}
	return count
}

// Index returns absence through a value that cannot be confused with a boundary.
func Index(source Text, separator Text) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "index.index") }()
	Text_Invariants(source, "index.source")
	Text_Invariants(separator, "index.separator")
	if len(separator) == 0 {
		return 0
	}
	if len(separator) > len(source) {
		return INDEX_ABSENT
	}
	final_start := len(source) - len(separator)
	for start := 0; start <= final_start; start++ {
		if source[start:start+len(separator)] == separator {
			return Index_Value(start)
		}
	}
	return INDEX_ABSENT
}

// Last_Index admits the final boundary for an empty separator.
func Last_Index(source Text, separator Text) (index Boundary_Index) {
	defer func() { Boundary_Index_Invariants(index, "last_index.index") }()
	Text_Invariants(source, "last_index.source")
	Text_Invariants(separator, "last_index.separator")
	if len(separator) == 0 {
		return Boundary_Index(len(source))
	}
	if len(separator) > len(source) {
		return INDEX_ABSENT
	}
	for start := len(source) - len(separator); start >= 0; start-- {
		if source[start:start+len(separator)] == separator {
			return Boundary_Index(start)
		}
	}
	return INDEX_ABSENT
}

// Index_Byte keeps byte search separate from Unicode character search.
func Index_Byte(source Text, value Byte) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "index_byte.index") }()
	Text_Invariants(source, "index_byte.source")
	Byte_Invariants(value, "index_byte.value")
	repeated := uint64(value) * BYTE_LOW_BITS
	position, rest := 0, source
	for len(rest) >= WORD_BYTE_COUNT {
		// Explicit indexing gives the compiler one native load without an unsafe
		// conversion, while the public text boundary proves that all eight bytes exist.
		word := uint64(rest[0]) |
			uint64(rest[1])<<bits.BIT_COUNT_8_MAXIMUM |
			uint64(rest[2])<<(2*bits.BIT_COUNT_8_MAXIMUM) |
			uint64(rest[3])<<(3*bits.BIT_COUNT_8_MAXIMUM) |
			uint64(rest[4])<<(4*bits.BIT_COUNT_8_MAXIMUM) |
			uint64(rest[5])<<(5*bits.BIT_COUNT_8_MAXIMUM) |
			uint64(rest[6])<<(6*bits.BIT_COUNT_8_MAXIMUM) |
			uint64(rest[7])<<(7*bits.BIT_COUNT_8_MAXIMUM)
		matches := word ^ repeated
		if (matches-BYTE_LOW_BITS)&^matches&BYTE_HIGH_BITS != 0 {
			for byte_index := 0; byte_index < WORD_BYTE_COUNT; byte_index++ {
				if rest[byte_index] == byte(value) {
					return Index_Value(position + byte_index)
				}
			}
		}
		rest = rest[WORD_BYTE_COUNT:]
		position += WORD_BYTE_COUNT
	}
	for byte_index := range len(rest) {
		if rest[byte_index] == byte(value) {
			return Index_Value(position + byte_index)
		}
	}
	return INDEX_ABSENT
}

// Index_Byte_Or_Non_ASCII returns the first byte that is in values or outside ASCII.
func Index_Byte_Or_Non_ASCII(source Text, values Text) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "index_byte_or_non_ascii.index") }()
	Text_Invariants(source, "index_byte_or_non_ascii.source")
	Text_Invariants(values, "index_byte_or_non_ascii.values")
	if len(values) == 3 {
		if len(source) > 0 {
			safe_byte := source[0]
			safe_unselected := safe_byte < byte(utf8.CHARACTER_SELF)
			for value_index := range len(values) {
				if safe_byte == values[value_index] {
					safe_unselected = false
				}
			}
			if safe_unselected {
				repeated := uint64(safe_byte) * BYTE_LOW_BITS
				rest := source
				all_repeated := true
				for len(rest) >= WORD_BYTE_COUNT {
					word := uint64(rest[0]) |
						uint64(rest[1])<<bits.BIT_COUNT_8_MAXIMUM |
						uint64(rest[2])<<(2*bits.BIT_COUNT_8_MAXIMUM) |
						uint64(rest[3])<<(3*bits.BIT_COUNT_8_MAXIMUM) |
						uint64(rest[4])<<(4*bits.BIT_COUNT_8_MAXIMUM) |
						uint64(rest[5])<<(5*bits.BIT_COUNT_8_MAXIMUM) |
						uint64(rest[6])<<(6*bits.BIT_COUNT_8_MAXIMUM) |
						uint64(rest[7])<<(7*bits.BIT_COUNT_8_MAXIMUM)
					if word != repeated {
						all_repeated = false
						break
					}
					rest = rest[WORD_BYTE_COUNT:]
				}
				if all_repeated {
					for byte_index := range len(rest) {
						if rest[byte_index] != safe_byte {
							all_repeated = false
							break
						}
					}
				}
				if all_repeated {
					return INDEX_ABSENT
				}
			}
		}
	}
	return index_byte_or_non_ascii_general(source, values)
}

// Scans text after the uniform fast proof cannot answer the byte question.
func index_byte_or_non_ascii_general(source Text, values Text) (index Index_Value) {
	defer func() {
		Index_Value_Invariants(index, "index_byte_or_non_ascii_general.index")
	}()
	Text_Invariants(source, "index_byte_or_non_ascii_general.source")
	Text_Invariants(values, "index_byte_or_non_ascii_general.values")
	position, rest := 0, source
	first_repeated, second_repeated, third_repeated := uint64(0), uint64(0), uint64(0)
	if len(values) == 3 {
		first_repeated = uint64(values[0]) * BYTE_LOW_BITS
		second_repeated = uint64(values[1]) * BYTE_LOW_BITS
		third_repeated = uint64(values[2]) * BYTE_LOW_BITS
	}
	for len(rest) >= WORD_BYTE_COUNT {
		word := uint64(rest[0]) | uint64(rest[1])<<bits.BIT_COUNT_8_MAXIMUM |
			uint64(rest[2])<<(2*bits.BIT_COUNT_8_MAXIMUM) |
			uint64(rest[3])<<(3*bits.BIT_COUNT_8_MAXIMUM) |
			uint64(rest[4])<<(4*bits.BIT_COUNT_8_MAXIMUM) |
			uint64(rest[5])<<(5*bits.BIT_COUNT_8_MAXIMUM) |
			uint64(rest[6])<<(6*bits.BIT_COUNT_8_MAXIMUM) |
			uint64(rest[7])<<(7*bits.BIT_COUNT_8_MAXIMUM)
		matches := word & BYTE_HIGH_BITS
		if len(values) == 3 {
			first_difference := word ^ first_repeated
			second_difference := word ^ second_repeated
			third_difference := word ^ third_repeated
			matches |= (first_difference - BYTE_LOW_BITS) &^
				first_difference & BYTE_HIGH_BITS
			matches |= (second_difference - BYTE_LOW_BITS) &^
				second_difference & BYTE_HIGH_BITS
			matches |= (third_difference - BYTE_LOW_BITS) &^
				third_difference & BYTE_HIGH_BITS
		} else {
			for value_index := range len(values) {
				repeated := uint64(values[value_index]) * BYTE_LOW_BITS
				difference := word ^ repeated
				matches |= (difference - BYTE_LOW_BITS) &^
					difference & BYTE_HIGH_BITS
			}
		}
		if matches != 0 {
			for byte_index := range WORD_BYTE_COUNT {
				character := rest[byte_index]
				if character >= byte(utf8.CHARACTER_SELF) {
					return Index_Value(position + byte_index)
				}
				for value_index := range len(values) {
					if character == values[value_index] {
						return Index_Value(position + byte_index)
					}
				}
			}
		}
		rest = rest[WORD_BYTE_COUNT:]
		position += WORD_BYTE_COUNT
	}
	for byte_index := range len(rest) {
		character := rest[byte_index]
		if character >= byte(utf8.CHARACTER_SELF) {
			return Index_Value(position + byte_index)
		}
		for value_index := range len(values) {
			if character == values[value_index] {
				return Index_Value(position + byte_index)
			}
		}
	}
	return INDEX_ABSENT
}

// Last_Index_Byte keeps reverse byte search in the byte index domain.
func Last_Index_Byte(source Text, value Byte) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "last_index_byte.index") }()
	Text_Invariants(source, "last_index_byte.source")
	Byte_Invariants(value, "last_index_byte.value")
	for position := len(source) - 1; position >= 0; position-- {
		if source[position] == byte(value) {
			return Index_Value(position)
		}
	}
	return INDEX_ABSENT
}

// Index_Rune reports the byte index of a decoded character.
func Index_Rune(source Text, character Character) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "index_rune.index") }()
	Text_Invariants(source, "index_rune.source")
	Character_Invariants(character, "index_rune.character")
	if !utf8.Valid_Character(utf8.Character(character)) {
		return INDEX_ABSENT
	}
	for position, decoded := range string(source) {
		if decoded == rune(character) {
			return Index_Value(position)
		}
	}
	return INDEX_ABSENT
}

// Index_Any searches a character set and reports a byte index.
func Index_Any(source Text, characters Text) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "index_any.index") }()
	Text_Invariants(source, "index_any.source")
	Text_Invariants(characters, "index_any.characters")
	if len(characters) == 0 {
		return INDEX_ABSENT
	}
	for position, character := range string(source) {
		if Index_Rune(characters, Character(character)) >= 0 {
			return Index_Value(position)
		}
	}
	return INDEX_ABSENT
}

// Last_Index_Any searches a character set from the final byte.
func Last_Index_Any(source Text, characters Text) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "last_index_any.index") }()
	Text_Invariants(source, "last_index_any.source")
	Text_Invariants(characters, "last_index_any.characters")
	if len(characters) == 0 {
		return INDEX_ABSENT
	}
	for position_count := len(source); position_count > 0; {
		character, size := utf8.Decode_Final_Character_Text(
			utf8.Text(source[:position_count]),
		)
		position_count -= int(size)
		if Index_Rune(characters, Character(character)) >= 0 {
			return Index_Value(position_count)
		}
	}
	return INDEX_ABSENT
}

// Index_Function reports the first byte index that an injected predicate selects.
func Index_Function(
	source Text, predicate func(rune) (matches bool),
) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "index_function.index") }()
	Text_Invariants(source, "index_function.source")
	for position, character := range string(source) {
		if predicate(character) {
			return Index_Value(position)
		}
	}
	return INDEX_ABSENT
}

// Last_Index_Function reports the final byte index that a predicate selects.
func Last_Index_Function(
	source Text, predicate func(rune) (matches bool),
) (index Index_Value) {
	defer func() { Index_Value_Invariants(index, "last_index_function.index") }()
	Text_Invariants(source, "last_index_function.source")
	for position_count := len(source); position_count > 0; {
		character, size := utf8.Decode_Final_Character_Text(
			utf8.Text(source[:position_count]),
		)
		position_count -= int(size)
		if predicate(rune(character)) {
			return Index_Value(position_count)
		}
	}
	return INDEX_ABSENT
}

// Split_N keeps the standard limit sentinel inside a declared domain.
func Split_N(source Text, separator Text, limit Limit) (parts Texts) {
	defer func() { Texts_Invariants(parts, "split_n.parts") }()
	Text_Invariants(source, "split_n.source")
	Text_Invariants(separator, "split_n.separator")
	Limit_Invariants(limit, "split_n.limit")
	return split_text(source, separator, 0, limit)
}

// Split_After_N keeps each selected separator in the preceding result.
func Split_After_N(source Text, separator Text, limit Limit) (parts Texts) {
	defer func() { Texts_Invariants(parts, "split_after_n.parts") }()
	Text_Invariants(source, "split_after_n.source")
	Text_Invariants(separator, "split_after_n.separator")
	Limit_Invariants(limit, "split_after_n.limit")
	return split_text(source, separator, Boundary(len(separator)), limit)
}

// Split permits the extra empty result that every-byte separators produce.
func Split(source Text, separator Text) (parts Texts) {
	defer func() { Texts_Invariants(parts, "split.parts") }()
	Text_Invariants(source, "split.source")
	Text_Invariants(separator, "split.separator")
	return split_text(source, separator, 0, LIMIT_MINIMUM)
}

// Split_After preserves each non-overlapping separator in its preceding part.
func Split_After(source Text, separator Text) (parts Texts) {
	defer func() { Texts_Invariants(parts, "split_after.parts") }()
	Text_Invariants(source, "split_after.source")
	Text_Invariants(separator, "split_after.separator")
	return split_text(
		source, separator, Boundary(len(separator)), LIMIT_MINIMUM,
	)
}

// Fields uses the more narrow field collection domain.
func Fields(source Text) (fields Field_Texts) {
	defer func() { Field_Texts_Invariants(fields, "fields.fields") }()
	Text_Invariants(source, "fields.source")
	space := func(character rune) (yes bool) {
		return bool(ucd.Is_Space(ucd.Character(character)))
	}
	return fields_text(source, space)
}

// Fields_Function keeps injected separator rules inside the field count bound.
func Fields_Function(
	source Text, predicate func(rune) (matches bool),
) (fields Field_Texts) {
	defer func() { Field_Texts_Invariants(fields, "fields_function.fields") }()
	Text_Invariants(source, "fields_function.source")
	return fields_text(source, predicate)
}

// Join calculates the result budget before the standard algorithm allocates it.
func Join(parts Texts, separator Text) (joined Text) {
	defer func() { Text_Invariants(joined, "join.joined") }()
	Texts_Invariants(parts, "join.parts")
	Text_Invariants(separator, "join.separator")
	joined_size := 0
	for _, part := range parts {
		Text_Invariants(part, "join.part")
		if len(part) > TEXT_SIZE_MAXIMUM-joined_size {
			panic(Error_Too_Large)
		}
		joined_size += len(part)
	}
	if len(parts) > 1 {
		if len(separator) > 0 {
			separator_count := len(parts) - 1
			if separator_count > (TEXT_SIZE_MAXIMUM-joined_size)/len(separator) {
				panic(Error_Too_Large)
			}
			joined_size += separator_count * len(separator)
		}
	}
	content := make([]byte, 0, joined_size)
	for index, part := range parts {
		if index > 0 {
			content = append(content, separator...)
		}
		content = append(content, part...)
	}
	return Text(string(content))
}

// Has_Prefix keeps the prefix fact in the Boolean coverage domain.
func Has_Prefix(source Text, prefix Text) (present Boolean) {
	defer func() { Boolean_Invariants(present, "has_prefix.present") }()
	Text_Invariants(source, "has_prefix.source")
	Text_Invariants(prefix, "has_prefix.prefix")
	if len(prefix) > len(source) {
		return false
	}
	return Boolean(source[:len(prefix)] == prefix)
}

// Has_Suffix keeps the suffix fact in the Boolean coverage domain.
func Has_Suffix(source Text, suffix Text) (present Boolean) {
	defer func() { Boolean_Invariants(present, "has_suffix.present") }()
	Text_Invariants(source, "has_suffix.source")
	Text_Invariants(suffix, "has_suffix.suffix")
	if len(suffix) > len(source) {
		return false
	}
	return Boolean(source[len(source)-len(suffix):] == suffix)
}

// Map admits the result only after every injected mapping ran.
func Map(mapping func(rune) (mapped rune), source Text) (mapped Text) {
	defer func() { Text_Invariants(mapped, "map.mapped") }()
	Text_Invariants(source, "map.source")
	content := make([]byte, 0, len(source))
	for _, character := range string(source) {
		mapped_character := mapping(character)
		if mapped_character < 0 {
			continue
		}
		content = []byte(utf8.Append_Character(
			utf8.Bytes(content), utf8.Character(mapped_character),
		))
		if len(content) > TEXT_SIZE_MAXIMUM {
			panic(Error_Too_Large)
		}
	}
	return Text(string(content))
}

// Repeat checks multiplication before the standard algorithm allocates the result.
func Repeat(source Text, count Repeat_Count) (repeated Text) {
	defer func() { Text_Invariants(repeated, "repeat.repeated") }()
	Text_Invariants(source, "repeat.source")
	Repeat_Count_Invariants(count, "repeat.count")
	if len(source) > 0 {
		if int(count) > TEXT_SIZE_MAXIMUM/len(source) {
			panic(Error_Too_Large)
		}
	}
	repeated_size := len(source) * int(count)
	content := make([]byte, repeated_size)
	copied := copy(content, source)
	for copied < repeated_size {
		copied += copy(content[copied:], content[:copied])
	}
	return Text(string(content))
}

// To_Upper rejects a Unicode expansion that exceeds the text budget.
func To_Upper(source Text) (upper Text) {
	defer func() { Text_Invariants(upper, "to_upper.upper") }()
	Text_Invariants(source, "to_upper.source")
	mapping := func(character rune) (mapped rune) {
		return rune(ucd.To_Upper(ucd.Character(character)))
	}
	return Map(mapping, source)
}

// To_Lower rejects a Unicode expansion that exceeds the text budget.
func To_Lower(source Text) (lower Text) {
	defer func() { Text_Invariants(lower, "to_lower.lower") }()
	Text_Invariants(source, "to_lower.source")
	mapping := func(character rune) (mapped rune) {
		return rune(ucd.To_Lower(ucd.Character(character)))
	}
	return Map(mapping, source)
}

// To_Title rejects a Unicode expansion that exceeds the text budget.
func To_Title(source Text) (title Text) {
	defer func() { Text_Invariants(title, "to_title.title") }()
	Text_Invariants(source, "to_title.source")
	mapping := func(character rune) (mapped rune) {
		return rune(ucd.To_Title(ucd.Character(character)))
	}
	return Map(mapping, source)
}

// To_Upper_Special keeps an injected special case inside the text budget.
func To_Upper_Special(special ucd.Special_Case, source Text) (upper Text) {
	defer func() { Text_Invariants(upper, "to_upper_special.upper") }()
	ucd.Special_Case_Invariants(special, "to_upper_special.special")
	Text_Invariants(source, "to_upper_special.source")
	mapping := func(character rune) (mapped rune) {
		return rune(ucd.Special_Case_To_Upper(special, ucd.Character(character)))
	}
	return Map(mapping, source)
}

// To_Lower_Special keeps an injected special case inside the text budget.
func To_Lower_Special(special ucd.Special_Case, source Text) (lower Text) {
	defer func() { Text_Invariants(lower, "to_lower_special.lower") }()
	ucd.Special_Case_Invariants(special, "to_lower_special.special")
	Text_Invariants(source, "to_lower_special.source")
	mapping := func(character rune) (mapped rune) {
		return rune(ucd.Special_Case_To_Lower(special, ucd.Character(character)))
	}
	return Map(mapping, source)
}

// To_Title_Special keeps an injected special case inside the text budget.
func To_Title_Special(special ucd.Special_Case, source Text) (title Text) {
	defer func() { Text_Invariants(title, "to_title_special.title") }()
	ucd.Special_Case_Invariants(special, "to_title_special.special")
	Text_Invariants(source, "to_title_special.source")
	mapping := func(character rune) (mapped rune) {
		return rune(ucd.Special_Case_To_Title(special, ucd.Character(character)))
	}
	return Map(mapping, source)
}

// To_Valid_UTF8 checks replacement expansion before it crosses the package boundary.
func To_Valid_UTF8(source Text, replacement Text) (valid Text) {
	defer func() { Text_Invariants(valid, "to_valid_utf8.valid") }()
	Text_Invariants(source, "to_valid_utf8.source")
	Text_Invariants(replacement, "to_valid_utf8.replacement")
	content := make([]byte, 0, len(source))
	invalid := false
	for position := 0; position < len(source); {
		character, size := utf8.Decode_Character_Text(utf8.Text(source[position:]))
		if character == utf8.REPLACEMENT_CHARACTER {
			if size == 1 {
				if !invalid {
					content = append(content, replacement...)
					if len(content) > TEXT_SIZE_MAXIMUM {
						panic(Error_Too_Large)
					}
				}
				invalid = true
				position++
				continue
			}
		}
		invalid = false
		content = append(content, source[position:position+int(size)]...)
		position += int(size)
	}
	return Text(string(content))
}

// Title preserves the deprecated standard word-boundary behavior for port compatibility.
func Title(source Text) (title Text) {
	defer func() { Text_Invariants(title, "title.title") }()
	Text_Invariants(source, "title.source")
	previous := rune(' ')
	mapping := func(character rune) (mapped rune) {
		separator := false
		if previous <= 0x7f {
			separator = true
			if previous >= '0' {
				if previous <= '9' {
					separator = false
				}
			}
			if previous >= 'a' {
				if previous <= 'z' {
					separator = false
				}
			}
			if previous >= 'A' {
				if previous <= 'Z' {
					separator = false
				}
			}
			if previous == '_' {
				separator = false
			}
		} else if ucd.Is_Space(ucd.Character(previous)) {
			separator = true
		}
		if separator {
			previous = character
			return rune(ucd.To_Title(ucd.Character(character)))
		}
		previous = character
		return character
	}
	return Map(mapping, source)
}

// Trim keeps a substring result inside its source budget.
func Trim(source Text, cutset Text) (trimmed Text) {
	defer func() { Text_Invariants(trimmed, "trim.trimmed") }()
	Text_Invariants(source, "trim.source")
	Text_Invariants(cutset, "trim.cutset")
	predicate := func(character rune) (matches bool) {
		return bool(Contains_Rune(cutset, Character(character)))
	}
	return Trim_Function(source, predicate)
}

// Trim_Left keeps a substring result inside its source budget.
func Trim_Left(source Text, cutset Text) (trimmed Text) {
	defer func() { Text_Invariants(trimmed, "trim_left.trimmed") }()
	Text_Invariants(source, "trim_left.source")
	Text_Invariants(cutset, "trim_left.cutset")
	predicate := func(character rune) (matches bool) {
		return bool(Contains_Rune(cutset, Character(character)))
	}
	return Trim_Left_Function(source, predicate)
}

// Trim_Right keeps a substring result inside its source budget.
func Trim_Right(source Text, cutset Text) (trimmed Text) {
	defer func() { Text_Invariants(trimmed, "trim_right.trimmed") }()
	Text_Invariants(source, "trim_right.source")
	Text_Invariants(cutset, "trim_right.cutset")
	predicate := func(character rune) (matches bool) {
		return bool(Contains_Rune(cutset, Character(character)))
	}
	return Trim_Right_Function(source, predicate)
}

// Trim_Function keeps an injected character rule explicit.
func Trim_Function(
	source Text, predicate func(rune) (matches bool),
) (trimmed Text) {
	defer func() { Text_Invariants(trimmed, "trim_function.trimmed") }()
	Text_Invariants(source, "trim_function.source")
	return Trim_Right_Function(Trim_Left_Function(source, predicate), predicate)
}

// Trim_Left_Function keeps an injected character rule explicit.
func Trim_Left_Function(
	source Text, predicate func(rune) (matches bool),
) (trimmed Text) {
	defer func() { Text_Invariants(trimmed, "trim_left_function.trimmed") }()
	Text_Invariants(source, "trim_left_function.source")
	for position, character := range string(source) {
		if !predicate(character) {
			return source[position:]
		}
	}
	return ""
}

// Trim_Right_Function keeps an injected character rule explicit.
func Trim_Right_Function(
	source Text, predicate func(rune) (matches bool),
) (trimmed Text) {
	defer func() { Text_Invariants(trimmed, "trim_right_function.trimmed") }()
	Text_Invariants(source, "trim_right_function.source")
	position_count := len(source)
	for position_count > 0 {
		character, size := utf8.Decode_Final_Character_Text(
			utf8.Text(source[:position_count]),
		)
		if !predicate(rune(character)) {
			break
		}
		position_count -= int(size)
	}
	return source[:position_count]
}

// Trim_Space uses the Unicode space set that Fields also uses.
func Trim_Space(source Text) (trimmed Text) {
	defer func() { Text_Invariants(trimmed, "trim_space.trimmed") }()
	Text_Invariants(source, "trim_space.source")
	space := func(character rune) (yes bool) {
		return bool(ucd.Is_Space(ucd.Character(character)))
	}
	return Trim_Function(source, space)
}

// Trim_Prefix preserves the source when the prefix is absent.
func Trim_Prefix(source Text, prefix Text) (trimmed Text) {
	defer func() { Text_Invariants(trimmed, "trim_prefix.trimmed") }()
	Text_Invariants(source, "trim_prefix.source")
	Text_Invariants(prefix, "trim_prefix.prefix")
	if Has_Prefix(source, prefix) {
		return source[len(prefix):]
	}
	return source
}

// Trim_Suffix preserves the source when the suffix is absent.
func Trim_Suffix(source Text, suffix Text) (trimmed Text) {
	defer func() { Text_Invariants(trimmed, "trim_suffix.trimmed") }()
	Text_Invariants(source, "trim_suffix.source")
	Text_Invariants(suffix, "trim_suffix.suffix")
	if Has_Suffix(source, suffix) {
		return source[:len(source)-len(suffix)]
	}
	return source
}

// Replace admits the standard all-results sentinel through Replacement_Count.
func Replace(
	source Text, old Text, replacement Text, count Replacement_Count,
) (replaced Text) {
	defer func() { Text_Invariants(replaced, "replace.replaced") }()
	Text_Invariants(source, "replace.source")
	Text_Invariants(old, "replace.old")
	Text_Invariants(replacement, "replace.replacement")
	Replacement_Count_Invariants(count, "replace.count")
	if old == replacement {
		return source
	}
	if count == 0 {
		return source
	}
	match_count := int(Count(source, old))
	if match_count == 0 {
		return source
	}
	replacement_count := int(count)
	if replacement_count < 0 {
		replacement_count = match_count
	} else if replacement_count > match_count {
		replacement_count = match_count
	}
	result_size := len(source) + replacement_count*(len(replacement)-len(old))
	if result_size > TEXT_SIZE_MAXIMUM {
		panic(Error_Too_Large)
	}
	content := make([]byte, 0, result_size)
	start := 0
	if len(old) > 0 {
		for replaced_index := 0; replaced_index < replacement_count; replaced_index++ {
			position := start + int(Index(source[start:], old))
			content = append(content, source[start:position]...)
			content = append(content, replacement...)
			start = position + len(old)
		}
	} else {
		content = append(content, replacement...)
		for replaced_index := 1; replaced_index < replacement_count; replaced_index++ {
			_, size := utf8.Decode_Character_Text(utf8.Text(source[start:]))
			position := start + int(size)
			content = append(content, source[start:position]...)
			content = append(content, replacement...)
			start = position
		}
	}
	content = append(content, source[start:]...)
	return Text(string(content))
}

// Replace_All checks the complete replacement expansion against the text budget.
func Replace_All(source Text, old Text, replacement Text) (replaced Text) {
	defer func() { Text_Invariants(replaced, "replace_all.replaced") }()
	Text_Invariants(source, "replace_all.source")
	Text_Invariants(old, "replace_all.old")
	Text_Invariants(replacement, "replace_all.replacement")
	return Replace(source, old, replacement, REPLACEMENT_COUNT_MINIMUM)
}

// Equal_Fold keeps Unicode simple folding in the Boolean coverage domain.
func Equal_Fold(left Text, right Text) (equal Boolean) {
	defer func() { Boolean_Invariants(equal, "equal_fold.equal") }()
	Text_Invariants(left, "equal_fold.left")
	Text_Invariants(right, "equal_fold.right")
	left_tail := left
	right_tail := right
	for len(left_tail) > 0 {
		if len(right_tail) == 0 {
			return false
		}
		left_character, left_size := utf8.Decode_Character_Text(utf8.Text(left_tail))
		right_character, right_size := utf8.Decode_Character_Text(utf8.Text(right_tail))
		left_tail = left_tail[left_size:]
		right_tail = right_tail[right_size:]
		if left_character == right_character {
			continue
		}
		matched := false
		left_code_point := ucd.Character(left_character)
		right_code_point := ucd.Character(right_character)
		folded := ucd.Simple_Fold(left_code_point)
		for folded != left_code_point {
			if folded == right_code_point {
				matched = true
				break
			}
			folded = ucd.Simple_Fold(folded)
		}
		if !matched {
			return false
		}
	}
	return Boolean(len(right_tail) == 0)
}

// Cut preserves both substrings and reports separator presence separately.
func Cut(source Text, separator Text) (before Text, after Text, found Boolean) {
	defer func() {
		Text_Invariants(before, "cut.before")
		Text_Invariants(after, "cut.after")
		Boolean_Invariants(found, "cut.found")
	}()
	Text_Invariants(source, "cut.source")
	Text_Invariants(separator, "cut.separator")
	position := Index(source, separator)
	if position == INDEX_ABSENT {
		return source, "", false
	}
	return source[:position], source[int(position)+len(separator):], true
}

// Cut_Prefix preserves the source and reports absence as a separate fact.
func Cut_Prefix(source Text, prefix Text) (after Text, found Boolean) {
	defer func() {
		Text_Invariants(after, "cut_prefix.after")
		Boolean_Invariants(found, "cut_prefix.found")
	}()
	Text_Invariants(source, "cut_prefix.source")
	Text_Invariants(prefix, "cut_prefix.prefix")
	if Has_Prefix(source, prefix) {
		return source[len(prefix):], true
	}
	return source, false
}

// Cut_Suffix preserves the source and reports absence as a separate fact.
func Cut_Suffix(source Text, suffix Text) (before Text, found Boolean) {
	defer func() {
		Text_Invariants(before, "cut_suffix.before")
		Boolean_Invariants(found, "cut_suffix.found")
	}()
	Text_Invariants(source, "cut_suffix.source")
	Text_Invariants(suffix, "cut_suffix.suffix")
	if Has_Suffix(source, suffix) {
		return source[:len(source)-len(suffix)], true
	}
	return source, false
}

// Lines avoids a result collection while each yielded value remains a source substring.
func Lines(source Text) (sequence iter.Seq[Text]) {
	Text_Invariants(source, "lines.source")
	return func(yield func(Text) (continue_iteration bool)) {
		tail := source
		for len(tail) > 0 {
			position := Index_Byte(tail, '\n')
			if position == INDEX_ABSENT {
				yield(tail)
				return
			}
			end := int(position) + 1
			if !yield(tail[:end]) {
				return
			}
			tail = tail[end:]
		}
	}
}

// Split_Sequence avoids the collection that Split allocates.
func Split_Sequence(source Text, separator Text) (sequence iter.Seq[Text]) {
	Text_Invariants(source, "split_sequence.source")
	Text_Invariants(separator, "split_sequence.separator")
	return split_sequence_text(source, separator, 0)
}

// Split_After_Sequence keeps separators without a result collection.
func Split_After_Sequence(source Text, separator Text) (sequence iter.Seq[Text]) {
	Text_Invariants(source, "split_after_sequence.source")
	Text_Invariants(separator, "split_after_sequence.separator")
	return split_sequence_text(source, separator, Boundary(len(separator)))
}

// Fields_Sequence avoids the field collection while it uses Unicode space.
func Fields_Sequence(source Text) (sequence iter.Seq[Text]) {
	Text_Invariants(source, "fields_sequence.source")
	space := func(character rune) (yes bool) {
		return bool(ucd.Is_Space(ucd.Character(character)))
	}
	return fields_sequence_text(source, space)
}

// Fields_Function_Sequence avoids a collection for an injected field rule.
func Fields_Function_Sequence(
	source Text, predicate func(rune) (matches bool),
) (sequence iter.Seq[Text]) {
	Text_Invariants(source, "fields_function_sequence.source")
	return fields_sequence_text(source, predicate)
}

func split_text(
	source Text, separator Text, saved Boundary, limit Limit,
) (parts Texts) {
	defer func() { Texts_Invariants(parts, "split_text.parts") }()
	Text_Invariants(source, "split_text.source")
	Text_Invariants(separator, "split_text.separator")
	Boundary_Invariants(saved, "split_text.saved")
	Limit_Invariants(limit, "split_text.limit")
	if limit == 0 {
		return nil
	}
	if len(separator) == 0 {
		result_count := int(utf8.Character_Count_Text(utf8.Text(source)))
		if limit >= 0 {
			if int(limit) < result_count {
				result_count = int(limit)
			}
		}
		if result_count == 0 {
			return nil
		}
		parts = make(Texts, result_count)
		tail := source
		for index := 0; index < result_count-1; index++ {
			_, size := utf8.Decode_Character_Text(utf8.Text(tail))
			parts[index] = tail[:size]
			tail = tail[size:]
		}
		parts[result_count-1] = tail
		return parts
	}
	result_count := int(Count(source, separator)) + 1
	if limit > 0 {
		if int(limit) < result_count {
			result_count = int(limit)
		}
	}
	parts = make(Texts, result_count)
	tail := source
	for index := 0; index < result_count-1; index++ {
		position := int(Index(tail, separator))
		parts[index] = tail[:position+int(saved)]
		tail = tail[position+len(separator):]
	}
	parts[result_count-1] = tail
	return parts
}

func fields_text(
	source Text, predicate func(rune) (matches bool),
) (fields Field_Texts) {
	defer func() { Field_Texts_Invariants(fields, "fields_text.fields") }()
	Text_Invariants(source, "fields_text.source")
	start := Index_Value(INDEX_ABSENT)
	for position, character := range string(source) {
		if predicate(character) {
			if start >= 0 {
				fields = append(fields, source[start:position])
				start = INDEX_ABSENT
			}
		} else if start == INDEX_ABSENT {
			start = Index_Value(position)
		}
	}
	if start >= 0 {
		fields = append(fields, source[start:])
	}
	return fields
}

func split_sequence_text(
	source Text, separator Text, saved Boundary,
) (sequence iter.Seq[Text]) {
	Text_Invariants(source, "split_sequence_text.source")
	Text_Invariants(separator, "split_sequence_text.separator")
	Boundary_Invariants(saved, "split_sequence_text.saved")
	return func(yield func(Text) (continue_iteration bool)) {
		tail := source
		if len(separator) == 0 {
			for len(tail) > 0 {
				_, size := utf8.Decode_Character_Text(utf8.Text(tail))
				if !yield(tail[:size]) {
					return
				}
				tail = tail[size:]
			}
			return
		}
		position := Index(tail, separator)
		for position != INDEX_ABSENT {
			if !yield(tail[:int(position)+int(saved)]) {
				return
			}
			tail = tail[int(position)+len(separator):]
			position = Index(tail, separator)
		}
		yield(tail)
	}
}

func fields_sequence_text(
	source Text, predicate func(rune) (matches bool),
) (sequence iter.Seq[Text]) {
	Text_Invariants(source, "fields_sequence_text.source")
	return func(yield func(Text) (continue_iteration bool)) {
		start := Index_Value(INDEX_ABSENT)
		for position, character := range string(source) {
			if predicate(character) {
				if start >= 0 {
					if !yield(source[start:position]) {
						return
					}
					start = INDEX_ABSENT
				}
			} else if start == INDEX_ABSENT {
				start = Index_Value(position)
			}
		}
		if start >= 0 {
			yield(source[start:])
		}
	}
}
