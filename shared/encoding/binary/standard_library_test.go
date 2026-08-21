package binary

import (
	"errors"
	"reflect"
	"testing"
	"unsafe"

	"local/james-orcales/shared/sim/nbio"
	"local/james-orcales/shared/testify"
)

// TEST_PREFIX_SIZE exposes preservation before appended bytes.
const TEST_PREFIX_SIZE = 3

// TEST_ARRAY_COUNT covers fixed arrays and boolean mapping.
const TEST_ARRAY_COUNT = 4

// TEST_PAIR_COUNT covers two-element slice traversal.
const TEST_PAIR_COUNT = 2

// TEST_SINGLE_ELEMENT_COUNT exposes one-element structural boundaries.
const TEST_SINGLE_ELEMENT_COUNT = 1

// TEST_EMPTY_ELEMENT_COUNT exposes zero-element structural boundaries.
const TEST_EMPTY_ELEMENT_COUNT = 0

// TEST_STRUCTURE_SIZE is independent sum of fixture field widths.
const TEST_STRUCTURE_SIZE = 39

// TEST_BLANK_SIZE includes visible fields and reserved padding.
const TEST_BLANK_SIZE = 12

// TEST_STORAGE_SIZE leaves room around short-buffer probes.
const TEST_STORAGE_SIZE = 16

// TEST_SHORT_UINT_64_SIZE is one byte below required width.
const TEST_SHORT_UINT_64_SIZE = 7

// TEST_ALLOCATION_STORAGE_SIZE keeps all allocation probes on caller stack.
const TEST_ALLOCATION_STORAGE_SIZE = 128

type nested_structure struct {
	Word uint16
}

type structure struct {
	Int8      int8
	Int16     int16
	Int32     int32
	Int64     int64
	Uint8     uint8
	Uint16    uint16
	Uint32    uint32
	Uint64    uint64
	Array_0   uint8
	Array_1   uint8
	Array_2   uint8
	Array_3   uint8
	Boolean   bool
	Boolean_0 bool
	Boolean_1 bool
	Boolean_2 bool
	Boolean_3 bool
}

type blank_fields struct {
	First uint32
	_     int32
	Last  uint16
	_     uint8
	_     uint8
}

type blank_probe struct {
	First   uint32
	Padding int32
	Last    uint16
	Tail_0  uint8
	Tail_1  uint8
}

type memory_stream struct {
	Storage  Bytes
	Source   []byte
	Position int
	Written  int
	Failure  error
}

func memory_to_stream(state *memory_stream) (stream nbio.Stream) {
	return nbio.Stream{State: unsafe.Pointer(state), Procedure: memory_stream_procedure}
}

func memory_stream_procedure(
	state_pointer unsafe.Pointer, completion *nbio.Completion, mode nbio.Stream_Mode,
	buffer []byte, _ int64, _ nbio.Seek_From, callback nbio.Stream_Callback,
) {
	state := (*memory_stream)(state_pointer)
	count, err := memory_stream_transfer(state, mode, buffer)
	completion.Data = count
	completion.Error = err
	nbio.Stream_Callback_Call(callback, completion)
}

func memory_stream_transfer(
	stream *memory_stream, mode nbio.Stream_Mode, buffer []byte,
) (count int, err error) {
	if mode == nbio.STREAM_MODE_READ {
		return memory_stream_read(stream, buffer)
	}
	if mode == nbio.STREAM_MODE_WRITE {
		return memory_stream_write(stream, buffer)
	}
	return 0, nbio.Stream_Empty
}

// Failure retires before mutation so tests can prove destination atomicity.
func memory_stream_read(stream *memory_stream, destination []byte) (count int, err error) {
	if stream.Failure != nil {
		return 0, stream.Failure
	}
	if stream.Position == len(stream.Source) {
		return 0, nbio.Stream_EOF
	}
	available_count := len(stream.Source) - stream.Position
	count = len(destination)
	if count > available_count {
		count = available_count
	}
	copy(destination, stream.Source[stream.Position:stream.Position+count])
	stream.Position += count
	return count, nil
}

// Failure retires before mutation so tests can prove source ownership.
func memory_stream_write(stream *memory_stream, source []byte) (count int, err error) {
	if stream.Failure != nil {
		return 0, stream.Failure
	}
	copy(stream.Storage[stream.Written:], source)
	stream.Written += len(source)
	return len(source), nil
}

func read_memory(
	stream *memory_stream, scratch Bytes, destination any, order Byte_Order,
) (completion nbio.Completion) {
	var reader Reader
	Reader_Init(&reader, memory_to_stream(stream), scratch)
	Read(&reader, &reader.Completion, destination, order, memory_completion)
	return reader.Completion
}

func write_memory(
	stream *memory_stream, scratch Bytes, source any, order Byte_Order,
) (completion nbio.Completion) {
	var writer Writer
	Writer_Init(&writer, memory_to_stream(stream), scratch)
	Write(&writer, &writer.Completion, source, order, memory_completion)
	return writer.Completion
}

func memory_completion(completion nbio.Completion_Handle) {
	if completion == nil {
		return
	}
}

func structure_value() (value structure) {
	return structure{
		Int8:      0x01,
		Int16:     0x0203,
		Int32:     0x04050607,
		Int64:     0x08090a0b0c0d0e0f,
		Uint8:     0x10,
		Uint16:    0x1112,
		Uint32:    0x13141516,
		Uint64:    0x1718191a1b1c1d1e,
		Array_0:   31,
		Array_1:   32,
		Array_2:   33,
		Array_3:   34,
		Boolean:   true,
		Boolean_0: true,
		Boolean_1: false,
		Boolean_2: true,
		Boolean_3: false,
	}
}

func big_structure_bytes() (value Bytes) {
	return Bytes{
		1,
		2, 3,
		4, 5, 6, 7,
		8, 9, 10, 11, 12, 13, 14, 15,
		16,
		17, 18,
		19, 20, 21, 22,
		23, 24, 25, 26, 27, 28, 29, 30,
		31, 32, 33, 34,
		1,
		1, 0, 1, 0,
	}
}

func little_structure_bytes() (value Bytes) {
	return Bytes{
		1,
		3, 2,
		7, 6, 5, 4,
		15, 14, 13, 12, 11, 10, 9, 8,
		16,
		18, 17,
		22, 21, 20, 19,
		30, 29, 28, 27, 26, 25, 24, 23,
		31, 32, 33, 34,
		1,
		1, 0, 1, 0,
	}
}

// Test_Structured_Encode_Decode ports upstream complete arithmetic structure fixture.
func Test_Structured_Encode_Decode(t *testing.T) {
	value := structure_value()
	cases := [...]struct {
		Order Byte_Order
		Bytes Bytes
	}{
		{Order: BIG_ENDIAN, Bytes: big_structure_bytes()},
		{Order: LITTLE_ENDIAN, Bytes: little_structure_bytes()},
	}
	for _, one := range cases {
		var storage [TEST_STRUCTURE_SIZE]byte
		count, err := Encode(storage[:], value, one.Order)
		if err != nil {
			t.Fatal("Encode returned error")
		}
		if count != Byte_Count(len(storage)) {
			t.Fatal("Encode returned wrong structured size")
		}
		if !reflect.DeepEqual(storage[:], []byte(one.Bytes)) {
			t.Fatal("Encode wrote wrong structured bytes")
		}
		var decoded structure
		count, err = Decode(storage[:], &decoded, one.Order)
		if err != nil {
			t.Fatal("Decode returned error")
		}
		if count != Byte_Count(len(storage)) {
			t.Fatal("Decode returned wrong structured size")
		}
		if decoded != value {
			t.Fatal("Decode returned wrong structured value")
		}
		var append_storage [TEST_STRUCTURE_SIZE + TEST_PREFIX_SIZE]byte
		buffer := append_storage[:TEST_PREFIX_SIZE:len(append_storage)]
		buffer, err = Append(buffer, value, one.Order)
		if err != nil {
			t.Fatal("Append returned error")
		}
		if len(buffer) != len(append_storage) {
			t.Fatal("Append returned wrong structured size")
		}
		destination_index := TEST_PREFIX_SIZE
		for source_index := range len(storage) {
			if buffer[destination_index] != storage[source_index] {
				t.Fatal("Append wrote wrong structured bytes")
			}
			destination_index++
		}
	}
}

// Test_Structured_Size protects scalar, pointer, slice, array, and nested structure grammar.
func Test_Structured_Size(t *testing.T) {
	value := structure_value()
	integers := [TEST_PAIR_COUNT]int32{1, 2}
	nested := [TEST_PAIR_COUNT]nested_structure{{Word: 1}, {Word: 2}}
	cases := [...]struct {
		Value any
		Size  Value_Size
	}{
		{Value: bool(false), Size: 1},
		{Value: int8(0), Size: 1},
		{Value: uint8(0), Size: 1},
		{Value: int16(0), Size: 2},
		{Value: uint16(0), Size: 2},
		{Value: int32(0), Size: 4},
		{Value: uint32(0), Size: 4},
		{Value: int64(0), Size: 8},
		{Value: uint64(0), Size: 8},
		{Value: value, Size: 39},
		{Value: &value, Size: 39},
		{Value: integers, Size: 8},
		{Value: integers[:], Size: 8},
		{Value: nested, Size: 4},
		{Value: nested[:], Size: 4},
		{Value: struct{}{}, Size: 0},
	}
	for _, one := range cases {
		if Size(one.Value) != one.Size {
			t.Fatal("Size returned wrong fixed size")
		}
	}
}

// Test_Structured_Slices protects top-level slice round trip and boolean nonzero decoding.
func Test_Structured_Slices(t *testing.T) {
	source := [...]int32{0x01020304, 0x05060708}
	var storage [UINT_64_SIZE]byte
	count, err := Encode(storage[:], source[:], BIG_ENDIAN)
	if err != nil {
		t.Fatal("Encode returned error")
	}
	if count != Byte_Count(len(storage)) {
		t.Fatal("Encode rejected fixed-size slice")
	}
	var destination [TEST_PAIR_COUNT]int32
	count, err = Decode(storage[:], destination[:], BIG_ENDIAN)
	if err != nil {
		t.Fatal("Decode returned error")
	}
	if count != Byte_Count(len(storage)) {
		t.Fatal("Decode returned wrong fixed-size slice size")
	}
	if destination != source {
		t.Fatal("Decode returned wrong fixed-size slice")
	}
	boolean_bytes := [...]byte{0, 1, 2, 255}
	var booleans [TEST_ARRAY_COUNT]bool
	count, err = Decode(boolean_bytes[:], booleans[:], BIG_ENDIAN)
	want := [TEST_ARRAY_COUNT]bool{false, true, true, true}
	if err != nil {
		t.Fatal("Decode returned error")
	}
	if count != Byte_Count(len(boolean_bytes)) {
		t.Fatal("Decode returned wrong boolean size")
	}
	if booleans != want {
		t.Fatal("Decode used nonstandard boolean mapping")
	}
}

// Test_Structured_Blank_Fields keeps padding independent from dirty destination storage.
func Test_Structured_Blank_Fields(t *testing.T) {
	value := blank_fields{First: 0x01020304, Last: 0x0506}
	var storage [TEST_BLANK_SIZE]byte
	for index := range storage {
		storage[index] = 0xff
	}
	count, err := Encode(storage[:], value, BIG_ENDIAN)
	if err != nil {
		t.Fatal("Encode returned error")
	}
	if count != Byte_Count(len(storage)) {
		t.Fatal("Encode rejected blank fields")
	}
	want := [TEST_BLANK_SIZE]byte{1, 2, 3, 4, 0, 0, 0, 0, 5, 6, 0, 0}
	if storage != want {
		t.Fatal("Encode did not zero blank fields")
	}
	probe_bytes := [TEST_BLANK_SIZE]byte{1, 2, 3, 4, 9, 8, 7, 6, 5, 6, 4, 3}
	var decoded blank_fields
	count, err = Decode(probe_bytes[:], &decoded, BIG_ENDIAN)
	if err != nil {
		t.Fatal("Decode returned error")
	}
	if count != Byte_Count(len(probe_bytes)) {
		t.Fatal("Decode returned wrong blank-field size")
	}
	if decoded != value {
		t.Fatal("Decode did not discard blank fields")
	}
	var probe blank_probe
	count, err = Decode(probe_bytes[:], &probe, BIG_ENDIAN)
	if err != nil {
		t.Fatal("Decode returned error")
	}
	if count != Byte_Count(len(probe_bytes)) {
		t.Fatal("Decode returned wrong probe size")
	}
	if probe.Padding != 0x09080706 {
		t.Fatal("fixture did not expose nonzero padding")
	}
	if probe.Tail_0 != 4 {
		t.Fatal("fixture did not expose nonzero padding")
	}
	if probe.Tail_1 != 3 {
		t.Fatal("fixture did not expose nonzero padding")
	}
}

// Test_Structured_Stream_IO keeps scratch ownership explicit while preserving stream errors.
func Test_Structured_Stream_IO(t *testing.T) {
	value := structure_value()
	encoded := big_structure_bytes()
	var scratch [TEST_STRUCTURE_SIZE]byte
	reader := memory_stream{Source: encoded[:]}
	var decoded structure
	read_completion := read_memory(&reader, scratch[:], &decoded, BIG_ENDIAN)
	testify.No_Error(t, read_completion.Error)
	testify.Equal(t, value, decoded)
	testify.Equal(t, len(encoded), reader.Position)
	writer := memory_stream{Storage: make(Bytes, BYTE_SIZE_MAXIMUM)}
	write_completion := write_memory(&writer, scratch[:], value, BIG_ENDIAN)
	testify.No_Error(t, write_completion.Error)
	testify.Equal(t, len(encoded), writer.Written)
	for index := range len(encoded) {
		if writer.Storage[index] != encoded[index] {
			t.Fatal("Write emitted wrong structured bytes")
		}
	}
}

// Test_Structured_Errors protects invalid grammar, short buffers, and short streams.
func Test_Structured_Errors(t *testing.T) {
	var storage [TEST_STORAGE_SIZE]byte
	verify_invalid_structured_values(t, storage[:])
	var nil_int16 *int16
	if Size(nil_int16) != VALUE_SIZE_INVALID {
		t.Fatal("Size accepted nil pointer")
	}
	value := uint64(1)
	if _, err := Encode(
		storage[:TEST_SHORT_UINT_64_SIZE], value, BIG_ENDIAN,
	); err != Error_Buffer_Too_Small {
		t.Fatal("Encode returned wrong short-buffer error")
	}
	if _, err := Decode(
		storage[:TEST_SHORT_UINT_64_SIZE], &value, BIG_ENDIAN,
	); err != Error_Buffer_Too_Small {
		t.Fatal("Decode returned wrong short-buffer error")
	}
	if _, err := Decode(
		storage[:UINT_64_SIZE], value, BIG_ENDIAN,
	); err != Error_Invalid_Type {
		t.Fatal("Decode accepted non-addressable destination")
	}
	if !raises(func() {
		Append(storage[:1:1], value, BIG_ENDIAN)
	}) {
		t.Fatal("Append accepted insufficient capacity")
	}
	if !raises(func() {
		var scratch [TEST_SHORT_UINT_64_SIZE]byte
		var writer memory_stream
		write_memory(&writer, scratch[:], value, BIG_ENDIAN)
	}) {
		t.Fatal("Write accepted short scratch storage")
	}
	if !raises(func() {
		var scratch [TEST_SHORT_UINT_64_SIZE]byte
		var reader memory_stream
		read_memory(&reader, scratch[:], &value, BIG_ENDIAN)
	}) {
		t.Fatal("Read accepted short scratch storage")
	}
	for size_index := 0; size_index < UINT_64_SIZE; size_index++ {
		reader := memory_stream{Source: storage[:size_index]}
		var scratch [UINT_64_SIZE]byte
		completion := read_memory(&reader, scratch[:], &value, BIG_ENDIAN)
		want := nbio.Stream_Unexpected_EOF
		if size_index == 0 {
			want = nbio.Stream_EOF
		}
		if completion.Error != want {
			t.Fatal("Read returned wrong truncated-stream error")
		}
	}
}

func verify_invalid_structured_values(t *testing.T, storage []byte) {
	invalid_values := [...]any{
		int(0), uint(0), uintptr(0), "text", []int{}, struct{ Text string }{},
	}
	for _, value := range invalid_values {
		if Size(value) != VALUE_SIZE_INVALID {
			t.Fatal("Size accepted unsupported value")
		}
		if _, err := Encode(storage, value, BIG_ENDIAN); err != Error_Invalid_Type {
			t.Fatal("Encode returned wrong invalid-type error")
		}
		result, err := Append(storage[:1], value, BIG_ENDIAN)
		if err != Error_Invalid_Type {
			t.Fatal("Append returned wrong invalid-type error")
		}
		if result != nil {
			t.Fatal("Append retained input after invalid type")
		}
	}
}

// Test_Structured_Bounds keeps hostile type shape outside fixed traversal storage.
func Test_Structured_Bounds(t *testing.T) {
	var maximum_slice_storage [BYTE_SIZE_MAXIMUM]byte
	if Size(maximum_slice_storage[:0]) != 0 {
		t.Fatal("Size rejected empty supported slice")
	}
	if Size(maximum_slice_storage[:1]) != 1 {
		t.Fatal("Size rejected one-element slice")
	}
	if Size(maximum_slice_storage[:]) != VALUE_SIZE_MAXIMUM {
		t.Fatal("Size rejected maximum-element slice")
	}
	var zero_element_storage [TEST_SINGLE_ELEMENT_COUNT]struct{}
	if Size(zero_element_storage[:]) != 0 {
		t.Fatal("Size rejected zero-size slice element")
	}
	var maximum_element_storage [TEST_SINGLE_ELEMENT_COUNT][BYTE_SIZE_MAXIMUM]byte
	if Size(maximum_element_storage[:]) != VALUE_SIZE_MAXIMUM {
		t.Fatal("Size rejected maximum-size slice element")
	}
	var empty_array [TEST_EMPTY_ELEMENT_COUNT]byte
	if Size(empty_array) != 0 {
		t.Fatal("Size rejected empty supported array")
	}
	var too_many [BYTE_SIZE_MAXIMUM + 1]struct{}
	if !raises(func() { Size(too_many) }) {
		t.Fatal("Size accepted excessive element count")
	}
	var too_large [BYTE_SIZE_MAXIMUM]uint16
	if !raises(func() { Size(too_large) }) {
		t.Fatal("Size accepted excessive encoded size")
	}
	nested_type := reflect.TypeOf(uint8(0))
	for range VALUE_NESTING_MAXIMUM {
		nested_type = reflect.ArrayOf(1, nested_type)
	}
	nested := reflect.New(nested_type).Elem().Interface()
	if Size(nested) != 1 {
		t.Fatal("Size rejected maximum nesting")
	}
	nested_type = reflect.ArrayOf(1, nested_type)
	nested = reflect.New(nested_type).Elem().Interface()
	if !raises(func() { Size(nested) }) {
		t.Fatal("Size accepted excessive nesting")
	}
	nested_type = reflect.TypeOf(uint8(0))
	for range VALUE_NESTING_MAXIMUM {
		nested_type = reflect.StructOf([]reflect.StructField{{
			Name: "Value",
			Type: nested_type,
		}})
	}
	nested = reflect.New(nested_type).Elem().Interface()
	if Size(nested) != 1 {
		t.Fatal("Size rejected maximum record nesting")
	}
}

// Test_Stream_Errors keeps dependency failures unchanged at the I/O boundary.
func Test_Stream_Errors(t *testing.T) {
	var scratch [UINT_64_SIZE]byte
	value := uint64(1)
	failure := errors.New("binary test stream failed")
	reader := memory_stream{Failure: failure}
	read_completion := read_memory(&reader, scratch[:], &value, BIG_ENDIAN)
	if read_completion.Error != failure {
		t.Fatal("Read replaced reader failure")
	}
	if value != 1 {
		t.Fatal("Read mutated destination after reader failure")
	}
	writer := memory_stream{Failure: failure}
	write_completion := write_memory(&writer, scratch[:], value, BIG_ENDIAN)
	if write_completion.Error != failure {
		t.Fatal("Write replaced writer failure")
	}
}

func raises(action func()) (raised bool) {
	defer func() {
		raised = recover() != nil
	}()
	action()
	return false
}

type allocation_case struct {
	Name string
	Run  func()
}

type varint_reader struct {
	Source   []byte
	Position int
}

// Explicit Position reset keeps repeated allocation runs independent.
func read_varint_byte(state Read_Byte_State) (value byte, err error) {
	reader := (*varint_reader)(state)
	if reader.Position == len(reader.Source) {
		return 0, nbio.Stream_EOF
	}
	value = reader.Source[reader.Position]
	reader.Position++
	return value, nil
}

type allocation_fixture struct {
	Storage          Bytes
	Scratch          Bytes
	Stream           memory_stream
	Structure        structure
	Decoded          structure
	Words            []uint32
	Decoded_Words    []uint32
	Decoded_Word_Set []uint32
	Varint_Reader    varint_reader
	Reader           Reader
	Writer           Writer
	Bytes            Nonempty_Bytes
	Uint_16_Bytes    Uint_16_Bytes
	Uint_32_Bytes    Uint_32_Bytes
	Uint_64_Bytes    Uint_64_Bytes
	Structured_Bytes Bytes
	Byte_Count       Byte_Count
	Varint_Size      Varint_Size
	Value_Size       Value_Size
	Varint_Count     Varint_Count
	Uint16           Word_16
	Uint32           Word_32
	Uint64           Word_64
	Signed           Integer_64
	Error            error
	Name             Byte_Order_Name
	Go_Name          Byte_Order_Go_Name
}

// Test_Zero_Allocation measures each public data operation separately.
func Test_Zero_Allocation(t *testing.T) {
	fixture := allocation_fixture{
		Structure:     structure_value(),
		Storage:       make(Bytes, TEST_ALLOCATION_STORAGE_SIZE),
		Scratch:       make(Bytes, TEST_ALLOCATION_STORAGE_SIZE),
		Words:         []uint32{1, 2},
		Decoded_Words: make([]uint32, TEST_PAIR_COUNT),
	}
	fixture.Stream.Storage = make(Bytes, BYTE_SIZE_MAXIMUM)
	fixture.Decoded_Word_Set = fixture.Decoded_Words[:]
	encoded := big_structure_bytes()
	copy(fixture.Storage[:], encoded[:])
	fixture.Stream.Source = fixture.Storage[:len(encoded)]
	fixture.Varint_Reader.Source = fixture.Storage[:VARINT_SIZE_64_MAXIMUM]
	fixture.Storage[0] = 1
	stream := memory_to_stream(&fixture.Stream)
	Reader_Init(&fixture.Reader, stream, fixture.Scratch[:])
	Writer_Init(&fixture.Writer, stream, fixture.Scratch[:])
	groups := [...][]allocation_case{
		allocation_fixed_checks(&fixture),
		allocation_varint_checks(&fixture),
		allocation_structured_checks(&fixture),
		allocation_structured_shape_checks(&fixture),
	}
	for _, group := range groups {
		for _, check := range group {
			t.Run(check.Name, func(t *testing.T) {
				testify.Zero_Allocation(t, check.Run, check.Name)
			})
		}
	}
	if fixture.Uint64 == 0 {
		if fixture.Byte_Count == 0 {
			if fixture.Value_Size == 0 {
				t.Fatal("allocation probes produced no observation")
			}
		}
	}
}

func allocation_fixed_checks(
	fixture *allocation_fixture,
) (checks []allocation_case) {
	return []allocation_case{
		{Name: "Byte_Order_String", Run: func() {
			fixture.Name = Byte_Order_String(BIG_ENDIAN)
		}},
		{Name: "Byte_Order_Go_String", Run: func() {
			fixture.Go_Name = Byte_Order_Go_String(BIG_ENDIAN)
		}},
		{Name: "Uint_16", Run: func() {
			fixture.Uint16 = Uint_16(fixture.Storage[:UINT_16_SIZE], BIG_ENDIAN)
		}},
		{Name: "Put_Uint_16", Run: func() {
			Put_Uint_16(fixture.Storage[:UINT_16_SIZE], fixture.Uint16, BIG_ENDIAN)
		}},
		{Name: "Append_Uint_16", Run: func() {
			fixture.Uint_16_Bytes = Append_Uint_16(
				fixture.Storage[:0], 1, BIG_ENDIAN,
			)
		}},
		{Name: "Uint_32", Run: func() {
			fixture.Uint32 = Uint_32(fixture.Storage[:UINT_32_SIZE], BIG_ENDIAN)
		}},
		{Name: "Put_Uint_32", Run: func() {
			Put_Uint_32(fixture.Storage[:UINT_32_SIZE], fixture.Uint32, BIG_ENDIAN)
		}},
		{Name: "Append_Uint_32", Run: func() {
			fixture.Uint_32_Bytes = Append_Uint_32(
				fixture.Storage[:0], 1, BIG_ENDIAN,
			)
		}},
		{Name: "Uint_64", Run: func() {
			fixture.Uint64 = Uint_64(fixture.Storage[:UINT_64_SIZE], BIG_ENDIAN)
		}},
		{Name: "Put_Uint_64", Run: func() {
			Put_Uint_64(fixture.Storage[:UINT_64_SIZE], fixture.Uint64, BIG_ENDIAN)
		}},
		{Name: "Append_Uint_64", Run: func() {
			fixture.Uint_64_Bytes = Append_Uint_64(
				fixture.Storage[:0], 1, BIG_ENDIAN,
			)
		}},
	}
}

func allocation_varint_checks(
	fixture *allocation_fixture,
) (checks []allocation_case) {
	return []allocation_case{
		{Name: "Put_Unsigned_Varint", Run: func() {
			fixture.Varint_Size = Put_Unsigned_Varint(
				fixture.Storage[:], fixture.Uint64,
			)
		}},
		{Name: "Append_Unsigned_Varint", Run: func() {
			fixture.Bytes = Append_Unsigned_Varint(fixture.Storage[:0], fixture.Uint64)
		}},
		{Name: "Unsigned_Varint", Run: func() {
			fixture.Uint64, fixture.Varint_Count = Unsigned_Varint(
				fixture.Storage[:VARINT_SIZE_64_MAXIMUM],
			)
		}},
		{Name: "Put_Varint", Run: func() {
			fixture.Varint_Size = Put_Varint(fixture.Storage[:], fixture.Signed)
		}},
		{Name: "Append_Varint", Run: func() {
			fixture.Bytes = Append_Varint(fixture.Storage[:0], fixture.Signed)
		}},
		{Name: "Varint", Run: func() {
			fixture.Signed, fixture.Varint_Count = Varint(
				fixture.Storage[:VARINT_SIZE_64_MAXIMUM],
			)
		}},
		{Name: "Read_Unsigned_Varint", Run: func() {
			fixture.Varint_Reader.Position = 0
			state := Read_Byte_State(unsafe.Pointer(&fixture.Varint_Reader))
			fixture.Uint64, fixture.Error = Read_Unsigned_Varint(
				state, read_varint_byte,
			)
		}},
		{Name: "Read_Varint", Run: func() {
			fixture.Varint_Reader.Position = 0
			state := Read_Byte_State(unsafe.Pointer(&fixture.Varint_Reader))
			fixture.Signed, fixture.Error = Read_Varint(
				state, read_varint_byte,
			)
		}},
	}
}

func allocation_structured_checks(
	fixture *allocation_fixture,
) (checks []allocation_case) {
	return []allocation_case{
		{Name: "Size", Run: func() {
			fixture.Value_Size = Size(&fixture.Structure)
		}},
		{Name: "Encode", Run: func() {
			fixture.Byte_Count, fixture.Error = Encode(
				fixture.Storage[:], &fixture.Structure, BIG_ENDIAN,
			)
		}},
		{Name: "Decode", Run: func() {
			fixture.Byte_Count, fixture.Error = Decode(
				fixture.Storage[:], &fixture.Decoded, BIG_ENDIAN,
			)
		}},
		{Name: "Append", Run: func() {
			fixture.Structured_Bytes, fixture.Error = Append(
				fixture.Storage[:0], &fixture.Structure, BIG_ENDIAN,
			)
		}},
		{Name: "Read", Run: func() {
			fixture.Stream.Position = 0
			Read(
				&fixture.Reader, &fixture.Reader.Completion, &fixture.Decoded,
				BIG_ENDIAN, memory_completion,
			)
			fixture.Error = fixture.Reader.Completion.Error
		}},
		{Name: "Write", Run: func() {
			fixture.Stream.Written = 0
			Write(
				&fixture.Writer, &fixture.Writer.Completion, &fixture.Structure,
				BIG_ENDIAN, memory_completion,
			)
			fixture.Error = fixture.Writer.Completion.Error
		}},
	}
}

func allocation_structured_shape_checks(
	fixture *allocation_fixture,
) (checks []allocation_case) {
	return []allocation_case{
		{Name: "Size_Value", Run: func() {
			fixture.Value_Size = Size(fixture.Structure)
		}},
		{Name: "Size_Slice", Run: func() {
			fixture.Value_Size = Size(fixture.Words[:])
		}},
		{Name: "Encode_Value", Run: func() {
			fixture.Byte_Count, fixture.Error = Encode(
				fixture.Storage[:], fixture.Structure, BIG_ENDIAN,
			)
		}},
		{Name: "Encode_Slice", Run: func() {
			fixture.Byte_Count, fixture.Error = Encode(
				fixture.Storage[:], fixture.Words[:], BIG_ENDIAN,
			)
		}},
		{Name: "Decode_Slice", Run: func() {
			fixture.Byte_Count, fixture.Error = Decode(
				fixture.Storage[:], fixture.Decoded_Words[:], BIG_ENDIAN,
			)
		}},
		{Name: "Append_Slice", Run: func() {
			fixture.Structured_Bytes, fixture.Error = Append(
				fixture.Storage[:0], fixture.Words[:], BIG_ENDIAN,
			)
		}},
		{Name: "Read_Slice", Run: func() {
			fixture.Stream.Position = 0
			Read(
				&fixture.Reader, &fixture.Reader.Completion,
				&fixture.Decoded_Word_Set, BIG_ENDIAN, memory_completion,
			)
			fixture.Error = fixture.Reader.Completion.Error
		}},
		{Name: "Write_Value", Run: func() {
			fixture.Stream.Written = 0
			Write(
				&fixture.Writer, &fixture.Writer.Completion, fixture.Structure,
				BIG_ENDIAN, memory_completion,
			)
			fixture.Error = fixture.Writer.Completion.Error
		}},
	}
}
