package pbkdf2_test

import (
	"testing"

	"local/james-orcales/shared/crypto/hmac"
	"local/james-orcales/shared/crypto/pbkdf2"
	"local/james-orcales/shared/crypto/sha1"
	"local/james-orcales/shared/crypto/sha512"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// Test_Reference_Value binds derivation to RFC 6070 test vector three.
func Test_Reference_Value(t *testing.T) {
	rfc_want := [...]byte{
		0x4b, 0x00, 0x79, 0x01, 0xb7, 0x65, 0x48, 0x9a,
		0xbe, 0xad, 0x49, 0xd9, 0x26, 0xf7, 0x21, 0xd0,
		0x65, 0xa4, 0x29, 0xc1,
	}
	var rfc_output [len(rfc_want)]byte
	rfc_count, rfc_status := pbkdf2.Key_Into(
		rfc_output[:], hmac.KIND_SHA_1,
		pbkdf2.Password("password"), pbkdf2.Salt("salt"),
		REFERENCE_ITERATION_COUNT,
	)
	testify.Equal(t, pbkdf2.Count(len(rfc_output)), rfc_count)
	testify.Equal(t, pbkdf2.STATUS_OK, rfc_status)
	testify.Equal(t, rfc_want, rfc_output)

	password := pbkdf2.Password("password")
	salt := pbkdf2.Salt("salt")
	for _, kind := range test_kinds() {
		digest_size := test_digest_size(kind)
		var output [TEST_OUTPUT_SIZE]byte
		count, status := pbkdf2.Key_Into(
			output[:digest_size], kind, password, salt,
			pbkdf2.ITERATION_COUNT_MINIMUM,
		)
		testify.Equal(t, pbkdf2.Count(digest_size), count)
		testify.Equal(t, pbkdf2.STATUS_OK, status)
		var block [len("salt") + binary.UINT_32_SIZE]byte
		copy(block[:], salt)
		block[len(block)-binary.UINT_8_SIZE] = byte(binary.UINT_8_SIZE)
		var digest hmac.Digest
		hmac.Digest_Init(&digest, kind, hmac.Key(password))
		hmac.Digest_Write(&digest, block[:])
		var want [TEST_OUTPUT_SIZE]byte
		want_count, want_status := hmac.Digest_Sum_Into(
			&digest, want[:digest_size],
		)
		testify.Equal(t, hmac.Output_Count(digest_size), want_count)
		testify.Equal(t, hmac.OUTPUT_STATUS_OK, want_status)
		testify.Equal(t, want[:digest_size], output[:digest_size])
	}
}

// Test_Caller_Owned_Output accepts empty output without changing caller storage.
func Test_Caller_Owned_Output(t *testing.T) {
	count, status := pbkdf2.Key_Into(
		nil, hmac.KIND_SHA_512, nil, nil, pbkdf2.ITERATION_COUNT_MAXIMUM,
	)
	testify.Equal(t, pbkdf2.COUNT_EMPTY, count)
	testify.Equal(t, pbkdf2.STATUS_OK, status)
}

// Test_Work_Bound rejects hostile total work before changing caller storage.
func Test_Work_Bound(t *testing.T) {
	if pbkdf2.PRF_EVALUATION_COUNT_MAXIMUM != bits.MEBIBYTE_BYTES {
		t.Fatalf(
			"PRF_EVALUATION_COUNT_MAXIMUM = %d; want %d",
			pbkdf2.PRF_EVALUATION_COUNT_MAXIMUM, bits.MEBIBYTE_BYTES,
		)
	}
	var output [sha1.DIGEST_SIZE + binary.UINT_8_SIZE]byte
	output[bits.BIT_COUNT_MINIMUM] = TEST_SENTINEL
	count, status := pbkdf2.Key_Into(
		output[:], hmac.KIND_SHA_1, nil, nil, pbkdf2.ITERATION_COUNT_MAXIMUM,
	)
	testify.Equal(t, pbkdf2.COUNT_EMPTY, count)
	testify.Equal(t, pbkdf2.STATUS_WORK_TOO_LARGE, status)
	testify.Equal(t, TEST_SENTINEL, output[bits.BIT_COUNT_MINIMUM])
}

// Test_Bounds rejects values outside each public domain.
func Test_Bounds(t *testing.T) {
	var oversized_input [pbkdf2.INPUT_SIZE_MAXIMUM + binary.UINT_8_SIZE]byte
	var oversized_output [pbkdf2.OUTPUT_SIZE_MAXIMUM + binary.UINT_8_SIZE]byte
	testify.Panics(t, func() {
		pbkdf2.Key_Into(
			nil, hmac.KIND_SHA_256, oversized_input[:], nil,
			pbkdf2.ITERATION_COUNT_MINIMUM,
		)
	})
	testify.Panics(t, func() {
		pbkdf2.Key_Into(
			nil, hmac.KIND_SHA_256, nil, oversized_input[:],
			pbkdf2.ITERATION_COUNT_MINIMUM,
		)
	})
	testify.Panics(t, func() {
		pbkdf2.Key_Into(
			oversized_output[:], hmac.KIND_SHA_256, nil, nil,
			pbkdf2.ITERATION_COUNT_MINIMUM,
		)
	})
	testify.Panics(t, func() {
		pbkdf2.Key_Into(
			nil, hmac.Kind(hmac.KIND_COUNT), nil, nil,
			pbkdf2.ITERATION_COUNT_MINIMUM,
		)
	})
	testify.Panics(t, func() {
		pbkdf2.Key_Into(
			nil, hmac.KIND_SHA_256, nil, nil,
			pbkdf2.ITERATION_COUNT_MINIMUM-binary.UINT_8_SIZE,
		)
	})
	testify.Panics(t, func() {
		pbkdf2.Key_Into(
			nil, hmac.KIND_SHA_256, nil, nil,
			pbkdf2.ITERATION_COUNT_MAXIMUM+binary.UINT_8_SIZE,
		)
	})
}

// Test_Invariant_Domains reaches each owned scalar and collection boundary.
func Test_Invariant_Domains(t *testing.T) {
	test_input_domains()
	test_output_domains()
	test_iteration_domains()
}

// Test_Allocation measures derivation for every supported HMAC kind.
func Test_Allocation(t *testing.T) {
	for _, kind := range test_kinds() {
		fixture := allocation_fixture{
			Output:     make(output_storage, TEST_OUTPUT_SIZE),
			Kind:       kind,
			Password:   pbkdf2.Password("password"),
			Salt:       pbkdf2.Salt("salt"),
			Iterations: TEST_ITERATION_COUNT,
		}
		testify.Zero_Allocation(t, func() {
			fixture.Count, fixture.Status = pbkdf2.Key_Into(
				pbkdf2.Destination(fixture.Output), fixture.Kind, fixture.Password,
				fixture.Salt, fixture.Iterations,
			)
		})
	}
}

const TEST_OUTPUT_SIZE = sha512.DIGEST_512_SIZE

const REFERENCE_ITERATION_COUNT pbkdf2.Iteration_Count = 1 << 12

const TEST_ITERATION_COUNT = pbkdf2.ITERATION_COUNT_MINIMUM + binary.UINT_16_SIZE

const TEST_SENTINEL byte = bits.WORD_8_MAXIMUM

type kind_list []hmac.Kind

func test_kinds() (kinds kind_list) {
	return kind_list{
		hmac.KIND_MD5,
		hmac.KIND_SHA_1,
		hmac.KIND_SHA_224,
		hmac.KIND_SHA_256,
		hmac.KIND_SHA_384,
		hmac.KIND_SHA_512_224,
		hmac.KIND_SHA_512_256,
		hmac.KIND_SHA_512,
	}
}

func test_digest_size(kind hmac.Kind) (size int) {
	var digest hmac.Digest
	hmac.Digest_Init(&digest, kind, nil)
	return int(hmac.Digest_Size(&digest))
}

func test_input_domains() {
	var input [pbkdf2.INPUT_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		pbkdf2.INPUT_SIZE_MINIMUM,
		pbkdf2.INPUT_SIZE_MINIMUM + binary.UINT_8_SIZE,
		pbkdf2.INPUT_SIZE_MINIMUM + binary.UINT_16_SIZE,
		pbkdf2.INPUT_SIZE_MAXIMUM - binary.UINT_8_SIZE,
		pbkdf2.INPUT_SIZE_MAXIMUM,
	} {
		pbkdf2.Key_Into(
			nil, hmac.KIND_SHA_256, input[:size], input[:size],
			pbkdf2.ITERATION_COUNT_MINIMUM,
		)
	}
}

func test_output_domains() {
	var output [pbkdf2.OUTPUT_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		pbkdf2.OUTPUT_SIZE_MINIMUM,
		pbkdf2.OUTPUT_SIZE_MINIMUM + binary.UINT_8_SIZE,
		pbkdf2.OUTPUT_SIZE_MINIMUM + binary.UINT_16_SIZE,
		pbkdf2.OUTPUT_SIZE_MAXIMUM - binary.UINT_8_SIZE,
		pbkdf2.OUTPUT_SIZE_MAXIMUM,
	} {
		pbkdf2.Key_Into(
			output[:size], hmac.KIND_MD5, nil, nil,
			pbkdf2.ITERATION_COUNT_MINIMUM,
		)
	}
}

func test_iteration_domains() {
	for _, iterations := range [...]pbkdf2.Iteration_Count{
		pbkdf2.ITERATION_COUNT_MINIMUM,
		pbkdf2.ITERATION_COUNT_MINIMUM + binary.UINT_8_SIZE,
		pbkdf2.ITERATION_COUNT_MINIMUM + binary.UINT_16_SIZE,
		pbkdf2.ITERATION_COUNT_MAXIMUM - binary.UINT_8_SIZE,
		pbkdf2.ITERATION_COUNT_MAXIMUM,
	} {
		pbkdf2.Key_Into(nil, hmac.KIND_SHA_512, nil, nil, iterations)
	}
}

type allocation_fixture struct {
	Output     output_storage
	Kind       hmac.Kind
	Password   pbkdf2.Password
	Salt       pbkdf2.Salt
	Iterations pbkdf2.Iteration_Count
	Count      pbkdf2.Count
	Status     pbkdf2.Status
}

type output_storage []byte
