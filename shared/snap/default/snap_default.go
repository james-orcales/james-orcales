// Package snap is the composition-tier sibling of the snap library. It wires
// the snap library to the real OS (filesystem, stderr, runtime.Callers) and
// re-exports the surface so callers can write:
//
//	import snap "github.com/james-orcales/james-orcales/shared/snap/default"
//
// and use snap.Init / snap.Edit / snap.Expect / … as if no split had happened.
package snap

import (
	"bytes"
	"os"
	"runtime"
	"testing"

	"github.com/james-orcales/james-orcales/shared/snap"
)

// Snapper re-exports the library's Snapper so callers need only this import.
type Snapper = snap.Snapper

// Snapshot re-exports the library's Snapshot.
type Snapshot = snap.Snapshot

// File_Edit re-exports the library's File_Edit.
type File_Edit = snap.File_Edit

// Frame_Information re-exports the library's Frame_Information.
type Frame_Information = snap.Frame_Information

// New_Snapshot_Input re-exports the library's New_Snapshot_Input.
type New_Snapshot_Input = snap.New_Snapshot_Input

// Entry re-exports the library's Entry.
type Entry[T any] = snap.Entry[T]

// Default is the OS-bound Snapper used by the package-level Init / Edit / …
// convenience functions. Tests that need to redirect I/O construct their own
// Snapper directly via the snap package; Default exists for the common case.
var Default = Init_Default_Snapper()

// Init_Default_Snapper builds a Snapper wired to the host OS: the local
// filesystem, os.Stderr, os.WriteFile, and runtime.Callers. This is the one
// place in the snap tree where ambient binding is permitted.
func Init_Default_Snapper() (snapper *snap.Snapper) {
	return &snap.Snapper{
		File_System: os.DirFS("/"),
		Output:      os.Stderr,
		Write_File:  os.WriteFile,
		Get_Caller: func(skip int) (frame_information snap.Frame_Information, err error) {
			callers := [1]uintptr{}
			count := runtime.Callers(skip, callers[:])
			frame, _ := runtime.CallersFrames(callers[:count]).Next()
			return snap.Frame_Information{File: frame.File, Line: frame.Line}, nil
		},
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Edits:  make(map[string][]snap.File_Edit),
	}
}

// Init creates a snapshot bound to Default with the call-site location
// captured via runtime.Callers. WARN: brittle under go:generate — the
// source location is captured at runtime.
func Init(data string) (snapshot snap.Snapshot) {
	// Runtime.Callers frames from inside Get_Caller: 0 Callers, 1 Get_Caller,
	// 2 Snapper_Init_At, 3 this wrapper, 4 the test call site.
	return snap.Snapper_Init_At(Default, 4, data, false)
}

// Edit creates a snapshot bound to Default that will rewrite the source line
// with the actual output on the next test run. Use this temporarily to
// update a specific snapshot, then change it back to Init.
func Edit(data string) (snapshot snap.Snapshot) {
	// Same frame depth as Init: the test call site is 4 frames above Callers.
	return snap.Snapper_Init_At(Default, 4, data, true)
}

// New_Snapshot forwards to snap.New_Snapshot.
func New_Snapshot(input *snap.New_Snapshot_Input) (snapshot snap.Snapshot) {
	return snap.New_Snapshot(input)
}

// Snapshot_Is_Equal forwards to snap.Snapshot_Is_Equal.
func Snapshot_Is_Equal(snapshot snap.Snapshot, actual string) (equal bool) {
	return snap.Snapshot_Is_Equal(snapshot, actual)
}

// Expect forwards to snap.Expect.
func Expect(t *testing.T, snapshot snap.Snapshot, actual any) (got string) {
	t.Helper()
	return snap.Expect(t, snapshot, actual)
}

// Expect_Panic forwards to snap.Expect_Panic.
func Expect_Panic(t *testing.T, snapshot snap.Snapshot, callback func()) {
	t.Helper()
	snap.Expect_Panic(t, snapshot, callback)
}

// Run forwards to snap.Run.
func Run(t *testing.T, function func(), snapshot snap.Snapshot) (output string, err string) {
	t.Helper()
	return snap.Run(t, function, snapshot)
}

// Batch_Expect forwards to snap.Batch_Expect.
func Batch_Expect[T any](t *testing.T, function func(T) (result any), entries []Entry[T]) {
	t.Helper()
	snap.Batch_Expect(t, function, entries)
}

// Batch_Expect_Panic forwards to snap.Batch_Expect_Panic.
func Batch_Expect_Panic[T any](t *testing.T, function func(T), entries []Entry[T]) {
	t.Helper()
	snap.Batch_Expect_Panic(t, function, entries)
}

// Edits_For returns the recorded line-delta edits Default has accumulated for
// path. Exposed for the package-level use case where tests want to inspect
// edits without holding a *Snapper themselves.
func Edits_For(path string) (edits []File_Edit) {
	Default.Edits_Mu.Lock()
	defer Default.Edits_Mu.Unlock()
	return append([]File_Edit(nil), Default.Edits[path]...)
}
