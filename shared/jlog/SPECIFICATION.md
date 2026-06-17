
# Message Renders Level And Message

An emit call writes the level field, then the caller's fields, then the message
field last, closing the object with a newline.

# Empty Message Is Omitted

An empty message string produces no message field.

# Scalar Fields

String, Boolean, Integer, Uint64, and Float64 fields render as their JSON scalar
forms, in call order, before the message.

# String Escape Sequences

String values escape JSON control characters, quotes, and backslashes.

# Bytes Hexadecimal Raw JSON

Bytes render as a JSON string, Hexadecimal as a hex string, and Raw_JSON copies
pre-encoded bytes verbatim.

# Timestamp Uses Injected Clock

The Timestamp field renders the injected clock's realtime reading as an RFC 3339
UTC timestamp string.

# Auto Timestamp

A logger built with Auto_Timestamp stamps every line without an explicit field.

# Time And Duration

A Moment renders as an RFC 3339 UTC timestamp string; a Duration renders as an
integer in the configured unit.

# Network Fields

IP_Address and MAC_Address render via their stdlib string forms.

# Scalar Arrays

Strings and Integers render as flat JSON arrays referencing the caller's slice.

# Err With Stack

When a stack marshaler is configured, Err writes the rendered stack before the
error string.

# Err Without Stack

Without a stack marshaler, Err writes only the error string.

# Caller Uses Injected Function

The Caller field renders the location returned by the injected caller lookup.

# Child Logger Carries Context

Logger_With returns a child whose fixed prefix fields precede each line's own
fields.

# Level Floor Filters

A line below the logger's floor produces no output.

# From Context Round Trips

A logger carried by Logger_With_Context is recovered by From_Context.

# From Context Missing Is Disabled

From_Context on a context with no logger returns a disabled no-op logger.

# Hot Path Is Zero Allocation

A steady-state scalar log to a ready writer performs zero heap allocations.
