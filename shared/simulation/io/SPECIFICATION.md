
# Stream

A Stream is a byte transport behind one procedure, a port of Odin core:io. One procedure
pointer covers ten modes, thus the value stays two words. A Stream moves memory only, because
a transport that waits on the world is an IO operation with a Completion.

### Read

Read moves bytes from the cursor into the buffer and advances the cursor. A read at the end
of the memory reports Stream_EOF.

### Write

Write stores the buffer at the cursor and advances the cursor. A write that reaches the end
stores the bytes that fit and reports Stream_Short_Write. It does not increase the memory.

### Read At

Read_At reads at the given offset. It does not move the cursor.

### Write At

Write_At stores at the given offset. It does not move the cursor.

### Seek

Seek moves the cursor to an offset from the start, the cursor, or the end. An origin outside
the three reports Stream_Invalid_Whence. A position outside the memory reports
Stream_Invalid_Offset, because a silent move hides an arithmetic error.

### Size

Size reports the whole memory, not the bytes after the cursor.

### Query

Query reports each mode the stream answers, as one bit for each mode. A caller thus learns
what a stream cannot do, and does not learn it from a failure.

### Flush

Flush pushes buffered bytes onward. Memory holds no buffer, thus its flush succeeds and does
nothing.

### Close

Close ends the stream. It is idempotent, and a closed stream answers no data mode.

### Destroy

Destroy releases what the stream holds. A memory stream holds only its cursor, thus destroy
is a close. The mode exists because an Odin stream can hold an allocation.

### Errors

A zero Stream, a mode that the stream does not answer, and a closed stream each report
Stream_Empty. A count below zero, above the buffer, or below the buffer with no error reports
Stream_Negative_Read, Stream_Invalid_Write, or Stream_Short_Write.

### Memory

A memory stream answers each of the ten modes, because memory is the only transport that
stores. The caller supplies the slice and the state. The slice is the whole budget, thus the
stream allocates nothing.

### Discard

A discard stream absorbs each write and reports the whole buffer stored. It answers no read
mode and no seek mode.

### Limit

A limit passes bytes to the stream behind it until its budget is empty. The bytes after the
budget do not reach that stream. Reads then report Stream_EOF and writes report
Stream_Short_Write.

### Count

A count passes each mode to the stream behind it and adds each byte to its tally. Over a
discard stream it measures what an encoder makes. Over a memory stream it measures what an
encoder made.

### Tee

A tee writes each buffer to two streams and reports the smaller count. The smaller count
tells the caller about the more narrow of the two destinations.

### Composition

The transforms compose. One encoder writes through a tee, a count, a limit, and memory. The
tally, the truncation, and the stored bytes agree.

### Derived

The derived functions are the Odin util set: Read_At_Least, Write_String,
Read_Byte, Write_Byte, Read_Rune, and Write_Rune. An end found after some bytes reports
Stream_Unexpected_EOF, because the caller asked for a quantity and received a part of one.

### Pointer

Read_Pointer and Write_Pointer move raw memory of a size that the caller states. A size below
zero reports Stream_Negative_Count.

### Read Full

Read_Full submits one asynchronous read per pass and rearms at the new offset on a short
count, delivering one callback with the full buffer or the first error. A pass with no bytes
delivers the count so far, so an exhausted file cannot spin the rearm loop.

### Write Full

Write_Full submits one asynchronous write per pass and rearms at the new offset on a short
count, delivering one callback with the whole buffer written or the first error. A pass with
no bytes delivers the count so far.
