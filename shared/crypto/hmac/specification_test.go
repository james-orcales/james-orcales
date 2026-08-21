package hmac_test

import (
	standard_hmac "crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"testing"

	shared_hmac "local/james-orcales/shared/crypto/hmac"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// Test_Package_Owned_State checks kind-selected state without shared hash dispatch.
func Test_Package_Owned_State(t *testing.T) {
	for _, kind := range test_kinds() {
		var digest shared_hmac.Digest
		shared_hmac.Digest_Init(&digest, kind, shared_hmac.Key("key"))
		testify.Equal(t, shared_hmac.Size(test_digest_size(kind)),
			shared_hmac.Digest_Size(&digest))
		testify.Equal(t, shared_hmac.Block_Size(test_block_size(kind)),
			shared_hmac.Digest_Block_Size(&digest))
	}
}

// Test_Reference_Values binds every supported kind to crypto/hmac.
func Test_Reference_Values(t *testing.T) {
	key := shared_hmac.Key("Jefe")
	source := shared_hmac.Source("what do ya want for nothing?")
	for _, kind := range test_kinds() {
		verify_reference(t, kind, key, source)
	}
}

// Test_Stream checks call boundaries do not change tag.
func Test_Stream(t *testing.T) {
	key := shared_hmac.Key("stream key")
	source := shared_hmac.Source("bounded messages retain HMAC state across calls")
	for _, kind := range test_kinds() {
		var digest shared_hmac.Digest
		shared_hmac.Digest_Init(&digest, kind, key)
		midpoint := len(source) >> binary.UINT_8_SIZE
		shared_hmac.Digest_Write(&digest, source[:midpoint])
		shared_hmac.Digest_Write(&digest, source[midpoint:])
		verify_digest(t, &digest, kind, key, source)
	}
}

// Test_Key_Reduction checks keys beyond both supported block widths.
func Test_Key_Reduction(t *testing.T) {
	var key [shared_hmac.BLOCK_SIZE_MAXIMUM + binary.UINT_8_SIZE]byte
	for index := range key {
		key[index] = byte(index)
	}
	for _, kind := range test_kinds() {
		verify_reference(t, kind, key[:], shared_hmac.Source("key reduction"))
	}
}

// Test_Caller_Owned_Output keeps short storage untouched and reports selected width.
func Test_Caller_Owned_Output(t *testing.T) {
	for _, kind := range test_kinds() {
		var digest shared_hmac.Digest
		shared_hmac.Digest_Init(&digest, kind, shared_hmac.Key("key"))
		shared_hmac.Digest_Write(&digest, shared_hmac.Source("message"))
		size := int(shared_hmac.Digest_Size(&digest))
		var short [shared_hmac.DIGEST_SIZE_MAXIMUM]byte
		short[size-binary.UINT_8_SIZE] = TEST_SENTINEL
		count, status := shared_hmac.Digest_Sum_Into(
			&digest, short[:size-binary.UINT_8_SIZE],
		)
		testify.Equal(t, shared_hmac.Output_Count(size), count)
		testify.Equal(t, shared_hmac.OUTPUT_STATUS_TOO_SMALL, status)
		testify.Equal(t, TEST_SENTINEL, short[size-binary.UINT_8_SIZE])
		var output [shared_hmac.DIGEST_SIZE_MAXIMUM]byte
		count, status = shared_hmac.Digest_Sum_Into(&digest, output[:size])
		testify.Equal(t, shared_hmac.Output_Count(size), count)
		testify.Equal(t, shared_hmac.OUTPUT_STATUS_OK, status)
	}
}

// Test_Reset_And_Clone checks key retention and independent live state.
func Test_Reset_And_Clone(t *testing.T) {
	for _, kind := range test_kinds() {
		var source shared_hmac.Digest
		shared_hmac.Digest_Init(&source, kind, shared_hmac.Key("key"))
		shared_hmac.Digest_Write(&source, shared_hmac.Source("prefix"))
		var clone shared_hmac.Digest
		shared_hmac.Digest_Clone_Into(&clone, &source)
		shared_hmac.Digest_Write(&source, shared_hmac.Source(" source"))
		shared_hmac.Digest_Write(&clone, shared_hmac.Source(" clone"))
		verify_digest(t, &source, kind, shared_hmac.Key("key"),
			shared_hmac.Source("prefix source"))
		verify_digest(t, &clone, kind, shared_hmac.Key("key"),
			shared_hmac.Source("prefix clone"))
		shared_hmac.Digest_Reset(&source)
		verify_digest(t, &source, kind, shared_hmac.Key("key"), nil)
	}
}

// Test_Constant_Time_Equality checks equal, unequal, and different-width tags.
func Test_Constant_Time_Equality(t *testing.T) {
	left := shared_hmac.Tag("same tag")
	testify.True(t, bool(shared_hmac.Equal(left, shared_hmac.Tag("same tag"))))
	testify.False(t, bool(shared_hmac.Equal(left, shared_hmac.Tag("same tam"))))
	testify.False(t, bool(shared_hmac.Equal(left, shared_hmac.Tag("short"))))
}

// Test_Bounds rejects hostile collections and uninitialized state.
func Test_Bounds(t *testing.T) {
	var digest shared_hmac.Digest
	testify.Panics(t, func() { shared_hmac.Digest_Write(&digest, nil) })
	testify.Panics(t, func() { shared_hmac.Digest_Sum(&digest) })
	var oversized [shared_hmac.SOURCE_SIZE_MAXIMUM + binary.UINT_8_SIZE]byte
	testify.Panics(t, func() {
		shared_hmac.Digest_Init(&digest, shared_hmac.KIND_SHA_256, oversized[:])
	})
	shared_hmac.Digest_Init(&digest, shared_hmac.KIND_SHA_256, nil)
	testify.Panics(t, func() { shared_hmac.Digest_Write(&digest, oversized[:]) })
	testify.Panics(t, func() { shared_hmac.Digest_Sum_Into(&digest, oversized[:]) })
	testify.Panics(t, func() { shared_hmac.Equal(oversized[:], nil) })
	testify.Panics(t, func() { shared_hmac.Equal(nil, oversized[:]) })
	testify.Panics(t, func() {
		shared_hmac.Digest_Init(&digest, shared_hmac.Kind(shared_hmac.KIND_COUNT), nil)
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
			Kind: kind,
			Key:  shared_hmac.Key("allocation key"),
			Left: shared_hmac.Tag("same"),
		}
		fixture.Source = shared_hmac.Source("allocation source")
		shared_hmac.Digest_Init(&fixture.Digest, kind, fixture.Key)
		test_kind_allocation(t, &fixture)
	}
}

const TEST_SENTINEL byte = bits.WORD_8_MAXIMUM

func test_kinds() (kinds [shared_hmac.KIND_COUNT]shared_hmac.Kind) {
	return [shared_hmac.KIND_COUNT]shared_hmac.Kind{
		shared_hmac.KIND_MD5,
		shared_hmac.KIND_SHA_1,
		shared_hmac.KIND_SHA_224,
		shared_hmac.KIND_SHA_256,
		shared_hmac.KIND_SHA_384,
		shared_hmac.KIND_SHA_512_224,
		shared_hmac.KIND_SHA_512_256,
		shared_hmac.KIND_SHA_512,
	}
}

func verify_reference(
	t *testing.T,
	kind shared_hmac.Kind,
	key shared_hmac.Key,
	source shared_hmac.Source,
) {
	var digest shared_hmac.Digest
	shared_hmac.Digest_Init(&digest, kind, key)
	shared_hmac.Digest_Write(&digest, source)
	verify_digest(t, &digest, kind, key, source)
}

func verify_digest(
	t *testing.T,
	digest *shared_hmac.Digest,
	kind shared_hmac.Kind,
	key shared_hmac.Key,
	source shared_hmac.Source,
) {
	value, count := shared_hmac.Digest_Sum(digest)
	want, want_count := standard_sum(kind, key, source)
	testify.Equal(t, want_count, count)
	testify.Equal(t, want[:want_count], value[:count])
}

func standard_sum(
	kind shared_hmac.Kind,
	key shared_hmac.Key,
	source shared_hmac.Source,
) (value shared_hmac.Value, count shared_hmac.Output_Count) {
	standard := standard_hmac.New(sha512.New, key)
	switch kind {
	case shared_hmac.KIND_MD5:
		standard = standard_hmac.New(md5.New, key)
	case shared_hmac.KIND_SHA_1:
		standard = standard_hmac.New(sha1.New, key)
	case shared_hmac.KIND_SHA_224:
		standard = standard_hmac.New(sha256.New224, key)
	case shared_hmac.KIND_SHA_256:
		standard = standard_hmac.New(sha256.New, key)
	case shared_hmac.KIND_SHA_384:
		standard = standard_hmac.New(sha512.New384, key)
	case shared_hmac.KIND_SHA_512_224:
		standard = standard_hmac.New(sha512.New512_224, key)
	case shared_hmac.KIND_SHA_512_256:
		standard = standard_hmac.New(sha512.New512_256, key)
	}
	standard.Write(source)
	tag := standard.Sum(nil)
	copy(value[:], tag)
	return value, shared_hmac.Output_Count(len(tag))
}

func test_digest_size(kind shared_hmac.Kind) (size int) {
	_, count := standard_sum(kind, nil, nil)
	return int(count)
}

func test_block_size(kind shared_hmac.Kind) (size int) {
	if kind <= shared_hmac.KIND_SHA_256 {
		return md5.BlockSize
	}
	return sha512.BlockSize
}

func test_key_and_kind_domains() {
	var key [shared_hmac.KEY_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		shared_hmac.KEY_SIZE_MINIMUM,
		shared_hmac.KEY_SIZE_MINIMUM + binary.UINT_8_SIZE,
		shared_hmac.KEY_SIZE_MINIMUM + binary.UINT_16_SIZE,
		shared_hmac.BLOCK_SIZE_MINIMUM + binary.UINT_8_SIZE,
		shared_hmac.KEY_SIZE_MAXIMUM,
	} {
		for _, kind := range test_kinds() {
			var digest shared_hmac.Digest
			shared_hmac.Digest_Init(&digest, kind, key[:size])
		}
	}
}

func test_source_domains() {
	var source [shared_hmac.SOURCE_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		shared_hmac.SOURCE_SIZE_MINIMUM,
		shared_hmac.SOURCE_SIZE_MINIMUM + binary.UINT_8_SIZE,
		shared_hmac.SOURCE_SIZE_MINIMUM + binary.UINT_16_SIZE,
		shared_hmac.SOURCE_SIZE_MAXIMUM,
	} {
		var digest shared_hmac.Digest
		shared_hmac.Digest_Init(&digest, shared_hmac.KIND_SHA_256, nil)
		shared_hmac.Digest_Write(&digest, source[:size])
	}
}

func test_destination_and_tag_domains() {
	var destination [shared_hmac.DESTINATION_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		shared_hmac.DESTINATION_SIZE_MINIMUM,
		shared_hmac.DESTINATION_SIZE_MINIMUM + binary.UINT_8_SIZE,
		shared_hmac.DESTINATION_SIZE_MINIMUM + binary.UINT_16_SIZE,
		shared_hmac.DESTINATION_SIZE_MAXIMUM,
	} {
		var digest shared_hmac.Digest
		shared_hmac.Digest_Init(&digest, shared_hmac.KIND_SHA_256, nil)
		shared_hmac.Digest_Sum_Into(&digest, destination[:size])
		shared_hmac.Equal(destination[:size], destination[:size])
	}
	shared_hmac.Equal(shared_hmac.Tag("same"), shared_hmac.Tag("different"))
}

func test_state_domains() {
	for _, kind := range test_kinds() {
		var digest shared_hmac.Digest
		shared_hmac.Digest_Init(&digest, kind, nil)
		shared_hmac.Digest_Init(&digest, kind, nil)
		shared_hmac.Digest_Reset(&digest)
		shared_hmac.Digest_Size(&digest)
		shared_hmac.Digest_Block_Size(&digest)
		shared_hmac.Digest_Sum(&digest)
		var clone shared_hmac.Digest
		shared_hmac.Digest_Clone_Into(&clone, &digest)
	}
}

func test_kind_allocation(t *testing.T, fixture *allocation_fixture) {
	testify.Zero_Allocation(t, func() {
		shared_hmac.Digest_Init(&fixture.Digest, fixture.Kind, fixture.Key)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Count = shared_hmac.Digest_Write(&fixture.Digest, fixture.Source)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Value, fixture.Output_Count = shared_hmac.Digest_Sum(&fixture.Digest)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Output_Count, fixture.Output_Status = shared_hmac.Digest_Sum_Into(
			&fixture.Digest, fixture.Destination[:],
		)
	})
	testify.Zero_Allocation(t, func() { shared_hmac.Digest_Reset(&fixture.Digest) })
	testify.Zero_Allocation(t, func() {
		shared_hmac.Digest_Clone_Into(&fixture.Clone, &fixture.Digest)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Size = shared_hmac.Digest_Size(&fixture.Digest)
		fixture.Block_Size = shared_hmac.Digest_Block_Size(&fixture.Digest)
	})
	testify.Zero_Allocation(t, func() {
		fixture.Equal = shared_hmac.Equal(fixture.Left, fixture.Left)
	})
}

type allocation_fixture struct {
	Digest        shared_hmac.Digest
	Clone         shared_hmac.Digest
	Destination   [shared_hmac.DESTINATION_SIZE_MAXIMUM]byte
	Key           shared_hmac.Key
	Source        shared_hmac.Source
	Left          shared_hmac.Tag
	Kind          shared_hmac.Kind
	Value         shared_hmac.Value
	Count         shared_hmac.Count
	Output_Count  shared_hmac.Output_Count
	Output_Status shared_hmac.Output_Status
	Size          shared_hmac.Size
	Block_Size    shared_hmac.Block_Size
	Equal         shared_hmac.Equality
}
