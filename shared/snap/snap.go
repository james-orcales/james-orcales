// Package snap provides inline snapshot testing for Go tests.
//
// Snapshots capture expected output directly in test files, similar to Jest or Vitest.
// When tests fail, the package displays a Myers diff showing what changed.
// Snapshots can be updated individually with snap.Edit().
//
// This package is the pure library tier. All dependencies arrive as fields of
// Snapper. For an OS-bound default ready to drop into tests, import the sibling
// composition-tier package snap/default.
package snap

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	invariant "github.com/james-orcales/james-orcales/shared/invariant/default"
	"github.com/james-orcales/james-orcales/shared/myers"
)

// Keys the diff colors so readers can map - / + to red / green without
// consulting docs. Embedded in every Snapshot mismatch header.
const mismatch_legend = "\033[31mexpected\033[0m vs \033[32mactual\033[0m"

// Snapshot represents an expected output value captured at a specific source location.
// Snapshots are compared against actual test output to verify correctness.
type Snapshot struct {
	// Expected_Output is the captured expected value.
	Expected_Output string
	// File_Path is the absolute path of the source file holding the literal.
	File_Path string
	// Line is the 1-based source line of the snapshot literal.
	Line int
	// Should_Edit requests rewriting the source literal with the actual output.
	Should_Edit bool
	// Snapper is the bound Snapper providing I/O and edit state.
	Snapper *Snapper
}

// File_Edit tracks a snapshot edit operation for adjusting line numbers in subsequent edits.
type File_Edit struct {
	// Line is the source line the edit occurred on.
	Line int
	// Delta is the change in line count the edit introduced.
	Delta int
}

// Snapper holds injectable dependencies and per-instance edit state.
// Construct a Snapper directly for in-process tests that need to redirect I/O
// away from disk, or import snap_default for an OS-bound default.
type Snapper struct {
	// File_System reads source files. Paths from Get_Caller are absolute OS paths;
	// lookups strip the leading "/" before calling fs.ReadFile.
	File_System fs.FS
	// W, when non-nil, receives modified source-file content produced by Edit
	// snapshots instead of writing back to disk via Write_File.
	W io.Writer
	// Output receives all diagnostic output: mismatch diffs and UPDATED notices.
	Output io.Writer
	// Write_File writes content to a file path.
	Write_File func(path string, data []byte, perm fs.FileMode) (err error)
	// Get_Caller returns the frame information for the caller at the given skip depth.
	Get_Caller func(skip int) (frame_information Frame_Information, err error)
	// Stdout is reset by Run before calling function, then read after.
	Stdout *bytes.Buffer
	// Stderr is reset by Run before calling function, then read after. function is
	// expected to close over the Snapper and write to Snapper.Stdout/Stderr.
	Stderr *bytes.Buffer
	// Edits records per-file line deltas accumulated by edit-mode snapshots.
	Edits map[string][]File_Edit
	// Edits_Mu guards Edits.
	Edits_Mu sync.Mutex
}

// Frame_Information contains caller frame information.
type Frame_Information struct {
	// File is the caller's source file path.
	File string
	// Line is the caller's 1-based source line.
	Line int
}

// Snapper_Init_At creates a snapshot bound to s with the call-site location
// captured via s.Get_Caller(skip). Composition-tier wrappers call this
// directly so they can choose a skip count matching their own call depth.
// Snapper_Init / Snapper_Edit are convenience wrappers around this.
func Snapper_Init_At(s *Snapper, skip int, data string, should_edit bool) (snapshot Snapshot) {
	frame_information, _ := s.Get_Caller(skip)
	return Snapshot{
		Expected_Output: data,
		File_Path:       frame_information.File,
		Line:            frame_information.Line,
		Should_Edit:     should_edit,
		Snapper:         s,
	}
}

// Snapper_Init creates a snapshot bound to s with expected output captured at the call site.
func Snapper_Init(s *Snapper, data string) (snapshot Snapshot) {
	return Snapper_Init_At(s, 2, data, false)
}

// Snapper_Edit creates an edit-mode snapshot bound to s.
func Snapper_Edit(s *Snapper, data string) (snapshot Snapshot) {
	return Snapper_Init_At(s, 2, data, true)
}

// New_Snapshot_Input is the input for New_Snapshot.
type New_Snapshot_Input struct {
	// Snapper is the Snapper the snapshot binds to.
	Snapper *Snapper
	// File_Path is the absolute path of the source file.
	File_Path string
	// Line is the 1-based source line of the snapshot literal.
	Line int
	// Expected is the expected output value.
	Expected string
	// Should_Edit requests rewriting the source literal with the actual output.
	Should_Edit bool
}

// New_Snapshot builds a Snapshot with explicit location and content.
// Useful when testing snap itself where the runtime.Caller location would not
// match the in-memory FS keys.
func New_Snapshot(input *New_Snapshot_Input) (snapshot Snapshot) {
	return Snapshot{
		Expected_Output: input.Expected,
		File_Path:       input.File_Path,
		Line:            input.Line,
		Should_Edit:     input.Should_Edit,
		Snapper:         input.Snapper,
	}
}

func expect_fail_mismatch(t *testing.T) {
	t.Helper()
	t.Fatal("Snapshot mismatch")
}

func expect_panic_fail_no_panic(t *testing.T, expected string) {
	t.Helper()
	t.Fatalf("Expected panic but none occurred. Expected: %s", expected)
}

// Panics anew when the recovered value does not match the snapshot.
func expect_panic_mismatch(recovered any, snapshot Snapshot) {
	actual_string := fmt.Sprint(recovered)
	if Snapshot_Is_Equal(snapshot, actual_string) {
		return
	}
	panic(fmt.Sprintf("Expected a different panic. Got: %s", actual_string))
}

// Expect compares the actual output against the snapshot's expected output.
// If they don't match, the test fails and displays a Myers diff of the changes.
func Expect(t *testing.T, snapshot Snapshot, actual any) (got string) {
	t.Helper()
	actual_string := fmt.Sprint(actual)
	if !Snapshot_Is_Equal(snapshot, actual_string) {
		expect_fail_mismatch(t)
	}
	return actual_string
}

// Expect_Panic verifies that the callback function panics with a message matching the snapshot.
// If the panic message doesn't match or no panic occurs, the test fails.
func Expect_Panic(t *testing.T, snapshot Snapshot, callback func()) {
	t.Helper()
	did_panic := false
	defer func() {
		if r := recover(); r != nil {
			did_panic = true
			expect_panic_mismatch(r, snapshot)
		}
	}()
	callback()
	if !did_panic {
		expect_panic_fail_no_panic(t, snapshot.Expected_Output)
	}
}

// Entry represents a test case with input and expected output snapshot.
// T is the input type for the test case.
type Entry[T any] struct {
	// Name is the subtest name.
	Name string
	// Input is the test-case input passed to the batch function.
	Input T
	// Snapshot is the expected-output snapshot for this case.
	Snapshot Snapshot
}

// Batch_Expect runs multiple test cases as subtests, each with snapshot validation.
// The function callback transforms each input into output for snapshot comparison.
func Batch_Expect[T any](t *testing.T, function func(T) (result any), entries []Entry[T]) {
	t.Helper()
	for _, e := range entries {
		t.Run(e.Name, func(st *testing.T) {
			st.Helper()
			result := function(e.Input)
			Expect(st, e.Snapshot, result)
		})
	}
}

// Batch_Expect_Panic runs multiple panic test cases as subtests with a shared callback.
// The function callback is expected to panic for each input.
func Batch_Expect_Panic[T any](t *testing.T, function func(T), entries []Entry[T]) {
	t.Helper()
	for _, e := range entries {
		t.Run(e.Name, func(st *testing.T) {
			st.Helper()
			Expect_Panic(st, e.Snapshot, func() {
				function(e.Input)
			})
		})
	}
}

// Should_Edit branch of Snapshot_Is_Equal. Called with s.Edits_Mu held.
// Updates the source file, records the delta, and returns true.
func snapper_is_equal_edit(
	s *Snapper, snapshot Snapshot, actual string, is_equal bool,
) (updated bool) {
	compile_time_line := snapshot.Line
	if edits, ok := s.Edits[snapshot.File_Path]; ok {
		offset := 0
		for _, edit := range edits {
			if edit.Line < compile_time_line {
				offset += edit.Delta
			}
		}
		snapshot.Line += offset
	}

	content, err := fs.ReadFile(s.File_System, strings.TrimPrefix(snapshot.File_Path, "/"))
	if err != nil {
		panic(fmt.Sprintf("Update snapshot | can't read file: %s\n", err))
	}

	span := snapper_locate_edit(s, snapshot, content)
	if !span.Found {
		return false
	}

	search := "snap.Edit(`"
	replace := "snap.Init(`"
	// Equal lengths keep the byte math below (Open+1-len(search)) aligned.
	invariant.Dot_Product(invariant.Always(len(search) == len(replace)))
	new_content := &bytes.Buffer{}
	new_content.Grow(len(content))
	new_content.Write(content[:span.Open+1-len(search)])
	new_content.WriteString(replace)
	new_content.WriteString(actual)
	new_content.Write(content[span.Close:])

	if s.W != nil {
		if _, write_err := s.W.Write(new_content.Bytes()); write_err != nil {
			panic(write_err)
		}
	} else {
		write_err := s.Write_File(snapshot.File_Path, new_content.Bytes(), 0o664)
		if write_err != nil {
			panic(write_err)
		}
	}

	if !is_equal {
		delta := strings.Count(actual, "\n") - strings.Count(snapshot.Expected_Output, "\n")
		if _, ok := s.Edits[snapshot.File_Path]; !ok {
			s.Edits[snapshot.File_Path] = make([]File_Edit, 0)
		}
		s.Edits[snapshot.File_Path] = append(
			s.Edits[snapshot.File_Path],
			File_Edit{Line: compile_time_line, Delta: delta},
		)
	}

	fmt.Fprintf(s.Output, "UPDATED SNAPSHOT %s:%d\n", snapshot.File_Path, snapshot.Line)
	return true
}

// Byte range of one source line, half-open as [Start, End).
type snapper_line_bounds struct {
	// Start is the byte offset of the first character on the line.
	Start int
	// End is the byte offset of the line's terminating newline.
	End int
}

// Locates the byte range of the 1-based line in content.
func snapper_find_line(content []byte, line int) (bounds snapper_line_bounds) {
	line_count := 1
	bounds.Start, bounds.End = -1, -1
	for i, b := range content {
		if b == '\n' {
			line_count++
			if line_count < line {
				continue
			} else if line_count == line {
				bounds.Start = i + 1
			} else if line_count == line+1 {
				bounds.End = i
				break
			}
		}
	}
	return bounds
}

// Byte offsets of the backticks delimiting an edit-mode snapshot's raw string.
type snapper_edit_span struct {
	// Open is the offset of the opening backtick.
	Open int
	// Close is the offset of the closing backtick.
	Close int
	// Found reports whether a well-formed snap.Edit call was located.
	Found bool
}

// Finds the backtick span of the snap.Edit literal on the snapshot's line,
// printing a diagnostic and reporting Found=false when the call is malformed.
func snapper_locate_edit(s *Snapper, snapshot Snapshot, content []byte) (span snapper_edit_span) {
	bounds := snapper_find_line(content, snapshot.Line)
	// Sequential, not one Dot_Product: each guard must hold before the next line's
	// content[...] index is evaluated, or an eager out-of-range read would panic first.
	invariant.Dot_Product(invariant.Always(bounds.Start >= 0 && bounds.End >= 0))
	invariant.Dot_Product(invariant.Always(bounds.Start > 1))
	invariant.Dot_Product(invariant.Always(content[bounds.Start-1] == '\n'))
	invariant.Dot_Product(invariant.Always(content[bounds.End] == '\n'))

	line := string(content[bounds.Start:bounds.End])
	search := "snap.Edit(`"
	if strings.Count(line, search) == 0 {
		fmt.Fprintf(s.Output,
			"snap.Edit at %s:%d must use a backticked raw string (snap.Edit(`...`))\n",
			snapshot.File_Path, snapshot.Line,
		)
		return span
	}
	invariant.Dot_Product(invariant.Always(strings.Count(line, search) == 1))

	call_offset := strings.Index(line, search) + bounds.Start
	span.Open = call_offset + len(search) - 1
	span.Close = -1
	for i, b := range content[span.Open+1:] {
		if b == '`' {
			span.Close = i + span.Open + 1
			break
		}
	}
	invariant.Dot_Product(invariant.Always(span.Open >= 0))
	invariant.Dot_Product(invariant.Always(span.Close >= 0))
	invariant.Dot_Product(invariant.Always(span.Open < span.Close))
	span.Found = true
	return span
}

// Snapshot_Is_Equal compares actual output against the expected snapshot value.
// If snapshot editing is enabled (via Edit), it updates the source file to replace
// the old snapshot with the actual output and returns true.
// On mismatch without editing, it prints a Myers diff to s.Out and returns false.
func Snapshot_Is_Equal(snapshot Snapshot, actual string) (equal bool) {
	invariant.Dot_Product(invariant.Always(snapshot.Snapper != nil))
	s := snapshot.Snapper
	invariant.Dot_Product(invariant.Always(snapshot.Line > 0))
	invariant.Dot_Product(invariant.Always(strings.Count(snapshot.Expected_Output, "`") == 0))
	invariant.Dot_Product(invariant.Always(strings.Count(actual, "`") == 0))
	invariant.Dot_Product(invariant.Always(filepath.IsAbs(snapshot.File_Path)))

	is_equal := actual == snapshot.Expected_Output
	if snapshot.Should_Edit {
		s.Edits_Mu.Lock()
		defer s.Edits_Mu.Unlock()
		return snapper_is_equal_edit(s, snapshot, actual, is_equal)
	} else if !is_equal {
		d := myers.New(myers.New_Input{Old: snapshot.Expected_Output, New: actual})
		fmt.Fprintf(s.Output, "Snapshot mismatch %s:%d  (%s)\n",
			snapshot.File_Path, snapshot.Line, mismatch_legend)
		for line := range strings.SplitSeq(myers.Differ_Line_Diff(d), "\n") {
			if len(line) == 0 {
				continue
			}
			switch line[0] {
			case '+':
				fmt.Fprintln(s.Output, "\033[32m"+line+"\033[0m")
			case '-':
				fmt.Fprintln(s.Output, "\033[31m"+line+"\033[0m")
			default:
				fmt.Fprintln(s.Output, line)
			}
		}
	}
	return is_equal
}

// Run executes function, captures what function writes to s.Stdout and s.Stderr, and asserts
// the combined output against snapshot.
// function is expected to close over the Snapper and write to Snapper.Stdout/Stderr.
// Run resets Stdout and Stderr before calling function and reads them after.
func Run(t *testing.T, function func(), snapshot Snapshot) (output string, err string) {
	t.Helper()
	invariant.Dot_Product(invariant.Always(snapshot.Snapper != nil))
	s := snapshot.Snapper
	s.Stdout.Reset()
	s.Stderr.Reset()
	function()
	output = s.Stdout.String()
	err = s.Stderr.String()
	if output == "" {
		if err == "" {
			Expect(t, snapshot, "snap.Run: no output")
		} else {
			Expect(t, snapshot, fmt.Sprintf("\nSTDERR:\n%s\n", err))
		}
	} else if err == "" {
		Expect(t, snapshot, fmt.Sprintf("\nSTDOUT:\n%s\n", output))
	} else {
		Expect(t, snapshot, fmt.Sprintf("\nSTDOUT:\n%s\n\nSTDERR:\n%s\n", output, err))
	}
	return output, err
}
