package uuid_test

import (
	"testing"

	"local/james-orcales/shared/crypto/prng"
	"local/james-orcales/shared/sim/aver/default"
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
	entropy := seed_source{Seed: [prng.KEY_BYTES]byte{1}}
	generator := system_uuid.New_Operating_System_Generator(
		entropy.Read, &source, uuid.Generator_State{},
	)
	first := uuid.Must(uuid.Generator_V4(&generator))
	second := uuid.Must(uuid.Generator_V4(&generator))
	if first == second {
		t.Fatalf("two V4 UUIDs collided: %v", first)
	}
	if uuid.UUID_Version(first) != 4 {
		t.Fatalf("V4 version = %v, want 4", uuid.UUID_Version(first))
	}
	seven := uuid.Must(uuid.Generator_V7(&generator))
	if uuid.UUID_Version(seven) != 7 {
		t.Fatalf("V7 version = %v, want 7", uuid.UUID_Version(seven))
	}
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
		entropy := seed_source{Seed: [prng.KEY_BYTES]byte{byte(index + 1)}}
		generator := system_uuid.New_Operating_System_Generator(
			entropy.Read, &source, states[index],
		)
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
}

type seed_source struct {
	Seed [prng.KEY_BYTES]byte
}

func (source *seed_source) Read(destination []byte) (count int, err error) {
	return copy(destination, source.Seed[:]), nil
}
