package main

import (
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"testing"

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

// The root preserves one IO boundary while it makes sequential setup policy wait for callbacks.
func Test_Main_Drains_Shared_IO(t *testing.T) {
	loop, driver, _ := sysio.New_Sim(0)
	t.Cleanup(driver.Deinit)
	system := draining_io(loop, driver)
	if mkdir_err := system.Make_Directory("/setup"); mkdir_err != nil {
		t.Fatalf("make directory: %v", mkdir_err)
	}
	file, create_err := system.Create("/setup/file")
	if create_err != nil {
		t.Fatalf("create file: %v", create_err)
	}
	written := -1
	completion := sysio.Completion{}
	system.Write(&completion, func(_ *sysio.Completion, count int, err error) {
		if err != nil {
			t.Errorf("write: %v", err)
		}
		written = count
	}, file, []byte("setup"), 0)
	if written != len("setup") {
		t.Fatalf("write count = %d, want %d", written, len("setup"))
	}
	closed := false
	system.Close(&completion, func(_ *sysio.Completion, err error) {
		if err != nil {
			t.Errorf("close: %v", err)
		}
		closed = true
	}, file)
	if !closed {
		t.Fatal("close callback did not retire before return")
	}
	spawned := false
	system.Spawn(&completion, func(
		_ *sysio.Completion, _ sysio.Process_Result, err error,
	) {
		if err != nil {
			t.Errorf("spawn: %v", err)
		}
		spawned = true
	}, sysio.Process_Request{Path: "setup"}, systime.SECOND)
	if !spawned {
		t.Fatal("spawn callback did not retire before return")
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
	buffer := make([]byte, SETUP_SOURCE_BYTES_MAX+1)
	count, read_err := io.ReadFull(file, buffer)
	if read_err != nil {
		if read_err != io.ErrUnexpectedEOF {
			t.Fatalf("read %s: %v", path, read_err)
		}
	}
	if count > SETUP_SOURCE_BYTES_MAX {
		t.Fatalf("%s exceeds the source size limit", path)
	}
	return buffer[:count]
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
		entries, read_err := directory_file.ReadDir(SETUP_ENTRY_COUNT_MAX + 1)
		close_err := directory_file.Close()
		if read_err != nil {
			if read_err != io.EOF {
				t.Fatalf("read directory %s: %v", directory, read_err)
			}
		}
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
