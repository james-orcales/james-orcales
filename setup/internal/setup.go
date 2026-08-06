// Package setup is the pure library tier of the setup binary, the one-shot
// bootstrap for this machine. It mirrors a tree of dotfiles into the home
// directory in one direction, repo to home — Plan diffs two injected filesystems
// and returns the writes a sync would make, and Mirror performs them through an
// injected writer — Install_Neovim builds and installs the vendored Neovim, and
// Install_Fonts copies the vendored fonts into the OS font directory. Every entry
// binds to no real filesystem and stays a black box under test.
package setup

import (
	"container/list"
	"errors"
	"fmt"
	"path/filepath"

	shared_bytes "local/james-orcales/shared/bytes"
	invariant "local/james-orcales/shared/invariant/default"
	sysio "local/james-orcales/shared/io"
	"local/james-orcales/shared/jlog"
	shared_slices "local/james-orcales/shared/slices"
	shared_strings "local/james-orcales/shared/strings"
	systime "local/james-orcales/shared/time"
)

// DOTFILE_BYTES_MAX bounds a single dotfile read into one fixed buffer. 1 MiB
// dwarfs any real configuration file yet caps memory against a pathological
// input, satisfying the linter's unbounded-read ban.
const DOTFILE_BYTES_MAX = 1048576

// DOTFILE_PAYLOAD_BYTES_MAX leaves one byte for the EOF probe at the read limit.
const DOTFILE_PAYLOAD_BYTES_MAX = DOTFILE_BYTES_MAX - 1

// COPY_BYTES_MAX bounds one copied font file at 64 MiB.
const COPY_BYTES_MAX = 67108864

// EXIT_FAILURE is the process exit code for a planning or write failure during
// an otherwise well-formed run.
const EXIT_FAILURE Exit_Code = 1

// EXIT_USAGE identifies an invalid setup execution environment.
const EXIT_USAGE Exit_Code = 2

// EXIT_SUCCESS is the process exit code for a successful setup run.
const EXIT_SUCCESS Exit_Code = 0

// SCHEDULER_ENTRY_COUNT permits cleanup operations without a large kernel ring. Setup submits
// commands and file operations in sequence.
const SCHEDULER_ENTRY_COUNT uint16 = 32

// SCHEDULER_FLAGS selects the default scheduler behavior on each operating system.
const SCHEDULER_FLAGS uint32 = 0

// PROCESS_DURATION_MAX permits a slow first build and stops a process group that cannot finish.
const PROCESS_DURATION_MAX = 6 * systime.HOUR

// PUMP_DURATION_MAX lets every bounded process deadline retire before the root guard expires.
const PUMP_DURATION_MAX = PROCESS_DURATION_MAX + systime.SECOND

// IO_OPERATION_INCOMPLETE reports that the root deadline expired before a callback retired.
const IO_OPERATION_INCOMPLETE = "the IO operation did not complete"

// ROOT_REFUSAL prevents the bootstrap from creating root-owned home files.
const ROOT_REFUSAL = "run as your normal user, not root"

// Main constructs the complete runner from injected capabilities and returns without driving IO.
func Main(input *Main_Input) (runner Runner) {
	defer func() { Runner_Invariants(runner, "Main.runner") }()
	Main_Input_Invariants(input, "Main.input")
	state, runner := new_runner()
	environment, valid := New_Environment(input.Environment)
	if !valid {
		runner_stop(state, EXIT_USAGE)
		return runner
	}
	bootstrap_input := &Bootstrap_Steps_Input{
		Home_Directory:   environment.Home_Directory,
		Operating_System: environment.Operating_System,
		Cargo_Directory:  environment.Cargo_Directory,
		Data_Directory:   environment.Data_Directory,
		IO:               input.IO,
		Logger:           environment.Logger,
		Stdout:           environment.Stdout,
		Stderr:           environment.Stderr,
	}
	bootstrap_runner_start(
		state, runner_steps(bootstrap_input), environment.Logger)
	return runner
}

// Environment_Input contains operating-system facts. Package main reads each fact once, while
// this package owns validation and logger policy.
type Environment_Input struct {
	// Clock supplies timestamped logging.
	Clock systime.Clock
	// Console receives setup log records.
	Console sysio.Stream
	// Color reports whether Console supports terminal color.
	Color Console_Color
	// Effective_User_Identifier identifies root ownership before setup writes home files.
	Effective_User_Identifier Effective_User_Identifier
	// Home_Directory is the resolved user home directory.
	Home_Directory Home_Directory
	// Home_Error reports that the operating system could not resolve Home_Directory.
	Home_Error error
	// Operating_System selects platform-specific setup steps.
	Operating_System Operating_System
	// Cargo_Directory preserves an explicit CARGO_HOME value.
	Cargo_Directory Cargo_Directory
	// Data_Directory preserves an explicit XDG_DATA_HOME value.
	Data_Directory Data_Directory
	// Stdout receives live command output.
	Stdout sysio.Stream
	// Stderr receives live command diagnostics.
	Stderr sysio.Stream
}

// Environment_Input_Invariants states each environment fact that has a preset.
func Environment_Input_Invariants(input *Environment_Input, namespace invariant.Namespace) {
	systime.Clock_Invariants(input.Clock, namespace)
	Console_Color_Invariants(input.Color, namespace)
	Effective_User_Identifier_Invariants(input.Effective_User_Identifier, namespace)
	Home_Directory_Invariants(input.Home_Directory, namespace)
	Operating_System_Invariants(input.Operating_System, namespace)
	Cargo_Directory_Invariants(input.Cargo_Directory, namespace)
	Data_Directory_Invariants(input.Data_Directory, namespace)
}

// Existing logger and process APIs require the standard writer method at their final boundary.
type Stream_Writer struct {
	// Stream retains the shared output boundary while external APIs call Write.
	Stream sysio.Stream
}

// Stream_Writer_Invariants rejects an adapter that cannot forward a write.
func Stream_Writer_Invariants(writer Stream_Writer, _ invariant.Namespace) {
	invariant.Always(writer.Stream.Procedure != nil, "The stream writer can write.")
}

// Write forwards the required writer operation without adding a second setup output type.
func (writer Stream_Writer) Write(content []byte) (count int, err error) {
	Stream_Writer_Invariants(writer, "Stream_Writer.Write.writer")
	written, write_err := sysio.Write(writer.Stream, content)
	return int(written), write_err
}

// Environment contains validated facts and the logger that setup injects into each step.
type Environment struct {
	// Logger reports setup progress and startup errors.
	Logger jlog.Logger
	// Home_Directory is the destination for dotfiles and user tools.
	Home_Directory Home_Directory
	// Operating_System selects platform-specific setup steps.
	Operating_System Operating_System
	// Cargo_Directory preserves an explicit CARGO_HOME value.
	Cargo_Directory Cargo_Directory
	// Data_Directory preserves an explicit XDG_DATA_HOME value.
	Data_Directory Data_Directory
	// Stdout receives live command output.
	Stdout sysio.Stream
	// Stderr receives live command diagnostics.
	Stderr sysio.Stream
}

// Environment_Invariants states each validated environment fact that has a preset.
func Environment_Invariants(environment Environment, namespace invariant.Namespace) {
	Home_Directory_Invariants(environment.Home_Directory, namespace)
	Operating_System_Invariants(environment.Operating_System, namespace)
	Cargo_Directory_Invariants(environment.Cargo_Directory, namespace)
	Data_Directory_Invariants(environment.Data_Directory, namespace)
}

// File operations require these shared IO members.
func file_io_requirements(system sysio.IO, _ invariant.Namespace) {
	invariant.Always(system.Read_Directory != nil, "Setup IO has a directory reader.")
	invariant.Always(system.Status != nil, "Setup IO has a status reader.")
	invariant.Always(system.Open != nil, "Setup IO has a file opener.")
	invariant.Always(system.Create != nil, "Setup IO has a file creator.")
	invariant.Always(system.Make_Directory != nil, "Setup IO has a directory creator.")
	invariant.Always(system.Read != nil, "Setup IO has a file read submission.")
	invariant.Always(system.Write != nil, "Setup IO has a file write submission.")
	invariant.Always(system.Close != nil, "Setup IO has a file close submission.")
}

// Process operations require the shared spawn member.
func process_io_requirements(system sysio.IO, _ invariant.Namespace) {
	invariant.Always(system.Spawn != nil, "Setup IO has a process spawn submission.")
}

// File_Path is one validated absolute path submitted through shared IO.
type File_Path string

// File_Path_Invariants bounds a shared IO file path.
func File_Path_Invariants(path File_Path, namespace invariant.Namespace) {
	invariant.Tree(path, namespace).Range_Int(
		len(path), FILE_PATH_BYTES_MIN, FILE_PATH_BYTES_MAX).Ensure()
}

// File_read lets synchronous specification IO execute the same runner operation as production.

// File_write lets synchronous specification IO execute the production runner operation.

// File_close lets synchronous specification IO execute the production close operation.

// Process_spawn lets synchronous specification IO execute the production spawn operation.

// Main_Input contains the complete injected environment for one setup run.
type Main_Input struct {
	// Environment contains the operating-system facts that Main validates.
	Environment *Environment_Input
	// IO supplies every setup file and process operation.
	IO sysio.IO
}

// Main_Input_Invariants states the complete setup dependency set.
func Main_Input_Invariants(input *Main_Input, namespace invariant.Namespace) {
	Environment_Input_Invariants(input.Environment, namespace)
	file_io_requirements(input.IO, namespace)
	process_io_requirements(input.IO, namespace)
}

// New_Environment applies startup policy to operating-system facts and returns the required exit
// status when setup cannot continue.
func New_Environment(input *Environment_Input) (
	environment Environment, valid Environment_Valid,
) {
	defer func() {
		Environment_Invariants(environment, "New_Environment.environment")
		Environment_Valid_Invariants(valid, "New_Environment.valid")
	}()
	Environment_Input_Invariants(input, "New_Environment.input")
	logger_input := jlog.New_Console_Logger_Input{
		Color: bool(input.Color),
		Floor: jlog.LEVEL_DEBUG, Clock: input.Clock,
	}
	if input.Console.Procedure != nil {
		logger_input.Console = Stream_Writer{Stream: input.Console}
	}
	logger := jlog.New_Console_Logger(logger_input)
	environment = Environment{
		Logger: logger, Home_Directory: input.Home_Directory,
		Operating_System: input.Operating_System, Cargo_Directory: input.Cargo_Directory,
		Data_Directory: input.Data_Directory, Stdout: input.Stdout, Stderr: input.Stderr,
	}
	if input.Effective_User_Identifier == 0 {
		jlog.Logger_Error(logger, ROOT_REFUSAL)
		return environment, false
	}
	if input.Home_Error != nil {
		jlog.Logger_Error(
			logger, "cannot resolve home directory", jlog.Err(input.Home_Error),
		)
		return environment, false
	}
	return environment, true
}

// Mirror_Input carries the injected dependencies Mirror needs to sync dotfiles.
type Mirror_Input struct {
	// IO is the shared boundary used to walk, read, write, and spawn.
	IO sysio.IO
	// Source_Directory is the absolute dotfiles tree walked in full from its root.
	Source_Directory Source_Directory
	// Destination_Directory is the absolute home directory writes land under, and the diff
	// reads existing files from.
	Destination_Directory Destination_Directory
	// Operating_System gates the macos defaults step, which runs only on
	// "darwin" (a runtime.GOOS value).
	Operating_System Operating_System
	// Logger records the sync's narration: one line per file written, the scan's
	// progress, and a diagnostic when planning or a write fails. The zero Logger is a
	// disabled no-op, so a caller wanting silence passes none.
	Logger jlog.Logger
	// Stdout receives live macOS defaults output.
	Stdout sysio.Stream
	// Stderr receives live macOS defaults diagnostics.
	Stderr sysio.Stream
}

// Mirror_Input_Invariants states the mirror dependency set and its path facts.
func Mirror_Input_Invariants(input *Mirror_Input, namespace invariant.Namespace) {
	file_io_requirements(input.IO, namespace)
	process_io_requirements(input.IO, namespace)
	Source_Directory_Invariants(input.Source_Directory, namespace)
	Destination_Directory_Invariants(input.Destination_Directory, namespace)
	Operating_System_Invariants(input.Operating_System, namespace)
}

// Mirror syncs the source dotfiles into the home directory and, on darwin, applies the macos
// defaults. The separate name keeps Main as the complete binary policy entry point.

// Reports whether the injected file system contains a path. A status error cannot prove that the
// file exists, so the idempotency gate treats that result as absent.
func file_present(system sysio.IO, path Font_Path) (present File_Presence) {
	defer func() { File_Presence_Invariants(present, "file_present.present") }()
	Font_Path_Invariants(path, "file_present.path")
	file_io_requirements(system, "file_present.system")
	status, status_err := system.Status(string(path))
	if status_err != nil {
		return false
	}
	return File_Presence(status.Exists)
}

// Copies one bounded file through the injected file system. The larger font limit stays separate
// from the dotfile limit because a font is binary installation data, not configuration text.

// Takes shared IO and the output policy, not the whole Mirror_Input. Runs
// every macos defaults command, stopping at the first that fails. It logs
// nothing per command — 35 lines of `defaults write` is noise, not progress.

// MACOS_DEFAULTS_SCRIPT lists the `defaults write` and `killall` commands that
// configure macOS, one per line, parsed into commands at run time. The clock
// date format is appended separately by macos_commands because its value holds
// spaces that whitespace-splitting a line would break apart.
const MACOS_DEFAULTS_SCRIPT = `
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

// One external program invocation parsed from MACOS_DEFAULTS_SCRIPT.
type Macos_Command struct {
	// Name is the external program to invoke.
	Name Step_Name
	// Arguments are the program's arguments, in invocation order.
	Arguments Macos_Arguments
}

// Macos_Command_Invariants states one bounded macOS command.
func Macos_Command_Invariants(command Macos_Command, namespace invariant.Namespace) {
	Step_Name_Invariants(command.Name, namespace)
	Macos_Arguments_Invariants(command.Arguments, namespace)
}

// Parses MACOS_DEFAULTS_SCRIPT into one command per non-blank line and appends
// the clock date format command, whose spaced value cannot share the line format.
func macos_commands() (commands Commands) {
	defer func() { Commands_Invariants(commands, "macos_commands.commands") }()
	commands = Commands{}
	for line := range shared_strings.Lines(MACOS_DEFAULTS_SCRIPT) {
		fields := shared_strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		arguments := make(Macos_Arguments, len(fields)-1)
		for index := 1; index < len(fields); index++ {
			arguments[index-1] = string(fields[index])
		}
		commands = append(commands, Macos_Command{
			Name:      Step_Name(fields[0]),
			Arguments: arguments,
		})
	}
	return append(commands, Macos_Command{
		Name: "defaults",
		Arguments: Macos_Arguments{
			"write", "com.apple.menuextra.clock", "DateFormat",
			"-string", "EEE MMM d mm:HH",
		},
	})
}

// File_Write is a single pending write the sync would perform.
type File_Write struct {
	// Destination_Path is the absolute path the contents must be written to: the
	// source file's path relative to the source root, joined onto the home dir.
	Destination_Path Destination_Path
	// Contents is the exact source bytes to write. Comparing them against the
	// destination is what decided this write was needed at all.
	Contents Dotfile_Bytes
}

// File_Write_Invariants states one bounded mirror write.
func File_Write_Invariants(write File_Write, namespace invariant.Namespace) {
	Destination_Path_Invariants(write.Destination_Path, namespace)
	Dotfile_Bytes_Invariants(write.Contents, namespace)
}

// Plan_Input carries the injected dependencies Plan needs to decide what to sync.
type Plan_Input struct {
	// IO is the shared boundary the source tree is walked and read through, and the
	// destination read through for diffing.
	IO sysio.IO
	// Source_Directory is the absolute dotfiles tree, walked in full from its root.
	Source_Directory Source_Directory
	// Destination_Directory is the absolute home directory the relative source
	// paths are mirrored under to form each write's Destination_Path.
	Destination_Directory Destination_Directory
	// Logger records one line naming each directory as the walk reads it. A converged
	// scan writes nothing, so without this the step looks hung while it walks; narrating
	// each directory proves the scan is live. The zero Logger is a disabled no-op, so a
	// direct caller that wants no narration pays nothing.
	Logger jlog.Logger
}

// Plan_Input_Invariants states the mirror plan dependency set.
func Plan_Input_Invariants(input *Plan_Input, namespace invariant.Namespace) {
	file_io_requirements(input.IO, namespace)
	process_io_requirements(input.IO, namespace)
	Source_Directory_Invariants(input.Source_Directory, namespace)
	Destination_Directory_Invariants(input.Destination_Directory, namespace)
}

// Plan returns the writes that would bring the home directory in line with the source
// dotfiles: every regular file under the source tree, each emitted only when the destination
// is missing or its contents differ. It walks through the loop's Read_Directory, classifies each
// directory with one Is_Ignored batch, and prunes an ignored directory so the install tree under
// .local is never descended into.

// Announces the directory the walk is about to read, one line each, so a scan that
// writes nothing still shows it is advancing. The scan is chatter — one line per
// directory — so it logs at debug, below the progress the install steps report.
func plan_narrate(logger jlog.Logger, directory Traversal_Directory) {
	Traversal_Directory_Invariants(directory, "plan_narrate.directory")
	jlog.Logger_Debug(logger, "scanning", jlog.String("dir", string(directory)))
}

// Returns the set of a level's entries that are gitignored, classifying them all in one
// Is_Ignored call. A nil predicate — or an empty level — ignores nothing, so the filter
// stays opt-in and an empty level spawns no probe.

// Decides whether the source file at relative needs syncing. planned is false when the
// source is absent or the destination already holds identical bytes; otherwise it returns
// the write mirroring the relative path under the home directory.

// Reports whether the home directory already holds exactly source_contents at path. A
// destination that is absent or unreadable counts as a mismatch, so the file is written.

// Shared byte operations have a smaller boundary than a dotfile, so each call compares one
// bounded part while this package keeps its existing dotfile limit.
func dotfile_contents_equal(left Dotfile_Bytes, right Dotfile_Bytes) (equal Plan_Decision) {
	defer func() { Plan_Decision_Invariants(equal, "dotfile_contents_equal.equal") }()
	Dotfile_Bytes_Invariants(left, "dotfile_contents_equal.left")
	Dotfile_Bytes_Invariants(right, "dotfile_contents_equal.right")
	if len(left) != len(right) {
		return false
	}
	for offset := 0; offset < len(left); offset += shared_bytes.SLICE_SIZE_MAXIMUM {
		end := offset + shared_bytes.SLICE_SIZE_MAXIMUM
		if end > len(left) {
			end = len(left)
		}
		if !shared_bytes.Equal(
			shared_bytes.Slice(left[offset:end]), shared_bytes.Slice(right[offset:end]),
		) {
			return false
		}
	}
	return true
}

// Runs path with arguments and returns its captured standard output, trimmed — the version
// probes' one use. It passes no sink, so the loop captures the output for parsing rather than
// streaming it. Trimming strips the trailing newline `which` appends, so a resolved path is
// usable as the next probe's executable.

// Runs the command named by the first argument and reports success, streaming the process's
// output live to its sinks so a multi-minute build's progress reaches the user as it
// happens rather than in one burst at the end.

// Installed_Input names the binary an install step probes and the version it must
// report. Two string fields, so it is a struct rather than two parameters.
type Installed_Input struct {
	// IO runs the version probe.
	IO sysio.IO
	// Executable is the binary's exact managed path — probed directly, not via
	// PATH, so a missing symlink never hides a present install.
	Executable Managed_Executable
	// Version is the prefix the binary's --version output must start with; a prefix
	// so build or commit suffixes do not matter.
	Version Version_Prefix
}

// Installed_Input_Invariants states one managed executable version probe.
func Installed_Input_Invariants(input *Installed_Input, namespace invariant.Namespace) {
	process_io_requirements(input.IO, namespace)
	Managed_Executable_Invariants(input.Executable, namespace)
	Version_Prefix_Invariants(input.Version, namespace)
}

// Installed is the shared idempotency probe: it reports whether the binary at its
// managed path already reports the wanted version. An install step that is
// Installed does no work; one that is not reinstalls. direnv and Neovim verify the
// same rule their own way (a version subcommand, and a which-resolved path).

// NEOVIM_SOURCE_SUBPATH locates the vendored Neovim source relative to the
// checkout root. make is pointed at it with -C, so the build needs no
// working-directory plumbing through the injected runner.
const NEOVIM_SOURCE_SUBPATH = "third_party/neovim"

// NEOVIM_PREFIX_SUBPATH is the install prefix relative to the checkout root.
// Neovim installs to <prefix>/bin/nvim and derives its runtime as
// <prefix>/share/nvim/runtime by stripping the binary's name and its parent
// "bin" component from the resolved path. home/.local mirrors ~/.local, so nvim
// lands at home/.local/bin/nvim — on PATH — and finds its own runtime, no symlink.
const NEOVIM_PREFIX_SUBPATH = "home/.local"

// NEOVIM_BUILD_TYPE is the CMAKE_BUILD_TYPE the bootstrap compiles: optimized,
// but with enough debug info to recover a backtrace if Neovim ever crashes.
const NEOVIM_BUILD_TYPE = "RelWithDebInfo"

// NEOVIM_INSTALL_GOAL is the second make goal, run after the configure-build
// pass to copy the binary, runtime, and parsers under the prefix.
const NEOVIM_INSTALL_GOAL = "install"

// NEOVIM_VERSION is the release the bootstrap wants, tracking the vendored
// source's NVIM_VERSION_* in third_party/neovim/CMakeLists.txt. The build is
// skipped when an nvim already on PATH reports it, so a bootstrap that already
// has the wanted nvim does no work. Bump it with the vendored source.
const NEOVIM_VERSION = "v0.12.3"

// Install_Neovim_Input carries the injected dependencies Install_Neovim needs to
// build and install the vendored Neovim.
type Install_Neovim_Input struct {
	// Repository_Directory is the absolute checkout root the build subpaths join
	// onto to form the source and prefix locations.
	Repository_Directory Repository_Directory
	// IO runs every probe and make process.
	IO sysio.IO
	// Logger records setup progress.
	Logger jlog.Logger
	// Stdout receives streamed make output.
	Stdout sysio.Stream
	// Stderr receives streamed make diagnostics.
	Stderr sysio.Stream
}

// Install_Neovim_Input_Invariants states the Neovim build dependency set.
func Install_Neovim_Input_Invariants(
	input *Install_Neovim_Input, namespace invariant.Namespace,
) {
	Repository_Directory_Invariants(input.Repository_Directory, namespace)
	process_io_requirements(input.IO, namespace)
}

// Install_Neovim builds the vendored Neovim and installs it under the local
// prefix, where it lands at local/bin/nvim — already on PATH — and finds its own
// runtime. Every step is idempotent: make rebuilds only what changed and install
// re-copies, so a repeated bootstrap converges without error.

// Returns the two make invocations the build runs in order: configure-and-build,
// then install. Each begins with the executable, "make", because run_spawn
// reads the first argument as the program. Both carry the same prefix so the
// Makefile's checkprefix never re-runs cmake between them; the install pass
// differs only by the appended install goal. -C aims make at the vendored source
// without disturbing the caller's working directory.
func neovim_make_invocations(
	repository_directory Repository_Directory,
) (invocations Invocations) {
	defer func() {
		Invocations_Invariants(invocations, "neovim_make_invocations.invocations")
	}()
	Repository_Directory_Invariants(
		repository_directory, "neovim_make_invocations.repository_directory")
	source := filepath.Join(string(repository_directory), NEOVIM_SOURCE_SUBPATH)
	prefix := filepath.Join(string(repository_directory), NEOVIM_PREFIX_SUBPATH)
	build := []string{
		"make", "-C", source,
		"CMAKE_BUILD_TYPE=" + NEOVIM_BUILD_TYPE,
		"CMAKE_INSTALL_PREFIX=" + prefix,
	}
	install := append(shared_slices.Clone(build), NEOVIM_INSTALL_GOAL)
	return Invocations{build, install}
}

// Reports whether version_output — the text `nvim --version` prints — names the
// wanted release NEOVIM_VERSION. Only the leading vMAJOR.MINOR.PATCH token is
// compared, so a build's -dev or +commit suffix does not matter. Empty output,
// from no nvim on PATH, is not the release, so the build runs.
func neovim_version_present(version_output Version_Output) (present File_Presence) {
	defer func() { File_Presence_Invariants(present, "neovim_version_present.present") }()
	Version_Output_Invariants(version_output, "neovim_version_present.version_output")
	first_line, _, _ := shared_strings.Cut(shared_strings.Text(version_output), "\n")
	for _, field := range shared_strings.Fields(first_line) {
		if !shared_strings.Has_Prefix(field, "v") {
			continue
		}
		base, _, _ := shared_strings.Cut(field, "-")
		return base == NEOVIM_VERSION
	}
	return false
}

// Reports whether the nvim on PATH is the checkout's own build at the wanted
// release. It resolves nvim first and rejects a path outside the repository, so a
// system package at the same version cannot stand in; only an in-repository
// binary then has its version checked.

// Returns the Iosevka TTF filenames the install copies. A function rather than a
// package var so the list stays within the deterministic tier's value rules.
func iosevka_font_files() (files File_Paths) {
	defer func() { File_Paths_Invariants(files, "iosevka_font_files.files") }()
	return File_Paths{
		"IosevkaNerdFontMono-Regular.ttf",
		"IosevkaNerdFontMono-Bold.ttf",
		"IosevkaNerdFontMono-Oblique.ttf",
		"IosevkaNerdFontMono-BoldOblique.ttf",
	}
}

// Install_Fonts_Input carries the shared IO and paths needed to place vendored fonts.
type Install_Fonts_Input struct {
	// IO supplies status, copy, and optional cache-refresh operations.
	IO sysio.IO
	// Home_Directory locates the vendored font source below the checkout.
	Home_Directory Home_Directory
	// Font_Directory is the absolute destination, already resolved per operating
	// system. Empty means this OS has no known user font directory, so the step
	// does nothing.
	Font_Directory Font_Directory
	// Refresh_Cache reports whether the operating system needs fc-cache after copies.
	Refresh_Cache Cache_Refresh
	// Logger records one line per face naming each copy and each skip, and a diagnostic
	// when a copy or refresh fails. The zero Logger is a disabled no-op.
	Logger jlog.Logger
	// Stdout receives live cache-refresh output.
	Stdout sysio.Stream
	// Stderr receives live cache-refresh diagnostics.
	Stderr sysio.Stream
}

// Install_Fonts_Input_Invariants states the font install destination.
func Install_Fonts_Input_Invariants(
	input *Install_Fonts_Input, namespace invariant.Namespace,
) {
	file_io_requirements(input.IO, namespace)
	process_io_requirements(input.IO, namespace)
	Home_Directory_Invariants(input.Home_Directory, namespace)
	Font_Directory_Invariants(input.Font_Directory, namespace)
	Cache_Refresh_Invariants(input.Refresh_Cache, namespace)
}

// Install_Fonts places the vendored Iosevka faces into the per-OS user font
// directory and, where the OS needs it, refreshes the font cache. It is
// idempotent: a destination already holding the fonts copies nothing and skips
// the refresh, so a repeat bootstrap does no work.

// DIRENV_VERSION is the release the bootstrap wants — the prefix of `direnv
// --version`. direnv embeds version.txt, so a plain build self-reports it; tracks
// the vendored third_party/direnv source, bump it with the source.
const DIRENV_VERSION = "2.37.1"

// Install_Direnv_Input carries the injected dependencies Install_Direnv needs to
// build the vendored direnv with the Go toolchain and place it on PATH.
type Install_Direnv_Input struct {
	// Direnv_Directory is the vendored direnv Go module go build compiles, offline
	// against its committed vendor tree.
	Direnv_Directory Direnv_Directory
	// Binary_Directory is the directory on PATH direnv is built into; it is both the
	// install location and the gate's probe target.
	Binary_Directory Binary_Directory
	// IO runs the version gate and the Go build.
	IO sysio.IO
	// Logger records setup progress.
	Logger jlog.Logger
	// Stdout receives streamed build output.
	Stdout sysio.Stream
	// Stderr receives streamed build diagnostics.
	Stderr sysio.Stream
}

// Install_Direnv_Input_Invariants states the direnv build dependency set.
func Install_Direnv_Input_Invariants(
	input *Install_Direnv_Input, namespace invariant.Namespace,
) {
	Direnv_Directory_Invariants(input.Direnv_Directory, namespace)
	Binary_Directory_Invariants(input.Binary_Directory, namespace)
	process_io_requirements(input.IO, namespace)
}

// Install_Direnv builds the vendored direnv straight into the bin directory, where
// the shell hook and every .envrc can find it. It is the bootstrap's first step
// because everything downstream is driven by direnv. It is idempotent: when the
// built direnv already reports the wanted version, it does nothing.

// Reports whether the direnv binary in the bin directory already reports the
// wanted version. Probing that exact path leaves a present build alone.

// RUST_VERSION is the toolchain rustup installs and the gate checks. Pinned, not
// "stable", so the idempotency check has a fixed version to match; bump it
// deliberately. `rustc --version` prints "rustc <version> (<commit> <date>)".
const RUST_VERSION = "1.96.0"

// Install_Rust_Input carries the injected dependencies Install_Rust needs to
// install the Rust toolchain and expose it on PATH.
type Install_Rust_Input struct {
	// Cargo_Directory is CARGO_HOME, where rustup places cargo, rustc, and rustup
	// under bin/. Empty means CARGO_HOME is not configured, so the step does
	// nothing rather than install into an unknown location.
	Cargo_Directory Cargo_Directory
	// Link_Directory is the one directory on PATH the toolchain binaries are
	// symlinked into, so PATH carries a single entry rather than CARGO_HOME/bin as
	// well. Empty disables the step for the same reason as an empty Cargo_Directory.
	Link_Directory Rust_Link_Directory
	// IO runs the probes, rustup script, and link commands. Rustup reads CARGO_HOME and
	// RUSTUP_HOME from the inherited environment.
	IO sysio.IO
	// Logger records setup progress.
	Logger jlog.Logger
	// Stdout receives streamed process output.
	Stdout sysio.Stream
	// Stderr receives streamed process diagnostics.
	Stderr sysio.Stream
}

// Install_Rust_Input_Invariants states the Rust install dependency set.
func Install_Rust_Input_Invariants(input *Install_Rust_Input, namespace invariant.Namespace) {
	Cargo_Directory_Invariants(input.Cargo_Directory, namespace)
	Rust_Link_Directory_Invariants(input.Link_Directory, namespace)
	process_io_requirements(input.IO, namespace)
}

// Install_Rust installs the Rust toolchain with rustup and symlinks cargo,
// rustup, and rustc into the PATH directory, so they are reachable without
// CARGO_HOME/bin on PATH. It is idempotent: when all three already resolve inside
// the link directory, it does nothing, so a repeat bootstrap does no work.

// Returns the invocation that installs rustup non-interactively. -y answers every
// prompt, --no-modify-path leaves the shell rc alone because rust reaches PATH
// through the symlinks, and the toolchain is pinned to the wanted channel.
// RUSTUP_INIT_SKIP_PATH_CHECK silences the "existing Rust" warning those symlinks
// trip. curl --fail/-sSf surfaces a download error instead of piping a half script.
func rust_install_invocation() (arguments Install_Arguments) {
	defer func() {
		Install_Arguments_Invariants(arguments, "rust_install_invocation.arguments")
	}()
	return Install_Arguments{
		"sh", "-c",
		"curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | " +
			"RUSTUP_INIT_SKIP_PATH_CHECK=yes sh -s -- " +
			"-y --no-modify-path --default-toolchain " + RUST_VERSION,
	}
}

// Reports whether the pinned rust toolchain is installed at the current
// CARGO_HOME, probing rustc (which carries the toolchain version; cargo and rustup
// install with it). A stale symlink to an old CARGO_HOME or a wrong version does
// not pass, so a moved or mismatched toolchain is reinstalled.

// Carries the arguments for symlinking the rust toolchain onto PATH.
type Rust_Link_Input struct {
	// IO runs the link commands.
	IO sysio.IO
	// Stdout receives link output.
	Stdout sysio.Stream
	// Stderr receives link diagnostics.
	Stderr sysio.Stream
	// Cargo_Directory is the CARGO_HOME whose bin holds the toolchain to link from.
	Cargo_Directory Required_Cargo_Directory
	// Link_Directory is the PATH entry the toolchain is symlinked into.
	Link_Directory Rust_Link_Directory
}

// Rust_Link_Input_Invariants states the Rust link dependency set.
func Rust_Link_Input_Invariants(input *Rust_Link_Input, namespace invariant.Namespace) {
	process_io_requirements(input.IO, namespace)
	Required_Cargo_Directory_Invariants(input.Cargo_Directory, namespace)
	Rust_Link_Directory_Invariants(input.Link_Directory, namespace)
}

// Symlinks cargo, rustup, and rustc from CARGO_HOME/bin into the link directory
// with ln -sf, so the managed toolchain is reachable from the one PATH entry and
// any stale link there is overwritten. Reports whether every link succeeded.

// FZF_VERSION is the release the bootstrap wants, the exact line `fzf --version`
// prints. It tracks the vendored third_party/fzf source; bump it with the source.
// The build injects it via ldflags so the binary self-reports this string.
const FZF_VERSION = "0.73.1"

// Install_Fzf_Input carries the injected dependencies Install_Fzf needs to build
// the vendored fzf with the Go toolchain and place it on PATH.
type Install_Fzf_Input struct {
	// Fzf_Directory is the vendored fzf Go module go build compiles, offline
	// against its committed vendor tree.
	Fzf_Directory Fzf_Directory
	// Binary_Directory is the directory on PATH the fzf binary is built into; it is
	// both the install location and the gate's probe target.
	Binary_Directory Binary_Directory
	// IO runs the version gate and the Go build.
	IO sysio.IO
	// Logger records setup progress.
	Logger jlog.Logger
	// Stdout receives streamed build output.
	Stdout sysio.Stream
	// Stderr receives streamed build diagnostics.
	Stderr sysio.Stream
}

// Install_Fzf_Input_Invariants states the fzf build dependency set.
func Install_Fzf_Input_Invariants(input *Install_Fzf_Input, namespace invariant.Namespace) {
	Fzf_Directory_Invariants(input.Fzf_Directory, namespace)
	Binary_Directory_Invariants(input.Binary_Directory, namespace)
	process_io_requirements(input.IO, namespace)
}

// Install_Fzf builds fzf from the vendored source straight into the bin directory.
// It is idempotent: when the built fzf already reports the wanted version, it does
// nothing, so a repeat bootstrap does no work. fzf is a Go binary, so the build
// output is the install — no separate copy or symlink.

// Reports whether the fzf binary in the bin directory already reports the wanted
// version. Probing that exact path leaves a present build alone.

// Install_Command_Input carries the injected dependencies Install_Command needs to
// build a command from this repository with the Go toolchain and place it on PATH.
type Install_Command_Input struct {
	// Package_Directory is the Go package in this repository go build compiles.
	// Unlike the vendored tools, these depend only on the standard library and this
	// module's own packages, so the build needs no vendor tree and no network.
	Package_Directory Command_Directory
	// Binary_Directory is the directory on PATH the command is built into; it is the
	// install location, the same one the build's output names.
	Binary_Directory Binary_Directory
	// Binary_Name is the built command's filename — the name it is invoked by on PATH
	// and the name the idempotency gate looks up. markdown_to_pdf installs as "m2p",
	// so a command's name is not always its package directory's name.
	Binary_Name Command_Name
	// IO runs the PATH-presence gate and the Go build.
	IO sysio.IO
	// Logger records setup progress.
	Logger jlog.Logger
	// Stdout receives streamed build output.
	Stdout sysio.Stream
	// Stderr receives streamed build diagnostics.
	Stderr sysio.Stream
}

// Install_Command_Input_Invariants states one repository command build.
func Install_Command_Input_Invariants(
	input *Install_Command_Input, namespace invariant.Namespace,
) {
	Command_Directory_Invariants(input.Package_Directory, namespace)
	Binary_Directory_Invariants(input.Binary_Directory, namespace)
	Command_Name_Invariants(input.Binary_Name, namespace)
	process_io_requirements(input.IO, namespace)
}

// Install_Command builds a command from this repository straight into the bin
// directory. Its one idempotency check is whether the command already resolves on
// PATH: these are this repo's own programs with no pinned version to match, rebuilt
// freely, so a name already on PATH is left alone and an absent one is built. Like
// fzf and direnv, it is a Go build, so the build output is the install — no separate
// copy or symlink.

// Reports whether a command of the given name already resolves on PATH — the only
// idempotency gate for this repo's own commands. `which` prints the resolved path
// on stdout and nothing when the name is unknown, so a non-empty result means the
// command is present and the build is skipped.

// JJ_VERSION is the release the bootstrap wants — the prefix of `jj --version`.
// jj's build.rs appends a commit hash, so the gate prefix-matches; tracks the
// vendored third_party/jj workspace version, bump it with the source.
const JJ_VERSION = "jj 0.42.0"

// Install_Jj_Input carries the injected dependencies Install_Jj needs to build the
// vendored jj with cargo and install it into the binary directory.
type Install_Jj_Input struct {
	// Jj_Directory is the vendored jj workspace root cargo builds from; its
	// .cargo/config.toml points the offline build at the committed vendor tree.
	Jj_Directory Jj_Directory
	// Binary_Directory is where the jj binary is installed and probed. cargo install
	// writes into it via --root (its parent), the same directory the Go tools build
	// into, so no symlink is needed.
	Binary_Directory Binary_Directory
	// IO runs the version gate and the Cargo build.
	IO sysio.IO
	// Logger records setup progress.
	Logger jlog.Logger
	// Stdout receives streamed build output.
	Stdout sysio.Stream
	// Stderr receives streamed build diagnostics.
	Stderr sysio.Stream
}

// Install_Jj_Input_Invariants states the jj build dependency set.
func Install_Jj_Input_Invariants(input *Install_Jj_Input, namespace invariant.Namespace) {
	Jj_Directory_Invariants(input.Jj_Directory, namespace)
	Binary_Directory_Invariants(input.Binary_Directory, namespace)
	process_io_requirements(input.IO, namespace)
}

// Install_Jj builds jj from the vendored workspace with cargo, installing the binary
// straight into the binary directory. It is idempotent: when the built jj already
// reports the wanted version the build is skipped, so a repeat bootstrap does no
// heavy work.

// Returns the invocation that builds and installs jj from the vendored workspace.
// cd into the workspace root so its .cargo/config.toml maps crates to the vendor
// tree; --offline/--locked build with no network; --bin jj --path cli installs only
// the jj binary from the jj-cli package, not its test helpers; --root installs it
// into the binary directory (cargo writes to <root>/bin).
func jj_install_invocation(input *Install_Jj_Input) (arguments Install_Arguments) {
	defer func() {
		Install_Arguments_Invariants(arguments, "jj_install_invocation.arguments")
	}()
	Install_Jj_Input_Invariants(input, "jj_install_invocation.input")
	root := filepath.Dir(string(input.Binary_Directory))
	return Install_Arguments{
		"sh", "-c",
		"cd " + string(input.Jj_Directory) + " && " +
			"cargo install --offline --locked --bin jj --path cli --root " + root,
	}
}

// Reports whether the jj binary already installed in the binary directory reports
// the wanted version. Probing the exact path, not PATH, keeps a build present at a
// non-PATH location from being needlessly recompiled.

// RIPGREP_VERSION is the release the bootstrap wants — the prefix of `rg
// --version`. ripgrep's build.rs appends a git rev, so the gate prefix-matches;
// tracks the vendored third_party/ripgrep Cargo.toml version, bump with it.
const RIPGREP_VERSION = "ripgrep 15.1.0"

// Install_Ripgrep_Input carries the injected dependencies Install_Ripgrep needs to
// build the vendored ripgrep with cargo and install its rg binary.
type Install_Ripgrep_Input struct {
	// Ripgrep_Directory is the vendored ripgrep crate cargo builds from; its
	// .cargo/config.toml points the offline build at the committed vendor tree.
	Ripgrep_Directory Ripgrep_Directory
	// Binary_Directory is where the rg binary is installed and probed. cargo install
	// writes into it via --root (its parent), the same directory the Go tools build
	// into, so no symlink is needed.
	Binary_Directory Binary_Directory
	// IO runs the version gate and the Cargo build.
	IO sysio.IO
	// Logger records setup progress.
	Logger jlog.Logger
	// Stdout receives streamed build output.
	Stdout sysio.Stream
	// Stderr receives streamed build diagnostics.
	Stderr sysio.Stream
}

// Install_Ripgrep_Input_Invariants states the ripgrep build dependency set.
func Install_Ripgrep_Input_Invariants(
	input *Install_Ripgrep_Input, namespace invariant.Namespace,
) {
	Ripgrep_Directory_Invariants(input.Ripgrep_Directory, namespace)
	Binary_Directory_Invariants(input.Binary_Directory, namespace)
	process_io_requirements(input.IO, namespace)
}

// Install_Ripgrep builds ripgrep from the vendored crate with cargo, installing the
// rg binary straight into the binary directory. It is idempotent: when the built rg
// already reports the wanted version the build is skipped, so a repeat bootstrap
// does no heavy work.

// Returns the invocation that builds and installs rg from the vendored crate. cd
// into the crate so its .cargo/config.toml maps crates to the vendor tree;
// --offline/--locked build with no network; --features pcre2 enables the
// look-around/backreference engine, off by default; --root installs it into the
// binary directory (cargo writes to <root>/bin).
func ripgrep_install_invocation(input *Install_Ripgrep_Input) (
	arguments Install_Arguments,
) {
	defer func() {
		Install_Arguments_Invariants(arguments, "ripgrep_install_invocation.arguments")
	}()
	Install_Ripgrep_Input_Invariants(input, "ripgrep_install_invocation.input")
	root := filepath.Dir(string(input.Binary_Directory))
	return Install_Arguments{
		"sh", "-c",
		"cd " + string(input.Ripgrep_Directory) + " && " +
			"cargo install --offline --locked --features pcre2 " +
			"--bin rg --path . --root " + root,
	}
}

// Reports whether the rg binary already installed in the binary directory reports
// the wanted version. Probing the exact path, not PATH, keeps a build present at a
// non-PATH location from being needlessly recompiled.

// Fd_version is the release the bootstrap wants — the prefix of `fd --version`.
// Tracks the vendored third_party/fd Cargo.toml version, bump it with the source.
const FDCLI_VERSION = "fd 10.4.2"

// Install_Fdcli_Input carries the injected dependencies Install_Fdcli needs to build the
// vendored fd with cargo and install it.
type Install_Fdcli_Input struct {
	// Fdcli_Directory is the vendored fd crate cargo builds from; its
	// .cargo/config.toml points the offline build at the committed vendor tree.
	Fdcli_Directory Fdcli_Directory
	// Binary_Directory is where the fd binary is installed and probed. cargo install
	// writes into it via --root (its parent), the same directory the Go tools build
	// into, so no symlink is needed.
	Binary_Directory Binary_Directory
	// IO runs the version gate and the Cargo build.
	IO sysio.IO
	// Logger records setup progress.
	Logger jlog.Logger
	// Stdout receives streamed build output.
	Stdout sysio.Stream
	// Stderr receives streamed build diagnostics.
	Stderr sysio.Stream
}

// Install_Fdcli_Input_Invariants states the fd build dependency set.
func Install_Fdcli_Input_Invariants(
	input *Install_Fdcli_Input, namespace invariant.Namespace,
) {
	Fdcli_Directory_Invariants(input.Fdcli_Directory, namespace)
	Binary_Directory_Invariants(input.Binary_Directory, namespace)
	process_io_requirements(input.IO, namespace)
}

// Install_Fdcli builds fd from the vendored crate with cargo, installing the binary
// straight into the binary directory. It is idempotent: when the built fd already
// reports the wanted version the build is skipped, so a repeat bootstrap does no
// heavy work.

// Returns the invocation that builds and installs fd from the vendored crate. cd
// into the crate so its .cargo/config.toml maps crates to the vendor tree;
// --offline/--locked build with no network against the default features; --root
// installs it into the binary directory (cargo writes to <root>/bin).
func fdcli_install_invocation(input *Install_Fdcli_Input) (arguments Install_Arguments) {
	defer func() {
		Install_Arguments_Invariants(arguments, "fdcli_install_invocation.arguments")
	}()
	Install_Fdcli_Input_Invariants(input, "fdcli_install_invocation.input")
	root := filepath.Dir(string(input.Binary_Directory))
	return Install_Arguments{
		"sh", "-c",
		"cd " + string(input.Fdcli_Directory) + " && " +
			"cargo install --offline --locked --bin fd --path . --root " + root,
	}
}

// Reports whether the fd binary already installed in the binary directory reports
// the wanted version. Probing the exact path, not PATH, keeps a build present at a
// non-PATH location from being needlessly recompiled.

// GHOSTTY_VERSION is the release the bootstrap wants — the prefix of `ghostty
// --version`, whose first line reads "Ghostty <version>". Tracks the pinned DMG
// below; bump it, the url, and the SHA256 together.
const GHOSTTY_VERSION = "Ghostty 1.3.1"

// GHOSTTY_DMG_URL is the pinned macOS DMG the install downloads. Ghostty is a
// notarized app bundle, not buildable source, so it is fetched rather than vendored
// and built; the version lives in the path, so the url always serves that build.
const GHOSTTY_DMG_URL = "https://release.files.ghostty.org/1.3.1/Ghostty.dmg"

// GHOSTTY_DMG_SHA256 is the SHA256 the download is verified against, so a corrupt
// or tampered DMG aborts the install instead of placing bad bytes in the
// applications directory. The published hash for the 1.3.1 DMG; bump it with the url.
const GHOSTTY_DMG_SHA256 = "18cff2b0a6cee90eead9c7d3064e808a252a40baf214aa752c1ecb793b8f5f69"

// GHOSTTY_TEAM_IDENTIFIER is the Apple Developer Team ID Ghostty is signed under,
// pinned in the signature gate so an app signed by anyone else is rejected. Tied to
// the developer's account and stable for years; bump it only if that identity changes.
const GHOSTTY_TEAM_IDENTIFIER = "24VZTF6M5V"

// GHOSTTY_APPLICATION_BINARY_SUBPATH locates the app's command-line binary inside
// the bundle, relative to the applications directory. It is both the gate's probe
// target and the source the PATH symlink points at.
const GHOSTTY_APPLICATION_BINARY_SUBPATH = "Ghostty.app/Contents/MacOS/ghostty"

// Install_Ghostty_Input carries the injected dependencies Install_Ghostty needs to
// download the pinned Ghostty DMG, install the app, and expose its CLI on PATH.
type Install_Ghostty_Input struct {
	// Applications_Directory is the macOS app directory Ghostty.app installs into and
	// the gate probes under. Empty — every non-darwin OS, which has no DMG to install
	// — makes the step do nothing, the same skip an unknown font directory triggers.
	Applications_Directory Applications_Directory
	// Link_Directory is the one directory on PATH the app's ghostty CLI is symlinked
	// into, so `ghostty` works in a terminal as well as via the app. Empty disables
	// the step.
	Link_Directory Ghostty_Link_Directory
	// IO runs the version gate, install script, and link command.
	IO sysio.IO
	// Logger records setup progress.
	Logger jlog.Logger
	// Stdout receives streamed process output.
	Stdout sysio.Stream
	// Stderr receives streamed process diagnostics.
	Stderr sysio.Stream
}

// Install_Ghostty_Input_Invariants states the Ghostty install dependency set.
func Install_Ghostty_Input_Invariants(
	input *Install_Ghostty_Input, namespace invariant.Namespace,
) {
	Applications_Directory_Invariants(input.Applications_Directory, namespace)
	Ghostty_Link_Directory_Invariants(input.Link_Directory, namespace)
	process_io_requirements(input.IO, namespace)
}

// Install_Ghostty installs Ghostty from its pinned DMG into the applications
// directory and symlinks the app's CLI into the PATH directory. It is idempotent:
// when the installed app already reports the wanted version, the download is skipped
// and only the symlink is refreshed, so a repeat bootstrap does no network work.

// REPOSITORY_SUBPATH is the fixed checkout path below the home directory.
const REPOSITORY_SUBPATH = "code/james-orcales"

// DOTFILES_SUBPATH is the dotfiles source path below the home directory.
const DOTFILES_SUBPATH = REPOSITORY_SUBPATH + "/home"

// IOSEVKA_SUBPATH is the vendored font path below the home directory.
const IOSEVKA_SUBPATH = REPOSITORY_SUBPATH + "/third_party/iosevka_nerd_font_mono"

// File_Copy_Input identifies one file copy.
type File_Copy_Input struct {
	// Source is the source file path.
	Source Font_Source_Path
	// Destination is the destination file path.
	Destination Font_Destination_Path
}

// File_Copy_Input_Invariants states both file paths in one copy.
func File_Copy_Input_Invariants(input *File_Copy_Input, namespace invariant.Namespace) {
	Font_Source_Path_Invariants(input.Source, namespace)
	Font_Destination_Path_Invariants(input.Destination, namespace)
}

// Bootstrap_Steps_Input supplies the host facts that define the bootstrap steps.
type Bootstrap_Steps_Input struct {
	// Home_Directory is the destination home directory.
	Home_Directory Home_Directory
	// Operating_System is the runtime operating-system name.
	Operating_System Operating_System
	// Cargo_Directory is the CARGO_HOME value.
	Cargo_Directory Cargo_Directory
	// Data_Directory is the XDG_DATA_HOME value.
	Data_Directory Data_Directory
	// IO supplies every file and process operation.
	IO sysio.IO
	// Logger records setup progress.
	Logger jlog.Logger
	// Stdout receives streamed process output.
	Stdout sysio.Stream
	// Stderr receives streamed process diagnostics.
	Stderr sysio.Stream
}

// Bootstrap_Steps_Input_Invariants states the complete bootstrap dependency set.
func Bootstrap_Steps_Input_Invariants(
	input *Bootstrap_Steps_Input, namespace invariant.Namespace,
) {
	Home_Directory_Invariants(input.Home_Directory, namespace)
	Operating_System_Invariants(input.Operating_System, namespace)
	Cargo_Directory_Invariants(input.Cargo_Directory, namespace)
	Data_Directory_Invariants(input.Data_Directory, namespace)
	file_io_requirements(input.IO, namespace)
	process_io_requirements(input.IO, namespace)
}

// Bootstrap_Steps returns the complete setup policy in execution order.

// Returns the step that builds the vendored direnv binary.

// Returns the step that synchronizes dotfiles and applies macOS defaults.

// Returns the step that copies each absent vendored font.

// Returns the step that builds the vendored Neovim checkout.

// Returns the step that builds the vendored fzf checkout.

// Returns the step that installs the Rust toolchain.

// Returns the step that builds the vendored jj checkout.

// Returns the step that builds the vendored ripgrep checkout.

// Returns the step that builds the vendored fd checkout.

// Returns the step that installs the signed Ghostty application.

// Font_Destination_Input supplies the operating-system font path facts.
type Font_Destination_Input struct {
	// Home_Directory is the user home directory.
	Home_Directory Home_Directory
	// Operating_System is the runtime operating-system name.
	Operating_System Operating_System
	// Data_Directory is the XDG_DATA_HOME value.
	Data_Directory Data_Directory
}

// Font_Destination_Input_Invariants states the host facts for a font path.
func Font_Destination_Input_Invariants(
	input *Font_Destination_Input, namespace invariant.Namespace,
) {
	Home_Directory_Invariants(input.Home_Directory, namespace)
	Operating_System_Invariants(input.Operating_System, namespace)
	Data_Directory_Invariants(input.Data_Directory, namespace)
}

// Returns the user font directory and whether it needs an explicit cache refresh.
func font_destination(input *Font_Destination_Input) (
	directory Font_Directory, refresh_cache Cache_Refresh, supported Font_Support,
) {
	defer func() {
		Font_Directory_Invariants(directory, "font_destination.directory")
		Cache_Refresh_Invariants(refresh_cache, "font_destination.refresh_cache")
		Font_Support_Invariants(supported, "font_destination.supported")
	}()
	Font_Destination_Input_Invariants(input, "font_destination.input")
	switch input.Operating_System {
	case "darwin":
		return Font_Directory(filepath.Join(
			string(input.Home_Directory), "Library", "Fonts")), false, true
	case "linux":
		data_directory := input.Data_Directory
		if data_directory == "" {
			data_directory = Data_Directory(filepath.Join(
				string(input.Home_Directory), ".local", "share"))
		}
		return Font_Directory(filepath.Join(string(data_directory), "fonts")), true, true
	default:
		// A bounded candidate keeps the return contract valid. Font_Support prevents
		// every unsupported host from using it.
		return Font_Directory(filepath.Join(
			string(input.Home_Directory), ".fonts")), false, false
	}
}

// Returns one batched gitignore classifier for directory.

// A batch can exceed one shared text value, so each joined group stays inside the shared
// boundary while the complete process input keeps the setup batch limit.
func check_ignore_input(request *Check_Ignore_Request) {
	Check_Ignore_Request_Invariants(request, "check_ignore_input.request")
	content := []byte{}
	group := shared_strings.Texts{}
	group_size := 0
	flush := func() {
		if len(group) == 0 {
			return
		}
		content = append(content, string(shared_strings.Join(group, "\n"))...)
		content = append(content, '\n')
		group = group[:0]
		group_size = 0
	}
	target_count := 0
	request.Targets(func(target string) (next bool) {
		invariant.Always(
			target_count < PLAN_ENTRY_COUNT_MAX,
			"A check-ignore request has at most PLAN_ENTRY_COUNT_MAX targets.",
		)
		target_count++
		invariant.Always(
			len(target) <= CHECK_IGNORE_TARGET_BYTES_MAX,
			"A check-ignore target fits one source path.",
		)
		separator_size := 0
		if len(group) != 0 {
			separator_size = 1
		}
		if group_size+separator_size+len(target) > shared_strings.TEXT_SIZE_MAXIMUM {
			flush()
		}
		group = append(group, shared_strings.Text(target))
		group_size += len(target)
		if len(group) > 1 {
			group_size++
		}
		return true
	})
	flush()
	invariant.Always(
		len(content) <= CHECK_IGNORE_INPUT_BYTES_MAX,
		"A check-ignore request input fits every bounded target path.",
	)
	request.Process.Input = content
}

// Process output is untrusted, so each newline search reads one bounded part. An overlong line
// cannot become a map key.
func check_ignore_matches(response *Check_Ignore_Result) {
	Check_Ignore_Result_Invariants(response, "check_ignore_matches.response")
	result := response.Process
	printed := map[string]bool{}
	line_start := 0
	search_start := 0
	for search_start < len(result.Output) {
		search_end := search_start + shared_bytes.SLICE_SIZE_MAXIMUM
		if search_end > len(result.Output) {
			search_end = len(result.Output)
		}
		line_index := shared_bytes.Index_Byte(
			shared_bytes.Slice(result.Output[search_start:search_end]), '\n')
		if line_index == shared_bytes.INDEX_ABSENT {
			search_start = search_end
			continue
		}
		line_end := search_start + int(line_index)
		if line_end != line_start {
			if line_end-line_start <= CHECK_IGNORE_TARGET_BYTES_MAX {
				line := string(result.Output[line_start:line_end])
				if response.Target(line) {
					printed[line] = true
				}
			}
		}
		line_start = line_end + 1
		search_start = line_start
	}
	if line_start < len(result.Output) {
		if len(result.Output)-line_start <= CHECK_IGNORE_TARGET_BYTES_MAX {
			line := string(result.Output[line_start:])
			if response.Target(line) {
				printed[line] = true
			}
		}
	}
	invariant.Always(
		len(printed) <= IGNORE_COUNT_MAX,
		"A check-ignore result has at most IGNORE_COUNT_MAX matches.",
	)
	response.Contains = func(path string) (present bool) {
		return printed[path]
	}
}

// GHOSTTY_INSTALL_SCRIPT downloads the pinned DMG, verifies its SHA256, mounts it,
// and replaces the app bundle. %[1]s is the url, %[2]s the SHA256, %[3]s the
// applications directory. set -e aborts on the first failure — a SHA256 mismatch
// included — so a tampered or truncated download never reaches the applications
// directory; the trap detaches the volume and clears the scratch dir on every exit.
const GHOSTTY_INSTALL_SCRIPT = `set -e
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
func ghostty_install_invocation(applications_directory Applications_Directory) (
	arguments Install_Arguments,
) {
	defer func() {
		Install_Arguments_Invariants(arguments, "ghostty_install_invocation.arguments")
	}()
	Applications_Directory_Invariants(
		applications_directory, "ghostty_install_invocation.applications_directory")
	script := fmt.Sprintf(
		GHOSTTY_INSTALL_SCRIPT,
		GHOSTTY_DMG_URL,
		GHOSTTY_DMG_SHA256,
		string(applications_directory),
	)
	return Install_Arguments{"sh", "-c", script}
}

// Reports whether the installed Ghostty app is the wanted version AND still carries
// a verifying code signature. Probing the app's own binary, not PATH, keeps a
// missing symlink from forcing a needless re-download of an app already in place.

// Returns the codesign invocation that verifies the bundle and pins it to Ghostty's
// signing Team ID, so a tampered app or one validly signed by another developer
// fails. The -R requirement's leading = marks it as requirement text, not a file.
func ghostty_codesign_invocation(application Application_Path) (arguments Codesign_Arguments) {
	defer func() {
		Codesign_Arguments_Invariants(arguments, "ghostty_codesign_invocation.arguments")
	}()
	Application_Path_Invariants(application, "ghostty_codesign_invocation.application")
	requirement := `=anchor apple generic and certificate leaf[subject.OU] = "` +
		GHOSTTY_TEAM_IDENTIFIER + `"`
	return Codesign_Arguments{
		"codesign", "--verify", "--deep", "--strict", "-R", requirement,
		string(application),
	}
}

// EFFECTIVE_USER_IDENTIFIER_MAX is the largest host user identifier.
const EFFECTIVE_USER_IDENTIFIER_MAX = 2147483647

// EFFECTIVE_USER_IDENTIFIER_MIN is the root user identity.
const EFFECTIVE_USER_IDENTIFIER_MIN = 0

// DIRECTORY_BYTES_MIN permits an absent optional directory.
const DIRECTORY_BYTES_MIN = 0

// HOME_DIRECTORY_BYTES_MIN rejects a home that cannot name one path component.
const HOME_DIRECTORY_BYTES_MIN = 2

// HOST_DIRECTORY_BYTES_MAX bounds injected home, Cargo, and data directories.
const HOST_DIRECTORY_BYTES_MAX = 64

// REQUIRED_DIRECTORY_BYTES_MIN rejects an absent required host directory.
const REQUIRED_DIRECTORY_BYTES_MIN = 1

// SOURCE_DIRECTORY_BYTES_MIN is the fixed repository suffix below the shortest home.
const SOURCE_DIRECTORY_BYTES_MIN = 26

// SOURCE_DIRECTORY_BYTES_MAX is the fixed repository suffix below the longest home.
const SOURCE_DIRECTORY_BYTES_MAX = 88

// REPOSITORY_DIRECTORY_BYTES_MIN is the checkout suffix below the shortest home.
const REPOSITORY_DIRECTORY_BYTES_MIN = 21

// REPOSITORY_DIRECTORY_BYTES_MAX is the checkout suffix below the longest home.
const REPOSITORY_DIRECTORY_BYTES_MAX = 83

// BINARY_DIRECTORY_BYTES_MIN is the managed binary suffix below the shortest home.
const BINARY_DIRECTORY_BYTES_MIN = 37

// BINARY_DIRECTORY_BYTES_MAX is the managed binary suffix below the longest home.
const BINARY_DIRECTORY_BYTES_MAX = 99

// RUST_LINK_DIRECTORY_BYTES_MIN is the managed link suffix below the shortest home.
const RUST_LINK_DIRECTORY_BYTES_MIN = 32

// RUST_LINK_DIRECTORY_BYTES_MAX is the managed link suffix below the longest home.
const RUST_LINK_DIRECTORY_BYTES_MAX = 94

// GHOSTTY_LINK_DIRECTORY_BYTES_MIN is the Ghostty link suffix below the shortest home.
const GHOSTTY_LINK_DIRECTORY_BYTES_MIN = 37

// GHOSTTY_LINK_DIRECTORY_BYTES_MAX is the Ghostty link suffix below the longest home.
const GHOSTTY_LINK_DIRECTORY_BYTES_MAX = 99

// APPLICATIONS_DIRECTORY_BYTES is the fixed macOS applications path.
const APPLICATIONS_DIRECTORY_BYTES = 13

// DIRENV_DIRECTORY_BYTES_MIN includes the fixed Direnv suffix below the shortest home.
const DIRENV_DIRECTORY_BYTES_MIN = 40

// DIRENV_DIRECTORY_BYTES_MAX includes the fixed Direnv suffix below the longest home.
const DIRENV_DIRECTORY_BYTES_MAX = 102

// FZF_DIRECTORY_BYTES_MIN includes the fixed fzf suffix below the shortest home.
const FZF_DIRECTORY_BYTES_MIN = 37

// FZF_DIRECTORY_BYTES_MAX includes the fixed fzf suffix below the longest home.
const FZF_DIRECTORY_BYTES_MAX = 99

// COMMAND_DIRECTORY_BYTES_MIN is the shortest managed command checkout path.
const COMMAND_DIRECTORY_BYTES_MIN = 26

// COMMAND_DIRECTORY_BYTES_MAX is the longest managed command checkout path.
const COMMAND_DIRECTORY_BYTES_MAX = 99

// PACKAGE_DIRECTORY_BYTES_MIN is the shortest managed command package name.
const PACKAGE_DIRECTORY_BYTES_MIN = 4

// PACKAGE_DIRECTORY_BYTES_MAX is the longest managed command package name.
const PACKAGE_DIRECTORY_BYTES_MAX = 15

// JJ_DIRECTORY_BYTES_MIN includes the fixed jj suffix below the shortest home.
const JJ_DIRECTORY_BYTES_MIN = 36

// JJ_DIRECTORY_BYTES_MAX includes the fixed jj suffix below the longest home.
const JJ_DIRECTORY_BYTES_MAX = 98

// RIPGREP_DIRECTORY_BYTES_MIN includes the fixed ripgrep suffix below the shortest home.
const RIPGREP_DIRECTORY_BYTES_MIN = 41

// RIPGREP_DIRECTORY_BYTES_MAX includes the fixed ripgrep suffix below the longest home.
const RIPGREP_DIRECTORY_BYTES_MAX = 103

// FDCLI_DIRECTORY_BYTES_MIN includes the fixed fd suffix below the shortest home.
const FDCLI_DIRECTORY_BYTES_MIN = 36

// FDCLI_DIRECTORY_BYTES_MAX includes the fixed fd suffix below the longest home.
const FDCLI_DIRECTORY_BYTES_MAX = 98

// OPERATING_SYSTEM_BYTES_MIN is the shortest GOOS name.
const OPERATING_SYSTEM_BYTES_MIN = 3

// OPERATING_SYSTEM_BYTES_MAX is the longest GOOS name.
const OPERATING_SYSTEM_BYTES_MAX = 9

// BYTE_SEQUENCE_COUNT_MIN permits an empty file.
const BYTE_SEQUENCE_COUNT_MIN = 0

// BYTE_COUNT_MIN is the initial byte position.
const BYTE_COUNT_MIN = 0

// BUFFER_SIZE_MIN permits a zero-byte read request.
const BUFFER_SIZE_MIN = 0

// STEP_NAME_BYTES_MIN is the shortest bootstrap label.
const STEP_NAME_BYTES_MIN = 2

// STEP_NAME_BYTES_MAX is the longest bootstrap label.
const STEP_NAME_BYTES_MAX = 8

// COMMAND_NAME_BYTES_MIN is the shortest repository command name.
const COMMAND_NAME_BYTES_MIN = 3

// COMMAND_NAME_BYTES_MAX is the longest repository command name.
const COMMAND_NAME_BYTES_MAX = 7

// MACOS_ARGUMENT_COUNT_MIN is the Finder restart argument.
const MACOS_ARGUMENT_COUNT_MIN = 1

// MACOS_ARGUMENT_COUNT_MAX is the longest defaults command argument list.
const MACOS_ARGUMENT_COUNT_MAX = 6

// PROBE_ARGUMENT_COUNT is the single argument for a version or path probe.
const PROBE_ARGUMENT_COUNT = 1

// INSTALL_ARGUMENT_COUNT is the shell, flag, and script install tuple.
const INSTALL_ARGUMENT_COUNT = 3

// BUILD_INVOCATION_ARGUMENT_COUNT is the common shell build tuple.
const BUILD_INVOCATION_ARGUMENT_COUNT = 3

// SPAWN_ARGUMENT_COUNT_MIN is the shortest streamed shell invocation.
const SPAWN_ARGUMENT_COUNT_MIN = 3

// SPAWN_ARGUMENT_COUNT_MAX is the Ghostty signature invocation.
const SPAWN_ARGUMENT_COUNT_MAX = 7

// CODESIGN_ARGUMENT_COUNT is the complete Ghostty signature invocation.
const CODESIGN_ARGUMENT_COUNT = 7

// MANAGED_EXECUTABLE_BYTES_MIN is the shortest executable path setup probes.
const MANAGED_EXECUTABLE_BYTES_MIN = 11

// MANAGED_EXECUTABLE_BYTES_MAX is the longest executable path setup probes.
const MANAGED_EXECUTABLE_BYTES_MAX = 106

// VERSIONED_EXECUTABLE_BYTES_MIN is the shortest managed build destination.
const VERSIONED_EXECUTABLE_BYTES_MIN = BINARY_DIRECTORY_BYTES_MIN + 3

// PROBE_PATH_BYTES_MIN is the shortest command name that setup probes.
const PROBE_PATH_BYTES_MIN = 5

// PROBE_PATH_BYTES_MAX is the longest executable path that setup probes.
const PROBE_PATH_BYTES_MAX = 106

// VERSION_PREFIX_BYTES_MIN is the shortest pinned version prefix.
const VERSION_PREFIX_BYTES_MIN = 6

// VERSION_PREFIX_BYTES_MAX is the longest pinned version prefix.
const VERSION_PREFIX_BYTES_MAX = 14

// COMMAND_OUTPUT_BYTES_MAX is the longest bounded probe output.
const COMMAND_OUTPUT_BYTES_MAX = 104

// VERSION_OUTPUT_BYTES_MAX is the longest Neovim version output under test.
const VERSION_OUTPUT_BYTES_MAX = 12

// APPLICATION_PATH_BYTES is the fixed Ghostty application path.
const APPLICATION_PATH_BYTES = 25

// RELATIVE_FILE_PATH_BYTES_MIN is the shortest simulated source entry.
const RELATIVE_FILE_PATH_BYTES_MIN = 2

// RELATIVE_FILE_PATH_BYTES_MAX leaves room for the validated source prefix.
const RELATIVE_FILE_PATH_BYTES_MAX = 4000

// TRAVERSAL_DIRECTORY_BYTES_MIN is the source root marker.
const TRAVERSAL_DIRECTORY_BYTES_MIN = 1

// TRAVERSAL_DIRECTORY_BYTES_MAX leaves room for the validated source prefix.
const TRAVERSAL_DIRECTORY_BYTES_MAX = 4000

// MIRROR_PATH_BYTES_MIN is the shortest destination path in a valid plan.
const MIRROR_PATH_BYTES_MIN = 5

// MIRROR_PATH_BYTES_MAX joins the longest home and relative path.
const MIRROR_PATH_BYTES_MAX = 4065

// DESTINATION_PATH_BYTES_MIN is the shortest candidate mirror destination.
const DESTINATION_PATH_BYTES_MIN = 5

// DESTINATION_PATH_BYTES_MAX joins the longest home and relative path.
const DESTINATION_PATH_BYTES_MAX = 4065

// FILE_PATH_BYTES_MIN rejects a path that cannot name one absolute file.
const FILE_PATH_BYTES_MIN = DESTINATION_PATH_BYTES_MIN

// FILE_PATH_BYTES_MAX admits the longest validated source path submitted to shared IO.
const FILE_PATH_BYTES_MAX = CHECK_IGNORE_TARGET_BYTES_MAX

// FONT_PATH_BYTES_MIN is the shortest generated font destination.
const FONT_PATH_BYTES_MIN = 36

// FONT_PATH_BYTES_MAX is the longest generated font destination.
const FONT_PATH_BYTES_MAX = 119

// FONT_SOURCE_PATH_BYTES_MIN is the shortest vendored font path.
const FONT_SOURCE_PATH_BYTES_MIN = 85

// FONT_SOURCE_PATH_BYTES_MAX is the longest vendored font path.
const FONT_SOURCE_PATH_BYTES_MAX = 154

// FONT_DIRECTORY_BYTES_MIN is the shortest supported user font directory.
const FONT_DIRECTORY_BYTES_MIN = 7

// FONT_DIRECTORY_BYTES_MAX is the longest user font directory derived from host input.
const FONT_DIRECTORY_BYTES_MAX = 83

// STEP_COUNT_MIN permits an empty bootstrap plan.
const STEP_COUNT_MIN = 0

// COMMAND_COUNT_MIN permits an empty macOS command list.
const COMMAND_COUNT_MIN = 0

// WRITE_COUNT_MAX bounds one mirror plan before writes start.
const WRITE_COUNT_MAX = 4096

// TRAVERSAL_DIRECTORY_COUNT_MAX bounds one complete source traversal.
const TRAVERSAL_DIRECTORY_COUNT_MAX = 4096

// PLAN_ENTRY_COUNT_MAX bounds one classified traversal level.
const PLAN_ENTRY_COUNT_MAX = 4096

// IGNORE_COUNT_MAX bounds ignored entries returned for one traversal level.
const IGNORE_COUNT_MAX = 4096

// CHECK_IGNORE_TARGET_BYTES_MAX joins the longest source directory and relative path.
const CHECK_IGNORE_TARGET_BYTES_MAX = SOURCE_DIRECTORY_BYTES_MAX + 1 +
	RELATIVE_FILE_PATH_BYTES_MAX

// CHECK_IGNORE_INPUT_BYTES_MAX includes one newline after each longest target path.
const CHECK_IGNORE_INPUT_BYTES_MAX = PLAN_ENTRY_COUNT_MAX * (CHECK_IGNORE_TARGET_BYTES_MAX + 1)

// INVOCATION_COUNT_MIN permits an empty invocation list.
const INVOCATION_COUNT_MIN = 0

// FILE_PATH_COUNT_MIN permits an empty file path list.
const FILE_PATH_COUNT_MIN = 0

// BOOTSTRAP_STEP_COUNT is the complete ordered setup policy.
const BOOTSTRAP_STEP_COUNT = 14

// MACOS_COMMAND_COUNT is the complete defaults command set.
const MACOS_COMMAND_COUNT = 36

// IOSEVKA_FONT_COUNT is the complete vendored font set.
const IOSEVKA_FONT_COUNT = 4

// NEOVIM_INVOCATION_COUNT is the configure and install pair.
const NEOVIM_INVOCATION_COUNT = 2

// Exit_Code is one setup process result.
type Exit_Code uint8

// Exit_Code_Invariants limits setup results to the declared process results.
func Exit_Code_Invariants(code Exit_Code, namespace invariant.Namespace) {
	invariant.Tree(code, namespace).Enum_3_Uint8(
		uint8(code), uint8(EXIT_SUCCESS), uint8(EXIT_FAILURE), uint8(EXIT_USAGE)).
		Ensure()
}

// Console_Color is the terminal color capability.
type Console_Color bool

// Console_Color_Invariants states both terminal color states.
func Console_Color_Invariants(color Console_Color, namespace invariant.Namespace) {
	invariant.Tree(color, namespace).
		Sometimes(bool(color), "The console supports color.").
		Ensure()
}

// Environment_Valid reports whether the host facts permit setup to start.
type Environment_Valid bool

// Environment_Valid_Invariants states both environment validation results.
func Environment_Valid_Invariants(valid Environment_Valid, namespace invariant.Namespace) {
	invariant.Tree(valid, namespace).
		Sometimes(bool(valid), "The setup environment is valid.").
		Ensure()
}

// Step_Success reports whether one setup step completed its contract.
type Step_Success bool

// Step_Success_Invariants states both setup step results.
func Step_Success_Invariants(succeeded Step_Success, namespace invariant.Namespace) {
	invariant.Tree(succeeded, namespace).
		Sometimes(bool(succeeded), "A setup step succeeds.").
		Ensure()
}

// Effective_User_Identifier is the operating-system user identity.
type Effective_User_Identifier int

// Effective_User_Identifier_Invariants rejects identities outside the host domain.
func Effective_User_Identifier_Invariants(
	identifier Effective_User_Identifier, namespace invariant.Namespace,
) {
	invariant.Tree(identifier, namespace).Range_Int(
		int(identifier), EFFECTIVE_USER_IDENTIFIER_MIN,
		EFFECTIVE_USER_IDENTIFIER_MAX).Ensure()
}

// Directory is one absolute or relative file-system directory.
type Directory string

// Directory_Invariants bounds a directory before a file-system operation uses it.
func Directory_Invariants(directory Directory, namespace invariant.Namespace) {
	invariant.Tree(directory, namespace).
		Range_Int(len(directory), SOURCE_DIRECTORY_BYTES_MIN, SOURCE_DIRECTORY_BYTES_MAX).
		Ensure()
}

// Home_Directory is the user home that receives setup files.
type Home_Directory string

// Home_Directory_Invariants bounds the user home path.
func Home_Directory_Invariants(directory Home_Directory, namespace invariant.Namespace) {
	invariant.Tree(directory, namespace).
		Range_Int(len(directory), HOME_DIRECTORY_BYTES_MIN, HOST_DIRECTORY_BYTES_MAX).
		Ensure()
}

// Cargo_Directory is the Rust tool home.
type Cargo_Directory string

// Cargo_Directory_Invariants bounds the Rust tool home path.
func Cargo_Directory_Invariants(directory Cargo_Directory, namespace invariant.Namespace) {
	invariant.Tree(directory, namespace).
		Range_Int(len(directory), DIRECTORY_BYTES_MIN, HOST_DIRECTORY_BYTES_MAX).
		Ensure()
}

// Required_Cargo_Directory is a configured Cargo home after the empty gate.
type Required_Cargo_Directory string

// Required_Cargo_Directory_Invariants bounds a configured Cargo home.
func Required_Cargo_Directory_Invariants(
	directory Required_Cargo_Directory, namespace invariant.Namespace,
) {
	invariant.Tree(directory, namespace).
		Range_Int(len(directory), REQUIRED_DIRECTORY_BYTES_MIN, HOST_DIRECTORY_BYTES_MAX).
		Ensure()
}

// Data_Directory is the operating-system data home.
type Data_Directory string

// Data_Directory_Invariants bounds the data home path.
func Data_Directory_Invariants(directory Data_Directory, namespace invariant.Namespace) {
	invariant.Tree(directory, namespace).
		Range_Int(len(directory), DIRECTORY_BYTES_MIN, HOST_DIRECTORY_BYTES_MAX).
		Ensure()
}

// Source_Directory is the root of one mirror source tree.
type Source_Directory string

// Source_Directory_Invariants bounds a mirror source path.
func Source_Directory_Invariants(directory Source_Directory, namespace invariant.Namespace) {
	invariant.Tree(directory, namespace).
		Range_Int(len(directory), SOURCE_DIRECTORY_BYTES_MIN, SOURCE_DIRECTORY_BYTES_MAX).
		Ensure()
}

// Destination_Directory is the root of one mirror destination tree.
type Destination_Directory string

// Destination_Directory_Invariants bounds a mirror destination path.
func Destination_Directory_Invariants(
	directory Destination_Directory, namespace invariant.Namespace,
) {
	invariant.Tree(directory, namespace).
		Range_Int(len(directory), HOME_DIRECTORY_BYTES_MIN, HOST_DIRECTORY_BYTES_MAX).
		Ensure()
}

// Repository_Directory is the setup source checkout.
type Repository_Directory string

// Repository_Directory_Invariants bounds the setup checkout path.
func Repository_Directory_Invariants(
	directory Repository_Directory, namespace invariant.Namespace,
) {
	invariant.Tree(directory, namespace).
		Range_Int(
			len(directory), REPOSITORY_DIRECTORY_BYTES_MIN,
			REPOSITORY_DIRECTORY_BYTES_MAX).
		Ensure()
}

// Font_Directory is the operating-system font destination.
type Font_Directory string

// Font_Directory_Invariants bounds the font destination path.
func Font_Directory_Invariants(directory Font_Directory, namespace invariant.Namespace) {
	invariant.Tree(directory, namespace).
		Range_Int(len(directory), FONT_DIRECTORY_BYTES_MIN, FONT_DIRECTORY_BYTES_MAX).
		Ensure()
}

// Direnv_Directory is the checkout that builds the Direnv command.
type Direnv_Directory string

// Direnv_Directory_Invariants bounds the path derived from the validated home.
func Direnv_Directory_Invariants(
	directory Direnv_Directory, namespace invariant.Namespace,
) {
	invariant.Tree(directory, namespace).
		Range_Int(len(directory), DIRENV_DIRECTORY_BYTES_MIN, DIRENV_DIRECTORY_BYTES_MAX).
		Ensure()
}

// Fzf_Directory is the checkout that installs fzf.
type Fzf_Directory string

// Fzf_Directory_Invariants bounds the path derived from the validated home.
func Fzf_Directory_Invariants(directory Fzf_Directory, namespace invariant.Namespace) {
	invariant.Tree(directory, namespace).
		Range_Int(len(directory), FZF_DIRECTORY_BYTES_MIN, FZF_DIRECTORY_BYTES_MAX).
		Ensure()
}

// Command_Directory is the checkout that contains a managed command package.
type Command_Directory string

// Command_Directory_Invariants bounds a checkout path derived from the validated home.
func Command_Directory_Invariants(
	directory Command_Directory, namespace invariant.Namespace,
) {
	invariant.Tree(directory, namespace).
		Range_Int(len(directory), COMMAND_DIRECTORY_BYTES_MIN, COMMAND_DIRECTORY_BYTES_MAX).
		Ensure()
}

// Package_Directory identifies one package inside a managed command checkout.
type Package_Directory string

// Package_Directory_Invariants bounds the closed set of package-name lengths.
func Package_Directory_Invariants(
	directory Package_Directory, namespace invariant.Namespace,
) {
	invariant.Tree(directory, namespace).
		Range_Int(len(directory), PACKAGE_DIRECTORY_BYTES_MIN, PACKAGE_DIRECTORY_BYTES_MAX).
		Ensure()
}

// Jj_Directory is the checkout that builds jj.
type Jj_Directory string

// Jj_Directory_Invariants bounds the path derived from the validated home.
func Jj_Directory_Invariants(directory Jj_Directory, namespace invariant.Namespace) {
	invariant.Tree(directory, namespace).
		Range_Int(len(directory), JJ_DIRECTORY_BYTES_MIN, JJ_DIRECTORY_BYTES_MAX).
		Ensure()
}

// Ripgrep_Directory is the checkout that builds ripgrep.
type Ripgrep_Directory string

// Ripgrep_Directory_Invariants bounds the path derived from the validated home.
func Ripgrep_Directory_Invariants(
	directory Ripgrep_Directory, namespace invariant.Namespace,
) {
	invariant.Tree(directory, namespace).
		Range_Int(len(directory), RIPGREP_DIRECTORY_BYTES_MIN, RIPGREP_DIRECTORY_BYTES_MAX).
		Ensure()
}

// Fdcli_Directory is the checkout that builds fd.
type Fdcli_Directory string

// Fdcli_Directory_Invariants bounds the path derived from the validated home.
func Fdcli_Directory_Invariants(directory Fdcli_Directory, namespace invariant.Namespace) {
	invariant.Tree(directory, namespace).
		Range_Int(len(directory), FDCLI_DIRECTORY_BYTES_MIN, FDCLI_DIRECTORY_BYTES_MAX).
		Ensure()
}

// Binary_Directory is the managed command destination.
type Binary_Directory string

// Binary_Directory_Invariants bounds a managed command path.
func Binary_Directory_Invariants(directory Binary_Directory, namespace invariant.Namespace) {
	invariant.Tree(directory, namespace).
		Range_Int(len(directory), BINARY_DIRECTORY_BYTES_MIN, BINARY_DIRECTORY_BYTES_MAX).
		Ensure()
}

// Rust_Link_Directory is the managed Rust symbolic-link destination.
type Rust_Link_Directory string

// Rust_Link_Directory_Invariants bounds the Rust link destination.
func Rust_Link_Directory_Invariants(
	directory Rust_Link_Directory, namespace invariant.Namespace,
) {
	invariant.Tree(directory, namespace).
		Range_Int(
			len(directory), RUST_LINK_DIRECTORY_BYTES_MIN,
			RUST_LINK_DIRECTORY_BYTES_MAX).
		Ensure()
}

// Ghostty_Link_Directory is the managed Ghostty symbolic-link destination.
type Ghostty_Link_Directory string

// Ghostty_Link_Directory_Invariants bounds the Ghostty link destination.
func Ghostty_Link_Directory_Invariants(
	directory Ghostty_Link_Directory, namespace invariant.Namespace,
) {
	invariant.Tree(directory, namespace).
		Range_Int(
			len(directory), GHOSTTY_LINK_DIRECTORY_BYTES_MIN,
			GHOSTTY_LINK_DIRECTORY_BYTES_MAX).
		Ensure()
}

// Applications_Directory is the macOS application destination.
type Applications_Directory string

// Applications_Directory_Invariants bounds the macOS application path.
func Applications_Directory_Invariants(
	directory Applications_Directory, namespace invariant.Namespace,
) {
	invariant.Always(
		len(directory) == APPLICATIONS_DIRECTORY_BYTES,
		"Ghostty uses the macOS applications directory.")
}

// Operating_System is one Go operating-system name.
type Operating_System string

// Operating_System_Invariants bounds an operating-system name.
func Operating_System_Invariants(system Operating_System, namespace invariant.Namespace) {
	invariant.Tree(system, namespace).
		Range_Int(len(system), OPERATING_SYSTEM_BYTES_MIN, OPERATING_SYSTEM_BYTES_MAX).
		Ensure()
}

// Dotfile_Bytes is one bounded configuration-file payload.
type Dotfile_Bytes []byte

// Dotfile_Bytes_Invariants applies the stricter configuration-file limit.
func Dotfile_Bytes_Invariants(contents Dotfile_Bytes, namespace invariant.Namespace) {
	invariant.Tree(contents, namespace).
		Range_Int(len(contents), BYTE_SEQUENCE_COUNT_MIN, DOTFILE_PAYLOAD_BYTES_MAX).
		Ensure()
}

// Check_Ignore_Request keeps the large process input behind its bounded serializer.
type Check_Ignore_Request struct {
	// Targets yields each path without exposing the source collection to the serializer.
	Targets func(yield func(target string) (next bool))
	// Process prevents the spawn boundary from repeating the serializer policy.
	Process sysio.Process_Request
}

// Check_Ignore_Request_Invariants prevents a nil carrier from bypassing input validation.
func Check_Ignore_Request_Invariants(
	request *Check_Ignore_Request, _ invariant.Namespace,
) {
	invariant.Always(request != nil, "A check-ignore request is present.")
	invariant.Always(request.Targets != nil, "A check-ignore target sequence is present.")
}

// Check_Ignore_Result keeps untrusted process output behind its bounded parser.
type Check_Ignore_Result struct {
	// Process stays unavailable to policy until the parser validates each printed line.
	Process sysio.Process_Result
	// Target rejects output that did not originate in the bounded request.
	Target func(path string) (present bool)
	// Contains exposes only lines that meet the count and path limits.
	Contains func(path string) (present bool)
}

// Check_Ignore_Result_Invariants prevents a nil carrier from bypassing output validation.
func Check_Ignore_Result_Invariants(response *Check_Ignore_Result, _ invariant.Namespace) {
	invariant.Always(response != nil, "A check-ignore result is present.")
	invariant.Always(response.Target != nil, "A check-ignore target predicate is present.")
}

// File_Presence reports whether one file exists.
type File_Presence bool

// File_Presence_Invariants states both file presence results.
func File_Presence_Invariants(present File_Presence, namespace invariant.Namespace) {
	invariant.Tree(present, namespace).
		Sometimes(bool(present), "A setup file is present.").
		Ensure()
}

// Font_Path is one generated font destination path.
type Font_Path string

// Font_Path_Invariants bounds paths passed to the font presence check.
func Font_Path_Invariants(path Font_Path, namespace invariant.Namespace) {
	invariant.Tree(path, namespace).
		Range_Int(len(path), FONT_PATH_BYTES_MIN, FONT_PATH_BYTES_MAX).
		Ensure()
}

// Relative_File_Path is one path below the mirror source root.
type Relative_File_Path string

// Relative_File_Path_Invariants bounds paths emitted by one simulated tree.
func Relative_File_Path_Invariants(
	path Relative_File_Path, namespace invariant.Namespace,
) {
	invariant.Tree(path, namespace).
		Range_Int(
			len(path), RELATIVE_FILE_PATH_BYTES_MIN, RELATIVE_FILE_PATH_BYTES_MAX).
		Ensure()
}

// Traversal_Directory is one source-relative directory being scanned.
type Traversal_Directory string

// Traversal_Directory_Invariants bounds paths queued by the breadth-first walk.
func Traversal_Directory_Invariants(
	directory Traversal_Directory, namespace invariant.Namespace,
) {
	invariant.Tree(directory, namespace).
		Range_Int(
			len(directory), TRAVERSAL_DIRECTORY_BYTES_MIN,
			TRAVERSAL_DIRECTORY_BYTES_MAX).
		Ensure()
}

// Mirror_Path is an existing or proposed destination under the validated home.
type Mirror_Path string

// Mirror_Path_Invariants bounds paths read during destination comparison.
func Mirror_Path_Invariants(path Mirror_Path, namespace invariant.Namespace) {
	invariant.Tree(path, namespace).
		Range_Int(len(path), MIRROR_PATH_BYTES_MIN, MIRROR_PATH_BYTES_MAX).
		Ensure()
}

// Managed_Executable is the absolute path of one installed command.
type Managed_Executable string

// Managed_Executable_Invariants bounds paths derived from setup destinations.
func Managed_Executable_Invariants(
	executable Managed_Executable, namespace invariant.Namespace,
) {
	invariant.Tree(executable, namespace).
		Range_Int(
			len(executable), MANAGED_EXECUTABLE_BYTES_MIN,
			MANAGED_EXECUTABLE_BYTES_MAX).
		Ensure()
}

// Versioned_Executable is one managed executable produced by the common build gate.
type Versioned_Executable string

// Versioned_Executable_Invariants bounds common build destinations.
func Versioned_Executable_Invariants(
	executable Versioned_Executable, namespace invariant.Namespace,
) {
	invariant.Tree(executable, namespace).
		Range_Int(
			len(executable), VERSIONED_EXECUTABLE_BYTES_MIN,
			MANAGED_EXECUTABLE_BYTES_MAX).
		Ensure()
}

// Probe_Path is one command name or absolute executable path.
type Probe_Path string

// Probe_Path_Invariants bounds every path passed to a captured probe.
func Probe_Path_Invariants(path Probe_Path, namespace invariant.Namespace) {
	invariant.Tree(path, namespace).
		Range_Int(len(path), PROBE_PATH_BYTES_MIN, PROBE_PATH_BYTES_MAX).
		Ensure()
}

// Application_Path is the fixed Ghostty application bundle path.
type Application_Path string

// Application_Path_Invariants pins signature checks to the expected bundle.
func Application_Path_Invariants(path Application_Path, namespace invariant.Namespace) {
	invariant.Always(
		len(path) == APPLICATION_PATH_BYTES,
		"The signature check uses the Ghostty application path.")
}

// Font_Source_Path is one vendored font file.
type Font_Source_Path string

// Font_Source_Path_Invariants bounds paths derived from the vendored font root.
func Font_Source_Path_Invariants(path Font_Source_Path, namespace invariant.Namespace) {
	invariant.Tree(path, namespace).
		Range_Int(len(path), FONT_SOURCE_PATH_BYTES_MIN, FONT_SOURCE_PATH_BYTES_MAX).
		Ensure()
}

// Font_Destination_Path is one copied font destination.
type Font_Destination_Path string

// Font_Destination_Path_Invariants bounds generated font destination paths.
func Font_Destination_Path_Invariants(
	path Font_Destination_Path, namespace invariant.Namespace,
) {
	invariant.Tree(path, namespace).
		Range_Int(len(path), FONT_PATH_BYTES_MIN, FONT_PATH_BYTES_MAX).
		Ensure()
}

// Destination_Path is one candidate mirror destination.
type Destination_Path string

// Destination_Path_Invariants bounds a path before a plan returns it.
func Destination_Path_Invariants(path Destination_Path, namespace invariant.Namespace) {
	invariant.Tree(path, namespace).
		Range_Int(
			len(path), DESTINATION_PATH_BYTES_MIN, DESTINATION_PATH_BYTES_MAX).
		Ensure()
}

// Step_Name is one bootstrap step label.
type Step_Name string

// Step_Name_Invariants bounds a bootstrap step label.
func Step_Name_Invariants(name Step_Name, namespace invariant.Namespace) {
	invariant.Tree(name, namespace).
		Range_Int(len(name), STEP_NAME_BYTES_MIN, STEP_NAME_BYTES_MAX).
		Ensure()
}

// Command_Name is one repository command name used by its PATH gate.
type Command_Name string

// Command_Name_Invariants bounds the four repository command names.
func Command_Name_Invariants(name Command_Name, namespace invariant.Namespace) {
	invariant.Tree(name, namespace).
		Range_Int(len(name), COMMAND_NAME_BYTES_MIN, COMMAND_NAME_BYTES_MAX).
		Ensure()
}

// Macos_Arguments is one ordered defaults or Finder argument list.
type Macos_Arguments []string

// Macos_Arguments_Invariants bounds the parsed macOS policy commands.
func Macos_Arguments_Invariants(arguments Macos_Arguments, namespace invariant.Namespace) {
	invariant.Tree(arguments, namespace).
		Range_Int(len(arguments), MACOS_ARGUMENT_COUNT_MIN, MACOS_ARGUMENT_COUNT_MAX).
		Ensure()
}

// Probe_Arguments is the single option or command name for a captured probe.
type Probe_Arguments []string

// Probe_Arguments_Invariants requires one bounded probe argument.
func Probe_Arguments_Invariants(arguments Probe_Arguments, namespace invariant.Namespace) {
	invariant.Always(
		len(arguments) == PROBE_ARGUMENT_COUNT, "A probe has one argument.")
}

// Install_Arguments is one shell command that performs an installation.
type Install_Arguments []string

// Install_Arguments_Invariants requires the shell, flag, and script tuple.
func Install_Arguments_Invariants(arguments Install_Arguments, namespace invariant.Namespace) {
	invariant.Always(
		len(arguments) == INSTALL_ARGUMENT_COUNT, "An install uses one shell script.")
}

// Spawn_Arguments is one streamed process invocation, including its command.
type Spawn_Arguments []string

// Spawn_Arguments_Invariants bounds streamed setup process invocations.
func Spawn_Arguments_Invariants(arguments Spawn_Arguments, namespace invariant.Namespace) {
	invariant.Tree(arguments, namespace).
		Range_Int(len(arguments), SPAWN_ARGUMENT_COUNT_MIN, SPAWN_ARGUMENT_COUNT_MAX).
		Ensure()
}

// Build_Invocation is the shell, flag, and script used by a common versioned build.
type Build_Invocation []string

// Build_Invocation_Invariants requires the complete shell tuple.
func Build_Invocation_Invariants(arguments Build_Invocation, _ invariant.Namespace) {
	invariant.Always(
		len(arguments) == BUILD_INVOCATION_ARGUMENT_COUNT,
		"A versioned build uses one shell script.",
	)
}

// Codesign_Arguments is the complete Ghostty signature check.
type Codesign_Arguments []string

// Codesign_Arguments_Invariants requires every signature restriction.
func Codesign_Arguments_Invariants(arguments Codesign_Arguments, namespace invariant.Namespace) {
	invariant.Always(
		len(arguments) == CODESIGN_ARGUMENT_COUNT,
		"The signature check has every required restriction.")
}

// Commands is one ordered macOS command list.
type Commands []Macos_Command

// Commands_Invariants bounds one macOS command list.
func Commands_Invariants(commands Commands, namespace invariant.Namespace) {
	invariant.Always(
		len(commands) == MACOS_COMMAND_COUNT, "macOS has every setup command.")
}

// Writes is one ordered mirror write plan.
type Writes struct {
	// List retains the complete plan before Mirror performs its first write.
	List list.List
}

// Writes_Invariants states empty and non-empty mirror plans.
func Writes_Invariants(writes Writes, namespace invariant.Namespace) {
	invariant.Tree(writes, namespace).
		Sometimes(writes.List.Len() != 0, "A mirror plan has pending writes.").
		Ensure()
}

// Plan_Decision reports whether a source file needs a write.
type Plan_Decision bool

// Plan_Decision_Invariants states both mirror plan decisions.
func Plan_Decision_Invariants(planned Plan_Decision, namespace invariant.Namespace) {
	invariant.Tree(planned, namespace).
		Sometimes(bool(planned), "A source file needs a mirror write.").
		Ensure()
}

// Version_Prefix is one pinned tool version used by an install gate.
type Version_Prefix string

// Version_Prefix_Invariants bounds the versions pinned by setup.
func Version_Prefix_Invariants(value Version_Prefix, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), VERSION_PREFIX_BYTES_MIN, VERSION_PREFIX_BYTES_MAX).
		Ensure()
}

// Command_Output is captured and trimmed probe output.
type Command_Output string

// Command_Output_Invariants bounds output before setup parses it.
func Command_Output_Invariants(value Command_Output, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), BYTE_SEQUENCE_COUNT_MIN, COMMAND_OUTPUT_BYTES_MAX).
		Ensure()
}

// Version_Output is the first bounded Neovim version report.
type Version_Output string

// Version_Output_Invariants bounds text parsed by the Neovim version gate.
func Version_Output_Invariants(value Version_Output, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), BYTE_SEQUENCE_COUNT_MIN, VERSION_OUTPUT_BYTES_MAX).
		Ensure()
}

// Command_Success reports whether one command exits successfully.
type Command_Success bool

// Command_Success_Invariants states both command results.
func Command_Success_Invariants(success Command_Success, namespace invariant.Namespace) {
	invariant.Tree(success, namespace).
		Sometimes(bool(success), "A setup command succeeds.").
		Ensure()
}

// Invocations is one ordered command invocation list.
type Invocations [][]string

// Invocations_Invariants bounds one command invocation list.
func Invocations_Invariants(invocations Invocations, namespace invariant.Namespace) {
	invariant.Always(
		len(invocations) == NEOVIM_INVOCATION_COUNT, "Neovim has both build invocations.")
}

// File_Paths is one ordered file path list.
type File_Paths []string

// File_Paths_Invariants bounds one file path list.
func File_Paths_Invariants(paths File_Paths, namespace invariant.Namespace) {
	invariant.Always(len(paths) == IOSEVKA_FONT_COUNT, "Iosevka has every setup font.")
}

// Cache_Refresh reports whether a font cache needs an update.
type Cache_Refresh bool

// Cache_Refresh_Invariants states both font cache results.
func Cache_Refresh_Invariants(refresh Cache_Refresh, namespace invariant.Namespace) {
	invariant.Tree(refresh, namespace).
		Sometimes(bool(refresh), "A font installation refreshes the cache.").
		Ensure()
}

// Font_Support reports whether the host has a managed font destination.
type Font_Support bool

// Font_Support_Invariants states supported and unsupported host results.
func Font_Support_Invariants(supported Font_Support, namespace invariant.Namespace) {
	invariant.Tree(supported, namespace).
		Sometimes(bool(supported), "The host has a managed font destination.").
		Ensure()
}

// Runner exposes setup progress without giving policy code the event-loop driver.
type Runner struct {
	// Rearm submits one continuation that a prior completion recorded.
	Rearm func() (armed Runner_Work_Queued)
	// Work_Queued reports whether Rearm can submit a continuation.
	Work_Queued func() (queued Runner_Work_Queued)
	// Stopped reports whether setup reached a terminal status.
	Stopped func() (stopped Runner_Stopped)
	// Status returns the terminal process status.
	Status func() (status Exit_Code)
}

// Runner_Invariants states the complete root control surface.
func Runner_Invariants(runner Runner, _ invariant.Namespace) {
	invariant.Always(runner.Rearm != nil, "A setup runner has a rearm operation.")
	invariant.Always(
		runner.Work_Queued != nil, "A setup runner reports its queued work.")
	invariant.Always(runner.Stopped != nil, "A setup runner reports its terminal state.")
	invariant.Always(runner.Status != nil, "A setup runner reports its process status.")
}

// Runner_Work_Queued reports whether one continuation is ready for root submission.
type Runner_Work_Queued bool

// Runner_Work_Queued_Invariants requires both queue states across the simulation sweep.
func Runner_Work_Queued_Invariants(
	queued Runner_Work_Queued, namespace invariant.Namespace,
) {
	invariant.Tree(queued, namespace).
		Sometimes(bool(queued), "A setup runner has queued work.").
		Ensure()
}

// Runner_Stopped reports whether setup has one terminal exit status.
type Runner_Stopped bool

// Runner_Stopped_Invariants requires active and terminal runner states across the sweep.
func Runner_Stopped_Invariants(stopped Runner_Stopped, namespace invariant.Namespace) {
	invariant.Tree(stopped, namespace).
		Sometimes(bool(stopped), "A setup runner has stopped.").
		Ensure()
}

// Runner_State retains one bounded continuation, armed submissions, and the terminal result.
type Runner_State struct {
	// Action is the one continuation that the root can submit next.
	Action func()
	// Operations retains each submission until its callback retires.
	Operations *list.List
	// Terminal returns the exit status after setup stops.
	Terminal func() (status Exit_Code)
}

// Runner_State_Invariants requires one private state allocation.
func Runner_State_Invariants(state *Runner_State, _ invariant.Namespace) {
	invariant.Always(state != nil, "A setup runner has private state.")
	invariant.Always(state.Operations != nil, "A setup runner tracks armed IO operations.")
}

// New_runner returns the root surface for one private continuation state.
func new_runner() (state *Runner_State, runner Runner) {
	defer func() {
		Runner_State_Invariants(state, "new_runner.state")
		Runner_Invariants(runner, "new_runner.runner")
	}()
	state = &Runner_State{Operations: list.New()}
	runner = Runner{
		Rearm: func() (armed Runner_Work_Queued) {
			return runner_rearm(state)
		},
		Work_Queued: func() (queued Runner_Work_Queued) {
			return state.Action != nil
		},
		Stopped: func() (stopped Runner_Stopped) {
			return state.Terminal != nil && state.Operations.Len() == 0
		},
		Status: func() (status Exit_Code) {
			if state.Terminal == nil {
				return EXIT_FAILURE
			}
			return state.Terminal()
		},
	}
	return state, runner
}

// Runner_rearm removes one recorded continuation before it submits that continuation.
func runner_rearm(state *Runner_State) (armed Runner_Work_Queued) {
	defer func() {
		Runner_Work_Queued_Invariants(armed, "runner_rearm.armed")
	}()
	Runner_State_Invariants(state, "runner_rearm.state")
	if state.Action == nil {
		return false
	}
	action := state.Action
	state.Action = nil
	action()
	return true
}

// Runner_queue records one continuation after pure work or an IO callback completes.
func runner_queue(state *Runner_State, action func()) {
	Runner_State_Invariants(state, "runner_queue.state")
	invariant.Always(state.Terminal == nil, "A stopped setup runner queues no work.")
	invariant.Always(state.Action == nil, "A setup runner does not replace queued work.")
	invariant.Always(action != nil, "A setup runner queues a nonnil continuation.")
	state.Action = action
}

// Runner_operation_start keeps the root active until one submitted callback retires.
func runner_operation_start(state *Runner_State) (operation *list.Element) {
	Runner_State_Invariants(state, "runner_operation_start.state")
	invariant.Always(state.Terminal == nil, "A stopped setup runner submits no IO.")
	return state.Operations.PushBack(struct{}{})
}

// Runner_complete_io joins one submission before it transfers control back to the root.
func runner_complete_io(
	state *Runner_State, operation *list.Element, action func(),
) {
	Runner_State_Invariants(state, "runner_complete_io.state")
	invariant.Always(operation != nil, "An IO callback retires one tracked operation.")
	invariant.Always(action != nil, "An IO callback has a nonnil continuation.")
	state.Operations.Remove(operation)
	if state.Terminal != nil {
		return
	}
	runner_queue(state, action)
}

// Runner_stop records one terminal result and removes no work because callers stop in sequence.
func runner_stop(state *Runner_State, status Exit_Code) {
	Runner_State_Invariants(state, "runner_stop.state")
	Exit_Code_Invariants(status, "runner_stop.status")
	invariant.Always(state.Action == nil, "A setup runner stops with no queued work.")
	invariant.Always(state.Terminal == nil, "A setup runner stops one time.")
	state.Terminal = func() (terminal_status Exit_Code) { return status }
}

// File_Size_Limit is one permitted bound for a complete file payload.
type File_Size_Limit int

// File_Size_Limit_Invariants restricts reads to the dotfile or font payload bound.
func File_Size_Limit_Invariants(limit File_Size_Limit, namespace invariant.Namespace) {
	invariant.Tree(limit, namespace).
		Enum_Int(int(limit), DOTFILE_PAYLOAD_BYTES_MAX, COPY_BYTES_MAX).
		Ensure()
}

// File_Operation_Subject identifies the payload class named by completion diagnostics.
type File_Operation_Subject uint8

// File_Operation_Subject_Invariants restricts diagnostics to the two bounded file classes.
func File_Operation_Subject_Invariants(
	subject File_Operation_Subject, namespace invariant.Namespace,
) {
	invariant.Tree(subject, namespace).
		Enum_Uint8(uint8(subject), uint8(FILE_OPERATION_SUBJECT_DOTFILE),
			uint8(FILE_OPERATION_SUBJECT_FONT)).
		Ensure()
}

// FILE_OPERATION_SUBJECT_DOTFILE identifies mirrored configuration data.
const FILE_OPERATION_SUBJECT_DOTFILE File_Operation_Subject = 0

// FILE_OPERATION_SUBJECT_FONT identifies vendored font installation data.
const FILE_OPERATION_SUBJECT_FONT File_Operation_Subject = 1

// File_Chunks keeps large setup files in shared-algorithm-sized pieces.
type File_Chunks struct {
	// List avoids one allocation whose size comes from an untrusted file status.
	List *list.List
}

// File_Chunks_Invariants bounds every piece and their complete payload.
func File_Chunks_Invariants(chunks File_Chunks, _ invariant.Namespace) {
	invariant.Always(chunks.List != nil, "A setup file has a chunk list.")
	file_chunks_validate(chunks.List)
}

// File_Chunk_Remainder contains write chunks that IO has not accepted.
type File_Chunk_Remainder struct {
	// List retains the unwritten suffix without a phase-specific byte cursor.
	List *list.List
}

// File_Chunk_Remainder_Invariants bounds the unaccepted write suffix.
func File_Chunk_Remainder_Invariants(
	chunks File_Chunk_Remainder, _ invariant.Namespace,
) {
	invariant.Always(chunks.List != nil, "A file write has a remainder list.")
	file_chunks_validate(chunks.List)
}

// File_Chunk_History contains write chunks that IO accepted.
type File_Chunk_History struct {
	// List derives the next file offset without a phase-specific byte cursor.
	List *list.List
}

// File_Chunk_History_Invariants bounds the accepted write prefix.
func File_Chunk_History_Invariants(
	chunks File_Chunk_History, _ invariant.Namespace,
) {
	invariant.Always(chunks.List != nil, "A file write has a history list.")
	file_chunks_validate(chunks.List)
}

// File_chunks_validate keeps traversal out of the invariant bundle control flow.
func file_chunks_validate(elements *list.List) {
	invariant.Always(elements != nil, "A chunk validator has a list.")
	size := 0
	for element := elements.Front(); element != nil; element = element.Next() {
		content, valid := element.Value.([]byte)
		invariant.Always(valid, "A setup file chunk contains bytes.")
		invariant.Always(len(content) != 0, "A retained setup file chunk is not empty.")
		invariant.Always(
			len(content) <= shared_bytes.SLICE_SIZE_MAXIMUM,
			"A setup file chunk fits the shared slice limit.",
		)
		invariant.Always(
			size <= COPY_BYTES_MAX-len(content),
			"Setup file chunks cannot overflow their complete size.",
		)
		size += len(content)
	}
}

// File_chunks_size returns the validated size through a consumer to keep it call-local.
func file_chunks_size(chunks File_Chunks, consume func(size int)) {
	File_Chunks_Invariants(chunks, "file_chunks_size.chunks")
	file_chunk_list_size(chunks.List, consume)
}

// File_chunk_list_size measures validated storage owned by a larger operation bundle.
func file_chunk_list_size(elements *list.List, consume func(size int)) {
	file_chunks_validate(elements)
	invariant.Always(consume != nil, "A setup file size consumer is present.")
	size := 0
	for element := elements.Front(); element != nil; element = element.Next() {
		size += len(element.Value.([]byte))
	}
	consume(size)
}

// File_chunks_dotfile materializes only the smaller configuration-file domain.
func file_chunks_dotfile(chunks File_Chunks) (contents Dotfile_Bytes) {
	defer func() {
		Dotfile_Bytes_Invariants(contents, "file_chunks_dotfile.contents")
	}()
	File_Chunks_Invariants(chunks, "file_chunks_dotfile.chunks")
	size := 0
	file_chunks_size(chunks, func(measured int) { size = measured })
	invariant.Always(
		size <= DOTFILE_PAYLOAD_BYTES_MAX,
		"Configuration file chunks fit the dotfile limit.",
	)
	contents = make(Dotfile_Bytes, size)
	offset := 0
	for element := chunks.List.Front(); element != nil; element = element.Next() {
		offset += copy(contents[offset:], element.Value.([]byte))
	}
	return contents
}

// Dotfile_chunks exposes a planned write through the same bounded write path as a font.
func dotfile_chunks(contents Dotfile_Bytes) (chunks File_Chunks) {
	defer func() { File_Chunks_Invariants(chunks, "dotfile_chunks.chunks") }()
	Dotfile_Bytes_Invariants(contents, "dotfile_chunks.contents")
	chunks.List = list.New()
	for offset := 0; offset < len(contents); {
		end := offset + shared_bytes.SLICE_SIZE_MAXIMUM
		if end > len(contents) {
			end = len(contents)
		}
		chunks.List.PushBack([]byte(contents[offset:end]))
		offset = end
	}
	return chunks
}

// File_Read_Operation retains one buffer and descriptor until its completion retires.
type File_Read_Operation struct {
	// Runner records the next read or the consumer continuation.
	Runner *Runner_State
	// IO is the original shared operation surface.
	IO sysio.IO
	// File is the descriptor that remains open through the complete bounded read.
	File sysio.File
	// Limit selects the dotfile or font payload boundary.
	Limit File_Size_Limit
	// Subject names the payload class in completion diagnostics.
	Subject File_Operation_Subject
	// Contents retains all bytes from completed read submissions.
	Contents *File_Chunks
	// Continue gives the root the next read submission without a static call cycle.
	Continue func()
	// Duplicate retains a callback-contract failure until the operation's one close retires.
	Duplicate func() (err error)
	// Finish receives the complete payload after the descriptor closes.
	Finish func(contents File_Chunks, found File_Presence, err error)
}

// File_Read_Operation_Invariants states the operation payload bound and required continuations.
func File_Read_Operation_Invariants(
	operation *File_Read_Operation, namespace invariant.Namespace,
) {
	File_Size_Limit_Invariants(operation.Limit, namespace)
	File_Operation_Subject_Invariants(operation.Subject, namespace)
	File_Chunks_Invariants(*operation.Contents, namespace)
	Runner_State_Invariants(operation.Runner, namespace)
	contents_size := 0
	file_chunks_size(*operation.Contents, func(measured int) { contents_size = measured })
	invariant.Always(
		contents_size <= int(operation.Limit),
		"A file read stays inside its selected file-class limit.",
	)
	invariant.Always(operation.Continue != nil, "A file read has a continuation.")
	invariant.Always(operation.Finish != nil, "A file read has a consumer.")
}

// File_Write_Operation retains caller-owned bytes until every write completion retires.
type File_Write_Operation struct {
	// Runner records the next partial write or the close continuation.
	Runner *Runner_State
	// IO is the original shared operation surface.
	IO sysio.IO
	// File is the destination descriptor.
	File sysio.File
	// Subject names the payload class in completion diagnostics.
	Subject File_Operation_Subject
	// Contents remains unchanged while IO owns each submitted suffix.
	Remainder *File_Chunk_Remainder
	// Written preserves the file offset without a phase-specific integer field.
	Written *File_Chunk_History
	// Continue gives the root the next write submission without a static call cycle.
	Continue func()
	// Duplicate retains a callback-contract failure until the operation's one close retires.
	Duplicate func() (err error)
	// Finish receives the close or write error.
	Finish func(err error)
}

// File_Write_Operation_Invariants states the bounded payload and required continuations.
func File_Write_Operation_Invariants(
	operation *File_Write_Operation, namespace invariant.Namespace,
) {
	File_Operation_Subject_Invariants(operation.Subject, namespace)
	File_Chunk_Remainder_Invariants(*operation.Remainder, namespace)
	File_Chunk_History_Invariants(*operation.Written, namespace)
	Runner_State_Invariants(operation.Runner, namespace)
	invariant.Always(operation.Continue != nil, "A file write has a continuation.")
	invariant.Always(operation.Finish != nil, "A file write has a consumer.")
	written_size := 0
	file_chunk_list_size(
		operation.Written.List, func(measured int) { written_size = measured })
	remainder_size := 0
	file_chunk_list_size(
		operation.Remainder.List, func(measured int) { remainder_size = measured })
	invariant.Always(
		written_size <= COPY_BYTES_MAX-remainder_size,
		"A file write stays inside the largest accepted file class.",
	)
}

// Process_spawn_start submits one process and records its consumer for root rearm.
func process_spawn_start(
	state *Runner_State, system sysio.IO, request sysio.Process_Request,
	continuation func(result sysio.Process_Result, err error),
) {
	Runner_State_Invariants(state, "process_spawn_start.state")
	process_io_requirements(system, "process_spawn_start.system")
	retired := false
	completion := &sysio.Completion{}
	operation := runner_operation_start(state)
	system.Spawn(completion, func(
		_ *sysio.Completion, result sysio.Process_Result, operation_err error,
	) {
		if retired {
			runner_replace_queued(state, func() {
				continuation(
					sysio.Process_Result{Exit: 1},
					errors.New("the process spawn retired more than once"),
				)
			})
			return
		}
		retired = true
		runner_complete_io(state, operation, func() {
			continuation(result, operation_err)
		})
	}, request, PROCESS_DURATION_MAX)
}

// File_close_start submits one close and records its consumer for root rearm.
func file_close_start(
	state *Runner_State, system sysio.IO, file sysio.File,
	continuation func(err error),
) {
	Runner_State_Invariants(state, "file_close_start.state")
	retired := false
	completion := &sysio.Completion{}
	operation := runner_operation_start(state)
	system.Close(completion, func(_ *sysio.Completion, operation_err error) {
		if retired {
			runner_replace_queued(state, func() {
				continuation(errors.New("the file close retired more than once"))
			})
			return
		}
		retired = true
		runner_complete_io(state, operation, func() {
			continuation(operation_err)
		})
	}, file)
}

// File_read_start validates and opens one file before it queues the first bounded read.
func file_read_start(
	state *Runner_State, system sysio.IO, path File_Path, limit File_Size_Limit,
	subject File_Operation_Subject,
	continuation func(contents File_Chunks, found File_Presence, err error),
) {
	Runner_State_Invariants(state, "file_read_start.state")
	File_Path_Invariants(path, "file_read_start.path")
	File_Size_Limit_Invariants(limit, "file_read_start.limit")
	File_Operation_Subject_Invariants(subject, "file_read_start.subject")
	file_io_requirements(system, "file_read_start.system")
	status, status_err := system.Status(string(path))
	if status_err != nil {
		continuation(File_Chunks{List: list.New()}, false, status_err)
		return
	}
	if !status.Exists {
		continuation(File_Chunks{List: list.New()}, false, nil)
		return
	}
	if !status.Is_Regular {
		continuation(
			File_Chunks{List: list.New()}, true,
			errors.New("the file is not regular"),
		)
		return
	}
	if status.Size < 0 {
		continuation(
			File_Chunks{List: list.New()}, true,
			errors.New("the file size is negative"),
		)
		return
	}
	if status.Size > int64(limit) {
		continuation(
			File_Chunks{List: list.New()}, true,
			errors.New("the file exceeds its size limit"),
		)
		return
	}
	file, open_err := system.Open(string(path))
	if open_err != nil {
		continuation(File_Chunks{List: list.New()}, true, open_err)
		return
	}
	operation := &File_Read_Operation{
		Runner: state, IO: system, File: file, Limit: limit,
		Subject: subject, Contents: &File_Chunks{List: list.New()}, Finish: continuation,
	}
	operation.Continue = func() { file_read_rearm(operation) }
	runner_queue(state, operation.Continue)
}

// File_read_rearm submits one bounded suffix while the descriptor remains open.
func file_read_rearm(operation *File_Read_Operation) {
	File_Read_Operation_Invariants(operation, "file_read_rearm.operation")
	if operation.Duplicate != nil {
		file_read_finish(operation, nil)
		return
	}
	contents_size := 0
	file_chunks_size(*operation.Contents, func(measured int) { contents_size = measured })
	capacity := int(operation.Limit) - contents_size + 1
	buffer_size := shared_bytes.SLICE_SIZE_MAXIMUM
	if capacity < buffer_size {
		buffer_size = capacity
	}
	buffer := make([]byte, buffer_size)
	retired := false
	completion := &sysio.Completion{}
	submission := runner_operation_start(operation.Runner)
	operation.IO.Read(
		completion,
		func(_ *sysio.Completion, count int, operation_err error) {
			if retired {
				operation.Duplicate = func() (err error) {
					return file_read_duplicate_error(operation.Subject)
				}
				return
			}
			retired = true
			runner_complete_io(operation.Runner, submission, func() {
				if operation.Duplicate != nil {
					file_read_finish(operation, nil)
					return
				}
				if operation_err != nil {
					file_read_finish(operation, operation_err)
					return
				}
				if count < 0 {
					count_err := file_read_negative_error(operation.Subject)
					file_read_finish(operation, count_err)
					return
				}
				if count > len(buffer) {
					file_read_finish(
						operation, file_read_count_error(operation.Subject))
					return
				}
				if count == 0 {
					file_read_finish(operation, nil)
					return
				}
				operation.Contents.List.PushBack(buffer[:count])
				completed_size := 0
				file_chunks_size(*operation.Contents, func(measured int) {
					completed_size = measured
				})
				if completed_size > int(operation.Limit) {
					limit_err := errors.New("the file exceeds its size limit")
					file_read_finish(operation, limit_err)
					return
				}
				runner_queue(operation.Runner, operation.Continue)
			})
		},
		operation.File, buffer, int64(contents_size),
	)
}

// File_read_finish closes the descriptor before it gives the payload to its consumer.
func file_read_finish(operation *File_Read_Operation, operation_err error) {
	File_Read_Operation_Invariants(operation, "file_read_finish.operation")
	file_close_start(operation.Runner, operation.IO, operation.File, func(close_err error) {
		duplicate_err := error(nil)
		if operation.Duplicate != nil {
			duplicate_err = operation.Duplicate()
		}
		operation.Finish(
			*operation.Contents, true,
			errors.Join(operation_err, duplicate_err, close_err),
		)
	})
}

// File_write_start creates one destination before it queues the first partial write.
func file_write_start(
	state *Runner_State, system sysio.IO, path Destination_Path, contents File_Chunks,
	subject File_Operation_Subject,
	continuation func(err error),
) {
	Runner_State_Invariants(state, "file_write_start.state")
	Destination_Path_Invariants(path, "file_write_start.path")
	File_Chunks_Invariants(contents, "file_write_start.contents")
	File_Operation_Subject_Invariants(subject, "file_write_start.subject")
	file_io_requirements(system, "file_write_start.system")
	if mkdir_err := system.Make_Directory(filepath.Dir(string(path))); mkdir_err != nil {
		continuation(mkdir_err)
		return
	}
	file, create_err := system.Create(string(path))
	if create_err != nil {
		continuation(create_err)
		return
	}
	remainder := &File_Chunk_Remainder{List: list.New()}
	for element := contents.List.Front(); element != nil; element = element.Next() {
		remainder.List.PushBack(element.Value)
	}
	operation := &File_Write_Operation{
		Runner: state, IO: system, File: file, Subject: subject,
		Remainder: remainder,
		Written:   &File_Chunk_History{List: list.New()},
		Finish:    continuation,
	}
	operation.Continue = func() { file_write_rearm(operation) }
	if contents.List.Len() == 0 {
		file_close_start(state, system, file, continuation)
		return
	}
	runner_queue(state, operation.Continue)
}

// File_write_rearm submits the unwritten suffix and retains it until callback retirement.
func file_write_rearm(operation *File_Write_Operation) {
	File_Write_Operation_Invariants(operation, "file_write_rearm.operation")
	if operation.Duplicate != nil {
		file_write_finish(operation, nil)
		return
	}
	current := operation.Remainder.List.Front()
	invariant.Always(current != nil, "A queued file write has a pending chunk.")
	buffer := current.Value.([]byte)
	offset := 0
	file_chunk_list_size(
		operation.Written.List, func(measured int) { offset = measured })
	retired := false
	completion := &sysio.Completion{}
	submission := runner_operation_start(operation.Runner)
	operation.IO.Write(
		completion,
		func(_ *sysio.Completion, count int, operation_err error) {
			if retired {
				operation.Duplicate = func() (err error) {
					return file_write_duplicate_error(operation.Subject)
				}
				return
			}
			retired = true
			runner_complete_io(operation.Runner, submission, func() {
				if operation.Duplicate != nil {
					file_write_finish(operation, nil)
					return
				}
				if operation_err != nil {
					file_write_finish(operation, operation_err)
					return
				}
				if count <= 0 {
					progress_err := file_write_progress_error(operation.Subject)
					file_write_finish(operation, progress_err)
					return
				}
				if count > len(buffer) {
					count_err := file_write_count_error(operation.Subject)
					file_write_finish(operation, count_err)
					return
				}
				operation.Written.List.PushBack(buffer[:count])
				if count == len(buffer) {
					operation.Remainder.List.Remove(current)
				} else {
					current.Value = buffer[count:]
				}
				if operation.Remainder.List.Len() == 0 {
					file_write_finish(operation, nil)
					return
				}
				runner_queue(operation.Runner, operation.Continue)
			})
		},
		operation.File, buffer, int64(offset),
	)
}

// File_write_finish closes the destination before it reports the combined result.
func file_write_finish(operation *File_Write_Operation, operation_err error) {
	File_Write_Operation_Invariants(operation, "file_write_finish.operation")
	file_close_start(operation.Runner, operation.IO, operation.File, func(close_err error) {
		duplicate_err := error(nil)
		if operation.Duplicate != nil {
			duplicate_err = operation.Duplicate()
		}
		operation.Finish(errors.Join(operation_err, duplicate_err, close_err))
	})
}

// Copy_file_start reads one bounded font before it creates the destination.
func copy_file_start(
	state *Runner_State, system sysio.IO, input *File_Copy_Input,
	continuation func(err error),
) {
	Runner_State_Invariants(state, "copy_file_start.state")
	File_Copy_Input_Invariants(input, "copy_file_start.input")
	file_read_start(
		state, system, File_Path(input.Source), COPY_BYTES_MAX,
		FILE_OPERATION_SUBJECT_FONT,
		func(contents File_Chunks, found File_Presence, read_err error) {
			if read_err != nil {
				continuation(read_err)
				return
			}
			if !found {
				continuation(errors.New("copy source is absent"))
				return
			}
			file_write_start(
				state, system, Destination_Path(input.Destination),
				contents,
				FILE_OPERATION_SUBJECT_FONT, continuation)
		},
	)
}

// Runner_replace_queued preserves a duplicate failure across separate loop passes.
func runner_replace_queued(state *Runner_State, action func()) {
	Runner_State_Invariants(state, "runner_replace_queued.state")
	invariant.Always(action != nil, "A duplicate IO callback has a nonnil continuation.")
	if state.Terminal != nil {
		return
	}
	state.Action = action
}

// File_read_duplicate_error keeps the two payload-class diagnostics explicit.
func file_read_duplicate_error(subject File_Operation_Subject) (err error) {
	File_Operation_Subject_Invariants(subject, "file_read_duplicate_error.subject")
	if subject == FILE_OPERATION_SUBJECT_FONT {
		return errors.New("the font read retired more than once")
	}
	return errors.New("the file read retired more than once")
}

// File_read_negative_error keeps the two payload-class diagnostics explicit.
func file_read_negative_error(subject File_Operation_Subject) (err error) {
	File_Operation_Subject_Invariants(subject, "file_read_negative_error.subject")
	if subject == FILE_OPERATION_SUBJECT_FONT {
		return errors.New("the font read returned a negative byte count")
	}
	return errors.New("the file read returned a negative byte count")
}

// File_read_count_error keeps the two payload-class diagnostics explicit.
func file_read_count_error(subject File_Operation_Subject) (err error) {
	File_Operation_Subject_Invariants(subject, "file_read_count_error.subject")
	if subject == FILE_OPERATION_SUBJECT_FONT {
		return errors.New("the font read returned an invalid byte count")
	}
	return errors.New("the file read returned an invalid byte count")
}

// File_write_duplicate_error keeps the two payload-class diagnostics explicit.
func file_write_duplicate_error(subject File_Operation_Subject) (err error) {
	File_Operation_Subject_Invariants(subject, "file_write_duplicate_error.subject")
	if subject == FILE_OPERATION_SUBJECT_FONT {
		return errors.New("the font write retired more than once")
	}
	return errors.New("the file write retired more than once")
}

// File_write_progress_error keeps the two payload-class diagnostics explicit.
func file_write_progress_error(subject File_Operation_Subject) (err error) {
	File_Operation_Subject_Invariants(subject, "file_write_progress_error.subject")
	if subject == FILE_OPERATION_SUBJECT_FONT {
		return errors.New("the font write made no progress")
	}
	return errors.New("the file write made no progress")
}

// File_write_count_error keeps the two payload-class diagnostics explicit.
func file_write_count_error(subject File_Operation_Subject) (err error) {
	File_Operation_Subject_Invariants(subject, "file_write_count_error.subject")
	if subject == FILE_OPERATION_SUBJECT_FONT {
		return errors.New("the font write returned an invalid byte count")
	}
	return errors.New("the file write returned an invalid byte count")
}

// PROCESS_INVOCATION_COUNT_MINIMUM is the two-phase Neovim sequence.
const PROCESS_INVOCATION_COUNT_MINIMUM = 2

// PROCESS_SEQUENCE_INDEX_MINIMUM is the first invocation position.
const PROCESS_SEQUENCE_INDEX_MINIMUM = 0

// PROCESS_SEQUENCE_INDEX_MAXIMUM is the last macOS command position.
const PROCESS_SEQUENCE_INDEX_MAXIMUM = MACOS_COMMAND_COUNT - 1

// INSTALL_ALREADY_MESSAGE_BYTES_MINIMUM is the shortest converged-install diagnostic.
const INSTALL_ALREADY_MESSAGE_BYTES_MINIMUM = 16

// INSTALL_ALREADY_MESSAGE_BYTES_MAXIMUM is the longest converged-install diagnostic.
const INSTALL_ALREADY_MESSAGE_BYTES_MAXIMUM = 24

// INSTALL_BUILD_MESSAGE_BYTES_MINIMUM is the shortest required-build diagnostic.
const INSTALL_BUILD_MESSAGE_BYTES_MINIMUM = 11

// INSTALL_BUILD_MESSAGE_BYTES_MAXIMUM is the longest required-build diagnostic.
const INSTALL_BUILD_MESSAGE_BYTES_MAXIMUM = 16

// INSTALL_FAILURE_MESSAGE_BYTES_MINIMUM is the shortest failed-build diagnostic.
const INSTALL_FAILURE_MESSAGE_BYTES_MINIMUM = 15

// INSTALL_FAILURE_MESSAGE_BYTES_MAXIMUM is the longest failed-build diagnostic.
const INSTALL_FAILURE_MESSAGE_BYTES_MAXIMUM = 20

// Install_Already_Message is one fixed converged-install diagnostic.
type Install_Already_Message string

// Install_Already_Message_Invariants rejects an absent converged-install diagnostic.
func Install_Already_Message_Invariants(
	message Install_Already_Message, namespace invariant.Namespace,
) {
	invariant.Tree(message, namespace).
		Range_Int(
			len(message), INSTALL_ALREADY_MESSAGE_BYTES_MINIMUM,
			INSTALL_ALREADY_MESSAGE_BYTES_MAXIMUM).
		Ensure()
}

// Install_Build_Message is one fixed required-build diagnostic.
type Install_Build_Message string

// Install_Build_Message_Invariants rejects an absent required-build diagnostic.
func Install_Build_Message_Invariants(
	message Install_Build_Message, namespace invariant.Namespace,
) {
	invariant.Tree(message, namespace).
		Range_Int(
			len(message), INSTALL_BUILD_MESSAGE_BYTES_MINIMUM,
			INSTALL_BUILD_MESSAGE_BYTES_MAXIMUM).
		Ensure()
}

// Install_Failure_Message is one fixed failed-build diagnostic.
type Install_Failure_Message string

// Install_Failure_Message_Invariants rejects an absent failed-build diagnostic.
func Install_Failure_Message_Invariants(
	message Install_Failure_Message, namespace invariant.Namespace,
) {
	invariant.Tree(message, namespace).
		Range_Int(
			len(message), INSTALL_FAILURE_MESSAGE_BYTES_MINIMUM,
			INSTALL_FAILURE_MESSAGE_BYTES_MAXIMUM).
		Ensure()
}

// Process_Sequence_Index selects one invocation from a bounded process list.
type Process_Sequence_Index int

// Process_Sequence_Index_Invariants bounds the largest retained process sequence.
func Process_Sequence_Index_Invariants(
	index Process_Sequence_Index, namespace invariant.Namespace,
) {
	invariant.Tree(index, namespace).
		Range_Int(
			int(index), PROCESS_SEQUENCE_INDEX_MINIMUM,
			PROCESS_SEQUENCE_INDEX_MAXIMUM).
		Ensure()
}

// Run_pipe_start captures one successful process output and records empty output on failure.
func run_pipe_start(
	state *Runner_State, system sysio.IO, path Probe_Path, arguments Probe_Arguments,
	continuation func(output Command_Output),
) {
	Runner_State_Invariants(state, "run_pipe_start.state")
	Probe_Path_Invariants(path, "run_pipe_start.path")
	Probe_Arguments_Invariants(arguments, "run_pipe_start.arguments")
	process_spawn_start(state, system, sysio.Process_Request{
		Path: string(path), Arguments: []string(arguments),
	}, func(result sysio.Process_Result, spawn_err error) {
		if spawn_err != nil {
			continuation("")
			return
		}
		if result.Exit != 0 {
			continuation("")
			return
		}
		continuation(Command_Output(
			shared_strings.Trim_Space(shared_strings.Text(result.Output))))
	})
}

// Run_spawn_start streams one process through shared streams and reports its exit result.
func run_spawn_start(
	state *Runner_State, system sysio.IO, stdout sysio.Stream, stderr sysio.Stream,
	arguments Spawn_Arguments, continuation func(succeeded Step_Success),
) {
	Runner_State_Invariants(state, "run_spawn_start.state")
	Spawn_Arguments_Invariants(arguments, "run_spawn_start.arguments")
	request := sysio.Process_Request{
		Path: arguments[0], Arguments: arguments[1:],
		Stdout: stdout, Stderr: stderr,
	}
	process_spawn_start(state, system, request, func(
		result sysio.Process_Result, spawn_err error,
	) {
		if spawn_err != nil {
			continuation(false)
			return
		}
		continuation(result.Exit == 0)
	})
}

// Installed_start probes one managed executable and reports whether its version matches.
func installed_start(
	state *Runner_State, input *Installed_Input,
	continuation func(installed File_Presence),
) {
	Runner_State_Invariants(state, "installed_start.state")
	Installed_Input_Invariants(input, "installed_start.input")
	run_pipe_start(
		state, input.IO, Probe_Path(input.Executable), Probe_Arguments{"--version"},
		func(version Command_Output) {
			continuation(File_Presence(shared_strings.Has_Prefix(
				shared_strings.Text(version), shared_strings.Text(input.Version))))
		},
	)
}

// Command_on_path_start resolves one command without introducing another process seam.
func command_on_path_start(
	state *Runner_State, system sysio.IO, name Command_Name,
	continuation func(present File_Presence),
) {
	Runner_State_Invariants(state, "command_on_path_start.state")
	Command_Name_Invariants(name, "command_on_path_start.name")
	run_pipe_start(state, system, "which", Probe_Arguments{string(name)}, func(
		output Command_Output,
	) {
		continuation(output != "")
	})
}

// Versioned_install_start runs one version gate and one build when the gate fails.
func versioned_install_start(
	state *Runner_State, input *Versioned_Install_Input,
	continuation func(succeeded Step_Success),
) {
	Runner_State_Invariants(state, "versioned_install_start.state")
	Versioned_Install_Input_Invariants(input, "versioned_install_start.input")
	installed_start(state, &Installed_Input{
		IO: input.IO, Executable: Managed_Executable(input.Executable),
		Version: input.Version,
	}, func(installed File_Presence) {
		if installed {
			jlog.Logger_Info(input.Logger, string(input.Already_Message))
			continuation(true)
			return
		}
		jlog.Logger_Info(input.Logger, string(input.Build_Message))
		run_spawn_start(
			state, input.IO, input.Stdout, input.Stderr,
			Spawn_Arguments(input.Invocation),
			func(succeeded Step_Success) {
				if !succeeded {
					jlog.Logger_Error(
						input.Logger, string(input.Failure_Message))
				}
				continuation(succeeded)
			},
		)
	})
}

// Versioned_Install_Input contains one common version gate and build operation.
type Versioned_Install_Input struct {
	// IO submits the probe and build process.
	IO sysio.IO
	// Executable is the exact managed binary that the gate probes.
	Executable Versioned_Executable
	// Version is the required output prefix.
	Version Version_Prefix
	// Invocation is the build or install process.
	Invocation Build_Invocation
	// Logger records the selected branch and a process failure.
	Logger jlog.Logger
	// Stdout receives live process output.
	Stdout sysio.Stream
	// Stderr receives live process diagnostics.
	Stderr sysio.Stream
	// Already_Message describes a matching managed executable.
	Already_Message Install_Already_Message
	// Build_Message describes the required build.
	Build_Message Install_Build_Message
	// Failure_Message describes a failed build.
	Failure_Message Install_Failure_Message
}

// Versioned_Install_Input_Invariants states the shared version gate facts.
func Versioned_Install_Input_Invariants(
	input *Versioned_Install_Input, namespace invariant.Namespace,
) {
	process_io_requirements(input.IO, namespace)
	Versioned_Executable_Invariants(input.Executable, namespace)
	Version_Prefix_Invariants(input.Version, namespace)
	Build_Invocation_Invariants(input.Invocation, namespace)
	Install_Already_Message_Invariants(input.Already_Message, namespace)
	Install_Build_Message_Invariants(input.Build_Message, namespace)
	Install_Failure_Message_Invariants(input.Failure_Message, namespace)
}

// Process_Invocations is one bounded sequence of general setup processes.
type Process_Invocations [][]string

// Process_Invocations_Invariants bounds a sequence from one build to all macOS commands.
func Process_Invocations_Invariants(
	invocations Process_Invocations, namespace invariant.Namespace,
) {
	invariant.Tree(invocations, namespace).
		Range_Int(
			len(invocations), PROCESS_INVOCATION_COUNT_MINIMUM,
			MACOS_COMMAND_COUNT).
		Ensure()
}

// Process_Sequence retains one fixed process list while callbacks advance its index.
type Process_Sequence struct {
	// Runner records each next process submission.
	Runner *Runner_State
	// IO is the original shared operation surface.
	IO sysio.IO
	// Invocations contains the fixed process order.
	Invocations Process_Invocations
	// Index selects the next invocation.
	Index Process_Sequence_Index
	// Stdout receives live process output.
	Stdout sysio.Stream
	// Stderr receives live process diagnostics.
	Stderr sysio.Stream
	// Before records process-specific progress before submission.
	Before func(index int)
	// Continue gives the root the next process without a static call cycle.
	Continue func()
	// Finish receives the first failure or the complete success result.
	Finish func(succeeded Step_Success)
}

// Process_Sequence_Invariants states its fixed process list and required continuations.
func Process_Sequence_Invariants(sequence *Process_Sequence, namespace invariant.Namespace) {
	Process_Invocations_Invariants(sequence.Invocations, namespace)
	Runner_State_Invariants(sequence.Runner, namespace)
	Process_Sequence_Index_Invariants(sequence.Index, namespace)
	invariant.Always(sequence.Continue != nil, "A process sequence has a continuation.")
	invariant.Always(sequence.Finish != nil, "A process sequence has a consumer.")
	invariant.Always(sequence.Index >= 0, "A process sequence index is not negative.")
	invariant.Always(
		int(sequence.Index) <= len(sequence.Invocations),
		"A process sequence index stays inside its invocation list.",
	)
}

// Process_sequence_start records a bounded list for sequential process submission.
func process_sequence_start(
	state *Runner_State, system sysio.IO, stdout sysio.Stream, stderr sysio.Stream,
	invocations Process_Invocations, before func(index int),
	continuation func(succeeded Step_Success),
) {
	Runner_State_Invariants(state, "process_sequence_start.state")
	Process_Invocations_Invariants(invocations, "process_sequence_start.invocations")
	if len(invocations) == 0 {
		continuation(true)
		return
	}
	sequence := &Process_Sequence{
		Runner: state, IO: system, Invocations: invocations,
		Stdout: stdout, Stderr: stderr, Before: before, Finish: continuation,
	}
	sequence.Continue = func() { process_sequence_rearm(sequence) }
	runner_queue(state, sequence.Continue)
}

// Process_sequence_rearm submits one invocation and records the next index after success.
func process_sequence_rearm(sequence *Process_Sequence) {
	Process_Sequence_Invariants(sequence, "process_sequence_rearm.sequence")
	if sequence.Before != nil {
		sequence.Before(int(sequence.Index))
	}
	invocation := sequence.Invocations[int(sequence.Index)]
	request := sysio.Process_Request{
		Path: invocation[0], Arguments: invocation[1:],
		Stdout: sequence.Stdout, Stderr: sequence.Stderr,
	}
	process_spawn_start(
		sequence.Runner, sequence.IO, request,
		func(result sysio.Process_Result, process_err error) {
			succeeded := Step_Success(false)
			if process_err == nil {
				succeeded = result.Exit == 0
			}
			if !succeeded {
				sequence.Finish(false)
				return
			}
			sequence.Index++
			if int(sequence.Index) == len(sequence.Invocations) {
				sequence.Finish(true)
				return
			}
			runner_queue(sequence.Runner, sequence.Continue)
		},
	)
}

// Install_direnv_start runs the vendored direnv gate and build.
func install_direnv_start(
	state *Runner_State, input *Install_Direnv_Input,
	continuation func(succeeded Step_Success),
) {
	Runner_State_Invariants(state, "install_direnv_start.state")
	Install_Direnv_Input_Invariants(input, "install_direnv_start.input")
	if input.Direnv_Directory == "" {
		continuation(true)
		return
	}
	if input.Binary_Directory == "" {
		continuation(true)
		return
	}
	destination := filepath.Join(string(input.Binary_Directory), "direnv")
	build := "cd " + string(input.Direnv_Directory) +
		" && CGO_ENABLED=0 go build -mod=vendor -o " + destination + " ."
	versioned_install_start(state, &Versioned_Install_Input{
		IO:         input.IO,
		Executable: Versioned_Executable(destination),
		Version:    DIRENV_VERSION,
		Invocation: Build_Invocation{"sh", "-c", build}, Logger: input.Logger,
		Stdout: input.Stdout, Stderr: input.Stderr,
		Already_Message: "direnv already installed", Build_Message: "building direnv",
		Failure_Message: "direnv build failed",
	}, continuation)
}

// Install_fzf_start runs the vendored fzf gate and build.
func install_fzf_start(
	state *Runner_State, input *Install_Fzf_Input,
	continuation func(succeeded Step_Success),
) {
	Runner_State_Invariants(state, "install_fzf_start.state")
	Install_Fzf_Input_Invariants(input, "install_fzf_start.input")
	if input.Fzf_Directory == "" {
		continuation(true)
		return
	}
	if input.Binary_Directory == "" {
		continuation(true)
		return
	}
	destination := filepath.Join(string(input.Binary_Directory), "fzf")
	build := "cd " + string(input.Fzf_Directory) +
		" && go build -mod=vendor" +
		" -ldflags '-s -w -X main.version=" + FZF_VERSION + " -X main.revision='" +
		" -o " + destination + " ."
	versioned_install_start(state, &Versioned_Install_Input{
		IO:         input.IO,
		Executable: Versioned_Executable(destination),
		Version:    FZF_VERSION,
		Invocation: Build_Invocation{"sh", "-c", build}, Logger: input.Logger,
		Stdout: input.Stdout, Stderr: input.Stderr,
		Already_Message: "fzf already installed", Build_Message: "building fzf",
		Failure_Message: "fzf build failed",
	}, continuation)
}

// Install_jj_start runs the vendored jj gate and build.
func install_jj_start(
	state *Runner_State, input *Install_Jj_Input,
	continuation func(succeeded Step_Success),
) {
	Runner_State_Invariants(state, "install_jj_start.state")
	Install_Jj_Input_Invariants(input, "install_jj_start.input")
	if input.Jj_Directory == "" {
		continuation(true)
		return
	}
	if input.Binary_Directory == "" {
		continuation(true)
		return
	}
	executable := Versioned_Executable(filepath.Join(
		string(input.Binary_Directory), "jj"))
	versioned_install_start(state, &Versioned_Install_Input{
		IO:         input.IO,
		Executable: executable,
		Version:    JJ_VERSION, Invocation: Build_Invocation(jj_install_invocation(input)),
		Logger: input.Logger, Stdout: input.Stdout, Stderr: input.Stderr,
		Already_Message: "jj already built", Build_Message: "building jj",
		Failure_Message: "jj build failed",
	}, continuation)
}

// Install_ripgrep_start runs the vendored ripgrep gate and build.
func install_ripgrep_start(
	state *Runner_State, input *Install_Ripgrep_Input,
	continuation func(succeeded Step_Success),
) {
	Runner_State_Invariants(state, "install_ripgrep_start.state")
	Install_Ripgrep_Input_Invariants(input, "install_ripgrep_start.input")
	if input.Ripgrep_Directory == "" {
		continuation(true)
		return
	}
	if input.Binary_Directory == "" {
		continuation(true)
		return
	}
	executable := Versioned_Executable(filepath.Join(
		string(input.Binary_Directory), "rg"))
	versioned_install_start(state, &Versioned_Install_Input{
		IO:         input.IO,
		Executable: executable,
		Version:    RIPGREP_VERSION,
		Invocation: Build_Invocation(ripgrep_install_invocation(input)),
		Logger:     input.Logger, Stdout: input.Stdout, Stderr: input.Stderr,
		Already_Message: "ripgrep already built", Build_Message: "building ripgrep",
		Failure_Message: "ripgrep build failed",
	}, continuation)
}

// Install_fdcli_start runs the vendored fd gate and build.
func install_fdcli_start(
	state *Runner_State, input *Install_Fdcli_Input,
	continuation func(succeeded Step_Success),
) {
	Runner_State_Invariants(state, "install_fdcli_start.state")
	Install_Fdcli_Input_Invariants(input, "install_fdcli_start.input")
	if input.Fdcli_Directory == "" {
		continuation(true)
		return
	}
	if input.Binary_Directory == "" {
		continuation(true)
		return
	}
	executable := Versioned_Executable(filepath.Join(
		string(input.Binary_Directory), "fd"))
	versioned_install_start(state, &Versioned_Install_Input{
		IO:         input.IO,
		Executable: executable,
		Version:    FDCLI_VERSION,
		Invocation: Build_Invocation(fdcli_install_invocation(input)),
		Logger:     input.Logger, Stdout: input.Stdout, Stderr: input.Stderr,
		Already_Message: "fd already built", Build_Message: "building fd",
		Failure_Message: "fd build failed",
	}, continuation)
}

// Install_command_start runs one PATH gate and builds an absent repository command.
func install_command_start(
	state *Runner_State, input *Install_Command_Input,
	continuation func(succeeded Step_Success),
) {
	Runner_State_Invariants(state, "install_command_start.state")
	Install_Command_Input_Invariants(input, "install_command_start.input")
	if input.Package_Directory == "" {
		continuation(true)
		return
	}
	if input.Binary_Directory == "" {
		continuation(true)
		return
	}
	if input.Binary_Name == "" {
		continuation(true)
		return
	}
	command_on_path_start(state, input.IO, input.Binary_Name, func(present File_Presence) {
		if present {
			jlog.Logger_Info(input.Logger, "command already installed",
				jlog.String("command", string(input.Binary_Name)))
			continuation(true)
			return
		}
		jlog.Logger_Info(input.Logger, "building command",
			jlog.String("command", string(input.Binary_Name)))
		destination := filepath.Join(
			string(input.Binary_Directory), string(input.Binary_Name))
		build := "cd " + string(input.Package_Directory) +
			" && go build -o " + destination + " ."
		run_spawn_start(
			state, input.IO, input.Stdout, input.Stderr,
			Spawn_Arguments{"sh", "-c", build},
			func(succeeded Step_Success) {
				if !succeeded {
					jlog.Logger_Error(input.Logger, "command build failed",
						jlog.String("command", string(input.Binary_Name)))
				}
				continuation(succeeded)
			},
		)
	})
}

// Install_neovim_start runs the repository path gate and both required make phases.
func install_neovim_start(
	state *Runner_State, input *Install_Neovim_Input,
	continuation func(succeeded Step_Success),
) {
	Runner_State_Invariants(state, "install_neovim_start.state")
	Install_Neovim_Input_Invariants(input, "install_neovim_start.input")
	run_pipe_start(state, input.IO, "which", Probe_Arguments{"nvim"}, func(
		executable Command_Output,
	) {
		if !shared_strings.Has_Prefix(
			shared_strings.Text(executable),
			shared_strings.Text(input.Repository_Directory)+"/",
		) {
			neovim_build_start(state, input, continuation)
			return
		}
		run_pipe_start(
			state, input.IO, Probe_Path(executable), Probe_Arguments{"--version"},
			func(version Command_Output) {
				if neovim_version_present(Version_Output(version)) {
					jlog.Logger_Info(input.Logger, "neovim already installed",
						jlog.String("version", NEOVIM_VERSION))
					continuation(true)
					return
				}
				neovim_build_start(state, input, continuation)
			},
		)
	})
}

// Neovim_build_start submits the configure and install phases as one process sequence.
func neovim_build_start(
	state *Runner_State, input *Install_Neovim_Input,
	continuation func(succeeded Step_Success),
) {
	Runner_State_Invariants(state, "neovim_build_start.state")
	Install_Neovim_Input_Invariants(input, "neovim_build_start.input")
	process_sequence_start(
		state, input.IO, input.Stdout, input.Stderr,
		Process_Invocations(neovim_make_invocations(input.Repository_Directory)),
		func(index int) {
			if index == 0 {
				jlog.Logger_Info(input.Logger, "configuring neovim prefix")
				return
			}
			jlog.Logger_Info(input.Logger, "installing neovim")
		},
		func(succeeded Step_Success) {
			if !succeeded {
				jlog.Logger_Error(input.Logger, "neovim build failed")
			}
			continuation(succeeded)
		},
	)
}

// Install_rust_start gates the toolchain install and refreshes its three links.
func install_rust_start(
	state *Runner_State, input *Install_Rust_Input,
	continuation func(succeeded Step_Success),
) {
	Runner_State_Invariants(state, "install_rust_start.state")
	Install_Rust_Input_Invariants(input, "install_rust_start.input")
	if input.Cargo_Directory == "" {
		continuation(true)
		return
	}
	if input.Link_Directory == "" {
		continuation(true)
		return
	}
	cargo := Required_Cargo_Directory(input.Cargo_Directory)
	links := &Rust_Link_Input{
		IO: input.IO, Stdout: input.Stdout, Stderr: input.Stderr,
		Cargo_Directory: cargo, Link_Directory: input.Link_Directory,
	}
	installed_start(state, &Installed_Input{
		IO:         input.IO,
		Executable: Managed_Executable(filepath.Join(string(cargo), "bin", "rustc")),
		Version:    "rustc " + RUST_VERSION,
	}, func(installed File_Presence) {
		if installed {
			jlog.Logger_Info(input.Logger, "rust already installed")
			rust_links_start(state, links, input.Logger, continuation)
			return
		}
		jlog.Logger_Info(input.Logger, "installing rust")
		run_spawn_start(
			state, input.IO, input.Stdout, input.Stderr,
			Spawn_Arguments(rust_install_invocation()),
			func(succeeded Step_Success) {
				if !succeeded {
					jlog.Logger_Error(input.Logger, "rust install failed")
					continuation(false)
					return
				}
				rust_links_start(state, links, input.Logger, continuation)
			},
		)
	})
}

// Rust_links_start submits the three link commands in their fixed order.
func rust_links_start(
	state *Runner_State, input *Rust_Link_Input, logger jlog.Logger,
	continuation func(succeeded Step_Success),
) {
	Runner_State_Invariants(state, "rust_links_start.state")
	Rust_Link_Input_Invariants(input, "rust_links_start.input")
	invocations := Process_Invocations{}
	for _, tool := range []string{"cargo", "rustup", "rustc"} {
		source := filepath.Join(string(input.Cargo_Directory), "bin", tool)
		target := filepath.Join(string(input.Link_Directory), tool)
		invocations = append(invocations, []string{"ln", "-sf", source, target})
	}
	process_sequence_start(
		state, input.IO, input.Stdout, input.Stderr, invocations, nil,
		func(succeeded Step_Success) {
			if !succeeded {
				jlog.Logger_Error(logger, "rust link failed")
			}
			continuation(succeeded)
		},
	)
}

// Install_ghostty_start gates the signed app install and refreshes its CLI link.
func install_ghostty_start(
	state *Runner_State, input *Install_Ghostty_Input,
	continuation func(succeeded Step_Success),
) {
	Runner_State_Invariants(state, "install_ghostty_start.state")
	Install_Ghostty_Input_Invariants(input, "install_ghostty_start.input")
	binary := Managed_Executable(filepath.Join(
		string(input.Applications_Directory), GHOSTTY_APPLICATION_BINARY_SUBPATH))
	installed_start(state, &Installed_Input{
		IO: input.IO, Executable: binary, Version: GHOSTTY_VERSION,
	}, func(version_present File_Presence) {
		if !version_present {
			ghostty_install_start(state, input, continuation)
			return
		}
		application := Application_Path(filepath.Join(
			string(input.Applications_Directory), "Ghostty.app"))
		run_spawn_start(
			state, input.IO, sysio.Stream{}, sysio.Stream{},
			Spawn_Arguments(ghostty_codesign_invocation(application)),
			func(signature_valid Step_Success) {
				if !signature_valid {
					ghostty_install_start(state, input, continuation)
					return
				}
				jlog.Logger_Info(input.Logger, "ghostty already installed")
				ghostty_link_start(state, input, continuation)
			},
		)
	})
}

// Ghostty_install_start runs the install script before it records the link continuation.
func ghostty_install_start(
	state *Runner_State, input *Install_Ghostty_Input,
	continuation func(succeeded Step_Success),
) {
	Runner_State_Invariants(state, "ghostty_install_start.state")
	Install_Ghostty_Input_Invariants(input, "ghostty_install_start.input")
	jlog.Logger_Info(input.Logger, "installing ghostty")
	run_spawn_start(
		state, input.IO, input.Stdout, input.Stderr,
		Spawn_Arguments(ghostty_install_invocation(input.Applications_Directory)),
		func(succeeded Step_Success) {
			if !succeeded {
				jlog.Logger_Error(input.Logger, "ghostty install failed")
				continuation(false)
				return
			}
			ghostty_link_start(state, input, continuation)
		},
	)
}

// Ghostty_link_start refreshes the one managed CLI link after either gate branch.
func ghostty_link_start(
	state *Runner_State, input *Install_Ghostty_Input,
	continuation func(succeeded Step_Success),
) {
	Runner_State_Invariants(state, "ghostty_link_start.state")
	Install_Ghostty_Input_Invariants(input, "ghostty_link_start.input")
	source := filepath.Join(
		string(input.Applications_Directory), GHOSTTY_APPLICATION_BINARY_SUBPATH)
	target := filepath.Join(string(input.Link_Directory), "ghostty")
	run_spawn_start(
		state, input.IO, input.Stdout, input.Stderr,
		Spawn_Arguments{"ln", "-sf", source, target},
		func(succeeded Step_Success) {
			if !succeeded {
				jlog.Logger_Error(input.Logger, "ghostty link failed")
			}
			continuation(succeeded)
		},
	)
}

// Plan_Operation retains the bounded traversal between shared IO completions.
type Plan_Operation struct {
	// Runner records the next directory or file transition.
	Runner *Runner_State
	// Input contains the original shared IO and mirror roots.
	Input *Plan_Input
	// Directories contains each discovered directory that the traversal has not read.
	Directories *list.List
	// Directory is the level whose entries the current process result classifies.
	Directory *list.Element
	// Entries contains the current bounded directory level.
	Entries *list.List
	// Paths contains the relative path for each current entry.
	Paths *list.List
	// Ignored records the current Git classification.
	Ignored func(path string) (ignored bool)
	// Entry selects the next current-level entry.
	Entry *list.Element
	// Path selects the relative path for Entry.
	Path *list.Element
	// Discovered bounds all directories, including those already processed.
	Discovered *list.List
	// Writes contains the bounded result in traversal order.
	Writes *list.List
	// Directory_Step breaks the static recursion that a traversal phase enum would conceal.
	Directory_Step func()
	// Entry_Step breaks the static recursion that a traversal phase enum would conceal.
	Entry_Step func()
	// Next selects the operation that Continue submits.
	Next func()
	// Continue gives the root the next transition without a static call cycle.
	Continue func()
	// Finish receives the complete plan or the first error.
	Finish func(writes Writes, err error)
}

// Plan_Operation_Invariants states the source facts and bounded result.
func Plan_Operation_Invariants(operation *Plan_Operation, namespace invariant.Namespace) {
	Plan_Input_Invariants(operation.Input, namespace)
	Runner_State_Invariants(operation.Runner, namespace)
	invariant.Always(operation.Directories != nil, "A plan has a directory queue.")
	invariant.Always(operation.Entries != nil, "A plan has a directory level.")
	invariant.Always(operation.Paths != nil, "A plan has relative paths.")
	invariant.Always(operation.Discovered != nil, "A plan counts discovered directories.")
	invariant.Always(operation.Writes != nil, "A plan has a write list.")
	invariant.Always(operation.Ignored != nil, "A plan can classify an ignored path.")
	invariant.Always(operation.Directory_Step != nil, "A plan can read its next directory.")
	invariant.Always(operation.Entry_Step != nil, "A plan can inspect its next entry.")
	invariant.Always(operation.Next != nil, "A plan has a selected operation.")
	invariant.Always(operation.Continue != nil, "A plan operation has a continuation.")
	invariant.Always(operation.Finish != nil, "A plan operation has a consumer.")
}

// Plan_start records the traversal root before the root submits asynchronous work.
func plan_start(
	state *Runner_State, input *Plan_Input,
	continuation func(writes Writes, err error),
) {
	Runner_State_Invariants(state, "plan_start.state")
	Plan_Input_Invariants(input, "plan_start.input")
	operation := &Plan_Operation{
		Runner: state, Input: input, Directories: list.New(), Entries: list.New(),
		Paths: list.New(), Discovered: list.New(), Writes: list.New(),
		Ignored: func(_ string) (ignored bool) { return false }, Finish: continuation,
	}
	operation.Directories.PushBack(Traversal_Directory("."))
	operation.Discovered.PushBack(struct{}{})
	operation.Directory_Step = func() { plan_directory_rearm(operation) }
	operation.Entry_Step = func() { plan_entry_rearm(operation) }
	operation.Next = operation.Directory_Step
	operation.Continue = func() { plan_rearm(operation) }
	runner_queue(state, operation.Continue)
}

// Plan_rearm submits the operation selected by the retained traversal phase.
func plan_rearm(operation *Plan_Operation) {
	Plan_Operation_Invariants(operation, "plan_rearm.operation")
	operation.Next()
}

// Plan_directory_rearm reads one level and submits one batched ignore probe.
func plan_directory_rearm(operation *Plan_Operation) {
	Plan_Operation_Invariants(operation, "plan_directory_rearm.operation")
	operation.Directory = operation.Directories.Front()
	invariant.Always(operation.Directory != nil, "A queued plan directory is present.")
	operation.Directories.Remove(operation.Directory)
	directory := operation.Directory.Value.(Traversal_Directory)
	Traversal_Directory_Invariants(directory, "plan_directory_rearm.directory")
	plan_narrate(operation.Input.Logger, directory)
	entries, read_err := operation.Input.IO.Read_Directory(filepath.Join(
		string(operation.Input.Source_Directory), string(directory)))
	if read_err != nil {
		operation.Finish(Writes{}, read_err)
		return
	}
	if len(entries) > PLAN_ENTRY_COUNT_MAX {
		entry_err := errors.New("mirror traversal entries exceed their limit")
		operation.Finish(Writes{}, entry_err)
		return
	}
	operation.Entries.Init()
	operation.Paths.Init()
	for _, child := range entries {
		relative := filepath.Join(string(directory), child.Name)
		if len(relative) > RELATIVE_FILE_PATH_BYTES_MAX {
			operation.Finish(
				Writes{}, errors.New("mirror relative path exceeds its limit"))
			return
		}
		operation.Entries.PushBack(child)
		operation.Paths.PushBack(relative)
	}
	if len(entries) == 0 {
		operation.Ignored = func(_ string) (ignored bool) { return false }
		operation.Entry = nil
		operation.Path = nil
		operation.Next = operation.Entry_Step
		runner_queue(operation.Runner, operation.Continue)
		return
	}
	request := plan_ignore_request(operation)
	process_spawn_start(operation.Runner, operation.Input.IO, request, func(
		result sysio.Process_Result, spawn_err error,
	) {
		if spawn_err != nil {
			operation.Finish(Writes{}, spawn_err)
			return
		}
		if result.Exit != 0 {
			operation.Finish(Writes{}, errors.New(
				"the gitignore probe exited with a nonzero status"))
			return
		}
		response := plan_ignore_result(operation, result)
		operation.Ignored = response.Contains
		operation.Entry = operation.Entries.Front()
		operation.Path = operation.Paths.Front()
		operation.Next = operation.Entry_Step
		runner_queue(operation.Runner, operation.Continue)
	})
}

// Plan_ignore_targets keeps path construction identical for the request and its validator.
func plan_ignore_targets(
	operation *Plan_Operation, yield func(target string) (next bool),
) {
	Plan_Operation_Invariants(operation, "plan_ignore_targets.operation")
	invariant.Always(yield != nil, "A plan ignore target callback is present.")
	for element := operation.Paths.Front(); element != nil; element = element.Next() {
		relative := element.Value.(string)
		target := filepath.Join(string(operation.Input.Source_Directory), relative)
		if !yield(target) {
			return
		}
	}
}

// Plan_ignore_result validates that each ignored path came from the submitted request.
func plan_ignore_result(
	operation *Plan_Operation, result sysio.Process_Result,
) (response *Check_Ignore_Result) {
	defer func() {
		Check_Ignore_Result_Invariants(response, "plan_ignore_result.response")
	}()
	Plan_Operation_Invariants(operation, "plan_ignore_result.operation")
	response = &Check_Ignore_Result{}
	response.Process = result
	response.Target = func(path string) (present bool) {
		plan_ignore_targets(operation, func(target string) (next bool) {
			present = target == path
			return !present
		})
		return present
	}
	check_ignore_matches(response)
	return response
}

// Plan_ignore_request builds one process request and its accepted output set.
func plan_ignore_request(operation *Plan_Operation) (request sysio.Process_Request) {
	Plan_Operation_Invariants(operation, "plan_ignore_request.operation")
	check := Check_Ignore_Request{
		Targets: func(yield func(target string) (next bool)) {
			plan_ignore_targets(operation, yield)
		},
	}
	check_ignore_input(&check)
	return sysio.Process_Request{
		Path: "git",
		Arguments: []string{
			"-C", string(operation.Input.Source_Directory), "check-ignore", "--stdin",
		},
		Input: check.Process.Input,
	}
}

// Plan_entry_rearm advances entries until one regular file needs asynchronous reads.
func plan_entry_rearm(operation *Plan_Operation) {
	Plan_Operation_Invariants(operation, "plan_entry_rearm.operation")
	advance := func() {
		operation.Entry = operation.Entry.Next()
		operation.Path = operation.Path.Next()
	}
	for operation.Entry != nil {
		child := operation.Entry.Value.(sysio.Directory_Entry)
		relative := Relative_File_Path(operation.Path.Value.(string))
		target := filepath.Join(
			string(operation.Input.Source_Directory), string(relative))
		if operation.Ignored(target) {
			advance()
			continue
		}
		if child.Is_Directory {
			if operation.Discovered.Len() == TRAVERSAL_DIRECTORY_COUNT_MAX {
				directory_err := errors.New(
					"mirror directory count exceeds its limit")
				operation.Finish(Writes{}, directory_err)
				return
			}
			operation.Directories.PushBack(Traversal_Directory(relative))
			operation.Discovered.PushBack(struct{}{})
			advance()
			continue
		}
		plan_file_start(operation, relative)
		return
	}
	if operation.Directories.Len() != 0 {
		operation.Entries.Init()
		operation.Paths.Init()
		operation.Entry = nil
		operation.Path = nil
		operation.Next = operation.Directory_Step
		runner_queue(operation.Runner, operation.Continue)
		return
	}
	writes := Writes{}
	for element := operation.Writes.Front(); element != nil; element = element.Next() {
		writes.List.PushBack(element.Value)
	}
	operation.Finish(writes, nil)
}

// Plan_file_start reads one source and its destination before it records a required write.
func plan_file_start(operation *Plan_Operation, relative Relative_File_Path) {
	Plan_Operation_Invariants(operation, "plan_file_start.operation")
	Relative_File_Path_Invariants(relative, "plan_file_start.relative")
	destination := Destination_Path(filepath.Join(
		string(operation.Input.Destination_Directory), string(relative)))
	if len(destination) > DESTINATION_PATH_BYTES_MAX {
		operation.Finish(Writes{}, errors.New("mirror destination path exceeds its limit"))
		return
	}
	source := File_Path(filepath.Join(
		string(operation.Input.Source_Directory), string(relative)))
	file_read_start(
		operation.Runner, operation.Input.IO, source, DOTFILE_PAYLOAD_BYTES_MAX,
		FILE_OPERATION_SUBJECT_DOTFILE,
		func(contents File_Chunks, found File_Presence, read_err error) {
			if read_err != nil {
				operation.Finish(Writes{}, read_err)
				return
			}
			if !found {
				plan_file_complete(operation)
				return
			}
			plan_destination_start(
				operation, destination, file_chunks_dotfile(contents))
		},
	)
}

// Plan_destination_start treats an unreadable destination as a required write.
func plan_destination_start(
	operation *Plan_Operation, destination Destination_Path, source Dotfile_Bytes,
) {
	Plan_Operation_Invariants(operation, "plan_destination_start.operation")
	Destination_Path_Invariants(destination, "plan_destination_start.destination")
	Dotfile_Bytes_Invariants(source, "plan_destination_start.source")
	file_read_start(
		operation.Runner, operation.Input.IO, File_Path(destination),
		DOTFILE_PAYLOAD_BYTES_MAX, FILE_OPERATION_SUBJECT_DOTFILE,
		func(contents File_Chunks, found File_Presence, read_err error) {
			matches := false
			if read_err == nil {
				if found {
					matches = bool(dotfile_contents_equal(
						source, file_chunks_dotfile(contents)))
				}
			}
			if !matches {
				if operation.Writes.Len() == WRITE_COUNT_MAX {
					write_err := errors.New(
						"mirror write plan exceeds its limit")
					operation.Finish(Writes{}, write_err)
					return
				}
				write := File_Write{
					Destination_Path: destination,
					Contents:         Dotfile_Bytes(source),
				}
				File_Write_Invariants(write, "plan_destination_start.write")
				operation.Writes.PushBack(write)
			}
			plan_file_complete(operation)
		},
	)
}

// Plan_file_complete records the next entry only after both file reads retire.
func plan_file_complete(operation *Plan_Operation) {
	Plan_Operation_Invariants(operation, "plan_file_complete.operation")
	operation.Entry = operation.Entry.Next()
	operation.Path = operation.Path.Next()
	runner_queue(operation.Runner, operation.Continue)
}

// Mirror_Operation retains the write cursor after its asynchronous plan completes.
type Mirror_Operation struct {
	// Runner records each write or macOS process continuation.
	Runner *Runner_State
	// Input contains the original mirror dependencies.
	Input *Mirror_Input
	// Write is the next list element that the runner must apply.
	Write *list.Element
	// Continue gives the root the next write without a static call cycle.
	Continue func()
	// Finish receives the terminal mirror result.
	Finish func(succeeded Step_Success)
}

// Mirror_Operation_Invariants states the mirror facts and bounded plan.
func Mirror_Operation_Invariants(operation *Mirror_Operation, namespace invariant.Namespace) {
	Mirror_Input_Invariants(operation.Input, namespace)
	Runner_State_Invariants(operation.Runner, namespace)
	invariant.Always(operation.Continue != nil, "A mirror operation has a continuation.")
	invariant.Always(operation.Finish != nil, "A mirror operation has a consumer.")
}

// Mirror_start plans the complete sync before it queues any destination write.
func mirror_start(
	state *Runner_State, input *Mirror_Input,
	continuation func(succeeded Step_Success),
) {
	Runner_State_Invariants(state, "mirror_start.state")
	Mirror_Input_Invariants(input, "mirror_start.input")
	plan_start(state, &Plan_Input{
		IO: input.IO, Source_Directory: input.Source_Directory,
		Destination_Directory: input.Destination_Directory, Logger: input.Logger,
	}, func(writes Writes, plan_err error) {
		if plan_err != nil {
			jlog.Logger_Error(input.Logger, "plan failed", jlog.Err(plan_err))
			continuation(false)
			return
		}
		operation := &Mirror_Operation{
			Runner: state,
			Input:  input,
			Write:  writes.List.Front(),
			Finish: continuation,
		}
		operation.Continue = func() { mirror_write_rearm(operation) }
		if operation.Write == nil {
			jlog.Logger_Info(input.Logger, "dotfiles up to date")
			mirror_defaults_start(operation)
			return
		}
		runner_queue(state, operation.Continue)
	})
}

// Mirror_write_rearm applies one planned file and records the next list element.
func mirror_write_rearm(operation *Mirror_Operation) {
	Mirror_Operation_Invariants(operation, "mirror_write_rearm.operation")
	write := operation.Write.Value.(File_Write)
	file_write_start(
		operation.Runner, operation.Input.IO, write.Destination_Path,
		dotfile_chunks(write.Contents), FILE_OPERATION_SUBJECT_DOTFILE,
		func(write_err error) {
			if write_err != nil {
				jlog.Logger_Error(
					operation.Input.Logger, "write failed", jlog.Err(write_err))
				operation.Finish(false)
				return
			}
			jlog.Logger_Info(operation.Input.Logger, "wrote",
				jlog.String("path", write.Destination_Path))
			operation.Write = operation.Write.Next()
			if operation.Write == nil {
				mirror_defaults_start(operation)
				return
			}
			runner_queue(operation.Runner, operation.Continue)
		},
	)
}

// Mirror_defaults_start runs the fixed macOS process sequence only on Darwin.
func mirror_defaults_start(operation *Mirror_Operation) {
	Mirror_Operation_Invariants(operation, "mirror_defaults_start.operation")
	if operation.Input.Operating_System != "darwin" {
		operation.Finish(true)
		return
	}
	invocations := Process_Invocations{}
	for _, command := range macos_commands() {
		invocations = append(invocations, append(
			[]string{string(command.Name)}, []string(command.Arguments)...))
	}
	process_sequence_start(
		operation.Runner, operation.Input.IO,
		operation.Input.Stdout, operation.Input.Stderr, invocations, nil,
		func(succeeded Step_Success) {
			if !succeeded {
				jlog.Logger_Error(operation.Input.Logger, "macos defaults failed")
			}
			operation.Finish(succeeded)
		},
	)
}

// FONT_INDEX_MINIMUM is the first managed face position.
const FONT_INDEX_MINIMUM = 0

// Font_Index selects one face from the fixed managed list.
type Font_Index int

// Font_Index_Invariants bounds the cursor through the terminal list position.
func Font_Index_Invariants(index Font_Index, namespace invariant.Namespace) {
	invariant.Tree(index, namespace).
		Range_Int(int(index), FONT_INDEX_MINIMUM, IOSEVKA_FONT_COUNT).
		Ensure()
}

// Font_Copied reports whether setup must refresh the platform font cache.
type Font_Copied bool

// Font_Copied_Invariants requires copied and converged runs across the seed sweep.
func Font_Copied_Invariants(copied Font_Copied, namespace invariant.Namespace) {
	invariant.Tree(copied, namespace).
		Sometimes(bool(copied), "A font installation copied a face.").
		Ensure()
}

// Font_Install_Operation retains the fixed face index between asynchronous copies.
type Font_Install_Operation struct {
	// Runner records each absent font copy.
	Runner *Runner_State
	// Input contains the original shared IO and font paths.
	Input *Install_Fonts_Input
	// Files is the fixed managed face list.
	Files File_Paths
	// Index selects the next managed face.
	Index Font_Index
	// Copied reports whether the cache refresh is required.
	Copied Font_Copied
	// Continue gives the root the next face without a static call cycle.
	Continue func()
	// Finish receives the terminal font install result.
	Finish func(succeeded Step_Success)
}

// Font_Install_Operation_Invariants states its fixed file list and required continuations.
func Font_Install_Operation_Invariants(
	operation *Font_Install_Operation, namespace invariant.Namespace,
) {
	Install_Fonts_Input_Invariants(operation.Input, namespace)
	File_Paths_Invariants(operation.Files, namespace)
	Runner_State_Invariants(operation.Runner, namespace)
	Font_Index_Invariants(operation.Index, namespace)
	Font_Copied_Invariants(operation.Copied, namespace)
	invariant.Always(operation.Continue != nil, "A font install has a continuation.")
	invariant.Always(operation.Finish != nil, "A font install has a consumer.")
	invariant.Always(operation.Index >= 0, "A font install index is not negative.")
	invariant.Always(
		int(operation.Index) <= len(operation.Files),
		"A font install index stays inside its face list.",
	)
}

// Install_fonts_start records the fixed face list before it evaluates the first status.
func install_fonts_start(
	state *Runner_State, input *Install_Fonts_Input,
	continuation func(succeeded Step_Success),
) {
	Runner_State_Invariants(state, "install_fonts_start.state")
	Install_Fonts_Input_Invariants(input, "install_fonts_start.input")
	operation := &Font_Install_Operation{
		Runner: state, Input: input, Files: iosevka_font_files(), Finish: continuation,
	}
	operation.Continue = func() { install_font_rearm(operation) }
	runner_queue(state, operation.Continue)
}

// Install_font_rearm advances present faces and starts one absent face copy.
func install_font_rearm(operation *Font_Install_Operation) {
	Font_Install_Operation_Invariants(operation, "install_font_rearm.operation")
	for int(operation.Index) < len(operation.Files) {
		file := operation.Files[int(operation.Index)]
		destination := Font_Path(filepath.Join(
			string(operation.Input.Font_Directory), file))
		if file_present(operation.Input.IO, destination) {
			jlog.Logger_Info(operation.Input.Logger, "font present, skipped",
				jlog.String("file", file))
			operation.Index++
			continue
		}
		copy_file_start(operation.Runner, operation.Input.IO, &File_Copy_Input{
			Source: Font_Source_Path(filepath.Join(
				string(operation.Input.Home_Directory), IOSEVKA_SUBPATH, file)),
			Destination: Font_Destination_Path(destination),
		}, func(copy_err error) {
			if copy_err != nil {
				jlog.Logger_Error(operation.Input.Logger, "font copy failed",
					jlog.Err(copy_err))
				operation.Finish(false)
				return
			}
			jlog.Logger_Info(
				operation.Input.Logger, "copied font", jlog.String("file", file))
			operation.Copied = true
			operation.Index++
			runner_queue(operation.Runner, operation.Continue)
		})
		return
	}
	if !operation.Copied {
		operation.Finish(true)
		return
	}
	if !operation.Input.Refresh_Cache {
		operation.Finish(true)
		return
	}
	run_spawn_start(
		operation.Runner, operation.Input.IO,
		operation.Input.Stdout, operation.Input.Stderr,
		Spawn_Arguments{"fc-cache", "-f", string(operation.Input.Font_Directory)},
		func(succeeded Step_Success) {
			if !succeeded {
				jlog.Logger_Error(
					operation.Input.Logger, "font cache refresh failed")
			}
			operation.Finish(succeeded)
		},
	)
}

// BOOTSTRAP_STEP_INDEX_MINIMUM is the first stage position.
const BOOTSTRAP_STEP_INDEX_MINIMUM = 0

// BOOTSTRAP_STEP_INDEX_MAXIMUM is the last stage position.
const BOOTSTRAP_STEP_INDEX_MAXIMUM = BOOTSTRAP_STEP_COUNT - 1

// Bootstrap_Step_Index selects one stage from the complete bootstrap plan.
type Bootstrap_Step_Index int

// Bootstrap_Step_Index_Invariants prevents a queued stage from passing the plan boundary.
func Bootstrap_Step_Index_Invariants(
	index Bootstrap_Step_Index, namespace invariant.Namespace,
) {
	invariant.Tree(index, namespace).
		Range_Int(
			int(index), BOOTSTRAP_STEP_INDEX_MINIMUM,
			BOOTSTRAP_STEP_INDEX_MAXIMUM).
		Ensure()
}

// Runner_Step is one named asynchronous bootstrap stage.
type Runner_Step struct {
	// Name is the progress label that precedes the stage.
	Name Step_Name
	// Start submits the first operation or reports an inline result.
	Start func(state *Runner_State, continuation func(succeeded Step_Success))
}

// Runner_Step_Invariants states the stage label and required start operation.
func Runner_Step_Invariants(step Runner_Step, namespace invariant.Namespace) {
	Step_Name_Invariants(step.Name, namespace)
	invariant.Always(step.Start != nil, "A runner step has a start operation.")
}

// Runner_Steps is the fixed asynchronous bootstrap plan.
type Runner_Steps []Runner_Step

// Runner_Steps_Invariants requires every bootstrap stage.
func Runner_Steps_Invariants(steps Runner_Steps, _ invariant.Namespace) {
	invariant.Always(
		len(steps) == BOOTSTRAP_STEP_COUNT, "A setup runner has every bootstrap step.")
}

// RUNNER_HOST_STEP_COUNT is each fixed host-policy section size.
const RUNNER_HOST_STEP_COUNT = 5

// RUNNER_COMMAND_STEP_COUNT is the fixed repository-command section size.
const RUNNER_COMMAND_STEP_COUNT = 4

// Runner_Host_Steps is one fixed five-stage host-policy section.
type Runner_Host_Steps []Runner_Step

// Runner_Host_Steps_Invariants keeps both host sections structurally identical.
func Runner_Host_Steps_Invariants(steps Runner_Host_Steps, _ invariant.Namespace) {
	invariant.Always(
		len(steps) == RUNNER_HOST_STEP_COUNT,
		"A runner host section has five steps.",
	)
}

// Runner_Command_Step keeps a repository command inside its narrower name domain.
type Runner_Command_Step struct {
	// Name stays narrower than labels such as jj and dotfiles.
	Name Command_Name
	// Start preserves the same callback-owned transition as a complete runner step.
	Start func(state *Runner_State, continuation func(succeeded Step_Success))
}

// Runner_Command_Step_Invariants states one repository command stage.
func Runner_Command_Step_Invariants(
	step Runner_Command_Step, namespace invariant.Namespace,
) {
	Command_Name_Invariants(step.Name, namespace)
	invariant.Always(step.Start != nil, "A runner command step has a start operation.")
}

// Runner_Command_Steps is the fixed repository-command section.
type Runner_Command_Steps []Runner_Command_Step

// Runner_Command_Steps_Invariants requires every repository command stage.
func Runner_Command_Steps_Invariants(steps Runner_Command_Steps, _ invariant.Namespace) {
	invariant.Always(
		len(steps) == RUNNER_COMMAND_STEP_COUNT,
		"A runner command section has four steps.",
	)
}

// Bootstrap_Operation retains the current step between process and file completions.
type Bootstrap_Operation struct {
	// Runner records each next stage.
	Runner *Runner_State
	// Steps is the fixed complete bootstrap plan.
	Steps Runner_Steps
	// Index selects the next stage.
	Index Bootstrap_Step_Index
	// Logger records each stage before it starts.
	Logger jlog.Logger
	// Continue gives the root the next stage without a static call cycle.
	Continue func()
}

// Bootstrap_Operation_Invariants states its fixed plan and current index.
func Bootstrap_Operation_Invariants(
	operation *Bootstrap_Operation, namespace invariant.Namespace,
) {
	Runner_Steps_Invariants(operation.Steps, namespace)
	Runner_State_Invariants(operation.Runner, namespace)
	Bootstrap_Step_Index_Invariants(operation.Index, namespace)
	invariant.Always(operation.Continue != nil, "A bootstrap operation has a continuation.")
	invariant.Always(operation.Index >= 0, "A bootstrap step index is not negative.")
	invariant.Always(
		int(operation.Index) < len(operation.Steps),
		"A bootstrap step index stays inside its complete plan.",
	)
}

// Bootstrap_runner_start records the fixed plan before the root submits its first stage.
func bootstrap_runner_start(
	state *Runner_State, steps Runner_Steps, logger jlog.Logger,
) {
	Runner_State_Invariants(state, "bootstrap_runner_start.state")
	Runner_Steps_Invariants(steps, "bootstrap_runner_start.steps")
	operation := &Bootstrap_Operation{Runner: state, Steps: steps, Logger: logger}
	operation.Continue = func() { bootstrap_runner_rearm(operation) }
	runner_queue(state, operation.Continue)
}

// Bootstrap_runner_rearm starts one stage and records its successor after success.
func bootstrap_runner_rearm(operation *Bootstrap_Operation) {
	Bootstrap_Operation_Invariants(operation, "bootstrap_runner_rearm.operation")
	step := operation.Steps[int(operation.Index)]
	Runner_Step_Invariants(step, "bootstrap_runner_rearm.step")
	jlog.Logger_Info(operation.Logger, "step", jlog.String("name", step.Name))
	step.Start(operation.Runner, func(succeeded Step_Success) {
		if !succeeded {
			runner_stop(operation.Runner, EXIT_FAILURE)
			return
		}
		operation.Index++
		if int(operation.Index) == len(operation.Steps) {
			runner_stop(operation.Runner, EXIT_SUCCESS)
			return
		}
		runner_queue(operation.Runner, operation.Continue)
	})
}

// Runner_steps builds the complete asynchronous policy from validated host facts.
func runner_steps(input *Bootstrap_Steps_Input) (steps Runner_Steps) {
	defer func() { Runner_Steps_Invariants(steps, "runner_steps.steps") }()
	Bootstrap_Steps_Input_Invariants(input, "runner_steps.input")
	repository := Repository_Directory(filepath.Join(
		string(input.Home_Directory), REPOSITORY_SUBPATH))
	binary_directory := Binary_Directory(filepath.Join(
		string(repository), "home", ".local", "bin"))
	font_directory, refresh_cache, font_supported := font_destination(&Font_Destination_Input{
		Home_Directory: input.Home_Directory, Operating_System: input.Operating_System,
		Data_Directory: input.Data_Directory,
	})
	steps = append(steps, runner_initial_steps(
		input, repository, binary_directory,
		font_directory, refresh_cache, font_supported)...)
	for _, command := range runner_command_steps(
		input, repository, binary_directory,
	) {
		steps = append(steps, Runner_Step{
			Name: Step_Name(command.Name), Start: command.Start,
		})
	}
	steps = append(steps, runner_final_steps(
		input, repository, binary_directory)...)
	return steps
}

// Runner_initial_steps builds the file and first tool section.
func runner_initial_steps(
	input *Bootstrap_Steps_Input, repository Repository_Directory,
	binary_directory Binary_Directory, font_directory Font_Directory,
	refresh_cache Cache_Refresh, font_supported Font_Support,
) (steps Runner_Host_Steps) {
	defer func() {
		Runner_Host_Steps_Invariants(steps, "runner_initial_steps.steps")
	}()
	Bootstrap_Steps_Input_Invariants(input, "runner_initial_steps.input")
	Repository_Directory_Invariants(repository, "runner_initial_steps.repository")
	Binary_Directory_Invariants(binary_directory, "runner_initial_steps.binary_directory")
	Font_Directory_Invariants(font_directory, "runner_initial_steps.font_directory")
	Cache_Refresh_Invariants(refresh_cache, "runner_initial_steps.refresh_cache")
	Font_Support_Invariants(font_supported, "runner_initial_steps.font_supported")
	return Runner_Host_Steps{
		{Name: "direnv", Start: func(
			state *Runner_State, continuation func(succeeded Step_Success),
		) {
			install_direnv_start(state, &Install_Direnv_Input{
				Direnv_Directory: Direnv_Directory(filepath.Join(
					string(repository), "third_party", "direnv")),
				Binary_Directory: binary_directory,
				IO:               input.IO, Logger: input.Logger,
				Stdout: input.Stdout, Stderr: input.Stderr,
			}, continuation)
		}},
		{Name: "dotfiles", Start: func(
			state *Runner_State, continuation func(succeeded Step_Success),
		) {
			mirror_start(state, &Mirror_Input{
				IO: input.IO, Source_Directory: Source_Directory(filepath.Join(
					string(input.Home_Directory), DOTFILES_SUBPATH)),
				Destination_Directory: Destination_Directory(input.Home_Directory),
				Operating_System:      input.Operating_System, Logger: input.Logger,
				Stdout: input.Stdout, Stderr: input.Stderr,
			}, continuation)
		}},
		{Name: "fonts", Start: func(
			state *Runner_State, continuation func(succeeded Step_Success),
		) {
			if !font_supported {
				continuation(true)
				return
			}
			install_fonts_start(state, &Install_Fonts_Input{
				IO: input.IO, Home_Directory: input.Home_Directory,
				Font_Directory: font_directory, Refresh_Cache: refresh_cache,
				Logger: input.Logger, Stdout: input.Stdout, Stderr: input.Stderr,
			}, continuation)
		}},
		{Name: "neovim", Start: func(
			state *Runner_State, continuation func(succeeded Step_Success),
		) {
			install_neovim_start(state, &Install_Neovim_Input{
				Repository_Directory: Repository_Directory(repository),
				IO:                   input.IO, Logger: input.Logger,
				Stdout: input.Stdout, Stderr: input.Stderr,
			}, continuation)
		}},
		{Name: "fzf", Start: func(
			state *Runner_State, continuation func(succeeded Step_Success),
		) {
			install_fzf_start(state, &Install_Fzf_Input{
				Fzf_Directory: Fzf_Directory(filepath.Join(
					string(repository), "third_party", "fzf")),
				Binary_Directory: binary_directory,
				IO:               input.IO, Logger: input.Logger,
				Stdout: input.Stdout, Stderr: input.Stderr,
			}, continuation)
		}},
	}
}

// Runner_command_steps builds the four repository command stages.
func runner_command_steps(
	input *Bootstrap_Steps_Input, repository Repository_Directory,
	binary_directory Binary_Directory,
) (steps Runner_Command_Steps) {
	defer func() {
		Runner_Command_Steps_Invariants(steps, "runner_command_steps.steps")
	}()
	Bootstrap_Steps_Input_Invariants(input, "runner_command_steps.input")
	Repository_Directory_Invariants(repository, "runner_command_steps.repository")
	Binary_Directory_Invariants(binary_directory, "runner_command_steps.binary_directory")
	return Runner_Command_Steps{
		runner_command_step(input, repository, binary_directory, "maddox", "maddox"),
		runner_command_step(
			input, repository, binary_directory, "markdown_to_pdf", "m2p"),
		runner_command_step(input, repository, binary_directory, "sloc", "sloc"),
		runner_command_step(input, repository, binary_directory, "timeout", "timeout"),
	}
}

// Runner_final_steps builds the toolchain and platform section.
func runner_final_steps(
	input *Bootstrap_Steps_Input, repository Repository_Directory,
	binary_directory Binary_Directory,
) (steps Runner_Host_Steps) {
	defer func() {
		Runner_Host_Steps_Invariants(steps, "runner_final_steps.steps")
	}()
	Bootstrap_Steps_Input_Invariants(input, "runner_final_steps.input")
	Repository_Directory_Invariants(repository, "runner_final_steps.repository")
	Binary_Directory_Invariants(binary_directory, "runner_final_steps.binary_directory")
	return Runner_Host_Steps{
		{Name: "rust", Start: func(
			state *Runner_State, continuation func(succeeded Step_Success),
		) {
			install_rust_start(state, &Install_Rust_Input{
				Cargo_Directory: input.Cargo_Directory,
				Link_Directory: Rust_Link_Directory(filepath.Join(
					string(repository), ".local", "bin")),
				IO: input.IO, Logger: input.Logger,
				Stdout: input.Stdout, Stderr: input.Stderr,
			}, continuation)
		}},
		{Name: "jj", Start: func(
			state *Runner_State, continuation func(succeeded Step_Success),
		) {
			install_jj_start(state, &Install_Jj_Input{
				Jj_Directory: Jj_Directory(filepath.Join(
					string(repository), "third_party", "jj")),
				Binary_Directory: binary_directory,
				IO:               input.IO, Logger: input.Logger,
				Stdout: input.Stdout, Stderr: input.Stderr,
			}, continuation)
		}},
		{Name: "ripgrep", Start: func(
			state *Runner_State, continuation func(succeeded Step_Success),
		) {
			install_ripgrep_start(state, &Install_Ripgrep_Input{
				Ripgrep_Directory: Ripgrep_Directory(filepath.Join(
					string(repository), "third_party", "ripgrep")),
				Binary_Directory: binary_directory,
				IO:               input.IO, Logger: input.Logger,
				Stdout: input.Stdout, Stderr: input.Stderr,
			}, continuation)
		}},
		{Name: "fd", Start: func(
			state *Runner_State, continuation func(succeeded Step_Success),
		) {
			install_fdcli_start(state, &Install_Fdcli_Input{
				Fdcli_Directory: Fdcli_Directory(filepath.Join(
					string(repository), "third_party", "fd")),
				Binary_Directory: binary_directory,
				IO:               input.IO, Logger: input.Logger,
				Stdout: input.Stdout, Stderr: input.Stderr,
			}, continuation)
		}},
		{Name: "ghostty", Start: func(
			state *Runner_State, continuation func(succeeded Step_Success),
		) {
			if input.Operating_System != "darwin" {
				continuation(true)
				return
			}
			install_ghostty_start(state, &Install_Ghostty_Input{
				Applications_Directory: "/Applications",
				Link_Directory: Ghostty_Link_Directory(filepath.Join(
					string(repository), "home", ".local", "bin")),
				IO: input.IO, Logger: input.Logger,
				Stdout: input.Stdout, Stderr: input.Stderr,
			}, continuation)
		}},
	}
}

// Runner_command_step creates one repository command stage from its package and binary names.
func runner_command_step(
	input *Bootstrap_Steps_Input, repository Repository_Directory,
	binary_directory Binary_Directory,
	package_directory Package_Directory, binary_name Command_Name,
) (step Runner_Command_Step) {
	defer func() {
		Runner_Command_Step_Invariants(step, "runner_command_step.step")
	}()
	Bootstrap_Steps_Input_Invariants(input, "runner_command_step.input")
	Repository_Directory_Invariants(repository, "runner_command_step.repository")
	Binary_Directory_Invariants(binary_directory, "runner_command_step.binary_directory")
	Package_Directory_Invariants(package_directory, "runner_command_step.package_directory")
	Command_Name_Invariants(binary_name, "runner_command_step.binary_name")
	return Runner_Command_Step{Name: binary_name, Start: func(
		state *Runner_State, continuation func(succeeded Step_Success),
	) {
		install_command_start(state, &Install_Command_Input{
			Package_Directory: Command_Directory(filepath.Join(
				string(repository), string(package_directory))),
			Binary_Directory: binary_directory, Binary_Name: binary_name,
			IO: input.IO, Logger: input.Logger,
			Stdout: input.Stdout, Stderr: input.Stderr,
		}, continuation)
	}}
}
