// Package prng is the composition tier: the operating-system-seeded generator. It declares
// package prng so callers import ".../crypto/prng/default" and read it as the library with no
// alias, while the composition root retains the operating-system entropy reader.
package prng

import (
	"local/james-orcales/shared/crypto/prng"

	"local/james-orcales/shared/sim/aver/default"
)

// SEED_BYTE_COUNT lets root storage match one complete ChaCha20 key.
const SEED_BYTE_COUNT = 32

// Seed_Read keeps OS entropy ownership at the caller's composition root.
type Seed_Read func(destination []byte) (count int, err error)

// Chacha_Init receives seed storage because scratch passed through an injected reader escapes.
func Chacha_Init(
	generator prng.Chacha_Handle, read Seed_Read, seed prng.Seed, position prng.Cursor,
) {
	prng.Chacha_Handle_Invariants(generator, "Chacha_Init.generator.input")
	prng.Seed_Invariants(seed, "Chacha_Init.seed")
	prng.Cursor_Invariants(position, "Chacha_Init.position")
	aver.Always(read != nil, "operating system entropy reader is injected")
	count, read_error := read(seed)
	aver.Always(read_error == nil, "operating system entropy read succeeds")
	aver.Always(count == len(seed), "operating system entropy fills the complete seed")
	prng.Chacha_Init(generator, seed, position)
}
