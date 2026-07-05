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
	"path/filepath"
	"slices"
	"strings"

	sysio "local/james-orcales/shared/io"
	"local/james-orcales/shared/jlog"
	systime "local/james-orcales/shared/time"
)

// Dotfile_bytes_max bounds a single dotfile read into one fixed buffer. 1 MiB
// dwarfs any real configuration file yet caps memory against a pathological
// input, satisfying the linter's unbounded-read ban.
const dotfile_bytes_max = 1048576

// Exit_failure is the process exit code for a planning or write failure during
// an otherwise well-formed run.
const exit_failure = 1

// File_System is the injected filesystem capability the sync runs on: the shared/io loop it
// submits reads, writes, and traversal to, and the Run_Until pump that drives each submitted
// op to completion. package main backs Run_Until with the real driver (the loop is ticked
// only there) and the simulation with the sim driver; this tier submits but never drives.
type File_System struct {
	// Loop is the submit surface for the file ops: Open/Read/Write/Close are pumped, while
	// Read_Directory/Status/Make_Directory return inline.
	Loop sysio.IO
	// Run_Until drives the loop until a submitted op reports done, capped by timeout
	// (sysio.FOREVER waits unbounded) — the per-op pump; returns whether the op completed.
	Run_Until func(done func() (finished bool), timeout systime.Duration) (completed bool)
}

// Main_Input carries the injected dependencies Main needs to sync dotfiles.
type Main_Input struct {
	// File_System is the loop the dotfiles tree is walked, read, and written through.
	File_System File_System
	// Source_Directory is the absolute dotfiles tree walked in full from its root.
	Source_Directory string
	// Destination_Directory is the absolute home directory writes land under, and the diff
	// reads existing files from.
	Destination_Directory string
	// Operating_System gates the macos defaults step, which runs only on
	// "darwin" (a runtime.GOOS value).
	Operating_System string
	// Run_Command runs an external program by name with arguments. It is injected
	// so this library tier never executes anything; package main supplies the
	// exec-backed runner used for the macos defaults.
	Run_Command func(name string, arguments []string) (err error)
	// Is_Ignored classifies a batch of source paths, returning the set that is
	// gitignored; it is threaded to Plan so the install tree under .local is not
	// mirrored into home. Batched because the gitignore probe is a subprocess and one
	// spawn per path made a large tree scan for seconds — one call classifies a whole
	// tree level. Nil ignores nothing.
	Is_Ignored func(relative_paths []string) (ignored map[string]bool)
	// Logger records the sync's narration: one line per file written, the scan's
	// progress, and a diagnostic when planning or a write fails. The zero Logger is a
	// disabled no-op, so a caller wanting silence passes none.
	Logger jlog.Logger
}

// Main syncs the source dotfiles into the home directory and, on darwin, applies
// the macos defaults, returning a process exit code. It is the binary's single
// entry point, kept here so package main stays a thin, untested shell.
func Main(input *Main_Input) (status_code int) {
	writes, plan_err := Plan(&Plan_Input{
		File_System:           input.File_System,
		Source_Directory:      input.Source_Directory,
		Destination_Directory: input.Destination_Directory,
		Is_Ignored:            input.Is_Ignored,
		Logger:                input.Logger,
	})
	if plan_err != nil {
		jlog.Logger_Error(input.Logger, "plan failed", jlog.Err(plan_err))
		return exit_failure
	}
	for _, write := range writes {
		write_err := write_file(&input.File_System, write.Destination_Path, write.Contents)
		if write_err != nil {
			jlog.Logger_Error(input.Logger, "write failed", jlog.Err(write_err))
			return exit_failure
		}
		jlog.Logger_Info(input.Logger, "wrote", jlog.String("path", write.Destination_Path))
	}
	// A converged tree writes nothing, so without a closing line the step would look
	// stuck after the last scan line; say it finished and had no work.
	if len(writes) == 0 {
		jlog.Logger_Info(input.Logger, "dotfiles up to date")
	}
	// The macos defaults touch macOS-only preference domains, so they run there
	// and nowhere else.
	if input.Operating_System != "darwin" {
		return 0
	}
	return apply_macos_defaults(input.Run_Command, input.Logger)
}

// Takes just the runner and the logger it uses, not the whole Main_Input: a
// function whose parameter is *Main_Input would be forced to be named Main. Runs
// every macos defaults command, stopping at the first that fails. It logs
// nothing per command — 35 lines of `defaults write` is noise, not progress.
func apply_macos_defaults(
	run func(name string, arguments []string) (err error), logger jlog.Logger,
) (status_code int) {
	for _, command := range macos_commands() {
		run_err := run(command.Name, command.Arguments)
		if run_err != nil {
			jlog.Logger_Error(logger, "macos defaults failed", jlog.Err(run_err))
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

// Bootstrap_Input carries the ordered steps and the logger their progress is
// announced to.
type Bootstrap_Input struct {
	// Steps run in slice order; the first to return non-zero stops the rest.
	Steps []Step
	// Logger records one line naming each step as it starts. The zero Logger is a
	// disabled no-op.
	Logger jlog.Logger
}

// Bootstrap runs the setup steps in their fixed order of operations, announcing
// each by name before it runs and returning the first non-zero status — skipping
// the rest — so a cheap early failure surfaces before later, heavier work.
// Package main supplies the steps.
func Bootstrap(input *Bootstrap_Input) (status_code int) {
	for _, step := range input.Steps {
		jlog.Logger_Info(input.Logger, "step", jlog.String("name", step.Name))
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
	// File_System is the loop the source tree is walked and read through, and the
	// destination read through for diffing.
	File_System File_System
	// Source_Directory is the absolute dotfiles tree, walked in full from its root.
	Source_Directory string
	// Destination_Directory is the absolute home directory the relative source
	// paths are mirrored under to form each write's Destination_Path.
	Destination_Directory string
	// Is_Ignored classifies a batch of source paths, each relative to the source root,
	// returning the set that is gitignored. An ignored file is not synced and an ignored
	// directory is pruned, so the generated install tree under .local never reaches the
	// home directory. It takes a batch, not one path, because the gitignore probe is a
	// subprocess: the walk hands it a whole tree level at once so the scan spawns a
	// process per depth, not per entry. Nil ignores nothing, the sync's behavior before
	// the filter existed.
	Is_Ignored func(relative_paths []string) (ignored map[string]bool)
	// Logger records one line naming each directory as the walk reads it. A converged
	// scan writes nothing, so without this the step looks hung while it walks; narrating
	// each directory proves the scan is live. The zero Logger is a disabled no-op, so a
	// direct caller that wants no narration pays nothing.
	Logger jlog.Logger
}

// Plan returns the writes that would bring the home directory in line with the source
// dotfiles: every regular file under the source tree, each emitted only when the destination
// is missing or its contents differ. It walks the tree one level at a time through the loop's
// Read_Directory, classifying each level's entries with a single Is_Ignored batch — so the
// gitignore probe is one subprocess per depth, not per entry — and pruning an ignored
// directory so the install tree under .local is never descended into.
func Plan(input *Plan_Input) (writes []File_Write, err error) {
	writes = []File_Write{}
	level := []string{"."}
	for len(level) > 0 {
		entries, read_err := plan_read_level(input, level)
		if read_err != nil {
			return nil, read_err
		}
		// One probe classifies the whole level: the batch is what keeps a large tree from
		// spawning a gitignore process per entry.
		ignored := plan_ignored(input.Is_Ignored, entries)
		next := []string{}
		for _, entry := range entries {
			if ignored[entry.Relative] {
				continue
			}
			if entry.Is_Directory {
				next = append(next, entry.Relative)
				continue
			}
			write, planned, plan_err := plan_file(input, entry.Relative)
			if plan_err != nil {
				return nil, plan_err
			}
			if planned {
				writes = append(writes, write)
			}
		}
		level = next
	}
	return writes, nil
}

// One child seen during the walk: its path relative to the source root and whether it is a
// directory, the two facts the level walk needs to prune and recurse.
type plan_entry struct {
	// Relative is the child's path from the source root, the key Is_Ignored classifies it by.
	Relative string
	// Is_Directory reports whether the child is a directory, so the walk knows to descend.
	Is_Directory bool
}

// Reads every directory in one level of the walk, returning all their children as a single
// batch of entries — the unit Is_Ignored classifies at once. Narrates each directory as it
// reads it, so the scan shows progress even when it ultimately writes nothing.
func plan_read_level(input *Plan_Input, level []string) (entries []plan_entry, err error) {
	entries = []plan_entry{}
	for _, directory := range level {
		plan_narrate(input.Logger, directory)
		read, read_err := input.File_System.Loop.Read_Directory(
			filepath.Join(input.Source_Directory, directory))
		if read_err != nil {
			return nil, read_err
		}
		for _, child := range read {
			entries = append(entries, plan_entry{
				Relative:     filepath.Join(directory, child.Name),
				Is_Directory: child.Is_Directory,
			})
		}
	}
	return entries, nil
}

// Announces the directory the walk is about to read, one line each, so a scan that
// writes nothing still shows it is advancing. The scan is chatter — one line per
// directory — so it logs at debug, below the progress the install steps report.
func plan_narrate(logger jlog.Logger, directory string) {
	jlog.Logger_Debug(logger, "scanning", jlog.String("dir", directory))
}

// Returns the set of a level's entries that are gitignored, classifying them all in one
// Is_Ignored call. A nil predicate — or an empty level — ignores nothing, so the filter
// stays opt-in and an empty level spawns no probe.
func plan_ignored(
	is_ignored func(relative_paths []string) (ignored map[string]bool), entries []plan_entry,
) (ignored map[string]bool) {
	if is_ignored == nil {
		return nil
	}
	if len(entries) == 0 {
		return nil
	}
	paths := make([]string, len(entries))
	for index, entry := range entries {
		paths[index] = entry.Relative
	}
	return is_ignored(paths)
}

// Decides whether the source file at relative needs syncing. planned is false when the
// source is absent or the destination already holds identical bytes; otherwise it returns
// the write mirroring the relative path under the home directory.
func plan_file(input *Plan_Input, relative string) (write File_Write, planned bool, err error) {
	source_contents, found, read_err := read_bounded(
		&input.File_System, filepath.Join(input.Source_Directory, relative))
	if read_err != nil {
		return File_Write{}, false, read_err
	}
	if !found {
		return File_Write{}, false, nil
	}
	destination_path := filepath.Join(input.Destination_Directory, relative)
	if destination_matches(&input.File_System, destination_path, source_contents) {
		return File_Write{}, false, nil
	}
	return File_Write{Destination_Path: destination_path, Contents: source_contents}, true, nil
}

// Reports whether the home directory already holds exactly source_contents at path. A
// destination that is absent or unreadable counts as a mismatch, so the file is written.
func destination_matches(
	system *File_System, path string, source_contents []byte,
) (matches bool) {
	destination_contents, found, err := read_bounded(system, path)
	if err != nil {
		return false
	}
	if !found {
		return false
	}
	return bytes.Equal(source_contents, destination_contents)
}

// Reads the file at path in full through the loop, bounded by dotfile_bytes_max into one
// fixed buffer. found is false when the path is absent — the caller treats that as a
// mismatch — and a file overflowing the cap errors, so a truncated dotfile never passes.
func read_bounded(system *File_System, path string) (contents []byte, found bool, err error) {
	file, open_err := system.Loop.Open(path)
	if open_err != nil {
		return nil, false, nil
	}
	buffer := make([]byte, dotfile_bytes_max)
	total := 0
	for total < len(buffer) {
		count, read_err := loop_read(system, file, buffer[total:], int64(total))
		if read_err != nil {
			loop_close(system, file)
			return nil, false, read_err
		}
		total += count
		if count == 0 {
			loop_close(system, file)
			return buffer[:total], true, nil
		}
	}
	loop_close(system, file)
	return nil, false, errors.New("dotfile exceeds the maximum size")
}

// Writes contents to path through the loop, creating its parent directories first. It is the
// filesystem binding the sync writes through, replacing the injected os writer.
func write_file(system *File_System, path string, contents []byte) (err error) {
	mkdir_err := system.Loop.Make_Directory(filepath.Dir(path))
	if mkdir_err != nil {
		return mkdir_err
	}
	file, create_err := system.Loop.Create(path)
	if create_err != nil {
		return create_err
	}
	write_err := loop_write(system, file, contents, 0)
	loop_close(system, file)
	return write_err
}

// Reads up to len(buffer) bytes from file at offset through the loop, driving the submitted
// read to completion and returning its byte count.
func loop_read(
	system *File_System, file sysio.File, buffer []byte, offset int64,
) (count int, err error) {
	var completion sysio.Completion
	done := false
	system.Loop.Read(&completion, func(_ *sysio.Completion, read int, read_err error) {
		count = read
		err = read_err
		done = true
	}, file, buffer, offset)
	system.Run_Until(func() (finished bool) { return done }, sysio.FOREVER)
	return count, err
}

// Writes buffer to file at offset through the loop, driving the submitted write to completion.
func loop_write(
	system *File_System, file sysio.File, buffer []byte, offset int64,
) (err error) {
	var completion sysio.Completion
	done := false
	system.Loop.Write(&completion, func(_ *sysio.Completion, _ int, write_err error) {
		err = write_err
		done = true
	}, file, buffer, offset)
	system.Run_Until(func() (finished bool) { return done }, sysio.FOREVER)
	return err
}

// Closes file through the loop, driving the submitted close to completion.
func loop_close(system *File_System, file sysio.File) {
	var completion sysio.Completion
	done := false
	system.Loop.Close(&completion, func(_ *sysio.Completion, _ error) { done = true }, file)
	system.Run_Until(func() (finished bool) { return done }, sysio.FOREVER)
}

// Spawn runs one command to completion and returns its outcome — the synchronous adapter
// over the shared/io loop's async Spawn. package main backs it with a Run_Until pump (the
// loop is ticked only there) and tests with a recording fake, so this library tier submits
// work but never drives the loop and spawns nothing itself.
type Spawn func(request sysio.Process_Request) (result sysio.Process_Result)

// Shell is the injected subprocess capability the install steps run commands through: Spawn
// executes one command through the loop, the two sinks receive a build's streamed output, and
// Logger records setup's own narration of the step. A thin carrier over the loop, not a revival
// of a shell package — the library binds no process itself.
type Shell struct {
	// Spawn runs a command to completion and returns its outcome.
	Spawn Spawn
	// Stdout receives a build's streamed standard output.
	Stdout io.Writer
	// Stderr receives a build's streamed standard error.
	Stderr io.Writer
	// Logger records setup's narration of the install step — the "building"/"installed"
	// progress and a diagnostic on failure. The zero Logger is a disabled no-op.
	Logger jlog.Logger
}

// Runs path with arguments and returns its captured standard output, trimmed — the version
// probes' one use. It passes no sink, so the loop captures the output for parsing rather than
// streaming it. Trimming strips the trailing newline `which` appends, so a resolved path is
// usable as the next probe's executable.
func run_pipe(shell Shell, path string, arguments ...string) (output string) {
	result := shell.Spawn(sysio.Process_Request{Path: path, Arguments: arguments})
	return strings.TrimSpace(string(result.Output))
}

// Runs the command named by the first argument and reports success, streaming the process's
// output live to the shell's sinks so a multi-minute build's progress reaches the user as it
// happens rather than in one burst at the end.
func run_spawn(shell Shell, arguments ...string) (ok bool) {
	return shell.Spawn(sysio.Process_Request{
		Path: arguments[0], Arguments: arguments[1:],
		Stdout: shell.Stdout, Stderr: shell.Stderr,
	}).Exit == 0
}

// Installed_Input names the binary an install step probes and the version it must
// report. Two string fields, so it is a struct rather than two parameters.
type Installed_Input struct {
	// Shell runs the version probe.
	Shell Shell
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
	version := run_pipe(input.Shell, input.Executable, "--version")
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
	// Shell runs make; its sinks receive setup's narration and the streamed make output.
	Shell Shell
}

// Install_Neovim builds the vendored Neovim and installs it under the local
// prefix, where it lands at local/bin/nvim — already on PATH — and finds its own
// runtime. Every step is idempotent: make rebuilds only what changed and install
// re-copies, so a repeated bootstrap converges without error.
func Install_Neovim(input *Install_Neovim_Input) (status_code int) {
	// Building Neovim is the expensive step, so it is gated on the checkout's own
	// nvim not already being installed: a bootstrap that has it does no work.
	if neovim_already_installed(input.Shell, input.Repository_Directory) {
		jlog.Logger_Info(input.Shell.Logger, "neovim already installed",
			jlog.String("version", neovim_version))
		return 0
	}
	for index, arguments := range neovim_make_invocations(input.Repository_Directory) {
		jlog.Logger_Info(input.Shell.Logger, neovim_make_phase(index))
		// The first argument is the executable; make's output streams to the sinks, so
		// a generic line is all setup adds.
		if !run_spawn(input.Shell, arguments...) {
			jlog.Logger_Error(input.Shell.Logger, "neovim build failed")
			return exit_failure
		}
	}
	return 0
}

// Returns the progress message for make invocation index: invocation 0 configures
// and builds against the prefix, invocation 1 installs.
func neovim_make_phase(index int) (phase string) {
	if index == 0 {
		return "configuring neovim prefix"
	}
	return "installing neovim"
}

// Returns the two make invocations the build runs in order: configure-and-build,
// then install. Each begins with the executable, "make", because run_spawn
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
func neovim_already_installed(shell Shell, repository_directory string) (installed bool) {
	executable := run_pipe(shell, "which", "nvim")
	if !strings.HasPrefix(executable, repository_directory+"/") {
		return false
	}
	return neovim_version_present(run_pipe(shell, executable, "--version"))
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
	// Logger records one line per face naming each copy and each skip, and a diagnostic
	// when a copy or the refresh fails. The zero Logger is a disabled no-op.
	Logger jlog.Logger
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
			jlog.Logger_Info(input.Logger, "font present, skipped",
				jlog.String("file", file))
			continue
		}
		copy_err := input.Copy_Font(file)
		if copy_err != nil {
			jlog.Logger_Error(input.Logger, "font copy failed", jlog.Err(copy_err))
			return exit_failure
		}
		jlog.Logger_Info(input.Logger, "copied font", jlog.String("file", file))
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
		jlog.Logger_Error(input.Logger, "font cache refresh failed", jlog.Err(refresh_err))
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
	Shell Shell
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
		jlog.Logger_Info(input.Shell.Logger, "direnv already installed")
		return 0
	}
	jlog.Logger_Info(input.Shell.Logger, "building direnv")
	destination := filepath.Join(input.Binary_Directory, "direnv")
	build := "cd " + input.Direnv_Directory +
		" && CGO_ENABLED=0 go build -mod=vendor" +
		" -o " + destination + " ."
	if !run_spawn(input.Shell, "sh", "-c", build) {
		jlog.Logger_Error(input.Shell.Logger, "direnv build failed")
		return exit_failure
	}
	return 0
}

// Reports whether the direnv binary in the bin directory already reports the
// wanted version. Probing that exact path leaves a present build alone.
func direnv_built(shell Shell, binary_directory string) (built bool) {
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
	// Shell runs the `which` probes that gate the step, the rustup script, and the ln
	// calls. rustup reads CARGO_HOME and RUSTUP_HOME from the inherited environment, so
	// the toolchain lands under Cargo_Directory without plumbing.
	Shell Shell
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
		jlog.Logger_Info(input.Shell.Logger, "rust already installed")
	} else {
		jlog.Logger_Info(input.Shell.Logger, "installing rust")
		if !run_spawn(input.Shell, rust_install_invocation()...) {
			jlog.Logger_Error(input.Shell.Logger, "rust install failed")
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
		jlog.Logger_Error(input.Shell.Logger, "rust link failed")
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
func rust_installed(shell Shell, cargo_directory string) (installed bool) {
	return Installed(&Installed_Input{
		Shell:      shell,
		Executable: filepath.Join(cargo_directory, "bin", "rustc"),
		Version:    "rustc " + rust_version,
	})
}

// Carries the arguments for symlinking the rust toolchain onto PATH.
type rust_link_input struct {
	Shell           Shell
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
		if !run_spawn(input.Shell, "ln", "-sf", source, target) {
			return false
		}
	}
	return true
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
	Shell Shell
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
		jlog.Logger_Info(input.Shell.Logger, "fzf already installed")
		return 0
	}
	jlog.Logger_Info(input.Shell.Logger, "building fzf")
	destination := filepath.Join(input.Binary_Directory, "fzf")
	build := "cd " + input.Fzf_Directory +
		" && go build -mod=vendor" +
		" -ldflags '-s -w -X main.version=" + fzf_version + " -X main.revision='" +
		" -o " + destination + " ."
	if !run_spawn(input.Shell, "sh", "-c", build) {
		jlog.Logger_Error(input.Shell.Logger, "fzf build failed")
		return exit_failure
	}
	return 0
}

// Reports whether the fzf binary in the bin directory already reports the wanted
// version. Probing that exact path leaves a present build alone.
func fzf_built(shell Shell, binary_directory string) (built bool) {
	return Installed(&Installed_Input{
		Shell:      shell,
		Executable: filepath.Join(binary_directory, "fzf"),
		Version:    fzf_version,
	})
}

// Install_Command_Input carries the injected dependencies Install_Command needs to
// build a command from this repository with the Go toolchain and place it on PATH.
type Install_Command_Input struct {
	// Package_Directory is the Go package in this repository go build compiles.
	// Unlike the vendored tools, these depend only on the standard library and this
	// module's own packages, so the build needs no vendor tree and no network.
	Package_Directory string
	// Binary_Directory is the directory on PATH the command is built into; it is the
	// install location, the same one the build's output names.
	Binary_Directory string
	// Binary_Name is the built command's filename — the name it is invoked by on PATH
	// and the name the idempotency gate looks up. markdown_to_pdf installs as "m2p",
	// so a command's name is not always its package directory's name.
	Binary_Name string
	// Shell runs the PATH-presence gate and the go build.
	Shell Shell
}

// Install_Command builds a command from this repository straight into the bin
// directory. Its one idempotency check is whether the command already resolves on
// PATH: these are this repo's own programs with no pinned version to match, rebuilt
// freely, so a name already on PATH is left alone and an absent one is built. Like
// fzf and direnv, it is a Go build, so the build output is the install — no separate
// copy or symlink.
func Install_Command(input *Install_Command_Input) (status_code int) {
	if input.Package_Directory == "" {
		return 0
	}
	if input.Binary_Directory == "" {
		return 0
	}
	if input.Binary_Name == "" {
		return 0
	}
	if command_on_path(input.Shell, input.Binary_Name) {
		jlog.Logger_Info(input.Shell.Logger, "command already installed",
			jlog.String("command", input.Binary_Name))
		return 0
	}
	jlog.Logger_Info(input.Shell.Logger, "building command",
		jlog.String("command", input.Binary_Name))
	destination := filepath.Join(input.Binary_Directory, input.Binary_Name)
	build := "cd " + input.Package_Directory +
		" && go build -o " + destination + " ."
	if !run_spawn(input.Shell, "sh", "-c", build) {
		jlog.Logger_Error(input.Shell.Logger, "command build failed",
			jlog.String("command", input.Binary_Name))
		return exit_failure
	}
	return 0
}

// Reports whether a command of the given name already resolves on PATH — the only
// idempotency gate for this repo's own commands. `which` prints the resolved path
// on stdout and nothing when the name is unknown, so a non-empty result means the
// command is present and the build is skipped.
func command_on_path(shell Shell, name string) (present bool) {
	return run_pipe(shell, "which", name) != ""
}

// Jj_version is the release the bootstrap wants — the prefix of `jj --version`.
// jj's build.rs appends a commit hash, so the gate prefix-matches; tracks the
// vendored third_party/jj workspace version, bump it with the source.
const jj_version = "jj 0.42.0"

// Install_Jj_Input carries the injected dependencies Install_Jj needs to build the
// vendored jj with cargo and install it into the binary directory.
type Install_Jj_Input struct {
	// Jj_Directory is the vendored jj workspace root cargo builds from; its
	// .cargo/config.toml points the offline build at the committed vendor tree.
	Jj_Directory string
	// Binary_Directory is where the jj binary is installed and probed. cargo install
	// writes into it via --root (its parent), the same directory the Go tools build
	// into, so no symlink is needed.
	Binary_Directory string
	// Shell runs the `jj --version` gate and the cargo build.
	Shell Shell
}

// Install_Jj builds jj from the vendored workspace with cargo, installing the binary
// straight into the binary directory. It is idempotent: when the built jj already
// reports the wanted version the build is skipped, so a repeat bootstrap does no
// heavy work.
func Install_Jj(input *Install_Jj_Input) (status_code int) {
	if input.Jj_Directory == "" {
		return 0
	}
	if input.Binary_Directory == "" {
		return 0
	}
	if jj_built(input.Shell, input.Binary_Directory) {
		jlog.Logger_Info(input.Shell.Logger, "jj already built")
		return 0
	}
	jlog.Logger_Info(input.Shell.Logger, "building jj")
	if !run_spawn(input.Shell,
		jj_install_invocation(input)...) {
		jlog.Logger_Error(input.Shell.Logger, "jj build failed")
		return exit_failure
	}
	return 0
}

// Returns the invocation that builds and installs jj from the vendored workspace.
// cd into the workspace root so its .cargo/config.toml maps crates to the vendor
// tree; --offline/--locked build with no network; --bin jj --path cli installs only
// the jj binary from the jj-cli package, not its test helpers; --root installs it
// into the binary directory (cargo writes to <root>/bin).
func jj_install_invocation(input *Install_Jj_Input) (arguments []string) {
	root := filepath.Dir(input.Binary_Directory)
	return []string{
		"sh", "-c",
		"cd " + input.Jj_Directory + " && " +
			"cargo install --offline --locked --bin jj --path cli --root " + root,
	}
}

// Reports whether the jj binary already installed in the binary directory reports
// the wanted version. Probing the exact path, not PATH, keeps a build present at a
// non-PATH location from being needlessly recompiled.
func jj_built(shell Shell, binary_directory string) (built bool) {
	return Installed(&Installed_Input{
		Shell:      shell,
		Executable: filepath.Join(binary_directory, "jj"),
		Version:    jj_version,
	})
}

// Ripgrep_version is the release the bootstrap wants — the prefix of `rg
// --version`. ripgrep's build.rs appends a git rev, so the gate prefix-matches;
// tracks the vendored third_party/ripgrep Cargo.toml version, bump with it.
const ripgrep_version = "ripgrep 15.1.0"

// Install_Ripgrep_Input carries the injected dependencies Install_Ripgrep needs to
// build the vendored ripgrep with cargo and install its rg binary.
type Install_Ripgrep_Input struct {
	// Ripgrep_Directory is the vendored ripgrep crate cargo builds from; its
	// .cargo/config.toml points the offline build at the committed vendor tree.
	Ripgrep_Directory string
	// Binary_Directory is where the rg binary is installed and probed. cargo install
	// writes into it via --root (its parent), the same directory the Go tools build
	// into, so no symlink is needed.
	Binary_Directory string
	// Shell runs the `rg --version` gate and the cargo build.
	Shell Shell
}

// Install_Ripgrep builds ripgrep from the vendored crate with cargo, installing the
// rg binary straight into the binary directory. It is idempotent: when the built rg
// already reports the wanted version the build is skipped, so a repeat bootstrap
// does no heavy work.
func Install_Ripgrep(input *Install_Ripgrep_Input) (status_code int) {
	if input.Ripgrep_Directory == "" {
		return 0
	}
	if input.Binary_Directory == "" {
		return 0
	}
	if ripgrep_built(input.Shell, input.Binary_Directory) {
		jlog.Logger_Info(input.Shell.Logger, "ripgrep already built")
		return 0
	}
	jlog.Logger_Info(input.Shell.Logger, "building ripgrep")
	invocation := ripgrep_install_invocation(input)
	if !run_spawn(input.Shell, invocation...) {
		jlog.Logger_Error(input.Shell.Logger, "ripgrep build failed")
		return exit_failure
	}
	return 0
}

// Returns the invocation that builds and installs rg from the vendored crate. cd
// into the crate so its .cargo/config.toml maps crates to the vendor tree;
// --offline/--locked build with no network; --features pcre2 enables the
// look-around/backreference engine, off by default; --root installs it into the
// binary directory (cargo writes to <root>/bin).
func ripgrep_install_invocation(input *Install_Ripgrep_Input) (arguments []string) {
	root := filepath.Dir(input.Binary_Directory)
	return []string{
		"sh", "-c",
		"cd " + input.Ripgrep_Directory + " && " +
			"cargo install --offline --locked --features pcre2 " +
			"--bin rg --path . --root " + root,
	}
}

// Reports whether the rg binary already installed in the binary directory reports
// the wanted version. Probing the exact path, not PATH, keeps a build present at a
// non-PATH location from being needlessly recompiled.
func ripgrep_built(shell Shell, binary_directory string) (built bool) {
	return Installed(&Installed_Input{
		Shell:      shell,
		Executable: filepath.Join(binary_directory, "rg"),
		Version:    ripgrep_version,
	})
}

// Fd_version is the release the bootstrap wants — the prefix of `fd --version`.
// Tracks the vendored third_party/fd Cargo.toml version, bump it with the source.
const fdcli_version = "fd 10.4.2"

// Install_Fdcli_Input carries the injected dependencies Install_Fdcli needs to build the
// vendored fd with cargo and install it.
type Install_Fdcli_Input struct {
	// Fdcli_Directory is the vendored fd crate cargo builds from; its
	// .cargo/config.toml points the offline build at the committed vendor tree.
	Fdcli_Directory string
	// Binary_Directory is where the fd binary is installed and probed. cargo install
	// writes into it via --root (its parent), the same directory the Go tools build
	// into, so no symlink is needed.
	Binary_Directory string
	// Shell runs the `fd --version` gate and the cargo build.
	Shell Shell
}

// Install_Fdcli builds fd from the vendored crate with cargo, installing the binary
// straight into the binary directory. It is idempotent: when the built fd already
// reports the wanted version the build is skipped, so a repeat bootstrap does no
// heavy work.
func Install_Fdcli(input *Install_Fdcli_Input) (status_code int) {
	if input.Fdcli_Directory == "" {
		return 0
	}
	if input.Binary_Directory == "" {
		return 0
	}
	if fdcli_built(input.Shell, input.Binary_Directory) {
		jlog.Logger_Info(input.Shell.Logger, "fd already built")
		return 0
	}
	jlog.Logger_Info(input.Shell.Logger, "building fd")
	invocation := fdcli_install_invocation(input)
	if !run_spawn(input.Shell, invocation...) {
		jlog.Logger_Error(input.Shell.Logger, "fd build failed")
		return exit_failure
	}
	return 0
}

// Returns the invocation that builds and installs fd from the vendored crate. cd
// into the crate so its .cargo/config.toml maps crates to the vendor tree;
// --offline/--locked build with no network against the default features; --root
// installs it into the binary directory (cargo writes to <root>/bin).
func fdcli_install_invocation(input *Install_Fdcli_Input) (arguments []string) {
	root := filepath.Dir(input.Binary_Directory)
	return []string{
		"sh", "-c",
		"cd " + input.Fdcli_Directory + " && " +
			"cargo install --offline --locked --bin fd --path . --root " + root,
	}
}

// Reports whether the fd binary already installed in the binary directory reports
// the wanted version. Probing the exact path, not PATH, keeps a build present at a
// non-PATH location from being needlessly recompiled.
func fdcli_built(shell Shell, binary_directory string) (built bool) {
	return Installed(&Installed_Input{
		Shell:      shell,
		Executable: filepath.Join(binary_directory, "fd"),
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
	Shell Shell
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
		jlog.Logger_Info(input.Shell.Logger, "ghostty already installed")
	} else {
		jlog.Logger_Info(input.Shell.Logger, "installing ghostty")
		invocation := ghostty_install_invocation(input.Applications_Directory)
		if !run_spawn(input.Shell, invocation...) {
			jlog.Logger_Error(input.Shell.Logger, "ghostty install failed")
			return exit_failure
		}
	}
	// Always (re)link, even when the install was skipped, so a missing symlink is
	// restored without re-downloading. ln -sf is idempotent.
	source := filepath.Join(input.Applications_Directory, ghostty_application_binary_subpath)
	target := filepath.Join(input.Link_Directory, "ghostty")
	if !run_spawn(input.Shell, "ln", "-sf", source, target) {
		jlog.Logger_Error(input.Shell.Logger, "ghostty link failed")
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
func ghostty_installed(shell Shell, applications_directory string) (installed bool) {
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
	return run_spawn(shell, ghostty_codesign_invocation(application)...)
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
