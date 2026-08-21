package aver_test

import (
	"testing"

	"local/james-orcales/shared/simulation/aver"
	"local/james-orcales/shared/testify"
)

// Allocation_Subject keeps plan lookup shape real without sharing mode-specific test fixtures.
type Allocation_Subject int

// One callback holds every exported runtime entry point. Any hidden allocation then fails one
// measured execution instead of disappearing in an average across unrelated calls.
func Test_All_Runtime_Entry_Points_Have_Zero_Allocations(t *testing.T) {
	recorder := &aver.Recorder{}
	testify.Zero_Allocation(t, func() {
		aver.Recorder_Always(recorder, true, "always")
		aver.Recorder_Sometimes(recorder, true, "sometimes")
		assert_inline_ranges(recorder)
		assert_inline_holed_ranges(recorder)
		assert_inline_enums(recorder)
		builder := aver.Recorder_Tree(
			recorder, Allocation_Subject(0), "all runtime entry points")
		builder = builder.Sometimes(true, "sometimes")
		builder = assert_builder_ranges(builder)
		builder = assert_builder_holed_ranges(builder)
		builder = assert_builder_enums(builder)
		builder.Ensure()
	})
}

func assert_inline_ranges(recorder *aver.Recorder) {
	aver.Recorder_Range(recorder, 5, 0, 10, "range int")
	aver.Recorder_Range(recorder, int8(5), int8(0), int8(10), "range int8")
	aver.Recorder_Range(recorder, int16(5), int16(0), int16(10), "range int16")
	aver.Recorder_Range(recorder, int32(5), int32(0), int32(10), "range int32")
	aver.Recorder_Range(recorder, int64(5), int64(0), int64(10), "range int64")
	aver.Recorder_Range(recorder, uint(5), uint(0), uint(10), "range uint")
	aver.Recorder_Range(recorder, uint8(5), uint8(0), uint8(10), "range uint8")
	aver.Recorder_Range(recorder, uint16(5), uint16(0), uint16(10), "range uint16")
	aver.Recorder_Range(recorder, uint32(5), uint32(0), uint32(10), "range uint32")
	aver.Recorder_Range(recorder, uint64(5), uint64(0), uint64(10), "range uint64")
}

func assert_inline_holed_ranges(recorder *aver.Recorder) {
	aver.Recorder_Range_Holed(recorder, 5, 0, 10, 1, 2, 3, 4, "holed int")
	aver.Recorder_Range_Holed(recorder,
		int8(5), int8(0), int8(10), int8(1), int8(2), int8(3), int8(4), "holed int8")
	aver.Recorder_Range_Holed(recorder,
		int16(5), int16(0), int16(10), int16(1), int16(2), int16(3), int16(4),
		"holed int16")
	aver.Recorder_Range_Holed(recorder,
		int32(5), int32(0), int32(10), int32(1), int32(2), int32(3), int32(4),
		"holed int32")
	aver.Recorder_Range_Holed(recorder,
		int64(5), int64(0), int64(10), int64(1), int64(2), int64(3), int64(4),
		"holed int64")
	aver.Recorder_Range_Holed(recorder,
		uint(5), uint(0), uint(10), uint(1), uint(2), uint(3), uint(4), "holed uint")
	aver.Recorder_Range_Holed(recorder,
		uint8(5), uint8(0), uint8(10), uint8(1), uint8(2), uint8(3), uint8(4),
		"holed uint8")
	aver.Recorder_Range_Holed(recorder,
		uint16(5), uint16(0), uint16(10), uint16(1), uint16(2), uint16(3), uint16(4),
		"holed uint16")
	aver.Recorder_Range_Holed(recorder,
		uint32(5), uint32(0), uint32(10), uint32(1), uint32(2), uint32(3), uint32(4),
		"holed uint32")
	aver.Recorder_Range_Holed(recorder,
		uint64(5), uint64(0), uint64(10), uint64(1), uint64(2), uint64(3), uint64(4),
		"holed uint64")
}

func assert_inline_enums(recorder *aver.Recorder) {
	aver.Recorder_Enum(recorder, 1, 1, 2, "enum int")
	aver.Recorder_Enum(recorder, int8(1), int8(1), int8(2), "enum int8")
	aver.Recorder_Enum(recorder, int16(1), int16(1), int16(2), "enum int16")
	aver.Recorder_Enum(recorder, int32(1), int32(1), int32(2), "enum int32")
	aver.Recorder_Enum(recorder, int64(1), int64(1), int64(2), "enum int64")
	aver.Recorder_Enum(recorder, uint(1), uint(1), uint(2), "enum uint")
	aver.Recorder_Enum(recorder, uint8(1), uint8(1), uint8(2), "enum uint8")
	aver.Recorder_Enum(recorder, uint16(1), uint16(1), uint16(2), "enum uint16")
	aver.Recorder_Enum(recorder, uint32(1), uint32(1), uint32(2), "enum uint32")
	aver.Recorder_Enum(recorder, uint64(1), uint64(1), uint64(2), "enum uint64")
}

func assert_builder_ranges(
	builder aver.Assertion_Builder,
) (next aver.Assertion_Builder) {
	builder = builder.Range_Int(5, 0, 10)
	builder = builder.Range_Int8(5, 0, 10)
	builder = builder.Range_Int16(5, 0, 10)
	builder = builder.Range_Int32(5, 0, 10)
	builder = builder.Range_Int64(5, 0, 10)
	builder = builder.Range_Uint(5, 0, 10)
	builder = builder.Range_Uint8(5, 0, 10)
	builder = builder.Range_Uint16(5, 0, 10)
	builder = builder.Range_Uint32(5, 0, 10)
	return builder.Range_Uint64(5, 0, 10)
}

func assert_builder_holed_ranges(
	builder aver.Assertion_Builder,
) (next aver.Assertion_Builder) {
	builder = builder.Range_Holed_Int(5, 0, 10, 1, 2, 3, 4)
	builder = builder.Range_Holed_Int8(5, 0, 10, 1, 2, 3, 4)
	builder = builder.Range_Holed_Int16(5, 0, 10, 1, 2, 3, 4)
	builder = builder.Range_Holed_Int32(5, 0, 10, 1, 2, 3, 4)
	builder = builder.Range_Holed_Int64(5, 0, 10, 1, 2, 3, 4)
	builder = builder.Range_Holed_Uint(5, 0, 10, 1, 2, 3)
	builder = builder.Range_Holed_Uint8(5, 0, 10, 1, 2, 3)
	builder = builder.Range_Holed_Uint16(5, 0, 10, 1, 2, 3)
	builder = builder.Range_Holed_Uint32(5, 0, 10, 1, 2, 3)
	return builder.Range_Holed_Uint64(5, 0, 10, 1, 2, 3)
}

func assert_builder_enums(
	builder aver.Assertion_Builder,
) (next aver.Assertion_Builder) {
	builder = builder.Enum_Int(1, 1, 2)
	builder = builder.Enum_Int8(1, 1, 2)
	builder = builder.Enum_Int16(1, 1, 2)
	builder = builder.Enum_Int32(1, 1, 2)
	builder = builder.Enum_Int64(1, 1, 2)
	builder = builder.Enum_Uint(1, 1, 2)
	builder = builder.Enum_Uint8(1, 1, 2)
	builder = builder.Enum_Uint16(1, 1, 2)
	builder = builder.Enum_Uint32(1, 1, 2)
	builder = builder.Enum_Uint64(1, 1, 2)
	builder = builder.Enum_3_Int(2, 1, 2, 3)
	builder = builder.Enum_3_Int8(2, 1, 2, 3)
	builder = builder.Enum_3_Int16(2, 1, 2, 3)
	builder = builder.Enum_3_Int32(2, 1, 2, 3)
	builder = builder.Enum_3_Int64(2, 1, 2, 3)
	builder = builder.Enum_3_Uint(2, 1, 2, 3)
	builder = builder.Enum_3_Uint8(2, 1, 2, 3)
	builder = builder.Enum_3_Uint16(2, 1, 2, 3)
	builder = builder.Enum_3_Uint32(2, 1, 2, 3)
	builder = builder.Enum_3_Uint64(2, 1, 2, 3)
	builder = builder.Enum_4_Int(3, 1, 2, 3, 4)
	builder = builder.Enum_4_Int8(3, 1, 2, 3, 4)
	builder = builder.Enum_4_Int16(3, 1, 2, 3, 4)
	builder = builder.Enum_4_Int32(3, 1, 2, 3, 4)
	builder = builder.Enum_4_Int64(3, 1, 2, 3, 4)
	builder = builder.Enum_4_Uint(3, 1, 2, 3, 4)
	builder = builder.Enum_4_Uint8(3, 1, 2, 3, 4)
	builder = builder.Enum_4_Uint16(3, 1, 2, 3, 4)
	builder = builder.Enum_4_Uint32(3, 1, 2, 3, 4)
	return builder.Enum_4_Uint64(3, 1, 2, 3, 4)
}
