package invariant_test

import (
	"bytes"
	"errors"
	"io/fs"
	"math"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"

	invariant "github.com/james-orcales/james-orcales/shared/invariant/v2"
	snap "github.com/james-orcales/james-orcales/shared/snap/v2/snap_default"
)

// Bundles the optional inputs to test_recorder so we can avoid Tiger Style's
// check_input_struct flagging multiple-same-type params.
type test_recorder_input struct {
	Source_Files map[string]string
	Caller_File  string
	Caller_Line  int
	Is_Test      bool
}

// Builds a Recorder wired to in-memory I/O for unit tests. Get_Caller returns
// the fixed Caller_File / Caller_Line so tests can match a pre-registered
// tracker key without depending on real runtime frames.
func test_recorder(input *test_recorder_input) (
	recorder *invariant.Recorder,
	output_buffer *bytes.Buffer,
	exit_code *atomic.Int32,
) {
	memory_file_system := fstest.MapFS{}
	for path, content := range input.Source_Files {
		memory_file_system[path] = &fstest.MapFile{Data: []byte(content)}
	}
	output_buffer = &bytes.Buffer{}
	exit_code = &atomic.Int32{}
	recorder = &invariant.Recorder{
		Output:      output_buffer,
		File_System: memory_file_system,
		Get_Caller: func(skip int) (frame_information invariant.Frame_Information, err error) {
			return invariant.Frame_Information{
				File: input.Caller_File,
				Line: input.Caller_Line,
			}, nil
		},
		Exit: func(code int) {
			exit_code.Store(int32(code))
		},
		Is_Test:             input.Is_Test,
		Packages_To_Analyze: []string{"."},
	}
	return recorder, output_buffer, exit_code
}

// Test_Ensure_Passes verifies that Ensure on a true condition does not panic
// or emit a failure message, and that the assertion's tracker entry counter
// reaches 1.
func Test_Always_Passes(t *testing.T) {
	recorder, output_buffer, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 7,
		Is_Test:     true,
	})
	recorder.Assertions.Store("/fake/test.go:7", &invariant.Assertion_Metadata{Kind: "Always", Message: "ok"})
	invariant.Recorder_Is_Always(recorder, true, "ok")
	if output_buffer.Len() != 0 {
		t.Fatalf("expected no output, got: %s", output_buffer.String())
	}
	value, _ := recorder.Assertions.Load("/fake/test.go:7")
	if value.(*invariant.Assertion_Metadata).Frequency.Load() != 1 {
		t.Fatalf("expected Frequency=1, got %d", value.(*invariant.Assertion_Metadata).Frequency.Load())
	}
}

// Test_Ensure_Fails_Panics verifies that Ensure on a false condition emits the
// failure message via Failure_Callback and panics with the prefixed message.
func Test_Always_Fails_Panics(t *testing.T) {
	recorder, _, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 7,
		Is_Test:     true,
	})
	captured := ""
	recorder.Failure_Callback = func(message string) {
		captured = message
	}
	defer func() {
		recovered := recover()
		if !invariant.Is_Assertion_Failure(recovered) {
			t.Fatalf("expected assertion failure panic, got: %v", recovered)
		}
		if !strings.Contains(captured, "test message") {
			t.Fatalf("expected captured message to contain 'test message', got: %s", captured)
		}
	}()
	invariant.Recorder_Is_Always(recorder, false, "test message")
	t.Fatal("expected Ensure to panic, but it returned")
}

// Test_Always_Fails_Includes_Location verifies that a failing Is_Always
// embeds the caller's file:line in the failure message.
func Test_Always_Fails_Includes_Location(t *testing.T) {
	recorder, _, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 7,
		Is_Test:     true,
	})
	captured := ""
	recorder.Failure_Callback = func(message string) {
		captured = message
	}
	defer func() {
		recover()
		if !strings.HasPrefix(captured, "/fake/test.go:7:0 ") {
			t.Fatalf("expected failure message to start with caller location, got: %s", captured)
		}
	}()
	invariant.Recorder_Is_Always(recorder, false, "test message")
	t.Fatal("expected Is_Always to panic")
}

// Test_Boundary_Fails_Includes_Location verifies that an out-of-range
// Recorder_Distinct_Boundary embeds the caller's file:line in the failure message.
func Test_Boundary_Fails_Includes_Location(t *testing.T) {
	recorder, _, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 42,
		Is_Test:     true,
	})
	captured := ""
	recorder.Failure_Callback = func(message string) {
		captured = message
	}
	defer func() {
		recover()
		if !strings.HasPrefix(captured, "/fake/test.go:42:0 ") {
			t.Fatalf("expected failure message to start with caller location, got: %s", captured)
		}
	}()
	invariant.Recorder_Distinct_Boundary(recorder, &invariant.Boundary_Input[int]{
		X: -1, Lo: 0, Hi: 10, Message: "Count within range",
	})
	t.Fatal("expected Boundary to panic")
}

// Test_Always_Captures_Failure verifies that Always emits the failure message
// and panics when the condition is false.
func Test_Always_Captures_Failure(t *testing.T) {
	recorder, output_buffer, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 7,
		Is_Test:     true,
	})
	defer func() {
		recover()
		if !strings.Contains(output_buffer.String(), "always fails") {
			t.Fatalf("expected output to contain failure message, got: %s", output_buffer.String())
		}
	}()
	invariant.Recorder_Is_Always(recorder, false, "always fails")
}

// Test_Sometimes_Tracks_Both verifies that Sometimes records true and false
// evaluations separately in Frequency vs False_Frequency.
func Test_Sometimes_Tracks_Both(t *testing.T) {
	recorder, _, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 9,
		Is_Test:     true,
	})
	recorder.Assertions.Store("/fake/test.go:9", &invariant.Assertion_Metadata{Kind: "Sometimes", Message: "both"})
	invariant.Recorder_Is_Sometimes(recorder, true, "both")
	invariant.Recorder_Is_Sometimes(recorder, false, "both")
	value, _ := recorder.Assertions.Load("/fake/test.go:9")
	metadata := value.(*invariant.Assertion_Metadata)
	if metadata.Frequency.Load() != 1 {
		t.Fatalf("expected Frequency=1, got %d", metadata.Frequency.Load())
	}
	if metadata.False_Frequency.Load() != 1 {
		t.Fatalf("expected False_Frequency=1, got %d", metadata.False_Frequency.Load())
	}
}

// Test_Custom_Failure_Callback verifies that injecting a callback intercepts
// every failure message instead of writing them to Output.
func Test_Custom_Failure_Callback(t *testing.T) {
	recorder, output_buffer, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 7,
		Is_Test:     true,
	})
	captured := ""
	recorder.Failure_Callback = func(message string) {
		captured = message
	}
	defer func() {
		recover()
		if !strings.Contains(captured, "intercepted") {
			t.Fatalf("expected captured to contain 'intercepted', got: %s", captured)
		}
		if output_buffer.Len() != 0 {
			t.Fatalf("expected no output (callback should suppress), got: %s", output_buffer.String())
		}
	}()
	invariant.Recorder_Is_Always(recorder, false, "intercepted")
}

// Test_Ensure_Nil_Error_Passes_On_Nil verifies the happy path.
func Test_Always_Nil_Error_Passes_On_Nil(t *testing.T) {
	recorder, output_buffer, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 7,
		Is_Test:     true,
	})
	invariant.Recorder_Is_Always_Nil_Error(recorder, nil, "no error expected")
	if output_buffer.Len() != 0 {
		t.Fatalf("expected no output, got: %s", output_buffer.String())
	}
}

// Test_Ensure_Nil_Error_Fails_On_Err verifies that a non-nil error triggers
// the failure path with the error embedded in the message.
func Test_Always_Nil_Error_Fails_On_Err(t *testing.T) {
	recorder, _, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 7,
		Is_Test:     true,
	})
	captured := ""
	recorder.Failure_Callback = func(message string) {
		captured = message
	}
	defer func() {
		recovered := recover()
		if !invariant.Is_Assertion_Failure(recovered) {
			t.Fatalf("expected assertion failure panic, got: %v", recovered)
		}
		if !strings.Contains(captured, "context") {
			t.Fatalf("expected message to contain wrapper context, got: %s", captured)
		}
		if !strings.Contains(captured, "bad error") {
			t.Fatalf("expected message to embed the error, got: %s", captured)
		}
	}()
	invariant.Recorder_Is_Always_Nil_Error(recorder, errors.New("bad error"), "context")
	t.Fatal("expected Ensure_Nil_Error to panic")
}

// Test_Until_Runaway_Panics verifies that exhausting Until's limit panics.
func Test_Until_Runaway_Panics(t *testing.T) {
	defer func() {
		recovered := recover()
		if !invariant.Is_Assertion_Failure(recovered) {
			t.Fatalf("expected runaway loop panic, got: %v", recovered)
		}
	}()
	iterations := 0
	for range invariant.Until(3) {
		iterations++
	}
	t.Fatal("expected Until to panic before completing")
}

// Test_Is_Assertion_Failure_Detects_Prefix verifies recovery detection.
func Test_Is_Assertion_Failure_Detects_Prefix(t *testing.T) {
	if !invariant.Is_Assertion_Failure(invariant.Assertion_Failure_Message_Prefix + ": something") {
		t.Fatal("expected prefixed string to be detected")
	}
	if invariant.Is_Assertion_Failure("ordinary panic") {
		t.Fatal("expected ordinary string to be rejected")
	}
	if invariant.Is_Assertion_Failure(42) {
		t.Fatal("expected non-string to be rejected")
	}
}

// Test_Register_Packages_For_Analysis_Pre_Registers verifies that the AST
// walker seeds the tracker with entries for every invariant.X call in the
// source files of registered packages.
func Test_Register_Packages_For_Analysis_Pre_Registers(t *testing.T) {
	source := `package example

import "github.com/james-orcales/james-orcales/shared/invariant/v2"

func Run(r *invariant.Recorder) {
	invariant.Recorder_Is_Always(r, true, "Registered example message holds")
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	registered := false
	recorder.Assertions.Range(func(key, value any) (continue_iter bool) {
		if value.(*invariant.Assertion_Metadata).Message == "Registered example message holds" {
			registered = true
			return false
		}
		return true
	})
	if !registered {
		t.Fatal("expected pre-registered entry for invariant.Recorder_Is_Always call")
	}
}

// Test_Register_Packages_For_Analysis_Panics_On_Non_Literal_Message
// verifies that an assertion whose message argument isn't a string literal
// (e.g., fmt.Sprintf, concatenation, a variable) is caught at registration
// time with a framework-precondition panic that names the offending file
// and line. Without this, the site is silently skipped and the user sees
// "huh, why doesn't this assertion appear in the never-fired report?".
func Test_Register_Packages_For_Analysis_Panics_On_Non_Literal_Message(t *testing.T) {
	source := `package example

import "fmt"
import "github.com/james-orcales/james-orcales/shared/invariant/v2"

func Run(r *invariant.Recorder, x int) {
	invariant.Recorder_Is_Always(r, x > 0, fmt.Sprintf("x must be positive, got %d", x))
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	defer func() {
		recovered := recover()
		recovered_string, is_string := recovered.(string)
		if !is_string {
			t.Fatalf("expected string panic, got: %T %v", recovered, recovered)
		}
		if !strings.HasPrefix(recovered_string, invariant.Framework_Precondition_Violation_Prefix) {
			t.Fatalf("expected framework-precondition prefix, got: %s", recovered_string)
		}
		if !strings.Contains(recovered_string, "Recorder_Is_Always") {
			t.Fatalf("expected message to name the assertion kind, got: %s", recovered_string)
		}
		if !strings.Contains(recovered_string, "/fake/example.go:7") {
			t.Fatalf("expected message to name the file:line of the offending call, got: %s", recovered_string)
		}
		if !strings.Contains(recovered_string, "string literal") {
			t.Fatalf("expected message to explain the literal requirement, got: %s", recovered_string)
		}
	}()
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	t.Fatal("expected Register_Packages_For_Analysis to panic on non-literal message")
}

// Test_Register_Folds_String_Literal_Concatenation verifies that a message
// expressed as `"foo " + "bar"` registers as the folded string "foo bar".
// Without this, multi-line messages can't be split without losing the
// AST-time coverage hook.
func Test_Register_Folds_String_Literal_Concatenation(t *testing.T) {
	source := `package example

import "github.com/james-orcales/james-orcales/shared/invariant/v2"

func Run(r *invariant.Recorder) {
	invariant.Recorder_Is_Always(r, true, "First half " + "second half")
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	found := false
	recorder.Assertions.Range(func(key, value any) (continue_iter bool) {
		if value.(*invariant.Assertion_Metadata).Message == "First half second half" {
			found = true
			return false
		}
		return true
	})
	if !found {
		t.Fatal("expected folded message \"First half second half\" to be registered")
	}
}

// Test_Register_Folds_Three_Way_Concatenation verifies that left-associative
// chains of `+` (`"a" + "b" + "c"` → `(("a"+"b")+"c")`) recurse correctly.
func Test_Register_Folds_Three_Way_Concatenation(t *testing.T) {
	source := `package example

import "github.com/james-orcales/james-orcales/shared/invariant/v2"

func Run(r *invariant.Recorder) {
	invariant.Recorder_Is_Always(r, true, "Alpha " + "beta " + "gamma")
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	found := false
	recorder.Assertions.Range(func(key, value any) (continue_iter bool) {
		if value.(*invariant.Assertion_Metadata).Message == "Alpha beta gamma" {
			found = true
			return false
		}
		return true
	})
	if !found {
		t.Fatal("expected folded message \"Alpha beta gamma\" to be registered")
	}
}

// Test_Register_Rejects_Non_Literal_In_Concatenation verifies that a `+`
// expression with a non-literal operand (variable, call) is rejected with
// the existing framework-precondition panic — the literal-message contract
// extends to "literal or concatenation of literals", not "any string-typed
// expression".
func Test_Register_Rejects_Non_Literal_In_Concatenation(t *testing.T) {
	source := `package example

import "github.com/james-orcales/james-orcales/shared/invariant/v2"

func Run(r *invariant.Recorder, name string) {
	invariant.Recorder_Is_Always(r, true, "Hello " + name)
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	defer func() {
		recovered := recover()
		recovered_string, is_string := recovered.(string)
		if !is_string {
			t.Fatalf("expected string panic, got: %T %v", recovered, recovered)
		}
		if !strings.HasPrefix(recovered_string, invariant.Framework_Precondition_Violation_Prefix) {
			t.Fatalf("expected framework-precondition prefix, got: %s", recovered_string)
		}
		if !strings.Contains(recovered_string, "string literal") {
			t.Fatalf("expected message to explain the literal requirement, got: %s", recovered_string)
		}
	}()
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	t.Fatal("expected Register_Packages_For_Analysis to panic on non-literal concat operand")
}

// Test_Register_Rejects_Lowercase_Start verifies that the validator rejects
// a message whose first character isn't uppercase. The panic must name the
// capital-letter rule so the author knows what to fix.
func Test_Register_Rejects_Lowercase_Start(t *testing.T) {
	source := `package example

import "github.com/james-orcales/james-orcales/shared/invariant/v2"

func Run(r *invariant.Recorder) {
	invariant.Recorder_Is_Always(r, true, "x must be positive")
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	defer func() {
		recovered := recover()
		recovered_string, is_string := recovered.(string)
		if !is_string {
			t.Fatalf("expected string panic, got: %T %v", recovered, recovered)
		}
		if !strings.HasPrefix(recovered_string, invariant.Framework_Precondition_Violation_Prefix) {
			t.Fatalf("expected framework-precondition prefix, got: %s", recovered_string)
		}
		if !strings.Contains(recovered_string, "uppercase") {
			t.Fatalf("expected message to name the capital-letter rule, got: %s", recovered_string)
		}
		if !strings.Contains(recovered_string, "x must be positive") {
			t.Fatalf("expected message to quote the offending text, got: %s", recovered_string)
		}
	}()
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	t.Fatal("expected Register_Packages_For_Analysis to panic on lowercase-start message")
}

// Test_Register_Rejects_Each_Negative_Token table-drives the negative-token
// rule over every banned form. Each fixture starts with a capital letter so
// only the negative-token rule can fire. The panic must name the offending
// token verbatim so the author knows what to rephrase.
func Test_Register_Rejects_Each_Negative_Token(t *testing.T) {
	tokens := []string{
		"not", "never", "no", "none", "cannot",
		"can't", "don't", "doesn't", "didn't", "isn't",
		"wasn't", "aren't", "weren't", "won't", "shouldn't",
		"wouldn't", "couldn't", "hasn't", "haven't", "hadn't",
		"fail", "fails", "failed", "broken", "invalid", "illegal",
	}
	for _, token := range tokens {
		t.Run(token, func(t *testing.T) {
			message := "Subject " + token + " predicate"
			source := `package example

import "github.com/james-orcales/james-orcales/shared/invariant/v2"

func Run(r *invariant.Recorder) {
	invariant.Recorder_Is_Always(r, true, "` + message + `")
}
`
			recorder, _, _ := test_recorder(&test_recorder_input{
				Source_Files: map[string]string{"fake/example.go": source},
				Is_Test:      true,
			})
			recorder.Packages_To_Analyze = []string{"/fake"}
			defer func() {
				recovered := recover()
				recovered_string, is_string := recovered.(string)
				if !is_string {
					t.Fatalf("expected string panic, got: %T %v", recovered, recovered)
				}
				if !strings.HasPrefix(recovered_string, invariant.Framework_Precondition_Violation_Prefix) {
					t.Fatalf("expected framework-precondition prefix, got: %s", recovered_string)
				}
				if !strings.Contains(recovered_string, strconv.Quote(token)) {
					t.Fatalf("expected message to name the offending token %q, got: %s", token, recovered_string)
				}
			}()
			invariant.Recorder_Register_Packages_For_Analysis(recorder)
			t.Fatalf("expected Register_Packages_For_Analysis to panic on negative token %q", token)
		})
	}
}

// Test_Register_Rejects_Negative_Token_In_Boundary_Message pins that the
// validator runs on Boundary_Input.Message too — not just on the trailing
// positional message of Is_* helpers. Without this, the rule would have a
// loophole on any input-struct-shaped axis.
func Test_Register_Rejects_Negative_Token_In_Boundary_Message(t *testing.T) {
	source := `package example

import invariant "github.com/james-orcales/james-orcales/shared/invariant/v2/invariant_default"

func Run(count int) {
	invariant.Is_Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: count, Lo: 0, Hi: 10, Message: "Count never exceeds limit",
	})
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	defer func() {
		recovered := recover()
		recovered_string, is_string := recovered.(string)
		if !is_string {
			t.Fatalf("expected string panic, got: %T %v", recovered, recovered)
		}
		if !strings.HasPrefix(recovered_string, invariant.Framework_Precondition_Violation_Prefix) {
			t.Fatalf("expected framework-precondition prefix, got: %s", recovered_string)
		}
		if !strings.Contains(recovered_string, `"never"`) {
			t.Fatalf("expected message to name the negative token %q, got: %s", "never", recovered_string)
		}
	}()
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	t.Fatal("expected Register_Packages_For_Analysis to panic on negative token in Boundary_Input.Message")
}

// Test_Register_Rejects_Too_Few_Words verifies that a message with fewer
// than three whitespace-separated tokens is rejected. One- or two-word
// messages ("OK", "Axis A") leak no information about what property is
// being claimed; three words is the floor where a subject+verb+object
// shape becomes possible.
func Test_Register_Rejects_Too_Few_Words(t *testing.T) {
	source := `package example

import "github.com/james-orcales/james-orcales/shared/invariant/v2"

func Run(r *invariant.Recorder) {
	invariant.Recorder_Is_Always(r, true, "Axis A")
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	defer func() {
		recovered := recover()
		recovered_string, is_string := recovered.(string)
		if !is_string {
			t.Fatalf("expected string panic, got: %T %v", recovered, recovered)
		}
		if !strings.HasPrefix(recovered_string, invariant.Framework_Precondition_Violation_Prefix) {
			t.Fatalf("expected framework-precondition prefix, got: %s", recovered_string)
		}
		if !strings.Contains(recovered_string, "at least 3 words") {
			t.Fatalf("expected message to name the word-count rule, got: %s", recovered_string)
		}
	}()
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	t.Fatal("expected Register_Packages_For_Analysis to panic on too-few-word message")
}

// Test_Register_Rejects_Too_Few_Characters verifies that a short three-
// word message ("A B C", 5 characters) is rejected by the minimum-length
// rule. Three words is necessary but not sufficient — a 10-character floor
// keeps single-letter or abbreviation-heavy phrasings from satisfying the
// word-count rule on a technicality.
func Test_Register_Rejects_Too_Few_Characters(t *testing.T) {
	source := `package example

import "github.com/james-orcales/james-orcales/shared/invariant/v2"

func Run(r *invariant.Recorder) {
	invariant.Recorder_Is_Always(r, true, "X y z")
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	defer func() {
		recovered := recover()
		recovered_string, is_string := recovered.(string)
		if !is_string {
			t.Fatalf("expected string panic, got: %T %v", recovered, recovered)
		}
		if !strings.HasPrefix(recovered_string, invariant.Framework_Precondition_Violation_Prefix) {
			t.Fatalf("expected framework-precondition prefix, got: %s", recovered_string)
		}
		if !strings.Contains(recovered_string, "at least 10 characters") {
			t.Fatalf("expected message to name the length rule, got: %s", recovered_string)
		}
	}()
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	t.Fatal("expected Register_Packages_For_Analysis to panic on too-short message")
}

// Test_Register_Allows_Word_Containing_Banned_Substring pins the
// word-boundary contract: substrings of larger words ("Notebook"
// containing "not", "Negative" containing "no", "Failure" containing
// "fail") must register cleanly. Without the boundary check, every
// well-formed English word that happens to contain a banned subsequence
// would be rejected.
func Test_Register_Allows_Word_Containing_Banned_Substring(t *testing.T) {
	source := `package example

import "github.com/james-orcales/james-orcales/shared/invariant/v2"

func Run(r *invariant.Recorder) {
	invariant.Recorder_Is_Always(r, true, "Notebook is open here")
	invariant.Recorder_Is_Always(r, true, "Negative axis is well-formed")
	invariant.Recorder_Is_Always(r, true, "Notification arrived intact")
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	want := map[string]bool{
		"Notebook is open here":        false,
		"Negative axis is well-formed": false,
		"Notification arrived intact":  false,
	}
	recorder.Assertions.Range(func(key, value any) (continue_iter bool) {
		message := value.(*invariant.Assertion_Metadata).Message
		if _, tracked := want[message]; tracked {
			want[message] = true
		}
		return true
	})
	for message, registered := range want {
		if !registered {
			t.Fatalf("expected message %q to register, but it was missing", message)
		}
	}
}

// Test_Analyze_Reports_Never_Fired_And_Exits verifies that an assertion which
// was pre-registered but never incremented is reported and triggers Exit(1).
// The pre-registered key uses a path the Get_Caller stub does not return, so
// the bootstrap Ensure inside Analyze can't accidentally satisfy it.
func Test_Analyze_Reports_Never_Fired_And_Exits(t *testing.T) {
	recorder, output_buffer, exit_code := test_recorder(&test_recorder_input{
		Caller_File: "/stub/caller.go",
		Caller_Line: 1,
		Is_Test:     true,
	})
	recorder.Assertions.Store("/distinct/file.go:99", &invariant.Assertion_Metadata{
		Kind: "Always", Message: "never fires", Site: "/distinct/file.go:99",
	})
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	if !strings.Contains(output_buffer.String(), "never fires") {
		t.Fatalf("expected report to mention the assertion, got: %s", output_buffer.String())
	}
	if exit_code.Load() != 1 {
		t.Fatalf("expected exit code 1, got %d", exit_code.Load())
	}
}

// Test_Skip_Outside_Test_Environment verifies that primitives no-op when
// Is_Test is false (the production case).
func Test_Skip_Outside_Test_Environment(t *testing.T) {
	recorder, _, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 7,
		Is_Test:     false,
	})
	recorder.Assertions.Store("/fake/test.go:7", &invariant.Assertion_Metadata{Kind: "Always", Message: "ok"})
	invariant.Recorder_Is_Always(recorder, true, "ok")
	value, _ := recorder.Assertions.Load("/fake/test.go:7")
	if value.(*invariant.Assertion_Metadata).Frequency.Load() != 0 {
		t.Fatalf("expected Frequency=0 outside test environment, got %d",
			value.(*invariant.Assertion_Metadata).Frequency.Load())
	}
}

// Test_File_Path_Walk_Strips_Leading_Slash verifies the leading-slash strip
// convention for absolute paths against an fs.FS root. Captures path strings
// passed through fs.WalkDir to confirm the strip happens before lookup.
func Test_File_Path_Walk_Strips_Leading_Slash(t *testing.T) {
	source := "package empty\n"
	memory_file_system := fstest.MapFS{
		"my/dir/a.go": &fstest.MapFile{Data: []byte(source)},
	}
	visited := []string{}
	err := fs.WalkDir(memory_file_system, "my/dir", func(path string, d fs.DirEntry, err error) (returned error) {
		if d == nil {
			return err
		}
		visited = append(visited, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk failed: %s", err)
	}
	if len(visited) == 0 {
		t.Fatal("expected at least one visited entry")
	}
	if !strings.Contains(strings.Join(visited, " "), "a.go") {
		t.Fatalf("expected to visit a.go, got: %v", visited)
	}
}

// Test_Custom_Working_Directory_Trim verifies that Working_Directory is
// stripped from report paths when Full_Location is false.
func Test_Custom_Working_Directory_Trim(t *testing.T) {
	recorder, output_buffer, _ := test_recorder(&test_recorder_input{
		Is_Test: true,
	})
	recorder.Working_Directory = "/home/user/project"
	recorder.Full_Location = false
	recorder.Assertions.Store("/home/user/project/pkg/file.go:42", &invariant.Assertion_Metadata{
		Kind: "Always", Message: "trimmed", Site: "/home/user/project/pkg/file.go:42",
	})
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	if strings.Contains(output_buffer.String(), "/home/user/project") {
		t.Fatalf("expected Working_Directory to be trimmed from output, got: %s", output_buffer.String())
	}
	if !strings.Contains(output_buffer.String(), "pkg/file.go:42") {
		t.Fatalf("expected trimmed remainder, got: %s", output_buffer.String())
	}
}

// Test_Frame_Format verifies that Frame_Information yields the expected
// file:line key under a stub Get_Caller. Sanity check on the helper.
func Test_Frame_Format(t *testing.T) {
	frame := invariant.Frame_Information{File: "/x/y.go", Line: 42}
	expected := "/x/y.go:" + strconv.Itoa(42)
	actual := frame.File + ":" + strconv.Itoa(frame.Line)
	if actual != expected {
		t.Fatalf("expected %q, got %q", expected, actual)
	}
}

// Test_Coverage_Snapshot_Never_Fired_Single_Ensure captures the report emitted
// when one pre-registered Ensure assertion is never exercised.
func Test_Coverage_Snapshot_Never_Fired_Single_Ensure(t *testing.T) {
	recorder, output_buffer, _ := test_recorder(&test_recorder_input{
		Caller_File: "/stub/caller.go",
		Caller_Line: 1,
		Is_Test:     true,
	})
	recorder.Assertions.Store("/pkg/foo.go:42", &invariant.Assertion_Metadata{
		Kind:    "Always",
		Message: "input must be valid",
		Site:    "/pkg/foo.go:42",
	})
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	snap.Expect(t, snap.Init(`TEST SUITE FAILED: 🚨 1 coverage gap at 1 site 🚨

# Reachability gaps
/pkg/foo.go:42  Always — line never reached: "input must be valid"
TEST SUITE FAILED: 🚨 1 coverage gap at 1 site 🚨
`), output_buffer.String())
}

// Test_Coverage_Snapshot_Never_Fired_Multiple captures the sorted ordering of
// multiple never-fired assertions.
func Test_Coverage_Snapshot_Never_Fired_Multiple(t *testing.T) {
	recorder, output_buffer, _ := test_recorder(&test_recorder_input{
		Caller_File: "/stub/caller.go",
		Caller_Line: 1,
		Is_Test:     true,
	})
	recorder.Assertions.Store("/pkg/b.go:10", &invariant.Assertion_Metadata{
		Kind: "Always", Message: "b check", Site: "/pkg/b.go:10",
	})
	recorder.Assertions.Store("/pkg/a.go:5", &invariant.Assertion_Metadata{
		Kind: "Always", Message: "a check", Site: "/pkg/a.go:5",
	})
	recorder.Assertions.Store("/pkg/c.go:20", &invariant.Assertion_Metadata{
		Kind: "X_Always", Message: "c branch", Site: "/pkg/c.go:20",
	})
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	snap.Expect(t, snap.Init(`TEST SUITE FAILED: 🚨 3 coverage gaps at 3 sites 🚨

# Reachability gaps
/pkg/a.go:5  Always — line never reached: "a check"
/pkg/b.go:10  Always — line never reached: "b check"
/pkg/c.go:20  X_Always — line never reached: "c branch"
TEST SUITE FAILED: 🚨 3 coverage gaps at 3 sites 🚨
`), output_buffer.String())
}

// Test_Coverage_Snapshot_Never_False_Sometimes captures the never-false report
// for a Sometimes assertion whose true-branch fired but whose false-branch
// never did.
func Test_Coverage_Snapshot_Never_False_Sometimes(t *testing.T) {
	recorder, output_buffer, _ := test_recorder(&test_recorder_input{
		Caller_File: "/stub/caller.go",
		Caller_Line: 1,
		Is_Test:     true,
	})
	metadata := &invariant.Assertion_Metadata{Kind: "Sometimes", Message: "rare path", Site: "/pkg/edge.go:88"}
	metadata.Frequency.Store(7)
	recorder.Assertions.Store("/pkg/edge.go:88", metadata)
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	snap.Expect(t, snap.Init(`TEST SUITE FAILED: 🚨 1 coverage gap at 1 site 🚨

# Branch gaps
/pkg/edge.go:88  Sometimes — false branch unobserved: "rare path"
TEST SUITE FAILED: 🚨 1 coverage gap at 1 site 🚨
`), output_buffer.String())
}

// Test_Coverage_Snapshot_Both_Never_Fired_And_Never_False captures the
// combined report when both classes of failure occur in the same run.
func Test_Coverage_Snapshot_Both_Never_Fired_And_Never_False(t *testing.T) {
	recorder, output_buffer, _ := test_recorder(&test_recorder_input{
		Caller_File: "/stub/caller.go",
		Caller_Line: 1,
		Is_Test:     true,
	})
	recorder.Assertions.Store("/pkg/foo.go:42", &invariant.Assertion_Metadata{
		Kind: "Always", Message: "always check", Site: "/pkg/foo.go:42",
	})
	some_metadata := &invariant.Assertion_Metadata{Kind: "Sometimes", Message: "rare path", Site: "/pkg/edge.go:88"}
	some_metadata.Frequency.Store(3)
	recorder.Assertions.Store("/pkg/edge.go:88", some_metadata)
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	snap.Expect(t, snap.Init(`TEST SUITE FAILED: 🚨 2 coverage gaps at 2 sites 🚨

# Branch gaps
/pkg/edge.go:88  Sometimes — false branch unobserved: "rare path"

# Reachability gaps
/pkg/foo.go:42  Always — line never reached: "always check"
TEST SUITE FAILED: 🚨 2 coverage gaps at 2 sites 🚨
`), output_buffer.String())
}

// Test_Coverage_Snapshot_Working_Directory_Trimmed captures the report after
// Working_Directory is stripped from the absolute paths.
func Test_Coverage_Snapshot_Working_Directory_Trimmed(t *testing.T) {
	recorder, output_buffer, _ := test_recorder(&test_recorder_input{
		Caller_File: "/stub/caller.go",
		Caller_Line: 1,
		Is_Test:     true,
	})
	recorder.Working_Directory = "/home/user/project"
	recorder.Assertions.Store("/home/user/project/pkg/foo.go:42", &invariant.Assertion_Metadata{
		Kind: "Always", Message: "trimmed path", Site: "/home/user/project/pkg/foo.go:42",
	})
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	snap.Expect(t, snap.Init(`TEST SUITE FAILED: 🚨 1 coverage gap at 1 site 🚨

# Reachability gaps
/pkg/foo.go:42  Always — line never reached: "trimmed path"
TEST SUITE FAILED: 🚨 1 coverage gap at 1 site 🚨
`), output_buffer.String())
}

// Test_Failure_Snapshot_Always captures the failure message Always emits and
// the panic on violation.
func Test_Failure_Snapshot_Always(t *testing.T) {
	recorder, output_buffer, _ := test_recorder(&test_recorder_input{
		Caller_File: "/stub/caller.go",
		Caller_Line: 1,
		Is_Test:     true,
	})
	defer func() {
		recover()
		snap.Expect(t, snap.Init(`/stub/caller.go:1:0 🚨 Assertion Failure 🚨: balance never goes negative
`), output_buffer.String())
	}()
	invariant.Recorder_Is_Always(recorder, false, "balance never goes negative")
}

// Test_Failure_Snapshot_Always_Nil_Error captures the err-embedded format
// emitted when the error is non-nil. Panics like Always.
func Test_Failure_Snapshot_Always_Nil_Error(t *testing.T) {
	recorder, output_buffer, _ := test_recorder(&test_recorder_input{
		Caller_File: "/stub/caller.go",
		Caller_Line: 1,
		Is_Test:     true,
	})
	defer func() {
		recover()
		snap.Expect(t, snap.Init(`/stub/caller.go:1:0 🚨 Assertion Failure 🚨: error must be nil. got "conn refused". reading config
`), output_buffer.String())
	}()
	invariant.Recorder_Is_Always_Nil_Error(recorder, errors.New("conn refused"), "reading config")
}

// Test_Failure_Snapshot_Unreachable captures Unreachable's failure message.
func Test_Failure_Snapshot_Unreachable(t *testing.T) {
	recorder, output_buffer, _ := test_recorder(&test_recorder_input{
		Caller_File: "/stub/caller.go",
		Caller_Line: 1,
		Is_Test:     true,
	})
	defer func() {
		recover()
		snap.Expect(t, snap.Init(`🚨 Assertion Failure 🚨: switch default for action: surprise
`), output_buffer.String())
	}()
	invariant.Recorder_Unreachable(recorder, "switch default for action: surprise")
}

// Test_Failure_Snapshot_Unimplemented captures Unimplemented's failure message.
func Test_Failure_Snapshot_Unimplemented(t *testing.T) {
	recorder, output_buffer, _ := test_recorder(&test_recorder_input{
		Caller_File: "/stub/caller.go",
		Caller_Line: 1,
		Is_Test:     true,
	})
	defer func() {
		recover()
		snap.Expect(t, snap.Init(`🚨 Assertion Failure 🚨: TODO: implement IPv6 path
`), output_buffer.String())
	}()
	invariant.Recorder_Unimplemented(recorder, "TODO: implement IPv6 path")
}

// Test_Failure_Snapshot_Always_Fatal_Failures verifies that with Fatal_Failures
// enabled, a violated Always exits with code 1 (via the injected Exit stub)
// instead of panicking — yet still emits the failure message to Output.
func Test_Failure_Snapshot_Always_Fatal_Failures(t *testing.T) {
	recorder, output_buffer, exit_code := test_recorder(&test_recorder_input{
		Caller_File: "/stub/caller.go",
		Caller_Line: 1,
		Is_Test:     true,
	})
	recorder.Fatal_Failures = true
	invariant.Recorder_Is_Always(recorder, false, "fatal mode triggered")
	snap.Expect(t, snap.Init(`/stub/caller.go:1:0 🚨 Assertion Failure 🚨: fatal mode triggered
`), output_buffer.String())
	if exit_code.Load() != 1 {
		t.Fatalf("expected exit code 1 under Fatal_Failures, got %d", exit_code.Load())
	}
}

// Test_Failure_Snapshot_Unreachable_Fatal_Failures verifies Unreachable also
// honors Fatal_Failures.
func Test_Failure_Snapshot_Unreachable_Fatal_Failures(t *testing.T) {
	recorder, output_buffer, exit_code := test_recorder(&test_recorder_input{
		Caller_File: "/stub/caller.go",
		Caller_Line: 1,
		Is_Test:     true,
	})
	recorder.Fatal_Failures = true
	invariant.Recorder_Unreachable(recorder, "impossible state")
	snap.Expect(t, snap.Init(`🚨 Assertion Failure 🚨: impossible state
`), output_buffer.String())
	if exit_code.Load() != 1 {
		t.Fatalf("expected exit code 1 under Fatal_Failures, got %d", exit_code.Load())
	}
}

// Counts the Cross-kind entries the analyzer registered for a source. Used
// by the constraint-filter tests below to assert that the registration
// loop honors Always-derived tuple exclusions instead of registering
// every cross-product combination.
func count_cross_entries(t *testing.T, source string) (count int) {
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	recorder.Assertions.Range(func(key, value any) (continue_iter bool) {
		if value.(*invariant.Assertion_Metadata).Kind == "Cross" {
			count++
		}
		return true
	})
	return count
}

// Test_Coverage_Snapshot_Cross_Never_Fired captures the report for a
// Cross site where one cell of the 2×2 cross-product was never hit.
func Test_Coverage_Snapshot_Cross_Never_Fired(t *testing.T) {
	recorder, output_buffer, _ := test_recorder(&test_recorder_input{
		Caller_File: "/stub/caller.go",
		Caller_Line: 1,
		Is_Test:     true,
	})
	recorder.Assertions.Store("/pkg/foo.go:42:tuple=(0,0)", &invariant.Assertion_Metadata{
		Kind: "Cross", Site: "/pkg/foo.go:42",
		Cross_Buckets: []invariant.Cross_Bucket{
			{Composable_Kind: "Sometimes", Value_Expression: "x", Bucket_Name: "x==false"},
			{Composable_Kind: "Sometimes", Value_Expression: "y", Bucket_Name: "y==false"},
		},
		Tuple_Indices: []int{0, 0},
	})
	recorder.Assertions.Store("/pkg/foo.go:42:tuple=(1,1)", &invariant.Assertion_Metadata{
		Kind: "Cross", Site: "/pkg/foo.go:42",
		Cross_Buckets: []invariant.Cross_Bucket{
			{Composable_Kind: "Sometimes", Value_Expression: "x", Bucket_Name: "x==true"},
			{Composable_Kind: "Sometimes", Value_Expression: "y", Bucket_Name: "y==true"},
		},
		Tuple_Indices: []int{1, 1},
	})
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	snap.Expect(t, snap.Init(`TEST SUITE FAILED: 🚨 2 coverage gaps at 1 site 🚨

# Cross-product gaps

/pkg/foo.go:42  Cross_Product — 2 of 2 tuples never observed
  axes:
    Sometimes x
    Sometimes y
  missing tuples:
    (x==false, y==false)
    (x==true, y==true)
TEST SUITE FAILED: 🚨 2 coverage gaps at 1 site 🚨
`), output_buffer.String())
}

// Test_Cross_Product_All_Tier1_Helpers_AST verifies the analyzer pre-registers
// the cross-product when multiple Sometimes axes appear together.
func Test_Cross_Product_All_Tier1_Helpers_AST(t *testing.T) {
	source := `package example

import invariant "github.com/james-orcales/james-orcales/shared/invariant/v2/invariant_default"

func F(a bool, b bool, c bool) {
	invariant.Cross_Product(
		invariant.Sometimes(a, "First axis is exercised"),
		invariant.Sometimes(b, "Second axis is exercised"),
		invariant.Sometimes(c, "Third axis is exercised"),
	)
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	count := 0
	recorder.Assertions.Range(func(key, value any) (continue_iter bool) {
		if value.(*invariant.Assertion_Metadata).Kind == "Cross" {
			count++
		}
		return true
	})
	// Three 2-bucket Sometimes axes → 2^3 = 8 cells.
	if count != 8 {
		t.Fatalf("expected 8 Cross entries (2^3 from three Sometimes axes), got %d", count)
	}
}

// Test_Boundary_Returns_Bucket_For_Each_Position pins Boundary's runtime
// contract under design C. Bucket_Index 0 with Bucket_Name "Lo" when X equals
// Lo; index 1 with name "Hi" when X equals Hi; sentinel index -1 for any
// interior value so the Cross_Product key lookup misses on non-endpoint
// observations (interior calls satisfy the Always part but contribute no
// coverage). The bucket name carries no literal value or source expression —
// the AST analyzer pre-registers off the helper's identity alone.
func Test_Boundary_Returns_Bucket_For_Each_Position(t *testing.T) {
	recorder, _, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 7,
		Is_Test:     true,
	})
	at_lo := invariant.Recorder_Distinct_Boundary(recorder, &invariant.Boundary_Input[int]{
		X: 0, Lo: 0, Hi: 10, Message: "count within window",
	})
	if at_lo.Axis_Kind != "Distinct_Boundary" {
		t.Fatalf("expected Axis_Kind=Boundary at Lo, got %q", at_lo.Axis_Kind)
	}
	if at_lo.Bucket_Index != 0 {
		t.Fatalf("expected Bucket_Index=0 at Lo, got %d", at_lo.Bucket_Index)
	}
	if at_lo.Bucket_Name != "Lo" {
		t.Fatalf("expected Bucket_Name=Lo, got %q", at_lo.Bucket_Name)
	}
	at_hi := invariant.Recorder_Distinct_Boundary(recorder, &invariant.Boundary_Input[int]{
		X: 10, Lo: 0, Hi: 10, Message: "count within window",
	})
	if at_hi.Bucket_Index != 1 {
		t.Fatalf("expected Bucket_Index=1 at Hi, got %d", at_hi.Bucket_Index)
	}
	if at_hi.Bucket_Name != "Hi" {
		t.Fatalf("expected Bucket_Name=Hi, got %q", at_hi.Bucket_Name)
	}
	interior := invariant.Recorder_Distinct_Boundary(recorder, &invariant.Boundary_Input[int]{
		X: 5, Lo: 0, Hi: 10, Message: "count within window",
	})
	if interior.Bucket_Index != -1 {
		t.Fatalf("expected Bucket_Index=-1 for interior, got %d", interior.Bucket_Index)
	}
}

// Test_Boundary_Fails_When_X_Outside_Range verifies the Always portion: an
// X outside [Lo, Hi] panics with an Assertion_Failure-prefixed message
// embedding the user annotation. No tracker entry is needed for this path —
// an outside-range observation is a fail-fatal, not a coverage observation.
func Test_Boundary_Fails_When_X_Outside_Range(t *testing.T) {
	recorder, _, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 7,
		Is_Test:     true,
	})
	captured := ""
	recorder.Failure_Callback = func(message string) {
		captured = message
	}
	defer func() {
		recovered := recover()
		if !invariant.Is_Assertion_Failure(recovered) {
			t.Fatalf("expected assertion failure panic, got: %v", recovered)
		}
		if !strings.Contains(captured, "count within window") {
			t.Fatalf("expected captured message to contain user annotation, got: %s", captured)
		}
	}()
	invariant.Recorder_Distinct_Boundary(recorder, &invariant.Boundary_Input[int]{
		X: -1, Lo: 0, Hi: 10, Message: "count within window",
	})
	t.Fatal("expected Boundary to panic on out-of-range value")
}

// Test_Cross_Product_Boundary_AST verifies the analyzer pre-registers exactly
// two Cross entries for a Boundary axis inside a Cross_Product — one with
// Bucket_Name "Lo" at tuple=(0), one with "Hi" at tuple=(1). The Composable_Kind
// is "Distinct_Boundary"; Value_Expression captures the X-field source ("count"); the
// Message field literal flows through to Axis_Message. The Lo/Hi argument
// expressions themselves are never consulted (design C): only the helper
// identity determines bucket count and naming.
func Test_Cross_Product_Boundary_AST(t *testing.T) {
	source := `package example

import invariant "github.com/james-orcales/james-orcales/shared/invariant/v2/invariant_default"

func F(count int) {
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: count, Lo: 0, Hi: 10, Message: "Count within window",
		}),
	)
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	boundary_entries := []*invariant.Assertion_Metadata{}
	recorder.Assertions.Range(func(key, value any) (continue_iter bool) {
		metadata := value.(*invariant.Assertion_Metadata)
		if metadata.Kind != "Cross" {
			return true
		}
		if len(metadata.Cross_Buckets) != 1 {
			return true
		}
		if metadata.Cross_Buckets[0].Composable_Kind != "Distinct_Boundary" {
			return true
		}
		boundary_entries = append(boundary_entries, metadata)
		return true
	})
	if len(boundary_entries) != 2 {
		t.Fatalf("expected 2 Boundary tuple entries, got %d", len(boundary_entries))
	}
	found_lo := false
	found_hi := false
	for _, metadata := range boundary_entries {
		bucket := metadata.Cross_Buckets[0]
		if bucket.Value_Expression != "count" {
			t.Fatalf("expected Value_Expression=count, got %q", bucket.Value_Expression)
		}
		if bucket.Axis_Message != "Count within window" {
			t.Fatalf("expected Axis_Message=%q, got %q", "Count within window", bucket.Axis_Message)
		}
		switch bucket.Bucket_Name {
		case "Lo":
			found_lo = true
			if len(metadata.Tuple_Indices) != 1 {
				t.Fatalf("expected Tuple_Indices=[0] for Lo, got %v", metadata.Tuple_Indices)
			}
			if metadata.Tuple_Indices[0] != 0 {
				t.Fatalf("expected Tuple_Indices=[0] for Lo, got %v", metadata.Tuple_Indices)
			}
		case "Hi":
			found_hi = true
			if len(metadata.Tuple_Indices) != 1 {
				t.Fatalf("expected Tuple_Indices=[1] for Hi, got %v", metadata.Tuple_Indices)
			}
			if metadata.Tuple_Indices[0] != 1 {
				t.Fatalf("expected Tuple_Indices=[1] for Hi, got %v", metadata.Tuple_Indices)
			}
		default:
			t.Fatalf("unexpected Bucket_Name %q (design C names only Lo and Hi)", bucket.Bucket_Name)
		}
	}
	if !found_lo {
		t.Fatal("expected a Boundary entry with Bucket_Name=Lo")
	}
	if !found_hi {
		t.Fatal("expected a Boundary entry with Bucket_Name=Hi")
	}
}

// Test_Cross_Product_Per_Axis_Registration verifies that, alongside tuple
// entries, the analyzer registers one entry per axis inside a Cross_Product
// call. Each axis is an independent property and should be tracked as such
// — the tuple count alone collapses a 5-axis Always Cross_Product into one
// "tested" credit, hiding the four other properties being asserted.
//
// Axis entries are keyed by `<site>:axis=N` (and for Boundary, suffixed with
// `:property=<lo_ge|hi_le|lo_hit|hi_hit>`). A Boundary axis decomposes into
// four properties: two Always (bound enforcement on each side) and two
// Sometimes (endpoint observation).
func Test_Cross_Product_Per_Axis_Registration(t *testing.T) {
	source := `package example

import invariant "github.com/james-orcales/james-orcales/shared/invariant/v2/invariant_default"

func F(p *int, flag bool, count int) {
	invariant.Cross_Product(
		invariant.Always(p != nil, "Pointer is non-nil"),
		invariant.Sometimes(flag, "Flag is set sometimes"),
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: count, Lo: 0, Hi: 10, Message: "Count within window",
		}),
	)
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	invariant.Recorder_Register_Packages_For_Analysis(recorder)

	axis_entries := map[string]*invariant.Assertion_Metadata{}
	recorder.Assertions.Range(func(key, value any) (continue_iter bool) {
		metadata := value.(*invariant.Assertion_Metadata)
		switch metadata.Kind {
		case "Cross_Axis_Always", "Cross_Axis_Sometimes":
			axis_entries[key.(string)] = metadata
		}
		return true
	})

	expected_keys := []string{
		"/fake/example.go:6:axis=0",
		"/fake/example.go:6:axis=1",
		"/fake/example.go:6:axis=2:property=lo_ge",
		"/fake/example.go:6:axis=2:property=hi_le",
		"/fake/example.go:6:axis=2:property=lo_hit",
		"/fake/example.go:6:axis=2:property=hi_hit",
	}
	if len(axis_entries) != len(expected_keys) {
		got := make([]string, 0, len(axis_entries))
		for key := range axis_entries {
			got = append(got, key)
		}
		t.Fatalf("expected %d axis entries, got %d: %v", len(expected_keys), len(axis_entries), got)
	}
	for _, key := range expected_keys {
		if _, ok := axis_entries[key]; !ok {
			t.Errorf("missing expected axis entry %q", key)
		}
	}
	expected_kinds := map[string]string{
		"/fake/example.go:6:axis=0":                 "Cross_Axis_Always",
		"/fake/example.go:6:axis=1":                 "Cross_Axis_Sometimes",
		"/fake/example.go:6:axis=2:property=lo_ge":  "Cross_Axis_Always",
		"/fake/example.go:6:axis=2:property=hi_le":  "Cross_Axis_Always",
		"/fake/example.go:6:axis=2:property=lo_hit": "Cross_Axis_Sometimes",
		"/fake/example.go:6:axis=2:property=hi_hit": "Cross_Axis_Sometimes",
	}
	for key, want_kind := range expected_kinds {
		metadata, ok := axis_entries[key]
		if !ok {
			continue
		}
		if metadata.Kind != want_kind {
			t.Errorf("entry %q: want Kind=%s, got %s", key, want_kind, metadata.Kind)
		}
	}
}

// Test_Distinct_Boundary_Fails_When_Lo_Ge_Hi pins the user-domain
// precondition: Distinct_Boundary requires Lo < Hi. Lo == Hi (degenerate
// single-point) and Lo > Hi (inverted) both fail-fatal with the standard
// Assertion_Failure_Message_Prefix. The runtime check stops the call
// before the bound-range or bucket logic runs, so the rest of the
// machinery (handle pool, Cross_Product key lookup) never observes the
// degenerate shape.
func Test_Distinct_Boundary_Fails_When_Lo_Ge_Hi(t *testing.T) {
	cases := []struct {
		Name    string
		Lo      int
		Hi      int
		Want_In string
	}{
		{Name: "Lo equals Hi", Lo: 5, Hi: 5, Want_In: "requires Lo < Hi; got Lo=5, Hi=5"},
		{Name: "Lo greater than Hi", Lo: 10, Hi: 3, Want_In: "requires Lo < Hi; got Lo=10, Hi=3"},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			recorder, _, _ := test_recorder(&test_recorder_input{Is_Test: true})
			captured := ""
			recorder.Failure_Callback = func(message string) { captured = message }
			defer func() {
				recover()
				if !strings.Contains(captured, tc.Want_In) {
					t.Errorf("captured failure %q missing %q", captured, tc.Want_In)
				}
				if !strings.Contains(captured, invariant.Assertion_Failure_Message_Prefix) {
					t.Errorf("captured failure %q missing assertion-failure prefix", captured)
				}
			}()
			invariant.Recorder_Distinct_Boundary(recorder, &invariant.Boundary_Input[int]{
				X: tc.Lo, Lo: tc.Lo, Hi: tc.Hi, Message: "bound check",
			})
			t.Fatal("expected Distinct_Boundary to panic on Lo>=Hi")
		})
	}
}

// Test_Distinct_Boundary_Fails_When_Lo_Or_Hi_Is_NaN pins the user-domain
// precondition that float endpoints must be finite. NaN endpoints make
// Lo < Hi unsatisfiable in the IEEE-754 sense (NaN is unordered), so the
// runtime fail-fatals before bounds checks run.
func Test_Distinct_Boundary_Fails_When_Lo_Or_Hi_Is_NaN(t *testing.T) {
	nan := math.NaN()
	cases := []struct {
		Name    string
		Lo      float64
		Hi      float64
		Want_In string
	}{
		{Name: "Lo is NaN", Lo: nan, Hi: 1.0, Want_In: "requires Lo and Hi to be finite"},
		{Name: "Hi is NaN", Lo: 0.0, Hi: nan, Want_In: "requires Lo and Hi to be finite"},
		{Name: "Both NaN", Lo: nan, Hi: nan, Want_In: "requires Lo and Hi to be finite"},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			recorder, _, _ := test_recorder(&test_recorder_input{Is_Test: true})
			captured := ""
			recorder.Failure_Callback = func(message string) { captured = message }
			defer func() {
				recover()
				if !strings.Contains(captured, tc.Want_In) {
					t.Errorf("captured failure %q missing %q", captured, tc.Want_In)
				}
				if !strings.Contains(captured, invariant.Assertion_Failure_Message_Prefix) {
					t.Errorf("captured failure %q missing assertion-failure prefix", captured)
				}
			}()
			invariant.Recorder_Distinct_Boundary(recorder, &invariant.Boundary_Input[float64]{
				X: 0.5, Lo: tc.Lo, Hi: tc.Hi, Message: "bound check",
			})
			t.Fatal("expected Distinct_Boundary to panic on NaN endpoints")
		})
	}
}

// Test_Cross_Product_5_Axis_All_Always_Counts_6_Properties pins the user
// concern: a five-axis all-Always Cross_Product is FIVE individual properties
// plus ONE tuple combination — six entries, not one. Counting only tuples
// would collapse the five hard-bound checks into a single "tested" credit.
func Test_Cross_Product_5_Axis_All_Always_Counts_6_Properties(t *testing.T) {
	source := `package example

import invariant "github.com/james-orcales/james-orcales/shared/invariant/v2/invariant_default"

func F(a, b, c, d, e bool) {
	invariant.Cross_Product(
		invariant.Always(a, "First always holds"),
		invariant.Always(b, "Second always holds"),
		invariant.Always(c, "Third always holds"),
		invariant.Always(d, "Fourth always holds"),
		invariant.Always(e, "Fifth always holds"),
	)
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	invariant.Recorder_Register_Packages_For_Analysis(recorder)

	tuple_count := 0
	axis_count := 0
	recorder.Assertions.Range(func(key, value any) (continue_iter bool) {
		metadata := value.(*invariant.Assertion_Metadata)
		switch metadata.Kind {
		case "Cross":
			tuple_count++
		case "Cross_Axis_Always", "Cross_Axis_Sometimes":
			axis_count++
		}
		return true
	})
	if tuple_count != 1 {
		t.Errorf("want 1 tuple entry, got %d", tuple_count)
	}
	if axis_count != 5 {
		t.Errorf("want 5 axis entries, got %d", axis_count)
	}
	if total := tuple_count + axis_count; total != 6 {
		t.Errorf("want 6 properties total, got %d", total)
	}
}

// Test_Cross_Product_Axis_Never_Fired_Reported verifies that the never-fired
// collector picks up Cross_Axis_* entries that have Frequency=0 — the per-axis
// coverage feeds the same gap-detection pass as bare Is_* entries.
func Test_Cross_Product_Axis_Never_Fired_Reported(t *testing.T) {
	source := `package example

import invariant "github.com/james-orcales/james-orcales/shared/invariant/v2/invariant_default"

func F(flag bool) {
	invariant.Cross_Product(
		invariant.Sometimes(flag, "Flag is set sometimes"),
	)
}
`
	recorder, output_buffer, exit_code := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	if exit_code.Load() == 0 {
		t.Fatalf("expected non-zero exit code after never-fired axis entry; output:\n%s", output_buffer.String())
	}
	if !strings.Contains(output_buffer.String(), "Flag is set sometimes") {
		t.Fatalf("expected report to mention the axis message; got:\n%s", output_buffer.String())
	}
}

// Test_Cross_Product_Axis_Sometimes_Needs_Both_Branches verifies the
// never-false collector picks up Cross_Axis_Sometimes entries whose
// False_Frequency is zero — same shape as bare Sometimes.
func Test_Cross_Product_Axis_Sometimes_Needs_Both_Branches(t *testing.T) {
	recorder, output_buffer, exit_code := test_recorder(&test_recorder_input{
		Caller_File: "/stub/caller.go",
		Caller_Line: 1,
		Is_Test:     true,
	})
	metadata := &invariant.Assertion_Metadata{
		Kind:    "Cross_Axis_Sometimes",
		Message: "A is set sometimes",
		Site:    "/pkg/edge.go:88",
	}
	metadata.Frequency.Store(7)
	recorder.Assertions.Store("/pkg/edge.go:88:axis=0", metadata)
	invariant.Recorder_Analyze_Assertion_Frequency(recorder)
	if exit_code.Load() == 0 {
		t.Fatalf("expected non-zero exit after never-false on Cross_Axis_Sometimes; output:\n%s", output_buffer.String())
	}
	if !strings.Contains(output_buffer.String(), "A is set sometimes") {
		t.Fatalf("expected report to mention the axis message; got:\n%s", output_buffer.String())
	}
}

// Test_Cross_Product_Axis_Sometimes_Excluding_Preserves_Both_Branches:
// when an Excluding clause forbids one specific tuple but each axis bucket
// remains reachable via some other tuple, the axes should still register as
// Cross_Axis_Sometimes with both branches required.
func Test_Cross_Product_Axis_Sometimes_Excluding_Preserves_Both_Branches(t *testing.T) {
	source := `package example

import invariant "github.com/james-orcales/james-orcales/shared/invariant/v2/invariant_default"

func F(a bool, b bool) {
	a_axis := invariant.Sometimes(a, "A is set sometimes")
	b_axis := invariant.Sometimes(b, "B is set sometimes")
	invariant.Cross_Product(
		a_axis, b_axis,
		invariant.Excluding("True-True tuple is unreachable in this fixture", invariant.Bucket_True(a_axis), invariant.Bucket_True(b_axis)),
	)
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	invariant.Recorder_Register_Packages_For_Analysis(recorder)

	axis_a_kind := ""
	axis_b_kind := ""
	recorder.Assertions.Range(func(key, value any) (continue_iter bool) {
		metadata := value.(*invariant.Assertion_Metadata)
		switch key.(string) {
		case "/fake/example.go:8:axis=0":
			axis_a_kind = metadata.Kind
		case "/fake/example.go:8:axis=1":
			axis_b_kind = metadata.Kind
		}
		return true
	})
	if axis_a_kind != "Cross_Axis_Sometimes" {
		t.Errorf("axis a: want Cross_Axis_Sometimes (both branches reachable), got %q", axis_a_kind)
	}
	if axis_b_kind != "Cross_Axis_Sometimes" {
		t.Errorf("axis b: want Cross_Axis_Sometimes (both branches reachable), got %q", axis_b_kind)
	}
}

// Test_Cross_Product_Axis_Frequency_Increments verifies that the runtime
// increments per-axis entries alongside the tuple entry. Always-axis fires
// always; Sometimes-axis tracks both branches; Boundary-axis bumps the
// lo_ge/hi_le bound-properties unconditionally and the lo_hit/hi_hit
// endpoint-properties on the observed bucket (false-branch for the unobserved
// endpoint).
func Test_Cross_Product_Axis_Frequency_Increments(t *testing.T) {
	recorder, _, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 7,
		Is_Test:     true,
	})
	site := "/fake/test.go:7"
	axis_always := site + ":axis=0"
	axis_sometimes := site + ":axis=1"
	axis_b_lo_ge := site + ":axis=2:property=lo_ge"
	axis_b_hi_le := site + ":axis=2:property=hi_le"
	axis_b_lo_hit := site + ":axis=2:property=lo_hit"
	axis_b_hi_hit := site + ":axis=2:property=hi_hit"
	recorder.Assertions.Store(axis_always, &invariant.Assertion_Metadata{
		Kind: "Cross_Axis_Always", Site: site,
		Cross_Buckets: []invariant.Cross_Bucket{{Composable_Kind: "Always", Bucket_Name: "true"}},
	})
	recorder.Assertions.Store(axis_sometimes, &invariant.Assertion_Metadata{
		Kind: "Cross_Axis_Sometimes", Site: site,
		Cross_Buckets: []invariant.Cross_Bucket{{Composable_Kind: "Sometimes"}},
	})
	recorder.Assertions.Store(axis_b_lo_ge, &invariant.Assertion_Metadata{
		Kind: "Cross_Axis_Always", Site: site,
		Cross_Buckets: []invariant.Cross_Bucket{{Composable_Kind: "Distinct_Boundary", Bucket_Name: "lo_ge"}},
	})
	recorder.Assertions.Store(axis_b_hi_le, &invariant.Assertion_Metadata{
		Kind: "Cross_Axis_Always", Site: site,
		Cross_Buckets: []invariant.Cross_Bucket{{Composable_Kind: "Distinct_Boundary", Bucket_Name: "hi_le"}},
	})
	recorder.Assertions.Store(axis_b_lo_hit, &invariant.Assertion_Metadata{
		Kind: "Cross_Axis_Sometimes", Site: site,
		Cross_Buckets: []invariant.Cross_Bucket{{Composable_Kind: "Distinct_Boundary", Bucket_Name: "lo_hit"}},
	})
	recorder.Assertions.Store(axis_b_hi_hit, &invariant.Assertion_Metadata{
		Kind: "Cross_Axis_Sometimes", Site: site,
		Cross_Buckets: []invariant.Cross_Bucket{{Composable_Kind: "Distinct_Boundary", Bucket_Name: "hi_hit"}},
	})
	// Tuple entry (so the existing tuple-increment path also exercises).
	recorder.Assertions.Store(site+":tuple=(0,1,1)", &invariant.Assertion_Metadata{
		Kind: "Cross", Site: site,
	})
	// Runtime fire: Always (true), Sometimes (true), Boundary (X=Hi).
	invariant.Recorder_Cross_Product(recorder,
		invariant.Recorder_Always(recorder, true, "Always holds"),
		invariant.Recorder_Sometimes(recorder, true, "Sometimes true here"),
		invariant.Recorder_Distinct_Boundary(recorder, &invariant.Boundary_Input[int]{
			X: 10, Lo: 0, Hi: 10, Message: "Count at Hi",
		}),
	)
	check_freq := func(key string, want_true, want_false int64) {
		t.Helper()
		raw, ok := recorder.Assertions.Load(key)
		if !ok {
			t.Errorf("entry %q missing", key)
			return
		}
		metadata := raw.(*invariant.Assertion_Metadata)
		if got := metadata.Frequency.Load(); got != want_true {
			t.Errorf("%s: want Frequency=%d, got %d", key, want_true, got)
		}
		if got := metadata.False_Frequency.Load(); got != want_false {
			t.Errorf("%s: want False_Frequency=%d, got %d", key, want_false, got)
		}
	}
	check_freq(axis_always, 1, 0)
	check_freq(axis_sometimes, 1, 0)
	check_freq(axis_b_lo_ge, 1, 0)
	check_freq(axis_b_hi_le, 1, 0)
	check_freq(axis_b_lo_hit, 0, 1) // Lo not hit (X==Hi) → false branch fires
	check_freq(axis_b_hi_hit, 1, 0) // Hi hit
}

// Test_Is_Distinct_Boundary_Increments_Tracker_For_Endpoint verifies that the single-
// axis sugar form increments the per-bucket tracker entry at the matching
// tuple index. With X == Lo, the file:line:tuple=(0) entry's Frequency
// reaches 1 and the file:line:tuple=(1) entry stays at 0.
func Test_Is_Distinct_Boundary_Increments_Tracker_For_Endpoint(t *testing.T) {
	recorder, _, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 7,
		Is_Test:     true,
	})
	recorder.Assertions.Store("/fake/test.go:7:tuple=(0)", &invariant.Assertion_Metadata{
		Kind: "Cross", Site: "/fake/test.go:7",
		Cross_Buckets: []invariant.Cross_Bucket{{
			Composable_Kind: "Distinct_Boundary", Value_Expression: "x", Bucket_Name: "Lo",
		}},
		Tuple_Indices: []int{0},
	})
	recorder.Assertions.Store("/fake/test.go:7:tuple=(1)", &invariant.Assertion_Metadata{
		Kind: "Cross", Site: "/fake/test.go:7",
		Cross_Buckets: []invariant.Cross_Bucket{{
			Composable_Kind: "Distinct_Boundary", Value_Expression: "x", Bucket_Name: "Hi",
		}},
		Tuple_Indices: []int{1},
	})
	invariant.Recorder_Is_Distinct_Boundary(recorder, &invariant.Boundary_Input[int]{
		X: 0, Lo: 0, Hi: 10, Message: "x at lower bound",
	})
	lo_value, _ := recorder.Assertions.Load("/fake/test.go:7:tuple=(0)")
	if lo_value.(*invariant.Assertion_Metadata).Frequency.Load() != 1 {
		t.Fatalf("expected Lo Frequency=1, got %d", lo_value.(*invariant.Assertion_Metadata).Frequency.Load())
	}
	hi_value, _ := recorder.Assertions.Load("/fake/test.go:7:tuple=(1)")
	if hi_value.(*invariant.Assertion_Metadata).Frequency.Load() != 0 {
		t.Fatalf("expected Hi Frequency=0, got %d", hi_value.(*invariant.Assertion_Metadata).Frequency.Load())
	}
}

// Test_Is_Distinct_Boundary_AST verifies the analyzer pre-registers two tuple entries
// for a bare Is_Distinct_Boundary site (no Cross_Product wrapper) via the of-path.
// Same shape as the Cross_Product case but dispatched through the single-
// axis route.
func Test_Is_Distinct_Boundary_AST(t *testing.T) {
	source := `package example

import invariant "github.com/james-orcales/james-orcales/shared/invariant/v2/invariant_default"

func F(count int) {
	invariant.Is_Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: count, Lo: 0, Hi: 10, Message: "Count within window",
	})
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	boundary_entries := 0
	recorder.Assertions.Range(func(key, value any) (continue_iter bool) {
		metadata := value.(*invariant.Assertion_Metadata)
		if metadata.Kind != "Cross" {
			return true
		}
		if len(metadata.Cross_Buckets) != 1 {
			return true
		}
		if metadata.Cross_Buckets[0].Composable_Kind != "Distinct_Boundary" {
			return true
		}
		boundary_entries++
		return true
	})
	if boundary_entries != 2 {
		t.Fatalf("expected 2 Boundary tuple entries from Is_Distinct_Boundary site, got %d", boundary_entries)
	}
}

// Test_Cross_Product_Boundary_End_To_End wires AST registration and runtime
// emission together: the analyzer seeds the two tuple entries for a
// Cross_Product+Boundary site, then a runtime call to Recorder_Cross_Product
// with a Boundary record at X==Lo increments the (0) bucket. The Hi bucket
// remains at zero, showing up later as a coverage gap if the test ended here.
func Test_Cross_Product_Boundary_End_To_End(t *testing.T) {
	source := `package example

import invariant "github.com/james-orcales/james-orcales/shared/invariant/v2/invariant_default"

func F(count int) {
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: count, Lo: 0, Hi: 10, Message: "Count within window",
		}),
	)
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Caller_File:  "/fake/example.go",
		Caller_Line:  6,
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	invariant.Recorder_Cross_Product(recorder,
		invariant.Recorder_Distinct_Boundary(recorder, &invariant.Boundary_Input[int]{
			X: 0, Lo: 0, Hi: 10, Message: "count within window",
		}),
	)
	lo_value, lo_ok := recorder.Assertions.Load("/fake/example.go:6:tuple=(0)")
	if !lo_ok {
		t.Fatal("expected pre-registered Lo entry at /fake/example.go:6:tuple=(0)")
	}
	if lo_value.(*invariant.Assertion_Metadata).Frequency.Load() != 1 {
		t.Fatalf("expected Lo Frequency=1 after firing, got %d", lo_value.(*invariant.Assertion_Metadata).Frequency.Load())
	}
	hi_value, hi_ok := recorder.Assertions.Load("/fake/example.go:6:tuple=(1)")
	if !hi_ok {
		t.Fatal("expected pre-registered Hi entry at /fake/example.go:6:tuple=(1)")
	}
	if hi_value.(*invariant.Assertion_Metadata).Frequency.Load() != 0 {
		t.Fatalf("expected Hi Frequency=0 (never hit), got %d", hi_value.(*invariant.Assertion_Metadata).Frequency.Load())
	}
}

// Test_Cross_Product_Excluding_AST verifies that an Excluding clause removes
// exactly one tuple from the pre-registered tracker — registering 3 entries
// instead of the full Cartesian product's 4. The forbidden cell (enabled=false,
// absent=true) is gone; the surviving cells are (false,false), (true,false),
// (true,true).
func Test_Cross_Product_Excluding_AST(t *testing.T) {
	source := `package example

import "github.com/james-orcales/james-orcales/shared/invariant/v2"

func F(r *invariant.Recorder, a, b bool) {
	enabled := invariant.Recorder_Sometimes(r, a, "Enabled is set")
	absent := invariant.Recorder_Sometimes(r, b, "Absent is set")
	invariant.Recorder_Cross_Product(r,
		enabled, absent,
		invariant.Excluding("Enabled false implies absent false in this fixture", invariant.Bucket_False(enabled), invariant.Bucket_True(absent)),
	)
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	invariant.Recorder_Register_Packages_For_Analysis(recorder)

	cross_entries := []*invariant.Assertion_Metadata{}
	recorder.Assertions.Range(func(key, value any) (continue_iter bool) {
		metadata := value.(*invariant.Assertion_Metadata)
		if metadata.Kind == "Cross" {
			cross_entries = append(cross_entries, metadata)
		}
		return true
	})
	if len(cross_entries) != 3 {
		t.Fatalf("expected 3 Cross entries after Excluding filter, got %d", len(cross_entries))
	}
	for _, metadata := range cross_entries {
		if len(metadata.Tuple_Indices) != 2 {
			t.Fatalf("expected 2-axis tuple, got %v", metadata.Tuple_Indices)
		}
		if metadata.Tuple_Indices[0] == 0 && metadata.Tuple_Indices[1] == 1 {
			t.Fatalf("forbidden tuple (0,1) should have been filtered out, got registered: %v", metadata.Tuple_Indices)
		}
	}
}

// Test_Cross_Product_Multiple_Excluding_AST verifies that multiple Excluding
// clauses each forbid a distinct cell. With 2 Sometimes axes (product=4) and
// two Excluding clauses, exactly 2 tuples remain.
func Test_Cross_Product_Multiple_Excluding_AST(t *testing.T) {
	source := `package example

import "github.com/james-orcales/james-orcales/shared/invariant/v2"

func F(r *invariant.Recorder, p, q bool) {
	a := invariant.Recorder_Sometimes(r, p, "A axis is set")
	b := invariant.Recorder_Sometimes(r, q, "B axis is set")
	invariant.Recorder_Cross_Product(r,
		a, b,
		invariant.Excluding("Fixture forbids (false, false) tuple", invariant.Bucket_False(a), invariant.Bucket_False(b)),
		invariant.Excluding("Fixture forbids (true, true) tuple", invariant.Bucket_True(a), invariant.Bucket_True(b)),
	)
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	invariant.Recorder_Register_Packages_For_Analysis(recorder)

	cross_entries := []*invariant.Assertion_Metadata{}
	recorder.Assertions.Range(func(key, value any) (continue_iter bool) {
		metadata := value.(*invariant.Assertion_Metadata)
		if metadata.Kind == "Cross" {
			cross_entries = append(cross_entries, metadata)
		}
		return true
	})
	if len(cross_entries) != 2 {
		t.Fatalf("expected 2 Cross entries after both Excluding filters, got %d", len(cross_entries))
	}
}

// Test_Cross_Product_Boundary_Excluding_AST verifies that Bucket_Lo / Bucket_Hi
// resolve correctly against a Boundary axis inside an Excluding clause.
func Test_Cross_Product_Boundary_Excluding_AST(t *testing.T) {
	source := `package example

import "github.com/james-orcales/james-orcales/shared/invariant/v2"

func F(r *invariant.Recorder, count int, flag bool) {
	c := invariant.Recorder_Distinct_Boundary(r, &invariant.Boundary_Input[int]{
		X: count, Lo: 0, Hi: 10, Message: "Count within window",
	})
	f := invariant.Recorder_Sometimes(r, flag, "Flag axis is set")
	invariant.Recorder_Cross_Product(r,
		c, f,
		invariant.Excluding("Fixture forbids the Lo c with false f tuple", invariant.Bucket_Lo(c), invariant.Bucket_False(f)),
	)
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	invariant.Recorder_Register_Packages_For_Analysis(recorder)

	cross_entries := []*invariant.Assertion_Metadata{}
	recorder.Assertions.Range(func(key, value any) (continue_iter bool) {
		metadata := value.(*invariant.Assertion_Metadata)
		if metadata.Kind == "Cross" {
			cross_entries = append(cross_entries, metadata)
		}
		return true
	})
	if len(cross_entries) != 3 {
		t.Fatalf("expected 3 Cross entries after Boundary+Excluding filter, got %d", len(cross_entries))
	}
	for _, metadata := range cross_entries {
		if metadata.Tuple_Indices[0] == 0 && metadata.Tuple_Indices[1] == 0 {
			t.Fatalf("forbidden tuple (Lo, false)=(0,0) should have been filtered out, got registered: %v", metadata.Tuple_Indices)
		}
	}
}

// Test_Cross_Product_Excluding_Panics_On_Forbidden_Tuple verifies that the
// runtime check fail-fatals when an observation matches an Excluding clause.
// The panic message must use the assertion-failure prefix (this is a user-
// domain invariant violation, not a framework precondition violation).
func Test_Cross_Product_Excluding_Panics_On_Forbidden_Tuple(t *testing.T) {
	recorder, _, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 7,
		Is_Test:     true,
	})
	defer func() {
		recovered := recover()
		recovered_string, is_string := recovered.(string)
		if !is_string {
			t.Fatalf("expected string panic, got: %T %v", recovered, recovered)
		}
		if !strings.HasPrefix(recovered_string, invariant.Assertion_Failure_Message_Prefix) {
			t.Fatalf("expected user-domain assertion failure prefix, got: %s", recovered_string)
		}
	}()

	enabled := invariant.Recorder_Sometimes(recorder, false, "Enabled is set")
	absent := invariant.Recorder_Sometimes(recorder, true, "Absent is set")
	invariant.Recorder_Cross_Product(recorder,
		enabled, absent,
		invariant.Excluding("Enabled false implies absent false in this fixture", invariant.Bucket_False(enabled), invariant.Bucket_True(absent)),
	)
	t.Fatal("expected Cross_Product to fail-fatal on the forbidden cell observation")
}

// Test_Cross_Product_Excluding_Permits_Non_Forbidden_Tuple verifies that a
// non-matching observation passes through to the tracker increment.
func Test_Cross_Product_Excluding_Permits_Non_Forbidden_Tuple(t *testing.T) {
	recorder, _, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 7,
		Is_Test:     true,
	})
	recorder.Assertions.Store("/fake/test.go:7:tuple=(1,1)", &invariant.Assertion_Metadata{
		Kind:          "Cross",
		Site:          "/fake/test.go:7",
		Tuple_Indices: []int{1, 1},
	})

	enabled := invariant.Recorder_Sometimes(recorder, true, "Enabled is set")
	absent := invariant.Recorder_Sometimes(recorder, true, "Absent is set")
	invariant.Recorder_Cross_Product(recorder,
		enabled, absent,
		invariant.Excluding("Enabled false implies absent false in this fixture", invariant.Bucket_False(enabled), invariant.Bucket_True(absent)),
	)
	value, ok := recorder.Assertions.Load("/fake/test.go:7:tuple=(1,1)")
	if !ok {
		t.Fatal("expected tracker entry for tuple (1,1)")
	}
	if value.(*invariant.Assertion_Metadata).Frequency.Load() != 1 {
		t.Fatalf("expected Frequency=1 after non-forbidden observation, got %d", value.(*invariant.Assertion_Metadata).Frequency.Load())
	}
}

// Test_Cross_Product_Constraint_Enforces_In_Production_Mode verifies that
// the constraint check runs above the test-time gate — Is_Test=false still
// fail-fatals on a forbidden cell. This is the entire point of the feature:
// authors get production enforcement, not just test-time coverage shaping.
func Test_Cross_Product_Constraint_Enforces_In_Production_Mode(t *testing.T) {
	recorder, _, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 7,
		Is_Test:     false,
	})
	defer func() {
		recovered := recover()
		recovered_string, is_string := recovered.(string)
		if !is_string {
			t.Fatalf("expected string panic in production mode, got: %T %v", recovered, recovered)
		}
		if !strings.HasPrefix(recovered_string, invariant.Assertion_Failure_Message_Prefix) {
			t.Fatalf("expected user-domain assertion failure prefix, got: %s", recovered_string)
		}
	}()

	enabled := invariant.Recorder_Sometimes(recorder, false, "Enabled is set")
	absent := invariant.Recorder_Sometimes(recorder, true, "Absent is set")
	invariant.Recorder_Cross_Product(recorder,
		enabled, absent,
		invariant.Excluding("Enabled false implies absent false in this fixture", invariant.Bucket_False(enabled), invariant.Bucket_True(absent)),
	)
	t.Fatal("expected production-mode Cross_Product to fail-fatal on the forbidden cell")
}

// Test_Register_Framework_Panics_On_Malformed_Excluding_Clause verifies that
// an Excluding arg whose receiver is not an axis identifier (here, a literal
// function call) is caught at registration with the framework-precondition
// prefix.
//
// 🛑 If this test breaks, do NOT make the registration "tolerant" by silently
// skipping the site. The whole point of this panic is that a malformed
// constraint must fail LOUDLY — silently disabling a Cross_Product site is
// the failure mode the feature was built to prevent. Read the plan at
// .claude/plans/plan-it-and-use-tingly-parnas.md first.
func Test_Register_Framework_Panics_On_Malformed_Excluding_Clause(t *testing.T) {
	source := `package example

import "github.com/james-orcales/james-orcales/shared/invariant/v2"

func make_axis(r *invariant.Recorder) invariant.Cross_Axis {
	return invariant.Recorder_Sometimes(r, true, "Axis comes from function call")
}

func F(r *invariant.Recorder, b bool) {
	other := invariant.Recorder_Sometimes(r, b, "Other axis is set")
	invariant.Recorder_Cross_Product(r,
		other,
		invariant.Excluding("Fixture forbids the false axis tuple", invariant.Bucket_False(make_axis(r))),
	)
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	defer func() {
		recovered := recover()
		recovered_string, is_string := recovered.(string)
		if !is_string {
			t.Fatalf("expected string panic, got: %T %v", recovered, recovered)
		}
		if !strings.HasPrefix(recovered_string, invariant.Framework_Precondition_Violation_Prefix) {
			t.Fatalf("expected framework-precondition prefix, got: %s", recovered_string)
		}
	}()
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	t.Fatal("expected registration to panic on Bucket_False with non-axis receiver")
}

// Test_Register_Framework_Panics_On_Excluding_Refs_Foreign_Axis verifies that
// referencing an axis variable that isn't part of the parent Cross_Product
// call's axis list is caught at registration.
//
// 🛑 If this test breaks, do NOT make the registration "tolerant" — read the
// guard comment on Test_Register_Framework_Panics_On_Malformed_Excluding_Clause.
func Test_Register_Framework_Panics_On_Excluding_Refs_Foreign_Axis(t *testing.T) {
	source := `package example

import "github.com/james-orcales/james-orcales/shared/invariant/v2"

func F(r *invariant.Recorder, a, b, c bool) {
	x := invariant.Recorder_Sometimes(r, a, "X axis is set")
	y := invariant.Recorder_Sometimes(r, b, "Y axis is set")
	stray := invariant.Recorder_Sometimes(r, c, "Stray axis sits outside the product")
	invariant.Recorder_Cross_Product(r,
		x, y,
		invariant.Excluding("Fixture forbids the stray false tuple", invariant.Bucket_False(stray), invariant.Bucket_True(y)),
	)
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	defer func() {
		recovered := recover()
		recovered_string, is_string := recovered.(string)
		if !is_string {
			t.Fatalf("expected string panic, got: %T %v", recovered, recovered)
		}
		if !strings.HasPrefix(recovered_string, invariant.Framework_Precondition_Violation_Prefix) {
			t.Fatalf("expected framework-precondition prefix, got: %s", recovered_string)
		}
	}()
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	t.Fatal("expected registration to panic on Excluding referencing a foreign axis")
}

// Test_Register_Framework_Panics_On_All_Cells_Filtered verifies that an
// Excluding configuration that filters every tuple in the product is caught
// at registration — a Cross_Product with nothing left to verify is a bug.
//
// 🛑 If this test breaks, do NOT make registration "tolerant" of empty
// products — read the guard comment on the malformed-clause test.
func Test_Register_Framework_Panics_On_All_Cells_Filtered(t *testing.T) {
	source := `package example

import "github.com/james-orcales/james-orcales/shared/invariant/v2"

func F(r *invariant.Recorder, a, b bool) {
	x := invariant.Recorder_Sometimes(r, a, "X axis is set")
	y := invariant.Recorder_Sometimes(r, b, "Y axis is set")
	invariant.Recorder_Cross_Product(r,
		x, y,
		invariant.Excluding("Fixture forbids the false x false y tuple", invariant.Bucket_False(x), invariant.Bucket_False(y)),
		invariant.Excluding("Fixture forbids the false x true y tuple", invariant.Bucket_False(x), invariant.Bucket_True(y)),
		invariant.Excluding("Fixture forbids the true x false y tuple", invariant.Bucket_True(x), invariant.Bucket_False(y)),
		invariant.Excluding("Fixture forbids the true x true y tuple", invariant.Bucket_True(x), invariant.Bucket_True(y)),
	)
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	defer func() {
		recovered := recover()
		recovered_string, is_string := recovered.(string)
		if !is_string {
			t.Fatalf("expected string panic, got: %T %v", recovered, recovered)
		}
		if !strings.HasPrefix(recovered_string, invariant.Framework_Precondition_Violation_Prefix) {
			t.Fatalf("expected framework-precondition prefix, got: %s", recovered_string)
		}
	}()
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	t.Fatal("expected registration to panic when constraints filter the entire product")
}

// Test_Register_Framework_Panics_On_Single_Boundary_Bucket_Excluding verifies
// that an Excluding clause whose cells all name the same Boundary endpoint
// (Bucket_Hi or Bucket_Lo on a single Distinct_Boundary axis) is caught at
// registration. The shape defeats the point of a boundary: a boundary's
// purpose is that BOTH endpoints are observable in the verified space, so
// declaring one endpoint unreachable should be an Always against the bound,
// not an Excluding stuck onto a Distinct_Boundary.
//
// Covers single-cell `Excluding(Bucket_Hi(b))` AND any number of duplicate
// cells naming the same (axis, bucket) pair like
// `Excluding(Bucket_Hi(b), Bucket_Hi(b))`.
//
// 🛑 If this test breaks, do NOT make registration "tolerant" of this shape —
// the assertion needs to be honest about which endpoint is observable.
func Test_Register_Framework_Panics_On_Single_Boundary_Bucket_Excluding(t *testing.T) {
	tests := []struct {
		Name   string
		Source string
	}{
		{
			Name: "single-cell Bucket_Hi on a boundary axis",
			Source: `package example
import "github.com/james-orcales/james-orcales/shared/invariant/v2"
func F(r *invariant.Recorder, n int) {
	b := invariant.Recorder_Distinct_Boundary(r, &invariant.Boundary_Input[int]{
		X: n, Lo: 0, Hi: 5, Message: "N is bounded",
	})
	invariant.Recorder_Cross_Product(r, b, invariant.Excluding("Fixture forbids the Hi b tuple", invariant.Bucket_Hi(b)))
}
`,
		},
		{
			Name: "single-cell Bucket_Lo on a boundary axis",
			Source: `package example
import "github.com/james-orcales/james-orcales/shared/invariant/v2"
func F(r *invariant.Recorder, n int) {
	b := invariant.Recorder_Distinct_Boundary(r, &invariant.Boundary_Input[int]{
		X: n, Lo: 0, Hi: 5, Message: "N is bounded",
	})
	invariant.Recorder_Cross_Product(r, b, invariant.Excluding("Fixture forbids the Lo b tuple", invariant.Bucket_Lo(b)))
}
`,
		},
		{
			Name: "duplicate Bucket_Hi cells on the same boundary axis",
			Source: `package example
import "github.com/james-orcales/james-orcales/shared/invariant/v2"
func F(r *invariant.Recorder, n int) {
	b := invariant.Recorder_Distinct_Boundary(r, &invariant.Boundary_Input[int]{
		X: n, Lo: 0, Hi: 5, Message: "N is bounded",
	})
	invariant.Recorder_Cross_Product(r, b, invariant.Excluding("Fixture forbids duplicate Hi b cells", invariant.Bucket_Hi(b), invariant.Bucket_Hi(b)))
}
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			recorder, _, _ := test_recorder(&test_recorder_input{
				Source_Files: map[string]string{"fake/example.go": tt.Source},
				Is_Test:      true,
			})
			recorder.Packages_To_Analyze = []string{"/fake"}
			defer func() {
				recovered := recover()
				recovered_string, is_string := recovered.(string)
				if !is_string {
					t.Fatalf("expected string panic, got: %T %v", recovered, recovered)
				}
				if !strings.HasPrefix(recovered_string, invariant.Framework_Precondition_Violation_Prefix) {
					t.Fatalf("expected framework-precondition prefix, got: %s", recovered_string)
				}
			}()
			invariant.Recorder_Register_Packages_For_Analysis(recorder)
			t.Fatal("expected registration to panic on single-endpoint Excluding")
		})
	}
}

// Test_Register_Framework_Accepts_Excluding_Two_Different_Boundary_Buckets
// guards the inverse of the above: Excluding clauses that legitimately name
// two distinct cells — same axis but different buckets, or different axes —
// remain valid. Without this guard the ban could over-fire and reject honest
// "this joint extreme is impossible" assertions like
// `Excluding(Bucket_Hi(a), Bucket_Hi(b))`.
func Test_Register_Framework_Accepts_Excluding_Two_Different_Boundary_Buckets(t *testing.T) {
	tests := []struct {
		Name   string
		Source string
	}{
		{
			Name: "different axes, same Bucket_Hi",
			Source: `package example
import "github.com/james-orcales/james-orcales/shared/invariant/v2"
func F(r *invariant.Recorder, m, n int) {
	a := invariant.Recorder_Distinct_Boundary(r, &invariant.Boundary_Input[int]{X: m, Lo: 0, Hi: 5, Message: "M is bounded"})
	b := invariant.Recorder_Distinct_Boundary(r, &invariant.Boundary_Input[int]{X: n, Lo: 0, Hi: 5, Message: "N is bounded"})
	invariant.Recorder_Cross_Product(r, a, b, invariant.Excluding("Fixture forbids the Hi a Hi b tuple", invariant.Bucket_Hi(a), invariant.Bucket_Hi(b)))
}
`,
		},
		{
			Name: "same axis, different buckets",
			Source: `package example
import "github.com/james-orcales/james-orcales/shared/invariant/v2"
func F(r *invariant.Recorder, n int) {
	b := invariant.Recorder_Distinct_Boundary(r, &invariant.Boundary_Input[int]{X: n, Lo: 0, Hi: 5, Message: "N is bounded"})
	invariant.Recorder_Cross_Product(r, b, invariant.Excluding("Fixture forbids both b endpoints", invariant.Bucket_Hi(b), invariant.Bucket_Lo(b)))
}
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			recorder, _, _ := test_recorder(&test_recorder_input{
				Source_Files: map[string]string{"fake/example.go": tt.Source},
				Is_Test:      true,
			})
			recorder.Packages_To_Analyze = []string{"/fake"}
			invariant.Recorder_Register_Packages_For_Analysis(recorder)
		})
	}
}

// Test_Register_Framework_Panics_On_Excluding_Outside_Cross_Product verifies
// that an Excluding call that isn't a direct argument of a Cross_Product is
// caught by the misuse detector during registration.
//
// 🛑 If this test breaks, do NOT add a silent skip for "stray" Excluding
// calls. Stray Excluding is the entire failure mode this detector exists to
// prevent — read the guard comment on the malformed-clause test.
func Test_Register_Framework_Panics_On_Excluding_Outside_Cross_Product(t *testing.T) {
	source := `package example

import "github.com/james-orcales/james-orcales/shared/invariant/v2"

func F(r *invariant.Recorder, a bool) {
	x := invariant.Recorder_Sometimes(r, a, "X axis is set")
	_ = invariant.Excluding("Fixture exercises stray Excluding detector", invariant.Bucket_False(x))
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	defer func() {
		recovered := recover()
		recovered_string, is_string := recovered.(string)
		if !is_string {
			t.Fatalf("expected string panic, got: %T %v", recovered, recovered)
		}
		if !strings.HasPrefix(recovered_string, invariant.Framework_Precondition_Violation_Prefix) {
			t.Fatalf("expected framework-precondition prefix, got: %s", recovered_string)
		}
	}()
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	t.Fatal("expected registration to panic on Excluding outside Cross_Product")
}

// Test_Register_Framework_Panics_On_Orphan_Axis_Constructor verifies that
// an axis constructor (Sometimes / Always / Distinct_Boundary or any
// Recorder_ variant) called as a top-level expression statement — neither
// passed to a Cross_Product as a direct argument nor bound to a name that
// later appears in a Cross_Product's argument list — is caught at
// registration. The returned Cross_Axis is dropped on the floor without
// reaching r.Assertions, so the assertion contributes nothing at runtime.
// Same severity as Excluding-outside-Cross_Product: surfacing the misuse
// loudly rather than silently letting an apparent assertion vanish.
//
// 🛑 If this test breaks, do NOT add a silent skip for "stray" axis
// constructors. The whole point of this detector is that a bare
// Sometimes / Always / Distinct_Boundary call past any consumer is dead
// — it registers nothing and tracks no branch coverage. Silently
// accepting it lets authors ship apparent assertions that do nothing.
func Test_Register_Framework_Panics_On_Orphan_Axis_Constructor(t *testing.T) {
	source := `package example

import "github.com/james-orcales/james-orcales/shared/invariant/v2"

func F(r *invariant.Recorder, a bool) {
	invariant.Recorder_Sometimes(r, a, "Fixture exercises stray axis-constructor detector")
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	defer func() {
		recovered := recover()
		recovered_string, is_string := recovered.(string)
		if !is_string {
			t.Fatalf("expected string panic, got: %T %v", recovered, recovered)
		}
		if !strings.HasPrefix(recovered_string, invariant.Framework_Precondition_Violation_Prefix) {
			t.Fatalf("expected framework-precondition prefix, got: %s", recovered_string)
		}
	}()
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	t.Fatal("expected registration to panic on bare axis constructor outside Cross_Product")
}

// Test_Register_Framework_Panics_On_Wrong_Bucket_For_Axis_Kind verifies
// that using a Boundary-only bucket constructor (Bucket_Lo) on a Sometimes
// axis is caught at registration.
//
// 🛑 If this test breaks, do NOT add a "best-effort" mapping silently
// converting Bucket_Lo to Bucket_False — the whole point of the bucket-kind
// match is to catch typos and misuses LOUDLY. Read the guard comment on the
// malformed-clause test.
func Test_Register_Framework_Panics_On_Wrong_Bucket_For_Axis_Kind(t *testing.T) {
	source := `package example

import "github.com/james-orcales/james-orcales/shared/invariant/v2"

func F(r *invariant.Recorder, a, b bool) {
	x := invariant.Recorder_Sometimes(r, a, "X axis is set")
	y := invariant.Recorder_Sometimes(r, b, "Y axis is set")
	invariant.Recorder_Cross_Product(r,
		x, y,
		invariant.Excluding("Fixture forbids the Lo x true y tuple", invariant.Bucket_Lo(x), invariant.Bucket_True(y)),
	)
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	defer func() {
		recovered := recover()
		recovered_string, is_string := recovered.(string)
		if !is_string {
			t.Fatalf("expected string panic, got: %T %v", recovered, recovered)
		}
		if !strings.HasPrefix(recovered_string, invariant.Framework_Precondition_Violation_Prefix) {
			t.Fatalf("expected framework-precondition prefix, got: %s", recovered_string)
		}
	}()
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	t.Fatal("expected registration to panic on Bucket_Lo applied to a Sometimes axis")
}

// Test_Axis_Handles_Distinct_Within_Single_Cross_Product verifies that two
// distinct axes in the same call carry distinct Handle pointers — the
// invariant the runtime constraint check relies on.
func Test_Axis_Handles_Distinct_Within_Single_Cross_Product(t *testing.T) {
	recorder, _, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 7,
		Is_Test:     true,
	})
	a := invariant.Recorder_Sometimes(recorder, true, "A axis")
	b := invariant.Recorder_Sometimes(recorder, false, "B axis")
	if a.Handle == nil {
		t.Fatal("expected Recorder_Sometimes to stamp a non-nil Handle on the returned Cross_Axis")
	}
	if b.Handle == nil {
		t.Fatal("expected Recorder_Sometimes to stamp a non-nil Handle on the returned Cross_Axis")
	}
	if a.Handle == b.Handle {
		t.Fatal("expected two Cross_Axis values from the same Recorder to carry distinct Handles")
	}
}

// Test_Axis_Handles_Reused_Across_Cross_Product_Calls verifies that after a
// Cross_Product call completes, the handles are released to the pool — the
// next call may reuse them, and correctness is preserved (handle identity
// only needs to be unique within a single call).
func Test_Axis_Handles_Reused_Across_Cross_Product_Calls(t *testing.T) {
	recorder, _, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 7,
		Is_Test:     true,
	})
	recorder.Assertions.Store("/fake/test.go:7:tuple=(0)", &invariant.Assertion_Metadata{
		Kind:          "Cross",
		Site:          "/fake/test.go:7",
		Tuple_Indices: []int{0},
	})
	recorder.Assertions.Store("/fake/test.go:7:tuple=(1)", &invariant.Assertion_Metadata{
		Kind:          "Cross",
		Site:          "/fake/test.go:7",
		Tuple_Indices: []int{1},
	})

	for i := 0; i < 2; i++ {
		a := invariant.Recorder_Sometimes(recorder, i == 0, "A axis")
		invariant.Recorder_Cross_Product(recorder, a)
	}

	value_0, ok_0 := recorder.Assertions.Load("/fake/test.go:7:tuple=(0)")
	value_1, ok_1 := recorder.Assertions.Load("/fake/test.go:7:tuple=(1)")
	if !ok_0 || !ok_1 {
		t.Fatal("expected both tracker entries to remain")
	}
	if value_0.(*invariant.Assertion_Metadata).Frequency.Load() != 1 {
		t.Fatalf("expected tuple (0) Frequency=1, got %d", value_0.(*invariant.Assertion_Metadata).Frequency.Load())
	}
	if value_1.(*invariant.Assertion_Metadata).Frequency.Load() != 1 {
		t.Fatalf("expected tuple (1) Frequency=1, got %d", value_1.(*invariant.Assertion_Metadata).Frequency.Load())
	}
}

// Test_Git_Input_Check_Constraint_Replacement is the migration parity test:
// the new Excluding-form constraint replaces the older Is_Sometimes×2 +
// Is_Always workaround. The forbidden tuple (Enabled=false, Absent=true)
// must fail-fatal at runtime — same enforcement as the previous Is_Always.
func Test_Git_Input_Check_Constraint_Replacement(t *testing.T) {
	recorder, _, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/git.go",
		Caller_Line: 7,
		Is_Test:     true,
	})
	defer func() {
		recovered := recover()
		recovered_string, is_string := recovered.(string)
		if !is_string {
			t.Fatalf("expected string panic for the forbidden tuple, got: %T %v", recovered, recovered)
		}
		if !strings.HasPrefix(recovered_string, invariant.Assertion_Failure_Message_Prefix) {
			t.Fatalf("expected user-domain assertion failure prefix, got: %s", recovered_string)
		}
	}()
	enabled := invariant.Recorder_Sometimes(recorder, false, "Git history tier is enabled")
	absent := invariant.Recorder_Sometimes(recorder, true, "Main reference is absent on shallow checkouts")
	invariant.Recorder_Cross_Product(recorder,
		enabled, absent,
		invariant.Excluding("Enabled false implies absent false in this fixture", invariant.Bucket_False(enabled), invariant.Bucket_True(absent)),
	)
	t.Fatal("expected forbidden (Enabled=false, Absent=true) tuple to fail-fatal")
}

// Test_Excluding_Clause_All_Cells_Must_Match verifies that an Excluding clause
// fires only when EVERY cell in the conjunction matches. A clause naming two
// cells whose axes match only one cell at runtime must not panic.
func Test_Excluding_Clause_All_Cells_Must_Match(t *testing.T) {
	recorder, _, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 7,
		Is_Test:     true,
	})
	recorder.Assertions.Store("/fake/test.go:7:tuple=(1,0,0)", &invariant.Assertion_Metadata{
		Kind: "Cross", Site: "/fake/test.go:7", Tuple_Indices: []int{1, 0, 0},
	})

	a := invariant.Recorder_Sometimes(recorder, true, "A axis is set")
	b := invariant.Recorder_Sometimes(recorder, false, "B axis is set")
	c := invariant.Recorder_Sometimes(recorder, false, "C axis is set")
	invariant.Recorder_Cross_Product(recorder,
		a, b, c,
		invariant.Excluding("Fixture forbids (true, true) tuple", invariant.Bucket_True(a), invariant.Bucket_True(b)),
	)
	value, ok := recorder.Assertions.Load("/fake/test.go:7:tuple=(1,0,0)")
	if !ok {
		t.Fatal("expected tracker entry for tuple (1,0,0)")
	}
	if value.(*invariant.Assertion_Metadata).Frequency.Load() != 1 {
		t.Fatalf("expected Frequency=1 — only A matches the clause, B does not, so clause should not fire; got %d", value.(*invariant.Assertion_Metadata).Frequency.Load())
	}
}

// Test_Excluding_Partial_Axis_Wildcards verifies that an Excluding clause
// naming a single axis (k=1 of N total axes) forbids every tuple where that
// axis is in the named bucket, regardless of the other axes' buckets.
func Test_Excluding_Partial_Axis_Wildcards(t *testing.T) {
	recorder, _, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 7,
		Is_Test:     true,
	})
	defer func() {
		recovered := recover()
		recovered_string, is_string := recovered.(string)
		if !is_string {
			t.Fatalf("expected string panic, got: %T %v", recovered, recovered)
		}
		if !strings.HasPrefix(recovered_string, invariant.Assertion_Failure_Message_Prefix) {
			t.Fatalf("expected user-domain assertion failure prefix, got: %s", recovered_string)
		}
	}()

	a := invariant.Recorder_Sometimes(recorder, true, "A axis is set")
	b := invariant.Recorder_Sometimes(recorder, false, "B axis is set")
	// Excluding names only a; the wildcard over b means any b value with a=true triggers.
	invariant.Recorder_Cross_Product(recorder,
		a, b,
		invariant.Excluding("Fixture forbids the true a tuple", invariant.Bucket_True(a)),
	)
	t.Fatal("expected (a=true, b=anything) to fail-fatal under partial-axis Excluding")
}

// Test_Three_Axis_Cross_Product_With_Excluding verifies that constraints
// scale to 3-axis products: the forbidden cell is recognized; cells matching
// only some of the constraint's cells are admitted.
func Test_Three_Axis_Cross_Product_With_Excluding(t *testing.T) {
	recorder, _, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 7,
		Is_Test:     true,
	})
	defer func() {
		recovered := recover()
		recovered_string, is_string := recovered.(string)
		if !is_string {
			t.Fatalf("expected string panic, got: %T %v", recovered, recovered)
		}
		if !strings.HasPrefix(recovered_string, invariant.Assertion_Failure_Message_Prefix) {
			t.Fatalf("expected user-domain assertion failure prefix, got: %s", recovered_string)
		}
	}()

	a := invariant.Recorder_Sometimes(recorder, true, "A axis is set")
	b := invariant.Recorder_Sometimes(recorder, true, "B axis is set")
	c := invariant.Recorder_Sometimes(recorder, true, "C axis is set")
	invariant.Recorder_Cross_Product(recorder,
		a, b, c,
		invariant.Excluding("Fixture forbids the all-true triple", invariant.Bucket_True(a), invariant.Bucket_True(b), invariant.Bucket_True(c)),
	)
	t.Fatal("expected (true, true, true) to fail-fatal under 3-axis Excluding")
}

// Test_Three_Axis_Cross_Product_AST_Filter verifies that AST pre-registration
// filters the right number of cells from a 3-axis product (2^3 = 8 cells; one
// Excluding clause removes one).
func Test_Three_Axis_Cross_Product_AST_Filter(t *testing.T) {
	source := `package example

import "github.com/james-orcales/james-orcales/shared/invariant/v2"

func F(r *invariant.Recorder, p, q, s bool) {
	a := invariant.Recorder_Sometimes(r, p, "A axis is set")
	b := invariant.Recorder_Sometimes(r, q, "B axis is set")
	c := invariant.Recorder_Sometimes(r, s, "C axis is set")
	invariant.Recorder_Cross_Product(r,
		a, b, c,
		invariant.Excluding("Fixture forbids the all-true triple", invariant.Bucket_True(a), invariant.Bucket_True(b), invariant.Bucket_True(c)),
	)
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	invariant.Recorder_Register_Packages_For_Analysis(recorder)

	cross_entries := []*invariant.Assertion_Metadata{}
	recorder.Assertions.Range(func(key, value any) (continue_iter bool) {
		metadata := value.(*invariant.Assertion_Metadata)
		if metadata.Kind == "Cross" {
			cross_entries = append(cross_entries, metadata)
		}
		return true
	})
	if len(cross_entries) != 7 {
		t.Fatalf("expected 7 of 8 cells after one Excluding clause, got %d", len(cross_entries))
	}
}

// Test_Cross_Product_No_Constraints_Unchanged verifies that the migration
// hasn't broken the no-constraint path: a Cross_Product with only axes
// pre-registers the full Cartesian product as before.
func Test_Cross_Product_No_Constraints_Unchanged(t *testing.T) {
	source := `package example

import "github.com/james-orcales/james-orcales/shared/invariant/v2"

func F(r *invariant.Recorder, p, q bool) {
	invariant.Recorder_Cross_Product(r,
		invariant.Recorder_Sometimes(r, p, "A axis is set"),
		invariant.Recorder_Sometimes(r, q, "B axis is set"),
	)
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	invariant.Recorder_Register_Packages_For_Analysis(recorder)

	cross_entries := 0
	recorder.Assertions.Range(func(key, value any) (continue_iter bool) {
		if value.(*invariant.Assertion_Metadata).Kind == "Cross" {
			cross_entries++
		}
		return true
	})
	if cross_entries != 4 {
		t.Fatalf("expected full 2x2=4 Cartesian product without constraints, got %d", cross_entries)
	}
}

// Test_Constraint_Violation_Names_Offending_Axes verifies the panic message
// contains the axis Message strings — without that, the report is unreadable.
func Test_Constraint_Violation_Names_Offending_Axes(t *testing.T) {
	recorder, _, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 7,
		Is_Test:     true,
	})
	defer func() {
		recovered := recover()
		recovered_string, is_string := recovered.(string)
		if !is_string {
			t.Fatalf("expected string panic, got: %T %v", recovered, recovered)
		}
		if !strings.Contains(recovered_string, "Enabled is set") {
			t.Fatalf("expected panic to name the Enabled axis message; got: %s", recovered_string)
		}
		if !strings.Contains(recovered_string, "Absent is set") {
			t.Fatalf("expected panic to name the Absent axis message; got: %s", recovered_string)
		}
	}()

	enabled := invariant.Recorder_Sometimes(recorder, false, "Enabled is set")
	absent := invariant.Recorder_Sometimes(recorder, true, "Absent is set")
	invariant.Recorder_Cross_Product(recorder,
		enabled, absent,
		invariant.Excluding("Enabled false implies absent false in this fixture", invariant.Bucket_False(enabled), invariant.Bucket_True(absent)),
	)
	t.Fatal("expected violation to fail-fatal")
}

// Test_Constraint_Violation_Includes_Bucket_Names verifies the panic message
// names the observed bucket (e.g., "true" / "false") for each axis.
func Test_Constraint_Violation_Includes_Bucket_Names(t *testing.T) {
	recorder, _, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 7,
		Is_Test:     true,
	})
	defer func() {
		recovered := recover()
		recovered_string, is_string := recovered.(string)
		if !is_string {
			t.Fatalf("expected string panic, got: %T %v", recovered, recovered)
		}
		if !strings.Contains(recovered_string, "false") {
			t.Fatalf("expected panic to name the 'false' bucket; got: %s", recovered_string)
		}
		if !strings.Contains(recovered_string, "true") {
			t.Fatalf("expected panic to name the 'true' bucket; got: %s", recovered_string)
		}
	}()

	a := invariant.Recorder_Sometimes(recorder, false, "A axis is set")
	b := invariant.Recorder_Sometimes(recorder, true, "B axis is set")
	invariant.Recorder_Cross_Product(recorder,
		a, b,
		invariant.Excluding("Fixture forbids the false a true b tuple", invariant.Bucket_False(a), invariant.Bucket_True(b)),
	)
	t.Fatal("expected violation to fail-fatal")
}

// Test_Cross_Product_Inside_Loop_Pool_Reuse stresses handle pool reuse: many
// sequential calls don't cause the constraint check to confuse handles
// across iterations.
func Test_Cross_Product_Inside_Loop_Pool_Reuse(t *testing.T) {
	recorder, _, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 7,
		Is_Test:     true,
	})
	recorder.Assertions.Store("/fake/test.go:7:tuple=(0,1)", &invariant.Assertion_Metadata{
		Kind: "Cross", Site: "/fake/test.go:7", Tuple_Indices: []int{0, 1},
	})
	recorder.Assertions.Store("/fake/test.go:7:tuple=(1,0)", &invariant.Assertion_Metadata{
		Kind: "Cross", Site: "/fake/test.go:7", Tuple_Indices: []int{1, 0},
	})

	// Alternate enabled between true/false, keep absent always false, so
	// every observation is either (0,0) or (1,0) — both admitted by the
	// Excluding clause that forbids (0,1).
	for i := 0; i < 50; i++ {
		enabled := invariant.Recorder_Sometimes(recorder, i%2 == 0, "Enabled is set")
		absent := invariant.Recorder_Sometimes(recorder, false, "Absent is set")
		invariant.Recorder_Cross_Product(recorder,
			enabled, absent,
			invariant.Excluding("Enabled false implies absent false in this fixture", invariant.Bucket_False(enabled), invariant.Bucket_True(absent)),
		)
	}

	value, ok := recorder.Assertions.Load("/fake/test.go:7:tuple=(1,0)")
	if !ok {
		t.Fatal("expected tracker entry for (1,0) after loop")
	}
	if value.(*invariant.Assertion_Metadata).Frequency.Load() == 0 {
		t.Fatal("expected (1,0) Frequency > 0 after 50 iterations")
	}
}

// Test_Register_Framework_Panics_On_Bucket_True_On_Boundary_Axis is the
// reverse of Test_Register_Framework_Panics_On_Wrong_Bucket_For_Axis_Kind:
// the existing test catches Bucket_Lo on a Sometimes axis; this one catches
// Bucket_True on a Boundary axis. Both directions of the kind mismatch must
// framework-panic at registration.
//
// 🛑 DO NOT downgrade this framework-panic to a silent skip. The bucket-kind
// check is the only thing that stops Bucket_True(boundary_axis) from
// silently meaning "Bucket_Hi" via index-1 aliasing. Read the assertion's
// docstring on ast_bucket_index_for_selector before "fixing" this.
func Test_Register_Framework_Panics_On_Bucket_True_On_Boundary_Axis(t *testing.T) {
	source := `package example

import "github.com/james-orcales/james-orcales/shared/invariant/v2"

func F(r *invariant.Recorder, count int) {
	c := invariant.Recorder_Distinct_Boundary(r, &invariant.Boundary_Input[int]{
		X: count, Lo: 0, Hi: 10, Message: "Count within window",
	})
	invariant.Recorder_Cross_Product(r,
		c,
		invariant.Excluding("Fixture forbids the true c tuple", invariant.Bucket_True(c)),
	)
}
`
	recorder, _, _ := test_recorder(&test_recorder_input{
		Source_Files: map[string]string{"fake/example.go": source},
		Is_Test:      true,
	})
	recorder.Packages_To_Analyze = []string{"/fake"}
	defer func() {
		recovered := recover()
		recovered_string, is_string := recovered.(string)
		if !is_string {
			t.Fatalf("expected string panic, got: %T %v", recovered, recovered)
		}
		if !strings.HasPrefix(recovered_string, invariant.Framework_Precondition_Violation_Prefix) {
			t.Fatalf("expected framework-precondition prefix, got: %s", recovered_string)
		}
	}()
	invariant.Recorder_Register_Packages_For_Analysis(recorder)
	t.Fatal("expected registration to panic on Bucket_True applied to a Boundary axis")
}

// Test_Boundary_Interior_Without_Constraint_Silent verifies that an interior
// Boundary observation is silent at the tracker layer — the key built from
// Bucket_Index = -1 matches no pre-registered entry, no Frequency is
// incremented, and the call returns without error.
func Test_Boundary_Interior_Without_Constraint_Silent(t *testing.T) {
	recorder, _, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 7,
		Is_Test:     true,
	})
	recorder.Assertions.Store("/fake/test.go:7:tuple=(0)", &invariant.Assertion_Metadata{
		Kind: "Cross", Site: "/fake/test.go:7", Tuple_Indices: []int{0},
	})
	recorder.Assertions.Store("/fake/test.go:7:tuple=(1)", &invariant.Assertion_Metadata{
		Kind: "Cross", Site: "/fake/test.go:7", Tuple_Indices: []int{1},
	})

	c := invariant.Recorder_Distinct_Boundary(recorder, &invariant.Boundary_Input[int]{
		X: 5, Lo: 0, Hi: 10, Message: "Count within window",
	})
	invariant.Recorder_Cross_Product(recorder, c)

	for _, key := range []string{"/fake/test.go:7:tuple=(0)", "/fake/test.go:7:tuple=(1)"} {
		value, ok := recorder.Assertions.Load(key)
		if !ok {
			t.Fatalf("expected tracker entry %q to remain", key)
		}
		if value.(*invariant.Assertion_Metadata).Frequency.Load() != 0 {
			t.Fatalf("expected interior observation to leave Frequency=0 at %q; got %d",
				key, value.(*invariant.Assertion_Metadata).Frequency.Load())
		}
	}
}

// Test_Boundary_Interior_With_Excluding_Admitted verifies that Excluding does
// NOT panic on interior observations: the interior bucket index (-1) matches
// no Excluding cell (cells name only 0 or 1). The observation passes the
// constraint check and falls through to the tracker's silent miss.
func Test_Boundary_Interior_With_Excluding_Admitted(t *testing.T) {
	recorder, _, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 7,
		Is_Test:     true,
	})

	c := invariant.Recorder_Distinct_Boundary(recorder, &invariant.Boundary_Input[int]{
		X: 5, Lo: 0, Hi: 10, Message: "Count within window",
	})
	// No panic expected — Excluding(Bucket_Lo) only forbids the Lo cell;
	// interior X is neither Lo nor Hi, so the clause doesn't apply.
	invariant.Recorder_Cross_Product(recorder, c, invariant.Excluding("Fixture forbids the Lo c tuple", invariant.Bucket_Lo(c)))
}

// Test_Multiple_Cross_Product_Sites_Independent verifies that two distinct
// Cross_Product call sites in the same test maintain independent constraint
// state — a violation at site A doesn't bleed into the tracker for site B.
func Test_Multiple_Cross_Product_Sites_Independent(t *testing.T) {
	recorder, _, _ := test_recorder(&test_recorder_input{
		Caller_File: "/fake/test.go",
		Caller_Line: 7,
		Is_Test:     true,
	})
	recorder.Assertions.Store("/fake/test.go:7:tuple=(1,1)", &invariant.Assertion_Metadata{
		Kind: "Cross", Site: "/fake/test.go:7", Tuple_Indices: []int{1, 1},
	})

	// Site A: admit (true, true); site B (same caller line for test simplicity)
	// independently fires. Run only one to verify site state.
	a := invariant.Recorder_Sometimes(recorder, true, "A axis is set")
	b := invariant.Recorder_Sometimes(recorder, true, "B axis is set")
	invariant.Recorder_Cross_Product(recorder,
		a, b,
		invariant.Excluding("Fixture forbids the false a true b tuple", invariant.Bucket_False(a), invariant.Bucket_True(b)),
	)

	value, ok := recorder.Assertions.Load("/fake/test.go:7:tuple=(1,1)")
	if !ok {
		t.Fatal("expected (1,1) tracker entry")
	}
	if value.(*invariant.Assertion_Metadata).Frequency.Load() != 1 {
		t.Fatalf("expected (1,1) Frequency=1; got %d", value.(*invariant.Assertion_Metadata).Frequency.Load())
	}
}
