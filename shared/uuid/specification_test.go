package uuid_test

import (
	"testing"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/database/driver"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/time"
	"local/james-orcales/shared/strings"
	"local/james-orcales/shared/testify"
	"local/james-orcales/shared/uuid"
)

// Test_Parse_Round_Trips_String checks Parse recovers a UUID from each string form.
func Test_Parse_Round_Trips_String(t *testing.T) {
	generator := fixed_generator(1)
	original := generated_v4(&generator)
	canonical := text_uuid(original)
	forms := []string{
		canonical,
		urn_uuid(original),
		"{" + canonical + "}",
		canonical[:8] + canonical[9:13] + canonical[14:18] + canonical[19:23] +
			canonical[24:],
	}
	for _, form := range forms {
		parsed, parse_err := parsed_uuid(uuid.Text_Unvalidated(form))
		if parse_err != uuid.ERROR_NONE {
			t.Fatalf("parse %q: %v", form, parse_err)
		}
		if !bool(bytes.Equal(bytes.Slice(parsed), bytes.Slice(original))) {
			t.Fatalf("parse %q got %v, want %v", form, parsed, original)
		}
	}
	if bad := uuid.Parse(uuid_storage(), "not a uuid"); bad == uuid.ERROR_NONE {
		t.Fatalf("expected an error for a malformed string")
	}
	// A 38-character string is the braced form only when both braces are present. Any
	// other wrapper is a different encoding, not a UUID this package accepts.
	mismatched := []string{
		"(" + canonical + ")",
		"{" + canonical + "!",
		"!" + canonical + "}",
		" " + canonical + " ",
	}
	for _, form := range mismatched {
		if bad := uuid.Parse(
			uuid_storage(), uuid.Text_Unvalidated(form),
		); bad == uuid.ERROR_NONE {
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
	check("v1", generated_v1(&generator), 1)
	check("v4", generated_v4(&generator), 4)
	check("v6", generated_v6(&generator), 6)
	check("v7", generated_v7(&generator), 7)
	dce := generated_dce(&generator, uuid.DOMAIN_PERSON, uuid.Identifier(1000))
	check("v2", dce, 2)
	check("v3", v3_uuid(dns_namespace(), uuid.Name("example.com")), 3)
	check("v5", v5_uuid(dns_namespace(), uuid.Name("example.com")), 5)
}

// Test_Seed_Reproduces_Sequence checks one seed replays and distinct seeds diverge.
func Test_Seed_Reproduces_Sequence(t *testing.T) {
	first := fixed_generator(7)
	again := fixed_generator(7)
	for draw_index := 0; draw_index < 16; draw_index++ {
		left := generated_v4(&first)
		right := generated_v4(&again)
		if !bool(bytes.Equal(bytes.Slice(left), bytes.Slice(right))) {
			t.Fatalf("draw %d diverged: %v vs %v", draw_index, left, right)
		}
	}
	seven := fixed_generator(7)
	eight := fixed_generator(8)
	if bool(bytes.Equal(
		bytes.Slice(generated_v4(&seven)),
		bytes.Slice(generated_v4(&eight)),
	)) {
		t.Fatalf("distinct seeds produced the same UUID")
	}
}

// Test_V7_Is_Monotonic checks successive V7 draws strictly increase.
func Test_V7_Is_Monotonic(t *testing.T) {
	generator := fixed_generator(3)
	previous := generated_v7(&generator)
	for draw_index := 0; draw_index < 100; draw_index++ {
		next := generated_v7(&generator)
		if bytes.Compare(bytes.Slice(previous), bytes.Slice(next)) >= 0 {
			t.Fatalf("draw %d not increasing: %v then %v", draw_index, previous, next)
		}
		previous = next
	}
}

// Test_Text_Marshaling_Round_Trips checks free text operations are inverses.
func Test_Text_Marshaling_Round_Trips(t *testing.T) {
	generator := fixed_generator(4)
	original := generated_v4(&generator)
	text_storage := make(uuid.Text_Bytes, uuid.UUID_TEXT_BYTE_COUNT)
	text := uuid.UUID_Marshal_Text(original, text_storage)
	if string(text) != text_uuid(original) {
		t.Fatalf("text = %q, want %q", text, text_uuid(original))
	}
	restored := nil_uuid()
	if unmarshal_err := uuid.UUID_Unmarshal_Text(
		&restored, bytes.Slice(text),
	); unmarshal_err != uuid.ERROR_NONE {
		t.Fatalf("unmarshal text: %v", unmarshal_err)
	}
	if !bool(bytes.Equal(bytes.Slice(restored), bytes.Slice(original))) {
		t.Fatalf("restored %v, want %v", restored, original)
	}
}

// Test_Binary_Marshaling_Round_Trips checks 16-byte binary form round-trips.
func Test_Binary_Marshaling_Round_Trips(t *testing.T) {
	generator := fixed_generator(5)
	original := generated_v4(&generator)
	data := uuid.UUID_Marshal_Binary(original)
	if len(data) != 16 {
		t.Fatalf("binary length = %d, want 16", len(data))
	}
	restored := nil_uuid()
	if unmarshal_err := uuid.UUID_Unmarshal_Binary(
		&restored, bytes.Slice(data),
	); unmarshal_err != uuid.ERROR_NONE {
		t.Fatalf("unmarshal binary: %v", unmarshal_err)
	}
	if !bool(bytes.Equal(bytes.Slice(restored), bytes.Slice(original))) {
		t.Fatalf("restored %v, want %v", restored, original)
	}
	if bad := uuid.UUID_Unmarshal_Binary(
		&restored, bytes.Slice{1, 2, 3},
	); bad == uuid.ERROR_NONE {
		t.Fatalf("expected an error for a short slice")
	}
}

// Test_JSON_Null_Round_Trips checks a valid Null_UUID and a null one both round-trip.
func Test_JSON_Null_Round_Trips(t *testing.T) {
	generator := fixed_generator(6)
	valid := uuid.Null_UUID{UUID: generated_v4(&generator), Valid: true}
	json_storage := make(uuid.JSON_Storage, uuid.UUID_BRACED_BYTE_COUNT)
	blob := uuid.Null_UUID_Marshal_JSON(valid, json_storage)
	restored := uuid.Null_UUID{UUID: nil_uuid(), Valid: true}
	if unmarshal_err := uuid.Null_UUID_Unmarshal_JSON(
		&restored, bytes.Slice(blob),
	); unmarshal_err != uuid.ERROR_NONE {
		t.Fatalf("unmarshal valid: %v", unmarshal_err)
	}
	if !bool(restored.Valid) {
		t.Fatalf("restored is invalid, want valid %+v", valid)
	}
	if !bool(bytes.Equal(bytes.Slice(restored.UUID), bytes.Slice(valid.UUID))) {
		t.Fatalf("restored %v, want %v", restored.UUID, valid.UUID)
	}
	empty := uuid.Null_UUID_Marshal_JSON(
		uuid.Null_UUID{UUID: nil_uuid()}, json_storage,
	)
	if string(empty) != "null" {
		t.Fatalf("invalid marshaled to %q, want null", empty)
	}
	back := uuid.Null_UUID{UUID: nil_uuid()}
	if unmarshal_err := uuid.Null_UUID_Unmarshal_JSON(
		&back, bytes.Slice("null"),
	); unmarshal_err != uuid.ERROR_NONE {
		t.Fatalf("unmarshal null: %v", unmarshal_err)
	}
	if bool(back.Valid) {
		t.Fatalf("null unmarshaled to a valid value")
	}
}

// Test_Database_Scan_Reads_Value checks shared driver forms and UUID_Value.
func Test_Database_Scan_Reads_Value(t *testing.T) {
	generator := fixed_generator(9)
	original := generated_v4(&generator)
	status_ok := driver.Validation_Status(driver.STATUS_OK)
	text_storage := make(uuid.Text_Bytes, uuid.UUID_TEXT_BYTE_COUNT)
	var text_value driver.Value
	text_status := driver.Value_Of_Text(
		&text_value, driver.Text_Unvalidated(uuid.UUID_String(original, text_storage)))
	if text_status != status_ok {
		t.Fatalf("build text value: %v", text_status)
	}
	from_string := nil_uuid()
	if status := uuid.UUID_Scan(&from_string, text_value); status != status_ok {
		t.Fatalf("scan string: %v", status)
	}
	if !bool(bytes.Equal(bytes.Slice(from_string), bytes.Slice(original))) {
		t.Fatalf("from string %v, want %v", from_string, original)
	}
	var bytes_value driver.Value
	bytes_status := driver.Value_Of_Bytes(
		&bytes_value, driver.Bytes_Unvalidated(original),
	)
	if bytes_status != status_ok {
		t.Fatalf("build bytes value: %v", bytes_status)
	}
	from_bytes := nil_uuid()
	if status := uuid.UUID_Scan(&from_bytes, bytes_value); status != status_ok {
		t.Fatalf("scan bytes: %v", status)
	}
	if !bool(bytes.Equal(bytes.Slice(from_bytes), bytes.Slice(original))) {
		t.Fatalf("from bytes %v, want %v", from_bytes, original)
	}
	value := uuid.UUID_Value(original, text_storage)
	text, text_status := driver.Value_As_Text(driver.Value(value))
	if text_status != status_ok {
		t.Fatalf("value text: %v", text_status)
	}
	if string(text) != text_uuid(original) {
		t.Fatalf("value = %v, want %v", text, text_uuid(original))
	}
	assert_nullable_scan(t, original, text_value, status_ok)
}

// Test_Name_Based_Is_Deterministic checks V3 and V5 are pure functions of their inputs.
func Test_Name_Based_Is_Deterministic(t *testing.T) {
	first := v5_uuid(dns_namespace(), uuid.Name("example.com"))
	again := v5_uuid(dns_namespace(), uuid.Name("example.com"))
	if !bool(bytes.Equal(bytes.Slice(first), bytes.Slice(again))) {
		t.Fatalf("v5 was not deterministic: %v vs %v", first, again)
	}
	other := v5_uuid(dns_namespace(), uuid.Name("other.com"))
	if bool(bytes.Equal(bytes.Slice(first), bytes.Slice(other))) {
		t.Fatalf("distinct data produced the same v5")
	}
	md5_based := v3_uuid(dns_namespace(), uuid.Name("example.com"))
	if bool(bytes.Equal(bytes.Slice(first), bytes.Slice(md5_based))) {
		t.Fatalf("v3 and v5 of the same input collided")
	}
	v3_reference := must_parsed_uuid("3d813cbb-47fb-32ba-91df-831e1593ac29")
	if got := uuid.V3(
		uuid_storage(), dns_namespace(), uuid.Name("www.widgets.com"),
	); !bool(bytes.Equal(bytes.Slice(got), bytes.Slice(v3_reference))) {
		t.Fatalf("v3 reference got %v, want %v", got, v3_reference)
	}
	v5_reference := must_parsed_uuid("21f7f8de-8051-5b89-8680-0195ef798b6a")
	if got := uuid.V5(
		uuid_storage(), dns_namespace(), uuid.Name("www.widgets.com"),
	); !bool(bytes.Equal(bytes.Slice(got), bytes.Slice(v5_reference))) {
		t.Fatalf("v5 reference got %v, want %v", got, v5_reference)
	}
}

// Test_Time_Round_Trips_Through_UUID checks UUID_Time and Time_Unix recover the clock.
func Test_Time_Round_Trips_Through_UUID(t *testing.T) {
	generator := fixed_generator(10)
	value := generated_v7(&generator)
	timestamp := uuid.UUID_Time(value)
	seconds, _ := uuid.Time_Unix(timestamp)
	if seconds != FIXED_EPOCH_SECONDS {
		t.Fatalf("seconds = %d, want %d", seconds, FIXED_EPOCH_SECONDS)
	}
}

// Test_DCE_Embeds_Domain_And_Identifier checks a DCE UUID carries its domain and id.
func Test_DCE_Embeds_Domain_And_Identifier(t *testing.T) {
	generator := fixed_generator(11)
	value := generated_dce(&generator, uuid.DOMAIN_GROUP, uuid.Identifier(4242))
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

// Test_Entire_Package_Is_Zero_Allocation covers every public operation and rejection.
func Test_Entire_Package_Is_Zero_Allocation(t *testing.T) {
	zero_allocation_core(t)
	zero_allocation_syntax(t)
	zero_allocation_encoding(t)
	zero_allocation_nullable(t)
}

// Caller arrays keep retained UUIDs outside measured package work.
func zero_allocation_core(t *testing.T) {
	generator := fixed_generator(12)
	negative := generator_at(13, -1, uuid.Node{}, uuid.Generator_State{})
	exhausted := generator_at(14, 0, uuid.Node{}, uuid.Generator_State{
		Last_V7: uuid.V7_State(uuid.V7_VALUE_MAXIMUM),
	})
	var value_storage [uuid.UUID_BYTE_COUNT]byte
	var namespace_storage [uuid.UUID_BYTE_COUNT]byte
	value := uuid.UUID(value_storage[:])
	namespace := uuid.Must_Parse(
		uuid.UUID(namespace_storage[:]), "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	)
	errors := []uuid.Error{
		uuid.ERROR_NONE, uuid.ERROR_INVALID_FORMAT, uuid.ERROR_INVALID_BRACKETED_FORMAT,
		uuid.ERROR_INVALID_SIZE, uuid.ERROR_INVALID_URN_PREFIX,
		uuid.ERROR_V7_STATE_EXHAUSTED, uuid.ERROR_V7_TIME_OUTPUT_OF_RANGE,
	}
	testify.Zero_Allocation(t, func() {
		uuid.New(generator.Source, generator.Clock, uuid.Node{}, uuid.Generator_State{})
		uuid.Nil(value)
		uuid.Max(value)
		uuid.Name_Space_DNS(value)
		uuid.Name_Space_URL(value)
		uuid.Name_Space_OID(value)
		uuid.Name_Space_X500(value)
		uuid.Generator_V1(&generator, value)
		uuid.Generator_V4(&generator, value)
		uuid.Generator_V6(&generator, value)
		uuid.Generator_DCE_Security(&generator, value, uuid.DOMAIN_PERSON, 1)
		uuid.Generator_V7(&generator, value)
		uuid.Generator_V7(&negative, value)
		uuid.Generator_V7(&exhausted, value)
		uuid.V3(value, namespace, uuid.Name("name"))
		uuid.V5(value, namespace, uuid.Name("name"))
		for _, failure := range errors {
			uuid.Error_Text(failure)
		}
	})
}

// Shared destination proves successful and hostile syntax avoid temporary objects.
func zero_allocation_syntax(t *testing.T) {
	var value_storage [uuid.UUID_BYTE_COUNT]byte
	value := uuid.UUID(value_storage[:])
	canonical := uuid.Text_Unvalidated("00000000-0000-0000-0000-000000000000")
	valid_urn := uuid.Text_Unvalidated("urn:uuid:00000000-0000-0000-0000-000000000000")
	valid_braced := uuid.Text_Unvalidated("{00000000-0000-0000-0000-000000000000}")
	valid_plain := uuid.Text_Unvalidated("00000000000000000000000000000000")
	invalid := uuid.Text_Unvalidated("x0000000-0000-0000-0000-000000000000")
	bracket := uuid.Text_Unvalidated("(00000000-0000-0000-0000-000000000000)")
	size := uuid.Text_Unvalidated("x")
	urn := uuid.Text_Unvalidated("bad:uuid:00000000-0000-0000-0000-000000000000")
	canonical_bytes := bytes.Slice([]byte(canonical))
	urn_valid_bytes := bytes.Slice([]byte(valid_urn))
	braced_valid_bytes := bytes.Slice([]byte(valid_braced))
	plain_valid_bytes := bytes.Slice([]byte(valid_plain))
	invalid_bytes := bytes.Slice([]byte(invalid))
	bracket_bytes := bytes.Slice([]byte(bracket))
	size_bytes := bytes.Slice([]byte(size))
	urn_bytes := bytes.Slice([]byte(urn))
	var raw [uuid.UUID_BYTE_COUNT]byte
	short := bytes.Slice(raw[:1])
	testify.Zero_Allocation(t, func() {
		uuid.Parse(value, canonical)
		uuid.Parse(value, valid_urn)
		uuid.Parse(value, valid_braced)
		uuid.Parse(value, valid_plain)
		uuid.Parse(value, invalid)
		uuid.Parse(value, bracket)
		uuid.Parse(value, size)
		uuid.Parse(value, urn)
		uuid.Parse_Bytes(value, canonical_bytes)
		uuid.Parse_Bytes(value, urn_valid_bytes)
		uuid.Parse_Bytes(value, braced_valid_bytes)
		uuid.Parse_Bytes(value, plain_valid_bytes)
		uuid.Parse_Bytes(value, invalid_bytes)
		uuid.Parse_Bytes(value, bracket_bytes)
		uuid.Parse_Bytes(value, size_bytes)
		uuid.Parse_Bytes(value, urn_bytes)
		uuid.Validate(canonical)
		uuid.Validate(invalid)
		uuid.Validate(bracket)
		uuid.Validate(size)
		uuid.Validate(urn)
		uuid.Must_Parse(value, canonical)
		uuid.Must(value, uuid.ERROR_NONE)
		uuid.From_Bytes(value, bytes.Slice(raw[:]))
		uuid.From_Bytes(value, short)
	})
}

// Borrowed text and collection storage make every representation retained without copy.
func zero_allocation_encoding(t *testing.T) {
	var value_storage [uuid.UUID_BYTE_COUNT]byte
	var other_storage [uuid.UUID_BYTE_COUNT]byte
	var text_storage [uuid.UUID_TEXT_BYTE_COUNT]byte
	var other_text_storage [uuid.UUID_TEXT_BYTE_COUNT]byte
	var urn_storage [uuid.UUID_URN_BYTE_COUNT]byte
	value := uuid.Must_Parse(
		uuid.UUID(value_storage[:]), "00112233-4455-1677-8899-aabbccddeeff",
	)
	value[9] = byte(uuid.DOMAIN_PERSON)
	other := uuid.UUID(other_storage[:])
	values := uuid.UUIDs{value, other}
	texts := uuid.Text_Storage{
		uuid.Text_Bytes(text_storage[:]), uuid.Text_Bytes(other_text_storage[:]),
	}
	rendered := uuid.Strings{"", ""}
	testify.Zero_Allocation(t, func() {
		uuid.UUID_String(value, uuid.Text_Bytes(text_storage[:]))
		uuid.UUID_URN(value, uuid.URN_Bytes(urn_storage[:]))
		uuid.UUID_Version(value)
		uuid.UUID_Variant(value)
		uuid.UUID_Node_Identifier(value)
		uuid.UUID_Domain(value)
		uuid.UUID_Identifier(value)
		uuid.UUID_Time(value)
		uuid.UUID_Clock_Sequence(value)
		uuid.UUIDs_Strings(values, texts, rendered)
		uuid.Time_Unix(uuid.UUID_Time(value))
	})
}

func zero_allocation_nullable(t *testing.T) {
	zero_allocation_uuid_io(t)
	zero_allocation_null_io(t)
}

// Driver sources exist before measurement because construction belongs to driver.
func zero_allocation_uuid_io(t *testing.T) {
	var value_storage [uuid.UUID_BYTE_COUNT]byte
	var destination_storage [uuid.UUID_BYTE_COUNT]byte
	var text_storage [uuid.UUID_TEXT_BYTE_COUNT]byte
	value := uuid.Must_Parse(
		uuid.UUID(value_storage[:]), "00112233-4455-1677-8899-aabbccddeeff",
	)
	destination := uuid.UUID(destination_storage[:])
	text := uuid.UUID_Marshal_Text(value, uuid.Text_Bytes(text_storage[:]))
	invalid := bytes.Slice([]byte("x0000000-0000-0000-0000-000000000000"))
	bracket := bytes.Slice([]byte("(00000000-0000-0000-0000-000000000000)"))
	size := bytes.Slice([]byte("x"))
	urn := bytes.Slice([]byte("bad:uuid:00000000-0000-0000-0000-000000000000"))
	var null_value driver.Value
	var text_value driver.Value
	var invalid_text_value driver.Value
	var bytes_value driver.Value
	var encoded_bytes_value driver.Value
	var empty_text_value driver.Value
	var empty_bytes_value driver.Value
	driver.Value_Null(&null_value)
	driver.Value_Of_Text(&text_value, driver.Text_Unvalidated(text_uuid(value)))
	driver.Value_Of_Text(&invalid_text_value, "x")
	driver.Value_Of_Bytes(&bytes_value, driver.Bytes_Unvalidated(value))
	driver.Value_Of_Bytes(&encoded_bytes_value, driver.Bytes_Unvalidated(text))
	driver.Value_Of_Text(&empty_text_value, "")
	driver.Value_Of_Bytes(&empty_bytes_value, driver.Bytes_Unvalidated{})
	other_value := driver.Value{Kind: driver.Value_Kind_Unvalidated(driver.VALUE_INTEGER)}
	bad_value := driver.Value{Kind: driver.Value_Kind_Unvalidated(driver.VALUE_TIME + 1)}
	testify.Zero_Allocation(t, func() {
		uuid.UUID_Marshal_Text(value, uuid.Text_Bytes(text_storage[:]))
		uuid.UUID_Unmarshal_Text(&destination, bytes.Slice(text))
		uuid.UUID_Unmarshal_Text(&destination, invalid)
		uuid.UUID_Unmarshal_Text(&destination, bracket)
		uuid.UUID_Unmarshal_Text(&destination, size)
		uuid.UUID_Unmarshal_Text(&destination, urn)
		uuid.UUID_Marshal_Binary(value)
		uuid.UUID_Unmarshal_Binary(&destination, bytes.Slice(value))
		uuid.UUID_Unmarshal_Binary(&destination, size)
		uuid.UUID_Scan(&destination, null_value)
		uuid.UUID_Scan(&destination, text_value)
		uuid.UUID_Scan(&destination, invalid_text_value)
		uuid.UUID_Scan(&destination, bytes_value)
		uuid.UUID_Scan(&destination, encoded_bytes_value)
		uuid.UUID_Scan(&destination, empty_text_value)
		uuid.UUID_Scan(&destination, empty_bytes_value)
		uuid.UUID_Scan(&destination, other_value)
		uuid.UUID_Scan(&destination, bad_value)
		uuid.UUID_Value(value, uuid.Text_Bytes(text_storage[:]))
	})
}

// Nullable outputs share caller buffers across present, absent, and rejected paths.
func zero_allocation_null_io(t *testing.T) {
	var value_storage [uuid.UUID_BYTE_COUNT]byte
	var destination_storage [uuid.UUID_BYTE_COUNT]byte
	var input_text_storage [uuid.UUID_TEXT_BYTE_COUNT]byte
	var output_text_storage [uuid.UUID_TEXT_BYTE_COUNT]byte
	var input_json_storage [uuid.UUID_BRACED_BYTE_COUNT]byte
	var output_json_storage [uuid.UUID_BRACED_BYTE_COUNT]byte
	value := uuid.Must_Parse(
		uuid.UUID(value_storage[:]), "00112233-4455-1677-8899-aabbccddeeff",
	)
	valid := uuid.Null_UUID{UUID: value, Valid: true}
	invalid := uuid.Null_UUID{UUID: uuid.UUID(destination_storage[:])}
	text := uuid.UUID_Marshal_Text(value, uuid.Text_Bytes(input_text_storage[:]))
	json := uuid.Null_UUID_Marshal_JSON(valid, uuid.JSON_Storage(input_json_storage[:]))
	bad_text := bytes.Slice([]byte("x"))
	bad_json := bytes.Slice([]byte("x"))
	null_json := bytes.Slice([]byte("null"))
	bad_body := bytes.Slice([]byte("\"x0112233-4455-1677-8899-aabbccddeeff\""))
	bad_open := bytes.Slice([]byte("!00112233-4455-1677-8899-aabbccddeeff\""))
	bad_close := bytes.Slice([]byte("\"00112233-4455-1677-8899-aabbccddeeff!"))
	var null_value driver.Value
	var text_value driver.Value
	var invalid_text_value driver.Value
	driver.Value_Null(&null_value)
	driver.Value_Of_Text(&text_value, driver.Text_Unvalidated(text_uuid(value)))
	driver.Value_Of_Text(&invalid_text_value, "x")
	bad_value := driver.Value{Kind: driver.Value_Kind_Unvalidated(driver.VALUE_TIME + 1)}
	testify.Zero_Allocation(t, func() {
		uuid.Null_UUID_Scan(&invalid, null_value)
		uuid.Null_UUID_Scan(&invalid, text_value)
		uuid.Null_UUID_Scan(&invalid, invalid_text_value)
		uuid.Null_UUID_Scan(&invalid, bad_value)
		uuid.Null_UUID_Value(valid, uuid.Text_Bytes(output_text_storage[:]))
		uuid.Null_UUID_Value(invalid, uuid.Text_Bytes(output_text_storage[:]))
		uuid.Null_UUID_Marshal_Binary(valid)
		uuid.Null_UUID_Marshal_Binary(invalid)
		uuid.Null_UUID_Unmarshal_Binary(&invalid, bytes.Slice(value))
		uuid.Null_UUID_Unmarshal_Binary(&invalid, bad_text)
		uuid.Null_UUID_Marshal_Text(
			valid, uuid.Nullable_Text_Storage(output_text_storage[:]),
		)
		uuid.Null_UUID_Marshal_Text(
			invalid, uuid.Nullable_Text_Storage(output_text_storage[:]),
		)
		uuid.Null_UUID_Unmarshal_Text(&invalid, bytes.Slice(text))
		uuid.Null_UUID_Unmarshal_Text(&invalid, bad_text)
		uuid.Null_UUID_Marshal_JSON(valid, uuid.JSON_Storage(output_json_storage[:]))
		uuid.Null_UUID_Marshal_JSON(invalid, uuid.JSON_Storage(output_json_storage[:]))
		uuid.Null_UUID_Unmarshal_JSON(&invalid, null_json)
		uuid.Null_UUID_Unmarshal_JSON(&invalid, bytes.Slice(json))
		uuid.Null_UUID_Unmarshal_JSON(&invalid, bad_json)
		uuid.Null_UUID_Unmarshal_JSON(&invalid, bad_body)
		uuid.Null_UUID_Unmarshal_JSON(&invalid, bad_open)
		uuid.Null_UUID_Unmarshal_JSON(&invalid, bad_close)
	})
}

// Nullable scan shares the same driver values but owns separate validity assertions.
func assert_nullable_scan(
	t *testing.T, original uuid.UUID, text_value driver.Value,
	status_ok driver.Validation_Status,
) {
	var null_value driver.Value
	driver.Value_Null(&null_value)
	nullable := uuid.Null_UUID{UUID: nil_uuid()}
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
	if !bool(bytes.Equal(bytes.Slice(nullable.UUID), bytes.Slice(original))) {
		t.Fatalf("nullable text did not survive")
	}
	var invalid_value driver.Value
	invalid_value_status := driver.Value_Of_Text(&invalid_value, "x")
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

// Test_Null_Text_And_Binary_Round_Trips covers both nullable wire forms.
func Test_Null_Text_And_Binary_Round_Trips(t *testing.T) {
	generator := fixed_generator(7)
	original := generated_v4(&generator)
	valid := uuid.Null_UUID{UUID: original, Valid: true}
	invalid := uuid.Null_UUID{UUID: nil_uuid()}
	binary_data := uuid.Null_UUID_Marshal_Binary(valid)
	empty_binary := uuid.Null_UUID_Marshal_Binary(invalid)
	if len(empty_binary) != 0 {
		t.Fatalf("marshal invalid binary = %v", empty_binary)
	}
	restored_binary := uuid.Null_UUID{UUID: nil_uuid(), Valid: true}
	if err := uuid.Null_UUID_Unmarshal_Binary(
		&restored_binary, bytes.Slice(binary_data),
	); err != uuid.ERROR_NONE {
		t.Fatalf("unmarshal binary: %v", err)
	}
	if !bool(bytes.Equal(bytes.Slice(restored_binary.UUID), bytes.Slice(original))) {
		t.Fatalf("binary UUID changed")
	}
	text_storage := make(uuid.Nullable_Text_Storage, uuid.UUID_TEXT_BYTE_COUNT)
	text := uuid.Null_UUID_Marshal_Text(valid, text_storage)
	null_storage := make(uuid.Nullable_Text_Storage, uuid.UUID_TEXT_BYTE_COUNT)
	null_text := uuid.Null_UUID_Marshal_Text(invalid, null_storage)
	if string(null_text) != uuid.JSON_NULL {
		t.Fatalf("marshal invalid text = %q", null_text)
	}
	restored_text := uuid.Null_UUID{UUID: nil_uuid(), Valid: true}
	if err := uuid.Null_UUID_Unmarshal_Text(
		&restored_text, bytes.Slice(text),
	); err != uuid.ERROR_NONE {
		t.Fatalf("unmarshal text: %v", err)
	}
	if !bool(bytes.Equal(bytes.Slice(restored_text.UUID), bytes.Slice(original))) {
		t.Fatalf("text UUID changed")
	}
}

// Test_Errors_Have_Static_Text reaches every bounded diagnostic.
func Test_Errors_Have_Static_Text(t *testing.T) {
	errors := []uuid.Error{
		uuid.ERROR_NONE,
		uuid.ERROR_INVALID_FORMAT,
		uuid.ERROR_INVALID_BRACKETED_FORMAT,
		uuid.ERROR_INVALID_SIZE,
		uuid.ERROR_INVALID_URN_PREFIX,
		uuid.ERROR_V7_STATE_EXHAUSTED,
		uuid.ERROR_V7_TIME_OUTPUT_OF_RANGE,
	}
	for _, failure := range errors {
		if uuid.Error_Text(failure) == "" {
			t.Fatalf("error %d has no text", failure)
		}
	}
}

// Test_Syntax_Statuses_Reach_Entries ties each public parser to its complete status domain.
func Test_Syntax_Statuses_Reach_Entries(t *testing.T) {
	canonical := "00000000-0000-0000-0000-000000000000"
	inputs := []uuid.Text_Unvalidated{
		uuid.Text_Unvalidated(canonical),
		"x0000000-0000-0000-0000-000000000000",
		"(" + uuid.Text_Unvalidated(canonical) + ")",
		"x",
		"bad:uuid:" + uuid.Text_Unvalidated(canonical),
	}
	wants := []uuid.Syntax_Status{
		uuid.ERROR_NONE,
		uuid.ERROR_INVALID_FORMAT,
		uuid.ERROR_INVALID_BRACKETED_FORMAT,
		uuid.ERROR_INVALID_SIZE,
		uuid.ERROR_INVALID_URN_PREFIX,
	}
	for index, input := range inputs {
		raw := bytes.Slice([]byte(input))
		if status := uuid.Parse(uuid_storage(), input); status != wants[index] {
			t.Fatalf("Parse status %d got %d", index, status)
		}
		if status := uuid.Parse_Bytes(uuid_storage(), raw); status != wants[index] {
			t.Fatalf("Parse_Bytes status %d got %d", index, status)
		}
		if status := uuid.Validate(input); status != wants[index] {
			t.Fatalf("Validate status %d got %d", index, status)
		}
		assert_text_status(t, raw, wants[index])
	}
}

func assert_text_status(t *testing.T, raw bytes.Slice, want uuid.Syntax_Status) {
	destination := nil_uuid()
	if status := uuid.UUID_Unmarshal_Text(&destination, raw); status != want {
		t.Fatalf("UUID_Unmarshal_Text got %d, want %d", status, want)
	}
	null_destination := uuid.Null_UUID{UUID: nil_uuid()}
	if status := uuid.Null_UUID_Unmarshal_Text(
		&null_destination, raw,
	); status != want {
		t.Fatalf("Null_UUID_Unmarshal_Text got %d, want %d", status, want)
	}
}

// Test_Must_Status_Boundaries proves both Version 7 failures stop the caller.
func Test_Must_Status_Boundaries(t *testing.T) {
	statuses := []uuid.Generation_Status{
		uuid.ERROR_NONE,
		uuid.ERROR_V7_STATE_EXHAUSTED,
		uuid.ERROR_V7_TIME_OUTPUT_OF_RANGE,
	}
	for _, status := range statuses {
		panicked := false
		func() {
			defer func() { panicked = recover() != nil }()
			uuid.Must(nil_uuid(), status)
		}()
		if panicked != (status != uuid.ERROR_NONE) {
			t.Fatalf("Must panic for status %d = %t", status, panicked)
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
		storage := uuid_storage()
		generator := generator_at(uint64(index), 0, uuid.Node{}, state)
		uuid.Generator_V4(&generator, storage)
		generator = generator_at(uint64(index), 0, uuid.Node{}, state)
		uuid.Generator_V1(&generator, storage)
		generator = generator_at(uint64(index), 0, uuid.Node{}, state)
		uuid.Generator_V6(&generator, storage)
		generator = generator_at(uint64(index), 0, uuid.Node{}, state)
		uuid.Generator_DCE_Security(&generator, storage, uuid.DOMAIN_PERSON, 0)
		generator = generator_at(uint64(index), 0, uuid.Node{}, state)
		_, failure := uuid.Generator_V7(&generator, storage)
		if state.Last_V7 == uuid.V7_State(uuid.V7_VALUE_MAXIMUM) {
			if failure == uuid.ERROR_NONE {
				t.Fatalf("state %d V7 accepted exhausted state", index)
			}
			continue
		}
		if failure != uuid.ERROR_NONE {
			t.Fatalf("state %d V7: %v", index, failure)
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
		uuid.Generator_V1(&generator, uuid_storage())
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
		value := generated_v1(&generator)
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
		if _, failure := uuid.Generator_V7(
			&generator, uuid_storage(),
		); failure != uuid.ERROR_NONE {
			t.Fatalf("V7 boundary %d: %v", index, failure)
		}
	}
	generator := generator_at(31, -1, uuid.Node{}, uuid.Generator_State{})
	if _, failure := uuid.Generator_V7(
		&generator, uuid_storage(),
	); failure == uuid.ERROR_NONE {
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
		destination := nil_uuid()
		uuid.Parse(destination, text)
		uuid.Parse_Bytes(destination, bytes.Slice(raw))
		uuid.From_Bytes(destination, bytes.Slice(raw))
		uuid.Validate(text)
		uuid.UUID_Unmarshal_Binary(&destination, bytes.Slice(raw))
		uuid.UUID_Unmarshal_Text(&destination, bytes.Slice(raw))
		null_destination := uuid.Null_UUID{UUID: nil_uuid()}
		uuid.Null_UUID_Unmarshal_Binary(&null_destination, bytes.Slice(raw))
		uuid.Null_UUID_Unmarshal_Text(&null_destination, bytes.Slice(raw))
		uuid.Null_UUID_Unmarshal_JSON(&null_destination, bytes.Slice(raw))
		var value driver.Value
		value_status := driver.Value_Of_Text(&value, driver.Text_Unvalidated(text))
		if value_status != driver.Validation_Status(driver.STATUS_OK) {
			t.Fatalf("build hostile text size %d: %v", size, value_status)
		}
		scanned := nil_uuid()
		uuid.UUID_Scan(&scanned, value)
		panicked := false
		func() {
			defer func() { panicked = recover() != nil }()
			uuid.Must_Parse(destination, text)
		}()
		if !panicked {
			t.Fatalf("Must_Parse accepted hostile size %d", size)
		}
	}
	raw := make([]byte, uuid.UUID_BYTE_COUNT)
	if failure := uuid.From_Bytes(
		uuid_storage(), bytes.Slice(raw),
	); failure != uuid.ERROR_NONE {
		t.Fatalf("From_Bytes rejected exact storage: %v", failure)
	}
}

// Test_Name_Boundaries_Reach_Hashes proves both hash versions accept their full
// bounded name domain without moving work or allocation into a hidden helper.
func Test_Name_Boundaries_Reach_Hashes(t *testing.T) {
	namespace := dns_namespace()
	for _, size := range []int{0, 1, 2, bytes.SLICE_SIZE_MAXIMUM} {
		name := make(uuid.Name, size)
		uuid.V3(uuid_storage(), namespace, name)
		uuid.V5(uuid_storage(), namespace, name)
	}
}

// Test_Decoded_Field_Boundaries proves accessors expose their complete declared
// domains from UUID bytes, not by calling invariant functions from the test.
func Test_Decoded_Field_Boundaries(t *testing.T) {
	for _, version := range []uuid.Version{0, 1, 2, uuid.VERSION_MAXIMUM} {
		value := nil_uuid()
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
		value := nil_uuid()
		value[variant_index] = test.Control
		if got := uuid.UUID_Variant(value); got != test.Want {
			t.Fatalf("variant control %d got %d, want %d", test.Control, got, test.Want)
		}
	}
	for _, timestamp := range []uuid.Time{0, 1, 2} {
		value := nil_uuid()
		binary.Put_Uint_32(
			binary.Bytes(value[:]), binary.Word_32(timestamp), binary.BIG_ENDIAN,
		)
		if got := uuid.UUID_Time(value); got != timestamp {
			t.Fatalf("time got %d, want %d", got, timestamp)
		}
	}
	if got := uuid.UUID_Time(uuid.Max(uuid_storage())); got != uuid.Time(uuid.TIME_MAXIMUM) {
		t.Fatalf("maximum time got %d, want %d", got, uuid.TIME_MAXIMUM)
	}
	for _, identifier := range []uuid.Identifier{
		0, 1, 2, uuid.Identifier(bits.WORD_32_MAXIMUM),
	} {
		value := nil_uuid()
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
		value := generated_dce(&generator, test.Domain, test.Identifier)
		if uuid.UUID_Domain(value) != test.Domain {
			t.Fatalf("domain %d did not survive", test.Domain)
		}
		if uuid.UUID_Identifier(value) != test.Identifier {
			t.Fatalf("identifier %d did not survive", test.Identifier)
		}
	}
}

// Test_Node_Boundaries_Reach_Generators proves complete 48-bit node storage.
func Test_Node_Boundaries_Reach_Generators(t *testing.T) {
	nodes := []uuid.Node{
		{},
		{Low: 1},
		{High: 1},
		{High: 2, Low: 2},
		{High: bits.Word_16(bits.WORD_16_MAXIMUM), Low: bits.Word_32(bits.WORD_32_MAXIMUM)},
	}
	for index, node := range nodes {
		storage := uuid_storage()
		generator := generator_at(uint64(index+60), 0, node, uuid.Generator_State{})
		uuid.Generator_V4(&generator, storage)
		generator = generator_at(uint64(index+70), 0, node, uuid.Generator_State{})
		uuid.Generator_V7(&generator, storage)
		generator = generator_at(uint64(index+80), 0, node, uuid.Generator_State{})
		version_1 := generated_v1(&generator)
		generator = generator_at(uint64(index+90), 0, node, uuid.Generator_State{})
		uuid.Generator_V6(&generator, storage)
		generator = generator_at(uint64(index+100), 0, node, uuid.Generator_State{})
		uuid.Generator_DCE_Security(&generator, storage, uuid.DOMAIN_PERSON, 0)
		resolved := uuid.UUID_Node_Identifier(version_1)
		if node != (uuid.Node{}) {
			if resolved != node {
				t.Fatalf("node %d changed", index)
			}
		}
	}
}

// Test_Database_Scan_Boundaries covers hostile closed-union payload dimensions.
func Test_Database_Scan_Boundaries(t *testing.T) {
	values := []driver.Value{
		{
			Kind:    driver.Value_Kind_Unvalidated(driver.VALUE_NULL),
			Integer: driver.Integer(bits.INTEGER_64_MINIMUM),
			Moment:  time.Moment(bits.INTEGER_64_MINIMUM),
		},
		{
			Kind:    driver.Value_Kind_Unvalidated(driver.VALUE_BOOLEAN),
			Boolean: true,
			Integer: 1,
			Float:   1,
			Bytes:   make(driver.Bytes, 1),
			Text:    "x",
			Moment:  1,
		},
		{
			Kind:    driver.Value_Kind_Unvalidated(driver.VALUE_INTEGER),
			Integer: 2,
			Float:   2,
			Bytes:   make(driver.Bytes, 2),
			Text:    "xx",
			Moment:  2,
		},
		{
			Kind:    driver.Value_Kind_Unvalidated(driver.VALUE_FLOAT),
			Integer: -1,
			Moment:  -1,
		},
		{
			Kind:    driver.Value_Kind_Unvalidated(driver.VALUE_TIME + 1),
			Boolean: true,
			Integer: driver.Integer(bits.INTEGER_64_MAXIMUM),
			Float:   driver.Float(bits.WORD_64_MAXIMUM),
			Bytes:   make(driver.Bytes, bytes.SLICE_SIZE_MAXIMUM),
			Text:    driver.Text(string(make([]byte, strings.TEXT_SIZE_MAXIMUM))),
			Moment:  time.Moment(bits.INTEGER_64_MAXIMUM),
		},
	}
	for _, value := range values {
		destination := nil_uuid()
		uuid.UUID_Scan(&destination, value)
		null_destination := uuid.Null_UUID{UUID: nil_uuid()}
		uuid.Null_UUID_Scan(&null_destination, value)
	}
}

// Test_Collection_And_Value_Boundaries_Reach_Production covers collection sizes,
// every comparison result, and both nullable database values.
func Test_Collection_And_Value_Boundaries_Reach_Production(t *testing.T) {
	for _, size := range []int{0, 1, 2, uuid.UUIDS_COUNT_MAXIMUM} {
		values := make(uuid.UUIDs, size)
		zero := nil_uuid()
		storage := make(uuid.Text_Storage, size)
		for index := range values {
			values[index] = zero
			storage[index] = make(uuid.Text_Bytes, uuid.UUID_TEXT_BYTE_COUNT)
		}
		rendered_storage := make(uuid.Strings, size)
		if rendered := uuid.UUIDs_Strings(
			values, storage, rendered_storage,
		); len(rendered) != size {
			t.Fatalf("rendered %d UUIDs, want %d", len(rendered), size)
		}
	}
	zero := nil_uuid()
	uuid_max := uuid.Max(uuid_storage())
	if bytes.Compare(bytes.Slice(zero), bytes.Slice(zero)) != 0 {
		t.Fatalf("equal UUIDs did not compare equal")
	}
	if bytes.Compare(bytes.Slice(uuid_max), bytes.Slice(zero)) != 1 {
		t.Fatalf("maximum UUID did not compare after nil")
	}
	text_storage := make(uuid.Text_Bytes, uuid.UUID_TEXT_BYTE_COUNT)
	null_value := uuid.Null_UUID_Value(uuid.Null_UUID{UUID: nil_uuid()}, text_storage)
	null_kind, null_status := driver.Value_Kind_Of(driver.Value(null_value))
	if null_status != driver.Validation_Status(driver.STATUS_OK) {
		t.Fatalf("invalid nullable value kind: %v", null_status)
	}
	if null_kind != driver.VALUE_NULL {
		t.Fatalf("invalid nullable value is not database null")
	}
	valid_value := uuid.Null_UUID_Value(
		uuid.Null_UUID{UUID: uuid_max, Valid: true}, text_storage,
	)
	valid_kind, valid_status := driver.Value_Kind_Of(driver.Value(valid_value))
	if valid_status != driver.Validation_Status(driver.STATUS_OK) {
		t.Fatalf("valid nullable value kind: %v", valid_status)
	}
	if valid_kind != driver.VALUE_TEXT {
		t.Fatalf("valid nullable value is not database text")
	}
}

// Test_Time_Unix_Boundaries_Reach_Production proves timestamp and Unix-second
// endpoints plus exact signed 100-nanosecond remainders.
func Test_Time_Unix_Boundaries_Reach_Production(t *testing.T) {
	generator := generator_at(50, 0, uuid.Node{}, uuid.Generator_State{})
	epoch := uuid.UUID_Time(generated_v7(&generator))
	timestamps := []uuid.Time{
		0,
		1,
		2,
		uuid.Time(uuid.TIME_MAXIMUM),
		epoch - uuid.Time(uuid.UUID_TICK_COUNT_PER_SECOND),
		epoch - 1,
		epoch,
		epoch + 1,
		epoch + 2,
		epoch + uuid.Time(uuid.UUID_TICK_COUNT_PER_SECOND),
		epoch + uuid.Time(2*uuid.UUID_TICK_COUNT_PER_SECOND),
	}
	for _, timestamp := range timestamps {
		uuid.Time_Unix(timestamp)
	}
	_, minimum := uuid.Time_Unix(
		epoch - uuid.Time(uuid.UUID_TICK_COUNT_PER_SECOND-1),
	)
	minimum_nanoseconds := int64(minimum.Ticks) * uuid.UUID_TICK_NANOSECOND_COUNT
	if minimum_nanoseconds != uuid.NANOSECOND_MINIMUM {
		t.Fatalf(
			"nanosecond minimum got %d, want %d",
			minimum_nanoseconds, uuid.NANOSECOND_MINIMUM,
		)
	}
	_, maximum := uuid.Time_Unix(
		epoch + uuid.Time(uuid.UUID_TICK_COUNT_PER_SECOND-1),
	)
	maximum_nanoseconds := int64(maximum.Ticks) * uuid.UUID_TICK_NANOSECOND_COUNT
	if maximum_nanoseconds != uuid.NANOSECOND_MAXIMUM {
		t.Fatalf(
			"nanosecond maximum got %d, want %d",
			maximum_nanoseconds, uuid.NANOSECOND_MAXIMUM,
		)
	}
}
