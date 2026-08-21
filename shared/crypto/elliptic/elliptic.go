// Package elliptic implements bounded caller-owned NIST P-256 group operations.
package elliptic

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
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

// CONDITION_LIMB_COUNT gives branchless decisions fixed array identity.
const CONDITION_LIMB_COUNT = binary.UINT_8_SIZE

// Point stores validated projective X/Z and Y/Z coordinates.
type Point struct {
	// X is the projective x numerator.
	X [FIELD_LIMB_COUNT]uint64
	// Y is the projective y numerator.
	Y [FIELD_LIMB_COUNT]uint64
	// Z is the shared projective denominator; zero denotes infinity.
	Z [FIELD_LIMB_COUNT]uint64
}

// Point_Invariants proves canonical residues and the complete projective curve relation.
func Point_Invariants(value Point, _ invariant.Namespace) {
	invariant.Always(len(value.X) == FIELD_LIMB_COUNT, "Point x has P-256 width.")
	invariant.Always(len(value.Y) == FIELD_LIMB_COUNT, "Point y has P-256 width.")
	invariant.Always(len(value.Z) == FIELD_LIMB_COUNT, "Point z has P-256 width.")
	x_canonical := field_canonical(&value.X)
	y_canonical := field_canonical(&value.Y)
	z_canonical := field_canonical(&value.Z)
	invariant.Always(
		x_canonical[bits.BIT_COUNT_MINIMUM] == binary.UINT_8_SIZE,
		"Point x is a canonical field residue.",
	)
	invariant.Always(
		y_canonical[bits.BIT_COUNT_MINIMUM] == binary.UINT_8_SIZE,
		"Point y is a canonical field residue.",
	)
	invariant.Always(
		z_canonical[bits.BIT_COUNT_MINIMUM] == binary.UINT_8_SIZE,
		"Point z is a canonical field residue.",
	)
	z_zero := field_is_zero(&value.Z)
	on_curve := point_projective_on_curve(&value.X, &value.Y, &value.Z)
	invariant.Always(
		z_zero[bits.BIT_COUNT_MINIMUM]|on_curve[bits.BIT_COUNT_MINIMUM] ==
			binary.UINT_8_SIZE,
		"A finite Point satisfies the P-256 projective equation.",
	)
}

// Point_Handle gives mutable caller placement a nonnil storage identity.
type Point_Handle *Point

// Point_Handle_Invariants proves storage exists before separate point validation.
func Point_Handle_Invariants(value Point_Handle, _ invariant.Namespace) {
	invariant.Always(value != nil, "A Point handle has caller-owned storage.")
}

// Encoding_Unvalidated is one bounded hostile SEC 1 encoding.
type Encoding_Unvalidated []byte

// Encoding_Unvalidated_Invariants bounds parsing before prefix or coordinate access.
func Encoding_Unvalidated_Invariants(
	value Encoding_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), ENCODING_UNVALIDATED_SIZE_MINIMUM,
			ENCODING_UNVALIDATED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Destination is caller-owned SEC 1 output storage.
type Destination []byte

// Destination_Invariants bounds output before any write.
func Destination_Invariants(value Destination, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), DESTINATION_SIZE_MINIMUM, DESTINATION_SIZE_MAXIMUM).
		Ensure()
}

// Scalar_Unvalidated is one bounded hostile big-endian scalar.
type Scalar_Unvalidated []byte

// Scalar_Unvalidated_Invariants admits exact width and bounded refusal lengths.
func Scalar_Unvalidated_Invariants(
	value Scalar_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), SCALAR_UNVALIDATED_SIZE_MINIMUM,
			SCALAR_UNVALIDATED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Encoding_Kind selects compressed or uncompressed finite output.
type Encoding_Kind uint8

// Encoding_Kind_Invariants covers both SEC 1 finite forms.
func Encoding_Kind_Invariants(value Encoding_Kind, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(ENCODING_UNCOMPRESSED), uint8(ENCODING_COMPRESSED),
		).
		Ensure()
}

// Count is one complete SEC 1 output width.
type Count uint8

// Count_Invariants admits only infinity, compressed, and uncompressed widths.
func Count_Invariants(value Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(COUNT_INFINITY), uint8(COUNT_COMPRESSED),
			uint8(COUNT_UNCOMPRESSED),
		).
		Ensure()
}

// Parse_Status reports SEC 1 validation.
type Parse_Status uint8

// Parse_Status_Invariants covers complete and refused point input.
func Parse_Status_Invariants(value Parse_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(PARSE_STATUS_OK), uint8(PARSE_STATUS_INPUT_INVALID),
		).
		Ensure()
}

// Output_Status reports caller encoding capacity.
type Output_Status uint8

// Output_Status_Invariants covers complete and short output.
func Output_Status_Invariants(value Output_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(OUTPUT_STATUS_OK),
			uint8(OUTPUT_STATUS_DESTINATION_TOO_SMALL),
		).
		Ensure()
}

// Scalar_Status reports exact-width scalar validation.
type Scalar_Status uint8

// Scalar_Status_Invariants covers complete and refused scalar input.
func Scalar_Status_Invariants(value Scalar_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(SCALAR_STATUS_OK), uint8(SCALAR_STATUS_INPUT_INVALID),
		).
		Ensure()
}

// Equality reports projective group equality.
type Equality bool

// Equality_Invariants covers equal and distinct P-256 points.
func Equality_Invariants(value Equality, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "P-256 points are equal.").
		Ensure()
}

// Point_Identity returns the canonical SEC 1 point at infinity.
func Point_Identity() (point Point) {
	defer func() { Point_Invariants(point, "Point_Identity.point") }()
	point.Y = field_one()
	return point
}

// Point_Generator returns the NIST P-256 canonical generator.
func Point_Generator() (point Point) {
	defer func() { Point_Invariants(point, "Point_Generator.point") }()
	point.X = field_generator_x()
	point.Y = field_generator_y()
	point.Z = field_one()
	return point
}

// Point_Set_Bytes validates a complete SEC 1 encoding before replacing destination.
func Point_Set_Bytes(
	destination Point_Handle, source Encoding_Unvalidated,
) (status Parse_Status) {
	defer func() {
		Parse_Status_Invariants(status, "Point_Set_Bytes.status")
		Point_Invariants(*destination, "Point_Set_Bytes.destination.output")
	}()
	Point_Handle_Invariants(destination, "Point_Set_Bytes.destination.input.handle")
	Point_Invariants(*destination, "Point_Set_Bytes.destination.input.point")
	Encoding_Unvalidated_Invariants(source, "Point_Set_Bytes.source")
	if len(source) > ENCODING_UNVALIDATED_SIZE_MAXIMUM {
		panic("elliptic: encoding exceeds bound")
	}
	var point Point
	switch {
	case len(source) == ENCODING_INFINITY_SIZE:
		if source[bits.BIT_COUNT_MINIMUM] != POINT_INFINITY_PREFIX {
			return PARSE_STATUS_INPUT_INVALID
		}
		point = Point_Identity()
	case len(source) == ENCODING_UNCOMPRESSED_SIZE:
		if source[bits.BIT_COUNT_MINIMUM] != POINT_UNCOMPRESSED_PREFIX {
			return PARSE_STATUS_INPUT_INVALID
		}
		var x_encoding, y_encoding [FIELD_ELEMENT_SIZE]byte
		copy(x_encoding[:], source[ENCODING_INFINITY_SIZE:ENCODING_COMPRESSED_SIZE])
		copy(y_encoding[:], source[ENCODING_COMPRESSED_SIZE:])
		x_valid := field_decode(&point.X, &x_encoding)
		y_valid := field_decode(&point.Y, &y_encoding)
		on_curve := point_affine_on_curve(&point.X, &point.Y)
		if x_valid[bits.BIT_COUNT_MINIMUM]&y_valid[bits.BIT_COUNT_MINIMUM]&
			on_curve[bits.BIT_COUNT_MINIMUM] != binary.UINT_8_SIZE {
			return PARSE_STATUS_INPUT_INVALID
		}
		point.Z = field_one()
	case len(source) == ENCODING_COMPRESSED_SIZE:
		if source[bits.BIT_COUNT_MINIMUM] != POINT_COMPRESSED_EVEN_PREFIX {
			if source[bits.BIT_COUNT_MINIMUM] != POINT_COMPRESSED_ODD_PREFIX {
				return PARSE_STATUS_INPUT_INVALID
			}
		}
		var x_encoding [FIELD_ELEMENT_SIZE]byte
		copy(x_encoding[:], source[ENCODING_INFINITY_SIZE:])
		x_valid := field_decode(&point.X, &x_encoding)
		var polynomial [FIELD_LIMB_COUNT]uint64
		field_polynomial(&polynomial, &point.X)
		y_valid := field_square_root(&point.Y, &polynomial)
		if x_valid[bits.BIT_COUNT_MINIMUM]&y_valid[bits.BIT_COUNT_MINIMUM] !=
			binary.UINT_8_SIZE {
			return PARSE_STATUS_INPUT_INVALID
		}
		var negative [FIELD_LIMB_COUNT]uint64
		field_negate(&negative, &point.Y)
		wanted_parity := uint64(
			source[bits.BIT_COUNT_MINIMUM] & byte(binary.UINT_8_SIZE),
		)
		actual_parity := point.Y[bits.BIT_COUNT_MINIMUM] & binary.UINT_8_SIZE
		field_select(
			&point.Y, &negative, &point.Y,
			[CONDITION_LIMB_COUNT]uint64{wanted_parity ^ actual_parity},
		)
		point.Z = field_one()
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
) (count Count, status Output_Status) {
	defer func() {
		Count_Invariants(count, "Point_Bytes_Into.count")
		Output_Status_Invariants(status, "Point_Bytes_Into.status")
	}()
	Destination_Invariants(destination, "Point_Bytes_Into.destination")
	Point_Handle_Invariants(point, "Point_Bytes_Into.point.handle")
	Point_Invariants(*point, "Point_Bytes_Into.point.value")
	Encoding_Kind_Invariants(kind, "Point_Bytes_Into.kind")
	if len(destination) > DESTINATION_SIZE_MAXIMUM {
		panic("elliptic: destination exceeds bound")
	}
	if kind > ENCODING_COMPRESSED {
		panic("elliptic: encoding kind is invalid")
	}
	z_zero := field_is_zero(&point.Z)
	if z_zero[bits.BIT_COUNT_MINIMUM] == binary.UINT_8_SIZE {
		count = COUNT_INFINITY
		if len(destination) < int(count) {
			return count, OUTPUT_STATUS_DESTINATION_TOO_SMALL
		}
		destination[bits.BIT_COUNT_MINIMUM] = POINT_INFINITY_PREFIX
		return count, OUTPUT_STATUS_OK
	}
	if kind == ENCODING_COMPRESSED {
		count = COUNT_COMPRESSED
	} else {
		count = COUNT_UNCOMPRESSED
	}
	if len(destination) < int(count) {
		return count, OUTPUT_STATUS_DESTINATION_TOO_SMALL
	}
	var inverse, x, y [FIELD_LIMB_COUNT]uint64
	field_inverse(&inverse, &point.Z)
	field_multiply(&x, &point.X, &inverse)
	field_multiply(&y, &point.Y, &inverse)
	var x_encoding, y_encoding [FIELD_ELEMENT_SIZE]byte
	field_encode(&x_encoding, &x)
	field_encode(&y_encoding, &y)
	if kind == ENCODING_COMPRESSED {
		destination[bits.BIT_COUNT_MINIMUM] = POINT_COMPRESSED_EVEN_PREFIX |
			byte(y[bits.BIT_COUNT_MINIMUM]&binary.UINT_8_SIZE)
		copy(destination[ENCODING_INFINITY_SIZE:count], x_encoding[:])
		return count, OUTPUT_STATUS_OK
	}
	destination[bits.BIT_COUNT_MINIMUM] = POINT_UNCOMPRESSED_PREFIX
	copy(destination[ENCODING_INFINITY_SIZE:ENCODING_COMPRESSED_SIZE], x_encoding[:])
	copy(destination[ENCODING_COMPRESSED_SIZE:count], y_encoding[:])
	return count, OUTPUT_STATUS_OK
}

// Point_Add uses a complete formula so identity and exceptional inputs need no secret branch.
func Point_Add(destination Point_Handle, left Point_Handle, right Point_Handle) {
	Point_Handle_Invariants(destination, "Point_Add.destination.input.handle")
	Point_Invariants(*destination, "Point_Add.destination.input.point")
	Point_Handle_Invariants(left, "Point_Add.left.handle")
	Point_Invariants(*left, "Point_Add.left.point")
	Point_Handle_Invariants(right, "Point_Add.right.handle")
	Point_Invariants(*right, "Point_Add.right.point")
	point_add(destination, left, right)
	Point_Invariants(*destination, "Point_Add.destination.output")
}

// Point_Double uses the same complete curve model as addition.
func Point_Double(destination Point_Handle, source Point_Handle) {
	Point_Handle_Invariants(destination, "Point_Double.destination.input.handle")
	Point_Invariants(*destination, "Point_Double.destination.input.point")
	Point_Handle_Invariants(source, "Point_Double.source.handle")
	Point_Invariants(*source, "Point_Double.source.point")
	point_double(destination, source)
	Point_Invariants(*destination, "Point_Double.destination.output")
}

// Point_Scalar_Multiply commits only after exact-width scalar validation.
func Point_Scalar_Multiply(
	destination Point_Handle, point Point_Handle, scalar Scalar_Unvalidated,
) (status Scalar_Status) {
	defer func() {
		Scalar_Status_Invariants(status, "Point_Scalar_Multiply.status")
		Point_Invariants(*destination, "Point_Scalar_Multiply.destination.output")
	}()
	Point_Handle_Invariants(
		destination, "Point_Scalar_Multiply.destination.input.handle",
	)
	Point_Invariants(*destination, "Point_Scalar_Multiply.destination.input.point")
	Point_Handle_Invariants(point, "Point_Scalar_Multiply.point.handle")
	Point_Invariants(*point, "Point_Scalar_Multiply.point.value")
	Scalar_Unvalidated_Invariants(scalar, "Point_Scalar_Multiply.scalar")
	if len(scalar) > SCALAR_UNVALIDATED_SIZE_MAXIMUM {
		panic("elliptic: scalar exceeds bound")
	}
	if len(scalar) != SCALAR_SIZE {
		return SCALAR_STATUS_INPUT_INVALID
	}
	var encoding [SCALAR_SIZE]byte
	copy(encoding[:], scalar)
	var reduced [FIELD_LIMB_COUNT]uint64
	scalar_decode(&reduced, &encoding)
	var result Point
	point_scalar_multiply(&result, point, &reduced)
	*destination = result
	return SCALAR_STATUS_OK
}

// Point_Scalar_Base_Multiply keeps fixed generator choice outside scalar control flow.
func Point_Scalar_Base_Multiply(
	destination Point_Handle, scalar Scalar_Unvalidated,
) (status Scalar_Status) {
	defer func() {
		Scalar_Status_Invariants(status, "Point_Scalar_Base_Multiply.status")
		Point_Invariants(*destination, "Point_Scalar_Base_Multiply.destination.output")
	}()
	Point_Handle_Invariants(
		destination, "Point_Scalar_Base_Multiply.destination.input.handle",
	)
	Point_Invariants(
		*destination, "Point_Scalar_Base_Multiply.destination.input.point",
	)
	Scalar_Unvalidated_Invariants(scalar, "Point_Scalar_Base_Multiply.scalar")
	if len(scalar) > SCALAR_UNVALIDATED_SIZE_MAXIMUM {
		panic("elliptic: scalar exceeds bound")
	}
	if len(scalar) != SCALAR_SIZE {
		return SCALAR_STATUS_INPUT_INVALID
	}
	generator := Point_Generator()
	return Point_Scalar_Multiply(destination, &generator, scalar)
}

// Point_Equal compares projective coordinates without affine inversion.
func Point_Equal(left Point_Handle, right Point_Handle) (equal Equality) {
	defer func() { Equality_Invariants(equal, "Point_Equal.equal") }()
	Point_Handle_Invariants(left, "Point_Equal.left.handle")
	Point_Invariants(*left, "Point_Equal.left.point")
	Point_Handle_Invariants(right, "Point_Equal.right.handle")
	Point_Invariants(*right, "Point_Equal.right.point")
	left_zero := field_is_zero(&left.Z)
	right_zero := field_is_zero(&right.Z)
	var left_x, right_x, left_y, right_y [FIELD_LIMB_COUNT]uint64
	field_multiply(&left_x, &left.X, &right.Z)
	field_multiply(&right_x, &right.X, &left.Z)
	field_multiply(&left_y, &left.Y, &right.Z)
	field_multiply(&right_y, &right.Y, &left.Z)
	x_equal := field_equal(&left_x, &right_x)
	y_equal := field_equal(&left_y, &right_y)
	both_finite := (left_zero[bits.BIT_COUNT_MINIMUM] ^ binary.UINT_8_SIZE) &
		(right_zero[bits.BIT_COUNT_MINIMUM] ^ binary.UINT_8_SIZE)
	both_infinite := left_zero[bits.BIT_COUNT_MINIMUM] &
		right_zero[bits.BIT_COUNT_MINIMUM]
	decision := both_infinite | both_finite&x_equal[bits.BIT_COUNT_MINIMUM]&
		y_equal[bits.BIT_COUNT_MINIMUM]
	return Equality(decision == binary.UINT_8_SIZE)
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
	var t0, t1, t2, t3, t4 [FIELD_LIMB_COUNT]uint64
	var x3, y3, z3 [FIELD_LIMB_COUNT]uint64
	b := field_b()
	field_multiply(&t0, &left.X, &right.X)
	field_multiply(&t1, &left.Y, &right.Y)
	field_multiply(&t2, &left.Z, &right.Z)
	field_add(&t3, &left.X, &left.Y)
	field_add(&t4, &right.X, &right.Y)
	field_multiply(&t3, &t3, &t4)
	field_add(&t4, &t0, &t1)
	field_subtract(&t3, &t3, &t4)
	field_add(&t4, &left.Y, &left.Z)
	field_add(&x3, &right.Y, &right.Z)
	field_multiply(&t4, &t4, &x3)
	field_add(&x3, &t1, &t2)
	field_subtract(&t4, &t4, &x3)
	field_add(&x3, &left.X, &left.Z)
	field_add(&y3, &right.X, &right.Z)
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
	destination.X, destination.Y, destination.Z = x3, y3, z3
}

// Complete doubling formula keeps zero and all finite intermediates inside one path.
func point_double(destination Point_Handle, source Point_Handle) {
	Point_Handle_Invariants(destination, "point_double.destination")
	Point_Handle_Invariants(source, "point_double.source")
	var point Point
	point_canonicalize_identity(&point, source)
	source = &point
	var t0, t1, t2, t3 [FIELD_LIMB_COUNT]uint64
	var x3, y3, z3 [FIELD_LIMB_COUNT]uint64
	b := field_b()
	field_square(&t0, &source.X)
	field_square(&t1, &source.Y)
	field_square(&t2, &source.Z)
	field_multiply(&t3, &source.X, &source.Y)
	field_add(&t3, &t3, &t3)
	field_multiply(&z3, &source.X, &source.Z)
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
	field_multiply(&t0, &source.Y, &source.Z)
	field_add(&t0, &t0, &t0)
	field_multiply(&z3, &t0, &z3)
	field_subtract(&x3, &x3, &z3)
	field_multiply(&z3, &t0, &t1)
	field_add(&z3, &z3, &z3)
	field_add(&z3, &z3, &z3)
	destination.X, destination.Y, destination.Z = x3, y3, z3
}

func point_scalar_multiply(
	destination Point_Handle,
	point Point_Handle,
	scalar *[FIELD_LIMB_COUNT]uint64,
) {
	Point_Handle_Invariants(destination, "point_scalar_multiply.destination")
	Point_Handle_Invariants(point, "point_scalar_multiply.point")
	var base Point
	point_canonicalize_identity(&base, point)
	result := Point{Y: field_one()}
	for bit_count := P256_BIT_COUNT; bit_count > bits.BIT_COUNT_MINIMUM; bit_count-- {
		bit_index := bit_count - binary.UINT_8_SIZE
		var doubled, added Point
		point_double(&doubled, &result)
		point_add(&added, &doubled, &base)
		word_index := bit_index / bits.BIT_COUNT_64_MAXIMUM
		word_shift := uint(bit_index % bits.BIT_COUNT_64_MAXIMUM)
		bit := scalar[word_index] >> word_shift & binary.UINT_8_SIZE
		point_select(
			&result, &added, &doubled, [CONDITION_LIMB_COUNT]uint64{bit},
		)
	}
	*destination = result
}

func point_canonicalize_identity(destination Point_Handle, source Point_Handle) {
	Point_Handle_Invariants(destination, "point_canonicalize_identity.destination")
	Point_Handle_Invariants(source, "point_canonicalize_identity.source")
	identity := Point{Y: field_one()}
	z_zero := field_is_zero(&source.Z)
	point_select(destination, &identity, source, z_zero)
}

func point_select(
	destination Point_Handle,
	first Point_Handle,
	second Point_Handle,
	condition [CONDITION_LIMB_COUNT]uint64,
) {
	Point_Handle_Invariants(destination, "point_select.destination")
	Point_Handle_Invariants(first, "point_select.first")
	Point_Handle_Invariants(second, "point_select.second")
	field_select(&destination.X, &first.X, &second.X, condition)
	field_select(&destination.Y, &first.Y, &second.Y, condition)
	field_select(&destination.Z, &first.Z, &second.Z, condition)
}

func point_affine_on_curve(
	x *[FIELD_LIMB_COUNT]uint64, y *[FIELD_LIMB_COUNT]uint64,
) (valid [CONDITION_LIMB_COUNT]uint64) {
	var left, right [FIELD_LIMB_COUNT]uint64
	field_square(&left, y)
	field_polynomial(&right, x)
	return field_equal(&left, &right)
}

func point_projective_on_curve(
	x *[FIELD_LIMB_COUNT]uint64,
	y *[FIELD_LIMB_COUNT]uint64,
	z *[FIELD_LIMB_COUNT]uint64,
) (valid [CONDITION_LIMB_COUNT]uint64) {
	var left, x_square, x_cube, z_square, z_cube [FIELD_LIMB_COUNT]uint64
	var x_z_square, twice, thrice, b_z_cube, right [FIELD_LIMB_COUNT]uint64
	field_square(&left, y)
	field_multiply(&left, &left, z)
	field_square(&x_square, x)
	field_multiply(&x_cube, &x_square, x)
	field_square(&z_square, z)
	field_multiply(&z_cube, &z_square, z)
	field_multiply(&x_z_square, x, &z_square)
	field_add(&twice, &x_z_square, &x_z_square)
	field_add(&thrice, &twice, &x_z_square)
	b := field_b()
	field_multiply(&b_z_cube, &b, &z_cube)
	field_subtract(&right, &x_cube, &thrice)
	field_add(&right, &right, &b_z_cube)
	return field_equal(&left, &right)
}

func field_polynomial(
	destination *[FIELD_LIMB_COUNT]uint64, x *[FIELD_LIMB_COUNT]uint64,
) {
	var square, cube, twice, thrice [FIELD_LIMB_COUNT]uint64
	field_square(&square, x)
	field_multiply(&cube, &square, x)
	field_add(&twice, x, x)
	field_add(&thrice, &twice, x)
	field_subtract(destination, &cube, &thrice)
	b := field_b()
	field_add(destination, destination, &b)
}

// Bit-serial multiplication trades throughput for small auditable constant-time storage.
func field_multiply(
	destination *[FIELD_LIMB_COUNT]uint64,
	left *[FIELD_LIMB_COUNT]uint64,
	right *[FIELD_LIMB_COUNT]uint64,
) {
	var result [FIELD_LIMB_COUNT]uint64
	addend := *left
	multiplier := *right
	for bit_index := bits.BIT_COUNT_MINIMUM; bit_index < P256_BIT_COUNT; bit_index++ {
		var candidate [FIELD_LIMB_COUNT]uint64
		field_add(&candidate, &result, &addend)
		word_index := bit_index / bits.BIT_COUNT_64_MAXIMUM
		word_shift := uint(bit_index % bits.BIT_COUNT_64_MAXIMUM)
		bit := multiplier[word_index] >> word_shift & binary.UINT_8_SIZE
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
	carry := limbs_add(&sum, left, right)
	modulus := field_modulus()
	borrow := limbs_subtract(&reduced, &sum, &modulus)
	reduce := carry[bits.BIT_COUNT_MINIMUM] |
		(borrow[bits.BIT_COUNT_MINIMUM] ^ binary.UINT_8_SIZE)
	limbs_select(
		destination, &reduced, &sum,
		[CONDITION_LIMB_COUNT]uint64{reduce},
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

func field_select(
	destination *[FIELD_LIMB_COUNT]uint64,
	first *[FIELD_LIMB_COUNT]uint64,
	second *[FIELD_LIMB_COUNT]uint64,
	condition [CONDITION_LIMB_COUNT]uint64,
) {
	mask := uint64(bits.WORD_64_MINIMUM) - condition[bits.BIT_COUNT_MINIMUM]
	var selected [FIELD_LIMB_COUNT]uint64
	for index := range selected {
		selected[index] = second[index] ^ mask&(first[index]^second[index])
	}
	*destination = selected
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

func field_canonical(
	value *[FIELD_LIMB_COUNT]uint64,
) (canonical [CONDITION_LIMB_COUNT]uint64) {
	var difference [FIELD_LIMB_COUNT]uint64
	modulus := field_modulus()
	return limbs_subtract(&difference, value, &modulus)
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
	factor := *base
	for bit_count := P256_BIT_COUNT; bit_count > bits.BIT_COUNT_MINIMUM; bit_count-- {
		bit_index := bit_count - binary.UINT_8_SIZE
		var square, product [FIELD_LIMB_COUNT]uint64
		field_square(&square, &result)
		field_multiply(&product, &square, &factor)
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

func field_decode(
	destination *[FIELD_LIMB_COUNT]uint64,
	source *[FIELD_ELEMENT_SIZE]byte,
) (valid [CONDITION_LIMB_COUNT]uint64) {
	for index := bits.BIT_COUNT_MINIMUM; index < FIELD_LIMB_COUNT; index++ {
		source_index := FIELD_ELEMENT_SIZE -
			(index+binary.UINT_8_SIZE)*binary.UINT_64_SIZE
		destination[index] = uint64(binary.Uint_64(
			source[source_index:source_index+binary.UINT_64_SIZE], binary.BIG_ENDIAN,
		))
	}
	return field_canonical(destination)
}

func field_encode(
	destination *[FIELD_ELEMENT_SIZE]byte, source *[FIELD_LIMB_COUNT]uint64,
) {
	for index := bits.BIT_COUNT_MINIMUM; index < FIELD_LIMB_COUNT; index++ {
		destination_index := FIELD_ELEMENT_SIZE -
			(index+binary.UINT_8_SIZE)*binary.UINT_64_SIZE
		binary.Put_Uint_64(
			destination[destination_index:destination_index+binary.UINT_64_SIZE],
			binary.Word_64(source[index]), binary.BIG_ENDIAN,
		)
	}
}

func scalar_decode(
	destination *[FIELD_LIMB_COUNT]uint64, source *[SCALAR_SIZE]byte,
) {
	for index := bits.BIT_COUNT_MINIMUM; index < FIELD_LIMB_COUNT; index++ {
		source_index := SCALAR_SIZE -
			(index+binary.UINT_8_SIZE)*binary.UINT_64_SIZE
		destination[index] = uint64(binary.Uint_64(
			source[source_index:source_index+binary.UINT_64_SIZE], binary.BIG_ENDIAN,
		))
	}
	order := scalar_order()
	var reduced [FIELD_LIMB_COUNT]uint64
	borrow := limbs_subtract(&reduced, destination, &order)
	limbs_select(
		destination, &reduced, destination,
		[CONDITION_LIMB_COUNT]uint64{
			borrow[bits.BIT_COUNT_MINIMUM] ^ binary.UINT_8_SIZE,
		},
	)
}

func limbs_add(
	destination *[FIELD_LIMB_COUNT]uint64,
	left *[FIELD_LIMB_COUNT]uint64,
	right *[FIELD_LIMB_COUNT]uint64,
) (carry [CONDITION_LIMB_COUNT]uint64) {
	carry_value := uint64(bits.WORD_64_MINIMUM)
	var sum [FIELD_LIMB_COUNT]uint64
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

func limbs_subtract(
	destination *[FIELD_LIMB_COUNT]uint64,
	left *[FIELD_LIMB_COUNT]uint64,
	right *[FIELD_LIMB_COUNT]uint64,
) (borrow [CONDITION_LIMB_COUNT]uint64) {
	borrow_value := uint64(bits.WORD_64_MINIMUM)
	var difference [FIELD_LIMB_COUNT]uint64
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

func limbs_select(
	destination *[FIELD_LIMB_COUNT]uint64,
	first *[FIELD_LIMB_COUNT]uint64,
	second *[FIELD_LIMB_COUNT]uint64,
	condition [CONDITION_LIMB_COUNT]uint64,
) {
	mask := uint64(bits.WORD_64_MINIMUM) - condition[bits.BIT_COUNT_MINIMUM]
	var selected [FIELD_LIMB_COUNT]uint64
	for index := range selected {
		selected[index] = second[index] ^ mask&(first[index]^second[index])
	}
	*destination = selected
}

// These limbs are the canonical NIST P-256 prime in little-endian machine order.
func field_modulus() (value [FIELD_LIMB_COUNT]uint64) {
	return [FIELD_LIMB_COUNT]uint64{
		0xffffffffffffffff,
		0x00000000ffffffff,
		0x0000000000000000,
		0xffffffff00000001,
	}
}

// P-256's curve coefficient is the SEC 2 value in little-endian machine order.
func field_b() (value [FIELD_LIMB_COUNT]uint64) {
	return [FIELD_LIMB_COUNT]uint64{
		0x3bce3c3e27d2604b,
		0x651d06b0cc53b0f6,
		0xb3ebbd55769886bc,
		0x5ac635d8aa3a93e7,
	}
}

func field_one() (value [FIELD_LIMB_COUNT]uint64) {
	value[bits.BIT_COUNT_MINIMUM] = binary.UINT_8_SIZE
	return value
}

// Generator coordinates are the canonical SEC 2 P-256 base point.
func field_generator_x() (value [FIELD_LIMB_COUNT]uint64) {
	return [FIELD_LIMB_COUNT]uint64{
		0xf4a13945d898c296,
		0x77037d812deb33a0,
		0xf8bce6e563a440f2,
		0x6b17d1f2e12c4247,
	}
}

func field_generator_y() (value [FIELD_LIMB_COUNT]uint64) {
	return [FIELD_LIMB_COUNT]uint64{
		0xcbb6406837bf51f5,
		0x2bce33576b315ece,
		0x8ee7eb4a7c0f9e16,
		0x4fe342e2fe1a7f9b,
	}
}

// Fermat inversion uses p minus two, derived from field_modulus.
func field_inverse_exponent() (value [FIELD_LIMB_COUNT]uint64) {
	return [FIELD_LIMB_COUNT]uint64{
		0xfffffffffffffffd,
		0x00000000ffffffff,
		0x0000000000000000,
		0xffffffff00000001,
	}
}

// P-256 is three modulo four, so square roots use the exponent (p plus one) divided by four.
func field_square_root_exponent() (value [FIELD_LIMB_COUNT]uint64) {
	return [FIELD_LIMB_COUNT]uint64{
		0x0000000000000000,
		0x0000000040000000,
		0x4000000000000000,
		0x3fffffffc0000000,
	}
}

// The group order is the canonical SEC 2 P-256 order in little-endian machine order.
func scalar_order() (value [FIELD_LIMB_COUNT]uint64) {
	return [FIELD_LIMB_COUNT]uint64{
		0xf3b9cac2fc632551,
		0xbce6faada7179e84,
		0xffffffffffffffff,
		0xffffffff00000000,
	}
}
