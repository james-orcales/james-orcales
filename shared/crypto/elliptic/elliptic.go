// Package elliptic implements bounded caller-owned NIST P-256 group operations.
package elliptic

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
)

// FIELD_LIMB_COUNT derives P-256 storage from its bit and machine-word widths.
const FIELD_LIMB_COUNT = P256_BIT_COUNT / bits.BIT_COUNT_64_MAXIMUM

// P256_BIT_COUNT is the NIST P-256 field and scalar width.
const P256_BIT_COUNT = bits.BIT_COUNT_64_MAXIMUM * binary.UINT_32_SIZE

// FIELD_ELEMENT_SIZE derives the fixed coordinate encoding width.
const FIELD_ELEMENT_SIZE = P256_BIT_COUNT / bits.BIT_COUNT_8_MAXIMUM

// SCALAR_SIZE is the fixed P-256 scalar width.
const SCALAR_SIZE = FIELD_ELEMENT_SIZE

// ENCODING_INFINITY_SIZE is the single SEC 1 infinity marker.
const ENCODING_INFINITY_SIZE = binary.UINT_8_SIZE

// ENCODING_COMPRESSED_SIZE holds one prefix and one coordinate.
const ENCODING_COMPRESSED_SIZE = ENCODING_INFINITY_SIZE + FIELD_ELEMENT_SIZE

// ENCODING_UNCOMPRESSED_SIZE holds one prefix and two coordinates.
const ENCODING_UNCOMPRESSED_SIZE = ENCODING_INFINITY_SIZE +
	FIELD_ELEMENT_SIZE + FIELD_ELEMENT_SIZE

// ENCODING_UNVALIDATED_SIZE_MINIMUM admits empty hostile input.
const ENCODING_UNVALIDATED_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// ENCODING_UNVALIDATED_SIZE_MAXIMUM admits one bounded invalid byte past complete encoding.
const ENCODING_UNVALIDATED_SIZE_MAXIMUM = ENCODING_UNCOMPRESSED_SIZE + binary.UINT_8_SIZE

// DESTINATION_SIZE_MINIMUM admits caller storage too short for infinity.
const DESTINATION_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// DESTINATION_SIZE_MAXIMUM holds the longest SEC 1 encoding.
const DESTINATION_SIZE_MAXIMUM = ENCODING_UNCOMPRESSED_SIZE

// SCALAR_UNVALIDATED_SIZE_MINIMUM admits empty hostile scalar input.
const SCALAR_UNVALIDATED_SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// SCALAR_UNVALIDATED_SIZE_MAXIMUM admits one bounded invalid byte past P-256 width.
const SCALAR_UNVALIDATED_SIZE_MAXIMUM = SCALAR_SIZE + binary.UINT_8_SIZE

// POINT_INFINITY_PREFIX is SEC 1's identity encoding.
const POINT_INFINITY_PREFIX byte = bits.WORD_8_MINIMUM

// POINT_COMPRESSED_EVEN_PREFIX selects an even affine y-coordinate.
const POINT_COMPRESSED_EVEN_PREFIX byte = POINT_INFINITY_PREFIX + binary.UINT_16_SIZE

// POINT_COMPRESSED_ODD_PREFIX selects an odd affine y-coordinate.
const POINT_COMPRESSED_ODD_PREFIX byte = POINT_COMPRESSED_EVEN_PREFIX + binary.UINT_8_SIZE

// POINT_UNCOMPRESSED_PREFIX selects two explicit affine coordinates.
const POINT_UNCOMPRESSED_PREFIX byte = POINT_COMPRESSED_ODD_PREFIX + binary.UINT_8_SIZE

// ENCODING_UNCOMPRESSED selects SEC 1 affine x and y output.
const ENCODING_UNCOMPRESSED Encoding_Kind = Encoding_Kind(bits.WORD_8_MINIMUM)

// ENCODING_COMPRESSED selects SEC 1 prefix and affine x output.
const ENCODING_COMPRESSED Encoding_Kind = ENCODING_UNCOMPRESSED + binary.UINT_8_SIZE

// COUNT_INFINITY is the complete identity encoding width.
const COUNT_INFINITY Count = ENCODING_INFINITY_SIZE

// COUNT_COMPRESSED is the complete compressed encoding width.
const COUNT_COMPRESSED Count = ENCODING_COMPRESSED_SIZE

// COUNT_UNCOMPRESSED is the complete uncompressed encoding width.
const COUNT_UNCOMPRESSED Count = ENCODING_UNCOMPRESSED_SIZE

// PARSE_STATUS_OK means a complete point encoding committed.
const PARSE_STATUS_OK Parse_Status = Parse_Status(bits.WORD_8_MINIMUM)

// PARSE_STATUS_INPUT_INVALID leaves a point destination unchanged.
const PARSE_STATUS_INPUT_INVALID Parse_Status = PARSE_STATUS_OK + binary.UINT_8_SIZE

// OUTPUT_STATUS_OK means a complete point encoding reached caller storage.
const OUTPUT_STATUS_OK Output_Status = Output_Status(bits.WORD_8_MINIMUM)

// OUTPUT_STATUS_DESTINATION_TOO_SMALL leaves byte storage unchanged.
const OUTPUT_STATUS_DESTINATION_TOO_SMALL Output_Status = OUTPUT_STATUS_OK + binary.UINT_8_SIZE

// SCALAR_STATUS_OK means exact-width scalar multiplication committed.
const SCALAR_STATUS_OK Scalar_Status = Scalar_Status(bits.WORD_8_MINIMUM)

// SCALAR_STATUS_INPUT_INVALID leaves a point destination unchanged.
const SCALAR_STATUS_INPUT_INVALID Scalar_Status = SCALAR_STATUS_OK + binary.UINT_8_SIZE

// DECISION_FALSE is a rejected constant-time decision.
const DECISION_FALSE Decision = Decision(bits.WORD_64_MINIMUM)

// DECISION_TRUE is an accepted constant-time decision.
const DECISION_TRUE Decision = DECISION_FALSE + binary.UINT_8_SIZE

// LIMB_INDEX_MINIMUM identifies the least-significant word.
const LIMB_INDEX_MINIMUM Limb_Index = Limb_Index(bits.WORD_8_MINIMUM)

// LIMB_INDEX_SECOND identifies the second word.
const LIMB_INDEX_SECOND = LIMB_INDEX_MINIMUM + binary.UINT_8_SIZE

// LIMB_INDEX_THIRD identifies the third word.
const LIMB_INDEX_THIRD = LIMB_INDEX_SECOND + binary.UINT_8_SIZE

// LIMB_INDEX_MAXIMUM identifies the most-significant word.
const LIMB_INDEX_MAXIMUM = LIMB_INDEX_THIRD + binary.UINT_8_SIZE

// Limb_Index selects one fixed arithmetic word.
type Limb_Index uint8

// Limb_Index_Invariants covers every arithmetic word position.
func Limb_Index_Invariants(value Limb_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(LIMB_INDEX_MINIMUM), uint8(LIMB_INDEX_SECOND),
			uint8(LIMB_INDEX_THIRD), uint8(LIMB_INDEX_MAXIMUM),
		).
		Ensure()
}

// Limb_Value carries one arithmetic word across safe accessors.
type Limb_Value uint64

// Limb_Value_Invariants spans every word value.
func Limb_Value_Invariants(value Limb_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Limb is one machine word of a P-256 integer.
type Limb uint64

// Limb_Invariants spans every stored word.
func Limb_Invariants(value Limb, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Storage_Limb_0 gives the first arithmetic word one layout identity.
type Storage_Limb_0 Limb

// Storage_Limb_0_Invariants spans every stored word.
func Storage_Limb_0_Invariants(value Storage_Limb_0, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Storage_Limb_1 gives the second arithmetic word one layout identity.
type Storage_Limb_1 Limb

// Storage_Limb_1_Invariants spans every stored word.
func Storage_Limb_1_Invariants(value Storage_Limb_1, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Storage_Limb_2 gives the third arithmetic word one layout identity.
type Storage_Limb_2 Limb

// Storage_Limb_2_Invariants spans every stored word.
func Storage_Limb_2_Invariants(value Storage_Limb_2, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Storage_Limb_3 gives the fourth arithmetic word one layout identity.
type Storage_Limb_3 Limb

// Storage_Limb_3_Invariants spans every stored word.
func Storage_Limb_3_Invariants(value Storage_Limb_3, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Limbs stores one P-256 integer in little-endian machine words.
type Limbs struct {
	// Limb_0 holds the least-significant word.
	Limb_0 Storage_Limb_0
	// Limb_1 holds the second word.
	Limb_1 Storage_Limb_1
	// Limb_2 holds the third word.
	Limb_2 Storage_Limb_2
	// Limb_3 holds the most-significant word.
	Limb_3 Storage_Limb_3
}

// Limbs_Invariants composes every arithmetic word once.
func Limbs_Invariants(value Limbs, namespace aver.Namespace) {
	Storage_Limb_0_Invariants(value.Limb_0, namespace)
	Storage_Limb_1_Invariants(value.Limb_1, namespace)
	Storage_Limb_2_Invariants(value.Limb_2, namespace)
	Storage_Limb_3_Invariants(value.Limb_3, namespace)
}

// Limbs_Handle names mutable P-256-width storage.
type Limbs_Handle *Limbs

// Limbs_Handle_Invariants composes present limb storage.
func Limbs_Handle_Invariants(value Limbs_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Limbs_Invariants(*value, namespace)
}

// Decision is one constant-time Boolean limb.
type Decision uint64

// Decision_Invariants admits only rejected and accepted decisions.
func Decision_Invariants(value Decision, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint64(uint64(value), uint64(DECISION_FALSE), uint64(DECISION_TRUE)).
		Ensure()
}

// Field_Encoding is one exact-width field-element serialization.
type Field_Encoding []byte

// Field_Encoding_Invariants fixes P-256 field width.
func Field_Encoding_Invariants(value Field_Encoding, _ aver.Namespace) {
	aver.Always(len(value) == FIELD_ELEMENT_SIZE, "Field encoding has P-256 width.")
}

// Scalar_Encoding is one exact-width group scalar serialization.
type Scalar_Encoding []byte

// Scalar_Encoding_Invariants fixes P-256 scalar width.
func Scalar_Encoding_Invariants(value Scalar_Encoding, _ aver.Namespace) {
	aver.Always(len(value) == SCALAR_SIZE, "Scalar encoding has P-256 width.")
}

// X_Limb_0 gives the first x word independent invariant identity.
type X_Limb_0 uint64

// X_Limb_0_Invariants spans every stored x word.
func X_Limb_0_Invariants(value X_Limb_0, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// X_Limb_1 gives the second x word independent invariant identity.
type X_Limb_1 uint64

// X_Limb_1_Invariants spans every stored x word.
func X_Limb_1_Invariants(value X_Limb_1, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// X_Limb_2 gives the third x word independent invariant identity.
type X_Limb_2 uint64

// X_Limb_2_Invariants spans every stored x word.
func X_Limb_2_Invariants(value X_Limb_2, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// X_Limb_3 gives the fourth x word independent invariant identity.
type X_Limb_3 uint64

// X_Limb_3_Invariants spans every stored x word.
func X_Limb_3_Invariants(value X_Limb_3, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// X_Coordinate stores one projective x numerator.
type X_Coordinate struct {
	// Limb_0 keeps this position independent in the point invariant chain.
	Limb_0 X_Limb_0
	// Limb_1 keeps this position independent in the point invariant chain.
	Limb_1 X_Limb_1
	// Limb_2 keeps this position independent in the point invariant chain.
	Limb_2 X_Limb_2
	// Limb_3 keeps this position independent in the point invariant chain.
	Limb_3 X_Limb_3
}

// X_Coordinate_Invariants composes every x word once.
func X_Coordinate_Invariants(value X_Coordinate, namespace aver.Namespace) {
	X_Limb_0_Invariants(value.Limb_0, namespace)
	X_Limb_1_Invariants(value.Limb_1, namespace)
	X_Limb_2_Invariants(value.Limb_2, namespace)
	X_Limb_3_Invariants(value.Limb_3, namespace)
}

// X_Coordinate_Handle names mutable x storage.
type X_Coordinate_Handle *X_Coordinate

// X_Coordinate_Handle_Invariants composes present x storage.
func X_Coordinate_Handle_Invariants(value X_Coordinate_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	X_Coordinate_Invariants(*value, namespace)
}

// Y_Limb_0 gives the first y word independent invariant identity.
type Y_Limb_0 uint64

// Y_Limb_0_Invariants spans every stored y word.
func Y_Limb_0_Invariants(value Y_Limb_0, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Y_Limb_1 gives the second y word independent invariant identity.
type Y_Limb_1 uint64

// Y_Limb_1_Invariants spans every stored y word.
func Y_Limb_1_Invariants(value Y_Limb_1, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Y_Limb_2 gives the third y word independent invariant identity.
type Y_Limb_2 uint64

// Y_Limb_2_Invariants spans every stored y word.
func Y_Limb_2_Invariants(value Y_Limb_2, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Y_Limb_3 gives the fourth y word independent invariant identity.
type Y_Limb_3 uint64

// Y_Limb_3_Invariants spans every stored y word.
func Y_Limb_3_Invariants(value Y_Limb_3, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Y_Coordinate stores one projective y numerator.
type Y_Coordinate struct {
	// Limb_0 keeps this position independent in the point invariant chain.
	Limb_0 Y_Limb_0
	// Limb_1 keeps this position independent in the point invariant chain.
	Limb_1 Y_Limb_1
	// Limb_2 keeps this position independent in the point invariant chain.
	Limb_2 Y_Limb_2
	// Limb_3 keeps this position independent in the point invariant chain.
	Limb_3 Y_Limb_3
}

// Y_Coordinate_Invariants composes every y word once.
func Y_Coordinate_Invariants(value Y_Coordinate, namespace aver.Namespace) {
	Y_Limb_0_Invariants(value.Limb_0, namespace)
	Y_Limb_1_Invariants(value.Limb_1, namespace)
	Y_Limb_2_Invariants(value.Limb_2, namespace)
	Y_Limb_3_Invariants(value.Limb_3, namespace)
}

// Y_Coordinate_Handle names mutable y storage.
type Y_Coordinate_Handle *Y_Coordinate

// Y_Coordinate_Handle_Invariants composes present y storage.
func Y_Coordinate_Handle_Invariants(value Y_Coordinate_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Y_Coordinate_Invariants(*value, namespace)
}

// Z_Limb_0 gives the first denominator word independent invariant identity.
type Z_Limb_0 uint64

// Z_Limb_0_Invariants spans every stored denominator word.
func Z_Limb_0_Invariants(value Z_Limb_0, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Z_Limb_1 gives the second denominator word independent invariant identity.
type Z_Limb_1 uint64

// Z_Limb_1_Invariants spans every stored denominator word.
func Z_Limb_1_Invariants(value Z_Limb_1, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Z_Limb_2 gives the third denominator word independent invariant identity.
type Z_Limb_2 uint64

// Z_Limb_2_Invariants spans every stored denominator word.
func Z_Limb_2_Invariants(value Z_Limb_2, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Z_Limb_3 gives the fourth denominator word independent invariant identity.
type Z_Limb_3 uint64

// Z_Limb_3_Invariants spans every stored denominator word.
func Z_Limb_3_Invariants(value Z_Limb_3, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Z_Coordinate stores one projective denominator.
type Z_Coordinate struct {
	// Limb_0 keeps this position independent in the point invariant chain.
	Limb_0 Z_Limb_0
	// Limb_1 keeps this position independent in the point invariant chain.
	Limb_1 Z_Limb_1
	// Limb_2 keeps this position independent in the point invariant chain.
	Limb_2 Z_Limb_2
	// Limb_3 keeps this position independent in the point invariant chain.
	Limb_3 Z_Limb_3
}

// Z_Coordinate_Invariants composes every denominator word once.
func Z_Coordinate_Invariants(value Z_Coordinate, namespace aver.Namespace) {
	Z_Limb_0_Invariants(value.Limb_0, namespace)
	Z_Limb_1_Invariants(value.Limb_1, namespace)
	Z_Limb_2_Invariants(value.Limb_2, namespace)
	Z_Limb_3_Invariants(value.Limb_3, namespace)
}

// Z_Coordinate_Handle names mutable z storage.
type Z_Coordinate_Handle *Z_Coordinate

// Z_Coordinate_Handle_Invariants composes present z storage.
func Z_Coordinate_Handle_Invariants(value Z_Coordinate_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Z_Coordinate_Invariants(*value, namespace)
}

// Point stores validated projective X/Z and Y/Z coordinates.
type Point struct {
	// X is the projective x numerator.
	X X_Coordinate
	// Y is the projective y numerator.
	Y Y_Coordinate
	// Z is the shared projective denominator; zero denotes infinity.
	Z Z_Coordinate
}

// Point_Invariants composes transparent coordinate storage.
func Point_Invariants(value Point, namespace aver.Namespace) {
	X_Coordinate_Invariants(value.X, namespace)
	Y_Coordinate_Invariants(value.Y, namespace)
	Z_Coordinate_Invariants(value.Z, namespace)
}

// Curve validation belongs at public trust boundaries; putting it in the structural bundle makes
// every internal complete-formula step recursively prove the whole curve again.
func point_require(value Point_Handle) {
	Point_Handle_Invariants(value, "point_require.value")
	x := x_limbs(&value.X)
	y := y_limbs(&value.Y)
	z := z_limbs(&value.Z)
	x_canonical := field_canonical(&x)
	y_canonical := field_canonical(&y)
	z_canonical := field_canonical(&z)
	aver.Always(
		x_canonical == DECISION_TRUE,
		"Point x is a canonical field residue.",
	)
	aver.Always(
		y_canonical == DECISION_TRUE,
		"Point y is a canonical field residue.",
	)
	aver.Always(
		z_canonical == DECISION_TRUE,
		"Point z is a canonical field residue.",
	)
	z_zero := field_is_zero(&z)
	on_curve := point_projective_on_curve(&x, &y, &z)
	aver.Always(
		z_zero|on_curve == DECISION_TRUE,
		"A finite Point satisfies the P-256 projective equation.",
	)
}

// Point_Handle gives mutable caller placement a nonnil storage identity.
type Point_Handle *Point

// Point_Handle_Invariants proves storage exists before separate point validation.
func Point_Handle_Invariants(value Point_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Point_Invariants(*value, namespace)
}

func point_set_bytes_output(value Point_Handle) {
	Point_Handle_Invariants(value, "point_set_bytes_output.value")
	Point_Invariants(*value, "Point_Set_Bytes.destination.output")
	point_require(value)
}

func point_scalar_multiply_output(value Point_Handle) {
	Point_Handle_Invariants(value, "point_scalar_multiply_output.value")
	Point_Invariants(*value, "Point_Scalar_Multiply.destination.output")
	point_require(value)
}

func point_scalar_base_multiply_output(value Point_Handle) {
	Point_Handle_Invariants(value, "point_scalar_base_multiply_output.value")
	Point_Invariants(*value, "Point_Scalar_Base_Multiply.destination.output")
	point_require(value)
}

// Distinct coordinate types keep invariant identity. Explicit conversion avoids making layout
// identity a hidden precondition of arithmetic.
func x_limbs(value X_Coordinate_Handle) (limbs Limbs) {
	defer func() { Limbs_Invariants(limbs, "x_limbs.limbs") }()
	X_Coordinate_Handle_Invariants(value, "x_limbs.value")
	aver.Always(value != nil, "X coordinate storage exists.")
	return Limbs{
		Limb_0: Storage_Limb_0(value.Limb_0),
		Limb_1: Storage_Limb_1(value.Limb_1),
		Limb_2: Storage_Limb_2(value.Limb_2),
		Limb_3: Storage_Limb_3(value.Limb_3),
	}
}

func y_limbs(value Y_Coordinate_Handle) (limbs Limbs) {
	defer func() { Limbs_Invariants(limbs, "y_limbs.limbs") }()
	Y_Coordinate_Handle_Invariants(value, "y_limbs.value")
	aver.Always(value != nil, "Y coordinate storage exists.")
	return Limbs{
		Limb_0: Storage_Limb_0(value.Limb_0),
		Limb_1: Storage_Limb_1(value.Limb_1),
		Limb_2: Storage_Limb_2(value.Limb_2),
		Limb_3: Storage_Limb_3(value.Limb_3),
	}
}

func z_limbs(value Z_Coordinate_Handle) (limbs Limbs) {
	defer func() { Limbs_Invariants(limbs, "z_limbs.limbs") }()
	Z_Coordinate_Handle_Invariants(value, "z_limbs.value")
	aver.Always(value != nil, "Z coordinate storage exists.")
	return Limbs{
		Limb_0: Storage_Limb_0(value.Limb_0),
		Limb_1: Storage_Limb_1(value.Limb_1),
		Limb_2: Storage_Limb_2(value.Limb_2),
		Limb_3: Storage_Limb_3(value.Limb_3),
	}
}

func x_coordinate(destination X_Coordinate_Handle, value Limbs) {
	defer func() {
		X_Coordinate_Invariants(*destination, "x_coordinate.destination.output")
	}()
	X_Coordinate_Handle_Invariants(destination, "x_coordinate.destination.input")
	Limbs_Invariants(value, "x_coordinate.value")
	*destination = X_Coordinate{
		Limb_0: X_Limb_0(value.Limb_0),
		Limb_1: X_Limb_1(value.Limb_1),
		Limb_2: X_Limb_2(value.Limb_2),
		Limb_3: X_Limb_3(value.Limb_3),
	}
}

func y_coordinate(destination Y_Coordinate_Handle, value Limbs) {
	defer func() {
		Y_Coordinate_Invariants(*destination, "y_coordinate.destination.output")
	}()
	Y_Coordinate_Handle_Invariants(destination, "y_coordinate.destination.input")
	Limbs_Invariants(value, "y_coordinate.value")
	*destination = Y_Coordinate{
		Limb_0: Y_Limb_0(value.Limb_0),
		Limb_1: Y_Limb_1(value.Limb_1),
		Limb_2: Y_Limb_2(value.Limb_2),
		Limb_3: Y_Limb_3(value.Limb_3),
	}
}

func z_coordinate(destination Z_Coordinate_Handle, value Limbs) {
	defer func() {
		Z_Coordinate_Invariants(*destination, "z_coordinate.destination.output")
	}()
	Z_Coordinate_Handle_Invariants(destination, "z_coordinate.destination.input")
	Limbs_Invariants(value, "z_coordinate.value")
	*destination = Z_Coordinate{
		Limb_0: Z_Limb_0(value.Limb_0),
		Limb_1: Z_Limb_1(value.Limb_1),
		Limb_2: Z_Limb_2(value.Limb_2),
		Limb_3: Z_Limb_3(value.Limb_3),
	}
}

func limb(value Limbs_Handle, index Limb_Index) (result Limb_Value) {
	defer func() { Limb_Value_Invariants(result, "limb.result") }()
	Limbs_Handle_Invariants(value, "limb.value")
	Limb_Index_Invariants(index, "limb.index")
	switch index {
	case LIMB_INDEX_MINIMUM:
		return Limb_Value(value.Limb_0)
	case LIMB_INDEX_SECOND:
		return Limb_Value(value.Limb_1)
	case LIMB_INDEX_THIRD:
		return Limb_Value(value.Limb_2)
	default:
		return Limb_Value(value.Limb_3)
	}
}

func limb_set(destination Limbs_Handle, index Limb_Index, value Limb_Value) {
	Limbs_Handle_Invariants(destination, "limb_set.destination")
	Limb_Index_Invariants(index, "limb_set.index")
	Limb_Value_Invariants(value, "limb_set.value")
	switch index {
	case LIMB_INDEX_MINIMUM:
		destination.Limb_0 = Storage_Limb_0(value)
	case LIMB_INDEX_SECOND:
		destination.Limb_1 = Storage_Limb_1(value)
	case LIMB_INDEX_THIRD:
		destination.Limb_2 = Storage_Limb_2(value)
	default:
		destination.Limb_3 = Storage_Limb_3(value)
	}
}

// Encoding_Unvalidated is one bounded hostile SEC 1 encoding.
type Encoding_Unvalidated []byte

// Encoding_Unvalidated_Invariants bounds parsing before prefix or coordinate access.
func Encoding_Unvalidated_Invariants(
	value Encoding_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), ENCODING_UNVALIDATED_SIZE_MINIMUM,
			ENCODING_UNVALIDATED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Destination is caller-owned SEC 1 output storage.
type Destination []byte

// Destination_Invariants bounds output before any write.
func Destination_Invariants(value Destination, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), DESTINATION_SIZE_MINIMUM, DESTINATION_SIZE_MAXIMUM).
		Ensure()
}

// Scalar_Unvalidated is one bounded hostile big-endian scalar.
type Scalar_Unvalidated []byte

// Scalar_Unvalidated_Invariants admits exact width and bounded refusal lengths.
func Scalar_Unvalidated_Invariants(
	value Scalar_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), SCALAR_UNVALIDATED_SIZE_MINIMUM,
			SCALAR_UNVALIDATED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Encoding_Kind selects compressed or uncompressed finite output.
type Encoding_Kind uint8

// Encoding_Kind_Invariants covers both SEC 1 finite forms.
func Encoding_Kind_Invariants(value Encoding_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(ENCODING_UNCOMPRESSED), uint8(ENCODING_COMPRESSED),
		).
		Ensure()
}

// Count is one complete SEC 1 output width.
type Count uint8

// Count_Invariants admits only infinity, compressed, and uncompressed widths.
func Count_Invariants(value Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(COUNT_INFINITY), uint8(COUNT_COMPRESSED),
			uint8(COUNT_UNCOMPRESSED),
		).
		Ensure()
}

// Parse_Status reports SEC 1 validation.
type Parse_Status uint8

// Parse_Status_Invariants covers complete and refused point input.
func Parse_Status_Invariants(value Parse_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(PARSE_STATUS_OK), uint8(PARSE_STATUS_INPUT_INVALID),
		).
		Ensure()
}

// Output_Status reports caller encoding capacity.
type Output_Status uint8

// Output_Status_Invariants covers complete and short output.
func Output_Status_Invariants(value Output_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(OUTPUT_STATUS_OK),
			uint8(OUTPUT_STATUS_DESTINATION_TOO_SMALL),
		).
		Ensure()
}

// Scalar_Status reports exact-width scalar validation.
type Scalar_Status uint8

// Scalar_Status_Invariants covers complete and refused scalar input.
func Scalar_Status_Invariants(value Scalar_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(SCALAR_STATUS_OK), uint8(SCALAR_STATUS_INPUT_INVALID),
		).
		Ensure()
}

// Equality reports projective group equality.
type Equality bool

// Equality_Invariants covers equal and distinct P-256 points.
func Equality_Invariants(value Equality, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "P-256 points are equal.").
		Ensure()
}

// Point_Identity writes the canonical SEC 1 point at infinity.
func Point_Identity(destination Point_Handle) {
	Point_Handle_Invariants(destination, "Point_Identity.destination")
	var one Limbs
	field_one(&one)
	*destination = Point{}
	y_coordinate(&destination.Y, one)
	point_require(destination)
}

// Point_Generator writes the NIST P-256 canonical generator.
func Point_Generator(destination Point_Handle) {
	Point_Handle_Invariants(destination, "Point_Generator.destination")
	var x, y, one Limbs
	field_generator_x(&x)
	field_generator_y(&y)
	field_one(&one)
	x_coordinate(&destination.X, x)
	y_coordinate(&destination.Y, y)
	z_coordinate(&destination.Z, one)
	point_require(destination)
}

// Point_Set_Bytes validates a complete SEC 1 encoding before replacing destination.
func Point_Set_Bytes(destination Point_Handle, source Encoding_Unvalidated) (
	status Parse_Status,
) {
	defer func() { Parse_Status_Invariants(status, "Point_Set_Bytes.status") }()
	Point_Handle_Invariants(destination, "Point_Set_Bytes.destination.input.handle")
	Point_Invariants(*destination, "Point_Set_Bytes.destination.input.point")
	Encoding_Unvalidated_Invariants(source, "Point_Set_Bytes.source")
	defer func() { point_set_bytes_output(destination) }()
	point_require(destination)
	var point Point
	switch {
	case len(source) == ENCODING_INFINITY_SIZE:
		if source[bits.BIT_COUNT_MINIMUM] != POINT_INFINITY_PREFIX {
			return PARSE_STATUS_INPUT_INVALID
		}
		Point_Identity(&point)
	case len(source) == ENCODING_UNCOMPRESSED_SIZE:
		if source[bits.BIT_COUNT_MINIMUM] != POINT_UNCOMPRESSED_PREFIX {
			return PARSE_STATUS_INPUT_INVALID
		}
		var x_encoding, y_encoding [FIELD_ELEMENT_SIZE]byte
		copy(x_encoding[:], source[ENCODING_INFINITY_SIZE:ENCODING_COMPRESSED_SIZE])
		copy(y_encoding[:], source[ENCODING_COMPRESSED_SIZE:])
		var x, y Limbs
		x_valid := field_decode(&x, Field_Encoding(x_encoding[:]))
		y_valid := field_decode(&y, Field_Encoding(y_encoding[:]))
		on_curve := point_affine_on_curve(&x, &y)
		if x_valid&y_valid&on_curve != DECISION_TRUE {
			return PARSE_STATUS_INPUT_INVALID
		}
		x_coordinate(&point.X, x)
		y_coordinate(&point.Y, y)
		var one Limbs
		field_one(&one)
		z_coordinate(&point.Z, one)
	case len(source) == ENCODING_COMPRESSED_SIZE:
		if source[bits.BIT_COUNT_MINIMUM] != POINT_COMPRESSED_EVEN_PREFIX {
			if source[bits.BIT_COUNT_MINIMUM] != POINT_COMPRESSED_ODD_PREFIX {
				return PARSE_STATUS_INPUT_INVALID
			}
		}
		var x_encoding [FIELD_ELEMENT_SIZE]byte
		copy(x_encoding[:], source[ENCODING_INFINITY_SIZE:])
		var x Limbs
		x_valid := field_decode(&x, Field_Encoding(x_encoding[:]))
		var polynomial Limbs
		field_polynomial(&polynomial, &x)
		var y Limbs
		y_valid := field_square_root(&y, &polynomial)
		if x_valid&y_valid != DECISION_TRUE {
			return PARSE_STATUS_INPUT_INVALID
		}
		var negative Limbs
		field_negate(&negative, &y)
		parity_difference := (Decision(source[bits.BIT_COUNT_MINIMUM]) ^
			Decision(y.Limb_0)) & DECISION_TRUE
		field_select(&y, &negative, &y, parity_difference)
		x_coordinate(&point.X, x)
		y_coordinate(&point.Y, y)
		var one Limbs
		field_one(&one)
		z_coordinate(&point.Z, one)
	default:
		return PARSE_STATUS_INPUT_INVALID
	}
	*destination = point
	return PARSE_STATUS_OK
}

// Point_Bytes_Into derives affine coordinates before touching caller output.
func Point_Bytes_Into(
	destination Destination,
	point Point_Handle,
	kind Encoding_Kind,
) (count Count, _ Output_Status) {
	defer func() { Count_Invariants(count, "Point_Bytes_Into.count") }()
	Destination_Invariants(destination, "Point_Bytes_Into.destination")
	Point_Handle_Invariants(point, "Point_Bytes_Into.point.handle")
	Point_Invariants(*point, "Point_Bytes_Into.point.value")
	Encoding_Kind_Invariants(kind, "Point_Bytes_Into.kind")
	status := OUTPUT_STATUS_OK
	defer func() { Output_Status_Invariants(status, "Point_Bytes_Into.status") }()
	point_require(point)
	z := z_limbs(&point.Z)
	z_zero := field_is_zero(&z)
	if z_zero == DECISION_TRUE {
		count = COUNT_INFINITY
		if len(destination) < int(count) {
			status = OUTPUT_STATUS_DESTINATION_TOO_SMALL
			return count, status
		}
		destination[bits.BIT_COUNT_MINIMUM] = POINT_INFINITY_PREFIX
		return count, status
	}
	if kind == ENCODING_COMPRESSED {
		count = COUNT_COMPRESSED
	} else {
		count = COUNT_UNCOMPRESSED
	}
	if len(destination) < int(count) {
		status = OUTPUT_STATUS_DESTINATION_TOO_SMALL
		return count, status
	}
	var inverse, x, y Limbs
	field_inverse(&inverse, &z)
	point_x := x_limbs(&point.X)
	point_y := y_limbs(&point.Y)
	field_multiply(&x, &point_x, &inverse)
	field_multiply(&y, &point_y, &inverse)
	var x_encoding, y_encoding [FIELD_ELEMENT_SIZE]byte
	field_encode(Field_Encoding(x_encoding[:]), &x)
	field_encode(Field_Encoding(y_encoding[:]), &y)
	if kind == ENCODING_COMPRESSED {
		destination[bits.BIT_COUNT_MINIMUM] = POINT_COMPRESSED_EVEN_PREFIX |
			byte(Decision(y.Limb_0)&DECISION_TRUE)
		copy(destination[ENCODING_INFINITY_SIZE:count], x_encoding[:])
		return count, status
	}
	destination[bits.BIT_COUNT_MINIMUM] = POINT_UNCOMPRESSED_PREFIX
	copy(destination[ENCODING_INFINITY_SIZE:ENCODING_COMPRESSED_SIZE], x_encoding[:])
	copy(destination[ENCODING_COMPRESSED_SIZE:count], y_encoding[:])
	return count, status
}

// Point_Add uses a complete formula so identity and exceptional inputs need no secret branch.
func Point_Add(destination Point_Handle, left Point_Handle, right Point_Handle) {
	Point_Handle_Invariants(destination, "Point_Add.destination.input.handle")
	Point_Invariants(*destination, "Point_Add.destination.input.point")
	Point_Handle_Invariants(left, "Point_Add.left.handle")
	Point_Invariants(*left, "Point_Add.left.point")
	Point_Handle_Invariants(right, "Point_Add.right.handle")
	Point_Invariants(*right, "Point_Add.right.point")
	point_require(destination)
	point_require(left)
	point_require(right)
	point_add(destination, left, right)
	point_require(destination)
}

// Point_Double uses the same complete curve model as addition.
func Point_Double(destination Point_Handle, source Point_Handle) {
	Point_Handle_Invariants(destination, "Point_Double.destination.input.handle")
	Point_Invariants(*destination, "Point_Double.destination.input.point")
	Point_Handle_Invariants(source, "Point_Double.source.handle")
	Point_Invariants(*source, "Point_Double.source.point")
	point_require(destination)
	point_require(source)
	point_double(destination, source)
	point_require(destination)
}

// Point_Scalar_Multiply commits only after exact-width scalar validation.
func Point_Scalar_Multiply(
	destination Point_Handle, point Point_Handle, scalar Scalar_Unvalidated,
) (status Scalar_Status) {
	defer func() { Scalar_Status_Invariants(status, "Point_Scalar_Multiply.status") }()
	Point_Handle_Invariants(
		destination, "Point_Scalar_Multiply.destination.input.handle",
	)
	Point_Invariants(*destination, "Point_Scalar_Multiply.destination.input.point")
	Point_Handle_Invariants(point, "Point_Scalar_Multiply.point.handle")
	Point_Invariants(*point, "Point_Scalar_Multiply.point.value")
	Scalar_Unvalidated_Invariants(scalar, "Point_Scalar_Multiply.scalar")
	defer func() { point_scalar_multiply_output(destination) }()
	point_require(destination)
	point_require(point)
	if len(scalar) != SCALAR_SIZE {
		return SCALAR_STATUS_INPUT_INVALID
	}
	var encoding [SCALAR_SIZE]byte
	copy(encoding[:], scalar)
	var reduced Limbs
	scalar_decode(&reduced, Scalar_Encoding(encoding[:]))
	var result Point
	point_scalar_multiply(&result, point, &reduced)
	*destination = result
	return SCALAR_STATUS_OK
}

// Point_Scalar_Base_Multiply keeps fixed generator choice outside scalar control flow.
func Point_Scalar_Base_Multiply(
	destination Point_Handle, scalar Scalar_Unvalidated,
) (status Scalar_Status) {
	defer func() { Scalar_Status_Invariants(status, "Point_Scalar_Base_Multiply.status") }()
	Point_Handle_Invariants(
		destination, "Point_Scalar_Base_Multiply.destination.input.handle",
	)
	Point_Invariants(
		*destination, "Point_Scalar_Base_Multiply.destination.input.point",
	)
	Scalar_Unvalidated_Invariants(scalar, "Point_Scalar_Base_Multiply.scalar")
	defer func() { point_scalar_base_multiply_output(destination) }()
	point_require(destination)
	if len(scalar) != SCALAR_SIZE {
		return SCALAR_STATUS_INPUT_INVALID
	}
	var generator Point
	Point_Generator(&generator)
	return Point_Scalar_Multiply(destination, &generator, scalar)
}

// Point_Equal compares projective coordinates without affine inversion.
func Point_Equal(left Point_Handle, right Point_Handle) (equal Equality) {
	defer func() { Equality_Invariants(equal, "Point_Equal.equal") }()
	Point_Handle_Invariants(left, "Point_Equal.left.handle")
	Point_Invariants(*left, "Point_Equal.left.point")
	Point_Handle_Invariants(right, "Point_Equal.right.handle")
	Point_Invariants(*right, "Point_Equal.right.point")
	point_require(left)
	point_require(right)
	left_x_coordinate := x_limbs(&left.X)
	left_y_coordinate := y_limbs(&left.Y)
	left_z_coordinate := z_limbs(&left.Z)
	right_x_coordinate := x_limbs(&right.X)
	right_y_coordinate := y_limbs(&right.Y)
	right_z_coordinate := z_limbs(&right.Z)
	left_zero := field_is_zero(&left_z_coordinate)
	right_zero := field_is_zero(&right_z_coordinate)
	var left_x, right_x, left_y, right_y Limbs
	field_multiply(&left_x, &left_x_coordinate, &right_z_coordinate)
	field_multiply(&right_x, &right_x_coordinate, &left_z_coordinate)
	field_multiply(&left_y, &left_y_coordinate, &right_z_coordinate)
	field_multiply(&right_y, &right_y_coordinate, &left_z_coordinate)
	x_equal := field_equal(&left_x, &right_x)
	y_equal := field_equal(&left_y, &right_y)
	both_finite := (left_zero ^ DECISION_TRUE) & (right_zero ^ DECISION_TRUE)
	both_infinite := left_zero & right_zero
	decision := both_infinite | both_finite&x_equal&y_equal
	return Equality(decision == DECISION_TRUE)
}

// Complete addition formula for a = -3 from Renes, Costello, and Batina section A.2.
func point_add(destination Point_Handle, left Point_Handle, right Point_Handle) {
	Point_Handle_Invariants(destination, "point_add.destination")
	Point_Handle_Invariants(left, "point_add.left")
	Point_Handle_Invariants(right, "point_add.right")
	var first, second Point
	point_canonicalize_identity(&first, left)
	point_canonicalize_identity(&second, right)
	left, right = &first, &second
	left_x := x_limbs(&left.X)
	left_y := y_limbs(&left.Y)
	left_z := z_limbs(&left.Z)
	right_x := x_limbs(&right.X)
	right_y := y_limbs(&right.Y)
	right_z := z_limbs(&right.Z)
	var t0, t1, t2, t3, t4 Limbs
	var x3, y3, z3 Limbs
	var b Limbs
	field_b(&b)
	field_multiply(&t0, &left_x, &right_x)
	field_multiply(&t1, &left_y, &right_y)
	field_multiply(&t2, &left_z, &right_z)
	field_add(&t3, &left_x, &left_y)
	field_add(&t4, &right_x, &right_y)
	field_multiply(&t3, &t3, &t4)
	field_add(&t4, &t0, &t1)
	field_subtract(&t3, &t3, &t4)
	field_add(&t4, &left_y, &left_z)
	field_add(&x3, &right_y, &right_z)
	field_multiply(&t4, &t4, &x3)
	field_add(&x3, &t1, &t2)
	field_subtract(&t4, &t4, &x3)
	field_add(&x3, &left_x, &left_z)
	field_add(&y3, &right_x, &right_z)
	field_multiply(&x3, &x3, &y3)
	field_add(&y3, &t0, &t2)
	field_subtract(&y3, &x3, &y3)
	field_multiply(&z3, &b, &t2)
	field_subtract(&x3, &y3, &z3)
	field_add(&z3, &x3, &x3)
	field_add(&x3, &x3, &z3)
	field_subtract(&z3, &t1, &x3)
	field_add(&x3, &t1, &x3)
	field_multiply(&y3, &b, &y3)
	field_add(&t1, &t2, &t2)
	field_add(&t2, &t1, &t2)
	field_subtract(&y3, &y3, &t2)
	field_subtract(&y3, &y3, &t0)
	field_add(&t1, &y3, &y3)
	field_add(&y3, &t1, &y3)
	field_add(&t1, &t0, &t0)
	field_add(&t0, &t1, &t0)
	field_subtract(&t0, &t0, &t2)
	field_multiply(&t1, &t4, &y3)
	field_multiply(&t2, &t0, &y3)
	field_multiply(&y3, &x3, &z3)
	field_add(&y3, &y3, &t2)
	field_multiply(&x3, &t3, &x3)
	field_subtract(&x3, &x3, &t1)
	field_multiply(&z3, &t4, &z3)
	field_multiply(&t1, &t3, &t0)
	field_add(&z3, &z3, &t1)
	x_coordinate(&destination.X, x3)
	y_coordinate(&destination.Y, y3)
	z_coordinate(&destination.Z, z3)
}

// Complete doubling formula keeps zero and all finite intermediates inside one path.
func point_double(destination Point_Handle, source Point_Handle) {
	Point_Handle_Invariants(destination, "point_double.destination")
	Point_Handle_Invariants(source, "point_double.source")
	var point Point
	point_canonicalize_identity(&point, source)
	source = &point
	source_x := x_limbs(&source.X)
	source_y := y_limbs(&source.Y)
	source_z := z_limbs(&source.Z)
	var t0, t1, t2, t3 Limbs
	var x3, y3, z3 Limbs
	var b Limbs
	field_b(&b)
	field_square(&t0, &source_x)
	field_square(&t1, &source_y)
	field_square(&t2, &source_z)
	field_multiply(&t3, &source_x, &source_y)
	field_add(&t3, &t3, &t3)
	field_multiply(&z3, &source_x, &source_z)
	field_add(&z3, &z3, &z3)
	field_multiply(&y3, &b, &t2)
	field_subtract(&y3, &y3, &z3)
	field_add(&x3, &y3, &y3)
	field_add(&y3, &x3, &y3)
	field_subtract(&x3, &t1, &y3)
	field_add(&y3, &t1, &y3)
	field_multiply(&y3, &x3, &y3)
	field_multiply(&x3, &x3, &t3)
	field_add(&t3, &t2, &t2)
	field_add(&t2, &t2, &t3)
	field_multiply(&z3, &b, &z3)
	field_subtract(&z3, &z3, &t2)
	field_subtract(&z3, &z3, &t0)
	field_add(&t3, &z3, &z3)
	field_add(&z3, &z3, &t3)
	field_add(&t3, &t0, &t0)
	field_add(&t0, &t3, &t0)
	field_subtract(&t0, &t0, &t2)
	field_multiply(&t0, &t0, &z3)
	field_add(&y3, &y3, &t0)
	field_multiply(&t0, &source_y, &source_z)
	field_add(&t0, &t0, &t0)
	field_multiply(&z3, &t0, &z3)
	field_subtract(&x3, &x3, &z3)
	field_multiply(&z3, &t0, &t1)
	field_add(&z3, &z3, &z3)
	field_add(&z3, &z3, &z3)
	x_coordinate(&destination.X, x3)
	y_coordinate(&destination.Y, y3)
	z_coordinate(&destination.Z, z3)
}

func point_scalar_multiply(
	destination Point_Handle,
	point Point_Handle,
	scalar Limbs_Handle,
) {
	Point_Handle_Invariants(destination, "point_scalar_multiply.destination")
	Point_Handle_Invariants(point, "point_scalar_multiply.point")
	Limbs_Handle_Invariants(scalar, "point_scalar_multiply.scalar")
	var base Point
	point_canonicalize_identity(&base, point)
	var result Point
	var one Limbs
	field_one(&one)
	y_coordinate(&result.Y, one)
	for bit_count := P256_BIT_COUNT; bit_count > bits.BIT_COUNT_MINIMUM; bit_count-- {
		bit_index := bit_count - binary.UINT_8_SIZE
		var doubled, added Point
		point_double(&doubled, &result)
		point_add(&added, &doubled, &base)
		word_index := bit_index / bits.BIT_COUNT_64_MAXIMUM
		word_shift := uint(bit_index % bits.BIT_COUNT_64_MAXIMUM)
		bit := uint64(limb(scalar, Limb_Index(word_index))) >> word_shift &
			binary.UINT_8_SIZE
		point_select(
			&result, &added, &doubled, Decision(bit),
		)
	}
	*destination = result
}

func point_canonicalize_identity(destination Point_Handle, source Point_Handle) {
	Point_Handle_Invariants(destination, "point_canonicalize_identity.destination")
	Point_Handle_Invariants(source, "point_canonicalize_identity.source")
	var identity Point
	var one Limbs
	field_one(&one)
	y_coordinate(&identity.Y, one)
	source_z := z_limbs(&source.Z)
	z_zero := field_is_zero(&source_z)
	point_select(destination, &identity, source, z_zero)
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
	first_x := x_limbs(&first.X)
	first_y := y_limbs(&first.Y)
	first_z := z_limbs(&first.Z)
	second_x := x_limbs(&second.X)
	second_y := y_limbs(&second.Y)
	second_z := z_limbs(&second.Z)
	var selected_x, selected_y, selected_z Limbs
	field_select(&selected_x, &first_x, &second_x, condition)
	field_select(&selected_y, &first_y, &second_y, condition)
	field_select(&selected_z, &first_z, &second_z, condition)
	x_coordinate(&destination.X, selected_x)
	y_coordinate(&destination.Y, selected_y)
	z_coordinate(&destination.Z, selected_z)
}

func point_affine_on_curve(
	x Limbs_Handle, y Limbs_Handle,
) (valid Decision) {
	defer func() { Decision_Invariants(valid, "point_affine_on_curve.valid") }()
	Limbs_Handle_Invariants(x, "point_affine_on_curve.x")
	Limbs_Handle_Invariants(y, "point_affine_on_curve.y")
	var left, right Limbs
	field_square(&left, y)
	field_polynomial(&right, x)
	return field_equal(&left, &right)
}

func point_projective_on_curve(
	x Limbs_Handle,
	y Limbs_Handle,
	z Limbs_Handle,
) (valid Decision) {
	defer func() { Decision_Invariants(valid, "point_projective_on_curve.valid") }()
	Limbs_Handle_Invariants(x, "point_projective_on_curve.x")
	Limbs_Handle_Invariants(y, "point_projective_on_curve.y")
	Limbs_Handle_Invariants(z, "point_projective_on_curve.z")
	var left, x_square, x_cube, z_square, z_cube Limbs
	var x_z_square, twice, thrice, b_z_cube, right Limbs
	field_square(&left, y)
	field_multiply(&left, &left, z)
	field_square(&x_square, x)
	field_multiply(&x_cube, &x_square, x)
	field_square(&z_square, z)
	field_multiply(&z_cube, &z_square, z)
	field_multiply(&x_z_square, x, &z_square)
	field_add(&twice, &x_z_square, &x_z_square)
	field_add(&thrice, &twice, &x_z_square)
	var b Limbs
	field_b(&b)
	field_multiply(&b_z_cube, &b, &z_cube)
	field_subtract(&right, &x_cube, &thrice)
	field_add(&right, &right, &b_z_cube)
	return field_equal(&left, &right)
}

func field_polynomial(
	destination Limbs_Handle, x Limbs_Handle,
) {
	Limbs_Handle_Invariants(destination, "field_polynomial.destination")
	Limbs_Handle_Invariants(x, "field_polynomial.x")
	var square, cube, twice, thrice Limbs
	field_square(&square, x)
	field_multiply(&cube, &square, x)
	field_add(&twice, x, x)
	field_add(&thrice, &twice, x)
	field_subtract(destination, &cube, &thrice)
	var b Limbs
	field_b(&b)
	field_add(destination, destination, &b)
}

// Bit-serial multiplication trades throughput for small auditable constant-time storage.
func field_multiply(destination Limbs_Handle, left Limbs_Handle, right Limbs_Handle) {
	Limbs_Handle_Invariants(destination, "field_multiply.destination")
	Limbs_Handle_Invariants(left, "field_multiply.left")
	Limbs_Handle_Invariants(right, "field_multiply.right")
	var result_words, addend_words, multiplier, modulus_words [FIELD_LIMB_COUNT]uint64
	var modulus Limbs
	field_modulus(&modulus)
	for index := range result_words {
		limb_index := Limb_Index(index)
		addend_words[index] = uint64(limb(left, limb_index))
		multiplier[index] = uint64(limb(right, limb_index))
		modulus_words[index] = uint64(limb(&modulus, limb_index))
	}
	for bit_index := bits.BIT_COUNT_MINIMUM; bit_index < P256_BIT_COUNT; bit_index++ {
		var additions [binary.UINT_16_SIZE][FIELD_LIMB_COUNT]uint64
		left_operands := [...]*[FIELD_LIMB_COUNT]uint64{&result_words, &addend_words}
		for operation_index := range additions {
			left_words := left_operands[operation_index]
			carry_value := uint64(bits.WORD_64_MINIMUM)
			var sum [FIELD_LIMB_COUNT]uint64
			for index := range sum {
				partial := left_words[index] + addend_words[index]
				partial_carry := ((left_words[index] & addend_words[index]) |
					((left_words[index] | addend_words[index]) & ^partial)) >>
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
			reduce := carry_value | (borrow_value ^ uint64(DECISION_TRUE))
			mask := uint64(bits.WORD_64_MINIMUM) - reduce
			for index := range additions[operation_index] {
				additions[operation_index][index] = sum[index] ^
					mask&(reduced[index]^sum[index])
			}
		}
		word_index, word_shift := bit_index/bits.BIT_COUNT_64_MAXIMUM,
			uint(bit_index%bits.BIT_COUNT_64_MAXIMUM)
		bit := multiplier[word_index] >> word_shift & binary.UINT_8_SIZE
		mask := uint64(bits.WORD_64_MINIMUM) - bit
		for index := range result_words {
			result_words[index] ^= mask & (additions[0][index] ^ result_words[index])
		}
		addend_words = additions[1]
	}
	for index := range result_words {
		limb_set(destination, Limb_Index(index), Limb_Value(result_words[index]))
	}
}

func field_square(
	destination Limbs_Handle, source Limbs_Handle,
) {
	Limbs_Handle_Invariants(destination, "field_square.destination")
	Limbs_Handle_Invariants(source, "field_square.source")
	field_multiply(destination, source, source)
}

func field_add(
	destination Limbs_Handle,
	left Limbs_Handle,
	right Limbs_Handle,
) {
	Limbs_Handle_Invariants(destination, "field_add.destination")
	Limbs_Handle_Invariants(left, "field_add.left")
	Limbs_Handle_Invariants(right, "field_add.right")
	var sum, reduced Limbs
	carry := limbs_add(&sum, left, right)
	var modulus Limbs
	field_modulus(&modulus)
	borrow := limbs_subtract(&reduced, &sum, &modulus)
	reduce := carry | (borrow ^ DECISION_TRUE)
	limbs_select(destination, &reduced, &sum, reduce)
}

func field_subtract(
	destination Limbs_Handle,
	left Limbs_Handle,
	right Limbs_Handle,
) {
	Limbs_Handle_Invariants(destination, "field_subtract.destination")
	Limbs_Handle_Invariants(left, "field_subtract.left")
	Limbs_Handle_Invariants(right, "field_subtract.right")
	var difference, correction Limbs
	borrow := limbs_subtract(&difference, left, right)
	var modulus Limbs
	field_modulus(&modulus)
	mask := uint64(bits.WORD_64_MINIMUM) - uint64(borrow)
	for index := bits.BIT_COUNT_MINIMUM; index < FIELD_LIMB_COUNT; index++ {
		limb_index := Limb_Index(index)
		value := uint64(limb(&modulus, limb_index)) & mask
		limb_set(&correction, limb_index, Limb_Value(value))
	}
	limbs_add(destination, &difference, &correction)
}

func field_negate(
	destination Limbs_Handle, source Limbs_Handle,
) {
	Limbs_Handle_Invariants(destination, "field_negate.destination")
	Limbs_Handle_Invariants(source, "field_negate.source")
	var zero Limbs
	field_subtract(destination, &zero, source)
}

func field_select(
	destination Limbs_Handle,
	first Limbs_Handle,
	second Limbs_Handle,
	condition Decision,
) {
	Limbs_Handle_Invariants(destination, "field_select.destination")
	Limbs_Handle_Invariants(first, "field_select.first")
	Limbs_Handle_Invariants(second, "field_select.second")
	Decision_Invariants(condition, "field_select.condition")
	mask := uint64(bits.WORD_64_MINIMUM) - uint64(condition)
	var selected Limbs
	for index := bits.BIT_COUNT_MINIMUM; index < FIELD_LIMB_COUNT; index++ {
		limb_index := Limb_Index(index)
		first_value := uint64(limb(first, limb_index))
		second_value := uint64(limb(second, limb_index))
		selected_value := second_value ^ mask&(first_value^second_value)
		limb_set(&selected, limb_index, Limb_Value(selected_value))
	}
	*destination = selected
}

func field_equal(
	left Limbs_Handle, right Limbs_Handle,
) (equal Decision) {
	defer func() { Decision_Invariants(equal, "field_equal.equal") }()
	Limbs_Handle_Invariants(left, "field_equal.left")
	Limbs_Handle_Invariants(right, "field_equal.right")
	difference := uint64(bits.WORD_64_MINIMUM)
	for index := bits.BIT_COUNT_MINIMUM; index < FIELD_LIMB_COUNT; index++ {
		limb_index := Limb_Index(index)
		difference |= uint64(limb(left, limb_index)) ^ uint64(limb(right, limb_index))
	}
	equal = Decision((difference|-difference)>>
		(bits.BIT_COUNT_64_MAXIMUM-binary.UINT_8_SIZE) ^ binary.UINT_8_SIZE)
	return equal
}

func field_is_zero(value Limbs_Handle) (zero Decision) {
	defer func() { Decision_Invariants(zero, "field_is_zero.zero") }()
	Limbs_Handle_Invariants(value, "field_is_zero.value")
	var empty Limbs
	return field_equal(value, &empty)
}

func field_canonical(value Limbs_Handle) (canonical Decision) {
	defer func() { Decision_Invariants(canonical, "field_canonical.canonical") }()
	Limbs_Handle_Invariants(value, "field_canonical.value")
	var difference Limbs
	var modulus Limbs
	field_modulus(&modulus)
	return limbs_subtract(&difference, value, &modulus)
}

func field_inverse(
	destination Limbs_Handle, source Limbs_Handle,
) {
	Limbs_Handle_Invariants(destination, "field_inverse.destination")
	Limbs_Handle_Invariants(source, "field_inverse.source")
	var exponent Limbs
	field_inverse_exponent(&exponent)
	field_power(destination, source, &exponent)
}

func field_square_root(
	destination Limbs_Handle,
	source Limbs_Handle,
) (valid Decision) {
	defer func() { Decision_Invariants(valid, "field_square_root.valid") }()
	Limbs_Handle_Invariants(destination, "field_square_root.destination")
	Limbs_Handle_Invariants(source, "field_square_root.source")
	var exponent Limbs
	field_square_root_exponent(&exponent)
	var candidate, square Limbs
	field_power(&candidate, source, &exponent)
	field_square(&square, &candidate)
	valid = field_equal(&square, source)
	*destination = candidate
	return valid
}

func field_power(
	destination Limbs_Handle,
	base Limbs_Handle,
	exponent Limbs_Handle,
) {
	Limbs_Handle_Invariants(destination, "field_power.destination")
	Limbs_Handle_Invariants(base, "field_power.base")
	Limbs_Handle_Invariants(exponent, "field_power.exponent")
	var result Limbs
	field_one(&result)
	factor := *base
	for bit_count := P256_BIT_COUNT; bit_count > bits.BIT_COUNT_MINIMUM; bit_count-- {
		bit_index := bit_count - binary.UINT_8_SIZE
		var square, product Limbs
		field_square(&square, &result)
		field_multiply(&product, &square, &factor)
		word_index := bit_index / bits.BIT_COUNT_64_MAXIMUM
		word_shift := uint(bit_index % bits.BIT_COUNT_64_MAXIMUM)
		bit := uint64(limb(exponent, Limb_Index(word_index))) >> word_shift &
			binary.UINT_8_SIZE
		field_select(&result, &product, &square, Decision(bit))
	}
	*destination = result
}

func field_decode(
	destination Limbs_Handle,
	source Field_Encoding,
) (valid Decision) {
	defer func() { Decision_Invariants(valid, "field_decode.valid") }()
	Limbs_Handle_Invariants(destination, "field_decode.destination")
	Field_Encoding_Invariants(source, "field_decode.source")
	for index := bits.BIT_COUNT_MINIMUM; index < FIELD_LIMB_COUNT; index++ {
		source_index := FIELD_ELEMENT_SIZE -
			(index+binary.UINT_8_SIZE)*binary.UINT_64_SIZE
		value := binary.Uint_64(
			binary.Bytes(source[source_index:source_index+binary.UINT_64_SIZE]),
			binary.BIG_ENDIAN,
		)
		limb_set(destination, Limb_Index(index), Limb_Value(value))
	}
	return field_canonical(destination)
}

func field_encode(
	destination Field_Encoding, source Limbs_Handle,
) {
	Field_Encoding_Invariants(destination, "field_encode.destination")
	Limbs_Handle_Invariants(source, "field_encode.source")
	for index := bits.BIT_COUNT_MINIMUM; index < FIELD_LIMB_COUNT; index++ {
		destination_index := FIELD_ELEMENT_SIZE -
			(index+binary.UINT_8_SIZE)*binary.UINT_64_SIZE
		destination_end := destination_index + binary.UINT_64_SIZE
		binary.Put_Uint_64(
			binary.Bytes(destination[destination_index:destination_end]),
			binary.Word_64(limb(source, Limb_Index(index))), binary.BIG_ENDIAN,
		)
	}
}

func scalar_decode(
	destination Limbs_Handle, source Scalar_Encoding,
) {
	Limbs_Handle_Invariants(destination, "scalar_decode.destination")
	Scalar_Encoding_Invariants(source, "scalar_decode.source")
	for index := bits.BIT_COUNT_MINIMUM; index < FIELD_LIMB_COUNT; index++ {
		source_index := SCALAR_SIZE -
			(index+binary.UINT_8_SIZE)*binary.UINT_64_SIZE
		value := binary.Uint_64(
			binary.Bytes(source[source_index:source_index+binary.UINT_64_SIZE]),
			binary.BIG_ENDIAN,
		)
		limb_set(destination, Limb_Index(index), Limb_Value(value))
	}
	var order Limbs
	scalar_order(&order)
	var reduced Limbs
	borrow := limbs_subtract(&reduced, destination, &order)
	limbs_select(destination, &reduced, destination, borrow^DECISION_TRUE)
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
	carry_value := uint64(bits.WORD_64_MINIMUM)
	var sum Limbs
	for index := bits.BIT_COUNT_MINIMUM; index < FIELD_LIMB_COUNT; index++ {
		limb_index := Limb_Index(index)
		left_value := uint64(limb(left, limb_index))
		right_value := uint64(limb(right, limb_index))
		partial := left_value + right_value
		partial_carry := ((left_value & right_value) |
			((left_value | right_value) & ^partial)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		value := partial + carry_value
		carry_carry := ((partial & carry_value) |
			((partial | carry_value) & ^value)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		limb_set(&sum, limb_index, Limb_Value(value))
		carry_value = partial_carry | carry_carry
	}
	*destination = sum
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
	borrow_value := uint64(bits.WORD_64_MINIMUM)
	var difference Limbs
	for index := bits.BIT_COUNT_MINIMUM; index < FIELD_LIMB_COUNT; index++ {
		limb_index := Limb_Index(index)
		left_value := uint64(limb(left, limb_index))
		right_value := uint64(limb(right, limb_index))
		partial := left_value - right_value
		partial_borrow := ((^left_value & right_value) |
			(^(left_value ^ right_value) & partial)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		value := partial - borrow_value
		borrow_borrow := ((^partial & borrow_value) |
			(^(partial ^ borrow_value) & value)) >>
			(bits.BIT_COUNT_64_MAXIMUM - binary.UINT_8_SIZE)
		limb_set(&difference, limb_index, Limb_Value(value))
		borrow_value = partial_borrow | borrow_borrow
	}
	*destination = difference
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
	var selected Limbs
	for index := bits.BIT_COUNT_MINIMUM; index < FIELD_LIMB_COUNT; index++ {
		limb_index := Limb_Index(index)
		first_value := uint64(limb(first, limb_index))
		second_value := uint64(limb(second, limb_index))
		selected_value := second_value ^ mask&(first_value^second_value)
		limb_set(&selected, limb_index, Limb_Value(selected_value))
	}
	*destination = selected
}

// These limbs are the canonical NIST P-256 prime in little-endian machine order.
func field_modulus(destination Limbs_Handle) {
	Limbs_Handle_Invariants(destination, "field_modulus.destination")
	*destination = Limbs{
		Limb_0: 0xffffffffffffffff,
		Limb_1: 0x00000000ffffffff,
		Limb_2: 0x0000000000000000,
		Limb_3: 0xffffffff00000001,
	}
}

// P-256's curve coefficient is the SEC 2 value in little-endian machine order.
func field_b(destination Limbs_Handle) {
	Limbs_Handle_Invariants(destination, "field_b.destination")
	*destination = Limbs{
		Limb_0: 0x3bce3c3e27d2604b,
		Limb_1: 0x651d06b0cc53b0f6,
		Limb_2: 0xb3ebbd55769886bc,
		Limb_3: 0x5ac635d8aa3a93e7,
	}
}

func field_one(destination Limbs_Handle) {
	Limbs_Handle_Invariants(destination, "field_one.destination")
	*destination = Limbs{Limb_0: binary.UINT_8_SIZE}
}

// Generator coordinates are the canonical SEC 2 P-256 base point.
func field_generator_x(destination Limbs_Handle) {
	Limbs_Handle_Invariants(destination, "field_generator_x.destination")
	*destination = Limbs{
		Limb_0: 0xf4a13945d898c296,
		Limb_1: 0x77037d812deb33a0,
		Limb_2: 0xf8bce6e563a440f2,
		Limb_3: 0x6b17d1f2e12c4247,
	}
}

func field_generator_y(destination Limbs_Handle) {
	Limbs_Handle_Invariants(destination, "field_generator_y.destination")
	*destination = Limbs{
		Limb_0: 0xcbb6406837bf51f5,
		Limb_1: 0x2bce33576b315ece,
		Limb_2: 0x8ee7eb4a7c0f9e16,
		Limb_3: 0x4fe342e2fe1a7f9b,
	}
}

// Fermat inversion uses p minus two, derived from field_modulus.
func field_inverse_exponent(destination Limbs_Handle) {
	Limbs_Handle_Invariants(destination, "field_inverse_exponent.destination")
	*destination = Limbs{
		Limb_0: 0xfffffffffffffffd,
		Limb_1: 0x00000000ffffffff,
		Limb_2: 0x0000000000000000,
		Limb_3: 0xffffffff00000001,
	}
}

// P-256 is three modulo four, so square roots use the exponent (p plus one) divided by four.
func field_square_root_exponent(destination Limbs_Handle) {
	Limbs_Handle_Invariants(destination, "field_square_root_exponent.destination")
	*destination = Limbs{
		Limb_0: 0x0000000000000000,
		Limb_1: 0x0000000040000000,
		Limb_2: 0x4000000000000000,
		Limb_3: 0x3fffffffc0000000,
	}
}

// The group order is the canonical SEC 2 P-256 order in little-endian machine order.
func scalar_order(destination Limbs_Handle) {
	Limbs_Handle_Invariants(destination, "scalar_order.destination")
	*destination = Limbs{
		Limb_0: 0xf3b9cac2fc632551,
		Limb_1: 0xbce6faada7179e84,
		Limb_2: 0xffffffffffffffff,
		Limb_3: 0xffffffff00000000,
	}
}
