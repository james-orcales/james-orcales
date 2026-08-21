// Package prng is the composition tier: the operating-system-seeded generator. It declares
// package prng so callers import ".../crypto/prng/default" and read it as the library with no
// alias, while the composition root retains the operating-system entropy reader.
package prng

import (
	"local/james-orcales/shared/crypto/prng"

	"local/james-orcales/shared/simulation/aver/default"
)

// SEED_BYTE_COUNT gives ChaCha20 the full key width from one operating-system read.
const SEED_BYTE_COUNT = 32

// Seed_Read keeps OS entropy ownership at the caller's composition root.
type Seed_Read func(destination []byte) (count int, err error)

// New_Operating_System_Chacha requires the root-owned source to fill one complete seed.
func New_Operating_System_Chacha(
	read Seed_Read, position prng.Cursor,
) (generator prng.Chacha) {
	defer func() {
		prng.Chacha_Invariants(generator, "new_operating_system_chacha.generator")
	}()
	prng.Cursor_Invariants(position, "new_operating_system_chacha.position")
	aver.Always(read != nil, "operating system entropy reader is injected")
	var seed [SEED_BYTE_COUNT]byte
	count, read_error := read(seed[:])
	aver.Always(read_error == nil, "operating system entropy read succeeds")
	aver.Always(count == len(seed), "operating system entropy fills the complete seed")
	return prng.New(seed, position)
}
