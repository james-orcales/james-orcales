package shell_sdk_test

import (
	"bytes"
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
