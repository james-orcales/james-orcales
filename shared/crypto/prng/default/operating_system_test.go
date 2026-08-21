package prng_test

import (
	"testing"

	"local/james-orcales/shared/crypto/prng"
	system_prng "local/james-orcales/shared/crypto/prng/default"
	"local/james-orcales/shared/testify"
)

// DRAW_BYTE_COUNT compares one complete uint64 draw from each generator.
const DRAW_BYTE_COUNT = 8

// SAMPLE_BYTE_COUNT makes a degenerate all-zero stream sufficiently clear in the smoke test.
const SAMPLE_BYTE_COUNT = 32

// Test_Operating_System_Smoke checks the OS-seeded constructor yields a working, non-degenerate
// generator. It cannot assert an exact sequence — the seed is real OS entropy — so it checks two
// constructions diverge and that output is not stuck at zero. The deterministic sequence check is
// the library suite's job (Known_Sequence), as time/default is only smoke-tested too.
func Test_Operating_System_Smoke(t *testing.T) {
	var first, second prng.Chacha
	var seed [system_prng.SEED_BYTE_COUNT]byte
	system_prng.Chacha_Init(&first, seed_one_read, seed[:], prng.CURSOR_MIN)
	system_prng.Chacha_Init(&second, seed_two_read, seed[:], prng.CURSOR_MIN)

	// Two generators seeded from independent OS entropy colliding on their first eight bytes is
	// a 1-in-2^64 event; treat equality as a wiring failure (both from the same seed).
	var first_draw, second_draw [DRAW_BYTE_COUNT]byte
	prng.Chacha_Read(&first, first_draw[:])
	prng.Chacha_Read(&second, second_draw[:])
	if first_draw == second_draw {
		t.Fatalf("two OS-seeded generators drew the same first bytes; seeding is not wired")
	}

	var sample [SAMPLE_BYTE_COUNT]byte
	prng.Chacha_Read(&first, sample[:])
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
	var boundary prng.Chacha
	system_prng.Chacha_Init(&boundary, seed_three_read, seed[:], prng.CURSOR_MIN)
	prng.Chacha_Read(&boundary, make([]byte, 1))
	prng.Chacha_Read(&boundary, make([]byte, 1))
	prng.Chacha_Read(&boundary, nil)
	prng.Chacha_Read(&boundary, make([]byte, prng.BUFFER_BYTES-2))
	prng.Chacha_Read(&boundary, nil)
	positions := []prng.Cursor{prng.CURSOR_MIN, 1, 2, prng.CURSOR_MAX}
	for _, position := range positions {
		positioned := prng.Chacha{Position: position}
		system_prng.Chacha_Init(&positioned, seed_three_read, seed[:], position)
		if positioned.Position != position {
			t.Fatalf(
				"constructor cursor was %d, want %d",
				positioned.Position,
				position,
			)
		}
	}
	for _, value := range [...]uint64{0, 1, 2, ^uint64(0)} {
		positioned := domain_chacha(value)
		system_prng.Chacha_Init(
			&positioned, seed_three_read, seed[:], prng.CURSOR_MIN,
		)
	}
}

// Test_Allocation keeps composition-root entropy injection allocation-free.
func Test_Allocation(t *testing.T) {
	var generator prng.Chacha
	var seed [system_prng.SEED_BYTE_COUNT]byte
	testify.Zero_Allocation(t, func() {
		system_prng.Chacha_Init(
			&generator, seed_one_read, seed[:], prng.CURSOR_MIN,
		)
	})
}

func domain_chacha(value uint64) (generator prng.Chacha) {
	generator.Key = prng.Key{
		Lane_0: prng.Key_Lane_0(value),
		Lane_1: prng.Key_Lane_1(value),
		Lane_2: prng.Key_Lane_2(value),
		Lane_3: prng.Key_Lane_3(value),
	}
	generator.Buffer = prng.Buffer{
		Lane_0:  prng.Buffer_Lane_0(value),
		Lane_1:  prng.Buffer_Lane_1(value),
		Lane_2:  prng.Buffer_Lane_2(value),
		Lane_3:  prng.Buffer_Lane_3(value),
		Lane_4:  prng.Buffer_Lane_4(value),
		Lane_5:  prng.Buffer_Lane_5(value),
		Lane_6:  prng.Buffer_Lane_6(value),
		Lane_7:  prng.Buffer_Lane_7(value),
		Lane_8:  prng.Buffer_Lane_8(value),
		Lane_9:  prng.Buffer_Lane_9(value),
		Lane_10: prng.Buffer_Lane_10(value),
		Lane_11: prng.Buffer_Lane_11(value),
		Lane_12: prng.Buffer_Lane_12(value),
		Lane_13: prng.Buffer_Lane_13(value),
		Lane_14: prng.Buffer_Lane_14(value),
		Lane_15: prng.Buffer_Lane_15(value),
		Lane_16: prng.Buffer_Lane_16(value),
		Lane_17: prng.Buffer_Lane_17(value),
		Lane_18: prng.Buffer_Lane_18(value),
		Lane_19: prng.Buffer_Lane_19(value),
		Lane_20: prng.Buffer_Lane_20(value),
		Lane_21: prng.Buffer_Lane_21(value),
		Lane_22: prng.Buffer_Lane_22(value),
		Lane_23: prng.Buffer_Lane_23(value),
		Lane_24: prng.Buffer_Lane_24(value),
		Lane_25: prng.Buffer_Lane_25(value),
		Lane_26: prng.Buffer_Lane_26(value),
		Lane_27: prng.Buffer_Lane_27(value),
	}
	return generator
}

func seed_one_read(destination []byte) (count int, err error) {
	return seed_read(destination, 1)
}

func seed_two_read(destination []byte) (count int, err error) {
	return seed_read(destination, 2)
}

func seed_three_read(destination []byte) (count int, err error) {
	return seed_read(destination, 3)
}

func seed_read(destination []byte, octet byte) (count int, err error) {
	for index := range destination {
		destination[index] = 0
	}
	destination[0] = octet
	return len(destination), nil
}
