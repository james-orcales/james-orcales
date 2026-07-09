package uuid_test

import (
	"local/james-orcales/shared/prng"
	"local/james-orcales/shared/time"
	"local/james-orcales/shared/uuid"
)

// FIXED_EPOCH_SECONDS is the Unix second the fixed generator's virtual clock reads,
// so a Version 7 timestamp round-trips to a known value.
const FIXED_EPOCH_SECONDS = 1_700_000_000

// Deterministic_Reader is an io.Reader whose bytes come from a seeded prng, standing
// in for crypto/rand so a test's UUIDs are a pure function of the seed.
type Deterministic_Reader struct {
	// Generator is the seeded xoshiro stream the bytes are drawn from.
	Generator prng.Generator
	// Word is the current 64-bit draw being dispensed one byte at a time.
	Word uint64
	// Available is how many bytes of Word remain undispensed.
	Available int
}

// Read fills buffer from the prng, satisfying io.Reader; it never returns an error.
func (reader *Deterministic_Reader) Read(buffer []byte) (count int, err error) {
	for index := range buffer {
		if reader.Available == 0 {
			reader.Word = prng.Generator_Next(&reader.Generator)
			reader.Available = 8
		}
		buffer[index] = byte(reader.Word)
		reader.Word >>= 8
		reader.Available--
	}
	return len(buffer), nil
}

// Builds a Generator whose entropy is a seeded prng and whose clock is a frozen
// virtual clock at FIXED_EPOCH_SECONDS, so every draw is reproducible.
func fixed_generator(seed uint64) (generator uuid.Generator) {
	reader := &Deterministic_Reader{Generator: prng.New(seed)}
	clock, _ := time.Virtual_Clock_To_Clock(time.Virtual_Clock{
		Resolution: time.MILLISECOND,
		Epoch:      time.Moment(FIXED_EPOCH_SECONDS * int64(time.SECOND)),
	})
	return uuid.New(reader, clock, [6]byte{1, 2, 3, 4, 5, 6})
}
