// Package csprng is the composition tier: the operating-system-seeded generator. It declares
// package csprng so callers import ".../csprng/default" and read it as the library with no alias,
// and it is the one place in the tree permitted to read crypto/rand — the linter bans that import
// in the library tier.
package csprng

import (
	"crypto/rand"

	"local/james-orcales/shared/random/csprng"

	invariant "local/james-orcales/shared/invariant/default"
)

// SEED_BYTE_COUNT gives ChaCha20 the full key width from one operating-system read.
const SEED_BYTE_COUNT = 32

// New_Operating_System_Generator seeds a Generator from the host operating system's CSPRNG. It is
// the sole sanctioned caller of crypto/rand in the tree. crypto/rand.Read (Go 1.24 and later) fills
// its buffer fully or crashes the process, so OS entropy is infallible and there is no error to
// return; the assertion documents that contract and trips loudly if a future runtime breaks it.
func New_Operating_System_Generator(
	position csprng.Cursor,
) (generator csprng.Generator) {
	defer func() {
		csprng.Generator_Invariants(generator, "new_operating_system_generator.generator")
	}()
	csprng.Cursor_Invariants(position, "new_operating_system_generator.position")
	var seed [SEED_BYTE_COUNT]byte
	_, read_error := rand.Read(seed[:])
	invariant.Always(read_error == nil, "operating system entropy read succeeds")
	return csprng.New(seed, position)
}
