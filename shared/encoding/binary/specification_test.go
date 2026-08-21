package binary_test

import (
	"io"
	"testing"

	"local/james-orcales/shared/encoding/binary"
)

// Test_Allocation keeps public data paths free from hidden heap ownership.
func Test_Allocation(t *testing.T) {
	var storage [FIXED_STORAGE_SIZE]byte
	var value binary.Word_64
	call := func() {
		binary.Put_Uint_64(storage[:], value, binary.BIG_ENDIAN)
		value = binary.Uint_64(storage[:], binary.BIG_ENDIAN)
	}
	if testing.AllocsPerRun(100, call) != 0 {
		t.Fatal("fixed-width API allocated heap storage")
	}
}

// Test_Bounds keeps byte and varint domains tied to repository limit.
func Test_Bounds(t *testing.T) {
	if binary.BYTE_SIZE_MINIMUM != 0 {
		t.Fatal("byte minimum is not empty storage")
	}
	if binary.BYTE_SIZE_MAXIMUM != 4096 {
		t.Fatal("byte maximum differs from repository boundary")
	}
	if binary.VALUE_SIZE_INVALID != -1 {
		t.Fatal("invalid structured size sentinel changed")
	}
	verify_fixed_width_boundaries(t)
	verify_varint_boundaries(t)
	verify_structured_boundaries(t)
	verify_stream_boundaries(t)
}

// Test_Byte_Order keeps three order identities and published byte layouts.
func Test_Byte_Order(t *testing.T) {
	verify_byte_order_names(t)
	verify_fixed_width_bytes(t)
}

// Test_Fixed_Width protects both byte orders, full-width values, and caller-owned append.
func Test_Fixed_Width(t *testing.T) {
	verify_fixed_width(t)
}

// Test_Structured_Values keeps reflection grammar on caller storage.
func Test_Structured_Values(t *testing.T) {
	source := structured_value{First: 0x0102, Second: 0x03040506}
	var storage [binary.UINT_16_SIZE + binary.UINT_32_SIZE]byte
	count, err := binary.Encode(storage[:], source, binary.BIG_ENDIAN)
	if err != nil {
		t.Fatal("Encode rejected fixed-size structure")
	}
	if count != binary.Byte_Count(len(storage)) {
		t.Fatal("Encode returned wrong structured size")
	}
	var destination structured_value
	count, err = binary.Decode(storage[:], &destination, binary.BIG_ENDIAN)
	if err != nil {
		t.Fatal("Decode rejected fixed-size structure")
	}
	if count != binary.Byte_Count(len(storage)) {
		t.Fatal("Decode returned wrong structured size")
	}
	if destination != source {
		t.Fatal("Decode returned wrong structured value")
	}
}

// Test_Stream_IO keeps scratch storage explicit at blocking boundary.
func Test_Stream_IO(t *testing.T) {
	var storage [binary.UINT_64_SIZE]byte
	stream := byte_stream{Source: storage[:]}
	value := uint64(0x0102030405060708)
	if binary.Write(&stream, storage[:], value, binary.BIG_ENDIAN) != nil {
		t.Fatal("Write rejected fixed-size value")
	}
	stream.Source = stream.Storage[:binary.UINT_64_SIZE]
	var decoded uint64
	if binary.Read(&stream, storage[:], &decoded, binary.BIG_ENDIAN) != nil {
		t.Fatal("Read rejected fixed-size value")
	}
	if decoded != value {
		t.Fatal("Read returned wrong fixed-size value")
	}
}

// Test_Varints ports upstream value and malformed-input sweeps.
func Test_Varints(t *testing.T) {
	verify_varint_constants(t)
	verify_varint_round_trip(t)
	verify_varint_incomplete_and_overflow(t)
	verify_varint_noncanonical_zero(t)
}

// Test_Domain_Errors protects panic and sentinel boundaries before mutation.
func Test_Domain_Errors(t *testing.T) {
	verify_fixed_width_domain_errors(t)
	var storage [FIXED_STORAGE_SIZE]byte
	if _, err := binary.Encode(
		storage[:binary.UINT_16_SIZE], uint64(1), binary.BIG_ENDIAN,
	); err != binary.Error_Buffer_Too_Small {
		t.Fatal("Encode returned wrong short-buffer error")
	}
}

const PREFIX_SIZE = 3
const FIXED_STORAGE_SIZE = binary.UINT_64_SIZE

type structured_value struct {
	First  uint16
	Second uint32
}

type byte_reader struct {
	Source   []byte
	Position int
}

type byte_stream struct {
	Storage  [binary.BYTE_SIZE_MAXIMUM]byte
	Source   []byte
	Position int
}

// Read gives stream behavior without storage ownership.
func (stream *byte_stream) Read(destination []byte) (count int, err error) {
	if stream.Position == len(stream.Source) {
		return 0, io.EOF
	}
	count = copy(destination, stream.Source[stream.Position:])
	stream.Position += count
	return count, nil
}

// Write gives stream behavior without storage growth.
func (stream *byte_stream) Write(source []byte) (count int, err error) {
	return copy(stream.Storage[:], source), nil
}

// Caller owns cursor, so varint reader needs no hidden storage.
func read_byte(reader *byte_reader) (value byte, err error) {
	if reader.Position == len(reader.Source) {
		return 0, io.EOF
	}
	value = reader.Source[reader.Position]
	reader.Position++
	return value, nil
}

// Each public fixed-width boundary must reject short storage and accept the package maximum.
func verify_fixed_width_boundaries(t *testing.T) {
	var storage [binary.BYTE_SIZE_MAXIMUM]byte
	verify_uint_16_boundaries(t, storage[:])
	verify_uint_32_boundaries(t, storage[:])
	verify_uint_64_boundaries(t, storage[:])
}

func verify_uint_16_boundaries(t *testing.T, storage []byte) {
	for index := 0; index < binary.UINT_16_SIZE; index++ {
		if !panics(func() { binary.Uint_16(storage[:index], binary.BIG_ENDIAN) }) {
			t.Fatal("Uint_16 accepted short source")
		}
		if !panics(func() {
			binary.Put_Uint_16(storage[:index], 0, binary.BIG_ENDIAN)
		}) {
			t.Fatal("Put_Uint_16 accepted short destination")
		}
	}
	binary.Put_Uint_16(storage, 2, binary.NATIVE_ENDIAN)
	if binary.Uint_16(storage, binary.NATIVE_ENDIAN) != 2 {
		t.Fatal("Uint_16 rejected maximum storage")
	}
	binary.Put_Uint_16(storage, 1, binary.BIG_ENDIAN)
	if binary.Uint_16(storage, binary.BIG_ENDIAN) != 1 {
		t.Fatal("Uint_16 changed one")
	}
	for index := 0; index <= binary.UINT_16_SIZE; index++ {
		binary.Append_Uint_16(storage[:index], 1, binary.BIG_ENDIAN)
	}
	binary.Append_Uint_16(storage[:0], 2, binary.NATIVE_ENDIAN)
	result := binary.Append_Uint_16(
		storage[:binary.BYTE_SIZE_MAXIMUM-binary.UINT_16_SIZE], 1,
		binary.BIG_ENDIAN,
	)
	if len(result) != binary.BYTE_SIZE_MAXIMUM {
		t.Fatal("Append_Uint_16 rejected maximum result")
	}
	if !panics(func() {
		binary.Append_Uint_16(storage[:], 1, binary.BIG_ENDIAN)
	}) {
		t.Fatal("Append_Uint_16 accepted oversized result")
	}
}

func verify_uint_32_boundaries(t *testing.T, storage []byte) {
	for index := 0; index <= binary.UINT_16_SIZE; index++ {
		if !panics(func() { binary.Uint_32(storage[:index], binary.BIG_ENDIAN) }) {
			t.Fatal("Uint_32 accepted short source")
		}
		if !panics(func() {
			binary.Put_Uint_32(storage[:index], 0, binary.BIG_ENDIAN)
		}) {
			t.Fatal("Put_Uint_32 accepted short destination")
		}
	}
	binary.Put_Uint_32(storage, 2, binary.NATIVE_ENDIAN)
	if binary.Uint_32(storage, binary.NATIVE_ENDIAN) != 2 {
		t.Fatal("Uint_32 rejected maximum storage")
	}
	binary.Put_Uint_32(storage, 1, binary.BIG_ENDIAN)
	if binary.Uint_32(storage, binary.BIG_ENDIAN) != 1 {
		t.Fatal("Uint_32 changed one")
	}
	for index := 0; index <= binary.UINT_16_SIZE; index++ {
		binary.Append_Uint_32(storage[:index], 1, binary.BIG_ENDIAN)
	}
	binary.Append_Uint_32(storage[:0], 2, binary.NATIVE_ENDIAN)
	result := binary.Append_Uint_32(
		storage[:binary.BYTE_SIZE_MAXIMUM-binary.UINT_32_SIZE], 1,
		binary.BIG_ENDIAN,
	)
	if len(result) != binary.BYTE_SIZE_MAXIMUM {
		t.Fatal("Append_Uint_32 rejected maximum result")
	}
	if !panics(func() {
		binary.Append_Uint_32(storage[:], 1, binary.BIG_ENDIAN)
	}) {
		t.Fatal("Append_Uint_32 accepted oversized result")
	}
}

func verify_uint_64_boundaries(t *testing.T, storage []byte) {
	for index := 0; index <= binary.UINT_16_SIZE; index++ {
		if !panics(func() { binary.Uint_64(storage[:index], binary.BIG_ENDIAN) }) {
			t.Fatal("Uint_64 accepted short source")
		}
		if !panics(func() {
			binary.Put_Uint_64(storage[:index], 0, binary.BIG_ENDIAN)
		}) {
			t.Fatal("Put_Uint_64 accepted short destination")
		}
	}
	binary.Put_Uint_64(storage, 2, binary.NATIVE_ENDIAN)
	if binary.Uint_64(storage, binary.NATIVE_ENDIAN) != 2 {
		t.Fatal("Uint_64 rejected maximum storage")
	}
	binary.Put_Uint_64(storage, 1, binary.BIG_ENDIAN)
	if binary.Uint_64(storage, binary.BIG_ENDIAN) != 1 {
		t.Fatal("Uint_64 changed one")
	}
	for index := 0; index <= binary.UINT_16_SIZE; index++ {
		binary.Append_Uint_64(storage[:index], 1, binary.BIG_ENDIAN)
	}
	binary.Append_Uint_64(storage[:0], 2, binary.NATIVE_ENDIAN)
	result := binary.Append_Uint_64(
		storage[:binary.BYTE_SIZE_MAXIMUM-binary.UINT_64_SIZE], 1,
		binary.BIG_ENDIAN,
	)
	if len(result) != binary.BYTE_SIZE_MAXIMUM {
		t.Fatal("Append_Uint_64 rejected maximum result")
	}
	if !panics(func() {
		binary.Append_Uint_64(storage[:], 1, binary.BIG_ENDIAN)
	}) {
		t.Fatal("Append_Uint_64 accepted oversized result")
	}
}

func verify_varint_boundaries(t *testing.T) {
	var storage [binary.BYTE_SIZE_MAXIMUM]byte
	if !panics(func() { binary.Put_Unsigned_Varint(storage[:0], 0) }) {
		t.Fatal("Put_Unsigned_Varint accepted empty destination")
	}
	if binary.Put_Unsigned_Varint(storage[:], binary.Word_64(^uint64(0))) != 10 {
		t.Fatal("Put_Unsigned_Varint rejected maximum destination")
	}
	if !panics(func() { binary.Put_Varint(storage[:0], 0) }) {
		t.Fatal("Put_Varint accepted empty destination")
	}
	if binary.Put_Varint(storage[:1], 0) != 1 {
		t.Fatal("Put_Varint rejected one-byte destination")
	}
	if binary.Put_Varint(storage[:2], 64) != 2 {
		t.Fatal("Put_Varint rejected two-byte destination")
	}
	binary.Put_Varint(storage[:], -1<<63)
	verify_varint_append_boundaries(t, storage[:])
	verify_varint_decode_boundaries(t, storage[:])
}

func verify_varint_append_boundaries(t *testing.T, storage []byte) {
	if len(binary.Append_Unsigned_Varint(storage[:0], 128)) != 2 {
		t.Fatal("Append_Unsigned_Varint rejected two-byte result")
	}
	binary.Append_Unsigned_Varint(storage[:1], 0)
	binary.Append_Unsigned_Varint(storage[:2], 0)
	unsigned_result := binary.Append_Unsigned_Varint(
		storage[:binary.BYTE_SIZE_MAXIMUM-1], 0,
	)
	if len(unsigned_result) != binary.BYTE_SIZE_MAXIMUM {
		t.Fatal("Append_Unsigned_Varint rejected maximum result")
	}
	if !panics(func() { binary.Append_Unsigned_Varint(storage, 0) }) {
		t.Fatal("Append_Unsigned_Varint accepted oversized result")
	}
	if len(binary.Append_Varint(storage[:0], 64)) != 2 {
		t.Fatal("Append_Varint rejected two-byte result")
	}
	binary.Append_Varint(storage[:1], 0)
	binary.Append_Varint(storage[:2], 0)
	signed_result := binary.Append_Varint(storage[:binary.BYTE_SIZE_MAXIMUM-1], 0)
	if len(signed_result) != binary.BYTE_SIZE_MAXIMUM {
		t.Fatal("Append_Varint rejected maximum result")
	}
	if !panics(func() { binary.Append_Varint(storage, 0) }) {
		t.Fatal("Append_Varint accepted oversized result")
	}
}

func verify_varint_decode_boundaries(t *testing.T, storage []byte) {
	for index := range storage {
		storage[index] = 0x80
	}
	if _, count := binary.Unsigned_Varint(storage); count != -11 {
		t.Fatal("Unsigned_Varint rejected maximum source")
	}
	if _, count := binary.Varint(storage[:0]); count != 0 {
		t.Fatal("Varint rejected empty source")
	}
	if _, count := binary.Varint(storage); count != -11 {
		t.Fatal("Varint rejected maximum source")
	}
}

func verify_structured_boundaries(t *testing.T) {
	var storage [binary.BYTE_SIZE_MAXIMUM]byte
	var maximum [binary.BYTE_SIZE_MAXIMUM]byte
	if binary.Size(&maximum) != binary.VALUE_SIZE_MAXIMUM {
		t.Fatal("Size rejected maximum structured value")
	}
	verify_encode_boundary(t, storage[:0], struct{}{}, binary.LITTLE_ENDIAN, 0)
	verify_encode_boundary(t, storage[:1], uint8(1), binary.NATIVE_ENDIAN, 1)
	verify_encode_boundary(t, storage[:2], uint16(1), binary.BIG_ENDIAN, 2)
	verify_encode_boundary(
		t, storage[:], &maximum, binary.BIG_ENDIAN, binary.BYTE_SIZE_MAXIMUM,
	)
	var empty struct{}
	var octet uint8
	var word uint16
	var decoded [binary.BYTE_SIZE_MAXIMUM]byte
	verify_decode_boundary(t, storage[:0], &empty, binary.LITTLE_ENDIAN, 0)
	verify_decode_boundary(t, storage[:1], &octet, binary.NATIVE_ENDIAN, 1)
	verify_decode_boundary(t, storage[:2], &word, binary.BIG_ENDIAN, 2)
	verify_decode_boundary(
		t, storage[:], &decoded, binary.BIG_ENDIAN, binary.BYTE_SIZE_MAXIMUM,
	)
	verify_append_boundaries(t, storage[:])
}

func verify_encode_boundary(
	t *testing.T, destination []byte, source any,
	order binary.Byte_Order, want binary.Byte_Count,
) {
	count, err := binary.Encode(destination, source, order)
	if err != nil {
		t.Fatal("Encode rejected structured boundary")
	}
	if count != want {
		t.Fatal("Encode returned wrong structured boundary count")
	}
}

func verify_decode_boundary(
	t *testing.T, source []byte, destination any,
	order binary.Byte_Order, want binary.Byte_Count,
) {
	count, err := binary.Decode(source, destination, order)
	if err != nil {
		t.Fatal("Decode rejected structured boundary")
	}
	if count != want {
		t.Fatal("Decode returned wrong structured boundary count")
	}
}

func verify_append_boundaries(t *testing.T, storage []byte) {
	result, err := binary.Append(storage[:0], struct{}{}, binary.LITTLE_ENDIAN)
	if err != nil {
		t.Fatal("Append rejected empty result")
	}
	if len(result) != 0 {
		t.Fatal("Append returned wrong empty result")
	}
	result, err = binary.Append(storage[:0], uint8(1), binary.NATIVE_ENDIAN)
	if err != nil {
		t.Fatal("Append rejected one-byte result")
	}
	if len(result) != 1 {
		t.Fatal("Append returned wrong one-byte result")
	}
	result, err = binary.Append(storage[:1], uint8(1), binary.BIG_ENDIAN)
	if err != nil {
		t.Fatal("Append rejected two-byte result")
	}
	if len(result) != 2 {
		t.Fatal("Append returned wrong two-byte result")
	}
	result, err = binary.Append(storage[:2], struct{}{}, binary.BIG_ENDIAN)
	if err != nil {
		t.Fatal("Append rejected two-byte buffer")
	}
	result, err = binary.Append(storage, struct{}{}, binary.BIG_ENDIAN)
	if err != nil {
		t.Fatal("Append rejected maximum buffer")
	}
	if len(result) != binary.BYTE_SIZE_MAXIMUM {
		t.Fatal("Append rejected maximum result")
	}
}

func verify_stream_boundaries(t *testing.T) {
	var scratch [binary.BYTE_SIZE_MAXIMUM]byte
	var maximum [binary.BYTE_SIZE_MAXIMUM]byte
	var empty struct{}
	var octet uint8
	var word uint16
	verify_read_boundary(t, scratch[:0], &empty, binary.LITTLE_ENDIAN)
	verify_read_boundary(t, scratch[:1], &octet, binary.NATIVE_ENDIAN)
	verify_read_boundary(t, scratch[:2], &word, binary.BIG_ENDIAN)
	verify_read_boundary(t, scratch[:], &maximum, binary.BIG_ENDIAN)
	verify_write_boundary(t, scratch[:0], empty, binary.LITTLE_ENDIAN)
	verify_write_boundary(t, scratch[:1], octet, binary.NATIVE_ENDIAN)
	verify_write_boundary(t, scratch[:2], word, binary.BIG_ENDIAN)
	verify_write_boundary(t, scratch[:], &maximum, binary.BIG_ENDIAN)
}

func verify_read_boundary(
	t *testing.T, scratch []byte, destination any, order binary.Byte_Order,
) {
	stream := byte_stream{Source: scratch}
	if binary.Read(&stream, scratch, destination, order) != nil {
		t.Fatal("Read rejected structured boundary")
	}
}

func verify_write_boundary(
	t *testing.T, scratch []byte, source any, order binary.Byte_Order,
) {
	var stream byte_stream
	if binary.Write(&stream, scratch, source, order) != nil {
		t.Fatal("Write rejected structured boundary")
	}
}

func verify_fixed_width(t *testing.T) {
	values := [...]binary.Word_64{
		0,
		0x0123456789abcdef,
		0xffffffffffffffff,
	}
	orders := [...]binary.Byte_Order{binary.LITTLE_ENDIAN, binary.BIG_ENDIAN}
	for _, order := range orders {
		for _, value := range values {
			var storage [binary.BYTE_SIZE_MAXIMUM]byte
			binary.Put_Uint_16(storage[:2], binary.Word_16(value), order)
			if binary.Uint_16(storage[:2], order) != binary.Word_16(value) {
				t.Fatal("Uint_16 did not reverse Put_Uint_16")
			}
			binary.Put_Uint_32(storage[:4], binary.Word_32(value), order)
			if binary.Uint_32(storage[:4], order) != binary.Word_32(value) {
				t.Fatal("Uint_32 did not reverse Put_Uint_32")
			}
			binary.Put_Uint_64(storage[:8], value, order)
			if binary.Uint_64(storage[:8], order) != value {
				t.Fatal("Uint_64 did not reverse Put_Uint_64")
			}
			buffer := storage[: PREFIX_SIZE : PREFIX_SIZE+8]
			buffer = binary.Append_Uint_16(buffer, binary.Word_16(value), order)
			if len(buffer) != PREFIX_SIZE+2 {
				t.Fatal("Append_Uint_16 returned wrong buffer")
			}
			if binary.Uint_16(buffer[PREFIX_SIZE:], order) != binary.Word_16(value) {
				t.Fatal("Append_Uint_16 returned wrong buffer")
			}
			buffer = storage[: PREFIX_SIZE : PREFIX_SIZE+8]
			buffer = binary.Append_Uint_32(buffer, binary.Word_32(value), order)
			if len(buffer) != PREFIX_SIZE+4 {
				t.Fatal("Append_Uint_32 returned wrong buffer")
			}
			if binary.Uint_32(buffer[PREFIX_SIZE:], order) != binary.Word_32(value) {
				t.Fatal("Append_Uint_32 returned wrong buffer")
			}
			buffer = storage[: PREFIX_SIZE : PREFIX_SIZE+8]
			buffer = binary.Append_Uint_64(buffer, value, order)
			if len(buffer) != PREFIX_SIZE+8 {
				t.Fatal("Append_Uint_64 returned wrong buffer")
			}
			if binary.Uint_64(buffer[PREFIX_SIZE:], order) != value {
				t.Fatal("Append_Uint_64 returned wrong buffer")
			}
		}
	}
}

// Published byte layout stays independent from round trips.
func verify_fixed_width_bytes(t *testing.T) {
	var storage [FIXED_STORAGE_SIZE]byte
	binary.Put_Uint_64(storage[:], 0x0123456789abcdef, binary.BIG_ENDIAN)
	want_big := [...]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef}
	if storage != want_big {
		t.Fatal("BIG_ENDIAN wrote wrong byte order")
	}
	binary.Put_Uint_64(storage[:], 0x0123456789abcdef, binary.LITTLE_ENDIAN)
	want_little := [...]byte{0xef, 0xcd, 0xab, 0x89, 0x67, 0x45, 0x23, 0x01}
	if storage != want_little {
		t.Fatal("LITTLE_ENDIAN wrote wrong byte order")
	}
}

// Method-free names retain standard identities.
func verify_byte_order_names(t *testing.T) {
	cases := [...]struct {
		Order binary.Byte_Order
		Name  binary.Byte_Order_Name
		Go    binary.Byte_Order_Go_Name
	}{
		{Order: binary.LITTLE_ENDIAN, Name: "LittleEndian", Go: "binary.LittleEndian"},
		{Order: binary.BIG_ENDIAN, Name: "BigEndian", Go: "binary.BigEndian"},
		{Order: binary.NATIVE_ENDIAN, Name: "NativeEndian", Go: "binary.NativeEndian"},
	}
	for _, one := range cases {
		if binary.Byte_Order_String(one.Order) != one.Name {
			t.Fatal("Byte_Order_String returned wrong name")
		}
		if binary.Byte_Order_Go_String(one.Order) != one.Go {
			t.Fatal("Byte_Order_Go_String returned wrong name")
		}
	}
}

// Fixed-width failure happens before partial writes.
func verify_fixed_width_domain_errors(t *testing.T) {
	var storage [FIXED_STORAGE_SIZE]byte
	if !panics(func() {
		binary.Uint_64(storage[:FIXED_STORAGE_SIZE-1], binary.LITTLE_ENDIAN)
	}) {
		t.Fatal("Uint_64 accepted short source")
	}
	if !panics(func() {
		binary.Put_Uint_64(
			storage[:FIXED_STORAGE_SIZE-1], 1, binary.LITTLE_ENDIAN,
		)
	}) {
		t.Fatal("Put_Uint_64 accepted short destination")
	}
	if !panics(func() {
		binary.Append_Uint_64(storage[:1:1], 1, binary.LITTLE_ENDIAN)
	}) {
		t.Fatal("Append_Uint_64 accepted missing capacity")
	}
	if !panics(func() { binary.Uint_16(storage[:2], binary.Byte_Order(99)) }) {
		t.Fatal("Uint_16 accepted invalid byte order")
	}
}

// Published maxima remain derived from wire format.
func verify_varint_constants(t *testing.T) {
	if binary.VARINT_SIZE_16_MAXIMUM != 3 {
		t.Fatal("16-bit varint maximum differs from format")
	}
	if binary.VARINT_SIZE_32_MAXIMUM != 5 {
		t.Fatal("32-bit varint maximum differs from format")
	}
	if binary.VARINT_SIZE_64_MAXIMUM != 10 {
		t.Fatal("varint maximum lengths differ from format")
	}
}

// Upstream signed and unsigned boundary sweep stays intact.
func verify_varint_round_trip(t *testing.T) {
	values := [...]binary.Integer_64{
		-1 << 63,
		-1<<63 + 1,
		-1,
		0,
		1,
		2,
		10,
		20,
		63,
		64,
		65,
		127,
		128,
		129,
		255,
		256,
		257,
		1<<63 - 1,
	}
	for _, value := range values {
		verify_signed_varint(t, value)
		verify_signed_varint(t, -value)
		verify_unsigned_varint(t, binary.Word_64(value))
	}
	for value := binary.Integer_64(0x7); value != 0; value <<= 1 {
		verify_signed_varint(t, value)
		verify_signed_varint(t, -value)
	}
	for value := binary.Word_64(0x7); value != 0; value <<= 1 {
		verify_unsigned_varint(t, value)
	}
}

func verify_signed_varint(t *testing.T, value binary.Integer_64) {
	t.Helper()
	var storage [binary.VARINT_SIZE_64_MAXIMUM]byte
	count := binary.Put_Varint(storage[:], value)
	decoded, decoded_count := binary.Varint(storage[:count])
	if decoded != value {
		t.Fatal("Varint did not reverse Put_Varint")
	}
	if decoded_count != binary.Varint_Count(count) {
		t.Fatal("Varint did not reverse Put_Varint")
	}
	var appended_storage [binary.VARINT_SIZE_64_MAXIMUM + PREFIX_SIZE]byte
	buffer := appended_storage[:PREFIX_SIZE:len(appended_storage)]
	buffer = binary.Append_Varint(buffer, value)
	if len(buffer) != PREFIX_SIZE+int(count) {
		t.Fatal("Append_Varint returned wrong size")
	}
	destination_index := PREFIX_SIZE
	for source_index := range int(count) {
		if buffer[destination_index] != storage[source_index] {
			t.Fatal("Append_Varint wrote wrong bytes")
		}
		destination_index++
	}
	reader := byte_reader{Source: storage[:count]}
	read, err := binary.Read_Varint(&reader, read_byte)
	if err != nil {
		t.Fatal("Read_Varint returned error")
	}
	if read != value {
		t.Fatal("Read_Varint returned wrong value")
	}
}

func verify_unsigned_varint(t *testing.T, value binary.Word_64) {
	t.Helper()
	var storage [binary.VARINT_SIZE_64_MAXIMUM]byte
	count := binary.Put_Unsigned_Varint(storage[:], value)
	decoded, decoded_count := binary.Unsigned_Varint(storage[:count])
	if decoded != value {
		t.Fatal("Unsigned_Varint did not reverse Put_Unsigned_Varint")
	}
	if decoded_count != binary.Varint_Count(count) {
		t.Fatal("Unsigned_Varint did not reverse Put_Unsigned_Varint")
	}
	var appended_storage [binary.VARINT_SIZE_64_MAXIMUM + PREFIX_SIZE]byte
	buffer := appended_storage[:PREFIX_SIZE:len(appended_storage)]
	buffer = binary.Append_Unsigned_Varint(buffer, value)
	if len(buffer) != PREFIX_SIZE+int(count) {
		t.Fatal("Append_Unsigned_Varint returned wrong size")
	}
	destination_index := PREFIX_SIZE
	for source_index := range int(count) {
		if buffer[destination_index] != storage[source_index] {
			t.Fatal("Append_Unsigned_Varint wrote wrong bytes")
		}
		destination_index++
	}
	reader := byte_reader{Source: storage[:count]}
	read, err := binary.Read_Unsigned_Varint(&reader, read_byte)
	if err != nil {
		t.Fatal("Read_Unsigned_Varint returned error")
	}
	if read != value {
		t.Fatal("Read_Unsigned_Varint returned wrong value")
	}
}

// Upstream malformed-input behavior distinguishes truncation from overflow.
func verify_varint_incomplete_and_overflow(t *testing.T) {
	incomplete := [...]byte{0x80, 0x80, 0x80, 0x80}
	for size_index := range len(incomplete) + 1 {
		value, count := binary.Unsigned_Varint(incomplete[:size_index])
		if value != 0 {
			t.Fatal("Unsigned_Varint accepted incomplete value")
		}
		if count != 0 {
			t.Fatal("Unsigned_Varint accepted incomplete value")
		}
		reader := byte_reader{Source: incomplete[:size_index]}
		read, err := binary.Read_Unsigned_Varint(&reader, read_byte)
		want := io.EOF
		if size_index > 0 {
			want = io.ErrUnexpectedEOF
		}
		if read != 0 {
			t.Fatal("Read_Unsigned_Varint returned wrong incomplete value")
		}
		if err != want {
			t.Fatal("Read_Unsigned_Varint returned wrong incomplete error")
		}
	}
	overflow_tenth := [...]byte{
		0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x02,
	}
	value, count := binary.Unsigned_Varint(overflow_tenth[:])
	if value != 0 {
		t.Fatal("Unsigned_Varint returned value for tenth-byte overflow")
	}
	if count != -10 {
		t.Fatal("Unsigned_Varint missed tenth-byte overflow")
	}
	reader := byte_reader{Source: overflow_tenth[:]}
	_, err := binary.Read_Unsigned_Varint(&reader, read_byte)
	if err != binary.Error_Varint_Overflow {
		t.Fatal("Read_Unsigned_Varint returned wrong overflow error")
	}
	if reader.Position != 10 {
		t.Fatal("Read_Unsigned_Varint missed bounded overflow")
	}
	overflow_eleventh := [...]byte{
		0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01,
	}
	value, count = binary.Unsigned_Varint(overflow_eleventh[:])
	if value != 0 {
		t.Fatal("Unsigned_Varint returned value for eleventh-byte overflow")
	}
	if count != -11 {
		t.Fatal("Unsigned_Varint missed eleventh-byte overflow")
	}
	reader = byte_reader{Source: overflow_eleventh[:]}
	_, err = binary.Read_Unsigned_Varint(&reader, read_byte)
	if err != binary.Error_Varint_Overflow {
		t.Fatal("Read_Unsigned_Varint returned wrong overflow error")
	}
	if reader.Position != 10 {
		t.Fatal("Read_Unsigned_Varint read beyond format maximum")
	}
}

// Noncanonical zero remains format-compatible.
func verify_varint_noncanonical_zero(t *testing.T) {
	source := [...]byte{0x80, 0x80, 0x80, 0}
	value, count := binary.Unsigned_Varint(source[:])
	if value != 0 {
		t.Fatal("Unsigned_Varint returned nonzero noncanonical zero")
	}
	if count != 4 {
		t.Fatal("Unsigned_Varint rejected noncanonical zero")
	}
}

func panics(action func()) (raised bool) {
	defer func() {
		raised = recover() != nil
	}()
	action()
	return false
}
