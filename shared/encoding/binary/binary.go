// Package binary translates fixed-width numbers and varints to and from caller-owned bytes.
package binary

import (
	"errors"
	"reflect"
	"unsafe"

	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/sim/nbio"
)

// BYTE_SIZE_MINIMUM is empty byte storage size.
const BYTE_SIZE_MINIMUM = 0

// BYTE_SIZE_MAXIMUM matches repository-wide bounded slice domain.
const BYTE_SIZE_MAXIMUM = 4096

// NONEMPTY_BYTE_SIZE_MINIMUM is first append-result size.
const NONEMPTY_BYTE_SIZE_MINIMUM = 1

// BITS_PER_BYTE keeps width conversion tied to binary byte definition.
const BITS_PER_BYTE = 8

// UINT_8_SIZE derives encoded width from shared primitive width.
const UINT_8_SIZE = bits.BIT_COUNT_8_MAXIMUM / BITS_PER_BYTE

// UINT_16_SIZE derives encoded width from shared primitive width.
const UINT_16_SIZE = bits.BIT_COUNT_16_MAXIMUM / BITS_PER_BYTE

// UINT_32_SIZE derives encoded width from shared primitive width.
const UINT_32_SIZE = bits.BIT_COUNT_32_MAXIMUM / BITS_PER_BYTE

// UINT_64_SIZE derives encoded width from shared primitive width.
const UINT_64_SIZE = bits.BIT_COUNT_64_MAXIMUM / BITS_PER_BYTE

// VALUE_NESTING_MAXIMUM bounds reflected type recursion before traversal.
const VALUE_NESTING_MAXIMUM = 64

// VALUE_STACK_SIZE includes root above maximum nested descendants.
const VALUE_STACK_SIZE = VALUE_NESTING_MAXIMUM + 1

// ACTIVE_STACK_COUNT_MINIMUM keeps one root frame while traversal runs.
const ACTIVE_STACK_COUNT_MINIMUM = 1

// ACTIVE_STACK_COUNT_MAXIMUM is deepest record frame that can push one leaf.
const ACTIVE_STACK_COUNT_MAXIMUM = VALUE_NESTING_MAXIMUM

// VALUE_ELEMENT_COUNT_MAXIMUM applies repository collection boundary to arrays and slices.
const VALUE_ELEMENT_COUNT_MAXIMUM = BYTE_SIZE_MAXIMUM

// VALUE_SIZE_INVALID reports unsupported structured value.
const VALUE_SIZE_INVALID = -1

// VALUE_SIZE_MINIMUM includes unsupported structured value sentinel.
const VALUE_SIZE_MINIMUM = VALUE_SIZE_INVALID

// VALUE_SIZE_MAXIMUM matches byte storage boundary because encoded values occupy bytes.
const VALUE_SIZE_MAXIMUM = BYTE_SIZE_MAXIMUM

// BYTE_ORDER_LITTLE_VALUE gives little-endian order distinct identity.
const BYTE_ORDER_LITTLE_VALUE uint8 = 0

// BYTE_ORDER_BIG_VALUE gives big-endian order distinct identity.
const BYTE_ORDER_BIG_VALUE uint8 = 1

// BYTE_ORDER_NATIVE_VALUE preserves native order identity even when machine order matches another.
const BYTE_ORDER_NATIVE_VALUE uint8 = 2

// LITTLE_ENDIAN places least-significant byte first.
const LITTLE_ENDIAN Byte_Order = Byte_Order(BYTE_ORDER_LITTLE_VALUE)

// BIG_ENDIAN places most-significant byte first.
const BIG_ENDIAN Byte_Order = Byte_Order(BYTE_ORDER_BIG_VALUE)

// NATIVE_ENDIAN selects target machine byte order.
const NATIVE_ENDIAN Byte_Order = Byte_Order(BYTE_ORDER_NATIVE_VALUE)

// VARINT_SIZE_16_MAXIMUM is maximum encoded size of 16-bit varint.
const VARINT_SIZE_16_MAXIMUM = 3

// VARINT_SIZE_32_MAXIMUM is maximum encoded size of 32-bit varint.
const VARINT_SIZE_32_MAXIMUM = 5

// VARINT_SIZE_64_MAXIMUM is maximum encoded size of 64-bit varint.
const VARINT_SIZE_64_MAXIMUM = 10

// VARINT_SIZE_MINIMUM is shortest complete varint.
const VARINT_SIZE_MINIMUM = 1

// VARINT_COUNT_MINIMUM reports overflow after eleventh byte.
const VARINT_COUNT_MINIMUM = -(VARINT_SIZE_64_MAXIMUM + 1)

// VARINT_COUNT_MAXIMUM is longest valid varint size.
const VARINT_COUNT_MAXIMUM = VARINT_SIZE_64_MAXIMUM

// VARINT_COUNT_GAP_NEGATIVE_NINE starts impossible overflow-count gap.
const VARINT_COUNT_GAP_NEGATIVE_NINE = -9

// VARINT_COUNT_GAP_NEGATIVE_EIGHT is impossible before 64-bit overflow.
const VARINT_COUNT_GAP_NEGATIVE_EIGHT = -8

// VARINT_COUNT_GAP_NEGATIVE_SEVEN is impossible before 64-bit overflow.
const VARINT_COUNT_GAP_NEGATIVE_SEVEN = -7

// VARINT_COUNT_GAP_NEGATIVE_SIX is impossible before 64-bit overflow.
const VARINT_COUNT_GAP_NEGATIVE_SIX = -6

// VARINT_COUNT_GAP_NEGATIVE_FIVE is impossible before 64-bit overflow.
const VARINT_COUNT_GAP_NEGATIVE_FIVE = -5

// VARINT_COUNT_GAP_NEGATIVE_FOUR is impossible before 64-bit overflow.
const VARINT_COUNT_GAP_NEGATIVE_FOUR = -4

// VARINT_COUNT_GAP_NEGATIVE_THREE is impossible before 64-bit overflow.
const VARINT_COUNT_GAP_NEGATIVE_THREE = -3

// VARINT_COUNT_GAP_NEGATIVE_TWO is impossible before 64-bit overflow.
const VARINT_COUNT_GAP_NEGATIVE_TWO = -2

// VARINT_COUNT_GAP_NEGATIVE_ONE ends impossible overflow-count gap.
const VARINT_COUNT_GAP_NEGATIVE_ONE = -1

// BYTE_ORDER_NAME_SIZE_MINIMUM is BigEndian length.
const BYTE_ORDER_NAME_SIZE_MINIMUM = 9

// BYTE_ORDER_NAME_SIZE_MAXIMUM is LittleEndian and NativeEndian length.
const BYTE_ORDER_NAME_SIZE_MAXIMUM = 12

// BYTE_ORDER_GO_NAME_SIZE_MINIMUM is binary.BigEndian length.
const BYTE_ORDER_GO_NAME_SIZE_MINIMUM = 16

// BYTE_ORDER_GO_NAME_SIZE_MAXIMUM is long Go-syntax byte-order name length.
const BYTE_ORDER_GO_NAME_SIZE_MAXIMUM = 19

// INITIAL_DEPTH_ROOT starts traversal at top-level value.
const INITIAL_DEPTH_ROOT = 0

// INITIAL_DEPTH_SLICE_ELEMENT accounts for top-level slice frame.
const INITIAL_DEPTH_SLICE_ELEMENT = 1

// Error_Buffer_Too_Small reports structured storage shorter than encoded value.
var Error_Buffer_Too_Small = errors.New("binary: buffer too small")

// Error_Invalid_Type reports value outside fixed-size type grammar.
var Error_Invalid_Type = errors.New("binary: invalid type")

// Error_Varint_Overflow reports integer wider than 64 bits.
var Error_Varint_Overflow = errors.New("binary: varint overflows a 64-bit integer")

// Byte_Order selects byte significance order.
type Byte_Order uint8

// Byte_Order_Invariants covers two explicit orders and native-order identity.
func Byte_Order_Invariants(value Byte_Order, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), BYTE_ORDER_LITTLE_VALUE,
			BYTE_ORDER_BIG_VALUE, BYTE_ORDER_NATIVE_VALUE,
		).
		Ensure()
}

// Byte_Order_Name is standard display name.
type Byte_Order_Name string

// Byte_Order_Name_Invariants admits short and long standard names.
func Byte_Order_Name_Invariants(value Byte_Order_Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Int(
			len(value), BYTE_ORDER_NAME_SIZE_MINIMUM, BYTE_ORDER_NAME_SIZE_MAXIMUM,
		).
		Ensure()
}

// Byte_Order_Go_Name is standard Go-syntax display name.
type Byte_Order_Go_Name string

// Byte_Order_Go_Name_Invariants admits short and long Go-syntax names.
func Byte_Order_Go_Name_Invariants(
	value Byte_Order_Go_Name, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Int(
			len(value), BYTE_ORDER_GO_NAME_SIZE_MINIMUM,
			BYTE_ORDER_GO_NAME_SIZE_MAXIMUM,
		).
		Ensure()
}

// Boolean gives binary decisions one typed coverage identity.
type Boolean bool

// Boolean_Invariants requires both decision states.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A binary decision is true.").
		Ensure()
}

// Word_16 is complete unsigned 16-bit value domain.
type Word_16 uint16

// Word_16_Invariants covers every 16-bit word.
func Word_16_Invariants(value Word_16, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), bits.WORD_16_MINIMUM, bits.WORD_16_MAXIMUM).
		Ensure()
}

// Word_32 is complete unsigned 32-bit value domain.
type Word_32 uint32

// Word_32_Invariants covers every 32-bit word.
func Word_32_Invariants(value Word_32, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, bits.WORD_32_MAXIMUM).
		Ensure()
}

// Word_64 is complete unsigned 64-bit value domain.
type Word_64 uint64

// Word_64_Invariants covers every 64-bit word.
func Word_64_Invariants(value Word_64, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Integer_64 is complete signed 64-bit value domain.
type Integer_64 int64

// Integer_64_Invariants covers every signed 64-bit value.
func Integer_64_Invariants(value Integer_64, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Initial_Depth distinguishes top-level values from top-level slice elements.
type Initial_Depth int

// Initial_Depth_Invariants admits only two traversal entry depths.
func Initial_Depth_Invariants(value Initial_Depth, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Int(int(value), INITIAL_DEPTH_ROOT, INITIAL_DEPTH_SLICE_ELEMENT).
		Ensure()
}

// Active_Stack_Count is one nonempty fixed traversal stack size.
type Active_Stack_Count int

// Active_Stack_Count_Invariants keeps frame access inside stack storage.
func Active_Stack_Count_Invariants(
	value Active_Stack_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), ACTIVE_STACK_COUNT_MINIMUM, ACTIVE_STACK_COUNT_MAXIMUM,
		).
		Ensure()
}

// Stack_Count includes empty traversal after final frame retires.
type Stack_Count int

// Stack_Count_Invariants keeps loop state inside fixed stack storage.
func Stack_Count_Invariants(value Stack_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), BYTE_SIZE_MINIMUM, VALUE_STACK_SIZE).
		Ensure()
}

// Element_Count is one bounded reflected collection size.
type Element_Count int

// Element_Count_Invariants applies package collection boundary.
func Element_Count_Invariants(value Element_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), BYTE_SIZE_MINIMUM, VALUE_ELEMENT_COUNT_MAXIMUM).
		Ensure()
}

// Fixed_Size is one primitive wire width.
type Fixed_Size int

// Fixed_Size_Invariants admits supported primitive widths.
func Fixed_Size_Invariants(value Fixed_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Int(int(value), UINT_8_SIZE, UINT_16_SIZE, UINT_32_SIZE, UINT_64_SIZE).
		Ensure()
}

// Bytes is bounded caller-owned byte storage.
type Bytes []byte

// Bytes_Invariants applies shared binary byte boundary.
func Bytes_Invariants(value Bytes, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), BYTE_SIZE_MINIMUM, BYTE_SIZE_MAXIMUM).
		Ensure()
}

// Nonempty_Bytes is append result containing newly encoded bytes.
type Nonempty_Bytes []byte

// Nonempty_Bytes_Invariants excludes empty append result.
func Nonempty_Bytes_Invariants(value Nonempty_Bytes, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), NONEMPTY_BYTE_SIZE_MINIMUM, BYTE_SIZE_MAXIMUM).
		Ensure()
}

// Uint_16_Bytes is one buffer after a two-byte append.
type Uint_16_Bytes []byte

// Uint_16_Bytes_Invariants excludes lengths that cannot contain the appended value.
func Uint_16_Bytes_Invariants(value Uint_16_Bytes, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), UINT_16_SIZE, BYTE_SIZE_MAXIMUM).
		Ensure()
}

// Uint_32_Bytes is one buffer after a four-byte append.
type Uint_32_Bytes []byte

// Uint_32_Bytes_Invariants excludes lengths that cannot contain the appended value.
func Uint_32_Bytes_Invariants(value Uint_32_Bytes, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), UINT_32_SIZE, BYTE_SIZE_MAXIMUM).
		Ensure()
}

// Uint_64_Bytes is one buffer after an eight-byte append.
type Uint_64_Bytes []byte

// Uint_64_Bytes_Invariants excludes lengths that cannot contain the appended value.
func Uint_64_Bytes_Invariants(value Uint_64_Bytes, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), UINT_64_SIZE, BYTE_SIZE_MAXIMUM).
		Ensure()
}

// Byte_Count is successful encoded byte count.
type Byte_Count int

// Byte_Count_Invariants bounds one count to caller storage domain.
func Byte_Count_Invariants(value Byte_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), BYTE_SIZE_MINIMUM, BYTE_SIZE_MAXIMUM).
		Ensure()
}

// Varint_Size is one successful varint encoding size.
type Varint_Size int

// Varint_Size_Invariants matches the complete 64-bit varint width domain.
func Varint_Size_Invariants(value Varint_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), VARINT_SIZE_MINIMUM, VARINT_SIZE_64_MAXIMUM).
		Ensure()
}

// Value_Size is encoded size or VALUE_SIZE_INVALID.
type Value_Size int

// Value_Size_Invariants includes unsupported-value sentinel and bounded valid sizes.
func Value_Size_Invariants(value Value_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), VALUE_SIZE_MINIMUM, VALUE_SIZE_MAXIMUM).
		Ensure()
}

// Varint_Count is consumed byte count, zero for incomplete input, or negative for overflow.
type Varint_Count int

// Varint_Count_Invariants covers incomplete, valid, and overflow results.
func Varint_Count_Invariants(value Varint_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int(
			int(value), VARINT_COUNT_MINIMUM, VARINT_COUNT_MAXIMUM,
			VARINT_COUNT_GAP_NEGATIVE_NINE, VARINT_COUNT_GAP_NEGATIVE_EIGHT,
			VARINT_COUNT_GAP_NEGATIVE_SEVEN, VARINT_COUNT_GAP_NEGATIVE_ONE,
		).
		Range_Holed_Int(
			int(value), VARINT_COUNT_MINIMUM, VARINT_COUNT_MAXIMUM,
			VARINT_COUNT_GAP_NEGATIVE_SIX, VARINT_COUNT_GAP_NEGATIVE_FIVE,
			VARINT_COUNT_GAP_NEGATIVE_FOUR, VARINT_COUNT_GAP_NEGATIVE_ONE,
		).
		Range_Holed_Int(
			int(value), VARINT_COUNT_MINIMUM, VARINT_COUNT_MAXIMUM,
			VARINT_COUNT_GAP_NEGATIVE_THREE, VARINT_COUNT_GAP_NEGATIVE_TWO,
			VARINT_COUNT_GAP_NEGATIVE_ONE, VARINT_COUNT_GAP_NEGATIVE_ONE,
		).
		Ensure()
}

// Operation_Active prevents one callback slot from serving overlapping operations.
type Operation_Active bool

// Operation_Active_Invariants covers idle and borrowed callback states.
func Operation_Active_Invariants(
	value Operation_Active, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Binary Stream operation is active.").
		Ensure()
}

// Stream_Initialized separates zero state from injected transport and scratch.
type Stream_Initialized bool

// Stream_Initialized_Invariants covers unbound and initialized state.
func Stream_Initialized_Invariants(
	value Stream_Initialized, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Binary Stream state is initialized.").
		Ensure()
}

// Submission_Active marks one Stream Procedure frame on stack.
type Submission_Active bool

// Submission_Active_Invariants covers inline and deferred Stream retirement.
func Submission_Active_Invariants(
	value Submission_Active, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Binary Stream submission frame is active.").
		Ensure()
}

// Wait_Active marks one Reader transfer not yet retired.
type Wait_Active bool

// Wait_Active_Invariants covers idle and in-flight Reader transfers.
func Wait_Active_Invariants(value Wait_Active, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Binary Reader waits for Stream retirement.").
		Ensure()
}

// Submission_Continue turns inline callback recursion into iteration.
type Submission_Continue bool

// Submission_Continue_Invariants covers both trampoline decisions.
func Submission_Continue_Invariants(
	value Submission_Continue, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Binary Reader has inline retirement to process.").
		Ensure()
}

// Stream_Size is one valid structured width retained across Stream completion.
type Stream_Size int

// Stream_Size_Invariants bounds valid structured width without invalid-type sentinel.
func Stream_Size_Invariants(value Stream_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), BYTE_SIZE_MINIMUM, BYTE_SIZE_MAXIMUM).
		Ensure()
}

// Value carries arbitrary reflected binary input or destination.
type Value interface{}

// Error carries binary or injected stream failure.
type Error interface {
	error
}

// Reader retains caller scratch and one structured read continuation.
type Reader struct {
	// Completion stays first so static callback recovers Reader without allocating closure.
	Completion nbio.Completion
	// Stream owns transport and callback timing.
	Stream nbio.Stream
	// Callback retires after complete structured value, not each transfer.
	Callback nbio.Callback
	// Scratch remains caller-owned across every partial transfer.
	Scratch Bytes
	// Destination remains borrowed until decode or transport failure.
	Destination Value
	// Order stays stable across deferred completion.
	Order Byte_Order
	// Size is complete encoded width required before decode.
	Size Stream_Size
	// Count is source bytes retired across partial reads.
	Count Byte_Count
	// Active protects retained destination and callback.
	Active Operation_Active
	// Initialized rejects use before dependency binding.
	Initialized Stream_Initialized
	// Submission_Active marks Stream Procedure stack lifetime.
	Submission_Active Submission_Active
	// Wait_Active marks one submitted read.
	Wait_Active Wait_Active
	// Continue requests next inline trampoline iteration.
	Continue Submission_Continue
}

// Reader_Invariants keeps transfer cursor inside caller scratch.
func Reader_Invariants(value Reader, namespace aver.Namespace) {
	aver.Always(
		unsafe.Pointer(&value) == unsafe.Pointer(&value.Completion),
		"Binary Reader completion stays first for static callback recovery.",
	)
	Bytes_Invariants(value.Scratch, namespace)
	Byte_Order_Invariants(value.Order, namespace)
	Stream_Size_Invariants(value.Size, namespace)
	Byte_Count_Invariants(value.Count, namespace)
	Operation_Active_Invariants(value.Active, namespace)
	Stream_Initialized_Invariants(value.Initialized, namespace)
	Submission_Active_Invariants(value.Submission_Active, namespace)
	Wait_Active_Invariants(value.Wait_Active, namespace)
	Submission_Continue_Invariants(value.Continue, namespace)
	aver.Always(
		int(value.Count) <= int(value.Size),
		"Binary Reader cursor does not cross structured width.",
	)
	aver.Always(
		int(value.Size) <= len(value.Scratch),
		"Binary Reader operation stays inside caller scratch.",
	)
}

// Reader_Handle keeps Reader identity across deferred Stream retirement.
type Reader_Handle *Reader

// Reader_Handle_Invariants keeps dereferenced state visible at the dependency boundary.
func Reader_Handle_Invariants(value Reader_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Reader_Invariants(*value, namespace)
}

// Writer retains caller scratch and one structured write continuation.
type Writer struct {
	// Completion stays first so static callback recovers Writer without allocating closure.
	Completion nbio.Completion
	// Stream owns transport and callback timing.
	Stream nbio.Stream
	// Callback retires after encoded bytes leave caller scratch.
	Callback nbio.Callback
	// Scratch remains caller-owned until Stream retirement.
	Scratch Bytes
	// Order records encoded byte layout during active operation.
	Order Byte_Order
	// Size is exact encoded prefix borrowed by Stream.
	Size Stream_Size
	// Active protects retained scratch and callback.
	Active Operation_Active
	// Initialized rejects use before dependency binding.
	Initialized Stream_Initialized
}

// Writer_Invariants keeps encoded prefix inside caller scratch.
func Writer_Invariants(value Writer, namespace aver.Namespace) {
	aver.Always(
		unsafe.Pointer(&value) == unsafe.Pointer(&value.Completion),
		"Binary Writer completion stays first for static callback recovery.",
	)
	Bytes_Invariants(value.Scratch, namespace)
	Byte_Order_Invariants(value.Order, namespace)
	Stream_Size_Invariants(value.Size, namespace)
	Operation_Active_Invariants(value.Active, namespace)
	Stream_Initialized_Invariants(value.Initialized, namespace)
	aver.Always(
		int(value.Size) <= len(value.Scratch),
		"Binary Writer operation stays inside caller scratch.",
	)
}

// Writer_Handle keeps Writer identity across deferred Stream retirement.
type Writer_Handle *Writer

// Writer_Handle_Invariants keeps dereferenced state visible at the dependency boundary.
func Writer_Handle_Invariants(value Writer_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Writer_Invariants(*value, namespace)
}

// Byte_Order_String gives standard display name without forbidden method API.
func Byte_Order_String(order Byte_Order) (name Byte_Order_Name) {
	defer func() { Byte_Order_Name_Invariants(name, "byte_order_string.name") }()
	Byte_Order_Invariants(order, "byte_order_string.order")
	switch order {
	case LITTLE_ENDIAN:
		return "LittleEndian"
	case BIG_ENDIAN:
		return "BigEndian"
	}
	return "NativeEndian"
}

// Byte_Order_Go_String gives standard Go-syntax name without forbidden method API.
func Byte_Order_Go_String(order Byte_Order) (name Byte_Order_Go_Name) {
	defer func() { Byte_Order_Go_Name_Invariants(name, "byte_order_go_string.name") }()
	Byte_Order_Invariants(order, "byte_order_go_string.order")
	switch order {
	case LITTLE_ENDIAN:
		return "binary.LittleEndian"
	case BIG_ENDIAN:
		return "binary.BigEndian"
	}
	return "binary.NativeEndian"
}

// Resolves native identity once per operation while keeping public enum stable.
func little_order(order Byte_Order) (little Boolean) {
	defer func() { Boolean_Invariants(little, "little_order.little") }()
	Byte_Order_Invariants(order, "little_order.order")
	if order == NATIVE_ENDIAN {
		return Boolean(NATIVE_BYTE_ORDER == LITTLE_ENDIAN)
	}
	return Boolean(order == LITTLE_ENDIAN)
}

// Uint_16 decodes first two source bytes.
func Uint_16(source Bytes, order Byte_Order) (value Word_16) {
	defer func() { Word_16_Invariants(value, "uint_16.value") }()
	Bytes_Invariants(source, "uint_16.source")
	Byte_Order_Invariants(order, "uint_16.order")
	little := little_order(order)
	return uint_16_raw(source, little)
}

// Put_Uint_16 encodes value into first two destination bytes.
func Put_Uint_16(destination Bytes, value Word_16, order Byte_Order) {
	Bytes_Invariants(destination, "put_uint_16.destination")
	Word_16_Invariants(value, "put_uint_16.value")
	Byte_Order_Invariants(order, "put_uint_16.order")
	little := little_order(order)
	put_uint_16_raw(destination, value, little)
}

// Keeps reflected traversal behind already-validated public boundary.
func uint_16_raw(source Bytes, little Boolean) (value Word_16) {
	defer func() { Word_16_Invariants(value, "uint_16_raw.value") }()
	Bytes_Invariants(source, "uint_16_raw.source")
	Boolean_Invariants(little, "uint_16_raw.little")
	aver.Always(len(source) >= UINT_16_SIZE, "Uint16 source is large enough.")
	source = source[:UINT_16_SIZE]
	if little {
		return Word_16(uint16(source[0]) | uint16(source[1])<<8)
	}
	return Word_16(uint16(source[1]) | uint16(source[0])<<8)
}

// Keeps reflected traversal behind already-validated public boundary.
func put_uint_16_raw(destination Bytes, value Word_16, little Boolean) {
	Bytes_Invariants(destination, "put_uint_16_raw.destination")
	Word_16_Invariants(value, "put_uint_16_raw.value")
	Boolean_Invariants(little, "put_uint_16_raw.little")
	aver.Always(len(destination) >= UINT_16_SIZE, "Uint16 destination is large enough.")
	destination = destination[:UINT_16_SIZE]
	if little {
		destination[0] = byte(value)
		destination[1] = byte(value >> 8)
		return
	}
	destination[0] = byte(value >> 8)
	destination[1] = byte(value)
}

// Append_Uint_16 extends buffer inside existing caller capacity.
func Append_Uint_16(
	buffer Bytes, value Word_16, order Byte_Order,
) (result Uint_16_Bytes) {
	defer func() { Uint_16_Bytes_Invariants(result, "append_uint_16.result") }()
	Bytes_Invariants(buffer, "append_uint_16.buffer")
	Word_16_Invariants(value, "append_uint_16.value")
	Byte_Order_Invariants(order, "append_uint_16.order")
	result = Uint_16_Bytes(extend(buffer, Varint_Size(UINT_16_SIZE)))
	Put_Uint_16(Bytes(result[len(buffer):]), value, order)
	return result
}

// Uint_32 decodes first four source bytes.
func Uint_32(source Bytes, order Byte_Order) (value Word_32) {
	defer func() { Word_32_Invariants(value, "uint_32.value") }()
	Bytes_Invariants(source, "uint_32.source")
	Byte_Order_Invariants(order, "uint_32.order")
	little := little_order(order)
	return uint_32_raw(source, little)
}

// Keeps reflected traversal behind already-validated public boundary.
func uint_32_raw(source Bytes, little Boolean) (value Word_32) {
	defer func() { Word_32_Invariants(value, "uint_32_raw.value") }()
	Bytes_Invariants(source, "uint_32_raw.source")
	Boolean_Invariants(little, "uint_32_raw.little")
	aver.Always(len(source) >= UINT_32_SIZE, "Uint32 source is large enough.")
	source = source[:UINT_32_SIZE]
	if little {
		return Word_32(uint32(source[0]) | uint32(source[1])<<8 |
			uint32(source[2])<<16 | uint32(source[3])<<24)
	}
	return Word_32(uint32(source[3]) | uint32(source[2])<<8 |
		uint32(source[1])<<16 | uint32(source[0])<<24)
}

// Put_Uint_32 encodes value into first four destination bytes.
func Put_Uint_32(destination Bytes, value Word_32, order Byte_Order) {
	Bytes_Invariants(destination, "put_uint_32.destination")
	Word_32_Invariants(value, "put_uint_32.value")
	Byte_Order_Invariants(order, "put_uint_32.order")
	little := little_order(order)
	put_uint_32_raw(destination, value, little)
}

// Keeps reflected traversal behind already-validated public boundary.
func put_uint_32_raw(destination Bytes, value Word_32, little Boolean) {
	Bytes_Invariants(destination, "put_uint_32_raw.destination")
	Word_32_Invariants(value, "put_uint_32_raw.value")
	Boolean_Invariants(little, "put_uint_32_raw.little")
	aver.Always(len(destination) >= UINT_32_SIZE, "Uint32 destination is large enough.")
	destination = destination[:UINT_32_SIZE]
	if little {
		destination[0] = byte(value)
		destination[1] = byte(value >> 8)
		destination[2] = byte(value >> 16)
		destination[3] = byte(value >> 24)
		return
	}
	destination[0] = byte(value >> 24)
	destination[1] = byte(value >> 16)
	destination[2] = byte(value >> 8)
	destination[3] = byte(value)
}

// Append_Uint_32 extends buffer inside existing caller capacity.
func Append_Uint_32(
	buffer Bytes, value Word_32, order Byte_Order,
) (result Uint_32_Bytes) {
	defer func() { Uint_32_Bytes_Invariants(result, "append_uint_32.result") }()
	Bytes_Invariants(buffer, "append_uint_32.buffer")
	Word_32_Invariants(value, "append_uint_32.value")
	Byte_Order_Invariants(order, "append_uint_32.order")
	result = Uint_32_Bytes(extend(buffer, Varint_Size(UINT_32_SIZE)))
	Put_Uint_32(Bytes(result[len(buffer):]), value, order)
	return result
}

// Uint_64 decodes first eight source bytes.
func Uint_64(source Bytes, order Byte_Order) (value Word_64) {
	defer func() { Word_64_Invariants(value, "uint_64.value") }()
	Bytes_Invariants(source, "uint_64.source")
	Byte_Order_Invariants(order, "uint_64.order")
	little := little_order(order)
	return uint_64_raw(source, little)
}

// Keeps reflected traversal behind already-validated public boundary.
func uint_64_raw(source Bytes, little Boolean) (value Word_64) {
	defer func() { Word_64_Invariants(value, "uint_64_raw.value") }()
	Bytes_Invariants(source, "uint_64_raw.source")
	Boolean_Invariants(little, "uint_64_raw.little")
	aver.Always(len(source) >= UINT_64_SIZE, "Uint64 source is large enough.")
	source = source[:UINT_64_SIZE]
	if little {
		return Word_64(uint64(source[0]) | uint64(source[1])<<8 |
			uint64(source[2])<<16 | uint64(source[3])<<24 |
			uint64(source[4])<<32 | uint64(source[5])<<40 |
			uint64(source[6])<<48 | uint64(source[7])<<56)
	}
	return Word_64(uint64(source[7]) | uint64(source[6])<<8 |
		uint64(source[5])<<16 | uint64(source[4])<<24 |
		uint64(source[3])<<32 | uint64(source[2])<<40 |
		uint64(source[1])<<48 | uint64(source[0])<<56)
}

// Put_Uint_64 encodes value into first eight destination bytes.
func Put_Uint_64(destination Bytes, value Word_64, order Byte_Order) {
	Bytes_Invariants(destination, "put_uint_64.destination")
	Word_64_Invariants(value, "put_uint_64.value")
	Byte_Order_Invariants(order, "put_uint_64.order")
	little := little_order(order)
	put_uint_64_raw(destination, value, little)
}

// Keeps reflected traversal behind already-validated public boundary.
func put_uint_64_raw(destination Bytes, value Word_64, little Boolean) {
	Bytes_Invariants(destination, "put_uint_64_raw.destination")
	Word_64_Invariants(value, "put_uint_64_raw.value")
	Boolean_Invariants(little, "put_uint_64_raw.little")
	aver.Always(len(destination) >= UINT_64_SIZE, "Uint64 destination is large enough.")
	destination = destination[:UINT_64_SIZE]
	if little {
		destination[0] = byte(value)
		destination[1] = byte(value >> 8)
		destination[2] = byte(value >> 16)
		destination[3] = byte(value >> 24)
		destination[4] = byte(value >> 32)
		destination[5] = byte(value >> 40)
		destination[6] = byte(value >> 48)
		destination[7] = byte(value >> 56)
		return
	}
	destination[0] = byte(value >> 56)
	destination[1] = byte(value >> 48)
	destination[2] = byte(value >> 40)
	destination[3] = byte(value >> 32)
	destination[4] = byte(value >> 24)
	destination[5] = byte(value >> 16)
	destination[6] = byte(value >> 8)
	destination[7] = byte(value)
}

// Append_Uint_64 extends buffer inside existing caller capacity.
func Append_Uint_64(
	buffer Bytes, value Word_64, order Byte_Order,
) (result Uint_64_Bytes) {
	defer func() { Uint_64_Bytes_Invariants(result, "append_uint_64.result") }()
	Bytes_Invariants(buffer, "append_uint_64.buffer")
	Word_64_Invariants(value, "append_uint_64.value")
	Byte_Order_Invariants(order, "append_uint_64.order")
	result = Uint_64_Bytes(extend(buffer, Varint_Size(UINT_64_SIZE)))
	Put_Uint_64(Bytes(result[len(buffer):]), value, order)
	return result
}

// Extends view only after complete size and capacity validation.
func extend(buffer Bytes, additional Varint_Size) (result Nonempty_Bytes) {
	defer func() { Nonempty_Bytes_Invariants(result, "extend.result") }()
	Bytes_Invariants(buffer, "extend.buffer")
	Varint_Size_Invariants(additional, "extend.additional")
	size := len(buffer) + int(additional)
	aver.Always(
		size <= BYTE_SIZE_MAXIMUM,
		"Extended binary bytes stay inside package size boundary.",
	)
	aver.Always(
		size <= cap(buffer),
		"Extended binary bytes fit caller-owned capacity.",
	)
	return Nonempty_Bytes(buffer[:size])
}

// Counts bytes before Put or Append so short storage fails before mutation.
func unsigned_varint_size(value Word_64) (size Varint_Size) {
	defer func() { Varint_Size_Invariants(size, "unsigned_varint_size.size") }()
	Word_64_Invariants(value, "unsigned_varint_size.value")
	size = VARINT_SIZE_MINIMUM
	for value >= 0x80 {
		size++
		value >>= 7
	}
	return size
}

// Put_Unsigned_Varint encodes unsigned value into caller storage.
func Put_Unsigned_Varint(destination Bytes, value Word_64) (size Varint_Size) {
	defer func() { Varint_Size_Invariants(size, "put_unsigned_varint.size") }()
	Bytes_Invariants(destination, "put_unsigned_varint.destination")
	Word_64_Invariants(value, "put_unsigned_varint.value")
	encoded_size := unsigned_varint_size(value)
	aver.Always(len(destination) >= int(encoded_size), "Varint destination is large enough.")
	destination = destination[:encoded_size]
	position := 0
	for value >= 0x80 {
		destination[position] = byte(value) | 0x80
		value >>= 7
		position++
	}
	destination[position] = byte(value)
	size = Varint_Size(position + 1)
	return size
}

// Append_Unsigned_Varint extends buffer inside existing caller capacity.
func Append_Unsigned_Varint(buffer Bytes, value Word_64) (result Nonempty_Bytes) {
	defer func() {
		Nonempty_Bytes_Invariants(result, "append_unsigned_varint.result")
	}()
	Bytes_Invariants(buffer, "append_unsigned_varint.buffer")
	Word_64_Invariants(value, "append_unsigned_varint.value")
	size := unsigned_varint_size(value)
	result = extend(buffer, size)
	Put_Unsigned_Varint(Bytes(result[len(buffer):]), value)
	return result
}

// Unsigned_Varint decodes one unsigned value from source prefix.
func Unsigned_Varint(source Bytes) (value Word_64, count Varint_Count) {
	defer func() {
		Word_64_Invariants(value, "unsigned_varint.value")
		Varint_Count_Invariants(count, "unsigned_varint.count")
	}()
	Bytes_Invariants(source, "unsigned_varint.source")
	var shift uint
	for index, octet := range source {
		if index == VARINT_SIZE_64_MAXIMUM {
			return 0, Varint_Count(-(index + 1))
		}
		if octet < 0x80 {
			if index == VARINT_SIZE_64_MAXIMUM-1 {
				if octet > 1 {
					return 0, Varint_Count(-(index + 1))
				}
			}
			return value | Word_64(octet)<<shift, Varint_Count(index + 1)
		}
		value |= Word_64(octet&0x7f) << shift
		shift += 7
	}
	return 0, 0
}

// Put_Varint zig-zag maps signed value into unsigned varint.
func Put_Varint(destination Bytes, value Integer_64) (size Varint_Size) {
	defer func() { Varint_Size_Invariants(size, "put_varint.size") }()
	Bytes_Invariants(destination, "put_varint.destination")
	Integer_64_Invariants(value, "put_varint.value")
	unsigned := Word_64(value) << 1
	if value < 0 {
		unsigned = ^unsigned
	}
	size = Put_Unsigned_Varint(destination, unsigned)
	return size
}

// Append_Varint extends buffer with zig-zag mapped signed value.
func Append_Varint(buffer Bytes, value Integer_64) (result Nonempty_Bytes) {
	defer func() { Nonempty_Bytes_Invariants(result, "append_varint.result") }()
	Bytes_Invariants(buffer, "append_varint.buffer")
	Integer_64_Invariants(value, "append_varint.value")
	unsigned := Word_64(value) << 1
	if value < 0 {
		unsigned = ^unsigned
	}
	result = Append_Unsigned_Varint(buffer, unsigned)
	return result
}

// Varint decodes one zig-zag mapped signed value.
func Varint(source Bytes) (value Integer_64, count Varint_Count) {
	defer func() {
		Integer_64_Invariants(value, "varint.value")
		Varint_Count_Invariants(count, "varint.count")
	}()
	Bytes_Invariants(source, "varint.source")
	unsigned, count := Unsigned_Varint(source)
	value = Integer_64(unsigned >> 1)
	if unsigned&1 != 0 {
		value = ^value
	}
	return value, count
}

// Read_Byte_State keeps arbitrary caller state concrete without owning it.
type Read_Byte_State unsafe.Pointer

// Read_Byte_State_Invariants rejects missing injected state.
func Read_Byte_State_Invariants(value Read_Byte_State, _ aver.Namespace) {
	aver.Always(value != nil, "Varint reader state exists.")
}

// Read_Byte_Function injects one byte read without behavioral interface or closure capture.
type Read_Byte_Function func(state Read_Byte_State) (value byte, err error)

// Read_Unsigned_Varint consumes no more than format maximum from reader.
func Read_Unsigned_Varint(
	state Read_Byte_State, read_byte Read_Byte_Function,
) (value Word_64, err Error) {
	defer func() { Word_64_Invariants(value, "read_unsigned_varint.value") }()
	Read_Byte_State_Invariants(state, "read_unsigned_varint.state")
	var shift uint
	for index := 0; index < VARINT_SIZE_64_MAXIMUM; index++ {
		octet, read_error := read_byte(state)
		if read_error != nil {
			if index > 0 {
				if read_error == nbio.Stream_EOF {
					read_error = nbio.Stream_Unexpected_EOF
				}
			}
			return value, read_error
		}
		if octet < 0x80 {
			if index == VARINT_SIZE_64_MAXIMUM-1 {
				if octet > 1 {
					return value, Error_Varint_Overflow
				}
			}
			return value | Word_64(octet)<<shift, nil
		}
		value |= Word_64(octet&0x7f) << shift
		shift += 7
	}
	return value, Error_Varint_Overflow
}

// Read_Varint decodes one zig-zag mapped signed value from reader.
func Read_Varint(
	state Read_Byte_State, read_byte Read_Byte_Function,
) (value Integer_64, err Error) {
	defer func() { Integer_64_Invariants(value, "read_varint.value") }()
	Read_Byte_State_Invariants(state, "read_varint.state")
	unsigned, err := Read_Unsigned_Varint(state, read_byte)
	value = Integer_64(unsigned >> 1)
	if unsigned&1 != 0 {
		value = ^value
	}
	return value, err
}

// Size rejects variable-width grammar before any encode or stream side effect.
func Size(source Value) (size Value_Size) {
	defer func() { Value_Size_Invariants(size, "size.size") }()
	subject := reflect.ValueOf(source)
	if !subject.IsValid() {
		return VALUE_SIZE_INVALID
	}
	if subject.Kind() == reflect.Pointer {
		if subject.IsNil() {
			return VALUE_SIZE_INVALID
		}
		subject = subject.Elem()
	}
	encoded_size := reflected_size(subject)
	if encoded_size == VALUE_SIZE_INVALID {
		return VALUE_SIZE_INVALID
	}
	aver.Always(
		encoded_size <= VALUE_SIZE_MAXIMUM,
		"Structured binary value fits bounded byte storage.",
	)
	return encoded_size
}

// Computes top-level slice size from runtime length; nested slices remain invalid.
func reflected_size(value reflect.Value) (size Value_Size) {
	defer func() { Value_Size_Invariants(size, "reflected_size.size") }()
	if value.Kind() != reflect.Slice {
		return type_size(value.Type(), INITIAL_DEPTH_ROOT)
	}
	element_count := Element_Count(value.Len())
	Element_Count_Invariants(element_count, "reflected_size.element_count")
	aver.Always(
		int(element_count) <= VALUE_ELEMENT_COUNT_MAXIMUM,
		"Structured slice stays inside package element boundary.",
	)
	element_size := type_size(value.Type().Elem(), INITIAL_DEPTH_SLICE_ELEMENT)
	if element_size == VALUE_SIZE_INVALID {
		return VALUE_SIZE_INVALID
	}
	size = Value_Size(multiplied_size(Byte_Count(element_size), element_count))
	return size
}

// Explicit fixed stack makes hostile nesting consume bounded memory.
func type_size(value_type reflect.Type, depth Initial_Depth) (size Value_Size) {
	defer func() { Value_Size_Invariants(size, "type_size.size") }()
	Initial_Depth_Invariants(depth, "type_size.depth")
	aver.Always(
		int(depth) <= VALUE_NESTING_MAXIMUM,
		"Structured value stays inside package nesting boundary.",
	)
	var types [VALUE_STACK_SIZE]reflect.Type
	var positions [VALUE_STACK_SIZE]int
	var depths [VALUE_STACK_SIZE]int
	var multipliers [VALUE_STACK_SIZE]int
	types[0] = value_type
	depths[0] = int(depth)
	multipliers[0] = 1
	stack_count := Stack_Count(ACTIVE_STACK_COUNT_MINIMUM)
	for stack_count > 0 {
		stack_index := int(stack_count) - 1
		current_type := types[stack_index]
		if current_type.Kind() == reflect.Array {
			array_size := Element_Count(current_type.Len())
			Element_Count_Invariants(array_size, "type_size.array_size")
			aver.Always(
				int(array_size) <= VALUE_ELEMENT_COUNT_MAXIMUM,
				"Structured array stays inside package element boundary.",
			)
			multiplier := multiplied_size(
				Byte_Count(multipliers[stack_index]), array_size,
			)
			depths[stack_index]++
			aver.Always(
				depths[stack_index] <= VALUE_NESTING_MAXIMUM,
				"Structured array stays inside package nesting boundary.",
			)
			types[stack_index] = current_type.Elem()
			multipliers[stack_index] = int(multiplier)
			continue
		}
		if current_type.Kind() == reflect.Struct {
			stack_count = push_struct_field(
				Type_Stack(types[:]), Position_Stack(positions[:]),
				Depth_Stack(depths[:]), Multiplier_Stack(multipliers[:]),
				Active_Stack_Count(stack_count),
			)
			continue
		}
		leaf_size, supported := primitive_size(current_type.Kind())
		if !supported {
			return VALUE_SIZE_INVALID
		}
		field_size := multiplied_size(
			Byte_Count(leaf_size), Element_Count(multipliers[stack_index]),
		)
		aver.Always(
			int(size) <= VALUE_SIZE_MAXIMUM-int(field_size),
			"Structured value stays inside package encoded-size boundary.",
		)
		size += Value_Size(field_size)
		stack_count--
	}
	return size
}

// Type_Stack retains one bounded iterative type traversal.
type Type_Stack []reflect.Type

// Type_Stack_Invariants fixes storage to the nesting boundary plus root.
func Type_Stack_Invariants(value Type_Stack, _ aver.Namespace) {
	aver.Always(len(value) == VALUE_STACK_SIZE, "Type stack has one slot per level.")
}

// Position_Stack retains the next record field at each traversal level.
type Position_Stack []int

// Position_Stack_Invariants fixes storage to the nesting boundary plus root.
func Position_Stack_Invariants(value Position_Stack, _ aver.Namespace) {
	aver.Always(len(value) == VALUE_STACK_SIZE, "Position stack has one slot per level.")
}

// Depth_Stack retains the validated depth at each traversal level.
type Depth_Stack []int

// Depth_Stack_Invariants fixes storage to the nesting boundary plus root.
func Depth_Stack_Invariants(value Depth_Stack, _ aver.Namespace) {
	aver.Always(len(value) == VALUE_STACK_SIZE, "Depth stack has one slot per level.")
}

// Multiplier_Stack retains array cardinality at each traversal level.
type Multiplier_Stack []int

// Multiplier_Stack_Invariants fixes storage to the nesting boundary plus root.
func Multiplier_Stack_Invariants(value Multiplier_Stack, _ aver.Namespace) {
	aver.Always(len(value) == VALUE_STACK_SIZE, "Multiplier stack has one slot per level.")
}

// Separate frame transition keeps hostile record nesting on the fixed stack.
func push_struct_field(
	types Type_Stack, positions Position_Stack, depths Depth_Stack,
	multipliers Multiplier_Stack, stack_count Active_Stack_Count,
) (next Stack_Count) {
	defer func() { Stack_Count_Invariants(next, "push_struct_field.next") }()
	Type_Stack_Invariants(types, "push_struct_field.types")
	Position_Stack_Invariants(positions, "push_struct_field.positions")
	Depth_Stack_Invariants(depths, "push_struct_field.depths")
	Multiplier_Stack_Invariants(multipliers, "push_struct_field.multipliers")
	Active_Stack_Count_Invariants(stack_count, "push_struct_field.stack_count")
	stack_index := int(stack_count) - 1
	value_type := types[stack_index]
	aver.Always(
		value_type.NumField() <= VALUE_ELEMENT_COUNT_MAXIMUM,
		"Structured record stays inside package field boundary.",
	)
	if positions[stack_index] == value_type.NumField() {
		return Stack_Count(stack_count - 1)
	}
	child_depth := depths[stack_index] + 1
	aver.Always(
		child_depth <= VALUE_NESTING_MAXIMUM,
		"Structured record stays inside package nesting boundary.",
	)
	field := value_type.Field(positions[stack_index])
	positions[stack_index]++
	types[int(stack_count)] = field.Type
	positions[int(stack_count)] = 0
	depths[int(stack_count)] = child_depth
	multipliers[int(stack_count)] = multipliers[stack_index]
	return Stack_Count(stack_count + 1)
}

// Primitive width lookup keeps collection traversal branch-free at leaves.
func primitive_size(kind reflect.Kind) (size Fixed_Size, supported Boolean) {
	defer func() {
		Fixed_Size_Invariants(size, "primitive_size.size")
		Boolean_Invariants(supported, "primitive_size.supported")
	}()
	size = UINT_8_SIZE
	switch kind {
	case reflect.Bool, reflect.Int8, reflect.Uint8:
		return UINT_8_SIZE, true
	case reflect.Int16, reflect.Uint16:
		return UINT_16_SIZE, true
	case reflect.Int32, reflect.Uint32:
		return UINT_32_SIZE, true
	case reflect.Int64, reflect.Uint64:
		return UINT_64_SIZE, true
	}
	return size, false
}

// Collection kind shares iterative child traversal.
func kind_is_collection(kind reflect.Kind) (collection Boolean) {
	defer func() { Boolean_Invariants(collection, "kind_is_collection.collection") }()
	switch kind {
	case reflect.Array, reflect.Slice:
		return true
	}
	return false
}

// Caps multiplication before machine integer overflow can hide oversized value.
func multiplied_size(
	element_size Byte_Count, count Element_Count,
) (size Byte_Count) {
	defer func() { Byte_Count_Invariants(size, "multiplied_size.size") }()
	Byte_Count_Invariants(element_size, "multiplied_size.element_size")
	Element_Count_Invariants(count, "multiplied_size.count")
	if element_size == 0 {
		return 0
	}
	if count == 0 {
		return 0
	}
	aver.Always(
		int(element_size) <= VALUE_SIZE_MAXIMUM/int(count),
		"Structured collection stays inside package encoded-size boundary.",
	)
	return element_size * Byte_Count(count)
}

// Encode writes only after full type and destination validation.
func Encode(
	destination Bytes, source Value, order Byte_Order,
) (count Byte_Count, err Error) {
	defer func() { Byte_Count_Invariants(count, "encode.count") }()
	Bytes_Invariants(destination, "encode.destination")
	Byte_Order_Invariants(order, "encode.order")
	size := Size(source)
	if size == VALUE_SIZE_INVALID {
		return 0, Error_Invalid_Type
	}
	if len(destination) < int(size) {
		return 0, Error_Buffer_Too_Small
	}
	subject := reflect.ValueOf(source)
	if subject.Kind() == reflect.Pointer {
		subject = subject.Elem()
	}
	encode_value(destination[:int(size)], subject, little_order(order))
	return Byte_Count(size), nil
}

// Append retains caller ownership by extending only existing capacity.
func Append(
	buffer Bytes, source Value, order Byte_Order,
) (result Bytes, err Error) {
	defer func() { Bytes_Invariants(result, "append.result") }()
	Bytes_Invariants(buffer, "append.buffer")
	Byte_Order_Invariants(order, "append.order")
	size := Size(source)
	if size == VALUE_SIZE_INVALID {
		return nil, Error_Invalid_Type
	}
	wanted := len(buffer) + int(size)
	aver.Always(
		wanted <= BYTE_SIZE_MAXIMUM,
		"Appended structured binary value stays inside package size boundary.",
	)
	aver.Always(
		wanted <= cap(buffer),
		"Appended structured binary value fits caller-owned capacity.",
	)
	result = buffer[:wanted]
	_, err = Encode(result[len(buffer):], source, order)
	return result, err
}

// Decode validates complete source before mutating caller destination.
func Decode(
	source Bytes, destination Value, order Byte_Order,
) (count Byte_Count, err Error) {
	defer func() { Byte_Count_Invariants(count, "decode.count") }()
	Bytes_Invariants(source, "decode.source")
	Byte_Order_Invariants(order, "decode.order")
	target, size, valid := decode_target(destination)
	if !valid {
		return 0, Error_Invalid_Type
	}
	if len(source) < int(size) {
		return 0, Error_Buffer_Too_Small
	}
	decode_value(source[:int(size)], target, little_order(order))
	return Byte_Count(size), nil
}

// Separates addressability validation from source mutation boundary.
func decode_target(
	destination Value,
) (value reflect.Value, size Value_Size, valid Boolean) {
	defer func() {
		Value_Size_Invariants(size, "decode_target.size")
		Boolean_Invariants(valid, "decode_target.valid")
	}()
	value = reflect.ValueOf(destination)
	if !value.IsValid() {
		return reflect.Value{}, VALUE_SIZE_INVALID, false
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return reflect.Value{}, VALUE_SIZE_INVALID, false
		}
		value = value.Elem()
	} else if value.Kind() != reflect.Slice {
		return reflect.Value{}, VALUE_SIZE_INVALID, false
	}
	size = reflected_size(value)
	if size == VALUE_SIZE_INVALID {
		return reflect.Value{}, VALUE_SIZE_INVALID, false
	}
	aver.Always(
		int(size) <= VALUE_SIZE_MAXIMUM,
		"Decoded structured binary value fits bounded byte storage.",
	)
	return value, size, true
}

// Reader_Init binds injected Stream and caller scratch without submitting work.
func Reader_Init(reader Reader_Handle, stream nbio.Stream, scratch Bytes) {
	Reader_Handle_Invariants(reader, "Reader_Init.reader")
	Bytes_Invariants(scratch, "Reader_Init.scratch")
	aver.Always(!reader.Active, "Reader_Init owns idle Reader state.")
	aver.Always(stream.Procedure != nil, "Reader_Init has concrete Stream.")
	*reader = Reader{Stream: stream, Scratch: scratch, Initialized: true}
}

// Read defers decode until Stream supplies complete fixed-width value.
func Read(
	reader Reader_Handle, completion nbio.Completion_Handle,
	destination Value, order Byte_Order,
	callback nbio.Callback,
) {
	Reader_Handle_Invariants(reader, "Read.reader")
	Reader_Invariants(*reader, "Read.reader_value")
	Byte_Order_Invariants(order, "Read.order")
	aver.Always(reader.Initialized, "Read uses initialized Reader.")
	aver.Always(completion != nil, "Read has completion storage.")
	aver.Always(
		completion == &reader.Completion,
		"Read submits completion owned by Reader.",
	)
	aver.Always(callback != nil, "Read has callback.")
	aver.Always(!reader.Active, "Read owns free Reader callback slot.")
	target := reflect.ValueOf(destination)
	if !target.IsValid() {
		completion.Data = 0
		completion.Error = Error_Invalid_Type
		callback(completion)
		return
	}
	if target.Kind() != reflect.Pointer {
		completion.Data = 0
		completion.Error = Error_Invalid_Type
		callback(completion)
		return
	}
	_, size, valid := decode_target(destination)
	if !valid {
		completion.Data = 0
		completion.Error = Error_Invalid_Type
		callback(completion)
		return
	}
	aver.Always(
		len(reader.Scratch) >= int(size),
		"Read scratch holds complete structured binary value.",
	)
	reader.Callback = callback
	reader.Destination = destination
	reader.Order = order
	reader.Size = Stream_Size(size)
	reader.Count = 0
	reader.Active = true
	completion.Data = 0
	completion.Error = nil
	reader_progress(completion)
}

// Reader progress uses trampoline because concrete Stream may retire inline.
func reader_progress(completion nbio.Completion_Handle) {
	reader := (*Reader)(unsafe.Pointer(completion))
	for bool(reader.Active) && !bool(reader.Wait_Active) {
		if reader.Count == Byte_Count(reader.Size) {
			reader_decode_finish(completion)
			return
		}
		start := int(reader.Count)
		end := int(reader.Size)
		reader.Wait_Active = true
		reader.Submission_Active = true
		reader.Continue = false
		nbio.Read(
			reader.Stream, completion, reader.Scratch[start:end],
			reader_stream_complete,
		)
		reader.Submission_Active = false
		retired_inline := reader.Continue
		reader.Continue = false
		if !retired_inline {
			return
		}
	}
}

func reader_stream_complete(completion nbio.Completion_Handle) {
	reader := (*Reader)(unsafe.Pointer(completion))
	aver.Always(reader.Active, "Reader callback belongs to active operation.")
	aver.Always(reader.Wait_Active, "Reader callback retires submitted transfer.")
	reader.Wait_Active = false
	requested := int(reader.Size) - int(reader.Count)
	count := completion.Data
	if count < 0 {
		count = 0
		completion.Error = nbio.Stream_Negative_Read
	}
	if count > requested {
		count = 0
		completion.Error = nbio.Stream_Short_Buffer
	}
	reader.Count += Byte_Count(count)
	completion.Data = count
	if reader.Count == Byte_Count(reader.Size) {
		completion.Error = nil
		reader_decode_finish(completion)
		return
	}
	if completion.Error != nil {
		if completion.Error == nbio.Stream_EOF {
			if reader.Count > 0 {
				completion.Error = nbio.Stream_Unexpected_EOF
			}
		}
		reader_finish(completion)
		return
	}
	if count == 0 {
		completion.Error = nbio.Stream_No_Progress
		reader_finish(completion)
		return
	}
	if reader.Submission_Active {
		reader.Continue = true
		return
	}
	reader_progress(completion)
}

func reader_decode_finish(completion nbio.Completion_Handle) {
	reader := (*Reader)(unsafe.Pointer(completion))
	_, completion.Error = Decode(
		reader.Scratch[:reader.Size], reader.Destination, reader.Order,
	)
	reader_finish(completion)
}

func reader_finish(completion nbio.Completion_Handle) {
	reader := (*Reader)(unsafe.Pointer(completion))
	callback := reader.Callback
	count := reader.Count
	reader.Callback = nil
	reader.Destination = nil
	reader.Active = false
	reader.Submission_Active = false
	reader.Wait_Active = false
	reader.Continue = false
	completion.Data = int(count)
	callback(completion)
}

// Writer_Init binds injected Stream and caller scratch without submitting work.
func Writer_Init(writer Writer_Handle, stream nbio.Stream, scratch Bytes) {
	Writer_Handle_Invariants(writer, "Writer_Init.writer")
	Bytes_Invariants(scratch, "Writer_Init.scratch")
	aver.Always(!writer.Active, "Writer_Init owns idle Writer state.")
	aver.Always(stream.Procedure != nil, "Writer_Init has concrete Stream.")
	*writer = Writer{Stream: stream, Scratch: scratch, Initialized: true}
}

// Write encodes before submission so invalid grammar never mutates Stream.
func Write(
	writer Writer_Handle, completion nbio.Completion_Handle, source Value, order Byte_Order,
	callback nbio.Callback,
) {
	Writer_Handle_Invariants(writer, "Write.writer")
	Writer_Invariants(*writer, "Write.writer_value")
	Byte_Order_Invariants(order, "Write.order")
	aver.Always(writer.Initialized, "Write uses initialized Writer.")
	aver.Always(completion != nil, "Write has completion storage.")
	aver.Always(
		completion == &writer.Completion,
		"Write submits completion owned by Writer.",
	)
	aver.Always(callback != nil, "Write has callback.")
	aver.Always(!writer.Active, "Write owns free Writer callback slot.")
	size := Size(source)
	if size == VALUE_SIZE_INVALID {
		completion.Data = 0
		completion.Error = Error_Invalid_Type
		callback(completion)
		return
	}
	aver.Always(
		len(writer.Scratch) >= int(size),
		"Write scratch holds complete structured binary value.",
	)
	_, encode_error := Encode(writer.Scratch[:size], source, order)
	if encode_error != nil {
		completion.Data = 0
		completion.Error = encode_error
		callback(completion)
		return
	}
	writer.Callback = callback
	writer.Order = order
	writer.Size = Stream_Size(size)
	writer.Active = true
	completion.Data = 0
	completion.Error = nil
	if size == 0 {
		writer_finish(completion)
		return
	}
	nbio.Write(writer.Stream, completion, writer.Scratch[:size], writer_stream_complete)
}

func writer_stream_complete(completion nbio.Completion_Handle) {
	writer := (*Writer)(unsafe.Pointer(completion))
	aver.Always(writer.Active, "Writer callback belongs to active operation.")
	count := completion.Data
	if count < 0 {
		count = 0
		completion.Error = nbio.Stream_Negative_Write
	}
	if count > int(writer.Size) {
		count = 0
		completion.Error = nbio.Stream_Invalid_Write
	}
	if completion.Error == nil {
		if count < int(writer.Size) {
			completion.Error = nbio.Stream_Short_Write
		}
	}
	completion.Data = count
	writer_finish(completion)
}

func writer_finish(completion nbio.Completion_Handle) {
	writer := (*Writer)(unsafe.Pointer(completion))
	callback := writer.Callback
	count := completion.Data
	writer.Callback = nil
	writer.Active = false
	completion.Data = count
	callback(completion)
}

// Walk order matches Go field and element order; blank fields reserve zeroed format space.
func encode_value(
	destination Bytes, subject reflect.Value, little Boolean,
) {
	Bytes_Invariants(destination, "encode_value.destination")
	Boolean_Invariants(little, "encode_value.little")
	var values [VALUE_STACK_SIZE]reflect.Value
	var child_positions [VALUE_STACK_SIZE]int
	values[0] = subject
	stack_count := 1
	position := 0
	for stack_count > 0 {
		stack_index := stack_count - 1
		current := values[stack_index]
		if kind_is_collection(current.Kind()) {
			if child_positions[stack_index] == current.Len() {
				stack_count--
				continue
			}
			child := current.Index(child_positions[stack_index])
			child_positions[stack_index]++
			values[stack_count] = child
			child_positions[stack_count] = 0
			stack_count++
			continue
		}
		if current.Kind() == reflect.Struct {
			if child_positions[stack_index] == current.NumField() {
				stack_count--
				continue
			}
			field_index := child_positions[stack_index]
			child_positions[stack_index]++
			field := current.Type().Field(field_index)
			if field.Name == "_" {
				padding_size := int(type_size(
					field.Type, INITIAL_DEPTH_SLICE_ELEMENT,
				))
				limit := position + padding_size
				for position < limit {
					destination[position] = 0
					position++
				}
				continue
			}
			values[stack_count] = current.Field(field_index)
			child_positions[stack_count] = 0
			stack_count++
			continue
		}
		position += int(encode_leaf(
			Nonempty_Bytes(destination[position:]), current, little,
		))
		stack_count--
	}
}

// Leaf encoding stays separate so iterative traversal remains bounded and short.
func encode_leaf(
	destination Nonempty_Bytes, subject reflect.Value, little Boolean,
) (size Fixed_Size) {
	defer func() { Fixed_Size_Invariants(size, "encode_leaf.size") }()
	Nonempty_Bytes_Invariants(destination, "encode_leaf.destination")
	Boolean_Invariants(little, "encode_leaf.little")
	size = UINT_8_SIZE
	switch subject.Kind() {
	case reflect.Bool:
		if subject.Bool() {
			destination[0] = 1
		} else {
			destination[0] = 0
		}
	case reflect.Int8:
		destination[0] = byte(subject.Int())
	case reflect.Uint8:
		destination[0] = byte(subject.Uint())
	case reflect.Int16:
		size = UINT_16_SIZE
		put_uint_16_raw(Bytes(destination), Word_16(uint16(subject.Int())), little)
	case reflect.Uint16:
		size = UINT_16_SIZE
		put_uint_16_raw(Bytes(destination), Word_16(subject.Uint()), little)
	case reflect.Int32:
		size = UINT_32_SIZE
		put_uint_32_raw(Bytes(destination), Word_32(uint32(subject.Int())), little)
	case reflect.Uint32:
		size = UINT_32_SIZE
		put_uint_32_raw(Bytes(destination), Word_32(subject.Uint()), little)
	case reflect.Int64:
		size = UINT_64_SIZE
		put_uint_64_raw(Bytes(destination), Word_64(subject.Int()), little)
	case reflect.Uint64:
		size = UINT_64_SIZE
		put_uint_64_raw(Bytes(destination), Word_64(subject.Uint()), little)
	}
	return size
}

// Blank fields advance cursor without touching inaccessible reflect values.
func decode_value(source Bytes, target reflect.Value, little Boolean) {
	Bytes_Invariants(source, "decode_value.source")
	Boolean_Invariants(little, "decode_value.little")
	var values [VALUE_STACK_SIZE]reflect.Value
	var child_positions [VALUE_STACK_SIZE]int
	values[0] = target
	stack_count := 1
	position := 0
	for stack_count > 0 {
		stack_index := stack_count - 1
		current := values[stack_index]
		if kind_is_collection(current.Kind()) {
			if child_positions[stack_index] == current.Len() {
				stack_count--
				continue
			}
			child := current.Index(child_positions[stack_index])
			child_positions[stack_index]++
			values[stack_count] = child
			child_positions[stack_count] = 0
			stack_count++
			continue
		}
		if current.Kind() == reflect.Struct {
			if child_positions[stack_index] == current.NumField() {
				stack_count--
				continue
			}
			field_index := child_positions[stack_index]
			child_positions[stack_index]++
			field := current.Type().Field(field_index)
			if field.Name == "_" {
				position += int(type_size(
					field.Type, INITIAL_DEPTH_SLICE_ELEMENT,
				))
				continue
			}
			values[stack_count] = current.Field(field_index)
			child_positions[stack_count] = 0
			stack_count++
			continue
		}
		position += int(decode_leaf(
			Nonempty_Bytes(source[position:]), current, little,
		))
		stack_count--
	}
}

// Leaf decoding stays separate so iterative traversal remains bounded and short.
func decode_leaf(
	source Nonempty_Bytes, target reflect.Value, little Boolean,
) (size Fixed_Size) {
	defer func() { Fixed_Size_Invariants(size, "decode_leaf.size") }()
	Nonempty_Bytes_Invariants(source, "decode_leaf.source")
	Boolean_Invariants(little, "decode_leaf.little")
	size = UINT_8_SIZE
	switch target.Kind() {
	case reflect.Bool:
		target.SetBool(source[0] != 0)
	case reflect.Int8:
		target.SetInt(int64(int8(source[0])))
	case reflect.Uint8:
		target.SetUint(uint64(source[0]))
	case reflect.Int16:
		size = UINT_16_SIZE
		word := uint_16_raw(Bytes(source), little)
		target.SetInt(int64(int16(word)))
	case reflect.Uint16:
		size = UINT_16_SIZE
		target.SetUint(uint64(uint_16_raw(Bytes(source), little)))
	case reflect.Int32:
		size = UINT_32_SIZE
		word := uint_32_raw(Bytes(source), little)
		target.SetInt(int64(int32(word)))
	case reflect.Uint32:
		size = UINT_32_SIZE
		target.SetUint(uint64(uint_32_raw(Bytes(source), little)))
	case reflect.Int64:
		size = UINT_64_SIZE
		target.SetInt(int64(uint_64_raw(Bytes(source), little)))
	case reflect.Uint64:
		size = UINT_64_SIZE
		target.SetUint(uint64(uint_64_raw(Bytes(source), little)))
	}
	return size
}
