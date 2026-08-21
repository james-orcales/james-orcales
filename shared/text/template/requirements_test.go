package template_test

import (
	"testing"
	"text/template/parse"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/math/big"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/strconv"
	"local/james-orcales/shared/testify"
	"local/james-orcales/shared/text/template"
	"local/james-orcales/shared/unicode/utf8"
)

// Test_Compile prevents hidden parser ownership or source copying.
func Test_Compile(t *testing.T) {
	program := compile_from(t, "hello {{.Name}}")
	testify.Equal(t, "hello {{.Name}}", string(program.Source.Data.Value))
	var workspace template.Syntax_Workspace
	input := parse_workspace_from(&workspace)
	source, source_status := template.Source_Validate([]byte("{{.}}"))
	testify.Equal_Values(t, template.PARSE_STATUS_OK, source_status)
	configuration, configuration_status := template.New_Configuration(
		template.Configuration_Input{},
	)
	testify.Equal_Values(t, template.PARSE_STATUS_OK, configuration_status)
	_, _, parse_status := template.Parse_Into(
		template.Source_Header_Unvalidated(source),
		configuration,
		input,
	)
	testify.Equal_Values(t, template.PARSE_STATUS_OK, parse_status)
	test_compile_input_boundaries(t)
	test_compile_diagnostic_boundaries(t)
}

// Test_Values prevents reflection and ownership from entering value conversion.
func Test_Values(t *testing.T) {
	value := template.Value_Of_Text([]byte("Ada"))
	text, status := template.Value_As_Text(template.Value(value))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "Ada", string(text))

	_, status = template.Value_As_Text(value_integer(TEST_COUNT_ONE))
	testify.Equal_Values(t, template.STATUS_VALUE_INVALID, status)
	template.Value_Of_Unsigned(TEST_COUNT_ONE)
	test_value_boundaries(t)
}

// Test_Execute keeps syntax, value, and function paths interoperable.
func Test_Execute(t *testing.T) {
	test_execute_fields(t)
	test_small_shapes(t)
	test_small_counts(t)
	test_construct_positions(t)
	test_execute_control(t)
	test_execute_pipeline(t)
	test_execute_templates(t)
	test_execute_builtins(t)
	test_execute_numbers(t)
	test_execute_map_order(t)
	test_execute_format_builtins(t)
	test_execute_composite_output(t)
	test_standard_library_execution(t)
	test_standard_library_value_execution(t)
	test_standard_library_collection_execution(t)
	test_standard_library_slice_capacity(t)
}

// Test_Bounds keeps forged caller state outside executor traversal.
func Test_Bounds(t *testing.T) {
	t.Parallel()
	test_formulas(t)
	input := template.Compile_Input{Source: []byte("x")}
	_, _, status := template.Compile(input)
	testify.Equal_Values(t, template.PARSE_STATUS_WORKSPACE_INVALID, status)

	program := compile_from(t, "hello")
	var empty template.Workspace
	var empty_input template.Workspace_Input
	empty_input.State.Value = template.Workspace_Unvalidated_Pointer{
		Value: (*template.Workspace_Unvalidated)(&empty),
	}
	_, _, execution_status := template.Execute_Into(template.Execute_Input{
		Program:   template.Program_Unvalidated(program),
		Value:     template.Value_Unvalidated(template.Value_Nil()),
		Workspace: empty_input,
	})
	testify.Equal_Values(t, template.STATUS_WORKSPACE_INVALID, execution_status)

	test_hostile_documents(t)
	program.Document.Root = template.Document_Root(template.NO_NODE)
	var execution execution_storage
	var output [template.OUTPUT_SIZE_MAXIMUM]byte
	_, _, execution_status = template.Execute_Into(template.Execute_Input{
		Destination: output[:], Program: template.Program_Unvalidated(program),
		Value:     template.Value_Unvalidated(template.Value_Nil()),
		Workspace: workspace_from(&execution),
	})
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, execution_status)
	execution = execution_storage{}
	_, _, execution_status = template.Execute_Into(template.Execute_Input{
		Program:   template.Program_Unvalidated(template.Program{}),
		Value:     template.Value_Unvalidated(template.Value_Nil()),
		Workspace: workspace_from(&execution),
	})
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, execution_status)
	test_execute_program_bounds(t)
	test_execute_value_bounds(t)
	test_execute_generated_bounds(t)
	test_execute_input_boundaries(t)
	test_workspace_float_boundaries(t)
}

// Test_Output keeps atomic commit load-bearing on destination refusal.
func Test_Output(t *testing.T) {
	program := compile_from(t, "hello")
	var execution execution_storage
	var short [len("hell")]byte
	for index := range short {
		short[index] = 'x'
	}
	count, diagnostic, status := template.Execute_Into(template.Execute_Input{
		Destination: short[:], Program: template.Program_Unvalidated(program),
		Value:     template.Value_Unvalidated(template.Value_Nil()),
		Workspace: workspace_from(&execution),
	})
	testify.Equal_Values(t, template.STATUS_OUTPUT_TOO_SMALL, status)
	testify.Equal_Values(t, template.Output_Count(template.OUTPUT_SIZE_MINIMUM), count)
	testify.Equal_Values(t, template.STATUS_OUTPUT_TOO_SMALL, diagnostic.Code)
	testify.Equal(t, "xxxx", string(short[:]))
	test_execute_output_bounds(t)
	test_execute_output_maximum(t)
	test_execute_diagnostic_positions(t)
}

// Test_Allocation prevents caller-owned execution storage from escaping.
func Test_Allocation(t *testing.T) {
	source := []byte("hello {{.Name}}")
	var syntax template.Syntax_Workspace
	compile_input := template.Compile_Input{
		Source: source, Workspace: syntax_from(&syntax),
	}
	var program template.Program
	var parse_status template.Parse_Status
	testify.Zero_Allocation(t, func() {
		program, _, parse_status = template.Compile(compile_input)
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, parse_status)

	fields := [...]template.Field{{
		Name:  []byte("Name"),
		Value: template.Field_Value{Data: value_text([]byte("Ada"))},
	}}
	root := template.Value_Of_Object(fields[:])
	var execution execution_storage
	var output [template.OUTPUT_SIZE_MAXIMUM]byte
	input := template.Execute_Input{
		Destination: output[:], Program: template.Program_Unvalidated(program),
		Value:     template.Value_Unvalidated(root),
		Workspace: workspace_from(&execution),
	}
	var count template.Output_Count
	var status template.Status
	testify.Zero_Allocation(t, func() {
		count, _, status = template.Execute_Into(input)
	})
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "hello Ada", string(output[:count]))
	test_builtin_allocation(t)
	test_scalar_value_allocation(t)
	test_borrowed_value_allocation(t)
	test_compile_branch_allocation(t)
	test_execution_branch_allocation(t)
}

const TEST_COUNT_ZERO = template.SOURCE_SIZE_MINIMUM

// TEST_ALPHABET_SIZE counts the lower-case letters one generated name uses.
const TEST_ALPHABET_SIZE = 'z' - 'a' + TEST_COUNT_ONE
const TEST_COUNT_ONE = 1
const TEST_COUNT_TWO = TEST_COUNT_ONE + TEST_COUNT_ONE
const TEST_COUNT_THREE = TEST_COUNT_TWO + TEST_COUNT_ONE

// TEST_PADDED_STRIDE spaces the sources that also run behind comment padding.
const TEST_PADDED_STRIDE = TEST_COUNT_THREE + TEST_COUNT_THREE + TEST_COUNT_TWO

// TEST_LIMIT_SMALL is the largest node policy the small-count sources sweep.
const TEST_LIMIT_SMALL = template.Node_Limit(TEST_COUNT_THREE + TEST_COUNT_TWO)
const TEST_COUNT_FOUR = TEST_COUNT_TWO * TEST_COUNT_TWO
const TEST_COUNT_FIVE = TEST_COUNT_FOUR + TEST_COUNT_ONE
const TEST_COUNT_SEVEN = strconv.DECIMAL_BASE - TEST_COUNT_THREE
const TEST_COUNT_TEN = strconv.DECIMAL_BASE
const TEST_COUNT_THIRTEEN = TEST_COUNT_TEN + TEST_COUNT_THREE
const TEST_COUNT_FOURTEEN = TEST_COUNT_TEN + TEST_COUNT_FOUR
const TEST_COUNT_TWENTY_TWO = TEST_COUNT_TWO*TEST_COUNT_TEN + TEST_COUNT_TWO

// TEST_NODE_FIELD_COUNT counts the fields one forged record carries.
const TEST_NODE_FIELD_COUNT = TEST_COUNT_SEVEN

// TEST_HOSTILE_NODE_COUNT bounds the records one forged program carries.
const TEST_HOSTILE_NODE_COUNT = TEST_COUNT_SEVEN

// One forged record reaches execution, so every node bound holds under hostile syntax.
func test_hostile_nodes(t *testing.T) {
	t.Helper()
	// Each forged field stands past the bound its record admits, so validation refuses it.
	extremes := [...]uint16{
		template.SOURCE_SIZE_MAXIMUM, uint16(template.NODE_COUNT_MAXIMUM),
		bits.WORD_16_MAXIMUM,
	}
	var output [template.OUTPUT_SIZE_MAXIMUM]byte
	for _, extreme := range extremes {
		for index := TEST_COUNT_ZERO; index < TEST_HOSTILE_NODE_COUNT; index++ {
			for field := TEST_COUNT_ZERO; field < TEST_NODE_FIELD_COUNT; field++ {
				var syntax template.Syntax_Workspace
				program, _, status := template.Compile(template.Compile_Input{
					Source:    []byte(`{{if .}}{{print . .}}{{end}}`),
					Workspace: syntax_from(&syntax),
				})
				testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
				hostile_node_set(&syntax.Nodes[index], field, extreme)
				var execution execution_storage
				template.Execute_Into(template.Execute_Input{
					Destination: output[:],
					Program:     template.Program_Unvalidated(program),
					Value:       template.Value_Unvalidated(value_text(nil)),
					Workspace:   workspace_from(&execution),
				})
			}
		}
	}
}

// Forges one field of one record.
func hostile_node_set(node *template.Node, field int, value uint16) {
	switch field {
	case TEST_COUNT_ZERO:
		node.Kind = template.Node_Kind(value)
	case TEST_COUNT_ONE:
		node.Start = template.Node_Start(value)
	case TEST_COUNT_TWO:
		node.End = template.Node_End(value)
	case TEST_COUNT_THREE:
		node.Value_Start = template.Node_Value_Start(value)
	case TEST_COUNT_FOUR:
		node.Value_End = template.Node_Value_End(value)
	case TEST_COUNT_FIVE:
		node.First_Child = template.Node_First_Child(value)
	default:
		node.Next_Sibling = template.Node_Next_Sibling(value)
	}
}

// Parser and executor formulas must stay attached to authoritative shared domains.
func test_formulas(t *testing.T) {
	testify.Equal(t, bytes.SLICE_SIZE_MAXIMUM, template.SOURCE_SIZE_MAXIMUM)
	testify.Equal(t,
		template.SOURCE_SIZE_MAXIMUM+utf8.CHARACTER_SIZE_MINIMUM,
		template.SOURCE_SIZE_UNVALIDATED_MAXIMUM,
	)
	testify.Equal_Values(t, big.BASE_BINARY, template.NUMBER_BASE_BINARY)
	testify.Equal_Values(t, big.BASE_OCTAL, template.NUMBER_BASE_OCTAL)
	testify.Equal_Values(t, big.BASE_DECIMAL, template.NUMBER_BASE_DECIMAL)
	testify.Equal_Values(t, big.BASE_HEXADECIMAL, template.NUMBER_BASE_HEXADECIMAL)
	testify.Equal_Values(t,
		strconv.HEXADECIMAL_DIGIT_COUNT, template.QUOTED_HEX_DIGIT_COUNT,
	)
	testify.Equal_Values(t,
		strconv.SHORT_UNICODE_DIGIT_COUNT, template.QUOTED_SHORT_UNICODE_DIGIT_COUNT,
	)
	testify.Equal_Values(t,
		strconv.LONG_UNICODE_DIGIT_COUNT, template.QUOTED_LONG_UNICODE_DIGIT_COUNT,
	)
	testify.Equal(t,
		template.OUTPUT_SIZE_MAXIMUM+template.SOURCE_SIZE_MAXIMUM,
		template.GENERATED_SIZE_MAXIMUM,
	)
	testify.Equal(t,
		template.SLICE_INDEX_CAPACITY+template.SLICE_INDEX_INCREMENT,
		template.SLICE_INDEX_COUNT,
	)
	testify.Equal(t, utf8.CHARACTER_SIZE_MINIMUM, template.SOURCE_POSITION_INCREMENT)
	testify.Equal_Values(t,
		template.TOKEN_EOF+template.TOKEN_KIND_INCREMENT,
		template.TOKEN_ASSIGN,
	)
	testify.Equal_Values(t,
		template.NODE_TEXT+template.NODE_KIND_INCREMENT,
		template.NODE_ACTION,
	)
	testify.Equal_Values(t,
		template.VALUE_NIL+template.VALUE_KIND_INCREMENT,
		template.VALUE_BOOLEAN,
	)
	testify.Equal_Values(t, big.FLOAT_64_SIGN_MASK, template.FLOAT_64_SIGN_MASK)
	testify.Equal_Values(t, big.FLOAT_64_MANTISSA_MASK, template.FLOAT_64_FRACTION_MASK)
}

func test_scalar_value_allocation(t *testing.T) {
	t.Helper()
	var value template.Value
	testify.Zero_Allocation(t, func() {
		value = value_nil()
	})
	testify.Zero_Allocation(t, func() {
		value = value_boolean(true)
	})
	testify.Zero_Allocation(t, func() {
		value = value_boolean(false)
	})
	minimum := template.Integer(bits.INTEGER_64_MINIMUM)
	testify.Zero_Allocation(
		t,
		func() {
			value = value_integer(minimum)
		},
	)
	testify.Zero_Allocation(
		t,
		func() {
			value = value_unsigned(template.Unsigned(bits.WORD_64_MAXIMUM))

		},
	)
	testify.Zero_Allocation(
		t,
		func() {
			value = value_float(template.Float(bits.WORD_64_MAXIMUM))

		},
	)
	complex_value := template.Complex{
		Real:      template.Complex_Real(bits.WORD_64_MINIMUM),
		Imaginary: template.Complex_Imaginary(bits.WORD_64_MAXIMUM),
	}
	testify.Zero_Allocation(t, func() {
		value = value_complex(complex_value)
	})
	testify.Equal_Values(t, template.VALUE_COMPLEX, value.Kind)
}

func test_borrowed_value_allocation(t *testing.T) {
	t.Helper()
	text := template.Text("text")
	values := [...]template.Value{value_nil()}
	fields := [...]template.Field{{Name: []byte("Name")}}
	var value template.Value
	testify.Zero_Allocation(t, func() {
		value = value_text(text)
	})
	testify.Zero_Allocation(t, func() {
		value = value_sequence(values[:])
	})
	testify.Zero_Allocation(t, func() {
		value = value_object(fields[:])
	})
	testify.Zero_Allocation(t, func() {
		value = value_map(fields[:])
	})
	testify.Zero_Allocation(t, func() {
		value = value_function(identity)
	})
	text_value := template.Value_Of_Text(text)
	var extracted template.Text
	var status template.Value_Text_Status
	testify.Zero_Allocation(t, func() {
		extracted, status = template.Value_As_Text(template.Value(text_value))
	})
	testify.Equal_Values(t, template.VALUE_TEXT_STATUS_OK, status)
	testify.Equal(t, "text", string(extracted))
	testify.Zero_Allocation(t, func() {
		extracted, status = template.Value_As_Text(value)
	})
	testify.Equal_Values(t, template.VALUE_TEXT_STATUS_INVALID, status)
}

func test_compile_branch_allocation(t *testing.T) {
	t.Helper()
	for _, source := range [...]string{
		`{{if .Ready}}{{range .Items}}{{.}}{{else}}empty{{end}}{{else}}no{{end}}`,
		`{{range $key, $value := .}}{{$key}}={{$value}};{{end}}`,
		`{{template "named" .}}{{define "named"}}value{{end}}`,
		`{{identity .}}`,
	} {
		var syntax template.Syntax_Workspace
		input := template.Compile_Input{
			Source: []byte(source), Workspace: syntax_from(&syntax),
		}
		var status template.Parse_Status
		testify.Zero_Allocation(t, func() {
			_, _, status = template.Compile(input)
		})
		testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	}
}

func test_execution_branch_allocation(t *testing.T) {
	t.Helper()
	items := [...]template.Value{value_integer(TEST_COUNT_ONE), value_integer(TEST_COUNT_TWO)}
	fields := [...]template.Field{
		{
			Name:  []byte("Ready"),
			Value: template.Field_Value{Data: value_boolean(true)},
		},
		{
			Name: []byte("Items"),
			Value: template.Field_Value{
				Data: value_sequence(items[:]),
			},
		},
	}
	test_execution_allocation(
		t, `{{if .Ready}}{{range .Items}}{{.}}{{else}}empty{{end}}{{else}}no{{end}}`,
		value_object(fields[:]), nil, "12",
	)
	entries := [...]template.Field{
		{
			Key: template.Field_Key{Data: value_text([]byte("b"))},
			Value: template.Field_Value{
				Data: value_integer(TEST_COUNT_TWO),
			},
		},
		{
			Key: template.Field_Key{Data: value_text([]byte("a"))},
			Value: template.Field_Value{
				Data: value_integer(TEST_COUNT_ONE),
			},
		},
	}
	test_execution_allocation(
		t, `{{range $key, $value := .}}{{$key}}={{$value}};{{end}}`, value_map(entries[:]),
		nil, "a=1;b=2;",
	)
	test_execution_allocation(
		t, `{{range .}}{{.}}{{else}}empty{{end}}`, value_integer(-TEST_COUNT_ONE), nil,
		"empty",
	)
	capacity_values := [...]template.Value{
		value_integer(TEST_COUNT_THREE),
		value_integer(TEST_COUNT_FOUR),
		value_integer(TEST_COUNT_FIVE),
		value_integer(TEST_COUNT_ZERO),
		value_integer(TEST_COUNT_ZERO),
	}
	test_execution_allocation(
		t, `{{slice . 3 5}}`, value_sequence(capacity_values[:TEST_COUNT_THREE]), nil,
		"[0 0]",
	)
	test_execution_allocation(
		t, `{{template "named" .}}{{define "named"}}value{{end}}`, value_nil(), nil,
		"value",
	)
	functions := [...]template.Function{{Name: []byte("identity"), Call: identity}}
	test_execution_allocation(
		t, `{{identity .}}`, value_text([]byte("value")), functions[:], "value",
	)
}

func test_execution_allocation(
	t *testing.T, source string, value template.Value, functions template.Functions,
	expected string,
) {
	t.Helper()
	program := compile_from(t, source)
	var execution execution_storage
	var output [template.OUTPUT_SIZE_MAXIMUM]byte
	input := template.Execute_Input{
		Destination: output[:], Program: template.Program_Unvalidated(program),
		Value:     template.Value_Unvalidated(value),
		Functions: functions, Workspace: workspace_from(&execution),
	}
	var count template.Output_Count
	var status template.Status
	testify.Zero_Allocation(t, func() {
		count, _, status = template.Execute_Into(input)
	})
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, expected, string(output[:count]))
}

func test_compile_input_boundaries(t *testing.T) {
	t.Helper()
	var workspace template.Syntax_Workspace
	program, _, status := template.Compile(template.Compile_Input{
		Source: []byte{}, Workspace: syntax_from(&workspace),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	testify.Equal_Values(t, template.Node_Count(TEST_COUNT_ONE), program.Document.Node_Count)
	workspace = template.Syntax_Workspace{}
	program, _, status = template.Compile(template.Compile_Input{
		Source: []byte(`x{{define "x"}}{{end}}`), Workspace: syntax_from(&workspace),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	testify.Equal_Values(
		t, template.Document_First_Template(
			template.DOCUMENT_FIRST_TEMPLATE_MINIMUM+
				template.NODE_COUNT_INCREMENT+template.NODE_COUNT_INCREMENT,
		),
		program.Document.First_Template,
	)

	workspace = template.Syntax_Workspace{}
	_, _, status = template.Compile(template.Compile_Input{
		Source: []byte("xx"), Workspace: syntax_from(&workspace),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	var maximum [template.SOURCE_SIZE_MAXIMUM]byte
	workspace = template.Syntax_Workspace{}
	_, _, status = template.Compile(template.Compile_Input{
		Source: maximum[:], Workspace: syntax_from(&workspace),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	var oversized [template.SOURCE_SIZE_UNVALIDATED_MAXIMUM]byte
	_, _, status = template.Compile(template.Compile_Input{Source: oversized[:]})
	testify.Equal_Values(t, template.PARSE_STATUS_INPUT_INVALID, status)

	test_compile_delimiter_boundaries(t)
	test_compile_mode_boundaries(t)
}

func test_value_boundaries(t *testing.T) {
	t.Helper()
	template.Value_Of_Integer(template.Integer(bits.INTEGER_64_MINIMUM))
	template.Value_Of_Integer(template.Integer(bits.INTEGER_64_MAXIMUM))
	template.Value_Of_Unsigned(template.Unsigned(bits.WORD_64_MINIMUM))
	template.Value_Of_Unsigned(template.Unsigned(bits.WORD_64_MAXIMUM))
	template.Value_Of_Float(template.Float(bits.WORD_64_MINIMUM + TEST_COUNT_ONE))
	template.Value_Of_Float(template.Float(bits.WORD_64_MINIMUM + TEST_COUNT_TWO))
	template.Value_Of_Float(template.Float(bits.WORD_64_MAXIMUM))
	template.Value_Of_Complex(template.Complex{
		Real: TEST_COUNT_ONE, Imaginary: TEST_COUNT_ONE,
	})
	template.Value_Of_Complex(template.Complex{
		Real: TEST_COUNT_TWO, Imaginary: TEST_COUNT_TWO,
	})
	template.Value_Of_Complex(template.Complex{
		Real:      template.Complex_Real(bits.WORD_64_MAXIMUM),
		Imaginary: template.Complex_Imaginary(bits.WORD_64_MAXIMUM),
	})

	var text [template.OUTPUT_SIZE_MAXIMUM]byte
	value := template.Value_Of_Text(nil)
	_, status := template.Value_As_Text(template.Value(value))
	testify.Equal_Values(t, template.VALUE_TEXT_STATUS_OK, status)
	value = template.Value_Of_Text(text[:])
	result, status := template.Value_As_Text(template.Value(value))
	testify.Equal_Values(t, template.VALUE_TEXT_STATUS_OK, status)
	testify.Equal(t, len(text), len(result))
	test_collection_constructor_boundaries(t)
}

func test_collection_constructor_boundaries(t *testing.T) {
	t.Helper()
	var values [template.VALUE_COUNT_UNVALIDATED_MAXIMUM]template.Value
	template.Value_Of_Sequence(nil)
	template.Value_Of_Sequence(values[:])
	var fields [template.VALUE_COUNT_UNVALIDATED_MAXIMUM]template.Field
	template.Value_Of_Object(nil)
	template.Value_Of_Object(fields[:])
	template.Value_Of_Map(nil)
	template.Value_Of_Map(fields[:TEST_COUNT_TWO])
	template.Value_Of_Map(fields[:])
}

func test_workspace_float_boundaries(t *testing.T) {
	t.Helper()
	test_workspace_float(t, big.Float{
		Precision: big.Float_Precision(big.FLOAT_PRECISION_MAXIMUM),
		Mode:      big.Rounding_Mode(big.ROUND_TO_POSITIVE_INFINITY),
		Accuracy:  big.ACCURACY_BELOW,
		Form:      big.FLOAT_FORM_INFINITY,
		Negative:  big.POLARITY_NEGATIVE,
	})
	test_workspace_float(t, big.Float{
		Precision: TEST_COUNT_TWO,
		Mode:      big.Rounding_Mode(big.ROUND_TO_NEAREST_AWAY),
		Accuracy:  big.ACCURACY_ABOVE,
		Negative:  big.POLARITY_NEGATIVE,
	})
	test_workspace_finite_float(t, TEST_COUNT_ONE, big.FLOAT_EXPONENT_MINIMUM, TEST_COUNT_ONE,
		big.ROUND_TO_ZERO)
	test_workspace_finite_float(t, TEST_COUNT_TWO, big.FLOAT_EXPONENT_MAXIMUM, TEST_COUNT_ONE,
		big.ROUND_AWAY_FROM_ZERO)
	test_workspace_finite_float(t, TEST_COUNT_TWO, TEST_COUNT_ONE, TEST_COUNT_ONE,
		big.ROUND_TO_NEGATIVE_INFINITY)
	test_workspace_finite_float(t, TEST_COUNT_TWO, TEST_COUNT_TWO, TEST_COUNT_ONE,
		big.ROUND_TO_POSITIVE_INFINITY)
	test_workspace_finite_float(
		t, big.WORD_BIT_COUNT+big.WORD_COUNT_INCREMENT, -TEST_COUNT_ONE,
		big.WORD_COUNT_INCREMENT+big.WORD_COUNT_INCREMENT, big.ROUND_TO_NEAREST_EVEN,
	)
	test_workspace_finite_float(
		t, big.FLOAT_PRECISION_MAXIMUM, TEST_COUNT_ZERO, big.WORD_COUNT_MAXIMUM,
		big.ROUND_TO_NEAREST_EVEN,
	)
}

func test_workspace_finite_float(
	t *testing.T, precision int, exponent int, count int, mode big.Rounding_Mode_Unvalidated,
) {
	t.Helper()
	value := big.Float{
		Precision: big.Float_Precision(precision),
		Mode:      big.Rounding_Mode(mode),
		Accuracy:  big.ACCURACY_EXACT,
		Form:      big.FLOAT_FORM_FINITE,
		Exponent:  big.Float_Exponent(exponent),
	}
	test_float_initialize(&value)
	value.Mantissa.Count = big.Word_Count(count)
	value.Mantissa.Words[count-big.WORD_COUNT_INCREMENT] = TEST_COUNT_ONE
	test_workspace_float(t, value)
}

func test_workspace_float(t *testing.T, value big.Float) {
	t.Helper()
	program := compile_from(t, "")
	var execution execution_storage
	test_float_parse_workspace_initialize(&execution.Float_Parse)
	if value.Mantissa.Words == nil {
		test_float_initialize(&value)
	}
	execution.Float_Parse.Values.Result = big.Float_Rat_Result(value)
	_, _, status := template.Execute_Into(template.Execute_Input{
		Program:   template.Program_Unvalidated(program),
		Value:     template.Value_Unvalidated(template.Value_Nil()),
		Workspace: workspace_from(&execution),
	})
	testify.Equal_Values(t, template.STATUS_OK, status)
}

func test_execute_input_boundaries(t *testing.T) {
	t.Helper()
	empty := compile_from(t, "")
	var destination [TEST_COUNT_TWO]byte
	for size := TEST_COUNT_ONE; size <= len(destination); size++ {
		var execution execution_storage
		count, _, status := template.Execute_Into(template.Execute_Input{
			Destination: destination[:size],
			Program:     template.Program_Unvalidated(empty),
			Value:       template.Value_Unvalidated(template.Value_Nil()),
			Workspace:   workspace_from(&execution),
		})
		testify.Equal_Values(t, template.STATUS_OK, status)
		testify.Equal_Values(t, template.Output_Count(template.OUTPUT_SIZE_MINIMUM), count)
	}

	program := compile_from(t, "x")
	var two [TEST_COUNT_TWO]template.Function
	var execution execution_storage
	_, _, status := template.Execute_Into(template.Execute_Input{
		Program:   template.Program_Unvalidated(program),
		Value:     template.Value_Unvalidated(template.Value_Nil()),
		Functions: two[:],
		Workspace: workspace_from(&execution),
	})
	testify.Equal_Values(t, template.STATUS_FUNCTION_INVALID, status)
	var maximum [template.FUNCTION_COUNT_UNVALIDATED_MAXIMUM]template.Function
	execution = execution_storage{}
	_, _, status = template.Execute_Into(template.Execute_Input{
		Program:   template.Program_Unvalidated(program),
		Value:     template.Value_Unvalidated(template.Value_Nil()),
		Functions: maximum[:],
		Workspace: workspace_from(&execution),
	})
	testify.Equal_Values(t, template.STATUS_FUNCTION_INVALID, status)
	definition := compile_from(t, `x{{define "x"}}{{end}}`)
	test_execute_program_input(t, definition, template.STATUS_OK)
}

func test_execute_program_input(
	t *testing.T, program template.Program, expected template.Status,
) {
	t.Helper()
	var execution execution_storage
	var destination [template.OUTPUT_SIZE_MAXIMUM]byte
	_, _, status := template.Execute_Into(template.Execute_Input{
		Destination: destination[:], Program: template.Program_Unvalidated(program),
		Value:     template.Value_Unvalidated(template.Value_Nil()),
		Workspace: workspace_from(&execution),
	})
	testify.Equal_Values(t, expected, status)
}

func test_execute_output_maximum(t *testing.T) {
	t.Helper()
	var source, output [template.OUTPUT_SIZE_MAXIMUM]byte
	for index := range source {
		source[index] = 'x'
	}
	program := compile_from(t, string(source[:]))
	var execution execution_storage
	count, _, status := template.Execute_Into(template.Execute_Input{
		Destination: output[:], Program: template.Program_Unvalidated(program),
		Value:     template.Value_Unvalidated(template.Value_Nil()),
		Workspace: workspace_from(&execution),
	})
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal_Values(t, template.Output_Count(template.OUTPUT_SIZE_MAXIMUM), count)
}

func test_execute_diagnostic_positions(t *testing.T) {
	t.Helper()
	for prefix_size := TEST_COUNT_ONE; prefix_size <= TEST_COUNT_TWO; prefix_size++ {
		var prefix [TEST_COUNT_TWO]byte
		for index := range prefix_size {
			prefix[index] = 'x'
		}
		program := compile_from(t, string(prefix[:prefix_size])+"{{.Name}}")
		var execution execution_storage
		_, diagnostic, status := template.Execute_Into(template.Execute_Input{
			Program: template.Program_Unvalidated(program),
			Value: template.Value_Unvalidated(
				template.Value_Of_Integer(TEST_COUNT_ONE),
			),
			Workspace: workspace_from(&execution),
		})
		testify.Equal_Values(t, template.STATUS_EXECUTION_INVALID, status)
		testify.Equal_Values(
			t, template.Diagnostic_Position(prefix_size), diagnostic.Position,
		)
	}
	test_execute_maximum_diagnostic(t)
}

func test_execute_maximum_diagnostic(t *testing.T) {
	t.Helper()
	var source [template.SOURCE_SIZE_MAXIMUM]byte
	for index := range source {
		source[index] = 'x'
	}
	copy(source[:], []byte("{{.}}"))
	program := compile_from(t, string(source[:]))
	root := syntax_node(program, template.Node_Reference(program.Document.Root))
	action := syntax_node(program, template.Node_Reference(root.First_Child))
	action.Start = template.Node_Start(template.SOURCE_SIZE_MAXIMUM)
	action.End = template.Node_End(template.SOURCE_SIZE_MAXIMUM)
	pipe := syntax_node(program, template.Node_Reference(action.First_Child))
	pipe.Kind = template.NODE_TEXT
	var execution execution_storage
	_, diagnostic, status := template.Execute_Into(template.Execute_Input{
		Program:   template.Program_Unvalidated(program),
		Value:     template.Value_Unvalidated(template.Value_Nil()),
		Workspace: workspace_from(&execution),
	})
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)
	testify.Equal_Values(
		t, template.Diagnostic_Position(template.SOURCE_SIZE_MAXIMUM), diagnostic.Position,
	)
}

func test_compile_delimiter_boundaries(t *testing.T) {
	t.Helper()
	for _, input := range [...]template.Compile_Input{
		{Source: []byte("<.>"), Left: []byte("<"), Right: []byte(">")},
		{Source: []byte("<%.%>"), Left: []byte("<%"), Right: []byte("%>")},
	} {
		var workspace template.Syntax_Workspace
		input.Workspace = syntax_from(&workspace)
		_, _, status := template.Compile(input)
		testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	}
	var left, right [template.DELIMITER_SIZE_UNVALIDATED_MAXIMUM]byte
	var workspace template.Syntax_Workspace
	_, _, status := template.Compile(template.Compile_Input{
		Source: []byte("x"), Left: left[:], Right: right[:],
		Workspace: syntax_from(&workspace),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_CONFIGURATION_INVALID, status)
}

func test_compile_mode_boundaries(t *testing.T) {
	t.Helper()
	for _, mode := range [...]template.Mode{
		template.PARSE_COMMENTS, template.SKIP_FUNCTION_CHECK,
	} {
		var workspace template.Syntax_Workspace
		_, _, status := template.Compile(template.Compile_Input{
			Source: []byte("x"), Mode: mode, Workspace: syntax_from(&workspace),
		})
		testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	}
	var workspace template.Syntax_Workspace
	_, _, status := template.Compile(template.Compile_Input{
		Source: []byte("x"), Mode: template.Mode(bits.WORD_MAXIMUM),
		Workspace: syntax_from(&workspace),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_CONFIGURATION_INVALID, status)
}

func test_compile_diagnostic_boundaries(t *testing.T) {
	t.Helper()
	workspace := template.Syntax_Workspace{
		Node_Limit: template.Node_Limit_Storage{
			Value: template.Node_Limit_Stored(template.NODE_COUNT_INCREMENT),
		},
	}
	_, diagnostic, status := template.Compile(template.Compile_Input{
		Source: []byte("x"), Workspace: syntax_from(&workspace),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_CAPACITY_EXCEEDED, status)
	testify.Equal_Values(t, template.DIAGNOSTIC_CAPACITY_EXCEEDED, diagnostic.Code)

	for _, source := range [...]string{"<", "{{"} {
		workspace = template.Syntax_Workspace{}
		input := template.Compile_Input{Source: []byte(source)}
		if len(source) == TEST_COUNT_ONE {
			input.Left, input.Right = []byte("<"), []byte(">")
		}
		input.Workspace = syntax_from(&workspace)
		_, diagnostic, status = template.Compile(input)
		testify.Equal_Values(t, template.PARSE_STATUS_SYNTAX_INVALID, status)
		testify.Equal_Values(
			t, template.Parse_Diagnostic_Position(len(source)), diagnostic.Position,
		)
	}
	test_compile_maximum_diagnostic(t)
}

func test_compile_maximum_diagnostic(t *testing.T) {
	t.Helper()
	ending := [...]byte{'{', '{', 'i', 'f', ' ', '.', '}', '}'}
	var source [template.SOURCE_SIZE_MAXIMUM]byte
	limit := len(source) - len(ending)
	for position := range limit {
		source[position] = 'x'
	}
	copy(source[limit:], ending[:])
	var workspace template.Syntax_Workspace
	_, diagnostic, status := template.Compile(template.Compile_Input{
		Source: source[:], Workspace: syntax_from(&workspace),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_SYNTAX_INVALID, status)
	testify.Equal_Values(
		t, template.Parse_Diagnostic_Position(template.SOURCE_SIZE_MAXIMUM),
		diagnostic.Position,
	)
}

func test_execute_program_bounds(t *testing.T) {
	test_execute_literal_program_bounds(t)
	test_execute_lookup_program_bounds(t)
	test_execute_evaluation_program_bounds(t)
	test_execute_evaluation_limits(t)
}

func test_execute_evaluation_program_bounds(t *testing.T) {
	test_execute_node_record_bounds(t)
	test_execute_lookup_span_bounds(t)

	program := compile_from(t, "x")
	root := syntax_node(program, template.Node_Reference(program.Document.Root))
	text := syntax_node(program, template.Node_Reference(root.First_Child))
	text.End = template.Node_End(template.SOURCE_SIZE_MAXIMUM)
	_, status := execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)

	program = compile_from(t, "{{.}}")
	root = syntax_node(program, template.Node_Reference(program.Document.Root))
	action := syntax_node(program, template.Node_Reference(root.First_Child))
	action.First_Child = template.Node_First_Child(template.NO_NODE)
	_, status = execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)

	program = compile_from(t, "x")
	program.Source.Data.Value = make(
		template.Source_Storage_Value, template.SOURCE_SIZE_UNVALIDATED_MAXIMUM)
	_, status = execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)

	program = compile_from(t, "x")
	root = syntax_node(program, template.Node_Reference(program.Document.Root))
	text = syntax_node(program, template.Node_Reference(root.First_Child))
	text.Kind = template.NODE_BREAK
	_, status = execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)

	program = compile_from(t, "{{.}}")
	pipe := syntax_first_pipe(program)
	command := syntax_node(program, template.Node_Reference(pipe.First_Child))
	command.Kind = template.NODE_TEXT
	_, status = execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)

	program = compile_from(t, "{{.}}")
	pipe = syntax_first_pipe(program)
	pipe.First_Child = template.Node_First_Child(template.NO_NODE)
	_, status = execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)

	program = compile_from(t, `{{$x := 1}}{{$x = 2}}`)
	data := program.Source.Data.Value
	data[len(`{{$x := 1}}{{$`)] = 'y'
	_, status = execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_EXECUTION_INVALID, status)
	program = compile_from(t, `{{$x := 1}}{{$x = 2}}{{$x}}`)
	output, status := execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "2", output)

	test_execute_child_program_bounds(t)
}

func test_execute_node_record_bounds(t *testing.T) {
	t.Helper()
	program := compile_from(t, "xx")
	_, status := execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_OK, status)

	program = compile_from(t, "x")
	root := syntax_node(program, template.Node_Reference(program.Document.Root))
	root.First_Child = template.Node_First_Child(template.NODE_REFERENCE_MAXIMUM)
	_, status = execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)

	program = compile_from(t, "{{.}}")
	root = syntax_node(program, template.Node_Reference(program.Document.Root))
	root.First_Child = template.Node_First_Child(template.ROOT_NODE_COUNT)
	_, status = execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)

	program = compile_from(t, "x")
	root = syntax_node(program, template.Node_Reference(program.Document.Root))
	text := syntax_node(program, template.Node_Reference(root.First_Child))
	text.Next_Sibling = template.Node_Next_Sibling(template.ROOT_NODE_COUNT)
	_, status = execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)

	program = compile_from(t, "{{if .}}x{{else}}y{{end}}")
	root = syntax_node(program, template.Node_Reference(program.Document.Root))
	control := syntax_node(program, template.Node_Reference(root.First_Child))
	pipe := syntax_node(program, template.Node_Reference(control.First_Child))
	body := syntax_node(program, template.Node_Reference(pipe.Next_Sibling))
	alternate := syntax_node(program, template.Node_Reference(body.Next_Sibling))
	alternate.Kind = template.NODE_TEXT
	_, status = execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)
}

func test_execute_lookup_span_bounds(t *testing.T) {
	t.Helper()
	for _, kind := range [...]template.Node_Kind{
		template.NODE_FIELD, template.NODE_VARIABLE,
	} {
		program := compile_from(t, "{{.}}")
		term := syntax_node(
			program, template.Node_Reference(syntax_first_pipe(program).First_Child),
		)
		term = syntax_node(program, template.Node_Reference(term.First_Child))
		term.Kind = kind
		term.Value_Start = template.Node_Value_Start(len("{{"))
		term.Value_End = template.Node_Value_End(
			len("{{") + template.SOURCE_POSITION_INCREMENT,
		)
		_, status := execute_from(t, program, value_nil())
		expected := template.STATUS_EXECUTION_INVALID
		if kind == template.NODE_FIELD {
			expected = template.STATUS_PROGRAM_INVALID
		}
		testify.Equal_Values(t, expected, status)

		var maximum [template.SOURCE_SIZE_MAXIMUM]byte
		for index := range maximum {
			maximum[index] = 'x'
		}
		if kind == template.NODE_FIELD {
			maximum[template.SOURCE_SIZE_MINIMUM] = '.'
		}
		program.Source.Data.Value = maximum[:]
		term.Value_Start = template.Node_Value_Start(template.SOURCE_SIZE_MINIMUM)
		term.Value_End = template.Node_Value_End(len(maximum))
		_, status = execute_from(t, program, value_nil())
		testify.Equal_Values(t, template.STATUS_EXECUTION_INVALID, status)
	}
}

func test_execute_child_program_bounds(t *testing.T) {
	program := compile_from(t, `{{$x := .}}{{($x)}}`)
	root := syntax_node(program, template.Node_Reference(program.Document.Root))
	declaration := syntax_node(program, template.Node_Reference(root.First_Child))
	declaration_pipe := syntax_node(program, template.Node_Reference(declaration.First_Child))
	action := syntax_node(program, template.Node_Reference(declaration.Next_Sibling))
	pipe := syntax_node(program, template.Node_Reference(action.First_Child))
	command := syntax_node(program, template.Node_Reference(pipe.First_Child))
	inner_pipe := syntax_node(program, template.Node_Reference(command.First_Child))
	inner_pipe.First_Child = declaration_pipe.First_Child
	_, status := execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)

	program = compile_from(t, `{{(.).X}}`)
	_, status = execute_from(t, program, value_integer(TEST_COUNT_ONE))
	testify.Equal_Values(t, template.STATUS_EXECUTION_INVALID, status)
	program = compile_from(t, `{{(.).X}}`)
	pipe = syntax_first_pipe(program)
	command = syntax_node(program, template.Node_Reference(pipe.First_Child))
	chain := syntax_node(program, template.Node_Reference(command.First_Child))
	chain.Value_Start = template.Node_Value_Start(chain.Value_End)
	_, status = execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)

	program = compile_from(t, `{{range .}}{{else}}x{{end}}`)
	root = syntax_node(program, template.Node_Reference(program.Document.Root))
	range_node := syntax_node(program, template.Node_Reference(root.First_Child))
	pipe = syntax_node(program, template.Node_Reference(range_node.First_Child))
	body := syntax_node(program, template.Node_Reference(pipe.Next_Sibling))
	alternate := syntax_node(program, template.Node_Reference(body.Next_Sibling))
	alternate.Kind = template.NODE_TEXT
	_, status = execute_from(t, program, value_sequence([]template.Value{}))
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)
}

func test_execute_evaluation_limits(t *testing.T) {
	program := compile_from(t, `{{(.)}}`)
	pipe := syntax_first_pipe(program)
	command := syntax_node(program, template.Node_Reference(pipe.First_Child))
	term_reference := template.Node_Reference(command.First_Child)
	term := syntax_node(program, term_reference)
	inner_pipe_reference := term_reference
	if term.Kind == template.NODE_CHAIN {
		inner_pipe_reference = template.Node_Reference(term.First_Child)
	}
	inner_pipe := syntax_node(program, inner_pipe_reference)
	inner_command := syntax_node(program, template.Node_Reference(inner_pipe.First_Child))
	inner_command.First_Child = template.Node_First_Child(inner_pipe_reference)
	_, status := execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_LIMIT_EXCEEDED, status)

	program = compile_from(t, `{{print .}}`)
	pipe = syntax_first_pipe(program)
	command = syntax_node(program, template.Node_Reference(pipe.First_Child))
	identifier := syntax_node(program, template.Node_Reference(command.First_Child))
	argument_reference := template.Node_Reference(identifier.Next_Sibling)
	argument := syntax_node(program, argument_reference)
	argument.Next_Sibling = template.Node_Next_Sibling(argument_reference)
	_, status = execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_LIMIT_EXCEEDED, status)

	program = compile_from(
		t, `{{define "cycle"}}{{$x := .}}{{template "cycle" .}}{{end}}`+
			`{{template "cycle" .}}`,
	)
	_, status = execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_LIMIT_EXCEEDED, status)

	program = compile_from(t, "x")
	root := syntax_node(program, template.Node_Reference(program.Document.Root))
	text_reference := template.Node_Reference(root.First_Child)
	text := syntax_node(program, text_reference)
	text.Value_End = template.Node_Value_End(text.Value_Start)
	text.Next_Sibling = template.Node_Next_Sibling(text_reference)
	_, status = execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_LIMIT_EXCEEDED, status)
}

func test_execute_literal_program_bounds(t *testing.T) {
	source := []byte(`{{"ok"}}`)
	var syntax template.Syntax_Workspace
	program, _, parse_status := template.Compile(template.Compile_Input{
		Source: source, Workspace: syntax_from(&syntax),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, parse_status)
	source[len(source)-len(`k"}}`)] = '\\'
	_, execution_status := execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, execution_status)

	source = []byte(`{{'a'}}`)
	syntax = template.Syntax_Workspace{}
	program, _, parse_status = template.Compile(template.Compile_Input{
		Source: source, Workspace: syntax_from(&syntax),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, parse_status)
	source[len("{{'")] = '\\'
	_, execution_status = execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, execution_status)

	source = []byte(`{{1}}`)
	syntax = template.Syntax_Workspace{}
	program, _, parse_status = template.Compile(template.Compile_Input{
		Source: source, Workspace: syntax_from(&syntax),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, parse_status)
	source[len("{{")] = 'x'
	_, execution_status = execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_EXECUTION_INVALID, execution_status)

	source = []byte(`{{1+2i}}`)
	syntax = template.Syntax_Workspace{}
	program, _, parse_status = template.Compile(template.Compile_Input{
		Source: source, Workspace: syntax_from(&syntax),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, parse_status)
	source[len("{{1")] = 'i'
	_, execution_status = execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_EXECUTION_INVALID, execution_status)

	source = []byte(`{{$x := 1}}`)
	syntax = template.Syntax_Workspace{}
	program, _, parse_status = template.Compile(template.Compile_Input{
		Source: source, Workspace: syntax_from(&syntax),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, parse_status)
	source[len("{{$x ")] = '?'
	_, execution_status = execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, execution_status)
}

func test_execute_lookup_program_bounds(t *testing.T) {
	source := []byte(`{{template "x"}}`)
	var syntax template.Syntax_Workspace
	program, _, parse_status := template.Compile(template.Compile_Input{
		Source: source, Workspace: syntax_from(&syntax),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, parse_status)
	_, execution_status := execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_EXECUTION_INVALID, execution_status)
	source[len(`{{template "`)] = '\\'
	_, execution_status = execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, execution_status)

	source = []byte(`{{.Name}}`)
	syntax = template.Syntax_Workspace{}
	program, _, parse_status = template.Compile(template.Compile_Input{
		Source: source, Workspace: syntax_from(&syntax),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, parse_status)
	_, execution_status = execute_from(t, program, value_integer(TEST_COUNT_ONE))
	testify.Equal_Values(t, template.STATUS_EXECUTION_INVALID, execution_status)
	source[len("{{.")] = '.'
	_, execution_status = execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, execution_status)

	source = []byte(`{{$x := 1}}{{$x}}`)
	syntax = template.Syntax_Workspace{}
	program, _, parse_status = template.Compile(template.Compile_Input{
		Source: source, Workspace: syntax_from(&syntax),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, parse_status)
	source[len(source)-len("x}}")] = 'y'
	_, execution_status = execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_EXECUTION_INVALID, execution_status)

	source = []byte(`{{$x := .}}{{$x.Name}}`)
	syntax = template.Syntax_Workspace{}
	program, _, parse_status = template.Compile(template.Compile_Input{
		Source: source, Workspace: syntax_from(&syntax),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, parse_status)
	source[len(source)-len("Name}}")] = '.'
	_, execution_status = execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, execution_status)

	program = compile_from(
		t, `{{define "cycle"}}{{template "cycle"}}{{end}}`+
			`{{template "cycle"}}`,
	)
	_, execution_status = execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_LIMIT_EXCEEDED, execution_status)
}

func test_execute_value_bounds(t *testing.T) {
	program := compile_from(t, "x")
	invalid := template.Value_Nil()
	invalid.Kind = template.Value_Kind_Storage_Value(template.VALUE_FUNCTION + TEST_COUNT_ONE)
	_, execution_status := execute_from(t, program, template.Value(invalid))
	testify.Equal_Values(t, template.STATUS_VALUE_INVALID, execution_status)

	var leaves, branches [template.VALUE_COUNT_MAXIMUM]template.Value
	branches[len(branches)-TEST_COUNT_ONE] = value_sequence(leaves[:])
	_, execution_status = execute_from(t, program, value_sequence(branches[:]))
	testify.Equal_Values(t, template.STATUS_LIMIT_EXCEEDED, execution_status)

	cycle := [...]template.Value{value_nil()}
	cyclic := template.Value_Of_Sequence(cycle[:])
	cycle[template.VALUE_PAYLOAD_FIELD] = template.Value(cyclic)
	_, execution_status = execute_from(t, program, template.Value(cyclic))
	testify.Equal_Values(t, template.STATUS_LIMIT_EXCEEDED, execution_status)

	invalid_key := [...]template.Value{value_nil()}
	entries := [...]template.Field{{
		Key: template.Field_Key{
			Data: value_sequence(invalid_key[:]),
		},
		Value: template.Field_Value{Data: value_nil()},
	}}
	_, execution_status = execute_from(t, program, value_map(entries[:]))
	testify.Equal_Values(t, template.STATUS_VALUE_INVALID, execution_status)

	duplicate_fields := [...]template.Field{
		{Name: []byte("Name")},
		{Name: []byte("Name")},
	}
	_, execution_status = execute_from(t, program, value_object(duplicate_fields[:]))
	testify.Equal_Values(t, template.STATUS_VALUE_INVALID, execution_status)

	duplicate_entries := [...]template.Field{
		{Key: template.Field_Key{Data: value_integer(TEST_COUNT_ONE)}},
		{Key: template.Field_Key{Data: value_integer(TEST_COUNT_ONE)}},
	}
	_, execution_status = execute_from(t, program, value_map(duplicate_entries[:]))
	testify.Equal_Values(t, template.STATUS_VALUE_INVALID, execution_status)
}

func test_builtin_allocation(t *testing.T) {
	program := compile_from(
		t, `{{html "<&>"}}|{{printf "%s=%d" "x" 2}}|{{println "a" 2}}|{{.Items}}`,
	)
	items := [...]template.Value{value_integer(TEST_COUNT_ONE), value_integer(TEST_COUNT_TWO)}
	fields := [...]template.Field{{
		Name: []byte("Items"),
		Value: template.Field_Value{
			Data: value_sequence(items[:]),
		},
	}}
	var execution execution_storage
	var output [template.OUTPUT_SIZE_MAXIMUM]byte
	input := template.Execute_Input{
		Destination: output[:], Program: template.Program_Unvalidated(program),
		Value:     template.Value_Unvalidated(template.Value_Of_Object(fields[:])),
		Workspace: workspace_from(&execution),
	}
	var count template.Output_Count
	var status template.Status
	testify.Zero_Allocation(t, func() {
		count, _, status = template.Execute_Into(input)
	})
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "&lt;&amp;&gt;|x=2|a 2\n|[1 2]", string(output[:count]))
}

func test_execute_output_bounds(t *testing.T) {
	program := compile_from(t, `{{.Pad}}{{.Value}}`)
	var pad [template.OUTPUT_SIZE_MAXIMUM]byte
	for byte_index := range pad {
		pad[byte_index] = 'x'
	}
	const FLOAT_ONE_BITS = uint64(big.FLOAT_64_EXPONENT_BIAS) <<
		big.FLOAT_64_EXPONENT_SHIFT
	sequence := [...]template.Value{value_integer(TEST_COUNT_ONE)}
	for _, value := range []template.Value{
		value_text([]byte("x")),
		value_boolean(true),
		value_float(template.Float(FLOAT_ONE_BITS)),
		value_complex(template.Complex{
			Real: template.Complex_Real(FLOAT_ONE_BITS),
		}),
		value_sequence(sequence[:]),
	} {
		fields := [...]template.Field{
			{
				Name: []byte("Pad"),
				Value: template.Field_Value{
					Data: value_text(pad[:]),
				},
			},
			{Name: []byte("Value"), Value: template.Field_Value{Data: value}},
		}
		_, status := execute_from(t, program, value_object(fields[:]))
		testify.Equal_Values(t, template.STATUS_OUTPUT_TOO_LARGE, status)
	}

	program = compile_from(t, `{{.}}`)
	var text [template.OUTPUT_SIZE_MAXIMUM - TEST_COUNT_ONE]byte
	sequence = [...]template.Value{value_text(text[:])}
	_, status := execute_from(t, program, value_sequence(sequence[:]))
	testify.Equal_Values(t, template.STATUS_OUTPUT_TOO_LARGE, status)

	var key [template.OUTPUT_SIZE_MAXIMUM]byte
	entries := [...]template.Field{{
		Key: template.Field_Key{Data: value_text(key[:])},
		Value: template.Field_Value{
			Data: value_integer(TEST_COUNT_ONE),
		},
	}}
	_, status = execute_from(t, program, value_map(entries[:]))
	testify.Equal_Values(t, template.STATUS_OUTPUT_TOO_LARGE, status)
}

func test_execute_generated_bounds(t *testing.T) {
	control_one := [...]byte{TEST_COUNT_ONE}
	output, status := execute_from(
		t, compile_from(t, `{{printf "%q" .}}`), value_text(control_one[:]),
	)
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, `"\x01"`, output)

	var plain, ampersand, control [template.OUTPUT_SIZE_MAXIMUM]byte
	for index := range plain {
		plain[index] = 'x'
		ampersand[index] = '&'
		control[index] = TEST_COUNT_ONE
	}
	for _, one := range []struct {
		Name   string
		Source string
		Value  template.Value
	}{
		{
			Name:   "integer",
			Source: `{{html . | printf "%04096d" 1}}`,
			Value:  value_text(ampersand[:template.OUTPUT_SIZE_MAXIMUM/len("&amp;")]),
		},
		{
			Name:   "type",
			Source: `{{html . | printf "%T"}}`,
			Value:  value_text(plain[:template.OUTPUT_SIZE_MAXIMUM-TEST_COUNT_ONE]),
		},
		{
			Name: "quote", Source: `{{printf "%q" .}}`,
			Value: value_text(control[:]),
		},
		{
			Name: "html", Source: `{{html .}}`,
			Value: value_text(ampersand[:]),
		},
		{
			Name: "javascript", Source: `{{js .}}`,
			Value: value_text(control[:]),
		},
		{
			Name: "url query", Source: `{{urlquery .}}`,
			Value: value_text(ampersand[:]),
		},
		{
			Name: "result", Source: `{{html .}}`,
			Value: value_text(ampersand[:template.OUTPUT_SIZE_MAXIMUM/len("&amp;")+
				TEST_COUNT_ONE]),
		},
		{
			Name: "arena", Source: `{{html . | print}}`,
			Value: value_text(plain[:]),
		},
	} {
		t.Run(one.Name, func(t *testing.T) {
			_, status = execute_from(t, compile_from(t, one.Source), one.Value)
			testify.Equal_Values(t, template.STATUS_LIMIT_EXCEEDED, status)
		})
	}
}

func test_execute_fields(t *testing.T) {
	program := compile_from(t, "Hello {{.Name}}! {{if .Ready}}yes{{else}}no{{end}}")
	fields := [...]template.Field{
		{
			Name: []byte("Name"),
			Value: template.Field_Value{Data: trusted(
				template.Value_Unvalidated(template.Value_Of_Text([]byte("Ada"))),
			)},
		},
		{
			Name:  []byte("Ready"),
			Value: template.Field_Value{Data: value_boolean(true)},
		},
	}
	root := template.Value_Of_Object(fields[:])
	output, status := execute_from(t, program, template.Value(root))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "Hello Ada! yes", output)
}

func test_execute_control(t *testing.T) {
	program := compile_from(
		t,
		"{{with .User}}{{.}}{{else}}missing{{end}}:"+
			"{{range $index, $value := .Items}}"+
			"{{if eq $value 2}}{{continue}}{{end}}"+
			"{{$index}}={{$value}};{{if eq $value 3}}{{break}}{{end}}"+
			"{{else}}empty{{end}}",
	)
	items := [...]template.Value{
		value_integer(TEST_COUNT_ONE),
		value_integer(TEST_COUNT_TWO),
		value_integer(TEST_COUNT_THREE),
		value_integer(TEST_COUNT_FOUR),
	}
	fields := [...]template.Field{
		{
			Name: []byte("User"),
			Value: template.Field_Value{Data: trusted(
				template.Value_Unvalidated(template.Value_Of_Text([]byte("Ada"))),
			)},
		},
		{
			Name: []byte("Items"),
			Value: template.Field_Value{Data: trusted(
				template.Value_Unvalidated(template.Value_Of_Sequence(items[:])),
			)},
		},
	}
	output, status := execute_from(t, program, value_object(fields[:]))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "Ada:0=1;2=3;", output)
	test_execute_range_values(t)
}

func test_execute_range_values(t *testing.T) {
	t.Helper()
	_, status := execute_from(t, compile_from(t, `{{range true}}{{end}}`), value_nil())
	testify.Equal_Values(t, template.STATUS_EXECUTION_INVALID, status)
	program := compile_from(t, `{{range .}}{{.}}{{end}}`)
	_, status = execute_from(
		t, program,
		value_integer(template.Integer(template.VALUE_COUNT_UNVALIDATED_MAXIMUM)),
	)
	testify.Equal_Values(t, template.STATUS_LIMIT_EXCEEDED, status)
	output, status := execute_from(t, program, value_unsigned(TEST_COUNT_TWO))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "01", output)
	output, status = execute_from(t, program, value_integer(TEST_COUNT_TWO))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "01", output)

	one := [...]template.Value{value_integer(TEST_COUNT_SEVEN)}
	program = compile_from(t, `{{range .}}{{end}}`)
	_, status = execute_from(t, program, value_sequence(one[:]))
	testify.Equal_Values(t, template.STATUS_OK, status)
	var maximum [template.VALUE_COUNT_MAXIMUM]template.Value
	_, status = execute_from(t, program, value_sequence(maximum[:]))
	testify.Equal_Values(t, template.STATUS_OK, status)

	program = compile_from(t, `{{$value := 0}}{{range $value = .}}{{end}}{{$value}}`)
	output, status = execute_from(t, program, value_sequence(one[:]))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "7", output)
	program = compile_from(
		t, `{{$key := 0}}{{$value := 0}}`+
			`{{range $key, $value = .}}{{end}}{{$key}}/{{$value}}`,
	)
	output, status = execute_from(t, program, value_sequence(one[:]))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "0/7", output)
}

func test_execute_pipeline(t *testing.T) {
	program := compile_from(t, `{{$value := .Name | upper}}{{join "hello" $value}}`)
	functions := [...]template.Function{
		{Name: []byte("upper"), Call: upper},
		{Name: []byte("join"), Call: join},
		{Name: []byte("identity"), Call: identity},
	}
	fields := [...]template.Field{{
		Name:  []byte("Name"),
		Value: template.Field_Value{Data: value_text([]byte("ada"))},
	}}
	output, status := execute_functions_from(t, program, value_object(fields[:]), functions[:])
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "hello ADA", output)

	program = compile_from(t, `{{join "hello" (upper .Name)}}|{{call identity "ok"}}`)
	output, status = execute_functions_from(t, program, value_object(fields[:]), functions[:])
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "hello ADA|ok", output)

	program = compile_from(
		t, `{{$first := printf "%s" "first"}}`+
			`{{$second := printf "%s" "second"}}{{$first}}/{{$second}}`,
	)
	output, status = execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "first/second", output)

	program = compile_from(t, `{{call missing "ok"}}`)
	_, status = execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_FUNCTION_ABSENT, status)

	program = compile_from(t, `{{reject}}`)
	reject_function := [...]template.Function{{Name: []byte("reject"), Call: reject}}
	_, status = execute_functions_from(t, program, value_nil(), reject_function[:])
	testify.Equal_Values(t, template.STATUS_FUNCTION_INVALID, status)

	program = compile_from(t, `{{invalid_nested}}`)
	invalid_function := [...]template.Function{{
		Name: []byte("invalid_nested"), Call: invalid_nested,
	}}
	_, status = execute_functions_from(t, program, value_nil(), invalid_function[:])
	testify.Equal_Values(t, template.STATUS_FUNCTION_INVALID, status)
	test_variable_lookup_boundaries(t)
}

func test_variable_lookup_boundaries(t *testing.T) {
	t.Helper()
	program := compile_from(t, `{{$}}`)
	output, status := execute_from(t, program, value_integer(TEST_COUNT_SEVEN))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "7", output)

	declaration := [...]byte{'<', '$', ':', '=', '.', '>'}
	var source [template.SOURCE_SIZE_MAXIMUM]byte
	position := template.SOURCE_SIZE_MINIMUM
	for range template.SYNTAX_VARIABLE_COUNT_MAXIMUM {
		copy(source[position:], declaration[:])
		position += len(declaration)
	}
	for position < len(source) {
		source[position] = 'x'
		position++
	}
	var syntax template.Syntax_Workspace
	program, _, parse_status := template.Compile(template.Compile_Input{
		Source: source[:], Left: []byte("<"), Right: []byte(">"),
		Workspace: syntax_from(&syntax),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, parse_status)
	root := syntax_node(program, template.Node_Reference(program.Document.Root))
	var action *template.Node
	for reference := template.Node_Reference(root.First_Child); reference != template.NO_NODE; {
		node := syntax_node(program, reference)
		if node.Kind == template.NODE_ACTION {
			action = node
		}
		reference = template.Node_Reference(node.Next_Sibling)
	}
	testify.True(t, action != nil)
	pipe := syntax_node(program, template.Node_Reference(action.First_Child))
	source[pipe.Value_Start] = '='
	pipe.Value_End = template.Node_Value_End(
		int(pipe.Value_Start) + template.SOURCE_POSITION_INCREMENT,
	)
	output, status = execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "xxxx", output)
}

func test_execute_templates(t *testing.T) {
	program := compile_from(
		t,
		`root {{template "card" .Name}} `+
			`{{block "footer" .Footer}}default {{.}}{{end}}`+
			`{{define "card"}}card {{.}}{{end}}`,
	)
	fields := [...]template.Field{
		{
			Name: []byte("Name"),
			Value: template.Field_Value{Data: trusted(
				template.Value_Unvalidated(template.Value_Of_Text([]byte("Ada"))),
			)},
		},
		{
			Name: []byte("Footer"),
			Value: template.Field_Value{Data: trusted(
				template.Value_Unvalidated(template.Value_Of_Text([]byte("done"))),
			)},
		},
	}
	output, status := execute_from(t, program, value_object(fields[:]))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "root card Ada default done", output)
}

func test_execute_builtins(t *testing.T) {
	program := compile_from(
		t,
		`{{and 0 (missing)}}|{{or 1 (missing)}}|`+
			`{{lt 1 2}}|{{le 2 2}}|{{gt 3 2}}|{{ge 3 3}}|`+
			`{{not false}}|{{ne 1 2}}|{{eq 0.0 -0.0}}|{{eq 1.0 2.0}}|`+
			`{{eq 1+2i 1+2i}}|`+
			`{{index .Items 1}}|{{index .Lookup "x"}}|`+
			`{{len (slice .Items 1 3)}}|{{call .Identity "ok"}}`,
	)
	items := [...]template.Value{
		value_text([]byte("a")),
		value_text([]byte("b")),
		value_text([]byte("c")),
	}
	entries := [...]template.Field{{
		Key: template.Field_Key{Data: value_text([]byte("x"))},
		Value: template.Field_Value{Data: trusted(
			template.Value_Unvalidated(template.Value_Of_Integer(TEST_COUNT_SEVEN)),
		)},
	}}
	fields := [...]template.Field{
		{
			Name: []byte("Items"),
			Value: template.Field_Value{
				Data: value_sequence(items[:]),
			},
		},
		{
			Name: []byte("Lookup"),
			Value: template.Field_Value{
				Data: value_map(entries[:]),
			},
		},
		{
			Name: []byte("Identity"),
			Value: template.Field_Value{
				Data: value_function(identity),
			},
		},
	}
	output, status := execute_from(t, program, value_object(fields[:]))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "0|1|true|true|true|true|true|true|true|false|true|b|7|2|ok", output)
	test_execute_text_order_boundaries(t)
	test_execute_builtin_refusals(t)
}

func test_execute_text_order_boundaries(t *testing.T) {
	t.Helper()
	var maximum [template.OUTPUT_SIZE_MAXIMUM]byte
	texts := [...]template.Text{
		nil, maximum[:TEST_COUNT_ONE], maximum[:TEST_COUNT_TWO], maximum[:],
	}
	fields := [...]template.Field{
		{Name: []byte("Left")}, {Name: []byte("Right")},
	}
	for _, source := range [...]string{`{{lt .Left .Right}}`, `{{eq .Left .Right}}`} {
		program := compile_from(t, source)
		for rotation := range texts {
			fields[TEST_COUNT_ZERO].Value.Data = trusted(
				template.Value_Unvalidated(template.Value_Of_Text(texts[rotation])),
			)
			right := texts[(rotation+TEST_COUNT_ONE)%len(texts)]
			fields[TEST_COUNT_ONE].Value.Data = value_text(right)

			_, status := execute_from(t, program, value_object(fields[:]))
			testify.Equal_Values(t, template.STATUS_OK, status)
		}
	}
}

func test_execute_builtin_refusals(t *testing.T) {
	t.Helper()
	var status template.Status
	for _, source := range []string{
		`{{and}}`, `{{or}}`, `{{eq 1}}`, `{{ne 1}}`, `{{not}}`,
	} {
		_, status = execute_from(t, compile_from(t, source), value_nil())
		testify.Equal_Values(t, template.STATUS_FUNCTION_INVALID, status)
	}
	for _, one := range []struct {
		Source string
		Status template.Status
	}{
		{Source: `{{call}}`, Status: template.STATUS_FUNCTION_INVALID},
		{Source: `{{call 1}}`, Status: template.STATUS_EXECUTION_INVALID},
		{Source: `{{len}}`, Status: template.STATUS_FUNCTION_INVALID},
		{Source: `{{len 1}}`, Status: template.STATUS_EXECUTION_INVALID},
		{Source: `{{lt 1}}`, Status: template.STATUS_FUNCTION_INVALID},
		{Source: `{{lt 1 "x"}}`, Status: template.STATUS_EXECUTION_INVALID},
		{Source: `{{index "x"}}`, Status: template.STATUS_FUNCTION_INVALID},
		{Source: `{{index 1 0}}`, Status: template.STATUS_EXECUTION_INVALID},
		{Source: `{{slice}}`, Status: template.STATUS_FUNCTION_INVALID},
		{Source: `{{slice 1 0}}`, Status: template.STATUS_EXECUTION_INVALID},
	} {
		_, status = execute_from(t, compile_from(t, one.Source), value_nil())
		testify.Equal_Values(t, one.Status, status)
	}
	test_slice_capacity_refusal(t)
	test_index_slice_boundaries(t)
}

func test_index_slice_boundaries(t *testing.T) {
	t.Helper()
	var maximum [template.VALUE_COUNT_MAXIMUM]template.Value
	for index := range maximum {
		maximum[index] = value_integer(template.Integer(index))
	}
	fields := [...]template.Field{
		{Name: []byte("Values")},
		{Name: []byte("Index")},
		{Name: []byte("End")},
	}
	index_program := compile_from(t, `{{index .Values .Index}}`)
	for _, one := range []struct {
		Values []template.Value
		Index  template.Integer
		Status template.Status
	}{
		{Values: maximum[:TEST_COUNT_ZERO], Status: template.STATUS_EXECUTION_INVALID},
		{Values: maximum[:TEST_COUNT_ONE], Status: template.STATUS_OK},
		{
			Values: maximum[:TEST_COUNT_THREE], Index: TEST_COUNT_TWO,
			Status: template.STATUS_OK,
		},
		{
			Values: maximum[:],
			Index:  template.Integer(template.VALUE_COUNT_MAXIMUM - TEST_COUNT_ONE),
			Status: template.STATUS_OK,
		},
	} {
		fields[TEST_COUNT_ZERO].Value.Data = value_sequence(one.Values)

		fields[TEST_COUNT_ONE].Value.Data = value_integer(one.Index)

		_, status := execute_from(t, index_program, value_object(fields[:]))
		testify.Equal_Values(t, one.Status, status)
	}

	slice_program := compile_from(t, `{{len (slice .Values 0 .End)}}`)
	for _, count := range [...]int{
		TEST_COUNT_ZERO, TEST_COUNT_ONE, TEST_COUNT_TWO,
		template.VALUE_COUNT_MAXIMUM,
	} {
		fields[TEST_COUNT_ZERO].Value.Data = value_sequence(maximum[:count:count])

		fields[TEST_COUNT_TWO].Value.Data = value_integer(template.Integer(count))

		_, status := execute_from(t, slice_program, value_object(fields[:]))
		testify.Equal_Values(t, template.STATUS_OK, status)
	}
}

func test_slice_capacity_refusal(t *testing.T) {
	t.Helper()
	const HOSTILE_COUNT = template.VALUE_COUNT_UNVALIDATED_MAXIMUM +
		(template.VALUE_COUNT_UNVALIDATED_MAXIMUM - template.VALUE_COUNT_MAXIMUM)
	var values [HOSTILE_COUNT]template.Value
	fields := [...]template.Field{
		{
			Name: []byte("Values"),
			Value: template.Field_Value{
				Data: value_sequence(values[:TEST_COUNT_ONE]),
			},
		},
		{
			Name: []byte("End"),
			Value: template.Field_Value{
				Data: value_integer(HOSTILE_COUNT),
			},
		},
	}
	program := compile_from(t, `{{slice .Values 0 .End}}`)
	var execution execution_storage
	var output [template.OUTPUT_SIZE_MAXIMUM]byte
	input := template.Execute_Input{
		Destination: output[:], Program: template.Program_Unvalidated(program),
		Value:     template.Value_Unvalidated(template.Value_Of_Object(fields[:])),
		Workspace: workspace_from(&execution),
	}
	var status template.Status
	testify.Zero_Allocation(t, func() {
		_, _, status = template.Execute_Into(input)
	})
	testify.Equal_Values(t, template.STATUS_EXECUTION_INVALID, status)
}

func test_execute_numbers(t *testing.T) {
	for _, one := range []struct {
		Source string
		Want   string
	}{
		{Source: `{{1.5}}`, Want: "1.5"},
		{Source: `{{2i}}`, Want: "(0+2i)"},
		{Source: `{{1+2i}}`, Want: "(1+2i)"},
		{Source: `{{0x1p+2}}`, Want: "4"},
		{Source: `{{print 0xef}}`, Want: "239"},
		{Source: `{{print 1e9}}`, Want: "1e+09"},
		{Source: `{{print 073i}}`, Want: "(0+73i)"},
		{Source: `{{print -1.2+4.2i}}`, Want: "(-1.2+4.2i)"},
		{Source: `{{print +0x1.ep+2}}`, Want: "7.5"},
	} {
		t.Run(
			one.Source,
			func(t *testing.T) {
				program := compile_from(t, one.Source)
				output, status := execute_from(t, program, value_nil())
				testify.Equal_Values(t, template.STATUS_OK, status)
				testify.Equal(t, one.Want, output)
			},
		)
	}
	test_execute_numeric_boundaries(t)
}

func test_execute_numeric_boundaries(t *testing.T) {
	t.Helper()
	output_program := compile_from(t, `{{.}}`)
	for _, value := range [...]template.Value{
		value_float(template.Float(TEST_COUNT_ONE)),
		value_float(template.Float(TEST_COUNT_TWO)),
		value_float(template.Float(bits.WORD_64_MAXIMUM)),
		value_complex(template.Complex{
			Real: TEST_COUNT_ONE, Imaginary: TEST_COUNT_ONE,
		}),
		value_complex(template.Complex{
			Real: TEST_COUNT_TWO, Imaginary: TEST_COUNT_TWO,
		}),
		value_complex(template.Complex{
			Real:      template.Complex_Real(bits.WORD_64_MAXIMUM),
			Imaginary: template.Complex_Imaginary(bits.WORD_64_MAXIMUM),
		}),
	} {
		_, status := execute_from(t, output_program, value)
		testify.Equal_Values(t, template.STATUS_OK, status)
	}

	order_program := compile_from(t, `{{lt .Left .Right}}`)
	fields := [...]template.Field{{Name: []byte("Left")}, {Name: []byte("Right")}}
	integers := [...]template.Value{
		value_integer(template.Integer(bits.INTEGER_64_MINIMUM)),
		value_integer(-TEST_COUNT_ONE),
		value_integer(template.Integer(bits.INTEGER_64_MAXIMUM)),
	}
	for rotation := range integers {
		fields[TEST_COUNT_ZERO].Value.Data = integers[rotation]
		right := (rotation + TEST_COUNT_ONE) % len(integers)
		fields[TEST_COUNT_ONE].Value.Data = integers[right]
		_, status := execute_from(t, order_program, value_object(fields[:]))
		testify.Equal_Values(t, template.STATUS_OK, status)
	}
	unsigned := [...]template.Value{
		value_unsigned(template.Unsigned(bits.WORD_64_MINIMUM)),
		value_unsigned(template.Unsigned(bits.WORD_64_MAXIMUM)),
	}
	for rotation := range unsigned {
		fields[TEST_COUNT_ZERO].Value.Data = unsigned[rotation]
		right := (rotation + TEST_COUNT_ONE) % len(unsigned)
		fields[TEST_COUNT_ONE].Value.Data = unsigned[right]
		_, status := execute_from(t, order_program, value_object(fields[:]))
		testify.Equal_Values(t, template.STATUS_OK, status)
	}
	for _, integer := range [...]template.Integer{
		template.Integer(bits.INTEGER_64_MINIMUM), template.Integer(-TEST_COUNT_ONE),
		template.Integer(TEST_COUNT_ZERO), template.Integer(TEST_COUNT_ONE),
		template.Integer(TEST_COUNT_TWO), template.Integer(bits.INTEGER_64_MAXIMUM),
	} {
		fields[TEST_COUNT_ZERO].Value.Data = value_integer(integer)
		for _, magnitude := range [...]template.Unsigned{TEST_COUNT_ZERO, TEST_COUNT_TWO} {
			fields[TEST_COUNT_ONE].Value.Data = trusted(
				template.Value_Unvalidated(template.Value_Of_Unsigned(magnitude)),
			)
			_, status := execute_from(t, order_program, value_object(fields[:]))
			testify.Equal_Values(t, template.STATUS_OK, status)
		}
	}
}

func test_execute_map_order(t *testing.T) {
	program := compile_from(t, `{{range $key, $value := .}}{{$key}}{{$value}}{{end}}`)
	entries := [...]template.Field{
		{
			Key: template.Field_Key{Data: value_text([]byte("c"))},
			Value: template.Field_Value{
				Data: value_integer(TEST_COUNT_THREE),
			},
		},
		{
			Key: template.Field_Key{Data: value_text([]byte("a"))},
			Value: template.Field_Value{
				Data: value_integer(TEST_COUNT_ONE),
			},
		},
		{
			Key: template.Field_Key{Data: value_text([]byte("b"))},
			Value: template.Field_Value{
				Data: value_integer(TEST_COUNT_TWO),
			},
		},
	}
	output, status := execute_from(t, program, value_map(entries[:]))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "a1b2c3", output)

	var maximum [template.RANGE_MAP_ENTRY_COUNT_MAXIMUM]template.Field
	for index := range maximum {
		key := len(maximum) - index - TEST_COUNT_ONE
		item := template.Value_Of_Boolean(false)
		if index == len(maximum)-TEST_COUNT_TWO {
			item = template.Value_Of_Boolean(true)
		}
		maximum[index] = template.Field{
			Key: template.Field_Key{
				Data: value_integer(template.Integer(key)),
			},
			Value: template.Field_Value{Data: template.Value(item)},
		}
	}
	program = compile_from(t, `{{range .}}{{if .}}{{break}}{{end}}{{end}}`)
	_, status = execute_from(t, program, value_map(maximum[:]))
	testify.Equal_Values(t, template.STATUS_OK, status)
	test_execute_float_map_order(t, program)
}

// Rotation makes each binary64 boundary serve as candidate and prior selection.
func test_execute_float_map_order(t *testing.T, program template.Program) {
	t.Helper()
	encodings := [...]template.Float{
		template.Float(bits.WORD_64_MINIMUM),
		template.Float(bits.WORD_64_MINIMUM + TEST_COUNT_ONE),
		template.Float(bits.WORD_64_MINIMUM + TEST_COUNT_TWO),
		template.Float(bits.WORD_64_MAXIMUM - TEST_COUNT_ONE),
		template.Float(bits.WORD_64_MAXIMUM),
	}
	var entries [len(encodings)]template.Field
	for rotation := range encodings {
		for index := range entries {
			encoding := encodings[(index+rotation)%len(encodings)]
			entries[index] = template.Field{
				Key: template.Field_Key{Data: trusted(
					template.Value_Unvalidated(
						template.Value_Of_Float(encoding),
					),
				)},
				Value: template.Field_Value{Data: value_nil()},
			}
		}
		_, status := execute_from(t, program, value_map(entries[:]))
		testify.Equal_Values(t, template.STATUS_OK, status)
	}
}

func test_execute_format_builtins(t *testing.T) {
	program := compile_from(
		t,
		`{{html "<&>"}}|{{js "\"<&"}}|{{urlquery "a b&"}}|`+
			`{{print "a" "b" 1 2}}|{{printf "%s=%d" "x" 2}}|{{println "a" 2}}`+
			`|{{printf "%04x" 127}}|{{printf "%T" 0xef}}|{{print nil}}`+
			`|{{js "\u2028"}}|{{printf "%#q" "zeroArgs"}}`,
	)
	output, status := execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(
		t,
		"&lt;&amp;&gt;|\\\"\\u003C\\u0026|a+b%26|ab1 2|x=2|a 2\n"+
			"|007f|int|<nil>|\\u2028|`zeroArgs`",
		output,
	)
	for _, one := range []struct {
		Source string
		Status template.Status
	}{
		{Source: `{{printf "%"}}`, Status: template.STATUS_EXECUTION_INVALID},
		{Source: `{{printf "%s" 1}}`, Status: template.STATUS_EXECUTION_INVALID},
		{Source: `{{printf "%04097d" 1}}`, Status: template.STATUS_LIMIT_EXCEEDED},
	} {
		_, status = execute_from(t, compile_from(t, one.Source), value_nil())
		testify.Equal_Values(t, one.Status, status)
	}
	test_execute_format_boundaries(t)
}

func test_execute_format_boundaries(t *testing.T) {
	t.Helper()
	program := compile_from(t, `{{printf "%1d|%2d|%b|%o|%X" 1 2 2 8 255}}`)
	output, status := execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "1| 2|10|10|FF", output)

	program = compile_from(t, `{{printf "%T|%T|%T" nil true .}}`)
	output, status = execute_from(t, program, value_function(identity))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "<nil>|bool|template.Function", output)

	for _, source := range [...]string{
		`{{printf "%\x00" 1}}`, `{{printf "%\x01" 1}}`,
		`{{printf "%\x02" 1}}`, `{{printf "%\xff" 1}}`,
	} {
		_, status = execute_from(t, compile_from(t, source), value_nil())
		testify.Equal_Values(t, template.STATUS_EXECUTION_INVALID, status)
	}

	program = compile_from(t, "{{printf \"%#q\" \"a`b\"}}")
	output, status = execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "\"a`b\"", output)
	test_quote_backquote_boundaries(t)
	test_url_query_byte_boundaries(t)
	test_escape_text_boundaries(t)

	quoted_controls := [...]byte{
		byte(bits.WORD_8_MINIMUM), byte(bits.WORD_8_MINIMUM + TEST_COUNT_ONE),
		byte(bits.WORD_8_MINIMUM + TEST_COUNT_TWO),
		' ' - TEST_COUNT_TWO, ' ' - TEST_COUNT_ONE,
		byte(utf8.CHARACTER_SELF - TEST_COUNT_ONE),
	}
	program = compile_from(t, `{{printf "%q" .}}`)
	output, status = execute_from(t, program, value_text(quoted_controls[:]))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, `"\x00\x01\x02\x1E\x1F\x7F"`, output)

	var format [template.OUTPUT_SIZE_MAXIMUM]byte
	format[TEST_COUNT_ZERO] = '%'
	for index := TEST_COUNT_ONE; index < len(format)-TEST_COUNT_ONE; index++ {
		format[index] = '#'
	}
	format[len(format)-TEST_COUNT_ONE] = '%'
	fields := [...]template.Field{{
		Name: []byte("Format"),
		Value: template.Field_Value{
			Data: value_text(format[:]),
		},
	}}
	program = compile_from(t, `{{printf .Format}}`)
	output, status = execute_from(t, program, value_object(fields[:]))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "%", output)
}

func test_quote_backquote_boundaries(t *testing.T) {
	t.Helper()
	program := compile_from(t, `{{printf "%#q" .}}`)
	var maximum [template.OUTPUT_SIZE_MAXIMUM]byte
	for index := range maximum {
		maximum[index] = 'x'
	}
	texts := [...]template.Text{
		nil, maximum[:TEST_COUNT_ONE], maximum[:TEST_COUNT_TWO], maximum[:],
	}
	for index := range texts {
		_, status := execute_from(t, program, value_text(texts[index]))
		if index == len(texts)-TEST_COUNT_ONE {
			testify.Equal_Values(t, template.STATUS_LIMIT_EXCEEDED, status)
			continue
		}
		testify.Equal_Values(t, template.STATUS_OK, status)
	}
}

func test_url_query_byte_boundaries(t *testing.T) {
	t.Helper()
	program := compile_from(t, `{{urlquery .}}`)
	value := [...]byte{
		byte(bits.WORD_8_MINIMUM), byte(bits.WORD_8_MINIMUM + TEST_COUNT_ONE),
		byte(bits.WORD_8_MINIMUM + TEST_COUNT_TWO),
		byte(bits.WORD_8_MAXIMUM - TEST_COUNT_ONE), byte(bits.WORD_8_MAXIMUM),
	}
	output, status := execute_from(t, program, value_text(value[:]))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "%00%01%02%FE%FF", output)
}

func test_escape_text_boundaries(t *testing.T) {
	t.Helper()
	values := [...]byte{
		byte(bits.WORD_8_MINIMUM), byte(bits.WORD_8_MINIMUM + TEST_COUNT_TWO),
		byte(utf8.CHARACTER_SELF - utf8.CHARACTER_SIZE_MINIMUM),
	}
	texts := [...]template.Text{
		nil, values[:TEST_COUNT_ONE], values[:TEST_COUNT_TWO], values[:],
	}
	for _, source := range [...]string{`{{html .}}`, `{{js .}}`, `{{urlquery .}}`} {
		program := compile_from(t, source)
		for index := range texts {
			_, status := execute_from(t, program, value_text(texts[index]))
			testify.Equal_Values(t, template.STATUS_OK, status)
		}
	}
	var maximum [utf8.CHARACTER_SIZE_MAXIMUM]byte
	size := utf8.Encode_Character(maximum[:], utf8.RUNE_MAX)
	program := compile_from(t, `{{js .}}`)
	_, status := execute_from(t, program, value_text(maximum[:size]))
	testify.Equal_Values(t, template.STATUS_OK, status)
}

func test_execute_composite_output(t *testing.T) {
	program := compile_from(t, `<{{.}}>`)
	output, status := execute_from(t, program, value_sequence(nil))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "<[]>", output)

	sequence := [...]template.Value{
		value_integer(-TEST_COUNT_ONE),
		value_integer(-TEST_COUNT_TWO),
		value_integer(-TEST_COUNT_THREE),
	}
	output, status = execute_from(t, program, value_sequence(sequence[:]))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "<[-1 -2 -3]>", output)

	entries := [...]template.Field{
		{
			Key: template.Field_Key{Data: trusted(
				template.Value_Unvalidated(template.Value_Of_Text([]byte("two"))),
			)},
			Value: template.Field_Value{
				Data: value_integer(TEST_COUNT_TWENTY_TWO),
			},
		},
	}
	output, status = execute_from(t, program, value_map(entries[:]))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "<map[two:22]>", output)

	fields := [...]template.Field{
		{
			Name: []byte("First"),
			Value: template.Field_Value{
				Data: value_integer(TEST_COUNT_SEVEN),
			},
		},
		{
			Name: []byte("Second"),
			Value: template.Field_Value{
				Data: value_text([]byte("seven")),
			},
		},
	}
	output, status = execute_from(t, program, value_object(fields[:]))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "<{7 seven}>", output)

	inner := [...]template.Value{value_integer(TEST_COUNT_ONE)}
	outer := [...]template.Value{value_sequence(inner[:])}
	output, status = execute_from(t, program, value_sequence(outer[:]))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "<[[1]]>", output)

	var maximum [template.VALUE_COUNT_MAXIMUM]template.Value
	_, status = execute_from(t, program, value_sequence(maximum[:]))
	testify.Equal_Values(t, template.STATUS_OUTPUT_TOO_LARGE, status)
}

func upper(input template.Function_Input) (result template.Function_Result) {
	if len(input.Arguments) != TEST_COUNT_ONE {
		return template.Function_Result{Status: template.FUNCTION_STATUS_INVALID}
	}
	text, status := template.Value_As_Text(input.Arguments[TEST_COUNT_ZERO])
	if status != template.VALUE_TEXT_STATUS_OK {
		return template.Function_Result{Status: template.FUNCTION_STATUS_INVALID}
	}
	if string(text) != "ada" {
		return template.Function_Result{Status: template.FUNCTION_STATUS_INVALID}
	}
	return template.Function_Result{
		Value:  template.Value_Unvalidated(template.Value_Of_Text([]byte("ADA"))),
		Status: template.FUNCTION_STATUS_OK,
	}
}

func join(input template.Function_Input) (result template.Function_Result) {
	if len(input.Arguments) != TEST_COUNT_TWO {
		return template.Function_Result{Status: template.FUNCTION_STATUS_INVALID}
	}
	left, left_status := template.Value_As_Text(input.Arguments[TEST_COUNT_ZERO])
	right, right_status := template.Value_As_Text(input.Arguments[TEST_COUNT_ONE])
	if left_status != template.VALUE_TEXT_STATUS_OK {
		return template.Function_Result{Status: template.FUNCTION_STATUS_INVALID}
	}
	if right_status != template.VALUE_TEXT_STATUS_OK {
		return template.Function_Result{Status: template.FUNCTION_STATUS_INVALID}
	}
	if string(left) != "hello" {
		return template.Function_Result{Status: template.FUNCTION_STATUS_INVALID}
	}
	if string(right) != "ADA" {
		return template.Function_Result{Status: template.FUNCTION_STATUS_INVALID}
	}
	return template.Function_Result{
		Value:  template.Value_Unvalidated(template.Value_Of_Text([]byte("hello ADA"))),
		Status: template.FUNCTION_STATUS_OK,
	}
}

func identity(input template.Function_Input) (result template.Function_Result) {
	if len(input.Arguments) != TEST_COUNT_ONE {
		return template.Function_Result{Status: template.FUNCTION_STATUS_INVALID}
	}
	return template.Function_Result{
		Value:  template.Value_Unvalidated(input.Arguments[TEST_COUNT_ZERO]),
		Status: template.FUNCTION_STATUS_OK,
	}
}

// Gives one fixed sequence so a template can index a collection by any value.
func listing(template.Function_Input) (result template.Function_Result) {
	items := []template.Value{value_integer(TEST_COUNT_ZERO), value_integer(TEST_COUNT_ONE)}
	return template.Function_Result{
		Value:  template.Value_Unvalidated(value_sequence(items)),
		Status: template.FUNCTION_STATUS_OK,
	}
}

// Gives one fixed map so a template can index entries by any key.
func mapping(template.Function_Input) (result template.Function_Result) {
	entries := []template.Field{
		{Key: template.Field_Key{Data: value_integer(TEST_COUNT_ZERO)}},
		{Key: template.Field_Key{Data: value_text([]byte("a"))}},
	}
	return template.Function_Result{
		Value:  template.Value_Unvalidated(value_map(entries)),
		Status: template.FUNCTION_STATUS_OK,
	}
}

func reject(template.Function_Input) (result template.Function_Result) {
	return template.Function_Result{Status: template.FUNCTION_STATUS_INVALID}
}

func invalid_nested(template.Function_Input) (result template.Function_Result) {
	invalid_kind := template.VALUE_FUNCTION + template.VALUE_KIND_INCREMENT
	values := [...]template.Value{{
		Kind: template.Value_Kind_Storage_Value(invalid_kind),
	}}
	return template.Function_Result{
		Value:  template.Value_Unvalidated(template.Value_Of_Sequence(values[:])),
		Status: template.FUNCTION_STATUS_OK,
	}
}

func test_standard_library_execution(t *testing.T) {
	for _, one := range []struct {
		Name   string
		Source string
		Want   string
	}{
		{Name: "print", Source: `{{print "hello, print"}}`, Want: "hello, print"},
		{Name: "print integers", Source: `{{print 1 2 3}}`, Want: "1 2 3"},
		{Name: "print nil", Source: `{{print nil}}`, Want: "<nil>"},
		{Name: "println", Source: `{{println 1 2 3}}`, Want: "1 2 3\n"},
		{Name: "printf integer", Source: `{{printf "%04x" 127}}`, Want: "007f"},
		{Name: "printf float", Source: `{{printf "%g" 3.5}}`, Want: "3.5"},
		{Name: "printf complex", Source: `{{printf "%g" 1+7i}}`, Want: "(1+7i)"},
		{Name: "printf text", Source: `{{printf "%s" "hello"}}`, Want: "hello"},
		{Name: "printf quote", Source: `{{printf "%#q" "zeroArgs"}}`, Want: "`zeroArgs`"},
		{
			Name: "html", Source: `{{html "<script>alert(\"XSS\");</script>"}}`,
			Want: "&lt;script&gt;alert(&#34;XSS&#34;);&lt;/script&gt;",
		},
		{Name: "js", Source: `{{js .}}`, Want: `It\'d be nice.`},
		{
			Name: "url query", Source: `{{"http://www.example.org/"|urlquery}}`,
			Want: "http%3A%2F%2Fwww.example.org%2F",
		},
		{Name: "not", Source: `{{not true}} {{not false}}`, Want: "false true"},
	} {
		t.Run(
			one.Name,
			func(t *testing.T) {
				program := compile_from(t, one.Source)
				value := template.Value_Nil()
				if one.Name == "js" {
					value = template.Nil_Value(
						template.Value_Of_Text([]byte("It'd be nice.")),
					)
				}
				output, status := execute_from(t, program, template.Value(value))
				testify.Equal_Values(t, template.STATUS_OK, status)
				testify.Equal(t, one.Want, output)
			},
		)
	}
}

func test_standard_library_value_execution(t *testing.T) {
	t.Helper()
	fields := [...]template.Field{
		{
			Name: []byte("Name"),
			Value: template.Field_Value{Data: trusted(
				template.Value_Unvalidated(template.Value_Of_Text([]byte("Ada"))),
			)},
		},
		{
			Name:  []byte("Ready"),
			Value: template.Field_Value{Data: value_boolean(true)},
		},
	}
	compare_execution(
		t, `{{.Name}}|{{if .Ready}}yes{{end}}`, value_object(fields[:]), "Ada|yes",
	)
	values := [...]template.Value{
		value_integer(TEST_COUNT_THREE),
		value_integer(TEST_COUNT_ONE),
		value_integer(TEST_COUNT_TWO),
	}
	compare_execution(
		t, `{{range $index, $value := .}}{{$index}}={{$value}};{{end}}`,
		value_sequence(values[:]), "0=3;1=1;2=2;",
	)
	entries := [...]template.Field{
		{
			Key: template.Field_Key{Data: value_text([]byte("b"))},
			Value: template.Field_Value{
				Data: value_integer(TEST_COUNT_TWO),
			},
		},
		{
			Key: template.Field_Key{Data: value_text([]byte("a"))},
			Value: template.Field_Value{
				Data: value_integer(TEST_COUNT_ONE),
			},
		},
	}
	compare_execution(
		t, `{{range $key, $value := .}}{{$key}}={{$value}};{{end}}`, value_map(entries[:]),
		"a=1;b=2;",
	)
	compare_execution(
		t, `{{index . 1}}|{{len (slice . 1 3)}}`, value_sequence(values[:]), "1|2",
	)
	compare_execution(t, `{{$value := 1}}{{$value = 2}}{{$value}}`, value_nil(), "2")
	compare_execution(
		t, `{{template "named" .}}{{define "named"}}value{{end}}`, value_nil(), "value",
	)
}

func test_standard_library_slice_capacity(t *testing.T) {
	t.Helper()
	shared_values := [...]template.Value{
		value_integer(TEST_COUNT_THREE),
		value_integer(TEST_COUNT_FOUR),
		value_integer(TEST_COUNT_FIVE),
		value_integer(TEST_COUNT_ZERO),
		value_integer(TEST_COUNT_ZERO),
	}
	compare_execution(
		t, `{{slice . 3 5}}`, value_sequence(shared_values[:TEST_COUNT_THREE]), "[0 0]",
	)
	compare_execution(
		t, `{{slice . 3 5 5}}`, value_sequence(shared_values[:TEST_COUNT_THREE]), "[0 0]",
	)
}

func test_standard_library_collection_execution(t *testing.T) {
	t.Helper()
	test_standard_library_scalar_execution(t)
	values := [...]template.Value{
		value_integer(TEST_COUNT_THREE),
		value_integer(TEST_COUNT_FOUR),
		value_integer(TEST_COUNT_FIVE),
	}
	compare_execution(
		t, `{{range $index, $value := .}}{{$index}}={{$value}};{{end}}`,
		value_sequence(values[:]), "0=3;1=4;2=5;",
	)
	compare_execution(
		t, `{{index . 1}}|{{slice . 1 3}}|{{len .}}`, value_sequence(values[:]),
		"4|[4 5]|3",
	)
	compare_execution(
		t, `{{index . 1}}|{{slice . 1 3}}|{{len .}}`, value_text([]byte("abc")), "98|bc|3",
	)
	mixed := [...]template.Value{value_integer(TEST_COUNT_ONE), value_unsigned(TEST_COUNT_ONE)}
	compare_execution(
		t, `{{eq (index . 0) (index . 1)}}|{{le (index . 0) (index . 1)}}`,
		value_sequence(mixed[:]), "true|true",
	)
	compare_execution(t, `{{if .}}true{{else}}false{{end}}`, value_object(nil), "true")
	entries := [...]template.Field{{
		Key: template.Field_Key{
			Data: value_text([]byte("present")),
		},
		Value: template.Field_Value{Data: trusted(
			template.Value_Unvalidated(template.Value_Of_Integer(TEST_COUNT_ONE)),
		)},
	}}
	compare_execution(t, `{{index . "missing"}}`, value_map(entries[:]), "<no value>")
}

func test_standard_library_scalar_execution(t *testing.T) {
	t.Helper()
	compare_execution(t, `<{{.}}>`, value_integer(-TEST_COUNT_THIRTEEN), "<-13>")
	compare_execution(t, `<{{.}}>`, value_unsigned(TEST_COUNT_FOURTEEN), "<14>")
	const FLOAT_ONE_BITS = uint64(big.FLOAT_64_EXPONENT_BIAS) <<
		big.FLOAT_64_EXPONENT_SHIFT
	compare_execution(t, `<{{.}}>`, value_float(template.Float(FLOAT_ONE_BITS)), "<1>")
	compare_execution(
		t, `<{{.}}>`, value_complex(template.Complex{
			Real:      template.Complex_Real(FLOAT_ONE_BITS),
			Imaginary: template.Complex_Imaginary(FLOAT_ONE_BITS),
		}), "<(1+1i)>",
	)
	compare_execution(
		t, `{{range $value := .}}{{$value}}{{else}}empty{{end}}`,
		value_integer(TEST_COUNT_THREE), "012",
	)
	compare_execution(
		t, `{{range $value := .}}{{$value}}{{else}}empty{{end}}`,
		value_unsigned(TEST_COUNT_THREE), "012",
	)
	compare_execution(
		t, `{{range $value := .}}{{$value}}{{else}}empty{{end}}`,
		value_integer(-TEST_COUNT_ONE), "empty",
	)
}

func compare_execution(
	t *testing.T, source string, shared_value template.Value, expected string,
) {
	t.Helper()
	program := compile_from(t, source)
	shared_output, status := execute_from(t, program, shared_value)
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, expected, shared_output)
}

func compile_from(
	t *testing.T, source_text string,
) (program template.Program) {
	t.Helper()
	syntax := &template.Syntax_Workspace{}
	program, _, status := template.Compile(template.Compile_Input{
		Source: []byte(source_text), Workspace: syntax_from(syntax),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	return program
}

func execute_from(
	t *testing.T, program template.Program, value template.Value,
) (output string, status template.Status) {
	t.Helper()
	return execute_functions_from(t, program, value, nil)
}

func execute_functions_from(
	t *testing.T, program template.Program, value template.Value, functions template.Functions,
) (output string, status template.Status) {
	t.Helper()
	var execution execution_storage
	return execute_shared_from(t, &execution, program, value, functions)
}

// One caller workspace serves many runs, so repeated runs stay inside the time budget.
func execute_shared_from(
	t *testing.T, execution *execution_storage, program template.Program,
	value template.Value, functions template.Functions,
) (output string, status template.Status) {
	t.Helper()
	var destination [template.OUTPUT_SIZE_MAXIMUM]byte
	count, _, status := template.Execute_Into(template.Execute_Input{
		Destination: destination[:], Program: template.Program_Unvalidated(program),
		Value:     template.Value_Unvalidated(value),
		Functions: functions, Workspace: workspace_from(execution),
	})
	return string(destination[:count]), status
}

func syntax_from(workspace *template.Syntax_Workspace) (input template.Syntax_Input) {
	syntax_workspace_prepare(workspace)
	input.State = (*template.Syntax_Workspace_Unvalidated)(workspace)
	input.Number = number_workspace()
	return input
}

func syntax_workspace_prepare(workspace *template.Syntax_Workspace) {
	if workspace.Nodes == nil {
		workspace.Nodes = make(template.Nodes, template.NODE_COUNT_MAXIMUM)
	}
	if workspace.Template_Name_Left == nil {
		workspace.Template_Name_Left = make(
			template.Template_Name_Left, template.TEMPLATE_NAME_SIZE_MAXIMUM,
		)
	}
	if workspace.Template_Name_Right == nil {
		workspace.Template_Name_Right = make(
			template.Template_Name_Right, template.TEMPLATE_NAME_SIZE_MAXIMUM,
		)
	}
}

func syntax_node(
	program template.Program, reference template.Node_Reference,
) (node *template.Node) {
	return &(*program.Syntax.Value)[reference-template.NODE_COUNT_INCREMENT]
}

func syntax_first_pipe(program template.Program) (pipe *template.Node) {
	root := syntax_node(program, template.Node_Reference(program.Document.Root))
	action := syntax_node(program, template.Node_Reference(root.First_Child))
	return syntax_node(program, template.Node_Reference(action.First_Child))
}

type execution_storage struct {
	Workspace       template.Workspace
	Output          []byte
	Literals        []byte
	Literal_Counts  []template.Literal_Count
	Literal_Offsets []template.Literal_Count
	Literal_Decoded []template.Boolean
	Name_Left       []byte
	Name_Right      []byte
	Number          []byte
	Generated       []byte
	Float_Parse     big.Float_Parse_Workspace
	Float_Text      big.Float_Text_Workspace
	Frames          []template.Frame_Storage
	Variables       []template.Variable
	Arguments       []template.Value
	Validation      []template.Value
	Evaluations     []template.Evaluation_Frame
	Format          []template.Value_Format_Frame_Storage
}

func execution_storage_prepare(storage *execution_storage) {
	if storage.Float_Parse.Source == nil {
		test_float_parse_workspace_initialize(&storage.Float_Parse)
	}
	if storage.Float_Text.Output == nil {
		test_float_text_workspace_initialize(&storage.Float_Text)
	}
	if storage.Output != nil {
		return
	}
	storage.Output = make([]byte, template.OUTPUT_SIZE_MAXIMUM)
	storage.Literals = make([]byte, template.QUOTED_DECODED_SIZE_MAXIMUM)
	storage.Literal_Counts = make([]template.Literal_Count, template.NODE_COUNT_MAXIMUM)
	storage.Literal_Offsets = make([]template.Literal_Count, template.NODE_COUNT_MAXIMUM)
	storage.Literal_Decoded = make([]template.Boolean, template.NODE_COUNT_MAXIMUM)
	storage.Name_Left = make([]byte, template.QUOTED_DECODED_SIZE_MAXIMUM)
	storage.Name_Right = make([]byte, template.QUOTED_DECODED_SIZE_MAXIMUM)
	storage.Number = make([]byte, template.NUMBER_SIZE_MAXIMUM)
	storage.Generated = make([]byte, template.GENERATED_SIZE_MAXIMUM)
	storage.Frames = make([]template.Frame_Storage, template.FRAME_COUNT_MAXIMUM)
	storage.Variables = make([]template.Variable, template.VARIABLE_COUNT_MAXIMUM)
	storage.Arguments = make([]template.Value, template.ARGUMENT_COUNT_MAXIMUM)
	storage.Validation = make([]template.Value, template.FRAME_COUNT_MAXIMUM)
	storage.Evaluations = make([]template.Evaluation_Frame, template.FRAME_COUNT_MAXIMUM)
	storage.Format = make([]template.Value_Format_Frame_Storage, template.FRAME_COUNT_MAXIMUM)
}

func workspace_from(storage *execution_storage) (input template.Workspace_Input) {
	execution_storage_prepare(storage)
	storage.Workspace = template.Workspace{
		Output: storage.Output[:], Literals: storage.Literals[:],
		Literal_Counts:  storage.Literal_Counts[:],
		Literal_Offsets: storage.Literal_Offsets[:],
		Literal_Decoded: storage.Literal_Decoded[:],
		Name_Left:       storage.Name_Left[:], Name_Right: storage.Name_Right[:],
		Number:    storage.Number[:],
		Generated: storage.Generated[:],
		Frames:    storage.Frames[:],
		Variables: storage.Variables[:], Arguments: storage.Arguments[:],
		Validation:  storage.Validation[:],
		Evaluations: storage.Evaluations[:],
		Format:      storage.Format[:],
	}
	input.State.Value = template.Workspace_Unvalidated_Pointer{
		Value: (*template.Workspace_Unvalidated)(&storage.Workspace),
	}
	input.Float_Parse = (*template.Float_Parse_Unvalidated)(&storage.Float_Parse)
	input.Float_Text = (*template.Float_Text_Unvalidated)(&storage.Float_Text)
	return input
}

// Test_Source refuses work beyond parser storage boundary.
func Test_Source(t *testing.T) {
	var pair [template.DEFAULT_DELIMITER_SIZE]byte
	_, status := template.Source_Validate(pair[:])
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)

	var maximum [template.SOURCE_SIZE_MAXIMUM]byte
	source, status := template.Source_Validate(maximum[:])
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	testify.Equal(t, len(maximum), len(source.Data.Value))

	var oversized [template.SOURCE_SIZE_UNVALIDATED_MAXIMUM]byte
	source, status = template.Source_Validate(oversized[:])
	testify.Equal_Values(t, template.PARSE_STATUS_INPUT_INVALID, status)
	testify.Equal(t, template.SOURCE_SIZE_MINIMUM, len(source.Data.Value))
}

// Test_Configuration selects standard delimiters and rejects half custom policy.
func Test_Configuration(t *testing.T) {
	configuration := configuration_from(t)
	testify.True(t, bool(template.Configuration_Valid(configuration)))

	_, status := template.New_Configuration(template.Configuration_Input{
		Left: template.Left_Delimiter_Unvalidated("<%"),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_CONFIGURATION_INVALID, status)

	custom := configuration_input_from(t, template.Configuration_Input{
		Left:  template.Left_Delimiter_Unvalidated("<%"),
		Right: template.Right_Delimiter_Unvalidated("%>"),
	})
	_, document, workspace := parse_configuration(t, "<%.%>", custom)
	root := node_at(t, document, &workspace, template.Node_Reference(document.Root))
	action := node_at(t, document, &workspace, template.Node_Reference(root.First_Child))
	testify.Equal_Values(t, template.NODE_ACTION, action.Kind)

	_, status = template.New_Configuration(template.Configuration_Input{
		Mode: template.SKIP_FUNCTION_CHECK,
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	_, status = template.New_Configuration(template.Configuration_Input{
		Mode: template.Mode(bits.WORD_MAXIMUM),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_CONFIGURATION_INVALID, status)
	var left [template.DELIMITER_SIZE_UNVALIDATED_MAXIMUM]byte
	var right [template.DELIMITER_SIZE_UNVALIDATED_MAXIMUM]byte
	_, status = template.New_Configuration(template.Configuration_Input{
		Left:  left[:],
		Right: right[:],
	})
	testify.Equal_Values(t, template.PARSE_STATUS_CONFIGURATION_INVALID, status)
}

// Test_Parse stores standard text, action, pipeline, command, and dot nodes.
func Test_Parse(t *testing.T) {
	source, document, workspace := parse_text(t, "hello {{.}}!")
	root := node_at(t, document, &workspace, template.Node_Reference(document.Root))
	testify.Equal_Values(t, template.NODE_LIST, root.Kind)

	text := node_at(t, document, &workspace, template.Node_Reference(root.First_Child))
	testify.Equal_Values(t, template.NODE_TEXT, text.Kind)
	testify.Equal(t, "hello ", node_value(source, text))

	action := node_at(t, document, &workspace, template.Node_Reference(text.Next_Sibling))
	testify.Equal_Values(t, template.NODE_ACTION, action.Kind)
	pipe := node_at(t, document, &workspace, template.Node_Reference(action.First_Child))
	testify.Equal_Values(t, template.NODE_PIPE, pipe.Kind)
	command := node_at(t, document, &workspace, template.Node_Reference(pipe.First_Child))
	testify.Equal_Values(t, template.NODE_COMMAND, command.Kind)
	dot := node_at(t, document, &workspace, template.Node_Reference(command.First_Child))
	testify.Equal_Values(t, template.NODE_DOT, dot.Kind)

	ending := node_at(t, document, &workspace, template.Node_Reference(action.Next_Sibling))
	testify.Equal_Values(t, template.NODE_TEXT, ending.Kind)
	testify.Equal(t, "!", node_value(source, ending))
	testify.Equal_Values(t, template.NO_NODE, ending.Next_Sibling)
	test_trim(t)
	test_pipeline(t)
	test_chain_terms(t)
	test_parentheses(t)
	test_variables(t)
	test_branches(t)
	test_comments(t)
	test_templates(t)
	test_template_reference_boundaries(t)
	test_lexical_widths(t)
	test_quoted_decode_bounds(t)
}

// Test_Diagnostics rejects an action missing its closing delimiter.
func Test_Diagnostics(t *testing.T) {
	configuration := configuration_from(t)
	source := source_from(t, "{{.")
	var workspace template.Syntax_Workspace
	document, diagnostic, status := template.Parse_Into(
		template.Source_Header_Unvalidated(source),
		configuration,
		parse_workspace_from(&workspace),
	)
	testify.Equal_Values(t, template.PARSE_STATUS_SYNTAX_INVALID, status)
	testify.Equal_Values(t, template.DIAGNOSTIC_SYNTAX_INVALID, diagnostic.Code)
	testify.Equal(t, template.Parse_Diagnostic_Position(len("{{.")), diagnostic.Position)
	testify.Equal_Values(t, template.NO_NODE, document.Root)

	source = source_from(t, "{{range $value := .}}{{$value}}{{end}}{{$value}}")
	document, diagnostic, status = template.Parse_Into(
		template.Source_Header_Unvalidated(source),
		configuration,
		parse_workspace_from(&workspace),
	)
	testify.Equal_Values(t, template.PARSE_STATUS_SYNTAX_INVALID, status)
	testify.Equal_Values(t, template.DIAGNOSTIC_SYNTAX_INVALID, diagnostic.Code)
	testify.Equal_Values(t, template.NO_NODE, document.Root)

	source = source_from(t, "{{if .}}x")
	document, diagnostic, status = template.Parse_Into(
		template.Source_Header_Unvalidated(source),
		configuration,
		parse_workspace_from(&workspace),
	)
	testify.Equal_Values(t, template.PARSE_STATUS_SYNTAX_INVALID, status)
	testify.Equal_Values(t, template.NO_NODE, document.Root)

	source = source_from(t, "{{$missing}}")
	document, diagnostic, status = template.Parse_Into(
		template.Source_Header_Unvalidated(source),
		configuration,
		parse_workspace_from(&workspace),
	)
	testify.Equal_Values(t, template.PARSE_STATUS_SYNTAX_INVALID, status)
	testify.Equal_Values(t, template.NO_NODE, document.Root)

	source = source_from(t, "{{$first, $second := .}}")
	document, diagnostic, status = template.Parse_Into(
		template.Source_Header_Unvalidated(source),
		configuration,
		parse_workspace_from(&workspace),
	)
	testify.Equal_Values(t, template.PARSE_STATUS_SYNTAX_INVALID, status)
	testify.Equal_Values(t, template.NO_NODE, document.Root)

	for _, malformed := range [...]string{
		"{{define @}}",
		`{{define "name" .}}{{end}}`,
		"{{if .}}{{else @}}{{end}}",
	} {
		source = source_from(t, malformed)
		document, diagnostic, status = template.Parse_Into(
			template.Source_Header_Unvalidated(source),
			configuration,
			parse_workspace_from(&workspace),
		)
		testify.Equal_Values(t, template.PARSE_STATUS_SYNTAX_INVALID, status)
		testify.Equal_Values(t, template.DIAGNOSTIC_SYNTAX_INVALID, diagnostic.Code)
		testify.Equal_Values(t, template.NO_NODE, document.Root)
	}
	test_hostile_bytes(t)
	test_hostile_quotes(t)
}

// Test_Bounds reports absent caller state before parsing.
func Test_Parse_Bounds(t *testing.T) {
	t.Parallel()
	configuration := configuration_from(t)
	source := source_from(t, "x")
	document, diagnostic, status := template.Parse_Into(
		template.Source_Header_Unvalidated(source),
		configuration,
		template.Syntax_Workspace_Input{},
	)
	testify.Equal_Values(t, template.PARSE_STATUS_WORKSPACE_INVALID, status)
	testify.Equal_Values(t, template.DIAGNOSTIC_WORKSPACE_INVALID, diagnostic.Code)
	testify.Equal_Values(t, template.NO_NODE, document.Root)

	test_hostile_public_inputs(t, source, configuration)
	test_workspace_state_bounds(t, source, configuration)

	limited := template.Syntax_Workspace{
		Node_Limit: template.Node_Limit_Storage{
			Value: template.Node_Limit_Stored(template.NODE_COUNT_INCREMENT),
		},
	}
	document, diagnostic, status = template.Parse_Into(
		template.Source_Header_Unvalidated(source),
		configuration,
		parse_workspace_from(&limited),
	)
	testify.Equal_Values(t, template.PARSE_STATUS_CAPACITY_EXCEEDED, status)
	testify.Equal_Values(t, template.DIAGNOSTIC_CAPACITY_EXCEEDED, diagnostic.Code)
	testify.Equal_Values(t, template.NO_NODE, document.Root)
	test_capacity_propagation(t)
	test_maximum_diagnostic_position(t)
	test_token_boundaries(t)
	test_maximum_node_bound(t)
	test_maximum_parenthesis_depth(t)
	test_maximum_control_depth(t)
	test_maximum_control_clause(t)
	test_maximum_variable_count(t)
	test_maximum_template_count(t)
	test_maximum_first_template(t)
}

// Test_Allocation proves source validation, configuration, and parsing allocate nothing.
func Test_Parse_Allocation(t *testing.T) {
	unvalidated := template.Source_Unvalidated(`hello {{printf "%v" ((.Field)).Name}}`)
	var source template.Source
	var source_status template.Source_Status
	testify.Zero_Allocation(t, func() {
		source, source_status = template.Source_Validate(unvalidated)
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, source_status)

	var configuration template.Configuration
	var configuration_status template.Configuration_Status
	testify.Zero_Allocation(t, func() {
		configuration, configuration_status = template.New_Configuration(
			template.Configuration_Input{Function: known_printf},
		)
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, configuration_status)
	var configuration_valid template.Configuration_Validity
	testify.Zero_Allocation(t, func() {
		configuration_valid = template.Configuration_Valid(configuration)
	})
	testify.True(t, bool(configuration_valid))

	var workspace template.Syntax_Workspace
	parse_input := parse_workspace_from(&workspace)
	var document template.Document
	var diagnostic template.Parse_Diagnostic
	var status template.Parse_Status
	testify.Zero_Allocation(t, func() {
		document, diagnostic, status = template.Parse_Into(
			template.Source_Header_Unvalidated(source),
			configuration,
			parse_input,
		)
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	testify.Equal_Values(t, template.DIAGNOSTIC_NONE, diagnostic.Code)
	testify.True(t, uint16(document.Root) != uint16(template.NO_NODE))

	quoted_source := source_from(t, `"a\n"`)
	var quoted_output [template.QUOTED_DECODED_SIZE_MAXIMUM]byte
	var quoted_count template.Quoted_Count
	var quoted_status template.Lex_Status
	quoted_span := template.Quoted_Span_Unvalidated{
		Start: template.Quoted_Start_Unvalidated(TEST_COUNT_ZERO),
		End:   template.Quoted_End_Unvalidated(len(`"a\n"`)),
	}
	testify.Zero_Allocation(t, func() {
		quoted_count, quoted_status = template.Quoted_Unquote_Into(
			quoted_output[:], quoted_source, quoted_span,
		)
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, quoted_status)
	testify.Equal(t, "a\n", string(quoted_output[:quoted_count]))
	test_parse_branch_allocation(t, configuration)
}

func test_parse_branch_allocation(
	t *testing.T, configuration template.Configuration,
) {
	t.Helper()
	for _, source_text := range [...]string{
		`{{if .Ready}}yes{{else if .Waiting}}wait{{else}}no{{end}}`,
		`{{range $key, $value := .}}{{$key}}={{$value}}{{else}}empty{{end}}`,
		`{{template "named" .}}{{define "named"}}value{{end}}`,
		`one {{- /* comment */ -}} two`,
		`{{-3}}|{{3.5}}|{{1+2i}}|{{'x'}}|{{"text"}}|{{` + "`raw`" + `}}`,
	} {
		source := source_from(t, source_text)
		var workspace template.Syntax_Workspace
		parse_input := parse_workspace_from(&workspace)
		var document template.Document
		var diagnostic template.Parse_Diagnostic
		var status template.Parse_Status
		testify.Zero_Allocation(t, func() {
			document, diagnostic, status = template.Parse_Into(
				template.Source_Header_Unvalidated(source),
				configuration,
				parse_input,
			)
		})
		testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
		testify.Equal_Values(t, template.DIAGNOSTIC_NONE, diagnostic.Code)
		testify.True(t, uint16(document.Root) != uint16(template.NO_NODE))
	}
}

// Quoted decoding uses exact source-derived output bound.
func test_quoted_decode_bounds(t *testing.T) {
	t.Helper()
	var output [template.QUOTED_DECODED_SIZE_MAXIMUM]byte
	source := source_from(t, `x"a\n"`)
	count, status := template.Quoted_Unquote_Into(
		output[:], source,
		template.Quoted_Span_Unvalidated{
			Start: template.Quoted_Start_Unvalidated(TEST_COUNT_ONE),
			End:   template.Quoted_End_Unvalidated(len(`x"a\n"`)),
		},
	)
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	testify.Equal(t, "a\n", string(output[:count]))

	source = source_from(t, `x""`)
	count, status = template.Quoted_Unquote_Into(
		output[:], source, template.Quoted_Span_Unvalidated{
			Start: template.Quoted_Start_Unvalidated(TEST_COUNT_ONE),
			End:   template.Quoted_End_Unvalidated(TEST_COUNT_THREE),
		},
	)
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	testify.Equal_Values(t, template.Quoted_Count(TEST_COUNT_ZERO), count)

	source = source_from(t, `""`)
	count, status = template.Quoted_Unquote_Into(
		output[:], source, template.Quoted_Span_Unvalidated{
			Start: template.Quoted_Start_Unvalidated(TEST_COUNT_ZERO),
			End:   template.Quoted_End_Unvalidated(TEST_COUNT_TWO),
		},
	)
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	testify.Equal_Values(t, template.Quoted_Count(TEST_COUNT_ZERO), count)

	source = source_from(t, `"x"`)
	count, status = template.Quoted_Unquote_Into(
		output[:], source, template.Quoted_Span_Unvalidated{
			Start: template.Quoted_Start_Unvalidated(TEST_COUNT_ZERO),
			End:   template.Quoted_End_Unvalidated(TEST_COUNT_THREE),
		},
	)
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	testify.Equal_Values(t, template.Quoted_Count(TEST_COUNT_ONE), count)

	test_quoted_decode_octal(t)

	source = source_from(t, `xx""`)
	count, status = template.Quoted_Unquote_Into(
		output[:], source, template.Quoted_Span_Unvalidated{
			Start: template.Quoted_Start_Unvalidated(TEST_COUNT_TWO),
			End:   template.Quoted_End_Unvalidated(TEST_COUNT_FOUR),
		},
	)
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	testify.Equal_Values(t, template.Quoted_Count(TEST_COUNT_ZERO), count)

	test_quoted_decode_maximum(t)

	source = source_from(t, "x/invalid/")
	_, status = template.Quoted_Unquote_Into(
		output[:], source, template.Quoted_Span_Unvalidated{
			Start: template.Quoted_Start_Unvalidated(TEST_COUNT_ONE),
			End:   template.Quoted_End_Unvalidated(TEST_COUNT_TEN),
		},
	)
	testify.Equal_Values(t, template.PARSE_STATUS_SYNTAX_INVALID, status)

	test_quoted_decode_rejects_malformed_span(t)
}

func test_quoted_decode_octal(t *testing.T) {
	t.Helper()
	var output [template.QUOTED_DECODED_SIZE_MAXIMUM]byte
	source := source_from(t, `"\141"`)
	count, status := template.Quoted_Unquote_Into(
		output[:], source, template.Quoted_Span_Unvalidated{
			Start: template.Quoted_Start_Unvalidated(TEST_COUNT_ZERO),
			End:   template.Quoted_End_Unvalidated(len(`"\141"`)),
		},
	)
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	testify.Equal(t, "a", string(output[:count]))

	source = source_from(t, `"\U0010FFFF"`)
	count, status = template.Quoted_Unquote_Into(
		output[:], source, template.Quoted_Span_Unvalidated{
			Start: template.Quoted_Start_Unvalidated(TEST_COUNT_ZERO),
			End:   template.Quoted_End_Unvalidated(len(`"\U0010FFFF"`)),
		},
	)
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	var maximum [utf8.CHARACTER_SIZE_MAXIMUM]byte
	size := utf8.Encode_Character(maximum[:], utf8.RUNE_MAX)
	testify.Equal(t, string(maximum[:size]), string(output[:count]))

	source = source_from(t, "`x`")
	count, status = template.Quoted_Unquote_Into(
		output[:], source, template.Quoted_Span_Unvalidated{
			Start: template.Quoted_Start_Unvalidated(TEST_COUNT_ZERO),
			End:   template.Quoted_End_Unvalidated(len("`x`")),
		},
	)
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	testify.Equal(t, "x", string(output[:count]))
}

// Exclusive end must name a distinct closing quote owned by span.
func test_quoted_decode_rejects_malformed_span(t *testing.T) {
	t.Helper()
	var output [template.QUOTED_DECODED_SIZE_MAXIMUM]byte
	for _, malformed := range [...]string{"", `"`, `"x]`} {
		source := source_from(t, malformed)
		_, status := template.Quoted_Unquote_Into(
			output[:], source, template.Quoted_Span_Unvalidated{
				Start: template.Quoted_Start_Unvalidated(TEST_COUNT_ZERO),
				End:   template.Quoted_End_Unvalidated(len(malformed)),
			},
		)
		testify.Equal_Values(t, template.PARSE_STATUS_SYNTAX_INVALID, status)
	}
	source := source_from(t, `""`)
	for _, span := range [...]template.Quoted_Span_Unvalidated{
		{
			Start: template.Quoted_Start_Unvalidated(TEST_COUNT_ZERO),
			End:   template.Quoted_End_Unvalidated(TEST_COUNT_ZERO),
		},
		{
			Start: template.Quoted_Start_Unvalidated(
				template.SOURCE_SIZE_UNVALIDATED_MAXIMUM,
			),
			End: template.Quoted_End_Unvalidated(
				template.SOURCE_SIZE_UNVALIDATED_MAXIMUM,
			),
		},
	} {
		_, status := template.Quoted_Unquote_Into(output[:], source, span)
		testify.Equal_Values(t, template.PARSE_STATUS_SYNTAX_INVALID, status)
	}
}

// Maximum malformed content expands each byte into one replacement character.
func test_quoted_decode_maximum(t *testing.T) {
	t.Helper()
	var output [template.QUOTED_DECODED_SIZE_MAXIMUM]byte
	var maximum [template.SOURCE_SIZE_MAXIMUM]byte
	maximum[TEST_COUNT_ZERO] = '"'
	position_index := TEST_COUNT_ONE
	for position_index < len(maximum)-TEST_COUNT_ONE {
		maximum[position_index] = bits.WORD_8_MAXIMUM
		position_index++
	}
	maximum[len(maximum)-TEST_COUNT_ONE] = '"'
	validated, source_status := template.Source_Validate(maximum[:])
	testify.Equal_Values(t, template.PARSE_STATUS_OK, source_status)
	count, status := template.Quoted_Unquote_Into(
		output[:], validated,
		template.Quoted_Span_Unvalidated{
			Start: template.Quoted_Start_Unvalidated(TEST_COUNT_ZERO),
			End:   template.Quoted_End_Unvalidated(len(maximum)),
		},
	)
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	testify.Equal_Values(t, template.Quoted_Count(len(output)), count)

	maximum[len(maximum)-TEST_COUNT_TWO] = '"'
	count, status = template.Quoted_Unquote_Into(
		output[:], validated,
		template.Quoted_Span_Unvalidated{
			Start: template.Quoted_Start_Unvalidated(len(maximum) - TEST_COUNT_TWO),
			End:   template.Quoted_End_Unvalidated(len(maximum)),
		},
	)
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	testify.Equal_Values(t, template.Quoted_Count(TEST_COUNT_ZERO), count)
}

// Every grammar allocation must propagate caller capacity without changing failure class.
func test_capacity_propagation(t *testing.T) {
	t.Helper()
	cases := [...]struct {
		Text  string
		Input template.Configuration_Input
	}{
		{Text: "{{.}}"},
		{
			Text:  "{{/* retained */}}",
			Input: template.Configuration_Input{Mode: template.PARSE_COMMENTS},
		},
		{Text: `{{template "name" .}}`},
		{Text: `{{template "name" ((.))}}`},
		{Text: `{{block "name" .}}body{{end}}`},
		{Text: `{{define "name"}}body{{end}}`},
		{Text: "{{if .}}left{{else}}right{{end}}"},
		{Text: "{{with .}}body{{end}}"},
		{Text: "{{range .}}{{break}}{{end}}"},
		{Text: "{{range .}}{{continue}}{{end}}"},
		{Text: "{{$value := .}}{{$value}}"},
		{
			Text:  "{{printf ((.Field).Value) | printf}}",
			Input: template.Configuration_Input{Function: known_printf},
		},
	}
	for _, current := range cases {
		source := source_from(t, current.Text)
		configuration := configuration_input_from(t, current.Input)
		var complete template.Syntax_Workspace
		document, _, status := template.Parse_Into(
			template.Source_Header_Unvalidated(source),
			configuration,
			parse_workspace_from(&complete),
		)
		testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
		for limit := template.Node_Limit(template.NODE_COUNT_INCREMENT); ; limit++ {
			if limit >= template.Node_Limit(document.Node_Count) {
				break
			}
			limited := template.Syntax_Workspace{
				Node_Limit: template.Node_Limit_Storage{
					Value: template.Node_Limit_Stored(limit),
				},
			}
			_, _, status = template.Parse_Into(
				template.Source_Header_Unvalidated(source),
				configuration,
				parse_workspace_from(&limited),
			)
			testify.Equal_Values(t, template.PARSE_STATUS_CAPACITY_EXCEEDED, status)
		}
	}
}

// Each arena field of a caller-supplied document meets validation at its own extremes.
func test_hostile_documents(t *testing.T) {
	t.Helper()
	// One source at its bound reaches execution through the public entry point.
	filled := make([]byte, TEST_COUNT_ZERO, template.SOURCE_SIZE_MAXIMUM)
	for len(filled) < template.SOURCE_SIZE_MAXIMUM {
		filled = append(filled, 'x')
	}
	execute_from(t, compile_from(t, string(filled)), value_nil())
	ceiling := uint16(bits.WORD_16_MAXIMUM)
	counts := [...]uint16{TEST_COUNT_TWO, ceiling}
	var execution execution_storage
	var output [template.OUTPUT_SIZE_MAXIMUM]byte
	for _, count := range counts {
		documents := [...]template.Document{
			{Root: template.Document_Root(count)},
			{Node_Count: template.Node_Count(count)},
			{First_Template: template.Document_First_Template(count)},
			{Template_Count: template.Template_Count(count)},
		}
		for index := range documents {
			hostile := compile_from(t, `{{.}}`)
			hostile.Document = documents[index]
			execution = execution_storage{}
			template.Execute_Into(template.Execute_Input{
				Destination: output[:],
				Program:     template.Program_Unvalidated(hostile),
				Value:       template.Value_Unvalidated(template.Value_Nil()),
				Workspace:   workspace_from(&execution),
			})
		}
	}
	test_hostile_nodes(t)
}

// Public aggregates must reject malicious values before parser state changes.
func test_hostile_public_inputs(
	t *testing.T, source template.Source, configuration template.Configuration,
) {
	t.Helper()
	var oversized [template.SOURCE_SIZE_UNVALIDATED_MAXIMUM]byte
	malicious_source := template.Source{
		Data: template.Source_Storage{Value: template.Source_Storage_Value(oversized[:])},
	}
	var workspace template.Syntax_Workspace
	document, diagnostic, status := template.Parse_Into(
		template.Source_Header_Unvalidated(malicious_source),
		configuration,
		parse_workspace_from(&workspace),
	)
	testify.Equal_Values(t, template.PARSE_STATUS_INPUT_INVALID, status)
	testify.Equal_Values(t, template.DIAGNOSTIC_INPUT_INVALID, diagnostic.Code)
	testify.Equal_Values(t, template.NO_NODE, document.Root)

	malicious_configuration := template.Configuration{
		Left: template.Opening_Delimiter_Storage{
			Value: template.Opening_Delimiter_Storage_Value("<"),
		},
	}
	document, diagnostic, status = template.Parse_Into(
		template.Source_Header_Unvalidated(source),
		malicious_configuration,
		parse_workspace_from(&workspace),
	)
	testify.Equal_Values(t, template.PARSE_STATUS_CONFIGURATION_INVALID, status)
	testify.Equal_Values(t, template.DIAGNOSTIC_CONFIGURATION_INVALID, diagnostic.Code)
	testify.Equal_Values(t, template.NO_NODE, document.Root)

	workspace.Node_Limit = template.Node_Limit_Storage{Value: template.Node_Limit_Stored(
		template.NODE_COUNT_MAXIMUM + template.NODE_COUNT_INCREMENT)}

	document, diagnostic, status = template.Parse_Into(
		template.Source_Header_Unvalidated(source),
		configuration,
		parse_workspace_from(&workspace),
	)
	testify.Equal_Values(t, template.PARSE_STATUS_WORKSPACE_INVALID, status)
	testify.Equal_Values(t, template.DIAGNOSTIC_WORKSPACE_INVALID, diagnostic.Code)
	testify.Equal_Values(t, template.NO_NODE, document.Root)
}

// Stale caller metadata must stay bounded and be reset before parsing.
func test_workspace_state_bounds(
	t *testing.T, source template.Source, configuration template.Configuration,
) {
	t.Helper()
	workspace_cases := [...]template.Syntax_Workspace{
		{
			Parse_Diagnostic: template.Syntax_Diagnostic_Storage{
				Value: template.Parse_Diagnostic_Stored(template.Parse_Diagnostic{
					Code:     template.DIAGNOSTIC_INPUT_INVALID,
					Position: template.Parse_Diagnostic_Position(len("<")),
				}),
			},
			Node_Limit: template.Node_Limit_Storage{
				Value: template.Node_Limit_Stored(
					template.NODE_COUNT_INCREMENT +
						template.NODE_COUNT_INCREMENT,
				),
			},
		},
		{
			Parse_Diagnostic: template.Syntax_Diagnostic_Storage{
				Value: template.Parse_Diagnostic_Stored(template.Parse_Diagnostic{
					Code: template.DIAGNOSTIC_CONFIGURATION_INVALID,
					Position: template.Parse_Diagnostic_Position(
						template.SOURCE_SIZE_MAXIMUM,
					),
				}),
			},
			Node_Limit: template.Node_Limit_Storage{
				Value: template.Node_Limit_Stored(template.NODE_COUNT_MAXIMUM),
			},
		},
		{
			Parse_Diagnostic: template.Syntax_Diagnostic_Storage{
				Value: template.Parse_Diagnostic_Stored(template.Parse_Diagnostic{
					Code: template.DIAGNOSTIC_CAPACITY_EXCEEDED,
				}),
			},
		},
	}
	for index := range workspace_cases {
		document, diagnostic, status := template.Parse_Into(
			template.Source_Header_Unvalidated(source),
			configuration,
			parse_workspace_from(&workspace_cases[index]),
		)
		testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
		testify.Equal_Values(t, template.DIAGNOSTIC_NONE, diagnostic.Code)
		testify.True(t, document.Root != template.Document_Root(template.NO_NODE))
	}
}

// First template reference must cover its exact post-root boundary positions.
func test_template_reference_boundaries(t *testing.T) {
	t.Helper()
	cases := [...]struct {
		Text      string
		Reference template.Document_First_Template
	}{
		{
			Text: `{{define "name"}}{{end}}`,
			Reference: template.Document_First_Template(
				template.DOCUMENT_FIRST_TEMPLATE_MINIMUM +
					template.NODE_COUNT_INCREMENT,
			),
		},
		{
			Text: `x{{define "name"}}{{end}}`,
			Reference: template.Document_First_Template(
				template.DOCUMENT_FIRST_TEMPLATE_MINIMUM +
					template.NODE_COUNT_INCREMENT +
					template.NODE_COUNT_INCREMENT,
			),
		},
		{
			Text: `x{{template "name"}}{{define "name"}}{{end}}`,
			Reference: template.Document_First_Template(
				template.DOCUMENT_FIRST_TEMPLATE_MINIMUM +
					template.NODE_COUNT_INCREMENT +
					template.NODE_COUNT_INCREMENT +
					template.NODE_COUNT_INCREMENT,
			),
		},
	}
	for _, current := range cases {
		_, document, _ := parse_text(t, current.Text)
		testify.Equal_Values(t, current.Reference, document.First_Template)
	}
}

// Lexical widths must retain each UTF-8 width and optional fractional sequence.
func test_lexical_widths(t *testing.T) {
	t.Helper()
	configuration := configuration_input_from(t, template.Configuration_Input{
		Mode: template.SKIP_FUNCTION_CHECK,
	})
	parse_configuration(t, "{{é}}", configuration)
	parse_configuration(t, "{{𐀀}}", configuration)
	parse_text(t, "{{1.}}")
}

// Terminal syntax failure must report the final representable source boundary.
func test_maximum_diagnostic_position(t *testing.T) {
	t.Helper()
	ending := [...]byte{'{', '{', 'i', 'f', ' ', '.', '}', '}'}
	var unvalidated [template.SOURCE_SIZE_MAXIMUM]byte
	limit := len(unvalidated) - len(ending)
	for position := range limit {
		unvalidated[position] = 'x'
	}
	copy(unvalidated[limit:], ending[:])
	source, source_status := template.Source_Validate(unvalidated[:])
	testify.Equal_Values(t, template.PARSE_STATUS_OK, source_status)
	var workspace template.Syntax_Workspace
	_, diagnostic, status := template.Parse_Into(
		template.Source_Header_Unvalidated(source),
		configuration_from(t),
		parse_workspace_from(&workspace),
	)
	testify.Equal_Values(t, template.PARSE_STATUS_SYNTAX_INVALID, status)
	testify.Equal_Values(
		t, template.Parse_Diagnostic_Position(template.SOURCE_SIZE_MAXIMUM),
		diagnostic.Position,
	)
}

// Token spans must reach the first and final action-content boundaries.
func test_token_boundaries(t *testing.T) {
	t.Helper()
	configuration := configuration_input_from(t, template.Configuration_Input{
		Left:  template.Left_Delimiter_Unvalidated("<"),
		Right: template.Right_Delimiter_Unvalidated(">"),
	})
	source := source_from(t, "<>")
	var workspace template.Syntax_Workspace
	_, _, status := template.Parse_Into(
		template.Source_Header_Unvalidated(source),
		configuration,
		parse_workspace_from(&workspace),
	)
	testify.Equal_Values(t, template.PARSE_STATUS_SYNTAX_INVALID, status)

	action := [...]byte{'<', '.', '>'}
	var unvalidated [template.SOURCE_SIZE_MAXIMUM]byte
	start := len(unvalidated) - len(action)
	for position := range start {
		unvalidated[position] = 'x'
	}
	copy(unvalidated[start:], action[:])
	source, source_status := template.Source_Validate(unvalidated[:])
	testify.Equal_Values(t, template.PARSE_STATUS_OK, source_status)
	workspace = template.Syntax_Workspace{}
	_, _, status = template.Parse_Into(
		template.Source_Header_Unvalidated(source),
		configuration,
		parse_workspace_from(&workspace),
	)
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
}

// Hostile byte extrema must reach lexical rejection without escaping source bounds.
func test_hostile_bytes(t *testing.T) {
	t.Helper()
	configuration := configuration_input_from(t, template.Configuration_Input{
		Left:  template.Left_Delimiter_Unvalidated("<%"),
		Right: template.Right_Delimiter_Unvalidated("%XYZ"),
	})
	for value := uint16(bits.WORD_8_MINIMUM); value <= uint16(bits.WORD_8_MAXIMUM); value++ {
		character := byte(value)
		direct := [...]byte{'<', '%', character, '%', 'X', 'Y', 'Z'}
		source, source_status := template.Source_Validate(direct[:])
		testify.Equal_Values(t, template.PARSE_STATUS_OK, source_status)
		var workspace template.Syntax_Workspace
		_, _, status := template.Parse_Into(
			template.Source_Header_Unvalidated(source),
			configuration,
			parse_workspace_from(&workspace),
		)
		testify.True(t,
			status == template.PARSE_STATUS_OK ||
				status == template.PARSE_STATUS_SYNTAX_INVALID,
		)

		number := [...]byte{'<', '%', '0', character, '%', 'X', 'Y', 'Z'}
		source, source_status = template.Source_Validate(number[:])
		testify.Equal_Values(t, template.PARSE_STATUS_OK, source_status)
		workspace = template.Syntax_Workspace{}
		_, _, status = template.Parse_Into(
			template.Source_Header_Unvalidated(source),
			configuration,
			parse_workspace_from(&workspace),
		)
		testify.True(t,
			status == template.PARSE_STATUS_OK ||
				status == template.PARSE_STATUS_SYNTAX_INVALID,
		)

		if value <= uint16(bits.WORD_8_MAXIMUM>>utf8.CHARACTER_SIZE_MINIMUM) {
			identifier := [...]byte{'<', '%', 'A', character, '%', 'X', 'Y', 'Z'}
			source, source_status = template.Source_Validate(identifier[:])
			testify.Equal_Values(t, template.PARSE_STATUS_OK, source_status)
			workspace = template.Syntax_Workspace{}
			_, _, status = template.Parse_Into(
				template.Source_Header_Unvalidated(source),
				configuration,
				parse_workspace_from(&workspace),
			)
			accepted := status == template.PARSE_STATUS_OK
			if status == template.PARSE_STATUS_SYNTAX_INVALID {
				accepted = true
			}
			testify.True(t, accepted)
		}
	}
}

// Quote bytes in hostile positions must end parse without crash.
func test_hostile_quotes(t *testing.T) {
	t.Helper()
	configuration := configuration_input_from(t, template.Configuration_Input{
		Left:  template.Left_Delimiter_Unvalidated("<%"),
		Right: template.Right_Delimiter_Unvalidated("%XYZ"),
	})
	for _, quote := range [...]byte{
		byte(template.QUOTE_STRING), byte(template.QUOTE_CHARACTER),
		byte(template.QUOTE_RAW_STRING),
	} {
		quoted := [...]byte{'<', '%', quote, quote, '%', 'X', 'Y', 'Z'}
		source, source_status := template.Source_Validate(quoted[:])
		testify.Equal_Values(t, template.PARSE_STATUS_OK, source_status)
		var workspace template.Syntax_Workspace
		template.Parse_Into(
			template.Source_Header_Unvalidated(source),
			configuration,
			parse_workspace_from(&workspace),
		)

		identifier := [...]byte{'<', '%', 'A', quote, quote, '%', 'X', 'Y', 'Z'}
		source, source_status = template.Source_Validate(identifier[:])
		testify.Equal_Values(t, template.PARSE_STATUS_OK, source_status)
		workspace = template.Syntax_Workspace{}
		template.Parse_Into(
			template.Source_Header_Unvalidated(source),
			configuration,
			parse_workspace_from(&workspace),
		)
	}
}

// One-byte delimiter density must fit fixed caller arena.
func test_maximum_node_bound(t *testing.T) {
	t.Helper()
	pattern := [...]byte{'<', '.', '>'}
	var unvalidated [template.SOURCE_SIZE_MAXIMUM]byte
	position := template.SOURCE_SIZE_MINIMUM
	for position+len(pattern) <= len(unvalidated) {
		copy(unvalidated[position:], pattern[:])
		position += len(pattern)
	}
	for position < len(unvalidated) {
		unvalidated[position] = 'x'
		position++
	}
	source, source_status := template.Source_Validate(unvalidated[:])
	testify.Equal_Values(t, template.PARSE_STATUS_OK, source_status)
	configuration := configuration_input_from(t, template.Configuration_Input{
		Left:  template.Left_Delimiter_Unvalidated("<"),
		Right: template.Right_Delimiter_Unvalidated(">"),
	})
	var workspace template.Syntax_Workspace
	document, _, status := template.Parse_Into(
		template.Source_Header_Unvalidated(source),
		configuration,
		parse_workspace_from(&workspace),
	)
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	testify.Equal_Values(t, template.NODE_COUNT_MAXIMUM, document.Node_Count)
	workspace = template.Syntax_Workspace{}
	program, _, status := template.Compile(template.Compile_Input{
		Source: unvalidated[:], Left: []byte("<"), Right: []byte(">"),
		Workspace: syntax_from(&workspace),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	testify.Equal_Values(t, template.NODE_COUNT_MAXIMUM, program.Document.Node_Count)
	test_execute_program_input(t, program, template.STATUS_OUTPUT_TOO_LARGE)
	// Empty text prints nothing, so evaluation reaches the final syntax node.
	var execution execution_storage
	var destination [template.OUTPUT_SIZE_MAXIMUM]byte
	_, _, execution_status := template.Execute_Into(template.Execute_Input{
		Destination: destination[:], Program: template.Program_Unvalidated(program),
		Value:     template.Value_Unvalidated(value_text(nil)),
		Workspace: workspace_from(&execution),
	})
	testify.Equal_Values(t, template.STATUS_OK, execution_status)
}

// Scratch must follow reachable nesting, not source bytes.
func test_maximum_parenthesis_depth(t *testing.T) {
	t.Helper()
	depth := (template.SOURCE_SIZE_MAXIMUM - len("<.>")) / len("()")
	testify.Equal(t, depth, template.PARENTHESIS_DEPTH_MAXIMUM)
	var unvalidated [template.SOURCE_SIZE_MAXIMUM]byte
	position := template.SOURCE_SIZE_MINIMUM
	unvalidated[position] = '<'
	position++
	for range depth {
		unvalidated[position] = '('
		position++
	}
	unvalidated[position] = '.'
	position++
	for range depth {
		unvalidated[position] = ')'
		position++
	}
	unvalidated[position] = '>'
	position++
	for position < len(unvalidated) {
		unvalidated[position] = 'x'
		position++
	}
	assert_maximum_parse(t, unvalidated[:])
}

// Every opened branch must retain room for its end action.
func test_maximum_control_depth(t *testing.T) {
	t.Helper()
	open := [...]byte{'<', 'i', 'f', '.', '>'}
	close_action := [...]byte{'<', 'e', 'n', 'd', '>'}
	depth := template.SOURCE_SIZE_MAXIMUM / (len(open) + len(close_action))
	testify.Equal(t, depth, template.CONTROL_DEPTH_MAXIMUM)
	var unvalidated [template.SOURCE_SIZE_MAXIMUM]byte
	position := template.SOURCE_SIZE_MINIMUM
	for range depth {
		copy(unvalidated[position:], open[:])
		position += len(open)
	}
	for range depth {
		copy(unvalidated[position:], close_action[:])
		position += len(close_action)
	}
	for position < len(unvalidated) {
		unvalidated[position] = 'x'
		position++
	}
	assert_maximum_parse(t, unvalidated[:])
}

// Each retained declaration must pay complete action bytes.
func test_maximum_variable_count(t *testing.T) {
	t.Helper()
	declaration := [...]byte{'<', '$', ':', '=', '.', '>'}
	count := template.SOURCE_SIZE_MAXIMUM / len(declaration)
	testify.Equal(t, count, template.SYNTAX_VARIABLE_COUNT_MAXIMUM)
	var unvalidated [template.SOURCE_SIZE_MAXIMUM]byte
	position := template.SOURCE_SIZE_MINIMUM
	for range count {
		copy(unvalidated[position:], declaration[:])
		position += len(declaration)
	}
	for position < len(unvalidated) {
		unvalidated[position] = 'x'
		position++
	}
	assert_maximum_parse(t, unvalidated[:])
}

// Distinct shortest names must fill the complete definition-count formula.
func test_maximum_template_count(t *testing.T) {
	t.Helper()
	open := [...]byte{'<', 'd', 'e', 'f', 'i', 'n', 'e', ' ', '`'}
	close := [...]byte{'`', '>', '<', 'e', 'n', 'd', '>'}
	var unvalidated [template.SOURCE_SIZE_MAXIMUM]byte
	position := template.SOURCE_SIZE_MINIMUM
	candidate := uint16(bits.WORD_8_MINIMUM)
	count := template.TEMPLATE_COUNT_MINIMUM
	for position+len(open)+TEST_COUNT_ONE+len(close) <= len(unvalidated) {
		name := byte(candidate)
		candidate++
		if name == '`' {
			continue
		}
		if name == '\r' {
			continue
		}
		copy(unvalidated[position:], open[:])
		position += len(open)
		unvalidated[position] = name
		position++
		copy(unvalidated[position:], close[:])
		position += len(close)
		count++
	}
	for position < len(unvalidated) {
		unvalidated[position] = 'x'
		position++
	}
	source, source_status := template.Source_Validate(unvalidated[:])
	testify.Equal_Values(t, template.PARSE_STATUS_OK, source_status)
	configuration := configuration_input_from(t, template.Configuration_Input{
		Left:  template.Left_Delimiter_Unvalidated("<"),
		Right: template.Right_Delimiter_Unvalidated(">"),
	})
	var workspace template.Syntax_Workspace
	document, _, status := template.Parse_Into(
		template.Source_Header_Unvalidated(source),
		configuration,
		parse_workspace_from(&workspace),
	)
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	testify.Equal_Values(t, template.Template_Count(count), document.Template_Count)
	workspace = template.Syntax_Workspace{}
	program, _, status := template.Compile(template.Compile_Input{
		Source: unvalidated[:], Left: []byte("<"), Right: []byte(">"),
		Workspace: syntax_from(&workspace),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	testify.Equal_Values(
		t, template.Template_Count(count), program.Document.Template_Count,
	)
	test_execute_program_input(t, program, template.STATUS_OK)
}

// Densest action prefix must place first definition at its formula maximum.
func test_maximum_first_template(t *testing.T) {
	t.Helper()
	action := [...]byte{'<', '.', '>'}
	definition := [...]byte{
		'<', 'b', 'l', 'o', 'c', 'k', ' ', '`', 'x', '`', ' ', '.', '>',
		'<', 'e', 'n', 'd', '>',
	}
	var unvalidated [template.SOURCE_SIZE_MAXIMUM]byte
	position := template.SOURCE_SIZE_MINIMUM
	for range template.TEMPLATE_PREFIX_ACTION_COUNT_MAXIMUM {
		copy(unvalidated[position:], action[:])
		position += len(action)
	}
	definition_start := len(unvalidated) - len(definition)
	for position < definition_start {
		unvalidated[position] = 'x'
		position++
	}
	copy(unvalidated[position:], definition[:])
	source, source_status := template.Source_Validate(unvalidated[:])
	testify.Equal_Values(t, template.PARSE_STATUS_OK, source_status)
	configuration := configuration_input_from(t, template.Configuration_Input{
		Left:  template.Left_Delimiter_Unvalidated("<"),
		Right: template.Right_Delimiter_Unvalidated(">"),
	})
	var workspace template.Syntax_Workspace
	document, _, status := template.Parse_Into(
		template.Source_Header_Unvalidated(source),
		configuration,
		parse_workspace_from(&workspace),
	)
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	testify.Equal_Values(
		t, template.Document_First_Template(template.DOCUMENT_FIRST_TEMPLATE_MAXIMUM),
		document.First_Template,
	)
	workspace = template.Syntax_Workspace{}
	program, _, status := template.Compile(template.Compile_Input{
		Source: unvalidated[:], Left: []byte("<"), Right: []byte(">"),
		Workspace: syntax_from(&workspace),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	testify.Equal_Values(
		t, template.Document_First_Template(template.DOCUMENT_FIRST_TEMPLATE_MAXIMUM),
		program.Document.First_Template,
	)
	test_execute_program_input(t, program, template.STATUS_OUTPUT_TOO_LARGE)
}

// One clause sits at the deepest control nesting, so its stage sees the depth bound.
func test_maximum_control_clause(t *testing.T) {
	t.Helper()
	open := `<if.>`
	shut := `<end>`
	clause := `<else>`
	depth := template.CONTROL_DEPTH_MAXIMUM
	source := make([]byte, TEST_COUNT_ZERO, template.SOURCE_SIZE_MAXIMUM)
	for range depth {
		source = append(source, open...)
	}
	source = append(source, clause...)
	for range depth {
		source = append(source, shut...)
	}
	assert_maximum_parse(t, source)
	execute_from(t, compile_from(t, string(source)), value_nil())
	// One range holds the deepest branch clause its own opener leaves room for.
	loop := `<range .>`
	branch := `<break>`
	nested := (template.SOURCE_SIZE_MAXIMUM - len(loop) - len(shut) - len(branch)) /
		(len(open) + len(shut))
	source = append(source[:TEST_COUNT_ZERO], loop...)
	for range nested {
		source = append(source, open...)
	}
	source = append(source, branch...)
	for range nested {
		source = append(source, shut...)
	}
	source = append(source, shut...)
	assert_maximum_parse(t, source)
	execute_from(t, compile_from(t, string(source)), value_nil())
}

func assert_maximum_parse(t *testing.T, unvalidated template.Source_Unvalidated) {
	t.Helper()
	source, source_status := template.Source_Validate(unvalidated)
	testify.Equal_Values(t, template.PARSE_STATUS_OK, source_status)
	configuration := configuration_input_from(t, template.Configuration_Input{
		Left:  template.Left_Delimiter_Unvalidated("<"),
		Right: template.Right_Delimiter_Unvalidated(">"),
	})
	var workspace template.Syntax_Workspace
	_, _, status := template.Parse_Into(
		template.Source_Header_Unvalidated(source),
		configuration,
		parse_workspace_from(&workspace),
	)
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
}

func test_chain_terms(t *testing.T) {
	t.Helper()
	source, document, workspace := parse_text(t, "{{.A.B}}")
	root := node_at(t, document, &workspace, template.Node_Reference(document.Root))
	action := node_at(t, document, &workspace, template.Node_Reference(root.First_Child))
	pipe := node_at(t, document, &workspace, template.Node_Reference(action.First_Child))
	command := node_at(t, document, &workspace, template.Node_Reference(pipe.First_Child))
	field := node_at(t, document, &workspace, template.Node_Reference(command.First_Child))
	testify.Equal_Values(t, template.NODE_FIELD, field.Kind)
	testify.Equal(t, ".A.B", node_value(source, field))
	testify.Equal_Values(t, template.NO_NODE, field.Next_Sibling)

	source, document, workspace = parse_text(t, "{{$x := .}}{{$x.A.B}}")
	root = node_at(t, document, &workspace, template.Node_Reference(document.Root))
	declaration := node_at(t, document, &workspace, template.Node_Reference(root.First_Child))
	use := node_at(t, document, &workspace, template.Node_Reference(declaration.Next_Sibling))
	pipe = node_at(t, document, &workspace, template.Node_Reference(use.First_Child))
	command = node_at(t, document, &workspace, template.Node_Reference(pipe.First_Child))
	variable := node_at(t, document, &workspace, template.Node_Reference(command.First_Child))
	testify.Equal_Values(t, template.NODE_VARIABLE, variable.Kind)
	testify.Equal(t, "$x.A.B", node_value(source, variable))
	testify.Equal_Values(t, template.NO_NODE, variable.Next_Sibling)

	source, document, workspace = parse_text(t, "{{$变量 := .}}{{$变量.字段}}{{.世界.值}}")
	root = node_at(t, document, &workspace, template.Node_Reference(document.Root))
	declaration = node_at(t, document, &workspace, template.Node_Reference(root.First_Child))
	use = node_at(t, document, &workspace, template.Node_Reference(declaration.Next_Sibling))
	pipe = node_at(t, document, &workspace, template.Node_Reference(use.First_Child))
	command = node_at(t, document, &workspace, template.Node_Reference(pipe.First_Child))
	variable = node_at(t, document, &workspace, template.Node_Reference(command.First_Child))
	testify.Equal(t, "$变量.字段", node_value(source, variable))
	field_action := node_at(t, document, &workspace, template.Node_Reference(use.Next_Sibling))
	pipe = node_at(t, document, &workspace, template.Node_Reference(field_action.First_Child))
	command = node_at(t, document, &workspace, template.Node_Reference(pipe.First_Child))
	field = node_at(t, document, &workspace, template.Node_Reference(command.First_Child))
	testify.Equal(t, ".世界.值", node_value(source, field))
}

func test_trim(t *testing.T) {
	t.Helper()
	source, document, workspace := parse_text(t, "a \n{{- . -}}\n b")
	root := node_at(t, document, &workspace, template.Node_Reference(document.Root))
	left := node_at(t, document, &workspace, template.Node_Reference(root.First_Child))
	action := node_at(t, document, &workspace, template.Node_Reference(left.Next_Sibling))
	right := node_at(t, document, &workspace, template.Node_Reference(action.Next_Sibling))
	testify.Equal(t, "a", node_value(source, left))
	testify.Equal(t, "b", node_value(source, right))
}

func test_pipeline(t *testing.T) {
	t.Helper()
	configuration := configuration_input_from(t, template.Configuration_Input{
		Function: known_printf,
	})
	source, document, workspace := parse_configuration(
		t,
		`{{printf true false nil 12 -3.5 1+2i 'x' "s" `+"`r`"+` $ .Field | printf "%v"}}`,
		configuration,
	)
	root := node_at(t, document, &workspace, template.Node_Reference(document.Root))
	action := node_at(t, document, &workspace, template.Node_Reference(root.First_Child))
	pipe := node_at(t, document, &workspace, template.Node_Reference(action.First_Child))
	first := node_at(t, document, &workspace, template.Node_Reference(pipe.First_Child))
	second := node_at(t, document, &workspace, template.Node_Reference(first.Next_Sibling))
	testify.Equal_Values(t, template.NODE_COMMAND, first.Kind)
	testify.Equal_Values(t, template.NODE_COMMAND, second.Kind)

	kinds := [...]template.Node_Kind{
		template.NODE_IDENTIFIER,
		template.NODE_BOOLEAN,
		template.NODE_BOOLEAN,
		template.NODE_NIL,
		template.NODE_NUMBER,
		template.NODE_NUMBER,
		template.NODE_NUMBER,
		template.NODE_NUMBER,
		template.NODE_STRING,
		template.NODE_STRING,
		template.NODE_VARIABLE,
		template.NODE_FIELD,
	}
	reference := template.Node_Reference(first.First_Child)
	for _, kind := range kinds {
		node := node_at(t, document, &workspace, template.Node_Reference(reference))
		testify.Equal_Values(t, kind, node.Kind)
		reference = template.Node_Reference(node.Next_Sibling)
	}
	testify.Equal_Values(t, template.NO_NODE, reference)
	testify.Equal(t, "printf", node_value(
		source,
		node_at(t, document, &workspace, template.Node_Reference(first.First_Child)),
	))
}

func test_parentheses(t *testing.T) {
	t.Helper()
	configuration := configuration_input_from(t, template.Configuration_Input{
		Function: known_printf,
	})
	_, document, workspace := parse_configuration(
		t, `{{printf "%v" ((.Field)).Name}}`, configuration,
	)
	root := node_at(t, document, &workspace, template.Node_Reference(document.Root))
	action := node_at(t, document, &workspace, template.Node_Reference(root.First_Child))
	pipe := node_at(t, document, &workspace, template.Node_Reference(action.First_Child))
	command := node_at(t, document, &workspace, template.Node_Reference(pipe.First_Child))
	identifier := node_at(t, document, &workspace, template.Node_Reference(command.First_Child))
	format := node_at(t, document, &workspace, template.Node_Reference(identifier.Next_Sibling))
	chain := node_at(t, document, &workspace, template.Node_Reference(format.Next_Sibling))
	testify.Equal_Values(t, template.NODE_CHAIN, chain.Kind)
	inner_pipe := node_at(t, document, &workspace, template.Node_Reference(chain.First_Child))
	testify.Equal_Values(t, template.NODE_PIPE, inner_pipe.Kind)
	inner_command := node_at(
		t, document, &workspace, template.Node_Reference(inner_pipe.First_Child),
	)
	deeper_pipe := node_at(
		t, document, &workspace, template.Node_Reference(inner_command.First_Child),
	)
	testify.Equal_Values(t, template.NODE_PIPE, deeper_pipe.Kind)
}

func test_variables(t *testing.T) {
	t.Helper()
	source, document, workspace := parse_text(
		t, "{{$value := .Field}}{{$value = .Other}}{{$value}}",
	)
	root := node_at(t, document, &workspace, template.Node_Reference(document.Root))
	declaration := node_at(t, document, &workspace, template.Node_Reference(root.First_Child))
	declaration_pipe := node_at(
		t, document, &workspace, template.Node_Reference(declaration.First_Child),
	)
	variable := node_at(
		t, document, &workspace, template.Node_Reference(declaration_pipe.First_Child),
	)
	command := node_at(t, document, &workspace, template.Node_Reference(variable.Next_Sibling))
	testify.Equal_Values(t, template.NODE_VARIABLE, variable.Kind)
	testify.Equal_Values(t, template.NODE_COMMAND, command.Kind)
	testify.Equal(t, ":=", node_value(source, declaration_pipe))

	assignment := node_at(
		t, document, &workspace, template.Node_Reference(declaration.Next_Sibling),
	)
	assignment_pipe := node_at(
		t, document, &workspace, template.Node_Reference(assignment.First_Child),
	)
	testify.Equal(t, "=", node_value(source, assignment_pipe))
	use := node_at(t, document, &workspace, template.Node_Reference(assignment.Next_Sibling))
	use_pipe := node_at(t, document, &workspace, template.Node_Reference(use.First_Child))
	use_command := node_at(
		t, document, &workspace, template.Node_Reference(use_pipe.First_Child),
	)
	use_variable := node_at(
		t, document, &workspace, template.Node_Reference(use_command.First_Child),
	)
	testify.Equal_Values(t, template.NODE_VARIABLE, use_variable.Kind)
}

func test_branches(t *testing.T) {
	t.Helper()
	source, document, workspace := parse_text(
		t,
		"{{if .Ready}}yes{{else}}no{{end}}"+
			"{{with .Value}}with{{end}}"+
			"{{range $index, $value := .Items}}"+
			"{{$index}}:{{$value}}{{continue}}{{break}}"+
			"{{else}}empty{{end}}",
	)
	root := node_at(t, document, &workspace, template.Node_Reference(document.Root))
	if_node := node_at(t, document, &workspace, template.Node_Reference(root.First_Child))
	testify.Equal_Values(t, template.NODE_IF, if_node.Kind)
	if_pipe := node_at(t, document, &workspace, template.Node_Reference(if_node.First_Child))
	if_list := node_at(t, document, &workspace, template.Node_Reference(if_pipe.Next_Sibling))
	else_list := node_at(t, document, &workspace, template.Node_Reference(if_list.Next_Sibling))
	testify.Equal_Values(t, template.NODE_LIST, if_list.Kind)
	testify.Equal_Values(t, template.NODE_LIST, else_list.Kind)
	testify.Equal(t, "yes", node_value(
		source,
		node_at(t, document, &workspace, template.Node_Reference(if_list.First_Child)),
	))
	testify.Equal(t, "no", node_value(
		source,
		node_at(t, document, &workspace, template.Node_Reference(else_list.First_Child)),
	))
	test_range_branches(t)
	test_nested_branches(t)
}

func test_range_branches(t *testing.T) {
	t.Helper()
	_, document, workspace := parse_text(
		t,
		"{{if .Ready}}yes{{else}}no{{end}}"+
			"{{with .Value}}with{{end}}"+
			"{{range $index, $value := .Items}}"+
			"{{$index}}:{{$value}}{{continue}}{{break}}"+
			"{{else}}empty{{end}}",
	)
	root := node_at(t, document, &workspace, template.Node_Reference(document.Root))
	if_node := node_at(t, document, &workspace, template.Node_Reference(root.First_Child))
	with_node := node_at(t, document, &workspace, template.Node_Reference(if_node.Next_Sibling))
	range_node := node_at(
		t, document, &workspace, template.Node_Reference(with_node.Next_Sibling),
	)
	testify.Equal_Values(t, template.NODE_WITH, with_node.Kind)
	testify.Equal_Values(t, template.NODE_RANGE, range_node.Kind)
	range_pipe := node_at(
		t, document, &workspace, template.Node_Reference(range_node.First_Child),
	)
	first_variable := node_at(
		t, document, &workspace, template.Node_Reference(range_pipe.First_Child),
	)
	second_variable := node_at(
		t, document, &workspace, template.Node_Reference(first_variable.Next_Sibling),
	)
	range_command := node_at(
		t, document, &workspace, template.Node_Reference(second_variable.Next_Sibling),
	)
	testify.Equal_Values(t, template.NODE_VARIABLE, first_variable.Kind)
	testify.Equal_Values(t, template.NODE_VARIABLE, second_variable.Kind)
	testify.Equal_Values(t, template.NODE_COMMAND, range_command.Kind)
	range_list := node_at(
		t, document, &workspace, template.Node_Reference(range_pipe.Next_Sibling),
	)
	first_action := node_at(
		t, document, &workspace, template.Node_Reference(range_list.First_Child),
	)
	colon := node_at(
		t, document, &workspace, template.Node_Reference(first_action.Next_Sibling),
	)
	second_action := node_at(
		t, document, &workspace, template.Node_Reference(colon.Next_Sibling),
	)
	continue_node := node_at(
		t, document, &workspace, template.Node_Reference(second_action.Next_Sibling),
	)
	break_node := node_at(
		t, document, &workspace, template.Node_Reference(continue_node.Next_Sibling),
	)
	testify.Equal_Values(t, template.NODE_CONTINUE, continue_node.Kind)
	testify.Equal_Values(t, template.NODE_BREAK, break_node.Kind)
}

func test_nested_branches(t *testing.T) {
	t.Helper()
	_, document, workspace := parse_text(
		t, "{{if .A}}a{{else if .B}}b{{else with .C}}c{{else}}d{{end}}",
	)
	root := node_at(t, document, &workspace, template.Node_Reference(document.Root))
	if_node := node_at(t, document, &workspace, template.Node_Reference(root.First_Child))
	if_pipe := node_at(t, document, &workspace, template.Node_Reference(if_node.First_Child))
	if_list := node_at(t, document, &workspace, template.Node_Reference(if_pipe.Next_Sibling))
	else_list := node_at(t, document, &workspace, template.Node_Reference(if_list.Next_Sibling))
	nested_if := node_at(
		t, document, &workspace, template.Node_Reference(else_list.First_Child),
	)
	testify.Equal_Values(t, template.NODE_IF, nested_if.Kind)
	nested_pipe := node_at(
		t, document, &workspace, template.Node_Reference(nested_if.First_Child),
	)
	nested_list := node_at(
		t, document, &workspace, template.Node_Reference(nested_pipe.Next_Sibling),
	)
	nested_else := node_at(
		t, document, &workspace, template.Node_Reference(nested_list.Next_Sibling),
	)
	nested_with := node_at(
		t, document, &workspace, template.Node_Reference(nested_else.First_Child),
	)
	testify.Equal_Values(t, template.NODE_WITH, nested_with.Kind)
}

func test_comments(t *testing.T) {
	t.Helper()
	source, document, workspace := parse_text(t, "a{{/* }} inside */}}b")
	root := node_at(t, document, &workspace, template.Node_Reference(document.Root))
	left := node_at(t, document, &workspace, template.Node_Reference(root.First_Child))
	right := node_at(t, document, &workspace, template.Node_Reference(left.Next_Sibling))
	testify.Equal(t, "a", node_value(source, left))
	testify.Equal(t, "b", node_value(source, right))

	configuration := configuration_input_from(t, template.Configuration_Input{
		Mode: template.PARSE_COMMENTS,
	})
	source, document, workspace = parse_configuration(t, "a{{/* comment */}}b", configuration)
	root = node_at(t, document, &workspace, template.Node_Reference(document.Root))
	left = node_at(t, document, &workspace, template.Node_Reference(root.First_Child))
	comment := node_at(t, document, &workspace, template.Node_Reference(left.Next_Sibling))
	right = node_at(t, document, &workspace, template.Node_Reference(comment.Next_Sibling))
	testify.Equal_Values(t, template.NODE_COMMENT, comment.Kind)
	testify.Equal(t, "/* comment */", node_value(source, comment))
	testify.Equal(t, "b", node_value(source, right))

	configuration = configuration_input_from(t, template.Configuration_Input{
		Function: known_printf,
	})
	parse_configuration(t, `{{printf "}}"}}`, configuration)
}

func test_templates(t *testing.T) {
	t.Helper()
	source, document, workspace := parse_text(
		t, `root{{template "card" .}}{{define "card"}}card {{.}}{{end}}`,
	)
	root := node_at(t, document, &workspace, template.Node_Reference(document.Root))
	text := node_at(t, document, &workspace, template.Node_Reference(root.First_Child))
	invocation := node_at(t, document, &workspace, template.Node_Reference(text.Next_Sibling))
	testify.Equal_Values(t, template.NODE_TEMPLATE, invocation.Kind)
	testify.Equal(t, `"card"`, node_value(source, invocation))
	testify.Equal(t, template.Template_Count(TEST_COUNT_ONE), document.Template_Count)
	definition := node_at(
		t, document, &workspace,
		template.Node_Reference(first_template_reference(document)),
	)
	testify.Equal_Values(t, template.NODE_TEMPLATE, definition.Kind)
	testify.Equal(t, `"card"`, node_value(source, definition))
	body := node_at(t, document, &workspace, template.Node_Reference(definition.First_Child))
	testify.Equal_Values(t, template.NODE_LIST, body.Kind)
	testify.Equal(t, "card ", node_value(
		source, node_at(t, document, &workspace, template.Node_Reference(body.First_Child)),
	))

	source, document, workspace = parse_text(t, `{{block "default" .}}value {{.}}{{end}}`)
	root = node_at(t, document, &workspace, template.Node_Reference(document.Root))
	invocation = node_at(t, document, &workspace, template.Node_Reference(root.First_Child))
	testify.Equal_Values(t, template.NODE_TEMPLATE, invocation.Kind)
	testify.Equal(t, template.Template_Count(TEST_COUNT_ONE), document.Template_Count)
	definition = node_at(
		t, document, &workspace,
		template.Node_Reference(first_template_reference(document)),
	)
	testify.Equal(t, `"default"`, node_value(source, definition))

	_, document, _ = parse_text(t, `{{define "first"}}{{end}}{{define "second"}}{{end}}`)
	definitions := [...]string{"first", "second"}
	testify.Equal_Values(t, template.Template_Count(len(definitions)), document.Template_Count)
}

func parse_text(
	t *testing.T, text string,
) (source template.Source, document template.Document, workspace template.Syntax_Workspace) {
	t.Helper()
	source = source_from(t, text)
	configuration := configuration_from(t)
	document, _, status := template.Parse_Into(
		template.Source_Header_Unvalidated(source),
		configuration,
		parse_workspace_from(&workspace),
	)
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	return source, document, workspace
}

func parse_configuration(
	t *testing.T, text string, configuration template.Configuration,
) (source template.Source, document template.Document, workspace template.Syntax_Workspace) {
	t.Helper()
	source = source_from(t, text)
	document, _, status := template.Parse_Into(
		template.Source_Header_Unvalidated(source),
		configuration,
		parse_workspace_from(&workspace),
	)
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	return source, document, workspace
}

func source_from(t *testing.T, text string) (source template.Source) {
	t.Helper()
	source, status := template.Source_Validate(template.Source_Unvalidated(text))
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	return source
}

func configuration_from(t *testing.T) (configuration template.Configuration) {
	t.Helper()
	return configuration_input_from(t, template.Configuration_Input{})
}

func configuration_input_from(
	t *testing.T, input template.Configuration_Input,
) (configuration template.Configuration) {
	t.Helper()
	configuration, status := template.New_Configuration(input)
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	return configuration
}

func known_printf(name template.Parse_Function_Name) (known template.Function_Known) {
	if len(name) != len("printf") {
		return false
	}
	for index, character := range [...]byte{'p', 'r', 'i', 'n', 't', 'f'} {
		if name[index] != character {
			return false
		}
	}
	return true
}

func parse_workspace_from(
	workspace *template.Syntax_Workspace,
) (input template.Syntax_Workspace_Input) {
	syntax_workspace_prepare(workspace)
	input.State = (*template.Syntax_Workspace_Unvalidated)(workspace)
	input.Number = number_workspace()
	return input
}

func first_template_reference(document template.Document) (reference template.Node_Reference) {
	return template.Node_Reference(document.First_Template) +
		template.Node_Reference(template.ROOT_NODE_COUNT)
}

func node_at(
	t *testing.T, document template.Document, workspace *template.Syntax_Workspace,
	reference template.Node_Reference,
) (node template.Node) {
	t.Helper()
	testify.True(t, uint16(reference) != uint16(template.NO_NODE))
	testify.True(t, int(reference) <= int(document.Node_Count))
	return workspace.Nodes[int(reference)-int(template.NODE_COUNT_INCREMENT)]
}

func node_value(source template.Source, node template.Node) (value string) {
	return string(source.Data.Value[node.Value_Start:node.Value_End])
}

// Test_Standard_Library_Number_Validity matches upstream numeric syntax decisions.
func Test_Standard_Library_Number_Validity(t *testing.T) {
	cases := [...]string{
		"{{0}}",
		"{{-0}}",
		"{{73}}",
		"{{7_3}}",
		"{{0b10_010_01}}",
		"{{0B10_010_01}}",
		"{{073}}",
		"{{0o73}}",
		"{{0O73}}",
		"{{0x73}}",
		"{{0X73}}",
		"{{0x7_3}}",
		"{{-7}}",
		"{{+73}}",
		"{{1_2}}",
		"{{1e9}}",
		"{{-1e9}}",
		"{{-1.2}}",
		"{{1e19}}",
		"{{1e1_9}}",
		"{{1E19}}",
		"{{-1e19}}",
		"{{0x_1p4}}",
		"{{0X_1P4}}",
		"{{0x_1p-4}}",
		"{{4i}}",
		"{{-1.2+4.2i}}",
		"{{073i}}",
		"{{0i}}",
		"{{-1.2+0i}}",
		"{{-12+0i}}",
		"{{13+0i}}",
		"{{0123}}",
		"{{-0x0}}",
		"{{0xdeadbeef}}",
		"{{0x1.e_fp4}}",
		"{{1+2i}}",
		"{{.2}}",
		"{{0xef}}",
		"{{12abc}}",
		"{{0x}}",
		"{{1e}}",
		"{{1+2}}",
		"{{0b2}}",
		"{{0b1e1}}",
		"{{0o1e1}}",
		"{{1__2}}",
		"{{+-2}}",
		"{{0x123.}}",
		"{{1e.}}",
		"{{0xi.}}",
		"{{1+2.}}",
	}
	assert_standard_library_numbers(t, cases[:])
}

// Test_Standard_Library_Number_Bounds matches upstream machine value limits.
func Test_Standard_Library_Number_Bounds(t *testing.T) {
	cases := [...]string{
		"{{1e1000}}",
		"{{1e100000}}",
		"{{1e-10000}}",
		"{{18446744073709551615}}",
		"{{18446744073709551616}}",
		"{{+9223372036854775807}}",
		"{{+9223372036854775808}}",
		"{{-9223372036854775808}}",
		"{{-9223372036854775809}}",
		"{{08}}",
		"{{09}}",
		"{{1.7976931348623157e308}}",
		"{{1.7976931348623159e308}}",
		"{{0x1.fffffffffffffp1023}}",
		"{{0x1p1024}}",
		"{{1e1000i}}",
		"{{1+1e1000i}}",
		"{{08i}}",
		"{{0e100000}}",
		"{{0x0p100000}}",
	}
	assert_standard_library_numbers(t, cases[:])
}

// Test_Standard_Library_Quoted_Validity matches upstream Go literal decoding.
func Test_Standard_Library_Quoted_Validity(t *testing.T) {
	cases := [...]string{
		`{{"text\n"}}`,
		`{{'\n'}}`,
		`{{'\x7f'}}`,
		`{{'\u263a'}}`,
		"{{`raw\ntext`}}",
		`{{"\q"}}`,
		`{{'ab'}}`,
		`{{''}}`,
		`{{'\xG0'}}`,
		`{{'\400'}}`,
		`{{'\uD800'}}`,
		`{{"\'"}}`,
		`{{'\"'}}`,
		`{{"\377"}}`,
		`{{"\400"}}`,
		`{{"\u0000"}}`,
		`{{"\U00110000"}}`,
		`{{"\UFFFFFFFF"}}`,
		`{{"\xFF"}}`,
		`{{"\u263A"}}`,
		`{{"\8"}}`,
		`{{"\07"}}`,
	}
	configuration := configuration_from(t)
	for _, source_text := range cases {
		t.Run(source_text, func(t *testing.T) {
			_, standard_failure := parse.Parse("test", source_text, "", "")
			source := source_from(t, source_text)
			var workspace template.Syntax_Workspace
			_, _, status := template.Parse_Into(
				template.Source_Header_Unvalidated(source),
				configuration,
				parse_workspace_from(&workspace),
			)
			testify.Equal(
				t, standard_failure == nil, status == template.PARSE_STATUS_OK,
			)
		})
	}
}

// Test_Standard_Library_Invalid_UTF8_Validity matches dynamic malformed source handling.
func Test_Standard_Library_Invalid_UTF8_Validity(t *testing.T) {
	invalid := string([]byte{byte(utf8.CHARACTER_SELF)})
	cases := [...]string{
		"{{\"" + invalid + "\"}}",
		"{{'" + invalid + "'}}",
		"{{`" + invalid + "`}}",
		"{{." + invalid + "}}",
		"{{define \"" + invalid + "\"}}x{{end}}" +
			"{{define \"\\uFFFD\"}}y{{end}}",
	}
	configuration := configuration_from(t)
	for _, source_text := range cases {
		t.Run(source_text, func(t *testing.T) {
			_, standard_failure := parse.Parse("test", source_text, "", "")
			source := source_from(t, source_text)
			var workspace template.Syntax_Workspace
			_, _, status := template.Parse_Into(
				template.Source_Header_Unvalidated(source),
				configuration,
				parse_workspace_from(&workspace),
			)
			testify.Equal(
				t, standard_failure == nil, status == template.PARSE_STATUS_OK,
			)
		})
	}
}

// Test_Standard_Library_Grammar_Validity covers the complete upstream grammar forms.
func Test_Standard_Library_Grammar_Validity(t *testing.T) {
	cases := [...]string{
		"",
		"{{/*\n\n\n*/}}",
		" \t\n",
		"some text",
		"{{.X}}",
		"{{printf}}",
		"{{$}}",
		"{{with $x := 3}}{{$x 23}}{{end}}",
		"{{$.I}}",
		"{{printf `%d` 23}}",
		"{{.X|.Y}}",
		"{{$x := .X|.Y}}",
		"{{.X (.Y .Z) (.A | .B .C) (.E)}}",
		"{{(.Y .Z).Field}}",
		"{{if .X}}hello{{end}}",
		"{{if .X}}true{{else}}false{{end}}",
		"{{if .X}}true{{else if .Y}}false{{end}}",
		"+{{if .X}}X{{else if .Y}}Y{{else if .Z}}Z{{end}}+",
		"{{range .X}}hello{{end}}",
		"{{range .X.Y.Z}}hello{{end}}",
		"{{range .X}}hello{{range .Y}}goodbye{{end}}{{end}}",
		"{{range .X}}true{{else}}false{{end}}",
		"{{range .X|.M}}true{{else}}false{{end}}",
		"{{range .SI}}{{.}}{{end}}",
		"{{range $x := .SI}}{{.}}{{end}}",
		"{{range $x, $y := .SI}}{{.}}{{end}}",
		"{{range .SI}}{{.}}{{break}}{{end}}",
		"{{range .SI}}{{.}}{{continue}}{{end}}",
		"{{range .SI 1 -3.2i true false 'a' nil}}{{end}}",
		"{{template `x`}}",
		"{{template `x` .Y}}",
		"{{with .X}}hello{{end}}",
		"{{with .X}}hello{{else}}goodbye{{end}}",
		"{{with .X}}hello{{else with .Y}}goodbye{{end}}",
		"{{with .X}}X{{else with .Y}}Y{{else with .Z}}Z{{end}}",
		"x \r\n\t{{- 3}}",
		"{{3 -}}\n\n\ty",
		"x \r\n\t{{- 3 -}}\n\n\ty",
		"x\n{{-  3   -}}\ny",
		"x \r\n\t{{- /* hi */}}",
		"{{/* hi */ -}}\n\n\ty",
		"x \r\n\t{{- /* */ -}}\n\n\ty",
		`{{block "foo" .}}hello{{end}}`,
		"{{ $x \n := \n 1 \n }}",
		"{{\n\"x\"\n|\nprintf\n}}",
		"{{/*\nhello\n*/}}",
		"{{-\n/*\nhello\n*/\n-}}",
		"{{range .SI}}{{.}}{{ continue }}{{end}}",
		"{{range .SI}}{{.}}{{ break }}{{end}}",
		"{{$x := 0}}{{$x}}",
		"{{$x:=.}}{{$x +2}}",
		"{{range $x := 0}}{{$x}}{{end}}",
		"{{range $x = 0}}{{$x}}{{end}}",
		"{{ (( 1 )) }}",
		"{{ ((( 1 ))) | printf }}",
		"{{ if ((( true ))) }}YES{{ end }}",
		`{{define "a"}} {{end}}{{define ` + "`a`" + `}}x{{end}}`,
		`{{define "a"}}x{{end}}{{define ` + "`a`" + `}} {{end}}`,
		`{{define "a"}} {{end}}{{define ` + "`a`" + `}}
{{end}}`,
	}
	assert_standard_library_grammar(t, cases[:])
}

// Test_Standard_Library_Grammar_Rejection covers upstream malformed forms.
func Test_Standard_Library_Grammar_Rejection(t *testing.T) {
	cases := [...]string{
		"{{}}",
		"{{\n}}",
		"hello{{range",
		"{{end}}",
		"{{else}}",
		"{{if .X}}hello{{end}}{{else}}",
		"{{if .X}}1{{else}}2{{else}}3{{end}}",
		"hello{{range .x}}",
		"hello{{range .x}}{{else}}",
		"hello{{undefined}}",
		"{{$x}}",
		"{{with $x := 4}}{{end}}{{$x}}",
		"{{template $v}}",
		"{{with $x.Y := 4}}{{end}}",
		"{{template .X}}",
		"{{printf 3, 4}}",
		"{{with $v, $u := 3}}{{end}}",
		"{{range $u, $v, $w := 3}}{{end}}",
		"{{printf (printf .).}}",
		"{{printf 3`x`}}",
		"{{printf `x`.}}",
		"{{if .X}}a{{else if .Y}}b{{end}}{{end}}",
		"{{range .}}{{end}} {{break}}",
		"{{range .}}{{end}} {{continue}}",
		"{{range .}}{{else}}{{break}}{{end}}",
		"{{range .}}{{else}}{{continue}}{{end}}",
		"{{$x += 1}}{{$x}}",
		"{{$x ! 2}}{{$x}}",
		"{{$x % 3}}{{$x}}",
		"{{range $x := $y := 3}}{{end}}",
		"{{$x:=.}}{{$x!2}}",
		"{{$x:=.}}{{$x+2}}",
		"{{1.E}}",
		"{{0.1.E}}",
		"{{true.E}}",
		"{{'a'.any}}",
		`{{"hello".guys}}`,
		"{{..E}}",
		"{{nil.E}}",
		"{{12|.}}",
		"{{.|12|printf}}",
		"{{.|printf|\"error\"}}",
		"{{12|printf|'e'}}",
		"{{.|true}}",
		"{{'c'|nil}}",
		`{{printf "%d" ( ) }}`,
		`{{block "foo"}}hello{{end}}`,
		"{{define `a`}}a{{end}}{{define `a`}}b{{end}}",
		`{{define "\x61"}}x{{end}}{{define ` + "`a`" + `}}y{{end}}`,
		"{{$a,$b,$c := 23}}",
		"{{range $k, $v}}{{end}}",
		"{{range $k,}}{{end}}",
		"{{range $k, $v := }}{{end}}",
		"{{range $k, .}}{{end}}",
		"{{range $k, 123 := .}}{{end}}",
		"hello-{{/* */ }}-world",
	}
	assert_standard_library_grammar(t, cases[:])
}

// Test_Standard_Library_Keyword_Function_Collision preserves callable new keywords.
func Test_Standard_Library_Keyword_Function_Collision(t *testing.T) {
	source_text := "{{range .X}}{{break 20}}{{continue 30}}{{end}}"
	_, standard_failure := parse.Parse(
		"test", source_text, "", "", map[string]any{
			"break": func() {}, "continue": func() {},
		},
	)
	configuration := configuration_input_from(t, template.Configuration_Input{
		Function: known_flow_function,
	})
	source := source_from(t, source_text)
	var workspace template.Syntax_Workspace
	_, _, status := template.Parse_Into(
		template.Source_Header_Unvalidated(source),
		configuration,
		parse_workspace_from(&workspace),
	)
	testify.Equal(t, standard_failure == nil, status == template.PARSE_STATUS_OK)
}

func assert_standard_library_grammar(t *testing.T, cases []string) {
	t.Helper()
	configuration := configuration_input_from(t, template.Configuration_Input{
		Function: known_printf,
	})
	for _, source_text := range cases {
		t.Run(source_text, func(t *testing.T) {
			_, standard_failure := parse.Parse(
				"test", source_text, "", "", map[string]any{"printf": func() {}},
			)
			source := source_from(t, source_text)
			var workspace template.Syntax_Workspace
			_, _, status := template.Parse_Into(
				template.Source_Header_Unvalidated(source),
				configuration,
				parse_workspace_from(&workspace),
			)
			testify.Equal(
				t, standard_failure == nil, status == template.PARSE_STATUS_OK,
			)
		})
	}
}

func assert_standard_library_numbers(t *testing.T, cases []string) {
	t.Helper()
	configuration := configuration_from(t)
	for _, source_text := range cases {
		t.Run(source_text, func(t *testing.T) {
			_, standard_failure := parse.Parse("test", source_text, "", "")
			source := source_from(t, source_text)
			var workspace template.Syntax_Workspace
			_, _, status := template.Parse_Into(
				template.Source_Header_Unvalidated(source),
				configuration,
				parse_workspace_from(&workspace),
			)
			testify.Equal(
				t, standard_failure == nil, status == template.PARSE_STATUS_OK,
			)
		})
	}
}

func known_flow_function(name template.Parse_Function_Name) (known template.Function_Known) {
	if len(name) == len("break") {
		for index, character := range [...]byte{'b', 'r', 'e', 'a', 'k'} {
			if name[index] != character {
				return false
			}
		}
		return true
	}
	if len(name) != len("continue") {
		return false
	}
	for index, character := range [...]byte{'c', 'o', 'n', 't', 'i', 'n', 'u', 'e'} {
		if name[index] != character {
			return false
		}
	}
	return true
}

func test_float_initialize(value *big.Float) {
	value.Mantissa.Words = make(big.Words, big.WORD_COUNT_MAXIMUM)
}

func test_float_parse_workspace_initialize(workspace *big.Float_Parse_Workspace) {
	workspace.Source = make(big.Float_Parse_Source, big.FLOAT_PARSE_TEXT_SIZE_MAXIMUM)
	workspace.Control = make(big.Float_Parse_Control, big.FLOAT_PARSE_CONTROL_COUNT)
	workspace.Parse.Words = make(big.Int_Parse_Words, big.WORD_COUNT_MAXIMUM)
	workspace.Integers.Numerator.Words = make(big.Words, big.WORD_COUNT_MAXIMUM)
	workspace.Integers.Denominator.Words = make(big.Words, big.WORD_COUNT_MAXIMUM)
	test_float_initialize((*big.Float)(&workspace.Values.Numerator))
	test_float_initialize((*big.Float)(&workspace.Values.Denominator))
	test_float_initialize((*big.Float)(&workspace.Values.Result))
	workspace.Division.Remainder = make(
		big.Float_Division_Remainder, big.FLOAT_DIVISION_WORD_COUNT_MAXIMUM,
	)
	workspace.Division.Divisor = make(
		big.Float_Division_Divisor, big.FLOAT_DIVISION_WORD_COUNT_MAXIMUM,
	)
}

func test_float_text_workspace_initialize(workspace *big.Float_Text_Workspace) {
	workspace.Integers.Significand.Words = make(big.Words, big.WORD_COUNT_MAXIMUM)
	workspace.Integers.Exponent.Words = make(big.Words, big.WORD_COUNT_MAXIMUM)
	test_float_initialize((*big.Float)(&workspace.Values.Source))
	test_float_initialize((*big.Float)(&workspace.Values.Rounded))
	workspace.Decimals.Value.Digits = make(
		big.Float_Text_Decimal_Digits, big.FLOAT_DECIMAL_DIGIT_COUNT_MAXIMUM,
	)
	workspace.Decimals.Lower.Digits = make(
		big.Float_Text_Decimal_Digits, big.FLOAT_DECIMAL_DIGIT_COUNT_MAXIMUM,
	)
	workspace.Decimals.Upper.Digits = make(
		big.Float_Text_Decimal_Digits, big.FLOAT_DECIMAL_DIGIT_COUNT_MAXIMUM,
	)
	workspace.Text.Significand.Words = make(big.Int_Text_Words, big.WORD_COUNT_MAXIMUM)
	workspace.Text.Significand.Digits = make(big.Int_Text_Digits, big.INT_TEXT_SIZE_MAXIMUM)
	workspace.Text.Exponent.Words = make(big.Int_Text_Words, big.WORD_COUNT_MAXIMUM)
	workspace.Text.Exponent.Digits = make(big.Int_Text_Digits, big.INT_TEXT_SIZE_MAXIMUM)
	workspace.Control = &big.Float_Text_Control{}
	workspace.Magnitude.Words = make(
		big.Float_Text_Magnitude_Words, big.FLOAT_TEXT_MAGNITUDE_WORD_COUNT_MAXIMUM,
	)
	workspace.Output = make(big.Float_Text_Output, big.FLOAT_TEXT_SIZE_MAXIMUM)
}

// Gives fresh caller numeric validation scratch.
func number_workspace() (number *template.Float_Parse_Unvalidated) {
	var workspace big.Float_Parse_Workspace
	test_float_parse_workspace_initialize(&workspace)
	return (*template.Float_Parse_Unvalidated)(&workspace)
}

// Treats one test-built value as already validated.
func trusted(unvalidated template.Value_Unvalidated) (value template.Value) {
	return template.Value(unvalidated)
}

// Each term stands one sibling before the arena end, behind the densest packing.
func Test_Dense_Term_Siblings(t *testing.T) {
	t.Parallel()
	test_dense_term_siblings(t, TEST_COUNT_ZERO)
}

// Test_Dense_Term_Tails packs the second third of those terms.
func Test_Dense_Term_Tails(t *testing.T) {
	t.Parallel()
	test_dense_term_siblings(t, TEST_COUNT_ONE)
}

// Test_Dense_Term_Ends packs the last third of those terms.
func Test_Dense_Term_Ends(t *testing.T) {
	t.Parallel()
	test_dense_term_siblings(t, TEST_COUNT_TWO)
}

func test_dense_term_siblings(t *testing.T, part int) {
	t.Helper()
	tails := [...]string{
		`<x 1 .>`, `<x "" .>`, `<x $ .>`, `<x x .>`, `<x (.).a .>`, `<(.).a .>`, `<$>`,
		`<(.)>`, `<x (.)>`, `<x (.) .>`, `<x (.)|x>`, `<(.).a>`, `<define ""><end>`,
		`<block "" $><end>`, `<x $>`, `<.|$>`, `<template "">`, `<template "" .>`,
		`<block "" 1><end>`, `<block "" ""><end>`, `<block "" x x><end>`,
		`<block "" x><end>`, `<block "" (.).a><end>`,
		`<range 1><end><define ""><end>`, `<block "" .><range 1><end><end>`,
		`<range 1><else><end>`, `<if 1><end>x`, `<with 1><end>x`,
		`<block "" .><end><define "a"><end>`, `<define ""><end><block "" .><end>`,
		`<block "" .><end><define ""><end>`,
		`<if 1><block "" .><end><end>`,
		`<if 1><template ""><end>`, `<block "a" .><end><define ""><end><define ""><end>`,
		`<range $a:=1><end>`, `<range $a,$b:=1><end>`,
		`<if $a:=1><end>`, `<x $:=1>`, `<range $a,$:=1><end>`,
		`<$:=1>`, `<range $,$:=1><end>`, `<$:=1><$>`,
		`<$:=1><define ""><end>`, `<block "" $:=.><end>`,
		`<if 1><if 1><else><end><end>`, `<if 1><else><end>`,
		`<if 1>x<else><end>`, `<if 1></**/><end>`, `<template "">x`, `<$:=(.)>`, `<$=(.)>`,
		`<range $,$:=(.)><end>`, `<$=1>`, `<range $,$=1><end>`, `<block "" $=.><end>`,
		`<if $=1><end>`, `<x $=1>`, `<range $=1><end>`, `<range $,$=(.)><end>`, `<$=>`,
		`<x $=>`, `<$,$=>`, `<else>`, `<end>`, `<block>`, `<define>`, `<if 1><else>`, `<if 1><else if 1>`, `<if 1>`,
		`<block "" 1>`, `<define "">`, `x<block>`, `<if 1><block>`, `x<if 1>`, `<if 1><if 1>`, `x<end>`, `<if 1>x<else>`, `x<if 1><else>`, `x<else>`, `<break>`, `x<break>`, `<if 1><break>`, `<if 1><.>`, `<if 1>x`,
	}
	functions := [...]template.Function{{Name: []byte("x"), Call: identity}}
	fields := [...]template.Field{{
		Name: []byte("a"), Value: template.Field_Value{Data: value_text(nil)},
	}}
	var execution execution_storage
	for index, tail := range tails {
		if index%TEST_COUNT_THREE != part {
			continue
		}
		space := template.SOURCE_SIZE_MAXIMUM - len(tail)
		source := make([]byte, space%len("<.>"), template.SOURCE_SIZE_MAXIMUM)
		for position := range source {
			source[position] = 'x'
		}
		for len(source) < space {
			source = append(source, "<.>"...)
		}
		source = append(source, tail...)
		var workspace template.Syntax_Workspace
		program, _, status := template.Compile(template.Compile_Input{
			Source: source, Left: []byte("<"), Right: []byte(">"),
			Mode:      template.SKIP_FUNCTION_CHECK | template.PARSE_COMMENTS,
			Workspace: syntax_from(&workspace),
		})
		if status != template.PARSE_STATUS_OK {
			continue
		}
		// Only a field read needs the object; empty text keeps every packed action silent.
		dot := value_text(nil)
		for position := TEST_COUNT_ONE; position < len(tail); position++ {
			if tail[position-TEST_COUNT_ONE] == '.' {
				if tail[position] == 'a' {
					dot = value_object(fields[:])
				}
			}
		}
		execute_shared_from(t, &execution, program, dot, functions[:])
	}
}

// The least key of the widest unordered map stands last, so its successor scan starts there.
func Test_Wide_Map_Scan(t *testing.T) {
	t.Parallel()
	entries := make([]template.Field, template.RANGE_MAP_ENTRY_COUNT_MAXIMUM)
	for index := range entries {
		key := template.Integer(index + TEST_COUNT_ONE)
		if index == len(entries)-TEST_COUNT_ONE {
			key = template.Integer(TEST_COUNT_ZERO)
		}
		entries[index] = template.Field{Key: template.Field_Key{Data: value_integer(key)}}
	}
	// The final entry sorts first, and its wide value fills the output soon after.
	wide := make([]byte, template.OUTPUT_SIZE_MAXIMUM-len("0map[0: ]"))
	entries[len(entries)-TEST_COUNT_ONE].Value.Data = value_text(wide)
	program := compile_from(t,
		`{{range $k, $v := .}}{{if eq $k 1}}{{break}}{{end}}{{$k}}{{end}}{{.}}`)
	_, status := execute_from(t, program, value_map(entries))
	testify.Equal_Values(t, template.STATUS_OUTPUT_TOO_LARGE, status)
}

// Each payload at its maximum extent crosses one field, one child, one branch and one index.
func Test_Large_Payloads(t *testing.T) {
	t.Parallel()
	values := matrix_largest(matrix_scalars(), matrix_text())
	for _, largest := range values[:len(values)-TEST_COUNT_ONE] {
		wrapper := [...]template.Field{{
			Name: []byte("a"), Value: template.Field_Value{Data: largest},
		}}
		values = append(values, value_object(wrapper[:]))
	}
	values = append(values, value_integer(template.VALUE_COUNT_MAXIMUM))
	sources := [...]string{
		`{{.a}}`, `{{.a.b}}`, `{{(.).a}}`, `{{print (identity .)}}`, `{{and (.) .}}`,
		`{{or (.) .}}`, `{{identity . | len}}`, `{{index "ab" .}}`, `{{slice "ab" .}}`,
		`{{slice . 0}}`, `{{range .}}{{end}}`, `{{. | identity (identity .)}}`,
		`{{. | identity .}}`, `{{. | and . .}}`, `{{. | or . .}}`,
		`{{$a := printf "%s" .}}{{$b := printf "%s" .}}{{$c := printf ""}}`,
	}
	functions := [...]template.Function{{Name: []byte("identity"), Call: identity}}
	var execution execution_storage
	for _, source := range sources {
		program := compile_from(t, source)
		for _, value := range values {
			execute_shared_from(t, &execution, program, value, functions[:])
		}
	}
}

// Test_Value_Matrix drives every value bound through every generic evaluation construct.
func Test_Value_Matrix(t *testing.T) {
	t.Parallel()
	test_value_matrix(t, TEST_COUNT_ZERO)
}

// Test_Value_Matrix_Tails runs the odd templates over every value.
func Test_Value_Matrix_Tails(t *testing.T) {
	t.Parallel()
	test_value_matrix(t, TEST_COUNT_ONE)
}

func test_value_matrix(t *testing.T, part int) {
	t.Helper()
	values := matrix_values()
	sources := matrix_sources()
	functions := [...]template.Function{
		{Name: []byte("identity"), Call: identity},
		{Name: []byte("listing"), Call: listing},
		{Name: []byte("mapping"), Call: mapping},
	}
	// Payloads at their maximum extent cost one scan each, so fewer templates carry them.
	var execution execution_storage
	for index, source := range sources {
		if index%TEST_COUNT_TWO != part {
			continue
		}
		program := compile_from(t, source)
		for _, value := range values {
			if matrix_large(value) {
				if index%TEST_COUNT_THREE != TEST_COUNT_ZERO {
					continue
				}
			}
			execute_shared_from(t, &execution, program, value, functions[:])
		}
	}
	if part == TEST_COUNT_ONE {
		return
	}
	// Comment padding puts each action at the end of the largest source.
	padded := []template.Value{values[TEST_COUNT_ZERO], value_function(identity)}
	for offset, source := range sources {
		if offset%TEST_PADDED_STRIDE != TEST_COUNT_ZERO {
			continue
		}
		padding_size := template.SOURCE_SIZE_MAXIMUM - len(source) - len("{{/**/}}")
		comment := make([]byte, padding_size)
		for index := range comment {
			comment[index] = 'a'
		}
		program := compile_from(t, "{{/*"+string(comment)+"*/}}"+source)
		for index := range padded {
			execute_shared_from(t, &execution, program, padded[index], functions[:])
		}
	}
	// Text extraction is one caller entry point over every value kind.
	for index := range values {
		template.Value_As_Text(values[index])
	}
	// Map iteration selects each next key by scan, so the largest map takes few templates.
	largest_map := matrix_largest_map(matrix_scalars())
	map_sources := [...]string{
		`{{len .}}`, `{{index . 0}}`, `{{range $k, $v := .}}{{end}}`, `{{with .}}{{end}}`,
	}
	for _, source := range map_sources {
		execute_functions_from(t, compile_from(t, source), largest_map, functions[:])
	}
}

// One value of each kind stands at the second step its storage counts.
func Test_Second_Values(t *testing.T) {
	t.Parallel()
	var execution execution_storage
	members := [...]template.Field{
		{
			Name: []byte("a"),
			Value: template.Field_Value{Data: value_complex(template.Complex{
				Imaginary: TEST_COUNT_TWO,
			})},
		},
		{Name: []byte("b"), Value: template.Field_Value{Data: value_nil()}},
	}
	items := [...]template.Value{value_nil(), value_nil()}
	execute_shared_from(t, &execution, compile_from(t, `{{.a}}`),
		value_object(members[:]), nil)
	for _, source := range [...]string{
		`{{$a := .}}{{block "b" .}}x{{end}}`,
		`{{$a := .}}{{$c := .}}{{block "b" .}}x{{end}}`,
		`{{$a := .}}{{range .}}{{break}}{{end}}`,
		`{{$a := .}}{{$c := .}}{{range .}}{{continue}}{{end}}`,
	} {
		execute_shared_from(t, &execution, compile_from(t, source),
			value_sequence(items[:]), nil)
	}
	keys := [...]template.Value{value_sequence(items[:]), value_object(members[:])}
	for _, key := range keys {
		entries := [...]template.Field{{Key: template.Field_Key{Data: key}}}
		execute_shared_from(t, &execution, compile_from(t, `{{range .}}{{end}}`),
			value_map(entries[:]), nil)
	}
}

// One definition stands below each term that reads the program behind it.
func Test_Second_Programs(t *testing.T) {
	t.Parallel()
	var execution execution_storage
	single := [...]template.Field{
		{Name: []byte("a"), Value: template.Field_Value{Data: value_nil()}},
	}
	object := value_object(single[:])
	for _, source := range [...]string{
		`{{define "t"}}x{{end}}{{$a := .}}{{$a}}`,
		`{{define "t"}}x{{end}}{{print printf}}`,
		`{{define "t"}}x{{end}}{{.a.a}}`,
		`{{define "a"}}x{{end}}{{define "b"}}y{{end}}{{"z"}}`,
		`{{range .}}{{$a := .}}{{break}}{{end}}`,
		`{{range .}}{{$a := .}}{{$c := .}}{{continue}}{{end}}`,
		`{{slice .}}`, `{{1i}}`, `{{index . .}}`, `{{slice . .}}`,
	} {
		var workspace template.Syntax_Workspace
		program, _, status := template.Compile(template.Compile_Input{
			Source: []byte(source), Mode: template.SKIP_FUNCTION_CHECK,
			Workspace: syntax_from(&workspace),
		})
		if status != template.PARSE_STATUS_OK {
			continue
		}
		execute_shared_from(t, &execution, program, object, nil)
	}
	wide := make([]byte, template.OUTPUT_SIZE_MAXIMUM)
	items := [...]template.Value{value_text(wide), value_text(wide)}
	_, status := execute_from(t, compile_from(t, `{{.}}`), value_sequence(items[:]))
	testify.Equal_Values(t, template.STATUS_OUTPUT_TOO_LARGE, status)
}

// One function term stands at the end of the densest source that still holds it.
func Test_Dense_Function_Term(t *testing.T) {
	t.Parallel()
	tail := `<eq b>`
	source := []byte{}
	for len(source)+len(`<.>`)+len(tail) <= template.SOURCE_SIZE_MAXIMUM {
		source = append(source, `<.>`...)
	}
	source = append(source, tail...)
	var workspace template.Syntax_Workspace
	program, _, status := template.Compile(template.Compile_Input{
		Source: source, Left: []byte("<"), Right: []byte(">"),
		Mode: template.SKIP_FUNCTION_CHECK, Workspace: syntax_from(&workspace),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	var execution execution_storage
	execute_shared_from(t, &execution, program, value_text(nil), nil)
}

// One child call stands at the end of the densest source that still holds it.
func Test_Dense_Child_Call(t *testing.T) {
	t.Parallel()
	tail := `<print (.) .>`
	source := []byte{}
	for len(source)+len(`<.>`)+len(tail) <= template.SOURCE_SIZE_MAXIMUM {
		source = append(source, `<.>`...)
	}
	source = append(source, tail...)
	var workspace template.Syntax_Workspace
	program, _, status := template.Compile(template.Compile_Input{
		Source: source, Left: []byte("<"), Right: []byte(">"),
		Workspace: syntax_from(&workspace),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	var execution execution_storage
	execute_shared_from(t, &execution, program, value_text(nil), nil)
}

// One variable name fills the action its source admits.
func Test_Wide_Variable_Name(t *testing.T) {
	t.Parallel()
	name := make([]byte, template.SOURCE_SIZE_MAXIMUM-len(`<$>`))
	for index := range name {
		name[index] = 'a'
	}
	source := append(append([]byte(`<$`), name...), '>')
	var workspace template.Syntax_Workspace
	_, _, status := template.Compile(template.Compile_Input{
		Source: source, Left: []byte("<"), Right: []byte(">"),
		Workspace: syntax_from(&workspace),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_SYNTAX_INVALID, status)
}

// One map key carries the widest payload its storage admits.
func Test_Wide_Map_Key(t *testing.T) {
	t.Parallel()
	program := compile_from(t, `{{range $key, $entry := .}}{{end}}`)
	items := make([]template.Value, template.VALUE_COUNT_MAXIMUM)
	members := make([]template.Field, template.VALUE_COUNT_MAXIMUM)
	for index := range members {
		members[index] = template.Field{Name: []byte("a")}
	}
	keys := [...]template.Value{value_sequence(items), value_object(members)}
	var execution execution_storage
	for _, key := range keys {
		entries := [...]template.Field{{Key: template.Field_Key{Data: key}}}
		execute_shared_from(t, &execution, program, value_map(entries[:]), nil)
	}
}

// One range orders maps of one, two, and every admitted key kind.
func Test_Small_Map_Order(t *testing.T) {
	t.Parallel()
	program := compile_from(t, `{{range $key, $entry := .}}{{$key}}{{$entry}}{{end}}`)
	entries := [...]template.Field{
		{
			Key:   template.Field_Key{Data: value_text([]byte("b"))},
			Value: template.Field_Value{Data: value_integer(TEST_COUNT_TWO)},
		},
		{
			Key:   template.Field_Key{Data: value_text([]byte("a"))},
			Value: template.Field_Value{Data: value_integer(TEST_COUNT_ONE)},
		},
	}
	var execution execution_storage
	for _, size := range [...]int{TEST_COUNT_ONE, TEST_COUNT_TWO} {
		execute_shared_from(t, &execution, program, value_map(entries[:size]), nil)
	}
}

// One index reads every value kind out of one sequence and one map.
func Test_Indexed_Kinds(t *testing.T) {
	t.Parallel()
	scalars := matrix_scalars()
	entries := make([]template.Field, len(scalars))
	for index, scalar := range scalars {
		entries[index] = template.Field{
			Key:   template.Field_Key{Data: value_integer(template.Integer(index))},
			Value: template.Field_Value{Data: scalar},
		}
	}
	pair := [...]template.Value{value_nil(), value_nil()}
	members := [...]template.Field{
		{Name: []byte("a"), Value: template.Field_Value{Data: value_nil()}},
		{Name: []byte("b"), Value: template.Field_Value{Data: value_nil()}},
	}
	composites := [...]template.Value{
		value_sequence(pair[:]), value_object(members[:TEST_COUNT_ONE]),
		value_object(members[:]),
	}
	var execution execution_storage
	sequences := [...]template.Value{
		value_sequence(composites[:]), value_sequence(scalars),
	}
	digits := "0123456789"
	for index := range digits {
		program := compile_from(t, `{{index . `+digits[index:index+1]+`}}`)
		for _, sequence := range sequences {
			execute_shared_from(t, &execution, program, sequence, nil)
		}
	}
	program := compile_from(t, `{{range $key, $entry := .}}{{index $entry}}{{end}}`)
	execute_shared_from(t, &execution, program, value_map(entries), nil)
}

// One overflowing number stands at the end of the largest source.
func Test_Wide_Overflow(t *testing.T) {
	t.Parallel()
	action := `<1e999999>`
	padded := make([]byte, template.SOURCE_SIZE_MAXIMUM-len(action))
	for index := range padded {
		padded[index] = 'x'
	}
	sources := [...][]byte{
		append(padded, action...), []byte(action), []byte(`<1e309>`),
	}
	var execution execution_storage
	for _, source := range sources {
		var workspace template.Syntax_Workspace
		program, _, status := template.Compile(template.Compile_Input{
			Source: source, Left: []byte("<"), Right: []byte(">"),
			Workspace: syntax_from(&workspace),
		})
		if status != template.PARSE_STATUS_OK {
			continue
		}
		execute_shared_from(t, &execution, program, value_nil(), nil)
	}
}

// Each source stands one construct deeper than its shortest form.
func Test_Second_Depth(t *testing.T) {
	t.Parallel()
	sources := [...]string{
		`{{if .}}{{if .}}x{{end}}{{end}}`,
		`{{range .}}{{range .}}x{{end}}{{end}}`,
		`{{with .}}{{with .}}x{{end}}{{end}}`,
		`{{define "t"}}{{if .}}x{{end}}{{end}}{{template "t" .}}`,
		`{{define "t"}}{{define "u"}}x{{end}}{{end}}{{template "t" .}}`,
		`{{not . .}}`, `{{len . .}}`, `{{print ((.))}}`,
		`{{$x := .}}{{if .}}{{$x}}{{end}}`,
		`{{range .}}{{break}}{{end}}`,
		`{{if .}}{{else if .}}x{{end}}`,
		`{{with .}}{{else}}x{{end}}`,
		`{{printf "%v%v" . .}}`,
		`{{if .}}{{define "t"}}x{{end}}{{end}}{{template "t" .}}`,
		`{{if .}}{{block "b" .}}x{{end}}{{end}}`,
		`{{if .}}{{range .}}{{break}}{{end}}{{end}}`,
		`{{range .}}{{if .}}{{continue}}{{end}}{{end}}`,
		`{{if .}}{{$y := .}}{{$y}}{{end}}`,
		`{{define "t"}}x{{end}}{{with .}}{{template "t" .}}{{end}}`,
		`{{range .}}{{$z := .}}{{range .}}{{$z}}{{end}}{{end}}`,
	}
	items := [...]template.Value{value_text(nil), value_text(nil)}
	dots := [...]template.Value{value_nil(), value_sequence(items[:])}
	var execution execution_storage
	for _, source := range sources {
		var workspace template.Syntax_Workspace
		program, _, status := template.Compile(template.Compile_Input{
			Source: []byte(source), Workspace: syntax_from(&workspace),
		})
		if status != template.PARSE_STATUS_OK {
			continue
		}
		for _, dot := range dots {
			execute_shared_from(t, &execution, program, dot, nil)
		}
	}
}

// Each widest payload crosses one child frame, one argument, and one variable.
func Test_Wide_Delivery(t *testing.T) {
	t.Parallel()
	scalars := matrix_scalars()
	values := matrix_largest(scalars, matrix_text())
	values = append(values, matrix_largest_map(scalars))
	sources := [...]string{
		`{{identity (identity .)}}`, `{{print (identity .)}}`, `{{len (identity .)}}`,
		`{{$x := identity .}}{{len $x}}`, `{{with (identity .)}}{{len .}}{{end}}`,
		`{{range (identity .)}}{{end}}`, `{{index (identity .) 0}}`,
		`{{define "t"}}{{len .}}{{end}}{{template "t" (identity .)}}`,
		`{{identity . | identity | len}}`, `{{eq (identity .) (identity .)}}`,
		`{{if (identity .)}}{{len .}}{{end}}`, `{{len (index (listing) 0)}}`,
	}
	functions := [...]template.Function{
		{Name: []byte("identity"), Call: identity},
		{Name: []byte("listing"), Call: listing},
		{Name: []byte("mapping"), Call: mapping},
	}
	var execution execution_storage
	for _, source := range sources {
		program := compile_from(t, source)
		for _, value := range values {
			execute_shared_from(t, &execution, program, value, functions[:])
		}
	}
}

// Dense_program packs the most shortest actions one source admits beside its tail.
func dense_program(tail string) (program template.Program, compiled bool) {
	source := append([]byte{}, `<define "t"><.><end><define "u"><.><end>`...)
	for len(source)+len("<.>")+len(tail) <= template.SOURCE_SIZE_MAXIMUM {
		source = append(source, "<.>"...)
	}
	source = append(source, tail...)
	var workspace template.Syntax_Workspace
	program, _, status := template.Compile(template.Compile_Input{
		Source: source, Left: []byte("<"), Right: []byte(">"),
		Mode: template.SKIP_FUNCTION_CHECK, Workspace: syntax_from(&workspace),
	})
	return program, status == template.PARSE_STATUS_OK
}

// Each dense program stands beside every construct one dot kind reaches.
func dense_execution(t *testing.T, part int) {
	t.Helper()
	tails := [...]string{
		`<.>`, `<range .>x<end>`, `<with .>x<end>`, `<if .>x<else>y<end>`,
		`<$x:=.><$x>`, `<identity .>`, `<.a>`, `<1>`, `<. | identity>`,
		`<range $i, $e := .>x<end>`, `<index . 0>`, `<len .>`, `<print . .>`,
		`<(.).a>`, `<define "t"><.><end><template "t" .>`,
		`<define "t"><.><end><define "u"><.><end><template "u" .>`,
	}
	functions := [...]template.Function{{Name: []byte("identity"), Call: identity}}
	var execution execution_storage
	for index, tail := range tails {
		if index%TEST_COUNT_TWO != part {
			continue
		}
		program, compiled := dense_program(tail)
		if !compiled {
			continue
		}
		// Empty text keeps packed actions silent, so the tail runs inside the output bound.
		execute_shared_from(t, &execution, program, value_text(nil), functions[:])
	}
}

// Test_Dense_Actions packs the shortest action beside each construct.
func Test_Dense_Actions(t *testing.T) {
	t.Parallel()
	dense_execution(t, TEST_COUNT_ZERO)
}

// Test_Dense_Action_Tails packs the shortest action beside the remaining constructs.
func Test_Dense_Action_Tails(t *testing.T) {
	t.Parallel()
	dense_execution(t, TEST_COUNT_ONE)
}

// Test_Oversized_Literal refuses a literal whose malformed bytes decode past one text value.
func Test_Oversized_Literal(t *testing.T) {
	source := make([]byte, template.SOURCE_SIZE_MAXIMUM)
	for index := range source {
		source[index] = bits.WORD_8_MAXIMUM
	}
	copy(source, `{{"`)
	copy(source[len(source)-len(`"}}`):], `"}}`)
	program := compile_from(t, string(source))
	_, status := execute_from(t, program, value_nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)
}

// Test_Source_Maximum places each construct at the end of the largest source.
func Test_Source_Maximum(t *testing.T) {
	t.Parallel()
	constructs := [...]string{
		`<.>`, `<- . ->`, "<print 1 2.5 'a' \"s\\n\" `r` 0x1F 1e3 1i -1 +1>",
		`<$x := .><$x = 1><$x>`, `<if .>a<else if .>b<else>c<end>`,
		`<with $x := .>a<else with .>b<end>`, `<range $i, $e := .><break><continue><end>`,
		`<range .>x<else>y<end>`, `<define "t">x<end><template "t" .>`,
		`<block "b" .>x<end>`, `<(.).a.b>`, `<.a.b>`, `<$.a>`, `<nil>`, `<true>`,
		`<and . .>`, `<. | print | printf "%v">`, `</* c */>`, `<- /* c */ ->`, `<(1)>`,
		`<((.))>`, `<template "t">`, `<if .><end>`, `<with .><else><end>`,
		// Shortest source each construct needs.
		`<>`, `<x>`, `<0>`, `<$>`, `<"">`, `<''>`, "<``>", `</**/>`, `<if>`,
		`<end>`, `<else>`, `<range>`, `<with>`, `<define>`, `<block>`, `<break>`,
		`<continue>`, `<template>`, `<.x>`, `<$x>`, `<0x1>`, `<1e1>`, `<-1>`, `<0.5>`,
		`<1i>`, `<01>`, `<"\a">`, `<"\0">`, `<"\x00">`, `<"\u0000">`, `<'a'>`,
		`<x y>`, `<x|y>`, `<$x:=.>`, `<$x=.>`, `<.|x>`, `<(.)>`, `<true>`, `<if .>`,
		`<define "">`, `<block "">`, `<template "">`, `<template "" .>`,
		`<define ""><end>`, `<if .><else>`, `<range .><break>`, `<- .>`, `<. ->`,
		`<1e-1>`, `<"\x0">`, `<block ""><end>`, "<block ``><end>", "<define ``><end>", `<define ""><end><define ""><end>`,
		`<template "" .>`, `<0.>`, `<.0>`, `<x.y>`, `<$x><$x>`, `<if x><end>`,
	}
	modes := [...]template.Mode{
		template.MODE_MINIMUM, template.PARSE_COMMENTS,
		template.SKIP_FUNCTION_CHECK, template.MODE_MAXIMUM,
	}
	functions := [...]template.Function{{Name: []byte("identity"), Call: identity}}
	scalars := matrix_scalars()
	executed := [...]template.Value{
		value_nil(), scalars[len(scalars)-TEST_COUNT_ONE],
		value_sequence(scalars[:TEST_COUNT_TWO]),
	}
	var execution execution_storage
	var syntax template.Syntax_Workspace
	for _, construct := range constructs {
		for _, mode := range modes {
			test_construct_placements(Construct_Run{
				Test: t, Execution: &execution, Syntax: &syntax,
				Source: []byte(construct), Mode: mode,
				Values: executed[:], Functions: functions[:], Left: "<", Right: ">",
			})
			test_explicit_node_limit(t, []byte(construct), mode, "<", ">")
		}
	}
}

// Each construct stands at the bound its own storage admits.
func Test_Construct_Bounds(t *testing.T) {
	t.Parallel()
	test_definition_maximum(t)
	test_nesting_maximum(t)
}

// Test_Recursion_Bounds recurses to the deepest invocation one workspace holds.
func Test_Recursion_Bounds(t *testing.T) {
	t.Parallel()
	test_recursion_maximum(t)
}

// Test_Frame_Stack_Bounds stacks the deepest frames one recursion reaches.
func Test_Frame_Stack_Bounds(t *testing.T) {
	t.Parallel()
	test_frame_stack_maximum(t)
}

// Each dense source fills the storage its construct draws on.
func Test_Dense_Bounds(t *testing.T) {
	t.Parallel()
	test_argument_maximum(t)
}

// Test_Delimiter_Bounds packs the widest delimiters into bounded source.
func Test_Delimiter_Bounds(t *testing.T) {
	t.Parallel()
	test_delimiter_maximum(t)
}

// Test_Variable_Bounds packs the most variables into bounded source.
func Test_Variable_Bounds(t *testing.T) {
	t.Parallel()
	test_variable_maximum(t)
}

// Each dense declaration fills the storage its construct draws on.
func Test_Dense_Declaration_Bounds(t *testing.T) {
	t.Parallel()
	test_declaration_reference_maximum(t)
}

// Test_Dense_Construct_Declarations packs each construct beside the densest declarations.
func Test_Dense_Construct_Declarations(t *testing.T) {
	t.Parallel()
	test_dense_construct(t)
}

// Test_Declared_Construct_Bounds runs each construct beside its declared variables.
func Test_Declared_Construct_Bounds(t *testing.T) {
	t.Parallel()
	test_declared_constructs(t)
}

// Each expression stands beside the values and shapes one caller can hand it.
func Test_Expression_Bounds(t *testing.T) {
	t.Parallel()
	test_function_documents(t)
	test_position_arguments(t)
	test_builtin_expressions(t)
	test_execution_failures(t)
	test_nested_scopes(t)
}

// Each store fills to the bound one run can reach.
func Test_Storage_Bounds(t *testing.T) {
	t.Parallel()
	test_range_variable_maximum(t, TEST_COUNT_ZERO)
}

// One declaration run stands ahead of each named construct.
func Test_Declared_Storage_Bounds(t *testing.T) {
	t.Parallel()
	test_range_variable_maximum(t, TEST_COUNT_ONE)
}

// One registry at its bound answers every lookup a run makes.
func Test_Registry_Bounds(t *testing.T) {
	t.Parallel()
	test_function_registry_maximum(t)
}

// Test_Widest_Function_Name registers one name that fills its action.
func Test_Widest_Function_Name(t *testing.T) {
	t.Parallel()
	test_widest_function_name(t)
}

// Derived text fills to the bound one run can reach.
func Test_Generated_Bounds(t *testing.T) {
	t.Parallel()
	test_generated_maximum(t)
	test_generated_ceiling(t)
}

// One call fills the source with arguments, so argument storage reaches its bound.
func test_argument_maximum(t *testing.T) {
	t.Helper()
	// Each builtin refuses the widest argument list one source can carry.
	test_builtin_argument_maximum(t)
	test_number_maximum(t)
	// One byte names the callable, so the rest of the source carries arguments.
	source := []byte("<x")
	for len(source)+len(" .>") <= template.SOURCE_SIZE_MAXIMUM {
		source = append(source, " ."...)
	}
	source = append(source, '>')
	var workspace template.Syntax_Workspace
	program, _, status := template.Compile(template.Compile_Input{
		Source: source, Left: []byte("<"), Right: []byte(">"),
		Workspace: syntax_from(&workspace),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	functions := [...]template.Function{{Name: []byte("x"), Call: identity}}
	execute_functions_from(t, program, value_nil(), functions[:])
	test_argument_base_maximum(t)
}

// Each nested term opens above the widest argument list its caller staged.
func test_argument_base_maximum(t *testing.T) {
	t.Helper()
	functions := [...]template.Function{{Name: []byte("x"), Call: identity}}
	for _, tail := range [...]string{" (x .)>", " (.).a>", " ((.))>"} {
		source := []byte("<x")
		for len(source)+len(" .")+len(tail) <= template.SOURCE_SIZE_MAXIMUM {
			source = append(source, " ."...)
		}
		source = append(source, tail...)
		var workspace template.Syntax_Workspace
		program, _, status := template.Compile(template.Compile_Input{
			Source: source, Left: []byte("<"), Right: []byte(">"),
			Workspace: syntax_from(&workspace),
		})
		testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
		execute_functions_from(t, program, value_nil(), functions[:])
	}
}

// One number fills the action, so numeric text reaches its bound.
func test_number_maximum(t *testing.T) {
	t.Helper()
	source := make([]byte, template.SOURCE_SIZE_MAXIMUM)
	for index := range source {
		source[index] = '9'
	}
	source[TEST_COUNT_ZERO] = '<'
	source[len(source)-TEST_COUNT_ONE] = '>'
	var workspace template.Syntax_Workspace
	program, _, status := template.Compile(template.Compile_Input{
		Source: source, Left: []byte("<"), Right: []byte(">"),
		Workspace: syntax_from(&workspace),
	})
	if status != template.PARSE_STATUS_OK {
		return
	}
	execute_from(t, program, value_nil())
	test_number_shapes(t)
}

// Each numeric shape reaches the scan that reads its exponent and its sign.
func test_number_shapes(t *testing.T) {
	t.Helper()
	sources := [...]string{
		`<1e-1>`, `<-1e-1>`, `<1.5e-10>`, `<x 1e-1>`, `<x 1e-1 2e-2>`,
		`<-1.5e-3>`, `<1e+1>`, `<0x1p-2>`, `<1e-1i>`, `<x -1e-1i>`,
	}
	functions := [...]template.Function{{Name: []byte("x"), Call: identity}}
	var execution execution_storage
	for _, source := range sources {
		var workspace template.Syntax_Workspace
		program, _, status := template.Compile(template.Compile_Input{
			Source: []byte(source), Left: []byte("<"), Right: []byte(">"),
			Workspace: syntax_from(&workspace),
		})
		if status != template.PARSE_STATUS_OK {
			continue
		}
		execute_shared_from(t, &execution, program, value_nil(), functions[:])
	}
}

// Builtins bound their own argument lists, so each one sees the widest list.
func test_builtin_argument_maximum(t *testing.T) {
	t.Helper()
	names := [...]string{
		"not", "len", "index", "slice", "call", "and", "eq", "lt", "print", "printf",
		"html", "or", "ne", "le", "gt", "ge", "js", "urlquery", "println",
	}
	// A truthy value keeps short-circuit builtins reading every argument.
	truthy := [...]string{"and", "or", "not", "eq", "ne", "lt", "le", "gt", "ge"}
	var execution execution_storage
	for _, name := range names {
		source := append([]byte("<"), name...)
		for len(source)+len(" .>") <= template.SOURCE_SIZE_MAXIMUM {
			source = append(source, " ."...)
		}
		source = append(source, '>')
		var workspace template.Syntax_Workspace
		program, _, status := template.Compile(template.Compile_Input{
			Source: source, Left: []byte("<"), Right: []byte(">"),
			Workspace: syntax_from(&workspace),
		})
		testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
		execute_shared_from(t, &execution, program, value_nil(), nil)
		for _, short_circuit := range truthy {
			if short_circuit != name {
				continue
			}
			execute_shared_from(t, &execution, program, value_boolean(true), nil)
		}
		empty := append(append([]byte("<"), name...), '>')
		var short template.Syntax_Workspace
		program, _, status = template.Compile(template.Compile_Input{
			Source: empty, Left: []byte("<"), Right: []byte(">"),
			Workspace: syntax_from(&short),
		})
		if status != template.PARSE_STATUS_OK {
			continue
		}
		execute_shared_from(t, &execution, program, value_nil(), nil)
	}
}

// Each widest payload the source holds scans against one-byte delimiters.
func test_widest_payloads_scan(t *testing.T, marker []byte) {
	t.Helper()
	// One source of text alone spans every byte the parser scans.
	var text_only template.Syntax_Workspace
	template.Compile(template.Compile_Input{
		Source: marker, Left: []byte("<"), Right: []byte(">"),
		Workspace: syntax_from(&text_only),
	})
	// One number fills the content the widest action holds.
	digits := make([]byte, template.ACTION_CONTENT_SIZE_MAXIMUM)
	for index := range digits {
		digits[index] = '1'
	}
	// One quoted literal fills the action the source holds.
	text := make([]byte, template.LITERAL_DECODED_SIZE_MAXIMUM)
	for index := range text {
		text[index] = 'a'
	}
	literal_source := append(append(append([]byte{}, `<"`...), text...), `">`...)
	// One action behind that literal reads the decoded store its caller filled.
	tail_source := append(append(append([]byte{}, `<"`...),
		text[:len(text)-len("<.>")]...), `"><.>`...)
	var tail_state template.Syntax_Workspace
	tail_program, _, tail_status := template.Compile(template.Compile_Input{
		Source: tail_source, Left: []byte("<"), Right: []byte(">"),
		Workspace: syntax_from(&tail_state),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, tail_status)
	execute_from(t, tail_program, value_nil())
	var literal_state template.Syntax_Workspace
	literal_program, _, literal_status := template.Compile(template.Compile_Input{
		Source: literal_source, Left: []byte("<"), Right: []byte(">"),
		Workspace: syntax_from(&literal_state),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, literal_status)
	execute_from(t, literal_program, value_nil())
	// One fraction fills the action with digits a float still parses.
	fraction := append(append([]byte{}, `<0.`...), digits[:len(digits)-TEST_COUNT_THREE]...)
	fraction = append(fraction, '1', '>')
	var fraction_state template.Syntax_Workspace
	fraction_program, _, fraction_status := template.Compile(template.Compile_Input{
		Source: fraction, Left: []byte("<"), Right: []byte(">"),
		Workspace: syntax_from(&fraction_state),
	})
	if fraction_status == template.PARSE_STATUS_OK {
		execute_from(t, fraction_program, value_nil())
	}
	// That same fraction names the imaginary part of one complex literal.
	imaginary := append(append([]byte{}, fraction[:len(fraction)-TEST_COUNT_ONE]...), 'i')
	imaginary = append(imaginary, '>')
	var imaginary_state template.Syntax_Workspace
	imaginary_program, _, imaginary_status := template.Compile(template.Compile_Input{
		Source: imaginary, Left: []byte("<"), Right: []byte(">"),
		Workspace: syntax_from(&imaginary_state),
	})
	if imaginary_status == template.PARSE_STATUS_OK {
		execute_from(t, imaginary_program, value_nil())
	}
	number_source := append(append(append([]byte{}, '<'), digits...), '>')
	var digit_state template.Syntax_Workspace
	template.Compile(template.Compile_Input{
		Source: number_source, Left: []byte("<"), Right: []byte(">"),
		Workspace: syntax_from(&digit_state),
	})
}

// One delimiter pair fills the source, so delimiter storage reaches its bound.
func test_delimiter_maximum(t *testing.T) {
	t.Helper()
	marker := make([]byte, template.DELIMITER_SIZE_MAXIMUM)
	for index := range marker {
		marker[index] = 'q'
	}
	// One configuration validates delimiters no source can hold beside content.
	configuration_input_from(t, template.Configuration_Input{
		Left: marker, Right: marker,
	})
	// One source of that same extent scans against delimiters it cannot close.
	var widest template.Syntax_Workspace
	template.Compile(template.Compile_Input{
		Source: marker, Left: marker, Right: marker,
		Workspace: syntax_from(&widest),
	})
	// One source holds the longest marker beside one action of least size.
	longest := marker[:template.PARSED_DELIMITER_SIZE_MAXIMUM]
	// Each term kind reads its content between markers of that same extent.
	for _, content := range [...]byte{'.', '$', 'x'} {
		source := append(append(append([]byte{}, longest...), content), '>')
		var workspace template.Syntax_Workspace
		template.Compile(template.Compile_Input{
			Source: source, Left: longest, Right: []byte(">"),
			Workspace: syntax_from(&workspace),
		})
		source = append(append(append([]byte{}, '<'), content), longest...)
		var second template.Syntax_Workspace
		template.Compile(template.Compile_Input{
			Source: source, Left: []byte("<"), Right: longest,
			Workspace: syntax_from(&second),
		})
	}
	test_clause_delimiter_maximum(t, marker[:template.CLAUSE_DELIMITER_SIZE_MAXIMUM])
	test_widest_payloads_scan(t, marker)
}

// Each keyword clause reads its own content between markers of the widest extent.
func test_clause_delimiter_maximum(t *testing.T, widest []byte) {
	t.Helper()
	clauses := [...]string{
		`/**/`, `if .`, `else`, `end`, `with .`, `range .`,
		`define "a"`, `block "a"`, `template "a"`, `template "a" .`,
	}
	// A block opens a definition, so it keeps room for the end that definition owes.
	room := ""
	for _, clause := range clauses {
		if clause == `block "a"` {
			room = "xxxxx"
		}
		source := append(append(append([]byte{}, widest...), clause...), '>')
		source = append(source, room...)
		var workspace template.Syntax_Workspace
		template.Compile(template.Compile_Input{
			Source: source, Left: widest, Right: []byte(">"),
			Workspace: syntax_from(&workspace),
		})
		source = append(append(append([]byte{}, '<'), clause...), widest...)
		source = append(source, room...)
		room = ""
		var second template.Syntax_Workspace
		template.Compile(template.Compile_Input{
			Source: source, Left: []byte("<"), Right: widest,
			Workspace: syntax_from(&second),
		})
	}
}

// One recursive definition exhausts caller frame and variable storage.
func test_recursion_maximum(t *testing.T) {
	t.Helper()
	sources := [...]string{
		`<define "a"><template "a"><end><template "a">`,
		`<define "a"><$x:=.><template "a"><end><template "a">`,
		`<define "a"><range .><template "a"><end><end><template "a">`,
	}
	for _, source := range sources {
		var workspace template.Syntax_Workspace
		program, _, status := template.Compile(template.Compile_Input{
			Source: []byte(source), Left: []byte("<"), Right: []byte(">"),
			Workspace: syntax_from(&workspace),
		})
		testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
		items := [...]template.Value{value_nil()}
		execute_from(t, program, value_sequence(items[:]))
	}
}

// Recursion under nested branches stacks frames until the arena refuses another.
func test_frame_stack_maximum(t *testing.T) {
	t.Helper()
	depths := [...]int{TEST_COUNT_FOUR, TEST_COUNT_TEN, TEST_COUNT_TWENTY_TWO}
	var execution execution_storage
	for _, depth := range depths {
		source := []byte(`<define "a">`)
		for index := TEST_COUNT_ZERO; index < depth; index++ {
			source = append(source, `<if .>`...)
		}
		source = append(source, `<template "a">`...)
		for index := TEST_COUNT_ZERO; index < depth; index++ {
			source = append(source, `<end>`...)
		}
		source = append(source, `<end><template "a">`...)
		var workspace template.Syntax_Workspace
		program, _, status := template.Compile(template.Compile_Input{
			Source: source, Left: []byte("<"), Right: []byte(">"),
			Workspace: syntax_from(&workspace),
		})
		testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
		items := [...]template.Value{value_nil()}
		execute_shared_from(t, &execution, program, value_sequence(items[:]), nil)
	}
	test_deep_recursion_bodies(t)
}

// Each body runs at the deepest frame one recursion can stack.
func test_deep_recursion_bodies(t *testing.T) {
	t.Helper()
	bodies := [...]string{
		`<.>`, `<if .>x<end>`, `<range .><end>`, `<print . .>`, `<$x:=.><$x>`,
		`<(.).a>`, `<with .>x<end>`, `<"a">`, `<1>`, `</*c*/>`, `<.a>`,
		`<if .>x<else>y<end>`, `<range .><break><end>`, `<range .><continue><end>`,
		`<with .>x<else>y<end>`, `<index . 0>`, `<len .>`, `<print (identity .)>`,
		`<eq . .>`, `<not .>`, `<and . .>`, `<. | print>`, `<block "b" .>x<end>`,
		`<range 1><template "a" 1><end>`,
		`<range 1><range 1><range 1><range 1><template "a" 1><end><end><end><end>`,
		`<range 1><range 1><range 1><range 1><.><break><end><end><end><end><template "a" 1>`,
	}
	var execution execution_storage
	for _, body := range bodies {
		source := []byte(`<define "a">`)
		source = append(source, body...)
		source = append(source, `<template "a"><end><template "a">`...)
		var workspace template.Syntax_Workspace
		program, _, status := template.Compile(template.Compile_Input{
			Source: source, Left: []byte("<"), Right: []byte(">"),
			Workspace: syntax_from(&workspace),
		})
		testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
		items := [...]template.Value{value_nil()}
		execute_shared_from(t, &execution, program, value_sequence(items[:]), nil)
	}
}

// Nested branches fill the source, so execution stacks its deepest frames.
func test_nesting_maximum(t *testing.T) {
	t.Helper()
	open, close := `<if .>`, `<end>`
	depth := (template.SOURCE_SIZE_MAXIMUM - TEST_COUNT_ONE) / (len(open) + len(close))
	source := make([]byte, TEST_COUNT_ZERO, template.SOURCE_SIZE_MAXIMUM)
	for index := TEST_COUNT_ZERO; index < depth; index++ {
		source = append(source, open...)
	}
	source = append(source, 'x')
	for index := TEST_COUNT_ZERO; index < depth; index++ {
		source = append(source, close...)
	}
	var workspace template.Syntax_Workspace
	program, _, status := template.Compile(template.Compile_Input{
		Source: source, Left: []byte("<"), Right: []byte(">"),
		Workspace: syntax_from(&workspace),
	})
	if status != template.PARSE_STATUS_OK {
		return
	}
	execute_from(t, program, value_boolean(true))
}

// Each construct closes the densest source, so its own references reach the arena end.
// One refusal stands past the source start, so its position leaves the first byte.
func test_execution_failures(t *testing.T) {
	t.Helper()
	const FLOAT_ONE_BITS = uint64(big.FLOAT_64_EXPONENT_BIAS) <<
		big.FLOAT_64_EXPONENT_SHIFT
	complex_fields := [...]template.Field{{
		Name: template.Field_Name("a"),
		Value: template.Field_Value{Data: value_complex(template.Complex{
			Real:      template.Complex_Real(FLOAT_ONE_BITS),
			Imaginary: template.Complex_Imaginary(FLOAT_ONE_BITS),
		})},
	}}
	key_fields := [...]template.Field{{
		Key:   template.Field_Key{Data: value_sequence(nil)},
		Value: template.Field_Value{Data: value_nil()},
	}}
	dots := [...]template.Value{
		value_object(complex_fields[:]), value_map(key_fields[:]), value_nil(),
	}
	sources := [...]string{
		`aaaa{{.a}}`, `aaaa{{index . 5}}`, `aaaa{{.a.b}}`, `{{.a}}`,
		`aaaa{{$x := .}}{{$x.a}}`, `{{range .}}{{.}}{{end}}`,
		`{{$a := .}}{{define "t"}}{{range .}}{{break}}{{end}}{{end}}{{template "t" .}}`,
		`{{$a := .}}{{define "u"}}{{range .}}{{continue}}{{end}}{{end}}{{template "u" .}}`,
	}
	var execution execution_storage
	for _, source := range sources {
		program := compile_from(t, source)
		for _, dot := range dots {
			execute_shared_from(t, &execution, program, dot, nil)
		}
	}
}

// Each builtin stands beside the argument shapes one caller can hand it.
func test_builtin_expressions(t *testing.T) {
	t.Helper()
	sources := [...]string{
		`{{and . . .}}`, `{{or . . .}}`, `{{not .}}`, `{{not (not .)}}`,
		`{{eq . .}}`, `{{ne . .}}`, `{{lt . .}}`, `{{le . .}}`, `{{gt . .}}`,
		`{{ge . .}}`, `{{eq . 1 2 "a"}}`, `{{len .}}`, `{{len (listing .)}}`,
		`{{call identity .}}`, `{{print . (listing .) (mapping .)}}`,
		`{{println . .}}`, `{{printf "%v %d %s %q %x" . 1 "a" "b" 255}}`,
		`{{printf "%v" (listing .)}}`, `{{html .}}`, `{{js .}}`, `{{urlquery .}}`,
		`{{html "<&>"}}`, `{{js "'\"\\"}}`, `{{urlquery "a b&c"}}`,
		`{{index (listing .) 0}}`, `{{slice (listing .) 0 1}}`,
		`{{. | print | printf "%s"}}`, `{{identity . | identity | identity}}`,
		`{{if eq . .}}a{{else if ne . .}}b{{else}}c{{end}}`,
		`{{with (mapping .)}}{{.a}}{{end}}`, `{{$x := mapping .}}{{$x.a}}`,
		`{{print ((((.))))}}`, `{{(identity .)}}`, `{{len (slice (listing .) 0 1)}}`,
		// Each nested call delivers its value while its caller holds staged arguments.
		`{{print 1 (print 2 (print 3))}}`, `{{identity 1 (identity 2 (identity 3))}}`,
		`{{print . (identity (listing .)) .}}`, `{{eq 1 (len (listing .)) 2}}`,
		`{{print (mapping .).a (identity (mapping .).a)}}`,
		// One short circuit steps over a chain its argument list still holds.
		`{{and (mapping .).a .}}`, `{{or (mapping .).a .}}`,
		`{{and . (mapping .).a}}`, `{{or . (mapping .).a}}`,
		`{{and (identity .) (mapping .).a .}}`, `{{or (mapping .).a (identity .)}}`,
		`{{if and (mapping .).a .}}x{{end}}`, `{{and (listing .) (mapping .).a}}`,
		`{{(and (mapping .) (mapping .)).a}}`, `{{(or (mapping .) .).a}}`,
		`{{(and . (mapping .)).a}}`, `{{(or . (mapping .)).a}}`,
		`{{print (and (mapping .) (mapping .)).a .}}`,
	}
	functions := [...]template.Function{
		{Name: []byte("identity"), Call: identity},
		{Name: []byte("listing"), Call: listing},
		{Name: []byte("mapping"), Call: mapping},
	}
	scalars := matrix_scalars()
	dots := [...]template.Value{
		value_text(template.Text("ab")), value_sequence(scalars[:TEST_COUNT_TWO]),
		value_nil(),
	}
	var execution execution_storage
	for _, source := range sources {
		program := compile_from(t, source)
		for _, dot := range dots {
			execute_shared_from(t, &execution, program, dot, functions[:])
		}
	}
}

// One registry at its bound answers the lookup each call makes.
func test_function_registry_maximum(t *testing.T) {
	t.Helper()
	functions := make([]template.Function, template.FUNCTION_COUNT_MAXIMUM)
	name := []byte{TEST_NAME_FIRST_BYTE}
	for index := range functions {
		functions[index] = template.Function{
			Name: append([]byte("f"), name...), Call: identity,
		}
		name = next_name(name)
	}
	functions[TEST_COUNT_ZERO] = template.Function{
		Name: []byte("identity"), Call: identity,
	}
	// One run validates the registry once, and every lookup reads it.
	program := compile_from(t, `{{if .}}{{identity .}}{{print identity}}{{end}}`+
		`{{define "t"}}{{identity .}}{{end}}{{template "t" .}}{{call identity 1}}`)
	var execution execution_storage
	execute_shared_from(t, &execution, program, value_text(nil), functions)
}

// One registered name fills the action content the source holds.
func test_widest_function_name(t *testing.T) {
	t.Helper()
	widest := make([]byte, template.ACTION_CONTENT_SIZE_MAXIMUM)
	for index := range widest {
		widest[index] = 'w'
	}
	source := append(append(append([]byte{}, '<'), widest...), '>')
	var workspace template.Syntax_Workspace
	program, _, status := template.Compile(template.Compile_Input{
		Source: source, Left: []byte("<"), Right: []byte(">"),
		Workspace: syntax_from(&workspace),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	registry := [...]template.Function{{Name: widest, Call: identity}}
	execute_functions_from(t, program, value_nil(), registry[:])
}

// One position argument of each kind reaches the collection bounds it names.
func test_position_arguments(t *testing.T) {
	t.Helper()
	sources := [...]string{
		`{{index . 0}}`, `{{slice . 0 1}}`, `{{index . 1}}`,
		`{{index . 1i}}`, `{{index . .}}`, `{{index . "a"}}`, `{{index . true}}`,
		`{{index . -1}}`, `{{index . nil}}`, `{{index . 1.5}}`,
		`{{slice . 1i}}`, `{{slice . .}}`, `{{slice . "a"}}`,
		`{{slice . 0 1i}}`, `{{slice . -1}}`, `{{slice . nil}}`, `{{slice . 1.5}}`,
		`{{slice . true}}`, `{{index . (mapping .)}}`, `{{index . (listing .)}}`,
		`{{index . identity}}`, `{{slice . (mapping .)}}`, `{{slice . identity}}`,
	}
	scalars := matrix_scalars()
	fields := [...]template.Field{{
		Key:   template.Field_Key{Data: value_text(template.Text("a"))},
		Value: template.Field_Value{Data: value_text(template.Text("b"))},
	}}
	// One payload at its bound stands where a position argument is read.
	widest := make(template.Values, template.VALUE_COUNT_MAXIMUM)
	text := make(template.Text, template.OUTPUT_SIZE_MAXIMUM)
	nested := [...]template.Field{{
		Key:   template.Field_Key{Data: value_map(fields[:])},
		Value: template.Field_Value{Data: value_nil()},
	}}
	dots := [...]template.Value{
		value_sequence(scalars[:TEST_COUNT_TWO]), value_map(fields[:]),
		value_text(template.Text("ab")), value_sequence(widest), value_text(text),
		value_sequence(scalars), value_map(nested[:]),
	}
	functions := [...]template.Function{
		{Name: []byte("identity"), Call: identity},
		{Name: []byte("listing"), Call: listing},
		{Name: []byte("mapping"), Call: mapping},
	}
	var execution execution_storage
	for _, source := range sources {
		program := compile_from(t, source)
		for _, dot := range dots {
			execute_shared_from(t, &execution, program, dot, functions[:])
		}
	}
	test_widest_payloads(t)
	test_generated_offsets(t)
}

// Each derived-text step follows one that already wrote a byte or two.
func test_generated_offsets(t *testing.T) {
	t.Helper()
	sources := [...]string{
		`{{print (printf "%s" "a") .}}`, `{{print (printf "%s" "ab") .}}`,
		`{{js (printf "%s" "a")}}`, `{{js (printf "%s" "ab")}}`,
		`{{html (printf "%s" "a")}}`, `{{html (printf "%s" "ab")}}`,
		`{{urlquery (printf "%s" "a")}}`, `{{urlquery (printf "%s" "ab")}}`,
		`{{printf "%s" (printf "%s" "a")}}`, `{{printf "%q" (printf "%s" "a")}}`,
		`{{printf "%d" (len (printf "%s" "a"))}}`,
		`{{print (printf "%s" "a") (printf "%s" "b")}}`,
		`{{printf "%v %v" (printf "%s" "a") .}}`,
	}
	dots := [...]template.Value{value_text(template.Text("z")), value_nil()}
	var execution execution_storage
	for _, source := range sources {
		program := compile_from(t, source)
		for _, dot := range dots {
			execute_shared_from(t, &execution, program, dot, nil)
		}
	}
}

// One payload at its bound stands where each evaluation stages its value.
func test_widest_payloads(t *testing.T) {
	t.Helper()
	items := make(template.Values, template.VALUE_COUNT_MAXIMUM)
	entries := make(template.Fields, template.RANGE_MAP_ENTRY_COUNT_MAXIMUM)
	for index := range entries {
		entries[index] = template.Field{
			Key: template.Field_Key{Data: value_integer(template.Integer(index))},
		}
	}
	text := make(template.Text, template.OUTPUT_SIZE_MAXIMUM)
	dots := [...]template.Value{
		value_sequence(items), value_map(entries), value_text(text),
		value_map(nil), value_sequence(nil), value_object(nil),
	}
	sources := [...]string{
		`{{print .}}`, `{{identity .}}`, `{{len .}}`, `{{.}}`, `{{print . .}}`,
		`{{if .}}{{identity .}}{{end}}`, `{{index . 0}}`,
		`{{print (identity .)}}`, `{{identity (identity .)}}`,
		`{{len (identity .)}}`, `{{index (identity .) 0}}`,
		`{{with (identity .)}}{{.}}{{end}}`, `{{range (identity .)}}{{end}}`,
		`{{range .}}{{end}}`, `{{range $k,$v := .}}{{end}}`, `{{with .}}{{end}}`,
	}
	functions := [...]template.Function{{Name: []byte("identity"), Call: identity}}
	var execution execution_storage
	for _, source := range sources {
		program := compile_from(t, source)
		for _, dot := range dots {
			execute_shared_from(t, &execution, program, dot, functions[:])
		}
	}
}

// One call stands inside a definition, so lookup runs against a named document.
func test_function_documents(t *testing.T) {
	t.Helper()
	sources := [...]string{
		`{{define "t"}}{{identity .}}{{end}}{{template "t" .}}`,
		`{{define "t"}}{{identity . | identity}}{{end}}{{block "b" .}}{{identity .}}{{end}}`,
		`{{block "b" .}}{{identity .}}{{end}}{{template "b" .}}`,
		`{{define "t"}}{{print identity}}{{end}}{{template "t" .}}`,
		`{{define "t"}}{{identity}}{{end}}{{template "t" .}}`,
		// One chain stands beside an argument already staged for its command.
		`{{print (mapping .).a (mapping .).a}}`,
		`{{identity (mapping .).a (listing .)}}`,
		`{{print (identity .) (mapping .).a (listing .)}}`,
		`{{print (mapping .).a | identity}}`,
	}
	functions := [...]template.Function{
		{Name: []byte("identity"), Call: identity},
		{Name: []byte("listing"), Call: listing},
		{Name: []byte("mapping"), Call: mapping},
	}
	var execution execution_storage
	for _, source := range sources {
		program := compile_from(t, source)
		execute_shared_from(t, &execution, program, value_text(nil), functions[:])
	}
	// One-byte delimiters open each call at the first source byte.
	narrow := [...]string{
		`<identity .>`, `<identity>`, `<print identity>`, `<(identity .)>`,
		`<identity . | identity>`, `<$x:=identity .><$x>`,
	}
	for _, source := range narrow {
		var workspace template.Syntax_Workspace
		program, _, status := template.Compile(template.Compile_Input{
			Source: []byte(source), Left: []byte("<"), Right: []byte(">"),
			Workspace: syntax_from(&workspace),
		})
		testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
		execute_shared_from(t, &execution, program, value_text(nil), functions[:])
	}
}

// Gives the construct list one declaration-filled run covers.
func tails_of(part int) (tails []string) {
	first := []string{
		`<range .><end>`, `<range $i,$e:=.><end>`, `<with .><end>`, `<if .>x<end>`,
		`<define "t"><end>`, `<block "b" .>x<end>`, `</*c*/>`,
	}
	second := []string{
		`<template "t" .>`, `<.a>`, `<$:=.>`, `<if .>x<else>y<end>`,
		`<define ""><range .>x<end><end><template "" .>`,
		`<define ""><if .>x<end><end><template "" .>`,
		`<define ""><$a:=.><$a><end><template "" .>`,
	}
	if part == TEST_COUNT_ZERO {
		return first
	}
	return second
}

// Declarations fill the table ahead of one range, so its marks reach their bound.
func test_range_variable_maximum(t *testing.T, part int) {
	t.Helper()
	tails := tails_of(part)
	scalars := matrix_scalars()
	var execution execution_storage
	for _, tail := range tails {
		source := make([]byte, TEST_COUNT_ZERO, template.SOURCE_SIZE_MAXIMUM)
		declaration := `<$:=.>`
		for len(source)+len(declaration)+len(tail) <= template.SOURCE_SIZE_MAXIMUM {
			source = append(source, declaration...)
		}
		source = append(source, tail...)
		var workspace template.Syntax_Workspace
		program, _, status := template.Compile(template.Compile_Input{
			Source: source, Left: []byte("<"), Right: []byte(">"),
			Workspace: syntax_from(&workspace),
		})
		if status != template.PARSE_STATUS_OK {
			continue
		}
		execute_shared_from(
			t, &execution, program, value_sequence(scalars[:TEST_COUNT_TWO]), nil)
	}
}

// Scopes nest, so an inner frame opens above the bindings its parent holds.
func test_nested_scopes(t *testing.T) {
	t.Helper()
	sources := [...]string{
		`{{$a := .}}{{range $i, $e := $a}}{{$b := $e}}{{$b}}{{$i}}{{end}}`,
		`{{with $a := .}}{{range $a}}{{.}}{{end}}{{end}}`,
		`{{$a := .}}{{with $a}}{{$b := .}}{{range $b}}{{.}}{{end}}{{end}}`,
		`{{range .}}{{$x := .}}{{with $x}}{{.}}{{end}}{{end}}`,
		`{{$a := .}}{{$b := .}}{{range $b}}{{$c := .}}{{range $c}}{{.}}{{end}}{{end}}`,
		`{{$a := .}}{{with $b := .}}{{range $b}}{{.}}{{end}}{{end}}`,
		`{{$a := .}}{{with $b := .}}{{with $c := $b}}{{range $i, $e := $c}}` +
			`{{$i}}{{$e}}{{end}}{{end}}{{end}}`,
		`{{$a := .}}{{range $b := .}}{{with $c := $b}}{{range $d := $c}}` +
			`{{$d}}{{end}}{{end}}{{end}}`,
		// One invocation opens its own scope above the bindings its caller holds.
		`{{$a := .}}{{define "t"}}{{range .}}{{.}}{{end}}{{end}}{{template "t" .}}`,
		`{{$a := .}}{{$b := .}}{{define "t"}}{{with $c := .}}{{range $c}}{{.}}` +
			`{{end}}{{end}}{{end}}{{template "t" .}}`,
		`{{$a := .}}{{define "t"}}{{$d := .}}{{range $i, $e := $d}}{{$i}}{{$e}}` +
			`{{end}}{{end}}{{template "t" .}}`,
	}
	scalars := matrix_scalars()
	inner := value_sequence(scalars[:TEST_COUNT_TWO])
	nested := [...]template.Value{inner, inner}
	var execution execution_storage
	for _, source := range sources {
		program := compile_from(t, source)
		_, status := execute_shared_from(
			t, &execution, program, value_sequence(nested[:]), nil)
		testify.Equal_Values(t, template.STATUS_OK, status)
	}
}

// One packed source fills the node arena, so node storage reaches its bound.
func test_node_maximum(t *testing.T) {
	t.Helper()
	source := make([]byte, TEST_COUNT_ZERO, template.SOURCE_SIZE_MAXIMUM)
	for len(source)+len("<.>") <= template.SOURCE_SIZE_MAXIMUM {
		source = append(source, "<.>"...)
	}
	// Leftover bytes after the packed actions take the trailing text record.
	for len(source) < template.SOURCE_SIZE_MAXIMUM {
		source = append(source, 'x')
	}
	var workspace template.Syntax_Workspace
	program, _, status := template.Compile(template.Compile_Input{
		Source: source, Left: []byte("<"), Right: []byte(">"),
		Workspace: syntax_from(&workspace),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	var execution execution_storage
	execute_shared_from(t, &execution, program, value_text(nil), nil)
	test_node_maximum_tails(t)
}

// One packed source ends on an action, so the last record comes from a pipeline step.
func test_node_maximum_tails(t *testing.T) {
	t.Helper()
	tails := [...]string{
		`<.a>`, `<(.)>`, `<.a.b>`, `<x . .>`, `<. | .>`, `</*a*/>`, `<.a .b>`,
		`<(.).a>`, `<$x:=.>`, `<.|.|.>`,
	}
	var execution execution_storage
	for _, tail := range tails {
		source := make([]byte, TEST_COUNT_ZERO, template.SOURCE_SIZE_MAXIMUM)
		for len(source)+len(tail)+len("<.>") <= template.SOURCE_SIZE_MAXIMUM {
			source = append(source, "<.>"...)
		}
		for len(source)+len(tail) < template.SOURCE_SIZE_MAXIMUM {
			source = append(source, 'x')
		}
		source = append(source, tail...)
		var workspace template.Syntax_Workspace
		program, _, status := template.Compile(template.Compile_Input{
			Source: source, Left: []byte("<"), Right: []byte(">"),
			Workspace: syntax_from(&workspace),
		})
		if status != template.PARSE_STATUS_OK {
			continue
		}
		execute_shared_from(t, &execution, program, value_text(nil), nil)
	}
}

func test_dense_construct(t *testing.T) {
	t.Helper()
	// Each dense parse costs one full arena, so the list holds one construct per shape.
	constructs := [...]string{
		`<if .>x<end>`, `<range $i,$e:=.>x<end>`, `<$a:=.><$a>`,
		`<define "t">x<end><template "t" .>`, `<x . .>`, `<(.).a>`,
		`<if .>x<else>y<end>`, `<if .>x<else if .>y<end>`, `<with .>x<end>`,
		`<range .>x<break><end>`, `<range .>x<continue><end>`, `</*c*/>`,
		`<block "b" .>x<end>`,
	}
	for _, construct := range constructs {
		source := make([]byte, TEST_COUNT_ZERO, template.SOURCE_SIZE_MAXIMUM)
		for len(source)+len(construct)+len("<.>") <= template.SOURCE_SIZE_MAXIMUM {
			source = append(source, "<.>"...)
		}
		source = append(source, construct...)
		var workspace template.Syntax_Workspace
		template.Compile(template.Compile_Input{
			Source: source, Left: []byte("<"), Right: []byte(">"),
			Workspace: syntax_from(&workspace),
		})
	}
	test_node_maximum(t)
}

// Escapes follow one full print, so derived text reaches its bound.
func test_generated_maximum(t *testing.T) {
	t.Helper()
	text := make([]byte, template.OUTPUT_SIZE_MAXIMUM)
	for index := range text {
		text[index] = 'a'
	}
	sources := [...]string{
		`{{html . .}}`, `{{js . .}}`, `{{urlquery . .}}`, `{{print . .}}`,
		`{{printf "%q" (print . .)}}`, `{{printf "%s%s" . .}}`, `{{println . .}}`,
		`{{html (print . .)}}`, `{{js (print . .)}}`, `{{urlquery (print . .)}}`,
		`{{printf "%v%v" . .}}`, `{{len (print . .)}}`,
		`{{print . . | html}}`, `{{print . . | js}}`, `{{print . . | urlquery}}`,
		`{{print . . | printf "%q"}}`, `{{print . . | print}}`, `{{print . .}}{{.}}`,
		`{{print . .}}{{html .}}`, `{{print . . | len}}`, `{{print . . | printf "%d"}}`,
		`{{len (print . .)}}{{.}}`, `{{len (print . .)}}{{html .}}`,
		`{{len (print . .)}}{{printf "%q" .}}`, `{{len (print . .)}}{{if .}}x{{end}}`,
		`{{len (print . .)}}{{js .}}`, `{{len (print . .)}}{{urlquery .}}`,
		`{{len (print . .)}}{{template "t" .}}{{define "t"}}{{.}}{{end}}`,
		`{{len (print . .)}}{{$a := .}}{{$a}}`, `{{len (print . .)}}{{with .}}{{.}}{{end}}`,
	}
	var execution execution_storage
	for _, source := range sources {
		execute_shared_from(t, &execution, compile_from(t, source), value_text(text), nil)
	}
}

// Small shapes carry the first counts each stage can hold.
// Gives every payload shape the smallest collections hold.
func small_shape_values() (values []template.Value) {
	// One and two members carry the first counts a payload can hold.
	single := [...]template.Field{{Name: []byte("a"),
		Value: template.Field_Value{Data: value_boolean(true)}}}
	pair := [...]template.Field{
		{Name: []byte("a"), Value: template.Field_Value{Data: value_boolean(true)}},
		{Name: []byte("b"), Value: template.Field_Value{Data: value_boolean(false)}},
	}
	items := [...]template.Value{value_boolean(true), value_boolean(false)}
	// One and two bytes carry the first extents a text payload can hold.
	texts := [...]template.Value{
		value_text([]byte("a")), value_text([]byte("ab")),
	}
	text_fields := [...]template.Field{
		{Name: []byte("a"), Value: template.Field_Value{Data: texts[TEST_COUNT_ZERO]}},
		{Name: []byte("b"), Value: template.Field_Value{Data: texts[TEST_COUNT_ONE]}},
	}
	// One widest bit pattern rides inside a member, a key, and an item.
	widest := value_complex(template.Complex{
		Real:      template.Complex_Real(bits.WORD_64_MAXIMUM),
		Imaginary: template.Complex_Imaginary(bits.WORD_64_MAXIMUM),
	})
	widest_fields := [...]template.Field{
		{Name: []byte("a"), Value: template.Field_Value{Data: widest}},
		{
			Key:   template.Field_Key{Data: value_function(identity)},
			Value: template.Field_Value{Data: widest},
		},
	}
	// The two smallest bit patterns ride the same way.
	subnormals := [...]template.Value{
		value_complex(template.Complex{
			Real: TEST_COUNT_ONE, Imaginary: TEST_COUNT_ONE,
		}),
		value_complex(template.Complex{
			Real: TEST_COUNT_TWO, Imaginary: TEST_COUNT_TWO,
		}),
	}
	subnormal_fields := [...]template.Field{
		{Name: []byte("a"), Value: template.Field_Value{Data: subnormals[TEST_COUNT_ZERO]}},
		{Name: []byte("b"), Value: template.Field_Value{Data: subnormals[TEST_COUNT_ONE]}},
	}
	widest_items := [...]template.Value{widest}
	nests := [...]template.Value{
		value_sequence(items[:TEST_COUNT_ONE]), value_sequence(items[:]),
		value_object(single[:]), value_map(single[:]),
	}
	// One field holds a pair, so a field access lands on a two-member payload.
	nested := [...]template.Field{{Name: []byte("a"),
		Value: template.Field_Value{Data: value_object(pair[:])}}}
	payloads := [...]template.Value{
		value_nil(), value_text([]byte("ab")), value_boolean(true),
		value_sequence(items[:TEST_COUNT_ONE]), value_sequence(items[:]),
		value_object(single[:]), value_object(pair[:]),
		value_map(single[:]), value_map(pair[:]), value_object(nested[:]),
		value_complex(template.Complex{
			Real: TEST_COUNT_ONE, Imaginary: TEST_COUNT_ONE,
		}),
		value_complex(template.Complex{
			Real: TEST_COUNT_TWO, Imaginary: TEST_COUNT_TWO,
		}),
		value_sequence(texts[:]), value_object(text_fields[:]),
		value_map(text_fields[:]), value_sequence(nests[:]),
		value_object(widest_fields[:TEST_COUNT_ONE]),
		value_map(widest_fields[TEST_COUNT_ONE:]), value_sequence(widest_items[:]),
		value_object(subnormal_fields[:]), value_sequence(subnormals[:]),
	}
	return payloads[:]
}

func test_small_shapes(t *testing.T) {
	t.Helper()
	sources := [...]string{
		`{{if .}}{{if .}}x{{end}}{{end}}`, `{{$a := .}}{{$b := .}}{{$a}}{{$b}}`,
		`{{range .}}{{range .}}x{{end}}{{end}}`, `{{with .}}{{with .}}x{{end}}{{end}}`,
		`{{define "a"}}x{{end}}{{define "b"}}y{{end}}{{template "a"}}{{template "b"}}`,
		`{{print . .}}`, `{{identity .}}`, `{{listing}}`, `{{index listing 0}}`,
		`{{if .}}{{$a := .}}{{$a}}{{end}}`, `{{range $k, $v := .}}{{$k}}{{$v}}{{end}}`,
		`{{with $a := .}}{{$a}}{{end}}`, `x`, `xy`, `{{.}}`, `{{.a}}`,
		// An absent definition leaves the lookup result empty for its caller.
		`{{template "absent"}}`, `{{template "absent" .}}`,
		`{{define "a"}}x{{end}}{{template "absent" .}}`,
		`{{define "a"}}{{template "absent"}}{{end}}{{template "a" .}}`,
		// Outer declarations put a ranged binding above the first scope slot.
		`{{$a := .}}{{range $k, $v := .}}{{$k}}{{$v}}{{end}}`,
		`{{$a := .}}{{$b := .}}{{range $k, $v := .}}{{$v}}{{end}}`,
		`{{$a := .}}{{range $i := .}}{{$i}}{{end}}`,
		// A bound body holds the branch inside it above the first scope slot.
		`{{with $a := .}}{{if .}}x{{end}}{{end}}`,
		`{{range $k, $v := .}}{{if .}}x{{end}}{{end}}`,
		`{{with $a := .}}{{with $b := .}}{{if .}}x{{end}}{{end}}{{end}}`,
		// Declarations before a branch put its body scope above the first slot.
		`{{$a := .}}{{if .}}x{{end}}`, `{{$a := .}}{{$b := .}}{{if .}}x{{end}}`,
		`{{$a := .}}{{range .}}x{{end}}`, `{{$a := .}}{{$b := .}}{{with .}}x{{end}}`,
		// A short circuit keeps a pending chain and stage on the frame it holds.
		`{{and (identity .).a .}}`, `{{or (identity .).a .}}`,
		`{{and (identity .) (identity .)}}`, `{{or .a.b .}}`,
		// A nested call delivers its child while the pipeline still owns declarations.
		`{{$a := .}}{{$a = identity (identity .)}}{{$a}}`,
		`{{range $k, $v := identity (identity .)}}{{$k}}{{$v}}{{end}}`,
		`{{$a := identity (identity .)}}{{$a}}`,
		`{{with $a := identity (identity .)}}{{$a}}{{end}}`,
		// One index lands on each payload the smallest collections hold.
		`{{index . 0}}`, `{{index . 1}}`, `{{index . "a"}}`, `{{len (index . 0)}}`,
	}
	values := small_shape_values()
	functions := [...]template.Function{
		{Name: []byte("identity"), Call: identity},
		{Name: []byte("listing"), Call: listing},
	}
	var execution execution_storage
	test_small_shape_runs(t, &execution, sources[:], values, functions[:])
	pair := [...]template.Field{
		{Name: []byte("a"), Value: template.Field_Value{Data: value_boolean(true)}},
		{Name: []byte("b"), Value: template.Field_Value{Data: value_boolean(false)}},
	}
	test_definition_counts(t, &execution, value_object(pair[:]), functions[:])
}

// Each small source meets every payload, and every other one meets the shortest list.
func test_small_shape_runs(
	t *testing.T, execution *execution_storage, sources []string,
	values []template.Value, functions template.Functions,
) {
	t.Helper()
	for offset, source := range sources {
		program := compile_from(t, source)
		for index := range values {
			// One registered function witnesses the shortest list a run can hold.
			if offset%TEST_COUNT_TWO == TEST_COUNT_ZERO {
				execute_shared_from(t, execution, program, values[index],
					functions[:TEST_COUNT_ONE])
			}
			execute_shared_from(t, execution, program, values[index], functions)
		}
	}
}

// One and two definitions stand beside the branches and calls that read their count.
func test_definition_counts(
	t *testing.T, execution *execution_storage, value template.Value,
	functions template.Functions,
) {
	t.Helper()
	sources := [...]string{
		`{{define "t"}}x{{end}}{{if .}}y{{end}}`,
		`{{define "t"}}x{{end}}{{define "u"}}y{{end}}{{range .}}z{{end}}`,
		`{{define "t"}}x{{end}}{{identity (identity .).a}}`,
		`{{define "t"}}x{{end}}{{define "u"}}y{{end}}{{identity (identity .).a}}`,
		`{{define "t"}}{{.a.b}}{{end}}{{template "t" .}}`,
		`{{define "t"}}{{identity (identity .).a}}{{end}}{{template "t" .}}`,
	}
	for _, source := range sources {
		execute_shared_from(t, execution, compile_from(t, source), value, functions)
	}
}

// Derived text near its bound meets each later action, so staging sees its ceiling.
func test_generated_ceiling(t *testing.T) {
	t.Helper()
	text := make([]byte, template.OUTPUT_SIZE_MAXIMUM)
	for index := range text {
		text[index] = 'a'
	}
	// Two staged prints fill derived storage, so the third action starts at the bound.
	filled := `{{len (print .)}}{{len (print .)}}`
	tails := [...]string{
		`{{len .}}`, `{{.}}`, `{{if .}}x{{end}}`, `{{html "a"}}`, `{{js "a"}}`,
		`{{urlquery "a"}}`, `{{printf "%q" "a"}}`, `{{printf "%d" 1}}`,
		`{{printf "%v" .}}`, `{{template "t" .}}{{define "t"}}x{{end}}`,
		`{{$a := .}}{{$a}}`, `{{with .}}x{{end}}`, `{{range .}}x{{end}}`,
		`{{print "a" "b"}}`, `{{len (slice . 0 1)}}`,
	}
	var execution execution_storage
	for _, tail := range tails {
		execute_shared_from(t, &execution, compile_from(t, filled+tail), value_text(text),
			nil)
	}
}

// The first counts a run can hold need their own sources, so each stage sees one and two.
func test_small_counts(t *testing.T) {
	t.Helper()
	sources := [...]string{
		`{{print "a"}}`, `{{print "ab"}}`, `{{print "a"}}{{.}}`, `{{print "ab"}}{{.}}`,
		`{{print "a"}}{{if .}}x{{end}}`, `{{print "ab"}}{{$a := .}}{{$a}}`,
		`{{$a := .}}{{$a}}`, `x{{$a := .}}{{$a}}`, `{{$a := .}}{{$b := .}}{{$a}}{{$b}}`,
		`{{define "a"}}x{{end}}{{define "b"}}y{{end}}{{template "a"}}`,
		`{{.}}{{define "a"}}x{{end}}`, `x{{define "a"}}y{{end}}`,
		`{{print "a"}}{{range .}}x{{end}}`, `{{print "ab"}}{{with .}}x{{end}}`,
	}
	var execution execution_storage
	for _, source := range sources {
		execute_shared_from(t, &execution, compile_from(t, source), value_boolean(true),
			nil)
	}
	// Node policies below the arena meet each parse stage before allocation refuses.
	for _, source := range [...]string{`<(a).b|c>`, `<$a:=.>`, `<if .><end>`, `<x y z>`, `<template "t" .>`} {
		for limit := template.Node_Limit(
			template.NODE_COUNT_INCREMENT); limit <= TEST_LIMIT_SMALL; limit++ {
			workspace := template.Syntax_Workspace{
				Node_Limit: template.Node_Limit_Storage{
					Value: template.Node_Limit_Stored(limit),
				},
			}
			template.Compile(template.Compile_Input{
				Source: []byte(source), Left: []byte("<"), Right: []byte(">"),
				Workspace: syntax_from(&workspace),
			})
		}
	}
}

// Each construct also stands first, second, and doubled, so early references stay covered.
func test_construct_positions(t *testing.T) {
	t.Helper()
	constructs := [...]string{
		`<.>`, `<x>`, `<if .>a<end>`, `<if .>a<else>b<end>`, `<range .>a<end>`,
		`<range $i,$e:=.>a<end>`, `<with .>a<end>`, `<$a:=.>`, `<$a:=.><$a>`,
		`<define "t">a<end>`, `<define "t">a<end><template "t" .>`, `<block "b" .>a<end>`,
		`</*c*/>`, `<(.)>`, `<(.).a>`, `<.a>`, `<.a.b>`, `<x .>`, `<x . .>`,
		`<. | x>`, `<x | x>`, `<range .><break><end>`, `<range .><continue><end>`,
		`<1>`, `<"s">`, `<nil>`, `<true>`, `<- . ->`,
		`<x (x .)>`, `<x (x (x .))>`, `<x (x .) (x .)>`, `<(x .).a>`, `<(x .).a.b>`,
		`<range (x .)>a<end>`, `<with (x .)>a<end>`, `<if (x .)>a<end>`,
		`<$a:=(x .)><$a>`, `<x . | x>`, `<x (x .) | x>`, `<.a | x>`,
		`<range .><if .>a<end><end>`, `<range .><$a:=.><$a><end>`,
		`<define "t"><x .><end><template "t" (x .)>`,
		`<x (x (x (x .)))>`, `<.a.b.c>`, `<(.a).b>`, `<$a:=.><$a=.><$a>`,
		`<$.a>`, `<$.a.b>`, `<x $.a>`,
		`<range .><range .>a<end><end>`, `<with .><with .>a<end><end>`,
		`<if .><if .>a<else>b<end><end>`, `<range .><else>b<end>`,
	}
	functions := [...]template.Function{{Name: []byte("x"), Call: identity}}
	var execution execution_storage
	for _, construct := range constructs {
		for _, source := range [...]string{construct, "x" + construct,
			construct + construct} {
			var workspace template.Syntax_Workspace
			program, _, status := template.Compile(template.Compile_Input{
				Source: []byte(source), Left: []byte("<"), Right: []byte(">"),
				Workspace: syntax_from(&workspace),
			})
			if status != template.PARSE_STATUS_OK {
				continue
			}
			execute_shared_from(t, &execution, program, value_boolean(true),
				functions[:])
		}
	}
}

// One dense source ends in a declaration, so its reference reaches the arena end.
func test_declaration_reference_maximum(t *testing.T) {
	t.Helper()
	for _, declaration := range [...]string{"<$a:=.>", "<range $a,$b:=.><end>"} {
		source := make([]byte, TEST_COUNT_ZERO, template.SOURCE_SIZE_MAXIMUM)
		for len(source)+len(declaration)+len("<.>") <= template.SOURCE_SIZE_MAXIMUM {
			source = append(source, "<.>"...)
		}
		source = append(source, declaration...)
		var workspace template.Syntax_Workspace
		program, _, status := template.Compile(template.Compile_Input{
			Source: source, Left: []byte("<"), Right: []byte(">"),
			Workspace: syntax_from(&workspace),
		})
		if status != template.PARSE_STATUS_OK {
			continue
		}
		execute_from(t, program, value_text(nil))
	}
}

// Declarations fill the source, so execution binds its last variable slot.
func test_variable_maximum(t *testing.T) {
	t.Helper()
	declaration := []byte(`<$aa:=.>`)
	prefix_count := len(`<$`)
	source := make([]byte, TEST_COUNT_ZERO, template.SOURCE_SIZE_MAXIMUM)
	index := TEST_COUNT_ZERO
	for len(source)+len(declaration) <= template.SOURCE_SIZE_MAXIMUM {
		declaration[prefix_count] = byte('a' + index/TEST_ALPHABET_SIZE)
		declaration[prefix_count+TEST_COUNT_ONE] = byte('a' + index%TEST_ALPHABET_SIZE)
		source = append(source, declaration...)
		index++
	}
	var workspace template.Syntax_Workspace
	program, _, status := template.Compile(template.Compile_Input{
		Source: source, Left: []byte("<"), Right: []byte(">"),
		Workspace: syntax_from(&workspace),
	})
	if status != template.PARSE_STATUS_OK {
		return
	}
	execute_from(t, program, value_nil())
	test_variable_maximum_tail(t)
}

// Anonymous declarations fill the source, so a trailing action meets the last slot.
func test_variable_maximum_tail(t *testing.T) {
	t.Helper()
	tails := [...]string{`<x>`, `<.>`}
	functions := [...]template.Function{{Name: []byte("x"), Call: identity}}
	var execution execution_storage
	for _, tail := range tails {
		source := make([]byte, TEST_COUNT_ZERO, template.SOURCE_SIZE_MAXIMUM)
		for len(source)+len(`<$:=.>`)+len(tail) <= template.SOURCE_SIZE_MAXIMUM {
			source = append(source, `<$:=.>`...)
		}
		source = append(source, tail...)
		var workspace template.Syntax_Workspace
		program, _, status := template.Compile(template.Compile_Input{
			Source: source, Left: []byte("<"), Right: []byte(">"),
			Workspace: syntax_from(&workspace),
		})
		if status != template.PARSE_STATUS_OK {
			continue
		}
		execute_shared_from(t, &execution, program, value_nil(), functions[:])
	}
}

// Definitions fill the source, so the document reports its largest template count.
func test_definition_maximum(t *testing.T) {
	t.Helper()
	bodies := [...]string{
		"", `<$x:=.><if $x>a<else>b<end>>`, `<print 1 "a" 'b' 1.5>`,
		`<range .><.><end><template "aa" .>`,
	}
	for _, body := range bodies {
		test_definition_body(t, body)
	}
}

// TEST_NAME_FIRST_BYTE opens the alphabet one quoted name counts through.
const TEST_NAME_FIRST_BYTE = ' '

// TEST_NAME_LAST_BYTE closes that alphabet at the last printable character.
const TEST_NAME_LAST_BYTE = int(utf8.CHARACTER_SELF) - TEST_COUNT_TWO

// Steps one name to the next distinct name the quoted alphabet admits.
func next_name(name []byte) (next []byte) {
	next = name
	for index := len(next) - TEST_COUNT_ONE; index >= TEST_COUNT_ZERO; index-- {
		next[index]++
		for next[index] == '"' || next[index] == '\\' {
			next[index]++
		}
		if int(next[index]) <= TEST_NAME_LAST_BYTE {
			return next
		}
		next[index] = TEST_NAME_FIRST_BYTE
	}
	return append([]byte{TEST_NAME_FIRST_BYTE}, next...)
}

// Distinct names pack the definition table to its bound, one body at a time.
func test_template_count_maximum(t *testing.T) {
	t.Helper()
	bodies := [...]string{
		`<if .><.><end>`, `<.a>`, `<identity>`, `<1>`, `<"a">`, `<$x:=.><$x>`,
		`<range .><end>`, `<(.).a>`, `<print . .>`, `<identity . .>`,
	}
	for _, body := range bodies {
		test_template_count_body(t, body)
	}
	// One table at its bound holds the shortest definition its source can repeat.
	filled := make([]byte, TEST_COUNT_ZERO, template.SOURCE_SIZE_MAXIMUM)
	for len(filled)+len(`<define ""><end>`) <= template.SOURCE_SIZE_MAXIMUM {
		filled = append(filled, `<define ""><end>`...)
	}
	var workspace template.Syntax_Workspace
	program, _, status := template.Compile(template.Compile_Input{
		Source: filled, Left: []byte("<"), Right: []byte(">"),
		Workspace: syntax_from(&workspace),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	execute_functions_from(t, program, value_text(nil), nil)
}

// Test_Merge_Bounds merges definitions across a table at its bound.
func Test_Merge_Bounds(t *testing.T) {
	t.Parallel()
	test_template_merge_maximum(t)
}

// Test_Executed_Table_Bounds runs each construct beside a table at its bound.
func Test_Executed_Table_Bounds(t *testing.T) {
	t.Parallel()
	test_executed_template_count_maximum(t)
}

// Test_First_Template_Bounds keeps the first definition at the arena end.
func Test_First_Template_Bounds(t *testing.T) {
	t.Parallel()
	test_first_template_maximum(t)
}

// One executed document names as many bodies as its source can define.
func test_executed_template_count_maximum(t *testing.T) {
	t.Helper()
	tails := [...]string{
		`<.>`, `<range .>x<end>`, `<with .>x<end>`, `<if .>x<end>`,
		`<$x:=.><$x>`, `<identity>`, `<.a>`, `<1>`, `<. | identity>`,
		"<template `a` .>", "<template `a`>", `<range $i, $e := .>x<end>`,
		`<eq identity 1>`,
	}
	for _, tail := range tails {
		test_executed_template_count_tail(t, tail)
	}
}

// One executed table at its bound stands beside the construct that follows it.
func test_executed_template_count_tail(t *testing.T, tail string) {
	t.Helper()
	source := make([]byte, TEST_COUNT_ZERO, template.SOURCE_SIZE_MAXIMUM)
	candidate := uint16(bits.WORD_8_MINIMUM)
	for len(source)+len("<define ``><end>")+TEST_COUNT_ONE+len(tail) <=
		template.SOURCE_SIZE_MAXIMUM {
		name := byte(candidate)
		candidate++
		if name == '`' {
			continue
		}
		if name == '\r' {
			continue
		}
		source = append(source, "<define `"...)
		source = append(source, name)
		source = append(source, "`><end>"...)
	}
	source = append(source, tail...)
	var workspace template.Syntax_Workspace
	program, _, status := template.Compile(template.Compile_Input{
		Source: source, Left: []byte("<"), Right: []byte(">"),
		Workspace: syntax_from(&workspace),
	})
	if status != template.PARSE_STATUS_OK {
		return
	}
	functions := [...]template.Function{{Name: []byte("identity"), Call: identity}}
	items := [...]template.Value{value_text(nil)}
	execute_functions_from(t, program, value_sequence(items[:]), functions[:])
}

// One branch stands in the densest program its source can hold.
func test_dense_branch(t *testing.T, part int) {
	t.Helper()
	constructs := [...]string{
		`<if .>x<end>`, `<range .>x<end>`, `<with .>x<end>`, `<.a>`, `<$a:=.><$a>`,
		`<(.).a>`, `<x .>`, `<1>`, `<"a">`, `<x>`, `<. | x>`, `<x . .>`,
	}
	functions := [...]template.Function{{Name: []byte("x"), Call: identity}}
	// One short iterable dot keeps the dense output inside its bound.
	items := [...]template.Value{value_text(nil)}
	dot := value_sequence(items[:])
	var execution execution_storage
	for index, construct := range constructs {
		if index%TEST_COUNT_FOUR != part {
			continue
		}
		source := make([]byte, TEST_COUNT_ZERO, template.SOURCE_SIZE_MAXIMUM)
		for len(source)+len("<.>")+len(construct) <= template.SOURCE_SIZE_MAXIMUM {
			source = append(source, "<.>"...)
		}
		for len(source)+len(construct) < template.SOURCE_SIZE_MAXIMUM {
			source = append(source, 'x')
		}
		source = append(source, construct...)
		var workspace template.Syntax_Workspace
		program, _, status := template.Compile(template.Compile_Input{
			Source: source, Left: []byte("<"), Right: []byte(">"),
			Workspace: syntax_from(&workspace),
		})
		if status != template.PARSE_STATUS_OK {
			continue
		}
		// Only a branch reads the sequence; empty text keeps packed actions silent.
		if index > TEST_COUNT_TWO {
			execute_shared_from(t, &execution, program, value_text(nil), functions[:])
			continue
		}
		execute_shared_from(t, &execution, program, dot, functions[:])
	}
}

// One definition at the arena end keeps the first-template reference at its bound.
func test_first_template_maximum(t *testing.T) {
	t.Helper()
	tails := [...]string{
		`<define "t"><end>`, `<define "t">x<end>`, `<block "b" .>x<end>`,
	}
	constructs := [...]string{
		``, `<if .>x<end>`, `<range .>x<end>`, `<with .>x<end>`, `<.a>`,
		`<$a:=.><$a>`, `<x .>`, `<(.).a>`, `<print . .>`,
	}
	functions := [...]template.Function{{Name: []byte("x"), Call: identity}}
	for index, tail := range tails {
		construct := constructs[index%len(constructs)]
		source := make([]byte, TEST_COUNT_ZERO, template.SOURCE_SIZE_MAXIMUM)
		reserve := len("<.>") + len(construct) + len(tail)
		for len(source)+reserve <= template.SOURCE_SIZE_MAXIMUM {
			source = append(source, "<.>"...)
		}
		source = append(source, construct...)
		source = append(source, tail...)
		var workspace template.Syntax_Workspace
		template.Compile(template.Compile_Input{
			Source: source, Left: []byte("<"), Right: []byte(">"),
			Workspace: syntax_from(&workspace),
		})
	}
	test_first_template_constructs(
		t, tails[TEST_COUNT_ZERO], constructs[:], value_text(nil), functions[:])
}

// Each construct runs beside a definition the arena holds at its end.
func test_first_template_constructs(
	t *testing.T, tail string, constructs []string, dot template.Value,
	functions template.Functions,
) {
	t.Helper()
	var execution execution_storage
	for _, construct := range constructs {
		source := make([]byte, TEST_COUNT_ZERO, template.SOURCE_SIZE_MAXIMUM)
		reserve := len("<.>") + len(construct) + len(tail)
		for len(source)+reserve <= template.SOURCE_SIZE_MAXIMUM {
			source = append(source, "<.>"...)
		}
		source = append(source, construct...)
		source = append(source, tail...)
		var workspace template.Syntax_Workspace
		program, _, status := template.Compile(template.Compile_Input{
			Source: source, Left: []byte("<"), Right: []byte(">"),
			Workspace: syntax_from(&workspace),
		})
		if status != template.PARSE_STATUS_OK {
			continue
		}
		execute_shared_from(t, &execution, program, dot, functions)
	}
}

// Repeated names merge into the table one definition at a time.
func test_template_merge_maximum(t *testing.T) {
	t.Helper()
	source := make([]byte, TEST_COUNT_ZERO, template.SOURCE_SIZE_MAXIMUM)
	name := []byte{TEST_NAME_FIRST_BYTE}
	definition := func(name []byte) (bytes []byte) {
		bytes = append(bytes, `<define "`...)
		bytes = append(bytes, name...)
		return append(bytes, `"><end>`...)
	}
	for len(source)+len(definition(name))*TEST_COUNT_TWO <=
		template.SOURCE_SIZE_MAXIMUM {
		source = append(source, definition(name)...)
		source = append(source, definition(name)...)
		name = next_name(name)
	}
	var workspace template.Syntax_Workspace
	program, _, status := template.Compile(template.Compile_Input{
		Source: source, Left: []byte("<"), Right: []byte(">"),
		Workspace: syntax_from(&workspace),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	execute_functions_from(t, program, value_text(nil), nil)
	test_template_merge_widest(t)
}

// One empty definition closes a table at its bound, so its merge scans every entry.
func test_template_merge_widest(t *testing.T) {
	t.Helper()
	source := append(make([]byte, TEST_COUNT_ZERO, template.SOURCE_SIZE_MAXIMUM),
		`<define ""><end>`...)
	candidate := uint16(bits.WORD_8_MINIMUM)
	for len(source)+len("<define ``><end>")+TEST_COUNT_ONE+len(`<define ""><end>`) <=
		template.SOURCE_SIZE_MAXIMUM {
		name := byte(candidate)
		candidate++
		if name == '`' {
			continue
		}
		if name == '\r' {
			continue
		}
		source = append(source, "<define `"...)
		source = append(source, name)
		source = append(source, "`><end>"...)
	}
	source = append(source, `<define ""><end>`...)
	var workspace template.Syntax_Workspace
	_, _, status := template.Compile(template.Compile_Input{
		Source: source, Left: []byte("<"), Right: []byte(">"),
		Workspace: syntax_from(&workspace),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
}

// One body runs inside the block that closes a table at its bound.
func test_template_count_body(t *testing.T, body string) {
	t.Helper()
	// Each distinct one-byte name keeps its own table entry beside the body.
	source := make([]byte, TEST_COUNT_ZERO, template.SOURCE_SIZE_MAXIMUM)
	candidate := uint16(bits.WORD_8_MINIMUM)
	for len(source)+len("<define ``><end>")+TEST_COUNT_ONE+len(body) <=
		template.SOURCE_SIZE_MAXIMUM {
		name := byte(candidate)
		candidate++
		if name == '`' {
			continue
		}
		if name == '\r' {
			continue
		}
		source = append(source, "<define `"...)
		source = append(source, name)
		source = append(source, "`><end>"...)
	}
	source = append(source, body...)
	var workspace template.Syntax_Workspace
	program, _, status := template.Compile(template.Compile_Input{
		Source: source, Left: []byte("<"), Right: []byte(">"),
		Workspace: syntax_from(&workspace),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	functions := [...]template.Function{{Name: []byte("identity"), Call: identity}}
	fields := [...]template.Field{{
		Name:  template.Field_Name("a"),
		Value: template.Field_Value{Data: value_text(template.Text("z"))},
	}}
	execute_functions_from(t, program, value_object(fields[:]), functions[:])
}

// One caller limit at the arena bound parses like the default limit.
func test_explicit_node_limit(
	t *testing.T, source []byte, mode template.Mode, left string, right string,
) {
	t.Helper()
	// The smallest caller limits hold the root alone, so every action overruns them.
	limits := [...]int{
		template.NODE_COUNT_MAXIMUM, TEST_COUNT_ONE, TEST_COUNT_TWO,
	}
	configuration := configuration_input_from(t, template.Configuration_Input{
		Left:  template.Left_Delimiter_Unvalidated(left),
		Right: template.Right_Delimiter_Unvalidated(right),
		Mode:  mode,
	})
	for _, limit := range limits {
		limited := template.Syntax_Workspace{
			Node_Limit: template.Node_Limit_Storage{
				Value: template.Node_Limit_Stored(limit),
			},
		}
		template.Parse_Into(
			template.Source_Header_Unvalidated(source_from(t, string(source))),
			configuration,
			parse_workspace_from(&limited),
		)
	}
}

// Definitions fill the source up to one trailing body that execution runs.
func test_definition_body(t *testing.T, body string) {
	t.Helper()
	// Each definition needs its own name, so two name bytes carry the count.
	source := make([]byte, TEST_COUNT_ZERO, template.SOURCE_SIZE_MAXIMUM)
	definition := []byte(`<define "aa">x<end>`)
	prefix_count := len(`<define "`)
	index := TEST_COUNT_ZERO
	limit := template.SOURCE_SIZE_MAXIMUM - len(body)
	for len(source)+len(definition) <= limit {
		definition[prefix_count] = byte('a' + index/TEST_ALPHABET_SIZE)
		definition[prefix_count+TEST_COUNT_ONE] = byte('a' + index%TEST_ALPHABET_SIZE)
		source = append(source, definition...)
		index++
	}
	source = append(source, body...)
	var workspace template.Syntax_Workspace
	functions := [...]template.Function{{Name: []byte("identity"), Call: identity}}
	program, _, status := template.Compile(template.Compile_Input{
		Source: source, Left: []byte("<"), Right: []byte(">"),
		Workspace: syntax_from(&workspace),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	execute_functions_from(t, program, value_text(nil), functions[:])
}

// Replaces each old byte run with new bytes.
func strings_replace(text string, old string, new string) (result string) {
	for index := 0; index < len(text); {
		if index+len(old) <= len(text) {
			if text[index:index+len(old)] == old {
				result += new
				index += len(old)
				continue
			}
		}
		result += text[index : index+1]
		index++
	}
	return result
}

// Construct_Run is one construct, the policy it compiles under, and the storage it runs in.
type Construct_Run struct {
	Test      *testing.T
	Execution *execution_storage
	Syntax    *template.Syntax_Workspace
	Source    []byte
	Mode      template.Mode
	Values    []template.Value
	Functions template.Functions
	Left      string
	Right     string
}

// One construct compiles under one mode, then runs against each given value.
// One construct stands at each source position its own bounds admit.
func test_construct_placements(run Construct_Run) {
	run.Test.Helper()
	construct := string(run.Source)
	doubled := strings_replace(strings_replace(construct, "<", "<<"), ">", ">>")
	head := run
	head.Values = run.Values[:TEST_COUNT_ONE]
	// Comment padding puts the construct at the end of the largest source.
	head.Source = padded_source(construct)
	test_construct_source(head)
	// Two-byte delimiters start that content at the second source byte.
	doubled_run := head
	doubled_run.Source, doubled_run.Left, doubled_run.Right = []byte(doubled), "<<", ">>"
	test_construct_source(doubled_run)
	// The bare construct starts its action content at the first source byte.
	test_construct_source(run)
	// One and two leading text bytes open the construct at those positions.
	head.Source = []byte("a" + construct)
	test_construct_source(head)
	head.Source = []byte("ab" + construct)
	test_construct_source(head)
}

// One declaration run ahead of the construct fills the lexical table it parses against.
func test_declared_constructs(t *testing.T) {
	t.Helper()
	constructs := [...]string{
		`<define "t"><end>`, `<block "t" .><end>`, `<template "t" .>`, `<if .><end>`,
		`<range .><break><end>`, `<with .><else><end>`, `<$a>`, `<$a:=.>`,
	}
	var execution execution_storage
	for _, construct := range constructs {
		var workspace template.Syntax_Workspace
		program, _, status := template.Compile(template.Compile_Input{
			Source: declared_source(construct), Left: []byte("<"), Right: []byte(">"),
			Workspace: syntax_from(&workspace),
		})
		if status != template.PARSE_STATUS_OK {
			continue
		}
		execute_shared_from(t, &execution, program, value_text(nil), nil)
	}
}

// One declaration run ahead of the construct ends the source at its last byte.
func declared_source(construct string) (source []byte) {
	source = make([]byte, TEST_COUNT_ZERO, template.SOURCE_SIZE_MAXIMUM)
	declaration := `<$a:=.>`
	for len(source)+len(declaration)+len(construct) <= template.SOURCE_SIZE_MAXIMUM {
		source = append(source, declaration...)
	}
	return append(source, construct...)
}

func test_construct_source(run Construct_Run) {
	run.Test.Helper()
	program, _, status := template.Compile(template.Compile_Input{
		Source: run.Source, Left: []byte(run.Left), Right: []byte(run.Right),
		Mode: run.Mode, Workspace: syntax_from(run.Syntax),
	})
	if status != template.PARSE_STATUS_OK {
		return
	}
	for _, value := range run.Values {
		execute_shared_from(run.Test, run.Execution, program, value, run.Functions)
	}
}

// Comment padding with one-byte delimiters ends the construct at the final source byte.
func padded_source(construct string) (source []byte) {
	source = make([]byte, template.SOURCE_SIZE_MAXIMUM-len(construct))
	for index := range source {
		source[index] = 'a'
	}
	copy(source, "</*")
	copy(source[len(source)-len("*/>"):], "*/>")
	return append(source, construct...)
}

// Reports whether one value carries a payload at its maximum extent.
func matrix_large(value template.Value) (large bool) {
	if len(value.Text) > TEST_COUNT_TWO {
		return true
	}
	if len(value.Values) > TEST_COUNT_TWO {
		return true
	}
	return len(value.Fields) > TEST_COUNT_TWO
}

// Gives the largest text one value holds.
func matrix_text() (text []byte) {
	text = make([]byte, template.OUTPUT_SIZE_MAXIMUM)
	for index := range text {
		text[index] = 'a'
	}
	return text
}

// Gives one value at each bound of every payload field.
func matrix_values() (values []template.Value) {
	text := matrix_text()
	scalars := matrix_scalars()
	values = append(values, scalars...)
	values = append(values, matrix_composites(scalars)...)
	values = append(values, matrix_composites(matrix_composites(scalars))...)
	largest := matrix_largest(scalars, text)
	values = append(values, largest...)
	// Graph validation holds one maximum payload only when it sits alone in its parent.
	for index := range largest {
		values = append(values, matrix_composites(largest[index:index+TEST_COUNT_ONE])...)
	}
	keys := matrix_keys(text)
	mixed := make([]template.Field, len(keys))
	for index := range keys {
		mixed[index] = template.Field{
			Key:   template.Field_Key{Data: keys[index]},
			Value: template.Field_Value{Data: scalars[index%len(scalars)]},
		}
	}
	// Reversed keys walk backward; one swap forces a scan per key.
	reversed := make([]template.Field, len(mixed))
	for index := range mixed {
		reversed[len(mixed)-TEST_COUNT_ONE-index] = mixed[index]
	}
	swapped := append([]template.Field(nil), mixed...)
	swapped[TEST_COUNT_ZERO] = mixed[TEST_COUNT_ONE]
	swapped[TEST_COUNT_ONE] = mixed[TEST_COUNT_ZERO]
	return append(values, value_map(mixed), value_map(reversed), value_map(swapped))
}

// Gives every template shape the matrix runs.
func matrix_sources() (sources []string) {
	return []string{
		`{{.}}`, `{{print .}}`, `{{println .}}`, `{{printf "%v" .}}`, `{{printf "%d" .}}`,
		`{{printf "%s" .}}`, `{{printf "%q" .}}`, `{{printf "%T" .}}`, `{{and .}}`, `{{and . .}}`, `{{or . .}}`,
		`{{not .}}`, `{{len .}}`, `{{index . 0}}`, `{{slice . 0}}`,
		`{{eq . .}}`, `{{ne . .}}`, `{{lt . .}}`, `{{le . .}}`,
		`{{if .}}y{{else}}n{{end}}`, `{{with .}}{{.}}{{end}}`, `{{range .}}{{.}}{{end}}`,
		`{{range $k, $v := .}}{{$k}}{{$v}}{{end}}`, `{{$x := .}}{{$x}}`,
		`{{template "t" .}}{{define "t"}}{{.}}{{end}}`, `{{identity .}}`, `{{html .}}`,
		`{{js .}}`, `{{call . .}}`, `{{index . "a"}}`,
		`{{. (.)}}`, `{{. .}}`, `{{(.) (.)}}`, `{{. | identity}}`,
		`{{. | identity | identity}}`, `{{(.).a}}`, `{{(.).a.b}}`, `{{.a}}`, `{{.a.b}}`,
		`{{(. | identity).a}}`, `{{$x := .}}{{$x = .}}{{$x}}`, `{{and (.) (.)}}`,
		`{{or (.) (.)}}`, `{{identity (.)}}`, `{{with $x := .}}{{$x}}{{end}}`,
		`{{index . .}}`, `{{slice . .}}`, `{{slice . . .}}`, `{{call identity .}}`,
		`{{. | identity (.)}}`, `{{$x := . (.)}}{{$x}}`, `{{eq . . .}}`, `{{le . .}}`,
		`x{{len .}}`, `{{$x := .}}{{if .}}{{$x}}{{end}}`,
		`{{$x := .}}{{range .}}{{$x}}{{end}}`, `{{$x := .}}{{with .}}{{$x}}{{end}}`,
		`{{define "t"}}{{$y := .}}{{$y}}{{end}}{{$z := .}}{{template "t" $z}}`,
		`{{index listing .}}`, `{{index mapping .}}`, `{{slice listing .}}`,
		`{{print . .}}`, `{{printf "%s%s" . .}}`, `{{print (print "a")}}`, `{{html (print "a") .}}`, `{{printf "%s" (print . .)}}`,
		`{{slice listing 0 .}}`, `{{index listing . .}}`, `{{len listing}}`,
		`{{range .}}{{template "t" .}}{{end}}{{define "t"}}{{.}}{{end}}`,
	}
}

// Gives every scalar payload bound.
func matrix_scalars() (values []template.Value) {
	values = []template.Value{
		value_nil(),
		value_boolean(true),
		value_boolean(false),
		value_integer(TEST_COUNT_ZERO),
		value_integer(TEST_COUNT_ONE),
		value_integer(TEST_COUNT_TWO),
		value_integer(-TEST_COUNT_ONE),
		value_unsigned(template.Unsigned(bits.WORD_64_MAXIMUM)),
		value_float(template.Float(bits.WORD_64_MAXIMUM)),
		value_function(identity),
	}
	imaginary := [...]template.Complex_Imaginary{
		TEST_COUNT_ZERO, TEST_COUNT_ONE, TEST_COUNT_TWO,
		template.Complex_Imaginary(bits.WORD_64_MAXIMUM),
	}
	for index := range imaginary {
		complex := template.Complex{Real: TEST_COUNT_ONE, Imaginary: imaginary[index]}
		values = append(values, value_complex(complex))
	}
	for _, text := range [...]string{"", "a", "aa"} {
		values = append(values, value_text([]byte(text)))
	}
	return values
}

// Gives sequence, object, and map holding each given value at extents one, two, and all.
func matrix_composites(children []template.Value) (values []template.Value) {
	extents := [...]int{TEST_COUNT_ONE, TEST_COUNT_TWO, len(children)}
	previous := TEST_COUNT_ZERO
	for _, extent_count := range extents {
		if extent_count > len(children) {
			continue
		}
		if extent_count == previous {
			continue
		}
		previous = extent_count
		members := make([]template.Field, extent_count)
		entries := make([]template.Field, extent_count)
		for index := range members {
			name := []byte{'f', byte(index)}
			value := template.Field_Value{Data: children[index]}
			members[index] = template.Field{Name: name, Value: value}
			key := template.Field_Key{Data: value_integer(template.Integer(index))}
			entries[index] = template.Field{Key: key, Value: value}
		}
		values = append(
			values, value_sequence(children[:extent_count]), value_object(members),
			value_map(entries),
		)
	}
	return values
}

// Gives every payload at its maximum extent.
func matrix_largest(scalars []template.Value, text []byte) (values []template.Value) {
	items := make([]template.Value, template.VALUE_COUNT_MAXIMUM)
	members := make([]template.Field, template.VALUE_COUNT_MAXIMUM)
	for index := range members {
		value := template.Field_Value{Data: scalars[index%len(scalars)]}
		items[index] = value.Data
		name := []byte{'f', byte(index >> bits.BIT_COUNT_8_MAXIMUM), byte(index)}
		members[index] = template.Field{Name: name, Value: value}
	}
	return []template.Value{value_sequence(items), value_object(members), value_text(text)}
}

// Gives map at its admitted maximum entry count.
func matrix_largest_map(scalars []template.Value) (value template.Value) {
	entries := make([]template.Field, template.RANGE_MAP_ENTRY_COUNT_MAXIMUM)
	for index := range entries {
		key := template.Field_Key{Data: value_integer(template.Integer(index))}
		data := template.Field_Value{Data: scalars[index%len(scalars)]}
		entries[index] = template.Field{Key: key, Value: data}
	}
	return value_map(entries)
}

// Gives every admitted key kind at its bounds in ascending map order.
func matrix_keys(text []byte) (keys []template.Value) {
	keys = []template.Value{value_nil(), value_boolean(false), value_boolean(true)}
	integers := [...]template.Integer{
		template.Integer(bits.INTEGER_64_MINIMUM), -TEST_COUNT_ONE, TEST_COUNT_ZERO,
		TEST_COUNT_ONE, TEST_COUNT_TWO,
		template.Integer(bits.INTEGER_64_MAXIMUM),
	}
	for _, integer := range integers {
		keys = append(keys, value_integer(integer))
	}
	unsigned := [...]template.Unsigned{
		TEST_COUNT_ZERO, TEST_COUNT_ONE, TEST_COUNT_TWO,
		template.Unsigned(bits.WORD_64_MAXIMUM),
	}
	for _, number := range unsigned {
		keys = append(keys, value_unsigned(number))
	}
	// Sort order puts NaN before every number.
	floats := [...]template.Float{
		template.Float(bits.WORD_64_MAXIMUM),
		TEST_COUNT_ZERO, TEST_COUNT_ONE, TEST_COUNT_TWO,
	}
	for _, float := range floats {
		keys = append(keys, value_float(float))
	}
	for _, float := range floats {
		imaginary := template.Complex_Imaginary(float)
		complex := template.Complex{Real: TEST_COUNT_ONE, Imaginary: imaginary}
		keys = append(keys, value_complex(complex))
	}
	extents := [...]int{
		TEST_COUNT_ZERO, TEST_COUNT_ONE, TEST_COUNT_TWO, template.OUTPUT_SIZE_MAXIMUM,
	}
	for _, extent := range extents {
		keys = append(keys, value_text(text[:extent]))
	}
	return keys
}

func value_nil() (value template.Value) {
	return template.Value(template.Value_Nil())
}

func value_boolean(truth template.Boolean) (value template.Value) {
	return template.Value(template.Value_Of_Boolean(truth))
}

func value_integer(integer template.Integer) (value template.Value) {
	return template.Value(template.Value_Of_Integer(integer))
}

func value_unsigned(unsigned template.Unsigned) (value template.Value) {
	return template.Value(template.Value_Of_Unsigned(unsigned))
}

func value_float(float template.Float) (value template.Value) {
	return template.Value(template.Value_Of_Float(float))
}

func value_complex(complex template.Complex) (value template.Value) {
	return template.Value(template.Value_Of_Complex(complex))
}

func value_text(text template.Text) (value template.Value) {
	return template.Value(template.Value_Of_Text(text))
}

func value_sequence(values template.Values) (value template.Value) {
	return template.Value(template.Value_Of_Sequence(values))
}

func value_object(fields template.Fields) (value template.Value) {
	return template.Value(template.Value_Of_Object(fields))
}

func value_map(fields template.Fields) (value template.Value) {
	return template.Value(template.Value_Of_Map(fields))
}

func value_function(call template.Function_Call) (value template.Value) {
	return template.Value(template.Value_Of_Function(call))
}

// One definition table at its bound answers every lookup its bodies make.
func Test_Table_Bounds(t *testing.T) {
	t.Parallel()
	test_template_count_maximum(t)
}

// Each construct stands in the densest program its source can hold.
func Test_Dense_Construct_Bounds(t *testing.T) {
	t.Parallel()
	test_dense_branch(t, TEST_COUNT_ZERO)
}

// Each remaining construct stands at the end of the densest source that still holds it.
func Test_Dense_Construct_Tails(t *testing.T) {
	t.Parallel()
	test_dense_branch(t, TEST_COUNT_ONE)
}

// Test_Dense_Construct_Middles stands the third quarter of constructs at the arena end.
func Test_Dense_Construct_Middles(t *testing.T) {
	t.Parallel()
	test_dense_branch(t, TEST_COUNT_TWO)
}

// Test_Dense_Construct_Ends stands the last quarter of constructs at the arena end.
func Test_Dense_Construct_Ends(t *testing.T) {
	t.Parallel()
	test_dense_branch(t, TEST_COUNT_THREE)
}

// Test_Hostile_Values refuses each caller value at its first unsupported bound.
func Test_Hostile_Values(t *testing.T) {
	program := compile_from(t, `{{.}}`)
	text := make([]byte, template.OUTPUT_SIZE_UNVALIDATED_MAXIMUM)
	items := make([]template.Value, template.VALUE_COUNT_UNVALIDATED_MAXIMUM)
	members := make([]template.Field, template.VALUE_COUNT_UNVALIDATED_MAXIMUM)
	boolean := template.Value_Kind_Storage_Value(template.VALUE_BOOLEAN)
	integer := template.Value_Kind_Storage_Value(template.VALUE_INTEGER)
	textual := template.Value_Kind_Storage_Value(template.VALUE_TEXT)
	sequence := template.Value_Kind_Storage_Value(template.VALUE_SEQUENCE)
	mapped := template.Value_Kind_Storage_Value(template.VALUE_MAP)
	hostile := [...]template.Value_Unvalidated{
		{Kind: template.Value_Kind_Storage_Value(bits.WORD_8_MAXIMUM)},
		{Kind: template.Value_Kind_Storage_Value(template.VALUE_TEXT), Text: text},
		{Kind: template.Value_Kind_Storage_Value(template.VALUE_SEQUENCE), Values: items},
		{Kind: template.Value_Kind_Storage_Value(template.VALUE_OBJECT), Fields: members},
		// Each forged value carries one payload its kind does not own.
		{Real: TEST_COUNT_ONE},
		{Kind: boolean, Real: TEST_COUNT_TWO},
		{Kind: integer, Imaginary: TEST_COUNT_ONE},
		{Kind: integer, Text: text[:TEST_COUNT_ONE]},
		{Kind: textual, Values: items[:TEST_COUNT_ONE]},
		{Kind: mapped, Values: items[:TEST_COUNT_ONE]},
		{Kind: sequence, Fields: members[:TEST_COUNT_ONE]},
		{Kind: textual, Function: identity},
	}
	functions := [...]template.Function{{Name: []byte("identity"), Call: identity}}
	for index := range hostile {
		var execution execution_storage
		var destination [template.OUTPUT_SIZE_MAXIMUM]byte
		_, _, status := template.Execute_Into(template.Execute_Input{
			Destination: destination[:], Program: template.Program_Unvalidated(program),
			Value:     hostile[index],
			Functions: functions[:], Workspace: workspace_from(&execution),
		})
		testify.Not_Equal(t, template.STATUS_OK, status)
	}
}

type workspace_mutation struct {
	Limit  int
	Resize func(storage *execution_storage, size int)
}

// Test_Hostile_Workspaces refuses each caller arena at its first unsupported size.
func Test_Hostile_Workspaces(t *testing.T) {
	program := compile_from(t, `{{.}}`)
	for _, mutation := range workspace_mutations() {
		// An empty arena is the first size a caller can hand over.
		sizes := [...]int{
			TEST_COUNT_ZERO, TEST_COUNT_ONE, TEST_COUNT_TWO, mutation.Limit,
		}
		for _, size := range sizes {
			var storage execution_storage
			input := workspace_from(&storage)
			mutation.Resize(&storage, size)
			var destination [template.OUTPUT_SIZE_MAXIMUM]byte
			template.Execute_Into(template.Execute_Input{
				Destination: destination[:],
				Program:     template.Program_Unvalidated(program),
				Value:       template.Value_Unvalidated(value_nil()),
				Workspace:   input,
			})
		}
	}
	var storage execution_storage
	input := workspace_from(&storage)
	input.Float_Parse, input.Float_Text = nil, nil
	var destination [template.OUTPUT_SIZE_MAXIMUM]byte
	template.Execute_Into(template.Execute_Input{
		Destination: destination[:], Program: template.Program_Unvalidated(program),
		Value: template.Value_Unvalidated(value_nil()), Workspace: input,
	})
}

// Gives one resize per caller arena; each limit is that arena's first refused size.
func workspace_mutations() (mutations []workspace_mutation) {
	mutations = append(workspace_arena_mutations(), workspace_frame_mutations()...)
	return append(mutations, float_scratch_mutations()...)
}

func workspace_arena_mutations() (mutations []workspace_mutation) {
	return []workspace_mutation{
		{
			Limit: template.OUTPUT_SIZE_UNVALIDATED_MAXIMUM,
			Resize: func(s *execution_storage, n int) {
				s.Workspace.Output = make(template.Output_Storage, n)
			}},
		{Limit: template.QUOTED_DECODED_SIZE_UNVALIDATED_MAXIMUM,
			Resize: func(s *execution_storage, n int) {
				s.Workspace.Literals = make(template.Literal_Storage, n)
			}},
		{
			Limit: template.NODE_LIMIT_UNVALIDATED_MAXIMUM,
			Resize: func(s *execution_storage, n int) {
				s.Workspace.Literal_Counts = make(template.Literal_Counts, n)
			}},
		{
			Limit: template.NODE_LIMIT_UNVALIDATED_MAXIMUM,
			Resize: func(s *execution_storage, n int) {
				s.Workspace.Literal_Offsets = make(template.Literal_Offsets, n)
			}},
		{
			Limit: template.NODE_LIMIT_UNVALIDATED_MAXIMUM,
			Resize: func(s *execution_storage, n int) {
				s.Workspace.Literal_Decoded = make(template.Literal_Decoded, n)
			}},
		{Limit: template.QUOTED_DECODED_SIZE_UNVALIDATED_MAXIMUM,
			Resize: func(s *execution_storage, n int) {
				s.Workspace.Name_Left = make(template.Name_Left_Storage, n)
			}},
		{Limit: template.QUOTED_DECODED_SIZE_UNVALIDATED_MAXIMUM,
			Resize: func(s *execution_storage, n int) {
				s.Workspace.Name_Right = make(template.Name_Right_Storage, n)
			}},
		{
			Limit: template.NUMBER_SIZE_UNVALIDATED_MAXIMUM,
			Resize: func(s *execution_storage, n int) {
				s.Workspace.Number = make(template.Number_Storage, n)
			}},
		{Limit: template.GENERATED_SIZE_UNVALIDATED_MAXIMUM,
			Resize: func(s *execution_storage, n int) {
				s.Workspace.Generated = make(template.Generated_Storage, n)
			}},
	}
}

func workspace_frame_mutations() (mutations []workspace_mutation) {
	return []workspace_mutation{
		{
			Limit: template.FRAME_COUNT_UNVALIDATED_MAXIMUM,
			Resize: func(s *execution_storage, n int) {
				s.Workspace.Frames = make(template.Frames, n)
			}},
		{
			Limit: template.VARIABLE_COUNT_UNVALIDATED_MAXIMUM,
			Resize: func(s *execution_storage, n int) {
				s.Workspace.Variables = make(template.Variables, n)
			}},
		{
			Limit: template.ARGUMENT_COUNT_UNVALIDATED_MAXIMUM,
			Resize: func(s *execution_storage, n int) {
				s.Workspace.Arguments = make(template.Arguments, n)
			}},
		{
			Limit: template.FRAME_COUNT_UNVALIDATED_MAXIMUM,
			Resize: func(s *execution_storage, n int) {
				s.Workspace.Validation = make(template.Validation_Frames, n)
			}},
		{
			Limit: template.FRAME_COUNT_UNVALIDATED_MAXIMUM,
			Resize: func(s *execution_storage, n int) {
				s.Workspace.Evaluations = make(template.Evaluations, n)
			}},
		{
			Limit: template.FRAME_COUNT_UNVALIDATED_MAXIMUM,
			Resize: func(s *execution_storage, n int) {
				s.Workspace.Format = make(template.Value_Format_Frames, n)
			}},
		{
			Limit: int(bits.WORD_16_MAXIMUM),
			Resize: func(s *execution_storage, n int) {
				hostile_cursors(s, n)
			}},
	}
}

func float_scratch_mutations() (mutations []workspace_mutation) {
	return []workspace_mutation{
		{
			Limit: template.WORD_COUNT_UNVALIDATED_MAXIMUM,
			Resize: func(s *execution_storage, n int) {
				s.Float_Text.Integers.Significand.Words = make(big.Words, n)
				s.Float_Text.Integers.Exponent.Words = make(big.Words, n)
				s.Float_Text.Values.Source.Mantissa.Words = make(big.Words, n)
				s.Float_Text.Values.Rounded.Mantissa.Words = make(big.Words, n)
				s.Float_Text.Text.Significand.Words = make(big.Int_Text_Words, n)
				s.Float_Text.Text.Exponent.Words = make(big.Int_Text_Words, n)
			}},
		{Limit: template.DECIMAL_DIGIT_COUNT_UNVALIDATED_MAXIMUM,
			Resize: func(s *execution_storage, n int) {
				hostile_digits(s, n)
				s.Float_Text.Control = nil
			}},
		{Limit: template.MAGNITUDE_WORD_COUNT_UNVALIDATED_MAXIMUM,
			Resize: func(s *execution_storage, n int) {
				s.Float_Text.Magnitude.Words = make(
					big.Float_Text_Magnitude_Words, n,
				)
			}},
		{Limit: template.FLOAT_TEXT_SIZE_UNVALIDATED_MAXIMUM,
			Resize: func(s *execution_storage, n int) {
				s.Float_Text.Output = make(big.Float_Text_Output, n)
			}},
		{Limit: template.FLOAT_PARSE_SOURCE_COUNT_UNVALIDATED_MAXIMUM,
			Resize: func(s *execution_storage, n int) {
				s.Float_Parse.Source = make(big.Float_Parse_Source, n)
			}},
		{Limit: template.FLOAT_PARSE_CONTROL_COUNT_UNVALIDATED_MAXIMUM,
			Resize: func(s *execution_storage, n int) {
				s.Float_Parse.Control = make(big.Float_Parse_Control, n)
			}},
		{
			Limit: int(bits.INTEGER_64_MAXIMUM),
			Resize: func(s *execution_storage, n int) {
				s.Float_Text.Magnitude.Count = big.Float_Text_Magnitude_Count(n)
			}},
		{
			Limit: int(bits.INTEGER_64_MINIMUM),
			Resize: func(s *execution_storage, n int) {
				s.Float_Text.Magnitude.Count = big.Float_Text_Magnitude_Count(n)
			}},
		{
			// One below zero is the first count a caller can spoil a magnitude with.
			Limit: -TEST_COUNT_ONE,
			Resize: func(s *execution_storage, n int) {
				s.Float_Text.Magnitude.Count = big.Float_Text_Magnitude_Count(n)
			}},
	}
}

// Sets every caller cursor and the stale diagnostic to one encoding.
func hostile_cursors(s *execution_storage, n int) {
	s.Workspace.Literal_Count.Value = template.Workspace_Literal_Count_Stored(n)
	s.Workspace.Generated_Count.Value = template.Workspace_Generated_Count_Stored(n)
	s.Workspace.Frame_Count.Value = template.Workspace_Frame_Count_Stored(n)
	s.Workspace.Variable_Count.Value = template.Workspace_Variable_Count_Stored(n)
	s.Workspace.Scope_Base.Value = template.Workspace_Scope_Base_Stored(n)
	s.Workspace.Diagnostic.Value.Position = template.Diagnostic_Position(n)
	s.Workspace.Diagnostic.Value.Code = template.Status(n)
}

// Resizes every float formatter decimal arena and drops its control scratch.
func hostile_digits(s *execution_storage, n int) {
	s.Float_Text.Decimals.Value.Digits = make(big.Float_Text_Decimal_Digits, n)
	s.Float_Text.Decimals.Lower.Digits = make(big.Float_Text_Decimal_Digits, n)
	s.Float_Text.Decimals.Upper.Digits = make(big.Float_Text_Decimal_Digits, n)
	s.Float_Text.Control = nil
}

type syntax_target struct {
	Workspace *template.Syntax_Workspace
	Number    *big.Float_Parse_Workspace
}

type syntax_mutation struct {
	Limit  int
	Resize func(target syntax_target, size int)
}

// Test_Hostile_Syntax_Workspaces refuses each caller parser arena at its first unsupported size.
func Test_Hostile_Syntax_Workspaces(t *testing.T) {
	for _, mutation := range syntax_mutations() {
		sizes := [...]int{TEST_COUNT_ZERO, TEST_COUNT_ONE, TEST_COUNT_TWO, mutation.Limit}
		for _, size := range sizes {
			workspace := &template.Syntax_Workspace{}
			input := syntax_from(workspace)
			unvalidated := (*template.Float_Parse_Unvalidated)(input.Number)
			number := (*big.Float_Parse_Workspace)(unvalidated)
			mutation.Resize(syntax_target{Workspace: workspace, Number: number}, size)
			template.Compile(template.Compile_Input{
				Source: []byte("x"), Workspace: input,
			})
		}
	}
}

func syntax_mutations() (mutations []syntax_mutation) {
	return []syntax_mutation{
		{
			Limit: template.NODE_COUNT_UNVALIDATED_MAXIMUM,
			Resize: func(target syntax_target, n int) {
				target.Workspace.Nodes = make(template.Nodes, n)
			},
		},
		{
			Limit: template.TEMPLATE_NAME_SIZE_UNVALIDATED_MAXIMUM,
			Resize: func(target syntax_target, n int) {
				right := make(template.Template_Name_Right, n)
				left := make(template.Template_Name_Left, n)
				target.Workspace.Template_Name_Left = left
				target.Workspace.Template_Name_Right = right
			},
		},
		{Limit: int(bits.WORD_16_MAXIMUM), Resize: func(target syntax_target, n int) {
			diagnostic := &target.Workspace.Parse_Diagnostic.Value
			diagnostic.Code = template.Diagnostic_Code(n)
			diagnostic.Position = template.Parse_Diagnostic_Position(n)
			target.Workspace.Node_Limit.Value = template.Node_Limit_Stored(n)
		}},
		{
			Limit: template.FLOAT_PARSE_SOURCE_COUNT_UNVALIDATED_MAXIMUM,
			Resize: func(target syntax_target, n int) {
				target.Number.Source = make(big.Float_Parse_Source, n)
			},
		},
		{
			Limit: template.FLOAT_PARSE_CONTROL_COUNT_UNVALIDATED_MAXIMUM,
			Resize: func(target syntax_target, n int) {
				target.Number.Control = make(big.Float_Parse_Control, n)
			},
		},
	}
}

// Each integer fits 64 unsigned bits yet leaves the signed range.
func Test_Signed_Overflow(t *testing.T) {
	t.Parallel()
	sources := [...]string{
		`{{9223372036854775808}}`, `{{-9223372036854775809}}`, `{{18446744073709551615}}`,
		`{{-1.7976931348623157e308}}`,
		`{{-1.7976931348623157e308-1.7976931348623157e308i}}`,
		`{{lt 1 .}}`, `{{index "ab" .}}`, `{{slice "ab" .}}`,
	}
	widest := value_complex(template.Complex{
		Imaginary: template.Complex_Imaginary(bits.WORD_64_MAXIMUM),
	})
	dots := [...]template.Value{
		value_unsigned(template.Unsigned(bits.WORD_64_MAXIMUM)), widest, widest,
	}
	var execution execution_storage
	for index, source := range sources {
		var workspace template.Syntax_Workspace
		program, _, status := template.Compile(template.Compile_Input{
			Source: []byte(source), Workspace: syntax_from(&workspace),
		})
		if status != template.PARSE_STATUS_OK {
			continue
		}
		dot := value_nil()
		if index >= len(sources)-len(dots) {
			dot = dots[index-len(sources)+len(dots)]
		}
		execute_shared_from(t, &execution, program, dot, nil)
	}
}

// Each source puts one count at its one or two witness.
func Test_Small_Witnesses(t *testing.T) {
	t.Parallel()
	sources := [...]string{
		`{{12i}}`, `{{$a := .}}{{$b := .}}{{range .}}x{{end}}`,
		`{{$a := .}}{{range .}}{{break}}{{end}}`,
		`{{$a := .}}{{define "t"}}{{range .}}{{break}}{{end}}{{end}}{{template "t" .}}`,
		`{{define "t"}}{{range .}}{{break}}{{end}}{{end}}{{template "t" .}}`,
		`{{range .}}{{range .}}{{break}}{{end}}{{end}}`,
		`{{with .}}{{range .}}{{break}}{{end}}{{end}}`,
		`{{$a := .}}{{$b := .}}{{define "t"}}{{range .}}{{continue}}{{end}}{{end}}{{template "t" .}}`,
		`{{$a := .}}{{$b := .}}{{if .}}{{else}}x{{end}}`,
		`{{$a := .}}{{$b := .}}{{with .}}{{else}}x{{end}}`,
		`{{"a"}}{{"b"}}{{define "t"}}x{{end}}{{template "t" .}}`,
		`x{{define "t"}}y{{end}}{{print 1}}`,
		`{{define "a"}}{{end}}{{define "b"}}{{end}}{{print 1}}`,
		`x{{define "t"}}y{{end}}{{$v := 1}}{{$v}}`, `x{{define "t"}}y{{end}}{{(.).a}}`,
		`{{(.).a . .}}`, `{{print 1 2 (.).a}}`, `{{print 1 2 (print (print 3))}}`,
		`{{print (print 1) (print 2) (print 3)}}`, `{{printf "a%T" 1}}`,
		`{{printf "ab%T" 1}}`, `{{printf "ab%q" "x"}}`, `{{printf "%q" "a\x01"}}`,
		`{{define "t"}}{{end}}{{printf "a"}}{{template "t" .}}`,
		`{{define "t"}}{{end}}{{printf "ab"}}{{template "t" .}}`, `{{print (print (print 1))}}`, `{{print 1 (print 2 (print 3))}}`,
		`{{index "ab" .}}`, `{{slice "ab" .}}`, `x{{define "t"}}y{{end}}{{template "t" .}}`,
		`{{define "a"}}{{end}}{{define "b"}}{{end}}{{template "b" .}}`,
		`x{{define "t"}}y{{end}}{{if .}}{{.a}}{{"s"}}{{range .}}{{end}}{{end}}{{identity 1}}`,
		`{{define "a"}}{{end}}{{define "b"}}{{end}}{{identity 1}}`,
		`{{define "a"}}{{end}}{{define "b"}}{{end}}{{define "b"}}y{{end}}`,
		`x{{define "t"}}y{{end}}{{eq identity 1}}`,
		`{{define "a"}}{{end}}{{define "b"}}{{end}}{{eq identity 1}}`,
	}
	functions := [...]template.Function{{Name: []byte("identity"), Call: identity}}
	items := [...]template.Value{value_text(nil), value_text(nil)}
	entries := [...]template.Field{{
		Key:   template.Field_Key{Data: value_text([]byte("a"))},
		Value: template.Field_Value{Data: value_integer(TEST_COUNT_ONE)},
	}}
	dots := [...]template.Value{
		value_nil(), value_sequence(items[:]), value_map(entries[:]),
	}
	var execution execution_storage
	for _, source := range sources {
		var workspace template.Syntax_Workspace
		program, _, status := template.Compile(template.Compile_Input{
			Source: []byte(source), Workspace: syntax_from(&workspace),
			Mode: template.SKIP_FUNCTION_CHECK,
		})
		if status != template.PARSE_STATUS_OK {
			continue
		}
		for _, dot := range dots {
			execute_shared_from(t, &execution, program, dot, functions[:])
		}
	}
}

// One break under a branch, outside every range, unwinds past two frames.
func Test_Stray_Break(t *testing.T) {
	t.Parallel()
	program := compile_from(t, "{{if .}}x{{end}}")
	nodes := (*program.Syntax.Value)[:program.Document.Node_Count]
	for index := range nodes {
		if nodes[index].Kind == template.NODE_TEXT {
			nodes[index].Kind = template.NODE_BREAK
		}
	}
	_, status := execute_from(t, program, value_text([]byte("a")))
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)
}

// Printing unordered keys scans for each next key.
func Test_Unsorted_Map_Print(t *testing.T) {
	t.Parallel()
	entries := [...]template.Field{
		{Key: template.Field_Key{Data: value_text([]byte("b"))}},
		{Key: template.Field_Key{Data: value_text([]byte("a"))}},
		{Key: template.Field_Key{Data: value_text([]byte("c"))}},
	}
	output, status := execute_from(t, compile_from(t, "{{.}}"), value_map(entries[:]))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "map[a:<no value> b:<no value> c:<no value>]", output)
}

// A template name followed by a stray closer rejects its argument.
func Test_Invocation_Argument_Syntax(t *testing.T) {
	t.Parallel()
	for _, source := range [...]string{`{{template "t")}}`, `{{template "t" |}}`} {
		var workspace template.Syntax_Workspace
		_, _, status := template.Compile(template.Compile_Input{
			Source: []byte(source), Workspace: syntax_from(&workspace),
		})
		testify.Equal_Values(t, template.PARSE_STATUS_SYNTAX_INVALID, status)
	}
}

// Each term opens one byte after the latest token, against the right delimiter.
func Test_Wide_Last_Start(t *testing.T) {
	t.Parallel()
	for _, action := range [...]string{`<(1>`, `<("">`, `<(>`, `<(">`, `<($>`, `<x $a:=1>`, `<x $:=1>`, `<($:=1>`, `<x`, `<1`, `<(`, `<x(>`, `<x(`, `<$=(>`, `<$=(.)>`, `<x|(>`, `<((>`, `<x $=1>`, `<($=1>`, `<$=1>`,
		`<$,$=1>`, `<$=>`, `<x $=>`, `<$,$=>`, `<range $=>`, `<($=>`, `<$:=>`, `<$:>`, `<x:>`, `<:>`} {
		padded := make([]byte, template.SOURCE_SIZE_MAXIMUM-len(action))
		for index := range padded {
			padded[index] = 'x'
		}
		var workspace template.Syntax_Workspace
		template.Compile(template.Compile_Input{
			Source: append(padded, action...), Left: []byte("<"), Right: []byte(">"),
			Workspace: syntax_from(&workspace),
		})
	}
}

// Each tail opens a template action under the deepest control nesting, or reads a variable
// after the most declarations one source holds.
func Test_Deep_Template_Actions(t *testing.T) {
	t.Parallel()
	opened, declared, parentheses, closed := "", "", "", ""
	for range template.ACTION_CONTENT_SIZE_MAXIMUM {
		parentheses += "("
	}
	// The deepest block that still fits every end it owes.
	for range template.TEMPLATE_DEFINITION_TRACKED_VALUE_MAXIMUM + TEST_COUNT_ONE {
		closed += `<end>`
	}
	for range template.CONTROL_DEPTH_MAXIMUM {
		opened += `<if.>`
	}
	for range template.SYNTAX_VARIABLE_COUNT_MAXIMUM {
		declared += `<$:=.>`
	}
	deepest := opened[:len(`<if.>`)*template.TEMPLATE_DEFINITION_TRACKED_VALUE_MAXIMUM] +
		`<block "a".>` + closed
	for _, source := range [...]string{
		opened + `<template "a">`, opened + `<define "a">`, opened + `<block "a".>`, opened + `<if.>`,
		opened[len(`<if.>`):] + `<block "a".>`, `<if.><if.><block "a".><end><end><end>`,
		deepest, `<block "">`,
		declared + `<$>`, `<if.><if.>`, `<$=.>`, `<=>`, `<$=>`, `<:>`, `<:=>`,
		"<" + parentheses[:template.PARENTHESIS_DEPTH_MAXIMUM] + ">",
		"<" + parentheses[:template.PARENTHESIS_DEPTH_MAXIMUM+TEST_COUNT_ONE] + ">",
		"<" + parentheses + ">",
	} {
		var workspace template.Syntax_Workspace
		template.Compile(template.Compile_Input{
			Source: []byte(source), Left: []byte("<"), Right: []byte(">"),
			Workspace: syntax_from(&workspace),
		})
	}
}

// Each number fills the whole action between one-byte delimiters.
func Test_Widest_Numbers(t *testing.T) {
	t.Parallel()
	digits := make([]byte, template.ACTION_CONTENT_SIZE_MAXIMUM)
	for index := range digits {
		digits[index] = '0'
	}
	imaginary := string(digits[:len(digits)-TEST_COUNT_ONE]) + "i"
	var execution execution_storage
	for _, source := range [...]string{`<` + string(digits) + `>`, `<` + imaginary + `>`} {
		var workspace template.Syntax_Workspace
		program, _, status := template.Compile(template.Compile_Input{
			Source: []byte(source), Left: []byte("<"), Right: []byte(">"),
			Workspace: syntax_from(&workspace),
		})
		testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
		execute_shared_from(t, &execution, program, value_nil(), nil)
	}
}

// Each malformed byte decodes to three, so few literals fill the decoded literal staging.
func Test_Malformed_Literals(t *testing.T) {
	t.Parallel()
	malformed := make([]byte, template.LITERAL_TEXT_SIZE_MAXIMUM/utf8.CHARACTER_SIZE_THREE)
	for index := range malformed {
		malformed[index] = 0xff
	}
	filled := `<"` + string(malformed) + `">`
	var execution execution_storage
	for _, source := range [...]string{
		filled + `<"">`, filled + `<'a'>`, filled + `<if 1><end>`, filled + filled,
		`<define ""><end>` + filled + `<template "">`,
	} {
		var workspace template.Syntax_Workspace
		program, _, status := template.Compile(template.Compile_Input{
			Source: []byte(source), Left: []byte("<"), Right: []byte(">"),
			Workspace: syntax_from(&workspace),
		})
		if status != template.PARSE_STATUS_OK {
			continue
		}
		execute_shared_from(t, &execution, program, value_nil(), nil)
	}
}

// Each recursion level holds four ranges, so its break and its return run at the deepest frames.
func Test_Deep_Range_Recursion(t *testing.T) {
	t.Parallel()
	// Eleven branches at the base fill the frame stack to its last slot.
	opened, closed := "", ""
	for index := TEST_COUNT_ZERO; index < TEST_COUNT_TEN+TEST_COUNT_ONE; index++ {
		opened += "{{with 1}}"
		closed += "{{end}}"
	}
	bases := [...]string{"{{1}}{{break}}", "{{template \"b\"}}", "{{with 1}}{{end}}"}
	items := make([]template.Value, template.FRAME_COUNT_MAXIMUM/TEST_COUNT_TEN-TEST_COUNT_TWO)
	for index := range items {
		items[index] = value_nil()
	}
	var execution execution_storage
	for _, base := range bases {
		program := compile_from(t, "{{define \"b\"}}{{end}}"+
			"{{define \"a\"}}{{range 1}}{{range 1}}{{range 1}}{{range 1}}"+
			"{{if $}}{{1}}{{template \"a\" (slice $ 1)}}{{else}}"+opened+base+closed+
			"{{end}}{{break}}{{end}}{{end}}{{end}}{{end}}{{end}}{{template \"a\" .}}")
		execute_shared_from(t, &execution, program, value_sequence(items), nil)
	}
	// One root variable per level fills the variable storage where the deepest level breaks.
	program := compile_from(t, "{{define \"c\"}}{{range $}}{{if slice $ 1}}"+
		"{{template \"c\" (slice $ 1)}}{{end}}{{break}}{{end}}{{end}}{{template \"c\" .}}")
	variables := make([]template.Value, template.VARIABLE_COUNT_MAXIMUM-TEST_COUNT_ONE)
	for index := range variables {
		variables[index] = value_nil()
	}
	execute_shared_from(t, &execution, program, value_sequence(variables), nil)
}

// The widest sequence of empty texts reaches its last element before the output bound closes it.
func Test_Widest_Sequence_Output(t *testing.T) {
	t.Parallel()
	items := make([]template.Value, template.VALUE_COUNT_MAXIMUM)
	for index := range items {
		items[index] = value_text(nil)
	}
	_, status := execute_from(t, compile_from(t, "{{.}}"), value_sequence(items))
	testify.Equal_Values(t, template.STATUS_OUTPUT_TOO_LARGE, status)
}

// The deepest parentheses one source holds wrap one chain, so its push stands on every frame.
func Test_Deepest_Chain(t *testing.T) {
	t.Parallel()
	depth := (template.SOURCE_SIZE_MAXIMUM - len("<(.).a>")) / len("()")
	source := make([]byte, TEST_COUNT_ZERO, template.SOURCE_SIZE_MAXIMUM)
	source = append(source, '<')
	for index := TEST_COUNT_ZERO; index < depth; index++ {
		source = append(source, '(')
	}
	source = append(source, "(.).a"...)
	for index := TEST_COUNT_ZERO; index < depth; index++ {
		source = append(source, ')')
	}
	source = append(source, '>')
	var workspace template.Syntax_Workspace
	program, _, status := template.Compile(template.Compile_Input{
		Source: source, Left: []byte("<"), Right: []byte(">"),
		Workspace: syntax_from(&workspace),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	fields := [...]template.Field{{
		Name: []byte("a"), Value: template.Field_Value{Data: value_text(nil)},
	}}
	var execution execution_storage
	execute_shared_from(t, &execution, program, value_object(fields[:]), nil)
}
