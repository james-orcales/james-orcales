// Package io is the synchronous byte transport used by the repository — a port of Odin
// core:io.
//
// FAITHFUL PORT: its Stream modes, its derived functions, and its error set follow Odin's
// core/io/stream.odin and core/io/util.odin. DO NOT DIVERGE.
//
// MEMORY ONLY. A Stream moves bytes that are already in this process. A transport that waits
// on the world is an asynchronous operation with a Completion, and it belongs in shared/nbio.
// Read_Full and Write_Full are the two deliberate exceptions: they are not Streams but
// derived functions composed above nbio's asynchronous Read and Write, taking a caller-owned
// completion and delivering one callback, so a caller moves a whole buffer with one call and
// the root's pump supplies the waiting.
package io

import (
	"errors"
	"unicode/utf8"
	"unsafe"

	invariant "local/james-orcales/shared/invariant/default"
	nbio "local/james-orcales/shared/simulation/nbio"
	"local/james-orcales/shared/simulation/time"
)

// Stream is a byte transport behind one procedure — a port of Odin core:io's Stream
// (core/io/stream.odin). One procedure pointer covers ten modes, so the value stays two words
// however many modes exist, and a caller names one type instead of ten vtable slots.
//
// MEMORY ONLY. A Stream moves bytes that are already in this process. It never waits on the
// world, and no Stream may ever be built over a file, a socket, a pipe, or a subprocess.
//
// This is not a limitation of the port; it is the difference between the two languages. Odin
// has os.stream_from_handle because core:os makes a blocking syscall, and Odin's timeline is
// whatever the kernel decides. This repository made the timeline an injected dependency: a file
// read is IO.Read with a Completion, ordered by the driver the composition root holds. A
// synchronous file read below that root would advance nothing, order nothing, and could not be
// faulted by the simulator — README.md section 2 states why that is an architectural bug and
// not a style preference. There is therefore nothing to build a file stream out of, because
// IO.Read and IO.Write are the only reads and writes there are, and both are asynchronous.
//
// The rule for anyone adding an implementation: if it waits on anything outside this process,
// it is not a Stream — it is an IO operation, and it belongs on the IO vtable with a Completion.
// If it only transforms bytes on their way through, it is a Stream.
//
// TigerBeetle draws the same boundary, and draws it twice. Its byte path erases nothing:
// third-party/tigerbeetle/src/storage.zig:12 is StorageType(comptime IO: type), monomorphized
// per backend, and its surface is read_sectors and write_sectors (storage.zig:215,386), each
// taking a caller-owned completion struct that holds an IO.Completion, the buffer, the offset,
// and the callback (storage.zig:22-40). No mode enum, no data pointer — which is what IO here
// already is. Its type-erased two-word writer, std.io.AnyWriter, appears only in inspect.zig,
// benchmark_load.zig, trace.zig:164, and snaptest.zig:340: diagnostics, dumps, and text. So a
// type-erased byte sink is for bytes whose order nobody replays, and the path that must stay
// deterministic gets static dispatch and explicit completions. This Stream is the first kind.
//
// That boundary is what the type is for. Its implementations are transforms, not destinations:
// Stream_Memory stores, Stream_Discard absorbs, Stream_Limit truncates, Stream_Count tallies,
// Stream_Tee forks. An encoder written once against Stream works against every one of them, and
// against every transform added later, none of which can smuggle a syscall in behind it.
type Stream struct {
	// Procedure runs every mode. A zero Stream has none, and reports Stream_Empty.
	Procedure Stream_Procedure
	// Data is the stream's own state. Its own Procedure asserts the concrete type.
	Data any
}

// Stream_Procedure runs one mode — Odin's Stream_Proc. One signature serves every mode, so
// count carries the byte count, the new position, the size, or the mode set, as the mode
// dictates, and the arguments a mode does not use are ignored.
type Stream_Procedure func(
	data any, mode Stream_Mode, buffer []byte, offset int64, whence Seek_From,
) (count int64, err error)

// Stream_Mode selects the operation a Stream_Procedure performs.
type Stream_Mode int

// STREAM_MODE_CLOSE ends the stream. It is idempotent.
const STREAM_MODE_CLOSE Stream_Mode = 0

// STREAM_MODE_FLUSH pushes buffered bytes to whatever is behind the stream.
const STREAM_MODE_FLUSH Stream_Mode = 1

// STREAM_MODE_READ reads at the cursor and advances it.
const STREAM_MODE_READ Stream_Mode = 2

// STREAM_MODE_READ_AT reads at an explicit offset and leaves the cursor.
const STREAM_MODE_READ_AT Stream_Mode = 3

// STREAM_MODE_WRITE writes at the cursor and advances it.
const STREAM_MODE_WRITE Stream_Mode = 4

// STREAM_MODE_WRITE_AT writes at an explicit offset and leaves the cursor.
const STREAM_MODE_WRITE_AT Stream_Mode = 5

// STREAM_MODE_SEEK moves the cursor.
const STREAM_MODE_SEEK Stream_Mode = 6

// STREAM_MODE_SIZE reports the whole size, not the bytes remaining.
const STREAM_MODE_SIZE Stream_Mode = 7

// STREAM_MODE_DESTROY releases what the stream owns. Odin separates it from Close because a
// stream there can hold an allocation.
const STREAM_MODE_DESTROY Stream_Mode = 8

// STREAM_MODE_QUERY reports the mode set, so a caller learns what a stream cannot do without
// provoking a failure.
const STREAM_MODE_QUERY Stream_Mode = 9

// Stream_Mode_Set names a set of modes, one bit per mode — Odin's bit_set.
type Stream_Mode_Set uint64

// Mode_Set_Add returns the set with mode added.
func Mode_Set_Add(modes Stream_Mode_Set, mode Stream_Mode) (extended Stream_Mode_Set) {
	return modes | 1<<uint(mode)
}

// Mode_Set_Has reports whether modes contains mode.
func Mode_Set_Has(modes Stream_Mode_Set, mode Stream_Mode) (present bool) {
	return modes&(1<<uint(mode)) != 0
}

// Seek_From selects the origin a seek offset is measured from.
type Seek_From int

// SEEK_FROM_START measures from the first byte.
const SEEK_FROM_START Seek_From = 0

// SEEK_FROM_CURRENT measures from the cursor.
const SEEK_FROM_CURRENT Seek_From = 1

// SEEK_FROM_END measures from one past the last byte.
const SEEK_FROM_END Seek_From = 2

// Stream_EOF reports that a read found nothing left. It is deliberately distinct from the
// standard library io.EOF: a Stream is not a stdlib reader, and one value must not stand for
// both contracts.
var Stream_EOF = errors.New("io: stream end of file")

// Stream_Unexpected_EOF reports an end reached partway through a value that needed more bytes.
var Stream_Unexpected_EOF = errors.New("io: stream unexpected end of file")

// Stream_Short_Write reports that a write stored fewer bytes than it was given.
var Stream_Short_Write = errors.New("io: stream short write")

// Stream_Invalid_Write reports a procedure that claimed more bytes than it was given.
var Stream_Invalid_Write = errors.New("io: stream invalid write")

// Stream_Short_Buffer reports a buffer too small to hold what the stream produced.
var Stream_Short_Buffer = errors.New("io: stream short buffer")

// Stream_No_Progress reports a read or write that moved no bytes and reported no reason.
var Stream_No_Progress = errors.New("io: stream made no progress")

// Stream_Invalid_Whence reports a seek origin outside the three Seek_From values.
var Stream_Invalid_Whence = errors.New("io: stream invalid whence")

// Stream_Invalid_Offset reports an offset outside the stream.
var Stream_Invalid_Offset = errors.New("io: stream invalid offset")

// Stream_Invalid_Unread reports an unread of a byte the stream did not read.
var Stream_Invalid_Unread = errors.New("io: stream invalid unread")

// Stream_Negative_Read reports a procedure that returned a negative read count.
var Stream_Negative_Read = errors.New("io: stream negative read")

// Stream_Negative_Write reports a procedure that returned a negative write count.
var Stream_Negative_Write = errors.New("io: stream negative write")

// Stream_Negative_Count reports a negative count where only a positive one is meaningful.
var Stream_Negative_Count = errors.New("io: stream negative count")

// Stream_Buffer_Full reports a buffer that cannot accept another byte.
var Stream_Buffer_Full = errors.New("io: stream buffer full")

// Stream_Unknown reports a failure the stream cannot name.
var Stream_Unknown = errors.New("io: stream unknown error")

// Stream_Empty reports a zero Stream, a mode the stream does not answer, and a stream that has
// been closed. All three mean the same thing to a caller: there is nothing there to do this.
var Stream_Empty = errors.New("io: stream empty")

// Read reads into buffer at the cursor and advances the cursor.
func Read(stream Stream, buffer []byte) (count int64, err error) {
	if stream.Procedure == nil {
		return 0, Stream_Empty
	}
	count, err = stream.Procedure(stream.Data, STREAM_MODE_READ, buffer, 0, SEEK_FROM_START)
	return stream_read_checked(count, err, buffer)
}

// Read_At reads into buffer at offset and leaves the cursor where it was.
func Read_At(stream Stream, buffer []byte, offset int64) (count int64, err error) {
	if stream.Procedure == nil {
		return 0, Stream_Empty
	}
	count, err = stream.Procedure(
		stream.Data, STREAM_MODE_READ_AT, buffer, offset, SEEK_FROM_START,
	)
	return stream_read_checked(count, err, buffer)
}

// Write writes buffer at the cursor and advances the cursor.
func Write(stream Stream, buffer []byte) (count int64, err error) {
	if stream.Procedure == nil {
		return 0, Stream_Empty
	}
	count, err = stream.Procedure(stream.Data, STREAM_MODE_WRITE, buffer, 0, SEEK_FROM_START)
	return stream_write_checked(count, err, buffer)
}

// Write_At writes buffer at offset and leaves the cursor where it was.
func Write_At(stream Stream, buffer []byte, offset int64) (count int64, err error) {
	if stream.Procedure == nil {
		return 0, Stream_Empty
	}
	count, err = stream.Procedure(
		stream.Data, STREAM_MODE_WRITE_AT, buffer, offset, SEEK_FROM_START,
	)
	return stream_write_checked(count, err, buffer)
}

// Seek moves the cursor to offset measured from whence and reports the new position.
func Seek(stream Stream, offset int64, whence Seek_From) (position int64, err error) {
	if stream.Procedure == nil {
		return 0, Stream_Empty
	}
	return stream.Procedure(stream.Data, STREAM_MODE_SEEK, nil, offset, whence)
}

// Size reports the whole size of the stream, not the bytes remaining after the cursor.
func Size(stream Stream) (size int64, err error) {
	if stream.Procedure == nil {
		return 0, Stream_Empty
	}
	return stream.Procedure(stream.Data, STREAM_MODE_SIZE, nil, 0, SEEK_FROM_START)
}

// Flush pushes any buffered bytes onward.
func Flush(stream Stream) (err error) {
	if stream.Procedure == nil {
		return Stream_Empty
	}
	_, flush_err := stream.Procedure(stream.Data, STREAM_MODE_FLUSH, nil, 0, SEEK_FROM_START)
	return flush_err
}

// Close ends the stream. It is idempotent, and a closed stream answers no data mode.
func Close(stream Stream) (err error) {
	if stream.Procedure == nil {
		return Stream_Empty
	}
	_, close_err := stream.Procedure(stream.Data, STREAM_MODE_CLOSE, nil, 0, SEEK_FROM_START)
	return close_err
}

// Destroy releases what the stream owns. A memory stream owns nothing but its cursor, so this
// is a close; the mode exists because Odin streams can hold an allocation.
func Destroy(stream Stream) (err error) {
	if stream.Procedure == nil {
		return Stream_Empty
	}
	_, destroy_err := stream.Procedure(
		stream.Data, STREAM_MODE_DESTROY, nil, 0, SEEK_FROM_START,
	)
	return destroy_err
}

// Query reports every mode the stream answers. A zero Stream answers none.
func Query(stream Stream) (modes Stream_Mode_Set) {
	if stream.Procedure == nil {
		return 0
	}
	count, err := stream.Procedure(stream.Data, STREAM_MODE_QUERY, nil, 0, SEEK_FROM_START)
	if err != nil {
		return 0
	}
	return Stream_Mode_Set(count)
}

// Applies Odin's read checks to a procedure result, so one wrong procedure cannot corrupt every
// caller that trusted its count.
func stream_read_checked(
	count int64, err error, buffer []byte,
) (checked int64, checked_err error) {
	if count < 0 {
		return 0, Stream_Negative_Read
	}
	if count > int64(len(buffer)) {
		return 0, Stream_Short_Buffer
	}
	return count, err
}

// Applies Odin's write checks: a count past the buffer is the procedure lying, and a short
// count with no error is the caller's silent data loss, so it becomes an error here.
func stream_write_checked(
	count int64, err error, buffer []byte,
) (checked int64, checked_err error) {
	if count < 0 {
		return 0, Stream_Negative_Write
	}
	if count > int64(len(buffer)) {
		return 0, Stream_Invalid_Write
	}
	if err != nil {
		return count, err
	}
	if count < int64(len(buffer)) {
		return count, Stream_Short_Write
	}
	return count, nil
}

// STREAM_MEMORY_MODES names every mode a memory stream answers, which is all ten: memory is
// the only transport that can honestly answer them all, because it is the only one that stores.
const STREAM_MEMORY_MODES Stream_Mode_Set = 1<<STREAM_MODE_CLOSE |
	1<<STREAM_MODE_FLUSH | 1<<STREAM_MODE_READ | 1<<STREAM_MODE_READ_AT |
	1<<STREAM_MODE_WRITE | 1<<STREAM_MODE_WRITE_AT | 1<<STREAM_MODE_SEEK |
	1<<STREAM_MODE_SIZE | 1<<STREAM_MODE_DESTROY | 1<<STREAM_MODE_QUERY

// STREAM_DISCARD_MODES names the modes a discard stream answers. It cannot seek or report a
// size, because it stores nothing to seek within.
const STREAM_DISCARD_MODES Stream_Mode_Set = 1<<STREAM_MODE_CLOSE |
	1<<STREAM_MODE_FLUSH | 1<<STREAM_MODE_WRITE | 1<<STREAM_MODE_WRITE_AT |
	1<<STREAM_MODE_DESTROY | 1<<STREAM_MODE_QUERY

// Stream_Memory is the state behind a memory stream. The caller owns it and passes its address
// to Memory_To_Stream, as Odin's bytes.buffer_to_stream takes a caller-owned Buffer: the stream
// allocates nothing, so a caller can see every byte the transport will ever hold.
type Stream_Memory struct {
	// Memory is the caller's slice. Its length is the stream's whole budget, and the stream
	// never grows it, so an encoder that writes without asking cannot make this allocate.
	Memory []byte
	// Cursor is where Read and Write act. Read_At and Write_At do not move it.
	Cursor int64
	// Closed reports that Close or Destroy ran. A closed stream answers no data mode.
	Closed bool
}

// Memory_To_Stream returns a Stream over the caller's memory. It answers every mode, because
// memory is the only transport that stores, and storing is what Seek and Size need.
//
// A write that reaches the end of the slice stores what fits and reports Stream_Short_Write.
// That is what makes this safe to hand to an encoder that does not know how much it is about to
// produce: the budget is stated once, at the slice, and the stream cannot exceed it.
func Memory_To_Stream(state *Stream_Memory) (stream Stream) {
	invariant.Always(state != nil, "A memory stream has state.")
	return Stream{Procedure: stream_memory_procedure, Data: state}
}

// Runs one mode against memory. Lifecycle first, so a closed stream answers no data mode.
func stream_memory_procedure(
	data any, mode Stream_Mode, buffer []byte, offset int64, whence Seek_From,
) (count int64, err error) {
	state, held := data.(*Stream_Memory)
	invariant.Always(held, "A memory stream procedure receives its own state.")
	if mode == STREAM_MODE_QUERY {
		return int64(STREAM_MEMORY_MODES), nil
	}
	if mode == STREAM_MODE_CLOSE {
		state.Closed = true
		return 0, nil
	}
	if mode == STREAM_MODE_DESTROY {
		state.Closed = true
		return 0, nil
	}
	if state.Closed {
		return 0, Stream_Empty
	}
	return stream_memory_data(state, mode, buffer, offset, whence)
}

// Runs the modes that touch the bytes, on a stream known to be open.
func stream_memory_data(
	state *Stream_Memory, mode Stream_Mode, buffer []byte, offset int64, whence Seek_From,
) (count int64, err error) {
	if mode == STREAM_MODE_FLUSH {
		return 0, nil
	}
	if mode == STREAM_MODE_SIZE {
		return int64(len(state.Memory)), nil
	}
	if mode == STREAM_MODE_SEEK {
		return stream_memory_seek(state, offset, whence)
	}
	if mode == STREAM_MODE_READ {
		moved, read_err := stream_memory_read(state, buffer, state.Cursor)
		state.Cursor = state.Cursor + moved
		return moved, read_err
	}
	if mode == STREAM_MODE_READ_AT {
		return stream_memory_read(state, buffer, offset)
	}
	if mode == STREAM_MODE_WRITE {
		moved, write_err := stream_memory_write(state, buffer, state.Cursor)
		state.Cursor = state.Cursor + moved
		return moved, write_err
	}
	if mode == STREAM_MODE_WRITE_AT {
		return stream_memory_write(state, buffer, offset)
	}
	return 0, Stream_Empty
}

// Copies out of the memory at offset. An offset at the end is the end of the data, not a fault.
func stream_memory_read(
	state *Stream_Memory, buffer []byte, offset int64,
) (count int64, err error) {
	if offset < 0 {
		return 0, Stream_Invalid_Offset
	}
	if offset > int64(len(state.Memory)) {
		return 0, Stream_Invalid_Offset
	}
	if offset == int64(len(state.Memory)) {
		return 0, Stream_EOF
	}
	return int64(copy(buffer, state.Memory[offset:])), nil
}

// Copies into the memory at offset. What does not fit is reported, never grown into.
func stream_memory_write(
	state *Stream_Memory, buffer []byte, offset int64,
) (count int64, err error) {
	if offset < 0 {
		return 0, Stream_Invalid_Offset
	}
	if offset > int64(len(state.Memory)) {
		return 0, Stream_Invalid_Offset
	}
	stored := int64(copy(state.Memory[offset:], buffer))
	if stored < int64(len(buffer)) {
		return stored, Stream_Short_Write
	}
	return stored, nil
}

// Moves the cursor. A position outside the memory is rejected rather than clamped: a stream
// that silently moves a cursor somewhere else hides the caller's arithmetic error.
func stream_memory_seek(
	state *Stream_Memory, offset int64, whence Seek_From,
) (position int64, err error) {
	target := offset
	if whence == SEEK_FROM_CURRENT {
		target = state.Cursor + offset
	}
	if whence == SEEK_FROM_END {
		target = int64(len(state.Memory)) + offset
	}
	if whence > SEEK_FROM_END {
		return 0, Stream_Invalid_Whence
	}
	if whence < SEEK_FROM_START {
		return 0, Stream_Invalid_Whence
	}
	if target < 0 {
		return 0, Stream_Invalid_Offset
	}
	if target > int64(len(state.Memory)) {
		return 0, Stream_Invalid_Offset
	}
	state.Cursor = target
	return target, nil
}

// Stream_Discard is the state behind a discard stream. It keeps nothing, so only the closed
// flag distinguishes one discard stream from another.
type Stream_Discard struct {
	// Closed reports that Close or Destroy ran.
	Closed bool
}

// Discard_To_Stream returns a Stream that absorbs every write and reports the whole buffer
// written. It is the sink an encoder writes to when the caller wants the size or the side
// effects, not the bytes. It answers no read mode: there is nothing there to read.
func Discard_To_Stream(state *Stream_Discard) (stream Stream) {
	invariant.Always(state != nil, "A discard stream has state.")
	return Stream{Procedure: stream_discard_procedure, Data: state}
}

// Runs one mode against nothing.
func stream_discard_procedure(
	data any, mode Stream_Mode, buffer []byte, offset int64, whence Seek_From,
) (count int64, err error) {
	state, held := data.(*Stream_Discard)
	invariant.Always(held, "A discard stream procedure receives its own state.")
	if mode == STREAM_MODE_QUERY {
		return int64(STREAM_DISCARD_MODES), nil
	}
	if mode == STREAM_MODE_CLOSE {
		state.Closed = true
		return 0, nil
	}
	if mode == STREAM_MODE_DESTROY {
		state.Closed = true
		return 0, nil
	}
	if state.Closed {
		return 0, Stream_Empty
	}
	if mode == STREAM_MODE_FLUSH {
		return 0, nil
	}
	if mode == STREAM_MODE_WRITE {
		return int64(len(buffer)), nil
	}
	if mode == STREAM_MODE_WRITE_AT {
		return int64(len(buffer)), nil
	}
	return 0, Stream_Empty
}

// STREAM_LIMIT_MODES names the modes a limit forwards. It drops Read_At, Write_At, Seek, and
// Size deliberately: a budget counts bytes as they pass, and an operation at an explicit offset
// passes no bytes through the budget, so the two ideas cannot both be honest at once.
const STREAM_LIMIT_MODES Stream_Mode_Set = 1<<STREAM_MODE_CLOSE |
	1<<STREAM_MODE_FLUSH | 1<<STREAM_MODE_READ | 1<<STREAM_MODE_WRITE |
	1<<STREAM_MODE_DESTROY | 1<<STREAM_MODE_QUERY

// STREAM_TEE_MODES names the modes a tee forwards. It cannot read, seek, or report a size,
// because two streams would give two answers and a tee has no way to choose between them.
const STREAM_TEE_MODES Stream_Mode_Set = 1<<STREAM_MODE_CLOSE |
	1<<STREAM_MODE_FLUSH | 1<<STREAM_MODE_WRITE | 1<<STREAM_MODE_DESTROY |
	1<<STREAM_MODE_QUERY

// Stream_Limit is the state behind a limit. It narrows a stream that is already bounded, which
// is not how a stream becomes bounded: every constructor states its own budget.
type Stream_Limit struct {
	// Inner is the stream the bytes pass through to.
	Inner Stream
	// Budget is how many bytes may still pass. It falls as they do.
	Budget int64
}

// Limit_To_Stream returns a Stream that passes bytes to Inner until Budget runs out, then
// reports Stream_EOF on reads and Stream_Short_Write on writes.
//
// The bytes past the budget never reach Inner. That is the difference between a limit and a
// caller who counts: a caller who counts can forget, and a limit cannot.
func Limit_To_Stream(state *Stream_Limit) (stream Stream) {
	invariant.Always(state != nil, "A limit stream has state.")
	return Stream{Procedure: stream_limit_procedure, Data: state}
}

// Runs one mode against the budget, then against the stream behind it.
func stream_limit_procedure(
	data any, mode Stream_Mode, buffer []byte, offset int64, whence Seek_From,
) (count int64, err error) {
	state, held := data.(*Stream_Limit)
	invariant.Always(held, "A limit stream procedure receives its own state.")
	if mode == STREAM_MODE_QUERY {
		return int64(Query(state.Inner) & STREAM_LIMIT_MODES), nil
	}
	if mode == STREAM_MODE_CLOSE {
		return 0, Close(state.Inner)
	}
	if mode == STREAM_MODE_DESTROY {
		return 0, Destroy(state.Inner)
	}
	if mode == STREAM_MODE_FLUSH {
		return 0, Flush(state.Inner)
	}
	if mode == STREAM_MODE_READ {
		return stream_limit_read(state, buffer)
	}
	if mode == STREAM_MODE_WRITE {
		return stream_limit_write(state, buffer)
	}
	return 0, Stream_Empty
}

// Reads no more than the budget allows, and spends what it read.
func stream_limit_read(state *Stream_Limit, buffer []byte) (count int64, err error) {
	if state.Budget <= 0 {
		return 0, Stream_EOF
	}
	allowed := int64(len(buffer))
	if state.Budget < allowed {
		allowed = state.Budget
	}
	moved, read_err := Read(state.Inner, buffer[:allowed])
	state.Budget = state.Budget - moved
	return moved, read_err
}

// Writes no more than the budget allows, and reports the truncation the caller cannot see.
func stream_limit_write(state *Stream_Limit, buffer []byte) (count int64, err error) {
	if state.Budget <= 0 {
		return 0, Stream_Short_Write
	}
	allowed := int64(len(buffer))
	if state.Budget < allowed {
		allowed = state.Budget
	}
	moved, write_err := Write(state.Inner, buffer[:allowed])
	state.Budget = state.Budget - moved
	if write_err != nil {
		return moved, write_err
	}
	if allowed < int64(len(buffer)) {
		return moved, Stream_Short_Write
	}
	return moved, nil
}

// Stream_Count is the state behind a count. Tally is the caller's: the stream adds to it and
// never reads it, so the caller decides when a measurement starts and stops.
type Stream_Count struct {
	// Inner is the stream the bytes pass through to.
	Inner Stream
	// Tally is every byte that has passed, read and written alike.
	Tally int64
}

// Count_To_Stream returns a Stream that forwards every mode to Inner and adds each byte to
// Tally. Over a discard stream it measures what an encoder would produce; over memory it
// measures what an encoder did produce. The encoder cannot tell the two apart, which is the
// point of measuring this way rather than running the encoder twice.
func Count_To_Stream(state *Stream_Count) (stream Stream) {
	invariant.Always(state != nil, "A count stream has state.")
	return Stream{Procedure: stream_count_procedure, Data: state}
}

// Runs one mode against the stream behind it and tallies what moved.
func stream_count_procedure(
	data any, mode Stream_Mode, buffer []byte, offset int64, whence Seek_From,
) (count int64, err error) {
	state, held := data.(*Stream_Count)
	invariant.Always(held, "A count stream procedure receives its own state.")
	if mode == STREAM_MODE_QUERY {
		return int64(Query(state.Inner)), nil
	}
	if mode == STREAM_MODE_CLOSE {
		return 0, Close(state.Inner)
	}
	if mode == STREAM_MODE_DESTROY {
		return 0, Destroy(state.Inner)
	}
	if mode == STREAM_MODE_FLUSH {
		return 0, Flush(state.Inner)
	}
	if mode == STREAM_MODE_SEEK {
		return Seek(state.Inner, offset, whence)
	}
	if mode == STREAM_MODE_SIZE {
		return Size(state.Inner)
	}
	return stream_count_moved(state, mode, buffer, offset)
}

// Runs the modes that move bytes and adds each one to the tally.
func stream_count_moved(
	state *Stream_Count, mode Stream_Mode, buffer []byte, offset int64,
) (count int64, err error) {
	if mode == STREAM_MODE_READ {
		moved, read_err := Read(state.Inner, buffer)
		state.Tally = state.Tally + moved
		return moved, read_err
	}
	if mode == STREAM_MODE_READ_AT {
		moved, read_err := Read_At(state.Inner, buffer, offset)
		state.Tally = state.Tally + moved
		return moved, read_err
	}
	if mode == STREAM_MODE_WRITE {
		moved, write_err := Write(state.Inner, buffer)
		state.Tally = state.Tally + moved
		return moved, write_err
	}
	if mode == STREAM_MODE_WRITE_AT {
		moved, write_err := Write_At(state.Inner, buffer, offset)
		state.Tally = state.Tally + moved
		return moved, write_err
	}
	return 0, Stream_Empty
}

// Stream_Tee is the state behind a tee: one write, two destinations.
type Stream_Tee struct {
	// First receives every buffer.
	First Stream
	// Second receives every buffer, after First.
	Second Stream
}

// Tee_To_Stream returns a Stream that writes each buffer to both streams and reports the
// smaller count. It reports the smaller one because the caller must learn about the tighter of
// the two destinations: a tee that reported the wider count would hide the loss on the other.
func Tee_To_Stream(state *Stream_Tee) (stream Stream) {
	invariant.Always(state != nil, "A tee stream has state.")
	return Stream{Procedure: stream_tee_procedure, Data: state}
}

// Runs one mode against both streams.
func stream_tee_procedure(
	data any, mode Stream_Mode, buffer []byte, offset int64, whence Seek_From,
) (count int64, err error) {
	state, held := data.(*Stream_Tee)
	invariant.Always(held, "A tee stream procedure receives its own state.")
	if mode == STREAM_MODE_QUERY {
		return int64(STREAM_TEE_MODES), nil
	}
	if mode == STREAM_MODE_CLOSE {
		return 0, stream_tee_both(Close(state.First), Close(state.Second))
	}
	if mode == STREAM_MODE_DESTROY {
		return 0, stream_tee_both(Destroy(state.First), Destroy(state.Second))
	}
	if mode == STREAM_MODE_FLUSH {
		return 0, stream_tee_both(Flush(state.First), Flush(state.Second))
	}
	if mode == STREAM_MODE_WRITE {
		return stream_tee_write(state, buffer)
	}
	return 0, Stream_Empty
}

// Reports the first failure of the two, so neither destination can fail in silence.
func stream_tee_both(first error, second error) (err error) {
	if first != nil {
		return first
	}
	return second
}

// Writes the buffer to both and reports the smaller count.
func stream_tee_write(state *Stream_Tee, buffer []byte) (count int64, err error) {
	first_count, first_err := Write(state.First, buffer)
	second_count, second_err := Write(state.Second, buffer)
	smaller := first_count
	if second_count < smaller {
		smaller = second_count
	}
	if first_err != nil {
		return smaller, first_err
	}
	if second_err != nil {
		return smaller, second_err
	}
	return smaller, nil
}

// STREAM_BYTE_SIZE is the buffer one byte needs, the unit Read_Byte and Write_Byte move.
const STREAM_BYTE_SIZE = 1

// Read_At_Least reads until it has at least minimum bytes, or until the stream stops. It is
// Odin's read_at_least, written as the loop Odin writes: a synchronous stream needs no
// continuation to read a second time.
//
// An end reached after some bytes is Stream_Unexpected_EOF, not Stream_EOF: the caller asked
// for a quantity and got part of one, which is a different fact from finding nothing at all.
func Read_At_Least(stream Stream, buffer []byte, minimum int64) (count int64, err error) {
	if minimum < 0 {
		return 0, Stream_Negative_Count
	}
	if int64(len(buffer)) < minimum {
		return 0, Stream_Short_Buffer
	}
	for count < minimum {
		if err != nil {
			break
		}
		moved, read_err := Read(stream, buffer[count:])
		count = count + moved
		err = read_err
	}
	if count >= minimum {
		return count, nil
	}
	if err == Stream_EOF {
		if count > 0 {
			return count, Stream_Unexpected_EOF
		}
	}
	return count, err
}

// One full-buffer read in progress: the target, the bytes moved so far, and the continuation.
type Read_Full_State struct {
	// Timeline supplies the Read primitive each pass uses.
	Timeline nbio.IO
	// Completion is the caller-owned completion every pass rearms.
	Completion *time.Completion
	// Callback runs once, with the full buffer or the first error.
	Callback nbio.Callback
	// File is the descriptor each pass reads.
	File nbio.File
	// Buffer receives the bytes; its length is the goal.
	Buffer []byte
	// Offset is where the first pass reads; later passes advance it by Count.
	Offset int64
	// Count is how many bytes the passes have moved.
	Count int
	// Continue takes the next pass. It is a field so a callback reaches the pass without
	// the pass naming itself — the shape nbio's Make_Directory uses for its own rearms.
	Continue func()
}

// Read_Full reads until the buffer is full, the file stops yielding bytes, or the first
// error — the canonical rearm loop as one derived function. It is not a Stream: it waits on
// the loop, so it takes a caller-owned completion and delivers one callback rather than
// returning a count that could not exist before the root's pump runs.
func Read_Full(
	loop nbio.IO, completion *time.Completion, callback nbio.Callback, file nbio.File,
	buffer []byte, offset int64,
) {
	invariant.Always(loop.Read != nil, "Read_Full needs the Read primitive.")
	state := &Read_Full_State{
		Timeline: loop, Completion: completion, Callback: callback,
		File: file, Buffer: buffer, Offset: offset,
	}
	state.Continue = func() { read_full_step(state) }
	state.Continue()
}

// Submits one pass at the current progress. A short pass rearms at the new offset; a pass
// with no bytes delivers the count so far, so an exhausted file cannot spin the rearm loop.
func read_full_step(state *Read_Full_State) {
	state.Timeline.Read(state.Completion, func(
		completed *time.Completion, count int, err error,
	) {
		state.Count += count
		if err != nil {
			state.Callback(completed, state.Count, err)
			return
		}
		if count == 0 {
			state.Callback(completed, state.Count, nil)
			return
		}
		if state.Count >= len(state.Buffer) {
			state.Callback(completed, state.Count, nil)
			return
		}
		state.Continue()
	}, state.File, state.Buffer[state.Count:], state.Offset+int64(state.Count))
}

// One full-buffer write in progress, the Write counterpart of Read_Full_State.
type Write_Full_State struct {
	// Timeline supplies the Write primitive each pass uses.
	Timeline nbio.IO
	// Completion is the caller-owned completion every pass rearms.
	Completion *time.Completion
	// Callback runs once, with the whole buffer written or the first error.
	Callback nbio.Callback
	// File is the descriptor each pass writes.
	File nbio.File
	// Buffer holds the bytes; its length is the goal.
	Buffer []byte
	// Offset is where the first pass writes; later passes advance it by Count.
	Offset int64
	// Count is how many bytes the passes have moved.
	Count int
	// Continue takes the next pass. It is a field so a callback reaches the pass without
	// the pass naming itself.
	Continue func()
}

// Write_Full writes until the buffer is drained, the file stops accepting bytes, or the
// first error — the Write counterpart of Read_Full.
func Write_Full(
	loop nbio.IO, completion *time.Completion, callback nbio.Callback, file nbio.File,
	buffer []byte, offset int64,
) {
	invariant.Always(loop.Write != nil, "Write_Full needs the Write primitive.")
	state := &Write_Full_State{
		Timeline: loop, Completion: completion, Callback: callback,
		File: file, Buffer: buffer, Offset: offset,
	}
	state.Continue = func() { write_full_step(state) }
	state.Continue()
}

// Submits one pass at the current progress, the Write counterpart of read_full_step.
func write_full_step(state *Write_Full_State) {
	state.Timeline.Write(state.Completion, func(
		completed *time.Completion, count int, err error,
	) {
		state.Count += count
		if err != nil {
			state.Callback(completed, state.Count, err)
			return
		}
		if count == 0 {
			state.Callback(completed, state.Count, nil)
			return
		}
		if state.Count >= len(state.Buffer) {
			state.Callback(completed, state.Count, nil)
			return
		}
		state.Continue()
	}, state.File, state.Buffer[state.Count:], state.Offset+int64(state.Count))
}

// Write_String writes the bytes of text.
//
// It copies. Odin transmutes the string, because a Stream there cannot write into what it was
// given; here nothing stops an implementation from doing so, and a stream that wrote into a Go
// string would corrupt a value the language guarantees is immutable.
func Write_String(stream Stream, text string) (count int64, err error) {
	return Write(stream, []byte(text))
}

// Read_Byte reads one byte. An end found here is Stream_Unexpected_EOF, because one byte was
// asked for and none arrived.
func Read_Byte(stream Stream) (value byte, err error) {
	var storage [STREAM_BYTE_SIZE]byte
	count, read_err := Read(stream, storage[:])
	if read_err != nil {
		return 0, read_err
	}
	if count != 1 {
		return 0, Stream_Unexpected_EOF
	}
	return storage[0], nil
}

// Write_Byte writes one byte.
func Write_Byte(stream Stream, value byte) (err error) {
	storage := [STREAM_BYTE_SIZE]byte{value}
	_, write_err := Write(stream, storage[:])
	return write_err
}

// Read_Rune reads one UTF-8 character and reports how many bytes it consumed. The lead byte
// states the length, so the rest is one further read, never a search.
func Read_Rune(stream Stream) (character rune, size int64, err error) {
	var sequence [utf8.UTFMax]byte
	count, read_err := Read(stream, sequence[:STREAM_BYTE_SIZE])
	if read_err != nil {
		return 0, 0, read_err
	}
	if count != 1 {
		return 0, 0, Stream_Unexpected_EOF
	}
	if sequence[0] < utf8.RuneSelf {
		return rune(sequence[0]), 1, nil
	}
	sequence_size := stream_sequence_size(sequence[0])
	if sequence_size == 0 {
		return utf8.RuneError, STREAM_BYTE_SIZE, nil
	}
	rest := sequence[STREAM_BYTE_SIZE:sequence_size]
	_, rest_err := Read_At_Least(stream, rest, int64(len(rest)))
	if rest_err != nil {
		return 0, 0, rest_err
	}
	decoded, decoded_size := utf8.DecodeRune(sequence[:sequence_size])
	return decoded, int64(decoded_size), nil
}

// Reports how many bytes a UTF-8 sequence holds, from its lead byte. Zero rejects a lead byte
// that starts no sequence, which is a continuation byte or an invalid one.
func stream_sequence_size(lead byte) (size int64) {
	if lead < 0xC0 {
		return 0
	}
	if lead < 0xE0 {
		return 2
	}
	if lead < 0xF0 {
		return 3
	}
	if lead < 0xF8 {
		return 4
	}
	return 0
}

// Write_Rune writes one UTF-8 character and reports how many bytes it wrote.
func Write_Rune(stream Stream, character rune) (size int64, err error) {
	var sequence [utf8.UTFMax]byte
	sequence_size := utf8.EncodeRune(sequence[:], character)
	return Write(stream, sequence[:sequence_size])
}

// Read_Pointer reads size bytes into the memory at pointer — Odin's read_ptr.
//
// The caller states the size, so this cannot check it. It is the one place a Stream trusts a
// caller with a length, and the only reason this package names unsafe.
func Read_Pointer(stream Stream, pointer unsafe.Pointer, size int64) (count int64, err error) {
	if size < 0 {
		return 0, Stream_Negative_Count
	}
	return Read(stream, unsafe.Slice((*byte)(pointer), size))
}

// Write_Pointer writes size bytes from the memory at pointer — Odin's write_ptr.
func Write_Pointer(stream Stream, pointer unsafe.Pointer, size int64) (count int64, err error) {
	if size < 0 {
		return 0, Stream_Negative_Count
	}
	return Write(stream, unsafe.Slice((*byte)(pointer), size))
}
