package cli

import (
	"errors"
	"testing"

	"local/james-orcales/shared/diff/levenshtein"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/nbio"
	"local/james-orcales/shared/slices"
	"local/james-orcales/shared/strings"
)

// Test_Parse_Error_Operation_Boundaries covers each borrowed diagnostic boundary.
func Test_Parse_Error_Operation_Boundaries(t *testing.T) {
	storage := make(Parse_Error_Storage, FAILURE_SIZE_MAXIMUM)
	storage[0] = 'x'
	storage[1] = 'y'
	values := [...]Parse_Error{
		{},
		{Storage: storage, Size: Parse_Error_Size{}},
		{Storage: storage, Size: Parse_Error_Size{Value: 1}},
		{Storage: storage, Size: Parse_Error_Size{Value: 2}},
		{
			Storage: storage,
			Size: Parse_Error_Size{
				Value: Parse_Error_Size_Value(FAILURE_SIZE_MAXIMUM),
			},
		},
	}
	for _, value := range values {
		Parse_Error_Present(value)
		Parse_Error_Bytes(value)
	}
	if !Parse_Error_Equals(values[0], "") {
		t.Fatal("empty parse error differs from empty diagnostic")
	}
	if !Parse_Error_Equals(values[2], "x") {
		t.Fatal("one-byte parse error differs from source")
	}
	if Parse_Error_Equals(values[3], "xz") {
		t.Fatal("different parse error bytes compare equal")
	}
	maximum := Failure_Fragment(string(make([]byte, strings.TEXT_SIZE_MAXIMUM)))
	if Parse_Error_Equals(values[4], maximum) {
		t.Fatal("different parse error lengths compare equal")
	}
}

// Test_Internal_Render_Boundaries exercises each private rendering phase with
// every caller cursor boundary.
func Test_Internal_Render_Boundaries(t *testing.T) {
	for operation := range 14 {
		for _, size := range [...]strings.Size_Value{
			strings.TEXT_SIZE_MINIMUM,
			1,
			2,
			strings.TEXT_SIZE_MAXIMUM,
		} {
			cli_internal_render_boundary(operation, size)
		}
	}
}

func cli_internal_render_boundary(operation int, size strings.Size_Value) {
	defer func() { recover() }()
	var output Output
	output.Builder.Storage = make([]byte, strings.TEXT_SIZE_MAXIMUM)
	output.Builder.Size = size
	reference := Output_Reference_Of(&output)
	type_state := Option_Type_State{Value: OPTION_TYPE_BOOLEAN}
	external_type := External_Type_State{Value: OPTION_TYPE_BOOLEAN}
	switch operation {
	case 0:
		environment_default(reference, Rendered_External{Type: external_type})
	case 1:
		help_flag_cell_write(
			reference, "x", OPTION_TYPE_BOOLEAN, Option_Enumeration{},
			INDENT_FLAG, false,
		)
	case 2:
		option_signature_write(reference, "x", type_state, Option_Enumeration{})
	case 3:
		print_command_catalog(reference, Command_Declarations{{
			Label: "x", Description: "x",
		}})
	case 4:
		print_environment_help(reference, Environment_Declarations{})
	case 5:
		print_help_flag(
			reference,
			Rendered_Option{Label: "x", Description: "x", Type: type_state},
			INDENT_FLAG, false,
			Flag_Cell_Size{Value: Flag_Cell_Size_Value(FLAG_CELL_SIZE_MINIMUM)},
			Default_Cell_Size{},
		)
	case 6:
		print_help_flag_section(reference, FLAG_SECTION_TITLE, Options{})
	case 7:
		print_help_flags(
			reference, Options{}, INDENT_FLAG, false,
			Flag_Cell_Size{Value: Flag_Cell_Size_Value(FLAG_CELL_SIZE_MINIMUM)},
			Default_Cell_Size{},
		)
	case 8:
		print_help_global_flag_section(reference, Help_Flags{
			Short_Label: HELP_SHORT_LABEL, Long_Label: HELP_LABEL,
			Description: HELP_DESCRIPTION,
		}, Global_Flags{})
	case 9:
		print_secret_help(reference, Secrets{})
	case 10:
		render_decimal(reference, Rendered_Integer{})
	case 11:
		render_external_type(reference, external_type, External_Enumeration{})
	case 12:
		output_write_newline(reference)
	case 13:
		output_write_space(reference)
	}
}

// Test_Internal_Failure_Boundaries exercises each private diagnostic phase
// with every live cursor boundary.
func Test_Internal_Failure_Boundaries(t *testing.T) {
	for operation := range 9 {
		for _, size := range [...]Failure_Size_Value{
			strings.TEXT_SIZE_MINIMUM,
			1,
			2,
			Failure_Size_Value(FAILURE_SIZE_MAXIMUM),
		} {
			cli_internal_failure_boundary(operation, size)
		}
	}
}

func cli_internal_failure_boundary(operation int, size Failure_Size_Value) {
	defer func() { recover() }()
	record := Failure_Size{Value: size}
	failure := Failure_Writer{
		Storage: make([]byte, FAILURE_SIZE_MAXIMUM), Size: &record,
	}
	switch operation {
	case 0:
		environment_parse(
			Environment_Variables{}, Process_Environment{},
			Environment_Warnings{}, Environment_Sources{}, failure,
		)
	case 1:
		environment_enum_failure_write(
			failure, "A", External_Enumeration{String: []string{"x"}},
			"y", external_string_enum_failure,
		)
	case 2:
		environment_source_failure_write(
			failure, "A", Environment_Source{}, environment_missing_failure,
		)
	case 3:
		failure_write(failure, "x")
	case 4:
		failure_write_decimal(failure, Failure_Integer{})
	case 5:
		failure_write_integer_set(failure, Failure_Integer_Set{0})
	case 6:
		failure_write_path(failure, Failure_Secret_Key{Value: "A"}, Path_Failure{
			Path: Path_Failure_Path{Value: "/x"},
			Kind: Path_Failure_Kind{Value: PATH_FAILURE_KIND_NONE},
		})
	case 7:
		failure_write_quoted(failure, "x")
	case 8:
		failure_write_string_set(failure, Failure_String_Set{"x"})
	}
}

// Test_Internal_Value_Boundaries exercises private value contracts that public
// flows reach only after earlier syntax has been consumed.
func Test_Internal_Value_Boundaries(t *testing.T) {
	Candidate_Equal(Candidate{}, Value_Text(cli_internal_text(1)))
	Candidate_Equal(Candidate{}, Value_Text(cli_internal_text(strings.TEXT_SIZE_MAXIMUM)))
	var output Output
	output.Builder.Storage = make([]byte, strings.TEXT_SIZE_MAXIMUM)
	Candidate_Write(Output_Reference_Of(&output), Candidate{State: Candidate_State{
		Integer: Integer(bits.INTEGER_MINIMUM), Has_Integer: true,
	}})
	for _, size := range [...]int{0, 1, 2, strings.TEXT_SIZE_MAXIMUM} {
		failure_cause(errors.New(cli_internal_text(size)))
	}
	cli_internal_suggestion_boundaries()
	cli_internal_option_boundaries()
	for operation_index := range 12 {
		cli_internal_render_value_boundary(operation_index)
	}
	for operation_index := 12; operation_index < 23; operation_index++ {
		cli_internal_render_result_boundary(operation_index)
	}
	cli_internal_program_boundaries()
	cli_internal_constructor_boundaries()
	cli_internal_secret_boundaries()
	cli_internal_empty_keys()
}

func cli_internal_text(size int) (text string) {
	value := make([]byte, size)
	for index := range value {
		value[index] = 'x'
	}
	return string(value)
}

func cli_internal_suggestion_boundaries() {
	for _, match := range [...]Label{
		Label(cli_internal_text(1)), Label(cli_internal_text(2)),
		Label(cli_internal_text(strings.TEXT_SIZE_MAXIMUM)),
	} {
		var workspace levenshtein.Workspace
		state := Suggestion_State{Match: match}
		closest_label(&workspace, "xxx", "xxy", &state)
	}
	for _, best := range [...]levenshtein.Distance_Value{
		2, levenshtein.Distance_Value(strings.TEXT_SIZE_MAXIMUM),
	} {
		var workspace levenshtein.Workspace
		state := Suggestion_State{Best: best}
		closest_label(&workspace, "xxx", "xxy", &state)
	}
	help_labels := Help_Labels{
		Short: HELP_SHORT_LABEL, Long: HELP_LABEL, Hidden: HELP_HIDDEN_INTERNAL,
	}
	closest_option_label(nil, help_labels, nil, nil, "x")
	help_labels.Hidden = HELP_HIDDEN_VISIBLE
	for _, match := range [...]Label{
		Label(cli_internal_text(1)), Label(cli_internal_text(2)),
		Label(cli_internal_text(OPTION_LABEL_SIZE_MAXIMUM)),
	} {
		option := Resolved_Option{
			Label: Option_Label(match),
			Type:  Option_Type_State{Value: OPTION_TYPE_STRING},
		}
		closest_option_label(nil, help_labels, Resolved_Arguments{option}, nil,
			Option_Label(match))
	}
}

func cli_internal_option_boundaries() {
	storage := make([]byte, FAILURE_SIZE_MAXIMUM)
	display := Display_Name{Name: Display_Option_Name{Value: "x"}}
	for _, integer := range [...]Integer{
		Integer(bits.INTEGER_MINIMUM), Integer(bits.INTEGER_MAXIMUM), -1, 2,
	} {
		option_check_enum(Option_Enumeration{}, "", integer, display, storage)
	}
	integers := make([]int, slices.COUNT_MAXIMUM)
	strings_enum := make([]string, slices.COUNT_MAXIMUM)
	option_check_enum(Option_Enumeration{Integers: []int{0}}, "", 0, display, storage)
	option_check_enum(Option_Enumeration{Integers: integers}, "", 0, display, storage)
	option_check_enum(Option_Enumeration{String: strings_enum}, "", 0, display, storage)
	cli_internal_set_positional(Resolved_Arguments{}, 0)
	cli_internal_set_positional(cli_internal_arguments(slices.COUNT_MAXIMUM), 0)
	cli_internal_set_positional(cli_internal_arguments(3), 2)
	cli_internal_set_positional(
		cli_internal_arguments(slices.COUNT_MAXIMUM), slices.FOUND_INDEX_MAXIMUM)
	source := make([]byte, strings.TEXT_SIZE_MAXIMUM)
	var output Output
	output.Builder.Storage = make([]byte, strings.TEXT_SIZE_MAXIMUM)
	Output_Write(Output_Reference_Of(&output), source)
	external_convert_state(
		External_Type_State{Value: OPTION_TYPE_STRING}, External_Enumeration{},
		Environment_Value_Text(cli_internal_text(ENVIRONMENT_VALUE_SIZE_MAXIMUM)),
	)
}

func cli_internal_arguments(count int) (arguments Resolved_Arguments) {
	arguments = make(Resolved_Arguments, count)
	for index := range arguments {
		arguments[index] = Resolved_Option{
			Label: "x", Type: Option_Type_State{Value: OPTION_TYPE_STRING},
		}
	}
	return arguments
}

func cli_internal_set_positional(arguments Resolved_Arguments, index Option_Index) {
	defer func() { recover() }()
	option_set_positional(
		arguments, index, "x", make([]byte, FAILURE_SIZE_MAXIMUM))
}

func cli_internal_render_value_boundary(operation int) {
	defer func() { recover() }()
	var output Output
	output.Builder.Storage = make([]byte, strings.TEXT_SIZE_MAXIMUM)
	reference := Output_Reference_Of(&output)
	integers := make([]int, slices.COUNT_MAXIMUM)
	strings_enum := make([]string, slices.COUNT_MAXIMUM)
	switch operation {
	case 0:
		help_flag_cell_size("", OPTION_TYPE_BOOLEAN, Option_Enumeration{},
			INDENT_FLAG, false)
	case 1:
		help_flag_cell_size(
			Option_Label(cli_internal_text(strings.TEXT_SIZE_MAXIMUM-
				FLAG_CELL_SIZE_MINIMUM)),
			OPTION_TYPE_BOOLEAN, Option_Enumeration{}, INDENT_FLAG, false)
	case 2:
		help_flag_cell_write(reference, "x", OPTION_TYPE_INTEGER,
			Option_Enumeration{Integers: integers}, INDENT_FLAG, false)
	case 3:
		help_flag_cell_write(reference, "x", OPTION_TYPE_STRING,
			Option_Enumeration{String: strings_enum}, INDENT_FLAG, false)
	case 4:
		help_flag_cell_write(reference,
			Option_Label(cli_internal_text(OPTION_LABEL_SIZE_MAXIMUM)),
			OPTION_TYPE_BOOLEAN, Option_Enumeration{}, INDENT_FLAG, false)
	case 5:
		option_signature_write(reference, "x",
			Option_Type_State{Value: OPTION_TYPE_INTEGER},
			Option_Enumeration{Integers: []int{0}})
	case 6:
		option_signature_write(reference, "x",
			Option_Type_State{Value: OPTION_TYPE_INTEGER},
			Option_Enumeration{Integers: []int{0, 0}})
	case 7:
		option_signature_write(reference, "x",
			Option_Type_State{Value: OPTION_TYPE_INTEGER},
			Option_Enumeration{Integers: integers})
	case 8:
		option_signature_write(reference, "x",
			Option_Type_State{Value: OPTION_TYPE_STRING},
			Option_Enumeration{String: []string{"", ""}})
	case 9:
		option_signature_write(reference, "x",
			Option_Type_State{Value: OPTION_TYPE_STRING},
			Option_Enumeration{String: strings_enum})
	case 10:
		option_signature_write(reference,
			Option_Label(cli_internal_text(OPTION_LABEL_SIZE_MAXIMUM)),
			Option_Type_State{Value: OPTION_TYPE_BOOLEAN}, Option_Enumeration{})
	case 11:
		option_signature_write(reference, "x",
			Option_Type_State{Value: OPTION_TYPE_INTEGERS}, Option_Enumeration{})
	}
}

func cli_internal_render_result_boundary(operation int) {
	defer func() { recover() }()
	var output Output
	output.Builder.Storage = make([]byte, strings.TEXT_SIZE_MAXIMUM)
	reference := Output_Reference_Of(&output)
	integers := make([]int, slices.COUNT_MAXIMUM)
	strings_enum := make([]string, slices.COUNT_MAXIMUM)
	switch operation {
	case 12:
		render_external_type(reference,
			External_Type_State{Value: OPTION_TYPE_INTEGER},
			External_Enumeration{Integers: []int{0, 0}})
	case 13:
		render_external_type(reference,
			External_Type_State{Value: OPTION_TYPE_INTEGER},
			External_Enumeration{Integers: integers})
	case 14:
		render_external_type(reference,
			External_Type_State{Value: OPTION_TYPE_STRING},
			External_Enumeration{String: []string{"", ""}})
	case 15:
		render_external_type(reference,
			External_Type_State{Value: OPTION_TYPE_STRING},
			External_Enumeration{String: strings_enum})
	case 16:
		cli_internal_print_flag(reference, Rendered_Option{},
			Flag_Cell_Size_Value(strings.TEXT_SIZE_MAXIMUM), 0)
	case 17:
		cli_internal_print_flag(reference, Rendered_Option{},
			Flag_Cell_Size_Value(FLAG_CELL_SIZE_MINIMUM),
			Default_Cell_Size_Value(strings.TEXT_SIZE_MAXIMUM))
	case 18:
		cli_internal_print_flag(reference, Rendered_Option{
			Label: Option_Label(cli_internal_text(OPTION_LABEL_SIZE_MAXIMUM)),
		}, Flag_Cell_Size_Value(FLAG_CELL_SIZE_MINIMUM), 0)
	case 19:
		cli_internal_print_flag(reference, Rendered_Option{
			Label: "x", Type: Option_Type_State{Value: OPTION_TYPE_INTEGERS},
		}, Flag_Cell_Size_Value(FLAG_CELL_SIZE_MINIMUM), 0)
	case 20:
		cli_internal_print_flag(reference, Rendered_Option{
			Label: "x", Type: Option_Type_State{Value: OPTION_TYPE_INTEGER},
			Enumeration: Option_Enumeration{Integers: integers},
		}, Flag_Cell_Size_Value(FLAG_CELL_SIZE_MINIMUM), 0)
	case 21:
		cli_internal_print_flag(reference, Rendered_Option{
			Label: "x", Enumeration: Option_Enumeration{String: strings_enum},
		}, Flag_Cell_Size_Value(FLAG_CELL_SIZE_MINIMUM), 0)
	case 22:
		cli_internal_print_flag(reference, Rendered_Option{
			Label:  "x",
			String: Value_Text(cli_internal_text(strings.TEXT_SIZE_MAXIMUM)),
		}, Flag_Cell_Size_Value(FLAG_CELL_SIZE_MINIMUM), 0)
	}
}

func cli_internal_print_flag(
	output Output_Reference, rendered Rendered_Option,
	cell_width Flag_Cell_Size_Value, default_width Default_Cell_Size_Value,
) {
	print_help_flags(output, Options{}, INDENT_FLAG, false,
		Flag_Cell_Size{Value: cell_width}, Default_Cell_Size{Value: default_width})
	print_help_flag(output, rendered, INDENT_FLAG, false,
		Flag_Cell_Size{Value: cell_width}, Default_Cell_Size{Value: default_width})
}

func cli_internal_program_boundaries() {
	program := New(New_Input{
		Label: "p", Description: "p", Mode: PROGRAM_MODE_MULTICALL,
		Commands: Command_Declarations{
			{Label: "one", Description: "one"},
			{Label: "two", Description: "two"},
		},
	})
	var output Output
	output.Builder.Storage = make([]byte, strings.TEXT_SIZE_MAXIMUM)
	Print_Help(Output_Reference_Of(&output), program)
	program_help_context(
		program.Selection.Commands, program.Selection.Mode, Help_Arguments{"one", ""})
}

func cli_internal_constructor_boundaries() {
	for _, size := range [...]int{1, 2, strings.TEXT_SIZE_MAXIMUM} {
		New(New_Input{
			Label: "p", Mode: PROGRAM_MODE_SINGLE, Hidden: true,
			Deprecated:  Deprecation(cli_internal_text(size)),
			Help_Hidden: HELP_HIDDEN_ENCODED_INTERNAL,
		})
	}
}

func cli_internal_secret_boundaries() {
	buffer := make([]byte, SECRET_BUFFER_BYTES_MAX)
	secret_read("A", External_Type_State{Value: OPTION_TYPE_STRING},
		External_Enumeration{}, false, 0, buffer, nbio.Completion{})
	secret_read_path(nbio.Completion{Error: errors.New("open")})
	secret_read_path(nbio.Completion{Data: -1})
	for _, deprecated := range [...]Deprecation{"x", "xx"} {
		secrets := Active_Secrets{{Secret: Secret{Key: "AA"}}}
		warnings := make(Active_Secret_Warnings, 1)
		secret_accept(secrets, warnings, 0,
			Secret_Value_State{Parsed: true}, "AA", deprecated)
	}
	secrets := Active_Secrets{{Secret: Secret{Key: "A"}}}
	warnings := make(Active_Secret_Warnings, 1)
	cli_internal_secret_accept_failure(secrets, warnings)
	warning_storage := make([]Warning, 0, 1)
	warning_store(Writable_Deprecation_Warnings(warning_storage), Warning{})
	cli_internal_secret_close_boundaries()
	cli_internal_secret_work_maximum()
}

func cli_internal_secret_accept_failure(
	secrets Active_Secrets, warnings Active_Secret_Warnings,
) {
	defer func() { recover() }()
	secret_accept(secrets, warnings, 0, Secret_Value_State{}, "A", "")
}

func cli_internal_secret_close_boundaries() {
	failure := Secret_Content_Failure{
		Kind: Secret_Content_Failure_Kind_Value(PATH_FAILURE_KIND_EMPTY),
	}
	secret_close_path("/A", make([]Path_Failure, 0, 1), true,
		failure, true, nil)
	secret_close_path("/A", make([]Path_Failure, 0, 1), false,
		failure, false, nil)
	secret_close_path("/A", make([]Path_Failure, 1, 2), true,
		failure, false, nil)
	secret_close_path("/A", make([]Path_Failure, 2, 3), true,
		failure, false, nil)
	secret_close_path("/A",
		make([]Path_Failure, slices.COUNT_MAXIMUM-1, slices.COUNT_MAXIMUM),
		true, failure, false, nil)
	secret_record_path("/A", make([]Path_Failure, 0, 1), false,
		Current_Failure{}, false)
}

func cli_internal_secret_work_maximum() {
	secrets := make(Active_Secrets, slices.COUNT_MAXIMUM)
	parsers := make(Active_Secret_Parsers, slices.COUNT_MAXIMUM)
	for index := range secrets {
		secrets[index].Paths = Secret_Paths{"/A"}
		parsers[index].Completion.Callback = secret_operation_complete
	}
	secret_advance_all(
		secrets, parsers, Secret_IO{},
		make(Active_Secret_Failures, slices.COUNT_MAXIMUM),
		make(Active_Secret_Warnings, slices.COUNT_MAXIMUM),
	)
}

func cli_internal_empty_keys() {
	record := Failure_Size{}
	failure := Failure_Writer{
		Storage: make([]byte, FAILURE_SIZE_MAXIMUM), Size: &record,
	}
	environment_enum_failure_write(
		failure, "", External_Enumeration{String: []string{"x"}},
		"y", external_string_enum_failure,
	)
	record.Value = 0
	failure_write_path(failure, Failure_Secret_Key{}, Path_Failure{
		Path: Path_Failure_Path{Value: "/A"},
		Kind: Path_Failure_Kind{Value: PATH_FAILURE_KIND_NONE},
	})
	buffer := make([]byte, SECRET_BUFFER_BYTES_MAX)
	secret_read("", External_Type_State{Value: OPTION_TYPE_STRING},
		External_Enumeration{}, false, 0, buffer, nbio.Completion{})
	secrets := Active_Secrets{{}}
	secret_accept(secrets, make(Active_Secret_Warnings, 1), 0,
		Secret_Value_State{Parsed: true}, "", "")
}
