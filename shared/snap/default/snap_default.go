// Package snap is the composition-tier sibling of the snap library. It wires
// the snap library to the real OS (filesystem, stderr, runtime.Callers) and
// re-exports the surface so callers can write:
//
//	import snap "local/james-orcales/shared/snap/default"
//
// and use snap.Init / snap.Edit / snap.Expect / … as if no split had happened.
package snap

import (
	"runtime"
	"syscall"
	"testing"
	"unsafe"

	"local/james-orcales/shared/snap"
)

// CALLER_FRAME_COUNT requests only the program counter that identifies the test site.
const CALLER_FRAME_COUNT = 1

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
type Entry = snap.Entry

// Default is the OS-bound Snapper used by the package-level Init / Edit / …
// convenience functions. Tests that need to redirect I/O construct their own
// Snapper directly via the snap package; Default exists for the common case.
var Default = Init_Default_Snapper()

// Init_Default_Snapper builds a Snapper wired to the host OS: the local
// filesystem, standard error, file writes, and runtime.Callers. This is the one
// place in the snap tree where ambient binding is permitted.
func Init_Default_Snapper() (snapper *snap.Snapper) {
	return &snap.Snapper{
		Read_File:    operating_read_file,
		Output_Write: standard_error_write,
		Write_File:   operating_write_file,
		Get_Caller:   operating_caller,
		Stdout:       &snap.Buffer{},
		Stderr:       &snap.Buffer{},
		Edits:        make(map[string][]snap.File_Edit),
	}
}

func operating_caller(skip int) (frame_information snap.Frame_Information, err error) {
	callers := [CALLER_FRAME_COUNT]uintptr{}
	count := runtime.Callers(skip, callers[:])
	frame, _ := runtime.CallersFrames(callers[:count]).Next()
	return snap.Frame_Information{File: frame.File, Line: frame.Line}, nil
}

func operating_read_file(
	_ unsafe.Pointer, path string,
) (data snap.Data, err error) {
	file, open_error := syscall.Open("/"+path, syscall.O_RDONLY, 0)
	if open_error != nil {
		return nil, open_error
	}
	defer syscall.Close(file)
	var facts syscall.Stat_t
	if stat_error := syscall.Fstat(file, &facts); stat_error != nil {
		return nil, stat_error
	}
	data = make(snap.Data, int(facts.Size))
	read_count := 0
	for read_count < len(data) {
		count, read_error := syscall.Read(file, data[read_count:])
		if read_error != nil {
			return nil, read_error
		}
		if count == 0 {
			break
		}
		read_count += count
	}
	return data[:read_count], nil
}

func operating_write_file(
	_ unsafe.Pointer, path string, data snap.Data, permission snap.Permission,
) (err error) {
	file, open_error := syscall.Open(
		path, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_TRUNC, uint32(permission),
	)
	if open_error != nil {
		return open_error
	}
	defer syscall.Close(file)
	written := 0
	for written < len(data) {
		count, write_error := syscall.Write(file, data[written:])
		if write_error != nil {
			return write_error
		}
		written += count
	}
	return nil
}

func standard_error_write(
	_ unsafe.Pointer, data snap.Data,
) (written snap.Data_Size, err error) {
	count, write_error := syscall.Write(2, data)
	return snap.Data_Size(count), write_error
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
func Batch_Expect(t *testing.T, function func(string) (result any), entries []Entry) {
	t.Helper()
	snap.Batch_Expect(t, function, entries)
}

// Batch_Expect_Panic forwards to snap.Batch_Expect_Panic.
func Batch_Expect_Panic(t *testing.T, function func(string), entries []Entry) {
	t.Helper()
	snap.Batch_Expect_Panic(t, function, entries)
}
