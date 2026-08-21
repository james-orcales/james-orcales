package pbkdf2_test

import (
	"crypto/md5"
	standard_pbkdf2 "crypto/pbkdf2"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"hash"
	"testing"

	"local/james-orcales/shared/crypto/hmac"
	"local/james-orcales/shared/crypto/pbkdf2"
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
		var standard_output [TEST_OUTPUT_SIZE]byte
		standard_count, standard_status := pbkdf2.Key_Into(
			standard_output[:], kind, password, salt, TEST_ITERATION_COUNT,
		)
		testify.Equal(t, pbkdf2.Count(len(standard_output)), standard_count)
		testify.Equal(t, pbkdf2.STATUS_OK, standard_status)
		standard_want := standard_key(
			kind, password, salt, TEST_ITERATION_COUNT, len(standard_output),
		)
		testify.Equal(t, standard_want, standard_output[:])
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
	var output [sha1.Size + binary.UINT_8_SIZE]byte
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
			Kind:       kind,
			Password:   pbkdf2.Password("password"),
			Salt:       pbkdf2.Salt("salt"),
			Iterations: TEST_ITERATION_COUNT,
		}
		testify.Zero_Allocation(t, func() {
			fixture.Count, fixture.Status = pbkdf2.Key_Into(
				fixture.Output[:], fixture.Kind, fixture.Password,
				fixture.Salt, fixture.Iterations,
			)
		})
	}
}

const TEST_OUTPUT_SIZE = sha512.Size

const REFERENCE_ITERATION_COUNT pbkdf2.Iteration_Count = 1 << 12

const TEST_ITERATION_COUNT = pbkdf2.ITERATION_COUNT_MINIMUM + binary.UINT_16_SIZE

const TEST_SENTINEL byte = bits.WORD_8_MAXIMUM

func test_kinds() (kinds [hmac.KIND_COUNT]hmac.Kind) {
	return [hmac.KIND_COUNT]hmac.Kind{
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

func standard_key(
	kind hmac.Kind,
	password pbkdf2.Password,
	salt pbkdf2.Salt,
	iterations pbkdf2.Iteration_Count,
	output_size int,
) (output []byte) {
	var hash_new func() (digest hash.Hash)
	switch kind {
	case hmac.KIND_MD5:
		hash_new = md5.New
	case hmac.KIND_SHA_1:
		hash_new = sha1.New
	case hmac.KIND_SHA_224:
		hash_new = sha256.New224
	case hmac.KIND_SHA_256:
		hash_new = sha256.New
	case hmac.KIND_SHA_384:
		hash_new = sha512.New384
	case hmac.KIND_SHA_512_224:
		hash_new = sha512.New512_224
	case hmac.KIND_SHA_512_256:
		hash_new = sha512.New512_256
	case hmac.KIND_SHA_512:
		hash_new = sha512.New
	}
	output, err := standard_pbkdf2.Key(
		hash_new, string(password), salt, int(iterations), output_size,
	)
	if err != nil {
		panic(err)
	}
	return output
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
	Output     [TEST_OUTPUT_SIZE]byte
	Kind       hmac.Kind
	Password   pbkdf2.Password
	Salt       pbkdf2.Salt
	Iterations pbkdf2.Iteration_Count
	Count      pbkdf2.Count
	Status     pbkdf2.Status
}
