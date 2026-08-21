package asn1_test

import (
	"testing"

	"local/james-orcales/shared/encoding/asn1"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// Test_Element protects raw encode fields from leaking into decoded values.
func Test_Element(t *testing.T) {
	var destination [asn1.ENCODED_SIZE_MAXIMUM]byte
	input := asn1.Element_Input{
		Class: asn1.CLASS_APPLICATION, Tag: asn1.HIGH_TAG_MARKER,
		Constructed: true, Content: asn1.Content{1, 2},
	}
	count, encode_status := asn1.Encode_Into(destination[:], input)
	testify.Equal_Values(t, asn1.STATUS_OK, encode_status)
	element, consumed, _, decode_status := asn1.Decode(destination[:count])
	testify.Equal_Values(t, asn1.STATUS_OK, decode_status)
	testify.Equal(t, asn1.Class(input.Class), element.Class)
	testify.Equal(t, asn1.Tag(input.Tag), element.Tag)
	testify.Equal(t, input.Constructed, element.Constructed)
	testify.Equal(t, input.Content, element.Content)
	testify.Equal(t, asn1.Consumed_Count(count), consumed)
}

// Test_Encode protects canonical identifier and length forms.
func Test_Encode(t *testing.T) {
	var destination [asn1.ENCODED_SIZE_MAXIMUM]byte
	integer := asn1.Element_Input{
		Class: asn1.CLASS_UNIVERSAL, Tag: 2, Content: []byte{1},
	}
	count, status := asn1.Encoded_Size(integer)
	testify.Equal(t, asn1.Count(3), count)
	testify.Equal_Values(t, asn1.STATUS_OK, status)
	count, encode_status := asn1.Encode_Into(destination[:count], integer)
	testify.Equal_Values(t, asn1.STATUS_OK, encode_status)
	testify.Equal(t, []byte{2, 1, 1}, destination[:count])

	var content [asn1.CONTENT_SIZE_SHORT_MAXIMUM + 1]byte
	high_tag := asn1.Element_Input{
		Class: asn1.CLASS_CONTEXT_SPECIFIC, Tag: 201, Constructed: true,
		Content: content[:],
	}
	count, encode_status = asn1.Encode_Into(destination[:], high_tag)
	testify.Equal_Values(t, asn1.STATUS_OK, encode_status)
	testify.Equal(t, []byte{0xbf, 0x81, 0x49, 0x81, 0x80}, destination[:5])
}

// Test_Decode protects borrowed content and one-element consumption.
func Test_Decode(t *testing.T) {
	source := asn1.Encoded{0xbf, 0x81, 0x49, 0x81, 0x80}
	source = append(source, make([]byte, asn1.CONTENT_SIZE_SHORT_MAXIMUM+1)...)
	source = append(source, 0)
	element, consumed, position, status := asn1.Decode(source)
	testify.Equal_Values(t, asn1.STATUS_OK, status)
	testify.Equal(t, asn1.Class(asn1.CLASS_CONTEXT_SPECIFIC), element.Class)
	testify.Equal(t, asn1.Tag(201), element.Tag)
	testify.True(t, bool(element.Constructed))
	testify.Equal(t, asn1.Content(source[5:len(source)-1]), element.Content)
	testify.Equal(t, asn1.Consumed_Count(len(source)-1), consumed)
	testify.Equal(t, asn1.Position(0), position)
}

// Test_Bounds protects malicious caller lengths and aggregate growth.
func Test_Bounds(t *testing.T) {
	test_invalid(t)
	var oversized_source [asn1.ENCODED_SIZE_MAXIMUM + 1]byte
	var oversized_output [asn1.ENCODED_SIZE_MAXIMUM + 1]byte
	var oversized_content [asn1.CONTENT_SIZE_MAXIMUM + 1]byte
	testify.Panics(t, func() { asn1.Decode(oversized_source[:]) })
	testify.Panics(t, func() {
		asn1.Encode_Into(oversized_output[:], asn1.Element_Input{})
	})
	testify.Panics(t, func() {
		asn1.Encoded_Size(asn1.Element_Input{Content: oversized_content[:]})
	})
	test_element_refusals(t)
	test_domains(t)
}

// Test_Allocation protects every scalar status path from heap ownership.
func Test_Allocation(t *testing.T) {
	test_size_allocation(t)
	test_encode_allocation(t)
	test_decode_allocation(t)
}

func test_size_allocation(t *testing.T) {
	valid := asn1.Element_Input{Tag: 2, Content: asn1.Content{1}}
	invalid := asn1.Element_Input{Class: asn1.Class_Input(bits.WORD_8_MAXIMUM)}
	large := asn1.Element_Input{
		Tag:     asn1.Tag_Input(asn1.TAG_MAXIMUM),
		Content: make([]byte, asn1.CONTENT_SIZE_MAXIMUM),
	}
	var count asn1.Count
	var status asn1.Size_Status
	testify.Zero_Allocation(t, func() {
		count, status = asn1.Encoded_Size(valid)
	})
	testify.Equal_Values(t, asn1.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		count, status = asn1.Encoded_Size(invalid)
	})
	testify.Equal_Values(t, asn1.STATUS_ELEMENT_INVALID, status)
	testify.Zero_Allocation(t, func() {
		count, status = asn1.Encoded_Size(large)
	})
	testify.Equal_Values(t, asn1.STATUS_ELEMENT_TOO_LARGE, status)
	testify.True(t, count <= asn1.ENCODED_SIZE_MAXIMUM)
}

func test_encode_allocation(t *testing.T) {
	var destination [asn1.ENCODED_SIZE_MAXIMUM]byte
	valid := asn1.Element_Input{Tag: 2, Content: asn1.Content{1}}
	invalid := asn1.Element_Input{Class: asn1.Class_Input(bits.WORD_8_MAXIMUM)}
	large := asn1.Element_Input{
		Tag:     asn1.Tag_Input(asn1.TAG_MAXIMUM),
		Content: make([]byte, asn1.CONTENT_SIZE_MAXIMUM),
	}
	overlap := asn1.Content(destination[:1])
	maximum := asn1.Element_Input{Content: make([]byte, asn1.CONTENT_SIZE_MAXIMUM)}
	var count asn1.Count
	var status asn1.Encode_Status
	testify.Zero_Allocation(t, func() {
		count, status = asn1.Encode_Into(destination[:], valid)
	})
	testify.Equal_Values(t, asn1.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		count, status = asn1.Encode_Into(destination[:], invalid)
	})
	testify.Equal_Values(t, asn1.STATUS_ELEMENT_INVALID, status)
	testify.Zero_Allocation(t, func() {
		count, status = asn1.Encode_Into(destination[:], large)
	})
	testify.Equal_Values(t, asn1.STATUS_ELEMENT_TOO_LARGE, status)
	testify.Zero_Allocation(t, func() {
		count, status = asn1.Encode_Into(destination[:0], valid)
	})
	testify.Equal_Values(t, asn1.STATUS_OUTPUT_TOO_SMALL, status)
	testify.Zero_Allocation(t, func() {
		count, status = asn1.Encode_Into(
			destination[:], asn1.Element_Input{Content: overlap},
		)
	})
	testify.Equal_Values(t, asn1.STATUS_STORAGE_INVALID, status)
	testify.Zero_Allocation(t, func() {
		count, status = asn1.Encode_Into(destination[:], maximum)
	})
	testify.Equal_Values(t, asn1.STATUS_OK, status)
	testify.True(t, count <= asn1.ENCODED_SIZE_MAXIMUM)
}

func test_decode_allocation(t *testing.T) {
	var maximum [asn1.ENCODED_SIZE_MAXIMUM]byte
	maximum[0] = 0
	maximum[1] = 0x82
	maximum[2] = 0x0f
	maximum[3] = 0xfc
	encoded := asn1.Encoded{2, 1, 1}
	malformed := asn1.Encoded{2}
	var position asn1.Position
	var consumed asn1.Consumed_Count
	var element asn1.Element
	var status asn1.Decode_Status
	testify.Zero_Allocation(t, func() {
		element, consumed, position, status = asn1.Decode(encoded)
	})
	testify.Equal_Values(t, asn1.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		element, consumed, position, status = asn1.Decode(malformed)
	})
	testify.Equal_Values(t, asn1.STATUS_INPUT_INVALID, status)
	testify.Zero_Allocation(t, func() {
		element, consumed, position, status = asn1.Decode(maximum[:])
	})
	testify.Equal_Values(t, asn1.STATUS_OK, status)
	testify.True(t, position <= asn1.POSITION_MAXIMUM)
	testify.True(t, consumed <= asn1.ENCODED_SIZE_MAXIMUM)
	testify.True(t, element.Tag <= asn1.Tag(asn1.TAG_MAXIMUM))
}

func test_invalid(t *testing.T) {
	tests := [...]struct {
		Source   asn1.Encoded
		Position asn1.Position
	}{
		{nil, 1},
		{asn1.Encoded{0x1f}, 2},
		{asn1.Encoded{0x1f, 0x80}, 2},
		{asn1.Encoded{0x1f, 1, 0}, 2},
		{asn1.Encoded{0x1f, 2, 0}, 2},
		{asn1.Encoded{0x1f, 0x80, 0}, 2},
		{asn1.Encoded{2, 0x80}, 2},
		{asn1.Encoded{2, 0x81, 1, 0}, 3},
		{asn1.Encoded{2, 2, 0}, 4},
		{asn1.Encoded{0x1f, 0xff, 0xff, 0xff, 0xff, 0x7f, 0}, 6},
		{asn1.Encoded{0x1f, 0x87, 0xff, 0xff, 0xff, 0xff, 0}, 7},
		{asn1.Encoded{0x1f, 0x87, 0xff, 0xff, 0xff, 0x7f, 0x82}, 8},
	}
	for _, one := range tests {
		element, consumed, position, status := asn1.Decode(one.Source)
		testify.Equal_Values(t, asn1.STATUS_INPUT_INVALID, status)
		testify.Equal(t, asn1.Element{}, element)
		testify.Equal(t, asn1.Consumed_Count(0), consumed)
		testify.Equal(t, one.Position, position)
	}
}

func test_element_refusals(t *testing.T) {
	var destination [asn1.ENCODED_SIZE_MAXIMUM]byte
	invalid_class := asn1.Element_Input{Class: asn1.Class_Input(bits.WORD_8_MAXIMUM)}
	_, size_status := asn1.Encoded_Size(invalid_class)
	testify.Equal_Values(t, asn1.STATUS_ELEMENT_INVALID, size_status)
	_, status := asn1.Encode_Into(destination[:], invalid_class)
	testify.Equal_Values(t, asn1.STATUS_ELEMENT_INVALID, status)
	invalid_tag := asn1.Element_Input{Tag: asn1.Tag_Input(bits.WORD_32_MAXIMUM)}
	_, size_status = asn1.Encoded_Size(invalid_tag)
	testify.Equal_Values(t, asn1.STATUS_ELEMENT_INVALID, size_status)
	_, status = asn1.Encode_Into(destination[:], invalid_tag)
	testify.Equal_Values(t, asn1.STATUS_ELEMENT_INVALID, status)
	large := asn1.Element_Input{
		Class: asn1.CLASS_PRIVATE, Tag: asn1.Tag_Input(asn1.TAG_MAXIMUM),
		Content: make([]byte, asn1.CONTENT_SIZE_MAXIMUM),
	}
	_, size_status = asn1.Encoded_Size(large)
	testify.Equal_Values(t, asn1.STATUS_ELEMENT_TOO_LARGE, size_status)
	_, status = asn1.Encode_Into(destination[:], large)
	testify.Equal_Values(t, asn1.STATUS_ELEMENT_TOO_LARGE, status)
	_, status = asn1.Encode_Into(destination[:0], asn1.Element_Input{})
	testify.Equal_Values(t, asn1.STATUS_OUTPUT_TOO_SMALL, status)
	_, status = asn1.Encode_Into(destination[:1], asn1.Element_Input{})
	testify.Equal_Values(t, asn1.STATUS_OUTPUT_TOO_SMALL, status)
	_, status = asn1.Encode_Into(
		destination[:2], asn1.Element_Input{Content: asn1.Content{0}},
	)
	testify.Equal_Values(t, asn1.STATUS_OUTPUT_TOO_SMALL, status)
	storage := asn1.Content(destination[:1])
	_, status = asn1.Encode_Into(destination[:], asn1.Element_Input{Content: storage})
	testify.Equal_Values(t, asn1.STATUS_STORAGE_INVALID, status)
}

func test_domains(t *testing.T) {
	var destination [asn1.ENCODED_SIZE_MAXIMUM]byte
	classes := [...]asn1.Class_Input{0, 1, 2, asn1.Class_Input(bits.WORD_8_MAXIMUM)}
	tags := [...]asn1.Tag_Input{
		0, 1, 2, asn1.Tag_Input(asn1.TAG_MAXIMUM), asn1.Tag_Input(bits.WORD_32_MAXIMUM),
	}
	for _, class := range classes {
		for _, tag := range tags {
			asn1.Encoded_Size(asn1.Element_Input{Class: class, Tag: tag})
		}
	}
	for _, source := range [...]asn1.Encoded{
		nil, {0, 0}, {1, 0}, {2, 0}, {0x40, 0}, {0x80, 0}, {0xc0, 0},
		{2, 2, 0, 0}, {0x1f, 0x87, 0xff, 0xff, 0xff, 0x7f, 0},
		{0, 0x82, 0x0f, 0xfc},
	} {
		asn1.Decode(source)
	}
	for _, element := range [...]asn1.Element_Input{
		{Class: asn1.CLASS_APPLICATION},
		{Class: asn1.CLASS_PRIVATE},
		{Tag: 1},
		{Tag: asn1.HIGH_TAG_MARKER},
		{Tag: asn1.Tag_Input(asn1.TAG_MAXIMUM)},
		{Content: asn1.Content{0, 0}},
	} {
		asn1.Encode_Into(destination[:], element)
	}
	const SIZE_FIELD_THREE_CONTENT_SIZE = 1 << bits.BIT_COUNT_8_MAXIMUM
	var header_maximum [asn1.IDENTIFIER_SIZE_MAXIMUM +
		asn1.CONTENT_SIZE_FIELD_SIZE_MAXIMUM + SIZE_FIELD_THREE_CONTENT_SIZE]byte
	copy(header_maximum[:], []byte{
		asn1.HIGH_TAG_MARKER, 0x87, 0xff, 0xff, 0xff, 0x7f,
		0x82, 0x01, 0x00,
	})
	_, _, _, header_status := asn1.Decode(header_maximum[:])
	testify.Equal_Values(t, asn1.STATUS_OK, header_status)
	var invalid_maximum [asn1.ENCODED_SIZE_MAXIMUM]byte
	invalid_maximum[0] = asn1.HIGH_TAG_MARKER
	invalid_maximum[1] = asn1.HIGH_TAG_MARKER
	invalid_maximum[2] = 0x82
	invalid_maximum[3] = 0x0f
	invalid_maximum[4] = 0xfc
	_, _, position, decode_status := asn1.Decode(invalid_maximum[:])
	testify.Equal(t, asn1.Position(asn1.POSITION_MAXIMUM), position)
	testify.Equal_Values(t, asn1.STATUS_INPUT_INVALID, decode_status)
	maximum := asn1.Element_Input{Content: make([]byte, asn1.CONTENT_SIZE_MAXIMUM)}
	count, status := asn1.Encode_Into(destination[:], maximum)
	testify.Equal(t, asn1.Count(asn1.ENCODED_SIZE_MAXIMUM), count)
	testify.Equal_Values(t, asn1.STATUS_OK, status)
	decoded, consumed, _, decode_status := asn1.Decode(destination[:count])
	testify.Equal_Values(t, asn1.STATUS_OK, decode_status)
	testify.Equal(t, asn1.Content(maximum.Content), decoded.Content)
	testify.Equal(t, asn1.Consumed_Count(count), consumed)
}
