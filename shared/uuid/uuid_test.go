package uuid_test

import (
	"testing"

	"local/james-orcales/shared/crypto/prng"
	"local/james-orcales/shared/simulation/aver/default"
	"local/james-orcales/shared/simulation/time"
	"local/james-orcales/shared/uuid"
)

// TestMain register UUID invariant roots before specification run.
func TestMain(m *testing.M) {
	aver.Run_Test_Main(m)
}

// FIXED_EPOCH_SECONDS is the Unix second the fixed generator's virtual clock reads,
// so a Version 7 timestamp round-trips to a known value.
const FIXED_EPOCH_SECONDS = 1_700_000_000

// Builds a Generator whose entropy is a seeded CSPRNG and whose clock is a frozen
// virtual clock at FIXED_EPOCH_SECONDS, so every draw is reproducible.
func fixed_generator(seed uint64) (generator uuid.Generator) {
	return generator_at(
		seed, time.Moment(FIXED_EPOCH_SECONDS*int64(time.SECOND)),
		uuid.Node{1, 2, 3, 4, 5, 6}, uuid.Generator_State{},
	)
}

// Boundary clocks and replay state need the same deterministic entropy wiring as
// ordinary examples, or invariant witnesses would accidentally test another root.
func generator_at(
	seed uint64, moment time.Moment, node uuid.Node, state uuid.Generator_State,
) (generator uuid.Generator) {
	var seed_bytes [prng.KEY_BYTES]byte
	for index := range prng.WORD_BYTE_COUNT {
		seed_bytes[index] = byte(seed >> (index * prng.WORD_BYTE_COUNT))
	}
	source := new(prng.Chacha)
	*source = prng.New(seed_bytes, prng.CURSOR_MIN)
	virtual := time.Virtual_Clock{
		Resolution: time.MILLISECOND,
		Epoch:      moment,
	}
	clock := time.Virtual_Clock_To_Clock(&virtual)
	return uuid.New(prng.Chacha_To_Source(source), clock, node, state)
}
