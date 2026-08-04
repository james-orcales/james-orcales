
# Parse

Program_Parse starts a parser that resolves the active command and populates its options.
Parser_Done returns nil while secret I/O continues. It returns the command and error when all
work stops. A flag has a default. A positional argument has no default and can use its position.

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

Every program carries auto-injected -h and -help flags. When either token appears,
parsing stops and returns Help_Requested and the applicable command context.
Help shows public environment defaults and full secret paths, but it does not show secret values.

### Environment Variables

A program declares typed environment variables for all commands. Program_Parse reads declared
keys from an injected KEY=value list and ignores other entries. A declaration can be required,
can permit an empty value, can have a default, and can restrict the value to an enum.

### Secrets

A secret has ordered absolute paths that use one uppercase base filename as the key. The parser
starts secrets in parallel, accepts regular files of at most 64 KiB, and rejects a final link.
It removes one final LF or CRLF, converts the value, and closes each opened file.

### External Errors

The parser joins external errors in declaration order and does not disclose secret values.
Help and CLI syntax errors stop before external validation or file I/O.

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

A deprecated declaration stays usable but does not appear in help or completion output.
Program_Parse adds its guidance to the command when the applicable source supplies a value.
Print_Deprecations writes each warning.

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

# Get Environment

Get_Environment returns the environment variable that has a specified key.

### Lookup

A declared key returns its resolved environment variable. An undeclared key panics.

# Get Secret

Get_Secret returns the secret that has a specified key.

### Lookup

A declared key returns its resolved secret. An undeclared key panics.

# New

New validates a program's configuration and panics when it is malformed.

### Validation

An invalid label, collision, non-terminal slice, malformed enum, or reserved name panics.
An invalid external key, source collision, or required environment default also panics.
A secret with no path, a relative path, or different base filenames panics.
