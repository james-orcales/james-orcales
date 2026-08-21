package flatjson_test

import (
	"reflect"
	"testing"

	"local/james-orcales/shared/encoding/flatjson"
	"local/james-orcales/shared/strconv"
)

// Test_Nested_Struct_Flattens_To_Prefixed_Keys checks nested fields join into one key.
func Test_Nested_Struct_Flattens_To_Prefixed_Keys(t *testing.T) {
	data, err := flatjson.Marshal(outer{Name: "bob", Address: inner{City: "nyc", Zip: 10001}})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"name":"bob","addr_city":"nyc","addr_zip":10001}`
	if string(data) != want {
		t.Fatalf("got  %s\nwant %s", data, want)
	}
}

// Test_Scalar_Fields_Marshal_To_Their_JSON_Forms checks scalars emit their JSON scalar forms.
func Test_Scalar_Fields_Marshal_To_Their_JSON_Forms(t *testing.T) {
	data, err := flatjson.Marshal(scalars{
		Text: "<hi>\n\u2028", Count: -7, Ok: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"text":"\u003chi\u003e\n\u2028","count":-7,"ok":true}`
	if string(data) != want {
		t.Fatalf("got  %s\nwant %s", data, want)
	}
	empty, empty_err := flatjson.Marshal(struct {
		Text string `json:"text"`
	}{})
	if empty_err != nil {
		t.Fatal(empty_err)
	}
	if string(empty) != `{"text":""}` {
		t.Fatalf("empty scalar = %s", empty)
	}
	maximum, maximum_err := flatjson.Marshal(struct {
		Value string `json:"x"`
	}{Value: repeated_byte('a', 4088)})
	if maximum_err != nil {
		t.Fatal(maximum_err)
	}
	if len(maximum) != 4096 {
		t.Fatalf("maximum output size = %d", len(maximum))
	}
	if _, maximum_err = flatjson.Marshal(struct {
		Value string `json:"x"`
	}{Value: repeated_byte('a', 4096)}); maximum_err == nil {
		t.Fatal("maximum input string fit beside JSON framing")
	}
}

// Test_Scalar_Slices_Pass_Through checks scalar slices stay JSON arrays under one key.
func Test_Scalar_Slices_Pass_Through(t *testing.T) {
	data, err := flatjson.Marshal(withtags{
		Tags: []string{"a", "b"}, Nums: []int{1, 2}, Empty: nil,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"tags":["a","b"],"nums":[1,2],"empty":null}`
	if string(data) != want {
		t.Fatalf("got  %s\nwant %s", data, want)
	}
}

// Test_Json_Tag_Names_The_Leaf checks the json tag overrides the field name as the key.
func Test_Json_Tag_Names_The_Leaf(t *testing.T) {
	data, err := flatjson.Marshal(tagged{Renamed: "x", Plain: "y"})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"renamed":"x","Plain":"y"}`
	if string(data) != want {
		t.Fatalf("got  %s\nwant %s", data, want)
	}
	maximum_tag := reflect.StructTag(`json:"` + repeated_byte('a', 4096) + `"`)
	maximum_type := reflect.StructOf([]reflect.StructField{{
		Name: "Value", Type: reflect.TypeOf(""), Tag: maximum_tag,
	}})
	if _, maximum_err := flatjson.Marshal(
		reflect.New(maximum_type).Elem().Interface(),
	); maximum_err == nil {
		t.Fatal("maximum key fit beside JSON framing")
	}
}

// Test_Embedded_Struct_Adds_No_Segment checks embedded fields promote without a prefix.
func Test_Embedded_Struct_Adds_No_Segment(t *testing.T) {
	data, err := flatjson.Marshal(place{Coordinates: Coordinates{X: 1, Y: 2}, Name: "park"})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"x":1,"y":2,"name":"park"}`
	if string(data) != want {
		t.Fatalf("got  %s\nwant %s", data, want)
	}
	short, short_err := flatjson.Marshal(short_outer{})
	if short_err != nil {
		t.Fatal(short_err)
	}
	if string(short) != `{"a_x":0}` {
		t.Fatalf("short prefix = %s", short)
	}
	maximum_prefix_type := nested_tag_type(4095)
	if _, maximum_err := flatjson.Marshal(
		reflect.New(maximum_prefix_type).Elem().Interface(),
	); maximum_err == nil {
		t.Fatal("maximum prefix fit another leaf name")
	}
	deep_type := deep_struct_type()
	if _, deep_err := flatjson.Marshal(
		reflect.New(deep_type).Elem().Interface(),
	); deep_err != nil {
		t.Fatal(deep_err)
	}
	wide_type := skipped_fields_type()
	wide, wide_err := flatjson.Marshal(reflect.New(wide_type).Elem().Interface())
	if wide_err != nil {
		t.Fatal(wide_err)
	}
	if string(wide) != `{}` {
		t.Fatalf("skipped fields = %s", wide)
	}
}

// Test_Nil_Pointer_Emits_Null checks a nil pointer is never dropped: scalar null, nested null
// leaves, so the key set stays stable.
func Test_Nil_Pointer_Emits_Null(t *testing.T) {
	data, err := flatjson.Marshal(optional{Name: "bob"})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"name":"bob","addr_city":null,"addr_zip":null,"note":null}`
	if string(data) != want {
		t.Fatalf("got  %s\nwant %s", data, want)
	}
}

// Test_Marshaler_Leaf_Is_Delegated checks a struct implementing json.Marshaler is encoded
// as a leaf, not recursed into (it would otherwise flatten to "when_seconds").
func Test_Marshaler_Leaf_Is_Delegated(t *testing.T) {
	data, err := flatjson.Marshal(event{When: stamp{Seconds: 42}, Label: text_stamp{}})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"when":42,"label":"clock"}`
	if string(data) != want {
		t.Fatalf("got  %s\nwant %s", data, want)
	}
	if _, wide_err := flatjson.Marshal(struct {
		Wide wide_stamp `json:"wide"`
	}{}); wide_err == nil {
		t.Fatal("maximum marshaler output fit beside its key")
	}
}

// Test_Top_Level_Array_Marshals checks a slice marshals to a top-level array of flat objects.
func Test_Top_Level_Array_Marshals(t *testing.T) {
	source := []outer{
		{Name: "a", Address: inner{City: "x", Zip: 1}},
		{Name: "b", Address: inner{City: "y", Zip: 2}},
	}
	data, err := flatjson.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"name":"a","addr_city":"x","addr_zip":1},` +
		`{"name":"b","addr_city":"y","addr_zip":2}]`
	if string(data) != want {
		t.Fatalf("got  %s\nwant %s", data, want)
	}
	empty, empty_err := flatjson.Marshal([]outer{})
	if empty_err != nil {
		t.Fatal(empty_err)
	}
	if string(empty) != `[]` {
		t.Fatalf("empty array = %s", empty)
	}
}

// Test_Non_Flat_Field_Is_Rejected checks a map field is a marshal error, not nested output.
func Test_Non_Flat_Field_Is_Rejected(t *testing.T) {
	data, marshal_err := flatjson.Marshal(withmap{Labels: map[string]string{"k": "v"}})
	if marshal_err == nil {
		t.Fatalf("expected an error marshalling a map field, got %s", data)
	}
	if _, marshal_err = flatjson.Marshal(struct {
		Items []inner `json:"items"`
	}{}); marshal_err == nil {
		t.Fatal("expected an error marshalling a struct slice")
	}
}

// Test_Colliding_Keys_Are_Rejected checks two fields sharing a flat key are a marshal error.
func Test_Colliding_Keys_Are_Rejected(t *testing.T) {
	value := collide{Address: inner{City: "x", Zip: 1}, Address_City: "y"}
	data, marshal_err := flatjson.Marshal(value)
	if marshal_err == nil {
		t.Fatalf("expected an error for colliding keys, got %s", data)
	}
}

// Test_Stream_Write_Encodes_To_Writer checks Marshal_Write writes the flat JSON to a writer.
func Test_Stream_Write_Encodes_To_Writer(t *testing.T) {
	source := outer{Name: "bob", Address: inner{City: "nyc", Zip: 1}}
	buffer := &memory_writer{}
	if err := flatjson.Marshal_Write(buffer.Write, source); err != nil {
		t.Fatal(err)
	}
	want := `{"name":"bob","addr_city":"nyc","addr_zip":1}`
	if string(buffer.Data) != want {
		t.Fatalf("got  %s\nwant %s", buffer.Data, want)
	}
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
	When  stamp      `json:"when"`
	Label text_stamp `json:"label"`
}

// A struct that marshals itself, so flatjson treats it as a leaf rather than recursing
// into Seconds. MarshalJSON matches the stdlib interface (lint-allowed).
type stamp struct {
	Seconds int
}

func (moment stamp) MarshalJSON() (data []byte, err error) {
	var storage [strconv.INTEGER_TEXT_SIZE_MAXIMUM]byte
	count := strconv.Format_Decimal_Into(
		storage[:], strconv.Machine_Integer(moment.Seconds),
	)
	return append(data, storage[:count]...), nil
}

type text_stamp struct{}

func (text_stamp) MarshalText() (text []byte, err error) {
	return []byte("clock"), nil
}

type wide_stamp struct{}

func (wide_stamp) MarshalJSON() (data []byte, err error) {
	return []byte(repeated_byte('1', 4096)), nil
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

func (writer *memory_writer) Write(data []byte) (written int, err error) {
	writer.Data = append(writer.Data, data...)
	return len(data), nil
}

func repeated_byte(value byte, size int) (text string) {
	storage := make([]byte, size)
	for index := range storage {
		storage[index] = value
	}
	return string(storage)
}

func nested_tag_type(tag_size int) (value_type reflect.Type) {
	child := reflect.StructOf([]reflect.StructField{{
		Name: "X", Type: reflect.TypeOf(0), Tag: `json:"x"`,
	}})
	tag := reflect.StructTag(`json:"` + repeated_byte('a', tag_size) + `"`)
	return reflect.StructOf([]reflect.StructField{{Name: "A", Type: child, Tag: tag}})
}

func deep_struct_type() (value_type reflect.Type) {
	value_type = reflect.StructOf([]reflect.StructField{{
		Name: "X", Type: reflect.TypeOf(0), Tag: `json:"x"`,
	}})
	for depth_index := 1; depth_index < flatjson.FRAME_COUNT_MAXIMUM; depth_index++ {
		value_type = reflect.StructOf([]reflect.StructField{{
			Name: "A", Type: value_type, Tag: `json:"a"`,
		}})
	}
	return value_type
}

func skipped_fields_type() (value_type reflect.Type) {
	fields := make([]reflect.StructField, 4096)
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
