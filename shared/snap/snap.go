// Package snap provides inline snapshot testing for Go tests.
//
// Snapshots capture expected output directly in test files, similar to Jest or Vitest.
// When tests fail, the package displays a Myers diff showing what changed.
// Snapshots can be updated individually with snap.Edit().
//
// This package is the pure library tier. All dependencies arrive as fields of
// Snapper. For an OS-bound default ready to drop into tests, import the sibling
// composition-tier package snap/default.
//
// Every byte the package stages lives inside the caller's Snapper: capture
// buffers, the diagnostic line, the rewritten source, the Run report, and the
// edit table. Nothing here allocates at runtime.
package snap

import (
	"fmt"
	"sync"
	"testing"
	"unsafe"

	"local/james-orcales/shared/diff/myers"
	"local/james-orcales/shared/path"
	"local/james-orcales/shared/sim/aver/default"
)

// Keys the diff colors so readers can map - / + to red / green without
// consulting docs. Embedded in every Snapshot mismatch header.
const MISMATCH_LEGEND = "\033[31mexpected\033[0m vs \033[32mactual\033[0m"

// BUFFER_SIZE_MAXIMUM bounds one captured stream. Test output past it is a
// test defect, so Buffer_Write panics rather than growing.
const BUFFER_SIZE_MAXIMUM = 1 << 16

// PRINT_SIZE_MAXIMUM bounds one diagnostic line: a source location plus its
// message, or one colored diff line. A diff line is at most a myers line diff.
const PRINT_SIZE_MAXIMUM = myers.LINE_DIFF_SIZE_MAXIMUM

// SOURCE_SIZE_MAXIMUM bounds one source file read for an edit. Snapshot test
// files in this repository stay well under 1 MiB.
const SOURCE_SIZE_MAXIMUM = 1 << 20

// REWRITE_SIZE_MAXIMUM bounds one rewritten source file: the source plus the
// actual output, which Snapshot_Is_Equal already bounds by myers.
const REWRITE_SIZE_MAXIMUM = SOURCE_SIZE_MAXIMUM + myers.TEXT_SIZE_MAXIMUM

// EDIT_COUNT_MAXIMUM bounds edits recorded by one Snapper. Edit mode is a
// manual one-shot operation, so hundreds per test binary is generous.
const EDIT_COUNT_MAXIMUM = 256

// REPORT_DECORATION_SIZE counts the framing bytes Run adds around both
// streams: "\nSTDOUT:\n" (9), "\n\nSTDERR:\n" (10), and the closing "\n" (1).
const REPORT_DECORATION_SIZE = 20

// REPORT_SIZE_MAXIMUM bounds Run's combined report of both captured streams.
const REPORT_SIZE_MAXIMUM = BUFFER_SIZE_MAXIMUM*2 + REPORT_DECORATION_SIZE

// LINE_DIGIT_COUNT_MAXIMUM bounds the decimal digits of a source line number.
// A 64-bit int has 19 digits.
const LINE_DIGIT_COUNT_MAXIMUM = 19

// DECIMAL_BASE is the radix line numbers print in.
const DECIMAL_BASE = 10

// Data is one byte sequence crossing an injected sink or file boundary.
type Data []byte

// Data_Size is one completed write size.
type Data_Size int

// Permission is the caller-owned filesystem mode word.
type Permission uint32

// Write sends bytes to explicit caller state.
type Write func(state unsafe.Pointer, data Data) (written Data_Size, err error)

// Read_File reads one absolute source path through explicit caller state into
// destination and returns the filled prefix. A file past destination is an error.
type Read_File func(state unsafe.Pointer, path string, destination Data) (data Data, err error)

// Write_File replaces one source path through explicit caller state.
type Write_File func(
	state unsafe.Pointer, path string, data Data, permission Permission,
) (err error)

// Buffer owns captured test output over root-provided storage.
type Buffer struct {
	// Bytes is root-provided storage of BUFFER_SIZE_MAXIMUM; only the first
	// Size bytes are live.
	Bytes Data
	// Size counts bytes captured since the last Buffer_Reset.
	Size Data_Size
}

// Buffer_String borrows captured text. The view aliases Bytes, so it is valid
// only until the next Buffer_Reset or Buffer_Write; borrowing avoids copying.
func Buffer_String(buffer *Buffer) (text string) {
	if buffer.Size == 0 {
		return ""
	}
	return unsafe.String(&buffer.Bytes[0], int(buffer.Size))
}

// Buffer_Reset retains storage between captured runs.
func Buffer_Reset(buffer *Buffer) { buffer.Size = 0 }

// Buffer_Data borrows captured bytes.
func Buffer_Data(buffer *Buffer) (data Data) { return Data(buffer.Bytes[:buffer.Size]) }

// Buffer_Size reports captured bytes.
func Buffer_Size(buffer *Buffer) (size Data_Size) { return buffer.Size }

// Buffer_Write copies bytes into capture storage. Overflow panics: the bound
// is the contract, not a hint.
func Buffer_Write(buffer *Buffer, data Data) (written Data_Size, err error) {
	aver.Always(int(buffer.Size)+len(data) <= len(buffer.Bytes),
		"captured output fits the root-provided buffer")
	copied := copy(buffer.Bytes[buffer.Size:], data)
	buffer.Size += Data_Size(copied)
	return Data_Size(copied), nil
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

// File_Edit records one snapshot rewrite so later edits in the same file can
// shift their target line.
type File_Edit struct {
	// File_Path is the absolute path of the rewritten source file.
	File_Path string
	// Line is the compile-time source line the edit occurred on.
	Line int
	// Delta is the change in line count the edit introduced.
	Delta int
}

// Edit_Storage is root-provided storage for EDIT_COUNT_MAXIMUM edits.
type Edit_Storage []File_Edit

// Storage is every byte a Snapper writes into, provided by the root that
// owns it. Slices rather than arrays so the root chooses where they live.
type Storage struct {
	// Stdout backs Snapper.Stdout; BUFFER_SIZE_MAXIMUM bytes.
	Stdout Data
	// Stderr backs Snapper.Stderr; BUFFER_SIZE_MAXIMUM bytes.
	Stderr Data
	// Print backs Snapper.Print; PRINT_SIZE_MAXIMUM bytes.
	Print Data
	// Source backs Snapper.Source; SOURCE_SIZE_MAXIMUM bytes.
	Source Data
	// Rewrite backs Snapper.Rewrite; REWRITE_SIZE_MAXIMUM bytes.
	Rewrite Data
	// Report backs Snapper.Report; REPORT_SIZE_MAXIMUM bytes.
	Report Data
	// Edits backs Snapper.Edits; EDIT_COUNT_MAXIMUM entries.
	Edits Edit_Storage
}

// Snapper_Bind_Storage hands root storage to s. Exact sizes are the
// contract: every bound in this package is a constant, so a short slice
// would only move the panic to a later, less readable site.
func Snapper_Bind_Storage(s *Snapper, storage Storage) {
	aver.Always(len(storage.Stdout) == BUFFER_SIZE_MAXIMUM, "Stdout storage is exact")
	aver.Always(len(storage.Stderr) == BUFFER_SIZE_MAXIMUM, "Stderr storage is exact")
	aver.Always(len(storage.Print) == PRINT_SIZE_MAXIMUM, "Print storage is exact")
	aver.Always(len(storage.Source) == SOURCE_SIZE_MAXIMUM, "Source storage is exact")
	aver.Always(len(storage.Rewrite) == REWRITE_SIZE_MAXIMUM, "Rewrite storage is exact")
	aver.Always(len(storage.Report) == REPORT_SIZE_MAXIMUM, "Report storage is exact")
	aver.Always(len(storage.Edits) == EDIT_COUNT_MAXIMUM, "Edits storage is exact")
	s.Stdout = Buffer{Bytes: storage.Stdout}
	s.Stderr = Buffer{Bytes: storage.Stderr}
	s.Print = storage.Print
	s.Source = storage.Source
	s.Rewrite = storage.Rewrite
	s.Report = storage.Report
	s.Edits = storage.Edits
	s.Edit_Count = 0
}

// Snapper holds injectable dependencies, per-instance edit state, and every
// staging buffer the library writes into. Construct one directly for
// in-process tests that need to redirect I/O away from disk, or import
// snap/default for an OS-bound default. Storage arrives through
// Snapper_Bind_Storage; an unbound Snapper panics on its first write.
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
	// Get_Caller returns the frame skip frames above Get_Caller itself: 0 is
	// Get_Caller's direct caller.
	Get_Caller func(skip int) (frame_information Frame_Information, err error)
	// Stdout is reset by Run before calling function, then read after.
	Stdout Buffer
	// Stderr is reset by Run before calling function, then read after.
	Stderr Buffer
	// Edits records rewrites in occurrence order; only the first Edit_Count are live.
	Edits Edit_Storage
	// Edit_Count counts live Edits.
	Edit_Count int
	// Print stages one diagnostic line before Output_Write. Snapper-owned so
	// the slice handed to the injected writer needs no fresh storage.
	Print Data
	// Source receives one source file from Read_File before an edit.
	Source Data
	// Rewrite stages one rewritten source file before Write or Write_File.
	Rewrite Data
	// Report stages Run's combined stream report before comparison.
	Report Data
	// Edits_Mu guards Edits and every staging buffer.
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
	frame_information, err := s.Get_Caller(skip)
	aver.Always(err == nil, "caller frame lookup succeeds")
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
	// Frames above Get_Caller: 0 Snapper_Init_At, 1 Snapper_Init, 2 the call site.
	return Snapper_Init_At(s, 2, data, false)
}

// Snapper_Edit creates an edit-mode snapshot bound to s.
func Snapper_Edit(s *Snapper, data string) (snapshot Snapshot) {
	// Same depth as Snapper_Init.
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

// The mismatch header already went to Output_Write, so the test log carries
// only the failure mark; t.Fatal would box its message.
func expect_fail_mismatch(t *testing.T) {
	t.Helper()
	t.FailNow()
}

func expect_panic_fail_no_panic(t *testing.T, snapshot Snapshot) {
	t.Helper()
	s := snapshot.Snapper
	offset := print_text(s, 0, "Expected panic but none occurred. Expected: ")
	offset = print_text(s, offset, snapshot.Expected_Output)
	offset = print_text(s, offset, "\n")
	print_flush(s, offset)
	t.FailNow()
}

// Stages text into s.Print at offset and returns the new end.
func print_text(s *Snapper, offset int, text string) (end int) {
	aver.Always(offset+len(text) <= len(s.Print), "diagnostic line fits Print")
	return offset + copy(s.Print[offset:], text)
}

// Stages bytes into s.Print at offset and returns the new end.
func print_bytes(s *Snapper, offset int, data []byte) (end int) {
	aver.Always(offset+len(data) <= len(s.Print), "diagnostic line fits Print")
	return offset + copy(s.Print[offset:], data)
}

// Stages a decimal line number into s.Print at offset and returns the new end.
func print_line_number(s *Snapper, offset int, line int) (end int) {
	aver.Always(line > 0, "printed line number is 1-based")
	var digits [LINE_DIGIT_COUNT_MAXIMUM]byte
	digit_size := 0
	for quotient := line; quotient > 0; quotient /= DECIMAL_BASE {
		digits[digit_size] = byte('0' + quotient%DECIMAL_BASE)
		digit_size++
	}
	aver.Always(offset+digit_size <= len(s.Print), "diagnostic line fits Print")
	end = offset
	for digit_index := digit_size - 1; digit_index >= 0; digit_index-- {
		s.Print[end] = digits[digit_index]
		end++
	}
	return end
}

// Hands the staged line to Output_Write. A nil Output_Write drops diagnostics.
func print_flush(s *Snapper, size int) {
	if s.Output_Write == nil {
		return
	}
	s.Output_Write(s.Output_State, Data(s.Print[:size]))
}

// Prints "<prefix><file>:<line><suffix>".
func print_location(s *Snapper, prefix string, snapshot Snapshot, suffix string) {
	offset := print_text(s, 0, prefix)
	offset = print_text(s, offset, snapshot.File_Path)
	offset = print_text(s, offset, ":")
	offset = print_line_number(s, offset, snapshot.Line)
	offset = print_text(s, offset, suffix)
	print_flush(s, offset)
}

// Prints "<prefix><line><suffix>" for one diff line.
func print_diff_line(s *Snapper, prefix string, line []byte, suffix string) {
	offset := print_text(s, 0, prefix)
	offset = print_bytes(s, offset, line)
	offset = print_text(s, offset, suffix)
	print_flush(s, offset)
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

// The assertion preserves panic behavior when the recovered value does not match.
func expect_panic_match(recovered any, snapshot Snapshot) {
	actual_string := fmt.Sprint(recovered)
	aver.Always(Snapshot_Is_Equal(snapshot, actual_string),
		"recovered panic output matches its snapshot")
}

// A string actual needs no formatting; fmt.Sprint only for other kinds.
func actual_text(actual any) (text string) {
	if actual_string, is_string := actual.(string); is_string {
		return actual_string
	}
	return fmt.Sprint(actual)
}

// Expect compares the actual output against the snapshot's expected output.
// If they don't match, the test fails and displays a Myers diff of the changes.
func Expect(t *testing.T, snapshot Snapshot, actual any) (got string) {
	t.Helper()
	actual_string := actual_text(actual)
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
	defer expect_panic_recover(snapshot, &did_panic)
	callback()
	if !did_panic {
		expect_panic_fail_no_panic(t, snapshot)
	}
}

// Recover must run directly inside deferred function.
func expect_panic_recover(snapshot Snapshot, did_panic *bool) {
	if r := recover(); r != nil {
		*did_panic = true
		expect_panic_match(r, snapshot)
	}
}

// Entry represents a string test case with expected output snapshot.
type Entry struct {
	// Name is the entry name printed when the entry mismatches.
	Name string
	// Input is the test-case input passed to the batch function.
	Input string
	// Snapshot is the expected-output snapshot for this case.
	Snapshot Snapshot
}

// Batch_Expect checks every entry in order. The function callback transforms
// each input into output for snapshot comparison. Entries run inline rather
// than as t.Run subtests: a subtest closure escapes to the heap.
func Batch_Expect(t *testing.T, function func(string) (result any), entries []Entry) {
	t.Helper()
	for entry_index := 0; entry_index < len(entries); entry_index++ {
		entry := &entries[entry_index]
		result := function(entry.Input)
		if !Snapshot_Is_Equal(entry.Snapshot, actual_text(result)) {
			batch_fail_entry(t, entry)
		}
	}
}

// Batch_Expect_Panic checks every entry's panic in order with a shared
// callback. The function callback is expected to panic for each input.
func Batch_Expect_Panic(t *testing.T, function func(string), entries []Entry) {
	t.Helper()
	for entry_index := 0; entry_index < len(entries); entry_index++ {
		entry := &entries[entry_index]
		Expect_Panic(t, entry.Snapshot, func() {
			function(entry.Input)
		})
	}
}

// Names the failed entry on Output before failing, since inline entries have
// no subtest name in the test log.
func batch_fail_entry(t *testing.T, entry *Entry) {
	t.Helper()
	s := entry.Snapshot.Snapper
	offset := print_text(s, 0, "BATCH ENTRY ")
	offset = print_text(s, offset, entry.Name)
	offset = print_text(s, offset, "\n")
	print_flush(s, offset)
	expect_fail_mismatch(t)
}

// Sums the line deltas of earlier edits in the same file that sit above line.
func snapper_edit_offset(s *Snapper, file_path string, line int) (offset int) {
	for edit_index := 0; edit_index < s.Edit_Count; edit_index++ {
		edit := &s.Edits[edit_index]
		if edit.File_Path != file_path {
			continue
		}
		if edit.Line < line {
			offset += edit.Delta
		}
	}
	return offset
}

// Stages head + replace + actual + tail in s.Rewrite and returns the staged size.
func snapper_stage_rewrite(
	s *Snapper, content []byte, span Snapper_Edit_Span, actual string,
) (size int) {
	search := "snap.Edit(`"
	replace := "snap.Init(`"
	// Equal lengths keep the byte math below (Open+1-len(search)) aligned.
	aver.Always(len(search) == len(replace),
		"snap.Edit and snap.Init prefixes are equal length")
	head := content[:span.Open+1-len(search)]
	tail := content[span.Close:]
	aver.Always(len(head)+len(replace)+len(actual)+len(tail) <= len(s.Rewrite),
		"rewritten source fits Rewrite")
	size = copy(s.Rewrite[:], head)
	size += copy(s.Rewrite[size:], replace)
	size += copy(s.Rewrite[size:], actual)
	size += copy(s.Rewrite[size:], tail)
	return size
}

// Should_Edit branch of Snapshot_Is_Equal. Called with s.Edits_Mu held.
// Updates the source file, records the delta, and returns true.
func snapper_is_equal_edit(
	s *Snapper, snapshot Snapshot, actual string, is_equal bool,
) (updated bool) {
	compile_time_line := snapshot.Line
	snapshot.Line += snapper_edit_offset(s, snapshot.File_Path, compile_time_line)

	content, err := s.Read_File(s.File_System_State, snapshot.File_Path, s.Source)
	aver.Always(err == nil, "snapshot source file read succeeds")

	span := snapper_locate_edit(s, snapshot, content)
	if !span.Found {
		return false
	}

	size := snapper_stage_rewrite(s, content, span, actual)
	new_content := Data(s.Rewrite[:size])
	if s.Write != nil {
		_, write_err := s.Write(s.Writer_State, new_content)
		aver.Always(write_err == nil, "injected snapshot write succeeds")
	} else {
		write_err := s.Write_File(
			s.Write_File_State, snapshot.File_Path, new_content, 0o664,
		)
		aver.Always(write_err == nil, "snapshot source file write succeeds")
	}

	if !is_equal {
		delta := string_count(actual, "\n") - string_count(snapshot.Expected_Output, "\n")
		aver.Always(s.Edit_Count < len(s.Edits), "edit table has room")
		s.Edits[s.Edit_Count] = File_Edit{
			File_Path: snapshot.File_Path, Line: compile_time_line, Delta: delta,
		}
		s.Edit_Count++
	}

	print_location(s, "UPDATED SNAPSHOT ", snapshot, "\n")
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

	// Borrowed view: content outlives every use of line below, and a copy would allocate.
	line := unsafe.String(&content[bounds.Start], bounds.End-bounds.Start)
	search := "snap.Edit(`"
	if string_count(line, search) == 0 {
		print_location(s, "snap.Edit at ", snapshot,
			" must use a backticked raw string (snap.Edit(`...`))\n")
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
	aver.Always(bool(path.Is_Absolute(path.Text(snapshot.File_Path))),
		"snapshot file path is absolute")

	is_equal := actual == snapshot.Expected_Output
	if snapshot.Should_Edit {
		s.Edits_Mu.Lock()
		defer s.Edits_Mu.Unlock()
		return snapper_is_equal_edit(s, snapshot, actual, is_equal)
	} else if !is_equal {
		print_location(s, "Snapshot mismatch ", snapshot, "  ("+MISMATCH_LEGEND+")\n")
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
		snapper_print_lines(s, output[:count])
	}
	return is_equal
}

func snapper_print_lines(snapper *Snapper, text []byte) {
	for len(text) > 0 {
		index := bytes_index_newline(text)
		line := text
		if index >= 0 {
			line = text[:index]
			text = text[index+1:]
		} else {
			text = text[len(text):]
		}
		if len(line) == 0 {
			continue
		}
		switch line[0] {
		case '+':
			print_diff_line(snapper, "\033[32m", line, "\033[0m\n")
		case '-':
			print_diff_line(snapper, "\033[31m", line, "\033[0m\n")
		default:
			print_diff_line(snapper, "", line, "\n")
		}
	}
}

func bytes_index_newline(text []byte) (index int) {
	for byte_index := 0; byte_index < len(text); byte_index++ {
		if text[byte_index] == '\n' {
			return byte_index
		}
	}
	return -1
}

// Stages "<prefix><data>" into s.Report at offset and returns the new end.
func report_text(s *Snapper, offset int, text string) (end int) {
	aver.Always(offset+len(text) <= len(s.Report), "run report fits Report")
	return offset + copy(s.Report[offset:], text)
}

// Frames the captured streams the way readers expect them in a snapshot
// literal. The result borrows s.Report and is valid until the next Run.
func run_report(s *Snapper, output string, err string) (report string) {
	if output == "" {
		if err == "" {
			return "snap.Run: no output"
		}
		offset := report_text(s, 0, "\nSTDERR:\n")
		offset = report_text(s, offset, err)
		offset = report_text(s, offset, "\n")
		return unsafe.String(&s.Report[0], offset)
	}
	offset := report_text(s, 0, "\nSTDOUT:\n")
	offset = report_text(s, offset, output)
	offset = report_text(s, offset, "\n")
	if err != "" {
		offset = report_text(s, offset, "\nSTDERR:\n")
		offset = report_text(s, offset, err)
		offset = report_text(s, offset, "\n")
	}
	return unsafe.String(&s.Report[0], offset)
}

// Run executes function, captures what function writes through Buffer_Write, and asserts the
// combined output against snapshot.
// Run resets Stdout and Stderr before calling function and reads them after.
// The returned strings borrow the Snapper's buffers until the next Run.
func Run(t *testing.T, function func(), snapshot Snapshot) (output string, err string) {
	t.Helper()
	aver.Always(snapshot.Snapper != nil, "Run snapshot is bound to a Snapper")
	s := snapshot.Snapper
	Buffer_Reset(&s.Stdout)
	Buffer_Reset(&s.Stderr)
	function()
	output = Buffer_String(&s.Stdout)
	err = Buffer_String(&s.Stderr)
	if !Snapshot_Is_Equal(snapshot, run_report(s, output, err)) {
		expect_fail_mismatch(t)
	}
	return output, err
}
