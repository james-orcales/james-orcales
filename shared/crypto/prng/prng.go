// Package prng is a cryptographically secure pseudo-random generator: a ChaCha20 keystream
// (RFC 8439) seeded once from a 32-byte seed. It is the cryptographic sibling of simulation/prng —
// where that is a deterministic, non-cryptographic xoshiro256++ for reproducible simulation, this
// produces unpredictable bytes and bounded integers for keys, tokens, nonces, and unbiased choice.
//
// The seed is injected, never read here: this library tier holds no operating-system dependency and
// the linter forbids crypto/rand in it. The one sanctioned reader of OS entropy is the composition
// tier, crypto/prng/default, whose New_Operating_System_Chacha draws the seed from crypto/rand.
// A test or simulation instead calls New with a seed of its own, and the stream is then a pure,
// deterministic function of that seed.
//
// Forward secrecy comes from fast-key-erasure (Bernstein's design, as in arc4random): each refill
// overwrites the key with fresh keystream and each delivered byte is zeroed, so disclosing a live
// Chacha cannot reconstruct output it already handed out. A Chacha is single-owner, like
// prng.Xoshiro: it is NOT safe for concurrent use, and each goroutine holds its own.
//
// Read fills bytes and satisfies io.Reader; Chacha_Below draws an unbiased bounded integer. The
// house forbids returning raw unbounded entropy from a free function (it has no witnessable
// invariant), so a raw 64-bit draw is a Read into eight bytes.
//
//	generator := system_prng.New_Operating_System_Chacha(0)
//	token := make([]byte, 32)
//	generator.Read(token)
//	victim := prng.Chacha_Below(&generator, prng.Bound(replica_count))
//
// ChaCha20 is by Daniel J. Bernstein; the construction and vectors follow RFC 8439.
package prng

import (
	"unsafe"

	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
)

// CHACHA_CONSTANT_FIRST is the little-endian word for ASCII "expa", first of the four constants
// that fix ChaCha's top row and domain-separate the 256-bit-key cipher (RFC 8439 2.3).
const CHACHA_CONSTANT_FIRST = 0x61707865

// CHACHA_CONSTANT_SECOND is the little-endian word for ASCII "nd 3".
const CHACHA_CONSTANT_SECOND = 0x3320646e

// CHACHA_CONSTANT_THIRD is the little-endian word for ASCII "2-by".
const CHACHA_CONSTANT_THIRD = 0x79622d32

// CHACHA_CONSTANT_FOURTH is the little-endian word for ASCII "te k".
const CHACHA_CONSTANT_FOURTH = 0x6b206574

// KEY_BYTES is the ChaCha20 key width, and so the seed width: 256 bits.
const KEY_BYTES = 32

// WORD_BYTE_COUNT matches the uint64 draw that Chacha_Below assembles.
const WORD_BYTE_COUNT = 8

// CHACHA_BLOCK_BYTE_COUNT is the block width that RFC 8439 specifies.
const CHACHA_BLOCK_BYTE_COUNT = 64

// NONCE_BYTE_COUNT is the nonce width that RFC 8439 specifies.
const NONCE_BYTE_COUNT = 12

// CHACHA_STATE_WORD_COUNT is the state width that RFC 8439 specifies.
const CHACHA_STATE_WORD_COUNT = 16

// QUARTER_ROUND_WORD_COUNT limits each round to its four coupled state words.
const QUARTER_ROUND_WORD_COUNT = 4

// REFILL_BLOCKS is how many 64-byte ChaCha20 blocks one refill produces. Four yield 256 bytes: the
// first 32 reseed the key (fast-key-erasure), the remaining 224 are output.
const REFILL_BLOCKS = 4

// REFILL_BYTES is one refill's total keystream, 256 bytes.
const REFILL_BYTES = REFILL_BLOCKS * CHACHA_BLOCK_BYTE_COUNT

// BUFFER_BYTES is the output a refill leaves after reseeding the key: 224 bytes.
const BUFFER_BYTES = REFILL_BYTES - KEY_BYTES

// CURSOR_MIN is the low bound of a buffer cursor: the start of the buffer.
const CURSOR_MIN Cursor = 0

// CURSOR_MAX is the high bound of a buffer cursor: a spent buffer rests here.
const CURSOR_MAX Cursor = BUFFER_BYTES

// BLOCK_COUNTER_MIN is the first block counter of a refill.
const BLOCK_COUNTER_MIN = 0

// BLOCK_COUNTER_SECOND identifies the second block that one refill requires.
const BLOCK_COUNTER_SECOND = 1

// BLOCK_COUNTER_THIRD identifies the third block that one refill requires.
const BLOCK_COUNTER_THIRD = 2

// BLOCK_COUNTER_MAX is the last block counter of a refill.
const BLOCK_COUNTER_MAX = REFILL_BLOCKS - 1

// BOUND_MIN is the smallest bound Chacha_Below accepts: one.
const BOUND_MIN Bound = 1

// BOUND_MAX caps a draw bound well below where the 64-bit multiply could overflow meaning.
const BOUND_MAX Bound = 1 << 62

// INDEX_MIN is the low bound of a draw result: zero.
const INDEX_MIN Index = 0

// INDEX_MAX is one below the largest bound because Chacha_Below's range is half-open.
const INDEX_MAX Index = (1 << 62) - 1

// SINK_MIN is the smallest fill request: an empty sink.
const SINK_MIN = 0

// SINK_MAX keeps an accidental bulk fill bounded while staying far above any key, token, or nonce
// a caller fills.
const SINK_MAX = 1 << 20

// WORD_MINIMUM preserves the complete draw domain of a Source slot.
const WORD_MINIMUM Word = Word(bits.WORD_64_MINIMUM)

// WORD_MAXIMUM preserves the complete draw domain of a Source slot.
const WORD_MAXIMUM Word = Word(bits.WORD_64_MAXIMUM)

// CHACHA_WORD_BYTE_COUNT is the byte width of one 32-bit ChaCha state word.
const CHACHA_WORD_BYTE_COUNT = 4

// Block_Counter is the ChaCha20 block index within one refill, running BLOCK_COUNTER_MIN to
// BLOCK_COUNTER_MAX.
type Block_Counter uint32

// Block_Counter_Invariants bounds a block counter to one refill's range.
func Block_Counter_Invariants(counter Block_Counter, namespace invariant.Namespace) {
	invariant.Tree(counter, namespace).
		Enum_4_Uint32(
			uint32(counter),
			BLOCK_COUNTER_MIN,
			BLOCK_COUNTER_SECOND,
			BLOCK_COUNTER_THIRD,
			BLOCK_COUNTER_MAX,
		).
		Ensure()
}

// Cursor is the next unread byte of a Chacha's buffer, from CURSOR_MIN to CURSOR_MAX.
type Cursor uint

// Cursor_Invariants bounds a buffer cursor to the buffer.
func Cursor_Invariants(cursor Cursor, namespace invariant.Namespace) {
	invariant.Tree(cursor, namespace).
		Range_Uint(uint(cursor), uint(CURSOR_MIN), uint(CURSOR_MAX)).
		Ensure()
}

// Bound is the exclusive upper limit of a Chacha_Below draw: a positive count of outcomes.
type Bound uint64

// Bound_Invariants requires a bound to be positive and within the overflow-safe ceiling.
func Bound_Invariants(bound Bound, namespace invariant.Namespace) {
	invariant.Tree(bound, namespace).
		Range_Uint64(uint64(bound), uint64(BOUND_MIN), uint64(BOUND_MAX)).
		Ensure()
}

// Index is a Chacha_Below draw: a value in the half-open range zero to its bound.
type Index uint64

// Index_Invariants bounds a draw result.
func Index_Invariants(index Index, namespace invariant.Namespace) {
	invariant.Tree(index, namespace).
		Range_Uint64(uint64(index), uint64(INDEX_MIN), uint64(INDEX_MAX)).
		Ensure()
}

// Sink is a caller's buffer a draw fills — a defined type so the byte draw takes no raw slice.
type Sink []byte

// Sink_Invariants bounds a fill request's length.
func Sink_Invariants(sink Sink, namespace invariant.Namespace) {
	invariant.Tree(sink, namespace).
		Range_Int(len(sink), SINK_MIN, SINK_MAX).
		Ensure()
}

// Chacha is the state of a fast-key-erasure ChaCha20 keystream. Construct it with New; the zero
// value is degenerate. Its fields are transparent, like prng.Xoshiro's State: a Chacha is
// single-owner, so copying it forks the stream, and forward secrecy rests on rolling the key and
// zeroing delivered bytes, not on hiding these fields.
type Chacha struct {
	// Key is the current ChaCha20 key. Each refill overwrites it with fresh keystream, so the
	// seed and every earlier key vanish the moment their output is produced.
	Key [KEY_BYTES]byte
	// Buffer holds output from the last refill. Delivered bytes are zeroed in place, so
	// Buffer[:Position] is always zero and a disclosure cannot recover handed-out output.
	Buffer [BUFFER_BYTES]byte
	// Position is the next unread byte in Buffer; CURSOR_MAX means the buffer is spent and a
	// draw must refill.
	Position Cursor
}

// Chacha_Invariants states a Chacha's buffer position; its key and buffer are fixed-size
// arrays with no bundle of their own.
func Chacha_Invariants(generator Chacha, namespace invariant.Namespace) {
	Cursor_Invariants(generator.Position, namespace)
}

// New seeds a Chacha at position and performs the first fast-key-erasure refill, so the
// caller-supplied seed is erased from the Chacha before New returns. The zero Chacha is
// degenerate; always construct through New.
func New(seed [KEY_BYTES]byte, position Cursor) (generator Chacha) {
	defer func() {
		Chacha_Invariants(generator, "new.generator")
	}()
	Cursor_Invariants(position, "new.position")
	generator.Key = seed
	chacha_refill(&generator)
	for index := 0; index < int(position); index++ {
		generator.Buffer[index] = 0
	}
	generator.Position = position
	return generator
}

// Read fills p with cryptographically secure bytes, always fully, returning len(p) and a nil error.
// It is the one method the house style permits: it exists so *Chacha satisfies io.Reader. The
// keystream is inexhaustible, so a read never comes up short and never errors.
func (generator *Chacha) Read(p []byte) (n int, err error) {
	chacha_drain(generator, Sink(p))
	return len(p), nil
}

// Word is one full draw through a Source slot.
type Word uint64

// Word_Invariants preserves the complete draw domain.
func Word_Invariants(word Word, namespace invariant.Namespace) {
	invariant.Tree(word, namespace).
		Range_Uint64(uint64(word), uint64(WORD_MINIMUM), uint64(WORD_MAXIMUM)).
		Ensure()
}

// Source is the C-style vtable a caller injects where it needs cryptographic bytes: caller-owned
// backend state behind an unsafe.Pointer and one procedure that receives it. The house bans
// interfaces and closures that capture state, so this is the one shape a backend can take. A
// production root binds a Chacha through Chacha_To_Source. A simulation binds a xoshiro stream
// through simulation/prng's Xoshiro_To_Source, and that call is the one place a fake enters, so a
// grep for it finds every test that signs with predictable bytes.
type Source struct {
	// State is the caller-owned backend generator; the procedure casts it back to its own type.
	State unsafe.Pointer
	// Next draws one full word from the backend behind state. The slot returns a value and
	// receives no sink, because a pointer handed to a procedure value escapes to the heap under
	// Go's escape analysis, and a caller's stack sink must stay on its stack.
	Next func(state unsafe.Pointer) (value Word)
}

// Source_Invariants proves both halves of the vtable are bound before any draw.
func Source_Invariants(source Source, _ invariant.Namespace) {
	invariant.Always(source.State != nil, "A Source has caller-owned state.")
	invariant.Always(source.Next != nil, "A Source has a bound draw procedure.")
}

// Source_Read fills sink through the vtable, one word per eight bytes, little-endian so a shorter
// read is a prefix of a longer one from the same state. A partial tail spends a whole word. The
// bytes are packed here, on the caller's side of the slot, so the sink never crosses it.
func Source_Read(source Source, sink Sink) {
	Source_Invariants(source, "source_read.source")
	Sink_Invariants(sink, "source_read.sink")
	for filled := 0; filled < len(sink); filled += WORD_BYTE_COUNT {
		octet := word_to_bytes(source.Next(source.State))
		copy(sink[filled:], octet[:])
	}
}

// Chacha_To_Source binds caller-owned state into Source without a captured function
// environment. Only a production root calls this; a simulation binds a xoshiro stream instead.
func Chacha_To_Source(generator *Chacha) (source Source) {
	defer func() { Source_Invariants(source, "chacha_to_source.source") }()
	Chacha_Invariants(*generator, "chacha_to_source.generator")
	source = Source{
		State: unsafe.Pointer(generator),
		Next:  chacha_source_next,
	}
	return source
}

// The vtable slot: eight stream bytes assembled little-endian, the same way Chacha_Below
// assembles its draw, so Source_Read unpacks them back into stream order.
func chacha_source_next(state unsafe.Pointer) (value Word) {
	defer func() { Word_Invariants(value, "chacha_source_next.value") }()
	var octet [WORD_BYTE_COUNT]byte
	chacha_drain((*Chacha)(state), Sink(octet[:]))
	return word_from_bytes(octet)
}

// Assembles eight bytes into one little-endian word. Spelled here rather than through
// encoding/binary, because that package reaches simulation/prng through nbio, and simulation/prng
// imports this one to bind a Xoshiro into Source.
func word_from_bytes(octet [WORD_BYTE_COUNT]byte) (word Word) {
	defer func() { Word_Invariants(word, "word_from_bytes.word") }()
	for index := WORD_BYTE_COUNT - 1; index >= 0; index-- {
		word = word<<bits.BIT_COUNT_8_MAXIMUM | Word(octet[index])
	}
	return word
}

// Splits one word into eight little-endian bytes, the inverse of word_from_bytes.
func word_to_bytes(word Word) (octet [WORD_BYTE_COUNT]byte) {
	Word_Invariants(word, "word_to_bytes.word")
	for index := 0; index < WORD_BYTE_COUNT; index++ {
		octet[index] = byte(word >> (index * bits.BIT_COUNT_8_MAXIMUM))
	}
	return octet
}

// Chacha_Below returns a value in the half-open range zero to bound, never bound itself, using
// Lemire's method so the result is unbiased.
func Chacha_Below(generator *Chacha, bound Bound) (index Index) {
	defer func() { Index_Invariants(index, "chacha_below.index") }()
	Chacha_Invariants(*generator, "chacha_below.generator")
	Bound_Invariants(bound, "chacha_below.bound")
	limit := uint64(bound)
	// The raw draw stays a local: a free function returning unbounded entropy has no
	// witnessable invariant, so the word is read into bytes and assembled here, not returned.
	var octet [WORD_BYTE_COUNT]byte
	chacha_drain(generator, Sink(octet[:]))
	word := uint64(word_from_bytes(octet))
	high_word, low_word := bits.Multiply_64(
		bits.Word_64(word), bits.Multiplier_64(limit),
	)
	high, low := uint64(high_word), uint64(low_word)
	if low < limit {
		threshold := (-limit) % limit
		for low < threshold {
			chacha_drain(generator, Sink(octet[:]))
			word = uint64(word_from_bytes(octet))
			high_word, low_word = bits.Multiply_64(
				bits.Word_64(word), bits.Multiplier_64(limit),
			)
			high, low = uint64(high_word), uint64(low_word)
		}
	}
	return Index(high)
}

// Copies the next len(destination) bytes of keystream into destination, refilling as it exhausts
// the buffer, and zeroes each delivered byte. The zeroing is the second half of fast-key-erasure:
// with the key already rolled forward on each refill, wiping delivered bytes means a disclosed
// Chacha holds neither an earlier key nor any output it already returned.
func chacha_drain(generator *Chacha, destination Sink) {
	Chacha_Invariants(*generator, "chacha_drain.generator")
	Sink_Invariants(destination, "chacha_drain.destination")
	filled := 0
	for filled < len(destination) {
		if generator.Position == CURSOR_MAX {
			chacha_refill(generator)
		}
		position := int(generator.Position)
		take := BUFFER_BYTES - position
		if remainder := len(destination) - filled; remainder < take {
			take = remainder
		}
		source := generator.Buffer[position : position+take]
		copy(destination[filled:filled+take], source)
		for index := 0; index < take; index++ {
			source[index] = 0
		}
		generator.Position = Cursor(position + take)
		filled += take
	}
}

// Runs the ChaCha20 block function REFILL_BLOCKS times over the current key, reseeds the key from
// the first 32 output bytes, and stores the remaining 224 as the buffer. Nonce is always zero and
// the counter only runs its refill range: safe against reuse because each key expands exactly one
// 256-byte block and is then discarded, so no (key, nonce, counter) triple ever repeats.
func chacha_refill(generator *Chacha) {
	Chacha_Invariants(*generator, "chacha_refill.generator")
	var stream [REFILL_BYTES]byte
	var block [CHACHA_BLOCK_BYTE_COUNT]byte
	for block_index := 0; block_index < REFILL_BLOCKS; block_index++ {
		chacha20_block(
			generator.Key,
			Block_Counter(block_index),
			[NONCE_BYTE_COUNT]byte{},
			&block,
		)
		copy(stream[block_index*CHACHA_BLOCK_BYTE_COUNT:], block[:])
	}
	copy(generator.Key[:], stream[:KEY_BYTES])
	copy(generator.Buffer[:], stream[KEY_BYTES:])
	generator.Position = 0
}

// Computes one 64-byte ChaCha20 keystream block for the key, block counter, and nonce, writing
// it to output (RFC 8439 2.3). The 16-word state is the four constants, the eight key words, the
// counter, and the three nonce words, all little-endian; twenty rounds (ten column-and-diagonal
// double rounds) mix a scratch copy, which is added back to the original and serialized.
func chacha20_block(
	key [KEY_BYTES]byte,
	counter Block_Counter,
	nonce [NONCE_BYTE_COUNT]byte,
	output *[CHACHA_BLOCK_BYTE_COUNT]byte,
) {
	Block_Counter_Invariants(counter, "chacha20_block.counter")
	var state [CHACHA_STATE_WORD_COUNT]uint32
	state[0] = CHACHA_CONSTANT_FIRST
	state[1] = CHACHA_CONSTANT_SECOND
	state[2] = CHACHA_CONSTANT_THIRD
	state[3] = CHACHA_CONSTANT_FOURTH
	// Key and nonce words assemble little-endian by hand: encoding/binary reaches
	// simulation/prng through nbio, and simulation/prng imports this package to bind a Xoshiro.
	for word_index := 0; word_index < 8; word_index++ {
		for byte_index := CHACHA_WORD_BYTE_COUNT - 1; byte_index >= 0; byte_index-- {
			state[4+word_index] = state[4+word_index]<<bits.BIT_COUNT_8_MAXIMUM |
				uint32(key[word_index*CHACHA_WORD_BYTE_COUNT+byte_index])
		}
	}
	state[12] = uint32(counter)
	for word_index := 0; word_index < 3; word_index++ {
		for byte_index := CHACHA_WORD_BYTE_COUNT - 1; byte_index >= 0; byte_index-- {
			state[13+word_index] = state[13+word_index]<<bits.BIT_COUNT_8_MAXIMUM |
				uint32(nonce[word_index*CHACHA_WORD_BYTE_COUNT+byte_index])
		}
	}
	scratch := state
	for round_index := 0; round_index < 10; round_index++ {
		quarter_round(&scratch, [QUARTER_ROUND_WORD_COUNT]int{0, 4, 8, 12})
		quarter_round(&scratch, [QUARTER_ROUND_WORD_COUNT]int{1, 5, 9, 13})
		quarter_round(&scratch, [QUARTER_ROUND_WORD_COUNT]int{2, 6, 10, 14})
		quarter_round(&scratch, [QUARTER_ROUND_WORD_COUNT]int{3, 7, 11, 15})
		quarter_round(&scratch, [QUARTER_ROUND_WORD_COUNT]int{0, 5, 10, 15})
		quarter_round(&scratch, [QUARTER_ROUND_WORD_COUNT]int{1, 6, 11, 12})
		quarter_round(&scratch, [QUARTER_ROUND_WORD_COUNT]int{2, 7, 8, 13})
		quarter_round(&scratch, [QUARTER_ROUND_WORD_COUNT]int{3, 4, 9, 14})
	}
	for word_index := 0; word_index < 16; word_index++ {
		scratch[word_index] += state[word_index]
		for byte_index := 0; byte_index < CHACHA_WORD_BYTE_COUNT; byte_index++ {
			output[word_index*CHACHA_WORD_BYTE_COUNT+byte_index] =
				byte(scratch[word_index] >> (byte_index * bits.BIT_COUNT_8_MAXIMUM))
		}
	}
}

// Applies the ChaCha quarter-round in place to the four state words at indices (RFC 8439 2.1):
// four add-xor-rotate steps with rotations of 16, 12, 8, and 7 bits. It takes the state and indices
// rather than four words so it mutates the shared state the block function threads through it.
func quarter_round(
	state *[CHACHA_STATE_WORD_COUNT]uint32,
	indices [QUARTER_ROUND_WORD_COUNT]int,
) {
	a, b, c, d := indices[0], indices[1], indices[2], indices[3]
	state[a] += state[b]
	state[d] ^= state[a]
	state[d] = uint32(bits.Rotate_Left_32(bits.Word_32(state[d]), 16))
	state[c] += state[d]
	state[b] ^= state[c]
	state[b] = uint32(bits.Rotate_Left_32(bits.Word_32(state[b]), 12))
	state[a] += state[b]
	state[d] ^= state[a]
	state[d] = uint32(bits.Rotate_Left_32(bits.Word_32(state[d]), 8))
	state[c] += state[d]
	state[b] ^= state[c]
	state[b] = uint32(bits.Rotate_Left_32(bits.Word_32(state[b]), 7))
}
