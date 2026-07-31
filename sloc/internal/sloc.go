// Package sloc counts the code, comment, and blank lines of source files. A
// generic per-line scanner, configured by a Language, classifies each physical
// line; a walker counts a tree in parallel; a renderer prints an aligned table.
package sloc

import (
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
		Paths:          Paths(cli.Get_Option(command.Arguments, "path").Value.([]string)),
		No_Ignore:      cli.Get_Option(command.Flags, "no-ignore").Value.(bool),
		Include_Hidden: cli.Get_Option(command.Flags, "hidden").Value.(bool),
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
		json_err := Render_Json(input.Output, report)
		if json_err != nil {
			fmt.Fprintf(input.Error_Output, "sloc: %v\n", json_err)
			return EXIT_FAILURE
		}
		return EXIT_SUCCESS
	}
	Render(input.Output, Render_Input{
		Report:     report,
		Show_Files: cli.Get_Option(command.Flags, "files").Value.(bool),
	})
	return EXIT_SUCCESS
}

// Exit_Code is a process exit status. It is a distinct type so the status a run
// returns carries the declared range rather than the whole integer domain.
type Exit_Code uint8

// Exit_Code_Invariants bounds a status to the declared codes. Every value in the range
// is a declared code, so the range saturates and each of the three is demanded.
func Exit_Code_Invariants(code Exit_Code, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Uint8(uint8(code), uint8(EXIT_SUCCESS), uint8(EXIT_USAGE)).
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
		skipped_add(&report.Skipped, partial.Skipped)
	}
	return report, nil
}

// Counts a single path: a directory is walked, a file is classified directly.
func main_input_one(
	input *Main_Input, root Root, scope Main_Scope,
) (report Report, err error) {
	defer func() { Report_Invariants(report, "main_input_one.report") }()
	Main_Input_Invariants(*input, "main_input_one.input")
	Root_Invariants(root, "main_input_one.root")
	Main_Scope_Invariants(scope, "main_input_one.scope")
	directory, stat_err := input.Path_Is_Directory(File_Path(root))
	if stat_err != nil {
		return Report{}, stat_err
	}
	if directory {
		return main_input_directory(input, root, scope)
	}
	file, recognized, read_err := main_input_file(input, File_Path(root))
	if read_err != nil {
		return Report{}, read_err
	}
	if recognized {
		report.Files = append(report.Files, file)
	}
	return report, nil
}

// Walks one directory, prefixing each file path with the root so files from different
// roots stay distinguishable.
func main_input_directory(
	input *Main_Input, root Root, scope Main_Scope,
) (report Report, err error) {
	defer func() { Report_Invariants(report, "main_input_directory.report") }()
	Main_Input_Invariants(*input, "main_input_directory.input")
	Root_Invariants(root, "main_input_directory.root")
	Main_Scope_Invariants(scope, "main_input_directory.scope")
	directory_report, count_err := Count(Count_Input{
		File_System:    input.Open(root),
		Is_Ignored:     main_input_ignore(input, root, scope.No_Ignore),
		Include_Hidden: scope.Include_Hidden,
		Concurrency:    input.Concurrency,
	})
	if count_err != nil {
		return Report{}, count_err
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
	input *Main_Input, root Root, no_ignore bool,
) (is_ignored Ignore_Predicate) {
	Main_Input_Invariants(*input, "main_input_ignore.input")
	Root_Invariants(root, "main_input_ignore.root")
	invariant.Boolean_Invariants(no_ignore, "main_input_ignore.no_ignore")
	if no_ignore {
		return nil
	}
	return input.Ignore_For(root)
}

// Classifies one explicitly named file, reporting whether its extension is recognized.
func main_input_file(
	input *Main_Input, name File_Path,
) (file File_Count, recognized bool, err error) {
	defer func() {
		File_Count_Invariants(file, "main_input_file.file")
		invariant.Boolean_Invariants(recognized, "main_input_file.recognized")
	}()
	Main_Input_Invariants(*input, "main_input_file.input")
	File_Path_Invariants(name, "main_input_file.name")
	// The unrecognized and unreadable paths carry the name back rather than the zero
	// File_Count: a path is never empty, and the output invariant states that.
	unknown := File_Count{Path: name, Language: "", Counts: Counts{}, Is_Test: false}
	language, known := language_for_path(name)
	if !known {
		return unknown, false, nil
	}
	content, read_err := input.Read_File(name)
	if read_err != nil {
		return unknown, false, read_err
	}
	counts := Classify_File(Classify_File_Input{
		Source:   content,
		Language: language,
	})
	return File_Count{
		Path:     name,
		Language: language.Name,
		Counts:   counts,
		Is_Test:  false,
	}, true, nil
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
	invariant.Assertions(namespace).
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
	invariant.Assertions(namespace).
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
	invariant.Assertions(namespace).
		Range_Int(len(root), ROOT_BYTES_MIN, ROOT_BYTES_MAX).
		Ensure()
}

// Main_Scope is the resolved scope and override flags of one run.
type Main_Scope struct {
	// Paths is the list of files or directories to count.
	Paths Paths
	// No_Ignore counts gitignored files too.
	No_Ignore bool
	// Include_Hidden counts hidden dot-files and dot-directories.
	Include_Hidden bool
}

// Main_Scope_Invariants states the scope's roots and both override flags.
func Main_Scope_Invariants(scope Main_Scope, namespace invariant.Namespace) {
	Paths_Invariants(scope.Paths, "Main_Scope.Paths")
	invariant.Boolean_Invariants(scope.No_Ignore, "Main_Scope.No_Ignore")
	invariant.Boolean_Invariants(scope.Include_Hidden, "Main_Scope.Include_Hidden")
}

// Main_Input carries the command line and the host bindings Main needs. The library
// tier does no ambient I/O, so the filesystem, stat, read, and ignore operations are
// injected.
type Main_Input struct {
	// Arguments is the command line, including the program name.
	Arguments Arguments
	// Output is where the table is written.
	Output io.Writer
	// Error_Output is where usage and errors are written.
	Error_Output io.Writer
	// Open views a directory as a read-only file system rooted at it.
	Open func(root Root) (file_system fs.FS)
	// Path_Is_Directory reports whether a path names a directory.
	Path_Is_Directory func(name File_Path) (is_directory bool, err error)
	// Read_File reads a single file's bytes.
	Read_File func(name File_Path) (content Source, err error)
	// Ignore_For builds the ignore filter for a directory root, or returns nil.
	Ignore_For func(root Root) (is_ignored Ignore_Predicate)
	// Concurrency bounds the file-counting worker pool. It stays an unbounded integer
	// because it is host-supplied and count_classify clamps it at both ends.
	Concurrency int
}

// Main_Input_Invariants states the command line and the worker bound. The writers and
// the injected operations are function and interface values with no preset of their own.
func Main_Input_Invariants(input Main_Input, namespace invariant.Namespace) {
	Arguments_Invariants(input.Arguments, "Main_Input.Arguments")
	invariant.Int_Invariants(input.Concurrency, "Main_Input.Concurrency")
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
	Block_Comment_Nests bool
	// Verbatim_Strings are the multi-line string delimiters whose bodies are verbatim.
	Verbatim_Strings Verbatim_Delimiters
	// Quote_Strings are the single-line string and character delimiters.
	Quote_Strings Quote_Delimiters
	// Long_Bracket enables Lua's leveled long brackets: [[ … ]] and [=[ … ]=] as
	// strings, and the same after a line-comment token (--[[ … ]]) as comments.
	Long_Bracket bool
	// Test_Prefixes are base-name prefixes that mark a test file, like Python's test_.
	Test_Prefixes Name_Prefixes
	// Test_Infixes are case-sensitive base-name substrings that mark a test file, like
	// Go's _test. or JavaScript's .test. / .spec., delimited so latest.go does not match.
	Test_Infixes Name_Infixes
	// Heredoc enables <<DELIM heredocs (Shell, Ruby, Perl): the body lines up to a line
	// equal to DELIM are code, so a # inside the body is not read as a comment.
	Heredoc bool
}

// Language_Invariants states every field of a language configuration. The three
// booleans and the four token collections are all the scanner reads, so stating them
// here is stating the whole of what a language is.
func Language_Invariants(language Language, namespace invariant.Namespace) {
	Language_Name_Invariants(language.Name, "Language.Name")
	Comment_Tokens_Invariants(language.Line_Comment, "Language.Line_Comment")
	Block_Comment_Opener_Invariants(
		language.Block_Comment_Open, "Language.Block_Comment_Open")
	Block_Comment_Closer_Invariants(
		language.Block_Comment_Close, "Language.Block_Comment_Close")
	invariant.Boolean_Invariants(
		language.Block_Comment_Nests, "Language.Block_Comment_Nests")
	Verbatim_Delimiters_Invariants(language.Verbatim_Strings, "Language.Verbatim_Strings")
	Quote_Delimiters_Invariants(language.Quote_Strings, "Language.Quote_Strings")
	invariant.Boolean_Invariants(language.Long_Bracket, "Language.Long_Bracket")
	Name_Prefixes_Invariants(language.Test_Prefixes, "Language.Test_Prefixes")
	Name_Infixes_Invariants(language.Test_Infixes, "Language.Test_Infixes")
	invariant.Boolean_Invariants(language.Heredoc, "Language.Heredoc")
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
	invariant.Assertions(namespace).
		Range_Int(len(name), LANGUAGE_NAME_BYTES_MIN, LANGUAGE_NAME_BYTES_MAX).
		Ensure()
}

// COMMENT_TOKENS_COUNT_MIN is a language with no line comment at all, like HTML.
const COMMENT_TOKENS_COUNT_MIN = 0

// COMMENT_TOKENS_COUNT_MAX is the two line-comment tokens the richest seeds carry.
const COMMENT_TOKENS_COUNT_MAX = 2

// Comment_Tokens are the tokens that begin a comment running to end of line.
type Comment_Tokens []string

// Comment_Tokens_Invariants bounds how many line-comment tokens a language declares.
// Every count in the range occurs, so the range saturates.
func Comment_Tokens_Invariants(tokens Comment_Tokens, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Int(len(tokens), COMMENT_TOKENS_COUNT_MIN, COMMENT_TOKENS_COUNT_MAX).
		Ensure()
}

// BLOCK_COMMENT_OPENER_BYTES_MIN is the empty opener of a language with no block
// comment.
const BLOCK_COMMENT_OPENER_BYTES_MIN = 0

// BLOCK_COMMENT_OPENER_BYTES_MAX is the four-byte "<!--" of the markup languages.
const BLOCK_COMMENT_OPENER_BYTES_MAX = 4

// BLOCK_COMMENT_OPENER_BYTES_ABSENT is the one width no seeded opener has: the seeds
// run "", "{", the two-byte pairs, and "<!--", so three bytes never occurs. Excluding
// it keeps the grid satisfiable without weakening the bound to a guard.
const BLOCK_COMMENT_OPENER_BYTES_ABSENT = 3

// Block_Comment_Opener begins a block comment, or is empty when there is none.
type Block_Comment_Opener string

// Block_Comment_Opener_Invariants bounds an opener's byte length, carving the width no
// seeded language uses.
func Block_Comment_Opener_Invariants(
	opener Block_Comment_Opener, namespace invariant.Namespace,
) {
	invariant.Assertions(namespace).
		Range_Holed_Int(
			len(opener), BLOCK_COMMENT_OPENER_BYTES_MIN, BLOCK_COMMENT_OPENER_BYTES_MAX,
			BLOCK_COMMENT_OPENER_BYTES_ABSENT, BLOCK_COMMENT_OPENER_BYTES_ABSENT,
			BLOCK_COMMENT_OPENER_BYTES_ABSENT, BLOCK_COMMENT_OPENER_BYTES_ABSENT).
		Ensure()
}

// BLOCK_COMMENT_CLOSER_BYTES_MIN is the empty closer of a language with no block
// comment.
const BLOCK_COMMENT_CLOSER_BYTES_MIN = 0

// BLOCK_COMMENT_CLOSER_BYTES_MAX is the three-byte "-->" of the markup languages. It is
// one shorter than the opener max, which is why the two are distinct types: a shared
// bound would demand a width one of them can never reach.
const BLOCK_COMMENT_CLOSER_BYTES_MAX = 3

// Block_Comment_Closer ends a block comment, or is empty when there is none.
type Block_Comment_Closer string

// Block_Comment_Closer_Invariants bounds a closer's byte length.
func Block_Comment_Closer_Invariants(
	closer Block_Comment_Closer, namespace invariant.Namespace,
) {
	invariant.Assertions(namespace).
		Range_Int(
			len(closer),
			BLOCK_COMMENT_CLOSER_BYTES_MIN, BLOCK_COMMENT_CLOSER_BYTES_MAX).
		Ensure()
}

// NAME_PREFIXES_COUNT_MIN is a language whose test files are not marked by a prefix.
const NAME_PREFIXES_COUNT_MIN = 0

// NAME_PREFIXES_COUNT_MAX is the one prefix the seeds use, Python's "test_".
const NAME_PREFIXES_COUNT_MAX = 1

// Name_Prefixes are base-name prefixes that mark a test file.
type Name_Prefixes []string

// Name_Prefixes_Invariants bounds how many test prefixes a language declares.
func Name_Prefixes_Invariants(prefixes Name_Prefixes, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Int(len(prefixes), NAME_PREFIXES_COUNT_MIN, NAME_PREFIXES_COUNT_MAX).
		Ensure()
}

// NAME_INFIXES_COUNT_MIN is a language whose test files are not marked by an infix.
const NAME_INFIXES_COUNT_MIN = 0

// NAME_INFIXES_COUNT_MAX is the three infixes the richest seed carries.
const NAME_INFIXES_COUNT_MAX = 3

// Name_Infixes are case-sensitive base-name substrings that mark a test file.
type Name_Infixes []string

// Name_Infixes_Invariants bounds how many test infixes a language declares.
func Name_Infixes_Invariants(infixes Name_Infixes, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Int(len(infixes), NAME_INFIXES_COUNT_MIN, NAME_INFIXES_COUNT_MAX).
		Ensure()
}

// VERBATIM_DELIMITERS_COUNT_MIN is a language with no verbatim string at all.
const VERBATIM_DELIMITERS_COUNT_MIN = 0

// VERBATIM_DELIMITERS_COUNT_MAX is the two verbatim forms the richest seeds carry.
const VERBATIM_DELIMITERS_COUNT_MAX = 2

// Verbatim_Delimiters are the multi-line string delimiters whose bodies are verbatim.
type Verbatim_Delimiters []Verbatim_Delimiter

// Verbatim_Delimiters_Invariants bounds how many verbatim forms a language declares.
func Verbatim_Delimiters_Invariants(
	delimiters Verbatim_Delimiters, namespace invariant.Namespace,
) {
	invariant.Assertions(namespace).
		Range_Int(
			len(delimiters), VERBATIM_DELIMITERS_COUNT_MIN,
			VERBATIM_DELIMITERS_COUNT_MAX).
		Ensure()
}

// QUOTE_DELIMITERS_COUNT_MIN is a language with no quoted string at all.
const QUOTE_DELIMITERS_COUNT_MIN = 0

// QUOTE_DELIMITERS_COUNT_MAX is the string-and-character pair most seeds carry.
const QUOTE_DELIMITERS_COUNT_MAX = 2

// Quote_Delimiters are the single-line string and character delimiters.
type Quote_Delimiters []Quote_Delimiter

// Quote_Delimiters_Invariants bounds how many quoted forms a language declares.
func Quote_Delimiters_Invariants(
	delimiters Quote_Delimiters, namespace invariant.Namespace,
) {
	invariant.Assertions(namespace).
		Range_Int(
			len(delimiters), QUOTE_DELIMITERS_COUNT_MIN, QUOTE_DELIMITERS_COUNT_MAX).
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
	invariant.Assertions(namespace).
		Range_Int(len(delimiters), QUOTE_PAIR_COUNT, QUOTE_PAIR_COUNT).
		Ensure()
}

// QUOTE_SINGLE_COUNT is the one form a lone double-quoted string set carries.
const QUOTE_SINGLE_COUNT = 1

// Quote_Single is a shared one-delimiter set, fixed by construction like Quote_Pair.
type Quote_Single []Quote_Delimiter

// Quote_Single_Invariants pins a lone shared delimiter to the one form it holds.
func Quote_Single_Invariants(delimiters Quote_Single, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Int(len(delimiters), QUOTE_SINGLE_COUNT, QUOTE_SINGLE_COUNT).
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
	Hashable bool
}

// Verbatim_Delimiter_Invariants states a verbatim form's two delimiters and its shape.
func Verbatim_Delimiter_Invariants(
	delimiter Verbatim_Delimiter, namespace invariant.Namespace,
) {
	Verbatim_Opener_Invariants(delimiter.Open, "Verbatim_Delimiter.Open")
	Verbatim_Closer_Invariants(delimiter.Close, "Verbatim_Delimiter.Close")
	invariant.Boolean_Invariants(delimiter.Hashable, "Verbatim_Delimiter.Hashable")
}

// VERBATIM_OPENER_BYTES_MIN is the one-byte backtick and Rust's one-byte "r" lead.
const VERBATIM_OPENER_BYTES_MIN = 1

// VERBATIM_OPENER_BYTES_MAX is the three-byte triple quote.
const VERBATIM_OPENER_BYTES_MAX = 3

// Verbatim_Opener opens a verbatim string, or leads the hashes of a hashable one.
type Verbatim_Opener string

// Verbatim_Opener_Invariants bounds a verbatim opener's byte length. Every width in the
// range occurs — the backtick, Rust's "br", the triple quote — so the range saturates.
func Verbatim_Opener_Invariants(opener Verbatim_Opener, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Int(len(opener), VERBATIM_OPENER_BYTES_MIN, VERBATIM_OPENER_BYTES_MAX).
		Ensure()
}

// VERBATIM_CLOSER_BYTES_MIN is the empty closer a hashable form carries, whose
// terminator is computed from the hash count rather than declared.
const VERBATIM_CLOSER_BYTES_MIN = 0

// VERBATIM_CLOSER_BYTES_MAX is the three-byte triple quote.
const VERBATIM_CLOSER_BYTES_MAX = 3

// Verbatim_Closer terminates a fixed verbatim string, empty when the form is hashable.
type Verbatim_Closer string

// Verbatim_Closer_Invariants bounds a verbatim closer's byte length. Every width in the
// range occurs: the empty closer of a hashable form, the one-byte backtick, Python's
// two-byte empty pair, and the three-byte triple quote.
func Verbatim_Closer_Invariants(closer Verbatim_Closer, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Int(len(closer), VERBATIM_CLOSER_BYTES_MIN, VERBATIM_CLOSER_BYTES_MAX).
		Ensure()
}

// Quote_Delimiter describes a single-line string or character literal.
type Quote_Delimiter struct {
	// Open is the opening delimiter.
	Open Quote_Mark
	// Close is the closing delimiter.
	Close Quote_Mark
	// Escape is the byte that escapes the following character, or zero for none.
	Escape Escape_Byte
	// Character_Like marks a character or rune literal, whose apostrophe must be told
	// apart from a Rust lifetime tick that opens nothing.
	Character_Like bool
}

// Quote_Delimiter_Invariants states a quoted form's marks, its escape, and its shape.
func Quote_Delimiter_Invariants(
	delimiter Quote_Delimiter, namespace invariant.Namespace,
) {
	Quote_Mark_Invariants(delimiter.Open, "Quote_Delimiter.Open")
	Quote_Mark_Invariants(delimiter.Close, "Quote_Delimiter.Close")
	Escape_Byte_Invariants(delimiter.Escape, "Quote_Delimiter.Escape")
	invariant.Boolean_Invariants(
		delimiter.Character_Like, "Quote_Delimiter.Character_Like")
}

// QUOTE_MARK_BYTES_MIN is the width of every quote mark: a single byte.
const QUOTE_MARK_BYTES_MIN = 1

// QUOTE_MARK_BYTES_MAX is the same single byte. A quoted form's delimiter is exactly
// one byte in every seeded language, so the bound is a point, not an interval.
const QUOTE_MARK_BYTES_MAX = 1

// Quote_Mark opens or closes a single-line string or character literal.
type Quote_Mark string

// Quote_Mark_Invariants pins a quote mark to the single byte every seeded form uses.
func Quote_Mark_Invariants(mark Quote_Mark, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Int(len(mark), QUOTE_MARK_BYTES_MIN, QUOTE_MARK_BYTES_MAX).
		Ensure()
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
	invariant.Assertions(namespace).
		Enum_Uint8(
			uint8(escape), uint8(ESCAPE_BYTE_NONE), uint8(ESCAPE_BYTE_BACKSLASH)).
		Ensure()
}

// Language_Go returns the Go configuration: // line comments, non-nesting /* */
// block comments, backtick raw strings, and quoted literals with backslash escapes.
func Language_Go() (language Language) {
	defer func() { Language_Invariants(language, "Language_Go.language") }()
	return Language{
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
	}
}

// Language_Rust returns the Rust configuration: // line comments, nesting /* */
// block comments, raw strings with hash matching, and quoted literals.
func Language_Rust() (language Language) {
	defer func() { Language_Invariants(language, "Language_Rust.language") }()
	return Language{
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
	}
}

// Language_Python returns the Python configuration: # line comments, no block
// comments, triple-quoted docstrings that span lines, and quoted strings. A docstring
// is a string, so its lines count as code.
func Language_Python() (language Language) {
	defer func() { Language_Invariants(language, "Language_Python.language") }()
	return Language{
		Name:                "Python",
		Test_Prefixes:       []string{"test_"},
		Test_Infixes:        []string{"_test."},
		Line_Comment:        []string{"#"},
		Block_Comment_Open:  "",
		Block_Comment_Close: "",
		Block_Comment_Nests: false,
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
	}
}

// Language_Java_Script returns the JavaScript configuration: // and non-nesting
// /* */ comments, backtick template literals that span lines, and quoted strings.
func Language_Java_Script() (language Language) {
	defer func() { Language_Invariants(language, "Language_Java_Script.language") }()
	return Language{
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
	}
}

// Language_Type_Script returns the TypeScript configuration, which lexes like
// JavaScript for counting: // and /* */ comments, template literals, quoted strings.
func Language_Type_Script() (language Language) {
	defer func() { Language_Invariants(language, "Language_Type_Script.language") }()
	return Language{
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
	}
}

// Language_C returns the C configuration: // and /* */ comments, with quoted strings
// and character literals.
func Language_C() (language Language) {
	defer func() { Language_Invariants(language, "Language_C.language") }()
	return Language{
		Name:                "C",
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings: []Quote_Delimiter{
			{Open: "\"", Close: "\"", Escape: '\\'},
			{Open: "'", Close: "'", Escape: '\\', Character_Like: true},
		},
	}
}

// Language_Cpp returns the C++ configuration: // and /* */ comments, with quoted
// strings and character literals.
func Language_Cpp() (language Language) {
	defer func() { Language_Invariants(language, "Language_Cpp.language") }()
	return Language{
		Name:                "C++",
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings: []Quote_Delimiter{
			{Open: "\"", Close: "\"", Escape: '\\'},
			{Open: "'", Close: "'", Escape: '\\', Character_Like: true},
		},
	}
}

// Language_C_Sharp returns the C# configuration: // and /* */ comments, """ raw
// strings, and quoted strings with character literals.
func Language_C_Sharp() (language Language) {
	defer func() { Language_Invariants(language, "Language_C_Sharp.language") }()
	return Language{
		Name:                "C#",
		Test_Infixes:        []string{"Test.", "Tests."},
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Verbatim_Strings:    []Verbatim_Delimiter{{Open: "\"\"\"", Close: "\"\"\""}},
		Quote_Strings: []Quote_Delimiter{
			{Open: "\"", Close: "\"", Escape: '\\'},
			{Open: "'", Close: "'", Escape: '\\', Character_Like: true},
		},
	}
}

// Language_Java returns the Java configuration: // and /* */ comments, """ text
// blocks, and quoted strings with character literals.
func Language_Java() (language Language) {
	defer func() { Language_Invariants(language, "Language_Java.language") }()
	return Language{
		Name:                "Java",
		Test_Infixes:        []string{"Test.", "Tests."},
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Verbatim_Strings:    []Verbatim_Delimiter{{Open: "\"\"\"", Close: "\"\"\""}},
		Quote_Strings: []Quote_Delimiter{
			{Open: "\"", Close: "\"", Escape: '\\'},
			{Open: "'", Close: "'", Escape: '\\', Character_Like: true},
		},
	}
}

// Language_Swift returns the Swift configuration: // and nesting /* */ comments, """
// multi-line strings, and double-quoted strings.
func Language_Swift() (language Language) {
	defer func() { Language_Invariants(language, "Language_Swift.language") }()
	return Language{
		Name:                "Swift",
		Test_Infixes:        []string{"Tests.", "Test."},
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Block_Comment_Nests: true,
		Verbatim_Strings:    []Verbatim_Delimiter{{Open: "\"\"\"", Close: "\"\"\""}},
		Quote_Strings:       []Quote_Delimiter{{Open: "\"", Close: "\"", Escape: '\\'}},
	}
}

// Language_Kotlin returns the Kotlin configuration: // and nesting /* */ comments,
// """ raw strings, and quoted strings with character literals.
func Language_Kotlin() (language Language) {
	defer func() { Language_Invariants(language, "Language_Kotlin.language") }()
	return Language{
		Name:                "Kotlin",
		Test_Infixes:        []string{"Test.", "Tests."},
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Block_Comment_Nests: true,
		Verbatim_Strings:    []Verbatim_Delimiter{{Open: "\"\"\"", Close: "\"\"\""}},
		Quote_Strings: []Quote_Delimiter{
			{Open: "\"", Close: "\"", Escape: '\\'},
			{Open: "'", Close: "'", Escape: '\\', Character_Like: true},
		},
	}
}

// Language_Scala returns the Scala configuration: // and nesting /* */ comments, """
// multi-line strings, and quoted strings with character literals.
func Language_Scala() (language Language) {
	defer func() { Language_Invariants(language, "Language_Scala.language") }()
	return Language{
		Name:                "Scala",
		Test_Infixes:        []string{"Test.", "Tests.", "Spec."},
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Block_Comment_Nests: true,
		Verbatim_Strings:    []Verbatim_Delimiter{{Open: "\"\"\"", Close: "\"\"\""}},
		Quote_Strings: []Quote_Delimiter{
			{Open: "\"", Close: "\"", Escape: '\\'},
			{Open: "'", Close: "'", Escape: '\\', Character_Like: true},
		},
	}
}

// Language_Shell returns the Shell configuration: # line comments, double-quoted
// strings with escapes, and literal single-quoted strings.
func Language_Shell() (language Language) {
	defer func() { Language_Invariants(language, "Language_Shell.language") }()
	return Language{
		Name:         "Shell",
		Line_Comment: []string{"#"},
		Heredoc:      true,
		Quote_Strings: []Quote_Delimiter{
			{Open: "\"", Close: "\"", Escape: '\\'},
			{Open: "'", Close: "'"},
		},
	}
}

// Language_Ruby returns the Ruby configuration: # line comments and quoted strings.
func Language_Ruby() (language Language) {
	defer func() { Language_Invariants(language, "Language_Ruby.language") }()
	return Language{
		Name:         "Ruby",
		Test_Infixes: []string{"_spec.", "_test."},
		Line_Comment: []string{"#"},
		Heredoc:      true,
		Quote_Strings: []Quote_Delimiter{
			{Open: "\"", Close: "\"", Escape: '\\'},
			{Open: "'", Close: "'", Escape: '\\'},
		},
	}
}

// Language_Yaml returns the YAML configuration: # line comments and quoted strings.
func Language_Yaml() (language Language) {
	defer func() { Language_Invariants(language, "Language_Yaml.language") }()
	return Language{
		Name:         "YAML",
		Line_Comment: []string{"#"},
		Quote_Strings: []Quote_Delimiter{
			{Open: "\"", Close: "\"", Escape: '\\'},
			{Open: "'", Close: "'", Escape: '\\'},
		},
	}
}

// Language_Toml returns the TOML configuration: # line comments, """ and ”' multi-
// line strings, and quoted strings.
func Language_Toml() (language Language) {
	defer func() { Language_Invariants(language, "Language_Toml.language") }()
	return Language{
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
	}
}

// Language_Sql returns the SQL configuration: -- and /* */ comments, with quoted
// strings and quoted identifiers.
func Language_Sql() (language Language) {
	defer func() { Language_Invariants(language, "Language_Sql.language") }()
	return Language{
		Name:                "SQL",
		Line_Comment:        []string{"--"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings: []Quote_Delimiter{
			{Open: "\"", Close: "\"", Escape: '\\'},
			{Open: "'", Close: "'", Escape: '\\'},
		},
	}
}

// Language_Makefile returns the Makefile configuration: # line comments.
func Language_Makefile() (language Language) {
	defer func() { Language_Invariants(language, "Language_Makefile.language") }()
	return Language{
		Name:         "Makefile",
		Line_Comment: []string{"#"},
	}
}

// Language_Dockerfile returns the Dockerfile configuration: # line comments.
func Language_Dockerfile() (language Language) {
	defer func() { Language_Invariants(language, "Language_Dockerfile.language") }()
	return Language{
		Name:         "Dockerfile",
		Line_Comment: []string{"#"},
	}
}

// Language_Html returns the HTML configuration: <!-- --> comments and no line comment.
func Language_Html() (language Language) {
	defer func() { Language_Invariants(language, "Language_Html.language") }()
	return Language{
		Name:                "HTML",
		Block_Comment_Open:  "<!--",
		Block_Comment_Close: "-->",
	}
}

// Language_Xml returns the XML configuration: <!-- --> comments and no line comment.
func Language_Xml() (language Language) {
	defer func() { Language_Invariants(language, "Language_Xml.language") }()
	return Language{
		Name:                "XML",
		Block_Comment_Open:  "<!--",
		Block_Comment_Close: "-->",
	}
}

// Language_Css returns the CSS configuration: /* */ comments and quoted strings.
func Language_Css() (language Language) {
	defer func() { Language_Invariants(language, "Language_Css.language") }()
	return Language{
		Name:                "CSS",
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings: []Quote_Delimiter{
			{Open: "\"", Close: "\"", Escape: '\\'},
			{Open: "'", Close: "'", Escape: '\\'},
		},
	}
}

// Language_Scss returns the SCSS configuration: // and /* */ comments and quoted
// strings.
func Language_Scss() (language Language) {
	defer func() { Language_Invariants(language, "Language_Scss.language") }()
	return Language{
		Name:                "SCSS",
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings: []Quote_Delimiter{
			{Open: "\"", Close: "\"", Escape: '\\'},
			{Open: "'", Close: "'", Escape: '\\'},
		},
	}
}

// Language_Less returns the LESS configuration: // and /* */ comments and quoted
// strings.
func Language_Less() (language Language) {
	defer func() { Language_Invariants(language, "Language_Less.language") }()
	return Language{
		Name:                "LESS",
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings: []Quote_Delimiter{
			{Open: "\"", Close: "\"", Escape: '\\'},
			{Open: "'", Close: "'", Escape: '\\'},
		},
	}
}

// Language_Lua returns the Lua configuration: -- line comments, --[[ ]] block
// comments and [[ ]] long strings (both leveled with = signs), and quoted strings.
func Language_Lua() (language Language) {
	defer func() { Language_Invariants(language, "Language_Lua.language") }()
	return Language{
		Name:         "Lua",
		Line_Comment: []string{"--"},
		Long_Bracket: true,
		Quote_Strings: []Quote_Delimiter{
			{Open: "\"", Close: "\"", Escape: '\\'},
			{Open: "'", Close: "'", Escape: '\\'},
		},
	}
}

// Language_Odin returns the Odin configuration: // line comments, nesting /* */ block
// comments, backtick raw strings, and quoted strings with rune literals.
func Language_Odin() (language Language) {
	defer func() { Language_Invariants(language, "Language_Odin.language") }()
	return Language{
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
	}
}

// Language_Zig returns the Zig configuration: // line comments and no block comments,
// with quoted strings and character literals. A \\ multi-line string line is code
// because its leading backslashes are code, so it needs no special handling.
func Language_Zig() (language Language) {
	defer func() { Language_Invariants(language, "Language_Zig.language") }()
	return Language{
		Name:         "Zig",
		Line_Comment: []string{"//"},
		Quote_Strings: []Quote_Delimiter{
			{Open: "\"", Close: "\"", Escape: '\\'},
			{Open: "'", Close: "'", Escape: '\\', Character_Like: true},
		},
	}
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

// Language_Objective_C returns the Objective-C configuration: // and /* */ comments.
func Language_Objective_C() (language Language) {
	defer func() { Language_Invariants(language, "Language_Objective_C.language") }()
	return Language{
		Name:                "Objective-C",
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings:       Quote_Delimiters(c_family_quotes()),
	}
}

// Language_Dart returns the Dart configuration: // and nesting /* */ comments, with
// ”' and """ multi-line strings.
func Language_Dart() (language Language) {
	defer func() { Language_Invariants(language, "Language_Dart.language") }()
	return Language{
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
	}
}

// Language_Php returns the PHP configuration: //, #, and /* */ comments.
func Language_Php() (language Language) {
	defer func() { Language_Invariants(language, "Language_Php.language") }()
	return Language{
		Name:                "PHP",
		Test_Infixes:        []string{"Test."},
		Line_Comment:        []string{"//", "#"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings:       Quote_Delimiters(plain_quotes()),
	}
}

// Language_Solidity returns the Solidity configuration: // and /* */ comments.
func Language_Solidity() (language Language) {
	defer func() { Language_Invariants(language, "Language_Solidity.language") }()
	return Language{
		Name:                "Solidity",
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings:       Quote_Delimiters(plain_quotes()),
	}
}

// Language_Groovy returns the Groovy configuration: // and /* */ comments, with ”' and
// """ multi-line strings.
func Language_Groovy() (language Language) {
	defer func() { Language_Invariants(language, "Language_Groovy.language") }()
	return Language{
		Name:                "Groovy",
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Verbatim_Strings: []Verbatim_Delimiter{
			{Open: "\"\"\"", Close: "\"\"\""},
			{Open: "'''", Close: "'''"},
		},
		Quote_Strings: Quote_Delimiters(plain_quotes()),
	}
}

// Language_Verilog returns the Verilog configuration: // and /* */ comments.
func Language_Verilog() (language Language) {
	defer func() { Language_Invariants(language, "Language_Verilog.language") }()
	return Language{
		Name:                "Verilog",
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings:       Quote_Delimiters(double_quote()),
	}
}

// Language_Glsl returns the GLSL configuration: // and /* */ comments.
func Language_Glsl() (language Language) {
	defer func() { Language_Invariants(language, "Language_Glsl.language") }()
	return Language{
		Name:                "GLSL",
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings:       Quote_Delimiters(double_quote()),
	}
}

// Language_Hlsl returns the HLSL configuration: // and /* */ comments.
func Language_Hlsl() (language Language) {
	defer func() { Language_Invariants(language, "Language_Hlsl.language") }()
	return Language{
		Name:                "HLSL",
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings:       Quote_Delimiters(double_quote()),
	}
}

// Language_Arduino returns the Arduino configuration: // and /* */ comments.
func Language_Arduino() (language Language) {
	defer func() { Language_Invariants(language, "Language_Arduino.language") }()
	return Language{
		Name:                "Arduino",
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings:       Quote_Delimiters(c_family_quotes()),
	}
}

// Language_Protobuf returns the Protocol Buffers configuration: // and /* */ comments.
func Language_Protobuf() (language Language) {
	defer func() { Language_Invariants(language, "Language_Protobuf.language") }()
	return Language{
		Name:                "Protobuf",
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings:       Quote_Delimiters(plain_quotes()),
	}
}

// Language_Thrift returns the Thrift configuration: //, #, and /* */ comments.
func Language_Thrift() (language Language) {
	defer func() { Language_Invariants(language, "Language_Thrift.language") }()
	return Language{
		Name:                "Thrift",
		Line_Comment:        []string{"//", "#"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings:       Quote_Delimiters(plain_quotes()),
	}
}

// Language_Jsonc returns the JSONC/JSON5 configuration: // and /* */ comments.
func Language_Jsonc() (language Language) {
	defer func() { Language_Invariants(language, "Language_Jsonc.language") }()
	return Language{
		Name:                "JSONC",
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings:       Quote_Delimiters(double_quote()),
	}
}

// Language_Hcl returns the HCL/Terraform configuration: #, //, and /* */ comments.
func Language_Hcl() (language Language) {
	defer func() { Language_Invariants(language, "Language_Hcl.language") }()
	return Language{
		Name:                "HCL",
		Line_Comment:        []string{"#", "//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings:       Quote_Delimiters(double_quote()),
	}
}

// Language_Nix returns the Nix configuration: # and /* */ comments, with ” ” strings.
func Language_Nix() (language Language) {
	defer func() { Language_Invariants(language, "Language_Nix.language") }()
	return Language{
		Name:                "Nix",
		Line_Comment:        []string{"#"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Verbatim_Strings:    []Verbatim_Delimiter{{Open: "''", Close: "''"}},
		Quote_Strings:       Quote_Delimiters(double_quote()),
	}
}

// Language_Markdown returns the Markdown configuration: <!-- --> comments only.
func Language_Markdown() (language Language) {
	defer func() { Language_Invariants(language, "Language_Markdown.language") }()
	return Language{
		Name:                "Markdown",
		Block_Comment_Open:  "<!--",
		Block_Comment_Close: "-->",
	}
}

// Language_Vue returns the Vue configuration: <!-- --> comments only.
func Language_Vue() (language Language) {
	defer func() { Language_Invariants(language, "Language_Vue.language") }()
	return Language{
		Name:                "Vue",
		Block_Comment_Open:  "<!--",
		Block_Comment_Close: "-->",
	}
}

// Language_Svelte returns the Svelte configuration: <!-- --> comments only.
func Language_Svelte() (language Language) {
	defer func() { Language_Invariants(language, "Language_Svelte.language") }()
	return Language{
		Name:                "Svelte",
		Block_Comment_Open:  "<!--",
		Block_Comment_Close: "-->",
	}
}

// Language_Astro returns the Astro configuration: <!-- --> comments only.
func Language_Astro() (language Language) {
	defer func() { Language_Invariants(language, "Language_Astro.language") }()
	return Language{
		Name:                "Astro",
		Block_Comment_Open:  "<!--",
		Block_Comment_Close: "-->",
	}
}

// Language_Xaml returns the XAML configuration: <!-- --> comments only.
func Language_Xaml() (language Language) {
	defer func() { Language_Invariants(language, "Language_Xaml.language") }()
	return Language{
		Name:                "XAML",
		Block_Comment_Open:  "<!--",
		Block_Comment_Close: "-->",
	}
}

// Language_Xslt returns the XSLT configuration: <!-- --> comments only.
func Language_Xslt() (language Language) {
	defer func() { Language_Invariants(language, "Language_Xslt.language") }()
	return Language{
		Name:                "XSLT",
		Block_Comment_Open:  "<!--",
		Block_Comment_Close: "-->",
	}
}

// Language_Haskell returns the Haskell configuration: -- line comments and nesting
// {- -} block comments.
func Language_Haskell() (language Language) {
	defer func() { Language_Invariants(language, "Language_Haskell.language") }()
	return Language{
		Name:                "Haskell",
		Line_Comment:        []string{"--"},
		Block_Comment_Open:  "{-",
		Block_Comment_Close: "-}",
		Block_Comment_Nests: true,
		Quote_Strings:       Quote_Delimiters(double_quote()),
	}
}

// Language_Ocaml returns the OCaml configuration: nesting (* *) block comments and no
// line comment.
func Language_Ocaml() (language Language) {
	defer func() { Language_Invariants(language, "Language_Ocaml.language") }()
	return Language{
		Name:                "OCaml",
		Block_Comment_Open:  "(*",
		Block_Comment_Close: "*)",
		Block_Comment_Nests: true,
		Quote_Strings:       Quote_Delimiters(double_quote()),
	}
}

// Language_F_Sharp returns the F# configuration: // line comments, nesting (* *) block
// comments, and """ strings.
func Language_F_Sharp() (language Language) {
	defer func() { Language_Invariants(language, "Language_F_Sharp.language") }()
	return Language{
		Name:                "F#",
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "(*",
		Block_Comment_Close: "*)",
		Block_Comment_Nests: true,
		Verbatim_Strings:    []Verbatim_Delimiter{{Open: "\"\"\"", Close: "\"\"\""}},
		Quote_Strings:       Quote_Delimiters(double_quote()),
	}
}

// Language_Julia returns the Julia configuration: # line comments, nesting #= =# block
// comments, and """ strings.
func Language_Julia() (language Language) {
	defer func() { Language_Invariants(language, "Language_Julia.language") }()
	return Language{
		Name:                "Julia",
		Line_Comment:        []string{"#"},
		Block_Comment_Open:  "#=",
		Block_Comment_Close: "=#",
		Block_Comment_Nests: true,
		Verbatim_Strings:    []Verbatim_Delimiter{{Open: "\"\"\"", Close: "\"\"\""}},
		Quote_Strings:       Quote_Delimiters(double_quote()),
	}
}

// Language_Nim returns the Nim configuration: # line comments, nesting #[ ]# block
// comments, and """ strings.
func Language_Nim() (language Language) {
	defer func() { Language_Invariants(language, "Language_Nim.language") }()
	return Language{
		Name:                "Nim",
		Line_Comment:        []string{"#"},
		Block_Comment_Open:  "#[",
		Block_Comment_Close: "]#",
		Block_Comment_Nests: true,
		Verbatim_Strings:    []Verbatim_Delimiter{{Open: "\"\"\"", Close: "\"\"\""}},
		Quote_Strings:       Quote_Delimiters(double_quote()),
	}
}

// Language_Common_Lisp returns the Common Lisp configuration: ; line comments and
// nesting #| |# block comments.
func Language_Common_Lisp() (language Language) {
	defer func() { Language_Invariants(language, "Language_Common_Lisp.language") }()
	return Language{
		Name:                "Common Lisp",
		Line_Comment:        []string{";"},
		Block_Comment_Open:  "#|",
		Block_Comment_Close: "|#",
		Block_Comment_Nests: true,
		Quote_Strings:       Quote_Delimiters(double_quote()),
	}
}

// Language_Scheme returns the Scheme configuration: ; line comments and nesting #| |#
// block comments.
func Language_Scheme() (language Language) {
	defer func() { Language_Invariants(language, "Language_Scheme.language") }()
	return Language{
		Name:                "Scheme",
		Line_Comment:        []string{";"},
		Block_Comment_Open:  "#|",
		Block_Comment_Close: "|#",
		Block_Comment_Nests: true,
		Quote_Strings:       Quote_Delimiters(double_quote()),
	}
}

// Language_Racket returns the Racket configuration: ; line comments and nesting #| |#
// block comments.
func Language_Racket() (language Language) {
	defer func() { Language_Invariants(language, "Language_Racket.language") }()
	return Language{
		Name:                "Racket",
		Line_Comment:        []string{";"},
		Block_Comment_Open:  "#|",
		Block_Comment_Close: "|#",
		Block_Comment_Nests: true,
		Quote_Strings:       Quote_Delimiters(double_quote()),
	}
}

// Language_Clojure returns the Clojure configuration: ; line comments only.
func Language_Clojure() (language Language) {
	defer func() { Language_Invariants(language, "Language_Clojure.language") }()
	return Language{
		Name:          "Clojure",
		Line_Comment:  []string{";"},
		Quote_Strings: Quote_Delimiters(double_quote()),
	}
}

// Language_Emacs_Lisp returns the Emacs Lisp configuration: ; line comments only.
func Language_Emacs_Lisp() (language Language) {
	defer func() { Language_Invariants(language, "Language_Emacs_Lisp.language") }()
	return Language{
		Name:          "Emacs Lisp",
		Line_Comment:  []string{";"},
		Quote_Strings: Quote_Delimiters(double_quote()),
	}
}

// Language_Erlang returns the Erlang configuration: % line comments only.
func Language_Erlang() (language Language) {
	defer func() { Language_Invariants(language, "Language_Erlang.language") }()
	return Language{
		Name:          "Erlang",
		Line_Comment:  []string{"%"},
		Quote_Strings: Quote_Delimiters(double_quote()),
	}
}

// Language_Fortran returns the Fortran configuration: ! line comments only.
func Language_Fortran() (language Language) {
	defer func() { Language_Invariants(language, "Language_Fortran.language") }()
	return Language{
		Name:          "Fortran",
		Line_Comment:  []string{"!"},
		Quote_Strings: Quote_Delimiters(plain_quotes()),
	}
}

// Language_Ada returns the Ada configuration: -- line comments, with strings and the
// apostrophe attribute/character distinction.
func Language_Ada() (language Language) {
	defer func() { Language_Invariants(language, "Language_Ada.language") }()
	return Language{
		Name:          "Ada",
		Line_Comment:  []string{"--"},
		Quote_Strings: Quote_Delimiters(c_family_quotes()),
	}
}

// Language_D returns the D configuration: // and /* */ comments, backtick raw strings,
// and strings with character literals.
func Language_D() (language Language) {
	defer func() { Language_Invariants(language, "Language_D.language") }()
	return Language{
		Name:                "D",
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Verbatim_Strings:    []Verbatim_Delimiter{{Open: "`", Close: "`"}},
		Quote_Strings:       Quote_Delimiters(c_family_quotes()),
	}
}

// Language_Pascal returns the Pascal configuration: // line comments, { } block
// comments, and single-quoted strings.
func Language_Pascal() (language Language) {
	defer func() { Language_Invariants(language, "Language_Pascal.language") }()
	return Language{
		Name:                "Pascal",
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "{",
		Block_Comment_Close: "}",
		Quote_Strings:       []Quote_Delimiter{{Open: "'", Close: "'"}},
	}
}

// Language_R returns the R configuration: # line comments only.
func Language_R() (language Language) {
	defer func() { Language_Invariants(language, "Language_R.language") }()
	return Language{
		Name:          "R",
		Line_Comment:  []string{"#"},
		Quote_Strings: Quote_Delimiters(plain_quotes()),
	}
}

// Language_Elixir returns the Elixir configuration: # line comments and """ / ”'
// heredoc strings.
func Language_Elixir() (language Language) {
	defer func() { Language_Invariants(language, "Language_Elixir.language") }()
	return Language{
		Name:         "Elixir",
		Test_Infixes: []string{"_test."},
		Line_Comment: []string{"#"},
		Verbatim_Strings: []Verbatim_Delimiter{
			{Open: "\"\"\"", Close: "\"\"\""},
			{Open: "'''", Close: "'''"},
		},
		Quote_Strings: Quote_Delimiters(plain_quotes()),
	}
}

// Language_Crystal returns the Crystal configuration: # line comments only.
func Language_Crystal() (language Language) {
	defer func() { Language_Invariants(language, "Language_Crystal.language") }()
	return Language{
		Name:          "Crystal",
		Line_Comment:  []string{"#"},
		Quote_Strings: Quote_Delimiters(double_quote()),
	}
}

// Language_Power_Shell returns the PowerShell configuration: # line comments and <# #>
// block comments.
func Language_Power_Shell() (language Language) {
	defer func() { Language_Invariants(language, "Language_Power_Shell.language") }()
	return Language{
		Name:                "PowerShell",
		Line_Comment:        []string{"#"},
		Block_Comment_Open:  "<#",
		Block_Comment_Close: "#>",
		Quote_Strings:       Quote_Delimiters(plain_quotes()),
	}
}

// Language_Fish returns the Fish shell configuration: # line comments.
func Language_Fish() (language Language) {
	defer func() { Language_Invariants(language, "Language_Fish.language") }()
	return Language{
		Name:          "Fish",
		Line_Comment:  []string{"#"},
		Quote_Strings: Quote_Delimiters(plain_quotes()),
	}
}

// Language_Nushell returns the Nushell configuration: # line comments.
func Language_Nushell() (language Language) {
	defer func() { Language_Invariants(language, "Language_Nushell.language") }()
	return Language{
		Name:          "Nushell",
		Line_Comment:  []string{"#"},
		Quote_Strings: Quote_Delimiters(plain_quotes()),
	}
}

// Language_Cmake returns the CMake configuration: # line comments and #[[ ]] bracket
// comments, reusing the leveled long-bracket machinery.
func Language_Cmake() (language Language) {
	defer func() { Language_Invariants(language, "Language_Cmake.language") }()
	return Language{
		Name:          "CMake",
		Line_Comment:  []string{"#"},
		Long_Bracket:  true,
		Quote_Strings: Quote_Delimiters(double_quote()),
	}
}

// Language_Tcl returns the Tcl configuration: # line comments only.
func Language_Tcl() (language Language) {
	defer func() { Language_Invariants(language, "Language_Tcl.language") }()
	return Language{
		Name:          "Tcl",
		Line_Comment:  []string{"#"},
		Quote_Strings: Quote_Delimiters(double_quote()),
	}
}

// Language_Perl returns the Perl configuration: # line comments only.
func Language_Perl() (language Language) {
	defer func() { Language_Invariants(language, "Language_Perl.language") }()
	return Language{
		Name:          "Perl",
		Line_Comment:  []string{"#"},
		Heredoc:       true,
		Quote_Strings: Quote_Delimiters(plain_quotes()),
	}
}

// Language_Tex returns the TeX/LaTeX configuration: % line comments only.
func Language_Tex() (language Language) {
	defer func() { Language_Invariants(language, "Language_Tex.language") }()
	return Language{
		Name:         "TeX",
		Line_Comment: []string{"%"},
	}
}

// Language_Visual_Basic returns the Visual Basic configuration: ' line comments only.
func Language_Visual_Basic() (language Language) {
	defer func() { Language_Invariants(language, "Language_Visual_Basic.language") }()
	return Language{
		Name:          "Visual Basic",
		Line_Comment:  []string{"'"},
		Quote_Strings: Quote_Delimiters(double_quote()),
	}
}

// Language_For_Extension returns the seeded language for a file extension, with the
// leading dot, and whether one matched.
func Language_For_Extension(extension Extension) (language Language, recognized bool) {
	defer func() {
		Language_Invariants(language, "Language_For_Extension.language")
		invariant.Boolean_Invariants(recognized, "Language_For_Extension.recognized")
	}()
	Extension_Invariants(extension, "Language_For_Extension.extension")
	switch extension {
	case ".go":
		return Language_Go(), true
	case ".rs":
		return Language_Rust(), true
	case ".py":
		return Language_Python(), true
	case ".js", ".jsx", ".mjs", ".cjs":
		return Language_Java_Script(), true
	case ".ts", ".tsx":
		return Language_Type_Script(), true
	case ".lua":
		return Language_Lua(), true
	case ".odin":
		return Language_Odin(), true
	case ".zig":
		return Language_Zig(), true
	case ".c", ".h":
		return Language_C(), true
	case ".cpp", ".cc", ".cxx", ".hpp", ".hh", ".hxx":
		return Language_Cpp(), true
	case ".cs":
		return Language_C_Sharp(), true
	case ".java":
		return Language_Java(), true
	case ".swift":
		return Language_Swift(), true
	case ".kt", ".kts":
		return Language_Kotlin(), true
	case ".scala", ".sc":
		return Language_Scala(), true
	case ".sh", ".bash", ".zsh":
		return Language_Shell(), true
	case ".rb":
		return Language_Ruby(), true
	case ".yaml", ".yml":
		return Language_Yaml(), true
	case ".toml":
		return Language_Toml(), true
	case ".sql":
		return Language_Sql(), true
	case ".mk":
		return Language_Makefile(), true
	case ".dockerfile":
		return Language_Dockerfile(), true
	case ".html", ".htm":
		return Language_Html(), true
	case ".xml", ".svg":
		return Language_Xml(), true
	case ".css":
		return Language_Css(), true
	case ".scss":
		return Language_Scss(), true
	case ".less":
		return Language_Less(), true
	}
	return extension_match_more(extension)
}

// Continues Language_For_Extension's lookup for the C-style and markup additions.
func extension_match_more(extension Extension) (language Language, recognized bool) {
	defer func() {
		Language_Invariants(language, "extension_match_more.language")
		invariant.Boolean_Invariants(recognized, "extension_match_more.recognized")
	}()
	Extension_Invariants(extension, "extension_match_more.extension")
	switch extension {
	case ".m", ".mm":
		return Language_Objective_C(), true
	case ".dart":
		return Language_Dart(), true
	case ".php", ".phtml":
		return Language_Php(), true
	case ".sol":
		return Language_Solidity(), true
	case ".groovy", ".gradle":
		return Language_Groovy(), true
	case ".v", ".sv", ".svh":
		return Language_Verilog(), true
	case ".glsl", ".vert", ".frag", ".comp", ".geom":
		return Language_Glsl(), true
	case ".hlsl":
		return Language_Hlsl(), true
	case ".ino":
		return Language_Arduino(), true
	case ".proto":
		return Language_Protobuf(), true
	case ".thrift":
		return Language_Thrift(), true
	case ".jsonc", ".json5":
		return Language_Jsonc(), true
	case ".tf", ".hcl", ".tfvars":
		return Language_Hcl(), true
	case ".nix":
		return Language_Nix(), true
	case ".md", ".markdown":
		return Language_Markdown(), true
	case ".vue":
		return Language_Vue(), true
	case ".svelte":
		return Language_Svelte(), true
	case ".astro":
		return Language_Astro(), true
	case ".xaml":
		return Language_Xaml(), true
	case ".xsl", ".xslt":
		return Language_Xslt(), true
	}
	return extension_match_rest(extension)
}

// Continues Language_For_Extension's lookup for the remaining languages.
func extension_match_rest(extension Extension) (language Language, recognized bool) {
	defer func() {
		Language_Invariants(language, "extension_match_rest.language")
		invariant.Boolean_Invariants(recognized, "extension_match_rest.recognized")
	}()
	Extension_Invariants(extension, "extension_match_rest.extension")
	switch extension {
	case ".hs", ".lhs":
		return Language_Haskell(), true
	case ".ml", ".mli":
		return Language_Ocaml(), true
	case ".fs", ".fsx", ".fsi":
		return Language_F_Sharp(), true
	case ".jl":
		return Language_Julia(), true
	case ".nim", ".nims":
		return Language_Nim(), true
	case ".lisp", ".lsp", ".cl":
		return Language_Common_Lisp(), true
	case ".scm", ".ss":
		return Language_Scheme(), true
	case ".rkt":
		return Language_Racket(), true
	case ".clj", ".cljs", ".cljc", ".edn":
		return Language_Clojure(), true
	case ".el":
		return Language_Emacs_Lisp(), true
	case ".erl", ".hrl":
		return Language_Erlang(), true
	case ".f90", ".f95", ".f03", ".f08", ".f", ".for":
		return Language_Fortran(), true
	case ".adb", ".ads", ".ada":
		return Language_Ada(), true
	case ".d":
		return Language_D(), true
	case ".pas", ".pp", ".dpr":
		return Language_Pascal(), true
	case ".r", ".R":
		return Language_R(), true
	case ".ex", ".exs":
		return Language_Elixir(), true
	case ".cr":
		return Language_Crystal(), true
	case ".ps1", ".psm1", ".psd1":
		return Language_Power_Shell(), true
	case ".fish":
		return Language_Fish(), true
	case ".nu":
		return Language_Nushell(), true
	case ".cmake":
		return Language_Cmake(), true
	case ".tcl":
		return Language_Tcl(), true
	case ".pl", ".pm", ".t", ".pod":
		return Language_Perl(), true
	case ".tex", ".sty", ".cls", ".ltx":
		return Language_Tex(), true
	case ".vb":
		return Language_Visual_Basic(), true
	}
	return Language{}, false
}

// Language_For_Filename returns the language for an extensionless file recognized by
// its name, and whether one matched.
func Language_For_Filename(name File_Name) (language Language, recognized bool) {
	defer func() {
		Language_Invariants(language, "Language_For_Filename.language")
		invariant.Boolean_Invariants(recognized, "Language_For_Filename.recognized")
	}()
	File_Name_Invariants(name, "Language_For_Filename.name")
	switch name {
	case "Makefile", "makefile", "GNUmakefile":
		return Language_Makefile(), true
	case "Dockerfile":
		return Language_Dockerfile(), true
	case "CMakeLists.txt":
		return Language_Cmake(), true
	}
	return Language{}, false
}

// Resolves the language for a path by its extension, or for an extensionless file by
// its name, and whether one matched.
func language_for_path(file_path File_Path) (language Language, recognized bool) {
	defer func() {
		Language_Invariants(language, "language_for_path.language")
		invariant.Boolean_Invariants(recognized, "language_for_path.recognized")
	}()
	File_Path_Invariants(file_path, "language_for_path.file_path")
	language, recognized = Language_For_Extension(Extension(path.Ext(string(file_path))))
	if recognized {
		return language, true
	}
	return Language_For_Filename(File_Name(path.Base(string(file_path))))
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
	invariant.Assertions(namespace).
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
	invariant.Assertions(namespace).
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
	invariant.Assertions(namespace).
		Range_Int(len(file_path), FILE_PATH_BYTES_MIN, FILE_PATH_BYTES_MAX).
		Ensure()
}

// Counts is the line partition of a file or a group of files.
type Counts struct {
	// Code is the number of lines bearing code.
	Code Line_Count
	// Comment is the number of comment-only lines.
	Comment Line_Count
	// Blank is the number of empty or whitespace-only lines.
	Blank Line_Count
	// Dropped is how many lines were wider than the scan window and so were only
	// partly read. It is not a fourth partition — such a line is still counted as
	// code, comment, or blank — but a classification made from a prefix, which the
	// report states rather than hides.
	Dropped Dropped_Count
}

// Counts_Invariants states each partition of the line count and the lines read short.
func Counts_Invariants(counts Counts, namespace invariant.Namespace) {
	Line_Count_Invariants(counts.Code, "Counts.Code")
	Line_Count_Invariants(counts.Comment, "Counts.Comment")
	Line_Count_Invariants(counts.Blank, "Counts.Blank")
	Dropped_Count_Invariants(counts.Dropped, "Counts.Dropped")
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
	invariant.Assertions(namespace).
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
	invariant.Assertions(namespace).
		Range_Int(int(count), LINE_COUNT_MIN, LINE_COUNT_MAX).
		Ensure()
}

// Returns the total physical line count: code, comment, and blank lines sum to it
// because every line is counted exactly once.
func counts_lines(counts Counts) (line_count Line_Count) {
	defer func() { Line_Count_Invariants(line_count, "counts_lines.line_count") }()
	Counts_Invariants(counts, "counts_lines.counts")
	return counts.Code + counts.Comment + counts.Blank
}

// Classify_File_Input is one file's bytes and the language to read them as.
type Classify_File_Input struct {
	// Source is the file's bytes.
	Source Source
	// Language is the language to read the source as.
	Language Language
}

// Classify_File_Input_Invariants states the bytes to read and the language to read
// them as.
func Classify_File_Input_Invariants(
	input Classify_File_Input, namespace invariant.Namespace,
) {
	Source_Invariants(input.Source, "Classify_File_Input.Source")
	Language_Invariants(input.Language, "Classify_File_Input.Language")
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
	invariant.Assertions(namespace).
		Range_Int(len(source), SOURCE_BYTES_MIN, SOURCE_BYTES_MAX).
		Ensure()
}

// Classify_File partitions every physical line of the source into code, comment, and
// blank counts. Each line is counted once, so the three sum to the line count.
func Classify_File(input Classify_File_Input) (counts Counts) {
	defer func() { Counts_Invariants(counts, "Classify_File.counts") }()
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
		if stop_count-start > LINE_BYTES_MAX {
			stop_count = start + LINE_BYTES_MAX
			// The tally saturates; the table says so rather than reporting a total
			// it stopped keeping.
			if counts.Dropped < DROPPED_COUNT_MAX {
				counts.Dropped++
			}
		}
		kind, carry = classify_line(Line(source[start:stop_count]), carry, &prepared)
		counts_tally(&counts, kind)
		start = index + 1
	}
	// Bytes after the last newline are a final line only when non-empty, so a trailing
	// newline adds no phantom line and the empty file is zero lines.
	if start < len(source) {
		stop_count := len(source)
		if stop_count-start > LINE_BYTES_MAX {
			stop_count = start + LINE_BYTES_MAX
			// The tally saturates; the table says so rather than reporting a total
			// it stopped keeping.
			if counts.Dropped < DROPPED_COUNT_MAX {
				counts.Dropped++
			}
		}
		kind, _ = classify_line(Line(source[start:stop_count]), carry, &prepared)
		counts_tally(&counts, kind)
	}
	return counts
}

// Adds one line's verdict to the running partition.
func counts_tally(counts *Counts, kind Line_Kind) {
	Counts_Invariants(*counts, "counts_tally.counts")
	Line_Kind_Invariants(kind, "counts_tally.kind")
	switch kind {
	case LINE_KIND_CODE:
		counts.Code++
	case LINE_KIND_COMMENT:
		counts.Comment++
	case LINE_KIND_BLANK:
		counts.Blank++
	}
}

// The partition a single physical line falls into.
type Line_Kind int

// Line_Kind_Invariants bounds a partition to the three a line can fall into. Every
// value in the range is one of them, so the range saturates.
func Line_Kind_Invariants(kind Line_Kind, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Int(int(kind), int(LINE_KIND_BLANK), int(LINE_KIND_COMMENT)).
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
	invariant.Assertions(namespace).
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
	invariant.Assertions(namespace).
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
	invariant.Assertions(namespace).
		Range_Int(int(cursor), CURSOR_MIN, CURSOR_MAX).
		Ensure()
}

// Scan_Position is the line being read and how far along it the scan has reached.
// Carrying the pair in one value is what keeps the scanner's coverage tractable: every
// helper states this one type, so the line and cursor bounds are witnessed once rather
// than separately at each of the two dozen functions that walk a line.
type Scan_Position struct {
	// Line is the non-blank line being read, truncated to the scan window.
	Line Scan_Line
	// Cursor is the byte offset the scan has reached within the line.
	Cursor Cursor
}

// Scan_Position_Invariants states the line and the offset reached within it.
func Scan_Position_Invariants(at Scan_Position, namespace invariant.Namespace) {
	Scan_Line_Invariants(at.Line, "Scan_Position.Line")
	Cursor_Invariants(at.Cursor, "Scan_Position.Cursor")
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
	invariant.Assertions(namespace).
		Range_Int(int(depth), NESTING_DEPTH_MIN, NESTING_DEPTH_MAX).
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
	invariant.Assertions(namespace).
		Range_Holed_Int(
			len(closer), BRACKET_CLOSER_BYTES_MIN, BRACKET_CLOSER_BYTES_MAX,
			BRACKET_CLOSER_BYTES_ABSENT, BRACKET_CLOSER_BYTES_ABSENT,
			BRACKET_CLOSER_BYTES_ABSENT, BRACKET_CLOSER_BYTES_ABSENT).
		Ensure()
}

// Closer is the terminator an open verbatim string needs, or empty when none is open.
type Closer string

// Closer_Invariants bounds a carried terminator's byte length.
func Closer_Invariants(closer Closer, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Int(len(closer), CLOSER_BYTES_MIN, CLOSER_BYTES_MAX).
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
	invariant.Assertions(namespace).
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
	Nesting_Depth_Invariants(carry.Block_Comment_Depth, "Scan_Carry.Block_Comment_Depth")
	Closer_Invariants(carry.Raw_String_Close, "Scan_Carry.Raw_String_Close")
	Comment_Closer_Invariants(carry.Comment_Close, "Scan_Carry.Comment_Close")
	Terminator_Invariants(carry.Heredoc_Terminator, "Scan_Carry.Heredoc_Terminator")
}

// Accumulates one line's verdict as the scanner walks it.
type Line_Scan struct {
	// Position is the line being read and how far along it the scan has reached.
	Position Scan_Position
	// State is the carried scanner state, updated as openers and closers are met.
	State Scan_Carry
	// Has_Code records that the line bears code.
	Has_Code bool
	// Has_Comment records that the line bears a comment.
	Has_Comment bool
}

// Line_Scan_Invariants states the scan's position, its carried state, and its verdict
// so far.
func Line_Scan_Invariants(scan Line_Scan, namespace invariant.Namespace) {
	Scan_Position_Invariants(scan.Position, "Line_Scan.Position")
	Scan_Carry_Invariants(scan.State, "Line_Scan.State")
	invariant.Boolean_Invariants(scan.Has_Code, "Line_Scan.Has_Code")
	invariant.Boolean_Invariants(scan.Has_Comment, "Line_Scan.Has_Comment")
}

// A scanner is one language prepared for scanning: its configuration plus a table of
// the bytes that can begin something the scan must inspect, so a run of ordinary code
// bytes is skipped in bulk instead of re-dispatched through every opener check.
type Scanner struct {
	// Language is the configuration the scan reads against.
	Language *Language
	// Trigger[b] is true when byte b can begin a comment or string opener, a heredoc, or
	// a long bracket — the only bytes a fresh-state scan must stop on. Every other
	// non-space byte is plain code.
	Trigger [256]bool
}

// Scanner_Invariants states the language a scanner reads against. The trigger table is
// a fixed array whose width the type itself pins, so it carries no bound of its own.
func Scanner_Invariants(scanner Scanner, namespace invariant.Namespace) {
	Language_Invariants(*scanner.Language, "Scanner.Language")
}

// TOKEN_BYTES_MIN is the one-byte token: a quote mark, or a single-character comment
// lead. A token is only ever matched when it is present, so it is never empty.
const TOKEN_BYTES_MIN = 1

// TOKEN_BYTES_MAX is the widest token a match is ever asked for: a long bracket's
// computed closer at the hash bound.
const TOKEN_BYTES_MAX = CLOSER_BYTES_MAX

// Token is a byte sequence the scan matches at a position: a declared delimiter, or a
// terminator computed when one was opened.
type Token string

// Token_Invariants bounds a matched token's byte length.
func Token_Invariants(token Token, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Int(len(token), TOKEN_BYTES_MIN, TOKEN_BYTES_MAX).
		Ensure()
}

// Prepares a scanner for a language. The trigger table is the union of the first byte
// of every opener the language defines, taken from its fields alone so no language is
// special-cased: miss a field and a real opener would be skipped as if it were code.
func language_scanner(language *Language) (prepared Scanner) {
	defer func() { Scanner_Invariants(prepared, "language_scanner.prepared") }()
	Language_Invariants(*language, "language_scanner.language")
	prepared.Language = language
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
		return LINE_KIND_CODE, classify_heredoc_line(Scan_Line(line), carry)
	}
	// The blank verdict above is why the scan carries a Scan_Line rather than a Line:
	// past this point the line always has content.
	scan := Line_Scan{
		Position:    Scan_Position{Line: Scan_Line(line), Cursor: CURSOR_MIN},
		State:       carry,
		Has_Code:    false,
		Has_Comment: false,
	}
	// The dispatch is inline rather than a step function of its own: the scanner and
	// the language it carries are loop-invariant, and stating them at a second
	// per-byte boundary costs more than the whole rest of the scan.
	for int(scan.Position.Cursor) < len(scan.Position.Line) {
		if scan.State.Comment_Close != "" {
			line_scan_long_comment_body(&scan)
			continue
		}
		if scan.State.Raw_String_Close != "" {
			line_scan_raw(&scan)
			continue
		}
		if scan.State.Block_Comment_Depth > 0 {
			line_scan_block(&scan, scan_with.Language)
			continue
		}
		// A byte that begins no opener reads nothing from the language, so it takes a
		// path that does not state one. That is almost every byte of a source file.
		if !scan_with.Trigger[scan.Position.Line[scan.Position.Cursor]] {
			line_scan_plain(&scan, &scan_with.Trigger)
			continue
		}
		line_scan_fresh(&scan, scan_with)
	}
	return Line_Kind(line_scan_verdict(&scan)), scan.State
}

// Reads a line inside a heredoc body: the line is code, and a line equal to the
// terminator ends the heredoc.
func classify_heredoc_line(line Scan_Line, carry Scan_Carry) (carry_after Scan_Carry) {
	defer func() {
		Scan_Carry_Invariants(carry_after, "classify_heredoc_line.carry_after")
	}()
	Scan_Line_Invariants(line, "classify_heredoc_line.line")
	Scan_Carry_Invariants(carry, "classify_heredoc_line.carry")
	if Terminator(strings.TrimSpace(string(line))) == carry.Heredoc_Terminator {
		carry.Heredoc_Terminator = ""
	}
	return carry
}

// Advances past a byte that can begin nothing: insignificant whitespace, or plain
// code. The first such code byte makes the line code, after which the run of ordinary
// bytes is skipped to the next trigger in one tight loop. The language is not stated
// here because none of it is read — which is the point, since almost every byte of a
// source file takes this path and stating a language costs more than reading one.
func line_scan_plain(scan *Line_Scan, trigger *[256]bool) {
	Line_Scan_Invariants(*scan, "line_scan_plain.scan")
	line := scan.Position.Line
	cursor := int(scan.Position.Cursor)
	if byte_is_space(Source_Byte(line[cursor])) {
		scan.Position.Cursor++
		return
	}
	scan.Has_Code = true
	cursor++
	for cursor < len(line) && !trigger[line[cursor]] {
		cursor++
	}
	scan.Position.Cursor = Cursor(cursor)
}

// Advances inside a verbatim string, where every byte is code and only the matching
// close ends it.
func line_scan_raw(scan *Line_Scan) {
	Line_Scan_Invariants(*scan, "line_scan_raw.scan")
	scan.Has_Code = true
	if has_prefix_at(scan.Position, Token(scan.State.Raw_String_Close)) {
		scan.Position.Cursor += Cursor(len(scan.State.Raw_String_Close))
		scan.State.Raw_String_Close = ""
		return
	}
	scan.Position.Cursor++
}

// Advances inside a block comment, where every byte is comment and only an open (when
// nesting) or a close moves the depth.
func line_scan_block(scan *Line_Scan, language *Language) {
	Line_Scan_Invariants(*scan, "line_scan_block.scan")
	Language_Invariants(*language, "line_scan_block.language")
	scan.Has_Comment = true
	if language.Block_Comment_Nests {
		if has_prefix_at(scan.Position, Token(language.Block_Comment_Open)) {
			// The depth saturates at its bound rather than growing with the file: a
			// comment nested deeper than the bound closes early, which keeps the
			// carried depth stated by a reachable range.
			if scan.State.Block_Comment_Depth < NESTING_DEPTH_MAX {
				scan.State.Block_Comment_Depth++
			}
			scan.Position.Cursor += Cursor(len(language.Block_Comment_Open))
			return
		}
	}
	if has_prefix_at(scan.Position, Token(language.Block_Comment_Close)) {
		scan.State.Block_Comment_Depth--
		scan.Position.Cursor += Cursor(len(language.Block_Comment_Close))
		return
	}
	scan.Position.Cursor++
}

// Advances inside a long-bracket comment, where every byte is comment and only the
// matching leveled closer ends it.
func line_scan_long_comment_body(scan *Line_Scan) {
	Line_Scan_Invariants(*scan, "line_scan_long_comment_body.scan")
	scan.Has_Comment = true
	if has_prefix_at(scan.Position, Token(scan.State.Comment_Close)) {
		scan.Position.Cursor += Cursor(len(scan.State.Comment_Close))
		scan.State.Comment_Close = ""
		return
	}
	scan.Position.Cursor++
}

// Reports whether a long-bracket comment — a line-comment token then a long bracket,
// like --[[ or --[=[ — opens at the cursor, recording the comment and its closer.
func line_scan_long_comment(scan *Line_Scan, language *Language) (opened bool) {
	defer func() {
		invariant.Boolean_Invariants(opened, "line_scan_long_comment.opened")
	}()
	Line_Scan_Invariants(*scan, "line_scan_long_comment.scan")
	Language_Invariants(*language, "line_scan_long_comment.language")
	if !language.Long_Bracket {
		return false
	}
	for _, token := range language.Line_Comment {
		if !has_prefix_at(scan.Position, Token(token)) {
			continue
		}
		// The cursor steps past the comment token so the bracket match reads from
		// the scan's own position; a failed match winds it back.
		saved := scan.Position.Cursor
		scan.Position.Cursor += Cursor(len(token))
		closer, bracketed := long_bracket_open(scan)
		if bracketed {
			scan.State.Comment_Close = Comment_Closer(closer)
			scan.Has_Comment = true
			return true
		}
		scan.Position.Cursor = saved
	}
	return false
}

// Reports whether a long-bracket string — like [[ or [=[ — opens at the cursor,
// recording its leveled closer.
func line_scan_long_string(scan *Line_Scan, language *Language) (opened bool) {
	defer func() {
		invariant.Boolean_Invariants(opened, "line_scan_long_string.opened")
	}()
	Line_Scan_Invariants(*scan, "line_scan_long_string.scan")
	Language_Invariants(*language, "line_scan_long_string.language")
	if !language.Long_Bracket {
		return false
	}
	closer, bracketed := long_bracket_open(scan)
	if !bracketed {
		return false
	}
	scan.State.Raw_String_Close = Closer(closer)
	scan.Has_Code = true
	return true
}

// Reports whether a long bracket — '[' then a run of '=' then '[' — opens at the
// cursor, recording the matching closer ']' run-of-'=' ']' and stepping the cursor past
// the opener. The cursor is left where it was when no bracket opens.
func long_bracket_open(scan *Line_Scan) (closer Bracket_Closer, opened bool) {
	defer func() {
		Bracket_Closer_Invariants(closer, "long_bracket_open.closer")
		invariant.Boolean_Invariants(opened, "long_bracket_open.opened")
	}()
	Line_Scan_Invariants(*scan, "long_bracket_open.scan")
	line := scan.Position.Line
	cursor := int(scan.Position.Cursor)
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
	scan.Position.Cursor = Cursor(read + 1)
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

// Bracket_Closer is the terminator a long bracket computes for itself. It is distinct
// from Hash_Closer because a bracket carries two brackets around its run while a raw
// string carries one quote before its hashes, so their widths never coincide.
type Bracket_Closer string

// Bracket_Closer_Invariants bounds a long bracket's computed terminator.
func Bracket_Closer_Invariants(closer Bracket_Closer, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Holed_Int(
			len(closer), BRACKET_CLOSER_BYTES_MIN, BRACKET_CLOSER_BYTES_MAX,
			BRACKET_CLOSER_BYTES_ABSENT, BRACKET_CLOSER_BYTES_ABSENT,
			BRACKET_CLOSER_BYTES_ABSENT, BRACKET_CLOSER_BYTES_ABSENT).
		Ensure()
}

// HASH_CLOSER_BYTES_MIN is the empty terminator a failed match reports alongside its
// false; a match yields at least the bare quote.
const HASH_CLOSER_BYTES_MIN = 0

// HASH_CLOSER_BYTES_MAX is a raw string at the hash bound: the quote and its hashes.
const HASH_CLOSER_BYTES_MAX = HASH_COUNT_MAX + 1

// Hash_Closer is the terminator a hashable raw string computes for itself.
type Hash_Closer string

// Hash_Closer_Invariants bounds a raw string's computed terminator.
func Hash_Closer_Invariants(closer Hash_Closer, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Int(len(closer), HASH_CLOSER_BYTES_MIN, HASH_CLOSER_BYTES_MAX).
		Ensure()
}

// Dispatches the token at the cursor when not inside a comment or string: whitespace,
// a line comment, a block-comment open, a string, or code.
func line_scan_fresh(scan *Line_Scan, scan_with *Scanner) {
	Line_Scan_Invariants(*scan, "line_scan_fresh.scan")
	Scanner_Invariants(*scan_with, "line_scan_fresh.scan_with")
	line := scan.Position.Line
	language := scan_with.Language
	if line_scan_block_open(scan, language) {
		scan.Position.Cursor += Cursor(len(language.Block_Comment_Open))
		return
	}
	// The long-bracket comment is tried before the plain line comment so Lua's --[[
	// opens a block rather than reading as a -- line comment.
	if line_scan_long_comment(scan, language) {
		return
	}
	if starts_with_any(scan.Position, language.Line_Comment) {
		// A line comment runs to end of line and cannot cross it.
		scan.Has_Comment = true
		scan.Position.Cursor = Cursor(len(line))
		return
	}
	if line_scan_long_string(scan, language) {
		return
	}
	if line_scan_heredoc(scan, language) {
		return
	}
	if verbatim_open(scan, language) {
		return
	}
	if quote_open(scan, language) {
		return
	}
	// A trigger byte that opened nothing — a lone '/', a shift '<<', a division — is just
	// code; advance one byte so the next byte re-enters the dispatch.
	scan.Has_Code = true
	scan.Position.Cursor++
}

// Reports whether a heredoc opens at the cursor, and if so records its terminator so
// the following lines are read as code until the terminator line.
func line_scan_heredoc(scan *Line_Scan, language *Language) (opened bool) {
	defer func() { invariant.Boolean_Invariants(opened, "line_scan_heredoc.opened") }()
	Line_Scan_Invariants(*scan, "line_scan_heredoc.scan")
	Language_Invariants(*language, "line_scan_heredoc.language")
	if !language.Heredoc {
		return false
	}
	terminator, found := heredoc_open(scan)
	if !found {
		return false
	}
	scan.State.Heredoc_Terminator = terminator
	scan.Has_Code = true
	return true
}

// Reports whether a heredoc opener begins at the cursor — << then an optional - or ~,
// optional space, then a quoted word or an uppercase/underscore word — and if so its
// terminator word and the opener's byte length. The uppercase rule tells <<EOF apart
// from the a << b shift operator.
func heredoc_open(scan *Line_Scan) (terminator Terminator, opened bool) {
	defer func() {
		Terminator_Invariants(terminator, "heredoc_open.terminator")
		invariant.Boolean_Invariants(opened, "heredoc_open.opened")
	}()
	Line_Scan_Invariants(*scan, "heredoc_open.scan")
	if !has_prefix_at(scan.Position, "<<") {
		return "", false
	}
	// The word is read through a position of its own so a failed match leaves the
	// scan's cursor where the dispatch found it.
	at := Scan_Position{Line: scan.Position.Line, Cursor: scan.Position.Cursor + 2}
	heredoc_skip_sigil(&at)
	heredoc_skip_spaces(&at)
	quoted := false
	if int(at.Cursor) < len(at.Line) {
		if heredoc_is_quote(Source_Byte(at.Line[at.Cursor])) {
			quoted = true
			at.Cursor++
		}
	}
	start := at.Cursor
	heredoc_skip_identifier(&at)
	if at.Cursor == start {
		return "", false
	}
	if !quoted {
		if !heredoc_word_start(Identifier_Byte(at.Line[start])) {
			return "", false
		}
	}
	terminator = Terminator(at.Line[start:at.Cursor])
	if quoted {
		if int(at.Cursor) < len(at.Line) {
			at.Cursor++
		}
	}
	scan.Position.Cursor = at.Cursor
	return terminator, true
}

// Skips an optional <<- or <<~ heredoc sigil.
func heredoc_skip_sigil(at *Scan_Position) {
	Scan_Position_Invariants(*at, "heredoc_skip_sigil.at")
	if int(at.Cursor) >= len(at.Line) {
		return
	}
	if at.Line[at.Cursor] == '-' {
		at.Cursor++
		return
	}
	if at.Line[at.Cursor] == '~' {
		at.Cursor++
	}
}

// Skips spaces and tabs between the heredoc operator and its delimiter.
func heredoc_skip_spaces(at *Scan_Position) {
	Scan_Position_Invariants(*at, "heredoc_skip_spaces.at")
	for int(at.Cursor) < len(at.Line) && byte_is_space(Source_Byte(at.Line[at.Cursor])) {
		at.Cursor++
	}
}

// Skips a run of identifier bytes.
func heredoc_skip_identifier(at *Scan_Position) {
	Scan_Position_Invariants(*at, "heredoc_skip_identifier.at")
	for int(at.Cursor) < len(at.Line) {
		if !byte_is_identifier(Source_Byte(at.Line[at.Cursor])) {
			return
		}
		at.Cursor++
	}
}

// Reports whether a byte opens a quoted heredoc delimiter.
func heredoc_is_quote(character Source_Byte) (quote bool) {
	defer func() { invariant.Boolean_Invariants(quote, "heredoc_is_quote.quote") }()
	Source_Byte_Invariants(character, "heredoc_is_quote.character")
	switch character {
	case '\'', '"', '`':
		return true
	}
	return false
}

// Reports whether a byte may begin an unquoted heredoc delimiter: an uppercase letter
// or underscore, the convention that keeps a << b from looking like a heredoc.
func heredoc_word_start(character Identifier_Byte) (start bool) {
	defer func() { invariant.Boolean_Invariants(start, "heredoc_word_start.start") }()
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
	invariant.Assertions(namespace).
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
	invariant.Assertions(namespace).
		Range_Uint8(uint8(character), uint8(SOURCE_BYTE_MIN), uint8(SOURCE_BYTE_MAX)).
		Ensure()
}

// Reports whether a block comment opens at the cursor and, when it does, records the
// comment and the new depth.
func line_scan_block_open(scan *Line_Scan, language *Language) (opened bool) {
	defer func() { invariant.Boolean_Invariants(opened, "line_scan_block_open.opened") }()
	Line_Scan_Invariants(*scan, "line_scan_block_open.scan")
	Language_Invariants(*language, "line_scan_block_open.language")
	if language.Block_Comment_Open == "" {
		return false
	}
	if !has_prefix_at(scan.Position, Token(language.Block_Comment_Open)) {
		return false
	}
	scan.State.Block_Comment_Depth = 1
	scan.Has_Comment = true
	return true
}

// Reads the line's partition from the accumulated scan: code wins a line it shares
// with a comment, then a comment, else blank.
func line_scan_verdict(scan *Line_Scan) (kind Scan_Verdict) {
	defer func() { Scan_Verdict_Invariants(kind, "line_scan_verdict.kind") }()
	Line_Scan_Invariants(*scan, "line_scan_verdict.scan")
	if scan.Has_Code {
		return Scan_Verdict(LINE_KIND_CODE)
	}
	return Scan_Verdict(LINE_KIND_COMMENT)
}

// Scan_Verdict is the partition a scanned line falls into. It is distinct from
// Line_Kind because a scan is only ever built for a line with content, and the fresh
// dispatch marks every content byte as code or comment — so a scanned line is never
// blank, whatever a physical line may be.
type Scan_Verdict int

// Scan_Verdict_Invariants bounds a scanned line's verdict to the two it can take.
func Scan_Verdict_Invariants(kind Scan_Verdict, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Int(int(kind), int(LINE_KIND_CODE), int(LINE_KIND_COMMENT)).
		Ensure()
}

// Reports whether a verbatim string begins at the cursor and, if so, its terminator
// and the opener's byte length.
func verbatim_open(scan *Line_Scan, language *Language) (opened bool) {
	defer func() { invariant.Boolean_Invariants(opened, "verbatim_open.opened") }()
	Line_Scan_Invariants(*scan, "verbatim_open.scan")
	Language_Invariants(*language, "verbatim_open.language")
	for _, delimiter := range language.Verbatim_Strings {
		if !has_prefix_at(scan.Position, Token(delimiter.Open)) {
			continue
		}
		if verbatim_lead_middle_identifier(scan.Position, delimiter) {
			continue
		}
		if !delimiter.Hashable {
			scan.State.Raw_String_Close = Closer(delimiter.Close)
			scan.Has_Code = true
			scan.Position.Cursor += Cursor(len(delimiter.Open))
			return true
		}
		closer, hashed := verbatim_hashable(scan, delimiter)
		if hashed {
			scan.State.Raw_String_Close = Closer(closer)
			scan.Has_Code = true
			return true
		}
	}
	return false
}

// Reports whether a letter-led opener sits in the middle of an identifier, where the
// lead is a name character rather than a string start.
func verbatim_lead_middle_identifier(
	at Scan_Position, delimiter Verbatim_Delimiter,
) (middle bool) {
	defer func() {
		invariant.Boolean_Invariants(middle, "verbatim_lead_middle_identifier.middle")
	}()
	Scan_Position_Invariants(at, "verbatim_lead_middle_identifier.at")
	Verbatim_Delimiter_Invariants(delimiter, "verbatim_lead_middle_identifier.delimiter")
	if !byte_is_identifier(Source_Byte(delimiter.Open[0])) {
		return false
	}
	if at.Cursor == CURSOR_MIN {
		return false
	}
	return byte_is_identifier(Source_Byte(at.Line[at.Cursor-1]))
}

// Matches a Rust-style raw-string opener: the lead, then hashes, then a quote, moving
// the cursor past it. Without the quote the lead is a raw identifier, not a string.
func verbatim_hashable(
	scan *Line_Scan, delimiter Verbatim_Delimiter,
) (closer Hash_Closer, opened bool) {
	defer func() {
		Hash_Closer_Invariants(closer, "verbatim_hashable.closer")
		invariant.Boolean_Invariants(opened, "verbatim_hashable.opened")
	}()
	Line_Scan_Invariants(*scan, "verbatim_hashable.scan")
	Verbatim_Delimiter_Invariants(delimiter, "verbatim_hashable.delimiter")
	line := scan.Position.Line
	read := int(scan.Position.Cursor) + len(delimiter.Open)
	hash_count := 0
	// The run is bounded so the computed closer stays within its stated width.
	for read < len(line) && line[read] == '#' && hash_count < HASH_COUNT_MAX {
		hash_count++
		read++
	}
	if read >= len(line) {
		return "", false
	}
	if line[read] != '"' {
		return "", false
	}
	scan.Position.Cursor += Cursor(len(delimiter.Open) + hash_count + 1)
	return Hash_Closer("\"" + strings.Repeat("#", hash_count)), true
}

// Reports whether a single-line string or character literal begins at the cursor,
// moving the cursor past what it consumes.
func quote_open(scan *Line_Scan, language *Language) (opened bool) {
	defer func() { invariant.Boolean_Invariants(opened, "quote_open.opened") }()
	Line_Scan_Invariants(*scan, "quote_open.scan")
	Language_Invariants(*language, "quote_open.language")
	for _, one := range language.Quote_Strings {
		Quote_Delimiter_Invariants(one, "quote_open.delimiter")
	}
	for _, delimiter := range language.Quote_Strings {
		if !has_prefix_at(scan.Position, Token(delimiter.Open)) {
			continue
		}
		scan.Has_Code = true
		if delimiter.Character_Like {
			scan_character_or_lifetime(&scan.Position)
			return true
		}
		scan_quoted(&scan.Position, delimiter)
		return true
	}
	return false
}

// Moves the cursor past a single-line quoted string starting at its opening delimiter,
// stopping at the first unescaped close or end of line.
func scan_quoted(at *Scan_Position, delimiter Quote_Delimiter) {
	Scan_Position_Invariants(*at, "scan_quoted.at")
	Quote_Delimiter_Invariants(delimiter, "scan_quoted.delimiter")
	line := at.Line
	read := int(at.Cursor) + len(delimiter.Open)
	for read < len(line) {
		probe := Scan_Position{Line: line, Cursor: Cursor(read)}
		if quote_escapes_here(probe, delimiter) {
			read += 2 // skip the escape byte and the character it escapes
			continue
		}
		if has_prefix_at(probe, Token(delimiter.Close)) {
			at.Cursor = Cursor(read + len(delimiter.Close))
			return
		}
		read++
	}
	at.Cursor = Cursor(len(line))
}

// Reports whether an escape sequence begins at the scan position.
func quote_escapes_here(at Scan_Position, delimiter Quote_Delimiter) (escapes bool) {
	defer func() {
		invariant.Boolean_Invariants(escapes, "quote_escapes_here.escapes")
	}()
	Scan_Position_Invariants(at, "quote_escapes_here.at")
	Quote_Delimiter_Invariants(delimiter, "quote_escapes_here.delimiter")
	// Pascal's string has no escape at all, so the zero escape must not match a zero
	// byte in the line.
	if delimiter.Escape == ESCAPE_BYTE_NONE {
		return false
	}
	return at.Line[at.Cursor] == byte(delimiter.Escape)
}

// Moves the cursor past a character or rune literal starting at the apostrophe, or by
// one when the apostrophe is a Rust lifetime tick rather than a literal — so a
// lifetime never opens a string that eats the line.
func scan_character_or_lifetime(at *Scan_Position) {
	Scan_Position_Invariants(*at, "scan_character_or_lifetime.at")
	if character_is_escaped(*at) {
		scan_escaped_character(at)
		return
	}
	if simple_character(at) {
		return
	}
	at.Cursor++
}

// Reports whether a backslash escape follows the apostrophe.
func character_is_escaped(at Scan_Position) (escaped bool) {
	defer func() {
		invariant.Boolean_Invariants(escaped, "character_is_escaped.escaped")
	}()
	Scan_Position_Invariants(at, "character_is_escaped.at")
	if int(at.Cursor)+1 >= len(at.Line) {
		return false
	}
	return at.Line[at.Cursor+1] == '\\'
}

// Moves the cursor past an escaped character literal, looking for the close past the
// escaped character, bounded so a stray apostrophe cannot scan the whole line.
func scan_escaped_character(at *Scan_Position) {
	Scan_Position_Invariants(*at, "scan_escaped_character.at")
	cursor := int(at.Cursor)
	limit := cursor + CHARACTER_ESCAPE_BYTES_MAX
	for read := cursor + 3; read < len(at.Line) && read <= limit; read++ {
		if at.Line[read] == '\'' {
			at.Cursor = Cursor(read + 1)
			return
		}
	}
	at.Cursor++
}

// CHARACTER_ESCAPE_BYTES_MAX bounds how far past an apostrophe an escaped character
// literal's close is looked for, so a stray apostrophe cannot swallow the line.
const CHARACTER_ESCAPE_BYTES_MAX = 12

// Moves the cursor past a single-rune character literal, reporting whether the
// apostrophe opened one at all.
func simple_character(at *Scan_Position) (matched bool) {
	defer func() { invariant.Boolean_Invariants(matched, "simple_character.matched") }()
	Scan_Position_Invariants(*at, "simple_character.at")
	cursor := int(at.Cursor)
	line := at.Line
	if cursor+1 >= len(line) {
		return false
	}
	if line[cursor+1] == '\'' {
		return false
	}
	_, size := utf8.DecodeRune(line[cursor+1:])
	if size <= 0 {
		return false
	}
	if cursor+1+size >= len(line) {
		return false
	}
	if line[cursor+1+size] != '\'' {
		return false
	}
	at.Cursor = Cursor(cursor + 1 + size + 1)
	return true
}

// Reports whether a line is empty or only ASCII whitespace.
func line_is_blank(line Line) (blank bool) {
	defer func() { invariant.Boolean_Invariants(blank, "line_is_blank.blank") }()
	Line_Invariants(line, "line_is_blank.line")
	for _, character := range line {
		if !byte_is_space(Source_Byte(character)) {
			return false
		}
	}
	return true
}

// Reports whether a byte is ASCII whitespace. Newline is excluded because the source
// is already split on it.
func byte_is_space(character Source_Byte) (space bool) {
	defer func() { invariant.Boolean_Invariants(space, "byte_is_space.space") }()
	Source_Byte_Invariants(character, "byte_is_space.character")
	switch character {
	case ' ', '\t', '\r', '\f', '\v':
		return true
	}
	return false
}

// Reports whether a byte may appear in an identifier, used to keep a raw-string lead
// from being recognized in the middle of a name.
func byte_is_identifier(character Source_Byte) (identifier bool) {
	defer func() {
		invariant.Boolean_Invariants(identifier, "byte_is_identifier.identifier")
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

// Reports whether prefix occurs in the line at the cursor.
func has_prefix_at(at Scan_Position, prefix Token) (match bool) {
	defer func() { invariant.Boolean_Invariants(match, "has_prefix_at.match") }()
	Scan_Position_Invariants(at, "has_prefix_at.at")
	Token_Invariants(prefix, "has_prefix_at.prefix")
	cursor := int(at.Cursor)
	if cursor+len(prefix) > len(at.Line) {
		return false
	}
	for index := 0; index < len(prefix); index++ {
		if at.Line[cursor+index] != prefix[index] {
			return false
		}
	}
	return true
}

// Reports whether any prefix occurs in the line at the cursor.
func starts_with_any(at Scan_Position, prefixes Comment_Tokens) (match bool) {
	defer func() { invariant.Boolean_Invariants(match, "starts_with_any.match") }()
	Scan_Position_Invariants(at, "starts_with_any.at")
	Comment_Tokens_Invariants(prefixes, "starts_with_any.prefixes")
	for _, prefix := range prefixes {
		if has_prefix_at(at, Token(prefix)) {
			return true
		}
	}
	return false
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
	Is_Test bool
}

// File_Count_Invariants states a counted file's path, language, partition, and role.
func File_Count_Invariants(file File_Count, namespace invariant.Namespace) {
	File_Path_Invariants(file.Path, "File_Count.Path")
	Language_Name_Invariants(file.Language, "File_Count.Language")
	Counts_Invariants(file.Counts, "File_Count.Counts")
	invariant.Boolean_Invariants(file.Is_Test, "File_Count.Is_Test")
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
	invariant.Assertions(namespace).
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
	File_Counts_Invariants(report.Files, "Report.Files")
	Skipped_Invariants(report.Skipped, "Report.Skipped")
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
	Include_Hidden bool
	// Concurrency bounds the read-and-classify worker pool; below one means one. It
	// stays an unbounded integer because it is host-supplied and count_classify
	// clamps it at both ends.
	Concurrency int
}

// Count_Input_Invariants states the walk's two settings. The file system and the
// ignore predicate are an interface and a function value with no preset of their own.
func Count_Input_Invariants(input Count_Input, namespace invariant.Namespace) {
	invariant.Boolean_Invariants(input.Include_Hidden, "Count_Input.Include_Hidden")
	invariant.Int_Invariants(input.Concurrency, "Count_Input.Concurrency")
}

// Count walks the tree, classifies every recognized file, and returns one File_Count
// per file in the lexical order the walk visited them. Reading and classifying fan
// out across workers, which does not affect the result order.
func Count(input Count_Input) (report Report, err error) {
	defer func() { Report_Invariants(report, "Count.report") }()
	Count_Input_Invariants(input, "Count.input")
	candidates, overflow, walk_err := count_candidates(
		input.File_System, input.Is_Ignored, input.Include_Hidden)
	if walk_err != nil {
		return Report{}, walk_err
	}
	files, skipped := count_classify(input.File_System, candidates, input.Concurrency)
	skipped.Overflow = overflow
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
	return Report{Files: kept, Skipped: skipped}, nil
}

// A file the walk selected for counting and the language to read it as.
type Candidate struct {
	// Path is the file's path relative to the walked root.
	Path Counted_Path
	// Language is the language the file's extension resolved to.
	Language Language
}

// Candidate_Invariants states a selected file's path and the language to read it as.
func Candidate_Invariants(candidate Candidate, namespace invariant.Namespace) {
	Counted_Path_Invariants(candidate.Path, "Candidate.Path")
	Language_Invariants(candidate.Language, "Candidate.Language")
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
	invariant.Assertions(namespace).
		Range_Int(len(file_path), COUNTED_PATH_BYTES_MIN, COUNTED_PATH_BYTES_MAX).
		Ensure()
}

// Candidates are the files a walk selected for counting.
type Candidates []Candidate

// Candidates_Invariants bounds how many files a walk selected.
func Candidates_Invariants(candidates Candidates, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Int(len(candidates), FILES_COUNT_MIN, FILES_COUNT_MAX).
		Ensure()
}

// Walks the tree and returns the recognized files to count, pruning hidden and
// ignored directories so their contents are never read.
func count_candidates(
	file_system fs.FS, is_ignored Ignore_Predicate, include_hidden bool,
) (candidates Candidates, overflow Dropped_Count, err error) {
	defer func() {
		Candidates_Invariants(candidates, "count_candidates.candidates")
		Dropped_Count_Invariants(overflow, "count_candidates.overflow")
	}()
	invariant.Boolean_Invariants(include_hidden, "count_candidates.include_hidden")
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
	is_ignored Ignore_Predicate, include_hidden bool,
) (result error) {
	Candidates_Invariants(*candidates, "count_visit.candidates")
	Dropped_Count_Invariants(*overflow, "count_visit.overflow")
	File_Path_Invariants(file_path, "count_visit.file_path")
	invariant.Boolean_Invariants(include_hidden, "count_visit.include_hidden")
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
		Language: language,
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

// Reports whether a path is a hidden entry to skip. The root, named ".", also begins
// with a dot and must not be mistaken for one.
func count_skip_hidden(file_path File_Path, include_hidden bool) (skip bool) {
	defer func() { invariant.Boolean_Invariants(skip, "count_skip_hidden.skip") }()
	File_Path_Invariants(file_path, "count_skip_hidden.file_path")
	invariant.Boolean_Invariants(include_hidden, "count_skip_hidden.include_hidden")
	if include_hidden {
		return false
	}
	if file_path == "." {
		return false
	}
	return path_is_hidden(file_path)
}

// Reports whether the injected predicate ignores a path.
func count_skip_ignored(
	file_path File_Path, entry fs.DirEntry, is_ignored Ignore_Predicate,
) (skip bool) {
	defer func() { invariant.Boolean_Invariants(skip, "count_skip_ignored.skip") }()
	File_Path_Invariants(file_path, "count_skip_ignored.file_path")
	if is_ignored == nil {
		return false
	}
	return is_ignored(string(file_path), entry.IsDir())
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

// Skip_Reason_Invariants bounds an outcome to the four that exist.
func Skip_Reason_Invariants(reason Skip_Reason, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Uint8(
			uint8(reason), uint8(SKIP_REASON_COUNTED), uint8(SKIP_REASON_BINARY)).
		Ensure()
}

// Count_Result is one candidate's outcome: the file, and why it was not counted when it
// was not.
type Count_Result struct {
	// File is the counted file, or the path alone when it was not counted.
	File File_Count
	// Reason is what became of it.
	Reason Skip_Reason
}

// Count_Result_Invariants states one candidate's outcome.
func Count_Result_Invariants(result Count_Result, namespace invariant.Namespace) {
	File_Count_Invariants(result.File, "Count_Result.File")
	Skip_Reason_Invariants(result.Reason, "Count_Result.Reason")
}

// Count_Results are the per-candidate slots the workers write into, one per candidate.
type Count_Results []Count_Result

// Count_Results_Invariants bounds how many result slots a run allocates.
func Count_Results_Invariants(results Count_Results, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Int(len(results), FILES_COUNT_MIN, FILES_COUNT_MAX).
		Ensure()
}

// Skipped is what the walk found but did not count, tallied by reason. A report carries
// it so that a file left out of the totals is stated rather than silently missing.
type Skipped struct {
	// Unreadable is how many files could not be read.
	Unreadable File_Tally
	// Oversized is how many files were wider than the source bound.
	Oversized File_Tally
	// Binary is how many files held a zero byte in their opening chunk.
	Binary File_Tally
	// Past_Lines is how many files were left out because their own language had already
	// reached the line bound. The tree is still counted as far as it goes.
	Past_Lines File_Tally
	// Overflow is how many recognized files the walk found past the file bound and so
	// could not take. The walk keeps going in order to count them, because a report
	// that says only "there were more" is the silence it exists to prevent.
	Overflow Dropped_Count
}

// Skipped_Invariants states each tally of uncounted files and whether the walk stopped
// short of the whole tree.
func Skipped_Invariants(skipped Skipped, namespace invariant.Namespace) {
	File_Tally_Invariants(skipped.Unreadable, "Skipped.Unreadable")
	File_Tally_Invariants(skipped.Oversized, "Skipped.Oversized")
	File_Tally_Invariants(skipped.Binary, "Skipped.Binary")
	File_Tally_Invariants(skipped.Past_Lines, "Skipped.Past_Lines")
	Dropped_Count_Invariants(skipped.Overflow, "Skipped.Overflow")
}

// Adds one report's uncounted tallies into another, in place.
func skipped_add(into *Skipped, more Skipped) {
	Skipped_Invariants(*into, "skipped_add.into")
	Skipped_Invariants(more, "skipped_add.more")
	into.Unreadable += more.Unreadable
	into.Oversized += more.Oversized
	into.Binary += more.Binary
	into.Past_Lines += more.Past_Lines
	into.Overflow += more.Overflow
	if into.Overflow > DROPPED_COUNT_MAX {
		into.Overflow = DROPPED_COUNT_MAX
	}
}

// Reads and classifies each candidate concurrently, dropping any unreadable or binary
// file, and returns the results in candidate order.
func count_classify(
	file_system fs.FS, candidates Candidates, concurrency int,
) (files File_Counts, skipped Skipped) {
	defer func() {
		File_Counts_Invariants(files, "count_classify.files")
		Skipped_Invariants(skipped, "count_classify.skipped")
	}()
	Candidates_Invariants(candidates, "count_classify.candidates")
	invariant.Int_Invariants(concurrency, "count_classify.concurrency")
	// Each worker writes its own slot, so candidate order is preserved without locking
	// the result slice, and every candidate leaves a slot saying what became of it.
	// Nothing the walk selected disappears without being accounted for.
	results := make(Count_Results, len(candidates))
	worker_count := concurrency
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
		go count_worker(&group, jobs, results, file_system, candidates)
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
		Count_Result_Invariants(one, "count_classify.one")
		if one.Reason == SKIP_REASON_COUNTED {
			files = append(files, one.File)
			continue
		}
		skipped_tally(&skipped, one.Reason)
	}
	return files, skipped
}

// Adds one uncounted file to the tally of its reason.
func skipped_tally(skipped *Skipped, reason Skip_Reason) {
	Skipped_Invariants(*skipped, "skipped_tally.skipped")
	Skip_Reason_Invariants(reason, "skipped_tally.reason")
	switch reason {
	case SKIP_REASON_UNREADABLE:
		skipped.Unreadable++
	case SKIP_REASON_OVERSIZED:
		skipped.Oversized++
	case SKIP_REASON_BINARY:
		skipped.Binary++
	}
}

// Drains the job channel, classifying each candidate into its slot.
func count_worker(
	group *sync.WaitGroup, jobs <-chan int, results Count_Results,
	file_system fs.FS, candidates Candidates,
) {
	Count_Results_Invariants(results, "count_worker.results")
	Candidates_Invariants(candidates, "count_worker.candidates")
	defer group.Done()
	for index := range jobs {
		one, reason := count_one(file_system, candidates[index])
		results[index] = Count_Result{File: one, Reason: reason}
	}
}

// Reads and classifies a single candidate, reporting why it was not counted when it
// was not. Every outcome is named, so nothing the walk selected leaves the pipeline
// without the report being able to say what became of it.
func count_one(file_system fs.FS, one Candidate) (file File_Count, reason Skip_Reason) {
	defer func() {
		File_Count_Invariants(file, "count_one.file")
		Skip_Reason_Invariants(reason, "count_one.reason")
	}()
	Candidate_Invariants(one, "count_one.candidate")
	uncounted := File_Count{
		Path:     File_Path(one.Path),
		Language: "",
		Counts:   Counts{Code: 0, Comment: 0, Blank: 0, Dropped: 0},
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
	counts := Classify_File(Classify_File_Input{
		Source:   Source(content),
		Language: one.Language,
	})
	return File_Count{
		Path:     File_Path(one.Path),
		Language: one.Language.Name,
		Counts:   counts,
		Is_Test:  path_is_test(one.Path, one.Language),
	}, SKIP_REASON_COUNTED
}

// Reports whether a path is test code: it lives under a test directory, or its base
// name carries the language's test prefix or infix.
func path_is_test(file_path Counted_Path, language Language) (test bool) {
	defer func() { invariant.Boolean_Invariants(test, "path_is_test.test") }()
	Counted_Path_Invariants(file_path, "path_is_test.file_path")
	Language_Invariants(language, "path_is_test.language")
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

// Reports whether any component of a path is a conventional test directory.
func path_has_test_directory(file_path Counted_Path) (found bool) {
	defer func() {
		invariant.Boolean_Invariants(found, "path_has_test_directory.found")
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

// Reports whether content holds a NUL byte in its first chunk, the cheap heuristic
// for "not text" that also guards against a no-newline blob.
func content_is_binary(content Source) (binary bool) {
	defer func() { invariant.Boolean_Invariants(binary, "content_is_binary.binary") }()
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

// Reports whether a path's final element begins with a dot.
func path_is_hidden(file_path File_Path) (hidden bool) {
	defer func() { invariant.Boolean_Invariants(hidden, "path_is_hidden.hidden") }()
	File_Path_Invariants(file_path, "path_is_hidden.file_path")
	base := path.Base(string(file_path))
	if len(base) == 0 {
		return false
	}
	return base[0] == '.'
}

// Render_Input is a report and whether to break each language into its files.
type Render_Input struct {
	// Report is the report to render.
	Report Report
	// Show_Files lists each file indented under its language.
	Show_Files bool
}

// Render_Input_Invariants states the report to render and the breakdown setting.
func Render_Input_Invariants(input Render_Input, namespace invariant.Namespace) {
	Report_Invariants(input.Report, "Render_Input.Report")
	invariant.Boolean_Invariants(input.Show_Files, "Render_Input.Show_Files")
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
	invariant.Assertions(namespace).
		Range_Int(int(count), NONZERO_COUNT_MIN, DROPPED_COUNT_MAX).
		Ensure()
}

// Nonzero_Tally is a count of files a report is actually printing, bounded like any
// file tally but never zero, for the same reason Nonzero_Count is never zero.
type Nonzero_Tally int

// Nonzero_Tally_Invariants bounds a printed file tally.
func Nonzero_Tally_Invariants(tally Nonzero_Tally, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Int(int(tally), NONZERO_COUNT_MIN, FILES_COUNT_MAX).
		Ensure()
}

// SKIP_LABEL_BYTES_MIN is the shortest label, "  files binary".
const SKIP_LABEL_BYTES_MIN = 14

// SKIP_LABEL_BYTES_MAX is the longest, "  files unreadable" and "  files past lines".
const SKIP_LABEL_BYTES_MAX = 18

// Skip_Label names one way a run left files out. It is distinct from Row_Name because
// these labels are a fixed set written here, not a path that came from a tree.
type Skip_Label string

// Skip_Label_Invariants bounds a skip label's width.
func Skip_Label_Invariants(label Skip_Label, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Int(len(label), SKIP_LABEL_BYTES_MIN, SKIP_LABEL_BYTES_MAX).
		Ensure()
}

// Returns the printed tally of lines read short. At the bound the count stops rising,
// so the cell says that it saturated rather than naming a total it stopped keeping.
func dropped_cell(dropped Nonzero_Count) (cell Line_Cell) {
	defer func() { Line_Cell_Invariants(cell, "dropped_cell.cell") }()
	Nonzero_Count_Invariants(dropped, "dropped_cell.dropped")
	if dropped == DROPPED_COUNT_MAX {
		return DROPPED_SATURATED_TEXT
	}
	return Line_Cell(with_thousands_separators(Line_Count(dropped)))
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
	return append(Dropped_Rows{skipped_label_row()}, detail...)
}

// Returns one row for each thing the run actually left out, and none for the rest.
func dropped_detail_rows(
	skipped Skipped, dropped Dropped_Count,
) (rows Dropped_Details) {
	defer func() { Dropped_Details_Invariants(rows, "dropped_detail_rows.rows") }()
	Skipped_Invariants(skipped, "dropped_detail_rows.skipped")
	Dropped_Count_Invariants(dropped, "dropped_detail_rows.dropped")
	if dropped > 0 {
		rows = append(rows, skipped_line_row(Nonzero_Count(dropped)))
	}
	if skipped.Unreadable > 0 {
		rows = append(rows, skipped_file_row(
			"  files unreadable", Nonzero_Tally(skipped.Unreadable)))
	}
	if skipped.Oversized > 0 {
		rows = append(rows, skipped_file_row(
			"  files oversized", Nonzero_Tally(skipped.Oversized)))
	}
	if skipped.Binary > 0 {
		rows = append(rows, skipped_file_row(
			"  files binary", Nonzero_Tally(skipped.Binary)))
	}
	if skipped.Past_Lines > 0 {
		rows = append(rows, skipped_file_row(
			"  files past lines", Nonzero_Tally(skipped.Past_Lines)))
	}
	if skipped.Overflow > 0 {
		rows = append(rows, skipped_overflow_row(Nonzero_Count(skipped.Overflow)))
	}
	return rows
}

// Returns the row naming how many recognized files the walk found past the file bound.
func skipped_overflow_row(overflow Nonzero_Count) (row Render_Row) {
	defer func() { Render_Row_Invariants(row, "skipped_overflow_row.row") }()
	Nonzero_Count_Invariants(overflow, "skipped_overflow_row.overflow")
	tally := File_Cell(with_thousands_separators(Line_Count(overflow)))
	if overflow == DROPPED_COUNT_MAX {
		tally = DROPPED_SATURATED_TEXT
	}
	return Render_Row{
		Name: "  files past bound", Files: tally, Lines: "",
		Code: "", Comments: "", Blanks: "", Percent: "",
	}
}

// DROPPED_DETAILS_COUNT_MIN is the empty detail of a run that left nothing out.
const DROPPED_DETAILS_COUNT_MIN = 0

// DROPPED_DETAILS_COUNT_MAX is one row for each way a run can leave something out.
const DROPPED_DETAILS_COUNT_MAX = 6

// Dropped_Details are the rows naming what a run left out, one per thing that happened.
type Dropped_Details []Render_Row

// Dropped_Details_Invariants bounds how many kinds of omission one run reports.
func Dropped_Details_Invariants(rows Dropped_Details, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
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
	invariant.Assertions(namespace).
		Range_Holed_Int(
			len(rows), DROPPED_ROWS_COUNT_MIN, DROPPED_ROWS_COUNT_MAX,
			DROPPED_ROWS_COUNT_ABSENT, DROPPED_ROWS_COUNT_ABSENT,
			DROPPED_ROWS_COUNT_ABSENT, DROPPED_ROWS_COUNT_ABSENT).
		Ensure()
}

// Returns the section's label row.
func skipped_label_row() (row Render_Row) {
	defer func() { Render_Row_Invariants(row, "skipped_label_row.row") }()
	return Render_Row{
		Name: "Dropped", Files: "", Lines: "", Code: "",
		Comments: "", Blanks: "", Percent: "",
	}
}

// Returns the row naming how many lines were read short of their full width.
func skipped_line_row(dropped Nonzero_Count) (row Render_Row) {
	defer func() { Render_Row_Invariants(row, "skipped_line_row.row") }()
	Nonzero_Count_Invariants(dropped, "skipped_line_row.dropped")
	return Render_Row{
		Name: "  lines read short", Files: "", Lines: dropped_cell(dropped),
		Code: "", Comments: "", Blanks: "", Percent: "",
	}
}

// Returns a row naming one tally of files the run did not count.
func skipped_file_row(name Skip_Label, tally Nonzero_Tally) (row Render_Row) {
	defer func() { Render_Row_Invariants(row, "skipped_file_row.row") }()
	Skip_Label_Invariants(name, "skipped_file_row.name")
	Nonzero_Tally_Invariants(tally, "skipped_file_row.tally")
	return Render_Row{
		Name:     Row_Name(name),
		Files:    File_Cell(with_thousands_separators(Line_Count(tally))),
		Lines:    "",
		Code:     "",
		Comments: "",
		Blanks:   "",
		Percent:  "",
	}
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
	invariant.Assertions(namespace).
		Range_Int(len(line), TABLE_WIDTH_MIN, TABLE_WIDTH_MAX).
		Ensure()
}

// Table_Line is one rendered line of the table, laid out but not yet trimmed. It spans
// both a row and a rule, so its bound is the wider of the two.
type Table_Line string

// Table_Line_Invariants bounds a rendered line's width.
func Table_Line_Invariants(line Table_Line, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Int(len(line), TABLE_LINE_BYTES_MIN, TABLE_LINE_BYTES_MAX).
		Ensure()
}

// A line partition with its file count, as serialized.
type Json_Counts struct {
	// Files is the number of files in the partition.
	Files File_Tally `json:"files"`
	// Code is the code-line count.
	Code Line_Count `json:"code"`
	// Comments is the comment-line count.
	Comments Line_Count `json:"comments"`
	// Blanks is the blank-line count.
	Blanks Line_Count `json:"blanks"`
}

// Json_Counts_Invariants states a serialized partition's file and line counts.
func Json_Counts_Invariants(partition Json_Counts, namespace invariant.Namespace) {
	File_Tally_Invariants(partition.Files, "Json_Counts.Files")
	Line_Count_Invariants(partition.Code, "Json_Counts.Code")
	Line_Count_Invariants(partition.Comments, "Json_Counts.Comments")
	Line_Count_Invariants(partition.Blanks, "Json_Counts.Blanks")
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
	invariant.Assertions(namespace).
		Range_Int(int(tally), GROUP_TALLY_MIN, FILES_COUNT_MAX).
		Ensure()
}

// File_Tally is a number of files, per partition or summed across a report.
type File_Tally int

// File_Tally_Invariants bounds a file tally by the same bound the walk stops at.
func File_Tally_Invariants(tally File_Tally, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Int(int(tally), FILES_COUNT_MIN, FILES_COUNT_MAX).
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
	Source Json_Counts `json:"source"`
	// Tests is the test partition.
	Tests Json_Counts `json:"tests"`
}

// Json_Language_Invariants states a serialized row's identity and its two partitions.
func Json_Language_Invariants(row Json_Language, namespace invariant.Namespace) {
	Known_Name_Invariants(row.Name, "Json_Language.Name")
	Category_Invariants(row.Category, "Json_Language.Category")
	Json_Counts_Invariants(row.Source, "Json_Language.Source")
	Json_Counts_Invariants(row.Tests, "Json_Language.Tests")
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
func Render_Json(output io.Writer, report Report) (err error) {
	Report_Invariants(report, "Render_Json.report")
	languages := []Json_Language{}
	for _, group := range report_groups(report) {
		languages = append(languages, Json_Language{
			Name:     group.Name,
			Category: group.Category,
			Source:   json_partition(group.Source_Files, group.Source),
			Tests:    json_partition(group.Test_Files, group.Test),
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
	Lines Line_Cell
	// Code is the code-line count.
	Code Line_Cell
	// Comments is the comment-line count.
	Comments Comments_Cell
	// Blanks is the blank-line count.
	Blanks Line_Cell
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
	Row_Name_Invariants(row.Name, "Render_Row.Name")
	File_Cell_Invariants(row.Files, "Render_Row.Files")
	Line_Cell_Invariants(row.Lines, "Render_Row.Lines")
	Line_Cell_Invariants(row.Code, "Render_Row.Code")
	Comments_Cell_Invariants(row.Comments, "Render_Row.Comments")
	Line_Cell_Invariants(row.Blanks, "Render_Row.Blanks")
	Percent_Cell_Invariants(row.Percent, "Render_Row.Percent")
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
	invariant.Assertions(namespace).
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
	invariant.Assertions(namespace).
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
	invariant.Assertions(namespace).
		Range_Int(len(text), TALLY_TEXT_BYTES_MIN, LINE_CELL_BYTES_MAX).
		Ensure()
}

// Line_Cell is a table row's printed line tally.
type Line_Cell string

// Line_Cell_Invariants bounds a printed line tally's width.
func Line_Cell_Invariants(cell Line_Cell, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Int(len(cell), LINE_CELL_BYTES_MIN, LINE_CELL_BYTES_MAX).
		Ensure()
}

// COMMENTS_CELL_BYTES_MAX is the comment column's own header label, "Comments", which
// is wider than any tally the column can print. A header label is a cell like any
// other, so the widest cell of this column is the label rather than a number.
const COMMENTS_CELL_BYTES_MAX = LINE_CELL_BYTES_MAX

// Comments_Cell is a table row's printed comment tally. It is distinct from Line_Cell
// because its header label is the widest thing the column ever holds.
type Comments_Cell string

// Comments_Cell_Invariants bounds a printed comment tally's width.
func Comments_Cell_Invariants(cell Comments_Cell, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Int(len(cell), LINE_CELL_BYTES_MIN, COMMENTS_CELL_BYTES_MAX).
		Ensure()
}

// PERCENT_CELL_BYTES_MIN is the empty share a row without a denominator prints.
const PERCENT_CELL_BYTES_MIN = 0

// PERCENT_CELL_BYTES_MAX is the widest share, "100.0%".
const PERCENT_CELL_BYTES_MAX = 6

// PERCENT_CELL_BYTES_NARROW is the narrowest share actually printed, "0.0%". A share
// is empty or at least this wide; the widths between are carved because one decimal
// place and a percent sign cannot be spelled in fewer bytes.
const PERCENT_CELL_BYTES_NARROW = 4

// Percent_Cell is a table row's printed share of code.
type Percent_Cell string

// Percent_Cell_Invariants bounds a printed share's width, carving the widths a
// one-decimal percentage cannot occupy.
func Percent_Cell_Invariants(cell Percent_Cell, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Holed_Int(
			len(cell), PERCENT_CELL_BYTES_MIN, PERCENT_CELL_BYTES_MAX,
			PERCENT_CELL_BYTES_NARROW-3, PERCENT_CELL_BYTES_NARROW-2,
			PERCENT_CELL_BYTES_NARROW-1, PERCENT_CELL_BYTES_NARROW-1).
		Ensure()
}

// Returns the table's column header row.
func render_header_row() (header Render_Row) {
	defer func() { Render_Row_Invariants(header, "render_header_row.header") }()
	return Render_Row{
		Name: "Language", Files: "Files", Lines: "Lines",
		Code: "Code", Comments: "Comments", Blanks: "Blanks", Percent: "%Code",
	}
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
	Counts Counts
	// Source_Files is the number of non-test files in the group.
	Source_Files File_Tally
	// Source is the summed partition of the group's non-test files.
	Source Counts
	// Test_Files is the number of test files in the group.
	Test_Files File_Tally
	// Test is the summed partition of the group's test files.
	Test Counts
	// Members are the group's files in report order.
	Members Group_Members
}

// Language_Group_Invariants states a group's identity, its three tallies, its three
// partitions, and the files it holds.
func Language_Group_Invariants(group Language_Group, namespace invariant.Namespace) {
	Known_Name_Invariants(group.Name, "Language_Group.Name")
	Category_Invariants(group.Category, "Language_Group.Category")
	Group_Tally_Invariants(group.Files, "Language_Group.Files")
	Counts_Invariants(group.Counts, "Language_Group.Counts")
	File_Tally_Invariants(group.Source_Files, "Language_Group.Source_Files")
	Counts_Invariants(group.Source, "Language_Group.Source")
	File_Tally_Invariants(group.Test_Files, "Language_Group.Test_Files")
	Counts_Invariants(group.Test, "Language_Group.Test")
	Group_Members_Invariants(group.Members, "Language_Group.Members")
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
	invariant.Assertions(namespace).
		Range_Int(len(name), KNOWN_NAME_BYTES_MIN, LANGUAGE_NAME_BYTES_MAX).
		Ensure()
}

// GROUP_MEMBERS_COUNT_MIN is the one file that brought the group into being.
const GROUP_MEMBERS_COUNT_MIN = 1

// Group_Members are one language group's files, in report order.
type Group_Members []File_Count

// Group_Members_Invariants bounds a group's membership.
func Group_Members_Invariants(members Group_Members, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
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
	invariant.Assertions(namespace).
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
	invariant.Assertions(namespace).
		Range_Int(len(groups), LANGUAGE_GROUPS_COUNT_MIN, LANGUAGE_GROUPS_COUNT_MAX).
		Ensure()
}

// Adds one line partition into another in place.
func counts_add(into *Counts, more Counts) {
	Counts_Invariants(*into, "counts_add.into")
	Counts_Invariants(more, "counts_add.more")
	into.Code += more.Code
	into.Comment += more.Comment
	into.Blank += more.Blank
	into.Dropped += more.Dropped
	if into.Dropped > DROPPED_COUNT_MAX {
		into.Dropped = DROPPED_COUNT_MAX
	}
}

// Folds a report's files into per-language groups sorted by name, splitting each
// group's partition into source and test and preserving its files in report order.
func report_groups(report Report) (groups Language_Groups) {
	defer func() { Language_Groups_Invariants(groups, "report_groups.groups") }()
	Report_Invariants(report, "report_groups.report")
	position_of := map[Language_Name]int{}
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
		counts_add(&group.Counts, file.Counts)
		if file.Is_Test {
			group.Test_Files++
			counts_add(&group.Test, file.Counts)
		} else {
			group.Source_Files++
			counts_add(&group.Source, file.Counts)
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

// CATEGORIES_COUNT_MIN is the fixed number of taxonomy buckets, which is also its
// maximum: the order is a declared list, not something a report varies.
const CATEGORIES_COUNT_MIN = 9

// CATEGORIES_COUNT_MAX is that same fixed number.
const CATEGORIES_COUNT_MAX = 9

// Categories is the fixed display order of the taxonomy buckets.
type Categories []Category

// Categories_Invariants pins the taxonomy to its declared size.
func Categories_Invariants(order Categories, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Int(len(order), CATEGORIES_COUNT_MIN, CATEGORIES_COUNT_MAX).
		Ensure()
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
	Category_Invariants(group.Name, "Category_Group.Name")
	Category_Languages_Invariants(group.Languages, "Category_Group.Languages")
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
	invariant.Assertions(namespace).
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
	invariant.Assertions(namespace).
		Range_Int(len(categories), CATEGORIES_COUNT_MIN, CATEGORIES_COUNT_MAX).
		Ensure()
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
func report_rows(categories Category_Groups, show_files bool) (rows Render_Rows) {
	defer func() { Render_Rows_Invariants(rows, "report_rows.rows") }()
	Category_Groups_Invariants(categories, "report_rows.categories")
	for _, one := range categories {
		Category_Group_Invariants(one, "report_rows.category")
	}
	invariant.Boolean_Invariants(show_files, "report_rows.show_files")
	rows = Render_Rows{render_header_row()}
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
	group Language_Group, show_files bool,
) (output Language_Rows) {
	defer func() { Language_Rows_Invariants(output, "report_language_rows.output") }()
	Language_Group_Invariants(group, "report_language_rows.group")
	invariant.Boolean_Invariants(show_files, "report_language_rows.show_files")
	output = Language_Rows{counts_row(&Counts_Row_Input{
		Name:       Row_Name("  " + group.Name),
		Files:      File_Tally(group.Files),
		Counts:     group.Counts,
		Total_Code: 0,
	})}
	if show_files {
		for _, member := range group.Members {
			output = append(output, file_row(
				Counted_Path(member.Path), member.Counts))
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

// LANGUAGE_ROWS_COUNT_MAX is that row plus one per file, which the per-file breakdown
// of a tree whose every file is that one language reaches.
const LANGUAGE_ROWS_COUNT_MAX = 1 + FILES_COUNT_MAX

// Language_Rows are the rows one language contributes to the table. They are returned
// rather than appended to the caller's accumulator, so the width they can reach is a
// property of one language rather than of wherever in the table it landed.
type Language_Rows []Render_Row

// Language_Rows_Invariants bounds one language's contribution to the table.
func Language_Rows_Invariants(rows Language_Rows, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Int(len(rows), LANGUAGE_ROWS_COUNT_MIN, LANGUAGE_ROWS_COUNT_MAX).
		Ensure()
}

// Carries split_rows's accumulator and the source and test partitions.
type Split_Rows_Input struct {
	// Source_Files is the non-test file count.
	Source_Files File_Tally
	// Source is the non-test partition.
	Source Counts
	// Test_Files is the test file count.
	Test_Files File_Tally
	// Test is the test partition.
	Test Counts
}

// Split_Rows_Input_Invariants states the accumulator and both partitions.
func Split_Rows_Input_Invariants(
	input Split_Rows_Input, namespace invariant.Namespace,
) {
	File_Tally_Invariants(input.Source_Files, "Split_Rows_Input.Source_Files")
	Counts_Invariants(input.Source, "Split_Rows_Input.Source")
	File_Tally_Invariants(input.Test_Files, "Split_Rows_Input.Test_Files")
	Counts_Invariants(input.Test, "Split_Rows_Input.Test")
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
const CATEGORIES_SHOWN_MAX = CATEGORIES_COUNT_MAX - 1

// RENDER_ROWS_COUNT_MAX is the header, every taxonomy label, a row per language, and —
// under the per-file breakdown — a row per counted file.
const RENDER_ROWS_COUNT_MAX = 1 + CATEGORIES_SHOWN_MAX +
	LANGUAGE_GROUPS_COUNT_MAX + FILES_COUNT_MAX

// RENDER_ROWS_COUNT_ABSENT is the shape that never occurs: a table is the header alone
// or the header plus a taxonomy label and at least the language row beneath it, so two
// rows is unreachable.
const RENDER_ROWS_COUNT_ABSENT = 2

// Render_Rows are the printable rows of a table, in print order.
type Render_Rows []Render_Row

// Render_Rows_Invariants bounds how many rows a table prints.
func Render_Rows_Invariants(rows Render_Rows, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
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
	return Split_Row_Pair{
		counts_row(&Counts_Row_Input{
			Name:       Row_Name(SPLIT_ROW_INDENT + "source"),
			Files:      input.Source_Files,
			Counts:     input.Source,
			Total_Code: own_code,
		}),
		counts_row(&Counts_Row_Input{
			Name:       Row_Name(SPLIT_ROW_INDENT + "tests"),
			Files:      input.Test_Files,
			Counts:     input.Test,
			Total_Code: own_code,
		}),
	}
}

// SPLIT_ROWS_COUNT_MIN is the empty split a group without tests prints.
const SPLIT_ROWS_COUNT_MIN = 0

// SPLIT_ROWS_COUNT_MAX is the source and test pair, which always appear together.
const SPLIT_ROWS_COUNT_MAX = 2

// SPLIT_ROWS_COUNT_ABSENT is the lone sub-row that never occurs: the split is a pair or
// it is nothing, since a group with tests has a source side even when that side is
// empty.
const SPLIT_ROWS_COUNT_ABSENT = 1

// Split_Row_Pair are the source and test rows printed under a group, or nothing.
type Split_Row_Pair []Render_Row

// Split_Row_Pair_Invariants pins the split to the two shapes it takes.
func Split_Row_Pair_Invariants(rows Split_Row_Pair, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Holed_Int(
			len(rows), SPLIT_ROWS_COUNT_MIN, SPLIT_ROWS_COUNT_MAX,
			SPLIT_ROWS_COUNT_ABSENT, SPLIT_ROWS_COUNT_ABSENT,
			SPLIT_ROWS_COUNT_ABSENT, SPLIT_ROWS_COUNT_ABSENT).
		Ensure()
}

// Carries counts_row's data: a name, a file count, the partition, and the code total
// the row's code is a share of.
type Counts_Row_Input struct {
	// Name is the row's label.
	Name Row_Name
	// Files is the row's file count.
	Files File_Tally
	// Counts is the row's line partition.
	Counts Counts
	// Total_Code is the denominator for the %Code share — the group's own code on a
	// source or test sub-row. Zero leaves the percentage blank, as on a language total.
	Total_Code Line_Count
}

// Counts_Row_Input_Invariants states a row's label, tally, partition, and denominator.
func Counts_Row_Input_Invariants(
	input Counts_Row_Input, namespace invariant.Namespace,
) {
	Row_Name_Invariants(input.Name, "Counts_Row_Input.Name")
	File_Tally_Invariants(input.Files, "Counts_Row_Input.Files")
	Counts_Invariants(input.Counts, "Counts_Row_Input.Counts")
	Line_Count_Invariants(input.Total_Code, "Counts_Row_Input.Total_Code")
}

// Builds an aggregate row: a name, a file count, the partition, and the code's share
// of the given code total — blank when that total is zero.
func counts_row(input *Counts_Row_Input) (row Render_Row) {
	defer func() { Render_Row_Invariants(row, "counts_row.row") }()
	Counts_Row_Input_Invariants(*input, "counts_row.input")
	percent := Percent_Cell("")
	if input.Total_Code > 0 {
		share := float64(input.Counts.Code) / float64(input.Total_Code) * 100
		percent = Percent_Cell(fmt.Sprintf("%.1f%%", share))
	}
	return Render_Row{
		Name:     input.Name,
		Files:    File_Cell(with_thousands_separators(Line_Count(input.Files))),
		Lines:    Line_Cell(with_thousands_separators(counts_lines(input.Counts))),
		Code:     Line_Cell(with_thousands_separators(input.Counts.Code)),
		Comments: Comments_Cell(with_thousands_separators(input.Counts.Comment)),
		Blanks:   Line_Cell(with_thousands_separators(input.Counts.Blank)),
		Percent:  percent,
	}
}

// Builds a per-file row: like an aggregate row but without a file count, since the
// row is itself one file, and without a percentage, which is a per-language fact.
func file_row(file_path Counted_Path, counts Counts) (row Render_Row) {
	defer func() { Render_Row_Invariants(row, "file_row.row") }()
	Counted_Path_Invariants(file_path, "file_row.file_path")
	Counts_Invariants(counts, "file_row.counts")
	row = counts_row(&Counts_Row_Input{
		Name:       Row_Name("    " + file_path),
		Files:      0,
		Counts:     counts,
		Total_Code: 0,
	})
	row.Files = ""
	return row
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
	Name_Width_Invariants(widths.Name, "Render_Column_Widths.Name")
	File_Width_Invariants(widths.Files, "Render_Column_Widths.Files")
	Lines_Width_Invariants(widths.Lines, "Render_Column_Widths.Lines")
	Code_Width_Invariants(widths.Code, "Render_Column_Widths.Code")
	Comments_Width_Invariants(widths.Comments, "Render_Column_Widths.Comments")
	Blanks_Width_Invariants(widths.Blanks, "Render_Column_Widths.Blanks")
	Percent_Width_Invariants(widths.Percent, "Render_Column_Widths.Percent")
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
	invariant.Assertions(namespace).
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

// File_Width_Invariants bounds the file-tally column's width.
func File_Width_Invariants(width File_Width, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Int(int(width), FILE_WIDTH_MIN, FILE_WIDTH_MAX).
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
	invariant.Assertions(namespace).
		Range_Int(int(width), LINES_WIDTH_MIN, LINE_WIDTH_MAX).
		Ensure()
}

// CODE_WIDTH_MIN is the header's own label, "Code".
const CODE_WIDTH_MIN = 4

// Code_Width is the printed width of the code-line column.
type Code_Width int

// Code_Width_Invariants bounds the code-line column's width.
func Code_Width_Invariants(width Code_Width, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Int(int(width), CODE_WIDTH_MIN, LINE_WIDTH_MAX).
		Ensure()
}

// COMMENTS_WIDTH_MIN is the header's own label, "Comments".
const COMMENTS_WIDTH_MIN = 8

// COMMENTS_WIDTH_MAX is that same label: it is wider than any tally the column can
// print, so this column never grows past its header.
const COMMENTS_WIDTH_MAX = LINE_WIDTH_MAX

// Comments_Width is the printed width of the comment-line column.
type Comments_Width int

// Comments_Width_Invariants bounds the comment-line column's width.
func Comments_Width_Invariants(width Comments_Width, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Int(int(width), COMMENTS_WIDTH_MIN, COMMENTS_WIDTH_MAX).
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
	invariant.Assertions(namespace).
		Range_Int(int(width), BLANKS_WIDTH_MIN, BLANKS_WIDTH_MAX).
		Ensure()
}

// PERCENT_WIDTH_MIN is the header's own label, "%Code".
const PERCENT_WIDTH_MIN = 5

// PERCENT_WIDTH_MAX is the widest share column, "100.0%".
const PERCENT_WIDTH_MAX = PERCENT_CELL_BYTES_MAX

// Percent_Width is the printed width of the code-share column.
type Percent_Width int

// Percent_Width_Invariants bounds the share column's width.
func Percent_Width_Invariants(width Percent_Width, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
		Range_Int(int(width), PERCENT_WIDTH_MIN, PERCENT_WIDTH_MAX).
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
	invariant.Assertions(namespace).
		Range_Int(int(width), COLUMN_WIDTH_MIN, COLUMN_WIDTH_MAX).
		Ensure()
}

// Table_Width is the printed width of a whole row, which the rules match. It is wider
// than any single column, so it carries the line's bound rather than a column's.
type Table_Width int

// Table_Width_Invariants bounds a whole row's printed width in characters, which is
// what the rule is drawn to and what the columns sum to.
func Table_Width_Invariants(width Table_Width, namespace invariant.Namespace) {
	invariant.Assertions(namespace).
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
	invariant.Assertions(namespace).
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
	invariant.Assertions(namespace).
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
	invariant.Assertions(namespace).
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
