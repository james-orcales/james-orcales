package uuid_test

import (
	"testing"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/crypto/prng"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/testify"
	"local/james-orcales/shared/uuid"
	system_uuid "local/james-orcales/shared/uuid/default"
)

// TestMain register operating-system UUID invariant roots before smoke run.
func TestMain(m *testing.M) {
	aver.Run_Test_Main(m)
}

// Test_Operating_System_Generator_Smoke checks the host-wired generator mints valid,
// distinct V4 and V7 UUIDs. Real entropy is non-deterministic, so this is a smoke test.
func Test_Operating_System_Generator_Smoke(t *testing.T) {
	var source prng.Chacha
	seed := make([]byte, prng.KEY_BYTES)
	seed_storage := make([]byte, prng.KEY_BYTES)
	seed[0] = 1
	entropy := func(destination []byte) (count int, err error) {
		return copy(destination, seed), nil
	}
	constructed := system_uuid.New_Operating_System_Generator(
		entropy, prng.Seed(seed_storage), &source, uuid.Generator_State{},
	)
	generator := uuid.Generator(constructed)
	var first_storage [uuid.UUID_BYTE_COUNT]byte
	var second_storage [uuid.UUID_BYTE_COUNT]byte
	first := uuid.Generator_V4(&generator, uuid.UUID(first_storage[:]))
	second := uuid.Generator_V4(&generator, uuid.UUID(second_storage[:]))
	if bool(bytes.Equal(bytes.Slice(first), bytes.Slice(second))) {
		t.Fatalf("two V4 UUIDs collided: %v", first)
	}
	if uuid.UUID_Version(first) != 4 {
		t.Fatalf("V4 version = %v, want 4", uuid.UUID_Version(first))
	}
	var seven_storage [uuid.UUID_BYTE_COUNT]byte
	seven, failure := uuid.Generator_V7(&generator, uuid.UUID(seven_storage[:]))
	if failure != uuid.ERROR_NONE {
		t.Fatalf("V7 generation: %v", failure)
	}
	if uuid.UUID_Version(seven) != 7 {
		t.Fatalf("V7 version = %v, want 7", uuid.UUID_Version(seven))
	}
}

// Test_Operating_System_Generator_Zero_Allocation proves host wiring keeps caller state.
func Test_Operating_System_Generator_Zero_Allocation(t *testing.T) {
	var source prng.Chacha
	var seed [prng.KEY_BYTES]byte
	var seed_storage [prng.KEY_BYTES]byte
	seed[0] = 1
	entropy := func(destination []byte) (count int, err error) {
		return copy(destination, seed[:]), nil
	}
	testify.Zero_Allocation(t, func() {
		system_uuid.New_Operating_System_Generator(
			entropy, prng.Seed(seed_storage[:]), &source, uuid.Generator_State{},
		)
	})
}

// Test_Operating_System_Generator_Accepts_Replay_Boundaries proves composition
// preserves every explicit state boundary while replacing every source cursor.
func Test_Operating_System_Generator_Accepts_Replay_Boundaries(t *testing.T) {
	cursors := []prng.Cursor{
		prng.CURSOR_MIN, 1, 2, prng.CURSOR_MAX,
	}
	states := []uuid.Generator_State{
		{},
		{Clock_Sequence: 1, Last_Time: 1, Last_V7: 1},
		{Clock_Sequence: 2, Last_Time: 2, Last_V7: 2},
		{
			Clock_Sequence: uuid.CLOCK_SEQUENCE_VALUE_COUNT,
			Last_Time:      uuid.Timestamp_State(uuid.TIME_MAXIMUM),
			Last_V7:        uuid.V7_State(uuid.V7_VALUE_MAXIMUM),
		},
	}
	for index, cursor := range cursors {
		source := prng.Chacha{Position: cursor}
		seed := make([]byte, prng.KEY_BYTES)
		seed_storage := make([]byte, prng.KEY_BYTES)
		seed[0] = byte(index + 1)
		entropy := func(destination []byte) (count int, err error) {
			return copy(destination, seed), nil
		}
		constructed := system_uuid.New_Operating_System_Generator(
			entropy, prng.Seed(seed_storage), &source, states[index],
		)
		generator := uuid.Generator(constructed)
		if generator.Clock_Sequence != states[index].Clock_Sequence {
			t.Fatalf("state %d clock sequence changed", index)
		}
		if generator.Last_Time != states[index].Last_Time {
			t.Fatalf("state %d timestamp changed", index)
		}
		if generator.Last_V7 != states[index].Last_V7 {
			t.Fatalf("state %d V7 value changed", index)
		}
	}
	for _, value := range []uint64{0, 1, 2, bits.WORD_64_MAXIMUM} {
		source := domain_chacha(value)
		seed := make([]byte, prng.KEY_BYTES)
		seed_storage := make([]byte, prng.KEY_BYTES)
		entropy := func(destination []byte) (count int, err error) {
			return copy(destination, seed), nil
		}
		system_uuid.New_Operating_System_Generator(
			entropy, prng.Seed(seed_storage), &source, uuid.Generator_State{},
		)
	}
}

// Complete state witnesses keep source invariants tied to real constructor entry.
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
