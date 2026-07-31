// Package primitive_mandates exists because registration ignores test files and must parse literal
// callsites before the test can invoke the real default helper source.
package primitive_mandates

import invariant "local/james-orcales/shared/invariant/default"

func signed_int(value int) {
	invariant.Int_Invariants(value, "primitive.int")
}

func signed_int8(value int8) {
	invariant.Int8_Invariants(value, "primitive.int8")
}

func signed_int16(value int16) {
	invariant.Int16_Invariants(value, "primitive.int16")
}

func signed_int32(value int32) {
	invariant.Int32_Invariants(value, "primitive.int32")
}

func signed_int64(value int64) {
	invariant.Int64_Invariants(value, "primitive.int64")
}

func unsigned_int(value uint) {
	invariant.Uint_Invariants(value, "primitive.uint")
}

func unsigned_int8(value uint8) {
	invariant.Uint8_Invariants(value, "primitive.uint8")
}

func unsigned_int16(value uint16) {
	invariant.Uint16_Invariants(value, "primitive.uint16")
}

func unsigned_int32(value uint32) {
	invariant.Uint32_Invariants(value, "primitive.uint32")
}

func unsigned_int64(value uint64) {
	invariant.Uint64_Invariants(value, "primitive.uint64")
}

func float32_values(value float32) {
	invariant.Float32_Invariants(value, "primitive.float32")
}

func float64_values(value float64) {
	invariant.Float64_Invariants(value, "primitive.float64")
}

func boolean_values(value bool) {
	invariant.Boolean_Invariants(value, "primitive.boolean")
}
