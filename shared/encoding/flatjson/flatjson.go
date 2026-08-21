// Package flatjson writes one flat JSON object, or one array of flat objects.
// Nested struct paths become underscore-separated keys. Maps and nested arrays stay rejected.
package flatjson

import (
	"errors"
	"reflect"
	"unsafe"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/strconv"
	"local/james-orcales/shared/strings"
)

// KEY_SEPARATOR preserves the external flat-json path convention.
const KEY_SEPARATOR = "_"

// FRAME_COUNT_MAXIMUM bounds hostile reflected nesting independently from output size.
const FRAME_COUNT_MAXIMUM = 64

// FRAME_COUNT_MINIMUM keeps one root while traversal is active.
const FRAME_COUNT_MINIMUM = 1

// STRUCTURE_FIELD_COUNT_MAXIMUM keeps reflect.Type.Field on its static index storage.
const STRUCTURE_FIELD_COUNT_MAXIMUM = 256

// FIELD_INDEX_MAXIMUM is the final field within static reflection metadata.
const FIELD_INDEX_MAXIMUM = STRUCTURE_FIELD_COUNT_MAXIMUM - 1

// JSON_ESCAPE_SIZE is one forced four-digit Unicode escape.
const JSON_ESCAPE_SIZE = 6

// NUMBER_BASE fixes JSON integers to decimal text.
const NUMBER_BASE = strconv.DECIMAL_BASE

// IMPOSSIBLE_DATA_SIZE separates errors from the smallest valid JSON document.
const IMPOSSIBLE_DATA_SIZE = 1

// OUTPUT_SIZE_MAXIMUM follows shared bounded text storage.
const OUTPUT_SIZE_MAXIMUM = strings.TEXT_SIZE_MAXIMUM

// FLATTENED_FIELD_SIZE_MINIMUM is one key, one scalar byte, and their framing.
const FLATTENED_FIELD_SIZE_MINIMUM = 6

// COLLISION_DATA_SIZE_MAXIMUM leaves one final object-closing byte.
const COLLISION_DATA_SIZE_MAXIMUM = OUTPUT_SIZE_MAXIMUM - len("}")

// FLATTENED_FIELD_COUNT_MAXIMUM leaves one final object-closing byte.
const FLATTENED_FIELD_COUNT_MAXIMUM = COLLISION_DATA_SIZE_MAXIMUM / FLATTENED_FIELD_SIZE_MINIMUM

// TAG_LITERAL_SIZE_MINIMUM is two quote bytes around empty metadata.
const TAG_LITERAL_SIZE_MINIMUM = 2

// IMPOSSIBLE_TAG_LITERAL_SIZE separates absent metadata from a complete quoted literal.
const IMPOSSIBLE_TAG_LITERAL_SIZE = TAG_LITERAL_SIZE_MINIMUM - 1

// JSON_TAG_PREFIX_SIZE is the metadata key and its colon.
const JSON_TAG_PREFIX_SIZE = len("json:")

// TAG_LITERAL_SIZE_MAXIMUM leaves the JSON metadata prefix inside the tag bound.
const TAG_LITERAL_SIZE_MAXIMUM = OUTPUT_SIZE_MAXIMUM - JSON_TAG_PREFIX_SIZE

// ERR_SHORT_WRITE distinguishes silent partial acceptance from an ordinary writer error.
var ERR_SHORT_WRITE = errors.New("flatjson: short write")

// ERR_OUTPUT_TOO_SMALL preserves caller output before complete encoding exists.
var ERR_OUTPUT_TOO_SMALL = errors.New("flatjson: output too small")

// ERR_POINTER_DEPTH_EXCEEDED rejects malicious indirection depth.
var ERR_POINTER_DEPTH_EXCEEDED = errors.New("flatjson: pointer nesting exceeds limit")

// ERR_NIL_ROOT rejects absent top-level source state.
var ERR_NIL_ROOT = errors.New("flatjson: Marshal of a nil pointer")

// ERR_ROOT_TYPE rejects roots outside one object or an array of objects.
var ERR_ROOT_TYPE = errors.New("flatjson: Marshal requires a struct or a slice of them")

// ERR_STRUCTURE_FIELD_COUNT rejects hostile reflected width.
var ERR_STRUCTURE_FIELD_COUNT = errors.New("flatjson: structure field count exceeds limit")

// ERR_ARRAY_ELEMENT_COUNT rejects hostile top-level array width.
var ERR_ARRAY_ELEMENT_COUNT = errors.New("flatjson: array element count exceeds limit")

// ERR_ARRAY_ELEMENT_TYPE rejects nested top-level elements.
var ERR_ARRAY_ELEMENT_TYPE = errors.New("flatjson: array element is not flat")

// ERR_SCALAR_ELEMENT_COUNT rejects hostile leaf array width.
var ERR_SCALAR_ELEMENT_COUNT = errors.New("flatjson: scalar array element count exceeds limit")

// ERR_STRING_SIZE rejects hostile string width before escaping.
var ERR_STRING_SIZE = errors.New("flatjson: string exceeds limit")

// ERR_VALUE_NOT_FLAT rejects maps, floats, complex values, and nested collections.
var ERR_VALUE_NOT_FLAT = errors.New("flatjson: value is not flat")

// ERR_STRUCTURE_DEPTH rejects malicious structural depth.
var ERR_STRUCTURE_DEPTH = errors.New("flatjson: structure nesting exceeds limit")

// ERR_DUPLICATE_KEY rejects ambiguous flattened output.
var ERR_DUPLICATE_KEY = errors.New("flatjson: duplicate flattened key")

// ERR_FLATTENED_FIELD_COUNT rejects more keys than bounded metadata can record.
var ERR_FLATTENED_FIELD_COUNT = errors.New("flatjson: flattened field count exceeds limit")

// ERR_FLATTENED_KEY_SIZE rejects a path that cannot fit output storage.
var ERR_FLATTENED_KEY_SIZE = errors.New("flatjson: flattened key exceeds limit")

// ERR_DESCENT_TYPE rejects pointer chains that do not end at a structure.
var ERR_DESCENT_TYPE = errors.New("flatjson: Marshal only descends into struct values")

// ERR_UNEXPORTED_LEAF rejects inaccessible reflected state.
var ERR_UNEXPORTED_LEAF = errors.New("flatjson: marshalled leaf is not exported")

// ERR_ENCODED_OUTPUT_SIZE rejects output beyond the package bound.
var ERR_ENCODED_OUTPUT_SIZE = errors.New("flatjson: encoded output exceeds limit")

// ERR_FIELD_NAME_SIZE rejects hostile reflected field metadata.
var ERR_FIELD_NAME_SIZE = errors.New("flatjson: field name exceeds limit")

// ERR_FIELD_TAG_SIZE rejects hostile reflected tag metadata.
var ERR_FIELD_TAG_SIZE = errors.New("flatjson: field tag exceeds limit")

// ERR_FIELD_TAG_SYNTAX rejects malformed reflected tag metadata.
var ERR_FIELD_TAG_SYNTAX = errors.New("flatjson: field tag syntax is invalid")

// ERR_WRITE_NIL rejects absent blocking ownership before encoding starts.
var ERR_WRITE_NIL = errors.New("flatjson: writer is nil")

// Write keeps blocking ownership at the caller's composition boundary.
type Write func([]byte) (written int, err error)

// Write_Invariants covers present and hostile absent writer bindings.
func Write_Invariants(value Write, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(value != nil, "Writer binding is present.").
		Ensure()
}

// Value carries caller-created reflection state without interface boxing.
type Value reflect.Value

// Value_Invariants covers valid and hostile invalid reflection state.
func Value_Invariants(value Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(reflect.Value(value).IsValid(), "Reflected source is valid.").
		Ensure()
}

// Error carries reflection, encoding, or writer failure.
type Error interface {
	error
}

// Data is either empty on error or one complete bounded JSON document.
type Data []byte

// Data_Invariants excludes the byte count no complete JSON value can have.
func Data_Invariants(value Data, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int(
			len(value), strings.TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM,
			IMPOSSIBLE_DATA_SIZE, IMPOSSIBLE_DATA_SIZE,
			IMPOSSIBLE_DATA_SIZE, IMPOSSIBLE_DATA_SIZE,
		).
		Ensure()
}

// Output is caller-owned JSON storage.
type Output []byte

// Output_Invariants follows shared bounded byte storage.
func Output_Invariants(value Output, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM).
		Ensure()
}

// Field_Index selects one field within static reflection metadata.
type Field_Index int

// Field_Index_Invariants matches bounded reflected struct width.
func Field_Index_Invariants(value Field_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), strings.TEXT_SIZE_MINIMUM, FIELD_INDEX_MAXIMUM).
		Ensure()
}

// Field_Position includes every field index plus its completed boundary.
type Field_Position int

// Field_Position_Invariants matches bounded reflected struct traversal.
func Field_Position_Invariants(value Field_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), strings.TEXT_SIZE_MINIMUM, STRUCTURE_FIELD_COUNT_MAXIMUM).
		Ensure()
}

// IMPOSSIBLE_PREFIX_SIZE excludes a path that cannot hold a name and separator.
const IMPOSSIBLE_PREFIX_SIZE = 1

// Prefix_Size selects active bytes in caller-stack path storage.
type Prefix_Size int

// Prefix_Size_Invariants excludes one byte because each prefix ends in separator.
func Prefix_Size_Invariants(value Prefix_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int(
			int(value), strings.TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM,
			IMPOSSIBLE_PREFIX_SIZE, IMPOSSIBLE_PREFIX_SIZE,
			IMPOSSIBLE_PREFIX_SIZE, IMPOSSIBLE_PREFIX_SIZE,
		).
		Ensure()
}

// Path is complete fixed flattened-key storage exposed as an exact caller slice.
type Path []byte

// Path_Invariants fixes traversal storage width and prevents hidden growth.
func Path_Invariants(value Path, _ aver.Namespace) {
	aver.Always(len(value) == OUTPUT_SIZE_MAXIMUM, "Flat JSON path has complete width.")
	aver.Always(cap(value) == len(value), "Flat JSON path cannot grow.")
}

// Tag_Buffer holds one decoded reflected tag value.
type Tag_Buffer []byte

// Tag_Buffer_Invariants fixes metadata scratch and prevents hidden growth.
func Tag_Buffer_Invariants(value Tag_Buffer, _ aver.Namespace) {
	aver.Always(len(value) == OUTPUT_SIZE_MAXIMUM, "Tag buffer has complete width.")
	aver.Always(cap(value) == len(value), "Tag buffer cannot grow.")
}

// Tag_Text is one nonempty bounded reflected tag suffix.
type Tag_Text string

// Tag_Text_Invariants bounds metadata parsing before every byte access.
func Tag_Text_Invariants(value Tag_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), KEY_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// TAG_NAME_POSITION_MAXIMUM is the final byte where a bounded delimiter can occur.
const TAG_NAME_POSITION_MAXIMUM = OUTPUT_SIZE_MAXIMUM - 1

// Tag_Name_Position is a key delimiter or the invalid zero result.
type Tag_Name_Position int

// Tag_Name_Position_Invariants includes every bounded metadata boundary.
func Tag_Name_Position_Invariants(value Tag_Name_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), strings.TEXT_SIZE_MINIMUM, TAG_NAME_POSITION_MAXIMUM).
		Ensure()
}

// Tag_Literal is one quoted json tag value within complete metadata storage.
type Tag_Literal string

// Tag_Literal_Invariants excludes unquoted and overbound metadata.
func Tag_Literal_Invariants(value Tag_Literal, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int(
			len(value), strings.TEXT_SIZE_MINIMUM, TAG_LITERAL_SIZE_MAXIMUM,
			IMPOSSIBLE_TAG_LITERAL_SIZE, IMPOSSIBLE_TAG_LITERAL_SIZE,
			IMPOSSIBLE_TAG_LITERAL_SIZE, IMPOSSIBLE_TAG_LITERAL_SIZE,
		).
		Ensure()
}

// Key_Position is one boundary inside bounded encoded output.
type Key_Position int

// Key_Position_Invariants starts after root framing and includes full output.
func Key_Position_Invariants(value Key_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), ENCODER_SIZE_MINIMUM, OUTPUT_SIZE_MAXIMUM).
		Ensure()
}

// Key_Positions holds one boundary for every possible flattened field.
type Key_Positions []Key_Position

// Key_Positions_Invariants fixes collision metadata storage and prevents hidden growth.
func Key_Positions_Invariants(value Key_Positions, _ aver.Namespace) {
	aver.Always(
		len(value) == FLATTENED_FIELD_COUNT_MAXIMUM,
		"Key positions have complete width.",
	)
	aver.Always(cap(value) == len(value), "Key positions cannot grow.")
}

// Key_Count includes empty objects and complete collision metadata storage.
type Key_Count int

// Key_Count_Invariants bounds flattened leaves independently from each struct width.
func Key_Count_Invariants(value Key_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), strings.TEXT_SIZE_MINIMUM, FLATTENED_FIELD_COUNT_MAXIMUM).
		Ensure()
}

// Collision_Data is object content after one field and before its closing byte.
type Collision_Data []byte

// Collision_Data_Invariants states the only builder span inspected for duplicate keys.
func Collision_Data_Invariants(value Collision_Data, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), FLATTENED_FIELD_SIZE_MINIMUM, COLLISION_DATA_SIZE_MAXIMUM,
		).
		Ensure()
}

// KEY_SIZE_MINIMUM is one exported Go field name byte.
const KEY_SIZE_MINIMUM = 1

// Key is one nonempty bounded flattened field path.
type Key string

// Key_Invariants rejects empty object member names from generated field paths.
func Key_Invariants(value Key, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), KEY_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Json_Text is bounded string content before JSON quoting.
type Json_Text string

// Json_Text_Invariants includes empty string values and largest hostile inputs.
func Json_Text_Invariants(value Json_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), strings.TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// NONEMPTY_TEXT_SIZE_MINIMUM is one output byte.
const NONEMPTY_TEXT_SIZE_MINIMUM = 1

// Nonempty_Text is one bounded fragment copied into the encoder.
type Nonempty_Text string

// Nonempty_Text_Invariants rejects writes that cannot advance output.
func Nonempty_Text_Invariants(value Nonempty_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), NONEMPTY_TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Frame owns one explicit DFS position so no recursive call can consume an unbounded stack.
type Frame struct {
	// Structure stays reflective while a nonnull source owns field values.
	Structure reflect.Value
	// Structure_Type keeps nil subtrees traversable without constructing zero values.
	Structure_Type reflect.Type
	// Index includes the end boundary needed to pop a completed frame.
	Index Field_Position
	// Prefix_Size restores active path after child frame pops.
	Prefix_Size Prefix_Size
	// Null preserves leaf keys beneath an absent pointer.
	Null strings.Boolean
}

// Frame_Invariants composes one bounded traversal position.
func Frame_Invariants(value Frame, namespace aver.Namespace) {
	aver.Always(value.Structure_Type != nil, "A traversal frame holds structure metadata.")
	aver.Always(
		bool(value.Null) == !value.Structure.IsValid(),
		"Only null traversal frames omit fabricated values.",
	)
	Field_Position_Invariants(value.Index, namespace)
	Prefix_Size_Invariants(value.Prefix_Size, namespace)
	strings.Boolean_Invariants(value.Null, namespace)
}

// Frames is complete caller-owned traversal storage.
type Frames []Frame

// Frames_Invariants fixes traversal capacity so descent cannot grow hidden storage.
func Frames_Invariants(value Frames, _ aver.Namespace) {
	aver.Always(len(value) == FRAME_COUNT_MAXIMUM, "Traversal storage has complete width.")
	aver.Always(cap(value) == len(value), "Traversal storage cannot grow.")
}

// Active_Frame owns one field proven inside its structure metadata.
type Active_Frame struct {
	// Structure stays reflective while a nonnull source owns field values.
	Structure reflect.Value
	// Structure_Type keeps nil subtrees traversable without constructing zero values.
	Structure_Type reflect.Type
	// Index has already been proven to select a field.
	Index Field_Index
	// Prefix_Size restores the active path after a child frame pops.
	Prefix_Size Prefix_Size
	// Null preserves leaf keys beneath an absent pointer.
	Null strings.Boolean
}

// Active_Frame_Invariants narrows traversal from completed position to one field.
func Active_Frame_Invariants(value Active_Frame, namespace aver.Namespace) {
	aver.Always(value.Structure_Type != nil, "An active frame holds structure metadata.")
	aver.Always(
		bool(value.Null) == !value.Structure.IsValid(),
		"Only null active frames omit fabricated values.",
	)
	Field_Index_Invariants(value.Index, namespace)
	Prefix_Size_Invariants(value.Prefix_Size, namespace)
	strings.Boolean_Invariants(value.Null, namespace)
}

// Frame_Count is active caller-stack traversal depth.
type Frame_Count int

// Frame_Count_Invariants prevents reflection depth from exhausting process stack.
func Frame_Count_Invariants(value Frame_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), FRAME_COUNT_MINIMUM, FRAME_COUNT_MAXIMUM).
		Ensure()
}

// ENCODER_SIZE_MINIMUM is one emitted framing byte.
const ENCODER_SIZE_MINIMUM = strings.TEXT_SIZE_MINIMUM + 1

// Encoder_State preserves one stack-owned builder identity across encoding stages.
type Encoder_State unsafe.Pointer

// Encoder_State_Invariants rejects missing caller-owned builder storage.
func Encoder_State_Invariants(value Encoder_State, _ aver.Namespace) {
	aver.Always(value != nil, "An Encoder state has storage.")
}

// Empty_Encoder_Size is builder position before root framing.
type Empty_Encoder_Size int

// Empty_Encoder_Size_Invariants fixes root entry at empty storage.
func Empty_Encoder_Size_Invariants(value Empty_Encoder_Size, _ aver.Namespace) {
	aver.Always(
		int(value) == strings.TEXT_SIZE_MINIMUM,
		"Empty Encoder size is zero.",
	)
}

// Started_Encoder_Size follows root framing through complete bounded output.
type Started_Encoder_Size int

// Started_Encoder_Size_Invariants rejects state before root framing.
func Started_Encoder_Size_Invariants(
	value Started_Encoder_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), ENCODER_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Write_Encoder_Size admits first root framing and final full storage.
type Write_Encoder_Size int

// Write_Encoder_Size_Invariants covers every write position.
func Write_Encoder_Size_Invariants(value Write_Encoder_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), strings.TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM,
		).
		Ensure()
}

// Empty_Encoder binds empty root state to caller-owned builder storage.
type Empty_Encoder struct {
	// State retains stack-owned builder identity.
	State Encoder_State
	// Size proves no root framing exists.
	Size Empty_Encoder_Size
}

// Empty_Encoder_Invariants keeps size evidence synchronized with pointed storage.
func Empty_Encoder_Invariants(value Empty_Encoder, namespace aver.Namespace) {
	Encoder_State_Invariants(value.State, namespace)
	Empty_Encoder_Size_Invariants(value.Size, namespace)
	builder := (*strings.Builder)(unsafe.Pointer(value.State))
	aver.Always(
		int(value.Size) == int(builder.Size),
		"Empty Encoder size matches builder position.",
	)
}

// Started_Encoder binds framed output state to caller-owned builder storage.
type Started_Encoder struct {
	// State retains stack-owned builder identity.
	State Encoder_State
	// Size proves root framing exists.
	Size Started_Encoder_Size
}

// Started_Encoder_Invariants keeps size evidence synchronized with pointed storage.
func Started_Encoder_Invariants(value Started_Encoder, namespace aver.Namespace) {
	Encoder_State_Invariants(value.State, namespace)
	Started_Encoder_Size_Invariants(value.Size, namespace)
	builder := (*strings.Builder)(unsafe.Pointer(value.State))
	aver.Always(
		int(value.Size) == int(builder.Size),
		"Started Encoder size matches builder position.",
	)
}

// Write_Encoder binds any writable position to caller-owned builder storage.
type Write_Encoder struct {
	// State retains stack-owned builder identity.
	State Encoder_State
	// Size records current write position.
	Size Write_Encoder_Size
}

// Write_Encoder_Invariants keeps size evidence synchronized with pointed storage.
func Write_Encoder_Invariants(value Write_Encoder, namespace aver.Namespace) {
	Encoder_State_Invariants(value.State, namespace)
	Write_Encoder_Size_Invariants(value.Size, namespace)
	builder := (*strings.Builder)(unsafe.Pointer(value.State))
	aver.Always(
		int(value.Size) == int(builder.Size),
		"Write Encoder size matches builder position.",
	)
}

func empty_encoder(state Encoder_State) (encoder Empty_Encoder) {
	defer func() { Empty_Encoder_Invariants(encoder, "empty_encoder.encoder") }()
	Encoder_State_Invariants(state, "empty_encoder.state")
	builder := (*strings.Builder)(unsafe.Pointer(state))
	return Empty_Encoder{
		State: state, Size: Empty_Encoder_Size(builder.Size),
	}
}

func started_encoder(state Encoder_State) (encoder Started_Encoder) {
	defer func() { Started_Encoder_Invariants(encoder, "started_encoder.encoder") }()
	Encoder_State_Invariants(state, "started_encoder.state")
	builder := (*strings.Builder)(unsafe.Pointer(state))
	return Started_Encoder{
		State: state, Size: Started_Encoder_Size(builder.Size),
	}
}

func write_encoder(state Encoder_State) (encoder Write_Encoder) {
	defer func() { Write_Encoder_Invariants(encoder, "write_encoder.encoder") }()
	Encoder_State_Invariants(state, "write_encoder.state")
	builder := (*strings.Builder)(unsafe.Pointer(state))
	return Write_Encoder{
		State: state, Size: Write_Encoder_Size(builder.Size),
	}
}

// Reflection_State keeps one synchronous reflected value behind explicit local storage.
type Reflection_State unsafe.Pointer

// Reflection_State_Invariants rejects missing reflected storage.
func Reflection_State_Invariants(value Reflection_State, _ aver.Namespace) {
	aver.Always(value != nil, "Reflected value has storage.")
}

// Marshal_Into encodes through fixed scratch before changing caller output.
func Marshal_Into(destination Output, value Value) (data Data, err Error) {
	defer func() { Data_Invariants(data, "marshal_into.data") }()
	Output_Invariants(destination, "marshal_into.destination")
	Value_Invariants(value, "marshal_into.value")
	root := reflect.Value(value)
	for depth := 0; root.IsValid() && root.Kind() == reflect.Pointer; depth++ {
		if depth == FRAME_COUNT_MAXIMUM {
			return nil, ERR_POINTER_DEPTH_EXCEEDED
		}
		if root.IsNil() {
			return nil, ERR_NIL_ROOT
		}
		root = root.Elem()
	}
	if !root.IsValid() {
		return nil, ERR_ROOT_TYPE
	}
	var builder strings.Builder
	state := Encoder_State(unsafe.Pointer(&builder))
	root_state := Reflection_State(unsafe.Pointer(&root))
	switch root.Kind() {
	case reflect.Slice, reflect.Array:
		err = marshal_array(empty_encoder(state), root_state)
	case reflect.Struct:
		err = marshal_object(empty_encoder(state), root_state)
	default:
		return nil, ERR_ROOT_TYPE
	}
	if err != nil {
		return nil, err
	}
	content := strings.Builder_Bytes(&builder)
	if len(destination) < len(content) {
		return nil, ERR_OUTPUT_TOO_SMALL
	}
	copy(destination, content)
	return Data(destination[:len(content)]), nil
}

// Marshal_Write keeps writer failure separate from reflection failure.
func Marshal_Write(write Write, destination Output, value Value) (err Error) {
	Write_Invariants(write, "marshal_write.write")
	Output_Invariants(destination, "marshal_write.destination")
	Value_Invariants(value, "marshal_write.value")
	if write == nil {
		return ERR_WRITE_NIL
	}
	data, marshal_err := Marshal_Into(destination, value)
	if marshal_err != nil {
		return marshal_err
	}
	written, write_err := write(data)
	if write_err != nil {
		return write_err
	}
	if written != len(data) {
		return ERR_SHORT_WRITE
	}
	return nil
}

func marshal_object(encoder Empty_Encoder, state Reflection_State) (err Error) {
	Empty_Encoder_Invariants(encoder, "marshal_object.encoder")
	Reflection_State_Invariants(state, "marshal_object.state")
	root := *(*reflect.Value)(unsafe.Pointer(state))
	builder := (*strings.Builder)(unsafe.Pointer(encoder.State))
	if root.NumField() > STRUCTURE_FIELD_COUNT_MAXIMUM {
		return ERR_STRUCTURE_FIELD_COUNT
	}
	if err = write_text(write_encoder(encoder.State), "{"); err != nil {
		return err
	}
	if err = flatten_struct(started_encoder(encoder.State), root); err != nil {
		return err
	}
	if err = write_text(write_encoder(encoder.State), "}"); err != nil {
		return err
	}
	content := strings.Builder_Bytes(builder)
	aver.Always(content[0] == '{', "A marshalled object opens with a brace.")
	aver.Always(
		content[len(content)-1] == '}', "A marshalled object closes with a brace.",
	)
	return nil
}

func marshal_array(encoder Empty_Encoder, state Reflection_State) (err Error) {
	Empty_Encoder_Invariants(encoder, "marshal_array.encoder")
	Reflection_State_Invariants(state, "marshal_array.state")
	root := *(*reflect.Value)(unsafe.Pointer(state))
	builder := (*strings.Builder)(unsafe.Pointer(encoder.State))
	if root.Len() > strings.TEXT_SIZE_MAXIMUM {
		return ERR_ARRAY_ELEMENT_COUNT
	}
	if err = write_text(write_encoder(encoder.State), "["); err != nil {
		return err
	}
	for index := 0; index < root.Len(); index++ {
		if index > 0 {
			if err = write_text(write_encoder(encoder.State), ","); err != nil {
				return err
			}
		}
		if err = marshal_element(
			started_encoder(encoder.State), root.Index(index),
		); err != nil {
			return err
		}
	}
	if err = write_text(write_encoder(encoder.State), "]"); err != nil {
		return err
	}
	content := strings.Builder_Bytes(builder)
	aver.Always(content[0] == '[', "A marshalled array opens with a bracket.")
	aver.Always(
		content[len(content)-1] == ']', "A marshalled array closes with a bracket.",
	)
	return nil
}

func marshal_element(encoder Started_Encoder, value reflect.Value) (err Error) {
	Started_Encoder_Invariants(encoder, "marshal_element.encoder")
	for depth := 0; value.Kind() == reflect.Pointer; depth++ {
		if depth == FRAME_COUNT_MAXIMUM {
			return ERR_POINTER_DEPTH_EXCEEDED
		}
		if value.IsNil() {
			return write_text(write_encoder(encoder.State), "null")
		}
		value = value.Elem()
	}
	if is_leaf(value.Type()) {
		return marshal_flat_value(started_encoder(encoder.State), value)
	}
	if value.Kind() != reflect.Struct {
		return ERR_ARRAY_ELEMENT_TYPE
	}
	if err = write_text(write_encoder(encoder.State), "{"); err != nil {
		return err
	}
	if err = flatten_struct(started_encoder(encoder.State), value); err != nil {
		return err
	}
	return write_text(write_encoder(encoder.State), "}")
}

func marshal_flat_value(encoder Started_Encoder, value reflect.Value) (err Error) {
	Started_Encoder_Invariants(encoder, "marshal_flat_value.encoder")
	for depth := 0; value.Kind() == reflect.Pointer; depth++ {
		if depth == FRAME_COUNT_MAXIMUM {
			return ERR_POINTER_DEPTH_EXCEEDED
		}
		if value.IsNil() {
			return write_text(write_encoder(encoder.State), "null")
		}
		value = value.Elem()
	}
	switch value.Kind() {
	case reflect.Slice, reflect.Array:
	default:
		return marshal_scalar(started_encoder(encoder.State), value)
	}
	if value.Kind() == reflect.Slice {
		if value.IsNil() {
			return write_text(write_encoder(encoder.State), "null")
		}
	}
	if value.Len() > strings.TEXT_SIZE_MAXIMUM {
		return ERR_SCALAR_ELEMENT_COUNT
	}
	if err = write_text(write_encoder(encoder.State), "["); err != nil {
		return err
	}
	for index := 0; index < value.Len(); index++ {
		if index > 0 {
			if err = write_text(write_encoder(encoder.State), ","); err != nil {
				return err
			}
		}
		if err = marshal_scalar(
			started_encoder(encoder.State), value.Index(index),
		); err != nil {
			return err
		}
	}
	return write_text(write_encoder(encoder.State), "]")
}

func marshal_scalar(encoder Started_Encoder, value reflect.Value) (err Error) {
	Started_Encoder_Invariants(encoder, "marshal_scalar.encoder")
	for depth := 0; value.Kind() == reflect.Pointer; depth++ {
		if depth == FRAME_COUNT_MAXIMUM {
			return ERR_POINTER_DEPTH_EXCEEDED
		}
		if value.IsNil() {
			return write_text(write_encoder(encoder.State), "null")
		}
		value = value.Elem()
	}
	switch value.Kind() {
	case reflect.Bool:
		if value.Bool() {
			return write_text(write_encoder(encoder.State), "true")
		}
		return write_text(write_encoder(encoder.State), "false")
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var storage [strconv.INTEGER_TEXT_SIZE_MAXIMUM]byte
		number := strconv.Signed_Integer(value.Int())
		count := strconv.Format_Integer_Into(storage[:], number, NUMBER_BASE)
		text := unsafe.String(&storage[0], count)
		return write_text(write_encoder(encoder.State), Nonempty_Text(text))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var storage [strconv.DIGIT_TEXT_SIZE_MAXIMUM]byte
		number := strconv.Unsigned_Integer(value.Uint())
		count := strconv.Format_Unsigned_Integer_Into(storage[:], number, NUMBER_BASE)
		text := unsafe.String(&storage[0], count)
		return write_text(write_encoder(encoder.State), Nonempty_Text(text))
	case reflect.String:
		if value.Len() > strings.TEXT_SIZE_MAXIMUM {
			return ERR_STRING_SIZE
		}
		return append_json_string(started_encoder(encoder.State), Json_Text(value.String()))
	}
	return ERR_VALUE_NOT_FLAT
}

func flatten_struct(encoder Started_Encoder, root reflect.Value) (err Error) {
	Started_Encoder_Invariants(encoder, "flatten_struct.encoder")
	frame_storage := [FRAME_COUNT_MAXIMUM]Frame{{Structure: root, Structure_Type: root.Type()}}
	var path_storage, tag_storage [OUTPUT_SIZE_MAXIMUM]byte
	path, tag_buffer := Path(path_storage[:]), Tag_Buffer(tag_storage[:])
	var key_starts, key_ends [FLATTENED_FIELD_COUNT_MAXIMUM]Key_Position
	frame_count, key_count := FRAME_COUNT_MINIMUM, strings.TEXT_SIZE_MINIMUM
	for frame_count > 0 {
		Frame_Count_Invariants(Frame_Count(frame_count), "flatten_struct.frame_count")
		depth, current := frame_count-1, frame_storage[frame_count-1]
		Frame_Invariants(current, "flatten_struct.frame")
		if int(current.Index) >= current.Structure_Type.NumField() {
			frame_count = depth
			continue
		}
		frame_storage[depth].Index++
		active := Active_Frame{
			Structure: current.Structure, Structure_Type: current.Structure_Type,
			Index: Field_Index(current.Index), Prefix_Size: current.Prefix_Size,
			Null: current.Null,
		}
		field, value, text, skip, field_err := frame_field(active, tag_buffer)
		if field_err != nil {
			return field_err
		}
		if skip {
			continue
		}
		name := Key(text)
		structure, structure_type, prefix, null, descend, child_err := nested_frame(
			field, value, name, path, current.Prefix_Size, current.Null)
		if child_err != nil {
			return child_err
		}
		if descend {
			next_count, push_err := push_frame(
				Frames(frame_storage[:]), Frame_Count(frame_count),
				structure, structure_type, prefix, null)
			if push_err != nil {
				return push_err
			}
			frame_count = int(next_count)
			continue
		}
		key_start, key_end, leaf_err := flatten_leaf(
			started_encoder(encoder.State), path, current.Prefix_Size,
			name, value, current.Null)
		if leaf_err != nil {
			return leaf_err
		}
		content := strings.Builder_Bytes((*strings.Builder)(unsafe.Pointer(encoder.State)))
		if duplicate_key(
			Collision_Data(content), Key_Positions(key_starts[:]),
			Key_Positions(key_ends[:]),
			Key_Count(key_count), key_start, key_end,
		) {
			return ERR_DUPLICATE_KEY
		}
		if key_count == FLATTENED_FIELD_COUNT_MAXIMUM {
			return ERR_FLATTENED_FIELD_COUNT
		}
		key_starts[key_count], key_ends[key_count] = key_start, key_end
		key_count++
	}
	return nil
}

func push_frame(
	storage Frames, count Frame_Count,
	structure reflect.Value, structure_type reflect.Type,
	prefix Prefix_Size, null strings.Boolean,
) (next_count Frame_Count, err Error) {
	defer func() { Frame_Count_Invariants(next_count, "push_frame.next_count") }()
	Frames_Invariants(storage, "push_frame.storage")
	Frame_Count_Invariants(count, "push_frame.count")
	Prefix_Size_Invariants(prefix, "push_frame.prefix")
	strings.Boolean_Invariants(null, "push_frame.null")
	if count == FRAME_COUNT_MAXIMUM {
		return count, ERR_STRUCTURE_DEPTH
	}
	if structure_type.NumField() > STRUCTURE_FIELD_COUNT_MAXIMUM {
		return count, ERR_STRUCTURE_FIELD_COUNT
	}
	storage[int(count)] = Frame{
		Structure: structure, Structure_Type: structure_type,
		Prefix_Size: prefix, Null: null,
	}
	return count + 1, nil
}

func frame_field(
	frame Active_Frame, tag_buffer Tag_Buffer,
) (
	field reflect.StructField, value reflect.Value, name Json_Text,
	skip strings.Boolean, err Error,
) {
	defer func() {
		Json_Text_Invariants(name, "frame_field.name")
		strings.Boolean_Invariants(skip, "frame_field.skip")
	}()
	Active_Frame_Invariants(frame, "frame_field.frame")
	Tag_Buffer_Invariants(tag_buffer, "frame_field.tag_buffer")
	field = frame.Structure_Type.Field(int(frame.Index))
	if field.PkgPath != "" {
		return field, reflect.Value{}, "", true, nil
	}
	name, skip, err = field_json(field, tag_buffer)
	if err != nil {
		return field, reflect.Value{}, name, skip, err
	}
	if !frame.Null {
		value = frame.Structure.Field(int(frame.Index))
	}
	return field, value, name, skip, nil
}

func duplicate_key(
	content Collision_Data, starts Key_Positions, ends Key_Positions, count Key_Count,
	start Key_Position, end Key_Position,
) (duplicate strings.Boolean) {
	defer func() { strings.Boolean_Invariants(duplicate, "duplicate_key.duplicate") }()
	Collision_Data_Invariants(content, "duplicate_key.content")
	Key_Positions_Invariants(starts, "duplicate_key.starts")
	Key_Positions_Invariants(ends, "duplicate_key.ends")
	Key_Count_Invariants(count, "duplicate_key.count")
	Key_Position_Invariants(start, "duplicate_key.start")
	Key_Position_Invariants(end, "duplicate_key.end")
	for previous := strings.TEXT_SIZE_MINIMUM; previous < int(count); previous++ {
		current := bytes.Slice(content[start:end])
		prior := bytes.Slice(content[starts[previous]:ends[previous]])
		if bytes.Equal(current, prior) {
			return true
		}
	}
	return false
}

func nested_frame(
	field reflect.StructField, value reflect.Value, name Key,
	path Path, prefix Prefix_Size, parent_null strings.Boolean,
) (
	structure reflect.Value, structure_type reflect.Type, child_prefix Prefix_Size,
	child_null strings.Boolean, descend strings.Boolean, err Error,
) {
	defer func() {
		Prefix_Size_Invariants(child_prefix, "nested_frame.child_prefix")
		strings.Boolean_Invariants(child_null, "nested_frame.child_null")
		strings.Boolean_Invariants(descend, "nested_frame.descend")
	}()
	Key_Invariants(name, "nested_frame.name")
	Path_Invariants(path, "nested_frame.path")
	Prefix_Size_Invariants(prefix, "nested_frame.prefix")
	strings.Boolean_Invariants(parent_null, "nested_frame.parent_null")
	child_prefix = prefix
	child_null = parent_null
	if !is_embedded_struct(field) {
		if is_leaf(field.Type) {
			return reflect.Value{}, nil, child_prefix, child_null, false, nil
		}
		nested_size := int(prefix) + len(name) + len(KEY_SEPARATOR)
		if nested_size > OUTPUT_SIZE_MAXIMUM {
			return reflect.Value{}, nil, child_prefix, child_null, false,
				ERR_FLATTENED_KEY_SIZE
		}
		copy(path[prefix:], name)
		path[int(prefix)+len(name)] = KEY_SEPARATOR[0]
		child_prefix = Prefix_Size(nested_size)
	}
	child, child_type, is_null, deref_err := deref_struct(value, field.Type)
	if deref_err != nil {
		return reflect.Value{}, nil, child_prefix, child_null, false, deref_err
	}
	if child_type.Kind() != reflect.Struct {
		return reflect.Value{}, nil, child_prefix, child_null, false,
			ERR_DESCENT_TYPE
	}
	child_null = strings.Boolean(bool(is_null) || bool(parent_null))
	return child, child_type, child_prefix, child_null, true, nil
}

func deref_struct(
	value reflect.Value, value_type reflect.Type,
) (structure reflect.Value, structure_type reflect.Type, null strings.Boolean, err Error) {
	defer func() { strings.Boolean_Invariants(null, "deref_struct.null") }()
	for depth := 0; value_type.Kind() == reflect.Pointer; depth++ {
		if depth == FRAME_COUNT_MAXIMUM {
			return reflect.Value{}, nil, false, ERR_POINTER_DEPTH_EXCEEDED
		}
		value_type = value_type.Elem()
		if value.IsValid() {
			if value.IsNil() {
				value = reflect.Value{}
			} else {
				value = value.Elem()
			}
		}
	}
	return value, value_type, strings.Boolean(!value.IsValid()), nil
}

func flatten_leaf(
	encoder Started_Encoder, path Path, prefix Prefix_Size,
	name Key, value reflect.Value, null strings.Boolean,
) (key_start Key_Position, key_end Key_Position, err Error) {
	defer func() {
		Key_Position_Invariants(key_start, "flatten_leaf.key_start")
		Key_Position_Invariants(key_end, "flatten_leaf.key_end")
	}()
	Started_Encoder_Invariants(encoder, "flatten_leaf.encoder")
	Path_Invariants(path, "flatten_leaf.path")
	Prefix_Size_Invariants(prefix, "flatten_leaf.prefix")
	Key_Invariants(name, "flatten_leaf.name")
	strings.Boolean_Invariants(null, "flatten_leaf.null")
	builder := (*strings.Builder)(unsafe.Pointer(encoder.State))
	key_start = Key_Position(builder.Size)
	key_end = key_start
	if !null {
		if !is_flat_leaf(value.Type()) {
			return key_start, key_end, ERR_VALUE_NOT_FLAT
		}
	}
	content := strings.Builder_Bytes(builder)
	if content[len(content)-1] != '{' {
		if err = write_text(write_encoder(encoder.State), ","); err != nil {
			return key_start, key_end, err
		}
	}
	key_size := int(prefix) + len(name)
	if key_size > OUTPUT_SIZE_MAXIMUM {
		return key_start, key_end, ERR_FLATTENED_KEY_SIZE
	}
	var key_storage [OUTPUT_SIZE_MAXIMUM]byte
	copy(key_storage[:], path[:prefix])
	copy(key_storage[prefix:], name)
	key := unsafe.String(&key_storage[0], key_size)
	key_start = Key_Position(builder.Size)
	if err = append_json_string(started_encoder(encoder.State), Json_Text(key)); err != nil {
		return key_start, key_end, err
	}
	key_end = Key_Position(builder.Size)
	if err = write_text(write_encoder(encoder.State), ":"); err != nil {
		return key_start, key_end, err
	}
	if null {
		err = write_text(write_encoder(encoder.State), "null")
		return key_start, key_end, err
	}
	if !value.CanInterface() {
		return key_start, key_end,
			ERR_UNEXPORTED_LEAF
	}
	err = marshal_flat_value(started_encoder(encoder.State), value)
	return key_start, key_end, err
}

func append_json_string(encoder Started_Encoder, text Json_Text) (err Error) {
	Started_Encoder_Invariants(encoder, "append_json_string.encoder")
	Json_Text_Invariants(text, "append_json_string.text")
	if err = write_text(write_encoder(encoder.State), "\""); err != nil {
		return err
	}
	const HEXADECIMAL = "0123456789abcdef"
	for _, character := range text {
		switch character {
		case '"', '\\':
			escaped := Nonempty_Text([]rune{'\\', character})
			if err = write_text(write_encoder(encoder.State), escaped); err != nil {
				return err
			}
		case '\b':
			err = write_text(write_encoder(encoder.State), "\\b")
		case '\f':
			err = write_text(write_encoder(encoder.State), "\\f")
		case '\n':
			err = write_text(write_encoder(encoder.State), "\\n")
		case '\r':
			err = write_text(write_encoder(encoder.State), "\\r")
		case '\t':
			err = write_text(write_encoder(encoder.State), "\\t")
		case '<', '>', '&', '\u2028', '\u2029':
			var escaped [JSON_ESCAPE_SIZE]byte
			escaped[0] = '\\'
			escaped[1] = 'u'
			escaped[2] = HEXADECIMAL[character>>12]
			escaped[3] = HEXADECIMAL[character>>8&0x0f]
			escaped[4] = HEXADECIMAL[character>>4&0x0f]
			escaped[5] = HEXADECIMAL[character&0x0f]
			err = write_text(write_encoder(encoder.State), Nonempty_Text(escaped[:]))
		default:
			if character < ' ' {
				var escaped [JSON_ESCAPE_SIZE]byte
				escaped[0] = '\\'
				escaped[1] = 'u'
				escaped[2] = '0'
				escaped[3] = '0'
				escaped[4] = HEXADECIMAL[character>>4&0x0f]
				escaped[5] = HEXADECIMAL[character&0x0f]
				err = write_text(
					write_encoder(encoder.State), Nonempty_Text(escaped[:]),
				)
			} else {
				err = write_text(
					write_encoder(encoder.State),
					Nonempty_Text(string(character)),
				)
			}
		}
		if err != nil {
			return err
		}
	}
	return write_text(write_encoder(encoder.State), "\"")
}

func write_text(encoder Write_Encoder, text Nonempty_Text) (err Error) {
	Write_Encoder_Invariants(encoder, "write_text.encoder")
	Nonempty_Text_Invariants(text, "write_text.text")
	builder := (*strings.Builder)(unsafe.Pointer(encoder.State))
	if len(text) > strings.TEXT_SIZE_MAXIMUM-int(builder.Size) {
		return ERR_ENCODED_OUTPUT_SIZE
	}
	strings.Builder_Write_Text(builder, strings.Text(text))
	return nil
}

func field_json(
	field reflect.StructField, destination Tag_Buffer,
) (name Json_Text, skip strings.Boolean, err Error) {
	defer func() {
		Json_Text_Invariants(name, "field_json.name")
		strings.Boolean_Invariants(skip, "field_json.skip")
	}()
	Tag_Buffer_Invariants(destination, "field_json.destination")
	selected := field.Name
	literal, found, tag_err := json_tag_literal(field.Tag)
	if tag_err != nil {
		return "", false, tag_err
	}
	if found {
		count, unquote_err := strconv.Unquote_Into(
			strconv.Buffer(destination), strconv.Text(literal),
		)
		if unquote_err != nil {
			return "", false, ERR_FIELD_TAG_SYNTAX
		}
		tag := unsafe.String(&destination[0], int(count))
		leaf := tag
		for index := 0; index < len(tag); index++ {
			if tag[index] == ',' {
				leaf = tag[:index]
				break
			}
		}
		if leaf == "-" {
			return "", true, nil
		}
		if leaf != "" {
			selected = leaf
		}
	}
	if len(selected) > strings.TEXT_SIZE_MAXIMUM {
		return "", false, ERR_FIELD_NAME_SIZE
	}
	return Json_Text(selected), false, nil
}

func json_tag_literal(
	tag reflect.StructTag,
) (literal Tag_Literal, found strings.Boolean, err Error) {
	defer func() {
		Tag_Literal_Invariants(literal, "json_tag_literal.literal")
		strings.Boolean_Invariants(found, "json_tag_literal.found")
	}()
	suffix := string(tag)
	if len(suffix) > strings.TEXT_SIZE_MAXIMUM {
		return "", false, ERR_FIELD_TAG_SIZE
	}
	for len(suffix) > 0 {
		for len(suffix) > 0 {
			if suffix[0] != ' ' {
				break
			}
			suffix = suffix[1:]
		}
		if len(suffix) == 0 {
			return "", false, nil
		}
		name_end, valid := tag_name_end(Tag_Text(suffix))
		if !valid {
			return "", false, ERR_FIELD_TAG_SYNTAX
		}
		value := suffix[int(name_end)+1:]
		quoted, quote_err := strconv.Quoted_Prefix(strconv.Text(value))
		if quote_err != nil {
			return "", false, ERR_FIELD_TAG_SYNTAX
		}
		if suffix[:name_end] == "json" {
			return Tag_Literal(quoted), true, nil
		}
		suffix = value[len(quoted):]
	}
	return "", false, nil
}

func tag_name_end(text Tag_Text) (end Tag_Name_Position, valid strings.Boolean) {
	defer func() {
		Tag_Name_Position_Invariants(end, "tag_name_end.end")
		strings.Boolean_Invariants(valid, "tag_name_end.valid")
	}()
	Tag_Text_Invariants(text, "tag_name_end.text")
	for index := 0; index < len(text); index++ {
		if text[index] == ':' {
			return Tag_Name_Position(index), strings.Boolean(index > 0)
		}
		if text[index] <= ' ' {
			return 0, false
		}
		if text[index] == '"' {
			return 0, false
		}
	}
	return 0, false
}

func is_embedded_struct(field reflect.StructField) (embedded strings.Boolean) {
	defer func() {
		strings.Boolean_Invariants(embedded, "is_embedded_struct.embedded")
	}()
	if !field.Anonymous {
		return false
	}
	return !is_leaf(field.Type)
}

func is_leaf(value_type reflect.Type) (leaf strings.Boolean) {
	defer func() { strings.Boolean_Invariants(leaf, "is_leaf.leaf") }()
	base := value_type
	for depth := 0; base.Kind() == reflect.Pointer; depth++ {
		if depth == FRAME_COUNT_MAXIMUM {
			return false
		}
		base = base.Elem()
	}
	if has_methods(base) {
		return true
	}
	return strings.Boolean(base.Kind() != reflect.Struct)
}

func is_flat_leaf(value_type reflect.Type) (flat strings.Boolean) {
	defer func() { strings.Boolean_Invariants(flat, "is_flat_leaf.flat") }()
	base := value_type
	for depth := 0; base.Kind() == reflect.Pointer; depth++ {
		if depth == FRAME_COUNT_MAXIMUM {
			return false
		}
		base = base.Elem()
	}
	if has_methods(base) {
		return false
	}
	if is_scalar_kind(base.Kind()) {
		return true
	}
	switch base.Kind() {
	case reflect.Slice, reflect.Array:
		return is_scalar_element(base.Elem())
	}
	return false
}

func is_scalar_element(value_type reflect.Type) (scalar strings.Boolean) {
	defer func() { strings.Boolean_Invariants(scalar, "is_scalar_element.scalar") }()
	base := value_type
	for depth := 0; base.Kind() == reflect.Pointer; depth++ {
		if depth == FRAME_COUNT_MAXIMUM {
			return false
		}
		base = base.Elem()
	}
	if has_methods(base) {
		return false
	}
	return is_scalar_kind(base.Kind())
}

func has_methods(value_type reflect.Type) (methods strings.Boolean) {
	defer func() { strings.Boolean_Invariants(methods, "has_methods.methods") }()
	if value_type.NumMethod() > 0 {
		return true
	}
	return strings.Boolean(reflect.PointerTo(value_type).NumMethod() > 0)
}

func is_scalar_kind(kind reflect.Kind) (scalar strings.Boolean) {
	defer func() { strings.Boolean_Invariants(scalar, "is_scalar_kind.scalar") }()
	switch kind {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.String:
		return true
	}
	return false
}
