package shell_sdk_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	shell_sdk "local/james-orcales/shell_sdk/internal"
)

// Test_Dispatch_Unknown_Verb checks that an unrecognized invoking name is a
// usage error.
func Test_Dispatch_Unknown_Verb(t *testing.T) {
	output := strings.Builder{}
	error_output := strings.Builder{}
	code := shell_sdk.Main(&shell_sdk.Main_Input{
		Arguments:    []string{"bogus"},
		Output:       &output,
		Error_Output: &error_output,
	})
	if code != 2 {
		t.Fatalf("an unknown verb should exit 2, got %d", code)
	}
}

// Test_Dispatch_Help checks that -help prints a verb's usage and succeeds even when
// the verb's required arguments are absent.
func Test_Dispatch_Help(t *testing.T) {
	output, code := drive_verb("filter", []string{"-help"}, nil)
	if code != 0 {
		t.Fatalf("help should exit 0, got %d", code)
	}
	if !strings.Contains(output, "filter") {
		t.Fatalf("expected the verb usage, got %q", output)
	}
}

// Test_Dispatch_Enum_Argument checks a verb argument confined to a fixed set rejects an
// out-of-set value as a usage error.
func Test_Dispatch_Enum_Argument(t *testing.T) {
	_, code := drive_verb("from", []string{"bogus"}, nil)
	if code != 2 {
		t.Fatalf("an out-of-set format should exit 2, got %d", code)
	}
}

// Test_Wire_Round_Trip checks that decoding an encoded value reproduces it, by
// re-encoding the decode and comparing the bytes.
func Test_Wire_Round_Trip(t *testing.T) {
	tags := shell_sdk.List_Value([]shell_sdk.Value{shell_sdk.String_Value("x")})
	original := shell_sdk.Record_Value([]shell_sdk.Field{
		{Name: "name", Value: shell_sdk.String_Value("ada")},
		{Name: "age", Value: shell_sdk.Number_Value("36")},
		{Name: "tags", Value: tags},
	})
	encoded := shell_sdk.Wire_Encode(original)
	decoded, decode_err := shell_sdk.Wire_Decode(encoded)
	if decode_err != nil {
		t.Fatalf("decode: %v", decode_err)
	}
	if !bytes.Equal(encoded, shell_sdk.Wire_Encode(decoded)) {
		t.Fatalf("round trip changed the encoding")
	}
}

// Test_Json_Round_Trip checks that parsing then emitting reproduces the compact
// input, preserving key order.
func Test_Json_Round_Trip(t *testing.T) {
	input := `[{"name":"ada","age":36},{"name":"bob","age":19}]`
	parsed, parse_err := shell_sdk.Json_Parse([]byte(input))
	if parse_err != nil {
		t.Fatalf("parse: %v", parse_err)
	}
	emitted := shell_sdk.Json_Emit(parsed)
	if emitted != input {
		t.Fatalf("emit changed the text: %s", emitted)
	}
}

// Test_Csv_Round_Trip checks that parsing CSV then emitting reproduces the text,
// with the header naming the columns.
func Test_Csv_Round_Trip(t *testing.T) {
	input := "name,age\nada,36\nbob,19\n"
	parsed, parse_err := shell_sdk.Csv_Parse([]byte(input))
	if parse_err != nil {
		t.Fatalf("parse: %v", parse_err)
	}
	emitted, emit_err := shell_sdk.Csv_Emit(parsed)
	if emit_err != nil {
		t.Fatalf("emit: %v", emit_err)
	}
	if emitted != input {
		t.Fatalf("emit changed the text: %q", emitted)
	}
}

// Test_Input_Empty checks that empty standard input decodes to a null value.
func Test_Input_Empty(t *testing.T) {
	output, code := drive_verb("to", []string{"json"}, []byte{})
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if output != "null\n" {
		t.Fatalf("expected null, got %q", output)
	}
}

// Test_Input_Limit checks that an over-limit read from the injected reader becomes a
// runtime failure, not a truncated value.
func Test_Input_Limit(t *testing.T) {
	output := strings.Builder{}
	problems := strings.Builder{}
	code := shell_sdk.Main(&shell_sdk.Main_Input{
		Arguments:    []string{"count"},
		Output:       &output,
		Error_Output: &problems,
		Read_Stdin: func() (data []byte, err error) {
			return nil, errors.New("input exceeds 256 bytes")
		},
		Read_File: func(name string) (data []byte, err error) { return nil, nil },
	})
	if code != 1 {
		t.Fatalf("an over-limit read should exit 1, got %d", code)
	}
	if !strings.Contains(problems.String(), "exceeds") {
		t.Fatalf("expected the over-limit error, got %q", problems.String())
	}
}

// Test_Verbs_Sort_By checks that sort-by orders records by a numeric field.
func Test_Verbs_Sort_By(t *testing.T) {
	stdin := []byte(`[{"v":3},{"v":1},{"v":2}]`)
	output, _ := drive_verb("sort-by", []string{"v", "-json"}, stdin)
	if output != `[{"v":1},{"v":2},{"v":3}]` {
		t.Fatalf("got %s", output)
	}
}

// Test_Verbs_Group_By checks that group-by buckets records in first-seen order.
func Test_Verbs_Group_By(t *testing.T) {
	stdin := []byte(`[{"d":"x","n":1},{"d":"y","n":2},{"d":"x","n":3}]`)
	output, _ := drive_verb("group-by", []string{"d", "-json"}, stdin)
	want := `{"x":[{"d":"x","n":1},{"d":"x","n":3}],"y":[{"d":"y","n":2}]}`
	if output != want {
		t.Fatalf("got %s", output)
	}
}

// Test_Verbs_Distinct checks that distinct keeps the first of each equal element.
func Test_Verbs_Distinct(t *testing.T) {
	output, _ := drive_verb("distinct", []string{"-json"}, []byte(`[1,2,2,3,1]`))
	if output != `[1,2,3]` {
		t.Fatalf("got %s", output)
	}
}

// Test_Verbs_Count checks that count returns the number of list items.
func Test_Verbs_Count(t *testing.T) {
	output, _ := drive_verb("count", []string{"-json"}, []byte(`[10,20,30]`))
	if output != "3" {
		t.Fatalf("got %s", output)
	}
}

// Test_Verbs_Shape checks that a list verb given a non-list is a runtime error.
func Test_Verbs_Shape(t *testing.T) {
	_, code := drive_verb("sort-by", []string{"v"}, []byte(`{"a":1}`))
	if code != 1 {
		t.Fatalf("a shape mismatch should exit 1, got %d", code)
	}
}
