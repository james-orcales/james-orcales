// Package testify is a dependency-injected port of github.com/stretchr/testify's
// assert/require surface (MIT, Mat Ryer and Tyler Bunnell; see LICENSE.mit.stretchr).
// It keeps testify's assertion vocabulary but rebuilds the internals and calling
// convention in the house idiom, so it drops into deterministic tests the way
// shared/snap and shared/lru do.
//
// Every deviation from upstream is deliberate:
//   - There is one flat package, not assert plus require. Every assertion reports through
//     t.Errorf and then terminates the test with t.FailNow, so a failure cannot leave the
//     test in a state that later checks might mistake for valid.
//   - Assertions take a concrete testing handle, not testify's TestingT interface, which
//     the linter bans. The pure decisions (Objects_Are_Equal, Is_Nil, Is_Empty,
//     Same_Pointers, Contains_Element, Diff_Lists) are factored out so they take no handle
//     and are tested directly, mirroring snap's Snapshot_Is_Equal.
//   - The fluent Assertions method API is gone; the linter bans methods that do not
//     satisfy a stdlib interface, so every assertion is a free function.
//   - testify's CallerInfo stack is gone; each assertion calls t.Helper, so the standard
//     library reports the failing line itself.
//   - Two operands that repeat a type would force a house input struct, so where the two
//     values are only compared or boxed they take distinct type parameters instead, which
//     keeps the plain call shape. Only the arithmetic and ordering assertions, whose two
//     operands share one concrete type, use an input struct, exactly as fixedpoint does.
//   - Failure values render with fmt, not the vendored go-spew, and diffs render with
//     shared/diff/myers, not the vendored go-difflib.
//   - InDelta and InEpsilon were float64, which a deterministic package bans, so they are
//     rebuilt on shared/math/fixedpoint. WithinDuration compares shared/time moments.
//   - The environment-dependent assertions take an injected Asserter: File_Exists reads an
//     fs.FS, and Eventually and Never poll on an injected io loop the caller drives.
//   - HTTP assertions, YAMLEq (JSON_Eq stays), and the mock, suite, and http sub-packages
//     are not ported.
package testify

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"local/james-orcales/shared/diff/myers"
	"local/james-orcales/shared/math/fixedpoint"
	"local/james-orcales/shared/simulation/time"
)

// Asserter carries the ambient collaborators the environment-dependent assertions need.
// The pure assertions ignore it and take only the testing handle; only the File and
// Eventually families reach through it. The sibling testify/default binds a Default
// Asserter to the host OS so callers rarely construct one by hand.
type Asserter struct {
	// File_System backs the File_Exists and Dir_Exists family. Paths handed to those
	// assertions are operating-system absolute; the leading separator is stripped first.
	File_System fs.FS
	// Clock measures elapsed time for Eventually and Never against Now_Monotonic.
	Clock time.Clock
	// IO is the loop the Eventually and Never poll rides: the assertion arms a repeating
	// Timeout and the caller's driver fires it, because a library never drives the loop.
	IO *time.Timeline
}

// Panics with message when condition is false. It is the local stand-in for a
// precondition the caller can only get wrong in code, never at runtime.
func assert(condition bool, message string) {
	if !condition {
		panic(message)
	}
}

// Fail reports failure_message, plus any trailing caller message, and terminates the test.
// Its bool result remains for source compatibility with existing assertion call sites.
func Fail(t *testing.T, failure_message string, message_and_args ...any) (passed bool) {
	t.Helper()
	message := failure_message
	extra := message_from_message_and_args(message_and_args...)
	if extra != "" {
		message = message + "\nMessages: " + extra
	}
	t.Errorf("%s", message)
	t.FailNow()
	return false
}

// Fail_Now remains an explicit spelling for Fail, which already aborts the test.
func Fail_Now(t *testing.T, failure_message string, message_and_args ...any) (passed bool) {
	return Fail(t, failure_message, message_and_args...)
}

// Renders testify's trailing message tail: a lone string is the message, a lone non-string
// is formatted, and a leading format string consumes the rest as its arguments.
func message_from_message_and_args(message_and_args ...any) (message string) {
	if len(message_and_args) == 0 {
		return ""
	}
	if len(message_and_args) == 1 {
		text, ok := message_and_args[0].(string)
		if ok {
			return text
		}
		return fmt.Sprintf("%+v", message_and_args[0])
	}
	format, ok := message_and_args[0].(string)
	if ok {
		return fmt.Sprintf(format, message_and_args[1:]...)
	}
	return fmt.Sprintf("%+v", message_and_args)
}

// Objects_Are_Equal reports structural equality: identical nil-ness, bytes.Equal for two
// byte slices, else reflect.DeepEqual.
func Objects_Are_Equal[E, A any](expected E, actual A) (equal bool) {
	boxed_expected := any(expected)
	boxed_actual := any(actual)
	if boxed_expected == nil {
		return boxed_actual == nil
	}
	if boxed_actual == nil {
		return false
	}
	expected_bytes, expected_ok := boxed_expected.([]byte)
	if !expected_ok {
		return reflect.DeepEqual(boxed_expected, boxed_actual)
	}
	actual_bytes, actual_ok := boxed_actual.([]byte)
	if !actual_ok {
		return false
	}
	if expected_bytes == nil {
		return actual_bytes == nil
	}
	if actual_bytes == nil {
		return false
	}
	return bytes.Equal(expected_bytes, actual_bytes)
}

// Objects_Are_Equal_Values reports equality after a convertibility coercion, so values that
// differ only by numeric type compare equal. Both numeric operands widen to the larger type
// first to avoid overflow false positives.
func Objects_Are_Equal_Values[E, A any](expected E, actual A) (equal bool) {
	if Objects_Are_Equal(expected, actual) {
		return true
	}
	expected_value := reflect.ValueOf(any(expected))
	actual_value := reflect.ValueOf(any(actual))
	if !expected_value.IsValid() {
		return false
	}
	if !actual_value.IsValid() {
		return false
	}
	expected_type := expected_value.Type()
	actual_type := actual_value.Type()
	if !expected_type.ConvertibleTo(actual_type) {
		return false
	}
	if !is_numeric_type(expected_type) {
		converted := expected_value.Convert(actual_type).Interface()
		return reflect.DeepEqual(converted, any(actual))
	}
	if !is_numeric_type(actual_type) {
		converted := expected_value.Convert(actual_type).Interface()
		return reflect.DeepEqual(converted, any(actual))
	}
	if expected_type.Size() >= actual_type.Size() {
		return actual_value.Convert(expected_type).Interface() == any(expected)
	}
	return expected_value.Convert(actual_type).Interface() == any(actual)
}

// Reports whether a type is one of the integer, float, or complex kinds.
func is_numeric_type(candidate_type reflect.Type) (numeric bool) {
	return candidate_type.Kind() >= reflect.Int && candidate_type.Kind() <= reflect.Complex128
}

// Is_Nil reports whether a value is nil, seeing through a typed nil held in a channel,
// func, interface, map, pointer, slice, or unsafe pointer.
func Is_Nil(object any) (empty bool) {
	if object == nil {
		return true
	}
	value := reflect.ValueOf(object)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface,
		reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		return value.IsNil()
	}
	return false
}

// Is_Empty reports whether a value is its zero value, or an empty collection, string, or a
// nil-or-empty-pointed pointer.
func Is_Empty(object any) (empty bool) {
	if object == nil {
		return true
	}
	return is_empty_value(reflect.ValueOf(object))
}

// Reports Is_Empty over a reflect.Value, descending through pointer chains so a non-nil
// pointer is empty when its pointee is.
func is_empty_value(value reflect.Value) (empty bool) {
	for value.Kind() == reflect.Ptr {
		if value.IsZero() {
			return true
		}
		value = value.Elem()
	}
	if value.IsZero() {
		return true
	}
	switch value.Kind() {
	case reflect.Chan, reflect.Map, reflect.Slice:
		return value.Len() == 0
	}
	return false
}

// Same_Pointers reports whether two values are pointers of the same type addressing the
// same object. addressable is false when either argument is not a pointer.
func Same_Pointers[F, S any](first F, second S) (same bool, addressable bool) {
	boxed_first := any(first)
	boxed_second := any(second)
	first_value := reflect.ValueOf(boxed_first)
	second_value := reflect.ValueOf(boxed_second)
	if first_value.Kind() != reflect.Ptr {
		return false, false
	}
	if second_value.Kind() != reflect.Ptr {
		return false, false
	}
	if reflect.TypeOf(boxed_first) != reflect.TypeOf(boxed_second) {
		return false, true
	}
	return boxed_first == boxed_second, true
}

// Contains_Element reports whether list holds element: a substring for a string, a present
// key for a map, an equal member for an array or slice. searchable is false when list is
// not a searchable kind.
func Contains_Element[L, E any](list L, element E) (searchable bool, found bool) {
	boxed_list := any(list)
	list_value := reflect.ValueOf(boxed_list)
	list_type := reflect.TypeOf(boxed_list)
	if list_type == nil {
		return false, false
	}
	defer func() {
		if recover() != nil {
			searchable = false
			found = false
		}
	}()
	if list_type.Kind() == reflect.String {
		fragment := reflect.ValueOf(any(element)).String()
		return true, strings.Contains(list_value.String(), fragment)
	}
	if list_type.Kind() == reflect.Map {
		for _, key := range list_value.MapKeys() {
			if Objects_Are_Equal(key.Interface(), element) {
				return true, true
			}
		}
		return true, false
	}
	for member_index := 0; member_index < list_value.Len(); member_index++ {
		if Objects_Are_Equal(list_value.Index(member_index).Interface(), element) {
			return true, true
		}
	}
	return true, false
}

// Diff_Lists returns the elements present only in list_a and only in list_b, counting
// duplicates separately and ignoring order — the core of Elements_Match.
func Diff_Lists[A, B any](list_a A, list_b B) (extra_a []any, extra_b []any) {
	a_value := reflect.ValueOf(any(list_a))
	b_value := reflect.ValueOf(any(list_b))
	a_size := a_value.Len()
	b_size := b_value.Len()
	visited := make([]bool, b_size)
	for a_index := 0; a_index < a_size; a_index++ {
		element := a_value.Index(a_index).Interface()
		found := false
		for b_index := 0; b_index < b_size; b_index++ {
			if visited[b_index] {
				continue
			}
			if Objects_Are_Equal(b_value.Index(b_index).Interface(), element) {
				visited[b_index] = true
				found = true
				break
			}
		}
		if !found {
			extra_a = append(extra_a, element)
		}
	}
	for b_index := 0; b_index < b_size; b_index++ {
		if visited[b_index] {
			continue
		}
		extra_b = append(extra_b, b_value.Index(b_index).Interface())
	}
	return extra_a, extra_b
}

// Equal asserts that expected and actual are structurally equal.
func Equal[E, A any](
	t *testing.T, expected E, actual A, message_and_args ...any,
) (equal bool) {
	t.Helper()
	invalid := validate_equal_args(expected, actual)
	if invalid != nil {
		message := fmt.Sprintf("Invalid operation: %#v == %#v (%s)",
			any(expected), any(actual), invalid)
		return Fail(t, message, message_and_args...)
	}
	if Objects_Are_Equal(expected, actual) {
		return true
	}
	expected_text, actual_text := format_unequal_values(expected, actual)
	message := fmt.Sprintf("Not equal: \nexpected: %s\nactual  : %s%s",
		expected_text, actual_text, value_diff(expected, actual))
	return Fail(t, message, message_and_args...)
}

// Not_Equal asserts that expected and actual are not structurally equal.
func Not_Equal[E, A any](
	t *testing.T, expected E, actual A, message_and_args ...any,
) (unequal bool) {
	t.Helper()
	invalid := validate_equal_args(expected, actual)
	if invalid != nil {
		message := fmt.Sprintf("Invalid operation: %#v != %#v (%s)",
			any(expected), any(actual), invalid)
		return Fail(t, message, message_and_args...)
	}
	if Objects_Are_Equal(expected, actual) {
		message := fmt.Sprintf("Should not be equal: %#v", any(actual))
		return Fail(t, message, message_and_args...)
	}
	return true
}

// Equal_Values asserts equality after a numeric or convertible coercion.
func Equal_Values[E, A any](
	t *testing.T, expected E, actual A, message_and_args ...any,
) (equal bool) {
	t.Helper()
	if Objects_Are_Equal_Values(expected, actual) {
		return true
	}
	expected_text, actual_text := format_unequal_values(expected, actual)
	message := fmt.Sprintf("Not equal: \nexpected: %s\nactual  : %s%s",
		expected_text, actual_text, value_diff(expected, actual))
	return Fail(t, message, message_and_args...)
}

// Not_Equal_Values asserts inequality even after coercion to a common type.
func Not_Equal_Values[E, A any](
	t *testing.T, expected E, actual A, message_and_args ...any,
) (unequal bool) {
	t.Helper()
	if Objects_Are_Equal_Values(expected, actual) {
		message := fmt.Sprintf("Should not be equal: %#v", any(actual))
		return Fail(t, message, message_and_args...)
	}
	return true
}

// Exactly asserts that expected and actual have identical dynamic type and equal value.
func Exactly[E, A any](
	t *testing.T, expected E, actual A, message_and_args ...any,
) (equal bool) {
	t.Helper()
	if reflect.TypeOf(any(expected)) != reflect.TypeOf(any(actual)) {
		message := fmt.Sprintf("Types expected to match exactly\n\t%T != %T",
			any(expected), any(actual))
		return Fail(t, message, message_and_args...)
	}
	return Equal(t, expected, actual, message_and_args...)
}

// Same asserts that expected and actual are pointers to the same object.
func Same[E, A any](
	t *testing.T, expected E, actual A, message_and_args ...any,
) (same bool) {
	t.Helper()
	pointer_same, addressable := Same_Pointers(expected, actual)
	if !addressable {
		return Fail(t, "Both arguments must be pointers", message_and_args...)
	}
	if !pointer_same {
		message := fmt.Sprintf("Not same: \nexpected: %#v\nactual  : %#v",
			any(expected), any(actual))
		return Fail(t, message, message_and_args...)
	}
	return true
}

// Not_Same asserts that expected and actual are not pointers to the same object.
func Not_Same[E, A any](
	t *testing.T, expected E, actual A, message_and_args ...any,
) (distinct bool) {
	t.Helper()
	pointer_same, addressable := Same_Pointers(expected, actual)
	if !addressable {
		return Fail(t, "Both arguments must be pointers", message_and_args...)
	}
	if pointer_same {
		message := fmt.Sprintf("Expected and actual point to the same object: %#v",
			any(expected))
		return Fail(t, message, message_and_args...)
	}
	return true
}

// Rejects func-typed operands, whose equality is undecidable.
func validate_equal_args[E, A any](expected E, actual A) (invalid error) {
	if is_function(any(expected)) {
		return errors.New("cannot take func type as argument")
	}
	if is_function(any(actual)) {
		return errors.New("cannot take func type as argument")
	}
	return nil
}

// Reports whether argument's kind is Func.
func is_function(argument any) (function bool) {
	if argument == nil {
		return false
	}
	return reflect.TypeOf(argument).Kind() == reflect.Func
}

// Renders two values for a mismatch message, prefixing the type when the two differ so the
// reader can tell an int from an int64.
func format_unequal_values[E, A any](
	expected E, actual A,
) (expected_text string, actual_text string) {
	boxed_expected := any(expected)
	boxed_actual := any(actual)
	if reflect.TypeOf(boxed_expected) != reflect.TypeOf(boxed_actual) {
		expected_text = fmt.Sprintf("%T(%#v)", boxed_expected, boxed_expected)
		actual_text = fmt.Sprintf("%T(%#v)", boxed_actual, boxed_actual)
		return expected_text, actual_text
	}
	return fmt.Sprintf("%#v", boxed_expected), fmt.Sprintf("%#v", boxed_actual)
}

// Returns a myers line diff of two like-typed struct, map, slice, array, or string values,
// or an empty string when a diff would not help. It replaces testify's spew-and-difflib
// block.
func value_diff[E, A any](expected E, actual A) (difference string) {
	boxed_expected := any(expected)
	boxed_actual := any(actual)
	if boxed_expected == nil {
		return ""
	}
	if boxed_actual == nil {
		return ""
	}
	expected_type, expected_kind := type_and_kind(boxed_expected)
	actual_type, _ := type_and_kind(boxed_actual)
	if expected_type != actual_type {
		return ""
	}
	if !diffable_kind(expected_kind) {
		return ""
	}
	expected_text := render_for_diff(boxed_expected, expected_kind)
	actual_text := render_for_diff(boxed_actual, expected_kind)
	if expected_text == actual_text {
		return ""
	}
	if len(expected_text) > myers.TEXT_SIZE_MAXIMUM {
		return ""
	}
	if len(actual_text) > myers.TEXT_SIZE_MAXIMUM {
		return ""
	}
	workspace := myers.Workspace{
		Old_Runes: make(myers.Old_Rune_Storage, len(expected_text)+1),
		New_Runes: make(myers.New_Rune_Storage, len(actual_text)+1),
		Matrix: make(
			myers.Matrix_Storage, (len(expected_text)+2)*(len(actual_text)+2),
		),
	}
	output := make(myers.Line_Output, myers.LINE_DIFF_SIZE_MAXIMUM)
	count, status := myers.Line_Diff_Into(myers.Line_Diff_Input{
		Output:    output,
		Workspace: &workspace,
		Old:       myers.Line_Old_Text_Unvalidated(expected_text),
		New:       myers.Line_New_Text_Unvalidated(actual_text),
	})
	if status != myers.STATUS_OK {
		return ""
	}
	return "\n\nDiff:\n" + string(output[:count])
}

// Reports whether a kind renders into a diff worth showing.
func diffable_kind(kind reflect.Kind) (diffable bool) {
	switch kind {
	case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array, reflect.String:
		return true
	}
	return false
}

// Renders a value into the text a diff compares.
func render_for_diff(value any, kind reflect.Kind) (text string) {
	if kind == reflect.String {
		return reflect.ValueOf(value).String()
	}
	return fmt.Sprintf("%#v", value)
}

// Returns a value's type and kind, unwrapping one pointer.
func type_and_kind(value any) (value_type reflect.Type, kind reflect.Kind) {
	value_type = reflect.TypeOf(value)
	kind = value_type.Kind()
	if kind == reflect.Ptr {
		value_type = value_type.Elem()
		kind = value_type.Kind()
	}
	return value_type, kind
}

// Nil asserts that object is nil.
func Nil(t *testing.T, object any, message_and_args ...any) (empty bool) {
	t.Helper()
	if Is_Nil(object) {
		return true
	}
	return Fail(t, fmt.Sprintf("Expected nil, but got: %#v", object), message_and_args...)
}

// Not_Nil asserts that object is not nil.
func Not_Nil(t *testing.T, object any, message_and_args ...any) (present bool) {
	t.Helper()
	if !Is_Nil(object) {
		return true
	}
	return Fail(t, "Expected value not to be nil.", message_and_args...)
}

// True asserts that value is true.
func True(t *testing.T, value bool, message_and_args ...any) (correct bool) {
	t.Helper()
	if !value {
		return Fail(t, "Should be true", message_and_args...)
	}
	return true
}

// False asserts that value is false.
func False(t *testing.T, value bool, message_and_args ...any) (correct bool) {
	t.Helper()
	if value {
		return Fail(t, "Should be false", message_and_args...)
	}
	return true
}

// Zero asserts that value is the zero value of its type.
func Zero(t *testing.T, value any, message_and_args ...any) (empty bool) {
	t.Helper()
	if value == nil {
		return true
	}
	if reflect.DeepEqual(value, reflect.Zero(reflect.TypeOf(value)).Interface()) {
		return true
	}
	return Fail(t, fmt.Sprintf("Should be zero, but was %v", value), message_and_args...)
}

// Not_Zero asserts that value is not the zero value of its type.
func Not_Zero(t *testing.T, value any, message_and_args ...any) (nonzero bool) {
	t.Helper()
	if value == nil {
		return Fail(t, "Should not be zero, but was <nil>", message_and_args...)
	}
	if reflect.DeepEqual(value, reflect.Zero(reflect.TypeOf(value)).Interface()) {
		message := fmt.Sprintf("Should not be zero, but was %v", value)
		return Fail(t, message, message_and_args...)
	}
	return true
}

// Empty asserts that object is empty.
func Empty(t *testing.T, object any, message_and_args ...any) (empty bool) {
	t.Helper()
	if !Is_Empty(object) {
		message := fmt.Sprintf("Should be empty, but was %v", object)
		return Fail(t, message, message_and_args...)
	}
	return true
}

// Not_Empty asserts that object is not empty.
func Not_Empty(t *testing.T, object any, message_and_args ...any) (filled bool) {
	t.Helper()
	if Is_Empty(object) {
		message := fmt.Sprintf("Should NOT be empty, but was %v", object)
		return Fail(t, message, message_and_args...)
	}
	return true
}

// Count asserts that object holds the given number of elements. It is testify's Len.
func Count(t *testing.T, object any, count int, message_and_args ...any) (correct bool) {
	t.Helper()
	actual, countable := count_of(object)
	if !countable {
		return Fail(t, fmt.Sprintf("%v is not countable", object), message_and_args...)
	}
	if actual != count {
		message := fmt.Sprintf("%v should hold %d item(s), but holds %d",
			object, count, actual)
		return Fail(t, message, message_and_args...)
	}
	return true
}

// Returns the element count of x when x is a countable kind, else zero and false.
func count_of(x any) (count int, countable bool) {
	value := reflect.ValueOf(x)
	defer func() {
		countable = recover() == nil
	}()
	return value.Len(), true
}

// Contains asserts that haystack holds element.
func Contains[E comparable](
	t *testing.T, haystack []E, element E, message_and_args ...any,
) (found bool) {
	t.Helper()
	for _, candidate := range haystack {
		if candidate == element {
			return true
		}
	}
	message := fmt.Sprintf("%#v does not contain %#v", haystack, element)
	return Fail(t, message, message_and_args...)
}

// Not_Contains asserts that haystack does not hold element.
func Not_Contains[E comparable](
	t *testing.T, haystack []E, element E, message_and_args ...any,
) (absent bool) {
	t.Helper()
	for _, candidate := range haystack {
		if candidate == element {
			message := fmt.Sprintf("%#v should not contain %#v", haystack, element)
			return Fail(t, message, message_and_args...)
		}
	}
	return true
}

// Contains_Any asserts that a string, list, or map contains a substring, member, or key.
func Contains_Any[H, E any](
	t *testing.T, haystack H, element E, message_and_args ...any,
) (found bool) {
	t.Helper()
	searchable, contained := Contains_Element(haystack, element)
	if !searchable {
		message := fmt.Sprintf("%#v is not searchable", any(haystack))
		return Fail(t, message, message_and_args...)
	}
	if !contained {
		message := fmt.Sprintf("%#v does not contain %#v", any(haystack), any(element))
		return Fail(t, message, message_and_args...)
	}
	return true
}

// Not_Contains_Any asserts that a string, list, or map lacks a substring, member, or key.
func Not_Contains_Any[H, E any](
	t *testing.T, haystack H, element E, message_and_args ...any,
) (absent bool) {
	t.Helper()
	searchable, contained := Contains_Element(haystack, element)
	if !searchable {
		message := fmt.Sprintf("%#v is not searchable", any(haystack))
		return Fail(t, message, message_and_args...)
	}
	if contained {
		message := fmt.Sprintf("%#v should not contain %#v", any(haystack), any(element))
		return Fail(t, message, message_and_args...)
	}
	return true
}

// Subset asserts that list contains every element of subset. A nil subset always holds.
func Subset[L, S any](
	t *testing.T, list L, subset S, message_and_args ...any,
) (contained bool) {
	t.Helper()
	boxed_subset := any(subset)
	if boxed_subset == nil {
		return true
	}
	subset_value := reflect.ValueOf(boxed_subset)
	if subset_value.Kind() == reflect.Map {
		return subset_covers_map(t, any(list), subset_value, message_and_args...)
	}
	return subset_covers_list(t, any(list), subset_value, message_and_args...)
}

// Reports whether list contains every key-value pair of a map subset.
func subset_covers_map(
	t *testing.T, list any, subset_value reflect.Value, message_and_args ...any,
) (covered bool) {
	t.Helper()
	list_value := reflect.ValueOf(list)
	for _, key := range subset_value.MapKeys() {
		if list_value.Kind() != reflect.Map {
			searchable, found := Contains_Element(list, key.Interface())
			if !searchable {
				message := fmt.Sprintf("%#v is not searchable", list)
				return Fail(t, message, message_and_args...)
			}
			if !found {
				message := fmt.Sprintf("%#v does not contain %#v",
					list, key.Interface())
				return Fail(t, message, message_and_args...)
			}
			continue
		}
		list_entry := list_value.MapIndex(key)
		if !list_entry.IsValid() {
			message := fmt.Sprintf("%#v does not contain %#v",
				list, subset_value.Interface())
			return Fail(t, message, message_and_args...)
		}
		subset_entry := subset_value.MapIndex(key).Interface()
		if !Objects_Are_Equal(subset_entry, list_entry.Interface()) {
			message := fmt.Sprintf("%#v does not contain %#v",
				list, subset_value.Interface())
			return Fail(t, message, message_and_args...)
		}
	}
	return true
}

// Reports whether list contains every element of an array or slice subset.
func subset_covers_list(
	t *testing.T, list any, subset_value reflect.Value, message_and_args ...any,
) (covered bool) {
	t.Helper()
	for element_index := 0; element_index < subset_value.Len(); element_index++ {
		element := subset_value.Index(element_index).Interface()
		searchable, found := Contains_Element(list, element)
		if !searchable {
			message := fmt.Sprintf("%#v is not searchable", list)
			return Fail(t, message, message_and_args...)
		}
		if !found {
			message := fmt.Sprintf("%#v does not contain %#v", list, element)
			return Fail(t, message, message_and_args...)
		}
	}
	return true
}

// Not_Subset asserts that list is missing at least one element of subset.
func Not_Subset[L, S any](
	t *testing.T, list L, subset S, message_and_args ...any,
) (incomplete bool) {
	t.Helper()
	boxed_subset := any(subset)
	if boxed_subset == nil {
		message := "nil is the empty set which is a subset of every set"
		return Fail(t, message, message_and_args...)
	}
	subset_value := reflect.ValueOf(boxed_subset)
	for element_index := 0; element_index < subset_value.Len(); element_index++ {
		element := subset_value.Index(element_index).Interface()
		searchable, found := Contains_Element(any(list), element)
		if !searchable {
			return true
		}
		if !found {
			return true
		}
	}
	message := fmt.Sprintf("%#v is a subset of %#v", boxed_subset, any(list))
	return Fail(t, message, message_and_args...)
}

// Elements_Match asserts that two lists are equal as multisets, ignoring order.
func Elements_Match[A, B any](
	t *testing.T, list_a A, list_b B, message_and_args ...any,
) (matched bool) {
	t.Helper()
	if Is_Empty(list_a) {
		if Is_Empty(list_b) {
			return true
		}
	}
	if !is_list(t, any(list_a), message_and_args...) {
		return false
	}
	if !is_list(t, any(list_b), message_and_args...) {
		return false
	}
	extra_a, extra_b := Diff_Lists(list_a, list_b)
	if len(extra_a) == 0 {
		if len(extra_b) == 0 {
			return true
		}
	}
	return Fail(t, format_list_difference(extra_a, extra_b), message_and_args...)
}

// Not_Elements_Match asserts that two lists differ as multisets.
func Not_Elements_Match[A, B any](
	t *testing.T, list_a A, list_b B, message_and_args ...any,
) (mismatched bool) {
	t.Helper()
	if Is_Empty(list_a) {
		if Is_Empty(list_b) {
			message := "listA and listB contain the same elements"
			return Fail(t, message, message_and_args...)
		}
	}
	if !is_list(t, any(list_a), message_and_args...) {
		return false
	}
	if !is_list(t, any(list_b), message_and_args...) {
		return false
	}
	extra_a, extra_b := Diff_Lists(list_a, list_b)
	if len(extra_a) == 0 {
		if len(extra_b) == 0 {
			message := "listA and listB contain the same elements"
			return Fail(t, message, message_and_args...)
		}
	}
	return true
}

// Reports whether list is an array or slice, failing through t otherwise.
func is_list(t *testing.T, list any, message_and_args ...any) (listlike bool) {
	t.Helper()
	kind := reflect.TypeOf(list).Kind()
	if kind == reflect.Array {
		return true
	}
	if kind == reflect.Slice {
		return true
	}
	return Fail(t, fmt.Sprintf("%#v is not an array or slice", list), message_and_args...)
}

// Renders the extra-element report for Elements_Match.
func format_list_difference[A, B any](extra_a []A, extra_b []B) (message string) {
	var report strings.Builder
	report.WriteString("elements differ")
	if len(extra_a) > 0 {
		report.WriteString(fmt.Sprintf("\n\nextra elements in list A:\n%#v", extra_a))
	}
	if len(extra_b) > 0 {
		report.WriteString(fmt.Sprintf("\n\nextra elements in list B:\n%#v", extra_b))
	}
	return report.String()
}

// No_Error asserts that err is nil.
func No_Error(t *testing.T, err error, message_and_args ...any) (absent bool) {
	t.Helper()
	if err != nil {
		message := fmt.Sprintf("Received unexpected error:\n%+v", err)
		return Fail(t, message, message_and_args...)
	}
	return true
}

// Error asserts that err is not nil.
func Error(t *testing.T, err error, message_and_args ...any) (present bool) {
	t.Helper()
	if err == nil {
		return Fail(t, "An error is expected but got nil.", message_and_args...)
	}
	return true
}

// Equal_Error asserts that err is non-nil and its message equals err_string.
func Equal_Error(
	t *testing.T, err error, err_string string, message_and_args ...any,
) (equal bool) {
	t.Helper()
	if !Error(t, err, message_and_args...) {
		return false
	}
	if err.Error() != err_string {
		message := fmt.Sprintf("Error message not equal:\nexpected: %q\nactual  : %q",
			err_string, err.Error())
		return Fail(t, message, message_and_args...)
	}
	return true
}

// Error_Contains asserts that err is non-nil and its message contains contains.
func Error_Contains(
	t *testing.T, err error, contains string, message_and_args ...any,
) (found bool) {
	t.Helper()
	if !Error(t, err, message_and_args...) {
		return false
	}
	if !strings.Contains(err.Error(), contains) {
		message := fmt.Sprintf("Error %q does not contain %q", err.Error(), contains)
		return Fail(t, message, message_and_args...)
	}
	return true
}

// Error_Is asserts that target is in err's chain, wrapping errors.Is.
func Error_Is[E, T error](
	t *testing.T, err E, target T, message_and_args ...any,
) (matched bool) {
	t.Helper()
	if errors.Is(err, target) {
		return true
	}
	message := fmt.Sprintf("Target error should be in err chain:\nexpected: %v\nin chain: %+v",
		any(target), any(err))
	return Fail(t, message, message_and_args...)
}

// Not_Error_Is asserts that target is not in err's chain.
func Not_Error_Is[E, T error](
	t *testing.T, err E, target T, message_and_args ...any,
) (absent bool) {
	t.Helper()
	if !errors.Is(err, target) {
		return true
	}
	message := fmt.Sprintf("Target error should not be in err chain:\nfound: %v\nin chain: %+v",
		any(target), any(err))
	return Fail(t, message, message_and_args...)
}

// Error_As asserts that some error in err's chain matches target, wrapping errors.As.
func Error_As(
	t *testing.T, err error, target any, message_and_args ...any,
) (matched bool) {
	t.Helper()
	if errors.As(err, target) {
		return true
	}
	message := fmt.Sprintf("Should be in error chain:\nexpected: %s\nin chain: %+v",
		reflect.TypeOf(target).Elem().String(), err)
	return Fail(t, message, message_and_args...)
}

// Not_Error_As asserts that no error in err's chain matches target.
func Not_Error_As(
	t *testing.T, err error, target any, message_and_args ...any,
) (absent bool) {
	t.Helper()
	if !errors.As(err, target) {
		return true
	}
	message := fmt.Sprintf("Target error should not be in err chain:\nfound: %s\nin chain: %+v",
		reflect.TypeOf(target).Elem().String(), err)
	return Fail(t, message, message_and_args...)
}

// Implements asserts that object's type satisfies the interface named by a nil interface
// pointer, for example (*io.Reader)(nil).
func Implements[I, O any](
	t *testing.T, interface_pointer I, object O, message_and_args ...any,
) (satisfies bool) {
	t.Helper()
	interface_type := reflect.TypeOf(any(interface_pointer)).Elem()
	if any(object) == nil {
		message := fmt.Sprintf("Cannot check if nil implements %v", interface_type)
		return Fail(t, message, message_and_args...)
	}
	if !reflect.TypeOf(any(object)).Implements(interface_type) {
		message := fmt.Sprintf("%T must implement %v", any(object), interface_type)
		return Fail(t, message, message_and_args...)
	}
	return true
}

// Not_Implements asserts that object's type does not satisfy the named interface.
func Not_Implements[I, O any](
	t *testing.T, interface_pointer I, object O, message_and_args ...any,
) (lacks bool) {
	t.Helper()
	interface_type := reflect.TypeOf(any(interface_pointer)).Elem()
	if any(object) == nil {
		message := fmt.Sprintf("Cannot check if nil does not implement %v", interface_type)
		return Fail(t, message, message_and_args...)
	}
	if reflect.TypeOf(any(object)).Implements(interface_type) {
		message := fmt.Sprintf("%T implements %v", any(object), interface_type)
		return Fail(t, message, message_and_args...)
	}
	return true
}

// Is_Type asserts that object has the same dynamic type as expected_type.
func Is_Type[E, O any](
	t *testing.T, expected_type E, object O, message_and_args ...any,
) (same bool) {
	t.Helper()
	if Objects_Are_Equal(reflect.TypeOf(any(object)), reflect.TypeOf(any(expected_type))) {
		return true
	}
	message := fmt.Sprintf("Object expected to be of type %T, but was %T",
		any(expected_type), any(object))
	return Fail(t, message, message_and_args...)
}

// Is_Not_Type asserts that object has a different dynamic type than the_type.
func Is_Not_Type[E, O any](
	t *testing.T, the_type E, object O, message_and_args ...any,
) (different bool) {
	t.Helper()
	if !Objects_Are_Equal(reflect.TypeOf(any(object)), reflect.TypeOf(any(the_type))) {
		return true
	}
	message := fmt.Sprintf("Object type expected to be different than %T", any(the_type))
	return Fail(t, message, message_and_args...)
}

// Greater_Input pairs the two ordered operands of Greater, which repeat a type.
type Greater_Input[V cmp.Ordered] struct {
	// First is the value expected to be the larger.
	First V
	// Second is the value expected to be the smaller.
	Second V
}

// Greater asserts input.First is greater than input.Second.
func Greater[V cmp.Ordered](
	t *testing.T, input *Greater_Input[V], message_and_args ...any,
) (greater bool) {
	t.Helper()
	if input.First > input.Second {
		return true
	}
	message := fmt.Sprintf("%v is not greater than %v", input.First, input.Second)
	return Fail(t, message, message_and_args...)
}

// Greater_Or_Equal_Input pairs the two ordered operands of Greater_Or_Equal.
type Greater_Or_Equal_Input[V cmp.Ordered] struct {
	// First is the value expected to be the larger or equal.
	First V
	// Second is the value expected to be the smaller or equal.
	Second V
}

// Greater_Or_Equal asserts input.First is greater than or equal to input.Second.
func Greater_Or_Equal[V cmp.Ordered](
	t *testing.T, input *Greater_Or_Equal_Input[V], message_and_args ...any,
) (at_least bool) {
	t.Helper()
	if input.First >= input.Second {
		return true
	}
	message := fmt.Sprintf("%v is not greater than or equal to %v", input.First, input.Second)
	return Fail(t, message, message_and_args...)
}

// Less_Input pairs the two ordered operands of Less.
type Less_Input[V cmp.Ordered] struct {
	// First is the value expected to be the smaller.
	First V
	// Second is the value expected to be the larger.
	Second V
}

// Less asserts input.First is less than input.Second.
func Less[V cmp.Ordered](
	t *testing.T, input *Less_Input[V], message_and_args ...any,
) (less bool) {
	t.Helper()
	if input.First < input.Second {
		return true
	}
	message := fmt.Sprintf("%v is not less than %v", input.First, input.Second)
	return Fail(t, message, message_and_args...)
}

// Less_Or_Equal_Input pairs the two ordered operands of Less_Or_Equal.
type Less_Or_Equal_Input[V cmp.Ordered] struct {
	// First is the value expected to be the smaller or equal.
	First V
	// Second is the value expected to be the larger or equal.
	Second V
}

// Less_Or_Equal asserts input.First is less than or equal to input.Second.
func Less_Or_Equal[V cmp.Ordered](
	t *testing.T, input *Less_Or_Equal_Input[V], message_and_args ...any,
) (at_most bool) {
	t.Helper()
	if input.First <= input.Second {
		return true
	}
	message := fmt.Sprintf("%v is not less than or equal to %v", input.First, input.Second)
	return Fail(t, message, message_and_args...)
}

// Positive asserts value is greater than its type's zero.
func Positive[V cmp.Ordered](
	t *testing.T, value V, message_and_args ...any,
) (positive bool) {
	t.Helper()
	var zero V
	if value > zero {
		return true
	}
	return Fail(t, fmt.Sprintf("%v is not positive", value), message_and_args...)
}

// Negative asserts value is less than its type's zero.
func Negative[V cmp.Ordered](
	t *testing.T, value V, message_and_args ...any,
) (negative bool) {
	t.Helper()
	var zero V
	if value < zero {
		return true
	}
	return Fail(t, fmt.Sprintf("%v is not negative", value), message_and_args...)
}

// Panics asserts that callback panics.
func Panics(t *testing.T, callback func(), message_and_args ...any) (panicked bool) {
	t.Helper()
	raised, value := did_panic(callback)
	if !raised {
		message := fmt.Sprintf("func should panic\n\tPanic value:\t%#v", value)
		return Fail(t, message, message_and_args...)
	}
	return true
}

// Not_Panics asserts that callback does not panic.
func Not_Panics(t *testing.T, callback func(), message_and_args ...any) (calm bool) {
	t.Helper()
	raised, value := did_panic(callback)
	if raised {
		message := fmt.Sprintf("func should not panic\n\tPanic value:\t%v", value)
		return Fail(t, message, message_and_args...)
	}
	return true
}

// Panics_With_Value asserts that callback panics with a value equal to expected.
func Panics_With_Value[E any](
	t *testing.T, expected E, callback func(), message_and_args ...any,
) (panicked bool) {
	t.Helper()
	raised, value := did_panic(callback)
	if !raised {
		message := fmt.Sprintf("func should panic\n\tPanic value:\t%#v", value)
		return Fail(t, message, message_and_args...)
	}
	if !Objects_Are_Equal(value, expected) {
		message := fmt.Sprintf("func should panic with value:\t%#v\n\tPanic value:\t%#v",
			any(expected), value)
		return Fail(t, message, message_and_args...)
	}
	return true
}

// Panics_With_Error asserts that callback panics with an error whose message equals
// err_string.
func Panics_With_Error(
	t *testing.T, err_string string, callback func(), message_and_args ...any,
) (panicked bool) {
	t.Helper()
	raised, value := did_panic(callback)
	if !raised {
		message := fmt.Sprintf("func should panic\n\tPanic value:\t%#v", value)
		return Fail(t, message, message_and_args...)
	}
	panic_error, is_error := value.(error)
	if !is_error {
		message := fmt.Sprintf("func should panic with an error, but got %#v", value)
		return Fail(t, message, message_and_args...)
	}
	if panic_error.Error() != err_string {
		message := fmt.Sprintf("func should panic with error message %q, but got %q",
			err_string, panic_error.Error())
		return Fail(t, message, message_and_args...)
	}
	return true
}

// Runs callback and reports whether it panicked and the recovered value.
func did_panic(callback func()) (raised bool, recovered any) {
	raised = true
	defer func() {
		recovered = recover()
	}()
	callback()
	raised = false
	return raised, recovered
}

// In_Delta_Input pairs the operands of In_Delta, whose two values repeat a type.
type In_Delta_Input struct {
	// Expected is the reference value.
	Expected fixedpoint.Number
	// Actual is the value checked against Expected.
	Actual fixedpoint.Number
	// Delta is the largest allowed absolute difference.
	Delta fixedpoint.Number
}

// In_Delta asserts that input.Expected and input.Actual differ by at most input.Delta.
func In_Delta(t *testing.T, input *In_Delta_Input, message_and_args ...any) (within bool) {
	t.Helper()
	difference := fixedpoint_absolute(input.Expected - input.Actual)
	if difference <= input.Delta {
		return true
	}
	message := fmt.Sprintf("Max difference between %s and %s allowed is %s, but was %s",
		fixedpoint.Format(input.Expected, 6), fixedpoint.Format(input.Actual, 6),
		fixedpoint.Format(input.Delta, 6), fixedpoint.Format(difference, 6))
	return Fail(t, message, message_and_args...)
}

// In_Delta_Slice_Input pairs the operands of In_Delta_Slice, whose two slices repeat a type.
type In_Delta_Slice_Input struct {
	// Expected is the reference slice.
	Expected []fixedpoint.Number
	// Actual is the slice checked against Expected.
	Actual []fixedpoint.Number
	// Delta is the largest allowed absolute difference at each index.
	Delta fixedpoint.Number
}

// In_Delta_Slice asserts In_Delta elementwise over two equal-length slices.
func In_Delta_Slice(
	t *testing.T, input *In_Delta_Slice_Input, message_and_args ...any,
) (within bool) {
	t.Helper()
	if len(input.Expected) != len(input.Actual) {
		message := fmt.Sprintf("Slices must have the same length: %d != %d",
			len(input.Expected), len(input.Actual))
		return Fail(t, message, message_and_args...)
	}
	for element_index := range input.Expected {
		element := &In_Delta_Input{
			Expected: input.Expected[element_index],
			Actual:   input.Actual[element_index],
			Delta:    input.Delta,
		}
		if !In_Delta(t, element, message_and_args...) {
			return false
		}
	}
	return true
}

// In_Delta_Map_Values_Input pairs the operands of In_Delta_Map_Values, whose maps repeat a
// type.
type In_Delta_Map_Values_Input[K comparable] struct {
	// Expected is the reference map.
	Expected map[K]fixedpoint.Number
	// Actual is the map checked against Expected.
	Actual map[K]fixedpoint.Number
	// Delta is the largest allowed absolute difference at each key.
	Delta fixedpoint.Number
}

// In_Delta_Map_Values asserts In_Delta over the values of two maps with the same keys.
func In_Delta_Map_Values[K comparable](
	t *testing.T, input *In_Delta_Map_Values_Input[K], message_and_args ...any,
) (within bool) {
	t.Helper()
	if len(input.Expected) != len(input.Actual) {
		return Fail(t, "Maps must have the same number of keys", message_and_args...)
	}
	for key, expected_value := range input.Expected {
		actual_value, present := input.Actual[key]
		if !present {
			message := fmt.Sprintf("missing key %v in actual map", key)
			return Fail(t, message, message_and_args...)
		}
		element := &In_Delta_Input{
			Expected: expected_value,
			Actual:   actual_value,
			Delta:    input.Delta,
		}
		if !In_Delta(t, element, message_and_args...) {
			return false
		}
	}
	return true
}

// In_Epsilon_Input pairs the operands of In_Epsilon, whose two values repeat a type.
type In_Epsilon_Input struct {
	// Expected is the reference value; it must be non-zero.
	Expected fixedpoint.Number
	// Actual is the value checked against Expected.
	Actual fixedpoint.Number
	// Epsilon is the largest allowed relative error.
	Epsilon fixedpoint.Ratio
}

// In_Epsilon asserts that the relative error between the operands is at most Epsilon.
func In_Epsilon(t *testing.T, input *In_Epsilon_Input, message_and_args ...any) (within bool) {
	t.Helper()
	if input.Expected == 0 {
		message := "expected value must be non-zero to compute a relative error"
		return Fail(t, message, message_and_args...)
	}
	relative := fixedpoint.Divide(
		fixedpoint.Dividend(fixedpoint_absolute(input.Expected-input.Actual)),
		fixedpoint.Divisor(fixedpoint_absolute(input.Expected)),
	)
	// Number and Ratio share SCALE, so the relative error compares to Epsilon directly.
	if int64(relative) <= int64(input.Epsilon) {
		return true
	}
	message := fmt.Sprintf("Relative error is too high: %s allowed, but was %s",
		fixedpoint.Format(fixedpoint.Number(input.Epsilon), 6),
		fixedpoint.Format(relative, 6))
	return Fail(t, message, message_and_args...)
}

// In_Epsilon_Slice_Input pairs the operands of In_Epsilon_Slice, whose slices repeat a type.
type In_Epsilon_Slice_Input struct {
	// Expected is the reference slice.
	Expected []fixedpoint.Number
	// Actual is the slice checked against Expected.
	Actual []fixedpoint.Number
	// Epsilon is the largest allowed relative error at each index.
	Epsilon fixedpoint.Ratio
}

// In_Epsilon_Slice asserts In_Epsilon elementwise over two equal-length slices.
func In_Epsilon_Slice(
	t *testing.T, input *In_Epsilon_Slice_Input, message_and_args ...any,
) (within bool) {
	t.Helper()
	if len(input.Expected) != len(input.Actual) {
		message := fmt.Sprintf("Slices must have the same length: %d != %d",
			len(input.Expected), len(input.Actual))
		return Fail(t, message, message_and_args...)
	}
	for element_index := range input.Expected {
		element := &In_Epsilon_Input{
			Expected: input.Expected[element_index],
			Actual:   input.Actual[element_index],
			Epsilon:  input.Epsilon,
		}
		if !In_Epsilon(t, element, message_and_args...) {
			return false
		}
	}
	return true
}

// Returns the magnitude of a fixed-point number.
func fixedpoint_absolute(value fixedpoint.Number) (magnitude fixedpoint.Number) {
	if value < 0 {
		return -value
	}
	return value
}

// Within_Duration_Input pairs the two moments Within_Duration compares, which repeat a type.
type Within_Duration_Input struct {
	// Expected is the reference moment.
	Expected time.Moment
	// Actual is the moment checked against Expected.
	Actual time.Moment
	// Delta is the largest allowed separation between Expected and Actual.
	Delta time.Duration
}

// Within_Duration asserts that the two moments are within input.Delta of each other.
func Within_Duration(
	t *testing.T, input *Within_Duration_Input, message_and_args ...any,
) (within bool) {
	t.Helper()
	difference := input.Expected - input.Actual
	delta := time.Moment(input.Delta)
	if difference >= -delta {
		if difference <= delta {
			return true
		}
	}
	message := fmt.Sprintf("Max difference between %d and %d allowed is %d ns, but was %d ns",
		int64(input.Expected), int64(input.Actual), int64(input.Delta), int64(difference))
	return Fail(t, message, message_and_args...)
}

// Within_Range_Input pairs the three moments Within_Range compares, which repeat a type.
type Within_Range_Input struct {
	// Actual is the moment checked against the range.
	Actual time.Moment
	// Start is the inclusive lower bound of the range.
	Start time.Moment
	// End is the inclusive upper bound of the range.
	End time.Moment
}

// Within_Range asserts that input.Actual lies within the inclusive range.
func Within_Range(
	t *testing.T, input *Within_Range_Input, message_and_args ...any,
) (within bool) {
	t.Helper()
	if input.End < input.Start {
		return Fail(t, "Start should be before end", message_and_args...)
	}
	if input.Actual < input.Start {
		return Fail(t, within_range_message(input), message_and_args...)
	}
	if input.Actual > input.End {
		return Fail(t, within_range_message(input), message_and_args...)
	}
	return true
}

// Renders the out-of-range failure text.
func within_range_message(input *Within_Range_Input) (message string) {
	return fmt.Sprintf("Moment %d expected to be in range %d to %d",
		int64(input.Actual), int64(input.Start), int64(input.End))
}

// Regexp asserts that pattern (a *regexp.Regexp or a string source) matches subject.
func Regexp[P, S any](
	t *testing.T, pattern P, subject S, message_and_args ...any,
) (matched bool) {
	t.Helper()
	if regexp_matches(pattern, subject) {
		return true
	}
	message := fmt.Sprintf("Expect %q to match %q",
		fmt.Sprint(any(subject)), fmt.Sprint(any(pattern)))
	return Fail(t, message, message_and_args...)
}

// Not_Regexp asserts that pattern does not match subject.
func Not_Regexp[P, S any](
	t *testing.T, pattern P, subject S, message_and_args ...any,
) (unmatched bool) {
	t.Helper()
	if !regexp_matches(pattern, subject) {
		return true
	}
	message := fmt.Sprintf("Expect %q to NOT match %q",
		fmt.Sprint(any(subject)), fmt.Sprint(any(pattern)))
	return Fail(t, message, message_and_args...)
}

// Reports whether pattern matches subject, compiling a string pattern.
func regexp_matches[P, S any](pattern P, subject S) (matched bool) {
	boxed_pattern := any(pattern)
	var expression *regexp.Regexp
	compiled, is_compiled := boxed_pattern.(*regexp.Regexp)
	if is_compiled {
		expression = compiled
	} else {
		expression = regexp.MustCompile(fmt.Sprint(boxed_pattern))
	}
	boxed_subject := any(subject)
	text, is_bytes := boxed_subject.([]byte)
	if is_bytes {
		return expression.Match(text)
	}
	return expression.MatchString(fmt.Sprint(boxed_subject))
}

// JSON_Eq asserts that two JSON documents are semantically equal.
func JSON_Eq[E, A ~string](
	t *testing.T, expected E, actual A, message_and_args ...any,
) (equal bool) {
	t.Helper()
	var expected_value any
	expected_error := json.Unmarshal([]byte(string(expected)), &expected_value)
	if expected_error != nil {
		message := fmt.Sprintf("Expected value (%q) is not valid json: %s",
			string(expected), expected_error)
		return Fail(t, message, message_and_args...)
	}
	if string(actual) == string(expected) {
		return true
	}
	var actual_value any
	actual_error := json.Unmarshal([]byte(string(actual)), &actual_value)
	if actual_error != nil {
		message := fmt.Sprintf("Input (%q) needs to be valid json: %s",
			string(actual), actual_error)
		return Fail(t, message, message_and_args...)
	}
	return Equal(t, expected_value, actual_value, message_and_args...)
}

// Zero_Allocation owns bounded run count so caller cannot turn assertion into
// unbounded work.
func Zero_Allocation(
	t *testing.T, callback func(), message_and_args ...any,
) (zero bool) {
	t.Helper()
	const RUN_COUNT = 1
	allocations := testing.AllocsPerRun(RUN_COUNT, callback)
	if allocations == 0 {
		return true
	}
	message := fmt.Sprintf("callback allocated %.1f times per run; want zero", allocations)
	return Fail(t, message, message_and_args...)
}

// Condition asserts that comparison returns true.
func Condition(
	t *testing.T, comparison func() (satisfied bool), message_and_args ...any,
) (met bool) {
	t.Helper()
	if comparison() {
		return true
	}
	return Fail(t, "Condition failed!", message_and_args...)
}

// Asserter_File_Exists asserts that a's File_System holds a file, not a directory, at path.
func Asserter_File_Exists(
	a *Asserter, t *testing.T, path string, message_and_args ...any,
) (exists bool) {
	t.Helper()
	information, err := fs.Stat(a.File_System, file_system_path(path))
	if err != nil {
		message := fmt.Sprintf("unable to find file %q: %v", path, err)
		return Fail(t, message, message_and_args...)
	}
	if information.IsDir() {
		return Fail(t, fmt.Sprintf("%q is a directory", path), message_and_args...)
	}
	return true
}

// Asserter_No_File_Exists asserts that a's File_System holds no file at path.
func Asserter_No_File_Exists(
	a *Asserter, t *testing.T, path string, message_and_args ...any,
) (absent bool) {
	t.Helper()
	information, err := fs.Stat(a.File_System, file_system_path(path))
	if err != nil {
		return true
	}
	if information.IsDir() {
		return true
	}
	return Fail(t, fmt.Sprintf("file %q exists", path), message_and_args...)
}

// Asserter_Directory_Exists asserts that a's File_System holds a directory at path.
func Asserter_Directory_Exists(
	a *Asserter, t *testing.T, path string, message_and_args ...any,
) (exists bool) {
	t.Helper()
	information, err := fs.Stat(a.File_System, file_system_path(path))
	if err != nil {
		message := fmt.Sprintf("unable to find directory %q: %v", path, err)
		return Fail(t, message, message_and_args...)
	}
	if !information.IsDir() {
		return Fail(t, fmt.Sprintf("%q is a file", path), message_and_args...)
	}
	return true
}

// Asserter_No_Directory_Exists asserts that a's File_System holds no directory at path.
func Asserter_No_Directory_Exists(
	a *Asserter, t *testing.T, path string, message_and_args ...any,
) (absent bool) {
	t.Helper()
	information, err := fs.Stat(a.File_System, file_system_path(path))
	if err != nil {
		return true
	}
	if !information.IsDir() {
		return true
	}
	return Fail(t, fmt.Sprintf("directory %q exists", path), message_and_args...)
}

// Turns an operating-system absolute path into an fs.FS path: fs.FS names are slash-rooted
// and carry no leading separator, so a bound os.DirFS resolves the original path.
func file_system_path(path string) (name string) {
	return strings.TrimPrefix(path, "/")
}

// Asserter_Eventually_Input pairs the two durations of Asserter_Eventually, which repeat a
// type.
type Asserter_Eventually_Input struct {
	// Wait is how long the condition has to hold before the poll gives up.
	Wait time.Duration
	// Tick is the interval between successive polls of the condition.
	Tick time.Duration
}

// Asserter_Eventually arms a repeating poll of condition on a's io loop; if condition never
// holds before input.Wait elapses it reports failure through t. It is asynchronous: the
// caller drives the loop and the verdict lands during that drive, because a library may
// never drive the loop itself.
func Asserter_Eventually(
	a *Asserter,
	t *testing.T,
	condition func() (satisfied bool),
	input *Asserter_Eventually_Input,
) {
	t.Helper()
	assert(a.Clock.Now_Monotonic != nil, "testify: Asserter clock is required for Eventually")
	assert(a.IO != nil, "testify: Asserter io is required for Eventually")
	deadline := time.Clock_Now_Monotonic(a.Clock) + time.Monotonic_Moment(input.Wait)
	var poll time.Callback
	poll = func(completion *time.Completion) {
		if condition() {
			return
		}
		if time.Clock_Now_Monotonic(a.Clock) >= deadline {
			message := fmt.Sprintf(
				"Condition never satisfied within %d ns", int64(input.Wait))
			Fail(t, message)
			return
		}
		time.Timeline_Timeout(*a.IO, completion, input.Tick, poll)
	}
	time.Timeline_Timeout(*a.IO, &time.Completion{}, input.Tick, poll)
}

// Asserter_Never_Input pairs the two durations of Asserter_Never, which repeat a type.
type Asserter_Never_Input struct {
	// Wait is how long the condition must stay false.
	Wait time.Duration
	// Tick is the interval between successive polls of the condition.
	Tick time.Duration
}

// Asserter_Never arms a repeating poll of condition on a's io loop; if condition ever holds
// before input.Wait elapses it reports failure through t. Like Asserter_Eventually it is
// asynchronous and the caller drives the loop.
func Asserter_Never(
	a *Asserter, t *testing.T, condition func() (satisfied bool), input *Asserter_Never_Input,
) {
	t.Helper()
	assert(a.Clock.Now_Monotonic != nil, "testify: Asserter clock is required for Never")
	assert(a.IO != nil, "testify: Asserter io is required for Never")
	deadline := time.Clock_Now_Monotonic(a.Clock) + time.Monotonic_Moment(input.Wait)
	var poll time.Callback
	poll = func(completion *time.Completion) {
		if condition() {
			Fail(t, "Condition satisfied, but should never be")
			return
		}
		if time.Clock_Now_Monotonic(a.Clock) >= deadline {
			return
		}
		time.Timeline_Timeout(*a.IO, completion, input.Tick, poll)
	}
	time.Timeline_Timeout(*a.IO, &time.Completion{}, input.Tick, poll)
}
