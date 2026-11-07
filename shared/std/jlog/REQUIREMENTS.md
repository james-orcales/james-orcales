
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

# Nil Logger Is Disabled

A nil Logger emits nothing.

# Hot Path Is Zero Allocation

A steady-state scalar log to a ready stream performs zero heap allocations.

# Construction Is Zero Allocation

New fills caller-owned configuration storage and Logger_With fills the child's own storage;
neither allocates.

# Borrowed Line Drops

A line is submitted from the Logger's own storage, which stays borrowed until the Stream
retires the Write. An emit while it is borrowed is dropped and counted in Logger.Dropped;
the retirement frees the storage. jlog neither waits for nor drives the retirement.

# Oversized Line Drops

A line that outgrows its storage is dropped whole and counted in Logger.Dropped, so the Stream
never sees a torn line; the next line encodes as usual.

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

# Level Filter Drops Below Floor

A Level_Filter forwards only the lines whose level is at or above its floor and drops the rest,
so one logger can feed a verbose sink and a quiet one at once.

# Level Filter Passes Level Less And Non JSON

A Level_Filter passes a line with no level field, as Logger_Log writes, and a line that is not
a JSON object, since it suppresses by level alone and never swallows output it cannot classify.

# Console Logger Tees To Capture

New_Console_Logger renders info and up to the console sink and tees every level as raw JSON to the
capture sink, so one logger drives a quiet console and a full capture from injected writers.

# Transforms Are Zero Allocation

A Console render and a Level_Filter pass over a ready inner stream perform zero heap
allocations: each works in caller-owned storage.

# Oversized Payload Retires Buffer Full

A Write whose transform outgrows its caller-owned storage retires with nbio.Stream_Buffer_Full
and passes nothing to the inner stream.

# Transform Busy Drops Second Write

A Write that arrives while an earlier Write still holds the transform's storage, as two loggers
sharing one Console produce, retires with nbio.Stream_Buffer_Full and leaves the in-flight bytes
untouched.
