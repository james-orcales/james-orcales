package subtle_test

import (
	"testing"

	"local/james-orcales/shared/crypto/subtle"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// Test_Constant_Time checks every mismatch position without making content select loop length.
func Test_Constant_Time(t *testing.T) {
	var left [bits.BIT_COUNT_8_MAXIMUM]byte
	var right [bits.BIT_COUNT_8_MAXIMUM]byte
	for index := range right {
		right[index] = byte(index)
		left[index] = right[index]
	}
	for index := range right {
		right[index] ^= byte(binary.UINT_8_SIZE)
		testify.Equal(t, subtle.DECISION_FALSE,
			subtle.Constant_Time_Compare(left[:], right[:]))
		right[index] ^= byte(binary.UINT_8_SIZE)
	}
	testify.Equal(t, subtle.DECISION_TRUE,
		subtle.Constant_Time_Compare(left[:], right[:]))
}

// Test_Reference_Behavior binds each defined scalar result to its truth table.
func Test_Reference_Behavior(t *testing.T) {
	test_reference_compare(t)
	test_reference_select(t)
	test_reference_byte_equal(t)
	test_reference_integer_32_equal(t)
	test_reference_less_or_equal(t)
}

// Test_Caller_Owned_Storage protects copy selection and XOR output ownership.
func Test_Caller_Owned_Storage(t *testing.T) {
	destination := storage_pattern(bits.WORD_8_MINIMUM)
	input := storage_pattern(byte(TEST_STORAGE_SIZE))
	preserved := storage_pattern(bits.WORD_8_MINIMUM)
	subtle.Constant_Time_Copy(
		subtle.DECISION_FALSE, subtle.Destination(destination), subtle.Source(input),
	)
	testify.Equal(t, preserved, destination)
	subtle.Constant_Time_Copy(
		subtle.DECISION_TRUE, subtle.Destination(destination), subtle.Source(input),
	)
	testify.Equal(t, input, destination)

	destination = storage_pattern(bits.WORD_8_MINIMUM)
	want := storage_pattern(bits.WORD_8_MINIMUM)
	left := storage_pattern(byte(TEST_STORAGE_SIZE))
	right := source_pattern(byte(TEST_STORAGE_SIZE + TEST_STORAGE_SIZE))
	for index := range right {
		want[index] = left[index] ^ right[index]
	}
	count := subtle.XOR_Bytes(
		subtle.Destination(destination), subtle.Source(left), subtle.Source(right),
	)
	testify.Equal(t, subtle.Count(len(right)), count)
	testify.Equal(t, want, destination)
}

// Test_Bounds rejects hostile collections before destination mutation.
func Test_Bounds(t *testing.T) {
	destination := source_pattern(bits.WORD_8_MINIMUM)
	input := short_pattern(byte(TEST_SOURCE_SIZE))
	destination_before := source_pattern(bits.WORD_8_MINIMUM)
	testify.Panics(t, func() {
		subtle.Constant_Time_Copy(
			subtle.DECISION_TRUE, subtle.Destination(destination), subtle.Source(input),
		)
	})
	testify.Equal(t, destination_before, destination)

	short_destination := short_pattern(bits.WORD_8_MINIMUM)
	left := source_pattern(byte(TEST_SHORT_SIZE))
	right := source_pattern(byte(TEST_SHORT_SIZE + TEST_SOURCE_SIZE))
	short_before := short_pattern(bits.WORD_8_MINIMUM)
	testify.Panics(t, func() {
		subtle.XOR_Bytes(
			subtle.Destination(short_destination),
			subtle.Source(left), subtle.Source(right),
		)
	})
	testify.Equal(t, short_before, short_destination)

	var oversized [subtle.SOURCE_SIZE_MAXIMUM + binary.UINT_8_SIZE]byte
	testify.Panics(t, func() { subtle.Constant_Time_Compare(oversized[:], nil) })
	testify.Panics(t, func() { subtle.Constant_Time_Compare(nil, oversized[:]) })
	testify.Panics(t, func() {
		subtle.Constant_Time_Copy(subtle.DECISION_FALSE, oversized[:], oversized[:])
	})
	testify.Panics(t, func() { subtle.XOR_Bytes(oversized[:], nil, nil) })
	testify.Panics(t, func() { subtle.XOR_Bytes(nil, oversized[:], nil) })
	testify.Panics(t, func() { subtle.XOR_Bytes(nil, nil, oversized[:]) })
}

// Test_Overlap protects exact zero-copy aliases and refuses partial aliases before writes.
func Test_Overlap(t *testing.T) {
	right := source_pattern(byte(TEST_SOURCE_SIZE))
	overlap := storage_pattern(bits.WORD_8_MINIMUM)
	overlap_before := storage_pattern(bits.WORD_8_MINIMUM)
	testify.Panics(t, func() {
		subtle.XOR_Bytes(
			subtle.Destination(overlap[binary.UINT_8_SIZE:]),
			subtle.Source(overlap[:TEST_SOURCE_SIZE]), subtle.Source(right),
		)
	})
	testify.Equal(t, overlap_before, overlap)

	left_destination := source_pattern(bits.WORD_8_MINIMUM)
	left_want := source_pattern(bits.WORD_8_MINIMUM)
	for index := range left_want {
		left_want[index] ^= right[index]
	}
	count := subtle.XOR_Bytes(
		subtle.Destination(left_destination), subtle.Source(left_destination),
		subtle.Source(right),
	)
	testify.Equal(t, subtle.Count(len(left_destination)), count)
	testify.Equal(t, left_want, left_destination)

	right_destination := source_pattern(bits.WORD_8_MINIMUM)
	left := source_pattern(byte(TEST_SOURCE_SIZE))
	right_want := source_pattern(bits.WORD_8_MINIMUM)
	for index := range right_want {
		right_want[index] ^= left[index]
	}
	count = subtle.XOR_Bytes(
		subtle.Destination(right_destination), subtle.Source(left),
		subtle.Source(right_destination),
	)
	testify.Equal(t, subtle.Count(len(right_destination)), count)
	testify.Equal(t, right_want, right_destination)
}

// Test_Invariant_Domains drives each legal collection boundary through public operations.
func Test_Invariant_Domains(t *testing.T) {
	test_collection_invariant_domains()
	test_scalar_invariant_domains()
}

// Test_Allocation measures every exported runtime operation with assertions enabled.
func Test_Allocation(t *testing.T) {
	fixture := allocation_fixture{
		Destination: make(destination_storage, subtle.DESTINATION_SIZE_MAXIMUM),
		Left:        subtle.Source("left"),
		Right:       subtle.Source("lest"),
	}
	testify.Zero_Allocation(t, func() {
		fixture.Decision = subtle.Constant_Time_Compare(fixture.Left, fixture.Right)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Integer = subtle.Constant_Time_Select(
			subtle.DECISION_TRUE, subtle.INTEGER_MINIMUM, subtle.INTEGER_MAXIMUM,
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Decision = subtle.Constant_Time_Byte_Equal(
			subtle.BYTE_MINIMUM,
			subtle.BYTE_MINIMUM+subtle.Byte(binary.UINT_8_SIZE),
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Decision = subtle.Constant_Time_Integer_32_Equal(
			subtle.INTEGER_32_MINIMUM, subtle.INTEGER_32_MAXIMUM,
		)
	})
	testify.Zero_Allocation(t, func() {
		subtle.Constant_Time_Copy(
			subtle.DECISION_TRUE,
			subtle.Destination(fixture.Destination[:len(fixture.Left)]), fixture.Left,
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Decision = subtle.Constant_Time_Less_Or_Equal(
			subtle.NONNEGATIVE_INTEGER_MINIMUM,
			subtle.NONNEGATIVE_INTEGER_MINIMUM+
				subtle.Nonnegative_Integer(binary.UINT_8_SIZE),
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Count = subtle.XOR_Bytes(
			subtle.Destination(fixture.Destination), fixture.Left, fixture.Right,
		)
	})
	testify.True(t, fixture.Integer <= subtle.INTEGER_MAXIMUM)
}

const TEST_STORAGE_SIZE = bits.BIT_COUNT_32_MAXIMUM / bits.BIT_COUNT_8_MAXIMUM

const TEST_SOURCE_SIZE = TEST_STORAGE_SIZE - binary.UINT_8_SIZE

const TEST_SHORT_SIZE = TEST_SOURCE_SIZE - binary.UINT_8_SIZE

type storage []byte

func storage_pattern(offset byte) (value storage) {
	value = make(storage, TEST_STORAGE_SIZE)
	for index := range value {
		value[index] = offset + byte(index) + byte(binary.UINT_8_SIZE)
	}
	return value
}

type source []byte

func source_pattern(offset byte) (value source) {
	value = make(source, TEST_SOURCE_SIZE)
	for index := range value {
		value[index] = offset + byte(index) + byte(binary.UINT_8_SIZE)
	}
	return value
}

type short []byte

func short_pattern(offset byte) (value short) {
	value = make(short, TEST_SHORT_SIZE)
	for index := range value {
		value[index] = offset + byte(index) + byte(binary.UINT_8_SIZE)
	}
	return value
}

func test_reference_compare(t *testing.T) {
	for _, test := range [...]struct {
		Left  []byte
		Right []byte
		Want  subtle.Decision
	}{
		{Left: nil, Right: nil, Want: subtle.DECISION_TRUE},
		{Left: []byte("same"), Right: []byte("same"), Want: subtle.DECISION_TRUE},
		{Left: []byte("left"), Right: []byte("lest"), Want: subtle.DECISION_FALSE},
		{Left: []byte("short"), Right: []byte("longer"), Want: subtle.DECISION_FALSE},
	} {
		testify.Equal(
			t, test.Want, subtle.Constant_Time_Compare(test.Left, test.Right),
		)
	}
}

func test_reference_select(t *testing.T) {
	for _, test := range [...]struct {
		Selector subtle.Decision
		Left     subtle.Integer
		Right    subtle.Integer
		Want     subtle.Integer
	}{
		{
			Selector: subtle.DECISION_FALSE,
			Left:     subtle.INTEGER_MINIMUM,
			Right:    subtle.INTEGER_MAXIMUM,
			Want:     subtle.INTEGER_MAXIMUM,
		},
		{
			Selector: subtle.DECISION_TRUE,
			Left:     subtle.INTEGER_MINIMUM,
			Right:    subtle.INTEGER_MAXIMUM,
			Want:     subtle.INTEGER_MINIMUM,
		},
	} {
		testify.Equal(t, test.Want,
			subtle.Constant_Time_Select(test.Selector, test.Left, test.Right))
	}
}

func test_reference_byte_equal(t *testing.T) {
	for _, test := range [...]struct {
		Left  subtle.Byte
		Right subtle.Byte
		Want  subtle.Decision
	}{
		{
			Left: subtle.BYTE_MINIMUM, Right: subtle.BYTE_MINIMUM,
			Want: subtle.DECISION_TRUE,
		},
		{
			Left: subtle.BYTE_MINIMUM, Right: subtle.BYTE_MAXIMUM,
			Want: subtle.DECISION_FALSE,
		},
		{
			Left: subtle.BYTE_MAXIMUM, Right: subtle.BYTE_MAXIMUM,
			Want: subtle.DECISION_TRUE,
		},
	} {
		testify.Equal(
			t, test.Want, subtle.Constant_Time_Byte_Equal(test.Left, test.Right),
		)
	}
}

func test_reference_integer_32_equal(t *testing.T) {
	for _, test := range [...]struct {
		Left  subtle.Integer_32
		Right subtle.Integer_32
		Want  subtle.Decision
	}{
		{
			Left: subtle.INTEGER_32_MINIMUM, Right: subtle.INTEGER_32_MINIMUM,
			Want: subtle.DECISION_TRUE,
		},
		{
			Left: subtle.INTEGER_32_MINIMUM, Right: subtle.INTEGER_32_MAXIMUM,
			Want: subtle.DECISION_FALSE,
		},
		{
			Left: subtle.INTEGER_32_MAXIMUM, Right: subtle.INTEGER_32_MAXIMUM,
			Want: subtle.DECISION_TRUE,
		},
	} {
		testify.Equal(t, test.Want,
			subtle.Constant_Time_Integer_32_Equal(test.Left, test.Right))
	}
}

func test_reference_less_or_equal(t *testing.T) {
	for _, test := range [...]struct {
		Left  subtle.Nonnegative_Integer
		Right subtle.Nonnegative_Integer
		Want  subtle.Decision
	}{
		{
			Left:  subtle.NONNEGATIVE_INTEGER_MINIMUM,
			Right: subtle.NONNEGATIVE_INTEGER_MINIMUM,
			Want:  subtle.DECISION_TRUE,
		},
		{
			Left:  subtle.NONNEGATIVE_INTEGER_MINIMUM,
			Right: subtle.NONNEGATIVE_INTEGER_MAXIMUM,
			Want:  subtle.DECISION_TRUE,
		},
		{
			Left:  subtle.NONNEGATIVE_INTEGER_MAXIMUM,
			Right: subtle.NONNEGATIVE_INTEGER_MINIMUM,
			Want:  subtle.DECISION_FALSE,
		},
		{
			Left:  subtle.NONNEGATIVE_INTEGER_MAXIMUM,
			Right: subtle.NONNEGATIVE_INTEGER_MAXIMUM,
			Want:  subtle.DECISION_TRUE,
		},
	} {
		testify.Equal(t, test.Want,
			subtle.Constant_Time_Less_Or_Equal(test.Left, test.Right))
	}
}

func test_collection_invariant_domains() {
	var left [subtle.SOURCE_SIZE_MAXIMUM]byte
	var right [subtle.SOURCE_SIZE_MAXIMUM]byte
	var destination [subtle.DESTINATION_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		subtle.SOURCE_SIZE_MINIMUM,
		subtle.SOURCE_SIZE_MINIMUM + binary.UINT_8_SIZE,
		subtle.SOURCE_SIZE_MINIMUM + binary.UINT_16_SIZE,
		subtle.SOURCE_SIZE_MAXIMUM,
	} {
		subtle.Constant_Time_Compare(left[:size], right[:size])
		subtle.Constant_Time_Copy(
			subtle.DECISION_FALSE, destination[:size], left[:size],
		)
		subtle.Constant_Time_Copy(
			subtle.DECISION_TRUE, destination[:size], left[:size],
		)
		subtle.XOR_Bytes(destination[:size], left[:size], right[:size])
	}
	subtle.Constant_Time_Compare(
		left[:binary.UINT_8_SIZE], right[:binary.UINT_16_SIZE],
	)
	subtle.XOR_Bytes(
		destination[:], left[:binary.UINT_8_SIZE], right[:binary.UINT_16_SIZE],
	)
	subtle.XOR_Bytes(
		destination[:], left[:binary.UINT_16_SIZE], right[:binary.UINT_8_SIZE],
	)
}

func test_scalar_invariant_domains() {
	for _, value := range [...]subtle.Integer{
		subtle.INTEGER_MINIMUM,
		-subtle.Integer(binary.UINT_8_SIZE),
		subtle.Integer(bits.WORD_MINIMUM),
		subtle.Integer(binary.UINT_8_SIZE),
		subtle.Integer(binary.UINT_16_SIZE),
		subtle.INTEGER_MAXIMUM,
	} {
		subtle.Constant_Time_Select(subtle.DECISION_FALSE, value, value)
		subtle.Constant_Time_Select(subtle.DECISION_TRUE, value, value)
	}
	for _, value := range [...]subtle.Byte{
		subtle.BYTE_MINIMUM,
		subtle.BYTE_MINIMUM + subtle.Byte(binary.UINT_8_SIZE),
		subtle.BYTE_MINIMUM + subtle.Byte(binary.UINT_16_SIZE),
		subtle.BYTE_MAXIMUM,
	} {
		subtle.Constant_Time_Byte_Equal(value, value)
		subtle.Constant_Time_Byte_Equal(value, value^subtle.Byte(binary.UINT_8_SIZE))
	}
	for _, value := range [...]subtle.Integer_32{
		subtle.INTEGER_32_MINIMUM,
		-subtle.Integer_32(binary.UINT_8_SIZE),
		subtle.Integer_32(bits.WORD_32_MINIMUM),
		subtle.Integer_32(binary.UINT_8_SIZE),
		subtle.Integer_32(binary.UINT_16_SIZE),
		subtle.INTEGER_32_MAXIMUM,
	} {
		subtle.Constant_Time_Integer_32_Equal(value, value)
		subtle.Constant_Time_Integer_32_Equal(
			value, value^subtle.Integer_32(binary.UINT_8_SIZE),
		)
	}
	for _, value := range [...]subtle.Nonnegative_Integer{
		subtle.NONNEGATIVE_INTEGER_MINIMUM,
		subtle.NONNEGATIVE_INTEGER_MINIMUM +
			subtle.Nonnegative_Integer(binary.UINT_8_SIZE),
		subtle.NONNEGATIVE_INTEGER_MINIMUM +
			subtle.Nonnegative_Integer(binary.UINT_16_SIZE),
		subtle.NONNEGATIVE_INTEGER_MAXIMUM,
	} {
		subtle.Constant_Time_Less_Or_Equal(value, value)
		subtle.Constant_Time_Less_Or_Equal(subtle.NONNEGATIVE_INTEGER_MINIMUM, value)
		subtle.Constant_Time_Less_Or_Equal(value, subtle.NONNEGATIVE_INTEGER_MINIMUM)
	}
}

type allocation_fixture struct {
	Destination destination_storage
	Left        subtle.Source
	Right       subtle.Source
	Decision    subtle.Decision
	Count       subtle.Count
	Integer     subtle.Integer
}

type destination_storage []byte
