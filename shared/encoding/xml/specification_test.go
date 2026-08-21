package xml_test

import (
	"testing"

	"local/james-orcales/shared/encoding/xml"
	"local/james-orcales/shared/testify"
)

// Test_Validate protects document structure instead of accepting lexical fragments.
func Test_Validate(t *testing.T) {
	valid := [...]xml.Document{
		xml.Document(`<root a="&amp;"><child><![CDATA[x<y]]></child><!-- ok --><?pi x?></root>`),
		xml.Document(`<a/>`),
		xml.Document(`<世界/>`),
		xml.Document("<?pi?>\n<a>€&#x20AC;</a>"),
		xml.Document("<!DOCTYPE root><root/>"),
		xml.Document("<!DOCTYPE root [<!ELEMENT root ANY><!-- x -->]><root/>"),
		xml.Document("<!DOCTYPE root [<!-- x--->]><root/>"),
		xml.Document("<root><!ENTITY x 'y'></root>"),
		xml.Document("<!<- x --><root/>"),
		xml.Document("<?xml version='1.0' encoding='UtF-8'?><root/>"),
		xml.Document("<?xml?><root/>"),
		xml.Document("<?xml version='' encoding=''?><root/>"),
		xml.Document("<:root/>"),
		xml.Document("<root:/>"),
		xml.Document("<!--\x00--><root/>"),
		xml.Document("<?pi \xff?><root/>"),
		xml.Document("<?p\x00 x?><root/>"),
		xml.Document("<!D'\x00'><root/>"),
	}
	for _, source := range valid {
		position, status := xml.Validate(source)
		testify.Equal(t, xml.Position(0), position)
		testify.Equal_Values(t, xml.STATUS_OK, status)
	}
	invalid := [...]struct {
		Source xml.Document
	}{
		{nil},
		{xml.Document(`text`)},
		{xml.Document(`<a></b>`)},
		{xml.Document(`<a>`)},
		{xml.Document(`<a>&bad;</a>`)},
		{xml.Document(`<a x=y/>`)},
		{xml.Document(`<a/><b/>`)},
		{xml.Document(`<a x="1" x="2"/>`)},
		{xml.Document(`<a>]]></a>`)},
		{xml.Document(`<!-- -- --><a/>`)},
		{xml.Document(`<??><a/>`)},
		{xml.Document(`<a></>`)},
		{xml.Document(`<a><![CDATA[x</a>`)},
		{xml.Document("<\U000F0000/>")},
		{xml.Document("<!DOCTYPE root")},
		{xml.Document("<!DOCTYPE root [<!-- x ]><root/>")},
		{xml.Document("<![- x --><root/>")},
		{xml.Document("<!- x><root/>")},
		{xml.Document("<root:name:tail/>")},
		{xml.Document("<?xml version='1.1'?><root/>")},
		{xml.Document("<?xml encoding='ISO-8859-1'?><root/>")},
		{xml.Document{'<', 'a', '>', 0xff, '<', '/', 'a', '>'}},
	}
	for _, one := range invalid {
		position, status := xml.Validate(one.Source)
		testify.True(t, position > 0)
		testify.Equal_Values(t, xml.STATUS_INPUT_INVALID, status)
	}
}

// Test_Escape protects exact standard text replacements.
func Test_Escape(t *testing.T) {
	source := xml.Text("<&>\"'\t\n\r")
	expected := "&lt;&amp;&gt;&#34;&#39;&#x9;&#xA;&#xD;"
	var destination [xml.OUTPUT_SIZE_MAXIMUM]byte
	count, status := xml.Escape_Size(source)
	testify.Equal(t, xml.Count(len(expected)), count)
	testify.Equal_Values(t, xml.STATUS_OK, status)
	count, encode_status := xml.Escape_Into(destination[:count], source)
	testify.Equal_Values(t, xml.STATUS_OK, encode_status)
	testify.Equal(t, expected, string(destination[:count]))
	invalid_utf8 := xml.Text{0xff}
	count, encode_status = xml.Escape_Into(destination[:], invalid_utf8)
	testify.Equal_Values(t, xml.STATUS_OK, encode_status)
	testify.Equal(t, "�", string(destination[:count]))
}

// Test_Unescape protects predefined, decimal, hexadecimal, and UTF-8 references.
func Test_Unescape(t *testing.T) {
	source := xml.Escaped("&lt;&amp;&#65;&#x20AC;")
	expected := "<&A€"
	var destination [xml.OUTPUT_SIZE_MAXIMUM]byte
	count, position, status := xml.Unescape_Size(source)
	testify.Equal(t, xml.Count(len(expected)), count)
	testify.Equal(t, xml.Position(0), position)
	testify.Equal_Values(t, xml.STATUS_OK, status)
	count, position, decode_status := xml.Unescape_Into(destination[:count], source)
	testify.Equal(t, xml.Position(0), position)
	testify.Equal_Values(t, xml.STATUS_OK, decode_status)
	testify.Equal(t, expected, string(destination[:count]))
}

// Test_Bounds protects malicious sizes, nesting, entities, and overlap.
func Test_Bounds(t *testing.T) {
	test_invalid_entities(t)
	test_refusals(t)
	test_domains(t)
}

// Test_Allocation protects every status and maximum-size path from heap ownership.
func Test_Allocation(t *testing.T) {
	test_validate_allocation(t)
	test_escape_allocation(t)
	test_unescape_allocation(t)
}

func test_invalid_entities(t *testing.T) {
	invalid := [...]xml.Escaped{
		xml.Escaped(`&bad;`),
		xml.Escaped(`&#;`),
		xml.Escaped(`&#x;`),
		xml.Escaped(`&#0;`),
		xml.Escaped(`&#xD800;`),
		xml.Escaped(`&amp`),
		xml.Escaped(`a&bad;`),
	}
	var destination [xml.OUTPUT_SIZE_MAXIMUM]byte
	for _, source := range invalid {
		count, position, status := xml.Unescape_Size(source)
		testify.Equal(t, xml.Count(0), count)
		testify.True(t, position > 0)
		testify.Equal_Values(t, xml.STATUS_INPUT_INVALID, status)
		count, position, decode_status := xml.Unescape_Into(destination[:], source)
		testify.Equal(t, xml.Count(0), count)
		testify.True(t, position > 0)
		testify.Equal_Values(t, xml.STATUS_INPUT_INVALID, decode_status)
	}
}

func test_refusals(t *testing.T) {
	var oversized [xml.TEXT_SIZE_MAXIMUM + 1]byte
	var oversized_output [xml.OUTPUT_SIZE_MAXIMUM + 1]byte
	testify.Panics(t, func() { xml.Validate(oversized[:]) })
	testify.Panics(t, func() { xml.Escape_Size(oversized[:]) })
	testify.Panics(t, func() { xml.Escape_Into(oversized_output[:], nil) })
	var destination [xml.OUTPUT_SIZE_MAXIMUM]byte
	var expanding_source [xml.TEXT_SIZE_MAXIMUM]byte
	growth := xml.Text(expanding_source[:])
	for index := range growth {
		growth[index] = '&'
	}
	count, status := xml.Escape_Size(growth)
	testify.Equal(t, xml.Count(0), count)
	testify.Equal_Values(t, xml.STATUS_OUTPUT_TOO_LARGE, status)
	_, encode_status := xml.Escape_Into(destination[:], growth)
	testify.Equal_Values(t, xml.STATUS_OUTPUT_TOO_LARGE, encode_status)
	overlap := xml.Text(destination[:])
	_, encode_status = xml.Escape_Into(destination[:], overlap)
	testify.Equal_Values(t, xml.STATUS_STORAGE_INVALID, encode_status)
	_, encode_status = xml.Escape_Into(destination[:0], xml.Text("a"))
	testify.Equal_Values(t, xml.STATUS_OUTPUT_TOO_SMALL, encode_status)
	_, encode_status = xml.Escape_Into(destination[:1], xml.Text("a"))
	testify.Equal_Values(t, xml.STATUS_OK, encode_status)
	_, encode_status = xml.Escape_Into(destination[:2], xml.Text("ab"))
	testify.Equal_Values(t, xml.STATUS_OK, encode_status)
	_, _, decode_status := xml.Unescape_Into(destination[:0], xml.Escaped("a"))
	testify.Equal_Values(t, xml.STATUS_OUTPUT_TOO_SMALL, decode_status)
	_, _, decode_status = xml.Unescape_Into(destination[:], xml.Escaped(destination[:1]))
	testify.Equal_Values(t, xml.STATUS_STORAGE_INVALID, decode_status)
	_, _, decode_status = xml.Unescape_Into(destination[:2], xml.Escaped("ab"))
	testify.Equal_Values(t, xml.STATUS_OK, decode_status)
}

func test_domains(t *testing.T) {
	var destination [xml.OUTPUT_SIZE_MAXIMUM]byte
	for _, source := range [...]xml.Text{nil, {'a'}, {'a', 'b'}, xml.Text("¢")} {
		xml.Escape_Into(destination[:], source)
	}
	for _, source := range [...]xml.Escaped{
		nil, {'a'}, {'a', 'b'}, xml.Escaped(`&#x80;`), xml.Escaped(`&#x10000;`),
	} {
		xml.Unescape_Into(destination[:], source)
	}
	maximum := xml.Text(make([]byte, xml.TEXT_SIZE_MAXIMUM))
	for index := range maximum {
		maximum[index] = 'a'
	}
	count, status := xml.Escape_Into(destination[:], maximum)
	testify.Equal(t, xml.Count(xml.OUTPUT_SIZE_MAXIMUM), count)
	testify.Equal_Values(t, xml.STATUS_OK, status)
	count, _, decode_status := xml.Unescape_Into(destination[:], xml.Escaped(maximum))
	testify.Equal(t, xml.Count(xml.OUTPUT_SIZE_MAXIMUM), count)
	testify.Equal_Values(t, xml.STATUS_OK, decode_status)
	var invalid_entity_maximum [xml.TEXT_SIZE_MAXIMUM]byte
	for index := range len(invalid_entity_maximum) - 1 {
		invalid_entity_maximum[index] = 'a'
	}
	invalid_entity_maximum[len(invalid_entity_maximum)-1] = '&'
	_, error_position, size_status := xml.Unescape_Size(invalid_entity_maximum[:])
	testify.Equal(t, xml.Position(xml.POSITION_MAXIMUM), error_position)
	testify.Equal_Values(t, xml.STATUS_INPUT_INVALID, size_status)
	_, error_position, unescape_status := xml.Unescape_Into(
		destination[:], invalid_entity_maximum[:],
	)
	testify.Equal(t, xml.Position(xml.POSITION_MAXIMUM), error_position)
	testify.Equal_Values(t, xml.STATUS_INPUT_INVALID, unescape_status)

	for _, source := range [...]xml.Document{nil, {'<'}, {'<', '>'}} {
		xml.Validate(source)
	}
	var invalid_document_maximum [xml.DOCUMENT_SIZE_MAXIMUM]byte
	copy(invalid_document_maximum[:], "<a>")
	for fill_count := len("<a>"); fill_count < len(invalid_document_maximum); fill_count++ {
		invalid_document_maximum[fill_count] = 'a'
	}
	validate_position, validate_status := xml.Validate(invalid_document_maximum[:])
	testify.Equal(t, xml.Position(xml.POSITION_MAXIMUM), validate_position)
	testify.Equal_Values(t, xml.STATUS_INPUT_INVALID, validate_status)

	var nested [xml.DEPTH_MAXIMUM*len("<a>") + xml.DEPTH_MAXIMUM*len("</a>")]byte
	position := 0
	for range xml.DEPTH_MAXIMUM {
		position += copy(nested[position:], "<a>")
	}
	for range xml.DEPTH_MAXIMUM {
		position += copy(nested[position:], "</a>")
	}
	validate_position, validate_status = xml.Validate(nested[:])
	testify.Equal(t, xml.Position(0), validate_position)
	testify.Equal_Values(t, xml.STATUS_OK, validate_status)
}

func test_validate_allocation(t *testing.T) {
	var nested [xml.DEPTH_MAXIMUM*len("<a>") + xml.DEPTH_MAXIMUM*len("</a>")]byte
	nested_count := 0
	for range xml.DEPTH_MAXIMUM {
		nested_count += copy(nested[nested_count:], "<a>")
	}
	for range xml.DEPTH_MAXIMUM {
		nested_count += copy(nested[nested_count:], "</a>")
	}
	valid := xml.Document(`<a>text</a>`)
	invalid := xml.Document(`<a>`)
	var position xml.Position
	var status xml.Validate_Status
	testify.Zero_Allocation(t, func() {
		position, status = xml.Validate(valid)
	})
	testify.Equal_Values(t, xml.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		position, status = xml.Validate(invalid)
	})
	testify.Equal_Values(t, xml.STATUS_INPUT_INVALID, status)
	testify.Zero_Allocation(t, func() {
		position, status = xml.Validate(nested[:])
	})
	testify.Equal_Values(t, xml.STATUS_OK, status)
	testify.True(t, position <= xml.POSITION_MAXIMUM)
}

func test_escape_allocation(t *testing.T) {
	var destination [xml.OUTPUT_SIZE_MAXIMUM]byte
	var growth_source [xml.TEXT_SIZE_MAXIMUM]byte
	var maximum_source [xml.TEXT_SIZE_MAXIMUM]byte
	plain := xml.Text("text")
	growth := xml.Text(growth_source[:])
	for index := range growth {
		growth[index] = '&'
	}
	maximum := xml.Text(maximum_source[:])
	for index := range maximum {
		maximum[index] = 'a'
	}
	overlap := xml.Text(destination[:])
	var count xml.Count
	var size_status xml.Escape_Size_Status
	var status xml.Escape_Status
	testify.Zero_Allocation(t, func() {
		count, size_status = xml.Escape_Size(plain)
	})
	testify.Equal_Values(t, xml.STATUS_OK, size_status)
	testify.Zero_Allocation(t, func() {
		count, size_status = xml.Escape_Size(growth)
	})
	testify.Equal_Values(t, xml.STATUS_OUTPUT_TOO_LARGE, size_status)
	testify.Zero_Allocation(t, func() {
		count, status = xml.Escape_Into(destination[:], plain)
	})
	testify.Equal_Values(t, xml.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		count, status = xml.Escape_Into(destination[:0], plain)
	})
	testify.Equal_Values(t, xml.STATUS_OUTPUT_TOO_SMALL, status)
	testify.Zero_Allocation(t, func() {
		count, status = xml.Escape_Into(destination[:], growth)
	})
	testify.Equal_Values(t, xml.STATUS_OUTPUT_TOO_LARGE, status)
	testify.Zero_Allocation(t, func() {
		count, status = xml.Escape_Into(destination[:], overlap)
	})
	testify.Equal_Values(t, xml.STATUS_STORAGE_INVALID, status)
	testify.Zero_Allocation(t, func() {
		count, status = xml.Escape_Into(destination[:], maximum)
	})
	testify.Equal_Values(t, xml.STATUS_OK, status)
	testify.True(t, count <= xml.OUTPUT_SIZE_MAXIMUM)
}

func test_unescape_allocation(t *testing.T) {
	var destination [xml.OUTPUT_SIZE_MAXIMUM]byte
	var maximum_source [xml.TEXT_SIZE_MAXIMUM]byte
	for index := range maximum_source {
		maximum_source[index] = 'a'
	}
	maximum := xml.Escaped(maximum_source[:])
	valid := xml.Escaped("&amp;")
	invalid := xml.Escaped("&bad;")
	overlap := xml.Escaped(destination[:1])
	var count xml.Count
	var position xml.Position
	var size_status xml.Unescape_Size_Status
	var status xml.Unescape_Status
	testify.Zero_Allocation(t, func() {
		count, position, size_status = xml.Unescape_Size(valid)
	})
	testify.Equal_Values(t, xml.STATUS_OK, size_status)
	testify.Zero_Allocation(t, func() {
		count, position, size_status = xml.Unescape_Size(invalid)
	})
	testify.Equal_Values(t, xml.STATUS_INPUT_INVALID, size_status)
	testify.Zero_Allocation(t, func() {
		count, position, status = xml.Unescape_Into(destination[:], valid)
	})
	testify.Equal_Values(t, xml.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		count, position, status = xml.Unescape_Into(destination[:], invalid)
	})
	testify.Equal_Values(t, xml.STATUS_INPUT_INVALID, status)
	testify.Zero_Allocation(t, func() {
		count, position, status = xml.Unescape_Into(destination[:0], valid)
	})
	testify.Equal_Values(t, xml.STATUS_OUTPUT_TOO_SMALL, status)
	testify.Zero_Allocation(t, func() {
		count, position, status = xml.Unescape_Into(destination[:1], overlap)
	})
	testify.Equal_Values(t, xml.STATUS_STORAGE_INVALID, status)
	testify.Zero_Allocation(t, func() {
		count, position, status = xml.Unescape_Into(destination[:], maximum)
	})
	testify.Equal_Values(t, xml.STATUS_OK, status)
	testify.True(t, count <= xml.OUTPUT_SIZE_MAXIMUM)
	testify.True(t, position <= xml.POSITION_MAXIMUM)
}
