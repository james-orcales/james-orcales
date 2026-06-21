// Package flatjson encodes Go values as FLAT JSON: a nested struct field is not emitted as
// a nested object but flattened into the parent, its leaves keyed by the field path joined
// with an underscore (Addr.City becomes "addr_city").
//
//	type Place struct {
//	    Name string  `json:"name"`
//	    Addr Address `json:"addr"`
//	}
//	// flatjson.Marshal(place) => {"name":"...","addr_city":"...","addr_zip":...}
//
// Flat means flat (see documentation/resources/kellybrazil.*): the output is an object, or a
// top-level array of objects, with NO further nesting. The package is encode-only — there is
// no Unmarshal — because flattening cannot be inverted: a delimiter-joined key is not
// self-describing, and a nil sub-object has no flat key to carry its absence, so a round trip
// cannot be lossless. For data you read back, use nested encoding/json; see README.md.
//
// Struct nesting flattens to prefixed keys; scalars and scalar slices pass through; a
// json.Marshaler (such as time.Time) is a leaf. Anything that would nest and cannot flatten —
// a map, or a slice of structs — is a marshal error, not silent nesting. Nothing is ever
// dropped: a nil pointer emits null (a nested nil emits null for each of its leaf keys), and
// two fields producing the same key are a marshal error. Embedded structs of an exported type
// promote without a path segment, matching encoding/json. The tree is walked with an explicit
// stack because the house style bans recursion.
package flatjson

import (
	"encoding"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"

	invariant "github.com/james-orcales/james-orcales/shared/invariant/default"
)

// The path-segment joiner: Addr.City becomes "addr_city".
const key_separator = "_"

// One node of the explicit DFS stack that stands in for recursion through the struct tree.
type frame struct {
	// Structure is the struct value whose fields this frame is walking.
	Structure reflect.Value
	// Index is the next field to visit; advancing it before descending into a child
	// preserves in-place depth-first order.
	Index int
	// Prefix is the key prefix accumulated from the ancestor path.
	Prefix string
	// Null marks a subtree under a nil pointer: its leaves emit null rather than a value,
	// so the key set stays stable whether or not the pointer is nil.
	Null bool
}

// Marshal encodes value as flat JSON: a flat object for a struct, or — the resource's other
// allowed top-level shape — a flat array for a slice or array of them. value may be a pointer.
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

// Marshal_Write encodes value as flat JSON and writes it to writer.
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

// Encodes one struct as a flat JSON object.
func marshal_object(root reflect.Value) (data []byte, err error) {
	data = []byte{'{'}
	data, err = flatten_struct(data, root, map[string]struct{}{})
	if err != nil {
		return nil, err
	}
	data = append(data, '}')
	invariant.Always(data[0] == '{', "A marshalled object opens with a brace.")
	invariant.Always(data[len(data)-1] == '}', "A marshalled object closes with a brace.")
	return data, nil
}

// Encodes a slice or array as a top-level JSON array of flat elements — the array-of-objects
// shape the resource allows.
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
	invariant.Always(data[0] == '[', "A marshalled array opens with a bracket.")
	invariant.Always(data[len(data)-1] == ']', "A marshalled array closes with a bracket.")
	return data, nil
}

// Encodes one array element: a flat object for a struct, otherwise a flat scalar.
func marshal_element(destination []byte, value reflect.Value) (output []byte, err error) {
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

// Appends a single flat scalar value, rejecting anything that would nest.
func marshal_flat_value(destination []byte, value reflect.Value) (output []byte, err error) {
	if !is_flat_leaf(value.Type()) {
		return nil, errors.New("flatjson: array element is not flat")
	}
	raw, marshal_err := json.Marshal(value.Interface())
	if marshal_err != nil {
		return nil, marshal_err
	}
	return append(destination, raw...), nil
}

// Walks root depth-first with an explicit stack, appending each leaf as a "key":value pair
// and descending into nested and embedded structs. seen carries the keys already written so a
// collision is rejected rather than emitted as a duplicate.
func flatten_struct(
	destination []byte, root reflect.Value, seen map[string]struct{},
) (output []byte, err error) {
	stack := []frame{{Structure: root, Index: 0, Prefix: "", Null: false}}
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
			next := frame{Structure: child, Index: 0, Prefix: prefix, Null: child_null}
			stack = append(stack, next)
			continue
		}
		if !is_leaf(field.Type) {
			child, child_null := deref_struct(value)
			child_null = child_null || current.Null
			is_struct := child.Kind() == reflect.Struct
			invariant.Always(is_struct, "Marshal only descends into struct values.")
			nested := prefix + name + key_separator
			next := frame{Structure: child, Index: 0, Prefix: nested, Null: child_null}
			stack = append(stack, next)
			continue
		}
		destination, err = flatten_leaf(destination, prefix+name, value, current.Null, seen)
		if err != nil {
			return nil, err
		}
	}
	return destination, nil
}

// Dereferences a possibly-pointer struct value. null is true for a nil pointer, in which case
// it returns a zero value of the element type so the walk can still emit its leaf keys (as
// null), keeping the schema stable.
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

// Appends one "key":value pair for a leaf, rejecting a colliding key or a non-flat type. A
// null leaf (under a nil subtree) emits null without consulting the value.
func flatten_leaf(
	destination []byte, key string, value reflect.Value, null bool, seen map[string]struct{},
) (output []byte, err error) {
	if _, duplicate := seen[key]; duplicate {
		return nil, errors.New(
			"flatjson: key " + key + " set by two fields; rename a json tag")
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
	invariant.Always(value.CanInterface(), "A marshalled leaf is an exported value.")
	raw, marshal_err := json.Marshal(value.Interface())
	if marshal_err != nil {
		return nil, marshal_err
	}
	return append(destination, raw...), nil
}

// Separates object members: appends a comma unless the object is still empty.
func append_comma(destination []byte) (output []byte) {
	if len(destination) == 0 {
		return destination
	}
	if destination[len(destination)-1] == '{' {
		return destination
	}
	return append(destination, ',')
}

// Appends key as a JSON string. A string never fails to marshal; the guard is defensive.
func append_key(destination []byte, key string) (output []byte) {
	encoded, marshal_err := json.Marshal(key)
	if marshal_err != nil {
		return append(destination, '"', '"')
	}
	return append(destination, encoded...)
}

// Reads a field's json tag, returning its key name and whether the field is skipped. The tag's
// options (omitempty and friends) are ignored: flatjson never drops a field, for a stable schema.
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

// Reports whether field is an anonymous embedded struct, whose fields are promoted to the
// parent object without a path segment.
func is_embedded_struct(field reflect.StructField) (embedded bool) {
	if !field.Anonymous {
		return false
	}
	return !is_leaf(field.Type)
}

// Reports whether t is encoded as a value rather than flattened: anything that is not a plain
// struct, plus any struct that marshals itself (json.Marshaler / TextMarshaler).
func is_leaf(t reflect.Type) (leaf bool) {
	base := t
	for base.Kind() == reflect.Pointer {
		base = base.Elem()
	}
	if implements_marshaler(base) {
		return true
	}
	return base.Kind() != reflect.Struct
}

// Reports whether a leaf type encodes to a flat value — a scalar or a slice/array of scalars —
// keeping the output flat. A map or an array of objects is not flat.
func is_flat_leaf(t reflect.Type) (flat bool) {
	base := t
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

// Reports whether a slice or array element type is a scalar (or self-marshalling) value.
func is_scalar_element(t reflect.Type) (scalar bool) {
	base := t
	for base.Kind() == reflect.Pointer {
		base = base.Elem()
	}
	if implements_marshaler(base) {
		return true
	}
	return is_scalar_kind(base.Kind())
}

// Reports whether kind is a JSON scalar kind.
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

// Reports whether base or its pointer implements json.Marshaler or encoding.TextMarshaler.
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
