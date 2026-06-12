// Package setup is the pure library tier of the setup binary, the one-shot
// bootstrap for this machine. It mirrors a tree of dotfiles into the home
// directory in one direction, repo to home — Plan diffs two injected filesystems
// and returns the writes a sync would make, and Main performs them through an
// injected writer — Install_Neovim builds and installs the vendored Neovim, and
// Install_Fonts copies the vendored fonts into the OS font directory. Every entry
// binds to no real filesystem and stays a black box under test.
package setup

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/james-orcales/james-orcales/shared/sh"
)

// Dotfile_bytes_max bounds a single dotfile read into one fixed buffer. 1 MiB
// dwarfs any real configuration file yet caps memory against a pathological
// input, satisfying the linter's unbounded-read ban.
const dotfile_bytes_max = 1048576

// Exit_failure is the process exit code for a planning or write failure during
// an otherwise well-formed run.
const exit_failure = 1

// Main_Input carries the injected dependencies Main needs to sync dotfiles.
type Main_Input struct {
	// Source is the dotfiles tree to mirror from.
	Source fs.FS
	// Destination is a read-only view of the home directory, used for diffing.
	Destination fs.FS
	// Destination_Directory is the absolute home directory writes land under.
	Destination_Directory string
	// Write_File persists contents at an absolute path, creating parent
	// directories. It is injected so this library tier never binds to os
	// directly; package main supplies the filesystem-backed implementation.
	Write_File func(path string, contents []byte) (err error)
	// Operating_System gates the macos defaults step, which runs only on
	// "darwin" (a runtime.GOOS value).
	Operating_System string
	// Run_Command runs an external program by name with arguments. It is injected
	// so this library tier never executes anything; package main supplies the
	// exec-backed runner used for the macos defaults.
	Run_Command func(name string, arguments []string) (err error)
	// Is_Ignored reports whether a source path is gitignored; it is threaded to
	// Plan so the install tree under .local is not mirrored into home. Nil ignores
	// nothing.
	Is_Ignored func(relative_path string) (ignored bool)
	// Stdout receives one line naming each file written.
	Stdout io.Writer
	// Stderr receives a diagnostic line when planning or a write fails.
	Stderr io.Writer
}

// Main syncs the source dotfiles into the home directory and, on darwin, applies
// the macos defaults, returning a process exit code. It is the binary's single
// entry point, kept here so package main stays a thin, untested shell.
func Main(input *Main_Input) (status_code int) {
	writes, plan_err := Plan(&Plan_Input{
		Source:                input.Source,
		Destination:           input.Destination,
		Destination_Directory: input.Destination_Directory,
		Is_Ignored:            input.Is_Ignored,
	})
	if plan_err != nil {
		fmt.Fprintf(input.Stderr, "setup: %v\n", plan_err)
		return exit_failure
	}
	for _, write := range writes {
		write_err := input.Write_File(write.Destination_Path, write.Contents)
		if write_err != nil {
			fmt.Fprintf(input.Stderr, "setup: %v\n", write_err)
			return exit_failure
		}
		fmt.Fprintf(input.Stdout, "setup: wrote %s\n", write.Destination_Path)
	}
	// The macos defaults touch macOS-only preference domains, so they run there
	// and nowhere else.
	if input.Operating_System != "darwin" {
		return 0
	}
	return apply_macos_defaults(input.Run_Command, input.Stderr)
}

// Takes just the runner and the error sink it uses, not the whole Main_Input: a
// function whose parameter is *Main_Input would be forced to be named Main. Runs
// every macos defaults command, stopping at the first that fails. It prints
// nothing per command — 35 lines of `defaults write` is noise, not progress.
func apply_macos_defaults(
	run func(name string, arguments []string) (err error), stderr io.Writer,
) (status_code int) {
	for _, command := range macos_commands() {
		run_err := run(command.Name, command.Arguments)
		if run_err != nil {
			fmt.Fprintf(stderr, "setup: %v\n", run_err)
			return exit_failure
		}
	}
	return 0
}

// Step is one named stage of the bootstrap: the label Bootstrap announces before
// running it, and the work itself.
type Step struct {
	// Name is the label Bootstrap announces before running this step.
	Name string
	// Run performs the step and returns its process exit code.
	Run func() (status_code int)
}

// Bootstrap_Input carries the ordered steps and the sink their progress is
// announced to.
type Bootstrap_Input struct {
	// Steps run in slice order; the first to return non-zero stops the rest.
	Steps []Step
	// Stdout receives one line naming each step as it starts.
	Stdout io.Writer
}

// Bootstrap runs the setup steps in their fixed order of operations, announcing
// each by name before it runs and returning the first non-zero status — skipping
// the rest — so a cheap early failure surfaces before later, heavier work.
// Package main supplies the steps.
func Bootstrap(input *Bootstrap_Input) (status_code int) {
	for _, step := range input.Steps {
		fmt.Fprintf(input.Stdout, "setup: %s\n", step.Name)
		step_status := step.Run()
		if step_status != 0 {
			return step_status
		}
	}
	return 0
}

// Macos_defaults_script lists the `defaults write` and `killall` commands that
// configure macOS, one per line, parsed into commands at run time. The clock
// date format is appended separately by macos_commands because its value holds
// spaces that whitespace-splitting a line would break apart.
const macos_defaults_script = `
defaults write com.apple.dock autohide -bool true
defaults write com.apple.dock autohide-delay -float 0
defaults write com.apple.dock autohide-time-modifier -int 0
defaults write com.apple.dock orientation -string left
defaults write com.apple.dock show-recents -bool false
killall Dock
defaults write com.apple.finder AppleShowAllExtensions -bool true
defaults write com.apple.finder AppleShowAllFiles -bool true
defaults write com.apple.finder AppleShowScrollBars -bool true
defaults write com.apple.finder ShowPathbar -bool true
defaults write com.apple.finder ShowStatusBar -bool true
defaults write com.apple.finder NewWindowTarget -string Home
defaults write com.apple.finder FXPreferredViewStyle -string Nlsv
defaults write com.apple.finder FXDefaultSearchScope -string SCcf
defaults write com.apple.finder _FXSortFoldersFirst -bool true
defaults write com.apple.finder _FXShowPosixPathInTitle -bool true
killall Finder
defaults write com.apple.screensaver askForPassword -int 1
defaults write com.apple.screensaver askForPasswordDelay -int 0
defaults write com.apple.AdLib allowApplePersonalizedAdvertising -bool false
defaults write com.apple.desktopservices DSDontWriteNetworkStores -bool true
defaults write com.apple.desktopservices DSDontWriteUSBStores -bool true
defaults write com.apple.SoftwareUpdate AutomaticCheckEnabled -bool true
defaults write com.apple.SoftwareUpdate ScheduleFrequency -int 1
defaults write com.apple.SoftwareUpdate AutomaticDownload -int 0
defaults write com.apple.SoftwareUpdate CriticalUpdateInstall -int 1
defaults write NSGlobalDomain com.apple.mouse.linear -bool true
defaults write NSGlobalDomain WebKitDeveloperExtras -bool true
defaults write NSGlobalDomain AppleShowScrollBars -string always
defaults write NSGlobalDomain NSAutomaticCapitalizationEnabled -bool false
defaults write NSGlobalDomain NSAutomaticDashSubstitutionEnabled -bool false
defaults write NSGlobalDomain NSAutomaticInlinePredictionEnabled -bool false
defaults write NSGlobalDomain NSAutomaticPeriodSubstitutionEnabled -bool false
defaults write NSGlobalDomain NSAutomaticQuoteSubstitutionEnabled -bool false
defaults write NSGlobalDomain NSAutomaticSpellingCorrectionEnabled -bool false
`

// One external program invocation parsed from macos_defaults_script.
type macos_command struct {
	Name      string
	Arguments []string
}

// Parses macos_defaults_script into one command per non-blank line and appends
// the clock date format command, whose spaced value cannot share the line format.
func macos_commands() (commands []macos_command) {
	commands = []macos_command{}
	for line := range strings.Lines(macos_defaults_script) {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		commands = append(commands, macos_command{
			Name:      fields[0],
			Arguments: fields[1:],
		})
	}
	return append(commands, macos_command{
		Name: "defaults",
		Arguments: []string{
			"write", "com.apple.menuextra.clock", "DateFormat",
			"-string", "EEE MMM d mm:HH",
		},
	})
}

// File_Write is a single pending write the sync would perform.
type File_Write struct {
	// Destination_Path is the absolute path the contents must be written to: the
	// source file's path relative to the source root, joined onto the home dir.
	Destination_Path string
	// Contents is the exact source bytes to write. Comparing them against the
	// destination is what decided this write was needed at all.
	Contents []byte
}

// Plan_Input carries the injected dependencies Plan needs to decide what to sync.
type Plan_Input struct {
	// Source is the dotfiles tree, walked in full from its root.
	Source fs.FS
	// Destination is a read-only view of the home directory, used only to read
	// existing files so Plan can skip writes that would change nothing.
	Destination fs.FS
	// Destination_Directory is the absolute home directory the relative source
	// paths are mirrored under to form each write's Destination_Path.
	Destination_Directory string
	// Is_Ignored reports whether a source path, relative to the source root, is
	// gitignored. An ignored file is not synced and an ignored directory is pruned,
	// so the generated install tree under .local never reaches the home directory.
	// Nil ignores nothing, the sync's behavior before the filter existed.
	Is_Ignored func(relative_path string) (ignored bool)
}

// Plan returns the writes that would bring the home directory in line with the
// source dotfiles: every regular file under the source tree, each emitted only
// when the destination is missing or its contents differ.
func Plan(input *Plan_Input) (writes []File_Write, err error) {
	writes = []File_Write{}
	walk_err := fs.WalkDir(input.Source, ".",
		func(source_path string, entry fs.DirEntry, step_err error) (result error) {
			if step_err != nil {
				return step_err
			}
			if plan_is_ignored(input.Is_Ignored, source_path) {
				// Prune an ignored directory so its contents — the install tree can
				// be thousands of files — are never walked or read; skip a file.
				if entry.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if entry.IsDir() {
				return nil
			}
			write, planned, plan_err := plan_file(&plan_file_input{
				Input:       input,
				Source_Path: source_path,
			})
			if plan_err != nil {
				return plan_err
			}
			if planned {
				writes = append(writes, write)
			}
			return nil
		})
	if walk_err != nil {
		return nil, walk_err
	}
	return writes, nil
}

// Reports whether the walk should skip source_path because it is gitignored. A
// nil predicate ignores nothing, so the filter stays opt-in per Plan_Input.
func plan_is_ignored(
	is_ignored func(relative_path string) (ignored bool), source_path string,
) (ignored bool) {
	if is_ignored == nil {
		return false
	}
	return is_ignored(source_path)
}

// Carries the arguments for planning one source file.
type plan_file_input struct {
	Input       *Plan_Input
	Source_Path string
}

// Decides whether one source file needs syncing. planned is false when the
// destination already holds identical bytes; otherwise it returns the write
// mirroring the source path under the home directory.
func plan_file(input *plan_file_input) (write File_Write, planned bool, err error) {
	source_contents, read_err := read_source(input.Input.Source, input.Source_Path)
	if read_err != nil {
		return File_Write{}, false, read_err
	}
	if destination_matches(input.Input.Destination, input.Source_Path, source_contents) {
		return File_Write{}, false, nil
	}
	destination_path := filepath.Join(input.Input.Destination_Directory, input.Source_Path)
	return File_Write{
		Destination_Path: destination_path,
		Contents:         source_contents,
	}, true, nil
}

// Reads a source file's full contents, bounded by dotfile_bytes_max.
func read_source(source fs.FS, source_path string) (contents []byte, err error) {
	file, open_err := source.Open(source_path)
	if open_err != nil {
		return nil, open_err
	}
	defer file.Close()
	return read_bounded(file)
}

// Reports whether the home directory already holds exactly source_contents at
// relative_path. A destination that is absent or unreadable counts as a
// mismatch, so the file is written — the rule the original installer used.
func destination_matches(
	destination fs.FS, relative_path string, source_contents []byte) (matches bool) {
	file, open_err := destination.Open(relative_path)
	if open_err != nil {
		return false
	}
	defer file.Close()
	destination_contents, read_err := read_bounded(file)
	if read_err != nil {
		return false
	}
	return bytes.Equal(source_contents, destination_contents)
}

// Reads up to dotfile_bytes_max bytes from file into one fixed buffer, erroring
// if the file overflows the cap or a read fails mid-stream, so a silently
// truncated dotfile never passes as the whole file.
func read_bounded(file fs.File) (contents []byte, err error) {
	buffer := make([]byte, dotfile_bytes_max)
	read_total := 0
	for read_total < len(buffer) {
		n, read_err := file.Read(buffer[read_total:])
		read_total += n
		if read_err == io.EOF {
			return buffer[:read_total], nil
		}
		if read_err != nil {
			return nil, read_err
		}
	}
	return nil, errors.New("dotfile exceeds the maximum size")
}

// Installed_Input names the binary an install step probes and the version it must
// report. Two string fields, so it is a struct rather than two parameters.
type Installed_Input struct {
	// Shell runs the version probe.
	Shell *sh.Shell
	// Executable is the binary's exact managed path — probed directly, not via
	// PATH, so a missing symlink never hides a present install.
	Executable string
	// Version is the prefix the binary's --version output must start with; a prefix
	// so build or commit suffixes do not matter.
	Version string
}

// Installed is the shared idempotency probe: it reports whether the binary at its
// managed path already reports the wanted version. An install step that is
// Installed does no work; one that is not reinstalls. direnv and Neovim verify the
// same rule their own way (a version subcommand, and a which-resolved path).
func Installed(input *Installed_Input) (yes bool) {
	version := sh.Shell_Pipe(input.Shell, input.Executable, "--version")
	return strings.HasPrefix(version, input.Version)
}

// Neovim_source_subpath locates the vendored Neovim source relative to the
// checkout root. make is pointed at it with -C, so the build needs no
// working-directory plumbing through the injected runner.
const neovim_source_subpath = "third_party/neovim"

// Neovim_prefix_subpath is the install prefix relative to the checkout root.
// Neovim installs to <prefix>/bin/nvim and derives its runtime as
// <prefix>/share/nvim/runtime by stripping the binary's name and its parent
// "bin" component from the resolved path. home/.local mirrors ~/.local, so nvim
// lands at home/.local/bin/nvim — on PATH — and finds its own runtime, no symlink.
const neovim_prefix_subpath = "home/.local"

// Neovim_build_type is the CMAKE_BUILD_TYPE the bootstrap compiles: optimized,
// but with enough debug info to recover a backtrace if Neovim ever crashes.
const neovim_build_type = "RelWithDebInfo"

// Neovim_install_goal is the second make goal, run after the configure-build
// pass to copy the binary, runtime, and parsers under the prefix.
const neovim_install_goal = "install"

// Neovim_version is the release the bootstrap wants, tracking the vendored
// source's NVIM_VERSION_* in third_party/neovim/CMakeLists.txt. The build is
// skipped when an nvim already on PATH reports it, so a bootstrap that already
// has the wanted nvim does no work. Bump it with the vendored source.
const neovim_version = "v0.12.3"

// Install_Neovim_Input carries the injected dependencies Install_Neovim needs to
// build and install the vendored Neovim.
type Install_Neovim_Input struct {
	// Repository_Directory is the absolute checkout root the build subpaths join
	// onto to form the source and prefix locations.
	Repository_Directory string
	// Shell runs make. Its Stdout and Stderr also receive setup's own narration,
	// so the build output and the lines reporting on it share one stream.
	Shell *sh.Shell
}

// Install_Neovim builds the vendored Neovim and installs it under the local
// prefix, where it lands at local/bin/nvim — already on PATH — and finds its own
// runtime. Every step is idempotent: make rebuilds only what changed and install
// re-copies, so a repeated bootstrap converges without error.
func Install_Neovim(input *Install_Neovim_Input) (status_code int) {
	// Building Neovim is the expensive step, so it is gated on the checkout's own
	// nvim not already being installed: a bootstrap that has it does no work.
	if neovim_already_installed(input.Shell, input.Repository_Directory) {
		fmt.Fprintf(input.Shell.Stdout, "setup: nvim %s installed\n", neovim_version)
		return 0
	}
	for index, arguments := range neovim_make_invocations(input.Repository_Directory) {
		fmt.Fprintf(input.Shell.Stdout, "setup: neovim: %s\n", neovim_make_phase(index))
		// Shell_Spawn reads the first argument as the executable; make's own output
		// already streamed to the Shell, so a generic line is all setup adds.
		if !sh.Shell_Spawn(input.Shell, arguments...) {
			fmt.Fprintln(input.Shell.Stderr, "setup: neovim build failed")
			return exit_failure
		}
	}
	return 0
}

// Returns the progress label for make invocation index: invocation 0 configures
// and builds against the prefix, invocation 1 installs.
func neovim_make_phase(index int) (phase string) {
	if index == 0 {
		return "configuring prefix..."
	}
	return "installing..."
}

// Returns the two make invocations the build runs in order: configure-and-build,
// then install. Each begins with the executable, "make", because Shell_Spawn
// reads the first argument as the program. Both carry the same prefix so the
// Makefile's checkprefix never re-runs cmake between them; the install pass
// differs only by the appended install goal. -C aims make at the vendored source
// without disturbing the caller's working directory.
func neovim_make_invocations(repository_directory string) (invocations [][]string) {
	source := filepath.Join(repository_directory, neovim_source_subpath)
	prefix := filepath.Join(repository_directory, neovim_prefix_subpath)
	build := []string{
		"make", "-C", source,
		"CMAKE_BUILD_TYPE=" + neovim_build_type,
		"CMAKE_INSTALL_PREFIX=" + prefix,
	}
	install := append(slices.Clone(build), neovim_install_goal)
	return [][]string{build, install}
}

// Reports whether version_output — the text `nvim --version` prints — names the
// wanted release neovim_version. Only the leading vMAJOR.MINOR.PATCH token is
// compared, so a build's -dev or +commit suffix does not matter. Empty output,
// from no nvim on PATH, is not the release, so the build runs.
func neovim_version_present(version_output string) (present bool) {
	first_line, _, _ := strings.Cut(version_output, "\n")
	for _, field := range strings.Fields(first_line) {
		if !strings.HasPrefix(field, "v") {
			continue
		}
		base, _, _ := strings.Cut(field, "-")
		return base == neovim_version
	}
	return false
}

// Reports whether the nvim on PATH is the checkout's own build at the wanted
// release. It resolves nvim first and rejects a path outside the repository, so a
// system package at the same version cannot stand in; only an in-repository
// binary then has its version checked.
func neovim_already_installed(shell *sh.Shell, repository_directory string) (installed bool) {
	executable := sh.Shell_Pipe(shell, "which", "nvim")
	if !strings.HasPrefix(executable, repository_directory+"/") {
		return false
	}
	return neovim_version_present(sh.Shell_Pipe(shell, executable, "--version"))
}

// Returns the Iosevka TTF filenames the install copies. A function rather than a
// package var so the list stays within the deterministic tier's value rules.
func iosevka_font_files() (files []string) {
	return []string{
		"IosevkaNerdFontMono-Regular.ttf",
		"IosevkaNerdFontMono-Bold.ttf",
		"IosevkaNerdFontMono-Oblique.ttf",
		"IosevkaNerdFontMono-BoldOblique.ttf",
	}
}

// Install_Fonts_Input carries the injected dependencies Install_Fonts needs to
// place the vendored fonts where the operating system's font system looks. The
// filesystem effects are injected so this library tier never touches the OS;
// package main backs them with os/io rather than by shelling out.
type Install_Fonts_Input struct {
	// Font_Directory is the absolute destination, already resolved per operating
	// system. Empty means this OS has no known user font directory, so the step
	// does nothing.
	Font_Directory string
	// Font_Present reports whether a font filename already exists in the
	// destination, so each missing face is copied and present ones are left alone.
	Font_Present func(file string) (present bool)
	// Copy_Font copies one vendored font filename into the destination, creating
	// the directory as needed.
	Copy_Font func(file string) (err error)
	// Refresh rebuilds the font cache (Linux's fc-cache), run after the copies. Nil
	// when the OS auto-detects fonts (macOS), so no cache step runs.
	Refresh func() (err error)
	// Stdout receives one line per face naming each copy and each skip.
	Stdout io.Writer
	// Stderr receives a diagnostic line when a copy or the refresh fails.
	Stderr io.Writer
}

// Install_Fonts places the vendored Iosevka faces into the per-OS user font
// directory and, where the OS needs it, refreshes the font cache. It is
// idempotent: a destination already holding the fonts copies nothing and skips
// the refresh, so a repeat bootstrap does no work.
func Install_Fonts(input *Install_Fonts_Input) (status_code int) {
	if input.Font_Directory == "" {
		return 0
	}
	copied := false
	for _, file := range iosevka_font_files() {
		if input.Font_Present(file) {
			fmt.Fprintf(input.Stdout, "setup: fonts: skipped %s (present)\n", file)
			continue
		}
		copy_err := input.Copy_Font(file)
		if copy_err != nil {
			fmt.Fprintf(input.Stderr, "setup: %v\n", copy_err)
			return exit_failure
		}
		fmt.Fprintf(input.Stdout, "setup: fonts: copied %s\n", file)
		copied = true
	}
	if !copied {
		return 0
	}
	if input.Refresh == nil {
		return 0
	}
	refresh_err := input.Refresh()
	if refresh_err != nil {
		fmt.Fprintf(input.Stderr, "setup: %v\n", refresh_err)
		return exit_failure
	}
	return 0
}

// Direnv_version is the release the bootstrap wants — the prefix of `direnv
// --version`. direnv embeds version.txt, so a plain build self-reports it; tracks
// the vendored third_party/direnv source, bump it with the source.
const direnv_version = "2.37.1"

// Install_Direnv_Input carries the injected dependencies Install_Direnv needs to
// build the vendored direnv with the Go toolchain and place it on PATH.
type Install_Direnv_Input struct {
	// Direnv_Directory is the vendored direnv Go module go build compiles, offline
	// against its committed vendor tree.
	Direnv_Directory string
	// Binary_Directory is the directory on PATH direnv is built into; it is both the
	// install location and the gate's probe target.
	Binary_Directory string
	// Shell runs the version gate and the go build.
	Shell *sh.Shell
}

// Install_Direnv builds the vendored direnv straight into the bin directory, where
// the shell hook and every .envrc can find it. It is the bootstrap's first step
// because everything downstream is driven by direnv. It is idempotent: when the
// built direnv already reports the wanted version, it does nothing.
func Install_Direnv(input *Install_Direnv_Input) (status_code int) {
	if input.Direnv_Directory == "" {
		return 0
	}
	if input.Binary_Directory == "" {
		return 0
	}
	if direnv_built(input.Shell, input.Binary_Directory) {
		fmt.Fprintln(input.Shell.Stdout, "setup: direnv installed")
		return 0
	}
	fmt.Fprintln(input.Shell.Stdout, "setup: direnv: building...")
	// Build from inside the module so its vendor tree resolves; GOWORK=off because
	// direnv is a foreign module outside the repo's go.work; CGO_ENABLED=0 for a
	// static binary (no cc); direnv embeds version.txt, so no ldflags are needed.
	destination := filepath.Join(input.Binary_Directory, "direnv")
	build := "cd " + input.Direnv_Directory +
		" && GOWORK=off CGO_ENABLED=0 go build -mod=vendor" +
		" -o " + destination + " ."
	if !sh.Shell_Spawn(input.Shell, "sh", "-c", build) {
		fmt.Fprintln(input.Shell.Stderr, "setup: direnv build failed")
		return exit_failure
	}
	return 0
}

// Reports whether the direnv binary in the bin directory already reports the
// wanted version. Probing that exact path leaves a present build alone.
func direnv_built(shell *sh.Shell, binary_directory string) (built bool) {
	return Installed(&Installed_Input{
		Shell:      shell,
		Executable: filepath.Join(binary_directory, "direnv"),
		Version:    direnv_version,
	})
}

// Rust_version is the toolchain rustup installs and the gate checks. Pinned, not
// "stable", so the idempotency check has a fixed version to match; bump it
// deliberately. `rustc --version` prints "rustc <version> (<commit> <date>)".
const rust_version = "1.96.0"

// Install_Rust_Input carries the injected dependencies Install_Rust needs to
// install the Rust toolchain and expose it on PATH.
type Install_Rust_Input struct {
	// Cargo_Directory is CARGO_HOME, where rustup places cargo, rustc, and rustup
	// under bin/. Empty means CARGO_HOME is not configured, so the step does
	// nothing rather than install into an unknown location.
	Cargo_Directory string
	// Link_Directory is the one directory on PATH the toolchain binaries are
	// symlinked into, so PATH carries a single entry rather than CARGO_HOME/bin as
	// well. Empty disables the step for the same reason as an empty Cargo_Directory.
	Link_Directory string
	// Shell runs the `which` probes that gate the step, the rustup script, and the
	// ln calls. rustup reads CARGO_HOME and RUSTUP_HOME from the Shell's
	// environment, so the toolchain lands under Cargo_Directory without plumbing.
	Shell *sh.Shell
}

// Install_Rust installs the Rust toolchain with rustup and symlinks cargo,
// rustup, and rustc into the PATH directory, so they are reachable without
// CARGO_HOME/bin on PATH. It is idempotent: when all three already resolve inside
// the link directory, it does nothing, so a repeat bootstrap does no work.
func Install_Rust(input *Install_Rust_Input) (status_code int) {
	if input.Cargo_Directory == "" {
		return 0
	}
	if input.Link_Directory == "" {
		return 0
	}
	if rust_installed(input.Shell, input.Cargo_Directory) {
		fmt.Fprintln(input.Shell.Stdout, "setup: rust installed")
	} else {
		fmt.Fprintln(input.Shell.Stdout, "setup: rust: installing...")
		if !sh.Shell_Spawn(input.Shell, rust_install_invocation()...) {
			fmt.Fprintln(input.Shell.Stderr, "setup: rust install failed")
			return exit_failure
		}
	}
	// Always (re)link, even when the install was skipped, so a stale or missing
	// symlink is repointed at the current CARGO_HOME without reinstalling.
	if !rust_link(&rust_link_input{
		Shell:           input.Shell,
		Cargo_Directory: input.Cargo_Directory,
		Link_Directory:  input.Link_Directory,
	}) {
		fmt.Fprintln(input.Shell.Stderr, "setup: rust link failed")
		return exit_failure
	}
	return 0
}

// Returns the invocation that installs rustup non-interactively. -y answers every
// prompt, --no-modify-path leaves the shell rc alone because rust reaches PATH
// through the symlinks, and the toolchain is pinned to the wanted channel.
// RUSTUP_INIT_SKIP_PATH_CHECK silences the "existing Rust" warning those symlinks
// trip. curl --fail/-sSf surfaces a download error instead of piping a half script.
func rust_install_invocation() (arguments []string) {
	return []string{
		"sh", "-c",
		"curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | " +
			"RUSTUP_INIT_SKIP_PATH_CHECK=yes sh -s -- " +
			"-y --no-modify-path --default-toolchain " + rust_version,
	}
}

// Reports whether the pinned rust toolchain is installed at the current
// CARGO_HOME, probing rustc (which carries the toolchain version; cargo and rustup
// install with it). A stale symlink to an old CARGO_HOME or a wrong version does
// not pass, so a moved or mismatched toolchain is reinstalled.
func rust_installed(shell *sh.Shell, cargo_directory string) (installed bool) {
	return Installed(&Installed_Input{
		Shell:      shell,
		Executable: filepath.Join(cargo_directory, "bin", "rustc"),
		Version:    "rustc " + rust_version,
	})
}

// Carries the arguments for symlinking the rust toolchain onto PATH.
type rust_link_input struct {
	Shell           *sh.Shell
	Cargo_Directory string
	Link_Directory  string
}

// Symlinks cargo, rustup, and rustc from CARGO_HOME/bin into the link directory
// with ln -sf, so the managed toolchain is reachable from the one PATH entry and
// any stale link there is overwritten. Reports whether every link succeeded.
func rust_link(input *rust_link_input) (linked bool) {
	for _, tool := range []string{"cargo", "rustup", "rustc"} {
		source := filepath.Join(input.Cargo_Directory, "bin", tool)
		target := filepath.Join(input.Link_Directory, tool)
		if !sh.Shell_Spawn(input.Shell, "ln", "-sf", source, target) {
			return false
		}
	}
	return true
}

// Fish_version is the release the bootstrap wants, the exact line `fish --version`
// prints. It tracks the vendored third_party/fish-shell source; bump it with the
// source. The build is skipped when the installed fish already reports it.
const fish_version = "fish, version 4.7.1"

// Install_Fish_Input carries the injected dependencies Install_Fish needs to build
// the vendored fish with cargo and expose it on PATH.
type Install_Fish_Input struct {
	// Fish_Directory is the vendored fish crate cargo builds with --path; its
	// .cargo/config.toml points the offline build at the committed vendor tree.
	Fish_Directory string
	// Cargo_Directory is CARGO_HOME, where cargo install places the fish binary
	// under bin/, the source the symlink points at.
	Cargo_Directory string
	// Link_Directory is the one directory on PATH fish is symlinked into.
	Link_Directory string
	// Shell runs the `fish --version` gate, the cargo build, and the ln call. cargo
	// — linked onto PATH by the rust step — reads CARGO_HOME from the environment.
	Shell *sh.Shell
}

// Install_Fish builds fish from the vendored source with cargo and symlinks fish,
// fish_indent, and fish_key_reader into the PATH directory. It is idempotent: when
// the built fish already reports the wanted version, the build is skipped and only
// the symlinks are refreshed, so a repeat bootstrap does no heavy work.
func Install_Fish(input *Install_Fish_Input) (status_code int) {
	if input.Fish_Directory == "" {
		return 0
	}
	if input.Cargo_Directory == "" {
		return 0
	}
	if input.Link_Directory == "" {
		return 0
	}
	if fish_built(input.Shell, input.Cargo_Directory) {
		fmt.Fprintln(input.Shell.Stdout, "setup: fish built")
	} else {
		fmt.Fprintln(input.Shell.Stdout, "setup: fish: building...")
		if !sh.Shell_Spawn(input.Shell, fish_install_invocation(input.Fish_Directory)...) {
			fmt.Fprintln(input.Shell.Stderr, "setup: fish build failed")
			return exit_failure
		}
	}
	// Always (re)link, even when the build was skipped, so a missing symlink is
	// restored without recompiling. ln -sf is idempotent. cargo install builds all
	// three fish binaries, so link each.
	for _, binary := range []string{"fish", "fish_indent", "fish_key_reader"} {
		source := filepath.Join(input.Cargo_Directory, "bin", binary)
		target := filepath.Join(input.Link_Directory, binary)
		if !sh.Shell_Spawn(input.Shell, "ln", "-sf", source, target) {
			fmt.Fprintln(input.Shell.Stderr, "setup: fish link failed")
			return exit_failure
		}
	}
	return 0
}

// Returns the invocation that builds and installs fish from the vendored crate. cd
// into the crate so its .cargo/config.toml maps every source to the committed
// vendor tree; --offline/--locked then build with no network. RUSTFLAGS is fish's
// documented static-crt build flag.
func fish_install_invocation(fish_directory string) (arguments []string) {
	return []string{
		"sh", "-c",
		"cd " + fish_directory + " && " +
			"RUSTFLAGS='-C target-feature=+crt-static' " +
			"cargo install --offline --locked --path .",
	}
}

// Reports whether the fish binary already installed under CARGO_HOME reports the
// wanted version. Probing the exact path, not PATH, keeps a missing symlink from
// triggering a needless multi-minute recompile of an existing build.
func fish_built(shell *sh.Shell, cargo_directory string) (built bool) {
	return Installed(&Installed_Input{
		Shell:      shell,
		Executable: filepath.Join(cargo_directory, "bin", "fish"),
		Version:    fish_version,
	})
}

// Fzf_version is the release the bootstrap wants, the exact line `fzf --version`
// prints. It tracks the vendored third_party/fzf source; bump it with the source.
// The build injects it via ldflags so the binary self-reports this string.
const fzf_version = "0.73.1"

// Install_Fzf_Input carries the injected dependencies Install_Fzf needs to build
// the vendored fzf with the Go toolchain and place it on PATH.
type Install_Fzf_Input struct {
	// Fzf_Directory is the vendored fzf Go module go build compiles, offline
	// against its committed vendor tree.
	Fzf_Directory string
	// Binary_Directory is the directory on PATH the fzf binary is built into; it is
	// both the install location and the gate's probe target.
	Binary_Directory string
	// Shell runs the version gate and the go build.
	Shell *sh.Shell
}

// Install_Fzf builds fzf from the vendored source straight into the bin directory.
// It is idempotent: when the built fzf already reports the wanted version, it does
// nothing, so a repeat bootstrap does no work. fzf is a Go binary, so the build
// output is the install — no separate copy or symlink.
func Install_Fzf(input *Install_Fzf_Input) (status_code int) {
	if input.Fzf_Directory == "" {
		return 0
	}
	if input.Binary_Directory == "" {
		return 0
	}
	if fzf_built(input.Shell, input.Binary_Directory) {
		fmt.Fprintln(input.Shell.Stdout, "setup: fzf installed")
		return 0
	}
	fmt.Fprintln(input.Shell.Stdout, "setup: fzf: building...")
	// Build from inside the module so its vendor tree resolves; GOWORK=off because
	// fzf is a foreign module outside the repo's go.work; ldflags inject the version
	// so the binary self-reports it; output lands straight in the bin directory.
	destination := filepath.Join(input.Binary_Directory, "fzf")
	build := "cd " + input.Fzf_Directory +
		" && GOWORK=off go build -mod=vendor" +
		" -ldflags '-s -w -X main.version=" + fzf_version + " -X main.revision='" +
		" -o " + destination + " ."
	if !sh.Shell_Spawn(input.Shell, "sh", "-c", build) {
		fmt.Fprintln(input.Shell.Stderr, "setup: fzf build failed")
		return exit_failure
	}
	return 0
}

// Reports whether the fzf binary in the bin directory already reports the wanted
// version. Probing that exact path leaves a present build alone.
func fzf_built(shell *sh.Shell, binary_directory string) (built bool) {
	return Installed(&Installed_Input{
		Shell:      shell,
		Executable: filepath.Join(binary_directory, "fzf"),
		Version:    fzf_version,
	})
}

// Jj_version is the release the bootstrap wants — the prefix of `jj --version`.
// jj's build.rs appends a commit hash, so the gate prefix-matches; tracks the
// vendored third_party/jj workspace version, bump it with the source.
const jj_version = "jj 0.42.0"

// Install_Jj_Input carries the injected dependencies Install_Jj needs to build the
// vendored jj with cargo and expose it on PATH.
type Install_Jj_Input struct {
	// Jj_Directory is the vendored jj workspace root cargo builds from; its
	// .cargo/config.toml points the offline build at the committed vendor tree.
	Jj_Directory string
	// Cargo_Directory is CARGO_HOME, where cargo install places the jj binary under
	// bin/, the source the symlink points at.
	Cargo_Directory string
	// Link_Directory is the one directory on PATH jj is symlinked into.
	Link_Directory string
	// Shell runs the `jj --version` gate, the cargo build, and the ln call.
	Shell *sh.Shell
}

// Install_Jj builds jj from the vendored workspace with cargo and symlinks the
// binary into the PATH directory. It is idempotent: when the built jj already
// reports the wanted version, the build is skipped and only the symlink is
// refreshed, so a repeat bootstrap does no heavy work.
func Install_Jj(input *Install_Jj_Input) (status_code int) {
	if input.Jj_Directory == "" {
		return 0
	}
	if input.Cargo_Directory == "" {
		return 0
	}
	if input.Link_Directory == "" {
		return 0
	}
	if jj_built(input.Shell, input.Cargo_Directory) {
		fmt.Fprintln(input.Shell.Stdout, "setup: jj built")
	} else {
		fmt.Fprintln(input.Shell.Stdout, "setup: jj: building...")
		if !sh.Shell_Spawn(input.Shell, jj_install_invocation(input.Jj_Directory)...) {
			fmt.Fprintln(input.Shell.Stderr, "setup: jj build failed")
			return exit_failure
		}
	}
	// Always (re)link, even when the build was skipped, so a missing symlink is
	// restored without recompiling. ln -sf is idempotent.
	source := filepath.Join(input.Cargo_Directory, "bin", "jj")
	target := filepath.Join(input.Link_Directory, "jj")
	if !sh.Shell_Spawn(input.Shell, "ln", "-sf", source, target) {
		fmt.Fprintln(input.Shell.Stderr, "setup: jj link failed")
		return exit_failure
	}
	return 0
}

// Returns the invocation that builds and installs jj from the vendored workspace.
// cd into the workspace root so its .cargo/config.toml maps crates to the vendor
// tree; --offline/--locked build with no network; --bin jj --path cli installs
// only the jj binary from the jj-cli package, not its test helpers.
func jj_install_invocation(jj_directory string) (arguments []string) {
	return []string{
		"sh", "-c",
		"cd " + jj_directory + " && " +
			"cargo install --offline --locked --bin jj --path cli",
	}
}

// Reports whether the jj binary already installed under CARGO_HOME reports the
// wanted version. Probing the exact path, not PATH, keeps a missing symlink from
// triggering a needless multi-minute recompile of an existing build.
func jj_built(shell *sh.Shell, cargo_directory string) (built bool) {
	return Installed(&Installed_Input{
		Shell:      shell,
		Executable: filepath.Join(cargo_directory, "bin", "jj"),
		Version:    jj_version,
	})
}

// Ripgrep_version is the release the bootstrap wants — the prefix of `rg
// --version`. ripgrep's build.rs appends a git rev, so the gate prefix-matches;
// tracks the vendored third_party/ripgrep Cargo.toml version, bump with it.
const ripgrep_version = "ripgrep 15.1.0"

// Install_Ripgrep_Input carries the injected dependencies Install_Ripgrep needs to
// build the vendored ripgrep with cargo and expose its rg binary on PATH.
type Install_Ripgrep_Input struct {
	// Ripgrep_Directory is the vendored ripgrep crate cargo builds from; its
	// .cargo/config.toml points the offline build at the committed vendor tree.
	Ripgrep_Directory string
	// Cargo_Directory is CARGO_HOME, where cargo install places the rg binary under
	// bin/, the source the symlink points at.
	Cargo_Directory string
	// Link_Directory is the one directory on PATH rg is symlinked into.
	Link_Directory string
	// Shell runs the `rg --version` gate, the cargo build, and the ln call.
	Shell *sh.Shell
}

// Install_Ripgrep builds ripgrep from the vendored crate with cargo and symlinks
// the rg binary into the PATH directory. It is idempotent: when the built rg
// already reports the wanted version, the build is skipped and only the symlink is
// refreshed, so a repeat bootstrap does no heavy work.
func Install_Ripgrep(input *Install_Ripgrep_Input) (status_code int) {
	if input.Ripgrep_Directory == "" {
		return 0
	}
	if input.Cargo_Directory == "" {
		return 0
	}
	if input.Link_Directory == "" {
		return 0
	}
	if ripgrep_built(input.Shell, input.Cargo_Directory) {
		fmt.Fprintln(input.Shell.Stdout, "setup: ripgrep built")
	} else {
		fmt.Fprintln(input.Shell.Stdout, "setup: ripgrep: building...")
		invocation := ripgrep_install_invocation(input.Ripgrep_Directory)
		if !sh.Shell_Spawn(input.Shell, invocation...) {
			fmt.Fprintln(input.Shell.Stderr, "setup: ripgrep build failed")
			return exit_failure
		}
	}
	// Always (re)link, even when the build was skipped, so a missing symlink is
	// restored without recompiling. ln -sf is idempotent.
	source := filepath.Join(input.Cargo_Directory, "bin", "rg")
	target := filepath.Join(input.Link_Directory, "rg")
	if !sh.Shell_Spawn(input.Shell, "ln", "-sf", source, target) {
		fmt.Fprintln(input.Shell.Stderr, "setup: ripgrep link failed")
		return exit_failure
	}
	return 0
}

// Returns the invocation that builds and installs rg from the vendored crate. cd
// into the crate so its .cargo/config.toml maps crates to the vendor tree;
// --offline/--locked build with no network; --features pcre2 enables the
// look-around/backreference engine, which is otherwise off by default.
func ripgrep_install_invocation(ripgrep_directory string) (arguments []string) {
	return []string{
		"sh", "-c",
		"cd " + ripgrep_directory + " && " +
			"cargo install --offline --locked --features pcre2 --bin rg --path .",
	}
}

// Reports whether the rg binary already installed under CARGO_HOME reports the
// wanted version. Probing the exact path, not PATH, keeps a missing symlink from
// triggering a needless multi-minute recompile of an existing build.
func ripgrep_built(shell *sh.Shell, cargo_directory string) (built bool) {
	return Installed(&Installed_Input{
		Shell:      shell,
		Executable: filepath.Join(cargo_directory, "bin", "rg"),
		Version:    ripgrep_version,
	})
}

// Fd_version is the release the bootstrap wants — the prefix of `fd --version`.
// Tracks the vendored third_party/fd Cargo.toml version, bump it with the source.
const fdcli_version = "fd 10.4.2"

// Install_Fdcli_Input carries the injected dependencies Install_Fdcli needs to build the
// vendored fd with cargo and expose it on PATH.
type Install_Fdcli_Input struct {
	// Fdcli_Directory is the vendored fd crate cargo builds from; its
	// .cargo/config.toml points the offline build at the committed vendor tree.
	Fdcli_Directory string
	// Cargo_Directory is CARGO_HOME, where cargo install places the fd binary under
	// bin/, the source the symlink points at.
	Cargo_Directory string
	// Link_Directory is the one directory on PATH fd is symlinked into.
	Link_Directory string
	// Shell runs the `fd --version` gate, the cargo build, and the ln call.
	Shell *sh.Shell
}

// Install_Fdcli builds fd from the vendored crate with cargo and symlinks the binary
// into the PATH directory. It is idempotent: when the built fd already reports the
// wanted version, the build is skipped and only the symlink is refreshed, so a
// repeat bootstrap does no heavy work.
func Install_Fdcli(input *Install_Fdcli_Input) (status_code int) {
	if input.Fdcli_Directory == "" {
		return 0
	}
	if input.Cargo_Directory == "" {
		return 0
	}
	if input.Link_Directory == "" {
		return 0
	}
	if fdcli_built(input.Shell, input.Cargo_Directory) {
		fmt.Fprintln(input.Shell.Stdout, "setup: fd built")
	} else {
		fmt.Fprintln(input.Shell.Stdout, "setup: fd: building...")
		invocation := fdcli_install_invocation(input.Fdcli_Directory)
		if !sh.Shell_Spawn(input.Shell, invocation...) {
			fmt.Fprintln(input.Shell.Stderr, "setup: fd build failed")
			return exit_failure
		}
	}
	// Always (re)link, even when the build was skipped, so a missing symlink is
	// restored without recompiling. ln -sf is idempotent.
	source := filepath.Join(input.Cargo_Directory, "bin", "fd")
	target := filepath.Join(input.Link_Directory, "fd")
	if !sh.Shell_Spawn(input.Shell, "ln", "-sf", source, target) {
		fmt.Fprintln(input.Shell.Stderr, "setup: fd link failed")
		return exit_failure
	}
	return 0
}

// Returns the invocation that builds and installs fd from the vendored crate. cd
// into the crate so its .cargo/config.toml maps crates to the vendor tree;
// --offline/--locked build with no network against the default features.
func fdcli_install_invocation(fdcli_directory string) (arguments []string) {
	return []string{
		"sh", "-c",
		"cd " + fdcli_directory + " && " +
			"cargo install --offline --locked --bin fd --path .",
	}
}

// Reports whether the fd binary already installed under CARGO_HOME reports the
// wanted version. Probing the exact path, not PATH, keeps a missing symlink from
// triggering a needless multi-minute recompile of an existing build.
func fdcli_built(shell *sh.Shell, cargo_directory string) (built bool) {
	return Installed(&Installed_Input{
		Shell:      shell,
		Executable: filepath.Join(cargo_directory, "bin", "fd"),
		Version:    fdcli_version,
	})
}

// Ghostty_version is the release the bootstrap wants — the prefix of `ghostty
// --version`, whose first line reads "Ghostty <version>". Tracks the pinned DMG
// below; bump it, the url, and the SHA256 together.
const ghostty_version = "Ghostty 1.3.1"

// Ghostty_dmg_url is the pinned macOS DMG the install downloads. Ghostty is a
// notarized app bundle, not buildable source, so it is fetched rather than vendored
// and built; the version lives in the path, so the url always serves that build.
const ghostty_dmg_url = "https://release.files.ghostty.org/1.3.1/Ghostty.dmg"

// Ghostty_dmg_sha256 is the SHA256 the download is verified against, so a corrupt
// or tampered DMG aborts the install instead of placing bad bytes in the
// applications directory. The published hash for the 1.3.1 DMG; bump it with the url.
const ghostty_dmg_sha256 = "18cff2b0a6cee90eead9c7d3064e808a252a40baf214aa752c1ecb793b8f5f69"

// Ghostty_team_identifier is the Apple Developer Team ID Ghostty is signed under,
// pinned in the signature gate so an app signed by anyone else is rejected. Tied to
// the developer's account and stable for years; bump it only if that identity changes.
const ghostty_team_identifier = "24VZTF6M5V"

// Ghostty_application_binary_subpath locates the app's command-line binary inside
// the bundle, relative to the applications directory. It is both the gate's probe
// target and the source the PATH symlink points at.
const ghostty_application_binary_subpath = "Ghostty.app/Contents/MacOS/ghostty"

// Install_Ghostty_Input carries the injected dependencies Install_Ghostty needs to
// download the pinned Ghostty DMG, install the app, and expose its CLI on PATH.
type Install_Ghostty_Input struct {
	// Applications_Directory is the macOS app directory Ghostty.app installs into and
	// the gate probes under. Empty — every non-darwin OS, which has no DMG to install
	// — makes the step do nothing, the same skip an unknown font directory triggers.
	Applications_Directory string
	// Link_Directory is the one directory on PATH the app's ghostty CLI is symlinked
	// into, so `ghostty` works in a terminal as well as via the app. Empty disables
	// the step.
	Link_Directory string
	// Shell runs the version gate, the download-and-install script, and the ln call.
	Shell *sh.Shell
}

// Install_Ghostty installs Ghostty from its pinned DMG into the applications
// directory and symlinks the app's CLI into the PATH directory. It is idempotent:
// when the installed app already reports the wanted version, the download is skipped
// and only the symlink is refreshed, so a repeat bootstrap does no network work.
func Install_Ghostty(input *Install_Ghostty_Input) (status_code int) {
	if input.Applications_Directory == "" {
		return 0
	}
	if input.Link_Directory == "" {
		return 0
	}
	if ghostty_installed(input.Shell, input.Applications_Directory) {
		fmt.Fprintln(input.Shell.Stdout, "setup: ghostty installed")
	} else {
		fmt.Fprintln(input.Shell.Stdout, "setup: ghostty: installing...")
		invocation := ghostty_install_invocation(input.Applications_Directory)
		if !sh.Shell_Spawn(input.Shell, invocation...) {
			fmt.Fprintln(input.Shell.Stderr, "setup: ghostty install failed")
			return exit_failure
		}
	}
	// Always (re)link, even when the install was skipped, so a missing symlink is
	// restored without re-downloading. ln -sf is idempotent.
	source := filepath.Join(input.Applications_Directory, ghostty_application_binary_subpath)
	target := filepath.Join(input.Link_Directory, "ghostty")
	if !sh.Shell_Spawn(input.Shell, "ln", "-sf", source, target) {
		fmt.Fprintln(input.Shell.Stderr, "setup: ghostty link failed")
		return exit_failure
	}
	return 0
}

// Ghostty_install_script downloads the pinned DMG, verifies its SHA256, mounts it,
// and replaces the app bundle. %[1]s is the url, %[2]s the SHA256, %[3]s the
// applications directory. set -e aborts on the first failure — a SHA256 mismatch
// included — so a tampered or truncated download never reaches the applications
// directory; the trap detaches the volume and clears the scratch dir on every exit.
const ghostty_install_script = `set -e
work=$(mktemp -d)
trap 'hdiutil detach -quiet "$work/mnt" 2>/dev/null; rm -rf "$work"' EXIT
curl --proto '=https' --tlsv1.2 -fsSL -o "$work/Ghostty.dmg" %[1]s
echo "%[2]s  $work/Ghostty.dmg" | shasum -a 256 -c -
hdiutil attach -nobrowse -quiet -mountpoint "$work/mnt" "$work/Ghostty.dmg"
rm -rf %[3]s/Ghostty.app
cp -R "$work/mnt/Ghostty.app" %[3]s/
`

// Returns the invocation that installs Ghostty from the pinned DMG, the script run
// through sh so the dynamic mount point and the cleanup trap have a shell.
func ghostty_install_invocation(applications_directory string) (arguments []string) {
	script := fmt.Sprintf(
		ghostty_install_script,
		ghostty_dmg_url,
		ghostty_dmg_sha256,
		applications_directory,
	)
	return []string{"sh", "-c", script}
}

// Reports whether the installed Ghostty app is the wanted version AND still carries
// a verifying code signature. Probing the app's own binary, not PATH, keeps a
// missing symlink from forcing a needless re-download of an app already in place.
func ghostty_installed(shell *sh.Shell, applications_directory string) (installed bool) {
	binary := filepath.Join(applications_directory, ghostty_application_binary_subpath)
	version_present := Installed(&Installed_Input{
		Shell:      shell,
		Executable: binary,
		Version:    ghostty_version,
	})
	if !version_present {
		return false
	}
	// A matching version alone is not enough: an app whose binary was swapped or
	// re-signed by another developer still reports 1.3.1, so the install counts only
	// if codesign verifies the bundle against Ghostty's signing identity. codesign is
	// offline and deterministic, so a failure means a tampered or foreign bundle.
	application := filepath.Join(applications_directory, "Ghostty.app")
	return sh.Shell_Spawn(shell, ghostty_codesign_invocation(application)...)
}

// Returns the codesign invocation that verifies the bundle and pins it to Ghostty's
// signing Team ID, so a tampered app or one validly signed by another developer
// fails. The -R requirement's leading = marks it as requirement text, not a file.
func ghostty_codesign_invocation(application string) (arguments []string) {
	requirement := `=anchor apple generic and certificate leaf[subject.OU] = "` +
		ghostty_team_identifier + `"`
	return []string{
		"codesign", "--verify", "--deep", "--strict", "-R", requirement, application,
	}
}
