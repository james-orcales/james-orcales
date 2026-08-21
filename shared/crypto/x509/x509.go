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

// READY_EMPTY marks caller storage without a parsed certificate.
const READY_EMPTY Ready = Ready(bits.WORD_8_MINIMUM)

// READY_COMPLETE marks a fully validated certificate.
const READY_COMPLETE Ready = READY_EMPTY + binary.UINT_8_SIZE

// DECISION_FALSE is zero parser acceptance.
const DECISION_FALSE Decision = Decision(bits.WORD_64_MINIMUM)

// DECISION_TRUE follows false by one binary state.
const DECISION_TRUE Decision = DECISION_FALSE + binary.UINT_8_SIZE

// Storage is exact parser scratch detached from hostile input ownership.
type Storage []byte

// Storage_Invariants fixes every internal position to one complete scratch region.
func Storage_Invariants(value Storage, _ aver.Namespace) {
	aver.Always(len(value) == ENCODED_SIZE_MAXIMUM, "X.509 scratch has maximum DER width.")
}

// Span_Start stores one inclusive parser position.
type Span_Start int

// Span_Start_Invariants covers every scratch position.
func Span_Start_Invariants(value Span_Start, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), ENCODED_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Span_End stores one exclusive parser position.
type Span_End int

// Span_End_Invariants covers every scratch position.
func Span_End_Invariants(value Span_End, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), ENCODED_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Span keeps parser positions inline without a fixed-array boundary.
type Span struct {
	// Start is inclusive because every borrowed field slices directly from it.
	Start Span_Start
	// End is exclusive because Go slices and DER consumption share that convention.
	End Span_End
}

// Span_Invariants composes both positions and their ordering.
func Span_Invariants(value Span, namespace aver.Namespace) {
	Span_Start_Invariants(value.Start, namespace)
	Span_End_Invariants(value.End, namespace)
	aver.Always(
		int(value.Start) >= ENCODED_SIZE_MINIMUM,
		"A parser span starts at a nonnegative scratch position.",
	)
	aver.Always(
		int(value.Start) <= ENCODED_SIZE_MAXIMUM,
		"A parser span starts inside exact scratch.",
	)
	aver.Always(
		int(value.End) >= ENCODED_SIZE_MINIMUM,
		"A parser span ends at a nonnegative scratch position.",
	)
	aver.Always(
		int(value.End) <= ENCODED_SIZE_MAXIMUM,
		"A parser span ends inside exact scratch.",
	)
	aver.Always(int(value.Start) <= int(value.End), "A parser span cannot run backward.")
}

// Span_Handle names mutable parser-position storage.
type Span_Handle *Span

// Span_Handle_Invariants composes present parser positions.
func Span_Handle_Invariants(value Span_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Span_Invariants(*value, namespace)
}

// TBS_Span gives authenticated certificate bytes independent field identity.
type TBS_Span Span

// TBS_Span_Invariants guards its two bounded positions.
func TBS_Span_Invariants(value TBS_Span, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Start), ENCODED_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Range_Int(int(value.End), ENCODED_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Signature_Span gives signature bytes independent field identity.
type Signature_Span Span

// Signature_Span_Invariants guards its two bounded positions.
func Signature_Span_Invariants(value Signature_Span, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Start), ENCODED_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Range_Int(int(value.End), ENCODED_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Issuer_Raw_Span gives issuer identity independent field identity.
type Issuer_Raw_Span Span

// Issuer_Raw_Span_Invariants guards its two bounded positions.
func Issuer_Raw_Span_Invariants(value Issuer_Raw_Span, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Start), ENCODED_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Range_Int(int(value.End), ENCODED_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Issuer_Common_Name_Span gives issuer presentation text independent field identity.
type Issuer_Common_Name_Span Span

// Issuer_Common_Name_Span_Invariants guards its two bounded positions.
func Issuer_Common_Name_Span_Invariants(
	value Issuer_Common_Name_Span, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Start), ENCODED_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Range_Int(int(value.End), ENCODED_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Subject_Raw_Span gives subject identity independent field identity.
type Subject_Raw_Span Span

// Subject_Raw_Span_Invariants guards its two bounded positions.
func Subject_Raw_Span_Invariants(value Subject_Raw_Span, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Start), ENCODED_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Range_Int(int(value.End), ENCODED_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Subject_Common_Name_Span gives subject presentation text independent field identity.
type Subject_Common_Name_Span Span

// Subject_Common_Name_Span_Invariants guards its two bounded positions.
func Subject_Common_Name_Span_Invariants(
	value Subject_Common_Name_Span, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Start), ENCODED_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Range_Int(int(value.End), ENCODED_SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Identifier_Class stores one decoder-owned DER class field.
type Identifier_Class uint32

// Identifier_Class_Invariants covers the complete two-bit class field.
func Identifier_Class_Invariants(value Identifier_Class, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint32(
			uint32(value), asn1.CLASS_UNIVERSAL, asn1.CLASS_APPLICATION,
			asn1.CLASS_CONTEXT_SPECIFIC, asn1.CLASS_PRIVATE,
		).
		Ensure()
}

// Identifier_Tag stores one decoder-owned DER tag field.
type Identifier_Tag uint32

// Identifier_Tag_Invariants covers the decoder's signed tag domain.
func Identifier_Tag_Invariants(value Identifier_Tag, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, asn1.TAG_MAXIMUM).
		Ensure()
}

// Constructed stores the DER primitive or constructed bit.
type Constructed uint32

// Constructed_Invariants covers primitive and constructed encodings.
func Constructed_Invariants(value Constructed, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint32(uint32(value), CONSTRUCTED_FALSE, CONSTRUCTED_TRUE).
		Ensure()
}

// Identifier keeps one decoded DER identity inline.
type Identifier struct {
	// Class separates universal tags from context-specific certificate fields.
	Class Identifier_Class
	// Tag identifies the element meaning within Class.
	Tag Identifier_Tag
	// Constructed prevents primitive encodings from impersonating containers.
	Constructed Constructed
}

// Identifier_Invariants composes each decoded field and decoder promise once.
func Identifier_Invariants(value Identifier, namespace aver.Namespace) {
	Identifier_Class_Invariants(value.Class, namespace)
	Identifier_Tag_Invariants(value.Tag, namespace)
	Constructed_Invariants(value.Constructed, namespace)
	aver.Always(
		uint32(value.Class) <= uint32(asn1.CLASS_MAXIMUM),
		"A DER class retains its two-bit domain.",
	)
	aver.Always(
		uint32(value.Tag) <= asn1.TAG_MAXIMUM,
		"A DER tag remains decoder-bounded.",
	)
	aver.Always(
		uint32(value.Constructed) <= CONSTRUCTED_TRUE,
		"A DER construction field is binary.",
	)
}

// Identifier_Handle names mutable DER identity storage.
type Identifier_Handle *Identifier

// Identifier_Handle_Invariants composes one present DER identity.
func Identifier_Handle_Invariants(value Identifier_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Identifier_Invariants(*value, namespace)
}

// Decision is one parser acceptance bit.
type Decision uint64

// Decision_Invariants admits only refused and accepted parser states.
func Decision_Invariants(value Decision, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint64(uint64(value), uint64(DECISION_FALSE), uint64(DECISION_TRUE)).
		Ensure()
}

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
type Ready uint8

// Ready_Invariants admits empty and complete certificate storage.
func Ready_Invariants(value Ready, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(READY_EMPTY), uint8(READY_COMPLETE)).
		Ensure()
}

// Ed25519_Public_Key_Storage keeps the inactive union member copy-safe and allocation-free.
type Ed25519_Public_Key_Storage ed25519.Limb_Storage

// Ed25519_Public_Key_Storage_Invariants covers every packed key word.
func Ed25519_Public_Key_Storage_Invariants(
	value Ed25519_Public_Key_Storage, _ aver.Namespace,
) {
	converted := ed25519.Limb_Storage(value)
	aver.Always(
		Ed25519_Public_Key_Storage(converted) == value,
		"Safe key storage conversion preserves every compressed-point word.",
	)
}

// Ed25519_Public_Key_Destination names mutable inline public-key storage.
type Ed25519_Public_Key_Destination *Ed25519_Public_Key_Storage

// Ed25519_Public_Key_Destination_Invariants composes present inline key storage.
func Ed25519_Public_Key_Destination_Invariants(
	value Ed25519_Public_Key_Destination, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Ed25519_Public_Key_Storage_Invariants(*value, namespace)
}

func ed25519_public_key_set(
	destination Ed25519_Public_Key_Destination, source ed25519.Public_Key,
) {
	Ed25519_Public_Key_Destination_Invariants(destination, "ed25519_public_key_set.destination")
	ed25519.Public_Key_Invariants(source, "ed25519_public_key_set.source")
	destination.Limb_0 = ed25519.Limb_0(binary.Uint_64(
		binary.Bytes(source[0:8]), binary.LITTLE_ENDIAN,
	))
	destination.Limb_1 = ed25519.Limb_1(binary.Uint_64(
		binary.Bytes(source[8:16]), binary.LITTLE_ENDIAN,
	))
	destination.Limb_2 = ed25519.Limb_2(binary.Uint_64(
		binary.Bytes(source[16:24]), binary.LITTLE_ENDIAN,
	))
	destination.Limb_3 = ed25519.Limb_3(binary.Uint_64(
		binary.Bytes(source[24:32]), binary.LITTLE_ENDIAN,
	))
}

func ed25519_public_key_copy(
	destination ed25519.Public_Key, source Ed25519_Public_Key_Storage,
) {
	ed25519.Public_Key_Invariants(destination, "ed25519_public_key_copy.destination")
	Ed25519_Public_Key_Storage_Invariants(source, "ed25519_public_key_copy.source")
	binary.Put_Uint_64(
		binary.Bytes(destination[0:8]), binary.Word_64(source.Limb_0), binary.LITTLE_ENDIAN,
	)
	binary.Put_Uint_64(
		binary.Bytes(destination[8:16]), binary.Word_64(source.Limb_1),
		binary.LITTLE_ENDIAN,
	)
	binary.Put_Uint_64(
		binary.Bytes(destination[16:24]), binary.Word_64(source.Limb_2),
		binary.LITTLE_ENDIAN,
	)
	binary.Put_Uint_64(
		binary.Bytes(destination[24:32]), binary.Word_64(source.Limb_3),
		binary.LITTLE_ENDIAN,
	)
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
	Ed25519_Public_Key Ed25519_Public_Key_Storage
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
	Ed25519_Public_Key_Storage_Invariants(value.Ed25519_Public_Key, namespace)
	Ready_Invariants(value.Ready, namespace)
}

// Parsed_Certificate keeps byte positions internal until one full parse commits.
type Parsed_Certificate struct {
	// TBS maps authenticated bytes back to caller input.
	TBS TBS_Span
	// Signature maps BIT STRING content back to caller input.
	Signature Signature_Span
	// Issuer_Raw maps the validated issuer Name back to caller input.
	Issuer_Raw Issuer_Raw_Span
	// Issuer_Common_Name maps selected issuer text back to caller input.
	Issuer_Common_Name Issuer_Common_Name_Span
	// Subject_Raw maps the validated subject Name back to caller input.
	Subject_Raw Subject_Raw_Span
	// Subject_Common_Name maps selected subject text back to caller input.
	Subject_Common_Name Subject_Common_Name_Span
	// Public_Key_Algorithm selects one fixed key member.
	Public_Key_Algorithm Public_Key_Algorithm
	// RSA_Public_Key stores the RSA union member.
	RSA_Public_Key rsa.Public_Key
	// ECDSA_Public_Key stores the P-256 union member.
	ECDSA_Public_Key ecdsa.Public_Key
	// Ed25519_Public_Key stores the Edwards union member.
	Ed25519_Public_Key Ed25519_Public_Key_Storage
}

// Parsed_Certificate_Invariants composes fixed spans and fixed key storage.
func Parsed_Certificate_Invariants(
	value Parsed_Certificate, namespace aver.Namespace,
) {
	TBS_Span_Invariants(value.TBS, namespace)
	Signature_Span_Invariants(value.Signature, namespace)
	Issuer_Raw_Span_Invariants(value.Issuer_Raw, namespace)
	Issuer_Common_Name_Span_Invariants(value.Issuer_Common_Name, namespace)
	Subject_Raw_Span_Invariants(value.Subject_Raw, namespace)
	Subject_Common_Name_Span_Invariants(value.Subject_Common_Name, namespace)
	Public_Key_Algorithm_Invariants(value.Public_Key_Algorithm, namespace)
	rsa.Public_Key_Invariants(value.RSA_Public_Key, namespace)
	ecdsa.Public_Key_Invariants(value.ECDSA_Public_Key, namespace)
	Ed25519_Public_Key_Storage_Invariants(value.Ed25519_Public_Key, namespace)
}

// Parsed_Certificate_Destination names transactional parser output.
type Parsed_Certificate_Destination *Parsed_Certificate

// Parsed_Certificate_Destination_Invariants composes present parser output.
func Parsed_Certificate_Destination_Invariants(
	value Parsed_Certificate_Destination, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Parsed_Certificate_Invariants(*value, namespace)
}

// Parsed_Public_Key keeps the selected key and its concrete union storage together.
type Parsed_Public_Key struct {
	// Algorithm selects the committed key member.
	Algorithm Public_Key_Algorithm
	// RSA_Key stores the supported RSA member.
	RSA_Key rsa.Public_Key
	// ECDSA_Key stores the supported P-256 member.
	ECDSA_Key ecdsa.Public_Key
	// Ed25519_Key stores the supported Edwards member.
	Ed25519_Key Ed25519_Public_Key_Storage
}

// Parsed_Public_Key_Invariants composes every concrete union member.
func Parsed_Public_Key_Invariants(value Parsed_Public_Key, namespace aver.Namespace) {
	Public_Key_Algorithm_Invariants(value.Algorithm, namespace)
	rsa.Public_Key_Invariants(value.RSA_Key, namespace)
	ecdsa.Public_Key_Invariants(value.ECDSA_Key, namespace)
	Ed25519_Public_Key_Storage_Invariants(value.Ed25519_Key, namespace)
}

// Parsed_Public_Key_Destination names transactional public-key output.
type Parsed_Public_Key_Destination *Parsed_Public_Key

// Parsed_Public_Key_Destination_Invariants composes present public-key output.
func Parsed_Public_Key_Destination_Invariants(
	value Parsed_Public_Key_Destination, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Parsed_Public_Key_Invariants(*value, namespace)
}

// Certificate_Destination is nonnil caller-owned parsed storage.
type Certificate_Destination *Certificate

// Certificate_Destination_Invariants proves caller storage exists.
func Certificate_Destination_Invariants(
	value Certificate_Destination, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Certificate_Invariants(*value, namespace)
}

// Certificate_Handle is one nonnil parsed certificate authority.
type Certificate_Handle *Certificate

// Certificate_Handle_Invariants requires a complete parsed certificate.
func Certificate_Handle_Invariants(
	value Certificate_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Certificate_Invariants(*value, namespace)
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
	var storage_bytes [ENCODED_SIZE_MAXIMUM]byte
	storage := Storage(storage_bytes[:])
	copy(storage, source)
	source_span := Span{Start: Span_Start(ENCODED_SIZE_MINIMUM), End: Span_End(len(source))}
	var parsed Parsed_Certificate
	signature_algorithm, accepted := parse(&parsed, storage, source_span)
	if accepted != DECISION_TRUE {
		return PARSE_STATUS_INPUT_INVALID
	}
	tbs_span := Span(parsed.TBS)
	tbs := source[tbs_span.Start:tbs_span.End]
	signature_span := Span(parsed.Signature)
	signature := source[signature_span.Start:signature_span.End]
	issuer_raw_span := Span(parsed.Issuer_Raw)
	issuer_raw := source[issuer_raw_span.Start:issuer_raw_span.End]
	issuer_common_name_span := Span(parsed.Issuer_Common_Name)
	issuer_common_name := source[issuer_common_name_span.Start:issuer_common_name_span.End]
	subject_raw_span := Span(parsed.Subject_Raw)
	subject_raw := source[subject_raw_span.Start:subject_raw_span.End]
	subject_common_name_span := Span(parsed.Subject_Common_Name)
	subject_common_name := source[subject_common_name_span.Start:subject_common_name_span.End]
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
		Ready:              READY_COMPLETE,
	}
	return PARSE_STATUS_OK
}

func parse(
	destination Parsed_Certificate_Destination, storage Storage, source Span,
) (
	signature_algorithm Signature_Algorithm,
	accepted Decision,
) {
	defer func() {
		Signature_Algorithm_Invariants(signature_algorithm, "parse.signature_algorithm")
		Decision_Invariants(accepted, "parse.accepted")
	}()
	Parsed_Certificate_Destination_Invariants(destination, "parse.destination")
	Storage_Invariants(storage, "parse.storage")
	Span_Invariants(source, "parse.source")
	var outer_identifier Identifier
	var outer_content, outer_encoded, tail Span
	accepted = take_element(
		&outer_identifier, &outer_content, &outer_encoded, &tail, storage, source,
	)
	if accepted != DECISION_TRUE {
		return signature_algorithm, accepted
	}
	if outer_encoded != source {
		accepted = DECISION_FALSE
		return signature_algorithm, accepted
	}
	if int(tail.Start) != int(tail.End) {
		accepted = DECISION_FALSE
		return signature_algorithm, accepted
	}
	var expected Identifier
	sequence_identifier(&expected)
	if identifier_matches(outer_identifier, expected) != DECISION_TRUE {
		accepted = DECISION_FALSE
		return signature_algorithm, accepted
	}
	return parse_certificate_content(destination, storage, outer_content)
}

func parse_certificate_content(
	destination Parsed_Certificate_Destination, storage Storage, content Span,
) (
	signature_algorithm Signature_Algorithm,
	accepted Decision,
) {
	defer func() {
		Signature_Algorithm_Invariants(
			signature_algorithm, "parse_certificate_content.signature_algorithm",
		)
		Decision_Invariants(accepted, "parse_certificate_content.accepted")
	}()
	Parsed_Certificate_Destination_Invariants(
		destination, "parse_certificate_content.destination",
	)
	Storage_Invariants(storage, "parse_certificate_content.storage")
	Span_Invariants(content, "parse_certificate_content.content")
	var parsed, tbs_parsed Parsed_Certificate
	var tbs_identifier, signature_identifier, ignored_identifier Identifier
	var ignored, tbs_span, tail, algorithm_span, signature_content, signature_span Span
	accepted = take_element(
		&tbs_identifier, &ignored, &tbs_span, &tail, storage, content,
	)
	if accepted != DECISION_TRUE {
		return signature_algorithm, accepted
	}
	accepted = take_element(
		&ignored_identifier, &ignored, &algorithm_span, &tail, storage, tail,
	)
	if accepted != DECISION_TRUE {
		return signature_algorithm, accepted
	}
	accepted = take_element(
		&signature_identifier, &signature_content, &ignored, &tail, storage, tail,
	)
	if accepted != DECISION_TRUE {
		return signature_algorithm, accepted
	}
	if int(tail.Start) != int(tail.End) {
		accepted = DECISION_FALSE
		return signature_algorithm, accepted
	}
	var expected Identifier
	sequence_identifier(&expected)
	if identifier_matches(tbs_identifier, expected) != DECISION_TRUE {
		accepted = DECISION_FALSE
		return signature_algorithm, accepted
	}
	signature_algorithm, accepted = signature_algorithm_parse(storage, algorithm_span)
	if accepted != DECISION_TRUE {
		return signature_algorithm, accepted
	}
	accepted = bit_string_content(
		&signature_span, storage, signature_identifier, signature_content,
	)
	if accepted != DECISION_TRUE {
		return signature_algorithm, accepted
	}
	if int(signature_span.End)-int(signature_span.Start) >
		SIGNATURE_SIZE_MAXIMUM {
		accepted = DECISION_FALSE
		return signature_algorithm, accepted
	}
	accepted = parse_tbs(&tbs_parsed, storage, tbs_span, algorithm_span)
	if accepted != DECISION_TRUE {
		return signature_algorithm, accepted
	}
	parsed = tbs_parsed
	parsed.TBS = TBS_Span(tbs_span)
	parsed.Signature = Signature_Span(signature_span)
	*destination = parsed
	return signature_algorithm, accepted
}

func parse_tbs(
	destination Parsed_Certificate_Destination,
	storage Storage,
	source Span,
	outer_algorithm Span,
) (accepted Decision) {
	defer func() { Decision_Invariants(accepted, "parse_tbs.accepted") }()
	Parsed_Certificate_Destination_Invariants(destination, "parse_tbs.destination")
	Storage_Invariants(storage, "parse_tbs.storage")
	Span_Invariants(source, "parse_tbs.source")
	Span_Invariants(outer_algorithm, "parse_tbs.outer_algorithm")
	var identifier, serial_identifier Identifier
	var content, encoded, tail, serial_content, ignored Span
	accepted = take_element(&identifier, &content, &encoded, &tail, storage, source)
	if accepted != DECISION_TRUE {
		return accepted
	}
	if encoded != source {
		accepted = DECISION_FALSE
		return accepted
	}
	if int(tail.Start) != int(tail.End) {
		accepted = DECISION_FALSE
		return accepted
	}
	var expected Identifier
	sequence_identifier(&expected)
	if identifier_matches(identifier, expected) != DECISION_TRUE {
		accepted = DECISION_FALSE
		return accepted
	}
	accepted = take_element(
		&serial_identifier, &serial_content, &ignored, &content, storage, content,
	)
	if accepted != DECISION_TRUE {
		return accepted
	}
	if serial_identifier.Class == Identifier_Class(asn1.CLASS_CONTEXT_SPECIFIC) {
		if version_valid(
			storage, serial_identifier, serial_content,
		) != DECISION_TRUE {
			return accepted
		}
		accepted = take_element(
			&serial_identifier, &serial_content, &ignored, &content, storage, content,
		)
		if accepted != DECISION_TRUE {
			return accepted
		}
	}
	if serial_valid(
		storage, serial_identifier, serial_content,
	) != DECISION_TRUE {
		accepted = DECISION_FALSE
		return accepted
	}
	return parse_tbs_fields(destination, storage, content, outer_algorithm)
}

func parse_tbs_fields(
	destination Parsed_Certificate_Destination,
	storage Storage,
	content Span,
	outer_algorithm Span,
) (accepted Decision) {
	defer func() { Decision_Invariants(accepted, "parse_tbs_fields.accepted") }()
	Parsed_Certificate_Destination_Invariants(destination, "parse_tbs_fields.destination")
	Storage_Invariants(storage, "parse_tbs_fields.storage")
	Span_Invariants(content, "parse_tbs_fields.content")
	Span_Invariants(outer_algorithm, "parse_tbs_fields.outer_algorithm")
	var parsed Parsed_Certificate
	var ignored_identifier, validity_identifier Identifier
	var ignored_span, algorithm Span
	accepted = take_element(
		&ignored_identifier, &ignored_span, &algorithm, &content, storage, content,
	)
	if accepted != DECISION_TRUE {
		return accepted
	}
	algorithm_bytes := storage[int(algorithm.Start):int(algorithm.End)]
	outer_bytes := storage[int(outer_algorithm.Start):int(outer_algorithm.End)]
	if !bool(bytes.Equal(bytes.Slice(algorithm_bytes), bytes.Slice(outer_bytes))) {
		accepted = DECISION_FALSE
		return accepted
	}
	var issuer_raw Span
	accepted = take_element(
		&ignored_identifier, &ignored_span, &issuer_raw, &content, storage, content,
	)
	if accepted != DECISION_TRUE {
		return accepted
	}
	parsed.Issuer_Raw = Issuer_Raw_Span(issuer_raw)
	var issuer_common_name Span
	accepted = parse_name(&issuer_common_name, storage, issuer_raw)
	if accepted != DECISION_TRUE {
		return accepted
	}
	parsed.Issuer_Common_Name = Issuer_Common_Name_Span(issuer_common_name)
	var validity_content Span
	accepted = take_element(
		&validity_identifier, &validity_content, &ignored_span, &content, storage, content,
	)
	if accepted != DECISION_TRUE {
		return accepted
	}
	if validity_valid(
		storage, validity_identifier, validity_content,
	) != DECISION_TRUE {
		accepted = DECISION_FALSE
		return accepted
	}
	accepted = parse_tbs_subject(&parsed, storage, content)
	if accepted == DECISION_TRUE {
		*destination = parsed
	}
	return accepted
}

func parse_tbs_subject(
	destination Parsed_Certificate_Destination, storage Storage, content Span,
) (accepted Decision) {
	defer func() { Decision_Invariants(accepted, "parse_tbs_subject.accepted") }()
	Parsed_Certificate_Destination_Invariants(destination, "parse_tbs_subject.destination")
	Storage_Invariants(storage, "parse_tbs_subject.storage")
	Span_Invariants(content, "parse_tbs_subject.content")
	parsed := *destination
	var ignored_identifier Identifier
	var ignored_span Span
	var subject_raw Span
	accepted = take_element(
		&ignored_identifier, &ignored_span, &subject_raw, &content, storage, content,
	)
	if accepted != DECISION_TRUE {
		return accepted
	}
	parsed.Subject_Raw = Subject_Raw_Span(subject_raw)
	var subject_common_name Span
	accepted = parse_name(&subject_common_name, storage, subject_raw)
	if accepted != DECISION_TRUE {
		return accepted
	}
	parsed.Subject_Common_Name = Subject_Common_Name_Span(subject_common_name)
	var public_key Span
	accepted = take_element(
		&ignored_identifier, &ignored_span, &public_key, &content, storage, content,
	)
	if accepted != DECISION_TRUE {
		return accepted
	}
	var parsed_public_key Parsed_Public_Key
	accepted = parse_public_key(&parsed_public_key, storage, public_key)
	if accepted != DECISION_TRUE {
		return accepted
	}
	parsed.Public_Key_Algorithm = parsed_public_key.Algorithm
	parsed.RSA_Public_Key = parsed_public_key.RSA_Key
	parsed.ECDSA_Public_Key = parsed_public_key.ECDSA_Key
	parsed.Ed25519_Public_Key = parsed_public_key.Ed25519_Key
	accepted = optional_fields_valid(storage, content)
	if accepted == DECISION_TRUE {
		*destination = parsed
	}
	return accepted
}

func parse_public_key(
	destination Parsed_Public_Key_Destination, storage Storage, source Span,
) (accepted Decision) {
	defer func() { Decision_Invariants(accepted, "parse_public_key.accepted") }()
	Parsed_Public_Key_Destination_Invariants(destination, "parse_public_key.destination")
	Storage_Invariants(storage, "parse_public_key.storage")
	Span_Invariants(source, "parse_public_key.source")
	var identifier, ignored_identifier, key_identifier Identifier
	var content, encoded, tail, ignored, algorithm_span, key_content Span
	accepted = take_element(&identifier, &content, &encoded, &tail, storage, source)
	if accepted != DECISION_TRUE {
		return accepted
	}
	if encoded != source {
		accepted = DECISION_FALSE
		return accepted
	}
	if int(tail.Start) != int(tail.End) {
		accepted = DECISION_FALSE
		return accepted
	}
	var expected Identifier
	sequence_identifier(&expected)
	if identifier_matches(identifier, expected) != DECISION_TRUE {
		accepted = DECISION_FALSE
		return accepted
	}
	accepted = take_element(
		&ignored_identifier, &ignored, &algorithm_span, &content, storage, content,
	)
	if accepted != DECISION_TRUE {
		return accepted
	}
	accepted = take_element(
		&key_identifier, &key_content, &ignored, &content, storage, content,
	)
	if accepted != DECISION_TRUE {
		return accepted
	}
	if int(content.Start) != int(content.End) {
		accepted = DECISION_FALSE
		return accepted
	}
	var key Span
	accepted = bit_string_content(&key, storage, key_identifier, key_content)
	if accepted != DECISION_TRUE {
		return accepted
	}
	return public_key_parse(destination, storage, algorithm_span, key)
}

func public_key_parse(
	destination Parsed_Public_Key_Destination,
	storage Storage,
	algorithm Span,
	key Span,
) (accepted Decision) {
	defer func() { Decision_Invariants(accepted, "public_key_parse.accepted") }()
	Parsed_Public_Key_Destination_Invariants(destination, "public_key_parse.destination")
	Storage_Invariants(storage, "public_key_parse.storage")
	Span_Invariants(algorithm, "public_key_parse.algorithm")
	Span_Invariants(key, "public_key_parse.key")
	algorithm_bytes := storage[int(algorithm.Start):int(algorithm.End)]
	key_bytes := storage[int(key.Start):int(key.End)]
	var parsed Parsed_Public_Key
	if string(algorithm_bytes) == RSA_PUBLIC_IDENTIFIER_ENCODING {
		accepted = rsa_public_key_parse(&parsed.RSA_Key, storage, key)
		if accepted != DECISION_TRUE {
			return accepted
		}
		parsed.Algorithm = PUBLIC_KEY_ALGORITHM_RSA
		*destination = parsed
		return accepted
	}
	if string(algorithm_bytes) == EC_PUBLIC_IDENTIFIER_ENCODING {
		if ecdsa.Public_Key_Set_Bytes(
			&parsed.ECDSA_Key, ecdsa.Public_Key_Unvalidated(key_bytes),
		) !=
			ecdsa.KEY_STATUS_OK {
			return accepted
		}
		parsed.Algorithm = PUBLIC_KEY_ALGORITHM_ECDSA
		accepted = DECISION_TRUE
		*destination = parsed
		return accepted
	}
	if string(algorithm_bytes) == ED25519_IDENTIFIER_ENCODING {
		if len(key_bytes) != ed25519.PUBLIC_KEY_SIZE {
			return accepted
		}
		ed25519_public_key_set(&parsed.Ed25519_Key, ed25519.Public_Key(key_bytes))
		parsed.Algorithm = PUBLIC_KEY_ALGORITHM_ED25519
		accepted = DECISION_TRUE
		*destination = parsed
	}
	return accepted
}

func rsa_public_key_parse(
	destination rsa.Public_Key_Destination, storage Storage, source Span,
) (accepted Decision) {
	defer func() { Decision_Invariants(accepted, "rsa_public_key_parse.accepted") }()
	rsa.Public_Key_Destination_Invariants(destination, "rsa_public_key_parse.destination")
	Storage_Invariants(storage, "rsa_public_key_parse.storage")
	Span_Invariants(source, "rsa_public_key_parse.source")
	var identifier, modulus_identifier, exponent_identifier Identifier
	var content, encoded, tail, modulus_content, exponent_content, ignored Span
	accepted = take_element(&identifier, &content, &encoded, &tail, storage, source)
	if accepted != DECISION_TRUE {
		return accepted
	}
	if encoded != source {
		accepted = DECISION_FALSE
		return accepted
	}
	if int(tail.Start) != int(tail.End) {
		accepted = DECISION_FALSE
		return accepted
	}
	var expected Identifier
	sequence_identifier(&expected)
	if identifier_matches(identifier, expected) != DECISION_TRUE {
		accepted = DECISION_FALSE
		return accepted
	}
	accepted = take_element(
		&modulus_identifier, &modulus_content, &ignored, &content, storage, content,
	)
	if accepted != DECISION_TRUE {
		return accepted
	}
	accepted = take_element(
		&exponent_identifier, &exponent_content, &ignored, &content, storage, content,
	)
	if accepted != DECISION_TRUE {
		return accepted
	}
	if int(content.Start) != int(content.End) {
		accepted = DECISION_FALSE
		return accepted
	}
	if positive_integer_valid(storage, modulus_identifier, modulus_content) != DECISION_TRUE {
		accepted = DECISION_FALSE
		return accepted
	}
	if positive_integer_valid(storage, exponent_identifier, exponent_content) != DECISION_TRUE {
		accepted = DECISION_FALSE
		return accepted
	}
	if rsa_exponent_valid(storage, exponent_content) != DECISION_TRUE {
		accepted = DECISION_FALSE
		return accepted
	}
	if int(modulus_content.End)-int(modulus_content.Start) ==
		rsa.MODULUS_SIZE+binary.UINT_8_SIZE {
		modulus_content.Start += Span_Start(binary.UINT_8_SIZE)
	}
	modulus_bytes := storage[int(modulus_content.Start):int(modulus_content.End)]
	if rsa.Public_Key_Set_Bytes(
		destination, rsa.Modulus_Unvalidated(modulus_bytes),
	) != rsa.KEY_STATUS_OK {
		accepted = DECISION_FALSE
		return accepted
	}
	accepted = DECISION_TRUE
	return accepted
}

func rsa_exponent_valid(storage Storage, content Span) (valid Decision) {
	defer func() { Decision_Invariants(valid, "rsa_exponent_valid.valid") }()
	Storage_Invariants(storage, "rsa_exponent_valid.storage")
	Span_Invariants(content, "rsa_exponent_valid.content")
	exponent := storage[int(content.Start):int(content.End)]
	if string(exponent) == RSA_EXPONENT_ENCODING {
		valid = DECISION_TRUE
	}
	return valid
}

func parse_name(
	destination Span_Handle, storage Storage, source Span,
) (accepted Decision) {
	defer func() { Decision_Invariants(accepted, "parse_name.accepted") }()
	Span_Handle_Invariants(destination, "parse_name.destination")
	Storage_Invariants(storage, "parse_name.storage")
	Span_Invariants(source, "parse_name.source")
	var common_name Span
	var identifier Identifier
	var content, encoded, tail Span
	accepted = take_element(&identifier, &content, &encoded, &tail, storage, source)
	if accepted != DECISION_TRUE {
		return accepted
	}
	if encoded != source {
		accepted = DECISION_FALSE
		return accepted
	}
	if int(tail.Start) != int(tail.End) {
		accepted = DECISION_FALSE
		return accepted
	}
	var expected Identifier
	sequence_identifier(&expected)
	if identifier_matches(identifier, expected) != DECISION_TRUE {
		accepted = DECISION_FALSE
		return accepted
	}
	for int(content.Start) != int(content.End) {
		var set_identifier Identifier
		var set_content, ignored, next_content Span
		valid := take_element(
			&set_identifier, &set_content, &ignored, &next_content, storage, content,
		)
		if valid != DECISION_TRUE {
			return accepted
		}
		if name_set_valid(
			storage, set_identifier, set_content,
		) != DECISION_TRUE {
			return accepted
		}
		name_common_name(&common_name, storage, set_content)
		content = next_content
	}
	*destination = common_name
	accepted = DECISION_TRUE
	return accepted
}

func name_common_name(destination Span_Handle, storage Storage, set_content Span) {
	Span_Handle_Invariants(destination, "name_common_name.destination")
	Storage_Invariants(storage, "name_common_name.storage")
	Span_Invariants(set_content, "name_common_name.set_content")
	var ignored_identifier Identifier
	var attribute_content, attribute_encoded, attribute_tail Span
	take_element(
		&ignored_identifier,
		&attribute_content,
		&attribute_encoded,
		&attribute_tail,
		storage,
		set_content,
	)
	var oid_content, oid_encoded, value_source Span
	take_element(
		&ignored_identifier,
		&oid_content,
		&oid_encoded,
		&value_source,
		storage,
		attribute_content,
	)
	var value_content, value_encoded, value_tail Span
	take_element(
		&ignored_identifier,
		&value_content,
		&value_encoded,
		&value_tail,
		storage,
		value_source,
	)
	oid_bytes := storage[int(oid_content.Start):int(oid_content.End)]
	if string(oid_bytes) == COMMON_NAME_OID_ENCODING {
		if destination.End == Span_End(ENCODED_SIZE_MINIMUM) {
			*destination = value_content
		}
	}
}

func name_set_valid(
	storage Storage, set_identifier Identifier, set_content Span,
) (accepted Decision) {
	defer func() { Decision_Invariants(accepted, "name_set_valid.accepted") }()
	Storage_Invariants(storage, "name_set_valid.storage")
	Identifier_Invariants(set_identifier, "name_set_valid.set_identifier")
	Span_Invariants(set_content, "name_set_valid.set_content")
	var expected Identifier
	set_identifier_expected(&expected)
	if identifier_matches(set_identifier, expected) != DECISION_TRUE {
		return accepted
	}
	var attribute_identifier Identifier
	var attribute_content, ignored, tail Span
	accepted = take_element(
		&attribute_identifier, &attribute_content, &ignored, &tail, storage, set_content,
	)
	if accepted != DECISION_TRUE {
		return accepted
	}
	if int(tail.Start) != int(tail.End) {
		return accepted
	}
	sequence_identifier(&expected)
	if identifier_matches(attribute_identifier, expected) != DECISION_TRUE {
		return accepted
	}
	var oid_identifier Identifier
	var oid_content, oid_encoded, value_source Span
	accepted = take_element(
		&oid_identifier,
		&oid_content,
		&oid_encoded,
		&value_source,
		storage,
		attribute_content,
	)
	if accepted != DECISION_TRUE {
		return accepted
	}
	oid_identifier_expected(&expected)
	if identifier_matches(oid_identifier, expected) != DECISION_TRUE {
		return accepted
	}
	var value_identifier Identifier
	var value_content, value_encoded, value_tail Span
	accepted = take_element(
		&value_identifier,
		&value_content,
		&value_encoded,
		&value_tail,
		storage,
		value_source,
	)
	if accepted != DECISION_TRUE {
		return accepted
	}
	if int(value_tail.Start) != int(value_tail.End) {
		return accepted
	}
	accepted = DECISION_TRUE
	return accepted
}

func signature_algorithm_parse(
	storage Storage, algorithm Span,
) (result Signature_Algorithm, recognized Decision) {
	defer func() {
		Signature_Algorithm_Invariants(result, "signature_algorithm_parse.result")
		Decision_Invariants(recognized, "signature_algorithm_parse.recognized")
	}()
	Storage_Invariants(storage, "signature_algorithm_parse.storage")
	Span_Invariants(algorithm, "signature_algorithm_parse.algorithm")
	algorithm_bytes := storage[int(algorithm.Start):int(algorithm.End)]
	if string(algorithm_bytes) == RSA_SIGNATURE_IDENTIFIER_ENCODING {
		result = SIGNATURE_ALGORITHM_RSA_SHA_256
		recognized = DECISION_TRUE
		return result, recognized
	}
	if string(algorithm_bytes) == ECDSA_SIGNATURE_IDENTIFIER_ENCODING {
		result = SIGNATURE_ALGORITHM_ECDSA_SHA_256
		recognized = DECISION_TRUE
		return result, recognized
	}
	if string(algorithm_bytes) == ED25519_IDENTIFIER_ENCODING {
		result = SIGNATURE_ALGORITHM_ED25519
		recognized = DECISION_TRUE
	}
	return result, recognized
}

func take_element(
	identifier Identifier_Handle,
	content Span_Handle,
	encoded Span_Handle,
	tail Span_Handle,
	storage Storage,
	source Span,
) (accepted Decision) {
	defer func() { Decision_Invariants(accepted, "take_element.accepted") }()
	Identifier_Handle_Invariants(identifier, "take_element.identifier")
	Span_Handle_Invariants(content, "take_element.content")
	Span_Handle_Invariants(encoded, "take_element.encoded")
	Span_Handle_Invariants(tail, "take_element.tail")
	Storage_Invariants(storage, "take_element.storage")
	Span_Invariants(source, "take_element.source")
	source_bytes := storage[int(source.Start):int(source.End)]
	element, consumed, _, status := asn1.Decode(asn1.Encoded(source_bytes))
	if status != asn1.STATUS_OK {
		*tail = source
		return accepted
	}
	*identifier = Identifier{
		Class: Identifier_Class(element.Class),
		Tag:   Identifier_Tag(element.Tag),
	}
	if bool(element.Constructed) {
		identifier.Constructed = Constructed(CONSTRUCTED_TRUE)
	}
	encoded.Start = source.Start
	encoded.End = Span_End(int(source.Start) + int(consumed))
	content.End = encoded.End
	content.Start = Span_Start(int(content.End) - len(element.Content))
	tail.Start = Span_Start(encoded.End)
	tail.End = source.End
	accepted = DECISION_TRUE
	return accepted
}

func identifier_matches(
	identifier Identifier, expected Identifier,
) (matches Decision) {
	defer func() { Decision_Invariants(matches, "identifier_matches.matches") }()
	Identifier_Invariants(identifier, "identifier_matches.identifier")
	Identifier_Invariants(expected, "identifier_matches.expected")
	if identifier != expected {
		return matches
	}
	matches = DECISION_TRUE
	return matches
}

func sequence_identifier(destination Identifier_Handle) {
	Identifier_Handle_Invariants(destination, "sequence_identifier.destination")
	*destination = Identifier{
		Class:       Identifier_Class(asn1.CLASS_UNIVERSAL),
		Tag:         Identifier_Tag(TAG_SEQUENCE),
		Constructed: Constructed(CONSTRUCTED_TRUE),
	}
}

func set_identifier_expected(destination Identifier_Handle) {
	Identifier_Handle_Invariants(destination, "set_identifier_expected.destination")
	*destination = Identifier{
		Class:       Identifier_Class(asn1.CLASS_UNIVERSAL),
		Tag:         Identifier_Tag(TAG_SEQUENCE + binary.UINT_8_SIZE),
		Constructed: Constructed(CONSTRUCTED_TRUE),
	}
}

func oid_identifier_expected(destination Identifier_Handle) {
	Identifier_Handle_Invariants(destination, "oid_identifier_expected.destination")
	*destination = Identifier{
		Class:       Identifier_Class(asn1.CLASS_UNIVERSAL),
		Tag:         Identifier_Tag(TAG_OBJECT_IDENTIFIER),
		Constructed: Constructed(CONSTRUCTED_FALSE),
	}
}

func bit_string_identifier(destination Identifier_Handle) {
	Identifier_Handle_Invariants(destination, "bit_string_identifier.destination")
	*destination = Identifier{
		Class:       Identifier_Class(asn1.CLASS_UNIVERSAL),
		Tag:         Identifier_Tag(TAG_BIT_STRING),
		Constructed: Constructed(CONSTRUCTED_FALSE),
	}
}

func integer_identifier(destination Identifier_Handle) {
	Identifier_Handle_Invariants(destination, "integer_identifier.destination")
	*destination = Identifier{
		Class:       Identifier_Class(asn1.CLASS_UNIVERSAL),
		Tag:         Identifier_Tag(TAG_INTEGER),
		Constructed: Constructed(CONSTRUCTED_FALSE),
	}
}

func bit_string_content(
	destination Span_Handle, storage Storage, identifier Identifier, content Span,
) (valid Decision) {
	defer func() { Decision_Invariants(valid, "bit_string_content.valid") }()
	Span_Handle_Invariants(destination, "bit_string_content.destination")
	Storage_Invariants(storage, "bit_string_content.storage")
	Identifier_Invariants(identifier, "bit_string_content.identifier")
	Span_Invariants(content, "bit_string_content.content")
	var expected Identifier
	bit_string_identifier(&expected)
	if identifier_matches(identifier, expected) != DECISION_TRUE {
		return valid
	}
	if int(content.End)-int(content.Start) <
		BIT_STRING_UNUSED_SIZE {
		return valid
	}
	if storage[int(content.Start)] != BIT_STRING_UNUSED_EMPTY {
		return valid
	}
	var value Span
	value.Start = content.Start + BIT_STRING_UNUSED_SIZE
	value.End = content.End
	*destination = value
	valid = DECISION_TRUE
	return valid
}

func version_valid(
	storage Storage, identifier Identifier, content Span,
) (valid Decision) {
	defer func() { Decision_Invariants(valid, "version_valid.valid") }()
	Storage_Invariants(storage, "version_valid.storage")
	Identifier_Invariants(identifier, "version_valid.identifier")
	Span_Invariants(content, "version_valid.content")
	expected := Identifier{
		Class:       Identifier_Class(asn1.CLASS_CONTEXT_SPECIFIC),
		Tag:         Identifier_Tag(VERSION_TAG),
		Constructed: Constructed(CONSTRUCTED_TRUE),
	}
	if identifier_matches(identifier, expected) != DECISION_TRUE {
		return valid
	}
	var version_identifier Identifier
	var version_content, encoded, tail Span
	accepted := take_element(
		&version_identifier, &version_content, &encoded, &tail, storage, content,
	)
	if accepted != DECISION_TRUE {
		return valid
	}
	if encoded != content {
		return valid
	}
	if int(tail.Start) != int(tail.End) {
		return valid
	}
	var integer Identifier
	integer_identifier(&integer)
	if identifier_matches(version_identifier, integer) != DECISION_TRUE {
		return valid
	}
	if int(version_content.End)-int(version_content.Start) !=
		binary.UINT_8_SIZE {
		return valid
	}
	if storage[int(version_content.Start)] == VERSION_V3 {
		valid = DECISION_TRUE
	}
	return valid
}

func serial_valid(
	storage Storage, identifier Identifier, content Span,
) (valid Decision) {
	defer func() { Decision_Invariants(valid, "serial_valid.valid") }()
	Storage_Invariants(storage, "serial_valid.storage")
	Identifier_Invariants(identifier, "serial_valid.identifier")
	Span_Invariants(content, "serial_valid.content")
	if positive_integer_valid(
		storage, identifier, content,
	) != DECISION_TRUE {
		return valid
	}
	if int(content.End)-int(content.Start) >
		SERIAL_NUMBER_SIZE_MAXIMUM {
		return valid
	}
	var nonzero byte
	serial_bytes := storage[int(content.Start):int(content.End)]
	for _, value := range serial_bytes {
		nonzero |= value
	}
	if nonzero != bits.WORD_8_MINIMUM {
		valid = DECISION_TRUE
	}
	return valid
}

func positive_integer_valid(
	storage Storage, identifier Identifier, content Span,
) (valid Decision) {
	defer func() { Decision_Invariants(valid, "positive_integer_valid.valid") }()
	Storage_Invariants(storage, "positive_integer_valid.storage")
	Identifier_Invariants(identifier, "positive_integer_valid.identifier")
	Span_Invariants(content, "positive_integer_valid.content")
	var expected Identifier
	integer_identifier(&expected)
	if identifier_matches(identifier, expected) != DECISION_TRUE {
		return valid
	}
	size := int(content.End) - int(content.Start)
	if size == ENCODED_SIZE_MINIMUM {
		return valid
	}
	bytes := storage[int(content.Start):int(content.End)]
	if bytes[bits.BIT_COUNT_MINIMUM]&
		byte(binary.UINT_8_SIZE<<(bits.BIT_COUNT_8_MAXIMUM-binary.UINT_8_SIZE)) !=
		bits.WORD_8_MINIMUM {
		return valid
	}
	if size == binary.UINT_8_SIZE {
		valid = DECISION_TRUE
		return valid
	}
	if bytes[bits.BIT_COUNT_MINIMUM] != bits.WORD_8_MINIMUM {
		valid = DECISION_TRUE
		return valid
	}
	if bytes[binary.UINT_8_SIZE]&
		byte(binary.UINT_8_SIZE<<(bits.BIT_COUNT_8_MAXIMUM-binary.UINT_8_SIZE)) !=
		bits.WORD_8_MINIMUM {
		valid = DECISION_TRUE
	}
	return valid
}

func validity_valid(
	storage Storage, identifier Identifier, content Span,
) (valid Decision) {
	defer func() { Decision_Invariants(valid, "validity_valid.valid") }()
	Storage_Invariants(storage, "validity_valid.storage")
	Identifier_Invariants(identifier, "validity_valid.identifier")
	Span_Invariants(content, "validity_valid.content")
	var expected Identifier
	sequence_identifier(&expected)
	if identifier_matches(identifier, expected) != DECISION_TRUE {
		return valid
	}
	var not_before_identifier, not_after_identifier Identifier
	var not_before_content, not_after_content, ignored, tail Span
	accepted := take_element(
		&not_before_identifier,
		&not_before_content,
		&ignored,
		&tail,
		storage,
		content,
	)
	if accepted != DECISION_TRUE {
		return valid
	}
	accepted = take_element(
		&not_after_identifier,
		&not_after_content,
		&ignored,
		&tail,
		storage,
		tail,
	)
	if accepted != DECISION_TRUE {
		return valid
	}
	if int(tail.Start) != int(tail.End) {
		return valid
	}
	if time_element_valid(
		not_before_identifier, not_before_content,
	)&time_element_valid(
		not_after_identifier, not_after_content,
	) == DECISION_TRUE {
		valid = DECISION_TRUE
	}
	return valid
}

func time_element_valid(
	identifier Identifier, content Span,
) (valid Decision) {
	defer func() { Decision_Invariants(valid, "time_element_valid.valid") }()
	Identifier_Invariants(identifier, "time_element_valid.identifier")
	Span_Invariants(content, "time_element_valid.content")
	if identifier.Class != Identifier_Class(asn1.CLASS_UNIVERSAL) {
		return valid
	}
	if identifier.Constructed != Constructed(CONSTRUCTED_FALSE) {
		return valid
	}
	if identifier.Tag != Identifier_Tag(TAG_TIME_UTC) {
		if identifier.Tag != Identifier_Tag(TAG_TIME_GENERALIZED) {
			return valid
		}
	}
	if int(content.Start) != int(content.End) {
		valid = DECISION_TRUE
	}
	return valid
}

func optional_fields_valid(
	storage Storage, content Span,
) (valid Decision) {
	defer func() { Decision_Invariants(valid, "optional_fields_valid.valid") }()
	Storage_Invariants(storage, "optional_fields_valid.storage")
	Span_Invariants(content, "optional_fields_valid.content")
	previous_tag := Identifier_Tag(OPTIONAL_TAG_MINIMUM - binary.UINT_8_SIZE)
	for int(content.Start) != int(content.End) {
		var identifier Identifier
		var ignored_content, ignored_encoded, tail Span
		accepted := take_element(
			&identifier,
			&ignored_content,
			&ignored_encoded,
			&tail,
			storage,
			content,
		)
		if accepted != DECISION_TRUE {
			return valid
		}
		if identifier.Class != Identifier_Class(asn1.CLASS_CONTEXT_SPECIFIC) {
			return valid
		}
		tag := identifier.Tag
		if tag < Identifier_Tag(OPTIONAL_TAG_MINIMUM) {
			return valid
		}
		if tag > Identifier_Tag(OPTIONAL_TAG_MAXIMUM) {
			return valid
		}
		if tag <= previous_tag {
			return valid
		}
		if tag == Identifier_Tag(OPTIONAL_TAG_MAXIMUM) {
			if identifier.Constructed != Constructed(CONSTRUCTED_TRUE) {
				return valid
			}
		}
		previous_tag = tag
		content = tail
	}
	valid = DECISION_TRUE
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
	aver.Always(
		certificate.Ready == READY_COMPLETE,
		"X.509 verification requires a parsed certificate.",
	)
	aver.Always(
		issuer.Ready == READY_COMPLETE,
		"X.509 verification requires a parsed issuer.",
	)
	if certificate.Signature_Algorithm == SIGNATURE_ALGORITHM_RSA_SHA_256 {
		if issuer.Public_Key_Algorithm != PUBLIC_KEY_ALGORITHM_RSA {
			return false
		}
		var digest_bytes [rsa.HASH_SIZE]byte
		digest := rsa.Digest(digest_bytes[:])
		digest_sha_256(Digest_Destination(digest), sha256.Source(certificate.TBS))
		return Verification(rsa.Verify_PKCS1_V1_5_SHA_256(
			issuer.RSA_Public_Key, digest,
			rsa.Signature_Unvalidated(certificate.Signature),
		))
	}
	if certificate.Signature_Algorithm == SIGNATURE_ALGORITHM_ECDSA_SHA_256 {
		if issuer.Public_Key_Algorithm != PUBLIC_KEY_ALGORITHM_ECDSA {
			return false
		}
		var signature_storage [ENCODED_SIZE_MAXIMUM]byte
		copy(signature_storage[:], certificate.Signature)
		storage := Storage(signature_storage[:])
		signature_span := Span{
			Start: Span_Start(ENCODED_SIZE_MINIMUM),
			End:   Span_End(len(certificate.Signature)),
		}
		var signature_bytes [ecdsa.SIGNATURE_SIZE]byte
		signature := ecdsa.Signature(signature_bytes[:])
		accepted := ecdsa_signature_parse(
			signature, storage, signature_span,
		)
		if accepted != DECISION_TRUE {
			return false
		}
		var digest_bytes [ecdsa.SCALAR_SIZE]byte
		digest := ecdsa.Digest(digest_bytes[:])
		digest_sha_256(Digest_Destination(digest), sha256.Source(certificate.TBS))
		return Verification(ecdsa.Verify(
			issuer.ECDSA_Public_Key, digest, ecdsa.Signature_Unvalidated(signature),
		))
	}
	if certificate.Signature_Algorithm == SIGNATURE_ALGORITHM_ED25519 {
		if issuer.Public_Key_Algorithm != PUBLIC_KEY_ALGORITHM_ED25519 {
			return false
		}
		var public_key [ed25519.PUBLIC_KEY_SIZE]byte
		ed25519_public_key_copy(
			ed25519.Public_Key(public_key[:]), issuer.Ed25519_Public_Key,
		)
		return Verification(ed25519.Verify(
			ed25519.Public_Key(public_key[:]),
			ed25519.Message(certificate.TBS),
			ed25519.Signature_Unvalidated(certificate.Signature),
		))
	}
	return false
}

// Digest_Destination fixes certificate signature input to SHA-256 width.
type Digest_Destination []byte

// Digest_Destination_Invariants rejects partial signature digests.
func Digest_Destination_Invariants(value Digest_Destination, _ aver.Namespace) {
	aver.Always(
		len(value) == sha256.DIGEST_256_SIZE,
		"X.509 signature digest has SHA-256 width.",
	)
}

func digest_sha_256(destination Digest_Destination, source sha256.Source) {
	Digest_Destination_Invariants(destination, "digest_sha_256.destination")
	sha256.Source_Invariants(source, "digest_sha_256.source")
	count, status := sha256.Checksum_Into(
		sha256.Destination(destination), sha256.KIND_SHA_256, source,
	)
	aver.Always(
		count == sha256.OUTPUT_COUNT_256_REQUIRED,
		"X.509 SHA-256 receives exact digest storage.",
	)
	aver.Always(
		status == sha256.OUTPUT_STATUS_OK,
		"X.509 SHA-256 writes the complete digest.",
	)
}

func ecdsa_signature_parse(
	destination ecdsa.Signature, storage Storage, source Span,
) (accepted Decision) {
	defer func() { Decision_Invariants(accepted, "ecdsa_signature_parse.accepted") }()
	ecdsa.Signature_Invariants(destination, "ecdsa_signature_parse.destination")
	Storage_Invariants(storage, "ecdsa_signature_parse.storage")
	Span_Invariants(source, "ecdsa_signature_parse.source")
	var identifier, r_identifier, s_identifier Identifier
	var content, encoded, tail, r_content, s_content, ignored Span
	accepted = take_element(&identifier, &content, &encoded, &tail, storage, source)
	if accepted != DECISION_TRUE {
		return accepted
	}
	if encoded != source {
		accepted = DECISION_FALSE
		return accepted
	}
	if int(tail.Start) != int(tail.End) {
		accepted = DECISION_FALSE
		return accepted
	}
	var expected Identifier
	sequence_identifier(&expected)
	if identifier_matches(identifier, expected) != DECISION_TRUE {
		accepted = DECISION_FALSE
		return accepted
	}
	accepted = take_element(
		&r_identifier, &r_content, &ignored, &content, storage, content,
	)
	if accepted != DECISION_TRUE {
		return accepted
	}
	accepted = take_element(
		&s_identifier, &s_content, &ignored, &content, storage, content,
	)
	if accepted != DECISION_TRUE {
		return accepted
	}
	if int(content.Start) != int(content.End) {
		accepted = DECISION_FALSE
		return accepted
	}
	var r_bytes [ecdsa.SCALAR_SIZE]byte
	r_scalar := ecdsa.Scalar_Encoding(r_bytes[:])
	accepted = integer_scalar(r_scalar, storage, r_identifier, r_content)
	if accepted != DECISION_TRUE {
		return accepted
	}
	var s_bytes [ecdsa.SCALAR_SIZE]byte
	s_scalar := ecdsa.Scalar_Encoding(s_bytes[:])
	accepted = integer_scalar(s_scalar, storage, s_identifier, s_content)
	if accepted != DECISION_TRUE {
		return accepted
	}
	copy(destination[:ecdsa.SCALAR_SIZE], r_scalar)
	copy(destination[ecdsa.SCALAR_SIZE:], s_scalar)
	accepted = DECISION_TRUE
	return accepted
}

func integer_scalar(
	destination ecdsa.Scalar_Encoding,
	storage Storage,
	identifier Identifier,
	content Span,
) (accepted Decision) {
	defer func() { Decision_Invariants(accepted, "integer_scalar.accepted") }()
	ecdsa.Scalar_Encoding_Invariants(destination, "integer_scalar.destination")
	Storage_Invariants(storage, "integer_scalar.storage")
	Identifier_Invariants(identifier, "integer_scalar.identifier")
	Span_Invariants(content, "integer_scalar.content")
	if positive_integer_valid(
		storage, identifier, content,
	) != DECISION_TRUE {
		return accepted
	}
	if int(content.End)-int(content.Start) ==
		len(destination)+binary.UINT_8_SIZE {
		content.Start += Span_Start(binary.UINT_8_SIZE)
	}
	content_size := int(content.End) - int(content.Start)
	if content_size > len(destination) {
		return accepted
	}
	offset := len(destination) - content_size
	copy(destination[offset:], storage[int(content.Start):int(content.End)])
	accepted = DECISION_TRUE
	return accepted
}
