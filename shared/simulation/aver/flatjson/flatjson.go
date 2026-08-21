// Package aver_flatjson owns invariant reporting's flat JSON copy. It stays separate because
// shared/encoding/flatjson depends on shared/simulation/aver/default for its own checked
// implementation.
package aver_flatjson

import (
	"encoding"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
)

// KEY_SEPARATOR preserves flat field paths without nested JSON objects.
const KEY_SEPARATOR = "_"

// Frame replaces recursive reflection traversal.
type Frame struct {
	// Structure stays in each frame because iteration must resume after child traversal.
	Structure reflect.Value
	// Index preserves parent progress while child frame owns traversal.
	Index int
	// Prefix prevents nested fields from losing flattened identity.
	Prefix string
	// Null propagates nil ancestry because zero child values cannot retain that fact.
	Null bool
}

// Marshal encodes one struct or one top-level array of structs as flat JSON.
func Marshal(value any) (data []byte, err error) {
	root := reflect.ValueOf(value)
	for root.Kind() == reflect.Pointer {
		if root.IsNil() {
			return nil, errors.New("flatjson: Marshal of a nil pointer")
		}
		root = root.Elem()
	}
	if root.Kind() == reflect.Slice {
		return marshal_array(root)
	}
	if root.Kind() == reflect.Array {
		return marshal_array(root)
	}
	if root.Kind() != reflect.Struct {
		return nil, errors.New("flatjson: Marshal requires a struct or a slice of them")
	}
	return marshal_object(root)
}

// Marshal_Write keeps injected writer failure separate from encoding failure.
func Marshal_Write(writer io.Writer, value any) (err error) {
	data, marshal_err := Marshal(value)
	if marshal_err != nil {
		return marshal_err
	}
	written, write_err := writer.Write(data)
	if write_err != nil {
		return write_err
	}
	if written != len(data) {
		return io.ErrShortWrite
	}
	return nil
}

func marshal_object(root reflect.Value) (data []byte, err error) {
	data = []byte{'{'}
	data, err = flatten_struct(data, root, map[string]struct{}{})
	if err != nil {
		return nil, err
	}
	data = append(data, '}')
	Always(data[0] == '{', "A marshalled object opens with a brace.")
	Always(data[len(data)-1] == '}', "A marshalled object closes with a brace.")
	return data, nil
}

func marshal_array(root reflect.Value) (data []byte, err error) {
	data = []byte{'['}
	for index := 0; index < root.Len(); index++ {
		if index > 0 {
			data = append(data, ',')
		}
		data, err = marshal_element(data, root.Index(index))
		if err != nil {
			return nil, err
		}
	}
	data = append(data, ']')
	Always(data[0] == '[', "A marshalled array opens with a bracket.")
	Always(data[len(data)-1] == ']', "A marshalled array closes with a bracket.")
	return data, nil
}

func marshal_element(
	destination []byte, value reflect.Value,
) (output []byte, err error) {
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return append(destination, 'n', 'u', 'l', 'l'), nil
		}
		value = value.Elem()
	}
	if is_leaf(value.Type()) {
		return marshal_flat_value(destination, value)
	}
	destination = append(destination, '{')
	destination, err = flatten_struct(destination, value, map[string]struct{}{})
	if err != nil {
		return nil, err
	}
	return append(destination, '}'), nil
}

func marshal_flat_value(
	destination []byte, value reflect.Value,
) (output []byte, err error) {
	if !is_flat_leaf(value.Type()) {
		return nil, errors.New("flatjson: array element is not flat")
	}
	raw, marshal_err := json.Marshal(value.Interface())
	if marshal_err != nil {
		return nil, marshal_err
	}
	return append(destination, raw...), nil
}

func flatten_struct(
	destination []byte, root reflect.Value, seen map[string]struct{},
) (output []byte, err error) {
	stack := []Frame{{Structure: root}}
	for len(stack) > 0 {
		depth := len(stack) - 1
		current := stack[depth]
		if current.Index >= current.Structure.NumField() {
			stack = stack[:depth]
			continue
		}
		stack[depth].Index++
		field := current.Structure.Type().Field(current.Index)
		if field.PkgPath != "" {
			continue
		}
		name, skip := field_json(field)
		if skip {
			continue
		}
		value := current.Structure.Field(current.Index)
		prefix := current.Prefix
		if is_embedded_struct(field) {
			child, child_null := deref_struct(value)
			child_null = child_null || current.Null
			stack = append(stack, Frame{
				Structure: child, Prefix: prefix, Null: child_null,
			})
			continue
		}
		if !is_leaf(field.Type) {
			child, child_null := deref_struct(value)
			child_null = child_null || current.Null
			Always(
				child.Kind() == reflect.Struct,
				"Marshal only descends into struct values.",
			)
			stack = append(stack, Frame{
				Structure: child,
				Prefix:    prefix + name + KEY_SEPARATOR,
				Null:      child_null,
			})
			continue
		}
		destination, err = flatten_leaf(
			destination, prefix+name, value, current.Null, seen,
		)
		if err != nil {
			return nil, err
		}
	}
	return destination, nil
}

func deref_struct(value reflect.Value) (structure reflect.Value, null bool) {
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			element := value.Type().Elem()
			for element.Kind() == reflect.Pointer {
				element = element.Elem()
			}
			return reflect.New(element).Elem(), true
		}
		value = value.Elem()
	}
	return value, false
}

func flatten_leaf(
	destination []byte, key string, value reflect.Value, null bool,
	seen map[string]struct{},
) (output []byte, err error) {
	if _, duplicate := seen[key]; duplicate {
		return nil, errors.New(
			"flatjson: key " + key + " set by two fields; rename a json tag",
		)
	}
	if !null {
		if !is_flat_leaf(value.Type()) {
			return nil, errors.New("flatjson: " + key + " is not flat")
		}
	}
	seen[key] = struct{}{}
	destination = append_comma(destination)
	destination = append_key(destination, key)
	destination = append(destination, ':')
	if null {
		return append(destination, 'n', 'u', 'l', 'l'), nil
	}
	Always(value.CanInterface(), "A marshalled leaf is an exported value.")
	raw, marshal_err := json.Marshal(value.Interface())
	if marshal_err != nil {
		return nil, marshal_err
	}
	return append(destination, raw...), nil
}

func append_comma(destination []byte) (output []byte) {
	if len(destination) == 0 {
		return destination
	}
	if destination[len(destination)-1] == '{' {
		return destination
	}
	return append(destination, ',')
}

func append_key(destination []byte, key string) (output []byte) {
	encoded, marshal_err := json.Marshal(key)
	if marshal_err != nil {
		return append(destination, '"', '"')
	}
	return append(destination, encoded...)
}

func field_json(field reflect.StructField) (name string, skip bool) {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", true
	}
	name = field.Name
	leaf := strings.Split(tag, ",")[0]
	if leaf != "" {
		name = leaf
	}
	return name, false
}

func is_embedded_struct(field reflect.StructField) (embedded bool) {
	if !field.Anonymous {
		return false
	}
	return !is_leaf(field.Type)
}

func is_leaf(value_type reflect.Type) (leaf bool) {
	base := value_type
	for base.Kind() == reflect.Pointer {
		base = base.Elem()
	}
	if implements_marshaler(base) {
		return true
	}
	return base.Kind() != reflect.Struct
}

func is_flat_leaf(value_type reflect.Type) (flat bool) {
	base := value_type
	for base.Kind() == reflect.Pointer {
		base = base.Elem()
	}
	if implements_marshaler(base) {
		return true
	}
	if is_scalar_kind(base.Kind()) {
		return true
	}
	if base.Kind() == reflect.Slice {
		return is_scalar_element(base.Elem())
	}
	if base.Kind() == reflect.Array {
		return is_scalar_element(base.Elem())
	}
	return false
}

func is_scalar_element(value_type reflect.Type) (scalar bool) {
	base := value_type
	for base.Kind() == reflect.Pointer {
		base = base.Elem()
	}
	if implements_marshaler(base) {
		return true
	}
	return is_scalar_kind(base.Kind())
}

func is_scalar_kind(kind reflect.Kind) (scalar bool) {
	switch kind {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String:
		return true
	}
	return false
}

func implements_marshaler(base reflect.Type) (yes bool) {
	pointer := reflect.PointerTo(base)
	json_marshaler := reflect.TypeFor[json.Marshaler]()
	text_marshaler := reflect.TypeFor[encoding.TextMarshaler]()
	if base.Implements(json_marshaler) {
		return true
	}
	if pointer.Implements(json_marshaler) {
		return true
	}
	if base.Implements(text_marshaler) {
		return true
	}
	if pointer.Implements(text_marshaler) {
		return true
	}
	return false
}
