// Package setup is the pure library tier of the setup binary, the one-shot
// bootstrap for this machine. It mirrors a tree of dotfiles into the home
// directory in one direction, repo to home — Plan diffs two injected filesystems
// and returns the writes a sync would make, and Mirror performs them through an
// injected writer — Install_Neovim builds and installs the vendored Neovim, and
// Install_Fonts copies the vendored fonts into the OS font directory. Every entry
// binds to no real filesystem and stays a black box under test.
package setup

import (
	"bytes"
	"container/list"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"

	invariant "local/james-orcales/shared/invariant/default"
	sysio "local/james-orcales/shared/io"
	"local/james-orcales/shared/jlog"
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

// ROOT_REFUSAL prevents the bootstrap from creating root-owned home files.
const ROOT_REFUSAL = "run as your normal user, not root"

// Main constructs the complete setup policy from injected capabilities and returns the first
// failing status. Package main binds the real world and calls only this setup entry point.
func Main(input *Main_Input) (status_code Exit_Code) {
	defer func() { Exit_Code_Invariants(status_code, "Main.status_code") }()
	Main_Input_Invariants(input, "Main.input")
	environment, valid := New_Environment(input.Environment)
	if !valid {
		return EXIT_USAGE
	}
	input.Shell.Logger = environment.Logger
	input.Shell.Stdout = environment.Stdout
	input.Shell.Stderr = environment.Stderr
	steps := Bootstrap_Steps(&Bootstrap_Steps_Input{
		Home_Directory:   environment.Home_Directory,
		Operating_System: environment.Operating_System,
		Cargo_Directory:  environment.Cargo_Directory,
		Data_Directory:   environment.Data_Directory,
		File_System:      input.File_System,
		Shell:            input.Shell,
	})
	if !Bootstrap(&Bootstrap_Input{Steps: steps, Logger: environment.Logger}) {
		return EXIT_FAILURE
	}
	return EXIT_SUCCESS
}

// Environment_Input contains operating-system facts. Package main reads each fact once, while
// this package owns validation and logger policy.
type Environment_Input struct {
	// Clock supplies timestamped logging.
	Clock systime.Clock
	// Console receives setup log records.
	Console io.Writer
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
	Stdout io.Writer
	// Stderr receives live command diagnostics.
	Stderr io.Writer
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
	Stdout io.Writer
	// Stderr receives live command diagnostics.
	Stderr io.Writer
}

// Environment_Invariants states each validated environment fact that has a preset.
func Environment_Invariants(environment Environment, namespace invariant.Namespace) {
	Home_Directory_Invariants(environment.Home_Directory, namespace)
	Operating_System_Invariants(environment.Operating_System, namespace)
	Cargo_Directory_Invariants(environment.Cargo_Directory, namespace)
	Data_Directory_Invariants(environment.Data_Directory, namespace)
}

// File_System gives the sync synchronous file operations. The composition root converts its
// asynchronous IO into these operations because only the composition root can drive the loop.
type File_System struct {
	// Read_Directory returns the immediate entries in one directory.
	Read_Directory func(path string) (entries []sysio.Directory_Entry, err error)
	// Status reports whether a path exists without an asynchronous operation.
	Status func(path string) (status sysio.File_Status, err error)
	// Read returns one bounded file and reports an absent path without an error.
	Read func(path string, buffer_size int) (contents []byte, found bool, err error)
	// Write replaces one file after it creates the necessary parent directories.
	Write func(path string, contents []byte) (err error)
}

// File_System_Invariants states that each required file-system operation is present.
func File_System_Invariants(system File_System, namespace invariant.Namespace) {
	invariant.Always(system.Read_Directory != nil, "A file system has a directory reader.")
	invariant.Always(system.Status != nil, "A file system has a status reader.")
	invariant.Always(system.Read != nil, "A file system has a file reader.")
	invariant.Always(system.Write != nil, "A file system has a file writer.")
}

// Transition_Ready reports that one callback-owned transition retired.
type Transition_Ready func() (ready bool)

// Transition_Complete reports that one callback-owned operation retired all transitions.
type Transition_Complete func() (complete bool)

// Transition_Rearm submits the next callback-owned transition.
type Transition_Rearm func()

// Asynchronous_File_Read_Result reports one result after the root observes completion.
type Asynchronous_File_Read_Result func() (contents []byte, found bool, err error)

// Asynchronous_File_Read submits one bounded read without advancing the IO timeline.
type Asynchronous_File_Read func(
	path string, size int,
) (
	ready Transition_Ready, complete Transition_Complete,
	rearm Transition_Rearm, result Asynchronous_File_Read_Result,
)

// Asynchronous_File_Open validates one read request before it returns a descriptor.
type Asynchronous_File_Open func(
	path string, size int,
) (file sysio.File, present File_Presence, err error)

// Asynchronous_File_Write_Result reports one result after the root observes completion.
type Asynchronous_File_Write_Result func() (err error)

// Asynchronous_File_Write submits one bounded write without advancing the IO timeline.
type Asynchronous_File_Write func(
	path string, contents []byte,
) (
	ready Transition_Ready, complete Transition_Complete,
	rearm Transition_Rearm, result Asynchronous_File_Write_Result,
)

// Asynchronous_Process_Result reports one result after the root observes completion.
type Asynchronous_Process_Result func() (result sysio.Process_Result)

// Asynchronous_Process submits one bounded process without advancing the IO timeline.
type Asynchronous_Process func(
	request sysio.Process_Request,
) (
	ready Transition_Ready, complete Transition_Complete,
	rearm Transition_Rearm, result Asynchronous_Process_Result,
)

// File_Read_Adapter binds callback sequencing without receiving the root Driver.
func File_Read_Adapter(loop sysio.IO) (read Asynchronous_File_Read) {
	open := File_Open_Adapter(loop)
	return func(path string, size int) (
		ready Transition_Ready, complete Transition_Complete,
		rearm Transition_Rearm, result Asynchronous_File_Read_Result,
	) {
		retired, done, close_next := false, false, false
		completion, buffer := sysio.Completion{}, make([]byte, size)
		total, found := 0, false
		contents, result_err := []byte(nil), error(nil)
		file, present, open_err := open(path, size)
		result_err = open_err
		done = open_err != nil || !bool(present)
		var submit, close_submit func()
		close_submit = func() {
			loop.Close(&completion, func(_ *sysio.Completion, close_err error) {
				result_err = errors.Join(result_err, close_err)
				done, retired = true, true
			}, file)
		}
		submit = func() {
			loop.Read(&completion, func(
				_ *sysio.Completion, count int, read_err error,
			) {
				if read_err != nil {
					result_err, close_next, retired = read_err, true, true
					return
				}
				if count < 0 {
					result_err = errors.New(
						"the file read returned a negative byte count")
					close_next, retired = true, true
					return
				}
				if count > len(buffer)-total {
					result_err = errors.New(
						"the file read returned an invalid byte count")
					close_next, retired = true, true
					return
				}
				total += count
				if count == 0 {
					contents, found, close_next = buffer[:total], true, true
				}
				if total == len(buffer) {
					result_err = errors.New("the file exceeds its limit")
					close_next = true
				}
				retired = true
			}, file, buffer[total:], int64(total))
		}
		ready = func() (ready bool) { return retired }
		complete = func() (complete bool) { return done }
		rearm = func() {
			retired = false
			if close_next {
				close_submit()
				return
			}
			submit()
		}
		result = func() (read []byte, present bool, err error) {
			return contents, found, result_err
		}
		if !done {
			submit()
		}
		return ready, complete, rearm, result
	}
}

// File_Open_Adapter keeps unvalidated root values outside the typed setup policy.
func File_Open_Adapter(loop sysio.IO) (open Asynchronous_File_Open) {
	return func(path string, size int) (
		file sysio.File, present File_Presence, err error,
	) {
		if len(path) < DESTINATION_PATH_BYTES_MIN {
			return 0, false, errors.New("the file path is outside its limit")
		}
		if len(path) > DESTINATION_PATH_BYTES_MAX {
			return 0, false, errors.New("the file path is outside its limit")
		}
		if size != DOTFILE_BYTES_MAX {
			if size != COPY_BYTES_MAX {
				return 0, false, errors.New("the file byte limit is invalid")
			}
		}
		status, status_err := loop.Status(path)
		if status_err != nil {
			return 0, false, status_err
		}
		if !status.Exists {
			return 0, false, nil
		}
		if !status.Is_Regular {
			return 0, true, errors.New("the file path is not a regular file")
		}
		file, err = loop.Open(path)
		return file, true, err
	}
}

// Asynchronous_File_Create validates one write request before it returns a descriptor.
type Asynchronous_File_Create func(path string) (file sysio.File, err error)

// File_Write_Adapter binds callback sequencing without receiving the root Driver.
func File_Write_Adapter(loop sysio.IO) (write Asynchronous_File_Write) {
	create := File_Create_Adapter(loop)
	return func(path string, contents []byte) (
		ready Transition_Ready, complete Transition_Complete,
		rearm Transition_Rearm, result Asynchronous_File_Write_Result,
	) {
		retired, done, close_next := false, false, false
		completion := sysio.Completion{}
		written := 0
		var result_err error
		file, create_err := create(path)
		if create_err != nil {
			result_err, done = create_err, true
		}
		var submit, close_submit func()
		close_submit = func() {
			loop.Close(&completion, func(_ *sysio.Completion, close_err error) {
				result_err = errors.Join(result_err, close_err)
				done, retired = true, true
			}, file)
		}
		submit = func() {
			if written == len(contents) {
				close_submit()
				return
			}
			chunk := contents[written:]
			loop.Write(&completion, func(
				_ *sysio.Completion, count int, write_err error,
			) {
				if write_err != nil {
					result_err, close_next, retired = write_err, true, true
					return
				}
				if count <= 0 {
					result_err = errors.New("the file write made no progress")
					close_next, retired = true, true
					return
				}
				if count > len(chunk) {
					result_err = errors.New(
						"the file write returned an invalid byte count")
					close_next, retired = true, true
					return
				}
				written, retired = written+count, true
			}, file, chunk, int64(written))
		}
		ready = func() (ready bool) { return retired }
		complete = func() (complete bool) { return done }
		rearm = func() {
			retired = false
			if close_next {
				close_submit()
				return
			}
			submit()
		}
		result = func() (err error) { return result_err }
		if !done {
			submit()
		}
		return ready, complete, rearm, result
	}
}

// File_Create_Adapter keeps unvalidated root values outside the typed setup policy.
func File_Create_Adapter(loop sysio.IO) (create Asynchronous_File_Create) {
	return func(path string) (file sysio.File, err error) {
		if len(path) < DESTINATION_PATH_BYTES_MIN {
			return 0, errors.New("the file path is outside its limit")
		}
		if len(path) > DESTINATION_PATH_BYTES_MAX {
			return 0, errors.New("the file path is outside its limit")
		}
		if mkdir_err := loop.Make_Directory(filepath.Dir(path)); mkdir_err != nil {
			return 0, mkdir_err
		}
		return loop.Create(path)
	}
}

// Process_Adapter binds callback state without receiving the root Driver.
func Process_Adapter(loop sysio.IO) (process Asynchronous_Process) {
	return func(request sysio.Process_Request) (
		ready Transition_Ready, complete Transition_Complete,
		rearm Transition_Rearm, result Asynchronous_Process_Result,
	) {
		done := false
		completed := sysio.Process_Result{}
		var result_err error
		completion := sysio.Completion{}
		loop.Spawn(&completion, func(
			_ *sysio.Completion, process_result sysio.Process_Result, err error,
		) {
			completed, result_err, done = process_result, err, true
		}, request, PROCESS_DURATION_MAX)
		ready = func() (ready bool) { return done }
		complete = func() (complete bool) { return done }
		rearm = func() {}
		result = func() (process_result sysio.Process_Result) {
			process_result = completed
			if result_err != nil {
				process_result.Exit = 1
			}
			return process_result
		}
		return ready, complete, rearm, result
	}
}

// Main_Input contains the complete injected environment for one setup run.
type Main_Input struct {
	// Environment contains the operating-system facts that Main validates.
	Environment *Environment_Input
	// File_System supplies all setup file operations.
	File_System File_System
	// Shell supplies all setup process operations.
	Shell Shell
}

// Main_Input_Invariants states the complete setup dependency set.
func Main_Input_Invariants(input *Main_Input, namespace invariant.Namespace) {
	Environment_Input_Invariants(input.Environment, namespace)
	File_System_Invariants(input.File_System, namespace)
	Shell_Invariants(input.Shell, namespace)
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
	logger := jlog.New_Console_Logger(jlog.New_Console_Logger_Input{
		Console: input.Console, Color: bool(input.Color),
		Floor: jlog.LEVEL_DEBUG, Clock: input.Clock,
	})
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
	// File_System is the loop the dotfiles tree is walked, read, and written through.
	File_System File_System
	// Source_Directory is the absolute dotfiles tree walked in full from its root.
	Source_Directory Source_Directory
	// Destination_Directory is the absolute home directory writes land under, and the diff
	// reads existing files from.
	Destination_Directory Destination_Directory
	// Operating_System gates the macos defaults step, which runs only on
	// "darwin" (a runtime.GOOS value).
	Operating_System Operating_System
	// Run_Command runs an external program by name with arguments. It is injected
	// so this library tier never executes anything; package main supplies the
	// exec-backed runner used for the macos defaults.
	Run_Command func(name string, arguments []string) (err error)
	// Is_Ignored classifies a batch of source paths, returning the set that is
	// gitignored; it is threaded to Plan so the install tree under .local is not
	// mirrored into home. Batched because the gitignore probe is a subprocess and one
	// spawn per path made a large tree scan for seconds. One call classifies one
	// directory. Nil ignores nothing.
	Is_Ignored func(relative_paths []string) (ignored map[string]bool)
	// Logger records the sync's narration: one line per file written, the scan's
	// progress, and a diagnostic when planning or a write fails. The zero Logger is a
	// disabled no-op, so a caller wanting silence passes none.
	Logger jlog.Logger
}

// Mirror_Input_Invariants states the mirror dependency set and its path facts.
func Mirror_Input_Invariants(input *Mirror_Input, namespace invariant.Namespace) {
	File_System_Invariants(input.File_System, namespace)
	Source_Directory_Invariants(input.Source_Directory, namespace)
	Destination_Directory_Invariants(input.Destination_Directory, namespace)
	Operating_System_Invariants(input.Operating_System, namespace)
}

// Mirror syncs the source dotfiles into the home directory and, on darwin, applies the macos
// defaults. The separate name keeps Main as the complete binary policy entry point.
func Mirror(input *Mirror_Input) (succeeded Step_Success) {
	defer func() { Step_Success_Invariants(succeeded, "Mirror.succeeded") }()
	Mirror_Input_Invariants(input, "Mirror.input")
	writes, plan_err := Plan(&Plan_Input{
		File_System:           input.File_System,
		Source_Directory:      input.Source_Directory,
		Destination_Directory: input.Destination_Directory,
		Is_Ignored:            input.Is_Ignored,
		Logger:                input.Logger,
	})
	if plan_err != nil {
		jlog.Logger_Error(input.Logger, "plan failed", jlog.Err(plan_err))
		return false
	}
	for element := writes.List.Front(); element != nil; element = element.Next() {
		write := element.Value.(File_Write)
		write_err := input.File_System.Write(
			string(write.Destination_Path), []byte(write.Contents))
		if write_err != nil {
			jlog.Logger_Error(input.Logger, "write failed", jlog.Err(write_err))
			return false
		}
		jlog.Logger_Info(input.Logger, "wrote", jlog.String("path", write.Destination_Path))
	}
	// A converged tree writes nothing, so without a closing line the step would look
	// stuck after the last scan line; say it finished and had no work.
	if writes.List.Len() == 0 {
		jlog.Logger_Info(input.Logger, "dotfiles up to date")
	}
	// The macos defaults touch macOS-only preference domains, so they run there
	// and nowhere else.
	if input.Operating_System != "darwin" {
		return true
	}
	return apply_macos_defaults(input.Run_Command, input.Logger)
}

// Reports whether the injected file system contains a path. A status error cannot prove that the
// file exists, so the idempotency gate treats that result as absent.
func file_present(system File_System, path Font_Path) (present File_Presence) {
	defer func() { File_Presence_Invariants(present, "file_present.present") }()
	File_System_Invariants(system, "file_present.system")
	Font_Path_Invariants(path, "file_present.path")
	status, status_err := system.Status(string(path))
	if status_err != nil {
		return false
	}
	return File_Presence(status.Exists)
}

// Copies one bounded file through the injected file system. The larger font limit stays separate
// from the dotfile limit because a font is binary installation data, not configuration text.
func copy_file(system File_System, input *File_Copy_Input) (err error) {
	File_System_Invariants(system, "copy_file.system")
	File_Copy_Input_Invariants(input, "copy_file.input")
	contents, found, read_err := system.Read(string(input.Source), COPY_BYTES_MAX)
	if read_err != nil {
		return read_err
	}
	if !found {
		return errors.New("copy source is absent")
	}
	return system.Write(string(input.Destination), contents)
}

// Takes just the runner and the logger it uses, not the whole Mirror_Input. Runs
// every macos defaults command, stopping at the first that fails. It logs
// nothing per command — 35 lines of `defaults write` is noise, not progress.
func apply_macos_defaults(
	run func(name string, arguments []string) (err error), logger jlog.Logger,
) (succeeded Step_Success) {
	defer func() { Step_Success_Invariants(succeeded, "apply_macos_defaults.succeeded") }()
	for _, command := range macos_commands() {
		run_err := run(string(command.Name), []string(command.Arguments))
		if run_err != nil {
			jlog.Logger_Error(logger, "macos defaults failed", jlog.Err(run_err))
			return false
		}
	}
	return true
}

// Step is one named stage of the bootstrap: the label Bootstrap announces before
// running it, and the work itself.
type Step struct {
	// Name is the label Bootstrap announces before running this step.
	Name Step_Name
	// Run performs the step and returns its process exit code.
	Run func() (succeeded Step_Success)
}

// Step_Invariants states the step label.
func Step_Invariants(step Step, namespace invariant.Namespace) {
	Step_Name_Invariants(step.Name, namespace)
}

// Bootstrap_Input carries the ordered steps and the logger their progress is
// announced to.
type Bootstrap_Input struct {
	// Steps run in slice order; the first to return non-zero stops the rest.
	Steps Steps
	// Logger records one line naming each step as it starts. The zero Logger is a
	// disabled no-op.
	Logger jlog.Logger
}

// Bootstrap_Input_Invariants states the ordered bootstrap plan.
func Bootstrap_Input_Invariants(input *Bootstrap_Input, namespace invariant.Namespace) {
	Steps_Invariants(input.Steps, namespace)
}

// Bootstrap runs the setup steps in their fixed order of operations, announcing
// each by name before it runs and returning the first non-zero status — skipping
// the rest — so a cheap early failure surfaces before later, heavier work.
func Bootstrap(input *Bootstrap_Input) (succeeded Step_Success) {
	defer func() { Step_Success_Invariants(succeeded, "Bootstrap.succeeded") }()
	Bootstrap_Input_Invariants(input, "Bootstrap.input")
	for _, step := range input.Steps {
		jlog.Logger_Info(input.Logger, "step", jlog.String("name", step.Name))
		if !step.Run() {
			return false
		}
	}
	return true
}

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
	for line := range strings.Lines(MACOS_DEFAULTS_SCRIPT) {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		commands = append(commands, Macos_Command{
			Name:      Step_Name(fields[0]),
			Arguments: Macos_Arguments(fields[1:]),
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
	// File_System is the loop the source tree is walked and read through, and the
	// destination read through for diffing.
	File_System File_System
	// Source_Directory is the absolute dotfiles tree, walked in full from its root.
	Source_Directory Source_Directory
	// Destination_Directory is the absolute home directory the relative source
	// paths are mirrored under to form each write's Destination_Path.
	Destination_Directory Destination_Directory
	// Is_Ignored classifies a batch of source paths, each relative to the source root,
	// returning the set that is gitignored. An ignored file is not synced and an ignored
	// directory is pruned, so the generated install tree under .local never reaches the
	// home directory. It takes one directory batch, not one path, because the gitignore
	// probe is a subprocess. Nil ignores nothing, the sync's behavior before the filter.
	Is_Ignored func(relative_paths []string) (ignored map[string]bool)
	// Logger records one line naming each directory as the walk reads it. A converged
	// scan writes nothing, so without this the step looks hung while it walks; narrating
	// each directory proves the scan is live. The zero Logger is a disabled no-op, so a
	// direct caller that wants no narration pays nothing.
	Logger jlog.Logger
}

// Plan_Input_Invariants states the mirror plan dependency set.
func Plan_Input_Invariants(input *Plan_Input, namespace invariant.Namespace) {
	File_System_Invariants(input.File_System, namespace)
	Source_Directory_Invariants(input.Source_Directory, namespace)
	Destination_Directory_Invariants(input.Destination_Directory, namespace)
}

// Plan returns the writes that would bring the home directory in line with the source
// dotfiles: every regular file under the source tree, each emitted only when the destination
// is missing or its contents differ. It walks through the loop's Read_Directory, classifies each
// directory with one Is_Ignored batch, and prunes an ignored directory so the install tree under
// .local is never descended into.
func Plan(input *Plan_Input) (writes Writes, err error) {
	defer func() { Writes_Invariants(writes, "Plan.writes") }()
	Plan_Input_Invariants(input, "Plan.input")
	writes = Writes{}
	directories := []Traversal_Directory{"."}
	directory_count := 1
	write_count := 0
	for len(directories) > 0 {
		directory := directories[0]
		directories = directories[1:]
		plan_err := plan_directory(
			input, directory,
			func(next Traversal_Directory) (err error) {
				if directory_count == TRAVERSAL_DIRECTORY_COUNT_MAX {
					return errors.New(
						"mirror directory count exceeds its limit")
				}
				directories = append(directories, next)
				directory_count++
				return nil
			},
			func(write File_Write) (err error) {
				if write_count == WRITE_COUNT_MAX {
					return errors.New("mirror write plan exceeds its limit")
				}
				write_plan_append(&writes, write)
				write_count++
				return nil
			},
		)
		if plan_err != nil {
			return Writes{}, plan_err
		}
	}
	return writes, nil
}

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
func plan_directory(
	input *Plan_Input, directory Traversal_Directory,
	enqueue func(directory Traversal_Directory) (err error),
	emit func(write File_Write) (err error),
) (err error) {
	Plan_Input_Invariants(input, "plan_directory.input")
	Traversal_Directory_Invariants(directory, "plan_directory.directory")
	plan_narrate(input.Logger, directory)
	entries, read_err := input.File_System.Read_Directory(
		filepath.Join(string(input.Source_Directory), string(directory)))
	if read_err != nil {
		return read_err
	}
	if len(entries) > PLAN_ENTRY_COUNT_MAX {
		return errors.New("mirror traversal entries exceed their limit")
	}
	paths := make([]string, len(entries))
	for index, child := range entries {
		paths[index] = filepath.Join(string(directory), child.Name)
		if len(paths[index]) > RELATIVE_FILE_PATH_BYTES_MAX {
			return errors.New("mirror relative path exceeds its limit")
		}
	}
	ignored := map[string]bool{}
	if input.Is_Ignored != nil {
		ignored = input.Is_Ignored(paths)
	}
	if len(ignored) > IGNORE_COUNT_MAX {
		return errors.New("mirror ignore result exceeds its limit")
	}
	for index, child := range entries {
		relative := Relative_File_Path(paths[index])
		if ignored[string(relative)] {
			continue
		}
		if child.Is_Directory {
			if enqueue(Traversal_Directory(relative)) != nil {
				return errors.New("mirror directory enqueue failed")
			}
			continue
		}
		write, planned, plan_err := plan_file(input, relative)
		if plan_err != nil {
			return plan_err
		}
		if planned {
			if emit(write) != nil {
				return errors.New("mirror write emit failed")
			}
		}
	}
	return nil
}

// Decides whether the source file at relative needs syncing. planned is false when the
// source is absent or the destination already holds identical bytes; otherwise it returns
// the write mirroring the relative path under the home directory.
func plan_file(input *Plan_Input, relative Relative_File_Path) (
	write File_Write, planned Plan_Decision, err error,
) {
	defer func() {
		File_Write_Invariants(write, "plan_file.write")
		Plan_Decision_Invariants(planned, "plan_file.planned")
	}()
	Plan_Input_Invariants(input, "plan_file.input")
	Relative_File_Path_Invariants(relative, "plan_file.relative")
	destination_path := Destination_Path(filepath.Join(
		string(input.Destination_Directory), string(relative)))
	if len(destination_path) > DESTINATION_PATH_BYTES_MAX {
		return File_Write{Destination_Path: destination_path}, false,
			errors.New("mirror destination path exceeds its limit")
	}
	source_contents, found, read_err := input.File_System.Read(
		filepath.Join(string(input.Source_Directory), string(relative)), DOTFILE_BYTES_MAX)
	if read_err != nil {
		return File_Write{Destination_Path: destination_path}, false, read_err
	}
	if !found {
		return File_Write{Destination_Path: destination_path}, false, nil
	}
	write = File_Write{
		Destination_Path: destination_path, Contents: Dotfile_Bytes(source_contents)}
	if destination_matches(
		input.File_System, Mirror_Path(destination_path), Dotfile_Bytes(source_contents)) {
		return write, false, nil
	}
	return write, true, nil
}

// Reports whether the home directory already holds exactly source_contents at path. A
// destination that is absent or unreadable counts as a mismatch, so the file is written.
func destination_matches(
	system File_System, path Mirror_Path, source_contents Dotfile_Bytes,
) (
	matches Plan_Decision,
) {
	defer func() { Plan_Decision_Invariants(matches, "destination_matches.matches") }()
	File_System_Invariants(system, "destination_matches.system")
	Mirror_Path_Invariants(path, "destination_matches.path")
	Dotfile_Bytes_Invariants(source_contents, "destination_matches.source_contents")
	destination_contents, found, err := system.Read(string(path), DOTFILE_BYTES_MAX)
	if err != nil {
		return false
	}
	if !found {
		return false
	}
	return Plan_Decision(bytes.Equal(source_contents, destination_contents))
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

// Shell_Invariants states that the required process operation is present.
func Shell_Invariants(shell Shell, namespace invariant.Namespace) {
	invariant.Always(shell.Spawn != nil, "A setup shell has a process operation.")
}

// Runs path with arguments and returns its captured standard output, trimmed — the version
// probes' one use. It passes no sink, so the loop captures the output for parsing rather than
// streaming it. Trimming strips the trailing newline `which` appends, so a resolved path is
// usable as the next probe's executable.
func run_pipe(shell Shell, path Probe_Path, arguments Probe_Arguments) (
	output Command_Output,
) {
	defer func() { Command_Output_Invariants(output, "run_pipe.output") }()
	Shell_Invariants(shell, "run_pipe.shell")
	Probe_Path_Invariants(path, "run_pipe.path")
	Probe_Arguments_Invariants(arguments, "run_pipe.arguments")
	result := shell.Spawn(sysio.Process_Request{
		Path: string(path), Arguments: []string(arguments)})
	return Command_Output(strings.TrimSpace(string(result.Output)))
}

// Runs the command named by the first argument and reports success, streaming the process's
// output live to the shell's sinks so a multi-minute build's progress reaches the user as it
// happens rather than in one burst at the end.
func run_spawn(shell Shell, arguments Spawn_Arguments) (ok Command_Success) {
	defer func() { Command_Success_Invariants(ok, "run_spawn.ok") }()
	Shell_Invariants(shell, "run_spawn.shell")
	Spawn_Arguments_Invariants(arguments, "run_spawn.arguments")
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
	Executable Managed_Executable
	// Version is the prefix the binary's --version output must start with; a prefix
	// so build or commit suffixes do not matter.
	Version Version_Prefix
}

// Installed_Input_Invariants states one managed executable version probe.
func Installed_Input_Invariants(input *Installed_Input, namespace invariant.Namespace) {
	Shell_Invariants(input.Shell, namespace)
	Managed_Executable_Invariants(input.Executable, namespace)
	Version_Prefix_Invariants(input.Version, namespace)
}

// Installed is the shared idempotency probe: it reports whether the binary at its
// managed path already reports the wanted version. An install step that is
// Installed does no work; one that is not reinstalls. direnv and Neovim verify the
// same rule their own way (a version subcommand, and a which-resolved path).
func Installed(input *Installed_Input) (yes File_Presence) {
	defer func() { File_Presence_Invariants(yes, "Installed.yes") }()
	Installed_Input_Invariants(input, "Installed.input")
	version := run_pipe(input.Shell, Probe_Path(input.Executable), Probe_Arguments{"--version"})
	return File_Presence(strings.HasPrefix(string(version), string(input.Version)))
}

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
	// Shell runs make; its sinks receive setup's narration and the streamed make output.
	Shell Shell
}

// Install_Neovim_Input_Invariants states the Neovim build dependency set.
func Install_Neovim_Input_Invariants(
	input *Install_Neovim_Input, namespace invariant.Namespace,
) {
	Repository_Directory_Invariants(input.Repository_Directory, namespace)
	Shell_Invariants(input.Shell, namespace)
}

// Install_Neovim builds the vendored Neovim and installs it under the local
// prefix, where it lands at local/bin/nvim — already on PATH — and finds its own
// runtime. Every step is idempotent: make rebuilds only what changed and install
// re-copies, so a repeated bootstrap converges without error.
func Install_Neovim(input *Install_Neovim_Input) (succeeded Step_Success) {
	defer func() { Step_Success_Invariants(succeeded, "Install_Neovim.succeeded") }()
	Install_Neovim_Input_Invariants(input, "Install_Neovim.input")
	// Building Neovim is the expensive step, so it is gated on the checkout's own
	// nvim not already being installed: a bootstrap that has it does no work.
	if neovim_already_installed(input.Shell, input.Repository_Directory) {
		jlog.Logger_Info(input.Shell.Logger, "neovim already installed",
			jlog.String("version", NEOVIM_VERSION))
		return true
	}
	for index, arguments := range neovim_make_invocations(input.Repository_Directory) {
		phase := "configuring neovim prefix"
		if index != 0 {
			phase = "installing neovim"
		}
		jlog.Logger_Info(input.Shell.Logger, phase)
		// The first argument is the executable; make's output streams to the sinks, so
		// a generic line is all setup adds.
		if !run_spawn(input.Shell, Spawn_Arguments(arguments)) {
			jlog.Logger_Error(input.Shell.Logger, "neovim build failed")
			return false
		}
	}
	return true
}

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
	install := append(slices.Clone(build), NEOVIM_INSTALL_GOAL)
	return Invocations{build, install}
}

// Reports whether version_output — the text `nvim --version` prints — names the
// wanted release NEOVIM_VERSION. Only the leading vMAJOR.MINOR.PATCH token is
// compared, so a build's -dev or +commit suffix does not matter. Empty output,
// from no nvim on PATH, is not the release, so the build runs.
func neovim_version_present(version_output Version_Output) (present File_Presence) {
	defer func() { File_Presence_Invariants(present, "neovim_version_present.present") }()
	Version_Output_Invariants(version_output, "neovim_version_present.version_output")
	first_line, _, _ := strings.Cut(string(version_output), "\n")
	for _, field := range strings.Fields(first_line) {
		if !strings.HasPrefix(field, "v") {
			continue
		}
		base, _, _ := strings.Cut(field, "-")
		return base == NEOVIM_VERSION
	}
	return false
}

// Reports whether the nvim on PATH is the checkout's own build at the wanted
// release. It resolves nvim first and rejects a path outside the repository, so a
// system package at the same version cannot stand in; only an in-repository
// binary then has its version checked.
func neovim_already_installed(
	shell Shell, repository_directory Repository_Directory,
) (installed File_Presence) {
	defer func() { File_Presence_Invariants(installed, "neovim_already_installed.installed") }()
	Shell_Invariants(shell, "neovim_already_installed.shell")
	Repository_Directory_Invariants(
		repository_directory, "neovim_already_installed.repository_directory")
	executable := run_pipe(shell, "which", Probe_Arguments{"nvim"})
	if !strings.HasPrefix(string(executable), string(repository_directory)+"/") {
		return false
	}
	return neovim_version_present(Version_Output(run_pipe(
		shell, Probe_Path(executable), Probe_Arguments{"--version"})))
}

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

// Install_Fonts_Input carries the injected dependencies Install_Fonts needs to
// place the vendored fonts where the operating system's font system looks. The
// filesystem effects are injected so this library tier never touches the OS;
// package main backs them with os/io rather than by shelling out.
type Install_Fonts_Input struct {
	// Font_Directory is the absolute destination, already resolved per operating
	// system. Empty means this OS has no known user font directory, so the step
	// does nothing.
	Font_Directory Font_Directory
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

// Install_Fonts_Input_Invariants states the font install destination.
func Install_Fonts_Input_Invariants(
	input *Install_Fonts_Input, namespace invariant.Namespace,
) {
	Font_Directory_Invariants(input.Font_Directory, namespace)
}

// Install_Fonts places the vendored Iosevka faces into the per-OS user font
// directory and, where the OS needs it, refreshes the font cache. It is
// idempotent: a destination already holding the fonts copies nothing and skips
// the refresh, so a repeat bootstrap does no work.
func Install_Fonts(input *Install_Fonts_Input) (succeeded Step_Success) {
	defer func() { Step_Success_Invariants(succeeded, "Install_Fonts.succeeded") }()
	Install_Fonts_Input_Invariants(input, "Install_Fonts.input")
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
			return false
		}
		jlog.Logger_Info(input.Logger, "copied font", jlog.String("file", file))
		copied = true
	}
	if !copied {
		return true
	}
	if input.Refresh == nil {
		return true
	}
	refresh_err := input.Refresh()
	if refresh_err != nil {
		jlog.Logger_Error(input.Logger, "font cache refresh failed", jlog.Err(refresh_err))
		return false
	}
	return true
}

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
	// Shell runs the version gate and the go build.
	Shell Shell
}

// Install_Direnv_Input_Invariants states the direnv build dependency set.
func Install_Direnv_Input_Invariants(
	input *Install_Direnv_Input, namespace invariant.Namespace,
) {
	Direnv_Directory_Invariants(input.Direnv_Directory, namespace)
	Binary_Directory_Invariants(input.Binary_Directory, namespace)
	Shell_Invariants(input.Shell, namespace)
}

// Install_Direnv builds the vendored direnv straight into the bin directory, where
// the shell hook and every .envrc can find it. It is the bootstrap's first step
// because everything downstream is driven by direnv. It is idempotent: when the
// built direnv already reports the wanted version, it does nothing.
func Install_Direnv(input *Install_Direnv_Input) (succeeded Step_Success) {
	defer func() { Step_Success_Invariants(succeeded, "Install_Direnv.succeeded") }()
	Install_Direnv_Input_Invariants(input, "Install_Direnv.input")
	if input.Direnv_Directory == "" {
		return true
	}
	if input.Binary_Directory == "" {
		return true
	}
	if direnv_built(input.Shell, input.Binary_Directory) {
		jlog.Logger_Info(input.Shell.Logger, "direnv already installed")
		return true
	}
	jlog.Logger_Info(input.Shell.Logger, "building direnv")
	destination := filepath.Join(string(input.Binary_Directory), "direnv")
	build := "cd " + string(input.Direnv_Directory) +
		" && CGO_ENABLED=0 go build -mod=vendor" +
		" -o " + destination + " ."
	if !run_spawn(input.Shell, Spawn_Arguments{"sh", "-c", build}) {
		jlog.Logger_Error(input.Shell.Logger, "direnv build failed")
		return false
	}
	return true
}

// Reports whether the direnv binary in the bin directory already reports the
// wanted version. Probing that exact path leaves a present build alone.
func direnv_built(shell Shell, binary_directory Binary_Directory) (built File_Presence) {
	defer func() { File_Presence_Invariants(built, "direnv_built.built") }()
	Shell_Invariants(shell, "direnv_built.shell")
	Binary_Directory_Invariants(binary_directory, "direnv_built.binary_directory")
	return Installed(&Installed_Input{
		Shell:      shell,
		Executable: Managed_Executable(filepath.Join(string(binary_directory), "direnv")),
		Version:    DIRENV_VERSION,
	})
}

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
	// Shell runs the `which` probes that gate the step, the rustup script, and the ln
	// calls. rustup reads CARGO_HOME and RUSTUP_HOME from the inherited environment, so
	// the toolchain lands under Cargo_Directory without plumbing.
	Shell Shell
}

// Install_Rust_Input_Invariants states the Rust install dependency set.
func Install_Rust_Input_Invariants(input *Install_Rust_Input, namespace invariant.Namespace) {
	Cargo_Directory_Invariants(input.Cargo_Directory, namespace)
	Rust_Link_Directory_Invariants(input.Link_Directory, namespace)
	Shell_Invariants(input.Shell, namespace)
}

// Install_Rust installs the Rust toolchain with rustup and symlinks cargo,
// rustup, and rustc into the PATH directory, so they are reachable without
// CARGO_HOME/bin on PATH. It is idempotent: when all three already resolve inside
// the link directory, it does nothing, so a repeat bootstrap does no work.
func Install_Rust(input *Install_Rust_Input) (succeeded Step_Success) {
	defer func() { Step_Success_Invariants(succeeded, "Install_Rust.succeeded") }()
	Install_Rust_Input_Invariants(input, "Install_Rust.input")
	if input.Cargo_Directory == "" {
		return true
	}
	if input.Link_Directory == "" {
		return true
	}
	if rust_installed(input.Shell, Required_Cargo_Directory(input.Cargo_Directory)) {
		jlog.Logger_Info(input.Shell.Logger, "rust already installed")
	} else {
		jlog.Logger_Info(input.Shell.Logger, "installing rust")
		if !run_spawn(input.Shell, Spawn_Arguments(rust_install_invocation())) {
			jlog.Logger_Error(input.Shell.Logger, "rust install failed")
			return false
		}
	}
	// Always (re)link, even when the install was skipped, so a stale or missing
	// symlink is repointed at the current CARGO_HOME without reinstalling.
	if !rust_link(&Rust_Link_Input{
		Shell:           input.Shell,
		Cargo_Directory: Required_Cargo_Directory(input.Cargo_Directory),
		Link_Directory:  input.Link_Directory,
	}) {
		jlog.Logger_Error(input.Shell.Logger, "rust link failed")
		return false
	}
	return true
}

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
func rust_installed(
	shell Shell, cargo_directory Required_Cargo_Directory,
) (installed File_Presence) {
	defer func() { File_Presence_Invariants(installed, "rust_installed.installed") }()
	Shell_Invariants(shell, "rust_installed.shell")
	Required_Cargo_Directory_Invariants(cargo_directory, "rust_installed.cargo_directory")
	return Installed(&Installed_Input{
		Shell: shell,
		Executable: Managed_Executable(filepath.Join(
			string(cargo_directory), "bin", "rustc")),
		Version: "rustc " + RUST_VERSION,
	})
}

// Carries the arguments for symlinking the rust toolchain onto PATH.
type Rust_Link_Input struct {
	// Shell is the command runner the symlinks are created through.
	Shell Shell
	// Cargo_Directory is the CARGO_HOME whose bin holds the toolchain to link from.
	Cargo_Directory Required_Cargo_Directory
	// Link_Directory is the PATH entry the toolchain is symlinked into.
	Link_Directory Rust_Link_Directory
}

// Rust_Link_Input_Invariants states the Rust link dependency set.
func Rust_Link_Input_Invariants(input *Rust_Link_Input, namespace invariant.Namespace) {
	Shell_Invariants(input.Shell, namespace)
	Required_Cargo_Directory_Invariants(input.Cargo_Directory, namespace)
	Rust_Link_Directory_Invariants(input.Link_Directory, namespace)
}

// Symlinks cargo, rustup, and rustc from CARGO_HOME/bin into the link directory
// with ln -sf, so the managed toolchain is reachable from the one PATH entry and
// any stale link there is overwritten. Reports whether every link succeeded.
func rust_link(input *Rust_Link_Input) (linked Command_Success) {
	defer func() { Command_Success_Invariants(linked, "rust_link.linked") }()
	Rust_Link_Input_Invariants(input, "rust_link.input")
	for _, tool := range []string{"cargo", "rustup", "rustc"} {
		source := filepath.Join(string(input.Cargo_Directory), "bin", tool)
		target := filepath.Join(string(input.Link_Directory), tool)
		if !run_spawn(input.Shell, Spawn_Arguments{"ln", "-sf", source, target}) {
			return false
		}
	}
	return true
}

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
	// Shell runs the version gate and the go build.
	Shell Shell
}

// Install_Fzf_Input_Invariants states the fzf build dependency set.
func Install_Fzf_Input_Invariants(input *Install_Fzf_Input, namespace invariant.Namespace) {
	Fzf_Directory_Invariants(input.Fzf_Directory, namespace)
	Binary_Directory_Invariants(input.Binary_Directory, namespace)
	Shell_Invariants(input.Shell, namespace)
}

// Install_Fzf builds fzf from the vendored source straight into the bin directory.
// It is idempotent: when the built fzf already reports the wanted version, it does
// nothing, so a repeat bootstrap does no work. fzf is a Go binary, so the build
// output is the install — no separate copy or symlink.
func Install_Fzf(input *Install_Fzf_Input) (succeeded Step_Success) {
	defer func() { Step_Success_Invariants(succeeded, "Install_Fzf.succeeded") }()
	Install_Fzf_Input_Invariants(input, "Install_Fzf.input")
	if input.Fzf_Directory == "" {
		return true
	}
	if input.Binary_Directory == "" {
		return true
	}
	if fzf_built(input.Shell, input.Binary_Directory) {
		jlog.Logger_Info(input.Shell.Logger, "fzf already installed")
		return true
	}
	jlog.Logger_Info(input.Shell.Logger, "building fzf")
	destination := filepath.Join(string(input.Binary_Directory), "fzf")
	build := "cd " + string(input.Fzf_Directory) +
		" && go build -mod=vendor" +
		" -ldflags '-s -w -X main.version=" + FZF_VERSION + " -X main.revision='" +
		" -o " + destination + " ."
	if !run_spawn(input.Shell, Spawn_Arguments{"sh", "-c", build}) {
		jlog.Logger_Error(input.Shell.Logger, "fzf build failed")
		return false
	}
	return true
}

// Reports whether the fzf binary in the bin directory already reports the wanted
// version. Probing that exact path leaves a present build alone.
func fzf_built(shell Shell, binary_directory Binary_Directory) (built File_Presence) {
	defer func() { File_Presence_Invariants(built, "fzf_built.built") }()
	Shell_Invariants(shell, "fzf_built.shell")
	Binary_Directory_Invariants(binary_directory, "fzf_built.binary_directory")
	return Installed(&Installed_Input{
		Shell:      shell,
		Executable: Managed_Executable(filepath.Join(string(binary_directory), "fzf")),
		Version:    FZF_VERSION,
	})
}

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
	Binary_Name Step_Name
	// Shell runs the PATH-presence gate and the go build.
	Shell Shell
}

// Install_Command_Input_Invariants states one repository command build.
func Install_Command_Input_Invariants(
	input *Install_Command_Input, namespace invariant.Namespace,
) {
	Command_Directory_Invariants(input.Package_Directory, namespace)
	Binary_Directory_Invariants(input.Binary_Directory, namespace)
	Step_Name_Invariants(input.Binary_Name, namespace)
	Shell_Invariants(input.Shell, namespace)
}

// Install_Command builds a command from this repository straight into the bin
// directory. Its one idempotency check is whether the command already resolves on
// PATH: these are this repo's own programs with no pinned version to match, rebuilt
// freely, so a name already on PATH is left alone and an absent one is built. Like
// fzf and direnv, it is a Go build, so the build output is the install — no separate
// copy or symlink.
func Install_Command(input *Install_Command_Input) (succeeded Step_Success) {
	defer func() { Step_Success_Invariants(succeeded, "Install_Command.succeeded") }()
	Install_Command_Input_Invariants(input, "Install_Command.input")
	if input.Package_Directory == "" {
		return true
	}
	if input.Binary_Directory == "" {
		return true
	}
	if input.Binary_Name == "" {
		return true
	}
	if command_on_path(input.Shell, input.Binary_Name) {
		jlog.Logger_Info(input.Shell.Logger, "command already installed",
			jlog.String("command", string(input.Binary_Name)))
		return true
	}
	jlog.Logger_Info(input.Shell.Logger, "building command",
		jlog.String("command", string(input.Binary_Name)))
	destination := filepath.Join(string(input.Binary_Directory), string(input.Binary_Name))
	build := "cd " + string(input.Package_Directory) +
		" && go build -o " + destination + " ."
	if !run_spawn(input.Shell, Spawn_Arguments{"sh", "-c", build}) {
		jlog.Logger_Error(input.Shell.Logger, "command build failed",
			jlog.String("command", string(input.Binary_Name)))
		return false
	}
	return true
}

// Reports whether a command of the given name already resolves on PATH — the only
// idempotency gate for this repo's own commands. `which` prints the resolved path
// on stdout and nothing when the name is unknown, so a non-empty result means the
// command is present and the build is skipped.
func command_on_path(shell Shell, name Step_Name) (present File_Presence) {
	defer func() { File_Presence_Invariants(present, "command_on_path.present") }()
	Shell_Invariants(shell, "command_on_path.shell")
	Step_Name_Invariants(name, "command_on_path.name")
	return run_pipe(shell, "which", Probe_Arguments{string(name)}) != ""
}

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
	// Shell runs the `jj --version` gate and the cargo build.
	Shell Shell
}

// Install_Jj_Input_Invariants states the jj build dependency set.
func Install_Jj_Input_Invariants(input *Install_Jj_Input, namespace invariant.Namespace) {
	Jj_Directory_Invariants(input.Jj_Directory, namespace)
	Binary_Directory_Invariants(input.Binary_Directory, namespace)
	Shell_Invariants(input.Shell, namespace)
}

// Install_Jj builds jj from the vendored workspace with cargo, installing the binary
// straight into the binary directory. It is idempotent: when the built jj already
// reports the wanted version the build is skipped, so a repeat bootstrap does no
// heavy work.
func Install_Jj(input *Install_Jj_Input) (succeeded Step_Success) {
	defer func() { Step_Success_Invariants(succeeded, "Install_Jj.succeeded") }()
	Install_Jj_Input_Invariants(input, "Install_Jj.input")
	if input.Jj_Directory == "" {
		return true
	}
	if input.Binary_Directory == "" {
		return true
	}
	if jj_built(input.Shell, input.Binary_Directory) {
		jlog.Logger_Info(input.Shell.Logger, "jj already built")
		return true
	}
	jlog.Logger_Info(input.Shell.Logger, "building jj")
	if !run_spawn(input.Shell, Spawn_Arguments(jj_install_invocation(input))) {
		jlog.Logger_Error(input.Shell.Logger, "jj build failed")
		return false
	}
	return true
}

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
func jj_built(shell Shell, binary_directory Binary_Directory) (built File_Presence) {
	defer func() { File_Presence_Invariants(built, "jj_built.built") }()
	Shell_Invariants(shell, "jj_built.shell")
	Binary_Directory_Invariants(binary_directory, "jj_built.binary_directory")
	return Installed(&Installed_Input{
		Shell:      shell,
		Executable: Managed_Executable(filepath.Join(string(binary_directory), "jj")),
		Version:    JJ_VERSION,
	})
}

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
	// Shell runs the `rg --version` gate and the cargo build.
	Shell Shell
}

// Install_Ripgrep_Input_Invariants states the ripgrep build dependency set.
func Install_Ripgrep_Input_Invariants(
	input *Install_Ripgrep_Input, namespace invariant.Namespace,
) {
	Ripgrep_Directory_Invariants(input.Ripgrep_Directory, namespace)
	Binary_Directory_Invariants(input.Binary_Directory, namespace)
	Shell_Invariants(input.Shell, namespace)
}

// Install_Ripgrep builds ripgrep from the vendored crate with cargo, installing the
// rg binary straight into the binary directory. It is idempotent: when the built rg
// already reports the wanted version the build is skipped, so a repeat bootstrap
// does no heavy work.
func Install_Ripgrep(input *Install_Ripgrep_Input) (succeeded Step_Success) {
	defer func() { Step_Success_Invariants(succeeded, "Install_Ripgrep.succeeded") }()
	Install_Ripgrep_Input_Invariants(input, "Install_Ripgrep.input")
	if input.Ripgrep_Directory == "" {
		return true
	}
	if input.Binary_Directory == "" {
		return true
	}
	if ripgrep_built(input.Shell, input.Binary_Directory) {
		jlog.Logger_Info(input.Shell.Logger, "ripgrep already built")
		return true
	}
	jlog.Logger_Info(input.Shell.Logger, "building ripgrep")
	invocation := ripgrep_install_invocation(input)
	if !run_spawn(input.Shell, Spawn_Arguments(invocation)) {
		jlog.Logger_Error(input.Shell.Logger, "ripgrep build failed")
		return false
	}
	return true
}

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
func ripgrep_built(shell Shell, binary_directory Binary_Directory) (built File_Presence) {
	defer func() { File_Presence_Invariants(built, "ripgrep_built.built") }()
	Shell_Invariants(shell, "ripgrep_built.shell")
	Binary_Directory_Invariants(binary_directory, "ripgrep_built.binary_directory")
	return Installed(&Installed_Input{
		Shell:      shell,
		Executable: Managed_Executable(filepath.Join(string(binary_directory), "rg")),
		Version:    RIPGREP_VERSION,
	})
}

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
	// Shell runs the `fd --version` gate and the cargo build.
	Shell Shell
}

// Install_Fdcli_Input_Invariants states the fd build dependency set.
func Install_Fdcli_Input_Invariants(
	input *Install_Fdcli_Input, namespace invariant.Namespace,
) {
	Fdcli_Directory_Invariants(input.Fdcli_Directory, namespace)
	Binary_Directory_Invariants(input.Binary_Directory, namespace)
	Shell_Invariants(input.Shell, namespace)
}

// Install_Fdcli builds fd from the vendored crate with cargo, installing the binary
// straight into the binary directory. It is idempotent: when the built fd already
// reports the wanted version the build is skipped, so a repeat bootstrap does no
// heavy work.
func Install_Fdcli(input *Install_Fdcli_Input) (succeeded Step_Success) {
	defer func() { Step_Success_Invariants(succeeded, "Install_Fdcli.succeeded") }()
	Install_Fdcli_Input_Invariants(input, "Install_Fdcli.input")
	if input.Fdcli_Directory == "" {
		return true
	}
	if input.Binary_Directory == "" {
		return true
	}
	if fdcli_built(input.Shell, input.Binary_Directory) {
		jlog.Logger_Info(input.Shell.Logger, "fd already built")
		return true
	}
	jlog.Logger_Info(input.Shell.Logger, "building fd")
	invocation := fdcli_install_invocation(input)
	if !run_spawn(input.Shell, Spawn_Arguments(invocation)) {
		jlog.Logger_Error(input.Shell.Logger, "fd build failed")
		return false
	}
	return true
}

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
func fdcli_built(shell Shell, binary_directory Binary_Directory) (built File_Presence) {
	defer func() { File_Presence_Invariants(built, "fdcli_built.built") }()
	Shell_Invariants(shell, "fdcli_built.shell")
	Binary_Directory_Invariants(binary_directory, "fdcli_built.binary_directory")
	return Installed(&Installed_Input{
		Shell:      shell,
		Executable: Managed_Executable(filepath.Join(string(binary_directory), "fd")),
		Version:    FDCLI_VERSION,
	})
}

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
	// Shell runs the version gate, the download-and-install script, and the ln call.
	Shell Shell
}

// Install_Ghostty_Input_Invariants states the Ghostty install dependency set.
func Install_Ghostty_Input_Invariants(
	input *Install_Ghostty_Input, namespace invariant.Namespace,
) {
	Applications_Directory_Invariants(input.Applications_Directory, namespace)
	Ghostty_Link_Directory_Invariants(input.Link_Directory, namespace)
	Shell_Invariants(input.Shell, namespace)
}

// Install_Ghostty installs Ghostty from its pinned DMG into the applications
// directory and symlinks the app's CLI into the PATH directory. It is idempotent:
// when the installed app already reports the wanted version, the download is skipped
// and only the symlink is refreshed, so a repeat bootstrap does no network work.
func Install_Ghostty(input *Install_Ghostty_Input) (succeeded Step_Success) {
	defer func() { Step_Success_Invariants(succeeded, "Install_Ghostty.succeeded") }()
	Install_Ghostty_Input_Invariants(input, "Install_Ghostty.input")
	if ghostty_installed(input.Shell, input.Applications_Directory) {
		jlog.Logger_Info(input.Shell.Logger, "ghostty already installed")
	} else {
		jlog.Logger_Info(input.Shell.Logger, "installing ghostty")
		invocation := ghostty_install_invocation(input.Applications_Directory)
		if !run_spawn(input.Shell, Spawn_Arguments(invocation)) {
			jlog.Logger_Error(input.Shell.Logger, "ghostty install failed")
			return false
		}
	}
	// Always (re)link, even when the install was skipped, so a missing symlink is
	// restored without re-downloading. ln -sf is idempotent.
	source := filepath.Join(
		string(input.Applications_Directory), GHOSTTY_APPLICATION_BINARY_SUBPATH)
	target := filepath.Join(string(input.Link_Directory), "ghostty")
	if !run_spawn(input.Shell, Spawn_Arguments{"ln", "-sf", source, target}) {
		jlog.Logger_Error(input.Shell.Logger, "ghostty link failed")
		return false
	}
	return true
}

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
	// File_System supplies the dotfiles read and write operations.
	File_System File_System
	// Shell supplies the process operation and output streams.
	Shell Shell
}

// Bootstrap_Steps_Input_Invariants states the complete bootstrap dependency set.
func Bootstrap_Steps_Input_Invariants(
	input *Bootstrap_Steps_Input, namespace invariant.Namespace,
) {
	Home_Directory_Invariants(input.Home_Directory, namespace)
	Operating_System_Invariants(input.Operating_System, namespace)
	Cargo_Directory_Invariants(input.Cargo_Directory, namespace)
	Data_Directory_Invariants(input.Data_Directory, namespace)
	File_System_Invariants(input.File_System, namespace)
	Shell_Invariants(input.Shell, namespace)
}

// Bootstrap_Steps returns the complete setup policy in execution order.
func Bootstrap_Steps(input *Bootstrap_Steps_Input) (steps Steps) {
	defer func() { Steps_Invariants(steps, "Bootstrap_Steps.steps") }()
	Bootstrap_Steps_Input_Invariants(input, "Bootstrap_Steps.input")
	return Steps{
		{Name: "direnv", Run: direnv_step(input)},
		{Name: "dotfiles", Run: dotfiles_step(input)},
		{Name: "fonts", Run: fonts_step(input)},
		{Name: "neovim", Run: neovim_step(input)},
		{Name: "fzf", Run: fzf_step(input)},
		{Name: "maddox", Run: command_step(&Command_Step_Input{
			Bootstrap: input, Package_Directory: "maddox", Binary_Name: "maddox",
		})},
		{Name: "m2p", Run: command_step(&Command_Step_Input{
			Bootstrap: input, Package_Directory: "markdown_to_pdf", Binary_Name: "m2p",
		})},
		{Name: "sloc", Run: command_step(&Command_Step_Input{
			Bootstrap: input, Package_Directory: "sloc", Binary_Name: "sloc",
		})},
		{Name: "timeout", Run: command_step(&Command_Step_Input{
			Bootstrap: input, Package_Directory: "timeout", Binary_Name: "timeout",
		})},
		{Name: "rust", Run: rust_step(input)},
		{Name: "jj", Run: jj_step(input)},
		{Name: "ripgrep", Run: ripgrep_step(input)},
		{Name: "fd", Run: fdcli_step(input)},
		{Name: "ghostty", Run: ghostty_step(input)},
	}
}

// Returns the step that builds the vendored direnv binary.
func direnv_step(input *Bootstrap_Steps_Input) (run func() (succeeded Step_Success)) {
	Bootstrap_Steps_Input_Invariants(input, "direnv_step.input")
	repository := filepath.Join(string(input.Home_Directory), REPOSITORY_SUBPATH)
	return func() (succeeded Step_Success) {
		return Install_Direnv(&Install_Direnv_Input{
			Direnv_Directory: Direnv_Directory(
				filepath.Join(repository, "third_party", "direnv")),
			Binary_Directory: Binary_Directory(
				filepath.Join(repository, "home", ".local", "bin")),
			Shell: input.Shell,
		})
	}
}

// Returns the step that synchronizes dotfiles and applies macOS defaults.
func dotfiles_step(input *Bootstrap_Steps_Input) (run func() (succeeded Step_Success)) {
	Bootstrap_Steps_Input_Invariants(input, "dotfiles_step.input")
	dotfiles_directory := filepath.Join(string(input.Home_Directory), DOTFILES_SUBPATH)
	return func() (succeeded Step_Success) {
		return Mirror(&Mirror_Input{
			File_System:           input.File_System,
			Source_Directory:      Source_Directory(dotfiles_directory),
			Destination_Directory: Destination_Directory(input.Home_Directory),
			Operating_System:      input.Operating_System,
			Run_Command:           run_command(input.Shell),
			Is_Ignored: git_ignores(
				input.Shell, Directory(dotfiles_directory)),
			Logger: input.Shell.Logger,
		})
	}
}

// Returns the step that copies each absent vendored font.
func fonts_step(input *Bootstrap_Steps_Input) (run func() (succeeded Step_Success)) {
	Bootstrap_Steps_Input_Invariants(input, "fonts_step.input")
	font_directory, refresh_cache, supported := font_destination(&Font_Destination_Input{
		Home_Directory: input.Home_Directory, Operating_System: input.Operating_System,
		Data_Directory: input.Data_Directory,
	})
	font_source := filepath.Join(string(input.Home_Directory), IOSEVKA_SUBPATH)
	var refresh func() (err error)
	if refresh_cache {
		refresh = func() (err error) {
			return run_command(input.Shell)(
				"fc-cache", []string{"-f", string(font_directory)})
		}
	}
	return func() (succeeded Step_Success) {
		if !supported {
			return true
		}
		return Install_Fonts(&Install_Fonts_Input{
			Font_Directory: font_directory,
			Font_Present: func(file string) (present bool) {
				path := filepath.Join(string(font_directory), file)
				return bool(file_present(input.File_System, Font_Path(path)))
			},
			Copy_Font: func(file string) (err error) {
				return copy_file(input.File_System, &File_Copy_Input{
					Source: Font_Source_Path(filepath.Join(font_source, file)),
					Destination: Font_Destination_Path(
						filepath.Join(string(font_directory), file)),
				})
			},
			Refresh: refresh, Logger: input.Shell.Logger,
		})
	}
}

// Returns the step that builds the vendored Neovim checkout.
func neovim_step(input *Bootstrap_Steps_Input) (run func() (succeeded Step_Success)) {
	Bootstrap_Steps_Input_Invariants(input, "neovim_step.input")
	return func() (succeeded Step_Success) {
		return Install_Neovim(&Install_Neovim_Input{
			Repository_Directory: Repository_Directory(filepath.Join(
				string(input.Home_Directory), REPOSITORY_SUBPATH)),
			Shell: input.Shell,
		})
	}
}

// Returns the step that builds the vendored fzf checkout.
func fzf_step(input *Bootstrap_Steps_Input) (run func() (succeeded Step_Success)) {
	Bootstrap_Steps_Input_Invariants(input, "fzf_step.input")
	repository := filepath.Join(string(input.Home_Directory), REPOSITORY_SUBPATH)
	return func() (succeeded Step_Success) {
		return Install_Fzf(&Install_Fzf_Input{
			Fzf_Directory: Fzf_Directory(
				filepath.Join(repository, "third_party", "fzf")),
			Binary_Directory: Binary_Directory(
				filepath.Join(repository, "home", ".local", "bin")),
			Shell: input.Shell,
		})
	}
}

// Command_Step_Input identifies one repository command build step.
type Command_Step_Input struct {
	// Bootstrap supplies the home directory and process operation.
	Bootstrap *Bootstrap_Steps_Input
	// Package_Directory is the package path below the repository.
	Package_Directory Package_Directory
	// Binary_Name is the installed command name.
	Binary_Name Step_Name
}

// Command_Step_Input_Invariants states one repository command step.
func Command_Step_Input_Invariants(
	input *Command_Step_Input, namespace invariant.Namespace,
) {
	Bootstrap_Steps_Input_Invariants(input.Bootstrap, namespace)
	Package_Directory_Invariants(input.Package_Directory, namespace)
	Step_Name_Invariants(input.Binary_Name, namespace)
}

// Returns a step that builds one command from this repository.
func command_step(input *Command_Step_Input) (run func() (succeeded Step_Success)) {
	Command_Step_Input_Invariants(input, "command_step.input")
	repository := filepath.Join(string(input.Bootstrap.Home_Directory), REPOSITORY_SUBPATH)
	return func() (succeeded Step_Success) {
		return Install_Command(&Install_Command_Input{
			Package_Directory: Command_Directory(
				filepath.Join(repository, string(input.Package_Directory))),
			Binary_Directory: Binary_Directory(
				filepath.Join(repository, "home", ".local", "bin")),
			Binary_Name: input.Binary_Name,
			Shell:       input.Bootstrap.Shell,
		})
	}
}

// Returns the step that installs the Rust toolchain.
func rust_step(input *Bootstrap_Steps_Input) (run func() (succeeded Step_Success)) {
	Bootstrap_Steps_Input_Invariants(input, "rust_step.input")
	return func() (succeeded Step_Success) {
		return Install_Rust(&Install_Rust_Input{
			Cargo_Directory: input.Cargo_Directory,
			Link_Directory: Rust_Link_Directory(filepath.Join(
				string(input.Home_Directory), REPOSITORY_SUBPATH, ".local", "bin")),
			Shell: input.Shell,
		})
	}
}

// Returns the step that builds the vendored jj checkout.
func jj_step(input *Bootstrap_Steps_Input) (run func() (succeeded Step_Success)) {
	Bootstrap_Steps_Input_Invariants(input, "jj_step.input")
	repository := filepath.Join(string(input.Home_Directory), REPOSITORY_SUBPATH)
	return func() (succeeded Step_Success) {
		return Install_Jj(&Install_Jj_Input{
			Jj_Directory: Jj_Directory(
				filepath.Join(repository, "third_party", "jj")),
			Binary_Directory: Binary_Directory(
				filepath.Join(repository, "home", ".local", "bin")),
			Shell: input.Shell,
		})
	}
}

// Returns the step that builds the vendored ripgrep checkout.
func ripgrep_step(input *Bootstrap_Steps_Input) (run func() (succeeded Step_Success)) {
	Bootstrap_Steps_Input_Invariants(input, "ripgrep_step.input")
	repository := filepath.Join(string(input.Home_Directory), REPOSITORY_SUBPATH)
	return func() (succeeded Step_Success) {
		return Install_Ripgrep(&Install_Ripgrep_Input{
			Ripgrep_Directory: Ripgrep_Directory(
				filepath.Join(repository, "third_party", "ripgrep")),
			Binary_Directory: Binary_Directory(
				filepath.Join(repository, "home", ".local", "bin")),
			Shell: input.Shell,
		})
	}
}

// Returns the step that builds the vendored fd checkout.
func fdcli_step(input *Bootstrap_Steps_Input) (run func() (succeeded Step_Success)) {
	Bootstrap_Steps_Input_Invariants(input, "fdcli_step.input")
	repository := filepath.Join(string(input.Home_Directory), REPOSITORY_SUBPATH)
	return func() (succeeded Step_Success) {
		return Install_Fdcli(&Install_Fdcli_Input{
			Fdcli_Directory: Fdcli_Directory(
				filepath.Join(repository, "third_party", "fd")),
			Binary_Directory: Binary_Directory(
				filepath.Join(repository, "home", ".local", "bin")),
			Shell: input.Shell,
		})
	}
}

// Returns the step that installs the signed Ghostty application.
func ghostty_step(input *Bootstrap_Steps_Input) (run func() (succeeded Step_Success)) {
	Bootstrap_Steps_Input_Invariants(input, "ghostty_step.input")
	if input.Operating_System != "darwin" {
		return func() (succeeded Step_Success) { return true }
	}
	repository := filepath.Join(string(input.Home_Directory), REPOSITORY_SUBPATH)
	return func() (succeeded Step_Success) {
		return Install_Ghostty(&Install_Ghostty_Input{
			Applications_Directory: Applications_Directory("/Applications"),
			Link_Directory: Ghostty_Link_Directory(
				filepath.Join(repository, "home", ".local", "bin")),
			Shell: input.Shell,
		})
	}
}

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

// Returns an external-program runner that streams output through shell.
func run_command(shell Shell) (run func(name string, arguments []string) (err error)) {
	Shell_Invariants(shell, "run_command.shell")
	return func(name string, arguments []string) (err error) {
		result := shell.Spawn(sysio.Process_Request{
			Path: name, Arguments: arguments,
			Stdout: shell.Stdout, Stderr: shell.Stderr,
		})
		if result.Exit != 0 {
			return fmt.Errorf("%s exited with status %d", name, result.Exit)
		}
		return nil
	}
}

// Returns one batched gitignore classifier for directory.
func git_ignores(
	shell Shell, directory Directory,
) (is_ignored func(relative_paths []string) (ignored map[string]bool)) {
	Shell_Invariants(shell, "git_ignores.shell")
	Directory_Invariants(directory, "git_ignores.directory")
	return func(relative_paths []string) (ignored map[string]bool) {
		ignored = map[string]bool{}
		if len(relative_paths) == 0 {
			return ignored
		}
		targets := make([]string, len(relative_paths))
		for index, relative := range relative_paths {
			targets[index] = filepath.Join(string(directory), relative)
		}
		result := shell.Spawn(sysio.Process_Request{
			Path:      "git",
			Arguments: []string{"-C", string(directory), "check-ignore", "--stdin"},
			Input:     []byte(strings.Join(targets, "\n") + "\n"),
		})
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
func ghostty_installed(
	shell Shell, applications_directory Applications_Directory,
) (installed File_Presence) {
	defer func() { File_Presence_Invariants(installed, "ghostty_installed.installed") }()
	Shell_Invariants(shell, "ghostty_installed.shell")
	Applications_Directory_Invariants(
		applications_directory, "ghostty_installed.applications_directory")
	binary := Managed_Executable(filepath.Join(
		string(applications_directory), GHOSTTY_APPLICATION_BINARY_SUBPATH))
	version_present := Installed(&Installed_Input{
		Shell:      shell,
		Executable: binary,
		Version:    GHOSTTY_VERSION,
	})
	if !version_present {
		return false
	}
	// A matching version alone is not enough: an app whose binary was swapped or
	// re-signed by another developer still reports 1.3.1, so the install counts only
	// if codesign verifies the bundle against Ghostty's signing identity. codesign is
	// offline and deterministic, so a failure means a tampered or foreign bundle.
	application := Application_Path(filepath.Join(
		string(applications_directory), "Ghostty.app"))
	return File_Presence(run_spawn(
		shell, Spawn_Arguments(ghostty_codesign_invocation(application))))
}

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
const STEP_NAME_BYTES_MIN = 3

// STEP_NAME_BYTES_MAX is the longest bootstrap label.
const STEP_NAME_BYTES_MAX = 7

// MACOS_ARGUMENT_COUNT_MIN is the Finder restart argument.
const MACOS_ARGUMENT_COUNT_MIN = 1

// MACOS_ARGUMENT_COUNT_MAX is the longest defaults command argument list.
const MACOS_ARGUMENT_COUNT_MAX = 6

// PROBE_ARGUMENT_COUNT is the single argument for a version or path probe.
const PROBE_ARGUMENT_COUNT = 1

// INSTALL_ARGUMENT_COUNT is the shell, flag, and script install tuple.
const INSTALL_ARGUMENT_COUNT = 3

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

// Codesign_Arguments is the complete Ghostty signature check.
type Codesign_Arguments []string

// Codesign_Arguments_Invariants requires every signature restriction.
func Codesign_Arguments_Invariants(arguments Codesign_Arguments, namespace invariant.Namespace) {
	invariant.Always(
		len(arguments) == CODESIGN_ARGUMENT_COUNT,
		"The signature check has every required restriction.")
}

// Steps is one ordered bootstrap plan.
type Steps []Step

// Steps_Invariants bounds one bootstrap plan.
func Steps_Invariants(steps Steps, namespace invariant.Namespace) {
	invariant.Always(
		len(steps) == BOOTSTRAP_STEP_COUNT, "Setup has every bootstrap step.")
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

// Write_plan_append appends one write while Plan still owns the list.
func write_plan_append(writes *Writes, write File_Write) {
	Writes_Invariants(*writes, "write_plan_append.writes")
	File_Write_Invariants(write, "write_plan_append.write")
	writes.List.PushBack(write)
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
