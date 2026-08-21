// Package prng is a cryptographically secure pseudo-random generator: a ChaCha20 keystream
// (RFC 8439) seeded once from a 32-byte seed. It is the cryptographic sibling of sim/prng —
// where that is a deterministic, non-cryptographic xoshiro256++ for reproducible simulation, this
// produces unpredictable bytes and bounded integers for keys, tokens, nonces, and unbiased choice.
//
// The seed is injected, never read here: this library tier holds no operating-system dependency and
// the linter forbids crypto/rand in it. The one sanctioned reader of OS entropy is the composition
// tier, crypto/prng/default, whose Chacha_Init draws the seed from crypto/rand. A test or
// simulation instead calls Chacha_Init with a seed of its own, and the stream is then a pure,
// deterministic function of that seed.
//
// Forward secrecy comes from fast-key-erasure (Bernstein's design, as in arc4random): each refill
// overwrites the key with fresh keystream and each delivered byte is zeroed, so disclosing a live
// Chacha cannot reconstruct output it already handed out. A Chacha is single-owner, like
// prng.Xoshiro: it is NOT safe for concurrent use, and each goroutine holds its own.
//
// Chacha_Read fills bytes; Chacha_Below draws an unbiased bounded integer. The
// house forbids returning raw unbounded entropy from a free function (it has no witnessable
// invariant), so a raw 64-bit draw is a Read into eight bytes.
//
//	var generator prng.Chacha
//	var seed [system_prng.SEED_BYTE_COUNT]byte
//	system_prng.Chacha_Init(&generator, read, seed[:], 0)
//	token := make([]byte, 32)
//	prng.Chacha_Read(&generator, token)
//	victim := prng.Chacha_Below(&generator, prng.Bound(replica_count))
//
// ChaCha20 is by Daniel J. Bernstein; the construction and vectors follow RFC 8439.
package prng

import (
	"unsafe"

	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
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

// QUARTER_A_INDEX_MINIMUM begins the first quarter-round lane partition.
const QUARTER_A_INDEX_MINIMUM = 0

// QUARTER_A_INDEX_SECOND identifies the second first-partition lane.
const QUARTER_A_INDEX_SECOND = 1

// QUARTER_A_INDEX_THIRD identifies the third first-partition lane.
const QUARTER_A_INDEX_THIRD = 2

// QUARTER_A_INDEX_MAXIMUM ends the first quarter-round lane partition.
const QUARTER_A_INDEX_MAXIMUM = 3

// QUARTER_B_INDEX_MINIMUM begins the second quarter-round lane partition.
const QUARTER_B_INDEX_MINIMUM = 4

// QUARTER_B_INDEX_SECOND identifies the second second-partition lane.
const QUARTER_B_INDEX_SECOND = 5

// QUARTER_B_INDEX_THIRD identifies the third second-partition lane.
const QUARTER_B_INDEX_THIRD = 6

// QUARTER_B_INDEX_MAXIMUM ends the second quarter-round lane partition.
const QUARTER_B_INDEX_MAXIMUM = 7

// QUARTER_C_INDEX_MINIMUM begins the third quarter-round lane partition.
const QUARTER_C_INDEX_MINIMUM = 8

// QUARTER_C_INDEX_SECOND identifies the second third-partition lane.
const QUARTER_C_INDEX_SECOND = 9

// QUARTER_C_INDEX_THIRD identifies the third third-partition lane.
const QUARTER_C_INDEX_THIRD = 10

// QUARTER_C_INDEX_MAXIMUM ends the third quarter-round lane partition.
const QUARTER_C_INDEX_MAXIMUM = 11

// QUARTER_D_INDEX_MINIMUM begins the fourth quarter-round lane partition.
const QUARTER_D_INDEX_MINIMUM = 12

// QUARTER_D_INDEX_SECOND identifies the second fourth-partition lane.
const QUARTER_D_INDEX_SECOND = 13

// QUARTER_D_INDEX_THIRD identifies the third fourth-partition lane.
const QUARTER_D_INDEX_THIRD = 14

// QUARTER_D_INDEX_MAXIMUM ends the fourth quarter-round lane partition.
const QUARTER_D_INDEX_MAXIMUM = 15

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
func Block_Counter_Invariants(counter Block_Counter, namespace aver.Namespace) {
	aver.Tree(counter, namespace).
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
func Cursor_Invariants(cursor Cursor, namespace aver.Namespace) {
	aver.Tree(cursor, namespace).
		Range_Uint(uint(cursor), uint(CURSOR_MIN), uint(CURSOR_MAX)).
		Ensure()
}

// Bound is the exclusive upper limit of a Chacha_Below draw: a positive count of outcomes.
type Bound uint64

// Bound_Invariants requires a bound to be positive and within the overflow-safe ceiling.
func Bound_Invariants(bound Bound, namespace aver.Namespace) {
	aver.Tree(bound, namespace).
		Range_Uint64(uint64(bound), uint64(BOUND_MIN), uint64(BOUND_MAX)).
		Ensure()
}

// Index is a Chacha_Below draw: a value in the half-open range zero to its bound.
type Index uint64

// Index_Invariants bounds a draw result.
func Index_Invariants(index Index, namespace aver.Namespace) {
	aver.Tree(index, namespace).
		Range_Uint64(uint64(index), uint64(INDEX_MIN), uint64(INDEX_MAX)).
		Ensure()
}

// Sink is a caller's buffer a draw fills — a defined type so the byte draw takes no raw slice.
type Sink []byte

// Sink_Invariants bounds a fill request's length.
func Sink_Invariants(sink Sink, namespace aver.Namespace) {
	aver.Tree(sink, namespace).
		Range_Int(len(sink), SINK_MIN, SINK_MAX).
		Ensure()
}

// Seed is one exact ChaCha20 key-width input.
type Seed []byte

// Seed_Invariants rejects partial or oversized construction input.
func Seed_Invariants(seed Seed, _ aver.Namespace) {
	aver.Always(len(seed) == KEY_BYTES, "A ChaCha seed has key width.")
}

// Count is bytes filled by one bounded Chacha_Read.
type Count int

// Count_Invariants follows the caller sink bound.
func Count_Invariants(count Count, namespace aver.Namespace) {
	aver.Tree(count, namespace).
		Range_Int(int(count), SINK_MIN, SINK_MAX).
		Ensure()
}

// Key_Source is one complete serialized ChaCha key.
type Key_Source []byte

// Key_Source_Invariants rejects partial key replacement.
func Key_Source_Invariants(source Key_Source, _ aver.Namespace) {
	aver.Always(len(source) == KEY_BYTES, "ChaCha key input has key width.")
}

// Key_Destination is exact scratch for serialized ChaCha key bytes.
type Key_Destination []byte

// Key_Destination_Invariants rejects partial key extraction.
func Key_Destination_Invariants(destination Key_Destination, _ aver.Namespace) {
	aver.Always(len(destination) == KEY_BYTES, "ChaCha key scratch has key width.")
}

// Buffer_Source is one complete serialized output buffer.
type Buffer_Source []byte

// Buffer_Source_Invariants rejects partial buffer replacement.
func Buffer_Source_Invariants(source Buffer_Source, _ aver.Namespace) {
	aver.Always(len(source) == BUFFER_BYTES, "ChaCha buffer input has refill width.")
}

// Buffer_Destination is exact scratch for serialized output bytes.
type Buffer_Destination []byte

// Buffer_Destination_Invariants rejects partial buffer extraction.
func Buffer_Destination_Invariants(destination Buffer_Destination, _ aver.Namespace) {
	aver.Always(len(destination) == BUFFER_BYTES, "ChaCha buffer scratch has refill width.")
}

// Nonce is one complete ChaCha20 nonce.
type Nonce []byte

// Nonce_Invariants fixes RFC 8439 nonce width.
func Nonce_Invariants(nonce Nonce, _ aver.Namespace) {
	aver.Always(len(nonce) == NONCE_BYTE_COUNT, "ChaCha nonce has RFC width.")
}

// Block_Destination is exact ChaCha20 block output storage.
type Block_Destination []byte

// Block_Destination_Invariants fixes RFC 8439 block width.
func Block_Destination_Invariants(destination Block_Destination, _ aver.Namespace) {
	aver.Always(
		len(destination) == CHACHA_BLOCK_BYTE_COUNT,
		"ChaCha block output has RFC width.",
	)
}

// State is exact mutable ChaCha20 word scratch.
type State []uint32

// State_Invariants fixes RFC 8439 state width.
func State_Invariants(state State, _ aver.Namespace) {
	aver.Always(len(state) == CHACHA_STATE_WORD_COUNT, "ChaCha state has RFC width.")
}

// Quarter_A_Index selects the first state lane in a quarter round.
type Quarter_A_Index uint8

// Quarter_A_Index_Invariants bounds the first lane to its state partition.
func Quarter_A_Index_Invariants(value Quarter_A_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value),
			QUARTER_A_INDEX_MINIMUM,
			QUARTER_A_INDEX_SECOND,
			QUARTER_A_INDEX_THIRD,
			QUARTER_A_INDEX_MAXIMUM,
		).
		Ensure()
}

// Quarter_B_Index selects the second state lane in a quarter round.
type Quarter_B_Index uint8

// Quarter_B_Index_Invariants bounds the second lane to its state partition.
func Quarter_B_Index_Invariants(value Quarter_B_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value),
			QUARTER_B_INDEX_MINIMUM,
			QUARTER_B_INDEX_SECOND,
			QUARTER_B_INDEX_THIRD,
			QUARTER_B_INDEX_MAXIMUM,
		).
		Ensure()
}

// Quarter_C_Index selects the third state lane in a quarter round.
type Quarter_C_Index uint8

// Quarter_C_Index_Invariants bounds the third lane to its state partition.
func Quarter_C_Index_Invariants(value Quarter_C_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value),
			QUARTER_C_INDEX_MINIMUM,
			QUARTER_C_INDEX_SECOND,
			QUARTER_C_INDEX_THIRD,
			QUARTER_C_INDEX_MAXIMUM,
		).
		Ensure()
}

// Quarter_D_Index selects the fourth state lane in a quarter round.
type Quarter_D_Index uint8

// Quarter_D_Index_Invariants bounds the fourth lane to its state partition.
func Quarter_D_Index_Invariants(value Quarter_D_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value),
			QUARTER_D_INDEX_MINIMUM,
			QUARTER_D_INDEX_SECOND,
			QUARTER_D_INDEX_THIRD,
			QUARTER_D_INDEX_MAXIMUM,
		).
		Ensure()
}

// Indices select the four independent state partitions coupled by a quarter round.
type Indices struct {
	// A selects the first partition.
	A Quarter_A_Index
	// B selects the second partition.
	B Quarter_B_Index
	// C selects the third partition.
	C Quarter_C_Index
	// D selects the fourth partition.
	D Quarter_D_Index
}

// Indices_Invariants composes every quarter-round lane partition.
func Indices_Invariants(indices Indices, namespace aver.Namespace) {
	Quarter_A_Index_Invariants(indices.A, namespace)
	Quarter_B_Index_Invariants(indices.B, namespace)
	Quarter_C_Index_Invariants(indices.C, namespace)
	Quarter_D_Index_Invariants(indices.D, namespace)
}

// Key_Lane_0 keeps first packed key word visible.
type Key_Lane_0 uint64

// Key_Lane_0_Invariants preserves complete word domain.
func Key_Lane_0_Invariants(value Key_Lane_0, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Key_Lane_1 keeps second packed key word visible.
type Key_Lane_1 uint64

// Key_Lane_1_Invariants preserves complete word domain.
func Key_Lane_1_Invariants(value Key_Lane_1, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Key_Lane_2 keeps third packed key word visible.
type Key_Lane_2 uint64

// Key_Lane_2_Invariants preserves complete word domain.
func Key_Lane_2_Invariants(value Key_Lane_2, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Key_Lane_3 keeps fourth packed key word visible.
type Key_Lane_3 uint64

// Key_Lane_3_Invariants preserves complete word domain.
func Key_Lane_3_Invariants(value Key_Lane_3, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Key packs caller-owned state without hidden array storage.
type Key struct {
	// Lane_0 avoids array-hidden state.
	Lane_0 Key_Lane_0
	// Lane_1 avoids array-hidden state.
	Lane_1 Key_Lane_1
	// Lane_2 avoids array-hidden state.
	Lane_2 Key_Lane_2
	// Lane_3 avoids array-hidden state.
	Lane_3 Key_Lane_3
}

// Key_Invariants exposes every packed key word.
func Key_Invariants(value Key, namespace aver.Namespace) {
	Key_Lane_0_Invariants(value.Lane_0, namespace)
	Key_Lane_1_Invariants(value.Lane_1, namespace)
	Key_Lane_2_Invariants(value.Lane_2, namespace)
	Key_Lane_3_Invariants(value.Lane_3, namespace)
}

// Key_Handle names mutable packed key state.
type Key_Handle *Key

// Key_Handle_Invariants composes key state when present.
func Key_Handle_Invariants(value Key_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Key_Invariants(*value, namespace)
}

// Buffer_Lane_0 keeps packed output word visible.
type Buffer_Lane_0 uint64

// Buffer_Lane_0_Invariants preserves complete word domain.
func Buffer_Lane_0_Invariants(value Buffer_Lane_0, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_1 keeps packed output word visible.
type Buffer_Lane_1 uint64

// Buffer_Lane_1_Invariants preserves complete word domain.
func Buffer_Lane_1_Invariants(value Buffer_Lane_1, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_2 keeps packed output word visible.
type Buffer_Lane_2 uint64

// Buffer_Lane_2_Invariants preserves complete word domain.
func Buffer_Lane_2_Invariants(value Buffer_Lane_2, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_3 keeps packed output word visible.
type Buffer_Lane_3 uint64

// Buffer_Lane_3_Invariants preserves complete word domain.
func Buffer_Lane_3_Invariants(value Buffer_Lane_3, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_4 keeps packed output word visible.
type Buffer_Lane_4 uint64

// Buffer_Lane_4_Invariants preserves complete word domain.
func Buffer_Lane_4_Invariants(value Buffer_Lane_4, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_5 keeps packed output word visible.
type Buffer_Lane_5 uint64

// Buffer_Lane_5_Invariants preserves complete word domain.
func Buffer_Lane_5_Invariants(value Buffer_Lane_5, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_6 keeps packed output word visible.
type Buffer_Lane_6 uint64

// Buffer_Lane_6_Invariants preserves complete word domain.
func Buffer_Lane_6_Invariants(value Buffer_Lane_6, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_7 keeps packed output word visible.
type Buffer_Lane_7 uint64

// Buffer_Lane_7_Invariants preserves complete word domain.
func Buffer_Lane_7_Invariants(value Buffer_Lane_7, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_8 keeps packed output word visible.
type Buffer_Lane_8 uint64

// Buffer_Lane_8_Invariants preserves complete word domain.
func Buffer_Lane_8_Invariants(value Buffer_Lane_8, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_9 keeps packed output word visible.
type Buffer_Lane_9 uint64

// Buffer_Lane_9_Invariants preserves complete word domain.
func Buffer_Lane_9_Invariants(value Buffer_Lane_9, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_10 keeps packed output word visible.
type Buffer_Lane_10 uint64

// Buffer_Lane_10_Invariants preserves complete word domain.
func Buffer_Lane_10_Invariants(value Buffer_Lane_10, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_11 keeps packed output word visible.
type Buffer_Lane_11 uint64

// Buffer_Lane_11_Invariants preserves complete word domain.
func Buffer_Lane_11_Invariants(value Buffer_Lane_11, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_12 keeps packed output word visible.
type Buffer_Lane_12 uint64

// Buffer_Lane_12_Invariants preserves complete word domain.
func Buffer_Lane_12_Invariants(value Buffer_Lane_12, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_13 keeps packed output word visible.
type Buffer_Lane_13 uint64

// Buffer_Lane_13_Invariants preserves complete word domain.
func Buffer_Lane_13_Invariants(value Buffer_Lane_13, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_14 keeps packed output word visible.
type Buffer_Lane_14 uint64

// Buffer_Lane_14_Invariants preserves complete word domain.
func Buffer_Lane_14_Invariants(value Buffer_Lane_14, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_15 keeps packed output word visible.
type Buffer_Lane_15 uint64

// Buffer_Lane_15_Invariants preserves complete word domain.
func Buffer_Lane_15_Invariants(value Buffer_Lane_15, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_16 keeps packed output word visible.
type Buffer_Lane_16 uint64

// Buffer_Lane_16_Invariants preserves complete word domain.
func Buffer_Lane_16_Invariants(value Buffer_Lane_16, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_17 keeps packed output word visible.
type Buffer_Lane_17 uint64

// Buffer_Lane_17_Invariants preserves complete word domain.
func Buffer_Lane_17_Invariants(value Buffer_Lane_17, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_18 keeps packed output word visible.
type Buffer_Lane_18 uint64

// Buffer_Lane_18_Invariants preserves complete word domain.
func Buffer_Lane_18_Invariants(value Buffer_Lane_18, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_19 keeps packed output word visible.
type Buffer_Lane_19 uint64

// Buffer_Lane_19_Invariants preserves complete word domain.
func Buffer_Lane_19_Invariants(value Buffer_Lane_19, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_20 keeps packed output word visible.
type Buffer_Lane_20 uint64

// Buffer_Lane_20_Invariants preserves complete word domain.
func Buffer_Lane_20_Invariants(value Buffer_Lane_20, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_21 keeps packed output word visible.
type Buffer_Lane_21 uint64

// Buffer_Lane_21_Invariants preserves complete word domain.
func Buffer_Lane_21_Invariants(value Buffer_Lane_21, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_22 keeps packed output word visible.
type Buffer_Lane_22 uint64

// Buffer_Lane_22_Invariants preserves complete word domain.
func Buffer_Lane_22_Invariants(value Buffer_Lane_22, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_23 keeps packed output word visible.
type Buffer_Lane_23 uint64

// Buffer_Lane_23_Invariants preserves complete word domain.
func Buffer_Lane_23_Invariants(value Buffer_Lane_23, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_24 keeps packed output word visible.
type Buffer_Lane_24 uint64

// Buffer_Lane_24_Invariants preserves complete word domain.
func Buffer_Lane_24_Invariants(value Buffer_Lane_24, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_25 keeps packed output word visible.
type Buffer_Lane_25 uint64

// Buffer_Lane_25_Invariants preserves complete word domain.
func Buffer_Lane_25_Invariants(value Buffer_Lane_25, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_26 keeps packed output word visible.
type Buffer_Lane_26 uint64

// Buffer_Lane_26_Invariants preserves complete word domain.
func Buffer_Lane_26_Invariants(value Buffer_Lane_26, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer_Lane_27 keeps packed output word visible.
type Buffer_Lane_27 uint64

// Buffer_Lane_27_Invariants preserves complete word domain.
func Buffer_Lane_27_Invariants(value Buffer_Lane_27, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Buffer packs one refill without hidden array storage.
type Buffer struct {
	// Lane_0 avoids array-hidden state.
	Lane_0 Buffer_Lane_0
	// Lane_1 avoids array-hidden state.
	Lane_1 Buffer_Lane_1
	// Lane_2 avoids array-hidden state.
	Lane_2 Buffer_Lane_2
	// Lane_3 avoids array-hidden state.
	Lane_3 Buffer_Lane_3
	// Lane_4 avoids array-hidden state.
	Lane_4 Buffer_Lane_4
	// Lane_5 avoids array-hidden state.
	Lane_5 Buffer_Lane_5
	// Lane_6 avoids array-hidden state.
	Lane_6 Buffer_Lane_6
	// Lane_7 avoids array-hidden state.
	Lane_7 Buffer_Lane_7
	// Lane_8 avoids array-hidden state.
	Lane_8 Buffer_Lane_8
	// Lane_9 avoids array-hidden state.
	Lane_9 Buffer_Lane_9
	// Lane_10 avoids array-hidden state.
	Lane_10 Buffer_Lane_10
	// Lane_11 avoids array-hidden state.
	Lane_11 Buffer_Lane_11
	// Lane_12 avoids array-hidden state.
	Lane_12 Buffer_Lane_12
	// Lane_13 avoids array-hidden state.
	Lane_13 Buffer_Lane_13
	// Lane_14 avoids array-hidden state.
	Lane_14 Buffer_Lane_14
	// Lane_15 avoids array-hidden state.
	Lane_15 Buffer_Lane_15
	// Lane_16 avoids array-hidden state.
	Lane_16 Buffer_Lane_16
	// Lane_17 avoids array-hidden state.
	Lane_17 Buffer_Lane_17
	// Lane_18 avoids array-hidden state.
	Lane_18 Buffer_Lane_18
	// Lane_19 avoids array-hidden state.
	Lane_19 Buffer_Lane_19
	// Lane_20 avoids array-hidden state.
	Lane_20 Buffer_Lane_20
	// Lane_21 avoids array-hidden state.
	Lane_21 Buffer_Lane_21
	// Lane_22 avoids array-hidden state.
	Lane_22 Buffer_Lane_22
	// Lane_23 avoids array-hidden state.
	Lane_23 Buffer_Lane_23
	// Lane_24 avoids array-hidden state.
	Lane_24 Buffer_Lane_24
	// Lane_25 avoids array-hidden state.
	Lane_25 Buffer_Lane_25
	// Lane_26 avoids array-hidden state.
	Lane_26 Buffer_Lane_26
	// Lane_27 avoids array-hidden state.
	Lane_27 Buffer_Lane_27
}

// Buffer_Invariants exposes every packed output word.
func Buffer_Invariants(value Buffer, namespace aver.Namespace) {
	Buffer_Lane_0_Invariants(value.Lane_0, namespace)
	Buffer_Lane_1_Invariants(value.Lane_1, namespace)
	Buffer_Lane_2_Invariants(value.Lane_2, namespace)
	Buffer_Lane_3_Invariants(value.Lane_3, namespace)
	Buffer_Lane_4_Invariants(value.Lane_4, namespace)
	Buffer_Lane_5_Invariants(value.Lane_5, namespace)
	Buffer_Lane_6_Invariants(value.Lane_6, namespace)
	Buffer_Lane_7_Invariants(value.Lane_7, namespace)
	Buffer_Lane_8_Invariants(value.Lane_8, namespace)
	Buffer_Lane_9_Invariants(value.Lane_9, namespace)
	Buffer_Lane_10_Invariants(value.Lane_10, namespace)
	Buffer_Lane_11_Invariants(value.Lane_11, namespace)
	Buffer_Lane_12_Invariants(value.Lane_12, namespace)
	Buffer_Lane_13_Invariants(value.Lane_13, namespace)
	Buffer_Lane_14_Invariants(value.Lane_14, namespace)
	Buffer_Lane_15_Invariants(value.Lane_15, namespace)
	Buffer_Lane_16_Invariants(value.Lane_16, namespace)
	Buffer_Lane_17_Invariants(value.Lane_17, namespace)
	Buffer_Lane_18_Invariants(value.Lane_18, namespace)
	Buffer_Lane_19_Invariants(value.Lane_19, namespace)
	Buffer_Lane_20_Invariants(value.Lane_20, namespace)
	Buffer_Lane_21_Invariants(value.Lane_21, namespace)
	Buffer_Lane_22_Invariants(value.Lane_22, namespace)
	Buffer_Lane_23_Invariants(value.Lane_23, namespace)
	Buffer_Lane_24_Invariants(value.Lane_24, namespace)
	Buffer_Lane_25_Invariants(value.Lane_25, namespace)
	Buffer_Lane_26_Invariants(value.Lane_26, namespace)
	Buffer_Lane_27_Invariants(value.Lane_27, namespace)
}

// Buffer_Handle names mutable packed output state.
type Buffer_Handle *Buffer

// Buffer_Handle_Invariants composes buffer state when present.
func Buffer_Handle_Invariants(value Buffer_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Buffer_Invariants(*value, namespace)
}

// Chacha is the state of a fast-key-erasure ChaCha20 keystream. Initialize caller storage before
// use. Its fields are transparent, like prng.Xoshiro's State: a Chacha is
// single-owner, so copying it forks the stream, and forward secrecy rests on rolling the key and
// zeroing delivered bytes, not on hiding these fields.
type Chacha struct {
	// Key is the current ChaCha20 key. Each refill overwrites it with fresh keystream, so the
	// seed and every earlier key vanish the moment their output is produced.
	Key Key
	// Buffer holds output from the last refill. Delivered bytes are zeroed in place, so
	// Buffer[:Position] is always zero and a disclosure cannot recover handed-out output.
	Buffer Buffer
	// Position is the next unread byte in Buffer; CURSOR_MAX means the buffer is spent and a
	// draw must refill.
	Position Cursor
}

// Chacha_Invariants states a Chacha's buffer position; its key and buffer are fixed-size
// arrays with no bundle of their own.
func Chacha_Invariants(generator Chacha, namespace aver.Namespace) {
	Key_Invariants(generator.Key, namespace)
	Buffer_Invariants(generator.Buffer, namespace)
	Cursor_Invariants(generator.Position, namespace)
}

// Chacha_Handle names one mutable single-owner generator.
type Chacha_Handle *Chacha

// Chacha_Handle_Invariants composes generator state when present.
func Chacha_Handle_Invariants(value Chacha_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Chacha_Invariants(*value, namespace)
}

func key_copy(source Key, destination Key_Destination) {
	Key_Invariants(source, "key_copy.source")
	Key_Destination_Invariants(destination, "key_copy.destination")
	lanes := [...]uint64{
		uint64(source.Lane_0), uint64(source.Lane_1),
		uint64(source.Lane_2), uint64(source.Lane_3),
	}
	for index, lane := range lanes {
		start := index * WORD_BYTE_COUNT
		for byte_index := 0; byte_index < WORD_BYTE_COUNT; byte_index++ {
			destination[start+byte_index] =
				byte(lane >> (byte_index * bits.BIT_COUNT_8_MAXIMUM))
		}
	}
}

func key_replace(destination Key_Handle, source Key_Source) {
	Key_Handle_Invariants(destination, "key_replace.destination")
	Key_Source_Invariants(source, "key_replace.source")
	var lanes [KEY_BYTES / WORD_BYTE_COUNT]uint64
	for index := range lanes {
		start := index * WORD_BYTE_COUNT
		for byte_index := WORD_BYTE_COUNT - 1; byte_index >= 0; byte_index-- {
			lanes[index] = lanes[index]<<bits.BIT_COUNT_8_MAXIMUM |
				uint64(source[start+byte_index])
		}
	}
	*destination = Key{
		Lane_0: Key_Lane_0(lanes[0]),
		Lane_1: Key_Lane_1(lanes[1]),
		Lane_2: Key_Lane_2(lanes[2]),
		Lane_3: Key_Lane_3(lanes[3]),
	}
}

func buffer_copy(source Buffer, destination Buffer_Destination) {
	Buffer_Invariants(source, "buffer_copy.source")
	Buffer_Destination_Invariants(destination, "buffer_copy.destination")
	lanes := [...]uint64{
		uint64(source.Lane_0), uint64(source.Lane_1),
		uint64(source.Lane_2), uint64(source.Lane_3),
		uint64(source.Lane_4), uint64(source.Lane_5),
		uint64(source.Lane_6), uint64(source.Lane_7),
		uint64(source.Lane_8), uint64(source.Lane_9),
		uint64(source.Lane_10), uint64(source.Lane_11),
		uint64(source.Lane_12), uint64(source.Lane_13),
		uint64(source.Lane_14), uint64(source.Lane_15),
		uint64(source.Lane_16), uint64(source.Lane_17),
		uint64(source.Lane_18), uint64(source.Lane_19),
		uint64(source.Lane_20), uint64(source.Lane_21),
		uint64(source.Lane_22), uint64(source.Lane_23),
		uint64(source.Lane_24), uint64(source.Lane_25),
		uint64(source.Lane_26), uint64(source.Lane_27),
	}
	for index, lane := range lanes {
		start := index * WORD_BYTE_COUNT
		for byte_index := 0; byte_index < WORD_BYTE_COUNT; byte_index++ {
			destination[start+byte_index] =
				byte(lane >> (byte_index * bits.BIT_COUNT_8_MAXIMUM))
		}
	}
}

func buffer_replace(destination Buffer_Handle, source Buffer_Source) {
	Buffer_Handle_Invariants(destination, "buffer_replace.destination")
	Buffer_Source_Invariants(source, "buffer_replace.source")
	var lanes [BUFFER_BYTES / WORD_BYTE_COUNT]uint64
	for index := range lanes {
		start := index * WORD_BYTE_COUNT
		for byte_index := WORD_BYTE_COUNT - 1; byte_index >= 0; byte_index-- {
			lanes[index] = lanes[index]<<bits.BIT_COUNT_8_MAXIMUM |
				uint64(source[start+byte_index])
		}
	}
	*destination = Buffer{
		Lane_0:  Buffer_Lane_0(lanes[0]),
		Lane_1:  Buffer_Lane_1(lanes[1]),
		Lane_2:  Buffer_Lane_2(lanes[2]),
		Lane_3:  Buffer_Lane_3(lanes[3]),
		Lane_4:  Buffer_Lane_4(lanes[4]),
		Lane_5:  Buffer_Lane_5(lanes[5]),
		Lane_6:  Buffer_Lane_6(lanes[6]),
		Lane_7:  Buffer_Lane_7(lanes[7]),
		Lane_8:  Buffer_Lane_8(lanes[8]),
		Lane_9:  Buffer_Lane_9(lanes[9]),
		Lane_10: Buffer_Lane_10(lanes[10]),
		Lane_11: Buffer_Lane_11(lanes[11]),
		Lane_12: Buffer_Lane_12(lanes[12]),
		Lane_13: Buffer_Lane_13(lanes[13]),
		Lane_14: Buffer_Lane_14(lanes[14]),
		Lane_15: Buffer_Lane_15(lanes[15]),
		Lane_16: Buffer_Lane_16(lanes[16]),
		Lane_17: Buffer_Lane_17(lanes[17]),
		Lane_18: Buffer_Lane_18(lanes[18]),
		Lane_19: Buffer_Lane_19(lanes[19]),
		Lane_20: Buffer_Lane_20(lanes[20]),
		Lane_21: Buffer_Lane_21(lanes[21]),
		Lane_22: Buffer_Lane_22(lanes[22]),
		Lane_23: Buffer_Lane_23(lanes[23]),
		Lane_24: Buffer_Lane_24(lanes[24]),
		Lane_25: Buffer_Lane_25(lanes[25]),
		Lane_26: Buffer_Lane_26(lanes[26]),
		Lane_27: Buffer_Lane_27(lanes[27]),
	}
}

// Chacha_Init seeds caller storage and performs the first fast-key-erasure refill, so the
// caller-supplied seed vanishes from live state before control returns.
func Chacha_Init(generator Chacha_Handle, seed Seed, position Cursor) {
	Chacha_Handle_Invariants(generator, "Chacha_Init.generator.input")
	Seed_Invariants(seed, "Chacha_Init.seed")
	Cursor_Invariants(position, "Chacha_Init.position")
	key_replace(Key_Handle(&generator.Key), Key_Source(seed))
	chacha_refill(generator)
	var buffer [BUFFER_BYTES]byte
	buffer_copy(generator.Buffer, Buffer_Destination(buffer[:]))
	for index := 0; index < int(position); index++ {
		buffer[index] = 0
	}
	buffer_replace(Buffer_Handle(&generator.Buffer), Buffer_Source(buffer[:]))
	generator.Position = position
	Cursor_Invariants(generator.Position, "Chacha_Init.generator.position.output")
}

// Chacha_Read fills one bounded caller sink completely.
func Chacha_Read(generator Chacha_Handle, sink Sink) (count Count) {
	defer func() { Count_Invariants(count, "Chacha_Read.count") }()
	Chacha_Handle_Invariants(generator, "Chacha_Read.generator")
	Sink_Invariants(sink, "Chacha_Read.sink")
	chacha_drain(generator, sink)
	return Count(len(sink))
}

// Word is one full draw through a Source slot.
type Word uint64

// Word_Invariants preserves the complete draw domain.
func Word_Invariants(word Word, namespace aver.Namespace) {
	aver.Tree(word, namespace).
		Range_Uint64(uint64(word), uint64(WORD_MINIMUM), uint64(WORD_MAXIMUM)).
		Ensure()
}

// Next is one injected full-word entropy draw.
type Next func(state unsafe.Pointer) (value Word)

// Source is the C-style vtable a caller injects where it needs cryptographic bytes: caller-owned
// backend state behind an unsafe.Pointer and one procedure that receives it. The house bans
// interfaces and closures that capture state, so this is the one shape a backend can take. A
// production root binds a Chacha through Chacha_To_Source. A simulation binds a xoshiro stream
// through sim/prng's Xoshiro_To_Source, and that call is the one place a fake enters, so a
// grep for it finds every test that signs with predictable bytes.
type Source struct {
	// State is the caller-owned backend generator; the procedure casts it back to its own type.
	State unsafe.Pointer
	// Next draws one full word from the backend behind state. The slot returns a value and
	// receives no sink, because a pointer handed to a procedure value escapes to the heap under
	// Go's escape analysis, and a caller's stack sink must stay on its stack.
	Next Next
}

// Source_Invariants proves both halves of the vtable are bound before any draw.
func Source_Invariants(source Source, _ aver.Namespace) {
	aver.Always(source.State != nil, "A Source has caller-owned state.")
	aver.Always(source.Next != nil, "A Source has a bound draw procedure.")
}

// Source_Read fills sink through the vtable, one word per eight bytes, little-endian so a shorter
// read is a prefix of a longer one from the same state. A partial tail spends a whole word. The
// bytes are packed here, on the caller's side of the slot, so the sink never crosses it.
func Source_Read(source Source, sink Sink) {
	Source_Invariants(source, "source_read.source")
	Sink_Invariants(sink, "source_read.sink")
	for filled := 0; filled < len(sink); filled += WORD_BYTE_COUNT {
		word := source.Next(source.State)
		byte_count := len(sink) - filled
		if byte_count > WORD_BYTE_COUNT {
			byte_count = WORD_BYTE_COUNT
		}
		for index := 0; index < byte_count; index++ {
			sink[filled+index] = byte(word >> (index * bits.BIT_COUNT_8_MAXIMUM))
		}
	}
}

// Chacha_To_Source binds caller-owned state into Source without a captured function
// environment. Only a production root calls this; a simulation binds a xoshiro stream instead.
func Chacha_To_Source(generator Chacha_Handle) (source Source) {
	defer func() { Source_Invariants(source, "chacha_to_source.source") }()
	Chacha_Handle_Invariants(generator, "chacha_to_source.generator")
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
	chacha_drain(Chacha_Handle((*Chacha)(state)), Sink(octet[:]))
	for index := WORD_BYTE_COUNT - 1; index >= 0; index-- {
		value = value<<bits.BIT_COUNT_8_MAXIMUM | Word(octet[index])
	}
	return value
}

// Chacha_Below returns a value in the half-open range zero to bound, never bound itself, using
// Lemire's method so the result is unbiased.
func Chacha_Below(generator Chacha_Handle, bound Bound) (index Index) {
	defer func() { Index_Invariants(index, "chacha_below.index") }()
	Chacha_Handle_Invariants(generator, "chacha_below.generator")
	Bound_Invariants(bound, "chacha_below.bound")
	limit := uint64(bound)
	// The raw draw stays a local: a free function returning unbounded entropy has no
	// witnessable invariant, so the word is read into bytes and assembled here, not returned.
	var octet [WORD_BYTE_COUNT]byte
	chacha_drain(generator, Sink(octet[:]))
	word := uint64(0)
	for byte_index := WORD_BYTE_COUNT - 1; byte_index >= 0; byte_index-- {
		word = word<<bits.BIT_COUNT_8_MAXIMUM | uint64(octet[byte_index])
	}
	high_word, low_word := bits.Multiply_64(
		bits.Word_64(word), bits.Multiplier_64(limit),
	)
	high, low := uint64(high_word), uint64(low_word)
	if low < limit {
		threshold := (-limit) % limit
		for low < threshold {
			chacha_drain(generator, Sink(octet[:]))
			word = 0
			for byte_index := WORD_BYTE_COUNT - 1; byte_index >= 0; byte_index-- {
				word = word<<bits.BIT_COUNT_8_MAXIMUM | uint64(octet[byte_index])
			}
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
func chacha_drain(generator Chacha_Handle, destination Sink) {
	Chacha_Handle_Invariants(generator, "chacha_drain.generator")
	Sink_Invariants(destination, "chacha_drain.destination")
	var buffer [BUFFER_BYTES]byte
	buffer_copy(generator.Buffer, Buffer_Destination(buffer[:]))
	filled := 0
	for filled < len(destination) {
		if generator.Position == CURSOR_MAX {
			chacha_refill(generator)
			buffer_copy(generator.Buffer, Buffer_Destination(buffer[:]))
		}
		position := int(generator.Position)
		take := BUFFER_BYTES - position
		if remainder := len(destination) - filled; remainder < take {
			take = remainder
		}
		source := buffer[position : position+take]
		copy(destination[filled:filled+take], source)
		for index := 0; index < take; index++ {
			source[index] = 0
		}
		generator.Position = Cursor(position + take)
		filled += take
	}
	buffer_replace(Buffer_Handle(&generator.Buffer), Buffer_Source(buffer[:]))
}

// Runs the ChaCha20 block function REFILL_BLOCKS times over the current key, reseeds the key from
// the first 32 output bytes, and stores the remaining 224 as the buffer. Nonce is always zero and
// the counter only runs its refill range: safe against reuse because each key expands exactly one
// 256-byte block and is then discarded, so no (key, nonce, counter) triple ever repeats.
func chacha_refill(generator Chacha_Handle) {
	Chacha_Handle_Invariants(generator, "chacha_refill.generator")
	var stream [REFILL_BYTES]byte
	var block [CHACHA_BLOCK_BYTE_COUNT]byte
	var key [KEY_BYTES]byte
	key_copy(generator.Key, Key_Destination(key[:]))
	var nonce [NONCE_BYTE_COUNT]byte
	for block_index := 0; block_index < REFILL_BLOCKS; block_index++ {
		chacha20_block(
			Key_Source(key[:]),
			Block_Counter(block_index),
			Nonce(nonce[:]),
			Block_Destination(block[:]),
		)
		copy(stream[block_index*CHACHA_BLOCK_BYTE_COUNT:], block[:])
	}
	key_replace(Key_Handle(&generator.Key), Key_Source(stream[:KEY_BYTES]))
	buffer_replace(Buffer_Handle(&generator.Buffer), Buffer_Source(stream[KEY_BYTES:]))
	generator.Position = 0
}

// Computes one 64-byte ChaCha20 keystream block for the key, block counter, and nonce, writing
// it to output (RFC 8439 2.3). The 16-word state is the four constants, the eight key words, the
// counter, and the three nonce words, all little-endian; twenty rounds (ten column-and-diagonal
// double rounds) mix a scratch copy, which is added back to the original and serialized.
func chacha20_block(
	key Key_Source,
	counter Block_Counter,
	nonce Nonce,
	output Block_Destination,
) {
	Key_Source_Invariants(key, "chacha20_block.key")
	Block_Counter_Invariants(counter, "chacha20_block.counter")
	Nonce_Invariants(nonce, "chacha20_block.nonce")
	Block_Destination_Invariants(output, "chacha20_block.output")
	var state_storage [CHACHA_STATE_WORD_COUNT]uint32
	state := State(state_storage[:])
	state[0] = CHACHA_CONSTANT_FIRST
	state[1] = CHACHA_CONSTANT_SECOND
	state[2] = CHACHA_CONSTANT_THIRD
	state[3] = CHACHA_CONSTANT_FOURTH
	// Key and nonce words assemble little-endian by hand: encoding/binary reaches
	// sim/prng through nbio, and sim/prng imports this package to bind a Xoshiro.
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
	var scratch_storage [CHACHA_STATE_WORD_COUNT]uint32
	copy(scratch_storage[:], state)
	scratch := State(scratch_storage[:])
	for round_index := 0; round_index < 10; round_index++ {
		quarter_round(scratch, Indices{A: 0, B: 4, C: 8, D: 12})
		quarter_round(scratch, Indices{A: 1, B: 5, C: 9, D: 13})
		quarter_round(scratch, Indices{A: 2, B: 6, C: 10, D: 14})
		quarter_round(scratch, Indices{A: 3, B: 7, C: 11, D: 15})
		quarter_round(scratch, Indices{A: 0, B: 5, C: 10, D: 15})
		quarter_round(scratch, Indices{A: 1, B: 6, C: 11, D: 12})
		quarter_round(scratch, Indices{A: 2, B: 7, C: 8, D: 13})
		quarter_round(scratch, Indices{A: 3, B: 4, C: 9, D: 14})
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
	state State,
	indices Indices,
) {
	State_Invariants(state, "quarter_round.state")
	Indices_Invariants(indices, "quarter_round.indices")
	a := int(indices.A)
	b := int(indices.B)
	c := int(indices.C)
	d := int(indices.D)
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
