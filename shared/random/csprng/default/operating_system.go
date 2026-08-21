// Package csprng is the composition tier: the operating-system-seeded generator. It declares
// package csprng so callers import ".../csprng/default" and read it as the library with no alias,
// while the composition root retains the operating-system entropy reader.
package csprng

import (
	"local/james-orcales/shared/random/csprng"

	"local/james-orcales/shared/invariant/default"
)

// SEED_BYTE_COUNT gives ChaCha20 the full key width from one operating-system read.
const SEED_BYTE_COUNT = 32

// Seed_Read keeps OS entropy ownership at the caller's composition root.
type Seed_Read func(destination []byte) (count int, err error)

// New_Operating_System_Generator requires the root-owned source to fill one complete seed.
func New_Operating_System_Generator(
	read Seed_Read, position csprng.Cursor,
) (generator csprng.Generator) {
	defer func() {
		csprng.Generator_Invariants(generator, "new_operating_system_generator.generator")
	}()
	csprng.Cursor_Invariants(position, "new_operating_system_generator.position")
	invariant.Always(read != nil, "operating system entropy reader is injected")
	var seed [SEED_BYTE_COUNT]byte
	count, read_error := read(seed[:])
	invariant.Always(read_error == nil, "operating system entropy read succeeds")
	invariant.Always(count == len(seed), "operating system entropy fills the complete seed")
	return csprng.New(seed, position)
}
