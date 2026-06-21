package flatjson_test

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/james-orcales/james-orcales/shared/flatjson"
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
	data, err := flatjson.Marshal(scalars{Text: "hi", Count: -7, Ok: true, Ratio: 1.5})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"text":"hi","count":-7,"ok":true,"ratio":1.5}`
	if string(data) != want {
		t.Fatalf("got  %s\nwant %s", data, want)
	}
}

// Test_Scalar_Slices_Pass_Through checks scalar slices stay JSON arrays under one key.
func Test_Scalar_Slices_Pass_Through(t *testing.T) {
	data, err := flatjson.Marshal(withtags{Tags: []string{"a", "b"}, Nums: []int{1, 2}})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"tags":["a","b"],"nums":[1,2]}`
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
	data, err := flatjson.Marshal(event{When: stamp{Seconds: 42}})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"when":42}`
	if string(data) != want {
		t.Fatalf("got  %s\nwant %s", data, want)
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
}

// Test_Non_Flat_Field_Is_Rejected checks a map field is a marshal error, not nested output.
func Test_Non_Flat_Field_Is_Rejected(t *testing.T) {
	data, marshal_err := flatjson.Marshal(withmap{Labels: map[string]string{"k": "v"}})
	if marshal_err == nil {
		t.Fatalf("expected an error marshalling a map field, got %s", data)
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
	buffer := &bytes.Buffer{}
	if err := flatjson.Marshal_Write(buffer, source); err != nil {
		t.Fatal(err)
	}
	want := `{"name":"bob","addr_city":"nyc","addr_zip":1}`
	if buffer.String() != want {
		t.Fatalf("got  %s\nwant %s", buffer.String(), want)
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
	Text  string  `json:"text"`
	Count int     `json:"count"`
	Ok    bool    `json:"ok"`
	Ratio float64 `json:"ratio"`
}

type withtags struct {
	Tags []string `json:"tags"`
	Nums []int    `json:"nums"`
}

type withmap struct {
	Labels map[string]string `json:"labels"`
}

type tagged struct {
	Renamed string `json:"renamed"`
	Plain   string
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
	When stamp `json:"when"`
}

// A struct that marshals itself, so flatjson treats it as a leaf rather than recursing
// into Seconds. MarshalJSON matches the stdlib interface (lint-allowed).
type stamp struct {
	Seconds int
}

func (moment stamp) MarshalJSON() (data []byte, err error) {
	return []byte(strconv.Itoa(moment.Seconds)), nil
}

type collide struct {
	Address      inner  `json:"addr"`
	Address_City string `json:"addr_city"`
}
