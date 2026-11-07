
# Constants

Wire offsets derive cumulatively from named field widths. Block, footer, metadata, timestamp, and
workspace bounds derive from wire layout or shared unit and machine-width constants.

# Stream

Reader and Writer retain one injected `nbio.Stream`. Each operation submits work and retires
through the caller callback. Package code never owns or runs an `io.Driver`. A stream may retire
inline or later; both paths produce the same state transition.

# Storage

Reader_Storage owns one BLOCK_SIZE header block and bounded extension metadata. Writer_Init borrows
Writer_Header_Storage for header, padding, and footer staging. Header text uses Header_Storage. No
operation grows storage.

# Formats

Reader accepts V7, USTAR, PAX, GNU, and STAR headers. Local PAX records and GNU long fields resolve
before one logical header escapes. Writer emits V7, USTAR, PAX, or GNU.
Unknown selects USTAR when fields fit and PAX otherwise. STAR and sparse writing remain unsupported.

# Bounds

One archive, entry, or content operation covers at most ARCHIVE_SIZE_MAXIMUM. One PAX payload or
GNU long field covers at most SPECIAL_FILE_SIZE_MAXIMUM. Every wire size and cursor validates
before access. Short storage returns Status.

# Reader

Reader_Next discards unread content and padding through Stream before reading next logical header.
Reader_Read submits at most requested content. STATUS_END needs zero returned bytes.
Extension metadata remains valid until next Reader_Next call.

# Writer

Writer_Write_Header requires prior content complete, stages a complete header sequence, then
submits it. Writer_Write rejects content beyond declared Size. Writer_Close requires content
complete, submits padding plus ARCHIVE_FOOTER_SIZE, and reports total archive bytes.

# Allocation

Every exported operation performs zero heap allocation. Caller state owns streams, completions,
callbacks, headers, metadata, and wire buffers. Production code contains no `append`, `make`,
`new`, or mutable package state.

# Untrusted Input

Checksums, numeric fields, PAX records, sparse extensions, type sizes, padding, field storage,
stream counts, and every offset validate before use. Malformed bytes return Status. Transport
failures remain in `Completion.Error`.
