// Package main is the setup command: it mirrors this repository's dotfiles into
// the home directory in one direction, writing only the files that are missing
// or differ. It is the thin composition root over the internal library tier's
// Main, and the one place allowed to bind the real filesystem.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/james-orcales/james-orcales/setup/internal"
)

// Exit_usage marks a home directory that cannot be resolved, kept distinct from
// a sync failure so a caller can tell setup from the work it was asked to do.
const exit_usage = 2

// Dotfiles_subpath locates the dotfiles tree relative to the home directory, the
// fixed clone location this machine setup assumes.
const dotfiles_subpath = "code/james-orcales/dotfiles"

// Directory_permissions is applied to parent directories write_file creates:
// owner-writable, world-readable.
const directory_permissions = 0o755

// File_permissions is applied to the dotfiles write_file writes: owner-writable,
// world-readable.
const file_permissions = 0o644

func main() {
	home, home_err := os.UserHomeDir()
	if home_err != nil {
		fmt.Fprintf(os.Stderr, "setup: %v\n", home_err)
		os.Exit(exit_usage)
	}
	os.Exit(setup.Main(&setup.Main_Input{
		Source:                os.DirFS(filepath.Join(home, dotfiles_subpath)),
		Destination:           os.DirFS(home),
		Destination_Directory: home,
		Operating_System:      runtime.GOOS,
		Write_File:            write_file,
		Run_Command:           run_command,
		Stdout:                os.Stdout,
		Stderr:                os.Stderr,
	}))
}

// Writes contents at path, creating its parent directories first. It is the real
// filesystem binding injected into setup.Main so the library tier stays pure.
func write_file(path string, contents []byte) (err error) {
	mkdir_err := os.MkdirAll(filepath.Dir(path), directory_permissions)
	if mkdir_err != nil {
		return mkdir_err
	}
	return os.WriteFile(path, contents, file_permissions)
}

// Runs name with arguments, forwarding setup's own stdout and stderr so a failing
// default surfaces its diagnostics. It is the real binding injected into
// setup.Main so the library tier never executes anything itself.
func run_command(name string, arguments []string) (err error) {
	command := exec.Command(name, arguments...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}
