// Package driver defines bounded procedure tables implemented by database drivers.
package driver

import (
	"unsafe"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/sim/time"
	"local/james-orcales/shared/slices"
	"local/james-orcales/shared/strings"
	"local/james-orcales/shared/unicode/ucd"
	"local/james-orcales/shared/unicode/utf8"
)

// Status keeps driver outcomes scalar and allocation-free.
type Status uint8

// Status_Invariants closes current driver outcome domain.
func Status_Invariants(value Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_UNSUPPORTED)).
		Ensure()
}

// Validation_Status admits only success or rejected hostile input.
type Validation_Status Status

// Validation_Status_Invariants keeps pure validation outcomes exact.
func Validation_Status_Invariants(
	value Validation_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
		).
		Ensure()
}

// Optional_Status admits success or an absent optional driver capability.
type Optional_Status Status

// Optional_Status_Invariants keeps optional result outcomes exact.
func Optional_Status_Invariants(
	value Optional_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_UNSUPPORTED),
		).
		Ensure()
}

// STATUS_OK reports completed driver work.
const STATUS_OK Status = Status(bits.WORD_8_MINIMUM)

// STATUS_DONE reports normal row exhaustion.
const STATUS_DONE Status = STATUS_OK + 1

// STATUS_INPUT_INVALID rejects hostile caller data before driver call.
const STATUS_INPUT_INVALID Status = STATUS_DONE + 1

// STATUS_STORAGE_INVALID rejects missing or wrongly sized caller storage.
const STATUS_STORAGE_INVALID Status = STATUS_INPUT_INVALID + 1

// STATUS_ERROR reports driver-specific failure without owned diagnostic text.
const STATUS_ERROR Status = STATUS_STORAGE_INVALID + 1

// STATUS_BAD_CONNECTION asks sql pool to discard one connection.
const STATUS_BAD_CONNECTION Status = STATUS_ERROR + 1

// STATUS_UNSUPPORTED reports absent optional driver procedure or result.
const STATUS_UNSUPPORTED Status = STATUS_BAD_CONNECTION + 1

// Boolean keeps database Boolean payload distinct from control flow.
type Boolean bool

// Boolean_Invariants requires both database truth values.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A driver Boolean value is true.").
		Ensure()
}

// Integer is standard signed 64-bit driver value.
type Integer int64

// Integer_Invariants keeps complete standard integer domain.
func Integer_Invariants(value Integer, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Float stores deterministic IEEE 754 binary64 bits without floating arithmetic.
type Float uint64

// Float_Invariants keeps every binary64 encoding, including NaN payloads.
func Float_Invariants(value Float, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// BYTE_COUNT_UNVALIDATED_MAXIMUM admits one rejected byte boundary witness.
const BYTE_COUNT_UNVALIDATED_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM +
	utf8.CHARACTER_SIZE_MINIMUM

// Bytes_Unvalidated admits one rejected boundary witness beyond shared byte limit.
type Bytes_Unvalidated []byte

// Bytes_Unvalidated_Invariants bounds hostile input before validation work.
func Bytes_Unvalidated_Invariants(value Bytes_Unvalidated, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM,
			BYTE_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Bytes is validated borrowed byte value.
type Bytes []byte

// Bytes_Invariants shares repository byte boundary.
func Bytes_Invariants(value Bytes, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, bytes.SLICE_SIZE_MAXIMUM).
		Ensure()
}

// TEXT_SIZE_UNVALIDATED_MAXIMUM admits one rejected text boundary witness.
const TEXT_SIZE_UNVALIDATED_MAXIMUM = strings.TEXT_SIZE_MAXIMUM +
	utf8.CHARACTER_SIZE_MINIMUM

// Text_Unvalidated admits one rejected boundary witness beyond shared text limit.
type Text_Unvalidated string

// Text_Unvalidated_Invariants bounds hostile input before validation work.
func Text_Unvalidated_Invariants(value Text_Unvalidated, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), strings.TEXT_SIZE_MINIMUM,
			TEXT_SIZE_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Text is validated borrowed driver text.
type Text string

// Text_Invariants shares repository text boundary.
func Text_Invariants(value Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Value_Kind selects one member of Value closed union.
type Value_Kind uint8

// Value_Kind_Invariants closes standard driver value kinds.
func Value_Kind_Invariants(value Value_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(VALUE_NULL), uint8(VALUE_TIME)).
		Ensure()
}

// VALUE_NULL carries SQL NULL.
const VALUE_NULL Value_Kind = Value_Kind(bits.WORD_8_MINIMUM)

// VALUE_BOOLEAN carries Boolean.
const VALUE_BOOLEAN Value_Kind = VALUE_NULL + 1

// VALUE_INTEGER carries signed 64-bit integer.
const VALUE_INTEGER Value_Kind = VALUE_BOOLEAN + 1

// VALUE_FLOAT carries IEEE 754 binary64 bits.
const VALUE_FLOAT Value_Kind = VALUE_INTEGER + 1

// VALUE_BYTES carries borrowed bytes.
const VALUE_BYTES Value_Kind = VALUE_FLOAT + 1

// VALUE_TEXT carries borrowed driver-defined text.
const VALUE_TEXT Value_Kind = VALUE_BYTES + 1

// VALUE_TIME carries realtime nanoseconds from Unix epoch.
const VALUE_TIME Value_Kind = VALUE_TEXT + 1

// VALUE_KIND_UNVALIDATED_MAXIMUM admits one rejected kind witness.
const VALUE_KIND_UNVALIDATED_MAXIMUM = VALUE_TIME + 1

// Value_Kind_Unvalidated preserves hostile aggregate input for validation.
type Value_Kind_Unvalidated Value_Kind

// Value_Kind_Unvalidated_Invariants bounds hostile kind before validation.
func Value_Kind_Unvalidated_Invariants(
	value Value_Kind_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(VALUE_NULL), uint8(VALUE_KIND_UNVALIDATED_MAXIMUM),
		).
		Ensure()
}

// VALUE_STORAGE_SIZE keeps Value representation equal to its explicit closed-union fields.
const VALUE_STORAGE_SIZE = unsafe.Sizeof(struct {
	Kind    Value_Kind_Unvalidated
	Boolean Boolean
	Integer Integer
	Float   Float
	Bytes   Bytes
	Text    Text
	Moment  time.Moment
}{})

// Value replaces reflection and any with finite driver value domain.
type Value struct {
	// Kind prevents inactive payload interpretation.
	Kind Value_Kind_Unvalidated
	// Boolean avoids interface boxing for VALUE_BOOLEAN.
	Boolean Boolean
	// Integer avoids interface boxing for VALUE_INTEGER.
	Integer Integer
	// Float preserves exact VALUE_FLOAT binary64 bits.
	Float Float
	// Bytes keeps VALUE_BYTES storage caller-owned.
	Bytes Bytes
	// Text keeps VALUE_TEXT storage caller-owned.
	Text Text
	// Moment keeps realtime type identity for VALUE_TIME.
	Moment time.Moment
}

// Value_Invariants preserves storage while active operations validate payload.
func Value_Invariants(value Value, namespace aver.Namespace) {
	Value_Kind_Unvalidated_Invariants(value.Kind, namespace)
	Boolean_Invariants(value.Boolean, namespace)
	Integer_Invariants(value.Integer, namespace)
	Float_Invariants(value.Float, namespace)
	Bytes_Invariants(value.Bytes, namespace)
	Text_Invariants(value.Text, namespace)
	time.Moment_Invariants(value.Moment, namespace)
	aver.Always(
		unsafe.Sizeof(value) == VALUE_STORAGE_SIZE,
		"Explicit driver value fields preserve closed-union storage size.",
	)
}

// Value_Pointer names caller-owned Value storage.
type Value_Pointer *Value

// Value_Pointer_Invariants admits absent optional storage.
func Value_Pointer_Invariants(value Value_Pointer, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Value_Invariants(*value, namespace)
}

// Value_Null overwrites caller storage so construction needs no heap.
func Value_Null(destination Value_Pointer) {
	Value_Pointer_Invariants(destination, "value_null.destination")
	*destination = Value{Kind: Value_Kind_Unvalidated(VALUE_NULL)}
}

// Value_Of_Boolean overwrites caller storage so construction needs no heap.
func Value_Of_Boolean(destination Value_Pointer, value Boolean) {
	Value_Pointer_Invariants(destination, "value_of_boolean.destination")
	Boolean_Invariants(value, "value_of_boolean.value")
	*destination = Value{
		Kind:    Value_Kind_Unvalidated(VALUE_BOOLEAN),
		Boolean: value,
	}
}

// Value_Of_Integer overwrites caller storage so construction needs no heap.
func Value_Of_Integer(destination Value_Pointer, value Integer) {
	Value_Pointer_Invariants(destination, "value_of_integer.destination")
	Integer_Invariants(value, "value_of_integer.value")
	*destination = Value{
		Kind:    Value_Kind_Unvalidated(VALUE_INTEGER),
		Integer: value,
	}
}

// Value_Of_Float overwrites caller storage so construction needs no heap.
func Value_Of_Float(destination Value_Pointer, value Float) {
	Value_Pointer_Invariants(destination, "value_of_float.destination")
	Float_Invariants(value, "value_of_float.value")
	*destination = Value{
		Kind:  Value_Kind_Unvalidated(VALUE_FLOAT),
		Float: value,
	}
}

// Value_Of_Bytes overwrites caller storage and borrows validated bytes.
func Value_Of_Bytes(
	destination Value_Pointer, value Bytes_Unvalidated,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "value_of_bytes.status") }()
	Value_Pointer_Invariants(destination, "value_of_bytes.destination")
	Bytes_Unvalidated_Invariants(value, "value_of_bytes.value")
	*destination = Value{}
	if len(value) > bytes.SLICE_SIZE_MAXIMUM {
		return Validation_Status(STATUS_INPUT_INVALID)
	}
	*destination = Value{
		Kind:  Value_Kind_Unvalidated(VALUE_BYTES),
		Bytes: Bytes(value),
	}
	return Validation_Status(STATUS_OK)
}

// Value_Of_Text overwrites caller storage and borrows validated text.
func Value_Of_Text(
	destination Value_Pointer, value Text_Unvalidated,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "value_of_text.status") }()
	Value_Pointer_Invariants(destination, "value_of_text.destination")
	Text_Unvalidated_Invariants(value, "value_of_text.value")
	*destination = Value{}
	if len(value) > strings.TEXT_SIZE_MAXIMUM {
		return Validation_Status(STATUS_INPUT_INVALID)
	}
	*destination = Value{
		Kind: Value_Kind_Unvalidated(VALUE_TEXT),
		Text: Text(value),
	}
	return Validation_Status(STATUS_OK)
}

// Value_Of_Time overwrites caller storage so construction needs no heap.
func Value_Of_Time(destination Value_Pointer, value time.Moment) {
	Value_Pointer_Invariants(destination, "value_of_time.destination")
	time.Moment_Invariants(value, "value_of_time.value")
	*destination = Value{
		Kind:   Value_Kind_Unvalidated(VALUE_TIME),
		Moment: value,
	}
}

// Value_Kind_Of rejects hostile tags before exposing closed-union member.
func Value_Kind_Of(value Value) (kind Value_Kind, status Validation_Status) {
	defer func() {
		Value_Kind_Invariants(kind, "value_kind_of.kind")
		Validation_Status_Invariants(status, "value_kind_of.status")
	}()
	Value_Invariants(value, "value_kind_of.value")
	if value.Kind > Value_Kind_Unvalidated(VALUE_TIME) {
		return VALUE_NULL, Validation_Status(STATUS_INPUT_INVALID)
	}
	return Value_Kind(value.Kind), Validation_Status(STATUS_OK)
}

// Value_As_Boolean reads Boolean only when tag agrees.
func Value_As_Boolean(value Value) (result Boolean, status Validation_Status) {
	defer func() {
		Boolean_Invariants(result, "value_as_boolean.result")
		Validation_Status_Invariants(status, "value_as_boolean.status")
	}()
	Value_Invariants(value, "value_as_boolean.value")
	if Value_Kind(value.Kind) != VALUE_BOOLEAN {
		return false, Validation_Status(STATUS_INPUT_INVALID)
	}
	return value.Boolean, Validation_Status(STATUS_OK)
}

// Value_As_Integer reads integer only when tag agrees.
func Value_As_Integer(value Value) (result Integer, status Validation_Status) {
	defer func() {
		Integer_Invariants(result, "value_as_integer.result")
		Validation_Status_Invariants(status, "value_as_integer.status")
	}()
	Value_Invariants(value, "value_as_integer.value")
	if Value_Kind(value.Kind) != VALUE_INTEGER {
		return Integer(bits.WORD_64_MINIMUM), Validation_Status(STATUS_INPUT_INVALID)
	}
	return value.Integer, Validation_Status(STATUS_OK)
}

// Value_As_Float reads binary64 bits only when tag agrees.
func Value_As_Float(value Value) (result Float, status Validation_Status) {
	defer func() {
		Float_Invariants(result, "value_as_float.result")
		Validation_Status_Invariants(status, "value_as_float.status")
	}()
	Value_Invariants(value, "value_as_float.value")
	if Value_Kind(value.Kind) != VALUE_FLOAT {
		return Float(bits.WORD_64_MINIMUM), Validation_Status(STATUS_INPUT_INVALID)
	}
	return value.Float, Validation_Status(STATUS_OK)
}

// Value_As_Bytes returns borrowed bytes only when tag agrees.
func Value_As_Bytes(value Value) (result Bytes, status Validation_Status) {
	defer func() {
		Bytes_Invariants(result, "value_as_bytes.result")
		Validation_Status_Invariants(status, "value_as_bytes.status")
	}()
	Value_Invariants(value, "value_as_bytes.value")
	if Value_Kind(value.Kind) != VALUE_BYTES {
		return nil, Validation_Status(STATUS_INPUT_INVALID)
	}
	return value.Bytes, Validation_Status(STATUS_OK)
}

// Value_As_Text returns borrowed text only when tag agrees.
func Value_As_Text(value Value) (result Text, status Validation_Status) {
	defer func() {
		Text_Invariants(result, "value_as_text.result")
		Validation_Status_Invariants(status, "value_as_text.status")
	}()
	Value_Invariants(value, "value_as_text.value")
	if Value_Kind(value.Kind) != VALUE_TEXT {
		return "", Validation_Status(STATUS_INPUT_INVALID)
	}
	return value.Text, Validation_Status(STATUS_OK)
}

// Value_As_Time reads timestamp only when tag agrees.
func Value_As_Time(value Value) (result time.Moment, status Validation_Status) {
	defer func() {
		time.Moment_Invariants(result, "value_as_time.result")
		Validation_Status_Invariants(status, "value_as_time.status")
	}()
	Value_Invariants(value, "value_as_time.value")
	if Value_Kind(value.Kind) != VALUE_TIME {
		return time.Moment(bits.WORD_64_MINIMUM), Validation_Status(STATUS_INPUT_INVALID)
	}
	return value.Moment, Validation_Status(STATUS_OK)
}

// Query_Unvalidated admits one rejected text-boundary witness.
type Query_Unvalidated string

// Query_Unvalidated_Invariants bounds hostile query before validation.
func Query_Unvalidated_Invariants(value Query_Unvalidated, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), strings.TEXT_SIZE_MINIMUM,
			TEXT_SIZE_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Query is validated driver query text.
type Query string

// Query_Invariants shares repository text boundary.
func Query_Invariants(value Query, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Query_Validate changes raw caller text into bounded Query.
func Query_Validate(
	unvalidated Query_Unvalidated,
) (query Query, status Validation_Status) {
	defer func() {
		Query_Invariants(query, "query_validate.query")
		Validation_Status_Invariants(status, "query_validate.status")
	}()
	Query_Unvalidated_Invariants(unvalidated, "query_validate.unvalidated")
	if len(unvalidated) > strings.TEXT_SIZE_MAXIMUM {
		return "", Validation_Status(STATUS_INPUT_INVALID)
	}
	return Query(unvalidated), Validation_Status(STATUS_OK)
}

// Data_Source_Unvalidated admits one rejected text-boundary witness.
type Data_Source_Unvalidated string

// Data_Source_Unvalidated_Invariants bounds hostile driver text before validation.
func Data_Source_Unvalidated_Invariants(
	value Data_Source_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), strings.TEXT_SIZE_MINIMUM,
			TEXT_SIZE_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Data_Source is validated driver-specific connection text.
type Data_Source string

// Data_Source_Invariants shares repository text boundary.
func Data_Source_Invariants(value Data_Source, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Data_Source_Validate changes raw caller text into bounded Data_Source.
func Data_Source_Validate(
	unvalidated Data_Source_Unvalidated,
) (data_source Data_Source, status Validation_Status) {
	defer func() {
		Data_Source_Invariants(data_source, "data_source_validate.data_source")
		Validation_Status_Invariants(status, "data_source_validate.status")
	}()
	Data_Source_Unvalidated_Invariants(unvalidated, "data_source_validate.unvalidated")
	if len(unvalidated) > strings.TEXT_SIZE_MAXIMUM {
		return "", Validation_Status(STATUS_INPUT_INVALID)
	}
	return Data_Source(unvalidated), Validation_Status(STATUS_OK)
}

// Argument_Name_Unvalidated is raw standard named-argument name.
type Argument_Name_Unvalidated string

// Argument_Name_Unvalidated_Invariants bounds hostile name before letter scan.
func Argument_Name_Unvalidated_Invariants(
	value Argument_Name_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), strings.TEXT_SIZE_MINIMUM,
			TEXT_SIZE_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Argument_Name is validated optional parameter name.
type Argument_Name string

// Argument_Name_Invariants shares repository text boundary.
func Argument_Name_Invariants(value Argument_Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// ARGUMENT_COUNT_MAXIMUM shares bounded collection domain.
const ARGUMENT_COUNT_MAXIMUM = slices.SLICE_COUNT_MAXIMUM

// ARGUMENT_COUNT_UNVALIDATED_MAXIMUM admits one rejected ordinal witness.
const ARGUMENT_COUNT_UNVALIDATED_MAXIMUM = ARGUMENT_COUNT_MAXIMUM +
	utf8.CHARACTER_SIZE_MINIMUM

// Argument_Ordinal_Unvalidated admits zero and one rejected high boundary.
type Argument_Ordinal_Unvalidated int

// Argument_Ordinal_Unvalidated_Invariants bounds hostile ordinal before validation.
func Argument_Ordinal_Unvalidated_Invariants(
	value Argument_Ordinal_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), slices.COUNT_MINIMUM,
			ARGUMENT_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Argument_Ordinal is one-based for valid values and zero for invalid zero struct.
type Argument_Ordinal int

// Argument_Ordinal_Invariants includes zero invalid sentinel and bounded positions.
func Argument_Ordinal_Invariants(value Argument_Ordinal, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), slices.COUNT_MINIMUM, ARGUMENT_COUNT_MAXIMUM).
		Ensure()
}

// Named_Value_Validity marks constructor-approved name, ordinal, and value.
type Named_Value_Validity bool

// Named_Value_Validity_Invariants requires valid and rejected argument paths.
func Named_Value_Validity_Invariants(
	value Named_Value_Validity, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A named value passed bounded validation.").
		Ensure()
}

// NAMED_VALUE_STORAGE_SIZE keeps argument representation equal to explicit fields.
const NAMED_VALUE_STORAGE_SIZE = unsafe.Sizeof(struct {
	Name     Argument_Name
	Ordinal  Argument_Ordinal
	Value    Value
	Validity Named_Value_Validity
}{})

// Named_Value pairs typed value with standard named or positional identity.
type Named_Value struct {
	// Name stays empty for positional argument.
	Name Argument_Name
	// Ordinal preserves standard one-based identity.
	Ordinal Argument_Ordinal
	// Value prevents untyped driver payloads.
	Value Value
	// Validity prevents zero struct from becoming accepted argument.
	Validity Named_Value_Validity
}

// Named_Value_Invariants preserves storage while operations validate active fields.
func Named_Value_Invariants(value Named_Value, namespace aver.Namespace) {
	Argument_Name_Invariants(value.Name, namespace)
	Argument_Ordinal_Invariants(value.Ordinal, namespace)
	Value_Invariants(value.Value, namespace)
	Named_Value_Validity_Invariants(value.Validity, namespace)
	aver.Always(
		unsafe.Sizeof(value) == NAMED_VALUE_STORAGE_SIZE,
		"Explicit named value fields preserve argument storage size.",
	)
}

// Named_Value_Pointer names caller-owned Named_Value storage.
type Named_Value_Pointer *Named_Value

// Named_Value_Pointer_Invariants admits absent optional storage.
func Named_Value_Pointer_Invariants(
	value Named_Value_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Named_Value_Invariants(*value, namespace)
}

// Named_Value_Of overwrites caller storage only after every argument validates.
func Named_Value_Of(
	destination Named_Value_Pointer, name Argument_Name_Unvalidated,
	ordinal Argument_Ordinal_Unvalidated, value Value,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "named_value_of.status") }()
	Named_Value_Pointer_Invariants(destination, "named_value_of.destination")
	Argument_Name_Unvalidated_Invariants(name, "named_value_of.name")
	Argument_Ordinal_Unvalidated_Invariants(ordinal, "named_value_of.ordinal")
	Value_Invariants(value, "named_value_of.value")
	*destination = Named_Value{}
	if ordinal < Argument_Ordinal_Unvalidated(utf8.CHARACTER_SIZE_MINIMUM) {
		return Validation_Status(STATUS_INPUT_INVALID)
	}
	if ordinal > ARGUMENT_COUNT_MAXIMUM {
		return Validation_Status(STATUS_INPUT_INVALID)
	}
	if len(name) > strings.TEXT_SIZE_MAXIMUM {
		return Validation_Status(STATUS_INPUT_INVALID)
	}
	if name != "" {
		character, _ := utf8.Decode_Character_Text(utf8.Text(name))
		if !bool(ucd.Is_Letter(ucd.Character(character))) {
			return Validation_Status(STATUS_INPUT_INVALID)
		}
	}
	if value_validate(value) != Validation_Status(STATUS_OK) {
		return Validation_Status(STATUS_INPUT_INVALID)
	}
	*destination = Named_Value{
		Name:     Argument_Name(name),
		Ordinal:  Argument_Ordinal(ordinal),
		Value:    value,
		Validity: true,
	}
	return Validation_Status(STATUS_OK)
}

// Named_Value_Name reports optional parameter name.
func Named_Value_Name(value Named_Value) (name Argument_Name) {
	defer func() { Argument_Name_Invariants(name, "named_value_name.name") }()
	Named_Value_Invariants(value, "named_value_name.value")
	return value.Name
}

// Named_Value_Ordinal reports one-based parameter position.
func Named_Value_Ordinal(value Named_Value) (ordinal Argument_Ordinal) {
	defer func() { Argument_Ordinal_Invariants(ordinal, "named_value_ordinal.ordinal") }()
	Named_Value_Invariants(value, "named_value_ordinal.value")
	return value.Ordinal
}

// Named_Value_Value reports typed parameter value.
func Named_Value_Value(value Named_Value) (result Value) {
	defer func() { Value_Invariants(result, "named_value_value.result") }()
	Named_Value_Invariants(value, "named_value_value.value")
	return value.Value
}

// Arguments is bounded borrowed parameter storage.
type Arguments []Named_Value

// Arguments_Invariants bounds parameter storage.
func Arguments_Invariants(value Arguments, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, ARGUMENT_COUNT_MAXIMUM).
		Ensure()
}

// REQUEST_STORAGE_SIZE keeps request representation equal to query and argument views.
const REQUEST_STORAGE_SIZE = unsafe.Sizeof(struct {
	Query     Query
	Arguments Arguments
}{})

// Request is bounded query and borrowed typed arguments.
type Request struct {
	// Query keeps driver text validated before dispatch.
	Query Query
	// Arguments keeps parameter storage caller-owned.
	Arguments Arguments
}

// Request_Invariants preserves storage while Request_Validate checks borrowed content.
func Request_Invariants(value Request, namespace aver.Namespace) {
	Query_Invariants(value.Query, namespace)
	Arguments_Invariants(value.Arguments, namespace)
	aver.Always(
		unsafe.Sizeof(value) == REQUEST_STORAGE_SIZE,
		"Explicit request fields preserve query and argument storage size.",
	)
}

// Request_Of validates and stores one bounded query with caller arguments.
func Request_Of(
	query Query, arguments Arguments,
) (request Request, status Validation_Status) {
	defer func() {
		Request_Invariants(request, "request_of.request")
		Validation_Status_Invariants(status, "request_of.status")
	}()
	Query_Invariants(query, "request_of.query")
	Arguments_Invariants(arguments, "request_of.arguments")
	request = Request{
		Query:     query,
		Arguments: arguments,
	}
	status = Request_Validate(request)
	return request, status
}

// Request_Validate checks argument validity and exact standard ordinal sequence.
func Request_Validate(request Request) (status Validation_Status) {
	defer func() {
		Validation_Status_Invariants(status, "request_validate.status")
	}()
	Request_Invariants(request, "request_validate.request")
	arguments := request.Arguments
	if len(arguments) > ARGUMENT_COUNT_MAXIMUM {
		return Validation_Status(STATUS_INPUT_INVALID)
	}
	for index := range arguments {
		argument := arguments[index]
		if !bool(argument.Validity) {
			return Validation_Status(STATUS_INPUT_INVALID)
		}
		if int(argument.Ordinal) != index+utf8.CHARACTER_SIZE_MINIMUM {
			return Validation_Status(STATUS_INPUT_INVALID)
		}
		if value_validate(argument.Value) !=
			Validation_Status(STATUS_OK) {
			return Validation_Status(STATUS_INPUT_INVALID)
		}
	}
	return Validation_Status(STATUS_OK)
}

// Isolation_Level selects one standard transaction isolation contract.
type Isolation_Level uint8

// Isolation_Level_Invariants closes standard isolation domain.
func Isolation_Level_Invariants(value Isolation_Level, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(ISOLATION_DEFAULT), uint8(ISOLATION_LINEARIZABLE),
		).
		Ensure()
}

// ISOLATION_DEFAULT lets the driver select its default isolation.
const ISOLATION_DEFAULT Isolation_Level = Isolation_Level(bits.WORD_8_MINIMUM)

// ISOLATION_READ_UNCOMMITTED permits dirty reads.
const ISOLATION_READ_UNCOMMITTED Isolation_Level = ISOLATION_DEFAULT + 1

// ISOLATION_READ_COMMITTED rejects dirty reads.
const ISOLATION_READ_COMMITTED Isolation_Level = ISOLATION_READ_UNCOMMITTED + 1

// ISOLATION_WRITE_COMMITTED delays write visibility until commit.
const ISOLATION_WRITE_COMMITTED Isolation_Level = ISOLATION_READ_COMMITTED + 1

// ISOLATION_REPEATABLE_READ keeps read values stable through transaction.
const ISOLATION_REPEATABLE_READ Isolation_Level = ISOLATION_WRITE_COMMITTED + 1

// ISOLATION_SNAPSHOT reads one committed snapshot.
const ISOLATION_SNAPSHOT Isolation_Level = ISOLATION_REPEATABLE_READ + 1

// ISOLATION_SERIALIZABLE orders transactions as serial execution.
const ISOLATION_SERIALIZABLE Isolation_Level = ISOLATION_SNAPSHOT + 1

// ISOLATION_LINEARIZABLE preserves realtime ordering.
const ISOLATION_LINEARIZABLE Isolation_Level = ISOLATION_SERIALIZABLE + 1

// Transaction_Read_Only distinguishes read-only transaction request.
type Transaction_Read_Only bool

// Transaction_Read_Only_Invariants requires both transaction access modes.
func Transaction_Read_Only_Invariants(
	value Transaction_Read_Only, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A transaction is read-only.").
		Ensure()
}

// Transaction_Options carries standard isolation and read-only request.
type Transaction_Options struct {
	// Isolation_Level prevents driver-specific isolation values.
	Isolation_Level Isolation_Level
	// Read_Only preserves access mode without interface boxing.
	Read_Only Transaction_Read_Only
}

// Transaction_Options_Invariants composes standard transaction controls.
func Transaction_Options_Invariants(
	value Transaction_Options, namespace aver.Namespace,
) {
	Isolation_Level_Invariants(value.Isolation_Level, namespace)
	Transaction_Read_Only_Invariants(value.Read_Only, namespace)
}

// Transaction_Options_Of constructs one bounded standard transaction request.
func Transaction_Options_Of(
	isolation Isolation_Level, read_only Transaction_Read_Only,
) (options Transaction_Options) {
	defer func() {
		Transaction_Options_Invariants(options, "transaction_options_of.options")
	}()
	Isolation_Level_Invariants(isolation, "transaction_options_of.isolation")
	Transaction_Read_Only_Invariants(read_only, "transaction_options_of.read_only")
	return Transaction_Options{
		Isolation_Level: isolation,
		Read_Only:       read_only,
	}
}

// Last_Insert_Identifier is optional standard execution counter.
type Last_Insert_Identifier int64

// Last_Insert_Identifier_Invariants keeps complete driver counter domain.
func Last_Insert_Identifier_Invariants(
	value Last_Insert_Identifier, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Rows_Affected is optional standard execution counter.
type Rows_Affected int64

// Rows_Affected_Invariants keeps complete driver counter domain.
func Rows_Affected_Invariants(value Rows_Affected, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Last_Insert_Identifier_Validity marks reported generated identity.
type Last_Insert_Identifier_Validity bool

// Last_Insert_Identifier_Validity_Invariants requires present and absent paths.
func Last_Insert_Identifier_Validity_Invariants(
	value Last_Insert_Identifier_Validity, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A result reports last insert identifier.").
		Ensure()
}

// Rows_Affected_Validity marks reported affected count.
type Rows_Affected_Validity bool

// Rows_Affected_Validity_Invariants requires present and absent paths.
func Rows_Affected_Validity_Invariants(
	value Rows_Affected_Validity, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A result reports rows affected.").
		Ensure()
}

// RESULT_STORAGE_SIZE keeps optional counters equal to explicit value and validity fields.
const RESULT_STORAGE_SIZE = unsafe.Sizeof(struct {
	Last_Insert_Identifier          Last_Insert_Identifier
	Rows_Affected                   Rows_Affected
	Last_Insert_Identifier_Validity Last_Insert_Identifier_Validity
	Rows_Affected_Validity          Rows_Affected_Validity
}{})

// Result holds optional standard execution counters without interface box.
type Result struct {
	// Last_Insert_Identifier remains meaningful only when matching validity is true.
	Last_Insert_Identifier Last_Insert_Identifier
	// Rows_Affected remains meaningful only when matching validity is true.
	Rows_Affected Rows_Affected
	// Last_Insert_Identifier_Validity distinguishes unsupported from zero identifier.
	Last_Insert_Identifier_Validity Last_Insert_Identifier_Validity
	// Rows_Affected_Validity distinguishes unsupported from zero count.
	Rows_Affected_Validity Rows_Affected_Validity
}

// Result_Invariants preserves storage while access validates active member.
func Result_Invariants(value Result, namespace aver.Namespace) {
	Last_Insert_Identifier_Invariants(value.Last_Insert_Identifier, namespace)
	Rows_Affected_Invariants(value.Rows_Affected, namespace)
	Last_Insert_Identifier_Validity_Invariants(
		value.Last_Insert_Identifier_Validity, namespace,
	)
	Rows_Affected_Validity_Invariants(value.Rows_Affected_Validity, namespace)
	aver.Always(
		unsafe.Sizeof(value) == RESULT_STORAGE_SIZE,
		"Explicit result fields preserve optional counter storage size.",
	)
}

// Result_Pointer names caller-owned Result storage.
type Result_Pointer *Result

// Result_Pointer_Invariants admits absent optional storage.
func Result_Pointer_Invariants(value Result_Pointer, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Result_Invariants(*value, namespace)
}

// Result_Set_Last_Insert_Identifier lets driver report generated identity.
func Result_Set_Last_Insert_Identifier(result Result_Pointer, value Last_Insert_Identifier) {
	Result_Pointer_Invariants(result, "result_set_last_insert_identifier.result")
	Last_Insert_Identifier_Invariants(value, "result_set_last_insert_identifier.value")
	result.Last_Insert_Identifier = value
	result.Last_Insert_Identifier_Validity = true
}

// Result_Set_Rows_Affected lets driver report affected count.
func Result_Set_Rows_Affected(result Result_Pointer, value Rows_Affected) {
	Result_Pointer_Invariants(result, "result_set_rows_affected.result")
	Rows_Affected_Invariants(value, "result_set_rows_affected.value")
	result.Rows_Affected = value
	result.Rows_Affected_Validity = true
}

// Result_Last_Insert_Identifier reads generated identity when reported.
func Result_Last_Insert_Identifier(
	result Result_Pointer,
) (value Last_Insert_Identifier, status Optional_Status) {
	defer func() {
		Last_Insert_Identifier_Invariants(value, "result_last_insert_identifier.value")
		Optional_Status_Invariants(status, "result_last_insert_identifier.status")
	}()
	Result_Pointer_Invariants(result, "result_last_insert_identifier.result")
	Last_Insert_Identifier_Validity_Invariants(
		result.Last_Insert_Identifier_Validity,
		"result_last_insert_identifier.valid",
	)
	if !bool(result.Last_Insert_Identifier_Validity) {
		return Last_Insert_Identifier(bits.WORD_64_MINIMUM),
			Optional_Status(STATUS_UNSUPPORTED)
	}
	return result.Last_Insert_Identifier, Optional_Status(STATUS_OK)
}

// Result_Rows_Affected reads affected count when reported.
func Result_Rows_Affected(
	result Result_Pointer,
) (value Rows_Affected, status Optional_Status) {
	defer func() {
		Rows_Affected_Invariants(value, "result_rows_affected.value")
		Optional_Status_Invariants(status, "result_rows_affected.status")
	}()
	Result_Pointer_Invariants(result, "result_rows_affected.result")
	Rows_Affected_Validity_Invariants(
		result.Rows_Affected_Validity,
		"result_rows_affected.valid",
	)
	if !bool(result.Rows_Affected_Validity) {
		return Rows_Affected(bits.WORD_64_MINIMUM), Optional_Status(STATUS_UNSUPPORTED)
	}
	return result.Rows_Affected, Optional_Status(STATUS_OK)
}

func value_validate(value Value) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "value_validate.status") }()
	Value_Invariants(value, "value_valid.value")
	switch Value_Kind(value.Kind) {
	case VALUE_NULL, VALUE_BOOLEAN, VALUE_INTEGER, VALUE_FLOAT, VALUE_TIME:
		return Validation_Status(STATUS_OK)
	case VALUE_BYTES:
		if len(value.Bytes) > bytes.SLICE_SIZE_MAXIMUM {
			return Validation_Status(STATUS_INPUT_INVALID)
		}
		return Validation_Status(STATUS_OK)
	case VALUE_TEXT:
		if len(value.Text) > strings.TEXT_SIZE_MAXIMUM {
			return Validation_Status(STATUS_INPUT_INVALID)
		}
		return Validation_Status(STATUS_OK)
	default:
		return Validation_Status(STATUS_INPUT_INVALID)
	}
}

// STATEMENT_ARGUMENT_COUNT_UNKNOWN lets a driver validate count itself.
const STATEMENT_ARGUMENT_COUNT_UNKNOWN = -utf8.CHARACTER_SIZE_MINIMUM

// Statement_Argument_Count is exact parameter count or unknown sentinel.
type Statement_Argument_Count int

// Statement_Argument_Count_Invariants bounds known count and unknown sentinel.
func Statement_Argument_Count_Invariants(
	value Statement_Argument_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), STATEMENT_ARGUMENT_COUNT_UNKNOWN, ARGUMENT_COUNT_MAXIMUM,
		).
		Ensure()
}

// Statement_Exec_Procedure executes one prepared statement.
type Statement_Exec_Procedure func(
	state unsafe.Pointer, arguments Arguments, result Result_Pointer,
) (status Status)

// Statement_Query_Procedure opens rows from one prepared statement.
type Statement_Query_Procedure func(
	state unsafe.Pointer, arguments Arguments, rows Rows_Pointer,
) (status Status)

// Statement_Close_Procedure releases one prepared statement.
type Statement_Close_Procedure func(state unsafe.Pointer) (status Status)

// STATEMENT_STORAGE_SIZE keeps prepared statement representation equal to explicit fields.
const STATEMENT_STORAGE_SIZE = unsafe.Sizeof(struct {
	State           unsafe.Pointer
	Argument_Count  Statement_Argument_Count
	Exec_Procedure  Statement_Exec_Procedure
	Query_Procedure Statement_Query_Procedure
	Close_Procedure Statement_Close_Procedure
}{})

// Statement is one prepared driver statement procedure table.
type Statement struct {
	// State stays opaque while driver owns prepared statement.
	State unsafe.Pointer
	// Argument_Count prevents dispatch with wrong argument width.
	Argument_Count Statement_Argument_Count
	// Exec_Procedure keeps driver execution injected.
	Exec_Procedure Statement_Exec_Procedure
	// Query_Procedure keeps driver query injection allocation-free.
	Query_Procedure Statement_Query_Procedure
	// Close_Procedure keeps resource ownership explicit.
	Close_Procedure Statement_Close_Procedure
}

// Statement_Invariants preserves storage while statement_live validates active procedures.
func Statement_Invariants(value Statement, namespace aver.Namespace) {
	Statement_Argument_Count_Invariants(value.Argument_Count, namespace)
	aver.Always(
		unsafe.Sizeof(value) == STATEMENT_STORAGE_SIZE,
		"Explicit statement fields preserve procedure table storage size.",
	)
}

// Statement_Pointer names caller-owned Statement storage.
type Statement_Pointer *Statement

// Statement_Pointer_Invariants admits absent optional storage.
func Statement_Pointer_Invariants(
	value Statement_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Statement_Invariants(*value, namespace)
}

// Transaction_Commit_Procedure makes transaction changes durable.
type Transaction_Commit_Procedure func(state unsafe.Pointer) (status Status)

// Transaction_Rollback_Procedure abandons transaction changes.
type Transaction_Rollback_Procedure func(state unsafe.Pointer) (status Status)

// Transaction is one live driver transaction terminal procedure table.
type Transaction struct {
	// State stays opaque while driver owns transaction.
	State unsafe.Pointer
	// Commit_Procedure keeps terminal durability operation injected.
	Commit_Procedure Transaction_Commit_Procedure
	// Rollback_Procedure keeps terminal abandonment operation injected.
	Rollback_Procedure Transaction_Rollback_Procedure
}

// Transaction_Invariants admits zero storage and live transactions.
func Transaction_Invariants(subject Transaction, namespace aver.Namespace) {
	aver.Sometimes(
		subject.Commit_Procedure != nil,
		"Driver transaction storage is live.",
	)
}

// Transaction_Pointer names caller-owned Transaction storage.
type Transaction_Pointer *Transaction

// Transaction_Pointer_Invariants admits absent optional storage.
func Transaction_Pointer_Invariants(
	value Transaction_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Transaction_Invariants(*value, namespace)
}

// Driver_Connect_Procedure opens one driver connection.
type Driver_Connect_Procedure func(
	state unsafe.Pointer, data_source Data_Source, connection Connection_Pointer,
) (status Status)

// DRIVER_STORAGE_SIZE keeps injected driver representation equal to state and procedure.
const DRIVER_STORAGE_SIZE = unsafe.Sizeof(struct {
	State             unsafe.Pointer
	Connect_Procedure Driver_Connect_Procedure
}{})

// Driver injects connection creation over explicit caller-owned state.
type Driver struct {
	// State belongs to driver root and survives every connection.
	State unsafe.Pointer
	// Connect_Procedure fills caller connection storage.
	Connect_Procedure Driver_Connect_Procedure
}

// Driver_Invariants preserves dependency storage while Connect validates live procedure.
func Driver_Invariants(value Driver, _ aver.Namespace) {
	aver.Always(
		unsafe.Sizeof(value) == DRIVER_STORAGE_SIZE,
		"Explicit driver fields preserve injected dependency storage size.",
	)
}

// Connection_Close_Procedure releases one driver connection.
type Connection_Close_Procedure func(state unsafe.Pointer) (status Status)

// Connection_Probe_Procedure checks one driver connection.
type Connection_Probe_Procedure func(state unsafe.Pointer) (status Status)

// Connection_Exec_Procedure executes one request.
type Connection_Exec_Procedure func(
	state unsafe.Pointer, request Request, result Result_Pointer,
) (status Status)

// Connection_Query_Procedure opens rows for one request.
type Connection_Query_Procedure func(
	state unsafe.Pointer, request Request, rows Rows_Pointer,
) (status Status)

// Connection_Prepare_Procedure creates one prepared statement.
type Connection_Prepare_Procedure func(
	state unsafe.Pointer, query Query, statement Statement_Pointer,
) (status Status)

// Connection_Begin_Procedure starts one transaction.
type Connection_Begin_Procedure func(
	state unsafe.Pointer, options Transaction_Options,
	transaction Transaction_Pointer,
) (status Status)

// Connection is one live driver connection procedure table.
type Connection struct {
	// State stays opaque while driver owns connection.
	State unsafe.Pointer
	// Close_Procedure keeps resource ownership explicit.
	Close_Procedure Connection_Close_Procedure
	// Probe_Procedure remains nil when driver lacks optional probe support.
	Probe_Procedure Connection_Probe_Procedure
	// Exec_Procedure keeps driver execution injected.
	Exec_Procedure Connection_Exec_Procedure
	// Query_Procedure keeps driver query injection allocation-free.
	Query_Procedure Connection_Query_Procedure
	// Prepare_Procedure keeps statement storage caller-owned.
	Prepare_Procedure Connection_Prepare_Procedure
	// Begin_Procedure keeps transaction storage caller-owned.
	Begin_Procedure Connection_Begin_Procedure
}

// Connection_Invariants admits zero storage and live connections.
func Connection_Invariants(subject Connection, namespace aver.Namespace) {
	aver.Sometimes(
		subject.Close_Procedure != nil,
		"Driver connection storage is live.",
	)
}

// Connection_Pointer names caller-owned Connection storage.
type Connection_Pointer *Connection

// Connection_Pointer_Invariants admits absent optional storage.
func Connection_Pointer_Invariants(
	value Connection_Pointer, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Connection_Invariants(*value, namespace)
}

// Column_Count is exact width of one result row.
type Column_Count int

// Column_Count_Invariants shares bounded collection domain.
func Column_Count_Invariants(value Column_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), slices.COUNT_MINIMUM, slices.SLICE_COUNT_MAXIMUM).
		Ensure()
}

// Values is caller-owned row destination storage.
type Values []Value

// Values_Invariants bounds row destination storage.
func Values_Invariants(value Values, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.SLICE_COUNT_MAXIMUM).
		Ensure()
}

// Rows_Next_Procedure writes one result row.
type Rows_Next_Procedure func(state unsafe.Pointer, values Values) (status Status)

// Rows_Close_Procedure releases one row cursor.
type Rows_Close_Procedure func(state unsafe.Pointer) (status Status)

// ROWS_STORAGE_SIZE keeps cursor representation equal to explicit fields.
const ROWS_STORAGE_SIZE = unsafe.Sizeof(struct {
	State           unsafe.Pointer
	Column_Count    Column_Count
	Next_Procedure  Rows_Next_Procedure
	Close_Procedure Rows_Close_Procedure
}{})

// Rows is one live driver cursor procedure table.
type Rows struct {
	// State stays opaque while driver owns cursor.
	State unsafe.Pointer
	// Column_Count prevents writes into wrong destination width.
	Column_Count Column_Count
	// Next_Procedure keeps cursor advancement injected.
	Next_Procedure Rows_Next_Procedure
	// Close_Procedure keeps cursor ownership explicit.
	Close_Procedure Rows_Close_Procedure
}

// Rows_Invariants preserves cursor storage while rows_live validates active row width.
func Rows_Invariants(value Rows, namespace aver.Namespace) {
	Column_Count_Invariants(value.Column_Count, namespace)
	aver.Always(
		unsafe.Sizeof(value) == ROWS_STORAGE_SIZE,
		"Explicit rows fields preserve cursor storage size.",
	)
}

// Rows_Pointer names caller-owned Rows storage.
type Rows_Pointer *Rows

// Rows_Pointer_Invariants admits absent optional storage.
func Rows_Pointer_Invariants(value Rows_Pointer, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Rows_Invariants(*value, namespace)
}

// Connect asks injected driver to fill one caller-owned live connection.
func Connect(
	driver Driver, data_source Data_Source, destination Connection_Pointer,
) (status Status) {
	defer func() { Status_Invariants(status, "connect.status") }()
	Driver_Invariants(driver, "connect.driver")
	Data_Source_Invariants(data_source, "connect.data_source")
	Connection_Pointer_Invariants(destination, "connect.destination")
	driver_live(driver)
	*destination = Connection{}
	status = driver.Connect_Procedure(driver.State, data_source, destination)
	Status_Invariants(status, "connect.driver_status")
	if status != STATUS_OK {
		return status
	}
	connection_live(destination)
	return STATUS_OK
}

// Connection_Probe checks liveness when driver exposes optional procedure.
func Connection_Probe(connection Connection_Pointer) (status Status) {
	defer func() { Status_Invariants(status, "connection_probe.status") }()
	Connection_Pointer_Invariants(connection, "connection_probe.connection")
	connection_live(connection)
	if connection.Probe_Procedure == nil {
		return STATUS_OK
	}
	status = connection.Probe_Procedure(connection.State)
	Status_Invariants(status, "connection_probe.driver_status")
	return status
}

// Connection_Exec validates request before driver execution.
func Connection_Exec(
	connection Connection_Pointer, request Request, result Result_Pointer,
) (status Status) {
	defer func() { Status_Invariants(status, "connection_exec.status") }()
	Connection_Pointer_Invariants(connection, "connection_exec.connection")
	Request_Invariants(request, "connection_exec.request")
	Result_Pointer_Invariants(result, "connection_exec.result")
	connection_live(connection)
	status = Status(Request_Validate(request))
	if status != STATUS_OK {
		return status
	}
	*result = Result{}
	if connection.Exec_Procedure == nil {
		return STATUS_UNSUPPORTED
	}
	status = connection.Exec_Procedure(
		connection.State, request, result,
	)
	Status_Invariants(status, "connection_exec.driver_status")
	return status
}

// Connection_Query validates request and caller rows before driver call.
func Connection_Query(
	connection Connection_Pointer, request Request, rows Rows_Pointer,
) (status Status) {
	defer func() { Status_Invariants(status, "connection_query.status") }()
	Connection_Pointer_Invariants(connection, "connection_query.connection")
	Request_Invariants(request, "connection_query.request")
	Rows_Pointer_Invariants(rows, "connection_query.rows")
	connection_live(connection)
	status = Status(Request_Validate(request))
	if status != STATUS_OK {
		return status
	}
	*rows = Rows{}
	if connection.Query_Procedure == nil {
		return STATUS_UNSUPPORTED
	}
	status = connection.Query_Procedure(
		connection.State, request, rows,
	)
	Status_Invariants(status, "connection_query.driver_status")
	if status != STATUS_OK {
		return status
	}
	rows_live(rows)
	return STATUS_OK
}

// Connection_Prepare asks driver to fill one caller-owned statement.
func Connection_Prepare(
	connection Connection_Pointer, query Query, statement Statement_Pointer,
) (status Status) {
	defer func() { Status_Invariants(status, "connection_prepare.status") }()
	Connection_Pointer_Invariants(connection, "connection_prepare.connection")
	Query_Invariants(query, "connection_prepare.query")
	Statement_Pointer_Invariants(statement, "connection_prepare.statement")
	connection_live(connection)
	if connection.Prepare_Procedure == nil {
		return STATUS_UNSUPPORTED
	}
	*statement = Statement{}
	status = connection.Prepare_Procedure(
		connection.State, query, statement,
	)
	Status_Invariants(status, "connection_prepare.driver_status")
	if status != STATUS_OK {
		return status
	}
	statement_live(statement)
	return STATUS_OK
}

// Connection_Begin asks driver to fill one caller-owned transaction.
func Connection_Begin(
	connection Connection_Pointer, options Transaction_Options,
	transaction Transaction_Pointer,
) (status Status) {
	defer func() { Status_Invariants(status, "connection_begin.status") }()
	Connection_Pointer_Invariants(connection, "connection_begin.connection")
	Transaction_Options_Invariants(options, "connection_begin.options")
	Transaction_Pointer_Invariants(transaction, "connection_begin.transaction")
	connection_live(connection)
	transaction_options_live(options)
	if connection.Begin_Procedure == nil {
		return STATUS_UNSUPPORTED
	}
	*transaction = Transaction{}
	status = connection.Begin_Procedure(
		connection.State, options, transaction,
	)
	Status_Invariants(status, "connection_begin.driver_status")
	if status != STATUS_OK {
		return status
	}
	transaction_live(transaction)
	return STATUS_OK
}

// Connection_Close releases one live driver resource.
func Connection_Close(connection Connection_Pointer) (status Status) {
	defer func() { Status_Invariants(status, "connection_close.status") }()
	Connection_Pointer_Invariants(connection, "connection_close.connection")
	connection_live(connection)
	status = connection.Close_Procedure(connection.State)
	Status_Invariants(status, "connection_close.driver_status")
	return status
}

// Rows_Next writes one exact-width row into caller storage.
func Rows_Next(rows Rows_Pointer, destination Values) (status Status) {
	defer func() { Status_Invariants(status, "rows_next.status") }()
	Rows_Pointer_Invariants(rows, "rows_next.rows")
	Values_Invariants(destination, "rows_next.destination")
	rows_live(rows)
	if len(destination) != int(rows.Column_Count) {
		return STATUS_STORAGE_INVALID
	}
	status = rows.Next_Procedure(rows.State, destination)
	Status_Invariants(status, "rows_next.driver_status")
	if status != STATUS_OK {
		return status
	}
	for index := range destination {
		Value_Invariants(destination[index], "rows_next.value")
		if value_validate(destination[index]) != Validation_Status(STATUS_OK) {
			return STATUS_ERROR
		}
	}
	return STATUS_OK
}

// Rows_Close releases one live driver cursor.
func Rows_Close(rows Rows_Pointer) (status Status) {
	defer func() { Status_Invariants(status, "rows_close.status") }()
	Rows_Pointer_Invariants(rows, "rows_close.rows")
	rows_live(rows)
	status = rows.Close_Procedure(rows.State)
	Status_Invariants(status, "rows_close.driver_status")
	return status
}

// Statement_Exec validates exact argument count before prepared execution.
func Statement_Exec(
	statement Statement_Pointer, arguments Arguments, result Result_Pointer,
) (status Status) {
	defer func() { Status_Invariants(status, "statement_exec.status") }()
	Statement_Pointer_Invariants(statement, "statement_exec.statement")
	Arguments_Invariants(arguments, "statement_exec.arguments")
	Result_Pointer_Invariants(result, "statement_exec.result")
	statement_live(statement)
	if arguments_validate(arguments) != Validation_Status(STATUS_OK) {
		return STATUS_INPUT_INVALID
	}
	if statement_argument_count_validate(statement, arguments) !=
		Validation_Status(STATUS_OK) {
		return STATUS_INPUT_INVALID
	}
	*result = Result{}
	status = statement.Exec_Procedure(
		statement.State, arguments, result,
	)
	Status_Invariants(status, "statement_exec.driver_status")
	return status
}

// Statement_Query validates exact argument count before opening prepared rows.
func Statement_Query(
	statement Statement_Pointer, arguments Arguments, rows Rows_Pointer,
) (status Status) {
	defer func() { Status_Invariants(status, "statement_query.status") }()
	Statement_Pointer_Invariants(statement, "statement_query.statement")
	Arguments_Invariants(arguments, "statement_query.arguments")
	Rows_Pointer_Invariants(rows, "statement_query.rows")
	statement_live(statement)
	if arguments_validate(arguments) != Validation_Status(STATUS_OK) {
		return STATUS_INPUT_INVALID
	}
	if statement_argument_count_validate(statement, arguments) !=
		Validation_Status(STATUS_OK) {
		return STATUS_INPUT_INVALID
	}
	*rows = Rows{}
	status = statement.Query_Procedure(
		statement.State, arguments, rows,
	)
	Status_Invariants(status, "statement_query.driver_status")
	if status != STATUS_OK {
		return status
	}
	rows_live(rows)
	return STATUS_OK
}

// Statement_Close releases one prepared driver resource.
func Statement_Close(statement Statement_Pointer) (status Status) {
	defer func() { Status_Invariants(status, "statement_close.status") }()
	Statement_Pointer_Invariants(statement, "statement_close.statement")
	statement_live(statement)
	status = statement.Close_Procedure(statement.State)
	Status_Invariants(status, "statement_close.driver_status")
	return status
}

// Transaction_Commit makes one live transaction durable.
func Transaction_Commit(transaction Transaction_Pointer) (status Status) {
	defer func() { Status_Invariants(status, "transaction_commit.status") }()
	Transaction_Pointer_Invariants(transaction, "transaction_commit.transaction")
	transaction_live(transaction)
	status = transaction.Commit_Procedure(transaction.State)
	Status_Invariants(status, "transaction_commit.driver_status")
	return status
}

// Transaction_Rollback abandons one live transaction.
func Transaction_Rollback(transaction Transaction_Pointer) (status Status) {
	defer func() { Status_Invariants(status, "transaction_rollback.status") }()
	Transaction_Pointer_Invariants(transaction, "transaction_rollback.transaction")
	transaction_live(transaction)
	status = transaction.Rollback_Procedure(transaction.State)
	Status_Invariants(status, "transaction_rollback.driver_status")
	return status
}

func arguments_validate(arguments Arguments) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "arguments_validate.status") }()
	Arguments_Invariants(arguments, "arguments_validate.arguments")
	for index := range arguments {
		argument := arguments[index]
		if !bool(argument.Validity) {
			return Validation_Status(STATUS_INPUT_INVALID)
		}
		if int(argument.Ordinal) != index+utf8.CHARACTER_SIZE_MINIMUM {
			return Validation_Status(STATUS_INPUT_INVALID)
		}
		if value_validate(argument.Value) !=
			Validation_Status(STATUS_OK) {
			return Validation_Status(STATUS_INPUT_INVALID)
		}
	}
	return Validation_Status(STATUS_OK)
}

func statement_argument_count_validate(
	statement Statement_Pointer, arguments Arguments,
) (status Validation_Status) {
	defer func() {
		Validation_Status_Invariants(status, "statement_argument_count_validate.status")
	}()
	Statement_Pointer_Invariants(
		statement, "statement_argument_count_validate.statement",
	)
	Arguments_Invariants(arguments, "statement_argument_count_validate.arguments")
	count := statement.Argument_Count
	Statement_Argument_Count_Invariants(count, "statement_argument_count_validate.count")
	if count == STATEMENT_ARGUMENT_COUNT_UNKNOWN {
		return Validation_Status(STATUS_OK)
	}
	if int(count) != len(arguments) {
		return Validation_Status(STATUS_INPUT_INVALID)
	}
	return Validation_Status(STATUS_OK)
}

func connection_live(subject Connection_Pointer) {
	Connection_Pointer_Invariants(subject, "connection_live.subject")
	aver.Always(
		subject.Close_Procedure != nil,
		"A live driver connection can release its resource.",
	)
}

func driver_live(value Driver) {
	Driver_Invariants(value, "driver_live.value")
	aver.Always(
		value.Connect_Procedure != nil,
		"Live driver can open one connection.",
	)
}

func rows_live(subject Rows_Pointer) {
	Rows_Pointer_Invariants(subject, "rows_live.subject")
	aver.Always(
		subject.Column_Count >= slices.COUNT_MINIMUM,
		"Live driver rows has no negative column count.",
	)
	aver.Always(
		subject.Column_Count <= slices.SLICE_COUNT_MAXIMUM,
		"Live driver rows fits bounded column storage.",
	)
	aver.Always(
		subject.Next_Procedure != nil,
		"Live driver rows can advance one row.",
	)
	aver.Always(
		subject.Close_Procedure != nil,
		"Live driver rows can release cursor state.",
	)
}

func statement_live(subject Statement_Pointer) {
	Statement_Pointer_Invariants(subject, "statement_live.subject")
	Statement_Argument_Count_Invariants(
		subject.Argument_Count, "statement_live.argument_count",
	)
	aver.Always(
		subject.Exec_Procedure != nil,
		"A live statement can execute.",
	)
	aver.Always(
		subject.Query_Procedure != nil,
		"A live statement can query.",
	)
	aver.Always(
		subject.Close_Procedure != nil,
		"A live statement can release its resource.",
	)
}

func transaction_options_live(options Transaction_Options) {
	Transaction_Options_Invariants(options, "transaction_options_live.options")
	Isolation_Level_Invariants(
		options.Isolation_Level,
		"transaction_options_live.isolation",
	)
	Transaction_Read_Only_Invariants(
		options.Read_Only,
		"transaction_options_live.read_only",
	)
}

func transaction_live(subject Transaction_Pointer) {
	Transaction_Pointer_Invariants(subject, "transaction_live.subject")
	aver.Always(
		subject.Commit_Procedure != nil,
		"A live transaction can commit.",
	)
	aver.Always(
		subject.Rollback_Procedure != nil,
		"A live transaction can rollback.",
	)
}
