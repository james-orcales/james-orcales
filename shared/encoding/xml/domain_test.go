package xml

import (
	"testing"

	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/unicode/utf8"
)

// Test_Internal_Character_Boundaries reaches scalar XML boundaries through production code.
func Test_Internal_Character_Boundaries(_ *testing.T) {
	characters := [...]XML_Character{0, 1, 2, XML_Character(utf8.RUNE_MAX)}
	sizes := [...]Character_Size{
		utf8.CHARACTER_SIZE_MINIMUM, utf8.CHARACTER_SIZE_TWO,
		utf8.CHARACTER_SIZE_THREE, utf8.CHARACTER_SIZE_MAXIMUM,
	}
	var source_storage [TEXT_SIZE_MAXIMUM]byte
	var output_storage [OUTPUT_SIZE_MAXIMUM]byte
	source := Nonempty_Text(source_storage[:])
	output := Nonempty_Output(output_storage[:])
	for _, character := range characters {
		xml_character(character)
		name_character(character)
		name_start_character(character)
		name_start_character_lower(character)
		name_start_character_upper(character)
		for _, size := range sizes {
			decoded_xml_character(character, size)
			escaped_character_size(character, size)
			write_escaped_character(output, source, 0, 0, character, size)
		}
	}
	for _, value := range [...]XML_Byte{0, 1, 2, XML_Byte(bits.WORD_8_MAXIMUM)} {
		xml_space(value)
	}
}

// Test_Internal_Position_Boundaries reaches XML position boundaries through production code.
func Test_Internal_Position_Boundaries(_ *testing.T) {
	var document_storage [DOCUMENT_SIZE_MAXIMUM]byte
	var output_storage [OUTPUT_SIZE_MAXIMUM]byte
	document := Nonempty_Document(document_storage[:])
	space_document := Space_Document(document_storage[:])
	output := Nonempty_Output(output_storage[:])
	for _, position := range [...]Document_Position{0, 1, 2, DOCUMENT_SIZE_MAXIMUM} {
		prefix_at(document, position, "<?")
		skip_xml_space(space_document, position)
	}
	for _, semicolon := range [...]Document_Index{0, 1, 2, DOCUMENT_INDEX_MAXIMUM} {
		for _, character := range [...]XML_Character{
			0, 1, 2, XML_Character(utf8.RUNE_MAX),
		} {
			write_entity(output, character, semicolon)
		}
	}
	for _, index := range [...]Document_Index{0, 1, 2, DOCUMENT_INDEX_MAXIMUM} {
		write_escaped_character(
			output, Nonempty_Text(document_storage[:]), index, index, 0, 1,
		)
	}
	for _, pattern := range [...]Pattern{"<?", "xml", "true", "<![CDATA["} {
		text_equal(Entity_Name(pattern), pattern)
	}
}

// Test_Internal_Scan_Position_Boundaries drives legal scan edges.
func Test_Internal_Scan_Position_Boundaries(_ *testing.T) {
	var document_storage [DOCUMENT_SIZE_MAXIMUM]byte
	document := Nonempty_Document(document_storage[:])
	positions := [...]Document_Position{0, 1, 2, DOCUMENT_SIZE_MAXIMUM}
	var name_start_storage [DEPTH_MAXIMUM]int
	var name_end_storage [DEPTH_MAXIMUM]int
	name_starts := Name_Starts(name_start_storage[:])
	name_ends := Name_Ends(name_end_storage[:])
	for _, position := range positions {
		scan_document_text(document, position, 0)
		scan_document_text(document, position, 2)
		scan_document_text(document, position, Depth(DEPTH_MAXIMUM))
		scan_document_special(document, position, 0)
		scan_text(document, position)
		scan_start_tag(document, position)
		scan_attribute(Attribute_Document(document), position)
		scan_end_tag(End_Tag_Document(document), position)
		scan_name(document, position)
		scan_qualified_name(document, position)
		scan_comment(Comment_Document(document), position)
		scan_instruction(Special_Document(document), position)
		scan_cdata(CDATA_Document(document), position)
		scan_directive(Special_Document(document), position)
		scan_directive_comment(Directive_Comment_Document(document), position)
		parse_entity(document, Nonempty_Output(document_storage[:]), position)
		instruction_parameter(
			XML_Instruction_Document(document), position, position, "<?",
		)
		xml_instruction_valid(
			Terminated_Instruction_Document(document),
			position, position, position,
		)
		depth := Depth(0)
		close_document_element(
			Special_Document(document), position, name_starts, name_ends,
			Depth_Handle(&depth),
		)
		open_document_element(
			document, position, name_starts, name_ends, Depth_Handle(&depth),
		)
	}
	for _, position := range [...]Start_Tag_Position{
		START_TAG_POSITION_MINIMUM, 2, DOCUMENT_SIZE_MAXIMUM,
	} {
		scan_start_tag_attributes(document[:position], position)
	}
}

// Test_Internal_Entity_Position_Boundaries drives numeric entity edges.
func Test_Internal_Entity_Position_Boundaries(_ *testing.T) {
	var document_storage [DOCUMENT_SIZE_MAXIMUM]byte
	var output_storage [OUTPUT_SIZE_MAXIMUM]byte
	document := Numeric_Entity_Document(document_storage[:])
	output := Nonempty_Output(output_storage[:])
	positions := [...]Document_Position{0, 1, 2, DOCUMENT_SIZE_MAXIMUM}
	for _, start := range positions {
		parse_numeric_entity(document, output, start, start)
	}
	for _, semicolon := range positions {
		parse_numeric_entity(document, output, semicolon, semicolon)
	}
}
