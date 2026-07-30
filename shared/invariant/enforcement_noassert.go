//go:build noassert

// This file is the silent half of the compile-time dual described in enforcement.go: the same
// entry points, the same signatures, and bodies that do nothing. Selecting it with `-tags noassert`
// yields a binary in which no condition is enforced, no shape is validated, and no coverage is
// recorded.
//
// The tag is for measuring what enforcement costs and for a build that has already been measured
// and cannot afford it. It is not a release default: the framework's contract is that enforcement
// fires in every run mode, and a binary built this way answers "would this have held?" with
// silence. Nothing gates the tag, so choosing it is choosing to ship unchecked.
//
// A no-op link must still return a Product because the chain is an expression — `Dot_Product(ns).
// Sometimes(...).Ensure()` only type-checks if every link yields the next receiver. Returning the
// receiver unchanged keeps every method a trivially inlinable identity, so the whole chain folds
// away and only the caller's own condition expressions remain. Those are argument evaluation at the
// callsite, which no library-level dual can reach; a chain over pure comparisons costs nothing after
// inlining, while one that calls out to real work still pays for that work.

package invariant

// The zero Product is deliberate: with every method a no-op, no field is ever read, so the chain
// root skips the namespace check and the shape map lookup that dominate the enforcing build.
func Recorder_Dot_Product(recorder *Recorder, namespace Namespace) (product Product) {
	return Product{}
}

// Always is the one guard that fires without a chain, so its silence is the tag's whole meaning.
func Recorder_Always[T ~bool](recorder *Recorder, condition T, message string) {}

func (product Product) Sometimes(condition bool, message string) (next Product) {
	return product
}

func (product Product) Impossible(
	message string, references ...Dot_Element_Reference,
) (next Product) {
	return product
}

func (product Product) Range_Int(
	value int, minimum int, maximum int, excluded ...int,
) (next Product) {
	return product
}

func (product Product) Range_Int8(
	value int8, minimum int8, maximum int8, excluded ...int8,
) (next Product) {
	return product
}

func (product Product) Range_Int16(
	value int16, minimum int16, maximum int16, excluded ...int16,
) (next Product) {
	return product
}

func (product Product) Range_Int32(
	value int32, minimum int32, maximum int32, excluded ...int32,
) (next Product) {
	return product
}

func (product Product) Range_Int64(
	value int64, minimum int64, maximum int64, excluded ...int64,
) (next Product) {
	return product
}

func (product Product) Range_Uint(
	value uint, minimum uint, maximum uint, excluded ...uint,
) (next Product) {
	return product
}

func (product Product) Range_Uint8(
	value uint8, minimum uint8, maximum uint8, excluded ...uint8,
) (next Product) {
	return product
}

func (product Product) Range_Uint16(
	value uint16, minimum uint16, maximum uint16, excluded ...uint16,
) (next Product) {
	return product
}

func (product Product) Range_Uint32(
	value uint32, minimum uint32, maximum uint32, excluded ...uint32,
) (next Product) {
	return product
}

func (product Product) Range_Uint64(
	value uint64, minimum uint64, maximum uint64, excluded ...uint64,
) (next Product) {
	return product
}

func (product Product) Enum_Int(value int, members ...int) (next Product) {
	return product
}

func (product Product) Enum_Int8(value int8, members ...int8) (next Product) {
	return product
}

func (product Product) Enum_Int16(value int16, members ...int16) (next Product) {
	return product
}

func (product Product) Enum_Int32(value int32, members ...int32) (next Product) {
	return product
}

func (product Product) Enum_Int64(value int64, members ...int64) (next Product) {
	return product
}

func (product Product) Enum_Uint(value uint, members ...uint) (next Product) {
	return product
}

func (product Product) Enum_Uint8(value uint8, members ...uint8) (next Product) {
	return product
}

func (product Product) Enum_Uint16(value uint16, members ...uint16) (next Product) {
	return product
}

func (product Product) Enum_Uint32(value uint32, members ...uint32) (next Product) {
	return product
}

func (product Product) Enum_Uint64(value uint64, members ...uint64) (next Product) {
	return product
}

// Ensure is where the enforcing build validates the shape, panics for every matching carve, and
// credits the grid. Here it terminates the expression and nothing else.
func (product Product) Ensure() {}
