package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	setup "local/james-orcales/setup/internal"
	shared_bytes "local/james-orcales/shared/bytes"
	sysio "local/james-orcales/shared/io"
	systime "local/james-orcales/shared/time"
)

// The root owns timeline advancement, so no internal package can drive an injected IO loop.
func Test_Main_Owns_Driver(t *testing.T) {
	source := main_source(t)
	if !source_contains(source, []byte("driver.Run_Until(")) {
		t.Fatal("main.go does not own the IO Driver")
	}
}

// A callback on the injected IO value must wake the same runner that the root drives.
func Test_Main_Drive_Runner_Advances_Original_IO(t *testing.T) {
	loop, driver, _ := sysio.New_Sim(1)
	t.Cleanup(driver.Deinit)
	retired := false
	stopped := false
	var action func()
	action = func() {
		completion := &sysio.Completion{}
		loop.Next_Tick(completion, func(_ *sysio.Completion) {
			retired = true
			action = func() { stopped = true }
		}, sysio.NEXT_TICK_VSR)
	}
	runner := setup.Runner{
		Rearm: func() (armed setup.Runner_Work_Queued) {
			if action == nil {
				return false
			}
			next := action
			action = nil
			next()
			return true
		},
		Work_Queued: func() (queued setup.Runner_Work_Queued) {
			return action != nil
		},
		Stopped: func() (finished setup.Runner_Stopped) {
			return setup.Runner_Stopped(stopped)
		},
		Status: func() (status setup.Exit_Code) { return setup.EXIT_SUCCESS },
	}
	status := drive_runner(driver, runner, sysio.Stream{})
	if status != setup.EXIT_SUCCESS {
		t.Fatalf("status = %d, want success", status)
	}
	if !retired {
		t.Fatal("the original IO callback did not retire")
	}
}

// The process callback must retire before the outer pump guard can expire.
func Test_Main_Pump_Guard_Exceeds_Process_Deadline(t *testing.T) {
	observed := systime.Duration(0)
	driver := sysio.Driver{
		Run_Until: func(
			_ func() (finished bool), timeout systime.Duration,
		) (completed bool, err error) {
			observed = timeout
			return false, nil
		},
	}
	runner := setup.Runner{
		Rearm:       func() (armed setup.Runner_Work_Queued) { return false },
		Work_Queued: func() (queued setup.Runner_Work_Queued) { return false },
		Stopped:     func() (finished setup.Runner_Stopped) { return false },
		Status:      func() (status setup.Exit_Code) { return setup.EXIT_FAILURE },
	}
	drive_runner(driver, runner, sysio.Stream{})
	if observed <= setup.PROCESS_DURATION_MAX {
		t.Fatalf(
			"pump guard = %d, want more than process deadline %d",
			observed, setup.PROCESS_DURATION_MAX,
		)
	}
}

// Driver deinitialization requires the runner to join every callback-owned operation.
func Test_Main_Deinitializes_Only_Joined_Runner(t *testing.T) {
	deinitialized := 0
	driver := sysio.Driver{Deinit: func() { deinitialized++ }}
	active := setup.Runner{
		Stopped: func() (finished setup.Runner_Stopped) { return false },
	}
	deinitialize_driver(driver, active)
	if deinitialized != 0 {
		t.Fatal("the root deinitialized an active runner")
	}
	joined := setup.Runner{
		Stopped: func() (finished setup.Runner_Stopped) { return true },
	}
	deinitialize_driver(driver, joined)
	if deinitialized != 1 {
		t.Fatalf("deinitializations = %d, want one", deinitialized)
	}
}

// The root must not hide timeline advancement behind a copied IO value.
func Test_Main_Does_Not_Build_Blocking_IO(t *testing.T) {
	source := main_source(t)
	for _, forbidden := range [][]byte{
		[]byte("func draining_io("),
		[]byte("system = loop"),
		[]byte("system.Read ="),
		[]byte("system.Write ="),
		[]byte("system.Close ="),
		[]byte("system.Spawn ="),
	} {
		if source_contains(source, forbidden) {
			t.Errorf("main.go builds a blocking IO wrapper %q", forbidden)
		}
	}
}

// One internal entry point keeps the ordered bootstrap policy out of the composition root.
func Test_Main_Calls_Internal_Main(t *testing.T) {
	source := main_source(t)
	if !source_contains(source, []byte("setup.Main(")) {
		t.Fatal("main.go does not call internal.Main")
	}
	if source_contains(source, []byte("setup.Bootstrap(")) {
		t.Fatal("main.go owns bootstrap orchestration")
	}
}

// Test_Main_Imports_No_Standard_IO prevents a second IO type system in the setup tree.
func Test_Main_Imports_No_Standard_IO(t *testing.T) {
	t.Parallel()
	for _, path := range setup_go_files(t, ".") {
		syntax, parse_err := parser.ParseFile(
			token.NewFileSet(), path, setup_source(t, path), parser.ImportsOnly)
		if parse_err != nil {
			t.Fatalf("parse %s: %v", path, parse_err)
		}
		for _, imported := range syntax.Imports {
			if imported.Path.Value == `"io"` {
				t.Errorf("standard-library io import in %s", path)
			}
		}
	}
}

// One shared IO value preserves the operation boundary from the composition root through setup.
func Test_Internal_Uses_Shared_IO(t *testing.T) {
	source := setup_source(t, filepath.Join("internal", "setup.go"))
	for _, forbidden := range [][]byte{
		[]byte("type File_System struct"),
		[]byte("type Shell struct"),
		[]byte("type Spawn func"),
		[]byte("Run_Command func"),
		[]byte("File_Read_Adapter"),
		[]byte("File_Write_Adapter"),
		[]byte("Process_Adapter"),
		[]byte("Is_Ignored func"),
		[]byte("Font_Present func"),
		[]byte("Copy_Font func"),
		[]byte("Refresh func"),
	} {
		if source_contains(source, forbidden) {
			t.Errorf("setup declares a second IO seam %q", forbidden)
		}
	}
	if !source_contains(source, []byte("IO sysio.IO")) {
		t.Fatal("setup does not inject shared/io.IO")
	}
}

const SETUP_SEARCH_CHUNK_BYTES_MAX = 4096

// Overlap keeps a declaration visible when it crosses one bounded search chunk.
func source_contains(source []byte, target []byte) (contained bool) {
	for offset := 0; offset < len(source); {
		end := offset + SETUP_SEARCH_CHUNK_BYTES_MAX
		if end > len(source) {
			end = len(source)
		}
		if shared_bytes.Contains(
			shared_bytes.Slice(source[offset:end]), shared_bytes.Slice(target),
		) {
			return true
		}
		if end == len(source) {
			return false
		}
		offset = end - len(target) + 1
	}
	return false
}

// Shared algorithm packages keep one bounded implementation for all first-party callers.
func Test_Uses_Shared_Algorithm_Packages(t *testing.T) {
	t.Parallel()
	for _, path := range setup_go_files(t, ".") {
		syntax, parse_err := parser.ParseFile(
			token.NewFileSet(), path, setup_source(t, path), parser.ImportsOnly)
		if parse_err != nil {
			t.Fatalf("parse %s: %v", path, parse_err)
		}
		for _, imported := range syntax.Imports {
			switch imported.Path.Value {
			case `"bytes"`, `"slices"`, `"strings"`:
				t.Errorf(
					"stdlib import %s in %s has a shared equivalent",
					imported.Path.Value,
					path,
				)
			}
		}
	}
}

const SETUP_GO_FILE_COUNT_MAX = 64

const SETUP_ENTRY_COUNT_MAX = 64

const SETUP_SOURCE_BYTES_MAX = 1048576

// The same bounded source reader keeps the composition-root checks and the import audit finite.
func main_source(t *testing.T) (source []byte) {
	return setup_source(t, "main.go")
}

// A repository source file cannot consume an arbitrary quantity of test memory.
func setup_source(t *testing.T, path string) (source []byte) {
	file, open_err := os.Open(path)
	if open_err != nil {
		t.Fatalf("open %s: %v", path, open_err)
	}
	t.Cleanup(func() {
		if close_err := file.Close(); close_err != nil {
			t.Errorf("close %s: %v", path, close_err)
		}
	})
	information, stat_err := file.Stat()
	if stat_err != nil {
		t.Fatalf("stat %s: %v", path, stat_err)
	}
	if information.Size() < 0 {
		t.Fatalf("%s has a negative source size", path)
	}
	if information.Size() > SETUP_SOURCE_BYTES_MAX {
		t.Fatalf("%s exceeds the source size limit", path)
	}
	source = make([]byte, int(information.Size()))
	if len(source) == 0 {
		return source
	}
	count, read_err := file.ReadAt(source, 0)
	if read_err != nil {
		t.Fatalf("read %s: %v", path, read_err)
	}
	if count != len(source) {
		t.Fatalf("read %s: got %d bytes, want %d", path, count, len(source))
	}
	return source
}

// Fixed entry and file counts prevent a generated source tree from making this audit unbounded.
func setup_go_files(t *testing.T, root string) (paths []string) {
	t.Helper()
	directories := []string{root}
	entry_count := 0
	for len(directories) != 0 {
		last := len(directories) - 1
		directory := directories[last]
		directories = directories[:last]
		directory_file, open_err := os.Open(directory)
		if open_err != nil {
			t.Fatalf("open directory %s: %v", directory, open_err)
		}
		entries, _ := directory_file.ReadDir(SETUP_ENTRY_COUNT_MAX + 1)
		close_err := directory_file.Close()
		if close_err != nil {
			t.Fatalf("close directory %s: %v", directory, close_err)
		}
		if len(entries) > SETUP_ENTRY_COUNT_MAX {
			t.Fatal("a setup directory exceeds the entry count limit")
		}
		for _, entry := range entries {
			if entry_count == SETUP_ENTRY_COUNT_MAX {
				t.Fatal("the setup tree exceeds the entry count limit")
			}
			entry_count++
			path := filepath.Join(directory, entry.Name())
			if entry.IsDir() {
				directories = append(directories, path)
				continue
			}
			if filepath.Ext(path) != ".go" {
				continue
			}
			if len(paths) == SETUP_GO_FILE_COUNT_MAX {
				t.Fatal("the setup Go file count exceeds its limit")
			}
			paths = append(paths, path)
		}
	}
	return paths
}
