package hkdf_test

import (
	"testing"

	"local/james-orcales/shared/crypto/hkdf"
	"local/james-orcales/shared/crypto/hmac"
	"local/james-orcales/shared/crypto/sha256"
	"local/james-orcales/shared/crypto/sha512"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// Test_Reference_Values binds extract and expand to RFC 5869 test case one.
func Test_Reference_Values(t *testing.T) {
	secret := hkdf.Secret{
		0x0b, 0x0b, 0x0b, 0x0b, 0x0b, 0x0b, 0x0b, 0x0b,
		0x0b, 0x0b, 0x0b, 0x0b, 0x0b, 0x0b, 0x0b, 0x0b,
		0x0b, 0x0b, 0x0b, 0x0b, 0x0b, 0x0b,
	}
	salt := hkdf.Salt{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06,
		0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c,
	}
	info := hkdf.Info{0xf0, 0xf1, 0xf2, 0xf3, 0xf4, 0xf5, 0xf6, 0xf7, 0xf8, 0xf9}
	want_pseudorandom_key := [...]byte{
		0x07, 0x77, 0x09, 0x36, 0x2c, 0x2e, 0x32, 0xdf,
		0x0d, 0xdc, 0x3f, 0x0d, 0xc4, 0x7b, 0xba, 0x63,
		0x90, 0xb6, 0xc7, 0x3b, 0xb5, 0x0f, 0x9c, 0x31,
		0x22, 0xec, 0x84, 0x4a, 0xd7, 0xc2, 0xb3, 0xe5,
	}
	want_output := [...]byte{
		0x3c, 0xb2, 0x5f, 0x25, 0xfa, 0xac, 0xd5, 0x7a,
		0x90, 0x43, 0x4f, 0x64, 0xd0, 0x36, 0x2f, 0x2a,
		0x2d, 0x2d, 0x0a, 0x90, 0xcf, 0x1a, 0x5a, 0x4c,
		0x5d, 0xb0, 0x2d, 0x56, 0xec, 0xc4, 0xc5, 0xbf,
		0x34, 0x00, 0x72, 0x08, 0xd5, 0xb8, 0x87, 0x18,
		0x58, 0x65,
	}
	var pseudorandom_key [hmac.DIGEST_SIZE_MAXIMUM]byte
	count, status := hkdf.Extract_Into(
		pseudorandom_key[:], hmac.KIND_SHA_256, secret, salt,
	)
	testify.Equal(t, hkdf.Extract_Count(len(want_pseudorandom_key)), count)
	testify.Equal(t, hkdf.EXTRACT_STATUS_OK, status)
	testify.Equal(t, want_pseudorandom_key[:], pseudorandom_key[:count])
	var output [len(want_output)]byte
	derived, expand_status := hkdf.Expand_Into(
		output[:], hmac.KIND_SHA_256, pseudorandom_key[:count], info,
	)
	testify.Equal(t, hkdf.Count(len(output)), derived)
	testify.Equal(t, hkdf.EXPAND_STATUS_OK, expand_status)
	testify.Equal(t, want_output, output)
}

// Test_Extract_And_Expand checks direct derivation composes the public bounded stages.
func Test_Extract_And_Expand(t *testing.T) {
	secret := hkdf.Secret("input key material")
	salt := hkdf.Salt("independent salt")
	info := hkdf.Info("context")
	for _, kind := range test_kinds() {
		var output [TEST_OUTPUT_SIZE]byte
		count, status := hkdf.Key_Into(output[:], kind, secret, salt, info)
		testify.Equal(t, hkdf.Count(len(output)), count)
		testify.Equal(t, hkdf.EXPAND_STATUS_OK, status)
		var pseudorandom_key [hmac.DIGEST_SIZE_MAXIMUM]byte
		extract_count, extract_status := hkdf.Extract_Into(
			pseudorandom_key[:], kind, secret, salt,
		)
		testify.Equal(t, hkdf.EXTRACT_STATUS_OK, extract_status)
		var expanded [TEST_OUTPUT_SIZE]byte
		expand_count, expand_status := hkdf.Expand_Into(
			expanded[:], kind, pseudorandom_key[:extract_count], info,
		)
		testify.Equal(t, hkdf.Count(len(expanded)), expand_count)
		testify.Equal(t, hkdf.EXPAND_STATUS_OK, expand_status)
		testify.Equal(t, output, expanded)
	}
}

// Test_Caller_Owned_Output keeps refused extract and expansion storage untouched.
func Test_Caller_Owned_Output(t *testing.T) {
	var pseudorandom_key [hmac.DIGEST_SIZE_MAXIMUM]byte
	pseudorandom_key[bits.BIT_COUNT_MINIMUM] = TEST_SENTINEL
	count, status := hkdf.Extract_Into(
		pseudorandom_key[:sha256.DIGEST_256_SIZE-binary.UINT_8_SIZE],
		hmac.KIND_SHA_256, nil, nil,
	)
	testify.Equal(t, hkdf.Extract_Count(sha256.DIGEST_256_SIZE), count)
	testify.Equal(t, hkdf.EXTRACT_STATUS_TOO_SMALL, status)
	testify.Equal(t, TEST_SENTINEL, pseudorandom_key[bits.BIT_COUNT_MINIMUM])

	limit := int(hkdf.Output_Size_Maximum(hmac.KIND_SHA_1))
	var output [hkdf.OUTPUT_SIZE_MAXIMUM]byte
	output[bits.BIT_COUNT_MINIMUM] = TEST_SENTINEL
	derived, expand_status := hkdf.Expand_Into(
		output[:limit+binary.UINT_8_SIZE], hmac.KIND_SHA_1,
		hkdf.Pseudorandom_Key("key"), nil,
	)
	testify.Equal(t, hkdf.COUNT_EMPTY, derived)
	testify.Equal(t, hkdf.EXPAND_STATUS_TOO_LARGE, expand_status)
	testify.Equal(t, TEST_SENTINEL, output[bits.BIT_COUNT_MINIMUM])
}

// Test_Output_Limit checks formula maximum succeeds for widest selected hash.
func Test_Output_Limit(t *testing.T) {
	var output [hkdf.OUTPUT_SIZE_MAXIMUM]byte
	count, status := hkdf.Expand_Into(
		output[:], hmac.KIND_SHA_512, hkdf.Pseudorandom_Key("key"), nil,
	)
	testify.Equal(t, hkdf.Count(len(output)), count)
	testify.Equal(t, hkdf.EXPAND_STATUS_OK, status)
	testify.Equal(t, hkdf.Size(len(output)), hkdf.Output_Size_Maximum(hmac.KIND_SHA_512))
}

// Test_Bounds rejects hostile inputs and output beyond formula capacity.
func Test_Bounds(t *testing.T) {
	var oversized [hkdf.INPUT_SIZE_MAXIMUM + binary.UINT_8_SIZE]byte
	var output [hkdf.OUTPUT_SIZE_MAXIMUM]byte
	testify.Panics(t, func() {
		hkdf.Extract_Into(output[:], hmac.KIND_SHA_256, oversized[:], nil)
	})
	testify.Panics(t, func() {
		hkdf.Extract_Into(output[:], hmac.KIND_SHA_256, nil, oversized[:])
	})
	testify.Panics(t, func() {
		hkdf.Expand_Into(output[:], hmac.KIND_SHA_256, oversized[:], nil)
	})
	testify.Panics(t, func() {
		hkdf.Expand_Into(output[:], hmac.KIND_SHA_256, nil, oversized[:])
	})
	var output_oversized [hkdf.OUTPUT_SIZE_MAXIMUM + binary.UINT_8_SIZE]byte
	testify.Panics(t, func() {
		hkdf.Key_Into(output_oversized[:], hmac.KIND_SHA_512, nil, nil, nil)
	})
	testify.Panics(t, func() {
		hkdf.Key_Into(output[:], hmac.Kind(hmac.KIND_COUNT), nil, nil, nil)
	})
}

// Test_Invariant_Domains drives each scalar, input, and output boundary through public APIs.
func Test_Invariant_Domains(t *testing.T) {
	test_kind_domains()
	test_input_domains()
	test_output_domains()
}

// Test_Allocation measures extract, expand, key, and size operations for every kind.
func Test_Allocation(t *testing.T) {
	for _, kind := range test_kinds() {
		fixture := allocation_fixture{
			Extract: make(extract_storage, hmac.DIGEST_SIZE_MAXIMUM),
			Output:  make(output_storage, TEST_OUTPUT_SIZE),
			Kind:    kind,
			Secret:  hkdf.Secret("secret"),
			Salt:    hkdf.Salt("salt"),
			Info:    hkdf.Info("info"),
		}
		fixture.Pseudorandom_Key = hkdf.Pseudorandom_Key("pseudorandom key")
		test_kind_allocation(t, &fixture)
	}
}

const TEST_OUTPUT_SIZE = sha512.DIGEST_512_SIZE

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

func test_kind_domains() {
	for _, kind := range test_kinds() {
		var output [TEST_OUTPUT_SIZE]byte
		hkdf.Key_Into(output[:], kind, nil, nil, nil)
		hkdf.Pseudorandom_Key_Size(kind)
		hkdf.Output_Size_Maximum(kind)
	}
}

func test_input_domains() {
	var input [hkdf.INPUT_SIZE_MAXIMUM]byte
	var extract [hmac.DIGEST_SIZE_MAXIMUM]byte
	var output [TEST_OUTPUT_SIZE]byte
	for _, size := range [...]int{
		hkdf.INPUT_SIZE_MINIMUM,
		hkdf.INPUT_SIZE_MINIMUM + binary.UINT_8_SIZE,
		hkdf.INPUT_SIZE_MINIMUM + binary.UINT_16_SIZE,
		hkdf.INPUT_SIZE_MAXIMUM,
	} {
		hkdf.Extract_Into(extract[:], hmac.KIND_SHA_256, input[:size], input[:size])
		hkdf.Expand_Into(output[:], hmac.KIND_SHA_256, input[:size], input[:size])
		hkdf.Key_Into(
			output[:], hmac.KIND_SHA_256, input[:size], input[:size], input[:size],
		)
	}
}

func test_output_domains() {
	var output [hkdf.OUTPUT_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		hkdf.OUTPUT_SIZE_MINIMUM,
		hkdf.OUTPUT_SIZE_MINIMUM + binary.UINT_8_SIZE,
		hkdf.OUTPUT_SIZE_MINIMUM + binary.UINT_16_SIZE,
		hkdf.OUTPUT_SIZE_MAXIMUM,
	} {
		hkdf.Expand_Into(output[:size], hmac.KIND_SHA_512, nil, nil)
		hkdf.Key_Into(output[:size], hmac.KIND_SHA_512, nil, nil, nil)
	}
	for _, size := range [...]int{
		hkdf.EXTRACT_DESTINATION_SIZE_MINIMUM,
		hkdf.EXTRACT_DESTINATION_SIZE_MINIMUM + binary.UINT_8_SIZE,
		hkdf.EXTRACT_DESTINATION_SIZE_MINIMUM + binary.UINT_16_SIZE,
		hkdf.EXTRACT_DESTINATION_SIZE_MAXIMUM,
	} {
		hkdf.Extract_Into(output[:size], hmac.KIND_SHA_512, nil, nil)
	}
	limit := int(hkdf.Output_Size_Maximum(hmac.KIND_SHA_1))
	hkdf.Expand_Into(output[:limit+binary.UINT_8_SIZE], hmac.KIND_SHA_1, nil, nil)
	hkdf.Key_Into(output[:limit+binary.UINT_8_SIZE], hmac.KIND_SHA_1, nil, nil, nil)
}

func test_kind_allocation(t *testing.T, fixture *allocation_fixture) {
	testify.Zero_Allocation(t, func() {
		fixture.Extract_Count, fixture.Extract_Status = hkdf.Extract_Into(
			hkdf.Extract_Destination(fixture.Extract), fixture.Kind,
			fixture.Secret, fixture.Salt,
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Count, fixture.Expand_Status = hkdf.Expand_Into(
			hkdf.Destination(fixture.Output), fixture.Kind,
			fixture.Pseudorandom_Key, fixture.Info,
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Count, fixture.Expand_Status = hkdf.Key_Into(
			hkdf.Destination(fixture.Output), fixture.Kind,
			fixture.Secret, fixture.Salt, fixture.Info,
		)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Extract_Count = hkdf.Pseudorandom_Key_Size(fixture.Kind)
		fixture.Size = hkdf.Output_Size_Maximum(fixture.Kind)
	})
}

type allocation_fixture struct {
	Extract          extract_storage
	Output           output_storage
	Secret           hkdf.Secret
	Salt             hkdf.Salt
	Pseudorandom_Key hkdf.Pseudorandom_Key
	Info             hkdf.Info
	Kind             hmac.Kind
	Extract_Count    hkdf.Extract_Count
	Extract_Status   hkdf.Extract_Status
	Count            hkdf.Count
	Expand_Status    hkdf.Expand_Status
	Size             hkdf.Size
}

type extract_storage []byte

type output_storage []byte
