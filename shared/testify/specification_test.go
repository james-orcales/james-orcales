package testify_test

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	"local/james-orcales/shared/io"
	"local/james-orcales/shared/math/fixedpoint"
	testify "local/james-orcales/shared/testify"
	"local/james-orcales/shared/time"
)

// Test_Equality checks the equality family passes on equal operands and its predicates
// decide both ways.
func Test_Equality(t *testing.T) {
	t.Parallel()
	if !testify.Equal(t, 3, 3) {
		t.Errorf("Equal on equal values should pass")
	}
	if !testify.Not_Equal(t, 3, 4) {
		t.Errorf("Not_Equal on unequal values should pass")
	}
	if !testify.Equal_Values(t, int32(7), int64(7)) {
		t.Errorf("Equal_Values should coerce numeric types")
	}
	if !testify.Not_Equal_Values(t, 7, "7") {
		t.Errorf("Not_Equal_Values should reject unlike values")
	}
	if !testify.Exactly(t, int64(7), int64(7)) {
		t.Errorf("Exactly should pass on identical type and value")
	}
	first := 5
	second := 5
	if !testify.Same(t, &first, &first) {
		t.Errorf("Same should pass on one pointer")
	}
	if !testify.Not_Same(t, &first, &second) {
		t.Errorf("Not_Same should pass on distinct pointers")
	}
}

// Test_Nil_And_Truth checks the nil, boolean, and zero assertions pass on matching input.
func Test_Nil_And_Truth(t *testing.T) {
	t.Parallel()
	var empty_pointer *int
	if !testify.Nil(t, empty_pointer) {
		t.Errorf("Nil should see through a typed nil pointer")
	}
	if !testify.Not_Nil(t, &struct{}{}) {
		t.Errorf("Not_Nil should pass on a live pointer")
	}
	if !testify.True(t, true) {
		t.Errorf("True should pass on true")
	}
	if !testify.False(t, false) {
		t.Errorf("False should pass on false")
	}
	if !testify.Zero(t, 0) {
		t.Errorf("Zero should pass on a zero value")
	}
	if !testify.Not_Zero(t, 1) {
		t.Errorf("Not_Zero should pass on a non-zero value")
	}
}

// Test_Emptiness_And_Count checks the emptiness and element-count assertions.
func Test_Emptiness_And_Count(t *testing.T) {
	t.Parallel()
	if !testify.Empty(t, []int{}) {
		t.Errorf("Empty should pass on an empty slice")
	}
	if !testify.Not_Empty(t, []int{1}) {
		t.Errorf("Not_Empty should pass on a filled slice")
	}
	if !testify.Count(t, []int{1, 2, 3}, 3) {
		t.Errorf("Count should pass on a matching length")
	}
}

// Test_Collections checks the membership, subset, and multiset assertions.
func Test_Collections(t *testing.T) {
	t.Parallel()
	if !testify.Contains(t, []int{1, 2, 3}, 2) {
		t.Errorf("Contains should find a present member")
	}
	if !testify.Not_Contains(t, []int{1, 2, 3}, 9) {
		t.Errorf("Not_Contains should pass on an absent member")
	}
	if !testify.Contains_Any(t, "hello world", "world") {
		t.Errorf("Contains_Any should find a substring")
	}
	if !testify.Not_Contains_Any(t, "hello", "earth") {
		t.Errorf("Not_Contains_Any should pass on an absent substring")
	}
	if !testify.Subset(t, []int{1, 2, 3}, []int{1, 2}) {
		t.Errorf("Subset should pass when list covers subset")
	}
	if !testify.Not_Subset(t, []int{1, 2}, []int{1, 9}) {
		t.Errorf("Not_Subset should pass when a member is missing")
	}
	if !testify.Elements_Match(t, []int{1, 3, 2, 3}, []int{3, 2, 3, 1}) {
		t.Errorf("Elements_Match should ignore order and honor duplicates")
	}
	if !testify.Not_Elements_Match(t, []int{1, 2}, []int{1, 2, 3}) {
		t.Errorf("Not_Elements_Match should pass on differing multisets")
	}
}

// Test_Errors checks the error assertions on a wrapped chain.
func Test_Errors(t *testing.T) {
	t.Parallel()
	root := &sentinel_error{Label: "root cause"}
	wrapped := fmt.Errorf("context: %w", root)
	if !testify.No_Error(t, nil) {
		t.Errorf("No_Error should pass on nil")
	}
	if !testify.Error(t, wrapped) {
		t.Errorf("Error should pass on a non-nil error")
	}
	if !testify.Equal_Error(t, root, "root cause") {
		t.Errorf("Equal_Error should match the message")
	}
	if !testify.Error_Contains(t, wrapped, "root cause") {
		t.Errorf("Error_Contains should find the substring")
	}
	if !testify.Error_Is(t, wrapped, root) {
		t.Errorf("Error_Is should find the wrapped cause")
	}
	if !testify.Not_Error_Is(t, root, errors.New("other")) {
		t.Errorf("Not_Error_Is should pass on an unrelated target")
	}
	var target *sentinel_error
	if !testify.Error_As(t, wrapped, &target) {
		t.Errorf("Error_As should bind the concrete error")
	}
}

// Test_Types checks the interface and dynamic-type assertions.
func Test_Types(t *testing.T) {
	t.Parallel()
	if !testify.Implements(t, (*error)(nil), errors.New("x")) {
		t.Errorf("Implements should accept an error value")
	}
	if !testify.Not_Implements(t, (*error)(nil), 5) {
		t.Errorf("Not_Implements should reject a non-error")
	}
	if !testify.Is_Type(t, 0, 7) {
		t.Errorf("Is_Type should match two ints")
	}
	if !testify.Is_Not_Type(t, 0, "seven") {
		t.Errorf("Is_Not_Type should reject unlike types")
	}
}

// Test_Ordering checks the ordered-comparison assertions.
func Test_Ordering(t *testing.T) {
	t.Parallel()
	if !testify.Greater(t, &testify.Greater_Input[int]{First: 3, Second: 2}) {
		t.Errorf("Greater should pass on a larger first")
	}
	if !testify.Greater_Or_Equal(t, &testify.Greater_Or_Equal_Input[int]{First: 3, Second: 3}) {
		t.Errorf("Greater_Or_Equal should pass on equal operands")
	}
	if !testify.Less(t, &testify.Less_Input[int]{First: 2, Second: 3}) {
		t.Errorf("Less should pass on a smaller first")
	}
	if !testify.Less_Or_Equal(t, &testify.Less_Or_Equal_Input[int]{First: 3, Second: 3}) {
		t.Errorf("Less_Or_Equal should pass on equal operands")
	}
	if !testify.Positive(t, 1) {
		t.Errorf("Positive should pass on a positive value")
	}
	if !testify.Negative(t, -1) {
		t.Errorf("Negative should pass on a negative value")
	}
}

// Test_Panics checks the panic assertions.
func Test_Panics(t *testing.T) {
	t.Parallel()
	if !testify.Panics(t, func() { panic("boom") }) {
		t.Errorf("Panics should observe a panic")
	}
	if !testify.Not_Panics(t, func() {}) {
		t.Errorf("Not_Panics should pass on a calm func")
	}
	if !testify.Panics_With_Value(t, "boom", func() { panic("boom") }) {
		t.Errorf("Panics_With_Value should match the value")
	}
	if !testify.Panics_With_Error(t, "boom", func() { panic(errors.New("boom")) }) {
		t.Errorf("Panics_With_Error should match the message")
	}
}

// Test_Approximation_And_Time checks the fixed-point tolerance and moment assertions.
func Test_Approximation_And_Time(t *testing.T) {
	t.Parallel()
	delta := &testify.In_Delta_Input{
		Expected: fixedpoint.Number(fixedpoint.From_Integer(10)),
		Actual:   fixedpoint.Number(fixedpoint.From_Integer(10)),
		Delta:    fixedpoint.Number(fixedpoint.From_Integer(1)),
	}
	if !testify.In_Delta(t, delta) {
		t.Errorf("In_Delta should pass within the delta")
	}
	epsilon := &testify.In_Epsilon_Input{
		Expected: fixedpoint.Number(fixedpoint.From_Integer(100)),
		Actual:   fixedpoint.Number(fixedpoint.From_Integer(100)),
		Epsilon:  fixedpoint.SCALE / 100,
	}
	if !testify.In_Epsilon(t, epsilon) {
		t.Errorf("In_Epsilon should pass within the epsilon")
	}
	duration := &testify.Within_Duration_Input{Expected: 1000, Actual: 1005, Delta: 10}
	if !testify.Within_Duration(t, duration) {
		t.Errorf("Within_Duration should pass within the delta")
	}
	span := &testify.Within_Range_Input{Actual: 5, Start: 1, End: 9}
	if !testify.Within_Range(t, span) {
		t.Errorf("Within_Range should pass inside the range")
	}
}

// Test_Regexp_And_JSON checks the pattern and JSON-equality assertions.
func Test_Regexp_And_JSON(t *testing.T) {
	t.Parallel()
	if !testify.Regexp(t, "wor.d", "hello world") {
		t.Errorf("Regexp should match a string source")
	}
	if !testify.Regexp(t, regexp.MustCompile("^hel"), "hello") {
		t.Errorf("Regexp should accept a compiled expression")
	}
	if !testify.Not_Regexp(t, "^x", "hello") {
		t.Errorf("Not_Regexp should pass on no match")
	}
	if !testify.JSON_Eq(t, `{"a":1,"b":2}`, `{"b":2,"a":1}`) {
		t.Errorf("JSON_Eq should ignore key order")
	}
}

// Test_Control checks Condition, Fail, and the predicate split's reporting.
func Test_Control(t *testing.T) {
	t.Parallel()
	if !testify.Condition(t, func() (satisfied bool) { return true }) {
		t.Errorf("Condition should pass when the predicate holds")
	}
}

// Test_Predicates checks the pure decisions the assertions build on, both ways.
func Test_Predicates(t *testing.T) {
	t.Parallel()
	if !testify.Objects_Are_Equal(1, 1) {
		t.Errorf("Objects_Are_Equal should hold on equal values")
	}
	if testify.Objects_Are_Equal(1, 2) {
		t.Errorf("Objects_Are_Equal should fail on unequal values")
	}
	if !testify.Objects_Are_Equal_Values(int32(3), int64(3)) {
		t.Errorf("Objects_Are_Equal_Values should coerce")
	}
	var empty_pointer *int
	if !testify.Is_Nil(empty_pointer) {
		t.Errorf("Is_Nil should see a typed nil")
	}
	if testify.Is_Nil(1) {
		t.Errorf("Is_Nil should reject a non-nil value")
	}
	if !testify.Is_Empty([]int{}) {
		t.Errorf("Is_Empty should hold on an empty slice")
	}
	if testify.Is_Empty([]int{1}) {
		t.Errorf("Is_Empty should reject a filled slice")
	}
	predicates_pointers(t)
	predicates_lists(t)
}

// Test_Files checks the filesystem assertions against an in-memory file system.
func Test_Files(t *testing.T) {
	t.Parallel()
	file_system := fstest.MapFS{
		"config.txt":     &fstest.MapFile{Data: []byte("hello")},
		"data/inner.txt": &fstest.MapFile{Data: []byte("x")},
	}
	a := &testify.Asserter{File_System: file_system}
	if !testify.Asserter_File_Exists(a, t, "/config.txt") {
		t.Errorf("File_Exists should find a present file")
	}
	if !testify.Asserter_No_File_Exists(a, t, "/missing.txt") {
		t.Errorf("No_File_Exists should pass on an absent file")
	}
	if !testify.Asserter_Directory_Exists(a, t, "/data") {
		t.Errorf("Directory_Exists should find a present directory")
	}
	if !testify.Asserter_No_Directory_Exists(a, t, "/config.txt") {
		t.Errorf("No_Directory_Exists should pass on a file path")
	}
}

// Test_Eventually_And_Never checks the polling assertions under a driven sim loop.
func Test_Eventually_And_Never(t *testing.T) {
	t.Parallel()
	loop, driver, clock := io.New_Sim(1)
	a := &testify.Asserter{Clock: clock, IO: &loop}
	poll_count := 0
	condition := func() (satisfied bool) {
		poll_count++
		return poll_count >= 3
	}
	eventually := &testify.Asserter_Eventually_Input{
		Wait: 100 * time.NANOSECOND,
		Tick: 10 * time.NANOSECOND,
	}
	testify.Asserter_Eventually(a, t, condition, eventually)
	driver.Run_For(100 * time.NANOSECOND)

	never_loop, never_driver, never_clock := io.New_Sim(2)
	never_asserter := &testify.Asserter{Clock: never_clock, IO: &never_loop}
	never := &testify.Asserter_Never_Input{
		Wait: 50 * time.NANOSECOND,
		Tick: 10 * time.NANOSECOND,
	}
	testify.Asserter_Never(never_asserter, t, func() (satisfied bool) { return false }, never)
	never_driver.Run_For(60 * time.NANOSECOND)
}

// Test_Hard_Failures runs a failing assertion in a child test process because FailNow
// terminates the calling test goroutine before it can make an in-process observation.
func Test_Hard_Failures(t *testing.T) {
	if os.Getenv("TESTIFY_HARD_FAILURE") == "1" {
		testify.Equal(t, 1, 2)
		t.Fatal("a failing assertion returned")
	}

	command := exec.Command(os.Args[0], "-test.run=^Test_Hard_Failures$")
	command.Env = append(os.Environ(), "TESTIFY_HARD_FAILURE=1")
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatal("a failing assertion should fail the child test")
	}
	if strings.Contains(string(output), "a failing assertion returned") {
		t.Fatal("a failing assertion should terminate the child test before returning")
	}
}

// Checks the pointer-identity and membership predicates.
func predicates_pointers(t *testing.T) {
	t.Helper()
	value := 5
	same, addressable := testify.Same_Pointers(&value, &value)
	if !addressable {
		t.Errorf("Same_Pointers should mark two pointers addressable")
	}
	if !same {
		t.Errorf("Same_Pointers should hold on one address")
	}
	_, plain := testify.Same_Pointers(1, 2)
	if plain {
		t.Errorf("Same_Pointers should reject non-pointers")
	}
}

// Checks the list-search and list-difference predicates.
func predicates_lists(t *testing.T) {
	t.Helper()
	searchable, found := testify.Contains_Element([]int{1, 2, 3}, 2)
	if !searchable {
		t.Errorf("Contains_Element should mark a slice searchable")
	}
	if !found {
		t.Errorf("Contains_Element should find a present member")
	}
	extra_a, extra_b := testify.Diff_Lists([]int{1, 2}, []int{2, 3})
	if len(extra_a) != 1 {
		t.Errorf("Diff_Lists should report one element only in A")
	}
	if len(extra_b) != 1 {
		t.Errorf("Diff_Lists should report one element only in B")
	}
}

// Concrete error type for exercising Equal_Error and Error_As.
type sentinel_error struct {
	// Label is the error message.
	Label string
}

// Error renders the sentinel's label, satisfying the error interface.
func (sentinel *sentinel_error) Error() (message string) {
	return sentinel.Label
}
