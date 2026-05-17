// Package snap provides inline snapshot testing for Go tests.
//
// Snapshots capture expected output directly in test files, similar to Jest or Vitest.
// When tests fail, the package displays a Myers diff showing what changed. Snapshots
// can be updated individually with snap.Edit().
//
// Basic usage:
//
//	func TestFunction(t *testing.T) {
//	    result := DoSomething()
//	    snap.Init(`expected output`).Expect(t, result)
//	}
//
// Batch testing:
//
//	snap.BatchExpect(t, func(input string) any {
//	    return Process(input)
//	}, []snap.Entry[string]{
//	    {"case1", "input1", snap.Init(`output1`)},
//	    {"case2", "input2", snap.Init(`output2`)},
//	})
package snap

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/james-orcales/james-orcales/golang_snacks/invariant"
	"github.com/james-orcales/james-orcales/golang_snacks/sim"
	"github.com/james-orcales/james-orcales/golang_snacks/snap/myers"
)

// Snapshot represents an expected output value captured at a specific source location.
// Snapshots are compared against actual test output to verify correctness.
type Snapshot struct {
	ExpectedOutput string
	FilePath       string
	Line           int
	ShouldEdit     bool
}

// Init creates a snapshot with expected output captured at the call site.
// The snapshot remembers its source location for updating when tests fail.
//
// Usage:
//
//	snap.Init(`expected output`).Expect(t, actual)
//
// WARN: Brittle under go:generate - the source location is captured at runtime
// and may not work correctly with code generation tools.
func Init(data string) (snapshot Snapshot) {
	callers := [1]uintptr{}
	count := runtime.Callers(2, callers[:])
	frame, _ := runtime.CallersFrames(callers[:count]).Next()

	return Snapshot{
		ExpectedOutput: data,
		FilePath:       frame.File,
		Line:           frame.Line,
	}
}

// Edit creates a snapshot that will update itself with the actual output on the next test run.
// Use this temporarily to update a specific snapshot, then change it back to Init.
//
// Usage:
//
//	snap.Edit(`old output`).Expect(t, actual)  // Updates to actual on next run
//	// After update, change back to:
//	snap.Init(`new output`).Expect(t, actual)
func Edit(data string) (snapshot Snapshot) {
	callers := [1]uintptr{}
	count := runtime.Callers(2, callers[:])
	frame, _ := runtime.CallersFrames(callers[:count]).Next()

	return Snapshot{
		ExpectedOutput: data,
		FilePath:       frame.File,
		Line:           frame.Line,
		ShouldEdit:     true,
	}
}

// FileEdit tracks a snapshot edit operation for adjusting line numbers in subsequent edits.
type FileEdit struct {
	Line, Delta int
}

var filesEdited = make(map[string][]FileEdit)
var filesEditedMu = sync.Mutex{}

// Expect compares the actual output against the snapshot's expected output.
// If they don't match, the test fails and displays a Myers diff of the changes.
func (snapshot Snapshot) Expect(t *testing.T, actual any) (got string) {
	t.Helper()
	actualStr := fmt.Sprint(actual)
	if !snapshot.IsEqual(actualStr) {
		t.Fatal("Snapshot mismatch")
	}
	return actualStr
}

// ExpectPanic verifies that the callback function panics with a message matching the snapshot.
// If the panic message doesn't match or no panic occurs, the test fails.
func (snapshot Snapshot) ExpectPanic(t *testing.T, callback func()) {
	t.Helper()
	didPanic := false
	defer func() {
		if r := recover(); r != nil {
			didPanic = true
			actualStr := fmt.Sprint(r)
			if !snapshot.IsEqual(actualStr) {
				panic(fmt.Sprintf("Expected panic but a different one occurred. Expected: %s", actualStr))
			}
		}
	}()
	callback()
	if !didPanic {
		t.Fatalf("Expected panic but none occurred. Expected: %s", snapshot.ExpectedOutput)
	}
}

// Entry represents a test case with input and expected output snapshot.
// T is the input type for the test case.
type Entry[T any] struct {
	Name     string
	Input    T
	Snapshot Snapshot
}

// BatchExpect runs multiple test cases as subtests, each with snapshot validation.
// The fn callback transforms each input into output for snapshot comparison.
//
// Usage:
//
//	entries := []snap.Entry[[]string]{
//	    {"case1", []string{"arg1"}, snap.Init(`output1`)},
//	    {"case2", []string{"arg2"}, snap.Init(`output2`)},
//	}
//	snap.BatchExpect(t, func(args []string) any {
//	    return DoThing(args...)
//	}, entries)
func BatchExpect[T any](t *testing.T, fn func(T) any, entries []Entry[T]) {
	t.Helper()
	for _, e := range entries {
		t.Run(e.Name, func(st *testing.T) {
			st.Helper()
			result := fn(e.Input)
			e.Snapshot.Expect(st, result)
		})
	}
}

// BatchExpectPanic runs multiple panic test cases as subtests with a shared callback.
// The fn callback is expected to panic for each input.
//
// Usage:
//
//	entries := []snap.Entry[string]{
//	    {"nil input", nil, snap.Init(`panic message`)},
//	    {"empty input", "", snap.Init(`panic message`)},
//	}
//	snap.BatchExpectPanic(t, func(input string) {
//	    DoThing(input)  // expected to panic
//	}, entries)
func BatchExpectPanic[T any](t *testing.T, fn func(T), entries []Entry[T]) {
	t.Helper()
	for _, e := range entries {
		t.Run(e.Name, func(st *testing.T) {
			st.Helper()
			e.Snapshot.ExpectPanic(st, func() {
				fn(e.Input)
			})
		})
	}
}

// Run executes a test function with stdout and stderr buffers cleared before and after execution.
func Run(t *testing.T, fn func(), snap Snapshot) (out, err string) {
	t.Helper()
	sim.StdoutBuf.Reset()
	sim.StderrBuf.Reset()
	defer func() {
		sim.StdoutBuf.Reset()
		sim.StderrBuf.Reset()
	}()
	fn()
	out = sim.StdoutBuf.String()
	err = sim.StderrBuf.String()
	if out == "" && err == "" {
		snap.Expect(t, "snap.Run: no output")
	} else if out == "" {
		snap.Expect(t, fmt.Sprintf("\nSTDERR:\n%s\n", err))
	} else if err == "" {
		snap.Expect(t, fmt.Sprintf("\nSTDOUT:\n%s\n", out))
	} else {
		snap.Expect(t, fmt.Sprintf("\nSTDOUT:\n%s\n\nSTDERR:\n%s\n", out, err))
	}
	return out, err
}

// IsEqual compares actual output against the expected snapshot value.
// If snapshot editing is enabled (via Edit), this method updates the source file
// to replace the old snapshot with the actual output.
//
// The method tracks file edits to adjust line numbers for subsequent edits in the
// same test run.
//
// Returns true if the actual output matches the expected snapshot, or if editing is
// enabled (after updating the source file).
//
// - On snapshot mismatches, the diffs are relative to the expected output:
//   - Red (-): Expected output (what's in the snapshot)
//   - Green (+): Actual output (what the code produced)
func (snapshot Snapshot) IsEqual(actual string) (isEqual bool) {
	invariant.Ensure(snapshot.Line > 0, "Snapshot location is set")
	invariant.Ensure(strings.Count(snapshot.ExpectedOutput, "`") == 0, "Snapshot expected value does not contain backticks")
	invariant.Ensure(strings.Count(actual, "`") == 0, "Snapshot actual value does not contain backticks")
	invariant.Ensure(filepath.IsAbs(snapshot.FilePath), "Snapshot location is an absolute path")

	compileTimeLine := snapshot.Line
	isEqual = actual == snapshot.ExpectedOutput
	if snapshot.ShouldEdit {
		filesEditedMu.Lock()
		defer filesEditedMu.Unlock()

		if edits, ok := filesEdited[snapshot.FilePath]; ok {
			offset := 0
			for _, edit := range edits {
				if edit.Line < compileTimeLine {
					offset += edit.Delta
				}
			}
			snapshot.Line += offset
		}

		content, err := os.ReadFile(snapshot.FilePath)
		if err != nil {
			panic(fmt.Sprintf("Update snapshot | can't read file: %s\n", err))
		}

		lineCount := 1
		start, end := -1, -1
		for i, b := range content {
			if b == '\n' {
				lineCount++
				if lineCount < snapshot.Line {
					continue
				} else if lineCount == snapshot.Line {
					start = i + 1
				} else if lineCount == snapshot.Line+1 {
					end = i
					break
				}
			}
		}
		invariant.Ensure(start >= 0 && end >= 0, "Snapshot is found")
		invariant.Ensure(start > 1, "Go source have package declaration or comments in the first line")
		invariant.Ensure(content[start-1] == '\n', "Line starts after newline")
		invariant.Ensure(content[end] == '\n', "Line ends with newline")

		line := string(content[start:end])
		replace := "snap.Init(`"
		search := "snap.Init(`"
		if snapshot.ShouldEdit {
			search = "snap.Edit(`"
		}
		invariant.Ensure(len(search) == len(replace), "Find and replace strings are of equal length")
		invariant.Ensure(strings.Count(line, search) > 0, "Found snapshot in expected line")
		invariant.Ensure(strings.Count(line, search) == 1, "Only one snap call per line")

		snap_call_idx := strings.Index(line, search)
		invariant.Ensure(snap_call_idx >= 0, "Found snapshot call")
		snap_call_idx += start

		open, close := snap_call_idx+len(search)-1, -1
		for i, b := range content[open+1:] {
			if b == '`' {
				close = i + open + 1
				break
			}
		}
		invariant.Ensure(open >= 0, "Found open backtick")
		invariant.Ensure(close >= 0, "Found closed backtick")
		invariant.Ensure(open < close, "Open backtick comes before closed backtick")

		new_content := &bytes.Buffer{}
		new_content.Grow(len(content))
		new_content.Write(content[:open+1-len(search)])
		new_content.WriteString(replace)
		new_content.WriteString(actual)
		new_content.Write(content[close:])

		if writeErr := os.WriteFile(snapshot.FilePath, new_content.Bytes(), 0o664); writeErr != nil {
			panic(writeErr)
		}

		if !isEqual {
			delta := strings.Count(actual, "\n") - strings.Count(snapshot.ExpectedOutput, "\n")
			if _, ok := filesEdited[snapshot.FilePath]; !ok {
				filesEdited[snapshot.FilePath] = make([]FileEdit, 0)
			}
			filesEdited[snapshot.FilePath] = append(
				filesEdited[snapshot.FilePath],
				FileEdit{Line: compileTimeLine, Delta: delta},
			)
		}

		fmt.Printf("UPDATED SNAPSHOT %s:%d\n", snapshot.FilePath, snapshot.Line)
		return true
	} else if !isEqual {
		d := myers.New(myers.NewInput{Old: snapshot.ExpectedOutput, New: actual})
		fmt.Fprintf(os.Stderr, "Snapshot mismatch %s:%d\n", snapshot.FilePath, snapshot.Line)
		for line := range strings.SplitSeq(d.LineDiff(), "\n") {
			if len(line) == 0 {
				fmt.Println(line)
				continue
			}
			switch line[0] {
			case '+':
				fmt.Println("\033[32m" + line + "\033[0m")
			case '-':
				fmt.Println("\033[31m" + line + "\033[0m")
			default:
				fmt.Println(line)
			}
		}
	}
	return isEqual
}
