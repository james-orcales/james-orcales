// Package uuid is the composition tier for shared/uuid: it wires the host's
// entropy and clock into a Generator. It declares package uuid so callers import
// ".../uuid/default" and read it as the library with no alias, mirroring time/default.
//
// The node is left zero, so the Generator draws a random node from crypto/rand on
// the first V1 or V6 draw. Reading the host MAC through net.Interfaces is avoided
// on purpose: the io gateway bans it outside shared/io, and a random node also
// sidesteps the MAC-address privacy leak that has long dogged UUID version 1.
package uuid

import (
	"local/james-orcales/shared/crypto/prng"
	system_prng "local/james-orcales/shared/crypto/prng/default"
	"local/james-orcales/shared/sim/aver/default"
	sim_time "local/james-orcales/shared/sim/time"
	"local/james-orcales/shared/sim/time/default"
	"local/james-orcales/shared/uuid"
)

// Generator is host constructor output before caller claims mutable ownership.
type Generator uuid.Generator

// Generator_Invariants fixes host-selected random node state while composing replay state.
func Generator_Invariants(value Generator, namespace aver.Namespace) {
	prng.Source_Invariants(value.Source, namespace)
	sim_time.Clock_Invariants(value.Clock, namespace)
	aver.Always(value.Node == (uuid.Node{}), "Operating-system UUID node starts unresolved.")
	uuid.Clock_Sequence_State_Invariants(value.Clock_Sequence, namespace)
	uuid.Timestamp_State_Invariants(value.Last_Time, namespace)
	uuid.V7_State_Invariants(value.Last_V7, namespace)
}

// Source_Pointer names caller-owned operating-system-seeded state.
type Source_Pointer *prng.Chacha

// Source_Pointer_Invariants composes present source storage.
func Source_Pointer_Invariants(value Source_Pointer, namespace aver.Namespace) {
	if value == nil {
		return
	}
	prng.Chacha_Invariants(*value, namespace)
}

// New_Operating_System_Generator returns a Generator wired to the host: crypto/rand
// for entropy and the operating-system clock for the version 1, 6, and 7
// timestamps. The node is random, drawn from crypto/rand when first needed.
func New_Operating_System_Generator(
	read system_prng.Seed_Read, seed prng.Seed,
	source Source_Pointer,
	state uuid.Generator_State,
) (generator Generator) {
	defer func() {
		Generator_Invariants(generator, "new_operating_system_generator.generator")
	}()
	prng.Seed_Invariants(seed, "new_operating_system_generator.seed")
	Source_Pointer_Invariants(source, "new_operating_system_generator.source")
	uuid.Generator_State_Invariants(state, "new_operating_system_generator.state")
	aver.Always(read != nil, "Operating-system entropy reader is injected.")
	count, read_error := read(seed)
	aver.Always(read_error == nil, "Operating-system entropy read succeeds.")
	aver.Always(count == len(seed), "Operating-system entropy fills complete seed.")
	prng.Chacha_Init(prng.Chacha_Handle(source), seed, prng.CURSOR_MIN)
	clock := time.New_Operating_System_Clock()
	return Generator(uuid.New(
		prng.Chacha_To_Source(prng.Chacha_Handle(source)), clock, uuid.Node{}, state,
	))
}
