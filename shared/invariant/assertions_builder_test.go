//go:build !noassert

package invariant_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"local/james-orcales/shared/invariant"
)

// Test_Assertions_Defers_Coverage_Until_Ensure protects the chain's atomic commit boundary.
func Test_Assertions_Defers_Coverage_Until_Ensure(t *testing.T) {
	recorder, _, code := registered_fixture(`package fixture
func check(value bool) {
	invariant.Assertions("check").Sometimes(value, "observed").Ensure()
}
`)
	if code != -1 {
		t.Fatalf("registration exit = %d", code)
	}
	metadata := chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "check", Ordinal: 0, Message: "observed",
	})
	builder := invariant.Recorder_Assertions(recorder, "check").
		Sometimes(true, "observed")
	if metadata.Frequency.Load() != 0 {
		t.Fatal("Sometimes credited before Ensure")
	}
	builder.Ensure()
	if metadata.Frequency.Load() != 1 {
		t.Fatal("Ensure did not credit the observed branch")
	}
}

// Test_Assertions_Records_Independent_Branches_Without_Tuples prevents grid semantics returning.
func Test_Assertions_Records_Independent_Branches_Without_Tuples(t *testing.T) {
	recorder, _, code := registered_fixture(`package fixture
func check(left bool, right bool) {
	invariant.Assertions("check").
		Sometimes(left, "same").
		Sometimes(right, "same").
		Ensure()
}
`)
	if code != -1 {
		t.Fatalf("registration exit = %d", code)
	}
	invariant.Recorder_Assertions(recorder, "check").
		Sometimes(true, "same").
		Sometimes(false, "same").
		Ensure()
	first := chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "check", Ordinal: 0, Message: "same",
	})
	second := chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "check", Ordinal: 1, Message: "same",
	})
	if first.Frequency.Load() != 1 {
		t.Fatal("Ensure did not preserve the first ordinal observation")
	}
	if second.False_Frequency.Load() != 1 {
		t.Fatal("Ensure did not preserve ordinal-separated branch observations")
	}
	recorder.Events.Range(func(key any, _ any) (continue_iteration bool) {
		if strings.Contains(key.(string), ":tuple=") {
			t.Fatalf("tuple event survived: %q", key)
		}
		return true
	})
}

// Test_Assertions_Defers_Range_Failure_And_Credits_Atomically protects preset atomicity.
func Test_Assertions_Defers_Range_Failure_And_Credits_Atomically(t *testing.T) {
	recorder, _, code := registered_fixture(`package fixture
const Minimum = 0
const Maximum = 10
func check(value int) {
	invariant.Assertions("bounded").Range_Int(value, Minimum, Maximum).Ensure()
}
`)
	if code != -1 {
		t.Fatalf("registration exit = %d", code)
	}
	builder := invariant.Recorder_Assertions(recorder, "bounded").Range_Int(11, 0, 10)
	if count_covered(recorder) != 0 {
		t.Fatal("Range credited before Ensure")
	}
	message := panic_text(builder.Ensure)
	if !strings.Contains(message, "exceeds max") {
		t.Fatalf("panic = %q", message)
	}
	if count_covered(recorder) != 0 {
		t.Fatal("failed Ensure partially credited the plan")
	}
}

// Test_Assertions_Defers_Enum_Failure keeps membership enforcement at Ensure.
func Test_Assertions_Defers_Enum_Failure(t *testing.T) {
	builder := invariant.Recorder_Assertions(&invariant.Recorder{}, "mode").
		Enum_Int(3, 1, 2)
	message := panic_text(builder.Ensure)
	if !strings.Contains(message, "not a member") {
		t.Fatalf("panic = %q", message)
	}
}

// Test_Assertions_Has_Zero_Allocations protects the builder's register-value shape.
func Test_Assertions_Has_Zero_Allocations(t *testing.T) {
	recorder := &invariant.Recorder{}
	allocations := testing.AllocsPerRun(1000, func() {
		invariant.Recorder_Assertions(recorder, "allocation").
			Sometimes(true, "axis").Range_Int(5, 0, 10).Ensure()
	})
	if allocations != 0 {
		t.Fatalf("allocations = %f, want 0", allocations)
	}
	recording, _, code := registered_fixture(`package fixture
func check(value bool) {
	invariant.Assertions("allocation").Sometimes(value, "axis").Ensure()
}
`)
	if code != -1 {
		t.Fatalf("registration exit = %d", code)
	}
	allocations = testing.AllocsPerRun(1000, func() {
		invariant.Recorder_Assertions(recording, "allocation").
			Sometimes(true, "axis").Ensure()
	})
	if allocations != 0 {
		t.Fatalf("recording allocations = %f, want 0", allocations)
	}
	allocations = testing.AllocsPerRun(1000, func() {
		benchmark_dense_root(recorder, 5)
	})
	if allocations != 0 {
		t.Fatalf("dense enforcement allocations = %f, want 0", allocations)
	}
}

// Test_Assertions_Typed_Presets_Register_Every_Integer_Width protects exact static resolution.
func Test_Assertions_Typed_Presets_Register_Every_Integer_Width(t *testing.T) {
	fixtures := []struct {
		Value_Type string
		Method     string
		Minimum    string
		Maximum    string
	}{
		{"int", "Range_Int", "-9223372036854775808", "9223372036854775807"},
		{"int8", "Range_Int8", "-128", "127"},
		{"int16", "Range_Int16", "-32768", "32767"},
		{"int32", "Range_Int32", "-2147483648", "2147483647"},
		{"int64", "Range_Int64", "-9223372036854775808", "9223372036854775807"},
		{"uint", "Range_Uint", "0", "18446744073709551615"},
		{"uint8", "Range_Uint8", "0", "255"},
		{"uint16", "Range_Uint16", "0", "65535"},
		{"uint32", "Range_Uint32", "0", "4294967295"},
		{"uint64", "Range_Uint64", "0", "18446744073709551615"},
	}
	for _, fixture := range fixtures {
		source := fmt.Sprintf(`package fixture
func check(value %s) {
	invariant.Assertions(%q).%s(value, %s, %s).Ensure()
}
`, fixture.Value_Type, fixture.Method, fixture.Method, fixture.Minimum, fixture.Maximum)
		_, output, code := registered_fixture(source)
		if code != -1 {
			t.Fatalf("%s exit=%d output=%q", fixture.Method, code, output.String())
		}
	}
}

// Test_Assertions_Typed_Presets_Execute_Every_Integer_Width prevents wrapper drift.
func Test_Assertions_Typed_Presets_Execute_Every_Integer_Width(t *testing.T) {
	recorder := &invariant.Recorder{}
	builders := []invariant.Assertion_Builder{
		invariant.Recorder_Assertions(recorder, "int8").Range_Int8(1, 0, 2),
		invariant.Recorder_Assertions(recorder, "int16").Range_Int16(1, 0, 2),
		invariant.Recorder_Assertions(recorder, "int32").Range_Int32(1, 0, 2),
		invariant.Recorder_Assertions(recorder, "int64").Range_Int64(1, 0, 2),
		invariant.Recorder_Assertions(recorder, "uint").Range_Uint(1, 0, 2),
		invariant.Recorder_Assertions(recorder, "uint8").Range_Uint8(1, 0, 2),
		invariant.Recorder_Assertions(recorder, "uint16").Range_Uint16(1, 0, 2),
		invariant.Recorder_Assertions(recorder, "uint32").Range_Uint32(1, 0, 2),
		invariant.Recorder_Assertions(recorder, "uint64").Range_Uint64(1, 0, 2),
		invariant.Recorder_Assertions(recorder, "enum-int8").Enum_Int8(1, 1, 2),
		invariant.Recorder_Assertions(recorder, "enum-int16").Enum_Int16(1, 1, 2),
		invariant.Recorder_Assertions(recorder, "enum-int32").Enum_Int32(1, 1, 2),
		invariant.Recorder_Assertions(recorder, "enum-int64").Enum_Int64(1, 1, 2),
		invariant.Recorder_Assertions(recorder, "enum-uint").Enum_Uint(1, 1, 2),
		invariant.Recorder_Assertions(recorder, "enum-uint8").Enum_Uint8(1, 1, 2),
		invariant.Recorder_Assertions(recorder, "enum-uint16").Enum_Uint16(1, 1, 2),
		invariant.Recorder_Assertions(recorder, "enum-uint32").Enum_Uint32(1, 1, 2),
		invariant.Recorder_Assertions(recorder, "enum-uint64").Enum_Uint64(1, 1, 2),
	}
	for _, builder := range builders {
		builder.Ensure()
	}
}

// Test_Assertions_Defers_Every_Value_Failure keeps runtime failure precedence at Ensure.
func Test_Assertions_Defers_Every_Value_Failure(t *testing.T) {
	recorder := &invariant.Recorder{}
	failures := []struct {
		Builder invariant.Assertion_Builder
		Message string
	}{
		{invariant.Recorder_Assertions(recorder, "lower").Range_Int(-1, 0, 2),
			"below min"},
		{invariant.Recorder_Assertions(recorder, "excluded").Range_Int(1, 0, 2, 1),
			"is excluded"},
		{invariant.Recorder_Assertions(recorder, "enum-member").Enum_Int(3, 1, 2),
			"not a member"},
	}
	for _, failure := range failures {
		message := panic_text(failure.Builder.Ensure)
		if !strings.Contains(message, failure.Message) {
			t.Fatalf("panic = %q, want %q", message, failure.Message)
		}
	}
}

// Test_Assertions_Registration_Resolves_Arithmetic protects exact static preset expansion.
func Test_Assertions_Registration_Resolves_Arithmetic(t *testing.T) {
	_, output, code := registered_fixture(`package fixture
const Unit = 1
const Minimum = -(Unit << 3)
const Maximum = int(((Unit + 3) * 4) / 2)
func check(value int) {
	invariant.Assertions("arithmetic").Range_Int(value, Minimum, Maximum).Ensure()
}
`)
	if code != -1 {
		t.Fatalf("registration exit=%d output=%q", code, output.String())
	}
}

// Test_Assertions_Registration_Expands_Directory_Globs protects multi-package discovery.
func Test_Assertions_Registration_Expands_Directory_Globs(t *testing.T) {
	file_system := fstest.MapFS{
		"tree/top.go": &fstest.MapFile{Data: []byte(`package top
func top(v bool) { invariant.Assertions("top").Sometimes(v, "axis").Ensure() }
`)},
		"tree/child/child.go": &fstest.MapFile{Data: []byte(`package child
func child(v bool) { invariant.Assertions("child").Sometimes(v, "axis").Ensure() }
`)},
		"tree/child/grand/grand.go": &fstest.MapFile{Data: []byte(`package grand
func grand(v bool) { invariant.Assertions("grand").Sometimes(v, "axis").Ensure() }
`)},
	}
	recorder := &invariant.Recorder{
		File_System: file_system, Output: &bytes.Buffer{},
		Exit: func(int) {}, Is_Test: true,
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/tree/**")
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "top", Ordinal: 0, Message: "axis",
	})
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "child", Ordinal: 0, Message: "axis",
	})
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "grand", Ordinal: 0, Message: "axis",
	})
	immediate := &invariant.Recorder{
		File_System: file_system, Output: &bytes.Buffer{},
		Exit: func(int) {}, Is_Test: true,
	}
	invariant.Recorder_Register_Packages_For_Analysis(immediate, "/tree/*")
	if event_count(&immediate.Events) != 1 {
		t.Fatalf("single-star events = %d, want 1", event_count(&immediate.Events))
	}
	chain_metadata(t, immediate, chain_metadata_key{
		Namespace: "child", Ordinal: 0, Message: "axis",
	})
}

func count_covered(recorder *invariant.Recorder) (count int) {
	recorder.Events.Range(func(_, value any) (continue_iteration bool) {
		metadata := value.(*invariant.Assertion_Metadata)
		covered := metadata.Frequency.Load() != 0
		if metadata.False_Frequency.Load() != 0 {
			covered = true
		}
		if covered {
			count++
		}
		return true
	})
	return count
}

func Benchmark_Assertions_Enforcement(benchmark *testing.B) {
	recorder := &invariant.Recorder{}
	for benchmark.Loop() {
		invariant.Recorder_Assertions(recorder, "benchmark").
			Sometimes(true, "axis").Range_Int(5, 0, 10).Ensure()
	}
}

func Benchmark_Assertions_Sometimes(benchmark *testing.B) {
	recorder := &invariant.Recorder{}
	for benchmark.Loop() {
		invariant.Recorder_Assertions(recorder, "benchmark").
			Sometimes(true, "axis").Ensure()
	}
}

func Benchmark_Assertions_Range(benchmark *testing.B) {
	recorder := &invariant.Recorder{}
	for benchmark.Loop() {
		invariant.Recorder_Assertions(recorder, "benchmark").
			Range_Int(5, 0, 10).Ensure()
	}
}

func Benchmark_Assertions_Range_Uint8(benchmark *testing.B) {
	recorder := &invariant.Recorder{}
	for benchmark.Loop() {
		invariant.Recorder_Assertions(recorder, "benchmark").
			Range_Uint8(5, 0, 10).Ensure()
	}
}

func Benchmark_Assertions_Range_Uint8_Domain(benchmark *testing.B) {
	recorder := &invariant.Recorder{}
	for benchmark.Loop() {
		invariant.Recorder_Assertions(recorder, "benchmark").
			Range_Uint8(5, 0, ^uint8(0)).Ensure()
	}
}

func Benchmark_Assertions_Range_Holes(benchmark *testing.B) {
	recorder := &invariant.Recorder{}
	for benchmark.Loop() {
		invariant.Recorder_Assertions(recorder, "benchmark").
			Range_Int(5, 0, 10, 2, 4, 6, 8).Ensure()
	}
}

func Benchmark_Assertions_Enum(benchmark *testing.B) {
	recorder := &invariant.Recorder{}
	for benchmark.Loop() {
		invariant.Recorder_Assertions(recorder, "benchmark").
			Enum_Int(5, 1, 3, 5, 7).Ensure()
	}
}

func Benchmark_Assertions_Ensure(benchmark *testing.B) {
	builder := invariant.Recorder_Assertions(&invariant.Recorder{}, "benchmark").
		Sometimes(true, "axis")
	for benchmark.Loop() {
		builder.Ensure()
	}
}

func Benchmark_Assertions_Recording(benchmark *testing.B) {
	recorder, _, code := registered_fixture(`package fixture
func check(value int) {
	invariant.Assertions("benchmark").
		Sometimes(value == 5, "axis").Range_Int(value, 0, 10).Ensure()
}
`)
	if code != -1 {
		benchmark.Fatalf("registration exit = %d", code)
	}
	for benchmark.Loop() {
		invariant.Recorder_Assertions(recorder, "benchmark").
			Sometimes(true, "axis").Range_Int(5, 0, 10).Ensure()
	}
}

// Nested helpers preserve the amplification that a solitary fluent chain
// cannot represent.
func benchmark_dense_leaf(recorder *invariant.Recorder, value int) {
	invariant.Recorder_Assertions(recorder, "dense-leaf").
		Sometimes(value&1 == 1, "odd").
		Range_Int(value, 0, 10, 2, 4, 6, 8).
		Enum_Int(value, 1, 3, 5, 7, 9).
		Ensure()
}

func benchmark_dense_branch(recorder *invariant.Recorder, value int) {
	invariant.Recorder_Assertions(recorder, "dense-branch").
		Range_Int(value, 0, 10).
		Sometimes(value < 8, "below eight").
		Ensure()
	benchmark_dense_leaf(recorder, value)
}

func benchmark_dense_root(recorder *invariant.Recorder, value int) {
	invariant.Recorder_Assertions(recorder, "dense-root").
		Enum_Int(value, 1, 3, 5, 7, 9).
		Sometimes(value != 0, "nonzero").
		Ensure()
	benchmark_dense_branch(recorder, value)
	benchmark_dense_branch(recorder, value)
}

func Benchmark_Assertions_Dense(benchmark *testing.B) {
	recorder := &invariant.Recorder{}
	for benchmark.Loop() {
		benchmark_dense_root(recorder, 5)
	}
}
