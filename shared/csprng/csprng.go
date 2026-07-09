// Package csprng is a cryptographically secure pseudo-random generator: a ChaCha20 keystream
// (RFC 8439) seeded once from a 32-byte seed. It is the cryptographic sibling of prng — where prng
// is a deterministic, non-cryptographic xoshiro256++ for reproducible simulation, csprng produces
// unpredictable bytes and integers for keys, tokens, nonces, and unbiased choice.
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
// The house linter bans methods, so each draw is a free function named after its first parameter's
// type; the sole exception is Read, which exists to satisfy io.Reader. Seed with New; the zero
// Generator is degenerate.
//
//	generator := os_csprng.New_Operating_System_Generator()
//	token := make([]byte, 32)
//	generator.Read(token)
//	victim := csprng.Generator_Below(&generator, replica_count)
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
// first 32 reseed the key (fast-key-erasure), the remaining 224 are output. This is Bernstein's
// 256-byte granularity — small enough to discard a key quickly, large enough that the block counter
// only ever runs 0..3 under one key and never nears its bound.
const REFILL_BLOCKS = 4

// REFILL_BYTES is one refill's total keystream, 256 bytes.
const REFILL_BYTES = REFILL_BLOCKS * 64

// BUFFER_BYTES is the output a refill leaves after reseeding the key: 224 bytes.
const BUFFER_BYTES = REFILL_BYTES - KEY_BYTES

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
	// Position is the next unread byte in Buffer; BUFFER_BYTES means the buffer is spent and a
	// draw must refill.
	Position int
}

// New seeds a Generator and performs the first fast-key-erasure refill, so the caller-supplied seed
// is erased from the Generator before New returns. The zero Generator is degenerate; always
// construct through New.
func New(seed [KEY_BYTES]byte) (generator Generator) {
	generator.Key = seed
	generator_refill(&generator)
	return generator
}

// Read fills p with cryptographically secure bytes, always fully, returning len(p) and a nil error.
// It is the one method the house style permits: it exists so *Generator satisfies io.Reader and
// plugs in anywhere an entropy Reader is wanted. The keystream is inexhaustible, so a read never
// comes up short and never errors.
func (generator *Generator) Read(p []byte) (n int, err error) {
	generator_drain(generator, p)
	return len(p), nil
}

// Generator_Uint64 returns the next 64 bits of the keystream, assembled little-endian.
func Generator_Uint64(generator *Generator) (value uint64) {
	var octet [8]byte
	generator_drain(generator, octet[:])
	return binary.LittleEndian.Uint64(octet[:])
}

// Generator_Below returns a value in the half-open range zero to bound, never bound itself.
func Generator_Below(generator *Generator, bound int) (value int) {
	invariant.Always(bound > 0, "csprng below bound is positive")
	return int(generator_below_unsigned(generator, uint64(bound)))
}

// Returns a value in the half-open range zero to bound using Lemire's method, so the result is
// unbiased, not skewed the way a plain modulo would be. It mirrors prng's draw but pulls its 64-bit
// values from the ChaCha20 keystream. The caller guarantees bound is positive.
func generator_below_unsigned(generator *Generator, bound uint64) (value uint64) {
	random := Generator_Uint64(generator)
	high, low := bits.Mul64(random, bound)
	if low < bound {
		threshold := (-bound) % bound
		for low < threshold {
			random = Generator_Uint64(generator)
			high, low = bits.Mul64(random, bound)
		}
	}
	return high
}

// Copies the next len(dst) bytes of keystream into dst, refilling as it exhausts the buffer, and
// zeroes each delivered byte. The zeroing is the second half of fast-key-erasure: with the key
// already rolled forward on each refill, wiping delivered bytes means a disclosed Generator holds
// neither an earlier key nor any output it already returned.
func generator_drain(generator *Generator, dst []byte) {
	filled := 0
	for filled < len(dst) {
		if generator.Position == BUFFER_BYTES {
			generator_refill(generator)
		}
		take := BUFFER_BYTES - generator.Position
		if remainder := len(dst) - filled; remainder < take {
			take = remainder
		}
		source := generator.Buffer[generator.Position : generator.Position+take]
		copy(dst[filled:filled+take], source)
		for index := 0; index < take; index++ {
			source[index] = 0
		}
		generator.Position += take
		filled += take
	}
}

// Runs the ChaCha20 block function REFILL_BLOCKS times over the current key, reseeds the key from
// the first 32 output bytes, and stores the remaining 224 as the buffer. Nonce is always zero and
// the counter only runs 0..REFILL_BLOCKS-1: safe against reuse because each key expands exactly one
// 256-byte block and is then discarded, so no (key, nonce, counter) triple ever repeats.
func generator_refill(generator *Generator) {
	var stream [REFILL_BYTES]byte
	var block [64]byte
	for block_index := 0; block_index < REFILL_BLOCKS; block_index++ {
		chacha20_block(generator.Key, uint32(block_index), [12]byte{}, &block)
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
func chacha20_block(key [KEY_BYTES]byte, counter uint32, nonce [12]byte, output *[64]byte) {
	var state [16]uint32
	state[0] = CHACHA_CONSTANT_FIRST
	state[1] = CHACHA_CONSTANT_SECOND
	state[2] = CHACHA_CONSTANT_THIRD
	state[3] = CHACHA_CONSTANT_FOURTH
	for word_index := 0; word_index < 8; word_index++ {
		state[4+word_index] = binary.LittleEndian.Uint32(key[word_index*4:])
	}
	state[12] = counter
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
