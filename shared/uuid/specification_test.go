package uuid_test

import (
	"testing"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/database/driver"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/time"
	"local/james-orcales/shared/strings"
	"local/james-orcales/shared/uuid"
)

// Test_Parse_Round_Trips_String checks Parse recovers a UUID from each string form.
func Test_Parse_Round_Trips_String(t *testing.T) {
	generator := fixed_generator(1)
	original := uuid.Must(uuid.Generator_V4(&generator))
	forms := []string{
		original.String(),
		string(uuid.UUID_URN(original)),
		"{" + original.String() + "}",
		original.String()[:8] + original.String()[9:13] + original.String()[14:18] +
			original.String()[19:23] + original.String()[24:],
	}
	for _, form := range forms {
		parsed, parse_err := uuid.Parse(uuid.Text_Unvalidated(form))
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
		if _, bad := uuid.Parse(uuid.Text_Unvalidated(form)); bad == nil {
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
	dce := uuid.Must(uuid.Generator_DCE_Security(
		&generator, uuid.DOMAIN_PERSON, uuid.Identifier(1000),
	))
	check("v2", dce, 2)
	check("v3", uuid.V3(uuid.Name_Space_DNS(), uuid.Name("example.com")), 3)
	check("v5", uuid.V5(uuid.Name_Space_DNS(), uuid.Name("example.com")), 5)
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
	blob, err := valid.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal valid: %v", err)
	}
	var restored uuid.Null_UUID
	if unmarshal_err := restored.UnmarshalJSON(blob); unmarshal_err != nil {
		t.Fatalf("unmarshal valid: %v", unmarshal_err)
	}
	if !bool(restored.Valid) {
		t.Fatalf("restored is invalid, want valid %+v", valid)
	}
	if restored.UUID != valid.UUID {
		t.Fatalf("restored %v, want %v", restored.UUID, valid.UUID)
	}
	empty, err := (uuid.Null_UUID{}).MarshalJSON()
	if err != nil {
		t.Fatalf("marshal invalid: %v", err)
	}
	if string(empty) != "null" {
		t.Fatalf("invalid marshaled to %q, want null", empty)
	}
	var back uuid.Null_UUID
	if unmarshal_err := back.UnmarshalJSON([]byte("null")); unmarshal_err != nil {
		t.Fatalf("unmarshal null: %v", unmarshal_err)
	}
	if bool(back.Valid) {
		t.Fatalf("null unmarshaled to a valid value")
	}
}

// Test_Database_Scan_Reads_Value checks shared driver forms and UUID_Value.
func Test_Database_Scan_Reads_Value(t *testing.T) {
	generator := fixed_generator(9)
	original := uuid.Must(uuid.Generator_V4(&generator))
	status_ok := driver.Validation_Status(driver.STATUS_OK)
	text_value, text_status := driver.Value_Of_Text(
		driver.Text_Unvalidated(original.String()),
	)
	if text_status != status_ok {
		t.Fatalf("build text value: %v", text_status)
	}
	var from_string uuid.UUID
	if status := uuid.UUID_Scan(&from_string, text_value); status != status_ok {
		t.Fatalf("scan string: %v", status)
	}
	if from_string != original {
		t.Fatalf("from string %v, want %v", from_string, original)
	}
	bytes_value, bytes_status := driver.Value_Of_Bytes(
		driver.Bytes_Unvalidated(original[:]),
	)
	if bytes_status != status_ok {
		t.Fatalf("build bytes value: %v", bytes_status)
	}
	var from_bytes uuid.UUID
	if status := uuid.UUID_Scan(&from_bytes, bytes_value); status != status_ok {
		t.Fatalf("scan bytes: %v", status)
	}
	if from_bytes != original {
		t.Fatalf("from bytes %v, want %v", from_bytes, original)
	}
	value := uuid.UUID_Value(original)
	text, text_status := driver.Value_As_Text(value)
	if text_status != status_ok {
		t.Fatalf("value text: %v", text_status)
	}
	if string(text) != original.String() {
		t.Fatalf("value = %v, want %v", text, original.String())
	}
	null_value := driver.Value_Null()
	var nullable uuid.Null_UUID
	if status := uuid.Null_UUID_Scan(&nullable, null_value); status != status_ok {
		t.Fatalf("scan null: %v", status)
	}
	if bool(nullable.Valid) {
		t.Fatalf("database null became valid UUID")
	}
	nullable.Valid = true
	if status := uuid.Null_UUID_Scan(&nullable, text_value); status != status_ok {
		t.Fatalf("scan nullable text: %v", status)
	}
	if !bool(nullable.Valid) {
		t.Fatalf("nullable text stayed invalid")
	}
	if nullable.UUID != original {
		t.Fatalf("nullable text did not survive")
	}
	invalid_value, invalid_value_status := driver.Value_Of_Text("x")
	if invalid_value_status != status_ok {
		t.Fatalf("build invalid UUID text: %v", invalid_value_status)
	}
	if status := uuid.Null_UUID_Scan(&nullable, invalid_value); status == status_ok {
		t.Fatalf("nullable scan accepted invalid UUID text")
	}
	if bool(nullable.Valid) {
		t.Fatalf("rejected nullable text stayed valid")
	}
}

// Test_Name_Based_Is_Deterministic checks V3 and V5 are pure functions of their inputs.
func Test_Name_Based_Is_Deterministic(t *testing.T) {
	first := uuid.V5(uuid.Name_Space_DNS(), uuid.Name("example.com"))
	again := uuid.V5(uuid.Name_Space_DNS(), uuid.Name("example.com"))
	if first != again {
		t.Fatalf("v5 was not deterministic: %v vs %v", first, again)
	}
	other := uuid.V5(uuid.Name_Space_DNS(), uuid.Name("other.com"))
	if first == other {
		t.Fatalf("distinct data produced the same v5")
	}
	md5_based := uuid.V3(uuid.Name_Space_DNS(), uuid.Name("example.com"))
	if first == md5_based {
		t.Fatalf("v3 and v5 of the same input collided")
	}
	v3_reference := uuid.Must_Parse("3d813cbb-47fb-32ba-91df-831e1593ac29")
	if got := uuid.V3(
		uuid.Name_Space_DNS(), uuid.Name("www.widgets.com"),
	); got != v3_reference {
		t.Fatalf("v3 reference got %v, want %v", got, v3_reference)
	}
	v5_reference := uuid.Must_Parse("21f7f8de-8051-5b89-8680-0195ef798b6a")
	if got := uuid.V5(
		uuid.Name_Space_DNS(), uuid.Name("www.widgets.com"),
	); got != v5_reference {
		t.Fatalf("v5 reference got %v, want %v", got, v5_reference)
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
	value := uuid.Must(uuid.Generator_DCE_Security(
		&generator, uuid.DOMAIN_GROUP, uuid.Identifier(4242),
	))
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

// Test_Errors_Have_Static_Text reaches every bounded failure kind through error interface.
func Test_Errors_Have_Static_Text(t *testing.T) {
	errors := []uuid.Error{
		uuid.ERROR_INVALID_FORMAT,
		uuid.ERROR_INVALID_BRACKETED_FORMAT,
		uuid.ERROR_INVALID_SIZE,
		uuid.ERROR_INVALID_URN_PREFIX,
		uuid.ERROR_V7_STATE_EXHAUSTED,
		uuid.ERROR_V7_TIME_OUTPUT_OF_RANGE,
	}
	for _, failure := range errors {
		if failure.Error() == "" {
			t.Fatalf("error %d has no text", failure)
		}
	}
}

// Test_Replay_State_Boundaries_Reach_Generators proves every accepted replay
// boundary survives each generator entry point, including exhausted V7 state.
func Test_Replay_State_Boundaries_Reach_Generators(t *testing.T) {
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
	for index, state := range states {
		generator := generator_at(uint64(index), 0, uuid.Node{}, state)
		if _, err := uuid.Generator_V4(&generator); err != nil {
			t.Fatalf("state %d V4: %v", index, err)
		}
		generator = generator_at(uint64(index), 0, uuid.Node{}, state)
		if _, err := uuid.Generator_V1(&generator); err != nil {
			t.Fatalf("state %d V1: %v", index, err)
		}
		generator = generator_at(uint64(index), 0, uuid.Node{}, state)
		if _, err := uuid.Generator_V6(&generator); err != nil {
			t.Fatalf("state %d V6: %v", index, err)
		}
		generator = generator_at(uint64(index), 0, uuid.Node{}, state)
		if _, err := uuid.Generator_DCE_Security(
			&generator, uuid.DOMAIN_PERSON, 0,
		); err != nil {
			t.Fatalf("state %d DCE: %v", index, err)
		}
		generator = generator_at(uint64(index), 0, uuid.Node{}, state)
		_, err := uuid.Generator_V7(&generator)
		if state.Last_V7 == uuid.V7_State(uuid.V7_VALUE_MAXIMUM) {
			if err == nil {
				t.Fatalf("state %d V7 accepted exhausted state", index)
			}
			continue
		}
		if err != nil {
			t.Fatalf("state %d V7: %v", index, err)
		}
	}
}

// Test_Generator_Time_Boundaries_Reach_Fields proves signed Clock endpoints and
// every clock-sequence boundary survive actual V1 field encoding and decoding.
func Test_Generator_Time_Boundaries_Reach_Fields(t *testing.T) {
	for _, moment := range []time.Moment{
		time.Moment(bits.INTEGER_64_MINIMUM),
		time.Moment(bits.INTEGER_64_MAXIMUM),
	} {
		generator := generator_at(20, moment, uuid.Node{}, uuid.Generator_State{})
		if _, err := uuid.Generator_V1(&generator); err != nil {
			t.Fatalf("moment %d V1: %v", moment, err)
		}
	}
	states := []uuid.Clock_Sequence_State{
		1, 2, 3, uuid.CLOCK_SEQUENCE_VALUE_COUNT,
	}
	wants := []uuid.Clock_Sequence{
		0, 1, 2, uuid.CLOCK_SEQUENCE_MAXIMUM,
	}
	for index, state := range states {
		generator := generator_at(21, 0, uuid.Node{}, uuid.Generator_State{
			Clock_Sequence: state,
		})
		value := uuid.Must(uuid.Generator_V1(&generator))
		if got := uuid.UUID_Clock_Sequence(value); got != wants[index] {
			t.Fatalf("sequence state %d got %d, want %d", state, got, wants[index])
		}
	}
}

// Test_Generator_V7_Boundaries_Reach_Fields proves the accepted Clock image and
// full sub-millisecond sequence domain survive actual V7 generation.
func Test_Generator_V7_Boundaries_Reach_Fields(t *testing.T) {
	cases := []struct {
		Moment time.Moment
		State  uuid.V7_State
	}{
		{Moment: 0},
		{Moment: time.Moment(time.MILLISECOND)},
		{Moment: time.Moment(2 * time.MILLISECOND)},
		{Moment: time.Moment(bits.INTEGER_64_MAXIMUM)},
		{Moment: 0, State: 1},
		{Moment: 0, State: uuid.V7_SEQUENCE_MAXIMUM - 1},
	}
	for index, test := range cases {
		generator := generator_at(30, test.Moment, uuid.Node{}, uuid.Generator_State{
			Last_V7: test.State,
		})
		if _, err := uuid.Generator_V7(&generator); err != nil {
			t.Fatalf("V7 boundary %d: %v", index, err)
		}
	}
	generator := generator_at(31, -1, uuid.Node{}, uuid.Generator_State{})
	if _, err := uuid.Generator_V7(&generator); err == nil {
		t.Fatalf("V7 accepted pre-Unix clock")
	}
}

// Test_Unvalidated_Input_Boundaries_Reach_Parsers proves hostile sizes stop at
// bounded public entries, including Must_Parse's deliberate panic contract.
func Test_Unvalidated_Input_Boundaries_Reach_Parsers(t *testing.T) {
	if bytes.SLICE_SIZE_MAXIMUM != strings.TEXT_SIZE_MAXIMUM {
		t.Fatalf("byte and text input bounds diverged")
	}
	for _, size := range []int{0, 1, 2, bytes.SLICE_SIZE_MAXIMUM} {
		raw := make([]byte, size)
		text := uuid.Text_Unvalidated(string(raw))
		uuid.Parse(text)
		uuid.Parse_Bytes(bytes.Slice(raw))
		uuid.From_Bytes(bytes.Slice(raw))
		uuid.Validate(text)
		value, value_status := driver.Value_Of_Text(driver.Text_Unvalidated(text))
		if value_status != driver.Validation_Status(driver.STATUS_OK) {
			t.Fatalf("build hostile text size %d: %v", size, value_status)
		}
		var scanned uuid.UUID
		uuid.UUID_Scan(&scanned, value)
		panicked := false
		func() {
			defer func() { panicked = recover() != nil }()
			uuid.Must_Parse(text)
		}()
		if !panicked {
			t.Fatalf("Must_Parse accepted hostile size %d", size)
		}
	}
	raw := make([]byte, uuid.UUID_BYTE_COUNT)
	if _, err := uuid.From_Bytes(bytes.Slice(raw)); err != nil {
		t.Fatalf("From_Bytes rejected exact storage: %v", err)
	}
}

// Test_Name_Boundaries_Reach_Hashes proves both hash versions accept their full
// bounded name domain without moving work or allocation into a hidden helper.
func Test_Name_Boundaries_Reach_Hashes(t *testing.T) {
	namespace := uuid.Name_Space_DNS()
	for _, size := range []int{0, 1, 2, bytes.SLICE_SIZE_MAXIMUM} {
		name := make(uuid.Name, size)
		uuid.V3(namespace, name)
		uuid.V5(namespace, name)
	}
}

// Test_Decoded_Field_Boundaries proves accessors expose their complete declared
// domains from UUID bytes, not by calling invariant functions from the test.
func Test_Decoded_Field_Boundaries(t *testing.T) {
	for _, version := range []uuid.Version{0, 1, 2, uuid.VERSION_MAXIMUM} {
		value := uuid.Nil()
		value[6] = byte(version) << uuid.VERSION_BIT_COUNT
		if got := uuid.UUID_Version(value); got != version {
			t.Fatalf("version got %d, want %d", got, version)
		}
	}
	variant_index := uuid.UUID_BYTE_COUNT / 2
	variant_cases := []struct {
		Control byte
		Want    uuid.Variant
	}{
		{Control: 0, Want: uuid.VARIANT_RESERVED},
		{
			Control: 1 << (bits.BIT_COUNT_8_MAXIMUM - 1),
			Want:    uuid.VARIANT_RFC_4122,
		},
		{
			Control: 3 << (bits.BIT_COUNT_8_MAXIMUM - 2),
			Want:    uuid.VARIANT_MICROSOFT,
		},
		{
			Control: 7 << (bits.BIT_COUNT_8_MAXIMUM - 3),
			Want:    uuid.VARIANT_FUTURE,
		},
	}
	for _, test := range variant_cases {
		value := uuid.Nil()
		value[variant_index] = test.Control
		if got := uuid.UUID_Variant(value); got != test.Want {
			t.Fatalf("variant control %d got %d, want %d", test.Control, got, test.Want)
		}
	}
	for _, timestamp := range []uuid.Time{0, 1, 2} {
		value := uuid.Nil()
		binary.Put_Uint_32(
			binary.Bytes(value[:]), binary.Word_32(timestamp), binary.BIG_ENDIAN,
		)
		if got := uuid.UUID_Time(value); got != timestamp {
			t.Fatalf("time got %d, want %d", got, timestamp)
		}
	}
	if got := uuid.UUID_Time(uuid.Max()); got != uuid.Time(uuid.TIME_MAXIMUM) {
		t.Fatalf("maximum time got %d, want %d", got, uuid.TIME_MAXIMUM)
	}
	for _, identifier := range []uuid.Identifier{
		0, 1, 2, uuid.Identifier(bits.WORD_32_MAXIMUM),
	} {
		value := uuid.Nil()
		binary.Put_Uint_32(
			binary.Bytes(value[:]), binary.Word_32(identifier), binary.BIG_ENDIAN,
		)
		if got := uuid.UUID_Identifier(value); got != identifier {
			t.Fatalf("identifier got %d, want %d", got, identifier)
		}
	}
}

// Test_DCE_Boundaries_Reach_Accessors proves every domain and identifier bound
// survives Version 2 generation before field access.
func Test_DCE_Boundaries_Reach_Accessors(t *testing.T) {
	cases := []struct {
		Domain     uuid.Domain
		Identifier uuid.Identifier
	}{
		{Domain: uuid.DOMAIN_PERSON, Identifier: 0},
		{Domain: uuid.DOMAIN_GROUP, Identifier: 1},
		{Domain: uuid.DOMAIN_ORGANIZATION, Identifier: 2},
		{
			Domain:     uuid.DOMAIN_ORGANIZATION,
			Identifier: uuid.Identifier(bits.WORD_32_MAXIMUM),
		},
	}
	for index, test := range cases {
		generator := fixed_generator(uint64(index + 40))
		value := uuid.Must(uuid.Generator_DCE_Security(
			&generator, test.Domain, test.Identifier,
		))
		if uuid.UUID_Domain(value) != test.Domain {
			t.Fatalf("domain %d did not survive", test.Domain)
		}
		if uuid.UUID_Identifier(value) != test.Identifier {
			t.Fatalf("identifier %d did not survive", test.Identifier)
		}
	}
}

// Test_Collection_And_Value_Boundaries_Reach_Production covers collection sizes,
// every comparison result, and both nullable database values.
func Test_Collection_And_Value_Boundaries_Reach_Production(t *testing.T) {
	for _, size := range []int{0, 1, 2, uuid.UUIDS_COUNT_MAXIMUM} {
		values := make(uuid.UUIDs, size)
		if rendered := uuid.UUIDs_Strings(values); len(rendered) != size {
			t.Fatalf("rendered %d UUIDs, want %d", len(rendered), size)
		}
	}
	nil_uuid := uuid.Nil()
	uuid_max := uuid.Max()
	if uuid.Compare(&uuid.Compare_Input{A: nil_uuid, B: nil_uuid}) != 0 {
		t.Fatalf("equal UUIDs did not compare equal")
	}
	if uuid.Compare(&uuid.Compare_Input{A: uuid_max, B: nil_uuid}) != 1 {
		t.Fatalf("maximum UUID did not compare after nil")
	}
	null_value := uuid.Null_UUID_Value(uuid.Null_UUID{})
	if driver.Value_Kind_Of(null_value) != driver.VALUE_NULL {
		t.Fatalf("invalid nullable value is not database null")
	}
	valid_value := uuid.Null_UUID_Value(uuid.Null_UUID{
		UUID: uuid_max, Valid: true,
	})
	if driver.Value_Kind_Of(valid_value) != driver.VALUE_TEXT {
		t.Fatalf("valid nullable value is not database text")
	}
}

// Test_Time_Unix_Boundaries_Reach_Production proves timestamp and Unix-second
// endpoints plus exact signed 100-nanosecond remainders.
func Test_Time_Unix_Boundaries_Reach_Production(t *testing.T) {
	generator := generator_at(50, 0, uuid.Node{}, uuid.Generator_State{})
	epoch := uuid.UUID_Time(uuid.Must(uuid.Generator_V7(&generator)))
	timestamps := []uuid.Time{
		0,
		1,
		2,
		uuid.Time(uuid.TIME_MAXIMUM),
		epoch - uuid.Time(uuid.UUID_TICK_COUNT_PER_SECOND),
		epoch,
		epoch + uuid.Time(uuid.UUID_TICK_COUNT_PER_SECOND),
		epoch + uuid.Time(2*uuid.UUID_TICK_COUNT_PER_SECOND),
	}
	for _, timestamp := range timestamps {
		uuid.Time_Unix(timestamp)
	}
	_, minimum := uuid.Time_Unix(
		epoch - uuid.Time(uuid.UUID_TICK_COUNT_PER_SECOND-1),
	)
	if minimum[0] != uuid.NANOSECOND_MINIMUM {
		t.Fatalf("nanosecond minimum got %d, want %d", minimum[0], uuid.NANOSECOND_MINIMUM)
	}
	_, maximum := uuid.Time_Unix(
		epoch + uuid.Time(uuid.UUID_TICK_COUNT_PER_SECOND-1),
	)
	if maximum[0] != uuid.NANOSECOND_MAXIMUM {
		t.Fatalf("nanosecond maximum got %d, want %d", maximum[0], uuid.NANOSECOND_MAXIMUM)
	}
}
