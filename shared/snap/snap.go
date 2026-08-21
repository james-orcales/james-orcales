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
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"unsafe"

	"local/james-orcales/shared/diff/myers"
	"local/james-orcales/shared/simulation/aver/default"
)

// Keys the diff colors so readers can map - / + to red / green without
// consulting docs. Embedded in every Snapshot mismatch header.
const MISMATCH_LEGEND = "\033[31mexpected\033[0m vs \033[32mactual\033[0m"

// Data is one byte sequence crossing an injected sink or file boundary.
type Data []byte

// Data_Size is one completed write size.
type Data_Size int

// Permission is the caller-owned filesystem mode word.
type Permission uint32

// Write sends bytes to explicit caller state.
type Write func(state unsafe.Pointer, data Data) (written Data_Size, err error)

// Read_File reads one source path through explicit caller state.
type Read_File func(state unsafe.Pointer, path string) (data Data, err error)

// Write_File replaces one source path through explicit caller state.
type Write_File func(
	state unsafe.Pointer, path string, data Data, permission Permission,
) (err error)

// Buffer owns captured test output.
type Buffer []byte

// Write satisfies the bounded writer seam used by fmt in test callbacks.
func (buffer *Buffer) Write(data []byte) (written int, err error) {
	*buffer = append(*buffer, data...)
	return len(data), nil
}

// String satisfies fmt.Stringer without exposing mutable storage.
func (buffer Buffer) String() (text string) { return string(buffer) }

// Buffer_Reset retains storage between captured runs.
func Buffer_Reset(buffer *Buffer) { *buffer = (*buffer)[:0] }

// Buffer_Data borrows captured bytes.
func Buffer_Data(buffer *Buffer) (data Data) { return Data(*buffer) }

// Buffer_Size reports captured bytes.
func Buffer_Size(buffer *Buffer) (size Data_Size) { return Data_Size(len(*buffer)) }

// Buffer_Write appends bytes to capture storage.
func Buffer_Write(buffer *Buffer, data Data) (written Data_Size, err error) {
	*buffer = append(*buffer, data...)
	return Data_Size(len(data)), nil
}

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
	// File_System_State belongs to Read_File.
	File_System_State unsafe.Pointer
	// Read_File reads snapshot source.
	Read_File Read_File
	// Writer_State belongs to Write; nil Write falls through to Write_File.
	Writer_State unsafe.Pointer
	// Write receives rewritten content when the caller captures edits.
	Write Write
	// Output_State belongs to Output_Write.
	Output_State unsafe.Pointer
	// Output_Write receives mismatch and update diagnostics.
	Output_Write Write
	// Write_File writes content to a file path.
	Write_File Write_File
	// Write_File_State belongs to Write_File.
	Write_File_State unsafe.Pointer
	// Get_Caller returns the frame information for the caller at the given skip depth.
	Get_Caller func(skip int) (frame_information Frame_Information, err error)
	// Stdout is reset by Run before calling function, then read after.
	Stdout *Buffer
	// Stderr is reset by Run before calling function, then read after. function is
	// expected to close over the Snapper and write to Snapper.Stdout/Stderr.
	Stderr *Buffer
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

func snapper_print(snapper *Snapper, format string, values ...any) {
	if snapper.Output_Write == nil {
		return
	}
	data := fmt.Appendf(nil, format, values...)
	snapper.Output_Write(snapper.Output_State, Data(data))
}

func string_trim_prefix(value string, prefix string) (trimmed string) {
	if len(prefix) > len(value) {
		return value
	}
	if value[:len(prefix)] == prefix {
		return value[len(prefix):]
	}
	return value
}

func string_index(value string, sought string) (index int) {
	if len(sought) == 0 {
		return 0
	}
	if len(sought) > len(value) {
		return -1
	}
	last_start := len(value) - len(sought)
	for start_index := 0; start_index <= last_start; start_index++ {
		matched := true
		value_index := start_index
		for sought_index := 0; sought_index < len(sought); sought_index++ {
			if value[value_index] != sought[sought_index] {
				matched = false
				break
			}
			value_index++
		}
		if matched {
			return start_index
		}
	}
	return -1
}

func string_count(value string, sought string) (count int) {
	if len(sought) == 0 {
		return len(value) + 1
	}
	for len(value) >= len(sought) {
		index := string_index(value, sought)
		if index < 0 {
			return count
		}
		count++
		value = value[index+len(sought):]
	}
	return count
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

	content, err := s.Read_File(
		s.File_System_State, string_trim_prefix(snapshot.File_Path, "/"),
	)
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
	aver.Always(len(search) == len(replace),
		"snap.Edit and snap.Init prefixes are equal length")
	new_content := make(Buffer, 0, len(content)+len(actual))
	new_content = append(new_content, content[:span.Open+1-len(search)]...)
	new_content = append(new_content, replace...)
	new_content = append(new_content, actual...)
	new_content = append(new_content, content[span.Close:]...)

	if s.Write != nil {
		if _, write_err := s.Write(s.Writer_State, Data(new_content)); write_err != nil {
			panic(write_err)
		}
	} else {
		write_err := s.Write_File(
			s.Write_File_State, snapshot.File_Path, Data(new_content), 0o664,
		)
		if write_err != nil {
			panic(write_err)
		}
	}

	if !is_equal {
		delta := string_count(actual, "\n") - string_count(snapshot.Expected_Output, "\n")
		if _, ok := s.Edits[snapshot.File_Path]; !ok {
			s.Edits[snapshot.File_Path] = make([]File_Edit, 0)
		}
		s.Edits[snapshot.File_Path] = append(
			s.Edits[snapshot.File_Path],
			File_Edit{Line: compile_time_line, Delta: delta},
		)
	}

	snapper_print(s, "UPDATED SNAPSHOT %s:%d\n", snapshot.File_Path, snapshot.Line)
	return true
}

// Byte range of one source line, half-open as [Start, End).
type Snapper_Line_Bounds struct {
	// Start is the byte offset of the first character on the line.
	Start int
	// End is the byte offset of the line's terminating newline.
	End int
}

// Locates the byte range of the 1-based line in content.
func snapper_find_line(content []byte, line int) (bounds Snapper_Line_Bounds) {
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
type Snapper_Edit_Span struct {
	// Open is the offset of the opening backtick.
	Open int
	// Close is the offset of the closing backtick.
	Close int
	// Found reports whether a well-formed snap.Edit call was located.
	Found bool
}

// Finds the backtick span of the snap.Edit literal on the snapshot's line,
// printing a diagnostic and reporting Found=false when the call is malformed.
func snapper_locate_edit(s *Snapper, snapshot Snapshot, content []byte) (span Snapper_Edit_Span) {
	bounds := snapper_find_line(content, snapshot.Line)
	// Sequential because each eager guard must hold before the next line's
	// content[...] index is evaluated, or an eager out-of-range read would panic first.
	aver.Always(bounds.Start >= 0 && bounds.End >= 0, "line bounds were located")
	aver.Always(bounds.Start > 1, "line is not the first")
	aver.Always(content[bounds.Start-1] == '\n', "byte before line start is a newline")
	aver.Always(content[bounds.End] == '\n', "line ends on a newline")

	line := string(content[bounds.Start:bounds.End])
	search := "snap.Edit(`"
	if string_count(line, search) == 0 {
		snapper_print(s,
			"snap.Edit at %s:%d must use a backticked raw string (snap.Edit(`...`))\n",
			snapshot.File_Path, snapshot.Line,
		)
		return span
	}
	aver.Always(string_count(line, search) == 1, "exactly one snap.Edit on the line")

	call_offset := string_index(line, search) + bounds.Start
	span.Open = call_offset + len(search) - 1
	span.Close = -1
	for i, b := range content[span.Open+1:] {
		if b == '`' {
			span.Close = i + span.Open + 1
			break
		}
	}
	aver.Always(span.Open >= 0, "opening backtick offset is non-negative")
	aver.Always(span.Close >= 0, "closing backtick offset is non-negative")
	aver.Always(span.Open < span.Close, "opening backtick precedes the closing backtick")
	span.Found = true
	return span
}

// Snapshot_Is_Equal compares actual output against the expected snapshot value.
// If snapshot editing is enabled (via Edit), it updates the source file to replace
// the old snapshot with the actual output and returns true.
// On mismatch without editing, it prints a Myers diff to s.Out and returns false.
func Snapshot_Is_Equal(snapshot Snapshot, actual string) (equal bool) {
	aver.Always(snapshot.Snapper != nil,
		"Snapshot_Is_Equal snapshot is bound to a Snapper")
	s := snapshot.Snapper
	aver.Always(snapshot.Line > 0, "snapshot line is 1-based")
	aver.Always(string_count(snapshot.Expected_Output, "`") == 0,
		"expected output has no backtick")
	aver.Always(string_count(actual, "`") == 0, "actual output has no backtick")
	aver.Always(filepath.IsAbs(snapshot.File_Path), "snapshot file path is absolute")

	is_equal := actual == snapshot.Expected_Output
	if snapshot.Should_Edit {
		s.Edits_Mu.Lock()
		defer s.Edits_Mu.Unlock()
		return snapper_is_equal_edit(s, snapshot, actual, is_equal)
	} else if !is_equal {
		snapper_print(s, "Snapshot mismatch %s:%d  (%s)\n",
			snapshot.File_Path, snapshot.Line, MISMATCH_LEGEND)
		if len(snapshot.Expected_Output) > myers.TEXT_SIZE_MAXIMUM {
			return false
		}
		if len(actual) > myers.TEXT_SIZE_MAXIMUM {
			return false
		}
		workspace := myers.Workspace{
			Old_Runes: make(myers.Old_Rune_Storage, len(snapshot.Expected_Output)+1),
			New_Runes: make(myers.New_Rune_Storage, len(actual)+1),
			Matrix: make(
				myers.Matrix_Storage,
				(len(snapshot.Expected_Output)+2)*(len(actual)+2),
			),
		}
		output := make(myers.Line_Output, myers.LINE_DIFF_SIZE_MAXIMUM)
		count, status := myers.Line_Diff_Into(myers.Line_Diff_Input{
			Output:    output,
			Workspace: &workspace,
			Old:       myers.Line_Old_Text_Unvalidated(snapshot.Expected_Output),
			New:       myers.Line_New_Text_Unvalidated(actual),
		})
		if status != myers.STATUS_OK {
			return false
		}
		snapper_print_lines(s, string(output[:count]))
	}
	return is_equal
}

func snapper_print_lines(snapper *Snapper, text string) {
	for len(text) > 0 {
		index := string_index(text, "\n")
		line := text
		if index >= 0 {
			line = text[:index]
			text = text[index+1:]
		} else {
			text = ""
		}
		if len(line) == 0 {
			continue
		}
		switch line[0] {
		case '+':
			snapper_print(snapper, "\033[32m%s\033[0m\n", line)
		case '-':
			snapper_print(snapper, "\033[31m%s\033[0m\n", line)
		default:
			snapper_print(snapper, "%s\n", line)
		}
	}
}

// Run executes function, captures what function writes to s.Stdout and s.Stderr, and asserts
// the combined output against snapshot.
// function is expected to close over the Snapper and write to Snapper.Stdout/Stderr.
// Run resets Stdout and Stderr before calling function and reads them after.
func Run(t *testing.T, function func(), snapshot Snapshot) (output string, err string) {
	t.Helper()
	aver.Always(snapshot.Snapper != nil, "Run snapshot is bound to a Snapper")
	s := snapshot.Snapper
	Buffer_Reset(s.Stdout)
	Buffer_Reset(s.Stderr)
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
