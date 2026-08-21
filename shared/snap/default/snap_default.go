// Package snap is the composition-tier sibling of the snap library. It wires
// the snap library to the real OS (the filesystem and stderr through the os gateway,
// runtime.Caller) and re-exports the surface so callers can write:
//
//	import snap "local/james-orcales/shared/snap/default"
//
// and use snap.Init / snap.Edit / snap.Expect / … as if no split had happened.
package snap

import (
	"runtime"
	"testing"
	"unsafe"

	"local/james-orcales/shared/os/default"
	"local/james-orcales/shared/snap"
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
type Entry = snap.Entry

// Default is the Snapper the package-level Init / Edit / … convenience
// functions bind to, wired to the host OS: the local filesystem, standard
// error, file writes, and runtime.Caller. This is the one place in the snap
// tree where ambient binding is permitted, and the one place that allocates:
// a struct field cannot be a fixed array, a package may declare no var but
// Default, and TestMain must be exactly aver.Run_Test_Main(m), so no root
// exists to own the storage. Six allocations, once, at package init. Built
// in place because Snapper carries a mutex and must not be copied. Tests
// that need to redirect I/O construct their own Snapper via the snap package.
var Default = snap.Snapper{
	Read_File:    operating_read_file,
	Output_Write: standard_error_write,
	Write_File:   operating_write_file,
	Get_Caller:   operating_caller,
	Stdout:       snap.Buffer{Bytes: make(snap.Data, snap.BUFFER_SIZE_MAXIMUM)},
	Stderr:       snap.Buffer{Bytes: make(snap.Data, snap.BUFFER_SIZE_MAXIMUM)},
	Print:        make(snap.Data, snap.PRINT_SIZE_MAXIMUM),
	Source:       make(snap.Data, snap.SOURCE_SIZE_MAXIMUM),
	Rewrite:      make(snap.Data, snap.REWRITE_SIZE_MAXIMUM),
	Report:       make(snap.Data, snap.REPORT_SIZE_MAXIMUM),
	Edits:        make(snap.Edit_Storage, snap.EDIT_COUNT_MAXIMUM),
}

// Counts skip from operating_caller's own caller, matching the Get_Caller
// contract exactly; runtime.Caller allocates nothing where runtime.Callers
// needs a Frames iterator.
func operating_caller(skip int) (frame_information snap.Frame_Information, err error) {
	_, file, line, _ := runtime.Caller(skip + 1)
	return snap.Frame_Information{File: file, Line: line}, nil
}

func operating_read_file(
	_ unsafe.Pointer, path string, destination snap.Data,
) (data snap.Data, err error) {
	content, read_err := os.File_Read_Into(os.File_Path(path), os.File_Content(destination))
	return snap.Data(content), read_err
}

func operating_write_file(
	_ unsafe.Pointer, path string, data snap.Data, permission snap.Permission,
) (err error) {
	return os.File_Write(
		os.File_Path(path), os.File_Content(data), os.File_Permission(permission),
	)
}

func standard_error_write(
	_ unsafe.Pointer, data snap.Data,
) (written snap.Data_Size, err error) {
	if write_err := os.Standard_Error_Write(os.File_Content(data)); write_err != nil {
		return 0, write_err
	}
	return snap.Data_Size(len(data)), nil
}

// Init creates a snapshot bound to Default with the call-site location
// captured via runtime.Caller. WARN: brittle under go:generate — the
// source location is captured at runtime.
func Init(data string) (snapshot snap.Snapshot) {
	// Frames above Get_Caller: 0 Snapper_Init_At, 1 this wrapper, 2 the test call site.
	return snap.Snapper_Init_At(&Default, 2, data, false)
}

// Edit creates a snapshot bound to Default that will rewrite the source line
// with the actual output on the next test run. Use this temporarily to
// update a specific snapshot, then change it back to Init.
func Edit(data string) (snapshot snap.Snapshot) {
	// Same frame depth as Init.
	return snap.Snapper_Init_At(&Default, 2, data, true)
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
