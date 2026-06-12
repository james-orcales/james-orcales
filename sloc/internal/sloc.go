// Package sloc counts the code, comment, and blank lines of source files. A
// generic per-line scanner, configured by a Language, classifies each physical
// line; a walker counts a tree in parallel; a renderer prints an aligned table.
package sloc

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/james-orcales/james-orcales/shared/cli"
)

// The process exit codes.
const exit_success = 0
const exit_usage = 2
const exit_failure = 1

// Main_Input carries the command line and the host bindings Main needs. The library
// tier does no ambient I/O, so the filesystem, stat, read, and ignore operations are
// injected.
type Main_Input struct {
	// Arguments is the command line, including the program name.
	Arguments []string
	// Output is where the table is written.
	Output io.Writer
	// Error_Output is where usage and errors are written.
	Error_Output io.Writer
	// Open views a directory as a read-only file system rooted at it.
	Open func(root string) (file_system fs.FS)
	// Path_Is_Directory reports whether a path names a directory.
	Path_Is_Directory func(name string) (is_directory bool, err error)
	// Read_File reads a single file's bytes.
	Read_File func(name string) (content []byte, err error)
	// Ignore_For builds the ignore filter for a directory root, or returns nil.
	Ignore_For func(root string) (is_ignored Ignore_Predicate)
	// Concurrency bounds the file-counting worker pool.
	Concurrency int
}

// The resolved scope and override flags of one run.
type main_scope struct {
	// Paths is the list of files or directories to count.
	Paths []string
	// No_Ignore counts gitignored files too.
	No_Ignore bool
	// Include_Hidden counts hidden dot-files and dot-directories.
	Include_Hidden bool
}

// Main parses the command line, counts every path, and renders the table, returning a
// process exit code.
func Main(input *Main_Input) (status_code int) {
	program := main_program()
	command, parse_err := cli.Program_Parse(&program, input.Arguments)
	if parse_err != nil {
		fmt.Fprintf(input.Error_Output, "sloc: %v\n\n", parse_err)
		cli.Print_Help(input.Error_Output, program)
		return exit_usage
	}
	paths := cli.Get_Option(command.Arguments, "path").Value.([]string)
	// No path given counts the current directory, the obvious default for a tool run
	// inside a project.
	if len(paths) == 0 {
		paths = []string{"."}
	}
	report, count_err := main_input_collect(input, main_scope{
		Paths:          paths,
		No_Ignore:      cli.Get_Option(command.Flags, "no-ignore").Value.(bool),
		Include_Hidden: cli.Get_Option(command.Flags, "hidden").Value.(bool),
	})
	if count_err != nil {
		fmt.Fprintf(input.Error_Output, "sloc: %v\n", count_err)
		return exit_failure
	}
	if cli.Get_Option(command.Flags, "json").Value.(bool) {
		json_err := Render_Json(input.Output, report)
		if json_err != nil {
			fmt.Fprintf(input.Error_Output, "sloc: %v\n", json_err)
			return exit_failure
		}
		return exit_success
	}
	Render(input.Output, Render_Input{
		Report:     report,
		Show_Files: cli.Get_Option(command.Flags, "files").Value.(bool),
	})
	return exit_success
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
func main_input_collect(input *Main_Input, scope main_scope) (report Report, err error) {
	for _, root := range scope.Paths {
		trimmed := strings.TrimSpace(root)
		if trimmed == "" {
			continue
		}
		partial, path_err := main_input_one(input, trimmed, scope)
		if path_err != nil {
			return Report{}, path_err
		}
		report.Files = append(report.Files, partial.Files...)
	}
	return report, nil
}

// Counts a single path: a directory is walked, a file is classified directly.
func main_input_one(input *Main_Input, root string, scope main_scope) (report Report, err error) {
	directory, stat_err := input.Path_Is_Directory(root)
	if stat_err != nil {
		return Report{}, stat_err
	}
	if directory {
		return main_input_directory(input, root, scope)
	}
	file, recognized, read_err := main_input_file(input, root)
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
	input *Main_Input, root string, scope main_scope,
) (report Report, err error) {
	directory_report, count_err := Count(Count_Input{
		File_System:    input.Open(root),
		Is_Ignored:     main_input_ignore(input, root, scope.No_Ignore),
		Include_Hidden: scope.Include_Hidden,
		Concurrency:    input.Concurrency,
	})
	if count_err != nil {
		return Report{}, count_err
	}
	for _, file := range directory_report.Files {
		file.Path = path.Join(root, file.Path)
		report.Files = append(report.Files, file)
	}
	return report, nil
}

// Builds the ignore filter for a root, or nil when ignoring is off.
func main_input_ignore(
	input *Main_Input, root string, no_ignore bool,
) (is_ignored Ignore_Predicate) {
	if no_ignore {
		return nil
	}
	return input.Ignore_For(root)
}

// Classifies one explicitly named file, reporting whether its extension is recognized.
func main_input_file(input *Main_Input, name string) (file File_Count, recognized bool, err error) {
	language, known := language_for_path(name)
	if !known {
		return File_Count{}, false, nil
	}
	content, read_err := input.Read_File(name)
	if read_err != nil {
		return File_Count{}, false, read_err
	}
	counts := Classify_File(Classify_File_Input{Source: content, Language: language})
	return File_Count{Path: name, Language: language.Name, Counts: counts}, true, nil
}

// Language describes how one language's lines are read: the tokens that begin a
// comment or string, and whether its block comments nest. The scanner is generic
// over this value, so seeding another language is adding a Language.
type Language struct {
	// Name is the language's display name, shown in the rendered table.
	Name string
	// Line_Comment holds the tokens that begin a comment running to end of line.
	Line_Comment []string
	// Block_Comment_Open begins a block comment, or is empty when there is none.
	Block_Comment_Open string
	// Block_Comment_Close ends a block comment.
	Block_Comment_Close string
	// Block_Comment_Nests is true when an inner open raises the nesting depth, so a
	// single close does not end the comment (Rust); false when the first close ends
	// it (Go, C). It is the only behavioral difference between the seeds' comments.
	Block_Comment_Nests bool
	// Verbatim_Strings are the multi-line string delimiters whose bodies are verbatim.
	Verbatim_Strings []Verbatim_Delimiter
	// Quote_Strings are the single-line string and character delimiters.
	Quote_Strings []Quote_Delimiter
	// Long_Bracket enables Lua's leveled long brackets: [[ … ]] and [=[ … ]=] as
	// strings, and the same after a line-comment token (--[[ … ]]) as comments.
	Long_Bracket bool
	// Test_Prefixes are base-name prefixes that mark a test file, like Python's test_.
	Test_Prefixes []string
	// Test_Infixes are case-sensitive base-name substrings that mark a test file, like
	// Go's _test. or JavaScript's .test. / .spec., delimited so latest.go does not match.
	Test_Infixes []string
	// Heredoc enables <<DELIM heredocs (Shell, Ruby, Perl): the body lines up to a line
	// equal to DELIM are code, so a # inside the body is not read as a comment.
	Heredoc bool
}

// Verbatim_Delimiter describes a string whose body is taken verbatim and may span
// lines, so a comment token inside it is inert.
type Verbatim_Delimiter struct {
	// Open is the opening delimiter of a fixed verbatim string (a backtick or triple
	// quote), or, when Hashable, the lead before the hashes (Rust's "r" or "br").
	Open string
	// Close is the terminator of a fixed verbatim string, unused when Hashable.
	Close string
	// Hashable marks a Rust-style raw string: the lead, then N hashes, then a quote;
	// it closes only on a quote followed by exactly N hashes. Without the quote the
	// lead is a raw identifier, not a string.
	Hashable bool
}

// Quote_Delimiter describes a single-line string or character literal.
type Quote_Delimiter struct {
	// Open is the opening delimiter.
	Open string
	// Close is the closing delimiter.
	Close string
	// Escape is the byte that escapes the following character, or zero for none.
	Escape byte
	// Character_Like marks a character or rune literal, whose apostrophe must be told
	// apart from a Rust lifetime tick that opens nothing.
	Character_Like bool
}

// Language_Go returns the Go configuration: // line comments, non-nesting /* */
// block comments, backtick raw strings, and quoted literals with backslash escapes.
func Language_Go() (language Language) {
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
	return Language{
		Name:         "Makefile",
		Line_Comment: []string{"#"},
	}
}

// Language_Dockerfile returns the Dockerfile configuration: # line comments.
func Language_Dockerfile() (language Language) {
	return Language{
		Name:         "Dockerfile",
		Line_Comment: []string{"#"},
	}
}

// Language_Html returns the HTML configuration: <!-- --> comments and no line comment.
func Language_Html() (language Language) {
	return Language{
		Name:                "HTML",
		Block_Comment_Open:  "<!--",
		Block_Comment_Close: "-->",
	}
}

// Language_Xml returns the XML configuration: <!-- --> comments and no line comment.
func Language_Xml() (language Language) {
	return Language{
		Name:                "XML",
		Block_Comment_Open:  "<!--",
		Block_Comment_Close: "-->",
	}
}

// Language_Css returns the CSS configuration: /* */ comments and quoted strings.
func Language_Css() (language Language) {
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
func c_family_quotes() (delimiters []Quote_Delimiter) {
	return []Quote_Delimiter{
		{Open: "\"", Close: "\"", Escape: '\\'},
		{Open: "'", Close: "'", Escape: '\\', Character_Like: true},
	}
}

// Returns plain double- and single-quoted string delimiters with backslash escapes.
func plain_quotes() (delimiters []Quote_Delimiter) {
	return []Quote_Delimiter{
		{Open: "\"", Close: "\"", Escape: '\\'},
		{Open: "'", Close: "'", Escape: '\\'},
	}
}

// Returns a single double-quoted string delimiter with backslash escapes.
func double_quote() (delimiters []Quote_Delimiter) {
	return []Quote_Delimiter{{Open: "\"", Close: "\"", Escape: '\\'}}
}

// Language_Objective_C returns the Objective-C configuration: // and /* */ comments.
func Language_Objective_C() (language Language) {
	return Language{
		Name:                "Objective-C",
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings:       c_family_quotes(),
	}
}

// Language_Dart returns the Dart configuration: // and nesting /* */ comments, with
// ”' and """ multi-line strings.
func Language_Dart() (language Language) {
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
		Quote_Strings: plain_quotes(),
	}
}

// Language_Php returns the PHP configuration: //, #, and /* */ comments.
func Language_Php() (language Language) {
	return Language{
		Name:                "PHP",
		Test_Infixes:        []string{"Test."},
		Line_Comment:        []string{"//", "#"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings:       plain_quotes(),
	}
}

// Language_Solidity returns the Solidity configuration: // and /* */ comments.
func Language_Solidity() (language Language) {
	return Language{
		Name:                "Solidity",
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings:       plain_quotes(),
	}
}

// Language_Groovy returns the Groovy configuration: // and /* */ comments, with ”' and
// """ multi-line strings.
func Language_Groovy() (language Language) {
	return Language{
		Name:                "Groovy",
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Verbatim_Strings: []Verbatim_Delimiter{
			{Open: "\"\"\"", Close: "\"\"\""},
			{Open: "'''", Close: "'''"},
		},
		Quote_Strings: plain_quotes(),
	}
}

// Language_Verilog returns the Verilog configuration: // and /* */ comments.
func Language_Verilog() (language Language) {
	return Language{
		Name:                "Verilog",
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings:       double_quote(),
	}
}

// Language_Glsl returns the GLSL configuration: // and /* */ comments.
func Language_Glsl() (language Language) {
	return Language{
		Name:                "GLSL",
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings:       double_quote(),
	}
}

// Language_Hlsl returns the HLSL configuration: // and /* */ comments.
func Language_Hlsl() (language Language) {
	return Language{
		Name:                "HLSL",
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings:       double_quote(),
	}
}

// Language_Arduino returns the Arduino configuration: // and /* */ comments.
func Language_Arduino() (language Language) {
	return Language{
		Name:                "Arduino",
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings:       c_family_quotes(),
	}
}

// Language_Protobuf returns the Protocol Buffers configuration: // and /* */ comments.
func Language_Protobuf() (language Language) {
	return Language{
		Name:                "Protobuf",
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings:       plain_quotes(),
	}
}

// Language_Thrift returns the Thrift configuration: //, #, and /* */ comments.
func Language_Thrift() (language Language) {
	return Language{
		Name:                "Thrift",
		Line_Comment:        []string{"//", "#"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings:       plain_quotes(),
	}
}

// Language_Jsonc returns the JSONC/JSON5 configuration: // and /* */ comments.
func Language_Jsonc() (language Language) {
	return Language{
		Name:                "JSONC",
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings:       double_quote(),
	}
}

// Language_Hcl returns the HCL/Terraform configuration: #, //, and /* */ comments.
func Language_Hcl() (language Language) {
	return Language{
		Name:                "HCL",
		Line_Comment:        []string{"#", "//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Quote_Strings:       double_quote(),
	}
}

// Language_Nix returns the Nix configuration: # and /* */ comments, with ” ” strings.
func Language_Nix() (language Language) {
	return Language{
		Name:                "Nix",
		Line_Comment:        []string{"#"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Verbatim_Strings:    []Verbatim_Delimiter{{Open: "''", Close: "''"}},
		Quote_Strings:       double_quote(),
	}
}

// Language_Markdown returns the Markdown configuration: <!-- --> comments only.
func Language_Markdown() (language Language) {
	return Language{
		Name:                "Markdown",
		Block_Comment_Open:  "<!--",
		Block_Comment_Close: "-->",
	}
}

// Language_Vue returns the Vue configuration: <!-- --> comments only.
func Language_Vue() (language Language) {
	return Language{
		Name:                "Vue",
		Block_Comment_Open:  "<!--",
		Block_Comment_Close: "-->",
	}
}

// Language_Svelte returns the Svelte configuration: <!-- --> comments only.
func Language_Svelte() (language Language) {
	return Language{
		Name:                "Svelte",
		Block_Comment_Open:  "<!--",
		Block_Comment_Close: "-->",
	}
}

// Language_Astro returns the Astro configuration: <!-- --> comments only.
func Language_Astro() (language Language) {
	return Language{
		Name:                "Astro",
		Block_Comment_Open:  "<!--",
		Block_Comment_Close: "-->",
	}
}

// Language_Xaml returns the XAML configuration: <!-- --> comments only.
func Language_Xaml() (language Language) {
	return Language{
		Name:                "XAML",
		Block_Comment_Open:  "<!--",
		Block_Comment_Close: "-->",
	}
}

// Language_Xslt returns the XSLT configuration: <!-- --> comments only.
func Language_Xslt() (language Language) {
	return Language{
		Name:                "XSLT",
		Block_Comment_Open:  "<!--",
		Block_Comment_Close: "-->",
	}
}

// Language_Haskell returns the Haskell configuration: -- line comments and nesting
// {- -} block comments.
func Language_Haskell() (language Language) {
	return Language{
		Name:                "Haskell",
		Line_Comment:        []string{"--"},
		Block_Comment_Open:  "{-",
		Block_Comment_Close: "-}",
		Block_Comment_Nests: true,
		Quote_Strings:       double_quote(),
	}
}

// Language_Ocaml returns the OCaml configuration: nesting (* *) block comments and no
// line comment.
func Language_Ocaml() (language Language) {
	return Language{
		Name:                "OCaml",
		Block_Comment_Open:  "(*",
		Block_Comment_Close: "*)",
		Block_Comment_Nests: true,
		Quote_Strings:       double_quote(),
	}
}

// Language_F_Sharp returns the F# configuration: // line comments, nesting (* *) block
// comments, and """ strings.
func Language_F_Sharp() (language Language) {
	return Language{
		Name:                "F#",
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "(*",
		Block_Comment_Close: "*)",
		Block_Comment_Nests: true,
		Verbatim_Strings:    []Verbatim_Delimiter{{Open: "\"\"\"", Close: "\"\"\""}},
		Quote_Strings:       double_quote(),
	}
}

// Language_Julia returns the Julia configuration: # line comments, nesting #= =# block
// comments, and """ strings.
func Language_Julia() (language Language) {
	return Language{
		Name:                "Julia",
		Line_Comment:        []string{"#"},
		Block_Comment_Open:  "#=",
		Block_Comment_Close: "=#",
		Block_Comment_Nests: true,
		Verbatim_Strings:    []Verbatim_Delimiter{{Open: "\"\"\"", Close: "\"\"\""}},
		Quote_Strings:       double_quote(),
	}
}

// Language_Nim returns the Nim configuration: # line comments, nesting #[ ]# block
// comments, and """ strings.
func Language_Nim() (language Language) {
	return Language{
		Name:                "Nim",
		Line_Comment:        []string{"#"},
		Block_Comment_Open:  "#[",
		Block_Comment_Close: "]#",
		Block_Comment_Nests: true,
		Verbatim_Strings:    []Verbatim_Delimiter{{Open: "\"\"\"", Close: "\"\"\""}},
		Quote_Strings:       double_quote(),
	}
}

// Language_Common_Lisp returns the Common Lisp configuration: ; line comments and
// nesting #| |# block comments.
func Language_Common_Lisp() (language Language) {
	return Language{
		Name:                "Common Lisp",
		Line_Comment:        []string{";"},
		Block_Comment_Open:  "#|",
		Block_Comment_Close: "|#",
		Block_Comment_Nests: true,
		Quote_Strings:       double_quote(),
	}
}

// Language_Scheme returns the Scheme configuration: ; line comments and nesting #| |#
// block comments.
func Language_Scheme() (language Language) {
	return Language{
		Name:                "Scheme",
		Line_Comment:        []string{";"},
		Block_Comment_Open:  "#|",
		Block_Comment_Close: "|#",
		Block_Comment_Nests: true,
		Quote_Strings:       double_quote(),
	}
}

// Language_Racket returns the Racket configuration: ; line comments and nesting #| |#
// block comments.
func Language_Racket() (language Language) {
	return Language{
		Name:                "Racket",
		Line_Comment:        []string{";"},
		Block_Comment_Open:  "#|",
		Block_Comment_Close: "|#",
		Block_Comment_Nests: true,
		Quote_Strings:       double_quote(),
	}
}

// Language_Clojure returns the Clojure configuration: ; line comments only.
func Language_Clojure() (language Language) {
	return Language{
		Name:          "Clojure",
		Line_Comment:  []string{";"},
		Quote_Strings: double_quote(),
	}
}

// Language_Emacs_Lisp returns the Emacs Lisp configuration: ; line comments only.
func Language_Emacs_Lisp() (language Language) {
	return Language{
		Name:          "Emacs Lisp",
		Line_Comment:  []string{";"},
		Quote_Strings: double_quote(),
	}
}

// Language_Erlang returns the Erlang configuration: % line comments only.
func Language_Erlang() (language Language) {
	return Language{
		Name:          "Erlang",
		Line_Comment:  []string{"%"},
		Quote_Strings: double_quote(),
	}
}

// Language_Fortran returns the Fortran configuration: ! line comments only.
func Language_Fortran() (language Language) {
	return Language{
		Name:          "Fortran",
		Line_Comment:  []string{"!"},
		Quote_Strings: plain_quotes(),
	}
}

// Language_Ada returns the Ada configuration: -- line comments, with strings and the
// apostrophe attribute/character distinction.
func Language_Ada() (language Language) {
	return Language{
		Name:          "Ada",
		Line_Comment:  []string{"--"},
		Quote_Strings: c_family_quotes(),
	}
}

// Language_D returns the D configuration: // and /* */ comments, backtick raw strings,
// and strings with character literals.
func Language_D() (language Language) {
	return Language{
		Name:                "D",
		Line_Comment:        []string{"//"},
		Block_Comment_Open:  "/*",
		Block_Comment_Close: "*/",
		Verbatim_Strings:    []Verbatim_Delimiter{{Open: "`", Close: "`"}},
		Quote_Strings:       c_family_quotes(),
	}
}

// Language_Pascal returns the Pascal configuration: // line comments, { } block
// comments, and single-quoted strings.
func Language_Pascal() (language Language) {
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
	return Language{
		Name:          "R",
		Line_Comment:  []string{"#"},
		Quote_Strings: plain_quotes(),
	}
}

// Language_Elixir returns the Elixir configuration: # line comments and """ / ”'
// heredoc strings.
func Language_Elixir() (language Language) {
	return Language{
		Name:         "Elixir",
		Test_Infixes: []string{"_test."},
		Line_Comment: []string{"#"},
		Verbatim_Strings: []Verbatim_Delimiter{
			{Open: "\"\"\"", Close: "\"\"\""},
			{Open: "'''", Close: "'''"},
		},
		Quote_Strings: plain_quotes(),
	}
}

// Language_Crystal returns the Crystal configuration: # line comments only.
func Language_Crystal() (language Language) {
	return Language{
		Name:          "Crystal",
		Line_Comment:  []string{"#"},
		Quote_Strings: double_quote(),
	}
}

// Language_Power_Shell returns the PowerShell configuration: # line comments and <# #>
// block comments.
func Language_Power_Shell() (language Language) {
	return Language{
		Name:                "PowerShell",
		Line_Comment:        []string{"#"},
		Block_Comment_Open:  "<#",
		Block_Comment_Close: "#>",
		Quote_Strings:       plain_quotes(),
	}
}

// Language_Fish returns the Fish shell configuration: # line comments.
func Language_Fish() (language Language) {
	return Language{
		Name:          "Fish",
		Line_Comment:  []string{"#"},
		Quote_Strings: plain_quotes(),
	}
}

// Language_Nushell returns the Nushell configuration: # line comments.
func Language_Nushell() (language Language) {
	return Language{
		Name:          "Nushell",
		Line_Comment:  []string{"#"},
		Quote_Strings: plain_quotes(),
	}
}

// Language_Cmake returns the CMake configuration: # line comments and #[[ ]] bracket
// comments, reusing the leveled long-bracket machinery.
func Language_Cmake() (language Language) {
	return Language{
		Name:          "CMake",
		Line_Comment:  []string{"#"},
		Long_Bracket:  true,
		Quote_Strings: double_quote(),
	}
}

// Language_Tcl returns the Tcl configuration: # line comments only.
func Language_Tcl() (language Language) {
	return Language{
		Name:          "Tcl",
		Line_Comment:  []string{"#"},
		Quote_Strings: double_quote(),
	}
}

// Language_Perl returns the Perl configuration: # line comments only.
func Language_Perl() (language Language) {
	return Language{
		Name:          "Perl",
		Line_Comment:  []string{"#"},
		Heredoc:       true,
		Quote_Strings: plain_quotes(),
	}
}

// Language_Tex returns the TeX/LaTeX configuration: % line comments only.
func Language_Tex() (language Language) {
	return Language{
		Name:         "TeX",
		Line_Comment: []string{"%"},
	}
}

// Language_Visual_Basic returns the Visual Basic configuration: ' line comments only.
func Language_Visual_Basic() (language Language) {
	return Language{
		Name:          "Visual Basic",
		Line_Comment:  []string{"'"},
		Quote_Strings: double_quote(),
	}
}

// Language_For_Extension returns the seeded language for a file extension, with the
// leading dot, and whether one matched.
func Language_For_Extension(extension string) (language Language, recognized bool) {
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
func extension_match_more(extension string) (language Language, recognized bool) {
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
func extension_match_rest(extension string) (language Language, recognized bool) {
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
func Language_For_Filename(name string) (language Language, recognized bool) {
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
func language_for_path(file_path string) (language Language, recognized bool) {
	language, recognized = Language_For_Extension(path.Ext(file_path))
	if recognized {
		return language, true
	}
	return Language_For_Filename(path.Base(file_path))
}

// Counts is the line partition of a file or a group of files.
type Counts struct {
	// Code is the number of lines bearing code.
	Code int
	// Comment is the number of comment-only lines.
	Comment int
	// Blank is the number of empty or whitespace-only lines.
	Blank int
}

// Returns the total physical line count: code, comment, and blank lines sum to it
// because every line is counted exactly once.
func counts_lines(counts Counts) (lines int) {
	return counts.Code + counts.Comment + counts.Blank
}

// Classify_File_Input is one file's bytes and the language to read them as.
type Classify_File_Input struct {
	// Source is the file's bytes.
	Source []byte
	// Language is the language to read the source as.
	Language Language
}

// Classify_File partitions every physical line of the source into code, comment, and
// blank counts. Each line is counted once, so the three sum to the line count.
func Classify_File(input Classify_File_Input) (counts Counts) {
	carry := scan_carry{}
	for _, line := range source_lines(input.Source) {
		kind := line_kind_blank
		kind, carry = classify_line(line, carry, input.Language)
		if kind == line_kind_code {
			counts.Code++
		}
		if kind == line_kind_comment {
			counts.Comment++
		}
		if kind == line_kind_blank {
			counts.Blank++
		}
	}
	return counts
}

// The partition a single physical line falls into.
type line_kind int

// The line partitions.
const line_kind_blank line_kind = 0
const line_kind_code line_kind = 1
const line_kind_comment line_kind = 2

// The scanner state that crosses line boundaries. Normal strings and
// character literals never cross a line, so they are not carried.
type scan_carry struct {
	// Block_Comment_Depth is the depth of nested block comments, zero outside one.
	Block_Comment_Depth int
	// Raw_String_Close is the terminator an open verbatim string needs, or "" when
	// not inside one.
	Raw_String_Close string
	// Comment_Close is the terminator an open long-bracket comment needs, or "" when
	// not inside one.
	Comment_Close string
	// Heredoc_Terminator is the line that ends an open heredoc, or "" when not in one.
	Heredoc_Terminator string
}

// Accumulates one line's verdict as the scanner walks it.
type line_scan struct {
	// State is the carried scanner state, updated as openers and closers are met.
	State scan_carry
	// Has_Code records that the line bears code.
	Has_Code bool
	// Has_Comment records that the line bears a comment.
	Has_Comment bool
}

// Splits the source into physical lines on newline. A trailing newline yields no
// extra empty line, so the line count tracks the newline count.
func source_lines(source []byte) (lines [][]byte) {
	start := 0
	for index, character := range source {
		if character == '\n' {
			lines = append(lines, source[start:index])
			start = index + 1
		}
	}
	// Bytes after the last newline are a final line only when non-empty.
	if start < len(source) {
		lines = append(lines, source[start:])
	}
	return lines
}

// Returns the partition of one line and the carry for the next.
func classify_line(
	line []byte, carry scan_carry, language Language,
) (kind line_kind, carry_after scan_carry) {
	// A whitespace-only line is blank regardless of carried state, and neither opens
	// nor closes anything, so the carry passes through unchanged.
	if line_is_blank(line) {
		return line_kind_blank, carry
	}
	if carry.Heredoc_Terminator != "" {
		return classify_heredoc_line(line, carry)
	}
	scan := line_scan{State: carry}
	cursor := 0
	for cursor < len(line) {
		cursor = line_scan_step(&scan, line, cursor, language)
	}
	return line_scan_verdict(&scan), scan.State
}

// Reads a line inside a heredoc body: the line is code, and a line equal to the
// terminator ends the heredoc.
func classify_heredoc_line(
	line []byte, carry scan_carry,
) (kind line_kind, carry_after scan_carry) {
	if strings.TrimSpace(string(line)) == carry.Heredoc_Terminator {
		carry.Heredoc_Terminator = ""
	}
	return line_kind_code, carry
}

// Consumes the token at the cursor, updating the scan, and returns the next cursor.
func line_scan_step(scan *line_scan, line []byte, cursor int, language Language) (next int) {
	if scan.State.Comment_Close != "" {
		return line_scan_long_comment_body(scan, line, cursor)
	}
	if scan.State.Raw_String_Close != "" {
		return line_scan_raw(scan, line, cursor)
	}
	if scan.State.Block_Comment_Depth > 0 {
		return line_scan_block(scan, line, cursor, language)
	}
	return line_scan_fresh(scan, line, cursor, language)
}

// Advances inside a verbatim string, where every byte is code and only the matching
// close ends it.
func line_scan_raw(scan *line_scan, line []byte, cursor int) (next int) {
	scan.Has_Code = true
	if has_prefix_at(line, cursor, scan.State.Raw_String_Close) {
		next = cursor + len(scan.State.Raw_String_Close)
		scan.State.Raw_String_Close = ""
		return next
	}
	return cursor + 1
}

// Advances inside a block comment, where every byte is comment and only an open (when
// nesting) or a close moves the depth.
func line_scan_block(scan *line_scan, line []byte, cursor int, language Language) (next int) {
	scan.Has_Comment = true
	if language.Block_Comment_Nests {
		if has_prefix_at(line, cursor, language.Block_Comment_Open) {
			scan.State.Block_Comment_Depth++
			return cursor + len(language.Block_Comment_Open)
		}
	}
	if has_prefix_at(line, cursor, language.Block_Comment_Close) {
		scan.State.Block_Comment_Depth--
		return cursor + len(language.Block_Comment_Close)
	}
	return cursor + 1
}

// Advances inside a long-bracket comment, where every byte is comment and only the
// matching leveled closer ends it.
func line_scan_long_comment_body(scan *line_scan, line []byte, cursor int) (next int) {
	scan.Has_Comment = true
	if has_prefix_at(line, cursor, scan.State.Comment_Close) {
		next = cursor + len(scan.State.Comment_Close)
		scan.State.Comment_Close = ""
		return next
	}
	return cursor + 1
}

// Reports whether a long-bracket comment — a line-comment token then a long bracket,
// like --[[ or --[=[ — opens at the cursor, recording the comment and its closer.
func line_scan_long_comment(
	scan *line_scan, line []byte, cursor int, language Language,
) (next int, opened bool) {
	if !language.Long_Bracket {
		return 0, false
	}
	for _, token := range language.Line_Comment {
		if !has_prefix_at(line, cursor, token) {
			continue
		}
		closer, opener_size, bracketed := long_bracket_open(line, cursor+len(token))
		if !bracketed {
			continue
		}
		scan.State.Comment_Close = closer
		scan.Has_Comment = true
		return cursor + len(token) + opener_size, true
	}
	return 0, false
}

// Reports whether a long-bracket string — like [[ or [=[ — opens at the cursor,
// recording its leveled closer.
func line_scan_long_string(
	scan *line_scan, line []byte, cursor int, language Language,
) (next int, opened bool) {
	if !language.Long_Bracket {
		return 0, false
	}
	closer, opener_size, bracketed := long_bracket_open(line, cursor)
	if !bracketed {
		return 0, false
	}
	scan.State.Raw_String_Close = closer
	scan.Has_Code = true
	return cursor + opener_size, true
}

// Reports whether a long bracket — '[' then a run of '=' then '[' — opens at the
// cursor, returning the matching closer ']' run-of-'=' ']' and the opener length.
func long_bracket_open(line []byte, cursor int) (closer string, opener_size int, opened bool) {
	if cursor >= len(line) {
		return "", 0, false
	}
	if line[cursor] != '[' {
		return "", 0, false
	}
	scan := cursor + 1
	equal_count := 0
	for scan < len(line) && line[scan] == '=' {
		equal_count++
		scan++
	}
	if scan >= len(line) {
		return "", 0, false
	}
	if line[scan] != '[' {
		return "", 0, false
	}
	closer = "]" + strings.Repeat("=", equal_count) + "]"
	return closer, scan - cursor + 1, true
}

// Dispatches the token at the cursor when not inside a comment or string: whitespace,
// a line comment, a block-comment open, a string, or code.
func line_scan_fresh(scan *line_scan, line []byte, cursor int, language Language) (next int) {
	if byte_is_space(line[cursor]) {
		return cursor + 1
	}
	if line_scan_block_open(scan, line, cursor, language) {
		return cursor + len(language.Block_Comment_Open)
	}
	// The long-bracket comment is tried before the plain line comment so Lua's --[[
	// opens a block rather than reading as a -- line comment.
	if advanced, opened := line_scan_long_comment(scan, line, cursor, language); opened {
		return advanced
	}
	if starts_with_any(line, cursor, language.Line_Comment) {
		// A line comment runs to end of line and cannot cross it.
		scan.Has_Comment = true
		return len(line)
	}
	if advanced, opened := line_scan_long_string(scan, line, cursor, language); opened {
		return advanced
	}
	if advanced, opened := line_scan_heredoc(scan, line, cursor, language); opened {
		return advanced
	}
	close_delimiter, opener_size, opened := verbatim_open(line, cursor, language)
	if opened {
		scan.State.Raw_String_Close = close_delimiter
		scan.Has_Code = true
		return cursor + opener_size
	}
	consumed, quoted := quote_open(line, cursor, language)
	if quoted {
		scan.Has_Code = true
		return cursor + consumed
	}
	scan.Has_Code = true
	return cursor + 1
}

// Reports whether a heredoc opens at the cursor, and if so records its terminator so
// the following lines are read as code until the terminator line.
func line_scan_heredoc(
	scan *line_scan, line []byte, cursor int, language Language,
) (next int, opened bool) {
	if !language.Heredoc {
		return 0, false
	}
	terminator, opener_size, found := heredoc_open(line, cursor)
	if !found {
		return 0, false
	}
	scan.State.Heredoc_Terminator = terminator
	scan.Has_Code = true
	return cursor + opener_size, true
}

// Reports whether a heredoc opener begins at the cursor — << then an optional - or ~,
// optional space, then a quoted word or an uppercase/underscore word — and if so its
// terminator word and the opener's byte length. The uppercase rule tells <<EOF apart
// from the a << b shift operator.
func heredoc_open(line []byte, cursor int) (terminator string, opener_size int, opened bool) {
	if !has_prefix_at(line, cursor, "<<") {
		return "", 0, false
	}
	scan := heredoc_skip_spaces(line, heredoc_skip_sigil(line, cursor+2))
	quoted := false
	if scan < len(line) {
		if heredoc_is_quote(line[scan]) {
			quoted = true
			scan++
		}
	}
	start := scan
	scan = heredoc_skip_identifier(line, scan)
	if scan == start {
		return "", 0, false
	}
	if !quoted {
		if !heredoc_word_start(line[start]) {
			return "", 0, false
		}
	}
	terminator = string(line[start:scan])
	if quoted {
		if scan < len(line) {
			scan++
		}
	}
	return terminator, scan - cursor, true
}

// Skips an optional <<- or <<~ heredoc sigil.
func heredoc_skip_sigil(line []byte, cursor int) (next int) {
	if cursor >= len(line) {
		return cursor
	}
	if line[cursor] == '-' {
		return cursor + 1
	}
	if line[cursor] == '~' {
		return cursor + 1
	}
	return cursor
}

// Skips spaces and tabs between the heredoc operator and its delimiter.
func heredoc_skip_spaces(line []byte, cursor int) (next int) {
	for cursor < len(line) && byte_is_space(line[cursor]) {
		cursor++
	}
	return cursor
}

// Skips a run of identifier bytes.
func heredoc_skip_identifier(line []byte, cursor int) (next int) {
	for cursor < len(line) && byte_is_identifier(line[cursor]) {
		cursor++
	}
	return cursor
}

// Reports whether a byte opens a quoted heredoc delimiter.
func heredoc_is_quote(character byte) (quote bool) {
	switch character {
	case '\'', '"', '`':
		return true
	}
	return false
}

// Reports whether a byte may begin an unquoted heredoc delimiter: an uppercase letter
// or underscore, the convention that keeps a << b from looking like a heredoc.
func heredoc_word_start(character byte) (start bool) {
	if character == '_' {
		return true
	}
	return character >= 'A' && character <= 'Z'
}

// Reports whether a block comment opens at the cursor and, when it does, records the
// comment and the new depth.
func line_scan_block_open(
	scan *line_scan, line []byte, cursor int, language Language,
) (opened bool) {
	if language.Block_Comment_Open == "" {
		return false
	}
	if !has_prefix_at(line, cursor, language.Block_Comment_Open) {
		return false
	}
	scan.State.Block_Comment_Depth = 1
	scan.Has_Comment = true
	return true
}

// Reads the line's partition from the accumulated scan: code wins a line it shares
// with a comment, then a comment, else blank.
func line_scan_verdict(scan *line_scan) (kind line_kind) {
	if scan.Has_Code {
		return line_kind_code
	}
	if scan.Has_Comment {
		return line_kind_comment
	}
	return line_kind_blank
}

// Reports whether a verbatim string begins at the cursor and, if so, its terminator
// and the opener's byte length.
func verbatim_open(
	line []byte, cursor int, language Language,
) (close_delimiter string, opener_size int, opened bool) {
	for _, delimiter := range language.Verbatim_Strings {
		if !has_prefix_at(line, cursor, delimiter.Open) {
			continue
		}
		if verbatim_lead_middle_identifier(line, cursor, delimiter) {
			continue
		}
		if !delimiter.Hashable {
			return delimiter.Close, len(delimiter.Open), true
		}
		close_delimiter, opener_size, opened = verbatim_hashable(line, cursor, delimiter)
		if opened {
			return close_delimiter, opener_size, true
		}
	}
	return "", 0, false
}

// Reports whether a letter-led opener sits in the middle of an identifier, where the
// lead is a name character rather than a string start.
func verbatim_lead_middle_identifier(
	line []byte, cursor int, delimiter Verbatim_Delimiter,
) (middle bool) {
	if !byte_is_identifier(delimiter.Open[0]) {
		return false
	}
	if cursor == 0 {
		return false
	}
	return byte_is_identifier(line[cursor-1])
}

// Matches a Rust-style raw-string opener: the lead, then hashes, then a quote.
// Without the quote the lead is a raw identifier, not a string.
func verbatim_hashable(
	line []byte, cursor int, delimiter Verbatim_Delimiter,
) (close_delimiter string, opener_size int, opened bool) {
	scan := cursor + len(delimiter.Open)
	hash_count := 0
	for scan < len(line) && line[scan] == '#' {
		hash_count++
		scan++
	}
	if scan >= len(line) {
		return "", 0, false
	}
	if line[scan] != '"' {
		return "", 0, false
	}
	close_delimiter = "\"" + strings.Repeat("#", hash_count)
	return close_delimiter, len(delimiter.Open) + hash_count + 1, true
}

// Reports whether a single-line string or character literal begins at the cursor and,
// if so, the byte length it consumes.
func quote_open(line []byte, cursor int, language Language) (consumed int, opened bool) {
	for _, delimiter := range language.Quote_Strings {
		if !has_prefix_at(line, cursor, delimiter.Open) {
			continue
		}
		if delimiter.Character_Like {
			return scan_character_or_lifetime(line, cursor), true
		}
		return scan_quoted(line, cursor, delimiter), true
	}
	return 0, false
}

// Returns the byte length of a single-line quoted string starting at its opening
// delimiter, stopping at the first unescaped close or end of line.
func scan_quoted(line []byte, cursor int, delimiter Quote_Delimiter) (consumed int) {
	scan := cursor + len(delimiter.Open)
	for scan < len(line) {
		if quote_escapes_here(line, scan, delimiter) {
			scan += 2 // skip the escape byte and the character it escapes
			continue
		}
		if has_prefix_at(line, scan, delimiter.Close) {
			return scan + len(delimiter.Close) - cursor
		}
		scan++
	}
	return len(line) - cursor
}

// Reports whether an escape sequence begins at the scan position.
func quote_escapes_here(line []byte, scan int, delimiter Quote_Delimiter) (escapes bool) {
	if delimiter.Escape == 0 {
		return false
	}
	return line[scan] == delimiter.Escape
}

// Returns the byte length of a character or rune literal starting at the apostrophe,
// or 1 when the apostrophe is a Rust lifetime tick rather than a literal — so a
// lifetime never opens a string that eats the line.
func scan_character_or_lifetime(line []byte, cursor int) (consumed int) {
	if character_is_escaped(line, cursor) {
		return scan_escaped_character(line, cursor)
	}
	simple := simple_character(line, cursor)
	if simple > 0 {
		return simple
	}
	return 1
}

// Reports whether a backslash escape follows the apostrophe.
func character_is_escaped(line []byte, cursor int) (escaped bool) {
	if cursor+1 >= len(line) {
		return false
	}
	return line[cursor+1] == '\\'
}

// Returns the length of an escaped character literal, looking for the close past the
// escaped character, bounded so a stray apostrophe cannot scan the whole line.
func scan_escaped_character(line []byte, cursor int) (consumed int) {
	limit := cursor + 12
	for scan := cursor + 3; scan < len(line) && scan <= limit; scan++ {
		if line[scan] == '\'' {
			return scan - cursor + 1
		}
	}
	return 1
}

// Returns the length of a single-rune character literal, or zero when the apostrophe
// does not open one.
func simple_character(line []byte, cursor int) (consumed int) {
	if cursor+1 >= len(line) {
		return 0
	}
	if line[cursor+1] == '\'' {
		return 0
	}
	_, size := utf8.DecodeRune(line[cursor+1:])
	if size <= 0 {
		return 0
	}
	if cursor+1+size >= len(line) {
		return 0
	}
	if line[cursor+1+size] != '\'' {
		return 0
	}
	return 1 + size + 1
}

// Reports whether a line is empty or only ASCII whitespace.
func line_is_blank(line []byte) (blank bool) {
	for _, character := range line {
		if !byte_is_space(character) {
			return false
		}
	}
	return true
}

// Reports whether a byte is ASCII whitespace. Newline is excluded because the source
// is already split on it.
func byte_is_space(character byte) (space bool) {
	switch character {
	case ' ', '\t', '\r', '\f', '\v':
		return true
	}
	return false
}

// Reports whether a byte may appear in an identifier, used to keep a raw-string lead
// from being recognized in the middle of a name.
func byte_is_identifier(character byte) (identifier bool) {
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

// Reports whether prefix occurs in line at the cursor.
func has_prefix_at(line []byte, cursor int, prefix string) (match bool) {
	if cursor+len(prefix) > len(line) {
		return false
	}
	for index := 0; index < len(prefix); index++ {
		if line[cursor+index] != prefix[index] {
			return false
		}
	}
	return true
}

// Reports whether any prefix occurs in line at the cursor.
func starts_with_any(line []byte, cursor int, prefixes []string) (match bool) {
	for _, prefix := range prefixes {
		if has_prefix_at(line, cursor, prefix) {
			return true
		}
	}
	return false
}

// File_Count is one counted file: its path, the language it was read as, and its
// line partition.
type File_Count struct {
	// Path is the file's path.
	Path string
	// Language is the display name of the language it was read as.
	Language string
	// Counts is the file's line partition.
	Counts Counts
	// Is_Test reports whether the file is test code rather than source.
	Is_Test bool
}

// Report is the result of counting a tree: one File_Count per counted file, in the
// lexical order the walk visited them.
type Report struct {
	// Files are the counted files.
	Files []File_Count
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
	// Concurrency bounds the read-and-classify worker pool; below one means one.
	Concurrency int
}

// Count walks the tree, classifies every recognized file, and returns one File_Count
// per file in the lexical order the walk visited them. Reading and classifying fan
// out across workers, which does not affect the result order.
func Count(input Count_Input) (report Report, err error) {
	candidates, walk_err := count_candidates(
		input.File_System, input.Is_Ignored, input.Include_Hidden)
	if walk_err != nil {
		return Report{}, walk_err
	}
	files := count_classify(input.File_System, candidates, input.Concurrency)
	return Report{Files: files}, nil
}

// A file the walk selected for counting and the language to read it as.
type candidate struct {
	// Path is the file's path relative to the walked root.
	Path string
	// Language is the language the file's extension resolved to.
	Language Language
}

// Walks the tree and returns the recognized files to count, pruning hidden and
// ignored directories so their contents are never read.
func count_candidates(
	file_system fs.FS, is_ignored Ignore_Predicate, include_hidden bool,
) (candidates []candidate, err error) {
	walk_err := fs.WalkDir(file_system, ".",
		func(file_path string, entry fs.DirEntry, step_err error) (result error) {
			return count_visit(
				&candidates, file_path, entry, step_err, is_ignored, include_hidden)
		})
	if walk_err != nil {
		return nil, walk_err
	}
	return candidates, nil
}

// Decides one walked entry: prune it, skip it, or append it as a candidate.
func count_visit(
	candidates *[]candidate, file_path string, entry fs.DirEntry, step_err error,
	is_ignored Ignore_Predicate, include_hidden bool,
) (result error) {
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
	*candidates = append(*candidates, candidate{Path: file_path, Language: language})
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
func count_skip_hidden(file_path string, include_hidden bool) (skip bool) {
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
	file_path string, entry fs.DirEntry, is_ignored Ignore_Predicate,
) (skip bool) {
	if is_ignored == nil {
		return false
	}
	return is_ignored(file_path, entry.IsDir())
}

// Reads and classifies each candidate concurrently, dropping any unreadable or binary
// file, and returns the results in candidate order.
func count_classify(
	file_system fs.FS, candidates []candidate, concurrency int,
) (files []File_Count) {
	// Each worker writes its own slot, so candidate order is preserved without locking
	// the result slice; a dropped file leaves a nil slot.
	results := make([]*File_Count, len(candidates))
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
		if one != nil {
			files = append(files, *one)
		}
	}
	return files
}

// Drains the job channel, classifying each candidate into its slot.
func count_worker(
	group *sync.WaitGroup, jobs <-chan int, results []*File_Count,
	file_system fs.FS, candidates []candidate,
) {
	defer group.Done()
	for index := range jobs {
		results[index] = count_one(file_system, candidates[index])
	}
}

// Reads and classifies a single candidate, returning nil to drop a file that is
// unreadable or binary.
func count_one(file_system fs.FS, one candidate) (file *File_Count) {
	content, read_err := fs.ReadFile(file_system, one.Path)
	if read_err != nil {
		return nil
	}
	// Detection is by extension, so a recognized extension holding binary data is
	// dropped here rather than counted as garbage lines.
	if content_is_binary(content) {
		return nil
	}
	counts := Classify_File(Classify_File_Input{Source: content, Language: one.Language})
	return &File_Count{
		Path:     one.Path,
		Language: one.Language.Name,
		Counts:   counts,
		Is_Test:  path_is_test(one.Path, one.Language),
	}
}

// Reports whether a path is test code: it lives under a test directory, or its base
// name carries the language's test prefix or infix.
func path_is_test(file_path string, language Language) (test bool) {
	if path_has_test_directory(file_path) {
		return true
	}
	base := path.Base(file_path)
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
func path_has_test_directory(file_path string) (found bool) {
	for _, component := range strings.Split(file_path, "/") {
		switch component {
		case "test", "tests", "spec", "__tests__":
			return true
		}
	}
	return false
}

// Bounds how far content_is_binary scans for a NUL byte.
const binary_sniff_bytes = 8192

// Reports whether content holds a NUL byte in its first chunk, the cheap heuristic
// for "not text" that also guards against a no-newline blob.
func content_is_binary(content []byte) (binary bool) {
	limit_count := len(content)
	if limit_count > binary_sniff_bytes {
		limit_count = binary_sniff_bytes
	}
	for index := 0; index < limit_count; index++ {
		if content[index] == 0 {
			return true
		}
	}
	return false
}

// Reports whether a path's final element begins with a dot.
func path_is_hidden(file_path string) (hidden bool) {
	base := path.Base(file_path)
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

// Render writes the report as an aligned table: one row per language sorted by name,
// a Total row, and — when Show_Files is set — each file indented under its language.
// Columns are sized to their widest cell, so the output is stable for a given report.
func Render(output io.Writer, input Render_Input) {
	groups := report_groups(input.Report)
	categories := report_categories(groups)
	rows := report_rows(categories, input.Show_Files)
	totals := report_total(groups)
	widths := render_widths(append(slices.Clone(rows), totals...))
	rule := strings.Repeat("─", render_column_widths_total(widths))
	render_write(output, rule)
	render_write(output, render_row_format(rows[0], widths))
	render_write(output, rule)
	for _, row := range rows[1:] {
		render_write(output, render_row_format(row, widths))
	}
	render_write(output, rule)
	for _, row := range totals {
		render_write(output, render_row_format(row, widths))
	}
	render_write(output, rule)
}

// Prints one table line, trimmed of the trailing padding a label row leaves.
func render_write(output io.Writer, line string) {
	fmt.Fprintln(output, strings.TrimRight(line, " "))
}

// A line partition with its file count, as serialized.
type json_counts struct {
	Files    int `json:"files"`
	Code     int `json:"code"`
	Comments int `json:"comments"`
	Blanks   int `json:"blanks"`
}

// One language's serialized counts, or the total when name and category are omitted.
type json_language struct {
	Name     string      `json:"name,omitempty"`
	Category string      `json:"category,omitempty"`
	Files    int         `json:"files"`
	Code     int         `json:"code"`
	Comments int         `json:"comments"`
	Blanks   int         `json:"blanks"`
	Source   json_counts `json:"source"`
	Tests    json_counts `json:"tests"`
}

// The serialized report: the languages and the total.
type json_report struct {
	Languages []json_language `json:"languages"`
	Total     json_language   `json:"total"`
}

// Builds a serialized partition from a file count and a partition.
func json_partition(files int, counts Counts) (partition json_counts) {
	return json_counts{
		Files:    files,
		Code:     counts.Code,
		Comments: counts.Comment,
		Blanks:   counts.Blank,
	}
}

// Render_Json writes the report as indented JSON: a name-sorted languages array, each
// with its category and source/test split, and a total.
func Render_Json(output io.Writer, report Report) (err error) {
	document := json_report{Languages: []json_language{}}
	for _, group := range report_groups(report) {
		document.Languages = append(document.Languages, json_language{
			Name:     group.Name,
			Category: group.Category,
			Files:    group.Files,
			Code:     group.Counts.Code,
			Comments: group.Counts.Comment,
			Blanks:   group.Counts.Blank,
			Source:   json_partition(group.Source_Files, group.Source),
			Tests:    json_partition(group.Test_Files, group.Test),
		})
		json_language_add(&document.Total, group)
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(document)
}

// Accumulates a language group into the JSON total.
func json_language_add(total *json_language, group language_group) {
	total.Files += group.Files
	total.Code += group.Counts.Code
	total.Comments += group.Counts.Comment
	total.Blanks += group.Counts.Blank
	total.Source.Files += group.Source_Files
	total.Source.Code += group.Source.Code
	total.Source.Comments += group.Source.Comment
	total.Source.Blanks += group.Source.Blank
	total.Tests.Files += group.Test_Files
	total.Tests.Code += group.Test.Code
	total.Tests.Comments += group.Test.Comment
	total.Tests.Blanks += group.Test.Blank
}

// One printable table row; every cell is already a string so the header
// and the numeric rows share one width and formatting path.
type render_row struct {
	// Name is the language name, or an indented file path.
	Name string
	// Files is the file count, empty on a per-file row.
	Files string
	// Lines is the total line count.
	Lines string
	// Code is the code-line count.
	Code string
	// Comments is the comment-line count.
	Comments string
	// Blanks is the blank-line count.
	Blanks string
	// Percent is the row's code as a share of total code.
	Percent string
}

// Returns the table's column header row.
func render_header_row() (header render_row) {
	return render_row{
		Name: "Language", Files: "Files", Lines: "Lines",
		Code: "Code", Comments: "Comments", Blanks: "Blanks", Percent: "%Code",
	}
}

// One language's files and their summed partition.
type language_group struct {
	// Name is the language's display name.
	Name string
	// Category is the language's taxonomy bucket.
	Category string
	// Files is the number of files in the group.
	Files int
	// Counts is the group's summed line partition.
	Counts Counts
	// Source_Files is the number of non-test files in the group.
	Source_Files int
	// Source is the summed partition of the group's non-test files.
	Source Counts
	// Test_Files is the number of test files in the group.
	Test_Files int
	// Test is the summed partition of the group's test files.
	Test Counts
	// Members are the group's files in report order.
	Members []File_Count
}

// Adds one line partition into another in place.
func counts_add(into *Counts, more Counts) {
	into.Code += more.Code
	into.Comment += more.Comment
	into.Blank += more.Blank
}

// Folds a report's files into per-language groups sorted by name, splitting each
// group's partition into source and test and preserving its files in report order.
func report_groups(report Report) (groups []language_group) {
	position_of := map[string]int{}
	for _, file := range report.Files {
		position, seen := position_of[file.Language]
		if !seen {
			position = len(groups)
			position_of[file.Language] = position
			groups = append(groups, language_group{
				Name:     file.Language,
				Category: language_category(file.Language),
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
	slices.SortFunc(groups, func(a language_group, b language_group) (order int) {
		return strings.Compare(a.Name, b.Name)
	})
	return groups
}

// Returns a language's taxonomy category by display name. This switch is the single
// place the classification lives.
func language_category(name string) (category string) {
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
func category_order() (order []string) {
	return []string{
		"Systems", "Managed", "Dynamically Typed", "Shell",
		"Markup & Data", "Build & Config", "Query", "Hardware", "Other",
	}
}

// A category and the language groups it holds, in name order.
type category_group struct {
	// Name is the category's display name.
	Name string
	// Languages are the category's language groups in name order.
	Languages []language_group
}

// Buckets the name-sorted language groups into categories in the fixed display order.
func report_categories(groups []language_group) (categories []category_group) {
	position_of := map[string]int{}
	for _, name := range category_order() {
		position_of[name] = len(categories)
		categories = append(categories, category_group{Name: name})
	}
	for _, group := range groups {
		position := position_of[group.Category]
		categories[position].Languages = append(categories[position].Languages, group)
	}
	return categories
}

// Builds the header, then each non-empty category: a label row followed by its
// languages, each with its files (show_files) or its source/test split.
func report_rows(categories []category_group, show_files bool) (rows []render_row) {
	rows = []render_row{render_header_row()}
	for _, category := range categories {
		if len(category.Languages) == 0 {
			continue
		}
		rows = append(rows, render_row{Name: category.Name})
		for _, group := range category.Languages {
			rows = report_language_rows(rows, group, show_files)
		}
	}
	return rows
}

// Appends a language's indented row, then its files (with show_files) or its source and
// test sub-rows.
func report_language_rows(
	rows []render_row, group language_group, show_files bool,
) (output []render_row) {
	output = append(rows, counts_row(&counts_row_input{
		Name:   "  " + group.Name,
		Files:  group.Files,
		Counts: group.Counts,
	}))
	if show_files {
		for _, member := range group.Members {
			output = append(output, file_row("    "+member.Path, member.Counts))
		}
		return output
	}
	return split_rows(&split_rows_input{
		Indent:       "    ",
		Rows:         output,
		Source_Files: group.Source_Files,
		Source:       group.Source,
		Test_Files:   group.Test_Files,
		Test:         group.Test,
	})
}

// Carries split_rows's accumulator and the source and test partitions.
type split_rows_input struct {
	// Indent is the leading whitespace for the source and test sub-rows.
	Indent string
	// Rows is the accumulator the sub-rows are appended to.
	Rows []render_row
	// Source_Files is the non-test file count.
	Source_Files int
	// Source is the non-test partition.
	Source Counts
	// Test_Files is the test file count.
	Test_Files int
	// Test is the test partition.
	Test Counts
}

// Appends indented source and test sub-rows, but only when test files are present — a
// language without tests shows just its single total row. The %Code on a sub-row is its
// share of the group's own code, so source and tests sum to 100% independent of how
// large the group is relative to the whole.
func split_rows(input *split_rows_input) (split []render_row) {
	split = input.Rows
	if input.Test_Files == 0 {
		return split
	}
	own_code := input.Source.Code + input.Test.Code
	split = append(split, counts_row(&counts_row_input{
		Name:       input.Indent + "source",
		Files:      input.Source_Files,
		Counts:     input.Source,
		Total_Code: own_code,
	}))
	split = append(split, counts_row(&counts_row_input{
		Name:       input.Indent + "tests",
		Files:      input.Test_Files,
		Counts:     input.Test,
		Total_Code: own_code,
	}))
	return split
}

// Sums every group into the Total row and its source and test sub-rows.
func report_total(groups []language_group) (totals []render_row) {
	combined := Counts{}
	source := Counts{}
	test := Counts{}
	files := 0
	source_files := 0
	test_files := 0
	for _, group := range groups {
		counts_add(&combined, group.Counts)
		counts_add(&source, group.Source)
		counts_add(&test, group.Test)
		files += group.Files
		source_files += group.Source_Files
		test_files += group.Test_Files
	}
	totals = []render_row{counts_row(&counts_row_input{
		Name:   "Total",
		Files:  files,
		Counts: combined,
	})}
	return split_rows(&split_rows_input{
		Indent:       "  ",
		Rows:         totals,
		Source_Files: source_files,
		Source:       source,
		Test_Files:   test_files,
		Test:         test,
	})
}

// Carries counts_row's data: a name, a file count, the partition, and the code total
// the row's code is a share of.
type counts_row_input struct {
	// Name is the row's label.
	Name string
	// Files is the row's file count.
	Files int
	// Counts is the row's line partition.
	Counts Counts
	// Total_Code is the denominator for the %Code share — the group's own code on a
	// source or test sub-row. Zero leaves the percentage blank, as on a language total.
	Total_Code int
}

// Builds an aggregate row: a name, a file count, the partition, and the code's share
// of the given code total — blank when that total is zero.
func counts_row(input *counts_row_input) (row render_row) {
	percent := ""
	if input.Total_Code > 0 {
		share := float64(input.Counts.Code) / float64(input.Total_Code) * 100
		percent = fmt.Sprintf("%.1f%%", share)
	}
	return render_row{
		Name:     input.Name,
		Files:    with_thousands_separators(input.Files),
		Lines:    with_thousands_separators(counts_lines(input.Counts)),
		Code:     with_thousands_separators(input.Counts.Code),
		Comments: with_thousands_separators(input.Counts.Comment),
		Blanks:   with_thousands_separators(input.Counts.Blank),
		Percent:  percent,
	}
}

// Builds a per-file row: like an aggregate row but without a file count, since the
// row is itself one file, and without a percentage, which is a per-language fact.
func file_row(name string, counts Counts) (row render_row) {
	row = counts_row(&counts_row_input{Name: name, Counts: counts})
	row.Files = ""
	return row
}

// The printed width of each table column.
type render_column_widths struct {
	// Name is the width of the language and file-path column.
	Name int
	// Files is the width of the file-count column.
	Files int
	// Lines is the width of the line-count column.
	Lines int
	// Code is the width of the code-count column.
	Code int
	// Comments is the width of the comment-count column.
	Comments int
	// Blanks is the width of the blank-count column.
	Blanks int
	// Percent is the width of the code-share column.
	Percent int
}

// Sizes each column to its widest cell across all rows.
func render_widths(rows []render_row) (widths render_column_widths) {
	for _, row := range rows {
		widths.Name = max(widths.Name, len(row.Name))
		widths.Files = max(widths.Files, len(row.Files))
		widths.Lines = max(widths.Lines, len(row.Lines))
		widths.Code = max(widths.Code, len(row.Code))
		widths.Comments = max(widths.Comments, len(row.Comments))
		widths.Blanks = max(widths.Blanks, len(row.Blanks))
		widths.Percent = max(widths.Percent, len(row.Percent))
	}
	return widths
}

// Returns the printed character width of any row, which the rules match. Every cell
// is ASCII, so character width equals byte length.
func render_column_widths_total(widths render_column_widths) (width int) {
	return 1 + widths.Name +
		2 + widths.Files +
		2 + widths.Lines +
		2 + widths.Code +
		2 + widths.Comments +
		2 + widths.Blanks +
		2 + widths.Percent
}

// Lays out one row: a leading space, the left-justified name, then each
// right-justified numeric column behind a two-space gap.
func render_row_format(row render_row, widths render_column_widths) (line string) {
	return " " + pad_right(row.Name, widths.Name) +
		"  " + pad_left(row.Files, widths.Files) +
		"  " + pad_left(row.Lines, widths.Lines) +
		"  " + pad_left(row.Code, widths.Code) +
		"  " + pad_left(row.Comments, widths.Comments) +
		"  " + pad_left(row.Blanks, widths.Blanks) +
		"  " + pad_left(row.Percent, widths.Percent)
}

// Right-justifies text in width by prefixing spaces.
func pad_left(text string, width int) (padded string) {
	return fmt.Sprintf("%*s", width, text)
}

// Left-justifies text in width by suffixing spaces.
func pad_right(text string, width int) (padded string) {
	return fmt.Sprintf("%-*s", width, text)
}

// Renders a non-negative count with a comma between each group of three digits, so
// large totals stay readable.
func with_thousands_separators(value int) (text string) {
	digits := strconv.Itoa(value)
	if len(digits) <= 3 {
		return digits
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
	return builder.String()
}
