package snap

import "runtime"
import "testing"

import "local/james-orcales/shared/std/snap"

func Test_Host(t *testing.T) {
  path := t.TempDir() + "/snapshot"
  if err := operating_write_file(nil, path, snap.Data("hello"), 0o600); err != nil {
    t.Fatal(err)
  }
  var storage [FILE_STORAGE_SIZE]byte
  data, err := operating_read_file(nil, path, storage[:])
  if err != nil {
    t.Fatal(err)
  }
  if string(data) != "hello" {
    t.Fatalf("file content = %q", data)
  }
  if &data[0] != &storage[0] {
    t.Fatal("file read did not use caller storage")
  }
  if _, missing_err := operating_read_file(nil, path+"/missing", storage[:]); missing_err == nil {
    t.Fatal("missing file read succeeded")
  }
  if blocked_err := operating_write_file(nil, path+"/missing", snap.Data("x"), 0o600); blocked_err ==
    nil {
    t.Fatal("write below a regular file succeeded")
  }
  _, file, line, _ := runtime.Caller(0)
  frame, err := operating_caller(0)
  if err != nil {
    t.Fatal(err)
  }
  if frame.File != file {
    t.Fatalf("caller file = %q, want %q", frame.File, file)
  }
  if frame.Line != line+1 {
    t.Fatalf("caller line = %d, want %d", frame.Line, line+1)
  }
  written, err := standard_error_write(nil, nil)
  if err != nil {
    t.Fatal(err)
  }
  if written != 0 {
    t.Fatalf("empty stderr write = %d", written)
  }
}

func Test_Snapshots(t *testing.T) {
  _, file, line, _ := runtime.Caller(0)
  snapshot := Init("expected")
  if snapshot.File_Path != file {
    t.Fatalf("snapshot file = %q, want %q", snapshot.File_Path, file)
  }
  if snapshot.Line != line+1 {
    t.Fatalf("snapshot line = %d, want %d", snapshot.Line, line+1)
  }
  if snapshot.Snapper != &Default {
    t.Fatal("snapshot is not bound to Default")
  }
  if snapshot.Should_Edit {
    t.Fatal("Init enabled editing")
  }
  if !Snapshot_Is_Equal(snapshot, "expected") {
    t.Fatal("equal snapshot did not match")
  }
  edited := Edit("original")
  if !edited.Should_Edit {
    t.Fatal("Edit did not enable editing")
  }
  if edited.Snapper != &Default {
    t.Fatal("edit is not bound to Default")
  }
  if edited.Expected_Output != "original" {
    t.Fatal("edit lost expected text")
  }
  copied := New_Snapshot(
    &snap.New_Snapshot_Input{
      Snapper: &Default, File_Path: file, Line: line, Expected: "copied",
    },
  )
  if !Snapshot_Is_Equal(copied, "copied") {
    t.Fatal("constructed snapshot did not match")
  }
}

// FILE_STORAGE_SIZE holds the test snapshot with room to spare.
const FILE_STORAGE_SIZE = 64
