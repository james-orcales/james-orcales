// Package asn1 implements bounded DER elements on caller-owned storage.
package asn1

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/simulation/aver/default"
)

// ENCODED_SIZE_MAXIMUM follows the repository byte-slice boundary.
const ENCODED_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// IDENTIFIER_SIZE_MINIMUM is one low-tag octet.
const IDENTIFIER_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM + 1

// CONTENT_SIZE_FIELD_SIZE_MINIMUM is one short-form size octet.
const CONTENT_SIZE_FIELD_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM + 1

// CONTENT_SIZE_VALUE_OCTET_COUNT_MAXIMUM covers bounded encoded storage.
const CONTENT_SIZE_VALUE_OCTET_COUNT_MAXIMUM = bits.BIT_COUNT_16_MAXIMUM /
	bits.BIT_COUNT_8_MAXIMUM

// CONTENT_SIZE_FIELD_SIZE_MAXIMUM includes the long-form lead octet.
const CONTENT_SIZE_FIELD_SIZE_MAXIMUM = CONTENT_SIZE_FIELD_SIZE_MINIMUM +
	CONTENT_SIZE_VALUE_OCTET_COUNT_MAXIMUM

// CONTENT_SIZE_FIELD_SIZE_MIDDLE is the one-octet long form.
const CONTENT_SIZE_FIELD_SIZE_MIDDLE = CONTENT_SIZE_FIELD_SIZE_MINIMUM + 1

// CONTENT_SIZE_MAXIMUM leaves the largest required canonical header.
const CONTENT_SIZE_MAXIMUM = ENCODED_SIZE_MAXIMUM - IDENTIFIER_SIZE_MINIMUM -
	CONTENT_SIZE_FIELD_SIZE_MAXIMUM

// POSITION_MAXIMUM includes unexpected end immediately after maximum input.
const POSITION_MAXIMUM = ENCODED_SIZE_MAXIMUM + 1

// CLASS_BIT_COUNT is the DER identifier class field width.
const CLASS_BIT_COUNT = 2

// CONSTRUCTED_BIT_COUNT is the DER identifier construction field width.
const CONSTRUCTED_BIT_COUNT = 1

// LOW_TAG_BIT_COUNT is the remaining identifier lead-octet width.
const LOW_TAG_BIT_COUNT = bits.BIT_COUNT_8_MAXIMUM - CLASS_BIT_COUNT -
	CONSTRUCTED_BIT_COUNT

// HIGH_TAG_GROUP_BIT_COUNT is one base-128 payload width.
const HIGH_TAG_GROUP_BIT_COUNT = bits.BIT_COUNT_8_MAXIMUM - 1

// CLASS_SHIFT places class above construction and low tag.
const CLASS_SHIFT = LOW_TAG_BIT_COUNT + CONSTRUCTED_BIT_COUNT

// CONSTRUCTED_MASK occupies the construction field.
const CONSTRUCTED_MASK = 1 << LOW_TAG_BIT_COUNT

// HIGH_TAG_MARKER selects the base-128 identifier form.
const HIGH_TAG_MARKER = 1<<LOW_TAG_BIT_COUNT - 1

// HIGH_TAG_VALUE_MASK removes the continuation bit.
const HIGH_TAG_VALUE_MASK = 1<<HIGH_TAG_GROUP_BIT_COUNT - 1

// CONTINUATION_MASK marks another base-128 or length octet.
const CONTINUATION_MASK = 1 << HIGH_TAG_GROUP_BIT_COUNT

// CONTENT_SIZE_SHORT_MAXIMUM is the largest short-form content size.
const CONTENT_SIZE_SHORT_MAXIMUM = CONTINUATION_MASK - 1

// CLASS_UNIVERSAL identifies standard ASN.1 tags.
const CLASS_UNIVERSAL = 0

// CLASS_APPLICATION identifies application-defined tags.
const CLASS_APPLICATION = CLASS_UNIVERSAL + 1

// CLASS_CONTEXT_SPECIFIC identifies schema-context tags.
const CLASS_CONTEXT_SPECIFIC = CLASS_APPLICATION + 1

// CLASS_PRIVATE identifies private tags.
const CLASS_PRIVATE = CLASS_CONTEXT_SPECIFIC + 1

// CLASS_MAXIMUM is the largest identifier class field.
const CLASS_MAXIMUM = CLASS_PRIVATE

// TAG_MAXIMUM keeps accumulation inside the signed ASN.1 tag boundary.
const TAG_MAXIMUM = uint32(bits.INTEGER_32_MAXIMUM)

// IDENTIFIER_SIZE_MAXIMUM is the lead octet plus every tag group.
const IDENTIFIER_SIZE_MAXIMUM = IDENTIFIER_SIZE_MINIMUM +
	(bits.BIT_COUNT_32_MAXIMUM+HIGH_TAG_GROUP_BIT_COUNT-1)/HIGH_TAG_GROUP_BIT_COUNT

// IDENTIFIER_POSITION_MAXIMUM is the byte after the largest identifier.
const IDENTIFIER_POSITION_MAXIMUM = IDENTIFIER_SIZE_MAXIMUM + 1

// CONTENT_START_INDEX_MAXIMUM is the boundary after the largest header.
const CONTENT_START_INDEX_MAXIMUM = IDENTIFIER_SIZE_MAXIMUM +
	CONTENT_SIZE_FIELD_SIZE_MAXIMUM

// CONTENT_SIZE_POSITION_MAXIMUM is the byte after the largest size value.
const CONTENT_SIZE_POSITION_MAXIMUM = IDENTIFIER_SIZE_MAXIMUM +
	CONTENT_SIZE_VALUE_OCTET_COUNT_MAXIMUM

// STATUS_OK means the operation completed.
const STATUS_OK = 0

// STATUS_ELEMENT_INVALID means class or tag cannot form DER.
const STATUS_ELEMENT_INVALID = STATUS_OK + 1

// STATUS_ELEMENT_TOO_LARGE means bounded fields exceed bounded output together.
const STATUS_ELEMENT_TOO_LARGE = STATUS_ELEMENT_INVALID + 1

// STATUS_OUTPUT_TOO_SMALL means caller output cannot hold the exact element.
const STATUS_OUTPUT_TOO_SMALL = STATUS_ELEMENT_TOO_LARGE + 1

// STATUS_STORAGE_INVALID means encoded output overlaps borrowed content.
const STATUS_STORAGE_INVALID = STATUS_OUTPUT_TOO_SMALL + 1

// STATUS_INPUT_INVALID means encoded input is not one canonical DER element.
const STATUS_INPUT_INVALID = STATUS_STORAGE_INVALID + 1

// COUNT_HOLE excludes the impossible one-byte DER element.
const COUNT_HOLE = bytes.SLICE_SIZE_MINIMUM + 1

// Class_Input retains every caller-supplied class byte before validation.
type Class_Input uint8

// Class_Input_Invariants covers the complete storage domain.
func Class_Input_Invariants(value Class_Input, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), bits.WORD_8_MINIMUM, bits.WORD_8_MAXIMUM).
		Ensure()
}

// Tag_Input retains every caller-supplied tag word before validation.
type Tag_Input uint32

// Tag_Input_Invariants covers the complete storage domain.
func Tag_Input_Invariants(value Tag_Input, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, bits.WORD_32_MAXIMUM).
		Ensure()
}

// Class is one validated DER identifier class.
type Class uint8

// Class_Invariants bounds decoded classes to their bit field.
func Class_Invariants(value Class, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), CLASS_UNIVERSAL, CLASS_APPLICATION,
			CLASS_CONTEXT_SPECIFIC, CLASS_PRIVATE,
		).
		Ensure()
}

// Tag is one validated nonnegative ASN.1 tag.
type Tag uint32

// Tag_Invariants keeps decoded accumulation within the DER tag boundary.
func Tag_Invariants(value Tag, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, TAG_MAXIMUM).
		Ensure()
}

// Constructed retains the DER construction field independently from tag.
type Constructed bool

// Constructed_Invariants covers primitive and constructed identifiers.
func Constructed_Invariants(value Constructed, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A DER identifier is constructed.").
		Ensure()
}

// Content is caller-owned input or a borrowed decoded value.
type Content []byte

// Content_Invariants leaves room for the largest canonical header.
func Content_Invariants(value Content, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, CONTENT_SIZE_MAXIMUM).
		Ensure()
}

// Encoded is bounded DER input.
type Encoded []byte

// Encoded_Invariants enforces the shared source boundary.
func Encoded_Invariants(value Encoded, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Output is caller-owned bounded DER storage.
type Output []byte

// Output_Invariants enforces the shared destination boundary.
func Output_Invariants(value Output, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Element_Input is one caller-supplied element before class and tag validation.
type Element_Input struct {
	// Class remains raw so invalid bit fields return status before output mutation.
	Class Class_Input
	// Tag remains raw so oversized values return status before output mutation.
	Tag Tag_Input
	// Constructed stays separate because tag numbers do not imply wire construction.
	Constructed Constructed
	// Content stays borrowed so encoding never owns caller bytes.
	Content Content
}

// Element_Input_Invariants composes complete caller storage domains.
func Element_Input_Invariants(value Element_Input, namespace aver.Namespace) {
	Class_Input_Invariants(value.Class, namespace)
	Tag_Input_Invariants(value.Tag, namespace)
	Constructed_Invariants(value.Constructed, namespace)
	Content_Invariants(value.Content, namespace)
}

// Element is one validated DER element borrowing its content.
type Element struct {
	// Class has passed the identifier bit-field boundary.
	Class Class
	// Tag has passed minimal high-tag decoding and overflow checks.
	Tag Tag
	// Constructed preserves the identifier bit instead of inferring schema.
	Constructed Constructed
	// Content borrows input so decoding needs no hidden storage.
	Content Content
}

// Element_Invariants composes canonical decoded fields.
func Element_Invariants(value Element, namespace aver.Namespace) {
	Class_Invariants(value.Class, namespace)
	Tag_Invariants(value.Tag, namespace)
	Constructed_Invariants(value.Constructed, namespace)
	Content_Invariants(value.Content, namespace)
}

// Count is exact encoded bytes or zero on refusal.
type Count int

// Count_Invariants keeps results inside bounded output.
func Count_Invariants(value Count, namespace aver.Namespace) {
	valid := value == 0 || int(value) > COUNT_HOLE
	aver.Always(valid, "A nonzero DER count contains identifier and length.")
	aver.Tree(value, namespace).
		Range_Holed_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM,
			COUNT_HOLE, COUNT_HOLE, COUNT_HOLE, COUNT_HOLE,
		).
		Ensure()
}

// Consumed_Count is one complete element prefix or zero on refusal.
type Consumed_Count int

// Consumed_Count_Invariants keeps consumption inside input.
func Consumed_Count_Invariants(value Consumed_Count, namespace aver.Namespace) {
	valid := value == 0 || int(value) > COUNT_HOLE
	aver.Always(valid, "A consumed DER prefix contains identifier and length.")
	aver.Tree(value, namespace).
		Range_Holed_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM,
			COUNT_HOLE, COUNT_HOLE, COUNT_HOLE, COUNT_HOLE,
		).
		Ensure()
}

// Position is one-based invalid input or zero when absent.
type Position int

// Position_Invariants includes unexpected end after maximum input.
func Position_Invariants(value Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, POSITION_MAXIMUM).
		Ensure()
}

// Size_Status reports validation and aggregate-size outcomes.
type Size_Status uint8

// Size_Status_Invariants lists all sizing outcomes.
func Size_Status_Invariants(value Size_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), STATUS_OK, STATUS_ELEMENT_INVALID,
			STATUS_ELEMENT_TOO_LARGE,
		).
		Ensure()
}

// Encode_Status adds destination and overlap refusals.
type Encode_Status uint8

// Encode_Status_Invariants lists every encode outcome.
func Encode_Status_Invariants(value Encode_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), STATUS_OK, STATUS_STORAGE_INVALID).
		Ensure()
}

// Decode_Status excludes encode-only refusals.
type Decode_Status uint8

// Decode_Status_Invariants lists valid and invalid DER.
func Decode_Status_Invariants(value Decode_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), STATUS_OK, STATUS_INPUT_INVALID).
		Ensure()
}

// Identifier_Size_Count is one low tag or a bounded high-tag identifier.
type Identifier_Size_Count int

// Identifier_Size_Count_Invariants follows the tag-word width.
func Identifier_Size_Count_Invariants(
	value Identifier_Size_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), IDENTIFIER_SIZE_MINIMUM, IDENTIFIER_SIZE_MAXIMUM).
		Ensure()
}

// Content_Size_Field_Count is one short form or a bounded long form.
type Content_Size_Field_Count int

// Content_Size_Field_Count_Invariants follows bounded content size.
func Content_Size_Field_Count_Invariants(
	value Content_Size_Field_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Int(
			int(value), CONTENT_SIZE_FIELD_SIZE_MINIMUM,
			CONTENT_SIZE_FIELD_SIZE_MIDDLE, CONTENT_SIZE_FIELD_SIZE_MAXIMUM,
		).
		Ensure()
}

// Content_Size_Count is a decoded bounded content byte count.
type Content_Size_Count int

// Content_Size_Count_Invariants keeps decoded content inside one element.
func Content_Size_Count_Invariants(
	value Content_Size_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, CONTENT_SIZE_MAXIMUM).
		Ensure()
}

// Element_Valid reports semantic input validation without an error interface.
type Element_Valid bool

// Element_Valid_Invariants covers accepted and rejected fields.
func Element_Valid_Invariants(value Element_Valid, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A caller DER element is valid.").
		Ensure()
}

// High_Tag_Value holds incomplete or validated base-128 accumulation.
type High_Tag_Value uint32

// High_Tag_Value_Invariants prevents parser arithmetic from escaping tag storage.
func High_Tag_Value_Invariants(value High_Tag_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, TAG_MAXIMUM).
		Ensure()
}

// Identifier_End_Index is zero on refusal or the boundary after a high tag.
type Identifier_End_Index int

// Identifier_End_Index_Invariants excludes the impossible one-byte high-tag result.
func Identifier_End_Index_Invariants(
	value Identifier_End_Index, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, IDENTIFIER_SIZE_MAXIMUM,
			COUNT_HOLE, COUNT_HOLE, COUNT_HOLE, COUNT_HOLE,
		).
		Ensure()
}

// Identifier_Position is an internal high-tag error or zero.
type Identifier_Position int

// Identifier_Position_Invariants excludes the impossible position before the high tag.
func Identifier_Position_Invariants(
	value Identifier_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, IDENTIFIER_POSITION_MAXIMUM,
			COUNT_HOLE, COUNT_HOLE, COUNT_HOLE, COUNT_HOLE,
		).
		Ensure()
}

// Content_Start_Index is zero on refusal or the boundary after the size field.
type Content_Start_Index int

// Content_Start_Index_Invariants excludes a boundary inside the identifier lead octet.
func Content_Start_Index_Invariants(
	value Content_Start_Index, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, CONTENT_START_INDEX_MAXIMUM,
			COUNT_HOLE, COUNT_HOLE, COUNT_HOLE, COUNT_HOLE,
		).
		Ensure()
}

// Content_Size_Position is an internal size-field error or zero.
type Content_Size_Position int

// Content_Size_Position_Invariants excludes the identifier lead octet.
func Content_Size_Position_Invariants(
	value Content_Size_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, CONTENT_SIZE_POSITION_MAXIMUM,
			COUNT_HOLE, COUNT_HOLE, COUNT_HOLE, COUNT_HOLE,
		).
		Ensure()
}

// High_Tag_Result keeps parser scalars together without an owned diagnostic.
type High_Tag_Result struct {
	// Tag stays raw until the complete base-128 sequence passes validation.
	Tag High_Tag_Value
	// Next avoids retaining an input suffix inside parser state.
	Next Identifier_End_Index
	// Position remains zero unless the identifier is invalid.
	Position Identifier_Position
	// Valid separates parser control from tag storage.
	Valid Element_Valid
}

// High_Tag_Result_Invariants bounds raw parser state before public conversion.
func High_Tag_Result_Invariants(value High_Tag_Result, namespace aver.Namespace) {
	High_Tag_Value_Invariants(value.Tag, namespace)
	Identifier_End_Index_Invariants(value.Next, namespace)
	Identifier_Position_Invariants(value.Position, namespace)
	Element_Valid_Invariants(value.Valid, namespace)
}

// Content_Size_Result keeps size-field parser scalars together.
type Content_Size_Result struct {
	// Content_Size stays bounded before slicing input.
	Content_Size Content_Size_Count
	// Next avoids retaining an input suffix inside parser state.
	Next Content_Start_Index
	// Position remains zero unless the size field is invalid.
	Position Content_Size_Position
	// Valid separates parser control from content size.
	Valid Element_Valid
}

// Content_Size_Result_Invariants bounds parser state before public conversion.
func Content_Size_Result_Invariants(
	value Content_Size_Result, namespace aver.Namespace,
) {
	Content_Size_Count_Invariants(value.Content_Size, namespace)
	Content_Start_Index_Invariants(value.Next, namespace)
	Content_Size_Position_Invariants(value.Position, namespace)
	Element_Valid_Invariants(value.Valid, namespace)
}

// Encoded_Size checks scalar fields before calculating aggregate storage.
func Encoded_Size(element Element_Input) (count Count, status Size_Status) {
	defer func() {
		Count_Invariants(count, "Encoded_Size.count")
		Size_Status_Invariants(status, "Encoded_Size.status")
	}()
	Element_Input_Invariants(element, "Encoded_Size.element")
	if !bool(element_valid(element)) {
		return 0, STATUS_ELEMENT_INVALID
	}
	identifier_count := identifier_size(element.Tag)
	content_size_count := content_size_field_count(element.Content)
	required := int(identifier_count) + int(content_size_count) + len(element.Content)
	if required > ENCODED_SIZE_MAXIMUM {
		return 0, STATUS_ELEMENT_TOO_LARGE
	}
	return Count(required), STATUS_OK
}

// Encode_Into rejects every refusal before changing caller output.
func Encode_Into(destination Output, element Element_Input) (
	count Count, status Encode_Status,
) {
	defer func() {
		Count_Invariants(count, "Encode_Into.count")
		Encode_Status_Invariants(status, "Encode_Into.status")
	}()
	Output_Invariants(destination, "Encode_Into.destination")
	Element_Input_Invariants(element, "Encode_Into.element")
	required, size_status := Encoded_Size(element)
	if size_status == STATUS_ELEMENT_INVALID {
		return 0, STATUS_ELEMENT_INVALID
	}
	if size_status == STATUS_ELEMENT_TOO_LARGE {
		return 0, STATUS_ELEMENT_TOO_LARGE
	}
	if bytes.Overlap(bytes.Slice(destination), bytes.Slice(element.Content)) {
		return 0, STATUS_STORAGE_INVALID
	}
	if len(destination) < int(required) {
		return 0, STATUS_OUTPUT_TOO_SMALL
	}
	encode_unchecked(
		destination, element.Class, element.Tag, element.Constructed, element.Content,
	)
	return required, STATUS_OK
}

// Decode validates one canonical element while borrowing its content.
func Decode(source Encoded) (
	element Element, consumed Consumed_Count, position Position, status Decode_Status,
) {
	defer func() {
		Element_Invariants(element, "Decode.element")
		Consumed_Count_Invariants(consumed, "Decode.consumed")
		Position_Invariants(position, "Decode.position")
		Decode_Status_Invariants(status, "Decode.status")
	}()
	Encoded_Invariants(source, "Decode.source")
	if len(source) == bytes.SLICE_SIZE_MINIMUM {
		return Element{}, 0, 1, STATUS_INPUT_INVALID
	}
	first := source[bytes.SLICE_SIZE_MINIMUM]
	element.Class = Class(first >> CLASS_SHIFT)
	element.Constructed = Constructed(first&CONSTRUCTED_MASK != 0)
	index := IDENTIFIER_SIZE_MINIMUM
	tag := Tag(first & HIGH_TAG_MARKER)
	if tag == HIGH_TAG_MARKER {
		high_tag := decode_high_tag(source, index)
		if !bool(high_tag.Valid) {
			return Element{}, 0, Position(high_tag.Position), STATUS_INPUT_INVALID
		}
		tag = Tag(high_tag.Tag)
		index = int(high_tag.Next)
	}
	element.Tag = tag
	size_result := decode_content_size(source, index)
	if !bool(size_result.Valid) {
		return Element{}, 0, Position(size_result.Position), STATUS_INPUT_INVALID
	}
	if int(size_result.Content_Size) > len(source)-int(size_result.Next) {
		return Element{}, 0, Position(len(source) + 1), STATUS_INPUT_INVALID
	}
	end_index := int(size_result.Next) + int(size_result.Content_Size)
	element.Content = Content(source[int(size_result.Next):end_index])
	return element, Consumed_Count(end_index), 0, STATUS_OK
}

func element_valid(element Element_Input) (valid Element_Valid) {
	defer func() { Element_Valid_Invariants(valid, "element_valid.valid") }()
	Element_Input_Invariants(element, "element_valid.element")
	if uint8(element.Class) > CLASS_MAXIMUM {
		return false
	}
	return Element_Valid(uint32(element.Tag) <= TAG_MAXIMUM)
}

func identifier_size[Tag_Value ~uint32](tag Tag_Value) (size Identifier_Size_Count) {
	defer func() {
		Identifier_Size_Count_Invariants(size, "identifier_size.size")
	}()
	if uint32(tag) < HIGH_TAG_MARKER {
		return IDENTIFIER_SIZE_MINIMUM
	}
	size = IDENTIFIER_SIZE_MINIMUM
	value := uint32(tag)
	for value > 0 {
		size++
		value >>= HIGH_TAG_GROUP_BIT_COUNT
	}
	return size
}

func content_size_field_count[Content_Value ~[]byte](
	content Content_Value,
) (size Content_Size_Field_Count) {
	defer func() {
		Content_Size_Field_Count_Invariants(size, "content_size_field_count.size")
	}()
	if len(content) <= CONTENT_SIZE_SHORT_MAXIMUM {
		return CONTENT_SIZE_FIELD_SIZE_MINIMUM
	}
	value_count := len(content)
	size = CONTENT_SIZE_FIELD_SIZE_MINIMUM
	for value_count > 0 {
		size++
		value_count >>= bits.BIT_COUNT_8_MAXIMUM
	}
	return size
}

func encode_unchecked[
	Destination ~[]byte, Class_Value ~uint8, Tag_Value ~uint32,
	Constructed_Value ~bool, Content_Value ~[]byte,
](
	destination Destination, class Class_Value, tag Tag_Value,
	constructed Constructed_Value, content Content_Value,
) {
	identifier_count := identifier_size(tag)
	encode_identifier(destination[:identifier_count], class, tag, constructed)
	content_size_count := content_size_field_count(content)
	content_size_start := int(identifier_count)
	encode_content_size(
		destination[content_size_start:content_size_start+int(content_size_count)],
		content,
	)
	copy(destination[content_size_start+int(content_size_count):], content)
}

func encode_identifier[
	Destination ~[]byte, Class_Value ~uint8, Tag_Value ~uint32,
	Constructed_Value ~bool,
](
	destination Destination, class Class_Value, tag Tag_Value,
	constructed Constructed_Value,
) {
	first := byte(class) << CLASS_SHIFT
	if constructed {
		first |= CONSTRUCTED_MASK
	}
	if uint32(tag) < HIGH_TAG_MARKER {
		destination[0] = first | byte(tag)
		return
	}
	destination[0] = first | HIGH_TAG_MARKER
	value := uint32(tag)
	for index := len(destination) - 1; index > 0; index-- {
		destination[index] = byte(value & HIGH_TAG_VALUE_MASK)
		if index < len(destination)-1 {
			destination[index] |= CONTINUATION_MASK
		}
		value >>= HIGH_TAG_GROUP_BIT_COUNT
	}
}

func encode_content_size[Destination ~[]byte, Content_Value ~[]byte](
	destination Destination, content Content_Value,
) {
	if len(content) <= CONTENT_SIZE_SHORT_MAXIMUM {
		destination[0] = byte(len(content))
		return
	}
	destination[0] = CONTINUATION_MASK |
		byte(len(destination)-CONTENT_SIZE_FIELD_SIZE_MINIMUM)
	value_count := len(content)
	for index := len(destination) - 1; index > 0; index-- {
		destination[index] = byte(value_count)
		value_count >>= bits.BIT_COUNT_8_MAXIMUM
	}
}

func decode_high_tag[Source ~[]byte, Index ~int](
	source Source, start Index,
) (result High_Tag_Result) {
	defer func() { High_Tag_Result_Invariants(result, "decode_high_tag.result") }()
	index := int(start)
	if index == len(source) {
		result.Position = Identifier_Position(len(source) + 1)
		return result
	}
	if source[index] == CONTINUATION_MASK {
		result.Position = Identifier_Position(index + 1)
		return result
	}
	for index < len(source) {
		value := source[index]
		if uint32(result.Tag) > TAG_MAXIMUM>>HIGH_TAG_GROUP_BIT_COUNT {
			result.Tag = 0
			result.Position = Identifier_Position(index + 1)
			return result
		}
		result.Tag = High_Tag_Value(
			uint32(result.Tag)<<HIGH_TAG_GROUP_BIT_COUNT |
				uint32(value&HIGH_TAG_VALUE_MASK),
		)
		index++
		if value&CONTINUATION_MASK == 0 {
			if result.Tag < HIGH_TAG_MARKER {
				result.Position = Identifier_Position(index)
				return result
			}
			result.Next = Identifier_End_Index(index)
			result.Valid = true
			return result
		}
	}
	result.Tag = 0
	result.Position = Identifier_Position(len(source) + 1)
	return result
}

func decode_content_size[Source ~[]byte, Index ~int](
	source Source, start Index,
) (result Content_Size_Result) {
	defer func() {
		Content_Size_Result_Invariants(result, "decode_content_size.result")
	}()
	index := int(start)
	if index == len(source) {
		result.Position = Content_Size_Position(len(source) + 1)
		return result
	}
	first := source[index]
	index++
	if first < CONTINUATION_MASK {
		result.Content_Size = Content_Size_Count(first)
		result.Next = Content_Start_Index(index)
		result.Valid = true
		return result
	}
	count := int(first & HIGH_TAG_VALUE_MASK)
	if count == bytes.SLICE_SIZE_MINIMUM {
		result.Position = Content_Size_Position(index)
		return result
	}
	if count > CONTENT_SIZE_VALUE_OCTET_COUNT_MAXIMUM {
		result.Position = Content_Size_Position(index)
		return result
	}
	if count > len(source)-index {
		result.Position = Content_Size_Position(len(source) + 1)
		return result
	}
	value_start := index
	if source[index] == 0 {
		result.Position = Content_Size_Position(index + 1)
		return result
	}
	for end_index := index + count; index < end_index; index++ {
		result.Content_Size = result.Content_Size<<bits.BIT_COUNT_8_MAXIMUM |
			Content_Size_Count(source[index])
	}
	if result.Content_Size <= CONTENT_SIZE_SHORT_MAXIMUM {
		result.Content_Size = 0
		result.Position = Content_Size_Position(value_start + 1)
		return result
	}
	if result.Content_Size > CONTENT_SIZE_MAXIMUM {
		result.Content_Size = 0
		result.Position = Content_Size_Position(value_start + 1)
		return result
	}
	result.Next = Content_Start_Index(index)
	result.Valid = true
	return result
}
