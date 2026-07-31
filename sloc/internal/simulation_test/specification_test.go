package simulation_test

import (
	"strings"
	"testing"
	"testing/fstest"
	"unicode/utf8"

	sloc "local/james-orcales/sloc/internal"
)

// The specification tests mirror SPECIFICATION.md: one leaf per heading, in heading
// order, each driving internal.Main through the shared scenario driver. They assert no
// invariant trips; the coverage recorder judges the boundaries the fuzz target reaches.

// Test_Witness_Tree drives a walked directory through classification and both
// renderers from one Main call.
func Test_Witness_Tree(t *testing.T) {
	drive(decode_scenario(build(base_scenario())))
	drive(decode_scenario(with(func(one *scenario) { one.Show_Files = true })))
	drive(decode_scenario(with(func(one *scenario) { one.Json = true })))
}

// Test_Witness_Languages counts one file per recognized extension and bare name, so
// every language constructor runs.
func Test_Witness_Languages(t *testing.T) {
	drive(decode_scenario(with(func(one *scenario) { one.Recipe = 2 })))
	drive(decode_scenario(with(func(one *scenario) {
		one.Recipe = 2
		one.Show_Files = true
	})))
}

// Test_Witness_Boundaries counts a line at the scan window, a tree at the file bound,
// and a table whose every column reaches its width.
func Test_Witness_Boundaries(t *testing.T) {
	drive(decode_scenario(with(func(one *scenario) {
		one.Recipe = 3
		one.Line_Bytes = 4096
	})))
	drive(decode_scenario(with(func(one *scenario) {
		one.Recipe = 31
	})))
}

// Test_Witness_Boundary_Model protects the reachability argument behind substituting the
// expensive byte derivation: every modeled file and aggregate must still fit production.
func Test_Witness_Boundary_Model(t *testing.T) {
	fixture := disk_line_bound(base_scenario())
	if len(fixture.Classifications) == 0 {
		t.Fatal("line-bound fixture has no modeled classifications")
	}
	assert_line_bound_aggregate(t, fixture)
	for _, name := range []sloc.Classified_Path{"Zpast00000.go", "Zpast00001.go"} {
		if fixture.Classifications[name] != (sloc.File_Partition{Code: 1}) {
			t.Errorf("past-bound path %q is not modeled exactly", name)
		}
	}
	assert_line_bound_equivalence(t)
	// An empty source is the only classification the model can derive without a
	// stored result, so it separately witnesses the model's empty cardinality.
	empty := disk_fixture{
		Disk: fstest.MapFS{
			"empty.go": file(""),
		},
		Classifications: map[sloc.Classified_Path]sloc.File_Partition{},
	}
	output := strings.Builder{}
	drive_main(&drive_main_input{
		Scenario: base_scenario(),
		Fixture:  empty,
		Output:   &output,
	})
}

// Test_Witness_Source_Bound_Model keeps the real bounded read while preventing its known
// repeated source from turning a byte-bound witness into another scanner stress test.
func Test_Witness_Source_Bound_Model(t *testing.T) {
	fixture := disk_oversized()
	exact := fixture.Disk["exact.go"]
	if len(exact.Data) != sloc.SOURCE_BYTES_MAX {
		t.Errorf("exact source bytes = %d, want %d", len(exact.Data), sloc.SOURCE_BYTES_MAX)
	}
	huge := fixture.Disk["huge.go"]
	if len(huge.Data) != sloc.SOURCE_BYTES_MAX+1 {
		t.Errorf("oversized source bytes = %d, want %d",
			len(huge.Data), sloc.SOURCE_BYTES_MAX+1)
	}
	want := sloc.File_Partition{Code: sloc.SOURCE_BYTES_MAX / 2}
	if fixture.Classifications["exact.go"] != want {
		t.Errorf("exact classification = %+v, want %+v",
			fixture.Classifications["exact.go"], want)
	}
	if _, modeled := fixture.Classifications["huge.go"]; modeled {
		t.Error("oversized source must be rejected before classification")
	}
}

// Test_Witness_Classifier_Input_Bounds keeps the largest accepted path and source on
// the production byte-classifier route. One wide line makes the source bound cheap to
// scan while preserving the exact input width.
func Test_Witness_Classifier_Input_Bounds(t *testing.T) {
	name := strings.Repeat("p", sloc.FILE_PATH_BYTES_MAX-len(".go")) + ".go"
	fixture := disk_fixture{Disk: fstest.MapFS{
		name: file(strings.Repeat("a", sloc.SOURCE_BYTES_MAX)),
	}}
	output := strings.Builder{}
	drive_main(&drive_main_input{
		Scenario: base_scenario(),
		Fixture:  fixture,
		Output:   &output,
	})
}

// Test_Witness_Model_Path_Minimum keeps the shortest recognized path on the modeled
// classifier route. Hidden selection is required because the path is the bare .c.
func Test_Witness_Model_Path_Minimum(t *testing.T) {
	fixture := disk_fixture{
		Disk: fstest.MapFS{".c": file("")},
		Classifications: map[sloc.Classified_Path]sloc.File_Partition{
			".c": {},
		},
	}
	one := base_scenario()
	one.Hidden = true
	output := strings.Builder{}
	drive_main(&drive_main_input{Scenario: one, Fixture: fixture, Output: &output})
}

// Test_Witness_Wide_Model prevents the performance model from changing which line kind a
// wide-table fixture contributes while allowing it to avoid scanning repeated source bytes.
func Test_Witness_Wide_Model(t *testing.T) {
	fixture := disk_wide_table(WIDE_KIND_COMMENT)
	if len(fixture.Classifications) == 0 {
		t.Fatal("wide-table fixture has no modeled classifications")
	}
	for name, counts := range fixture.Classifications {
		entry, found := fixture.Disk[string(name)]
		if !found {
			t.Errorf("modeled path %q is absent from disk", name)
			continue
		}
		if len(entry.Data) > len(wide_line(WIDE_KIND_COMMENT)) {
			t.Errorf("modeled path %q stores repeated source", name)
		}
		want := wide_counts(&wide_counts_input{
			Kind: WIDE_KIND_COMMENT, Line_Count: wide_lines(WIDE_KIND_COMMENT),
		})
		if counts != want {
			t.Errorf("modeled path %q counts = %+v, want %+v", name, counts, want)
		}
	}
	equivalence := disk_fixture{
		Disk:            fstest.MapFS{},
		Classifications: map[sloc.Classified_Path]sloc.File_Partition{},
	}
	for kind := range 6 {
		text := wide_line(kind)
		suffix := wide_suffix(kind)
		const LINE_COUNT = 3
		name := "scaled" + decimal(kind) + suffix
		equivalence.Disk[name] = file(strings.Repeat(text, LINE_COUNT))
		equivalence.Classifications[sloc.Classified_Path(name)] = wide_counts(
			&wide_counts_input{Kind: kind, Line_Count: LINE_COUNT})
	}
	assert_model_equivalence(t, &model_equivalence_input{
		Name: "wide-table", Fixture: equivalence,
	})
}

// Test_Witness_Table_Width_Model makes the compound width witness explicit so no large
// column can silently fall back to a different recipe and leave the sum one cell short.
func Test_Witness_Table_Width_Model(t *testing.T) {
	one := base_scenario()
	one.Recipe = 31
	one.Show_Files = true
	fixture := disk_line_bound(one)
	assert_table_width_inputs(t, fixture)
	output := &line_width_writer{}
	drive_main(&drive_main_input{Scenario: one, Fixture: fixture, Output: output})
	if output.Maximum != sloc.TABLE_WIDTH_MAX {
		t.Errorf("table width = %d, want %d", output.Maximum, sloc.TABLE_WIDTH_MAX)
	}
}

// Test_Witness_Shared_Properties verifies that the distinct source and test roles use
// the same file, line, and dropped-count properties.
func Test_Witness_Shared_Properties(t *testing.T) {
	fixture := disk_test_partitions()
	one := base_scenario()
	drive_main(&drive_main_input{Scenario: one, Fixture: fixture, Output: &strings.Builder{}})
	one.Json = true
	drive_main(&drive_main_input{Scenario: one, Fixture: fixture, Output: &strings.Builder{}})
}

// Test_Witness_Failure drives the stat failure, the read failure, and the trees whose
// files are dropped as binary, oversized, or unrecognized.
func Test_Witness_Failure(t *testing.T) {
	drive(decode_scenario(with(func(one *scenario) { one.Stat_Error = true })))
	drive(decode_scenario(with(func(one *scenario) {
		one.Explicit = true
		one.Read_Error = true
		one.Root_Bytes = 7
	})))
	drive(decode_scenario(with(func(one *scenario) { one.Recipe = 7 })))
	drive(decode_scenario(with(func(one *scenario) { one.Recipe = 11 })))
	drive(decode_scenario(with(func(one *scenario) { one.Recipe = 12 })))
}

// The model remains valid only while each modeled path is reachable by the production
// source and aggregate bounds.
func assert_line_bound_aggregate(t *testing.T, fixture disk_fixture) {
	t.Helper()
	columns := sloc.Counts{}
	language_lines := map[sloc.Language_Name]sloc.Line_Count{}
	for name, counts := range fixture.Classifications {
		entry, found := fixture.Disk[string(name)]
		if !found {
			t.Errorf("modeled path %q is absent from disk", name)
			continue
		}
		if len(entry.Data) > 2 {
			t.Errorf("modeled path %q stores %d bytes, want at most 2",
				name, len(entry.Data))
		}
		lines := sloc.Line_Count(counts.Code) +
			sloc.Line_Count(counts.Comment) + sloc.Line_Count(counts.Blank)
		if lines > sloc.SOURCE_BYTES_MAX {
			t.Errorf("modeled path %q has %d unreachable lines", name, lines)
		}
		if strings.HasPrefix(string(name), "Zpast") {
			continue
		}
		language, recognized := modeled_language(name)
		if !recognized {
			t.Errorf("modeled path %q is unrecognized", name)
			continue
		}
		language_lines[language] += lines
		columns.Code += counts.Code
		columns.Comment += counts.Comment
		columns.Blank += counts.Blank
	}
	if len(language_lines) != 3 {
		t.Errorf("modeled languages = %d, want 3", len(language_lines))
	}
	for language, lines := range language_lines {
		if lines != sloc.LINE_COUNT_MAX {
			t.Errorf("%s lines = %d, want %d", language, lines, sloc.LINE_COUNT_MAX)
		}
	}
	if columns.Code != sloc.LINE_COUNT_MAX {
		t.Errorf("code = %d, want %d", columns.Code, sloc.LINE_COUNT_MAX)
	}
	if columns.Comment != sloc.LINE_COUNT_MAX {
		t.Errorf("comment = %d, want %d", columns.Comment, sloc.LINE_COUNT_MAX)
	}
	if columns.Blank != sloc.LINE_COUNT_MAX {
		t.Errorf("blank = %d, want %d", columns.Blank, sloc.LINE_COUNT_MAX)
	}
}

// Representative production scans anchor the arithmetic that scales each line kind to
// its production bound.
func assert_line_bound_equivalence(t *testing.T) {
	t.Helper()
	fixture := disk_fixture{
		Disk:            fstest.MapFS{},
		Classifications: map[sloc.Classified_Path]sloc.File_Partition{},
	}
	for kind := range 3 {
		text, suffix := line_bound_line(kind)
		const LINE_COUNT = 3
		name := "scaled" + decimal(kind) + suffix
		fixture.Disk[name] = file(strings.Repeat(text, LINE_COUNT))
		fixture.Classifications[sloc.Classified_Path(name)] = line_bound_counts(
			&line_bound_counts_input{
				Kind: kind, Line_Count: LINE_COUNT,
			})
	}
	assert_model_equivalence(t, &model_equivalence_input{
		Name: "line-bound", Fixture: fixture,
	})
}

type model_equivalence_input struct {
	Name    string
	Fixture disk_fixture
}

// Two Main calls prove the model is observationally identical to the byte classifier
// on representative sources without granting the simulation a second entry point.
func assert_model_equivalence(t *testing.T, input *model_equivalence_input) {
	t.Helper()
	real_fixture := input.Fixture
	real_fixture.Classifications = nil
	real_output := strings.Builder{}
	drive_main(&drive_main_input{
		Scenario: base_scenario(), Fixture: real_fixture, Output: &real_output,
	})
	modeled_output := strings.Builder{}
	drive_main(&drive_main_input{
		Scenario: base_scenario(), Fixture: input.Fixture, Output: &modeled_output,
	})
	if real_output.String() != modeled_output.String() {
		t.Errorf("%s model differs from byte classification\nreal:\n%s\nmodeled:\n%s",
			input.Name, real_output.String(), modeled_output.String())
	}
}

// Only the modeled line-bound languages are admitted here; a new extension has to
// state its model identity rather than inheriting one accidentally.
func modeled_language(name sloc.Classified_Path) (language sloc.Language_Name, found bool) {
	switch {
	case strings.HasSuffix(string(name), ".go"):
		return "Go", true
	case strings.HasSuffix(string(name), ".py"):
		return "Python", true
	case strings.HasSuffix(string(name), ".rs"):
		return "Rust", true
	}
	return "", false
}

// The combined bound is easier to diagnose from its independent production inputs than
// from one thousands-character rendered row.
func assert_table_width_inputs(t *testing.T, fixture disk_fixture) {
	t.Helper()
	maximum_path := 0
	go_files := 0
	go_tests := 0
	var go_code sloc.Line_Count
	var python_comments sloc.Line_Count
	var rust_blanks sloc.Line_Count
	for name := range fixture.Disk {
		maximum_path = max(maximum_path, len(name))
	}
	for name, counts := range fixture.Classifications {
		if strings.HasPrefix(string(name), "Zpast") {
			continue
		}
		language, recognized := modeled_language(name)
		if !recognized {
			continue
		}
		switch language {
		case "Go":
			go_files++
			go_code += sloc.Line_Count(counts.Code)
			if strings.HasSuffix(string(name), "_test.go") {
				go_tests++
			}
		case "Python":
			python_comments += sloc.Line_Count(counts.Comment)
		case "Rust":
			rust_blanks += sloc.Line_Count(counts.Blank)
		}
	}
	if maximum_path != sloc.FILE_PATH_BYTES_MAX {
		t.Errorf("maximum path = %d, want %d", maximum_path, sloc.FILE_PATH_BYTES_MAX)
	}
	if go_files < LINE_BOUND_FILE_WIDTH_COUNT {
		t.Errorf("Go files = %d, want at least %d", go_files, LINE_BOUND_FILE_WIDTH_COUNT)
	}
	if go_tests == 0 {
		t.Error("Go group has no test partition")
	}
	if go_code != sloc.LINE_COUNT_MAX {
		t.Errorf("Go code = %d, want %d", go_code, sloc.LINE_COUNT_MAX)
	}
	if python_comments != sloc.LINE_COUNT_MAX {
		t.Errorf("Python comments = %d, want %d", python_comments, sloc.LINE_COUNT_MAX)
	}
	if rust_blanks != sloc.LINE_COUNT_MAX {
		t.Errorf("Rust blanks = %d, want %d", rust_blanks, sloc.LINE_COUNT_MAX)
	}
}

// A line_width_writer discards bytes after observing them because retaining a maximum-width
// table would turn the width assertion itself into the dominant allocation.
type line_width_writer struct {
	Maximum int
}

// Write measures rendered characters rather than UTF-8 bytes because the table rule uses
// three-byte box-drawing characters for one column each.
func (writer *line_width_writer) Write(data []byte) (count int, err error) {
	line := data
	if len(line) > 0 {
		if line[len(line)-1] == '\n' {
			line = line[:len(line)-1]
		}
	}
	writer.Maximum = max(writer.Maximum, utf8.RuneCount(line))
	return len(data), nil
}
