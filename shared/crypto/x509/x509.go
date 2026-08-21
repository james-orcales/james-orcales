// Package x509 implements bounded borrowed certificate parsing and verification.
package x509

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/crypto/ecdsa"
	"local/james-orcales/shared/crypto/ed25519"
	"local/james-orcales/shared/crypto/rsa"
	"local/james-orcales/shared/crypto/sha256"
	"local/james-orcales/shared/encoding/asn1"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
)

// ENCODED_SIZE_MINIMUM admits empty hostile certificate input.
const ENCODED_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// ENCODED_SIZE_MAXIMUM follows the shared DER boundary.
const ENCODED_SIZE_MAXIMUM = asn1.ENCODED_SIZE_MAXIMUM

// RSA_MODULUS_BIT_COUNT exposes the only RSA certificate-key width accepted here.
const RSA_MODULUS_BIT_COUNT = rsa.MODULUS_BIT_COUNT

// SIGNATURE_SIZE_MAXIMUM follows the widest supported certificate signature.
const SIGNATURE_SIZE_MAXIMUM = rsa.MODULUS_SIZE

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

// TAG_INTEGER is the DER universal integer tag.
const TAG_INTEGER asn1.Tag = 2

// TAG_BIT_STRING is the DER universal bit-string tag.
const TAG_BIT_STRING asn1.Tag = TAG_INTEGER + binary.UINT_8_SIZE

// TAG_OBJECT_IDENTIFIER is the DER universal OID tag.
const TAG_OBJECT_IDENTIFIER asn1.Tag = 6

// TAG_SEQUENCE is the DER universal sequence tag.
const TAG_SEQUENCE asn1.Tag = 16

// TAG_TIME_UTC is the DER UTC time tag.
const TAG_TIME_UTC asn1.Tag = 23

// TAG_TIME_GENERALIZED follows the UTC time tag.
const TAG_TIME_GENERALIZED asn1.Tag = TAG_TIME_UTC + binary.UINT_8_SIZE

// BIT_STRING_UNUSED_SIZE stores the required unused-bit count octet.
const BIT_STRING_UNUSED_SIZE = binary.UINT_8_SIZE

// BIT_STRING_UNUSED_EMPTY requires byte-aligned certificate keys and signatures.
const BIT_STRING_UNUSED_EMPTY byte = bits.WORD_8_MINIMUM

// CONSTRUCTED_FALSE requires a primitive DER element.
const CONSTRUCTED_FALSE uint32 = uint32(bits.WORD_8_MINIMUM)

// CONSTRUCTED_TRUE requires a constructed DER element.
const CONSTRUCTED_TRUE uint32 = CONSTRUCTED_FALSE + binary.UINT_8_SIZE

// VERSION_TAG is the explicit context-specific certificate version field.
const VERSION_TAG asn1.Tag = asn1.Tag(bits.WORD_32_MINIMUM)

// VERSION_V3 is the zero-based integer encoding of X.509 version 3.
const VERSION_V3 byte = byte(binary.UINT_8_SIZE + binary.UINT_8_SIZE)

// OPTIONAL_TAG_MINIMUM starts issuerUniqueID, subjectUniqueID, and extensions.
const OPTIONAL_TAG_MINIMUM asn1.Tag = VERSION_TAG + binary.UINT_8_SIZE

// OPTIONAL_TAG_MAXIMUM is the explicit extensions field.
const OPTIONAL_TAG_MAXIMUM asn1.Tag = OPTIONAL_TAG_MINIMUM + binary.UINT_16_SIZE

// SERIAL_NUMBER_SIZE_MAXIMUM is RFC 5280's positive serial-number octet ceiling.
const SERIAL_NUMBER_SIZE_MAXIMUM = 20

// RSA_EXPONENT_ENCODING is the canonical INTEGER content for exponent 65537.
const RSA_EXPONENT_ENCODING = "\x01\x00\x01"

// RSA_PARAMETERS_ENCODING requires rsaEncryption's canonical NULL parameters.
const RSA_PARAMETERS_ENCODING = "\x05\x00"

// RSA_PUBLIC_IDENTIFIER_ENCODING is rsaEncryption with required NULL parameters.
const RSA_PUBLIC_IDENTIFIER_ENCODING = "\x30\x0d\x06\x09\x2a\x86\x48\x86\xf7\x0d\x01\x01\x01" +
	RSA_PARAMETERS_ENCODING

// RSA_SIGNATURE_IDENTIFIER_ENCODING is sha256WithRSAEncryption with NULL parameters.
const RSA_SIGNATURE_IDENTIFIER_ENCODING = "\x30\x0d\x06\x09\x2a\x86\x48\x86\xf7\x0d\x01\x01\x0b" +
	RSA_PARAMETERS_ENCODING

// EC_PUBLIC_IDENTIFIER_ENCODING fixes id-ecPublicKey to named curve prime256v1.
const EC_PUBLIC_IDENTIFIER_ENCODING = "\x30\x13\x06\x07\x2a\x86\x48\xce\x3d\x02\x01" +
	"\x06\x08\x2a\x86\x48\xce\x3d\x03\x01\x07"

// ECDSA_SIGNATURE_IDENTIFIER_ENCODING is ecdsa-with-SHA256 without parameters.
const ECDSA_SIGNATURE_IDENTIFIER_ENCODING = "\x30\x0a\x06\x08\x2a\x86\x48\xce\x3d\x04\x03\x02"

// ED25519_IDENTIFIER_ENCODING is id-Ed25519 without parameters.
const ED25519_IDENTIFIER_ENCODING = "\x30\x05\x06\x03\x2b\x65\x70"

// COMMON_NAME_OID_ENCODING identifies id-at-commonName inside AttributeTypeAndValue.
const COMMON_NAME_OID_ENCODING = "\x55\x04\x03"

// PUBLIC_KEY_ALGORITHM_RSA identifies one fixed-width RSA public key.
const PUBLIC_KEY_ALGORITHM_RSA Public_Key_Algorithm = Public_Key_Algorithm(bits.WORD_8_MINIMUM)

// PUBLIC_KEY_ALGORITHM_ECDSA identifies one P-256 public key.
const PUBLIC_KEY_ALGORITHM_ECDSA Public_Key_Algorithm = (PUBLIC_KEY_ALGORITHM_RSA +
	binary.UINT_8_SIZE)

// PUBLIC_KEY_ALGORITHM_ED25519 identifies one Ed25519 public key.
const PUBLIC_KEY_ALGORITHM_ED25519 Public_Key_Algorithm = (PUBLIC_KEY_ALGORITHM_ECDSA +
	binary.UINT_8_SIZE)

// SIGNATURE_ALGORITHM_RSA_SHA_256 identifies PKCS1 v1.5 with SHA-256.
const SIGNATURE_ALGORITHM_RSA_SHA_256 Signature_Algorithm = Signature_Algorithm(bits.WORD_8_MINIMUM)

// SIGNATURE_ALGORITHM_ECDSA_SHA_256 identifies ECDSA with SHA-256.
const SIGNATURE_ALGORITHM_ECDSA_SHA_256 Signature_Algorithm = (SIGNATURE_ALGORITHM_RSA_SHA_256 +
	binary.UINT_8_SIZE)

// SIGNATURE_ALGORITHM_ED25519 identifies pure Ed25519.
const SIGNATURE_ALGORITHM_ED25519 Signature_Algorithm = (SIGNATURE_ALGORITHM_ECDSA_SHA_256 +
	binary.UINT_8_SIZE)

// PARSE_STATUS_OK means a complete certificate committed.
const PARSE_STATUS_OK Parse_Status = Parse_Status(bits.WORD_8_MINIMUM)

// PARSE_STATUS_INPUT_INVALID leaves caller certificate storage unchanged.
const PARSE_STATUS_INPUT_INVALID Parse_Status = PARSE_STATUS_OK + binary.UINT_8_SIZE

// READY_INDEX stores parsed-state identity.
const READY_INDEX = bits.BIT_COUNT_MINIMUM

// READY_WORD_COUNT holds one parsed-state octet.
const READY_WORD_COUNT = READY_INDEX + binary.UINT_8_SIZE

// READY_EMPTY marks caller storage without a parsed certificate.
const READY_EMPTY byte = bits.WORD_8_MINIMUM

// READY_COMPLETE marks a fully validated certificate.
const READY_COMPLETE byte = READY_EMPTY + binary.UINT_8_SIZE

// SPAN_START_INDEX stores an inclusive byte position.
const SPAN_START_INDEX = bits.BIT_COUNT_MINIMUM

// SPAN_END_INDEX stores an exclusive byte position.
const SPAN_END_INDEX = SPAN_START_INDEX + binary.UINT_8_SIZE

// SPAN_FIELD_COUNT stores both boundaries without a variable-size helper value.
const SPAN_FIELD_COUNT = SPAN_END_INDEX + binary.UINT_8_SIZE

// IDENTIFIER_CLASS_INDEX stores the DER class field.
const IDENTIFIER_CLASS_INDEX = bits.BIT_COUNT_MINIMUM

// IDENTIFIER_TAG_INDEX stores the DER tag field.
const IDENTIFIER_TAG_INDEX = IDENTIFIER_CLASS_INDEX + binary.UINT_8_SIZE

// IDENTIFIER_CONSTRUCTED_INDEX stores the DER construction bit.
const IDENTIFIER_CONSTRUCTED_INDEX = IDENTIFIER_TAG_INDEX + binary.UINT_8_SIZE

// IDENTIFIER_FIELD_COUNT stores the three scalar DER identifier fields.
const IDENTIFIER_FIELD_COUNT = IDENTIFIER_CONSTRUCTED_INDEX + binary.UINT_8_SIZE

// DECISION_INDEX selects one fixed parser decision word.
const DECISION_INDEX = bits.BIT_COUNT_MINIMUM

// DECISION_FALSE is zero parser acceptance.
const DECISION_FALSE uint64 = uint64(bits.WORD_64_MINIMUM)

// DECISION_TRUE follows false by one binary state.
const DECISION_TRUE uint64 = DECISION_FALSE + binary.UINT_8_SIZE

// Raw borrows one complete certificate.
type Raw []byte

// Raw_Invariants bounds complete borrowed certificate DER.
func Raw_Invariants(value Raw, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), ENCODED_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// TBS borrows one complete TBSCertificate element.
type TBS []byte

// TBS_Invariants bounds signed borrowed DER.
func TBS_Invariants(value TBS, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), ENCODED_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Signature borrows signature bytes after BIT STRING framing.
type Signature []byte

// Signature_Invariants bounds the widest supported signature encoding.
func Signature_Invariants(value Signature, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, SIGNATURE_SIZE_MAXIMUM).
		Ensure()
}

// Issuer_Raw borrows one validated issuer Name element.
type Issuer_Raw []byte

// Issuer_Raw_Invariants follows the complete DER boundary.
func Issuer_Raw_Invariants(value Issuer_Raw, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), ENCODED_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Issuer_Common_Name borrows issuer text from the certificate.
type Issuer_Common_Name []byte

// Issuer_Common_Name_Invariants follows the reachable PKIX name ceiling.
func Issuer_Common_Name_Invariants(
	value Issuer_Common_Name, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM, COMMON_NAME_SIZE_MAXIMUM,
		).
		Ensure()
}

// Issuer retains selected borrowed fields without duplicating Subject types.
type Issuer struct {
	// Raw is needed to preserve authenticated issuer identity.
	Raw Issuer_Raw
	// Common_Name is presentation data, never certificate identity.
	Common_Name Issuer_Common_Name
}

// Issuer_Invariants composes issuer-specific borrowed types.
func Issuer_Invariants(value Issuer, namespace aver.Namespace) {
	Issuer_Raw_Invariants(value.Raw, namespace)
	Issuer_Common_Name_Invariants(value.Common_Name, namespace)
}

// Subject_Raw borrows one validated subject Name element.
type Subject_Raw []byte

// Subject_Raw_Invariants follows the complete DER boundary.
func Subject_Raw_Invariants(value Subject_Raw, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), ENCODED_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Subject_Common_Name borrows subject text from the certificate.
type Subject_Common_Name []byte

// Subject_Common_Name_Invariants follows the reachable PKIX name ceiling.
func Subject_Common_Name_Invariants(
	value Subject_Common_Name, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM, COMMON_NAME_SIZE_MAXIMUM,
		).
		Ensure()
}

// Subject retains selected borrowed fields without conflating issuer identity.
type Subject struct {
	// Raw is needed to compare authenticated distinguished names.
	Raw Subject_Raw
	// Common_Name is presentation data, never certificate identity.
	Common_Name Subject_Common_Name
}

// Subject_Invariants composes subject-specific borrowed types.
func Subject_Invariants(value Subject, namespace aver.Namespace) {
	Subject_Raw_Invariants(value.Raw, namespace)
	Subject_Common_Name_Invariants(value.Common_Name, namespace)
}

// Public_Key_Algorithm selects the one ready public-key member.
type Public_Key_Algorithm uint8

// Public_Key_Algorithm_Invariants covers all supported certificate key types.
func Public_Key_Algorithm_Invariants(
	value Public_Key_Algorithm, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(PUBLIC_KEY_ALGORITHM_RSA),
			uint8(PUBLIC_KEY_ALGORITHM_ECDSA), uint8(PUBLIC_KEY_ALGORITHM_ED25519),
		).
		Ensure()
}

// Signature_Algorithm selects one supported certificate signature scheme.
type Signature_Algorithm uint8

// Signature_Algorithm_Invariants covers all supported certificate signatures.
func Signature_Algorithm_Invariants(
	value Signature_Algorithm, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(SIGNATURE_ALGORITHM_RSA_SHA_256),
			uint8(SIGNATURE_ALGORITHM_ECDSA_SHA_256),
			uint8(SIGNATURE_ALGORITHM_ED25519),
		).
		Ensure()
}

// Ready stores parsed-state identity.
type Ready [READY_WORD_COUNT]byte

// Ready_Invariants fixes certificate-state storage width.
func Ready_Invariants(value Ready, _ aver.Namespace) {
	aver.Always(len(value) == READY_WORD_COUNT, "X.509 state has fixed width.")
}

// Certificate holds fixed keys and borrowed authenticated fields.
type Certificate struct {
	// Raw is borrowed because copying the complete bounded certificate is unnecessary.
	Raw Raw
	// TBS is borrowed because signature verification hashes the authenticated encoding.
	TBS TBS
	// Signature is borrowed because verification consumes it synchronously.
	Signature Signature
	// Signature_Algorithm prevents OID interpretation at every verification call.
	Signature_Algorithm Signature_Algorithm
	// Public_Key_Algorithm selects exactly one fixed key representation.
	Public_Key_Algorithm Public_Key_Algorithm
	// Issuer keeps its types distinct from Subject inside one invariant chain.
	Issuer Issuer
	// Subject keeps its types distinct from Issuer inside one invariant chain.
	Subject Subject
	// RSA_Public_Key stores the supported RSA union member without an interface.
	RSA_Public_Key rsa.Public_Key
	// ECDSA_Public_Key stores the supported P-256 union member without an interface.
	ECDSA_Public_Key ecdsa.Public_Key
	// Ed25519_Public_Key stores the supported Edwards union member without an interface.
	Ed25519_Public_Key ed25519.Public_Key
	// Ready prevents partially parsed candidates from becoming verification authority.
	Ready Ready
}

// Certificate_Invariants composes every owned and borrowed certificate field.
func Certificate_Invariants(value Certificate, namespace aver.Namespace) {
	Raw_Invariants(value.Raw, namespace)
	TBS_Invariants(value.TBS, namespace)
	Signature_Invariants(value.Signature, namespace)
	Signature_Algorithm_Invariants(value.Signature_Algorithm, namespace)
	Public_Key_Algorithm_Invariants(value.Public_Key_Algorithm, namespace)
	Issuer_Invariants(value.Issuer, namespace)
	Subject_Invariants(value.Subject, namespace)
	rsa.Public_Key_Invariants(value.RSA_Public_Key, namespace)
	ecdsa.Public_Key_Invariants(value.ECDSA_Public_Key, namespace)
	ed25519.Public_Key_Invariants(value.Ed25519_Public_Key, namespace)
	Ready_Invariants(value.Ready, namespace)
	aver.Always(
		value.Ready[READY_INDEX] <= READY_COMPLETE,
		"An X.509 certificate has empty or complete state.",
	)
}

// Parsed_Certificate keeps byte positions internal until one full parse commits.
type Parsed_Certificate struct {
	// TBS maps authenticated bytes back to caller input.
	TBS [SPAN_FIELD_COUNT]int
	// Signature maps BIT STRING content back to caller input.
	Signature [SPAN_FIELD_COUNT]int
	// Issuer_Raw maps the validated issuer Name back to caller input.
	Issuer_Raw [SPAN_FIELD_COUNT]int
	// Issuer_Common_Name maps selected issuer text back to caller input.
	Issuer_Common_Name [SPAN_FIELD_COUNT]int
	// Subject_Raw maps the validated subject Name back to caller input.
	Subject_Raw [SPAN_FIELD_COUNT]int
	// Subject_Common_Name maps selected subject text back to caller input.
	Subject_Common_Name [SPAN_FIELD_COUNT]int
	// Public_Key_Algorithm selects one fixed key member.
	Public_Key_Algorithm Public_Key_Algorithm
	// RSA_Public_Key stores the RSA union member.
	RSA_Public_Key rsa.Public_Key
	// ECDSA_Public_Key stores the P-256 union member.
	ECDSA_Public_Key ecdsa.Public_Key
	// Ed25519_Public_Key stores the Edwards union member.
	Ed25519_Public_Key ed25519.Public_Key
}

// Parsed_Certificate_Invariants composes fixed spans and fixed key storage.
func Parsed_Certificate_Invariants(
	value Parsed_Certificate, namespace aver.Namespace,
) {
	aver.Always(
		len(value.TBS) == SPAN_FIELD_COUNT,
		"Parsed TBS span has both boundaries.",
	)
	aver.Always(
		len(value.Signature) == SPAN_FIELD_COUNT,
		"Parsed signature span has both boundaries.",
	)
	aver.Always(
		len(value.Issuer_Raw) == SPAN_FIELD_COUNT,
		"Parsed issuer span has both boundaries.",
	)
	aver.Always(
		len(value.Issuer_Common_Name) == SPAN_FIELD_COUNT,
		"Parsed issuer common-name span has both boundaries.",
	)
	aver.Always(
		len(value.Subject_Raw) == SPAN_FIELD_COUNT,
		"Parsed subject span has both boundaries.",
	)
	aver.Always(
		len(value.Subject_Common_Name) == SPAN_FIELD_COUNT,
		"Parsed subject common-name span has both boundaries.",
	)
	Public_Key_Algorithm_Invariants(value.Public_Key_Algorithm, namespace)
	rsa.Public_Key_Invariants(value.RSA_Public_Key, namespace)
	ecdsa.Public_Key_Invariants(value.ECDSA_Public_Key, namespace)
	ed25519.Public_Key_Invariants(value.Ed25519_Public_Key, namespace)
}

// Certificate_Destination is nonnil caller-owned parsed storage.
type Certificate_Destination *Certificate

// Certificate_Destination_Invariants proves caller storage exists.
func Certificate_Destination_Invariants(
	value Certificate_Destination, namespace aver.Namespace,
) {
	aver.Always(value != nil, "An X.509 certificate destination exists.")
	aver.Tree(value, namespace).
		Range_Int(len(value.Raw), ENCODED_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Range_Int(len(value.TBS), ENCODED_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Range_Int(
			len(value.Signature), bytes.SLICE_SIZE_MINIMUM, SIGNATURE_SIZE_MAXIMUM,
		).
		Enum_3_Uint8(
			uint8(value.Signature_Algorithm),
			uint8(SIGNATURE_ALGORITHM_RSA_SHA_256),
			uint8(SIGNATURE_ALGORITHM_ECDSA_SHA_256),
			uint8(SIGNATURE_ALGORITHM_ED25519),
		).
		Enum_3_Uint8(
			uint8(value.Public_Key_Algorithm), uint8(PUBLIC_KEY_ALGORITHM_RSA),
			uint8(PUBLIC_KEY_ALGORITHM_ECDSA), uint8(PUBLIC_KEY_ALGORITHM_ED25519),
		).
		Ensure()
	Issuer_Invariants(value.Issuer, namespace)
	Subject_Invariants(value.Subject, namespace)
	rsa.Public_Key_Invariants(value.RSA_Public_Key, namespace)
	ecdsa.Public_Key_Invariants(value.ECDSA_Public_Key, namespace)
	ed25519.Public_Key_Invariants(value.Ed25519_Public_Key, namespace)
	aver.Always(
		len(value.Ready) == READY_WORD_COUNT,
		"An X.509 certificate destination has state storage.",
	)
}

// Certificate_Handle is one nonnil parsed certificate authority.
type Certificate_Handle *Certificate

// Certificate_Handle_Invariants requires a complete parsed certificate.
func Certificate_Handle_Invariants(
	value Certificate_Handle, namespace aver.Namespace,
) {
	aver.Always(value != nil, "An X.509 certificate handle exists.")
	aver.Tree(value, namespace).
		Range_Int(len(value.Raw), ENCODED_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Range_Int(len(value.TBS), ENCODED_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Range_Int(
			len(value.Signature), bytes.SLICE_SIZE_MINIMUM, SIGNATURE_SIZE_MAXIMUM,
		).
		Enum_3_Uint8(
			uint8(value.Signature_Algorithm),
			uint8(SIGNATURE_ALGORITHM_RSA_SHA_256),
			uint8(SIGNATURE_ALGORITHM_ECDSA_SHA_256),
			uint8(SIGNATURE_ALGORITHM_ED25519),
		).
		Enum_3_Uint8(
			uint8(value.Public_Key_Algorithm), uint8(PUBLIC_KEY_ALGORITHM_RSA),
			uint8(PUBLIC_KEY_ALGORITHM_ECDSA), uint8(PUBLIC_KEY_ALGORITHM_ED25519),
		).
		Ensure()
	Issuer_Invariants(value.Issuer, namespace)
	Subject_Invariants(value.Subject, namespace)
	rsa.Public_Key_Invariants(value.RSA_Public_Key, namespace)
	ecdsa.Public_Key_Invariants(value.ECDSA_Public_Key, namespace)
	ed25519.Public_Key_Invariants(value.Ed25519_Public_Key, namespace)
	aver.Always(
		len(value.Ready) == READY_WORD_COUNT,
		"An X.509 certificate handle has state storage.",
	)
	aver.Always(
		value.Ready[READY_INDEX] == READY_COMPLETE,
		"An X.509 verification handle is completely parsed.",
	)
}

// Encoded is bounded hostile DER input.
type Encoded []byte

// Encoded_Invariants bounds parsing before DER access.
func Encoded_Invariants(value Encoded, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), ENCODED_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Parse_Status reports complete or refused certificate input.
type Parse_Status uint8

// Parse_Status_Invariants covers both parser outcomes.
func Parse_Status_Invariants(value Parse_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(PARSE_STATUS_OK), uint8(PARSE_STATUS_INPUT_INVALID),
		).
		Ensure()
}

// Verification reports signature validity without an owned error.
type Verification bool

// Verification_Invariants covers accepted and refused signatures.
func Verification_Invariants(value Verification, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "An X.509 certificate signature verifies.").
		Ensure()
}

// Parse_Certificate commits only one completely validated certificate.
func Parse_Certificate(
	destination Certificate_Destination, source Encoded,
) (status Parse_Status) {
	defer func() {
		Parse_Status_Invariants(status, "Parse_Certificate.status")
		Certificate_Invariants(*destination, "Parse_Certificate.destination.output")
	}()
	Certificate_Destination_Invariants(destination, "Parse_Certificate.destination")
	Encoded_Invariants(source, "Parse_Certificate.source")
	if len(source) > ENCODED_SIZE_MAXIMUM {
		panic("x509: certificate exceeds bound")
	}
	var storage [ENCODED_SIZE_MAXIMUM]byte
	copy(storage[:], source)
	source_span := [SPAN_FIELD_COUNT]int{SPAN_START_INDEX: ENCODED_SIZE_MINIMUM}
	source_span[SPAN_END_INDEX] = len(source)
	outer_identifier, outer_content, outer_encoded, tail, accepted :=
		take_element(&storage, source_span)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return PARSE_STATUS_INPUT_INVALID
	}
	if outer_encoded != source_span {
		return PARSE_STATUS_INPUT_INVALID
	}
	if tail[SPAN_START_INDEX] != tail[SPAN_END_INDEX] {
		return PARSE_STATUS_INPUT_INVALID
	}
	sequence := sequence_identifier()
	if identifier_matches(outer_identifier, sequence)[DECISION_INDEX] != DECISION_TRUE {
		return PARSE_STATUS_INPUT_INVALID
	}
	parsed, signature_algorithm, accepted := parse_certificate_content(
		&storage, outer_content,
	)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return PARSE_STATUS_INPUT_INVALID
	}
	start, end := parsed.TBS[SPAN_START_INDEX], parsed.TBS[SPAN_END_INDEX]
	tbs := source[start:end]
	start, end = parsed.Signature[SPAN_START_INDEX], parsed.Signature[SPAN_END_INDEX]
	signature := source[start:end]
	start, end = parsed.Issuer_Raw[SPAN_START_INDEX], parsed.Issuer_Raw[SPAN_END_INDEX]
	issuer_raw := source[start:end]
	start = parsed.Issuer_Common_Name[SPAN_START_INDEX]
	end = parsed.Issuer_Common_Name[SPAN_END_INDEX]
	issuer_common_name := source[start:end]
	start, end = parsed.Subject_Raw[SPAN_START_INDEX], parsed.Subject_Raw[SPAN_END_INDEX]
	subject_raw := source[start:end]
	start = parsed.Subject_Common_Name[SPAN_START_INDEX]
	end = parsed.Subject_Common_Name[SPAN_END_INDEX]
	subject_common_name := source[start:end]
	*destination = Certificate{
		Raw:                  Raw(source),
		TBS:                  TBS(tbs),
		Signature:            Signature(signature),
		Signature_Algorithm:  signature_algorithm,
		Public_Key_Algorithm: parsed.Public_Key_Algorithm,
		Issuer: Issuer{
			Raw:         Issuer_Raw(issuer_raw),
			Common_Name: Issuer_Common_Name(issuer_common_name),
		},
		Subject: Subject{
			Raw:         Subject_Raw(subject_raw),
			Common_Name: Subject_Common_Name(subject_common_name),
		},
		RSA_Public_Key:     parsed.RSA_Public_Key,
		ECDSA_Public_Key:   parsed.ECDSA_Public_Key,
		Ed25519_Public_Key: parsed.Ed25519_Public_Key,
		Ready:              Ready{READY_COMPLETE},
	}
	return PARSE_STATUS_OK
}

func parse_certificate_content(
	storage *[ENCODED_SIZE_MAXIMUM]byte, content [SPAN_FIELD_COUNT]int,
) (
	parsed Parsed_Certificate,
	signature_algorithm Signature_Algorithm,
	accepted [binary.UINT_8_SIZE]uint64,
) {
	defer func() {
		Parsed_Certificate_Invariants(parsed, "parse_certificate_content.parsed")
		Signature_Algorithm_Invariants(
			signature_algorithm, "parse_certificate_content.signature_algorithm",
		)
	}()
	tbs_identifier, _, tbs_span, tail, accepted := take_element(storage, content)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return parsed, signature_algorithm, accepted
	}
	_, _, algorithm_span, tail, accepted := take_element(storage, tail)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return parsed, signature_algorithm, accepted
	}
	signature_identifier, signature_content, _, tail, accepted :=
		take_element(storage, tail)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return parsed, signature_algorithm, accepted
	}
	if tail[SPAN_START_INDEX] != tail[SPAN_END_INDEX] {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return parsed, signature_algorithm, accepted
	}
	if identifier_matches(
		tbs_identifier, sequence_identifier(),
	)[DECISION_INDEX] != DECISION_TRUE {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return parsed, signature_algorithm, accepted
	}
	signature_algorithm, accepted = signature_algorithm_parse(
		storage, algorithm_span,
	)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return parsed, signature_algorithm, accepted
	}
	parsed.Signature, accepted = bit_string_content(
		storage, signature_identifier, signature_content,
	)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return parsed, signature_algorithm, accepted
	}
	if parsed.Signature[SPAN_END_INDEX]-parsed.Signature[SPAN_START_INDEX] >
		SIGNATURE_SIZE_MAXIMUM {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return parsed, signature_algorithm, accepted
	}
	tbs_parsed, accepted := parse_tbs(storage, tbs_span, algorithm_span)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return parsed, signature_algorithm, accepted
	}
	parsed.TBS = tbs_span
	parsed.Issuer_Raw = tbs_parsed.Issuer_Raw
	parsed.Issuer_Common_Name = tbs_parsed.Issuer_Common_Name
	parsed.Subject_Raw = tbs_parsed.Subject_Raw
	parsed.Subject_Common_Name = tbs_parsed.Subject_Common_Name
	parsed.Public_Key_Algorithm = tbs_parsed.Public_Key_Algorithm
	parsed.RSA_Public_Key = tbs_parsed.RSA_Public_Key
	parsed.ECDSA_Public_Key = tbs_parsed.ECDSA_Public_Key
	parsed.Ed25519_Public_Key = tbs_parsed.Ed25519_Public_Key
	return parsed, signature_algorithm, accepted
}

func parse_tbs(
	storage *[ENCODED_SIZE_MAXIMUM]byte,
	source [SPAN_FIELD_COUNT]int,
	outer_algorithm [SPAN_FIELD_COUNT]int,
) (parsed Parsed_Certificate, accepted [binary.UINT_8_SIZE]uint64) {
	defer func() { Parsed_Certificate_Invariants(parsed, "parse_tbs.parsed") }()
	identifier, content, encoded, tail, accepted := take_element(storage, source)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return parsed, accepted
	}
	if encoded != source {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return parsed, accepted
	}
	if tail[SPAN_START_INDEX] != tail[SPAN_END_INDEX] {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return parsed, accepted
	}
	if identifier_matches(
		identifier, sequence_identifier(),
	)[DECISION_INDEX] != DECISION_TRUE {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return parsed, accepted
	}
	serial_identifier, serial_content, _, content, accepted :=
		take_element(storage, content)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return parsed, accepted
	}
	if serial_identifier[IDENTIFIER_CLASS_INDEX] == asn1.CLASS_CONTEXT_SPECIFIC {
		if version_valid(
			storage, serial_identifier, serial_content,
		)[DECISION_INDEX] != DECISION_TRUE {
			return parsed, accepted
		}
		serial_identifier, serial_content, _, content, accepted =
			take_element(storage, content)
		if accepted[DECISION_INDEX] != DECISION_TRUE {
			return parsed, accepted
		}
	}
	if serial_valid(
		storage, serial_identifier, serial_content,
	)[DECISION_INDEX] != DECISION_TRUE {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return parsed, accepted
	}
	return parse_tbs_fields(storage, content, outer_algorithm)
}

func parse_tbs_fields(
	storage *[ENCODED_SIZE_MAXIMUM]byte,
	content [SPAN_FIELD_COUNT]int,
	outer_algorithm [SPAN_FIELD_COUNT]int,
) (parsed Parsed_Certificate, accepted [binary.UINT_8_SIZE]uint64) {
	defer func() { Parsed_Certificate_Invariants(parsed, "parse_tbs_fields.parsed") }()
	_, _, algorithm, content, accepted := take_element(storage, content)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return parsed, accepted
	}
	algorithm_bytes := storage[algorithm[SPAN_START_INDEX]:algorithm[SPAN_END_INDEX]]
	outer_bytes := storage[outer_algorithm[SPAN_START_INDEX]:outer_algorithm[SPAN_END_INDEX]]
	if !bool(bytes.Equal(bytes.Slice(algorithm_bytes), bytes.Slice(outer_bytes))) {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return parsed, accepted
	}
	_, _, parsed.Issuer_Raw, content, accepted = take_element(storage, content)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return parsed, accepted
	}
	parsed.Issuer_Common_Name, accepted = parse_name(storage, parsed.Issuer_Raw)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return parsed, accepted
	}
	validity_identifier, validity_content, _, content, accepted :=
		take_element(storage, content)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return parsed, accepted
	}
	if validity_valid(
		storage, validity_identifier, validity_content,
	)[DECISION_INDEX] != DECISION_TRUE {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return parsed, accepted
	}
	_, _, parsed.Subject_Raw, content, accepted = take_element(storage, content)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return parsed, accepted
	}
	parsed.Subject_Common_Name, accepted = parse_name(storage, parsed.Subject_Raw)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return parsed, accepted
	}
	_, _, public_key, content, accepted := take_element(storage, content)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return parsed, accepted
	}
	parsed.Public_Key_Algorithm, parsed.RSA_Public_Key,
		parsed.ECDSA_Public_Key, parsed.Ed25519_Public_Key,
		accepted = parse_public_key(storage, public_key)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return parsed, accepted
	}
	accepted = optional_fields_valid(storage, content)
	return parsed, accepted
}

func parse_public_key(
	storage *[ENCODED_SIZE_MAXIMUM]byte, source [SPAN_FIELD_COUNT]int,
) (
	algorithm Public_Key_Algorithm,
	rsa_key rsa.Public_Key,
	ecdsa_key ecdsa.Public_Key,
	ed25519_key ed25519.Public_Key,
	accepted [binary.UINT_8_SIZE]uint64,
) {
	defer func() {
		Public_Key_Algorithm_Invariants(algorithm, "parse_public_key.algorithm")
		rsa.Public_Key_Invariants(rsa_key, "parse_public_key.rsa_key")
		ecdsa.Public_Key_Invariants(ecdsa_key, "parse_public_key.ecdsa_key")
		ed25519.Public_Key_Invariants(ed25519_key, "parse_public_key.ed25519_key")
	}()
	identifier, content, encoded, tail, accepted := take_element(storage, source)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return algorithm, rsa_key, ecdsa_key, ed25519_key, accepted
	}
	if encoded != source {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return algorithm, rsa_key, ecdsa_key, ed25519_key, accepted
	}
	if tail[SPAN_START_INDEX] != tail[SPAN_END_INDEX] {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return algorithm, rsa_key, ecdsa_key, ed25519_key, accepted
	}
	if identifier_matches(
		identifier, sequence_identifier(),
	)[DECISION_INDEX] != DECISION_TRUE {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return algorithm, rsa_key, ecdsa_key, ed25519_key, accepted
	}
	_, _, algorithm_span, content, accepted := take_element(storage, content)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return algorithm, rsa_key, ecdsa_key, ed25519_key, accepted
	}
	key_identifier, key_content, _, content, accepted := take_element(storage, content)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return algorithm, rsa_key, ecdsa_key, ed25519_key, accepted
	}
	if content[SPAN_START_INDEX] != content[SPAN_END_INDEX] {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return algorithm, rsa_key, ecdsa_key, ed25519_key, accepted
	}
	key, accepted := bit_string_content(storage, key_identifier, key_content)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return algorithm, rsa_key, ecdsa_key, ed25519_key, accepted
	}
	return public_key_parse(storage, algorithm_span, key)
}

func public_key_parse(
	storage *[ENCODED_SIZE_MAXIMUM]byte,
	algorithm [SPAN_FIELD_COUNT]int,
	key [SPAN_FIELD_COUNT]int,
) (
	result Public_Key_Algorithm,
	rsa_key rsa.Public_Key,
	ecdsa_key ecdsa.Public_Key,
	ed25519_key ed25519.Public_Key,
	accepted [binary.UINT_8_SIZE]uint64,
) {
	defer func() {
		Public_Key_Algorithm_Invariants(result, "public_key_parse.result")
		rsa.Public_Key_Invariants(rsa_key, "public_key_parse.rsa_key")
		ecdsa.Public_Key_Invariants(ecdsa_key, "public_key_parse.ecdsa_key")
		ed25519.Public_Key_Invariants(ed25519_key, "public_key_parse.ed25519_key")
	}()
	algorithm_bytes := storage[algorithm[SPAN_START_INDEX]:algorithm[SPAN_END_INDEX]]
	key_bytes := storage[key[SPAN_START_INDEX]:key[SPAN_END_INDEX]]
	if string(algorithm_bytes) == RSA_PUBLIC_IDENTIFIER_ENCODING {
		rsa_key, accepted = rsa_public_key_parse(storage, key)
		if accepted[DECISION_INDEX] != DECISION_TRUE {
			return result, rsa_key, ecdsa_key, ed25519_key, accepted
		}
		result = PUBLIC_KEY_ALGORITHM_RSA
		return result, rsa_key, ecdsa_key, ed25519_key, accepted
	}
	if string(algorithm_bytes) == EC_PUBLIC_IDENTIFIER_ENCODING {
		if ecdsa.Public_Key_Set_Bytes(
			&ecdsa_key, ecdsa.Public_Key_Unvalidated(key_bytes),
		) !=
			ecdsa.KEY_STATUS_OK {
			return result, rsa_key, ecdsa_key, ed25519_key, accepted
		}
		result = PUBLIC_KEY_ALGORITHM_ECDSA
		accepted[DECISION_INDEX] = DECISION_TRUE
		return result, rsa_key, ecdsa_key, ed25519_key, accepted
	}
	if string(algorithm_bytes) == ED25519_IDENTIFIER_ENCODING {
		if len(key_bytes) != ed25519.PUBLIC_KEY_SIZE {
			return result, rsa_key, ecdsa_key, ed25519_key, accepted
		}
		copy(ed25519_key[:], key_bytes)
		result = PUBLIC_KEY_ALGORITHM_ED25519
		accepted[DECISION_INDEX] = DECISION_TRUE
	}
	return result, rsa_key, ecdsa_key, ed25519_key, accepted
}

func rsa_public_key_parse(
	storage *[ENCODED_SIZE_MAXIMUM]byte, source [SPAN_FIELD_COUNT]int,
) (
	public_key rsa.Public_Key, accepted [binary.UINT_8_SIZE]uint64,
) {
	defer func() {
		rsa.Public_Key_Invariants(public_key, "rsa_public_key_parse.public_key")
	}()
	identifier, content, encoded, tail, accepted := take_element(storage, source)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return public_key, accepted
	}
	if encoded != source {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return public_key, accepted
	}
	if tail[SPAN_START_INDEX] != tail[SPAN_END_INDEX] {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return public_key, accepted
	}
	if identifier_matches(
		identifier, sequence_identifier(),
	)[DECISION_INDEX] != DECISION_TRUE {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return public_key, accepted
	}
	modulus_identifier, modulus_content, _, content, accepted :=
		take_element(storage, content)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return public_key, accepted
	}
	exponent_identifier, exponent_content, _, content, accepted :=
		take_element(storage, content)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return public_key, accepted
	}
	if content[SPAN_START_INDEX] != content[SPAN_END_INDEX] {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return public_key, accepted
	}
	if positive_integer_valid(
		storage, modulus_identifier, modulus_content,
	)[DECISION_INDEX] != DECISION_TRUE {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return public_key, accepted
	}
	if positive_integer_valid(
		storage, exponent_identifier, exponent_content,
	)[DECISION_INDEX] != DECISION_TRUE {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return public_key, accepted
	}
	exponent_start := exponent_content[SPAN_START_INDEX]
	exponent_end := exponent_content[SPAN_END_INDEX]
	exponent_bytes := storage[exponent_start:exponent_end]
	if string(exponent_bytes) != RSA_EXPONENT_ENCODING {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return public_key, accepted
	}
	if modulus_content[SPAN_END_INDEX]-modulus_content[SPAN_START_INDEX] ==
		rsa.MODULUS_SIZE+binary.UINT_8_SIZE {
		modulus_content[SPAN_START_INDEX] += binary.UINT_8_SIZE
	}
	modulus_bytes := storage[modulus_content[SPAN_START_INDEX]:modulus_content[SPAN_END_INDEX]]
	if rsa.Public_Key_Set_Bytes(
		&public_key, rsa.Modulus_Unvalidated(modulus_bytes),
	) != rsa.KEY_STATUS_OK {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return public_key, accepted
	}
	accepted[DECISION_INDEX] = DECISION_TRUE
	return public_key, accepted
}

func parse_name(
	storage *[ENCODED_SIZE_MAXIMUM]byte, source [SPAN_FIELD_COUNT]int,
) (common_name [SPAN_FIELD_COUNT]int, accepted [binary.UINT_8_SIZE]uint64) {
	identifier, content, encoded, tail, accepted := take_element(storage, source)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return common_name, accepted
	}
	if encoded != source {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return common_name, accepted
	}
	if tail[SPAN_START_INDEX] != tail[SPAN_END_INDEX] {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return common_name, accepted
	}
	if identifier_matches(
		identifier, sequence_identifier(),
	)[DECISION_INDEX] != DECISION_TRUE {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return common_name, accepted
	}
	for content[SPAN_START_INDEX] != content[SPAN_END_INDEX] {
		set_identifier, set_content, _, next_content, valid := take_element(
			storage, content,
		)
		if valid[DECISION_INDEX] != DECISION_TRUE {
			return common_name, accepted
		}
		if name_set_valid(
			storage, set_identifier, set_content,
		)[DECISION_INDEX] != DECISION_TRUE {
			return common_name, accepted
		}
		_, attribute_content, _, _, _ := take_element(storage, set_content)
		_, _, _, value_source, _ := take_element(storage, attribute_content)
		_, value_content, _, _, _ := take_element(storage, value_source)
		_, oid_content, _, _, _ :=
			take_element(storage, attribute_content)
		oid_bytes := storage[oid_content[SPAN_START_INDEX]:oid_content[SPAN_END_INDEX]]
		if string(oid_bytes) == COMMON_NAME_OID_ENCODING {
			if common_name[SPAN_END_INDEX] == ENCODED_SIZE_MINIMUM {
				common_name = value_content
			}
		}
		content = next_content
	}
	accepted[DECISION_INDEX] = DECISION_TRUE
	return common_name, accepted
}

func name_set_valid(
	storage *[ENCODED_SIZE_MAXIMUM]byte,
	set_identifier [IDENTIFIER_FIELD_COUNT]uint32,
	set_content [SPAN_FIELD_COUNT]int,
) (accepted [binary.UINT_8_SIZE]uint64) {
	if identifier_matches(
		set_identifier, set_identifier_expected(),
	)[DECISION_INDEX] != DECISION_TRUE {
		return accepted
	}
	attribute_identifier, attribute_content, _, tail, accepted :=
		take_element(storage, set_content)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return accepted
	}
	if tail[SPAN_START_INDEX] != tail[SPAN_END_INDEX] {
		return accepted
	}
	if identifier_matches(
		attribute_identifier, sequence_identifier(),
	)[DECISION_INDEX] != DECISION_TRUE {
		return accepted
	}
	oid_identifier, _, _, value_source, accepted :=
		take_element(storage, attribute_content)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return accepted
	}
	if identifier_matches(
		oid_identifier, oid_identifier_expected(),
	)[DECISION_INDEX] != DECISION_TRUE {
		return accepted
	}
	_, _, _, value_tail, accepted := take_element(storage, value_source)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return accepted
	}
	if value_tail[SPAN_START_INDEX] != value_tail[SPAN_END_INDEX] {
		return accepted
	}
	accepted[DECISION_INDEX] = DECISION_TRUE
	return accepted
}

func signature_algorithm_parse(
	storage *[ENCODED_SIZE_MAXIMUM]byte, algorithm [SPAN_FIELD_COUNT]int,
) (result Signature_Algorithm, recognized [binary.UINT_8_SIZE]uint64) {
	defer func() {
		Signature_Algorithm_Invariants(result, "signature_algorithm_parse.result")
	}()
	algorithm_bytes := storage[algorithm[SPAN_START_INDEX]:algorithm[SPAN_END_INDEX]]
	if string(algorithm_bytes) == RSA_SIGNATURE_IDENTIFIER_ENCODING {
		result = SIGNATURE_ALGORITHM_RSA_SHA_256
		recognized[DECISION_INDEX] = DECISION_TRUE
		return result, recognized
	}
	if string(algorithm_bytes) == ECDSA_SIGNATURE_IDENTIFIER_ENCODING {
		result = SIGNATURE_ALGORITHM_ECDSA_SHA_256
		recognized[DECISION_INDEX] = DECISION_TRUE
		return result, recognized
	}
	if string(algorithm_bytes) == ED25519_IDENTIFIER_ENCODING {
		result = SIGNATURE_ALGORITHM_ED25519
		recognized[DECISION_INDEX] = DECISION_TRUE
	}
	return result, recognized
}

func take_element(
	storage *[ENCODED_SIZE_MAXIMUM]byte, source [SPAN_FIELD_COUNT]int,
) (
	identifier [IDENTIFIER_FIELD_COUNT]uint32,
	content [SPAN_FIELD_COUNT]int,
	encoded [SPAN_FIELD_COUNT]int,
	tail [SPAN_FIELD_COUNT]int,
	accepted [binary.UINT_8_SIZE]uint64,
) {
	source_bytes := storage[source[SPAN_START_INDEX]:source[SPAN_END_INDEX]]
	element, consumed, _, status := asn1.Decode(asn1.Encoded(source_bytes))
	if status != asn1.STATUS_OK {
		return identifier, content, encoded, source, accepted
	}
	identifier[IDENTIFIER_CLASS_INDEX] = uint32(element.Class)
	identifier[IDENTIFIER_TAG_INDEX] = uint32(element.Tag)
	if bool(element.Constructed) {
		identifier[IDENTIFIER_CONSTRUCTED_INDEX] = CONSTRUCTED_TRUE
	}
	encoded[SPAN_START_INDEX] = source[SPAN_START_INDEX]
	encoded[SPAN_END_INDEX] = source[SPAN_START_INDEX] + int(consumed)
	content[SPAN_END_INDEX] = encoded[SPAN_END_INDEX]
	content[SPAN_START_INDEX] = content[SPAN_END_INDEX] - len(element.Content)
	tail[SPAN_START_INDEX] = encoded[SPAN_END_INDEX]
	tail[SPAN_END_INDEX] = source[SPAN_END_INDEX]
	accepted[DECISION_INDEX] = DECISION_TRUE
	return identifier, content, encoded, tail, accepted
}

func identifier_matches(
	identifier [IDENTIFIER_FIELD_COUNT]uint32,
	expected [IDENTIFIER_FIELD_COUNT]uint32,
) (matches [binary.UINT_8_SIZE]uint64) {
	if identifier != expected {
		return matches
	}
	matches[DECISION_INDEX] = DECISION_TRUE
	return matches
}

func sequence_identifier() (identifier [IDENTIFIER_FIELD_COUNT]uint32) {
	return [IDENTIFIER_FIELD_COUNT]uint32{
		IDENTIFIER_CLASS_INDEX:       asn1.CLASS_UNIVERSAL,
		IDENTIFIER_TAG_INDEX:         uint32(TAG_SEQUENCE),
		IDENTIFIER_CONSTRUCTED_INDEX: CONSTRUCTED_TRUE,
	}
}

func set_identifier_expected() (identifier [IDENTIFIER_FIELD_COUNT]uint32) {
	return [IDENTIFIER_FIELD_COUNT]uint32{
		IDENTIFIER_CLASS_INDEX:       asn1.CLASS_UNIVERSAL,
		IDENTIFIER_TAG_INDEX:         uint32(TAG_SEQUENCE + binary.UINT_8_SIZE),
		IDENTIFIER_CONSTRUCTED_INDEX: CONSTRUCTED_TRUE,
	}
}

func oid_identifier_expected() (identifier [IDENTIFIER_FIELD_COUNT]uint32) {
	return [IDENTIFIER_FIELD_COUNT]uint32{
		IDENTIFIER_CLASS_INDEX:       asn1.CLASS_UNIVERSAL,
		IDENTIFIER_TAG_INDEX:         uint32(TAG_OBJECT_IDENTIFIER),
		IDENTIFIER_CONSTRUCTED_INDEX: CONSTRUCTED_FALSE,
	}
}

func bit_string_identifier() (identifier [IDENTIFIER_FIELD_COUNT]uint32) {
	return [IDENTIFIER_FIELD_COUNT]uint32{
		IDENTIFIER_CLASS_INDEX:       asn1.CLASS_UNIVERSAL,
		IDENTIFIER_TAG_INDEX:         uint32(TAG_BIT_STRING),
		IDENTIFIER_CONSTRUCTED_INDEX: CONSTRUCTED_FALSE,
	}
}

func integer_identifier() (identifier [IDENTIFIER_FIELD_COUNT]uint32) {
	return [IDENTIFIER_FIELD_COUNT]uint32{
		IDENTIFIER_CLASS_INDEX:       asn1.CLASS_UNIVERSAL,
		IDENTIFIER_TAG_INDEX:         uint32(TAG_INTEGER),
		IDENTIFIER_CONSTRUCTED_INDEX: CONSTRUCTED_FALSE,
	}
}

func bit_string_content(
	storage *[ENCODED_SIZE_MAXIMUM]byte,
	identifier [IDENTIFIER_FIELD_COUNT]uint32,
	content [SPAN_FIELD_COUNT]int,
) (value [SPAN_FIELD_COUNT]int, valid [binary.UINT_8_SIZE]uint64) {
	if identifier_matches(
		identifier, bit_string_identifier(),
	)[DECISION_INDEX] != DECISION_TRUE {
		return value, valid
	}
	if content[SPAN_END_INDEX]-content[SPAN_START_INDEX] <
		BIT_STRING_UNUSED_SIZE {
		return value, valid
	}
	if storage[content[SPAN_START_INDEX]] != BIT_STRING_UNUSED_EMPTY {
		return value, valid
	}
	value[SPAN_START_INDEX] = content[SPAN_START_INDEX] + BIT_STRING_UNUSED_SIZE
	value[SPAN_END_INDEX] = content[SPAN_END_INDEX]
	valid[DECISION_INDEX] = DECISION_TRUE
	return value, valid
}

func version_valid(
	storage *[ENCODED_SIZE_MAXIMUM]byte,
	identifier [IDENTIFIER_FIELD_COUNT]uint32,
	content [SPAN_FIELD_COUNT]int,
) (valid [binary.UINT_8_SIZE]uint64) {
	expected := [IDENTIFIER_FIELD_COUNT]uint32{
		IDENTIFIER_CLASS_INDEX:       asn1.CLASS_CONTEXT_SPECIFIC,
		IDENTIFIER_TAG_INDEX:         uint32(VERSION_TAG),
		IDENTIFIER_CONSTRUCTED_INDEX: CONSTRUCTED_TRUE,
	}
	if identifier_matches(identifier, expected)[DECISION_INDEX] != DECISION_TRUE {
		return valid
	}
	version_identifier, version_content, encoded, tail, accepted :=
		take_element(storage, content)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return valid
	}
	if encoded != content {
		return valid
	}
	if tail[SPAN_START_INDEX] != tail[SPAN_END_INDEX] {
		return valid
	}
	if identifier_matches(
		version_identifier, integer_identifier(),
	)[DECISION_INDEX] != DECISION_TRUE {
		return valid
	}
	if version_content[SPAN_END_INDEX]-version_content[SPAN_START_INDEX] !=
		binary.UINT_8_SIZE {
		return valid
	}
	if storage[version_content[SPAN_START_INDEX]] == VERSION_V3 {
		valid[DECISION_INDEX] = DECISION_TRUE
	}
	return valid
}

func serial_valid(
	storage *[ENCODED_SIZE_MAXIMUM]byte,
	identifier [IDENTIFIER_FIELD_COUNT]uint32,
	content [SPAN_FIELD_COUNT]int,
) (valid [binary.UINT_8_SIZE]uint64) {
	if positive_integer_valid(
		storage, identifier, content,
	)[DECISION_INDEX] != DECISION_TRUE {
		return valid
	}
	if content[SPAN_END_INDEX]-content[SPAN_START_INDEX] >
		SERIAL_NUMBER_SIZE_MAXIMUM {
		return valid
	}
	var nonzero byte
	serial_bytes := storage[content[SPAN_START_INDEX]:content[SPAN_END_INDEX]]
	for _, value := range serial_bytes {
		nonzero |= value
	}
	if nonzero != bits.WORD_8_MINIMUM {
		valid[DECISION_INDEX] = DECISION_TRUE
	}
	return valid
}

func positive_integer_valid(
	storage *[ENCODED_SIZE_MAXIMUM]byte,
	identifier [IDENTIFIER_FIELD_COUNT]uint32,
	content [SPAN_FIELD_COUNT]int,
) (valid [binary.UINT_8_SIZE]uint64) {
	if identifier_matches(
		identifier, integer_identifier(),
	)[DECISION_INDEX] != DECISION_TRUE {
		return valid
	}
	size := content[SPAN_END_INDEX] - content[SPAN_START_INDEX]
	if size == ENCODED_SIZE_MINIMUM {
		return valid
	}
	bytes := storage[content[SPAN_START_INDEX]:content[SPAN_END_INDEX]]
	if bytes[bits.BIT_COUNT_MINIMUM]&
		byte(binary.UINT_8_SIZE<<(bits.BIT_COUNT_8_MAXIMUM-binary.UINT_8_SIZE)) !=
		bits.WORD_8_MINIMUM {
		return valid
	}
	if size == binary.UINT_8_SIZE {
		valid[DECISION_INDEX] = DECISION_TRUE
		return valid
	}
	if bytes[bits.BIT_COUNT_MINIMUM] != bits.WORD_8_MINIMUM {
		valid[DECISION_INDEX] = DECISION_TRUE
		return valid
	}
	if bytes[binary.UINT_8_SIZE]&
		byte(binary.UINT_8_SIZE<<(bits.BIT_COUNT_8_MAXIMUM-binary.UINT_8_SIZE)) !=
		bits.WORD_8_MINIMUM {
		valid[DECISION_INDEX] = DECISION_TRUE
	}
	return valid
}

func validity_valid(
	storage *[ENCODED_SIZE_MAXIMUM]byte,
	identifier [IDENTIFIER_FIELD_COUNT]uint32,
	content [SPAN_FIELD_COUNT]int,
) (valid [binary.UINT_8_SIZE]uint64) {
	if identifier_matches(
		identifier, sequence_identifier(),
	)[DECISION_INDEX] != DECISION_TRUE {
		return valid
	}
	not_before_identifier, not_before_content, _, tail, accepted :=
		take_element(storage, content)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return valid
	}
	not_after_identifier, not_after_content, _, tail, accepted :=
		take_element(storage, tail)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return valid
	}
	if tail[SPAN_START_INDEX] != tail[SPAN_END_INDEX] {
		return valid
	}
	if time_element_valid(
		not_before_identifier, not_before_content,
	)[DECISION_INDEX]&time_element_valid(
		not_after_identifier, not_after_content,
	)[DECISION_INDEX] == DECISION_TRUE {
		valid[DECISION_INDEX] = DECISION_TRUE
	}
	return valid
}

func time_element_valid(
	identifier [IDENTIFIER_FIELD_COUNT]uint32,
	content [SPAN_FIELD_COUNT]int,
) (valid [binary.UINT_8_SIZE]uint64) {
	if identifier[IDENTIFIER_CLASS_INDEX] != asn1.CLASS_UNIVERSAL {
		return valid
	}
	if identifier[IDENTIFIER_CONSTRUCTED_INDEX] != CONSTRUCTED_FALSE {
		return valid
	}
	if identifier[IDENTIFIER_TAG_INDEX] != uint32(TAG_TIME_UTC) {
		if identifier[IDENTIFIER_TAG_INDEX] != uint32(TAG_TIME_GENERALIZED) {
			return valid
		}
	}
	if content[SPAN_START_INDEX] != content[SPAN_END_INDEX] {
		valid[DECISION_INDEX] = DECISION_TRUE
	}
	return valid
}

func optional_fields_valid(
	storage *[ENCODED_SIZE_MAXIMUM]byte, content [SPAN_FIELD_COUNT]int,
) (valid [binary.UINT_8_SIZE]uint64) {
	previous_tag := uint32(OPTIONAL_TAG_MINIMUM - binary.UINT_8_SIZE)
	for content[SPAN_START_INDEX] != content[SPAN_END_INDEX] {
		identifier, _, _, tail, accepted := take_element(storage, content)
		if accepted[DECISION_INDEX] != DECISION_TRUE {
			return valid
		}
		if identifier[IDENTIFIER_CLASS_INDEX] != asn1.CLASS_CONTEXT_SPECIFIC {
			return valid
		}
		tag := identifier[IDENTIFIER_TAG_INDEX]
		if tag < uint32(OPTIONAL_TAG_MINIMUM) {
			return valid
		}
		if tag > uint32(OPTIONAL_TAG_MAXIMUM) {
			return valid
		}
		if tag <= previous_tag {
			return valid
		}
		if tag == uint32(OPTIONAL_TAG_MAXIMUM) {
			if identifier[IDENTIFIER_CONSTRUCTED_INDEX] != CONSTRUCTED_TRUE {
				return valid
			}
		}
		previous_tag = tag
		content = tail
	}
	valid[DECISION_INDEX] = DECISION_TRUE
	return valid
}

// Verify_Signature_From checks certificate bytes against one parsed issuer key.
func Verify_Signature_From(
	certificate Certificate_Handle, issuer Certificate_Handle,
) (verified Verification) {
	defer func() {
		Verification_Invariants(verified, "Verify_Signature_From.verified")
	}()
	Certificate_Handle_Invariants(certificate, "Verify_Signature_From.certificate")
	Certificate_Handle_Invariants(issuer, "Verify_Signature_From.issuer")
	if certificate.Signature_Algorithm == SIGNATURE_ALGORITHM_RSA_SHA_256 {
		if issuer.Public_Key_Algorithm != PUBLIC_KEY_ALGORITHM_RSA {
			return false
		}
		digest := digest_sha_256(sha256.Source(certificate.TBS))
		return Verification(rsa.Verify_PKCS1_V1_5_SHA_256(
			issuer.RSA_Public_Key, rsa.Digest(digest),
			rsa.Signature_Unvalidated(certificate.Signature),
		))
	}
	if certificate.Signature_Algorithm == SIGNATURE_ALGORITHM_ECDSA_SHA_256 {
		if issuer.Public_Key_Algorithm != PUBLIC_KEY_ALGORITHM_ECDSA {
			return false
		}
		var signature_storage [ENCODED_SIZE_MAXIMUM]byte
		copy(signature_storage[:], certificate.Signature)
		signature_span := [SPAN_FIELD_COUNT]int{
			SPAN_START_INDEX: ENCODED_SIZE_MINIMUM,
			SPAN_END_INDEX:   len(certificate.Signature),
		}
		signature, accepted := ecdsa_signature_parse(
			&signature_storage, signature_span,
		)
		if accepted[DECISION_INDEX] != DECISION_TRUE {
			return false
		}
		digest := digest_sha_256(sha256.Source(certificate.TBS))
		return Verification(ecdsa.Verify(
			issuer.ECDSA_Public_Key, ecdsa.Digest(digest), signature[:],
		))
	}
	if certificate.Signature_Algorithm == SIGNATURE_ALGORITHM_ED25519 {
		if issuer.Public_Key_Algorithm != PUBLIC_KEY_ALGORITHM_ED25519 {
			return false
		}
		return Verification(ed25519.Verify(
			issuer.Ed25519_Public_Key, ed25519.Message(certificate.TBS),
			ed25519.Signature_Unvalidated(certificate.Signature),
		))
	}
	return false
}

func digest_sha_256(source sha256.Source) (digest sha256.Value_256) {
	defer func() {
		sha256.Value_256_Invariants(digest, "digest_sha_256.digest")
	}()
	sha256.Source_Invariants(source, "digest_sha_256.source")
	var state sha256.Digest
	sha256.Digest_Init(&state, sha256.KIND_SHA_256)
	sha256.Digest_Write(&state, sha256.Source(source))
	return sha256.Digest_Sum_256(&state)
}

func ecdsa_signature_parse(
	storage *[ENCODED_SIZE_MAXIMUM]byte, source [SPAN_FIELD_COUNT]int,
) (signature ecdsa.Signature, accepted [binary.UINT_8_SIZE]uint64) {
	defer func() {
		ecdsa.Signature_Invariants(signature, "ecdsa_signature_parse.signature")
	}()
	identifier, content, encoded, tail, accepted := take_element(storage, source)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return signature, accepted
	}
	if encoded != source {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return signature, accepted
	}
	if tail[SPAN_START_INDEX] != tail[SPAN_END_INDEX] {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return signature, accepted
	}
	if identifier_matches(
		identifier, sequence_identifier(),
	)[DECISION_INDEX] != DECISION_TRUE {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return signature, accepted
	}
	r_identifier, r_content, _, content, accepted := take_element(storage, content)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return signature, accepted
	}
	s_identifier, s_content, _, content, accepted := take_element(storage, content)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return signature, accepted
	}
	if content[SPAN_START_INDEX] != content[SPAN_END_INDEX] {
		accepted[DECISION_INDEX] = DECISION_FALSE
		return signature, accepted
	}
	r_scalar, accepted := integer_scalar(storage, r_identifier, r_content)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return signature, accepted
	}
	s_scalar, accepted := integer_scalar(storage, s_identifier, s_content)
	if accepted[DECISION_INDEX] != DECISION_TRUE {
		return signature, accepted
	}
	copy(signature[:ecdsa.SCALAR_SIZE], r_scalar[:])
	copy(signature[ecdsa.SCALAR_SIZE:], s_scalar[:])
	accepted[DECISION_INDEX] = DECISION_TRUE
	return signature, accepted
}

func integer_scalar(
	storage *[ENCODED_SIZE_MAXIMUM]byte,
	identifier [IDENTIFIER_FIELD_COUNT]uint32,
	content [SPAN_FIELD_COUNT]int,
) (
	scalar [ecdsa.SCALAR_SIZE]byte, accepted [binary.UINT_8_SIZE]uint64,
) {
	if positive_integer_valid(
		storage, identifier, content,
	)[DECISION_INDEX] != DECISION_TRUE {
		return scalar, accepted
	}
	if content[SPAN_END_INDEX]-content[SPAN_START_INDEX] ==
		len(scalar)+binary.UINT_8_SIZE {
		content[SPAN_START_INDEX] += binary.UINT_8_SIZE
	}
	content_size := content[SPAN_END_INDEX] - content[SPAN_START_INDEX]
	if content_size > len(scalar) {
		return scalar, accepted
	}
	offset := len(scalar) - content_size
	copy(scalar[offset:], storage[content[SPAN_START_INDEX]:content[SPAN_END_INDEX]])
	accepted[DECISION_INDEX] = DECISION_TRUE
	return scalar, accepted
}
