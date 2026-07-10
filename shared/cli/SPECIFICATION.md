
# Parse

Program_Parse resolves the active command and populates its options, returning an
error on malformed input. A flag carries a default value; a positional argument has
none and may also be given by position, without its label.

### Commands

A named command resolves to itself and an absent command defaults to the first
declared; an unknown name returns an error that suggests the closest command when one
is a likely typo.

### Single Command

A program built with New_Single has no command selector: the first token is the
first positional argument, so a name that would select a sibling command is read as
a positional. Help drops the selector and shows the program's own positionals.

### Multicall

A multicall program selects its command from the binary name in argv[0], as a
busybox-style symlinked binary does; every token after it is that command's
argument. When argv[0] is not a known command — the binary run by its own name rather
than a verb link — the first token selects the command instead, as `busybox ls` does, so
a bootstrap command works before the links exist. An unknown name suggests the closest
command.

### Arguments

Positionals fill the command's arguments in declaration order, skipping any already
set by name; an integer argument must convert or the parse returns an error.

### Named

Every argument and flag is also settable by -label=value, in any order with the
positionals. A scalar set twice errors; an unknown option errors and suggests the
closest known one when it looks like a typo.

### Variadic

A command's last argument may be a slice, collecting the positionals left after the
scalar arguments and appending each repeated -label=value; a slice declared before
the last argument is rejected at construction.

### Flags

A flag assigns its value by type; a double-dash flag or a non-boolean flag without a
value returns an error.

### Enum

An enum option restricts its value to a fixed set. A flag enum carries a default that
must be a member; an argument enum is required and has none, set by position or by name
like any argument. A value outside the set returns an error that suggests the closest
member for a string enum and otherwise lists the whole set.

### Help

Every program carries an auto-injected -help flag, shown under its global flags. When the
token appears anywhere, parsing short-circuits before any assignment — so help works even
with a missing required argument — and returns the Help_Requested sentinel alongside the
command context: the selected command, or the empty-label root. Print_Requested_Help
renders the whole program for the root and one command's usage otherwise.

# Completion

Complete returns shell-completion candidates for a partially typed command line, read
from the live program: command names, then flag and named-argument labels for a leading
dash, an enum option's members after -label=, and an enum positional's members;
everything else is empty, leaving file completion to the shell. Completion_Script emits a
bash, zsh, or fish script that calls back into `__complete` (one registration per verb for
a multicall program); an unsupported shell errors. Handle_Completion serves the reserved
`completion` and `__complete` invocations before parsing.

# Trim Quotes

A quoted flag value is unquoted during parsing when it is wrapped in a matching
pair of double or single quotes.

### Cases

Double, single, empty, and space-bearing quoted values are unquoted, while
unquoted and mismatched-quote values pass through unchanged.

# Get Option

Get_Option returns the option carrying a given label from a slice of options.

### Lookup

A present label returns its option; an absent label panics.

# New

New validates a program's configuration and panics when it is malformed.

### Validation

A command without a label panics; so does an argument label that is not flag-safe or
that collides with another option's name. An enum with an empty set, an element type
that does not match its value, or — for a flag — a default outside the set, panics; a
variadic option may not be an enum. The label "help" is reserved for the auto-injected
-help flag, so a user option claiming it panics.
