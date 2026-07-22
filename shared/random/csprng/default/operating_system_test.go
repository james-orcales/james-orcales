package csprng_test

import (
	"testing"

	invariant "local/james-orcales/shared/invariant/default"
	os_csprng "local/james-orcales/shared/random/csprng/default"
)

// TestMain registers this package's assertions with the invariant coverage recorder and reports any
// gaps after the suite runs.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}

// Test_Operating_System_Smoke checks the OS-seeded constructor yields a working, non-degenerate
// generator. It cannot assert an exact sequence — the seed is real OS entropy — so it checks two
// constructions diverge and that output is not stuck at zero. The deterministic sequence check is
// the library suite's job (Known_Sequence), as time/default is only smoke-tested too.
func Test_Operating_System_Smoke(t *testing.T) {
	first := os_csprng.New_Operating_System_Generator()
	second := os_csprng.New_Operating_System_Generator()

	// Two generators seeded from independent OS entropy colliding on their first eight bytes is
	// a 1-in-2^64 event; treat equality as a wiring failure (both from the same seed).
	var first_draw, second_draw [8]byte
	first.Read(first_draw[:])
	second.Read(second_draw[:])
	if first_draw == second_draw {
		t.Fatalf("two OS-seeded generators drew the same first bytes; seeding is not wired")
	}

	var sample [32]byte
	first.Read(sample[:])
	nonzero := false
	for _, octet := range sample {
		if octet != 0 {
			nonzero = true
		}
	}
	if !nonzero {
		t.Fatalf("OS-seeded generator produced only zeros")
	}
}
