package flatjson_test

import (
	"reflect"
	"regexp"
	"testing"

	"local/james-orcales/shared/encoding/flatjson"
	"local/james-orcales/shared/strconv"
	"local/james-orcales/shared/testify"
)

// Test_Allocation proves both public caller-storage paths own no hidden storage.
func Test_Allocation(t *testing.T) {
	test_object_allocation(t)
	test_array_allocation(t)
	test_output_allocation(t)
	test_reflection_bound_allocation(t)
	test_failure_allocation(t)
	test_write_allocation(t)
}

// Test_Nested_Struct_Flattens_To_Prefixed_Keys checks nested fields join into one key.
func Test_Nested_Struct_Flattens_To_Prefixed_Keys(t *testing.T) {
	data, err := marshal(&outer{Name: "bob", Address: inner{City: "nyc", Zip: 10001}})
	testify.No_Error(t, err)
	want := `{"name":"bob","addr_city":"nyc","addr_zip":10001}`
	testify.Equal(t, want, string(data))
}

// Test_Scalar_Fields_Marshal_To_Their_JSON_Forms checks scalars emit their JSON scalar forms.
func Test_Scalar_Fields_Marshal_To_Their_JSON_Forms(t *testing.T) {
	data, err := marshal(&scalars{
		Text: "<hi>\n\u2028", Count: -7, Ok: true,
	})
	testify.No_Error(t, err)
	want := `{"text":"\u003chi\u003e\n\u2028","count":-7,"ok":true}`
	testify.Equal(t, want, string(data))
	empty, empty_err := marshal(&struct {
		Text string `json:"text"`
	}{})
	testify.No_Error(t, empty_err)
	testify.Equal(t, `{"text":""}`, string(empty))
	maximum, maximum_err := marshal(&struct {
		Value string `json:"x"`
	}{Value: repeated_byte('a', 4088)})
	testify.No_Error(t, maximum_err)
	testify.Count(t, maximum, 4096)
	_, maximum_err = marshal(&struct {
		Value string `json:"x"`
	}{Value: repeated_byte('a', 4096)})
	testify.Error(t, maximum_err)
}

// Test_Scalar_Slices_Pass_Through checks scalar slices stay JSON arrays under one key.
func Test_Scalar_Slices_Pass_Through(t *testing.T) {
	data, err := marshal(&withtags{
		Tags: []string{"a", "b"}, Nums: []int{1, 2}, Empty: nil,
	})
	testify.No_Error(t, err)
	want := `{"tags":["a","b"],"nums":[1,2],"empty":null}`
	testify.Equal(t, want, string(data))
}

// Test_Json_Tag_Names_The_Leaf checks the json tag overrides the field name as the key.
func Test_Json_Tag_Names_The_Leaf(t *testing.T) {
	data, err := marshal(&tagged{Renamed: "x", Plain: "y"})
	testify.No_Error(t, err)
	want := `{"renamed":"x","Plain":"y"}`
	testify.Equal(t, want, string(data))
	maximum_tag := reflect.StructTag(`json:"` + repeated_byte('a', 4096) + `"`)
	maximum_type := reflect.StructOf([]reflect.StructField{{
		Name: "Value", Type: reflect.TypeOf(""), Tag: maximum_tag,
	}})
	_, maximum_err := marshal(reflect.New(maximum_type).Interface())
	testify.Error_Is(t, maximum_err, flatjson.ERR_FIELD_TAG_SIZE)
	maximum_name_type := structure_field_type(
		flatjson.OUTPUT_SIZE_MAXIMUM, reflect.TypeOf(0),
	)
	_, maximum_name_err := marshal(reflect.New(maximum_name_type).Interface())
	testify.Error_Is(t, maximum_name_err, flatjson.ERR_ENCODED_OUTPUT_SIZE)
}

// Test_Embedded_Struct_Adds_No_Segment checks embedded fields promote without a prefix.
func Test_Embedded_Struct_Adds_No_Segment(t *testing.T) {
	data, err := marshal(&place{Coordinates: Coordinates{X: 1, Y: 2}, Name: "park"})
	testify.No_Error(t, err)
	want := `{"x":1,"y":2,"name":"park"}`
	testify.Equal(t, want, string(data))
	short, short_err := marshal(&short_outer{})
	testify.No_Error(t, short_err)
	testify.Equal(t, `{"a_x":0}`, string(short))
	child_type := structure_field_type(1, reflect.TypeOf(0))
	maximum_prefix_type := structure_field_type(
		flatjson.OUTPUT_SIZE_MAXIMUM-1, child_type,
	)
	_, maximum_err := marshal(
		reflect.New(maximum_prefix_type).Interface(),
	)
	testify.Error_Is(t, maximum_err, flatjson.ERR_FLATTENED_KEY_SIZE)
	deep_type := deep_struct_type(flatjson.FRAME_COUNT_MAXIMUM)
	_, deep_err := marshal(reflect.New(deep_type).Interface())
	testify.No_Error(t, deep_err)
	too_deep_type := deep_struct_type(flatjson.FRAME_COUNT_MAXIMUM + 1)
	_, too_deep_err := marshal(reflect.New(too_deep_type).Interface())
	testify.Error_Is(t, too_deep_err, flatjson.ERR_STRUCTURE_DEPTH)
	wide_type := skipped_fields_type(flatjson.STRUCTURE_FIELD_COUNT_MAXIMUM)
	wide, wide_err := marshal(reflect.New(wide_type).Interface())
	testify.No_Error(t, wide_err)
	testify.Equal(t, `{}`, string(wide))
	too_wide_type := skipped_fields_type(flatjson.STRUCTURE_FIELD_COUNT_MAXIMUM + 1)
	_, too_wide_err := marshal(reflect.New(too_wide_type).Interface())
	testify.Error(t, too_wide_err)
}

// Test_Nil_Pointer_Emits_Null checks a nil pointer is never dropped: scalar null, nested null
// leaves, so the key set stays stable.
func Test_Nil_Pointer_Emits_Null(t *testing.T) {
	data, err := marshal(&optional{Name: "bob"})
	testify.No_Error(t, err)
	want := `{"name":"bob","addr_city":null,"addr_zip":null,"note":null}`
	testify.Equal(t, want, string(data))
}

// Test_Marshaler_Leaf_Is_Rejected checks allocation-owning method contracts do not bypass the
// primitive-only leaf boundary.
func Test_Marshaler_Leaf_Is_Rejected(t *testing.T) {
	_, err := marshal(&event{Label: *regexp.MustCompile("clock")})
	testify.Error(t, err)
}

// Test_Top_Level_Array_Marshals checks a slice marshals to a top-level array of flat objects.
func Test_Top_Level_Array_Marshals(t *testing.T) {
	source := []outer{
		{Name: "a", Address: inner{City: "x", Zip: 1}},
		{Name: "b", Address: inner{City: "y", Zip: 2}},
	}
	data, err := marshal(&source)
	testify.No_Error(t, err)
	want := `[{"name":"a","addr_city":"x","addr_zip":1},` +
		`{"name":"b","addr_city":"y","addr_zip":2}]`
	testify.Equal(t, want, string(data))
	empty_source := []outer{}
	empty, empty_err := marshal(&empty_source)
	testify.No_Error(t, empty_err)
	testify.Equal(t, `[]`, string(empty))
}

// Test_Non_Flat_Field_Is_Rejected checks a map field is a marshal error, not nested output.
func Test_Non_Flat_Field_Is_Rejected(t *testing.T) {
	_, marshal_err := marshal(&withmap{Labels: map[string]string{"k": "v"}})
	testify.Error(t, marshal_err)
	_, marshal_err = marshal(&struct {
		Items []inner `json:"items"`
	}{})
	testify.Error(t, marshal_err)
}

// Test_Colliding_Keys_Are_Rejected checks two fields sharing a flat key are a marshal error.
func Test_Colliding_Keys_Are_Rejected(t *testing.T) {
	value := collide{Address: inner{City: "x", Zip: 1}, Address_City: "y"}
	_, marshal_err := marshal(&value)
	testify.Error(t, marshal_err)
}

// Test_Stream_Write_Encodes_To_Writer checks Marshal_Write writes the flat JSON to a writer.
func Test_Stream_Write_Encodes_To_Writer(t *testing.T) {
	source := outer{Name: "bob", Address: inner{City: "nyc", Zip: 1}}
	buffer := &memory_writer{}
	write := func(data []byte) (written int, err error) {
		return memory_writer_write(buffer, data)
	}
	var storage [flatjson.OUTPUT_SIZE_MAXIMUM]byte
	value := flatjson.Value(reflect.ValueOf(&source))
	err := flatjson.Marshal_Write(write, storage[:], value)
	testify.No_Error(t, err)
	want := `{"name":"bob","addr_city":"nyc","addr_zip":1}`
	testify.Equal(t, want, string(buffer.Data))
}

const EMPTY_DOCUMENT_SIZE = len("{}")

func test_object_allocation(t *testing.T) {
	source := struct {
		Text     string   `json:"text"`
		Signed   int64    `json:"signed"`
		Unsigned uint64   `json:"unsigned"`
		Valid    bool     `json:"valid"`
		Names    []string `json:"names"`
		Counts   []int64  `json:"counts"`
	}{
		Text: "<>&\b\f\n\r\t\"\\世🙂", Signed: -1, Unsigned: ^uint64(0), Valid: true,
		Names: []string{"a", "世"}, Counts: []int64{-1, 0, 1},
	}
	var output [flatjson.OUTPUT_SIZE_MAXIMUM]byte
	var data flatjson.Data
	var marshal_error flatjson.Error
	value := flatjson.Value(reflect.ValueOf(&source))
	testify.Zero_Allocation(t, func() {
		data, marshal_error = flatjson.Marshal_Into(output[:], value)
	})
	testify.Not_Nil(t, data)
	testify.Nil(t, marshal_error)

	nil_source := optional{Name: "bob"}
	nil_value := flatjson.Value(reflect.ValueOf(&nil_source))
	testify.Zero_Allocation(t, func() {
		data, marshal_error = flatjson.Marshal_Into(output[:], nil_value)
	})
	testify.Not_Nil(t, data)
	testify.Nil(t, marshal_error)
}

func test_array_allocation(t *testing.T) {
	source := []outer{
		{Name: "a", Address: inner{City: "x", Zip: 1}},
		{Name: "b", Address: inner{City: "y", Zip: 2}},
	}
	value := flatjson.Value(reflect.ValueOf(&source))
	var output [flatjson.OUTPUT_SIZE_MAXIMUM]byte
	var data flatjson.Data
	var marshal_error flatjson.Error
	testify.Zero_Allocation(t, func() {
		data, marshal_error = flatjson.Marshal_Into(output[:], value)
	})
	testify.Not_Nil(t, data)
	testify.Nil(t, marshal_error)
}

func test_output_allocation(t *testing.T) {
	source := struct{}{}
	value := flatjson.Value(reflect.ValueOf(&source))
	var output [EMPTY_DOCUMENT_SIZE]byte
	var data flatjson.Data
	var marshal_error flatjson.Error
	testify.Zero_Allocation(t, func() {
		data, marshal_error = flatjson.Marshal_Into(output[:], value)
	})
	testify.Equal(t, `{}`, string(data))
	testify.Nil(t, marshal_error)
}

func test_reflection_bound_allocation(t *testing.T) {
	wide_type := skipped_fields_type(flatjson.STRUCTURE_FIELD_COUNT_MAXIMUM)
	wide_source := reflect.New(wide_type)
	wide_value := flatjson.Value(wide_source)
	var output [flatjson.OUTPUT_SIZE_MAXIMUM]byte
	var data flatjson.Data
	var marshal_error flatjson.Error
	testify.Zero_Allocation(t, func() {
		data, marshal_error = flatjson.Marshal_Into(output[:], wide_value)
	})
	testify.Not_Nil(t, data)
	testify.Nil(t, marshal_error)

	escaped_type := reflect.StructOf([]reflect.StructField{{
		Name: "Value", Type: reflect.TypeOf(""), Tag: `json:"line\\tbreak"`,
	}})
	escaped_source := reflect.New(escaped_type)
	escaped_value := flatjson.Value(escaped_source)
	testify.Zero_Allocation(t, func() {
		data, marshal_error = flatjson.Marshal_Into(output[:], escaped_value)
	})
	testify.Not_Nil(t, data)
	testify.Nil(t, marshal_error)

	wide_child_type := skipped_fields_type(flatjson.STRUCTURE_FIELD_COUNT_MAXIMUM + 1)
	nested_wide_type := structure_field_type(1, wide_child_type)
	nested_wide_value := flatjson.Value(reflect.New(nested_wide_type))
	testify.Zero_Allocation(t, func() {
		data, marshal_error = flatjson.Marshal_Into(output[:], nested_wide_value)
	})
	testify.Nil(t, data)
	testify.Error_Is(t, marshal_error, flatjson.ERR_STRUCTURE_FIELD_COUNT)
}

func test_failure_allocation(t *testing.T) {
	var output [flatjson.OUTPUT_SIZE_MAXIMUM]byte
	var invalid flatjson.Value
	assert_marshal_allocation(t, output[:], invalid, flatjson.ERR_ROOT_TYPE)

	source := outer{Name: "bob"}
	value := flatjson.Value(reflect.ValueOf(&source))
	assert_marshal_allocation(t, output[:0], value, flatjson.ERR_OUTPUT_TOO_SMALL)
	assert_marshal_allocation(t, output[:1], value, flatjson.ERR_OUTPUT_TOO_SMALL)

	var nil_source *outer
	nil_value := flatjson.Value(reflect.ValueOf(nil_source))
	assert_marshal_allocation(t, output[:], nil_value, flatjson.ERR_NIL_ROOT)

	map_source := withmap{Labels: map[string]string{"k": "v"}}
	map_value := flatjson.Value(reflect.ValueOf(&map_source))
	assert_marshal_allocation(t, output[:], map_value, flatjson.ERR_VALUE_NOT_FLAT)

	collision_source := collide{Address_City: "collision"}
	collision_value := flatjson.Value(reflect.ValueOf(&collision_source))
	assert_marshal_allocation(t, output[:], collision_value, flatjson.ERR_DUPLICATE_KEY)

	wide_type := skipped_fields_type(flatjson.STRUCTURE_FIELD_COUNT_MAXIMUM + 1)
	wide_value := flatjson.Value(reflect.New(wide_type))
	assert_marshal_allocation(t, output[:], wide_value, flatjson.ERR_STRUCTURE_FIELD_COUNT)

	method_source := event{Label: *regexp.MustCompile("clock")}
	method_value := flatjson.Value(reflect.ValueOf(&method_source))
	assert_marshal_allocation(t, output[:], method_value, flatjson.ERR_VALUE_NOT_FLAT)

	long_source := struct {
		Value string `json:"value"`
	}{Value: repeated_byte('a', flatjson.OUTPUT_SIZE_MAXIMUM)}
	long_value := flatjson.Value(reflect.ValueOf(&long_source))
	assert_marshal_allocation(t, output[:], long_value, flatjson.ERR_ENCODED_OUTPUT_SIZE)

	maximum_name_type := structure_field_type(
		flatjson.OUTPUT_SIZE_MAXIMUM, reflect.TypeOf(0),
	)
	maximum_name_value := flatjson.Value(reflect.New(maximum_name_type))
	assert_marshal_allocation(
		t, output[:], maximum_name_value, flatjson.ERR_ENCODED_OUTPUT_SIZE,
	)

	child_type := structure_field_type(1, reflect.TypeOf(0))
	maximum_prefix_type := structure_field_type(
		flatjson.OUTPUT_SIZE_MAXIMUM-1, child_type,
	)
	maximum_prefix_value := flatjson.Value(reflect.New(maximum_prefix_type))
	assert_marshal_allocation(
		t, output[:], maximum_prefix_value, flatjson.ERR_FLATTENED_KEY_SIZE,
	)

	too_deep_type := deep_struct_type(flatjson.FRAME_COUNT_MAXIMUM + 1)
	too_deep_value := flatjson.Value(reflect.New(too_deep_type))
	assert_marshal_allocation(t, output[:], too_deep_value, flatjson.ERR_STRUCTURE_DEPTH)
}

func test_write_allocation(t *testing.T) {
	source := outer{Name: "bob"}
	value := flatjson.Value(reflect.ValueOf(&source))
	var output [flatjson.OUTPUT_SIZE_MAXIMUM]byte
	var writer_storage [flatjson.OUTPUT_SIZE_MAXIMUM]byte
	write := func(data []byte) (written int, err error) {
		return copy(writer_storage[:], data), nil
	}
	var marshal_error flatjson.Error
	testify.Zero_Allocation(t, func() {
		marshal_error = flatjson.Marshal_Write(write, output[:], value)
	})
	testify.Nil(t, marshal_error)

	short := func(_ []byte) (written int, err error) { return 0, nil }
	testify.Zero_Allocation(t, func() {
		marshal_error = flatjson.Marshal_Write(short, output[:], value)
	})
	testify.Error_Is(t, marshal_error, flatjson.ERR_SHORT_WRITE)

	fail := func(_ []byte) (written int, err error) {
		return 0, flatjson.ERR_OUTPUT_TOO_SMALL
	}
	testify.Zero_Allocation(t, func() {
		marshal_error = flatjson.Marshal_Write(fail, output[:], value)
	})
	testify.Error_Is(t, marshal_error, flatjson.ERR_OUTPUT_TOO_SMALL)

	testify.Zero_Allocation(t, func() {
		marshal_error = flatjson.Marshal_Write(nil, output[:], value)
	})
	testify.Error_Is(t, marshal_error, flatjson.ERR_WRITE_NIL)

	var invalid flatjson.Value
	testify.Zero_Allocation(t, func() {
		marshal_error = flatjson.Marshal_Write(write, output[:], invalid)
	})
	testify.Error_Is(t, marshal_error, flatjson.ERR_ROOT_TYPE)

	empty_source := struct{}{}
	empty_value := flatjson.Value(reflect.ValueOf(&empty_source))
	var minimum_output [EMPTY_DOCUMENT_SIZE]byte
	testify.Zero_Allocation(t, func() {
		marshal_error = flatjson.Marshal_Write(write, minimum_output[:0], empty_value)
	})
	testify.Error_Is(t, marshal_error, flatjson.ERR_OUTPUT_TOO_SMALL)
	testify.Zero_Allocation(t, func() {
		marshal_error = flatjson.Marshal_Write(write, minimum_output[:1], empty_value)
	})
	testify.Error_Is(t, marshal_error, flatjson.ERR_OUTPUT_TOO_SMALL)
	testify.Zero_Allocation(t, func() {
		marshal_error = flatjson.Marshal_Write(write, minimum_output[:2], empty_value)
	})
	testify.Nil(t, marshal_error)
}

func assert_marshal_allocation(
	t *testing.T, destination flatjson.Output, value flatjson.Value, expected error,
) {
	var data flatjson.Data
	var marshal_error flatjson.Error
	testify.Zero_Allocation(t, func() {
		data, marshal_error = flatjson.Marshal_Into(destination, value)
	})
	testify.Nil(t, data)
	testify.Error_Is(t, marshal_error, expected)
}

type inner struct {
	City string `json:"city"`
	Zip  int    `json:"zip"`
}

type outer struct {
	Name    string `json:"name"`
	Address inner  `json:"addr"`
}

type scalars struct {
	Text  string `json:"text"`
	Count int    `json:"count"`
	Ok    bool   `json:"ok"`
}

type withtags struct {
	Tags  []string `json:"tags"`
	Nums  []int    `json:"nums"`
	Empty []string `json:"empty"`
}

type withmap struct {
	Labels map[string]string `json:"labels"`
}

type tagged struct {
	Renamed string `json:"renamed"`
	Plain   string
	Skipped string `json:"-"`
}

// Coordinates is exported so its embedded field name is exported and gets promoted.
type Coordinates struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type place struct {
	Coordinates
	Name string `json:"name"`
}

type optional struct {
	Name    string  `json:"name"`
	Address *inner  `json:"addr"`
	Note    *string `json:"note"`
}

type event struct {
	Label regexp.Regexp `json:"label"`
}

type short_inner struct {
	X int `json:"x"`
}

type short_outer struct {
	Child short_inner `json:"a"`
}

type memory_writer struct {
	Data []byte
}

func memory_writer_write(writer *memory_writer, data []byte) (written int, err error) {
	writer.Data = append(writer.Data, data...)
	return len(data), nil
}

func marshal(pointer any) (data flatjson.Data, err flatjson.Error) {
	var storage [flatjson.OUTPUT_SIZE_MAXIMUM]byte
	value := flatjson.Value(reflect.ValueOf(pointer))
	return flatjson.Marshal_Into(storage[:], value)
}

func repeated_byte(value byte, size int) (text string) {
	storage := make([]byte, size)
	for index := range storage {
		storage[index] = value
	}
	return string(storage)
}

func structure_field_type(name_size int, field_type reflect.Type) (value_type reflect.Type) {
	return reflect.StructOf([]reflect.StructField{{
		Name: repeated_byte('A', name_size), Type: field_type,
	}})
}

func deep_struct_type(depth_count int) (value_type reflect.Type) {
	value_type = reflect.StructOf([]reflect.StructField{{
		Name: "X", Type: reflect.TypeOf(0), Tag: `json:"x"`,
	}})
	for depth_index := 1; depth_index < depth_count; depth_index++ {
		value_type = reflect.StructOf([]reflect.StructField{{
			Name: "A", Type: value_type, Tag: `json:"a"`,
		}})
	}
	return value_type
}

func skipped_fields_type(field_count int) (value_type reflect.Type) {
	fields := make([]reflect.StructField, field_count)
	for index := range fields {
		var storage [strconv.INTEGER_TEXT_SIZE_MAXIMUM]byte
		count := strconv.Format_Decimal_Into(storage[:], strconv.Machine_Integer(index))
		fields[index] = reflect.StructField{
			Name: "F" + string(storage[:count]), Type: reflect.TypeOf(0), Tag: `json:"-"`,
		}
	}
	return reflect.StructOf(fields)
}

type collide struct {
	Address      inner  `json:"addr"`
	Address_City string `json:"addr_city"`
}
