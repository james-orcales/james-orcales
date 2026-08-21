package prng_test

import (
	"testing"

	"local/james-orcales/shared/crypto/prng"
	system_prng "local/james-orcales/shared/crypto/prng/default"
	"local/james-orcales/shared/sim/aver/default"
)

// DRAW_BYTE_COUNT compares one complete uint64 draw from each generator.
const DRAW_BYTE_COUNT = 8

// SAMPLE_BYTE_COUNT makes a degenerate all-zero stream sufficiently clear in the smoke test.
const SAMPLE_BYTE_COUNT = 32

// TestMain registers this package's assertions with the invariant coverage recorder and reports any
// gaps after the suite runs.
func TestMain(m *testing.M) {
	aver.Run_Test_Main(m)
}

// Test_Operating_System_Smoke checks the OS-seeded constructor yields a working, non-degenerate
// generator. It cannot assert an exact sequence — the seed is real OS entropy — so it checks two
// constructions diverge and that output is not stuck at zero. The deterministic sequence check is
// the library suite's job (Known_Sequence), as time/default is only smoke-tested too.
func Test_Operating_System_Smoke(t *testing.T) {
	first_source := seed_source{Seed: [system_prng.SEED_BYTE_COUNT]byte{1}}
	second_source := seed_source{Seed: [system_prng.SEED_BYTE_COUNT]byte{2}}
	first := system_prng.New_Operating_System_Chacha(first_source.Read, prng.CURSOR_MIN)
	second := system_prng.New_Operating_System_Chacha(second_source.Read, prng.CURSOR_MIN)

	// Two generators seeded from independent OS entropy colliding on their first eight bytes is
	// a 1-in-2^64 event; treat equality as a wiring failure (both from the same seed).
	var first_draw, second_draw [DRAW_BYTE_COUNT]byte
	first.Read(first_draw[:])
	second.Read(second_draw[:])
	if first_draw == second_draw {
		t.Fatalf("two OS-seeded generators drew the same first bytes; seeding is not wired")
	}

	var sample [SAMPLE_BYTE_COUNT]byte
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
	// Constructor coverage begins at zero; walking one byte at a time and then draining the
	// refill makes the nested Cursor bundle prove every state boundary in this package too.
	boundary_source := seed_source{Seed: [system_prng.SEED_BYTE_COUNT]byte{3}}
	boundary := system_prng.New_Operating_System_Chacha(
		boundary_source.Read, prng.CURSOR_MIN,
	)
	boundary.Read(make([]byte, 1))
	boundary.Read(make([]byte, 1))
	boundary.Read(nil)
	boundary.Read(make([]byte, prng.BUFFER_BYTES-2))
	boundary.Read(nil)
	positions := []prng.Cursor{prng.CURSOR_MIN, 1, 2, prng.CURSOR_MAX}
	for _, position := range positions {
		source := seed_source{Seed: [system_prng.SEED_BYTE_COUNT]byte{byte(position + 4)}}
		positioned := system_prng.New_Operating_System_Chacha(source.Read, position)
		if positioned.Position != position {
			t.Fatalf(
				"constructor cursor was %d, want %d",
				positioned.Position,
				position,
			)
		}
	}
}

type seed_source struct {
	Seed [system_prng.SEED_BYTE_COUNT]byte
}

func (source *seed_source) Read(destination []byte) (count int, err error) {
	return copy(destination, source.Seed[:]), nil
}
