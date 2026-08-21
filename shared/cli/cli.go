// Package cli provides a minimal command-line interface parser.
//
// It supports commands with positional arguments and optional flags.
// Arguments must appear before flags in the command line.
// Supported types: string, int, bool.
//
// Example:
//
//	program := cli.New(cli.New_Input{
//		Label:       "myapp",
//		Description: "does something useful",
//		Commands: []cli.Command{{
//			Label:     "add",
//			Arguments: []cli.Option{{Label: "name", Description: "item name"}},
//			Flags:     []cli.Option{{Label: "priority", Value: "low"}},
//		}},
//	})
//	var parser cli.Parser
//	cli.Program_Parse(&program, &parser, cli.Program_Parse_Input{Arguments: os.Args})
//	result, complete := cli.Parser_Done(&parser)
//	if !complete {
//		panic("a program without secrets must complete synchronously")
//	}
//	if result.Error != nil {
//		cli.Print_Help(os.Stderr, program)
//		os.Exit(1)
//	}
//	command := result.Command
//	_ = command
//
// Parse with Program_Parse and render help with Print_Help.
//
// For a binary that does one thing, build the program with New_Single instead of
// New: it has no command selector, so the first token after the program name is the
// first positional argument (as in `sloc ./src`) rather than a command name.
//
// A command's last argument may be variadic (New_Variadic), collecting zero or more
// trailing positionals into a slice, as in `sloc ./a ./b ./c`.
//
// An argument or flag may be an enum (New_Enum_Argument, New_Enum_Flag), restricting its
// value to a fixed set; parsing rejects anything outside it.
package cli

import (
	"errors"
	"path"
	"path/filepath"
	"unsafe"

	"local/james-orcales/shared/diff/levenshtein"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/simulation/aver/default"
	"local/james-orcales/shared/simulation/nbio"
	"local/james-orcales/shared/simulation/time"
	"local/james-orcales/shared/slices"
	"local/james-orcales/shared/strconv"
	"local/james-orcales/shared/strings"
)

// Help_Requested is returned when the command line contains -h or -help.
// Help has priority over an absent argument. Match the error with errors.Is.
// Then call Print_Requested_Help with the returned command.
var Help_Requested = errors.New("help requested")

// Unsupported_Completion_Shell reports a shell with no static registration syntax.
var Unsupported_Completion_Shell = errors.New("unsupported shell; use bash, zsh, or fish")

// HELP_LABEL is the reserved label for the default long help flag.
const HELP_LABEL = "help"

// HELP_SHORT_LABEL is the reserved label for the default short help flag.
const HELP_SHORT_LABEL = "h"

// HELP_FLAG_COUNT derives one injected flag for each nonempty reserved label.
const HELP_FLAG_COUNT = len(HELP_LABEL)/len(HELP_LABEL) +
	len(HELP_SHORT_LABEL)/len(HELP_SHORT_LABEL)

// SECRET_BYTES_MAX is the largest raw secret file that the parser accepts.
const SECRET_BYTES_MAX = 64 * bits.KIBIBYTE_BYTES

// SECRET_BUFFER_BYTES_MAX includes one overflow witness byte.
const SECRET_BUFFER_BYTES_MAX = SECRET_BYTES_MAX + 1

// SECRET_SIZE_MINIMUM accepts an empty secret file before presence policy.
const SECRET_SIZE_MINIMUM int64 = 0

// SECRET_SIZE_MAXIMUM follows accepted secret bytes.
const SECRET_SIZE_MAXIMUM int64 = SECRET_BYTES_MAX

// SECRET_READ_COUNT_MINIMUM admits the first rejected negative completion count.
const SECRET_READ_COUNT_MINIMUM = -1

// SECRET_READ_COUNT_MAXIMUM includes the overflow witness byte.
const SECRET_READ_COUNT_MAXIMUM = SECRET_BUFFER_BYTES_MAX

// DEPRECATION_WARNING_COUNT_MAXIMUM lets every minimum-size row fit fixed output.
const DEPRECATION_WARNING_COUNT_MAXIMUM = strings.TEXT_SIZE_MAXIMUM / len("warning: \n")

// Label is a bounded declaration or token identifier.
type Label string

// Label_Invariants keeps identifiers inside repository text capacity.
func Label_Invariants(value Label, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// OPTION_LABEL_SIZE_MAXIMUM reserves the mandatory named-option dash.
const OPTION_LABEL_SIZE_MAXIMUM = strings.TEXT_SIZE_MAXIMUM - len("-")

// Option_Label is one identifier that remains bounded after dash rendering.
type Option_Label string

// Option_Label_Invariants reserves the named-option prefix inside shared text capacity.
func Option_Label_Invariants(value Option_Label, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), strings.TEXT_SIZE_MINIMUM, OPTION_LABEL_SIZE_MAXIMUM,
		).
		Ensure()
}

// COMPLETION_OPTION_LABEL_SIZE_MAXIMUM reserves the dash and equals syntax.
const COMPLETION_OPTION_LABEL_SIZE_MAXIMUM = strings.TEXT_SIZE_MAXIMUM - len("-=")

// Completion_Option_Label is one possibly empty label split from a completion token.
type Completion_Option_Label string

// Completion_Option_Label_Invariants follows the complete -label= token capacity.
func Completion_Option_Label_Invariants(
	value Completion_Option_Label, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), strings.TEXT_SIZE_MINIMUM,
			COMPLETION_OPTION_LABEL_SIZE_MAXIMUM,
		).
		Ensure()
}

// ENUM_COMPLETION_TOKEN_SIZE_MINIMUM follows the shortest -label= syntax.
const ENUM_COMPLETION_TOKEN_SIZE_MINIMUM = len("-=")

// Enum_Completion_Token is one named completion token known to contain equals.
type Enum_Completion_Token string

// Enum_Completion_Token_Invariants excludes values rejected by the caller grammar.
func Enum_Completion_Token_Invariants(
	value Enum_Completion_Token, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), ENUM_COMPLETION_TOKEN_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM,
		).
		Ensure()
}

// OPTION_COMPLETION_PREFIX_SIZE_MINIMUM follows one leading dash.
const OPTION_COMPLETION_PREFIX_SIZE_MINIMUM = len("-")

// Option_Completion_Prefix is one bounded partial named token.
type Option_Completion_Prefix string

// Option_Completion_Prefix_Invariants keeps the leading dash inside text bounds.
func Option_Completion_Prefix_Invariants(
	value Option_Completion_Prefix, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), OPTION_COMPLETION_PREFIX_SIZE_MINIMUM,
			strings.TEXT_SIZE_MAXIMUM,
		).
		Ensure()
}

// Program_Label is one bounded application identity.
type Program_Label string

// Program_Label_Invariants keeps application identity inside repository text capacity.
func Program_Label_Invariants(value Program_Label, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Description is bounded human-readable declaration help.
type Description string

// Description_Invariants keeps help text inside repository text capacity.
func Description_Invariants(value Description, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// EXTERNAL_TYPE_TEXT_SIZE_MINIMUM follows the shortest admitted scalar type name.
const EXTERNAL_TYPE_TEXT_SIZE_MINIMUM = len("int")

// External_Type_Text is one rendered scalar type with an optional enum signature.
type External_Type_Text string

// External_Type_Text_Invariants excludes lengths no admitted Go scalar can render.
func External_Type_Text_Invariants(
	value External_Type_Text, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), EXTERNAL_TYPE_TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM,
		).
		Ensure()
}

// OPTION_SIGNATURE_SIZE_MINIMUM follows an empty label and the shortest scalar name.
const OPTION_SIGNATURE_SIZE_MINIMUM = len("<") + len(": ") + len("int") + len(">")

// Program_Description is one bounded application summary.
type Program_Description string

// Program_Description_Invariants keeps application help inside repository text capacity.
func Program_Description_Invariants(value Program_Description, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Deprecation is bounded migration guidance.
type Deprecation string

// Deprecation_Invariants keeps migration guidance inside repository text capacity.
func Deprecation_Invariants(value Deprecation, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Value_Text is bounded parsed scalar text.
type Value_Text string

// Value_Text_Invariants keeps scalar text inside repository text capacity.
func Value_Text_Invariants(value Value_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// ENVIRONMENT_WARNING_PREFIX names the deprecated environment declaration.
const ENVIRONMENT_WARNING_PREFIX = "environment variable "

// ENVIRONMENT_WARNING_SUFFIX separates the key from migration guidance.
const ENVIRONMENT_WARNING_SUFFIX = " is deprecated: "

// DEPRECATION_SIZE_MINIMUM follows one admitted guidance byte.
const DEPRECATION_SIZE_MINIMUM = len("d")

// ENVIRONMENT_WARNING_SIZE_MINIMUM follows the shortest key and guidance.
const ENVIRONMENT_WARNING_SIZE_MINIMUM = len(ENVIRONMENT_WARNING_PREFIX) +
	EXTERNAL_KEY_SIZE_MINIMUM + len(ENVIRONMENT_WARNING_SUFFIX) + DEPRECATION_SIZE_MINIMUM

// ENVIRONMENT_WARNING_ABSENT keeps the non-emitted result inside its scalar type.
const ENVIRONMENT_WARNING_ABSENT = ENVIRONMENT_WARNING_PREFIX + "E" +
	ENVIRONMENT_WARNING_SUFFIX + "d"

// Environment_Warning_Text is one nonempty bounded deprecation diagnostic.
type Environment_Warning_Text string

// Environment_Warning_Text_Invariants excludes the absent-warning sentinel.
func Environment_Warning_Text_Invariants(
	value Environment_Warning_Text, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), ENVIRONMENT_WARNING_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM,
		).
		Ensure()
}

// External_Value_Text admits the complete bounded secret file payload.
type External_Value_Text string

// External_Value_Text_Invariants follows the secret parser's accepted byte maximum.
func External_Value_Text_Invariants(value External_Value_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, SECRET_BYTES_MAX).
		Ensure()
}

// ENVIRONMENT_VALUE_SIZE_MAXIMUM reserves the shortest valid key and separator.
const ENVIRONMENT_VALUE_SIZE_MAXIMUM = strings.TEXT_SIZE_MAXIMUM - len("E=")

// Environment_Value_Text is the largest value inside one bounded KEY=value entry.
type Environment_Value_Text string

// Environment_Value_Text_Invariants reserves environment grammar inside text capacity.
func Environment_Value_Text_Invariants(
	value Environment_Value_Text, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), strings.TEXT_SIZE_MINIMUM, ENVIRONMENT_VALUE_SIZE_MAXIMUM,
		).
		Ensure()
}

// NAMED_VALUE_SIZE_MAXIMUM reserves the shortest complete -x=value prefix.
const NAMED_VALUE_SIZE_MAXIMUM = strings.TEXT_SIZE_MAXIMUM - len("-x=")

// NAMED_TOKEN_SIZE_MINIMUM follows the shortest admitted named token.
const NAMED_TOKEN_SIZE_MINIMUM = len("-x")

// Named_Token is one token already proven to be named rather than positional.
type Named_Token string

// Named_Token_Invariants excludes the lone dash and all shorter text.
func Named_Token_Invariants(value Named_Token, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), NAMED_TOKEN_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Assigned_Value is one named option value after token splitting.
type Assigned_Value string

// Assigned_Value_Invariants bounds named option conversion work.
func Assigned_Value_Invariants(value Assigned_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), strings.TEXT_SIZE_MINIMUM, NAMED_VALUE_SIZE_MAXIMUM,
		).
		Ensure()
}

// NAMED_STRING_VALUE_SIZE_MINIMUM follows the first required non-Boolean value byte.
const NAMED_STRING_VALUE_SIZE_MINIMUM = len("x")

// Named_String_Value is one nonempty named string after assignment validation.
type Named_String_Value string

// Named_String_Value_Invariants excludes the missing-value error path.
func Named_String_Value_Invariants(
	value Named_String_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), NAMED_STRING_VALUE_SIZE_MINIMUM, NAMED_VALUE_SIZE_MAXIMUM,
		).
		Ensure()
}

// Trimmed_Value is one named string after optional surrounding quote removal.
type Trimmed_Value string

// Trimmed_Value_Invariants preserves the bounded value after quote removal.
func Trimmed_Value_Invariants(value Trimmed_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, NAMED_VALUE_SIZE_MAXIMUM).
		Ensure()
}

// String_Enum_Default keeps a text default type-aligned with its permitted set.
type String_Enum_Default string

// String_Enum_Default_Invariants bounds text-default membership work.
func String_Enum_Default_Invariants(value String_Enum_Default, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Integer_Enum_Default keeps an integer default type-aligned with its permitted set.
type Integer_Enum_Default int

// Integer_Enum_Default_Invariants admits the full declared integer domain.
func Integer_Enum_Default_Invariants(value Integer_Enum_Default, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bits.INTEGER_MINIMUM, bits.INTEGER_MAXIMUM).
		Ensure()
}

// OPTION_NAME_SIZE_MINIMUM follows the first required label byte.
const OPTION_NAME_SIZE_MINIMUM = len("x")

// Option_Name is one resolved option identity retained for assignment diagnostics.
type Option_Name string

// Option_Name_Invariants bounds assignment diagnostic work.
func Option_Name_Invariants(value Option_Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), OPTION_NAME_SIZE_MINIMUM, OPTION_LABEL_SIZE_MAXIMUM,
		).
		Ensure()
}

// EXTERNAL_KEY_SIZE_MAXIMUM reserves the environment separator or absolute-path slash.
const EXTERNAL_KEY_SIZE_MAXIMUM = strings.TEXT_SIZE_MAXIMUM - len("=")

// External_Key is one bounded environment or secret identity.
type External_Key string

// External_Key_Invariants bounds identity validation work.
func External_Key_Invariants(value External_Key, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), strings.TEXT_SIZE_MINIMUM, EXTERNAL_KEY_SIZE_MAXIMUM,
		).
		Ensure()
}

// EXTERNAL_KEY_SIZE_MINIMUM follows the first required key byte.
const EXTERNAL_KEY_SIZE_MINIMUM = len("E")

// Resolved_External_Key is one validated nonempty declaration identity.
type Resolved_External_Key string

// Resolved_External_Key_Invariants excludes constructor-only invalid empty input.
func Resolved_External_Key_Invariants(
	value Resolved_External_Key, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), EXTERNAL_KEY_SIZE_MINIMUM, EXTERNAL_KEY_SIZE_MAXIMUM,
		).
		Ensure()
}

// CONVERTED_EXTERNAL_KEY_SIZE_MAXIMUM reserves one separator and scalar byte.
const CONVERTED_EXTERNAL_KEY_SIZE_MAXIMUM = strings.TEXT_SIZE_MAXIMUM - len("=0")

// Converted_External_Key names a source whose nonempty scalar reached enum validation.
type Converted_External_Key string

// Converted_External_Key_Invariants reserves the required scalar byte.
func Converted_External_Key_Invariants(
	value Converted_External_Key, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), EXTERNAL_KEY_SIZE_MINIMUM, CONVERTED_EXTERNAL_KEY_SIZE_MAXIMUM,
		).
		Ensure()
}

// SECRET_KEY_SIZE_MAXIMUM reserves the shortest absolute-path prefix.
const SECRET_KEY_SIZE_MAXIMUM = strings.TEXT_SIZE_MAXIMUM - len("/")

// Secret_Key is one basename carried by an absolute secret path.
type Secret_Key string

// Secret_Key_Invariants reserves one byte for absolute-path identity.
func Secret_Key_Invariants(value Secret_Key, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), strings.TEXT_SIZE_MINIMUM, SECRET_KEY_SIZE_MAXIMUM,
		).
		Ensure()
}

// Shell is one bounded completion target syntax name.
type Shell string

// Shell_Invariants bounds shell selection work.
func Shell_Invariants(value Shell, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// COMPLETION_BASH_FORMAT is one bash registration block.
const COMPLETION_BASH_FORMAT = "" +
	"_%[1]s_complete() {\n" +
	"    local IFS=$'\\n'\n" +
	"    COMPREPLY=($(%[1]s __complete \"${COMP_WORDS[@]}\"))\n" +
	"}\n" +
	"complete -o default -F _%[1]s_complete %[1]s\n"

// COMPLETION_ZSH_FORMAT is one zsh registration block.
const COMPLETION_ZSH_FORMAT = "" +
	"#compdef %[1]s\n" +
	"_%[1]s_complete() {\n" +
	"    local -a completions\n" +
	"    completions=(\"${(@f)$(%[1]s __complete \"${words[@]}\")}\")\n" +
	"    compadd -- $completions\n" +
	"}\n" +
	"compdef _%[1]s_complete %[1]s\n"

// COMPLETION_FISH_FORMAT is one fish registration block.
const COMPLETION_FISH_FORMAT = "" +
	"function _%[1]s_complete\n" +
	"    %[1]s __complete (commandline -opc) (commandline -ct)\n" +
	"end\n" +
	"complete -c %[1]s -f -a '(_%[1]s_complete)'\n"

// COMPLETION_FORMAT_DIRECTIVE is replaced by one borrowed command label.
const COMPLETION_FORMAT_DIRECTIVE = "%[1]s"

// COMPLETION_FORMAT_DIRECTIVE_SIZE is the replaced name directive width.
const COMPLETION_FORMAT_DIRECTIVE_SIZE = len(COMPLETION_FORMAT_DIRECTIVE)

// COMPLETION_FISH_NAME_COUNT follows the fish block name occurrences.
const COMPLETION_FISH_NAME_COUNT = len("namenamenamename") / len("name")

// COMPLETION_ZSH_NAME_COUNT follows the zsh block name occurrences.
const COMPLETION_ZSH_NAME_COUNT = len("namenamenamenamename") / len("name")

// COMPLETION_ZSH_SIZE_MINIMUM removes the empty name directives.
const COMPLETION_ZSH_SIZE_MINIMUM = len(COMPLETION_ZSH_FORMAT) -
	COMPLETION_ZSH_NAME_COUNT*COMPLETION_FORMAT_DIRECTIVE_SIZE

// COMPLETION_ZSH_LABEL_CAPACITY is the text left after the fixed zsh block.
const COMPLETION_ZSH_LABEL_CAPACITY = strings.TEXT_SIZE_MAXIMUM -
	COMPLETION_ZSH_SIZE_MINIMUM

// COMPLETION_ZSH_MAXIMUM_LABEL_SIZE fills one zsh block to the shared text bound.
const COMPLETION_ZSH_MAXIMUM_LABEL_SIZE = COMPLETION_ZSH_LABEL_CAPACITY /
	COMPLETION_ZSH_NAME_COUNT

// SCRIPT_SIZE_MINIMUM removes the four empty-name directives from the shortest block.
const SCRIPT_SIZE_MINIMUM = len(COMPLETION_FISH_FORMAT) -
	COMPLETION_FISH_NAME_COUNT*COMPLETION_FORMAT_DIRECTIVE_SIZE

// Script is nonempty bounded successful completion script text.
type Script string

// Script_Invariants bounds returned completion text.
func Script_Invariants(value Script, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SCRIPT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Message is bounded diagnostic text.
type Message string

// Message_Invariants bounds diagnostic construction work.
func Message_Invariants(value Message, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// DISPLAY_OPTION_NAME_COUNT isolates diagnostic bounds from declaration bounds.
const DISPLAY_OPTION_NAME_COUNT = len("name") / len("name")

// Display_Option_Name is one option identity held for diagnostic rendering.
type Display_Option_Name [DISPLAY_OPTION_NAME_COUNT]Option_Name

// Display_Option_Name_Invariants admits both argument and flag label bounds.
func Display_Option_Name_Invariants(
	value Display_Option_Name, _ aver.Namespace,
) {
	aver.Always(
		len(value[0]) >= OPTION_NAME_SIZE_MINIMUM,
		"Display name retains one identity byte.",
	)
	aver.Always(
		len(value[0]) <= OPTION_LABEL_SIZE_MAXIMUM,
		"Display name stays inside option identity capacity.",
	)
}

// Display_Name keeps option identity and optional dash separate until rendering.
type Display_Name struct {
	// Name is declaration identity without syntax.
	Name Display_Option_Name
	// Dashed reports named-option grammar.
	Dashed Boolean
}

// Display_Name_Invariants bounds diagnostic option rendering.
func Display_Name_Invariants(value Display_Name, namespace aver.Namespace) {
	Display_Option_Name_Invariants(value.Name, namespace)
	Boolean_Invariants(value.Dashed, namespace)
}

// Separator selects one enum rendering punctuation.
type Separator byte

// Separator_Invariants admits the two enum punctuation forms.
func Separator_Invariants(value Separator, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(SEPARATOR_PIPE), uint8(SEPARATOR_COMMA)).
		Ensure()
}

// SEPARATOR_PIPE joins compact enum signatures.
const SEPARATOR_PIPE Separator = 0

// SEPARATOR_COMMA joins diagnostic enum lists.
const SEPARATOR_COMMA Separator = SEPARATOR_PIPE + 1

// Section_Title is bounded help heading text.
type Section_Title string

// Section_Title_Invariants bounds help rendering work.
func Section_Title_Invariants(value Section_Title, _ aver.Namespace) {
	aver.Always(value == FLAG_SECTION_TITLE, "Flag section title is fixed syntax.")
	aver.Always(
		len(value) == FLAG_SECTION_TITLE_SIZE,
		"Flag section title has fixed rendering size.",
	)
}

// FLAG_SECTION_TITLE is the only private option-section heading.
const FLAG_SECTION_TITLE Section_Title = "Flags:"

// FLAG_SECTION_TITLE_SIZE follows the only admitted title syntax.
const FLAG_SECTION_TITLE_SIZE = len(FLAG_SECTION_TITLE)

// Indent selects one help row nesting level.
type Indent byte

// Indent_Invariants bounds help rendering work.
func Indent_Invariants(value Indent, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(INDENT_FLAG), uint8(INDENT_COMMAND_FLAG)).
		Ensure()
}

// INDENT_FLAG is one ordinary flag row.
const INDENT_FLAG Indent = 0

// INDENT_COMMAND_FLAG is one flag nested under a command catalog row.
const INDENT_COMMAND_FLAG Indent = INDENT_FLAG + 1

// Source_Name selects the declaration namespace used in diagnostics.
type Source_Name byte

// Source_Name_Invariants bounds diagnostic rendering work.
func Source_Name_Invariants(value Source_Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(SOURCE_ENVIRONMENT), uint8(SOURCE_SECRET)).
		Ensure()
}

// SOURCE_ENVIRONMENT names process environment declarations.
const SOURCE_ENVIRONMENT Source_Name = 0

// SOURCE_SECRET names file-backed declarations.
const SOURCE_SECRET Source_Name = SOURCE_ENVIRONMENT + 1

// Expected_Type selects one external scalar conversion diagnostic.
type Expected_Type byte

// Expected_Type_Invariants admits the two external scalar kinds.
func Expected_Type_Invariants(value Expected_Type, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(EXPECTED_INTEGER), uint8(EXPECTED_BOOLEAN)).
		Ensure()
}

// EXPECTED_INTEGER names decimal machine-integer input.
const EXPECTED_INTEGER Expected_Type = 0

// EXPECTED_BOOLEAN names accepted Boolean input.
const EXPECTED_BOOLEAN Expected_Type = EXPECTED_INTEGER + 1

// CHARACTER_MINIMUM is Unicode's first valid code point.
const CHARACTER_MINIMUM int32 = 0

// CHARACTER_MAXIMUM is Unicode's last valid code point.
const CHARACTER_MAXIMUM int32 = '\U0010FFFF'

// Character is one external key code point.
type Character rune

// Character_Invariants admits every code point range iteration can produce.
func Character_Invariants(value Character, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), CHARACTER_MINIMUM, CHARACTER_MAXIMUM).
		Ensure()
}

// Boolean is one CLI query result.
type Boolean bool

// Boolean_Invariants requires both query results.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The CLI query result is true.").
		Ensure()
}

// Equals_Present records whether a named token carried '='.
type Equals_Present bool

// Equals_Present_Invariants requires bare and explicit named tokens.
func Equals_Present_Invariants(value Equals_Present, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The named token carries an equals sign.").
		Ensure()
}

// Hidden controls declaration visibility.
type Hidden bool

// Hidden_Invariants requires visible and hidden declarations.
func Hidden_Invariants(value Hidden, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The declaration is hidden.").
		Ensure()
}

// Required controls external source absence.
type Required bool

// Required_Invariants requires optional and required declarations.
func Required_Invariants(value Required, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The external declaration is required.").
		Ensure()
}

// Allow_Empty controls whether an empty external source is present.
type Allow_Empty bool

// Allow_Empty_Invariants requires both empty-source policies.
func Allow_Empty_Invariants(value Allow_Empty, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The external declaration admits empty content.").
		Ensure()
}

// Parsed distinguishes resolved option storage from definition defaults.
type Parsed bool

// Parsed_Invariants requires definition and resolved option states.
func Parsed_Invariants(value Parsed, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The option holds a parsed value.").
		Ensure()
}

// Parser_Complete reports terminal parser publication.
type Parser_Complete bool

// Parser_Complete_Invariants requires pending and terminal parser states.
func Parser_Complete_Invariants(value Parser_Complete, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The parser published its terminal result.").
		Ensure()
}

// Is_Flag distinguishes named optional flags from positional arguments.
type Is_Flag bool

// Is_Flag_Invariants requires flag and positional declarations.
func Is_Flag_Invariants(value Is_Flag, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The option is a flag.").
		Ensure()
}

// Option_Type identifies one fixed scalar or collection representation.
type Option_Type byte

// Option_Type_Invariants bounds the closed option representation set.
func Option_Type_Invariants(value Option_Type, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(OPTION_TYPE_STRING), uint8(OPTION_TYPE_INTEGERS),
		).
		Ensure()
}

// OPTION_TYPE_STRING selects one text scalar.
const OPTION_TYPE_STRING Option_Type = 0

// OPTION_TYPE_INTEGER selects one machine integer scalar.
const OPTION_TYPE_INTEGER = OPTION_TYPE_STRING + 1

// OPTION_TYPE_BOOLEAN selects one Boolean scalar.
const OPTION_TYPE_BOOLEAN = OPTION_TYPE_INTEGER + 1

// OPTION_TYPE_STRINGS selects one text collection.
const OPTION_TYPE_STRINGS = OPTION_TYPE_BOOLEAN + 1

// OPTION_TYPE_INTEGERS selects one machine integer collection.
const OPTION_TYPE_INTEGERS = OPTION_TYPE_STRINGS + 1

// OPTION_TYPE_STATE_COUNT follows one fixed union discriminant cell.
const OPTION_TYPE_STATE_COUNT = len("type") / len("type")

// Option_Type_State keeps one discriminant without phase-wide branch obligations.
type Option_Type_State [OPTION_TYPE_STATE_COUNT]Option_Type

// Option_Type_State_Invariants bounds one intrinsically fixed union cell.
func Option_Type_State_Invariants(value Option_Type_State, _ aver.Namespace) {
	aver.Always(
		value[0] >= OPTION_TYPE_STRING,
		"Option state type does not precede the representation set.",
	)
	aver.Always(
		value[0] <= OPTION_TYPE_INTEGERS,
		"Option state type stays inside the representation set.",
	)
}

// Scalar_Option_Type excludes both collection representations.
type Scalar_Option_Type byte

// Scalar_Option_Type_Invariants bounds scalar constructor and rendering work.
func Scalar_Option_Type_Invariants(
	value Scalar_Option_Type, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(OPTION_TYPE_STRING), uint8(OPTION_TYPE_INTEGER),
			uint8(OPTION_TYPE_BOOLEAN),
		).
		Ensure()
}

// Option_Location identifies one option collection without returning a nullable pointer.
type Option_Location byte

// Option_Location_Invariants admits every option collection and the failed-search result.
func Option_Location_Invariants(value Option_Location, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(OPTION_LOCATION_ABSENT),
			uint8(OPTION_LOCATION_ARGUMENT), uint8(OPTION_LOCATION_COMMAND_FLAG),
			uint8(OPTION_LOCATION_GLOBAL_FLAG),
		).
		Ensure()
}

// OPTION_LOCATION_ABSENT is a failed option search.
const OPTION_LOCATION_ABSENT Option_Location = 0

// OPTION_LOCATION_ARGUMENT selects command arguments.
const OPTION_LOCATION_ARGUMENT = OPTION_LOCATION_ABSENT + 1

// OPTION_LOCATION_COMMAND_FLAG selects command flags.
const OPTION_LOCATION_COMMAND_FLAG = OPTION_LOCATION_ARGUMENT + 1

// OPTION_LOCATION_GLOBAL_FLAG selects program flags.
const OPTION_LOCATION_GLOBAL_FLAG = OPTION_LOCATION_COMMAND_FLAG + 1

// Resolved_Option_Location excludes the failed-search result after lookup succeeds.
type Resolved_Option_Location byte

// Resolved_Option_Location_Invariants admits the three assignable collections.
func Resolved_Option_Location_Invariants(
	value Resolved_Option_Location, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(OPTION_LOCATION_ARGUMENT),
			uint8(OPTION_LOCATION_COMMAND_FLAG), uint8(OPTION_LOCATION_GLOBAL_FLAG),
		).
		Ensure()
}

// Integer is one machine-sized parsed CLI value.
type Integer int

// Integer_Invariants admits the complete declared integer domain.
func Integer_Invariants(value Integer, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), bits.INTEGER_MINIMUM, bits.INTEGER_MAXIMUM).
		Ensure()
}

// Token_Index locates one token inside a bounded process argument snapshot.
type Token_Index int

// Token_Index_Invariants excludes absent and after-final positions.
func Token_Index_Invariants(value Token_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), slices.POSITION_MINIMUM, slices.FOUND_INDEX_MAXIMUM).
		Ensure()
}

// Option_Index locates one present option or command inside bounded storage.
type Option_Index int

// Option_Index_Invariants excludes the absent-search sentinel.
func Option_Index_Invariants(value Option_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), slices.POSITION_MINIMUM, slices.FOUND_INDEX_MAXIMUM).
		Ensure()
}

// Secret_Index locates one declaration inside bounded secret storage.
type Secret_Index int

// Secret_Index_Invariants bounds secret result indexing.
func Secret_Index_Invariants(value Secret_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), slices.POSITION_MINIMUM, slices.FOUND_INDEX_MAXIMUM).
		Ensure()
}

// Secret_Path_Index locates one ordered fallback path.
type Secret_Path_Index int

// Secret_Path_Index_Invariants bounds fallback indexing.
func Secret_Path_Index_Invariants(value Secret_Path_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), slices.POSITION_MINIMUM, slices.FOUND_INDEX_MAXIMUM).
		Ensure()
}

// Only_Absent reports whether every retired path was absent.
type Only_Absent bool

// Only_Absent_Invariants requires absence-only and operational failure paths.
func Only_Absent_Invariants(value Only_Absent, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Every retired secret path was absent.").
		Ensure()
}

// Current_Absent classifies one current path failure as absence.
type Current_Absent bool

// Current_Absent_Invariants requires absent and operational current path failures.
func Current_Absent_Invariants(value Current_Absent, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The current secret path is absent.").
		Ensure()
}

// Declaration_Count is a bounded external or option count.
type Declaration_Count int

// Declaration_Count_Invariants bounds outstanding declaration work.
func Declaration_Count_Invariants(value Declaration_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Occurrence_Count counts entries for one key in one bounded environment snapshot.
type Occurrence_Count int

// Occurrence_Count_Invariants bounds duplicate classification work.
func Occurrence_Count_Invariants(value Occurrence_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Secret_Size is a validated file length admitted by secret parsing.
type Secret_Size int64

// Secret_Size_Invariants binds allocation-free read storage to accepted bytes.
func Secret_Size_Invariants(value Secret_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), SECRET_SIZE_MINIMUM, SECRET_SIZE_MAXIMUM).
		Ensure()
}

// Secret_Read_Count admits accepted reads and first rejected boundaries.
type Secret_Read_Count int

// Secret_Read_Count_Invariants bounds hostile completion data.
func Secret_Read_Count_Invariants(value Secret_Read_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), SECRET_READ_COUNT_MINIMUM, SECRET_READ_COUNT_MAXIMUM).
		Ensure()
}

// Command_Declarations admits empty untrusted constructor input before validation.
type Command_Declarations []Command

// Command_Declarations_Invariants bounds constructor work before semantic validation.
func Command_Declarations_Invariants(
	value Command_Declarations, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// COMMAND_COUNT_MINIMUM follows the first selectable command.
const COMMAND_COUNT_MINIMUM = len("command") / len("command")

// Commands is a nonempty bounded borrowed command collection.
type Commands []Command

// Commands_Invariants binds active selection to at least one command.
func Commands_Invariants(value Commands, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COMMAND_COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Options is a bounded borrowed option collection.
type Options []Option

// Options_Invariants bounds option search and validation work.
func Options_Invariants(value Options, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Option_Collection admits each semantic option slice without copying its backing storage.
type Option_Collection interface {
	~[]Option
}

// OPTION_REFERENCE_COUNT follows one borrowed mutable option.
const OPTION_REFERENCE_COUNT = len("option") / len("option")

// Option_Reference stores one stable pointer into caller-owned parse storage.
type Option_Reference [OPTION_REFERENCE_COUNT]*Option

// Option_Reference_Invariants fixes one non-nil borrowed option reference.
func Option_Reference_Invariants(value Option_Reference, _ aver.Namespace) {
	aver.Always(value[0] != nil, "Assignment retains one caller-owned option reference.")
}

// Output keeps bounded formatting storage in caller ownership.
type Output struct {
	// Builder fixes output capacity without hiding a growing allocation.
	Builder strings.Builder
}

// Output_Invariants composes fixed caller formatting storage.
func Output_Invariants(value Output, namespace aver.Namespace) {
	strings.Builder_Invariants(value.Builder, namespace)
}

// OUTPUT_REFERENCE_COUNT follows one borrowed caller-owned output.
const OUTPUT_REFERENCE_COUNT = len("output") / len("output")

// Output_Reference stores one stable pointer to caller-owned formatting state.
type Output_Reference [OUTPUT_REFERENCE_COUNT]*Output

// Output_Reference_Invariants fixes one non-nil borrowed output reference.
func Output_Reference_Invariants(value Output_Reference, _ aver.Namespace) {
	aver.Always(value[0] != nil, "Rendering retains one caller-owned output reference.")
}

// Write satisfies standard formatting while preserving fixed builder capacity.
func (output *Output) Write(source []byte) (count int, err error) {
	Output_Invariants(*output, "output_write.output")
	written := strings.Builder_Write(&output.Builder, strings.Bytes(source))
	return int(written), nil
}

// Output_Bytes exposes initialized output without copying it into a string.
func Output_Bytes(output *Output) (content strings.Bytes) {
	defer func() { strings.Bytes_Invariants(content, "output_bytes.content") }()
	Output_Invariants(*output, "output_bytes.output")
	return strings.Builder_Bytes(&output.Builder)
}

// Output_Reset preserves storage while removing initialized output.
func Output_Reset(output *Output) {
	Output_Invariants(*output, "output_reset.output")
	strings.Builder_Reset(&output.Builder)
}

// Output_Write_Text avoids formatting machinery for already-rendered text.
func Output_Write_Text(output *Output, source Value_Text) {
	Output_Invariants(*output, "output_write_text.output")
	Value_Text_Invariants(source, "output_write_text.source")
	strings.Builder_Write_Text(&output.Builder, strings.Text(source))
}

// FAILURE_SIZE_MAXIMUM follows widest CLI diagnostic: enum value, match, and name.
const FAILURE_SIZE_MAXIMUM = len("invalid value  for -, did you mean ?") +
	2*strconv.QUOTED_TEXT_SIZE_MAXIMUM + OPTION_LABEL_SIZE_MAXIMUM

// Failure_Storage owns bounded diagnostic bytes.
type Failure_Storage [FAILURE_SIZE_MAXIMUM]byte

// Failure_Storage_Invariants fixes compile-time diagnostic capacity.
func Failure_Storage_Invariants(value Failure_Storage, _ aver.Namespace) {
	aver.Always(
		len(value) == FAILURE_SIZE_MAXIMUM,
		"Failure storage has exact widest CLI diagnostic capacity.",
	)
}

// FAILURE_SIZE_COUNT follows one initialized diagnostic byte count.
const FAILURE_SIZE_COUNT = len("size") / len("size")

// Failure_Size selects initialized diagnostic bytes.
type Failure_Size [FAILURE_SIZE_COUNT]int

// Failure_Size_Invariants bounds initialized diagnostic bytes.
func Failure_Size_Invariants(value Failure_Size, _ aver.Namespace) {
	aver.Always(
		value[0] >= strings.TEXT_SIZE_MINIMUM,
		"Failure size cannot be negative.",
	)
	aver.Always(
		value[0] <= FAILURE_SIZE_MAXIMUM,
		"Failure size stays inside fixed diagnostic storage.",
	)
}

// FAILURE_FRAGMENT_COUNT follows one borrowed diagnostic part.
const FAILURE_FRAGMENT_COUNT = len("fragment") / len("fragment")

// Failure_Fragment is one bounded borrowed diagnostic part.
type Failure_Fragment [FAILURE_FRAGMENT_COUNT]string

// Failure_Fragment_Invariants checks only write-time storage safety.
func Failure_Fragment_Invariants(value Failure_Fragment, _ aver.Namespace) {
	aver.Always(
		len(value[0]) <= strings.TEXT_SIZE_MAXIMUM,
		"Failure fragment stays inside shared text capacity.",
	)
}

// FAILURE_SECRET_KEY_COUNT isolates rendered identity from declaration phases.
const FAILURE_SECRET_KEY_COUNT = len("key") / len("key")

// Failure_Secret_Key is one validated secret identity rendered in a diagnostic.
type Failure_Secret_Key [FAILURE_SECRET_KEY_COUNT]Secret_Key

// Failure_Secret_Key_Invariants follows keys produced from valid absolute paths.
func Failure_Secret_Key_Invariants(value Failure_Secret_Key, _ aver.Namespace) {
	aver.Always(
		len(value[0]) >= EXTERNAL_KEY_SIZE_MINIMUM,
		"Rendered secret failure retains one key byte.",
	)
	aver.Always(
		len(value[0]) <= SECRET_KEY_SIZE_MAXIMUM,
		"Rendered secret failure key stays inside path capacity.",
	)
}

// FAILURE_BYTES_COUNT follows one rendered scalar fragment.
const FAILURE_BYTES_COUNT = len("bytes") / len("bytes")

// Failure_Bytes is one bounded rendered scalar fragment.
type Failure_Bytes [FAILURE_BYTES_COUNT][]byte

// Failure_Bytes_Invariants bounds one copied conversion result.
func Failure_Bytes_Invariants(value Failure_Bytes, _ aver.Namespace) {
	aver.Always(
		len(value[0]) <= strconv.QUOTED_TEXT_SIZE_MAXIMUM,
		"Failure bytes stay inside widest scalar rendering.",
	)
}

// FAILURE_INTEGER_COUNT follows one scalar diagnostic cell.
const FAILURE_INTEGER_COUNT = len("integer") / len("integer")

// Failure_Integer is one scalar rendered inside a diagnostic.
type Failure_Integer [FAILURE_INTEGER_COUNT]int

// Failure_Integer_Invariants admits machine integer diagnostic input.
func Failure_Integer_Invariants(value Failure_Integer, _ aver.Namespace) {
	aver.Always(
		value[0] >= bits.INTEGER_MINIMUM,
		"Failure integer admits machine minimum.",
	)
	aver.Always(
		value[0] <= bits.INTEGER_MAXIMUM,
		"Failure integer admits machine maximum.",
	)
}

// FAILURE_STRINGS_COUNT follows one permitted text slice.
const FAILURE_STRINGS_COUNT = len("strings") / len("strings")

// Failure_Strings is one permitted text set rendered into a diagnostic.
type Failure_Strings [FAILURE_STRINGS_COUNT][]string

// Failure_Strings_Invariants bounds permitted diagnostic work.
func Failure_Strings_Invariants(value Failure_Strings, _ aver.Namespace) {
	aver.Always(
		len(value[0]) <= slices.COUNT_MAXIMUM,
		"Failure text set stays inside shared collection capacity.",
	)
}

// FAILURE_INTEGERS_COUNT follows one permitted integer slice.
const FAILURE_INTEGERS_COUNT = len("integers") / len("integers")

// Failure_Integers is one permitted integer set rendered into a diagnostic.
type Failure_Integers [FAILURE_INTEGERS_COUNT][]int

// Failure_Integers_Invariants bounds permitted diagnostic work.
func Failure_Integers_Invariants(value Failure_Integers, _ aver.Namespace) {
	aver.Always(
		len(value[0]) <= slices.COUNT_MAXIMUM,
		"Failure integer set stays inside shared collection capacity.",
	)
}

// Failure keeps one bounded diagnostic inside parser ownership.
type Failure struct {
	// Storage prevents error-interface publication from owning rendered text.
	Storage Failure_Storage
	// Size selects initialized storage prefix.
	Size Failure_Size
}

// Failure_Invariants composes fixed diagnostic storage.
func Failure_Invariants(value Failure, namespace aver.Namespace) {
	Failure_Storage_Invariants(value.Storage, namespace)
	Failure_Size_Invariants(value.Size, namespace)
}

// Error exposes initialized diagnostic bytes without copying them.
func (failure *Failure) Error() (message string) {
	Failure_Invariants(*failure, "failure_error.failure")
	return unsafe.String(&failure.Storage[0], failure.Size[0])
}

// FAILURE_REFERENCE_COUNT follows one parser-owned diagnostic.
const FAILURE_REFERENCE_COUNT = len("failure") / len("failure")

// Failure_Reference keeps one stable parser diagnostic pointer.
type Failure_Reference [FAILURE_REFERENCE_COUNT]*Failure

// Failure_Reference_Invariants fixes one non-nil diagnostic reference.
func Failure_Reference_Invariants(value Failure_Reference, _ aver.Namespace) {
	aver.Always(value[0] != nil, "Failure reference retains parser storage.")
}

// Optional_Failure_Reference permits zero Parser state before initialization.
type Optional_Failure_Reference [FAILURE_REFERENCE_COUNT]*Failure

// Optional_Failure_Reference_Invariants observes both parser lifecycle phases.
func Optional_Failure_Reference_Invariants(
	value Optional_Failure_Reference, _ aver.Namespace,
) {
	aver.Sometimes(value[0] != nil, "Parser workspace has initialized failure storage.")
}

func failure_reset(reference Failure_Reference) {
	Failure_Reference_Invariants(reference, "failure_reset.reference")
	failure := reference[0]
	Failure_Invariants(*failure, "failure_reset.failure")
	failure.Size[0] = 0
}

func failure_write(reference Failure_Reference, text Failure_Fragment) {
	Failure_Reference_Invariants(reference, "failure_write.reference")
	Failure_Fragment_Invariants(text, "failure_write.text")
	failure := reference[0]
	Failure_Invariants(*failure, "failure_write.failure")
	start := failure.Size[0]
	end := start + len(text[0])
	if end > len(failure.Storage) {
		panic("CLI failure exceeds fixed diagnostic storage.")
	}
	copy(failure.Storage[start:end], text[0])
	failure.Size[0] = end
}

func failure_write_bytes(reference Failure_Reference, source Failure_Bytes) {
	Failure_Reference_Invariants(reference, "failure_write_bytes.reference")
	Failure_Bytes_Invariants(source, "failure_write_bytes.source")
	failure := reference[0]
	Failure_Invariants(*failure, "failure_write_bytes.failure")
	start := failure.Size[0]
	end := start + len(source[0])
	if end > len(failure.Storage) {
		panic("CLI failure exceeds fixed diagnostic storage.")
	}
	copy(failure.Storage[start:end], source[0])
	failure.Size[0] = end
}

func failure_write_quoted(reference Failure_Reference, text Failure_Fragment) {
	Failure_Reference_Invariants(reference, "failure_write_quoted.reference")
	Failure_Fragment_Invariants(text, "failure_write_quoted.text")
	failure := reference[0]
	Failure_Invariants(*failure, "failure_write_quoted.failure")
	start := failure.Size[0]
	end := start + strconv.QUOTED_TEXT_SIZE_MAXIMUM
	if end > len(failure.Storage) {
		panic("CLI failure exceeds fixed quoted diagnostic storage.")
	}
	count := strconv.Quote_Into(
		strconv.Buffer(failure.Storage[start:end]), strconv.Text(text[0]),
	)
	failure.Size[0] = start + int(count)
}

func failure_write_decimal(reference Failure_Reference, value Failure_Integer) {
	Failure_Reference_Invariants(reference, "failure_write_decimal.reference")
	Failure_Integer_Invariants(value, "failure_write_decimal.value")
	var storage [strconv.DECIMAL_TEXT_SIZE_MAXIMUM]byte
	count := strconv.Format_Decimal_Into(
		storage[:], strconv.Machine_Integer(value[0]),
	)
	failure_write_bytes(reference, Failure_Bytes{storage[:count]})
}

func failure_expected_integer(
	failure Failure_Reference, name Failure_Fragment, value Failure_Fragment,
) (err error) {
	Failure_Reference_Invariants(failure, "failure_expected_integer.failure")
	Failure_Fragment_Invariants(name, "failure_expected_integer.name")
	Failure_Fragment_Invariants(value, "failure_expected_integer.value")
	failure_reset(failure)
	failure_write(failure, name)
	failure_write(failure, Failure_Fragment{" expects a whole number, but got "})
	failure_write_quoted(failure, value)
	return failure[0]
}

func failure_write_string_set(
	failure Failure_Reference, members Failure_Strings,
) {
	Failure_Reference_Invariants(failure, "failure_write_string_set.failure")
	Failure_Strings_Invariants(members, "failure_write_string_set.members")
	size := 0
	for index, member := range members[0] {
		if index != 0 {
			size += len(", ")
		}
		size += len(member)
	}
	if size > strings.TEXT_SIZE_MAXIMUM {
		panic("Enum diagnostic exceeds shared text capacity.")
	}
	for index, member := range members[0] {
		if index != 0 {
			failure_write(failure, Failure_Fragment{", "})
		}
		failure_write(failure, Failure_Fragment{member})
	}
}

func failure_write_integer_set(
	failure Failure_Reference, members Failure_Integers,
) {
	Failure_Reference_Invariants(failure, "failure_write_integer_set.failure")
	Failure_Integers_Invariants(members, "failure_write_integer_set.members")
	size := 0
	for index, member := range members[0] {
		if index != 0 {
			size += len(", ")
		}
		var storage [strconv.DECIMAL_TEXT_SIZE_MAXIMUM]byte
		size += int(strconv.Format_Decimal_Into(
			storage[:], strconv.Machine_Integer(member),
		))
	}
	if size > strings.TEXT_SIZE_MAXIMUM {
		panic("Enum diagnostic exceeds shared text capacity.")
	}
	for index, member := range members[0] {
		if index != 0 {
			failure_write(failure, Failure_Fragment{", "})
		}
		failure_write_decimal(failure, Failure_Integer{member})
	}
}

// Arguments is a bounded positional declaration collection.
type Arguments []Option

// Arguments_Invariants bounds positional assignment work.
func Arguments_Invariants(value Arguments, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Flags is a bounded command flag declaration collection.
type Flags []Option

// Flags_Invariants bounds command flag search work.
func Flags_Invariants(value Flags, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Global_Flags is a bounded program-wide flag collection.
type Global_Flags []Option

// Global_Flags_Invariants bounds program-wide flag search work.
func Global_Flags_Invariants(value Global_Flags, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Help_Flags is fixed program-owned reserved option storage.
type Help_Flags [HELP_FLAG_COUNT]Option

// Help_Flags_Invariants fixes one option for each reserved help label.
func Help_Flags_Invariants(value Help_Flags, _ aver.Namespace) {
	aver.Always(
		len(value) == HELP_FLAG_COUNT,
		"Help flag storage matches reserved label count.",
	)
}

// SINGLE_COMMAND_COUNT follows the only selector-free command identity.
const SINGLE_COMMAND_COUNT = len("single") / len("single")

// Single_Commands is fixed inline storage for the one selector-free command.
type Single_Commands [SINGLE_COMMAND_COUNT]Command

// Single_Commands_Invariants keeps Program independent of borrowed command-slice storage.
func Single_Commands_Invariants(value Single_Commands, _ aver.Namespace) {
	aver.Always(
		len(value) == len(Single_Commands{}),
		"Selector-free command storage has its exact static length.",
	)
}

// PROGRAM_MODE_STORAGE_COUNT follows the one mutually exclusive selection mode.
const PROGRAM_MODE_STORAGE_COUNT = len("mode") / len("mode")

// Program_Mode stores one bounded command-selection mode without contradictory bits.
type Program_Mode [PROGRAM_MODE_STORAGE_COUNT]byte

// Program_Mode_Invariants rejects bytes beyond the three public modes.
func Program_Mode_Invariants(value Program_Mode, _ aver.Namespace) {
	aver.Always(
		value[0] <= PROGRAM_MODE_MULTICALL,
		"Program mode is commands, single, or multicall.",
	)
}

// PROGRAM_MODE_COMMANDS selects a command from an argument token.
const PROGRAM_MODE_COMMANDS byte = 0

// PROGRAM_MODE_SINGLE has one selector-free command.
const PROGRAM_MODE_SINGLE = PROGRAM_MODE_COMMANDS + 1

// PROGRAM_MODE_MULTICALL selects a command from the binary name.
const PROGRAM_MODE_MULTICALL = PROGRAM_MODE_SINGLE + 1

// Command_Argument_Storage is caller-owned active argument storage.
type Command_Argument_Storage []Option

// Command_Argument_Storage_Invariants bounds active argument copying.
func Command_Argument_Storage_Invariants(
	value Command_Argument_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Command_Flag_Storage is caller-owned active command flag storage.
type Command_Flag_Storage []Option

// Command_Flag_Storage_Invariants bounds active flag copying.
func Command_Flag_Storage_Invariants(
	value Command_Flag_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Global_Flag_Storage is caller-owned active global flag storage.
type Global_Flag_Storage []Option

// Global_Flag_Storage_Invariants bounds active global flag copying.
func Global_Flag_Storage_Invariants(
	value Global_Flag_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Environment_Variables is a bounded external declaration collection.
type Environment_Variables []Environment_Variable

// Environment_Variables_Invariants bounds environment resolution work.
func Environment_Variables_Invariants(
	value Environment_Variables, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Environment_Declarations is bounded program-owned environment metadata.
type Environment_Declarations []Environment_Variable

// Environment_Declarations_Invariants bounds program environment resolution work.
func Environment_Declarations_Invariants(
	value Environment_Declarations, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Secrets is a bounded secret declaration collection.
type Secrets []Secret

// Secrets_Invariants bounds secret submission work.
func Secrets_Invariants(value Secrets, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Secret_Declarations is bounded program-owned file metadata.
type Secret_Declarations []Secret

// Secret_Declarations_Invariants bounds program secret submission work.
func Secret_Declarations_Invariants(value Secret_Declarations, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// DEPRECATION_WARNING_SIZE_MAXIMUM holds the widest secret warning fields.
const DEPRECATION_WARNING_SIZE_MAXIMUM = len("command \"\" is deprecated: ") +
	2*strings.TEXT_SIZE_MAXIMUM

// DEPRECATION_WARNING_SIZE_MINIMUM follows one shortest valid deprecated flag.
const DEPRECATION_WARNING_SIZE_MINIMUM = len("-x is deprecated: d")

// Deprecation_Warning_Text is one bounded command, flag, environment, or secret warning.
type Deprecation_Warning_Text string

// Deprecation_Warning_Text_Invariants follows the widest warning construction.
func Deprecation_Warning_Text_Invariants(
	value Deprecation_Warning_Text, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), DEPRECATION_WARNING_SIZE_MINIMUM,
			DEPRECATION_WARNING_SIZE_MAXIMUM,
		).
		Ensure()
}

// WARNING_KIND_COUNT follows one source-kind tag.
const WARNING_KIND_COUNT = len("kind") / len("kind")

// Warning_Kind selects one deprecation message grammar.
type Warning_Kind [WARNING_KIND_COUNT]uint8

// Warning_Kind_Invariants admits command, flag, environment, and secret sources.
func Warning_Kind_Invariants(value Warning_Kind, _ aver.Namespace) {
	aver.Always(
		value[0] <= WARNING_KIND_SECRET,
		"Warning kind stays inside the four source grammars.",
	)
}

// WARNING_KIND_COMMAND selects quoted command grammar.
const WARNING_KIND_COMMAND uint8 = 0

// WARNING_KIND_FLAG selects dashed option grammar.
const WARNING_KIND_FLAG = WARNING_KIND_COMMAND + 1

// WARNING_KIND_ENVIRONMENT selects environment-variable grammar.
const WARNING_KIND_ENVIRONMENT = WARNING_KIND_FLAG + 1

// WARNING_KIND_SECRET selects secret grammar.
const WARNING_KIND_SECRET = WARNING_KIND_ENVIRONMENT + 1

// WARNING_NAME_COUNT follows one borrowed declaration identity.
const WARNING_NAME_COUNT = len("name") / len("name")

// Warning_Name keeps one source identity in phase-neutral storage.
type Warning_Name [WARNING_NAME_COUNT]Value_Text

// Warning_Name_Invariants bounds a borrowed warning identity structurally.
func Warning_Name_Invariants(value Warning_Name, _ aver.Namespace) {
	aver.Always(
		len(value[0]) <= strings.TEXT_SIZE_MAXIMUM,
		"Warning name stays inside shared text capacity.",
	)
}

// WARNING_GUIDANCE_COUNT follows one borrowed migration instruction.
const WARNING_GUIDANCE_COUNT = len("guidance") / len("guidance")

// Warning_Guidance keeps migration text in phase-neutral storage.
type Warning_Guidance [WARNING_GUIDANCE_COUNT]Deprecation

// Warning_Guidance_Invariants bounds borrowed guidance structurally.
func Warning_Guidance_Invariants(value Warning_Guidance, _ aver.Namespace) {
	aver.Always(
		len(value[0]) <= strings.TEXT_SIZE_MAXIMUM,
		"Warning guidance stays inside shared text capacity.",
	)
}

// Warning retains borrowed diagnostic parts without constructing owned text.
type Warning struct {
	// Kind selects punctuation and source spelling.
	Kind Warning_Kind
	// Name identifies the deprecated declaration.
	Name Warning_Name
	// Guidance tells the caller what replaces it.
	Guidance Warning_Guidance
}

// Warning_Invariants bounds borrowed parts without phase-specific boundary branches.
func Warning_Invariants(value Warning, namespace aver.Namespace) {
	Warning_Kind_Invariants(value.Kind, namespace)
	Warning_Name_Invariants(value.Name, namespace)
	Warning_Guidance_Invariants(value.Guidance, namespace)
}

// Deprecation_Warnings is bounded ready-to-write warning storage.
type Deprecation_Warnings []Warning

// Deprecation_Warnings_Invariants bounds warning publication work.
func Deprecation_Warnings_Invariants(
	value Deprecation_Warnings, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), slices.COUNT_MINIMUM, DEPRECATION_WARNING_COUNT_MAXIMUM,
		).
		Ensure()
}

// WRITABLE_DEPRECATION_WARNING_COUNT_MAXIMUM leaves one slot for the pending warning.
const WRITABLE_DEPRECATION_WARNING_COUNT_MAXIMUM = DEPRECATION_WARNING_COUNT_MAXIMUM -
	len("warning")/len("warning")

// Writable_Deprecation_Warnings is caller storage before one bounded extension.
type Writable_Deprecation_Warnings []Warning

// Writable_Deprecation_Warnings_Invariants proves one warning slot remains.
func Writable_Deprecation_Warnings_Invariants(
	value Writable_Deprecation_Warnings, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), slices.COUNT_MINIMUM,
			WRITABLE_DEPRECATION_WARNING_COUNT_MAXIMUM,
		).
		Ensure()
}

// Process_Arguments is one bounded process argument snapshot.
type Process_Arguments []string

// Process_Arguments_Invariants bounds parsing work.
func Process_Arguments_Invariants(value Process_Arguments, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Parse_Arguments is one process snapshot after the mandatory argv[0] guard.
type Parse_Arguments []string

// Parse_Arguments_Invariants excludes the rejected empty process snapshot.
func Parse_Arguments_Invariants(value Parse_Arguments, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), COMPLETION_WORD_COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// HELP_ARGUMENT_COUNT_MINIMUM follows argv[0] plus one reserved help token.
const HELP_ARGUMENT_COUNT_MINIMUM = len("program")/len("program") +
	len("help")/len("help")

// Help_Arguments is one process snapshot known to contain a help token.
type Help_Arguments []string

// Help_Arguments_Invariants excludes snapshots that cannot contain post-program help.
func Help_Arguments_Invariants(value Help_Arguments, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), HELP_ARGUMENT_COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Parsed_Tokens is any suffix after the mandatory program argument.
type Parsed_Tokens []string

// Parsed_Tokens_Invariants follows the maximum post-program token count.
func Parsed_Tokens_Invariants(value Parsed_Tokens, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, PARSED_TOKEN_COUNT_MAXIMUM).
		Ensure()
}

// Parse_Arguments_Start is the first single-command or selected-command value slot.
type Parse_Arguments_Start int

// Parse_Arguments_Start_Invariants admits the two parse grammar offsets.
func Parse_Arguments_Start_Invariants(
	value Parse_Arguments_Start, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Int(
			int(value), int(PARSE_ARGUMENTS_START_SINGLE),
			int(PARSE_ARGUMENTS_START_SELECTED),
		).
		Ensure()
}

// PARSE_ARGUMENTS_START_SINGLE skips the program word.
const PARSE_ARGUMENTS_START_SINGLE Parse_Arguments_Start = 1

// PARSE_ARGUMENTS_START_SELECTED also skips the command selector.
const PARSE_ARGUMENTS_START_SELECTED = PARSE_ARGUMENTS_START_SINGLE + 1

// COMPLETION_WORD_COUNT_MINIMUM follows the program word required after the public empty guard.
const COMPLETION_WORD_COUNT_MINIMUM = len("program") / len("program")

// Completion_Words is one nonempty shell-completion command line.
type Completion_Words []string

// Completion_Words_Invariants excludes the public Complete empty-input return.
func Completion_Words_Invariants(value Completion_Words, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), COMPLETION_WORD_COUNT_MINIMUM, slices.COUNT_MAXIMUM,
		).
		Ensure()
}

// Completion_Arguments_Start is the first single-command or selected-command value slot.
type Completion_Arguments_Start int

// Completion_Arguments_Start_Invariants admits the two command-line grammar offsets.
func Completion_Arguments_Start_Invariants(
	value Completion_Arguments_Start, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Int(
			int(value), int(COMPLETION_ARGUMENTS_START_SINGLE),
			int(COMPLETION_ARGUMENTS_START_SELECTED),
		).
		Ensure()
}

// COMPLETION_ARGUMENTS_START_SINGLE skips the program word.
const COMPLETION_ARGUMENTS_START_SINGLE Completion_Arguments_Start = 1

// COMPLETION_ARGUMENTS_START_SELECTED also skips the command selector.
const COMPLETION_ARGUMENTS_START_SELECTED = COMPLETION_ARGUMENTS_START_SINGLE + 1

// POSITIONAL_CURSOR_MAXIMUM reserves the program and current completion words.
const POSITIONAL_CURSOR_MAXIMUM = slices.COUNT_MAXIMUM - len("program")/len("program") -
	len("current")/len("current")

// Positional_Cursor counts completed bare words before the current word.
type Positional_Cursor int

// Positional_Cursor_Invariants follows the largest non-current completion prefix.
func Positional_Cursor_Invariants(value Positional_Cursor, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), slices.POSITION_MINIMUM, POSITIONAL_CURSOR_MAXIMUM).
		Ensure()
}

// Process_Environment is one bounded process environment snapshot.
type Process_Environment []string

// Process_Environment_Invariants bounds environment classification work.
func Process_Environment_Invariants(
	value Process_Environment, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// CANDIDATE_SEGMENT_COUNT holds dash, label, equals, and value without concatenation.
const CANDIDATE_SEGMENT_COUNT = len("dash")/len("dash") +
	len("label")/len("label") +
	len("equals")/len("equals") +
	len("value")/len("value")

// CANDIDATE_SIZE_MAXIMUM holds every bounded segment and integer rendering.
const CANDIDATE_SIZE_MAXIMUM = CANDIDATE_SEGMENT_COUNT*strings.TEXT_SIZE_MAXIMUM +
	strconv.DECIMAL_TEXT_SIZE_MAXIMUM

// Candidate_Size is complete structured completion text size.
type Candidate_Size int

// Candidate_Size_Invariants follows every independent bounded segment.
func Candidate_Size_Invariants(value Candidate_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), strings.TEXT_SIZE_MINIMUM, CANDIDATE_SIZE_MAXIMUM).
		Ensure()
}

// CANDIDATE_STATE_COUNT follows one scalar completion content cell.
const CANDIDATE_STATE_COUNT = len("state") / len("state")

// Candidate_State holds integer completion content outside the string union.
type Candidate_State [CANDIDATE_STATE_COUNT]struct {
	// Integer is rendered after Segments when Has_Integer is set.
	Integer Integer
	// Has_Integer distinguishes zero from absent integer content.
	Has_Integer Boolean
}

// Candidate_State_Invariants bounds the inactive or active scalar cell.
func Candidate_State_Invariants(value Candidate_State, _ aver.Namespace) {
	aver.Always(
		int(value[0].Integer) >= bits.INTEGER_MINIMUM,
		"Completion integer does not precede machine minimum.",
	)
	aver.Always(
		int(value[0].Integer) <= bits.INTEGER_MAXIMUM,
		"Completion integer does not follow machine maximum.",
	)
}

// Candidate is one completion assembled from borrowed segments or one integer scalar.
type Candidate struct {
	// Segments preserve rendered order without owning joined text.
	Segments [CANDIDATE_SEGMENT_COUNT]string
	// State keeps scalar union storage in one invariant cell.
	State Candidate_State
}

// Candidate_Invariants bounds each borrowed fragment; joining stays deferred.
func Candidate_Invariants(value Candidate, namespace aver.Namespace) {
	Candidate_State_Invariants(value.State, namespace)
	aver.Always(
		max(
			len(value.Segments[0]), len(value.Segments[1]),
			len(value.Segments[2]), len(value.Segments[3]),
		) <= strings.TEXT_SIZE_MAXIMUM,
		"Completion candidate segments fit bounded borrowed text.",
	)
}

// Candidate size measures complete rendering before fixed output accepts it.
func candidate_size(value Candidate) (size Candidate_Size) {
	defer func() { Candidate_Size_Invariants(size, "candidate_size.size") }()
	Candidate_Invariants(value, "candidate_size.value")
	for _, segment := range value.Segments {
		size += Candidate_Size(len(segment))
	}
	if value.State[0].Has_Integer {
		var storage [strconv.DECIMAL_TEXT_SIZE_MAXIMUM]byte
		size += Candidate_Size(strconv.Format_Decimal_Into(
			storage[:], strconv.Machine_Integer(value.State[0].Integer),
		))
	}
	return size
}

// CANDIDATE_COUNT_MAXIMUM covers arguments, command flags, help flags, and global flags.
const CANDIDATE_COUNT_MAXIMUM = 3*slices.COUNT_MAXIMUM + HELP_FLAG_COUNT

// Candidates is bounded caller-owned completion storage.
type Candidates []Candidate

// Candidates_Invariants bounds candidate search and rendering work.
func Candidates_Invariants(value Candidates, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, CANDIDATE_COUNT_MAXIMUM).
		Ensure()
}

// CANDIDATE_STORAGE_COUNT_MINIMUM follows one cell required for publication.
const CANDIDATE_STORAGE_COUNT_MINIMUM = len("candidate") / len("candidate")

// Candidate_Storage is caller storage after capacity was proven.
type Candidate_Storage []Candidate

// Candidate_Storage_Invariants excludes storage that cannot publish one candidate.
func Candidate_Storage_Invariants(
	value Candidate_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), CANDIDATE_STORAGE_COUNT_MINIMUM,
			CANDIDATE_COUNT_MAXIMUM,
		).
		Ensure()
}

// WRITABLE_CANDIDATE_COUNT_MAXIMUM leaves one caller cell for publication.
const WRITABLE_CANDIDATE_COUNT_MAXIMUM = CANDIDATE_COUNT_MAXIMUM -
	len("candidate")/len("candidate")

// Writable_Candidates is an initialized prefix before one candidate extension.
type Writable_Candidates []Candidate

// Writable_Candidates_Invariants proves one candidate cell remains.
func Writable_Candidates_Invariants(
	value Writable_Candidates, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), slices.COUNT_MINIMUM,
			WRITABLE_CANDIDATE_COUNT_MAXIMUM,
		).
		Ensure()
}

// Command_Candidates is one bounded selector label collection.
type Command_Candidates []Candidate

// Command_Candidates_Invariants follows command declaration count.
func Command_Candidates_Invariants(
	value Command_Candidates, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Enum_Candidates is one bounded enum member collection.
type Enum_Candidates []Candidate

// Enum_Candidates_Invariants follows enum member count.
func Enum_Candidates_Invariants(
	value Enum_Candidates, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Text_Parts is bounded borrowed formatting input.
type Text_Parts []string

// Text_Parts_Invariants bounds joining work.
func Text_Parts_Invariants(value Text_Parts, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// String_Enumeration is one bounded permitted-text collection.
type String_Enumeration []string

// String_Enumeration_Invariants bounds permitted-text membership work.
func String_Enumeration_Invariants(value String_Enumeration, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Integer_Enumeration is one bounded permitted-integer collection.
type Integer_Enumeration []int

// Integer_Enumeration_Invariants bounds permitted-integer membership work.
func Integer_Enumeration_Invariants(value Integer_Enumeration, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// ENUMERATION_COUNT_MINIMUM follows the first permitted member.
const ENUMERATION_COUNT_MINIMUM = len("member") / len("member")

// Permitted_Strings is one validated nonempty text enumeration.
type Permitted_Strings []string

// Permitted_Strings_Invariants binds conversion to a usable enumeration.
func Permitted_Strings_Invariants(value Permitted_Strings, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), ENUMERATION_COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Permitted_Integers is one validated nonempty integer enumeration.
type Permitted_Integers []int

// Permitted_Integers_Invariants binds conversion to a usable enumeration.
func Permitted_Integers_Invariants(
	value Permitted_Integers, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), ENUMERATION_COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Indexed_Tokens is bounded caller-owned token storage.
type Indexed_Tokens []Indexed_Token

// Indexed_Tokens_Invariants bounds token ordering work.
func Indexed_Tokens_Invariants(value Indexed_Tokens, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Positionals is caller-owned bare-token storage.
type Positionals []Indexed_Token

// Positionals_Invariants bounds positional assignment work.
func Positionals_Invariants(value Positionals, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Named_Variadic_Tokens is caller-owned named variadic token storage.
type Named_Variadic_Tokens []Indexed_Token

// Named_Variadic_Tokens_Invariants bounds variadic merge work.
func Named_Variadic_Tokens_Invariants(
	value Named_Variadic_Tokens, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// PARSED_TOKEN_COUNT_MAXIMUM reserves the mandatory program argument.
const PARSED_TOKEN_COUNT_MAXIMUM = slices.COUNT_MAXIMUM - len("program")/len("program")

// Parsed_Positionals is the active bare-token prefix after program selection.
type Parsed_Positionals []Indexed_Token

// Parsed_Positionals_Invariants follows the maximum remaining process arguments.
func Parsed_Positionals_Invariants(
	value Parsed_Positionals, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, PARSED_TOKEN_COUNT_MAXIMUM).
		Ensure()
}

// Parsed_Variadic_Tokens is the active named variadic prefix after program selection.
type Parsed_Variadic_Tokens []Indexed_Token

// Parsed_Variadic_Tokens_Invariants follows the maximum remaining process arguments.
func Parsed_Variadic_Tokens_Invariants(
	value Parsed_Variadic_Tokens, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, PARSED_TOKEN_COUNT_MAXIMUM).
		Ensure()
}

// Filled_Options records bounded option presence state.
type Filled_Options []bool

// Filled_Options_Invariants bounds presence scans.
func Filled_Options_Invariants(value Filled_Options, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// ASSIGNED_OPTION_COUNT_MINIMUM follows one option that defeats the parser fast path.
const ASSIGNED_OPTION_COUNT_MINIMUM = slices.COUNT_MINIMUM + len("option")/len("option")

// Assigned_Options records a nonempty command assignment phase.
type Assigned_Options []bool

// Assigned_Options_Invariants excludes the zero-option parser fast path.
func Assigned_Options_Invariants(value Assigned_Options, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), ASSIGNED_OPTION_COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Secret_Paths is one bounded ordered fallback path collection.
type Secret_Paths []string

// Secret_Paths_Invariants bounds fallback submission work.
func Secret_Paths_Invariants(value Secret_Paths, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// FAILURE_COUNT_MAXIMUM holds every environment and secret declaration failure.
const FAILURE_COUNT_MAXIMUM = 2 * slices.COUNT_MAXIMUM

// Failures is bounded caller-owned failure storage.
type Failures []error

// Failures_Invariants bounds failure publication work.
func Failures_Invariants(value Failures, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, FAILURE_COUNT_MAXIMUM).
		Ensure()
}

// Environment_Failures preserves environment declaration order.
type Environment_Failures []error

// Environment_Failures_Invariants bounds environment failure publication.
func Environment_Failures_Invariants(
	value Environment_Failures, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// PATH_FAILURE_KIND_NONE marks unused current failure state.
const PATH_FAILURE_KIND_NONE uint8 = uint8(len(""))

// PATH_FAILURE_KIND_STATUS wraps injected status failure.
const PATH_FAILURE_KIND_STATUS = PATH_FAILURE_KIND_NONE + 1

// PATH_FAILURE_KIND_ABSENT reports absent path.
const PATH_FAILURE_KIND_ABSENT = PATH_FAILURE_KIND_STATUS + 1

// PATH_FAILURE_KIND_NOT_REGULAR rejects non-regular file.
const PATH_FAILURE_KIND_NOT_REGULAR = PATH_FAILURE_KIND_ABSENT + 1

// PATH_FAILURE_KIND_NEGATIVE_SIZE rejects impossible status size.
const PATH_FAILURE_KIND_NEGATIVE_SIZE = PATH_FAILURE_KIND_NOT_REGULAR + 1

// PATH_FAILURE_KIND_FILE_TOO_LARGE rejects status or read overflow.
const PATH_FAILURE_KIND_FILE_TOO_LARGE = PATH_FAILURE_KIND_NEGATIVE_SIZE + 1

// PATH_FAILURE_KIND_OPEN wraps injected open failure.
const PATH_FAILURE_KIND_OPEN = PATH_FAILURE_KIND_FILE_TOO_LARGE + 1

// PATH_FAILURE_KIND_OPEN_NO_FILE rejects successful open without descriptor.
const PATH_FAILURE_KIND_OPEN_NO_FILE = PATH_FAILURE_KIND_OPEN + 1

// PATH_FAILURE_KIND_READ wraps injected read failure.
const PATH_FAILURE_KIND_READ = PATH_FAILURE_KIND_OPEN_NO_FILE + 1

// PATH_FAILURE_KIND_NEGATIVE_READ rejects negative byte count.
const PATH_FAILURE_KIND_NEGATIVE_READ = PATH_FAILURE_KIND_READ + 1

// PATH_FAILURE_KIND_SIZE_CHANGED rejects status/read disagreement.
const PATH_FAILURE_KIND_SIZE_CHANGED = PATH_FAILURE_KIND_NEGATIVE_READ + 1

// PATH_FAILURE_KIND_EMPTY rejects absent content.
const PATH_FAILURE_KIND_EMPTY = PATH_FAILURE_KIND_SIZE_CHANGED + 1

// PATH_FAILURE_KIND_STRING_ENUM rejects secret outside text set.
const PATH_FAILURE_KIND_STRING_ENUM = PATH_FAILURE_KIND_EMPTY + 1

// PATH_FAILURE_KIND_INTEGER rejects non-decimal secret.
const PATH_FAILURE_KIND_INTEGER = PATH_FAILURE_KIND_STRING_ENUM + 1

// PATH_FAILURE_KIND_INTEGER_ENUM rejects integer outside set.
const PATH_FAILURE_KIND_INTEGER_ENUM = PATH_FAILURE_KIND_INTEGER + 1

// PATH_FAILURE_KIND_BOOLEAN rejects non-Boolean secret.
const PATH_FAILURE_KIND_BOOLEAN = PATH_FAILURE_KIND_INTEGER_ENUM + 1

// PATH_FAILURE_KIND_COUNT isolates the tag from invariant sampling phases.
const PATH_FAILURE_KIND_COUNT = len("kind") / len("kind")

// Path_Failure_Kind selects one redacted path diagnostic.
type Path_Failure_Kind [PATH_FAILURE_KIND_COUNT]uint8

// Path_Failure_Kind_Invariants admits unused and every redacted cause.
func Path_Failure_Kind_Invariants(value Path_Failure_Kind, _ aver.Namespace) {
	aver.Always(
		value[0] <= PATH_FAILURE_KIND_BOOLEAN,
		"Path failure kind stays inside redacted diagnostic set.",
	)
}

// PATH_FAILURE_PATH_COUNT isolates borrowed path bounds from secret declaration bounds.
const PATH_FAILURE_PATH_COUNT = len("path") / len("path")

// Path_Failure_Path is one borrowed secret path identity.
type Path_Failure_Path [PATH_FAILURE_PATH_COUNT]string

// Path_Failure_Path_Invariants bounds diagnostic path identity.
func Path_Failure_Path_Invariants(value Path_Failure_Path, _ aver.Namespace) {
	aver.Always(
		len(value[0]) <= strings.TEXT_SIZE_MAXIMUM,
		"Path failure identity stays inside shared text capacity.",
	)
}

// Path_Failure keeps one ordered secret-path cause without rendered ownership.
type Path_Failure struct {
	// Path is borrowed declaration text.
	Path Path_Failure_Path
	// Kind selects redacted diagnostic grammar.
	Kind Path_Failure_Kind
	// Cause retains injected error identity.
	Cause error
	// Close_Cause preserves independent descriptor retirement failure.
	Close_Cause error
}

// Path_Failure_Invariants bounds borrowed path and failure tag.
func Path_Failure_Invariants(value Path_Failure, namespace aver.Namespace) {
	Path_Failure_Path_Invariants(value.Path, namespace)
	Path_Failure_Kind_Invariants(value.Kind, namespace)
}

// Secret_Failure keeps one declaration's ordered path failures.
type Secret_Failure struct {
	// Present distinguishes optional all-absent success.
	Present Boolean
	// Paths borrows initialized caller path records.
	Paths Path_Failures
}

// Secret_Failure_Invariants composes declaration failure state.
func Secret_Failure_Invariants(value Secret_Failure, namespace aver.Namespace) {
	Boolean_Invariants(value.Present, namespace)
	Path_Failures_Invariants(value.Paths, namespace)
}

// Secret_Failures preserves secret declaration order.
type Secret_Failures []Secret_Failure

// Secret_Failures_Invariants bounds secret failure publication.
func Secret_Failures_Invariants(value Secret_Failures, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Path_Failures preserves one secret's fallback order.
type Path_Failures []Path_Failure

// Path_Failures_Invariants bounds fallback failure publication.
func Path_Failures_Invariants(value Path_Failures, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Environment_Warnings preserves environment declaration order.
type Environment_Warnings []Warning

// Environment_Warnings_Invariants bounds environment warning publication.
func Environment_Warnings_Invariants(
	value Environment_Warnings, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), slices.COUNT_MINIMUM, DEPRECATION_WARNING_COUNT_MAXIMUM,
		).
		Ensure()
}

// Secret_Warnings preserves secret declaration order.
type Secret_Warnings []Warning

// Secret_Warnings_Invariants bounds secret warning publication.
func Secret_Warnings_Invariants(value Secret_Warnings, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// String_Values is bounded caller-owned string option storage.
type String_Values []string

// String_Values_Invariants bounds variadic string values.
func String_Values_Invariants(value String_Values, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Integer_Values is bounded caller-owned integer option storage.
type Integer_Values []int

// Integer_Values_Invariants bounds variadic integer values.
func Integer_Values_Invariants(value Integer_Values, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Secret_Bytes is caller-owned secret read storage plus overflow witness byte.
type Secret_Bytes []byte

// Secret_Bytes_Invariants admits only absent storage or the fixed overflow-witness buffer.
func Secret_Bytes_Invariants(value Secret_Bytes, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Int(len(value), slices.COUNT_MINIMUM, SECRET_BUFFER_BYTES_MAX).
		Ensure()
}

// Secret_Value_Bytes is one normalized view into caller-owned secret storage.
type Secret_Value_Bytes []byte

// Secret_Value_Bytes_Invariants bounds accepted content after newline removal.
func Secret_Value_Bytes_Invariants(value Secret_Value_Bytes, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, SECRET_BYTES_MAX).
		Ensure()
}

// Secret_Number_Bytes is the conversion prefix admitted by repository integer text.
type Secret_Number_Bytes []byte

// Secret_Number_Bytes_Invariants bounds decimal parsing work.
func Secret_Number_Bytes_Invariants(value Secret_Number_Bytes, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Secret_Buffers gives every declaration one independent overflow-witness buffer.
type Secret_Buffers []Secret_Bytes

// Secret_Buffers_Invariants bounds declaration-indexed buffer storage.
func Secret_Buffers_Invariants(value Secret_Buffers, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Secret_Parsers gives every declaration one stable asynchronous runner.
type Secret_Parsers []Secret_Parser

// Secret_Parsers_Invariants bounds declaration-indexed runner storage.
func Secret_Parsers_Invariants(value Secret_Parsers, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Secret_Path_Failures gives every declaration ordered fallback failure storage.
type Secret_Path_Failures []Path_Failures

// Secret_Path_Failures_Invariants bounds declaration-indexed fallback storage.
func Secret_Path_Failures_Invariants(
	value Secret_Path_Failures, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// External_Key_Set tracks bounded declaration presence.
type External_Key_Set map[External_Key]Boolean

// External_Key_Set_Invariants bounds duplicate detection storage.
func External_Key_Set_Invariants(value External_Key_Set, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Environment_Sources indexes bounded environment classifications.
type Environment_Sources map[External_Key]Environment_Source

// Environment_Sources_Invariants bounds source lookup storage.
func Environment_Sources_Invariants(
	value Environment_Sources, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// PROGRAM_SELECTION_COUNT follows one active command-selection arm.
const PROGRAM_SELECTION_COUNT = len("selection") / len("selection")

// Program_Selection groups mutually exclusive selector state in fixed inline storage.
type Program_Selection [PROGRAM_SELECTION_COUNT]struct {
	// Commands is selector-owned command storage.
	Commands Commands
	// Single_Commands is selector-free inline storage.
	Single_Commands Single_Commands
	// Global_Flags is selector-owned global flag storage.
	Global_Flags Global_Flags
	// Help_Flags keeps reserved named options beside global flags.
	Help_Flags Help_Flags
	// Mode chooses the active command-selection arm.
	Mode Program_Mode
}

// Program_Selection_Invariants bounds both arms without inventing impossible witnesses.
func Program_Selection_Invariants(value Program_Selection, namespace aver.Namespace) {
	selection := value[0]
	aver.Always(
		len(selection.Commands) <= slices.COUNT_MAXIMUM,
		"Selected commands stay inside shared collection capacity.",
	)
	Single_Commands_Invariants(selection.Single_Commands, namespace)
	aver.Always(
		len(selection.Global_Flags) <= slices.COUNT_MAXIMUM,
		"Selected global flags stay inside shared collection capacity.",
	)
	Help_Flags_Invariants(selection.Help_Flags, namespace)
	Program_Mode_Invariants(selection.Mode, namespace)
}

// Program represents a command-line application with one or more commands.
type Program struct {
	// Label is the program name shown in help output.
	Label Program_Label
	// Description is the one-line program summary shown in help output.
	Description Program_Description
	// Selection owns the one active command-resolution arm.
	Selection Program_Selection
	// Environment_Variables are process-wide external-value declarations.
	Environment_Variables Environment_Declarations
	// Secrets are process-wide file-backed external-value declarations.
	Secrets Secret_Declarations
}

// Program_Invariants composes program identity, declarations, and selection mode.
func Program_Invariants(value Program, namespace aver.Namespace) {
	Program_Label_Invariants(value.Label, namespace)
	Program_Description_Invariants(value.Description, namespace)
	Program_Selection_Invariants(value.Selection, namespace)
	Environment_Declarations_Invariants(value.Environment_Variables, namespace)
	Secret_Declarations_Invariants(value.Secrets, namespace)
}

// Resolved_Deprecation_Warnings is phase-produced warning storage.
type Resolved_Deprecation_Warnings []Warning

// Resolved_Deprecation_Warnings_Invariants bounds warnings by fixed output capacity.
func Resolved_Deprecation_Warnings_Invariants(
	value Resolved_Deprecation_Warnings, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), slices.COUNT_MINIMUM, DEPRECATION_WARNING_COUNT_MAXIMUM,
		).
		Ensure()
}

// Resolved_Environment is phase-produced environment storage.
type Resolved_Environment []Environment_Variable

// Resolved_Environment_Invariants bounds resolved environment structurally.
func Resolved_Environment_Invariants(
	value Resolved_Environment, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Resolved_Secrets is phase-produced secret storage.
type Resolved_Secrets []Secret

// Resolved_Secrets_Invariants bounds resolved secrets structurally.
func Resolved_Secrets_Invariants(value Resolved_Secrets, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Command represents a single command within a program.
// Commands have a label, optional arguments (required, ordered),
// and optional flags (optional, unordered).
type Command struct {
	// Label is the command name typed on the command line.
	Label Label
	// Description is the one-line command summary shown in help output.
	Description Description
	// Arguments are ordered and must ALL appear before flags. Each is required,
	// except a variadic last argument, which collects zero or more positionals.
	Arguments Arguments
	// Flags are optional and unordered.
	Flags Flags
	// Hidden omits the command from help and completion; it still resolves when named.
	Hidden Hidden
	// Deprecated, when non-empty, is the guidance shown when the command is used. A
	// deprecated command is hidden like a hidden one and warns on use.
	Deprecated Deprecation
	// Deprecation_Warnings is filled by Program_Parse on the returned command: one
	// ready-to-print message per deprecated command or flag the invocation actually
	// used. It is empty on a command held as a definition.
	Deprecation_Warnings Resolved_Deprecation_Warnings
	// Environment contains the resolved program-wide environment variables.
	Environment Resolved_Environment
	// Secrets contains the resolved program-wide file-backed secrets.
	Secrets Resolved_Secrets
}

// Command_Invariants composes command identity, declarations, and resolved externals.
func Command_Invariants(value Command, namespace aver.Namespace) {
	Label_Invariants(value.Label, namespace)
	Description_Invariants(value.Description, namespace)
	Arguments_Invariants(value.Arguments, namespace)
	Flags_Invariants(value.Flags, namespace)
	Hidden_Invariants(value.Hidden, namespace)
	Deprecation_Invariants(value.Deprecated, namespace)
	Resolved_Deprecation_Warnings_Invariants(value.Deprecation_Warnings, namespace)
	Resolved_Environment_Invariants(value.Environment, namespace)
	Resolved_Secrets_Invariants(value.Secrets, namespace)
}

// Option represents either an argument or a flag for a command.
type Option struct {
	// Label is the argument or flag name.
	Label Option_Label
	// Description is the one-line summary shown in help output.
	Description Description
	// Type prevents scalar and slice values from escaping through an interface box.
	Type Option_Type_State
	// Enumeration keeps both borrowed enum arms in fixed union storage.
	Enumeration Option_Enumeration
	// Value accepts legacy composite declarations until callers use Type and State.
	Value any
	// Enum accepts legacy composite declarations until callers use typed enum fields.
	Enum any
	// State keeps mutually exclusive parsed and flag-only fields in fixed inline storage.
	State Option_State
}

// Option_Invariants composes bounded definition and parsed scalar storage.
func Option_Invariants(value Option, namespace aver.Namespace) {
	Option_Label_Invariants(value.Label, namespace)
	Description_Invariants(value.Description, namespace)
	Option_Type_State_Invariants(value.Type, namespace)
	Option_Enumeration_Invariants(value.Enumeration, namespace)
	Option_State_Invariants(value.State, namespace)
}

// OPTION_ENUMERATION_COUNT follows one fixed enum union cell.
const OPTION_ENUMERATION_COUNT = len("enum") / len("enum")

// Option_Enumeration keeps typed permitted sets outside interface storage.
type Option_Enumeration [OPTION_ENUMERATION_COUNT]struct {
	String   String_Enumeration
	Integers Integer_Enumeration
}

// Option_Enumeration_Invariants bounds both inactive or active borrowed arms.
func Option_Enumeration_Invariants(value Option_Enumeration, _ aver.Namespace) {
	aver.Always(
		len(value[0].String) <= slices.COUNT_MAXIMUM,
		"Option text enum stays inside shared collection capacity.",
	)
	aver.Always(
		len(value[0].Integers) <= slices.COUNT_MAXIMUM,
		"Option integer enum stays inside shared collection capacity.",
	)
}

// OPTION_STATE_COUNT follows the one active option state record.
const OPTION_STATE_COUNT = len("state") / len("state")

// Option_State keeps parse storage and flag-only metadata out of common option shape.
type Option_State [OPTION_STATE_COUNT]struct {
	String     Value_Text
	Integer    Integer
	Boolean    Boolean
	Strings    String_Values
	Integers   Integer_Values
	Parsed     Parsed
	Is_Flag    Is_Flag
	Hidden     Hidden
	Deprecated Deprecation
}

// Option_State_Invariants enforces union storage without phase-specific branch trees.
func Option_State_Invariants(value Option_State, _ aver.Namespace) {
	state := value[0]
	aver.Always(
		len(state.String) <= strings.TEXT_SIZE_MAXIMUM,
		"Parsed string storage stays inside shared text capacity.",
	)
	aver.Always(
		len(state.Strings) <= slices.COUNT_MAXIMUM,
		"Parsed string slices stay inside shared collection capacity.",
	)
	aver.Always(
		len(state.Integers) <= slices.COUNT_MAXIMUM,
		"Parsed integer slices stay inside shared collection capacity.",
	)
	aver.Always(
		len(state.Deprecated) <= strings.TEXT_SIZE_MAXIMUM,
		"Option deprecation stays inside shared text capacity.",
	)
}

func option_type[T string | int | bool]() (value Scalar_Option_Type) {
	defer func() { Scalar_Option_Type_Invariants(value, "option_type.value") }()
	var zero T
	switch any(zero).(type) {
	case string:
		return Scalar_Option_Type(OPTION_TYPE_STRING)
	case int:
		return Scalar_Option_Type(OPTION_TYPE_INTEGER)
	case bool:
		return Scalar_Option_Type(OPTION_TYPE_BOOLEAN)
	}
	panic("Option type is outside its generic constraint.")
}

func option_state_default[T string | int | bool](value T) (state Option_State) {
	defer func() { Option_State_Invariants(state, "option_state_default.state") }()
	switch typed := any(value).(type) {
	case string:
		state[0].String = Value_Text(typed)
	case int:
		state[0].Integer = Integer(typed)
	case bool:
		state[0].Boolean = Boolean(typed)
	}
	return state
}

func option_legacy_initialize(option *Option) {
	Option_Invariants(*option, "option_legacy_initialize.option")
	if option.Value != nil {
		switch value := option.Value.(type) {
		case string:
			option.Type[0] = OPTION_TYPE_STRING
			if !option.State[0].Parsed {
				option.State[0].String = Value_Text(value)
			}
		case int:
			option.Type[0] = OPTION_TYPE_INTEGER
			if !option.State[0].Parsed {
				option.State[0].Integer = Integer(value)
			}
		case bool:
			option.Type[0] = OPTION_TYPE_BOOLEAN
			if !option.State[0].Parsed {
				option.State[0].Boolean = Boolean(value)
			}
		case []string:
			option.Type[0] = OPTION_TYPE_STRINGS
			if !option.State[0].Parsed {
				option.State[0].Strings = value
			}
		case []int:
			option.Type[0] = OPTION_TYPE_INTEGERS
			if !option.State[0].Parsed {
				option.State[0].Integers = value
			}
		}
	}
	switch enum := option.Enum.(type) {
	case []string:
		option.Enumeration[0].String = String_Enumeration(enum)
	case []int:
		option.Enumeration[0].Integers = Integer_Enumeration(enum)
	}
}

// New_Argument_Input is the input for New_Argument.
type New_Argument_Input struct {
	// Label is the argument name.
	Label Option_Label
	// Description is the one-line summary shown in help output.
	Description Description
}

// New_Argument_Input_Invariants composes bounded positional definition text.
func New_Argument_Input_Invariants(value New_Argument_Input, namespace aver.Namespace) {
	Option_Label_Invariants(value.Label, namespace)
	Description_Invariants(value.Description, namespace)
}

// New_Argument creates a required positional argument with the specified type.
// Arguments always have zero values; custom defaults are not allowed.
// Type parameter T must be string, int, or bool.
func New_Argument[T string | int | bool](input New_Argument_Input) (option Option) {
	defer func() { Option_Invariants(option, "new_argument.option") }()
	New_Argument_Input_Invariants(input, "new_argument.input")
	var zero_value T
	option = Option{
		Label:       input.Label,
		Description: input.Description,
		Type:        Option_Type_State{Option_Type(option_type[T]())},
		State:       option_state_default(zero_value),
	}
	return option
}

// New_Variadic_Input is the input for New_Variadic.
type New_Variadic_Input struct {
	// Label is the argument name.
	Label Option_Label
	// Description is the one-line summary shown in help output.
	Description Description
}

// New_Variadic_Input_Invariants composes bounded variadic definition text.
func New_Variadic_Input_Invariants(value New_Variadic_Input, namespace aver.Namespace) {
	Option_Label_Invariants(value.Label, namespace)
	Description_Invariants(value.Description, namespace)
}

// New_Variadic creates a slice-valued argument: it collects the trailing positionals
// and appends each repeated -label=value. It must be the last argument; scalar
// arguments may precede it. A slice argument is always optional — an empty list is
// valid, not an error — so a command that needs at least one element checks the
// slice's length itself. Type parameter T must be string or int; the Value is a []T,
// and that slice type is what marks the option variadic.
func New_Variadic[T string | int](input New_Variadic_Input) (option Option) {
	defer func() { Option_Invariants(option, "new_variadic.option") }()
	New_Variadic_Input_Invariants(input, "new_variadic.input")
	option = Option{Label: input.Label, Description: input.Description}
	var zero T
	switch any(zero).(type) {
	case string:
		option.Type[0] = OPTION_TYPE_STRINGS
	case int:
		option.Type[0] = OPTION_TYPE_INTEGERS
	}
	return option
}

// New_Flag_Input is the input for New_Flag.
type New_Flag_Input[T string | int | bool] struct {
	// Label is the flag name.
	Label Option_Label
	// Value is the flag's default value.
	Value T
	// Description is the one-line summary shown in help output.
	Description Description
	// Hidden omits the flag from help and completion; it still parses when named.
	Hidden Hidden
	// Deprecated, when non-empty, is the guidance shown when the flag is used.
	Deprecated Deprecation
}

// New_Flag_Input_Invariants composes bounded flag metadata.
func New_Flag_Input_Invariants[T string | int | bool](
	value New_Flag_Input[T], namespace aver.Namespace,
) {
	Option_Label_Invariants(value.Label, namespace)
	Description_Invariants(value.Description, namespace)
	Hidden_Invariants(value.Hidden, namespace)
	Deprecation_Invariants(value.Deprecated, namespace)
}

// New_Flag creates an optional flag with a default value.
// Type parameter T must be string, int, or bool.
func New_Flag[T string | int | bool](input New_Flag_Input[T]) (option Option) {
	defer func() { Option_Invariants(option, "new_flag.option") }()
	New_Flag_Input_Invariants(input, "new_flag.input")
	option = Option{
		Label:       input.Label,
		Description: input.Description,
		Type:        Option_Type_State{Option_Type(option_type[T]())},
		State: Option_State{{
			Is_Flag: true, Hidden: input.Hidden, Deprecated: input.Deprecated,
		}},
	}
	defaults := option_state_default(input.Value)
	option.State[0].String = defaults[0].String
	option.State[0].Integer = defaults[0].Integer
	option.State[0].Boolean = defaults[0].Boolean
	return option
}

// New_String_Enum_Flag_Input declares one permitted-text flag.
type New_String_Enum_Flag_Input struct {
	// Label shares the one named option namespace.
	Label Option_Label
	// Enum stays borrowed so construction owns no hidden storage.
	Enum String_Enumeration
	// Value must be validated against Enum during program construction.
	Value String_Enum_Default
	// Description stays metadata rather than parser state.
	Description Description
	// Hidden affects advertising without disabling exact parsing.
	Hidden Hidden
	// Deprecated preserves compatibility while suppressing advertising.
	Deprecated Deprecation
}

// New_String_Enum_Flag_Input_Invariants composes bounded text-enum flag metadata.
func New_String_Enum_Flag_Input_Invariants(
	value New_String_Enum_Flag_Input, namespace aver.Namespace,
) {
	Option_Label_Invariants(value.Label, namespace)
	String_Enumeration_Invariants(value.Enum, namespace)
	String_Enum_Default_Invariants(value.Value, namespace)
	Description_Invariants(value.Description, namespace)
	Hidden_Invariants(value.Hidden, namespace)
	Deprecation_Invariants(value.Deprecated, namespace)
}

// New_String_Enum_Flag creates one permitted-text flag.
func New_String_Enum_Flag(input New_String_Enum_Flag_Input) (option Option) {
	defer func() { Option_Invariants(option, "new_string_enum_flag.option") }()
	New_String_Enum_Flag_Input_Invariants(input, "new_string_enum_flag.input")
	option = Option{
		Label: input.Label, Description: input.Description,
		Type:        Option_Type_State{OPTION_TYPE_STRING},
		Enumeration: Option_Enumeration{{String: input.Enum}},
		State: Option_State{{
			String: Value_Text(input.Value), Is_Flag: true,
			Hidden: input.Hidden, Deprecated: input.Deprecated,
		}},
	}
	return option
}

// New_Integer_Enum_Flag_Input declares one permitted-integer flag.
type New_Integer_Enum_Flag_Input struct {
	// Label shares the one named option namespace.
	Label Option_Label
	// Enum stays borrowed so construction owns no hidden storage.
	Enum Integer_Enumeration
	// Value must be validated against Enum during program construction.
	Value Integer_Enum_Default
	// Description stays metadata rather than parser state.
	Description Description
	// Hidden affects advertising without disabling exact parsing.
	Hidden Hidden
	// Deprecated preserves compatibility while suppressing advertising.
	Deprecated Deprecation
}

// New_Integer_Enum_Flag_Input_Invariants composes bounded integer-enum flag metadata.
func New_Integer_Enum_Flag_Input_Invariants(
	value New_Integer_Enum_Flag_Input, namespace aver.Namespace,
) {
	Option_Label_Invariants(value.Label, namespace)
	Integer_Enumeration_Invariants(value.Enum, namespace)
	Integer_Enum_Default_Invariants(value.Value, namespace)
	Description_Invariants(value.Description, namespace)
	Hidden_Invariants(value.Hidden, namespace)
	Deprecation_Invariants(value.Deprecated, namespace)
}

// New_Integer_Enum_Flag creates one permitted-integer flag.
func New_Integer_Enum_Flag(input New_Integer_Enum_Flag_Input) (option Option) {
	defer func() { Option_Invariants(option, "new_integer_enum_flag.option") }()
	New_Integer_Enum_Flag_Input_Invariants(input, "new_integer_enum_flag.input")
	option = Option{
		Label: input.Label, Description: input.Description,
		Type:        Option_Type_State{OPTION_TYPE_INTEGER},
		Enumeration: Option_Enumeration{{Integers: input.Enum}},
		State: Option_State{{
			Integer: Integer(input.Value), Is_Flag: true,
			Hidden: input.Hidden, Deprecated: input.Deprecated,
		}},
	}
	return option
}

// New_String_Enum_Argument_Input declares one permitted-text positional.
type New_String_Enum_Argument_Input struct {
	// Label permits the same value by position or name.
	Label Option_Label
	// Enum stays borrowed so construction owns no hidden storage.
	Enum String_Enumeration
	// Description stays metadata rather than parser state.
	Description Description
}

// New_String_Enum_Argument_Input_Invariants composes bounded text-enum positional metadata.
func New_String_Enum_Argument_Input_Invariants(
	value New_String_Enum_Argument_Input, namespace aver.Namespace,
) {
	Option_Label_Invariants(value.Label, namespace)
	String_Enumeration_Invariants(value.Enum, namespace)
	Description_Invariants(value.Description, namespace)
}

// New_String_Enum_Argument creates one permitted-text positional.
func New_String_Enum_Argument(input New_String_Enum_Argument_Input) (option Option) {
	defer func() { Option_Invariants(option, "new_string_enum_argument.option") }()
	New_String_Enum_Argument_Input_Invariants(input, "new_string_enum_argument.input")
	option = Option{
		Label: input.Label, Description: input.Description,
		Type:        Option_Type_State{OPTION_TYPE_STRING},
		Enumeration: Option_Enumeration{{String: input.Enum}},
	}
	return option
}

// New_Integer_Enum_Argument_Input declares one permitted-integer positional.
type New_Integer_Enum_Argument_Input struct {
	// Label permits the same value by position or name.
	Label Option_Label
	// Enum stays borrowed so construction owns no hidden storage.
	Enum Integer_Enumeration
	// Description stays metadata rather than parser state.
	Description Description
}

// New_Integer_Enum_Argument_Input_Invariants composes bounded integer-enum metadata.
func New_Integer_Enum_Argument_Input_Invariants(
	value New_Integer_Enum_Argument_Input, namespace aver.Namespace,
) {
	Option_Label_Invariants(value.Label, namespace)
	Integer_Enumeration_Invariants(value.Enum, namespace)
	Description_Invariants(value.Description, namespace)
}

// New_Integer_Enum_Argument creates one permitted-integer positional.
func New_Integer_Enum_Argument(input New_Integer_Enum_Argument_Input) (option Option) {
	defer func() { Option_Invariants(option, "new_integer_enum_argument.option") }()
	New_Integer_Enum_Argument_Input_Invariants(input, "new_integer_enum_argument.input")
	option = Option{
		Label: input.Label, Description: input.Description,
		Type:        Option_Type_State{OPTION_TYPE_INTEGER},
		Enumeration: Option_Enumeration{{Integers: input.Enum}},
	}
	return option
}

// New_Input is the input for New.
type New_Input struct {
	// Label is the program name.
	Label Label
	// Description is the one-line program summary.
	Description Description
	// Global_Flags are flags accepted by every command.
	Global_Flags Global_Flags
	// Environment_Variables are process-wide external-value declarations.
	Environment_Variables Environment_Variables
	// Secrets are process-wide file-backed external-value declarations.
	Secrets Secrets
	// Commands are the program's commands.
	Commands Command_Declarations
}

// New_Input_Invariants composes bounded multi-command declarations.
func New_Input_Invariants(value New_Input, namespace aver.Namespace) {
	Label_Invariants(value.Label, namespace)
	Description_Invariants(value.Description, namespace)
	Global_Flags_Invariants(value.Global_Flags, namespace)
	Environment_Variables_Invariants(value.Environment_Variables, namespace)
	Secrets_Invariants(value.Secrets, namespace)
	Command_Declarations_Invariants(value.Commands, namespace)
}

// New creates a new Program from the given label, description, global flags, and
// commands. It validates that all commands have labels and that arguments and
// flags are properly configured, panicking when validation fails.
func New(input New_Input) (program Program) {
	defer func() { Program_Invariants(program, "new.program") }()
	New_Input_Invariants(input, "new.input")
	if len(input.Commands) == 0 {
		panic("Program has zero commands specified.")
	}

	// The injection occurs before validation because a reserved label must give a clear error.
	reserve_help_labels(Commands(input.Commands), input.Global_Flags)
	global_flags := input.Global_Flags

	for index := range global_flags {
		option_legacy_initialize(&global_flags[index])
		flag := global_flags[index]
		if !flag.State[0].Is_Flag {
			panic("Global flags must be created with New_Flag.")
		}
		validate_flag_label(flag)
	}

	program = Program{
		Label:       Program_Label(input.Label),
		Description: Program_Description(input.Description),
		Selection: Program_Selection{{
			Commands: Commands(input.Commands), Global_Flags: global_flags,
			Help_Flags: help_flags(),
		}},
		Environment_Variables: Environment_Declarations(input.Environment_Variables),
		Secrets:               Secret_Declarations(input.Secrets),
	}
	external_validate(input.Environment_Variables, input.Secrets)

	for _, command := range program.Selection[0].Commands {
		if command.Label == "" {
			panic("Program command label is unset.")
		}
		command_validate_options(
			command.Label, command.Arguments, command.Flags, global_flags,
		)
	}
	return program
}

// New_Single_Input is the input for New_Single.
type New_Single_Input struct {
	// Label is the program name.
	Label Label
	// Description is the one-line program summary.
	Description Description
	// Arguments are the program's required, ordered positional arguments. They must
	// ALL appear before flags.
	Arguments Arguments
	// Flags are the program's optional, unordered flags.
	Flags Flags
	// Environment_Variables are process-wide external-value declarations.
	Environment_Variables Environment_Variables
	// Secrets are process-wide file-backed external-value declarations.
	Secrets Secrets
}

// New_Single_Input_Invariants composes bounded selector-free declarations.
func New_Single_Input_Invariants(value New_Single_Input, namespace aver.Namespace) {
	Label_Invariants(value.Label, namespace)
	Description_Invariants(value.Description, namespace)
	Arguments_Invariants(value.Arguments, namespace)
	Flags_Invariants(value.Flags, namespace)
	Environment_Variables_Invariants(value.Environment_Variables, namespace)
	Secrets_Invariants(value.Secrets, namespace)
}

// New_Single creates a program with no command selector: the first token after the
// program name is the first positional argument, not a command name. Use it for a
// binary that does one thing, like `sloc ./src`. It validates arguments and flags
// the same way New does, panicking when validation fails.
//
// A single-command program has no command namespace, so its first positional can
// take any value. This is the only configuration where a defaulted command and its
// positional arguments do not collide on the first token: with sibling commands,
// that token must select a command, and a positional sharing a command's name would
// be unreachable. A program that needs sibling commands must use New.
func New_Single(input New_Single_Input) (program Program) {
	defer func() { Program_Invariants(program, "new_single.program") }()
	New_Single_Input_Invariants(input, "new_single.input")
	command := Command{
		Label:       input.Label,
		Description: input.Description,
		Arguments:   input.Arguments,
		Flags:       input.Flags,
	}
	reserve_help_command(command.Arguments, command.Flags)
	command_validate_options(command.Label, command.Arguments, command.Flags, nil)
	external_validate(input.Environment_Variables, input.Secrets)
	return Program{
		Label:       Program_Label(input.Label),
		Description: Program_Description(input.Description),
		Selection: Program_Selection{{
			Single_Commands: Single_Commands{command}, Help_Flags: help_flags(),
			Mode: Program_Mode{PROGRAM_MODE_SINGLE},
		}},
		Environment_Variables: Environment_Declarations(input.Environment_Variables),
		Secrets:               Secret_Declarations(input.Secrets),
	}
}

// These flags make the two help forms visible in help output and completion output.
// The parser reads the tokens before it assigns option values.
func help_flags() (flags Help_Flags) {
	defer func() { Help_Flags_Invariants(flags, "help_flags.flags") }()
	return Help_Flags{
		{
			Label: HELP_SHORT_LABEL, Description: "show this help",
			Type:  Option_Type_State{OPTION_TYPE_BOOLEAN},
			State: Option_State{{Is_Flag: true}},
		},
		{
			Label: HELP_LABEL, Description: "show this help",
			Type:  Option_Type_State{OPTION_TYPE_BOOLEAN},
			State: Option_State{{Is_Flag: true}},
		},
	}
}

// A program cannot replace a default help flag with a different function.
func reserve_help_labels(commands Commands, global_flags Global_Flags) {
	Commands_Invariants(commands, "reserve_help_labels.commands")
	Global_Flags_Invariants(global_flags, "reserve_help_labels.global_flags")
	for _, flag := range global_flags {
		if help_label_reserved(flag.Label) {
			panic("Option label is reserved for an auto-injected help flag.")
		}
	}
	for _, command := range commands {
		for _, argument := range command.Arguments {
			if help_label_reserved(argument.Label) {
				panic("Option label is reserved for an auto-injected help flag.")
			}
		}
		for _, flag := range command.Flags {
			if help_label_reserved(flag.Label) {
				panic("Option label is reserved for an auto-injected help flag.")
			}
		}
	}
}

func reserve_help_command(arguments Arguments, flags Flags) {
	Arguments_Invariants(arguments, "reserve_help_command.arguments")
	Flags_Invariants(flags, "reserve_help_command.flags")
	for _, argument := range arguments {
		if help_label_reserved(argument.Label) {
			panic("Option label is reserved for an auto-injected help flag.")
		}
	}
	for _, flag := range flags {
		if help_label_reserved(flag.Label) {
			panic("Option label is reserved for an auto-injected help flag.")
		}
	}
}

func help_label_reserved(label Option_Label) (reserved Boolean) {
	defer func() { Boolean_Invariants(reserved, "help_label_reserved.reserved") }()
	Option_Label_Invariants(label, "help_label_reserved.label")
	switch label {
	case HELP_SHORT_LABEL, HELP_LABEL:
		return true
	default:
		return false
	}
}

// New_Multicall_Input is the input for New_Multicall.
type New_Multicall_Input struct {
	// Label is the program name shown in help output.
	Label Label
	// Description is the one-line program summary.
	Description Description
	// Global_Flags are flags accepted by every command.
	Global_Flags Global_Flags
	// Environment_Variables are process-wide external-value declarations.
	Environment_Variables Environment_Variables
	// Secrets are process-wide file-backed external-value declarations.
	Secrets Secrets
	// Commands are the program's commands, each selected by its label matching the
	// binary name in argv[0].
	Commands Command_Declarations
}

// New_Multicall_Input_Invariants composes bounded binary-selected declarations.
func New_Multicall_Input_Invariants(
	value New_Multicall_Input, namespace aver.Namespace,
) {
	Label_Invariants(value.Label, namespace)
	Description_Invariants(value.Description, namespace)
	Global_Flags_Invariants(value.Global_Flags, namespace)
	Environment_Variables_Invariants(value.Environment_Variables, namespace)
	Secrets_Invariants(value.Secrets, namespace)
	Command_Declarations_Invariants(value.Commands, namespace)
}

// New_Multicall creates a program whose command is selected by the binary name in
// argv[0], not a token in slot 1 — the model of a busybox-style binary symlinked to
// each of its command names. It validates commands and global flags exactly as New
// does, panicking when validation fails.
func New_Multicall(input New_Multicall_Input) (program Program) {
	defer func() { Program_Invariants(program, "new_multicall.program") }()
	New_Multicall_Input_Invariants(input, "new_multicall.input")
	program = New(New_Input{
		Label:                 input.Label,
		Description:           input.Description,
		Global_Flags:          input.Global_Flags,
		Environment_Variables: input.Environment_Variables,
		Secrets:               input.Secrets,
		Commands:              input.Commands,
	})
	program.Selection[0].Mode[0] = PROGRAM_MODE_MULTICALL
	return program
}

// Panics when any of a command's options is malformed: a label that is empty, not
// flag-safe, or shared with another option (arguments and flags occupy one
// -label=value namespace), an unsupported value type, or a slice argument that is not
// last. The global flags are pre-seeded so an argument or flag colliding with one is
// caught here too.
func command_validate_options(
	label Label, arguments Arguments, flags Flags, global_flags Global_Flags,
) {
	Label_Invariants(label, "command_validate_options.label")
	Arguments_Invariants(arguments, "command_validate_options.arguments")
	Flags_Invariants(flags, "command_validate_options.flags")
	Global_Flags_Invariants(global_flags, "command_validate_options.global_flags")
	for argument_index := range arguments {
		option_legacy_initialize(&arguments[argument_index])
		argument := arguments[argument_index]
		option_validate_label(argument)
		// A slice argument absorbs every remaining positional, so anything declared
		// after it is unreachable; requiring it last also forbids a second slice.
		is_last := argument_index == len(arguments)-1
		if option_is_slice(argument.Type[0]) {
			if !is_last {
				panic("Slice argument must be the last argument.")
			}
		}
		switch argument.Type[0] {
		case OPTION_TYPE_STRING, OPTION_TYPE_INTEGER,
			OPTION_TYPE_STRINGS, OPTION_TYPE_INTEGERS:
		default:
			panic("Argument has unsupported type.")
		}
		validate_enum(Option_Name(argument.Label), Option_Enum{
			Type: argument.Type, Enumeration: argument.Enumeration,
			State: argument.State,
		})
		if command_option_label_seen(
			arguments, flags, global_flags, Option_Name(argument.Label),
			Option_Index(argument_index), true,
		) {
			panic("Command argument collides with another option.")
		}
	}
	for flag_index := range flags {
		option_legacy_initialize(&flags[flag_index])
		flag := flags[flag_index]
		validate_flag_label(flag)
		if command_option_label_seen(
			arguments, flags, global_flags, Option_Name(flag.Label),
			Option_Index(flag_index), false,
		) {
			panic("Command flag collides with another option.")
		}
	}
}

func command_option_label_seen(
	arguments Arguments, flags Flags, global_flags Global_Flags, label Option_Name,
	current_index Option_Index, argument Boolean,
) (seen Boolean) {
	defer func() { Boolean_Invariants(seen, "command_option_label_seen.seen") }()
	Arguments_Invariants(arguments, "command_option_label_seen.arguments")
	Flags_Invariants(flags, "command_option_label_seen.flags")
	Global_Flags_Invariants(global_flags, "command_option_label_seen.global_flags")
	Option_Name_Invariants(label, "command_option_label_seen.label")
	Option_Index_Invariants(current_index, "command_option_label_seen.current_index")
	Boolean_Invariants(argument, "command_option_label_seen.argument")
	for _, global_flag := range global_flags {
		if global_flag.Label == Option_Label(label) {
			return true
		}
	}
	for index, prior := range arguments {
		if Option_Index(index) == current_index {
			if argument {
				break
			}
		}
		if prior.Label == Option_Label(label) {
			return true
		}
	}
	if argument {
		return false
	}
	for index, prior := range flags {
		if Option_Index(index) == current_index {
			break
		}
		if prior.Label == Option_Label(label) {
			return true
		}
	}
	return false
}

// Reports whether an option holds a slice value. The slice type is what makes an
// option variadic: an argument that collects positionals, and any option that appends
// on each repeated -label=value.
func option_is_slice(value Option_Type) (is_slice Boolean) {
	defer func() { Boolean_Invariants(is_slice, "option_is_slice.is_slice") }()
	Option_Type_Invariants(value, "option_is_slice.value")
	switch value {
	case OPTION_TYPE_STRINGS, OPTION_TYPE_INTEGERS:
		return true
	}
	return false
}

// Panics when an option's label is empty or carries a character that cannot appear in
// a -label token. Arguments and flags share this rule because both are settable by
// name.
func option_validate_label(option Option) {
	Option_Invariants(option, "option_validate_label.option")
	if option.Label == "" {
		panic("Option has no label.")
	}
	if strings.Contains(strings.Text(option.Label), "_") {
		panic("Option labels cannot contain underscores; use dashes.")
	}
	if strings.Contains(strings.Text(option.Label), " ") {
		panic("Option labels cannot contain spaces.")
	}
}

// Panics when a flag's label is invalid or its value is not a supported scalar type.
func validate_flag_label(flag Option) {
	Option_Invariants(flag, "validate_flag_label.flag")
	option_validate_label(flag)
	switch flag.Type[0] {
	case OPTION_TYPE_STRING, OPTION_TYPE_BOOLEAN, OPTION_TYPE_INTEGER:
	default:
		panic("Flag has unsupported type.")
	}
	validate_enum(Option_Name(flag.Label), Option_Enum{
		Type: flag.Type, Enumeration: flag.Enumeration, State: flag.State,
	})
}

// Panics when an option carries a malformed enum: an Enum whose element type is not
// []string/[]int or does not match Value's type, an empty Enum, an enum on a variadic
// option, or — for a flag, which has a real default — a default Value outside the set.
// A non-enum option (Enum nil) is left untouched. An argument's default is its zero
// value, deliberately not checked for membership: the required check enforces that the
// argument is supplied, and the parse-time enum check enforces membership then.
// Option_Enum is one validated-label option's enum state.
type Option_Enum struct {
	// Type selects the scalar arm constrained by Enumeration.
	Type Option_Type_State
	// Enumeration preserves borrowed members without interface boxing.
	Enumeration Option_Enumeration
	// State supplies the default checked for flag membership.
	State Option_State
}

// Option_Enum_Invariants composes only state used after label validation.
func Option_Enum_Invariants(value Option_Enum, namespace aver.Namespace) {
	Option_Type_State_Invariants(value.Type, namespace)
	Option_Enumeration_Invariants(value.Enumeration, namespace)
	Option_State_Invariants(value.State, namespace)
}

func validate_enum(label Option_Name, option Option_Enum) {
	Option_Name_Invariants(label, "validate_enum.label")
	Option_Enum_Invariants(option, "validate_enum.option")
	if option.Enumeration[0].String == nil {
		if option.Enumeration[0].Integers == nil {
			return
		}
	}
	if option.Enumeration[0].String != nil {
		if option.Enumeration[0].Integers != nil {
			panic("Option has two enum types.")
		}
	}
	if option_is_slice(option.Type[0]) {
		panic("Variadic option cannot be an enum.")
	}
	if option.Enumeration[0].String != nil {
		if option.Type[0] != OPTION_TYPE_STRING {
			panic("Text enum has a non-text option value.")
		}
		if len(option.Enumeration[0].String) == 0 {
			panic("Option has an empty enum.")
		}
		if option.State[0].Is_Flag {
			if !slices.Contains(
				option.Enumeration[0].String, string(option.State[0].String),
			) {
				panic("Option default is not one of its enum values.")
			}
		}
		return
	}
	if option.Type[0] != OPTION_TYPE_INTEGER {
		panic("Integer enum has a non-integer option value.")
	}
	if len(option.Enumeration[0].Integers) == 0 {
		panic("Option has an empty enum.")
	}
	if option.State[0].Is_Flag {
		if !slices.Contains(
			option.Enumeration[0].Integers, int(option.State[0].Integer),
		) {
			panic("Option default is not one of its enum values.")
		}
	}
}

// Program_Parse parses command-line arguments and returns the active command with
// populated values. The operating_system_args slice should be os.Args from main. If
// no command is specified, the first declared command is used. Every option is
// settable by -label=value and arguments may also be given positionally; the two
// kinds of token may interleave freely. It returns an error for an unknown command,
// an unknown option, a scalar set more than once, a positional with no argument to
// fill, a missing required argument, or an invalid or absent value.
func program_parse_arguments(
	selection *Program_Selection, operating_system_args Process_Arguments,
	workspace *Workspace,
) (active_command Command, err error) {
	defer func() {
		Command_Invariants(active_command, "program_parse_arguments.active_command")
	}()
	Program_Selection_Invariants(*selection, "program_parse_arguments.selection")
	Workspace_Invariants(*workspace, "program_parse_arguments.workspace")
	Process_Arguments_Invariants(
		operating_system_args, "program_parse_arguments.operating_system_args",
	)
	if len(operating_system_args) == 0 {
		panic("Program_Parse needs at least one os_arg.")
	}
	arguments := Parse_Arguments(operating_system_args)

	// -help short-circuits before command resolution, so it works even when the command
	// slot holds -help itself or a required argument is missing. The caller renders the
	// resolved context with Print_Requested_Help and exits successfully.
	if program_wants_help(Parsed_Tokens(arguments[1:])) {
		return program_parse_arguments_finish(
			program_help_context(selection, Help_Arguments(arguments)), Help_Requested)
	}

	active_command, arguments_start, err := program_resolve_command(
		selection, arguments, workspace)
	if err != nil {
		return program_parse_arguments_finish(active_command, err)
	}
	// Nothing follows the consumed prefix (the program name, plus the command name in
	// multi-command mode), so the default command stands as-is.
	if len(arguments) <= int(arguments_start) {
		if len(active_command.Arguments) == 0 {
			active_command.Deprecation_Warnings = Resolved_Deprecation_Warnings(
				workspace.Deprecation_Warnings[:0],
			)
			return program_parse_arguments_finish(active_command, nil)
		}
	}

	tokens := Parsed_Tokens(arguments[arguments_start:])
	filled, positionals, slice_named, err := program_assign_named(
		selection[0].Global_Flags, selection[0].Help_Flags,
		active_command.Arguments, active_command.Flags, tokens, workspace)
	if err != nil {
		return program_parse_arguments_finish(active_command, err)
	}
	err = command_assign_positionals(&Command_Assign_Positionals_Input{
		Command:        active_command,
		Positionals:    positionals,
		Slice_Named:    slice_named,
		Filled:         filled,
		String_Values:  workspace.String_Values,
		Integer_Values: workspace.Integer_Values,
		Failure:        Failure_Reference(workspace.Failure),
	})
	if err != nil {
		return program_parse_arguments_finish(active_command, err)
	}
	err = command_validate_required(
		active_command, Assigned_Options(filled),
		Failure_Reference(workspace.Failure),
	)
	if err != nil {
		return program_parse_arguments_finish(active_command, err)
	}
	active_command.Deprecation_Warnings = command_deprecation_warnings(
		selection[0].Global_Flags, active_command, Assigned_Options(filled),
		workspace.Deprecation_Warnings,
	)
	return program_parse_arguments_finish(active_command, nil)
}

// Deprecated declarations remain silent unless this invocation selected them.
func command_deprecation_warnings(
	global_flags Global_Flags, command Command,
	filled Assigned_Options, storage Deprecation_Warnings,
) (warnings Resolved_Deprecation_Warnings) {
	defer func() {
		Resolved_Deprecation_Warnings_Invariants(
			warnings, "command_deprecation_warnings.warnings",
		)
	}()
	Global_Flags_Invariants(global_flags, "command_deprecation_warnings.global_flags")
	Command_Invariants(command, "command_deprecation_warnings.command")
	Assigned_Options_Invariants(filled, "command_deprecation_warnings.filled")
	Deprecation_Warnings_Invariants(storage, "command_deprecation_warnings.storage")
	warnings = Resolved_Deprecation_Warnings(storage[:0])
	if !program_has_deprecations(global_flags, command.Deprecated, command.Flags) {
		return warnings
	}
	return Resolved_Deprecation_Warnings(collect_deprecations(
		global_flags, command.Label, command.Deprecated,
		command.Arguments, command.Flags, filled, storage,
	))
}

func program_parse_arguments_finish(
	command Command, err error,
) (finished Command, finished_err error) {
	defer func() { Command_Invariants(finished, "program_parse_arguments_finish.finished") }()
	Command_Invariants(command, "program_parse_arguments_finish.command")
	argument_count := len(command.Arguments)
	flag_count := len(command.Flags)
	aver.Sometimes(argument_count > 0, "command has positional arguments")
	aver.Sometimes(flag_count > 0, "command has flags")
	return command, err
}

func program_has_deprecations(
	global_flags Global_Flags, command_deprecated Deprecation, command_flags Flags,
) (deprecated Boolean) {
	defer func() { Boolean_Invariants(deprecated, "program_has_deprecations.deprecated") }()
	Global_Flags_Invariants(global_flags, "program_has_deprecations.global_flags")
	Deprecation_Invariants(command_deprecated, "program_has_deprecations.command_deprecated")
	Flags_Invariants(command_flags, "program_has_deprecations.command_flags")
	if command_deprecated != "" {
		return true
	}
	for _, flag := range command_flags {
		if flag.State[0].Deprecated != "" {
			return true
		}
	}
	for _, flag := range global_flags {
		if flag.State[0].Deprecated != "" {
			return true
		}
	}
	return false
}

// Gathers a warning for each deprecated command or flag the invocation used: the
// resolved command when it is itself deprecated (invoking it is using it), and every
// deprecated command flag or global flag that was set by name (present in filled).
func collect_deprecations(
	global_flags Global_Flags, command_label Label,
	command_deprecated Deprecation, arguments Arguments, flags Flags,
	filled Assigned_Options, storage Deprecation_Warnings,
) (warnings Deprecation_Warnings) {
	defer func() {
		Deprecation_Warnings_Invariants(warnings, "collect_deprecations.warnings")
	}()
	Global_Flags_Invariants(global_flags, "collect_deprecations.global_flags")
	Label_Invariants(command_label, "collect_deprecations.command_label")
	Deprecation_Invariants(command_deprecated, "collect_deprecations.command_deprecated")
	Arguments_Invariants(arguments, "collect_deprecations.command_arguments")
	Flags_Invariants(flags, "collect_deprecations.command_flags")
	Assigned_Options_Invariants(filled, "collect_deprecations.filled")
	Deprecation_Warnings_Invariants(storage, "collect_deprecations.storage")
	warnings = storage[:0]
	if command_deprecated != "" {
		warnings = warning_store(Writable_Deprecation_Warnings(warnings), Warning{
			Kind:     Warning_Kind{WARNING_KIND_COMMAND},
			Name:     Warning_Name{Value_Text(command_label)},
			Guidance: Warning_Guidance{command_deprecated},
		})
	}
	warnings = deprecated_flags_used(
		warnings, Options(flags), filled, slices.Position(len(arguments)),
	)
	warnings = deprecated_flags_used(warnings,
		Options(global_flags), filled,
		slices.Position(len(arguments)+len(flags)),
	)
	return warnings
}

// Gathers a warning for each deprecated flag that was set by name.
func deprecated_flags_used(
	collected Deprecation_Warnings, flags Options,
	filled Assigned_Options, filled_offset slices.Position,
) (warnings Deprecation_Warnings) {
	defer func() {
		Deprecation_Warnings_Invariants(warnings, "deprecated_flags_used.warnings")
	}()
	Options_Invariants(flags, "deprecated_flags_used.flags")
	Assigned_Options_Invariants(filled, "deprecated_flags_used.filled")
	slices.Position_Invariants(filled_offset, "deprecated_flags_used.filled_offset")
	Deprecation_Warnings_Invariants(collected, "deprecated_flags_used.storage")
	warnings = collected
	for index := range flags {
		flag := flags[index]
		if flag.State[0].Deprecated == "" {
			continue
		}
		filled_index := int(filled_offset) + index
		if !filled[filled_index] {
			continue
		}
		warnings = warning_store(
			Writable_Deprecation_Warnings(warnings),
			Warning{
				Kind:     Warning_Kind{WARNING_KIND_FLAG},
				Name:     Warning_Name{Value_Text(flag.Label)},
				Guidance: Warning_Guidance{flag.State[0].Deprecated},
			},
		)
	}
	return warnings
}

func warning_store(
	warnings Writable_Deprecation_Warnings, warning Warning,
) (stored Deprecation_Warnings) {
	defer func() { Deprecation_Warnings_Invariants(stored, "warning_store.stored") }()
	Writable_Deprecation_Warnings_Invariants(
		warnings, "warning_store.warnings",
	)
	Warning_Invariants(warning, "warning_store.warning")
	if len(warnings) == cap(warnings) {
		panic("Program_Parse Deprecation_Warnings storage is too small.")
	}
	warning_count := len(warnings)
	stored = Deprecation_Warnings(warnings[:warning_count+1])
	stored[warning_count] = warning
	return stored
}

// Help has priority over command resolution because help must work for incomplete input.
func program_wants_help(tokens Parsed_Tokens) (wants Boolean) {
	defer func() { Boolean_Invariants(wants, "program_wants_help.wants") }()
	Parsed_Tokens_Invariants(tokens, "program_wants_help.tokens")
	for _, token := range tokens {
		switch token {
		case "-" + HELP_SHORT_LABEL, "-" + HELP_LABEL:
			return true
		}
	}
	return false
}

// Resolves the command a -help request refers to, for Print_Requested_Help. A multicall
// program uses the argv[0] verb; a multi-command program uses the slot-1 command when it
// names one. Anything else — a single-command program, an unknown or absent command —
// yields the zero-value root context (empty Label), which renders the whole program.
func program_help_context(
	selection *Program_Selection, operating_system_args Help_Arguments,
) (context Command) {
	defer func() { Command_Invariants(context, "program_help_context.context") }()
	Program_Selection_Invariants(*selection, "program_help_context.selection")
	Help_Arguments_Invariants(operating_system_args, "program_help_context.arguments")
	if selection[0].Mode[0] == PROGRAM_MODE_MULTICALL {
		name := Label(path.Base(operating_system_args[0]))
		index, _, found := program_select_command(selection[0].Commands, name)
		// A self-invoked multicall binary (run by its own name) names the verb in the
		// first token, so -help there resolves that verb, not the root.
		if !found {
			if len(operating_system_args) > 1 {
				token := Label(operating_system_args[1])
				index, _, found = program_select_command(
					selection[0].Commands, token,
				)
			}
		}
		if !found {
			return Command{}
		}
		return selection[0].Commands[index]
	}
	if selection[0].Mode[0] == PROGRAM_MODE_SINGLE {
		return Command{}
	}
	if len(operating_system_args) > 1 {
		index, _, found := program_select_command(
			selection[0].Commands, Label(operating_system_args[1]),
		)
		if !found {
			return Command{}
		}
		return selection[0].Commands[index]
	}
	return Command{}
}

// One command-line token paired with its position, so a slice argument can reassemble
// its positional and -label=value contributions in the order they were written.
type Indexed_Token struct {
	// Index is the token's position in the original argument list.
	Index Token_Index
	// Value is the token's text as written on the command line.
	Value Value_Text
}

// Indexed_Token_Invariants composes bounded token position and text.
func Indexed_Token_Invariants(value Indexed_Token, namespace aver.Namespace) {
	Token_Index_Invariants(value.Index, namespace)
	Value_Text_Invariants(value.Value, namespace)
}

// Reports whether a token sets an option by name. A lone "-" is a positional value.
func is_named_token(token Value_Text) (named Boolean) {
	defer func() { Boolean_Invariants(named, "is_named_token.named") }()
	Value_Text_Invariants(token, "is_named_token.token")
	if !strings.Has_Prefix(strings.Text(token), "-") {
		return false
	}
	return token != "-"
}

// Splits a -label=value token into its parts, rejecting the double-dash form.
func parse_named_token(token Named_Token, failure Failure_Reference) (
	label Option_Label, value Assigned_Value, value_was_set Boolean, err error,
) {
	defer func() {
		Option_Label_Invariants(label, "parse_named_token.label")
		Assigned_Value_Invariants(value, "parse_named_token.value")
		Boolean_Invariants(value_was_set, "parse_named_token.value_was_set")
	}()
	Named_Token_Invariants(token, "parse_named_token.token")
	Failure_Reference_Invariants(failure, "parse_named_token.failure")
	if strings.Has_Prefix(strings.Text(token), "--") {
		if len(token) > len("--") {
			failure_reset(failure)
			failure_write(failure, Failure_Fragment{"use a single dash: -"})
			failure_write(failure, Failure_Fragment{string(token[2:])})
			failure_write(failure, Failure_Fragment{", not --"})
			failure_write(failure, Failure_Fragment{string(token[2:])})
			return "", "", false, failure[0]
		}
	}
	// A lone "-" was already excluded by is_named_token.
	aver.Always(token != "-", "named token is not a lone dash")
	label_text, value_text, found := strings.Cut(strings.Text(token[1:]), "=")
	label = Option_Label(label_text)
	value = Assigned_Value(value_text)
	value_was_set = Boolean(found)
	return label, value, value_was_set, nil
}

// Applies every -label=value token to its option and returns the bare positionals
// plus, for the slice argument, the values given by name — both tagged with their
// position so the slice preserves command-line order. filled records the scalar
// options set by name: a scalar named twice is an error, and a named scalar argument
// is skipped by the positional pass.
func program_assign_named(
	global_flags Global_Flags, help_flags Help_Flags,
	arguments Arguments, flags Flags, tokens Parsed_Tokens, workspace *Workspace,
) (filled Filled_Options, positionals Parsed_Positionals,
	slice_named Parsed_Variadic_Tokens, err error,
) {
	defer func() {
		Filled_Options_Invariants(filled, "program_assign_named.filled")
		Parsed_Positionals_Invariants(positionals, "program_assign_named.positionals")
		Parsed_Variadic_Tokens_Invariants(slice_named, "program_assign_named.slice_named")
	}()
	Global_Flags_Invariants(global_flags, "program_assign_named.global_flags")
	Help_Flags_Invariants(help_flags, "program_assign_named.help_flags")
	Arguments_Invariants(arguments, "program_assign_named.command_arguments")
	Flags_Invariants(flags, "program_assign_named.command_flags")
	Workspace_Invariants(*workspace, "program_assign_named.workspace")
	Parsed_Tokens_Invariants(tokens, "program_assign_named.tokens")
	filled = program_assignment_storage(arguments, flags, global_flags, workspace)
	positionals = Parsed_Positionals(workspace.Positionals[:0])
	slice_named = Parsed_Variadic_Tokens(workspace.Slice_Named[:0])
	for index, token := range tokens {
		if !is_named_token(Value_Text(token)) {
			if len(positionals) == cap(workspace.Positionals) {
				panic("Program_Parse Positionals storage is too small.")
			}
			position_count := len(positionals)
			positionals = Parsed_Positionals(workspace.Positionals[:position_count+1])
			positionals[position_count] = Indexed_Token{
				Index: Token_Index(index), Value: Value_Text(token),
			}
			continue
		}
		failure := Failure_Reference(workspace.Failure)
		label, value, value_was_set, format_err := parse_named_token(
			Named_Token(token), failure,
		)
		if format_err != nil {
			return filled, positionals, slice_named, format_err
		}
		location, option_index, filled_index, is_slice_argument, find_err :=
			program_find_option(
				global_flags, help_flags, arguments, flags, label,
				Failure_Reference(workspace.Failure),
			)
		if find_err != nil {
			return filled, positionals, slice_named, find_err
		}
		if is_slice_argument {
			if len(slice_named) == cap(workspace.Slice_Named) {
				panic("Program_Parse Slice_Named storage is too small.")
			}
			named_count := len(slice_named)
			slice_named = Parsed_Variadic_Tokens(workspace.Slice_Named[:named_count+1])
			slice_named[named_count] = Indexed_Token{
				Index: Token_Index(index), Value: Value_Text(value),
			}
			continue
		}
		if filled[filled_index] {
			return filled, positionals, slice_named,
				failure_duplicate_option(failure, Failure_Fragment{string(label)})
		}
		set_err := program_assign_named_option(
			global_flags, arguments, flags, Resolved_Option_Location(location),
			Option_Index(option_index), value,
			Equals_Present(value_was_set), Option_Name(label),
			failure,
		)
		if set_err != nil {
			return filled, positionals, slice_named, set_err
		}
		filled[filled_index] = true
	}
	return filled, positionals, slice_named, nil
}

func failure_duplicate_option(
	failure Failure_Reference, label Failure_Fragment,
) (err error) {
	Failure_Reference_Invariants(failure, "failure_duplicate_option.failure")
	Failure_Fragment_Invariants(label, "failure_duplicate_option.label")
	failure_reset(failure)
	failure_write(failure, Failure_Fragment{"-"})
	failure_write(failure, label)
	failure_write(failure, Failure_Fragment{" may only be given once"})
	return failure[0]
}

// Assignment storage resets only the cells used by the active command.
func program_assignment_storage(
	arguments Arguments, flags Flags, global_flags Global_Flags, workspace *Workspace,
) (filled Filled_Options) {
	defer func() {
		Filled_Options_Invariants(filled, "program_assignment_storage.filled")
	}()
	Arguments_Invariants(arguments, "program_assignment_storage.arguments")
	Flags_Invariants(flags, "program_assignment_storage.flags")
	Global_Flags_Invariants(global_flags, "program_assignment_storage.global_flags")
	Workspace_Invariants(*workspace, "program_assignment_storage.workspace")
	filled_count := len(arguments) + len(flags) + len(global_flags)
	if len(workspace.Filled) < filled_count {
		panic("Program_Parse Filled storage is too small.")
	}
	filled = Filled_Options(workspace.Filled[:filled_count])
	for index := range filled {
		filled[index] = false
	}
	return filled
}

// Collection selection stays after successful lookup, so an absent location never indexes.
func program_assign_named_option(
	global_flags Global_Flags, arguments Arguments, flags Flags,
	location Resolved_Option_Location,
	option_index Option_Index, value Assigned_Value,
	value_was_set Equals_Present, name Option_Name, failure Failure_Reference,
) (err error) {
	Global_Flags_Invariants(global_flags, "program_assign_named_option.global_flags")
	Arguments_Invariants(
		arguments, "program_assign_named_option.command_arguments",
	)
	Flags_Invariants(flags, "program_assign_named_option.command_flags")
	Resolved_Option_Location_Invariants(location, "program_assign_named_option.location")
	Option_Index_Invariants(option_index, "program_assign_named_option.option_index")
	Assigned_Value_Invariants(value, "program_assign_named_option.value")
	Equals_Present_Invariants(value_was_set, "program_assign_named_option.value_was_set")
	Option_Name_Invariants(name, "program_assign_named_option.name")
	Failure_Reference_Invariants(failure, "program_assign_named_option.failure")
	var option *Option
	switch location {
	case Resolved_Option_Location(OPTION_LOCATION_ARGUMENT):
		option = &arguments[option_index]
	case Resolved_Option_Location(OPTION_LOCATION_COMMAND_FLAG):
		option = &flags[option_index]
	case Resolved_Option_Location(OPTION_LOCATION_GLOBAL_FLAG):
		option = &global_flags[option_index]
	default:
		panic("A successful option search returned no collection.")
	}
	return option_set_value(Option_Set_Value_Input{
		Option: Option_Reference{option}, Value: value,
		Value_Was_Set: value_was_set, Name: name, Failure: failure,
	})
}

// Finds the option named by a -label token across the command's arguments, then its
// flags, then the program's global flags, returning a pointer into the parse-time copy
// so assignment lands in the right slot. is_slice_argument is true when the match is a
// slice-valued argument, which appends rather than sets. Errors on an unknown label.
func program_find_option(
	global_flags Global_Flags, help_flags Help_Flags,
	arguments Arguments, flags Flags, label Option_Label, failure Failure_Reference,
) (
	location Option_Location, option_index slices.Found_Index,
	filled_index slices.Found_Index, is_slice_argument Boolean, err error,
) {
	defer func() {
		Option_Location_Invariants(location, "program_find_option.location")
		slices.Found_Index_Invariants(option_index, "program_find_option.option_index")
		slices.Found_Index_Invariants(filled_index, "program_find_option.filled_index")
		Boolean_Invariants(is_slice_argument, "program_find_option.is_slice_argument")
	}()
	Global_Flags_Invariants(global_flags, "program_find_option.global_flags")
	Help_Flags_Invariants(help_flags, "program_find_option.help_flags")
	Arguments_Invariants(arguments, "program_find_option.command_arguments")
	Flags_Invariants(flags, "program_find_option.command_flags")
	Option_Label_Invariants(label, "program_find_option.label")
	Failure_Reference_Invariants(failure, "program_find_option.failure")
	for index := range arguments {
		argument := &arguments[index]
		if argument.Label == label {
			return OPTION_LOCATION_ARGUMENT, slices.Found_Index(index),
				slices.Found_Index(index), option_is_slice(argument.Type[0]), nil
		}
	}
	for index := range flags {
		if flags[index].Label == label {
			return OPTION_LOCATION_COMMAND_FLAG, slices.Found_Index(index),
				slices.Found_Index(len(arguments) + index), false, nil
		}
	}
	for index := range global_flags {
		if global_flags[index].Label == label {
			global_filled_index := len(arguments) + len(flags) + index
			return OPTION_LOCATION_GLOBAL_FLAG, slices.Found_Index(index),
				slices.Found_Index(global_filled_index),
				false, nil
		}
	}
	suggestion := closest_option_label(
		global_flags, help_flags, arguments, flags, label,
	)
	if suggestion[0].Found {
		failure_reset(failure)
		failure_write(failure, Failure_Fragment{"unknown option -"})
		failure_write(failure, Failure_Fragment{string(label)})
		failure_write(failure, Failure_Fragment{", did you mean -"})
		failure_write(failure, Failure_Fragment{string(suggestion[0].Match)})
		failure_write(failure, Failure_Fragment{"?"})
		return OPTION_LOCATION_ABSENT, -1, -1, false, failure[0]
	}
	failure_reset(failure)
	failure_write(failure, Failure_Fragment{"unknown option -"})
	failure_write(failure, Failure_Fragment{string(label)})
	return OPTION_LOCATION_ABSENT, -1, -1, false, failure[0]
}

// Reports whether an option appears in help and completion. A hidden or deprecated
// flag still parses; it is only kept out of what the tool advertises.
func option_shown(option Option) (shown Boolean) {
	defer func() { Boolean_Invariants(shown, "option_shown.shown") }()
	Option_Invariants(option, "option_shown.option")
	if option.State[0].Hidden {
		return false
	}
	return option.State[0].Deprecated == ""
}

// Reports whether a command appears in help and completion. A hidden or deprecated
// command still resolves; it is only kept out of what the tool advertises.
func command_shown(command Command) (shown Boolean) {
	defer func() { Boolean_Invariants(shown, "command_shown.shown") }()
	Command_Invariants(command, "command_shown.command")
	if command.Hidden {
		return false
	}
	return command.Deprecated == ""
}

// SUGGESTION_STATE_COUNT follows one nearest-label accumulator.
const SUGGESTION_STATE_COUNT = len("state") / len("state")

// Suggestion_State keeps optional match and distance in one union cell.
type Suggestion_State [SUGGESTION_STATE_COUNT]struct {
	Match Label
	Found Boolean
	Best  levenshtein.Distance_Value
}

// Suggestion_State_Invariants bounds union storage without impossible branch coverage.
func Suggestion_State_Invariants(value Suggestion_State, _ aver.Namespace) {
	aver.Always(
		len(value[0].Match) <= strings.TEXT_SIZE_MAXIMUM,
		"Suggested label stays inside text capacity.",
	)
	aver.Always(
		int(value[0].Best) >= strings.TEXT_SIZE_MINIMUM,
		"Suggestion distance is not negative.",
	)
	aver.Always(
		int(value[0].Best) <= strings.TEXT_SIZE_MAXIMUM,
		"Suggestion distance stays inside text capacity.",
	)
}

// Updates one closest-label search without assembling a temporary candidate slice.
func closest_label(
	workspace *levenshtein.Workspace, target Label, candidate Label,
	state Suggestion_State,
) (next Suggestion_State) {
	defer func() { Suggestion_State_Invariants(next, "closest_label.next") }()
	levenshtein.Workspace_Invariants(*workspace, "closest_label.workspace")
	Label_Invariants(target, "closest_label.target")
	Label_Invariants(candidate, "closest_label.candidate")
	Suggestion_State_Invariants(state, "closest_label.state")
	distance, status := levenshtein.Distance(levenshtein.Distance_Input{
		Workspace: workspace,
		From:      levenshtein.From_Text_Unvalidated(target),
		To:        levenshtein.To_Text_Unvalidated(candidate),
	})
	if status != levenshtein.STATUS_OK {
		return state
	}
	threshold := levenshtein.Distance_Value(max(len(target), len(candidate)) / 3)
	if distance > threshold {
		return state
	}
	if state[0].Found {
		if distance >= state[0].Best {
			return state
		}
	}
	state[0].Match = candidate
	state[0].Found = true
	state[0].Best = distance
	return state
}

// Searches visible option labels directly so suggestion work stays caller-owned.
func closest_option_label(
	global_flags Global_Flags, help_flags Help_Flags,
	arguments Arguments, flags Flags, target Option_Label,
) (state Suggestion_State) {
	defer func() {
		Suggestion_State_Invariants(state, "closest_option_label.state")
	}()
	Global_Flags_Invariants(global_flags, "closest_option_label.global_flags")
	Help_Flags_Invariants(help_flags, "closest_option_label.help_flags")
	Arguments_Invariants(arguments, "closest_option_label.arguments")
	Flags_Invariants(flags, "closest_option_label.flags")
	Option_Label_Invariants(target, "closest_option_label.target")
	var workspace levenshtein.Workspace
	for index := range arguments {
		option := arguments[index]
		state = closest_label(
			&workspace, Label(target), Label(option.Label), state,
		)
	}
	for index := range flags {
		option := flags[index]
		if option_shown(option) {
			state = closest_label(
				&workspace, Label(target), Label(option.Label), state,
			)
		}
	}
	for index := range help_flags {
		option := help_flags[index]
		if option_shown(option) {
			state = closest_label(
				&workspace, Label(target), Label(option.Label), state,
			)
		}
	}
	for index := range global_flags {
		option := global_flags[index]
		if option_shown(option) {
			state = closest_label(
				&workspace, Label(target), Label(option.Label), state,
			)
		}
	}
	return state
}

// Finds the active command named by the args (defaulting to the first command) and
// deep-copies its Arguments and Flags so parsing never mutates the program.
// arguments_start is the index at which positionals and flags begin: after the
// program name and the command name in multi-command mode, after only the program
// name in single-command mode, where the first token is already a positional.
func program_resolve_command(
	selection *Program_Selection, operating_system_args Parse_Arguments,
	workspace *Workspace,
) (active_command Command, arguments_start Parse_Arguments_Start, err error) {
	defer func() {
		Command_Invariants(active_command, "program_resolve_command.active_command")
		Parse_Arguments_Start_Invariants(arguments_start,
			"program_resolve_command.arguments_start")
	}()
	Program_Selection_Invariants(*selection, "program_resolve_command.selection")
	Workspace_Invariants(*workspace, "program_resolve_command.workspace")
	Parse_Arguments_Invariants(operating_system_args, "program_resolve_command.arguments")
	command_index, command_name := Option_Index(0), Label("")
	suggestion, found := Suggestion_State{}, Boolean(true)
	arguments_start = PARSE_ARGUMENTS_START_SELECTED
	if selection[0].Mode[0] == PROGRAM_MODE_MULTICALL {
		arguments_start = PARSE_ARGUMENTS_START_SINGLE
		command_name = Label(path.Base(operating_system_args[0]))
		command_index, suggestion, found = program_select_command(
			selection[0].Commands, command_name,
		)
		// Self-invocation takes its verb from slot 1, so errors name that chosen token.
		if !found {
			if len(operating_system_args) > 1 {
				arguments_start = PARSE_ARGUMENTS_START_SELECTED
				command_name = Label(operating_system_args[1])
				command_index, suggestion, found = program_select_command(
					selection[0].Commands, command_name,
				)
			}
		}
	} else if selection[0].Mode[0] == PROGRAM_MODE_SINGLE {
		arguments_start = PARSE_ARGUMENTS_START_SINGLE
	} else if len(operating_system_args) > 1 {
		command_name = Label(operating_system_args[1])
		command_index, suggestion, found = program_select_command(
			selection[0].Commands, command_name,
		)
	}
	if !found {
		err = failure_unknown_command(
			Failure_Reference(workspace.Failure), command_name, suggestion,
		)
		if selection[0].Mode[0] == PROGRAM_MODE_SINGLE {
			return selection[0].Single_Commands[0], arguments_start, err
		}
		return selection[0].Commands[0], arguments_start, err
	}
	source := selection[0].Single_Commands[0]
	if selection[0].Mode[0] != PROGRAM_MODE_SINGLE {
		source = selection[0].Commands[command_index]
	}
	active_command = source
	if len(workspace.Command_Arguments) < len(source.Arguments) {
		panic("Program_Parse Command_Arguments storage is too small.")
	}
	active_command.Arguments = Arguments(
		workspace.Command_Arguments[:len(source.Arguments)],
	)
	copy(active_command.Arguments, source.Arguments)
	for index := range active_command.Arguments {
		option_legacy_initialize(&active_command.Arguments[index])
	}
	if len(workspace.Command_Flags) < len(source.Flags) {
		panic("Program_Parse Command_Flags storage is too small.")
	}
	active_command.Flags = Flags(workspace.Command_Flags[:len(source.Flags)])
	copy(active_command.Flags, source.Flags)
	for index := range active_command.Flags {
		option_legacy_initialize(&active_command.Flags[index])
	}
	return active_command, arguments_start, nil
}

// Resolves a command name to its index, suggesting the closest command when the name
// is an unrecognized near-miss. Shared by multicall selection (the argv[0] basename)
// and multi-command selection (the slot-1 token).
func program_select_command(
	commands Commands, name Label,
) (index Option_Index, suggestion Suggestion_State, found Boolean) {
	defer func() {
		Option_Index_Invariants(index, "program_select_command.index")
		Suggestion_State_Invariants(suggestion, "program_select_command.suggestion")
		Boolean_Invariants(found, "program_select_command.found")
	}()
	Commands_Invariants(commands, "program_select_command.commands")
	Label_Invariants(name, "program_select_command.name")
	for candidate_index, command := range commands {
		if command.Label == name {
			return Option_Index(candidate_index), Suggestion_State{}, true
		}
	}
	var workspace levenshtein.Workspace
	for command_index := range commands {
		if !command_shown(commands[command_index]) {
			continue
		}
		suggestion = closest_label(
			&workspace, name, commands[command_index].Label,
			suggestion,
		)
	}
	return 0, suggestion, false
}

func failure_unknown_command(
	failure Failure_Reference, name Label, suggestion Suggestion_State,
) (err error) {
	Failure_Reference_Invariants(failure, "failure_unknown_command.failure")
	Label_Invariants(name, "failure_unknown_command.name")
	Suggestion_State_Invariants(suggestion, "failure_unknown_command.suggestion")
	failure_reset(failure)
	failure_write(failure, Failure_Fragment{"unknown command "})
	failure_write_quoted(failure, Failure_Fragment{string(name)})
	if suggestion[0].Found {
		failure_write(failure, Failure_Fragment{", did you mean "})
		failure_write_quoted(
			failure, Failure_Fragment{string(suggestion[0].Match)},
		)
		failure_write(failure, Failure_Fragment{"?"})
	}
	return failure[0]
}

// Input for command_assign_positionals.
type Command_Assign_Positionals_Input struct {
	// Command is the active command whose arguments are filled in place.
	Command Command
	// Positionals are the bare tokens, each tagged with its command-line position.
	Positionals Parsed_Positionals
	// Slice_Named are the trailing slice argument's -label=value contributions, tagged
	// with position so they merge with the positionals in order.
	Slice_Named Parsed_Variadic_Tokens
	// Filled records the scalar arguments already set by name; it gains those set here.
	Filled Filled_Options
	// String_Values receives parsed string variadic elements.
	String_Values String_Values
	// Integer_Values receives parsed integer variadic elements.
	Integer_Values Integer_Values
	// Failure owns rejected positional diagnostics.
	Failure Failure_Reference
}

// Command_Assign_Positionals_Input_Invariants composes active positional state.
func Command_Assign_Positionals_Input_Invariants(
	value Command_Assign_Positionals_Input, namespace aver.Namespace,
) {
	Command_Invariants(value.Command, namespace)
	Parsed_Positionals_Invariants(value.Positionals, namespace)
	Parsed_Variadic_Tokens_Invariants(value.Slice_Named, namespace)
	Filled_Options_Invariants(value.Filled, namespace)
	String_Values_Invariants(value.String_Values, namespace)
	Integer_Values_Invariants(value.Integer_Values, namespace)
	Failure_Reference_Invariants(value.Failure, namespace)
}

// Fills the command's scalar arguments from the positional tokens in declaration
// order, skipping any already set by name, then routes the rest into the trailing
// slice argument together with its named contributions, in command-line order. Errors
// when a positional has no argument to fill. Filled gains every scalar argument set
// here so the required check can see it.
func command_assign_positionals(input *Command_Assign_Positionals_Input) (err error) {
	Command_Assign_Positionals_Input_Invariants(
		*input, "command_assign_positionals.input",
	)
	command := input.Command
	slice_index := command_slice_argument_index(command)
	slice_contributions := input.Slice_Named
	argument_cursor := 0
	for _, positional := range input.Positionals {
		for argument_cursor < len(command.Arguments) {
			if option_is_slice(command.Arguments[argument_cursor].Type[0]) {
				break
			}
			if !input.Filled[argument_cursor] {
				break
			}
			argument_cursor++
		}
		if argument_cursor < len(command.Arguments) {
			if argument_cursor != int(slice_index) {
				argument := &command.Arguments[argument_cursor]
				set_err := option_set_positional(
					Option_Reference{argument}, positional.Value, input.Failure,
				)
				if set_err != nil {
					return set_err
				}
				input.Filled[argument_cursor] = true
				argument_cursor++
				continue
			}
		}
		if slice_index < 0 {
			failure_reset(input.Failure)
			failure_write(input.Failure, Failure_Fragment{"too many arguments: "})
			failure_write_quoted(
				input.Failure, Failure_Fragment{string(positional.Value)},
			)
			failure_write(input.Failure, Failure_Fragment{" was not expected"})
			return input.Failure[0]
		}
		if len(slice_contributions) == cap(slice_contributions) {
			panic("Program_Parse Slice_Named storage is too small.")
		}
		contribution_count := len(slice_contributions)
		slice_contributions = slice_contributions[:contribution_count+1]
		slice_contributions[contribution_count] = positional
	}
	if slice_index < 0 {
		return nil
	}
	slices.Sort_Function(slice_contributions, func(
		left Indexed_Token, right Indexed_Token,
	) (order slices.Comparison) {
		return slices.Comparison(left.Index - right.Index)
	})
	return option_set_slice(
		Option_Reference{&command.Arguments[slice_index]}, slice_contributions,
		input.String_Values, input.Integer_Values, input.Failure,
	)
}

// Returns the index of the command's trailing slice argument, or -1 when the last
// argument is not a slice.
func command_slice_argument_index(command Command) (index slices.Found_Index) {
	defer func() {
		slices.Found_Index_Invariants(index, "command_slice_argument_index.index")
	}()
	Command_Invariants(command, "command_slice_argument_index.command")
	count := len(command.Arguments)
	if count == 0 {
		return -1
	}
	if option_is_slice(command.Arguments[count-1].Type[0]) {
		return slices.Found_Index(count - 1)
	}
	return -1
}

// Returns an error naming the first scalar argument left unset by both name and
// position. The slice argument is optional, so it is never required.
func command_validate_required(
	command Command, filled Assigned_Options, failure Failure_Reference,
) (err error) {
	Command_Invariants(command, "command_validate_required.command")
	Assigned_Options_Invariants(filled, "command_validate_required.filled")
	Failure_Reference_Invariants(failure, "command_validate_required.failure")
	for index := range command.Arguments {
		argument := command.Arguments[index]
		if option_is_slice(argument.Type[0]) {
			continue
		}
		if filled[index] {
			continue
		}
		failure_reset(failure)
		failure_write(failure, Failure_Fragment{"missing required argument "})
		failure_write_quoted(
			failure, Failure_Fragment{string(argument.Label)},
		)
		failure_write(
			failure, Failure_Fragment{"; pass it by position or as -"},
		)
		failure_write(failure, Failure_Fragment{string(argument.Label)})
		failure_write(failure, Failure_Fragment{"=value"})
		return failure[0]
	}
	return nil
}

// Converts a positional token to the scalar argument's type and assigns it. Unlike a
// named value, a positional is taken verbatim: no quote trimming, and empty is allowed.
func option_set_positional(
	argument_reference Option_Reference, value Value_Text, failure Failure_Reference,
) (err error) {
	Option_Reference_Invariants(argument_reference, "option_set_positional.argument")
	Value_Text_Invariants(value, "option_set_positional.value")
	Failure_Reference_Invariants(failure, "option_set_positional.failure")
	argument := argument_reference[0]
	switch argument.Type[0] {
	default:
		panic("unreachable")
	case OPTION_TYPE_STRING:
		argument.State[0].String = value
		argument.State[0].Parsed = true
	case OPTION_TYPE_INTEGER:
		number, parse_err := strconv.Parse_Decimal(strconv.Text(value))
		if parse_err != nil {
			return failure_expected_integer(
				failure, Failure_Fragment{string(argument.Label)},
				Failure_Fragment{string(value)},
			)
		}
		argument.State[0].Integer = Integer(number)
		argument.State[0].Parsed = true
	}
	return option_check_enum(
		argument_reference,
		Display_Name{Name: Display_Option_Name{Option_Name(argument.Label)}}, failure,
	)
}

// Builds the slice option's value from its contributions, already sorted into
// command-line order, converting each element to the slice's element type.
func option_set_slice(
	option_reference Option_Reference, contributions Parsed_Variadic_Tokens,
	string_values String_Values, integer_values Integer_Values,
	failure Failure_Reference,
) (err error) {
	Option_Reference_Invariants(option_reference, "option_set_slice.option")
	Parsed_Variadic_Tokens_Invariants(contributions, "option_set_slice.contributions")
	String_Values_Invariants(string_values, "option_set_slice.string_values")
	Integer_Values_Invariants(integer_values, "option_set_slice.integer_values")
	Failure_Reference_Invariants(failure, "option_set_slice.failure")
	option := option_reference[0]
	switch option.Type[0] {
	default:
		panic("unreachable")
	case OPTION_TYPE_STRINGS:
		if len(string_values) < len(contributions) {
			panic("Program_Parse String_Values storage is too small.")
		}
		values := string_values[:len(contributions)]
		for index, contribution := range contributions {
			values[index] = string(contribution.Value)
		}
		option.State[0].Strings = values
		option.State[0].Parsed = true
	case OPTION_TYPE_INTEGERS:
		if len(integer_values) < len(contributions) {
			panic("Program_Parse Integer_Values storage is too small.")
		}
		values := integer_values[:len(contributions)]
		for index, contribution := range contributions {
			number, parse_err := strconv.Parse_Decimal(strconv.Text(contribution.Value))
			if parse_err != nil {
				return failure_expected_integer(
					failure, Failure_Fragment{string(option.Label)},
					Failure_Fragment{string(contribution.Value)},
				)
			}
			values[index] = int(number)
		}
		option.State[0].Integers = values
		option.State[0].Parsed = true
	}
	return nil
}

// Input for option_set_value.
type Option_Set_Value_Input struct {
	// Option is the flag whose Value is assigned in place.
	Option Option_Reference
	// Value is the raw value text following '='.
	Value Assigned_Value
	// Value_Was_Set reports whether '=' appeared in the flag.
	Value_Was_Set Equals_Present
	// Name is the flag name, for error messages.
	Name Option_Name
	// Failure owns rejected assignment diagnostics.
	Failure Failure_Reference
}

// Option_Set_Value_Input_Invariants composes one named assignment.
func Option_Set_Value_Input_Invariants(
	value Option_Set_Value_Input, namespace aver.Namespace,
) {
	Option_Reference_Invariants(value.Option, namespace)
	Assigned_Value_Invariants(value.Value, namespace)
	Equals_Present_Invariants(value.Value_Was_Set, namespace)
	Option_Name_Invariants(value.Name, namespace)
	Failure_Reference_Invariants(value.Failure, namespace)
}

// Validates and assigns a parsed flag value by the option's declared type.
// Non-boolean flags require a non-empty value.
func option_set_value(input Option_Set_Value_Input) (err error) {
	Option_Set_Value_Input_Invariants(input, "option_set_value.input")
	option := input.Option[0]
	is_boolean := option.Type[0] == OPTION_TYPE_BOOLEAN
	if !is_boolean {
		absent := !input.Value_Was_Set
		if input.Value == "" {
			absent = true
		}
		if absent {
			failure_reset(input.Failure)
			failure_write(input.Failure, Failure_Fragment{"-"})
			failure_write(input.Failure, Failure_Fragment{string(input.Name)})
			failure_write(
				input.Failure, Failure_Fragment{" needs a value, e.g. -"},
			)
			failure_write(input.Failure, Failure_Fragment{string(input.Name)})
			failure_write(input.Failure, Failure_Fragment{"=value"})
			return input.Failure[0]
		}
	}
	switch option.Type[0] {
	case OPTION_TYPE_BOOLEAN:
		option.State[0].Boolean = true
		option.State[0].Parsed = true
	case OPTION_TYPE_STRING:
		option.State[0].String = Value_Text(trim_quotes(Named_String_Value(input.Value)))
		option.State[0].Parsed = true
	case OPTION_TYPE_INTEGER:
		number, parse_err := strconv.Parse_Decimal(strconv.Text(input.Value))
		if parse_err != nil {
			return failure_expected_integer(
				input.Failure, Failure_Fragment{string(input.Name)},
				Failure_Fragment{string(input.Value)},
			)
		}
		option.State[0].Integer = Integer(number)
		option.State[0].Parsed = true
	}
	if option.Enumeration[0].String == nil {
		if option.Enumeration[0].Integers == nil {
			return nil
		}
	}
	return option_check_enum(
		input.Option,
		Display_Name{Name: Display_Option_Name{input.Name}, Dashed: true},
		input.Failure,
	)
}

// Returns an error when a parsed value falls outside an enum option's permitted set, and
// nil when the option is not an enum or the value is a member. For a string enum it
// suggests the closest member on a near miss, mirroring the did-you-mean the parser
// already gives for unknown options; otherwise, and for an int enum, it lists the whole
// set. display_name is the option as the user wrote it — a -flag or a bare argument
// label — for the message.
func option_check_enum(
	option_reference Option_Reference, display_name Display_Name,
	failure Failure_Reference,
) (err error) {
	Option_Reference_Invariants(option_reference, "option_check_enum.option")
	Display_Name_Invariants(display_name, "option_check_enum.display_name")
	Failure_Reference_Invariants(failure, "option_check_enum.failure")
	option := option_reference[0]
	if option.Enumeration[0].String == nil {
		if option.Enumeration[0].Integers == nil {
			return nil
		}
	}
	if option.Enumeration[0].String != nil {
		enum := option.Enumeration[0].String
		value := option.State[0].String
		if slices.Contains(enum, string(value)) {
			return nil
		}
		var workspace levenshtein.Workspace
		match, ok, _ := levenshtein.Closest(levenshtein.Closest_Input{
			Workspace:  &workspace,
			Target:     levenshtein.Target_Text_Unvalidated(value),
			Candidates: levenshtein.Candidates_Unvalidated(enum),
		})
		failure_reset(failure)
		failure_write(failure, Failure_Fragment{"invalid value "})
		failure_write_quoted(failure, Failure_Fragment{string(value)})
		failure_write(failure, Failure_Fragment{" for "})
		failure_write_display_name(failure, display_name)
		if ok {
			failure_write(failure, Failure_Fragment{", did you mean "})
			failure_write_quoted(failure, Failure_Fragment{string(match)})
			failure_write(failure, Failure_Fragment{"?"})
			return failure[0]
		}
		failure_write(failure, Failure_Fragment{"; allowed: "})
		failure_write_string_set(failure, Failure_Strings{enum})
		return failure[0]
	}
	value := option.State[0].Integer
	if slices.Contains(option.Enumeration[0].Integers, int(value)) {
		return nil
	}
	failure_reset(failure)
	failure_write(failure, Failure_Fragment{"invalid value "})
	failure_write_decimal(failure, Failure_Integer{int(value)})
	failure_write(failure, Failure_Fragment{" for "})
	failure_write_display_name(failure, display_name)
	failure_write(failure, Failure_Fragment{"; allowed: "})
	failure_write_integer_set(
		failure, Failure_Integers{option.Enumeration[0].Integers},
	)
	return failure[0]
}

func failure_write_display_name(failure Failure_Reference, name Display_Name) {
	Failure_Reference_Invariants(failure, "failure_write_display_name.failure")
	Display_Name_Invariants(name, "failure_write_display_name.name")
	if name.Dashed {
		failure_write(failure, Failure_Fragment{"-"})
	}
	failure_write(failure, Failure_Fragment{string(name.Name[0])})
}

// Removes a matching pair of surrounding double or single quotes from a string
// value, leaving other strings untouched.
func trim_quotes(text Named_String_Value) (output Trimmed_Value) {
	defer func() { Trimmed_Value_Invariants(output, "trim_quotes.output") }()
	Named_String_Value_Invariants(text, "trim_quotes.text")
	if len(text) < 2 {
		return Trimmed_Value(text)
	}
	first := text[0]
	last := text[len(text)-1]
	if first == '"' {
		if last == '"' {
			return Trimmed_Value(text[1 : len(text)-1])
		}
	}
	if first == '\'' {
		if last == '\'' {
			return Trimmed_Value(text[1 : len(text)-1])
		}
	}
	return Trimmed_Value(text)
}

// RENDERED_SIZE_MINIMUM follows empty output or zero padding.
const RENDERED_SIZE_MINIMUM = len("")

// RENDERED_SIZE_MAXIMUM follows caller-owned builder capacity.
const RENDERED_SIZE_MAXIMUM = strings.TEXT_SIZE_MAXIMUM

// RENDERED_SIZE_COUNT follows one derived size cell.
const RENDERED_SIZE_COUNT = len("size") / len("size")

// Rendered_Size counts bytes already proven to fit one Output.
type Rendered_Size [RENDERED_SIZE_COUNT]int

// Rendered_Size_Invariants protects all help-size arithmetic before output writes.
func Rendered_Size_Invariants(value Rendered_Size, _ aver.Namespace) {
	aver.Always(
		len(value) == RENDERED_SIZE_COUNT,
		"Rendered size has one exact derived cell.",
	)
	aver.Always(
		value[0] >= RENDERED_SIZE_MINIMUM,
		"Rendered size cannot be negative.",
	)
	aver.Always(
		value[0] <= RENDERED_SIZE_MAXIMUM,
		"Rendered size stays inside caller-owned output capacity.",
	)
}

// RENDERED_OPTION_COUNT follows one option borrowed by a rendering phase.
const RENDERED_OPTION_COUNT = len("option") / len("option")

// Rendered_Option isolates rendering fields from construction and parse contracts.
type Rendered_Option [RENDERED_OPTION_COUNT]struct {
	// Option is one normalized borrowed declaration copy.
	Option Option
}

// Rendered_Option_Invariants checks only state read while rendering.
func Rendered_Option_Invariants(value Rendered_Option, namespace aver.Namespace) {
	option := value[0].Option
	aver.Always(
		len(option.Label) <= strings.TEXT_SIZE_MAXIMUM,
		"Rendered option label stays inside shared text capacity.",
	)
	aver.Always(
		len(option.Description) <= strings.TEXT_SIZE_MAXIMUM,
		"Rendered option description stays inside shared text capacity.",
	)
	Option_Type_State_Invariants(option.Type, namespace)
	Option_Enumeration_Invariants(option.Enumeration, namespace)
	Option_State_Invariants(option.State, namespace)
}

// RENDERED_INTEGER_COUNT follows one integer borrowed by a rendering phase.
const RENDERED_INTEGER_COUNT = len("integer") / len("integer")

// Rendered_Integer keeps machine-range proof outside phase-specific namespaces.
type Rendered_Integer [RENDERED_INTEGER_COUNT]Integer

// Rendered_Integer_Invariants fixes the one machine integer cell.
func Rendered_Integer_Invariants(value Rendered_Integer, _ aver.Namespace) {
	aver.Always(
		len(value) == RENDERED_INTEGER_COUNT,
		"Rendered integer has one exact derived cell.",
	)
}

// RENDERED_EXTERNAL_COUNT follows one environment default rendering cell.
const RENDERED_EXTERNAL_COUNT = len("external") / len("external")

// Rendered_External isolates default fields from declaration validation contracts.
type Rendered_External [RENDERED_EXTERNAL_COUNT]struct {
	// Variable is one normalized borrowed environment declaration copy.
	Variable Environment_Variable
}

// Rendered_External_Invariants checks only state read while rendering a default.
func Rendered_External_Invariants(value Rendered_External, namespace aver.Namespace) {
	variable := value[0].Variable
	External_Type_State_Invariants(variable.Type, namespace)
	External_State_Invariants(variable.State, namespace)
}

func output_write(output Output_Reference, source Value_Text) {
	Output_Reference_Invariants(output, "render_write.output")
	Value_Text_Invariants(source, "render_write.source")
	strings.Builder_Write_Text(&output[0].Builder, strings.Text(source))
}

func output_write_spaces(output Output_Reference, count Rendered_Size) {
	Output_Reference_Invariants(output, "output_write_spaces.output")
	Rendered_Size_Invariants(count, "output_write_spaces.count")
	for range count[0] {
		strings.Builder_Write_Byte(&output[0].Builder, ' ')
	}
}

func rendered_size_max(left Rendered_Size, right Rendered_Size) (result Rendered_Size) {
	defer func() { Rendered_Size_Invariants(result, "rendered_size_max.result") }()
	Rendered_Size_Invariants(left, "rendered_size_max.left")
	Rendered_Size_Invariants(right, "rendered_size_max.right")
	if left[0] > right[0] {
		return left
	}
	return right
}

func decimal_size(rendered Rendered_Integer) (size Rendered_Size) {
	defer func() { Rendered_Size_Invariants(size, "decimal_size.size") }()
	Rendered_Integer_Invariants(rendered, "decimal_size.rendered")
	var storage [strconv.DECIMAL_TEXT_SIZE_MAXIMUM]byte
	return Rendered_Size{int(strconv.Format_Decimal_Into(
		storage[:], strconv.Machine_Integer(rendered[0]),
	))}
}

func output_write_decimal(output Output_Reference, rendered Rendered_Integer) {
	Output_Reference_Invariants(output, "output_write_decimal.output")
	Rendered_Integer_Invariants(rendered, "output_write_decimal.rendered")
	var storage [strconv.DECIMAL_TEXT_SIZE_MAXIMUM]byte
	count := strconv.Format_Decimal_Into(
		storage[:], strconv.Machine_Integer(rendered[0]),
	)
	strings.Builder_Write(&output[0].Builder, storage[:count])
}

func option_enum_size(enumeration Option_Enumeration) (size Rendered_Size) {
	defer func() { Rendered_Size_Invariants(size, "option_enum_size.size") }()
	Option_Enumeration_Invariants(enumeration, "option_enum_size.enumeration")
	if enumeration[0].String != nil {
		for index, member := range enumeration[0].String {
			if index != 0 {
				size[0] += len("|")
			}
			size[0] += len(member)
		}
		return size
	}
	for index, member := range enumeration[0].Integers {
		if index != 0 {
			size[0] += len("|")
		}
		size[0] += decimal_size(Rendered_Integer{Integer(member)})[0]
	}
	return size
}

func option_enum_write(output Output_Reference, enumeration Option_Enumeration) {
	Output_Reference_Invariants(output, "option_enum_write.output")
	Option_Enumeration_Invariants(enumeration, "option_enum_write.enumeration")
	if enumeration[0].String != nil {
		for index, member := range enumeration[0].String {
			if index != 0 {
				output_write(output, "|")
			}
			output_write(output, Value_Text(member))
		}
		return
	}
	for index, member := range enumeration[0].Integers {
		if index != 0 {
			output_write(output, "|")
		}
		output_write_decimal(output, Rendered_Integer{Integer(member)})
	}
}

func option_signature_write(output Output_Reference, rendered Rendered_Option) {
	Output_Reference_Invariants(output, "option_signature_write.output")
	Rendered_Option_Invariants(rendered, "option_signature_write.rendered")
	argument := rendered[0].Option
	output_write(output, "<")
	output_write(output, Value_Text(argument.Label))
	output_write(output, ": ")
	if option_is_slice(argument.Type[0]) {
		output_write(output, Value_Text(option_type_text(
			Scalar_Option_Type(argument.Type[0]-OPTION_TYPE_STRINGS),
		)))
		output_write(output, "...>")
		return
	}
	if argument.Enumeration[0].String != nil {
		output_write(output, "(")
		option_enum_write(output, argument.Enumeration)
		output_write(output, ")>")
		return
	}
	if argument.Enumeration[0].Integers != nil {
		output_write(output, "(")
		option_enum_write(output, argument.Enumeration)
		output_write(output, ")>")
		return
	}
	output_write(output, Value_Text(option_type_text(
		Scalar_Option_Type(argument.Type[0]),
	)))
	output_write(output, ">")
}

// Print_Help writes a formatted help message to output: the program header,
// usage syntax, global flags, and each command with its arguments and flags.
func Print_Help(output *Output, program Program) {
	Output_Invariants(*output, "print_help.output")
	Program_Invariants(program, "print_help.program")
	reference := Output_Reference{output}
	output_write(reference, Value_Text(program.Label))
	output_write(reference, " ")
	output_write(reference, Value_Text(program.Description))
	output_write(reference, "\n\n")
	if program.Selection[0].Mode[0] == PROGRAM_MODE_SINGLE {
		Print_Command(output, program, program.Selection[0].Single_Commands[0])
		return
	}
	output_write(reference, "Usage:\n    ")
	output_write(reference, Value_Text(program.Label))
	output_write(reference, " <command> <arguments> [-flags[=value]]\n")
	if program_has_arguments(Program_Selection{
		program.Selection[0],
	}) {
		output_write(reference, NAMED_ARGUMENT_LEGEND)
		output_write(reference, "\n")
	}
	output_write(reference, "\n")

	print_help_global_flag_section(
		reference, program.Selection[0].Help_Flags, program.Selection[0].Global_Flags,
	)
	print_external_help(reference, program.Environment_Variables, program.Secrets)
	print_command_catalog(reference, program.Selection)
}

func print_command_catalog(output Output_Reference, selection Program_Selection) {
	Output_Reference_Invariants(output, "print_command_catalog.output")
	Program_Selection_Invariants(selection, "print_command_catalog.selection")
	output_write(output, "Available Commands:\n")
	for _, command := range selection[0].Commands {
		if !command_shown(command) {
			continue
		}
		cell_width := Rendered_Size{}
		default_width := Rendered_Size{}
		has_command_flag := false
		for _, flag := range command.Flags {
			if !option_shown(flag) {
				continue
			}
			has_command_flag = true
			option_legacy_initialize(&flag)
			rendered := Rendered_Option{{Option: flag}}
			cell_width = rendered_size_max(
				cell_width,
				help_flag_cell_size(rendered, INDENT_COMMAND_FLAG, false),
			)
			if flag.Type[0] != OPTION_TYPE_BOOLEAN {
				default_width = rendered_size_max(
					default_width, help_flag_default_size(rendered),
				)
			}
		}
		output_write(output, "    \033[34m")
		output_write(output, Value_Text(command.Label))
		output_write(output, "\033[0m ")
		for _, argument := range command.Arguments {
			option_legacy_initialize(&argument)
			option_signature_write(output, Rendered_Option{{Option: argument}})
			output_write(output, " ")
		}
		output_write(output, Value_Text(command.Description))
		output_write(output, "\n")
		if has_command_flag {
			output_write(output, "\n")
			for _, flag := range command.Flags {
				if option_shown(flag) {
					option_legacy_initialize(&flag)
					print_help_flag(
						output, Rendered_Option{{Option: flag}},
						INDENT_COMMAND_FLAG, false,
						cell_width, default_width,
					)
				}
			}
		}
		output_write(output, "\n")
	}
}

// Print_Requested_Help renders the help a Help_Requested parse asks for: the whole
// program — its command catalog, or a single-command program's own usage — when command
// is the zero-value root context, or one command's usage when a command was selected. It
// is the turnkey response to errors.Is(err, cli.Help_Requested).
func Print_Requested_Help(output *Output, program Program, command Command) {
	Output_Invariants(*output, "print_requested_help.output")
	Program_Invariants(program, "print_requested_help.program")
	Command_Invariants(command, "print_requested_help.command")
	if command.Label == "" {
		Print_Help(output, program)
		return
	}
	Print_Command(output, program, command)
}

// Print_Command writes help for a single command: a usage line carrying the command's
// own name and positional arguments, then its flags and the program's global flags. It
// is the per-verb help a multicall binary shows for the name in argv[0], and the body
// of a single-command program's help.
func Print_Command(output *Output, program Program, command Command) {
	Output_Invariants(*output, "print_command.output")
	Program_Invariants(program, "print_command.program")
	Command_Invariants(command, "print_command.command")
	reference := Output_Reference{output}
	output_write(reference, "Usage:\n    ")
	output_write(reference, Value_Text(command.Label))
	output_write(reference, " ")
	for _, argument := range command.Arguments {
		option_legacy_initialize(&argument)
		option_signature_write(reference, Rendered_Option{{Option: argument}})
		output_write(reference, " ")
	}
	output_write(reference, "[-flags[=value]]\n")
	if len(command.Arguments) > 0 {
		output_write(reference, NAMED_ARGUMENT_LEGEND)
		output_write(reference, "\n")
	}
	print_help_flag_section(reference, FLAG_SECTION_TITLE, Options(command.Flags))
	print_help_global_flag_section(
		reference, program.Selection[0].Help_Flags, program.Selection[0].Global_Flags,
	)
	print_external_help(reference, program.Environment_Variables, program.Secrets)
}

// Print_Deprecations writes a warning line for each deprecated command or flag the
// parse used, gathered by Program_Parse into command.Deprecation_Warnings. A binary
// calls it after a successful parse, before running, so the user is nudged off a
// deprecated option without the option ceasing to work.
func Print_Deprecations(output *Output, command Command) {
	Output_Invariants(*output, "print_deprecations.output")
	Command_Invariants(command, "print_deprecations.command")
	reference := Output_Reference{output}
	for _, warning := range command.Deprecation_Warnings {
		Warning_Invariants(warning, "print_deprecations.warning")
		output_write(reference, "warning: ")
		if warning.Guidance[0] == "" {
			output_write(reference, "\n")
			continue
		}
		switch warning.Kind[0] {
		case WARNING_KIND_COMMAND:
			output_write(reference, "command \"")
			output_write(reference, warning.Name[0])
			output_write(reference, "\"")
		case WARNING_KIND_FLAG:
			output_write(reference, "-")
			output_write(reference, warning.Name[0])
		case WARNING_KIND_ENVIRONMENT:
			output_write(reference, ENVIRONMENT_WARNING_PREFIX)
			output_write(reference, warning.Name[0])
		case WARNING_KIND_SECRET:
			output_write(reference, "secret ")
			output_write(reference, warning.Name[0])
		}
		output_write(reference, " is deprecated: ")
		output_write(reference, Value_Text(warning.Guidance[0]))
		output_write(reference, "\n")
	}
}

// Writes a titled section of flag rows, or nothing when no flag in it is shown.
func print_help_flag_section(
	output Output_Reference, title Section_Title, flags Options,
) {
	Output_Reference_Invariants(output, "print_help_flag_section.output")
	Section_Title_Invariants(title, "print_help_flag_section.title")
	Options_Invariants(flags, "print_help_flag_section.flags")
	has_shown := false
	for _, flag := range flags {
		if option_shown(flag) {
			has_shown = true
			break
		}
	}
	if !has_shown {
		return
	}
	cell_width := Rendered_Size{}
	default_width := Rendered_Size{}
	for _, flag := range flags {
		if !option_shown(flag) {
			continue
		}
		option_legacy_initialize(&flag)
		rendered := Rendered_Option{{Option: flag}}
		cell_width = rendered_size_max(
			cell_width, help_flag_cell_size(rendered, INDENT_FLAG, true),
		)
		if flag.Type[0] != OPTION_TYPE_BOOLEAN {
			default_width = rendered_size_max(
				default_width, help_flag_default_size(rendered),
			)
		}
	}
	output_write(output, "\n")
	output_write(output, Value_Text(title))
	output_write(output, "\n")
	for _, flag := range flags {
		if option_shown(flag) {
			option_legacy_initialize(&flag)
			print_help_flag(
				output, Rendered_Option{{Option: flag}}, INDENT_FLAG, true,
				cell_width, default_width,
			)
		}
	}
}

// Reserved and user flags share one visible section while retaining separate storage.
func print_help_global_flag_section(
	output Output_Reference, help_flags Help_Flags, declared_global_flags Global_Flags,
) {
	Output_Reference_Invariants(output, "print_help_global_flag_section.output")
	Help_Flags_Invariants(help_flags, "print_help_global_flag_section.help_flags")
	Global_Flags_Invariants(
		declared_global_flags, "print_help_global_flag_section.global_flags",
	)
	has_shown := false
	for _, flag := range help_flags {
		has_shown = has_shown || bool(option_shown(flag))
	}
	for _, flag := range declared_global_flags {
		has_shown = has_shown || bool(option_shown(flag))
	}
	if !has_shown {
		return
	}
	cell_width := Rendered_Size{}
	default_width := Rendered_Size{}
	for _, flag := range help_flags {
		if option_shown(flag) {
			option_legacy_initialize(&flag)
			rendered := Rendered_Option{{Option: flag}}
			cell_width = rendered_size_max(
				cell_width, help_flag_cell_size(rendered, INDENT_FLAG, true),
			)
			if flag.Type[0] != OPTION_TYPE_BOOLEAN {
				default_width = rendered_size_max(
					default_width, help_flag_default_size(rendered),
				)
			}
		}
	}
	for _, flag := range declared_global_flags {
		if !option_shown(flag) {
			continue
		}
		option_legacy_initialize(&flag)
		rendered := Rendered_Option{{Option: flag}}
		cell_width = rendered_size_max(
			cell_width, help_flag_cell_size(rendered, INDENT_FLAG, true),
		)
		if flag.Type[0] != OPTION_TYPE_BOOLEAN {
			default_width = rendered_size_max(
				default_width, help_flag_default_size(rendered),
			)
		}
	}
	output_write(output, "Global Flags:\n")
	for _, flag := range help_flags {
		if option_shown(flag) {
			option_legacy_initialize(&flag)
			print_help_flag(
				output, Rendered_Option{{Option: flag}}, INDENT_FLAG, true,
				cell_width, default_width,
			)
		}
	}
	for _, flag := range declared_global_flags {
		if option_shown(flag) {
			option_legacy_initialize(&flag)
			print_help_flag(
				output, Rendered_Option{{Option: flag}}, INDENT_FLAG, true,
				cell_width, default_width,
			)
		}
	}
	output_write(output, "\n\n")
}

// OPTION_TYPE_TEXT_SIZE_MINIMUM follows the shortest scalar spelling.
const OPTION_TYPE_TEXT_SIZE_MINIMUM = len("int")

// OPTION_TYPE_TEXT_SIZE_MAXIMUM follows the longest scalar spelling.
const OPTION_TYPE_TEXT_SIZE_MAXIMUM = len("string")

// OPTION_TYPE_TEXT_BOOLEAN_SIZE follows the middle scalar spelling.
const OPTION_TYPE_TEXT_BOOLEAN_SIZE = len("bool")

// Option_Type_Text is one fixed scalar spelling.
type Option_Type_Text string

// Option_Type_Text_Invariants admits only syntax emitted by option type rendering.
func Option_Type_Text_Invariants(value Option_Type_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Int(
			len(value), OPTION_TYPE_TEXT_SIZE_MINIMUM,
			OPTION_TYPE_TEXT_BOOLEAN_SIZE, OPTION_TYPE_TEXT_SIZE_MAXIMUM,
		).
		Ensure()
}

func option_type_text(value Scalar_Option_Type) (text Option_Type_Text) {
	defer func() { Option_Type_Text_Invariants(text, "option_type_text.text") }()
	Scalar_Option_Type_Invariants(value, "option_type_text.value")
	switch Option_Type(value) {
	case OPTION_TYPE_STRING, OPTION_TYPE_STRINGS:
		return "string"
	case OPTION_TYPE_INTEGER, OPTION_TYPE_INTEGERS:
		return "int"
	case OPTION_TYPE_BOOLEAN:
		return "bool"
	}
	panic("Option type is outside its declared domain.")
}

// The note printed under the usage line when a program has at least one positional
// argument: each may also be supplied by name, not only by position.
const NAMED_ARGUMENT_LEGEND = "    Positional arguments may also be supplied via -key=val syntax."

// Reports whether any of the program's commands declares a positional argument, the
// condition under which the named-argument legend is worth printing.
func program_has_arguments(selection Program_Selection) (has Boolean) {
	defer func() { Boolean_Invariants(has, "program_has_arguments.has") }()
	Program_Selection_Invariants(selection, "program_has_arguments.selection")
	if selection[0].Mode[0] == PROGRAM_MODE_SINGLE {
		return len(selection[0].Single_Commands[0].Arguments) > 0
	}
	for _, command := range selection[0].Commands {
		if len(command.Arguments) > 0 {
			return true
		}
	}
	return false
}

func help_flag_cell_size(
	rendered Rendered_Option, indent Indent, color Boolean,
) (size Rendered_Size) {
	defer func() { Rendered_Size_Invariants(size, "help_flag_cell_size.size") }()
	Rendered_Option_Invariants(rendered, "help_flag_cell_size.rendered")
	Indent_Invariants(indent, "help_flag_cell_size.indent")
	Boolean_Invariants(color, "help_flag_cell_size.color")
	flag := rendered[0].Option
	size[0] = len("    ") + len("-") + len(flag.Label)
	if indent == INDENT_COMMAND_FLAG {
		size[0] += len("    ")
	}
	if color {
		size[0] += len("\033[34m") + len("\033[0m")
	}
	if flag.Type[0] == OPTION_TYPE_BOOLEAN {
		return size
	}
	size[0] += len("=")
	if flag.Enumeration[0].String != nil {
		size[0] += len("()") + option_enum_size(flag.Enumeration)[0]
		return size
	}
	if flag.Enumeration[0].Integers != nil {
		size[0] += len("()") + option_enum_size(flag.Enumeration)[0]
		return size
	}
	size[0] += len(option_type_text(Scalar_Option_Type(flag.Type[0])))
	return size
}

func help_flag_cell_write(
	output Output_Reference, rendered Rendered_Option, indent Indent, color Boolean,
) {
	Output_Reference_Invariants(output, "help_flag_cell_write.output")
	Rendered_Option_Invariants(rendered, "help_flag_cell_write.rendered")
	Indent_Invariants(indent, "help_flag_cell_write.indent")
	Boolean_Invariants(color, "help_flag_cell_write.color")
	flag := rendered[0].Option
	output_write(output, "    ")
	if indent == INDENT_COMMAND_FLAG {
		output_write(output, "    ")
	}
	output_write(output, "-")
	if color {
		output_write(output, "\033[34m")
	}
	output_write(output, Value_Text(flag.Label))
	if color {
		output_write(output, "\033[0m")
	}
	if flag.Type[0] == OPTION_TYPE_BOOLEAN {
		return
	}
	output_write(output, "=")
	if flag.Enumeration[0].String != nil {
		output_write(output, "(")
		option_enum_write(output, flag.Enumeration)
		output_write(output, ")")
		return
	}
	if flag.Enumeration[0].Integers != nil {
		output_write(output, "(")
		option_enum_write(output, flag.Enumeration)
		output_write(output, ")")
		return
	}
	output_write(output, Value_Text(option_type_text(Scalar_Option_Type(flag.Type[0]))))
}

func help_flag_default_size(rendered Rendered_Option) (size Rendered_Size) {
	defer func() { Rendered_Size_Invariants(size, "help_flag_default_size.size") }()
	Rendered_Option_Invariants(rendered, "help_flag_default_size.rendered")
	flag := rendered[0].Option
	size[0] = len("(default: )")
	if flag.Type[0] == OPTION_TYPE_STRING {
		if flag.State[0].String == "" {
			size[0] += len(`""`)
			return size
		}
		size[0] += len(flag.State[0].String)
		return size
	}
	size[0] += decimal_size(Rendered_Integer{flag.State[0].Integer})[0]
	return size
}

func help_flag_default_write(output Output_Reference, rendered Rendered_Option) {
	Output_Reference_Invariants(output, "help_flag_default_write.output")
	Rendered_Option_Invariants(rendered, "help_flag_default_write.rendered")
	flag := rendered[0].Option
	output_write(output, "(default: ")
	if flag.Type[0] == OPTION_TYPE_STRING {
		if flag.State[0].String == "" {
			output_write(output, `""`)
		} else {
			output_write(output, flag.State[0].String)
		}
	} else {
		output_write_decimal(output, Rendered_Integer{flag.State[0].Integer})
	}
	output_write(output, ")")
}

func print_help_flag(
	output Output_Reference, rendered Rendered_Option, indent Indent, color Boolean,
	cell_width Rendered_Size, default_width Rendered_Size,
) {
	Rendered_Size_Invariants(cell_width, "print_help_flag.cell_width")
	Rendered_Size_Invariants(default_width, "print_help_flag.default_width")
	Output_Reference_Invariants(output, "print_help_flag.output")
	Rendered_Option_Invariants(rendered, "print_help_flag.rendered")
	Indent_Invariants(indent, "print_help_flag.indent")
	Boolean_Invariants(color, "print_help_flag.color")
	flag := rendered[0].Option
	cell_size := help_flag_cell_size(rendered, indent, color)
	help_flag_cell_write(output, rendered, indent, color)
	output_write_spaces(output, Rendered_Size{cell_width[0] - cell_size[0]})
	output_write(output, "  ")
	if flag.Type[0] != OPTION_TYPE_BOOLEAN {
		default_size := help_flag_default_size(rendered)
		help_flag_default_write(output, rendered)
		output_write_spaces(output, Rendered_Size{default_width[0] - default_size[0]})
		output_write(output, "  ")
	}
	output_write(output, Value_Text(flag.Description))
	output_write(output, "\n")
}

// Get_Option retrieves an option by label from a slice of options, panicking
// when the label is not found. Typed accessors read its fixed union storage.
func Get_Option[Collection Option_Collection](
	flags Collection, label Option_Label,
) (option Option) {
	defer func() { Option_Invariants(option, "get_option.option") }()
	Options_Invariants(Options(flags), "get_option.flags")
	Option_Label_Invariants(label, "get_option.label")
	for _, flag := range flags {
		if flag.Label == label {
			option_legacy_initialize(&flag)
			return flag
		}
	}
	panic("Option is not declared.")
}

// Option_String returns one text scalar without interface boxing.
func Option_String(option Option) (value Value_Text) {
	defer func() { Value_Text_Invariants(value, "option_string.value") }()
	Option_Invariants(option, "option_string.option")
	if option.Type[0] != OPTION_TYPE_STRING {
		panic("Option_String received a non-string option.")
	}
	return option.State[0].String
}

// Option_Integer returns one integer scalar without interface boxing.
func Option_Integer(option Option) (value Integer) {
	defer func() { Integer_Invariants(value, "option_integer.value") }()
	Option_Invariants(option, "option_integer.option")
	if option.Type[0] != OPTION_TYPE_INTEGER {
		panic("Option_Integer received a non-integer option.")
	}
	return option.State[0].Integer
}

// Option_Boolean returns one Boolean scalar without interface boxing.
func Option_Boolean(option Option) (value Boolean) {
	defer func() { Boolean_Invariants(value, "option_boolean.value") }()
	Option_Invariants(option, "option_boolean.option")
	if option.Type[0] != OPTION_TYPE_BOOLEAN {
		panic("Option_Boolean received a non-Boolean option.")
	}
	return option.State[0].Boolean
}

// Option_Strings returns caller-owned text collection storage.
func Option_Strings(option Option) (value String_Values) {
	defer func() { String_Values_Invariants(value, "option_strings.value") }()
	Option_Invariants(option, "option_strings.option")
	if option.Type[0] != OPTION_TYPE_STRINGS {
		panic("Option_Strings received a non-string collection option.")
	}
	return option.State[0].Strings
}

// Option_Integers returns caller-owned integer collection storage.
func Option_Integers(option Option) (value Integer_Values) {
	defer func() { Integer_Values_Invariants(value, "option_integers.value") }()
	Option_Invariants(option, "option_integers.option")
	if option.Type[0] != OPTION_TYPE_INTEGERS {
		panic("Option_Integers received a non-integer collection option.")
	}
	return option.State[0].Integers
}

// Writes one structured candidate into fixed text storage.
func candidate_write_builder(builder *strings.Builder, candidate Candidate) {
	strings.Builder_Invariants(*builder, "candidate_write_builder.builder")
	Candidate_Invariants(candidate, "candidate_write_builder.candidate")
	for _, segment := range candidate.Segments {
		strings.Builder_Write_Text(builder, strings.Text(segment))
	}
	if candidate.State[0].Has_Integer {
		var storage [strconv.DECIMAL_TEXT_SIZE_MAXIMUM]byte
		count := strconv.Format_Decimal_Into(
			storage[:], strconv.Machine_Integer(candidate.State[0].Integer),
		)
		strings.Builder_Write(builder, storage[:count])
	}
}

// Candidate_Write renders one completion directly into caller-owned output.
func Candidate_Write(output *Output, candidate Candidate) {
	Output_Invariants(*output, "candidate_write.output")
	Candidate_Invariants(candidate, "candidate_write.candidate")
	candidate_write_builder(&output.Builder, candidate)
}

// Candidate_Equal compares rendered candidate text without creating joined text.
func Candidate_Equal(candidate Candidate, expected Value_Text) (equal Boolean) {
	defer func() { Boolean_Invariants(equal, "candidate_equal.equal") }()
	Candidate_Invariants(candidate, "candidate_equal.candidate")
	Value_Text_Invariants(expected, "candidate_equal.expected")
	if candidate_size(candidate) != Candidate_Size(len(expected)) {
		return false
	}
	return candidate_has_prefix(candidate, expected)
}

// Candidate prefix comparison never joins borrowed segments.
func candidate_has_prefix(candidate Candidate, prefix Value_Text) (present Boolean) {
	defer func() { Boolean_Invariants(present, "candidate_has_prefix.present") }()
	Candidate_Invariants(candidate, "candidate_has_prefix.candidate")
	Value_Text_Invariants(prefix, "candidate_has_prefix.prefix")
	if Candidate_Size(len(prefix)) > candidate_size(candidate) {
		return false
	}
	prefix_index := 0
	for _, segment := range candidate.Segments {
		for segment_index := range segment {
			if prefix_index == len(prefix) {
				return true
			}
			if segment[segment_index] != prefix[prefix_index] {
				return false
			}
			prefix_index++
		}
	}
	if candidate.State[0].Has_Integer {
		var storage [strconv.DECIMAL_TEXT_SIZE_MAXIMUM]byte
		count := strconv.Format_Decimal_Into(
			storage[:], strconv.Machine_Integer(candidate.State[0].Integer),
		)
		for integer_index := 0; integer_index < int(count); integer_index++ {
			if prefix_index == len(prefix) {
				return true
			}
			if storage[integer_index] != prefix[prefix_index] {
				return false
			}
			prefix_index++
		}
	}
	return prefix_index == len(prefix)
}

// Candidate storage initializes the next caller-owned cell.
func candidate_store(
	storage Candidate_Storage, initialized Writable_Candidates, candidate Candidate,
) (stored Candidates) {
	defer func() { Candidates_Invariants(stored, "candidate_store.stored") }()
	Candidate_Storage_Invariants(storage, "candidate_store.storage")
	Writable_Candidates_Invariants(initialized, "candidate_store.initialized")
	Candidate_Invariants(candidate, "candidate_store.candidate")
	if len(initialized) == len(storage) {
		panic("Complete Candidates storage is too small.")
	}
	count := len(initialized)
	stored = Candidates(storage[:count+1])
	stored[count] = candidate
	return stored
}

// Candidate filtering keeps only candidates matching the current token.
func candidate_filter_store(
	storage Candidates, initialized Writable_Candidates,
	candidate Candidate, prefix Value_Text,
) (filtered Candidates) {
	defer func() { Candidates_Invariants(filtered, "candidate_filter_store.filtered") }()
	Candidates_Invariants(storage, "candidate_filter_store.storage")
	Writable_Candidates_Invariants(initialized, "candidate_filter_store.initialized")
	Candidate_Invariants(candidate, "candidate_filter_store.candidate")
	Value_Text_Invariants(prefix, "candidate_filter_store.prefix")
	if !candidate_has_prefix(candidate, prefix) {
		return Candidates(initialized)
	}
	if len(storage) == 0 {
		panic("Complete Candidates storage is too small.")
	}
	return candidate_store(
		Candidate_Storage(storage), initialized, candidate,
	)
}

// Complete commands writes visible labels into caller storage.
func complete_commands(
	commands Commands, current Value_Text, storage Candidates,
) (candidates Command_Candidates) {
	defer func() {
		Command_Candidates_Invariants(candidates, "complete_commands.candidates")
	}()
	Commands_Invariants(commands, "complete_commands.commands")
	Value_Text_Invariants(current, "complete_commands.current")
	Candidates_Invariants(storage, "complete_commands.storage")
	candidates = Command_Candidates(storage[:0])
	for index := range commands {
		if !command_shown(commands[index]) {
			continue
		}
		candidate := Candidate{}
		candidate.Segments[0] = string(commands[index].Label)
		candidates = Command_Candidates(candidate_filter_store(
			storage, Writable_Candidates(candidates), candidate, current,
		))
	}
	return candidates
}

// Complete option label conditionally publishes one named option.
func complete_option_label(
	option Option, shown Boolean, current Option_Completion_Prefix,
	storage Candidates, candidates Writable_Candidates,
) (completed Candidates) {
	defer func() {
		Candidates_Invariants(completed, "complete_option_label.completed")
	}()
	Option_Invariants(option, "complete_option_label.option")
	Boolean_Invariants(shown, "complete_option_label.shown")
	Option_Completion_Prefix_Invariants(current, "complete_option_label.current")
	Candidates_Invariants(storage, "complete_option_label.storage")
	Writable_Candidates_Invariants(candidates, "complete_option_label.candidates")
	if !shown {
		return Candidates(candidates)
	}
	candidate := Candidate{}
	candidate.Segments[0] = "-"
	candidate.Segments[1] = string(option.Label)
	return candidate_filter_store(
		storage, candidates, candidate, Value_Text(current),
	)
}

// Complete option labels writes every named option reachable from this command.
func complete_option_labels(
	global_flags Global_Flags, help_flags Help_Flags,
	arguments Arguments, flags Flags,
	current Option_Completion_Prefix, storage Candidates,
) (candidates Candidates) {
	defer func() {
		Candidates_Invariants(candidates, "complete_option_labels.candidates")
	}()
	Global_Flags_Invariants(global_flags, "complete_option_labels.global_flags")
	Help_Flags_Invariants(help_flags, "complete_option_labels.help_flags")
	Arguments_Invariants(arguments, "complete_option_labels.arguments")
	Flags_Invariants(flags, "complete_option_labels.flags")
	Option_Completion_Prefix_Invariants(current, "complete_option_labels.current")
	Candidates_Invariants(storage, "complete_option_labels.storage")
	candidates = storage[:0]
	for index := range arguments {
		candidates = complete_option_label(
			arguments[index], true, current, storage,
			Writable_Candidates(candidates),
		)
	}
	for index := range flags {
		candidates = complete_option_label(
			flags[index], option_shown(flags[index]), current, storage,
			Writable_Candidates(candidates),
		)
	}
	for index := range help_flags {
		candidates = complete_option_label(
			help_flags[index], option_shown(help_flags[index]),
			current, storage, Writable_Candidates(candidates),
		)
	}
	for index := range global_flags {
		candidates = complete_option_label(
			global_flags[index], option_shown(global_flags[index]),
			current, storage, Writable_Candidates(candidates),
		)
	}
	return candidates
}

// Complete enum members writes borrowed string or scalar integer members.
func complete_enum_members(
	option Option, base Candidate, current Value_Text, storage Candidates,
) (candidates Enum_Candidates, is_enum Boolean) {
	defer func() {
		Enum_Candidates_Invariants(candidates, "complete_enum_members.candidates")
		Boolean_Invariants(is_enum, "complete_enum_members.is_enum")
	}()
	Option_Invariants(option, "complete_enum_members.option")
	Candidate_Invariants(base, "complete_enum_members.base")
	Value_Text_Invariants(current, "complete_enum_members.current")
	Candidates_Invariants(storage, "complete_enum_members.storage")
	option_legacy_initialize(&option)
	candidates = Enum_Candidates(storage[:0])
	if option.Enumeration[0].String != nil {
		for _, member := range option.Enumeration[0].String {
			candidate := base
			candidate.Segments[CANDIDATE_SEGMENT_COUNT-1] = member
			candidates = Enum_Candidates(candidate_filter_store(
				storage, Writable_Candidates(candidates), candidate, current,
			))
		}
		return candidates, true
	}
	if option.Enumeration[0].Integers != nil {
		for _, member := range option.Enumeration[0].Integers {
			candidate := base
			candidate.State[0].Integer = Integer(member)
			candidate.State[0].Has_Integer = true
			candidates = Enum_Candidates(candidate_filter_store(
				storage, Writable_Candidates(candidates), candidate, current,
			))
		}
		return candidates, true
	}
	return candidates, false
}

// Complete returns the shell-completion candidates for a partially typed command line.
// words mirrors the argv being completed — words[0] is the program (or, for a multicall
// link, the verb) and the final element is the word under the cursor, possibly empty. It
// reads the live Program, so candidates track command, flag, and enum definitions with no
// separate registration. An empty result means "no cli candidate" — the shell falls back
// to file completion.
func Complete(
	program Program, words Process_Arguments, storage Candidates,
) (candidates Candidates) {
	defer func() { Candidates_Invariants(candidates, "complete.candidates") }()
	Program_Invariants(program, "complete.program")
	Process_Arguments_Invariants(words, "complete.words")
	Candidates_Invariants(storage, "complete.storage")
	if len(words) == 0 {
		return storage[:0]
	}
	current := Value_Text(words[len(words)-1])

	if program.Selection[0].Mode[0] == PROGRAM_MODE_SINGLE {
		return complete_in_command(
			program.Selection[0].Global_Flags, program.Selection[0].Help_Flags,
			program.Selection[0].Single_Commands[0].Arguments,
			program.Selection[0].Single_Commands[0].Flags,
			Completion_Words(words), COMPLETION_ARGUMENTS_START_SINGLE,
			current, storage,
		)
	}
	if len(program.Selection[0].Commands) == 0 {
		panic("Command completion requires one selectable command.")
	}
	if program.Selection[0].Mode[0] == PROGRAM_MODE_MULTICALL {
		index, _, found := program_select_command(
			program.Selection[0].Commands, Label(path.Base(words[0])),
		)
		if found {
			return complete_in_command(
				program.Selection[0].Global_Flags, program.Selection[0].Help_Flags,
				program.Selection[0].Commands[index].Arguments,
				program.Selection[0].Commands[index].Flags,
				Completion_Words(words), COMPLETION_ARGUMENTS_START_SINGLE,
				current, storage,
			)
		}
		// A non-verb argv[0] falls through to self-invocation, where slot 1 selects
		// the command, exactly like a multi-command program.
	}
	// Multi-command (or a self-invoked multicall binary): slot 1 selects the command.
	// While it is still being typed, the candidates are the command names themselves.
	if len(words) <= 2 {
		return Candidates(complete_commands(
			program.Selection[0].Commands, current, storage,
		))
	}
	index, _, found := program_select_command(
		program.Selection[0].Commands, Label(words[1]),
	)
	if !found {
		return storage[:0]
	}
	return complete_in_command(
		program.Selection[0].Global_Flags, program.Selection[0].Help_Flags,
		program.Selection[0].Commands[index].Arguments,
		program.Selection[0].Commands[index].Flags,
		Completion_Words(words), COMPLETION_ARGUMENTS_START_SELECTED, current, storage,
	)
}

// Completes the word under the cursor within a resolved command: an enum option's members
// after -label=, then flag and named-argument labels for a leading dash, then an enum
// positional's members. Anything else yields nil so the shell completes files.
func complete_in_command(
	global_flags Global_Flags, help_flags Help_Flags,
	arguments Arguments, flags Flags, words Completion_Words,
	args_start Completion_Arguments_Start, current Value_Text, storage Candidates,
) (candidates Candidates) {
	defer func() { Candidates_Invariants(candidates, "complete_in_command.candidates") }()
	Global_Flags_Invariants(global_flags, "complete_in_command.global_flags")
	Help_Flags_Invariants(help_flags, "complete_in_command.help_flags")
	Arguments_Invariants(arguments, "complete_in_command.arguments")
	Flags_Invariants(flags, "complete_in_command.flags")
	Completion_Words_Invariants(words, "complete_in_command.words")
	Completion_Arguments_Start_Invariants(args_start, "complete_in_command.args_start")
	Value_Text_Invariants(current, "complete_in_command.current")
	Candidates_Invariants(storage, "complete_in_command.storage")
	if strings.Has_Prefix(strings.Text(current), "-") {
		if strings.Contains(strings.Text(current), "=") {
			return Candidates(complete_enum_value(
				Global_Flags(global_flags), arguments, flags,
				Enum_Completion_Token(current), storage,
			))
		}
		return complete_option_labels(
			Global_Flags(global_flags), help_flags, arguments, flags,
			Option_Completion_Prefix(current), storage,
		)
	}
	position := positional_index(arguments, words, args_start)
	if position < 0 {
		return storage[:0]
	}
	if int(position) >= len(arguments) {
		return storage[:0]
	}
	enum_candidates, is_enum := complete_enum_members(
		arguments[position], Candidate{}, current, storage,
	)
	if is_enum {
		return Candidates(enum_candidates)
	}
	return storage[:0]
}

// Completes a -label=value token to that option's enum members, when label names an
// enum in scope. A non-enum or unknown label yields nil (the shell completes files).
func complete_enum_value(
	global_flags Global_Flags,
	arguments Arguments,
	flags Flags,
	current Enum_Completion_Token,
	storage Candidates,
) (candidates Enum_Candidates) {
	defer func() {
		Enum_Candidates_Invariants(candidates, "complete_enum_value.candidates")
	}()
	Global_Flags_Invariants(global_flags, "complete_enum_value.global_flags")
	Arguments_Invariants(arguments, "complete_enum_value.command_arguments")
	Flags_Invariants(flags, "complete_enum_value.command_flags")
	Enum_Completion_Token_Invariants(current, "complete_enum_value.current")
	Candidates_Invariants(storage, "complete_enum_value.storage")
	equals_offset := int(strings.Index(strings.Text(current), "="))
	label := Completion_Option_Label(current[1:equals_offset])
	location, index, found := command_scope_option(global_flags, arguments, flags, label)
	if !found {
		return Enum_Candidates(storage[:0])
	}
	option := Option{}
	switch location {
	case OPTION_LOCATION_ARGUMENT:
		option = arguments[index]
	case OPTION_LOCATION_COMMAND_FLAG:
		option = flags[index]
	case OPTION_LOCATION_GLOBAL_FLAG:
		option = global_flags[index]
	default:
		panic("A found completion option has no collection.")
	}
	base := Candidate{}
	base.Segments[0] = "-"
	base.Segments[1] = string(label)
	base.Segments[2] = "="
	candidates, is_enum := complete_enum_members(
		option, base, Value_Text(current), storage,
	)
	if !is_enum {
		return Enum_Candidates(storage[:0])
	}
	return candidates
}

// Reports the positional slot the cursor sits in: the count of bare (non-flag) words
// already typed after args_start, excluding the word under the cursor. A named argument
// (-label=value) is not counted, matching how completion offers members for the next bare
// slot.
func positional_index(
	arguments Arguments, words Completion_Words, args_start Completion_Arguments_Start,
) (position Positional_Cursor) {
	defer func() { Positional_Cursor_Invariants(position, "positional_index.position") }()
	Arguments_Invariants(arguments, "positional_index.arguments")
	Completion_Words_Invariants(words, "positional_index.words")
	Completion_Arguments_Start_Invariants(args_start, "positional_index.args_start")
	for index := int(args_start); index < len(words)-1; index++ {
		if !strings.Has_Prefix(strings.Text(words[index]), "-") {
			position++
		}
	}
	return position
}

// Finds an enum-capable option collection and index without copying union state.
func command_scope_option(
	global_flags Global_Flags,
	arguments Arguments,
	flags Flags,
	label Completion_Option_Label,
) (location Option_Location, index slices.Found_Index, found Boolean) {
	defer func() {
		Option_Location_Invariants(location, "command_scope_option.location")
		slices.Found_Index_Invariants(index, "command_scope_option.index")
		Boolean_Invariants(found, "command_scope_option.found")
	}()
	Global_Flags_Invariants(global_flags, "command_scope_option.global_flags")
	Arguments_Invariants(arguments, "command_scope_option.command_arguments")
	Flags_Invariants(flags, "command_scope_option.command_flags")
	Completion_Option_Label_Invariants(label, "command_scope_option.label")
	option_label := Option_Label(label)
	for argument_index := range arguments {
		if arguments[argument_index].Label == option_label {
			return OPTION_LOCATION_ARGUMENT, slices.Found_Index(argument_index), true
		}
	}
	for flag_index := range flags {
		if flags[flag_index].Label == option_label {
			return OPTION_LOCATION_COMMAND_FLAG, slices.Found_Index(flag_index), true
		}
	}
	for global_flag_index := range global_flags {
		if global_flags[global_flag_index].Label == option_label {
			return OPTION_LOCATION_GLOBAL_FLAG,
				slices.Found_Index(global_flag_index), true
		}
	}
	return OPTION_LOCATION_ABSENT, -1, false
}

// Handle_Completion is the pre-parse gate a binary calls before Program_Parse: it serves
// the reserved `completion <shell>` (prints the shell script) and `__complete <words...>`
// (prints candidates, one per line) invocations, returning true when it handled one. It
// runs before parsing so it works for a single-command program, whose first token is
// otherwise a positional. IO goes to output, like Print_Help.
func Handle_Completion(
	program Program, args Process_Arguments, output *Output, storage Candidates,
) (handled Boolean) {
	defer func() { Boolean_Invariants(handled, "handle_completion.handled") }()
	Output_Invariants(*output, "handle_completion.output")
	Program_Invariants(program, "handle_completion.program")
	Process_Arguments_Invariants(args, "handle_completion.args")
	Candidates_Invariants(storage, "handle_completion.storage")
	if len(args) < 2 {
		return false
	}
	switch args[1] {
	case "__complete":
		for _, candidate := range Complete(program, args[2:], storage) {
			Candidate_Write(output, candidate)
			strings.Builder_Write_Byte(&output.Builder, '\n')
		}
		return true
	case "completion":
		shell := Shell("")
		if len(args) > 2 {
			shell = Shell(args[2])
		}
		err := Completion_Script(program, shell, output)
		if err != nil {
			Output_Write_Text(output, Value_Text(err.Error()))
			strings.Builder_Write_Byte(&output.Builder, '\n')
			return true
		}
		return true
	}
	return false
}

// Completion_Script writes a bash, zsh, or fish script that wires shell completion
// to call back into `<name> __complete`, so candidates always come from the live Program.
// The script is tiny and static; all knowledge lives in the binary. A multicall program
// emits one registration per verb link. An unsupported shell is an error.
func Completion_Script(program Program, shell Shell, output *Output) (err error) {
	Program_Invariants(program, "completion_script.program")
	Shell_Invariants(shell, "completion_script.shell")
	Output_Invariants(*output, "completion_script.output")
	if block_err := completion_block(
		output, Label(program.Label), shell,
	); block_err != nil {
		return block_err
	}
	if program.Selection[0].Mode[0] == PROGRAM_MODE_MULTICALL {
		for index := range program.Selection[0].Commands {
			command := program.Selection[0].Commands[index]
			if !command_shown(command) {
				continue
			}
			if block_err := completion_block(
				output, command.Label, shell,
			); block_err != nil {
				return block_err
			}
		}
	}
	return nil
}

// Writes one shell block by replacing its fixed name directives without formatting.
func completion_block(output *Output, name Label, shell Shell) (err error) {
	Output_Invariants(*output, "completion_block.output")
	Label_Invariants(name, "completion_block.name")
	Shell_Invariants(shell, "completion_block.shell")
	format := ""
	switch shell {
	case "bash":
		format = COMPLETION_BASH_FORMAT
	case "zsh":
		format = COMPLETION_ZSH_FORMAT
	case "fish":
		format = COMPLETION_FISH_FORMAT
	default:
		return Unsupported_Completion_Shell
	}
	remainder := strings.Text(format)
	for len(remainder) > 0 {
		offset := strings.Index(remainder, COMPLETION_FORMAT_DIRECTIVE)
		if offset < 0 {
			Output_Write_Text(output, Value_Text(remainder))
			return nil
		}
		Output_Write_Text(output, Value_Text(remainder[:offset]))
		Output_Write_Text(output, Value_Text(name))
		remainder = remainder[int(offset)+len(COMPLETION_FORMAT_DIRECTIVE):]
	}
	return nil
}

// Environment_Variable declares one typed value from an injected process environment.
type Environment_Variable struct {
	// Key is the uppercase environment name.
	Key External_Key
	// Description is the one-line help text.
	Description Description
	// Type selects one scalar arm without interface inspection.
	Type External_Type_State
	// Enumeration keeps both borrowed enum arms outside interface storage.
	Enumeration External_Enumeration
	// Value is the default before parsing and the resolved value after parsing.
	// It accepts legacy composite declarations only.
	Value any
	// Enum restricts a string or integer value when it is not nil.
	// It accepts legacy composite declarations only.
	Enum any
	// Required rejects an absent source and permits no nonzero default.
	Required Required
	// Allow_Empty makes an empty source different from an absent source.
	Allow_Empty Allow_Empty
	// Hidden removes the declaration from help.
	Hidden Hidden
	// Deprecated is the guidance for a source-supplied value.
	Deprecated Deprecation
	// State holds resolved scalar storage without interface boxing.
	State External_State
}

// Environment_Variable_Invariants composes bounded environment metadata.
func Environment_Variable_Invariants(
	value Environment_Variable, namespace aver.Namespace,
) {
	External_Key_Invariants(value.Key, namespace)
	Description_Invariants(value.Description, namespace)
	Required_Invariants(value.Required, namespace)
	Allow_Empty_Invariants(value.Allow_Empty, namespace)
	Hidden_Invariants(value.Hidden, namespace)
	Deprecation_Invariants(value.Deprecated, namespace)
	External_Type_State_Invariants(value.Type, namespace)
	External_Enumeration_Invariants(value.Enumeration, namespace)
	External_State_Invariants(value.State, namespace)
}

// EXTERNAL_TYPE_STATE_COUNT follows one active scalar tag.
const EXTERNAL_TYPE_STATE_COUNT = len("type") / len("type")

// External_Type_State selects one admitted external scalar representation.
type External_Type_State [EXTERNAL_TYPE_STATE_COUNT]Scalar_Option_Type

// External_Type_State_Invariants admits only the three external scalar types.
func External_Type_State_Invariants(value External_Type_State, _ aver.Namespace) {
	aver.Always(
		value[0] >= Scalar_Option_Type(OPTION_TYPE_STRING),
		"External state type does not precede the scalar set.",
	)
	aver.Always(
		value[0] <= Scalar_Option_Type(OPTION_TYPE_BOOLEAN),
		"External state type stays inside the scalar set.",
	)
}

// EXTERNAL_ENUMERATION_COUNT follows one fixed enum union cell.
const EXTERNAL_ENUMERATION_COUNT = len("enum") / len("enum")

// External_Enumeration keeps typed permitted sets outside interface storage.
type External_Enumeration [EXTERNAL_ENUMERATION_COUNT]struct {
	String   String_Enumeration
	Integers Integer_Enumeration
}

// External_Enumeration_Invariants bounds both inactive or active borrowed arms.
func External_Enumeration_Invariants(value External_Enumeration, _ aver.Namespace) {
	aver.Always(
		len(value[0].String) <= slices.COUNT_MAXIMUM,
		"External text enum stays inside shared collection capacity.",
	)
	aver.Always(
		len(value[0].Integers) <= slices.COUNT_MAXIMUM,
		"External integer enum stays inside shared collection capacity.",
	)
}

// EXTERNAL_STATE_COUNT follows one resolved scalar cell.
const EXTERNAL_STATE_COUNT = len("state") / len("state")

// External_State keeps each scalar representation in fixed inline storage.
type External_State [EXTERNAL_STATE_COUNT]struct {
	String  External_Value_Text
	Bytes   Secret_Value_Bytes
	Integer Integer
	Boolean Boolean
	Parsed  Parsed
}

// External_State_Invariants bounds resolved external scalar storage.
func External_State_Invariants(value External_State, _ aver.Namespace) {
	state := value[0]
	aver.Always(
		len(state.String) <= SECRET_BYTES_MAX,
		"Resolved external string stays inside accepted source bytes.",
	)
	aver.Always(
		len(state.Bytes) <= SECRET_BYTES_MAX,
		"Resolved secret bytes stay inside accepted source bytes.",
	)
}

// Secret declares one typed value from an ordered list of files.
type Secret struct {
	// Key is the uppercase base filename that all Paths share.
	Key Secret_Key
	// Paths are the absolute fallback paths in declaration order.
	Paths Secret_Paths
	// Description is the one-line help text.
	Description Description
	// Type selects one scalar arm without interface inspection.
	Type External_Type_State
	// Enumeration keeps both borrowed enum arms outside interface storage.
	Enumeration External_Enumeration
	// Value is the resolved value or its type's zero value when an optional secret is absent.
	// It accepts legacy composite declarations only.
	Value any
	// Enum restricts a source-supplied string or integer value when it is not nil.
	// It accepts legacy composite declarations only.
	Enum any
	// Required rejects the declaration when no path supplies a value.
	Required Required
	// Allow_Empty makes an empty file different from an absent file.
	Allow_Empty Allow_Empty
	// Hidden removes the declaration from help.
	Hidden Hidden
	// Deprecated is the guidance for a source-supplied value.
	Deprecated Deprecation
	// State holds resolved scalar storage without interface boxing.
	State External_State
}

// Secret_Invariants composes bounded file-backed metadata.
func Secret_Invariants(value Secret, namespace aver.Namespace) {
	Secret_Key_Invariants(value.Key, namespace)
	Secret_Paths_Invariants(value.Paths, namespace)
	Description_Invariants(value.Description, namespace)
	Required_Invariants(value.Required, namespace)
	Allow_Empty_Invariants(value.Allow_Empty, namespace)
	Hidden_Invariants(value.Hidden, namespace)
	Deprecation_Invariants(value.Deprecated, namespace)
	External_Type_State_Invariants(value.Type, namespace)
	External_Enumeration_Invariants(value.Enumeration, namespace)
	External_State_Invariants(value.State, namespace)
}

// Legacy composites are normalized once so every later operation reads fixed typed state.
func external_legacy_fields(
	value any,
	enum any,
	type_state *External_Type_State,
	enumeration *External_Enumeration,
) {
	External_Type_State_Invariants(*type_state, "external_legacy_fields.type_state")
	External_Enumeration_Invariants(*enumeration, "external_legacy_fields.enumeration")
	switch value.(type) {
	case string:
		type_state[0] = Scalar_Option_Type(OPTION_TYPE_STRING)
	case int:
		type_state[0] = Scalar_Option_Type(OPTION_TYPE_INTEGER)
	case bool:
		type_state[0] = Scalar_Option_Type(OPTION_TYPE_BOOLEAN)
	}
	switch members := enum.(type) {
	case []string:
		enumeration[0].String = String_Enumeration(members)
	case []int:
		enumeration[0].Integers = Integer_Enumeration(members)
	}
}

func environment_legacy_initialize(variable *Environment_Variable) {
	Environment_Variable_Invariants(*variable, "environment_legacy_initialize.variable")
	external_legacy_fields(
		variable.Value, variable.Enum, &variable.Type, &variable.Enumeration,
	)
	if variable.Value != nil {
		if !variable.State[0].Parsed {
			variable.State = external_state_default(variable.Value)
		}
	}
}

func secret_legacy_initialize(secret *Secret) {
	Secret_Invariants(*secret, "secret_legacy_initialize.secret")
	external_legacy_fields(
		secret.Value, secret.Enum, &secret.Type, &secret.Enumeration,
	)
}

// New_Environment_Variable_Input contains one non-enum environment declaration.
type New_Environment_Variable_Input[T string | int | bool] struct {
	// Key is the uppercase environment name.
	Key External_Key
	// Description is the one-line help text.
	Description Description
	// Value supplies the type and the optional default.
	Value T
	// Required rejects an absent source and permits no nonzero default.
	Required Required
	// Allow_Empty makes an empty source different from an absent source.
	Allow_Empty Allow_Empty
	// Hidden removes the declaration from help.
	Hidden Hidden
	// Deprecated is the guidance for a source-supplied value.
	Deprecated Deprecation
}

// New_Environment_Variable_Input_Invariants composes bounded environment input metadata.
func New_Environment_Variable_Input_Invariants[T string | int | bool](
	value New_Environment_Variable_Input[T], namespace aver.Namespace,
) {
	External_Key_Invariants(value.Key, namespace)
	Description_Invariants(value.Description, namespace)
	Required_Invariants(value.Required, namespace)
	Allow_Empty_Invariants(value.Allow_Empty, namespace)
	Hidden_Invariants(value.Hidden, namespace)
	Deprecation_Invariants(value.Deprecated, namespace)
}

// New_Environment_Variable makes one typed environment declaration.
func New_Environment_Variable[T string | int | bool](
	input New_Environment_Variable_Input[T],
) (environment Environment_Variable) {
	defer func() {
		Environment_Variable_Invariants(environment, "new_environment_variable.environment")
	}()
	New_Environment_Variable_Input_Invariants(input, "new_environment_variable.input")
	return Environment_Variable{
		Key:         input.Key,
		Description: input.Description,
		Type:        External_Type_State{option_type[T]()},
		Required:    input.Required,
		Allow_Empty: input.Allow_Empty,
		Hidden:      input.Hidden,
		Deprecated:  input.Deprecated,
		State:       external_state_default(input.Value),
	}
}

// New_String_Enum_Environment_Variable_Input declares one permitted-text environment value.
type New_String_Enum_Environment_Variable_Input struct {
	// Key selects one process-environment entry.
	Key External_Key
	// Description stays safe to expose in help.
	Description Description
	// Value supplies a type-aligned optional default.
	Value String_Enum_Default
	// Enum stays borrowed so construction owns no hidden storage.
	Enum String_Enumeration
	// Required makes a nonzero compiled default contradictory.
	Required Required
	// Allow_Empty separates an explicit empty value from absence.
	Allow_Empty Allow_Empty
	// Hidden affects help without disabling resolution.
	Hidden Hidden
	// Deprecated warns only when the process supplied a value.
	Deprecated Deprecation
}

// New_String_Enum_Environment_Variable_Input_Invariants composes text-enum metadata.
func New_String_Enum_Environment_Variable_Input_Invariants(
	value New_String_Enum_Environment_Variable_Input, namespace aver.Namespace,
) {
	External_Key_Invariants(value.Key, namespace)
	Description_Invariants(value.Description, namespace)
	String_Enum_Default_Invariants(value.Value, namespace)
	String_Enumeration_Invariants(value.Enum, namespace)
	Required_Invariants(value.Required, namespace)
	Allow_Empty_Invariants(value.Allow_Empty, namespace)
	Hidden_Invariants(value.Hidden, namespace)
	Deprecation_Invariants(value.Deprecated, namespace)
}

// New_String_Enum_Environment_Variable makes one permitted-text environment declaration.
func New_String_Enum_Environment_Variable(
	input New_String_Enum_Environment_Variable_Input,
) (environment Environment_Variable) {
	defer func() {
		Environment_Variable_Invariants(
			environment, "new_string_enum_environment_variable.environment",
		)
	}()
	New_String_Enum_Environment_Variable_Input_Invariants(
		input, "new_string_enum_environment_variable.input",
	)
	return Environment_Variable{
		Key: input.Key, Description: input.Description,
		Type:        External_Type_State{Scalar_Option_Type(OPTION_TYPE_STRING)},
		Enumeration: External_Enumeration{{String: input.Enum}},
		Required:    input.Required,
		Allow_Empty: input.Allow_Empty, Hidden: input.Hidden, Deprecated: input.Deprecated,
		State: external_state_default(string(input.Value)),
	}
}

// New_Integer_Enum_Environment_Variable_Input declares one permitted integer.
type New_Integer_Enum_Environment_Variable_Input struct {
	// Key selects one process-environment entry.
	Key External_Key
	// Description stays safe to expose in help.
	Description Description
	// Value supplies a type-aligned optional default.
	Value Integer_Enum_Default
	// Enum stays borrowed so construction owns no hidden storage.
	Enum Integer_Enumeration
	// Required makes a nonzero compiled default contradictory.
	Required Required
	// Allow_Empty separates an explicit empty value from absence.
	Allow_Empty Allow_Empty
	// Hidden affects help without disabling resolution.
	Hidden Hidden
	// Deprecated warns only when the process supplied a value.
	Deprecated Deprecation
}

// New_Integer_Enum_Environment_Variable_Input_Invariants composes integer-enum metadata.
func New_Integer_Enum_Environment_Variable_Input_Invariants(
	value New_Integer_Enum_Environment_Variable_Input, namespace aver.Namespace,
) {
	External_Key_Invariants(value.Key, namespace)
	Description_Invariants(value.Description, namespace)
	Integer_Enum_Default_Invariants(value.Value, namespace)
	Integer_Enumeration_Invariants(value.Enum, namespace)
	Required_Invariants(value.Required, namespace)
	Allow_Empty_Invariants(value.Allow_Empty, namespace)
	Hidden_Invariants(value.Hidden, namespace)
	Deprecation_Invariants(value.Deprecated, namespace)
}

// New_Integer_Enum_Environment_Variable makes one permitted-integer declaration.
func New_Integer_Enum_Environment_Variable(
	input New_Integer_Enum_Environment_Variable_Input,
) (environment Environment_Variable) {
	defer func() {
		Environment_Variable_Invariants(
			environment, "new_integer_enum_environment_variable.environment",
		)
	}()
	New_Integer_Enum_Environment_Variable_Input_Invariants(
		input, "new_integer_enum_environment_variable.input",
	)
	return Environment_Variable{
		Key: input.Key, Description: input.Description,
		Type:        External_Type_State{Scalar_Option_Type(OPTION_TYPE_INTEGER)},
		Enumeration: External_Enumeration{{Integers: input.Enum}},
		Required:    input.Required,
		Allow_Empty: input.Allow_Empty, Hidden: input.Hidden, Deprecated: input.Deprecated,
		State: external_state_default(int(input.Value)),
	}
}

// New_Secret_Input contains one non-enum file-backed declaration.
type New_Secret_Input struct {
	// Paths are the absolute fallback paths in declaration order.
	Paths Secret_Paths
	// Description is the one-line help text.
	Description Description
	// Required rejects the declaration when no path supplies a value.
	Required Required
	// Allow_Empty makes an empty file different from an absent file.
	Allow_Empty Allow_Empty
	// Hidden removes the declaration from help.
	Hidden Hidden
	// Deprecated is the guidance for a source-supplied value.
	Deprecated Deprecation
}

// New_Secret_Input_Invariants composes bounded secret input metadata.
func New_Secret_Input_Invariants(value New_Secret_Input, namespace aver.Namespace) {
	Secret_Paths_Invariants(value.Paths, namespace)
	Description_Invariants(value.Description, namespace)
	Required_Invariants(value.Required, namespace)
	Allow_Empty_Invariants(value.Allow_Empty, namespace)
	Hidden_Invariants(value.Hidden, namespace)
	Deprecation_Invariants(value.Deprecated, namespace)
}

// New_Secret makes one typed file-backed declaration without a compiled-in default.
func New_Secret[T string | int | bool](input New_Secret_Input) (secret Secret) {
	defer func() { Secret_Invariants(secret, "new_secret.secret") }()
	New_Secret_Input_Invariants(input, "new_secret.input")
	return Secret{
		Key:         secret_key(input.Paths),
		Paths:       input.Paths,
		Description: input.Description,
		Type:        External_Type_State{option_type[T]()},
		Required:    input.Required,
		Allow_Empty: input.Allow_Empty,
		Hidden:      input.Hidden,
		Deprecated:  input.Deprecated,
	}
}

// New_String_Enum_Secret_Input declares one permitted-text file value.
type New_String_Enum_Secret_Input struct {
	// Paths preserve fallback order and derive one shared key.
	Paths Secret_Paths
	// Description is safe to expose because secret content never enters it.
	Description Description
	// Enum stays borrowed so construction owns no hidden storage.
	Enum String_Enumeration
	// Required turns total absence into a terminal failure.
	Required Required
	// Allow_Empty separates an empty file from absence.
	Allow_Empty Allow_Empty
	// Hidden affects help without disabling file reads.
	Hidden Hidden
	// Deprecated warns only after a file supplied a value.
	Deprecated Deprecation
}

// New_String_Enum_Secret_Input_Invariants composes text-enum secret metadata.
func New_String_Enum_Secret_Input_Invariants(
	value New_String_Enum_Secret_Input, namespace aver.Namespace,
) {
	Secret_Paths_Invariants(value.Paths, namespace)
	Description_Invariants(value.Description, namespace)
	String_Enumeration_Invariants(value.Enum, namespace)
	Required_Invariants(value.Required, namespace)
	Allow_Empty_Invariants(value.Allow_Empty, namespace)
	Hidden_Invariants(value.Hidden, namespace)
	Deprecation_Invariants(value.Deprecated, namespace)
}

// New_String_Enum_Secret makes one permitted-text secret.
func New_String_Enum_Secret(input New_String_Enum_Secret_Input) (secret Secret) {
	defer func() { Secret_Invariants(secret, "new_string_enum_secret.secret") }()
	New_String_Enum_Secret_Input_Invariants(input, "new_string_enum_secret.input")
	return Secret{
		Key: secret_key(input.Paths), Paths: input.Paths, Description: input.Description,
		Type:        External_Type_State{Scalar_Option_Type(OPTION_TYPE_STRING)},
		Enumeration: External_Enumeration{{String: input.Enum}}, Required: input.Required,
		Allow_Empty: input.Allow_Empty, Hidden: input.Hidden, Deprecated: input.Deprecated,
	}
}

// New_Integer_Enum_Secret_Input declares one permitted-integer file value.
type New_Integer_Enum_Secret_Input struct {
	// Paths preserve fallback order and derive one shared key.
	Paths Secret_Paths
	// Description is safe to expose because secret content never enters it.
	Description Description
	// Enum stays borrowed so construction owns no hidden storage.
	Enum Integer_Enumeration
	// Required turns total absence into a terminal failure.
	Required Required
	// Allow_Empty separates an empty file from absence.
	Allow_Empty Allow_Empty
	// Hidden affects help without disabling file reads.
	Hidden Hidden
	// Deprecated warns only after a file supplied a value.
	Deprecated Deprecation
}

// New_Integer_Enum_Secret_Input_Invariants composes integer-enum secret metadata.
func New_Integer_Enum_Secret_Input_Invariants(
	value New_Integer_Enum_Secret_Input, namespace aver.Namespace,
) {
	Secret_Paths_Invariants(value.Paths, namespace)
	Description_Invariants(value.Description, namespace)
	Integer_Enumeration_Invariants(value.Enum, namespace)
	Required_Invariants(value.Required, namespace)
	Allow_Empty_Invariants(value.Allow_Empty, namespace)
	Hidden_Invariants(value.Hidden, namespace)
	Deprecation_Invariants(value.Deprecated, namespace)
}

// New_Integer_Enum_Secret makes one permitted-integer secret.
func New_Integer_Enum_Secret(input New_Integer_Enum_Secret_Input) (secret Secret) {
	defer func() { Secret_Invariants(secret, "new_integer_enum_secret.secret") }()
	New_Integer_Enum_Secret_Input_Invariants(input, "new_integer_enum_secret.input")
	return Secret{
		Key: secret_key(input.Paths), Paths: input.Paths, Description: input.Description,
		Type:        External_Type_State{Scalar_Option_Type(OPTION_TYPE_INTEGER)},
		Enumeration: External_Enumeration{{Integers: input.Enum}}, Required: input.Required,
		Allow_Empty: input.Allow_Empty, Hidden: input.Hidden, Deprecated: input.Deprecated,
	}
}

// Program_Parse_Input contains deterministic process inputs and the injected I/O loop.
type Program_Parse_Input struct {
	// Arguments are the complete process argument list.
	Arguments Process_Arguments
	// Environment is the complete raw KEY=value process environment snapshot.
	Environment Process_Environment
	// Loop submits secret file operations. Programs without secrets can leave it unset.
	Loop nbio.IO
	// Command_Arguments receives active argument copies.
	Command_Arguments Command_Argument_Storage
	// Command_Flags receives active command flag copies.
	Command_Flags Command_Flag_Storage
	// Global_Flags receives active global flag copies.
	Global_Flags Global_Flag_Storage
	// Filled records one bit for each active argument, command flag, and global flag.
	Filled Filled_Options
	// Positionals receives non-named command tokens.
	Positionals Positionals
	// Slice_Named receives named variadic values.
	Slice_Named Named_Variadic_Tokens
	// String_Values receives string variadic values.
	String_Values String_Values
	// Integer_Values receives integer variadic values.
	Integer_Values Integer_Values
	// Deprecation_Warnings receives command, flag, environment, and secret warnings.
	Deprecation_Warnings Deprecation_Warnings
	// Environment_Values receives resolved declaration copies.
	Environment_Values Environment_Variables
	// Environment_Errors receives one ordered slot per declaration.
	Environment_Errors Environment_Failures
	// Environment_Warnings receives emitted environment deprecations.
	Environment_Warnings Environment_Warnings
	// Environment_Sources classifies declared raw entries without hidden map storage.
	Environment_Sources Environment_Sources
	// Secret_Values receives resolved declaration copies.
	Secret_Values Secrets
	// Secret_Errors receives one ordered slot per declaration.
	Secret_Errors Secret_Failures
	// Secret_Warnings receives one ordered warning slot per declaration.
	Secret_Warnings Secret_Warnings
	// Secret_Parsers receives one stable asynchronous runner per declaration.
	Secret_Parsers Secret_Parsers
	// Secret_Buffers gives every runner its overflow-witness read storage.
	Secret_Buffers Secret_Buffers
	// Secret_Path_Failures gives every runner ordered fallback error storage.
	Secret_Path_Failures Secret_Path_Failures
	// Failures receives compacted environment and secret failures for publication.
	Failures Failures
}

// Program_Parse_Input_Invariants composes deterministic input and caller storage bounds.
func Program_Parse_Input_Invariants(
	value Program_Parse_Input, namespace aver.Namespace,
) {
	nbio.IO_Invariants(value.Loop, namespace)
	Process_Arguments_Invariants(value.Arguments, namespace)
	Process_Environment_Invariants(value.Environment, namespace)
	Command_Argument_Storage_Invariants(value.Command_Arguments, namespace)
	Command_Flag_Storage_Invariants(value.Command_Flags, namespace)
	Global_Flag_Storage_Invariants(value.Global_Flags, namespace)
	Filled_Options_Invariants(value.Filled, namespace)
	Positionals_Invariants(value.Positionals, namespace)
	Named_Variadic_Tokens_Invariants(value.Slice_Named, namespace)
	String_Values_Invariants(value.String_Values, namespace)
	Integer_Values_Invariants(value.Integer_Values, namespace)
	Deprecation_Warnings_Invariants(value.Deprecation_Warnings, namespace)
	Environment_Variables_Invariants(value.Environment_Values, namespace)
	Environment_Failures_Invariants(value.Environment_Errors, namespace)
	Environment_Warnings_Invariants(value.Environment_Warnings, namespace)
	Environment_Sources_Invariants(value.Environment_Sources, namespace)
	Secrets_Invariants(value.Secret_Values, namespace)
	Secret_Failures_Invariants(value.Secret_Errors, namespace)
	Secret_Warnings_Invariants(value.Secret_Warnings, namespace)
	Secret_Parsers_Invariants(value.Secret_Parsers, namespace)
	Secret_Buffers_Invariants(value.Secret_Buffers, namespace)
	Secret_Path_Failures_Invariants(value.Secret_Path_Failures, namespace)
	Failures_Invariants(value.Failures, namespace)
}

// External_Parse_Input holds only storage used after CLI resolution succeeds.
type External_Parse_Input struct {
	// Environment freezes process state before asynchronous secret work starts.
	Environment Process_Environment
	// Environment_Values keeps resolved declarations in caller ownership.
	Environment_Values Environment_Variables
	// Environment_Errors preserves declaration order without hidden aggregation.
	Environment_Errors Environment_Failures
	// Environment_Warnings preserves declaration order without hidden aggregation.
	Environment_Warnings Environment_Warnings
	// Environment_Sources replaces an allocating lookup map.
	Environment_Sources Environment_Sources
	// Secret_Values keeps resolved declarations in caller ownership.
	Secret_Values Secrets
	// Secret_Errors preserves declaration order until every read retires.
	Secret_Errors Secret_Failures
	// Secret_Warnings preserves declaration order until publication.
	Secret_Warnings Secret_Warnings
	// Secret_Parsers keeps asynchronous state stable after this call returns.
	Secret_Parsers Secret_Parsers
	// Secret_Buffers gives each read an independent overflow witness byte.
	Secret_Buffers Secret_Buffers
	// Secret_Path_Failures keeps fallback failures bounded and ordered.
	Secret_Path_Failures Secret_Path_Failures
}

// External_Parse_Input_Invariants composes post-CLI caller storage bounds.
func External_Parse_Input_Invariants(
	value External_Parse_Input, namespace aver.Namespace,
) {
	Process_Environment_Invariants(value.Environment, namespace)
	Environment_Variables_Invariants(value.Environment_Values, namespace)
	Environment_Failures_Invariants(value.Environment_Errors, namespace)
	Environment_Warnings_Invariants(value.Environment_Warnings, namespace)
	Environment_Sources_Invariants(value.Environment_Sources, namespace)
	Secrets_Invariants(value.Secret_Values, namespace)
	Secret_Failures_Invariants(value.Secret_Errors, namespace)
	Secret_Warnings_Invariants(value.Secret_Warnings, namespace)
	Secret_Parsers_Invariants(value.Secret_Parsers, namespace)
	Secret_Buffers_Invariants(value.Secret_Buffers, namespace)
	Secret_Path_Failures_Invariants(value.Secret_Path_Failures, namespace)
}

// Parse_Result is the terminal parser result.
type Parse_Result struct {
	// Command contains all CLI and external values that resolved before Error.
	Command Command
	// Error joins all external failures after a successful CLI parse.
	Error error
}

// Parse_Result_Invariants composes the terminal command state.
func Parse_Result_Invariants(value Parse_Result, namespace aver.Namespace) {
	Command_Invariants(value.Command, namespace)
}

// Workspace_Arguments is phase-owned argument storage.
type Workspace_Arguments []Option

// Workspace_Arguments_Invariants bounds argument storage structurally.
func Workspace_Arguments_Invariants(
	value Workspace_Arguments, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Workspace_Flags is phase-owned command flag storage.
type Workspace_Flags []Option

// Workspace_Flags_Invariants bounds command flag storage structurally.
func Workspace_Flags_Invariants(value Workspace_Flags, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Workspace_Filled is phase-owned assignment-bit storage.
type Workspace_Filled []bool

// Workspace_Filled_Invariants bounds assignment bits structurally.
func Workspace_Filled_Invariants(value Workspace_Filled, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Workspace_Positionals is phase-owned positional storage.
type Workspace_Positionals []Indexed_Token

// Workspace_Positionals_Invariants bounds positional storage structurally.
func Workspace_Positionals_Invariants(
	value Workspace_Positionals, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Workspace_Variadic is phase-owned named variadic storage.
type Workspace_Variadic []Indexed_Token

// Workspace_Variadic_Invariants bounds named variadic storage structurally.
func Workspace_Variadic_Invariants(value Workspace_Variadic, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Workspace is caller-owned option and token storage used before result publication.
type Workspace struct {
	// Command_Arguments receives one active command's argument copies.
	Command_Arguments Workspace_Arguments
	// Command_Flags receives one active command's flag copies.
	Command_Flags Workspace_Flags
	// Filled records active option assignment.
	Filled Workspace_Filled
	// Positionals retains bare tokens until assignment.
	Positionals Workspace_Positionals
	// Slice_Named retains named variadic tokens until merge.
	Slice_Named Workspace_Variadic
	// String_Values receives one active string variadic value set.
	String_Values String_Values
	// Integer_Values receives one active integer variadic value set.
	Integer_Values Integer_Values
	// Deprecation_Warnings receives every warning published with the active command.
	Deprecation_Warnings Deprecation_Warnings
	// Failure points at publication diagnostic storage after initialization.
	Failure Optional_Failure_Reference
}

// Workspace_Invariants enforces caller storage capacity across parser phases.
func Workspace_Invariants(value Workspace, namespace aver.Namespace) {
	Workspace_Arguments_Invariants(value.Command_Arguments, namespace)
	Workspace_Flags_Invariants(value.Command_Flags, namespace)
	Workspace_Filled_Invariants(value.Filled, namespace)
	Workspace_Positionals_Invariants(value.Positionals, namespace)
	Workspace_Variadic_Invariants(value.Slice_Named, namespace)
	String_Values_Invariants(value.String_Values, namespace)
	Integer_Values_Invariants(value.Integer_Values, namespace)
	Deprecation_Warnings_Invariants(value.Deprecation_Warnings, namespace)
	Optional_Failure_Reference_Invariants(value.Failure, namespace)
}

// PARSER_COMPLETION_COUNT follows the one publication bit.
const PARSER_COMPLETION_COUNT = len("complete") / len("complete")

// Parser_Completion stores terminal state without phase-wide Boolean branch contracts.
type Parser_Completion [PARSER_COMPLETION_COUNT]Parser_Complete

// Parser_Completion_Invariants fixes the one intrinsically bounded Boolean cell.
func Parser_Completion_Invariants(value Parser_Completion, _ aver.Namespace) {
	aver.Always(
		len(value) == PARSER_COMPLETION_COUNT,
		"Parser completion storage has its exact static length.",
	)
}

// Publication owns terminal result and external-resolution state.
type Publication struct {
	// Result stays inside caller-owned parser storage.
	Result Parse_Result
	// Completion distinguishes pending work from terminal publication.
	Completion Parser_Completion
	// Loop is the injected submission surface.
	Loop nbio.IO
	// Environment_Errors keep environment declaration order.
	Environment_Errors Environment_Failures
	// Secret_Errors keep secret declaration order.
	Secret_Errors Secret_Failures
	// Environment_Warnings keep environment declaration order.
	Environment_Warnings Environment_Warnings
	// Secret_Warnings keep secret declaration order.
	Secret_Warnings Secret_Warnings
	// Secret_Count is the number of declarations that did not retire.
	Secret_Count Declaration_Count
	// Failure owns ordered external diagnostic text.
	Failure Failure
}

// Publication_Invariants composes ordered external and terminal state.
func Publication_Invariants(value Publication, namespace aver.Namespace) {
	nbio.IO_Invariants(value.Loop, namespace)
	Parse_Result_Invariants(value.Result, namespace)
	Parser_Completion_Invariants(value.Completion, namespace)
	Environment_Failures_Invariants(value.Environment_Errors, namespace)
	Secret_Failures_Invariants(value.Secret_Errors, namespace)
	Environment_Warnings_Invariants(value.Environment_Warnings, namespace)
	Secret_Warnings_Invariants(value.Secret_Warnings, namespace)
	Declaration_Count_Invariants(value.Secret_Count, namespace)
	Failure_Invariants(value.Failure, namespace)
}

// Parser holds stable completion storage until all secret operations retire.
type Parser struct {
	// Publication is embedded so result access remains direct after phase separation.
	Publication
	// Workspace stays separate because result state does not exist during token assignment.
	Workspace Workspace
}

// Parser_Invariants composes caller storage and ordered publication state.
func Parser_Invariants(value Parser, namespace aver.Namespace) {
	Publication_Invariants(value.Publication, namespace)
	Workspace_Invariants(value.Workspace, namespace)
}

// Parser_Done reports pending work or returns the terminal caller-owned result.
func Parser_Done(parser *Parser) (result Parse_Result, complete Parser_Complete) {
	defer func() {
		Parse_Result_Invariants(result, "parser_done.result")
		Parser_Complete_Invariants(complete, "parser_done.complete")
	}()
	Parser_Invariants(*parser, "parser_done.parser")
	return parser.Result, parser.Completion[0]
}

// Get_Environment returns a declared environment variable and panics for an unknown key.
func Get_Environment(
	environment Resolved_Environment,
	key External_Key,
) (variable Environment_Variable) {
	defer func() { Environment_Variable_Invariants(variable, "get_environment.variable") }()
	Resolved_Environment_Invariants(environment, "get_environment.environment")
	External_Key_Invariants(key, "get_environment.key")
	for _, candidate := range environment {
		if External_Key(candidate.Key) == key {
			return candidate
		}
	}
	panic("Environment variable is not declared.")
}

// Environment_String returns one string declaration without interface boxing.
func Environment_String(variable Environment_Variable) (value External_Value_Text) {
	defer func() {
		External_Value_Text_Invariants(value, "environment_string.value")
	}()
	Environment_Variable_Invariants(variable, "environment_string.variable")
	environment_legacy_initialize(&variable)
	if Option_Type(variable.Type[0]) != OPTION_TYPE_STRING {
		panic("Environment_String received a non-string declaration.")
	}
	return variable.State[0].String
}

// Environment_Integer returns one integer declaration without interface boxing.
func Environment_Integer(variable Environment_Variable) (value Integer) {
	defer func() { Integer_Invariants(value, "environment_integer.value") }()
	Environment_Variable_Invariants(variable, "environment_integer.variable")
	environment_legacy_initialize(&variable)
	if Option_Type(variable.Type[0]) != OPTION_TYPE_INTEGER {
		panic("Environment_Integer received a non-integer declaration.")
	}
	return variable.State[0].Integer
}

// Environment_Boolean returns one Boolean declaration without interface boxing.
func Environment_Boolean(variable Environment_Variable) (value Boolean) {
	defer func() { Boolean_Invariants(value, "environment_boolean.value") }()
	Environment_Variable_Invariants(variable, "environment_boolean.variable")
	environment_legacy_initialize(&variable)
	if Option_Type(variable.Type[0]) != OPTION_TYPE_BOOLEAN {
		panic("Environment_Boolean received a non-Boolean declaration.")
	}
	return variable.State[0].Boolean
}

// Get_Secret returns a declared secret and panics for an unknown key.
func Get_Secret(secrets Resolved_Secrets, key External_Key) (secret Secret) {
	defer func() { Secret_Invariants(secret, "get_secret.secret") }()
	Resolved_Secrets_Invariants(secrets, "get_secret.secrets")
	External_Key_Invariants(key, "get_secret.key")
	for _, candidate := range secrets {
		if External_Key(candidate.Key) == key {
			return candidate
		}
	}
	panic("Secret is not declared.")
}

// Secret_String returns one string declaration as caller-owned source bytes.
func Secret_String(secret Secret) (value Secret_Value_Bytes) {
	defer func() { Secret_Value_Bytes_Invariants(value, "secret_string.value") }()
	Secret_Invariants(secret, "secret_string.secret")
	secret_legacy_initialize(&secret)
	if Option_Type(secret.Type[0]) != OPTION_TYPE_STRING {
		panic("Secret_String received a non-string declaration.")
	}
	return secret.State[0].Bytes
}

// Secret_Integer returns one integer declaration without interface boxing.
func Secret_Integer(secret Secret) (value Integer) {
	defer func() { Integer_Invariants(value, "secret_integer.value") }()
	Secret_Invariants(secret, "secret_integer.secret")
	secret_legacy_initialize(&secret)
	if Option_Type(secret.Type[0]) != OPTION_TYPE_INTEGER {
		panic("Secret_Integer received a non-integer declaration.")
	}
	return secret.State[0].Integer
}

// Secret_Boolean returns one Boolean declaration without interface boxing.
func Secret_Boolean(secret Secret) (value Boolean) {
	defer func() { Boolean_Invariants(value, "secret_boolean.value") }()
	Secret_Invariants(secret, "secret_boolean.secret")
	secret_legacy_initialize(&secret)
	if Option_Type(secret.Type[0]) != OPTION_TYPE_BOOLEAN {
		panic("Secret_Boolean received a non-Boolean declaration.")
	}
	return secret.State[0].Boolean
}

// Program_Parse resolves CLI inputs before it reads environment variables or secret files.
func Program_Parse(
	program *Program, parser *Parser, input Program_Parse_Input,
) {
	Program_Invariants(*program, "program_parse.program")
	Parser_Invariants(*parser, "program_parse.parser")
	Program_Parse_Input_Invariants(input, "program_parse.input")
	selection := parser_initialize(program, parser, input)
	command, parse_err := program_parse_arguments(
		&selection, input.Arguments, &parser.Workspace)
	parser.Result.Command = command
	if parse_err != nil {
		parser.Result.Error = parse_err
		parser.Completion[0] = true
		return
	}
	program_parse_external(
		*program, Publication_Reference{&parser.Publication},
		External_Parse_Input{
			Environment:          input.Environment,
			Environment_Values:   input.Environment_Values,
			Environment_Errors:   input.Environment_Errors,
			Environment_Warnings: input.Environment_Warnings,
			Environment_Sources:  input.Environment_Sources,
			Secret_Values:        input.Secret_Values,
			Secret_Errors:        input.Secret_Errors,
			Secret_Warnings:      input.Secret_Warnings,
			Secret_Parsers:       input.Secret_Parsers,
			Secret_Buffers:       input.Secret_Buffers,
			Secret_Path_Failures: input.Secret_Path_Failures,
		},
	)
}

// Parser initialization copies only mutable definition slices into caller storage.
func parser_initialize(
	program *Program, parser *Parser, input Program_Parse_Input,
) (selection Program_Selection) {
	defer func() {
		Program_Selection_Invariants(selection, "parser_initialize.selection")
	}()
	Program_Invariants(*program, "parser_initialize.program")
	Parser_Invariants(*parser, "parser_initialize.parser")
	Program_Parse_Input_Invariants(input, "parser_initialize.input")
	selection = Program_Selection{
		program.Selection[0],
	}
	original_global_flags := selection[0].Global_Flags
	if len(input.Global_Flags) < len(original_global_flags) {
		panic("Program_Parse Global_Flags storage is too small.")
	}
	selection[0].Global_Flags = Global_Flags(input.Global_Flags[:len(original_global_flags)])
	copy(selection[0].Global_Flags, original_global_flags)
	for index := range selection[0].Global_Flags {
		option_legacy_initialize(&selection[0].Global_Flags[index])
	}
	program.Selection[0].Global_Flags = selection[0].Global_Flags
	*parser = Parser{
		Publication: Publication{
			Loop:                 input.Loop,
			Environment_Errors:   input.Environment_Errors[:0],
			Environment_Warnings: input.Environment_Warnings[:0],
			Failure:              Failure{},
		},
		Workspace: Workspace{
			Command_Arguments:    Workspace_Arguments(input.Command_Arguments),
			Command_Flags:        Workspace_Flags(input.Command_Flags),
			Filled:               Workspace_Filled(input.Filled),
			Positionals:          Workspace_Positionals(input.Positionals),
			Slice_Named:          Workspace_Variadic(input.Slice_Named),
			String_Values:        input.String_Values,
			Integer_Values:       input.Integer_Values,
			Deprecation_Warnings: input.Deprecation_Warnings,
		},
	}
	parser.Workspace.Failure = Optional_Failure_Reference{&parser.Publication.Failure}
	return selection
}

// External parsing stays separate because argument failures publish before source IO.
func program_parse_external(
	program Program, reference Publication_Reference, input External_Parse_Input,
) {
	Program_Invariants(program, "program_parse_external.program")
	Publication_Reference_Invariants(reference, "program_parse_external.reference")
	External_Parse_Input_Invariants(input, "program_parse_external.input")
	parser := reference[0]
	if len(program.Environment_Variables) == 0 {
		if len(program.Secrets) == 0 {
			parser_start_secrets(
				reference, input.Secret_Parsers,
				input.Secret_Buffers, input.Secret_Path_Failures,
			)
			parser.Completion[0] = true
			return
		}
	}
	parser.Result.Command.Environment = Resolved_Environment(
		copy_environment(program.Environment_Variables, input.Environment_Values),
	)
	parser.Result.Command.Secrets = Resolved_Secrets(
		copy_secrets(program.Secrets, input.Secret_Values),
	)
	failure_reset(Failure_Reference{&parser.Failure})
	parser.Environment_Errors, parser.Environment_Warnings = environment_parse(
		Environment_Variables(parser.Result.Command.Environment),
		input.Environment, input.Environment_Errors,
		input.Environment_Warnings, input.Environment_Sources,
		Failure_Reference{&parser.Failure},
	)
	if len(parser.Result.Command.Secrets) == 0 {
		parser_start_secrets(
			reference, input.Secret_Parsers,
			input.Secret_Buffers, input.Secret_Path_Failures,
		)
		parser_publish(reference)
		return
	}
	if parser.Loop.Storage.Open_At_Procedure == nil {
		panic("Program_Parse needs Loop for secret declarations.")
	}
	secret_count := len(parser.Result.Command.Secrets)
	if len(input.Secret_Errors) < secret_count {
		panic("Program_Parse Secret_Errors storage is too small.")
	}
	if len(input.Secret_Warnings) < secret_count {
		panic("Program_Parse Secret_Warnings storage is too small.")
	}
	parser.Secret_Errors = input.Secret_Errors[:secret_count]
	parser.Secret_Warnings = input.Secret_Warnings[:secret_count]
	for index := range parser.Secret_Errors {
		parser.Secret_Errors[index] = Secret_Failure{}
		parser.Secret_Warnings[index] = Warning{}
	}
	parser.Secret_Count = Declaration_Count(secret_count)
	parser_start_secrets(
		reference, input.Secret_Parsers,
		input.Secret_Buffers, input.Secret_Path_Failures,
	)
}

// Definitions stay immutable because one Program can parse more than one process snapshot.
func copy_environment(
	input Environment_Declarations, storage Environment_Variables,
) (output Environment_Variables) {
	defer func() {
		Environment_Variables_Invariants(output, "copy_environment.output")
	}()
	Environment_Declarations_Invariants(input, "copy_environment.input")
	Environment_Variables_Invariants(storage, "copy_environment.storage")
	if len(storage) < len(input) {
		panic("Program_Parse Environment_Values storage is too small.")
	}
	output = storage[:len(input)]
	copy(output, input)
	for index := range output {
		environment_legacy_initialize(&output[index])
		output[index].State[0].Parsed = false
	}
	return output
}

// Each parse owns its secret values and internal presence state.
func copy_secrets(input Secret_Declarations, storage Secrets) (output Secrets) {
	defer func() { Secrets_Invariants(output, "copy_secrets.output") }()
	Secret_Declarations_Invariants(input, "copy_secrets.input")
	Secrets_Invariants(storage, "copy_secrets.storage")
	if len(storage) < len(input) {
		panic("Program_Parse Secret_Values storage is too small.")
	}
	output = storage[:len(input)]
	copy(output, input)
	for index := range output {
		secret_legacy_initialize(&output[index])
		output[index].State = External_State{}
	}
	return output
}

// The first path supplies the key before construction validation examines the complete list.
func secret_key(paths Secret_Paths) (key Secret_Key) {
	defer func() { Secret_Key_Invariants(key, "secret_key.key") }()
	Secret_Paths_Invariants(paths, "secret_key.paths")
	if len(paths) == 0 {
		return ""
	}
	return Secret_Key(filepath.Base(paths[0]))
}

// Construction rejects source ambiguity and declarations that no loader can resolve safely.
func external_validate(environment Environment_Variables, secrets Secrets) {
	Environment_Variables_Invariants(environment, "external_validate.environment")
	Secrets_Invariants(secrets, "external_validate.secrets")
	seen := External_Key_Set{}
	for index := range environment {
		environment_legacy_initialize(&environment[index])
		variable := environment[index]
		external_key_validate(variable.Key, SOURCE_ENVIRONMENT)
		if seen[variable.Key] {
			panic("External key is declared more than once.")
		}
		seen[variable.Key] = true
		key := Resolved_External_Key(variable.Key)
		environment_validate_value(key, Environment_Default{{
			Value: External_Value{
				Type: variable.Type, Enumeration: variable.Enumeration,
				State: variable.State, Legacy_Value: variable.Value,
				Legacy_Enumeration: variable.Enum,
			},
			Required: variable.Required,
		}})
	}
	for index := range secrets {
		secret_legacy_initialize(&secrets[index])
		secret := secrets[index]
		secret_validate_paths(secret)
		key := External_Key(secret.Key)
		if seen[key] {
			panic("External key is declared more than once.")
		}
		seen[key] = true
		secret_validate_value(Resolved_External_Key(secret.Key), External_Value{
			Type: secret.Type, Enumeration: secret.Enumeration, State: secret.State,
			Legacy_Value: secret.Value, Legacy_Enumeration: secret.Enum,
		})
	}
}

// An uppercase key makes the environment namespace and filename namespace identical.
func external_key_validate(key External_Key, source Source_Name) {
	External_Key_Invariants(key, "external_key_validate.key")
	Source_Name_Invariants(source, "external_key_validate.source")
	if !external_key_valid(key) {
		panic("External key is not an uppercase environment name.")
	}
	if key == "HELP" {
		panic("External key is reserved.")
	}
}

// Reports the portable environment-name grammar selected for all external declarations.
func external_key_valid(key External_Key) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "external_key_valid.valid") }()
	External_Key_Invariants(key, "external_key_valid.key")
	if len(key) == 0 {
		return false
	}
	for index, character := range key {
		if index == 0 {
			if !ascii_uppercase(Character(character)) {
				return false
			}
			continue
		}
		if ascii_uppercase(Character(character)) {
			continue
		}
		if ascii_digit(Character(character)) {
			continue
		}
		if character == '_' {
			continue
		}
		return false
	}
	return true
}

// Reports one uppercase ASCII letter without locale-dependent classification.
func ascii_uppercase(character Character) (uppercase Boolean) {
	defer func() { Boolean_Invariants(uppercase, "ascii_uppercase.uppercase") }()
	Character_Invariants(character, "ascii_uppercase.character")
	if character < 'A' {
		return false
	}
	return character <= 'Z'
}

// Reports one ASCII digit without locale-dependent classification.
func ascii_digit(character Character) (digit Boolean) {
	defer func() { Boolean_Invariants(digit, "ascii_digit.digit") }()
	Character_Invariants(character, "ascii_digit.character")
	if character < '0' {
		return false
	}
	return character <= '9'
}

// External_Value carries the validated union without tying it to one source kind.
type External_Value struct {
	// Type selects the active scalar storage arm.
	Type External_Type_State
	// Enumeration retains the optional permitted set.
	Enumeration External_Enumeration
	// State contains the compiled default checked during construction.
	State External_State
	// Legacy_Value preserves unsupported-type diagnostics for raw composites.
	Legacy_Value any
	// Legacy_Enumeration preserves unsupported-type diagnostics for raw composites.
	Legacy_Enumeration any
}

// External_Value_Invariants composes the typed external union.
func External_Value_Invariants(value External_Value, namespace aver.Namespace) {
	External_Type_State_Invariants(value.Type, namespace)
	External_Enumeration_Invariants(value.Enumeration, namespace)
	External_State_Invariants(value.State, namespace)
}

// ENVIRONMENT_DEFAULT_COUNT follows one declaration-default validation cell.
const ENVIRONMENT_DEFAULT_COUNT = len("default") / len("default")

// Environment_Default isolates required-default validation from declaration identity.
type Environment_Default [ENVIRONMENT_DEFAULT_COUNT]struct {
	// Value is the normalized scalar union.
	Value External_Value
	// Required forbids a nonzero compiled default.
	Required Required
}

// Environment_Default_Invariants checks only state read during default validation.
func Environment_Default_Invariants(value Environment_Default, namespace aver.Namespace) {
	External_Value_Invariants(value[0].Value, namespace)
	aver.Always(
		bool(value[0].Required) == bool(value[0].Required),
		"Required state is a machine Boolean.",
	)
}

// Required environment declarations use typed state for the compiled default.
func environment_validate_value(key Resolved_External_Key, environment Environment_Default) {
	Resolved_External_Key_Invariants(key, "environment_validate_value.key")
	Environment_Default_Invariants(environment, "environment_validate_value.environment")
	state := environment[0]
	external_value_validate(key, state.Value, Boolean(!state.Required))
	if state.Required {
		if !external_value_zero(state.Value) {
			panic("Required environment variable has a nonzero default.")
		}
	}
}

// Secret absence is separate from enum membership, so its zero value is never an enum default.
func secret_validate_value(key Resolved_External_Key, value External_Value) {
	Resolved_External_Key_Invariants(key, "secret_validate_value.key")
	External_Value_Invariants(value, "secret_validate_value.value")
	external_value_validate(key, value, false)
}

// External enum validation keeps environment identity independent of dashed option bounds.
func external_value_validate(
	key Resolved_External_Key, value External_Value, default_is_value Boolean,
) {
	Resolved_External_Key_Invariants(key, "external_value_validate.key")
	External_Value_Invariants(value, "external_value_validate.value")
	Boolean_Invariants(default_is_value, "external_value_validate.default_is_value")
	external_legacy_validate(key, value)
	if value.Enumeration[0].String == nil {
		if value.Enumeration[0].Integers == nil {
			return
		}
	}
	if value.Enumeration[0].String != nil {
		external_string_enum_validate(key, value, default_is_value)
		return
	}
	external_integer_enum_validate(key, value, default_is_value)
}

func external_legacy_validate(key Resolved_External_Key, value External_Value) {
	Resolved_External_Key_Invariants(key, "external_legacy_validate.key")
	External_Value_Invariants(value, "external_legacy_validate.value")
	if value.Legacy_Value != nil {
		switch value.Legacy_Value.(type) {
		default:
			panic("External value has unsupported type.")
		case string, int, bool:
		}
	}
	if value.Legacy_Enumeration != nil {
		switch value.Legacy_Enumeration.(type) {
		default:
			panic("External value has unsupported enum type.")
		case []string, []int:
		}
	}
}

func external_string_enum_validate(
	key Resolved_External_Key, value External_Value, default_is_value Boolean,
) {
	Resolved_External_Key_Invariants(key, "external_string_enum_validate.key")
	External_Value_Invariants(value, "external_string_enum_validate.value")
	Boolean_Invariants(default_is_value, "external_string_enum_validate.default_is_value")
	if value.Enumeration[0].Integers != nil {
		panic("External value has two enum types.")
	}
	members := value.Enumeration[0].String
	if Option_Type(value.Type[0]) != OPTION_TYPE_STRING {
		panic("Text external enum has a non-text value.")
	}
	if len(members) == 0 {
		panic("External value has an empty enum.")
	}
	if default_is_value {
		if !slices.Contains(members, string(value.State[0].String)) {
			panic("External default is not one of its enum values.")
		}
	}
}

func external_integer_enum_validate(
	key Resolved_External_Key, value External_Value, default_is_value Boolean,
) {
	Resolved_External_Key_Invariants(key, "external_integer_enum_validate.key")
	External_Value_Invariants(value, "external_integer_enum_validate.value")
	Boolean_Invariants(default_is_value, "external_integer_enum_validate.default_is_value")
	members := value.Enumeration[0].Integers
	if Option_Type(value.Type[0]) != OPTION_TYPE_INTEGER {
		panic("Integer external enum has a non-integer value.")
	}
	if len(members) == 0 {
		panic("External value has an empty enum.")
	}
	if default_is_value {
		if !slices.Contains(members, int(value.State[0].Integer)) {
			panic("External default is not one of its enum values.")
		}
	}
}

// Reports the zero value for each permitted external declaration type.
func external_value_zero(value External_Value) (zero Boolean) {
	defer func() { Boolean_Invariants(zero, "external_value_zero.zero") }()
	External_Value_Invariants(value, "external_value_zero.value")
	switch Option_Type(value.Type[0]) {
	case OPTION_TYPE_STRING:
		return value.State[0].String == ""
	case OPTION_TYPE_INTEGER:
		return value.State[0].Integer == 0
	case OPTION_TYPE_BOOLEAN:
		return Boolean(!value.State[0].Boolean)
	}
	panic("External value has no scalar type.")
}

// Path validation makes the filename a stable key across all fallbacks.
func secret_validate_paths(secret Secret) {
	Secret_Invariants(secret, "secret_validate_paths.secret")
	if len(secret.Paths) == 0 {
		panic("Secret has zero paths.")
	}
	for _, path := range secret.Paths {
		if !filepath.IsAbs(path) {
			panic("Secret path is not absolute.")
		}
		key := External_Key(filepath.Base(path))
		external_key_validate(key, SOURCE_SECRET)
		if Secret_Key(key) != secret.Key {
			panic("Secret paths do not share one base filename.")
		}
	}
}

// One environment source record keeps malformed and duplicate states representable.
type Environment_Source struct {
	// Value is the text after the first equals sign.
	Value Environment_Value_Text
	// Occurrences counts all well-formed and malformed entries for the key.
	Occurrences Occurrence_Count
	// Malformed reports that an entry had no equals sign.
	Malformed Boolean
}

// Environment_Source_Invariants composes classified source state.
func Environment_Source_Invariants(value Environment_Source, namespace aver.Namespace) {
	Environment_Value_Text_Invariants(value.Value, namespace)
	Occurrence_Count_Invariants(value.Occurrences, namespace)
	Boolean_Invariants(value.Malformed, namespace)
}

// Environment failures stay indexed by declaration instead of source-list order.
func environment_parse(
	environment Environment_Variables,
	raw Process_Environment,
	failure_storage Environment_Failures,
	warning_storage Environment_Warnings,
	source_storage Environment_Sources,
	failure_output Failure_Reference,
) (failures Environment_Failures, warnings Environment_Warnings) {
	defer func() {
		Environment_Failures_Invariants(failures, "environment_parse.failures")
		Environment_Warnings_Invariants(warnings, "environment_parse.warnings")
	}()
	Environment_Variables_Invariants(environment, "environment_parse.environment")
	Process_Environment_Invariants(raw, "environment_parse.raw")
	Environment_Failures_Invariants(failure_storage, "environment_parse.failure_storage")
	Environment_Warnings_Invariants(warning_storage, "environment_parse.warning_storage")
	Environment_Sources_Invariants(source_storage, "environment_parse.source_storage")
	Failure_Reference_Invariants(failure_output, "environment_parse.failure_output")
	if len(failure_storage) < len(environment) {
		panic("Program_Parse Environment_Errors storage is too small.")
	}
	failures = failure_storage[:len(environment)]
	for index := range failures {
		failures[index] = nil
	}
	warnings = warning_storage[:0]
	sources := environment_sources(environment, raw, source_storage)
	for index := range environment {
		variable := &environment[index]
		source := sources[variable.Key]
		failure_start := failure_output[0].Size
		if failure_start[0] != 0 {
			failure_write(failure_output, Failure_Fragment{"\n"})
		}
		failure := environment_source_failure(variable, source, failure_output)
		if failure == nil {
			failure_output[0].Size = failure_start
		}
		failures[index] = failure
		if failure == nil {
			warning, emit_warning := environment_warning(
				Resolved_External_Key(variable.Key), variable.Deprecated,
				variable.Allow_Empty, source.Value,
				Boolean(source.Occurrences == 1),
			)
			if emit_warning {
				if len(warnings) == len(warning_storage) {
					panic(
						"Program_Parse Environment_Warnings " +
							"storage is too small.",
					)
				}
				warning_count := len(warnings)
				warnings = warning_storage[:warning_count+1]
				warnings[warning_count] = warning
			}
		}
	}
	return failures, warnings
}

// Undeclared malformed entries do not affect a program that inherits a large environment.
func environment_sources(
	environment Environment_Variables,
	raw Process_Environment,
	storage Environment_Sources,
) (sources Environment_Sources) {
	defer func() {
		Environment_Sources_Invariants(sources, "environment_sources.sources")
	}()
	Environment_Variables_Invariants(environment, "environment_sources.environment")
	Process_Environment_Invariants(raw, "environment_sources.raw")
	Environment_Sources_Invariants(storage, "environment_sources.storage")
	clear(storage)
	for _, variable := range environment {
		storage[variable.Key] = Environment_Source{}
	}
	sources = storage
	for _, entry := range raw {
		aver.Always(
			len(entry) <= strings.TEXT_SIZE_MAXIMUM,
			"Environment entries stay inside shared text capacity.",
		)
		key_text, value_text, has_equals := strings.Cut(strings.Text(entry), "=")
		key := External_Key(key_text)
		value := string(value_text)
		if !has_equals {
			entry_key := External_Key(entry)
			if _, declared := sources[entry_key]; declared {
				source := sources[entry_key]
				source.Occurrences++
				source.Malformed = true
				sources[entry_key] = source
			}
			continue
		}
		if _, declared := sources[key]; !declared {
			continue
		}
		source := sources[key]
		source.Occurrences++
		source.Value = Environment_Value_Text(value)
		sources[key] = source
	}
	return sources
}

// Resolves one declaration after the raw snapshot has classified source shape.
func environment_source_failure(
	variable *Environment_Variable,
	source Environment_Source,
	failure Failure_Reference,
) (err error) {
	Environment_Variable_Invariants(*variable, "environment_source_failure.variable")
	Environment_Source_Invariants(source, "environment_source_failure.source")
	Failure_Reference_Invariants(failure, "environment_source_failure.failure")
	if source.Malformed {
		failure_write(failure, Failure_Fragment{"environment variable "})
		failure_write(failure, Failure_Fragment{string(variable.Key)})
		failure_write(failure, Failure_Fragment{" has no '='"})
		if source.Occurrences > 1 {
			failure_write(failure, Failure_Fragment{"\nenvironment variable "})
			failure_write(failure, Failure_Fragment{string(variable.Key)})
			failure_write(failure, Failure_Fragment{" is present more than once"})
		}
		return failure[0]
	}
	if source.Occurrences > 1 {
		failure_write(failure, Failure_Fragment{"environment variable "})
		failure_write(failure, Failure_Fragment{string(variable.Key)})
		failure_write(failure, Failure_Fragment{" is present more than once"})
		return failure[0]
	}
	absent := source.Occurrences == 0
	if source.Value == "" {
		if !variable.Allow_Empty {
			absent = true
		}
	}
	if absent {
		if variable.Required {
			failure_write(failure, Failure_Fragment{"required environment variable "})
			failure_write(failure, Failure_Fragment{string(variable.Key)})
			failure_write(failure, Failure_Fragment{" is missing"})
			return failure[0]
		}
		return nil
	}
	state, convert_err := external_convert_state(
		Resolved_External_Key(variable.Key), variable.Type, variable.Enumeration,
		source.Value, failure,
	)
	if convert_err != nil {
		return convert_err
	}
	variable.State = state
	return nil
}

// Defaults occupy the same typed cell as resolved source values.
func external_state_default(value any) (state External_State) {
	defer func() {
		External_State_Invariants(state, "external_state_default.state")
	}()
	switch scalar := value.(type) {
	case string:
		state[0].String = External_Value_Text(scalar)
	case int:
		state[0].Integer = Integer(scalar)
	case bool:
		state[0].Boolean = Boolean(scalar)
	default:
		panic("External state received an unvalidated value type.")
	}
	return state
}

// Converts external text directly into fixed scalar storage.
func external_convert_state(
	key Resolved_External_Key,
	type_state External_Type_State,
	enumeration External_Enumeration,
	raw Environment_Value_Text,
	failure Failure_Reference,
) (state External_State, err error) {
	defer func() {
		External_State_Invariants(state, "external_convert_state.state")
	}()
	Resolved_External_Key_Invariants(key, "external_convert_state.key")
	External_Type_State_Invariants(type_state, "external_convert_state.type_state")
	External_Enumeration_Invariants(enumeration, "external_convert_state.enumeration")
	Environment_Value_Text_Invariants(raw, "external_convert_state.raw")
	Failure_Reference_Invariants(failure, "external_convert_state.failure")
	switch Option_Type(type_state[0]) {
	case OPTION_TYPE_STRING:
		if enumeration[0].String != nil {
			if enum_err := external_string_enum_error(
				key, raw, Permitted_Strings(enumeration[0].String), failure,
			); enum_err != nil {
				return state, enum_err
			}
		}
		state[0].String = External_Value_Text(raw)
	case OPTION_TYPE_INTEGER:
		if len(raw) > strings.TEXT_SIZE_MAXIMUM {
			return state, external_conversion_error(
				key, EXPECTED_INTEGER, raw, failure,
			)
		}
		number, parse_err := strconv.Parse_Decimal(strconv.Text(raw))
		if parse_err != nil {
			return state, external_conversion_error(
				key, EXPECTED_INTEGER, raw, failure,
			)
		}
		if enumeration[0].Integers != nil {
			enum_err := external_integer_enum_error(
				Converted_External_Key(key), Integer(number),
				Permitted_Integers(enumeration[0].Integers), failure,
			)
			if enum_err != nil {
				return state, enum_err
			}
		}
		state[0].Integer = Integer(number)
	case OPTION_TYPE_BOOLEAN:
		if len(raw) > strings.TEXT_SIZE_MAXIMUM {
			return state, external_conversion_error(
				key, EXPECTED_BOOLEAN, raw, failure,
			)
		}
		boolean, parse_err := strconv.Parse_Boolean(strconv.Text(raw))
		if parse_err != nil {
			return state, external_conversion_error(
				key, EXPECTED_BOOLEAN, raw, failure,
			)
		}
		state[0].Boolean = Boolean(boolean)
	default:
		panic("External conversion received an unvalidated value type.")
	}
	state[0].Parsed = true
	return state, nil
}

func external_string_enum_error(
	key Resolved_External_Key,
	raw Environment_Value_Text,
	members Permitted_Strings,
	failure Failure_Reference,
) (err error) {
	Resolved_External_Key_Invariants(key, "external_string_enum_error.key")
	Environment_Value_Text_Invariants(raw, "external_string_enum_error.raw")
	Permitted_Strings_Invariants(members, "external_string_enum_error.members")
	Failure_Reference_Invariants(failure, "external_string_enum_error.failure")
	if slices.Contains(members, string(raw)) {
		return nil
	}
	var workspace levenshtein.Workspace
	match, close, _ := levenshtein.Closest(levenshtein.Closest_Input{
		Workspace: &workspace, Target: levenshtein.Target_Text_Unvalidated(raw),
		Candidates: levenshtein.Candidates_Unvalidated(members),
	})
	failure_write(failure, Failure_Fragment{"invalid value "})
	failure_write_quoted(failure, Failure_Fragment{string(raw)})
	failure_write(failure, Failure_Fragment{" for "})
	failure_write(failure, Failure_Fragment{string(key)})
	if close {
		failure_write(failure, Failure_Fragment{", did you mean "})
		failure_write_quoted(failure, Failure_Fragment{string(match)})
		failure_write(failure, Failure_Fragment{"?"})
		return failure[0]
	}
	failure_write(failure, Failure_Fragment{"; allowed: "})
	failure_write_string_set(failure, Failure_Strings{members})
	return failure[0]
}

func external_integer_enum_error(
	key Converted_External_Key,
	value Integer,
	members Permitted_Integers,
	failure Failure_Reference,
) (err error) {
	Converted_External_Key_Invariants(key, "external_integer_enum_error.key")
	Integer_Invariants(value, "external_integer_enum_error.value")
	Permitted_Integers_Invariants(members, "external_integer_enum_error.members")
	Failure_Reference_Invariants(failure, "external_integer_enum_error.failure")
	if slices.Contains(members, int(value)) {
		return nil
	}
	failure_write(failure, Failure_Fragment{"invalid value "})
	failure_write_decimal(failure, Failure_Integer{int(value)})
	failure_write(failure, Failure_Fragment{" for "})
	failure_write(failure, Failure_Fragment{string(key)})
	failure_write(failure, Failure_Fragment{"; allowed: "})
	failure_write_integer_set(failure, Failure_Integers{members})
	return failure[0]
}

// A secret conversion error names only the expected type, never the source bytes.
func external_conversion_error(
	key Resolved_External_Key,
	expected Expected_Type,
	raw Environment_Value_Text,
	failure Failure_Reference,
) (err error) {
	Resolved_External_Key_Invariants(key, "external_conversion_error.key")
	Expected_Type_Invariants(expected, "external_conversion_error.expected")
	Environment_Value_Text_Invariants(raw, "external_conversion_error.raw")
	Failure_Reference_Invariants(failure, "external_conversion_error.failure")
	expected_text := "a whole number"
	if expected == EXPECTED_BOOLEAN {
		expected_text = "a Boolean"
	}
	failure_write(failure, Failure_Fragment{string(key)})
	failure_write(failure, Failure_Fragment{" expects "})
	failure_write(failure, Failure_Fragment{expected_text})
	failure_write(failure, Failure_Fragment{", but got "})
	failure_write_quoted(failure, Failure_Fragment{string(raw)})
	return failure[0]
}

// Secret conversion reads caller bytes directly so accepted content never becomes a string.
func external_convert_secret_state(
	key Resolved_External_Key,
	type_state External_Type_State,
	enumeration External_Enumeration,
	raw Secret_Value_Bytes,
) (state External_State, failure Path_Failure_Kind) {
	defer func() {
		External_State_Invariants(state, "external_convert_secret_state.state")
		Path_Failure_Kind_Invariants(
			failure, "external_convert_secret_state.failure",
		)
	}()
	Resolved_External_Key_Invariants(key, "external_convert_secret_state.key")
	External_Type_State_Invariants(type_state, "external_convert_secret_state.type_state")
	External_Enumeration_Invariants(enumeration, "external_convert_secret_state.enumeration")
	Secret_Value_Bytes_Invariants(raw, "external_convert_secret_state.raw")
	switch Option_Type(type_state[0]) {
	case OPTION_TYPE_STRING:
		if enumeration[0].String != nil {
			permitted := false
			for _, member := range enumeration[0].String {
				if secret_bytes_equal_text(raw, Value_Text(member)) {
					permitted = true
					break
				}
			}
			if !permitted {
				return state, Path_Failure_Kind{PATH_FAILURE_KIND_STRING_ENUM}
			}
		}
		state[0].Bytes = raw
	case OPTION_TYPE_INTEGER:
		if len(raw) > strings.TEXT_SIZE_MAXIMUM {
			return state, Path_Failure_Kind{PATH_FAILURE_KIND_INTEGER}
		}
		number, parsed := secret_decimal(Secret_Number_Bytes(raw))
		if !parsed {
			return state, Path_Failure_Kind{PATH_FAILURE_KIND_INTEGER}
		}
		if enumeration[0].Integers != nil {
			if !slices.Contains(enumeration[0].Integers, int(number)) {
				return state, Path_Failure_Kind{PATH_FAILURE_KIND_INTEGER_ENUM}
			}
		}
		state[0].Integer = number
	case OPTION_TYPE_BOOLEAN:
		boolean, parsed := secret_parse_boolean(raw)
		if !parsed {
			return state, Path_Failure_Kind{PATH_FAILURE_KIND_BOOLEAN}
		}
		state[0].Boolean = Boolean(boolean)
	default:
		panic("External conversion received an unvalidated value type.")
	}
	state[0].Parsed = true
	return state, Path_Failure_Kind{PATH_FAILURE_KIND_NONE}
}

func secret_bytes_equal_text(left Secret_Value_Bytes, right Value_Text) (equal Boolean) {
	defer func() { Boolean_Invariants(equal, "secret_bytes_equal_text.equal") }()
	Secret_Value_Bytes_Invariants(left, "secret_bytes_equal_text.left")
	Value_Text_Invariants(right, "secret_bytes_equal_text.right")
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func secret_decimal(raw Secret_Number_Bytes) (number Integer, parsed Boolean) {
	defer func() {
		Integer_Invariants(number, "secret_decimal.number")
		Boolean_Invariants(parsed, "secret_decimal.parsed")
	}()
	Secret_Number_Bytes_Invariants(raw, "secret_decimal.raw")
	if len(raw) == 0 {
		return 0, false
	}
	negative := raw[0] == '-'
	start := 0
	if negative {
		start = 1
	}
	if raw[0] == '+' {
		start = 1
	}
	if len(raw) == start {
		return 0, false
	}
	limit := uint64(bits.INTEGER_MAXIMUM)
	if negative {
		limit++
	}
	var magnitude uint64
	for _, character := range raw[start:] {
		if character < '0' {
			return 0, false
		}
		if character > '9' {
			return 0, false
		}
		digit := uint64(character - '0')
		if magnitude > (limit-digit)/10 {
			return 0, false
		}
		magnitude = magnitude*10 + digit
	}
	if negative {
		if magnitude == uint64(bits.INTEGER_MAXIMUM)+1 {
			return Integer(bits.INTEGER_MINIMUM), true
		}
		return Integer(-int(magnitude)), true
	}
	return Integer(magnitude), true
}

func secret_parse_boolean(raw Secret_Value_Bytes) (value Boolean, parsed Boolean) {
	defer func() {
		Boolean_Invariants(value, "secret_parse_boolean.value")
		Boolean_Invariants(parsed, "secret_parse_boolean.parsed")
	}()
	Secret_Value_Bytes_Invariants(raw, "secret_parse_boolean.raw")
	true_values := [...]string{"1", "t", "T", "TRUE", "true", "True"}
	for _, candidate := range true_values {
		if secret_bytes_equal_text(raw, Value_Text(candidate)) {
			return true, true
		}
	}
	false_values := [...]string{"0", "f", "F", "FALSE", "false", "False"}
	for _, candidate := range false_values {
		if secret_bytes_equal_text(raw, Value_Text(candidate)) {
			return false, true
		}
	}
	return false, false
}

// A default does not trigger a warning because no external source supplied it.
func environment_warning(
	key Resolved_External_Key,
	deprecated Deprecation,
	allow_empty Allow_Empty,
	value Environment_Value_Text,
	present Boolean,
) (warning Warning, emit Boolean) {
	defer func() {
		Warning_Invariants(warning, "environment_warning.warning")
		Boolean_Invariants(emit, "environment_warning.emit")
	}()
	Resolved_External_Key_Invariants(key, "environment_warning.key")
	Deprecation_Invariants(deprecated, "environment_warning.deprecated")
	Allow_Empty_Invariants(allow_empty, "environment_warning.allow_empty")
	Environment_Value_Text_Invariants(value, "environment_warning.value")
	Boolean_Invariants(present, "environment_warning.present")
	if !present {
		return Warning{}, false
	}
	if value == "" {
		if !allow_empty {
			return Warning{}, false
		}
	}
	if deprecated == "" {
		return Warning{}, false
	}
	return Warning{
		Kind:     Warning_Kind{WARNING_KIND_ENVIRONMENT},
		Name:     Warning_Name{Value_Text(key)},
		Guidance: Warning_Guidance{deprecated},
	}, true
}

// SECRET_STATE_COUNT follows the one mutable phase record per declaration.
const SECRET_STATE_COUNT = len("state") / len("state")

// SECRET_STAGE_IDLE_VALUE encodes no submitted file operation.
const SECRET_STAGE_IDLE_VALUE uint8 = uint8(len(""))

// SECRET_STAGE_OPEN_VALUE encodes submitted open.
const SECRET_STAGE_OPEN_VALUE = SECRET_STAGE_IDLE_VALUE + uint8(len("open")/len("open"))

// SECRET_STAGE_READ_VALUE encodes submitted read.
const SECRET_STAGE_READ_VALUE = SECRET_STAGE_OPEN_VALUE + uint8(len("read")/len("read"))

// SECRET_STAGE_CLOSE_VALUE encodes submitted close.
const SECRET_STAGE_CLOSE_VALUE = SECRET_STAGE_READ_VALUE + uint8(len("close")/len("close"))

// SECRET_STAGE_COUNT follows one phase discriminator cell.
const SECRET_STAGE_COUNT = len("stage") / len("stage")

// Secret_Stage identifies the one operation currently submitted by a runner.
type Secret_Stage [SECRET_STAGE_COUNT]uint8

// Secret_Stage_Invariants admits idle and every submitted file operation.
func Secret_Stage_Invariants(value Secret_Stage, _ aver.Namespace) {
	aver.Always(
		value[0] >= SECRET_STAGE_IDLE_VALUE,
		"Secret runner stage does not precede idle.",
	)
	aver.Always(
		value[0] <= SECRET_STAGE_CLOSE_VALUE,
		"Secret runner stage does not follow close.",
	)
}

// Secret_State stores fields whose legal values depend on open/read/close phase.
type Secret_State [SECRET_STATE_COUNT]struct {
	// Path_Index is the current fallback index.
	Path_Index Secret_Path_Index
	// Path_Failures keep the declared fallback order.
	Path_Failures Path_Failures
	// Only_Absent reports that all completed paths were absent or empty.
	Only_Absent Only_Absent
	// Status_Size is the file size before Open_At.
	Status_Size Secret_Size
	// File is the descriptor that Close must retire.
	File nbio.File
	// Buffer has one byte more than the accepted size.
	Buffer Secret_Bytes
	// Candidate waits for a successful Close.
	Candidate External_State
	// Current_Failure is redacted content and Close state for one path.
	Current_Failure Path_Failure
	// Current_Absent classifies a missing or empty path.
	Current_Absent Current_Absent
}

// Secret_State_Invariants enforces phase-union storage without impossible branch trees.
func Secret_State_Invariants(value Secret_State, namespace aver.Namespace) {
	state := value[0]
	aver.Always(
		int(state.Path_Index) >= slices.POSITION_MINIMUM,
		"Secret path index is not before the first path.",
	)
	aver.Always(
		int(state.Path_Index) <= slices.COUNT_MAXIMUM,
		"Secret path cursor includes the exhausted end boundary.",
	)
	aver.Always(
		len(state.Path_Failures) <= slices.COUNT_MAXIMUM,
		"Secret path failures stay inside fallback capacity.",
	)
	aver.Always(
		int64(state.Status_Size) >= SECRET_SIZE_MINIMUM,
		"Secret status size is not negative after validation.",
	)
	aver.Always(
		int64(state.Status_Size) <= SECRET_SIZE_MAXIMUM,
		"Secret status size stays inside accepted bytes.",
	)
	aver.Always(
		(len(state.Buffer) == slices.COUNT_MINIMUM) !=
			(len(state.Buffer) == SECRET_BUFFER_BYTES_MAX),
		"Secret read storage is absent or has one overflow witness byte.",
	)
	External_State_Invariants(state.Candidate, namespace)
	Path_Failure_Invariants(state.Current_Failure, namespace)
}

// PUBLICATION_REFERENCE_COUNT follows one borrowed caller-owned publication.
const PUBLICATION_REFERENCE_COUNT = len("publication") / len("publication")

// Publication_Reference stores one stable pointer to large caller-owned parser state.
type Publication_Reference [PUBLICATION_REFERENCE_COUNT]*Publication

// Publication_Reference_Invariants fixes one non-nil borrowed parent reference.
func Publication_Reference_Invariants(
	value Publication_Reference, _ aver.Namespace,
) {
	aver.Always(
		value[0] != nil,
		"Secret parser retains one caller-owned publication reference.",
	)
}

// One state owns every completion for one declaration and never moves after submission.
type Secret_Parser struct {
	// Completion stays first so one static callback recovers caller-owned runner state.
	Completion nbio.Completion
	// Parser is the parent state that owns the terminal result.
	Parser Publication_Reference
	// Index is the secret declaration index.
	Index Secret_Index
	// Stage identifies the submitted operation retired by Completion.
	Stage Secret_Stage
	// State holds the phase-dependent file operation record.
	State Secret_State
}

// Secret_Parser_Invariants composes bounded per-declaration asynchronous state.
func Secret_Parser_Invariants(value Secret_Parser, namespace aver.Namespace) {
	aver.Always(
		unsafe.Pointer(&value) == unsafe.Pointer(&value.Completion),
		"Secret completion stays first so one static callback recovers its owner.",
	)
	Publication_Reference_Invariants(value.Parser, namespace)
	Secret_Index_Invariants(value.Index, namespace)
	Secret_Stage_Invariants(value.Stage, namespace)
	Secret_State_Invariants(value.State, namespace)
}

// The parent owns every resolved secret, so asynchronous state carries only its index.
func secret_parser_secret(state *Secret_Parser) (secret *Secret) {
	defer func() { Secret_Invariants(*secret, "secret_parser_secret.secret") }()
	Secret_Parser_Invariants(*state, "secret_parser_secret.state")
	return &state.Parser[0].Result.Command.Secrets[state.Index]
}

// The count is set before submission, so an all-absent declaration cannot publish too early.
func parser_start_secrets(
	reference Publication_Reference,
	parsers Secret_Parsers,
	buffers Secret_Buffers,
	path_failures Secret_Path_Failures,
) {
	Publication_Reference_Invariants(reference, "parser_start_secrets.reference")
	Secret_Parsers_Invariants(parsers, "parser_start_secrets.parsers")
	Secret_Buffers_Invariants(buffers, "parser_start_secrets.buffers")
	Secret_Path_Failures_Invariants(path_failures, "parser_start_secrets.path_failures")
	secret_count := int(reference[0].Secret_Count)
	if len(parsers) < secret_count {
		panic("Program_Parse Secret_Parsers storage is too small.")
	}
	if len(buffers) < secret_count {
		panic("Program_Parse Secret_Buffers storage is too small.")
	}
	if len(path_failures) < secret_count {
		panic("Program_Parse Secret_Path_Failures storage is too small.")
	}
	for index := range reference[0].Result.Command.Secrets[:secret_count] {
		if len(buffers[index]) != SECRET_BUFFER_BYTES_MAX {
			panic("Program_Parse secret buffer has the wrong size.")
		}
		secret := reference[0].Result.Command.Secrets[index]
		if len(path_failures[index]) < len(secret.Paths) {
			panic("Program_Parse secret path failure storage is too small.")
		}
		state := &parsers[index]
		*state = Secret_Parser{
			Parser: reference,
			Index:  Secret_Index(index),
			State: Secret_State{{
				Only_Absent:   true,
				Buffer:        buffers[index],
				Path_Failures: path_failures[index][:0],
			}},
		}
		secret_start_paths(state)
	}
}

// Synchronous status failures advance in one bounded loop; asynchronous work returns.
func secret_start_paths(state *Secret_Parser) {
	Secret_Parser_Invariants(*state, "secret_start_paths.state")
	secret := secret_parser_secret(state)
	phase := &state.State[0]
	for int(phase.Path_Index) < len(secret.Paths) {
		path := secret.Paths[phase.Path_Index]
		status, status_err := nbio.Storage_Status(state.Parser[0].Loop.Storage, path)
		if status_err != nil {
			secret_record_path(state, Path_Failure{
				Path:  Path_Failure_Path{path},
				Kind:  Path_Failure_Kind{PATH_FAILURE_KIND_STATUS},
				Cause: status_err,
			}, false)
			phase.Path_Index++
			continue
		}
		if !status.Exists {
			secret_record_path(state, Path_Failure{
				Path: Path_Failure_Path{path},
				Kind: Path_Failure_Kind{PATH_FAILURE_KIND_ABSENT},
			}, true)
			phase.Path_Index++
			continue
		}
		if !nbio.File_Mode_Is_Regular(status.Mode) {
			secret_record_path(state, Path_Failure{
				Path: Path_Failure_Path{path},
				Kind: Path_Failure_Kind{PATH_FAILURE_KIND_NOT_REGULAR},
			}, false)
			phase.Path_Index++
			continue
		}
		if status.Size < 0 {
			secret_record_path(state, Path_Failure{
				Path: Path_Failure_Path{path},
				Kind: Path_Failure_Kind{PATH_FAILURE_KIND_NEGATIVE_SIZE},
			}, false)
			phase.Path_Index++
			continue
		}
		if status.Size > SECRET_BYTES_MAX {
			secret_record_path(state, Path_Failure{
				Path: Path_Failure_Path{path},
				Kind: Path_Failure_Kind{PATH_FAILURE_KIND_FILE_TOO_LARGE},
			}, false)
			phase.Path_Index++
			continue
		}
		phase.Status_Size = Secret_Size(status.Size)
		state.Stage = Secret_Stage{SECRET_STAGE_OPEN_VALUE}
		nbio.Storage_Open_At(
			state.Parser[0].Loop.Storage, &state.Completion,
			nbio.DIRECTORY_CURRENT, path,
			nbio.Open_At_Options{
				Access: nbio.OPEN_READ_ONLY, Flags: nbio.OPEN_AT_NO_FOLLOW,
			},
			secret_operation_complete,
		)
		return
	}
	secret_exhausted(state)
}

// One static callback recovers stable caller storage and dispatches the submitted stage.
func secret_operation_complete(completed *nbio.Completion) {
	state := (*Secret_Parser)(unsafe.Pointer(completed))
	Secret_Parser_Invariants(*state, "secret_operation_complete.state")
	switch state.Stage[0] {
	case SECRET_STAGE_OPEN_VALUE:
		secret_opened(state, nbio.File(completed.Data), completed.Error)
	case SECRET_STAGE_READ_VALUE:
		secret_read(state, Secret_Read_Count(completed.Data), completed.Error)
	case SECRET_STAGE_CLOSE_VALUE:
		secret_closed(state, completed.Error)
	default:
		panic("Secret completion retired without a submitted stage.")
	}
}

// An open failure has no descriptor to close, so the next path can start immediately.
func secret_opened(state *Secret_Parser, file nbio.File, open_err error) {
	Secret_Parser_Invariants(*state, "secret_opened.state")
	secret := secret_parser_secret(state)
	phase := &state.State[0]
	path := secret.Paths[phase.Path_Index]
	if open_err != nil {
		secret_record_path(state, Path_Failure{
			Path:  Path_Failure_Path{path},
			Kind:  Path_Failure_Kind{PATH_FAILURE_KIND_OPEN},
			Cause: open_err,
		}, false)
		phase.Path_Index++
		secret_start_paths(state)
		return
	}
	if file < 0 {
		secret_record_path(state, Path_Failure{
			Path: Path_Failure_Path{path},
			Kind: Path_Failure_Kind{PATH_FAILURE_KIND_OPEN_NO_FILE},
		}, false)
		phase.Path_Index++
		secret_start_paths(state)
		return
	}
	phase.File = file
	state.Stage = Secret_Stage{SECRET_STAGE_READ_VALUE}
	nbio.Storage_Read(
		state.Parser[0].Loop.Storage, &state.Completion, file, phase.Buffer,
		0, time.DAY, secret_operation_complete,
	)
}

// Read analysis records a candidate, but Close must succeed before the parser accepts it.
func secret_read(
	state *Secret_Parser,
	count Secret_Read_Count,
	read_err error,
) {
	Secret_Parser_Invariants(*state, "secret_read.state")
	Secret_Read_Count_Invariants(count, "secret_read.count")
	phase := &state.State[0]
	phase.Current_Failure = Path_Failure{}
	phase.Current_Absent = false
	if read_err != nil {
		phase.Current_Failure = Path_Failure{
			Kind: Path_Failure_Kind{PATH_FAILURE_KIND_READ}, Cause: read_err,
		}
	} else if count < 0 {
		phase.Current_Failure.Kind = Path_Failure_Kind{PATH_FAILURE_KIND_NEGATIVE_READ}
	} else if count > SECRET_BYTES_MAX {
		phase.Current_Failure.Kind = Path_Failure_Kind{PATH_FAILURE_KIND_FILE_TOO_LARGE}
	} else if Secret_Size(count) != phase.Status_Size {
		phase.Current_Failure.Kind = Path_Failure_Kind{PATH_FAILURE_KIND_SIZE_CHANGED}
	} else {
		secret_convert_candidate(state, Secret_Size(count))
	}
	state.Stage = Secret_Stage{SECRET_STAGE_CLOSE_VALUE}
	nbio.IO_Close(
		state.Parser[0].Loop, &state.Completion, phase.File, secret_operation_complete,
	)
}

// The normalized bytes stay local to the state and never enter a secret error.
func secret_convert_candidate(state *Secret_Parser, count Secret_Size) {
	Secret_Parser_Invariants(*state, "secret_convert_candidate.state")
	Secret_Size_Invariants(count, "secret_convert_candidate.count")
	secret := secret_parser_secret(state)
	phase := &state.State[0]
	content_count := int(count)
	content := Secret_Value_Bytes(phase.Buffer[:content_count])
	if content_count >= len("\r\n") {
		if content[content_count-len("\r\n")] == '\r' {
			if content[content_count-len("\n")] == '\n' {
				content = content[:len(content)-len("\r\n")]
			}
		}
	}
	if content_count == len(content) {
		if content_count >= len("\n") {
			if content[content_count-len("\n")] == '\n' {
				content = content[:len(content)-1]
			}
		}
	}
	if len(content) == 0 {
		if !secret.Allow_Empty {
			phase.Current_Failure.Kind = Path_Failure_Kind{PATH_FAILURE_KIND_EMPTY}
			phase.Current_Absent = true
			return
		}
	}
	value, conversion_failure := external_convert_secret_state(
		Resolved_External_Key(secret.Key),
		secret.Type,
		secret.Enumeration,
		content,
	)
	if conversion_failure[0] != PATH_FAILURE_KIND_NONE {
		phase.Current_Failure.Kind = conversion_failure
		return
	}
	phase.Candidate = value
}

// A close failure invalidates otherwise usable content because ownership did not retire cleanly.
func secret_closed(state *Secret_Parser, close_err error) {
	Secret_Parser_Invariants(*state, "secret_closed.state")
	secret := secret_parser_secret(state)
	phase := &state.State[0]
	path := secret.Paths[phase.Path_Index]
	if close_err != nil {
		phase.Current_Failure.Close_Cause = close_err
		phase.Current_Absent = false
	}
	if phase.Current_Failure.Kind[0] == PATH_FAILURE_KIND_NONE {
		if phase.Current_Failure.Close_Cause == nil {
			secret.State = phase.Candidate
			if secret.Deprecated != "" {
				state.Parser[0].Secret_Warnings[state.Index] = Warning{
					Kind:     Warning_Kind{WARNING_KIND_SECRET},
					Name:     Warning_Name{Value_Text(secret.Key)},
					Guidance: Warning_Guidance{secret.Deprecated},
				}
			}
			secret_complete(state)
			return
		}
	}
	phase.Current_Failure.Path = Path_Failure_Path{path}
	secret_record_path(state, phase.Current_Failure, phase.Current_Absent)
	phase.Path_Index++
	secret_start_paths(state)
}

// One non-absence failure prevents an optional declaration from hiding operational faults.
func secret_record_path(
	state *Secret_Parser, failure Path_Failure, absent Current_Absent,
) {
	Secret_Parser_Invariants(*state, "secret_record_path.state")
	Path_Failure_Invariants(failure, "secret_record_path.failure")
	Current_Absent_Invariants(absent, "secret_record_path.absent")
	phase := &state.State[0]
	if len(phase.Path_Failures) == cap(phase.Path_Failures) {
		panic("Program_Parse secret path failure storage is too small.")
	}
	failure_count := len(phase.Path_Failures)
	phase.Path_Failures = phase.Path_Failures[:failure_count+1]
	phase.Path_Failures[failure_count] = failure
	if !absent {
		phase.Only_Absent = false
	}
}

// Required absence and optional operational failure both retain every ordered path cause.
func secret_exhausted(state *Secret_Parser) {
	Secret_Parser_Invariants(*state, "secret_exhausted.state")
	secret := secret_parser_secret(state)
	phase := &state.State[0]
	if secret.Required {
		state.Parser[0].Secret_Errors[state.Index] = Secret_Failure{
			Present: true, Paths: phase.Path_Failures,
		}
	} else if !phase.Only_Absent {
		state.Parser[0].Secret_Errors[state.Index] = Secret_Failure{
			Present: true, Paths: phase.Path_Failures,
		}
	}
	secret_complete(state)
}

// The last declaration publishes one stable result in declaration order.
func secret_complete(state *Secret_Parser) {
	Secret_Parser_Invariants(*state, "secret_complete.state")
	state.Parser[0].Secret_Count--
	if state.Parser[0].Secret_Count == 0 {
		parser_publish(state.Parser)
	}
}

// Indexed caller records preserve declaration and fallback order until final rendering.
func parser_publish(reference Publication_Reference) {
	Publication_Reference_Invariants(reference, "parser_publish.reference")
	parser := reference[0]
	failure_output := Failure_Reference{&parser.Failure}
	for index, failure := range parser.Secret_Errors {
		if !failure.Present {
			continue
		}
		if parser.Failure.Size[0] != 0 {
			failure_write(failure_output, Failure_Fragment{"\n"})
		}
		failure_write(failure_output, Failure_Fragment{"secret "})
		failure_write(
			failure_output,
			Failure_Fragment{string(parser.Result.Command.Secrets[index].Key)},
		)
		failure_write(failure_output, Failure_Fragment{": "})
		for path_index, path_failure := range failure.Paths {
			if path_index != 0 {
				failure_write(failure_output, Failure_Fragment{"\n"})
			}
			failure_write_path(failure_output,
				Failure_Secret_Key{parser.Result.Command.Secrets[index].Key},
				path_failure)
		}
	}
	warnings := Deprecation_Warnings(parser.Result.Command.Deprecation_Warnings)
	for _, warning := range parser.Environment_Warnings {
		warnings = warning_store(
			Writable_Deprecation_Warnings(warnings), warning,
		)
	}
	for _, warning := range parser.Secret_Warnings {
		if warning.Guidance[0] != "" {
			warnings = warning_store(
				Writable_Deprecation_Warnings(warnings), warning,
			)
		}
	}
	parser.Result.Command.Deprecation_Warnings = Resolved_Deprecation_Warnings(warnings)
	if parser.Failure.Size[0] != 0 {
		parser.Result.Error = &parser.Failure
	} else {
		parser.Result.Error = nil
	}
	parser.Completion[0] = true
}

func failure_write_path(
	failure Failure_Reference, key Failure_Secret_Key, path_failure Path_Failure,
) {
	Failure_Reference_Invariants(failure, "failure_write_path.failure")
	Failure_Secret_Key_Invariants(key, "failure_write_path.key")
	Path_Failure_Invariants(path_failure, "failure_write_path.path_failure")
	failure_write(failure, Failure_Fragment{path_failure.Path[0]})
	failure_write(failure, Failure_Fragment{": "})
	switch path_failure.Kind[0] {
	case PATH_FAILURE_KIND_NONE:
	case PATH_FAILURE_KIND_STATUS:
		failure_write(failure, Failure_Fragment{"status: "})
		failure_write_cause(failure, path_failure.Cause)
	case PATH_FAILURE_KIND_ABSENT:
		failure_write(failure, Failure_Fragment{"missing"})
	case PATH_FAILURE_KIND_NOT_REGULAR:
		failure_write(failure, Failure_Fragment{"not a regular file"})
	case PATH_FAILURE_KIND_NEGATIVE_SIZE:
		failure_write(failure, Failure_Fragment{"negative file size"})
	case PATH_FAILURE_KIND_FILE_TOO_LARGE:
		failure_write(failure, Failure_Fragment{"file exceeds 64 KiB"})
	case PATH_FAILURE_KIND_OPEN:
		failure_write(failure, Failure_Fragment{"open: "})
		failure_write_cause(failure, path_failure.Cause)
	case PATH_FAILURE_KIND_OPEN_NO_FILE:
		failure_write(failure, Failure_Fragment{"open returned no file"})
	case PATH_FAILURE_KIND_READ:
		failure_write(failure, Failure_Fragment{"read: "})
		failure_write_cause(failure, path_failure.Cause)
	case PATH_FAILURE_KIND_NEGATIVE_READ:
		failure_write(failure, Failure_Fragment{"read returned a negative byte count"})
	case PATH_FAILURE_KIND_SIZE_CHANGED:
		failure_write(failure, Failure_Fragment{"file size changed after status"})
	case PATH_FAILURE_KIND_EMPTY:
		failure_write(failure, Failure_Fragment{"file is empty"})
	case PATH_FAILURE_KIND_STRING_ENUM, PATH_FAILURE_KIND_INTEGER_ENUM:
		failure_write(failure, Failure_Fragment{string(key[0])})
		failure_write(failure, Failure_Fragment{" is not a permitted enum value"})
	case PATH_FAILURE_KIND_INTEGER:
		failure_write(failure, Failure_Fragment{string(key[0])})
		failure_write(failure, Failure_Fragment{" is not a whole number"})
	case PATH_FAILURE_KIND_BOOLEAN:
		failure_write(failure, Failure_Fragment{string(key[0])})
		failure_write(failure, Failure_Fragment{" is not a Boolean"})
	default:
		panic("Secret path has unknown failure kind.")
	}
	if path_failure.Close_Cause != nil {
		if path_failure.Kind[0] != PATH_FAILURE_KIND_NONE {
			failure_write(failure, Failure_Fragment{"\n"})
		}
		failure_write(failure, Failure_Fragment{"close: "})
		failure_write_cause(failure, path_failure.Close_Cause)
	}
}

func failure_write_cause(failure Failure_Reference, cause error) {
	Failure_Reference_Invariants(failure, "failure_write_cause.failure")
	aver.Always(cause != nil, "Rendered failure cause is present.")
	text := cause.Error()
	aver.Always(
		len(text) <= strings.TEXT_SIZE_MAXIMUM,
		"Injected failure cause stays inside shared text capacity.",
	)
	failure_write(failure, Failure_Fragment{text})
}

// Program-wide external inputs appear in command help and root help by the same rules.
func print_external_help(
	output Output_Reference,
	environment Environment_Declarations,
	secrets Secret_Declarations,
) {
	Output_Reference_Invariants(output, "print_external_help.output")
	Environment_Declarations_Invariants(environment, "print_external_help.environment")
	Secret_Declarations_Invariants(secrets, "print_external_help.secrets")
	print_environment_help(output, Environment_Variables(environment))
	print_secret_help(output, Secrets(secrets))
}

// Environment help can show defaults because environment values are not secrets.
func print_environment_help(
	output Output_Reference,
	environment Environment_Variables,
) {
	Output_Reference_Invariants(output, "print_environment_help.output")
	Environment_Variables_Invariants(environment, "print_environment_help.environment")
	has_shown := false
	for _, variable := range environment {
		if external_shown(variable.Hidden, variable.Deprecated) {
			has_shown = true
			break
		}
	}
	if !has_shown {
		return
	}
	first_width := Rendered_Size{}
	second_width := Rendered_Size{}
	for index := range environment {
		variable := environment[index]
		if !external_shown(variable.Hidden, variable.Deprecated) {
			continue
		}
		environment_legacy_initialize(&variable)
		rendered := Rendered_External{{Variable: variable}}
		first_size := Rendered_Size{len("    ") + len(variable.Key)}
		if first_size[0] > first_width[0] {
			first_width = first_size
		}
		second_size := external_type_size(variable.Type, variable.Enumeration)
		second_size[0] += len(" ") + len("optional")
		if !variable.Required {
			second_size[0] += len(" default=") + environment_default_size(rendered)[0]
		}
		if second_size[0] > second_width[0] {
			second_width = second_size
		}
	}
	output_write(output, "Environment Variables:\n")
	for _, variable := range environment {
		if !external_shown(variable.Hidden, variable.Deprecated) {
			continue
		}
		environment_legacy_initialize(&variable)
		rendered := Rendered_External{{Variable: variable}}
		first_size := Rendered_Size{len("    ") + len(variable.Key)}
		output_write(output, "    ")
		output_write(output, Value_Text(variable.Key))
		output_write_spaces(output, Rendered_Size{first_width[0] - first_size[0]})
		second_size := external_type_size(variable.Type, variable.Enumeration)
		second_size[0] += len(" ") + len("optional")
		external_type_write(output, variable.Type, variable.Enumeration)
		output_write(output, " ")
		if variable.Required {
			output_write(output, "required")
		} else {
			output_write(output, "optional default=")
			environment_default_write(output, rendered)
			second_size[0] += len(" default=") + environment_default_size(rendered)[0]
		}
		output_write_spaces(output, Rendered_Size{second_width[0] - second_size[0]})
		output_write(output, Value_Text(variable.Description))
		output_write(output, "\n")
	}
	output_write(output, "\n")
}

// Secret help shows paths and type constraints but never reads or formats Value.
func print_secret_help(output Output_Reference, secrets Secrets) {
	Output_Reference_Invariants(output, "print_secret_help.output")
	Secrets_Invariants(secrets, "print_secret_help.secrets")
	has_shown := false
	for _, secret := range secrets {
		if external_shown(secret.Hidden, secret.Deprecated) {
			has_shown = true
			break
		}
	}
	if !has_shown {
		return
	}
	first_width := Rendered_Size{}
	second_width := Rendered_Size{}
	for index := range secrets {
		secret := secrets[index]
		if !external_shown(secret.Hidden, secret.Deprecated) {
			continue
		}
		secret_legacy_initialize(&secret)
		first_size := Rendered_Size{len("    ") + len(secret.Key)}
		if first_size[0] > first_width[0] {
			first_width = first_size
		}
		second_size := external_type_size(secret.Type, secret.Enumeration)
		second_size[0] += len(" ") + len("optional")
		if second_size[0] > second_width[0] {
			second_width = second_size
		}
	}
	output_write(output, "Secrets:\n")
	for _, secret := range secrets {
		if !external_shown(secret.Hidden, secret.Deprecated) {
			continue
		}
		secret_legacy_initialize(&secret)
		first_size := Rendered_Size{len("    ") + len(secret.Key)}
		output_write(output, "    ")
		output_write(output, Value_Text(secret.Key))
		output_write_spaces(output, Rendered_Size{first_width[0] - first_size[0]})
		second_size := external_type_size(secret.Type, secret.Enumeration)
		second_size[0] += len(" ") + len("optional")
		external_type_write(output, secret.Type, secret.Enumeration)
		output_write(output, " ")
		if secret.Required {
			output_write(output, "required")
		} else {
			output_write(output, "optional")
		}
		output_write_spaces(output, Rendered_Size{second_width[0] - second_size[0]})
		output_write(output, Value_Text(secret.Description))
		output_write(output, "\n")
		for _, path := range secret.Paths {
			output_write(output, "        ")
			output_write(output, Value_Text(path))
			output_write(output, "\n")
		}
	}
	output_write(output, "\n")
}

// Hidden and deprecated declarations stay usable but do not advertise their names or paths.
func external_shown(hidden Hidden, deprecated Deprecation) (shown Boolean) {
	defer func() { Boolean_Invariants(shown, "external_shown.shown") }()
	Hidden_Invariants(hidden, "external_shown.hidden")
	Deprecation_Invariants(deprecated, "external_shown.deprecated")
	if hidden {
		return false
	}
	return deprecated == ""
}

func external_has_enum(enumeration External_Enumeration) (has Boolean) {
	defer func() { Boolean_Invariants(has, "external_has_enum.has") }()
	External_Enumeration_Invariants(enumeration, "external_has_enum.enumeration")
	if enumeration[0].String != nil {
		return true
	}
	return enumeration[0].Integers != nil
}

func external_enum_size(enumeration External_Enumeration) (size Rendered_Size) {
	defer func() { Rendered_Size_Invariants(size, "external_enum_size.size") }()
	External_Enumeration_Invariants(enumeration, "external_enum_size.enumeration")
	if enumeration[0].String != nil {
		for index, member := range enumeration[0].String {
			if index != 0 {
				size[0] += len("|")
			}
			size[0] += len(member)
		}
		return size
	}
	for index, member := range enumeration[0].Integers {
		if index != 0 {
			size[0] += len("|")
		}
		size[0] += decimal_size(Rendered_Integer{Integer(member)})[0]
	}
	return size
}

func external_enum_write(output Output_Reference, enumeration External_Enumeration) {
	Output_Reference_Invariants(output, "external_enum_write.output")
	External_Enumeration_Invariants(enumeration, "external_enum_write.enumeration")
	if enumeration[0].String != nil {
		for index, member := range enumeration[0].String {
			if index != 0 {
				output_write(output, "|")
			}
			output_write(output, Value_Text(member))
		}
		return
	}
	for index, member := range enumeration[0].Integers {
		if index != 0 {
			output_write(output, "|")
		}
		output_write_decimal(output, Rendered_Integer{Integer(member)})
	}
}

func external_type_size(
	type_state External_Type_State, enumeration External_Enumeration,
) (size Rendered_Size) {
	defer func() { Rendered_Size_Invariants(size, "external_type_size.size") }()
	External_Type_State_Invariants(type_state, "external_type_size.type_state")
	External_Enumeration_Invariants(enumeration, "external_type_size.enumeration")
	size[0] = len(option_type_text(type_state[0]))
	if external_has_enum(enumeration) {
		size[0] += len("()") + external_enum_size(enumeration)[0]
	}
	return size
}

func external_type_write(
	output Output_Reference, type_state External_Type_State, enumeration External_Enumeration,
) {
	Output_Reference_Invariants(output, "external_type_write.output")
	External_Type_State_Invariants(type_state, "external_type_write.type_state")
	External_Enumeration_Invariants(enumeration, "external_type_write.enumeration")
	output_write(output, Value_Text(option_type_text(type_state[0])))
	if external_has_enum(enumeration) {
		output_write(output, "(")
		external_enum_write(output, enumeration)
		output_write(output, ")")
	}
}

func environment_default_size(rendered Rendered_External) (size Rendered_Size) {
	defer func() {
		Rendered_Size_Invariants(size, "environment_default_size.size")
	}()
	Rendered_External_Invariants(rendered, "environment_default_size.rendered")
	variable := rendered[0].Variable
	switch Option_Type(variable.Type[0]) {
	case OPTION_TYPE_STRING:
		return Rendered_Size{len(variable.State[0].String)}
	case OPTION_TYPE_INTEGER:
		return decimal_size(Rendered_Integer{variable.State[0].Integer})
	case OPTION_TYPE_BOOLEAN:
		if variable.State[0].Boolean {
			return Rendered_Size{len("true")}
		}
		return Rendered_Size{len("false")}
	}
	panic("Environment default has no scalar type.")
}

func environment_default_write(output Output_Reference, rendered Rendered_External) {
	Output_Reference_Invariants(output, "environment_default_write.output")
	Rendered_External_Invariants(rendered, "environment_default_write.rendered")
	variable := rendered[0].Variable
	switch Option_Type(variable.Type[0]) {
	case OPTION_TYPE_STRING:
		output_write(output, Value_Text(variable.State[0].String))
	case OPTION_TYPE_INTEGER:
		output_write_decimal(output, Rendered_Integer{variable.State[0].Integer})
	case OPTION_TYPE_BOOLEAN:
		if variable.State[0].Boolean {
			output_write(output, "true")
		} else {
			output_write(output, "false")
		}
	default:
		panic("Environment default has no scalar type.")
	}
}
