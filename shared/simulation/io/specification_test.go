package io_test

import (
	"testing"

	"local/james-orcales/shared/simulation/io"
)

// Test_Stream_Read verifies Read moves bytes from the cursor, advances the cursor, and
// reports Stream_EOF once the cursor has reached the end of the memory.
func Test_Stream_Read(t *testing.T) {
	stream := memory_stream([]byte("abcdef"))
	first := make([]byte, 3)
	count, err := io.Read(stream, first)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if count != 3 {
		t.Fatalf("read %d bytes, want 3", count)
	}
	if string(first) != "abc" {
		t.Fatalf("read %q, want abc", first)
	}
	second := make([]byte, 8)
	count, err = io.Read(stream, second)
	if err != nil {
		t.Fatalf("second read error: %v", err)
	}
	if count != 3 {
		t.Fatalf("second read %d bytes, want 3", count)
	}
	_, err = io.Read(stream, second)
	if err != io.Stream_EOF {
		t.Fatalf("exhausted read reported %v, want Stream_EOF", err)
	}
}

// Test_Stream_Write verifies Write stores bytes at the cursor, advances the cursor, and
// reports Stream_Short_Write when the remaining memory cannot hold the whole buffer.
func Test_Stream_Write(t *testing.T) {
	memory := make([]byte, 4)
	stream := memory_stream(memory)
	count, err := io.Write(stream, []byte("ab"))
	if err != nil {
		t.Fatalf("write error: %v", err)
	}
	if count != 2 {
		t.Fatalf("wrote %d bytes, want 2", count)
	}
	count, err = io.Write(stream, []byte("cdef"))
	if err != io.Stream_Short_Write {
		t.Fatalf("overflowing write reported %v, want Stream_Short_Write", err)
	}
	if count != 2 {
		t.Fatalf("overflowing write stored %d bytes, want 2", count)
	}
	if string(memory) != "abcd" {
		t.Fatalf("memory holds %q, want abcd", memory)
	}
}

// Test_Stream_Read_At verifies Read_At reads from an explicit offset and leaves the cursor
// where it was, the distinction Odin draws between Read and Read_At.
func Test_Stream_Read_At(t *testing.T) {
	stream := memory_stream([]byte("abcdef"))
	head := make([]byte, 2)
	_, read_err := io.Read(stream, head)
	if read_err != nil {
		t.Fatalf("read error: %v", read_err)
	}
	tail := make([]byte, 2)
	count, err := io.Read_At(stream, tail, 4)
	if err != nil {
		t.Fatalf("read at error: %v", err)
	}
	if count != 2 {
		t.Fatalf("read at moved %d bytes, want 2", count)
	}
	if string(tail) != "ef" {
		t.Fatalf("read at yielded %q, want ef", tail)
	}
	position, seek_err := io.Seek(stream, 0, io.SEEK_FROM_CURRENT)
	if seek_err != nil {
		t.Fatalf("seek error: %v", seek_err)
	}
	if position != 2 {
		t.Fatalf("read at moved the cursor to %d, want 2", position)
	}
}

// Test_Stream_Write_At verifies Write_At stores at an explicit offset and leaves the cursor
// where it was.
func Test_Stream_Write_At(t *testing.T) {
	memory := make([]byte, 6)
	stream := memory_stream(memory)
	count, err := io.Write_At(stream, []byte("xy"), 4)
	if err != nil {
		t.Fatalf("write at error: %v", err)
	}
	if count != 2 {
		t.Fatalf("write at stored %d bytes, want 2", count)
	}
	if memory[4] != 'x' {
		t.Fatalf("write at stored %q at 4, want x", memory[4])
	}
	position, seek_err := io.Seek(stream, 0, io.SEEK_FROM_CURRENT)
	if seek_err != nil {
		t.Fatalf("seek error: %v", seek_err)
	}
	if position != 0 {
		t.Fatalf("write at moved the cursor to %d, want 0", position)
	}
}

// Test_Stream_Seek verifies each Seek_From origin, and that an origin outside the three
// reports Stream_Invalid_Whence.
func Test_Stream_Seek(t *testing.T) {
	stream := memory_stream([]byte("abcdef"))
	position, err := io.Seek(stream, 2, io.SEEK_FROM_START)
	if err != nil {
		t.Fatalf("seek from start error: %v", err)
	}
	if position != 2 {
		t.Fatalf("seek from start reached %d, want 2", position)
	}
	position, err = io.Seek(stream, 1, io.SEEK_FROM_CURRENT)
	if err != nil {
		t.Fatalf("seek from current error: %v", err)
	}
	if position != 3 {
		t.Fatalf("seek from current reached %d, want 3", position)
	}
	position, err = io.Seek(stream, -1, io.SEEK_FROM_END)
	if err != nil {
		t.Fatalf("seek from end error: %v", err)
	}
	if position != 5 {
		t.Fatalf("seek from end reached %d, want 5", position)
	}
	_, err = io.Seek(stream, 0, io.Seek_From(9))
	if err != io.Stream_Invalid_Whence {
		t.Fatalf("unknown whence reported %v, want Stream_Invalid_Whence", err)
	}
	_, err = io.Seek(stream, -1, io.SEEK_FROM_START)
	if err != io.Stream_Invalid_Offset {
		t.Fatalf("negative position reported %v, want Stream_Invalid_Offset", err)
	}
}

// Test_Stream_Size verifies Size reports the whole memory, not the bytes remaining.
func Test_Stream_Size(t *testing.T) {
	stream := memory_stream([]byte("abcdef"))
	_, seek_err := io.Seek(stream, 4, io.SEEK_FROM_START)
	if seek_err != nil {
		t.Fatalf("seek error: %v", seek_err)
	}
	size, err := io.Size(stream)
	if err != nil {
		t.Fatalf("size error: %v", err)
	}
	if size != 6 {
		t.Fatalf("size reported %d, want 6", size)
	}
}

// Test_Stream_Query verifies Query names exactly the modes a stream answers, so a caller
// learns what a stream cannot do without provoking a failure.
func Test_Stream_Query(t *testing.T) {
	memory := io.Query(memory_stream(make([]byte, 4)))
	if !io.Mode_Set_Has(memory, io.STREAM_MODE_SEEK) {
		t.Fatal("a memory stream answers Seek")
	}
	if !io.Mode_Set_Has(memory, io.STREAM_MODE_SIZE) {
		t.Fatal("a memory stream answers Size")
	}
	discard := io.Query(discard_stream())
	if io.Mode_Set_Has(discard, io.STREAM_MODE_SEEK) {
		t.Fatal("a discard stream cannot seek")
	}
	if !io.Mode_Set_Has(discard, io.STREAM_MODE_WRITE) {
		t.Fatal("a discard stream answers Write")
	}
}

// Test_Stream_Flush verifies Flush succeeds on memory, which has nothing to flush, so a
// caller can flush any stream without asking what is behind it.
func Test_Stream_Flush(t *testing.T) {
	if err := io.Flush(memory_stream(make([]byte, 2))); err != nil {
		t.Fatalf("flush error: %v", err)
	}
	if err := io.Flush(discard_stream()); err != nil {
		t.Fatalf("discard flush error: %v", err)
	}
}

// Test_Stream_Close verifies Close is idempotent and that a closed stream answers no data
// mode.
func Test_Stream_Close(t *testing.T) {
	stream := memory_stream(make([]byte, 4))
	if err := io.Close(stream); err != nil {
		t.Fatalf("close error: %v", err)
	}
	if err := io.Close(stream); err != nil {
		t.Fatalf("second close error: %v", err)
	}
	_, err := io.Write(stream, []byte("a"))
	if err != io.Stream_Empty {
		t.Fatalf("write after close reported %v, want Stream_Empty", err)
	}
}

// Test_Stream_Destroy verifies Destroy closes the stream. Odin separates the two because a
// stream there can own an allocation; a memory stream owns nothing but its cursor.
func Test_Stream_Destroy(t *testing.T) {
	stream := memory_stream(make([]byte, 4))
	if err := io.Destroy(stream); err != nil {
		t.Fatalf("destroy error: %v", err)
	}
	_, err := io.Read(stream, make([]byte, 1))
	if err != io.Stream_Empty {
		t.Fatalf("read after destroy reported %v, want Stream_Empty", err)
	}
}

// Test_Stream_Errors verifies the dispatch checks Odin's write helper performs: a zero
// Stream reports Stream_Empty, and a mode a stream does not answer reports Stream_Empty.
func Test_Stream_Errors(t *testing.T) {
	var zero io.Stream
	_, err := io.Read(zero, make([]byte, 1))
	if err != io.Stream_Empty {
		t.Fatalf("zero stream read reported %v, want Stream_Empty", err)
	}
	if io.Query(zero) != 0 {
		t.Fatal("a zero stream answers no mode")
	}
	_, err = io.Seek(discard_stream(), 0, io.SEEK_FROM_START)
	if err != io.Stream_Empty {
		t.Fatalf("unsupported seek reported %v, want Stream_Empty", err)
	}
}

// Test_Stream_Memory verifies the memory stream never grows its slice: it is bounded by the
// slice it was built over, which is what makes it safe to hand to an unbounded encoder.
func Test_Stream_Memory(t *testing.T) {
	memory := make([]byte, 3)
	state := io.Stream_Memory{Memory: memory}
	stream := io.Memory_To_Stream(&state)
	count, err := io.Write(stream, []byte("abcdefgh"))
	if err != io.Stream_Short_Write {
		t.Fatalf("write past the end reported %v, want Stream_Short_Write", err)
	}
	if count != 3 {
		t.Fatalf("write past the end stored %d bytes, want 3", count)
	}
	if len(memory) != 3 {
		t.Fatalf("the memory grew to %d bytes", len(memory))
	}
	if string(memory) != "abc" {
		t.Fatalf("memory holds %q, want abc", memory)
	}
}

// Test_Stream_Discard verifies a discard stream absorbs every write, reports the whole buffer
// stored, and answers no read mode.
func Test_Stream_Discard(t *testing.T) {
	stream := discard_stream()
	count, err := io.Write(stream, []byte("abcdef"))
	if err != nil {
		t.Fatalf("discard write error: %v", err)
	}
	if count != 6 {
		t.Fatalf("discard reported %d bytes, want 6", count)
	}
	_, err = io.Read(stream, make([]byte, 4))
	if err != io.Stream_Empty {
		t.Fatalf("discard read reported %v, want Stream_Empty", err)
	}
}

// Test_Stream_Limit verifies a limit truncates at its budget and stops the bytes reaching the
// stream behind it, so the budget is a fact about the transport and not a caller convention.
func Test_Stream_Limit(t *testing.T) {
	memory := make([]byte, 8)
	inner := memory_stream(memory)
	state := io.Stream_Limit{Inner: inner, Budget: 3}
	stream := io.Limit_To_Stream(&state)
	count, err := io.Write(stream, []byte("abcde"))
	if err != io.Stream_Short_Write {
		t.Fatalf("write past the budget reported %v, want Stream_Short_Write", err)
	}
	if count != 3 {
		t.Fatalf("write past the budget passed %d bytes, want 3", count)
	}
	if string(memory[:4]) != "abc\x00" {
		t.Fatalf("the memory behind the limit holds %q, want abc and a zero", memory[:4])
	}
	_, err = io.Write(stream, []byte("f"))
	if err != io.Stream_Short_Write {
		t.Fatalf("write on a spent budget reported %v, want Stream_Short_Write", err)
	}
	if io.Mode_Set_Has(io.Query(stream), io.STREAM_MODE_SEEK) {
		t.Fatal("a limited stream cannot seek: a budget and a cursor disagree")
	}
}

// Test_Stream_Count verifies a count tallies every byte and changes nothing else, so the same
// encoder measures and stores without being told which it is doing.
func Test_Stream_Count(t *testing.T) {
	memory := make([]byte, 8)
	state := io.Stream_Count{Inner: memory_stream(memory)}
	stream := io.Count_To_Stream(&state)
	if _, err := io.Write(stream, []byte("ab")); err != nil {
		t.Fatalf("write error: %v", err)
	}
	if _, err := io.Write(stream, []byte("cde")); err != nil {
		t.Fatalf("second write error: %v", err)
	}
	if state.Tally != 5 {
		t.Fatalf("the tally is %d, want 5", state.Tally)
	}
	if string(memory[:5]) != "abcde" {
		t.Fatalf("the memory behind the count holds %q, want abcde", memory[:5])
	}
	measure := io.Stream_Count{Inner: discard_stream()}
	if _, err := io.Write(io.Count_To_Stream(&measure), []byte("abcd")); err != nil {
		t.Fatalf("measuring write error: %v", err)
	}
	if measure.Tally != 4 {
		t.Fatalf("the measuring tally is %d, want 4", measure.Tally)
	}
}

// Test_Stream_Tee verifies a tee writes each buffer to both streams and reports the smaller
// count, so a caller learns about the tighter of the two rather than the first.
func Test_Stream_Tee(t *testing.T) {
	wide := make([]byte, 8)
	narrow := make([]byte, 2)
	state := io.Stream_Tee{First: memory_stream(wide), Second: memory_stream(narrow)}
	stream := io.Tee_To_Stream(&state)
	count, err := io.Write(stream, []byte("abcd"))
	if err != io.Stream_Short_Write {
		t.Fatalf("tee onto a narrow stream reported %v, want Stream_Short_Write", err)
	}
	if count != 2 {
		t.Fatalf("the tee reported %d bytes, want the narrower 2", count)
	}
	if string(wide[:4]) != "abcd" {
		t.Fatalf("the wide stream holds %q, want abcd", wide[:4])
	}
	if string(narrow) != "ab" {
		t.Fatalf("the narrow stream holds %q, want ab", narrow)
	}
	if io.Mode_Set_Has(io.Query(stream), io.STREAM_MODE_READ) {
		t.Fatal("a tee cannot read: two streams give two answers")
	}
}

// Test_Stream_Composition verifies the transforms compose. One encoder writes through a tee,
// over a count, over a limit, over memory, and the tally, the truncation, and the stored bytes
// all agree — the property that makes the abstraction worth its indirection.
func Test_Stream_Composition(t *testing.T) {
	stored := make([]byte, 16)
	limit := io.Stream_Limit{Inner: memory_stream(stored), Budget: 6}
	count := io.Stream_Count{Inner: io.Limit_To_Stream(&limit)}
	audit := make([]byte, 16)
	tee := io.Stream_Tee{First: io.Count_To_Stream(&count), Second: memory_stream(audit)}
	stream := io.Tee_To_Stream(&tee)

	written, err := io.Write(stream, []byte("abcdefghij"))
	if err != io.Stream_Short_Write {
		t.Fatalf("composed write reported %v, want Stream_Short_Write", err)
	}
	if written != 6 {
		t.Fatalf("composed write reported %d bytes, want the limit's 6", written)
	}
	if count.Tally != 6 {
		t.Fatalf("the tally is %d, want 6", count.Tally)
	}
	if string(stored[:8]) != "abcdef\x00\x00" {
		t.Fatalf("the stored bytes are %q, want abcdef and two zeros", stored[:8])
	}
	if string(audit[:10]) != "abcdefghij" {
		t.Fatalf("the audit copy holds %q, want abcdefghij", audit[:10])
	}
}

// Builds a memory stream over bytes the test owns. Test_Stream_Memory writes the constructor
// out in full; every other test uses this, because the state is not what it is testing.
func memory_stream(memory []byte) (stream io.Stream) {
	state := io.Stream_Memory{Memory: memory}
	return io.Memory_To_Stream(&state)
}

// Builds a discard stream for the tests that need a stream answering only write modes.
func discard_stream() (stream io.Stream) {
	state := io.Stream_Discard{}
	return io.Discard_To_Stream(&state)
}
