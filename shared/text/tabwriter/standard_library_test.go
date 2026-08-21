package tabwriter_test

import (
	"testing"
	standard_tabwriter "text/tabwriter"

	"local/james-orcales/shared/simulation/aver/default"
	"local/james-orcales/shared/strconv"
	"local/james-orcales/shared/testify"
	"local/james-orcales/shared/text/tabwriter"
	"local/james-orcales/shared/unicode/utf8"
)

// TestMain keeps allocation probes on production assertion paths.
func TestMain(m *testing.M) {
	aver.Run_Test_Main(m)
}

type standard_output struct {
	Data  [tabwriter.OUTPUT_SIZE_MAXIMUM]byte
	Count int
}

func (output *standard_output) Write(source []byte) (count int, failure error) {
	count = copy(output.Data[output.Count:], source)
	output.Count += count
	return count, nil
}

// Test_Standard_Library compares semantic branches against upstream output.
func Test_Standard_Library(t *testing.T) {
	cases := [...]struct {
		Source string
		Input  tabwriter.Configuration_Input
	}{
		{"", configuration_input(
			TEST_WIDTH_EIGHT, TEST_WIDTH_ZERO, TEST_WIDTH_ONE, '.', TEST_FLAGS_NONE,
		)},
		{"\xff\xff", configuration_input(
			TEST_WIDTH_EIGHT, TEST_WIDTH_ZERO, TEST_WIDTH_ONE, '.',
			tabwriter.STRIP_ESCAPE,
		)},
		{"abc\xff\tdef", configuration_input(
			TEST_WIDTH_EIGHT, TEST_WIDTH_ZERO, TEST_WIDTH_ONE, '.', TEST_FLAGS_NONE,
		)},
		{"\n\n\n", configuration_input(
			TEST_WIDTH_EIGHT, TEST_WIDTH_ZERO, TEST_WIDTH_ONE, '.', TEST_FLAGS_NONE,
		)},
		{"*\t*\t", configuration_input(
			TEST_WIDTH_EIGHT, TEST_WIDTH_ZERO, TEST_WIDTH_ONE, '.', tabwriter.DEBUG,
		)},
		{"\t\n", configuration_input(
			TEST_WIDTH_EIGHT, TEST_WIDTH_ZERO, TEST_WIDTH_ONE, '.', TEST_FLAGS_NONE,
		)},
		{"f) f&lt;o\t<b>bar</b>\t\n", configuration_input(
			TEST_WIDTH_EIGHT, TEST_WIDTH_ZERO, TEST_WIDTH_ONE, '.',
			tabwriter.FILTER_HTML,
		)},
		{"1\t2\t3\t4\n11\t222\t3333\t44444\n", configuration_input(
			TEST_WIDTH_ONE, TEST_WIDTH_ZERO, TEST_WIDTH_ZERO, '.', TEST_FLAGS_NONE,
		)},
		{"1\t2\t3\t4\f11\t222\t3333\t44444\n", configuration_input(
			TEST_WIDTH_ONE, TEST_WIDTH_ZERO, TEST_WIDTH_ZERO, '.', tabwriter.DEBUG,
		)},
		{"本\tb\tc\naa\t本本本\tcccc\tddddd\naaa\tbbbb\n", configuration_input(
			TEST_WIDTH_EIGHT, TEST_WIDTH_ZERO, TEST_WIDTH_ONE, '.', TEST_FLAGS_NONE,
		)},
		{"a\tè\tc\t\naa\tèèè\tcccc\tddddd\t\naaa\tèèèè\t\n", configuration_input(
			TEST_WIDTH_EIGHT, TEST_WIDTH_ZERO, TEST_WIDTH_ONE, ' ',
			tabwriter.ALIGN_RIGHT,
		)},
		{"\t\ta\tb\n", configuration_input(
			TEST_WIDTH_ONE, TEST_WIDTH_EIGHT, TEST_WIDTH_ONE, ' ', tabwriter.TAB_INDENT,
		)},
		{"\v\va\vb\n", configuration_input(
			TEST_WIDTH_ONE, TEST_WIDTH_EIGHT, TEST_WIDTH_ONE, ' ',
			tabwriter.DISCARD_EMPTY_COLUMNS,
		)},
	}
	for _, one := range cases {
		shared := format_text(t, one.Source, one.Input)
		standard := standard_format(one.Source, one.Input)
		testify.Equal(t, standard, shared)
	}
	test_standard_library_extended(t)
}

func test_standard_library_extended(t *testing.T) {
	t.Helper()
	cases := [...]struct {
		Source string
		Input  tabwriter.Configuration_Input
	}{
		{"\xff\"foo\t\n\tbar\"\xff", configuration_input(
			TEST_WIDTH_EIGHT, TEST_WIDTH_ZERO, TEST_WIDTH_ONE, '.',
			tabwriter.STRIP_ESCAPE,
		)},
		{"g) f&lt;o\t<b>bar</b>\t non-terminated entity &amp", configuration_input(
			TEST_WIDTH_EIGHT, TEST_WIDTH_ZERO, TEST_WIDTH_ONE, '.',
			tabwriter.FILTER_HTML|tabwriter.DEBUG,
		)},
		{
			"4444\t日本語\t22\t1\t333\n" +
				"999999999\t22\n" +
				"7\t22\n" +
				"\t\t\t88888888\n" +
				"\n" +
				"666666\t666666\t666666\t4444\n" +
				"1\t1\t999999999\t0000000000\n",
			configuration_input(
				TEST_WIDTH_FOUR, TEST_WIDTH_ZERO, TEST_WIDTH_ONE, '-',
				TEST_FLAGS_NONE,
			),
		},
		{
			".0\t.3\t2.4\t-5.1\t\n" +
				"23.0\t12345678.9\t2.4\t-989.4\t\n" +
				"5.1\t12.0\t2.4\t-7.0\t\n" +
				".0\t0.0\t332.0\t8908.0\t\n" +
				".0\t-.3\t456.4\t22.1\t\n" +
				".0\t1.2\t44.4\t-13.3\t\t",
			configuration_input(
				TEST_WIDTH_ONE, TEST_WIDTH_ZERO, TEST_WIDTH_TWO, ' ',
				tabwriter.ALIGN_RIGHT|tabwriter.DEBUG,
			),
		},
		{
			"a\vb\v\vd\n" +
				"a\vb\v\vd\ve\n" +
				"a\n" +
				"a\vb\vc\vd\n" +
				"a\vb\vc\vd\ve\n",
			configuration_input(
				strconv.DECIMAL_PAIR_BASE, strconv.DECIMAL_PAIR_BASE,
				TEST_WIDTH_ZERO, '\t',
				tabwriter.DISCARD_EMPTY_COLUMNS|tabwriter.DEBUG,
			),
		},
	}
	for _, one := range cases {
		shared := format_text(t, one.Source, one.Input)
		standard := standard_format(one.Source, one.Input)
		testify.Equal(t, standard, shared)
	}
}

// Test_Standard_Library_Short_Exhaustive checks every short structural-byte arrangement.
func Test_Standard_Library_Short_Exhaustive(t *testing.T) {
	alphabet := [...]byte{
		'a', '\t', '\v', '\n', '\f', tabwriter.ESCAPE, '<', '>', '&', ';',
	}
	configurations := [...]tabwriter.Configuration_Input{
		configuration_input(
			TEST_WIDTH_ONE, TEST_WIDTH_ZERO, TEST_WIDTH_ZERO, '.', TEST_FLAGS_NONE,
		),
		configuration_input(
			TEST_WIDTH_FOUR, TEST_WIDTH_ZERO, TEST_WIDTH_ONE, '.', TEST_FLAGS_NONE,
		),
		configuration_input(
			TEST_WIDTH_FOUR, TEST_WIDTH_ZERO, TEST_WIDTH_ONE, '.',
			tabwriter.ALIGN_RIGHT,
		),
		configuration_input(
			TEST_WIDTH_FOUR, TEST_WIDTH_ZERO, TEST_WIDTH_ONE, '.',
			tabwriter.DISCARD_EMPTY_COLUMNS,
		),
		configuration_input(
			TEST_WIDTH_ONE, TEST_WIDTH_EIGHT, TEST_WIDTH_ONE, ' ',
			tabwriter.TAB_INDENT,
		),
		configuration_input(
			TEST_WIDTH_FOUR, TEST_WIDTH_EIGHT, TEST_WIDTH_ONE, '\t',
			tabwriter.FILTER_HTML,
		),
		configuration_input(
			TEST_WIDTH_FOUR, TEST_WIDTH_ZERO, TEST_WIDTH_ONE, '.',
			tabwriter.STRIP_ESCAPE,
		),
		configuration_input(
			TEST_WIDTH_FOUR, TEST_WIDTH_EIGHT, TEST_WIDTH_ONE, '\t',
			tabwriter.FILTER_HTML|tabwriter.DISCARD_EMPTY_COLUMNS|tabwriter.DEBUG,
		),
	}
	var source [len("abc")]byte
	for size := tabwriter.SOURCE_SIZE_MINIMUM; size <= len(source); size++ {
		combination_count := utf8.CHARACTER_SIZE_MINIMUM
		for range size {
			combination_count *= len(alphabet)
		}
		for combination := range combination_count {
			value := combination
			for index := range size {
				source[index] = alphabet[value%len(alphabet)]
				value /= len(alphabet)
			}
			text := string(source[:size])
			for _, configuration := range configurations {
				shared := format_text(t, text, configuration)
				standard := standard_format(text, configuration)
				if shared != standard {
					t.Fatalf(
						"source %q: shared %q, standard %q",
						text, shared, standard,
					)
				}
			}
		}
	}
}

func configuration_input(
	minimum_width int,
	tab_width int,
	padding int,
	pad_character byte,
	flags tabwriter.Flags,
) (input tabwriter.Configuration_Input) {
	input.Minimum_Width = tabwriter.Minimum_Width(minimum_width)
	input.Tab_Width = tabwriter.Tab_Width(tab_width)
	input.Padding = tabwriter.Padding(padding)
	input.Pad_Character = tabwriter.Pad_Character(pad_character)
	input.Flags = flags
	return input
}

func standard_format(
	source string, input tabwriter.Configuration_Input,
) (formatted string) {
	var output standard_output
	writer := new(standard_tabwriter.Writer)
	writer.Init(
		&output,
		int(input.Minimum_Width),
		int(input.Tab_Width),
		int(input.Padding),
		byte(input.Pad_Character),
		uint(input.Flags),
	)
	written, write_error := writer.Write([]byte(source))
	if write_error != nil {
		panic(write_error)
	}
	if written != len(source) {
		panic("standard tabwriter accepted a partial source")
	}
	flush_error := writer.Flush()
	if flush_error != nil {
		panic(flush_error)
	}
	return string(output.Data[:output.Count])
}
