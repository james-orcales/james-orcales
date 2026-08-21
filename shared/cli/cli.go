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
//	cli.Program_Parse(program, &parser, cli.Program_Parse_Input{Arguments: os.Args})
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
// PROGRAM_MODE_SINGLE removes the command selector, so the first token after the
// program name is the first positional argument.
//
// A command's last argument may be variadic (New_Option), collecting zero or more
// trailing positionals into a slice, as in `sloc ./a ./b ./c`.
//
// New_Option accepts a fixed enum set; parsing rejects anything outside it.
package cli

import (
	"errors"

	"local/james-orcales/shared/diff/levenshtein"
	"local/james-orcales/shared/filepath"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/path"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/sim/nbio"
	"local/james-orcales/shared/sim/time"
	"local/james-orcales/shared/slices"
	"local/james-orcales/shared/strconv"
	"local/james-orcales/shared/strings"
)

// HELP_REQUESTED identifies -h or -help without interface identity.
// Help has priority over an absent argument. Compare returned text with this constant.
// Then call Print_Requested_Help with the returned command.
const HELP_REQUESTED = "help requested"

// Unsupported_Completion_Shell reports a shell with no static registration syntax.
var Unsupported_Completion_Shell = errors.New("unsupported shell; use bash, zsh, or fish")

// Parse failure marks caller diagnostic storage as populated.
var parse_failure = errors.New("cli parse failed")

// Static identities keep validation allocation-free until one boundary renders caller storage.
var environment_malformed_failure = errors.New("environment source is malformed")

var environment_duplicate_failure = errors.New("environment source is duplicated")

var environment_missing_failure = errors.New("required environment source is missing")

var external_string_enum_failure = errors.New("environment text is outside enum")

var external_integer_enum_failure = errors.New("environment integer is outside enum")

var external_integer_parse_failure = errors.New("environment value is not an integer")

var external_boolean_parse_failure = errors.New("environment value is not a Boolean")

// HELP_LABEL is the reserved label for the default long help flag.
const HELP_LABEL = "help"

// HELP_SHORT_LABEL is the reserved label for the default short help flag.
const HELP_SHORT_LABEL = "h"

// HELP_DESCRIPTION is fixed reserved help guidance.
const HELP_DESCRIPTION = "show this help"

// HELP_LONG_LABEL_SIZE_MAXIMUM follows fixed long-help identity.
const HELP_LONG_LABEL_SIZE_MAXIMUM = len(HELP_LABEL)

// HELP_SHORT_LABEL_SIZE_MAXIMUM follows fixed short-help identity.
const HELP_SHORT_LABEL_SIZE_MAXIMUM = len(HELP_SHORT_LABEL)

// HELP_DESCRIPTION_SIZE_MAXIMUM follows fixed reserved help guidance.
const HELP_DESCRIPTION_SIZE_MAXIMUM = len(HELP_DESCRIPTION)

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

// SECRET_KEY_SIZE_MAXIMUM reserves shortest absolute-path prefix inside host path bound.
const SECRET_KEY_SIZE_MAXIMUM = filepath.PATH_SIZE_MAXIMUM - len("/")

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

// Resolved_Secret_Key is one validated nonempty file identity.
type Resolved_Secret_Key string

// Resolved_Secret_Key_Invariants excludes constructor-only invalid empty input.
func Resolved_Secret_Key_Invariants(
	value Resolved_Secret_Key, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), EXTERNAL_KEY_SIZE_MINIMUM, SECRET_KEY_SIZE_MAXIMUM,
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

// DISPLAY_OPTION_NAME_COUNT isolates diagnostic bounds from declaration bounds.
const DISPLAY_OPTION_NAME_COUNT = len("name") / len("name")

// Display_Option_Name is one option identity held for diagnostic rendering.
type Display_Option_Name struct {
	// Value is declaration identity without syntax.
	Value Option_Name
}

// Display_Option_Name_Invariants admits both argument and flag label bounds.
func Display_Option_Name_Invariants(
	value Display_Option_Name, namespace aver.Namespace,
) {
	Option_Name_Invariants(value.Value, namespace)
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

// Indent records whether a help row belongs beneath a command.
type Indent bool

// Indent_Invariants requires both help row nesting levels.
func Indent_Invariants(value Indent, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Help row is nested beneath a command.").
		Ensure()
}

// INDENT_FLAG is one ordinary flag row.
const INDENT_FLAG Indent = false

// INDENT_COMMAND_FLAG is one flag nested under a command catalog row.
const INDENT_COMMAND_FLAG Indent = true

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
const OPTION_TYPE_STRING = 0

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
type Option_Type_State struct {
	// Value selects active option representation.
	Value Option_Type
}

// Option_Type_State_Invariants bounds one intrinsically fixed union cell.
func Option_Type_State_Invariants(value Option_Type_State, namespace aver.Namespace) {
	Option_Type_Invariants(value.Value, namespace)
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

// Secret_Path_Cursor also admits the exhausted one-past-last state.
type Secret_Path_Cursor int

// Secret_Path_Cursor_Invariants bounds fallback progress.
func Secret_Path_Cursor_Invariants(value Secret_Path_Cursor, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), slices.POSITION_MINIMUM, slices.COUNT_MAXIMUM).
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

// Active_Secret_Count excludes initialization without secret declarations.
type Active_Secret_Count int

// Active_Secret_Count_Invariants bounds initialized secret work.
func Active_Secret_Count_Invariants(value Active_Secret_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), NONEMPTY_COUNT_MINIMUM, slices.COUNT_MAXIMUM,
		).
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

// Resolved_Options is a bounded borrowed parsed option collection.
type Resolved_Options []Resolved_Option

// Resolved_Options_Invariants bounds parsed option lookup work.
func Resolved_Options_Invariants(value Resolved_Options, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Found_Resolved_Options excludes empty storage after successful named lookup.
type Found_Resolved_Options []Resolved_Option

// Found_Resolved_Options_Invariants bounds one selected parsed option collection.
func Found_Resolved_Options_Invariants(
	value Found_Resolved_Options, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), NONEMPTY_COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// OPTION_REFERENCE_COUNT follows one borrowed mutable option.
const OPTION_REFERENCE_COUNT = len("option") / len("option")

// Option_Handle is one live caller-owned option.
type Option_Handle *Option

// Option_Handle_Invariants composes present option storage.
func Option_Handle_Invariants(value Option_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Option_Invariants(*value, namespace)
}

// Option_Reference stores one stable pointer into caller-owned parse storage.
type Option_Reference struct {
	// Value points into caller-owned parse storage.
	Value Option_Handle
}

// Option_Reference_Invariants fixes one non-nil borrowed option reference.
func Option_Reference_Invariants(value Option_Reference, namespace aver.Namespace) {
	Option_Handle_Invariants(value.Value, namespace)
}

// Output_Builder gives foreign fixed storage one direct invariant edge.
type Output_Builder strings.Builder

// Output_Builder_Invariants fixes storage capacity while preserving initialized size bounds.
func Output_Builder_Invariants(value Output_Builder, namespace aver.Namespace) {
	Output_Storage_Invariants(Output_Storage(value.Storage), namespace)
	strings.Size_Value_Invariants(value.Size, namespace)
	aver.Always(
		int(value.Size) <= len(value.Storage),
		"CLI output size does not exceed caller storage.",
	)
}

// Output_Storage keeps formatting capacity in caller-owned Output state.
type Output_Storage []byte

// Output_Storage_Invariants fixes capacity without inspecting unused bytes.
func Output_Storage_Invariants(value Output_Storage, _ aver.Namespace) {
	aver.Always(
		len(value) == strings.TEXT_SIZE_MAXIMUM,
		"CLI output owns complete shared text capacity.",
	)
}

// Output keeps bounded formatting storage in caller ownership.
type Output struct {
	// Builder fixes output capacity without hiding a growing allocation.
	Builder Output_Builder
}

// Output_Invariants composes fixed caller formatting storage.
func Output_Invariants(value Output, namespace aver.Namespace) {
	Output_Builder_Invariants(value.Builder, namespace)
}

// Output_Handle retains caller ownership while first use binds fixed storage.
type Output_Handle *Output

// Output_Handle_Invariants composes state after a nil-safe ownership check.
func Output_Handle_Invariants(value Output_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Output_Invariants(*value, namespace)
}

// OUTPUT_REFERENCE_COUNT follows one borrowed caller-owned output.
const OUTPUT_REFERENCE_COUNT = len("output") / len("output")

// Output_Reference stores one stable pointer to caller-owned formatting state.
type Output_Reference struct {
	// Value points to caller-owned formatting state.
	Value Builder_Handle
}

// Output_Reference_Invariants fixes one non-nil borrowed output reference.
func Output_Reference_Invariants(value Output_Reference, namespace aver.Namespace) {
	Builder_Handle_Invariants(value.Value, namespace)
}

// Output_Reference_Of borrows initialized caller-owned builder state.
func Output_Reference_Of(output Output_Handle) (reference Output_Reference) {
	defer func() { Output_Reference_Invariants(reference, "output_reference.reference") }()
	Output_Handle_Invariants(output, "output_reference.output")
	return Output_Reference{
		Value: Builder_Handle(&output.Builder),
	}
}

// Output_Write preserves fixed builder capacity without attaching an interface method.
func Output_Write(
	output Output_Reference, source strings.Bytes,
) (written strings.Size_Value) {
	defer func() { strings.Size_Value_Invariants(written, "output_write.written") }()
	Output_Reference_Invariants(output, "output_write.output")
	strings.Bytes_Invariants(source, "output_write.source")
	return strings.Builder_Write(
		strings.Builder_Handle((*strings.Builder)((*Output_Builder)(output.Value))),
		source,
	)
}

// Output_Bytes exposes initialized output without copying it into a string.
func Output_Bytes(output Output_Reference) (content strings.Bytes) {
	defer func() { strings.Bytes_Invariants(content, "output_bytes.content") }()
	Output_Reference_Invariants(output, "output_bytes.output")
	return strings.Builder_Bytes(
		strings.Builder_Handle((*strings.Builder)((*Output_Builder)(output.Value))),
	)
}

// Output_Reset preserves storage while removing initialized output.
func Output_Reset(output Output_Reference) {
	Output_Reference_Invariants(output, "output_reset.output")
	strings.Builder_Reset(strings.Builder_Handle(
		(*strings.Builder)((*Output_Builder)(output.Value)),
	))
}

// Output_Write_Text avoids formatting machinery for already-rendered text.
func Output_Write_Text(output Output_Reference, source Value_Text) {
	Output_Reference_Invariants(output, "output_write_text.output")
	Value_Text_Invariants(source, "output_write_text.source")
	strings.Builder_Write_Text(
		strings.Builder_Handle((*strings.Builder)((*Output_Builder)(output.Value))),
		strings.Text(source),
	)
}

// FAILURE_SIZE_MAXIMUM follows widest CLI diagnostic: enum value, match, and name.
const FAILURE_SIZE_MAXIMUM = len("invalid value  for -, did you mean ?") +
	2*strconv.QUOTED_TEXT_SIZE_MAXIMUM + OPTION_LABEL_SIZE_MAXIMUM

// Failure_Storage borrows exact diagnostic capacity from the parse caller.
type Failure_Storage []byte

// Failure_Storage_Invariants admits zero parser state and initialized caller storage.
func Failure_Storage_Invariants(value Failure_Storage, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Int(len(value), strings.TEXT_SIZE_MINIMUM, FAILURE_SIZE_MAXIMUM).
		Ensure()
}

// Parse_Failure_Storage excludes the parser's pre-initialization state.
type Parse_Failure_Storage []byte

// Parse_Failure_Storage_Invariants requires enough caller space for every diagnostic.
func Parse_Failure_Storage_Invariants(value Parse_Failure_Storage, _ aver.Namespace) {
	aver.Always(
		len(value) == FAILURE_SIZE_MAXIMUM,
		"Program parse failure storage has exact diagnostic capacity.",
	)
}

// FAILURE_SIZE_COUNT follows one initialized diagnostic byte count.
const FAILURE_SIZE_COUNT = len("size") / len("size")

// Failure_Size_Value carries checked diagnostic arithmetic.
type Failure_Size_Value int

// Failure_Size_Value_Invariants bounds initialized diagnostic bytes.
func Failure_Size_Value_Invariants(value Failure_Size_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), strings.TEXT_SIZE_MINIMUM, FAILURE_SIZE_MAXIMUM).
		Ensure()
}

// Failure_Size selects initialized diagnostic bytes.
type Failure_Size struct {
	// Value is initialized diagnostic byte count.
	Value Failure_Size_Value
}

// Failure_Size_Invariants bounds initialized diagnostic bytes.
func Failure_Size_Invariants(value Failure_Size, namespace aver.Namespace) {
	Failure_Size_Value_Invariants(value.Value, namespace)
}

// FAILURE_FRAGMENT_COUNT follows one borrowed diagnostic part.
const FAILURE_FRAGMENT_COUNT = len("fragment") / len("fragment")

// Failure_Fragment is one bounded borrowed diagnostic part.
type Failure_Fragment string

// Failure_Fragment_Invariants checks only write-time storage safety.
func Failure_Fragment_Invariants(value Failure_Fragment, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// FAILURE_SECRET_KEY_COUNT isolates rendered identity from declaration phases.
const FAILURE_SECRET_KEY_COUNT = len("key") / len("key")

// Failure_Secret_Key is one validated secret identity rendered in a diagnostic.
type Failure_Secret_Key struct {
	// Value is validated secret identity.
	Value Secret_Key
}

// Failure_Secret_Key_Invariants follows keys produced from valid absolute paths.
func Failure_Secret_Key_Invariants(
	value Failure_Secret_Key, namespace aver.Namespace,
) {
	Secret_Key_Invariants(value.Value, namespace)
}

// FAILURE_INTEGER_COUNT follows one scalar diagnostic cell.
const FAILURE_INTEGER_COUNT = len("integer") / len("integer")

// Failure_Integer is one scalar rendered inside a diagnostic.
type Failure_Integer struct {
	// Value is scalar diagnostic input.
	Value Integer
}

// Failure_Integer_Invariants admits machine integer diagnostic input.
func Failure_Integer_Invariants(value Failure_Integer, namespace aver.Namespace) {
	Integer_Invariants(value.Value, namespace)
}

// Failure_String_Set excludes empty enums that construction rejects.
type Failure_String_Set []string

// Failure_String_Set_Invariants bounds diagnostic set work.
func Failure_String_Set_Invariants(value Failure_String_Set, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), NONEMPTY_COUNT_MINIMUM, slices.COUNT_MAXIMUM,
		).
		Ensure()
}

// Failure_Integer_Set excludes empty enums that construction rejects.
type Failure_Integer_Set []int

// Failure_Integer_Set_Invariants bounds diagnostic set work.
func Failure_Integer_Set_Invariants(value Failure_Integer_Set, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), NONEMPTY_COUNT_MINIMUM, slices.COUNT_MAXIMUM,
		).
		Ensure()
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

// Failure_Size_Handle keeps mutation on the cursor rather than the whole diagnostic.
type Failure_Size_Handle *Failure_Size

// Failure_Size_Handle_Invariants composes a present bounded cursor.
func Failure_Size_Handle_Invariants(
	value Failure_Size_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Failure_Size_Invariants(*value, namespace)
}

// Exact storage stays separate so writer boundaries never admit absent parser state.
type Failure_Writer struct {
	// Storage is the caller-owned complete diagnostic destination.
	Storage Parse_Failure_Storage
	// Size is the only shared mutable writer state.
	Size Failure_Size_Handle
}

// Failure_Writer_Invariants keeps every write inside exact caller storage.
func Failure_Writer_Invariants(value Failure_Writer, namespace aver.Namespace) {
	Parse_Failure_Storage_Invariants(value.Storage, namespace)
	Failure_Size_Handle_Invariants(value.Size, namespace)
}

func failure_write(reference Failure_Writer, text Failure_Fragment) {
	Failure_Writer_Invariants(reference, "failure_write.reference")
	Failure_Fragment_Invariants(text, "failure_write.text")
	start := reference.Size.Value
	end := start + Failure_Size_Value(len(text))
	aver.Always(
		int(end) <= len(reference.Storage),
		"CLI failure stays inside fixed diagnostic storage.",
	)
	copy(reference.Storage[start:end], text)
	reference.Size.Value = end
	if int(end) < len(reference.Storage) {
		reference.Storage[end] = 0
	}
}

func failure_write_quoted(reference Failure_Writer, text Failure_Fragment) {
	Failure_Writer_Invariants(reference, "failure_write_quoted.reference")
	Failure_Fragment_Invariants(text, "failure_write_quoted.text")
	start := reference.Size.Value
	end := start + Failure_Size_Value(strconv.QUOTED_TEXT_SIZE_MAXIMUM)
	aver.Always(
		int(end) <= len(reference.Storage),
		"Quoted CLI failure stays inside fixed diagnostic storage.",
	)
	count := strconv.Quote_Into(
		strconv.Buffer(reference.Storage[start:end]), strconv.Text(text),
	)
	end = start + Failure_Size_Value(count)
	reference.Size.Value = end
	if int(end) < len(reference.Storage) {
		reference.Storage[end] = 0
	}
}

func failure_write_decimal(reference Failure_Writer, value Failure_Integer) {
	Failure_Writer_Invariants(reference, "failure_write_decimal.reference")
	Failure_Integer_Invariants(value, "failure_write_decimal.value")
	var storage [strconv.DECIMAL_TEXT_SIZE_MAXIMUM]byte
	count := strconv.Format_Decimal_Into(
		storage[:], strconv.Machine_Integer(value.Value),
	)
	start := reference.Size.Value
	end := start + Failure_Size_Value(count)
	aver.Always(
		int(end) <= len(reference.Storage),
		"Decimal CLI failure stays inside fixed diagnostic storage.",
	)
	copy(reference.Storage[start:end], storage[:count])
	reference.Size.Value = end
	if int(end) < len(reference.Storage) {
		reference.Storage[end] = 0
	}
}

func failure_write_string_set(
	failure Failure_Writer, members Failure_String_Set,
) {
	Failure_Writer_Invariants(failure, "failure_write_string_set.failure")
	Failure_String_Set_Invariants(members, "failure_write_string_set.members")
	size := 0
	for index, member := range members {
		if index != 0 {
			size += len(", ")
		}
		size += len(member)
	}
	aver.Always(
		size <= strings.TEXT_SIZE_MAXIMUM,
		"Text enum diagnostic stays inside shared text capacity.",
	)
	for index, member := range members {
		if index != 0 {
			failure_write(failure, Failure_Fragment(", "))
		}
		failure_write(failure, Failure_Fragment(member))
	}
}

func failure_write_integer_set(
	failure Failure_Writer, members Failure_Integer_Set,
) {
	Failure_Writer_Invariants(failure, "failure_write_integer_set.failure")
	Failure_Integer_Set_Invariants(members, "failure_write_integer_set.members")
	size := 0
	for index, member := range members {
		if index != 0 {
			size += len(", ")
		}
		var storage [strconv.DECIMAL_TEXT_SIZE_MAXIMUM]byte
		size += int(strconv.Format_Decimal_Into(
			storage[:], strconv.Machine_Integer(member),
		))
	}
	aver.Always(
		size <= strings.TEXT_SIZE_MAXIMUM,
		"Integer enum diagnostic stays inside shared text capacity.",
	)
	for index, member := range members {
		if index != 0 {
			failure_write(failure, Failure_Fragment(", "))
		}
		failure_write_decimal(failure, Failure_Integer{Value: Integer(member)})
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

// Resolved_Arguments is caller-owned parsed positional storage.
type Resolved_Arguments []Resolved_Option

// Resolved_Arguments_Invariants bounds parsed positional assignment work.
func Resolved_Arguments_Invariants(
	value Resolved_Arguments, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Variadic_Arguments excludes commands without trailing collection storage.
type Variadic_Arguments []Option

// Variadic_Arguments_Invariants bounds commands with one selected variadic argument.
func Variadic_Arguments_Invariants(
	value Variadic_Arguments, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), NONEMPTY_COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Resolved_Variadic_Arguments excludes parses without collection storage.
type Resolved_Variadic_Arguments []Resolved_Option

// Resolved_Variadic_Arguments_Invariants bounds one selected variadic argument.
func Resolved_Variadic_Arguments_Invariants(
	value Resolved_Variadic_Arguments, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), NONEMPTY_COUNT_MINIMUM, slices.COUNT_MAXIMUM).
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

// Resolved_Flags is caller-owned parsed command flag storage.
type Resolved_Flags []Resolved_Option

// Resolved_Flags_Invariants bounds parsed command flag search work.
func Resolved_Flags_Invariants(value Resolved_Flags, namespace aver.Namespace) {
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

// Resolved_Global_Flags is caller-owned parsed program flag storage.
type Resolved_Global_Flags []Resolved_Option

// Resolved_Global_Flags_Invariants bounds parsed global flag search work.
func Resolved_Global_Flags_Invariants(
	value Resolved_Global_Flags, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Help_Short_Label is bounded reserved short-help identity.
type Help_Short_Label string

// Help_Short_Label_Invariants bounds short-help rendering.
func Help_Short_Label_Invariants(value Help_Short_Label, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Int(
			len(value), strings.TEXT_SIZE_MINIMUM,
			HELP_SHORT_LABEL_SIZE_MAXIMUM,
		).
		Ensure()
}

// Help_Long_Label is bounded reserved long-help identity.
type Help_Long_Label string

// Help_Long_Label_Invariants bounds long-help rendering.
func Help_Long_Label_Invariants(value Help_Long_Label, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Int(
			len(value), strings.TEXT_SIZE_MINIMUM,
			HELP_LONG_LABEL_SIZE_MAXIMUM,
		).
		Ensure()
}

// Help_Description is shared fixed help text.
type Help_Description string

// Help_Description_Invariants bounds help row rendering.
func Help_Description_Invariants(value Help_Description, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Int(
			len(value), strings.TEXT_SIZE_MINIMUM,
			HELP_DESCRIPTION_SIZE_MAXIMUM,
		).
		Ensure()
}

// Help_Hidden records whether internal fixtures suppress reserved flags.
type Help_Hidden bool

// Help_Hidden_Invariants requires visible and hidden reserved flags.
func Help_Hidden_Invariants(value Help_Hidden, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Reserved help flags are hidden.").
		Ensure()
}

// HELP_HIDDEN_VISIBLE keeps reserved flags publicly discoverable.
const HELP_HIDDEN_VISIBLE Help_Hidden = false

// HELP_HIDDEN_INTERNAL suppresses reserved flags in internal fixtures.
const HELP_HIDDEN_INTERNAL Help_Hidden = true

// Help_Hidden_Unvalidated preserves numeric fixture input until validation.
type Help_Hidden_Unvalidated uint8

// Help_Hidden_Unvalidated_Invariants admits every hostile input byte.
func Help_Hidden_Unvalidated_Invariants(
	value Help_Hidden_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(HELP_HIDDEN_ENCODED_VISIBLE),
			uint8(HELP_HIDDEN_ENCODED_INTERNAL),
		).
		Ensure()
}

// HELP_HIDDEN_ENCODED_VISIBLE preserves zero-value fixture declarations.
const HELP_HIDDEN_ENCODED_VISIBLE Help_Hidden_Unvalidated = 0

// HELP_HIDDEN_ENCODED_INTERNAL preserves existing numeric fixture declarations.
const HELP_HIDDEN_ENCODED_INTERNAL Help_Hidden_Unvalidated = 1

// Help_Flags is fixed program-owned reserved option metadata.
type Help_Flags struct {
	// Short_Label is reserved short-help identity.
	Short_Label Help_Short_Label
	// Long_Label is reserved long-help identity.
	Long_Label Help_Long_Label
	// Description is shared by both generated options.
	Description Help_Description
	// Hidden remains numeric because existing fixture declarations use 0 and 1.
	Hidden Help_Hidden_Unvalidated
}

// Help_Flags_Invariants composes reserved help metadata.
func Help_Flags_Invariants(value Help_Flags, namespace aver.Namespace) {
	Help_Short_Label_Invariants(value.Short_Label, namespace)
	Help_Long_Label_Invariants(value.Long_Label, namespace)
	Help_Description_Invariants(value.Description, namespace)
	Help_Hidden_Unvalidated_Invariants(value.Hidden, namespace)
}

// Numeric compatibility stays at declaration boundary; internal branches use Boolean state.
func help_hidden_validate(value Help_Hidden_Unvalidated) (hidden Help_Hidden) {
	defer func() { Help_Hidden_Invariants(hidden, "help_hidden_validate.hidden") }()
	Help_Hidden_Unvalidated_Invariants(value, "help_hidden_validate.value")
	aver.Always(
		value <= HELP_HIDDEN_ENCODED_INTERNAL,
		"Reserved help visibility stays inside its encoded domain.",
	)
	return value == HELP_HIDDEN_ENCODED_INTERNAL
}

// Help_Labels keeps validated reserved-option discovery state outside rendering metadata.
type Help_Labels struct {
	// Short is reserved short-help identity.
	Short Help_Short_Label
	// Long is reserved long-help identity.
	Long Help_Long_Label
	// Hidden is validated before option discovery.
	Hidden Help_Hidden
}

// Help_Labels_Invariants composes reserved-option discovery state.
func Help_Labels_Invariants(value Help_Labels, namespace aver.Namespace) {
	Help_Short_Label_Invariants(value.Short, namespace)
	Help_Long_Label_Invariants(value.Long, namespace)
	Help_Hidden_Invariants(value.Hidden, namespace)
}

// SINGLE_COMMAND_COUNT follows the only selector-free command identity.
const SINGLE_COMMAND_COUNT = len("single") / len("single")

// Single_Commands is fixed inline storage for the one selector-free command.
type Single_Commands struct {
	// Command is selector-free command.
	Command Command
}

// Single_Commands_Invariants keeps Program independent of borrowed command-slice storage.
func Single_Commands_Invariants(value Single_Commands, namespace aver.Namespace) {
	Command_Invariants(value.Command, namespace)
}

// PROGRAM_MODE_STORAGE_COUNT follows the one mutually exclusive selection mode.
const PROGRAM_MODE_STORAGE_COUNT = len("mode") / len("mode")

// Program_Mode_Value is one bounded selector discriminant.
type Program_Mode_Value byte

// Program_Mode_Value_Invariants rejects bytes outside selector modes.
func Program_Mode_Value_Invariants(value Program_Mode_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(PROGRAM_MODE_COMMANDS), uint8(PROGRAM_MODE_SINGLE),
			uint8(PROGRAM_MODE_MULTICALL),
		).
		Ensure()
}

// Program_Mode stores one bounded command-selection mode without contradictory bits.
type Program_Mode struct {
	// Value selects command resolution mode.
	Value Program_Mode_Value
}

// Program_Mode_Invariants rejects bytes beyond the three public modes.
func Program_Mode_Invariants(value Program_Mode, namespace aver.Namespace) {
	Program_Mode_Value_Invariants(value.Value, namespace)
}

// PROGRAM_MODE_COMMANDS selects a command from an argument token.
const PROGRAM_MODE_COMMANDS Program_Mode_Value = 0

// PROGRAM_MODE_SINGLE has one selector-free command.
const PROGRAM_MODE_SINGLE = PROGRAM_MODE_COMMANDS + 1

// PROGRAM_MODE_MULTICALL selects a command from the binary name.
const PROGRAM_MODE_MULTICALL = PROGRAM_MODE_SINGLE + 1

// Command_Argument_Storage is caller-owned active argument storage.
type Command_Argument_Storage []Resolved_Option

// Command_Argument_Storage_Invariants bounds active argument copying.
func Command_Argument_Storage_Invariants(
	value Command_Argument_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Command_Flag_Storage is caller-owned active command flag storage.
type Command_Flag_Storage []Resolved_Option

// Command_Flag_Storage_Invariants bounds active flag copying.
func Command_Flag_Storage_Invariants(
	value Command_Flag_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Global_Flag_Storage is caller-owned active global flag storage.
type Global_Flag_Storage []Resolved_Option

// Global_Flag_Storage_Invariants bounds active global flag copying.
func Global_Flag_Storage_Invariants(
	value Global_Flag_Storage, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Environment_Variables is bounded caller-owned resolved environment storage.
type Environment_Variables []Resolved_Environment_Variable

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

// Warning_Kind_Value is one bounded warning grammar selector.
type Warning_Kind_Value uint8

// Warning_Kind_Value_Invariants admits each warning grammar.
func Warning_Kind_Value_Invariants(value Warning_Kind_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(WARNING_KIND_COMMAND), uint8(WARNING_KIND_FLAG),
			uint8(WARNING_KIND_ENVIRONMENT), uint8(WARNING_KIND_SECRET),
		).
		Ensure()
}

// Warning_Kind selects one deprecation message grammar.
type Warning_Kind struct {
	// Value selects source grammar.
	Value Warning_Kind_Value
}

// Warning_Kind_Invariants admits command, flag, environment, and secret sources.
func Warning_Kind_Invariants(value Warning_Kind, namespace aver.Namespace) {
	Warning_Kind_Value_Invariants(value.Value, namespace)
}

// WARNING_KIND_COMMAND selects quoted command grammar.
const WARNING_KIND_COMMAND Warning_Kind_Value = 0

// WARNING_KIND_FLAG selects dashed option grammar.
const WARNING_KIND_FLAG = WARNING_KIND_COMMAND + 1

// WARNING_KIND_ENVIRONMENT selects environment-variable grammar.
const WARNING_KIND_ENVIRONMENT = WARNING_KIND_FLAG + 1

// WARNING_KIND_SECRET selects secret grammar.
const WARNING_KIND_SECRET = WARNING_KIND_ENVIRONMENT + 1

// WARNING_NAME_COUNT follows one borrowed declaration identity.
const WARNING_NAME_COUNT = len("name") / len("name")

// Warning_Name keeps one source identity in phase-neutral storage.
type Warning_Name struct {
	// Value is borrowed declaration identity.
	Value Value_Text
}

// Warning_Name_Invariants bounds a borrowed warning identity structurally.
func Warning_Name_Invariants(value Warning_Name, namespace aver.Namespace) {
	Value_Text_Invariants(value.Value, namespace)
}

// WARNING_GUIDANCE_COUNT follows one borrowed migration instruction.
const WARNING_GUIDANCE_COUNT = len("guidance") / len("guidance")

// Warning_Guidance keeps migration text in phase-neutral storage.
type Warning_Guidance struct {
	// Value is borrowed migration text.
	Value Deprecation
}

// Warning_Guidance_Invariants bounds borrowed guidance structurally.
func Warning_Guidance_Invariants(value Warning_Guidance, namespace aver.Namespace) {
	Deprecation_Invariants(value.Value, namespace)
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

// CANDIDATE_STATE_COUNT follows one scalar completion content cell.
const CANDIDATE_STATE_COUNT = len("state") / len("state")

// Candidate_State holds integer completion content outside the string union.
type Candidate_State struct {
	// Integer is rendered after Segments when Has_Integer is set.
	Integer Integer
	// Has_Integer distinguishes zero from absent integer content.
	Has_Integer Boolean
}

// Candidate_State_Invariants bounds the inactive or active scalar cell.
func Candidate_State_Invariants(value Candidate_State, namespace aver.Namespace) {
	Integer_Invariants(value.Integer, namespace)
	Boolean_Invariants(value.Has_Integer, namespace)
}

// Candidate_Dash is bounded leading syntax or command text.
type Candidate_Dash string

// Candidate_Dash_Invariants protects deferred leading rendering.
func Candidate_Dash_Invariants(value Candidate_Dash, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Candidate_Label is one bounded option identity.
type Candidate_Label string

// Candidate_Label_Invariants protects deferred label rendering.
func Candidate_Label_Invariants(value Candidate_Label, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Candidate_Equals is bounded named-value syntax.
type Candidate_Equals string

// Candidate_Equals_Invariants protects deferred equals rendering.
func Candidate_Equals_Invariants(value Candidate_Equals, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// ENUM_CANDIDATE_SYNTAX_SIZE follows absent or one-byte enum completion syntax.
const ENUM_CANDIDATE_SYNTAX_SIZE = len("-")

// Enum_Candidate_Dash is absent or one named-option dash.
type Enum_Candidate_Dash string

// Enum_Candidate_Dash_Invariants bounds enum completion prefix syntax.
func Enum_Candidate_Dash_Invariants(
	value Enum_Candidate_Dash, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Int(len(value), strings.TEXT_SIZE_MINIMUM, ENUM_CANDIDATE_SYNTAX_SIZE).
		Ensure()
}

// Enum_Candidate_Equals is absent or one named-option separator.
type Enum_Candidate_Equals string

// Enum_Candidate_Equals_Invariants bounds enum completion separator syntax.
func Enum_Candidate_Equals_Invariants(
	value Enum_Candidate_Equals, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Int(len(value), strings.TEXT_SIZE_MINIMUM, ENUM_CANDIDATE_SYNTAX_SIZE).
		Ensure()
}

// Candidate_Value is one bounded enum member.
type Candidate_Value string

// Candidate_Value_Invariants protects deferred value rendering.
func Candidate_Value_Invariants(value Candidate_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Candidate_Segments preserve rendered order without an owned joined string.
type Candidate_Segments struct {
	// Dash holds named-option syntax.
	Dash Candidate_Dash
	// Label holds command or option identity.
	Label Candidate_Label
	// Equals holds named-value syntax.
	Equals Candidate_Equals
	// Value holds one textual enum member.
	Value Candidate_Value
}

// Candidate_Segments_Invariants bounds each independently borrowed fragment.
func Candidate_Segments_Invariants(
	value Candidate_Segments, namespace aver.Namespace,
) {
	Candidate_Dash_Invariants(value.Dash, namespace)
	Candidate_Label_Invariants(value.Label, namespace)
	Candidate_Equals_Invariants(value.Equals, namespace)
	Candidate_Value_Invariants(value.Value, namespace)
}

// Candidate is one completion assembled from borrowed segments or one integer scalar.
type Candidate struct {
	// Segments preserve rendered order without owning joined text.
	Segments Candidate_Segments
	// State keeps scalar union storage in one invariant cell.
	State Candidate_State
}

// Candidate_Invariants bounds each borrowed fragment; joining stays deferred.
func Candidate_Invariants(value Candidate, namespace aver.Namespace) {
	Candidate_State_Invariants(value.State, namespace)
	Candidate_Segments_Invariants(value.Segments, namespace)
}

// Candidate_Parts is one fixed borrowed rendering sequence.
type Candidate_Parts []string

// Candidate_Parts_Invariants fixes rendering order storage.
func Candidate_Parts_Invariants(value Candidate_Parts, _ aver.Namespace) {
	aver.Always(
		len(value) == CANDIDATE_SEGMENT_COUNT,
		"Candidate rendering keeps every text part.",
	)
}

// Candidate_Number stores one formatted integer without allocation.
type Candidate_Number []byte

// Candidate_Number_Invariants fixes decimal scratch storage.
func Candidate_Number_Invariants(value Candidate_Number, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Int(len(value), strings.TEXT_SIZE_MINIMUM, strconv.DECIMAL_TEXT_SIZE_MAXIMUM).
		Ensure()
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

// Assigned_Options records command assignment presence state.
type Assigned_Options []bool

// Assigned_Options_Invariants includes commands without options.
func Assigned_Options_Invariants(value Assigned_Options, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
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

// PATH_FAILURE_KIND_NONE marks unused current failure state.
const PATH_FAILURE_KIND_NONE Path_Failure_Kind_Value = 0

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

// Path_Failure_Kind_Value is one bounded redacted diagnostic selector.
type Path_Failure_Kind_Value uint8

// Path_Failure_Kind_Value_Invariants rejects unknown failure grammars.
func Path_Failure_Kind_Value_Invariants(
	value Path_Failure_Kind_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(PATH_FAILURE_KIND_NONE),
			uint8(PATH_FAILURE_KIND_BOOLEAN),
		).
		Ensure()
}

// Path_Failure_Kind selects one redacted path diagnostic.
type Path_Failure_Kind struct {
	// Value selects redacted failure grammar.
	Value Path_Failure_Kind_Value
}

// Path_Failure_Kind_Invariants admits unused and every redacted cause.
func Path_Failure_Kind_Invariants(
	value Path_Failure_Kind, namespace aver.Namespace,
) {
	Path_Failure_Kind_Value_Invariants(value.Value, namespace)
}

// PATH_FAILURE_PATH_COUNT isolates borrowed path bounds from secret declaration bounds.
const PATH_FAILURE_PATH_COUNT = len("path") / len("path")

// SECRET_PATH_SIZE_MINIMUM follows slash plus uppercase base name.
const SECRET_PATH_SIZE_MINIMUM = len("/X")

// Resolved_Secret_Path excludes malformed paths rejected during construction.
type Resolved_Secret_Path string

// Resolved_Secret_Path_Invariants bounds validated fallback identity.
func Resolved_Secret_Path_Invariants(
	value Resolved_Secret_Path, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SECRET_PATH_SIZE_MINIMUM, filepath.PATH_SIZE_MAXIMUM).
		Ensure()
}

// Path_Failure_Path is one borrowed secret path identity.
type Path_Failure_Path struct {
	// Value is borrowed path identity.
	Value Resolved_Secret_Path
}

// Path_Failure_Path_Invariants bounds diagnostic path identity.
func Path_Failure_Path_Invariants(
	value Path_Failure_Path, namespace aver.Namespace,
) {
	Resolved_Secret_Path_Invariants(value.Value, namespace)
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

// WRITABLE_PATH_FAILURE_COUNT_MAXIMUM leaves one fallback publication cell.
const WRITABLE_PATH_FAILURE_COUNT_MAXIMUM = slices.COUNT_MAXIMUM - NONEMPTY_COUNT_MINIMUM

// Writable_Path_Failures excludes full storage before one failure append.
type Writable_Path_Failures []Path_Failure

// Writable_Path_Failures_Invariants proves one fallback cell remains.
func Writable_Path_Failures_Invariants(
	value Writable_Path_Failures, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), slices.COUNT_MINIMUM, WRITABLE_PATH_FAILURE_COUNT_MAXIMUM,
		).
		Ensure()
}

// NONEMPTY_COUNT_MINIMUM follows one present element.
const NONEMPTY_COUNT_MINIMUM = len("element") / len("element")

// Active_Secret_Failures excludes parser states with no active declaration.
type Active_Secret_Failures []Secret_Failure

// Active_Secret_Failures_Invariants bounds active publication storage.
func Active_Secret_Failures_Invariants(
	value Active_Secret_Failures, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), NONEMPTY_COUNT_MINIMUM, slices.COUNT_MAXIMUM,
		).
		Ensure()
}

// Failed_Paths excludes terminal states before one attempted path.
type Failed_Paths []Path_Failure

// Failed_Paths_Invariants bounds one failed fallback sequence.
func Failed_Paths_Invariants(
	value Failed_Paths, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), NONEMPTY_COUNT_MINIMUM, slices.COUNT_MAXIMUM,
		).
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

// Active_Secret_Warnings excludes publication without secret declarations.
type Active_Secret_Warnings []Warning

// Active_Secret_Warnings_Invariants bounds active warning storage.
func Active_Secret_Warnings_Invariants(
	value Active_Secret_Warnings, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), NONEMPTY_COUNT_MINIMUM, slices.COUNT_MAXIMUM,
		).
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

// Secret_Buffer excludes parser state before caller storage is attached.
type Secret_Buffer []byte

// Secret_Buffer_Invariants requires the overflow witness byte during reads.
func Secret_Buffer_Invariants(value Secret_Buffer, _ aver.Namespace) {
	aver.Always(
		len(value) == SECRET_BUFFER_BYTES_MAX,
		"Secret reads retain one byte beyond accepted content.",
	)
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

// Active_Secret_Buffers excludes initialization without secret declarations.
type Active_Secret_Buffers []Secret_Bytes

// Active_Secret_Buffers_Invariants bounds active read storage.
func Active_Secret_Buffers_Invariants(
	value Active_Secret_Buffers, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), NONEMPTY_COUNT_MINIMUM, slices.COUNT_MAXIMUM,
		).
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

// Active_Secret_Parsers excludes initialization without secret declarations.
type Active_Secret_Parsers []Secret_Parser

// Active_Secret_Parsers_Invariants bounds active runner storage.
func Active_Secret_Parsers_Invariants(
	value Active_Secret_Parsers, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), NONEMPTY_COUNT_MINIMUM, slices.COUNT_MAXIMUM,
		).
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

// Active_Secret_Path_Failures excludes initialization without secret declarations.
type Active_Secret_Path_Failures []Path_Failures

// Active_Secret_Path_Failures_Invariants bounds active fallback storage.
func Active_Secret_Path_Failures_Invariants(
	value Active_Secret_Path_Failures, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), NONEMPTY_COUNT_MINIMUM, slices.COUNT_MAXIMUM,
		).
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
type Program_Selection struct {
	// Commands is selector-owned command storage.
	Commands Command_Declarations
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
	Command_Declarations_Invariants(value.Commands, namespace)
	Single_Commands_Invariants(value.Single_Commands, namespace)
	Global_Flags_Invariants(value.Global_Flags, namespace)
	Help_Flags_Invariants(value.Help_Flags, namespace)
	Program_Mode_Invariants(value.Mode, namespace)
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
type Resolved_Environment []Resolved_Environment_Variable

// Resolved_Environment_Invariants bounds resolved environment structurally.
func Resolved_Environment_Invariants(
	value Resolved_Environment, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Resolved_Secrets is phase-produced secret storage.
type Resolved_Secrets []Resolved_Secret

// Resolved_Secrets_Invariants bounds resolved secrets structurally.
func Resolved_Secrets_Invariants(value Resolved_Secrets, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Active_Secrets excludes parser phases without secret declarations.
type Active_Secrets []Resolved_Secret

// Active_Secrets_Invariants bounds active secret resolution.
func Active_Secrets_Invariants(value Active_Secrets, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), NONEMPTY_COUNT_MINIMUM, slices.COUNT_MAXIMUM,
		).
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
}

// Command_Invariants composes command identity and option declarations.
func Command_Invariants(value Command, namespace aver.Namespace) {
	Label_Invariants(value.Label, namespace)
	Description_Invariants(value.Description, namespace)
	Arguments_Invariants(value.Arguments, namespace)
	Flags_Invariants(value.Flags, namespace)
	Hidden_Invariants(value.Hidden, namespace)
	Deprecation_Invariants(value.Deprecated, namespace)
}

// Selected_Command adds caller-owned option copies to one selected declaration.
type Selected_Command struct {
	// Label is the selected command identity.
	Label Label
	// Description retains selected command metadata.
	Description Description
	// Arguments are caller-owned parsed positional values.
	Arguments Resolved_Arguments
	// Flags are caller-owned parsed command flags.
	Flags Resolved_Flags
	// Hidden retains selected command visibility.
	Hidden Hidden
	// Deprecated retains selected command guidance.
	Deprecated Deprecation
}

// Selected_Command_Invariants composes identity and caller-owned option state.
func Selected_Command_Invariants(value Selected_Command, namespace aver.Namespace) {
	Label_Invariants(value.Label, namespace)
	Description_Invariants(value.Description, namespace)
	Resolved_Arguments_Invariants(value.Arguments, namespace)
	Resolved_Flags_Invariants(value.Flags, namespace)
	Hidden_Invariants(value.Hidden, namespace)
	Deprecation_Invariants(value.Deprecated, namespace)
}

// Parsed_Command contains only state produced before external sources are read.
type Parsed_Command struct {
	// Selected_Command retains caller-owned option state.
	Selected_Command
	// Deprecation_Warnings records declaration warnings found during token parsing.
	Deprecation_Warnings Resolved_Deprecation_Warnings
}

// Parsed_Command_Invariants excludes state owned by later external resolution.
func Parsed_Command_Invariants(value Parsed_Command, namespace aver.Namespace) {
	Selected_Command_Invariants(value.Selected_Command, namespace)
	Resolved_Deprecation_Warnings_Invariants(value.Deprecation_Warnings, namespace)
}

// Resolved_Command adds external values to one parsed command.
type Resolved_Command struct {
	// Parsed_Command retains selected options and deprecation warnings.
	Parsed_Command
	// Environment contains the resolved program-wide environment variables.
	Environment Resolved_Environment
	// Secrets contains the resolved program-wide file-backed secrets.
	Secrets Resolved_Secrets
}

// Resolved_Command_Invariants composes parsed and external state.
func Resolved_Command_Invariants(value Resolved_Command, namespace aver.Namespace) {
	Parsed_Command_Invariants(value.Parsed_Command, namespace)
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

// Resolved_Option adds parser-owned values to immutable option metadata.
type Resolved_Option struct {
	// Label retains the declaration identity.
	Label Option_Label
	// Description retains help metadata.
	Description Description
	// Type selects parsed storage.
	Type Option_Type_State
	// Enumeration retains permitted values.
	Enumeration Option_Enumeration
	// State contains values owned by this parse.
	State Resolved_Option_State
}

// Resolved_Option_Invariants composes parsed metadata and value storage.
func Resolved_Option_Invariants(value Resolved_Option, namespace aver.Namespace) {
	Option_Label_Invariants(value.Label, namespace)
	Description_Invariants(value.Description, namespace)
	Option_Type_State_Invariants(value.Type, namespace)
	Option_Enumeration_Invariants(value.Enumeration, namespace)
	Resolved_Option_State_Invariants(value.State, namespace)
}

// OPTION_ENUMERATION_COUNT follows one fixed enum union cell.
const OPTION_ENUMERATION_COUNT = len("enum") / len("enum")

// Option_Enumeration keeps typed permitted sets outside interface storage.
type Option_Enumeration struct {
	// String is permitted text set.
	String String_Enumeration
	// Integers is permitted integer set.
	Integers Integer_Enumeration
}

// Option_Enumeration_Invariants bounds both inactive or active borrowed arms.
func Option_Enumeration_Invariants(value Option_Enumeration, namespace aver.Namespace) {
	String_Enumeration_Invariants(value.String, namespace)
	Integer_Enumeration_Invariants(value.Integers, namespace)
}

// OPTION_STATE_COUNT follows the one active option state record.
const OPTION_STATE_COUNT = len("state") / len("state")

// Option_State keeps declaration defaults and flag metadata together.
type Option_State struct {
	// String is the text default.
	String Value_Text
	// Integer is the integer default.
	Integer Integer
	// Boolean is the Boolean default.
	Boolean Boolean
	// Is_Flag distinguishes named and positional declarations.
	Is_Flag Is_Flag
	// Hidden controls public discovery.
	Hidden Hidden
	// Deprecated carries migration guidance.
	Deprecated Deprecation
}

// Option_State_Invariants enforces union storage without phase-specific branch trees.
func Option_State_Invariants(value Option_State, namespace aver.Namespace) {
	Value_Text_Invariants(value.String, namespace)
	Integer_Invariants(value.Integer, namespace)
	Boolean_Invariants(value.Boolean, namespace)
	Is_Flag_Invariants(value.Is_Flag, namespace)
	Hidden_Invariants(value.Hidden, namespace)
	Deprecation_Invariants(value.Deprecated, namespace)
}

// Resolved_Option_State owns values and metadata copied for one parse.
type Resolved_Option_State struct {
	// String is parsed text scalar.
	String Value_Text
	// Integer is parsed machine integer scalar.
	Integer Integer
	// Boolean is parsed Boolean scalar.
	Boolean Boolean
	// Strings are parsed text collection values.
	Strings String_Values
	// Integers are parsed machine integer collection values.
	Integers Integer_Values
	// Parsed distinguishes a default from supplied input.
	Parsed Parsed
	// Is_Flag distinguishes named and positional declarations.
	Is_Flag Is_Flag
	// Hidden controls public discovery.
	Hidden Hidden
	// Deprecated carries migration guidance.
	Deprecated Deprecation
}

// Resolved_Option_State_Invariants composes parser-owned option state.
func Resolved_Option_State_Invariants(
	value Resolved_Option_State, namespace aver.Namespace,
) {
	Value_Text_Invariants(value.String, namespace)
	Integer_Invariants(value.Integer, namespace)
	Boolean_Invariants(value.Boolean, namespace)
	String_Values_Invariants(value.Strings, namespace)
	Integer_Values_Invariants(value.Integers, namespace)
	Parsed_Invariants(value.Parsed, namespace)
	Is_Flag_Invariants(value.Is_Flag, namespace)
	Hidden_Invariants(value.Hidden, namespace)
	Deprecation_Invariants(value.Deprecated, namespace)
}

// Each parse owns option values while declarations remain reusable.
func copy_options(
	input Options, storage Resolved_Options,
) (output Resolved_Options) {
	defer func() { Resolved_Options_Invariants(output, "copy_options.output") }()
	Options_Invariants(input, "copy_options.input")
	Resolved_Options_Invariants(storage, "copy_options.storage")
	aver.Always(
		len(storage) >= len(input),
		"Program_Parse option storage fits declarations.",
	)
	output = storage[:len(input)]
	for index := range input {
		declaration := input[index]
		output[index] = Resolved_Option{
			Label: declaration.Label, Description: declaration.Description,
			Type: declaration.Type, Enumeration: declaration.Enumeration,
			State: Resolved_Option_State{
				String:     declaration.State.String,
				Integer:    declaration.State.Integer,
				Boolean:    declaration.State.Boolean,
				Is_Flag:    declaration.State.Is_Flag,
				Hidden:     declaration.State.Hidden,
				Deprecated: declaration.State.Deprecated,
			},
		}
	}
	return output
}

// New_Option_Input holds one concrete option declaration.
type New_Option_Input struct {
	// Label shares the one option namespace.
	Label Option_Label
	// Description stays metadata rather than parsed state.
	Description Description
	// String_Enum stays borrowed for the text arm.
	String_Enum String_Enumeration
	// Integer_Enum stays borrowed for the integer arm.
	Integer_Enum Integer_Enumeration
	// String is the text default.
	String Value_Text
	// Integer is the integer default.
	Integer Integer
	// Boolean is the Boolean default.
	Boolean Boolean
	// Type selects scalar or collection storage.
	Type Option_Type
	// Is_Flag distinguishes a named flag from a positional.
	Is_Flag Is_Flag
	// Hidden affects advertising without disabling exact parsing.
	Hidden Hidden
	// Deprecated preserves compatibility while suppressing advertising.
	Deprecated Deprecation
}

// New_Option_Input_Invariants composes concrete option metadata.
func New_Option_Input_Invariants(value New_Option_Input, namespace aver.Namespace) {
	Option_Label_Invariants(value.Label, namespace)
	Description_Invariants(value.Description, namespace)
	String_Enumeration_Invariants(value.String_Enum, namespace)
	Integer_Enumeration_Invariants(value.Integer_Enum, namespace)
	Value_Text_Invariants(value.String, namespace)
	Integer_Invariants(value.Integer, namespace)
	Boolean_Invariants(value.Boolean, namespace)
	Option_Type_Invariants(value.Type, namespace)
	Is_Flag_Invariants(value.Is_Flag, namespace)
	Hidden_Invariants(value.Hidden, namespace)
	Deprecation_Invariants(value.Deprecated, namespace)
}

// New_Option creates one positional or flag without runtime type discovery.
func New_Option(input New_Option_Input) (option Option) {
	defer func() { Option_Invariants(option, "new_option.option") }()
	New_Option_Input_Invariants(input, "new_option.input")
	return Option{
		Label: input.Label, Description: input.Description,
		Type: Option_Type_State{Value: input.Type},
		Enumeration: Option_Enumeration{
			String: input.String_Enum, Integers: input.Integer_Enum,
		},
		State: Option_State{
			String: input.String, Integer: input.Integer, Boolean: input.Boolean,
			Is_Flag: input.Is_Flag, Hidden: input.Hidden,
			Deprecated: input.Deprecated,
		},
	}
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
	Environment_Variables Environment_Declarations
	// Secrets are process-wide file-backed external-value declarations.
	Secrets Secrets
	// Commands are the program's commands.
	Commands Command_Declarations
	// Arguments belong to the selector-free command.
	Arguments Arguments
	// Flags belong to the selector-free command.
	Flags Flags
	// Hidden removes the selector-free command from discovery.
	Hidden Hidden
	// Deprecated retains selector-free command migration guidance.
	Deprecated Deprecation
	// Help_Hidden suppresses generated flags in internal fixtures.
	Help_Hidden Help_Hidden_Unvalidated
	// Mode selects token, selector-free, or binary-name command resolution.
	Mode Program_Mode_Value
}

// New_Input_Invariants composes bounded multi-command declarations.
func New_Input_Invariants(value New_Input, namespace aver.Namespace) {
	Label_Invariants(value.Label, namespace)
	Description_Invariants(value.Description, namespace)
	Global_Flags_Invariants(value.Global_Flags, namespace)
	Environment_Declarations_Invariants(value.Environment_Variables, namespace)
	Secrets_Invariants(value.Secrets, namespace)
	Command_Declarations_Invariants(value.Commands, namespace)
	Arguments_Invariants(value.Arguments, namespace)
	Flags_Invariants(value.Flags, namespace)
	Hidden_Invariants(value.Hidden, namespace)
	Deprecation_Invariants(value.Deprecated, namespace)
	Help_Hidden_Unvalidated_Invariants(value.Help_Hidden, namespace)
	Program_Mode_Value_Invariants(value.Mode, namespace)
}

// New keeps every selection mode in one constructor invariant chain.
func New(input New_Input) (program Program) {
	defer func() { Program_Invariants(program, "new.program") }()
	New_Input_Invariants(input, "new.input")
	external_validate(input.Environment_Variables, input.Secrets)
	if input.Mode == PROGRAM_MODE_SINGLE {
		command := Command{
			Label: input.Label, Description: input.Description,
			Arguments: input.Arguments, Flags: input.Flags,
			Hidden: input.Hidden, Deprecated: input.Deprecated,
		}
		reserve_help_command(command.Arguments, command.Flags)
		command_validate_options(command.Label, command.Arguments, command.Flags, nil)
		return Program{
			Label:       Program_Label(input.Label),
			Description: Program_Description(input.Description),
			Selection: Program_Selection{
				Single_Commands: Single_Commands{Command: command},
				Help_Flags: Help_Flags{
					Short_Label: HELP_SHORT_LABEL, Long_Label: HELP_LABEL,
					Description: HELP_DESCRIPTION, Hidden: input.Help_Hidden,
				},
				Mode: Program_Mode{Value: input.Mode},
			},
			Environment_Variables: input.Environment_Variables,
			Secrets:               Secret_Declarations(input.Secrets),
		}
	}
	aver.Always(len(input.Commands) > 0, "Program has at least one command.")

	// The injection occurs before validation because a reserved label must give a clear error.
	reserve_help_labels(Commands(input.Commands), input.Global_Flags)
	global_flags := input.Global_Flags

	for index := range global_flags {
		flag := global_flags[index]
		aver.Always(flag.State.Is_Flag, "Global flags use flag declarations.")
		validate_flag_label(flag.Label, Scalar_Option_Type(flag.Type.Value))
		validate_enum(Option_Name(flag.Label), Option_Enum{
			Type: flag.Type, Enumeration: flag.Enumeration,
			String: flag.State.String, Integer: flag.State.Integer,
			Is_Flag: flag.State.Is_Flag,
		})
	}

	program = Program{
		Label:       Program_Label(input.Label),
		Description: Program_Description(input.Description),
		Selection: Program_Selection{
			Commands: input.Commands, Global_Flags: global_flags,
			Help_Flags: Help_Flags{
				Short_Label: HELP_SHORT_LABEL, Long_Label: HELP_LABEL,
				Description: HELP_DESCRIPTION, Hidden: input.Help_Hidden,
			},
			Mode: Program_Mode{Value: input.Mode},
		},
		Environment_Variables: input.Environment_Variables,
		Secrets:               Secret_Declarations(input.Secrets),
	}
	for _, command := range program.Selection.Commands {
		aver.Always(command.Label != "", "Program command label is set.")
		command_validate_options(
			command.Label, command.Arguments, command.Flags, global_flags,
		)
	}
	return program
}

// A program cannot replace a default help flag with a different function.
func reserve_help_labels(commands Commands, global_flags Global_Flags) {
	Commands_Invariants(commands, "reserve_help_labels.commands")
	Global_Flags_Invariants(global_flags, "reserve_help_labels.global_flags")
	for _, flag := range global_flags {
		aver.Always(
			!help_label_reserved(flag.Label),
			"Global flag label does not replace an injected help flag.",
		)
	}
	for _, command := range commands {
		for _, argument := range command.Arguments {
			aver.Always(
				!help_label_reserved(argument.Label),
				"Command argument label does not replace an injected help flag.",
			)
		}
		for _, flag := range command.Flags {
			aver.Always(
				!help_label_reserved(flag.Label),
				"Command flag label does not replace an injected help flag.",
			)
		}
	}
}

func reserve_help_command(arguments Arguments, flags Flags) {
	Arguments_Invariants(arguments, "reserve_help_command.arguments")
	Flags_Invariants(flags, "reserve_help_command.flags")
	for _, argument := range arguments {
		aver.Always(
			!help_label_reserved(argument.Label),
			"Single command argument label does not replace an injected help flag.",
		)
	}
	for _, flag := range flags {
		aver.Always(
			!help_label_reserved(flag.Label),
			"Single command flag label does not replace an injected help flag.",
		)
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
		argument := arguments[argument_index]
		option_validate_label(argument.Label)
		// A slice argument absorbs every remaining positional, so anything declared
		// after it is unreachable; requiring it last also forbids a second slice.
		is_last := argument_index == len(arguments)-1
		if option_is_slice(Option_Type(argument.Type.Value)) {
			aver.Always(is_last, "Slice argument is last.")
		}
		supported := false
		switch argument.Type.Value {
		case OPTION_TYPE_STRING, OPTION_TYPE_INTEGER,
			OPTION_TYPE_STRINGS, OPTION_TYPE_INTEGERS:
			supported = true
		}
		aver.Always(supported, "Command argument type is supported.")
		validate_enum(Option_Name(argument.Label), Option_Enum{
			Type: argument.Type, Enumeration: argument.Enumeration,
			String: argument.State.String, Integer: argument.State.Integer,
			Is_Flag: argument.State.Is_Flag,
		})
		collision := command_option_label_seen(
			arguments, flags, global_flags, Option_Name(argument.Label),
			Option_Index(argument_index), true,
		)
		aver.Always(!collision, "Command argument label is unique.")
	}
	for flag_index := range flags {
		flag := flags[flag_index]
		validate_flag_label(flag.Label, Scalar_Option_Type(flag.Type.Value))
		validate_enum(Option_Name(flag.Label), Option_Enum{
			Type: flag.Type, Enumeration: flag.Enumeration,
			String: flag.State.String, Integer: flag.State.Integer,
			Is_Flag: flag.State.Is_Flag,
		})
		collision := command_option_label_seen(
			arguments, flags, global_flags, Option_Name(flag.Label),
			Option_Index(flag_index), false,
		)
		aver.Always(!collision, "Command flag label is unique.")
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
func option_validate_label(label Option_Label) {
	Option_Label_Invariants(label, "option_validate_label.label")
	aver.Always(label != "", "Option has a label.")
	aver.Always(
		!strings.Contains(strings.Text(label), "_"),
		"Option labels cannot contain underscores; use dashes.",
	)
	aver.Always(
		!strings.Contains(strings.Text(label), " "),
		"Option label contains no spaces.",
	)
}

// Panics when a flag's label is invalid or its value is not a supported scalar type.
func validate_flag_label(label Option_Label, type_value Scalar_Option_Type) {
	Option_Label_Invariants(label, "validate_flag_label.label")
	Scalar_Option_Type_Invariants(type_value, "validate_flag_label.type_value")
	option_validate_label(label)
	supported := false
	switch Option_Type(type_value) {
	case OPTION_TYPE_STRING, OPTION_TYPE_BOOLEAN, OPTION_TYPE_INTEGER:
		supported = true
	}
	aver.Always(supported, "Flag type is supported.")
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
	// String is the text default checked for flag membership.
	String Value_Text
	// Integer is the integer default checked for flag membership.
	Integer Integer
	// Is_Flag distinguishes defaults from required positional values.
	Is_Flag Is_Flag
}

// Option_Enum_Invariants composes only state used after label validation.
func Option_Enum_Invariants(value Option_Enum, namespace aver.Namespace) {
	Option_Type_State_Invariants(value.Type, namespace)
	Option_Enumeration_Invariants(value.Enumeration, namespace)
	Value_Text_Invariants(value.String, namespace)
	Integer_Invariants(value.Integer, namespace)
	Is_Flag_Invariants(value.Is_Flag, namespace)
}

func validate_enum(label Option_Name, option Option_Enum) {
	Option_Name_Invariants(label, "validate_enum.label")
	Option_Enum_Invariants(option, "validate_enum.option")
	if option.Enumeration.String == nil {
		if option.Enumeration.Integers == nil {
			return
		}
	}
	if option.Enumeration.String != nil {
		aver.Always(
			option.Enumeration.Integers == nil,
			"Option has only one enum representation.",
		)
	}
	aver.Always(
		!option_is_slice(Option_Type(option.Type.Value)),
		"Enum option is scalar.",
	)
	if option.Enumeration.String != nil {
		aver.Always(
			option.Type.Value == OPTION_TYPE_STRING,
			"Text enum has text option storage.",
		)
		aver.Always(len(option.Enumeration.String) > 0, "Text option enum is not empty.")
		if option.Is_Flag {
			aver.Always(
				slices.Contains(option.Enumeration.String, string(option.String)),
				"Text flag default belongs to its enum.",
			)
		}
		return
	}
	aver.Always(
		option.Type.Value == OPTION_TYPE_INTEGER,
		"Integer enum has integer option storage.",
	)
	aver.Always(len(option.Enumeration.Integers) > 0, "Integer option enum is not empty.")
	if option.Is_Flag {
		aver.Always(
			slices.Contains(option.Enumeration.Integers, int(option.Integer)),
			"Integer flag default belongs to its enum.",
		)
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
	selection Program_Selection, global_flags Resolved_Global_Flags,
	operating_system_args Process_Arguments,
	workspace Workspace, failure_storage Parse_Failure_Storage,
) (parsed Parsed_Command, err error) {
	defer func() { Parsed_Command_Invariants(parsed, "program_parse_arguments.command") }()
	Program_Selection_Invariants(selection, "program_parse_arguments.selection")
	Resolved_Global_Flags_Invariants(global_flags, "program_parse_arguments.global_flags")
	Workspace_Invariants(workspace, "program_parse_arguments.workspace")
	Parse_Failure_Storage_Invariants(failure_storage, "program_parse_arguments.failure_storage")
	Process_Arguments_Invariants(
		operating_system_args, "program_parse_arguments.operating_system_args")
	aver.Always(len(operating_system_args) > 0, "Program_Parse receives at least one os_arg.")
	arguments := Parse_Arguments(operating_system_args)
	// Help wins before command resolution, so incomplete input still renders its resolved
	// context through Print_Requested_Help.
	if program_wants_help(Parsed_Tokens(arguments[1:])) {
		command := program_help_context(
			selection.Commands, selection.Mode, Help_Arguments(arguments))
		record := Failure{Storage: Failure_Storage(failure_storage)}
		failure := Failure_Writer{Storage: failure_storage, Size: &record.Size}
		failure_write(failure, Failure_Fragment(HELP_REQUESTED))
		return Parsed_Command{
			Selected_Command: Selected_Command{
				Label: command.Label, Description: command.Description,
				Hidden: command.Hidden, Deprecated: command.Deprecated,
			},
		}, parse_failure
	}
	active_command, command_selected, err := program_resolve_command(selection.Commands,
		selection.Single_Commands, selection.Mode, arguments, workspace, failure_storage)
	if err != nil {
		return Parsed_Command{Selected_Command: active_command}, err
	}
	tokens := Parsed_Tokens(arguments[1:])
	if command_selected {
		tokens = tokens[1:]
	}
	warnings, err := program_parse_selected(
		global_flags, Help_Labels{
			Short:  selection.Help_Flags.Short_Label,
			Long:   selection.Help_Flags.Long_Label,
			Hidden: help_hidden_validate(selection.Help_Flags.Hidden),
		}, active_command, tokens, workspace, failure_storage,
	)
	if err != nil {
		return Parsed_Command{Selected_Command: active_command}, err
	}
	parsed = Parsed_Command{
		Selected_Command: active_command, Deprecation_Warnings: warnings,
	}
	return parsed, nil
}

// Assignment mutates only the caller-owned option copies selected for this parse.
func program_parse_selected(
	global_flags Resolved_Global_Flags, help_labels Help_Labels,
	command Selected_Command, tokens Parsed_Tokens,
	workspace Workspace, failure_storage Parse_Failure_Storage,
) (warnings Resolved_Deprecation_Warnings, err error) {
	defer func() {
		Resolved_Deprecation_Warnings_Invariants(
			warnings, "program_parse_selected.warnings",
		)
	}()
	Resolved_Global_Flags_Invariants(global_flags, "program_parse_selected.global_flags")
	Help_Labels_Invariants(help_labels, "program_parse_selected.help_labels")
	Selected_Command_Invariants(command, "program_parse_selected.command")
	Parsed_Tokens_Invariants(tokens, "program_parse_selected.tokens")
	Workspace_Invariants(workspace, "program_parse_selected.workspace")
	Parse_Failure_Storage_Invariants(failure_storage, "program_parse_selected.failure_storage")
	filled, positionals, slice_named, err := program_assign_named(
		global_flags, help_labels, command.Arguments, command.Flags,
		tokens, workspace, failure_storage,
	)
	if err != nil {
		return nil, err
	}
	err = command_assign_positionals(&Command_Assign_Positionals_Input{
		Arguments: command.Arguments, Positionals: positionals,
		Slice_Named: slice_named, Filled: filled,
		String_Values:   workspace.String_Values,
		Integer_Values:  workspace.Integer_Values,
		Failure_Storage: failure_storage,
	})
	if err != nil {
		return nil, err
	}
	err = command_validate_required(
		command.Arguments, Assigned_Options(filled), failure_storage,
	)
	if err != nil {
		return nil, err
	}
	return Resolved_Deprecation_Warnings(collect_deprecations(
		global_flags, command.Label, command.Deprecated,
		command.Arguments, command.Flags, Assigned_Options(filled),
		workspace.Deprecation_Warnings,
	)), nil
}

// Gathers a warning for each deprecated command or flag the invocation used: the
// resolved command when it is itself deprecated (invoking it is using it), and every
// deprecated command flag or global flag that was set by name (present in filled).
func collect_deprecations(
	global_flags Resolved_Global_Flags, command_label Label,
	command_deprecated Deprecation, arguments Resolved_Arguments, flags Resolved_Flags,
	filled Assigned_Options, storage Deprecation_Warnings,
) (warnings Deprecation_Warnings) {
	defer func() {
		Deprecation_Warnings_Invariants(warnings, "collect_deprecations.warnings")
	}()
	Resolved_Global_Flags_Invariants(global_flags, "collect_deprecations.global_flags")
	Label_Invariants(command_label, "collect_deprecations.command_label")
	Deprecation_Invariants(command_deprecated, "collect_deprecations.command_deprecated")
	Resolved_Arguments_Invariants(arguments, "collect_deprecations.command_arguments")
	Resolved_Flags_Invariants(flags, "collect_deprecations.command_flags")
	Assigned_Options_Invariants(filled, "collect_deprecations.filled")
	Deprecation_Warnings_Invariants(storage, "collect_deprecations.storage")
	warnings = storage[:0]
	if command_deprecated != "" {
		warnings = warning_store(Writable_Deprecation_Warnings(warnings), Warning{
			Kind:     Warning_Kind{Value: WARNING_KIND_COMMAND},
			Name:     Warning_Name{Value: Value_Text(command_label)},
			Guidance: Warning_Guidance{Value: command_deprecated},
		})
	}
	warnings = deprecated_flags_used(
		warnings, Resolved_Options(flags), filled, slices.Position(len(arguments)),
	)
	warnings = deprecated_flags_used(warnings,
		Resolved_Options(global_flags), filled,
		slices.Position(len(arguments)+len(flags)),
	)
	return warnings
}

// Gathers a warning for each deprecated flag that was set by name.
func deprecated_flags_used(
	collected Deprecation_Warnings, flags Resolved_Options,
	filled Assigned_Options, filled_offset slices.Position,
) (warnings Deprecation_Warnings) {
	defer func() {
		Deprecation_Warnings_Invariants(warnings, "deprecated_flags_used.warnings")
	}()
	Resolved_Options_Invariants(flags, "deprecated_flags_used.flags")
	Assigned_Options_Invariants(filled, "deprecated_flags_used.filled")
	slices.Position_Invariants(filled_offset, "deprecated_flags_used.filled_offset")
	Deprecation_Warnings_Invariants(collected, "deprecated_flags_used.storage")
	warnings = collected
	for index := range flags {
		flag := flags[index]
		if flag.State.Deprecated == "" {
			continue
		}
		filled_index := int(filled_offset) + index
		if !filled[filled_index] {
			continue
		}
		warnings = warning_store(
			Writable_Deprecation_Warnings(warnings),
			Warning{
				Kind: Warning_Kind{Value: WARNING_KIND_FLAG},
				Name: Warning_Name{Value: Value_Text(flag.Label)},
				Guidance: Warning_Guidance{
					Value: Deprecation(flag.State.Deprecated),
				},
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
	aver.Always(
		len(warnings) < cap(warnings),
		"Program_Parse Deprecation_Warnings storage has room.",
	)
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
	commands Command_Declarations, mode Program_Mode,
	operating_system_args Help_Arguments,
) (context Command) {
	defer func() { Command_Invariants(context, "program_help_context.context") }()
	Command_Declarations_Invariants(commands, "program_help_context.commands")
	Program_Mode_Invariants(mode, "program_help_context.mode")
	Help_Arguments_Invariants(operating_system_args, "program_help_context.arguments")
	if mode.Value == PROGRAM_MODE_MULTICALL {
		name := Label(path.Base(path.Text(operating_system_args[0])))
		index, _, _, found := program_select_command(Commands(commands), name)
		// A self-invoked multicall binary (run by its own name) names the verb in the
		// first token, so -help there resolves that verb, not the root.
		if !found {
			if len(operating_system_args) > 1 {
				token := Label(operating_system_args[1])
				index, _, _, found = program_select_command(
					Commands(commands), token,
				)
			}
		}
		if !found {
			return Command{}
		}
		return commands[index]
	}
	if mode.Value == PROGRAM_MODE_SINGLE {
		return Command{}
	}
	if len(operating_system_args) > 1 {
		index, _, _, found := program_select_command(
			Commands(commands), Label(operating_system_args[1]),
		)
		if !found {
			return Command{}
		}
		return commands[index]
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
func parse_named_token_result(
	label Option_Label, value Assigned_Value, value_was_set Boolean, err error,
) (_ Option_Label, _ Assigned_Value, _ Boolean, _ error) {
	Option_Label_Invariants(label, "parse_named_token.label")
	Assigned_Value_Invariants(value, "parse_named_token.value")
	Boolean_Invariants(value_was_set, "parse_named_token.value_was_set")
	return label, value, value_was_set, err
}

func parse_named_token(token Named_Token, failure_storage Parse_Failure_Storage) (
	_ Option_Label, _ Assigned_Value, _ Boolean, _ error,
) {
	Named_Token_Invariants(token, "parse_named_token.token")
	Parse_Failure_Storage_Invariants(
		failure_storage, "parse_named_token.failure_storage",
	)
	if strings.Has_Prefix(strings.Text(token), "--") {
		if len(token) > len("--") {
			record := Failure{Storage: Failure_Storage(failure_storage)}
			failure := Failure_Writer{Storage: failure_storage, Size: &record.Size}
			failure_write(failure, Failure_Fragment("use a single dash: -"))
			failure_write(failure, Failure_Fragment(string(token[2:])))
			failure_write(failure, Failure_Fragment(", not --"))
			failure_write(failure, Failure_Fragment(string(token[2:])))
			return parse_named_token_result("", "", false, parse_failure)
		}
	}
	// A lone "-" was already excluded by is_named_token.
	aver.Always(token != "-", "named token is not a lone dash")
	label_text, value_text, found := strings.Cut(strings.Text(token[1:]), "=")
	return parse_named_token_result(
		Option_Label(label_text), Assigned_Value(value_text), Boolean(found), nil,
	)
}

// Applies every -label=value token to its option and returns the bare positionals
// plus, for the slice argument, the values given by name — both tagged with their
// position so the slice preserves command-line order. filled records the scalar
// options set by name: a scalar named twice is an error, and a named scalar argument
// is skipped by the positional pass.
func program_assign_named_result(
	filled Filled_Options, positionals Parsed_Positionals,
	slice_named Parsed_Variadic_Tokens, err error,
) (_ Filled_Options, _ Parsed_Positionals, _ Parsed_Variadic_Tokens, _ error) {
	Filled_Options_Invariants(filled, "program_assign_named.filled")
	Parsed_Positionals_Invariants(positionals, "program_assign_named.positionals")
	Parsed_Variadic_Tokens_Invariants(slice_named, "program_assign_named.slice_named")
	return filled, positionals, slice_named, err
}

func program_assign_named(
	global_flags Resolved_Global_Flags, help_labels Help_Labels,
	arguments Resolved_Arguments, flags Resolved_Flags,
	tokens Parsed_Tokens, workspace Workspace,
	failure_storage Parse_Failure_Storage,
) (_ Filled_Options, _ Parsed_Positionals, _ Parsed_Variadic_Tokens, _ error) {
	Resolved_Global_Flags_Invariants(global_flags, "program_assign_named.global_flags")
	Help_Labels_Invariants(help_labels, "program_assign_named.help_labels")
	Resolved_Arguments_Invariants(arguments, "program_assign_named.command_arguments")
	Resolved_Flags_Invariants(flags, "program_assign_named.command_flags")
	Workspace_Invariants(workspace, "program_assign_named.workspace")
	Parse_Failure_Storage_Invariants(failure_storage, "program_assign_named.failure_storage")
	Parsed_Tokens_Invariants(tokens, "program_assign_named.tokens")
	filled := program_assignment_storage(arguments, flags, global_flags, workspace)
	positionals := Parsed_Positionals(workspace.Positionals[:0])
	slice_named := Parsed_Variadic_Tokens(workspace.Slice_Named[:0])
	for index, token := range tokens {
		if !is_named_token(Value_Text(token)) {
			aver.Always(len(positionals) < cap(workspace.Positionals),
				"Positionals storage has room.")
			positional_count := len(positionals)
			positionals = Parsed_Positionals(workspace.Positionals[:positional_count+1])
			positionals[positional_count] = Indexed_Token{
				Index: Token_Index(index), Value: Value_Text(token),
			}
			continue
		}
		label, value, value_was_set, format_err := parse_named_token(
			Named_Token(token), failure_storage)
		if format_err != nil {
			return program_assign_named_result(
				filled, positionals, slice_named, format_err)
		}
		location, option_index, filled_index, is_slice_argument, find_err :=
			program_find_option(
				global_flags, help_labels, arguments, flags, label,
				failure_storage,
			)
		if find_err != nil {
			return program_assign_named_result(
				filled, positionals, slice_named, find_err)
		}
		if is_slice_argument {
			aver.Always(len(slice_named) < cap(workspace.Slice_Named),
				"Named variadic storage has room.")
			named_count := len(slice_named)
			slice_named = Parsed_Variadic_Tokens(workspace.Slice_Named[:named_count+1])
			slice_named[named_count] = Indexed_Token{
				Index: Token_Index(index), Value: Value_Text(value),
			}
			continue
		}
		if filled[filled_index] {
			record := Failure{Storage: Failure_Storage(failure_storage)}
			failure := Failure_Writer{Storage: failure_storage, Size: &record.Size}
			failure_write(failure, Failure_Fragment("-"))
			failure_write(failure, Failure_Fragment(string(label)))
			failure_write(failure, Failure_Fragment(" may only be given once"))
			return program_assign_named_result(
				filled, positionals, slice_named, parse_failure)
		}
		set_err := program_assign_named_option(
			global_flags, arguments, flags, Resolved_Option_Location(location),
			Option_Index(option_index), value,
			Equals_Present(value_was_set), failure_storage,
		)
		if set_err != nil {
			return program_assign_named_result(
				filled, positionals, slice_named, set_err)
		}
		filled[filled_index] = true
	}
	return program_assign_named_result(filled, positionals, slice_named, nil)
}

// Assignment storage resets only the cells used by the active command.
func program_assignment_storage(
	arguments Resolved_Arguments, flags Resolved_Flags,
	global_flags Resolved_Global_Flags, workspace Workspace,
) (filled Filled_Options) {
	defer func() {
		Filled_Options_Invariants(filled, "program_assignment_storage.filled")
	}()
	Resolved_Arguments_Invariants(arguments, "program_assignment_storage.arguments")
	Resolved_Flags_Invariants(flags, "program_assignment_storage.flags")
	Resolved_Global_Flags_Invariants(
		global_flags, "program_assignment_storage.global_flags",
	)
	Workspace_Invariants(workspace, "program_assignment_storage.workspace")
	filled_count := len(arguments) + len(flags) + len(global_flags)
	aver.Always(
		len(workspace.Filled) >= filled_count,
		"Program_Parse Filled storage covers active options.",
	)
	filled = Filled_Options(workspace.Filled[:filled_count])
	for index := range filled {
		filled[index] = false
	}
	return filled
}

// Collection selection stays after successful lookup, so an absent location never indexes.
func program_assign_named_option(
	global_flags Resolved_Global_Flags,
	arguments Resolved_Arguments, flags Resolved_Flags,
	location Resolved_Option_Location,
	option_index Option_Index, value Assigned_Value,
	value_was_set Equals_Present, failure_storage Parse_Failure_Storage,
) (err error) {
	Resolved_Global_Flags_Invariants(
		global_flags, "program_assign_named_option.global_flags",
	)
	Resolved_Arguments_Invariants(
		arguments, "program_assign_named_option.command_arguments",
	)
	Resolved_Flags_Invariants(flags, "program_assign_named_option.command_flags")
	Resolved_Option_Location_Invariants(location, "program_assign_named_option.location")
	Option_Index_Invariants(option_index, "program_assign_named_option.option_index")
	Assigned_Value_Invariants(value, "program_assign_named_option.value")
	Equals_Present_Invariants(value_was_set, "program_assign_named_option.value_was_set")
	Parse_Failure_Storage_Invariants(
		failure_storage, "program_assign_named_option.failure_storage",
	)
	var options Resolved_Options
	resolved := true
	switch location {
	case Resolved_Option_Location(OPTION_LOCATION_ARGUMENT):
		options = Resolved_Options(arguments)
	case Resolved_Option_Location(OPTION_LOCATION_COMMAND_FLAG):
		options = Resolved_Options(flags)
	case Resolved_Option_Location(OPTION_LOCATION_GLOBAL_FLAG):
		options = Resolved_Options(global_flags)
	default:
		resolved = false
	}
	aver.Always(resolved, "Successful option search resolves one collection.")
	return option_set_value(Option_Set_Value_Input{
		Options: Found_Resolved_Options(options), Index: option_index, Value: value,
		Value_Was_Set: value_was_set, Failure_Storage: failure_storage,
	})
}

// Finds the option named by a -label token across the command's arguments, then its
// flags, then the program's global flags, returning a pointer into the parse-time copy
// so assignment lands in the right slot. is_slice_argument is true when the match is a
// slice-valued argument, which appends rather than sets. Errors on an unknown label.
func program_find_option_result(
	location Option_Location, option_index slices.Found_Index,
	filled_index slices.Found_Index, is_slice_argument Boolean, err error,
) (_ Option_Location, _ slices.Found_Index, _ slices.Found_Index, _ Boolean, _ error) {
	Option_Location_Invariants(location, "program_find_option.location")
	slices.Found_Index_Invariants(option_index, "program_find_option.option_index")
	slices.Found_Index_Invariants(filled_index, "program_find_option.filled_index")
	Boolean_Invariants(is_slice_argument, "program_find_option.is_slice_argument")
	return location, option_index, filled_index, is_slice_argument, err
}

func program_find_option(
	global_flags Resolved_Global_Flags, help_labels Help_Labels,
	arguments Resolved_Arguments, flags Resolved_Flags, label Option_Label,
	failure_storage Parse_Failure_Storage,
) (
	_ Option_Location, _ slices.Found_Index,
	_ slices.Found_Index, _ Boolean, _ error,
) {
	Resolved_Global_Flags_Invariants(global_flags, "program_find_option.global_flags")
	Help_Labels_Invariants(help_labels, "program_find_option.help_labels")
	Resolved_Arguments_Invariants(arguments, "program_find_option.command_arguments")
	Resolved_Flags_Invariants(flags, "program_find_option.command_flags")
	Option_Label_Invariants(label, "program_find_option.label")
	Parse_Failure_Storage_Invariants(
		failure_storage, "program_find_option.failure_storage",
	)
	for index := range arguments {
		argument := &arguments[index]
		if argument.Label == label {
			return program_find_option_result(
				OPTION_LOCATION_ARGUMENT, slices.Found_Index(index),
				slices.Found_Index(index),
				option_is_slice(Option_Type(argument.Type.Value)), nil,
			)
		}
	}
	for index := range flags {
		if flags[index].Label == label {
			return program_find_option_result(
				OPTION_LOCATION_COMMAND_FLAG, slices.Found_Index(index),
				slices.Found_Index(len(arguments)+index), false, nil,
			)
		}
	}
	for index := range global_flags {
		if global_flags[index].Label == label {
			global_filled_index := len(arguments) + len(flags) + index
			return program_find_option_result(
				OPTION_LOCATION_GLOBAL_FLAG, slices.Found_Index(index),
				slices.Found_Index(global_filled_index), false, nil,
			)
		}
	}
	suggestion_match, suggestion_found := closest_option_label(
		global_flags, help_labels, arguments, flags, label,
	)
	record := Failure{Storage: Failure_Storage(failure_storage)}
	failure := Failure_Writer{Storage: failure_storage, Size: &record.Size}
	if suggestion_found {
		failure_write(failure, Failure_Fragment("unknown option -"))
		failure_write(failure, Failure_Fragment(string(label)))
		failure_write(failure, Failure_Fragment(", did you mean -"))
		failure_write(failure, Failure_Fragment(string(suggestion_match)))
		failure_write(failure, Failure_Fragment("?"))
		return program_find_option_result(
			OPTION_LOCATION_ABSENT, -1, -1, false, parse_failure)
	}
	failure_write(failure, Failure_Fragment("unknown option -"))
	failure_write(failure, Failure_Fragment(string(label)))
	return program_find_option_result(
		OPTION_LOCATION_ABSENT, -1, -1, false, parse_failure)
}

// Reports whether an option appears in help and completion. A hidden or deprecated
// flag still parses; it is only kept out of what the tool advertises.
func option_shown(hidden Hidden, deprecated Deprecation) (shown Boolean) {
	defer func() { Boolean_Invariants(shown, "option_shown.shown") }()
	Hidden_Invariants(hidden, "option_shown.hidden")
	Deprecation_Invariants(deprecated, "option_shown.deprecated")
	if hidden {
		return false
	}
	return deprecated == ""
}

// Reports whether a command appears in help and completion. A hidden or deprecated
// command still resolves; it is only kept out of what the tool advertises.
func command_shown(hidden Hidden, deprecated Deprecation) (shown Boolean) {
	defer func() { Boolean_Invariants(shown, "command_shown.shown") }()
	Hidden_Invariants(hidden, "command_shown.hidden")
	Deprecation_Invariants(deprecated, "command_shown.deprecated")
	if hidden {
		return false
	}
	return deprecated == ""
}

// SUGGESTION_STATE_COUNT follows one nearest-label accumulator.
const SUGGESTION_STATE_COUNT = len("state") / len("state")

// Suggestion_State keeps optional match and distance in one union cell.
type Suggestion_State struct {
	// Match is nearest admitted label.
	Match Label
	// Found distinguishes absent match from empty label.
	Found Boolean
	// Best is nearest admitted distance.
	Best levenshtein.Distance_Value
}

// Suggestion_State_Invariants bounds union storage without impossible branch coverage.
func Suggestion_State_Invariants(value Suggestion_State, namespace aver.Namespace) {
	Label_Invariants(value.Match, namespace)
	Boolean_Invariants(value.Found, namespace)
	levenshtein.Distance_Value_Invariants(value.Best, namespace)
}

// Suggested_Label is empty on a miss or long enough for a nonzero edit
// threshold after exact matches have already returned.
type Suggested_Label string

// Suggested_Label_Invariants excludes identities that only an exact match can
// select before suggestion search.
func Suggested_Label_Invariants(value Suggested_Label, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int(
			len(value), strings.TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM,
			1, 2, 2, 2,
		).
		Ensure()
}

// Levenshtein_Workspace_Handle is one live nearest-label workspace.
type Levenshtein_Workspace_Handle *levenshtein.Workspace

// Levenshtein_Workspace_Handle_Invariants composes present distance storage.
func Levenshtein_Workspace_Handle_Invariants(
	value Levenshtein_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	levenshtein.Workspace_Invariants(*value, namespace)
}

// Suggestion_State_Handle keeps nearest-match mutation in caller storage.
type Suggestion_State_Handle *Suggestion_State

// Suggestion_State_Handle_Invariants composes present nearest-match storage.
func Suggestion_State_Handle_Invariants(
	value Suggestion_State_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Suggestion_State_Invariants(*value, namespace)
}

// Updates one closest-label search without assembling a temporary candidate slice.
func closest_label(
	workspace Levenshtein_Workspace_Handle, target Label, candidate Label,
	state Suggestion_State_Handle,
) {
	Levenshtein_Workspace_Handle_Invariants(workspace, "closest_label.workspace")
	Label_Invariants(target, "closest_label.target")
	Label_Invariants(candidate, "closest_label.candidate")
	Suggestion_State_Handle_Invariants(state, "closest_label.state")
	distance, status := levenshtein.Distance(levenshtein.Distance_Input{
		Workspace: workspace,
		From:      levenshtein.From_Text_Unvalidated(target),
		To:        levenshtein.To_Text_Unvalidated(candidate),
	})
	if status != levenshtein.STATUS_OK {
		return
	}
	threshold := levenshtein.Distance_Value(max(len(target), len(candidate)) / 3)
	if distance > threshold {
		return
	}
	if state.Found {
		if distance >= state.Best {
			return
		}
	}
	state.Match = candidate
	state.Found = true
	state.Best = distance
}

// Searches visible option labels directly so suggestion work stays caller-owned.
func closest_option_label_result(
	match Option_Label, found Boolean,
) (_ Option_Label, _ Boolean) {
	Option_Label_Invariants(match, "closest_option_label.match")
	Boolean_Invariants(found, "closest_option_label.found")
	return match, found
}

func closest_option_label(
	global_flags Resolved_Global_Flags, help_labels Help_Labels,
	arguments Resolved_Arguments, flags Resolved_Flags, target Option_Label,
) (_ Option_Label, _ Boolean) {
	Resolved_Global_Flags_Invariants(global_flags, "closest_option_label.global_flags")
	Help_Labels_Invariants(help_labels, "closest_option_label.help_labels")
	Resolved_Arguments_Invariants(arguments, "closest_option_label.arguments")
	Resolved_Flags_Invariants(flags, "closest_option_label.flags")
	Option_Label_Invariants(target, "closest_option_label.target")
	var workspace levenshtein.Workspace
	state := Suggestion_State{}
	for index := range arguments {
		option := arguments[index]
		closest_label(&workspace, Label(target), Label(option.Label), &state)
	}
	for index := range flags {
		option := flags[index]
		if option_shown(option.State.Hidden, option.State.Deprecated) {
			closest_label(&workspace, Label(target), Label(option.Label), &state)
		}
	}
	if help_labels.Hidden == HELP_HIDDEN_VISIBLE {
		closest_label(&workspace, Label(target), Label(help_labels.Short), &state)
		closest_label(&workspace, Label(target), Label(help_labels.Long), &state)
	}
	for index := range global_flags {
		option := global_flags[index]
		if option_shown(option.State.Hidden, option.State.Deprecated) {
			closest_label(&workspace, Label(target), Label(option.Label), &state)
		}
	}
	return closest_option_label_result(Option_Label(state.Match), state.Found)
}

// Caller-owned option storage keeps parsing from mutating Program declarations.
func program_resolve_command_result(
	active_command Selected_Command, command_selected Boolean, err error,
) (_ Selected_Command, _ Boolean, _ error) {
	Selected_Command_Invariants(active_command, "program_resolve_command.active_command")
	Boolean_Invariants(command_selected, "program_resolve_command.command_selected")
	return active_command, command_selected, err
}

func program_resolve_command(
	commands Command_Declarations, single Single_Commands, mode Program_Mode,
	operating_system_args Parse_Arguments,
	workspace Workspace, failure_storage Parse_Failure_Storage,
) (_ Selected_Command, _ Boolean, _ error) {
	Command_Declarations_Invariants(commands, "program_resolve_command.commands")
	Single_Commands_Invariants(single, "program_resolve_command.single")
	Program_Mode_Invariants(mode, "program_resolve_command.mode")
	Workspace_Invariants(workspace, "program_resolve_command.workspace")
	Parse_Failure_Storage_Invariants(failure_storage, "program_resolve_command.failure_storage")
	Parse_Arguments_Invariants(operating_system_args, "program_resolve_command.arguments")
	active_command := Selected_Command{}
	command_selected := Boolean(false)
	var err error
	command_index, command_name := Option_Index(0), Label("")
	suggestion_match := Suggested_Label("")
	suggestion_found, found := Boolean(false), Boolean(true)
	if mode.Value == PROGRAM_MODE_MULTICALL {
		command_name = Label(path.Base(path.Text(operating_system_args[0])))
		command_index, suggestion_match, suggestion_found, found = program_select_command(
			Commands(commands), command_name,
		)
		// Self-invocation takes its verb from slot 1, so errors name that chosen token.
		if !found {
			if len(operating_system_args) > 1 {
				command_selected = true
				command_name = Label(operating_system_args[1])
				command_index, suggestion_match, suggestion_found, found =
					program_select_command(Commands(commands), command_name)
			}
		}
	} else if mode.Value != PROGRAM_MODE_SINGLE {
		if len(operating_system_args) > 1 {
			command_selected = true
			command_name = Label(operating_system_args[1])
			command_index, suggestion_match, suggestion_found, found =
				program_select_command(Commands(commands), command_name)
		}
	}
	if !found {
		err = failure_unknown_command(
			failure_storage, command_name, suggestion_match, suggestion_found)
		if mode.Value == PROGRAM_MODE_SINGLE {
			source := single.Command
			return program_resolve_command_result(Selected_Command{
				Label: source.Label, Description: source.Description,
				Hidden: source.Hidden, Deprecated: source.Deprecated,
			}, command_selected, err)
		}
		source := commands[0]
		return program_resolve_command_result(Selected_Command{
			Label: source.Label, Description: source.Description,
			Hidden: source.Hidden, Deprecated: source.Deprecated,
		}, command_selected, err)
	}
	var source Command
	if mode.Value == PROGRAM_MODE_SINGLE {
		source = single.Command
	} else {
		source = commands[command_index]
	}
	active_command = copy_command(source, workspace)
	return program_resolve_command_result(active_command, command_selected, nil)
}

// Copying binds mutable option values to caller storage before token assignment.
func copy_command(source Command, workspace Workspace) (output Selected_Command) {
	defer func() { Selected_Command_Invariants(output, "copy_command.output") }()
	Command_Invariants(source, "copy_command.source")
	Workspace_Invariants(workspace, "copy_command.workspace")
	output = Selected_Command{
		Label: source.Label, Description: source.Description,
		Hidden: source.Hidden, Deprecated: source.Deprecated,
	}
	aver.Always(len(workspace.Command_Arguments) >= len(source.Arguments),
		"Command argument storage fits active arguments.")
	output.Arguments = Resolved_Arguments(copy_options(
		Options(source.Arguments), Resolved_Options(workspace.Command_Arguments),
	))
	aver.Always(
		len(workspace.Command_Flags) >= len(source.Flags),
		"Program_Parse Command_Flags storage fits active flags.",
	)
	output.Flags = Resolved_Flags(copy_options(
		Options(source.Flags), Resolved_Options(workspace.Command_Flags),
	))
	return output
}

// Resolves a command name to its index, suggesting the closest command when the name
// is an unrecognized near-miss. Shared by multicall selection (the argv[0] basename)
// and multi-command selection (the slot-1 token).
func program_select_command_result(
	index Option_Index, match Suggested_Label,
	suggestion_found Boolean, found Boolean,
) (_ Option_Index, _ Suggested_Label, _ Boolean, _ Boolean) {
	Option_Index_Invariants(index, "program_select_command.index")
	Suggested_Label_Invariants(match, "program_select_command.match")
	Boolean_Invariants(suggestion_found, "program_select_command.suggestion_found")
	Boolean_Invariants(found, "program_select_command.found")
	return index, match, suggestion_found, found
}

func program_select_command(
	commands Commands, name Label,
) (
	_ Option_Index, _ Suggested_Label,
	_ Boolean, _ Boolean,
) {
	Commands_Invariants(commands, "program_select_command.commands")
	Label_Invariants(name, "program_select_command.name")
	for candidate_index, command := range commands {
		if command.Label == name {
			return program_select_command_result(
				Option_Index(candidate_index), "", false, true)
		}
	}
	var workspace levenshtein.Workspace
	suggestion := Suggestion_State{}
	for command_index := range commands {
		if !command_shown(
			commands[command_index].Hidden, commands[command_index].Deprecated,
		) {
			continue
		}
		closest_label(&workspace, name, commands[command_index].Label, &suggestion)
	}
	return program_select_command_result(
		0, Suggested_Label(suggestion.Match), suggestion.Found, false)
}

func failure_unknown_command(
	storage Parse_Failure_Storage, name Label,
	match Suggested_Label, found Boolean,
) (err error) {
	Parse_Failure_Storage_Invariants(storage, "failure_unknown_command.storage")
	Label_Invariants(name, "failure_unknown_command.name")
	Suggested_Label_Invariants(match, "failure_unknown_command.match")
	Boolean_Invariants(found, "failure_unknown_command.found")
	record := Failure{Storage: Failure_Storage(storage)}
	failure := Failure_Writer{Storage: storage, Size: &record.Size}
	failure_write(failure, Failure_Fragment("unknown command "))
	failure_write_quoted(failure, Failure_Fragment(string(name)))
	if found {
		failure_write(failure, Failure_Fragment(", did you mean "))
		failure_write_quoted(
			failure, Failure_Fragment(string(match)),
		)
		failure_write(failure, Failure_Fragment("?"))
	}
	return parse_failure
}

// Input for command_assign_positionals.
type Command_Assign_Positionals_Input struct {
	// Arguments retain caller-owned storage while parsed values change.
	Arguments Resolved_Arguments
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
	// Failure_Storage owns rejected positional diagnostics.
	Failure_Storage Parse_Failure_Storage
}

// Command_Assign_Positionals_Input_Invariants composes active positional state.
func Command_Assign_Positionals_Input_Invariants(
	value Command_Assign_Positionals_Input, namespace aver.Namespace,
) {
	Resolved_Arguments_Invariants(value.Arguments, namespace)
	Parsed_Positionals_Invariants(value.Positionals, namespace)
	Parsed_Variadic_Tokens_Invariants(value.Slice_Named, namespace)
	Filled_Options_Invariants(value.Filled, namespace)
	String_Values_Invariants(value.String_Values, namespace)
	Integer_Values_Invariants(value.Integer_Values, namespace)
	Parse_Failure_Storage_Invariants(value.Failure_Storage, namespace)
}

// Command_Assign_Positionals_Input_Handle is one live positional assignment record.
type Command_Assign_Positionals_Input_Handle *Command_Assign_Positionals_Input

// Command_Assign_Positionals_Input_Handle_Invariants composes present assignment storage.
func Command_Assign_Positionals_Input_Handle_Invariants(
	value Command_Assign_Positionals_Input_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Command_Assign_Positionals_Input_Invariants(*value, namespace)
}

// Fills the command's scalar arguments from the positional tokens in declaration
// order, skipping any already set by name, then routes the rest into the trailing
// slice argument together with its named contributions, in command-line order. Errors
// when a positional has no argument to fill. Filled gains every scalar argument set
// here so the required check can see it.
func command_assign_positionals(
	input Command_Assign_Positionals_Input_Handle,
) (err error) {
	Command_Assign_Positionals_Input_Handle_Invariants(
		input, "command_assign_positionals.input",
	)
	arguments := input.Arguments
	slice_index := command_slice_argument_index(arguments)
	slice_contributions := input.Slice_Named
	argument_cursor := 0
	for _, positional := range input.Positionals {
		for argument_cursor < len(arguments) {
			argument := arguments[argument_cursor]
			if option_is_slice(Option_Type(argument.Type.Value)) {
				break
			}
			if !input.Filled[argument_cursor] {
				break
			}
			argument_cursor++
		}
		if argument_cursor < len(arguments) {
			if argument_cursor != int(slice_index) {
				set_err := option_set_positional(
					arguments, Option_Index(argument_cursor),
					positional.Value, input.Failure_Storage,
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
			record := Failure{Storage: Failure_Storage(input.Failure_Storage)}
			failure := Failure_Writer{
				Storage: input.Failure_Storage, Size: &record.Size,
			}
			failure_write(failure, Failure_Fragment("too many arguments: "))
			failure_write_quoted(
				failure, Failure_Fragment(string(positional.Value)),
			)
			failure_write(failure, Failure_Fragment(" was not expected"))
			return parse_failure
		}
		aver.Always(
			len(slice_contributions) < cap(slice_contributions),
			"Positional variadic contribution has storage.",
		)
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
		Resolved_Variadic_Arguments(arguments), Option_Index(slice_index),
		slice_contributions,
		input.String_Values, input.Integer_Values, input.Failure_Storage,
	)
}

// Returns the index of the command's trailing slice argument, or -1 when the last
// argument is not a slice.
func command_slice_argument_index(
	arguments Resolved_Arguments,
) (index slices.Found_Index) {
	defer func() {
		slices.Found_Index_Invariants(index, "command_slice_argument_index.index")
	}()
	Resolved_Arguments_Invariants(arguments, "command_slice_argument_index.arguments")
	count := len(arguments)
	if count == 0 {
		return -1
	}
	if option_is_slice(Option_Type(arguments[count-1].Type.Value)) {
		return slices.Found_Index(count - 1)
	}
	return -1
}

// Returns an error naming the first scalar argument left unset by both name and
// position. The slice argument is optional, so it is never required.
func command_validate_required(
	arguments Resolved_Arguments, filled Assigned_Options,
	failure_storage Parse_Failure_Storage,
) (err error) {
	Resolved_Arguments_Invariants(arguments, "command_validate_required.arguments")
	Assigned_Options_Invariants(filled, "command_validate_required.filled")
	Parse_Failure_Storage_Invariants(
		failure_storage, "command_validate_required.failure_storage",
	)
	for index := range arguments {
		argument := arguments[index]
		if option_is_slice(Option_Type(argument.Type.Value)) {
			continue
		}
		if filled[index] {
			continue
		}
		record := Failure{Storage: Failure_Storage(failure_storage)}
		failure := Failure_Writer{Storage: failure_storage, Size: &record.Size}
		failure_write(failure, Failure_Fragment("missing required argument "))
		failure_write_quoted(
			failure, Failure_Fragment(string(argument.Label)),
		)
		failure_write(
			failure, Failure_Fragment("; pass it by position or as -"),
		)
		failure_write(failure, Failure_Fragment(string(argument.Label)))
		failure_write(failure, Failure_Fragment("=value"))
		return parse_failure
	}
	return nil
}

// Converts a positional token to the scalar argument's type and assigns it. Unlike a
// named value, a positional is taken verbatim: no quote trimming, and empty is allowed.
func option_set_positional(
	arguments Resolved_Arguments, argument_index Option_Index,
	value Value_Text, failure_storage Parse_Failure_Storage,
) (err error) {
	Resolved_Arguments_Invariants(arguments, "option_set_positional.arguments")
	Option_Index_Invariants(argument_index, "option_set_positional.argument_index")
	Value_Text_Invariants(value, "option_set_positional.value")
	Parse_Failure_Storage_Invariants(
		failure_storage, "option_set_positional.failure_storage",
	)
	aver.Always(
		int(argument_index) < len(arguments),
		"Positional argument index belongs to active arguments.",
	)
	argument := &arguments[argument_index]
	aver.Enum(
		argument.Type.Value, OPTION_TYPE_STRING, OPTION_TYPE_INTEGER,
		"Positional argument has scalar storage.",
	)
	switch argument.Type.Value {
	case OPTION_TYPE_STRING:
		argument.State.String = Value_Text(value)
		argument.State.Parsed = true
	case OPTION_TYPE_INTEGER:
		number, parse_err := strconv.Parse_Decimal(strconv.Text(value))
		if parse_err != nil {
			record := Failure{Storage: Failure_Storage(failure_storage)}
			failure := Failure_Writer{Storage: failure_storage, Size: &record.Size}
			failure_write(failure, Failure_Fragment(string(argument.Label)))
			failure_write(
				failure, Failure_Fragment(" expects a whole number, but got "))
			failure_write_quoted(failure, Failure_Fragment(string(value)))
			return parse_failure
		}
		argument.State.Integer = Integer(number)
		argument.State.Parsed = true
	}
	return option_check_enum(
		argument.Enumeration, argument.State.String, argument.State.Integer,
		Display_Name{
			Name: Display_Option_Name{Value: Option_Name(argument.Label)},
		}, failure_storage,
	)
}

// Builds the slice option's value from its contributions, already sorted into
// command-line order, converting each element to the slice's element type.
func option_set_slice(
	arguments Resolved_Variadic_Arguments, argument_index Option_Index,
	contributions Parsed_Variadic_Tokens,
	string_values String_Values, integer_values Integer_Values,
	failure_storage Parse_Failure_Storage,
) (err error) {
	Resolved_Variadic_Arguments_Invariants(arguments, "option_set_slice.arguments")
	Option_Index_Invariants(argument_index, "option_set_slice.argument_index")
	Parsed_Variadic_Tokens_Invariants(contributions, "option_set_slice.contributions")
	String_Values_Invariants(string_values, "option_set_slice.string_values")
	Integer_Values_Invariants(integer_values, "option_set_slice.integer_values")
	Parse_Failure_Storage_Invariants(
		failure_storage, "option_set_slice.failure_storage",
	)
	aver.Always(
		int(argument_index) < len(arguments),
		"Variadic argument index belongs to active arguments.",
	)
	option := &arguments[argument_index]
	aver.Enum(
		option.Type.Value, OPTION_TYPE_STRINGS, OPTION_TYPE_INTEGERS,
		"Variadic argument has collection storage.",
	)
	switch option.Type.Value {
	case OPTION_TYPE_STRINGS:
		aver.Always(
			len(string_values) >= len(contributions),
			"Program_Parse String_Values storage fits contributions.",
		)
		values := string_values[:len(contributions)]
		for index, contribution := range contributions {
			values[index] = string(contribution.Value)
		}
		option.State.Strings = String_Values(values)
		option.State.Parsed = true
	case OPTION_TYPE_INTEGERS:
		aver.Always(
			len(integer_values) >= len(contributions),
			"Program_Parse Integer_Values storage fits contributions.",
		)
		values := integer_values[:len(contributions)]
		for index, contribution := range contributions {
			number, parse_err := strconv.Parse_Decimal(
				strconv.Text(contribution.Value),
			)
			if parse_err != nil {
				record := Failure{Storage: Failure_Storage(failure_storage)}
				failure := Failure_Writer{
					Storage: failure_storage, Size: &record.Size,
				}
				failure_write(failure, Failure_Fragment(string(option.Label)))
				failure_write(
					failure,
					Failure_Fragment(" expects a whole number, but got "))
				failure_write_quoted(
					failure, Failure_Fragment(string(contribution.Value)))
				return parse_failure
			}
			values[index] = int(number)
		}
		option.State.Integers = Integer_Values(values)
		option.State.Parsed = true
	}
	return nil
}

// Input for option_set_value.
type Option_Set_Value_Input struct {
	// Options excludes empty storage after lookup selected one index.
	Options Found_Resolved_Options
	// Index avoids an unbounded pointer into caller storage.
	Index Option_Index
	// Value is the raw value text following '='.
	Value Assigned_Value
	// Value_Was_Set reports whether '=' appeared in the flag.
	Value_Was_Set Equals_Present
	// Failure_Storage owns rejected assignment diagnostics.
	Failure_Storage Parse_Failure_Storage
}

// Option_Set_Value_Input_Invariants composes one named assignment.
func Option_Set_Value_Input_Invariants(
	value Option_Set_Value_Input, namespace aver.Namespace,
) {
	Found_Resolved_Options_Invariants(value.Options, namespace)
	Option_Index_Invariants(value.Index, namespace)
	Assigned_Value_Invariants(value.Value, namespace)
	Equals_Present_Invariants(value.Value_Was_Set, namespace)
	Parse_Failure_Storage_Invariants(value.Failure_Storage, namespace)
}

// Validates and assigns a parsed flag value by the option's declared type.
// Non-boolean flags require a non-empty value.
func option_set_value(input Option_Set_Value_Input) (err error) {
	Option_Set_Value_Input_Invariants(input, "option_set_value.input")
	aver.Always(
		int(input.Index) < len(input.Options),
		"Named option index belongs to active options.",
	)
	option := &input.Options[input.Index]
	name := Option_Name(option.Label)
	is_boolean := option.Type.Value == OPTION_TYPE_BOOLEAN
	if !is_boolean {
		absent := !input.Value_Was_Set
		if input.Value == "" {
			absent = true
		}
		if absent {
			record := Failure{Storage: Failure_Storage(input.Failure_Storage)}
			failure := Failure_Writer{
				Storage: input.Failure_Storage, Size: &record.Size,
			}
			failure_write(failure, Failure_Fragment("-"))
			failure_write(failure, Failure_Fragment(string(name)))
			failure_write(
				failure, Failure_Fragment(" needs a value, e.g. -"),
			)
			failure_write(failure, Failure_Fragment(string(name)))
			failure_write(failure, Failure_Fragment("=value"))
			return parse_failure
		}
	}
	switch option.Type.Value {
	case OPTION_TYPE_BOOLEAN:
		option.State.Boolean = true
		option.State.Parsed = true
	case OPTION_TYPE_STRING:
		option.State.String = Value_Text(
			trim_quotes(Named_String_Value(input.Value)),
		)
		option.State.Parsed = true
	case OPTION_TYPE_INTEGER:
		number, parse_err := strconv.Parse_Decimal(strconv.Text(input.Value))
		if parse_err != nil {
			record := Failure{Storage: Failure_Storage(input.Failure_Storage)}
			failure := Failure_Writer{
				Storage: input.Failure_Storage, Size: &record.Size,
			}
			failure_write(failure, Failure_Fragment(string(name)))
			failure_write(
				failure, Failure_Fragment(" expects a whole number, but got "))
			failure_write_quoted(failure, Failure_Fragment(string(input.Value)))
			return parse_failure
		}
		option.State.Integer = Integer(number)
		option.State.Parsed = true
	}
	if option.Enumeration.String == nil {
		if option.Enumeration.Integers == nil {
			return nil
		}
	}
	return option_check_enum(
		option.Enumeration, option.State.String, option.State.Integer,
		Display_Name{Name: Display_Option_Name{Value: name}, Dashed: true},
		input.Failure_Storage,
	)
}

// Returns an error when a parsed value falls outside an enum option's permitted set, and
// nil when the option is not an enum or the value is a member. For a string enum it
// suggests the closest member on a near miss, mirroring the did-you-mean the parser
// already gives for unknown options; otherwise, and for an int enum, it lists the whole
// set. display_name is the option as the user wrote it — a -flag or a bare argument
// label — for the message.
func option_check_enum(
	enumeration Option_Enumeration, string_value Value_Text, integer Integer,
	display_name Display_Name,
	failure_storage Parse_Failure_Storage,
) (err error) {
	Option_Enumeration_Invariants(enumeration, "option_check_enum.enumeration")
	Value_Text_Invariants(string_value, "option_check_enum.string_value")
	Integer_Invariants(integer, "option_check_enum.integer")
	Display_Name_Invariants(display_name, "option_check_enum.display_name")
	Parse_Failure_Storage_Invariants(
		failure_storage, "option_check_enum.failure_storage",
	)
	if enumeration.String == nil {
		if enumeration.Integers == nil {
			return nil
		}
	}
	if enumeration.String != nil {
		enum := enumeration.String
		value := string_value
		if slices.Contains(enum, string(value)) {
			return nil
		}
		var workspace levenshtein.Workspace
		match, ok, _ := levenshtein.Closest(levenshtein.Closest_Input{
			Workspace:  &workspace,
			Target:     levenshtein.Target_Text_Unvalidated(value),
			Candidates: levenshtein.Candidates_Unvalidated(enum),
		})
		record := Failure{Storage: Failure_Storage(failure_storage)}
		failure := Failure_Writer{Storage: failure_storage, Size: &record.Size}
		failure_write(failure, Failure_Fragment("invalid value "))
		failure_write_quoted(failure, Failure_Fragment(string(value)))
		failure_write(failure, Failure_Fragment(" for "))
		if display_name.Dashed {
			failure_write(failure, Failure_Fragment("-"))
		}
		failure_write(failure, Failure_Fragment(string(display_name.Name.Value)))
		if ok {
			failure_write(failure, Failure_Fragment(", did you mean "))
			failure_write_quoted(failure, Failure_Fragment(string(match)))
			failure_write(failure, Failure_Fragment("?"))
			return parse_failure
		}
		failure_write(failure, Failure_Fragment("; allowed: "))
		failure_write_string_set(failure, Failure_String_Set(enum))
		return parse_failure
	}
	value := integer
	if slices.Contains(enumeration.Integers, int(value)) {
		return nil
	}
	record := Failure{Storage: Failure_Storage(failure_storage)}
	failure := Failure_Writer{Storage: failure_storage, Size: &record.Size}
	failure_write(failure, Failure_Fragment("invalid value "))
	failure_write_decimal(failure, Failure_Integer{Value: Integer(value)})
	failure_write(failure, Failure_Fragment(" for "))
	if display_name.Dashed {
		failure_write(failure, Failure_Fragment("-"))
	}
	failure_write(failure, Failure_Fragment(string(display_name.Name.Value)))
	failure_write(failure, Failure_Fragment("; allowed: "))
	failure_write_integer_set(failure, Failure_Integer_Set(enumeration.Integers))
	return parse_failure
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

// Rendered_Size_Value is checked caller-output arithmetic.
type Rendered_Size_Value int

// Rendered_Size_Value_Invariants protects output bounds.
func Rendered_Size_Value_Invariants(value Rendered_Size_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), RENDERED_SIZE_MINIMUM, RENDERED_SIZE_MAXIMUM).
		Ensure()
}

// Rendered_Size counts bytes already proven to fit one Output.
type Rendered_Size struct {
	// Value is rendered byte count.
	Value Rendered_Size_Value
}

// Rendered_Size_Invariants protects all help-size arithmetic before output writes.
func Rendered_Size_Invariants(value Rendered_Size, namespace aver.Namespace) {
	Rendered_Size_Value_Invariants(value.Value, namespace)
}

// FLAG_CELL_SIZE_MINIMUM follows indentation, dash, and an omitted label.
const FLAG_CELL_SIZE_MINIMUM = len("    -")

// Flag_Cell_Size_Value excludes width state before one visible flag is measured.
type Flag_Cell_Size_Value int

// Flag_Cell_Size_Value_Invariants bounds one visible flag signature width.
func Flag_Cell_Size_Value_Invariants(
	value Flag_Cell_Size_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), FLAG_CELL_SIZE_MINIMUM, RENDERED_SIZE_MAXIMUM).
		Ensure()
}

// Flag_Cell_Size is one visible flag signature width.
type Flag_Cell_Size struct {
	// Value excludes an empty visible flag cell.
	Value Flag_Cell_Size_Value
}

// Flag_Cell_Size_Invariants bounds one visible flag signature width.
func Flag_Cell_Size_Invariants(value Flag_Cell_Size, namespace aver.Namespace) {
	Flag_Cell_Size_Value_Invariants(value.Value, namespace)
}

// External_Type_Size is the rendered width of one scalar or enumeration type.
type External_Type_Size int

// External_Type_Size_Invariants excludes widths shorter than any scalar spelling.
func External_Type_Size_Invariants(value External_Type_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), EXTERNAL_TYPE_TEXT_SIZE_MINIMUM,
			RENDERED_SIZE_MAXIMUM,
		).
		Ensure()
}

// DEFAULT_SIZE_PREFIX includes punctuation around one rendered default value.
const DEFAULT_SIZE_PREFIX = len("(default: )")

// DEFAULT_CELL_SIZE_HOLE_ONE cannot contain the prefix and one rendered value.
const DEFAULT_CELL_SIZE_HOLE_ONE = len("x")

// DEFAULT_CELL_SIZE_HOLE_TWO cannot contain the prefix and one rendered value.
const DEFAULT_CELL_SIZE_HOLE_TWO = len("xx")

// Default_Cell_Size_Value excludes padding widths shorter than rendered syntax.
type Default_Cell_Size_Value int

// Default_Cell_Size_Value_Invariants preserves zero for Boolean flags only.
func Default_Cell_Size_Value_Invariants(
	value Default_Cell_Size_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Int(
			int(value), RENDERED_SIZE_MINIMUM, RENDERED_SIZE_MAXIMUM,
			DEFAULT_CELL_SIZE_HOLE_ONE, DEFAULT_CELL_SIZE_HOLE_TWO,
			DEFAULT_CELL_SIZE_HOLE_TWO, DEFAULT_CELL_SIZE_HOLE_TWO,
		).
		Ensure()
}

// Zero remains necessary because a section can contain only Boolean flags.
type Default_Cell_Size struct {
	// Value keeps aligned default columns bounded by output capacity.
	Value Default_Cell_Size_Value
}

// Default_Cell_Size_Invariants preserves absent and rendered default widths.
func Default_Cell_Size_Invariants(value Default_Cell_Size, namespace aver.Namespace) {
	Default_Cell_Size_Value_Invariants(value.Value, namespace)
}

// RENDERED_OPTION_COUNT follows one option borrowed by a rendering phase.
const RENDERED_OPTION_COUNT = len("option") / len("option")

// Rendered_Option isolates rendering fields from construction and parse contracts.
type Rendered_Option struct {
	// Label is rendered option identity.
	Label Option_Label
	// Description is rendered help text.
	Description Description
	// Type selects rendered scalar or collection syntax.
	Type Option_Type_State
	// Enumeration supplies rendered permitted members.
	Enumeration Option_Enumeration
	// String is the text default arm.
	String Value_Text
	// Integer is the integer default arm.
	Integer Integer
}

// Rendered_Option_Invariants checks only state read while rendering.
func Rendered_Option_Invariants(value Rendered_Option, namespace aver.Namespace) {
	Option_Label_Invariants(value.Label, namespace)
	Description_Invariants(value.Description, namespace)
	Option_Type_State_Invariants(value.Type, namespace)
	Option_Enumeration_Invariants(value.Enumeration, namespace)
	Value_Text_Invariants(value.String, namespace)
	Integer_Invariants(value.Integer, namespace)
}

// Rendered_External is the resolved scalar exposed by environment help.
type Rendered_External struct {
	// Type selects one scalar arm.
	Type External_Type_State
	// String is the text arm.
	String Value_Text
	// Integer is the integer arm.
	Integer Integer
	// Boolean is the Boolean arm.
	Boolean Boolean
}

// Rendered_External_Invariants composes only rendered scalar fields.
func Rendered_External_Invariants(
	value Rendered_External, namespace aver.Namespace,
) {
	External_Type_State_Invariants(value.Type, namespace)
	Value_Text_Invariants(value.String, namespace)
	Integer_Invariants(value.Integer, namespace)
	Boolean_Invariants(value.Boolean, namespace)
}

// RENDERED_INTEGER_COUNT follows one integer borrowed by a rendering phase.
const RENDERED_INTEGER_COUNT = len("integer") / len("integer")

// Rendered_Integer keeps machine-range proof outside phase-specific namespaces.
type Rendered_Integer struct {
	// Value is machine integer rendered by help or diagnostics.
	Value Integer
}

// Rendered_Integer_Invariants fixes the one machine integer cell.
func Rendered_Integer_Invariants(value Rendered_Integer, namespace aver.Namespace) {
	Integer_Invariants(value.Value, namespace)
}

func output_write(output Output_Reference, source Value_Text) {
	Output_Reference_Invariants(output, "render_write.output")
	Value_Text_Invariants(source, "render_write.source")
	strings.Builder_Write_Text(
		strings.Builder_Handle((*strings.Builder)((*Output_Builder)(output.Value))),
		strings.Text(source),
	)
}

func output_write_newline(output Output_Reference) {
	Output_Reference_Invariants(output, "render_write_newline.output")
	strings.Builder_Write_Byte(
		strings.Builder_Handle((*strings.Builder)((*Output_Builder)(output.Value))),
		'\n',
	)
}

func output_write_space(output Output_Reference) {
	Output_Reference_Invariants(output, "render_write_space.output")
	strings.Builder_Write_Byte(
		strings.Builder_Handle((*strings.Builder)((*Output_Builder)(output.Value))),
		' ',
	)
}

func render_decimal(
	output Output_Reference, rendered Rendered_Integer,
) (size strconv.Decimal_Text_Count) {
	defer func() { strconv.Decimal_Text_Count_Invariants(size, "render_decimal.size") }()
	Output_Reference_Invariants(output, "render_decimal.output")
	Rendered_Integer_Invariants(rendered, "render_decimal.rendered")
	var storage [strconv.DECIMAL_TEXT_SIZE_MAXIMUM]byte
	size = strconv.Format_Decimal_Into(
		storage[:], strconv.Machine_Integer(rendered.Value),
	)
	if output.Value != nil {
		Output_Write(output, storage[:size])
	}
	return size
}

func option_signature_write(
	output Output_Reference, label Option_Label,
	type_state Option_Type_State, enumeration Option_Enumeration,
) {
	Output_Reference_Invariants(output, "option_signature_write.output")
	Option_Label_Invariants(label, "option_signature_write.label")
	Option_Type_State_Invariants(type_state, "option_signature_write.type_state")
	Option_Enumeration_Invariants(enumeration, "option_signature_write.enumeration")
	output_write(output, "<")
	output_write(output, Value_Text(label))
	output_write(output, ": ")
	if option_is_slice(Option_Type(type_state.Value)) {
		output_write(output, Value_Text(option_type_text(
			Scalar_Option_Type(type_state.Value-OPTION_TYPE_STRINGS),
		)))
		output_write(output, "...>")
		return
	}
	if enumeration.String != nil {
		output_write(output, "(")
		for index, member := range enumeration.String {
			if index != 0 {
				output_write(output, "|")
			}
			output_write(output, Value_Text(member))
		}
		output_write(output, ")>")
		return
	}
	if enumeration.Integers != nil {
		output_write(output, "(")
		for index, member := range enumeration.Integers {
			if index != 0 {
				output_write(output, "|")
			}
			render_decimal(output, Rendered_Integer{Value: Integer(member)})
		}
		output_write(output, ")>")
		return
	}
	output_write(output, Value_Text(option_type_text(
		Scalar_Option_Type(type_state.Value),
	)))
	output_write(output, ">")
}

// Print_Help writes a formatted help message to output: the program header,
// usage syntax, global flags, and each command with its arguments and flags.
func Print_Help(output Output_Reference, program Program) {
	Output_Reference_Invariants(output, "print_help.output")
	Program_Invariants(program, "print_help.program")
	reference := output
	output_write(reference, Value_Text(program.Label))
	output_write(reference, " ")
	output_write(reference, Value_Text(program.Description))
	output_write(reference, "\n\n")
	if program.Selection.Mode.Value == PROGRAM_MODE_SINGLE {
		Print_Command(output, program, program.Selection.Single_Commands.Command)
		return
	}
	output_write(reference, "Usage:\n    ")
	output_write(reference, Value_Text(program.Label))
	output_write(reference, " <command> <arguments> [-flags[=value]]\n")
	has_arguments := false
	for _, command := range program.Selection.Commands {
		if len(command.Arguments) > 0 {
			has_arguments = true
			break
		}
	}
	if has_arguments {
		output_write(reference, NAMED_ARGUMENT_LEGEND)
		output_write(reference, "\n")
	}
	output_write(reference, "\n")

	print_help_global_flag_section(
		reference, program.Selection.Help_Flags, program.Selection.Global_Flags)
	print_environment_help(reference, program.Environment_Variables)
	print_secret_help(reference, Secrets(program.Secrets))
	print_command_catalog(reference, program.Selection.Commands)
}

func print_command_catalog(output Output_Reference, commands Command_Declarations) {
	Output_Reference_Invariants(output, "print_command_catalog.output")
	Command_Declarations_Invariants(commands, "print_command_catalog.commands")
	output_write(output, "Available Commands:\n")
	for _, command := range commands {
		if !command_shown(command.Hidden, command.Deprecated) {
			continue
		}
		cell_width, default_width := Flag_Cell_Size{}, Default_Cell_Size{}
		has_command_flag := false
		for _, flag := range command.Flags {
			if !option_shown(flag.State.Hidden, flag.State.Deprecated) {
				continue
			}
			has_command_flag = true
			rendered := Rendered_Option{
				Label: flag.Label, Description: flag.Description, Type: flag.Type,
				Enumeration: flag.Enumeration, String: flag.State.String,
				Integer: flag.State.Integer,
			}
			cell_width.Value = max(
				cell_width.Value,
				help_flag_cell_size(
					rendered.Label, Scalar_Option_Type(rendered.Type.Value),
					rendered.Enumeration,
					INDENT_COMMAND_FLAG, false,
				).Value,
			)
			if flag.Type.Value != OPTION_TYPE_BOOLEAN {
				default_size := Default_Cell_Size_Value(DEFAULT_SIZE_PREFIX) +
					Default_Cell_Size_Value(render_decimal(Output_Reference{},
						Rendered_Integer{Value: rendered.Integer},
					))
				if flag.Type.Value == OPTION_TYPE_STRING {
					default_size = Default_Cell_Size_Value(
						DEFAULT_SIZE_PREFIX + len(rendered.String))
					if rendered.String == "" {
						default_size += Default_Cell_Size_Value(len(`""`))
					}
				}
				default_width.Value = max(default_width.Value, default_size)
			}
		}
		output_write(output, "    \033[34m")
		output_write(output, Value_Text(command.Label))
		output_write(output, "\033[0m ")
		for _, argument := range command.Arguments {
			rendered := Rendered_Option{
				Label: argument.Label, Description: argument.Description,
				Type: argument.Type, Enumeration: argument.Enumeration,
				String: argument.State.String, Integer: argument.State.Integer,
			}
			option_signature_write(
				output, rendered.Label, rendered.Type, rendered.Enumeration)
			output_write(output, " ")
		}
		output_write(output, Value_Text(command.Description))
		output_write(output, "\n")
		if has_command_flag {
			output_write(output, "\n")
			print_help_flags(
				output, Options(command.Flags), INDENT_COMMAND_FLAG, false,
				cell_width, default_width,
			)
		}
		output_write(output, "\n")
	}
}

// Visible flags reuse widths computed across their whole section.
func print_help_flags(
	output Output_Reference, flags Options, indent Indent, color Boolean,
	cell_width Flag_Cell_Size, default_width Default_Cell_Size,
) {
	Output_Reference_Invariants(output, "print_help_flags.output")
	Options_Invariants(flags, "print_help_flags.flags")
	Indent_Invariants(indent, "print_help_flags.indent")
	Boolean_Invariants(color, "print_help_flags.color")
	Flag_Cell_Size_Invariants(cell_width, "print_help_flags.cell_width")
	Default_Cell_Size_Invariants(default_width, "print_help_flags.default_width")
	for _, flag := range flags {
		if !option_shown(flag.State.Hidden, flag.State.Deprecated) {
			continue
		}
		rendered := Rendered_Option{
			Label: flag.Label, Description: flag.Description, Type: flag.Type,
			Enumeration: flag.Enumeration, String: flag.State.String,
			Integer: flag.State.Integer,
		}
		print_help_flag(
			output, rendered, indent, color,
			cell_width, default_width,
		)
	}
}

// Print_Requested_Help renders the help a HELP_REQUESTED parse asks for. Empty selected
// identity renders program help; any selected identity resolves from immutable Program.
func Print_Requested_Help(
	output Output_Reference, program Program, selected Label,
) {
	Output_Reference_Invariants(output, "print_requested_help.output")
	Program_Invariants(program, "print_requested_help.program")
	Label_Invariants(selected, "print_requested_help.selected")
	if selected == "" {
		Print_Help(output, program)
		return
	}
	command := program.Selection.Single_Commands.Command
	found := command.Label == selected
	if !found {
		for _, candidate := range program.Selection.Commands {
			if candidate.Label == selected {
				command = candidate
				found = true
				break
			}
		}
	}
	aver.Always(found, "Requested help command belongs to Program.")
	Print_Command(output, program, command)
}

// Print_Command writes help for a single command: a usage line carrying the command's
// own name and positional arguments, then its flags and the program's global flags. It
// is the per-verb help a multicall binary shows for the name in argv[0], and the body
// of a single-command program's help.
func Print_Command(output Output_Reference, program Program, command Command) {
	Output_Reference_Invariants(output, "print_command.output")
	Program_Invariants(program, "print_command.program")
	Command_Invariants(command, "print_command.command")
	reference := output
	output_write(reference, "Usage:\n    ")
	output_write(reference, Value_Text(command.Label))
	output_write(reference, " ")
	for _, argument := range command.Arguments {
		rendered := Rendered_Option{
			Label: argument.Label, Description: argument.Description,
			Type: argument.Type, Enumeration: argument.Enumeration,
			String: argument.State.String, Integer: argument.State.Integer,
		}
		option_signature_write(
			reference, rendered.Label, rendered.Type, rendered.Enumeration,
		)
		output_write(reference, " ")
	}
	output_write(reference, "[-flags[=value]]\n")
	if len(command.Arguments) > 0 {
		output_write(reference, NAMED_ARGUMENT_LEGEND)
		output_write(reference, "\n")
	}
	print_help_flag_section(reference, FLAG_SECTION_TITLE, Options(command.Flags))
	print_help_global_flag_section(
		reference, program.Selection.Help_Flags, program.Selection.Global_Flags)
	print_environment_help(reference, program.Environment_Variables)
	print_secret_help(reference, Secrets(program.Secrets))
}

// Print_Deprecations writes each warning gathered by Program_Parse. A binary calls it
// after a successful parse, before running, so deprecated inputs remain usable but noisy.
func Print_Deprecations(
	output Output_Reference, warnings Resolved_Deprecation_Warnings,
) {
	Output_Reference_Invariants(output, "print_deprecations.output")
	Resolved_Deprecation_Warnings_Invariants(warnings, "print_deprecations.warnings")
	reference := output
	for _, warning := range warnings {
		Warning_Invariants(warning, "print_deprecations.warning")
		output_write(reference, "warning: ")
		if warning.Guidance.Value == "" {
			output_write(reference, "\n")
			continue
		}
		switch warning.Kind.Value {
		case WARNING_KIND_COMMAND:
			output_write(reference, "command \"")
			output_write(reference, warning.Name.Value)
			output_write(reference, "\"")
		case WARNING_KIND_FLAG:
			output_write(reference, "-")
			output_write(reference, warning.Name.Value)
		case WARNING_KIND_ENVIRONMENT:
			output_write(reference, ENVIRONMENT_WARNING_PREFIX)
			output_write(reference, warning.Name.Value)
		case WARNING_KIND_SECRET:
			output_write(reference, "secret ")
			output_write(reference, warning.Name.Value)
		}
		output_write(reference, " is deprecated: ")
		output_write(reference, Value_Text(warning.Guidance.Value))
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
		if option_shown(flag.State.Hidden, flag.State.Deprecated) {
			has_shown = true
			break
		}
	}
	if !has_shown {
		return
	}
	cell_width, default_width := Flag_Cell_Size{}, Default_Cell_Size{}
	for _, flag := range flags {
		if !option_shown(flag.State.Hidden, flag.State.Deprecated) {
			continue
		}
		rendered := Rendered_Option{
			Label: flag.Label, Description: flag.Description, Type: flag.Type,
			Enumeration: flag.Enumeration, String: flag.State.String,
			Integer: flag.State.Integer,
		}
		cell_width.Value = max(
			cell_width.Value,
			help_flag_cell_size(
				rendered.Label, Scalar_Option_Type(rendered.Type.Value),
				rendered.Enumeration, INDENT_FLAG, true,
			).Value,
		)
		if flag.Type.Value != OPTION_TYPE_BOOLEAN {
			default_size := Default_Cell_Size_Value(DEFAULT_SIZE_PREFIX) +
				Default_Cell_Size_Value(render_decimal(Output_Reference{},
					Rendered_Integer{Value: rendered.Integer},
				))
			if flag.Type.Value == OPTION_TYPE_STRING {
				default_size = Default_Cell_Size_Value(
					DEFAULT_SIZE_PREFIX + len(rendered.String))
				if rendered.String == "" {
					default_size += Default_Cell_Size_Value(len(`""`))
				}
			}
			default_width.Value = max(default_width.Value, default_size)
		}
	}
	output_write(output, "\n")
	output_write(output, Value_Text(title))
	output_write(output, "\n")
	print_help_flags(output, flags, INDENT_FLAG, true, cell_width, default_width)
}

// Reserved and user flags share one visible section while retaining separate storage.
func print_help_global_flag_section(
	output Output_Reference, help_flags Help_Flags, global_flags Global_Flags,
) {
	Output_Reference_Invariants(output, "print_help_global_flag_section.output")
	Help_Flags_Invariants(help_flags, "print_help_global_flag_section.help_flags")
	Global_Flags_Invariants(global_flags, "print_help_global_flag_section.global_flags")
	short_label, long_label := Option_Label(help_flags.Short_Label),
		Option_Label(help_flags.Long_Label)
	type_state := Scalar_Option_Type(OPTION_TYPE_BOOLEAN)
	enumeration := Option_Enumeration{}
	help_shown := Boolean(help_hidden_validate(help_flags.Hidden) == HELP_HIDDEN_VISIBLE)
	has_shown := help_shown
	for _, flag := range global_flags {
		has_shown = has_shown || option_shown(flag.State.Hidden, flag.State.Deprecated)
	}
	if !has_shown {
		return
	}
	cell_width, default_width := Flag_Cell_Size{}, Default_Cell_Size{}
	short_width := help_flag_cell_size(short_label, type_state, enumeration, INDENT_FLAG, true)
	long_width := help_flag_cell_size(long_label, type_state, enumeration, INDENT_FLAG, true)
	if help_shown {
		cell_width.Value = max(cell_width.Value, short_width.Value)
		cell_width.Value = max(cell_width.Value, long_width.Value)
	}
	for _, flag := range global_flags {
		if !option_shown(flag.State.Hidden, flag.State.Deprecated) {
			continue
		}
		rendered := Rendered_Option{
			Label: flag.Label, Description: flag.Description, Type: flag.Type,
			Enumeration: flag.Enumeration, String: flag.State.String,
			Integer: flag.State.Integer,
		}
		cell_width.Value = max(
			cell_width.Value,
			help_flag_cell_size(
				rendered.Label, Scalar_Option_Type(rendered.Type.Value),
				rendered.Enumeration, INDENT_FLAG, true,
			).Value,
		)
		if flag.Type.Value != OPTION_TYPE_BOOLEAN {
			default_size := Default_Cell_Size_Value(DEFAULT_SIZE_PREFIX) +
				Default_Cell_Size_Value(render_decimal(Output_Reference{},
					Rendered_Integer{Value: rendered.Integer},
				))
			if flag.Type.Value == OPTION_TYPE_STRING {
				default_size = Default_Cell_Size_Value(
					DEFAULT_SIZE_PREFIX + len(rendered.String))
				if rendered.String == "" {
					default_size += Default_Cell_Size_Value(len(`""`))
				}
			}
			default_width.Value = max(default_width.Value, default_size)
		}
	}
	output_write(output, "Global Flags:\n")
	if help_shown {
		for _, label := range [...]Option_Label{short_label, long_label} {
			print_help_flag(output, Rendered_Option{
				Label: label, Description: Description(help_flags.Description),
				Type: Option_Type_State{Value: OPTION_TYPE_BOOLEAN},
			}, INDENT_FLAG, true, cell_width, default_width)
		}
	}
	print_help_flags(
		output, Options(global_flags), INDENT_FLAG, true,
		cell_width, default_width,
	)
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
		text = "string"
	case OPTION_TYPE_INTEGER, OPTION_TYPE_INTEGERS:
		text = "int"
	case OPTION_TYPE_BOOLEAN:
		text = "bool"
	}
	aver.Always(text != "", "Option type has rendered text.")
	return text
}

// The note printed under the usage line when a program has at least one positional
// argument: each may also be supplied by name, not only by position.
const NAMED_ARGUMENT_LEGEND = "    Positional arguments may also be supplied via -key=val syntax."

func help_flag_cell_size(
	label Option_Label, type_state Scalar_Option_Type,
	enumeration Option_Enumeration, indent Indent, color Boolean,
) (size Flag_Cell_Size) {
	defer func() { Flag_Cell_Size_Invariants(size, "help_flag_cell_size.size") }()
	Option_Label_Invariants(label, "help_flag_cell_size.label")
	Scalar_Option_Type_Invariants(type_state, "help_flag_cell_size.type_state")
	Option_Enumeration_Invariants(enumeration, "help_flag_cell_size.enumeration")
	Indent_Invariants(indent, "help_flag_cell_size.indent")
	Boolean_Invariants(color, "help_flag_cell_size.color")
	size.Value = Flag_Cell_Size_Value(len("    ") + len("-") + len(label))
	if indent == INDENT_COMMAND_FLAG {
		size.Value += Flag_Cell_Size_Value(len("    "))
	}
	if color {
		size.Value += Flag_Cell_Size_Value(len("\033[34m") + len("\033[0m"))
	}
	if Option_Type(type_state) == OPTION_TYPE_BOOLEAN {
		return size
	}
	size.Value += Flag_Cell_Size_Value(len("="))
	if enumeration.String != nil {
		size.Value += Flag_Cell_Size_Value(len("()"))
		for index, member := range enumeration.String {
			if index != 0 {
				size.Value += Flag_Cell_Size_Value(len("|"))
			}
			size.Value += Flag_Cell_Size_Value(len(member))
		}
		return size
	}
	if enumeration.Integers != nil {
		size.Value += Flag_Cell_Size_Value(len("()"))
		for index, member := range enumeration.Integers {
			if index != 0 {
				size.Value += Flag_Cell_Size_Value(len("|"))
			}
			size.Value += Flag_Cell_Size_Value(render_decimal(Output_Reference{},
				Rendered_Integer{Value: Integer(member)},
			))
		}
		return size
	}
	size.Value += Flag_Cell_Size_Value(len(option_type_text(type_state)))
	return size
}

func help_flag_cell_write(
	output Output_Reference, label Option_Label, type_state Scalar_Option_Type,
	enumeration Option_Enumeration, indent Indent, color Boolean,
) {
	Output_Reference_Invariants(output, "help_flag_cell_write.output")
	Option_Label_Invariants(label, "help_flag_cell_write.label")
	Scalar_Option_Type_Invariants(type_state, "help_flag_cell_write.type_state")
	Option_Enumeration_Invariants(enumeration, "help_flag_cell_write.enumeration")
	Indent_Invariants(indent, "help_flag_cell_write.indent")
	Boolean_Invariants(color, "help_flag_cell_write.color")
	output_write(output, "    ")
	if indent == INDENT_COMMAND_FLAG {
		output_write(output, "    ")
	}
	output_write(output, "-")
	if color {
		output_write(output, "\033[34m")
	}
	output_write(output, Value_Text(label))
	if color {
		output_write(output, "\033[0m")
	}
	if Option_Type(type_state) == OPTION_TYPE_BOOLEAN {
		return
	}
	output_write(output, "=")
	if enumeration.String != nil {
		output_write(output, "(")
		for index, member := range enumeration.String {
			if index != 0 {
				output_write(output, "|")
			}
			output_write(output, Value_Text(member))
		}
		output_write(output, ")")
		return
	}
	if enumeration.Integers != nil {
		output_write(output, "(")
		for index, member := range enumeration.Integers {
			if index != 0 {
				output_write(output, "|")
			}
			render_decimal(output, Rendered_Integer{Value: Integer(member)})
		}
		output_write(output, ")")
		return
	}
	output_write(output, Value_Text(option_type_text(type_state)))
}

func print_help_flag(
	output Output_Reference, rendered Rendered_Option, indent Indent, color Boolean,
	cell_width Flag_Cell_Size, default_width Default_Cell_Size,
) {
	Flag_Cell_Size_Invariants(cell_width, "print_help_flag.cell_width")
	Default_Cell_Size_Invariants(default_width, "print_help_flag.default_width")
	Output_Reference_Invariants(output, "print_help_flag.output")
	Rendered_Option_Invariants(rendered, "print_help_flag.rendered")
	Indent_Invariants(indent, "print_help_flag.indent")
	Boolean_Invariants(color, "print_help_flag.color")
	flag := rendered
	cell_size := help_flag_cell_size(
		flag.Label, Scalar_Option_Type(flag.Type.Value), flag.Enumeration, indent, color,
	)
	help_flag_cell_write(
		output, flag.Label, Scalar_Option_Type(flag.Type.Value),
		flag.Enumeration, indent, color,
	)
	for range cell_width.Value - cell_size.Value {
		output_write_space(output)
	}
	output_write(output, "  ")
	if flag.Type.Value != OPTION_TYPE_BOOLEAN {
		default_size := Default_Cell_Size_Value(DEFAULT_SIZE_PREFIX) +
			Default_Cell_Size_Value(render_decimal(
				Output_Reference{}, Rendered_Integer{Value: flag.Integer}))
		output_write(output, "(default: ")
		if flag.Type.Value == OPTION_TYPE_STRING {
			default_size = Default_Cell_Size_Value(
				DEFAULT_SIZE_PREFIX + len(flag.String))
			if flag.String == "" {
				default_size += Default_Cell_Size_Value(len(`""`))
				output_write(output, `""`)
			} else {
				output_write(output, flag.String)
			}
		} else {
			render_decimal(output, Rendered_Integer{Value: flag.Integer})
		}
		output_write(output, ")")
		for range default_width.Value - default_size {
			output_write_space(output)
		}
		output_write(output, "  ")
	}
	output_write(output, Value_Text(flag.Description))
	output_write(output, "\n")
}

// Get_Option retrieves an option by label from a slice of options, panicking
// when the label is not found. Typed accessors read its fixed union storage.
func Get_Option(
	flags Resolved_Options, label Option_Label,
) (option Resolved_Option) {
	defer func() { Resolved_Option_Invariants(option, "get_option.option") }()
	Resolved_Options_Invariants(flags, "get_option.flags")
	Option_Label_Invariants(label, "get_option.label")
	found := false
	for _, flag := range flags {
		if flag.Label == label {
			option = flag
			found = true
			break
		}
	}
	aver.Always(found, "Option is declared.")
	return option
}

// Option_String returns one named text scalar without publishing the full tagged state.
func Option_String(options Resolved_Options, label Option_Label) (value Value_Text) {
	defer func() { Value_Text_Invariants(value, "option_string.value") }()
	Resolved_Options_Invariants(options, "option_string.options")
	Option_Label_Invariants(label, "option_string.label")
	option := Get_Option(options, label)
	aver.Always(
		option.Type.Value == OPTION_TYPE_STRING,
		"Option_String receives text scalar storage.",
	)
	return Value_Text(option.State.String)
}

// Option_Integer returns one named integer scalar without publishing tagged state.
func Option_Integer(options Resolved_Options, label Option_Label) (value Integer) {
	defer func() { Integer_Invariants(value, "option_integer.value") }()
	Resolved_Options_Invariants(options, "option_integer.options")
	Option_Label_Invariants(label, "option_integer.label")
	option := Get_Option(options, label)
	aver.Always(
		option.Type.Value == OPTION_TYPE_INTEGER,
		"Option_Integer receives integer scalar storage.",
	)
	return Integer(option.State.Integer)
}

// Option_Boolean returns one named Boolean scalar without publishing tagged state.
func Option_Boolean(options Resolved_Options, label Option_Label) (value Boolean) {
	defer func() { Boolean_Invariants(value, "option_boolean.value") }()
	Resolved_Options_Invariants(options, "option_boolean.options")
	Option_Label_Invariants(label, "option_boolean.label")
	option := Get_Option(options, label)
	aver.Always(
		option.Type.Value == OPTION_TYPE_BOOLEAN,
		"Option_Boolean receives Boolean scalar storage.",
	)
	return option.State.Boolean
}

// Option_Strings returns one named caller-owned text collection.
func Option_Strings(options Resolved_Options, label Option_Label) (value String_Values) {
	defer func() { String_Values_Invariants(value, "option_strings.value") }()
	Resolved_Options_Invariants(options, "option_strings.options")
	Option_Label_Invariants(label, "option_strings.label")
	option := Get_Option(options, label)
	aver.Always(
		option.Type.Value == OPTION_TYPE_STRINGS,
		"Option_Strings receives text collection storage.",
	)
	return String_Values(option.State.Strings)
}

// Option_Integers returns one named caller-owned integer collection.
func Option_Integers(options Resolved_Options, label Option_Label) (value Integer_Values) {
	defer func() { Integer_Values_Invariants(value, "option_integers.value") }()
	Resolved_Options_Invariants(options, "option_integers.options")
	Option_Label_Invariants(label, "option_integers.label")
	option := Get_Option(options, label)
	aver.Always(
		option.Type.Value == OPTION_TYPE_INTEGERS,
		"Option_Integers receives integer collection storage.",
	)
	return Integer_Values(option.State.Integers)
}

// Builder_Handle keeps size-only rendering valid before output is bound.
type Builder_Handle *Output_Builder

// Builder_Handle_Invariants composes state only after output is bound.
func Builder_Handle_Invariants(value Builder_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Output_Builder_Invariants(*value, namespace)
}

// Candidate_Write renders one completion directly into caller-owned output.
func Candidate_Write(output Output_Reference, candidate Candidate) {
	Output_Reference_Invariants(output, "candidate_write.output")
	Candidate_Invariants(candidate, "candidate_write.candidate")
	segments := [CANDIDATE_SEGMENT_COUNT]string{
		string(candidate.Segments.Dash), string(candidate.Segments.Label),
		string(candidate.Segments.Equals), string(candidate.Segments.Value),
	}
	for _, segment := range segments {
		output_write(output, Value_Text(segment))
	}
	if candidate.State.Has_Integer {
		var storage [strconv.DECIMAL_TEXT_SIZE_MAXIMUM]byte
		count := strconv.Format_Decimal_Into(
			storage[:], strconv.Machine_Integer(candidate.State.Integer),
		)
		Output_Write(output, storage[:count])
	}
}

// Candidate_Equal compares rendered candidate text without creating joined text.
func Candidate_Equal(candidate Candidate, expected Value_Text) (equal Boolean) {
	defer func() { Boolean_Invariants(equal, "candidate_equal.equal") }()
	Candidate_Invariants(candidate, "candidate_equal.candidate")
	Value_Text_Invariants(expected, "candidate_equal.expected")
	size := len(candidate.Segments.Dash) + len(candidate.Segments.Label) +
		len(candidate.Segments.Equals) + len(candidate.Segments.Value)
	var number_storage [strconv.DECIMAL_TEXT_SIZE_MAXIMUM]byte
	number := Candidate_Number(number_storage[:])
	number_size := 0
	if candidate.State.Has_Integer {
		number_size = int(strconv.Format_Decimal_Into(
			strconv.Buffer(number), strconv.Machine_Integer(candidate.State.Integer)))
		size += number_size
	}
	parts := Candidate_Parts{
		string(candidate.Segments.Dash), string(candidate.Segments.Label),
		string(candidate.Segments.Equals), string(candidate.Segments.Value),
	}
	if !candidate_has_prefix(parts, number, expected) {
		return false
	}
	return size == len(expected)
}

// Candidate prefix comparison never joins borrowed segments.
func candidate_has_prefix(
	parts Candidate_Parts, number Candidate_Number, prefix Value_Text,
) (present Boolean) {
	defer func() { Boolean_Invariants(present, "candidate_has_prefix.present") }()
	Candidate_Parts_Invariants(parts, "candidate_has_prefix.parts")
	Candidate_Number_Invariants(number, "candidate_has_prefix.number")
	Value_Text_Invariants(prefix, "candidate_has_prefix.prefix")
	prefix_index := 0
	for _, part := range parts {
		for part_index := range part {
			if prefix_index == len(prefix) {
				return true
			}
			if part[part_index] != prefix[prefix_index] {
				return false
			}
			prefix_index++
		}
	}
	for _, digit := range number {
		if digit == 0 {
			break
		}
		if prefix_index == len(prefix) {
			return true
		}
		if digit != prefix[prefix_index] {
			return false
		}
		prefix_index++
	}
	return prefix_index == len(prefix)
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
		if !command_shown(commands[index].Hidden, commands[index].Deprecated) {
			continue
		}
		candidate := Candidate{}
		candidate.Segments.Dash = Candidate_Dash(commands[index].Label)
		parts := Candidate_Parts{string(candidate.Segments.Dash), "", "", ""}
		if candidate_has_prefix(parts, Candidate_Number{}, current) {
			aver.Always(
				len(candidates) < len(storage),
				"Command completion candidate has storage.",
			)
			count := len(candidates)
			candidates = Command_Candidates(storage[:count+1])
			candidates[count] = candidate
		}
	}
	return candidates
}

// Complete option label conditionally publishes one named option.
func complete_option_label(
	label Option_Label, shown Boolean, current Option_Completion_Prefix,
	storage Candidates, candidates Writable_Candidates,
) (completed Candidates) {
	defer func() {
		Candidates_Invariants(completed, "complete_option_label.completed")
	}()
	Option_Label_Invariants(label, "complete_option_label.label")
	Boolean_Invariants(shown, "complete_option_label.shown")
	Option_Completion_Prefix_Invariants(current, "complete_option_label.current")
	Candidates_Invariants(storage, "complete_option_label.storage")
	Writable_Candidates_Invariants(candidates, "complete_option_label.candidates")
	if !shown {
		return Candidates(candidates)
	}
	candidate := Candidate{}
	candidate.Segments.Dash = "-"
	candidate.Segments.Label = Candidate_Label(label)
	parts := Candidate_Parts{
		string(candidate.Segments.Dash), string(candidate.Segments.Label), "", "",
	}
	if !candidate_has_prefix(parts, Candidate_Number{}, Value_Text(current)) {
		return Candidates(candidates)
	}
	aver.Always(
		len(candidates) < len(storage),
		"Option completion candidate has storage.",
	)
	count := len(candidates)
	completed = storage[:count+1]
	completed[count] = candidate
	return completed
}

// Complete option labels writes every named option reachable from this command.
func complete_option_labels(
	global_flags Global_Flags, help_labels Help_Labels,
	arguments Arguments, flags Flags,
	current Option_Completion_Prefix, storage Candidates,
) (candidates Candidates) {
	defer func() {
		Candidates_Invariants(candidates, "complete_option_labels.candidates")
	}()
	Global_Flags_Invariants(global_flags, "complete_option_labels.global_flags")
	Help_Labels_Invariants(help_labels, "complete_option_labels.help_labels")
	Arguments_Invariants(arguments, "complete_option_labels.arguments")
	Flags_Invariants(flags, "complete_option_labels.flags")
	Option_Completion_Prefix_Invariants(current, "complete_option_labels.current")
	Candidates_Invariants(storage, "complete_option_labels.storage")
	candidates = storage[:0]
	for index := range arguments {
		candidates = complete_option_label(
			arguments[index].Label, true, current, storage,
			Writable_Candidates(candidates),
		)
	}
	for index := range flags {
		candidates = complete_option_label(
			flags[index].Label,
			option_shown(flags[index].State.Hidden, flags[index].State.Deprecated),
			current, storage,
			Writable_Candidates(candidates),
		)
	}
	shown := Boolean(help_labels.Hidden == HELP_HIDDEN_VISIBLE)
	candidates = complete_option_label(
		Option_Label(help_labels.Short), shown, current,
		storage, Writable_Candidates(candidates),
	)
	candidates = complete_option_label(
		Option_Label(help_labels.Long), shown, current,
		storage, Writable_Candidates(candidates),
	)
	for index := range global_flags {
		candidates = complete_option_label(
			global_flags[index].Label,
			option_shown(
				global_flags[index].State.Hidden,
				global_flags[index].State.Deprecated,
			),
			current, storage, Writable_Candidates(candidates),
		)
	}
	return candidates
}

// Complete enum members writes borrowed string or scalar integer members.
func complete_enum_members_result(
	candidates Enum_Candidates, is_enum Boolean,
) (_ Enum_Candidates, _ Boolean) {
	Enum_Candidates_Invariants(candidates, "complete_enum_members.candidates")
	Boolean_Invariants(is_enum, "complete_enum_members.is_enum")
	return candidates, is_enum
}

func complete_enum_members(
	enumeration Option_Enumeration, dash Enum_Candidate_Dash,
	label Completion_Option_Label, equals Enum_Candidate_Equals,
	current Value_Text, storage Candidates,
) (_ Enum_Candidates, _ Boolean) {
	Option_Enumeration_Invariants(enumeration, "complete_enum_members.enumeration")
	Enum_Candidate_Dash_Invariants(dash, "complete_enum_members.dash")
	Completion_Option_Label_Invariants(label, "complete_enum_members.label")
	Enum_Candidate_Equals_Invariants(equals, "complete_enum_members.equals")
	Value_Text_Invariants(current, "complete_enum_members.current")
	Candidates_Invariants(storage, "complete_enum_members.storage")
	candidates := Enum_Candidates(storage[:0])
	member_count := len(enumeration.String)
	is_enum := Boolean(false)
	if enumeration.String != nil {
		is_enum = true
	} else if enumeration.Integers != nil {
		member_count = len(enumeration.Integers)
		is_enum = true
	}
	for member_index := range member_count {
		candidate := Candidate{Segments: Candidate_Segments{
			Dash: Candidate_Dash(dash), Label: Candidate_Label(label),
			Equals: Candidate_Equals(equals),
		}}
		parts := Candidate_Parts{
			string(candidate.Segments.Dash), string(candidate.Segments.Label),
			string(candidate.Segments.Equals), "",
		}
		var number_storage [strconv.DECIMAL_TEXT_SIZE_MAXIMUM]byte
		number := Candidate_Number(number_storage[:0])
		if enumeration.String != nil {
			candidate.Segments.Value = Candidate_Value(enumeration.String[member_index])
			parts[3] = string(candidate.Segments.Value)
		} else {
			member := enumeration.Integers[member_index]
			candidate.State.Integer = Integer(member)
			candidate.State.Has_Integer = true
			number = Candidate_Number(number_storage[:])
			strconv.Format_Decimal_Into(
				strconv.Buffer(number), strconv.Machine_Integer(member),
			)
		}
		if !candidate_has_prefix(parts, number, current) {
			continue
		}
		aver.Always(len(candidates) < len(storage),
			"Enum completion candidate has storage.")
		count := len(candidates)
		candidates = Enum_Candidates(storage[:count+1])
		candidates[count] = candidate
	}
	return complete_enum_members_result(candidates, is_enum)
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
	help_labels := Help_Labels{
		Short:  program.Selection.Help_Flags.Short_Label,
		Long:   program.Selection.Help_Flags.Long_Label,
		Hidden: help_hidden_validate(program.Selection.Help_Flags.Hidden),
	}

	if program.Selection.Mode.Value == PROGRAM_MODE_SINGLE {
		return complete_in_command(
			program.Selection.Global_Flags, help_labels,
			program.Selection.Single_Commands.Command.Arguments,
			program.Selection.Single_Commands.Command.Flags,
			Completion_Words(words), false,
			current, storage,
		)
	}
	aver.Always(
		len(program.Selection.Commands) > 0,
		"Command completion has a selectable command.",
	)
	if program.Selection.Mode.Value == PROGRAM_MODE_MULTICALL {
		index, _, _, found := program_select_command(
			Commands(program.Selection.Commands), Label(path.Base(path.Text(words[0]))),
		)
		if found {
			return complete_in_command(
				program.Selection.Global_Flags, help_labels,
				program.Selection.Commands[index].Arguments,
				program.Selection.Commands[index].Flags,
				Completion_Words(words), false,
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
			Commands(program.Selection.Commands), current, storage,
		))
	}
	index, _, _, found := program_select_command(
		Commands(program.Selection.Commands), Label(words[1]),
	)
	if !found {
		return storage[:0]
	}
	return complete_in_command(
		program.Selection.Global_Flags, help_labels,
		program.Selection.Commands[index].Arguments,
		program.Selection.Commands[index].Flags,
		Completion_Words(words), true, current, storage,
	)
}

// Completes the word under the cursor within a resolved command: an enum option's members
// after -label=, then flag and named-argument labels for a leading dash, then an enum
// positional's members. Anything else yields nil so the shell completes files.
func complete_in_command(
	global_flags Global_Flags, help_labels Help_Labels,
	arguments Arguments, flags Flags, words Completion_Words,
	command_selected Boolean, current Value_Text, storage Candidates,
) (candidates Candidates) {
	defer func() { Candidates_Invariants(candidates, "complete_in_command.candidates") }()
	Global_Flags_Invariants(global_flags, "complete_in_command.global_flags")
	Help_Labels_Invariants(help_labels, "complete_in_command.help_labels")
	Arguments_Invariants(arguments, "complete_in_command.arguments")
	Flags_Invariants(flags, "complete_in_command.flags")
	Completion_Words_Invariants(words, "complete_in_command.words")
	Boolean_Invariants(command_selected, "complete_in_command.command_selected")
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
			Global_Flags(global_flags), help_labels, arguments, flags,
			Option_Completion_Prefix(current), storage,
		)
	}
	position := positional_index(arguments, words, command_selected)
	if position < 0 {
		return storage[:0]
	}
	if int(position) >= len(arguments) {
		return storage[:0]
	}
	option := arguments[position]
	enum_candidates, is_enum := complete_enum_members(
		option.Enumeration, "", "", "", current, storage,
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
	aver.Always(
		location != OPTION_LOCATION_ABSENT,
		"Found completion option has a collection.",
	)
	switch location {
	case OPTION_LOCATION_ARGUMENT:
		option = arguments[index]
	case OPTION_LOCATION_COMMAND_FLAG:
		option = flags[index]
	case OPTION_LOCATION_GLOBAL_FLAG:
		option = global_flags[index]
	}
	candidates, is_enum := complete_enum_members(
		option.Enumeration, "-", label, "=",
		Value_Text(current), storage,
	)
	if !is_enum {
		return Enum_Candidates(storage[:0])
	}
	return candidates
}

// Named arguments do not consume positional slots because parser accepts mixed ordering.
func positional_index(
	arguments Arguments, words Completion_Words, command_selected Boolean,
) (position Positional_Cursor) {
	defer func() { Positional_Cursor_Invariants(position, "positional_index.position") }()
	Arguments_Invariants(arguments, "positional_index.arguments")
	Completion_Words_Invariants(words, "positional_index.words")
	Boolean_Invariants(command_selected, "positional_index.command_selected")
	arguments_start := 1
	if command_selected {
		arguments_start++
	}
	for index := arguments_start; index < len(words)-1; index++ {
		if !strings.Has_Prefix(strings.Text(words[index]), "-") {
			position++
		}
	}
	return position
}

// Finds an enum-capable option collection and index without copying union state.
func command_scope_option_result(
	location Option_Location, index slices.Found_Index, found Boolean,
) (_ Option_Location, _ slices.Found_Index, _ Boolean) {
	Option_Location_Invariants(location, "command_scope_option.location")
	slices.Found_Index_Invariants(index, "command_scope_option.index")
	Boolean_Invariants(found, "command_scope_option.found")
	return location, index, found
}

func command_scope_option(
	global_flags Global_Flags,
	arguments Arguments,
	flags Flags,
	label Completion_Option_Label,
) (_ Option_Location, _ slices.Found_Index, _ Boolean) {
	Global_Flags_Invariants(global_flags, "command_scope_option.global_flags")
	Arguments_Invariants(arguments, "command_scope_option.command_arguments")
	Flags_Invariants(flags, "command_scope_option.command_flags")
	Completion_Option_Label_Invariants(label, "command_scope_option.label")
	option_label := Option_Label(label)
	for argument_index := range arguments {
		if arguments[argument_index].Label == option_label {
			return command_scope_option_result(
				OPTION_LOCATION_ARGUMENT, slices.Found_Index(argument_index), true)
		}
	}
	for flag_index := range flags {
		if flags[flag_index].Label == option_label {
			return command_scope_option_result(
				OPTION_LOCATION_COMMAND_FLAG, slices.Found_Index(flag_index), true)
		}
	}
	for global_flag_index := range global_flags {
		if global_flags[global_flag_index].Label == option_label {
			return command_scope_option_result(
				OPTION_LOCATION_GLOBAL_FLAG,
				slices.Found_Index(global_flag_index), true,
			)
		}
	}
	return command_scope_option_result(OPTION_LOCATION_ABSENT, -1, false)
}

// Handle_Completion is the pre-parse gate a binary calls before Program_Parse: it serves
// the reserved `completion <shell>` (prints the shell script) and `__complete <words...>`
// (prints candidates, one per line) invocations, returning true when it handled one. It
// runs before parsing so it works for a single-command program, whose first token is
// otherwise a positional. IO goes to output, like Print_Help.
func Handle_Completion(
	program Program, args Process_Arguments, output Output_Reference, storage Candidates,
) (handled Boolean) {
	defer func() { Boolean_Invariants(handled, "handle_completion.handled") }()
	Output_Reference_Invariants(output, "handle_completion.output")
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
			output_write_newline(output)
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
			output_write_newline(output)
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
func Completion_Script(program Program, shell Shell, output Output_Reference) (err error) {
	Program_Invariants(program, "completion_script.program")
	Shell_Invariants(shell, "completion_script.shell")
	Output_Reference_Invariants(output, "completion_script.output")
	if block_err := completion_block(
		output, Label(program.Label), shell,
	); block_err != nil {
		return block_err
	}
	if program.Selection.Mode.Value == PROGRAM_MODE_MULTICALL {
		for index := range program.Selection.Commands {
			command := program.Selection.Commands[index]
			if !command_shown(command.Hidden, command.Deprecated) {
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
func completion_block(output Output_Reference, name Label, shell Shell) (err error) {
	Output_Reference_Invariants(output, "completion_block.output")
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
	// Required rejects an absent source and permits no nonzero default.
	Required Required
	// Allow_Empty makes an empty source different from an absent source.
	Allow_Empty Allow_Empty
	// Hidden removes the declaration from help.
	Hidden Hidden
	// Deprecated is the guidance for a source-supplied value.
	Deprecated Deprecation
	// State holds only the declaration default.
	State Environment_Default
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
	Environment_Default_Invariants(value.State, namespace)
}

// Resolved_Environment_Variable adds source-produced state to one declaration.
type Resolved_Environment_Variable struct {
	// Key is the selected declaration identity.
	Key External_Key
	// Description is the selected declaration help text.
	Description Description
	// Type selects the converted scalar arm.
	Type External_Type_State
	// Enumeration retains permitted source values.
	Enumeration External_Enumeration
	// Required records source absence policy.
	Required Required
	// Allow_Empty records empty-source policy.
	Allow_Empty Allow_Empty
	// Hidden records help visibility.
	Hidden Hidden
	// Deprecated records source guidance.
	Deprecated Deprecation
	// State holds the selected default or converted source value.
	State Environment_State
}

// Resolved_Environment_Variable_Invariants composes declaration and source state.
func Resolved_Environment_Variable_Invariants(
	value Resolved_Environment_Variable, namespace aver.Namespace,
) {
	External_Key_Invariants(value.Key, namespace)
	Description_Invariants(value.Description, namespace)
	External_Type_State_Invariants(value.Type, namespace)
	External_Enumeration_Invariants(value.Enumeration, namespace)
	Required_Invariants(value.Required, namespace)
	Allow_Empty_Invariants(value.Allow_Empty, namespace)
	Hidden_Invariants(value.Hidden, namespace)
	Deprecation_Invariants(value.Deprecated, namespace)
	Environment_State_Invariants(value.State, namespace)
}

// Resolved_Secret adds phase-owned content to immutable file metadata.
type Resolved_Secret struct {
	// Secret retains the declaration that selected the content.
	Secret
	// State holds accepted scalar content.
	State Accepted_Secret_Value
}

// Resolved_Secret_Invariants composes declaration and accepted content.
func Resolved_Secret_Invariants(value Resolved_Secret, namespace aver.Namespace) {
	Secret_Invariants(value.Secret, namespace)
	Accepted_Secret_Value_Invariants(value.State, namespace)
}

// EXTERNAL_TYPE_STATE_COUNT follows one active scalar tag.
const EXTERNAL_TYPE_STATE_COUNT = len("type") / len("type")

// External_Type_State selects one admitted external scalar representation.
type External_Type_State struct {
	// Value separates external phases from option discriminator coverage.
	Value Scalar_Option_Type
}

// External_Type_State_Invariants admits only the three external scalar types.
func External_Type_State_Invariants(
	value External_Type_State, namespace aver.Namespace,
) {
	Scalar_Option_Type_Invariants(value.Value, namespace)
}

// EXTERNAL_ENUMERATION_COUNT follows one fixed enum union cell.
const EXTERNAL_ENUMERATION_COUNT = len("enum") / len("enum")

// External_Enumeration keeps typed permitted sets outside interface storage.
type External_Enumeration struct {
	// String avoids interface inspection during text conversion.
	String String_Enumeration
	// Integers avoids interface inspection during integer conversion.
	Integers Integer_Enumeration
}

// External_Enumeration_Invariants bounds both inactive or active borrowed arms.
func External_Enumeration_Invariants(
	value External_Enumeration, namespace aver.Namespace,
) {
	String_Enumeration_Invariants(value.String, namespace)
	Integer_Enumeration_Invariants(value.Integers, namespace)
}

// Environment_Default stores one typed declaration fallback.
type Environment_Default struct {
	// String retains the text fallback.
	String Value_Text
	// Integer retains the decimal fallback.
	Integer Integer
	// Boolean retains the Boolean fallback.
	Boolean Boolean
}

// Environment_Default_Invariants excludes source-presence state.
func Environment_Default_Invariants(value Environment_Default, namespace aver.Namespace) {
	Value_Text_Invariants(value.String, namespace)
	Integer_Invariants(value.Integer, namespace)
	Boolean_Invariants(value.Boolean, namespace)
}

// Environment_State excludes byte storage that only secret resolution owns.
type Environment_State struct {
	// String retains borrowed process text.
	String Value_Text
	// Integer retains parsed decimal state.
	Integer Integer
	// Boolean retains parsed Boolean state.
	Boolean Boolean
	// Parsed separates defaults from supplied values.
	Parsed Parsed
}

// Environment_State_Invariants bounds environment-only scalar storage.
func Environment_State_Invariants(value Environment_State, namespace aver.Namespace) {
	Value_Text_Invariants(value.String, namespace)
	Integer_Invariants(value.Integer, namespace)
	Boolean_Invariants(value.Boolean, namespace)
	Parsed_Invariants(value.Parsed, namespace)
}

// Converted_Environment_State retains the tighter KEY=value source bound until
// the declaration state owns it.
type Converted_Environment_State struct {
	// String borrows the value suffix of one bounded process entry.
	String Environment_Value_Text
	// Integer retains parsed decimal state.
	Integer Integer
	// Boolean retains parsed Boolean state.
	Boolean Boolean
	// Parsed distinguishes conversion failure from accepted source state.
	Parsed Parsed
}

// Converted_Environment_State_Invariants keeps source grammar separate from
// full-width declaration defaults.
func Converted_Environment_State_Invariants(
	value Converted_Environment_State, namespace aver.Namespace,
) {
	Environment_Value_Text_Invariants(value.String, namespace)
	Integer_Invariants(value.Integer, namespace)
	Boolean_Invariants(value.Boolean, namespace)
	Parsed_Invariants(value.Parsed, namespace)
}

// Secret_Value_State excludes text storage owned only by environment resolution.
type Secret_Value_State struct {
	// Bytes retains borrowed file content.
	Bytes Secret_Value_Bytes
	// Integer retains parsed decimal state.
	Integer Integer
	// Boolean retains parsed Boolean state.
	Boolean Boolean
	// Parsed separates unresolved state from accepted content.
	Parsed Parsed
}

// Secret_Value_State_Invariants bounds secret-only scalar storage.
func Secret_Value_State_Invariants(value Secret_Value_State, namespace aver.Namespace) {
	Secret_Value_Bytes_Invariants(value.Bytes, namespace)
	Integer_Invariants(value.Integer, namespace)
	Boolean_Invariants(value.Boolean, namespace)
	Parsed_Invariants(value.Parsed, namespace)
}

// Accepted_Secret_Value omits Parsed because acceptance itself proves it.
type Accepted_Secret_Value struct {
	// Bytes retains borrowed file content for the text arm.
	Bytes Secret_Value_Bytes
	// Integer retains the decimal arm without interface storage.
	Integer Integer
	// Boolean retains the Boolean arm without interface storage.
	Boolean Boolean
}

// Accepted_Secret_Value_Invariants keeps scalar bounds after parsing succeeds.
func Accepted_Secret_Value_Invariants(
	value Accepted_Secret_Value, namespace aver.Namespace,
) {
	Secret_Value_Bytes_Invariants(value.Bytes, namespace)
	Integer_Invariants(value.Integer, namespace)
	Boolean_Invariants(value.Boolean, namespace)
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
	// Required rejects the declaration when no path supplies a value.
	Required Required
	// Allow_Empty makes an empty file different from an absent file.
	Allow_Empty Allow_Empty
	// Hidden removes the declaration from help.
	Hidden Hidden
	// Deprecated is the guidance for a source-supplied value.
	Deprecated Deprecation
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
}

// Environment_Variable_Handle is one mutable environment declaration.
type Environment_Variable_Handle *Environment_Variable

// Environment_Variable_Handle_Invariants composes present declaration storage.
func Environment_Variable_Handle_Invariants(
	value Environment_Variable_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Environment_Variable_Invariants(*value, namespace)
}

// New_Environment_Variable_Input contains one environment declaration.
type New_Environment_Variable_Input struct {
	// Key is the uppercase environment name.
	Key External_Key
	// Type preserves an omitted zero default's scalar arm.
	Type Scalar_Option_Type
	// Description is the one-line help text.
	Description Description
	// String is the text default.
	String Value_Text
	// Integer is the integer default.
	Integer Integer
	// Boolean is the Boolean default.
	Boolean Boolean
	// String_Enum restricts text without interface-backed collection storage.
	String_Enum String_Enumeration
	// Integer_Enum restricts integers without interface-backed collection storage.
	Integer_Enum Integer_Enumeration
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
func New_Environment_Variable_Input_Invariants(
	value New_Environment_Variable_Input, namespace aver.Namespace,
) {
	External_Key_Invariants(value.Key, namespace)
	Scalar_Option_Type_Invariants(value.Type, namespace)
	Description_Invariants(value.Description, namespace)
	Value_Text_Invariants(value.String, namespace)
	Integer_Invariants(value.Integer, namespace)
	Boolean_Invariants(value.Boolean, namespace)
	String_Enumeration_Invariants(value.String_Enum, namespace)
	Integer_Enumeration_Invariants(value.Integer_Enum, namespace)
	Required_Invariants(value.Required, namespace)
	Allow_Empty_Invariants(value.Allow_Empty, namespace)
	Hidden_Invariants(value.Hidden, namespace)
	Deprecation_Invariants(value.Deprecated, namespace)
}

// New_Environment_Variable rejects defaults outside its three fixed scalar arms.
func New_Environment_Variable(
	input New_Environment_Variable_Input,
) (environment Environment_Variable) {
	defer func() {
		Environment_Variable_Invariants(environment, "new_environment_variable.environment")
	}()
	New_Environment_Variable_Input_Invariants(input, "new_environment_variable.input")
	environment = Environment_Variable{
		Key:         input.Key,
		Description: input.Description,
		Type:        External_Type_State{Value: input.Type},
		Enumeration: External_Enumeration{
			String: input.String_Enum, Integers: input.Integer_Enum,
		},
		Required:    input.Required,
		Allow_Empty: input.Allow_Empty,
		Hidden:      input.Hidden,
		Deprecated:  input.Deprecated,
		State: Environment_Default{
			String: input.String, Integer: input.Integer, Boolean: input.Boolean,
		},
	}
	return environment
}

// New_Secret_Input contains one non-enum file-backed declaration.
type New_Secret_Input struct {
	// Paths are the absolute fallback paths in declaration order.
	Paths Secret_Paths
	// Description is the one-line help text.
	Description Description
	// Type selects one concrete scalar conversion.
	Type Scalar_Option_Type
	// String_Enum restricts text without interface storage.
	String_Enum String_Enumeration
	// Integer_Enum restricts integers without interface storage.
	Integer_Enum Integer_Enumeration
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
	Scalar_Option_Type_Invariants(value.Type, namespace)
	String_Enumeration_Invariants(value.String_Enum, namespace)
	Integer_Enumeration_Invariants(value.Integer_Enum, namespace)
	Required_Invariants(value.Required, namespace)
	Allow_Empty_Invariants(value.Allow_Empty, namespace)
	Hidden_Invariants(value.Hidden, namespace)
	Deprecation_Invariants(value.Deprecated, namespace)
}

// New_Secret fixes one concrete conversion without observing secret content.
func New_Secret(input New_Secret_Input) (secret Secret) {
	defer func() { Secret_Invariants(secret, "new_secret.secret") }()
	New_Secret_Input_Invariants(input, "new_secret.input")
	return Secret{
		Key: secret_key(input.Paths), Paths: input.Paths, Description: input.Description,
		Type: External_Type_State{Value: input.Type},
		Enumeration: External_Enumeration{
			String: input.String_Enum, Integers: input.Integer_Enum,
		},
		Required: input.Required, Allow_Empty: input.Allow_Empty,
		Hidden: input.Hidden, Deprecated: input.Deprecated,
	}
}

// Secret_Status_Procedure reads metadata through one injected backend.
type Secret_Status_Procedure func(
	backend nbio.IO, path Resolved_Secret_Path,
) (status nbio.File_Status, operation_err error)

// Secret_Open_Procedure submits one bounded secret open.
type Secret_Open_Procedure func(
	backend nbio.IO, completion nbio.Completion_Handle,
	path Resolved_Secret_Path, callback nbio.Callback,
)

// Secret_Read_Procedure submits one bounded secret read.
type Secret_Read_Procedure func(
	backend nbio.IO, completion nbio.Completion_Handle,
	file nbio.File, buffer Secret_Buffer, callback nbio.Callback,
)

// Secret_Close_Procedure submits descriptor retirement.
type Secret_Close_Procedure func(
	backend nbio.IO, completion nbio.Completion_Handle,
	file nbio.File, callback nbio.Callback,
)

// Secret_IO narrows injected I/O to operations secret parsing owns.
type Secret_IO struct {
	// Backend carries concrete I/O state without exposing its pointer representation.
	Backend nbio.IO
	// Status_Procedure reads path metadata.
	Status_Procedure Secret_Status_Procedure
	// Open_Procedure opens one validated path.
	Open_Procedure Secret_Open_Procedure
	// Read_Procedure reads one bounded candidate.
	Read_Procedure Secret_Read_Procedure
	// Close_Procedure retires one opened descriptor.
	Close_Procedure Secret_Close_Procedure
}

// Secret_IO_Invariants requires either zero state or one complete procedure set.
func Secret_IO_Invariants(value Secret_IO, namespace aver.Namespace) {
	nbio.IO_Invariants(value.Backend, namespace)
	present := value.Status_Procedure != nil
	aver.Always(
		(value.Open_Procedure != nil) == present,
		"Secret IO status and open procedures have equal presence.",
	)
	aver.Always(
		(value.Read_Procedure != nil) == present,
		"Secret IO status and read procedures have equal presence.",
	)
	aver.Always(
		(value.Close_Procedure != nil) == present,
		"Secret IO status and close procedures have equal presence.",
	)
}

// Secret_IO_Of binds repository I/O without copying backend state into closures.
func Secret_IO_Of(backend nbio.IO) (loop Secret_IO) {
	defer func() { Secret_IO_Invariants(loop, "secret_io_of.loop") }()
	nbio.IO_Invariants(backend, "secret_io_of.backend")
	return Secret_IO{
		Backend: backend, Status_Procedure: secret_status_nbio,
		Open_Procedure: secret_open_nbio, Read_Procedure: secret_read_nbio,
		Close_Procedure: secret_close_nbio,
	}
}

func secret_status_nbio(
	backend nbio.IO, path Resolved_Secret_Path,
) (status nbio.File_Status, operation_err error) {
	nbio.IO_Invariants(backend, "secret_status_nbio.backend")
	Resolved_Secret_Path_Invariants(path, "secret_status_nbio.path")
	return nbio.Storage_Status(backend.Storage, string(path))
}

func secret_open_nbio(
	backend nbio.IO, completion nbio.Completion_Handle,
	path Resolved_Secret_Path, callback nbio.Callback,
) {
	nbio.IO_Invariants(backend, "secret_open_nbio.backend")
	Resolved_Secret_Path_Invariants(path, "secret_open_nbio.path")
	nbio.Storage_Open_At(
		backend.Storage, completion, nbio.DIRECTORY_CURRENT, string(path),
		nbio.Open_At_Options{
			Access: nbio.OPEN_READ_ONLY, Flags: nbio.OPEN_AT_NO_FOLLOW,
		},
		callback,
	)
}

func secret_read_nbio(
	backend nbio.IO, completion nbio.Completion_Handle,
	file nbio.File, buffer Secret_Buffer, callback nbio.Callback,
) {
	nbio.IO_Invariants(backend, "secret_read_nbio.backend")
	Secret_Buffer_Invariants(buffer, "secret_read_nbio.buffer")
	nbio.Storage_Read(backend.Storage, completion, file, buffer, 0, time.DAY, callback)
}

func secret_close_nbio(
	backend nbio.IO, completion nbio.Completion_Handle,
	file nbio.File, callback nbio.Callback,
) {
	nbio.IO_Invariants(backend, "secret_close_nbio.backend")
	nbio.IO_Close(backend, completion, file, callback)
}

// Program_Parse_Input contains deterministic process inputs and the injected I/O loop.
type Program_Parse_Input struct {
	// Arguments are the complete process argument list.
	Arguments Process_Arguments
	// Environment is the complete raw KEY=value process environment snapshot.
	Environment Process_Environment
	// Loop submits secret file operations. Programs without secrets can leave it unset.
	Loop Secret_IO
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
	// Failure_Storage receives bounded rendered diagnostics.
	Failure_Storage Parse_Failure_Storage
	// Deprecation_Warnings receives command, flag, environment, and secret warnings.
	Deprecation_Warnings Deprecation_Warnings
	// Environment_Values receives resolved declaration copies.
	Environment_Values Environment_Variables
	// Environment_Warnings receives emitted environment deprecations.
	Environment_Warnings Environment_Warnings
	// Environment_Sources classifies declared raw entries without hidden map storage.
	Environment_Sources Environment_Sources
	// Secret_Values receives resolved declaration copies.
	Secret_Values Resolved_Secrets
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
}

// Program_Parse_Input_Invariants composes deterministic input and caller storage bounds.
func Program_Parse_Input_Invariants(
	value Program_Parse_Input, namespace aver.Namespace,
) {
	Secret_IO_Invariants(value.Loop, namespace)
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
	Parse_Failure_Storage_Invariants(value.Failure_Storage, namespace)
	Deprecation_Warnings_Invariants(value.Deprecation_Warnings, namespace)
	Environment_Variables_Invariants(value.Environment_Values, namespace)
	Environment_Warnings_Invariants(value.Environment_Warnings, namespace)
	Environment_Sources_Invariants(value.Environment_Sources, namespace)
	Resolved_Secrets_Invariants(value.Secret_Values, namespace)
	Secret_Failures_Invariants(value.Secret_Errors, namespace)
	Secret_Warnings_Invariants(value.Secret_Warnings, namespace)
	Secret_Parsers_Invariants(value.Secret_Parsers, namespace)
	Secret_Buffers_Invariants(value.Secret_Buffers, namespace)
	Secret_Path_Failures_Invariants(value.Secret_Path_Failures, namespace)
}

// Parser_Initialize_Input excludes storage owned only by later external phases.
type Parser_Initialize_Input struct {
	// Arguments freezes process tokens before mutable parser state is published.
	Arguments Process_Arguments
	// Loop moves into publication before secret work starts.
	Loop Secret_IO
	// Command_Arguments receives active argument copies.
	Command_Arguments Command_Argument_Storage
	// Command_Flags receives active flag copies.
	Command_Flags Command_Flag_Storage
	// Global_Flags receives mutable global flag copies.
	Global_Flags Global_Flag_Storage
	// Filled records active scalar assignment.
	Filled Filled_Options
	// Positionals retains bare tokens until assignment.
	Positionals Positionals
	// Slice_Named retains named variadic contributions.
	Slice_Named Named_Variadic_Tokens
	// String_Values receives parsed string variadic values.
	String_Values String_Values
	// Integer_Values receives parsed integer variadic values.
	Integer_Values Integer_Values
	// Failure_Storage receives one bounded CLI diagnostic.
	Failure_Storage Parse_Failure_Storage
	// Deprecation_Warnings receives active declaration warnings.
	Deprecation_Warnings Deprecation_Warnings
}

// Parser_Initialize_Input_Invariants composes storage touched during initialization.
func Parser_Initialize_Input_Invariants(
	value Parser_Initialize_Input, namespace aver.Namespace,
) {
	Process_Arguments_Invariants(value.Arguments, namespace)
	Secret_IO_Invariants(value.Loop, namespace)
	Command_Argument_Storage_Invariants(value.Command_Arguments, namespace)
	Command_Flag_Storage_Invariants(value.Command_Flags, namespace)
	Global_Flag_Storage_Invariants(value.Global_Flags, namespace)
	Filled_Options_Invariants(value.Filled, namespace)
	Positionals_Invariants(value.Positionals, namespace)
	Named_Variadic_Tokens_Invariants(value.Slice_Named, namespace)
	String_Values_Invariants(value.String_Values, namespace)
	Integer_Values_Invariants(value.Integer_Values, namespace)
	Parse_Failure_Storage_Invariants(value.Failure_Storage, namespace)
	Deprecation_Warnings_Invariants(value.Deprecation_Warnings, namespace)
}

// Parse_Error_Storage borrows terminal diagnostic capacity.
type Parse_Error_Storage []byte

// Parse_Error_Storage_Invariants admits absent or initialized parser storage.
func Parse_Error_Storage_Invariants(value Parse_Error_Storage, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Int(len(value), strings.TEXT_SIZE_MINIMUM, FAILURE_SIZE_MAXIMUM).
		Ensure()
}

// Parse_Error_Size_Value selects initialized terminal diagnostic bytes.
type Parse_Error_Size_Value int

// Parse_Error_Size_Value_Invariants bounds terminal diagnostic bytes.
func Parse_Error_Size_Value_Invariants(
	value Parse_Error_Size_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), strings.TEXT_SIZE_MINIMUM, FAILURE_SIZE_MAXIMUM).
		Ensure()
}

// Parse_Error_Size owns terminal diagnostic cursor value.
type Parse_Error_Size struct {
	// Value is initialized terminal diagnostic byte count.
	Value Parse_Error_Size_Value
}

// Parse_Error_Size_Invariants composes terminal diagnostic cursor.
func Parse_Error_Size_Invariants(value Parse_Error_Size, namespace aver.Namespace) {
	Parse_Error_Size_Value_Invariants(value.Value, namespace)
}

// Parse_Error keeps terminal diagnostics in caller-owned storage.
type Parse_Error struct {
	// Storage borrows parser diagnostic storage.
	Storage Parse_Error_Storage
	// Size selects initialized diagnostic bytes.
	Size Parse_Error_Size
}

// Parse_Error_Invariants composes one bounded diagnostic view.
func Parse_Error_Invariants(value Parse_Error, namespace aver.Namespace) {
	Parse_Error_Storage_Invariants(value.Storage, namespace)
	Parse_Error_Size_Invariants(value.Size, namespace)
	aver.Always(
		int(value.Size.Value) <= len(value.Storage),
		"Parse error size fits borrowed storage.",
	)
}

// Parse_Error_Content is initialized terminal diagnostic bytes.
type Parse_Error_Content []byte

// Parse_Error_Content_Invariants bounds published diagnostic bytes.
func Parse_Error_Content_Invariants(
	value Parse_Error_Content, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, FAILURE_SIZE_MAXIMUM).
		Ensure()
}

// Parse_Error_Present reports initialized diagnostic bytes.
func Parse_Error_Present(value Parse_Error) (present Boolean) {
	defer func() { Boolean_Invariants(present, "parse_error_present.present") }()
	Parse_Error_Invariants(value, "parse_error_present.value")
	return value.Size.Value != 0
}

// Parse_Error_Bytes borrows initialized diagnostic bytes.
func Parse_Error_Bytes(value Parse_Error) (content Parse_Error_Content) {
	defer func() { Parse_Error_Content_Invariants(content, "parse_error_bytes.content") }()
	Parse_Error_Invariants(value, "parse_error_bytes.value")
	return Parse_Error_Content(value.Storage[:value.Size.Value:value.Size.Value])
}

// Parse_Error_Equals compares borrowed bytes without string conversion.
func Parse_Error_Equals(value Parse_Error, expected Failure_Fragment) (equal Boolean) {
	defer func() { Boolean_Invariants(equal, "parse_error_equals.equal") }()
	Parse_Error_Invariants(value, "parse_error_equals.value")
	Failure_Fragment_Invariants(expected, "parse_error_equals.expected")
	content := Parse_Error_Bytes(value)
	if len(content) != len(expected) {
		return false
	}
	for index := range content {
		if content[index] != expected[index] {
			return false
		}
	}
	return true
}

// Parse_Result is the terminal parser result.
type Parse_Result struct {
	// Command contains all CLI and external values that resolved before Error.
	Command Resolved_Command
	// Error joins all external failures after a successful CLI parse.
	Error Parse_Error
}

// Parse_Result_Invariants composes the terminal command state.
func Parse_Result_Invariants(value Parse_Result, namespace aver.Namespace) {
	Resolved_Command_Invariants(value.Command, namespace)
	Parse_Error_Invariants(value.Error, namespace)
}

// Workspace_Arguments is phase-owned argument storage.
type Workspace_Arguments []Resolved_Option

// Workspace_Arguments_Invariants bounds argument storage structurally.
func Workspace_Arguments_Invariants(
	value Workspace_Arguments, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Workspace_Flags is phase-owned command flag storage.
type Workspace_Flags []Resolved_Option

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
}

// PARSER_COMPLETION_COUNT follows the one publication bit.
const PARSER_COMPLETION_COUNT = len("complete") / len("complete")

// Parser_Completion stores terminal state without phase-wide Boolean branch contracts.
type Parser_Completion struct {
	// Value separates lifecycle storage from Boolean branch coverage.
	Value Parser_Complete
}

// Parser_Completion_Invariants fixes the one intrinsically bounded Boolean cell.
func Parser_Completion_Invariants(value Parser_Completion, namespace aver.Namespace) {
	Parser_Complete_Invariants(value.Value, namespace)
}

// Publication owns terminal result and external-resolution state.
type Publication struct {
	// Result stays inside caller-owned parser storage.
	Result Parse_Result
	// Completion distinguishes pending work from terminal publication.
	Completion Parser_Completion
	// Loop is the injected submission surface.
	Loop Secret_IO
	// Secret_Errors keep secret declaration order.
	Secret_Errors Secret_Failures
	// Environment_Warnings keep environment declaration order.
	Environment_Warnings Environment_Warnings
	// Secret_Warnings keep secret declaration order.
	Secret_Warnings Secret_Warnings
	// Secret_Count is the number of declarations that did not retire.
	Secret_Count Declaration_Count
	// Secret_Parsers lets Parser_Done advance operations retired by caller-driven I/O.
	Secret_Parsers Secret_Parsers
	// Failure owns ordered external diagnostic text.
	Failure Failure
}

// Publication_Invariants composes ordered external and terminal state.
func Publication_Invariants(value Publication, namespace aver.Namespace) {
	Secret_IO_Invariants(value.Loop, namespace)
	Parse_Result_Invariants(value.Result, namespace)
	Parser_Completion_Invariants(value.Completion, namespace)
	Secret_Failures_Invariants(value.Secret_Errors, namespace)
	Environment_Warnings_Invariants(value.Environment_Warnings, namespace)
	Secret_Warnings_Invariants(value.Secret_Warnings, namespace)
	Declaration_Count_Invariants(value.Secret_Count, namespace)
	Secret_Parsers_Invariants(value.Secret_Parsers, namespace)
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

// Parser_Handle is one live parser state.
type Parser_Handle *Parser

// Parser_Handle_Invariants composes present parser storage.
func Parser_Handle_Invariants(value Parser_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Parser_Invariants(*value, namespace)
}

// Parser_Done reports pending work or returns the terminal caller-owned result.
func parser_done_result(
	result Parse_Result, complete Parser_Complete,
) (_ Parse_Result, _ Parser_Complete) {
	Parse_Result_Invariants(result, "parser_done.result")
	Parser_Complete_Invariants(complete, "parser_done.complete")
	return result, complete
}

// Parser_Done reports pending work or returns the terminal caller-owned result.
func Parser_Done(parser Parser_Handle) (_ Parse_Result, _ Parser_Complete) {
	Parser_Handle_Invariants(parser, "parser_done.parser")
	if !parser.Completion.Value {
		if len(parser.Secret_Parsers) == 0 {
			parser.Secret_Count = 0
		} else {
			parser.Secret_Count = secret_advance_all(
				Active_Secrets(parser.Result.Command.Secrets),
				Active_Secret_Parsers(parser.Secret_Parsers), parser.Loop,
				Active_Secret_Failures(parser.Secret_Errors),
				Active_Secret_Warnings(parser.Secret_Warnings),
			)
		}
		if parser.Secret_Count == 0 {
			failure_output := Failure_Writer{Size: &parser.Failure.Size,
				Storage: Parse_Failure_Storage(parser.Failure.Storage)}
			command := &parser.Result.Command
			for index, failure := range parser.Secret_Errors {
				if !failure.Present {
					continue
				}
				if parser.Failure.Size.Value != 0 {
					failure_write(failure_output, Failure_Fragment("\n"))
				}
				failure_write(failure_output, Failure_Fragment("secret "))
				failure_write(failure_output,
					Failure_Fragment(string(command.Secrets[index].Key)))
				failure_write(failure_output, Failure_Fragment(": "))
				for path_index, path_failure := range failure.Paths {
					if path_index != 0 {
						failure_write(
							failure_output, Failure_Fragment("\n"),
						)
					}
					failure_write_path(failure_output, Failure_Secret_Key{
						Value: command.Secrets[index].Key,
					}, path_failure)
				}
			}
			warnings := Deprecation_Warnings(command.Deprecation_Warnings)
			for _, warning := range parser.Environment_Warnings {
				warnings = warning_store(
					Writable_Deprecation_Warnings(warnings), warning)
			}
			for _, warning := range parser.Secret_Warnings {
				if warning.Guidance.Value != "" {
					warnings = warning_store(
						Writable_Deprecation_Warnings(warnings), warning)
				}
			}
			command.Deprecation_Warnings = Resolved_Deprecation_Warnings(warnings)
			if parser.Failure.Size.Value != 0 {
				parser.Result.Error = Parse_Error{
					Storage: Parse_Error_Storage(parser.Failure.Storage),
					Size: Parse_Error_Size{Value: Parse_Error_Size_Value(
						parser.Failure.Size.Value)},
				}
			} else {
				parser.Result.Error = Parse_Error{}
			}
			parser.Completion.Value = true
		}
	}
	return parser_done_result(parser.Result, parser.Completion.Value)
}

// Get_Environment returns a declared environment variable and panics for an unknown key.
func Get_Environment(
	environment Resolved_Environment,
	key External_Key,
) (variable Resolved_Environment_Variable) {
	defer func() {
		Resolved_Environment_Variable_Invariants(variable, "get_environment.variable")
	}()
	Resolved_Environment_Invariants(environment, "get_environment.environment")
	External_Key_Invariants(key, "get_environment.key")
	found := false
	for _, candidate := range environment {
		if External_Key(candidate.Key) == key {
			variable = candidate
			found = true
			break
		}
	}
	aver.Always(found, "Environment variable is declared.")
	return variable
}

// Environment_String returns one named string without publishing tagged declaration state.
func Environment_String(
	environment Resolved_Environment, key External_Key,
) (value Value_Text) {
	defer func() {
		Value_Text_Invariants(value, "environment_string.value")
	}()
	Resolved_Environment_Invariants(environment, "environment_string.environment")
	External_Key_Invariants(key, "environment_string.key")
	variable := Get_Environment(environment, key)
	aver.Always(
		Option_Type(variable.Type.Value) == OPTION_TYPE_STRING,
		"Environment_String receives text scalar storage.",
	)
	return variable.State.String
}

// Environment_Integer returns one named integer without publishing tagged declaration state.
func Environment_Integer(
	environment Resolved_Environment, key External_Key,
) (value Integer) {
	defer func() { Integer_Invariants(value, "environment_integer.value") }()
	Resolved_Environment_Invariants(environment, "environment_integer.environment")
	External_Key_Invariants(key, "environment_integer.key")
	variable := Get_Environment(environment, key)
	aver.Always(
		Option_Type(variable.Type.Value) == OPTION_TYPE_INTEGER,
		"Environment_Integer receives integer scalar storage.",
	)
	return variable.State.Integer
}

// Environment_Boolean returns one named Boolean without publishing tagged declaration state.
func Environment_Boolean(
	environment Resolved_Environment, key External_Key,
) (value Boolean) {
	defer func() { Boolean_Invariants(value, "environment_boolean.value") }()
	Resolved_Environment_Invariants(environment, "environment_boolean.environment")
	External_Key_Invariants(key, "environment_boolean.key")
	variable := Get_Environment(environment, key)
	aver.Always(
		Option_Type(variable.Type.Value) == OPTION_TYPE_BOOLEAN,
		"Environment_Boolean receives Boolean scalar storage.",
	)
	return variable.State.Boolean
}

// Get_Secret returns a resolved secret and panics for an unknown key.
func Get_Secret(secrets Resolved_Secrets, key Secret_Key) (secret Resolved_Secret) {
	defer func() { Resolved_Secret_Invariants(secret, "get_secret.secret") }()
	Resolved_Secrets_Invariants(secrets, "get_secret.secrets")
	Secret_Key_Invariants(key, "get_secret.key")
	found := false
	for _, candidate := range secrets {
		if candidate.Key == key {
			secret = candidate
			found = true
			break
		}
	}
	aver.Always(found, "Secret is declared.")
	return secret
}

// Secret_String returns one named caller-owned source without tagged declaration state.
func Secret_String(
	secrets Resolved_Secrets, key Secret_Key,
) (value Secret_Value_Bytes) {
	defer func() { Secret_Value_Bytes_Invariants(value, "secret_string.value") }()
	Resolved_Secrets_Invariants(secrets, "secret_string.secrets")
	Secret_Key_Invariants(key, "secret_string.key")
	secret := Get_Secret(secrets, key)
	aver.Always(
		Option_Type(secret.Type.Value) == OPTION_TYPE_STRING,
		"Secret_String receives text scalar storage.",
	)
	return secret.State.Bytes
}

// Secret_Integer returns one named integer without publishing tagged declaration state.
func Secret_Integer(secrets Resolved_Secrets, key Secret_Key) (value Integer) {
	defer func() { Integer_Invariants(value, "secret_integer.value") }()
	Resolved_Secrets_Invariants(secrets, "secret_integer.secrets")
	Secret_Key_Invariants(key, "secret_integer.key")
	secret := Get_Secret(secrets, key)
	aver.Always(
		Option_Type(secret.Type.Value) == OPTION_TYPE_INTEGER,
		"Secret_Integer receives integer scalar storage.",
	)
	return secret.State.Integer
}

// Secret_Boolean returns one named Boolean without publishing tagged declaration state.
func Secret_Boolean(secrets Resolved_Secrets, key Secret_Key) (value Boolean) {
	defer func() { Boolean_Invariants(value, "secret_boolean.value") }()
	Resolved_Secrets_Invariants(secrets, "secret_boolean.secrets")
	Secret_Key_Invariants(key, "secret_boolean.key")
	secret := Get_Secret(secrets, key)
	aver.Always(
		Option_Type(secret.Type.Value) == OPTION_TYPE_BOOLEAN,
		"Secret_Boolean receives Boolean scalar storage.",
	)
	return secret.State.Boolean
}

// Program_Parse resolves CLI inputs before it reads environment variables or secret files.
func Program_Parse(program Program, parser Parser_Handle, input Program_Parse_Input) {
	Program_Invariants(program, "program_parse.program")
	Parser_Handle_Invariants(parser, "program_parse.parser")
	Program_Parse_Input_Invariants(input, "program_parse.input")
	if parser_parse_cli(program.Selection, parser, Parser_Initialize_Input{
		Arguments:            input.Arguments,
		Loop:                 input.Loop,
		Command_Arguments:    input.Command_Arguments,
		Command_Flags:        input.Command_Flags,
		Global_Flags:         input.Global_Flags,
		Filled:               input.Filled,
		Positionals:          input.Positionals,
		Slice_Named:          input.Slice_Named,
		String_Values:        input.String_Values,
		Integer_Values:       input.Integer_Values,
		Failure_Storage:      input.Failure_Storage,
		Deprecation_Warnings: input.Deprecation_Warnings,
	}) != nil {
		return
	}
	if len(program.Environment_Variables)+len(program.Secrets) == 0 {
		parser.Completion.Value = true
		return
	}
	parser.Result.Command.Environment = Resolved_Environment(
		copy_environment(program.Environment_Variables, input.Environment_Values),
	)
	parser.Result.Command.Secrets = Resolved_Secrets(
		copy_secrets(program.Secrets, input.Secret_Values),
	)
	parser.Failure.Size.Value = 0
	parser.Environment_Warnings = environment_parse(
		Environment_Variables(parser.Result.Command.Environment),
		input.Environment, input.Environment_Warnings, input.Environment_Sources,
		Failure_Writer{
			Storage: input.Failure_Storage, Size: &parser.Failure.Size,
		},
	)
	if len(parser.Result.Command.Secrets) == 0 {
		return
	}
	aver.Always(
		parser.Loop.Open_Procedure != nil,
		"Program_Parse secret declarations have Loop storage.",
	)
	secret_count := len(parser.Result.Command.Secrets)
	secret_results_initialize(
		Active_Secret_Failures(input.Secret_Errors),
		Active_Secret_Warnings(input.Secret_Warnings), Active_Secret_Count(secret_count),
	)
	parser.Secret_Errors = input.Secret_Errors[:secret_count]
	parser.Secret_Warnings = input.Secret_Warnings[:secret_count]
	parser.Secret_Parsers = input.Secret_Parsers[:secret_count]
	parser_initialize_secrets(
		Active_Secrets(parser.Result.Command.Secrets),
		Active_Secret_Parsers(input.Secret_Parsers),
		Active_Secret_Buffers(input.Secret_Buffers),
		Active_Secret_Path_Failures(input.Secret_Path_Failures),
	)
	parser.Secret_Count = secret_advance_all(
		Active_Secrets(parser.Result.Command.Secrets),
		Active_Secret_Parsers(input.Secret_Parsers), parser.Loop,
		Active_Secret_Failures(parser.Secret_Errors),
		Active_Secret_Warnings(parser.Secret_Warnings),
	)
}

// CLI parsing completes before any external source operation can start.
func parser_parse_cli(
	declaration Program_Selection, parser Parser_Handle, input Parser_Initialize_Input,
) (err error) {
	Program_Selection_Invariants(declaration, "parser_parse_cli.declaration")
	Parser_Handle_Invariants(parser, "parser_parse_cli.parser")
	Parser_Initialize_Input_Invariants(input, "parser_parse_cli.input")
	original_global_flags := declaration.Global_Flags
	aver.Always(
		len(input.Global_Flags) >= len(original_global_flags),
		"Program_Parse Global_Flags storage fits declarations.",
	)
	global_flags := Resolved_Global_Flags(copy_options(
		Options(original_global_flags), Resolved_Options(input.Global_Flags),
	))
	*parser = Parser{
		Publication: Publication{
			Loop: input.Loop,
			Failure: Failure{
				Storage: Failure_Storage(input.Failure_Storage),
			},
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
	parsed, err := program_parse_arguments(
		declaration, global_flags, input.Arguments, parser.Workspace,
		input.Failure_Storage,
	)
	parser.Result.Command = Resolved_Command{
		Parsed_Command: parsed,
	}
	if err != nil {
		failure_size := 0
		for failure_size < len(input.Failure_Storage) {
			if input.Failure_Storage[failure_size] == 0 {
				break
			}
			failure_size++
		}
		parser.Failure.Size.Value = Failure_Size_Value(failure_size)
		parser.Result.Error = Parse_Error{
			Storage: Parse_Error_Storage(parser.Failure.Storage),
			Size: Parse_Error_Size{
				Value: Parse_Error_Size_Value(parser.Failure.Size.Value),
			},
		}
		parser.Completion.Value = true
	}
	return err
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
	aver.Always(
		len(storage) >= len(input),
		"Program_Parse Environment_Values storage fits declarations.",
	)
	output = storage[:len(input)]
	for index := range output {
		declaration := input[index]
		output[index] = Resolved_Environment_Variable{
			Key: declaration.Key, Description: declaration.Description,
			Type: declaration.Type, Enumeration: declaration.Enumeration,
			Required: declaration.Required, Allow_Empty: declaration.Allow_Empty,
			Hidden: declaration.Hidden, Deprecated: declaration.Deprecated,
			State: Environment_State{
				String:  declaration.State.String,
				Integer: declaration.State.Integer,
				Boolean: declaration.State.Boolean,
			},
		}
	}
	return output
}

// Each parse owns its secret values and internal presence state.
func copy_secrets(
	input Secret_Declarations, storage Resolved_Secrets,
) (output Resolved_Secrets) {
	defer func() { Resolved_Secrets_Invariants(output, "copy_secrets.output") }()
	Secret_Declarations_Invariants(input, "copy_secrets.input")
	Resolved_Secrets_Invariants(storage, "copy_secrets.storage")
	aver.Always(
		len(storage) >= len(input),
		"Program_Parse Secret_Values storage fits declarations.",
	)
	output = storage[:len(input)]
	for index := range output {
		output[index] = Resolved_Secret{Secret: input[index]}
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
	return Secret_Key(filepath.Base(filepath.Text(paths[0])))
}

// Construction rejects source ambiguity and declarations that no loader can resolve safely.
func external_validate(environment Environment_Declarations, secrets Secrets) {
	Environment_Declarations_Invariants(environment, "external_validate.environment")
	Secrets_Invariants(secrets, "external_validate.secrets")
	seen := External_Key_Set{}
	for index := range environment {
		variable := environment[index]
		external_key_validate(variable.Key)
		aver.Always(!seen[variable.Key], "Environment external key is unique.")
		seen[variable.Key] = true
		value := External_Value{
			Type: variable.Type, Enumeration: variable.Enumeration,
			String: variable.State.String, Integer: variable.State.Integer,
		}
		external_value_validate(value, Boolean(!variable.Required))
		if variable.Required {
			nonzero := variable.State.String != ""
			if Option_Type(variable.Type.Value) == OPTION_TYPE_INTEGER {
				nonzero = variable.State.Integer != 0
			}
			if Option_Type(variable.Type.Value) == OPTION_TYPE_BOOLEAN {
				nonzero = bool(variable.State.Boolean)
			}
			aver.Always(
				!nonzero,
				"Required environment variable has zero default.",
			)
		}
	}
	for index := range secrets {
		secret := &secrets[index]
		secret_validate_paths(secret.Key, secret.Paths)
		key := External_Key(secret.Key)
		aver.Always(!seen[key], "Secret external key is unique.")
		seen[key] = true
		external_value_validate(External_Value{
			Type: secret.Type, Enumeration: secret.Enumeration,
		}, false)
	}
}

// An uppercase key makes the environment namespace and filename namespace identical.
func external_key_validate(key External_Key) {
	External_Key_Invariants(key, "external_key_validate.key")
	valid := len(key) > 0
	for index, character := range key {
		uppercase := character >= 'A' && character <= 'Z'
		if index == 0 {
			valid = uppercase
		} else {
			digit := character >= '0' && character <= '9'
			valid = uppercase || digit || character == '_'
		}
		if !valid {
			break
		}
	}
	aver.Always(valid, "External key is an uppercase environment name.")
	aver.Always(key != "HELP", "External key is not reserved.")
}

// External_Value carries the validated union without tying it to one source kind.
type External_Value struct {
	// Type selects the active scalar storage arm.
	Type External_Type_State
	// Enumeration retains the optional permitted set.
	Enumeration External_Enumeration
	// String is the text default checked for enum membership.
	String Value_Text
	// Integer is the integer default checked for enum membership.
	Integer Integer
}

// External_Value_Invariants composes the typed external union.
func External_Value_Invariants(value External_Value, namespace aver.Namespace) {
	External_Type_State_Invariants(value.Type, namespace)
	External_Enumeration_Invariants(value.Enumeration, namespace)
	Value_Text_Invariants(value.String, namespace)
	Integer_Invariants(value.Integer, namespace)
}

// External enum validation keeps environment identity independent of dashed option bounds.
func external_value_validate(value External_Value, default_is_value Boolean) {
	External_Value_Invariants(value, "external_value_validate.value")
	Boolean_Invariants(default_is_value, "external_value_validate.default_is_value")
	if value.Enumeration.String == nil {
		if value.Enumeration.Integers == nil {
			return
		}
	}
	if value.Enumeration.String != nil {
		aver.Always(
			value.Enumeration.Integers == nil,
			"Text external enum is sole enum representation.",
		)
		aver.Always(
			Option_Type(value.Type.Value) == OPTION_TYPE_STRING,
			"Text external enum has text scalar storage.",
		)
		aver.Always(
			len(value.Enumeration.String) > 0,
			"Text external enum has members.",
		)
		if default_is_value {
			aver.Always(
				slices.Contains(value.Enumeration.String, string(value.String)),
				"Text external default belongs to its enum.",
			)
		}
		return
	}
	aver.Always(
		Option_Type(value.Type.Value) == OPTION_TYPE_INTEGER,
		"Integer external enum has integer scalar storage.",
	)
	aver.Always(
		len(value.Enumeration.Integers) > 0,
		"Integer external enum has members.",
	)
	if default_is_value {
		aver.Always(
			slices.Contains(value.Enumeration.Integers, int(value.Integer)),
			"Integer external default belongs to its enum.",
		)
	}
}

// Path validation makes the filename a stable key across all fallbacks.
func secret_validate_paths(key Secret_Key, paths Secret_Paths) {
	Secret_Key_Invariants(key, "secret_validate_paths.key")
	Secret_Paths_Invariants(paths, "secret_validate_paths.paths")
	aver.Always(len(paths) > 0, "Secret has at least one path.")
	for _, path := range paths {
		aver.Always(
			bool(filepath.Is_Absolute(filepath.Text(path))),
			"Secret path is absolute.",
		)
		path_key := External_Key(filepath.Base(filepath.Text(path)))
		external_key_validate(path_key)
		aver.Always(
			Secret_Key(path_key) == key,
			"Secret paths share one base filename.",
		)
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
	environment Environment_Variables, raw Process_Environment,
	warning_storage Environment_Warnings,
	source_storage Environment_Sources,
	failure_output Failure_Writer,
) (warnings Environment_Warnings) {
	defer func() {
		Environment_Warnings_Invariants(warnings, "environment_parse.warnings")
	}()
	Environment_Variables_Invariants(environment, "environment_parse.environment")
	Process_Environment_Invariants(raw, "environment_parse.raw")
	Environment_Warnings_Invariants(
		warning_storage, "environment_parse.warning_storage")
	Environment_Sources_Invariants(source_storage, "environment_parse.source_storage")
	Failure_Writer_Invariants(failure_output, "environment_parse.failure_output")
	warnings = warning_storage[:0]
	sources := environment_sources(environment, raw, source_storage)
	for index := range environment {
		variable := &environment[index]
		source := sources[variable.Key]
		present, failure := environment_source_failure(
			variable.Allow_Empty, variable.Required, source)
		if present {
			state, conversion_failure := external_convert_state(
				variable.Type, variable.Enumeration, source.Value)
			variable.State = Environment_State{
				String: Value_Text(state.String), Integer: state.Integer,
				Boolean: state.Boolean, Parsed: state.Parsed,
			}
			failure = conversion_failure
		}
		if failure != nil {
			switch failure {
			case external_string_enum_failure, external_integer_enum_failure:
				environment_enum_failure_write(
					failure_output, variable.Key, variable.Enumeration,
					source.Value, failure)
			default:
				environment_source_failure_write(
					failure_output, variable.Key, source, failure)
			}
			continue
		}
		if environment_warning_needed(
			variable.Deprecated, variable.Allow_Empty, source.Value,
			Boolean(source.Occurrences == 1),
		) {
			aver.Always(
				len(warnings) < len(warning_storage),
				"Program_Parse Environment_Warnings storage has room.",
			)
			warning_count := len(warnings)
			warnings = warning_storage[:warning_count+1]
			warnings[warning_count] = Warning{
				Kind: Warning_Kind{Value: WARNING_KIND_ENVIRONMENT},
				Name: Warning_Name{
					Value: Value_Text(variable.Key),
				},
				Guidance: Warning_Guidance{Value: variable.Deprecated},
			}
		}
	}
	return warnings
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
	allow_empty Allow_Empty, required Required, source Environment_Source,
) (present Boolean, err error) {
	defer func() {
		Boolean_Invariants(present, "environment_source_failure.present")
	}()
	Allow_Empty_Invariants(allow_empty, "environment_source_failure.allow_empty")
	Required_Invariants(required, "environment_source_failure.required")
	Environment_Source_Invariants(source, "environment_source_failure.source")
	if source.Malformed {
		return false, environment_malformed_failure
	}
	if source.Occurrences > 1 {
		return false, environment_duplicate_failure
	}
	absent := source.Occurrences == 0
	if source.Value == "" {
		if !allow_empty {
			absent = true
		}
	}
	if absent {
		if required {
			return false, environment_missing_failure
		}
		return false, nil
	}
	return true, nil
}

// Converts external text directly into fixed scalar storage.
func external_convert_state(
	type_state External_Type_State,
	enumeration External_Enumeration,
	raw Environment_Value_Text,
) (state Converted_Environment_State, err error) {
	defer func() {
		Converted_Environment_State_Invariants(state, "external_convert_state.state")
	}()
	External_Type_State_Invariants(type_state, "external_convert_state.type_state")
	External_Enumeration_Invariants(enumeration, "external_convert_state.enumeration")
	Environment_Value_Text_Invariants(raw, "external_convert_state.raw")
	converted := true
	switch Option_Type(type_state.Value) {
	case OPTION_TYPE_STRING:
		if enumeration.String != nil {
			if !slices.Contains(enumeration.String, string(raw)) {
				return state, external_string_enum_failure
			}
		}
		state.String = raw
	case OPTION_TYPE_INTEGER:
		if len(raw) > strings.TEXT_SIZE_MAXIMUM {
			return state, external_integer_parse_failure
		}
		number, parse_err := strconv.Parse_Decimal(strconv.Text(raw))
		if parse_err != nil {
			return state, external_integer_parse_failure
		}
		if enumeration.Integers != nil {
			if !slices.Contains(enumeration.Integers, int(number)) {
				return state, external_integer_enum_failure
			}
		}
		state.Integer = Integer(number)
	case OPTION_TYPE_BOOLEAN:
		if len(raw) > strings.TEXT_SIZE_MAXIMUM {
			return state, external_boolean_parse_failure
		}
		boolean, parse_err := strconv.Parse_Boolean(strconv.Text(raw))
		if parse_err != nil {
			return state, external_boolean_parse_failure
		}
		state.Boolean = Boolean(boolean)
	default:
		converted = false
	}
	aver.Always(converted, "Environment conversion receives validated scalar type.")
	state.Parsed = true
	return state, nil
}

// Source failure rendering stays after validation so classification owns no output state.
func environment_source_failure_write(
	failure Failure_Writer, key External_Key, source Environment_Source, kind error,
) {
	Failure_Writer_Invariants(failure, "environment_source_failure_write.failure")
	External_Key_Invariants(key, "environment_source_failure_write.key")
	Environment_Source_Invariants(source, "environment_source_failure_write.source")
	if failure.Size.Value != 0 {
		failure_write(failure, Failure_Fragment("\n"))
	}
	known := true
	switch kind {
	case environment_malformed_failure:
		failure_write(failure, Failure_Fragment("environment variable "))
		failure_write(failure, Failure_Fragment(string(key)))
		failure_write(failure, Failure_Fragment(" has no '='"))
		if source.Occurrences > 1 {
			failure_write(failure, Failure_Fragment("\nenvironment variable "))
			failure_write(failure, Failure_Fragment(string(key)))
			failure_write(failure, Failure_Fragment(" is present more than once"))
		}
	case environment_duplicate_failure:
		failure_write(failure, Failure_Fragment("environment variable "))
		failure_write(failure, Failure_Fragment(string(key)))
		failure_write(failure, Failure_Fragment(" is present more than once"))
	case environment_missing_failure:
		failure_write(failure, Failure_Fragment("required environment variable "))
		failure_write(failure, Failure_Fragment(string(key)))
		failure_write(failure, Failure_Fragment(" is missing"))
	case external_integer_parse_failure, external_boolean_parse_failure:
		expected := "a whole number"
		if kind == external_boolean_parse_failure {
			expected = "a Boolean"
		}
		failure_write(failure, Failure_Fragment(string(key)))
		failure_write(failure, Failure_Fragment(" expects "))
		failure_write(failure, Failure_Fragment(expected))
		failure_write(failure, Failure_Fragment(", but got "))
		failure_write_quoted(failure, Failure_Fragment(string(source.Value)))
	default:
		known = false
	}
	aver.Always(known, "Environment source failure has known identity.")
}

// Enum failure rendering stays after conversion so membership owns no output state.
func environment_enum_failure_write(
	failure Failure_Writer, key External_Key, enumeration External_Enumeration,
	value Environment_Value_Text, kind error,
) {
	Failure_Writer_Invariants(failure, "environment_enum_failure_write.failure")
	External_Key_Invariants(key, "environment_enum_failure_write.key")
	External_Enumeration_Invariants(enumeration, "environment_enum_failure_write.enumeration")
	Environment_Value_Text_Invariants(value, "environment_enum_failure_write.value")
	if failure.Size.Value != 0 {
		failure_write(failure, Failure_Fragment("\n"))
	}
	known := true
	switch kind {
	case external_string_enum_failure:
		var workspace levenshtein.Workspace
		match, close, _ := levenshtein.Closest(levenshtein.Closest_Input{
			Workspace: &workspace,
			Target:    levenshtein.Target_Text_Unvalidated(value),
			Candidates: levenshtein.Candidates_Unvalidated(
				enumeration.String),
		})
		failure_write(failure, Failure_Fragment("invalid value "))
		failure_write_quoted(failure, Failure_Fragment(string(value)))
		failure_write(failure, Failure_Fragment(" for "))
		failure_write(failure, Failure_Fragment(string(key)))
		if close {
			failure_write(failure, Failure_Fragment(", did you mean "))
			failure_write_quoted(failure, Failure_Fragment(string(match)))
			failure_write(failure, Failure_Fragment("?"))
			return
		}
		failure_write(failure, Failure_Fragment("; allowed: "))
		failure_write_string_set(failure, Failure_String_Set(enumeration.String))
	case external_integer_enum_failure:
		number, _ := strconv.Parse_Decimal(strconv.Text(value))
		failure_write(failure, Failure_Fragment("invalid value "))
		failure_write_decimal(failure, Failure_Integer{Value: Integer(number)})
		failure_write(failure, Failure_Fragment(" for "))
		failure_write(failure, Failure_Fragment(string(key)))
		failure_write(failure, Failure_Fragment("; allowed: "))
		failure_write_integer_set(failure, Failure_Integer_Set(enumeration.Integers))
	default:
		known = false
	}
	aver.Always(known, "Environment enum failure has known identity.")
}

// Secret conversion reads caller bytes directly so accepted content never becomes a string.
func external_convert_secret_state_result(
	state Secret_Value_State, failure Secret_Content_Failure_Kind_Value,
) (_ Secret_Value_State, _ Secret_Content_Failure_Kind_Value) {
	Secret_Value_State_Invariants(state, "external_convert_secret_state.state")
	Secret_Content_Failure_Kind_Value_Invariants(
		failure, "external_convert_secret_state.failure",
	)
	return state, failure
}

func external_convert_secret_state(
	key Resolved_Secret_Key,
	type_state External_Type_State,
	enumeration External_Enumeration,
	raw Secret_Value_Bytes,
) (_ Secret_Value_State, _ Secret_Content_Failure_Kind_Value) {
	Resolved_Secret_Key_Invariants(key, "external_convert_secret_state.key")
	External_Type_State_Invariants(type_state, "external_convert_secret_state.type_state")
	External_Enumeration_Invariants(enumeration, "external_convert_secret_state.enumeration")
	Secret_Value_Bytes_Invariants(raw, "external_convert_secret_state.raw")
	state := Secret_Value_State{}
	converted := true
	switch Option_Type(type_state.Value) {
	case OPTION_TYPE_STRING:
		if enumeration.String != nil {
			permitted := false
			for _, member := range enumeration.String {
				if secret_bytes_equal_text(raw, Value_Text(member)) {
					permitted = true
					break
				}
			}
			if !permitted {
				return external_convert_secret_state_result(
					state, Secret_Content_Failure_Kind_Value(
						PATH_FAILURE_KIND_STRING_ENUM),
				)
			}
		}
		state.Bytes = raw
	case OPTION_TYPE_INTEGER:
		if len(raw) > strings.TEXT_SIZE_MAXIMUM {
			return external_convert_secret_state_result(
				state, Secret_Content_Failure_Kind_Value(PATH_FAILURE_KIND_INTEGER))
		}
		number, parsed := secret_decimal(Secret_Number_Bytes(raw))
		if !parsed {
			return external_convert_secret_state_result(
				state, Secret_Content_Failure_Kind_Value(PATH_FAILURE_KIND_INTEGER))
		}
		if enumeration.Integers != nil {
			if !slices.Contains(enumeration.Integers, int(number)) {
				return external_convert_secret_state_result(
					state, Secret_Content_Failure_Kind_Value(
						PATH_FAILURE_KIND_INTEGER_ENUM),
				)
			}
		}
		state.Integer = number
	case OPTION_TYPE_BOOLEAN:
		boolean, parsed := secret_parse_boolean(raw)
		if !parsed {
			return external_convert_secret_state_result(
				state, Secret_Content_Failure_Kind_Value(PATH_FAILURE_KIND_BOOLEAN))
		}
		state.Boolean = Boolean(boolean)
	default:
		converted = false
	}
	aver.Always(converted, "Secret conversion receives validated scalar type.")
	state.Parsed = true
	return external_convert_secret_state_result(
		state, Secret_Content_Failure_Kind_Value(PATH_FAILURE_KIND_NONE))
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

func secret_decimal_result(
	number Integer, parsed Boolean,
) (_ Integer, _ Boolean) {
	Integer_Invariants(number, "secret_decimal.number")
	Boolean_Invariants(parsed, "secret_decimal.parsed")
	return number, parsed
}

func secret_decimal(raw Secret_Number_Bytes) (_ Integer, _ Boolean) {
	Secret_Number_Bytes_Invariants(raw, "secret_decimal.raw")
	if len(raw) == 0 {
		return secret_decimal_result(0, false)
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
		return secret_decimal_result(0, false)
	}
	limit := uint64(bits.INTEGER_MAXIMUM)
	if negative {
		limit++
	}
	var magnitude uint64
	for _, character := range raw[start:] {
		if character < '0' {
			return secret_decimal_result(0, false)
		}
		if character > '9' {
			return secret_decimal_result(0, false)
		}
		digit := uint64(character - '0')
		if magnitude > (limit-digit)/10 {
			return secret_decimal_result(0, false)
		}
		magnitude = magnitude*10 + digit
	}
	if negative {
		if magnitude == uint64(bits.INTEGER_MAXIMUM)+1 {
			return secret_decimal_result(Integer(bits.INTEGER_MINIMUM), true)
		}
		return secret_decimal_result(Integer(-int(magnitude)), true)
	}
	return secret_decimal_result(Integer(magnitude), true)
}

func secret_parse_boolean_result(
	value Boolean, parsed Boolean,
) (_ Boolean, _ Boolean) {
	Boolean_Invariants(value, "secret_parse_boolean.value")
	Boolean_Invariants(parsed, "secret_parse_boolean.parsed")
	return value, parsed
}

func secret_parse_boolean(raw Secret_Value_Bytes) (_ Boolean, _ Boolean) {
	Secret_Value_Bytes_Invariants(raw, "secret_parse_boolean.raw")
	true_values := [...]string{"1", "t", "T", "TRUE", "true", "True"}
	for _, candidate := range true_values {
		if secret_bytes_equal_text(raw, Value_Text(candidate)) {
			return secret_parse_boolean_result(true, true)
		}
	}
	false_values := [...]string{"0", "f", "F", "FALSE", "false", "False"}
	for _, candidate := range false_values {
		if secret_bytes_equal_text(raw, Value_Text(candidate)) {
			return secret_parse_boolean_result(false, true)
		}
	}
	return secret_parse_boolean_result(false, false)
}

// A default does not trigger a warning because no external source supplied it.
func environment_warning_needed(
	deprecated Deprecation,
	allow_empty Allow_Empty,
	value Environment_Value_Text,
	present Boolean,
) (needed Boolean) {
	defer func() { Boolean_Invariants(needed, "environment_warning_needed.needed") }()
	Deprecation_Invariants(deprecated, "environment_warning_needed.deprecated")
	Allow_Empty_Invariants(allow_empty, "environment_warning_needed.allow_empty")
	Environment_Value_Text_Invariants(value, "environment_warning_needed.value")
	Boolean_Invariants(present, "environment_warning_needed.present")
	if !present {
		return false
	}
	if value == "" {
		if !allow_empty {
			return false
		}
	}
	if deprecated == "" {
		return false
	}
	return true
}

// SECRET_STATE_COUNT follows the one mutable phase record per declaration.
const SECRET_STATE_COUNT = len("state") / len("state")

// SECRET_STAGE_IDLE encodes no submitted file operation.
const SECRET_STAGE_IDLE Secret_Stage_Value = 0

// SECRET_STAGE_OPEN encodes submitted open.
const SECRET_STAGE_OPEN = SECRET_STAGE_IDLE + 1

// SECRET_STAGE_READ encodes submitted read.
const SECRET_STAGE_READ = SECRET_STAGE_OPEN + 1

// SECRET_STAGE_CLOSE encodes submitted close.
const SECRET_STAGE_CLOSE = SECRET_STAGE_READ + 1

// SECRET_STAGE_COUNT follows one phase discriminator cell.
const SECRET_STAGE_COUNT = len("stage") / len("stage")

// Secret_Stage_Value is one bounded runner operation selector.
type Secret_Stage_Value uint8

// Secret_Stage_Value_Invariants admits idle and submitted operations.
func Secret_Stage_Value_Invariants(value Secret_Stage_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(SECRET_STAGE_IDLE), uint8(SECRET_STAGE_OPEN),
			uint8(SECRET_STAGE_READ), uint8(SECRET_STAGE_CLOSE),
		).
		Ensure()
}

// Secret_Stage identifies the one operation currently submitted by a runner.
type Secret_Stage struct {
	// Value isolates runner phase from shared numeric coverage.
	Value Secret_Stage_Value
}

// Secret_Stage_Invariants admits idle and every submitted file operation.
func Secret_Stage_Invariants(value Secret_Stage, namespace aver.Namespace) {
	Secret_Stage_Value_Invariants(value.Value, namespace)
}

// Secret_Status_Failure_Kind_Value excludes failures after status validation.
type Secret_Status_Failure_Kind_Value uint8

// Secret_Status_Failure_Kind_Value_Invariants bounds the status rejection phase.
func Secret_Status_Failure_Kind_Value_Invariants(
	value Secret_Status_Failure_Kind_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(PATH_FAILURE_KIND_NONE),
			uint8(PATH_FAILURE_KIND_FILE_TOO_LARGE),
		).
		Ensure()
}

// Secret_Status_Failure is status rejection before open submission.
type Secret_Status_Failure struct {
	// Kind excludes failures impossible before open submission.
	Kind Secret_Status_Failure_Kind_Value
	// Cause keeps injected status failure identity.
	Cause error
}

// Secret_Status_Failure_Invariants bounds status rejection state.
func Secret_Status_Failure_Invariants(
	value Secret_Status_Failure, namespace aver.Namespace,
) {
	Secret_Status_Failure_Kind_Value_Invariants(value.Kind, namespace)
}

// Secret_Open_Failure_Kind_Value admits only open completion outcomes.
type Secret_Open_Failure_Kind_Value uint8

// Secret_Open_Failure_Kind_Value_Invariants bounds descriptor acquisition failures.
func Secret_Open_Failure_Kind_Value_Invariants(
	value Secret_Open_Failure_Kind_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(PATH_FAILURE_KIND_NONE),
			uint8(PATH_FAILURE_KIND_OPEN), uint8(PATH_FAILURE_KIND_OPEN_NO_FILE),
		).
		Ensure()
}

// Secret_Open_Failure is descriptor acquisition failure before read submission.
type Secret_Open_Failure struct {
	// Kind excludes status and content failures.
	Kind Secret_Open_Failure_Kind_Value
	// Cause keeps injected open failure identity.
	Cause error
}

// Secret_Open_Failure_Invariants bounds descriptor acquisition state.
func Secret_Open_Failure_Invariants(
	value Secret_Open_Failure, namespace aver.Namespace,
) {
	Secret_Open_Failure_Kind_Value_Invariants(value.Kind, namespace)
}

// Secret_Content_Failure_Kind_Value excludes status and open failures.
type Secret_Content_Failure_Kind_Value uint8

// Secret_Content_Failure_Kind_Value_Invariants bounds read and conversion failures.
func Secret_Content_Failure_Kind_Value_Invariants(
	value Secret_Content_Failure_Kind_Value, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint8(
			uint8(value), uint8(PATH_FAILURE_KIND_NONE),
			uint8(PATH_FAILURE_KIND_BOOLEAN), uint8(PATH_FAILURE_KIND_STATUS),
			uint8(PATH_FAILURE_KIND_ABSENT), uint8(PATH_FAILURE_KIND_NOT_REGULAR),
		).
		Ensure()
	aver.Always(
		value != Secret_Content_Failure_Kind_Value(PATH_FAILURE_KIND_NEGATIVE_SIZE),
		"Content failure follows successful status validation.",
	)
	aver.Always(
		value != Secret_Content_Failure_Kind_Value(PATH_FAILURE_KIND_OPEN),
		"Content failure cannot precede a successful open.",
	)
	aver.Always(
		value != Secret_Content_Failure_Kind_Value(PATH_FAILURE_KIND_OPEN_NO_FILE),
		"Content failure requires one acquired descriptor.",
	)
}

// Secret_Content_Failure is content state held until descriptor retirement.
type Secret_Content_Failure struct {
	// Kind excludes failures resolved before read submission.
	Kind Secret_Content_Failure_Kind_Value
	// Cause keeps injected read failure identity for later rendering.
	Cause error
	// Close_Cause keeps descriptor retirement failure separate from content failure.
	Close_Cause error
}

// Secret_Content_Failure_Invariants bounds content failure state.
func Secret_Content_Failure_Invariants(
	value Secret_Content_Failure, namespace aver.Namespace,
) {
	Secret_Content_Failure_Kind_Value_Invariants(value.Kind, namespace)
}

// Current_Failure holds content failure state until Close supplies path state.
type Current_Failure struct {
	// Kind classifies current read result before descriptor retirement.
	Kind Path_Failure_Kind
	// Cause keeps injected read failure identity for later rendering.
	Cause error
	// Close_Cause keeps descriptor retirement failure separate from content failure.
	Close_Cause error
}

// Current_Failure_Invariants excludes path state unavailable before Close.
func Current_Failure_Invariants(value Current_Failure, namespace aver.Namespace) {
	Path_Failure_Kind_Invariants(value.Kind, namespace)
}

// Secret_State stores fields whose legal values depend on open/read/close phase.
type Secret_State struct {
	// Cursor reaches path count after every fallback retires.
	Cursor Secret_Path_Cursor
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
	Candidate Secret_Value_State
	// Current_Failure is redacted content state before Close supplies path state.
	Current_Failure Secret_Content_Failure
	// Current_Absent classifies a missing or empty path.
	Current_Absent Current_Absent
}

// Secret_State_Invariants enforces phase-union storage without impossible branch trees.
func Secret_State_Invariants(value Secret_State, namespace aver.Namespace) {
	Secret_Path_Cursor_Invariants(value.Cursor, namespace)
	Path_Failures_Invariants(value.Path_Failures, namespace)
	Only_Absent_Invariants(value.Only_Absent, namespace)
	Secret_Size_Invariants(value.Status_Size, namespace)
	Secret_Bytes_Invariants(value.Buffer, namespace)
	Secret_Value_State_Invariants(value.Candidate, namespace)
	Secret_Content_Failure_Invariants(value.Current_Failure, namespace)
	Current_Absent_Invariants(value.Current_Absent, namespace)
}

// Publication_Handle is one live caller-owned publication.
type Publication_Handle *Publication

// Publication_Handle_Invariants composes present publication storage.
func Publication_Handle_Invariants(value Publication_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Publication_Invariants(*value, namespace)
}

// One state owns every completion for one declaration and never moves after submission.
type Secret_Parser struct {
	// Completion carries operation result back to caller-driven parser polling.
	Completion nbio.Completion
	// Index is the secret declaration index.
	Index Secret_Index
	// Stage identifies the submitted operation retired by Completion.
	Stage Secret_Stage
	// State holds the phase-dependent file operation record.
	State Secret_State
}

// Secret_Parser_Invariants composes bounded per-declaration asynchronous state.
func Secret_Parser_Invariants(value Secret_Parser, namespace aver.Namespace) {
	Secret_Index_Invariants(value.Index, namespace)
	Secret_Stage_Invariants(value.Stage, namespace)
	Secret_State_Invariants(value.State, namespace)
}

// Secret_Parser_Handle is one live asynchronous secret state.
type Secret_Parser_Handle *Secret_Parser

// Secret_Parser_Handle_Invariants composes present secret parser storage.
func Secret_Parser_Handle_Invariants(
	value Secret_Parser_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Secret_Parser_Invariants(*value, namespace)
}

// Secret publication storage covers each declaration before any operation starts.
func secret_results_initialize(
	errors Active_Secret_Failures, warnings Active_Secret_Warnings,
	count Active_Secret_Count,
) {
	Active_Secret_Failures_Invariants(errors, "secret_results_initialize.errors")
	Active_Secret_Warnings_Invariants(warnings, "secret_results_initialize.warnings")
	Active_Secret_Count_Invariants(count, "secret_results_initialize.count")
	aver.Always(
		len(errors) >= int(count),
		"Program_Parse Secret_Errors storage fits declarations.",
	)
	aver.Always(
		len(warnings) >= int(count),
		"Program_Parse Secret_Warnings storage fits declarations.",
	)
	clear(errors)
	clear(warnings)
}

// Parent-free initialization prevents storage checks from observing publication state.
func parser_initialize_secrets(
	secrets Active_Secrets,
	parsers Active_Secret_Parsers,
	buffers Active_Secret_Buffers,
	path_failures Active_Secret_Path_Failures,
) {
	Active_Secrets_Invariants(secrets, "parser_initialize_secrets.secrets")
	Active_Secret_Parsers_Invariants(parsers, "parser_initialize_secrets.parsers")
	Active_Secret_Buffers_Invariants(buffers, "parser_initialize_secrets.buffers")
	Active_Secret_Path_Failures_Invariants(
		path_failures, "parser_initialize_secrets.path_failures",
	)
	secret_count := len(secrets)
	aver.Always(
		len(parsers) >= secret_count,
		"Program_Parse Secret_Parsers storage fits declarations.",
	)
	aver.Always(
		len(buffers) >= secret_count,
		"Program_Parse Secret_Buffers storage fits declarations.",
	)
	aver.Always(
		len(path_failures) >= secret_count,
		"Program_Parse Secret_Path_Failures storage fits declarations.",
	)
	for index := range secrets {
		aver.Always(
			len(buffers[index]) == SECRET_BUFFER_BYTES_MAX,
			"Program_Parse secret buffer has fixed overflow witness size.",
		)
		secret := secrets[index]
		aver.Always(
			len(path_failures[index]) >= len(secret.Paths),
			"Program_Parse secret path failure storage fits fallback paths.",
		)
		state := &parsers[index]
		*state = Secret_Parser{
			Index: Secret_Index(index),
			State: Secret_State{
				Only_Absent:   true,
				Buffer:        buffers[index],
				Path_Failures: path_failures[index][:0],
			},
		}
	}
}

// Shared sweep keeps initial and resumed operation accounting identical.
func secret_advance_all(
	secrets Active_Secrets, parsers Active_Secret_Parsers, loop Secret_IO,
	errors Active_Secret_Failures, warnings Active_Secret_Warnings,
) (work_count Declaration_Count) {
	defer func() {
		Declaration_Count_Invariants(work_count, "secret_advance_all.work_count")
	}()
	Active_Secrets_Invariants(secrets, "secret_advance_all.secrets")
	Active_Secret_Parsers_Invariants(parsers, "secret_advance_all.parsers")
	Secret_IO_Invariants(loop, "secret_advance_all.loop")
	Active_Secret_Failures_Invariants(errors, "secret_advance_all.errors")
	Active_Secret_Warnings_Invariants(warnings, "secret_advance_all.warnings")
	aver.Always(
		len(parsers) >= len(secrets),
		"Active secret parsers fit declarations.",
	)
	for index := range secrets {
		state := &parsers[index]
		secret := &secrets[index]
		for int(state.State.Cursor) < len(secret.Paths) &&
			state.Completion.Callback == nil {
			secret_advance(
				parsers, Secret_Index(index), secrets, loop, errors, warnings)
		}
		if int(state.State.Cursor) < len(secret.Paths) {
			work_count++
		}
	}
	return work_count
}

// One state-machine boundary observes every phase before it mutates or submits work.
func secret_advance(parsers Active_Secret_Parsers, index Secret_Index,
	secrets Active_Secrets, loop Secret_IO,
	errors Active_Secret_Failures, warnings Active_Secret_Warnings) {
	Active_Secret_Parsers_Invariants(parsers, "secret_advance.parsers")
	Secret_Index_Invariants(index, "secret_advance.index")
	Active_Secrets_Invariants(secrets, "secret_advance.secrets")
	Secret_IO_Invariants(loop, "secret_advance.loop")
	Active_Secret_Failures_Invariants(errors, "secret_advance.errors")
	Active_Secret_Warnings_Invariants(warnings, "secret_advance.warnings")
	aver.Always(int(index) < len(parsers), "Secret parser index belongs to active storage.")
	state, secret, phase := &parsers[index], &secrets[index], &parsers[index].State
	for range secret.Paths {
		path := Resolved_Secret_Path(secret.Paths[phase.Cursor])
		switch state.Stage.Value {
		case SECRET_STAGE_IDLE:
			failure, absent, size, opened := secret_open_path(loop, path)
			if opened {
				phase.Status_Size, state.Stage.Value = size, SECRET_STAGE_OPEN
				secret_submit_open(loop, &state.Completion, path)
				return
			}
			current := Current_Failure{Cause: failure.Cause}
			current.Kind.Value = Path_Failure_Kind_Value(failure.Kind)
			path_failures := Writable_Path_Failures(phase.Path_Failures)
			recorded, absence := secret_record_path(
				path, path_failures, phase.Only_Absent, current, absent)
			phase.Path_Failures, phase.Only_Absent = Path_Failures(recorded), absence
			phase.Cursor++
		case SECRET_STAGE_OPEN:
			failure, file, read := secret_read_path(state.Completion)
			if read {
				phase.File, state.Stage.Value = file, SECRET_STAGE_READ
				secret_submit_read(loop, &state.Completion, phase.File,
					Secret_Buffer(phase.Buffer))
				return
			}
			current := Current_Failure{Cause: failure.Cause}
			current.Kind.Value = Path_Failure_Kind_Value(failure.Kind)
			path_failures := Writable_Path_Failures(phase.Path_Failures)
			recorded, absence := secret_record_path(
				path, path_failures, phase.Only_Absent, current, false)
			phase.Path_Failures, phase.Only_Absent = Path_Failures(recorded), absence
			phase.Cursor, state.Stage.Value = phase.Cursor+1, SECRET_STAGE_IDLE
		case SECRET_STAGE_READ:
			phase.Current_Failure, phase.Current_Absent, phase.Candidate = secret_read(
				secret.Key, secret.Type, secret.Enumeration, secret.Allow_Empty,
				phase.Status_Size, Secret_Buffer(phase.Buffer), state.Completion)
			state.Stage.Value = SECRET_STAGE_CLOSE
			secret_submit_close(loop, &state.Completion, phase.File)
			return
		case SECRET_STAGE_CLOSE:
			var closed Boolean
			phase.Path_Failures, phase.Only_Absent, closed = secret_close_path(
				path, Writable_Path_Failures(phase.Path_Failures),
				phase.Only_Absent, phase.Current_Failure,
				phase.Current_Absent, state.Completion.Error)
			if !closed {
				phase.Cursor, state.Stage.Value = phase.Cursor+1, SECRET_STAGE_IDLE
				continue
			}
			secret_accept(secrets, warnings, state.Index,
				phase.Candidate, secret.Key, secret.Deprecated)
			phase.Cursor = Secret_Path_Cursor(len(secret.Paths))
			return
		}
	}
	aver.Always(state.Stage.Value == SECRET_STAGE_IDLE,
		"Secret parser exhausts paths only after every operation retires.")
	secret_finish(errors, state.Index,
		secret.Required, phase.Only_Absent, Failed_Paths(phase.Path_Failures))
}

// Terminal absence and failure publish only after every fallback retires.
func secret_finish(
	errors Active_Secret_Failures, index Secret_Index, required Required,
	only_absent Only_Absent, failures Failed_Paths,
) {
	Active_Secret_Failures_Invariants(errors, "secret_finish.errors")
	Secret_Index_Invariants(index, "secret_finish.index")
	Required_Invariants(required, "secret_finish.required")
	Only_Absent_Invariants(only_absent, "secret_finish.only_absent")
	Failed_Paths_Invariants(failures, "secret_finish.failures")
	aver.Always(
		int(index) < len(errors),
		"Secret failure index belongs to publication storage.",
	)
	present := Boolean(required)
	if !present {
		present = !Boolean(only_absent)
	}
	if present {
		errors[index] = Secret_Failure{
			Present: true, Paths: Path_Failures(failures),
		}
	}
}

// Status rejection never opens a path that cannot become one accepted secret.
func secret_open_path_result(
	failure Secret_Status_Failure, absent Current_Absent,
	size Secret_Size, opened Boolean,
) (_ Secret_Status_Failure, _ Current_Absent, _ Secret_Size, _ Boolean) {
	Secret_Status_Failure_Invariants(failure, "secret_open_path.failure")
	Current_Absent_Invariants(absent, "secret_open_path.absent")
	Secret_Size_Invariants(size, "secret_open_path.size")
	Boolean_Invariants(opened, "secret_open_path.opened")
	return failure, absent, size, opened
}

func secret_open_path(
	loop Secret_IO, path Resolved_Secret_Path,
) (
	_ Secret_Status_Failure, _ Current_Absent,
	_ Secret_Size, _ Boolean,
) {
	Secret_IO_Invariants(loop, "secret_open_path.loop")
	Resolved_Secret_Path_Invariants(path, "secret_open_path.path")
	failure := Secret_Status_Failure{}
	absent := Current_Absent(false)
	size := Secret_Size(0)
	opened := Boolean(false)
	status, status_err := loop.Status_Procedure(loop.Backend, path)
	switch {
	case status_err != nil:
		failure.Kind = Secret_Status_Failure_Kind_Value(PATH_FAILURE_KIND_STATUS)
		failure.Cause = status_err
	case !status.Exists:
		failure.Kind = Secret_Status_Failure_Kind_Value(PATH_FAILURE_KIND_ABSENT)
		absent = true
	case !nbio.File_Mode_Is_Regular(status.Mode):
		failure.Kind = Secret_Status_Failure_Kind_Value(PATH_FAILURE_KIND_NOT_REGULAR)
	case status.Size < 0:
		failure.Kind = Secret_Status_Failure_Kind_Value(PATH_FAILURE_KIND_NEGATIVE_SIZE)
	case status.Size > SECRET_BYTES_MAX:
		failure.Kind = Secret_Status_Failure_Kind_Value(PATH_FAILURE_KIND_FILE_TOO_LARGE)
	default:
		size = Secret_Size(status.Size)
		opened = true
	}
	return secret_open_path_result(failure, absent, size, opened)
}

// Open submission follows stage publication so synchronous doubles see open state.
func secret_submit_open(
	loop Secret_IO, completion nbio.Completion_Handle, path Resolved_Secret_Path,
) {
	Secret_IO_Invariants(loop, "secret_submit_open.loop")
	Resolved_Secret_Path_Invariants(path, "secret_submit_open.path")
	loop.Open_Procedure(loop.Backend, completion, path, secret_operation_complete)
}

// Open result owns descriptor validation before any read submission.
func secret_read_path_result(
	failure Secret_Open_Failure, file nbio.File, read Boolean,
) (_ Secret_Open_Failure, _ nbio.File, _ Boolean) {
	Secret_Open_Failure_Invariants(failure, "secret_read_path.failure")
	Boolean_Invariants(read, "secret_read_path.read")
	return failure, file, read
}

func secret_read_path(
	completion nbio.Completion,
) (_ Secret_Open_Failure, _ nbio.File, _ Boolean) {
	failure := Secret_Open_Failure{}
	file := nbio.File(0)
	if completion.Error != nil {
		failure.Kind = Secret_Open_Failure_Kind_Value(PATH_FAILURE_KIND_OPEN)
		failure.Cause = completion.Error
		return secret_read_path_result(failure, file, false)
	}
	if completion.Data < 0 {
		failure.Kind = Secret_Open_Failure_Kind_Value(PATH_FAILURE_KIND_OPEN_NO_FILE)
		return secret_read_path_result(failure, file, false)
	}
	file = nbio.File(completion.Data)
	return secret_read_path_result(failure, file, true)
}

// Read submission follows descriptor publication so synchronous doubles see read state.
func secret_submit_read(
	loop Secret_IO, completion nbio.Completion_Handle,
	file nbio.File, buffer Secret_Buffer,
) {
	Secret_IO_Invariants(loop, "secret_submit_read.loop")
	Secret_Buffer_Invariants(buffer, "secret_submit_read.buffer")
	loop.Read_Procedure(
		loop.Backend, completion, file, buffer, secret_operation_complete,
	)
}

// Close submission follows content publication so synchronous doubles see close state.
func secret_submit_close(
	loop Secret_IO, completion nbio.Completion_Handle, file nbio.File,
) {
	Secret_IO_Invariants(loop, "secret_submit_close.loop")
	loop.Close_Procedure(loop.Backend, completion, file, secret_operation_complete)
}

// Read classification keeps hostile completion counts outside slice indexing.
func secret_read_result(
	failure Secret_Content_Failure, absent Current_Absent,
	candidate Secret_Value_State,
) (_ Secret_Content_Failure, _ Current_Absent, _ Secret_Value_State) {
	Secret_Content_Failure_Invariants(failure, "secret_read.failure")
	Current_Absent_Invariants(absent, "secret_read.absent")
	Secret_Value_State_Invariants(candidate, "secret_read.candidate")
	return failure, absent, candidate
}

func secret_read(
	key Secret_Key, type_state External_Type_State,
	enumeration External_Enumeration, allow_empty Allow_Empty,
	status_size Secret_Size, buffer Secret_Buffer, completion nbio.Completion,
) (_ Secret_Content_Failure, _ Current_Absent, _ Secret_Value_State) {
	Secret_Key_Invariants(key, "secret_read.key")
	External_Type_State_Invariants(type_state, "secret_read.type_state")
	External_Enumeration_Invariants(enumeration, "secret_read.enumeration")
	Allow_Empty_Invariants(allow_empty, "secret_read.allow_empty")
	Secret_Size_Invariants(status_size, "secret_read.status_size")
	Secret_Buffer_Invariants(buffer, "secret_read.buffer")
	failure := Secret_Content_Failure{}
	candidate := Secret_Value_State{}
	count := Secret_Read_Count(completion.Data)
	Secret_Read_Count_Invariants(count, "secret_read.count")
	if completion.Error != nil {
		failure.Kind = Secret_Content_Failure_Kind_Value(PATH_FAILURE_KIND_READ)
		failure.Cause = completion.Error
		return secret_read_result(failure, false, candidate)
	}
	if count < 0 {
		failure.Kind = Secret_Content_Failure_Kind_Value(PATH_FAILURE_KIND_NEGATIVE_READ)
		return secret_read_result(failure, false, candidate)
	}
	if count > SECRET_BYTES_MAX {
		failure.Kind = Secret_Content_Failure_Kind_Value(PATH_FAILURE_KIND_FILE_TOO_LARGE)
		return secret_read_result(failure, false, candidate)
	}
	if Secret_Size(count) != status_size {
		failure.Kind = Secret_Content_Failure_Kind_Value(PATH_FAILURE_KIND_SIZE_CHANGED)
		return secret_read_result(failure, false, candidate)
	}
	content := Secret_Value_Bytes(buffer[:count])
	if len(content) >= len("\r\n") {
		if content[len(content)-len("\r\n")] == '\r' {
			if content[len(content)-len("\n")] == '\n' {
				content = content[:len(content)-len("\r\n")]
			}
		} else if content[len(content)-len("\n")] == '\n' {
			content = content[:len(content)-len("\n")]
		}
	} else if len(content) >= len("\n") {
		if content[len(content)-len("\n")] == '\n' {
			content = content[:len(content)-len("\n")]
		}
	}
	if len(content) == 0 {
		if !allow_empty {
			failure.Kind = Secret_Content_Failure_Kind_Value(PATH_FAILURE_KIND_EMPTY)
			return secret_read_result(failure, true, candidate)
		}
	}
	candidate, failure.Kind = external_convert_secret_state(
		Resolved_Secret_Key(key), type_state, enumeration, content,
	)
	return secret_read_result(failure, false, candidate)
}

// Close failure defeats a successful read because descriptor retirement is required.
func secret_close_path_result(
	stored Path_Failures, absence Only_Absent, complete Boolean,
) (_ Path_Failures, _ Only_Absent, _ Boolean) {
	Path_Failures_Invariants(stored, "secret_close_path.stored")
	Only_Absent_Invariants(absence, "secret_close_path.absence")
	Boolean_Invariants(complete, "secret_close_path.complete")
	return stored, absence, complete
}

func secret_close_path(
	path Resolved_Secret_Path, path_failures Writable_Path_Failures,
	only_absent Only_Absent,
	failure Secret_Content_Failure, absent Current_Absent, close_err error,
) (_ Path_Failures, _ Only_Absent, _ Boolean) {
	Resolved_Secret_Path_Invariants(path, "secret_close_path.path")
	Writable_Path_Failures_Invariants(path_failures, "secret_close_path.path_failures")
	Only_Absent_Invariants(only_absent, "secret_close_path.only_absent")
	Secret_Content_Failure_Invariants(failure, "secret_close_path.failure")
	Current_Absent_Invariants(absent, "secret_close_path.absent")
	complete := Boolean(
		failure.Kind == Secret_Content_Failure_Kind_Value(PATH_FAILURE_KIND_NONE))
	if complete {
		complete = close_err == nil
	}
	if complete {
		return secret_close_path_result(
			Path_Failures(path_failures), only_absent, true)
	}
	if close_err != nil {
		absent = false
	}
	failure.Close_Cause = close_err
	current := Current_Failure{
		Cause: failure.Cause, Close_Cause: failure.Close_Cause,
	}
	current.Kind.Value = Path_Failure_Kind_Value(failure.Kind)
	recorded, absence := secret_record_path(
		path, path_failures, only_absent, current, absent)
	return secret_close_path_result(Path_Failures(recorded), absence, false)
}

// Accepted content publishes only after descriptor retirement succeeds.
func secret_accept(
	secrets Active_Secrets, warnings Active_Secret_Warnings,
	index Secret_Index, candidate Secret_Value_State,
	key Secret_Key, deprecated Deprecation,
) {
	Active_Secrets_Invariants(secrets, "secret_accept.secrets")
	Active_Secret_Warnings_Invariants(warnings, "secret_accept.warnings")
	Secret_Index_Invariants(index, "secret_accept.index")
	Secret_Value_State_Invariants(candidate, "secret_accept.candidate")
	Secret_Key_Invariants(key, "secret_accept.key")
	Deprecation_Invariants(deprecated, "secret_accept.deprecated")
	aver.Always(candidate.Parsed, "Accepted secret candidate completed conversion.")
	aver.Always(
		int(index) < len(secrets),
		"Accepted secret index belongs to active secrets.",
	)
	aver.Always(
		int(index) < len(warnings),
		"Accepted secret warning index belongs to publication storage.",
	)
	secrets[index].State = Accepted_Secret_Value{
		Bytes: candidate.Bytes, Integer: candidate.Integer, Boolean: candidate.Boolean,
	}
	if deprecated != "" {
		warnings[index] = Warning{
			Kind:     Warning_Kind{Value: WARNING_KIND_SECRET},
			Name:     Warning_Name{Value: Value_Text(key)},
			Guidance: Warning_Guidance{Value: deprecated},
		}
	}
}

// Caller-driven Parser_Done observes retirement through cleared Completion.Callback.
func secret_operation_complete(completed nbio.Completion_Handle) {
	aver.Always(completed != nil, "Secret operation returns its completion.")
}

// One non-absence failure prevents an optional declaration from hiding operational faults.
func secret_record_path_result(
	stored Failed_Paths, absence Only_Absent,
) (_ Failed_Paths, _ Only_Absent) {
	Failed_Paths_Invariants(stored, "secret_record_path.stored")
	Only_Absent_Invariants(absence, "secret_record_path.absence")
	return stored, absence
}

func secret_record_path(
	path Resolved_Secret_Path, path_failures Writable_Path_Failures,
	only_absent Only_Absent, failure Current_Failure, absent Current_Absent,
) (_ Failed_Paths, _ Only_Absent) {
	Resolved_Secret_Path_Invariants(path, "secret_record_path.path")
	Writable_Path_Failures_Invariants(path_failures, "secret_record_path.path_failures")
	Only_Absent_Invariants(only_absent, "secret_record_path.only_absent")
	Current_Failure_Invariants(failure, "secret_record_path.current_failure")
	Current_Absent_Invariants(absent, "secret_record_path.absent")
	aver.Always(
		len(path_failures) < cap(path_failures),
		"Program_Parse secret path failure storage has room.",
	)
	failure_count := len(path_failures)
	stored := Failed_Paths(path_failures[:failure_count+1])
	path_failure := Path_Failure{
		Path: Path_Failure_Path{Value: path},
		Kind: failure.Kind, Cause: failure.Cause, Close_Cause: failure.Close_Cause,
	}
	stored[failure_count] = path_failure
	absence := only_absent
	if !absent {
		absence = false
	}
	return secret_record_path_result(stored, absence)
}

func failure_write_path(
	failure Failure_Writer, key Failure_Secret_Key, path_failure Path_Failure,
) {
	Failure_Writer_Invariants(failure, "failure_write_path.failure")
	Failure_Secret_Key_Invariants(key, "failure_write_path.key")
	Path_Failure_Invariants(path_failure, "failure_write_path.path_failure")
	failure_write(failure, Failure_Fragment(path_failure.Path.Value))
	failure_write(failure, Failure_Fragment(": "))
	switch path_failure.Kind.Value {
	case PATH_FAILURE_KIND_NONE:
	case PATH_FAILURE_KIND_STATUS:
		failure_write(failure, Failure_Fragment("status: "))
		failure_write(failure, failure_cause(path_failure.Cause))
	case PATH_FAILURE_KIND_ABSENT:
		failure_write(failure, Failure_Fragment("missing"))
	case PATH_FAILURE_KIND_NOT_REGULAR:
		failure_write(failure, Failure_Fragment("not a regular file"))
	case PATH_FAILURE_KIND_NEGATIVE_SIZE:
		failure_write(failure, Failure_Fragment("negative file size"))
	case PATH_FAILURE_KIND_FILE_TOO_LARGE:
		failure_write(failure, Failure_Fragment("file exceeds 64 KiB"))
	case PATH_FAILURE_KIND_OPEN:
		failure_write(failure, Failure_Fragment("open: "))
		failure_write(failure, failure_cause(path_failure.Cause))
	case PATH_FAILURE_KIND_OPEN_NO_FILE:
		failure_write(failure, Failure_Fragment("open returned no file"))
	case PATH_FAILURE_KIND_READ:
		failure_write(failure, Failure_Fragment("read: "))
		failure_write(failure, failure_cause(path_failure.Cause))
	case PATH_FAILURE_KIND_NEGATIVE_READ:
		failure_write(failure, Failure_Fragment("read returned a negative byte count"))
	case PATH_FAILURE_KIND_SIZE_CHANGED:
		failure_write(failure, Failure_Fragment("file size changed after status"))
	case PATH_FAILURE_KIND_EMPTY:
		failure_write(failure, Failure_Fragment("file is empty"))
	case PATH_FAILURE_KIND_STRING_ENUM, PATH_FAILURE_KIND_INTEGER_ENUM:
		failure_write(failure, Failure_Fragment(string(key.Value)))
		failure_write(failure, Failure_Fragment(" is not a permitted enum value"))
	case PATH_FAILURE_KIND_INTEGER:
		failure_write(failure, Failure_Fragment(string(key.Value)))
		failure_write(failure, Failure_Fragment(" is not a whole number"))
	case PATH_FAILURE_KIND_BOOLEAN:
		failure_write(failure, Failure_Fragment(string(key.Value)))
		failure_write(failure, Failure_Fragment(" is not a Boolean"))
	}
	if path_failure.Close_Cause != nil {
		if path_failure.Kind.Value != PATH_FAILURE_KIND_NONE {
			failure_write(failure, Failure_Fragment("\n"))
		}
		failure_write(failure, Failure_Fragment("close: "))
		failure_write(failure, failure_cause(path_failure.Close_Cause))
	}
}

func failure_cause(cause error) (text Failure_Fragment) {
	defer func() { Failure_Fragment_Invariants(text, "failure_cause.text") }()
	aver.Always(cause != nil, "Rendered failure cause is present.")
	message := cause.Error()
	aver.Always(
		len(message) <= strings.TEXT_SIZE_MAXIMUM,
		"Injected failure cause stays inside shared text capacity.",
	)
	return Failure_Fragment(message)
}

// Environment help can show defaults because environment values are not secrets.
func print_environment_help(output Output_Reference, environment Environment_Declarations) {
	Output_Reference_Invariants(output, "print_environment_help.output")
	Environment_Declarations_Invariants(environment, "print_environment_help.environment")
	has_shown := false
	first_width, second_width := Rendered_Size{}, Rendered_Size{}
	for index := range environment {
		variable := environment[index]
		if !external_shown(variable.Hidden, variable.Deprecated) {
			continue
		}
		has_shown = true
		value := Rendered_External{Type: variable.Type, String: variable.State.String,
			Integer: variable.State.Integer, Boolean: variable.State.Boolean}
		first_size := Rendered_Size_Value(len("    ") + len(variable.Key))
		if first_size > first_width.Value {
			first_width.Value = first_size
		}
		second_size := Rendered_Size{Value: Rendered_Size_Value(
			render_external_type(
				Output_Reference{}, variable.Type, variable.Enumeration),
		)}
		second_size.Value += Rendered_Size_Value(len(" ") + len("optional"))
		if !variable.Required {
			second_size.Value += Rendered_Size_Value(len(" default=")) +
				environment_default(Output_Reference{}, value).Value
		}
		if second_size.Value > second_width.Value {
			second_width = second_size
		}
	}
	if !has_shown {
		return
	}
	output_write(output, "Environment Variables:\n")
	for _, variable := range environment {
		if !external_shown(variable.Hidden, variable.Deprecated) {
			continue
		}
		value := Rendered_External{Type: variable.Type, String: variable.State.String,
			Integer: variable.State.Integer, Boolean: variable.State.Boolean}
		first_size := Rendered_Size_Value(len("    ") + len(variable.Key))
		output_write(output, "    ")
		output_write(output, Value_Text(variable.Key))
		for range first_width.Value - first_size {
			output_write_space(output)
		}
		second_size := Rendered_Size{Value: Rendered_Size_Value(
			render_external_type(output, variable.Type, variable.Enumeration),
		)}
		second_size.Value += Rendered_Size_Value(len(" ") + len("optional"))
		output_write(output, " ")
		if variable.Required {
			output_write(output, "required")
		} else {
			output_write(output, "optional default=")
			second_size.Value += Rendered_Size_Value(len(" default=")) +
				environment_default(output, value).Value
		}
		for range second_width.Value - second_size.Value {
			output_write_space(output)
		}
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
		first_size := Rendered_Size_Value(len("    ") + len(secret.Key))
		if first_size > first_width.Value {
			first_width.Value = first_size
		}
		second_size := Rendered_Size{Value: Rendered_Size_Value(
			render_external_type(Output_Reference{}, secret.Type, secret.Enumeration),
		)}
		second_size.Value += Rendered_Size_Value(len(" ") + len("optional"))
		if second_size.Value > second_width.Value {
			second_width = second_size
		}
	}
	output_write(output, "Secrets:\n")
	for _, secret := range secrets {
		if !external_shown(secret.Hidden, secret.Deprecated) {
			continue
		}
		first_size := Rendered_Size_Value(len("    ") + len(secret.Key))
		output_write(output, "    ")
		output_write(output, Value_Text(secret.Key))
		for range first_width.Value - first_size {
			output_write_space(output)
		}
		second_size := Rendered_Size{Value: Rendered_Size_Value(
			render_external_type(output, secret.Type, secret.Enumeration),
		)}
		second_size.Value += Rendered_Size_Value(len(" ") + len("optional"))
		output_write(output, " ")
		if secret.Required {
			output_write(output, "required")
		} else {
			output_write(output, "optional")
		}
		for range second_width.Value - second_size.Value {
			output_write_space(output)
		}
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

func render_external_type(
	output Output_Reference, type_state External_Type_State,
	enumeration External_Enumeration,
) (size External_Type_Size) {
	defer func() { External_Type_Size_Invariants(size, "render_external_type.size") }()
	Output_Reference_Invariants(output, "render_external_type.output")
	External_Type_State_Invariants(type_state, "render_external_type.type_state")
	External_Enumeration_Invariants(enumeration, "render_external_type.enumeration")
	size = External_Type_Size(len(option_type_text(type_state.Value)))
	if output.Value != nil {
		output_write(output, Value_Text(option_type_text(type_state.Value)))
	}
	if enumeration.String != nil {
		size += External_Type_Size(len("()"))
		if output.Value != nil {
			output_write(output, "(")
		}
		for index, member := range enumeration.String {
			if index != 0 {
				size += External_Type_Size(len("|"))
				if output.Value != nil {
					output_write(output, "|")
				}
			}
			size += External_Type_Size(len(member))
			if output.Value != nil {
				output_write(output, Value_Text(member))
			}
		}
		if output.Value != nil {
			output_write(output, ")")
		}
		return size
	}
	if enumeration.Integers != nil {
		size += External_Type_Size(len("()"))
		if output.Value != nil {
			output_write(output, "(")
		}
		for index, member := range enumeration.Integers {
			if index != 0 {
				size += External_Type_Size(len("|"))
				if output.Value != nil {
					output_write(output, "|")
				}
			}
			size += External_Type_Size(render_decimal(output,
				Rendered_Integer{Value: Integer(member)},
			))
		}
		if output.Value != nil {
			output_write(output, ")")
		}
	}
	return size
}

// Environment default writes when output exists and always returns rendered width.
func environment_default(
	output Output_Reference, value Rendered_External,
) (size Rendered_Size) {
	defer func() { Rendered_Size_Invariants(size, "environment_default.size") }()
	Output_Reference_Invariants(output, "environment_default.output")
	Rendered_External_Invariants(value, "environment_default.value")
	switch Option_Type(value.Type.Value) {
	case OPTION_TYPE_STRING:
		if output.Value != nil {
			output_write(output, Value_Text(value.String))
		}
		return Rendered_Size{Value: Rendered_Size_Value(len(value.String))}
	case OPTION_TYPE_INTEGER:
		return Rendered_Size{Value: Rendered_Size_Value(render_decimal(
			output, Rendered_Integer{Value: value.Integer},
		))}
	case OPTION_TYPE_BOOLEAN:
		if value.Boolean {
			if output.Value != nil {
				output_write(output, "true")
			}
			return Rendered_Size{Value: Rendered_Size_Value(len("true"))}
		}
		if output.Value != nil {
			output_write(output, "false")
		}
		return Rendered_Size{Value: Rendered_Size_Value(len("false"))}
	}
	return size
}
