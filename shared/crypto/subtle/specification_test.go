package subtle_test

import (
	standard_subtle "crypto/subtle"
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

// Test_Reference_Behavior binds each defined scalar result to crypto/subtle.
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
	source := storage_pattern(byte(TEST_STORAGE_SIZE))
	preserved := destination
	subtle.Constant_Time_Copy(subtle.DECISION_FALSE, destination[:], source[:])
	testify.Equal(t, preserved, destination)
	subtle.Constant_Time_Copy(subtle.DECISION_TRUE, destination[:], source[:])
	testify.Equal(t, source, destination)

	destination = storage_pattern(bits.WORD_8_MINIMUM)
	want := destination
	left := storage_pattern(byte(TEST_STORAGE_SIZE))
	right := source_pattern(byte(TEST_STORAGE_SIZE + TEST_STORAGE_SIZE))
	for index := range right {
		want[index] = left[index] ^ right[index]
	}
	count := subtle.XOR_Bytes(destination[:], left[:], right[:])
	testify.Equal(t, subtle.Count(len(right)), count)
	testify.Equal(t, want, destination)
}

// Test_Bounds rejects hostile collections before destination mutation.
func Test_Bounds(t *testing.T) {
	destination := source_pattern(bits.WORD_8_MINIMUM)
	source := short_pattern(byte(TEST_SOURCE_SIZE))
	destination_before := destination
	testify.Panics(t, func() {
		subtle.Constant_Time_Copy(subtle.DECISION_TRUE, destination[:], source[:])
	})
	testify.Equal(t, destination_before, destination)

	short := short_pattern(bits.WORD_8_MINIMUM)
	left := source_pattern(byte(TEST_SHORT_SIZE))
	right := source_pattern(byte(TEST_SHORT_SIZE + TEST_SOURCE_SIZE))
	short_before := short
	testify.Panics(t, func() { subtle.XOR_Bytes(short[:], left[:], right[:]) })
	testify.Equal(t, short_before, short)

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
	overlap_before := overlap
	testify.Panics(t, func() {
		subtle.XOR_Bytes(
			overlap[binary.UINT_8_SIZE:], overlap[:TEST_SOURCE_SIZE], right[:],
		)
	})
	testify.Equal(t, overlap_before, overlap)

	left_destination := source_pattern(bits.WORD_8_MINIMUM)
	left_want := left_destination
	for index := range left_want {
		left_want[index] ^= right[index]
	}
	count := subtle.XOR_Bytes(left_destination[:], left_destination[:], right[:])
	testify.Equal(t, subtle.Count(len(left_destination)), count)
	testify.Equal(t, left_want, left_destination)

	right_destination := source_pattern(bits.WORD_8_MINIMUM)
	left := source_pattern(byte(TEST_SOURCE_SIZE))
	right_want := right_destination
	for index := range right_want {
		right_want[index] ^= left[index]
	}
	count = subtle.XOR_Bytes(right_destination[:], left[:], right_destination[:])
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
		Left:  subtle.Source("left"),
		Right: subtle.Source("lest"),
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
			subtle.DECISION_TRUE, fixture.Destination[:len(fixture.Left)], fixture.Left,
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
			fixture.Destination[:], fixture.Left, fixture.Right,
		)
	})
	testify.True(t, fixture.Integer <= subtle.INTEGER_MAXIMUM)
}

const TEST_STORAGE_SIZE = bits.BIT_COUNT_32_MAXIMUM / bits.BIT_COUNT_8_MAXIMUM

const TEST_SOURCE_SIZE = TEST_STORAGE_SIZE - binary.UINT_8_SIZE

const TEST_SHORT_SIZE = TEST_SOURCE_SIZE - binary.UINT_8_SIZE

func storage_pattern(offset byte) (value [TEST_STORAGE_SIZE]byte) {
	for index := range value {
		value[index] = offset + byte(index) + byte(binary.UINT_8_SIZE)
	}
	return value
}

func source_pattern(offset byte) (value [TEST_SOURCE_SIZE]byte) {
	for index := range value {
		value[index] = offset + byte(index) + byte(binary.UINT_8_SIZE)
	}
	return value
}

func short_pattern(offset byte) (value [TEST_SHORT_SIZE]byte) {
	for index := range value {
		value[index] = offset + byte(index) + byte(binary.UINT_8_SIZE)
	}
	return value
}

func test_reference_compare(t *testing.T) {
	for _, test := range [...]struct {
		Left  []byte
		Right []byte
	}{
		{Left: nil, Right: nil},
		{Left: []byte("same"), Right: []byte("same")},
		{Left: []byte("left"), Right: []byte("lest")},
		{Left: []byte("short"), Right: []byte("longer")},
	} {
		want := subtle.Decision(standard_subtle.ConstantTimeCompare(test.Left, test.Right))
		testify.Equal(t, want, subtle.Constant_Time_Compare(test.Left, test.Right))
	}
}

func test_reference_select(t *testing.T) {
	for _, test := range [...]struct {
		Selector subtle.Decision
		Left     subtle.Integer
		Right    subtle.Integer
	}{
		{
			Selector: subtle.DECISION_FALSE,
			Left:     subtle.INTEGER_MINIMUM,
			Right:    subtle.INTEGER_MAXIMUM,
		},
		{
			Selector: subtle.DECISION_TRUE,
			Left:     subtle.INTEGER_MINIMUM,
			Right:    subtle.INTEGER_MAXIMUM,
		},
	} {
		want := subtle.Integer(standard_subtle.ConstantTimeSelect(
			int(test.Selector), int(test.Left), int(test.Right),
		))
		testify.Equal(t, want,
			subtle.Constant_Time_Select(test.Selector, test.Left, test.Right))
	}
}

func test_reference_byte_equal(t *testing.T) {
	for _, test := range [...]struct {
		Left  subtle.Byte
		Right subtle.Byte
	}{
		{Left: subtle.BYTE_MINIMUM, Right: subtle.BYTE_MINIMUM},
		{Left: subtle.BYTE_MINIMUM, Right: subtle.BYTE_MAXIMUM},
		{Left: subtle.BYTE_MAXIMUM, Right: subtle.BYTE_MAXIMUM},
	} {
		want := subtle.Decision(standard_subtle.ConstantTimeByteEq(
			byte(test.Left), byte(test.Right),
		))
		testify.Equal(t, want, subtle.Constant_Time_Byte_Equal(test.Left, test.Right))
	}
}

func test_reference_integer_32_equal(t *testing.T) {
	for _, test := range [...]struct {
		Left  subtle.Integer_32
		Right subtle.Integer_32
	}{
		{Left: subtle.INTEGER_32_MINIMUM, Right: subtle.INTEGER_32_MINIMUM},
		{Left: subtle.INTEGER_32_MINIMUM, Right: subtle.INTEGER_32_MAXIMUM},
		{Left: subtle.INTEGER_32_MAXIMUM, Right: subtle.INTEGER_32_MAXIMUM},
	} {
		want := subtle.Decision(standard_subtle.ConstantTimeEq(
			int32(test.Left), int32(test.Right),
		))
		testify.Equal(t, want,
			subtle.Constant_Time_Integer_32_Equal(test.Left, test.Right))
	}
}

func test_reference_less_or_equal(t *testing.T) {
	for _, test := range [...]struct {
		Left  subtle.Nonnegative_Integer
		Right subtle.Nonnegative_Integer
	}{
		{
			Left:  subtle.NONNEGATIVE_INTEGER_MINIMUM,
			Right: subtle.NONNEGATIVE_INTEGER_MINIMUM,
		},
		{
			Left:  subtle.NONNEGATIVE_INTEGER_MINIMUM,
			Right: subtle.NONNEGATIVE_INTEGER_MAXIMUM,
		},
		{
			Left:  subtle.NONNEGATIVE_INTEGER_MAXIMUM,
			Right: subtle.NONNEGATIVE_INTEGER_MINIMUM,
		},
		{
			Left:  subtle.NONNEGATIVE_INTEGER_MAXIMUM,
			Right: subtle.NONNEGATIVE_INTEGER_MAXIMUM,
		},
	} {
		want := subtle.Decision(standard_subtle.ConstantTimeLessOrEq(
			int(test.Left), int(test.Right),
		))
		testify.Equal(t, want,
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
	Destination [subtle.DESTINATION_SIZE_MAXIMUM]byte
	Left        subtle.Source
	Right       subtle.Source
	Decision    subtle.Decision
	Count       subtle.Count
	Integer     subtle.Integer
}
