// Package main is the setup command: it mirrors this repository's dotfiles into
// the home directory in one direction, writing only the files that are missing
// or differ. It is the thin composition root over the internal library tier's
// Main, and the one place allowed to bind the real filesystem.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"local/james-orcales/setup/internal"
	sysio "local/james-orcales/shared/io"
	iodefault "local/james-orcales/shared/io/default"
	jlog "local/james-orcales/shared/jlog/default"
	timeos "local/james-orcales/shared/time/default"
)

// EXIT_USAGE marks a home directory that cannot be resolved, kept distinct from
// a sync failure so a caller can tell setup from the work it was asked to do.
const EXIT_USAGE = 2

// REPOSITORY_SUBPATH locates this checkout relative to the home directory, the
// fixed clone location this machine setup assumes.
const REPOSITORY_SUBPATH = "code/james-orcales"

// DOTFILES_SUBPATH locates the dotfiles tree — the repo's `home/` directory,
// mirrored into the home directory — derived from the checkout so the two never
// drift apart.
const DOTFILES_SUBPATH = REPOSITORY_SUBPATH + "/home"

// IOSEVKA_SUBPATH locates the vendored Iosevka TTFs relative to the home
// directory, the source Install_Fonts copies from.
const IOSEVKA_SUBPATH = REPOSITORY_SUBPATH + "/third_party/iosevka_nerd_font_mono"

// DIRECTORY_PERMISSIONS is applied to parent directories write_file creates:
// owner-writable, world-readable.
const DIRECTORY_PERMISSIONS = 0o755

// ROOT_REFUSAL is logged when setup is invoked as root: os.UserHomeDir would then
// resolve to root's home, so the bootstrap would build and write everything in the
// wrong place and leave root-owned files behind.
const ROOT_REFUSAL = "run as your normal user, not root"

func main() {
	// The logger is built first, so the root and home checks below report through it. Its
	// clock also drives the io loop, so the logger and the loop share one system clock. The
	// logger renders each flat-JSON line to stderr through a Console — human-readable and
	// colored only when stderr is a terminal — at debug and below, so the dotfiles scan's
	// per-directory chatter shows and every line is stamped from the clock.
	clock, _ := timeos.New_Operating_System_Clock()
	logger := jlog.New(jlog.New_Input{
		Writer: jlog.New_Console(jlog.New_Console_Input{
			Writer: os.Stderr,
			Color:  is_terminal(os.Stderr),
		}),
		Clock:          clock,
		Floor:          jlog.LEVEL_DEBUG,
		Auto_Timestamp: true,
	})
	if os.Geteuid() == 0 {
		jlog.Logger_Error(logger, ROOT_REFUSAL)
		os.Exit(EXIT_USAGE)
	}
	home, home_err := os.UserHomeDir()
	if home_err != nil {
		jlog.Logger_Error(logger, "cannot resolve home directory", jlog.Err(home_err))
		os.Exit(EXIT_USAGE)
	}
	// The install steps spawn every subprocess through one shared io loop, ticked only
	// here. setup runs its commands sequentially, so a single loop driven by successive
	// Run_Until pumps is correct and never re-enters the driver.
	loop, driver := iodefault.New_Operating_System_IO(clock)
	spawn := spawn_command(loop, driver)
	system := setup.File_System{Loop: loop, Run_Until: driver.Run_Until}
	// One Shell — the loop-backed spawn, the process stdout/stderr a build streams to, and the
	// logger setup narrates through — is shared by every step; they run sequentially.
	shell := setup.Shell{Spawn: spawn, Stdout: os.Stdout, Stderr: os.Stderr, Logger: logger}
	// One bootstrap, in order: install direnv (everything downstream is driven by
	// it), sync the dotfiles, install fonts and Neovim, then the Go-toolchain builds
	// (fzf and this repo's own commands — maddox, m2p, sloc), the cargo builds (rust,
	// jj, ripgrep, fd), and finally Ghostty — the one network download — last,
	// so a cheaper earlier failure surfaces before heavy work.
	os.Exit(setup.Bootstrap(&setup.Bootstrap_Input{
		Logger: logger,
		Steps: []setup.Step{
			{Name: "direnv", Run: direnv_step(home, shell)},
			{Name: "dotfiles", Run: dotfiles_step(home, system, shell)},
			{Name: "fonts", Run: fonts_step(home, shell)},
			{Name: "neovim", Run: neovim_step(home, shell)},
			{Name: "fzf", Run: fzf_step(home, shell)},
			{Name: "maddox", Run: maddox_step(home, shell)},
			{Name: "m2p", Run: m2p_step(home, shell)},
			{Name: "sloc", Run: sloc_step(home, shell)},
			{Name: "rust", Run: rust_step(home, shell)},
			{Name: "jj", Run: jj_step(home, shell)},
			{Name: "ripgrep", Run: ripgrep_step(home, shell)},
			{Name: "fd", Run: fdcli_step(home, shell)},
			{Name: "ghostty", Run: ghostty_step(home, shell)},
		},
	}))
}

// Reports whether file is a terminal by its character-device bit — the stdlib-only TTY check,
// no golang.org/x/term dependency, so color is on for an interactive run and off in a pipe or CI.
func is_terminal(file *os.File) (terminal bool) {
	info, stat_err := file.Stat()
	if stat_err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// Returns the synchronous Spawn the install steps are injected with, backed by the real io
// loop: it submits one Spawn op and drives the loop until that op completes, the one place
// the loop is ticked. A start failure folds into a non-zero exit so the seam stays
// exit-only, matching the outcome of a command that ran and failed.
func spawn_command(loop sysio.IO, driver sysio.Driver) (spawn setup.Spawn) {
	return func(request sysio.Process_Request) (result sysio.Process_Result) {
		var completion sysio.Completion
		done := false
		// A failed start yields no exit code of its own, so report it as a failure.
		complete := func(_ *sysio.Completion, spawned sysio.Process_Result, err error) {
			if err != nil {
				spawned.Exit = 1
			}
			result = spawned
			done = true
		}
		loop.Spawn(&completion, complete, request)
		driver.Run_Until(func() (finished bool) { return done }, sysio.FOREVER)
		return result
	}
}

// Returns the bootstrap step that builds direnv from the vendored source with the
// Go toolchain straight into home/.local/bin. It runs first because the shell hook
// and every .envrc depend on direnv being on PATH.
func direnv_step(home string, shell setup.Shell) (run func() (status_code int)) {
	repository := filepath.Join(home, REPOSITORY_SUBPATH)
	return func() (status_code int) {
		return setup.Install_Direnv(&setup.Install_Direnv_Input{
			Direnv_Directory: filepath.Join(repository, "third_party", "direnv"),
			Binary_Directory: filepath.Join(repository, "home", ".local", "bin"),
			Shell:            shell,
		})
	}
}

// Returns the bootstrap step that mirrors the dotfiles tree into the home
// directory and, on darwin, applies the macos defaults.
func dotfiles_step(
	home string, system setup.File_System, shell setup.Shell,
) (run func() (status_code int)) {
	dotfiles_directory := filepath.Join(home, DOTFILES_SUBPATH)
	return func() (status_code int) {
		return setup.Main(&setup.Main_Input{
			File_System:           system,
			Source_Directory:      dotfiles_directory,
			Destination_Directory: home,
			Operating_System:      runtime.GOOS,
			Run_Command:           run_command(shell.Spawn),
			Is_Ignored:            git_ignores(shell.Spawn, dotfiles_directory),
			Logger:                shell.Logger,
		})
	}
}

// Returns the bootstrap step that copies the vendored Iosevka faces into the
// per-OS font directory, refreshing the cache where the OS needs it.
func fonts_step(home string, shell setup.Shell) (run func() (status_code int)) {
	font_directory, refresh_cache := font_destination(home)
	font_source := filepath.Join(home, IOSEVKA_SUBPATH)
	var refresh func() (err error)
	if refresh_cache {
		runner := run_command(shell.Spawn)
		refresh = func() (err error) {
			return runner("fc-cache", []string{"-f", font_directory})
		}
	}
	return func() (status_code int) {
		return setup.Install_Fonts(&setup.Install_Fonts_Input{
			Font_Directory: font_directory,
			Font_Present: func(file string) (present bool) {
				_, stat_err := os.Stat(filepath.Join(font_directory, file))
				return stat_err == nil
			},
			Copy_Font: func(file string) (err error) {
				return copy_file(&copy_file_input{
					Source:      filepath.Join(font_source, file),
					Destination: filepath.Join(font_directory, file),
				})
			},
			Refresh: refresh,
			Logger:  shell.Logger,
		})
	}
}

// Returns the bootstrap step that builds and installs the vendored Neovim.
func neovim_step(home string, shell setup.Shell) (run func() (status_code int)) {
	return func() (status_code int) {
		return setup.Install_Neovim(&setup.Install_Neovim_Input{
			Repository_Directory: filepath.Join(home, REPOSITORY_SUBPATH),
			Shell:                shell,
		})
	}
}

// Returns the bootstrap step that builds fzf from the vendored source with the Go
// toolchain straight into home/.local/bin. fzf is a Go binary, so the build output
// is the install; it only needs the Go toolchain, not cargo, so it runs before the
// rust step.
func fzf_step(home string, shell setup.Shell) (run func() (status_code int)) {
	repository := filepath.Join(home, REPOSITORY_SUBPATH)
	return func() (status_code int) {
		return setup.Install_Fzf(&setup.Install_Fzf_Input{
			Fzf_Directory:    filepath.Join(repository, "third_party", "fzf"),
			Binary_Directory: filepath.Join(repository, "home", ".local", "bin"),
			Shell:            shell,
		})
	}
}

// Returns the bootstrap step that builds this repository's maddox command into
// home/.local/bin with the Go toolchain. maddox is one of this repo's own tools,
// so — unlike the vendored builds — its only idempotency check is whether maddox
// already resolves on PATH; an absent one is rebuilt.
func maddox_step(home string, shell setup.Shell) (run func() (status_code int)) {
	repository := filepath.Join(home, REPOSITORY_SUBPATH)
	return func() (status_code int) {
		return setup.Install_Command(&setup.Install_Command_Input{
			Package_Directory: filepath.Join(repository, "maddox"),
			Binary_Directory:  filepath.Join(repository, "home", ".local", "bin"),
			Binary_Name:       "maddox",
			Shell:             shell,
		})
	}
}

// Returns the bootstrap step that builds this repository's markdown_to_pdf command
// into home/.local/bin as m2p — the name it is invoked by, which is why the package
// directory and the binary name differ. Its only idempotency check is whether m2p
// already resolves on PATH.
func m2p_step(home string, shell setup.Shell) (run func() (status_code int)) {
	repository := filepath.Join(home, REPOSITORY_SUBPATH)
	return func() (status_code int) {
		return setup.Install_Command(&setup.Install_Command_Input{
			Package_Directory: filepath.Join(repository, "markdown_to_pdf"),
			Binary_Directory:  filepath.Join(repository, "home", ".local", "bin"),
			Binary_Name:       "m2p",
			Shell:             shell,
		})
	}
}

// Returns the bootstrap step that builds this repository's sloc command into
// home/.local/bin with the Go toolchain. Its only idempotency check is whether sloc
// already resolves on PATH.
func sloc_step(home string, shell setup.Shell) (run func() (status_code int)) {
	repository := filepath.Join(home, REPOSITORY_SUBPATH)
	return func() (status_code int) {
		return setup.Install_Command(&setup.Install_Command_Input{
			Package_Directory: filepath.Join(repository, "sloc"),
			Binary_Directory:  filepath.Join(repository, "home", ".local", "bin"),
			Binary_Name:       "sloc",
			Shell:             shell,
		})
	}
}

// Returns the bootstrap step that installs the Rust toolchain with rustup and
// symlinks it onto PATH. CARGO_HOME (where rustup installs) comes from the
// environment the .envrc exports; the binaries are linked into the repo's
// .local/bin, the single directory the .envrc puts on PATH.
func rust_step(home string, shell setup.Shell) (run func() (status_code int)) {
	return func() (status_code int) {
		return setup.Install_Rust(&setup.Install_Rust_Input{
			Cargo_Directory: os.Getenv("CARGO_HOME"),
			Link_Directory:  filepath.Join(home, REPOSITORY_SUBPATH, ".local", "bin"),
			Shell:           shell,
		})
	}
}

// Returns the bootstrap step that builds jj from the vendored workspace with cargo
// and installs it into home/.local/bin — a user-facing program alongside Neovim,
// not the repo .local/bin that holds the dev toolchain.
func jj_step(home string, shell setup.Shell) (run func() (status_code int)) {
	repository := filepath.Join(home, REPOSITORY_SUBPATH)
	return func() (status_code int) {
		return setup.Install_Jj(&setup.Install_Jj_Input{
			Jj_Directory:     filepath.Join(repository, "third_party", "jj"),
			Binary_Directory: filepath.Join(repository, "home", ".local", "bin"),
			Shell:            shell,
		})
	}
}

// Returns the bootstrap step that builds ripgrep (rg) from the vendored crate with
// cargo and installs it into home/.local/bin alongside the other user-facing tools.
func ripgrep_step(home string, shell setup.Shell) (run func() (status_code int)) {
	repository := filepath.Join(home, REPOSITORY_SUBPATH)
	return func() (status_code int) {
		return setup.Install_Ripgrep(&setup.Install_Ripgrep_Input{
			Ripgrep_Directory: filepath.Join(repository, "third_party", "ripgrep"),
			Binary_Directory:  filepath.Join(repository, "home", ".local", "bin"),
			Shell:             shell,
		})
	}
}

// Returns the bootstrap step that builds fd from the vendored crate with cargo and
// installs it into home/.local/bin alongside the other user-facing tools.
func fdcli_step(home string, shell setup.Shell) (run func() (status_code int)) {
	repository := filepath.Join(home, REPOSITORY_SUBPATH)
	return func() (status_code int) {
		return setup.Install_Fdcli(&setup.Install_Fdcli_Input{
			Fdcli_Directory:  filepath.Join(repository, "third_party", "fd"),
			Binary_Directory: filepath.Join(repository, "home", ".local", "bin"),
			Shell:            shell,
		})
	}
}

// Returns the bootstrap step that installs Ghostty from its pinned DMG into the
// macOS applications directory and symlinks the app's CLI into home/.local/bin
// alongside the other user-facing tools. It runs last because it is the one network
// download, after every vendored build; off darwin it does nothing.
func ghostty_step(home string, shell setup.Shell) (run func() (status_code int)) {
	repository := filepath.Join(home, REPOSITORY_SUBPATH)
	return func() (status_code int) {
		return setup.Install_Ghostty(&setup.Install_Ghostty_Input{
			Applications_Directory: ghostty_applications_directory(),
			Link_Directory:         filepath.Join(repository, "home", ".local", "bin"),
			Shell:                  shell,
		})
	}
}

// Returns the macOS applications directory Ghostty.app installs into, or empty on
// any other OS — which has no DMG to install — so Install_Ghostty does nothing
// there, the same empty-directory skip the font step uses off its known systems.
func ghostty_applications_directory() (directory string) {
	if runtime.GOOS != "darwin" {
		return ""
	}
	return "/Applications"
}

// Returns the per-OS user font directory and whether its font cache must be
// rebuilt after install. macOS auto-detects ~/Library/Fonts; Linux registers
// fonts under $XDG_DATA_HOME/fonts only after fc-cache. An unknown OS yields no
// directory, so Install_Fonts does nothing there.
func font_destination(home string) (directory string, refresh_cache bool) {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Fonts"), false
	case "linux":
		data_home := os.Getenv("XDG_DATA_HOME")
		if data_home == "" {
			data_home = filepath.Join(home, ".local", "share")
		}
		return filepath.Join(data_home, "fonts"), true
	}
	return "", false
}

// Copy_file_input names the source and destination copy_file moves between,
// bundled so the repeated-string-param rule is satisfied.
// COPY_BYTES_MAX bounds a single streamed copy. 64 MiB dwarfs an Iosevka face
// (~13 MiB) or the direnv binary yet caps the read, satisfying the linter's
// unbounded-read ban.
const COPY_BYTES_MAX = 67108864

type copy_file_input struct {
	Source      string
	Destination string
}

// Streams Source to Destination, creating Destination's parent directory first
// and capping the copy at COPY_BYTES_MAX. It is the real filesystem binding the
// library tier copies through, so it never shells out for a plain file copy.
func copy_file(input *copy_file_input) (err error) {
	source, open_err := os.Open(input.Source)
	if open_err != nil {
		return open_err
	}
	defer source.Close()
	mkdir_err := os.MkdirAll(filepath.Dir(input.Destination), DIRECTORY_PERMISSIONS)
	if mkdir_err != nil {
		return mkdir_err
	}
	destination, create_err := os.Create(input.Destination)
	if create_err != nil {
		return create_err
	}
	// CopyN reports io.EOF once the source — shorter than the cap — is fully
	// copied; a nil error means the file filled the cap, so it is too large.
	_, copy_err := io.CopyN(destination, source, COPY_BYTES_MAX)
	if copy_err == nil {
		destination.Close()
		return errors.New("source exceeds the maximum copy size")
	}
	if !errors.Is(copy_err, io.EOF) {
		destination.Close()
		return copy_err
	}
	return destination.Close()
}

// Returns the external-program runner injected into setup.Main, also used for fc-cache,
// backed by the loop's spawn so setup executes nothing itself. It streams the command's
// output live — forwarding setup's own stdout and stderr — and reports a non-zero exit as
// an error, the signal the macos defaults and the cache refresh check.
func run_command(spawn setup.Spawn) (run func(name string, arguments []string) (err error)) {
	return func(name string, arguments []string) (err error) {
		result := spawn(sysio.Process_Request{
			Path:      name,
			Arguments: arguments,
			Stdout:    os.Stdout,
			Stderr:    os.Stderr,
		})
		if result.Exit != 0 {
			return fmt.Errorf("%s exited with status %d", name, result.Exit)
		}
		return nil
	}
}

// Returns a predicate classifying a batch of paths under directory by whether each is
// gitignored, backed by one `git check-ignore --stdin` run through the loop. The paths are
// fed on stdin — not as arguments — so a whole tree level fits without risking ARG_MAX, and
// the single spawn replaces the per-path spawn that made a large scan crawl through hundreds
// of sequential git processes. check-ignore prints each ignored path it was handed; anything
// it does not print — including on a git error such as no repository or git missing, which
// prints nothing — is treated as not ignored, so a file still syncs rather than silently
// vanishing.
func git_ignores(
	spawn setup.Spawn, directory string,
) (is_ignored func(relative_paths []string) (ignored map[string]bool)) {
	return func(relative_paths []string) (ignored map[string]bool) {
		ignored = map[string]bool{}
		if len(relative_paths) == 0 {
			return ignored
		}
		targets := make([]string, len(relative_paths))
		for index, relative := range relative_paths {
			targets[index] = filepath.Join(directory, relative)
		}
		result := spawn(sysio.Process_Request{
			Path:      "git",
			Arguments: []string{"-C", directory, "check-ignore", "--stdin"},
			Input:     []byte(strings.Join(targets, "\n") + "\n"),
		})
		// The output echoes each ignored input path verbatim, one per line, so map those
		// printed lines back to the relative paths the caller asked about.
		printed := map[string]bool{}
		for _, line := range strings.Split(string(result.Output), "\n") {
			if line != "" {
				printed[line] = true
			}
		}
		for index, target := range targets {
			if printed[target] {
				ignored[relative_paths[index]] = true
			}
		}
		return ignored
	}
}
