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

// VALUE_SLOT is sole slot of each closed-union storage array.
const VALUE_SLOT = slices.COUNT_MINIMUM

// VALUE_SLOT_COUNT lets Value keep payload domains out of aggregate invariant chain.
const VALUE_SLOT_COUNT = VALUE_SLOT + 1

// Value replaces reflection and any with finite driver value domain.
type Value struct {
	// Kinds selects active union member.
	Kinds [VALUE_SLOT_COUNT]Value_Kind
	// Booleans carries VALUE_BOOLEAN payload.
	Booleans [VALUE_SLOT_COUNT]Boolean
	// Integers carries VALUE_INTEGER payload.
	Integers [VALUE_SLOT_COUNT]Integer
	// Floats carries VALUE_FLOAT binary64 bits.
	Floats [VALUE_SLOT_COUNT]Float
	// Byte_Values carries VALUE_BYTES borrowed payload.
	Byte_Values [VALUE_SLOT_COUNT]Bytes
	// Texts carries VALUE_TEXT borrowed payload.
	Texts [VALUE_SLOT_COUNT]Text
	// Moments carries VALUE_TIME payload.
	Moments [VALUE_SLOT_COUNT]time.Moment
}

// Value_Invariants states fixed storage. Payload sites prove their active domain separately.
func Value_Invariants(value Value, namespace aver.Namespace) {
	aver.Always(
		len(value.Kinds) == VALUE_SLOT_COUNT,
		"A value holds one kind slot.",
	)
	aver.Always(
		len(value.Booleans) == VALUE_SLOT_COUNT,
		"A value holds one Boolean slot.",
	)
	aver.Always(
		len(value.Integers) == VALUE_SLOT_COUNT,
		"A value holds one integer slot.",
	)
	aver.Always(
		len(value.Floats) == VALUE_SLOT_COUNT,
		"A value holds one binary64 slot.",
	)
	aver.Always(
		len(value.Byte_Values) == VALUE_SLOT_COUNT,
		"A value holds one byte-view slot.",
	)
	aver.Always(
		len(value.Texts) == VALUE_SLOT_COUNT,
		"A value holds one text slot.",
	)
	aver.Always(
		len(value.Moments) == VALUE_SLOT_COUNT,
		"A value holds one timestamp slot.",
	)
}

// Value_Null constructs SQL NULL.
func Value_Null() (result Value) {
	defer func() { Value_Invariants(result, "value_null.result") }()
	return Value{Kinds: [VALUE_SLOT_COUNT]Value_Kind{VALUE_NULL}}
}

// Value_Of_Boolean constructs one Boolean driver value.
func Value_Of_Boolean(value Boolean) (result Value) {
	defer func() { Value_Invariants(result, "value_of_boolean.result") }()
	Boolean_Invariants(value, "value_of_boolean.value")
	return Value{
		Kinds:    [VALUE_SLOT_COUNT]Value_Kind{VALUE_BOOLEAN},
		Booleans: [VALUE_SLOT_COUNT]Boolean{value},
	}
}

// Value_Of_Integer constructs one signed integer driver value.
func Value_Of_Integer(value Integer) (result Value) {
	defer func() { Value_Invariants(result, "value_of_integer.result") }()
	Integer_Invariants(value, "value_of_integer.value")
	return Value{
		Kinds:    [VALUE_SLOT_COUNT]Value_Kind{VALUE_INTEGER},
		Integers: [VALUE_SLOT_COUNT]Integer{value},
	}
}

// Value_Of_Float constructs one binary64 driver value from exact bits.
func Value_Of_Float(value Float) (result Value) {
	defer func() { Value_Invariants(result, "value_of_float.result") }()
	Float_Invariants(value, "value_of_float.value")
	return Value{
		Kinds:  [VALUE_SLOT_COUNT]Value_Kind{VALUE_FLOAT},
		Floats: [VALUE_SLOT_COUNT]Float{value},
	}
}

// Value_Of_Bytes validates and borrows bytes instead of copying driver data.
func Value_Of_Bytes(
	value Bytes_Unvalidated,
) (result Value, status Validation_Status) {
	defer func() {
		Value_Invariants(result, "value_of_bytes.result")
		Validation_Status_Invariants(status, "value_of_bytes.status")
	}()
	Bytes_Unvalidated_Invariants(value, "value_of_bytes.value")
	if len(value) > bytes.SLICE_SIZE_MAXIMUM {
		return Value{}, Validation_Status(STATUS_INPUT_INVALID)
	}
	return Value{
		Kinds:       [VALUE_SLOT_COUNT]Value_Kind{VALUE_BYTES},
		Byte_Values: [VALUE_SLOT_COUNT]Bytes{Bytes(value)},
	}, Validation_Status(STATUS_OK)
}

// Value_Of_Text validates and borrows text instead of owning conversion storage.
func Value_Of_Text(
	value Text_Unvalidated,
) (result Value, status Validation_Status) {
	defer func() {
		Value_Invariants(result, "value_of_text.result")
		Validation_Status_Invariants(status, "value_of_text.status")
	}()
	Text_Unvalidated_Invariants(value, "value_of_text.value")
	if len(value) > strings.TEXT_SIZE_MAXIMUM {
		return Value{}, Validation_Status(STATUS_INPUT_INVALID)
	}
	return Value{
		Kinds: [VALUE_SLOT_COUNT]Value_Kind{VALUE_TEXT},
		Texts: [VALUE_SLOT_COUNT]Text{Text(value)},
	}, Validation_Status(STATUS_OK)
}

// Value_Of_Time constructs one realtime database timestamp.
func Value_Of_Time(value time.Moment) (result Value) {
	defer func() { Value_Invariants(result, "value_of_time.result") }()
	time.Moment_Invariants(value, "value_of_time.value")
	return Value{
		Kinds:   [VALUE_SLOT_COUNT]Value_Kind{VALUE_TIME},
		Moments: [VALUE_SLOT_COUNT]time.Moment{value},
	}
}

// Value_Kind_Of reports active closed-union member.
func Value_Kind_Of(value Value) (kind Value_Kind) {
	defer func() { Value_Kind_Invariants(kind, "value_kind_of.kind") }()
	Value_Invariants(value, "value_kind_of.value")
	return value.Kinds[VALUE_SLOT]
}

// Value_As_Boolean reads Boolean only when tag agrees.
func Value_As_Boolean(value Value) (result Boolean, status Validation_Status) {
	defer func() {
		Boolean_Invariants(result, "value_as_boolean.result")
		Validation_Status_Invariants(status, "value_as_boolean.status")
	}()
	Value_Invariants(value, "value_as_boolean.value")
	if value.Kinds[VALUE_SLOT] != VALUE_BOOLEAN {
		return false, Validation_Status(STATUS_INPUT_INVALID)
	}
	return value.Booleans[VALUE_SLOT], Validation_Status(STATUS_OK)
}

// Value_As_Integer reads integer only when tag agrees.
func Value_As_Integer(value Value) (result Integer, status Validation_Status) {
	defer func() {
		Integer_Invariants(result, "value_as_integer.result")
		Validation_Status_Invariants(status, "value_as_integer.status")
	}()
	Value_Invariants(value, "value_as_integer.value")
	if value.Kinds[VALUE_SLOT] != VALUE_INTEGER {
		return Integer(bits.WORD_64_MINIMUM), Validation_Status(STATUS_INPUT_INVALID)
	}
	return value.Integers[VALUE_SLOT], Validation_Status(STATUS_OK)
}

// Value_As_Float reads binary64 bits only when tag agrees.
func Value_As_Float(value Value) (result Float, status Validation_Status) {
	defer func() {
		Float_Invariants(result, "value_as_float.result")
		Validation_Status_Invariants(status, "value_as_float.status")
	}()
	Value_Invariants(value, "value_as_float.value")
	if value.Kinds[VALUE_SLOT] != VALUE_FLOAT {
		return Float(bits.WORD_64_MINIMUM), Validation_Status(STATUS_INPUT_INVALID)
	}
	return value.Floats[VALUE_SLOT], Validation_Status(STATUS_OK)
}

// Value_As_Bytes returns borrowed bytes only when tag agrees.
func Value_As_Bytes(value Value) (result Bytes, status Validation_Status) {
	defer func() {
		Bytes_Invariants(result, "value_as_bytes.result")
		Validation_Status_Invariants(status, "value_as_bytes.status")
	}()
	Value_Invariants(value, "value_as_bytes.value")
	if value.Kinds[VALUE_SLOT] != VALUE_BYTES {
		return nil, Validation_Status(STATUS_INPUT_INVALID)
	}
	return value.Byte_Values[VALUE_SLOT], Validation_Status(STATUS_OK)
}

// Value_As_Text returns borrowed text only when tag agrees.
func Value_As_Text(value Value) (result Text, status Validation_Status) {
	defer func() {
		Text_Invariants(result, "value_as_text.result")
		Validation_Status_Invariants(status, "value_as_text.status")
	}()
	Value_Invariants(value, "value_as_text.value")
	if value.Kinds[VALUE_SLOT] != VALUE_TEXT {
		return "", Validation_Status(STATUS_INPUT_INVALID)
	}
	return value.Texts[VALUE_SLOT], Validation_Status(STATUS_OK)
}

// Value_As_Time reads timestamp only when tag agrees.
func Value_As_Time(value Value) (result time.Moment, status Validation_Status) {
	defer func() {
		time.Moment_Invariants(result, "value_as_time.result")
		Validation_Status_Invariants(status, "value_as_time.status")
	}()
	Value_Invariants(value, "value_as_time.value")
	if value.Kinds[VALUE_SLOT] != VALUE_TIME {
		return time.Moment(bits.WORD_64_MINIMUM), Validation_Status(STATUS_INPUT_INVALID)
	}
	return value.Moments[VALUE_SLOT], Validation_Status(STATUS_OK)
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

// NAMED_VALUE_SLOT is sole slot for each named-value property.
const NAMED_VALUE_SLOT = slices.COUNT_MINIMUM

// NAMED_VALUE_SLOT_COUNT keeps aggregate storage fixed while each property keeps own domain.
const NAMED_VALUE_SLOT_COUNT = NAMED_VALUE_SLOT + 1

// Named_Value pairs typed value with standard named or positional identity.
type Named_Value struct {
	// Names holds empty name for positional argument.
	Names [NAMED_VALUE_SLOT_COUNT]Argument_Name
	// Ordinals holds one-based position after validation.
	Ordinals [NAMED_VALUE_SLOT_COUNT]Argument_Ordinal
	// Values holds closed driver value.
	Values [NAMED_VALUE_SLOT_COUNT]Value
	// Validities prevents zero struct from becoming accepted argument.
	Validities [NAMED_VALUE_SLOT_COUNT]Named_Value_Validity
}

// Named_Value_Invariants states fixed storage. Validation proves each active property.
func Named_Value_Invariants(value Named_Value, namespace aver.Namespace) {
	aver.Always(
		len(value.Names) == NAMED_VALUE_SLOT_COUNT,
		"A named value holds one name slot.",
	)
	aver.Always(
		len(value.Ordinals) == NAMED_VALUE_SLOT_COUNT,
		"A named value holds one ordinal slot.",
	)
	aver.Always(
		len(value.Values) == NAMED_VALUE_SLOT_COUNT,
		"A named value holds one value slot.",
	)
	aver.Always(
		len(value.Validities) == NAMED_VALUE_SLOT_COUNT,
		"A named value holds one validity slot.",
	)
}

// Named_Value_Of validates standard name head and bounded one-based ordinal.
func Named_Value_Of(
	name Argument_Name_Unvalidated, ordinal Argument_Ordinal_Unvalidated, value Value,
) (named Named_Value, status Validation_Status) {
	defer func() {
		Named_Value_Invariants(named, "named_value_of.named")
		Validation_Status_Invariants(status, "named_value_of.status")
	}()
	Argument_Name_Unvalidated_Invariants(name, "named_value_of.name")
	Argument_Ordinal_Unvalidated_Invariants(ordinal, "named_value_of.ordinal")
	Value_Invariants(value, "named_value_of.value")
	if ordinal < Argument_Ordinal_Unvalidated(utf8.CHARACTER_SIZE_MINIMUM) {
		return Named_Value{}, Validation_Status(STATUS_INPUT_INVALID)
	}
	if ordinal > ARGUMENT_COUNT_MAXIMUM {
		return Named_Value{}, Validation_Status(STATUS_INPUT_INVALID)
	}
	if len(name) > strings.TEXT_SIZE_MAXIMUM {
		return Named_Value{}, Validation_Status(STATUS_INPUT_INVALID)
	}
	if name != "" {
		character, _ := utf8.Decode_Character_Text(utf8.Text(name))
		if !bool(ucd.Is_Letter(ucd.Character(character))) {
			return Named_Value{}, Validation_Status(STATUS_INPUT_INVALID)
		}
	}
	if value_validate(value) != Validation_Status(STATUS_OK) {
		return Named_Value{}, Validation_Status(STATUS_INPUT_INVALID)
	}
	return Named_Value{
		Names:      [NAMED_VALUE_SLOT_COUNT]Argument_Name{Argument_Name(name)},
		Ordinals:   [NAMED_VALUE_SLOT_COUNT]Argument_Ordinal{Argument_Ordinal(ordinal)},
		Values:     [NAMED_VALUE_SLOT_COUNT]Value{value},
		Validities: [NAMED_VALUE_SLOT_COUNT]Named_Value_Validity{true},
	}, Validation_Status(STATUS_OK)
}

// Named_Value_Name reports optional parameter name.
func Named_Value_Name(value Named_Value) (name Argument_Name) {
	defer func() { Argument_Name_Invariants(name, "named_value_name.name") }()
	Named_Value_Invariants(value, "named_value_name.value")
	return value.Names[NAMED_VALUE_SLOT]
}

// Named_Value_Ordinal reports one-based parameter position.
func Named_Value_Ordinal(value Named_Value) (ordinal Argument_Ordinal) {
	defer func() { Argument_Ordinal_Invariants(ordinal, "named_value_ordinal.ordinal") }()
	Named_Value_Invariants(value, "named_value_ordinal.value")
	return value.Ordinals[NAMED_VALUE_SLOT]
}

// Named_Value_Value reports typed parameter value.
func Named_Value_Value(value Named_Value) (result Value) {
	defer func() { Value_Invariants(result, "named_value_value.result") }()
	Named_Value_Invariants(value, "named_value_value.value")
	return value.Values[NAMED_VALUE_SLOT]
}

// Arguments is bounded borrowed parameter storage.
type Arguments []Named_Value

// Arguments_Invariants bounds parameter storage.
func Arguments_Invariants(value Arguments, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, ARGUMENT_COUNT_MAXIMUM).
		Ensure()
}

// REQUEST_SLOT is sole request property slot.
const REQUEST_SLOT = slices.COUNT_MINIMUM

// REQUEST_SLOT_COUNT keeps request aggregate out of query and argument domain chains.
const REQUEST_SLOT_COUNT = REQUEST_SLOT + 1

// Request is bounded query and borrowed typed arguments.
type Request struct {
	// Queries holds validated driver text.
	Queries [REQUEST_SLOT_COUNT]Query
	// Argument_Sets holds caller-owned typed parameter storage.
	Argument_Sets [REQUEST_SLOT_COUNT]Arguments
}

// Request_Invariants states fixed aggregate storage.
func Request_Invariants(value Request, namespace aver.Namespace) {
	aver.Always(
		len(value.Queries) == REQUEST_SLOT_COUNT,
		"A request holds one query slot.",
	)
	aver.Always(
		len(value.Argument_Sets) == REQUEST_SLOT_COUNT,
		"A request holds one argument-set slot.",
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
		Queries:       [REQUEST_SLOT_COUNT]Query{query},
		Argument_Sets: [REQUEST_SLOT_COUNT]Arguments{arguments},
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
	arguments := request.Argument_Sets[REQUEST_SLOT]
	if len(arguments) > ARGUMENT_COUNT_MAXIMUM {
		return Validation_Status(STATUS_INPUT_INVALID)
	}
	for index := range arguments {
		argument := arguments[index]
		if !bool(argument.Validities[NAMED_VALUE_SLOT]) {
			return Validation_Status(STATUS_INPUT_INVALID)
		}
		if int(argument.Ordinals[NAMED_VALUE_SLOT]) != index+utf8.CHARACTER_SIZE_MINIMUM {
			return Validation_Status(STATUS_INPUT_INVALID)
		}
		if value_validate(argument.Values[NAMED_VALUE_SLOT]) !=
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

// TRANSACTION_OPTIONS_SLOT is sole transaction option slot.
const TRANSACTION_OPTIONS_SLOT = slices.COUNT_MINIMUM

// TRANSACTION_OPTIONS_SLOT_COUNT keeps transaction option storage fixed.
const TRANSACTION_OPTIONS_SLOT_COUNT = TRANSACTION_OPTIONS_SLOT + 1

// Transaction_Options carries standard isolation and read-only request.
type Transaction_Options struct {
	// Isolation_Levels holds one standard isolation request.
	Isolation_Levels [TRANSACTION_OPTIONS_SLOT_COUNT]Isolation_Level
	// Read_Only_Values holds one read-only request.
	Read_Only_Values [TRANSACTION_OPTIONS_SLOT_COUNT]Transaction_Read_Only
}

// Transaction_Options_Invariants states fixed option storage.
func Transaction_Options_Invariants(
	value Transaction_Options, namespace aver.Namespace,
) {
	aver.Always(
		len(value.Isolation_Levels) == TRANSACTION_OPTIONS_SLOT_COUNT,
		"Transaction options holds one isolation slot.",
	)
	aver.Always(
		len(value.Read_Only_Values) == TRANSACTION_OPTIONS_SLOT_COUNT,
		"Transaction options holds one read-only slot.",
	)
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
		Isolation_Levels: [TRANSACTION_OPTIONS_SLOT_COUNT]Isolation_Level{isolation},
		Read_Only_Values: [TRANSACTION_OPTIONS_SLOT_COUNT]Transaction_Read_Only{read_only},
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

// RESULT_SLOT is sole slot for each optional result property.
const RESULT_SLOT = slices.COUNT_MINIMUM

// RESULT_SLOT_COUNT keeps aggregate result storage fixed.
const RESULT_SLOT_COUNT = RESULT_SLOT + 1

// Result holds optional standard execution counters without interface box.
type Result struct {
	// Last_Insert_Identifiers holds value meaningful when matching validity is true.
	Last_Insert_Identifiers [RESULT_SLOT_COUNT]Last_Insert_Identifier
	// Rows_Affected_Counts holds value meaningful when matching validity is true.
	Rows_Affected_Counts [RESULT_SLOT_COUNT]Rows_Affected
	// Last_Insert_Identifier_Validities distinguishes unsupported from zero identifier.
	Last_Insert_Identifier_Validities [RESULT_SLOT_COUNT]Last_Insert_Identifier_Validity
	// Rows_Affected_Validities distinguishes unsupported from zero count.
	Rows_Affected_Validities [RESULT_SLOT_COUNT]Rows_Affected_Validity
}

// Result_Invariants states fixed optional-counter storage.
func Result_Invariants(subject *Result, namespace aver.Namespace) {
	aver.Always(subject != nil, "Result storage exists.")
	aver.Always(
		len(subject.Last_Insert_Identifiers) == RESULT_SLOT_COUNT,
		"A result holds one last insert identifier slot.",
	)
	aver.Always(
		len(subject.Rows_Affected_Counts) == RESULT_SLOT_COUNT,
		"A result holds one rows affected slot.",
	)
	aver.Always(
		len(subject.Last_Insert_Identifier_Validities) == RESULT_SLOT_COUNT,
		"A result holds one identifier-validity slot.",
	)
	aver.Always(
		len(subject.Rows_Affected_Validities) == RESULT_SLOT_COUNT,
		"A result holds one affected-count-validity slot.",
	)
}

// Result_Set_Last_Insert_Identifier lets driver report generated identity.
func Result_Set_Last_Insert_Identifier(result *Result, value Last_Insert_Identifier) {
	Result_Invariants(result, "result_set_last_insert_identifier.result")
	Last_Insert_Identifier_Invariants(value, "result_set_last_insert_identifier.value")
	result.Last_Insert_Identifiers[RESULT_SLOT] = value
	result.Last_Insert_Identifier_Validities[RESULT_SLOT] = true
}

// Result_Set_Rows_Affected lets driver report affected count.
func Result_Set_Rows_Affected(result *Result, value Rows_Affected) {
	Result_Invariants(result, "result_set_rows_affected.result")
	Rows_Affected_Invariants(value, "result_set_rows_affected.value")
	result.Rows_Affected_Counts[RESULT_SLOT] = value
	result.Rows_Affected_Validities[RESULT_SLOT] = true
}

// Result_Last_Insert_Identifier reads generated identity when reported.
func Result_Last_Insert_Identifier(
	result *Result,
) (value Last_Insert_Identifier, status Optional_Status) {
	defer func() {
		Last_Insert_Identifier_Invariants(value, "result_last_insert_identifier.value")
		Optional_Status_Invariants(status, "result_last_insert_identifier.status")
	}()
	Result_Invariants(result, "result_last_insert_identifier.result")
	Last_Insert_Identifier_Validity_Invariants(
		result.Last_Insert_Identifier_Validities[RESULT_SLOT],
		"result_last_insert_identifier.valid",
	)
	if !bool(result.Last_Insert_Identifier_Validities[RESULT_SLOT]) {
		return Last_Insert_Identifier(bits.WORD_64_MINIMUM),
			Optional_Status(STATUS_UNSUPPORTED)
	}
	return result.Last_Insert_Identifiers[RESULT_SLOT], Optional_Status(STATUS_OK)
}

// Result_Rows_Affected reads affected count when reported.
func Result_Rows_Affected(
	result *Result,
) (value Rows_Affected, status Optional_Status) {
	defer func() {
		Rows_Affected_Invariants(value, "result_rows_affected.value")
		Optional_Status_Invariants(status, "result_rows_affected.status")
	}()
	Result_Invariants(result, "result_rows_affected.result")
	Rows_Affected_Validity_Invariants(
		result.Rows_Affected_Validities[RESULT_SLOT],
		"result_rows_affected.valid",
	)
	if !bool(result.Rows_Affected_Validities[RESULT_SLOT]) {
		return Rows_Affected(bits.WORD_64_MINIMUM), Optional_Status(STATUS_UNSUPPORTED)
	}
	return result.Rows_Affected_Counts[RESULT_SLOT], Optional_Status(STATUS_OK)
}

func value_validate(value Value) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "value_validate.status") }()
	Value_Invariants(value, "value_valid.value")
	switch value.Kinds[VALUE_SLOT] {
	case VALUE_NULL, VALUE_BOOLEAN, VALUE_INTEGER, VALUE_FLOAT, VALUE_TIME:
		return Validation_Status(STATUS_OK)
	case VALUE_BYTES:
		if len(value.Byte_Values[VALUE_SLOT]) > bytes.SLICE_SIZE_MAXIMUM {
			return Validation_Status(STATUS_INPUT_INVALID)
		}
		return Validation_Status(STATUS_OK)
	case VALUE_TEXT:
		if len(value.Texts[VALUE_SLOT]) > strings.TEXT_SIZE_MAXIMUM {
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

// STATEMENT_SLOT is sole prepared statement property slot.
const STATEMENT_SLOT = slices.COUNT_MINIMUM

// STATEMENT_SLOT_COUNT keeps prepared statement storage fixed.
const STATEMENT_SLOT_COUNT = STATEMENT_SLOT + 1

// Statement is one prepared driver statement procedure table.
type Statement struct {
	// States belongs to driver until close.
	States [STATEMENT_SLOT_COUNT]unsafe.Pointer
	// Argument_Counts is exact count or STATEMENT_ARGUMENT_COUNT_UNKNOWN.
	Argument_Counts [STATEMENT_SLOT_COUNT]Statement_Argument_Count
	// Exec_Procedures executes prepared request into caller result.
	Exec_Procedures [STATEMENT_SLOT_COUNT]func(
		state unsafe.Pointer, arguments Arguments, result *Result,
	) (status Status)
	// Query_Procedures opens prepared rows into caller storage.
	Query_Procedures [STATEMENT_SLOT_COUNT]func(
		state unsafe.Pointer, arguments Arguments, rows *Rows,
	) (status Status)
	// Close_Procedures releases prepared driver state.
	Close_Procedures [STATEMENT_SLOT_COUNT]func(state unsafe.Pointer) (status Status)
}

// Statement_Invariants states fixed prepared statement storage.
func Statement_Invariants(subject *Statement, namespace aver.Namespace) {
	aver.Always(subject != nil, "Statement storage exists.")
	aver.Always(
		len(subject.States) == STATEMENT_SLOT_COUNT,
		"Statement holds one state slot.",
	)
	aver.Always(
		len(subject.Argument_Counts) == STATEMENT_SLOT_COUNT,
		"Statement holds one argument-count slot.",
	)
	aver.Always(
		len(subject.Exec_Procedures) == STATEMENT_SLOT_COUNT,
		"Statement holds one execute-procedure slot.",
	)
	aver.Always(
		len(subject.Query_Procedures) == STATEMENT_SLOT_COUNT,
		"Statement holds one query-procedure slot.",
	)
	aver.Always(
		len(subject.Close_Procedures) == STATEMENT_SLOT_COUNT,
		"Statement holds one close-procedure slot.",
	)
}

// TRANSACTION_SLOT is sole transaction property slot.
const TRANSACTION_SLOT = slices.COUNT_MINIMUM

// TRANSACTION_SLOT_COUNT keeps transaction storage fixed.
const TRANSACTION_SLOT_COUNT = TRANSACTION_SLOT + 1

// Transaction is one live driver transaction terminal procedure table.
type Transaction struct {
	// States belongs to driver until commit or rollback.
	States [TRANSACTION_SLOT_COUNT]unsafe.Pointer
	// Commit_Procedures makes transaction changes durable.
	Commit_Procedures [TRANSACTION_SLOT_COUNT]func(state unsafe.Pointer) (status Status)
	// Rollback_Procedures abandons transaction changes.
	Rollback_Procedures [TRANSACTION_SLOT_COUNT]func(state unsafe.Pointer) (status Status)
}

// Transaction_Invariants states fixed driver transaction storage.
func Transaction_Invariants(subject *Transaction, namespace aver.Namespace) {
	aver.Always(subject != nil, "Transaction storage exists.")
	aver.Always(
		len(subject.States) == TRANSACTION_SLOT_COUNT,
		"Transaction holds one state slot.",
	)
	aver.Always(
		len(subject.Commit_Procedures) == TRANSACTION_SLOT_COUNT,
		"Transaction holds one commit-procedure slot.",
	)
	aver.Always(
		len(subject.Rollback_Procedures) == TRANSACTION_SLOT_COUNT,
		"Transaction holds one rollback-procedure slot.",
	)
}

// Driver injects connection creation over explicit caller-owned state.
type Driver struct {
	// State belongs to driver root and survives every connection.
	State unsafe.Pointer
	// Connect_Procedure fills caller connection storage.
	Connect_Procedure func(
		state unsafe.Pointer, data_source Data_Source, destination *Connection,
	) (status Status)
}

// Driver_Invariants requires connection creation capability.
func Driver_Invariants(value Driver, namespace aver.Namespace) {
	aver.Always(value.Connect_Procedure != nil, "A driver can open one connection.")
}

// CONNECTION_SLOT is sole live connection property slot.
const CONNECTION_SLOT = slices.COUNT_MINIMUM

// CONNECTION_SLOT_COUNT keeps caller connection storage fixed.
const CONNECTION_SLOT_COUNT = CONNECTION_SLOT + 1

// Connection is one live driver connection procedure table.
type Connection struct {
	// States belongs to driver and remains live until close procedure returns.
	States [CONNECTION_SLOT_COUNT]unsafe.Pointer
	// Close_Procedures releases driver resource.
	Close_Procedures [CONNECTION_SLOT_COUNT]func(state unsafe.Pointer) (status Status)
	// Probe_Procedures checks liveness; nil keeps standard optional behavior.
	Probe_Procedures [CONNECTION_SLOT_COUNT]func(state unsafe.Pointer) (status Status)
	// Exec_Procedures executes request without result rows.
	Exec_Procedures [CONNECTION_SLOT_COUNT]func(
		state unsafe.Pointer, request Request, result *Result,
	) (status Status)
	// Query_Procedures opens result rows into caller storage.
	Query_Procedures [CONNECTION_SLOT_COUNT]func(
		state unsafe.Pointer, request Request, rows *Rows,
	) (status Status)
	// Prepare_Procedures prepares query into caller statement storage.
	Prepare_Procedures [CONNECTION_SLOT_COUNT]func(
		state unsafe.Pointer, query Query, statement *Statement,
	) (status Status)
	// Begin_Procedures starts transaction into caller storage.
	Begin_Procedures [CONNECTION_SLOT_COUNT]func(
		state unsafe.Pointer, options Transaction_Options, transaction *Transaction,
	) (status Status)
}

// Connection_Invariants states fixed caller storage before and after driver fill.
func Connection_Invariants(subject *Connection, namespace aver.Namespace) {
	aver.Always(subject != nil, "Connection storage exists.")
	aver.Always(
		len(subject.States) == CONNECTION_SLOT_COUNT,
		"A connection holds one state slot.",
	)
	aver.Always(
		len(subject.Close_Procedures) == CONNECTION_SLOT_COUNT,
		"A connection holds one close-procedure slot.",
	)
	aver.Always(
		len(subject.Probe_Procedures) == CONNECTION_SLOT_COUNT,
		"A connection holds one probe-procedure slot.",
	)
	aver.Always(
		len(subject.Exec_Procedures) == CONNECTION_SLOT_COUNT,
		"A connection holds one execute-procedure slot.",
	)
	aver.Always(
		len(subject.Query_Procedures) == CONNECTION_SLOT_COUNT,
		"A connection holds one query-procedure slot.",
	)
	aver.Always(
		len(subject.Prepare_Procedures) == CONNECTION_SLOT_COUNT,
		"A connection holds one prepare-procedure slot.",
	)
	aver.Always(
		len(subject.Begin_Procedures) == CONNECTION_SLOT_COUNT,
		"A connection holds one begin-procedure slot.",
	)
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

// ROWS_SLOT is sole cursor property slot.
const ROWS_SLOT = slices.COUNT_MINIMUM

// ROWS_SLOT_COUNT keeps cursor aggregate outside each property domain chain.
const ROWS_SLOT_COUNT = ROWS_SLOT + 1

// Rows is one live driver cursor procedure table.
type Rows struct {
	// States belongs to driver and remains live until close procedure returns.
	States [ROWS_SLOT_COUNT]unsafe.Pointer
	// Column_Counts fixes exact destination width every next procedure call needs.
	Column_Counts [ROWS_SLOT_COUNT]Column_Count
	// Next_Procedures writes one row into exact caller slots or returns STATUS_DONE.
	Next_Procedures [ROWS_SLOT_COUNT]func(
		state unsafe.Pointer, destination Values,
	) (status Status)
	// Close_Procedures releases cursor state.
	Close_Procedures [ROWS_SLOT_COUNT]func(state unsafe.Pointer) (status Status)
}

// Rows_Invariants states fixed caller storage before and after driver fill.
func Rows_Invariants(subject *Rows, namespace aver.Namespace) {
	aver.Always(subject != nil, "Driver rows storage exists.")
	aver.Always(
		len(subject.States) == ROWS_SLOT_COUNT,
		"Driver rows holds one state slot.",
	)
	aver.Always(
		len(subject.Column_Counts) == ROWS_SLOT_COUNT,
		"Driver rows holds one column-count slot.",
	)
	aver.Always(
		len(subject.Next_Procedures) == ROWS_SLOT_COUNT,
		"Driver rows holds one next-procedure slot.",
	)
	aver.Always(
		len(subject.Close_Procedures) == ROWS_SLOT_COUNT,
		"Driver rows holds one close-procedure slot.",
	)
}

// Connect asks injected driver to fill one caller-owned live connection.
func Connect(
	driver Driver, data_source Data_Source, destination *Connection,
) (status Status) {
	defer func() { Status_Invariants(status, "connect.status") }()
	Driver_Invariants(driver, "connect.driver")
	Data_Source_Invariants(data_source, "connect.data_source")
	Connection_Invariants(destination, "connect.destination")
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
func Connection_Probe(connection *Connection) (status Status) {
	defer func() { Status_Invariants(status, "connection_probe.status") }()
	Connection_Invariants(connection, "connection_probe.connection")
	connection_live(connection)
	if connection.Probe_Procedures[CONNECTION_SLOT] == nil {
		return STATUS_OK
	}
	status = connection.Probe_Procedures[CONNECTION_SLOT](
		connection.States[CONNECTION_SLOT],
	)
	Status_Invariants(status, "connection_probe.driver_status")
	return status
}

// Connection_Exec validates request before driver execution.
func Connection_Exec(
	connection *Connection, request Request, result *Result,
) (status Status) {
	defer func() { Status_Invariants(status, "connection_exec.status") }()
	Connection_Invariants(connection, "connection_exec.connection")
	Request_Invariants(request, "connection_exec.request")
	Result_Invariants(result, "connection_exec.result")
	connection_live(connection)
	status = Status(Request_Validate(request))
	if status != STATUS_OK {
		return status
	}
	*result = Result{}
	if connection.Exec_Procedures[CONNECTION_SLOT] == nil {
		return STATUS_UNSUPPORTED
	}
	status = connection.Exec_Procedures[CONNECTION_SLOT](
		connection.States[CONNECTION_SLOT], request, result,
	)
	Status_Invariants(status, "connection_exec.driver_status")
	Result_Invariants(result, "connection_exec.output")
	return status
}

// Connection_Query validates request and caller rows before driver call.
func Connection_Query(
	connection *Connection, request Request, rows *Rows,
) (status Status) {
	defer func() { Status_Invariants(status, "connection_query.status") }()
	Connection_Invariants(connection, "connection_query.connection")
	Request_Invariants(request, "connection_query.request")
	Rows_Invariants(rows, "connection_query.rows")
	connection_live(connection)
	status = Status(Request_Validate(request))
	if status != STATUS_OK {
		return status
	}
	*rows = Rows{}
	if connection.Query_Procedures[CONNECTION_SLOT] == nil {
		return STATUS_UNSUPPORTED
	}
	status = connection.Query_Procedures[CONNECTION_SLOT](
		connection.States[CONNECTION_SLOT], request, rows,
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
	connection *Connection, query Query, statement *Statement,
) (status Status) {
	defer func() { Status_Invariants(status, "connection_prepare.status") }()
	Connection_Invariants(connection, "connection_prepare.connection")
	Query_Invariants(query, "connection_prepare.query")
	Statement_Invariants(statement, "connection_prepare.statement")
	connection_live(connection)
	if connection.Prepare_Procedures[CONNECTION_SLOT] == nil {
		return STATUS_UNSUPPORTED
	}
	*statement = Statement{}
	status = connection.Prepare_Procedures[CONNECTION_SLOT](
		connection.States[CONNECTION_SLOT], query, statement,
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
	connection *Connection, options Transaction_Options, transaction *Transaction,
) (status Status) {
	defer func() { Status_Invariants(status, "connection_begin.status") }()
	Connection_Invariants(connection, "connection_begin.connection")
	Transaction_Options_Invariants(options, "connection_begin.options")
	Transaction_Invariants(transaction, "connection_begin.transaction")
	connection_live(connection)
	transaction_options_live(options)
	if connection.Begin_Procedures[CONNECTION_SLOT] == nil {
		return STATUS_UNSUPPORTED
	}
	*transaction = Transaction{}
	status = connection.Begin_Procedures[CONNECTION_SLOT](
		connection.States[CONNECTION_SLOT], options, transaction,
	)
	Status_Invariants(status, "connection_begin.driver_status")
	if status != STATUS_OK {
		return status
	}
	transaction_live(transaction)
	return STATUS_OK
}

// Connection_Close releases one live driver resource.
func Connection_Close(connection *Connection) (status Status) {
	defer func() { Status_Invariants(status, "connection_close.status") }()
	Connection_Invariants(connection, "connection_close.connection")
	connection_live(connection)
	status = connection.Close_Procedures[CONNECTION_SLOT](
		connection.States[CONNECTION_SLOT],
	)
	Status_Invariants(status, "connection_close.driver_status")
	return status
}

// Rows_Next writes one exact-width row into caller storage.
func Rows_Next(rows *Rows, destination Values) (status Status) {
	defer func() { Status_Invariants(status, "rows_next.status") }()
	Rows_Invariants(rows, "rows_next.rows")
	Values_Invariants(destination, "rows_next.destination")
	rows_live(rows)
	if len(destination) != int(rows.Column_Counts[ROWS_SLOT]) {
		return STATUS_STORAGE_INVALID
	}
	status = rows.Next_Procedures[ROWS_SLOT](rows.States[ROWS_SLOT], destination)
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
func Rows_Close(rows *Rows) (status Status) {
	defer func() { Status_Invariants(status, "rows_close.status") }()
	Rows_Invariants(rows, "rows_close.rows")
	rows_live(rows)
	status = rows.Close_Procedures[ROWS_SLOT](rows.States[ROWS_SLOT])
	Status_Invariants(status, "rows_close.driver_status")
	return status
}

// Statement_Exec validates exact argument count before prepared execution.
func Statement_Exec(
	statement *Statement, arguments Arguments, result *Result,
) (status Status) {
	defer func() { Status_Invariants(status, "statement_exec.status") }()
	Statement_Invariants(statement, "statement_exec.statement")
	Arguments_Invariants(arguments, "statement_exec.arguments")
	Result_Invariants(result, "statement_exec.result")
	statement_live(statement)
	if arguments_validate(arguments) != Validation_Status(STATUS_OK) {
		return STATUS_INPUT_INVALID
	}
	if statement_argument_count_validate(statement, arguments) !=
		Validation_Status(STATUS_OK) {
		return STATUS_INPUT_INVALID
	}
	*result = Result{}
	status = statement.Exec_Procedures[STATEMENT_SLOT](
		statement.States[STATEMENT_SLOT], arguments, result,
	)
	Status_Invariants(status, "statement_exec.driver_status")
	Result_Invariants(result, "statement_exec.output")
	return status
}

// Statement_Query validates exact argument count before opening prepared rows.
func Statement_Query(
	statement *Statement, arguments Arguments, rows *Rows,
) (status Status) {
	defer func() { Status_Invariants(status, "statement_query.status") }()
	Statement_Invariants(statement, "statement_query.statement")
	Arguments_Invariants(arguments, "statement_query.arguments")
	Rows_Invariants(rows, "statement_query.rows")
	statement_live(statement)
	if arguments_validate(arguments) != Validation_Status(STATUS_OK) {
		return STATUS_INPUT_INVALID
	}
	if statement_argument_count_validate(statement, arguments) !=
		Validation_Status(STATUS_OK) {
		return STATUS_INPUT_INVALID
	}
	*rows = Rows{}
	status = statement.Query_Procedures[STATEMENT_SLOT](
		statement.States[STATEMENT_SLOT], arguments, rows,
	)
	Status_Invariants(status, "statement_query.driver_status")
	if status != STATUS_OK {
		return status
	}
	rows_live(rows)
	return STATUS_OK
}

// Statement_Close releases one prepared driver resource.
func Statement_Close(statement *Statement) (status Status) {
	defer func() { Status_Invariants(status, "statement_close.status") }()
	Statement_Invariants(statement, "statement_close.statement")
	statement_live(statement)
	status = statement.Close_Procedures[STATEMENT_SLOT](
		statement.States[STATEMENT_SLOT],
	)
	Status_Invariants(status, "statement_close.driver_status")
	return status
}

// Transaction_Commit makes one live transaction durable.
func Transaction_Commit(transaction *Transaction) (status Status) {
	defer func() { Status_Invariants(status, "transaction_commit.status") }()
	Transaction_Invariants(transaction, "transaction_commit.transaction")
	transaction_live(transaction)
	status = transaction.Commit_Procedures[TRANSACTION_SLOT](
		transaction.States[TRANSACTION_SLOT],
	)
	Status_Invariants(status, "transaction_commit.driver_status")
	return status
}

// Transaction_Rollback abandons one live transaction.
func Transaction_Rollback(transaction *Transaction) (status Status) {
	defer func() { Status_Invariants(status, "transaction_rollback.status") }()
	Transaction_Invariants(transaction, "transaction_rollback.transaction")
	transaction_live(transaction)
	status = transaction.Rollback_Procedures[TRANSACTION_SLOT](
		transaction.States[TRANSACTION_SLOT],
	)
	Status_Invariants(status, "transaction_rollback.driver_status")
	return status
}

func arguments_validate(arguments Arguments) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "arguments_validate.status") }()
	Arguments_Invariants(arguments, "arguments_validate.arguments")
	for index := range arguments {
		argument := arguments[index]
		if !bool(argument.Validities[NAMED_VALUE_SLOT]) {
			return Validation_Status(STATUS_INPUT_INVALID)
		}
		if int(argument.Ordinals[NAMED_VALUE_SLOT]) != index+utf8.CHARACTER_SIZE_MINIMUM {
			return Validation_Status(STATUS_INPUT_INVALID)
		}
		if value_validate(argument.Values[NAMED_VALUE_SLOT]) !=
			Validation_Status(STATUS_OK) {
			return Validation_Status(STATUS_INPUT_INVALID)
		}
	}
	return Validation_Status(STATUS_OK)
}

func statement_argument_count_validate(
	statement *Statement, arguments Arguments,
) (status Validation_Status) {
	defer func() {
		Validation_Status_Invariants(status, "statement_argument_count_validate.status")
	}()
	Statement_Invariants(statement, "statement_argument_count_validate.statement")
	Arguments_Invariants(arguments, "statement_argument_count_validate.arguments")
	count := statement.Argument_Counts[STATEMENT_SLOT]
	Statement_Argument_Count_Invariants(count, "statement_argument_count_validate.count")
	if count == STATEMENT_ARGUMENT_COUNT_UNKNOWN {
		return Validation_Status(STATUS_OK)
	}
	if int(count) != len(arguments) {
		return Validation_Status(STATUS_INPUT_INVALID)
	}
	return Validation_Status(STATUS_OK)
}

func connection_live(subject *Connection) {
	Connection_Invariants(subject, "connection_live.subject")
	aver.Always(
		subject.Close_Procedures[CONNECTION_SLOT] != nil,
		"A live driver connection can release its resource.",
	)
}

func rows_live(subject *Rows) {
	Rows_Invariants(subject, "rows_live.subject")
	aver.Always(
		subject.Column_Counts[ROWS_SLOT] >= slices.COUNT_MINIMUM,
		"Live driver rows has no negative column count.",
	)
	aver.Always(
		subject.Column_Counts[ROWS_SLOT] <= slices.SLICE_COUNT_MAXIMUM,
		"Live driver rows fits bounded column storage.",
	)
	aver.Always(
		subject.Next_Procedures[ROWS_SLOT] != nil,
		"Live driver rows can advance one row.",
	)
	aver.Always(
		subject.Close_Procedures[ROWS_SLOT] != nil,
		"Live driver rows can release cursor state.",
	)
}

func statement_live(subject *Statement) {
	Statement_Invariants(subject, "statement_live.subject")
	Statement_Argument_Count_Invariants(
		subject.Argument_Counts[STATEMENT_SLOT], "statement_live.argument_count",
	)
	aver.Always(
		subject.Exec_Procedures[STATEMENT_SLOT] != nil,
		"A live statement can execute.",
	)
	aver.Always(
		subject.Query_Procedures[STATEMENT_SLOT] != nil,
		"A live statement can query.",
	)
	aver.Always(
		subject.Close_Procedures[STATEMENT_SLOT] != nil,
		"A live statement can release its resource.",
	)
}

func transaction_options_live(options Transaction_Options) {
	Transaction_Options_Invariants(options, "transaction_options_live.options")
	Isolation_Level_Invariants(
		options.Isolation_Levels[TRANSACTION_OPTIONS_SLOT],
		"transaction_options_live.isolation",
	)
	Transaction_Read_Only_Invariants(
		options.Read_Only_Values[TRANSACTION_OPTIONS_SLOT],
		"transaction_options_live.read_only",
	)
}

func transaction_live(subject *Transaction) {
	Transaction_Invariants(subject, "transaction_live.subject")
	aver.Always(
		subject.Commit_Procedures[TRANSACTION_SLOT] != nil,
		"A live transaction can commit.",
	)
	aver.Always(
		subject.Rollback_Procedures[TRANSACTION_SLOT] != nil,
		"A live transaction can rollback.",
	)
}
