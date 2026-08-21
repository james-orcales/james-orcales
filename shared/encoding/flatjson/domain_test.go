package flatjson

import (
	"reflect"
	"testing"
	"unsafe"

	"local/james-orcales/shared/strings"
	"local/james-orcales/shared/testify"
)

// Test_Write_Text_Maximum protects the fragment bound independently from JSON framing overhead.
func Test_Write_Text_Maximum(t *testing.T) {
	state := encoder_state(t, strings.TEXT_SIZE_MAXIMUM)
	builder := (*strings.Builder)(unsafe.Pointer(state))
	testify.Equal(t, strings.TEXT_SIZE_MAXIMUM, int(builder.Size))
}

// Test_Started_Encoder_Size_Minimum reaches scalar encoding after root framing.
func Test_Started_Encoder_Size_Minimum(t *testing.T) {
	state := encoder_state(t, ENCODER_SIZE_MINIMUM)
	err := marshal_flat_value(
		started_encoder(state), reflect.ValueOf(0),
	)
	testify.No_Error(t, err)
}

// Test_Started_Encoder_Size_Two reaches array element dispatch at its second position.
func Test_Started_Encoder_Size_Two(t *testing.T) {
	state := encoder_state(t, ENCODER_SIZE_MINIMUM+1)
	err := marshal_element(
		started_encoder(state), reflect.ValueOf(0),
	)
	testify.No_Error(t, err)
	leaf_state := encoder_state(t, ENCODER_SIZE_MINIMUM+1)
	var path_storage [OUTPUT_SIZE_MAXIMUM]byte
	_, _, err = flatten_leaf(
		started_encoder(leaf_state), Path(path_storage[:]), 0,
		"x", reflect.ValueOf(map[string]string{}), false,
	)
	testify.Error(t, err)
}

// Test_Started_Encoder_Size_Maximum reaches every scalar stage with full storage.
func Test_Started_Encoder_Size_Maximum(t *testing.T) {
	state := encoder_state(t, strings.TEXT_SIZE_MAXIMUM)
	err := marshal_element(
		started_encoder(state), reflect.ValueOf(0),
	)
	testify.Error(t, err)
	var path_storage [OUTPUT_SIZE_MAXIMUM]byte
	_, _, err = flatten_leaf(
		started_encoder(state), Path(path_storage[:]), 0,
		"x", reflect.ValueOf(true), false,
	)
	testify.Error(t, err)
	err = flatten_struct(
		started_encoder(state), reflect.ValueOf(struct{}{}),
	)
	testify.No_Error(t, err)
	err = append_json_string(started_encoder(state), "")
	testify.Error(t, err)
}

// Test_Duplicate_Key_Domains reaches each bounded collision-metadata boundary.
func Test_Duplicate_Key_Domains(t *testing.T) {
	var minimum [FLATTENED_FIELD_SIZE_MINIMUM]byte
	var maximum [COLLISION_DATA_SIZE_MAXIMUM]byte
	var starts, ends [FLATTENED_FIELD_COUNT_MAXIMUM]Key_Position
	testify.False(t, bool(duplicate_key(minimum[:], starts[:], ends[:], 0, 1, 1)))
	testify.False(t, bool(duplicate_key(
		maximum[:], starts[:], ends[:], 0,
		OUTPUT_SIZE_MAXIMUM, OUTPUT_SIZE_MAXIMUM,
	)))
	testify.False(t, bool(duplicate_key(
		maximum[:], starts[:], ends[:], Key_Count(FLATTENED_FIELD_COUNT_MAXIMUM), 1, 2,
	)))
}

// Test_Tag_Domains reaches absent, minimal, and complete reflected metadata.
func Test_Tag_Domains(t *testing.T) {
	literal, found, err := json_tag_literal("")
	testify.No_Error(t, err)
	testify.False(t, bool(found))
	testify.Empty(t, literal)
	literal, found, err = json_tag_literal(`json:""`)
	testify.No_Error(t, err)
	testify.True(t, bool(found))
	testify.Equal(t, Tag_Literal(`""`), literal)
	var tag [OUTPUT_SIZE_MAXIMUM]byte
	copy(tag[:], `json:"`)
	for position_count := len(`json:"`); position_count < len(tag)-1; position_count++ {
		tag[position_count] = 'a'
	}
	tag[len(tag)-1] = '"'
	literal, found, err = json_tag_literal(reflect.StructTag(tag[:]))
	testify.No_Error(t, err)
	testify.True(t, bool(found))
	testify.Count(t, literal, TAG_LITERAL_SIZE_MAXIMUM)
	_, valid := tag_name_end(":")
	testify.False(t, bool(valid))
	end, valid := tag_name_end("a:")
	testify.True(t, bool(valid))
	testify.Equal(t, Tag_Name_Position(1), end)
	end, valid = tag_name_end("aa:")
	testify.True(t, bool(valid))
	testify.Equal(t, Tag_Name_Position(2), end)
	var name [OUTPUT_SIZE_MAXIMUM]byte
	for position_count := range name {
		name[position_count] = 'a'
	}
	name[len(name)-1] = ':'
	end, valid = tag_name_end(Tag_Text(name[:]))
	testify.True(t, bool(valid))
	testify.Equal(t, Tag_Name_Position(OUTPUT_SIZE_MAXIMUM-1), end)
}

func encoder_state(t *testing.T, size int) (state Encoder_State) {
	t.Helper()
	storage := make([]byte, strings.TEXT_SIZE_MAXIMUM)
	builder := &strings.Builder{Storage: storage}
	state = Encoder_State(unsafe.Pointer(builder))
	text := make([]byte, size)
	for index := range text {
		text[index] = '0'
	}
	err := write_text(write_encoder(state), Nonempty_Text(text))
	testify.No_Error(t, err)
	return state
}
