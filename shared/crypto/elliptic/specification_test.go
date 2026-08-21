package elliptic_test

import (
	standard_elliptic "crypto/elliptic"
	"testing"

	"local/james-orcales/shared/crypto/elliptic"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// Test_Reference_Points binds the opaque generator to standard P-256.
func Test_Reference_Points(t *testing.T) {
	point := elliptic.Point_Generator()
	var encoding [elliptic.ENCODING_UNCOMPRESSED_SIZE]byte
	count, status := elliptic.Point_Bytes_Into(
		encoding[:], &point, elliptic.ENCODING_UNCOMPRESSED,
	)
	parameters := standard_elliptic.P256().Params()
	want := standard_elliptic.Marshal(
		standard_elliptic.P256(), parameters.Gx, parameters.Gy,
	)
	testify.Equal(t, elliptic.Count(len(encoding)), count)
	testify.Equal(t, elliptic.OUTPUT_STATUS_OK, status)
	testify.Equal(t, want, encoding[:])
}

// Test_Encoding covers SEC 1 infinity, compressed, and uncompressed forms.
func Test_Encoding(t *testing.T) {
	standard := standard_elliptic.P256()
	parameters := standard.Params()
	uncompressed := standard_elliptic.Marshal(standard, parameters.Gx, parameters.Gy)
	compressed := standard_elliptic.MarshalCompressed(standard, parameters.Gx, parameters.Gy)

	var point elliptic.Point
	status := elliptic.Point_Set_Bytes(&point, uncompressed)
	testify.Equal(t, elliptic.PARSE_STATUS_OK, status)
	verify_encoding(t, &point, elliptic.ENCODING_UNCOMPRESSED, uncompressed)
	verify_encoding(t, &point, elliptic.ENCODING_COMPRESSED, compressed)

	status = elliptic.Point_Set_Bytes(&point, compressed)
	testify.Equal(t, elliptic.PARSE_STATUS_OK, status)
	verify_encoding(t, &point, elliptic.ENCODING_UNCOMPRESSED, uncompressed)

	identity := elliptic.Point_Identity()
	var infinity [elliptic.ENCODING_INFINITY_SIZE]byte
	count, encode_status := elliptic.Point_Bytes_Into(
		infinity[:], &identity, elliptic.ENCODING_COMPRESSED,
	)
	testify.Equal(t, elliptic.Count(len(infinity)), count)
	testify.Equal(t, elliptic.OUTPUT_STATUS_OK, encode_status)
	testify.Equal(
		t, byte(elliptic.POINT_INFINITY_PREFIX), infinity[bits.BIT_COUNT_MINIMUM],
	)

	status = elliptic.Point_Set_Bytes(&point, infinity[:])
	testify.Equal(t, elliptic.PARSE_STATUS_OK, status)
	testify.True(t, bool(elliptic.Point_Equal(&point, &identity)))
}

// Test_Group_Operations compares complete addition and doubling with standard P-256.
func Test_Group_Operations(t *testing.T) {
	point_2 := scalar_base_point(t, scalar_with_last_byte(byte(binary.UINT_16_SIZE)))
	point_3 := scalar_base_point(
		t, scalar_with_last_byte(byte(binary.UINT_16_SIZE+binary.UINT_8_SIZE)),
	)
	var sum elliptic.Point
	elliptic.Point_Add(&sum, &point_2, &point_3)
	verify_standard_scalar(
		t, &sum, scalar_with_last_byte(byte(binary.UINT_32_SIZE+binary.UINT_8_SIZE)),
	)

	var doubled elliptic.Point
	elliptic.Point_Double(&doubled, &point_2)
	verify_standard_scalar(t, &doubled, scalar_with_last_byte(byte(binary.UINT_32_SIZE)))

	identity := elliptic.Point_Identity()
	elliptic.Point_Add(&sum, &point_2, &identity)
	testify.True(t, bool(elliptic.Point_Equal(&sum, &point_2)))
}

// Test_Scalar_Multiplication compares base and arbitrary point multiplication.
func Test_Scalar_Multiplication(t *testing.T) {
	for _, scalar := range [...][elliptic.SCALAR_SIZE]byte{
		scalar_with_last_byte(byte(binary.UINT_8_SIZE)),
		scalar_with_last_byte(byte(binary.UINT_16_SIZE)),
		scalar_index_pattern(),
	} {
		point := scalar_base_point(t, scalar)
		verify_standard_scalar(t, &point, scalar)

		generator := elliptic.Point_Generator()
		var multiplied elliptic.Point
		status := elliptic.Point_Scalar_Multiply(&multiplied, &generator, scalar[:])
		testify.Equal(t, elliptic.SCALAR_STATUS_OK, status)
		testify.True(t, bool(elliptic.Point_Equal(&point, &multiplied)))
	}
}

// Test_Bounds rejects hostile sizes and preserves destinations on refusal.
func Test_Bounds(t *testing.T) {
	var oversized_encoding [elliptic.ENCODING_UNVALIDATED_SIZE_MAXIMUM +
		binary.UINT_8_SIZE]byte
	var point elliptic.Point
	testify.Panics(t, func() {
		elliptic.Point_Set_Bytes(&point, oversized_encoding[:])
	})
	var oversized_destination [elliptic.DESTINATION_SIZE_MAXIMUM +
		binary.UINT_8_SIZE]byte
	testify.Panics(t, func() {
		elliptic.Point_Bytes_Into(
			oversized_destination[:], &point, elliptic.ENCODING_COMPRESSED,
		)
	})
	var oversized_scalar [elliptic.SCALAR_UNVALIDATED_SIZE_MAXIMUM + binary.UINT_8_SIZE]byte
	testify.Panics(t, func() {
		elliptic.Point_Scalar_Base_Multiply(&point, oversized_scalar[:])
	})

	point = elliptic.Point_Generator()
	before := point
	status := elliptic.Point_Set_Bytes(
		&point,
		elliptic.Encoding_Unvalidated{
			byte(elliptic.POINT_INFINITY_PREFIX + binary.UINT_8_SIZE),
		},
	)
	testify.Equal(t, elliptic.PARSE_STATUS_INPUT_INVALID, status)
	testify.True(t, bool(elliptic.Point_Equal(&point, &before)))

	var short [elliptic.ENCODING_COMPRESSED_SIZE - binary.UINT_8_SIZE]byte
	short[bits.BIT_COUNT_MINIMUM] = bits.WORD_8_MAXIMUM
	count, encode_status := elliptic.Point_Bytes_Into(
		short[:], &point, elliptic.ENCODING_COMPRESSED,
	)
	testify.Equal(t, elliptic.Count(elliptic.ENCODING_COMPRESSED_SIZE), count)
	testify.Equal(t, elliptic.OUTPUT_STATUS_DESTINATION_TOO_SMALL, encode_status)
	testify.Equal(t, byte(bits.WORD_8_MAXIMUM), short[bits.BIT_COUNT_MINIMUM])

	scalar_status := elliptic.Point_Scalar_Base_Multiply(&point, nil)
	testify.Equal(t, elliptic.SCALAR_STATUS_INPUT_INVALID, scalar_status)
	testify.True(t, bool(elliptic.Point_Equal(&point, &before)))
}

// Test_Invariant_Domains reaches every owned collection, enum, count, and decision value.
func Test_Invariant_Domains(t *testing.T) {
	test_encoding_domains()
	test_destination_domains()
	test_scalar_domains()
	testify.False(t, bool(elliptic.Point_Equal(
		point_pointer(elliptic.Point_Identity()),
		point_pointer(elliptic.Point_Generator()),
	)))
}

// Test_Allocation measures every exported runtime operation.
func Test_Allocation(t *testing.T) {
	scalar := scalar_with_last_byte(byte(binary.UINT_16_SIZE))
	generator := elliptic.Point_Generator()
	identity := elliptic.Point_Identity()
	generator_encoding := generator_uncompressed()
	var point, result elliptic.Point
	var output [elliptic.ENCODING_UNCOMPRESSED_SIZE]byte
	var count elliptic.Count
	var parse_status elliptic.Parse_Status
	var output_status elliptic.Output_Status
	var scalar_status elliptic.Scalar_Status
	var equal elliptic.Equality
	testify.Zero_Allocation(t, func() { generator = elliptic.Point_Generator() })
	testify.Zero_Allocation(t, func() { identity = elliptic.Point_Identity() })
	testify.Zero_Allocation(t, func() {
		parse_status = elliptic.Point_Set_Bytes(&point, generator_encoding[:])
	})
	testify.Zero_Allocation(t, func() {
		count, output_status = elliptic.Point_Bytes_Into(
			output[:], &point, elliptic.ENCODING_UNCOMPRESSED,
		)
	})
	testify.Zero_Allocation(t, func() {
		elliptic.Point_Add(&result, &generator, &identity)
	})
	testify.Zero_Allocation(t, func() { elliptic.Point_Double(&result, &generator) })
	testify.Zero_Allocation(t, func() {
		scalar_status = elliptic.Point_Scalar_Multiply(&result, &generator, scalar[:])
	})
	testify.Zero_Allocation(t, func() {
		scalar_status = elliptic.Point_Scalar_Base_Multiply(&result, scalar[:])
	})
	testify.Zero_Allocation(t, func() {
		equal = elliptic.Point_Equal(&result, &generator)
	})
	testify.Equal(t, elliptic.Count(elliptic.ENCODING_UNCOMPRESSED_SIZE), count)
	testify.Equal(t, elliptic.PARSE_STATUS_OK, parse_status)
	testify.Equal(t, elliptic.OUTPUT_STATUS_OK, output_status)
	testify.Equal(t, elliptic.SCALAR_STATUS_OK, scalar_status)
	testify.False(t, bool(equal))
}

func generator_uncompressed() (encoding [elliptic.ENCODING_UNCOMPRESSED_SIZE]byte) {
	return [elliptic.ENCODING_UNCOMPRESSED_SIZE]byte{
		0x04,
		0x6b, 0x17, 0xd1, 0xf2, 0xe1, 0x2c, 0x42, 0x47,
		0xf8, 0xbc, 0xe6, 0xe5, 0x63, 0xa4, 0x40, 0xf2,
		0x77, 0x03, 0x7d, 0x81, 0x2d, 0xeb, 0x33, 0xa0,
		0xf4, 0xa1, 0x39, 0x45, 0xd8, 0x98, 0xc2, 0x96,
		0x4f, 0xe3, 0x42, 0xe2, 0xfe, 0x1a, 0x7f, 0x9b,
		0x8e, 0xe7, 0xeb, 0x4a, 0x7c, 0x0f, 0x9e, 0x16,
		0x2b, 0xce, 0x33, 0x57, 0x6b, 0x31, 0x5e, 0xce,
		0xcb, 0xb6, 0x40, 0x68, 0x37, 0xbf, 0x51, 0xf5,
	}
}

func verify_encoding(
	t *testing.T,
	point *elliptic.Point,
	kind elliptic.Encoding_Kind,
	want []byte,
) {
	t.Helper()
	var output [elliptic.ENCODING_UNCOMPRESSED_SIZE]byte
	count, status := elliptic.Point_Bytes_Into(output[:], point, kind)
	testify.Equal(t, elliptic.Count(len(want)), count)
	testify.Equal(t, elliptic.OUTPUT_STATUS_OK, status)
	testify.Equal(t, want, output[:count])
}

func scalar_base_point(
	t *testing.T, scalar [elliptic.SCALAR_SIZE]byte,
) (point elliptic.Point) {
	t.Helper()
	status := elliptic.Point_Scalar_Base_Multiply(&point, scalar[:])
	testify.Equal(t, elliptic.SCALAR_STATUS_OK, status)
	return point
}

func verify_standard_scalar(
	t *testing.T,
	point *elliptic.Point,
	scalar [elliptic.SCALAR_SIZE]byte,
) {
	t.Helper()
	x, y := standard_elliptic.P256().ScalarBaseMult(scalar[:])
	want := standard_elliptic.Marshal(standard_elliptic.P256(), x, y)
	verify_encoding(t, point, elliptic.ENCODING_UNCOMPRESSED, want)
}

func scalar_with_last_byte(value byte) (scalar [elliptic.SCALAR_SIZE]byte) {
	scalar[len(scalar)-binary.UINT_8_SIZE] = value
	return scalar
}

func scalar_index_pattern() (scalar [elliptic.SCALAR_SIZE]byte) {
	for index := range scalar {
		scalar[index] = byte(index)
	}
	return scalar
}

func test_encoding_domains() {
	var point elliptic.Point
	var source [elliptic.ENCODING_UNVALIDATED_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		elliptic.ENCODING_UNVALIDATED_SIZE_MINIMUM,
		elliptic.ENCODING_UNVALIDATED_SIZE_MINIMUM + binary.UINT_8_SIZE,
		elliptic.ENCODING_UNVALIDATED_SIZE_MINIMUM + binary.UINT_16_SIZE,
		elliptic.ENCODING_UNVALIDATED_SIZE_MAXIMUM - binary.UINT_8_SIZE,
		elliptic.ENCODING_UNVALIDATED_SIZE_MAXIMUM,
	} {
		elliptic.Point_Set_Bytes(&point, source[:size])
	}
}

func test_destination_domains() {
	generator := elliptic.Point_Generator()
	identity := elliptic.Point_Identity()
	var output [elliptic.DESTINATION_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		elliptic.DESTINATION_SIZE_MINIMUM,
		elliptic.DESTINATION_SIZE_MINIMUM + binary.UINT_8_SIZE,
		elliptic.DESTINATION_SIZE_MINIMUM + binary.UINT_16_SIZE,
		elliptic.DESTINATION_SIZE_MAXIMUM - binary.UINT_8_SIZE,
		elliptic.DESTINATION_SIZE_MAXIMUM,
	} {
		elliptic.Point_Bytes_Into(
			output[:size], &generator, elliptic.ENCODING_UNCOMPRESSED,
		)
	}
	elliptic.Point_Bytes_Into(output[:], &generator, elliptic.ENCODING_COMPRESSED)
	elliptic.Point_Bytes_Into(output[:], &identity, elliptic.ENCODING_COMPRESSED)
}

func test_scalar_domains() {
	var point elliptic.Point
	var scalar [elliptic.SCALAR_UNVALIDATED_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		elliptic.SCALAR_UNVALIDATED_SIZE_MINIMUM,
		elliptic.SCALAR_UNVALIDATED_SIZE_MINIMUM + binary.UINT_8_SIZE,
		elliptic.SCALAR_UNVALIDATED_SIZE_MINIMUM + binary.UINT_16_SIZE,
		elliptic.SCALAR_SIZE,
		elliptic.SCALAR_UNVALIDATED_SIZE_MAXIMUM,
	} {
		elliptic.Point_Scalar_Base_Multiply(&point, scalar[:size])
		generator := elliptic.Point_Generator()
		elliptic.Point_Scalar_Multiply(&point, &generator, scalar[:size])
	}
}

func point_pointer(point elliptic.Point) (pointer *elliptic.Point) {
	return &point
}
