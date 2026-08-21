package tabwriter_test

import (
	"testing"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
	"local/james-orcales/shared/text/tabwriter"
	"local/james-orcales/shared/unicode/utf8"
)

// Test_Configuration rejects policy before bounded work can multiply.
func Test_Configuration(t *testing.T) {
	configuration := configuration_from(t, tabwriter.Configuration_Input{
		Minimum_Width: TEST_WIDTH_EIGHT,
		Tab_Width:     TEST_WIDTH_FOUR,
		Padding:       TEST_WIDTH_ONE,
		Pad_Character: '.',
		Flags:         tabwriter.ALIGN_RIGHT,
	})
	testify.Equal(t, ".......ab", format_configuration(t, "a\tb", configuration))

	for _, input := range [...]tabwriter.Configuration_Input{
		{Minimum_Width: TEST_WIDTH_NEGATIVE_ONE},
		{Tab_Width: TEST_WIDTH_NEGATIVE_ONE},
		{Padding: TEST_WIDTH_NEGATIVE_ONE},
		{Minimum_Width: tabwriter.WIDTH_MAXIMUM + TEST_WIDTH_ONE},
		{Tab_Width: tabwriter.WIDTH_MAXIMUM + TEST_WIDTH_ONE},
		{Padding: tabwriter.WIDTH_MAXIMUM + TEST_WIDTH_ONE},
		{Flags: tabwriter.FLAGS_MAXIMUM + tabwriter.FILTER_HTML},
	} {
		_, status := tabwriter.New_Configuration(input)
		testify.Equal_Values(t, tabwriter.STATUS_CONFIGURATION_INVALID, status)
	}
	corrupt := configuration
	corrupt.Minimum_Width = tabwriter.Minimum_Width(TEST_WIDTH_NEGATIVE_ONE)
	var output [tabwriter.OUTPUT_SIZE_MAXIMUM]byte
	workspace := workspace_new()
	_, status := tabwriter.Format_Into(
		output[:], source_from(t, "x"), corrupt, workspace_from(&workspace),
	)
	testify.Equal_Values(t, tabwriter.STATUS_CONFIGURATION_INVALID, status)
}

// Test_Source rejects oversize before workspace mutation.
func Test_Source(t *testing.T) {
	var maximum [tabwriter.SOURCE_SIZE_MAXIMUM]byte
	source, status := tabwriter.Source_Validate(maximum[:])
	testify.Equal_Values(t, tabwriter.STATUS_OK, status)
	testify.Equal(t, len(maximum), len(source))

	var oversized [tabwriter.SOURCE_SIZE_UNVALIDATED_MAXIMUM]byte
	source, status = tabwriter.Source_Validate(oversized[:])
	testify.Equal_Values(t, tabwriter.STATUS_INPUT_INVALID, status)
	testify.Equal(t, tabwriter.SOURCE_SIZE_MINIMUM, len(source))
}

// Test_Format preserves elastic column semantics without a retained writer.
func Test_Format(t *testing.T) {
	input := tabwriter.Configuration_Input{
		Minimum_Width: TEST_WIDTH_EIGHT,
		Padding:       TEST_WIDTH_ONE,
		Pad_Character: '.',
	}
	testify.Equal(t, "*.......*", format_text(t, "*\t*", input))
	testify.Equal(t, "a.......b\naa......bbb\n", format_text(
		t,
		"a\tb\naa\tbbb\n",
		input,
	))
	testify.Equal(t, ".......aè\n......aaèèè\n", format_text(
		t,
		"a\tè\naa\tèèè\n",
		tabwriter.Configuration_Input{
			Minimum_Width: TEST_WIDTH_EIGHT,
			Padding:       TEST_WIDTH_ONE,
			Pad_Character: '.',
			Flags:         tabwriter.ALIGN_RIGHT,
		},
	))
}

// Test_Escapes keeps structural bytes inert inside escape domains.
func Test_Escapes(t *testing.T) {
	testify.Equal(t, "a\xff\t\xffb", format_text(
		t,
		"a\xff\t\xffb",
		tabwriter.Configuration_Input{Pad_Character: ' '},
	))
	testify.Equal(t, "a\tb", format_text(
		t,
		"a\xff\t\xffb",
		tabwriter.Configuration_Input{
			Pad_Character: ' ',
			Flags:         tabwriter.STRIP_ESCAPE,
		},
	))
	testify.Equal(t, "f&lt;o.....<b>bar</b>.....\n", format_text(
		t,
		"f&lt;o\t<b>bar</b>\t\n",
		tabwriter.Configuration_Input{
			Minimum_Width: TEST_WIDTH_EIGHT,
			Padding:       TEST_WIDTH_ONE,
			Pad_Character: '.',
			Flags:         tabwriter.FILTER_HTML,
		},
	))
}

// Test_Columns keeps hard, soft, and section boundaries distinct.
func Test_Columns(t *testing.T) {
	testify.Equal(t, "1|2|3\n---\n11|22|33\n", format_text(
		t,
		"1\t2\t3\f11\t22\t33\n",
		tabwriter.Configuration_Input{
			Minimum_Width: TEST_WIDTH_ONE,
			Pad_Character: '.',
			Flags:         tabwriter.DEBUG,
		},
	))
	testify.Equal(t, "\ta b\n", format_text(
		t,
		"\v\ta\tb\n",
		tabwriter.Configuration_Input{
			Minimum_Width: TEST_WIDTH_ONE,
			Tab_Width:     TEST_WIDTH_FOUR,
			Padding:       TEST_WIDTH_ONE,
			Pad_Character: ' ',
			Flags: tabwriter.DISCARD_EMPTY_COLUMNS |
				tabwriter.TAB_INDENT,
		},
	))
}

// Test_Bounds leaves caller output unchanged for every refused write.
func Test_Bounds(t *testing.T) {
	test_formulas(t)
	test_scalar_domains(t)
	test_output_domains(t)
	test_maximum_domains(t)
	test_escape_domains(t)

	configuration := configuration_from(t, tabwriter.Configuration_Input{
		Minimum_Width: TEST_WIDTH_EIGHT,
		Padding:       TEST_WIDTH_ONE,
		Pad_Character: '.',
	})
	source := source_from(t, "a\tb")
	workspace := workspace_new()
	destination := [...]byte{'x'}
	count, status := tabwriter.Format_Into(
		destination[:], source, configuration, workspace_from(&workspace),
	)
	testify.Equal_Values(t, tabwriter.STATUS_OUTPUT_TOO_SMALL, status)
	testify.Equal(t, tabwriter.Output_Count(TEST_WIDTH_NINE), count)
	testify.Equal(t, byte('x'), destination[tabwriter.OUTPUT_SIZE_MINIMUM])

	large_configuration := configuration_from(t, tabwriter.Configuration_Input{
		Minimum_Width: tabwriter.WIDTH_MAXIMUM,
		Pad_Character: '.',
	})
	var output [tabwriter.OUTPUT_SIZE_MAXIMUM]byte
	count, status = tabwriter.Format_Into(
		output[:], source, large_configuration, workspace_from(&workspace),
	)
	testify.Equal_Values(t, tabwriter.STATUS_RESULT_TOO_LARGE, status)
	testify.Equal(t, tabwriter.Output_Count(tabwriter.OUTPUT_SIZE_UNREPRESENTABLE), count)

	var overlap [tabwriter.OUTPUT_SIZE_MAXIMUM]byte
	copy(overlap[:], source)
	count, status = tabwriter.Format_Into(
		overlap[:], tabwriter.Source(overlap[:len(source)]), configuration,
		workspace_from(&workspace),
	)
	testify.Equal_Values(t, tabwriter.STATUS_STORAGE_INVALID, status)
	testify.Equal(t, tabwriter.Output_Count(tabwriter.OUTPUT_SIZE_MINIMUM), count)

	count, status = tabwriter.Format_Into(
		output[:], source, configuration, tabwriter.Workspace_Input{},
	)
	testify.Equal_Values(t, tabwriter.STATUS_WORKSPACE_INVALID, status)
	testify.Equal(t, tabwriter.Output_Count(tabwriter.OUTPUT_SIZE_MINIMUM), count)

	var malformed_workspace tabwriter.Workspace
	count, status = tabwriter.Format_Into(
		output[:], source, configuration, workspace_from(&malformed_workspace),
	)
	testify.Equal_Values(t, tabwriter.STATUS_WORKSPACE_INVALID, status)
	testify.Equal(t, tabwriter.Output_Count(tabwriter.OUTPUT_SIZE_MINIMUM), count)
}

// Test_Allocation measures assertions and formatting on production paths.
func Test_Allocation(t *testing.T) {
	input := tabwriter.Configuration_Input{
		Minimum_Width: TEST_WIDTH_ONE,
		Tab_Width:     TEST_WIDTH_EIGHT,
		Padding:       TEST_WIDTH_ONE,
		Pad_Character: '\t',
		Flags:         tabwriter.FILTER_HTML | tabwriter.STRIP_ESCAPE,
	}
	var configuration tabwriter.Configuration
	var configuration_status tabwriter.Configuration_Status
	testify.Zero_Allocation(t, func() {
		configuration, configuration_status = tabwriter.New_Configuration(input)
	})
	testify.Equal_Values(t, tabwriter.STATUS_OK, configuration_status)
	var configuration_valid tabwriter.Configuration_Validity
	testify.Zero_Allocation(t, func() {
		configuration_valid = tabwriter.Configuration_Valid(configuration)
	})
	testify.True(t, bool(configuration_valid))

	unvalidated := tabwriter.Source_Unvalidated("a\t<b>è</b>\n")
	var source tabwriter.Source
	var source_status tabwriter.Source_Status
	testify.Zero_Allocation(t, func() {
		source, source_status = tabwriter.Source_Validate(unvalidated)
	})
	testify.Equal_Values(t, tabwriter.STATUS_OK, source_status)

	var output [tabwriter.OUTPUT_SIZE_MAXIMUM]byte
	workspace := workspace_new()
	workspace_storage := workspace_from(&workspace).State
	var workspace_valid tabwriter.Workspace_Validity
	testify.Zero_Allocation(t, func() {
		workspace_valid = tabwriter.Workspace_Valid(workspace_storage)
	})
	testify.True(t, bool(workspace_valid))
	var count tabwriter.Output_Count
	var status tabwriter.Format_Status
	testify.Zero_Allocation(t, func() {
		count, status = tabwriter.Format_Into(
			output[:], source, configuration, workspace_from(&workspace),
		)
	})
	testify.Equal_Values(t, tabwriter.STATUS_OK, status)
	testify.True(t, count > tabwriter.Output_Count(tabwriter.OUTPUT_SIZE_MINIMUM))
	test_format_branch_allocation(t)
}

const TEST_WIDTH_ZERO = tabwriter.WIDTH_MINIMUM
const TEST_WIDTH_ONE = TEST_WIDTH_ZERO + utf8.CHARACTER_SIZE_MINIMUM
const TEST_WIDTH_TWO = TEST_WIDTH_ONE + TEST_WIDTH_ONE
const TEST_WIDTH_FOUR = TEST_WIDTH_TWO * TEST_WIDTH_TWO
const TEST_WIDTH_EIGHT = TEST_WIDTH_FOUR * TEST_WIDTH_TWO
const TEST_WIDTH_NINE = TEST_WIDTH_EIGHT + TEST_WIDTH_ONE
const TEST_WIDTH_NEGATIVE_ONE = TEST_WIDTH_ZERO - TEST_WIDTH_ONE
const TEST_FLAGS_NONE = tabwriter.Flags(bits.WORD_MINIMUM)

// Formatting formulas must stay attached to source, output, and UTF-8 units.
func test_formulas(t *testing.T) {
	testify.Equal(t, bytes.SLICE_SIZE_MAXIMUM, tabwriter.SOURCE_SIZE_MAXIMUM)
	testify.Equal(t, bytes.SLICE_SIZE_MAXIMUM, tabwriter.OUTPUT_SIZE_MAXIMUM)
	testify.Equal(t,
		tabwriter.SOURCE_SIZE_MAXIMUM+utf8.CHARACTER_SIZE_MINIMUM,
		tabwriter.LINE_COUNT_MAXIMUM,
	)
	testify.Equal(t, utf8.CHARACTER_SIZE_TWO, tabwriter.ALIGNED_SOURCE_OVERHEAD)
	testify.Equal(t,
		tabwriter.SOURCE_SIZE_MAXIMUM-tabwriter.ALIGNED_SOURCE_OVERHEAD,
		tabwriter.ALIGNED_CELL_WIDTH_MAXIMUM,
	)
	testify.Equal_Values(t, tabwriter.FILTER_HTML<<utf8.CHARACTER_SIZE_MINIMUM,
		tabwriter.STRIP_ESCAPE)
	testify.Equal_Values(t, tabwriter.TAB_INDENT<<utf8.CHARACTER_SIZE_MINIMUM,
		tabwriter.DEBUG)
	testify.Equal_Values(t, bits.WORD_8_MAXIMUM, tabwriter.ESCAPE)
}

func test_format_branch_allocation(t *testing.T) {
	t.Helper()
	cases := [...]struct {
		Name   string
		Source string
		Input  tabwriter.Configuration_Input
	}{
		{
			Name:   "right",
			Source: "a\tbb\nccc\tdd\nx\ty\n",
			Input: configuration_input(
				TEST_WIDTH_FOUR, TEST_WIDTH_ZERO, TEST_WIDTH_ONE, '.',
				tabwriter.ALIGN_RIGHT,
			),
		},
		{
			Name:   "discard",
			Source: "\t\ta\tb\n",
			Input: configuration_input(
				TEST_WIDTH_ONE, TEST_WIDTH_EIGHT, TEST_WIDTH_ONE, ' ',
				tabwriter.TAB_INDENT|tabwriter.DISCARD_EMPTY_COLUMNS,
			),
		},
		{
			Name:   "html",
			Source: "<b>a</b>\t&amp;\n",
			Input: configuration_input(
				TEST_WIDTH_FOUR, TEST_WIDTH_EIGHT, TEST_WIDTH_ONE, '\t',
				tabwriter.FILTER_HTML,
			),
		},
		{
			Name:   "escape",
			Source: "\xffa\tb\xff\n",
			Input: configuration_input(
				TEST_WIDTH_FOUR, TEST_WIDTH_ZERO, TEST_WIDTH_ONE, '.',
				tabwriter.STRIP_ESCAPE|tabwriter.DEBUG,
			),
		},
		{
			Name:   "section",
			Source: "a\tb\fc\td\n",
			Input: configuration_input(
				TEST_WIDTH_ONE, TEST_WIDTH_ZERO, TEST_WIDTH_ZERO, '.',
				tabwriter.DEBUG,
			),
		},
	}
	for _, one := range cases {
		t.Run(one.Name, func(t *testing.T) {
			test_format_allocation(t, one.Source, one.Input)
		})
	}
}

func test_format_allocation(
	t *testing.T, source_text string, input tabwriter.Configuration_Input,
) {
	t.Helper()
	configuration := configuration_from(t, input)
	source := source_from(t, source_text)
	var output [tabwriter.OUTPUT_SIZE_MAXIMUM]byte
	workspace := workspace_new()
	var count tabwriter.Output_Count
	var status tabwriter.Format_Status
	testify.Zero_Allocation(t, func() {
		count, status = tabwriter.Format_Into(
			output[:], source, configuration, workspace_from(&workspace),
		)
	})
	testify.Equal_Values(t, tabwriter.STATUS_OK, status)
	testify.True(t, count > tabwriter.Output_Count(tabwriter.OUTPUT_SIZE_MINIMUM))
}

const OUTPUT_DOMAIN_SIZE_MAXIMUM = len("\txy")

func test_scalar_domains(t *testing.T) {
	for _, input := range [...]tabwriter.Configuration_Input{
		{Minimum_Width: tabwriter.Minimum_Width(bits.INTEGER_MINIMUM)},
		{Minimum_Width: tabwriter.Minimum_Width(bits.INTEGER_MAXIMUM)},
		{Minimum_Width: TEST_WIDTH_TWO},
		{Tab_Width: tabwriter.Tab_Width(bits.INTEGER_MINIMUM)},
		{Tab_Width: tabwriter.Tab_Width(bits.INTEGER_MAXIMUM)},
		{Tab_Width: TEST_WIDTH_ONE},
		{Tab_Width: TEST_WIDTH_TWO},
		{Padding: tabwriter.Padding(bits.INTEGER_MINIMUM)},
		{Padding: tabwriter.Padding(bits.INTEGER_MAXIMUM)},
		{Padding: TEST_WIDTH_TWO},
		{Pad_Character: utf8.CHARACTER_SIZE_MINIMUM},
		{Pad_Character: utf8.CHARACTER_SIZE_TWO},
		{Pad_Character: tabwriter.Pad_Character(bits.WORD_8_MAXIMUM)},
		{Flags: tabwriter.Flags(bits.WORD_MAXIMUM)},
	} {
		configuration, status := tabwriter.New_Configuration(input)
		known_status := status == tabwriter.STATUS_OK
		if status == tabwriter.STATUS_CONFIGURATION_INVALID {
			known_status = true
		}
		testify.True(t, known_status)
		testify.Equal(
			t, status == tabwriter.STATUS_OK,
			bool(tabwriter.Configuration_Valid(configuration)),
		)
	}
}

func test_output_domains(t *testing.T) {
	configuration := configuration_from(t, tabwriter.Configuration_Input{})
	for _, source_size := range [...]int{
		tabwriter.SOURCE_SIZE_MINIMUM,
		utf8.CHARACTER_SIZE_MINIMUM,
		utf8.CHARACTER_SIZE_TWO,
		OUTPUT_DOMAIN_SIZE_MAXIMUM,
	} {
		var source_storage [OUTPUT_DOMAIN_SIZE_MAXIMUM]byte
		for index := range source_storage {
			source_storage[index] = byte(index)
		}
		source, status := tabwriter.Source_Validate(source_storage[:source_size])
		testify.Equal_Values(t, tabwriter.STATUS_OK, status)
		var output [OUTPUT_DOMAIN_SIZE_MAXIMUM]byte
		workspace := workspace_new()
		count, format_status := tabwriter.Format_Into(
			output[:source_size], source, configuration, workspace_from(&workspace),
		)
		testify.Equal_Values(t, tabwriter.STATUS_OK, format_status)
		testify.Equal(t, tabwriter.Output_Count(source_size), count)
	}
	for _, fixture := range [...]struct {
		Source   string
		Expected string
	}{
		{"\tx", "x"},
		{"\txy", "xy"},
	} {
		source := source_from(t, fixture.Source)
		var output [OUTPUT_DOMAIN_SIZE_MAXIMUM]byte
		workspace := workspace_new()
		count, status := tabwriter.Format_Into(
			output[:len(fixture.Expected)], source,
			configuration, workspace_from(&workspace),
		)
		testify.Equal_Values(t, tabwriter.STATUS_OK, status)
		testify.Equal(t, fixture.Expected, string(output[:count]))
	}

	maximum_byte := string([]byte{tabwriter.ESCAPE})
	testify.Equal(t, maximum_byte, format_text(
		t, maximum_byte, tabwriter.Configuration_Input{},
	))
	padding_input := tabwriter.Configuration_Input{
		Minimum_Width: TEST_WIDTH_ONE,
		Pad_Character: tabwriter.Pad_Character(bits.WORD_8_MAXIMUM),
	}
	testify.Equal(t, string([]byte{bits.WORD_8_MAXIMUM, '\n'}), format_text(
		t, "\t\n", padding_input,
	))
}

func test_maximum_domains(t *testing.T) {
	configuration := configuration_from(t, tabwriter.Configuration_Input{})
	var plain [tabwriter.SOURCE_SIZE_MAXIMUM]byte
	for index := range plain {
		plain[index] = 'x'
	}
	format_bytes(t, plain[:], configuration)

	var cells [tabwriter.SOURCE_SIZE_MAXIMUM]byte
	for index := range cells {
		cells[index] = '\t'
	}
	cells[len(cells)-utf8.CHARACTER_SIZE_MINIMUM] = 'x'
	format_bytes(t, cells[:], configuration)

	var lines [tabwriter.SOURCE_SIZE_MAXIMUM]byte
	for index := range lines {
		lines[index] = '\n'
	}
	format_bytes(t, lines[:], configuration)

	var partial [tabwriter.SOURCE_SIZE_MAXIMUM]byte
	for index := range partial[:len(partial)-utf8.CHARACTER_SIZE_MINIMUM] {
		partial[index] = 'x'
	}
	partial[len(partial)-utf8.CHARACTER_SIZE_MINIMUM] = '<'
	html_configuration := configuration_from(t, tabwriter.Configuration_Input{
		Flags: tabwriter.FILTER_HTML,
	})
	format_bytes(t, partial[:], html_configuration)

	var column_lines [tabwriter.SOURCE_SIZE_MAXIMUM]byte
	for index := range column_lines[:len(column_lines)-utf8.CHARACTER_SIZE_TWO] {
		column_lines[index] = '\n'
	}
	column_lines[len(column_lines)-utf8.CHARACTER_SIZE_TWO] = '\t'
	column_lines[len(column_lines)-utf8.CHARACTER_SIZE_MINIMUM] = '\n'
	format_bytes(t, column_lines[:], configuration)

	var widest [tabwriter.SOURCE_SIZE_MAXIMUM]byte
	for index := range widest[:len(widest)-utf8.CHARACTER_SIZE_TWO] {
		widest[index] = 'x'
	}
	widest[len(widest)-utf8.CHARACTER_SIZE_TWO] = '\t'
	widest[len(widest)-utf8.CHARACTER_SIZE_MINIMUM] = 'x'
	assert_result_too_large(t, widest[:], tabwriter.Configuration_Input{
		Padding:       tabwriter.WIDTH_MAXIMUM,
		Pad_Character: '.',
	})
	assert_result_too_large(t, []byte("a\t\tb"), tabwriter.Configuration_Input{
		Minimum_Width: tabwriter.WIDTH_MAXIMUM,
		Pad_Character: '.',
	})
}

func test_escape_domains(t *testing.T) {
	escaped := string([]byte{
		tabwriter.ESCAPE,
		bits.WORD_8_MINIMUM,
		utf8.CHARACTER_SIZE_MINIMUM,
		utf8.CHARACTER_SIZE_TWO,
		tabwriter.ESCAPE,
	})
	format_text(t, escaped, tabwriter.Configuration_Input{})
	for _, source := range [...]string{"<", "&", "x<", "\nx", "\tx", "\t\tx"} {
		format_text(t, source, tabwriter.Configuration_Input{
			Flags: tabwriter.FILTER_HTML,
		})
	}
}

func assert_result_too_large(
	t *testing.T, source_storage []byte, input tabwriter.Configuration_Input,
) {
	t.Helper()
	source, source_status := tabwriter.Source_Validate(source_storage)
	testify.Equal_Values(t, tabwriter.STATUS_OK, source_status)
	configuration := configuration_from(t, input)
	var output [tabwriter.OUTPUT_SIZE_MAXIMUM]byte
	workspace := workspace_new()
	count, status := tabwriter.Format_Into(
		output[:], source, configuration, workspace_from(&workspace),
	)
	testify.Equal_Values(t, tabwriter.STATUS_RESULT_TOO_LARGE, status)
	testify.Equal(t, tabwriter.Output_Count(tabwriter.OUTPUT_SIZE_UNREPRESENTABLE), count)
}

func format_bytes(
	t *testing.T, source_storage []byte, configuration tabwriter.Configuration,
) {
	t.Helper()
	source, status := tabwriter.Source_Validate(source_storage)
	testify.Equal_Values(t, tabwriter.STATUS_OK, status)
	var output [tabwriter.OUTPUT_SIZE_MAXIMUM]byte
	workspace := workspace_new()
	_, format_status := tabwriter.Format_Into(
		output[:], source, configuration, workspace_from(&workspace),
	)
	testify.Equal_Values(t, tabwriter.STATUS_OK, format_status)
}

func configuration_from(
	t *testing.T, input tabwriter.Configuration_Input,
) (configuration tabwriter.Configuration) {
	t.Helper()
	configuration, status := tabwriter.New_Configuration(input)
	testify.Equal_Values(t, tabwriter.STATUS_OK, status)
	return configuration
}

func source_from(t *testing.T, text string) (source tabwriter.Source) {
	t.Helper()
	source, status := tabwriter.Source_Validate(tabwriter.Source_Unvalidated(text))
	testify.Equal_Values(t, tabwriter.STATUS_OK, status)
	return source
}

func format_text(
	t *testing.T, text string, input tabwriter.Configuration_Input,
) (formatted string) {
	t.Helper()
	configuration := configuration_from(t, input)
	source := source_from(t, text)
	var output [tabwriter.OUTPUT_SIZE_MAXIMUM]byte
	workspace := workspace_new()
	count, status := tabwriter.Format_Into(
		output[:], source, configuration, workspace_from(&workspace),
	)
	testify.Equal_Values(t, tabwriter.STATUS_OK, status)
	return string(output[:count])
}

func format_configuration(
	t *testing.T, text string, configuration tabwriter.Configuration,
) (formatted string) {
	t.Helper()
	source := source_from(t, text)
	var output [tabwriter.OUTPUT_SIZE_MAXIMUM]byte
	workspace := workspace_new()
	count, status := tabwriter.Format_Into(
		output[:], source, configuration, workspace_from(&workspace),
	)
	testify.Equal_Values(t, tabwriter.STATUS_OK, status)
	return string(output[:count])
}

func workspace_from(workspace *tabwriter.Workspace) (input tabwriter.Workspace_Input) {
	input.State.Cell_Starts = tabwriter.Cell_Starts_Input(workspace.Cell_Starts)
	input.State.Cell_Ends = tabwriter.Cell_Ends_Input(workspace.Cell_Ends)
	input.State.Cell_Widths = tabwriter.Cell_Widths_Input(workspace.Cell_Widths)
	input.State.Cell_Output_Size = tabwriter.Cell_Output_Sizes_Input(
		workspace.Cell_Output_Size,
	)
	input.State.Cell_Hard_Tabs = tabwriter.Cell_Hard_Tabs_Input(workspace.Cell_Hard_Tabs)
	input.State.Line_First_Cells = tabwriter.Line_First_Cells_Input(
		workspace.Line_First_Cells,
	)
	input.State.Line_Cell_Counts = tabwriter.Line_Cell_Counts_Input(
		workspace.Line_Cell_Counts,
	)
	input.State.Line_Terminators = tabwriter.Line_Terminators_Input(
		workspace.Line_Terminators,
	)
	return input
}

func workspace_new() (workspace tabwriter.Workspace) {
	workspace.Cell_Starts = make(tabwriter.Cell_Starts, tabwriter.CELL_COUNT_MAXIMUM)
	workspace.Cell_Ends = make(tabwriter.Cell_Ends, tabwriter.CELL_COUNT_MAXIMUM)
	workspace.Cell_Widths = make(tabwriter.Cell_Widths, tabwriter.CELL_COUNT_MAXIMUM)
	workspace.Cell_Output_Size = make(
		tabwriter.Cell_Output_Sizes, tabwriter.CELL_COUNT_MAXIMUM,
	)
	workspace.Cell_Hard_Tabs = make(tabwriter.Cell_Hard_Tabs, tabwriter.CELL_COUNT_MAXIMUM)
	workspace.Line_First_Cells = make(
		tabwriter.Line_First_Cells, tabwriter.LINE_COUNT_MAXIMUM,
	)
	workspace.Line_Cell_Counts = make(
		tabwriter.Line_Cell_Counts, tabwriter.LINE_COUNT_MAXIMUM,
	)
	workspace.Line_Terminators = make(
		tabwriter.Line_Terminators, tabwriter.LINE_COUNT_MAXIMUM,
	)
	return workspace
}
