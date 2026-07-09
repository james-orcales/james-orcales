package csprng_test

import (
	"testing"

	"local/james-orcales/shared/csprng"

	os_csprng "local/james-orcales/shared/csprng/default"
)

// Test_Operating_System_Smoke checks the OS-seeded constructor yields a working, non-degenerate
// generator. It cannot assert an exact sequence — the seed is real OS entropy — so it checks that
// two independent constructions diverge (different seeds) and that output is not stuck at zero. A
// deterministic exact-sequence check is the library suite's job (Known_Sequence), mirroring how
// time/default is only smoke-tested while the virtual clock gets exact assertions.
func Test_Operating_System_Smoke(t *testing.T) {
	first := os_csprng.New_Operating_System_Generator()
	second := os_csprng.New_Operating_System_Generator()

	// Two generators seeded from independent OS entropy colliding on their first draw is a
	// 1-in-2^64 event; treat equality as a wiring failure (both seeded from the same constant).
	if csprng.Generator_Uint64(&first) == csprng.Generator_Uint64(&second) {
		t.Fatalf("two OS-seeded generators drew the same first value; seeding is not wired")
	}

	nonzero := false
	for draw_index := 0; draw_index < 8; draw_index++ {
		if csprng.Generator_Uint64(&first) != 0 {
			nonzero = true
		}
	}
	if !nonzero {
		t.Fatalf("OS-seeded generator produced only zeros")
	}
}
