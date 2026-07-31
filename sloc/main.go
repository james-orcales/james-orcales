// Package main is the sloc command: the thin composition root over the internal
// library tier, and the one place allowed to bind the real filesystem and shell out
// to git.
package main

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	invariant "local/james-orcales/shared/invariant/default"
	sloc "local/james-orcales/sloc/internal"
)

// Caps a single explicitly-named file read at the bound the library states its source
// bytes against, so a pathologically large file cannot exceed what Source allows.
const MAIN_FILE_BYTES_MAX = sloc.SOURCE_BYTES_MAX

// The counting workers spend most of their time blocked reading files rather than on
// the CPU, so the pool is oversubscribed past the core count: while some workers wait
// on a read, the rest keep the cores busy classifying. Four per core is past the knee —
// on a large tree it matches the wall time of GOMAXPROCS=4×cores without disturbing the
// scheduler, and more workers do not help.
const MAIN_WORKERS_PER_CORE = 4

func main() {
	os.Exit(int(sloc.Main(sloc.Main_Input{
		Arguments:    os.Args,
		Output:       os.Stdout,
		Error_Output: os.Stderr,
		Open: func(root sloc.Root) (file_system fs.FS) {
			return os.DirFS(string(root))
		},
		Path_Is_Directory: main_is_directory,
		Read_File:         main_read_file,
		Ignore_For:        main_git_ignore,
		Classifier: sloc.File_Classifier{
			Kind: sloc.FILE_CLASSIFIER_KIND_BYTES,
		},
		Concurrency: runtime.GOMAXPROCS(0) * MAIN_WORKERS_PER_CORE,
	})))
}

// Reports whether a path names a directory.
func main_is_directory(name sloc.File_Path) (is_directory bool, err error) {
	defer func() {
		invariant.Boolean_Invariants(is_directory, "main_is_directory.is_directory")
	}()
	sloc.File_Path_Invariants(name, "main_is_directory.name")
	information, stat_err := os.Stat(string(name))
	if stat_err != nil {
		return false, stat_err
	}
	return information.IsDir(), nil
}

// Reads up to MAIN_FILE_BYTES_MAX bytes of a file — the bounded read the directory
// walk performs through the file system — capping memory on a huge file.
func main_read_file(name sloc.File_Path) (content sloc.Source, err error) {
	defer func() { sloc.Source_Invariants(content, "main_read_file.content") }()
	sloc.File_Path_Invariants(name, "main_read_file.name")
	file, open_err := os.Open(string(name))
	if open_err != nil {
		return nil, open_err
	}
	defer file.Close()
	information, stat_err := file.Stat()
	if stat_err != nil {
		return nil, stat_err
	}
	byte_size := information.Size()
	if byte_size > MAIN_FILE_BYTES_MAX {
		byte_size = MAIN_FILE_BYTES_MAX
	}
	buffer := make([]byte, byte_size)
	_, read_err := io.ReadFull(io.LimitReader(file, byte_size), buffer)
	if main_read_failed(read_err) {
		return nil, read_err
	}
	return buffer, nil
}

// Reports whether a bounded read ended in a real error rather than a short or empty
// file.
func main_read_failed(read_err error) (failed bool) {
	defer func() { invariant.Boolean_Invariants(failed, "main_read_failed.failed") }()
	if read_err == nil {
		return false
	}
	if errors.Is(read_err, io.EOF) {
		return false
	}
	return !errors.Is(read_err, io.ErrUnexpectedEOF)
}

// Builds an ignore predicate for a directory inside a git work tree: a path is
// ignored when git would not list it. One `git ls-files` lists every kept file under
// the root; a directory holding no kept file is pruned. When the root is not a git
// work tree, or git is unavailable, it returns nil so nothing is ignored.
func main_git_ignore(root sloc.Root) (is_ignored sloc.Ignore_Predicate) {
	sloc.Root_Invariants(root, "main_git_ignore.root")
	// --cached lists tracked files, --others untracked ones, --exclude-standard drops
	// the gitignored untracked files; together they are exactly the files git does not
	// ignore. -z keeps paths literal, and "." scopes the listing to this root.
	command := exec.Command(
		"git", "-C", string(root),
		"ls-files", "-z", "--cached", "--others", "--exclude-standard", "--", ".")
	output, run_err := command.Output()
	if run_err != nil {
		return nil
	}

	kept_files := map[string]bool{}
	kept_directories := map[string]bool{".": true}
	for _, name := range strings.Split(string(output), "\x00") {
		if name == "" {
			continue
		}
		kept_files[name] = true
		parent := path.Dir(name)
		for main_inside_root(sloc.File_Path(parent)) {
			kept_directories[parent] = true
			parent = path.Dir(parent)
		}
	}

	return func(relative_path string, is_directory bool) (ignored bool) {
		if is_directory {
			return !kept_directories[relative_path]
		}
		return !kept_files[relative_path]
	}
}

// Reports whether a path is a directory below the root, the stop condition for
// walking a file's ancestors.
func main_inside_root(parent sloc.File_Path) (inside bool) {
	defer func() { invariant.Boolean_Invariants(inside, "main_inside_root.inside") }()
	sloc.File_Path_Invariants(parent, "main_inside_root.parent")
	if parent == "." {
		return false
	}
	return parent != "/"
}
