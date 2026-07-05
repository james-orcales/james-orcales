
# Header Renders Time Level Message

A Console renders a flat jlog line as a header of the timestamp, the level, and the message,
in that order, whatever order those keys arrived in on the wire.

# Timestamp Drops Fraction

The header timestamp renders to the second: its fractional-second part is dropped, since a
human reading the console needs no nanosecond granularity, while the JSON line keeps precision.

# Level Is Three Letter Uppercase

A level renders as a three-letter uppercase tag: trace, debug, info, warn, and error become
TRC, DBG, INF, WRN, and ERR.

# Fields Render As Logfmt Pairs

Every key that is not the timestamp, level, or message renders after the message as a
space-separated key=value pair, in the line's emission order.

# String Value Is Quoted Only When Needed

A string field value renders bare when it is a simple token and double-quoted when it is
empty or carries a space, an equals sign, a quote, or a control byte.

# Number Value Is Exact

A numeric field renders its exact digits, so an integer past the float64 safe range is not
rounded.

# Compound Value Renders Compact

An array or embedded-object field value renders as its compacted JSON, so a scalar array
stays one flat token.

# Caller Renders Inline

The caller location renders inline among the fields as caller=file:line, not in the header.

# Error Value Is Red When Colored

The error field renders inline with its value painted red under color, and plain when color
is off.

# Message Is Bold When Colored

The message renders bold under color, and plain when color is off.

# No Color When Disabled

A Console built with color off emits no ANSI escape byte.

# Level None Omits Level

A line with no level field, as Logger_Log writes, renders no level tag.

# Malformed Line Passes Through

A line that is not a JSON object is written through byte for byte, so interleaved non-JSON
output is never lost or mangled.

# Terminal Logger Emits Through Console

New_Terminal_Logger returns a logger whose lines reach standard output in the Console's
human-readable form.
