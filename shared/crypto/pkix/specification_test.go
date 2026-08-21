package pkix_test

import (
	"testing"

	"local/james-orcales/shared/crypto/pkix"
	"local/james-orcales/shared/encoding/asn1"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// Test_Algorithm_Identifier recognizes every supported public and signature OID.
func Test_Algorithm_Identifier(t *testing.T) {
	for _, fixture := range [...]struct {
		Encoded   []byte
		Algorithm pkix.Algorithm
	}{
		{rsa_identifier(), pkix.ALGORITHM_RSA},
		{rsa_sha_256_identifier(), pkix.ALGORITHM_RSA_SHA_256},
		{ec_identifier(), pkix.ALGORITHM_EC},
		{ecdsa_sha_256_identifier(), pkix.ALGORITHM_ECDSA_SHA_256},
		{ed25519_identifier(), pkix.ALGORITHM_ED25519},
	} {
		var identifier pkix.Algorithm_Identifier
		status := pkix.Parse_Algorithm_Identifier(&identifier, fixture.Encoded)
		testify.Equal(t, pkix.PARSE_STATUS_OK, status)
		testify.Equal(t, fixture.Algorithm, identifier.Algorithm)
	}
}

// Test_Name extracts a borrowed common name from one canonical RDN sequence.
func Test_Name(t *testing.T) {
	encoded := []byte{
		0x30, 0x12, 0x31, 0x10, 0x30, 0x0e, 0x06, 0x03,
		0x55, 0x04, 0x03, 0x0c, 0x07, 'e', 'x', 'a',
		'm', 'p', 'l', 'e',
	}
	var name pkix.Name
	status := pkix.Parse_Name(&name, encoded)
	testify.Equal(t, pkix.PARSE_STATUS_OK, status)
	testify.Equal(t, "example", string(name.Common_Name))
}

// Test_Bounds rejects oversized and bounded malformed DER transactionally.
func Test_Bounds(t *testing.T) {
	var identifier pkix.Algorithm_Identifier
	var oversized [pkix.ENCODED_SIZE_MAXIMUM + binary.UINT_8_SIZE]byte
	testify.Panics(t, func() {
		pkix.Parse_Algorithm_Identifier(&identifier, oversized[:])
	})
	before := identifier
	status := pkix.Parse_Algorithm_Identifier(&identifier, nil)
	testify.Equal(t, pkix.PARSE_STATUS_INPUT_INVALID, status)
	testify.Equal(t, before, identifier)
}

// Test_Allocation measures both successful parsers.
func Test_Allocation(t *testing.T) {
	algorithm := []byte{0x30, 0x05, 0x06, 0x03, 0x2b, 0x65, 0x70}
	name_encoding := []byte{
		0x30, 0x12, 0x31, 0x10, 0x30, 0x0e, 0x06, 0x03,
		0x55, 0x04, 0x03, 0x0c, 0x07, 'e', 'x', 'a',
		'm', 'p', 'l', 'e',
	}
	var identifier pkix.Algorithm_Identifier
	var name pkix.Name
	var status pkix.Parse_Status
	testify.Zero_Allocation(t, func() {
		status = pkix.Parse_Algorithm_Identifier(&identifier, algorithm)
	})
	testify.Zero_Allocation(t, func() {
		status = pkix.Parse_Name(&name, name_encoding)
	})
	testify.Equal(t, pkix.PARSE_STATUS_OK, status)
}

// Test_Invariant_Domains reaches every bounded input length and parser status.
func Test_Invariant_Domains(t *testing.T) {
	var identifier pkix.Algorithm_Identifier
	var name pkix.Name
	var input [pkix.ENCODED_SIZE_MAXIMUM]byte
	boundary_sizes := [...]int{
		pkix.ENCODED_SIZE_MINIMUM,
		pkix.ENCODED_SIZE_MINIMUM + binary.UINT_8_SIZE,
		pkix.ENCODED_SIZE_MINIMUM + binary.UINT_16_SIZE,
		pkix.ENCODED_SIZE_MAXIMUM - binary.UINT_8_SIZE,
		pkix.ENCODED_SIZE_MAXIMUM,
	}
	for _, size := range boundary_sizes {
		pkix.Parse_Algorithm_Identifier(&identifier, input[:size])
		pkix.Parse_Name(&name, input[:size])
	}
	for _, size := range boundary_sizes {
		identifier.Parameters = input[:size]
		identifier.Raw = input[:size]
		pkix.Parse_Algorithm_Identifier(&identifier, nil)
		name.Raw = input[:size]
		pkix.Parse_Name(&name, nil)
	}
	for algorithm := pkix.ALGORITHM_MINIMUM; algorithm <= pkix.ALGORITHM_MAXIMUM; algorithm++ {
		identifier.Algorithm = algorithm
		pkix.Parse_Algorithm_Identifier(&identifier, nil)
	}
	for _, size := range [...]int{
		pkix.ENCODED_SIZE_MINIMUM,
		pkix.ENCODED_SIZE_MINIMUM + binary.UINT_8_SIZE,
		pkix.ENCODED_SIZE_MINIMUM + binary.UINT_16_SIZE,
		pkix.COMMON_NAME_SIZE_MAXIMUM - binary.UINT_8_SIZE,
		pkix.COMMON_NAME_SIZE_MAXIMUM,
	} {
		name.Common_Name = input[:size]
		pkix.Parse_Name(&name, nil)
	}
	for _, size := range [...]int{
		pkix.ENCODED_SIZE_MINIMUM,
		pkix.ENCODED_SIZE_MINIMUM + binary.UINT_8_SIZE,
		pkix.ENCODED_SIZE_MINIMUM + binary.UINT_16_SIZE,
		pkix.OBJECT_IDENTIFIER_SIZE_MAXIMUM - binary.UINT_8_SIZE,
		pkix.OBJECT_IDENTIFIER_SIZE_MAXIMUM,
	} {
		encoded := algorithm_identifier_with_oid_size(input[:], size)
		pkix.Parse_Algorithm_Identifier(&identifier, encoded)
	}
	for _, size := range [...]int{
		pkix.ENCODED_SIZE_MINIMUM,
		pkix.ENCODED_SIZE_MINIMUM + binary.UINT_8_SIZE,
		pkix.ENCODED_SIZE_MINIMUM + binary.UINT_16_SIZE,
		asn1.CONTENT_SIZE_MAXIMUM,
	} {
		encoded := sequence_with_content_size(input[:], size)
		pkix.Parse_Name(&name, encoded)
	}
	for _, size := range [...]int{
		pkix.ENCODED_SIZE_MINIMUM,
		pkix.ENCODED_SIZE_MINIMUM + binary.UINT_8_SIZE,
		pkix.ENCODED_SIZE_MINIMUM + binary.UINT_16_SIZE,
		pkix.COMMON_NAME_SIZE_MAXIMUM - binary.UINT_8_SIZE,
		pkix.COMMON_NAME_SIZE_MAXIMUM,
	} {
		encoded := name_with_common_name_size(input[:], size)
		status := pkix.Parse_Name(&name, encoded)
		testify.Equal(t, pkix.PARSE_STATUS_OK, status)
	}
}

const DER_ELEMENT_HEADER_SIZE_MINIMUM = asn1.IDENTIFIER_SIZE_MINIMUM +
	asn1.CONTENT_SIZE_FIELD_SIZE_MINIMUM

const DER_SEQUENCE_IDENTIFIER byte = byte(asn1.CONSTRUCTED_MASK) |
	byte(bits.BIT_COUNT_16_MAXIMUM)

const DER_SET_IDENTIFIER byte = DER_SEQUENCE_IDENTIFIER + byte(binary.UINT_8_SIZE)

const DER_OID_IDENTIFIER byte = 0x06

const DER_UTF_8_STRING_IDENTIFIER byte = 0x0c

const COMMON_NAME_OID_ELEMENT_ENCODING = "\x06\x03\x55\x04\x03"

func algorithm_identifier_with_oid_size(
	output []byte, oid_size int,
) (encoded []byte) {
	sequence_content_size := DER_ELEMENT_HEADER_SIZE_MINIMUM + oid_size
	position := write_der_header(output, DER_SEQUENCE_IDENTIFIER, sequence_content_size)
	position += write_der_header(output[position:], DER_OID_IDENTIFIER, oid_size)
	for index := range oid_size {
		output[position+index] = bits.WORD_8_MINIMUM
	}
	return output[:position+oid_size]
}

func sequence_with_content_size(output []byte, content_size int) (encoded []byte) {
	position := write_der_header(output, DER_SEQUENCE_IDENTIFIER, content_size)
	for index := range content_size {
		output[position+index] = bits.WORD_8_MINIMUM
	}
	return output[:position+content_size]
}

func name_with_common_name_size(
	output []byte, common_name_size int,
) (encoded []byte) {
	oid_size := len(COMMON_NAME_OID_ELEMENT_ENCODING)
	value_size := der_header_size(common_name_size) + common_name_size
	attribute_content_size := oid_size + value_size
	attribute_size := der_header_size(attribute_content_size) + attribute_content_size
	set_size := der_header_size(attribute_size) + attribute_size
	position := write_der_header(output, DER_SEQUENCE_IDENTIFIER, set_size)
	position += write_der_header(output[position:], DER_SET_IDENTIFIER, attribute_size)
	position += write_der_header(
		output[position:], DER_SEQUENCE_IDENTIFIER, attribute_content_size,
	)
	copy(output[position:], COMMON_NAME_OID_ELEMENT_ENCODING)
	position += oid_size
	position += write_der_header(
		output[position:], DER_UTF_8_STRING_IDENTIFIER, common_name_size,
	)
	for index := range common_name_size {
		output[position+index] = 'a'
	}
	return output[:position+common_name_size]
}

func write_der_header(
	output []byte, tag byte, content_size int,
) (header_size int) {
	header_size = der_header_size(content_size)
	output[bits.BIT_COUNT_MINIMUM] = tag
	content_size_index := asn1.IDENTIFIER_SIZE_MINIMUM
	if header_size == DER_ELEMENT_HEADER_SIZE_MINIMUM {
		output[content_size_index] = byte(content_size)
		return header_size
	}
	content_size_octet_count := header_size - asn1.IDENTIFIER_SIZE_MINIMUM -
		asn1.CONTENT_SIZE_FIELD_SIZE_MINIMUM
	output[content_size_index] = byte(asn1.CONTINUATION_MASK | content_size_octet_count)
	content_size_value_index := content_size_index + asn1.CONTENT_SIZE_FIELD_SIZE_MINIMUM
	if header_size == asn1.IDENTIFIER_SIZE_MINIMUM+
		asn1.CONTENT_SIZE_FIELD_SIZE_MIDDLE {
		output[content_size_value_index] = byte(content_size)
		return header_size
	}
	output[content_size_value_index] = byte(content_size >> bits.BIT_COUNT_8_MAXIMUM)
	output[content_size_value_index+binary.UINT_8_SIZE] = byte(content_size)
	return header_size
}

func der_header_size(content_size int) (header_size int) {
	if content_size <= asn1.CONTENT_SIZE_SHORT_MAXIMUM {
		return asn1.IDENTIFIER_SIZE_MINIMUM +
			asn1.CONTENT_SIZE_FIELD_SIZE_MINIMUM
	}
	if content_size <= int(bits.WORD_8_MAXIMUM) {
		return asn1.IDENTIFIER_SIZE_MINIMUM +
			asn1.CONTENT_SIZE_FIELD_SIZE_MIDDLE
	}
	return asn1.IDENTIFIER_SIZE_MINIMUM + asn1.CONTENT_SIZE_FIELD_SIZE_MAXIMUM
}

func rsa_identifier() (encoded []byte) {
	return []byte{
		0x30, 0x0d, 0x06, 0x09, 0x2a, 0x86, 0x48, 0x86,
		0xf7, 0x0d, 0x01, 0x01, 0x01, 0x05, 0x00,
	}
}

func rsa_sha_256_identifier() (encoded []byte) {
	return []byte{
		0x30, 0x0d, 0x06, 0x09, 0x2a, 0x86, 0x48, 0x86,
		0xf7, 0x0d, 0x01, 0x01, 0x0b, 0x05, 0x00,
	}
}

func ec_identifier() (encoded []byte) {
	return []byte{
		0x30, 0x13, 0x06, 0x07, 0x2a, 0x86, 0x48, 0xce,
		0x3d, 0x02, 0x01, 0x06, 0x08, 0x2a, 0x86, 0x48,
		0xce, 0x3d, 0x03, 0x01, 0x07,
	}
}

func ecdsa_sha_256_identifier() (encoded []byte) {
	return []byte{
		0x30, 0x0a, 0x06, 0x08, 0x2a, 0x86,
		0x48, 0xce, 0x3d, 0x04, 0x03, 0x02,
	}
}

func ed25519_identifier() (encoded []byte) {
	return []byte{0x30, 0x05, 0x06, 0x03, 0x2b, 0x65, 0x70}
}
