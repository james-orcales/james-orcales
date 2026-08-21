// Package flatjson writes one flat JSON object, or one array of flat objects.
// Nested struct paths become underscore-separated keys. Maps and nested arrays stay rejected.
package flatjson

import (
	"errors"
	"fmt"
	"reflect"

	"local/james-orcales/shared/simulation/aver/default"
	"local/james-orcales/shared/strconv"
	"local/james-orcales/shared/strings"
)

// KEY_SEPARATOR preserves the external flat-json path convention.
const KEY_SEPARATOR = "_"

// FRAME_COUNT_MAXIMUM bounds hostile reflected nesting independently from output size.
const FRAME_COUNT_MAXIMUM = 64

// FRAME_COUNT_MINIMUM keeps one root while traversal is active.
const FRAME_COUNT_MINIMUM = 1

// JSON_ESCAPE_SIZE is one forced four-digit Unicode escape.
const JSON_ESCAPE_SIZE = 6

// NUMBER_BASE fixes JSON integers to decimal text.
const NUMBER_BASE = strconv.DECIMAL_BASE

// MARSHALER_INPUT_COUNT includes one receiver.
const MARSHALER_INPUT_COUNT = 1

// MARSHALER_OUTPUT_COUNT includes bytes and error.
const MARSHALER_OUTPUT_COUNT = 2

// MARSHALER_BYTES_OUTPUT is encoded bytes.
const MARSHALER_BYTES_OUTPUT = 0

// MARSHALER_ERROR_OUTPUT is method failure.
const MARSHALER_ERROR_OUTPUT = 1

// MARSHAL_JSON_NAME avoids importing the interface package only to identify its method.
const MARSHAL_JSON_NAME = "MarshalJSON"

// MARSHAL_TEXT_NAME avoids importing the interface package only to identify its method.
const MARSHAL_TEXT_NAME = "MarshalText"

// IMPOSSIBLE_DATA_SIZE separates errors from the smallest valid JSON document.
const IMPOSSIBLE_DATA_SIZE = 1

// ERR_SHORT_WRITE distinguishes silent partial acceptance from an ordinary writer error.
var ERR_SHORT_WRITE = errors.New("flatjson: short write")

// Write keeps blocking ownership at the caller's composition boundary.
type Write func([]byte) (written int, err error)

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

// Field_Index includes root and final struct field boundaries.
type Field_Index int

// Field_Index_Invariants matches bounded reflected struct width.
func Field_Index_Invariants(value Field_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), strings.TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// IMPOSSIBLE_PREFIX_SIZE excludes a path that cannot hold a name and separator.
const IMPOSSIBLE_PREFIX_SIZE = 1

// Prefix is empty at root or one bounded ancestor path ending in a separator.
type Prefix string

// Prefix_Invariants excludes the one-byte shape no nested path can produce.
func Prefix_Invariants(value Prefix, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Holed_Int(
			len(value), strings.TEXT_SIZE_MINIMUM, strings.TEXT_SIZE_MAXIMUM,
			IMPOSSIBLE_PREFIX_SIZE, IMPOSSIBLE_PREFIX_SIZE,
			IMPOSSIBLE_PREFIX_SIZE, IMPOSSIBLE_PREFIX_SIZE,
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
	// Structure stays reflective because generated struct shapes have no static type.
	Structure reflect.Value
	// Index includes the end boundary needed to pop a completed frame.
	Index Field_Index
	// Prefix carries the already-validated ancestor path.
	Prefix Prefix
	// Null preserves leaf keys beneath an absent pointer.
	Null strings.Boolean
}

// Frame_Invariants composes one bounded traversal position.
func Frame_Invariants(value Frame, namespace aver.Namespace) {
	aver.Always(
		value.Structure.IsValid(), "A traversal frame holds a valid structure.",
	)
	Field_Index_Invariants(value.Index, namespace)
	Prefix_Invariants(value.Prefix, namespace)
	strings.Boolean_Invariants(value.Null, namespace)
}

// Frames is the single bounded ownership stack for one traversal.
type Frames []Frame

// Frames_Invariants prevents reflection depth from becoming process exhaustion.
func Frames_Invariants(value Frames, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), FRAME_COUNT_MINIMUM, FRAME_COUNT_MAXIMUM).
		Ensure()
}

// Encoder is the nonnil handle shared by bounded, nonrecursive encoding stages.
type Encoder *strings.Builder

// Encoder_Invariants rejects missing storage before any stage mutates it.
func Encoder_Invariants(value Encoder, _ aver.Namespace) {
	aver.Always(value != nil, "An Encoder has storage.")
}

// Marshaler_Kind selects direct JSON, quoted text, or ordinary scalar encoding.
type Marshaler_Kind int

// Marshaler_Kind_Invariants keeps method precedence explicit.
func Marshaler_Kind_Invariants(value Marshaler_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Int(
			int(value), int(MARSHALER_NONE), int(MARSHALER_JSON), int(MARSHALER_TEXT),
		).
		Ensure()
}

// Method_Kind selects one method that is known to exist.
type Method_Kind int

// Method_Kind_Invariants excludes ordinary scalar encoding from method invocation.
func Method_Kind_Invariants(value Method_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Int(int(value), int(MARSHALER_JSON), int(MARSHALER_TEXT)).
		Ensure()
}

// MARSHALER_NONE uses ordinary scalar encoding.
const MARSHALER_NONE Marshaler_Kind = 0

// MARSHALER_JSON delegates unquoted JSON.
const MARSHALER_JSON Marshaler_Kind = 1

// MARSHALER_TEXT delegates text and then quotes it.
const MARSHALER_TEXT Marshaler_Kind = 2

// Marshal owns output because callers cannot know reflection-expanded size beforehand.
func Marshal(value any) (data Data, err error) {
	defer func() { Data_Invariants(data, "marshal.data") }()
	root := reflect.ValueOf(value)
	for depth := 0; root.IsValid() && root.Kind() == reflect.Pointer; depth++ {
		if depth == FRAME_COUNT_MAXIMUM {
			return nil, errors.New("flatjson: pointer nesting exceeds limit")
		}
		if root.IsNil() {
			return nil, errors.New("flatjson: Marshal of a nil pointer")
		}
		root = root.Elem()
	}
	if !root.IsValid() {
		return nil, errors.New("flatjson: Marshal requires a struct or a slice of them")
	}
	var builder strings.Builder
	switch root.Kind() {
	case reflect.Slice, reflect.Array:
		err = marshal_array(&builder, root)
	case reflect.Struct:
		err = marshal_object(&builder, root)
	default:
		return nil, errors.New("flatjson: Marshal requires a struct or a slice of them")
	}
	if err != nil {
		return nil, err
	}
	return Data(strings.Builder_Bytes(&builder)), nil
}

// Marshal_Write keeps writer failure separate from reflection failure.
func Marshal_Write(write Write, value any) (err error) {
	data, marshal_err := Marshal(value)
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

func marshal_object(builder Encoder, root reflect.Value) (err error) {
	Encoder_Invariants(builder, "marshal_object.builder")
	if root.NumField() > strings.TEXT_SIZE_MAXIMUM {
		return errors.New("flatjson: structure field count exceeds limit")
	}
	if err = write_text(builder, "{"); err != nil {
		return err
	}
	if err = flatten_struct(builder, root); err != nil {
		return err
	}
	if err = write_text(builder, "}"); err != nil {
		return err
	}
	content := strings.Builder_Bytes(builder)
	aver.Always(content[0] == '{', "A marshalled object opens with a brace.")
	aver.Always(
		content[len(content)-1] == '}', "A marshalled object closes with a brace.",
	)
	return nil
}

func marshal_array(builder Encoder, root reflect.Value) (err error) {
	Encoder_Invariants(builder, "marshal_array.builder")
	if root.Len() > strings.TEXT_SIZE_MAXIMUM {
		return errors.New("flatjson: array element count exceeds limit")
	}
	if err = write_text(builder, "["); err != nil {
		return err
	}
	for index := 0; index < root.Len(); index++ {
		if index > 0 {
			if err = write_text(builder, ","); err != nil {
				return err
			}
		}
		if err = marshal_element(builder, root.Index(index)); err != nil {
			return err
		}
	}
	if err = write_text(builder, "]"); err != nil {
		return err
	}
	content := strings.Builder_Bytes(builder)
	aver.Always(content[0] == '[', "A marshalled array opens with a bracket.")
	aver.Always(
		content[len(content)-1] == ']', "A marshalled array closes with a bracket.",
	)
	return nil
}

func marshal_element(builder Encoder, value reflect.Value) (err error) {
	Encoder_Invariants(builder, "marshal_element.builder")
	for depth := 0; value.Kind() == reflect.Pointer; depth++ {
		if depth == FRAME_COUNT_MAXIMUM {
			return errors.New("flatjson: pointer nesting exceeds limit")
		}
		if value.IsNil() {
			return write_text(builder, "null")
		}
		value = value.Elem()
	}
	if is_leaf(value.Type()) {
		return marshal_flat_value(builder, value)
	}
	if value.Kind() != reflect.Struct {
		return errors.New("flatjson: array element is not flat")
	}
	if err = write_text(builder, "{"); err != nil {
		return err
	}
	if err = flatten_struct(builder, value); err != nil {
		return err
	}
	return write_text(builder, "}")
}

func marshal_flat_value(builder Encoder, value reflect.Value) (err error) {
	Encoder_Invariants(builder, "marshal_flat_value.builder")
	for depth := 0; value.Kind() == reflect.Pointer; depth++ {
		if depth == FRAME_COUNT_MAXIMUM {
			return errors.New("flatjson: pointer nesting exceeds limit")
		}
		if value.IsNil() {
			return write_text(builder, "null")
		}
		value = value.Elem()
	}
	switch value.Kind() {
	case reflect.Slice, reflect.Array:
	default:
		return marshal_scalar(builder, value)
	}
	if value.Kind() == reflect.Slice {
		if value.IsNil() {
			return write_text(builder, "null")
		}
	}
	if value.Len() > strings.TEXT_SIZE_MAXIMUM {
		return errors.New("flatjson: scalar array element count exceeds limit")
	}
	if err = write_text(builder, "["); err != nil {
		return err
	}
	for index := 0; index < value.Len(); index++ {
		if index > 0 {
			if err = write_text(builder, ","); err != nil {
				return err
			}
		}
		if err = marshal_scalar(builder, value.Index(index)); err != nil {
			return err
		}
	}
	return write_text(builder, "]")
}

func marshal_scalar(builder Encoder, value reflect.Value) (err error) {
	Encoder_Invariants(builder, "marshal_scalar.builder")
	for depth := 0; value.Kind() == reflect.Pointer; depth++ {
		if depth == FRAME_COUNT_MAXIMUM {
			return errors.New("flatjson: pointer nesting exceeds limit")
		}
		if value.IsNil() {
			return write_text(builder, "null")
		}
		value = value.Elem()
	}
	kind := marshaler_kind(value.Type())
	if kind != MARSHALER_NONE {
		return marshal_method(builder, value, Method_Kind(kind))
	}
	switch value.Kind() {
	case reflect.Bool:
		if value.Bool() {
			return write_text(builder, "true")
		}
		return write_text(builder, "false")
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var storage [strconv.INTEGER_TEXT_SIZE_MAXIMUM]byte
		number := strconv.Signed_Integer(value.Int())
		count := strconv.Format_Integer_Into(storage[:], number, NUMBER_BASE)
		return write_text(builder, Nonempty_Text(storage[:count]))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var storage [strconv.DIGIT_TEXT_SIZE_MAXIMUM]byte
		number := strconv.Unsigned_Integer(value.Uint())
		count := strconv.Format_Unsigned_Integer_Into(storage[:], number, NUMBER_BASE)
		return write_text(builder, Nonempty_Text(storage[:count]))
	case reflect.Float32, reflect.Float64:
		formatted := fmt.Sprint(value.Interface())
		switch formatted {
		case "NaN", "+Inf", "-Inf":
			return errors.New("flatjson: unsupported floating-point value")
		}
		return write_text(builder, Nonempty_Text(formatted))
	case reflect.String:
		if value.Len() > strings.TEXT_SIZE_MAXIMUM {
			return errors.New("flatjson: string exceeds limit")
		}
		return append_json_string(builder, Json_Text(value.String()))
	}
	return errors.New("flatjson: value is not flat")
}

// Reflection lookup retains json.Marshaler precedence without importing banned encoders.
func marshal_method(
	builder Encoder, value reflect.Value, kind Method_Kind,
) (err error) {
	Encoder_Invariants(builder, "marshal_method.builder")
	Method_Kind_Invariants(kind, "marshal_method.kind")
	name := MARSHAL_JSON_NAME
	if kind == Method_Kind(MARSHALER_TEXT) {
		name = MARSHAL_TEXT_NAME
	}
	receiver := value
	method := receiver.MethodByName(name)
	if !method.IsValid() {
		pointer := reflect.New(value.Type())
		pointer.Elem().Set(value)
		receiver = pointer
		method = receiver.MethodByName(name)
	}
	if !method.IsValid() {
		return errors.New("flatjson: marshaler method is unavailable")
	}
	results := method.Call(nil)
	if !results[MARSHALER_ERROR_OUTPUT].IsNil() {
		method_err, valid := results[MARSHALER_ERROR_OUTPUT].Interface().(error)
		if !valid {
			return errors.New("flatjson: marshaler returned a non-error failure")
		}
		return method_err
	}
	raw := results[MARSHALER_BYTES_OUTPUT].Bytes()
	if len(raw) > strings.TEXT_SIZE_MAXIMUM {
		return errors.New("flatjson: marshaler output exceeds limit")
	}
	if kind == Method_Kind(MARSHALER_TEXT) {
		return append_json_string(builder, Json_Text(raw))
	}
	if len(raw) == 0 {
		return errors.New("flatjson: marshaler returned empty JSON")
	}
	switch raw[0] {
	case '{', '[':
		return errors.New("flatjson: marshaler returned nested JSON")
	}
	return write_text(builder, Nonempty_Text(raw))
}

func flatten_struct(builder Encoder, root reflect.Value) (err error) {
	Encoder_Invariants(builder, "flatten_struct.builder")
	seen := map[Key]struct{}{}
	stack := Frames{{Structure: root}}
	for len(stack) > 0 {
		Frames_Invariants(stack, "flatten_struct.stack")
		depth := len(stack) - 1
		current := stack[depth]
		Frame_Invariants(current, "flatten_struct.frame")
		if int(current.Index) >= current.Structure.NumField() {
			stack = stack[:depth]
			continue
		}
		stack[depth].Index++
		field := current.Structure.Type().Field(int(current.Index))
		if field.PkgPath != "" {
			continue
		}
		field_name, skip, field_err := field_json(field)
		if field_err != nil {
			return field_err
		}
		if skip {
			continue
		}
		name := Key(field_name)
		value := current.Structure.Field(int(current.Index))
		structure, prefix, null, descend, child_err := nested_frame(
			field, value, name, current.Prefix, current.Null,
		)
		if child_err != nil {
			return child_err
		}
		if descend {
			if len(stack) == FRAME_COUNT_MAXIMUM {
				return errors.New("flatjson: structure nesting exceeds limit")
			}
			if structure.NumField() > strings.TEXT_SIZE_MAXIMUM {
				return errors.New("flatjson: structure field count exceeds limit")
			}
			stack = append(stack, Frame{
				Structure: structure, Prefix: prefix, Null: null,
			})
			continue
		}
		key_text := string(current.Prefix) + string(name)
		if len(key_text) > strings.TEXT_SIZE_MAXIMUM {
			return errors.New("flatjson: flattened key exceeds limit")
		}
		key := Key(key_text)
		if _, duplicate := seen[key]; duplicate {
			message := "flatjson: key " + string(key) +
				" set by two fields; rename a json tag"
			return errors.New(message)
		}
		if len(seen) == strings.TEXT_SIZE_MAXIMUM {
			return errors.New("flatjson: flattened field count exceeds limit")
		}
		seen[key] = struct{}{}
		if err = flatten_leaf(builder, key, value, current.Null); err != nil {
			return err
		}
	}
	return nil
}

func nested_frame(
	field reflect.StructField, value reflect.Value, name Key,
	prefix Prefix, parent_null strings.Boolean,
) (
	structure reflect.Value, child_prefix Prefix, child_null strings.Boolean,
	descend strings.Boolean, err error,
) {
	defer func() {
		Prefix_Invariants(child_prefix, "nested_frame.child_prefix")
		strings.Boolean_Invariants(child_null, "nested_frame.child_null")
		strings.Boolean_Invariants(descend, "nested_frame.descend")
	}()
	Key_Invariants(name, "nested_frame.name")
	Prefix_Invariants(prefix, "nested_frame.prefix")
	strings.Boolean_Invariants(parent_null, "nested_frame.parent_null")
	child_prefix = prefix
	child_null = parent_null
	if !is_embedded_struct(field) {
		if is_leaf(field.Type) {
			return reflect.Value{}, child_prefix, child_null, false, nil
		}
		nested_text := string(prefix) + string(name) + KEY_SEPARATOR
		if len(nested_text) > strings.TEXT_SIZE_MAXIMUM {
			return reflect.Value{}, child_prefix, child_null, false,
				errors.New("flatjson: flattened key exceeds limit")
		}
		child_prefix = Prefix(nested_text)
	}
	child, is_null, deref_err := deref_struct(value)
	if deref_err != nil {
		return reflect.Value{}, child_prefix, child_null, false, deref_err
	}
	if child.Kind() != reflect.Struct {
		return reflect.Value{}, child_prefix, child_null, false,
			errors.New("flatjson: Marshal only descends into struct values")
	}
	child_null = strings.Boolean(bool(is_null) || bool(parent_null))
	return child, child_prefix, child_null, true, nil
}

func deref_struct(
	value reflect.Value,
) (structure reflect.Value, null strings.Boolean, err error) {
	defer func() { strings.Boolean_Invariants(null, "deref_struct.null") }()
	for depth := 0; value.Kind() == reflect.Pointer; depth++ {
		if depth == FRAME_COUNT_MAXIMUM {
			return reflect.Value{}, false,
				errors.New("flatjson: pointer nesting exceeds limit")
		}
		if value.IsNil() {
			element := value.Type().Elem()
			for pointer_depth := 0; element.Kind() == reflect.Pointer; pointer_depth++ {
				if pointer_depth == FRAME_COUNT_MAXIMUM {
					return reflect.Value{}, false,
						errors.New(
							"flatjson: pointer nesting exceeds limit",
						)
				}
				element = element.Elem()
			}
			return reflect.New(element).Elem(), true, nil
		}
		value = value.Elem()
	}
	return value, false, nil
}

func flatten_leaf(
	builder Encoder, key Key, value reflect.Value, null strings.Boolean,
) (err error) {
	Encoder_Invariants(builder, "flatten_leaf.builder")
	Key_Invariants(key, "flatten_leaf.key")
	strings.Boolean_Invariants(null, "flatten_leaf.null")
	if !null {
		if !is_flat_leaf(value.Type()) {
			return errors.New("flatjson: " + string(key) + " is not flat")
		}
	}
	if builder.Storage[int(builder.Size)-1] != '{' {
		if err = write_text(builder, ","); err != nil {
			return err
		}
	}
	if err = append_json_string(builder, Json_Text(key)); err != nil {
		return err
	}
	if err = write_text(builder, ":"); err != nil {
		return err
	}
	if null {
		return write_text(builder, "null")
	}
	if !value.CanInterface() {
		return errors.New("flatjson: marshalled leaf is not exported")
	}
	return marshal_flat_value(builder, value)
}

func append_json_string(builder Encoder, text Json_Text) (err error) {
	Encoder_Invariants(builder, "append_json_string.builder")
	Json_Text_Invariants(text, "append_json_string.text")
	if err = write_text(builder, "\""); err != nil {
		return err
	}
	const HEXADECIMAL = "0123456789abcdef"
	for _, character := range text {
		switch character {
		case '"', '\\':
			escaped := Nonempty_Text([]rune{'\\', character})
			if err = write_text(builder, escaped); err != nil {
				return err
			}
		case '\b':
			err = write_text(builder, "\\b")
		case '\f':
			err = write_text(builder, "\\f")
		case '\n':
			err = write_text(builder, "\\n")
		case '\r':
			err = write_text(builder, "\\r")
		case '\t':
			err = write_text(builder, "\\t")
		case '<', '>', '&', '\u2028', '\u2029':
			var escaped [JSON_ESCAPE_SIZE]byte
			escaped[0] = '\\'
			escaped[1] = 'u'
			escaped[2] = HEXADECIMAL[character>>12]
			escaped[3] = HEXADECIMAL[character>>8&0x0f]
			escaped[4] = HEXADECIMAL[character>>4&0x0f]
			escaped[5] = HEXADECIMAL[character&0x0f]
			err = write_text(builder, Nonempty_Text(escaped[:]))
		default:
			if character < ' ' {
				var escaped [JSON_ESCAPE_SIZE]byte
				escaped[0] = '\\'
				escaped[1] = 'u'
				escaped[2] = '0'
				escaped[3] = '0'
				escaped[4] = HEXADECIMAL[character>>4&0x0f]
				escaped[5] = HEXADECIMAL[character&0x0f]
				err = write_text(builder, Nonempty_Text(escaped[:]))
			} else {
				err = write_text(builder, Nonempty_Text(string(character)))
			}
		}
		if err != nil {
			return err
		}
	}
	return write_text(builder, "\"")
}

func write_text(builder Encoder, text Nonempty_Text) (err error) {
	Encoder_Invariants(builder, "write_text.builder")
	Nonempty_Text_Invariants(text, "write_text.text")
	start := int(builder.Size)
	if len(text) > len(builder.Storage)-start {
		return errors.New("flatjson: encoded output exceeds limit")
	}
	copy(builder.Storage[start:], text)
	builder.Size += strings.Size_Value(len(text))
	return nil
}

func field_json(
	field reflect.StructField,
) (name Json_Text, skip strings.Boolean, err error) {
	defer func() {
		Json_Text_Invariants(name, "field_json.name")
		strings.Boolean_Invariants(skip, "field_json.skip")
	}()
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", true, nil
	}
	leaf := tag
	for index := 0; index < len(tag); index++ {
		if tag[index] == ',' {
			leaf = tag[:index]
			break
		}
	}
	selected := field.Name
	if leaf != "" {
		selected = leaf
	}
	if len(selected) > strings.TEXT_SIZE_MAXIMUM {
		return "", false, errors.New("flatjson: field name exceeds limit")
	}
	return Json_Text(selected), false, nil
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
	if marshaler_kind(base) != MARSHALER_NONE {
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
	if marshaler_kind(base) != MARSHALER_NONE {
		return true
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
	if marshaler_kind(base) != MARSHALER_NONE {
		return true
	}
	return is_scalar_kind(base.Kind())
}

func is_scalar_kind(kind reflect.Kind) (scalar strings.Boolean) {
	defer func() { strings.Boolean_Invariants(scalar, "is_scalar_kind.scalar") }()
	switch kind {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String:
		return true
	}
	return false
}

func marshaler_kind(base reflect.Type) (kind Marshaler_Kind) {
	defer func() { Marshaler_Kind_Invariants(kind, "marshaler_kind.kind") }()
	bytes_type := reflect.TypeOf([]byte(nil))
	error_type := reflect.TypeFor[error]()
	if method, found := base.MethodByName(MARSHAL_JSON_NAME); found {
		if method.Type.NumIn() == MARSHALER_INPUT_COUNT {
			if method.Type.NumOut() == MARSHALER_OUTPUT_COUNT {
				if method.Type.Out(MARSHALER_BYTES_OUTPUT) == bytes_type {
					if method.Type.Out(MARSHALER_ERROR_OUTPUT) == error_type {
						return MARSHALER_JSON
					}
				}
			}
		}
	}
	pointer := reflect.PointerTo(base)
	if method, found := pointer.MethodByName(MARSHAL_JSON_NAME); found {
		if method.Type.NumIn() == MARSHALER_INPUT_COUNT {
			if method.Type.NumOut() == MARSHALER_OUTPUT_COUNT {
				if method.Type.Out(MARSHALER_BYTES_OUTPUT) == bytes_type {
					if method.Type.Out(MARSHALER_ERROR_OUTPUT) == error_type {
						return MARSHALER_JSON
					}
				}
			}
		}
	}
	if method, found := base.MethodByName(MARSHAL_TEXT_NAME); found {
		if method.Type.NumIn() == MARSHALER_INPUT_COUNT {
			if method.Type.NumOut() == MARSHALER_OUTPUT_COUNT {
				if method.Type.Out(MARSHALER_BYTES_OUTPUT) == bytes_type {
					if method.Type.Out(MARSHALER_ERROR_OUTPUT) == error_type {
						return MARSHALER_TEXT
					}
				}
			}
		}
	}
	if method, found := pointer.MethodByName(MARSHAL_TEXT_NAME); found {
		if method.Type.NumIn() == MARSHALER_INPUT_COUNT {
			if method.Type.NumOut() == MARSHALER_OUTPUT_COUNT {
				if method.Type.Out(MARSHALER_BYTES_OUTPUT) == bytes_type {
					if method.Type.Out(MARSHALER_ERROR_OUTPUT) == error_type {
						return MARSHALER_TEXT
					}
				}
			}
		}
	}
	return MARSHALER_NONE
}
