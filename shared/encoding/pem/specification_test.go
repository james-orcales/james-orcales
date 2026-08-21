package pem_test

import (
	"testing"

	"local/james-orcales/shared/encoding/pem"
	"local/james-orcales/shared/testify"
)

// Test_Encode preserves canonical PEM framing and explicit header order.
func Test_Encode(t *testing.T) {
	var destination [pem.ENCODED_SIZE_MAXIMUM]byte
	headers := [...]pem.Header{
		{Key: pem.Header_Key("Proc-Type"), Value: pem.Header_Value("4,ENCRYPTED")},
		{Key: pem.Header_Key("Name"), Value: pem.Header_Value("value")},
	}
	block := pem.Block{
		Type: pem.Type("TEST"), Headers: headers[:], Data: pem.Data("hello"),
	}
	required, size_status := pem.Encoded_Size(block)
	testify.Equal_Values(t, pem.STATUS_OK, size_status)
	count, status := pem.Encode_Into(destination[:int(required)], block)
	testify.Equal(t, required, count)
	testify.Equal_Values(t, pem.STATUS_OK, status)
	testify.Equal(t, ENCODED_BLOCK, string(destination[:count]))

	empty := pem.Block{Type: pem.Type("EMPTY")}
	required, size_status = pem.Encoded_Size(empty)
	testify.Equal_Values(t, pem.STATUS_OK, size_status)
	count, status = pem.Encode_Into(destination[:int(required)], empty)
	testify.Equal_Values(t, pem.STATUS_OK, status)
	testify.Equal(t, ENCODED_EMPTY, string(destination[:count]))

	destination[0] = TEST_SENTINEL
	count, status = pem.Encode_Into(destination[:1], block)
	testify.Equal(t, pem.Encoded_Count(0), count)
	testify.Equal_Values(t, pem.STATUS_OUTPUT_TOO_SMALL, status)
	testify.Equal(t, TEST_SENTINEL, destination[0])
	count, status = pem.Encode_Into(destination[:2], empty)
	testify.Equal(t, pem.Encoded_Count(0), count)
	testify.Equal_Values(t, pem.STATUS_OUTPUT_TOO_SMALL, status)
	test_header_field_domains(t, destination[:])
}

// Test_Decode preserves borrowed metadata, decoded bytes, and consumed input.
func Test_Decode(t *testing.T) {
	var decoded [pem.DECODED_SIZE_MAXIMUM]byte
	var headers [pem.HEADER_COUNT_MAXIMUM]pem.Header
	source := []byte("ignored\n" + ENCODED_BLOCK + "tail")
	block, consumed, status := pem.Decode_Into(decoded[:], headers[:2], source)
	testify.Equal_Values(t, pem.STATUS_OK, status)
	testify.Equal(t, pem.Type("TEST"), block.Type)
	testify.Equal(t, pem.Data("hello"), block.Data)
	testify.Equal(t, pem.Consumed_Count(len(source)-len("tail")), consumed)
	testify.Equal(t, 2, len(block.Headers))
	testify.Equal(t, pem.Header_Key("Proc-Type"), block.Headers[0].Key)
	testify.Equal(t, pem.Header_Value("4,ENCRYPTED"), block.Headers[0].Value)
	testify.Equal(t, pem.Header_Key("Name"), block.Headers[1].Key)
	testify.Equal(t, pem.Header_Value("value"), block.Headers[1].Value)

	crlf := []byte("-----BEGIN X-----\r\nY Q\t= =\r\n-----END X-----\r\n")
	block, consumed, status = pem.Decode_Into(decoded[:], headers[:], crlf)
	testify.Equal_Values(t, pem.STATUS_OK, status)
	testify.Equal(t, pem.Data("a"), block.Data)
	testify.Equal(t, pem.Consumed_Count(len(crlf)), consumed)

	two := []byte(ENCODED_TWO)
	block, consumed, status = pem.Decode_Into(decoded[:2], headers[:], two)
	testify.Equal_Values(t, pem.STATUS_OK, status)
	testify.Equal(t, pem.Type("XY"), block.Type)
	testify.Equal(t, pem.Data("ab"), block.Data)
	testify.Equal(t, pem.Consumed_Count(len(two)), consumed)

	one_header := []byte(ENCODED_ONE_HEADER)
	block, _, status = pem.Decode_Into(decoded[:1], headers[:1], one_header)
	testify.Equal_Values(t, pem.STATUS_OK, status)
	testify.Equal(t, 1, len(block.Headers))
	testify.Equal(t, pem.Data("a"), block.Data)

	test_decoded_headers(t, decoded[:], headers[:])

	block, consumed, status = pem.Decode_Into(decoded[:], headers[:], []byte("none"))
	testify.Equal(t, pem.Block{}, block)
	testify.Equal(t, pem.Consumed_Count(0), consumed)
	testify.Equal_Values(t, pem.STATUS_NOT_FOUND, status)

	malformed := []byte("-----BEGIN X-----\nYQ==\n-----END Y-----\n")
	block, consumed, status = pem.Decode_Into(decoded[:], headers[:], malformed)
	testify.Equal(t, pem.Block{}, block)
	testify.Equal(t, pem.Consumed_Count(0), consumed)
	testify.Equal_Values(t, pem.STATUS_INPUT_INVALID, status)
	invalid_begin := []byte("-----BEGIN X----\n")
	_, _, status = pem.Decode_Into(decoded[:], headers[:], invalid_begin)
	testify.Equal_Values(t, pem.STATUS_INPUT_INVALID, status)
	invalid_shape := []byte("-----BEGIN X-----\n?\n-----END X-----\n")
	_, _, status = pem.Decode_Into(decoded[:], headers[:], invalid_shape)
	testify.Equal_Values(t, pem.STATUS_INPUT_INVALID, status)
	invalid_body := []byte("-----BEGIN X-----\n????\n-----END X-----\n")
	_, _, status = pem.Decode_Into(decoded[:], headers[:], invalid_body)
	testify.Equal_Values(t, pem.STATUS_INPUT_INVALID, status)
	test_later_valid_decode(t, decoded[:], headers[:])
	test_decode_input_domains(t, decoded[:], headers[:])
}

// Test_Bounds reaches exact maxima and every bounded refusal.
func Test_Bounds(t *testing.T) {
	test_short_decode_storage(t)
	test_overlap(t)
	test_maximum_type(t)
	test_maximum_data(t)
	test_maximum_decoded_data(t)
	test_maximum_headers(t)
	test_maximum_block_fields(t)
	test_bound_refusals(t)
}

// Test_Allocation measures every public status path under the production assertion tracker.
func Test_Allocation(t *testing.T) {
	test_size_allocation(t)
	test_encode_allocation(t)
	test_decode_allocation(t)
}

const TEST_SENTINEL byte = 0xa5

const ENCODED_BLOCK = `-----BEGIN TEST-----
Proc-Type: 4,ENCRYPTED
Name: value

aGVsbG8=
-----END TEST-----
`

const ENCODED_EMPTY = `-----BEGIN EMPTY-----
-----END EMPTY-----
`

const ENCODED_TWO = `-----BEGIN XY-----
YWI=
-----END XY-----
`

const ENCODED_ONE_HEADER = `-----BEGIN X-----
A: B

YQ==
-----END X-----
`

func test_decoded_headers(t *testing.T, decoded []byte, headers []pem.Header) {
	spaced := []byte("-----BEGIN X-----\nA:\rB\n\nYQ==\n-----END X-----\n")
	block, _, status := pem.Decode_Into(decoded[:1], headers[:1], spaced)
	testify.Equal_Values(t, pem.STATUS_OK, status)
	testify.Equal(t, pem.Header_Key("A"), block.Headers[0].Key)
	testify.Equal(t, pem.Header_Value("B"), block.Headers[0].Value)

	duplicate := []byte(
		"-----BEGIN X-----\nA: B\nA: D\n\nYQ==\n-----END X-----\n",
	)
	block, _, status = pem.Decode_Into(decoded[:1], headers[:2], duplicate)
	testify.Equal_Values(t, pem.STATUS_OK, status)
	testify.Equal(t, 2, len(block.Headers))
	testify.Equal(t, pem.Header_Key("A"), block.Headers[0].Key)
	testify.Equal(t, pem.Header_Value("B"), block.Headers[0].Value)
	testify.Equal(t, pem.Header_Key("A"), block.Headers[1].Key)
	testify.Equal(t, pem.Header_Value("D"), block.Headers[1].Value)
}

func test_later_valid_decode(t *testing.T, decoded []byte, headers []pem.Header) {
	source := []byte(
		"-----BEGIN X-----\nYQ==\n-----END Y-----\n" +
			"-----BEGIN Z-----\nYg==\n-----END Z-----\n",
	)
	block, consumed, status := pem.Decode_Into(decoded, headers, source)
	testify.Equal_Values(t, pem.STATUS_OK, status)
	testify.Equal(t, pem.Type("Z"), block.Type)
	testify.Equal(t, pem.Data("b"), block.Data)
	testify.Equal(t, pem.Consumed_Count(len(source)), consumed)
}

func test_header_field_domains(t *testing.T, destination []byte) {
	headers := [...]pem.Header{
		{Key: pem.Header_Key("A"), Value: pem.Header_Value("B")},
		{Key: pem.Header_Key("AB"), Value: pem.Header_Value("CD")},
	}
	block := pem.Block{Headers: headers[:]}
	required, size_status := pem.Encoded_Size(block)
	testify.Equal_Values(t, pem.STATUS_OK, size_status)
	count, status := pem.Encode_Into(destination[:int(required)], block)
	testify.Equal(t, required, count)
	testify.Equal_Values(t, pem.STATUS_OK, status)
	one := pem.Block{Headers: headers[:1]}
	required, size_status = pem.Encoded_Size(one)
	testify.Equal_Values(t, pem.STATUS_OK, size_status)
	_, status = pem.Encode_Into(destination[:int(required)], one)
	testify.Equal_Values(t, pem.STATUS_OK, status)
}

func test_decode_input_domains(t *testing.T, decoded []byte, headers []pem.Header) {
	block, consumed, status := pem.Decode_Into(decoded[:0], headers[:0], nil)
	testify.Equal(t, pem.Block{}, block)
	testify.Equal(t, pem.Consumed_Count(0), consumed)
	testify.Equal_Values(t, pem.STATUS_NOT_FOUND, status)
	block, consumed, status = pem.Decode_Into(decoded[:1], headers[:], []byte("x"))
	testify.Equal(t, pem.Block{}, block)
	testify.Equal(t, pem.Consumed_Count(0), consumed)
	testify.Equal_Values(t, pem.STATUS_NOT_FOUND, status)
	block, consumed, status = pem.Decode_Into(decoded[:], headers[:2], []byte("xy"))
	testify.Equal(t, pem.Block{}, block)
	testify.Equal(t, pem.Consumed_Count(0), consumed)
	testify.Equal_Values(t, pem.STATUS_NOT_FOUND, status)
}

func test_short_decode_storage(t *testing.T) {
	var decoded [pem.DECODED_SIZE_MAXIMUM]byte
	var headers [pem.HEADER_COUNT_MAXIMUM]pem.Header
	block, consumed, status := pem.Decode_Into(decoded[:4], headers[:], []byte(ENCODED_BLOCK))
	testify.Equal(t, pem.Block{}, block)
	testify.Equal(t, pem.Consumed_Count(0), consumed)
	testify.Equal_Values(t, pem.STATUS_OUTPUT_TOO_SMALL, status)
	block, consumed, status = pem.Decode_Into(decoded[:], headers[:1], []byte(ENCODED_BLOCK))
	testify.Equal(t, pem.Block{}, block)
	testify.Equal(t, pem.Consumed_Count(0), consumed)
	testify.Equal_Values(t, pem.STATUS_HEADERS_TOO_SMALL, status)
}

func test_overlap(t *testing.T) {
	var storage [pem.ENCODED_SIZE_MAXIMUM]byte
	copy(storage[:], ENCODED_EMPTY)
	var headers [pem.HEADER_COUNT_MAXIMUM]pem.Header
	_, _, decode_status := pem.Decode_Into(
		storage[:pem.DECODED_SIZE_MAXIMUM], headers[:], storage[:len(ENCODED_EMPTY)],
	)
	testify.Equal_Values(t, pem.STATUS_STORAGE_INVALID, decode_status)
	block := pem.Block{Type: pem.Type("X"), Data: storage[:1]}
	_, encode_status := pem.Encode_Into(storage[:], block)
	testify.Equal_Values(t, pem.STATUS_STORAGE_INVALID, encode_status)
}

func test_maximum_type(t *testing.T) {
	var type_storage [pem.TYPE_SIZE_MAXIMUM]byte
	var encoded [pem.ENCODED_SIZE_MAXIMUM]byte
	var decoded [pem.DECODED_SIZE_MAXIMUM]byte
	var headers [pem.HEADER_COUNT_MAXIMUM]pem.Header
	for index := range type_storage {
		type_storage[index] = 'A'
	}
	block := pem.Block{Type: type_storage[:]}
	required, size_status := pem.Encoded_Size(block)
	testify.Equal(t, pem.Encoded_Count(len(encoded)), required)
	testify.Equal_Values(t, pem.STATUS_OK, size_status)
	count, encode_status := pem.Encode_Into(encoded[:], block)
	testify.Equal(t, required, count)
	testify.Equal_Values(t, pem.STATUS_OK, encode_status)
	decoded_block, consumed, decode_status := pem.Decode_Into(
		decoded[:], headers[:], encoded[:],
	)
	testify.Equal_Values(t, pem.STATUS_OK, decode_status)
	testify.Equal(t, pem.Type(type_storage[:]), decoded_block.Type)
	testify.Equal(t, pem.Consumed_Count(len(encoded)), consumed)
}

func test_maximum_data(t *testing.T) {
	var source [pem.ENCODE_SOURCE_SIZE_MAXIMUM]byte
	var encoded [pem.ENCODED_SIZE_MAXIMUM]byte
	var decoded [pem.DECODED_SIZE_MAXIMUM]byte
	var headers [pem.HEADER_COUNT_MAXIMUM]pem.Header
	for index := range source {
		source[index] = byte(index)
	}
	block := pem.Block{Data: source[:]}
	required, size_status := pem.Encoded_Size(block)
	testify.Equal_Values(t, pem.STATUS_OK, size_status)
	count, encode_status := pem.Encode_Into(encoded[:int(required)], block)
	testify.Equal(t, required, count)
	testify.Equal_Values(t, pem.STATUS_OK, encode_status)
	decoded_block, _, decode_status := pem.Decode_Into(
		decoded[:], headers[:], encoded[:count],
	)
	testify.Equal_Values(t, pem.STATUS_OK, decode_status)
	testify.Equal(t, pem.Data(source[:]), decoded_block.Data)
}

func test_maximum_headers(t *testing.T) {
	var source_headers [pem.HEADER_COUNT_MAXIMUM]pem.Header
	var decoded_headers [pem.HEADER_COUNT_MAXIMUM]pem.Header
	var encoded [pem.ENCODED_SIZE_MAXIMUM]byte
	var decoded [pem.DECODED_SIZE_MAXIMUM]byte
	for index := range source_headers {
		source_headers[index] = pem.Header{}
	}
	block := pem.Block{Headers: source_headers[:]}
	required, size_status := pem.Encoded_Size(block)
	testify.Equal_Values(t, pem.STATUS_OK, size_status)
	count, encode_status := pem.Encode_Into(encoded[:int(required)], block)
	testify.Equal_Values(t, pem.STATUS_OK, encode_status)
	decoded_block, _, decode_status := pem.Decode_Into(
		decoded[:], decoded_headers[:], encoded[:count],
	)
	testify.Equal_Values(t, pem.STATUS_OK, decode_status)
	testify.Equal(t, len(source_headers), len(decoded_block.Headers))
}

func test_maximum_decoded_data(t *testing.T) {
	var source [pem.ENCODED_SIZE_MAXIMUM]byte
	var decoded [pem.DECODED_SIZE_MAXIMUM]byte
	var headers [pem.HEADER_COUNT_MAXIMUM]pem.Header
	position := copy(source[:], pem.BEGIN_PREFIX)
	position += copy(source[position:], pem.BOUNDARY_SUFFIX)
	source[position] = pem.LINE_FEED
	position++
	for range pem.DECODE_BODY_ENCODED_SIZE_MAXIMUM {
		source[position] = 'A'
		position++
	}
	source[position] = pem.LINE_FEED
	position++
	position += copy(source[position:], pem.END_PREFIX)
	position += copy(source[position:], pem.BOUNDARY_SUFFIX)
	source[position] = pem.LINE_FEED
	position++
	block, _, status := pem.Decode_Into(decoded[:], headers[:], source[:position])
	testify.Equal_Values(t, pem.STATUS_OK, status)
	testify.Equal(t, pem.DECODED_SIZE_MAXIMUM, len(block.Data))
}

func test_maximum_block_fields(t *testing.T) {
	var destination [pem.ENCODED_SIZE_MAXIMUM]byte
	var data [pem.DECODED_SIZE_MAXIMUM]byte
	var key [pem.ENCODED_SIZE_MAXIMUM]byte
	var value [pem.ENCODED_SIZE_MAXIMUM]byte
	data_block := pem.Block{Data: data[:]}
	_, size_status := pem.Encoded_Size(data_block)
	testify.Equal_Values(t, pem.STATUS_BLOCK_TOO_LARGE, size_status)
	_, encode_status := pem.Encode_Into(destination[:], data_block)
	testify.Equal_Values(t, pem.STATUS_BLOCK_TOO_LARGE, encode_status)
	key_block := pem.Block{Headers: []pem.Header{{Key: key[:]}}}
	_, size_status = pem.Encoded_Size(key_block)
	testify.Equal_Values(t, pem.STATUS_BLOCK_TOO_LARGE, size_status)
	value_block := pem.Block{Headers: []pem.Header{{Value: value[:]}}}
	_, size_status = pem.Encoded_Size(value_block)
	testify.Equal_Values(t, pem.STATUS_BLOCK_TOO_LARGE, size_status)
}

func test_bound_refusals(t *testing.T) {
	var encoded [pem.ENCODED_SIZE_MAXIMUM]byte
	var decoded [pem.DECODED_SIZE_MAXIMUM]byte
	var headers [pem.HEADER_COUNT_MAXIMUM]pem.Header
	var encoded_oversized [pem.ENCODED_SIZE_MAXIMUM + 1]byte
	var decoded_oversized [pem.DECODED_SIZE_MAXIMUM + 1]byte
	var type_oversized [pem.TYPE_SIZE_MAXIMUM + 1]byte
	var data_oversized [pem.DECODED_SIZE_MAXIMUM + 1]byte
	var headers_oversized [pem.HEADER_COUNT_MAXIMUM + 1]pem.Header
	testify.Panics(t, func() { pem.Encode_Into(encoded_oversized[:], pem.Block{}) })
	testify.Panics(t, func() {
		pem.Encode_Into(encoded[:], pem.Block{Type: type_oversized[:]})
	})
	testify.Panics(t, func() {
		pem.Encode_Into(encoded[:], pem.Block{Data: data_oversized[:]})
	})
	testify.Panics(t, func() {
		pem.Encode_Into(encoded[:], pem.Block{Headers: headers_oversized[:]})
	})
	testify.Panics(t, func() { pem.Decode_Into(decoded_oversized[:], headers[:], nil) })
	testify.Panics(t, func() { pem.Decode_Into(decoded[:], headers_oversized[:], nil) })
	testify.Panics(t, func() { pem.Decode_Into(decoded[:], headers[:], encoded_oversized[:]) })
}

func test_size_allocation(t *testing.T) {
	valid := pem.Block{Type: pem.Type("XY"), Data: pem.Data("ab")}
	invalid := pem.Block{Type: pem.Type("X\n")}
	too_large := pem.Block{
		Type: make(pem.Type, pem.TYPE_SIZE_MAXIMUM), Data: make(pem.Data, 1),
	}
	var count pem.Encoded_Count
	var status pem.Size_Status
	testify.Zero_Allocation(t, func() { count, status = pem.Encoded_Size(valid) })
	testify.Equal_Values(t, pem.STATUS_OK, status)
	testify.Zero_Allocation(t, func() { count, status = pem.Encoded_Size(invalid) })
	testify.Equal_Values(t, pem.STATUS_BLOCK_INVALID, status)
	testify.Zero_Allocation(t, func() { count, status = pem.Encoded_Size(too_large) })
	testify.Equal_Values(t, pem.STATUS_BLOCK_TOO_LARGE, status)
	testify.True(t, count <= pem.ENCODED_SIZE_MAXIMUM)
}

func test_encode_allocation(t *testing.T) {
	var destination [pem.ENCODED_SIZE_MAXIMUM]byte
	block := pem.Block{Type: pem.Type("XY"), Data: pem.Data("ab")}
	invalid := pem.Block{Type: pem.Type("X:")}
	too_large := pem.Block{
		Type: make(pem.Type, pem.TYPE_SIZE_MAXIMUM), Data: pem.Data("a"),
	}
	var count pem.Encoded_Count
	var status pem.Encode_Status
	testify.Zero_Allocation(t, func() {
		count, status = pem.Encode_Into(destination[:], block)
	})
	testify.Equal_Values(t, pem.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		count, status = pem.Encode_Into(destination[:0], block)
	})
	testify.Equal_Values(t, pem.STATUS_OUTPUT_TOO_SMALL, status)
	testify.Zero_Allocation(t, func() {
		count, status = pem.Encode_Into(destination[:], invalid)
	})
	testify.Equal_Values(t, pem.STATUS_BLOCK_INVALID, status)
	testify.Zero_Allocation(t, func() {
		count, status = pem.Encode_Into(destination[:], too_large)
	})
	testify.Equal_Values(t, pem.STATUS_BLOCK_TOO_LARGE, status)
	testify.Zero_Allocation(t, func() {
		count, status = pem.Encode_Into(destination[:], pem.Block{Data: destination[:1]})
	})
	testify.Equal_Values(t, pem.STATUS_STORAGE_INVALID, status)
	testify.True(t, count <= pem.ENCODED_SIZE_MAXIMUM)
}

func test_decode_allocation(t *testing.T) {
	var decoded [pem.DECODED_SIZE_MAXIMUM]byte
	var headers [pem.HEADER_COUNT_MAXIMUM]pem.Header
	storage := []byte(ENCODED_BLOCK)
	not_found := pem.Encoded("x")
	invalid := pem.Encoded("-----BEGIN X-----\n????\n-----END X-----\n")
	var block pem.Block
	var consumed pem.Consumed_Count
	var status pem.Decode_Status
	testify.Zero_Allocation(t, func() {
		block, consumed, status = pem.Decode_Into(decoded[:], headers[:], storage)
	})
	testify.Equal_Values(t, pem.STATUS_OK, status)
	testify.Zero_Allocation(t, func() {
		block, consumed, status = pem.Decode_Into(decoded[:], headers[:], not_found)
	})
	testify.Equal_Values(t, pem.STATUS_NOT_FOUND, status)
	testify.Zero_Allocation(t, func() {
		block, consumed, status = pem.Decode_Into(decoded[:], headers[:], invalid)
	})
	testify.Equal_Values(t, pem.STATUS_INPUT_INVALID, status)
	testify.Zero_Allocation(t, func() {
		block, consumed, status = pem.Decode_Into(decoded[:0], headers[:], storage)
	})
	testify.Equal_Values(t, pem.STATUS_OUTPUT_TOO_SMALL, status)
	testify.Zero_Allocation(t, func() {
		block, consumed, status = pem.Decode_Into(decoded[:], headers[:0], storage)
	})
	testify.Equal_Values(t, pem.STATUS_HEADERS_TOO_SMALL, status)
	testify.Zero_Allocation(t, func() {
		block, consumed, status = pem.Decode_Into(storage, headers[:], storage)
	})
	testify.Equal_Values(t, pem.STATUS_STORAGE_INVALID, status)
	testify.True(t, len(block.Data) <= pem.DECODED_SIZE_MAXIMUM)
	testify.True(t, consumed <= pem.ENCODED_SIZE_MAXIMUM)
}
