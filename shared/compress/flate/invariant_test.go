package flate

import (
	"testing"

	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// INVARIANT_BIT_STORAGE_COUNT fits widest field without unrelated capacity.
const INVARIANT_BIT_STORAGE_COUNT = BLOCK_KIND_BIT_COUNT

// INVARIANT_BYTE_STORAGE_COUNT isolates one direct byte write.
const INVARIANT_BYTE_STORAGE_COUNT = FINAL_BIT_COUNT

// INVARIANT_CODE_STORAGE_COUNT keeps all selected symbols writable.
const INVARIANT_CODE_STORAGE_COUNT = bits.BIT_COUNT_16_MAXIMUM

// INVARIANT_HASH_INPUT_COUNT is exact prefix-hash width.
const INVARIANT_HASH_INPUT_COUNT = HASH_INPUT_BYTE_COUNT

// INVARIANT_SMALL_SOURCE_COUNT leaves two candidate positions.
const INVARIANT_SMALL_SOURCE_COUNT = bits.BIT_COUNT_8_MAXIMUM

// INVARIANT_WRITER_ENCODE_STORED selects stored encoder.
const INVARIANT_WRITER_ENCODE_STORED = 0

// INVARIANT_WRITER_STORED_BLOCK selects stored block writer.
const INVARIANT_WRITER_STORED_BLOCK = INVARIANT_WRITER_ENCODE_STORED + 1

// INVARIANT_WRITER_ENCODE_LITERALS selects literal encoder.
const INVARIANT_WRITER_ENCODE_LITERALS = INVARIANT_WRITER_STORED_BLOCK + 1

// INVARIANT_WRITER_ENCODE_FIXED selects fixed encoder.
const INVARIANT_WRITER_ENCODE_FIXED = INVARIANT_WRITER_ENCODE_LITERALS + 1

// INVARIANT_WRITER_FIXED_SYMBOL selects fixed symbol writer.
const INVARIANT_WRITER_FIXED_SYMBOL = INVARIANT_WRITER_ENCODE_FIXED + 1

// INVARIANT_WRITER_FIXED_DISTANCE selects fixed distance writer.
const INVARIANT_WRITER_FIXED_DISTANCE = INVARIANT_WRITER_FIXED_SYMBOL + 1

// INVARIANT_WRITER_ALIGN selects writer alignment.
const INVARIANT_WRITER_ALIGN = INVARIANT_WRITER_FIXED_DISTANCE + 1

// INVARIANT_WRITER_FINISH selects writer finalization.
const INVARIANT_WRITER_FINISH = INVARIANT_WRITER_ALIGN + 1

// INVARIANT_WRITER_BYTE selects direct byte writer.
const INVARIANT_WRITER_BYTE = INVARIANT_WRITER_FINISH + 1

// INVARIANT_WRITER_BYTES selects direct byte-slice writer.
const INVARIANT_WRITER_BYTES = INVARIANT_WRITER_BYTE + 1

// INVARIANT_WRITER_OPERATION_COUNT covers each direct writer operation.
const INVARIANT_WRITER_OPERATION_COUNT = INVARIANT_WRITER_BYTES + 1

// INVARIANT_READER_ALIGN selects reader alignment.
const INVARIANT_READER_ALIGN = 0

// INVARIANT_READER_BLOCK selects block decoder.
const INVARIANT_READER_BLOCK = INVARIANT_READER_ALIGN + 1

// INVARIANT_READER_STORED selects stored decoder.
const INVARIANT_READER_STORED = INVARIANT_READER_BLOCK + 1

// INVARIANT_READER_CODE_SIZES selects dynamic code-size decoder.
const INVARIANT_READER_CODE_SIZES = INVARIANT_READER_STORED + 1

// INVARIANT_READER_DECODERS selects dynamic decoder construction.
const INVARIANT_READER_DECODERS = INVARIANT_READER_CODE_SIZES + 1

// INVARIANT_READER_SIZES selects dynamic size decoder.
const INVARIANT_READER_SIZES = INVARIANT_READER_DECODERS + 1

// INVARIANT_READER_HUFFMAN selects Huffman decoder.
const INVARIANT_READER_HUFFMAN = INVARIANT_READER_SIZES + 1

// INVARIANT_READER_MATCH selects match decoder.
const INVARIANT_READER_MATCH = INVARIANT_READER_HUFFMAN + 1

// INVARIANT_READER_REPEAT selects repeated-size decoder.
const INVARIANT_READER_REPEAT = INVARIANT_READER_MATCH + 1

// INVARIANT_READER_OPERATION_COUNT covers each direct reader operation.
const INVARIANT_READER_OPERATION_COUNT = INVARIANT_READER_REPEAT + 1

// INVARIANT_FINAL_HASH_BYTE_COUNT fills final match prefix and one mismatch byte.
const INVARIANT_FINAL_HASH_BYTE_COUNT = HASH_INPUT_BYTE_COUNT + 1

// INVARIANT_REPEATED_BYTE is nonzero match-search fixture byte.
const INVARIANT_REPEATED_BYTE = 7

// INVARIANT_DOMAIN_MINIMUM starts boundary witness sets.
const INVARIANT_DOMAIN_MINIMUM = 0

// INVARIANT_DOMAIN_SECOND follows minimum boundary witness.
const INVARIANT_DOMAIN_SECOND = INVARIANT_DOMAIN_MINIMUM + 1

// INVARIANT_DOMAIN_THIRD follows second boundary witness.
const INVARIANT_DOMAIN_THIRD = INVARIANT_DOMAIN_SECOND + 1

// INVARIANT_DOMAIN_BOUNDARY_COUNT covers zero and both following witnesses.
const INVARIANT_DOMAIN_BOUNDARY_COUNT = INVARIANT_DOMAIN_THIRD + FINAL_BIT_COUNT

// INVARIANT_FIXED_MATCH_STATE_COUNT adds the independent maximum witness.
const INVARIANT_FIXED_MATCH_STATE_COUNT = INVARIANT_DOMAIN_BOUNDARY_COUNT + FINAL_BIT_COUNT

// Test_Constant_Formulas protects shared machine limits and RFC relationships.
func Test_Constant_Formulas(t *testing.T) {
	testify.Equal(
		t, 1<<BYTE_COUNT_MEBIBYTE_EXPONENT, BYTE_COUNT_MEBIBYTE_COUNT,
	)
	testify.Equal(
		t,
		BYTE_COUNT_MEBIBYTE_COUNT*bits.MEBIBYTE_BYTES,
		BYTE_COUNT_MAXIMUM,
	)
	testify.Equal(t, 1<<WINDOW_BIT_COUNT, DICTIONARY_SIZE_MAXIMUM)
	testify.Equal(
		t, WINDOW_KIBIBYTE_COUNT*bits.KIBIBYTE_BYTES,
		DICTIONARY_SIZE_MAXIMUM,
	)
	testify.Equal_Values(t, bits.WORD_16_MAXIMUM, STORED_SIZE_MAXIMUM)
	testify.Equal(t, bits.BIT_COUNT_16_MAXIMUM, HASH_BIT_COUNT)
	testify.Equal_Values(t, bits.BIT_COUNT_16_MAXIMUM, BIT_COUNT_MAXIMUM)
	testify.Equal_Values(t, bits.WORD_8_MAXIMUM, BYTE_VALUE_MAXIMUM)
	testify.Equal_Values(t, bits.INTEGER_8_MINIMUM, LEVEL_UNVALIDATED_MINIMUM)
	testify.Equal_Values(t, bits.INTEGER_8_MAXIMUM, LEVEL_UNVALIDATED_MAXIMUM)
	testify.Equal(
		t, MATCH_SYMBOL_MINIMUM+MATCH_SYMBOL_COUNT-1, MATCH_SYMBOL_MAXIMUM,
	)
	testify.Equal(
		t, LITERAL_COUNT_MAXIMUM+FIXED_LITERAL_RESERVED_COUNT,
		FIXED_LITERAL_COUNT,
	)
	testify.Equal(
		t, FIXED_DISTANCE_COUNT-DISTANCE_SYMBOL_RESERVED_COUNT,
		DISTANCE_SYMBOL_COUNT,
	)
	testify.Equal(
		t, FINAL_BIT_COUNT+BLOCK_KIND_BIT_COUNT, BLOCK_HEADER_BIT_COUNT,
	)
	testify.Equal(
		t,
		REPEAT_ZERO_LONG_BASE+(1<<REPEAT_ZERO_LONG_EXTRA_BIT_COUNT)-1,
		REPEAT_COUNT_MAXIMUM,
	)
	testify.Equal(t, STATUS_OK+1, STATUS_INPUT_INVALID)
	testify.Equal(t, STATUS_INPUT_INVALID+1, STATUS_OUTPUT_TOO_SMALL)
	testify.Equal(t, STATUS_OUTPUT_TOO_SMALL+1, STATUS_LEVEL_INVALID)
	testify.Equal(t, STATUS_LEVEL_INVALID+1, STATUS_STORAGE_INVALID)
}

// Test_Bit_Primitive_Domains proves exact field widths at both bit boundaries.
func Test_Bit_Primitive_Domains(t *testing.T) {
	cases := [...]struct {
		Count Bit_Count
		Value Bit_Value
	}{
		{INVARIANT_DOMAIN_MINIMUM, INVARIANT_DOMAIN_MINIMUM},
		{INVARIANT_DOMAIN_SECOND, INVARIANT_DOMAIN_SECOND},
		{INVARIANT_DOMAIN_THIRD, INVARIANT_DOMAIN_THIRD},
		{BIT_COUNT_MAXIMUM, Bit_Value(BIT_VALUE_MAXIMUM)},
	}
	for _, test_case := range cases {
		var encoded [INVARIANT_BIT_STORAGE_COUNT]byte
		writer := Bit_Writer{Destination: encoded[:]}
		bit_writer_write_bits(Bit_Writer_Handle(&writer), test_case.Value, test_case.Count)
		bit_writer_finish(Bit_Writer_Handle(&writer))
		testify.Not_Equal(
			t, WRITER_STATE_EXHAUSTED, writer.State,
			"write %d bits failed", test_case.Count,
		)

		reader := Bit_Reader{Source: encoded[:writer.Position]}
		value, available := bit_reader_read(Bit_Reader_Handle(&reader), test_case.Count)
		testify.True(t, bool(available), "read %d bits unavailable", test_case.Count)
		testify.Equal(
			t, test_case.Value, value,
			"read %d bits = (%d, %t)", test_case.Count, value, available,
		)
		testify.Equal(
			t,
			test_reverse_bits(test_case.Value, test_case.Count),
			reverse_low_bits(test_case.Value, test_case.Count),
			"reverse %d bits failed", test_case.Count,
		)
	}

	for _, value := range []Byte_Value{
		INVARIANT_DOMAIN_MINIMUM,
		INVARIANT_DOMAIN_SECOND,
		INVARIANT_DOMAIN_THIRD,
		Byte_Value(BYTE_VALUE_MAXIMUM),
	} {
		var encoded [INVARIANT_BYTE_STORAGE_COUNT]byte
		writer := Bit_Writer{Destination: encoded[:]}
		bit_writer_write_byte(Bit_Writer_Handle(&writer), value)
		testify.Not_Equal(
			t, WRITER_STATE_EXHAUSTED, writer.State, "byte write %d failed", value,
		)
		testify.Equal(t, byte(value), encoded[0], "byte write %d failed", value)
	}

	maximum_storage := make([]byte, BYTE_COUNT_MAXIMUM)
	writer := Bit_Writer{
		Destination: maximum_storage,
		Position:    BYTE_COUNT_MAXIMUM,
		Bits:        Pending_Bits(PENDING_BITS_MAXIMUM),
		Bits_Count:  Pending_Bit_Count(PENDING_BIT_COUNT_MAXIMUM),
		State:       WRITER_STATE_EXHAUSTED,
	}
	bit_writer_write_bits(Bit_Writer_Handle(&writer), 0, 0)
	reader := Bit_Reader{
		Source:     maximum_storage,
		Position:   BYTE_COUNT_MAXIMUM,
		Bits:       Pending_Bits(PENDING_BITS_MAXIMUM),
		Bits_Count: Pending_Bit_Count(PENDING_BIT_COUNT_MAXIMUM),
	}
	_, available := bit_reader_read(Bit_Reader_Handle(&reader), 0)
	testify.True(t, bool(available), "maximum reader state unavailable")
}

// Test_Writer_State_Domains covers each boundary because writers resume mid-byte.
func Test_Writer_State_Domains(t *testing.T) {
	var one [INVARIANT_BYTE_STORAGE_COUNT]byte
	var two [INVARIANT_BIT_STORAGE_COUNT]byte
	maximum := make([]byte, BYTE_COUNT_MAXIMUM)
	states := [...]Bit_Writer{
		{},
		{
			Destination: one[:], Position: INVARIANT_DOMAIN_SECOND,
			Bits: INVARIANT_DOMAIN_SECOND, Bits_Count: INVARIANT_DOMAIN_SECOND,
		},
		{
			Destination: two[:], Position: INVARIANT_DOMAIN_THIRD,
			Bits: INVARIANT_DOMAIN_THIRD, Bits_Count: INVARIANT_DOMAIN_THIRD,
		},
		{
			Destination: maximum,
			Position:    BYTE_COUNT_MAXIMUM,
			Bits:        Pending_Bits(PENDING_BITS_MAXIMUM),
			Bits_Count:  Pending_Bit_Count(PENDING_BIT_COUNT_MAXIMUM),
			State:       WRITER_STATE_EXHAUSTED,
		},
	}
	var workspace_storage struct {
		Heads    [HASH_COUNT]int32
		Previous [WINDOW_SIZE]int32
	}
	workspace := Workspace{
		Heads:    workspace_storage.Heads[:],
		Previous: workspace_storage.Previous[:],
	}
	for _, state := range states {
		operation_count := INVARIANT_WRITER_OPERATION_COUNT
		for operation_index := 0; operation_index < operation_count; operation_index++ {
			writer := state
			writer_handle := Bit_Writer_Handle(&writer)
			switch operation_index {
			case INVARIANT_WRITER_ENCODE_STORED:
				encode_stored(writer_handle, nil)
			case INVARIANT_WRITER_STORED_BLOCK:
				stored_block(writer_handle, nil, true)
			case INVARIANT_WRITER_ENCODE_LITERALS:
				encode_literals(writer_handle, nil)
			case INVARIANT_WRITER_ENCODE_FIXED:
				encode_fixed(writer_handle, workspace, nil, nil, BEST_SPEED)
			case INVARIANT_WRITER_FIXED_SYMBOL:
				write_fixed_symbol(writer_handle, Fixed_Symbol(SYMBOL_MINIMUM))
			case INVARIANT_WRITER_FIXED_DISTANCE:
				write_fixed_distance(
					writer_handle, Distance_Symbol(DISTANCE_SYMBOL_MINIMUM),
				)
			case INVARIANT_WRITER_ALIGN:
				bit_writer_align(writer_handle)
			case INVARIANT_WRITER_FINISH:
				bit_writer_finish(writer_handle)
			case INVARIANT_WRITER_BYTE:
				bit_writer_write_byte(writer_handle, Byte_Value(BYTE_VALUE_MINIMUM))
			case INVARIANT_WRITER_BYTES:
				bit_writer_write_bytes(writer_handle, nil)
			}
			testify.Less_Or_Equal(
				t,
				&testify.Less_Or_Equal_Input[int]{
					First:  int(writer.Position),
					Second: len(writer.Destination),
				},
				"writer operation %d escaped destination", operation_index,
			)
		}
	}
}

// Test_Fast_Writer_State_Domains keeps the specialized path under the same boundaries.
func Test_Fast_Writer_State_Domains(t *testing.T) {
	var one [INVARIANT_BYTE_STORAGE_COUNT]byte
	var two [INVARIANT_BIT_STORAGE_COUNT]byte
	maximum := make([]byte, BYTE_COUNT_MAXIMUM)
	states := [...]Bit_Writer{
		{},
		{
			Destination: one[:], Position: INVARIANT_DOMAIN_SECOND,
			Bits: INVARIANT_DOMAIN_SECOND, Bits_Count: INVARIANT_DOMAIN_SECOND,
		},
		{
			Destination: two[:], Position: INVARIANT_DOMAIN_THIRD,
			Bits: INVARIANT_DOMAIN_THIRD, Bits_Count: INVARIANT_DOMAIN_THIRD,
		},
		{
			Destination: maximum, Position: BYTE_COUNT_MAXIMUM,
			Bits:       Pending_Bits(PENDING_BITS_MAXIMUM),
			Bits_Count: Pending_Bit_Count(PENDING_BIT_COUNT_MAXIMUM),
			State:      WRITER_STATE_EXHAUSTED,
		},
	}
	sources := [...]Source{nil, one[:], two[:], maximum}
	match_sizes := [...]Match_Size{
		MATCH_SIZE_MINIMUM, MATCH_SIZE_MINIMUM,
		MATCH_SIZE_MAXIMUM, MATCH_SIZE_MAXIMUM,
	}
	distance_sizes := [...]Distance{
		DISTANCE_MINIMUM,
		DISTANCE_MINIMUM + INVARIANT_DOMAIN_SECOND,
		DISTANCE_MAXIMUM,
		DISTANCE_MAXIMUM,
	}
	var heads [HASH_COUNT]int32
	for state_index, state := range states {
		writer := state
		encode_fixed_best_speed(Bit_Writer_Handle(&writer), heads[:], sources[state_index])
		testify.Less_Or_Equal(
			t,
			&testify.Less_Or_Equal_Input[int]{
				First: int(writer.Position), Second: len(writer.Destination),
			},
			"fast writer state %d escaped destination", state_index,
		)

		writer = state
		write_fixed_match(
			Bit_Writer_Handle(&writer), match_sizes[state_index],
			distance_sizes[state_index],
		)
		testify.Less_Or_Equal(
			t,
			&testify.Less_Or_Equal_Input[int]{
				First: int(writer.Position), Second: len(writer.Destination),
			},
			"fixed match state %d escaped destination", state_index,
		)
	}
}

// Test_Reader_State_Domains covers each boundary because input may end mid-code.
func Test_Reader_State_Domains(t *testing.T) {
	maximum := make([]byte, BYTE_COUNT_MAXIMUM)
	states := [...]Bit_Reader{
		{},
		{
			Source: []byte{INVARIANT_DOMAIN_MINIMUM}, Position: INVARIANT_DOMAIN_SECOND,
			Bits: INVARIANT_DOMAIN_SECOND, Bits_Count: INVARIANT_DOMAIN_SECOND,
		},
		{
			Source:   []byte{INVARIANT_DOMAIN_MINIMUM, INVARIANT_DOMAIN_MINIMUM},
			Position: INVARIANT_DOMAIN_THIRD,
			Bits:     INVARIANT_DOMAIN_THIRD, Bits_Count: INVARIANT_DOMAIN_THIRD,
		},
		{
			Source: maximum, Position: BYTE_COUNT_MAXIMUM,
			Bits:       Pending_Bits(PENDING_BITS_MAXIMUM),
			Bits_Count: Pending_Bit_Count(PENDING_BIT_COUNT_MAXIMUM),
		},
	}
	var code_sizes [CODE_COUNT]uint8
	var dynamic_storage [DYNAMIC_SIZES_MINIMUM]uint8
	for _, state := range states {
		operation_count := INVARIANT_READER_OPERATION_COUNT
		for operation_index := 0; operation_index < operation_count; operation_index++ {
			reader := state
			decoder := test_huffman_decoder()
			reader_handle := Bit_Reader_Handle(&reader)
			count, status := Count(0), Block_Status(STATUS_INPUT_INVALID)
			valid := Boolean(false)
			switch operation_index {
			case INVARIANT_READER_ALIGN:
				bit_reader_align(reader_handle)
			case INVARIANT_READER_BLOCK:
				count, status = decode_block(
					reader_handle, nil, nil, 0, BLOCK_KIND_RESERVED,
				)
			case INVARIANT_READER_STORED:
				count, status = decode_stored(reader_handle, nil, 0)
			case INVARIANT_READER_CODE_SIZES:
				valid = dynamic_code_sizes(
					reader_handle, code_sizes[:], ENCODED_CODE_COUNT_MINIMUM,
				)
			case INVARIANT_READER_DECODERS:
				valid = dynamic_decoders(reader_handle, decoder, decoder)
			case INVARIANT_READER_SIZES:
				valid = dynamic_sizes(reader_handle, decoder, dynamic_storage[:])
			case INVARIANT_READER_HUFFMAN:
				fixed_reader := reader
				decode_fixed(Bit_Reader_Handle(&fixed_reader), nil, nil, 0)
				count, status = decode_huffman(
					reader_handle, nil, nil, 0, decoder, decoder,
				)
			case INVARIANT_READER_MATCH:
				count, status = decode_match(
					reader_handle, nil, nil, 0, MATCH_SYMBOL_MINIMUM, decoder,
				)
			case INVARIANT_READER_REPEAT:
				next, next_valid := repeated_size(
					reader_handle, dynamic_storage[:], 0, BIT_COUNT_MAXIMUM,
				)
				count, valid = Count(next), next_valid
			}
			verify_reader_operation(t, operation_index, count, status, valid)
		}
	}
}

// Test_Fixed_Reader_State_Domains keeps the wide cursor bounded at every handoff.
func Test_Fixed_Reader_State_Domains(t *testing.T) {
	maximum := make([]byte, BYTE_COUNT_MAXIMUM)
	for state_index, state := range fixed_match_test_states(maximum) {
		decoded_cursor, decoded_match := state.Cursor, state.Match
		fixed_match_read(
			state.Source, Fixed_Bit_Cursor_Handle(&decoded_cursor), state.Symbol,
			Fixed_Match_Handle(&decoded_match),
		)
		decoded_position := int(decoded_cursor.Position)
		testify.Less_Or_Equal(
			t,
			&testify.Less_Or_Equal_Input[int]{
				First: decoded_position, Second: len(state.Source),
			},
			"fixed reader state %d escaped source", state_index,
		)

		reader := Bit_Reader{Source: state.Source, Position: state.Cursor.Position}
		bit_reader_store_raw(
			Bit_Reader_Handle(&reader), Fixed_Bit_Cursor_Handle(&state.Cursor),
		)
		testify.Less_Or_Equal(
			t,
			&testify.Less_Or_Equal_Input[int]{
				First: int(reader.Position), Second: len(reader.Source),
			},
			"stored reader state %d escaped source", state_index,
		)
		testify.Less_Or_Equal(
			t,
			&testify.Less_Or_Equal_Input[Pending_Bit_Count]{
				First:  reader.Bits_Count,
				Second: Pending_Bit_Count(PENDING_BIT_COUNT_MAXIMUM),
			},
			"stored reader state %d retained too many bits", state_index,
		)
	}
}

// Holds one real fixed-match operation at an invariant boundary.
type fixed_match_test_state struct {
	Source Bit_Storage
	Cursor Fixed_Bit_Cursor
	Symbol Match_Symbol
	Match  Fixed_Match
}

// Covers minimum, interior, and maximum fixed-match state through real operations.
func fixed_match_test_states(
	maximum Bit_Storage,
) (states []fixed_match_test_state) {
	return []fixed_match_test_state{
		{
			Symbol: MATCH_SYMBOL_MINIMUM,
			Match: Fixed_Match{
				Size: MATCH_SIZE_MINIMUM, Distance: DISTANCE_MINIMUM, Valid: true,
			},
		},
		{
			Source: []byte{INVARIANT_DOMAIN_MINIMUM},
			Cursor: Fixed_Bit_Cursor{
				Position: INVARIANT_DOMAIN_SECOND,
				Bits:     INVARIANT_DOMAIN_SECOND, Count: INVARIANT_DOMAIN_SECOND,
			},
			Symbol: MATCH_SYMBOL_MINIMUM,
			Match: Fixed_Match{
				Size:     MATCH_SIZE_MINIMUM,
				Distance: DISTANCE_MINIMUM + INVARIANT_DOMAIN_SECOND, Valid: true,
			},
		},
		{
			Source: []byte{
				INVARIANT_DOMAIN_MINIMUM, INVARIANT_DOMAIN_MINIMUM,
			},
			Cursor: Fixed_Bit_Cursor{
				Position: INVARIANT_DOMAIN_THIRD,
				Bits:     INVARIANT_DOMAIN_THIRD, Count: INVARIANT_DOMAIN_THIRD,
			},
			Symbol: MATCH_SYMBOL_MAXIMUM,
			Match: Fixed_Match{
				Size: MATCH_SIZE_MAXIMUM, Distance: DISTANCE_MAXIMUM, Valid: true,
			},
		},
		{
			Source: maximum,
			Cursor: Fixed_Bit_Cursor{
				Position: BYTE_COUNT_MAXIMUM,
				Bits:     Bit_Value(BIT_VALUE_MAXIMUM),
				Count:    Bit_Count(BIT_COUNT_MAXIMUM),
			},
			Symbol: MATCH_SYMBOL_MAXIMUM,
			Match: Fixed_Match{
				Size: MATCH_SIZE_MAXIMUM, Distance: DISTANCE_MAXIMUM, Valid: false,
			},
		},
	}
}

func verify_reader_operation(
	t *testing.T,
	operation_index int,
	count Count,
	status Block_Status,
	valid Boolean,
) {
	t.Helper()
	testify.Zero(
		t, count, "reader operation %d wrote %d bytes", operation_index, count,
	)
	testify.Equal(
		t, Block_Status(STATUS_INPUT_INVALID), status,
		"reader operation %d status = %d", operation_index, status,
	)
	testify.Equal(
		t, Boolean(false), valid,
		"reader operation %d accepted input", operation_index,
	)
}

// Test_Code_Domains proves encoder and decoder tables are exact inverses.
func Test_Code_Domains(t *testing.T) {
	for size := Match_Size(MATCH_SIZE_MINIMUM); size <= MATCH_SIZE_MAXIMUM; size++ {
		symbol, extra, extra_count := match_code(size)
		base, decoded_extra_count := decoded_match_code(symbol)
		testify.Equal(
			t, decoded_extra_count, extra_count,
			"match size %d suffix widths differ", size,
		)
		testify.Equal(
			t, size, base+Match_Size(extra), "match size %d round trip failed", size,
		)
	}

	for distance := Distance(DISTANCE_MINIMUM); distance <= DISTANCE_MAXIMUM; distance++ {
		symbol, extra, extra_count := distance_code(distance)
		base, decoded_extra_count := decoded_distance_code(symbol)
		testify.Equal(
			t, decoded_extra_count, extra_count,
			"distance %d suffix widths differ", distance,
		)
		testify.Equal(
			t, distance, Distance(base)+Distance(extra),
			"distance %d round trip failed", distance,
		)
	}

	var encoded [INVARIANT_CODE_STORAGE_COUNT]byte
	writer := Bit_Writer{Destination: encoded[:]}
	for _, symbol := range []Fixed_Symbol{
		INVARIANT_DOMAIN_MINIMUM,
		INVARIANT_DOMAIN_SECOND,
		INVARIANT_DOMAIN_THIRD,
		LITERAL_SYMBOL_MAXIMUM,
	} {
		write_fixed_symbol(Bit_Writer_Handle(&writer), symbol)
	}
	for _, symbol := range []Distance_Symbol{
		INVARIANT_DOMAIN_MINIMUM,
		INVARIANT_DOMAIN_SECOND,
		INVARIANT_DOMAIN_THIRD,
		DISTANCE_SYMBOL_COUNT - 1,
	} {
		write_fixed_distance(Bit_Writer_Handle(&writer), symbol)
	}
}

// Test_Huffman_Domains proves tiny, full, and malformed canonical alphabets.
func Test_Huffman_Domains(t *testing.T) {
	for _, test_case := range [...]struct {
		Sizes []uint8
		Valid Boolean
	}{
		{[]uint8{0}, true},
		{[]uint8{1}, true},
		{[]uint8{1, 1}, true},
		{[]uint8{2, 2}, false},
		{test_maximum_code_sizes(), true},
	} {
		decoder := test_huffman_decoder()
		valid := huffman_build(decoder, Code_Sizes(test_case.Sizes))
		testify.Equal(
			t, test_case.Valid, valid,
			"Huffman sizes %v validity = %t", test_case.Sizes, valid,
		)
	}
}

// Test_Huffman_Read_Domains makes every legal decoder bound consume bits.
func Test_Huffman_Read_Domains(t *testing.T) {
	for _, wanted := range []int{INVARIANT_DOMAIN_THIRD, SYMBOL_MAXIMUM} {
		var sizes [FIXED_LITERAL_COUNT]uint8
		sizes[wanted] = 1
		decoder := test_huffman_decoder()
		testify.Equal(
			t, Boolean(true), huffman_build(decoder, Code_Sizes(sizes[:wanted+1])),
			"Huffman symbol %d build failed", wanted,
		)
		reader := Bit_Reader{Source: []byte{0}}
		symbol, available := huffman_read(Bit_Reader_Handle(&reader), decoder)
		testify.True(t, bool(available), "Huffman symbol %d unavailable", wanted)
		testify.Equal(
			t, wanted, int(symbol),
			"Huffman symbol %d = (%d, %t)", wanted, symbol, available,
		)
	}

	for _, sizes := range [][]uint8{
		nil,
		{1},
		{2, 2, 2, 2},
		test_maximum_code_sizes(),
	} {
		decoder := test_huffman_decoder()
		if len(sizes) > 0 {
			testify.Equal(
				t, Boolean(true), huffman_build(decoder, sizes),
				"Huffman read sizes %v build failed", sizes,
			)
		}
		var source []byte
		if len(sizes) > 0 {
			source = make([]byte, INVARIANT_BIT_STORAGE_COUNT)
		}
		reader := Bit_Reader{Source: source}
		symbol, available := huffman_read(Bit_Reader_Handle(&reader), decoder)
		if len(sizes) == 0 {
			testify.False(t, bool(available), "empty Huffman alphabet decoded a symbol")
			continue
		}
		testify.True(t, bool(available), "Huffman read sizes %v unavailable", sizes)
		testify.Zero(t, symbol, "Huffman read sizes %v decoded %d", sizes, symbol)
	}
	maximum_source := make([]byte, BYTE_COUNT_MAXIMUM)
	reader := Bit_Reader{
		Source: maximum_source, Position: BYTE_COUNT_MAXIMUM,
	}
	decoder := test_huffman_decoder()
	symbol, available := huffman_read(Bit_Reader_Handle(&reader), decoder)
	testify.False(
		t, bool(available),
		"empty Huffman alphabet decoded at final source position",
	)
	testify.Zero(t, symbol, "empty Huffman alphabet decoded %d", symbol)
}

// Test_Huffman_State_Domains drives each decoder argument through real reads.
func Test_Huffman_State_Domains(t *testing.T) {
	for _, sizes := range []Code_Sizes{
		{0},
		{1},
		{1, 1},
		{2, 2, 2, 2},
		test_maximum_code_sizes(),
		test_fixed_code_sizes(),
	} {
		decoder := test_huffman_decoder()
		testify.Equal(
			t, Boolean(true), huffman_build(decoder, sizes),
			"Huffman state sizes %v build failed", sizes,
		)
		testify.Equal(
			t, Boolean(true), huffman_build(decoder, sizes),
			"Huffman state sizes %v rebuild failed", sizes,
		)
		literal_decoder := test_huffman_decoder()
		distance_decoder := test_huffman_decoder()
		reader := Bit_Reader{}
		reader_handle := Bit_Reader_Handle(&reader)
		valid := dynamic_decoders(reader_handle, literal_decoder, distance_decoder)
		testify.Equal(
			t, Boolean(false), valid,
			"dynamic state sizes %v decoded absent input", sizes,
		)

		var dynamic_storage [DYNAMIC_SIZES_MINIMUM]uint8
		valid = dynamic_sizes(reader_handle, decoder, dynamic_storage[:])
		testify.Equal(
			t, Boolean(false), valid,
			"Huffman state sizes %v decoded absent input", sizes,
		)

		count, status := decode_huffman(
			reader_handle,
			nil, nil, 0, decoder, decoder,
		)
		testify.Zero(t, count, "Huffman state sizes %v wrote %d bytes", sizes, count)
		testify.Equal(
			t, Block_Status(STATUS_INPUT_INVALID), status,
			"Huffman state sizes %v status = %d", sizes, status,
		)

		count, status = decode_match(
			reader_handle,
			nil, nil, 0, MATCH_SYMBOL_MINIMUM, decoder,
		)
		testify.Zero(t, count, "Huffman match sizes %v wrote %d bytes", sizes, count)
		testify.Equal(
			t, Block_Status(STATUS_INPUT_INVALID), status,
			"Huffman match sizes %v status = %d", sizes, status,
		)
	}
	verify_decode_huffman_state_domains(t)
}

func verify_decode_huffman_state_domains(t *testing.T) {
	t.Helper()
	maximum_destination := make([]byte, BYTE_COUNT_MAXIMUM)
	maximum_history := make([]byte, HISTORY_SIZE_MAXIMUM)
	for _, test_case := range []struct {
		Destination Destination
		History     History
		Count       Count
	}{
		{
			make([]byte, INVARIANT_DOMAIN_SECOND),
			make([]byte, INVARIANT_DOMAIN_SECOND), INVARIANT_DOMAIN_SECOND,
		},
		{
			make([]byte, INVARIANT_DOMAIN_THIRD),
			make([]byte, INVARIANT_DOMAIN_THIRD), INVARIANT_DOMAIN_THIRD,
		},
		{maximum_destination, maximum_history, BYTE_COUNT_MAXIMUM},
	} {
		reader := Bit_Reader{}
		decoder := test_huffman_decoder()
		count, status := decode_huffman(
			Bit_Reader_Handle(&reader), test_case.Destination, test_case.History,
			test_case.Count, decoder, decoder,
		)
		testify.Equal(t, test_case.Count, count, "empty Huffman input changed count")
		testify.Equal(
			t, Block_Status(STATUS_INPUT_INVALID), status,
			"empty Huffman input status = %d", status,
		)
	}
}

// Test_History_Position_Domains proves final legal virtual positions do real work.
func Test_History_Position_Domains(t *testing.T) {
	for _, test_case := range [...]struct {
		Value [INVARIANT_HASH_INPUT_COUNT]byte
		Hash  Hash
	}{
		{[INVARIANT_HASH_INPUT_COUNT]byte{0, 0, 0}, 0},
		{[INVARIANT_HASH_INPUT_COUNT]byte{0, 109, 103}, 1},
		{[INVARIANT_HASH_INPUT_COUNT]byte{0, 253, 254}, 2},
		{[INVARIANT_HASH_INPUT_COUNT]byte{1, 212, 245}, HASH_MAXIMUM},
	} {
		testify.Equal(
			t, test_case.Hash, sequence_hash(nil, test_case.Value[:], 0),
			"hash %v did not reach %d", test_case.Value, test_case.Hash,
		)
	}

	testify.Equal(
		t, Hash(1), sequence_hash(History{0, 109}, Source{103}, 0),
		"one-byte source hash failed",
	)
	testify.Equal(
		t, Hash(1), sequence_hash(History{0}, Source{109, 103}, 0),
		"two-byte source hash failed",
	)

	source := make([]byte, BYTE_COUNT_MAXIMUM)
	history := make([]byte, DICTIONARY_SIZE_MAXIMUM)
	for index := len(source) - INVARIANT_FINAL_HASH_BYTE_COUNT; index < len(source); index++ {
		source[index] = BYTE_VALUE_MAXIMUM
	}
	second := Sequence_Position(SEQUENCE_POSITION_MAXIMUM)
	candidate := Match_Position(MATCH_POSITION_MAXIMUM)
	testify.Equal(
		t,
		Byte_Value(BYTE_VALUE_MAXIMUM),
		history_byte(
			history, source, Virtual_Byte_Position(VIRTUAL_BYTE_POSITION_MAXIMUM),
		),
		"final virtual byte lookup failed",
	)

	var workspace_storage struct {
		Heads    [HASH_COUNT]int32
		Previous [WINDOW_SIZE]int32
	}
	workspace := Workspace{
		Heads:    workspace_storage.Heads[:],
		Previous: workspace_storage.Previous[:],
	}
	hash := sequence_hash(history, source, second)
	workspace.Heads[hash] = int32(candidate + 1)
	match := match_search(
		workspace, history, source, second, BEST_COMPRESSION,
	)
	testify.Equal(t, Boolean(true), match.Present, "final match absent")
	testify.Equal(
		t, candidate, match.Position, "final match position = %d", match.Position,
	)
	testify.Equal(
		t, Match_Size(MATCH_SIZE_MINIMUM), match.Size,
		"final match size = %d", match.Size,
	)
	workspace_insert(workspace, history, source, second)
	testify.Equal(
		t, int32(second+1), workspace.Heads[hash], "final workspace insertion failed",
	)

	verify_small_match_positions(t, &workspace)
}

// Candidate positions one and two must survive hashing.
func verify_small_match_positions(t *testing.T, workspace *Workspace) {
	t.Helper()
	var small_source [INVARIANT_SMALL_SOURCE_COUNT]byte
	for index := range small_source {
		small_source[index] = INVARIANT_REPEATED_BYTE
	}
	for _, wanted := range []Match_Position{
		INVARIANT_DOMAIN_SECOND, INVARIANT_DOMAIN_THIRD,
	} {
		clear(workspace.Heads)
		clear(workspace.Previous)
		later := Sequence_Position(wanted + MATCH_SIZE_MINIMUM)
		hash := sequence_hash(nil, small_source[:], later)
		workspace.Heads[hash] = int32(wanted + 1)
		match := match_search(
			*workspace, nil, small_source[:], later, BEST_SPEED,
		)
		testify.Equal(
			t, Boolean(true), match.Present, "match position %d absent", wanted,
		)
		testify.Equal(
			t, wanted, match.Position, "match position %d = %d", wanted, match.Position,
		)
	}
}

// Test_Matching_Size_Domains proves mismatch positions and maximum match.
func Test_Matching_Size_Domains(t *testing.T) {
	var source [MATCH_SIZE_MAXIMUM * BINARY_RADIX]byte
	for _, wanted := range []int{
		INVARIANT_DOMAIN_MINIMUM,
		INVARIANT_DOMAIN_SECOND,
		INVARIANT_DOMAIN_THIRD,
		MATCH_SIZE_MAXIMUM,
	} {
		clear(source[:])
		if wanted < MATCH_SIZE_MAXIMUM {
			source[MATCH_SIZE_MAXIMUM+wanted] = 1
		}
		observed := matching_size(
			nil, source[:], 0, MATCH_SIZE_MAXIMUM, MATCH_SIZE_MAXIMUM,
		)
		testify.Equal(t, wanted, int(observed), "matching size %d = %d", wanted, observed)
	}

	history := History{0, 0, 0, 0, 0}
	for _, source_variant := range []Source{nil, {0}, {0, 0}} {
		for _, second := range []Later_Sequence_Position{
			INVARIANT_DOMAIN_SECOND, INVARIANT_DOMAIN_THIRD,
		} {
			observed := matching_size(
				history, source_variant, 0, second, MATCH_SIZE_MINIMUM,
			)
			testify.Equal(
				t, Matching_Size(MATCH_SIZE_MINIMUM), observed,
				"source %d second %d match = %d",
				len(source_variant), second, observed,
			)
		}
	}
	for _, history_size := range []int{
		INVARIANT_DOMAIN_SECOND, INVARIANT_DOMAIN_THIRD,
	} {
		observed := matching_size(
			History(history[:history_size]), Source{0, 0, 0, 0}, 0, 1,
			MATCH_SIZE_MINIMUM,
		)
		testify.Equal(
			t, Matching_Size(MATCH_SIZE_MINIMUM), observed,
			"history %d match = %d", history_size, observed,
		)
	}
	for _, first := range []Match_Position{
		INVARIANT_DOMAIN_SECOND, INVARIANT_DOMAIN_THIRD,
	} {
		observed := matching_size(
			nil, source[:], first,
			Later_Sequence_Position(first+MATCH_SIZE_MINIMUM),
			MATCH_SIZE_MINIMUM,
		)
		testify.Equal(
			t, Matching_Size(MATCH_SIZE_MINIMUM), observed,
			"first %d match = %d", first, observed,
		)
	}

	testify.Equal(
		t, Byte_Value(INVARIANT_DOMAIN_THIRD),
		history_byte(History{INVARIANT_DOMAIN_THIRD}, nil, 0),
		"history byte two failed",
	)
	testify.Equal(
		t, Byte_Value(INVARIANT_DOMAIN_SECOND),
		history_byte(nil, Source{INVARIANT_DOMAIN_SECOND}, POSITION_MINIMUM),
		"source byte one failed",
	)
}

// Test_Match_Copy_Domains proves nearest, second, and window-end history reads.
func Test_Match_Copy_Domains(t *testing.T) {
	for _, distance_size := range []Distance{
		INVARIANT_DOMAIN_SECOND, INVARIANT_DOMAIN_THIRD, DISTANCE_MAXIMUM,
	} {
		history := make([]byte, distance_size)
		history[0] = byte(distance_size)
		var destination [MATCH_SIZE_MINIMUM]byte
		count, copied := match_copy(
			destination[:], history, 0, MATCH_SIZE_MINIMUM, distance_size,
		)
		testify.Equal(
			t, Boolean(true), copied, "distance %d copy failed", distance_size,
		)
		testify.Equal(
			t, len(destination), int(count),
			"distance %d count = %d", distance_size, count,
		)
	}

	var compressed [INVARIANT_CODE_STORAGE_COUNT]byte
	writer := Bit_Writer{Destination: compressed[:]}
	bit_writer_write_bits(
		Bit_Writer_Handle(&writer),
		Bit_Value(BLOCK_KIND_FIXED<<FINAL_BIT_COUNT|FINAL_BLOCK_BIT_VALUE),
		BLOCK_HEADER_BIT_COUNT,
	)
	write_fixed_symbol(Bit_Writer_Handle(&writer), MATCH_SYMBOL_MINIMUM)
	bit_writer_write_bits(Bit_Writer_Handle(&writer), 0, FIXED_DISTANCE_BIT_COUNT)
	write_fixed_symbol(Bit_Writer_Handle(&writer), LITERAL_SYMBOL_END)
	bit_writer_finish(Bit_Writer_Handle(&writer))
	reader := Bit_Reader{Source: compressed[:writer.Position]}
	final, available := bit_reader_read(Bit_Reader_Handle(&reader), FINAL_BIT_COUNT)
	testify.True(t, bool(available), "final match block header unavailable")
	testify.Equal(
		t, Bit_Value(FINAL_BLOCK_BIT_VALUE), final,
		"final match block bit = %d", final,
	)
	block_kind, available := bit_reader_read(
		Bit_Reader_Handle(&reader), BLOCK_KIND_BIT_COUNT,
	)
	testify.True(t, bool(available), "match block kind unavailable")
	destination := make([]byte, BYTE_COUNT_MAXIMUM)
	count, status := decode_block(
		Bit_Reader_Handle(&reader), destination, History{'x'}, BYTE_COUNT_MAXIMUM,
		Block_Kind(block_kind),
	)
	testify.Equal(t, Count(BYTE_COUNT_MAXIMUM), count, "maximum match count = %d", count)
	testify.Equal(
		t, Block_Status(STATUS_OUTPUT_TOO_SMALL), status,
		"maximum match status = %d", status,
	)

}

// Test_Fast_Match_Copy_Domains keeps complete copies on every count and distance bound.
func Test_Fast_Match_Copy_Domains(t *testing.T) {
	for _, initial_count := range []Count{
		INVARIANT_DOMAIN_SECOND,
		INVARIANT_DOMAIN_THIRD,
	} {
		destination := make([]byte, int(initial_count)+MATCH_SIZE_MAXIMUM)
		count, copied := match_copy(
			destination, nil, initial_count, MATCH_SIZE_MAXIMUM, DISTANCE_MINIMUM,
		)
		testify.Equal(t, Boolean(true), copied, "count %d copy failed", initial_count)
		testify.Equal(
			t, initial_count+MATCH_SIZE_MAXIMUM, count,
			"count %d copy ended at %d", initial_count, count,
		)
	}

	for _, distance_size := range []Distance{
		DISTANCE_MINIMUM + INVARIANT_DOMAIN_SECOND,
		DISTANCE_MAXIMUM,
	} {
		history := make([]byte, distance_size)
		var destination [MATCH_SIZE_MINIMUM]byte
		count := Count(BYTE_COUNT_MINIMUM)
		match_copy_complete(
			Match_Destination(destination[:]), history, Match_Count(count),
			MATCH_SIZE_MINIMUM, distance_size,
		)
		count += MATCH_SIZE_MINIMUM
		testify.Equal(
			t, Count(MATCH_SIZE_MINIMUM), count,
			"distance %d fast copy count = %d", distance_size, count,
		)
	}

}

// Test_Decode_Match_Domains keeps dynamic matches observable after fixed specialization.
func Test_Decode_Match_Domains(t *testing.T) {
	var distance_sizes [DISTANCE_SYMBOL_COUNT]uint8
	distance_sizes[DISTANCE_SYMBOL_MINIMUM] = FINAL_BIT_COUNT
	distance_decoder := test_huffman_decoder()
	testify.Equal(
		t, Boolean(true), huffman_build(distance_decoder, distance_sizes[:]),
		"distance decoder build failed",
	)
	maximum_destination := make([]byte, BYTE_COUNT_MAXIMUM)
	maximum_dictionary := make([]byte, DICTIONARY_SIZE_MAXIMUM)
	for _, test_case := range [...]struct {
		Destination Destination
		Dictionary  History
		Count       Count
		Symbol      Match_Symbol
		Next_Count  Count
		Status      Block_Status
	}{
		{
			nil, nil, Count(BYTE_COUNT_MINIMUM), MATCH_SYMBOL_MINIMUM,
			Count(BYTE_COUNT_MINIMUM), STATUS_INPUT_INVALID,
		},
		{
			make([]byte, INVARIANT_DOMAIN_SECOND),
			make([]byte, INVARIANT_DOMAIN_SECOND),
			INVARIANT_DOMAIN_SECOND, MATCH_SYMBOL_MINIMUM,
			INVARIANT_DOMAIN_SECOND, STATUS_OUTPUT_TOO_SMALL,
		},
		{
			make([]byte, INVARIANT_DOMAIN_THIRD),
			make([]byte, INVARIANT_DOMAIN_THIRD),
			INVARIANT_DOMAIN_THIRD, MATCH_SYMBOL_MINIMUM,
			INVARIANT_DOMAIN_THIRD, STATUS_OUTPUT_TOO_SMALL,
		},
		{
			maximum_destination, maximum_dictionary, BYTE_COUNT_MAXIMUM,
			MATCH_SYMBOL_MAXIMUM, BYTE_COUNT_MAXIMUM, STATUS_OUTPUT_TOO_SMALL,
		},
		{
			make([]byte, MATCH_SIZE_MINIMUM), make([]byte, DISTANCE_MINIMUM),
			Count(BYTE_COUNT_MINIMUM), MATCH_SYMBOL_MINIMUM,
			MATCH_SIZE_MINIMUM, STATUS_OK,
		},
	} {
		verify_decode_match_case(
			t, distance_decoder, test_case.Destination, test_case.Dictionary,
			test_case.Count, test_case.Symbol, test_case.Next_Count, test_case.Status,
		)
	}
}

func verify_decode_match_case(
	t *testing.T,
	distance_decoder Huffman_Decoder,
	destination Destination,
	dictionary History,
	initial_count Count,
	symbol Match_Symbol,
	expected_count Count,
	expected_status Block_Status,
) {
	t.Helper()
	reader := Bit_Reader{Source: []byte{byte(BIT_VALUE_MINIMUM)}}
	count, status := decode_match(
		Bit_Reader_Handle(&reader), destination, dictionary,
		initial_count, symbol, distance_decoder,
	)
	testify.Equal(
		t, expected_count, count,
		"match count %d symbol %d ended at %d", initial_count, symbol, count,
	)
	testify.Equal(
		t, expected_status, status,
		"match count %d symbol %d status = %d", initial_count, symbol, status,
	)
}

// Test_Dynamic_Position_Domains proves failure cursors and exact completion.
func Test_Dynamic_Position_Domains(t *testing.T) {
	sizes := make([]uint8, DYNAMIC_SIZES_MAXIMUM)
	for _, position := range []Dynamic_Position{
		INVARIANT_DOMAIN_SECOND,
		INVARIANT_DOMAIN_THIRD,
		DYNAMIC_POSITION_MAXIMUM,
	} {
		reader := Bit_Reader{}
		next, valid := repeated_size(Bit_Reader_Handle(&reader), sizes, position, 17)
		testify.Equal(t, Boolean(false), valid, "repeat position %d valid", position)
		testify.Equal(
			t, Dynamic_Cursor(position), next,
			"repeat position %d = (%d, %t)", position, next, valid,
		)
	}
	reader := Bit_Reader{Source: []byte{0}}
	next, valid := repeated_size(
		Bit_Reader_Handle(&reader), sizes,
		DYNAMIC_POSITION_MAXIMUM-(REPEAT_ZERO_SHORT_BASE-1),
		REPEAT_SYMBOL_MIDDLE,
	)
	testify.Equal(t, Boolean(true), valid, "repeat completion invalid")
	testify.Equal(
		t, Dynamic_Cursor(DYNAMIC_SIZES_MAXIMUM), next,
		"repeat completion = (%d, %t)", next, valid,
	)
}

func test_reverse_bits(value Bit_Value, count Bit_Count) (reversed Bit_Value) {
	for index := Bit_Count(0); index < count; index++ {
		reversed = reversed<<FINAL_BIT_COUNT | value&LOW_BIT_MASK
		value >>= FINAL_BIT_COUNT
	}
	return reversed
}

func test_maximum_code_sizes() (sizes []uint8) {
	for size := uint8(1); size < CODE_SIZE_MAXIMUM; size++ {
		sizes = append(sizes, size)
	}
	return append(sizes, CODE_SIZE_MAXIMUM, CODE_SIZE_MAXIMUM)
}

func test_fixed_code_sizes() (sizes Code_Sizes) {
	var storage [FIXED_LITERAL_COUNT]uint8
	for index := 0; index <= FIXED_LITERAL_FIRST_MAXIMUM; index++ {
		storage[index] = FIXED_CODE_SIZE
	}
	for index := FIXED_LITERAL_SECOND_MINIMUM; index <= FIXED_LITERAL_SECOND_MAXIMUM; index++ {
		storage[index] = FIXED_SECOND_CODE_SIZE
	}
	for index := FIXED_LITERAL_THIRD_MINIMUM; index <= FIXED_LITERAL_THIRD_MAXIMUM; index++ {
		storage[index] = FIXED_THIRD_CODE_SIZE
	}
	for index := FIXED_LITERAL_FOURTH_MINIMUM; index < len(storage); index++ {
		storage[index] = FIXED_CODE_SIZE
	}
	return storage[:]
}

func test_huffman_decoder() (decoder Huffman_Decoder) {
	return Huffman_Decoder{
		Counts:         make(Huffman_Counts, HUFFMAN_COUNT_SIZE),
		Symbols:        make(Huffman_Symbols, FIXED_LITERAL_COUNT),
		Lookup_Symbols: make(Huffman_Lookup_Symbols, HUFFMAN_LOOKUP_COUNT),
		Lookup_Sizes:   make(Huffman_Lookup_Sizes, HUFFMAN_LOOKUP_COUNT),
	}
}
