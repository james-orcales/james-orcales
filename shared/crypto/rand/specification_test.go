package random_test

import (
	"testing"

	"local/james-orcales/shared/crypto/rand"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/random/csprng"
	"local/james-orcales/shared/testify"
)

// Test_Injected_Entropy proves output is a function of the caller-owned generator.
func Test_Injected_Entropy(t *testing.T) {
	seed := [csprng.KEY_BYTES]byte{TEST_SEED_BYTE}
	subject := csprng.New(seed, csprng.CURSOR_MIN)
	reference := csprng.New(seed, csprng.CURSOR_MIN)
	var got, want [TEST_OUTPUT_SIZE]byte
	count := random.Read(&subject, got[:])
	reference.Read(want[:])
	testify.Equal(t, random.Count(len(got)), count)
	testify.Equal(t, want, got)
}

// Test_Caller_Owned_Output checks empty, buffered, and refill-crossing requests.
func Test_Caller_Owned_Output(t *testing.T) {
	generator := csprng.New([csprng.KEY_BYTES]byte{}, csprng.CURSOR_MIN)
	var output [csprng.BUFFER_BYTES + binary.UINT_8_SIZE]byte
	for _, size := range [...]int{
		random.DESTINATION_SIZE_MINIMUM,
		random.DESTINATION_SIZE_MINIMUM + binary.UINT_8_SIZE,
		random.DESTINATION_SIZE_MINIMUM + binary.UINT_16_SIZE,
		csprng.BUFFER_BYTES,
		csprng.BUFFER_BYTES + binary.UINT_8_SIZE,
	} {
		count := random.Read(&generator, output[:size])
		testify.Equal(t, random.Count(size), count)
	}
}

// Test_Bounds rejects absent state and hostile output size before source use.
func Test_Bounds(t *testing.T) {
	testify.Panics(t, func() { random.Read(nil, nil) })
	generator := csprng.New([csprng.KEY_BYTES]byte{}, csprng.CURSOR_MIN)
	var oversized [random.DESTINATION_SIZE_MAXIMUM + binary.UINT_8_SIZE]byte
	testify.Panics(t, func() { random.Read(&generator, oversized[:]) })
}

// Test_Invariant_Domains reaches every output and count boundary through Read.
func Test_Invariant_Domains(t *testing.T) {
	generator := csprng.New([csprng.KEY_BYTES]byte{}, csprng.CURSOR_MIN)
	var output [random.DESTINATION_SIZE_MAXIMUM]byte
	for _, size := range [...]int{
		random.DESTINATION_SIZE_MINIMUM,
		random.DESTINATION_SIZE_MINIMUM + binary.UINT_8_SIZE,
		random.DESTINATION_SIZE_MINIMUM + binary.UINT_16_SIZE,
		random.DESTINATION_SIZE_MAXIMUM - binary.UINT_8_SIZE,
		random.DESTINATION_SIZE_MAXIMUM,
	} {
		random.Read(&generator, output[:size])
	}
	for _, position := range [...]csprng.Cursor{
		csprng.CURSOR_MIN,
		csprng.CURSOR_MIN + binary.UINT_8_SIZE,
		csprng.CURSOR_MIN + binary.UINT_16_SIZE,
		csprng.CURSOR_MAX,
	} {
		positioned := csprng.New([csprng.KEY_BYTES]byte{}, position)
		random.Read(&positioned, nil)
	}
}

// Test_Allocation measures the only exported runtime operation.
func Test_Allocation(t *testing.T) {
	generator := csprng.New([csprng.KEY_BYTES]byte{}, csprng.CURSOR_MIN)
	var output [TEST_OUTPUT_SIZE]byte
	var count random.Count
	testify.Zero_Allocation(t, func() {
		count = random.Read(&generator, output[:])
	})
	testify.Equal(t, random.Count(len(output)), count)
}

const TEST_OUTPUT_SIZE = csprng.WORD_BYTE_COUNT

const TEST_SEED_BYTE byte = bits.WORD_8_MAXIMUM >> binary.UINT_8_SIZE
