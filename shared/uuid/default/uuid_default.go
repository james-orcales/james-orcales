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
	"local/james-orcales/shared/simulation/time/default"
	"local/james-orcales/shared/uuid"
)

// New_Operating_System_Generator returns a Generator wired to the host: crypto/rand
// for entropy and the operating-system clock for the version 1, 6, and 7
// timestamps. The node is random, drawn from crypto/rand when first needed.
func New_Operating_System_Generator(
	read system_prng.Seed_Read,
	source *prng.Chacha,
	state uuid.Generator_State,
) (generator uuid.Generator) {
	defer func() {
		uuid.Generator_Invariants(generator, "new_operating_system_generator.generator")
	}()
	prng.Chacha_Invariants(*source, "new_operating_system_generator.source")
	uuid.Generator_State_Invariants(state, "new_operating_system_generator.state")
	*source = system_prng.New_Operating_System_Chacha(read, prng.CURSOR_MIN)
	clock := time.New_Operating_System_Clock()
	return uuid.New(prng.Chacha_To_Source(source), clock, uuid.Node{}, state)
}
