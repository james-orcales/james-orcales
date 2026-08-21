package template_test

import (
	"testing"
	standard_template "text/template"
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
	testify.Equal(t, "hello {{.Name}}", string(
		program.Source.Data[template.SOURCE_FIELD],
	))
	var workspace template.Syntax_Workspace
	var input template.Syntax_Workspace_Input
	input.State[template.SYNTAX_WORKSPACE_FIELD] = &workspace
	source, source_status := template.Source_Validate([]byte("{{.}}"))
	testify.Equal_Values(t, template.PARSE_STATUS_OK, source_status)
	configuration, configuration_status := template.New_Configuration(
		template.Configuration_Input{},
	)
	testify.Equal_Values(t, template.PARSE_STATUS_OK, configuration_status)
	_, _, parse_status := template.Parse_Into(source, configuration, input)
	testify.Equal_Values(t, template.PARSE_STATUS_OK, parse_status)
	test_compile_input_boundaries(t)
	test_compile_diagnostic_boundaries(t)
}

// Test_Values prevents reflection and ownership from entering value conversion.
func Test_Values(t *testing.T) {
	value := template.Value_Of_Text([]byte("Ada"))
	text, status := template.Value_As_Text(value)
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "Ada", string(text))

	_, status = template.Value_As_Text(template.Value_Of_Integer(TEST_COUNT_ONE))
	testify.Equal_Values(t, template.STATUS_VALUE_INVALID, status)
	template.Value_Of_Unsigned(TEST_COUNT_ONE)
	test_value_boundaries(t)
}

// Test_Execute keeps syntax, value, and function paths interoperable.
func Test_Execute(t *testing.T) {
	test_execute_fields(t)
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
	test_formulas(t)
	input := template.Compile_Input{Source: []byte("x")}
	_, _, status := template.Compile(input)
	testify.Equal_Values(t, template.PARSE_STATUS_WORKSPACE_INVALID, status)

	program := compile_from(t, "hello")
	var empty template.Workspace
	var empty_input template.Workspace_Input
	empty_input.State[template.WORKSPACE_FIELD] = &empty
	_, _, execution_status := template.Execute_Into(template.Execute_Input{
		Program: program, Value: template.Value_Nil(), Workspace: empty_input,
	})
	testify.Equal_Values(t, template.STATUS_WORKSPACE_INVALID, execution_status)

	program.Document.Root = template.Document_Root(template.NO_NODE)
	var execution execution_storage
	var output [template.OUTPUT_SIZE_MAXIMUM]byte
	_, _, execution_status = template.Execute_Into(template.Execute_Input{
		Destination: output[:], Program: program, Value: template.Value_Nil(),
		Workspace: workspace_from(&execution),
	})
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, execution_status)
	execution = execution_storage{}
	_, _, execution_status = template.Execute_Into(template.Execute_Input{
		Program: template.Program{}, Value: template.Value_Nil(),
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
		Destination: short[:], Program: program, Value: template.Value_Nil(),
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
		Value: template.Field_Value{Data: template.Value_Of_Text([]byte("Ada"))},
	}}
	root := template.Value_Of_Object(fields[:])
	var execution execution_storage
	var output [template.OUTPUT_SIZE_MAXIMUM]byte
	input := template.Execute_Input{
		Destination: output[:], Program: program, Value: root,
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
const TEST_COUNT_ONE = 1
const TEST_COUNT_TWO = TEST_COUNT_ONE + TEST_COUNT_ONE
const TEST_COUNT_THREE = TEST_COUNT_TWO + TEST_COUNT_ONE
const TEST_COUNT_FOUR = TEST_COUNT_TWO * TEST_COUNT_TWO
const TEST_COUNT_FIVE = TEST_COUNT_FOUR + TEST_COUNT_ONE
const TEST_COUNT_SEVEN = strconv.DECIMAL_BASE - TEST_COUNT_THREE
const TEST_COUNT_TEN = strconv.DECIMAL_BASE
const TEST_COUNT_THIRTEEN = TEST_COUNT_TEN + TEST_COUNT_THREE
const TEST_COUNT_FOURTEEN = TEST_COUNT_TEN + TEST_COUNT_FOUR
const TEST_COUNT_TWENTY_TWO = TEST_COUNT_TWO*TEST_COUNT_TEN + TEST_COUNT_TWO

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

type standard_output struct {
	Data  [template.OUTPUT_SIZE_MAXIMUM]byte
	Count int
}

func (output *standard_output) Write(source []byte) (count int, failure error) {
	count = copy(output.Data[output.Count:], source)
	output.Count += count
	if count != len(source) {
		return count, standard_output_full{}
	}
	return count, nil
}

type standard_output_full struct{}

func (standard_output_full) Error() (message string) {
	return "standard output full"
}

func test_scalar_value_allocation(t *testing.T) {
	t.Helper()
	var value template.Value
	testify.Zero_Allocation(t, func() {
		value = template.Value_Nil()
	})
	testify.Zero_Allocation(t, func() {
		value = template.Value_Of_Boolean(true)
	})
	testify.Zero_Allocation(t, func() {
		value = template.Value_Of_Boolean(false)
	})
	testify.Zero_Allocation(t, func() {
		value = template.Value_Of_Integer(template.Integer(bits.INTEGER_64_MINIMUM))
	})
	testify.Zero_Allocation(t, func() {
		value = template.Value_Of_Unsigned(template.Unsigned(bits.WORD_64_MAXIMUM))
	})
	testify.Zero_Allocation(t, func() {
		value = template.Value_Of_Float(template.Float(bits.WORD_64_MAXIMUM))
	})
	complex_value := template.Complex{
		Real:      template.Complex_Real(bits.WORD_64_MINIMUM),
		Imaginary: template.Complex_Imaginary(bits.WORD_64_MAXIMUM),
	}
	testify.Zero_Allocation(t, func() {
		value = template.Value_Of_Complex(complex_value)
	})
	testify.Equal_Values(t, template.VALUE_COMPLEX, value.Kind[template.VALUE_PAYLOAD_FIELD])
}

func test_borrowed_value_allocation(t *testing.T) {
	t.Helper()
	text := template.Text("text")
	values := [...]template.Value{template.Value_Nil()}
	fields := [...]template.Field{{Name: []byte("Name")}}
	var value template.Value
	testify.Zero_Allocation(t, func() {
		value = template.Value_Of_Text(text)
	})
	testify.Zero_Allocation(t, func() {
		value = template.Value_Of_Sequence(values[:])
	})
	testify.Zero_Allocation(t, func() {
		value = template.Value_Of_Object(fields[:])
	})
	testify.Zero_Allocation(t, func() {
		value = template.Value_Of_Map(fields[:])
	})
	testify.Zero_Allocation(t, func() {
		value = template.Value_Of_Function(identity)
	})
	text_value := template.Value_Of_Text(text)
	var extracted template.Text
	var status template.Value_Text_Status
	testify.Zero_Allocation(t, func() {
		extracted, status = template.Value_As_Text(text_value)
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
	items := [...]template.Value{
		template.Value_Of_Integer(TEST_COUNT_ONE),
		template.Value_Of_Integer(TEST_COUNT_TWO),
	}
	fields := [...]template.Field{
		{
			Name:  []byte("Ready"),
			Value: template.Field_Value{Data: template.Value_Of_Boolean(true)},
		},
		{
			Name: []byte("Items"),
			Value: template.Field_Value{
				Data: template.Value_Of_Sequence(items[:]),
			},
		},
	}
	test_execution_allocation(
		t, `{{if .Ready}}{{range .Items}}{{.}}{{else}}empty{{end}}{{else}}no{{end}}`,
		template.Value_Of_Object(fields[:]), nil, "12",
	)
	entries := [...]template.Field{
		{
			Key: template.Field_Key{Data: template.Value_Of_Text([]byte("b"))},
			Value: template.Field_Value{
				Data: template.Value_Of_Integer(TEST_COUNT_TWO),
			},
		},
		{
			Key: template.Field_Key{Data: template.Value_Of_Text([]byte("a"))},
			Value: template.Field_Value{
				Data: template.Value_Of_Integer(TEST_COUNT_ONE),
			},
		},
	}
	test_execution_allocation(
		t, `{{range $key, $value := .}}{{$key}}={{$value}};{{end}}`,
		template.Value_Of_Map(entries[:]), nil, "a=1;b=2;",
	)
	test_execution_allocation(
		t, `{{range .}}{{.}}{{else}}empty{{end}}`,
		template.Value_Of_Integer(-TEST_COUNT_ONE), nil, "empty",
	)
	capacity_values := [...]template.Value{
		template.Value_Of_Integer(TEST_COUNT_THREE),
		template.Value_Of_Integer(TEST_COUNT_FOUR),
		template.Value_Of_Integer(TEST_COUNT_FIVE),
		template.Value_Of_Integer(TEST_COUNT_ZERO),
		template.Value_Of_Integer(TEST_COUNT_ZERO),
	}
	test_execution_allocation(
		t, `{{slice . 3 5}}`, template.Value_Of_Sequence(capacity_values[:TEST_COUNT_THREE]),
		nil, "[0 0]",
	)
	test_execution_allocation(
		t, `{{template "named" .}}{{define "named"}}value{{end}}`,
		template.Value_Nil(), nil, "value",
	)
	functions := [...]template.Function{{Name: []byte("identity"), Call: identity}}
	test_execution_allocation(
		t, `{{identity .}}`, template.Value_Of_Text([]byte("value")),
		functions[:], "value",
	)
}

func test_execution_allocation(
	t *testing.T, source string, value template.Value,
	functions template.Functions, expected string,
) {
	t.Helper()
	program := compile_from(t, source)
	var execution execution_storage
	var output [template.OUTPUT_SIZE_MAXIMUM]byte
	input := template.Execute_Input{
		Destination: output[:], Program: program, Value: value,
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
	_, status := template.Value_As_Text(value)
	testify.Equal_Values(t, template.VALUE_TEXT_STATUS_OK, status)
	value = template.Value_Of_Text(text[:])
	result, status := template.Value_As_Text(value)
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
		big.WORD_COUNT_INCREMENT+big.WORD_COUNT_INCREMENT,
		big.ROUND_TO_NEAREST_EVEN,
	)
	test_workspace_finite_float(
		t, big.FLOAT_PRECISION_MAXIMUM, TEST_COUNT_ZERO, big.WORD_COUNT_MAXIMUM,
		big.ROUND_TO_NEAREST_EVEN,
	)
}

func test_workspace_finite_float(
	t *testing.T, precision int, exponent int, count int,
	mode big.Rounding_Mode_Unvalidated,
) {
	t.Helper()
	value := big.Float{
		Precision: big.Float_Precision(precision),
		Mode:      big.Rounding_Mode(mode),
		Accuracy:  big.ACCURACY_EXACT,
		Form:      big.FLOAT_FORM_FINITE,
		Exponent:  big.Float_Exponent(exponent),
	}
	value.Mantissa.Count = big.Word_Count(count)
	value.Mantissa.Words[count-big.WORD_COUNT_INCREMENT] = TEST_COUNT_ONE
	test_workspace_float(t, value)
}

func test_workspace_float(t *testing.T, value big.Float) {
	t.Helper()
	program := compile_from(t, "")
	var execution execution_storage
	execution.Float_Value = value
	_, _, status := template.Execute_Into(template.Execute_Input{
		Program: program, Value: template.Value_Nil(),
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
			Destination: destination[:size], Program: empty,
			Value: template.Value_Nil(), Workspace: workspace_from(&execution),
		})
		testify.Equal_Values(t, template.STATUS_OK, status)
		testify.Equal_Values(t, template.Output_Count(template.OUTPUT_SIZE_MINIMUM), count)
	}

	program := compile_from(t, "x")
	var two [TEST_COUNT_TWO]template.Function
	var execution execution_storage
	_, _, status := template.Execute_Into(template.Execute_Input{
		Program: program, Value: template.Value_Nil(), Functions: two[:],
		Workspace: workspace_from(&execution),
	})
	testify.Equal_Values(t, template.STATUS_FUNCTION_INVALID, status)
	var maximum [template.FUNCTION_COUNT_UNVALIDATED_MAXIMUM]template.Function
	execution = execution_storage{}
	_, _, status = template.Execute_Into(template.Execute_Input{
		Program: program, Value: template.Value_Nil(), Functions: maximum[:],
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
		Destination: destination[:], Program: program, Value: template.Value_Nil(),
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
		Destination: output[:], Program: program, Value: template.Value_Nil(),
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
			Program: program, Value: template.Value_Of_Integer(TEST_COUNT_ONE),
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
		Program: program, Value: template.Value_Nil(),
		Workspace: workspace_from(&execution),
	})
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)
	testify.Equal_Values(
		t, template.Diagnostic_Position(template.SOURCE_SIZE_MAXIMUM),
		diagnostic.Position,
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
			template.Node_Limit(template.NODE_COUNT_INCREMENT),
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
	_, status := execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)

	program = compile_from(t, "{{.}}")
	root = syntax_node(program, template.Node_Reference(program.Document.Root))
	action := syntax_node(program, template.Node_Reference(root.First_Child))
	action.First_Child = template.Node_First_Child(template.NO_NODE)
	_, status = execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)

	program = compile_from(t, "x")
	root = syntax_node(program, template.Node_Reference(program.Document.Root))
	text = syntax_node(program, template.Node_Reference(root.First_Child))
	text.Kind = template.NODE_BREAK
	_, status = execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)

	program = compile_from(t, "{{.}}")
	pipe := syntax_first_pipe(program)
	command := syntax_node(program, template.Node_Reference(pipe.First_Child))
	command.Kind = template.NODE_TEXT
	_, status = execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)

	program = compile_from(t, "{{.}}")
	pipe = syntax_first_pipe(program)
	pipe.First_Child = template.Node_First_Child(template.NO_NODE)
	_, status = execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)

	program = compile_from(t, `{{$x := 1}}{{$x = 2}}`)
	data := program.Source.Data[template.SOURCE_FIELD]
	data[len(`{{$x := 1}}{{$`)] = 'y'
	_, status = execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_EXECUTION_INVALID, status)
	program = compile_from(t, `{{$x := 1}}{{$x = 2}}{{$x}}`)
	output, status := execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "2", output)

	test_execute_child_program_bounds(t)
}

func test_execute_node_record_bounds(t *testing.T) {
	t.Helper()
	program := compile_from(t, "xx")
	_, status := execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_OK, status)

	program = compile_from(t, "x")
	root := syntax_node(program, template.Node_Reference(program.Document.Root))
	root.First_Child = template.Node_First_Child(template.NODE_REFERENCE_MAXIMUM)
	_, status = execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)

	program = compile_from(t, "{{.}}")
	root = syntax_node(program, template.Node_Reference(program.Document.Root))
	root.First_Child = template.Node_First_Child(template.ROOT_NODE_COUNT)
	_, status = execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)

	program = compile_from(t, "x")
	root = syntax_node(program, template.Node_Reference(program.Document.Root))
	text := syntax_node(program, template.Node_Reference(root.First_Child))
	text.Next_Sibling = template.Node_Next_Sibling(template.ROOT_NODE_COUNT)
	_, status = execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)

	program = compile_from(t, "{{if .}}x{{else}}y{{end}}")
	root = syntax_node(program, template.Node_Reference(program.Document.Root))
	control := syntax_node(program, template.Node_Reference(root.First_Child))
	pipe := syntax_node(program, template.Node_Reference(control.First_Child))
	body := syntax_node(program, template.Node_Reference(pipe.Next_Sibling))
	alternate := syntax_node(program, template.Node_Reference(body.Next_Sibling))
	alternate.Kind = template.NODE_TEXT
	_, status = execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)
}

func test_execute_lookup_span_bounds(t *testing.T) {
	t.Helper()
	for _, kind := range [...]template.Node_Kind{
		template.NODE_FIELD, template.NODE_VARIABLE,
	} {
		program := compile_from(t, "{{.}}")
		term := syntax_node(
			program,
			template.Node_Reference(syntax_first_pipe(program).First_Child),
		)
		term = syntax_node(program, template.Node_Reference(term.First_Child))
		term.Kind = kind
		term.Value_Start = template.Node_Value_Start(len("{{"))
		term.Value_End = template.Node_Value_End(
			len("{{") + template.SOURCE_POSITION_INCREMENT,
		)
		_, status := execute_from(t, program, template.Value_Nil())
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
		program.Source.Data[template.SOURCE_FIELD] = maximum[:]
		term.Value_Start = template.Node_Value_Start(template.SOURCE_SIZE_MINIMUM)
		term.Value_End = template.Node_Value_End(len(maximum))
		_, status = execute_from(t, program, template.Value_Nil())
		testify.Equal_Values(t, template.STATUS_EXECUTION_INVALID, status)
	}
}

func test_execute_child_program_bounds(t *testing.T) {
	program := compile_from(t, `{{$x := .}}{{($x)}}`)
	root := syntax_node(program, template.Node_Reference(program.Document.Root))
	declaration := syntax_node(program, template.Node_Reference(root.First_Child))
	declaration_pipe := syntax_node(
		program, template.Node_Reference(declaration.First_Child),
	)
	action := syntax_node(program, template.Node_Reference(declaration.Next_Sibling))
	pipe := syntax_node(program, template.Node_Reference(action.First_Child))
	command := syntax_node(program, template.Node_Reference(pipe.First_Child))
	inner_pipe := syntax_node(program, template.Node_Reference(command.First_Child))
	inner_pipe.First_Child = declaration_pipe.First_Child
	_, status := execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)

	program = compile_from(t, `{{(.).X}}`)
	_, status = execute_from(t, program, template.Value_Of_Integer(TEST_COUNT_ONE))
	testify.Equal_Values(t, template.STATUS_EXECUTION_INVALID, status)
	program = compile_from(t, `{{(.).X}}`)
	pipe = syntax_first_pipe(program)
	command = syntax_node(program, template.Node_Reference(pipe.First_Child))
	chain := syntax_node(program, template.Node_Reference(command.First_Child))
	chain.Value_Start = template.Node_Value_Start(chain.Value_End)
	_, status = execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, status)

	program = compile_from(t, `{{range .}}{{else}}x{{end}}`)
	root = syntax_node(program, template.Node_Reference(program.Document.Root))
	range_node := syntax_node(program, template.Node_Reference(root.First_Child))
	pipe = syntax_node(program, template.Node_Reference(range_node.First_Child))
	body := syntax_node(program, template.Node_Reference(pipe.Next_Sibling))
	alternate := syntax_node(program, template.Node_Reference(body.Next_Sibling))
	alternate.Kind = template.NODE_TEXT
	_, status = execute_from(
		t, program, template.Value_Of_Sequence([]template.Value{}),
	)
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
	inner_command := syntax_node(
		program, template.Node_Reference(inner_pipe.First_Child),
	)
	inner_command.First_Child = template.Node_First_Child(inner_pipe_reference)
	_, status := execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_LIMIT_EXCEEDED, status)

	program = compile_from(t, `{{print .}}`)
	pipe = syntax_first_pipe(program)
	command = syntax_node(program, template.Node_Reference(pipe.First_Child))
	identifier := syntax_node(program, template.Node_Reference(command.First_Child))
	argument_reference := template.Node_Reference(identifier.Next_Sibling)
	argument := syntax_node(program, argument_reference)
	argument.Next_Sibling = template.Node_Next_Sibling(argument_reference)
	_, status = execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_LIMIT_EXCEEDED, status)

	program = compile_from(
		t, `{{define "cycle"}}{{$x := .}}{{template "cycle" .}}{{end}}`+
			`{{template "cycle" .}}`,
	)
	_, status = execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_LIMIT_EXCEEDED, status)

	program = compile_from(t, "x")
	root := syntax_node(program, template.Node_Reference(program.Document.Root))
	text_reference := template.Node_Reference(root.First_Child)
	text := syntax_node(program, text_reference)
	text.Value_End = template.Node_Value_End(text.Value_Start)
	text.Next_Sibling = template.Node_Next_Sibling(text_reference)
	_, status = execute_from(t, program, template.Value_Nil())
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
	_, execution_status := execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, execution_status)

	source = []byte(`{{'a'}}`)
	syntax = template.Syntax_Workspace{}
	program, _, parse_status = template.Compile(template.Compile_Input{
		Source: source, Workspace: syntax_from(&syntax),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, parse_status)
	source[len("{{'")] = '\\'
	_, execution_status = execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, execution_status)

	source = []byte(`{{1}}`)
	syntax = template.Syntax_Workspace{}
	program, _, parse_status = template.Compile(template.Compile_Input{
		Source: source, Workspace: syntax_from(&syntax),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, parse_status)
	source[len("{{")] = 'x'
	_, execution_status = execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_EXECUTION_INVALID, execution_status)

	source = []byte(`{{1+2i}}`)
	syntax = template.Syntax_Workspace{}
	program, _, parse_status = template.Compile(template.Compile_Input{
		Source: source, Workspace: syntax_from(&syntax),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, parse_status)
	source[len("{{1")] = 'i'
	_, execution_status = execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_EXECUTION_INVALID, execution_status)

	source = []byte(`{{$x := 1}}`)
	syntax = template.Syntax_Workspace{}
	program, _, parse_status = template.Compile(template.Compile_Input{
		Source: source, Workspace: syntax_from(&syntax),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, parse_status)
	source[len("{{$x ")] = '?'
	_, execution_status = execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, execution_status)
}

func test_execute_lookup_program_bounds(t *testing.T) {
	source := []byte(`{{template "x"}}`)
	var syntax template.Syntax_Workspace
	program, _, parse_status := template.Compile(template.Compile_Input{
		Source: source, Workspace: syntax_from(&syntax),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, parse_status)
	_, execution_status := execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_EXECUTION_INVALID, execution_status)
	source[len(`{{template "`)] = '\\'
	_, execution_status = execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, execution_status)

	source = []byte(`{{.Name}}`)
	syntax = template.Syntax_Workspace{}
	program, _, parse_status = template.Compile(template.Compile_Input{
		Source: source, Workspace: syntax_from(&syntax),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, parse_status)
	_, execution_status = execute_from(
		t, program, template.Value_Of_Integer(TEST_COUNT_ONE),
	)
	testify.Equal_Values(t, template.STATUS_EXECUTION_INVALID, execution_status)
	source[len("{{.")] = '.'
	_, execution_status = execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, execution_status)

	source = []byte(`{{$x := 1}}{{$x}}`)
	syntax = template.Syntax_Workspace{}
	program, _, parse_status = template.Compile(template.Compile_Input{
		Source: source, Workspace: syntax_from(&syntax),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, parse_status)
	source[len(source)-len("x}}")] = 'y'
	_, execution_status = execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_EXECUTION_INVALID, execution_status)

	source = []byte(`{{$x := .}}{{$x.Name}}`)
	syntax = template.Syntax_Workspace{}
	program, _, parse_status = template.Compile(template.Compile_Input{
		Source: source, Workspace: syntax_from(&syntax),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, parse_status)
	source[len(source)-len("Name}}")] = '.'
	_, execution_status = execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_PROGRAM_INVALID, execution_status)

	program = compile_from(
		t, `{{define "cycle"}}{{template "cycle"}}{{end}}`+
			`{{template "cycle"}}`,
	)
	_, execution_status = execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_LIMIT_EXCEEDED, execution_status)
}

func test_execute_value_bounds(t *testing.T) {
	program := compile_from(t, "x")
	invalid := template.Value_Nil()
	invalid.Kind[template.VALUE_PAYLOAD_FIELD] = template.VALUE_FUNCTION + TEST_COUNT_ONE
	_, execution_status := execute_from(t, program, invalid)
	testify.Equal_Values(t, template.STATUS_VALUE_INVALID, execution_status)

	var leaves, branches [template.VALUE_COUNT_MAXIMUM]template.Value
	branches[len(branches)-TEST_COUNT_ONE] = template.Value_Of_Sequence(leaves[:])
	_, execution_status = execute_from(
		t, program, template.Value_Of_Sequence(branches[:]),
	)
	testify.Equal_Values(t, template.STATUS_LIMIT_EXCEEDED, execution_status)

	cycle := [...]template.Value{template.Value_Nil()}
	cyclic := template.Value_Of_Sequence(cycle[:])
	cycle[template.VALUE_PAYLOAD_FIELD] = cyclic
	_, execution_status = execute_from(t, program, cyclic)
	testify.Equal_Values(t, template.STATUS_LIMIT_EXCEEDED, execution_status)

	invalid_key := [...]template.Value{template.Value_Nil()}
	entries := [...]template.Field{{
		Key: template.Field_Key{
			Data: template.Value_Of_Sequence(invalid_key[:]),
		},
		Value: template.Field_Value{Data: template.Value_Nil()},
	}}
	_, execution_status = execute_from(
		t, program, template.Value_Of_Map(entries[:]),
	)
	testify.Equal_Values(t, template.STATUS_VALUE_INVALID, execution_status)

	duplicate_fields := [...]template.Field{
		{Name: []byte("Name")},
		{Name: []byte("Name")},
	}
	_, execution_status = execute_from(
		t, program, template.Value_Of_Object(duplicate_fields[:]),
	)
	testify.Equal_Values(t, template.STATUS_VALUE_INVALID, execution_status)

	duplicate_entries := [...]template.Field{
		{Key: template.Field_Key{Data: template.Value_Of_Integer(TEST_COUNT_ONE)}},
		{Key: template.Field_Key{Data: template.Value_Of_Integer(TEST_COUNT_ONE)}},
	}
	_, execution_status = execute_from(
		t, program, template.Value_Of_Map(duplicate_entries[:]),
	)
	testify.Equal_Values(t, template.STATUS_VALUE_INVALID, execution_status)
}

func test_builtin_allocation(t *testing.T) {
	program := compile_from(
		t, `{{html "<&>"}}|{{printf "%s=%d" "x" 2}}|{{println "a" 2}}|{{.Items}}`,
	)
	items := [...]template.Value{
		template.Value_Of_Integer(TEST_COUNT_ONE),
		template.Value_Of_Integer(TEST_COUNT_TWO),
	}
	fields := [...]template.Field{{
		Name: []byte("Items"),
		Value: template.Field_Value{
			Data: template.Value_Of_Sequence(items[:]),
		},
	}}
	var execution execution_storage
	var output [template.OUTPUT_SIZE_MAXIMUM]byte
	input := template.Execute_Input{
		Destination: output[:], Program: program,
		Value:     template.Value_Of_Object(fields[:]),
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
	sequence := [...]template.Value{template.Value_Of_Integer(TEST_COUNT_ONE)}
	for _, value := range []template.Value{
		template.Value_Of_Text([]byte("x")),
		template.Value_Of_Boolean(true),
		template.Value_Of_Float(template.Float(FLOAT_ONE_BITS)),
		template.Value_Of_Complex(template.Complex{
			Real: template.Complex_Real(FLOAT_ONE_BITS),
		}),
		template.Value_Of_Sequence(sequence[:]),
	} {
		fields := [...]template.Field{
			{
				Name: []byte("Pad"),
				Value: template.Field_Value{
					Data: template.Value_Of_Text(pad[:]),
				},
			},
			{Name: []byte("Value"), Value: template.Field_Value{Data: value}},
		}
		_, status := execute_from(t, program, template.Value_Of_Object(fields[:]))
		testify.Equal_Values(t, template.STATUS_OUTPUT_TOO_LARGE, status)
	}

	program = compile_from(t, `{{.}}`)
	var text [template.OUTPUT_SIZE_MAXIMUM - TEST_COUNT_ONE]byte
	sequence = [...]template.Value{template.Value_Of_Text(text[:])}
	_, status := execute_from(t, program, template.Value_Of_Sequence(sequence[:]))
	testify.Equal_Values(t, template.STATUS_OUTPUT_TOO_LARGE, status)

	var key [template.OUTPUT_SIZE_MAXIMUM]byte
	entries := [...]template.Field{{
		Key: template.Field_Key{Data: template.Value_Of_Text(key[:])},
		Value: template.Field_Value{
			Data: template.Value_Of_Integer(TEST_COUNT_ONE),
		},
	}}
	_, status = execute_from(t, program, template.Value_Of_Map(entries[:]))
	testify.Equal_Values(t, template.STATUS_OUTPUT_TOO_LARGE, status)
}

func test_execute_generated_bounds(t *testing.T) {
	control_one := [...]byte{TEST_COUNT_ONE}
	output, status := execute_from(
		t, compile_from(t, `{{printf "%q" .}}`),
		template.Value_Of_Text(control_one[:]),
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
			Value: template.Value_Of_Text(
				ampersand[:template.OUTPUT_SIZE_MAXIMUM/len("&amp;")],
			),
		},
		{
			Name:   "type",
			Source: `{{html . | printf "%T"}}`,
			Value: template.Value_Of_Text(
				plain[:template.OUTPUT_SIZE_MAXIMUM-TEST_COUNT_ONE],
			),
		},
		{
			Name: "quote", Source: `{{printf "%q" .}}`,
			Value: template.Value_Of_Text(control[:]),
		},
		{
			Name: "html", Source: `{{html .}}`,
			Value: template.Value_Of_Text(ampersand[:]),
		},
		{
			Name: "javascript", Source: `{{js .}}`,
			Value: template.Value_Of_Text(control[:]),
		},
		{
			Name: "url query", Source: `{{urlquery .}}`,
			Value: template.Value_Of_Text(ampersand[:]),
		},
		{
			Name: "result", Source: `{{html .}}`,
			Value: template.Value_Of_Text(
				ampersand[:template.OUTPUT_SIZE_MAXIMUM/len("&amp;")+
					TEST_COUNT_ONE],
			),
		},
		{
			Name: "arena", Source: `{{html . | print}}`,
			Value: template.Value_Of_Text(plain[:]),
		},
	} {
		t.Run(one.Name, func(t *testing.T) {
			_, status = execute_from(
				t, compile_from(t, one.Source), one.Value,
			)
			testify.Equal_Values(t, template.STATUS_LIMIT_EXCEEDED, status)
		})
	}
}

func test_execute_fields(t *testing.T) {
	program := compile_from(t, "Hello {{.Name}}! {{if .Ready}}yes{{else}}no{{end}}")
	fields := [...]template.Field{
		{
			Name:  []byte("Name"),
			Value: template.Field_Value{Data: template.Value_Of_Text([]byte("Ada"))},
		},
		{
			Name:  []byte("Ready"),
			Value: template.Field_Value{Data: template.Value_Of_Boolean(true)},
		},
	}
	root := template.Value_Of_Object(fields[:])
	output, status := execute_from(t, program, root)
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
		template.Value_Of_Integer(TEST_COUNT_ONE),
		template.Value_Of_Integer(TEST_COUNT_TWO),
		template.Value_Of_Integer(TEST_COUNT_THREE),
		template.Value_Of_Integer(TEST_COUNT_FOUR),
	}
	fields := [...]template.Field{
		{
			Name:  []byte("User"),
			Value: template.Field_Value{Data: template.Value_Of_Text([]byte("Ada"))},
		},
		{
			Name:  []byte("Items"),
			Value: template.Field_Value{Data: template.Value_Of_Sequence(items[:])},
		},
	}
	output, status := execute_from(t, program, template.Value_Of_Object(fields[:]))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "Ada:0=1;2=3;", output)
	test_execute_range_values(t)
}

func test_execute_range_values(t *testing.T) {
	t.Helper()
	_, status := execute_from(
		t, compile_from(t, `{{range true}}{{end}}`), template.Value_Nil(),
	)
	testify.Equal_Values(t, template.STATUS_EXECUTION_INVALID, status)
	program := compile_from(t, `{{range .}}{{.}}{{end}}`)
	_, status = execute_from(
		t, program, template.Value_Of_Integer(
			template.Integer(template.VALUE_COUNT_UNVALIDATED_MAXIMUM),
		),
	)
	testify.Equal_Values(t, template.STATUS_LIMIT_EXCEEDED, status)
	output, status := execute_from(t, program, template.Value_Of_Unsigned(TEST_COUNT_TWO))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "01", output)
	output, status = execute_from(t, program, template.Value_Of_Integer(TEST_COUNT_TWO))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "01", output)

	one := [...]template.Value{template.Value_Of_Integer(TEST_COUNT_SEVEN)}
	program = compile_from(t, `{{range .}}{{end}}`)
	_, status = execute_from(t, program, template.Value_Of_Sequence(one[:]))
	testify.Equal_Values(t, template.STATUS_OK, status)
	var maximum [template.VALUE_COUNT_MAXIMUM]template.Value
	_, status = execute_from(t, program, template.Value_Of_Sequence(maximum[:]))
	testify.Equal_Values(t, template.STATUS_OK, status)

	program = compile_from(
		t, `{{$value := 0}}{{range $value = .}}{{end}}{{$value}}`,
	)
	output, status = execute_from(t, program, template.Value_Of_Sequence(one[:]))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "7", output)
	program = compile_from(
		t, `{{$key := 0}}{{$value := 0}}`+
			`{{range $key, $value = .}}{{end}}{{$key}}/{{$value}}`,
	)
	output, status = execute_from(t, program, template.Value_Of_Sequence(one[:]))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "0/7", output)
}

func test_execute_pipeline(t *testing.T) {
	program := compile_from(
		t, `{{$value := .Name | upper}}{{join "hello" $value}}`,
	)
	functions := [...]template.Function{
		{Name: []byte("upper"), Call: upper},
		{Name: []byte("join"), Call: join},
		{Name: []byte("identity"), Call: identity},
	}
	fields := [...]template.Field{{
		Name:  []byte("Name"),
		Value: template.Field_Value{Data: template.Value_Of_Text([]byte("ada"))},
	}}
	output, status := execute_functions_from(
		t, program, template.Value_Of_Object(fields[:]), functions[:],
	)
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "hello ADA", output)

	program = compile_from(t, `{{join "hello" (upper .Name)}}|{{call identity "ok"}}`)
	output, status = execute_functions_from(
		t, program, template.Value_Of_Object(fields[:]), functions[:],
	)
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "hello ADA|ok", output)

	program = compile_from(
		t, `{{$first := printf "%s" "first"}}`+
			`{{$second := printf "%s" "second"}}{{$first}}/{{$second}}`,
	)
	output, status = execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "first/second", output)

	program = compile_from(t, `{{call missing "ok"}}`)
	_, status = execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_FUNCTION_ABSENT, status)

	program = compile_from(t, `{{reject}}`)
	reject_function := [...]template.Function{{Name: []byte("reject"), Call: reject}}
	_, status = execute_functions_from(
		t, program, template.Value_Nil(), reject_function[:],
	)
	testify.Equal_Values(t, template.STATUS_FUNCTION_INVALID, status)

	program = compile_from(t, `{{invalid_nested}}`)
	invalid_function := [...]template.Function{{
		Name: []byte("invalid_nested"), Call: invalid_nested,
	}}
	_, status = execute_functions_from(
		t, program, template.Value_Nil(), invalid_function[:],
	)
	testify.Equal_Values(t, template.STATUS_FUNCTION_INVALID, status)
	test_variable_lookup_boundaries(t)
}

func test_variable_lookup_boundaries(t *testing.T) {
	t.Helper()
	program := compile_from(t, `{{$}}`)
	output, status := execute_from(t, program, template.Value_Of_Integer(TEST_COUNT_SEVEN))
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
	output, status = execute_from(t, program, template.Value_Nil())
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
			Name:  []byte("Name"),
			Value: template.Field_Value{Data: template.Value_Of_Text([]byte("Ada"))},
		},
		{
			Name:  []byte("Footer"),
			Value: template.Field_Value{Data: template.Value_Of_Text([]byte("done"))},
		},
	}
	output, status := execute_from(t, program, template.Value_Of_Object(fields[:]))
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
		template.Value_Of_Text([]byte("a")),
		template.Value_Of_Text([]byte("b")),
		template.Value_Of_Text([]byte("c")),
	}
	entries := [...]template.Field{{
		Key:   template.Field_Key{Data: template.Value_Of_Text([]byte("x"))},
		Value: template.Field_Value{Data: template.Value_Of_Integer(TEST_COUNT_SEVEN)},
	}}
	fields := [...]template.Field{
		{
			Name: []byte("Items"),
			Value: template.Field_Value{
				Data: template.Value_Of_Sequence(items[:]),
			},
		},
		{
			Name: []byte("Lookup"),
			Value: template.Field_Value{
				Data: template.Value_Of_Map(entries[:]),
			},
		},
		{
			Name: []byte("Identity"),
			Value: template.Field_Value{
				Data: template.Value_Of_Function(identity),
			},
		},
	}
	output, status := execute_from(
		t, program, template.Value_Of_Object(fields[:]),
	)
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(
		t, "0|1|true|true|true|true|true|true|true|false|true|b|7|2|ok", output,
	)
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
			fields[TEST_COUNT_ZERO].Value.Data = template.Value_Of_Text(texts[rotation])
			right := texts[(rotation+TEST_COUNT_ONE)%len(texts)]
			fields[TEST_COUNT_ONE].Value.Data = template.Value_Of_Text(right)
			_, status := execute_from(t, program, template.Value_Of_Object(fields[:]))
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
		_, status = execute_from(t, compile_from(t, source), template.Value_Nil())
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
		_, status = execute_from(t, compile_from(t, one.Source), template.Value_Nil())
		testify.Equal_Values(t, one.Status, status)
	}
	test_slice_capacity_refusal(t)
	test_index_slice_boundaries(t)
}

func test_index_slice_boundaries(t *testing.T) {
	t.Helper()
	var maximum [template.VALUE_COUNT_MAXIMUM]template.Value
	for index := range maximum {
		maximum[index] = template.Value_Of_Integer(template.Integer(index))
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
		fields[TEST_COUNT_ZERO].Value.Data = template.Value_Of_Sequence(one.Values)
		fields[TEST_COUNT_ONE].Value.Data = template.Value_Of_Integer(one.Index)
		_, status := execute_from(
			t, index_program, template.Value_Of_Object(fields[:]),
		)
		testify.Equal_Values(t, one.Status, status)
	}

	slice_program := compile_from(t, `{{len (slice .Values 0 .End)}}`)
	for _, count := range [...]int{
		TEST_COUNT_ZERO, TEST_COUNT_ONE, TEST_COUNT_TWO,
		template.VALUE_COUNT_MAXIMUM,
	} {
		fields[TEST_COUNT_ZERO].Value.Data = template.Value_Of_Sequence(
			maximum[:count:count],
		)
		fields[TEST_COUNT_TWO].Value.Data = template.Value_Of_Integer(
			template.Integer(count),
		)
		_, status := execute_from(
			t, slice_program, template.Value_Of_Object(fields[:]),
		)
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
				Data: template.Value_Of_Sequence(values[:TEST_COUNT_ONE]),
			},
		},
		{
			Name: []byte("End"),
			Value: template.Field_Value{
				Data: template.Value_Of_Integer(HOSTILE_COUNT),
			},
		},
	}
	program := compile_from(t, `{{slice .Values 0 .End}}`)
	var execution execution_storage
	var output [template.OUTPUT_SIZE_MAXIMUM]byte
	input := template.Execute_Input{
		Destination: output[:], Program: program,
		Value:     template.Value_Of_Object(fields[:]),
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
		t.Run(one.Source, func(t *testing.T) {
			program := compile_from(t, one.Source)
			output, status := execute_from(t, program, template.Value_Nil())
			testify.Equal_Values(t, template.STATUS_OK, status)
			testify.Equal(t, one.Want, output)
		})
	}
	test_execute_numeric_boundaries(t)
}

func test_execute_numeric_boundaries(t *testing.T) {
	t.Helper()
	output_program := compile_from(t, `{{.}}`)
	for _, value := range [...]template.Value{
		template.Value_Of_Float(template.Float(bits.WORD_64_MINIMUM + TEST_COUNT_ONE)),
		template.Value_Of_Float(template.Float(bits.WORD_64_MINIMUM + TEST_COUNT_TWO)),
		template.Value_Of_Float(template.Float(bits.WORD_64_MAXIMUM)),
		template.Value_Of_Complex(template.Complex{
			Real: TEST_COUNT_ONE, Imaginary: TEST_COUNT_ONE,
		}),
		template.Value_Of_Complex(template.Complex{
			Real: TEST_COUNT_TWO, Imaginary: TEST_COUNT_TWO,
		}),
		template.Value_Of_Complex(template.Complex{
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
		template.Value_Of_Integer(template.Integer(bits.INTEGER_64_MINIMUM)),
		template.Value_Of_Integer(template.Integer(-TEST_COUNT_ONE)),
		template.Value_Of_Integer(template.Integer(bits.INTEGER_64_MAXIMUM)),
	}
	for rotation := range integers {
		fields[TEST_COUNT_ZERO].Value.Data = integers[rotation]
		right := (rotation + TEST_COUNT_ONE) % len(integers)
		fields[TEST_COUNT_ONE].Value.Data = integers[right]
		_, status := execute_from(t, order_program, template.Value_Of_Object(fields[:]))
		testify.Equal_Values(t, template.STATUS_OK, status)
	}
	unsigned := [...]template.Value{
		template.Value_Of_Unsigned(template.Unsigned(bits.WORD_64_MINIMUM)),
		template.Value_Of_Unsigned(template.Unsigned(bits.WORD_64_MAXIMUM)),
	}
	for rotation := range unsigned {
		fields[TEST_COUNT_ZERO].Value.Data = unsigned[rotation]
		right := (rotation + TEST_COUNT_ONE) % len(unsigned)
		fields[TEST_COUNT_ONE].Value.Data = unsigned[right]
		_, status := execute_from(t, order_program, template.Value_Of_Object(fields[:]))
		testify.Equal_Values(t, template.STATUS_OK, status)
	}
	for _, integer := range [...]template.Integer{
		template.Integer(-TEST_COUNT_ONE), template.Integer(TEST_COUNT_ZERO),
		template.Integer(TEST_COUNT_ONE),
	} {
		fields[TEST_COUNT_ZERO].Value.Data = template.Value_Of_Integer(integer)
		fields[TEST_COUNT_ONE].Value.Data = template.Value_Of_Unsigned(TEST_COUNT_ZERO)
		_, status := execute_from(t, order_program, template.Value_Of_Object(fields[:]))
		testify.Equal_Values(t, template.STATUS_OK, status)
	}
}

func test_execute_map_order(t *testing.T) {
	program := compile_from(
		t, `{{range $key, $value := .}}{{$key}}{{$value}}{{end}}`,
	)
	entries := [...]template.Field{
		{
			Key: template.Field_Key{Data: template.Value_Of_Text([]byte("c"))},
			Value: template.Field_Value{
				Data: template.Value_Of_Integer(TEST_COUNT_THREE),
			},
		},
		{
			Key: template.Field_Key{Data: template.Value_Of_Text([]byte("a"))},
			Value: template.Field_Value{
				Data: template.Value_Of_Integer(TEST_COUNT_ONE),
			},
		},
		{
			Key: template.Field_Key{Data: template.Value_Of_Text([]byte("b"))},
			Value: template.Field_Value{
				Data: template.Value_Of_Integer(TEST_COUNT_TWO),
			},
		},
	}
	output, status := execute_from(t, program, template.Value_Of_Map(entries[:]))
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
				Data: template.Value_Of_Integer(template.Integer(key)),
			},
			Value: template.Field_Value{Data: item},
		}
	}
	program = compile_from(t, `{{range .}}{{if .}}{{break}}{{end}}{{end}}`)
	_, status = execute_from(t, program, template.Value_Of_Map(maximum[:]))
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
				Key:   template.Field_Key{Data: template.Value_Of_Float(encoding)},
				Value: template.Field_Value{Data: template.Value_Nil()},
			}
		}
		_, status := execute_from(t, program, template.Value_Of_Map(entries[:]))
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
	output, status := execute_from(t, program, template.Value_Nil())
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
		_, status = execute_from(t, compile_from(t, one.Source), template.Value_Nil())
		testify.Equal_Values(t, one.Status, status)
	}
	test_execute_format_boundaries(t)
}

func test_execute_format_boundaries(t *testing.T) {
	t.Helper()
	program := compile_from(
		t, `{{printf "%1d|%2d|%b|%o|%X" 1 2 2 8 255}}`,
	)
	output, status := execute_from(t, program, template.Value_Nil())
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "1| 2|10|10|FF", output)

	program = compile_from(t, `{{printf "%T|%T|%T" nil true .}}`)
	output, status = execute_from(
		t, program, template.Value_Of_Function(identity),
	)
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "<nil>|bool|template.Function", output)

	for _, source := range [...]string{
		`{{printf "%\x00" 1}}`, `{{printf "%\x01" 1}}`,
		`{{printf "%\x02" 1}}`, `{{printf "%\xff" 1}}`,
	} {
		_, status = execute_from(t, compile_from(t, source), template.Value_Nil())
		testify.Equal_Values(t, template.STATUS_EXECUTION_INVALID, status)
	}

	program = compile_from(t, "{{printf \"%#q\" \"a`b\"}}")
	output, status = execute_from(t, program, template.Value_Nil())
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
	output, status = execute_from(
		t, program, template.Value_Of_Text(quoted_controls[:]),
	)
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
			Data: template.Value_Of_Text(format[:]),
		},
	}}
	program = compile_from(t, `{{printf .Format}}`)
	output, status = execute_from(t, program, template.Value_Of_Object(fields[:]))
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
		_, status := execute_from(t, program, template.Value_Of_Text(texts[index]))
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
	output, status := execute_from(t, program, template.Value_Of_Text(value[:]))
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
			_, status := execute_from(t, program, template.Value_Of_Text(texts[index]))
			testify.Equal_Values(t, template.STATUS_OK, status)
		}
	}
	var maximum [utf8.CHARACTER_SIZE_MAXIMUM]byte
	size := utf8.Encode_Character(maximum[:], utf8.RUNE_MAX)
	program := compile_from(t, `{{js .}}`)
	_, status := execute_from(
		t, program, template.Value_Of_Text(maximum[:size]),
	)
	testify.Equal_Values(t, template.STATUS_OK, status)
}

func test_execute_composite_output(t *testing.T) {
	program := compile_from(t, `<{{.}}>`)
	output, status := execute_from(
		t, program, template.Value_Of_Sequence(nil),
	)
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "<[]>", output)

	sequence := [...]template.Value{
		template.Value_Of_Integer(-TEST_COUNT_ONE),
		template.Value_Of_Integer(-TEST_COUNT_TWO),
		template.Value_Of_Integer(-TEST_COUNT_THREE),
	}
	output, status = execute_from(
		t, program, template.Value_Of_Sequence(sequence[:]),
	)
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "<[-1 -2 -3]>", output)

	entries := [...]template.Field{
		{
			Key: template.Field_Key{Data: template.Value_Of_Text([]byte("two"))},
			Value: template.Field_Value{
				Data: template.Value_Of_Integer(TEST_COUNT_TWENTY_TWO),
			},
		},
	}
	output, status = execute_from(t, program, template.Value_Of_Map(entries[:]))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "<map[two:22]>", output)

	fields := [...]template.Field{
		{
			Name: []byte("First"),
			Value: template.Field_Value{
				Data: template.Value_Of_Integer(TEST_COUNT_SEVEN),
			},
		},
		{
			Name: []byte("Second"),
			Value: template.Field_Value{
				Data: template.Value_Of_Text([]byte("seven")),
			},
		},
	}
	output, status = execute_from(t, program, template.Value_Of_Object(fields[:]))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "<{7 seven}>", output)

	inner := [...]template.Value{template.Value_Of_Integer(TEST_COUNT_ONE)}
	outer := [...]template.Value{template.Value_Of_Sequence(inner[:])}
	output, status = execute_from(t, program, template.Value_Of_Sequence(outer[:]))
	testify.Equal_Values(t, template.STATUS_OK, status)
	testify.Equal(t, "<[[1]]>", output)

	var maximum [template.VALUE_COUNT_MAXIMUM]template.Value
	_, status = execute_from(t, program, template.Value_Of_Sequence(maximum[:]))
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
		Value: template.Value_Of_Text([]byte("ADA")), Status: template.FUNCTION_STATUS_OK,
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
		Value:  template.Value_Of_Text([]byte("hello ADA")),
		Status: template.FUNCTION_STATUS_OK,
	}
}

func identity(input template.Function_Input) (result template.Function_Result) {
	if len(input.Arguments) != TEST_COUNT_ONE {
		return template.Function_Result{Status: template.FUNCTION_STATUS_INVALID}
	}
	return template.Function_Result{
		Value: input.Arguments[TEST_COUNT_ZERO], Status: template.FUNCTION_STATUS_OK,
	}
}

func reject(template.Function_Input) (result template.Function_Result) {
	return template.Function_Result{Status: template.FUNCTION_STATUS_INVALID}
}

func invalid_nested(template.Function_Input) (result template.Function_Result) {
	invalid_kind := template.VALUE_FUNCTION + template.VALUE_KIND_INCREMENT
	values := [...]template.Value{{Kind: template.Value_Kind_Storage{invalid_kind}}}
	return template.Function_Result{
		Value: template.Value_Of_Sequence(values[:]), Status: template.FUNCTION_STATUS_OK,
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
		t.Run(one.Name, func(t *testing.T) {
			program := compile_from(t, one.Source)
			value := template.Value_Nil()
			var standard_value any
			if one.Name == "js" {
				value = template.Value_Of_Text([]byte("It'd be nice."))
				standard_value = "It'd be nice."
			}
			output, status := execute_from(t, program, value)
			testify.Equal_Values(t, template.STATUS_OK, status)
			testify.Equal(t, one.Want, output)
			testify.Equal(t, standard_execute(t, one.Source, standard_value), output)
		})
	}
}

func test_standard_library_value_execution(t *testing.T) {
	t.Helper()
	fields := [...]template.Field{
		{
			Name:  []byte("Name"),
			Value: template.Field_Value{Data: template.Value_Of_Text([]byte("Ada"))},
		},
		{
			Name:  []byte("Ready"),
			Value: template.Field_Value{Data: template.Value_Of_Boolean(true)},
		},
	}
	compare_standard_execution(
		t, `{{.Name}}|{{if .Ready}}yes{{end}}`, template.Value_Of_Object(fields[:]),
		map[string]any{"Name": "Ada", "Ready": true},
	)
	values := [...]template.Value{
		template.Value_Of_Integer(TEST_COUNT_THREE),
		template.Value_Of_Integer(TEST_COUNT_ONE),
		template.Value_Of_Integer(TEST_COUNT_TWO),
	}
	compare_standard_execution(
		t, `{{range $index, $value := .}}{{$index}}={{$value}};{{end}}`,
		template.Value_Of_Sequence(values[:]), []int{
			TEST_COUNT_THREE, TEST_COUNT_ONE, TEST_COUNT_TWO,
		},
	)
	entries := [...]template.Field{
		{
			Key: template.Field_Key{Data: template.Value_Of_Text([]byte("b"))},
			Value: template.Field_Value{
				Data: template.Value_Of_Integer(TEST_COUNT_TWO),
			},
		},
		{
			Key: template.Field_Key{Data: template.Value_Of_Text([]byte("a"))},
			Value: template.Field_Value{
				Data: template.Value_Of_Integer(TEST_COUNT_ONE),
			},
		},
	}
	compare_standard_execution(
		t, `{{range $key, $value := .}}{{$key}}={{$value}};{{end}}`,
		template.Value_Of_Map(entries[:]), map[string]int{
			"b": TEST_COUNT_TWO, "a": TEST_COUNT_ONE,
		},
	)
	compare_standard_execution(
		t, `{{index . 1}}|{{len (slice . 1 3)}}`,
		template.Value_Of_Sequence(values[:]), []int{
			TEST_COUNT_THREE, TEST_COUNT_ONE, TEST_COUNT_TWO,
		},
	)
	compare_standard_execution(
		t, `{{$value := 1}}{{$value = 2}}{{$value}}`, template.Value_Nil(), nil,
	)
	compare_standard_execution(
		t, `{{template "named" .}}{{define "named"}}value{{end}}`,
		template.Value_Nil(), nil,
	)
}

func test_standard_library_slice_capacity(t *testing.T) {
	t.Helper()
	shared_values := [...]template.Value{
		template.Value_Of_Integer(TEST_COUNT_THREE),
		template.Value_Of_Integer(TEST_COUNT_FOUR),
		template.Value_Of_Integer(TEST_COUNT_FIVE),
		template.Value_Of_Integer(TEST_COUNT_ZERO),
		template.Value_Of_Integer(TEST_COUNT_ZERO),
	}
	standard_values := [...]int{
		TEST_COUNT_THREE, TEST_COUNT_FOUR, TEST_COUNT_FIVE,
		TEST_COUNT_ZERO, TEST_COUNT_ZERO,
	}
	compare_standard_execution(
		t, `{{slice . 3 5}}`,
		template.Value_Of_Sequence(shared_values[:TEST_COUNT_THREE]),
		standard_values[:TEST_COUNT_THREE],
	)
	compare_standard_execution(
		t, `{{slice . 3 5 5}}`,
		template.Value_Of_Sequence(shared_values[:TEST_COUNT_THREE]),
		standard_values[:TEST_COUNT_THREE],
	)
}

func test_standard_library_collection_execution(t *testing.T) {
	t.Helper()
	test_standard_library_scalar_execution(t)
	values := [...]template.Value{
		template.Value_Of_Integer(TEST_COUNT_THREE),
		template.Value_Of_Integer(TEST_COUNT_FOUR),
		template.Value_Of_Integer(TEST_COUNT_FIVE),
	}
	compare_standard_execution(
		t, `{{range $index, $value := .}}{{$index}}={{$value}};{{end}}`,
		template.Value_Of_Sequence(values[:]), []int{
			TEST_COUNT_THREE, TEST_COUNT_FOUR, TEST_COUNT_FIVE,
		},
	)
	compare_standard_execution(
		t, `{{index . 1}}|{{slice . 1 3}}|{{len .}}`,
		template.Value_Of_Sequence(values[:]), []int{
			TEST_COUNT_THREE, TEST_COUNT_FOUR, TEST_COUNT_FIVE,
		},
	)
	compare_standard_execution(
		t, `{{index . 1}}|{{slice . 1 3}}|{{len .}}`,
		template.Value_Of_Text([]byte("abc")), "abc",
	)
	mixed := [...]template.Value{
		template.Value_Of_Integer(TEST_COUNT_ONE),
		template.Value_Of_Unsigned(TEST_COUNT_ONE),
	}
	compare_standard_execution(
		t, `{{eq (index . 0) (index . 1)}}|{{le (index . 0) (index . 1)}}`,
		template.Value_Of_Sequence(mixed[:]), []any{
			int64(TEST_COUNT_ONE), uint64(TEST_COUNT_ONE),
		},
	)
	compare_standard_execution(
		t, `{{if .}}true{{else}}false{{end}}`,
		template.Value_Of_Object(nil), struct{}{},
	)
	entries := [...]template.Field{{
		Key: template.Field_Key{
			Data: template.Value_Of_Text([]byte("present")),
		},
		Value: template.Field_Value{Data: template.Value_Of_Integer(TEST_COUNT_ONE)},
	}}
	compare_standard_execution(
		t, `{{index . "missing"}}`, template.Value_Of_Map(entries[:]),
		map[string]any{"present": TEST_COUNT_ONE},
	)
}

func test_standard_library_scalar_execution(t *testing.T) {
	t.Helper()
	compare_standard_execution(
		t, `<{{.}}>`, template.Value_Of_Integer(-TEST_COUNT_THIRTEEN),
		int64(-TEST_COUNT_THIRTEEN),
	)
	compare_standard_execution(
		t, `<{{.}}>`, template.Value_Of_Unsigned(TEST_COUNT_FOURTEEN),
		uint64(TEST_COUNT_FOURTEEN),
	)
	const FLOAT_ONE_BITS = uint64(big.FLOAT_64_EXPONENT_BIAS) <<
		big.FLOAT_64_EXPONENT_SHIFT
	compare_standard_execution(
		t, `<{{.}}>`, template.Value_Of_Float(template.Float(FLOAT_ONE_BITS)),
		"1",
	)
	compare_standard_execution(
		t, `<{{.}}>`, template.Value_Of_Complex(template.Complex{
			Real:      template.Complex_Real(FLOAT_ONE_BITS),
			Imaginary: template.Complex_Imaginary(FLOAT_ONE_BITS),
		}), complex(TEST_COUNT_ONE, TEST_COUNT_ONE),
	)
	compare_standard_execution(
		t, `{{range $value := .}}{{$value}}{{else}}empty{{end}}`,
		template.Value_Of_Integer(TEST_COUNT_THREE), int64(TEST_COUNT_THREE),
	)
	compare_standard_execution(
		t, `{{range $value := .}}{{$value}}{{else}}empty{{end}}`,
		template.Value_Of_Unsigned(TEST_COUNT_THREE), uint64(TEST_COUNT_THREE),
	)
	compare_standard_execution(
		t, `{{range $value := .}}{{$value}}{{else}}empty{{end}}`,
		template.Value_Of_Integer(-TEST_COUNT_ONE), int64(-TEST_COUNT_ONE),
	)
}

func compare_standard_execution(
	t *testing.T, source string, shared_value template.Value, standard_value any,
) {
	t.Helper()
	program := compile_from(t, source)
	shared_output, status := execute_from(t, program, shared_value)
	testify.Equal_Values(t, template.STATUS_OK, status)
	expected := standard_execute(t, source, standard_value)
	testify.Equal(t, expected, shared_output)
}

func standard_execute(t *testing.T, source string, value any) (output string) {
	t.Helper()
	program, failure := standard_template.New("subject").Parse(source)
	if failure != nil {
		t.Fatal(failure)
	}
	var destination standard_output
	failure = program.Execute(&destination, value)
	if failure != nil {
		t.Fatal(failure)
	}
	return string(destination.Data[:destination.Count])
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
	t *testing.T, program template.Program, value template.Value,
	functions template.Functions,
) (output string, status template.Status) {
	t.Helper()
	var execution execution_storage
	var destination [template.OUTPUT_SIZE_MAXIMUM]byte
	count, _, status := template.Execute_Into(template.Execute_Input{
		Destination: destination[:], Program: program, Value: value,
		Functions: functions, Workspace: workspace_from(&execution),
	})
	return string(destination[:count]), status
}

func syntax_from(workspace *template.Syntax_Workspace) (input template.Syntax_Input) {
	input.State[template.SYNTAX_FIELD] = workspace
	return input
}

func syntax_node(
	program template.Program, reference template.Node_Reference,
) (node *template.Node) {
	return &program.Syntax[template.SYNTAX_FIELD].Nodes[reference-template.NODE_COUNT_INCREMENT]
}

func syntax_first_pipe(program template.Program) (pipe *template.Node) {
	root := syntax_node(program, template.Node_Reference(program.Document.Root))
	action := syntax_node(program, template.Node_Reference(root.First_Child))
	return syntax_node(program, template.Node_Reference(action.First_Child))
}

type execution_storage struct {
	Workspace       template.Workspace
	Output          [template.OUTPUT_SIZE_MAXIMUM]byte
	Literals        [template.QUOTED_DECODED_SIZE_MAXIMUM]byte
	Literal_Counts  [template.NODE_COUNT_MAXIMUM]template.Literal_Count
	Literal_Offsets [template.NODE_COUNT_MAXIMUM]template.Literal_Count
	Literal_Decoded [template.NODE_COUNT_MAXIMUM]template.Boolean
	Name_Left       [template.QUOTED_DECODED_SIZE_MAXIMUM]byte
	Name_Right      [template.QUOTED_DECODED_SIZE_MAXIMUM]byte
	Number          [template.NUMBER_SIZE_MAXIMUM]byte
	Generated       [template.GENERATED_SIZE_MAXIMUM]byte
	Float_Value     big.Float
	Float_Parse     big.Float_Parse_Workspace
	Float_Text      big.Float_Text_Workspace
	Frames          [template.FRAME_COUNT_MAXIMUM]template.Frame_Storage
	Variables       [template.VARIABLE_COUNT_MAXIMUM]template.Variable
	Arguments       [template.ARGUMENT_COUNT_MAXIMUM]template.Value
	Validation      [template.FRAME_COUNT_MAXIMUM]template.Value
	Evaluations     [template.FRAME_COUNT_MAXIMUM]template.Evaluation_Frame
	Format          [template.FRAME_COUNT_MAXIMUM]template.Value_Format_Frame_Storage
}

func workspace_from(storage *execution_storage) (input template.Workspace_Input) {
	storage.Workspace = template.Workspace{
		Output: storage.Output[:], Literals: storage.Literals[:],
		Literal_Counts:  storage.Literal_Counts[:],
		Literal_Offsets: storage.Literal_Offsets[:],
		Literal_Decoded: storage.Literal_Decoded[:],
		Name_Left:       storage.Name_Left[:], Name_Right: storage.Name_Right[:],
		Number:      storage.Number[:],
		Generated:   storage.Generated[:],
		Float_Value: template.Float_Value_Workspace{storage.Float_Value},
		Float_Parse: template.Float_Parse_Workspace(storage.Float_Parse),
		Float_Text:  template.Float_Text_Workspace(storage.Float_Text),
		Frames:      storage.Frames[:],
		Variables:   storage.Variables[:], Arguments: storage.Arguments[:],
		Validation:  storage.Validation[:],
		Evaluations: storage.Evaluations[:],
		Format:      storage.Format[:],
	}
	input.State[template.WORKSPACE_FIELD] = &storage.Workspace
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
	testify.Equal(t, len(maximum), len(source.Data[template.SOURCE_FIELD]))

	var oversized [template.SOURCE_SIZE_UNVALIDATED_MAXIMUM]byte
	source, status = template.Source_Validate(oversized[:])
	testify.Equal_Values(t, template.PARSE_STATUS_INPUT_INVALID, status)
	testify.Equal(t, template.SOURCE_SIZE_MINIMUM, len(source.Data[template.SOURCE_FIELD]))
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
		source, configuration, parse_workspace_from(&workspace),
	)
	testify.Equal_Values(t, template.PARSE_STATUS_SYNTAX_INVALID, status)
	testify.Equal_Values(t, template.DIAGNOSTIC_SYNTAX_INVALID, diagnostic.Code)
	testify.Equal(t, template.Parse_Diagnostic_Position(len("{{.")), diagnostic.Position)
	testify.Equal_Values(t, template.NO_NODE, document.Root)

	source = source_from(t, "{{range $value := .}}{{$value}}{{end}}{{$value}}")
	document, diagnostic, status = template.Parse_Into(
		source, configuration, parse_workspace_from(&workspace),
	)
	testify.Equal_Values(t, template.PARSE_STATUS_SYNTAX_INVALID, status)
	testify.Equal_Values(t, template.DIAGNOSTIC_SYNTAX_INVALID, diagnostic.Code)
	testify.Equal_Values(t, template.NO_NODE, document.Root)

	source = source_from(t, "{{if .}}x")
	document, diagnostic, status = template.Parse_Into(
		source, configuration, parse_workspace_from(&workspace),
	)
	testify.Equal_Values(t, template.PARSE_STATUS_SYNTAX_INVALID, status)
	testify.Equal_Values(t, template.NO_NODE, document.Root)

	source = source_from(t, "{{$missing}}")
	document, diagnostic, status = template.Parse_Into(
		source, configuration, parse_workspace_from(&workspace),
	)
	testify.Equal_Values(t, template.PARSE_STATUS_SYNTAX_INVALID, status)
	testify.Equal_Values(t, template.NO_NODE, document.Root)

	source = source_from(t, "{{$first, $second := .}}")
	document, diagnostic, status = template.Parse_Into(
		source, configuration, parse_workspace_from(&workspace),
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
			source, configuration, parse_workspace_from(&workspace),
		)
		testify.Equal_Values(t, template.PARSE_STATUS_SYNTAX_INVALID, status)
		testify.Equal_Values(t, template.DIAGNOSTIC_SYNTAX_INVALID, diagnostic.Code)
		testify.Equal_Values(t, template.NO_NODE, document.Root)
	}
	test_hostile_bytes(t)
}

// Test_Bounds reports absent caller state before parsing.
func Test_Parse_Bounds(t *testing.T) {
	configuration := configuration_from(t)
	source := source_from(t, "x")
	document, diagnostic, status := template.Parse_Into(
		source, configuration, template.Syntax_Workspace_Input{},
	)
	testify.Equal_Values(t, template.PARSE_STATUS_WORKSPACE_INVALID, status)
	testify.Equal_Values(t, template.DIAGNOSTIC_WORKSPACE_INVALID, diagnostic.Code)
	testify.Equal_Values(t, template.NO_NODE, document.Root)

	test_hostile_public_inputs(t, source, configuration)
	test_workspace_state_bounds(t, source, configuration)

	limited := template.Syntax_Workspace{
		Node_Limit: template.Node_Limit_Storage{
			template.Node_Limit(template.NODE_COUNT_INCREMENT),
		},
	}
	document, diagnostic, status = template.Parse_Into(
		source, configuration, parse_workspace_from(&limited),
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
	var document template.Document
	var diagnostic template.Parse_Diagnostic
	var status template.Parse_Status
	testify.Zero_Allocation(t, func() {
		document, diagnostic, status = template.Parse_Into(
			source, configuration, parse_workspace_from(&workspace),
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
		var document template.Document
		var diagnostic template.Parse_Diagnostic
		var status template.Parse_Status
		testify.Zero_Allocation(t, func() {
			document, diagnostic, status = template.Parse_Into(
				source, configuration, parse_workspace_from(&workspace),
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
	for _, malformed := range [...]string{`"`, `"x]`} {
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
			source, configuration, parse_workspace_from(&complete),
		)
		testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
		for limit := template.Node_Limit(template.NODE_COUNT_INCREMENT); ; limit++ {
			if limit >= template.Node_Limit(document.Node_Count) {
				break
			}
			limited := template.Syntax_Workspace{
				Node_Limit: template.Node_Limit_Storage{limit},
			}
			_, _, status = template.Parse_Into(
				source, configuration, parse_workspace_from(&limited),
			)
			testify.Equal_Values(t, template.PARSE_STATUS_CAPACITY_EXCEEDED, status)
		}
	}
}

// Public aggregates must reject malicious values before parser state changes.
func test_hostile_public_inputs(
	t *testing.T, source template.Source, configuration template.Configuration,
) {
	t.Helper()
	var oversized [template.SOURCE_SIZE_UNVALIDATED_MAXIMUM]byte
	malicious_source := template.Source{
		Data: template.Source_Storage{template.Source_Data(oversized[:])},
	}
	var workspace template.Syntax_Workspace
	document, diagnostic, status := template.Parse_Into(
		malicious_source, configuration, parse_workspace_from(&workspace),
	)
	testify.Equal_Values(t, template.PARSE_STATUS_INPUT_INVALID, status)
	testify.Equal_Values(t, template.DIAGNOSTIC_INPUT_INVALID, diagnostic.Code)
	testify.Equal_Values(t, template.NO_NODE, document.Root)

	malicious_configuration := template.Configuration{
		Left: template.Opening_Delimiter_Storage{template.Opening_Delimiter("<")},
	}
	document, diagnostic, status = template.Parse_Into(
		source, malicious_configuration, parse_workspace_from(&workspace),
	)
	testify.Equal_Values(t, template.PARSE_STATUS_CONFIGURATION_INVALID, status)
	testify.Equal_Values(t, template.DIAGNOSTIC_CONFIGURATION_INVALID, diagnostic.Code)
	testify.Equal_Values(t, template.NO_NODE, document.Root)

	workspace.Node_Limit = template.Node_Limit_Storage{
		template.Node_Limit(template.NODE_COUNT_MAXIMUM) +
			template.Node_Limit(template.NODE_COUNT_INCREMENT),
	}
	document, diagnostic, status = template.Parse_Into(
		source, configuration, parse_workspace_from(&workspace),
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
				{
					Code:     template.DIAGNOSTIC_INPUT_INVALID,
					Position: template.Parse_Diagnostic_Position(len("<")),
				},
			},
			Node_Limit: template.Node_Limit_Storage{
				template.Node_Limit(
					template.NODE_COUNT_INCREMENT +
						template.NODE_COUNT_INCREMENT,
				),
			},
		},
		{
			Parse_Diagnostic: template.Syntax_Diagnostic_Storage{
				{
					Code: template.DIAGNOSTIC_CONFIGURATION_INVALID,
					Position: template.Parse_Diagnostic_Position(
						template.SOURCE_SIZE_MAXIMUM,
					),
				},
			},
			Node_Limit: template.Node_Limit_Storage{
				template.Node_Limit(template.NODE_COUNT_MAXIMUM),
			},
		},
		{
			Parse_Diagnostic: template.Syntax_Diagnostic_Storage{
				{Code: template.DIAGNOSTIC_CAPACITY_EXCEEDED},
			},
		},
	}
	for index := range workspace_cases {
		document, diagnostic, status := template.Parse_Into(
			source, configuration, parse_workspace_from(&workspace_cases[index]),
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
		source, configuration_from(t), parse_workspace_from(&workspace),
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
		source, configuration, parse_workspace_from(&workspace),
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
		source, configuration, parse_workspace_from(&workspace),
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
			source, configuration, parse_workspace_from(&workspace),
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
			source, configuration, parse_workspace_from(&workspace),
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
				source, configuration, parse_workspace_from(&workspace),
			)
			accepted := status == template.PARSE_STATUS_OK
			if status == template.PARSE_STATUS_SYNTAX_INVALID {
				accepted = true
			}
			testify.True(t, accepted)
		}
	}
	for _, quote := range [...]byte{
		byte(template.QUOTE_STRING), byte(template.QUOTE_CHARACTER),
		byte(template.QUOTE_RAW_STRING),
	} {
		quoted := [...]byte{'<', '%', quote, quote, '%', 'X', 'Y', 'Z'}
		source, source_status := template.Source_Validate(quoted[:])
		testify.Equal_Values(t, template.PARSE_STATUS_OK, source_status)
		var workspace template.Syntax_Workspace
		template.Parse_Into(source, configuration, parse_workspace_from(&workspace))

		identifier := [...]byte{'<', '%', 'A', quote, quote, '%', 'X', 'Y', 'Z'}
		source, source_status = template.Source_Validate(identifier[:])
		testify.Equal_Values(t, template.PARSE_STATUS_OK, source_status)
		workspace = template.Syntax_Workspace{}
		template.Parse_Into(source, configuration, parse_workspace_from(&workspace))
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
		source, configuration, parse_workspace_from(&workspace),
	)
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	testify.Equal_Values(t, template.NODE_COUNT_MAXIMUM, document.Node_Count)
	workspace = template.Syntax_Workspace{}
	program, _, status := template.Compile(template.Compile_Input{
		Source: unvalidated[:], Left: []byte("<"), Right: []byte(">"),
		Workspace: syntax_from(&workspace),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	testify.Equal_Values(
		t, template.NODE_COUNT_MAXIMUM, program.Document.Node_Count,
	)
	test_execute_program_input(t, program, template.STATUS_OUTPUT_TOO_LARGE)
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
	for count < template.TEMPLATE_COUNT_MAXIMUM {
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
		source, configuration, parse_workspace_from(&workspace),
	)
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	testify.Equal_Values(
		t, template.Template_Count(template.TEMPLATE_COUNT_MAXIMUM),
		document.Template_Count,
	)
	workspace = template.Syntax_Workspace{}
	program, _, status := template.Compile(template.Compile_Input{
		Source: unvalidated[:], Left: []byte("<"), Right: []byte(">"),
		Workspace: syntax_from(&workspace),
	})
	testify.Equal_Values(t, template.PARSE_STATUS_OK, status)
	testify.Equal_Values(
		t, template.Template_Count(template.TEMPLATE_COUNT_MAXIMUM),
		program.Document.Template_Count,
	)
	test_execute_program_input(t, program, template.STATUS_OK)
}

// Densest action prefix must place first definition at its formula maximum.
func test_maximum_first_template(t *testing.T) {
	t.Helper()
	action := [...]byte{'<', '.', '>'}
	definition := [...]byte{
		'<', 'd', 'e', 'f', 'i', 'n', 'e', ' ', '`', 'x', '`', '>',
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
		source, configuration, parse_workspace_from(&workspace),
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

func assert_maximum_parse(t *testing.T, unvalidated template.Source_Unvalidated) {
	t.Helper()
	source, source_status := template.Source_Validate(unvalidated)
	testify.Equal_Values(t, template.PARSE_STATUS_OK, source_status)
	configuration := configuration_input_from(t, template.Configuration_Input{
		Left:  template.Left_Delimiter_Unvalidated("<"),
		Right: template.Right_Delimiter_Unvalidated(">"),
	})
	var workspace template.Syntax_Workspace
	_, _, status := template.Parse_Into(source, configuration, parse_workspace_from(&workspace))
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

	source, document, workspace = parse_text(
		t, "{{$变量 := .}}{{$变量.字段}}{{.世界.值}}",
	)
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
	with_node := node_at(
		t, document, &workspace, template.Node_Reference(if_node.Next_Sibling),
	)
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
	source, document, workspace = parse_configuration(
		t, "a{{/* comment */}}b", configuration,
	)
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
		t,
		`root{{template "card" .}}{{define "card"}}card {{.}}{{end}}`,
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

	source, document, workspace = parse_text(
		t, `{{block "default" .}}value {{.}}{{end}}`,
	)
	root = node_at(t, document, &workspace, template.Node_Reference(document.Root))
	invocation = node_at(t, document, &workspace, template.Node_Reference(root.First_Child))
	testify.Equal_Values(t, template.NODE_TEMPLATE, invocation.Kind)
	testify.Equal(t, template.Template_Count(TEST_COUNT_ONE), document.Template_Count)
	definition = node_at(
		t, document, &workspace,
		template.Node_Reference(first_template_reference(document)),
	)
	testify.Equal(t, `"default"`, node_value(source, definition))

	_, document, _ = parse_text(
		t, `{{define "first"}}{{end}}{{define "second"}}{{end}}`,
	)
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
		source, configuration, parse_workspace_from(&workspace),
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
		source, configuration, parse_workspace_from(&workspace),
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
	input.State[template.SYNTAX_WORKSPACE_FIELD] = workspace
	return input
}

func first_template_reference(document template.Document) (reference template.Node_Reference) {
	return template.Node_Reference(document.First_Template) +
		template.Node_Reference(template.ROOT_NODE_COUNT)
}

func node_at(
	t *testing.T,
	document template.Document,
	workspace *template.Syntax_Workspace,
	reference template.Node_Reference,
) (node template.Node) {
	t.Helper()
	testify.True(t, uint16(reference) != uint16(template.NO_NODE))
	testify.True(t, int(reference) <= int(document.Node_Count))
	return workspace.Nodes[int(reference)-int(template.NODE_COUNT_INCREMENT)]
}

func node_value(source template.Source, node template.Node) (value string) {
	return string(source.Data[template.SOURCE_FIELD][node.Value_Start:node.Value_End])
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
			_, standard_failure := parse.Parse(
				"test", source_text, "", "",
			)
			source := source_from(t, source_text)
			var workspace template.Syntax_Workspace
			_, _, status := template.Parse_Into(
				source, configuration, parse_workspace_from(&workspace),
			)
			testify.Equal(
				t, standard_failure == nil,
				status == template.PARSE_STATUS_OK,
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
			_, standard_failure := parse.Parse(
				"test", source_text, "", "",
			)
			source := source_from(t, source_text)
			var workspace template.Syntax_Workspace
			_, _, status := template.Parse_Into(
				source, configuration, parse_workspace_from(&workspace),
			)
			testify.Equal(
				t, standard_failure == nil,
				status == template.PARSE_STATUS_OK,
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
		source, configuration, parse_workspace_from(&workspace),
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
				source, configuration, parse_workspace_from(&workspace),
			)
			testify.Equal(
				t, standard_failure == nil,
				status == template.PARSE_STATUS_OK,
			)
		})
	}
}

func assert_standard_library_numbers(t *testing.T, cases []string) {
	t.Helper()
	configuration := configuration_from(t)
	for _, source_text := range cases {
		t.Run(source_text, func(t *testing.T) {
			_, standard_failure := parse.Parse(
				"test", source_text, "", "",
			)
			source := source_from(t, source_text)
			var workspace template.Syntax_Workspace
			_, _, status := template.Parse_Into(
				source, configuration, parse_workspace_from(&workspace),
			)
			testify.Equal(
				t, standard_failure == nil,
				status == template.PARSE_STATUS_OK,
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
