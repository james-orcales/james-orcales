//go:build !invariant_disable_coverage && !prd && !prod && !production && !invariant_noop

package invariant_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"local/james-orcales/shared/invariant"
	"local/james-orcales/shared/testify"
)

// Fixture_Subject stands in for a bundle subject where the test drives the builder directly. A
// plan-free Recorder never reaches the chain-type lookup, so the type only has to compile.
type Fixture_Subject int

// Opens a chain over the fixture subject, so a preset table stays inside the line limit.
func fixture_assertions(
	recorder *invariant.Recorder, namespace invariant.Namespace,
) (builder invariant.Assertion_Builder) {
	return invariant.Recorder_Tree(recorder, Fixture_Subject(0), namespace)
}

// Test_Assertions_Defers_Coverage_Until_Ensure protects the chain's atomic commit boundary.
func Test_Assertions_Defers_Coverage_Until_Ensure(t *testing.T) {
	recorder, _, code := registered_fixture(
		bundle_fixture("check", ".Sometimes(value == 0, \"observed\")"))
	if code != -1 {
		t.Fatalf("registration exit = %d", code)
	}
	metadata := chain_metadata(t, recorder, fixture_chain_key("check", 0, "observed"))
	builder := fixture_assertions(recorder, "check").
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
	recorder, _, code := registered_fixture(bundle_fixture("check",
		".Sometimes(value == 0, \"same\").Sometimes(value == 1, \"same\")"))
	if code != -1 {
		t.Fatalf("registration exit = %d", code)
	}
	fixture_assertions(recorder, "check").
		Sometimes(true, "same").
		Sometimes(false, "same").
		Ensure()
	first := chain_metadata(t, recorder, fixture_chain_key("check", 0, "same"))
	second := chain_metadata(t, recorder, fixture_chain_key("check", 1, "same"))
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
type Fixture_Subject int
const Minimum = 0
const Maximum = 10
func Fixture_Subject_Invariants(value Fixture_Subject, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Range_Int(int(value), Minimum, Maximum).Ensure()
}
func check(value Fixture_Subject) { Fixture_Subject_Invariants(value, "bounded") }
`)
	if code != -1 {
		t.Fatalf("registration exit = %d", code)
	}
	builder := fixture_assertions(recorder, "bounded").
		Range_Int(11, 0, 10)
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
	builder := fixture_assertions(&invariant.Recorder{}, "mode").
		Enum_Int(3, 1, 2)
	message := panic_text(builder.Ensure)
	if !strings.Contains(message, "not a member") {
		t.Fatalf("panic = %q", message)
	}
}

// Test_Assertions_Has_Zero_Allocations protects the builder's register-value shape.
func Test_Assertions_Has_Zero_Allocations(t *testing.T) {
	recorder := &invariant.Recorder{}
	testify.Zero_Allocation(t, func() {
		fixture_assertions(recorder, "allocation").
			Sometimes(true, "axis").Range_Int(5, 0, 10).Ensure()
	})
	recording, _, code := registered_fixture(`package fixture
type Fixture_Subject int
func Fixture_Subject_Invariants(value Fixture_Subject, namespace invariant.Namespace) {
	invariant.Always(value >= 0, "allocation always")
	invariant.Tree(value, namespace).
		Sometimes(value == 5, "axis").
		Range_Int(int(value), 0, 10).
		Range_Holed_Int(int(value), 0, 10, 1, 2, 3, 4).
		Enum_Int(int(value), 5, 6).
		Enum_3_Int(int(value), 5, 6, 7).
		Enum_4_Int(int(value), 5, 6, 7, 8).
		Ensure()
}
func inline(value int) {
	invariant.Sometimes(value == 5, "allocation inline sometimes")
	invariant.Range(value, 0, 10, "allocation inline range")
	invariant.Range_Holed(value, 0, 10, 1, 2, 3, 4, "allocation inline holed range")
	invariant.Enum(value, 5, 6, "allocation inline enum")
}
func check(value Fixture_Subject) { Fixture_Subject_Invariants(value, "allocation") }
`)
	if code != -1 {
		t.Fatalf("registration exit = %d", code)
	}
	testify.Zero_Allocation(t, func() {
		invariant.Recorder_Always(recording, true, "allocation always")
		invariant.Recorder_Sometimes(recording, true, "allocation inline sometimes")
		invariant.Recorder_Range(recording, 5, 0, 10, "allocation inline range")
		invariant.Recorder_Range_Holed(
			recording, 5, 0, 10, 1, 2, 3, 4, "allocation inline holed range")
		invariant.Recorder_Enum(recording, 5, 5, 6, "allocation inline enum")
		fixture_assertions(recording, "allocation").
			Sometimes(true, "axis").
			Range_Int(5, 0, 10).
			Range_Holed_Int(5, 0, 10, 1, 2, 3, 4).
			Enum_Int(5, 5, 6).
			Enum_3_Int(5, 5, 6, 7).
			Enum_4_Int(5, 5, 6, 7, 8).
			Ensure()
	})
	testify.Zero_Allocation(t, func() {
		benchmark_dense_root(recorder, 5)
	})
}

// Test_Assertions_Typed_Presets_Register_Every_Integer_Width protects exact static resolution.
func Test_Assertions_Typed_Presets_Register_Every_Integer_Width(t *testing.T) {
	fixtures := []struct {
		Value_Type string
		Suffix     string
		Minimum    string
		Maximum    string
	}{
		{"int", "Int", "-9223372036854775808", "9223372036854775807"},
		{"int8", "Int8", "-128", "127"},
		{"int16", "Int16", "-32768", "32767"},
		{"int32", "Int32", "-2147483648", "2147483647"},
		{"int64", "Int64", "-9223372036854775808", "9223372036854775807"},
		{"uint", "Uint", "0", "18446744073709551615"},
		{"uint8", "Uint8", "0", "255"},
		{"uint16", "Uint16", "0", "65535"},
		{"uint32", "Uint32", "0", "4294967295"},
		{"uint64", "Uint64", "0", "18446744073709551615"},
	}
	for _, fixture := range fixtures {
		source := fmt.Sprintf(`package fixture
type Fixture_Subject %s
func Fixture_Subject_Invariants(value Fixture_Subject, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Range_%s(value, %s, %s).Ensure()
}
func check(value Fixture_Subject) { Fixture_Subject_Invariants(value, %q) }
`, fixture.Value_Type, fixture.Suffix, fixture.Minimum, fixture.Maximum, fixture.Suffix)
		_, output, code := registered_fixture(source)
		if code != -1 {
			t.Fatalf("Range_%s exit=%d output=%q",
				fixture.Suffix, code, output.String())
		}
		unsigned := strings.HasPrefix(fixture.Value_Type, "uint")
		holes := "-1, 0, 1, 1"
		if unsigned {
			holes = "1, 2, 2"
		}
		source = fmt.Sprintf(`package fixture
type Fixture_Subject %s
func Fixture_Subject_Invariants(value Fixture_Subject, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Range_Holed_%s(value, %s, %s, %s).Ensure()
}
func check(value Fixture_Subject) { Fixture_Subject_Invariants(value, %q) }
`, fixture.Value_Type, fixture.Suffix,
			fixture.Minimum, fixture.Maximum, holes,
			"Range_Holed_"+fixture.Suffix)
		_, output, code = registered_fixture(source)
		if code != -1 {
			t.Fatalf("Range_Holed_%s exit=%d output=%q",
				fixture.Suffix, code, output.String())
		}
		assert_enum_widths_register(t, fixture.Value_Type, fixture.Suffix)
	}
}

// Registers each Enum capacity for one integer width, keeping the width table's own loop short.
func assert_enum_widths_register(t *testing.T, value_type string, suffix string) {
	t.Helper()
	for capacity := 2; capacity <= 4; capacity++ {
		method := "Enum_" + suffix
		if capacity > 2 {
			method = fmt.Sprintf("Enum_%d_%s", capacity, suffix)
		}
		members := "0, 1"
		if capacity == 3 {
			members = "0, 1, 2"
		}
		if capacity == 4 {
			members = "0, 1, 2, 3"
		}
		source := fmt.Sprintf(`package fixture
type Fixture_Subject %s
func Fixture_Subject_Invariants(value Fixture_Subject, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).%s(value, %s).Ensure()
}
func check(value Fixture_Subject) { Fixture_Subject_Invariants(value, %q) }
`, value_type, method, members, method)
		_, output, code := registered_fixture(source)
		if code != -1 {
			t.Fatalf("%s exit=%d output=%q", method, code, output.String())
		}
	}
}

// Test_Assertions_Range_Families_Execute_Every_Integer_Width prevents wrapper drift.
func Test_Assertions_Range_Families_Execute_Every_Integer_Width(t *testing.T) {
	recorder := &invariant.Recorder{}
	builders := []invariant.Assertion_Builder{
		fixture_assertions(recorder, "int8").Range_Int8(1, 0, 2),
		fixture_assertions(recorder, "int16").Range_Int16(1, 0, 2),
		fixture_assertions(recorder, "int32").Range_Int32(1, 0, 2),
		fixture_assertions(recorder, "int64").Range_Int64(1, 0, 2),
		fixture_assertions(recorder, "uint").Range_Uint(1, 0, 2),
		fixture_assertions(recorder, "uint8").Range_Uint8(1, 0, 2),
		fixture_assertions(recorder, "uint16").Range_Uint16(1, 0, 2),
		fixture_assertions(recorder, "uint32").Range_Uint32(1, 0, 2),
		fixture_assertions(recorder, "uint64").Range_Uint64(1, 0, 2),
		fixture_assertions(recorder, "holed-int").
			Range_Holed_Int(3, 0, 9, 1, 2, 4, 4),
		fixture_assertions(recorder, "holed-int8").
			Range_Holed_Int8(3, 0, 9, 1, 2, 4, 4),
		fixture_assertions(recorder, "holed-int16").
			Range_Holed_Int16(3, 0, 9, 1, 2, 4, 4),
		fixture_assertions(recorder, "holed-int32").
			Range_Holed_Int32(3, 0, 9, 1, 2, 4, 4),
		fixture_assertions(recorder, "holed-int64").
			Range_Holed_Int64(3, 0, 9, 1, 2, 4, 4),
		fixture_assertions(recorder, "holed-uint").
			Range_Holed_Uint(3, 0, 9, 1, 2, 4),
		fixture_assertions(recorder, "holed-uint8").
			Range_Holed_Uint8(3, 0, 9, 1, 2, 4),
		fixture_assertions(recorder, "holed-uint16").
			Range_Holed_Uint16(3, 0, 9, 1, 2, 4),
		fixture_assertions(recorder, "holed-uint32").
			Range_Holed_Uint32(3, 0, 9, 1, 2, 4),
		fixture_assertions(recorder, "holed-uint64").
			Range_Holed_Uint64(3, 0, 9, 1, 2, 4),
	}
	for _, builder := range builders {
		builder.Ensure()
	}
}

// Test_Assertions_Enum_Families_Execute_Every_Integer_Width prevents capacity drift.
func Test_Assertions_Enum_Families_Execute_Every_Integer_Width(t *testing.T) {
	recorder := &invariant.Recorder{}
	builders := []invariant.Assertion_Builder{
		fixture_assertions(recorder, "enum-int8").Enum_Int8(1, 1, 2),
		fixture_assertions(recorder, "enum-int16").Enum_Int16(1, 1, 2),
		fixture_assertions(recorder, "enum-int32").Enum_Int32(1, 1, 2),
		fixture_assertions(recorder, "enum-int64").Enum_Int64(1, 1, 2),
		fixture_assertions(recorder, "enum-uint").Enum_Uint(1, 1, 2),
		fixture_assertions(recorder, "enum-uint8").Enum_Uint8(1, 1, 2),
		fixture_assertions(recorder, "enum-uint16").Enum_Uint16(1, 1, 2),
		fixture_assertions(recorder, "enum-uint32").Enum_Uint32(1, 1, 2),
		fixture_assertions(recorder, "enum-uint64").Enum_Uint64(1, 1, 2),
		fixture_assertions(recorder, "enum-3-int").Enum_3_Int(2, 1, 2, 3),
		fixture_assertions(recorder, "enum-3-int8").
			Enum_3_Int8(2, 1, 2, 3),
		fixture_assertions(recorder, "enum-3-int16").
			Enum_3_Int16(2, 1, 2, 3),
		fixture_assertions(recorder, "enum-3-int32").
			Enum_3_Int32(2, 1, 2, 3),
		fixture_assertions(recorder, "enum-3-int64").
			Enum_3_Int64(2, 1, 2, 3),
		fixture_assertions(recorder, "enum-3-uint").
			Enum_3_Uint(2, 1, 2, 3),
		fixture_assertions(recorder, "enum-3-uint8").
			Enum_3_Uint8(2, 1, 2, 3),
		fixture_assertions(recorder, "enum-3-uint16").
			Enum_3_Uint16(2, 1, 2, 3),
		fixture_assertions(recorder, "enum-3-uint32").
			Enum_3_Uint32(2, 1, 2, 3),
		fixture_assertions(recorder, "enum-3-uint64").
			Enum_3_Uint64(2, 1, 2, 3),
		fixture_assertions(recorder, "enum-4-int").
			Enum_4_Int(3, 1, 2, 3, 4),
		fixture_assertions(recorder, "enum-4-int8").
			Enum_4_Int8(3, 1, 2, 3, 4),
		fixture_assertions(recorder, "enum-4-int16").
			Enum_4_Int16(3, 1, 2, 3, 4),
		fixture_assertions(recorder, "enum-4-int32").
			Enum_4_Int32(3, 1, 2, 3, 4),
		fixture_assertions(recorder, "enum-4-int64").
			Enum_4_Int64(3, 1, 2, 3, 4),
		fixture_assertions(recorder, "enum-4-uint").
			Enum_4_Uint(3, 1, 2, 3, 4),
		fixture_assertions(recorder, "enum-4-uint8").
			Enum_4_Uint8(3, 1, 2, 3, 4),
		fixture_assertions(recorder, "enum-4-uint16").
			Enum_4_Uint16(3, 1, 2, 3, 4),
		fixture_assertions(recorder, "enum-4-uint32").
			Enum_4_Uint32(3, 1, 2, 3, 4),
		fixture_assertions(recorder, "enum-4-uint64").
			Enum_4_Uint64(3, 1, 2, 3, 4),
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
		{fixture_assertions(recorder, "lower").Range_Int(-1, 0, 2),
			"below min"},
		{fixture_assertions(recorder, "excluded").
			Range_Holed_Int(1, 0, 2, 1, 1, 1, 1),
			"is excluded"},
		{fixture_assertions(recorder, "enum-member").Enum_Int(3, 1, 2),
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
type Fixture_Subject int
const Unit = 1
const Minimum = -(Unit << 3)
const Maximum = int(((Unit + 3) * 4) / 2)
func Fixture_Subject_Invariants(value Fixture_Subject, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Range_Int(int(value), Minimum, Maximum).Ensure()
}
func check(value Fixture_Subject) { Fixture_Subject_Invariants(value, "arithmetic") }
`)
	if code != -1 {
		t.Fatalf("registration exit=%d output=%q", code, output.String())
	}
}

// Test_Assertions_Registration_Expands_Directory_Globs protects multi-package discovery.
func Test_Assertions_Registration_Expands_Directory_Globs(t *testing.T) {
	file_system := fstest.MapFS{
		"tree/top.go": &fstest.MapFile{Data: []byte(`package top
type Fixture_Subject int
func Fixture_Subject_Invariants(value Fixture_Subject, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "axis").Ensure()
}
func top(value Fixture_Subject) { Fixture_Subject_Invariants(value, "top") }
`)},
		"tree/child/child.go": &fstest.MapFile{Data: []byte(`package child
type Fixture_Subject int
func Fixture_Subject_Invariants(value Fixture_Subject, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "axis").Ensure()
}
func child(value Fixture_Subject) { Fixture_Subject_Invariants(value, "child") }
`)},
		"tree/child/grand/grand.go": &fstest.MapFile{Data: []byte(`package grand
type Fixture_Subject int
func Fixture_Subject_Invariants(value Fixture_Subject, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).Sometimes(value == 0, "axis").Ensure()
}
func grand(value Fixture_Subject) { Fixture_Subject_Invariants(value, "grand") }
`)},
	}
	recorder := &invariant.Recorder{
		File_System: file_system, Output: &bytes.Buffer{},
		Exit: func(int) {}, Is_Test: true,
	}
	invariant.Recorder_Register_Packages_For_Analysis(recorder, "/tree/**")
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "top", Package: "", Type: "Fixture_Subject", Ordinal: 0, Message: "axis",
	})
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "child", Package: "", Type: "Fixture_Subject",
		Ordinal: 0, Message: "axis",
	})
	chain_metadata(t, recorder, chain_metadata_key{
		Namespace: "grand", Package: "", Type: "Fixture_Subject",
		Ordinal: 0, Message: "axis",
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
		Namespace: "child", Package: "", Type: "Fixture_Subject",
		Ordinal: 0, Message: "axis",
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
		fixture_assertions(recorder, "benchmark").
			Sometimes(true, "axis").Range_Int(5, 0, 10).Ensure()
	}
}

func Benchmark_Assertions_Sometimes(benchmark *testing.B) {
	recorder := &invariant.Recorder{}
	for benchmark.Loop() {
		fixture_assertions(recorder, "benchmark").
			Sometimes(true, "axis").Ensure()
	}
}

func Benchmark_Assertions_Range(benchmark *testing.B) {
	recorder := &invariant.Recorder{}
	for benchmark.Loop() {
		fixture_assertions(recorder, "benchmark").
			Range_Int(5, 0, 10).Ensure()
	}
}

func Benchmark_Assertions_Range_Uint8(benchmark *testing.B) {
	recorder := &invariant.Recorder{}
	for benchmark.Loop() {
		fixture_assertions(recorder, "benchmark").
			Range_Uint8(5, 0, 10).Ensure()
	}
}

func Benchmark_Assertions_Range_Uint8_Domain(benchmark *testing.B) {
	recorder := &invariant.Recorder{}
	for benchmark.Loop() {
		fixture_assertions(recorder, "benchmark").
			Range_Uint8(5, 0, ^uint8(0)).Ensure()
	}
}

func Benchmark_Assertions_Range_Holes(benchmark *testing.B) {
	recorder := &invariant.Recorder{}
	for benchmark.Loop() {
		fixture_assertions(recorder, "benchmark").
			Range_Holed_Int(5, 0, 10, 2, 4, 6, 8).Ensure()
	}
}

func Benchmark_Assertions_Enum_4(benchmark *testing.B) {
	recorder := &invariant.Recorder{}
	for benchmark.Loop() {
		fixture_assertions(recorder, "benchmark").
			Enum_4_Int(5, 1, 3, 5, 7).Ensure()
	}
}

func Benchmark_Assertions_Ensure(benchmark *testing.B) {
	builder := fixture_assertions(&invariant.Recorder{}, "benchmark").
		Sometimes(true, "axis")
	for benchmark.Loop() {
		builder.Ensure()
	}
}

func Benchmark_Assertions_Recording(benchmark *testing.B) {
	recorder, _, code := registered_fixture(`package fixture
type Fixture_Subject int
func Fixture_Subject_Invariants(value Fixture_Subject, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(value == 5, "axis").Range_Int(int(value), 0, 10).Ensure()
}
func check(value Fixture_Subject) { Fixture_Subject_Invariants(value, "benchmark") }
`)
	if code != -1 {
		benchmark.Fatalf("registration exit = %d", code)
	}
	for benchmark.Loop() {
		fixture_assertions(recorder, "benchmark").
			Sometimes(true, "axis").Range_Int(5, 0, 10).Ensure()
	}
}

// Nested helpers preserve the amplification that a solitary fluent chain
// cannot represent.
func benchmark_dense_leaf(recorder *invariant.Recorder, value int) {
	fixture_assertions(recorder, "dense-leaf").
		Sometimes(value&1 == 1, "odd").
		Range_Holed_Int(value, 0, 10, 2, 4, 6, 8).
		Enum_4_Int(value, 1, 3, 5, 7).
		Ensure()
}

func benchmark_dense_branch(recorder *invariant.Recorder, value int) {
	fixture_assertions(recorder, "dense-branch").
		Range_Int(value, 0, 10).
		Sometimes(value < 8, "below eight").
		Ensure()
	benchmark_dense_leaf(recorder, value)
}

func benchmark_dense_root(recorder *invariant.Recorder, value int) {
	fixture_assertions(recorder, "dense-root").
		Enum_4_Int(value, 1, 3, 5, 7).
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
