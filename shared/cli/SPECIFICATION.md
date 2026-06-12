
# Parse

Program_Parse resolves the active command from the arguments and populates its
options, or returns an error describing the malformed input.

### Commands

A named command resolves to itself, no command defaults to the first declared
command, and an unrecognized name returns an error.

### Single Command

A program built with New_Single has no command selector: the first token is the
first positional argument, so a name that would select a sibling command is read as
a positional. Help drops the selector and shows the program's own positionals.

### Arguments

The positional count must match the command's arguments, and an integer argument
must convert or the parse returns an error.

### Variadic

A command's last argument may be variadic, collecting zero or more trailing
positionals into a slice; one declared elsewhere is rejected at construction. The
fixed arguments before it set the minimum, and each element converts by its type.

### Flags

A flag assigns its value by type; an unknown flag, a double-dash flag, or a
non-boolean flag without a value returns an error.

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

A command without a label panics during construction.
