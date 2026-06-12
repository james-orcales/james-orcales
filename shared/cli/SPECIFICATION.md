
# Parse

Program_Parse resolves the active command and populates its options, returning an
error on malformed input. A flag carries a default value; a positional argument has
none and may also be given by position, without its label.

### Commands

A named command resolves to itself, no command defaults to the first declared
command, and an unrecognized name returns an error.

### Single Command

A program built with New_Single has no command selector: the first token is the
first positional argument, so a name that would select a sibling command is read as
a positional. Help drops the selector and shows the program's own positionals.

### Arguments

Positionals fill the command's arguments in declaration order, skipping any already
set by name; an integer argument must convert or the parse returns an error.

### Named

Every argument and flag is also settable by -label=value, and named and positional
tokens may appear in any order. Setting a scalar option twice, or naming an option
the command does not declare, returns an error.

### Variadic

A command's last argument may be a slice, collecting the positionals left after the
scalar arguments and appending each repeated -label=value; a slice declared before
the last argument is rejected at construction.

### Flags

A flag assigns its value by type; a double-dash flag or a non-boolean flag without a
value returns an error.

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
that collides with another option's name.
