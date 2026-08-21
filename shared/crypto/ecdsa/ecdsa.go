// Package ecdsa implements bounded caller-owned P-256 signatures.
package ecdsa

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/crypto/elliptic"
	"local/james-orcales/shared/crypto/prng"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
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

// READY_INDEX stores caller key-state identity.
const READY_INDEX int = int(bits.WORD_8_MINIMUM)

// READY_WORD_COUNT holds one caller key-state octet.
const READY_WORD_COUNT = READY_INDEX + binary.UINT_8_SIZE

// READY_EMPTY marks caller storage that does not yet hold a key.
const READY_EMPTY byte = bits.WORD_8_MINIMUM

// READY_COMPLETE marks one validated key.
const READY_COMPLETE byte = READY_EMPTY + binary.UINT_8_SIZE

// CONDITION_LIMB_COUNT gives branchless scalar decisions fixed array identity.
const CONDITION_LIMB_COUNT = binary.UINT_8_SIZE

// Ready stores caller key-state identity.
type Ready [READY_WORD_COUNT]byte

// Ready_Invariants fixes key-state storage width.
func Ready_Invariants(value Ready, _ invariant.Namespace) {
	invariant.Always(len(value) == READY_WORD_COUNT, "ECDSA key state has fixed width.")
}

// Private_Key stores one canonical scalar and its validation state.
type Private_Key struct {
	// Scalar is the exact-width big-endian secret.
	Scalar [PRIVATE_KEY_SIZE]byte
	// Ready proves Scalar passed canonical nonzero validation.
	Ready Ready
}

// Private_Key_Invariants relates completed storage to canonical nonzero scalar form.
func Private_Key_Invariants(value Private_Key, namespace invariant.Namespace) {
	invariant.Always(
		len(value.Scalar) == PRIVATE_KEY_SIZE,
		"An ECDSA private scalar has P-256 width.",
	)
	Ready_Invariants(value.Ready, namespace)
	invariant.Always(
		value.Ready[READY_INDEX] <= READY_COMPLETE,
		"An ECDSA private key has empty or complete state.",
	)
	valid := scalar_encoding_valid(&value.Scalar)
	invariant.Always(
		uint64(value.Ready[READY_INDEX])&valid[bits.BIT_COUNT_MINIMUM] ==
			uint64(value.Ready[READY_INDEX]),
		"A completed ECDSA private key is canonical and nonzero.",
	)
}

// Private_Key_Destination is nonnil caller-owned key storage.
type Private_Key_Destination *Private_Key

// Private_Key_Destination_Invariants proves caller storage exists.
func Private_Key_Destination_Invariants(
	value Private_Key_Destination, _ invariant.Namespace,
) {
	invariant.Always(value != nil, "An ECDSA private key destination exists.")
	invariant.Always(
		len(value.Ready) == READY_WORD_COUNT,
		"An ECDSA private key destination has state storage.",
	)
}

// Public_Key stores one finite P-256 point and its validation state.
type Public_Key struct {
	// Point is the validated P-256 group element.
	Point elliptic.Point
	// Ready proves Point is finite.
	Ready Ready
}

// Public_Key_Invariants relates completed storage to a finite curve point.
func Public_Key_Invariants(value Public_Key, namespace invariant.Namespace) {
	elliptic.Point_Invariants(value.Point, namespace)
	Ready_Invariants(value.Ready, namespace)
	invariant.Always(
		value.Ready[READY_INDEX] <= READY_COMPLETE,
		"An ECDSA public key has empty or complete state.",
	)
	finite_word := value.Point.Z[bits.BIT_COUNT_MINIMUM] |
		value.Point.Z[binary.UINT_8_SIZE] |
		value.Point.Z[binary.UINT_16_SIZE] |
		value.Point.Z[binary.UINT_16_SIZE+binary.UINT_8_SIZE]
	finite := (finite_word | -finite_word) >>
		(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
	invariant.Always(
		uint64(value.Ready[READY_INDEX])&finite ==
			uint64(value.Ready[READY_INDEX]),
		"A completed ECDSA public key is finite.",
	)
}

// Public_Key_Destination is nonnil caller-owned key storage.
type Public_Key_Destination *Public_Key

// Public_Key_Destination_Invariants proves caller storage exists.
func Public_Key_Destination_Invariants(
	value Public_Key_Destination, namespace invariant.Namespace,
) {
	invariant.Always(value != nil, "An ECDSA public key destination exists.")
	elliptic.Point_Invariants(value.Point, namespace)
	invariant.Always(
		len(value.Ready) == READY_WORD_COUNT,
		"An ECDSA public key destination has state storage.",
	)
}

// Public_Key_Encoding is one complete uncompressed SEC 1 point.
type Public_Key_Encoding [PUBLIC_KEY_SIZE]byte

// Public_Key_Encoding_Invariants fixes public output width.
func Public_Key_Encoding_Invariants(value Public_Key_Encoding, _ invariant.Namespace) {
	invariant.Always(
		len(value) == PUBLIC_KEY_SIZE,
		"An ECDSA public key encoding has P-256 width.",
	)
}

// Public_Key_Encoding_Destination is nonnil caller-owned encoding storage.
type Public_Key_Encoding_Destination *Public_Key_Encoding

// Public_Key_Encoding_Destination_Invariants proves caller storage exists.
func Public_Key_Encoding_Destination_Invariants(
	value Public_Key_Encoding_Destination, _ invariant.Namespace,
) {
	invariant.Always(value != nil, "An ECDSA public key encoding destination exists.")
}

// Signature stores fixed-width IEEE P1363 r and s scalars.
type Signature [SIGNATURE_SIZE]byte

// Signature_Invariants fixes caller-owned output width.
func Signature_Invariants(value Signature, _ invariant.Namespace) {
	invariant.Always(len(value) == SIGNATURE_SIZE, "An ECDSA signature has fixed width.")
}

// Signature_Destination is nonnil caller-owned signature storage.
type Signature_Destination *Signature

// Signature_Destination_Invariants proves caller storage exists.
func Signature_Destination_Invariants(
	value Signature_Destination, _ invariant.Namespace,
) {
	invariant.Always(value != nil, "An ECDSA signature destination exists.")
}

// Private_Key_Unvalidated is one bounded hostile private-key encoding.
type Private_Key_Unvalidated []byte

// Private_Key_Unvalidated_Invariants bounds private-key parsing work.
func Private_Key_Unvalidated_Invariants(
	value Private_Key_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
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
	value Public_Key_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
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
	value Signature_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), SIGNATURE_UNVALIDATED_SIZE_MINIMUM,
			SIGNATURE_UNVALIDATED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Digest is one SHA-256 result.
type Digest [SCALAR_SIZE]byte

// Digest_Invariants fixes the P-256 digest width.
func Digest_Invariants(value Digest, _ invariant.Namespace) {
	invariant.Always(len(value) == SCALAR_SIZE, "An ECDSA digest has SHA-256 width.")
}

// Key_Status reports bounded key parsing.
type Key_Status uint8

// Key_Status_Invariants covers complete and refused key input.
func Key_Status_Invariants(value Key_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(KEY_STATUS_OK), uint8(KEY_STATUS_INPUT_INVALID)).
		Ensure()
}

// Sign_Status reports bounded nonce search.
type Sign_Status uint8

// Sign_Status_Invariants covers committed and exhausted signing.
func Sign_Status_Invariants(value Sign_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(SIGN_STATUS_OK), uint8(SIGN_STATUS_ENTROPY_EXHAUSTED),
		).
		Ensure()
}

// Verification reports signature validity.
type Verification bool

// Verification_Invariants covers accepted and refused signatures.
func Verification_Invariants(value Verification, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
	if len(source) > PRIVATE_KEY_UNVALIDATED_SIZE_MAXIMUM {
		panic("ecdsa: private key exceeds bound")
	}
	if len(source) != PRIVATE_KEY_SIZE {
		return KEY_STATUS_INPUT_INVALID
	}
	var encoding [PRIVATE_KEY_SIZE]byte
	copy(encoding[:], source)
	valid := scalar_encoding_valid(&encoding)
	if valid[bits.BIT_COUNT_MINIMUM] != binary.UINT_8_SIZE {
		return KEY_STATUS_INPUT_INVALID
	}
	*destination = Private_Key{
		Scalar: encoding, Ready: Ready{READY_COMPLETE},
	}
	return KEY_STATUS_OK
}

// Public_Key_From_Private derives the only public point for a validated private scalar.
func Public_Key_From_Private(private_key Private_Key) (public_key Public_Key) {
	defer func() { Public_Key_Invariants(public_key, "Public_Key_From_Private.public_key") }()
	Private_Key_Invariants(private_key, "Public_Key_From_Private.private_key")
	invariant.Always(
		private_key.Ready[READY_INDEX] == READY_COMPLETE,
		"Public derivation receives a ready ECDSA private key.",
	)
	if private_key.Ready[READY_INDEX] != READY_COMPLETE {
		panic("ecdsa: private key is not ready")
	}
	var point elliptic.Point
	status := elliptic.Point_Scalar_Base_Multiply(&point, private_key.Scalar[:])
	invariant.Always(status == elliptic.SCALAR_STATUS_OK, "A private scalar has exact width.")
	return Public_Key{Point: point, Ready: Ready{READY_COMPLETE}}
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
	if len(source) > PUBLIC_KEY_UNVALIDATED_SIZE_MAXIMUM {
		panic("ecdsa: public key exceeds bound")
	}
	if len(source) != PUBLIC_KEY_SIZE {
		return KEY_STATUS_INPUT_INVALID
	}
	var point elliptic.Point
	parse_status := elliptic.Point_Set_Bytes(&point, elliptic.Encoding_Unvalidated(source))
	if parse_status != elliptic.PARSE_STATUS_OK {
		return KEY_STATUS_INPUT_INVALID
	}
	identity := elliptic.Point_Identity()
	if bool(elliptic.Point_Equal(&point, &identity)) {
		return KEY_STATUS_INPUT_INVALID
	}
	*destination = Public_Key{
		Point: point, Ready: Ready{READY_COMPLETE},
	}
	return KEY_STATUS_OK
}

// Public_Key_Bytes_Into writes the complete uncompressed key into fixed caller storage.
func Public_Key_Bytes_Into(
	destination Public_Key_Encoding_Destination, public_key Public_Key,
) {
	defer func() {
		Public_Key_Encoding_Invariants(
			*destination, "Public_Key_Bytes_Into.destination.output",
		)
	}()
	Public_Key_Encoding_Destination_Invariants(
		destination, "Public_Key_Bytes_Into.destination",
	)
	Public_Key_Invariants(public_key, "Public_Key_Bytes_Into.public_key")
	invariant.Always(
		public_key.Ready[READY_INDEX] == READY_COMPLETE,
		"Public encoding receives a ready ECDSA public key.",
	)
	if public_key.Ready[READY_INDEX] != READY_COMPLETE {
		panic("ecdsa: public key is not ready for encoding")
	}
	count, status := elliptic.Point_Bytes_Into(
		destination[:], &public_key.Point, elliptic.ENCODING_UNCOMPRESSED,
	)
	invariant.Always(
		status == elliptic.OUTPUT_STATUS_OK,
		"Fixed public-key storage receives a complete point.",
	)
	invariant.Always(
		count == elliptic.COUNT_UNCOMPRESSED,
		"A finite public key uses uncompressed P-256 width.",
	)
}

// Sign samples at most one bounded entropy window before committing a low-S signature.
func Sign(
	destination Signature_Destination,
	generator prng.Source,
	private_key Private_Key,
	digest Digest,
) (status Sign_Status) {
	defer func() {
		Sign_Status_Invariants(status, "Sign.status")
		Signature_Invariants(*destination, "Sign.destination.output")
	}()
	Signature_Destination_Invariants(destination, "Sign.destination")
	Private_Key_Invariants(private_key, "Sign.private_key")
	Digest_Invariants(digest, "Sign.digest")
	prng.Source_Invariants(generator, "Sign.generator")
	if generator.State == nil {
		panic("ecdsa: generator is nil")
	}
	invariant.Always(
		private_key.Ready[READY_INDEX] == READY_COMPLETE,
		"Signing receives a ready ECDSA private key.",
	)
	if private_key.Ready[READY_INDEX] != READY_COMPLETE {
		panic("ecdsa: private key is not ready for signing")
	}
	var private_scalar, digest_scalar [SCALAR_LIMB_COUNT]uint64
	scalar_decode(&private_scalar, &private_key.Scalar)
	scalar_decode_reduce(&digest_scalar, (*[SCALAR_SIZE]byte)(&digest))
	for range SIGN_ATTEMPT_MAXIMUM {
		var nonce_encoding [SCALAR_SIZE]byte
		prng.Source_Read(generator, nonce_encoding[:])
		signature, accepted := sign_candidate(
			&private_scalar, &digest_scalar, &nonce_encoding,
		)
		if accepted[bits.BIT_COUNT_MINIMUM] != binary.UINT_8_SIZE {
			continue
		}
		*destination = signature
		return SIGN_STATUS_OK
	}
	return SIGN_STATUS_ENTROPY_EXHAUSTED
}

// One candidate stays transactional because rare zero scalars require another entropy draw.
func sign_candidate(
	private_scalar *[SCALAR_LIMB_COUNT]uint64,
	digest_scalar *[SCALAR_LIMB_COUNT]uint64,
	nonce_encoding *[SCALAR_SIZE]byte,
) (signature Signature, accepted [CONDITION_LIMB_COUNT]uint64) {
	defer func() { Signature_Invariants(signature, "sign_candidate.signature") }()
	if scalar_encoding_valid(nonce_encoding)[bits.BIT_COUNT_MINIMUM] !=
		binary.UINT_8_SIZE {
		return signature, accepted
	}
	var nonce [SCALAR_LIMB_COUNT]uint64
	scalar_decode(&nonce, nonce_encoding)
	var nonce_point elliptic.Point
	elliptic_status := elliptic.Point_Scalar_Base_Multiply(
		&nonce_point, nonce_encoding[:],
	)
	invariant.Always(
		elliptic_status == elliptic.SCALAR_STATUS_OK,
		"A nonce candidate has exact P-256 width.",
	)
	var point_encoding [elliptic.ENCODING_UNCOMPRESSED_SIZE]byte
	_, point_status := elliptic.Point_Bytes_Into(
		point_encoding[:], &nonce_point, elliptic.ENCODING_UNCOMPRESSED,
	)
	invariant.Always(
		point_status == elliptic.OUTPUT_STATUS_OK,
		"A nonzero nonce produces a finite point.",
	)
	var x_encoding [SCALAR_SIZE]byte
	x_bytes := point_encoding[elliptic.ENCODING_INFINITY_SIZE:elliptic.ENCODING_COMPRESSED_SIZE]
	copy(x_encoding[:], x_bytes)
	var r [SCALAR_LIMB_COUNT]uint64
	scalar_decode_reduce(&r, &x_encoding)
	if scalar_is_zero(&r)[bits.BIT_COUNT_MINIMUM] == binary.UINT_8_SIZE {
		return signature, accepted
	}
	var product, sum, inverse, s [SCALAR_LIMB_COUNT]uint64
	scalar_multiply(&product, &r, private_scalar)
	scalar_add(&sum, digest_scalar, &product)
	scalar_inverse(&inverse, &nonce)
	scalar_multiply(&s, &inverse, &sum)
	if scalar_is_zero(&s)[bits.BIT_COUNT_MINIMUM] == binary.UINT_8_SIZE {
		return signature, accepted
	}
	half := scalar_half_order()
	greater := scalar_greater(&s, &half)
	var negative [SCALAR_LIMB_COUNT]uint64
	scalar_negate(&negative, &s)
	scalar_select(&s, &negative, &s, greater)
	scalar_encode((*[SCALAR_SIZE]byte)(signature[:SCALAR_SIZE]), &r)
	scalar_encode((*[SCALAR_SIZE]byte)(signature[SCALAR_SIZE:]), &s)
	accepted[bits.BIT_COUNT_MINIMUM] = binary.UINT_8_SIZE
	return signature, accepted
}

// Verify rejects malformed scalar encodings before public group work.
func Verify(
	public_key Public_Key, digest Digest, signature Signature_Unvalidated,
) (verified Verification) {
	defer func() { Verification_Invariants(verified, "Verify.verified") }()
	Public_Key_Invariants(public_key, "Verify.public_key")
	Digest_Invariants(digest, "Verify.digest")
	Signature_Unvalidated_Invariants(signature, "Verify.signature")
	invariant.Always(
		public_key.Ready[READY_INDEX] == READY_COMPLETE,
		"Verification receives a ready ECDSA public key.",
	)
	if public_key.Ready[READY_INDEX] != READY_COMPLETE {
		panic("ecdsa: public key is not ready for verification")
	}
	if len(signature) > SIGNATURE_UNVALIDATED_SIZE_MAXIMUM {
		panic("ecdsa: signature exceeds bound")
	}
	if len(signature) != SIGNATURE_SIZE {
		return false
	}
	var r_encoding, s_encoding [SCALAR_SIZE]byte
	copy(r_encoding[:], signature[:SCALAR_SIZE])
	copy(s_encoding[:], signature[SCALAR_SIZE:])
	if scalar_encoding_valid(&r_encoding)[bits.BIT_COUNT_MINIMUM]&
		scalar_encoding_valid(&s_encoding)[bits.BIT_COUNT_MINIMUM] !=
		binary.UINT_8_SIZE {
		return false
	}
	var r, s, digest_scalar [SCALAR_LIMB_COUNT]uint64
	scalar_decode(&r, &r_encoding)
	scalar_decode(&s, &s_encoding)
	scalar_decode_reduce(&digest_scalar, (*[SCALAR_SIZE]byte)(&digest))
	var inverse, first_scalar, second_scalar [SCALAR_LIMB_COUNT]uint64
	scalar_inverse(&inverse, &s)
	scalar_multiply(&first_scalar, &digest_scalar, &inverse)
	scalar_multiply(&second_scalar, &r, &inverse)
	var first_encoding, second_encoding [SCALAR_SIZE]byte
	scalar_encode(&first_encoding, &first_scalar)
	scalar_encode(&second_encoding, &second_scalar)
	var first, second, sum elliptic.Point
	elliptic.Point_Scalar_Base_Multiply(&first, first_encoding[:])
	elliptic.Point_Scalar_Multiply(&second, &public_key.Point, second_encoding[:])
	elliptic.Point_Add(&sum, &first, &second)
	identity := elliptic.Point_Identity()
	if bool(elliptic.Point_Equal(&sum, &identity)) {
		return false
	}
	var point_encoding [elliptic.ENCODING_UNCOMPRESSED_SIZE]byte
	elliptic.Point_Bytes_Into(
		point_encoding[:], &sum, elliptic.ENCODING_UNCOMPRESSED,
	)
	var x_encoding [SCALAR_SIZE]byte
	copy(
		x_encoding[:],
		point_encoding[elliptic.ENCODING_INFINITY_SIZE:elliptic.ENCODING_COMPRESSED_SIZE],
	)
	var x [SCALAR_LIMB_COUNT]uint64
	scalar_decode_reduce(&x, &x_encoding)
	equal := scalar_equal(&x, &r)
	return Verification(equal[bits.BIT_COUNT_MINIMUM] == binary.UINT_8_SIZE)
}

func scalar_encoding_valid(
	encoding *[SCALAR_SIZE]byte,
) (valid [CONDITION_LIMB_COUNT]uint64) {
	var raw [SCALAR_LIMB_COUNT]uint64
	scalar_decode_raw(&raw, encoding)
	order := scalar_order()
	var difference [SCALAR_LIMB_COUNT]uint64
	canonical := scalar_limbs_subtract(&difference, &raw, &order)
	nonzero := scalar_is_zero(&raw)
	valid[bits.BIT_COUNT_MINIMUM] = canonical[bits.BIT_COUNT_MINIMUM] &
		(nonzero[bits.BIT_COUNT_MINIMUM] ^ binary.UINT_8_SIZE)
	return valid
}

func scalar_decode(
	destination *[SCALAR_LIMB_COUNT]uint64, source *[SCALAR_SIZE]byte,
) {
	scalar_decode_raw(destination, source)
}

func scalar_decode_reduce(
	destination *[SCALAR_LIMB_COUNT]uint64, source *[SCALAR_SIZE]byte,
) {
	scalar_decode_raw(destination, source)
	order := scalar_order()
	var reduced [SCALAR_LIMB_COUNT]uint64
	borrow := scalar_limbs_subtract(&reduced, destination, &order)
	scalar_select(
		destination, &reduced, destination,
		[CONDITION_LIMB_COUNT]uint64{
			borrow[bits.BIT_COUNT_MINIMUM] ^ binary.UINT_8_SIZE,
		},
	)
}

func scalar_decode_raw(
	destination *[SCALAR_LIMB_COUNT]uint64, source *[SCALAR_SIZE]byte,
) {
	for index := bits.BIT_COUNT_MINIMUM; index < SCALAR_LIMB_COUNT; index++ {
		source_index := SCALAR_SIZE -
			(index+binary.UINT_8_SIZE)*binary.UINT_64_SIZE
		destination[index] = uint64(binary.Uint_64(
			source[source_index:source_index+binary.UINT_64_SIZE], binary.BIG_ENDIAN,
		))
	}
}

func scalar_encode(
	destination *[SCALAR_SIZE]byte, source *[SCALAR_LIMB_COUNT]uint64,
) {
	for index := bits.BIT_COUNT_MINIMUM; index < SCALAR_LIMB_COUNT; index++ {
		destination_index := SCALAR_SIZE -
			(index+binary.UINT_8_SIZE)*binary.UINT_64_SIZE
		binary.Put_Uint_64(
			destination[destination_index:destination_index+binary.UINT_64_SIZE],
			binary.Word_64(source[index]), binary.BIG_ENDIAN,
		)
	}
}

func scalar_add(
	destination *[SCALAR_LIMB_COUNT]uint64,
	left *[SCALAR_LIMB_COUNT]uint64,
	right *[SCALAR_LIMB_COUNT]uint64,
) {
	var sum, reduced [SCALAR_LIMB_COUNT]uint64
	carry := scalar_limbs_add(&sum, left, right)
	order := scalar_order()
	borrow := scalar_limbs_subtract(&reduced, &sum, &order)
	scalar_select(
		destination, &reduced, &sum,
		[CONDITION_LIMB_COUNT]uint64{
			carry[bits.BIT_COUNT_MINIMUM] |
				(borrow[bits.BIT_COUNT_MINIMUM] ^ binary.UINT_8_SIZE),
		},
	)
}

func scalar_multiply(
	destination *[SCALAR_LIMB_COUNT]uint64,
	left *[SCALAR_LIMB_COUNT]uint64,
	right *[SCALAR_LIMB_COUNT]uint64,
) {
	var result [SCALAR_LIMB_COUNT]uint64
	addend := *left
	for bit_index := range SCALAR_SIZE * bits.BIT_COUNT_8_MAXIMUM {
		var candidate [SCALAR_LIMB_COUNT]uint64
		scalar_add(&candidate, &result, &addend)
		word_index := bit_index / bits.BIT_COUNT_64_MAXIMUM
		word_shift := uint(bit_index % bits.BIT_COUNT_64_MAXIMUM)
		bit := right[word_index] >> word_shift & binary.UINT_8_SIZE
		scalar_select(
			&result, &candidate, &result,
			[CONDITION_LIMB_COUNT]uint64{bit},
		)
		scalar_add(&addend, &addend, &addend)
	}
	*destination = result
}

func scalar_inverse(
	destination *[SCALAR_LIMB_COUNT]uint64,
	source *[SCALAR_LIMB_COUNT]uint64,
) {
	exponent := scalar_order_minus_two()
	result := scalar_one()
	scalar_bit_count := SCALAR_SIZE * bits.BIT_COUNT_8_MAXIMUM
	for processed_bit_count := range scalar_bit_count {
		bit_index := scalar_bit_count - processed_bit_count - binary.UINT_8_SIZE
		var square, product [SCALAR_LIMB_COUNT]uint64
		scalar_multiply(&square, &result, &result)
		scalar_multiply(&product, &square, source)
		word_index := bit_index / bits.BIT_COUNT_64_MAXIMUM
		word_shift := uint(bit_index % bits.BIT_COUNT_64_MAXIMUM)
		bit := exponent[word_index] >> word_shift & binary.UINT_8_SIZE
		scalar_select(
			&result, &product, &square,
			[CONDITION_LIMB_COUNT]uint64{bit},
		)
	}
	*destination = result
}

func scalar_negate(
	destination *[SCALAR_LIMB_COUNT]uint64,
	source *[SCALAR_LIMB_COUNT]uint64,
) {
	order := scalar_order()
	scalar_limbs_subtract(destination, &order, source)
}

func scalar_equal(
	left *[SCALAR_LIMB_COUNT]uint64,
	right *[SCALAR_LIMB_COUNT]uint64,
) (equal [CONDITION_LIMB_COUNT]uint64) {
	difference := uint64(bits.WORD_64_MINIMUM)
	for index := range left {
		difference |= left[index] ^ right[index]
	}
	equal[bits.BIT_COUNT_MINIMUM] = (difference|-difference)>>
		(bits.BIT_COUNT_64_MAXIMUM-binary.UINT_8_SIZE) ^ binary.UINT_8_SIZE
	return equal
}

func scalar_is_zero(
	value *[SCALAR_LIMB_COUNT]uint64,
) (zero [CONDITION_LIMB_COUNT]uint64) {
	var empty [SCALAR_LIMB_COUNT]uint64
	return scalar_equal(value, &empty)
}

func scalar_greater(
	left *[SCALAR_LIMB_COUNT]uint64,
	right *[SCALAR_LIMB_COUNT]uint64,
) (greater [CONDITION_LIMB_COUNT]uint64) {
	var difference [SCALAR_LIMB_COUNT]uint64
	borrow := scalar_limbs_subtract(&difference, left, right)
	equal := scalar_equal(left, right)
	greater[bits.BIT_COUNT_MINIMUM] =
		(borrow[bits.BIT_COUNT_MINIMUM] ^ binary.UINT_8_SIZE) &
			(equal[bits.BIT_COUNT_MINIMUM] ^ binary.UINT_8_SIZE)
	return greater
}

func scalar_select(
	destination *[SCALAR_LIMB_COUNT]uint64,
	first *[SCALAR_LIMB_COUNT]uint64,
	second *[SCALAR_LIMB_COUNT]uint64,
	choice [CONDITION_LIMB_COUNT]uint64,
) {
	mask := uint64(bits.WORD_64_MINIMUM) - choice[bits.BIT_COUNT_MINIMUM]
	var selected [SCALAR_LIMB_COUNT]uint64
	for index := range selected {
		selected[index] = second[index] ^ mask&(first[index]^second[index])
	}
	*destination = selected
}

func scalar_limbs_add(
	destination *[SCALAR_LIMB_COUNT]uint64,
	left *[SCALAR_LIMB_COUNT]uint64,
	right *[SCALAR_LIMB_COUNT]uint64,
) (carry [CONDITION_LIMB_COUNT]uint64) {
	carry_value := uint64(bits.WORD_64_MINIMUM)
	var sum [SCALAR_LIMB_COUNT]uint64
	for index := range sum {
		partial := left[index] + right[index]
		partial_carry := ((left[index] & right[index]) |
			((left[index] | right[index]) & ^partial)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		value := partial + carry_value
		carry_carry := ((partial & carry_value) |
			((partial | carry_value) & ^value)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		sum[index] = value
		carry_value = partial_carry | carry_carry
	}
	*destination = sum
	carry[bits.BIT_COUNT_MINIMUM] = carry_value
	return carry
}

func scalar_limbs_subtract(
	destination *[SCALAR_LIMB_COUNT]uint64,
	left *[SCALAR_LIMB_COUNT]uint64,
	right *[SCALAR_LIMB_COUNT]uint64,
) (borrow [CONDITION_LIMB_COUNT]uint64) {
	borrow_value := uint64(bits.WORD_64_MINIMUM)
	var difference [SCALAR_LIMB_COUNT]uint64
	for index := range difference {
		partial := left[index] - right[index]
		partial_borrow := ((^left[index] & right[index]) |
			(^(left[index] ^ right[index]) & partial)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		value := partial - borrow_value
		borrow_borrow := ((^partial & borrow_value) |
			(^(partial ^ borrow_value) & value)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		difference[index] = value
		borrow_value = partial_borrow | borrow_borrow
	}
	*destination = difference
	borrow[bits.BIT_COUNT_MINIMUM] = borrow_value
	return borrow
}

// These limbs are the SEC 2 P-256 order in little-endian machine order.
func scalar_order() (value [SCALAR_LIMB_COUNT]uint64) {
	return [SCALAR_LIMB_COUNT]uint64{
		0xf3b9cac2fc632551,
		0xbce6faada7179e84,
		0xffffffffffffffff,
		0xffffffff00000000,
	}
}

// Fermat inversion uses the group order minus two.
func scalar_order_minus_two() (value [SCALAR_LIMB_COUNT]uint64) {
	value = scalar_order()
	value[bits.BIT_COUNT_MINIMUM] -= binary.UINT_16_SIZE
	return value
}

func scalar_one() (value [SCALAR_LIMB_COUNT]uint64) {
	value[bits.BIT_COUNT_MINIMUM] = binary.UINT_8_SIZE
	return value
}

// Low-S normalization derives the inclusive boundary by shifting the odd order.
func scalar_half_order() (value [SCALAR_LIMB_COUNT]uint64) {
	order := scalar_order()
	for index := bits.BIT_COUNT_MINIMUM; index < SCALAR_LIMB_COUNT; index++ {
		value[index] = order[index] >> binary.UINT_8_SIZE
		if index+binary.UINT_8_SIZE < SCALAR_LIMB_COUNT {
			value[index] |= order[index+binary.UINT_8_SIZE] <<
				(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		}
	}
	return value
}
