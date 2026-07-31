// Package sloc counts the code, comment, and blank lines of source files. A
// generic per-line scanner, configured by a Language, classifies each physical
// line; a walker counts a tree in parallel; a renderer prints an aligned table.
package sloc

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"local/james-orcales/shared/cli"
	"local/james-orcales/shared/encoding/flatjson"
	invariant "local/james-orcales/shared/invariant/default"
)

// EXIT_SUCCESS is the status of a run that counted and rendered without fault.
const EXIT_SUCCESS Exit_Code = 0

// EXIT_FAILURE is the catch-all nonzero for a run that started but could not finish.
const EXIT_FAILURE Exit_Code = 1

// EXIT_USAGE is 2 by shell convention, keeping a command-line misuse distinct from a
// run failure.
const EXIT_USAGE Exit_Code = 2

// Main parses the command line, counts every path, and renders the table, returning a
// process exit code. The input is taken by value so Main_Input need not sit directly
// above it, leaving Main the first function declared in the file.
func Main(input Main_Input) (status_code Exit_Code) {
	defer func() { Exit_Code_Invariants(status_code, "Main.status_code") }()
	Main_Input_Invariants(input, "Main.input")
	program := main_program()
	if cli.Handle_Completion(program, input.Arguments, input.Output) {
		return EXIT_SUCCESS
	}
	command, parse_err := cli.Program_Parse(&program, input.Arguments)
	if errors.Is(parse_err, cli.Help_Requested) {
		cli.Print_Requested_Help(input.Output, program, command)
		return EXIT_SUCCESS
	}
	if parse_err != nil {
		fmt.Fprintf(input.Error_Output, "sloc: %v\n\n", parse_err)
		cli.Print_Help(input.Error_Output, program)
		return EXIT_USAGE
	}
	scope := Main_Scope{
		Paths:     Paths(cli.Get_Option(command.Arguments, "path").Value.([]string)),
		No_Ignore: No_Ignore(cli.Get_Option(command.Flags, "no-ignore").Value.(bool)),
		Include_Hidden: Include_Hidden(
			cli.Get_Option(command.Flags, "hidden").Value.(bool)),
	}
	// No path given counts the current directory, the obvious default for a tool run
	// inside a project.
	if len(scope.Paths) == 0 {
		scope.Paths = Paths{"."}
	}
	report, count_err := main_input_collect(&input, scope)
	if count_err != nil {
		fmt.Fprintf(input.Error_Output, "sloc: %v\n", count_err)
		return EXIT_FAILURE
	}
	if cli.Get_Option(command.Flags, "json").Value.(bool) {
		json_err := Render_Json(input.Output, report.Files)
		if json_err != nil {
			fmt.Fprintf(input.Error_Output, "sloc: %v\n", json_err)
			return EXIT_FAILURE
		}
		return EXIT_SUCCESS
	}
	Render(input.Output, Render_Input{
		Report:     report,
		Show_Files: File_Display(cli.Get_Option(command.Flags, "files").Value.(bool)),
	})
	return EXIT_SUCCESS
}

// Exit_Code is a process exit status. It is a distinct type so the status a run
// returns carries the declared range rather than the whole integer domain.
type Exit_Code uint8

// Exit_Code_Invariants holds a status to the three declared codes, and witnesses each
// code. The codes are an enumeration, so each member axis names the code it claims.
func Exit_Code_Invariants(code Exit_Code, namespace invariant.Namespace) {
	invariant.Tree(code, namespace).
		Enum_3_Uint8(
			uint8(code), uint8(EXIT_SUCCESS), uint8(EXIT_FAILURE), uint8(EXIT_USAGE)).
		Ensure()
}

// Declares the sloc command line: a commandless program whose positional arguments
// are the paths to scan, plus the override flags.
func main_program() (program cli.Program) {
	return cli.New_Single(cli.New_Single_Input{
		Label:       "sloc",
		Description: "count lines of code, comments, and blanks",
		Arguments: []cli.Option{
			cli.New_Variadic[string](cli.New_Variadic_Input{
				Label:       "path",
				Description: "files or directories to scan (default: .)",
			}),
		},
		Flags: []cli.Option{
			cli.New_Flag(cli.New_Flag_Input[bool]{
				Label:       "files",
				Value:       false,
				Description: "list every counted file under its language",
			}),
			cli.New_Flag(cli.New_Flag_Input[bool]{
				Label:       "no-ignore",
				Value:       false,
				Description: "count gitignored files too",
			}),
			cli.New_Flag(cli.New_Flag_Input[bool]{
				Label:       "hidden",
				Value:       false,
				Description: "include hidden dot-files and dot-directories",
			}),
			cli.New_Flag(cli.New_Flag_Input[bool]{
				Label:       "json",
				Value:       false,
				Description: "emit the counts as JSON instead of a table",
			}),
		},
	})
}

// Counts every path and merges the results into one report.
func main_input_collect(input *Main_Input, scope Main_Scope) (report Report, err error) {
	defer func() { Report_Invariants(report, "main_input_collect.report") }()
	Main_Input_Invariants(*input, "main_input_collect.input")
	Main_Scope_Invariants(scope, "main_input_collect.scope")
	for _, one := range scope.Paths {
		trimmed := Root(strings.TrimSpace(one))
		if trimmed == "" {
			continue
		}
		partial, path_err := main_input_one(input, trimmed, scope)
		if path_err != nil {
			return Report{}, path_err
		}
		report.Files = append(report.Files, partial.Files...)
		report.Skipped.Unreadable += partial.Skipped.Unreadable
		report.Skipped.Oversized += partial.Skipped.Oversized
		report.Skipped.Binary += partial.Skipped.Binary
		report.Skipped.Past_Lines += partial.Skipped.Past_Lines
		report.Skipped.Overflow += partial.Skipped.Overflow
		if report.Skipped.Overflow > DROPPED_COUNT_MAX {
			report.Skipped.Overflow = DROPPED_COUNT_MAX
		}
	}
	return report, nil
}

// Counts a single path: a directory is walked, a file is classified directly.
func main_input_one(
	input *Main_Input, root Root, scope Main_Scope,
) (report Root_Report, err error) {
	defer func() { Root_Report_Invariants(report, "main_input_one.report") }()
	Main_Input_Invariants(*input, "main_input_one.input")
	Root_Invariants(root, "main_input_one.root")
	Main_Scope_Invariants(scope, "main_input_one.scope")
	directory, stat_err := main_is_directory(
		input.Path_Information, File_Path(root))
	if stat_err != nil {
		return Root_Report{}, stat_err
	}
	if directory {
		return main_input_directory(input, root, scope)
	}
	file_path := File_Path(root)
	language, recognized := language_for_path(file_path)
	if !recognized {
		return report, nil
	}
	opened, open_err := input.File(file_path)
	if open_err != nil {
		return Root_Report{}, open_err
	}
	content := Source(nil)
	read_err := main_read_file(opened, func(read Source) { content = read })
	if read_err != nil {
		return Root_Report{}, read_err
	}
	counts := classify(input.Classifier, Classify_File_Input{
		Path: Classified_Path(file_path), Source: content,
		Language: Seeded_Language(language),
	})
	report.Files = append(report.Files, File_Count{
		Path: file_path, Language: language.Name,
		Counts: Counts(counts), Is_Test: false,
	})
	return report, nil
}

// Walks one directory, prefixing each file path with the root so files from different
// roots stay distinguishable.
func main_input_directory(
	input *Main_Input, root Root, scope Main_Scope,
) (report Root_Report, err error) {
	defer func() { Root_Report_Invariants(report, "main_input_directory.report") }()
	Main_Input_Invariants(*input, "main_input_directory.input")
	Root_Invariants(root, "main_input_directory.root")
	Main_Scope_Invariants(scope, "main_input_directory.scope")
	directory_report, count_err := Count(Count_Input{
		File_System:    input.File_System(root),
		Is_Ignored:     main_input_ignore(input, root, scope.No_Ignore),
		Include_Hidden: scope.Include_Hidden,
		Classifier:     input.Classifier,
		Concurrency:    input.Concurrency,
	})
	if count_err != nil {
		return Root_Report{}, count_err
	}
	// The paths are re-rooted into a fresh report, so what the walk left uncounted has
	// to come across with them or it would be lost in the copy.
	report.Skipped = directory_report.Skipped
	for _, file := range directory_report.Files {
		file.Path = File_Path(path.Join(string(root), string(file.Path)))
		report.Files = append(report.Files, file)
	}
	return report, nil
}

// Builds the ignore filter for a root, or nil when ignoring is off.
func main_input_ignore(
	input *Main_Input, root Root, no_ignore No_Ignore,
) (is_ignored Ignore_Predicate) {
	Main_Input_Invariants(*input, "main_input_ignore.input")
	Root_Invariants(root, "main_input_ignore.root")
	No_Ignore_Invariants(no_ignore, "main_input_ignore.no_ignore")
	if no_ignore {
		return nil
	}
	return main_git_ignore(input.Command, root)
}

// FILE_READ_BYTES_MAX limits one explicitly named file to the Source byte bound.
const FILE_READ_BYTES_MAX = SOURCE_BYTES_MAX

// WORKERS_PER_PROCESSOR lets file reads wait while other workers classify source.
const WORKERS_PER_PROCESSOR = 4

// Source_Consumer receives bytes from one successful bounded file read.
type Source_Consumer func(source Source)

// Reports whether a path names a directory.
func main_is_directory(
	information_of func(name File_Path) (information fs.FileInfo, err error),
	name File_Path,
) (is_directory Directory_Status, err error) {
	defer func() {
		Directory_Status_Invariants(is_directory, "main_is_directory.is_directory")
	}()
	File_Path_Invariants(name, "main_is_directory.name")
	information, information_err := information_of(name)
	if information_err != nil {
		return false, information_err
	}
	return Directory_Status(information.IsDir()), nil
}

// Reads one open file through the Source byte bound.
func main_read_file(file fs.File, consume Source_Consumer) (err error) {
	defer file.Close()
	information, information_err := file.Stat()
	if information_err != nil {
		return information_err
	}
	byte_size := information.Size()
	if byte_size < 0 {
		return errors.New("file size is negative")
	}
	if byte_size > FILE_READ_BYTES_MAX {
		byte_size = FILE_READ_BYTES_MAX
	}
	buffer := make([]byte, byte_size)
	_, read_err := io.ReadFull(io.LimitReader(file, byte_size), buffer)
	if main_read_failed(read_err) {
		return read_err
	}
	consume(Source(buffer))
	return nil
}

// Read_Failure is the failure state of one bounded file read.
type Read_Failure bool

// Read_Failure_Invariants states both bounded-read states.
func Read_Failure_Invariants(value Read_Failure, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "A bounded file read fails.").
		Ensure()
}

// Reports whether a bounded read ended in a real error.
func main_read_failed(read_err error) (failed Read_Failure) {
	defer func() { Read_Failure_Invariants(failed, "main_read_failed.failed") }()
	if read_err == nil {
		return false
	}
	if errors.Is(read_err, io.EOF) {
		return false
	}
	return Read_Failure(!errors.Is(read_err, io.ErrUnexpectedEOF))
}

// Builds the ignore predicate from one scoped Git listing.
func main_git_ignore(
	command func(name string, arguments []string) (output []byte, err error),
	root Root,
) (is_ignored Ignore_Predicate) {
	Root_Invariants(root, "main_git_ignore.root")
	output, run_err := command("git", []string{
		"-C", string(root), "ls-files", "-z", "--cached", "--others",
		"--exclude-standard", "--", ".",
	})
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
		for parent != "." && parent != "/" {
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

// ARGUMENTS_COUNT_MIN is the bare program name, which every invocation carries: the
// command line a shell hands over always names the program itself.
const ARGUMENTS_COUNT_MIN = 1

// ARGUMENTS_COUNT_MAX bounds the command line a shell can hand over. A run naming more
// paths than this is past what any invocation realistically carries.
const ARGUMENTS_COUNT_MAX = 4096

// Arguments is one command line, program name included. The underlying slice type is
// unnamed, so an Arguments passes straight to the command-line parser without a copy.
type Arguments []string

// Arguments_Invariants bounds the command line's word count.
func Arguments_Invariants(arguments Arguments, namespace invariant.Namespace) {
	invariant.Tree(arguments, namespace).
		Range_Int(len(arguments), ARGUMENTS_COUNT_MIN, ARGUMENTS_COUNT_MAX).
		Ensure()
}

// PATHS_COUNT_MIN is one path: Main substitutes the current directory when none is
// given, so the resolved scope never holds an empty list.
const PATHS_COUNT_MIN = 1

// PATHS_COUNT_MAX bounds the roots one run counts: the command-line bound less the
// program name, which always occupies the first slot.
const PATHS_COUNT_MAX = ARGUMENTS_COUNT_MAX - 1

// Paths is the resolved list of roots to count.
type Paths []string

// Paths_Invariants bounds the root count. The minimum is one because Main defaults an
// empty list to the current directory before the scope is built.
func Paths_Invariants(paths Paths, namespace invariant.Namespace) {
	invariant.Tree(paths, namespace).
		Range_Int(len(paths), PATHS_COUNT_MIN, PATHS_COUNT_MAX).
		Ensure()
}

// ROOT_BYTES_MIN is the one-byte root: the current directory, ".".
const ROOT_BYTES_MIN = 1

// ROOT_BYTES_MAX is the longest path the host file systems accept.
const ROOT_BYTES_MAX = 4096

// Root is one command-line path to count, a file or a directory. It is distinct from
// File_Path because a root may be the one-byte "." that no counted file's path can be.
type Root string

// Root_Invariants bounds a root's byte length.
func Root_Invariants(root Root, namespace invariant.Namespace) {
	invariant.Tree(root, namespace).
		Range_Int(len(root), ROOT_BYTES_MIN, ROOT_BYTES_MAX).
		Ensure()
}

// No_Ignore is the ignore-filter state of one count scope.
type No_Ignore bool

// No_Ignore_Invariants states both ignore-filter states.
func No_Ignore_Invariants(value No_Ignore, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "The ignore filter is disabled.").
		Ensure()
}

// Include_Hidden is the hidden-path state of one count scope.
type Include_Hidden bool

// Include_Hidden_Invariants states both hidden-path states.
func Include_Hidden_Invariants(value Include_Hidden, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "Hidden paths are included.").
		Ensure()
}

// Recognition is the result of a seeded-language lookup.
type Recognition bool

// Recognition_Invariants states both lookup results.
func Recognition_Invariants(value Recognition, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "A seeded language is recognized.").
		Ensure()
}

// Directory_Status is the directory state of one file path.
type Directory_Status bool

// Directory_Status_Invariants states both file-path states.
func Directory_Status_Invariants(value Directory_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "A file path names a directory.").
		Ensure()
}

// Main_Scope is the resolved scope and override flags of one run.
type Main_Scope struct {
	// Paths is the list of files or directories to count.
	Paths Paths
	// No_Ignore counts gitignored files too.
	No_Ignore No_Ignore
	// Include_Hidden counts hidden dot-files and dot-directories.
	Include_Hidden Include_Hidden
}

// Main_Scope_Invariants states the scope's roots and both override flags.
func Main_Scope_Invariants(scope Main_Scope, namespace invariant.Namespace) {
	Paths_Invariants(scope.Paths, namespace)
	No_Ignore_Invariants(scope.No_Ignore, namespace)
	Include_Hidden_Invariants(scope.Include_Hidden, namespace)
}

// CONCURRENCY_MIN is the smallest worker input before Count clamps the value.
const CONCURRENCY_MIN = -9223372036854775808

// CONCURRENCY_MAX is the largest worker input before Count clamps the value.
const CONCURRENCY_MAX = 9223372036854775807

// Concurrency is the requested worker count before Count applies its bounds.
type Concurrency int

// Concurrency_Invariants states the complete host-supplied worker-count domain.
func Concurrency_Invariants(value Concurrency, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), CONCURRENCY_MIN, CONCURRENCY_MAX).
		Ensure()
}

// Main_Input carries the command line and the operating-system capabilities Main needs.
// Main owns all policy that combines those capabilities.
type Main_Input struct {
	// Arguments is the command line, including the program name.
	Arguments Arguments
	// Output is where the table is written.
	Output io.Writer
	// Error_Output is where usage and errors are written.
	Error_Output io.Writer
	// File_System views a directory as a read-only file system rooted at it.
	File_System func(root Root) (file_system fs.FS)
	// Path_Information returns information about one command-line path.
	Path_Information func(name File_Path) (information fs.FileInfo, err error)
	// File opens one explicitly named file.
	File func(name File_Path) (file fs.File, err error)
	// Command runs one host command and returns its standard output.
	Command func(name string, arguments []string) (output []byte, err error)
	// Classifier partitions one recognized file's source into line kinds.
	Classifier File_Classifier
	// Concurrency bounds the file-counting worker pool. It stays an unbounded integer
	// because it is host-supplied and count_classify clamps it at both ends.
	Concurrency Concurrency
}

// Main_Input_Invariants states the command line, classifier, and worker bound. The
// writers and operating-system capabilities have no preset.
func Main_Input_Invariants(input Main_Input, namespace invariant.Namespace) {
	Arguments_Invariants(input.Arguments, namespace)
	File_Classifier_Invariants(input.Classifier, namespace)
	Concurrency_Invariants(input.Concurrency, namespace)
}

// Block_Comment_Recursion is the recursive state of one block-comment grammar.
type Block_Comment_Recursion bool

// Block_Comment_Recursion_Invariants states both block-comment recursive states.
func Block_Comment_Recursion_Invariants(
	value Block_Comment_Recursion, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "Block comments nest.").
		Ensure()
}

// Long_Bracket is the long-bracket state of one language grammar.
type Long_Bracket bool

// Long_Bracket_Invariants states both long-bracket states.
func Long_Bracket_Invariants(value Long_Bracket, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "Long brackets are enabled.").
		Ensure()
}

// Heredoc is the heredoc state of one language grammar.
type Heredoc bool

// Heredoc_Invariants states both heredoc states.
func Heredoc_Invariants(value Heredoc, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "Heredocs are enabled.").
		Ensure()
}

// Language describes how one language's lines are read: the tokens that begin a
// comment or string, and whether its block comments nest. The scanner is generic
// over this value, so seeding another language is adding a Language.
type Language struct {
	// Name is the language's display name, shown in the rendered table.
	Name Language_Name
	// Line_Comment holds the tokens that begin a comment running to end of line.
	Line_Comment Comment_Tokens
	// Block_Comment_Open begins a block comment, or is empty when there is none.
	Block_Comment_Open Block_Comment_Opener
	// Block_Comment_Close ends a block comment.
	Block_Comment_Close Block_Comment_Closer
	// Block_Comment_Nests is true when an inner open raises the nesting depth, so a
	// single close does not end the comment (Rust); false when the first close ends
	// it (Go, C). It is the only behavioral difference between the seeds' comments.
	Block_Comment_Nests Block_Comment_Recursion
	// Verbatim_Strings are the multi-line string delimiters whose bodies are verbatim.
	Verbatim_Strings Verbatim_Delimiters
	// Quote_Strings are the single-line string and character delimiters.
	Quote_Strings Quote_Delimiters
	// Long_Bracket enables Lua's leveled long brackets: [[ … ]] and [=[ … ]=] as
	// strings, and the same after a line-comment token (--[[ … ]]) as comments.
	Long_Bracket Long_Bracket
	// Test_Prefixes are base-name prefixes that mark a test file, like Python's test_.
	Test_Prefixes Name_Prefixes
	// Test_Infixes are case-sensitive base-name substrings that mark a test file, like
	// Go's _test. or JavaScript's .test. / .spec., delimited so latest.go does not match.
	Test_Infixes Name_Infixes
	// Heredoc enables <<DELIM heredocs (Shell, Ruby, Perl): the body lines up to a line
	// equal to DELIM are code, so a # inside the body is not read as a comment.
	Heredoc Heredoc
}

// Language_Invariants states every field of a language configuration. The three
// booleans and the four token collections are all the scanner reads, so stating them
// here is stating the whole of what a language is.
func Language_Invariants(language Language, namespace invariant.Namespace) {
	Language_Name_Invariants(language.Name, namespace)
	Comment_Tokens_Invariants(language.Line_Comment, namespace)
	Block_Comment_Opener_Invariants(
		language.Block_Comment_Open, namespace)
	Block_Comment_Closer_Invariants(
		language.Block_Comment_Close, namespace)
	Block_Comment_Recursion_Invariants(language.Block_Comment_Nests, namespace)
	Verbatim_Delimiters_Invariants(language.Verbatim_Strings, namespace)
	Quote_Delimiters_Invariants(language.Quote_Strings, namespace)
	Long_Bracket_Invariants(language.Long_Bracket, namespace)
	Name_Prefixes_Invariants(language.Test_Prefixes, namespace)
	Name_Infixes_Invariants(language.Test_Infixes, namespace)
	Heredoc_Invariants(language.Heredoc, namespace)
}

// Seeded_Language is a complete scanner configuration selected from the static table.
type Seeded_Language Language

// Seeded_Language_Invariants rejects a malformed static configuration. The language
// table supplies fixed values, so these are steady properties, not variable bounds.
func Seeded_Language_Invariants(
	language Seeded_Language, namespace invariant.Namespace,
) {
	invariant.Tree(language, namespace).
		Range_Int(len(language.Name), KNOWN_NAME_BYTES_MIN, LANGUAGE_NAME_BYTES_MAX).
		Enum_3_Int(
			len(language.Line_Comment), COMMENT_TOKENS_COUNT_MIN,
			COMMENT_TOKENS_COUNT_ONE, COMMENT_TOKENS_COUNT_MAX).
		Enum_4_Int(
			len(language.Block_Comment_Open), BLOCK_COMMENT_OPENER_BYTES_MIN,
			BLOCK_COMMENT_OPENER_BYTES_BRACE, BLOCK_COMMENT_OPENER_BYTES_PAIR,
			BLOCK_COMMENT_OPENER_BYTES_MAX).
		Enum_4_Int(
			len(language.Block_Comment_Close), BLOCK_COMMENT_CLOSER_BYTES_MIN,
			BLOCK_COMMENT_CLOSER_BYTES_BRACE, BLOCK_COMMENT_CLOSER_BYTES_PAIR,
			BLOCK_COMMENT_CLOSER_BYTES_MAX).
		Sometimes(
			bool(language.Block_Comment_Nests), "Seeded block comments can nest.").
		Enum_3_Int(
			len(language.Verbatim_Strings), VERBATIM_DELIMITERS_COUNT_MIN,
			VERBATIM_DELIMITERS_COUNT_ONE, VERBATIM_DELIMITERS_COUNT_MAX).
		Enum_3_Int(
			len(language.Quote_Strings), QUOTE_DELIMITERS_COUNT_MIN,
			QUOTE_DELIMITERS_COUNT_ONE, QUOTE_DELIMITERS_COUNT_MAX).
		Sometimes(bool(language.Long_Bracket), "Seeded long brackets are enabled.").
		Enum_Int(
			len(language.Test_Prefixes), NAME_PREFIXES_COUNT_MIN,
			NAME_PREFIXES_COUNT_MAX).
		Enum_4_Int(
			len(language.Test_Infixes), NAME_INFIXES_COUNT_MIN,
			NAME_INFIXES_COUNT_ONE, NAME_INFIXES_COUNT_TWO,
			NAME_INFIXES_COUNT_MAX).
		Sometimes(bool(language.Heredoc), "Seeded heredocs are enabled.").
		Ensure()
}

// LANGUAGE_NAME_BYTES_MIN is the empty name of the zero Language, which the lookups
// return alongside a false when no language matches.
const LANGUAGE_NAME_BYTES_MIN = 0

// LANGUAGE_NAME_BYTES_MAX is the longest seeded display name, "Visual Basic".
const LANGUAGE_NAME_BYTES_MAX = 12

// Language_Name is a language's display name. The unrecognized lookup yields the empty
// name, so the type spans the zero value as well as every seeded name.
type Language_Name string

// Language_Name_Invariants bounds a display name's byte length.
func Language_Name_Invariants(name Language_Name, namespace invariant.Namespace) {
	invariant.Tree(name, namespace).
		Range_Int(len(name), LANGUAGE_NAME_BYTES_MIN, LANGUAGE_NAME_BYTES_MAX).
		Ensure()
}

// COMMENT_TOKENS_COUNT_MIN is a language with no line comment at all, like HTML.
const COMMENT_TOKENS_COUNT_MIN = 0

// COMMENT_TOKENS_COUNT_ONE is the single line-comment token most languages carry.
const COMMENT_TOKENS_COUNT_ONE = 1

// COMMENT_TOKENS_COUNT_MAX is the two line-comment tokens the richest seeds carry.
const COMMENT_TOKENS_COUNT_MAX = 2

// Comment_Tokens are the tokens that begin a comment running to end of line.
type Comment_Tokens []string

// Comment_Tokens_Invariants holds how many line-comment tokens a language declares to
// the three counts that occur, and witnesses each count.
func Comment_Tokens_Invariants(tokens Comment_Tokens, namespace invariant.Namespace) {
	invariant.Tree(tokens, namespace).
		Enum_3_Int(
			len(tokens), COMMENT_TOKENS_COUNT_MIN, COMMENT_TOKENS_COUNT_ONE,
			COMMENT_TOKENS_COUNT_MAX).
		Ensure()
}

// Long_Bracket_Comment_Tokens is the one line-comment token of a long-bracket seed.
type Long_Bracket_Comment_Tokens Comment_Tokens

// Long_Bracket_Comment_Tokens_Invariants pins the shared single-token property.
func Long_Bracket_Comment_Tokens_Invariants(
	tokens Long_Bracket_Comment_Tokens, namespace invariant.Namespace,
) {
	invariant.Always(
		len(tokens) == COMMENT_TOKENS_COUNT_ONE,
		"A long-bracket language always has one line-comment token.")
}

// BLOCK_COMMENT_OPENER_BYTES_MIN is the empty opener of a language with no block
// comment.
const BLOCK_COMMENT_OPENER_BYTES_MIN = 0

// BLOCK_COMMENT_OPENER_BYTES_MAX is the four-byte "<!--" of the markup languages.
const BLOCK_COMMENT_OPENER_BYTES_MAX = 4

// BLOCK_COMMENT_OPENER_BYTES_BRACE is the one-byte "{" of the Pascal family.
const BLOCK_COMMENT_OPENER_BYTES_BRACE = 1

// BLOCK_COMMENT_OPENER_BYTES_PAIR is the two-byte openers like "/*" and "(*".
const BLOCK_COMMENT_OPENER_BYTES_PAIR = 2

// Block_Comment_Opener begins a block comment, or is empty when there is none.
type Block_Comment_Opener string

// Block_Comment_Opener_Invariants holds an opener's byte length to the four widths the
// seeds use, and witnesses each width. Three bytes is not a member: the seeds run "",
// "{", the two-byte pairs, and "<!--", so no seeded opener is three bytes wide.
func Block_Comment_Opener_Invariants(
	opener Block_Comment_Opener, namespace invariant.Namespace,
) {
	invariant.Tree(opener, namespace).
		Enum_4_Int(
			len(opener), BLOCK_COMMENT_OPENER_BYTES_MIN,
			BLOCK_COMMENT_OPENER_BYTES_BRACE, BLOCK_COMMENT_OPENER_BYTES_PAIR,
			BLOCK_COMMENT_OPENER_BYTES_MAX).
		Ensure()
}

// Active_Block_Comment_Opener is the nonempty opener of a block-comment seed.
type Active_Block_Comment_Opener Block_Comment_Opener

// Active_Block_Comment_Opener_Invariants excludes the shared empty member.
func Active_Block_Comment_Opener_Invariants(
	opener Active_Block_Comment_Opener, namespace invariant.Namespace,
) {
	invariant.Tree(opener, namespace).
		Enum_3_Int(
			len(opener), BLOCK_COMMENT_OPENER_BYTES_BRACE,
			BLOCK_COMMENT_OPENER_BYTES_PAIR, BLOCK_COMMENT_OPENER_BYTES_MAX).
		Ensure()
}

// BLOCK_COMMENT_CLOSER_BYTES_MIN is the empty closer of a language with no block
// comment.
const BLOCK_COMMENT_CLOSER_BYTES_MIN = 0

// BLOCK_COMMENT_CLOSER_BYTES_MAX is the three-byte "-->" of the markup languages. It is
// one shorter than the opener max, which is why the two are distinct types: a shared
// bound would demand a width one of them can never reach.
const BLOCK_COMMENT_CLOSER_BYTES_MAX = 3

// BLOCK_COMMENT_CLOSER_BYTES_BRACE is the one-byte "}" of the Pascal family.
const BLOCK_COMMENT_CLOSER_BYTES_BRACE = 1

// BLOCK_COMMENT_CLOSER_BYTES_PAIR is the two-byte closers like "*/" and "*)".
const BLOCK_COMMENT_CLOSER_BYTES_PAIR = 2

// Block_Comment_Closer ends a block comment, or is empty when there is none.
type Block_Comment_Closer string

// Block_Comment_Closer_Invariants holds a closer's byte length to the four widths the
// seeds use, and witnesses each width.
func Block_Comment_Closer_Invariants(
	closer Block_Comment_Closer, namespace invariant.Namespace,
) {
	invariant.Tree(closer, namespace).
		Enum_4_Int(
			len(closer), BLOCK_COMMENT_CLOSER_BYTES_MIN,
			BLOCK_COMMENT_CLOSER_BYTES_BRACE, BLOCK_COMMENT_CLOSER_BYTES_PAIR,
			BLOCK_COMMENT_CLOSER_BYTES_MAX).
		Ensure()
}

// Active_Block_Comment_Closer is the nonempty closer paired with a block opener.
type Active_Block_Comment_Closer Block_Comment_Closer

// Active_Block_Comment_Closer_Invariants excludes the shared empty member.
func Active_Block_Comment_Closer_Invariants(
	closer Active_Block_Comment_Closer, namespace invariant.Namespace,
) {
	invariant.Tree(closer, namespace).
		Enum_3_Int(
			len(closer), BLOCK_COMMENT_CLOSER_BYTES_BRACE,
			BLOCK_COMMENT_CLOSER_BYTES_PAIR, BLOCK_COMMENT_CLOSER_BYTES_MAX).
		Ensure()
}

// NAME_PREFIXES_COUNT_MIN is a language whose test files are not marked by a prefix.
const NAME_PREFIXES_COUNT_MIN = 0

// NAME_PREFIXES_COUNT_MAX is the one prefix the seeds use, Python's "test_".
const NAME_PREFIXES_COUNT_MAX = 1

// Name_Prefixes are base-name prefixes that mark a test file.
type Name_Prefixes []string

// Name_Prefixes_Invariants holds how many test prefixes a language declares to the two
// counts that occur, and witnesses each count.
func Name_Prefixes_Invariants(prefixes Name_Prefixes, namespace invariant.Namespace) {
	invariant.Tree(prefixes, namespace).
		Enum_Int(len(prefixes), NAME_PREFIXES_COUNT_MIN, NAME_PREFIXES_COUNT_MAX).
		Ensure()
}

// NAME_INFIXES_COUNT_MIN is a language whose test files are not marked by an infix.
const NAME_INFIXES_COUNT_MIN = 0

// NAME_INFIXES_COUNT_ONE is the single infix a language like Go carries, "_test.".
const NAME_INFIXES_COUNT_ONE = 1

// NAME_INFIXES_COUNT_TWO is the two infixes a language like JavaScript carries.
const NAME_INFIXES_COUNT_TWO = 2

// NAME_INFIXES_COUNT_MAX is the three infixes the richest seed carries.
const NAME_INFIXES_COUNT_MAX = 3

// Name_Infixes are case-sensitive base-name substrings that mark a test file.
type Name_Infixes []string

// Name_Infixes_Invariants holds how many test infixes a language declares to the four
// counts that occur, and witnesses each count.
func Name_Infixes_Invariants(infixes Name_Infixes, namespace invariant.Namespace) {
	invariant.Tree(infixes, namespace).
		Enum_4_Int(
			len(infixes), NAME_INFIXES_COUNT_MIN, NAME_INFIXES_COUNT_ONE,
			NAME_INFIXES_COUNT_TWO, NAME_INFIXES_COUNT_MAX).
		Ensure()
}

// VERBATIM_DELIMITERS_COUNT_MIN is a language with no verbatim string at all.
const VERBATIM_DELIMITERS_COUNT_MIN = 0

// VERBATIM_DELIMITERS_COUNT_ONE is the single verbatim form most seeds carry.
const VERBATIM_DELIMITERS_COUNT_ONE = 1

// VERBATIM_DELIMITERS_COUNT_MAX is the two verbatim forms the richest seeds carry.
const VERBATIM_DELIMITERS_COUNT_MAX = 2

// Verbatim_Delimiters are the multi-line string delimiters whose bodies are verbatim.
type Verbatim_Delimiters []Verbatim_Delimiter

// Verbatim_Delimiters_Invariants holds how many verbatim forms a language declares to
// the three counts that occur, and witnesses each count.
func Verbatim_Delimiters_Invariants(
	delimiters Verbatim_Delimiters, namespace invariant.Namespace,
) {
	invariant.Tree(delimiters, namespace).
		Enum_3_Int(
			len(delimiters), VERBATIM_DELIMITERS_COUNT_MIN,
			VERBATIM_DELIMITERS_COUNT_ONE, VERBATIM_DELIMITERS_COUNT_MAX).
		Ensure()
}

// Active_Verbatim_Delimiters are the one or two verbatim forms of a seed that has one.
type Active_Verbatim_Delimiters Verbatim_Delimiters

// Active_Verbatim_Delimiters_Invariants excludes the shared empty collection.
func Active_Verbatim_Delimiters_Invariants(
	delimiters Active_Verbatim_Delimiters, namespace invariant.Namespace,
) {
	invariant.Tree(delimiters, namespace).
		Enum_Int(
			len(delimiters), VERBATIM_DELIMITERS_COUNT_ONE,
			VERBATIM_DELIMITERS_COUNT_MAX).
		Ensure()
}

// QUOTE_DELIMITERS_COUNT_MIN is a language with no quoted string at all.
const QUOTE_DELIMITERS_COUNT_MIN = 0

// QUOTE_DELIMITERS_COUNT_ONE is the lone quoted form a language without a character
// literal carries.
const QUOTE_DELIMITERS_COUNT_ONE = 1

// QUOTE_DELIMITERS_COUNT_MAX is the string-and-character pair most seeds carry.
const QUOTE_DELIMITERS_COUNT_MAX = 2

// Quote_Delimiters are the single-line string and character delimiters.
type Quote_Delimiters []Quote_Delimiter

// Quote_Delimiters_Invariants holds how many quoted forms a language declares to the
// three counts that occur, and witnesses each count.
func Quote_Delimiters_Invariants(
	delimiters Quote_Delimiters, namespace invariant.Namespace,
) {
	invariant.Tree(delimiters, namespace).
		Enum_3_Int(
			len(delimiters), QUOTE_DELIMITERS_COUNT_MIN, QUOTE_DELIMITERS_COUNT_ONE,
			QUOTE_DELIMITERS_COUNT_MAX).
		Ensure()
}

// Active_Quote_Delimiters are the one or two quoted forms of a seed that has one.
type Active_Quote_Delimiters Quote_Delimiters

// Active_Quote_Delimiters_Invariants excludes the shared empty collection.
func Active_Quote_Delimiters_Invariants(
	delimiters Active_Quote_Delimiters, namespace invariant.Namespace,
) {
	invariant.Tree(delimiters, namespace).
		Enum_Int(
			len(delimiters), QUOTE_DELIMITERS_COUNT_ONE,
			QUOTE_DELIMITERS_COUNT_MAX).
		Ensure()
}

// QUOTE_PAIR_COUNT is the two forms a pair carries: a string and a character literal,
// or a double- and a single-quoted string.
const QUOTE_PAIR_COUNT = 2

// Quote_Pair is a shared two-delimiter set. It is distinct from Quote_Delimiters
// because its width is fixed by construction rather than varying per language, so its
// bound is a point and the counts a language may carry are not reachable here.
type Quote_Pair []Quote_Delimiter

// Quote_Pair_Invariants pins a shared pair to the two forms it always holds.
func Quote_Pair_Invariants(delimiters Quote_Pair, namespace invariant.Namespace) {
	invariant.Always(
		len(delimiters) == QUOTE_PAIR_COUNT, "A shared quote pair always holds two forms.")
}

// QUOTE_SINGLE_COUNT is the one form a lone double-quoted string set carries.
const QUOTE_SINGLE_COUNT = 1

// Quote_Single is a shared one-delimiter set, fixed by construction like Quote_Pair.
type Quote_Single []Quote_Delimiter

// Quote_Single_Invariants pins a lone shared delimiter to the one form it holds.
func Quote_Single_Invariants(delimiters Quote_Single, namespace invariant.Namespace) {
	invariant.Always(
		len(delimiters) == QUOTE_SINGLE_COUNT,
		"A lone shared quote delimiter always holds one form.")
}

// Hashability is the hash-delimiter state of one verbatim string form.
type Hashability bool

// Hashability_Invariants states both hash-delimiter states.
func Hashability_Invariants(value Hashability, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "The delimiter uses hashes.").
		Ensure()
}

// Verbatim_Delimiter describes a string whose body is taken verbatim and may span
// lines, so a comment token inside it is inert.
type Verbatim_Delimiter struct {
	// Open is the opening delimiter of a fixed verbatim string (a backtick or triple
	// quote), or, when Hashable, the lead before the hashes (Rust's "r" or "br").
	Open Verbatim_Opener
	// Close is the terminator of a fixed verbatim string, unused when Hashable.
	Close Verbatim_Closer
	// Hashable marks a Rust-style raw string: the lead, then N hashes, then a quote;
	// it closes only on a quote followed by exactly N hashes. Without the quote the
	// lead is a raw identifier, not a string.
	Hashable Hashability
}

// Verbatim_Delimiter_Invariants states a verbatim form's two delimiters and its shape.
func Verbatim_Delimiter_Invariants(
	delimiter Verbatim_Delimiter, namespace invariant.Namespace,
) {
	Verbatim_Opener_Invariants(delimiter.Open, namespace)
	Verbatim_Closer_Invariants(delimiter.Close, namespace)
	Hashability_Invariants(delimiter.Hashable, namespace)
}

// VERBATIM_OPENER_BYTES_MIN is the one-byte backtick and Rust's one-byte "r" lead.
const VERBATIM_OPENER_BYTES_MIN = 1

// VERBATIM_OPENER_BYTES_BYTE_LEAD is Rust's two-byte "br" lead, which adds the
// byte-string "b" to the raw "r".
const VERBATIM_OPENER_BYTES_BYTE_LEAD = 2

// VERBATIM_OPENER_BYTES_MAX is the three-byte triple quote.
const VERBATIM_OPENER_BYTES_MAX = 3

// Verbatim_Opener opens a verbatim string, or leads the hashes of a hashable one.
type Verbatim_Opener string

// Verbatim_Opener_Invariants holds a verbatim opener's byte length to the three widths
// that occur — the backtick, Rust's "br", the triple quote — and witnesses each width.
func Verbatim_Opener_Invariants(opener Verbatim_Opener, namespace invariant.Namespace) {
	invariant.Tree(opener, namespace).
		Enum_3_Int(
			len(opener), VERBATIM_OPENER_BYTES_MIN, VERBATIM_OPENER_BYTES_BYTE_LEAD,
			VERBATIM_OPENER_BYTES_MAX).
		Ensure()
}

// VERBATIM_CLOSER_BYTES_MIN is the empty closer a hashable form carries, whose
// terminator is computed from the hash count rather than declared.
const VERBATIM_CLOSER_BYTES_MIN = 0

// VERBATIM_CLOSER_BYTES_BACKTICK is the one-byte backtick.
const VERBATIM_CLOSER_BYTES_BACKTICK = 1

// VERBATIM_CLOSER_BYTES_PAIR is Python's two-byte empty pair.
const VERBATIM_CLOSER_BYTES_PAIR = 2

// VERBATIM_CLOSER_BYTES_MAX is the three-byte triple quote.
const VERBATIM_CLOSER_BYTES_MAX = 3

// Verbatim_Closer terminates a fixed verbatim string, empty when the form is hashable.
type Verbatim_Closer string

// Verbatim_Closer_Invariants holds a verbatim closer's byte length to the four widths
// that occur, and witnesses each width: the empty closer of a hashable form, the
// one-byte backtick, Python's two-byte empty pair, and the three-byte triple quote.
func Verbatim_Closer_Invariants(closer Verbatim_Closer, namespace invariant.Namespace) {
	invariant.Tree(closer, namespace).
		Enum_4_Int(
			len(closer), VERBATIM_CLOSER_BYTES_MIN, VERBATIM_CLOSER_BYTES_BACKTICK,
			VERBATIM_CLOSER_BYTES_PAIR, VERBATIM_CLOSER_BYTES_MAX).
		Ensure()
}

// Character_Likeness is the character-literal state of one quote delimiter.
type Character_Likeness bool

// Character_Likeness_Invariants states both character-literal states.
func Character_Likeness_Invariants(
	value Character_Likeness, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "The delimiter holds one character.").
		Ensure()
}

// Quote_Delimiter describes a single-line string or character literal.
type Quote_Delimiter struct {
	// Open is the opening delimiter.
	Open Quote_Opener
	// Close is the closing delimiter.
	Close Quote_Closer
	// Escape is the byte that escapes the following character, or zero for none.
	Escape Escape_Byte
	// Character_Like marks a character or rune literal, whose apostrophe must be told
	// apart from a Rust lifetime tick that opens nothing.
	Character_Like Character_Likeness
}

// Quote_Delimiter_Invariants states a quoted form's marks, its escape, and its shape.
func Quote_Delimiter_Invariants(
	delimiter Quote_Delimiter, namespace invariant.Namespace,
) {
	Quote_Opener_Invariants(delimiter.Open, namespace)
	Quote_Closer_Invariants(delimiter.Close, namespace)
	Escape_Byte_Invariants(delimiter.Escape, namespace)
	Character_Likeness_Invariants(delimiter.Character_Like, namespace)
}

// String_Delimiter contains only the fields used to scan a string. Character likeness
// selects this path before the conversion and is not part of the scan.
type String_Delimiter struct {
	// Open begins the string.
	Open Quote_Opener
	// Close ends the string.
	Close Quote_Closer
	// Escape marks the next byte as literal, or is zero when escaping is absent.
	Escape Escape_Byte
}

// String_Delimiter_Invariants states the fixed shape that reaches string scanning.
func String_Delimiter_Invariants(
	delimiter String_Delimiter, namespace invariant.Namespace,
) {
	Quote_Opener_Invariants(delimiter.Open, namespace)
	Quote_Closer_Invariants(delimiter.Close, namespace)
	Escape_Byte_Invariants(delimiter.Escape, namespace)
}

// QUOTE_MARK_BYTES is the width of every quote mark: a single byte. A quoted form's
// delimiter is exactly one byte in every seeded language, so the width is a point, not
// an interval.
const QUOTE_MARK_BYTES = 1

// Quote_Opener opens a single-line string or character literal.
type Quote_Opener string

// Quote_Opener_Invariants pins an opener to the single byte that each form uses.
func Quote_Opener_Invariants(mark Quote_Opener, namespace invariant.Namespace) {
	invariant.Always(
		len(mark) == QUOTE_MARK_BYTES, "A quote opener is always a single byte.")
}

// Quote_Closer closes a single-line string or character literal.
type Quote_Closer string

// Quote_Closer_Invariants pins a closer to the single byte that each form uses.
func Quote_Closer_Invariants(mark Quote_Closer, namespace invariant.Namespace) {
	invariant.Always(
		len(mark) == QUOTE_MARK_BYTES, "A quote closer is always a single byte.")
}

// ESCAPE_BYTE_NONE is the zero escape of a form that has no escape character at all,
// as Pascal's single-quoted string does not.
const ESCAPE_BYTE_NONE Escape_Byte = 0

// ESCAPE_BYTE_BACKSLASH is the only escape character any seeded language uses. It is
// spelled as its numeric value rather than as '\\' because a preset bound must resolve
// to a constant the recorder can read, and a rune literal does not.
const ESCAPE_BYTE_BACKSLASH Escape_Byte = 92

// Escape_Byte is the byte that escapes the following character inside a quoted form.
// The vocabulary is closed at two members with a wide gap between them, so it is an
// enumeration rather than a range: no value between them is ever legal.
type Escape_Byte uint8

// Escape_Byte_Invariants holds an escape to the two members that occur.
func Escape_Byte_Invariants(escape Escape_Byte, namespace invariant.Namespace) {
	invariant.Tree(escape, namespace).
		Enum_Uint8(
			uint8(escape), uint8(ESCAPE_BYTE_NONE), uint8(ESCAPE_BYTE_BACKSLASH)).
		Ensure()
}

// Returns the double-quoted string and single-quoted character delimiters shared by
// the C-family languages.
func c_family_quotes() (delimiters Quote_Pair) {
	defer func() { Quote_Pair_Invariants(delimiters, "c_family_quotes.delimiters") }()
	return Quote_Pair{
		{Open: "\"", Close: "\"", Escape: '\\'},
		{Open: "'", Close: "'", Escape: '\\', Character_Like: true},
	}
}

// Returns plain double- and single-quoted string delimiters with backslash escapes.
func plain_quotes() (delimiters Quote_Pair) {
	defer func() { Quote_Pair_Invariants(delimiters, "plain_quotes.delimiters") }()
	return Quote_Pair{
		{Open: "\"", Close: "\"", Escape: '\\'},
		{Open: "'", Close: "'", Escape: '\\'},
	}
}

// Returns a single double-quoted string delimiter with backslash escapes.
func double_quote() (delimiters Quote_Single) {
	defer func() { Quote_Single_Invariants(delimiters, "double_quote.delimiters") }()
	return Quote_Single{{Open: "\"", Close: "\"", Escape: '\\'}}
}

// LANGUAGE_SEED_BUCKET_COUNT keeps each static-data helper within the source bound.
const LANGUAGE_SEED_BUCKET_COUNT = 29

// LANGUAGE_SEED_BUCKET_MIN is the first table partition.
const LANGUAGE_SEED_BUCKET_MIN = 0

// LANGUAGE_SEED_BUCKET_MAX is the last table partition.
const LANGUAGE_SEED_BUCKET_MAX = LANGUAGE_SEED_BUCKET_COUNT - 1

// Language_Seed_Bucket is one partition index in the static seed table.
type Language_Seed_Bucket int

// Language_Seed_Bucket_Invariants bounds a seed table partition.
func Language_Seed_Bucket_Invariants(bucket Language_Seed_Bucket, namespace invariant.Namespace) {
	invariant.Tree(bucket, namespace).Range_Int(
		int(bucket), LANGUAGE_SEED_BUCKET_MIN, LANGUAGE_SEED_BUCKET_MAX).Ensure()
}

// LANGUAGE_SEEDS_COUNT is the fixed capacity of each table partition.
const LANGUAGE_SEEDS_COUNT = 6

// Language_Seeds is one fixed-capacity partition of the static seed table.
type Language_Seeds []Seeded_Language

// Language_Seeds_Invariants pins every table partition to the shared capacity.
func Language_Seeds_Invariants(seeds Language_Seeds, namespace invariant.Namespace) {
	invariant.Always(
		len(seeds) == LANGUAGE_SEEDS_COUNT,
		"A language seed partition always has its fixed capacity.")
}

// Language_Seeds_Source supplies one immutable partition of the static seed table.
type Language_Seeds_Source func() (seeds Language_Seeds)

// Selects one configuration from the static seed table. One boundary owns the whole
// seed domain, so a fixed seed does not claim every language property.
func language_seed(name Known_Name) (language Seeded_Language) {
	defer func() { Seeded_Language_Invariants(language, "language_seed.language") }()
	Known_Name_Invariants(name, "language_seed.name")
	sources := [...]Language_Seeds_Source{
		language_seed_bucket_0,
		language_seed_bucket_1,
		language_seed_bucket_2,
		language_seed_bucket_3,
		language_seed_bucket_4,
		language_seed_bucket_5,
		language_seed_bucket_6,
		language_seed_bucket_7,
		language_seed_bucket_8,
		language_seed_bucket_9,
		language_seed_bucket_10,
		language_seed_bucket_11,
		language_seed_bucket_12,
		language_seed_bucket_13,
		language_seed_bucket_14,
		language_seed_bucket_15,
		language_seed_bucket_16,
		language_seed_bucket_17,
		language_seed_bucket_18,
		language_seed_bucket_19,
		language_seed_bucket_20,
		language_seed_bucket_21,
		language_seed_bucket_22,
		language_seed_bucket_23,
		language_seed_bucket_24,
		language_seed_bucket_25,
		language_seed_bucket_26,
		language_seed_bucket_27,
		language_seed_bucket_28,
	}
	seeds := sources[int(language_seed_bucket(name))]()
	for _, seed := range seeds {
		if seed.Name == Language_Name(name) {
			return seed
		}
	}
	return Seeded_Language{}
}

// Maps a name to a stable partition without retaining mutable package state.
func language_seed_bucket(name Known_Name) (bucket Language_Seed_Bucket) {
	defer func() {
		Language_Seed_Bucket_Invariants(bucket, "language_seed_bucket.bucket")
	}()
	Known_Name_Invariants(name, "language_seed_bucket.name")
	for _, character := range name {
		bucket = Language_Seed_Bucket(
			(int(bucket)*33 + int(character)) % LANGUAGE_SEED_BUCKET_COUNT)
	}
	return bucket
}

func language_seed_bucket_0() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_0.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:          "Ada",
			Line_Comment:  []string{"--"},
			Quote_Strings: Quote_Delimiters(c_family_quotes()),
		},
		Seeded_Language{
			Name:          "Emacs Lisp",
			Line_Comment:  []string{";"},
			Quote_Strings: Quote_Delimiters(double_quote()),
		},
		Seeded_Language{
			Name:                "Kotlin",
			Test_Infixes:        []string{"Test.", "Tests."},
			Line_Comment:        []string{"//"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Block_Comment_Nests: true,
			Verbatim_Strings: []Verbatim_Delimiter{
				{Open: "\"\"\"", Close: "\"\"\""},
			},
			Quote_Strings: []Quote_Delimiter{
				{Open: "\"", Close: "\"", Escape: '\\'},
				{Open: "'", Close: "'", Escape: '\\', Character_Like: true},
			},
		},
		{},
		{},
		{},
	}
}

func language_seed_bucket_1() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_1.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:                "Groovy",
			Line_Comment:        []string{"//"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Verbatim_Strings: []Verbatim_Delimiter{
				{Open: "\"\"\"", Close: "\"\"\""},
				{Open: "'''", Close: "'''"},
			},
			Quote_Strings: Quote_Delimiters(plain_quotes()),
		},
		{},
		{},
		{},
		{},
		{},
	}
}

func language_seed_bucket_2() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_2.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:                "JSONC",
			Line_Comment:        []string{"//"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Quote_Strings:       Quote_Delimiters(double_quote()),
		},
		Seeded_Language{
			Name:                "Vue",
			Block_Comment_Open:  "<!--",
			Block_Comment_Close: "-->",
		},
		{},
		{},
		{},
		{},
	}
}

func language_seed_bucket_3() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_3.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:                "LESS",
			Line_Comment:        []string{"//"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Quote_Strings: []Quote_Delimiter{
				{Open: "\"", Close: "\"", Escape: '\\'},
				{Open: "'", Close: "'", Escape: '\\'},
			},
		},
		{},
		{},
		{},
		{},
		{},
	}
}

func language_seed_bucket_4() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_4.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:                "Astro",
			Block_Comment_Open:  "<!--",
			Block_Comment_Close: "-->",
		},
		Seeded_Language{
			Name:          "CMake",
			Line_Comment:  []string{"#"},
			Long_Bracket:  true,
			Quote_Strings: Quote_Delimiters(double_quote()),
		},
		Seeded_Language{
			Name:         "Elixir",
			Test_Infixes: []string{"_test."},
			Line_Comment: []string{"#"},
			Verbatim_Strings: []Verbatim_Delimiter{
				{Open: "\"\"\"", Close: "\"\"\""},
				{Open: "'''", Close: "'''"},
			},
			Quote_Strings: Quote_Delimiters(plain_quotes()),
		},
		Seeded_Language{
			Name:                "Haskell",
			Line_Comment:        []string{"--"},
			Block_Comment_Open:  "{-",
			Block_Comment_Close: "-}",
			Block_Comment_Nests: true,
			Quote_Strings:       Quote_Delimiters(double_quote()),
		},
		Seeded_Language{
			Name:                "Solidity",
			Line_Comment:        []string{"//"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Quote_Strings:       Quote_Delimiters(plain_quotes()),
		},
		Seeded_Language{
			Name:                "TypeScript",
			Test_Infixes:        []string{".test.", ".spec."},
			Line_Comment:        []string{"//"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Block_Comment_Nests: false,
			Verbatim_Strings:    []Verbatim_Delimiter{{Open: "`", Close: "`"}},
			Quote_Strings: []Quote_Delimiter{
				{Open: "\"", Close: "\"", Escape: '\\'},
				{Open: "'", Close: "'", Escape: '\\'},
			},
		},
	}
}

func language_seed_bucket_5() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_5.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:          "Nushell",
			Line_Comment:  []string{"#"},
			Quote_Strings: Quote_Delimiters(plain_quotes()),
		},
		Seeded_Language{
			Name:                "Racket",
			Line_Comment:        []string{";"},
			Block_Comment_Open:  "#|",
			Block_Comment_Close: "|#",
			Block_Comment_Nests: true,
			Quote_Strings:       Quote_Delimiters(double_quote()),
		},
		{},
		{},
		{},
		{},
	}
}

func language_seed_bucket_6() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_6.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:         "Ruby",
			Test_Infixes: []string{"_spec.", "_test."},
			Line_Comment: []string{"#"},
			Heredoc:      true,
			Quote_Strings: []Quote_Delimiter{
				{Open: "\"", Close: "\"", Escape: '\\'},
				{Open: "'", Close: "'", Escape: '\\'},
			},
		},
		Seeded_Language{
			Name:         "TOML",
			Line_Comment: []string{"#"},
			Verbatim_Strings: []Verbatim_Delimiter{
				{Open: "\"\"\"", Close: "\"\"\""},
				{Open: "'''", Close: "'''"},
			},
			Quote_Strings: []Quote_Delimiter{
				{Open: "\"", Close: "\"", Escape: '\\'},
				{Open: "'", Close: "'", Escape: '\\'},
			},
		},
		Seeded_Language{
			Name:          "Visual Basic",
			Line_Comment:  []string{"'"},
			Quote_Strings: Quote_Delimiters(double_quote()),
		},
		{},
		{},
		{},
	}
}

func language_seed_bucket_7() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_7.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:          "Clojure",
			Line_Comment:  []string{";"},
			Quote_Strings: Quote_Delimiters(double_quote()),
		},
		{},
		{},
		{},
		{},
		{},
	}
}

func language_seed_bucket_8() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_8.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:                "CSS",
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Quote_Strings: []Quote_Delimiter{
				{Open: "\"", Close: "\"", Escape: '\\'},
				{Open: "'", Close: "'", Escape: '\\'},
			},
		},
		Seeded_Language{
			Name:                "Nim",
			Line_Comment:        []string{"#"},
			Block_Comment_Open:  "#[",
			Block_Comment_Close: "]#",
			Block_Comment_Nests: true,
			Verbatim_Strings: []Verbatim_Delimiter{
				{Open: "\"\"\"", Close: "\"\"\""},
			},
			Quote_Strings: Quote_Delimiters(double_quote()),
		},
		{},
		{},
		{},
		{},
	}
}

func language_seed_bucket_9() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_9.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:                "C",
			Line_Comment:        []string{"//"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Quote_Strings: []Quote_Delimiter{
				{Open: "\"", Close: "\"", Escape: '\\'},
				{Open: "'", Close: "'", Escape: '\\', Character_Like: true},
			},
		},
		Seeded_Language{
			Name:                "Dart",
			Line_Comment:        []string{"//"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Block_Comment_Nests: true,
			Verbatim_Strings: []Verbatim_Delimiter{
				{Open: "\"\"\"", Close: "\"\"\""},
				{Open: "'''", Close: "'''"},
			},
			Quote_Strings: Quote_Delimiters(plain_quotes()),
		},
		Seeded_Language{
			Name:                "Swift",
			Test_Infixes:        []string{"Tests.", "Test."},
			Line_Comment:        []string{"//"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Block_Comment_Nests: true,
			Verbatim_Strings: []Verbatim_Delimiter{
				{Open: "\"\"\"", Close: "\"\"\""},
			},
			Quote_Strings: []Quote_Delimiter{
				{Open: "\"", Close: "\"", Escape: '\\'},
			},
		},
		Seeded_Language{
			Name:         "TeX",
			Line_Comment: []string{"%"},
		},
		Seeded_Language{
			Name:                "XAML",
			Block_Comment_Open:  "<!--",
			Block_Comment_Close: "-->",
		},
		{},
	}
}

func language_seed_bucket_10() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_10.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:                "D",
			Line_Comment:        []string{"//"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Verbatim_Strings:    []Verbatim_Delimiter{{Open: "`", Close: "`"}},
			Quote_Strings:       Quote_Delimiters(c_family_quotes()),
		},
		Seeded_Language{
			Name:          "Erlang",
			Line_Comment:  []string{"%"},
			Quote_Strings: Quote_Delimiters(double_quote()),
		},
		{},
		{},
		{},
		{},
	}
}

func language_seed_bucket_11() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_11.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:                "C++",
			Line_Comment:        []string{"//"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Quote_Strings: []Quote_Delimiter{
				{Open: "\"", Close: "\"", Escape: '\\'},
				{Open: "'", Close: "'", Escape: '\\', Character_Like: true},
			},
		},
		Seeded_Language{
			Name:                "Rust",
			Line_Comment:        []string{"//"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Block_Comment_Nests: true,
			Verbatim_Strings: []Verbatim_Delimiter{
				{Open: "br", Hashable: true},
				{Open: "r", Hashable: true},
			},
			Quote_Strings: []Quote_Delimiter{
				{Open: "\"", Close: "\"", Escape: '\\'},
				{Open: "'", Close: "'", Escape: '\\', Character_Like: true},
			},
		},
		Seeded_Language{
			Name:                "Svelte",
			Block_Comment_Open:  "<!--",
			Block_Comment_Close: "-->",
		},
		Seeded_Language{
			Name:                "XSLT",
			Block_Comment_Open:  "<!--",
			Block_Comment_Close: "-->",
		},
		{},
		{},
	}
}

func language_seed_bucket_12() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_12.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:         "Lua",
			Line_Comment: []string{"--"},
			Long_Bracket: true,
			Quote_Strings: []Quote_Delimiter{
				{Open: "\"", Close: "\"", Escape: '\\'},
				{Open: "'", Close: "'", Escape: '\\'},
			},
		},
		{},
		{},
		{},
		{},
		{},
	}
}

func language_seed_bucket_13() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_13.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:                "C#",
			Test_Infixes:        []string{"Test.", "Tests."},
			Line_Comment:        []string{"//"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Verbatim_Strings: []Verbatim_Delimiter{
				{Open: "\"\"\"", Close: "\"\"\""},
			},
			Quote_Strings: []Quote_Delimiter{
				{Open: "\"", Close: "\"", Escape: '\\'},
				{Open: "'", Close: "'", Escape: '\\', Character_Like: true},
			},
		},
		Seeded_Language{
			Name:                "Java",
			Test_Infixes:        []string{"Test.", "Tests."},
			Line_Comment:        []string{"//"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Verbatim_Strings: []Verbatim_Delimiter{
				{Open: "\"\"\"", Close: "\"\"\""},
			},
			Quote_Strings: []Quote_Delimiter{
				{Open: "\"", Close: "\"", Escape: '\\'},
				{Open: "'", Close: "'", Escape: '\\', Character_Like: true},
			},
		},
		Seeded_Language{
			Name:          "Python",
			Test_Prefixes: []string{"test_"},
			Test_Infixes:  []string{"_test."},
			Line_Comment:  []string{"#"},
			// The triple quotes precede the single quotes so the scanner takes "\"\"\""
			// whole rather than as an empty string followed by a quote.
			Verbatim_Strings: []Verbatim_Delimiter{
				{Open: "\"\"\"", Close: "\"\"\""},
				{Open: "'''", Close: "'''"},
			},
			Quote_Strings: []Quote_Delimiter{
				{Open: "\"", Close: "\"", Escape: '\\'},
				{Open: "'", Close: "'", Escape: '\\'},
			},
		},
		Seeded_Language{
			Name:                "SCSS",
			Line_Comment:        []string{"//"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Quote_Strings: []Quote_Delimiter{
				{Open: "\"", Close: "\"", Escape: '\\'},
				{Open: "'", Close: "'", Escape: '\\'},
			},
		},
		Seeded_Language{
			Name:                "Scheme",
			Line_Comment:        []string{";"},
			Block_Comment_Open:  "#|",
			Block_Comment_Close: "|#",
			Block_Comment_Nests: true,
			Quote_Strings:       Quote_Delimiters(double_quote()),
		},
		{},
	}
}

func language_seed_bucket_14() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_14.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:                "HTML",
			Block_Comment_Open:  "<!--",
			Block_Comment_Close: "-->",
		},
		{},
		{},
		{},
		{},
		{},
	}
}

func language_seed_bucket_15() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_15.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:                "OCaml",
			Block_Comment_Open:  "(*",
			Block_Comment_Close: "*)",
			Block_Comment_Nests: true,
			Quote_Strings:       Quote_Delimiters(double_quote()),
		},
		Seeded_Language{
			Name:         "YAML",
			Line_Comment: []string{"#"},
			Quote_Strings: []Quote_Delimiter{
				{Open: "\"", Close: "\"", Escape: '\\'},
				{Open: "'", Close: "'", Escape: '\\'},
			},
		},
		{},
		{},
		{},
		{},
	}
}

func language_seed_bucket_16() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_16.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:         "Shell",
			Line_Comment: []string{"#"},
			Heredoc:      true,
			Quote_Strings: []Quote_Delimiter{
				{Open: "\"", Close: "\"", Escape: '\\'},
				{Open: "'", Close: "'"},
			},
		},
		{},
		{},
		{},
		{},
		{},
	}
}

func language_seed_bucket_17() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_17.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:          "Crystal",
			Line_Comment:  []string{"#"},
			Quote_Strings: Quote_Delimiters(double_quote()),
		},
		Seeded_Language{
			Name:                "HCL",
			Line_Comment:        []string{"#", "//"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Quote_Strings:       Quote_Delimiters(double_quote()),
		},
		Seeded_Language{
			Name:                "SQL",
			Line_Comment:        []string{"--"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Quote_Strings: []Quote_Delimiter{
				{Open: "\"", Close: "\"", Escape: '\\'},
				{Open: "'", Close: "'", Escape: '\\'},
			},
		},
		{},
		{},
		{},
	}
}

func language_seed_bucket_18() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_18.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:                "Go",
			Test_Infixes:        []string{"_test."},
			Line_Comment:        []string{"//"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Block_Comment_Nests: false,
			Verbatim_Strings:    []Verbatim_Delimiter{{Open: "`", Close: "`"}},
			Quote_Strings: []Quote_Delimiter{
				{Open: "\"", Close: "\"", Escape: '\\'},
				{Open: "'", Close: "'", Escape: '\\', Character_Like: true},
			},
		},
		Seeded_Language{
			Name:                "Pascal",
			Line_Comment:        []string{"//"},
			Block_Comment_Open:  "{",
			Block_Comment_Close: "}",
			Quote_Strings:       []Quote_Delimiter{{Open: "'", Close: "'"}},
		},
		Seeded_Language{
			Name:                "Verilog",
			Line_Comment:        []string{"//"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Quote_Strings:       Quote_Delimiters(double_quote()),
		},
		{},
		{},
		{},
	}
}

func language_seed_bucket_19() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_19.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:                "Nix",
			Line_Comment:        []string{"#"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Verbatim_Strings:    []Verbatim_Delimiter{{Open: "''", Close: "''"}},
			Quote_Strings:       Quote_Delimiters(double_quote()),
		},
		{},
		{},
		{},
		{},
		{},
	}
}

func language_seed_bucket_20() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_20.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:         "Dockerfile",
			Line_Comment: []string{"#"},
		},
		Seeded_Language{
			Name:                "GLSL",
			Line_Comment:        []string{"//"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Quote_Strings:       Quote_Delimiters(double_quote()),
		},
		Seeded_Language{
			Name:         "Zig",
			Line_Comment: []string{"//"},
			Quote_Strings: []Quote_Delimiter{
				{Open: "\"", Close: "\"", Escape: '\\'},
				{Open: "'", Close: "'", Escape: '\\', Character_Like: true},
			},
		},
		{},
		{},
		{},
	}
}

func language_seed_bucket_21() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_21.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:          "Perl",
			Line_Comment:  []string{"#"},
			Heredoc:       true,
			Quote_Strings: Quote_Delimiters(plain_quotes()),
		},
		Seeded_Language{
			Name:          "Tcl",
			Line_Comment:  []string{"#"},
			Quote_Strings: Quote_Delimiters(double_quote()),
		},
		Seeded_Language{
			Name:                "Thrift",
			Line_Comment:        []string{"//", "#"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Quote_Strings:       Quote_Delimiters(plain_quotes()),
		},
		{},
		{},
		{},
	}
}

func language_seed_bucket_22() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_22.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:          "Fortran",
			Line_Comment:  []string{"!"},
			Quote_Strings: Quote_Delimiters(plain_quotes()),
		},
		Seeded_Language{
			Name:                "PowerShell",
			Line_Comment:        []string{"#"},
			Block_Comment_Open:  "<#",
			Block_Comment_Close: "#>",
			Quote_Strings:       Quote_Delimiters(plain_quotes()),
		},
		{},
		{},
		{},
		{},
	}
}

func language_seed_bucket_23() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_23.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:                "JavaScript",
			Test_Infixes:        []string{".test.", ".spec."},
			Line_Comment:        []string{"//"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Block_Comment_Nests: false,
			Verbatim_Strings:    []Verbatim_Delimiter{{Open: "`", Close: "`"}},
			Quote_Strings: []Quote_Delimiter{
				{Open: "\"", Close: "\"", Escape: '\\'},
				{Open: "'", Close: "'", Escape: '\\'},
			},
		},
		Seeded_Language{
			Name:                "Odin",
			Line_Comment:        []string{"//"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Block_Comment_Nests: true,
			Verbatim_Strings:    []Verbatim_Delimiter{{Open: "`", Close: "`"}},
			Quote_Strings: []Quote_Delimiter{
				{Open: "\"", Close: "\"", Escape: '\\'},
				{Open: "'", Close: "'", Escape: '\\', Character_Like: true},
			},
		},
		Seeded_Language{
			Name:                "XML",
			Block_Comment_Open:  "<!--",
			Block_Comment_Close: "-->",
		},
		{},
		{},
		{},
	}
}

func language_seed_bucket_24() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_24.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:                "Markdown",
			Block_Comment_Open:  "<!--",
			Block_Comment_Close: "-->",
		},
		Seeded_Language{
			Name:                "PHP",
			Test_Infixes:        []string{"Test."},
			Line_Comment:        []string{"//", "#"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Quote_Strings:       Quote_Delimiters(plain_quotes()),
		},
		Seeded_Language{
			Name:          "R",
			Line_Comment:  []string{"#"},
			Quote_Strings: Quote_Delimiters(plain_quotes()),
		},
		{},
		{},
		{},
	}
}

func language_seed_bucket_25() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_25.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:                "F#",
			Line_Comment:        []string{"//"},
			Block_Comment_Open:  "(*",
			Block_Comment_Close: "*)",
			Block_Comment_Nests: true,
			Verbatim_Strings: []Verbatim_Delimiter{
				{Open: "\"\"\"", Close: "\"\"\""},
			},
			Quote_Strings: Quote_Delimiters(double_quote()),
		},
		Seeded_Language{
			Name:          "Fish",
			Line_Comment:  []string{"#"},
			Quote_Strings: Quote_Delimiters(plain_quotes()),
		},
		Seeded_Language{
			Name:                "Julia",
			Line_Comment:        []string{"#"},
			Block_Comment_Open:  "#=",
			Block_Comment_Close: "=#",
			Block_Comment_Nests: true,
			Verbatim_Strings: []Verbatim_Delimiter{
				{Open: "\"\"\"", Close: "\"\"\""},
			},
			Quote_Strings: Quote_Delimiters(double_quote()),
		},
		{},
		{},
		{},
	}
}

func language_seed_bucket_26() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_26.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:                "Common Lisp",
			Line_Comment:        []string{";"},
			Block_Comment_Open:  "#|",
			Block_Comment_Close: "|#",
			Block_Comment_Nests: true,
			Quote_Strings:       Quote_Delimiters(double_quote()),
		},
		Seeded_Language{
			Name:                "HLSL",
			Line_Comment:        []string{"//"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Quote_Strings:       Quote_Delimiters(double_quote()),
		},
		Seeded_Language{
			Name:                "Protobuf",
			Line_Comment:        []string{"//"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Quote_Strings:       Quote_Delimiters(plain_quotes()),
		},
		{},
		{},
		{},
	}
}

func language_seed_bucket_27() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_27.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:                "Objective-C",
			Line_Comment:        []string{"//"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Quote_Strings:       Quote_Delimiters(c_family_quotes()),
		},
		Seeded_Language{
			Name:                "Scala",
			Test_Infixes:        []string{"Test.", "Tests.", "Spec."},
			Line_Comment:        []string{"//"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Block_Comment_Nests: true,
			Verbatim_Strings: []Verbatim_Delimiter{
				{Open: "\"\"\"", Close: "\"\"\""},
			},
			Quote_Strings: []Quote_Delimiter{
				{Open: "\"", Close: "\"", Escape: '\\'},
				{Open: "'", Close: "'", Escape: '\\', Character_Like: true},
			},
		},
		{},
		{},
		{},
		{},
	}
}

func language_seed_bucket_28() (seeds Language_Seeds) {
	defer func() { Language_Seeds_Invariants(seeds, "language_seed_bucket_28.seeds") }()
	return Language_Seeds{
		Seeded_Language{
			Name:                "Arduino",
			Line_Comment:        []string{"//"},
			Block_Comment_Open:  "/*",
			Block_Comment_Close: "*/",
			Quote_Strings:       Quote_Delimiters(c_family_quotes()),
		},
		Seeded_Language{
			Name:         "Makefile",
			Line_Comment: []string{"#"},
		},
		{},
		{},
		{},
		{},
	}
}

// Known_Name_Consumer receives a matched static seed name.
type Known_Name_Consumer func(name Known_Name)

// Language_For_Extension returns the seeded language for a file extension, with the
// leading dot, and whether one matched.
func Language_For_Extension(extension Extension) (language Language, recognized Recognition) {
	defer func() {
		Language_Invariants(language, "Language_For_Extension.language")
		Recognition_Invariants(recognized, "Language_For_Extension.recognized")
	}()
	Extension_Invariants(extension, "Language_For_Extension.extension")
	switch extension {
	case ".go":
		return Language(language_seed("Go")), true
	case ".rs":
		return Language(language_seed("Rust")), true
	case ".py":
		return Language(language_seed("Python")), true
	case ".js", ".jsx", ".mjs", ".cjs":
		return Language(language_seed("JavaScript")), true
	case ".ts", ".tsx":
		return Language(language_seed("TypeScript")), true
	case ".lua":
		return Language(language_seed("Lua")), true
	case ".odin":
		return Language(language_seed("Odin")), true
	case ".zig":
		return Language(language_seed("Zig")), true
	case ".c", ".h":
		return Language(language_seed("C")), true
	case ".cpp", ".cc", ".cxx", ".hpp", ".hh", ".hxx":
		return Language(language_seed("C++")), true
	case ".cs":
		return Language(language_seed("C#")), true
	case ".java":
		return Language(language_seed("Java")), true
	case ".swift":
		return Language(language_seed("Swift")), true
	case ".kt", ".kts":
		return Language(language_seed("Kotlin")), true
	case ".scala", ".sc":
		return Language(language_seed("Scala")), true
	case ".sh", ".bash", ".zsh":
		return Language(language_seed("Shell")), true
	case ".rb":
		return Language(language_seed("Ruby")), true
	case ".yaml", ".yml":
		return Language(language_seed("YAML")), true
	case ".toml":
		return Language(language_seed("TOML")), true
	case ".sql":
		return Language(language_seed("SQL")), true
	case ".mk":
		return Language(language_seed("Makefile")), true
	case ".dockerfile":
		return Language(language_seed("Dockerfile")), true
	case ".html", ".htm":
		return Language(language_seed("HTML")), true
	case ".xml", ".svg":
		return Language(language_seed("XML")), true
	case ".css":
		return Language(language_seed("CSS")), true
	case ".scss":
		return Language(language_seed("SCSS")), true
	case ".less":
		return Language(language_seed("LESS")), true
	}
	name := Known_Name("")
	recognized = extension_match_more(extension, func(match Known_Name) { name = match })
	if !recognized {
		return Language{}, false
	}
	return Language(language_seed(name)), true
}

// Continues Language_For_Extension's lookup for the C-style and markup additions.
func extension_match_more(
	extension Extension, consume Known_Name_Consumer,
) (recognized Recognition) {
	defer func() { Recognition_Invariants(recognized, "extension_match_more.recognized") }()
	Extension_Invariants(extension, "extension_match_more.extension")
	switch extension {
	case ".m", ".mm":
		consume("Objective-C")
	case ".dart":
		consume("Dart")
	case ".php", ".phtml":
		consume("PHP")
	case ".sol":
		consume("Solidity")
	case ".groovy", ".gradle":
		consume("Groovy")
	case ".v", ".sv", ".svh":
		consume("Verilog")
	case ".glsl", ".vert", ".frag", ".comp", ".geom":
		consume("GLSL")
	case ".hlsl":
		consume("HLSL")
	case ".ino":
		consume("Arduino")
	case ".proto":
		consume("Protobuf")
	case ".thrift":
		consume("Thrift")
	case ".jsonc", ".json5":
		consume("JSONC")
	case ".tf", ".hcl", ".tfvars":
		consume("HCL")
	case ".nix":
		consume("Nix")
	case ".md", ".markdown":
		consume("Markdown")
	case ".vue":
		consume("Vue")
	case ".svelte":
		consume("Svelte")
	case ".astro":
		consume("Astro")
	case ".xaml":
		consume("XAML")
	case ".xsl", ".xslt":
		consume("XSLT")
	default:
		return extension_match_rest(extension, consume)
	}
	return true
}

// Continues Language_For_Extension's lookup for the remaining languages.
func extension_match_rest(
	extension Extension, consume Known_Name_Consumer,
) (recognized Recognition) {
	defer func() { Recognition_Invariants(recognized, "extension_match_rest.recognized") }()
	Extension_Invariants(extension, "extension_match_rest.extension")
	switch extension {
	case ".hs", ".lhs":
		consume("Haskell")
	case ".ml", ".mli":
		consume("OCaml")
	case ".fs", ".fsx", ".fsi":
		consume("F#")
	case ".jl":
		consume("Julia")
	case ".nim", ".nims":
		consume("Nim")
	case ".lisp", ".lsp", ".cl":
		consume("Common Lisp")
	case ".scm", ".ss":
		consume("Scheme")
	case ".rkt":
		consume("Racket")
	case ".clj", ".cljs", ".cljc", ".edn":
		consume("Clojure")
	case ".el":
		consume("Emacs Lisp")
	case ".erl", ".hrl":
		consume("Erlang")
	case ".f90", ".f95", ".f03", ".f08", ".f", ".for":
		consume("Fortran")
	case ".adb", ".ads", ".ada":
		consume("Ada")
	case ".d":
		consume("D")
	case ".pas", ".pp", ".dpr":
		consume("Pascal")
	case ".r", ".R":
		consume("R")
	case ".ex", ".exs":
		consume("Elixir")
	case ".cr":
		consume("Crystal")
	case ".ps1", ".psm1", ".psd1":
		consume("PowerShell")
	case ".fish":
		consume("Fish")
	case ".nu":
		consume("Nushell")
	case ".cmake":
		consume("CMake")
	case ".tcl":
		consume("Tcl")
	case ".pl", ".pm", ".t", ".pod":
		consume("Perl")
	case ".tex", ".sty", ".cls", ".ltx":
		consume("TeX")
	case ".vb":
		consume("Visual Basic")
	default:
		return false
	}
	return true
}

// Matches an extensionless filename to its static seed name.
func language_name_for_filename(
	name File_Name, consume Known_Name_Consumer,
) (recognized Recognition) {
	defer func() {
		Recognition_Invariants(recognized, "Language_For_Filename.recognized")
	}()
	File_Name_Invariants(name, "Language_For_Filename.name")
	switch name {
	case "Makefile", "makefile", "GNUmakefile":
		consume("Makefile")
	case "Dockerfile":
		consume("Dockerfile")
	case "CMakeLists.txt":
		consume("CMake")
	default:
		return false
	}
	return true
}

// Resolves the language for a path by its extension, or for an extensionless file by
// its name, and whether one matched.
func language_for_path(file_path File_Path) (language Language, recognized Recognition) {
	defer func() {
		Language_Invariants(language, "language_for_path.language")
		Recognition_Invariants(recognized, "language_for_path.recognized")
	}()
	File_Path_Invariants(file_path, "language_for_path.file_path")
	language, recognized = Language_For_Extension(Extension(path.Ext(string(file_path))))
	if recognized {
		return language, true
	}
	name := Known_Name("")
	recognized = language_name_for_filename(
		File_Name(path.Base(string(file_path))), func(match Known_Name) {
			name = match
		})
	if !recognized {
		return Language{}, false
	}
	return Language(language_seed(name)), true
}

// EXTENSION_BYTES_MIN is the empty extension of a file whose name carries no dot.
const EXTENSION_BYTES_MIN = 0

// EXTENSION_BYTES_MAX is a whole base name: an extension is the suffix from the last
// dot of a name the walk found, so it is bounded by what the filesystem holds and not
// by what the language table happens to recognize. A name that is one long dotted
// suffix arrives here entire.
const EXTENSION_BYTES_MAX = FILE_NAME_BYTES_MAX

// Extension is a file extension with its leading dot, or empty when the name carries
// no dot. A bare "." is reachable too, from a name ending in a dot.
type Extension string

// Extension_Invariants bounds an extension's byte length.
func Extension_Invariants(extension Extension, namespace invariant.Namespace) {
	invariant.Tree(extension, namespace).
		Range_Int(len(extension), EXTENSION_BYTES_MIN, EXTENSION_BYTES_MAX).
		Ensure()
}

// FILE_NAME_BYTES_MIN is the one-byte base name, the shortest a path element can be.
const FILE_NAME_BYTES_MIN = 1

// FILE_NAME_BYTES_MAX is the whole path bound, not a per-element one: sloc never
// splits a path against a host's own name limit, so a path with no separator at all
// arrives here entire.
const FILE_NAME_BYTES_MAX = FILE_PATH_BYTES_MAX

// File_Name is the final element of a path, the part a filename lookup matches on.
type File_Name string

// File_Name_Invariants bounds a base name's byte length.
func File_Name_Invariants(name File_Name, namespace invariant.Namespace) {
	invariant.Tree(name, namespace).
		Range_Int(len(name), FILE_NAME_BYTES_MIN, FILE_NAME_BYTES_MAX).
		Ensure()
}

// FILE_PATH_BYTES_MIN is the one-byte path, a file named by a single character.
const FILE_PATH_BYTES_MIN = 1

// FILE_PATH_BYTES_MAX is the longest path the host file systems accept.
const FILE_PATH_BYTES_MAX = 4096

// File_Path is a counted file's path, relative to the tree it was walked from. It is
// distinct from Root because a root may be the one-byte "." that names no file.
type File_Path string

// File_Path_Invariants bounds a file path's byte length.
func File_Path_Invariants(file_path File_Path, namespace invariant.Namespace) {
	invariant.Tree(file_path, namespace).
		Range_Int(len(file_path), FILE_PATH_BYTES_MIN, FILE_PATH_BYTES_MAX).
		Ensure()
}

// Code_Count is the number of code lines in one partition.
type Code_Count Line_Count

// Code_Count_Invariants bounds a code-line count.
func Code_Count_Invariants(count Code_Count, namespace invariant.Namespace) {
	invariant.Tree(count, namespace).
		Range_Int(int(count), LINE_COUNT_MIN, LINE_COUNT_MAX).
		Ensure()
}

// Comment_Count is the number of comment-only lines in one partition.
type Comment_Count Line_Count

// Comment_Count_Invariants bounds a comment-only line count.
func Comment_Count_Invariants(count Comment_Count, namespace invariant.Namespace) {
	invariant.Tree(count, namespace).
		Range_Int(int(count), LINE_COUNT_MIN, LINE_COUNT_MAX).
		Ensure()
}

// Blank_Count is the number of blank lines in one partition.
type Blank_Count Line_Count

// Blank_Count_Invariants bounds a blank-line count.
func Blank_Count_Invariants(count Blank_Count, namespace invariant.Namespace) {
	invariant.Tree(count, namespace).
		Range_Int(int(count), LINE_COUNT_MIN, LINE_COUNT_MAX).
		Ensure()
}

// Counts is the line partition of a file or a group of files.
type Counts struct {
	// Code is the number of lines bearing code.
	Code Code_Count
	// Comment is the number of comment-only lines.
	Comment Comment_Count
	// Blank is the number of empty or whitespace-only lines.
	Blank Blank_Count
	// Dropped is how many lines were wider than the scan window and so were only
	// partly read. It is not a fourth partition — such a line is still counted as
	// code, comment, or blank — but a classification made from a prefix, which the
	// report states rather than hides.
	Dropped Dropped_Count
}

// Counts_Invariants states each partition of the line count and the lines read short.
func Counts_Invariants(counts Counts, namespace invariant.Namespace) {
	Code_Count_Invariants(counts.Code, namespace)
	Comment_Count_Invariants(counts.Comment, namespace)
	Blank_Count_Invariants(counts.Blank, namespace)
	Dropped_Count_Invariants(counts.Dropped, namespace)
}

// FILE_CONTENT_COUNT_MAX is the most code or comment lines one file can contain. Each
// needs one content byte and, except for the final line, one separator byte.
const FILE_CONTENT_COUNT_MAX = (SOURCE_BYTES_MAX + 1) / 2

// FILE_BLANK_COUNT_MAX is the most blank lines one file can contain. Each newline can
// terminate an empty line, so every source byte can contribute one.
const FILE_BLANK_COUNT_MAX = SOURCE_BYTES_MAX

// FILE_DROPPED_COUNT_MAX is the most lines one file can read short. Each needs one
// byte beyond the scan window and, except for the final line, one separator byte.
const FILE_DROPPED_COUNT_MAX = (SOURCE_BYTES_MAX + 1) / (LINE_BYTES_MAX + 2)

// File_Partition is the line partition produced from one source file. Its byte bound
// makes aggregate report ceilings unreachable at this stage.
type File_Partition Counts

// File_Partition_Invariants checks one file's production range without assigning the
// report's aggregate boundary witnesses to the byte classifier.
func File_Partition_Invariants(
	counts File_Partition, namespace invariant.Namespace,
) {
	invariant.Tree(counts, namespace).
		Range_Int(int(counts.Code), LINE_COUNT_MIN, FILE_CONTENT_COUNT_MAX).
		Range_Int(int(counts.Comment), LINE_COUNT_MIN, FILE_CONTENT_COUNT_MAX).
		Range_Int(int(counts.Blank), LINE_COUNT_MIN, FILE_BLANK_COUNT_MAX).
		Range_Int(
			int(counts.Dropped), DROPPED_COUNT_MIN, FILE_DROPPED_COUNT_MAX).
		Ensure()
	invariant.Always(
		int(counts.Code)+int(counts.Comment)+int(counts.Blank) <= SOURCE_BYTES_MAX,
		"A file line partition always fits its source byte bound.")
}

// DROPPED_COUNT_MIN is a report in which every line fit the scan window.
const DROPPED_COUNT_MIN = 0

// DROPPED_COUNT_MAX is where the tally of lines read short stops counting. It is far
// below the line bound on purpose: a dropped line is by definition a wide one, so
// witnessing this tally costs bytes rather than lines, and a report that has already
// dropped this many has made the point.
const DROPPED_COUNT_MAX = 100000

// DROPPED_SATURATED_TEXT is what the table prints once the tally saturates. It spells
// DROPPED_COUNT_MAX, so the two move together.
const DROPPED_SATURATED_TEXT = "100k+"

// Dropped_Count is how many lines were read short of their full width. It saturates
// rather than growing with the report, so it is its own type and not a line count.
type Dropped_Count int

// Dropped_Count_Invariants bounds the tally of lines read short.
func Dropped_Count_Invariants(count Dropped_Count, namespace invariant.Namespace) {
	invariant.Tree(count, namespace).
		Range_Int(int(count), DROPPED_COUNT_MIN, DROPPED_COUNT_MAX).
		Ensure()
}

// LINE_COUNT_MIN is the empty partition: a file, or a report, with no such line.
const LINE_COUNT_MIN = 0

// LINE_COUNT_MAX bounds the lines one run counts; Count refuses a tree whose total
// would exceed it. Every bound here is chosen to be *witnessable*: a boundary that no
// test can reach in reasonable time is a claim, not an invariant, so the ceilings are
// set where the suite can actually drive them rather than where the type could.
const LINE_COUNT_MAX = 99999999

// Line_Count is a number of physical lines, per file or summed across a report.
type Line_Count int

// Line_Count_Invariants bounds a line count.
func Line_Count_Invariants(count Line_Count, namespace invariant.Namespace) {
	invariant.Tree(count, namespace).
		Range_Int(int(count), LINE_COUNT_MIN, LINE_COUNT_MAX).
		Ensure()
}

// Returns the total physical line count: code, comment, and blank lines sum to it
// because every line is counted exactly once.
func counts_lines(counts Counts) (line_count Line_Count) {
	defer func() { Line_Count_Invariants(line_count, "counts_lines.line_count") }()
	Counts_Invariants(counts, "counts_lines.counts")
	return Line_Count(counts.Code) + Line_Count(counts.Comment) + Line_Count(counts.Blank)
}

// File_Classifier is the classification capability selected by the composition root.
// The concrete value keeps simulations on Main while still letting them replace only
// derivations whose production-reachable result they establish.
type File_Classifier struct {
	// Kind selects byte classification or a path-keyed classification model.
	Kind File_Classifier_Kind
	// Classifications holds the complete modeled results for nonempty modeled files.
	Classifications File_Classifications
}

// File_Classifier_Invariants states the selected implementation and model size.
func File_Classifier_Invariants(classifier File_Classifier, namespace invariant.Namespace) {
	File_Classifier_Kind_Invariants(classifier.Kind, namespace)
	File_Classifications_Invariants(
		classifier.Classifications, namespace)
}

// File_Classifier_Kind selects the concrete classification implementation Main uses.
type File_Classifier_Kind int

// File_Classifier_Kind_Invariants holds a selection to the two classifier
// implementations, and witnesses each one.
func File_Classifier_Kind_Invariants(
	kind File_Classifier_Kind, namespace invariant.Namespace,
) {
	invariant.Tree(kind, namespace).
		Enum_Int(
			int(kind),
			int(FILE_CLASSIFIER_KIND_BYTES),
			int(FILE_CLASSIFIER_KIND_MODEL)).
		Ensure()
}

// FILE_CLASSIFIER_KIND_BYTES selects the production byte scanner. One rather than
// zero keeps an omitted File_Classifier invalid instead of creating an ambient fallback.
const FILE_CLASSIFIER_KIND_BYTES File_Classifier_Kind = 1

// FILE_CLASSIFIER_KIND_MODEL selects exact path-keyed modeled results.
const FILE_CLASSIFIER_KIND_MODEL File_Classifier_Kind = 2

// File_Classifications are exact classifications keyed by recognized file path.
type File_Classifications map[Classified_Path]File_Partition

// File_Classifications_Invariants bounds a model to the widest boundary witness.
func File_Classifications_Invariants(
	classifications File_Classifications, namespace invariant.Namespace,
) {
	invariant.Tree(classifications, namespace).
		Range_Int(
			len(classifications),
			FILE_CLASSIFICATIONS_COUNT_MIN,
			FILE_CLASSIFICATIONS_COUNT_MAX).
		Ensure()
}

// FILE_CLASSIFICATIONS_COUNT_MIN is the absent model of the byte classifier.
const FILE_CLASSIFICATIONS_COUNT_MIN = 0

// FILE_CLASSIFICATIONS_COUNT_MAX is the two-root line-bound witness: one root fills
// its candidate bound while the other contributes 72 classifications needed to make
// their omitted-file tallies reach the aggregate bound.
const FILE_CLASSIFICATIONS_COUNT_MAX = 65607

// The consumer validates both sides of the injected boundary because a modeled
// implementation must obey the same domain and range as the byte implementation.
func classify(
	classifier File_Classifier, input Classify_File_Input,
) (counts File_Partition) {
	defer func() { File_Partition_Invariants(counts, "classify.counts") }()
	File_Classifier_Invariants(classifier, "classify.classifier")
	Classify_File_Input_Invariants(input, "classify.input")
	switch classifier.Kind {
	case FILE_CLASSIFIER_KIND_BYTES:
		if len(classifier.Classifications) != 0 {
			panic("byte classifier carries modeled classifications")
		}
		counts = File_Partition{}
		classify_bytes(input, func(kind Line_Kind, dropped Dropped_Count) {
			switch kind {
			case LINE_KIND_CODE:
				counts.Code++
			case LINE_KIND_COMMENT:
				counts.Comment++
			case LINE_KIND_BLANK:
				counts.Blank++
			}
			counts.Dropped += dropped
			if counts.Dropped > FILE_DROPPED_COUNT_MAX {
				counts.Dropped = FILE_DROPPED_COUNT_MAX
			}
		})
		return counts
	case FILE_CLASSIFIER_KIND_MODEL:
		return classify_model(classifier.Classifications, input)
	}
	panic("unknown file classifier kind")
}

// A model must be total for every nonempty file Main sends it. Empty files need no
// stored derivation because their only possible partition is the zero value.
func classify_model(
	classifications File_Classifications, input Classify_File_Input,
) (counts File_Partition) {
	defer func() { File_Partition_Invariants(counts, "classify_model.counts") }()
	File_Classifications_Invariants(classifications, "classify_model.classifications")
	Classify_File_Input_Invariants(input, "classify_model.input")
	counts, modeled := classifications[input.Path]
	if modeled {
		return counts
	}
	if len(input.Source) == 0 {
		return File_Partition{}
	}
	panic("classification model missing a nonempty file")
}

// CLASSIFIED_PATH_BYTES_MIN is the shortest recognized path, a bare extension such as .c.
const CLASSIFIED_PATH_BYTES_MIN = 2

// CLASSIFIED_PATH_BYTES_MAX is the host filesystem's path bound.
const CLASSIFIED_PATH_BYTES_MAX = FILE_PATH_BYTES_MAX

// Classified_Path is the identity of a file whose language was recognized.
type Classified_Path string

// Classified_Path_Invariants keeps the classifier's domain at recognized path widths.
func Classified_Path_Invariants(file_path Classified_Path, namespace invariant.Namespace) {
	invariant.Tree(file_path, namespace).
		Range_Int(len(file_path), CLASSIFIED_PATH_BYTES_MIN, CLASSIFIED_PATH_BYTES_MAX).
		Ensure()
}

// Classify_File_Input is one file's identity, bytes, and language.
type Classify_File_Input struct {
	// Path is the identity of the file whose source is classified.
	Path Classified_Path
	// Source is the file's bytes.
	Source Source
	// Language is the language to read the source as.
	Language Seeded_Language
}

// Classify_File_Input_Invariants states the bytes to read and the language to read
// them as.
func Classify_File_Input_Invariants(
	input Classify_File_Input, namespace invariant.Namespace,
) {
	Classified_Path_Invariants(input.Path, namespace)
	Source_Invariants(input.Source, namespace)
	Seeded_Language_Invariants(input.Language, namespace)
}

// SOURCE_BYTES_MIN is the empty file.
const SOURCE_BYTES_MIN = 0

// SOURCE_BYTES_MAX bounds one file's bytes. A larger file is left out of the report
// the way a binary one is, rather than counted against an unbounded buffer. It is the
// line bound exactly: a byte can begin at most one line, so a file within this bound
// can never on its own carry more lines than a whole run may count.
const SOURCE_BYTES_MAX = 1 << 22

// Source is one file's bytes, untrusted and bounded.
type Source []byte

// Source_Invariants bounds a file's byte count.
func Source_Invariants(source Source, namespace invariant.Namespace) {
	invariant.Tree(source, namespace).
		Range_Int(len(source), SOURCE_BYTES_MIN, SOURCE_BYTES_MAX).
		Ensure()
}

// Classified_Line_Consumer receives one physical line's partition and whether its
// classification read only the bounded prefix.
type Classified_Line_Consumer func(kind Line_Kind, dropped Dropped_Count)

// Classifies every physical line of the source without assigning the aggregate
// partition domain to the byte scanner.
func classify_bytes(input Classify_File_Input, consume Classified_Line_Consumer) {
	Classify_File_Input_Invariants(input, "Classify_File.input")
	prepared := language_scanner(&input.Language)
	carry := Scan_Carry{
		Block_Comment_Depth: 0,
		Raw_String_Close:    "",
		Comment_Close:       "",
		Heredoc_Terminator:  "",
	}
	source := input.Source
	// Lines are walked in place rather than materialized into a slice: one classifier
	// pass over millions of lines should not also allocate a slice header per line.
	start := 0
	var kind Line_Kind
	for index := 0; index < len(source); index++ {
		if source[index] != '\n' {
			continue
		}
		// A line wider than the scan window is truncated for reading but still counts
		// as exactly one line, so the totals stay exact for any input while the
		// scanner works against a bounded buffer.
		stop_count := index
		dropped := Dropped_Count(0)
		if stop_count-start > LINE_BYTES_MAX {
			stop_count = start + LINE_BYTES_MAX
			dropped = 1
		}
		kind, carry = classify_line(Line(source[start:stop_count]), carry, &prepared)
		consume(kind, dropped)
		start = index + 1
	}
	// Bytes after the last newline are a final line only when non-empty, so a trailing
	// newline adds no phantom line and the empty file is zero lines.
	if start < len(source) {
		stop_count := len(source)
		dropped := Dropped_Count(0)
		if stop_count-start > LINE_BYTES_MAX {
			stop_count = start + LINE_BYTES_MAX
			dropped = 1
		}
		kind, _ = classify_line(Line(source[start:stop_count]), carry, &prepared)
		consume(kind, dropped)
	}
}

// The partition a single physical line falls into.
type Line_Kind int

// Line_Kind_Invariants holds a partition to the three a line can fall into, and
// witnesses each one.
func Line_Kind_Invariants(kind Line_Kind, namespace invariant.Namespace) {
	invariant.Tree(kind, namespace).
		Enum_3_Int(
			int(kind), int(LINE_KIND_BLANK), int(LINE_KIND_CODE),
			int(LINE_KIND_COMMENT)).
		Ensure()
}

// LINE_KIND_BLANK tags a line that is empty or only whitespace.
const LINE_KIND_BLANK Line_Kind = 0

// LINE_KIND_CODE tags a line carrying at least one code token, comments aside.
const LINE_KIND_CODE Line_Kind = 1

// LINE_KIND_COMMENT tags a line whose only content is a comment.
const LINE_KIND_COMMENT Line_Kind = 2

// LINE_BYTES_MIN is the empty line, the span between two adjacent newlines.
const LINE_BYTES_MIN = 0

// LINE_BYTES_MAX is the scan window one line is read through. It bounds the scanner's
// working set against a minified or generated line of arbitrary width.
const LINE_BYTES_MAX = 256

// Line is one physical line of a source file, truncated to the scan window.
type Line []byte

// Line_Invariants bounds a line's byte count.
func Line_Invariants(line Line, namespace invariant.Namespace) {
	invariant.Tree(line, namespace).
		Range_Int(len(line), LINE_BYTES_MIN, LINE_BYTES_MAX).
		Ensure()
}

// SCAN_LINE_BYTES_MIN is the one-byte line: the scanner is only ever handed a line
// with content, since a blank one is decided before any scan begins.
const SCAN_LINE_BYTES_MIN = 1

// SCAN_LINE_BYTES_MAX is the scan window, as for any line.
const SCAN_LINE_BYTES_MAX = LINE_BYTES_MAX

// Scan_Line is a non-blank line the scanner walks. It is distinct from Line because
// the blank verdict is reached before a scan is built, so the scanner never sees the
// empty line that Line spans.
type Scan_Line []byte

// Scan_Line_Invariants bounds a scanned line's byte count.
func Scan_Line_Invariants(line Scan_Line, namespace invariant.Namespace) {
	invariant.Tree(line, namespace).
		Range_Int(len(line), SCAN_LINE_BYTES_MIN, SCAN_LINE_BYTES_MAX).
		Ensure()
}

// CURSOR_MIN is the start of a line, before any byte is read.
const CURSOR_MIN = 0

// CURSOR_MAX is the end of the widest line, where a completed scan comes to rest.
const CURSOR_MAX = LINE_BYTES_MAX

// Cursor is a byte offset into the line being scanned.
type Cursor int

// Cursor_Invariants bounds a cursor to the line it walks.
func Cursor_Invariants(cursor Cursor, namespace invariant.Namespace) {
	invariant.Tree(cursor, namespace).
		Range_Int(int(cursor), CURSOR_MIN, CURSOR_MAX).
		Ensure()
}

// Scan_Position is the line being read and how far along it has reached.
type Scan_Position struct {
	// Line is the non-blank line being read, truncated to the scan window.
	Line Scan_Line
	// Cursor is the byte offset the scan has reached within the line.
	Cursor Cursor
}

// Scan_Position_Invariants states the line and the offset reached within it.
func Scan_Position_Invariants(at Scan_Position, namespace invariant.Namespace) {
	Scan_Line_Invariants(at.Line, namespace)
	Cursor_Invariants(at.Cursor, namespace)
}

// Bounded_Position is a scanner cursor that stays in one nonempty line.
type Bounded_Position Scan_Position

// Bounded_Position_Invariants states the safety properties that all scanner probes
// share. The complete scan owns the variable cursor and line-width boundaries.
func Bounded_Position_Invariants(at Bounded_Position, namespace invariant.Namespace) {
	invariant.Tree(at, namespace).
		Range_Int(len(at.Line), SCAN_LINE_BYTES_MIN, SCAN_LINE_BYTES_MAX).
		Range_Int(int(at.Cursor), CURSOR_MIN, CURSOR_MAX).
		Ensure()
	invariant.Always(
		len(at.Line) >= SCAN_LINE_BYTES_MIN,
		"A bounded scanner position always has line data.")
	invariant.Always(
		int(at.Cursor) <= len(at.Line),
		"A bounded scanner position always stays in the line.")
}

// ACTIVE_CURSOR_MAX is the last unread byte of the widest scanned line.
const ACTIVE_CURSOR_MAX = CURSOR_MAX - 1

// Active_Position is a scanner cursor that points at unread data.
type Active_Position Scan_Position

// Active_Position_Invariants shares the line bound and excludes its terminal cursor.
func Active_Position_Invariants(at Active_Position, namespace invariant.Namespace) {
	invariant.Tree(at, namespace).
		Range_Int(len(at.Line), SCAN_LINE_BYTES_MIN, SCAN_LINE_BYTES_MAX).
		Range_Int(int(at.Cursor), CURSOR_MIN, ACTIVE_CURSOR_MAX).
		Ensure()
	invariant.Always(
		int(at.Cursor) < len(at.Line),
		"An active scanner position always points at unread data.")
}

// NESTING_DEPTH_MIN is the depth outside any block comment.
const NESTING_DEPTH_MIN = 0

// NESTING_DEPTH_MAX bounds how deep nested block comments are tracked. Past it the
// depth saturates: a comment nested deeper than this closes early, which no real
// source reaches and which keeps the carried depth bounded across a whole file.
const NESTING_DEPTH_MAX = 255

// Nesting_Depth is how many block comments are open at once.
type Nesting_Depth int

// Nesting_Depth_Invariants bounds the open block-comment depth.
func Nesting_Depth_Invariants(depth Nesting_Depth, namespace invariant.Namespace) {
	invariant.Tree(depth, namespace).
		Range_Int(int(depth), NESTING_DEPTH_MIN, NESTING_DEPTH_MAX).
		Ensure()
}

// ACTIVE_NESTING_DEPTH_MIN is the first depth inside a block comment.
const ACTIVE_NESTING_DEPTH_MIN = NESTING_DEPTH_MIN + 1

// Active_Nesting_Depth is the nonzero depth of an open block comment.
type Active_Nesting_Depth Nesting_Depth

// Active_Nesting_Depth_Invariants shares the depth ceiling and excludes zero.
func Active_Nesting_Depth_Invariants(
	depth Active_Nesting_Depth, namespace invariant.Namespace,
) {
	invariant.Tree(depth, namespace).
		Range_Int(int(depth), ACTIVE_NESTING_DEPTH_MIN, NESTING_DEPTH_MAX).
		Ensure()
}

// CLOSER_BYTES_MIN is the empty closer carried when nothing is open.
const CLOSER_BYTES_MIN = 0

// CLOSER_BYTES_MAX is the widest computed terminator: a long bracket's "]", its run of
// equals signs bounded by the hash bound, and its "]".
const CLOSER_BYTES_MAX = HASH_COUNT_MAX + 2

// HASH_COUNT_MAX bounds the hashes a Rust raw string or the equals signs a Lua long
// bracket may carry. Rust itself stops at 255, so the bound costs no real source.
const HASH_COUNT_MAX = 63

// Comment_Closer is the terminator an open long-bracket comment needs, or empty when
// none is open. It is distinct from Closer because only a long bracket ever computes
// it, and a bracket's terminator is never a single byte.
type Comment_Closer string

// Comment_Closer_Invariants bounds a carried long-bracket terminator, carving the width
// a bracket cannot produce.
func Comment_Closer_Invariants(closer Comment_Closer, namespace invariant.Namespace) {
	invariant.Tree(closer, namespace).
		Range_Holed_Int(
			len(closer), BRACKET_CLOSER_BYTES_MIN, BRACKET_CLOSER_BYTES_MAX,
			BRACKET_CLOSER_BYTES_ABSENT, BRACKET_CLOSER_BYTES_ABSENT,
			BRACKET_CLOSER_BYTES_ABSENT, BRACKET_CLOSER_BYTES_ABSENT).
		Ensure()
}

// ACTIVE_COMMENT_CLOSER_BYTES_MIN is the bare long-bracket closer, "]]".
const ACTIVE_COMMENT_CLOSER_BYTES_MIN = BRACKET_CLOSER_BYTES_ABSENT + 1

// Active_Comment_Closer is the nonempty terminator of a long-bracket comment.
type Active_Comment_Closer Comment_Closer

// Active_Comment_Closer_Invariants shares the bracket ceiling and excludes zero.
func Active_Comment_Closer_Invariants(
	closer Active_Comment_Closer, namespace invariant.Namespace,
) {
	invariant.Tree(closer, namespace).
		Range_Int(
			len(closer), ACTIVE_COMMENT_CLOSER_BYTES_MIN, BRACKET_CLOSER_BYTES_MAX).
		Ensure()
}

// Closer is the terminator an open verbatim string needs, or empty when none is open.
type Closer string

// Closer_Invariants bounds a carried terminator's byte length.
func Closer_Invariants(closer Closer, namespace invariant.Namespace) {
	invariant.Tree(closer, namespace).
		Range_Int(len(closer), CLOSER_BYTES_MIN, CLOSER_BYTES_MAX).
		Ensure()
}

// VERBATIM_TERMINATOR_BYTES_MAX is one quote after the bounded Rust hash run.
const VERBATIM_TERMINATOR_BYTES_MAX = HASH_COUNT_MAX + 1

// Verbatim_Terminator is the terminator selected by a verbatim string opener.
type Verbatim_Terminator string

// Verbatim_Terminator_Invariants shares the empty closer and has its own reachable
// maximum because a verbatim terminator has one edge byte, not two brackets.
func Verbatim_Terminator_Invariants(
	terminator Verbatim_Terminator, namespace invariant.Namespace,
) {
	invariant.Tree(terminator, namespace).
		Range_Int(
			len(terminator), CLOSER_BYTES_MIN, VERBATIM_TERMINATOR_BYTES_MAX).
		Ensure()
}

// ACTIVE_CLOSER_BYTES_MIN is the shortest nonempty verbatim-string terminator.
const ACTIVE_CLOSER_BYTES_MIN = CLOSER_BYTES_MIN + 1

// Active_Closer is the nonempty terminator of a verbatim string.
type Active_Closer Closer

// Active_Closer_Invariants shares the closer ceiling and excludes zero.
func Active_Closer_Invariants(closer Active_Closer, namespace invariant.Namespace) {
	invariant.Tree(closer, namespace).
		Range_Int(len(closer), ACTIVE_CLOSER_BYTES_MIN, CLOSER_BYTES_MAX).
		Ensure()
}

// TERMINATOR_BYTES_MIN is the empty terminator carried when no heredoc is open.
const TERMINATOR_BYTES_MIN = 0

// TERMINATOR_BYTES_MAX is the widest heredoc word a line can carry: the scan window
// less the two bytes of the "<<" that opens it.
const TERMINATOR_BYTES_MAX = LINE_BYTES_MAX - 2

// Terminator is the word whose own line ends an open heredoc, or empty when none is.
type Terminator string

// Terminator_Invariants bounds a heredoc terminator's byte length.
func Terminator_Invariants(terminator Terminator, namespace invariant.Namespace) {
	invariant.Tree(terminator, namespace).
		Range_Int(len(terminator), TERMINATOR_BYTES_MIN, TERMINATOR_BYTES_MAX).
		Ensure()
}

// The scanner state that crosses line boundaries. Normal strings and
// character literals never cross a line, so they are not carried.
type Scan_Carry struct {
	// Block_Comment_Depth is the depth of nested block comments, zero outside one.
	Block_Comment_Depth Nesting_Depth
	// Raw_String_Close is the terminator an open verbatim string needs, or "" when
	// not inside one.
	Raw_String_Close Closer
	// Comment_Close is the terminator an open long-bracket comment needs, or "" when
	// not inside one.
	Comment_Close Comment_Closer
	// Heredoc_Terminator is the line that ends an open heredoc, or "" when not in one.
	Heredoc_Terminator Terminator
}

// Scan_Carry_Invariants states every piece of state that crosses a line boundary.
func Scan_Carry_Invariants(carry Scan_Carry, namespace invariant.Namespace) {
	Nesting_Depth_Invariants(carry.Block_Comment_Depth, namespace)
	Closer_Invariants(carry.Raw_String_Close, namespace)
	Comment_Closer_Invariants(carry.Comment_Close, namespace)
	Terminator_Invariants(carry.Heredoc_Terminator, namespace)
}

// Delimited_Carry is the block-comment, verbatim-string, or long-comment state used
// while scanning ordinary source. Heredoc bodies use their own line mode.
type Delimited_Carry struct {
	// Block_Comment_Depth is the depth of nested block comments, zero outside one.
	Block_Comment_Depth Nesting_Depth
	// Raw_String_Close is the terminator of an open verbatim string.
	Raw_String_Close Closer
	// Comment_Close is the terminator of an open long-bracket comment.
	Comment_Close Comment_Closer
}

// Delimited_Carry_Invariants states the three delimiter states ordinary scanning uses.
func Delimited_Carry_Invariants(carry Delimited_Carry, namespace invariant.Namespace) {
	Nesting_Depth_Invariants(carry.Block_Comment_Depth, namespace)
	Closer_Invariants(carry.Raw_String_Close, namespace)
	Comment_Closer_Invariants(carry.Comment_Close, namespace)
}

// ACTIVE_TERMINATOR_BYTES_MIN is the shortest nonempty heredoc terminator.
const ACTIVE_TERMINATOR_BYTES_MIN = TERMINATOR_BYTES_MIN + 1

// Active_Terminator is the nonempty word that ends an active heredoc.
type Active_Terminator string

// Active_Terminator_Invariants shares the heredoc width ceiling and excludes the empty
// state that means no heredoc is open.
func Active_Terminator_Invariants(
	terminator Active_Terminator, namespace invariant.Namespace,
) {
	invariant.Tree(terminator, namespace).
		Range_Int(
			len(terminator), ACTIVE_TERMINATOR_BYTES_MIN, TERMINATOR_BYTES_MAX).
		Ensure()
}

// Active_Heredoc_Carry is the scanner state while a heredoc body is open. Other carry
// fields cannot coexist with a heredoc, so this value does not contain them.
type Active_Heredoc_Carry struct {
	// Heredoc_Terminator is the nonempty word that ends the body.
	Heredoc_Terminator Active_Terminator
}

// Active_Heredoc_Carry_Invariants states the exclusive heredoc state. A heredoc opens
// only from fresh code, so no comment or verbatim-string state can coexist with it.
func Active_Heredoc_Carry_Invariants(
	carry Active_Heredoc_Carry, namespace invariant.Namespace,
) {
	Active_Terminator_Invariants(carry.Heredoc_Terminator, namespace)
}

// Heredoc_Carry is the scanner state after one heredoc body line. It contains only the
// terminator, which remains active or becomes empty when this line closes it.
type Heredoc_Carry struct {
	// Heredoc_Terminator is the remaining terminator, or empty after its line.
	Heredoc_Terminator Terminator
}

// Heredoc_Carry_Invariants states the exclusive post-line heredoc state.
func Heredoc_Carry_Invariants(carry Heredoc_Carry, namespace invariant.Namespace) {
	Terminator_Invariants(carry.Heredoc_Terminator, namespace)
}

// Code_Presence is the code-presence state of one scanned line.
type Code_Presence bool

// Code_Presence_Invariants states both code-presence states.
func Code_Presence_Invariants(value Code_Presence, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "The line has code.").
		Ensure()
}

// Comment_Presence is the comment-presence state of one scanned line.
type Comment_Presence bool

// Comment_Presence_Invariants states both comment-presence states.
func Comment_Presence_Invariants(
	value Comment_Presence, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "The line has a comment.").
		Ensure()
}

// Active_Scan is a nonblank ordinary line while its cursor points at unread data.
type Active_Scan struct {
	// Position is the line being read and how far along it the scan has reached.
	Position Active_Position
	// State is the delimiter state, updated as openers and closers are met.
	State Delimited_Carry
	// Has_Code records that the line bears code.
	Has_Code Code_Presence
	// Has_Comment records that the line bears a comment.
	Has_Comment Comment_Presence
}

// Active_Scan_Invariants states one ordinary scanner step and its verdict so far.
func Active_Scan_Invariants(scan Active_Scan, namespace invariant.Namespace) {
	Active_Position_Invariants(scan.Position, namespace)
	Delimited_Carry_Invariants(scan.State, namespace)
	Code_Presence_Invariants(scan.Has_Code, namespace)
	Comment_Presence_Invariants(scan.Has_Comment, namespace)
}

// SOURCE_BYTE_VALUES_COUNT is the number of values that one source byte can hold.
const SOURCE_BYTE_VALUES_COUNT = 256

// A scanner is one language prepared for scanning: its configuration plus a table of
// the bytes that can begin something the scan must inspect, so a run of ordinary code
// bytes is skipped in bulk instead of re-dispatched through every opener check.
type Scanner struct {
	// Line_Comment excludes display and test metadata that scanning cannot use.
	Line_Comment Comment_Tokens
	// Block_Comment_Open is empty when the language has no block comments.
	Block_Comment_Open Block_Comment_Opener
	// Block_Comment_Close is paired with the opener.
	Block_Comment_Close Block_Comment_Closer
	// Block_Comment_Nests distinguishes Rust from first-close languages.
	Block_Comment_Nests Block_Comment_Recursion
	// Verbatim_Strings can carry state across physical lines.
	Verbatim_Strings Verbatim_Delimiters
	// Quote_Strings always end on their physical line.
	Quote_Strings Quote_Delimiters
	// Long_Bracket enables Lua's computed delimiters.
	Long_Bracket Long_Bracket
	// Heredoc enables a separate carried line mode.
	Heredoc Heredoc
	// Trigger[b] is true when byte b can begin a comment or string opener, a heredoc, or
	// a long bracket — the only bytes a fresh-state scan must stop on. Every other
	// non-space byte is plain code.
	Trigger [SOURCE_BYTE_VALUES_COUNT]bool
}

// Scanner_Invariants states every syntax property the scanner reads. The trigger table
// has a fixed width in its type, so it carries no separate bound.
func Scanner_Invariants(scanner Scanner, namespace invariant.Namespace) {
	Comment_Tokens_Invariants(scanner.Line_Comment, namespace)
	Block_Comment_Opener_Invariants(scanner.Block_Comment_Open, namespace)
	Block_Comment_Closer_Invariants(scanner.Block_Comment_Close, namespace)
	Block_Comment_Recursion_Invariants(scanner.Block_Comment_Nests, namespace)
	Verbatim_Delimiters_Invariants(scanner.Verbatim_Strings, namespace)
	Quote_Delimiters_Invariants(scanner.Quote_Strings, namespace)
	Long_Bracket_Invariants(scanner.Long_Bracket, namespace)
	Heredoc_Invariants(scanner.Heredoc, namespace)
}

// Prepares a scanner for a language. The trigger table is the union of the first byte
// of every opener the language defines, taken from its fields alone so no language is
// special-cased: miss a field and a real opener would be skipped as if it were code.
func language_scanner(language *Seeded_Language) (prepared Scanner) {
	defer func() { Scanner_Invariants(prepared, "language_scanner.prepared") }()
	Seeded_Language_Invariants(*language, "language_scanner.language")
	prepared = Scanner{
		Line_Comment:        language.Line_Comment,
		Block_Comment_Open:  language.Block_Comment_Open,
		Block_Comment_Close: language.Block_Comment_Close,
		Block_Comment_Nests: language.Block_Comment_Nests,
		Verbatim_Strings:    language.Verbatim_Strings,
		Quote_Strings:       language.Quote_Strings,
		Long_Bracket:        language.Long_Bracket,
		Heredoc:             language.Heredoc,
	}
	for _, token := range language.Line_Comment {
		prepared.Trigger[token[0]] = true
	}
	if language.Block_Comment_Open != "" {
		prepared.Trigger[language.Block_Comment_Open[0]] = true
	}
	for _, delimiter := range language.Verbatim_Strings {
		prepared.Trigger[delimiter.Open[0]] = true
	}
	for _, delimiter := range language.Quote_Strings {
		prepared.Trigger[delimiter.Open[0]] = true
	}
	if language.Heredoc {
		prepared.Trigger['<'] = true
	}
	if language.Long_Bracket {
		prepared.Trigger['['] = true
	}
	return prepared
}

// Returns the partition of one line and the carry for the next.
func classify_line(
	line Line, carry Scan_Carry, scan_with *Scanner,
) (kind Line_Kind, carry_after Scan_Carry) {
	defer func() {
		Line_Kind_Invariants(kind, "classify_line.kind")
		Scan_Carry_Invariants(carry_after, "classify_line.carry_after")
	}()
	Line_Invariants(line, "classify_line.line")
	Scan_Carry_Invariants(carry, "classify_line.carry")
	Scanner_Invariants(*scan_with, "classify_line.scan_with")
	// A whitespace-only line is blank regardless of carried state, and neither opens
	// nor closes anything, so the carry passes through unchanged.
	if line_is_blank(line) {
		return LINE_KIND_BLANK, carry
	}
	// The blank verdict above already ran, so a heredoc body line has content. Its
	// verdict is always code, so the reader returns only the carry: a verdict that
	// cannot vary is stated here rather than by a bound that could never see the rest.
	if carry.Heredoc_Terminator != "" {
		heredoc_carry := classify_heredoc_line(Scan_Line(line), Active_Heredoc_Carry{
			Heredoc_Terminator: Active_Terminator(carry.Heredoc_Terminator),
		})
		return LINE_KIND_CODE, Scan_Carry{
			Heredoc_Terminator: heredoc_carry.Heredoc_Terminator,
		}
	}
	state := Delimited_Carry{
		Block_Comment_Depth: carry.Block_Comment_Depth,
		Raw_String_Close:    carry.Raw_String_Close,
		Comment_Close:       carry.Comment_Close,
	}
	active_kind, active_carry := classify_active_line(Scan_Line(line), state, scan_with)
	return Line_Kind(active_kind), active_carry
}

// Scans one nonblank line outside heredoc line mode.
func classify_active_line(
	line Scan_Line, state Delimited_Carry, scan_with *Scanner,
) (kind Scan_Verdict, carry_after Scan_Carry) {
	defer func() {
		Scan_Verdict_Invariants(kind, "classify_active_line.kind")
		Scan_Carry_Invariants(carry_after, "classify_active_line.carry_after")
	}()
	Scan_Line_Invariants(line, "classify_active_line.line")
	Delimited_Carry_Invariants(state, "classify_active_line.state")
	Scanner_Invariants(*scan_with, "classify_active_line.scan_with")
	scan := Active_Scan{Position: Active_Position{Line: line, Cursor: CURSOR_MIN}, State: state}
	heredoc := Terminator("")
	for int(scan.Position.Cursor) < len(scan.Position.Line) {
		Active_Scan_Invariants(scan, "classify_active_line.scan")
		at := &scan.Position
		if scan.State.Comment_Close != "" {
			scan.Has_Comment = true
			scan.State.Comment_Close = line_scan_long_comment_body(
				at, Active_Comment_Closer(scan.State.Comment_Close))
			continue
		}
		if scan.State.Raw_String_Close != "" {
			scan.Has_Code = true
			scan.State.Raw_String_Close = line_scan_raw(
				at, Active_Closer(scan.State.Raw_String_Close))
			continue
		}
		if scan.State.Block_Comment_Depth > 0 {
			scan.Has_Comment = true
			scan.State.Block_Comment_Depth = line_scan_block(
				at, Active_Nesting_Depth(scan.State.Block_Comment_Depth),
				scan_with.Block_Comment_Nests,
				Active_Block_Comment_Opener(scan_with.Block_Comment_Open),
				Active_Block_Comment_Closer(scan_with.Block_Comment_Close))
			continue
		}
		// Ordinary bytes skip language lookup, which is the dominant source path.
		if !scan_with.Trigger[scan.Position.Line[scan.Position.Cursor]] {
			if line_scan_plain(at, &scan_with.Trigger) {
				scan.Has_Code = true
			}
			continue
		}
		line_scan_fresh(
			at, scan_with, Fresh_Consumers{
				Block: func() {
					scan.State.Block_Comment_Depth = ACTIVE_NESTING_DEPTH_MIN
					scan.Has_Comment = true
				},
				Comment: func(closer Comment_Closer) {
					scan.State.Comment_Close = closer
					scan.Has_Comment = true
				},
				Code: func(closer Closer) {
					scan.State.Raw_String_Close = closer
					scan.Has_Code = true
				},
				Heredoc: func(terminator Terminator) {
					heredoc = terminator
					scan.Has_Code = true
				},
			})
	}
	return line_scan_verdict(scan.Has_Code), Scan_Carry{
		Block_Comment_Depth: scan.State.Block_Comment_Depth,
		Raw_String_Close:    scan.State.Raw_String_Close,
		Comment_Close:       scan.State.Comment_Close,
		Heredoc_Terminator:  heredoc,
	}
}

// Reads a line inside a heredoc body: the line is code, and a line equal to the
// terminator ends the heredoc.
func classify_heredoc_line(
	line Scan_Line, carry Active_Heredoc_Carry,
) (carry_after Heredoc_Carry) {
	defer func() {
		Heredoc_Carry_Invariants(carry_after, "classify_heredoc_line.carry_after")
	}()
	Scan_Line_Invariants(line, "classify_heredoc_line.line")
	Active_Heredoc_Carry_Invariants(carry, "classify_heredoc_line.carry")
	terminator := Terminator(carry.Heredoc_Terminator)
	if Terminator(strings.TrimSpace(string(line))) == terminator {
		terminator = ""
	}
	return Heredoc_Carry{Heredoc_Terminator: terminator}
}

// Advances past a byte that can begin nothing: insignificant whitespace, or plain
// code. The first such code byte makes the line code, after which the run of ordinary
// bytes is skipped to the next trigger in one tight loop. The language is not stated
// here because none of it is read — which is the point, since almost every byte of a
// source file takes this path and stating a language costs more than reading one.
func line_scan_plain(
	at *Active_Position, trigger *[SOURCE_BYTE_VALUES_COUNT]bool,
) (code Code_Presence) {
	defer func() { Code_Presence_Invariants(code, "line_scan_plain.code") }()
	Active_Position_Invariants(*at, "line_scan_plain.at")
	line := at.Line
	cursor := int(at.Cursor)
	if byte_is_space(Source_Byte(line[cursor])) {
		at.Cursor++
		return false
	}
	cursor++
	for cursor < len(line) && !trigger[line[cursor]] {
		cursor++
	}
	at.Cursor = Cursor(cursor)
	return true
}

// Advances inside a verbatim string, where every byte is code and only the matching
// close ends it.
func line_scan_raw(
	at *Active_Position, closer Active_Closer,
) (closer_after Closer) {
	defer func() { Closer_Invariants(closer_after, "line_scan_raw.closer_after") }()
	Active_Position_Invariants(*at, "line_scan_raw.at")
	Active_Closer_Invariants(closer, "line_scan_raw.closer")
	if bytes.HasPrefix(at.Line[at.Cursor:], []byte(closer)) {
		at.Cursor += Cursor(len(closer))
		return ""
	}
	at.Cursor++
	return Closer(closer)
}

// Advances inside a block comment, where every byte is comment and only an open (when
// nesting) or a close moves the depth.
func line_scan_block(
	at *Active_Position, depth Active_Nesting_Depth,
	nests Block_Comment_Recursion, opener Active_Block_Comment_Opener,
	closer Active_Block_Comment_Closer,
) (depth_after Nesting_Depth) {
	defer func() { Nesting_Depth_Invariants(depth_after, "line_scan_block.depth_after") }()
	Active_Position_Invariants(*at, "line_scan_block.at")
	Active_Nesting_Depth_Invariants(depth, "line_scan_block.depth")
	Block_Comment_Recursion_Invariants(nests, "line_scan_block.nests")
	Active_Block_Comment_Opener_Invariants(opener, "line_scan_block.opener")
	Active_Block_Comment_Closer_Invariants(closer, "line_scan_block.closer")
	if nests {
		if bytes.HasPrefix(at.Line[at.Cursor:], []byte(opener)) {
			// The depth saturates at its bound rather than growing with the file: a
			// comment nested deeper than the bound closes early, which keeps the
			// carried depth stated by a reachable range.
			if depth < NESTING_DEPTH_MAX {
				depth++
			}
			at.Cursor += Cursor(len(opener))
			return Nesting_Depth(depth)
		}
	}
	if bytes.HasPrefix(at.Line[at.Cursor:], []byte(closer)) {
		at.Cursor += Cursor(len(closer))
		return Nesting_Depth(depth - 1)
	}
	at.Cursor++
	return Nesting_Depth(depth)
}

// Advances inside a long-bracket comment, where every byte is comment and only the
// matching leveled closer ends it.
func line_scan_long_comment_body(
	at *Active_Position, closer Active_Comment_Closer,
) (closer_after Comment_Closer) {
	defer func() {
		Comment_Closer_Invariants(closer_after, "line_scan_long_comment_body.closer_after")
	}()
	Active_Position_Invariants(*at, "line_scan_long_comment_body.at")
	Active_Comment_Closer_Invariants(closer, "line_scan_long_comment_body.closer")
	if bytes.HasPrefix(at.Line[at.Cursor:], []byte(closer)) {
		at.Cursor += Cursor(len(closer))
		return ""
	}
	at.Cursor++
	return Comment_Closer(closer)
}

// Opening is the result of a syntax-opener match.
type Opening bool

// Opening_Invariants states both syntax-opener results.
func Opening_Invariants(value Opening, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "A syntax form opens.").
		Ensure()
}

// Reports whether a long-bracket comment — a line-comment token then a long bracket,
// like --[[ or --[=[ — opens at the cursor, recording the comment and its closer.
func line_scan_long_comment(
	at *Active_Position, tokens Long_Bracket_Comment_Tokens,
) (closer Comment_Closer, opened Opening) {
	defer func() {
		Comment_Closer_Invariants(closer, "line_scan_long_comment.closer")
		Opening_Invariants(opened, "line_scan_long_comment.opened")
	}()
	Active_Position_Invariants(*at, "line_scan_long_comment.at")
	Long_Bracket_Comment_Tokens_Invariants(tokens, "line_scan_long_comment.tokens")
	for _, token := range tokens {
		if !bytes.HasPrefix(at.Line[at.Cursor:], []byte(token)) {
			continue
		}
		// The cursor steps past the comment token so the bracket match reads from
		// the scan's own position; a failed match winds it back.
		saved := at.Cursor
		at.Cursor += Cursor(len(token))
		bracket_closer, bracketed := long_bracket_open((*Bounded_Position)(at))
		if bracketed {
			return Comment_Closer(bracket_closer), true
		}
		at.Cursor = saved
	}
	return "", false
}

// Reports whether a long-bracket string — like [[ or [=[ — opens at the cursor,
// recording its leveled closer.
func line_scan_long_string(
	at *Active_Position,
) (closer Bracket_Closer, opened Opening) {
	defer func() {
		Bracket_Closer_Invariants(closer, "line_scan_long_string.closer")
		Opening_Invariants(opened, "line_scan_long_string.opened")
	}()
	Active_Position_Invariants(*at, "line_scan_long_string.at")
	bracket_closer, bracketed := long_bracket_open((*Bounded_Position)(at))
	if !bracketed {
		return "", false
	}
	return bracket_closer, true
}

// Reports whether a long bracket — '[' then a run of '=' then '[' — opens at the
// cursor, recording the matching closer ']' run-of-'=' ']' and stepping the cursor past
// the opener. The cursor is left where it was when no bracket opens.
func long_bracket_open(
	at *Bounded_Position,
) (closer Bracket_Closer, opened Opening) {
	defer func() {
		Bracket_Closer_Invariants(closer, "long_bracket_open.closer")
		Opening_Invariants(opened, "long_bracket_open.opened")
	}()
	Bounded_Position_Invariants(*at, "long_bracket_open.at")
	line := at.Line
	cursor := int(at.Cursor)
	if cursor >= len(line) {
		return "", false
	}
	if line[cursor] != '[' {
		return "", false
	}
	read := cursor + 1
	equal_count := 0
	// The run is bounded so the computed closer stays within its stated width.
	for read < len(line) && line[read] == '=' && equal_count < HASH_COUNT_MAX {
		equal_count++
		read++
	}
	if read >= len(line) {
		return "", false
	}
	if line[read] != '[' {
		return "", false
	}
	at.Cursor = Cursor(read + 1)
	return Bracket_Closer("]" + strings.Repeat("=", equal_count) + "]"), true
}

// BRACKET_CLOSER_BYTES_MIN is the empty terminator a failed match reports alongside
// its false.
const BRACKET_CLOSER_BYTES_MIN = 0

// BRACKET_CLOSER_BYTES_ABSENT is the one width a bracket terminator never has: a match
// yields at least the bare "]]", so a single byte is unreachable.
const BRACKET_CLOSER_BYTES_ABSENT = 1

// BRACKET_CLOSER_BYTES_MAX is a long bracket at the equals-run bound: two brackets
// around the run.
const BRACKET_CLOSER_BYTES_MAX = HASH_COUNT_MAX + 2

// Bracket_Closer is the terminator a long bracket computes for itself. Its missing
// one-byte width follows from the two brackets around its optional equals run.
type Bracket_Closer string

// Bracket_Closer_Invariants bounds a long bracket's computed terminator.
func Bracket_Closer_Invariants(closer Bracket_Closer, namespace invariant.Namespace) {
	invariant.Tree(closer, namespace).
		Range_Holed_Int(
			len(closer), BRACKET_CLOSER_BYTES_MIN, BRACKET_CLOSER_BYTES_MAX,
			BRACKET_CLOSER_BYTES_ABSENT, BRACKET_CLOSER_BYTES_ABSENT,
			BRACKET_CLOSER_BYTES_ABSENT, BRACKET_CLOSER_BYTES_ABSENT).
		Ensure()
}

// Fresh_Consumers applies the four state changes a fresh token can cause. The
// callbacks keep the token matcher independent from the line accumulator.
type Fresh_Consumers struct {
	// Block records a block-comment opener.
	Block func()
	// Comment records a line or long-bracket comment and its optional closer.
	Comment func(closer Comment_Closer)
	// Code records ordinary or quoted code and its optional verbatim closer.
	Code func(closer Closer)
	// Heredoc records the terminator that changes how following lines are read.
	Heredoc func(terminator Terminator)
}

// Fresh_Consumers_Invariants requires every state change to have an injected owner.
func Fresh_Consumers_Invariants(
	consume Fresh_Consumers, namespace invariant.Namespace,
) {
	invariant.Always(consume.Block != nil, "A fresh scan always records block comments.")
	invariant.Always(consume.Comment != nil, "A fresh scan always records comments.")
	invariant.Always(consume.Code != nil, "A fresh scan always records code.")
	invariant.Always(consume.Heredoc != nil, "A fresh scan always records heredocs.")
}

// Dispatches one trigger byte through the language syntax in precedence order.
func line_scan_fresh(
	at *Active_Position, scan_with *Scanner, consume Fresh_Consumers,
) {
	Active_Position_Invariants(*at, "line_scan_fresh.at")
	Scanner_Invariants(*scan_with, "line_scan_fresh.scan_with")
	Fresh_Consumers_Invariants(consume, "line_scan_fresh.consume")
	if scan_with.Block_Comment_Open != "" {
		opener := Active_Block_Comment_Opener(scan_with.Block_Comment_Open)
		if bytes.HasPrefix(at.Line[at.Cursor:], []byte(opener)) {
			at.Cursor += Cursor(len(opener))
			consume.Block()
			return
		}
	}
	if scan_with.Long_Bracket {
		closer, opened := line_scan_long_comment(
			at, Long_Bracket_Comment_Tokens(scan_with.Line_Comment))
		if opened {
			consume.Comment(closer)
			return
		}
	}
	for _, token := range scan_with.Line_Comment {
		if bytes.HasPrefix(at.Line[at.Cursor:], []byte(token)) {
			at.Cursor = Cursor(len(at.Line))
			consume.Comment("")
			return
		}
	}
	if scan_with.Long_Bracket {
		closer, opened := line_scan_long_string(at)
		if opened {
			consume.Code(Closer(closer))
			return
		}
	}
	if scan_with.Heredoc {
		terminator, opened := heredoc_open(at)
		if opened {
			at.Cursor = Cursor(len(at.Line))
			consume.Heredoc(terminator)
			return
		}
	}
	if len(scan_with.Verbatim_Strings) > 0 {
		closer, opened := verbatim_open(
			at, Active_Verbatim_Delimiters(scan_with.Verbatim_Strings))
		if opened {
			consume.Code(Closer(closer))
			return
		}
	}
	if len(scan_with.Quote_Strings) > 0 {
		if quote_open(at, Active_Quote_Delimiters(scan_with.Quote_Strings)) {
			consume.Code("")
			return
		}
	}
	at.Cursor++
	consume.Code("")
}

// Reports whether a heredoc opener begins at the cursor — << then an optional - or ~,
// optional space, then a quoted word or an uppercase/underscore word — and if so its
// terminator word and the opener's byte length. The uppercase rule tells <<EOF apart
// from the a << b shift operator.
func heredoc_open(
	at *Active_Position,
) (terminator Terminator, opened Opening) {
	defer func() {
		Terminator_Invariants(terminator, "heredoc_open.terminator")
		Opening_Invariants(opened, "heredoc_open.opened")
	}()
	Active_Position_Invariants(*at, "heredoc_open.at")
	if !bytes.HasPrefix(at.Line[at.Cursor:], []byte("<<")) {
		return "", false
	}
	// A separate cursor leaves the scan at the trigger when the opener is malformed.
	probe_cursor := at.Cursor + 2
	if int(probe_cursor) < len(at.Line) {
		if at.Line[probe_cursor] == '-' {
			probe_cursor++
		} else if at.Line[probe_cursor] == '~' {
			probe_cursor++
		}
	}
	for int(probe_cursor) < len(at.Line) &&
		byte_is_space(Source_Byte(at.Line[probe_cursor])) {
		probe_cursor++
	}
	quoted := false
	if int(probe_cursor) < len(at.Line) {
		if heredoc_is_quote(Source_Byte(at.Line[probe_cursor])) {
			quoted = true
			probe_cursor++
		}
	}
	start := probe_cursor
	for int(probe_cursor) < len(at.Line) &&
		byte_is_identifier(Source_Byte(at.Line[probe_cursor])) {
		probe_cursor++
	}
	if probe_cursor == start {
		return "", false
	}
	if !quoted {
		if !heredoc_word_start(Identifier_Byte(at.Line[start])) {
			return "", false
		}
	}
	terminator = Terminator(at.Line[start:probe_cursor])
	if quoted {
		if int(probe_cursor) < len(at.Line) {
			probe_cursor++
		}
	}
	at.Cursor = probe_cursor
	return terminator, true
}

// Heredoc_Quote is the quoted-delimiter state of one heredoc opener.
type Heredoc_Quote bool

// Heredoc_Quote_Invariants states both quoted-delimiter states.
func Heredoc_Quote_Invariants(value Heredoc_Quote, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "A heredoc delimiter is quoted.").
		Ensure()
}

// Reports whether a byte opens a quoted heredoc delimiter.
func heredoc_is_quote(character Source_Byte) (quote Heredoc_Quote) {
	defer func() { Heredoc_Quote_Invariants(quote, "heredoc_is_quote.quote") }()
	Source_Byte_Invariants(character, "heredoc_is_quote.character")
	switch character {
	case '\'', '"', '`':
		return true
	}
	return false
}

// Heredoc_Word_Start is the valid-start state of an unquoted heredoc word.
type Heredoc_Word_Start bool

// Heredoc_Word_Start_Invariants states both valid-start states.
func Heredoc_Word_Start_Invariants(
	value Heredoc_Word_Start, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "An unquoted heredoc word has a valid start.").
		Ensure()
}

// Reports whether a byte may begin an unquoted heredoc delimiter: an uppercase letter
// or underscore, the convention that keeps a << b from looking like a heredoc.
func heredoc_word_start(character Identifier_Byte) (start Heredoc_Word_Start) {
	defer func() {
		Heredoc_Word_Start_Invariants(start, "heredoc_word_start.start")
	}()
	Identifier_Byte_Invariants(character, "heredoc_word_start.character")
	if character == '_' {
		return true
	}
	return character >= 'A' && character <= 'Z'
}

// IDENTIFIER_BYTE_MIN is the lowest byte an identifier may begin with, the digit zero.
const IDENTIFIER_BYTE_MIN = 48

// IDENTIFIER_BYTE_MAX is the highest, the lowercase z.
const IDENTIFIER_BYTE_MAX = 122

// Identifier_Byte is a byte already known to belong to an identifier. It is distinct
// from Source_Byte because the word check is only ever reached once the identifier scan
// has advanced, so the bytes outside the identifier range never arrive here.
type Identifier_Byte uint8

// Identifier_Byte_Invariants bounds a byte known to belong to an identifier.
func Identifier_Byte_Invariants(
	character Identifier_Byte, namespace invariant.Namespace,
) {
	invariant.Tree(character, namespace).
		Range_Uint8(
			uint8(character), uint8(IDENTIFIER_BYTE_MIN), uint8(IDENTIFIER_BYTE_MAX)).
		Ensure()
}

// SOURCE_BYTE_MIN is the zero byte, which a file may carry past the binary sniff.
const SOURCE_BYTE_MIN = 0

// SOURCE_BYTE_MAX is the widest byte value, since source bytes are untrusted input.
const SOURCE_BYTE_MAX = 255

// Source_Byte is one byte of a source file, read as it was found.
type Source_Byte uint8

// Source_Byte_Invariants bounds a source byte to the whole byte domain, since nothing
// about untrusted input narrows it.
func Source_Byte_Invariants(character Source_Byte, namespace invariant.Namespace) {
	invariant.Tree(character, namespace).
		Range_Uint8(uint8(character), uint8(SOURCE_BYTE_MIN), uint8(SOURCE_BYTE_MAX)).
		Ensure()
}

// Reads the line's partition from the accumulated scan: code wins a line it shares
// with a comment, then a comment, else blank.
func line_scan_verdict(has_code Code_Presence) (kind Scan_Verdict) {
	defer func() { Scan_Verdict_Invariants(kind, "line_scan_verdict.kind") }()
	Code_Presence_Invariants(has_code, "line_scan_verdict.has_code")
	if has_code {
		return Scan_Verdict(LINE_KIND_CODE)
	}
	return Scan_Verdict(LINE_KIND_COMMENT)
}

// Scan_Verdict is the partition a scanned line falls into. It is distinct from
// Line_Kind because a scan is only ever built for a line with content, and the fresh
// dispatch marks every content byte as code or comment — so a scanned line is never
// blank, whatever a physical line may be.
type Scan_Verdict int

// Scan_Verdict_Invariants holds a scanned line's verdict to the two it can take, and
// witnesses each one.
func Scan_Verdict_Invariants(kind Scan_Verdict, namespace invariant.Namespace) {
	invariant.Tree(kind, namespace).
		Enum_Int(int(kind), int(LINE_KIND_CODE), int(LINE_KIND_COMMENT)).
		Ensure()
}

// Reports whether a verbatim string begins at the cursor and, if so, its terminator
// and the opener's byte length.
func verbatim_open(
	at *Active_Position, delimiters Active_Verbatim_Delimiters,
) (closer Verbatim_Terminator, opened Opening) {
	defer func() {
		Verbatim_Terminator_Invariants(closer, "verbatim_open.closer")
		Opening_Invariants(opened, "verbatim_open.opened")
	}()
	Active_Position_Invariants(*at, "verbatim_open.at")
	Active_Verbatim_Delimiters_Invariants(delimiters, "verbatim_open.delimiters")
	for _, delimiter := range delimiters {
		if !bytes.HasPrefix(at.Line[at.Cursor:], []byte(delimiter.Open)) {
			continue
		}
		if byte_is_identifier(Source_Byte(delimiter.Open[0])) {
			if at.Cursor > CURSOR_MIN {
				if byte_is_identifier(Source_Byte(at.Line[at.Cursor-1])) {
					continue
				}
			}
		}
		if !delimiter.Hashable {
			at.Cursor += Cursor(len(delimiter.Open))
			return Verbatim_Terminator(delimiter.Close), true
		}
		read := int(at.Cursor) + len(delimiter.Open)
		hash_count := 0
		for read < len(at.Line) && at.Line[read] == '#' && hash_count < HASH_COUNT_MAX {
			hash_count++
			read++
		}
		if read < len(at.Line) {
			if at.Line[read] == '"' {
				at.Cursor += Cursor(len(delimiter.Open) + hash_count + 1)
				return Verbatim_Terminator(
					"\"" + strings.Repeat("#", hash_count)), true
			}
		}
	}
	return "", false
}

// Reports whether a single-line string or character literal begins at the cursor,
// moving the cursor past what it consumes.
func quote_open(
	at *Active_Position, delimiters Active_Quote_Delimiters,
) (opened Opening) {
	defer func() { Opening_Invariants(opened, "quote_open.opened") }()
	Active_Position_Invariants(*at, "quote_open.at")
	Active_Quote_Delimiters_Invariants(delimiters, "quote_open.delimiters")
	for _, one := range delimiters {
		Quote_Delimiter_Invariants(one, "quote_open.delimiter")
	}
	for _, delimiter := range delimiters {
		if !bytes.HasPrefix(at.Line[at.Cursor:], []byte(delimiter.Open)) {
			continue
		}
		if delimiter.Character_Like {
			scan_character_or_lifetime(at)
			return true
		}
		scan_quoted(at, String_Delimiter{
			Open: delimiter.Open, Close: delimiter.Close, Escape: delimiter.Escape,
		})
		return true
	}
	return false
}

// Moves the cursor past a single-line quoted string starting at its opening delimiter,
// stopping at the first unescaped close or end of line.
func scan_quoted(at *Active_Position, delimiter String_Delimiter) {
	Active_Position_Invariants(*at, "scan_quoted.at")
	String_Delimiter_Invariants(delimiter, "scan_quoted.delimiter")
	line := at.Line
	read := int(at.Cursor) + len(delimiter.Open)
	for read < len(line) {
		// A zero escape means the syntax has no escapes. It must not match a zero
		// byte from untrusted source.
		if delimiter.Escape != ESCAPE_BYTE_NONE {
			if line[read] == byte(delimiter.Escape) {
				read += 2 // skip the escape byte and the character it escapes
				continue
			}
		}
		if bytes.HasPrefix(line[read:], []byte(delimiter.Close)) {
			at.Cursor = Cursor(read + len(delimiter.Close))
			return
		}
		read++
	}
	at.Cursor = Cursor(len(line))
}

// Moves the cursor past a character or rune literal starting at the apostrophe, or by
// one when the apostrophe is a Rust lifetime tick rather than a literal — so a
// lifetime never opens a string that eats the line.
func scan_character_or_lifetime(at *Active_Position) {
	Active_Position_Invariants(*at, "scan_character_or_lifetime.at")
	cursor := int(at.Cursor)
	line := at.Line
	if cursor+1 >= len(line) {
		at.Cursor++
		return
	}
	if line[cursor+1] == '\\' {
		limit := cursor + CHARACTER_ESCAPE_BYTES_MAX
		for read := cursor + 3; read < len(line) && read <= limit; read++ {
			if line[read] == '\'' {
				at.Cursor = Cursor(read + 1)
				return
			}
		}
		at.Cursor++
		return
	}
	if line[cursor+1] != '\'' {
		_, size := utf8.DecodeRune(line[cursor+1:])
		if size > 0 {
			if cursor+1+size < len(line) {
				if line[cursor+1+size] == '\'' {
					at.Cursor = Cursor(cursor + 1 + size + 1)
					return
				}
			}
		}
	}
	at.Cursor++
}

// CHARACTER_ESCAPE_BYTES_MAX bounds how far past an apostrophe an escaped character
// literal's close is looked for, so a stray apostrophe cannot swallow the line.
const CHARACTER_ESCAPE_BYTES_MAX = 12

// Blank_Line is the whitespace-only state of one physical line.
type Blank_Line bool

// Blank_Line_Invariants states both physical-line states.
func Blank_Line_Invariants(value Blank_Line, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "A physical line is blank.").
		Ensure()
}

// Reports whether a line is empty or only ASCII whitespace.
func line_is_blank(line Line) (blank Blank_Line) {
	defer func() { Blank_Line_Invariants(blank, "line_is_blank.blank") }()
	Line_Invariants(line, "line_is_blank.line")
	for _, character := range line {
		if !byte_is_space(Source_Byte(character)) {
			return false
		}
	}
	return true
}

// Whitespace is the ASCII-whitespace state of one source byte.
type Whitespace bool

// Whitespace_Invariants states both source-byte states.
func Whitespace_Invariants(value Whitespace, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "A source byte is ASCII whitespace.").
		Ensure()
}

// Reports whether a byte is ASCII whitespace. Newline is excluded because the source
// is already split on it.
func byte_is_space(character Source_Byte) (space Whitespace) {
	defer func() { Whitespace_Invariants(space, "byte_is_space.space") }()
	Source_Byte_Invariants(character, "byte_is_space.character")
	switch character {
	case ' ', '\t', '\r', '\f', '\v':
		return true
	}
	return false
}

// Identifier_Membership is the identifier-membership state of one source byte.
type Identifier_Membership bool

// Identifier_Membership_Invariants states both source-byte membership states.
func Identifier_Membership_Invariants(
	value Identifier_Membership, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "A source byte belongs to an identifier.").
		Ensure()
}

// Reports whether a byte may appear in an identifier, used to keep a raw-string lead
// from being recognized in the middle of a name.
func byte_is_identifier(character Source_Byte) (identifier Identifier_Membership) {
	defer func() {
		Identifier_Membership_Invariants(identifier, "byte_is_identifier.identifier")
	}()
	Source_Byte_Invariants(character, "byte_is_identifier.character")
	if character == '_' {
		return true
	}
	if character >= 'a' {
		if character <= 'z' {
			return true
		}
	}
	if character >= 'A' {
		if character <= 'Z' {
			return true
		}
	}
	return character >= '0' && character <= '9'
}

// Test_File_Status is the source-or-test role of one counted file.
type Test_File_Status bool

// Test_File_Status_Invariants states both counted-file roles.
func Test_File_Status_Invariants(
	value Test_File_Status, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "The file is test code.").
		Ensure()
}

// File_Count is one counted file: its path, the language it was read as, and its
// line partition.
type File_Count struct {
	// Path is the file's path.
	Path File_Path
	// Language is the display name of the language it was read as.
	Language Language_Name
	// Counts is the file's line partition.
	Counts Counts
	// Is_Test reports whether the file is test code rather than source.
	Is_Test Test_File_Status
}

// File_Count_Invariants states a counted file's path, language, partition, and role.
func File_Count_Invariants(file File_Count, namespace invariant.Namespace) {
	File_Path_Invariants(file.Path, namespace)
	Language_Name_Invariants(file.Language, namespace)
	Counts_Invariants(file.Counts, namespace)
	Test_File_Status_Invariants(file.Is_Test, namespace)
}

// FILES_COUNT_MIN is the empty report: a tree holding no recognized file.
const FILES_COUNT_MIN = 0

// FILES_COUNT_MAX bounds the files one run counts. A walk stops selecting candidates
// once it is reached, so a tree larger than this is reported short rather than
// counted against an unbounded slice.
const FILES_COUNT_MAX = 65535

// File_Counts are the counted files of a report, in the order the walk visited them.
type File_Counts []File_Count

// File_Counts_Invariants bounds how many files a report carries.
func File_Counts_Invariants(files File_Counts, namespace invariant.Namespace) {
	invariant.Tree(files, namespace).
		Range_Int(len(files), FILES_COUNT_MIN, FILES_COUNT_MAX).
		Ensure()
}

// Report is the result of counting a tree: one File_Count per counted file, in the
// lexical order the walk visited them.
type Report struct {
	// Files are the counted files.
	Files File_Counts
	// Skipped is what the walk found but did not count.
	Skipped Skipped
}

// Report_Invariants states the counted files a report carries and what it left out.
func Report_Invariants(report Report, namespace invariant.Namespace) {
	File_Counts_Invariants(report.Files, namespace)
	Skipped_Invariants(report.Skipped, namespace)
}

// Root_Report is the result of one command-line root before Main merges root tallies.
type Root_Report Report

// Root_Report_Invariants states the counted files and checks the root-level omission
// tallies. The merged Report owns their aggregate witnesses.
func Root_Report_Invariants(report Root_Report, namespace invariant.Namespace) {
	Root_Skipped_Invariants(Root_Skipped(report.Skipped), namespace)
	invariant.Tree(report, namespace).
		Range_Int(len(report.Files), FILES_COUNT_MIN, FILES_COUNT_MAX).
		Ensure()
}

// Ignore_Predicate reports whether a path, relative to its tree root, is ignored. An
// ignored directory is pruned, an ignored file is skipped.
type Ignore_Predicate func(relative_path string, is_directory bool) (ignored bool)

// Count_Input is a file tree to count and the filters to apply while walking it.
type Count_Input struct {
	// File_System is the tree to walk and read.
	File_System fs.FS
	// Is_Ignored is the ignore filter, or nil to ignore nothing.
	Is_Ignored Ignore_Predicate
	// Include_Hidden counts dot-prefixed entries that are skipped by default.
	Include_Hidden Include_Hidden
	// Classifier partitions each recognized file's source into line kinds.
	Classifier File_Classifier
	// Concurrency bounds the read-and-classify worker pool; below one means one. It
	// stays an unbounded integer because it is host-supplied and count_classify
	// clamps it at both ends.
	Concurrency Concurrency
}

// Count_Input_Invariants states the walk's classifier and settings. The file system
// and ignore predicate are interface and function values with no preset of their own.
func Count_Input_Invariants(input Count_Input, namespace invariant.Namespace) {
	Include_Hidden_Invariants(input.Include_Hidden, namespace)
	File_Classifier_Invariants(input.Classifier, namespace)
	Concurrency_Invariants(input.Concurrency, namespace)
}

// Count walks the tree, classifies every recognized file, and returns one File_Count
// per file in the lexical order the walk visited them. Reading and classifying fan
// out across workers, which does not affect the result order.
func Count(input Count_Input) (report Root_Report, err error) {
	defer func() { Root_Report_Invariants(report, "Count.report") }()
	Count_Input_Invariants(input, "Count.input")
	candidates, overflow, walk_err := count_candidates(
		input.File_System, input.Is_Ignored, input.Include_Hidden)
	if walk_err != nil {
		return Root_Report{}, walk_err
	}
	files, classified := count_classify(
		input.File_System, candidates, input.Classifier, input.Concurrency)
	skipped := Root_Skipped{
		Unreadable: classified.Unreadable,
		Oversized:  classified.Oversized,
		Binary:     classified.Binary,
		Overflow:   overflow,
	}
	// Nothing sums the run, but a language's own total is still stated as one number, so
	// that number is what the line bound has to hold. A file that would carry its own
	// language past the bound is left out and counted, rather than failing the run.
	lines_of := map[Language_Name]Line_Count{}
	kept := File_Counts{}
	for _, one := range files {
		if int(lines_of[one.Language])+int(counts_lines(one.Counts)) > LINE_COUNT_MAX {
			skipped.Past_Lines++
			continue
		}
		lines_of[one.Language] += counts_lines(one.Counts)
		kept = append(kept, one)
	}
	return Root_Report{Files: kept, Skipped: Skipped(skipped)}, nil
}

// A file the walk selected for counting and the language to read it as.
type Candidate struct {
	// Path is the file's path relative to the walked root.
	Path Counted_Path
	// Language is the language the file's extension resolved to.
	Language Seeded_Language
}

// Candidate_Invariants states a selected file's path and the language to read it as.
func Candidate_Invariants(candidate Candidate, namespace invariant.Namespace) {
	Counted_Path_Invariants(candidate.Path, namespace)
	Seeded_Language_Invariants(candidate.Language, namespace)
}

// COUNTED_PATH_BYTES_MIN is the shortest path a walk can select: a name that is a bare
// extension, like ".c", which only a run counting hidden entries reaches.
const COUNTED_PATH_BYTES_MIN = 2

// COUNTED_PATH_BYTES_MAX is the path bound, as for any path.
const COUNTED_PATH_BYTES_MAX = FILE_PATH_BYTES_MAX

// Counted_Path is the path of a file a walk selected. It is distinct from File_Path
// because a path named on the command line may be a single byte, while a selected one
// must carry at least a recognized extension.
type Counted_Path string

// Counted_Path_Invariants bounds a selected file's path length.
func Counted_Path_Invariants(file_path Counted_Path, namespace invariant.Namespace) {
	invariant.Tree(file_path, namespace).
		Range_Int(len(file_path), COUNTED_PATH_BYTES_MIN, COUNTED_PATH_BYTES_MAX).
		Ensure()
}

// Candidates are the files a walk selected for counting.
type Candidates []Candidate

// Candidates_Invariants bounds how many files a walk selected.
func Candidates_Invariants(candidates Candidates, namespace invariant.Namespace) {
	invariant.Tree(candidates, namespace).
		Range_Int(len(candidates), FILES_COUNT_MIN, FILES_COUNT_MAX).
		Ensure()
}

// Walks the tree and returns the recognized files to count, pruning hidden and
// ignored directories so their contents are never read.
func count_candidates(
	file_system fs.FS, is_ignored Ignore_Predicate, include_hidden Include_Hidden,
) (candidates Candidates, overflow Dropped_Count, err error) {
	defer func() {
		Candidates_Invariants(candidates, "count_candidates.candidates")
		Dropped_Count_Invariants(overflow, "count_candidates.overflow")
	}()
	Include_Hidden_Invariants(include_hidden, "count_candidates.include_hidden")
	walk_err := fs.WalkDir(file_system, ".",
		func(file_path string, entry fs.DirEntry, step_err error) (result error) {
			return count_visit(
				&candidates, &overflow, File_Path(file_path), entry, step_err,
				is_ignored, include_hidden)
		})
	if walk_err != nil {
		return nil, 0, walk_err
	}
	return candidates, overflow, nil
}

// Decides one walked entry: prune it, skip it, or append it as a candidate.
func count_visit(
	candidates *Candidates, overflow *Dropped_Count, file_path File_Path,
	entry fs.DirEntry, step_err error,
	is_ignored Ignore_Predicate, include_hidden Include_Hidden,
) (result error) {
	Candidates_Invariants(*candidates, "count_visit.candidates")
	Dropped_Count_Invariants(*overflow, "count_visit.overflow")
	File_Path_Invariants(file_path, "count_visit.file_path")
	Include_Hidden_Invariants(include_hidden, "count_visit.include_hidden")
	if step_err != nil {
		return step_err
	}
	if count_skip_hidden(file_path, include_hidden) {
		return count_prune(entry)
	}
	if count_skip_ignored(file_path, entry, is_ignored) {
		return count_prune(entry)
	}
	if entry.IsDir() {
		return nil
	}
	language, recognized := language_for_path(file_path)
	if !recognized {
		return nil
	}
	// Past the bound the walk keeps going and keeps counting: stopping would leave the
	// number of files it could not take unknown, which is exactly the thing the report
	// must not do. Only the selecting stops.
	if len(*candidates) >= FILES_COUNT_MAX {
		if *overflow < DROPPED_COUNT_MAX {
			*overflow++
		}
		return nil
	}
	*candidates = append(*candidates, Candidate{
		Path:     Counted_Path(file_path),
		Language: Seeded_Language(language),
	})
	return nil
}

// Skips a directory's whole subtree, or a single file.
func count_prune(entry fs.DirEntry) (result error) {
	if entry.IsDir() {
		return fs.SkipDir
	}
	return nil
}

// Skip_Decision is the decision to omit one walked path.
type Skip_Decision bool

// Skip_Decision_Invariants states both walk decisions.
func Skip_Decision_Invariants(value Skip_Decision, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "A walked path is omitted.").
		Ensure()
}

// Reports whether a path is a hidden entry to skip. The root, named ".", also begins
// with a dot and must not be mistaken for one.
func count_skip_hidden(
	file_path File_Path, include_hidden Include_Hidden,
) (skip Skip_Decision) {
	defer func() { Skip_Decision_Invariants(skip, "count_skip_hidden.skip") }()
	File_Path_Invariants(file_path, "count_skip_hidden.file_path")
	Include_Hidden_Invariants(include_hidden, "count_skip_hidden.include_hidden")
	if include_hidden {
		return false
	}
	if file_path == "." {
		return false
	}
	return Skip_Decision(path_is_hidden(file_path))
}

// Reports whether the injected predicate ignores a path.
func count_skip_ignored(
	file_path File_Path, entry fs.DirEntry, is_ignored Ignore_Predicate,
) (skip Skip_Decision) {
	defer func() { Skip_Decision_Invariants(skip, "count_skip_ignored.skip") }()
	File_Path_Invariants(file_path, "count_skip_ignored.file_path")
	if is_ignored == nil {
		return false
	}
	return Skip_Decision(is_ignored(string(file_path), entry.IsDir()))
}

// SKIP_REASON_COUNTED means the file was read and classified.
const SKIP_REASON_COUNTED Skip_Reason = 0

// SKIP_REASON_UNREADABLE means the file could not be read at all.
const SKIP_REASON_UNREADABLE Skip_Reason = 1

// SKIP_REASON_OVERSIZED means the file is wider than the source bound.
const SKIP_REASON_OVERSIZED Skip_Reason = 2

// SKIP_REASON_BINARY means the file holds a zero byte in its opening chunk.
const SKIP_REASON_BINARY Skip_Reason = 3

// Skip_Reason is what became of one candidate. Every outcome has a name, including the
// ordinary one, so a file cannot leave the pipeline unaccounted for.
type Skip_Reason uint8

// Skip_Reason_Invariants holds an outcome to the four that exist, and witnesses each one.
func Skip_Reason_Invariants(reason Skip_Reason, namespace invariant.Namespace) {
	invariant.Tree(reason, namespace).
		Enum_4_Uint8(
			uint8(reason), uint8(SKIP_REASON_COUNTED), uint8(SKIP_REASON_UNREADABLE),
			uint8(SKIP_REASON_OVERSIZED), uint8(SKIP_REASON_BINARY)).
		Ensure()
}

// Candidate_Path is the path returned by candidate classification. It has already
// passed the walk's recognized-path filter.
type Candidate_Path Counted_Path

// Candidate_Path_Invariants checks a selected path without assigning both path-width
// witnesses to every later pipeline phase.
func Candidate_Path_Invariants(path Candidate_Path, namespace invariant.Namespace) {
	invariant.Tree(path, namespace).
		Range_Int(len(path), COUNTED_PATH_BYTES_MIN, COUNTED_PATH_BYTES_MAX).
		Ensure()
}

// Candidate_File is one selected file after reading and classification. A skipped
// candidate carries its path and the zero partition.
type Candidate_File struct {
	// Path is the selected file's path.
	Path Candidate_Path
	// Language is the counted language, or empty when the candidate was skipped.
	Language Language_Name
	// Counts is the file partition, or zero when the candidate was skipped.
	Counts File_Partition
	// Is_Test reports the source-or-test role of a counted candidate.
	Is_Test Test_File_Status
}

// Candidate_File_Invariants states a candidate result's identity and partition.
func Candidate_File_Invariants(file Candidate_File, namespace invariant.Namespace) {
	Candidate_Path_Invariants(file.Path, namespace)
	Language_Name_Invariants(file.Language, namespace)
	File_Partition_Invariants(file.Counts, namespace)
	Test_File_Status_Invariants(file.Is_Test, namespace)
}

// Candidate_Result is one selected candidate's file state and outcome.
type Candidate_Result struct {
	// File is the selected file's state after its outcome.
	File Candidate_File
	// Reason is what became of it.
	Reason Skip_Reason
}

// Candidate_Result_Invariants states one candidate's outcome.
func Candidate_Result_Invariants(result Candidate_Result, namespace invariant.Namespace) {
	Candidate_File_Invariants(result.File, namespace)
	Skip_Reason_Invariants(result.Reason, namespace)
}

// Count_Results are the per-candidate slots the workers write into, one per candidate.
type Count_Results []Candidate_Result

// Count_Results_Invariants bounds how many result slots a run allocates.
func Count_Results_Invariants(results Count_Results, namespace invariant.Namespace) {
	invariant.Tree(results, namespace).
		Range_Int(len(results), FILES_COUNT_MIN, FILES_COUNT_MAX).
		Ensure()
}

// Unreadable_Tally is the number of selected files that a read could not open.
type Unreadable_Tally File_Tally

// Unreadable_Tally_Invariants bounds an unreadable-file tally.
func Unreadable_Tally_Invariants(
	unreadable Unreadable_Tally, namespace invariant.Namespace,
) {
	invariant.Tree(unreadable, namespace).
		Range_Int(int(unreadable), FILES_COUNT_MIN, FILES_COUNT_MAX).
		Ensure()
}

// Oversized_Tally is the number of selected files past the source bound.
type Oversized_Tally File_Tally

// Oversized_Tally_Invariants bounds an oversized-file tally.
func Oversized_Tally_Invariants(
	oversized Oversized_Tally, namespace invariant.Namespace,
) {
	invariant.Tree(oversized, namespace).
		Range_Int(int(oversized), FILES_COUNT_MIN, FILES_COUNT_MAX).
		Ensure()
}

// Binary_Tally is the number of selected files that contain binary data.
type Binary_Tally File_Tally

// Binary_Tally_Invariants bounds a binary-file tally.
func Binary_Tally_Invariants(binary Binary_Tally, namespace invariant.Namespace) {
	invariant.Tree(binary, namespace).
		Range_Int(int(binary), FILES_COUNT_MIN, FILES_COUNT_MAX).
		Ensure()
}

// Past_Lines_Tally is the number of files past a language line bound.
type Past_Lines_Tally File_Tally

// Past_Lines_Tally_Invariants bounds a past-lines file tally.
func Past_Lines_Tally_Invariants(
	past_lines Past_Lines_Tally, namespace invariant.Namespace,
) {
	invariant.Tree(past_lines, namespace).
		Range_Int(int(past_lines), FILES_COUNT_MIN, FILES_COUNT_MAX).
		Ensure()
}

// Skipped is what the walk found but did not count, tallied by reason. A report carries
// it so that a file left out of the totals is stated rather than silently missing.
type Skipped struct {
	// Unreadable is how many files could not be read.
	Unreadable Unreadable_Tally
	// Oversized is how many files were wider than the source bound.
	Oversized Oversized_Tally
	// Binary is how many files held a zero byte in their opening chunk.
	Binary Binary_Tally
	// Past_Lines is how many files were left out because their own language had already
	// reached the line bound. The tree is still counted as far as it goes.
	Past_Lines Past_Lines_Tally
	// Overflow is how many recognized files the walk found past the file bound and so
	// could not take. The walk keeps going in order to count them, because a report
	// that says only "there were more" is the silence it exists to prevent.
	Overflow Dropped_Count
}

// Skipped_Invariants states each tally of uncounted files and whether the walk stopped
// short of the whole tree.
func Skipped_Invariants(skipped Skipped, namespace invariant.Namespace) {
	Unreadable_Tally_Invariants(skipped.Unreadable, namespace)
	Oversized_Tally_Invariants(skipped.Oversized, namespace)
	Binary_Tally_Invariants(skipped.Binary, namespace)
	Past_Lines_Tally_Invariants(skipped.Past_Lines, namespace)
	Dropped_Count_Invariants(skipped.Overflow, namespace)
}

// ROOT_LINE_BOUND_FILES_MIN is the fewest files that can fill one language to the
// line ceiling. Blank lines use every source byte, so no other partition uses fewer.
const ROOT_LINE_BOUND_FILES_MIN = (LINE_COUNT_MAX + FILE_BLANK_COUNT_MAX - 1) / FILE_BLANK_COUNT_MAX

// ROOT_PAST_LINES_MAX leaves room for the files that established the line ceiling.
const ROOT_PAST_LINES_MAX = FILES_COUNT_MAX - ROOT_LINE_BOUND_FILES_MIN

// Root_Skipped is the omission tally produced by one command-line root.
type Root_Skipped Skipped

// Root_Skipped_Invariants checks one root's tallies. Aggregate boundary witnesses
// belong to the report that combines roots.
func Root_Skipped_Invariants(skipped Root_Skipped, namespace invariant.Namespace) {
	invariant.Tree(skipped, namespace).
		Range_Int(int(skipped.Unreadable), FILES_COUNT_MIN, FILES_COUNT_MAX).
		Range_Int(int(skipped.Oversized), FILES_COUNT_MIN, FILES_COUNT_MAX).
		Range_Int(int(skipped.Binary), FILES_COUNT_MIN, FILES_COUNT_MAX).
		Range_Int(int(skipped.Past_Lines), FILES_COUNT_MIN, ROOT_PAST_LINES_MAX).
		Range_Int(
			int(skipped.Overflow), DROPPED_COUNT_MIN, DROPPED_COUNT_MAX).
		Ensure()
}

// Classified_Skipped is the omission tally produced by candidate classification. It
// has no overflow or line-bound fields because later phases own those omissions.
type Classified_Skipped struct {
	// Unreadable is how many selected files could not be read.
	Unreadable Unreadable_Tally
	// Oversized is how many selected files exceeded the source bound.
	Oversized Oversized_Tally
	// Binary is how many selected files held binary data.
	Binary Binary_Tally
}

// Classified_Skipped_Invariants checks the three outcomes that classification owns.
func Classified_Skipped_Invariants(
	skipped Classified_Skipped, namespace invariant.Namespace,
) {
	Unreadable_Tally_Invariants(skipped.Unreadable, namespace)
	Oversized_Tally_Invariants(skipped.Oversized, namespace)
	Binary_Tally_Invariants(skipped.Binary, namespace)
}

// Reads and classifies each candidate concurrently, dropping any unreadable or binary
// file, and returns the results in candidate order.
func count_classify(
	file_system fs.FS, candidates Candidates, classifier File_Classifier,
	concurrency Concurrency,
) (files File_Counts, skipped Classified_Skipped) {
	defer func() {
		File_Counts_Invariants(files, "count_classify.files")
		Classified_Skipped_Invariants(skipped, "count_classify.skipped")
	}()
	Candidates_Invariants(candidates, "count_classify.candidates")
	File_Classifier_Invariants(classifier, "count_classify.classifier")
	Concurrency_Invariants(concurrency, "count_classify.concurrency")
	// Each worker writes its own slot, so candidate order is preserved without locking
	// the result slice, and every candidate leaves a slot saying what became of it.
	// Nothing the walk selected disappears without being accounted for.
	results := make(Count_Results, len(candidates))
	worker_count := int(concurrency)
	if worker_count > len(candidates) {
		worker_count = len(candidates)
	}
	if worker_count < 1 {
		worker_count = 1
	}
	jobs := make(chan int)
	group := sync.WaitGroup{}
	for worker_index := 0; worker_index < worker_count; worker_index++ {
		group.Add(1)
		go count_worker(&group, jobs, results, file_system, candidates, classifier)
	}
	for index := range candidates {
		jobs <- index
	}
	close(jobs)
	group.Wait()
	for _, one := range results {
		// The slots are stated here rather than where a worker fills one: a result is
		// only whole once its worker has returned, and this is the first point that is
		// true of every slot.
		Candidate_Result_Invariants(one, "count_classify.one")
		if one.Reason == SKIP_REASON_COUNTED {
			files = append(files, File_Count{
				Path:     File_Path(one.File.Path),
				Language: one.File.Language,
				Counts:   Counts(one.File.Counts),
				Is_Test:  one.File.Is_Test,
			})
			continue
		}
		switch one.Reason {
		case SKIP_REASON_UNREADABLE:
			skipped.Unreadable++
		case SKIP_REASON_OVERSIZED:
			skipped.Oversized++
		case SKIP_REASON_BINARY:
			skipped.Binary++
		}
	}
	return files, skipped
}

// Drains the job channel, classifying each candidate into its slot.
func count_worker(
	group *sync.WaitGroup, jobs <-chan int, results Count_Results,
	file_system fs.FS, candidates Candidates, classifier File_Classifier,
) {
	Count_Results_Invariants(results, "count_worker.results")
	Candidates_Invariants(candidates, "count_worker.candidates")
	File_Classifier_Invariants(classifier, "count_worker.classifier")
	defer group.Done()
	for index := range jobs {
		one, reason := count_one(file_system, candidates[index], classifier)
		results[index] = Candidate_Result{File: one, Reason: reason}
	}
}

// Reads and classifies a single candidate, reporting why it was not counted when it
// was not. Every outcome is named, so nothing the walk selected leaves the pipeline
// without the report being able to say what became of it.
func count_one(
	file_system fs.FS, one Candidate, classifier File_Classifier,
) (file Candidate_File, reason Skip_Reason) {
	defer func() {
		Candidate_File_Invariants(file, "count_one.file")
		Skip_Reason_Invariants(reason, "count_one.reason")
	}()
	Candidate_Invariants(one, "count_one.candidate")
	File_Classifier_Invariants(classifier, "count_one.classifier")
	uncounted := Candidate_File{
		Path:     Candidate_Path(one.Path),
		Language: "",
		Counts:   File_Partition{Code: 0, Comment: 0, Blank: 0, Dropped: 0},
		Is_Test:  false,
	}
	content, read_err := fs.ReadFile(file_system, string(one.Path))
	if read_err != nil {
		return uncounted, SKIP_REASON_UNREADABLE
	}
	// A file past the source bound is not counted, but it is reported.
	if len(content) > SOURCE_BYTES_MAX {
		return uncounted, SKIP_REASON_OVERSIZED
	}
	// Detection is by extension, so a recognized extension holding binary data is not
	// counted as garbage lines — and is reported rather than vanishing.
	if content_is_binary(Source(content)) {
		return uncounted, SKIP_REASON_BINARY
	}
	// A file's line count needs no separate check: a line costs at least its newline,
	// so the byte bound above already holds a file to SOURCE_BYTES_MAX lines, which is
	// far below what a report can state.
	counts := classify(classifier, Classify_File_Input{
		Path:     Classified_Path(one.Path),
		Source:   Source(content),
		Language: one.Language,
	})
	return Candidate_File{
		Path:     Candidate_Path(one.Path),
		Language: one.Language.Name,
		Counts:   counts,
		Is_Test:  Test_File_Status(path_is_test(one.Path, one.Language)),
	}, SKIP_REASON_COUNTED
}

// Reports whether a path is test code: it lives under a test directory, or its base
// name carries the language's test prefix or infix.
func path_is_test(
	file_path Counted_Path, language Seeded_Language,
) (test Test_File_Status) {
	defer func() { Test_File_Status_Invariants(test, "path_is_test.test") }()
	Counted_Path_Invariants(file_path, "path_is_test.file_path")
	Seeded_Language_Invariants(language, "path_is_test.language")
	if path_has_test_directory(file_path) {
		return true
	}
	base := path.Base(string(file_path))
	for _, prefix := range language.Test_Prefixes {
		if strings.HasPrefix(base, prefix) {
			return true
		}
	}
	for _, infix := range language.Test_Infixes {
		if strings.Contains(base, infix) {
			return true
		}
	}
	return false
}

// Test_Directory is the conventional-test-directory state of one file path.
type Test_Directory bool

// Test_Directory_Invariants states both path states.
func Test_Directory_Invariants(value Test_Directory, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "A path contains a test directory.").
		Ensure()
}

// Reports whether any component of a path is a conventional test directory.
func path_has_test_directory(file_path Counted_Path) (found Test_Directory) {
	defer func() {
		Test_Directory_Invariants(found, "path_has_test_directory.found")
	}()
	Counted_Path_Invariants(file_path, "path_has_test_directory.file_path")
	for _, component := range strings.Split(string(file_path), "/") {
		switch component {
		case "test", "tests", "spec", "__tests__":
			return true
		}
	}
	return false
}

// Bounds how far content_is_binary scans for a NUL byte.
const BINARY_SNIFF_BYTES = 2048

// Binary_Content is the binary-content state of one source.
type Binary_Content bool

// Binary_Content_Invariants states both source states.
func Binary_Content_Invariants(value Binary_Content, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "A source contains binary data.").
		Ensure()
}

// Reports whether content holds a NUL byte in its first chunk, the cheap heuristic
// for "not text" that also guards against a no-newline blob.
func content_is_binary(content Source) (binary Binary_Content) {
	defer func() { Binary_Content_Invariants(binary, "content_is_binary.binary") }()
	Source_Invariants(content, "content_is_binary.content")
	limit_count := len(content)
	if limit_count > BINARY_SNIFF_BYTES {
		limit_count = BINARY_SNIFF_BYTES
	}
	for index := 0; index < limit_count; index++ {
		if content[index] == 0 {
			return true
		}
	}
	return false
}

// Hidden_Path is the hidden-name state of one file path.
type Hidden_Path bool

// Hidden_Path_Invariants states both file-path states.
func Hidden_Path_Invariants(value Hidden_Path, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "A file path has a hidden final name.").
		Ensure()
}

// Reports whether a path's final element begins with a dot.
func path_is_hidden(file_path File_Path) (hidden Hidden_Path) {
	defer func() { Hidden_Path_Invariants(hidden, "path_is_hidden.hidden") }()
	File_Path_Invariants(file_path, "path_is_hidden.file_path")
	base := path.Base(string(file_path))
	if len(base) == 0 {
		return false
	}
	return base[0] == '.'
}

// File_Display is the per-file display state of one rendered report.
type File_Display bool

// File_Display_Invariants states both per-file display states.
func File_Display_Invariants(value File_Display, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "Files are shown.").
		Ensure()
}

// Render_Input is a report and whether to break each language into its files.
type Render_Input struct {
	// Report is the report to render.
	Report Report
	// Show_Files lists each file indented under its language.
	Show_Files File_Display
}

// Render_Input_Invariants states the report to render and the breakdown setting.
func Render_Input_Invariants(input Render_Input, namespace invariant.Namespace) {
	Report_Invariants(input.Report, namespace)
	File_Display_Invariants(input.Show_Files, namespace)
}

// Render writes the report as an aligned table: one row per language sorted by name,
// and — when Show_Files is set — each file indented under its language. Columns are
// sized to their widest cell, so the output is stable for a given report.
func Render(output io.Writer, input Render_Input) {
	Render_Input_Invariants(input, "Render.input")
	groups := report_groups(input.Report)
	categories := report_categories(groups)
	rows := report_rows(categories, input.Show_Files)
	widths := render_widths(rows)
	rule := Table_Line(strings.Repeat("─", int(render_column_widths_total(widths))))
	render_write(output, rule)
	render_write(output, Table_Line(render_row_format(rows[0], widths)))
	render_write(output, rule)
	for _, row := range rows[1:] {
		render_write(output, Table_Line(render_row_format(row, widths)))
	}
	render_write(output, rule)
	// Everything the run did not count is reported rather than left silent. A run that
	// dropped nothing prints no section at all, so the disclosure means something.
	dropped := dropped_rows(input.Report)
	if len(dropped) == 0 {
		return
	}
	for _, row := range dropped {
		render_write(output, Table_Line(render_row_format(row, widths)))
	}
	render_write(output, rule)
}

// NONZERO_COUNT_MIN is the smallest count a report prints. A zero one prints no row at
// all, so the row builders below never receive it.
const NONZERO_COUNT_MIN = 1

// Nonzero_Count is a tally of lines a report is actually printing. It is distinct from
// Dropped_Count because the section omits a zero tally rather than printing a zero, so
// zero cannot reach the builders that take this.
type Nonzero_Count int

// Nonzero_Count_Invariants bounds a printed line tally.
func Nonzero_Count_Invariants(count Nonzero_Count, namespace invariant.Namespace) {
	invariant.Tree(count, namespace).
		Range_Int(int(count), NONZERO_COUNT_MIN, DROPPED_COUNT_MAX).
		Ensure()
}

// DROPPED_CELL_BYTES_MIN is the one-digit smallest printed nonzero tally.
const DROPPED_CELL_BYTES_MIN = 1

// DROPPED_CELL_BYTES_MAX is 99,999, the widest tally before saturation prints 100k+.
const DROPPED_CELL_BYTES_MAX = 6

// Dropped_Cell is the nonempty tally printed for lines read short.
type Dropped_Cell string

// Dropped_Cell_Invariants bounds the numeric and saturated spellings.
func Dropped_Cell_Invariants(cell Dropped_Cell, namespace invariant.Namespace) {
	invariant.Tree(cell, namespace).
		Range_Int(len(cell), DROPPED_CELL_BYTES_MIN, DROPPED_CELL_BYTES_MAX).
		Ensure()
}

// Returns the printed tally of lines read short. At the bound the count stops rising,
// so the cell says that it saturated rather than naming a total it stopped keeping.
func dropped_cell(dropped Nonzero_Count) (cell Dropped_Cell) {
	defer func() { Dropped_Cell_Invariants(cell, "dropped_cell.cell") }()
	Nonzero_Count_Invariants(dropped, "dropped_cell.dropped")
	if dropped == DROPPED_COUNT_MAX {
		return Dropped_Cell(DROPPED_SATURATED_TEXT)
	}
	return Dropped_Cell(with_thousands_separators(Line_Count(dropped)))
}

// Returns the trailing section: everything the run did not count, and why. A line read
// short, a file that could not be read, one past the source bound, one holding binary,
// and a walk that stopped at the file bound all appear here. A total that is short of
// the tree says so in the output rather than leaving the reader to notice.
func dropped_rows(report Report) (rows Dropped_Rows) {
	defer func() { Dropped_Rows_Invariants(rows, "dropped_rows.rows") }()
	Report_Invariants(report, "dropped_rows.report")
	dropped := Dropped_Count(0)
	for _, file := range report.Files {
		dropped += file.Counts.Dropped
		if dropped > DROPPED_COUNT_MAX {
			dropped = DROPPED_COUNT_MAX
		}
	}
	// Only what actually happened is listed. A tally of zero is not a disclosure, so
	// it prints nothing, and a run that dropped nothing prints no section at all.
	detail := dropped_detail_rows(report.Skipped, dropped)
	if len(detail) == 0 {
		return nil
	}
	return append(Dropped_Rows{{
		Name: "Dropped", Files: "", Lines: "", Code: "",
		Comments: "", Blanks: "", Percent: "",
	}}, detail...)
}

// Returns one row for each thing the run actually left out, and none for the rest.
func dropped_detail_rows(
	skipped Skipped, dropped Dropped_Count,
) (rows Dropped_Details) {
	defer func() { Dropped_Details_Invariants(rows, "dropped_detail_rows.rows") }()
	Skipped_Invariants(skipped, "dropped_detail_rows.skipped")
	Dropped_Count_Invariants(dropped, "dropped_detail_rows.dropped")
	if dropped > 0 {
		rows = append(rows, Render_Row{
			Name: "  lines read short", Files: "",
			Lines: Lines_Cell(dropped_cell(Nonzero_Count(dropped))),
			Code:  "", Comments: "", Blanks: "", Percent: "",
		})
	}
	append_files := func(name Row_Name, file_count File_Tally) {
		rows = append(rows, Render_Row{
			Name:  name,
			Files: File_Cell(with_thousands_separators(Line_Count(file_count))),
			Lines: "", Code: "", Comments: "", Blanks: "", Percent: "",
		})
	}
	if skipped.Unreadable > 0 {
		append_files("  files unreadable", File_Tally(skipped.Unreadable))
	}
	if skipped.Oversized > 0 {
		append_files("  files oversized", File_Tally(skipped.Oversized))
	}
	if skipped.Binary > 0 {
		append_files("  files binary", File_Tally(skipped.Binary))
	}
	if skipped.Past_Lines > 0 {
		append_files("  files past lines", File_Tally(skipped.Past_Lines))
	}
	if skipped.Overflow > 0 {
		file_cell := File_Cell(with_thousands_separators(Line_Count(skipped.Overflow)))
		if skipped.Overflow == DROPPED_COUNT_MAX {
			file_cell = DROPPED_SATURATED_TEXT
		}
		rows = append(rows, Render_Row{
			Name: "  files past bound", Files: file_cell, Lines: "",
			Code: "", Comments: "", Blanks: "", Percent: "",
		})
	}
	return rows
}

// DROPPED_DETAILS_COUNT_MIN is the empty detail of a run that left nothing out.
const DROPPED_DETAILS_COUNT_MIN = 0

// DROPPED_DETAILS_COUNT_MAX is one row for each way a run can leave something out.
const DROPPED_DETAILS_COUNT_MAX = 6

// Dropped_Details are the rows naming what a run left out, one per thing that happened.
type Dropped_Details []Render_Row

// Dropped_Details_Invariants bounds how many kinds of omission one run reports.
func Dropped_Details_Invariants(rows Dropped_Details, namespace invariant.Namespace) {
	invariant.Tree(rows, namespace).
		Range_Int(len(rows), DROPPED_DETAILS_COUNT_MIN, DROPPED_DETAILS_COUNT_MAX).
		Ensure()
}

// DROPPED_ROWS_COUNT_MIN is the absent section of a run that dropped nothing.
const DROPPED_ROWS_COUNT_MIN = 0

// DROPPED_ROWS_COUNT_MAX is the full section: its label, the lines read short, one row
// per reason a file went uncounted, and the walk-truncated row.
const DROPPED_ROWS_COUNT_MAX = 7

// DROPPED_ROWS_COUNT_ABSENT is the one width the section never has: a label with no
// row under it, since the label is printed only once there is something to say.
const DROPPED_ROWS_COUNT_ABSENT = 1

// Dropped_Rows are the trailing section's rows.
type Dropped_Rows []Render_Row

// Dropped_Rows_Invariants bounds the trailing section to its two shapes.
func Dropped_Rows_Invariants(rows Dropped_Rows, namespace invariant.Namespace) {
	invariant.Tree(rows, namespace).
		Range_Holed_Int(
			len(rows), DROPPED_ROWS_COUNT_MIN, DROPPED_ROWS_COUNT_MAX,
			DROPPED_ROWS_COUNT_ABSENT, DROPPED_ROWS_COUNT_ABSENT,
			DROPPED_ROWS_COUNT_ABSENT, DROPPED_ROWS_COUNT_ABSENT).
		Ensure()
}

// Prints one table line, trimmed of the trailing padding a label row leaves.
func render_write(output io.Writer, line Table_Line) {
	Table_Line_Invariants(line, "render_write.line")
	fmt.Fprintln(output, strings.TrimRight(string(line), " "))
}

// TABLE_WIDTH_MIN is the narrowest table, which an empty report produces: every column
// is sized by the header alone, so the width is the leading space plus "Language",
// "Files", "Lines", "Code", "Comments", "Blanks", "%Code" and their six gaps.
const TABLE_WIDTH_MIN = 54

// TABLE_WIDTH_MAX is the widest table that can actually be printed, which is narrower
// than the sum of every column's own bound. The three partitions sum to the line count,
// so at most one of code, comments, and blanks fills its column while the other two sit
// at their header labels. Code is the partition taken here because its neighbours carry
// the widest labels, which makes that the widest of the three combinations. The share
// column does reach its own bound alongside, since the grand total prints its source
// and test sub-rows even under the per-file breakdown.
const TABLE_WIDTH_MAX = 1 + NAME_WIDTH_MAX +
	2 + FILE_WIDTH_MAX + 2 + LINE_WIDTH_MAX + 2 + LINE_WIDTH_MAX +
	2 + COMMENTS_WIDTH_MAX + 2 + BLANKS_WIDTH_MAX + 2 + PERCENT_WIDTH_MAX

// TABLE_LINE_BYTES_MIN is the narrowest printed line: a row at the narrowest table,
// whose cells are all ASCII, so its byte length is the table's width.
const TABLE_LINE_BYTES_MIN = TABLE_WIDTH_MIN

// TABLE_LINE_BYTES_MAX is the widest printed line, which is a rule rather than a row:
// the rule is drawn with a three-byte box-drawing character repeated once per column,
// so its byte length is three times the width a row of the same table occupies.
const TABLE_LINE_BYTES_MAX = 3 * TABLE_WIDTH_MAX

// Row_Line is one laid-out row of the table. Its cells are all ASCII, so its byte
// length is exactly the table's width — unlike a rule, which draws the same width in
// three-byte box characters and so cannot share this bound.
type Row_Line string

// Row_Line_Invariants bounds a laid-out row's byte length.
func Row_Line_Invariants(line Row_Line, namespace invariant.Namespace) {
	invariant.Tree(line, namespace).
		Range_Int(len(line), TABLE_WIDTH_MIN, TABLE_WIDTH_MAX).
		Ensure()
}

// Table_Line is one rendered line of the table, laid out but not yet trimmed. It spans
// both a row and a rule, so its bound is the wider of the two.
type Table_Line string

// Table_Line_Invariants bounds a rendered line's width.
func Table_Line_Invariants(line Table_Line, namespace invariant.Namespace) {
	invariant.Tree(line, namespace).
		Range_Int(len(line), TABLE_LINE_BYTES_MIN, TABLE_LINE_BYTES_MAX).
		Ensure()
}

// A line partition with its file count, as serialized.
type Json_Counts struct {
	// Files is the number of files in the partition.
	Files File_Tally `json:"files"`
	// Code is the code-line count.
	Code Code_Count `json:"code"`
	// Comments is the comment-line count.
	Comments Comment_Count `json:"comments"`
	// Blanks is the blank-line count.
	Blanks Blank_Count `json:"blanks"`
}

// Json_Counts_Invariants states a serialized partition's file and line counts.
func Json_Counts_Invariants(partition Json_Counts, namespace invariant.Namespace) {
	File_Tally_Invariants(partition.Files, namespace)
	Code_Count_Invariants(partition.Code, namespace)
	Comment_Count_Invariants(partition.Comments, namespace)
	Blank_Count_Invariants(partition.Blanks, namespace)
}

// GROUP_TALLY_MIN is the one file that brought a group into being: a language group
// exists only because a file resolved to it, so it is never empty.
const GROUP_TALLY_MIN = 1

// Group_Tally is the number of files in one language group. It is distinct from
// File_Tally because a group is never empty while a partition of one — its tests, say
// — may hold nothing at all.
type Group_Tally int

// Group_Tally_Invariants bounds a language group's file count.
func Group_Tally_Invariants(tally Group_Tally, namespace invariant.Namespace) {
	invariant.Tree(tally, namespace).
		Range_Int(int(tally), GROUP_TALLY_MIN, FILES_COUNT_MAX).
		Ensure()
}

// File_Tally is a number of files, per partition or summed across a report.
type File_Tally int

// File_Tally_Invariants bounds a file tally by the same bound the walk stops at.
func File_Tally_Invariants(tally File_Tally, namespace invariant.Namespace) {
	invariant.Tree(tally, namespace).
		Range_Int(int(tally), FILES_COUNT_MIN, FILES_COUNT_MAX).
		Ensure()
}

// Source_Json_Counts is the serialized non-test partition of one language.
type Source_Json_Counts Json_Counts

// Source_Json_Counts_Invariants states all source partition values.
func Source_Json_Counts_Invariants(
	partition Source_Json_Counts, namespace invariant.Namespace,
) {
	invariant.Tree(partition, namespace).
		Range_Int(int(partition.Files), FILES_COUNT_MIN, FILES_COUNT_MAX).
		Range_Int(int(partition.Code), LINE_COUNT_MIN, LINE_COUNT_MAX).
		Range_Int(int(partition.Comments), LINE_COUNT_MIN, LINE_COUNT_MAX).
		Range_Int(int(partition.Blanks), LINE_COUNT_MIN, LINE_COUNT_MAX).
		Ensure()
}

// Test_Json_Counts is the serialized test partition of one language.
type Test_Json_Counts Json_Counts

// Test_Json_Counts_Invariants states all test partition values.
func Test_Json_Counts_Invariants(
	partition Test_Json_Counts, namespace invariant.Namespace,
) {
	invariant.Tree(partition, namespace).
		Range_Int(int(partition.Files), FILES_COUNT_MIN, FILES_COUNT_MAX).
		Range_Int(int(partition.Code), LINE_COUNT_MIN, LINE_COUNT_MAX).
		Range_Int(int(partition.Comments), LINE_COUNT_MIN, LINE_COUNT_MAX).
		Range_Int(int(partition.Blanks), LINE_COUNT_MIN, LINE_COUNT_MAX).
		Ensure()
}

// One serialized row: a language's source/test split. flatjson flattens Source and
// Tests into source_* and tests_* keys.
type Json_Language struct {
	// Name is the language's display name.
	Name Known_Name `json:"name"`
	// Category is the language's taxonomy bucket.
	Category Category `json:"category"`
	// Source is the non-test partition.
	Source Source_Json_Counts `json:"source"`
	// Tests is the test partition.
	Tests Test_Json_Counts `json:"tests"`
}

// Json_Language_Invariants states a serialized row's identity and its two partitions.
func Json_Language_Invariants(row Json_Language, namespace invariant.Namespace) {
	Known_Name_Invariants(row.Name, namespace)
	Category_Invariants(row.Category, namespace)
	Source_Json_Counts_Invariants(row.Source, namespace)
	Test_Json_Counts_Invariants(row.Tests, namespace)
}

// Builds a serialized partition from a file count and a partition.
func json_partition(files File_Tally, counts Counts) (partition Json_Counts) {
	defer func() { Json_Counts_Invariants(partition, "json_partition.partition") }()
	File_Tally_Invariants(files, "json_partition.files")
	Counts_Invariants(counts, "json_partition.counts")
	return Json_Counts{
		Files:    files,
		Code:     counts.Code,
		Comments: counts.Comment,
		Blanks:   counts.Blank,
	}
}

// Render_Json writes the report as compact flat JSON: a name-sorted array of per-language
// rows, each with its category and source/test split.
func Render_Json(output io.Writer, files File_Counts) (err error) {
	File_Counts_Invariants(files, "Render_Json.files")
	languages := []Json_Language{}
	for _, group := range report_groups(Report{Files: files}) {
		languages = append(languages, Json_Language{
			Name:     group.Name,
			Category: group.Category,
			Source: Source_Json_Counts(json_partition(
				File_Tally(group.Source_Files), Counts(group.Source))),
			Tests: Test_Json_Counts(json_partition(
				File_Tally(group.Test_Files), Counts(group.Test))),
		})
		Json_Language_Invariants(languages[len(languages)-1], "Render_Json.row")
	}
	return flatjson.Marshal_Write(output, languages)
}

// One printable table row; every cell is already a string so the header
// and the numeric rows share one width and formatting path.
type Render_Row struct {
	// Name is the language name, or an indented file path.
	Name Row_Name
	// Files is the file count, empty on a per-file row.
	Files File_Cell
	// Lines is the total line count.
	Lines Lines_Cell
	// Code is the code-line count.
	Code Code_Cell
	// Comments is the comment-line count.
	Comments Comments_Cell
	// Blanks is the blank-line count.
	Blanks Blanks_Cell
	// Percent is the row's code as a share of total code.
	Percent Percent_Cell
}

// Render_Row_Invariants states every printed cell of a table row under one namespace.
//
// The cells once carried a bound each, on the reasoning that a file tally, a line tally
// and a percentage reach different widths. That reasoning inverts under the recorder: a
// per-column bound makes each column's widest cell a demanded witness, and the widest
// line cell is an eight-figure tally — a hundred-megabyte fixture the suite cannot
// afford (measured at 8m24s for ten million lines). One namespace over one bound is
// witnessed instead by the widest cell of any column, which is an ordinary long file
// path in the label. A cell is a printed string; its width is all a row promises.
func Render_Row_Invariants(row Render_Row, namespace invariant.Namespace) {
	Row_Name_Invariants(row.Name, namespace)
	File_Cell_Invariants(row.Files, namespace)
	Lines_Cell_Invariants(row.Lines, namespace)
	Code_Cell_Invariants(row.Code, namespace)
	Comments_Cell_Invariants(row.Comments, namespace)
	Blanks_Cell_Invariants(row.Blanks, namespace)
	Percent_Cell_Invariants(row.Percent, namespace)
}

// ROW_NAME_BYTES_MIN is the narrowest label: two spaces of indent and a one-letter
// language name, as C and D and R print.
const ROW_NAME_BYTES_MIN = 3

// ROW_NAME_BYTES_MAX is the widest label: four spaces of indent and a file path at its
// own bound, as a per-file row prints.
const ROW_NAME_BYTES_MAX = FILE_PATH_BYTES_MAX + 4

// Row_Name is a table row's label: a header, a category, an indented language, or an
// indented file path.
type Row_Name string

// Row_Name_Invariants bounds a row label's width.
func Row_Name_Invariants(name Row_Name, namespace invariant.Namespace) {
	invariant.Tree(name, namespace).
		Range_Int(len(name), ROW_NAME_BYTES_MIN, ROW_NAME_BYTES_MAX).
		Ensure()
}

// FILE_CELL_BYTES_MIN is the empty tally a per-file row prints.
const FILE_CELL_BYTES_MIN = 0

// FILE_CELL_BYTES_MAX is the widest printed file tally: the file bound with its
// thousands separators, "65,535".
const FILE_CELL_BYTES_MAX = 6

// File_Cell is a table row's printed file tally.
type File_Cell string

// File_Cell_Invariants bounds a printed file tally's width.
func File_Cell_Invariants(cell File_Cell, namespace invariant.Namespace) {
	invariant.Tree(cell, namespace).
		Range_Int(len(cell), FILE_CELL_BYTES_MIN, FILE_CELL_BYTES_MAX).
		Ensure()
}

// LINE_CELL_BYTES_MIN is the empty cell a header row leaves before its label is set.
const LINE_CELL_BYTES_MIN = 0

// LINE_CELL_BYTES_MAX is the widest printed line tally: the line bound with its
// thousands separators, "99,999,999".
const LINE_CELL_BYTES_MAX = 10

// TALLY_TEXT_BYTES_MIN is the single digit a zero tally prints as. A rendered tally is
// never empty, unlike the cell it lands in, which a label row leaves blank.
const TALLY_TEXT_BYTES_MIN = 1

// Tally_Text is a tally rendered with its thousands separators.
type Tally_Text string

// Tally_Text_Invariants bounds a rendered tally's width.
func Tally_Text_Invariants(text Tally_Text, namespace invariant.Namespace) {
	invariant.Tree(text, namespace).
		Range_Int(len(text), TALLY_TEXT_BYTES_MIN, LINE_CELL_BYTES_MAX).
		Ensure()
}

// Lines_Cell is a table row's printed total-line tally.
type Lines_Cell string

// Lines_Cell_Invariants bounds a printed total-line tally's width.
func Lines_Cell_Invariants(cell Lines_Cell, namespace invariant.Namespace) {
	invariant.Tree(cell, namespace).
		Range_Int(len(cell), LINE_CELL_BYTES_MIN, LINE_CELL_BYTES_MAX).
		Ensure()
}

// Code_Cell is a table row's printed code-line tally.
type Code_Cell string

// Code_Cell_Invariants bounds a printed code-line tally's width.
func Code_Cell_Invariants(cell Code_Cell, namespace invariant.Namespace) {
	invariant.Tree(cell, namespace).
		Range_Int(len(cell), LINE_CELL_BYTES_MIN, LINE_CELL_BYTES_MAX).
		Ensure()
}

// Blanks_Cell is a table row's printed blank-line tally.
type Blanks_Cell string

// Blanks_Cell_Invariants bounds a printed blank-line tally's width.
func Blanks_Cell_Invariants(cell Blanks_Cell, namespace invariant.Namespace) {
	invariant.Tree(cell, namespace).
		Range_Int(len(cell), LINE_CELL_BYTES_MIN, LINE_CELL_BYTES_MAX).
		Ensure()
}

// COMMENTS_CELL_BYTES_MAX is the comment column's own header label, "Comments", which
// is wider than any tally the column can print. A header label is a cell like any
// other, so the widest cell of this column is the label rather than a number.
const COMMENTS_CELL_BYTES_MAX = LINE_CELL_BYTES_MAX

// Comments_Cell is a table row's printed comment tally. Its header label is the
// widest thing that the column holds.
type Comments_Cell string

// Comments_Cell_Invariants bounds a printed comment tally's width.
func Comments_Cell_Invariants(cell Comments_Cell, namespace invariant.Namespace) {
	invariant.Tree(cell, namespace).
		Range_Int(len(cell), LINE_CELL_BYTES_MIN, COMMENTS_CELL_BYTES_MAX).
		Ensure()
}

// PERCENT_CELL_BYTES_MIN is the empty share a row without a denominator prints.
const PERCENT_CELL_BYTES_MIN = 0

// PERCENT_CELL_BYTES_MAX is the widest share, "100.0%".
const PERCENT_CELL_BYTES_MAX = 6

// PERCENT_CELL_BYTES_NARROW is the narrowest share actually printed, "0.0%". A share
// is empty or at least this wide, because one decimal place and a percent sign cannot
// be spelled in fewer bytes.
const PERCENT_CELL_BYTES_NARROW = 4

// PERCENT_CELL_BYTES_TENS is the two-digit share, "12.3%".
const PERCENT_CELL_BYTES_TENS = 5

// Percent_Cell is a table row's printed share of code.
type Percent_Cell string

// Percent_Cell_Invariants holds a printed share's width to the four a one-decimal
// percentage can occupy, and witnesses each width.
func Percent_Cell_Invariants(cell Percent_Cell, namespace invariant.Namespace) {
	invariant.Tree(cell, namespace).
		Enum_4_Int(
			len(cell), PERCENT_CELL_BYTES_MIN, PERCENT_CELL_BYTES_NARROW,
			PERCENT_CELL_BYTES_TENS, PERCENT_CELL_BYTES_MAX).
		Ensure()
}

// Group_Counts is the complete line partition of one language group.
type Group_Counts Counts

// Group_Counts_Invariants states all four group partition values.
func Group_Counts_Invariants(counts Group_Counts, namespace invariant.Namespace) {
	invariant.Tree(counts, namespace).
		Range_Int(int(counts.Code), LINE_COUNT_MIN, LINE_COUNT_MAX).
		Range_Int(int(counts.Comment), LINE_COUNT_MIN, LINE_COUNT_MAX).
		Range_Int(int(counts.Blank), LINE_COUNT_MIN, LINE_COUNT_MAX).
		Range_Int(
			int(counts.Dropped), DROPPED_COUNT_MIN, DROPPED_COUNT_MAX).
		Ensure()
}

// Source_Counts is the non-test line partition of one language group.
type Source_Counts Counts

// Source_Counts_Invariants states all four source partition values.
func Source_Counts_Invariants(counts Source_Counts, namespace invariant.Namespace) {
	invariant.Tree(counts, namespace).
		Range_Int(int(counts.Code), LINE_COUNT_MIN, LINE_COUNT_MAX).
		Range_Int(int(counts.Comment), LINE_COUNT_MIN, LINE_COUNT_MAX).
		Range_Int(int(counts.Blank), LINE_COUNT_MIN, LINE_COUNT_MAX).
		Range_Int(
			int(counts.Dropped), DROPPED_COUNT_MIN, DROPPED_COUNT_MAX).
		Ensure()
}

// Test_Counts is the test line partition of one language group.
type Test_Counts Counts

// Test_Counts_Invariants states all four test partition values.
func Test_Counts_Invariants(counts Test_Counts, namespace invariant.Namespace) {
	invariant.Tree(counts, namespace).
		Range_Int(int(counts.Code), LINE_COUNT_MIN, LINE_COUNT_MAX).
		Range_Int(int(counts.Comment), LINE_COUNT_MIN, LINE_COUNT_MAX).
		Range_Int(int(counts.Blank), LINE_COUNT_MIN, LINE_COUNT_MAX).
		Range_Int(
			int(counts.Dropped), DROPPED_COUNT_MIN, DROPPED_COUNT_MAX).
		Ensure()
}

// Source_File_Tally is the non-test file count of one language group.
type Source_File_Tally File_Tally

// Source_File_Tally_Invariants bounds the source file count.
func Source_File_Tally_Invariants(
	tally Source_File_Tally, namespace invariant.Namespace,
) {
	invariant.Tree(tally, namespace).
		Range_Int(int(tally), FILES_COUNT_MIN, FILES_COUNT_MAX).
		Ensure()
}

// Test_File_Tally is the test file count of one language group.
type Test_File_Tally File_Tally

// Test_File_Tally_Invariants bounds the test file count.
func Test_File_Tally_Invariants(tally Test_File_Tally, namespace invariant.Namespace) {
	invariant.Tree(tally, namespace).
		Range_Int(int(tally), FILES_COUNT_MIN, FILES_COUNT_MAX).
		Ensure()
}

// One language's files and their summed partition.
type Language_Group struct {
	// Name is the language's display name.
	Name Known_Name
	// Category is the language's taxonomy bucket.
	Category Category
	// Files is the number of files in the group.
	Files Group_Tally
	// Counts is the group's summed line partition.
	Counts Group_Counts
	// Source_Files is the number of non-test files in the group.
	Source_Files Source_File_Tally
	// Source is the summed partition of the group's non-test files.
	Source Source_Counts
	// Test_Files is the number of test files in the group.
	Test_Files Test_File_Tally
	// Test is the summed partition of the group's test files.
	Test Test_Counts
	// Members are the group's files in report order.
	Members Group_Members
}

// Language_Group_Invariants states a group's identity, its three tallies, its three
// partitions, and the files it holds.
func Language_Group_Invariants(group Language_Group, namespace invariant.Namespace) {
	Known_Name_Invariants(group.Name, namespace)
	Category_Invariants(group.Category, namespace)
	Group_Tally_Invariants(group.Files, namespace)
	Group_Counts_Invariants(group.Counts, namespace)
	Source_File_Tally_Invariants(group.Source_Files, namespace)
	Source_Counts_Invariants(group.Source, namespace)
	Test_File_Tally_Invariants(group.Test_Files, namespace)
	Test_Counts_Invariants(group.Test, namespace)
	Group_Members_Invariants(group.Members, namespace)
}

// KNOWN_NAME_BYTES_MIN is the shortest seeded display name, of which C and D and R are
// each one byte.
const KNOWN_NAME_BYTES_MIN = 1

// Known_Name is the display name of a language a file actually resolved to. It is
// distinct from Language_Name because the unrecognized lookup yields the empty name
// while a group, which exists only because a file resolved into it, never can.
type Known_Name string

// Known_Name_Invariants bounds a resolved language's display name.
func Known_Name_Invariants(name Known_Name, namespace invariant.Namespace) {
	invariant.Tree(name, namespace).
		Range_Int(len(name), KNOWN_NAME_BYTES_MIN, LANGUAGE_NAME_BYTES_MAX).
		Ensure()
}

// GROUP_MEMBERS_COUNT_MIN is the one file that brought the group into being.
const GROUP_MEMBERS_COUNT_MIN = 1

// Group_Members are one language group's files, in report order.
type Group_Members []File_Count

// Group_Members_Invariants bounds a group's membership.
func Group_Members_Invariants(members Group_Members, namespace invariant.Namespace) {
	invariant.Tree(members, namespace).
		Range_Int(len(members), GROUP_MEMBERS_COUNT_MIN, FILES_COUNT_MAX).
		Ensure()
}

// CATEGORY_BYTES_MIN is the shortest taxonomy bucket, "Other" and "Query" and "Shell".
const CATEGORY_BYTES_MIN = 5

// CATEGORY_BYTES_MAX is the longest taxonomy bucket, "Dynamically Typed".
const CATEGORY_BYTES_MAX = 17

// Category is a language's taxonomy bucket. Every language resolves to one, so unlike
// the serialized form it is never empty.
type Category string

// Category_Invariants bounds a taxonomy bucket's byte length.
func Category_Invariants(category Category, namespace invariant.Namespace) {
	invariant.Tree(category, namespace).
		Range_Int(len(category), CATEGORY_BYTES_MIN, CATEGORY_BYTES_MAX).
		Ensure()
}

// LANGUAGE_GROUPS_COUNT_MIN is the empty report: no file, so no group.
const LANGUAGE_GROUPS_COUNT_MIN = 0

// LANGUAGE_GROUPS_COUNT_MAX is one group per seeded language, the most a report can
// hold since a group is keyed by display name.
const LANGUAGE_GROUPS_COUNT_MAX = 73

// Language_Groups are a report's per-language groups in name order.
type Language_Groups []Language_Group

// Language_Groups_Invariants bounds how many languages a report holds.
func Language_Groups_Invariants(groups Language_Groups, namespace invariant.Namespace) {
	invariant.Tree(groups, namespace).
		Range_Int(len(groups), LANGUAGE_GROUPS_COUNT_MIN, LANGUAGE_GROUPS_COUNT_MAX).
		Ensure()
}

// Folds a report's files into per-language groups sorted by name, splitting each
// group's partition into source and test and preserving its files in report order.
func report_groups(report Report) (groups Language_Groups) {
	defer func() { Language_Groups_Invariants(groups, "report_groups.groups") }()
	Report_Invariants(report, "report_groups.report")
	position_of := map[Language_Name]int{}
	add_counts := func(into *Counts, more File_Partition) {
		into.Code += more.Code
		into.Comment += more.Comment
		into.Blank += more.Blank
		into.Dropped += more.Dropped
		if into.Dropped > DROPPED_COUNT_MAX {
			into.Dropped = DROPPED_COUNT_MAX
		}
	}
	for _, file := range report.Files {
		position, seen := position_of[file.Language]
		if !seen {
			position = len(groups)
			position_of[file.Language] = position
			groups = append(groups, Language_Group{
				Name:     Known_Name(file.Language),
				Category: language_category(Known_Name(file.Language)),
			})
		}
		group := &groups[position]
		group.Files++
		group_counts := Counts(group.Counts)
		add_counts(&group_counts, File_Partition(file.Counts))
		group.Counts = Group_Counts(group_counts)
		if file.Is_Test {
			group.Test_Files++
			test_counts := Counts(group.Test)
			add_counts(&test_counts, File_Partition(file.Counts))
			group.Test = Test_Counts(test_counts)
		} else {
			group.Source_Files++
			source_counts := Counts(group.Source)
			add_counts(&source_counts, File_Partition(file.Counts))
			group.Source = Source_Counts(source_counts)
		}
		group.Members = append(group.Members, file)
	}
	slices.SortFunc(groups, func(a Language_Group, b Language_Group) (order int) {
		return strings.Compare(string(a.Name), string(b.Name))
	})
	return groups
}

// Returns a language's taxonomy category by display name. This switch is the single
// place the classification lives.
func language_category(name Known_Name) (category Category) {
	defer func() { Category_Invariants(category, "language_category.category") }()
	Known_Name_Invariants(name, "language_category.name")
	switch name {
	case "C", "C++", "Rust", "Zig", "Odin", "Ada", "Fortran",
		"Pascal", "Arduino", "Solidity":
		return "Systems"
	case "Go", "Java", "C#", "Kotlin", "Scala", "Dart", "Crystal", "Nim",
		"D", "Swift", "Objective-C", "Haskell", "OCaml",
		"F#", "Visual Basic":
		return "Managed"
	case "Python", "Ruby", "JavaScript", "TypeScript", "Lua", "Perl", "PHP",
		"R", "Tcl", "Julia", "Groovy", "Elixir", "Erlang", "Clojure",
		"Scheme", "Common Lisp", "Racket", "Emacs Lisp":
		return "Dynamically Typed"
	case "Shell", "PowerShell", "Fish", "Nushell":
		return "Shell"
	case "HTML", "XML", "CSS", "SCSS", "LESS", "Markdown", "YAML", "TOML",
		"JSONC", "XAML", "XSLT", "Vue", "Svelte", "Astro", "Protobuf",
		"Thrift", "TeX":
		return "Markup & Data"
	case "Makefile", "Dockerfile", "CMake", "HCL", "Nix":
		return "Build & Config"
	case "SQL":
		return "Query"
	case "Verilog", "GLSL", "HLSL":
		return "Hardware"
	}
	return "Other"
}

// Returns the fixed display order of the categories.
func category_order() (order Categories) {
	defer func() { Categories_Invariants(order, "category_order.order") }()
	return Categories{
		"Systems", "Managed", "Dynamically Typed", "Shell",
		"Markup & Data", "Build & Config", "Query", "Hardware", "Other",
	}
}

// CATEGORIES_COUNT is the fixed number of taxonomy buckets: the order is a declared
// list, not something a report varies, so the size is a point, not an interval.
const CATEGORIES_COUNT = 9

// Categories is the fixed display order of the taxonomy buckets.
type Categories []Category

// Categories_Invariants pins the taxonomy to its declared size.
func Categories_Invariants(order Categories, namespace invariant.Namespace) {
	invariant.Always(
		len(order) == CATEGORIES_COUNT, "The taxonomy always holds its declared size.")
}

// A category and the language groups it holds, in name order.
type Category_Group struct {
	// Name is the category's display name.
	Name Category
	// Languages are the category's language groups in name order.
	Languages Category_Languages
}

// Category_Group_Invariants states a bucket's name and the groups it holds.
func Category_Group_Invariants(group Category_Group, namespace invariant.Namespace) {
	Category_Invariants(group.Name, namespace)
	Category_Languages_Invariants(group.Languages, namespace)
}

// CATEGORY_LANGUAGES_COUNT_MIN is a bucket no counted file fell into.
const CATEGORY_LANGUAGES_COUNT_MIN = 0

// CATEGORY_LANGUAGES_COUNT_MAX is the largest bucket in the taxonomy, the dynamically
// typed one. A bucket is a partition of the languages, so it can never hold them all —
// which is why this is its own bound rather than the whole-report one.
const CATEGORY_LANGUAGES_COUNT_MAX = 18

// Category_Languages are one taxonomy bucket's language groups in name order.
type Category_Languages []Language_Group

// Category_Languages_Invariants bounds how many languages one bucket holds.
func Category_Languages_Invariants(
	languages Category_Languages, namespace invariant.Namespace,
) {
	invariant.Tree(languages, namespace).
		Range_Int(
			len(languages), CATEGORY_LANGUAGES_COUNT_MIN,
			CATEGORY_LANGUAGES_COUNT_MAX).
		Ensure()
}

// Category_Groups are the taxonomy buckets of a rendered report.
type Category_Groups []Category_Group

// Category_Groups_Invariants pins the bucket list to the declared taxonomy size.
func Category_Groups_Invariants(
	categories Category_Groups, namespace invariant.Namespace,
) {
	invariant.Always(
		len(categories) == CATEGORIES_COUNT,
		"A rendered bucket list always holds the declared taxonomy size.")
}

// Buckets the name-sorted language groups into categories in the fixed display order.
func report_categories(groups Language_Groups) (categories Category_Groups) {
	defer func() {
		Category_Groups_Invariants(categories, "report_categories.categories")
	}()
	Language_Groups_Invariants(groups, "report_categories.groups")
	position_of := map[Category]int{}
	for _, name := range category_order() {
		position_of[name] = len(categories)
		categories = append(categories, Category_Group{
			Name:      name,
			Languages: nil,
		})
	}
	for _, group := range groups {
		position := position_of[group.Category]
		categories[position].Languages = append(categories[position].Languages, group)
	}
	return categories
}

// Builds the header, then each non-empty category: a label row followed by its
// languages, each with its files (show_files) or its source/test split.
func report_rows(categories Category_Groups, show_files File_Display) (rows Render_Rows) {
	defer func() { Render_Rows_Invariants(rows, "report_rows.rows") }()
	Category_Groups_Invariants(categories, "report_rows.categories")
	for _, one := range categories {
		Category_Group_Invariants(one, "report_rows.category")
	}
	File_Display_Invariants(show_files, "report_rows.show_files")
	rows = Render_Rows{{
		Name: "Language", Files: "Files", Lines: "Lines",
		Code: "Code", Comments: "Comments", Blanks: "Blanks", Percent: "%Code",
	}}
	for _, category := range categories {
		if len(category.Languages) == 0 {
			continue
		}
		rows = append(rows, Render_Row{
			Name:     Row_Name(category.Name),
			Files:    "",
			Lines:    "",
			Code:     "",
			Comments: "",
			Blanks:   "",
			Percent:  "",
		})
		for _, group := range category.Languages {
			rows = append(rows, report_language_rows(group, show_files)...)
		}
	}
	return rows
}

// Appends a language's indented row, then its files (with show_files) or its source and
// test sub-rows.
func report_language_rows(
	group Language_Group, show_files File_Display,
) (output Language_Rows) {
	defer func() { Language_Rows_Invariants(output, "report_language_rows.output") }()
	Language_Group_Invariants(group, "report_language_rows.group")
	File_Display_Invariants(show_files, "report_language_rows.show_files")
	group_counts := Counts(group.Counts)
	output = Language_Rows{{
		Name:  Row_Name("  " + group.Name),
		Files: File_Cell(with_thousands_separators(Line_Count(group.Files))),
		Lines: Lines_Cell(with_thousands_separators(counts_lines(group_counts))),
		Code: Code_Cell(
			with_thousands_separators(Line_Count(group_counts.Code))),
		Comments: Comments_Cell(
			with_thousands_separators(Line_Count(group_counts.Comment))),
		Blanks: Blanks_Cell(
			with_thousands_separators(Line_Count(group_counts.Blank))),
		Percent: "",
	}}
	if show_files {
		output = append(output, split_rows(&Split_Rows_Input{
			Source_Files: group.Source_Files,
			Source:       group.Source,
			Test_Files:   group.Test_Files,
			Test:         group.Test,
		})...)
		for _, member := range group.Members {
			member_counts := Counts(member.Counts)
			output = append(output, Render_Row{
				Name: Row_Name("    " + member.Path), Files: "",
				Lines: Lines_Cell(
					with_thousands_separators(counts_lines(member_counts))),
				Code: Code_Cell(
					with_thousands_separators(Line_Count(member_counts.Code))),
				Comments: Comments_Cell(with_thousands_separators(
					Line_Count(member_counts.Comment))),
				Blanks: Blanks_Cell(with_thousands_separators(
					Line_Count(member_counts.Blank))),
				Percent: "",
			})
		}
		return output
	}
	return append(output, split_rows(&Split_Rows_Input{
		Source_Files: group.Source_Files,
		Source:       group.Source,
		Test_Files:   group.Test_Files,
		Test:         group.Test,
	})...)
}

// LANGUAGE_ROWS_COUNT_MIN is one language's own row, which it always carries.
const LANGUAGE_ROWS_COUNT_MIN = 1

// LANGUAGE_ROWS_COUNT_MAX is the language row, its source/test pair, and one row per
// file, which a one-language per-file breakdown containing a test reaches.
const LANGUAGE_ROWS_COUNT_MAX = 3 + FILES_COUNT_MAX

// Language_Rows are the rows one language contributes to the table. They are returned
// rather than appended to the caller's accumulator, so the width they can reach is a
// property of one language rather than of wherever in the table it landed.
type Language_Rows []Render_Row

// Language_Rows_Invariants bounds one language's contribution to the table.
func Language_Rows_Invariants(rows Language_Rows, namespace invariant.Namespace) {
	invariant.Tree(rows, namespace).
		Range_Int(len(rows), LANGUAGE_ROWS_COUNT_MIN, LANGUAGE_ROWS_COUNT_MAX).
		Ensure()
}

// Carries split_rows's accumulator and the source and test partitions.
type Split_Rows_Input struct {
	// Source_Files is the non-test file count.
	Source_Files Source_File_Tally
	// Source is the non-test partition.
	Source Source_Counts
	// Test_Files is the test file count.
	Test_Files Test_File_Tally
	// Test is the test partition.
	Test Test_Counts
}

// Split_Rows_Input_Invariants states the accumulator and both partitions.
func Split_Rows_Input_Invariants(
	input Split_Rows_Input, namespace invariant.Namespace,
) {
	Source_File_Tally_Invariants(input.Source_Files, namespace)
	Source_Counts_Invariants(input.Source, namespace)
	Test_File_Tally_Invariants(input.Test_Files, namespace)
	Test_Counts_Invariants(input.Test, namespace)
}

// SPLIT_ROW_INDENT is the leading whitespace of a source or test row. It is a
// constant rather than a parameter because only one indent occurs: the split rows sit
// under a language row, and nothing else splits.
const SPLIT_ROW_INDENT = "    "

// RENDER_ROWS_COUNT_MIN is the header alone, which every table carries.
const RENDER_ROWS_COUNT_MIN = 1

// CATEGORIES_SHOWN_MAX is how many taxonomy labels a table can print. It is one fewer
// than the taxonomy itself: every seeded language resolves to a named bucket, so the
// catch-all is never printed, and an empty bucket is skipped.
const CATEGORIES_SHOWN_MAX = CATEGORIES_COUNT - 1

// RENDER_ROWS_COUNT_MAX is the header, every taxonomy label, a row per language, and —
// under the per-file breakdown — a row per counted file.
const RENDER_ROWS_COUNT_MAX = 1 + CATEGORIES_SHOWN_MAX +
	3*LANGUAGE_GROUPS_COUNT_MAX + FILES_COUNT_MAX

// RENDER_ROWS_COUNT_ABSENT is the shape that never occurs: a table is the header alone
// or the header plus a taxonomy label and at least the language row beneath it, so two
// rows is unreachable.
const RENDER_ROWS_COUNT_ABSENT = 2

// Render_Rows are the printable rows of a table, in print order.
type Render_Rows []Render_Row

// Render_Rows_Invariants bounds how many rows a table prints.
func Render_Rows_Invariants(rows Render_Rows, namespace invariant.Namespace) {
	invariant.Tree(rows, namespace).
		Range_Holed_Int(
			len(rows), RENDER_ROWS_COUNT_MIN, RENDER_ROWS_COUNT_MAX,
			RENDER_ROWS_COUNT_ABSENT, RENDER_ROWS_COUNT_ABSENT,
			RENDER_ROWS_COUNT_ABSENT, RENDER_ROWS_COUNT_ABSENT).
		Ensure()
}

// Appends indented source and test sub-rows, but only when test files are present — a
// language without tests shows just its single total row. The %Code on a sub-row is its
// share of the group's own code, so source and tests sum to 100% independent of how
// large the group is relative to the whole.
func split_rows(input *Split_Rows_Input) (split Split_Row_Pair) {
	defer func() { Split_Row_Pair_Invariants(split, "split_rows.split") }()
	Split_Rows_Input_Invariants(*input, "split_rows.input")
	if input.Test_Files == 0 {
		return nil
	}
	own_code := input.Source.Code + input.Test.Code
	row_for := func(
		name Row_Name, files File_Tally, counts Counts,
	) (row Render_Row) {
		percent := Percent_Cell("")
		if own_code > 0 {
			share := float64(counts.Code) / float64(own_code) * 100
			percent = Percent_Cell(fmt.Sprintf("%.1f%%", share))
		}
		return Render_Row{
			Name:  name,
			Files: File_Cell(with_thousands_separators(Line_Count(files))),
			Lines: Lines_Cell(with_thousands_separators(counts_lines(counts))),
			Code: Code_Cell(
				with_thousands_separators(Line_Count(counts.Code))),
			Comments: Comments_Cell(
				with_thousands_separators(Line_Count(counts.Comment))),
			Blanks: Blanks_Cell(
				with_thousands_separators(Line_Count(counts.Blank))),
			Percent: percent,
		}
	}
	return Split_Row_Pair{
		row_for(
			Row_Name(SPLIT_ROW_INDENT+"source"),
			File_Tally(input.Source_Files), Counts(input.Source)),
		row_for(
			Row_Name(SPLIT_ROW_INDENT+"tests"),
			File_Tally(input.Test_Files), Counts(input.Test)),
	}
}

// SPLIT_ROWS_COUNT_MIN is the empty split a group without tests prints.
const SPLIT_ROWS_COUNT_MIN = 0

// SPLIT_ROWS_COUNT_MAX is the source and test pair, which always appear together.
const SPLIT_ROWS_COUNT_MAX = 2

// Split_Row_Pair are the source and test rows printed under a group, or nothing.
type Split_Row_Pair []Render_Row

// Split_Row_Pair_Invariants pins the split to the two shapes it takes, and witnesses
// each shape. The lone sub-row never occurs: the split is a pair or it is nothing,
// since a group with tests has a source side even when that side is empty.
func Split_Row_Pair_Invariants(rows Split_Row_Pair, namespace invariant.Namespace) {
	invariant.Tree(rows, namespace).
		Enum_Int(len(rows), SPLIT_ROWS_COUNT_MIN, SPLIT_ROWS_COUNT_MAX).
		Ensure()
}

// The printed width of each table column.
type Render_Column_Widths struct {
	// Name is the width of the language and file-path column.
	Name Name_Width
	// Files is the width of the file-count column.
	Files File_Width
	// Lines is the width of the line-count column.
	Lines Lines_Width
	// Code is the width of the code-count column.
	Code Code_Width
	// Comments is the width of the comment-count column.
	Comments Comments_Width
	// Blanks is the width of the blank-count column.
	Blanks Blanks_Width
	// Percent is the width of the code-share column.
	Percent Percent_Width
}

// Render_Column_Widths_Invariants states each column's printed width. The four kinds of
// column carry four bounds rather than one, because a column is only ever as wide as
// its widest cell and a file tally, a line tally, a share, and a label reach different
// widths — one shared bound would state a width three of them can never occupy.
func Render_Column_Widths_Invariants(
	widths Render_Column_Widths, namespace invariant.Namespace,
) {
	Name_Width_Invariants(widths.Name, namespace)
	File_Width_Invariants(widths.Files, namespace)
	Lines_Width_Invariants(widths.Lines, namespace)
	Code_Width_Invariants(widths.Code, namespace)
	Comments_Width_Invariants(widths.Comments, namespace)
	Blanks_Width_Invariants(widths.Blanks, namespace)
	Percent_Width_Invariants(widths.Percent, namespace)
}

// NAME_WIDTH_MIN is the header's own label, "Language", which every table carries and
// which no narrower report can shrink the column below.
const NAME_WIDTH_MIN = 8

// NAME_WIDTH_MAX is the widest label column, which a per-file row at the path bound
// reaches.
const NAME_WIDTH_MAX = ROW_NAME_BYTES_MAX

// Name_Width is the printed width of the label column.
type Name_Width int

// Name_Width_Invariants bounds the label column's width.
func Name_Width_Invariants(width Name_Width, namespace invariant.Namespace) {
	invariant.Tree(width, namespace).
		Range_Int(int(width), NAME_WIDTH_MIN, NAME_WIDTH_MAX).
		Ensure()
}

// FILE_WIDTH_MIN is the header's own label, "Files", which is also as wide as the
// widest tally the column can print, so the column is a fixed width.
const FILE_WIDTH_MIN = 5

// FILE_WIDTH_MAX is the widest file column, the file bound with its separators.
const FILE_WIDTH_MAX = FILE_CELL_BYTES_MAX

// File_Width is the printed width of the file-tally column.
type File_Width int

// File_Width_Invariants holds the file-tally column's width to the two it takes, and
// witnesses each one.
func File_Width_Invariants(width File_Width, namespace invariant.Namespace) {
	invariant.Tree(width, namespace).
		Enum_Int(int(width), FILE_WIDTH_MIN, FILE_WIDTH_MAX).
		Ensure()
}

// LINE_WIDTH_MAX is the widest line column, the line bound with its separators. The
// four line columns share this ceiling but not their floors: each is at least as wide
// as its own header label, and those labels differ, so each carries its own type. A
// shared floor would state a width three of the four can never occupy.
const LINE_WIDTH_MAX = LINE_CELL_BYTES_MAX

// LINES_WIDTH_MIN is the header's own label, "Lines".
const LINES_WIDTH_MIN = 5

// Lines_Width is the printed width of the total-line column.
type Lines_Width int

// Lines_Width_Invariants bounds the total-line column's width.
func Lines_Width_Invariants(width Lines_Width, namespace invariant.Namespace) {
	invariant.Tree(width, namespace).
		Range_Int(int(width), LINES_WIDTH_MIN, LINE_WIDTH_MAX).
		Ensure()
}

// CODE_WIDTH_MIN is the header's own label, "Code".
const CODE_WIDTH_MIN = 4

// Code_Width is the printed width of the code-line column.
type Code_Width int

// Code_Width_Invariants bounds the code-line column's width.
func Code_Width_Invariants(width Code_Width, namespace invariant.Namespace) {
	invariant.Tree(width, namespace).
		Range_Int(int(width), CODE_WIDTH_MIN, LINE_WIDTH_MAX).
		Ensure()
}

// COMMENTS_WIDTH_MIN is the header's own label, "Comments".
const COMMENTS_WIDTH_MIN = 8

// COMMENTS_WIDTH_SEVEN_FIGURE is the width of a seven-figure tally with its thousands
// separators, "9,999,999" — the one width between the header label and the line bound.
const COMMENTS_WIDTH_SEVEN_FIGURE = 9

// COMMENTS_WIDTH_MAX is that same label: it is wider than any tally the column can
// print, so this column never grows past its header.
const COMMENTS_WIDTH_MAX = LINE_WIDTH_MAX

// Comments_Width is the printed width of the comment-line column.
type Comments_Width int

// Comments_Width_Invariants holds the comment-line column's width to the three it
// takes, and witnesses each one.
func Comments_Width_Invariants(width Comments_Width, namespace invariant.Namespace) {
	invariant.Tree(width, namespace).
		Enum_3_Int(
			int(width), COMMENTS_WIDTH_MIN, COMMENTS_WIDTH_SEVEN_FIGURE,
			COMMENTS_WIDTH_MAX).
		Ensure()
}

// BLANKS_WIDTH_MIN is the header's own label, "Blanks".
const BLANKS_WIDTH_MIN = 6

// BLANKS_WIDTH_MAX is the widest blank column: the line bound with its separators,
// which is exactly as wide as the header label, so this column is fixed too.
const BLANKS_WIDTH_MAX = LINE_CELL_BYTES_MAX

// Blanks_Width is the printed width of the blank-line column.
type Blanks_Width int

// Blanks_Width_Invariants bounds the blank-line column's width.
func Blanks_Width_Invariants(width Blanks_Width, namespace invariant.Namespace) {
	invariant.Tree(width, namespace).
		Range_Int(int(width), BLANKS_WIDTH_MIN, BLANKS_WIDTH_MAX).
		Ensure()
}

// PERCENT_WIDTH_MIN is the header's own label, "%Code".
const PERCENT_WIDTH_MIN = 5

// PERCENT_WIDTH_MAX is the widest share column, "100.0%".
const PERCENT_WIDTH_MAX = PERCENT_CELL_BYTES_MAX

// Percent_Width is the printed width of the code-share column.
type Percent_Width int

// Percent_Width_Invariants holds the share column's width to the two it takes, and
// witnesses each one.
func Percent_Width_Invariants(width Percent_Width, namespace invariant.Namespace) {
	invariant.Tree(width, namespace).
		Enum_Int(int(width), PERCENT_WIDTH_MIN, PERCENT_WIDTH_MAX).
		Ensure()
}

// COLUMN_WIDTH_MIN is the empty column: a width is the widest cell beneath it, and a
// column whose every cell is blank — the percentage column of a table without tests —
// is zero wide before its header is measured in.
const COLUMN_WIDTH_MIN = 4

// COLUMN_WIDTH_MAX is the widest column, which is the label column holding an indented
// file path. Every column shares this bound rather than carrying its own, for the same
// reason the cells do: a per-column maximum demands a witness only that column can
// produce, and for a line column that witness is an eight-figure tally.
const COLUMN_WIDTH_MAX = COMMENTS_CELL_BYTES_MAX

// Column_Width is the width the right-justifying padder lays a numeric cell out to.
type Column_Width int

// Column_Width_Invariants bounds the width a numeric cell is padded to.
func Column_Width_Invariants(width Column_Width, namespace invariant.Namespace) {
	invariant.Tree(width, namespace).
		Range_Int(int(width), COLUMN_WIDTH_MIN, COLUMN_WIDTH_MAX).
		Ensure()
}

// Table_Width is the printed width of a whole row, which the rules match. It is wider
// than any single column, so it carries the line's bound rather than a column's.
type Table_Width int

// Table_Width_Invariants bounds a whole row's printed width in characters, which is
// what the rule is drawn to and what the columns sum to.
func Table_Width_Invariants(width Table_Width, namespace invariant.Namespace) {
	invariant.Tree(width, namespace).
		Range_Int(int(width), TABLE_WIDTH_MIN, TABLE_WIDTH_MAX).
		Ensure()
}

// Sizes each column to its widest cell across all rows.
func render_widths(rows Render_Rows) (widths Render_Column_Widths) {
	defer func() { Render_Column_Widths_Invariants(widths, "render_widths.widths") }()
	Render_Rows_Invariants(rows, "render_widths.rows")
	// The widths are widened in place rather than through a helper: a half-built
	// accumulator is below every column's floor, and a helper would have to state it.
	for _, row := range rows {
		widths.Name = max(widths.Name, Name_Width(len(row.Name)))
		widths.Files = max(widths.Files, File_Width(len(row.Files)))
		widths.Lines = max(widths.Lines, Lines_Width(len(row.Lines)))
		widths.Code = max(widths.Code, Code_Width(len(row.Code)))
		widths.Comments = max(widths.Comments, Comments_Width(len(row.Comments)))
		widths.Blanks = max(widths.Blanks, Blanks_Width(len(row.Blanks)))
		widths.Percent = max(widths.Percent, Percent_Width(len(row.Percent)))
	}
	return widths
}

// Returns the printed character width of any row, which the rules match. Every cell
// is ASCII, so character width equals byte length.
func render_column_widths_total(widths Render_Column_Widths) (width Table_Width) {
	defer func() {
		Table_Width_Invariants(width, "render_column_widths_total.width")
	}()
	Render_Column_Widths_Invariants(widths, "render_column_widths_total.widths")
	return Table_Width(1) + Table_Width(widths.Name) +
		2 + Table_Width(widths.Files) +
		2 + Table_Width(widths.Lines) +
		2 + Table_Width(widths.Code) +
		2 + Table_Width(widths.Comments) +
		2 + Table_Width(widths.Blanks) +
		2 + Table_Width(widths.Percent)
}

// Lays out one row: a leading space, the left-justified name, then each
// right-justified numeric column behind a two-space gap.
func render_row_format(row Render_Row, widths Render_Column_Widths) (line Row_Line) {
	defer func() { Row_Line_Invariants(line, "render_row_format.line") }()
	Render_Row_Invariants(row, "render_row_format.row")
	Render_Column_Widths_Invariants(widths, "render_row_format.widths")
	return Row_Line(" " + string(pad_right(row.Name, widths.Name)) +
		"  " + string(pad_left(Cell_Text(row.Files), Column_Width(widths.Files))) +
		"  " + string(pad_left(Cell_Text(row.Lines), Column_Width(widths.Lines))) +
		"  " + string(pad_left(Cell_Text(row.Code), Column_Width(widths.Code))) +
		"  " + string(pad_left(
		Cell_Text(row.Comments), Column_Width(widths.Comments))) +
		"  " + string(pad_left(Cell_Text(row.Blanks), Column_Width(widths.Blanks))) +
		"  " + string(pad_left(
		Cell_Text(row.Percent), Column_Width(widths.Percent))))
}

// PADDED_CELL_BYTES_MIN is a cell laid out to the narrowest numeric column, since a
// padded cell is exactly its column's width.
const PADDED_CELL_BYTES_MIN = COLUMN_WIDTH_MIN

// PADDED_CELL_BYTES_MAX is the widest padded numeric cell, the line column at its bound.
const PADDED_CELL_BYTES_MAX = COLUMN_WIDTH_MAX

// Padded_Cell is one numeric cell laid out to its column's width.
type Padded_Cell string

// Padded_Cell_Invariants bounds a padded numeric cell's width.
func Padded_Cell_Invariants(padded Padded_Cell, namespace invariant.Namespace) {
	invariant.Tree(padded, namespace).
		Range_Int(len(padded), PADDED_CELL_BYTES_MIN, PADDED_CELL_BYTES_MAX).
		Ensure()
}

// PADDED_NAME_BYTES_MIN is the label column at its own floor, the header's "Language".
const PADDED_NAME_BYTES_MIN = NAME_WIDTH_MIN

// PADDED_NAME_BYTES_MAX is the label column at its bound.
const PADDED_NAME_BYTES_MAX = NAME_WIDTH_MAX

// Padded_Name is the label cell laid out to the label column's width. It is distinct
// from Padded_Cell because the label column can never be narrower than its header,
// while a numeric column whose cells are all empty collapses to nothing.
type Padded_Name string

// Padded_Name_Invariants bounds a padded label's width.
func Padded_Name_Invariants(padded Padded_Name, namespace invariant.Namespace) {
	invariant.Tree(padded, namespace).
		Range_Int(len(padded), PADDED_NAME_BYTES_MIN, PADDED_NAME_BYTES_MAX).
		Ensure()
}

// CELL_TEXT_BYTES_MIN is the empty cell a row leaves in a column it does not fill.
const CELL_TEXT_BYTES_MIN = 0

// CELL_TEXT_BYTES_MAX is the widest numeric cell, the line bound with its separators.
// The label cell is wider but goes to its own padder, so it is not spanned here.
const CELL_TEXT_BYTES_MAX = COMMENTS_CELL_BYTES_MAX

// Cell_Text is a numeric table cell handed to the right-justifying padder.
type Cell_Text string

// Cell_Text_Invariants bounds a cell's width before padding.
func Cell_Text_Invariants(text Cell_Text, namespace invariant.Namespace) {
	invariant.Tree(text, namespace).
		Range_Int(len(text), CELL_TEXT_BYTES_MIN, CELL_TEXT_BYTES_MAX).
		Ensure()
}

// Right-justifies text in width by prefixing spaces.
func pad_left(text Cell_Text, width Column_Width) (padded Padded_Cell) {
	defer func() { Padded_Cell_Invariants(padded, "pad_left.padded") }()
	Cell_Text_Invariants(text, "pad_left.text")
	Column_Width_Invariants(width, "pad_left.width")
	return Padded_Cell(fmt.Sprintf("%*s", int(width), string(text)))
}

// Left-justifies a label in the label column's width by suffixing spaces.
func pad_right(text Row_Name, width Name_Width) (padded Padded_Name) {
	defer func() { Padded_Name_Invariants(padded, "pad_right.padded") }()
	Row_Name_Invariants(text, "pad_right.text")
	Name_Width_Invariants(width, "pad_right.width")
	return Padded_Name(fmt.Sprintf("%-*s", int(width), string(text)))
}

// Renders a non-negative count with a comma between each group of three digits, so
// large totals stay readable.
func with_thousands_separators(value Line_Count) (text Tally_Text) {
	defer func() { Tally_Text_Invariants(text, "with_thousands_separators.text") }()
	Line_Count_Invariants(value, "with_thousands_separators.value")
	digits := strconv.Itoa(int(value))
	if len(digits) <= 3 {
		return Tally_Text(digits)
	}
	// The lead group holds the digits that do not fill a whole group of three.
	lead_count := len(digits) % 3
	if lead_count == 0 {
		lead_count = 3
	}
	builder := strings.Builder{}
	builder.WriteString(digits[:lead_count])
	for index := lead_count; index < len(digits); index += 3 {
		builder.WriteByte(',')
		builder.WriteString(digits[index : index+3])
	}
	return Tally_Text(builder.String())
}
