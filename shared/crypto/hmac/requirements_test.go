package hmac_test

import (
	"testing"

	"local/james-orcales/shared/crypto/hmac"
	"local/james-orcales/shared/crypto/md5"
	"local/james-orcales/shared/crypto/sha1"
	"local/james-orcales/shared/crypto/sha256"
	"local/james-orcales/shared/crypto/sha512"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/encoding/hex"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// Test_Package_Owned_State checks kind-selected state without shared hash dispatch.
func Test_Package_Owned_State(t *testing.T) {
	for _, kind := range test_kinds() {
		var digest hmac.Digest
		hmac.Digest_Init(&digest, kind, hmac.Key("key"))
		testify.Equal(t, hmac.Size(test_digest_size(kind)),
			hmac.Digest_Size(&digest))
		testify.Equal(t, hmac.Block_Size(test_block_size(kind)),
			hmac.Digest_Block_Size(&digest))
	}
}

// Test_Reference_Values binds every supported kind to fixed known answers.
func Test_Reference_Values(t *testing.T) {
	key := hmac.Key("Jefe")
	source := hmac.Source("what do ya want for nothing?")
	tests := [...]struct {
		Kind hmac.Kind
		Want string
	}{
		{Kind: hmac.KIND_MD5, Want: REFERENCE_MD5},
		{Kind: hmac.KIND_SHA_1, Want: REFERENCE_SHA_1},
		{Kind: hmac.KIND_SHA_224, Want: REFERENCE_SHA_224},
		{Kind: hmac.KIND_SHA_256, Want: REFERENCE_SHA_256},
		{Kind: hmac.KIND_SHA_384, Want: REFERENCE_SHA_384},
		{Kind: hmac.KIND_SHA_512_224, Want: REFERENCE_SHA_512_224},
		{Kind: hmac.KIND_SHA_512_256, Want: REFERENCE_SHA_512_256},
		{Kind: hmac.KIND_SHA_512, Want: REFERENCE_SHA_512},
	}
	for _, test := range tests {
		testify.Equal(t, test.Want, encoded(t, sum(test.Kind, key, source)))
	}
}

// Test_Stream checks call boundaries do not change tag.
func Test_Stream(t *testing.T) {
	key := hmac.Key("stream key")
	source := hmac.Source("bounded messages retain HMAC state across calls")
	for _, kind := range test_kinds() {
		var digest hmac.Digest
		hmac.Digest_Init(&digest, kind, key)
		midpoint := len(source) >> binary.UINT_8_SIZE
		hmac.Digest_Write(&digest, source[:midpoint])
		hmac.Digest_Write(&digest, source[midpoint:])
		verify_digest(t, &digest, kind, key, source)
	}
}

// Test_Key_Reduction checks keys beyond both supported block widths.
func Test_Key_Reduction(t *testing.T) {
	var key [hmac.BLOCK_SIZE_MAXIMUM + binary.UINT_8_SIZE]byte
	for index := range key {
		key[index] = byte(index)
	}
	for _, kind := range test_kinds() {
		source := hmac.Source("key reduction")
		testify.Equal(
			t, sum(kind, reduced_key(kind, key[:]), source), sum(kind, key[:], source),
		)
	}
}

// Test_Caller_Owned_Output keeps short storage untouched and reports selected width.
func Test_Caller_Owned_Output(t *testing.T) {
	for _, kind := range test_kinds() {
		var digest hmac.Digest
		hmac.Digest_Init(&digest, kind, hmac.Key("key"))
		hmac.Digest_Write(&digest, hmac.Source("message"))
		size := int(hmac.Digest_Size(&digest))
		var short [hmac.DIGEST_SIZE_MAXIMUM]byte
		short[size-binary.UINT_8_SIZE] = TEST_SENTINEL
		count, status := hmac.Digest_Sum_Into(
			&digest, short[:size-binary.UINT_8_SIZE],
		)
		testify.Equal(t, hmac.Output_Count(size), count)
		testify.Equal(t, hmac.OUTPUT_STATUS_TOO_SMALL, status)
		testify.Equal(t, TEST_SENTINEL, short[size-binary.UINT_8_SIZE])
		var output [hmac.DIGEST_SIZE_MAXIMUM]byte
		count, status = hmac.Digest_Sum_Into(&digest, output[:size])
		testify.Equal(t, hmac.Output_Count(size), count)
		testify.Equal(t, hmac.OUTPUT_STATUS_OK, status)
	}
}

// Test_Reset_And_Clone checks key retention and independent live state.
func Test_Reset_And_Clone(t *testing.T) {
	for _, kind := range test_kinds() {
		var source hmac.Digest
		hmac.Digest_Init(&source, kind, hmac.Key("key"))
		hmac.Digest_Write(&source, hmac.Source("prefix"))
		var clone hmac.Digest
		hmac.Digest_Clone_Into(&clone, &source)
		hmac.Digest_Write(&source, hmac.Source(" source"))
		hmac.Digest_Write(&clone, hmac.Source(" clone"))
		verify_digest(t, &source, kind, hmac.Key("key"),
			hmac.Source("prefix source"))
		verify_digest(t, &clone, kind, hmac.Key("key"),
			hmac.Source("prefix clone"))
		hmac.Digest_Reset(&source)
		verify_digest(t, &source, kind, hmac.Key("key"), nil)
	}
}

// Test_Constant_Time_Equality checks equal, unequal, and different-width tags.
func Test_Constant_Time_Equality(t *testing.T) {
	left := hmac.Tag("same tag")
	testify.True(t, bool(hmac.Equal(left, hmac.Tag("same tag"))))
	testify.False(t, bool(hmac.Equal(left, hmac.Tag("same tam"))))
	testify.False(t, bool(hmac.Equal(left, hmac.Tag("short"))))
}

// Test_Bounds rejects hostile collections and uninitialized state.
func Test_Bounds(t *testing.T) {
	var digest hmac.Digest
	testify.Panics(t, func() { hmac.Digest_Write(&digest, nil) })
	testify.Panics(t, func() { hmac.Digest_Sum_Into(&digest, nil) })
	testify.Panics(t, func() { hmac.Digest_Reset(&digest) })
	testify.Panics(t, func() { hmac.Digest_Size(&digest) })
	testify.Panics(t, func() { hmac.Digest_Block_Size(&digest) })
	var initialized hmac.Digest
	hmac.Digest_Init(&initialized, hmac.KIND_SHA_256, nil)
	testify.Panics(t, func() {
		hmac.Digest_Clone_Into(&initialized, &digest)
	})
	var oversized [hmac.SOURCE_SIZE_MAXIMUM + binary.UINT_8_SIZE]byte
	testify.Panics(t, func() {
		hmac.Digest_Init(&digest, hmac.KIND_SHA_256, oversized[:])
	})
	hmac.Digest_Init(&digest, hmac.KIND_SHA_256, nil)
	testify.Panics(t, func() { hmac.Digest_Write(&digest, oversized[:]) })
	testify.Panics(t, func() { hmac.Digest_Sum_Into(&digest, oversized[:]) })
	testify.Panics(t, func() { hmac.Equal(oversized[:], nil) })
	testify.Panics(t, func() { hmac.Equal(nil, oversized[:]) })
	testify.Panics(t, func() {
		hmac.Digest_Init(&digest, hmac.Kind(hmac.KIND_COUNT), nil)
	})
}

// Test_Invariant_Domains drives every scalar and collection boundary through runtime APIs.
func Test_Invariant_Domains(t *testing.T) {
	test_key_and_kind_domains()
	test_source_domains()
	test_destination_and_tag_domains()
	test_state_domains()
}

// Test_Allocation measures every exported operation for every selected hash.
func Test_Allocation(t *testing.T) {
	for _, kind := range test_kinds() {
		fixture := allocation_fixture{
			Destination: make(
				hmac.Destination, hmac.DESTINATION_SIZE_MAXIMUM,
			),
			Kind: kind,
			Key:  hmac.Key("allocation key"),
			Left: hmac.Tag("same"),
		}
		fixture.Source = hmac.Source("allocation source")
		hmac.Digest_Init(&fixture.Digest, kind, fixture.Key)
		test_kind_allocation(t, &fixture)
	}
}

const TEST_SENTINEL byte = bits.WORD_8_MAXIMUM

const REFERENCE_MD5 = "750c783e6ab0b503eaa86e310a5db738"

const REFERENCE_SHA_1 = "effcdf6ae5eb2fa2d27416d5f184df9c259a7c79"

const REFERENCE_SHA_224 = "a30e01098bc6dbbf45690f3a7e9e6d0f" +
	"8bbea2a39e6148008fd05e44"

const REFERENCE_SHA_256 = "5bdcc146bf60754e6a042426089575c75" +
	"a003f089d2739839dec58b964ec3843"

const REFERENCE_SHA_384 = "af45d2e376484031617f78d2b58a6b1b" +
	"9c7ef464f5a01b47e42ec3736322445e" +
	"8e2240ca5e69e2c78b3239ecfab21649"

const REFERENCE_SHA_512_224 = "4a530b31a79ebcce36916546317c45f2" +
	"47d83241dfb818fd37254bde"

const REFERENCE_SHA_512_256 = "6df7b24630d5ccb2ee335407081a8718" +
	"8c221489768fa2020513b2d593359456"

const REFERENCE_SHA_512 = "164b7a7bfcf819e2e395fbe73b56e0a3" +
	"87bd64222e831fd610270cd7ea250554" +
	"9758bf75c05a994a6d034f65f8f0e6fd" +
	"caeab1a34d4a6b4b636e070a38bce737"

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

func verify_digest(
	t *testing.T,
	digest *hmac.Digest,
	kind hmac.Kind,
	key hmac.Key,
	source hmac.Source,
) {
	size := hmac.Output_Count(hmac.Digest_Size(digest))
	value := make([]byte, size)
	hmac.Digest_Sum_Into(digest, value)
	want := sum(kind, key, source)
	testify.Equal(t, size, hmac.Output_Count(len(want)))
	testify.Equal(t, want, test_value(value))
}

type test_value []byte

func sum(
	kind hmac.Kind,
	key hmac.Key,
	source hmac.Source,
) (value test_value) {
	var digest hmac.Digest
	hmac.Digest_Init(&digest, kind, key)
	hmac.Digest_Write(&digest, source)
	value = make(test_value, hmac.Digest_Size(&digest))
	hmac.Digest_Sum_Into(&digest, hmac.Destination(value))
	return value
}

func reduced_key(kind hmac.Kind, key hmac.Key) (value hmac.Key) {
	value = make(hmac.Key, test_digest_size(kind))
	switch kind {
	case hmac.KIND_MD5:
		md5.Checksum_Into(md5.Destination(value), md5.Source(key))
	case hmac.KIND_SHA_1:
		sha1.Checksum_Into(sha1.Destination(value), sha1.Source(key))
	case hmac.KIND_SHA_224:
		sha256.Checksum_Into(
			sha256.Destination(value), sha256.KIND_SHA_224, sha256.Source(key),
		)
	case hmac.KIND_SHA_256:
		sha256.Checksum_Into(
			sha256.Destination(value), sha256.KIND_SHA_256, sha256.Source(key),
		)
	case hmac.KIND_SHA_384:
		sha512.Checksum_Into(
			sha512.Destination(value), sha512.KIND_SHA_384, sha512.Source(key),
		)
	case hmac.KIND_SHA_512_224:
		sha512.Checksum_Into(
			sha512.Destination(value), sha512.KIND_SHA_512_224, sha512.Source(key),
		)
	case hmac.KIND_SHA_512_256:
		sha512.Checksum_Into(
			sha512.Destination(value), sha512.KIND_SHA_512_256, sha512.Source(key),
		)
	case hmac.KIND_SHA_512:
		sha512.Checksum_Into(
			sha512.Destination(value), sha512.KIND_SHA_512, sha512.Source(key),
		)
	}
	return value
}

func encoded(t *testing.T, source test_value) (value string) {
	t.Helper()
	var output [hmac.DIGEST_SIZE_MAXIMUM * hex.ENCODED_BYTE_SIZE]byte
	count, status := hex.Encode_Into(output[:], hex.Source(source))
	testify.Equal(t, hex.Encode_Status(hex.STATUS_OK), status)
	return string(output[:count])
}

func test_digest_size(kind hmac.Kind) (size int) {
	return len(sum(kind, nil, nil))
}

func test_block_size(kind hmac.Kind) (size int) {
	if kind <= hmac.KIND_SHA_256 {
		return hmac.BLOCK_SIZE_MINIMUM
	}
	return hmac.BLOCK_SIZE_MAXIMUM
}

func test_key_and_kind_domains() {
	var key [hmac.KEY_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		hmac.KEY_SIZE_MINIMUM,
		hmac.KEY_SIZE_MINIMUM + binary.UINT_8_SIZE,
		hmac.KEY_SIZE_MINIMUM + binary.UINT_16_SIZE,
		hmac.BLOCK_SIZE_MINIMUM + binary.UINT_8_SIZE,
		hmac.KEY_SIZE_MAXIMUM,
	} {
		for _, kind := range test_kinds() {
			var digest hmac.Digest
			hmac.Digest_Init(&digest, kind, key[:size])
		}
	}
}

func test_source_domains() {
	var source [hmac.SOURCE_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		hmac.SOURCE_SIZE_MINIMUM,
		hmac.SOURCE_SIZE_MINIMUM + binary.UINT_8_SIZE,
		hmac.SOURCE_SIZE_MINIMUM + binary.UINT_16_SIZE,
		hmac.SOURCE_SIZE_MAXIMUM,
	} {
		var digest hmac.Digest
		hmac.Digest_Init(&digest, hmac.KIND_SHA_256, nil)
		hmac.Digest_Write(&digest, source[:size])
	}
}

func test_destination_and_tag_domains() {
	var destination [hmac.DESTINATION_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		hmac.DESTINATION_SIZE_MINIMUM,
		hmac.DESTINATION_SIZE_MINIMUM + binary.UINT_8_SIZE,
		hmac.DESTINATION_SIZE_MINIMUM + binary.UINT_16_SIZE,
		hmac.DESTINATION_SIZE_MAXIMUM,
	} {
		var digest hmac.Digest
		hmac.Digest_Init(&digest, hmac.KIND_SHA_256, nil)
		hmac.Digest_Sum_Into(&digest, destination[:size])
		hmac.Equal(destination[:size], destination[:size])
	}
	hmac.Equal(hmac.Tag("same"), hmac.Tag("different"))
}

func test_state_domains() {
	for _, kind := range test_kinds() {
		var digest hmac.Digest
		hmac.Digest_Init(&digest, kind, nil)
		hmac.Digest_Init(&digest, kind, nil)
		hmac.Digest_Reset(&digest)
		hmac.Digest_Size(&digest)
		hmac.Digest_Block_Size(&digest)
		var output [hmac.DIGEST_SIZE_MAXIMUM]byte
		hmac.Digest_Sum_Into(&digest, output[:hmac.Digest_Size(&digest)])
		var clone hmac.Digest
		hmac.Digest_Clone_Into(&clone, &digest)
	}
}

func test_kind_allocation(t *testing.T, fixture *allocation_fixture) {
	testify.Zero_Allocation(t, func() {
		hmac.Digest_Init(&fixture.Digest, fixture.Kind, fixture.Key)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Count = hmac.Digest_Write(&fixture.Digest, fixture.Source)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Output_Count, fixture.Output_Status = hmac.Digest_Sum_Into(
			&fixture.Digest, fixture.Destination,
		)
	})
	testify.Zero_Allocation(t, func() { hmac.Digest_Reset(&fixture.Digest) })
	testify.Zero_Allocation(t, func() {
		hmac.Digest_Clone_Into(&fixture.Clone, &fixture.Digest)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Size = hmac.Digest_Size(&fixture.Digest)
		fixture.Block_Size = hmac.Digest_Block_Size(&fixture.Digest)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Equal = hmac.Equal(fixture.Left, fixture.Left)
	})
}

type allocation_fixture struct {
	Digest        hmac.Digest
	Clone         hmac.Digest
	Destination   hmac.Destination
	Key           hmac.Key
	Source        hmac.Source
	Left          hmac.Tag
	Kind          hmac.Kind
	Count         hmac.Count
	Output_Count  hmac.Output_Count
	Output_Status hmac.Output_Status
	Size          hmac.Size
	Block_Size    hmac.Block_Size
	Equal         hmac.Equality
}
