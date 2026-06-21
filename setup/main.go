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
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/james-orcales/james-orcales/setup/internal"
	sh "github.com/james-orcales/james-orcales/shared/sh/default"
)

// Exit_usage marks a home directory that cannot be resolved, kept distinct from
// a sync failure so a caller can tell setup from the work it was asked to do.
const exit_usage = 2

// Repository_subpath locates this checkout relative to the home directory, the
// fixed clone location this machine setup assumes.
const repository_subpath = "code/james-orcales"

// Dotfiles_subpath locates the dotfiles tree — the repo's `home/` directory,
// mirrored into the home directory — derived from the checkout so the two never
// drift apart.
const dotfiles_subpath = repository_subpath + "/home"

// Iosevka_subpath locates the vendored Iosevka TTFs relative to the home
// directory, the source Install_Fonts copies from.
const iosevka_subpath = repository_subpath + "/third_party/iosevka_nerd_font_mono"

// Directory_permissions is applied to parent directories write_file creates:
// owner-writable, world-readable.
const directory_permissions = 0o755

// File_permissions is applied to the dotfiles write_file writes: owner-writable,
// world-readable.
const file_permissions = 0o644

// Root_refusal is printed when setup is invoked as root: os.UserHomeDir would then
// resolve to root's home, so the bootstrap would build and write everything in the
// wrong place and leave root-owned files behind.
const root_refusal = "setup: run as your normal user, not root"

func main() {
	if os.Geteuid() == 0 {
		fmt.Fprintln(os.Stderr, root_refusal)
		os.Exit(exit_usage)
	}
	home, home_err := os.UserHomeDir()
	if home_err != nil {
		fmt.Fprintf(os.Stderr, "setup: %v\n", home_err)
		os.Exit(exit_usage)
	}
	// One bootstrap, in order: install direnv (everything downstream is driven by
	// it), sync the dotfiles, install fonts and Neovim, then the Go-toolchain builds
	// (fzf and this repo's own commands — maddox, m2p, sloc), the cargo builds (rust,
	// fish, jj, ripgrep, fd), and finally Ghostty — the one network download — last,
	// so a cheaper earlier failure surfaces before heavy work.
	os.Exit(setup.Bootstrap(&setup.Bootstrap_Input{
		Stdout: os.Stdout,
		Steps: []setup.Step{
			{Name: "direnv", Run: direnv_step(home)},
			{Name: "dotfiles", Run: dotfiles_step(home)},
			{Name: "fonts", Run: fonts_step(home)},
			{Name: "neovim", Run: neovim_step(home)},
			{Name: "fzf", Run: fzf_step(home)},
			{Name: "maddox", Run: maddox_step(home)},
			{Name: "m2p", Run: m2p_step(home)},
			{Name: "sloc", Run: sloc_step(home)},
			{Name: "rust", Run: rust_step(home)},
			{Name: "fish", Run: fish_step(home)},
			{Name: "jj", Run: jj_step(home)},
			{Name: "ripgrep", Run: ripgrep_step(home)},
			{Name: "fd", Run: fdcli_step(home)},
			{Name: "ghostty", Run: ghostty_step(home)},
		},
	}))
}

// Returns the bootstrap step that builds direnv from the vendored source with the
// Go toolchain straight into home/.local/bin. It runs first because the shell hook
// and every .envrc depend on direnv being on PATH.
func direnv_step(home string) (run func() (status_code int)) {
	repository := filepath.Join(home, repository_subpath)
	return func() (status_code int) {
		return setup.Install_Direnv(&setup.Install_Direnv_Input{
			Direnv_Directory: filepath.Join(repository, "third_party", "direnv"),
			Binary_Directory: filepath.Join(repository, "home", ".local", "bin"),
			Shell:            sh.Init_Default_Shell(),
		})
	}
}

// Returns the bootstrap step that mirrors the dotfiles tree into the home
// directory and, on darwin, applies the macos defaults.
func dotfiles_step(home string) (run func() (status_code int)) {
	dotfiles_directory := filepath.Join(home, dotfiles_subpath)
	return func() (status_code int) {
		return setup.Main(&setup.Main_Input{
			Source:                os.DirFS(dotfiles_directory),
			Destination:           os.DirFS(home),
			Destination_Directory: home,
			Operating_System:      runtime.GOOS,
			Write_File:            write_file,
			Run_Command:           run_command,
			Is_Ignored:            git_ignores(dotfiles_directory),
			Stdout:                os.Stdout,
			Stderr:                os.Stderr,
		})
	}
}

// Returns the bootstrap step that copies the vendored Iosevka faces into the
// per-OS font directory, refreshing the cache where the OS needs it.
func fonts_step(home string) (run func() (status_code int)) {
	font_directory, refresh_cache := font_destination(home)
	font_source := filepath.Join(home, iosevka_subpath)
	var refresh func() (err error)
	if refresh_cache {
		refresh = func() (err error) {
			return run_command("fc-cache", []string{"-f", font_directory})
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
			Stdout:  os.Stdout,
			Stderr:  os.Stderr,
		})
	}
}

// Returns the bootstrap step that builds and installs the vendored Neovim.
func neovim_step(home string) (run func() (status_code int)) {
	return func() (status_code int) {
		return setup.Install_Neovim(&setup.Install_Neovim_Input{
			Repository_Directory: filepath.Join(home, repository_subpath),
			Shell:                sh.Init_Default_Shell(),
		})
	}
}

// Returns the bootstrap step that builds fzf from the vendored source with the Go
// toolchain straight into home/.local/bin. fzf is a Go binary, so the build output
// is the install; it only needs the Go toolchain, not cargo, so it runs before the
// rust and fish steps.
func fzf_step(home string) (run func() (status_code int)) {
	repository := filepath.Join(home, repository_subpath)
	return func() (status_code int) {
		return setup.Install_Fzf(&setup.Install_Fzf_Input{
			Fzf_Directory:    filepath.Join(repository, "third_party", "fzf"),
			Binary_Directory: filepath.Join(repository, "home", ".local", "bin"),
			Shell:            sh.Init_Default_Shell(),
		})
	}
}

// Returns the bootstrap step that builds this repository's maddox command into
// home/.local/bin with the Go toolchain. maddox is one of this repo's own tools,
// so — unlike the vendored builds — its only idempotency check is whether maddox
// already resolves on PATH; an absent one is rebuilt.
func maddox_step(home string) (run func() (status_code int)) {
	repository := filepath.Join(home, repository_subpath)
	return func() (status_code int) {
		return setup.Install_Command(&setup.Install_Command_Input{
			Package_Directory: filepath.Join(repository, "maddox"),
			Binary_Directory:  filepath.Join(repository, "home", ".local", "bin"),
			Binary_Name:       "maddox",
			Shell:             sh.Init_Default_Shell(),
		})
	}
}

// Returns the bootstrap step that builds this repository's markdown_to_pdf command
// into home/.local/bin as m2p — the name it is invoked by, which is why the package
// directory and the binary name differ. Its only idempotency check is whether m2p
// already resolves on PATH.
func m2p_step(home string) (run func() (status_code int)) {
	repository := filepath.Join(home, repository_subpath)
	return func() (status_code int) {
		return setup.Install_Command(&setup.Install_Command_Input{
			Package_Directory: filepath.Join(repository, "markdown_to_pdf"),
			Binary_Directory:  filepath.Join(repository, "home", ".local", "bin"),
			Binary_Name:       "m2p",
			Shell:             sh.Init_Default_Shell(),
		})
	}
}

// Returns the bootstrap step that builds this repository's sloc command into
// home/.local/bin with the Go toolchain. Its only idempotency check is whether sloc
// already resolves on PATH.
func sloc_step(home string) (run func() (status_code int)) {
	repository := filepath.Join(home, repository_subpath)
	return func() (status_code int) {
		return setup.Install_Command(&setup.Install_Command_Input{
			Package_Directory: filepath.Join(repository, "sloc"),
			Binary_Directory:  filepath.Join(repository, "home", ".local", "bin"),
			Binary_Name:       "sloc",
			Shell:             sh.Init_Default_Shell(),
		})
	}
}

// Returns the bootstrap step that installs the Rust toolchain with rustup and
// symlinks it onto PATH. CARGO_HOME (where rustup installs) comes from the
// environment the .envrc exports; the binaries are linked into the repo's
// .local/bin, the single directory the .envrc puts on PATH.
func rust_step(home string) (run func() (status_code int)) {
	return func() (status_code int) {
		return setup.Install_Rust(&setup.Install_Rust_Input{
			Cargo_Directory: os.Getenv("CARGO_HOME"),
			Link_Directory:  filepath.Join(home, repository_subpath, ".local", "bin"),
			Shell:           sh.Init_Default_Shell(),
		})
	}
}

// Returns the bootstrap step that builds fish from the vendored source with cargo
// and symlinks it onto PATH. cargo installs into CARGO_HOME (from the .envrc
// environment); fish and its tools, user-facing programs, are linked into
// home/.local/bin alongside Neovim, not the repo .local/bin that holds dev tools.
func fish_step(home string) (run func() (status_code int)) {
	repository := filepath.Join(home, repository_subpath)
	return func() (status_code int) {
		return setup.Install_Fish(&setup.Install_Fish_Input{
			Fish_Directory:  filepath.Join(repository, "third_party", "fish-shell"),
			Cargo_Directory: os.Getenv("CARGO_HOME"),
			Link_Directory:  filepath.Join(repository, "home", ".local", "bin"),
			Shell:           sh.Init_Default_Shell(),
		})
	}
}

// Returns the bootstrap step that builds jj from the vendored workspace with cargo
// and symlinks it into home/.local/bin. cargo installs into CARGO_HOME (from the
// .envrc environment); jj, a user-facing program, is linked alongside Neovim and
// fish, not the repo .local/bin that holds the dev toolchain.
func jj_step(home string) (run func() (status_code int)) {
	repository := filepath.Join(home, repository_subpath)
	return func() (status_code int) {
		return setup.Install_Jj(&setup.Install_Jj_Input{
			Jj_Directory:    filepath.Join(repository, "third_party", "jj"),
			Cargo_Directory: os.Getenv("CARGO_HOME"),
			Link_Directory:  filepath.Join(repository, "home", ".local", "bin"),
			Shell:           sh.Init_Default_Shell(),
		})
	}
}

// Returns the bootstrap step that builds ripgrep (rg) from the vendored crate with
// cargo and symlinks it into home/.local/bin alongside the other user-facing
// tools. cargo installs into CARGO_HOME (from the .envrc environment).
func ripgrep_step(home string) (run func() (status_code int)) {
	repository := filepath.Join(home, repository_subpath)
	return func() (status_code int) {
		return setup.Install_Ripgrep(&setup.Install_Ripgrep_Input{
			Ripgrep_Directory: filepath.Join(repository, "third_party", "ripgrep"),
			Cargo_Directory:   os.Getenv("CARGO_HOME"),
			Link_Directory:    filepath.Join(repository, "home", ".local", "bin"),
			Shell:             sh.Init_Default_Shell(),
		})
	}
}

// Returns the bootstrap step that builds fd from the vendored crate with cargo and
// symlinks it into home/.local/bin alongside the other user-facing tools. cargo
// installs into CARGO_HOME (from the .envrc environment).
func fdcli_step(home string) (run func() (status_code int)) {
	repository := filepath.Join(home, repository_subpath)
	return func() (status_code int) {
		return setup.Install_Fdcli(&setup.Install_Fdcli_Input{
			Fdcli_Directory: filepath.Join(repository, "third_party", "fd"),
			Cargo_Directory: os.Getenv("CARGO_HOME"),
			Link_Directory:  filepath.Join(repository, "home", ".local", "bin"),
			Shell:           sh.Init_Default_Shell(),
		})
	}
}

// Returns the bootstrap step that installs Ghostty from its pinned DMG into the
// macOS applications directory and symlinks the app's CLI into home/.local/bin
// alongside the other user-facing tools. It runs last because it is the one network
// download, after every vendored build; off darwin it does nothing.
func ghostty_step(home string) (run func() (status_code int)) {
	repository := filepath.Join(home, repository_subpath)
	return func() (status_code int) {
		return setup.Install_Ghostty(&setup.Install_Ghostty_Input{
			Applications_Directory: ghostty_applications_directory(),
			Link_Directory:         filepath.Join(repository, "home", ".local", "bin"),
			Shell:                  sh.Init_Default_Shell(),
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
// Copy_bytes_max bounds a single streamed copy. 64 MiB dwarfs an Iosevka face
// (~13 MiB) or the direnv binary yet caps the read, satisfying the linter's
// unbounded-read ban.
const copy_bytes_max = 67108864

type copy_file_input struct {
	Source      string
	Destination string
}

// Streams Source to Destination, creating Destination's parent directory first
// and capping the copy at copy_bytes_max. It is the real filesystem binding the
// library tier copies through, so it never shells out for a plain file copy.
func copy_file(input *copy_file_input) (err error) {
	source, open_err := os.Open(input.Source)
	if open_err != nil {
		return open_err
	}
	defer source.Close()
	mkdir_err := os.MkdirAll(filepath.Dir(input.Destination), directory_permissions)
	if mkdir_err != nil {
		return mkdir_err
	}
	destination, create_err := os.Create(input.Destination)
	if create_err != nil {
		return create_err
	}
	// CopyN reports io.EOF once the source — shorter than the cap — is fully
	// copied; a nil error means the file filled the cap, so it is too large.
	_, copy_err := io.CopyN(destination, source, copy_bytes_max)
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

// Returns a predicate reporting whether a path under directory is gitignored,
// backed by `git check-ignore`. A git error (not a repository, or git missing)
// reports not-ignored, so a file still syncs rather than silently vanishing.
func git_ignores(directory string) (is_ignored func(relative_path string) (ignored bool)) {
	return func(relative_path string) (ignored bool) {
		target := filepath.Join(directory, relative_path)
		probe := exec.Command("git", "-C", directory, "check-ignore", "--quiet", target)
		return probe.Run() == nil
	}
}
