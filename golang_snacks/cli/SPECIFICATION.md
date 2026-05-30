
# Parse

Program_Parse resolves the active command from the arguments and populates its
options, or returns an error describing the malformed input.

### Commands

A named command resolves to itself, no command defaults to the first declared
command, and an unrecognized name returns an error.

### Arguments

The positional count must match the command's arguments, and an integer argument
must convert or the parse returns an error.

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
