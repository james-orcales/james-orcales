// Package csprng is a cryptographically secure pseudo-random generator: a ChaCha20 keystream
// (RFC 8439) seeded once from a 32-byte seed. It is the cryptographic sibling of prng — where prng
// is a deterministic, non-cryptographic xoshiro256++ for reproducible simulation, csprng produces
// unpredictable bytes and bounded integers for keys, tokens, nonces, and unbiased choice.
//
// The seed is injected, never read here: this library tier holds no operating-system dependency and
// the linter forbids crypto/rand in it. The one sanctioned reader of OS entropy is the composition
// tier, csprng/default, whose New_Operating_System_Generator draws the seed from crypto/rand. A
// test or simulation instead calls New with a seed of its own, and the stream is then a pure,
// deterministic function of that seed.
//
// Forward secrecy comes from fast-key-erasure (Bernstein's design, as in arc4random): each refill
// overwrites the key with fresh keystream and each delivered byte is zeroed, so disclosing a live
// Generator cannot reconstruct output it already handed out. A Generator is single-owner, like
// prng.Generator: it is NOT safe for concurrent use, and each goroutine holds its own.
//
// Read fills bytes and satisfies io.Reader; Generator_Below draws an unbiased bounded integer. The
// house forbids returning raw unbounded entropy from a free function (it has no witnessable
// invariant), so a raw 64-bit draw is a Read into eight bytes.
//
//	generator := os_csprng.New_Operating_System_Generator()
//	token := make([]byte, 32)
//	generator.Read(token)
//	victim := csprng.Generator_Below(&generator, csprng.Bound(replica_count))
//
// ChaCha20 is by Daniel J. Bernstein; the construction and vectors follow RFC 8439.
package csprng

import (
	"encoding/binary"
	"math/bits"

	invariant "local/james-orcales/shared/invariant/default"
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

// REFILL_BLOCKS is how many 64-byte ChaCha20 blocks one refill produces. Four yield 256 bytes: the
// first 32 reseed the key (fast-key-erasure), the remaining 224 are output.
const REFILL_BLOCKS = 4

// REFILL_BYTES is one refill's total keystream, 256 bytes.
const REFILL_BYTES = REFILL_BLOCKS * 64

// BUFFER_BYTES is the output a refill leaves after reseeding the key: 224 bytes.
const BUFFER_BYTES = REFILL_BYTES - KEY_BYTES

// CURSOR_MIN is the low bound of a buffer cursor: the start of the buffer.
const CURSOR_MIN Cursor = 0

// CURSOR_MAX is the high bound of a buffer cursor: a spent buffer rests here.
const CURSOR_MAX Cursor = BUFFER_BYTES

// BLOCK_COUNTER_MIN is the first block counter of a refill.
const BLOCK_COUNTER_MIN Block_Counter = 0

// BLOCK_COUNTER_MAX is the last block counter of a refill.
const BLOCK_COUNTER_MAX Block_Counter = REFILL_BLOCKS - 1

// BOUND_MIN is the smallest bound Generator_Below accepts: one.
const BOUND_MIN Bound = 1

// BOUND_MAX caps a draw bound well below where the 64-bit multiply could overflow meaning.
const BOUND_MAX Bound = 1 << 62

// INDEX_MIN is the low bound of a draw result: zero.
const INDEX_MIN Index = 0

// INDEX_MAX caps a draw result at the same ceiling as the bound it is drawn below.
const INDEX_MAX Index = 1 << 62

// SINK_MIN is the smallest fill request: an empty sink.
const SINK_MIN = 0

// SINK_MAX caps one fill request far above any real key or token.
const SINK_MAX = 1 << 40

// Block_Counter is the ChaCha20 block index within one refill, running BLOCK_COUNTER_MIN to
// BLOCK_COUNTER_MAX.
type Block_Counter uint32

// Block_Counter_Invariants bounds a block counter to one refill's range.
func Block_Counter_Invariants(counter Block_Counter, namespace invariant.Namespace) {
	invariant.Always(counter >= BLOCK_COUNTER_MIN, "A block counter is at least its min.")
	invariant.Always(counter <= BLOCK_COUNTER_MAX, "A block counter is at most its max.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(counter == 0, "A block counter is zero."),
		invariant.Sometimes(counter == 1, "A block counter is one."),
		invariant.Sometimes(counter == 2, "A block counter is two."),
		invariant.Impossible(
			invariant.Event_True("A block counter is zero."),
			invariant.Event_True("A block counter is one."),
		),
		invariant.Impossible(
			invariant.Event_True("A block counter is zero."),
			invariant.Event_True("A block counter is two."),
		),
		invariant.Impossible(
			invariant.Event_True("A block counter is one."),
			invariant.Event_True("A block counter is two."),
		),
	)
}

// Cursor is the next unread byte of a Generator's buffer, from CURSOR_MIN to CURSOR_MAX.
type Cursor uint

// Cursor_Invariants bounds a buffer cursor to the buffer.
func Cursor_Invariants(cursor Cursor, namespace invariant.Namespace) {
	invariant.Always(cursor >= CURSOR_MIN, "A cursor is at least its min.")
	invariant.Always(cursor <= CURSOR_MAX, "A cursor is at most its max.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(cursor == 0, "A cursor is zero."),
		invariant.Sometimes(cursor == 1, "A cursor is one."),
		invariant.Sometimes(cursor == 2, "A cursor is two."),
		invariant.Impossible(
			invariant.Event_True("A cursor is zero."),
			invariant.Event_True("A cursor is one."),
		),
		invariant.Impossible(
			invariant.Event_True("A cursor is zero."),
			invariant.Event_True("A cursor is two."),
		),
		invariant.Impossible(
			invariant.Event_True("A cursor is one."),
			invariant.Event_True("A cursor is two."),
		),
	)
}

// Bound is the exclusive upper limit of a Generator_Below draw: a positive count of outcomes.
type Bound uint64

// Bound_Invariants requires a bound to be positive and within the overflow-safe ceiling.
func Bound_Invariants(bound Bound, namespace invariant.Namespace) {
	invariant.Always(bound >= BOUND_MIN, "A bound is at least its min.")
	invariant.Always(bound <= BOUND_MAX, "A bound is at most its max.")
	invariant.Always(bound != 0, "A bound is never zero.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(bound == 1, "A bound is one."),
		invariant.Sometimes(bound == 2, "A bound is two."),
		invariant.Impossible(
			invariant.Event_True("A bound is one."),
			invariant.Event_True("A bound is two."),
		),
	)
}

// Index is a Generator_Below draw: a value in the half-open range zero to its bound.
type Index uint64

// Index_Invariants bounds a draw result.
func Index_Invariants(index Index, namespace invariant.Namespace) {
	invariant.Always(index >= INDEX_MIN, "An index is at least its min.")
	invariant.Always(index <= INDEX_MAX, "An index is at most its max.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(index == 0, "An index is zero."),
		invariant.Sometimes(index == 1, "An index is one."),
		invariant.Sometimes(index == 2, "An index is two."),
		invariant.Impossible(
			invariant.Event_True("An index is zero."),
			invariant.Event_True("An index is one."),
		),
		invariant.Impossible(
			invariant.Event_True("An index is zero."),
			invariant.Event_True("An index is two."),
		),
		invariant.Impossible(
			invariant.Event_True("An index is one."),
			invariant.Event_True("An index is two."),
		),
	)
}

// Sink is a caller's buffer a draw fills — a defined type so the byte draw takes no raw slice.
type Sink []byte

// Sink_Invariants bounds a fill request's length.
func Sink_Invariants(sink Sink, namespace invariant.Namespace) {
	invariant.Always(len(sink) >= SINK_MIN, "A sink is at least its min.")
	invariant.Always(len(sink) <= SINK_MAX, "A sink is at most its max.")
	invariant.Dot_Product(namespace,
		invariant.Sometimes(len(sink) == 0, "A sink is empty."),
		invariant.Sometimes(len(sink) == 1, "A sink has one."),
		invariant.Sometimes(len(sink) == 2, "A sink has two."),
		invariant.Impossible(
			invariant.Event_True("A sink is empty."),
			invariant.Event_True("A sink has one."),
		),
		invariant.Impossible(
			invariant.Event_True("A sink is empty."),
			invariant.Event_True("A sink has two."),
		),
		invariant.Impossible(
			invariant.Event_True("A sink has one."),
			invariant.Event_True("A sink has two."),
		),
	)
}

// Generator is the state of a fast-key-erasure ChaCha20 keystream. Construct it with New; the zero
// value is degenerate. Its fields are transparent, like prng.Generator's State: a Generator is
// single-owner, so copying it forks the stream, and forward secrecy rests on rolling the key and
// zeroing delivered bytes, not on hiding these fields.
type Generator struct {
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

// Generator_Invariants states a Generator's buffer position; its key and buffer are fixed-size
// arrays with no bundle of their own.
func Generator_Invariants(generator Generator, namespace invariant.Namespace) {
	Cursor_Invariants(generator.Position, "Generator.Position")
}

// New seeds a Generator and performs the first fast-key-erasure refill, so the caller-supplied seed
// is erased from the Generator before New returns. The zero Generator is degenerate; always
// construct through New.
func New(seed [KEY_BYTES]byte) (generator Generator) {
	defer func() { Generator_Invariants(generator, "new.generator") }()
	generator.Key = seed
	generator_refill(&generator)
	return generator
}

// Read fills p with cryptographically secure bytes, always fully, returning len(p) and a nil error.
// It is the one method the house style permits: it exists so *Generator satisfies io.Reader. The
// keystream is inexhaustible, so a read never comes up short and never errors.
func (generator *Generator) Read(p []byte) (n int, err error) {
	generator_drain(generator, Sink(p))
	return len(p), nil
}

// Generator_Below returns a value in the half-open range zero to bound, never bound itself, using
// Lemire's method so the result is unbiased.
func Generator_Below(generator *Generator, bound Bound) (index Index) {
	defer func() { Index_Invariants(index, "generator_below.index") }()
	Generator_Invariants(*generator, "generator_below.generator")
	Bound_Invariants(bound, "generator_below.bound")
	limit := uint64(bound)
	// The raw draw stays a local: a free function returning unbounded entropy has no
	// witnessable invariant, so the word is read into bytes and assembled here, not returned.
	var octet [8]byte
	generator_drain(generator, Sink(octet[:]))
	random := binary.LittleEndian.Uint64(octet[:])
	high, low := bits.Mul64(random, limit)
	if low < limit {
		threshold := (-limit) % limit
		for low < threshold {
			generator_drain(generator, Sink(octet[:]))
			random = binary.LittleEndian.Uint64(octet[:])
			high, low = bits.Mul64(random, limit)
		}
	}
	return Index(high)
}

// Copies the next len(destination) bytes of keystream into destination, refilling as it exhausts
// the buffer, and zeroes each delivered byte. The zeroing is the second half of fast-key-erasure:
// with the key already rolled forward on each refill, wiping delivered bytes means a disclosed
// Generator holds neither an earlier key nor any output it already returned.
func generator_drain(generator *Generator, destination Sink) {
	Generator_Invariants(*generator, "generator_drain.generator")
	Sink_Invariants(destination, "generator_drain.destination")
	filled := 0
	for filled < len(destination) {
		if generator.Position == CURSOR_MAX {
			generator_refill(generator)
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
func generator_refill(generator *Generator) {
	Generator_Invariants(*generator, "generator_refill.generator")
	var stream [REFILL_BYTES]byte
	var block [64]byte
	for block_index := 0; block_index < REFILL_BLOCKS; block_index++ {
		chacha20_block(generator.Key, Block_Counter(block_index), [12]byte{}, &block)
		copy(stream[block_index*64:], block[:])
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
	key [KEY_BYTES]byte, counter Block_Counter, nonce [12]byte, output *[64]byte,
) {
	Block_Counter_Invariants(counter, "chacha20_block.counter")
	var state [16]uint32
	state[0] = CHACHA_CONSTANT_FIRST
	state[1] = CHACHA_CONSTANT_SECOND
	state[2] = CHACHA_CONSTANT_THIRD
	state[3] = CHACHA_CONSTANT_FOURTH
	for word_index := 0; word_index < 8; word_index++ {
		state[4+word_index] = binary.LittleEndian.Uint32(key[word_index*4:])
	}
	state[12] = uint32(counter)
	for word_index := 0; word_index < 3; word_index++ {
		state[13+word_index] = binary.LittleEndian.Uint32(nonce[word_index*4:])
	}
	scratch := state
	for round_index := 0; round_index < 10; round_index++ {
		quarter_round(&scratch, [4]int{0, 4, 8, 12})
		quarter_round(&scratch, [4]int{1, 5, 9, 13})
		quarter_round(&scratch, [4]int{2, 6, 10, 14})
		quarter_round(&scratch, [4]int{3, 7, 11, 15})
		quarter_round(&scratch, [4]int{0, 5, 10, 15})
		quarter_round(&scratch, [4]int{1, 6, 11, 12})
		quarter_round(&scratch, [4]int{2, 7, 8, 13})
		quarter_round(&scratch, [4]int{3, 4, 9, 14})
	}
	for word_index := 0; word_index < 16; word_index++ {
		scratch[word_index] += state[word_index]
		binary.LittleEndian.PutUint32(output[word_index*4:], scratch[word_index])
	}
}

// Applies the ChaCha quarter-round in place to the four state words at indices (RFC 8439 2.1):
// four add-xor-rotate steps with rotations of 16, 12, 8, and 7 bits. It takes the state and indices
// rather than four words so it mutates the shared state the block function threads through it.
func quarter_round(state *[16]uint32, indices [4]int) {
	a, b, c, d := indices[0], indices[1], indices[2], indices[3]
	state[a] += state[b]
	state[d] ^= state[a]
	state[d] = bits.RotateLeft32(state[d], 16)
	state[c] += state[d]
	state[b] ^= state[c]
	state[b] = bits.RotateLeft32(state[b], 12)
	state[a] += state[b]
	state[d] ^= state[a]
	state[d] = bits.RotateLeft32(state[d], 8)
	state[c] += state[d]
	state[b] ^= state[c]
	state[b] = bits.RotateLeft32(state[b], 7)
}
