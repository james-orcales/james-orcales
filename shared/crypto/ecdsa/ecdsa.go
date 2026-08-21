// Package ecdsa implements bounded caller-owned P-256 signatures.
package ecdsa

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/crypto/elliptic"
	"local/james-orcales/shared/crypto/prng"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
)

// SCALAR_SIZE is the P-256 group-order width.
const SCALAR_SIZE = elliptic.SCALAR_SIZE

// SCALAR_LIMB_COUNT derives scalar storage from its byte and machine-word widths.
const SCALAR_LIMB_COUNT = SCALAR_SIZE / binary.UINT_64_SIZE

// PRIVATE_KEY_SIZE is one exact-width canonical P-256 scalar.
const PRIVATE_KEY_SIZE = SCALAR_SIZE

// PUBLIC_KEY_SIZE is one uncompressed SEC 1 P-256 point.
const PUBLIC_KEY_SIZE = elliptic.ENCODING_UNCOMPRESSED_SIZE

// SIGNATURE_SIZE holds fixed-width r and s scalars.
const SIGNATURE_SIZE = SCALAR_SIZE + SCALAR_SIZE

// PRIVATE_KEY_UNVALIDATED_SIZE_MINIMUM admits empty hostile input.
const PRIVATE_KEY_UNVALIDATED_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// PRIVATE_KEY_UNVALIDATED_SIZE_MAXIMUM admits one invalid byte past complete input.
const PRIVATE_KEY_UNVALIDATED_SIZE_MAXIMUM = PRIVATE_KEY_SIZE + binary.UINT_8_SIZE

// PUBLIC_KEY_UNVALIDATED_SIZE_MINIMUM admits empty hostile input.
const PUBLIC_KEY_UNVALIDATED_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// PUBLIC_KEY_UNVALIDATED_SIZE_MAXIMUM admits one invalid byte past complete input.
const PUBLIC_KEY_UNVALIDATED_SIZE_MAXIMUM = PUBLIC_KEY_SIZE + binary.UINT_8_SIZE

// SIGNATURE_UNVALIDATED_SIZE_MINIMUM admits empty hostile input.
const SIGNATURE_UNVALIDATED_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// SIGNATURE_UNVALIDATED_SIZE_MAXIMUM admits one invalid byte past complete input.
const SIGNATURE_UNVALIDATED_SIZE_MAXIMUM = SIGNATURE_SIZE + binary.UINT_8_SIZE

// SIGN_ATTEMPT_MAXIMUM consumes at most one existing CSPRNG output buffer.
const SIGN_ATTEMPT_MAXIMUM = prng.BUFFER_BYTES / SCALAR_SIZE

// KEY_STATUS_OK means a validated key committed.
const KEY_STATUS_OK Key_Status = Key_Status(bits.WORD_8_MINIMUM)

// KEY_STATUS_INPUT_INVALID leaves key storage unchanged.
const KEY_STATUS_INPUT_INVALID Key_Status = KEY_STATUS_OK + binary.UINT_8_SIZE

// SIGN_STATUS_OK means a complete signature committed.
const SIGN_STATUS_OK Sign_Status = Sign_Status(bits.WORD_8_MINIMUM)

// SIGN_STATUS_ENTROPY_EXHAUSTED leaves signature storage unchanged.
const SIGN_STATUS_ENTROPY_EXHAUSTED Sign_Status = SIGN_STATUS_OK + binary.UINT_8_SIZE

// READY_EMPTY marks caller storage that does not yet hold a key.
const READY_EMPTY Ready = Ready(bits.WORD_8_MINIMUM)

// READY_COMPLETE marks one validated key.
const READY_COMPLETE Ready = READY_EMPTY + binary.UINT_8_SIZE

// DECISION_FALSE rejects one constant-time scalar condition.
const DECISION_FALSE Decision = Decision(bits.WORD_64_MINIMUM)

// DECISION_TRUE accepts one constant-time scalar condition.
const DECISION_TRUE Decision = DECISION_FALSE + binary.UINT_8_SIZE

// SCALAR_LIMB_INDEX_MINIMUM identifies least-significant word.
const SCALAR_LIMB_INDEX_MINIMUM Scalar_Limb_Index = Scalar_Limb_Index(bits.WORD_8_MINIMUM)

// SCALAR_LIMB_INDEX_SECOND identifies second word.
const SCALAR_LIMB_INDEX_SECOND = SCALAR_LIMB_INDEX_MINIMUM + binary.UINT_8_SIZE

// SCALAR_LIMB_INDEX_THIRD identifies third word.
const SCALAR_LIMB_INDEX_THIRD = SCALAR_LIMB_INDEX_SECOND + binary.UINT_8_SIZE

// SCALAR_LIMB_INDEX_MAXIMUM identifies most-significant word.
const SCALAR_LIMB_INDEX_MAXIMUM = SCALAR_LIMB_INDEX_THIRD + binary.UINT_8_SIZE

// Scalar_Limb_Index selects one fixed scalar word.
type Scalar_Limb_Index uint8

// Scalar_Limb_Index_Invariants covers every scalar word position.
func Scalar_Limb_Index_Invariants(value Scalar_Limb_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(SCALAR_LIMB_INDEX_MINIMUM),
			uint8(SCALAR_LIMB_INDEX_SECOND), uint8(SCALAR_LIMB_INDEX_THIRD),
			uint8(SCALAR_LIMB_INDEX_MAXIMUM),
		).
		Ensure()
}

// Scalar_Limb_Value carries one arithmetic word across safe accessors.
type Scalar_Limb_Value uint64

// Scalar_Limb_Value_Invariants spans every word value.
func Scalar_Limb_Value_Invariants(value Scalar_Limb_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Scalar_Limb_0 gives the first word one layout identity.
type Scalar_Limb_0 uint64

// Scalar_Limb_0_Invariants spans every stored word.
func Scalar_Limb_0_Invariants(value Scalar_Limb_0, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Scalar_Limb_1 gives the second word one layout identity.
type Scalar_Limb_1 uint64

// Scalar_Limb_1_Invariants spans every stored word.
func Scalar_Limb_1_Invariants(value Scalar_Limb_1, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Scalar_Limb_2 gives the third word one layout identity.
type Scalar_Limb_2 uint64

// Scalar_Limb_2_Invariants spans every stored word.
func Scalar_Limb_2_Invariants(value Scalar_Limb_2, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Scalar_Limb_3 gives the fourth word one layout identity.
type Scalar_Limb_3 uint64

// Scalar_Limb_3_Invariants spans every stored word.
func Scalar_Limb_3_Invariants(value Scalar_Limb_3, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Scalar stores one P-256 group-order integer in little-endian words.
type Scalar struct {
	// Limb_0 keeps the least-significant word stable across arithmetic helpers.
	Limb_0 Scalar_Limb_0
	// Limb_1 prevents layout changes from shifting the second arithmetic word.
	Limb_1 Scalar_Limb_1
	// Limb_2 prevents layout changes from shifting the third arithmetic word.
	Limb_2 Scalar_Limb_2
	// Limb_3 keeps the most-significant word stable across arithmetic helpers.
	Limb_3 Scalar_Limb_3
}

// Scalar_Invariants composes every fixed position once.
func Scalar_Invariants(value Scalar, namespace aver.Namespace) {
	Scalar_Limb_0_Invariants(value.Limb_0, namespace)
	Scalar_Limb_1_Invariants(value.Limb_1, namespace)
	Scalar_Limb_2_Invariants(value.Limb_2, namespace)
	Scalar_Limb_3_Invariants(value.Limb_3, namespace)
}

// Scalar_Handle names mutable scalar storage.
type Scalar_Handle *Scalar

// Scalar_Handle_Invariants composes present scalar storage.
func Scalar_Handle_Invariants(value Scalar_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Scalar_Invariants(*value, namespace)
}

// Scalar_Encoding is one exact-width big-endian scalar.
type Scalar_Encoding []byte

// Scalar_Encoding_Invariants fixes the external scalar width.
func Scalar_Encoding_Invariants(value Scalar_Encoding, _ aver.Namespace) {
	aver.Always(len(value) == SCALAR_SIZE, "Scalar encoding has P-256 width.")
}

// Decision is one constant-time scalar Boolean.
type Decision uint64

// Decision_Invariants admits only rejected and accepted decisions.
func Decision_Invariants(value Decision, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint64(uint64(value), uint64(DECISION_FALSE), uint64(DECISION_TRUE)).
		Ensure()
}

// Ready stores caller key-state identity.
type Ready uint8

// Ready_Invariants admits empty and validated key storage.
func Ready_Invariants(value Ready, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(READY_EMPTY), uint8(READY_COMPLETE)).
		Ensure()
}

// Private_Key stores one canonical scalar and its validation state.
type Private_Key struct {
	// Scalar is the validated secret integer.
	Scalar Scalar
	// Ready proves Scalar passed canonical nonzero validation.
	Ready Ready
}

// Private_Key_Invariants relates completed storage to canonical nonzero scalar form.
func Private_Key_Invariants(value Private_Key, namespace aver.Namespace) {
	Scalar_Invariants(value.Scalar, namespace)
	Ready_Invariants(value.Ready, namespace)
	var order Scalar
	scalar_order(&order)
	var difference Scalar
	canonical := scalar_limbs_subtract(&difference, &value.Scalar, &order)
	nonzero := scalar_is_zero(&value.Scalar)
	valid := canonical & (nonzero ^ DECISION_TRUE)
	aver.Always(
		uint64(value.Ready)&uint64(valid) == uint64(value.Ready),
		"A completed ECDSA private key is canonical and nonzero.",
	)
}

// Private_Key_Destination is nonnil caller-owned key storage.
type Private_Key_Destination *Private_Key

// Private_Key_Destination_Invariants proves caller storage exists.
func Private_Key_Destination_Invariants(
	value Private_Key_Destination, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Private_Key_Invariants(*value, namespace)
}

// Public_Key stores one finite P-256 point and its validation state.
type Public_Key struct {
	// Point is the validated P-256 group element.
	Point elliptic.Point
	// Ready proves Point is finite.
	Ready Ready
}

// Public_Key_Invariants relates completed storage to a finite curve point.
func Public_Key_Invariants(value Public_Key, namespace aver.Namespace) {
	elliptic.Point_Invariants(value.Point, namespace)
	Ready_Invariants(value.Ready, namespace)
	finite_word := uint64(value.Point.Z.Limb_0) |
		uint64(value.Point.Z.Limb_1) |
		uint64(value.Point.Z.Limb_2) |
		uint64(value.Point.Z.Limb_3)
	finite := (finite_word | -finite_word) >>
		(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
	aver.Always(
		uint64(value.Ready)&finite == uint64(value.Ready),
		"A completed ECDSA public key is finite.",
	)
}

// Public_Key_Destination is nonnil caller-owned key storage.
type Public_Key_Destination *Public_Key

// Public_Key_Destination_Invariants proves caller storage exists.
func Public_Key_Destination_Invariants(
	value Public_Key_Destination, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Public_Key_Invariants(*value, namespace)
}

// Public_Key_Encoding is one complete uncompressed SEC 1 point.
type Public_Key_Encoding []byte

// Public_Key_Encoding_Invariants fixes public output width.
func Public_Key_Encoding_Invariants(value Public_Key_Encoding, _ aver.Namespace) {
	aver.Always(
		len(value) == PUBLIC_KEY_SIZE,
		"An ECDSA public key encoding has P-256 width.",
	)
}

// Signature stores fixed-width IEEE P1363 r and s scalars.
type Signature []byte

// Signature_Invariants fixes caller-owned output width.
func Signature_Invariants(value Signature, _ aver.Namespace) {
	aver.Always(len(value) == SIGNATURE_SIZE, "An ECDSA signature has fixed width.")
}

// Private_Key_Unvalidated is one bounded hostile private-key encoding.
type Private_Key_Unvalidated []byte

// Private_Key_Unvalidated_Invariants bounds private-key parsing work.
func Private_Key_Unvalidated_Invariants(
	value Private_Key_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), PRIVATE_KEY_UNVALIDATED_SIZE_MINIMUM,
			PRIVATE_KEY_UNVALIDATED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Public_Key_Unvalidated is one bounded hostile SEC 1 encoding.
type Public_Key_Unvalidated []byte

// Public_Key_Unvalidated_Invariants bounds public-key parsing work.
func Public_Key_Unvalidated_Invariants(
	value Public_Key_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), PUBLIC_KEY_UNVALIDATED_SIZE_MINIMUM,
			PUBLIC_KEY_UNVALIDATED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Signature_Unvalidated is one bounded hostile fixed-width signature.
type Signature_Unvalidated []byte

// Signature_Unvalidated_Invariants bounds verification parsing work.
func Signature_Unvalidated_Invariants(
	value Signature_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), SIGNATURE_UNVALIDATED_SIZE_MINIMUM,
			SIGNATURE_UNVALIDATED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Digest is one SHA-256 result.
type Digest []byte

// Digest_Invariants fixes the P-256 digest width.
func Digest_Invariants(value Digest, _ aver.Namespace) {
	aver.Always(len(value) == SCALAR_SIZE, "An ECDSA digest has SHA-256 width.")
}

// Key_Status reports bounded key parsing.
type Key_Status uint8

// Key_Status_Invariants covers complete and refused key input.
func Key_Status_Invariants(value Key_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(KEY_STATUS_OK), uint8(KEY_STATUS_INPUT_INVALID)).
		Ensure()
}

// Sign_Status reports bounded nonce search.
type Sign_Status uint8

// Sign_Status_Invariants covers committed and exhausted signing.
func Sign_Status_Invariants(value Sign_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(SIGN_STATUS_OK), uint8(SIGN_STATUS_ENTROPY_EXHAUSTED),
		).
		Ensure()
}

// Verification reports signature validity.
type Verification bool

// Verification_Invariants covers accepted and refused signatures.
func Verification_Invariants(value Verification, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "An ECDSA signature verifies.").
		Ensure()
}

// Private_Key_Set_Bytes validates before replacing caller key storage.
func Private_Key_Set_Bytes(
	destination Private_Key_Destination, source Private_Key_Unvalidated,
) (status Key_Status) {
	defer func() {
		Key_Status_Invariants(status, "Private_Key_Set_Bytes.status")
		Private_Key_Invariants(*destination, "Private_Key_Set_Bytes.destination.output")
	}()
	Private_Key_Destination_Invariants(destination, "Private_Key_Set_Bytes.destination")
	Private_Key_Unvalidated_Invariants(source, "Private_Key_Set_Bytes.source")
	if len(source) != PRIVATE_KEY_SIZE {
		return KEY_STATUS_INPUT_INVALID
	}
	encoding := Scalar_Encoding(source)
	valid := scalar_encoding_valid(encoding)
	if valid != DECISION_TRUE {
		return KEY_STATUS_INPUT_INVALID
	}
	var scalar Scalar
	scalar_decode(&scalar, encoding)
	*destination = Private_Key{
		Scalar: scalar, Ready: READY_COMPLETE,
	}
	return KEY_STATUS_OK
}

// Public_Key_From_Private derives the only public point for a validated private scalar.
func Public_Key_From_Private(
	destination Public_Key_Destination, private_key Private_Key,
) {
	defer func() {
		Public_Key_Invariants(*destination, "Public_Key_From_Private.destination.output")
	}()
	Public_Key_Destination_Invariants(
		destination, "Public_Key_From_Private.destination.input",
	)
	Private_Key_Invariants(private_key, "Public_Key_From_Private.private_key")
	aver.Always(
		private_key.Ready == READY_COMPLETE,
		"Public derivation receives a ready ECDSA private key.",
	)
	var scalar_storage [SCALAR_SIZE]byte
	scalar := Scalar_Encoding(scalar_storage[:])
	scalar_encode(scalar, &private_key.Scalar)
	var point elliptic.Point
	status := elliptic.Point_Scalar_Base_Multiply(
		&point, elliptic.Scalar_Unvalidated(scalar),
	)
	aver.Always(status == elliptic.SCALAR_STATUS_OK, "A private scalar has exact width.")
	*destination = Public_Key{Point: point, Ready: READY_COMPLETE}
}

// Public_Key_Set_Bytes validates a finite SEC 1 point before replacement.
func Public_Key_Set_Bytes(
	destination Public_Key_Destination, source Public_Key_Unvalidated,
) (status Key_Status) {
	defer func() {
		Key_Status_Invariants(status, "Public_Key_Set_Bytes.status")
		Public_Key_Invariants(*destination, "Public_Key_Set_Bytes.destination.output")
	}()
	Public_Key_Destination_Invariants(destination, "Public_Key_Set_Bytes.destination")
	Public_Key_Unvalidated_Invariants(source, "Public_Key_Set_Bytes.source")
	if len(source) != PUBLIC_KEY_SIZE {
		return KEY_STATUS_INPUT_INVALID
	}
	var point elliptic.Point
	parse_status := elliptic.Point_Set_Bytes(&point, elliptic.Encoding_Unvalidated(source))
	if parse_status != elliptic.PARSE_STATUS_OK {
		return KEY_STATUS_INPUT_INVALID
	}
	var identity elliptic.Point
	elliptic.Point_Identity(&identity)
	if bool(elliptic.Point_Equal(&point, &identity)) {
		return KEY_STATUS_INPUT_INVALID
	}
	*destination = Public_Key{
		Point: point, Ready: READY_COMPLETE,
	}
	return KEY_STATUS_OK
}

// Public_Key_Bytes_Into writes the complete uncompressed key into fixed caller storage.
func Public_Key_Bytes_Into(
	destination Public_Key_Encoding, public_key Public_Key,
) {
	defer func() {
		Public_Key_Encoding_Invariants(
			destination, "Public_Key_Bytes_Into.destination.output",
		)
	}()
	Public_Key_Encoding_Invariants(destination, "Public_Key_Bytes_Into.destination")
	Public_Key_Invariants(public_key, "Public_Key_Bytes_Into.public_key")
	aver.Always(
		public_key.Ready == READY_COMPLETE,
		"Public encoding receives a ready ECDSA public key.",
	)
	count, status := elliptic.Point_Bytes_Into(
		elliptic.Destination(destination), &public_key.Point,
		elliptic.ENCODING_UNCOMPRESSED,
	)
	aver.Always(
		status == elliptic.OUTPUT_STATUS_OK,
		"Fixed public-key storage receives a complete point.",
	)
	aver.Always(
		count == elliptic.COUNT_UNCOMPRESSED,
		"A finite public key uses uncompressed P-256 width.",
	)
}

// Sign samples at most one bounded entropy window before committing a low-S signature.
func Sign(
	destination Signature,
	generator prng.Source,
	private_key Private_Key,
	digest Digest,
) (status Sign_Status) {
	defer func() {
		Sign_Status_Invariants(status, "Sign.status")
		Signature_Invariants(destination, "Sign.destination.output")
	}()
	Signature_Invariants(destination, "Sign.destination")
	Private_Key_Invariants(private_key, "Sign.private_key")
	Digest_Invariants(digest, "Sign.digest")
	prng.Source_Invariants(generator, "Sign.generator")
	aver.Always(
		private_key.Ready == READY_COMPLETE,
		"Signing receives a ready ECDSA private key.",
	)
	private_scalar := private_key.Scalar
	var digest_scalar Scalar
	scalar_decode_reduce(&digest_scalar, Scalar_Encoding(digest))
	for range SIGN_ATTEMPT_MAXIMUM {
		var nonce_storage [SCALAR_SIZE]byte
		nonce_encoding := Scalar_Encoding(nonce_storage[:])
		prng.Source_Read(generator, prng.Sink(nonce_encoding))
		var candidate_storage [SIGNATURE_SIZE]byte
		candidate := Signature(candidate_storage[:])
		accepted := sign_candidate(
			candidate, &private_scalar, &digest_scalar, nonce_encoding,
		)
		if accepted != DECISION_TRUE {
			continue
		}
		copy(destination, candidate)
		return SIGN_STATUS_OK
	}
	return SIGN_STATUS_ENTROPY_EXHAUSTED
}

// One candidate stays transactional because rare zero scalars require another entropy draw.
func sign_candidate(
	destination Signature,
	private_scalar Scalar_Handle,
	digest_scalar Scalar_Handle,
	nonce_encoding Scalar_Encoding,
) (accepted Decision) {
	defer func() {
		Decision_Invariants(accepted, "sign_candidate.accepted")
	}()
	Signature_Invariants(destination, "sign_candidate.destination")
	Scalar_Handle_Invariants(private_scalar, "sign_candidate.private_scalar")
	Scalar_Handle_Invariants(digest_scalar, "sign_candidate.digest_scalar")
	Scalar_Encoding_Invariants(nonce_encoding, "sign_candidate.nonce_encoding")
	if scalar_encoding_valid(nonce_encoding) != DECISION_TRUE {
		return accepted
	}
	var nonce Scalar
	scalar_decode(&nonce, nonce_encoding)
	var nonce_point elliptic.Point
	elliptic_status := elliptic.Point_Scalar_Base_Multiply(
		&nonce_point, elliptic.Scalar_Unvalidated(nonce_encoding),
	)
	aver.Always(
		elliptic_status == elliptic.SCALAR_STATUS_OK,
		"A nonce candidate has exact P-256 width.",
	)
	var point_encoding [elliptic.ENCODING_UNCOMPRESSED_SIZE]byte
	_, point_status := elliptic.Point_Bytes_Into(
		point_encoding[:], &nonce_point, elliptic.ENCODING_UNCOMPRESSED,
	)
	aver.Always(
		point_status == elliptic.OUTPUT_STATUS_OK,
		"A nonzero nonce produces a finite point.",
	)
	var x_storage [SCALAR_SIZE]byte
	x_encoding := Scalar_Encoding(x_storage[:])
	x_bytes := point_encoding[elliptic.ENCODING_INFINITY_SIZE:elliptic.ENCODING_COMPRESSED_SIZE]
	copy(x_encoding, x_bytes)
	var r Scalar
	scalar_decode_reduce(&r, x_encoding)
	if scalar_is_zero(&r) == DECISION_TRUE {
		return accepted
	}
	var product, sum, inverse, s Scalar
	scalar_multiply(&product, &r, private_scalar)
	scalar_add(&sum, digest_scalar, &product)
	scalar_inverse(&inverse, &nonce)
	scalar_multiply(&s, &inverse, &sum)
	if scalar_is_zero(&s) == DECISION_TRUE {
		return accepted
	}
	var half Scalar
	scalar_half_order(&half)
	greater := scalar_greater(&s, &half)
	var negative Scalar
	scalar_negate(&negative, &s)
	scalar_select(&s, &negative, &s, greater)
	scalar_encode(Scalar_Encoding(destination[:SCALAR_SIZE]), &r)
	scalar_encode(Scalar_Encoding(destination[SCALAR_SIZE:]), &s)
	accepted = DECISION_TRUE
	return accepted
}

// Verify rejects malformed scalar encodings before public group work.
func Verify(
	public_key Public_Key, digest Digest, signature Signature_Unvalidated,
) (verified Verification) {
	defer func() { Verification_Invariants(verified, "Verify.verified") }()
	Public_Key_Invariants(public_key, "Verify.public_key")
	Digest_Invariants(digest, "Verify.digest")
	Signature_Unvalidated_Invariants(signature, "Verify.signature")
	aver.Always(
		public_key.Ready == READY_COMPLETE,
		"Verification receives a ready ECDSA public key.",
	)
	if len(signature) != SIGNATURE_SIZE {
		return false
	}
	r_encoding := Scalar_Encoding(signature[:SCALAR_SIZE])
	s_encoding := Scalar_Encoding(signature[SCALAR_SIZE:])
	if scalar_encoding_valid(r_encoding)&scalar_encoding_valid(s_encoding) !=
		DECISION_TRUE {
		return false
	}
	var r, s, digest_scalar Scalar
	scalar_decode(&r, r_encoding)
	scalar_decode(&s, s_encoding)
	scalar_decode_reduce(&digest_scalar, Scalar_Encoding(digest))
	var inverse, first_scalar, second_scalar Scalar
	scalar_inverse(&inverse, &s)
	scalar_multiply(&first_scalar, &digest_scalar, &inverse)
	scalar_multiply(&second_scalar, &r, &inverse)
	var first_storage, second_storage [SCALAR_SIZE]byte
	first_encoding := Scalar_Encoding(first_storage[:])
	second_encoding := Scalar_Encoding(second_storage[:])
	scalar_encode(first_encoding, &first_scalar)
	scalar_encode(second_encoding, &second_scalar)
	var first, second, sum elliptic.Point
	elliptic.Point_Scalar_Base_Multiply(
		&first, elliptic.Scalar_Unvalidated(first_encoding),
	)
	elliptic.Point_Scalar_Multiply(
		&second, &public_key.Point, elliptic.Scalar_Unvalidated(second_encoding),
	)
	elliptic.Point_Add(&sum, &first, &second)
	var identity elliptic.Point
	elliptic.Point_Identity(&identity)
	if bool(elliptic.Point_Equal(&sum, &identity)) {
		return false
	}
	var point_encoding [elliptic.ENCODING_UNCOMPRESSED_SIZE]byte
	elliptic.Point_Bytes_Into(
		point_encoding[:], &sum, elliptic.ENCODING_UNCOMPRESSED,
	)
	var x_storage [SCALAR_SIZE]byte
	x_encoding := Scalar_Encoding(x_storage[:])
	copy(
		x_encoding,
		point_encoding[elliptic.ENCODING_INFINITY_SIZE:elliptic.ENCODING_COMPRESSED_SIZE],
	)
	var x Scalar
	scalar_decode_reduce(&x, x_encoding)
	equal := scalar_equal(&x, &r)
	return Verification(equal == DECISION_TRUE)
}

func scalar_encoding_valid(
	encoding Scalar_Encoding,
) (valid Decision) {
	defer func() { Decision_Invariants(valid, "scalar_encoding_valid.valid") }()
	Scalar_Encoding_Invariants(encoding, "scalar_encoding_valid.encoding")
	var raw Scalar
	scalar_decode_raw(&raw, encoding)
	var order Scalar
	scalar_order(&order)
	var difference Scalar
	canonical := scalar_limbs_subtract(&difference, &raw, &order)
	nonzero := scalar_is_zero(&raw)
	valid = canonical & (nonzero ^ DECISION_TRUE)
	return valid
}

func scalar_decode(
	destination Scalar_Handle, source Scalar_Encoding,
) {
	Scalar_Handle_Invariants(destination, "scalar_decode.destination")
	Scalar_Encoding_Invariants(source, "scalar_decode.source")
	scalar_decode_raw(destination, source)
}

func scalar_decode_reduce(
	destination Scalar_Handle, source Scalar_Encoding,
) {
	Scalar_Handle_Invariants(destination, "scalar_decode_reduce.destination")
	Scalar_Encoding_Invariants(source, "scalar_decode_reduce.source")
	scalar_decode_raw(destination, source)
	var order Scalar
	scalar_order(&order)
	var reduced Scalar
	borrow := scalar_limbs_subtract(&reduced, destination, &order)
	scalar_select(destination, &reduced, destination, borrow^DECISION_TRUE)
}

func scalar_limb(
	value Scalar_Handle, index Scalar_Limb_Index,
) (limb Scalar_Limb_Value) {
	defer func() { Scalar_Limb_Value_Invariants(limb, "scalar_limb.limb") }()
	Scalar_Handle_Invariants(value, "scalar_limb.value")
	Scalar_Limb_Index_Invariants(index, "scalar_limb.index")
	switch index {
	case bits.BIT_COUNT_MINIMUM:
		return Scalar_Limb_Value(value.Limb_0)
	case binary.UINT_8_SIZE:
		return Scalar_Limb_Value(value.Limb_1)
	case binary.UINT_16_SIZE:
		return Scalar_Limb_Value(value.Limb_2)
	default:
		return Scalar_Limb_Value(value.Limb_3)
	}
}

func scalar_limb_set(
	destination Scalar_Handle, index Scalar_Limb_Index, limb Scalar_Limb_Value,
) {
	Scalar_Handle_Invariants(destination, "scalar_limb_set.destination")
	Scalar_Limb_Index_Invariants(index, "scalar_limb_set.index")
	Scalar_Limb_Value_Invariants(limb, "scalar_limb_set.limb")
	switch index {
	case bits.BIT_COUNT_MINIMUM:
		destination.Limb_0 = Scalar_Limb_0(limb)
	case binary.UINT_8_SIZE:
		destination.Limb_1 = Scalar_Limb_1(limb)
	case binary.UINT_16_SIZE:
		destination.Limb_2 = Scalar_Limb_2(limb)
	default:
		destination.Limb_3 = Scalar_Limb_3(limb)
	}
}

func scalar_decode_raw(
	destination Scalar_Handle, source Scalar_Encoding,
) {
	Scalar_Handle_Invariants(destination, "scalar_decode_raw.destination")
	Scalar_Encoding_Invariants(source, "scalar_decode_raw.source")
	for index := bits.BIT_COUNT_MINIMUM; index < SCALAR_LIMB_COUNT; index++ {
		source_index := SCALAR_SIZE -
			(index+binary.UINT_8_SIZE)*binary.UINT_64_SIZE
		limb := binary.Uint_64(
			binary.Bytes(source[source_index:source_index+binary.UINT_64_SIZE]),
			binary.BIG_ENDIAN,
		)
		scalar_limb_set(destination, Scalar_Limb_Index(index), Scalar_Limb_Value(limb))
	}
}

func scalar_encode(
	destination Scalar_Encoding, source Scalar_Handle,
) {
	Scalar_Encoding_Invariants(destination, "scalar_encode.destination")
	Scalar_Handle_Invariants(source, "scalar_encode.source")
	for index := bits.BIT_COUNT_MINIMUM; index < SCALAR_LIMB_COUNT; index++ {
		destination_index := SCALAR_SIZE -
			(index+binary.UINT_8_SIZE)*binary.UINT_64_SIZE
		destination_end := destination_index + binary.UINT_64_SIZE
		limb := scalar_limb(source, Scalar_Limb_Index(index))
		binary.Put_Uint_64(
			binary.Bytes(destination[destination_index:destination_end]),
			binary.Word_64(limb), binary.BIG_ENDIAN,
		)
	}
}

func scalar_add(
	destination Scalar_Handle,
	left Scalar_Handle,
	right Scalar_Handle,
) {
	Scalar_Handle_Invariants(destination, "scalar_add.destination")
	Scalar_Handle_Invariants(left, "scalar_add.left")
	Scalar_Handle_Invariants(right, "scalar_add.right")
	var sum, reduced Scalar
	carry := scalar_limbs_add(&sum, left, right)
	var order Scalar
	scalar_order(&order)
	borrow := scalar_limbs_subtract(&reduced, &sum, &order)
	scalar_select(
		destination, &reduced, &sum,
		carry|(borrow^DECISION_TRUE),
	)
}

func scalar_multiply(
	destination Scalar_Handle,
	left Scalar_Handle,
	right Scalar_Handle,
) {
	Scalar_Handle_Invariants(destination, "scalar_multiply.destination")
	Scalar_Handle_Invariants(left, "scalar_multiply.left")
	Scalar_Handle_Invariants(right, "scalar_multiply.right")
	var result_words, addend_words, right_words, order_words [SCALAR_LIMB_COUNT]uint64
	var order Scalar
	scalar_order(&order)
	for index := bits.BIT_COUNT_MINIMUM; index < SCALAR_LIMB_COUNT; index++ {
		addend_words[index] = uint64(scalar_limb(left, Scalar_Limb_Index(index)))
		right_words[index] = uint64(scalar_limb(right, Scalar_Limb_Index(index)))
		order_words[index] = uint64(scalar_limb(&order, Scalar_Limb_Index(index)))
	}
	modular_add := func(destination_words, left_words, right_words *[SCALAR_LIMB_COUNT]uint64) {
		carry_value := uint64(bits.WORD_64_MINIMUM)
		var sum [SCALAR_LIMB_COUNT]uint64
		for index := range sum {
			partial := left_words[index] + right_words[index]
			partial_carry := ((left_words[index] & right_words[index]) |
				((left_words[index] | right_words[index]) & ^partial)) >>
				(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
			value := partial + carry_value
			carry_carry := ((partial & carry_value) |
				((partial | carry_value) & ^value)) >>
				(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
			sum[index] = value
			carry_value = partial_carry | carry_carry
		}
		borrow_value := uint64(bits.WORD_64_MINIMUM)
		var reduced [SCALAR_LIMB_COUNT]uint64
		for index := range reduced {
			partial := sum[index] - order_words[index]
			partial_borrow := ((^sum[index] & order_words[index]) |
				(^(sum[index] ^ order_words[index]) & partial)) >>
				(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
			value := partial - borrow_value
			borrow_borrow := ((^partial & borrow_value) |
				(^(partial ^ borrow_value) & value)) >>
				(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
			reduced[index] = value
			borrow_value = partial_borrow | borrow_borrow
		}
		reduce := carry_value | (borrow_value ^ uint64(DECISION_TRUE))
		mask := uint64(bits.WORD_64_MINIMUM) - reduce
		for index := range destination_words {
			destination_words[index] = sum[index] ^
				mask&(reduced[index]^sum[index])
		}
	}
	for bit_index := range SCALAR_SIZE * bits.BIT_COUNT_8_MAXIMUM {
		var candidate [SCALAR_LIMB_COUNT]uint64
		modular_add(&candidate, &result_words, &addend_words)
		word_index := bit_index / bits.BIT_COUNT_64_MAXIMUM
		word_shift := uint(bit_index % bits.BIT_COUNT_64_MAXIMUM)
		bit := right_words[word_index] >> word_shift & binary.UINT_8_SIZE
		mask := uint64(bits.WORD_64_MINIMUM) - bit
		for index := range result_words {
			result_words[index] ^= mask & (candidate[index] ^ result_words[index])
		}
		var doubled [SCALAR_LIMB_COUNT]uint64
		modular_add(&doubled, &addend_words, &addend_words)
		addend_words = doubled
	}
	for index := bits.BIT_COUNT_MINIMUM; index < SCALAR_LIMB_COUNT; index++ {
		limb_index := Scalar_Limb_Index(index)
		limb := Scalar_Limb_Value(result_words[index])
		scalar_limb_set(destination, limb_index, limb)
	}
}

func scalar_inverse(
	destination Scalar_Handle,
	source Scalar_Handle,
) {
	Scalar_Handle_Invariants(destination, "scalar_inverse.destination")
	Scalar_Handle_Invariants(source, "scalar_inverse.source")
	var exponent Scalar
	scalar_order_minus_two(&exponent)
	var result Scalar
	scalar_one(&result)
	scalar_bit_count := SCALAR_SIZE * bits.BIT_COUNT_8_MAXIMUM
	for processed_bit_count := range scalar_bit_count {
		bit_index := scalar_bit_count - processed_bit_count - binary.UINT_8_SIZE
		var square, product Scalar
		scalar_multiply(&square, &result, &result)
		scalar_multiply(&product, &square, source)
		word_index := bit_index / bits.BIT_COUNT_64_MAXIMUM
		word_shift := uint(bit_index % bits.BIT_COUNT_64_MAXIMUM)
		bit := uint64(scalar_limb(&exponent, Scalar_Limb_Index(word_index))) >>
			word_shift & binary.UINT_8_SIZE
		scalar_select(&result, &product, &square, Decision(bit))
	}
	*destination = result
}

func scalar_negate(
	destination Scalar_Handle,
	source Scalar_Handle,
) {
	Scalar_Handle_Invariants(destination, "scalar_negate.destination")
	Scalar_Handle_Invariants(source, "scalar_negate.source")
	var order Scalar
	scalar_order(&order)
	scalar_limbs_subtract(destination, &order, source)
}

func scalar_equal(
	left Scalar_Handle,
	right Scalar_Handle,
) (equal Decision) {
	defer func() { Decision_Invariants(equal, "scalar_equal.equal") }()
	Scalar_Handle_Invariants(left, "scalar_equal.left")
	Scalar_Handle_Invariants(right, "scalar_equal.right")
	difference := uint64(bits.WORD_64_MINIMUM)
	for index := bits.BIT_COUNT_MINIMUM; index < SCALAR_LIMB_COUNT; index++ {
		difference |= uint64(scalar_limb(left, Scalar_Limb_Index(index))) ^
			uint64(scalar_limb(right, Scalar_Limb_Index(index)))
	}
	equal = Decision((difference|-difference)>>
		(bits.BIT_COUNT_64_MAXIMUM-binary.UINT_8_SIZE) ^ binary.UINT_8_SIZE)
	return equal
}

func scalar_is_zero(
	value Scalar_Handle,
) (zero Decision) {
	defer func() { Decision_Invariants(zero, "scalar_is_zero.zero") }()
	Scalar_Handle_Invariants(value, "scalar_is_zero.value")
	var empty Scalar
	return scalar_equal(value, &empty)
}

func scalar_greater(
	left Scalar_Handle,
	right Scalar_Handle,
) (greater Decision) {
	defer func() { Decision_Invariants(greater, "scalar_greater.greater") }()
	Scalar_Handle_Invariants(left, "scalar_greater.left")
	Scalar_Handle_Invariants(right, "scalar_greater.right")
	var difference Scalar
	borrow := scalar_limbs_subtract(&difference, left, right)
	equal := scalar_equal(left, right)
	greater = (borrow ^ DECISION_TRUE) & (equal ^ DECISION_TRUE)
	return greater
}

func scalar_select(
	destination Scalar_Handle,
	first Scalar_Handle,
	second Scalar_Handle,
	choice Decision,
) {
	Scalar_Handle_Invariants(destination, "scalar_select.destination")
	Scalar_Handle_Invariants(first, "scalar_select.first")
	Scalar_Handle_Invariants(second, "scalar_select.second")
	Decision_Invariants(choice, "scalar_select.choice")
	mask := uint64(bits.WORD_64_MINIMUM) - uint64(choice)
	var selected Scalar
	for index := bits.BIT_COUNT_MINIMUM; index < SCALAR_LIMB_COUNT; index++ {
		first_word := uint64(scalar_limb(first, Scalar_Limb_Index(index)))
		second_word := uint64(scalar_limb(second, Scalar_Limb_Index(index)))
		scalar_limb_set(
			&selected, Scalar_Limb_Index(index),
			Scalar_Limb_Value(second_word^mask&(first_word^second_word)),
		)
	}
	*destination = selected
}

func scalar_limbs_add(
	destination Scalar_Handle,
	left Scalar_Handle,
	right Scalar_Handle,
) (carry Decision) {
	defer func() { Decision_Invariants(carry, "scalar_limbs_add.carry") }()
	Scalar_Handle_Invariants(destination, "scalar_limbs_add.destination")
	Scalar_Handle_Invariants(left, "scalar_limbs_add.left")
	Scalar_Handle_Invariants(right, "scalar_limbs_add.right")
	carry_value := uint64(bits.WORD_64_MINIMUM)
	var sum Scalar
	for index := bits.BIT_COUNT_MINIMUM; index < SCALAR_LIMB_COUNT; index++ {
		left_word := uint64(scalar_limb(left, Scalar_Limb_Index(index)))
		right_word := uint64(scalar_limb(right, Scalar_Limb_Index(index)))
		partial := left_word + right_word
		partial_carry := ((left_word & right_word) |
			((left_word | right_word) & ^partial)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		value := partial + carry_value
		carry_carry := ((partial & carry_value) |
			((partial | carry_value) & ^value)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		scalar_limb_set(&sum, Scalar_Limb_Index(index), Scalar_Limb_Value(value))
		carry_value = partial_carry | carry_carry
	}
	*destination = sum
	carry = Decision(carry_value)
	return carry
}

func scalar_limbs_subtract(
	destination Scalar_Handle,
	left Scalar_Handle,
	right Scalar_Handle,
) (borrow Decision) {
	defer func() { Decision_Invariants(borrow, "scalar_limbs_subtract.borrow") }()
	Scalar_Handle_Invariants(destination, "scalar_limbs_subtract.destination")
	Scalar_Handle_Invariants(left, "scalar_limbs_subtract.left")
	Scalar_Handle_Invariants(right, "scalar_limbs_subtract.right")
	borrow_value := uint64(bits.WORD_64_MINIMUM)
	var difference Scalar
	for index := bits.BIT_COUNT_MINIMUM; index < SCALAR_LIMB_COUNT; index++ {
		left_word := uint64(scalar_limb(left, Scalar_Limb_Index(index)))
		right_word := uint64(scalar_limb(right, Scalar_Limb_Index(index)))
		partial := left_word - right_word
		partial_borrow := ((^left_word & right_word) |
			(^(left_word ^ right_word) & partial)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		value := partial - borrow_value
		borrow_borrow := ((^partial & borrow_value) |
			(^(partial ^ borrow_value) & value)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		scalar_limb_set(&difference, Scalar_Limb_Index(index), Scalar_Limb_Value(value))
		borrow_value = partial_borrow | borrow_borrow
	}
	*destination = difference
	borrow = Decision(borrow_value)
	return borrow
}

// These limbs are the SEC 2 P-256 order in little-endian machine order.
func scalar_order(destination Scalar_Handle) {
	Scalar_Handle_Invariants(destination, "scalar_order.destination")
	*destination = Scalar{
		Limb_0: 0xf3b9cac2fc632551,
		Limb_1: 0xbce6faada7179e84,
		Limb_2: 0xffffffffffffffff,
		Limb_3: 0xffffffff00000000,
	}
}

// Fermat inversion uses the group order minus two.
func scalar_order_minus_two(destination Scalar_Handle) {
	Scalar_Handle_Invariants(destination, "scalar_order_minus_two.destination")
	scalar_order(destination)
	destination.Limb_0 -= binary.UINT_16_SIZE
}

func scalar_one(destination Scalar_Handle) {
	Scalar_Handle_Invariants(destination, "scalar_one.destination")
	*destination = Scalar{Limb_0: binary.UINT_8_SIZE}
}

// Low-S normalization derives the inclusive boundary by shifting the odd order.
func scalar_half_order(destination Scalar_Handle) {
	Scalar_Handle_Invariants(destination, "scalar_half_order.destination")
	var order Scalar
	scalar_order(&order)
	for index := bits.BIT_COUNT_MINIMUM; index < SCALAR_LIMB_COUNT; index++ {
		value := uint64(scalar_limb(&order, Scalar_Limb_Index(index))) >> binary.UINT_8_SIZE
		if index+binary.UINT_8_SIZE < SCALAR_LIMB_COUNT {
			value |= uint64(scalar_limb(
				&order, Scalar_Limb_Index(index+binary.UINT_8_SIZE),
			)) <<
				(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		}
		scalar_limb_set(
			destination, Scalar_Limb_Index(index), Scalar_Limb_Value(value),
		)
	}
}
