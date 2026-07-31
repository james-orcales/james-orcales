package uuid_test

import (
	"testing"

	"local/james-orcales/shared/uuid"
	system_uuid "local/james-orcales/shared/uuid/default"
)

// Test_Operating_System_Generator_Smoke checks the host-wired generator mints valid,
// distinct V4 and V7 UUIDs. Real entropy is non-deterministic, so this is a smoke test.
func Test_Operating_System_Generator_Smoke(t *testing.T) {
	generator := system_uuid.New_Operating_System_Generator()
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
