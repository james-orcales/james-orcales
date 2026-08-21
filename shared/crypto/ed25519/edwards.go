// Package ed25519 implements bounded caller-owned RFC 8032 signatures.
package ed25519

import (
	"unsafe"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/crypto/sha512"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
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

// Limb_0 gives the first machine word one layout identity.
type Limb_0 uint64

// Limb_0_Invariants spans every stored word.
func Limb_0_Invariants(value Limb_0, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Limb_1 gives the second machine word one layout identity.
type Limb_1 uint64

// Limb_1_Invariants spans every stored word.
func Limb_1_Invariants(value Limb_1, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Limb_2 gives the third machine word one layout identity.
type Limb_2 uint64

// Limb_2_Invariants spans every stored word.
func Limb_2_Invariants(value Limb_2, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Limb_3 gives the fourth machine word one layout identity.
type Limb_3 uint64

// Limb_3_Invariants spans every stored word.
func Limb_3_Invariants(value Limb_3, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Limb_Storage gives field and scalar values stable machine layout.
type Limb_Storage struct {
	// Limb_0 keeps the least-significant word stable under unsafe arithmetic views.
	Limb_0 Limb_0
	// Limb_1 prevents layout changes from shifting the second arithmetic word.
	Limb_1 Limb_1
	// Limb_2 prevents layout changes from shifting the third arithmetic word.
	Limb_2 Limb_2
	// Limb_3 keeps the most-significant word stable under unsafe arithmetic views.
	Limb_3 Limb_3
}

// Limb_Storage_Invariants composes every fixed position once.
func Limb_Storage_Invariants(value Limb_Storage, namespace aver.Namespace) {
	Limb_0_Invariants(value.Limb_0, namespace)
	Limb_1_Invariants(value.Limb_1, namespace)
	Limb_2_Invariants(value.Limb_2, namespace)
	Limb_3_Invariants(value.Limb_3, namespace)
}

// Field is opaque Edwards field arithmetic storage.
type Field Limb_Storage

// Field_Invariants guards unsafe arithmetic views against layout drift.
func Field_Invariants(value Field, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Limb_0), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Limb_1), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Limb_2), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Limb_3), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
	aver.Always(
		unsafe.Sizeof(value) == unsafe.Sizeof(Limb_Storage{}),
		"Field has aligned Ed25519 width.",
	)
}

// Field_Handle names mutable field storage.
type Field_Handle *Field

// Field_Handle_Invariants composes present field storage.
func Field_Handle_Invariants(value Field_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Field_Invariants(*value, namespace)
}

// Scalar is opaque Ed25519 group-order arithmetic storage.
type Scalar Limb_Storage

// Scalar_Invariants guards unsafe arithmetic views against layout drift.
func Scalar_Invariants(value Scalar, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Limb_0), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Limb_1), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Limb_2), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Limb_3), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
	aver.Always(
		unsafe.Sizeof(value) == unsafe.Sizeof(Limb_Storage{}),
		"Scalar has aligned Ed25519 width.",
	)
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

// Limbs is common storage for field and scalar carry arithmetic.
type Limbs Limb_Storage

// Limbs_Invariants guards unsafe arithmetic views against layout drift.
func Limbs_Invariants(value Limbs, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Limb_0), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Limb_1), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Limb_2), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Limb_3), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
	aver.Always(
		unsafe.Sizeof(value) == unsafe.Sizeof(Limb_Storage{}),
		"Limbs have aligned Ed25519 width.",
	)
}

// Limbs_Handle names mutable common arithmetic storage.
type Limbs_Handle *Limbs

// Limbs_Handle_Invariants composes present common storage.
func Limbs_Handle_Invariants(value Limbs_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Limbs_Invariants(*value, namespace)
}

func field_limbs(value Field_Handle) (limbs Limbs_Handle) {
	defer func() { Limbs_Handle_Invariants(limbs, "field_limbs.limbs") }()
	Field_Handle_Invariants(value, "field_limbs.value")
	return Limbs_Handle(unsafe.Pointer(value))
}

func scalar_limbs(value Scalar_Handle) (limbs Limbs_Handle) {
	defer func() { Limbs_Handle_Invariants(limbs, "scalar_limbs.limbs") }()
	Scalar_Handle_Invariants(value, "scalar_limbs.value")
	return Limbs_Handle(unsafe.Pointer(value))
}

// X_Field gives the first extended coordinate a unique invariant identity.
type X_Field Limb_Storage

// X_Field_Invariants guards the shared arithmetic view against layout drift.
func X_Field_Invariants(value X_Field, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Limb_0), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Limb_1), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Limb_2), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Limb_3), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
	aver.Always(
		unsafe.Sizeof(value) == unsafe.Sizeof(Field{}),
		"X keeps field width.",
	)
}

// Y_Field gives the second extended coordinate a unique invariant identity.
type Y_Field Limb_Storage

// Y_Field_Invariants guards the shared arithmetic view against layout drift.
func Y_Field_Invariants(value Y_Field, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Limb_0), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Limb_1), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Limb_2), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Limb_3), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
	aver.Always(
		unsafe.Sizeof(value) == unsafe.Sizeof(Field{}),
		"Y keeps field width.",
	)
}

// Z_Field gives the third extended coordinate a unique invariant identity.
type Z_Field Limb_Storage

// Z_Field_Invariants guards the shared arithmetic view against layout drift.
func Z_Field_Invariants(value Z_Field, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Limb_0), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Limb_1), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Limb_2), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Limb_3), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
	aver.Always(
		unsafe.Sizeof(value) == unsafe.Sizeof(Field{}),
		"Z keeps field width.",
	)
}

// T_Field gives the fourth extended coordinate a unique invariant identity.
type T_Field Limb_Storage

// T_Field_Invariants guards the shared arithmetic view against layout drift.
func T_Field_Invariants(value T_Field, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Limb_0), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Limb_1), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Limb_2), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Limb_3), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
	aver.Always(
		unsafe.Sizeof(value) == unsafe.Sizeof(Field{}),
		"T keeps field width.",
	)
}

// Point_Storage fixes extended coordinate order for complete formulas.
type Point_Storage struct {
	// X keeps the first formula operand at the expected offset.
	X X_Field
	// Y keeps the second formula operand at the expected offset.
	Y Y_Field
	// Z keeps the shared denominator at the expected offset.
	Z Z_Field
	// T keeps the extended product at the expected offset.
	T T_Field
}

// Point_Storage_Invariants composes each coordinate identity once.
func Point_Storage_Invariants(value Point_Storage, namespace aver.Namespace) {
	X_Field_Invariants(value.X, namespace)
	Y_Field_Invariants(value.Y, namespace)
	Z_Field_Invariants(value.Z, namespace)
	T_Field_Invariants(value.T, namespace)
}

// Point is opaque extended Edwards storage.
type Point Point_Storage

// Point_Invariants guards complete-formula layout.
func Point_Invariants(value Point, namespace aver.Namespace) {
	X_Field_Invariants(value.X, namespace)
	Y_Field_Invariants(value.Y, namespace)
	Z_Field_Invariants(value.Z, namespace)
	T_Field_Invariants(value.T, namespace)
	aver.Always(
		unsafe.Sizeof(value) == unsafe.Sizeof(Point_Storage{}),
		"Point preserves extended Edwards layout.",
	)
}

// Point_Handle names mutable extended point storage.
type Point_Handle *Point

// Point_Handle_Invariants composes present point storage.
func Point_Handle_Invariants(value Point_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Point_Invariants(*value, namespace)
}

// Decision is one constant-time Boolean.
type Decision uint64

// Decision_Invariants admits only rejected and accepted decisions.
func Decision_Invariants(value Decision, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint64(uint64(value), uint64(DECISION_FALSE), uint64(DECISION_TRUE)).
		Ensure()
}

// DECISION_FALSE rejects one constant-time condition.
const DECISION_FALSE Decision = Decision(bits.WORD_64_MINIMUM)

// DECISION_TRUE accepts one constant-time condition.
const DECISION_TRUE Decision = DECISION_FALSE + binary.UINT_8_SIZE

// Seed is one exact-width RFC 8032 private input.
type Seed []byte

// Seed_Invariants fixes private input width.
func Seed_Invariants(value Seed, _ aver.Namespace) {
	aver.Always(len(value) == SEED_SIZE, "An Ed25519 seed has fixed width.")
}

// Private_Key stores the authoritative seed without redundant public bytes.
type Private_Key Limb_Storage

// Private_Key_Invariants fixes caller-owned private storage width.
func Private_Key_Invariants(value Private_Key, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Limb_0), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Limb_1), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Limb_2), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.Limb_3), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
	aver.Always(
		unsafe.Sizeof(value) == unsafe.Sizeof(Limb_Storage{}),
		"An Ed25519 private key has aligned seed width.",
	)
}

// Private_Key_Destination is nonnil caller-owned private storage.
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

// Private_Key_Handle is a nonnil caller-owned signing key.
type Private_Key_Handle *Private_Key

// Private_Key_Handle_Invariants proves the key exists and composes its fixed storage.
func Private_Key_Handle_Invariants(
	value Private_Key_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Private_Key_Invariants(*value, namespace)
}

// Private key bytes stay inside the package boundary.
func private_key_seed(value Private_Key_Handle) (seed Seed) {
	defer func() { Seed_Invariants(seed, "private_key_seed.seed") }()
	Private_Key_Handle_Invariants(value, "private_key_seed.value")
	return Seed(unsafe.Slice((*byte)(unsafe.Pointer(value)), PRIVATE_KEY_SIZE))
}

// Public_Key is one compressed Edwards point encoding.
type Public_Key []byte

// Public_Key_Invariants fixes public input width.
func Public_Key_Invariants(value Public_Key, _ aver.Namespace) {
	aver.Always(len(value) == PUBLIC_KEY_SIZE, "An Ed25519 public key has fixed width.")
}

// Signature is one fixed-width RFC 8032 signature.
type Signature []byte

// Signature_Invariants fixes caller-owned signature width.
func Signature_Invariants(value Signature, _ aver.Namespace) {
	aver.Always(len(value) == SIGNATURE_SIZE, "An Ed25519 signature has fixed width.")
}

// Message is one bounded message.
type Message []byte

// Message_Invariants bounds both signing passes.
func Message_Invariants(value Message, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), MESSAGE_SIZE_MINIMUM, MESSAGE_SIZE_MAXIMUM).
		Ensure()
}

// Signature_Unvalidated is one bounded hostile signature.
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

// Verification reports signature validity.
type Verification bool

// Verification_Invariants covers accepted and refused signatures.
func Verification_Invariants(value Verification, namespace aver.Namespace) {
	aver.Tree(value, namespace).
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
	copy(private_key_seed(Private_Key_Handle(destination)), seed)
}

// Public_Key_From_Private derives compressed public bytes without retaining duplicate state.
func Public_Key_From_Private(destination Public_Key, private_key Private_Key_Handle) {
	Public_Key_Invariants(destination, "Public_Key_From_Private.destination")
	Private_Key_Handle_Invariants(private_key, "Public_Key_From_Private.private_key")
	public_key_derive(destination, private_key)
}

// Sign writes deterministic output only after bounded message validation.
func Sign(
	destination Signature,
	private_key Private_Key_Handle,
	message Message,
) {
	defer func() { Signature_Invariants(destination, "Sign.destination.output") }()
	Signature_Invariants(destination, "Sign.destination")
	Private_Key_Handle_Invariants(private_key, "Sign.private_key")
	Message_Invariants(message, "Sign.message")
	signature_create(destination, private_key, message)
}

// Verify rejects hostile sizes before Edwards decoding.
func Verify(
	public_key Public_Key, message Message, signature Signature_Unvalidated,
) (verified Verification) {
	defer func() { Verification_Invariants(verified, "Verify.verified") }()
	Public_Key_Invariants(public_key, "Verify.public_key")
	Message_Invariants(message, "Verify.message")
	Signature_Unvalidated_Invariants(signature, "Verify.signature")
	if len(signature) != SIGNATURE_SIZE {
		return false
	}
	return signature_verify(public_key, message, Signature(signature))
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

// Wide_Destination fixes SHA-512 scratch to one reduced scalar input.
type Wide_Destination []byte

// Wide_Destination_Invariants rejects partial SHA-512 scratch.
func Wide_Destination_Invariants(value Wide_Destination, _ aver.Namespace) {
	aver.Always(len(value) == WIDE_SCALAR_SIZE, "Wide scalar scratch has SHA-512 width.")
}

// Wide_Source is one complete SHA-512 scalar-reduction input.
type Wide_Source []byte

// Wide_Source_Invariants rejects partial reduction input.
func Wide_Source_Invariants(value Wide_Source, _ aver.Namespace) {
	aver.Always(len(value) == WIDE_SCALAR_SIZE, "Wide scalar input has SHA-512 width.")
}

// Prefix is the secret second half of the expanded seed digest.
type Prefix []byte

// Prefix_Invariants fixes deterministic nonce-key width.
func Prefix_Invariants(value Prefix, _ aver.Namespace) {
	aver.Always(len(value) == SEED_SIZE, "Nonce prefix has Ed25519 seed width.")
}

// Point_Encoding is one compressed Edwards point.
type Point_Encoding []byte

// Point_Encoding_Invariants fixes compressed point width.
func Point_Encoding_Invariants(value Point_Encoding, _ aver.Namespace) {
	aver.Always(len(value) == PUBLIC_KEY_SIZE, "Point encoding has Ed25519 width.")
}

// Scalar_Encoding is one canonical scalar candidate.
type Scalar_Encoding []byte

// Scalar_Encoding_Invariants fixes group scalar width.
func Scalar_Encoding_Invariants(value Scalar_Encoding, _ aver.Namespace) {
	aver.Always(len(value) == SEED_SIZE, "Scalar encoding has Ed25519 width.")
}

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
	scalar Scalar_Handle,
	prefix Prefix,
	private_key Private_Key_Handle,
) {
	Scalar_Handle_Invariants(scalar, "private_key_expand.scalar")
	Prefix_Invariants(prefix, "private_key_expand.prefix")
	Private_Key_Handle_Invariants(private_key, "private_key_expand.private_key")
	var digest [WIDE_SCALAR_SIZE]byte
	hash_seed(Wide_Destination(digest[:]), private_key)
	digest[bits.BIT_COUNT_MINIMUM] &= SCALAR_CLAMP_LOW_CLEAR_MASK
	digest[SEED_SIZE-binary.UINT_8_SIZE] &= SCALAR_CLAMP_HIGH_CLEAR_MASK
	digest[SEED_SIZE-binary.UINT_8_SIZE] |= SCALAR_CLAMP_HIGH_SET_MASK
	var wide [WIDE_SCALAR_SIZE]byte
	copy(wide[:SEED_SIZE], digest[:SEED_SIZE])
	scalar_reduce_wide(scalar, Wide_Source(wide[:]))
	copy(prefix, digest[SEED_SIZE:])
}

func public_key_derive(destination Public_Key, private_key Private_Key_Handle) {
	Public_Key_Invariants(destination, "public_key_derive.destination")
	Private_Key_Handle_Invariants(private_key, "public_key_derive.private_key")
	var scalar Scalar
	var prefix_storage [SEED_SIZE]byte
	private_key_expand(&scalar, Prefix(prefix_storage[:]), private_key)
	var point Point
	point_scalar_base_multiply(&point, &scalar)
	point_encode(Point_Encoding(destination), &point)
}

func signature_create(
	destination Signature,
	private_key Private_Key_Handle, message Message,
) {
	Signature_Invariants(destination, "signature_create.destination")
	Private_Key_Handle_Invariants(private_key, "signature_create.private_key")
	Message_Invariants(message, "signature_create.message")
	var private_scalar Scalar
	var prefix_storage [SEED_SIZE]byte
	prefix := Prefix(prefix_storage[:])
	private_key_expand(&private_scalar, prefix, private_key)
	var public_key_storage [PUBLIC_KEY_SIZE]byte
	public_key := Public_Key(public_key_storage[:])
	public_key_derive(public_key, private_key)
	var nonce_digest [WIDE_SCALAR_SIZE]byte
	hash_nonce(Wide_Destination(nonce_digest[:]), prefix, message)
	var nonce Scalar
	scalar_reduce_wide(&nonce, Wide_Source(nonce_digest[:]))
	var nonce_point Point
	point_scalar_base_multiply(&nonce_point, &nonce)
	encoded_point := Point_Encoding(destination[:PUBLIC_KEY_SIZE])
	point_encode(encoded_point, &nonce_point)
	var challenge_digest [WIDE_SCALAR_SIZE]byte
	hash_challenge(
		Wide_Destination(challenge_digest[:]),
		encoded_point, public_key, message,
	)
	var challenge Scalar
	scalar_reduce_wide(&challenge, Wide_Source(challenge_digest[:]))
	var product, response Scalar
	scalar_multiply(&product, &challenge, &private_scalar)
	scalar_add(&response, &product, &nonce)
	scalar_encode(Scalar_Encoding(destination[PUBLIC_KEY_SIZE:]), &response)
}

func signature_verify(
	public_key Public_Key, message Message, signature Signature,
) (verified Verification) {
	defer func() { Verification_Invariants(verified, "signature_verify.verified") }()
	Public_Key_Invariants(public_key, "signature_verify.public_key")
	Message_Invariants(message, "signature_verify.message")
	Signature_Invariants(signature, "signature_verify.signature")
	var public_point Point
	if point_decode(
		&public_point, Point_Encoding(public_key),
	) != DECISION_TRUE {
		return false
	}
	if point_has_small_order(&public_point) == DECISION_TRUE {
		return false
	}
	var response Scalar
	if scalar_decode_canonical(
		&response, Scalar_Encoding(signature[PUBLIC_KEY_SIZE:]),
	) != DECISION_TRUE {
		return false
	}
	var challenge_digest [WIDE_SCALAR_SIZE]byte
	hash_challenge(
		Wide_Destination(challenge_digest[:]),
		Point_Encoding(signature[:PUBLIC_KEY_SIZE]), public_key, message,
	)
	var challenge Scalar
	scalar_reduce_wide(&challenge, Wide_Source(challenge_digest[:]))
	var response_point, challenge_point Point
	point_scalar_base_multiply(&response_point, &response)
	point_scalar_multiply(&challenge_point, &public_point, &challenge)
	point_negate(&challenge_point, &challenge_point)
	var expected Point
	point_add(&expected, &response_point, &challenge_point)
	var expected_encoding [PUBLIC_KEY_SIZE]byte
	point_encode(Point_Encoding(expected_encoding[:]), &expected)
	difference := bits.WORD_8_MINIMUM
	for index := range expected_encoding {
		difference |= expected_encoding[index] ^ signature[index]
	}
	return Verification(difference == bits.WORD_8_MINIMUM)
}

func hash_seed(destination Wide_Destination, private_key Private_Key_Handle) {
	Wide_Destination_Invariants(destination, "hash_seed.destination")
	Private_Key_Handle_Invariants(private_key, "hash_seed.private_key")
	var digest sha512.Digest
	sha512.Digest_Init(&digest, sha512.KIND_SHA_512)
	sha512.Digest_Write(&digest, sha512.Source(private_key_seed(private_key)))
	sha512.Digest_Sum_Into(&digest, sha512.Destination(destination))
}

func hash_nonce(
	destination Wide_Destination,
	prefix Prefix, message Message,
) {
	Wide_Destination_Invariants(destination, "hash_nonce.destination")
	Prefix_Invariants(prefix, "hash_nonce.prefix")
	Message_Invariants(message, "hash_nonce.message")
	var digest sha512.Digest
	sha512.Digest_Init(&digest, sha512.KIND_SHA_512)
	sha512.Digest_Write(&digest, sha512.Source(prefix))
	sha512.Digest_Write(&digest, sha512.Source(message))
	sha512.Digest_Sum_Into(&digest, sha512.Destination(destination))
}

func hash_challenge(
	destination Wide_Destination,
	encoded_point Point_Encoding,
	public_key Public_Key,
	message Message,
) {
	Wide_Destination_Invariants(destination, "hash_challenge.destination")
	Point_Encoding_Invariants(encoded_point, "hash_challenge.encoded_point")
	Public_Key_Invariants(public_key, "hash_challenge.public_key")
	Message_Invariants(message, "hash_challenge.message")
	var digest sha512.Digest
	sha512.Digest_Init(&digest, sha512.KIND_SHA_512)
	sha512.Digest_Write(&digest, sha512.Source(encoded_point))
	sha512.Digest_Write(&digest, sha512.Source(public_key))
	sha512.Digest_Write(&digest, sha512.Source(message))
	sha512.Digest_Sum_Into(&digest, sha512.Destination(destination))
}

func point_scalar_base_multiply(
	destination Point_Handle,
	scalar Scalar_Handle,
) {
	Point_Handle_Invariants(destination, "point_scalar_base_multiply.destination")
	Scalar_Handle_Invariants(scalar, "point_scalar_base_multiply.scalar")
	var base Point
	point_base(&base)
	point_scalar_multiply(destination, &base, scalar)
}

func point_scalar_multiply(
	destination Point_Handle,
	point Point_Handle,
	scalar Scalar_Handle,
) {
	Point_Handle_Invariants(destination, "point_scalar_multiply.destination")
	Point_Handle_Invariants(point, "point_scalar_multiply.point")
	Scalar_Handle_Invariants(scalar, "point_scalar_multiply.scalar")
	scalar_words := (*[SCALAR_LIMB_COUNT]uint64)(unsafe.Pointer(scalar))
	var result Point
	point_identity(&result)
	for bit_count := SCALAR_BIT_COUNT; bit_count > bits.BIT_COUNT_MINIMUM; bit_count-- {
		bit_index := bit_count - binary.UINT_8_SIZE
		var doubled, added Point
		point_double(&doubled, &result)
		point_add(&added, &doubled, point)
		word_index := bit_index / bits.BIT_COUNT_64_MAXIMUM
		word_shift := uint(bit_index % bits.BIT_COUNT_64_MAXIMUM)
		bit := scalar_words[word_index] >> word_shift & binary.UINT_8_SIZE
		point_select(&result, &added, &doubled, Decision(bit))
	}
	*destination = result
}

// Extended Edwards addition stays complete for every valid curve pair.
func point_add(
	destination Point_Handle,
	left Point_Handle,
	right Point_Handle,
) {
	Point_Handle_Invariants(destination, "point_add.destination")
	Point_Handle_Invariants(left, "point_add.left")
	Point_Handle_Invariants(right, "point_add.right")
	destination_coordinates := (*[POINT_COORDINATE_COUNT]Field)(unsafe.Pointer(destination))
	left_coordinates := (*[POINT_COORDINATE_COUNT]Field)(unsafe.Pointer(left))
	right_coordinates := (*[POINT_COORDINATE_COUNT]Field)(unsafe.Pointer(right))
	var left_difference, right_difference Field
	var left_sum, right_sum Field
	var a, b, c, d, e, f, g, h Field
	field_subtract(
		&left_difference, &left_coordinates[POINT_Y_INDEX],
		&left_coordinates[POINT_X_INDEX],
	)
	field_subtract(
		&right_difference, &right_coordinates[POINT_Y_INDEX],
		&right_coordinates[POINT_X_INDEX],
	)
	field_multiply(&a, &left_difference, &right_difference)
	field_add(
		&left_sum, &left_coordinates[POINT_Y_INDEX], &left_coordinates[POINT_X_INDEX],
	)
	field_add(
		&right_sum, &right_coordinates[POINT_Y_INDEX], &right_coordinates[POINT_X_INDEX],
	)
	field_multiply(&b, &left_sum, &right_sum)
	var twice_d Field
	field_twice_d(&twice_d)
	field_multiply(&c, &left_coordinates[POINT_T_INDEX], &right_coordinates[POINT_T_INDEX])
	field_multiply(&c, &c, &twice_d)
	field_multiply(&d, &left_coordinates[POINT_Z_INDEX], &right_coordinates[POINT_Z_INDEX])
	field_add(&d, &d, &d)
	field_subtract(&e, &b, &a)
	field_subtract(&f, &d, &c)
	field_add(&g, &d, &c)
	field_add(&h, &b, &a)
	field_multiply(&destination_coordinates[POINT_X_INDEX], &e, &f)
	field_multiply(&destination_coordinates[POINT_Y_INDEX], &g, &h)
	field_multiply(&destination_coordinates[POINT_T_INDEX], &e, &h)
	field_multiply(&destination_coordinates[POINT_Z_INDEX], &f, &g)
}

func point_double(
	destination Point_Handle,
	source Point_Handle,
) {
	Point_Handle_Invariants(destination, "point_double.destination")
	Point_Handle_Invariants(source, "point_double.source")
	destination_coordinates := (*[POINT_COORDINATE_COUNT]Field)(unsafe.Pointer(destination))
	source_coordinates := (*[POINT_COORDINATE_COUNT]Field)(unsafe.Pointer(source))
	var a, b, c, d, e, f, g, h Field
	field_square(&a, &source_coordinates[POINT_X_INDEX])
	field_square(&b, &source_coordinates[POINT_Y_INDEX])
	field_square(&c, &source_coordinates[POINT_Z_INDEX])
	field_add(&c, &c, &c)
	field_negate(&d, &a)
	field_add(&e, &source_coordinates[POINT_X_INDEX], &source_coordinates[POINT_Y_INDEX])
	field_square(&e, &e)
	field_subtract(&e, &e, &a)
	field_subtract(&e, &e, &b)
	field_add(&g, &d, &b)
	field_subtract(&f, &g, &c)
	field_subtract(&h, &d, &b)
	field_multiply(&destination_coordinates[POINT_X_INDEX], &e, &f)
	field_multiply(&destination_coordinates[POINT_Y_INDEX], &g, &h)
	field_multiply(&destination_coordinates[POINT_T_INDEX], &e, &h)
	field_multiply(&destination_coordinates[POINT_Z_INDEX], &f, &g)
}

func point_negate(
	destination Point_Handle,
	source Point_Handle,
) {
	Point_Handle_Invariants(destination, "point_negate.destination")
	Point_Handle_Invariants(source, "point_negate.source")
	point := *source
	coordinates := (*[POINT_COORDINATE_COUNT]Field)(unsafe.Pointer(&point))
	field_negate(&coordinates[POINT_X_INDEX], &coordinates[POINT_X_INDEX])
	field_negate(&coordinates[POINT_T_INDEX], &coordinates[POINT_T_INDEX])
	*destination = point
}

func point_select(
	destination Point_Handle,
	first Point_Handle,
	second Point_Handle,
	condition Decision,
) {
	Point_Handle_Invariants(destination, "point_select.destination")
	Point_Handle_Invariants(first, "point_select.first")
	Point_Handle_Invariants(second, "point_select.second")
	Decision_Invariants(condition, "point_select.condition")
	destination_coordinates := (*[POINT_COORDINATE_COUNT]Field)(unsafe.Pointer(destination))
	first_coordinates := (*[POINT_COORDINATE_COUNT]Field)(unsafe.Pointer(first))
	second_coordinates := (*[POINT_COORDINATE_COUNT]Field)(unsafe.Pointer(second))
	for index := range destination_coordinates {
		field_select(
			&destination_coordinates[index], &first_coordinates[index],
			&second_coordinates[index], condition,
		)
	}
}

func point_encode(
	destination Point_Encoding,
	point Point_Handle,
) {
	Point_Encoding_Invariants(destination, "point_encode.destination")
	Point_Handle_Invariants(point, "point_encode.point")
	coordinates := (*[POINT_COORDINATE_COUNT]Field)(unsafe.Pointer(point))
	var inverse, x, y Field
	field_inverse(&inverse, &coordinates[POINT_Z_INDEX])
	field_multiply(&x, &coordinates[POINT_X_INDEX], &inverse)
	field_multiply(&y, &coordinates[POINT_Y_INDEX], &inverse)
	field_encode(destination, &y)
	destination[PUBLIC_KEY_SIZE-binary.UINT_8_SIZE] |=
		byte(uint64(x.Limb_0)&binary.UINT_8_SIZE) <<
			(bits.BIT_COUNT_8_MAXIMUM - binary.UINT_8_SIZE)
}

func point_decode(
	destination Point_Handle,
	encoding Point_Encoding,
) (valid Decision) {
	defer func() { Decision_Invariants(valid, "point_decode.valid") }()
	Point_Handle_Invariants(destination, "point_decode.destination")
	Point_Encoding_Invariants(encoding, "point_decode.encoding")
	destination_coordinates := (*[POINT_COORDINATE_COUNT]Field)(unsafe.Pointer(destination))
	var y_storage [PUBLIC_KEY_SIZE]byte
	copy(y_storage[:], encoding)
	y_encoding := Point_Encoding(y_storage[:])
	sign := uint64(
		y_encoding[PUBLIC_KEY_SIZE-binary.UINT_8_SIZE] >>
			(bits.BIT_COUNT_8_MAXIMUM - binary.UINT_8_SIZE),
	)
	y_encoding[PUBLIC_KEY_SIZE-binary.UINT_8_SIZE] &=
		byte(bits.WORD_8_MAXIMUM >> binary.UINT_8_SIZE)
	var y Field
	canonical := field_decode(&y, y_encoding)
	var y_square, numerator, denominator, inverse, x_square, x Field
	field_square(&y_square, &y)
	var one Field
	field_one(&one)
	field_subtract(&numerator, &y_square, &one)
	var d Field
	field_d(&d)
	field_multiply(&denominator, &d, &y_square)
	field_add(&denominator, &denominator, &one)
	field_inverse(&inverse, &denominator)
	field_multiply(&x_square, &numerator, &inverse)
	square := field_square_root(&x, &x_square)
	x_zero := field_is_zero(&x)
	var negative Field
	field_negate(&negative, &x)
	field_select(
		&x, &negative, &x,
		Decision(uint64(x.Limb_0)&binary.UINT_8_SIZE^sign),
	)
	valid = canonical & square & (x_zero&Decision(sign) ^ DECISION_TRUE)
	destination_coordinates[POINT_X_INDEX] = x
	destination_coordinates[POINT_Y_INDEX] = y
	destination_coordinates[POINT_Z_INDEX] = one
	field_multiply(&destination_coordinates[POINT_T_INDEX], &x, &y)
	return valid
}

func point_has_small_order(
	point Point_Handle,
) (small Decision) {
	defer func() { Decision_Invariants(small, "point_has_small_order.small") }()
	Point_Handle_Invariants(point, "point_has_small_order.point")
	multiple := *point
	for range COFACTOR_DOUBLING_COUNT {
		point_double(&multiple, &multiple)
	}
	var identity Point
	point_identity(&identity)
	return point_equal(&multiple, &identity)
}

func point_equal(
	left Point_Handle,
	right Point_Handle,
) (equal Decision) {
	defer func() { Decision_Invariants(equal, "point_equal.equal") }()
	Point_Handle_Invariants(left, "point_equal.left")
	Point_Handle_Invariants(right, "point_equal.right")
	left_coordinates := (*[POINT_COORDINATE_COUNT]Field)(unsafe.Pointer(left))
	right_coordinates := (*[POINT_COORDINATE_COUNT]Field)(unsafe.Pointer(right))
	var left_x, right_x, left_y, right_y Field
	field_multiply(&left_x, &left_coordinates[POINT_X_INDEX], &right_coordinates[POINT_Z_INDEX])
	field_multiply(
		&right_x, &right_coordinates[POINT_X_INDEX], &left_coordinates[POINT_Z_INDEX],
	)
	field_multiply(&left_y, &left_coordinates[POINT_Y_INDEX], &right_coordinates[POINT_Z_INDEX])
	field_multiply(
		&right_y, &right_coordinates[POINT_Y_INDEX], &left_coordinates[POINT_Z_INDEX],
	)
	x_equal := field_equal(&left_x, &right_x)
	y_equal := field_equal(&left_y, &right_y)
	equal = x_equal & y_equal
	return equal
}

func point_identity(destination Point_Handle) {
	Point_Handle_Invariants(destination, "point_identity.destination")
	coordinates := (*[POINT_COORDINATE_COUNT]Field)(unsafe.Pointer(destination))
	field_one(&coordinates[POINT_Y_INDEX])
	field_one(&coordinates[POINT_Z_INDEX])
}

func point_base(destination Point_Handle) {
	Point_Handle_Invariants(destination, "point_base.destination")
	encoding := [PUBLIC_KEY_SIZE]byte{
		0x58, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
		0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
		0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
		0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66, 0x66,
	}
	valid := point_decode(destination, Point_Encoding(encoding[:]))
	aver.Always(
		valid == DECISION_TRUE,
		"The RFC 8032 base point encoding is valid.",
	)
}

func field_multiply(
	destination Field_Handle,
	left Field_Handle,
	right Field_Handle,
) {
	Field_Handle_Invariants(destination, "field_multiply.destination")
	Field_Handle_Invariants(left, "field_multiply.left")
	Field_Handle_Invariants(right, "field_multiply.right")
	var result Field
	addend := *left
	result_words := (*[FIELD_LIMB_COUNT]uint64)(unsafe.Pointer(&result))
	addend_words := (*[FIELD_LIMB_COUNT]uint64)(unsafe.Pointer(&addend))
	right_words := (*[FIELD_LIMB_COUNT]uint64)(unsafe.Pointer(right))
	var modulus Field
	field_modulus(&modulus)
	modulus_words := (*[FIELD_LIMB_COUNT]uint64)(unsafe.Pointer(&modulus))
	// Local words passed the boundary. Per-bit recording would dominate the product.
	modular_add := func(
		destination_words *[FIELD_LIMB_COUNT]uint64,
		left_words *[FIELD_LIMB_COUNT]uint64,
		right_words *[FIELD_LIMB_COUNT]uint64,
	) {
		carry_value := uint64(bits.WORD_64_MINIMUM)
		var sum [FIELD_LIMB_COUNT]uint64
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
		var reduced [FIELD_LIMB_COUNT]uint64
		for index := range reduced {
			partial := sum[index] - modulus_words[index]
			partial_borrow := ((^sum[index] & modulus_words[index]) |
				(^(sum[index] ^ modulus_words[index]) & partial)) >>
				(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
			value := partial - borrow_value
			borrow_borrow := ((^partial & borrow_value) |
				(^(partial ^ borrow_value) & value)) >>
				(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
			reduced[index] = value
			borrow_value = partial_borrow | borrow_borrow
		}
		mask := uint64(bits.WORD_64_MINIMUM) - (borrow_value ^ binary.UINT_8_SIZE)
		for index := range destination_words {
			destination_words[index] = sum[index] ^
				mask&(reduced[index]^sum[index])
		}
	}
	for bit_index := bits.BIT_COUNT_MINIMUM; bit_index < FIELD_BIT_COUNT; bit_index++ {
		var candidate [FIELD_LIMB_COUNT]uint64
		modular_add(&candidate, result_words, addend_words)
		word_index := bit_index / bits.BIT_COUNT_64_MAXIMUM
		word_shift := uint(bit_index % bits.BIT_COUNT_64_MAXIMUM)
		bit := right_words[word_index] >> word_shift & binary.UINT_8_SIZE
		mask := uint64(bits.WORD_64_MINIMUM) - bit
		for index := range result_words {
			result_words[index] ^= mask & (candidate[index] ^ result_words[index])
		}
		var doubled [FIELD_LIMB_COUNT]uint64
		modular_add(&doubled, addend_words, addend_words)
		*addend_words = doubled
	}
	*destination = result
}

func field_square(
	destination Field_Handle, source Field_Handle,
) {
	Field_Handle_Invariants(destination, "field_square.destination")
	Field_Handle_Invariants(source, "field_square.source")
	field_multiply(destination, source, source)
}

func field_add(
	destination Field_Handle,
	left Field_Handle,
	right Field_Handle,
) {
	Field_Handle_Invariants(destination, "field_add.destination")
	Field_Handle_Invariants(left, "field_add.left")
	Field_Handle_Invariants(right, "field_add.right")
	var sum, reduced Field
	limbs_add(field_limbs(&sum), field_limbs(left), field_limbs(right))
	var modulus Field
	field_modulus(&modulus)
	borrow := limbs_subtract(
		field_limbs(&reduced), field_limbs(&sum), field_limbs(&modulus),
	)
	field_select(destination, &reduced, &sum, borrow^DECISION_TRUE)
}

func field_subtract(
	destination Field_Handle,
	left Field_Handle,
	right Field_Handle,
) {
	Field_Handle_Invariants(destination, "field_subtract.destination")
	Field_Handle_Invariants(left, "field_subtract.left")
	Field_Handle_Invariants(right, "field_subtract.right")
	var difference, correction Field
	borrow := limbs_subtract(
		field_limbs(&difference), field_limbs(left), field_limbs(right),
	)
	var modulus Field
	field_modulus(&modulus)
	mask := uint64(bits.WORD_64_MINIMUM) - uint64(borrow)
	correction_words := (*[FIELD_LIMB_COUNT]uint64)(unsafe.Pointer(&correction))
	modulus_words := (*[FIELD_LIMB_COUNT]uint64)(unsafe.Pointer(&modulus))
	for index := range correction_words {
		correction_words[index] = modulus_words[index] & mask
	}
	limbs_add(
		field_limbs(destination), field_limbs(&difference), field_limbs(&correction),
	)
}

func field_negate(
	destination Field_Handle, source Field_Handle,
) {
	Field_Handle_Invariants(destination, "field_negate.destination")
	Field_Handle_Invariants(source, "field_negate.source")
	var zero Field
	field_subtract(destination, &zero, source)
}

func field_inverse(
	destination Field_Handle, source Field_Handle,
) {
	Field_Handle_Invariants(destination, "field_inverse.destination")
	Field_Handle_Invariants(source, "field_inverse.source")
	var exponent Field
	field_inverse_exponent(&exponent)
	field_power(destination, source, &exponent)
}

func field_square_root(
	destination Field_Handle,
	source Field_Handle,
) (valid Decision) {
	defer func() { Decision_Invariants(valid, "field_square_root.valid") }()
	Field_Handle_Invariants(destination, "field_square_root.destination")
	Field_Handle_Invariants(source, "field_square_root.source")
	var exponent Field
	field_square_root_exponent(&exponent)
	var candidate, square Field
	field_power(&candidate, source, &exponent)
	field_square(&square, &candidate)
	equal := field_equal(&square, source)
	var square_root_m1 Field
	field_square_root_minus_one(&square_root_m1)
	var rotated Field
	field_multiply(&rotated, &candidate, &square_root_m1)
	field_select(&candidate, &candidate, &rotated, equal)
	field_square(&square, &candidate)
	valid = field_equal(&square, source)
	*destination = candidate
	return valid
}

func field_power(
	destination Field_Handle,
	base Field_Handle,
	exponent Field_Handle,
) {
	Field_Handle_Invariants(destination, "field_power.destination")
	Field_Handle_Invariants(base, "field_power.base")
	Field_Handle_Invariants(exponent, "field_power.exponent")
	exponent_words := (*[FIELD_LIMB_COUNT]uint64)(unsafe.Pointer(exponent))
	var result Field
	field_one(&result)
	for bit_count := FIELD_BIT_COUNT; bit_count > bits.BIT_COUNT_MINIMUM; bit_count-- {
		bit_index := bit_count - binary.UINT_8_SIZE
		var square, product Field
		field_square(&square, &result)
		field_multiply(&product, &square, base)
		word_index := bit_index / bits.BIT_COUNT_64_MAXIMUM
		word_shift := uint(bit_index % bits.BIT_COUNT_64_MAXIMUM)
		bit := exponent_words[word_index] >> word_shift & binary.UINT_8_SIZE
		field_select(&result, &product, &square, Decision(bit))
	}
	*destination = result
}

func field_select(
	destination Field_Handle,
	first Field_Handle,
	second Field_Handle,
	condition Decision,
) {
	Field_Handle_Invariants(destination, "field_select.destination")
	Field_Handle_Invariants(first, "field_select.first")
	Field_Handle_Invariants(second, "field_select.second")
	Decision_Invariants(condition, "field_select.condition")
	mask := uint64(bits.WORD_64_MINIMUM) - uint64(condition)
	destination_words := (*[FIELD_LIMB_COUNT]uint64)(unsafe.Pointer(destination))
	first_words := (*[FIELD_LIMB_COUNT]uint64)(unsafe.Pointer(first))
	second_words := (*[FIELD_LIMB_COUNT]uint64)(unsafe.Pointer(second))
	for index := range destination_words {
		destination_words[index] = second_words[index] ^
			mask&(first_words[index]^second_words[index])
	}
}

func field_equal(
	left Field_Handle, right Field_Handle,
) (equal Decision) {
	defer func() { Decision_Invariants(equal, "field_equal.equal") }()
	Field_Handle_Invariants(left, "field_equal.left")
	Field_Handle_Invariants(right, "field_equal.right")
	left_words := (*[FIELD_LIMB_COUNT]uint64)(unsafe.Pointer(left))
	right_words := (*[FIELD_LIMB_COUNT]uint64)(unsafe.Pointer(right))
	difference := uint64(bits.WORD_64_MINIMUM)
	for index := range left_words {
		difference |= left_words[index] ^ right_words[index]
	}
	equal = Decision((difference|-difference)>>
		(bits.BIT_COUNT_64_MAXIMUM-binary.UINT_8_SIZE) ^ binary.UINT_8_SIZE)
	return equal
}

func field_is_zero(
	value Field_Handle,
) (zero Decision) {
	defer func() { Decision_Invariants(zero, "field_is_zero.zero") }()
	Field_Handle_Invariants(value, "field_is_zero.value")
	var empty Field
	return field_equal(value, &empty)
}

func field_decode(
	destination Field_Handle, source Point_Encoding,
) (canonical Decision) {
	defer func() { Decision_Invariants(canonical, "field_decode.canonical") }()
	Field_Handle_Invariants(destination, "field_decode.destination")
	Point_Encoding_Invariants(source, "field_decode.source")
	destination_words := (*[FIELD_LIMB_COUNT]uint64)(unsafe.Pointer(destination))
	for index := bits.BIT_COUNT_MINIMUM; index < FIELD_LIMB_COUNT; index++ {
		source_index := index * binary.UINT_64_SIZE
		destination_words[index] = uint64(binary.Uint_64(
			binary.Bytes(source[source_index:source_index+binary.UINT_64_SIZE]),
			binary.LITTLE_ENDIAN,
		))
	}
	var modulus Field
	field_modulus(&modulus)
	var difference Field
	return limbs_subtract(
		field_limbs(&difference), field_limbs(destination), field_limbs(&modulus),
	)
}

func field_encode(
	destination Point_Encoding, source Field_Handle,
) {
	Point_Encoding_Invariants(destination, "field_encode.destination")
	Field_Handle_Invariants(source, "field_encode.source")
	source_words := (*[FIELD_LIMB_COUNT]uint64)(unsafe.Pointer(source))
	for index := bits.BIT_COUNT_MINIMUM; index < FIELD_LIMB_COUNT; index++ {
		destination_index := index * binary.UINT_64_SIZE
		destination_end := destination_index + binary.UINT_64_SIZE
		binary.Put_Uint_64(
			binary.Bytes(destination[destination_index:destination_end]),
			binary.Word_64(source_words[index]), binary.LITTLE_ENDIAN,
		)
	}
}

func scalar_reduce_wide(
	destination Scalar_Handle, source Wide_Source,
) {
	Scalar_Handle_Invariants(destination, "scalar_reduce_wide.destination")
	Wide_Source_Invariants(source, "scalar_reduce_wide.source")
	var result Scalar
	var one Scalar
	scalar_one(&one)
	for bit_count := WIDE_SCALAR_BIT_COUNT; bit_count > bits.BIT_COUNT_MINIMUM; bit_count-- {
		bit_index := bit_count - binary.UINT_8_SIZE
		var doubled, incremented Scalar
		scalar_add(&doubled, &result, &result)
		scalar_add(&incremented, &doubled, &one)
		byte_index := bit_index / bits.BIT_COUNT_8_MAXIMUM
		byte_shift := uint(bit_index % bits.BIT_COUNT_8_MAXIMUM)
		bit := uint64(source[byte_index]>>byte_shift) & binary.UINT_8_SIZE
		limbs_select(
			scalar_limbs(&result), scalar_limbs(&incremented), scalar_limbs(&doubled),
			Decision(bit),
		)
	}
	*destination = result
}

func scalar_decode_canonical(
	destination Scalar_Handle, source Scalar_Encoding,
) (canonical Decision) {
	defer func() { Decision_Invariants(canonical, "scalar_decode_canonical.canonical") }()
	Scalar_Handle_Invariants(destination, "scalar_decode_canonical.destination")
	Scalar_Encoding_Invariants(source, "scalar_decode_canonical.source")
	destination_words := (*[SCALAR_LIMB_COUNT]uint64)(unsafe.Pointer(destination))
	for index := bits.BIT_COUNT_MINIMUM; index < SCALAR_LIMB_COUNT; index++ {
		source_index := index * binary.UINT_64_SIZE
		destination_words[index] = uint64(binary.Uint_64(
			binary.Bytes(source[source_index:source_index+binary.UINT_64_SIZE]),
			binary.LITTLE_ENDIAN,
		))
	}
	var order Scalar
	scalar_order(&order)
	var difference Scalar
	return limbs_subtract(
		scalar_limbs(&difference), scalar_limbs(destination), scalar_limbs(&order),
	)
}

func scalar_encode(
	destination Scalar_Encoding, source Scalar_Handle,
) {
	Scalar_Encoding_Invariants(destination, "scalar_encode.destination")
	Scalar_Handle_Invariants(source, "scalar_encode.source")
	source_words := (*[SCALAR_LIMB_COUNT]uint64)(unsafe.Pointer(source))
	for index := bits.BIT_COUNT_MINIMUM; index < SCALAR_LIMB_COUNT; index++ {
		destination_index := index * binary.UINT_64_SIZE
		destination_end := destination_index + binary.UINT_64_SIZE
		binary.Put_Uint_64(
			binary.Bytes(destination[destination_index:destination_end]),
			binary.Word_64(source_words[index]), binary.LITTLE_ENDIAN,
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
	limbs_add(scalar_limbs(&sum), scalar_limbs(left), scalar_limbs(right))
	var order Scalar
	scalar_order(&order)
	borrow := limbs_subtract(
		scalar_limbs(&reduced), scalar_limbs(&sum), scalar_limbs(&order),
	)
	limbs_select(
		scalar_limbs(destination), scalar_limbs(&reduced), scalar_limbs(&sum),
		borrow^DECISION_TRUE,
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
	var result Scalar
	addend := *left
	right_words := (*[SCALAR_LIMB_COUNT]uint64)(unsafe.Pointer(right))
	for bit_index := bits.BIT_COUNT_MINIMUM; bit_index < SCALAR_BIT_COUNT; bit_index++ {
		var candidate Scalar
		scalar_add(&candidate, &result, &addend)
		word_index := bit_index / bits.BIT_COUNT_64_MAXIMUM
		word_shift := uint(bit_index % bits.BIT_COUNT_64_MAXIMUM)
		bit := right_words[word_index] >> word_shift & binary.UINT_8_SIZE
		limbs_select(
			scalar_limbs(&result), scalar_limbs(&candidate), scalar_limbs(&result),
			Decision(bit),
		)
		scalar_add(&addend, &addend, &addend)
	}
	*destination = result
}

func limbs_add(
	destination Limbs_Handle,
	left Limbs_Handle,
	right Limbs_Handle,
) (carry Decision) {
	defer func() { Decision_Invariants(carry, "limbs_add.carry") }()
	Limbs_Handle_Invariants(destination, "limbs_add.destination")
	Limbs_Handle_Invariants(left, "limbs_add.left")
	Limbs_Handle_Invariants(right, "limbs_add.right")
	destination_words := (*[SCALAR_LIMB_COUNT]uint64)(unsafe.Pointer(destination))
	left_words := (*[SCALAR_LIMB_COUNT]uint64)(unsafe.Pointer(left))
	right_words := (*[SCALAR_LIMB_COUNT]uint64)(unsafe.Pointer(right))
	carry_value := uint64(bits.WORD_64_MINIMUM)
	for index := range destination_words {
		partial := left_words[index] + right_words[index]
		partial_carry := ((left_words[index] & right_words[index]) |
			((left_words[index] | right_words[index]) & ^partial)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		value := partial + carry_value
		carry_carry := ((partial & carry_value) |
			((partial | carry_value) & ^value)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		destination_words[index] = value
		carry_value = partial_carry | carry_carry
	}
	carry = Decision(carry_value)
	return carry
}

func limbs_subtract(
	destination Limbs_Handle,
	left Limbs_Handle,
	right Limbs_Handle,
) (borrow Decision) {
	defer func() { Decision_Invariants(borrow, "limbs_subtract.borrow") }()
	Limbs_Handle_Invariants(destination, "limbs_subtract.destination")
	Limbs_Handle_Invariants(left, "limbs_subtract.left")
	Limbs_Handle_Invariants(right, "limbs_subtract.right")
	destination_words := (*[SCALAR_LIMB_COUNT]uint64)(unsafe.Pointer(destination))
	left_words := (*[SCALAR_LIMB_COUNT]uint64)(unsafe.Pointer(left))
	right_words := (*[SCALAR_LIMB_COUNT]uint64)(unsafe.Pointer(right))
	borrow_value := uint64(bits.WORD_64_MINIMUM)
	for index := range destination_words {
		partial := left_words[index] - right_words[index]
		partial_borrow := ((^left_words[index] & right_words[index]) |
			(^(left_words[index] ^ right_words[index]) & partial)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		value := partial - borrow_value
		borrow_borrow := ((^partial & borrow_value) |
			(^(partial ^ borrow_value) & value)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		destination_words[index] = value
		borrow_value = partial_borrow | borrow_borrow
	}
	borrow = Decision(borrow_value)
	return borrow
}

func limbs_select(
	destination Limbs_Handle,
	first Limbs_Handle,
	second Limbs_Handle,
	condition Decision,
) {
	Limbs_Handle_Invariants(destination, "limbs_select.destination")
	Limbs_Handle_Invariants(first, "limbs_select.first")
	Limbs_Handle_Invariants(second, "limbs_select.second")
	Decision_Invariants(condition, "limbs_select.condition")
	mask := uint64(bits.WORD_64_MINIMUM) - uint64(condition)
	destination_words := (*[SCALAR_LIMB_COUNT]uint64)(unsafe.Pointer(destination))
	first_words := (*[SCALAR_LIMB_COUNT]uint64)(unsafe.Pointer(first))
	second_words := (*[SCALAR_LIMB_COUNT]uint64)(unsafe.Pointer(second))
	for index := range destination_words {
		destination_words[index] = second_words[index] ^
			mask&(first_words[index]^second_words[index])
	}
}

func field_modulus(destination Field_Handle) {
	Field_Handle_Invariants(destination, "field_modulus.destination")
	*destination = Field{
		Limb_0: 0xffffffffffffffed,
		Limb_1: 0xffffffffffffffff,
		Limb_2: 0xffffffffffffffff,
		Limb_3: 0x7fffffffffffffff,
	}
}

func field_one(destination Field_Handle) {
	Field_Handle_Invariants(destination, "field_one.destination")
	*destination = Field{Limb_0: binary.UINT_8_SIZE}
}

// The RFC 8032 Edwards parameter is minus 121665 divided by 121666.
func field_d(destination Field_Handle) {
	Field_Handle_Invariants(destination, "field_d.destination")
	*destination = Field{
		Limb_0: 0x75eb4dca135978a3,
		Limb_1: 0x00700a4d4141d8ab,
		Limb_2: 0x8cc740797779e898,
		Limb_3: 0x52036cee2b6ffe73,
	}
}

func field_twice_d(destination Field_Handle) {
	Field_Handle_Invariants(destination, "field_twice_d.destination")
	var d Field
	field_d(&d)
	field_add(destination, &d, &d)
}

func field_inverse_exponent(destination Field_Handle) {
	Field_Handle_Invariants(destination, "field_inverse_exponent.destination")
	*destination = Field{
		Limb_0: 0xffffffffffffffeb,
		Limb_1: 0xffffffffffffffff,
		Limb_2: 0xffffffffffffffff,
		Limb_3: 0x7fffffffffffffff,
	}
}

func field_square_root_exponent(destination Field_Handle) {
	Field_Handle_Invariants(destination, "field_square_root_exponent.destination")
	*destination = Field{
		Limb_0: 0xfffffffffffffffe,
		Limb_1: 0xffffffffffffffff,
		Limb_2: 0xffffffffffffffff,
		Limb_3: 0x0fffffffffffffff,
	}
}

func field_square_root_minus_one(destination Field_Handle) {
	Field_Handle_Invariants(destination, "field_square_root_minus_one.destination")
	*destination = Field{
		Limb_0: 0xc4ee1b274a0ea0b0,
		Limb_1: 0x2f431806ad2fe478,
		Limb_2: 0x2b4d00993dfbd7a7,
		Limb_3: 0x2b8324804fc1df0b,
	}
}

func scalar_order(destination Scalar_Handle) {
	Scalar_Handle_Invariants(destination, "scalar_order.destination")
	*destination = Scalar{
		Limb_0: 0x5812631a5cf5d3ed,
		Limb_1: 0x14def9dea2f79cd6,
		Limb_2: 0x0000000000000000,
		Limb_3: 0x1000000000000000,
	}
}

func scalar_one(destination Scalar_Handle) {
	Scalar_Handle_Invariants(destination, "scalar_one.destination")
	*destination = Scalar{Limb_0: binary.UINT_8_SIZE}
}
