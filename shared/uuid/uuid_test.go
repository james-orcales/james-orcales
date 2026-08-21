package uuid_test

import (
	"testing"

	"local/james-orcales/shared/crypto/prng"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/sim/time"
	"local/james-orcales/shared/uuid"
)

// TestMain register UUID invariant roots before specification run.
func TestMain(m *testing.M) {
	aver.Run_Test_Main(m)
}

// FIXED_EPOCH_SECONDS is the Unix second the fixed generator's virtual clock reads,
// so a Version 7 timestamp round-trips to a known value.
const FIXED_EPOCH_SECONDS = 1_700_000_000

// Builds a Generator whose entropy is a seeded CSPRNG and whose clock is a frozen
// virtual clock at FIXED_EPOCH_SECONDS, so every draw is reproducible.
func fixed_generator(seed uint64) (generator uuid.Generator) {
	return generator_at(
		seed, time.Moment(FIXED_EPOCH_SECONDS*int64(time.SECOND)),
		uuid.Node{
			High: 0x0102,
			Low:  0x03040506,
		},
		uuid.Generator_State{},
	)
}

// Ordinary tests own heap storage so each assertion can retain prior UUIDs.
func uuid_storage() (storage uuid.UUID) {
	return make(uuid.UUID, uuid.UUID_BYTE_COUNT)
}

func nil_uuid() (value uuid.UUID) {
	return uuid.Nil(uuid_storage())
}

func generated_v1(generator uuid.Generator_Pointer) (value uuid.UUID) {
	return uuid.Generator_V1(generator, uuid_storage())
}

func generated_v4(generator uuid.Generator_Pointer) (value uuid.UUID) {
	return uuid.Generator_V4(generator, uuid_storage())
}

func generated_v6(generator uuid.Generator_Pointer) (value uuid.UUID) {
	return uuid.Generator_V6(generator, uuid_storage())
}

func generated_v7(generator uuid.Generator_Pointer) (result uuid.UUID) {
	value, failure := uuid.Generator_V7(generator, uuid_storage())
	return uuid.Must(value, failure)
}

func generated_dce(
	generator uuid.Generator_Pointer, domain uuid.Domain, identifier uuid.Identifier,
) (value uuid.UUID) {
	return uuid.Generator_DCE_Security(generator, uuid_storage(), domain, identifier)
}

func parsed_uuid(
	input uuid.Text_Unvalidated,
) (value uuid.UUID, status uuid.Syntax_Status) {
	value = uuid_storage()
	return value, uuid.Parse(value, input)
}

func must_parsed_uuid(input uuid.Text_Unvalidated) (value uuid.UUID) {
	return uuid.Must_Parse(uuid_storage(), input)
}

func text_uuid(value uuid.UUID) (text string) {
	storage := make(uuid.Text_Bytes, uuid.UUID_TEXT_BYTE_COUNT)
	return string(uuid.UUID_String(value, storage))
}

func urn_uuid(value uuid.UUID) (urn string) {
	storage := make(uuid.URN_Bytes, uuid.UUID_URN_BYTE_COUNT)
	return string(uuid.UUID_URN(value, storage))
}

func v3_uuid(namespace uuid.UUID, name uuid.Name) (value uuid.UUID) {
	return uuid.V3(uuid_storage(), namespace, name)
}

func v5_uuid(namespace uuid.UUID, name uuid.Name) (value uuid.UUID) {
	return uuid.V5(uuid_storage(), namespace, name)
}

func dns_namespace() (value uuid.UUID) {
	return uuid.Name_Space_DNS(uuid_storage())
}

// Boundary clocks and replay state need the same deterministic entropy wiring as
// ordinary examples, or invariant witnesses would accidentally test another root.
func generator_at(
	seed uint64, moment time.Moment, node uuid.Node, state uuid.Generator_State,
) (generator uuid.Generator) {
	var seed_bytes [prng.KEY_BYTES]byte
	for index := range prng.WORD_BYTE_COUNT {
		seed_bytes[index] = byte(seed >> (index * prng.WORD_BYTE_COUNT))
	}
	source := new(prng.Chacha)
	prng.Chacha_Init(
		prng.Chacha_Handle(source), prng.Seed(seed_bytes[:]), prng.CURSOR_MIN,
	)
	virtual := time.Virtual_Clock{
		Resolution: time.MILLISECOND,
		Epoch:      moment,
	}
	clock := time.Virtual_Clock_To_Clock(&virtual)
	return uuid.New(prng.Chacha_To_Source(source), clock, node, state)
}
