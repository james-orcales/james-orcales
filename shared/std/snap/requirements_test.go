package snap_test

import "errors"
import "testing"
import "unsafe"

import "local/james-orcales/shared/std/sim/aver"
import "local/james-orcales/shared/std/snap"

// Test_Equal_Match verifies a matching value returns true and emits no output.
func Test_Equal_Match(t *testing.T) {
  s, output_buffer, _ := test_snapper(nil)
  snapshot := snap.New_Snapshot(
    &snap.New_Snapshot_Input{
      Snapper:   s,
      File_Path: "/fake/test.go",
      Line:      5,
      Expected:  "hello",
    },
  )
  if !snap.Snapshot_Is_Equal(snapshot, "hello") {
    t.Fatal("expected Snapshot_Is_Equal to return true for matching strings")
  }
  if snap.Buffer_Size(output_buffer) != 0 {
    t.Fatalf(
      "expected no diagnostic output, got: %s", snap.Buffer_String(output_buffer),
    )
  }
}

// Test_Equal_Mismatch verifies a differing value returns false and prints a diff.
func Test_Equal_Mismatch(t *testing.T) {
  s, output_buffer, _ := test_snapper(nil)
  snapshot := snap.New_Snapshot(
    &snap.New_Snapshot_Input{
      Snapper:   s,
      File_Path: "/fake/test.go",
      Line:      5,
      Expected:  "expected",
    },
  )
  if snap.Snapshot_Is_Equal(snapshot, "actual") {
    t.Fatal("expected Snapshot_Is_Equal to return false for mismatching strings")
  }
  if !contains(snap.Buffer_String(output_buffer), "Snapshot mismatch /fake/test.go:5") {
    t.Fatalf(
      "expected mismatch header in output, got: %s",
      snap.Buffer_String(output_buffer),
    )
  }
}

// Test_Equal_Legend verifies the mismatch header carries a colored
// expected-vs-actual legend so readers can map - / + to red / green.
func Test_Equal_Legend(t *testing.T) {
  s, output_buffer, _ := test_snapper(nil)
  snapshot := snap.New_Snapshot(
    &snap.New_Snapshot_Input{
      Snapper:   s,
      File_Path: "/fake/test.go",
      Line:      5,
      Expected:  "expected",
    },
  )
  snap.Snapshot_Is_Equal(snapshot, "actual")

  output := snap.Buffer_String(output_buffer)
  if !contains(output, "\033[31mexpected\033[0m") {
    t.Fatalf("expected red 'expected' in legend, got:\n%s", output)
  }
  if !contains(output, "\033[32mactual\033[0m") {
    t.Fatalf("expected green 'actual' in legend, got:\n%s", output)
  }
  if !contains(output, "\033[31m-expected\033[0m\n") {
    t.Fatalf("expected red diff line, got:\n%s", output)
  }
  if !contains(output, "\033[32m+actual\033[0m\n") {
    t.Fatalf("expected green diff line, got:\n%s", output)
  }
}

// Test_Edit_Rewrite verifies Should_Edit writes the updated source content to W
// and prints an UPDATED SNAPSHOT notice to Output.
func Test_Edit_Rewrite(t *testing.T) {
  // Line 5 contains a snap.Edit call — the line-finder scans newlines to locate it.
  source := "package foo\n\nfunc F() {}\n\nvar _ = snap.Edit(`old`)\n"
  s, output_buffer, w_buffer := test_snapper(map[string]string{
    "/fake/test.go": source,
  })
  snapshot := snap.New_Snapshot(
    &snap.New_Snapshot_Input{
      Snapper:     s,
      File_Path:   "/fake/test.go",
      Line:        5,
      Expected:    "old",
      Should_Edit: true,
    },
  )
  if !snap.Snapshot_Is_Equal(snapshot, "new") {
    t.Fatal("expected Snapshot_Is_Equal with Should_Edit=true to return true")
  }
  if !contains(snap.Buffer_String(output_buffer), "UPDATED SNAPSHOT /fake/test.go:5\n") {
    t.Fatalf(
      "expected UPDATED SNAPSHOT notice, got: %s",
      snap.Buffer_String(output_buffer),
    )
  }
  if !contains(snap.Buffer_String(w_buffer), "snap.Init(`new`)") {
    t.Fatalf(
      "expected W to contain snap.Init(`new`), got: %s",
      snap.Buffer_String(w_buffer),
    )
  }
}

// Test_Edit_Line_Delta verifies a multi-line replacement records the correct
// line delta in Snapper.Edits for use by subsequent edits in the same file.
func Test_Edit_Line_Delta(t *testing.T) {
  source := "package foo\n\n\nvar a = snap.Edit(`x`)\nvar b = snap.Edit(`y`)\n"
  s, _, _ := test_snapper(map[string]string{
    "/fake/test.go": source,
  })
  // Rewrites must land in the memory file system so the second edit reads shifted lines.
  s.Write = nil
  s.Write_File = memory_write
  s.Write_File_State = s.File_System_State
  snapshot := snap.New_Snapshot(
    &snap.New_Snapshot_Input{
      Snapper:     s,
      File_Path:   "/fake/test.go",
      Line:        4,
      Expected:    "x",
      Should_Edit: true,
    },
  )
  // Replace "x" with "x\nexpanded": new content has one more newline, delta=1.
  snap.Snapshot_Is_Equal(snapshot, "x\nexpanded")
  if s.Edit_Count != 1 {
    t.Fatalf("expected 1 edit recorded, got %d", s.Edit_Count)
  }
  if s.Edits[0].File_Path != "/fake/test.go" {
    t.Fatalf("expected edit on /fake/test.go, got %s", s.Edits[0].File_Path)
  }
  if s.Edits[0].Delta != 1 {
    t.Fatalf("expected delta=1, got %d", s.Edits[0].Delta)
  }
  // Later edit on line 5 lands on the shifted line 6 of the rewritten source.
  later := snap.New_Snapshot(
    &snap.New_Snapshot_Input{
      Snapper:     s,
      File_Path:   "/fake/test.go",
      Line:        5,
      Expected:    "y",
      Should_Edit: true,
    },
  )
  snap.Snapshot_Is_Equal(later, "z")
  rewritten := (*(*memory_files)(s.File_System_State))["/fake/test.go"]
  if !contains(rewritten, "var b = snap.Init(`z`)") {
    t.Fatalf("expected shifted rewrite, got: %s", rewritten)
  }
  if s.Edit_Count != 2 {
    t.Fatalf("expected 2 edits recorded, got %d", s.Edit_Count)
  }
}

// Test_Edit_Malformed verifies an edit-mode snapshot whose source line has no
// backticked snap.Edit call is reported and returns false.
func Test_Edit_Malformed(t *testing.T) {
  source := "package foo\n\nfunc F() {}\n\nvar _ = snap.Edit(\"old\")\n"
  s, output_buffer, _ := test_snapper(map[string]string{
    "/fake/test.go": source,
  })
  snapshot := snap.New_Snapshot(
    &snap.New_Snapshot_Input{
      Snapper:     s,
      File_Path:   "/fake/test.go",
      Line:        5,
      Expected:    "old",
      Should_Edit: true,
    },
  )
  if snap.Snapshot_Is_Equal(snapshot, "new") {
    t.Fatal("expected malformed snap.Edit to return false")
  }
  if !contains(snap.Buffer_String(output_buffer), "snap.Edit at /fake/test.go:5 must") {
    t.Fatalf("expected malformed notice, got: %s", snap.Buffer_String(output_buffer))
  }
}

// Test_Panic_Match verifies Expect_Panic passes when the panic message matches.
func Test_Panic_Match(t *testing.T) {
  s, _, _ := test_snapper(nil)
  snapshot := snap.New_Snapshot(
    &snap.New_Snapshot_Input{
      Snapper:   s,
      File_Path: "/fake/test.go",
      Line:      5,
      Expected: aver.ASSERTION_FAILURE_MESSAGE_PREFIX +
        "Snapshot_Is_Equal snapshot is bound to a Snapper" +
        "  Always — condition was false: false",
    },
  )
  snap.Expect_Panic(t, snapshot, func() {
    snap.Snapshot_Is_Equal(snap.Snapshot{}, "")
  })
}

// Test_Panic_Mismatch verifies a panic message differing from the snapshot is
// reported as a failed comparison.
func Test_Panic_Mismatch(t *testing.T) {
  s, _, _ := test_snapper(nil)
  snapshot := snap.New_Snapshot(
    &snap.New_Snapshot_Input{
      Snapper:   s,
      File_Path: "/fake/test.go",
      Line:      5,
      Expected:  "boom",
    },
  )
  if snap.Snapshot_Is_Equal(snapshot, "not boom") {
    t.Fatal("expected Snapshot_Is_Equal to return false for mismatching panic messages")
  }
}

// Test_Batch_Expect verifies Batch_Expect checks every entry in order.
func Test_Batch_Expect(t *testing.T) {
  s, _, _ := test_snapper(nil)
  entries := []snap.Entry{
    {
      Name:  "upper",
      Input: "hello",
      Snapshot: snap.New_Snapshot(
        &snap.New_Snapshot_Input{
          Snapper:   s,
          File_Path: "/fake/test.go",
          Line:      5,
          Expected:  "HELLO",
        },
      ),
    },
    {
      Name:  "upper again",
      Input: "world",
      Snapshot: snap.New_Snapshot(
        &snap.New_Snapshot_Input{
          Snapper:   s,
          File_Path: "/fake/test.go",
          Line:      6,
          Expected:  "WORLD",
        },
      ),
    },
  }
  snap.Batch_Expect(t, func(in string) (result any) {
    return upper(in)
  }, entries)
}

// Test_Batch_Panic verifies Batch_Expect_Panic checks every entry's panic in order.
func Test_Batch_Panic(t *testing.T) {
  s, _, _ := test_snapper(nil)
  entries := []snap.Entry{
    {
      Name:  "boom",
      Input: "boom",
      Snapshot: snap.New_Snapshot(
        &snap.New_Snapshot_Input{
          Snapper:   s,
          File_Path: "/fake/test.go",
          Line:      5,
          Expected: aver.ASSERTION_FAILURE_MESSAGE_PREFIX +
            "Snapshot_Is_Equal snapshot is bound to a Snapper" +
            "  Always — condition was false: false",
        },
      ),
    },
  }
  snap.Batch_Expect_Panic(t, func(in string) {
    snap.Snapshot_Is_Equal(snap.Snapshot{}, in)
  }, entries)
}

// Test_Run_Capture verifies Run captures the callback's output and asserts it
// against the snapshot.
func Test_Run_Capture(t *testing.T) {
  s, _, _ := test_snapper(nil)
  // Buffer_Write writes "hello\n"; Run wraps it as "\nSTDOUT:\nhello\n\n".
  snapshot := snap.New_Snapshot(
    &snap.New_Snapshot_Input{
      Snapper:   s,
      File_Path: "/fake/test.go",
      Line:      5,
      Expected:  "\nSTDOUT:\nhello\n\n",
    },
  )
  output, err := snap.Run(t, func() {
    snap.Buffer_Write(&s.Stdout, snap.Data("hello\n"))
  }, snapshot)
  if output != "hello\n" {
    t.Fatalf("expected borrowed stdout, got: %q", output)
  }
  if err != "" {
    t.Fatalf("expected empty stderr, got: %q", err)
  }
}

// Test_Run_Both verifies Run reports Stdout and Stderr together, and Stderr alone.
func Test_Run_Both(t *testing.T) {
  s, _, _ := test_snapper(nil)
  both := snap.New_Snapshot(
    &snap.New_Snapshot_Input{
      Snapper:   s,
      File_Path: "/fake/test.go",
      Line:      5,
      Expected:  "\nSTDOUT:\nout\n\nSTDERR:\nerr\n",
    },
  )
  snap.Run(t, func() {
    snap.Buffer_Write(&s.Stdout, snap.Data("out"))
    snap.Buffer_Write(&s.Stderr, snap.Data("err"))
  }, both)
  only_err := snap.New_Snapshot(
    &snap.New_Snapshot_Input{
      Snapper:   s,
      File_Path: "/fake/test.go",
      Line:      5,
      Expected:  "\nSTDERR:\nerr\n",
    },
  )
  snap.Run(t, func() {
    snap.Buffer_Write(&s.Stderr, snap.Data("err"))
  }, only_err)
  none := snap.New_Snapshot(
    &snap.New_Snapshot_Input{
      Snapper:   s,
      File_Path: "/fake/test.go",
      Line:      5,
      Expected:  "snap.Run: no output",
    },
  )
  snap.Run(t, func() {}, none)
}

// Test_Run_Overflow verifies a capture past BUFFER_SIZE_MAXIMUM panics instead
// of growing storage.
func Test_Run_Overflow(t *testing.T) {
  s, _, _ := test_snapper(nil)
  var oversized [snap.BUFFER_SIZE_MAXIMUM + 1]byte
  if !panics(func() { snap.Buffer_Write(&s.Stdout, snap.Data(oversized[:])) }) {
    t.Fatal("expected an oversized capture to panic")
  }
}

// Test_Storage_Short verifies a short storage slice is refused at bind time.
func Test_Storage_Short(t *testing.T) {
  s, _, _ := test_snapper(nil)
  storage := test_storage()
  storage.Print = storage.Print[:1]
  if !panics(func() { snap.Snapper_Bind_Storage(s, storage) }) {
    t.Fatal("expected short Print storage to panic")
  }
}

// Builds a Snapper backed by an in-memory file system and capture buffers.
func test_snapper(
  source map[string]string,
) (
  s *snap.Snapper, output_buffer *snap.Buffer, w_buffer *snap.Buffer,
) {
  memory_file_system := memory_files(source)
  output_buffer = &snap.Buffer{Bytes: make(snap.Data, snap.BUFFER_SIZE_MAXIMUM)}
  w_buffer = &snap.Buffer{Bytes: make(snap.Data, snap.BUFFER_SIZE_MAXIMUM)}
  s = &snap.Snapper{
    File_System_State: unsafe.Pointer(&memory_file_system),
    Read_File:         memory_read,
    Writer_State:      unsafe.Pointer(w_buffer),
    Write:             buffer_write,
    Output_State:      unsafe.Pointer(output_buffer),
    Output_Write:      buffer_write,
    Write_File:        ignore_write,
    Get_Caller:        empty_caller,
  }
  snap.Snapper_Bind_Storage(s, test_storage())
  return s, output_buffer, w_buffer
}

// Test-tier root: heap is fine here, the library never grows it.
func test_storage() (storage snap.Storage) {
  return snap.Storage{
    Stdout:  make(snap.Data, snap.BUFFER_SIZE_MAXIMUM),
    Stderr:  make(snap.Data, snap.BUFFER_SIZE_MAXIMUM),
    Print:   make(snap.Data, snap.PRINT_SIZE_MAXIMUM),
    Source:  make(snap.Data, snap.SOURCE_SIZE_MAXIMUM),
    Rewrite: make(snap.Data, snap.REWRITE_SIZE_MAXIMUM),
    Report:  make(snap.Data, snap.REPORT_SIZE_MAXIMUM),
    Edits:   make(snap.Edit_Storage, snap.EDIT_COUNT_MAXIMUM),
  }
}

// Reports whether callback panicked. Recover must run directly in the deferred function.
func panics(callback func()) (panicked bool) {
  defer func() {
    store_result(&panicked, recover() != nil)
  }()
  callback()
  return false
}

type memory_files map[string]string

// Rewrites land back in the map so a later edit on the same file sees the shifted lines.
func memory_write(
  state unsafe.Pointer,
  path string,
  data snap.Data,
  permission snap.Permission,
) (err error) {
  (*(*memory_files)(state))[path] = string(data)
  return nil
}

func memory_read(
  state unsafe.Pointer,
  path string,
  destination snap.Data,
) (data snap.Data, err error) {
  content, found := (*(*memory_files)(state))[path]
  if !found {
    return nil, errors.New("missing file")
  }
  if len(content) > len(destination) {
    return nil, errors.New("file exceeds destination")
  }
  size := copy(destination, content)
  return destination[:size], nil
}

func buffer_write(
  state unsafe.Pointer, data snap.Data,
) (written snap.Data_Size, err error) {
  return snap.Buffer_Write((*snap.Buffer)(state), data)
}

func contains(value string, sought string) (present bool) {
  if len(sought) == 0 {
    return true
  }
  if len(sought) > len(value) {
    return false
  }
  last_start := len(value) - len(sought)
  for start_index := 0; !(start_index > last_start); start_index++ {
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
      return true
    }
  }
  return false
}

func store_result[T any](destination *T, value T) {
  *destination = value
}

func ignore_write(_ unsafe.Pointer, _ string, _ snap.Data, _ snap.Permission) (err error) {
  return nil
}

func empty_caller(_ int) (frame_information snap.Frame_Information, err error) {
  return snap.Frame_Information{}, nil
}

func upper(value string) (result string) {
  data := []byte(value)
  for index := 0; index < len(data); index++ {
    if !(data[index] < 'a') {
      if !(data[index] > 'z') {
        data[index] -= 'a' - 'A'
      }
    }
  }
  return string(data)
}
