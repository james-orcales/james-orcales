
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

A multicall program selects its command from the binary name in argv[0], as a busybox-
style symlinked binary does; every token after it is that command's argument. Run by its
own name rather than a link, the first token selects the command, as `busybox ls` does.

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

An enum option restricts its value to a fixed set: a flag enum defaults to a member, an
argument enum is required. A value outside the set errors, suggesting the closest member
for a string enum and otherwise listing the whole set.

### Help

Every program carries an auto-injected -help flag. When the token appears anywhere,
parsing short-circuits and returns the Help_Requested sentinel with the command context
— the selected command or the empty-label root — which Print_Requested_Help renders.

# Completion

Complete returns shell-completion candidates read from the live program: command names,
dashed option labels, and enum members. Completion_Script emits a bash, zsh, or fish
script that calls back into `__complete`, which Handle_Completion serves before parsing.

# Visibility

### Hidden

A hidden flag or command still parses and resolves, but appears in neither the help
output nor the completion candidates, so an internal or bootstrap option stays usable
without being advertised.

### Deprecated

A deprecated flag or command still parses but is hidden the same way, and using it
records a warning naming its guidance; Program_Parse gathers those warnings onto the
returned command and Print_Deprecations emits them.

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

A label that is empty, not flag-safe, or colliding panics, as does a non-terminal slice
argument. An enum that is empty, type-mismatched, defaulted outside its set, or on a
variadic panics; the reserved "help" label claimed by a user option panics too.
