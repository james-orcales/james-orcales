// Package ed25519 implements bounded caller-owned RFC 8032 signatures.
package ed25519

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/crypto/sha512"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
)

// SEED_SIZE is the RFC 8032 private seed width.
const SEED_SIZE = sha512.DIGEST_256_SIZE

// PRIVATE_KEY_SIZE stores only the authoritative seed.
const PRIVATE_KEY_SIZE = SEED_SIZE

// PUBLIC_KEY_SIZE is one compressed Edwards point.
const PUBLIC_KEY_SIZE = SEED_SIZE

// SIGNATURE_SIZE is one encoded R point and S scalar.
const SIGNATURE_SIZE = sha512.DIGEST_512_SIZE

// MESSAGE_SIZE_MINIMUM admits the RFC 8032 empty message.
const MESSAGE_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// MESSAGE_SIZE_MAXIMUM follows the repository byte-sequence bound.
const MESSAGE_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// SIGNATURE_UNVALIDATED_SIZE_MINIMUM admits empty hostile input.
const SIGNATURE_UNVALIDATED_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// SIGNATURE_UNVALIDATED_SIZE_MAXIMUM admits one invalid byte past complete input.
const SIGNATURE_UNVALIDATED_SIZE_MAXIMUM = SIGNATURE_SIZE + binary.UINT_8_SIZE

// Seed is one exact-width RFC 8032 private input.
type Seed [SEED_SIZE]byte

// Seed_Invariants fixes private input width.
func Seed_Invariants(value Seed, _ invariant.Namespace) {
	invariant.Always(len(value) == SEED_SIZE, "An Ed25519 seed has fixed width.")
}

// Private_Key stores the authoritative seed without redundant public bytes.
type Private_Key [PRIVATE_KEY_SIZE]byte

// Private_Key_Invariants fixes caller-owned private storage width.
func Private_Key_Invariants(value Private_Key, _ invariant.Namespace) {
	invariant.Always(
		len(value) == PRIVATE_KEY_SIZE,
		"An Ed25519 private key has fixed seed width.",
	)
}

// Private_Key_Destination is nonnil caller-owned private storage.
type Private_Key_Destination *Private_Key

// Private_Key_Destination_Invariants proves caller storage exists.
func Private_Key_Destination_Invariants(
	value Private_Key_Destination, _ invariant.Namespace,
) {
	invariant.Always(value != nil, "An Ed25519 private key destination exists.")
}

// Private_Key_Handle is a nonnil caller-owned signing key.
type Private_Key_Handle *Private_Key

// Private_Key_Handle_Invariants proves the key exists and composes its fixed storage.
func Private_Key_Handle_Invariants(
	value Private_Key_Handle, namespace invariant.Namespace,
) {
	invariant.Always(value != nil, "An Ed25519 private key handle exists.")
	Private_Key_Invariants(*value, namespace)
}

// Public_Key is one compressed Edwards point encoding.
type Public_Key [PUBLIC_KEY_SIZE]byte

// Public_Key_Invariants fixes public input width.
func Public_Key_Invariants(value Public_Key, _ invariant.Namespace) {
	invariant.Always(len(value) == PUBLIC_KEY_SIZE, "An Ed25519 public key has fixed width.")
}

// Signature is one fixed-width RFC 8032 signature.
type Signature [SIGNATURE_SIZE]byte

// Signature_Invariants fixes caller-owned signature width.
func Signature_Invariants(value Signature, _ invariant.Namespace) {
	invariant.Always(len(value) == SIGNATURE_SIZE, "An Ed25519 signature has fixed width.")
}

// Signature_Destination is nonnil caller-owned signature storage.
type Signature_Destination *Signature

// Signature_Destination_Invariants proves caller storage exists.
func Signature_Destination_Invariants(
	value Signature_Destination, _ invariant.Namespace,
) {
	invariant.Always(value != nil, "An Ed25519 signature destination exists.")
}

// Message is one bounded message.
type Message []byte

// Message_Invariants bounds both signing passes.
func Message_Invariants(value Message, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), MESSAGE_SIZE_MINIMUM, MESSAGE_SIZE_MAXIMUM).
		Ensure()
}

// Signature_Unvalidated is one bounded hostile signature.
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

// Verification reports signature validity.
type Verification bool

// Verification_Invariants covers accepted and refused signatures.
func Verification_Invariants(value Verification, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "An Ed25519 signature verifies.").
		Ensure()
}

// Private_Key_From_Seed copies exact seed authority into caller storage.
func Private_Key_From_Seed(destination Private_Key_Destination, seed Seed) {
	defer func() {
		Private_Key_Invariants(*destination, "Private_Key_From_Seed.destination.output")
	}()
	Private_Key_Destination_Invariants(destination, "Private_Key_From_Seed.destination")
	Seed_Invariants(seed, "Private_Key_From_Seed.seed")
	copy(destination[:], seed[:])
}

// Public_Key_From_Private derives compressed public bytes without retaining duplicate state.
func Public_Key_From_Private(private_key Private_Key_Handle) (public_key Public_Key) {
	defer func() {
		Public_Key_Invariants(public_key, "Public_Key_From_Private.public_key")
	}()
	Private_Key_Handle_Invariants(private_key, "Public_Key_From_Private.private_key")
	return public_key_derive(private_key)
}

// Sign writes deterministic output only after bounded message validation.
func Sign(
	destination Signature_Destination,
	private_key Private_Key_Handle,
	message Message,
) {
	defer func() { Signature_Invariants(*destination, "Sign.destination.output") }()
	Signature_Destination_Invariants(destination, "Sign.destination")
	Private_Key_Handle_Invariants(private_key, "Sign.private_key")
	Message_Invariants(message, "Sign.message")
	if len(message) > MESSAGE_SIZE_MAXIMUM {
		panic("ed25519: message exceeds bound")
	}
	*destination = signature_create(private_key, message)
}

// Verify rejects hostile sizes before Edwards decoding.
func Verify(
	public_key Public_Key, message Message, signature Signature_Unvalidated,
) (verified Verification) {
	defer func() { Verification_Invariants(verified, "Verify.verified") }()
	Public_Key_Invariants(public_key, "Verify.public_key")
	Message_Invariants(message, "Verify.message")
	Signature_Unvalidated_Invariants(signature, "Verify.signature")
	if len(message) > MESSAGE_SIZE_MAXIMUM {
		panic("ed25519: verification message exceeds bound")
	}
	if len(signature) > SIGNATURE_UNVALIDATED_SIZE_MAXIMUM {
		panic("ed25519: signature exceeds bound")
	}
	if len(signature) != SIGNATURE_SIZE {
		return false
	}
	var encoding [SIGNATURE_SIZE]byte
	copy(encoding[:], signature)
	return signature_verify(public_key, message, &encoding)
}

// FIELD_LIMB_COUNT derives field storage from encoding and machine-word widths.
const FIELD_LIMB_COUNT = SEED_SIZE / binary.UINT_64_SIZE

// FIELD_BIT_COUNT excludes the compressed-sign bit from a field element.
const FIELD_BIT_COUNT = SEED_SIZE*bits.BIT_COUNT_8_MAXIMUM - binary.UINT_8_SIZE

// SCALAR_LIMB_COUNT matches the encoded group-order width.
const SCALAR_LIMB_COUNT = FIELD_LIMB_COUNT

// SCALAR_BIT_COUNT scans one exact Ed25519 scalar encoding.
const SCALAR_BIT_COUNT = SEED_SIZE * bits.BIT_COUNT_8_MAXIMUM

// WIDE_SCALAR_SIZE is one SHA-512 digest reduced modulo the group order.
const WIDE_SCALAR_SIZE = sha512.DIGEST_512_SIZE

// WIDE_SCALAR_BIT_COUNT scans each digest bit during reduction.
const WIDE_SCALAR_BIT_COUNT = WIDE_SCALAR_SIZE * bits.BIT_COUNT_8_MAXIMUM

// POINT_COORDINATE_COUNT stores extended X, Y, Z, and T coordinates.
const POINT_COORDINATE_COUNT = binary.UINT_32_SIZE

// POINT_X_INDEX selects the extended X coordinate.
const POINT_X_INDEX = bits.BIT_COUNT_MINIMUM

// POINT_Y_INDEX selects the extended Y coordinate.
const POINT_Y_INDEX = POINT_X_INDEX + binary.UINT_8_SIZE

// POINT_Z_INDEX selects the extended Z coordinate.
const POINT_Z_INDEX = POINT_Y_INDEX + binary.UINT_8_SIZE

// POINT_T_INDEX selects the extended T coordinate.
const POINT_T_INDEX = POINT_Z_INDEX + binary.UINT_8_SIZE

// CONDITION_LIMB_COUNT gives branchless decisions fixed array identity.
const CONDITION_LIMB_COUNT = binary.UINT_8_SIZE

// COFACTOR_DOUBLING_COUNT derives multiplication by Ed25519's cofactor eight.
const COFACTOR_DOUBLING_COUNT = bits.BIT_COUNT_8_MAXIMUM/binary.UINT_16_SIZE - binary.UINT_8_SIZE

// SCALAR_CLAMP_LOW_CLEAR_MASK clears the three low scalar bits.
const SCALAR_CLAMP_LOW_CLEAR_MASK byte = bits.WORD_8_MAXIMUM ^
	byte((binary.UINT_8_SIZE<<COFACTOR_DOUBLING_COUNT)-binary.UINT_8_SIZE)

// SCALAR_CLAMP_HIGH_CLEAR_MASK clears the two high scalar bits.
const SCALAR_CLAMP_HIGH_CLEAR_MASK byte = bits.WORD_8_MAXIMUM >> binary.UINT_16_SIZE

// SCALAR_CLAMP_HIGH_SET_MASK sets the second-high scalar bit.
const SCALAR_CLAMP_HIGH_SET_MASK byte = binary.UINT_8_SIZE <<
	(bits.BIT_COUNT_8_MAXIMUM - binary.UINT_16_SIZE)

func private_key_expand(
	private_key Private_Key_Handle,
) (scalar [SCALAR_LIMB_COUNT]uint64, prefix [SEED_SIZE]byte) {
	Private_Key_Handle_Invariants(private_key, "private_key_expand.private_key")
	digest := hash_seed(private_key)
	digest[bits.BIT_COUNT_MINIMUM] &= SCALAR_CLAMP_LOW_CLEAR_MASK
	digest[SEED_SIZE-binary.UINT_8_SIZE] &= SCALAR_CLAMP_HIGH_CLEAR_MASK
	digest[SEED_SIZE-binary.UINT_8_SIZE] |= SCALAR_CLAMP_HIGH_SET_MASK
	var wide [WIDE_SCALAR_SIZE]byte
	copy(wide[:SEED_SIZE], digest[:SEED_SIZE])
	scalar_reduce_wide(&scalar, &wide)
	copy(prefix[:], digest[SEED_SIZE:])
	return scalar, prefix
}

func public_key_derive(private_key Private_Key_Handle) (public_key Public_Key) {
	defer func() { Public_Key_Invariants(public_key, "public_key_derive.public_key") }()
	Private_Key_Handle_Invariants(private_key, "public_key_derive.private_key")
	scalar, _ := private_key_expand(private_key)
	var point [POINT_COORDINATE_COUNT][FIELD_LIMB_COUNT]uint64
	point_scalar_base_multiply(&point, &scalar)
	point_encode((*[PUBLIC_KEY_SIZE]byte)(&public_key), &point)
	return public_key
}

func signature_create(
	private_key Private_Key_Handle, message Message,
) (signature Signature) {
	defer func() { Signature_Invariants(signature, "signature_create.signature") }()
	Private_Key_Handle_Invariants(private_key, "signature_create.private_key")
	Message_Invariants(message, "signature_create.message")
	private_scalar, prefix := private_key_expand(private_key)
	public_key := public_key_derive(private_key)
	nonce_digest := hash_nonce(&prefix, message)
	var nonce [SCALAR_LIMB_COUNT]uint64
	scalar_reduce_wide(&nonce, (*[WIDE_SCALAR_SIZE]byte)(&nonce_digest))
	var nonce_point [POINT_COORDINATE_COUNT][FIELD_LIMB_COUNT]uint64
	point_scalar_base_multiply(&nonce_point, &nonce)
	point_encode((*[PUBLIC_KEY_SIZE]byte)(signature[:PUBLIC_KEY_SIZE]), &nonce_point)
	challenge_digest := hash_challenge(
		(*[PUBLIC_KEY_SIZE]byte)(signature[:PUBLIC_KEY_SIZE]), public_key, message,
	)
	var challenge [SCALAR_LIMB_COUNT]uint64
	scalar_reduce_wide(&challenge, (*[WIDE_SCALAR_SIZE]byte)(&challenge_digest))
	var product, response [SCALAR_LIMB_COUNT]uint64
	scalar_multiply(&product, &challenge, &private_scalar)
	scalar_add(&response, &product, &nonce)
	scalar_encode((*[SEED_SIZE]byte)(signature[PUBLIC_KEY_SIZE:]), &response)
	return signature
}

func signature_verify(
	public_key Public_Key, message Message, signature *[SIGNATURE_SIZE]byte,
) (verified Verification) {
	defer func() { Verification_Invariants(verified, "signature_verify.verified") }()
	Public_Key_Invariants(public_key, "signature_verify.public_key")
	Message_Invariants(message, "signature_verify.message")
	var public_point [POINT_COORDINATE_COUNT][FIELD_LIMB_COUNT]uint64
	if point_decode(
		&public_point, (*[PUBLIC_KEY_SIZE]byte)(&public_key),
	)[bits.BIT_COUNT_MINIMUM] != binary.UINT_8_SIZE {
		return false
	}
	if point_has_small_order(&public_point)[bits.BIT_COUNT_MINIMUM] ==
		binary.UINT_8_SIZE {
		return false
	}
	var response [SCALAR_LIMB_COUNT]uint64
	if scalar_decode_canonical(
		&response, (*[SEED_SIZE]byte)(signature[PUBLIC_KEY_SIZE:]),
	)[bits.BIT_COUNT_MINIMUM] != binary.UINT_8_SIZE {
		return false
	}
	challenge_digest := hash_challenge(
		(*[PUBLIC_KEY_SIZE]byte)(signature[:PUBLIC_KEY_SIZE]), public_key, message,
	)
	var challenge [SCALAR_LIMB_COUNT]uint64
	scalar_reduce_wide(&challenge, (*[WIDE_SCALAR_SIZE]byte)(&challenge_digest))
	var response_point, challenge_point [POINT_COORDINATE_COUNT][FIELD_LIMB_COUNT]uint64
	point_scalar_base_multiply(&response_point, &response)
	point_scalar_multiply(&challenge_point, &public_point, &challenge)
	point_negate(&challenge_point, &challenge_point)
	var expected [POINT_COORDINATE_COUNT][FIELD_LIMB_COUNT]uint64
	point_add(&expected, &response_point, &challenge_point)
	var expected_encoding [PUBLIC_KEY_SIZE]byte
	point_encode(&expected_encoding, &expected)
	difference := bits.WORD_8_MINIMUM
	for index := range expected_encoding {
		difference |= expected_encoding[index] ^ signature[index]
	}
	return Verification(difference == bits.WORD_8_MINIMUM)
}

func hash_seed(private_key Private_Key_Handle) (value sha512.Value_512) {
	defer func() { sha512.Value_512_Invariants(value, "hash_seed.value") }()
	Private_Key_Handle_Invariants(private_key, "hash_seed.private_key")
	var digest sha512.Digest
	sha512.Digest_Init(&digest, sha512.KIND_SHA_512)
	sha512.Digest_Write(&digest, private_key[:])
	return sha512.Digest_Sum_512(&digest)
}

func hash_nonce(
	prefix *[SEED_SIZE]byte, message Message,
) (value sha512.Value_512) {
	defer func() { sha512.Value_512_Invariants(value, "hash_nonce.value") }()
	Message_Invariants(message, "hash_nonce.message")
	var digest sha512.Digest
	sha512.Digest_Init(&digest, sha512.KIND_SHA_512)
	sha512.Digest_Write(&digest, prefix[:])
	sha512.Digest_Write(&digest, sha512.Source(message))
	return sha512.Digest_Sum_512(&digest)
}

func hash_challenge(
	encoded_point *[PUBLIC_KEY_SIZE]byte,
	public_key Public_Key,
	message Message,
) (value sha512.Value_512) {
	defer func() { sha512.Value_512_Invariants(value, "hash_challenge.value") }()
	Public_Key_Invariants(public_key, "hash_challenge.public_key")
	Message_Invariants(message, "hash_challenge.message")
	var digest sha512.Digest
	sha512.Digest_Init(&digest, sha512.KIND_SHA_512)
	sha512.Digest_Write(&digest, encoded_point[:])
	sha512.Digest_Write(&digest, public_key[:])
	sha512.Digest_Write(&digest, sha512.Source(message))
	return sha512.Digest_Sum_512(&digest)
}

func point_scalar_base_multiply(
	destination *[POINT_COORDINATE_COUNT][FIELD_LIMB_COUNT]uint64,
	scalar *[SCALAR_LIMB_COUNT]uint64,
) {
	base := point_base()
	point_scalar_multiply(destination, &base, scalar)
}

func point_scalar_multiply(
	destination *[POINT_COORDINATE_COUNT][FIELD_LIMB_COUNT]uint64,
	point *[POINT_COORDINATE_COUNT][FIELD_LIMB_COUNT]uint64,
	scalar *[SCALAR_LIMB_COUNT]uint64,
) {
	result := point_identity()
	for bit_count := SCALAR_BIT_COUNT; bit_count > bits.BIT_COUNT_MINIMUM; bit_count-- {
		bit_index := bit_count - binary.UINT_8_SIZE
		var doubled, added [POINT_COORDINATE_COUNT][FIELD_LIMB_COUNT]uint64
		point_double(&doubled, &result)
		point_add(&added, &doubled, point)
		word_index := bit_index / bits.BIT_COUNT_64_MAXIMUM
		word_shift := uint(bit_index % bits.BIT_COUNT_64_MAXIMUM)
		bit := scalar[word_index] >> word_shift & binary.UINT_8_SIZE
		point_select(
			&result, &added, &doubled,
			[CONDITION_LIMB_COUNT]uint64{bit},
		)
	}
	*destination = result
}

// Extended Edwards addition stays complete for every valid curve pair.
func point_add(
	destination *[POINT_COORDINATE_COUNT][FIELD_LIMB_COUNT]uint64,
	left *[POINT_COORDINATE_COUNT][FIELD_LIMB_COUNT]uint64,
	right *[POINT_COORDINATE_COUNT][FIELD_LIMB_COUNT]uint64,
) {
	var left_difference, right_difference [FIELD_LIMB_COUNT]uint64
	var left_sum, right_sum [FIELD_LIMB_COUNT]uint64
	var a, b, c, d, e, f, g, h [FIELD_LIMB_COUNT]uint64
	field_subtract(&left_difference, &left[POINT_Y_INDEX], &left[POINT_X_INDEX])
	field_subtract(&right_difference, &right[POINT_Y_INDEX], &right[POINT_X_INDEX])
	field_multiply(&a, &left_difference, &right_difference)
	field_add(&left_sum, &left[POINT_Y_INDEX], &left[POINT_X_INDEX])
	field_add(&right_sum, &right[POINT_Y_INDEX], &right[POINT_X_INDEX])
	field_multiply(&b, &left_sum, &right_sum)
	twice_d := field_twice_d()
	field_multiply(&c, &left[POINT_T_INDEX], &right[POINT_T_INDEX])
	field_multiply(&c, &c, &twice_d)
	field_multiply(&d, &left[POINT_Z_INDEX], &right[POINT_Z_INDEX])
	field_add(&d, &d, &d)
	field_subtract(&e, &b, &a)
	field_subtract(&f, &d, &c)
	field_add(&g, &d, &c)
	field_add(&h, &b, &a)
	field_multiply(&destination[POINT_X_INDEX], &e, &f)
	field_multiply(&destination[POINT_Y_INDEX], &g, &h)
	field_multiply(&destination[POINT_T_INDEX], &e, &h)
	field_multiply(&destination[POINT_Z_INDEX], &f, &g)
}

func point_double(
	destination *[POINT_COORDINATE_COUNT][FIELD_LIMB_COUNT]uint64,
	source *[POINT_COORDINATE_COUNT][FIELD_LIMB_COUNT]uint64,
) {
	var a, b, c, d, e, f, g, h [FIELD_LIMB_COUNT]uint64
	field_square(&a, &source[POINT_X_INDEX])
	field_square(&b, &source[POINT_Y_INDEX])
	field_square(&c, &source[POINT_Z_INDEX])
	field_add(&c, &c, &c)
	field_negate(&d, &a)
	field_add(&e, &source[POINT_X_INDEX], &source[POINT_Y_INDEX])
	field_square(&e, &e)
	field_subtract(&e, &e, &a)
	field_subtract(&e, &e, &b)
	field_add(&g, &d, &b)
	field_subtract(&f, &g, &c)
	field_subtract(&h, &d, &b)
	field_multiply(&destination[POINT_X_INDEX], &e, &f)
	field_multiply(&destination[POINT_Y_INDEX], &g, &h)
	field_multiply(&destination[POINT_T_INDEX], &e, &h)
	field_multiply(&destination[POINT_Z_INDEX], &f, &g)
}

func point_negate(
	destination *[POINT_COORDINATE_COUNT][FIELD_LIMB_COUNT]uint64,
	source *[POINT_COORDINATE_COUNT][FIELD_LIMB_COUNT]uint64,
) {
	point := *source
	field_negate(&point[POINT_X_INDEX], &point[POINT_X_INDEX])
	field_negate(&point[POINT_T_INDEX], &point[POINT_T_INDEX])
	*destination = point
}

func point_select(
	destination *[POINT_COORDINATE_COUNT][FIELD_LIMB_COUNT]uint64,
	first *[POINT_COORDINATE_COUNT][FIELD_LIMB_COUNT]uint64,
	second *[POINT_COORDINATE_COUNT][FIELD_LIMB_COUNT]uint64,
	condition [CONDITION_LIMB_COUNT]uint64,
) {
	for coordinate_index := range POINT_COORDINATE_COUNT {
		field_select(
			&destination[coordinate_index], &first[coordinate_index],
			&second[coordinate_index], condition,
		)
	}
}

func point_encode(
	destination *[PUBLIC_KEY_SIZE]byte,
	point *[POINT_COORDINATE_COUNT][FIELD_LIMB_COUNT]uint64,
) {
	var inverse, x, y [FIELD_LIMB_COUNT]uint64
	field_inverse(&inverse, &point[POINT_Z_INDEX])
	field_multiply(&x, &point[POINT_X_INDEX], &inverse)
	field_multiply(&y, &point[POINT_Y_INDEX], &inverse)
	field_encode(destination, &y)
	destination[PUBLIC_KEY_SIZE-binary.UINT_8_SIZE] |=
		byte(x[bits.BIT_COUNT_MINIMUM]&binary.UINT_8_SIZE) <<
			(bits.BIT_COUNT_8_MAXIMUM - binary.UINT_8_SIZE)
}

func point_decode(
	destination *[POINT_COORDINATE_COUNT][FIELD_LIMB_COUNT]uint64,
	encoding *[PUBLIC_KEY_SIZE]byte,
) (valid [CONDITION_LIMB_COUNT]uint64) {
	y_encoding := *encoding
	sign := uint64(
		y_encoding[PUBLIC_KEY_SIZE-binary.UINT_8_SIZE] >>
			(bits.BIT_COUNT_8_MAXIMUM - binary.UINT_8_SIZE),
	)
	y_encoding[PUBLIC_KEY_SIZE-binary.UINT_8_SIZE] &=
		byte(bits.WORD_8_MAXIMUM >> binary.UINT_8_SIZE)
	var y [FIELD_LIMB_COUNT]uint64
	canonical := field_decode(&y, &y_encoding)
	var y_square, numerator, denominator, inverse, x_square, x [FIELD_LIMB_COUNT]uint64
	field_square(&y_square, &y)
	one := field_one()
	field_subtract(&numerator, &y_square, &one)
	d := field_d()
	field_multiply(&denominator, &d, &y_square)
	field_add(&denominator, &denominator, &one)
	field_inverse(&inverse, &denominator)
	field_multiply(&x_square, &numerator, &inverse)
	square := field_square_root(&x, &x_square)
	x_zero := field_is_zero(&x)
	var negative [FIELD_LIMB_COUNT]uint64
	field_negate(&negative, &x)
	field_select(
		&x, &negative, &x,
		[CONDITION_LIMB_COUNT]uint64{
			x[bits.BIT_COUNT_MINIMUM]&binary.UINT_8_SIZE ^ sign,
		},
	)
	valid[bits.BIT_COUNT_MINIMUM] = canonical[bits.BIT_COUNT_MINIMUM] &
		square[bits.BIT_COUNT_MINIMUM] &
		(x_zero[bits.BIT_COUNT_MINIMUM]&sign ^ binary.UINT_8_SIZE)
	destination[POINT_X_INDEX] = x
	destination[POINT_Y_INDEX] = y
	destination[POINT_Z_INDEX] = one
	field_multiply(&destination[POINT_T_INDEX], &x, &y)
	return valid
}

func point_has_small_order(
	point *[POINT_COORDINATE_COUNT][FIELD_LIMB_COUNT]uint64,
) (small [CONDITION_LIMB_COUNT]uint64) {
	multiple := *point
	for range COFACTOR_DOUBLING_COUNT {
		point_double(&multiple, &multiple)
	}
	identity := point_identity()
	return point_equal(&multiple, &identity)
}

func point_equal(
	left *[POINT_COORDINATE_COUNT][FIELD_LIMB_COUNT]uint64,
	right *[POINT_COORDINATE_COUNT][FIELD_LIMB_COUNT]uint64,
) (equal [CONDITION_LIMB_COUNT]uint64) {
	var left_x, right_x, left_y, right_y [FIELD_LIMB_COUNT]uint64
	field_multiply(&left_x, &left[POINT_X_INDEX], &right[POINT_Z_INDEX])
	field_multiply(&right_x, &right[POINT_X_INDEX], &left[POINT_Z_INDEX])
	field_multiply(&left_y, &left[POINT_Y_INDEX], &right[POINT_Z_INDEX])
	field_multiply(&right_y, &right[POINT_Y_INDEX], &left[POINT_Z_INDEX])
	x_equal := field_equal(&left_x, &right_x)
	y_equal := field_equal(&left_y, &right_y)
	equal[bits.BIT_COUNT_MINIMUM] = x_equal[bits.BIT_COUNT_MINIMUM] &
		y_equal[bits.BIT_COUNT_MINIMUM]
	return equal
}

func point_identity() (point [POINT_COORDINATE_COUNT][FIELD_LIMB_COUNT]uint64) {
	point[POINT_Y_INDEX] = field_one()
	point[POINT_Z_INDEX] = field_one()
	return point
}

func point_base() (point [POINT_COORDINATE_COUNT][FIELD_LIMB_COUNT]uint64) {
	encoding := [PUBLIC_KEY_SIZE]byte{
		0x58, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
		0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
		0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
		0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
	}
	valid := point_decode(&point, &encoding)
	invariant.Always(
		valid[bits.BIT_COUNT_MINIMUM] == binary.UINT_8_SIZE,
		"The RFC 8032 base point encoding is valid.",
	)
	return point
}

func field_multiply(
	destination *[FIELD_LIMB_COUNT]uint64,
	left *[FIELD_LIMB_COUNT]uint64,
	right *[FIELD_LIMB_COUNT]uint64,
) {
	var result [FIELD_LIMB_COUNT]uint64
	addend := *left
	for bit_index := bits.BIT_COUNT_MINIMUM; bit_index < FIELD_BIT_COUNT; bit_index++ {
		var candidate [FIELD_LIMB_COUNT]uint64
		field_add(&candidate, &result, &addend)
		word_index := bit_index / bits.BIT_COUNT_64_MAXIMUM
		word_shift := uint(bit_index % bits.BIT_COUNT_64_MAXIMUM)
		bit := right[word_index] >> word_shift & binary.UINT_8_SIZE
		field_select(
			&result, &candidate, &result,
			[CONDITION_LIMB_COUNT]uint64{bit},
		)
		field_add(&addend, &addend, &addend)
	}
	*destination = result
}

func field_square(
	destination *[FIELD_LIMB_COUNT]uint64, source *[FIELD_LIMB_COUNT]uint64,
) {
	field_multiply(destination, source, source)
}

func field_add(
	destination *[FIELD_LIMB_COUNT]uint64,
	left *[FIELD_LIMB_COUNT]uint64,
	right *[FIELD_LIMB_COUNT]uint64,
) {
	var sum, reduced [FIELD_LIMB_COUNT]uint64
	limbs_add(&sum, left, right)
	modulus := field_modulus()
	borrow := limbs_subtract(&reduced, &sum, &modulus)
	field_select(
		destination, &reduced, &sum,
		[CONDITION_LIMB_COUNT]uint64{
			borrow[bits.BIT_COUNT_MINIMUM] ^ binary.UINT_8_SIZE,
		},
	)
}

func field_subtract(
	destination *[FIELD_LIMB_COUNT]uint64,
	left *[FIELD_LIMB_COUNT]uint64,
	right *[FIELD_LIMB_COUNT]uint64,
) {
	var difference, correction [FIELD_LIMB_COUNT]uint64
	borrow := limbs_subtract(&difference, left, right)
	modulus := field_modulus()
	mask := uint64(bits.WORD_64_MINIMUM) - borrow[bits.BIT_COUNT_MINIMUM]
	for index := range correction {
		correction[index] = modulus[index] & mask
	}
	limbs_add(destination, &difference, &correction)
}

func field_negate(
	destination *[FIELD_LIMB_COUNT]uint64, source *[FIELD_LIMB_COUNT]uint64,
) {
	var zero [FIELD_LIMB_COUNT]uint64
	field_subtract(destination, &zero, source)
}

func field_inverse(
	destination *[FIELD_LIMB_COUNT]uint64, source *[FIELD_LIMB_COUNT]uint64,
) {
	exponent := field_inverse_exponent()
	field_power(destination, source, &exponent)
}

func field_square_root(
	destination *[FIELD_LIMB_COUNT]uint64,
	source *[FIELD_LIMB_COUNT]uint64,
) (valid [CONDITION_LIMB_COUNT]uint64) {
	exponent := field_square_root_exponent()
	var candidate, square [FIELD_LIMB_COUNT]uint64
	field_power(&candidate, source, &exponent)
	field_square(&square, &candidate)
	equal := field_equal(&square, source)
	square_root_m1 := field_square_root_minus_one()
	var rotated [FIELD_LIMB_COUNT]uint64
	field_multiply(&rotated, &candidate, &square_root_m1)
	field_select(&candidate, &candidate, &rotated, equal)
	field_square(&square, &candidate)
	valid = field_equal(&square, source)
	*destination = candidate
	return valid
}

func field_power(
	destination *[FIELD_LIMB_COUNT]uint64,
	base *[FIELD_LIMB_COUNT]uint64,
	exponent *[FIELD_LIMB_COUNT]uint64,
) {
	result := field_one()
	for bit_count := FIELD_BIT_COUNT; bit_count > bits.BIT_COUNT_MINIMUM; bit_count-- {
		bit_index := bit_count - binary.UINT_8_SIZE
		var square, product [FIELD_LIMB_COUNT]uint64
		field_square(&square, &result)
		field_multiply(&product, &square, base)
		word_index := bit_index / bits.BIT_COUNT_64_MAXIMUM
		word_shift := uint(bit_index % bits.BIT_COUNT_64_MAXIMUM)
		bit := exponent[word_index] >> word_shift & binary.UINT_8_SIZE
		field_select(
			&result, &product, &square,
			[CONDITION_LIMB_COUNT]uint64{bit},
		)
	}
	*destination = result
}

func field_select(
	destination *[FIELD_LIMB_COUNT]uint64,
	first *[FIELD_LIMB_COUNT]uint64,
	second *[FIELD_LIMB_COUNT]uint64,
	condition [CONDITION_LIMB_COUNT]uint64,
) {
	mask := uint64(bits.WORD_64_MINIMUM) - condition[bits.BIT_COUNT_MINIMUM]
	for index := range destination {
		destination[index] = second[index] ^ mask&(first[index]^second[index])
	}
}

func field_equal(
	left *[FIELD_LIMB_COUNT]uint64, right *[FIELD_LIMB_COUNT]uint64,
) (equal [CONDITION_LIMB_COUNT]uint64) {
	difference := uint64(bits.WORD_64_MINIMUM)
	for index := range left {
		difference |= left[index] ^ right[index]
	}
	equal[bits.BIT_COUNT_MINIMUM] = (difference|-difference)>>
		(bits.BIT_COUNT_64_MAXIMUM-binary.UINT_8_SIZE) ^ binary.UINT_8_SIZE
	return equal
}

func field_is_zero(
	value *[FIELD_LIMB_COUNT]uint64,
) (zero [CONDITION_LIMB_COUNT]uint64) {
	var empty [FIELD_LIMB_COUNT]uint64
	return field_equal(value, &empty)
}

func field_decode(
	destination *[FIELD_LIMB_COUNT]uint64, source *[PUBLIC_KEY_SIZE]byte,
) (canonical [CONDITION_LIMB_COUNT]uint64) {
	for index := bits.BIT_COUNT_MINIMUM; index < FIELD_LIMB_COUNT; index++ {
		source_index := index * binary.UINT_64_SIZE
		destination[index] = uint64(binary.Uint_64(
			source[source_index:source_index+binary.UINT_64_SIZE], binary.LITTLE_ENDIAN,
		))
	}
	modulus := field_modulus()
	var difference [FIELD_LIMB_COUNT]uint64
	return limbs_subtract(&difference, destination, &modulus)
}

func field_encode(
	destination *[PUBLIC_KEY_SIZE]byte, source *[FIELD_LIMB_COUNT]uint64,
) {
	for index := bits.BIT_COUNT_MINIMUM; index < FIELD_LIMB_COUNT; index++ {
		destination_index := index * binary.UINT_64_SIZE
		binary.Put_Uint_64(
			destination[destination_index:destination_index+binary.UINT_64_SIZE],
			binary.Word_64(source[index]), binary.LITTLE_ENDIAN,
		)
	}
}

func scalar_reduce_wide(
	destination *[SCALAR_LIMB_COUNT]uint64, source *[WIDE_SCALAR_SIZE]byte,
) {
	var result [SCALAR_LIMB_COUNT]uint64
	one := scalar_one()
	for bit_count := WIDE_SCALAR_BIT_COUNT; bit_count > bits.BIT_COUNT_MINIMUM; bit_count-- {
		bit_index := bit_count - binary.UINT_8_SIZE
		var doubled, incremented [SCALAR_LIMB_COUNT]uint64
		scalar_add(&doubled, &result, &result)
		scalar_add(&incremented, &doubled, &one)
		byte_index := bit_index / bits.BIT_COUNT_8_MAXIMUM
		byte_shift := uint(bit_index % bits.BIT_COUNT_8_MAXIMUM)
		bit := uint64(source[byte_index]>>byte_shift) & binary.UINT_8_SIZE
		limbs_select(
			&result, &incremented, &doubled,
			[CONDITION_LIMB_COUNT]uint64{bit},
		)
	}
	*destination = result
}

func scalar_decode_canonical(
	destination *[SCALAR_LIMB_COUNT]uint64, source *[SEED_SIZE]byte,
) (canonical [CONDITION_LIMB_COUNT]uint64) {
	for index := bits.BIT_COUNT_MINIMUM; index < SCALAR_LIMB_COUNT; index++ {
		source_index := index * binary.UINT_64_SIZE
		destination[index] = uint64(binary.Uint_64(
			source[source_index:source_index+binary.UINT_64_SIZE], binary.LITTLE_ENDIAN,
		))
	}
	order := scalar_order()
	var difference [SCALAR_LIMB_COUNT]uint64
	return limbs_subtract(&difference, destination, &order)
}

func scalar_encode(
	destination *[SEED_SIZE]byte, source *[SCALAR_LIMB_COUNT]uint64,
) {
	for index := bits.BIT_COUNT_MINIMUM; index < SCALAR_LIMB_COUNT; index++ {
		destination_index := index * binary.UINT_64_SIZE
		binary.Put_Uint_64(
			destination[destination_index:destination_index+binary.UINT_64_SIZE],
			binary.Word_64(source[index]), binary.LITTLE_ENDIAN,
		)
	}
}

func scalar_add(
	destination *[SCALAR_LIMB_COUNT]uint64,
	left *[SCALAR_LIMB_COUNT]uint64,
	right *[SCALAR_LIMB_COUNT]uint64,
) {
	var sum, reduced [SCALAR_LIMB_COUNT]uint64
	limbs_add(&sum, left, right)
	order := scalar_order()
	borrow := limbs_subtract(&reduced, &sum, &order)
	limbs_select(
		destination, &reduced, &sum,
		[CONDITION_LIMB_COUNT]uint64{
			borrow[bits.BIT_COUNT_MINIMUM] ^ binary.UINT_8_SIZE,
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
	for bit_index := bits.BIT_COUNT_MINIMUM; bit_index < SCALAR_BIT_COUNT; bit_index++ {
		var candidate [SCALAR_LIMB_COUNT]uint64
		scalar_add(&candidate, &result, &addend)
		word_index := bit_index / bits.BIT_COUNT_64_MAXIMUM
		word_shift := uint(bit_index % bits.BIT_COUNT_64_MAXIMUM)
		bit := right[word_index] >> word_shift & binary.UINT_8_SIZE
		limbs_select(
			&result, &candidate, &result,
			[CONDITION_LIMB_COUNT]uint64{bit},
		)
		scalar_add(&addend, &addend, &addend)
	}
	*destination = result
}

func limbs_add(
	destination *[FIELD_LIMB_COUNT]uint64,
	left *[FIELD_LIMB_COUNT]uint64,
	right *[FIELD_LIMB_COUNT]uint64,
) (carry [CONDITION_LIMB_COUNT]uint64) {
	carry_value := uint64(bits.WORD_64_MINIMUM)
	for index := range destination {
		partial := left[index] + right[index]
		partial_carry := ((left[index] & right[index]) |
			((left[index] | right[index]) & ^partial)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		value := partial + carry_value
		carry_carry := ((partial & carry_value) |
			((partial | carry_value) & ^value)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		destination[index] = value
		carry_value = partial_carry | carry_carry
	}
	carry[bits.BIT_COUNT_MINIMUM] = carry_value
	return carry
}

func limbs_subtract(
	destination *[FIELD_LIMB_COUNT]uint64,
	left *[FIELD_LIMB_COUNT]uint64,
	right *[FIELD_LIMB_COUNT]uint64,
) (borrow [CONDITION_LIMB_COUNT]uint64) {
	borrow_value := uint64(bits.WORD_64_MINIMUM)
	for index := range destination {
		partial := left[index] - right[index]
		partial_borrow := ((^left[index] & right[index]) |
			(^(left[index] ^ right[index]) & partial)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		value := partial - borrow_value
		borrow_borrow := ((^partial & borrow_value) |
			(^(partial ^ borrow_value) & value)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		destination[index] = value
		borrow_value = partial_borrow | borrow_borrow
	}
	borrow[bits.BIT_COUNT_MINIMUM] = borrow_value
	return borrow
}

func limbs_select(
	destination *[FIELD_LIMB_COUNT]uint64,
	first *[FIELD_LIMB_COUNT]uint64,
	second *[FIELD_LIMB_COUNT]uint64,
	condition [CONDITION_LIMB_COUNT]uint64,
) {
	mask := uint64(bits.WORD_64_MINIMUM) - condition[bits.BIT_COUNT_MINIMUM]
	for index := range destination {
		destination[index] = second[index] ^ mask&(first[index]^second[index])
	}
}

func field_modulus() (value [FIELD_LIMB_COUNT]uint64) {
	return [FIELD_LIMB_COUNT]uint64{
		0xffffffffffffffed,
		0xffffffffffffffff,
		0xffffffffffffffff,
		0x7fffffffffffffff,
	}
}

func field_one() (value [FIELD_LIMB_COUNT]uint64) {
	value[bits.BIT_COUNT_MINIMUM] = binary.UINT_8_SIZE
	return value
}

// The RFC 8032 Edwards parameter is minus 121665 divided by 121666.
func field_d() (value [FIELD_LIMB_COUNT]uint64) {
	return [FIELD_LIMB_COUNT]uint64{
		0x75eb4dca135978a3,
		0x00700a4d4141d8ab,
		0x8cc740797779e898,
		0x52036cee2b6ffe73,
	}
}

func field_twice_d() (value [FIELD_LIMB_COUNT]uint64) {
	d := field_d()
	field_add(&value, &d, &d)
	return value
}

func field_inverse_exponent() (value [FIELD_LIMB_COUNT]uint64) {
	return [FIELD_LIMB_COUNT]uint64{
		0xffffffffffffffeb,
		0xffffffffffffffff,
		0xffffffffffffffff,
		0x7fffffffffffffff,
	}
}

func field_square_root_exponent() (value [FIELD_LIMB_COUNT]uint64) {
	return [FIELD_LIMB_COUNT]uint64{
		0xfffffffffffffffe,
		0xffffffffffffffff,
		0xffffffffffffffff,
		0x0fffffffffffffff,
	}
}

func field_square_root_minus_one() (value [FIELD_LIMB_COUNT]uint64) {
	return [FIELD_LIMB_COUNT]uint64{
		0xc4ee1b274a0ea0b0,
		0x2f431806ad2fe478,
		0x2b4d00993dfbd7a7,
		0x2b8324804fc1df0b,
	}
}

func scalar_order() (value [SCALAR_LIMB_COUNT]uint64) {
	return [SCALAR_LIMB_COUNT]uint64{
		0x5812631a5cf5d3ed,
		0x14def9dea2f79cd6,
		0x0000000000000000,
		0x1000000000000000,
	}
}

func scalar_one() (value [SCALAR_LIMB_COUNT]uint64) {
	value[bits.BIT_COUNT_MINIMUM] = binary.UINT_8_SIZE
	return value
}
