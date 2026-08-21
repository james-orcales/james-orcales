// Package pkix implements bounded borrowed DER structures used by X.509.
package pkix

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/encoding/asn1"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
)

// ENCODED_SIZE_MINIMUM admits empty hostile DER input.
const ENCODED_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// ENCODED_SIZE_MAXIMUM follows the shared DER boundary.
const ENCODED_SIZE_MAXIMUM = asn1.ENCODED_SIZE_MAXIMUM

// TAG_OBJECT_IDENTIFIER is the DER universal OID tag.
const TAG_OBJECT_IDENTIFIER asn1.Tag = 6

// TAG_SEQUENCE is the DER universal sequence tag.
const TAG_SEQUENCE asn1.Tag = 16

// TAG_SET is the DER universal set tag.
const TAG_SET asn1.Tag = TAG_SEQUENCE + binary.UINT_8_SIZE

// RSA_OID_ENCODING is authoritative rsaEncryption OID content.
const RSA_OID_ENCODING = "\x2a\x86\x48\x86\xf7\x0d\x01\x01\x01"

// RSA_SHA_256_OID_ENCODING is authoritative sha256WithRSAEncryption OID content.
const RSA_SHA_256_OID_ENCODING = "\x2a\x86\x48\x86\xf7\x0d\x01\x01\x0b"

// EC_OID_ENCODING is authoritative id-ecPublicKey OID content.
const EC_OID_ENCODING = "\x2a\x86\x48\xce\x3d\x02\x01"

// ECDSA_SHA_256_OID_ENCODING is authoritative ecdsa-with-SHA256 OID content.
const ECDSA_SHA_256_OID_ENCODING = "\x2a\x86\x48\xce\x3d\x04\x03\x02"

// ED25519_OID_ENCODING is authoritative id-Ed25519 OID content.
const ED25519_OID_ENCODING = "\x2b\x65\x70"

// COMMON_NAME_OID_ENCODING is authoritative id-at-commonName OID content.
const COMMON_NAME_OID_ENCODING = "\x55\x04\x03"

// ALGORITHM_RSA identifies rsaEncryption.
const ALGORITHM_RSA Algorithm = Algorithm(bits.WORD_8_MINIMUM)

// ALGORITHM_RSA_SHA_256 identifies sha256WithRSAEncryption.
const ALGORITHM_RSA_SHA_256 Algorithm = ALGORITHM_RSA + binary.UINT_8_SIZE

// ALGORITHM_EC identifies id-ecPublicKey.
const ALGORITHM_EC Algorithm = ALGORITHM_RSA_SHA_256 + binary.UINT_8_SIZE

// ALGORITHM_ECDSA_SHA_256 identifies ecdsa-with-SHA256.
const ALGORITHM_ECDSA_SHA_256 Algorithm = ALGORITHM_EC + binary.UINT_8_SIZE

// ALGORITHM_ED25519 identifies the RFC 8410 Ed25519 OID.
const ALGORITHM_ED25519 Algorithm = ALGORITHM_ECDSA_SHA_256 + binary.UINT_8_SIZE

// ALGORITHM_MINIMUM is the first supported OID.
const ALGORITHM_MINIMUM = ALGORITHM_RSA

// ALGORITHM_MAXIMUM is the final supported OID.
const ALGORITHM_MAXIMUM = ALGORITHM_ED25519

// PARSE_STATUS_OK means a validated structure committed.
const PARSE_STATUS_OK Parse_Status = Parse_Status(bits.WORD_8_MINIMUM)

// PARSE_STATUS_INPUT_INVALID leaves destination storage unchanged.
const PARSE_STATUS_INPUT_INVALID Parse_Status = PARSE_STATUS_OK + binary.UINT_8_SIZE

// READY_EMPTY marks caller storage without a parsed structure.
const READY_EMPTY Ready = Ready(bits.WORD_8_MINIMUM)

// READY_COMPLETE marks a parsed structure.
const READY_COMPLETE Ready = READY_EMPTY + binary.UINT_8_SIZE

// OBJECT_IDENTIFIER_SIZE_MAXIMUM bounds every OID supported or inspected here.
const OBJECT_IDENTIFIER_SIZE_MAXIMUM = bits.BIT_COUNT_16_MAXIMUM

// COMMON_NAME_OID_CONTENT_SIZE is the three-octet id-at-commonName OID.
const COMMON_NAME_OID_CONTENT_SIZE = len(COMMON_NAME_OID_ENCODING)

// COMMON_NAME_OID_ENCODED_SIZE includes its one-octet tag and short length.
const COMMON_NAME_OID_ENCODED_SIZE = asn1.IDENTIFIER_SIZE_MINIMUM +
	asn1.CONTENT_SIZE_FIELD_SIZE_MINIMUM + COMMON_NAME_OID_CONTENT_SIZE

// COMMON_NAME_CONTAINER_COUNT covers Name, RDN, attribute, and value headers.
const COMMON_NAME_CONTAINER_COUNT = binary.UINT_8_SIZE + binary.UINT_8_SIZE +
	binary.UINT_8_SIZE + binary.UINT_8_SIZE

// COMMON_NAME_SIZE_MAXIMUM leaves every enclosing maximum DER header in bounds.
const COMMON_NAME_SIZE_MAXIMUM = ENCODED_SIZE_MAXIMUM -
	COMMON_NAME_CONTAINER_COUNT*(asn1.IDENTIFIER_SIZE_MINIMUM+
		asn1.CONTENT_SIZE_FIELD_SIZE_MAXIMUM) - COMMON_NAME_OID_ENCODED_SIZE

// RECOGNITION_TRUE marks supported OID content.
const RECOGNITION_TRUE Recognition = true

// Algorithm identifies one supported public-key or signature OID.
type Algorithm uint8

// Algorithm_Invariants covers the supported contiguous OID set.
func Algorithm_Invariants(value Algorithm, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(ALGORITHM_MINIMUM), uint8(ALGORITHM_MAXIMUM)).
		Ensure()
}

// Ready stores parsed-state identity without array indirection.
type Ready uint8

// Ready_Invariants admits only empty and complete parser states.
func Ready_Invariants(value Ready, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(READY_EMPTY), uint8(READY_COMPLETE)).
		Ensure()
}

// Recognition reports whether OID content names supported algorithm.
type Recognition bool

// Recognition_Invariants covers supported and unsupported OID content.
func Recognition_Invariants(value Recognition, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "OID content names supported algorithm.").
		Ensure()
}

// Borrowed is one bounded field that aliases encoded input.
type Borrowed []byte

// Borrowed_Invariants preserves the supported OID storage ceiling.
func Borrowed_Invariants(value Borrowed, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM, OBJECT_IDENTIFIER_SIZE_MAXIMUM,
		).
		Ensure()
}

// Algorithm_Parameters borrows schema-specific DER following an algorithm OID.
type Algorithm_Parameters []byte

// Algorithm_Parameters_Invariants bounds borrowed parameter DER.
func Algorithm_Parameters_Invariants(
	value Algorithm_Parameters, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Raw borrows one complete validated PKIX structure.
type Raw []byte

// Raw_Invariants bounds complete borrowed DER.
func Raw_Invariants(value Raw, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Common_Name borrows one selected RDN string value.
type Common_Name []byte

// Common_Name_Invariants bounds selected name text.
func Common_Name_Invariants(value Common_Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, COMMON_NAME_SIZE_MAXIMUM).
		Ensure()
}

// Encoded is bounded hostile DER input.
type Encoded []byte

// Encoded_Invariants bounds parsing before DER access.
func Encoded_Invariants(value Encoded, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), ENCODED_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Algorithm_Identifier borrows one recognized DER AlgorithmIdentifier.
type Algorithm_Identifier struct {
	// Algorithm is the recognized OID.
	Algorithm Algorithm
	// Parameters retains encoded parameter elements after the OID.
	Parameters Algorithm_Parameters
	// Raw retains the complete encoded sequence.
	Raw Raw
	// Ready proves complete DER and OID validation.
	Ready Ready
}

// Algorithm_Identifier_Invariants composes borrowed and parsed state.
func Algorithm_Identifier_Invariants(
	value Algorithm_Identifier, namespace aver.Namespace,
) {
	Algorithm_Invariants(value.Algorithm, namespace)
	Algorithm_Parameters_Invariants(value.Parameters, namespace)
	Raw_Invariants(value.Raw, namespace)
	Ready_Invariants(value.Ready, namespace)
}

// Algorithm_Identifier_Destination is nonnil caller-owned parsed storage.
type Algorithm_Identifier_Destination *Algorithm_Identifier

// Algorithm_Identifier_Destination_Invariants proves caller storage exists.
func Algorithm_Identifier_Destination_Invariants(
	value Algorithm_Identifier_Destination, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Algorithm_Identifier_Invariants(*value, namespace)
}

// Name borrows selected fields from one validated RDN sequence.
type Name struct {
	// Raw retains the complete encoded RDN sequence.
	Raw Raw
	// Common_Name borrows the first common-name value when present.
	Common_Name Common_Name
	// Ready proves complete RDN validation.
	Ready Ready
}

// Name_Invariants composes borrowed and parsed state.
func Name_Invariants(value Name, namespace aver.Namespace) {
	Raw_Invariants(value.Raw, namespace)
	Common_Name_Invariants(value.Common_Name, namespace)
	Ready_Invariants(value.Ready, namespace)
}

// Name_Destination is nonnil caller-owned parsed storage.
type Name_Destination *Name

// Name_Destination_Invariants proves caller storage exists.
func Name_Destination_Invariants(
	value Name_Destination, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Name_Invariants(*value, namespace)
}

// Parse_Status reports complete or refused DER input.
type Parse_Status uint8

// Parse_Status_Invariants covers both parser outcomes.
func Parse_Status_Invariants(value Parse_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(PARSE_STATUS_OK), uint8(PARSE_STATUS_INPUT_INVALID),
		).
		Ensure()
}

// Parse_Algorithm_Identifier recognizes an OID before transactional commit.
func Parse_Algorithm_Identifier(
	destination Algorithm_Identifier_Destination, source Encoded,
) (status Parse_Status) {
	defer func() { Parse_Status_Invariants(status, "Parse_Algorithm_Identifier.status") }()
	Algorithm_Identifier_Destination_Invariants(
		destination, "Parse_Algorithm_Identifier.destination",
	)
	Encoded_Invariants(source, "Parse_Algorithm_Identifier.source")
	defer func() {
		Algorithm_Identifier_Invariants(
			*destination, "Parse_Algorithm_Identifier.destination.output")
	}()
	sequence, consumed, _, decode_status := asn1.Decode(asn1.Encoded(source))
	if decode_status != asn1.STATUS_OK {
		return PARSE_STATUS_INPUT_INVALID
	}
	if int(consumed) != len(source) {
		return PARSE_STATUS_INPUT_INVALID
	}
	if sequence.Class != asn1.CLASS_UNIVERSAL {
		return PARSE_STATUS_INPUT_INVALID
	}
	if sequence.Tag != TAG_SEQUENCE {
		return PARSE_STATUS_INPUT_INVALID
	}
	if !bool(sequence.Constructed) {
		return PARSE_STATUS_INPUT_INVALID
	}
	oid, oid_consumed, _, oid_status := asn1.Decode(asn1.Encoded(sequence.Content))
	if oid_status != asn1.STATUS_OK {
		return PARSE_STATUS_INPUT_INVALID
	}
	if oid.Class != asn1.CLASS_UNIVERSAL {
		return PARSE_STATUS_INPUT_INVALID
	}
	if oid.Tag != TAG_OBJECT_IDENTIFIER {
		return PARSE_STATUS_INPUT_INVALID
	}
	if bool(oid.Constructed) {
		return PARSE_STATUS_INPUT_INVALID
	}
	algorithm, recognition := algorithm_from_oid(Borrowed(oid.Content))
	if !bool(recognition) {
		return PARSE_STATUS_INPUT_INVALID
	}
	parameters := sequence.Content[int(oid_consumed):]
	if len(parameters) > ENCODED_SIZE_MINIMUM {
		_, parameter_consumed, _, parameter_status := asn1.Decode(asn1.Encoded(parameters))
		if parameter_status != asn1.STATUS_OK {
			return PARSE_STATUS_INPUT_INVALID
		}
		if int(parameter_consumed) != len(parameters) {
			return PARSE_STATUS_INPUT_INVALID
		}
	}
	*destination = Algorithm_Identifier{
		Algorithm: algorithm, Parameters: Algorithm_Parameters(parameters),
		Raw:   Raw(source),
		Ready: READY_COMPLETE,
	}
	return PARSE_STATUS_OK
}

// Parse_Name validates each single-valued RDN before borrowing a common name.
func Parse_Name(destination Name_Destination, source Encoded) (status Parse_Status) {
	defer func() { Parse_Status_Invariants(status, "Parse_Name.status") }()
	Name_Destination_Invariants(destination, "Parse_Name.destination")
	Encoded_Invariants(source, "Parse_Name.source")
	defer func() { Name_Invariants(*destination, "Parse_Name.destination.output") }()
	sequence, consumed, _, decode_status := asn1.Decode(asn1.Encoded(source))
	if decode_status != asn1.STATUS_OK {
		return PARSE_STATUS_INPUT_INVALID
	}
	if int(consumed) != len(source) {
		return PARSE_STATUS_INPUT_INVALID
	}
	if sequence.Class != asn1.CLASS_UNIVERSAL {
		return PARSE_STATUS_INPUT_INVALID
	}
	if sequence.Tag != TAG_SEQUENCE {
		return PARSE_STATUS_INPUT_INVALID
	}
	if !bool(sequence.Constructed) {
		return PARSE_STATUS_INPUT_INVALID
	}
	common_name, content_status := parse_name_content(sequence.Content)
	if content_status != PARSE_STATUS_OK {
		return PARSE_STATUS_INPUT_INVALID
	}
	*destination = Name{
		Raw: Raw(source), Common_Name: common_name, Ready: READY_COMPLETE,
	}
	return PARSE_STATUS_OK
}

func parse_name_content(
	content asn1.Content,
) (common_name Common_Name, _ Parse_Status) {
	defer func() {
		Common_Name_Invariants(common_name, "parse_name_content.common_name")
	}()
	asn1.Content_Invariants(content, "parse_name_content.content")
	status := PARSE_STATUS_INPUT_INVALID
	defer func() { Parse_Status_Invariants(status, "parse_name_content.status") }()
	for len(content) > ENCODED_SIZE_MINIMUM {
		set, set_consumed, _, set_status := asn1.Decode(asn1.Encoded(content))
		if set_status != asn1.STATUS_OK {
			return common_name, status
		}
		if set.Class != asn1.CLASS_UNIVERSAL {
			return common_name, status
		}
		if set.Tag != TAG_SET {
			return common_name, status
		}
		if !bool(set.Constructed) {
			return common_name, status
		}
		attribute, attribute_consumed, _, attribute_status :=
			asn1.Decode(asn1.Encoded(set.Content))
		if attribute_status != asn1.STATUS_OK {
			return common_name, status
		}
		if int(attribute_consumed) != len(set.Content) {
			return common_name, status
		}
		if attribute.Class != asn1.CLASS_UNIVERSAL {
			return common_name, status
		}
		if attribute.Tag != TAG_SEQUENCE {
			return common_name, status
		}
		oid, oid_consumed, _, oid_status := asn1.Decode(asn1.Encoded(attribute.Content))
		if oid_status != asn1.STATUS_OK {
			return common_name, status
		}
		if oid.Tag != TAG_OBJECT_IDENTIFIER {
			return common_name, status
		}
		value_source := attribute.Content[int(oid_consumed):]
		value, value_consumed, _, value_status := asn1.Decode(asn1.Encoded(value_source))
		if value_status != asn1.STATUS_OK {
			return common_name, PARSE_STATUS_INPUT_INVALID
		}
		if int(value_consumed) != len(value_source) {
			return common_name, PARSE_STATUS_INPUT_INVALID
		}
		if string(oid.Content) == COMMON_NAME_OID_ENCODING {
			if common_name == nil {
				common_name = Common_Name(value.Content)
			}
		}
		content = content[int(set_consumed):]
	}
	status = PARSE_STATUS_OK
	return common_name, status
}

func algorithm_from_oid(
	oid Borrowed,
) (algorithm Algorithm, _ Recognition) {
	defer func() { Algorithm_Invariants(algorithm, "algorithm_from_oid.algorithm") }()
	Borrowed_Invariants(oid, "algorithm_from_oid.oid")
	var recognition Recognition
	defer func() {
		Recognition_Invariants(recognition, "algorithm_from_oid.recognition")
	}()
	for _, fixture := range [...]struct {
		OID       string
		Algorithm Algorithm
	}{
		{RSA_OID_ENCODING, ALGORITHM_RSA},
		{RSA_SHA_256_OID_ENCODING, ALGORITHM_RSA_SHA_256},
		{EC_OID_ENCODING, ALGORITHM_EC},
		{ECDSA_SHA_256_OID_ENCODING, ALGORITHM_ECDSA_SHA_256},
		{ED25519_OID_ENCODING, ALGORITHM_ED25519},
	} {
		if string(oid) == fixture.OID {
			algorithm = fixture.Algorithm
			recognition = RECOGNITION_TRUE
			return algorithm, recognition
		}
	}
	return algorithm, recognition
}
