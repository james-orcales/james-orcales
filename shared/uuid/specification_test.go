package uuid_test

import (
	"encoding/json"
	"strings"
	"testing"

	"local/james-orcales/shared/uuid"
)

// Test_Parse_Round_Trips_String checks Parse recovers a UUID from each string form.
func Test_Parse_Round_Trips_String(t *testing.T) {
	generator := fixed_generator(1)
	original := uuid.Must(uuid.Generator_V4(&generator))
	forms := []string{
		original.String(),
		uuid.UUID_URN(original),
		"{" + original.String() + "}",
		strings.ReplaceAll(original.String(), "-", ""),
	}
	for _, form := range forms {
		parsed, parse_err := uuid.Parse(form)
		if parse_err != nil {
			t.Fatalf("parse %q: %v", form, parse_err)
		}
		if parsed != original {
			t.Fatalf("parse %q got %v, want %v", form, parsed, original)
		}
	}
	if _, bad := uuid.Parse("not a uuid"); bad == nil {
		t.Fatalf("expected an error for a malformed string")
	}
	// A 38-character string is the braced form only when both braces are present. Any
	// other wrapper is a different encoding, not a UUID this package accepts.
	mismatched := []string{
		"(" + original.String() + ")",
		"{" + original.String() + "!",
		"!" + original.String() + "}",
		" " + original.String() + " ",
	}
	for _, form := range mismatched {
		if _, bad := uuid.Parse(form); bad == nil {
			t.Fatalf("parse %q returned no error, want a bracket error", form)
		}
	}
}

// Test_Version_And_Variant_Are_Stamped checks each generator sets its version and variant.
func Test_Version_And_Variant_Are_Stamped(t *testing.T) {
	generator := fixed_generator(2)
	check := func(name string, value uuid.UUID, want uuid.Version) {
		if uuid.UUID_Version(value) != want {
			t.Fatalf("%s version = %v, want %v", name, uuid.UUID_Version(value), want)
		}
		if uuid.UUID_Variant(value) != uuid.VARIANT_RFC_4122 {
			t.Fatalf("%s variant = %v, want RFC 4122", name, uuid.UUID_Variant(value))
		}
	}
	check("v1", uuid.Must(uuid.Generator_V1(&generator)), 1)
	check("v4", uuid.Must(uuid.Generator_V4(&generator)), 4)
	check("v6", uuid.Must(uuid.Generator_V6(&generator)), 6)
	check("v7", uuid.Must(uuid.Generator_V7(&generator)), 7)
	dce := uuid.Must(uuid.Generator_DCE_Security(&generator, uuid.DOMAIN_PERSON, 1000))
	check("v2", dce, 2)
	check("v3", uuid.V3(uuid.Name_Space_DNS(), []byte("example.com")), 3)
	check("v5", uuid.V5(uuid.Name_Space_DNS(), []byte("example.com")), 5)
}

// Test_Seed_Reproduces_Sequence checks one seed replays and distinct seeds diverge.
func Test_Seed_Reproduces_Sequence(t *testing.T) {
	first := fixed_generator(7)
	again := fixed_generator(7)
	for draw_index := 0; draw_index < 16; draw_index++ {
		left := uuid.Must(uuid.Generator_V4(&first))
		right := uuid.Must(uuid.Generator_V4(&again))
		if left != right {
			t.Fatalf("draw %d diverged: %v vs %v", draw_index, left, right)
		}
	}
	seven := fixed_generator(7)
	eight := fixed_generator(8)
	if uuid.Must(uuid.Generator_V4(&seven)) == uuid.Must(uuid.Generator_V4(&eight)) {
		t.Fatalf("distinct seeds produced the same UUID")
	}
}

// Test_V7_Is_Monotonic checks successive V7 draws strictly increase.
func Test_V7_Is_Monotonic(t *testing.T) {
	generator := fixed_generator(3)
	previous := uuid.Must(uuid.Generator_V7(&generator))
	for draw_index := 0; draw_index < 100; draw_index++ {
		next := uuid.Must(uuid.Generator_V7(&generator))
		if uuid.Compare(&uuid.Compare_Input{A: previous, B: next}) >= 0 {
			t.Fatalf("draw %d not increasing: %v then %v", draw_index, previous, next)
		}
		previous = next
	}
}

// Test_Text_Marshaling_Round_Trips checks MarshalText and UnmarshalText are inverses.
func Test_Text_Marshaling_Round_Trips(t *testing.T) {
	generator := fixed_generator(4)
	original := uuid.Must(uuid.Generator_V4(&generator))
	text, err := original.MarshalText()
	if err != nil {
		t.Fatalf("marshal text: %v", err)
	}
	if string(text) != original.String() {
		t.Fatalf("text = %q, want %q", text, original.String())
	}
	var restored uuid.UUID
	if unmarshal_err := restored.UnmarshalText(text); unmarshal_err != nil {
		t.Fatalf("unmarshal text: %v", unmarshal_err)
	}
	if restored != original {
		t.Fatalf("restored %v, want %v", restored, original)
	}
}

// Test_Binary_Marshaling_Round_Trips checks the 16-byte binary form round-trips.
func Test_Binary_Marshaling_Round_Trips(t *testing.T) {
	generator := fixed_generator(5)
	original := uuid.Must(uuid.Generator_V4(&generator))
	data, err := original.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal binary: %v", err)
	}
	if len(data) != 16 {
		t.Fatalf("binary length = %d, want 16", len(data))
	}
	var restored uuid.UUID
	if unmarshal_err := restored.UnmarshalBinary(data); unmarshal_err != nil {
		t.Fatalf("unmarshal binary: %v", unmarshal_err)
	}
	if restored != original {
		t.Fatalf("restored %v, want %v", restored, original)
	}
	if bad := restored.UnmarshalBinary([]byte{1, 2, 3}); bad == nil {
		t.Fatalf("expected an error for a short slice")
	}
}

// Test_JSON_Null_Round_Trips checks a valid Null_UUID and a null one both round-trip.
func Test_JSON_Null_Round_Trips(t *testing.T) {
	generator := fixed_generator(6)
	valid := uuid.Null_UUID{UUID: uuid.Must(uuid.Generator_V4(&generator)), Valid: true}
	blob, err := json.Marshal(valid)
	if err != nil {
		t.Fatalf("marshal valid: %v", err)
	}
	var restored uuid.Null_UUID
	if unmarshal_err := json.Unmarshal(blob, &restored); unmarshal_err != nil {
		t.Fatalf("unmarshal valid: %v", unmarshal_err)
	}
	if !restored.Valid {
		t.Fatalf("restored is invalid, want valid %+v", valid)
	}
	if restored.UUID != valid.UUID {
		t.Fatalf("restored %v, want %v", restored.UUID, valid.UUID)
	}
	empty, err := json.Marshal(uuid.Null_UUID{})
	if err != nil {
		t.Fatalf("marshal invalid: %v", err)
	}
	if string(empty) != "null" {
		t.Fatalf("invalid marshaled to %q, want null", empty)
	}
	var back uuid.Null_UUID
	if unmarshal_err := json.Unmarshal([]byte("null"), &back); unmarshal_err != nil {
		t.Fatalf("unmarshal null: %v", unmarshal_err)
	}
	if back.Valid {
		t.Fatalf("null unmarshaled to a valid value")
	}
}

// Test_SQL_Scan_Reads_Value checks Scan reads both column forms and UUID_Value emits one.
func Test_SQL_Scan_Reads_Value(t *testing.T) {
	generator := fixed_generator(9)
	original := uuid.Must(uuid.Generator_V4(&generator))
	var from_string uuid.UUID
	if err := from_string.Scan(original.String()); err != nil {
		t.Fatalf("scan string: %v", err)
	}
	if from_string != original {
		t.Fatalf("from string %v, want %v", from_string, original)
	}
	var from_bytes uuid.UUID
	if err := from_bytes.Scan(original[:]); err != nil {
		t.Fatalf("scan bytes: %v", err)
	}
	if from_bytes != original {
		t.Fatalf("from bytes %v, want %v", from_bytes, original)
	}
	value, err := uuid.UUID_Value(original)
	if err != nil {
		t.Fatalf("value: %v", err)
	}
	if value != original.String() {
		t.Fatalf("value = %v, want %v", value, original.String())
	}
}

// Test_Name_Based_Is_Deterministic checks V3 and V5 are pure functions of their inputs.
func Test_Name_Based_Is_Deterministic(t *testing.T) {
	first := uuid.V5(uuid.Name_Space_DNS(), []byte("example.com"))
	again := uuid.V5(uuid.Name_Space_DNS(), []byte("example.com"))
	if first != again {
		t.Fatalf("v5 was not deterministic: %v vs %v", first, again)
	}
	other := uuid.V5(uuid.Name_Space_DNS(), []byte("other.com"))
	if first == other {
		t.Fatalf("distinct data produced the same v5")
	}
	md5_based := uuid.V3(uuid.Name_Space_DNS(), []byte("example.com"))
	if first == md5_based {
		t.Fatalf("v3 and v5 of the same input collided")
	}
}

// Test_Time_Round_Trips_Through_UUID checks UUID_Time and Time_Unix recover the clock.
func Test_Time_Round_Trips_Through_UUID(t *testing.T) {
	generator := fixed_generator(10)
	value := uuid.Must(uuid.Generator_V7(&generator))
	timestamp := uuid.UUID_Time(value)
	seconds, _ := uuid.Time_Unix(timestamp)
	if seconds != FIXED_EPOCH_SECONDS {
		t.Fatalf("seconds = %d, want %d", seconds, FIXED_EPOCH_SECONDS)
	}
}

// Test_DCE_Embeds_Domain_And_Identifier checks a DCE UUID carries its domain and id.
func Test_DCE_Embeds_Domain_And_Identifier(t *testing.T) {
	generator := fixed_generator(11)
	value := uuid.Must(uuid.Generator_DCE_Security(&generator, uuid.DOMAIN_GROUP, 4242))
	if uuid.UUID_Version(value) != 2 {
		t.Fatalf("version = %v, want 2", uuid.UUID_Version(value))
	}
	if uuid.UUID_Domain(value) != uuid.DOMAIN_GROUP {
		t.Fatalf("domain = %v, want Group", uuid.UUID_Domain(value))
	}
	if uuid.UUID_Identifier(value) != 4242 {
		t.Fatalf("identifier = %d, want 4242", uuid.UUID_Identifier(value))
	}
}
