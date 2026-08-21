package flate_test

import (
	"testing"

	"local/james-orcales/shared/compress/flate"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
)

// Test_Raw_DEFLATE protects independent stored, fixed, and dynamic wire forms.
func Test_Raw_DEFLATE(t *testing.T) {
	stored := test_stored()
	fixed := test_fixed()
	dynamic := test_dynamic()
	verify_decode(t, stored[:], "stored block payload")
	verify_decode(t, fixed[:], TEST_FIXED_CONTENT)
	verify_decode(t, dynamic[:], "aaaabbbbccccddddeeeeffffgggghhhh")
	var prefixed [TEST_FIXED_SIZE + TEST_PREFIX_SUFFIX_SIZE]byte
	copy(prefixed[:], fixed[:])
	prefix_count, compressed_count, prefix_status := flate.Decode_Prefix_Into(
		make([]byte, TEST_OUTPUT_SIZE), prefixed[:],
	)
	testify.Equal_Values(
		t, flate.STATUS_OK, prefix_status,
		"Decode_Prefix_Into status = %d", prefix_status,
	)
	testify.Equal(
		t, len(TEST_FIXED_CONTENT), int(prefix_count),
		"Decode_Prefix_Into count = %d", prefix_count,
	)
	testify.Equal(
		t, len(fixed), int(compressed_count),
		"Decode_Prefix_Into compressed count = %d", compressed_count,
	)

	var destination [TEST_OUTPUT_SIZE]byte
	var workspace_storage test_workspace
	workspace := test_workspace_value(&workspace_storage)
	for _, level := range []flate.Level{
		flate.HUFFMAN_ONLY,
		flate.DEFAULT_COMPRESSION,
		flate.NO_COMPRESSION,
		flate.BEST_SPEED,
		flate.COMPRESSION_LEVEL_2,
		flate.COMPRESSION_LEVEL_3,
		flate.COMPRESSION_LEVEL_4,
		flate.COMPRESSION_LEVEL_5,
		flate.COMPRESSION_LEVEL_6,
		flate.COMPRESSION_LEVEL_7,
		flate.COMPRESSION_LEVEL_8,
		flate.BEST_COMPRESSION,
	} {
		count, status := flate.Encode_Into(
			destination[:], workspace, []byte(TEST_CONTENT),
			flate.Level_Unvalidated(level),
		)
		testify.Equal_Values(
			t, flate.STATUS_OK, status,
			"Encode_Into level %d status = %d", level, status,
		)
		verify_decode(t, destination[:count], TEST_CONTENT)
	}
}

// Test_Compression_Levels keeps standard library level identities.
func Test_Compression_Levels(t *testing.T) {
	levels := [TEST_LEVEL_COUNT]struct {
		Observed flate.Level
		Want     int
	}{
		{flate.NO_COMPRESSION, TEST_NO_COMPRESSION_LEVEL},
		{flate.BEST_SPEED, TEST_BEST_SPEED_LEVEL},
		{flate.BEST_COMPRESSION, TEST_BEST_COMPRESSION_LEVEL},
		{flate.DEFAULT_COMPRESSION, TEST_DEFAULT_COMPRESSION_LEVEL},
		{flate.HUFFMAN_ONLY, TEST_HUFFMAN_ONLY_LEVEL},
	}
	for _, level := range levels {
		testify.Equal(
			t, level.Want, int(level.Observed),
			"level = %d, want %d", level.Observed, level.Want,
		)
	}

	var destination [TEST_OUTPUT_SIZE]byte
	var workspace_storage test_workspace
	workspace := test_workspace_value(&workspace_storage)
	for _, level := range []flate.Level_Unvalidated{
		flate.HUFFMAN_ONLY - 1,
		flate.BEST_COMPRESSION + 1,
	} {
		_, status := flate.Encode_Into(destination[:], workspace, nil, level)
		testify.Equal_Values(
			t, flate.STATUS_LEVEL_INVALID, status,
			"invalid level %d status = %d", level, status,
		)
	}
}

// Test_Bounds protects every caller-owned resource boundary.
func Test_Bounds(t *testing.T) {
	testify.Equal(
		t,
		flate.BYTE_COUNT_MEBIBYTE_COUNT*bits.MEBIBYTE_BYTES,
		flate.BYTE_COUNT_MAXIMUM,
		"byte boundary changed",
	)
	testify.Equal(
		t, 1<<flate.WINDOW_BIT_COUNT, flate.DICTIONARY_SIZE_MAXIMUM,
		"dictionary boundary changed",
	)

	compressed := test_fixed()
	var short [TEST_SHORT_SIZE]byte
	count, status := flate.Decode_Into(short[:], compressed[:])
	testify.Equal_Values(
		t, flate.STATUS_OUTPUT_TOO_SMALL, status, "short decode status = %d", status,
	)
	testify.Equal(t, len(short), int(count), "short decode count = %d", count)

	var workspace_storage test_workspace
	workspace := test_workspace_value(&workspace_storage)
	encoded_count, encode_status := flate.Encode_Into(
		short[:], workspace, []byte(TEST_CONTENT), flate.BEST_SPEED,
	)
	testify.Equal_Values(
		t, flate.STATUS_OUTPUT_TOO_SMALL, encode_status,
		"short encode status = %d", encode_status,
	)
	testify.Less_Or_Equal(
		t,
		&testify.Less_Or_Equal_Input[int]{First: int(encoded_count), Second: len(short)},
		"short encode count = %d", encoded_count,
	)
	verify_oversized_bounds(t, workspace)
}

// Test_Allocation proves every public result keeps caller ownership.
func Test_Allocation(t *testing.T) {
	verify_decode_allocation(t)
	verify_encode_allocation(t)
}

// Test_Dictionaries protects preset history without hidden window storage.
func Test_Dictionaries(t *testing.T) {
	dictionary := []byte("common bounded dictionary prefix and repeated phrase")
	source := []byte("repeated phrase repeated phrase")
	compressed := test_dictionary_stream()
	var decoded [TEST_OUTPUT_SIZE]byte
	count, status := flate.Decode_Dictionary_Into(decoded[:], compressed[:], dictionary)
	testify.Equal_Values(
		t, flate.STATUS_OK, status, "dictionary decode status = %d", status,
	)
	testify.Equal(
		t, string(source), string(decoded[:count]),
		"dictionary decode = (%q, %d)", decoded[:count], status,
	)

	var encoded [TEST_OUTPUT_SIZE]byte
	var workspace_storage test_workspace
	workspace := test_workspace_value(&workspace_storage)
	encoded_count, encode_status := flate.Encode_Dictionary_Into(
		encoded[:], workspace, source, dictionary, flate.BEST_COMPRESSION,
	)
	testify.Equal_Values(
		t, flate.STATUS_OK, encode_status,
		"dictionary encode status = %d", encode_status,
	)
	decoded_count, decoded_status := flate.Decode_Dictionary_Into(
		decoded[:], encoded[:encoded_count], dictionary,
	)
	testify.Equal_Values(
		t, flate.STATUS_OK, decoded_status,
		"dictionary round trip status = %d", decoded_status,
	)
	testify.Equal(
		t, string(source), string(decoded[:decoded_count]),
		"dictionary round trip = (%q, %d)",
		decoded[:decoded_count], decoded_status,
	)
}

// Test_Untrusted_Input protects exact stream consumption and hostile storage aliases.
func Test_Untrusted_Input(t *testing.T) {
	compressed := test_dynamic()
	var destination [TEST_OUTPUT_SIZE]byte
	for index := 0; index < len(compressed); index++ {
		_, status := flate.Decode_Into(destination[:], compressed[:index])
		testify.Equal_Values(
			t, flate.STATUS_INPUT_INVALID, status,
			"truncated size %d status = %d", index, status,
		)
	}
	with_suffix := [TEST_DYNAMIC_SIZE + 1]byte{}
	copy(with_suffix[:], compressed[:])
	_, status := flate.Decode_Into(destination[:], with_suffix[:])
	testify.Equal_Values(
		t, flate.STATUS_INPUT_INVALID, status, "trailing input status = %d", status,
	)

	storage := [TEST_OUTPUT_SIZE]byte{}
	copy(storage[:], compressed[:])
	_, status = flate.Decode_Into(storage[:], storage[:len(compressed)])
	testify.Equal_Values(
		t, flate.STATUS_STORAGE_INVALID, status, "overlap status = %d", status,
	)
	_, encode_status := flate.Encode_Into(
		destination[:], flate.Workspace_Unvalidated{}, nil, flate.BEST_SPEED,
	)
	testify.Equal_Values(
		t, flate.STATUS_STORAGE_INVALID, encode_status,
		"missing workspace status = %d", encode_status,
	)

	stored := test_stored()
	fixed := test_fixed()
	dynamic := test_dynamic()
	variants := [][]byte{stored[:], fixed[:], dynamic[:]}
	for _, original := range variants {
		candidate := make([]byte, len(original))
		for index := range original {
			for bit_index := uint8(0); bit_index < flate.BYTE_BIT_COUNT; bit_index++ {
				copy(candidate, original)
				candidate[index] ^= 1 << bit_index
				observed_count, observed_status := flate.Decode_Into(
					destination[:], candidate,
				)
				testify.Greater_Or_Equal(
					t,
					&testify.Greater_Or_Equal_Input[flate.Count]{
						First: observed_count, Second: 0,
					},
					"mutation count = %d", observed_count,
				)
				testify.Less_Or_Equal(
					t,
					&testify.Less_Or_Equal_Input[flate.Decode_Status]{
						First:  observed_status,
						Second: flate.STATUS_OUTPUT_TOO_SMALL,
					},
					"mutation status = %d", observed_status,
				)
			}
		}
	}
}

// Test_Invariant_Domains drives public bounds through real operation entry points.
func Test_Invariant_Domains(t *testing.T) {
	maximum_bytes := make([]byte, flate.BYTE_COUNT_MAXIMUM)
	fixed := test_fixed()
	for _, size := range []int{
		TEST_DOMAIN_MINIMUM, TEST_DOMAIN_SECOND, TEST_DOMAIN_THIRD,
	} {
		flate.Decode_Into(maximum_bytes[:size], fixed[:])
		flate.Decode_Prefix_Into(maximum_bytes[:size], fixed[:])
		flate.Decode_Prefix_Into(nil, maximum_bytes[:size])
		var workspace_storage test_workspace
		workspace := test_workspace_value(&workspace_storage)
		flate.Encode_Into(
			maximum_bytes[:size], workspace, maximum_bytes[:size],
			flate.HUFFMAN_ONLY,
		)
	}
	flate.Decode_Into(maximum_bytes, nil)
	flate.Decode_Into(nil, maximum_bytes)
	flate.Decode_Dictionary_Into(nil, fixed[:], maximum_bytes)
	flate.Encode_Into(
		maximum_bytes, flate.Workspace_Unvalidated{}, nil,
		flate.NO_COMPRESSION,
	)
	flate.Encode_Into(
		nil, flate.Workspace_Unvalidated{}, maximum_bytes,
		flate.NO_COMPRESSION,
	)
	for _, level := range []flate.Level_Unvalidated{
		flate.Level_Unvalidated(flate.LEVEL_UNVALIDATED_MINIMUM),
		flate.Level_Unvalidated(flate.LEVEL_UNVALIDATED_MAXIMUM),
	} {
		flate.Encode_Into(nil, flate.Workspace_Unvalidated{}, nil, level)
	}
	verify_workspace_domains()
}

// Test_Maximum_Decode proves the output bound on a stream that reaches it.
func Test_Maximum_Decode(t *testing.T) {
	destination := make([]byte, flate.BYTE_COUNT_MAXIMUM)
	compressed := test_fixed_zeros_then_stored(flate.BYTE_COUNT_MAXIMUM)
	count, status := flate.Decode_Into(destination, compressed)
	testify.Equal_Values(
		t, flate.STATUS_OK, status, "maximum decode status = %d", status,
	)
	testify.Equal(
		t, flate.BYTE_COUNT_MAXIMUM, int(count), "maximum decode count = %d", count,
	)
	maximum_compressed := test_maximum_stored(t)
	count, status = flate.Decode_Into(destination, maximum_compressed)
	testify.Equal_Values(
		t, flate.STATUS_OK, status, "maximum prefix input status = %d", status,
	)
	testify.Equal(
		t, TEST_MAXIMUM_STORED_PAYLOAD_SIZE, int(count),
		"maximum prefix input output count = %d", count,
	)
}

// Test_Decode_Block_Domains proves carried counts and short match destinations.
func Test_Decode_Block_Domains(t *testing.T) {
	verify_carried_decode_counts(t)
	verify_short_match_destinations(t)
	verify_stored_decode_domains(t)
}

// Carried counts must remain exact across every next-block kind.
func verify_carried_decode_counts(t *testing.T) {
	t.Helper()
	fixed := test_fixed()
	for _, size := range []int{TEST_DOMAIN_SECOND, TEST_DOMAIN_THIRD} {
		compressed := test_stored_prefix(make([]byte, size), fixed[:])
		var destination [TEST_OUTPUT_SIZE]byte
		count, status := flate.Decode_Into(destination[:], compressed)
		testify.Equal_Values(
			t, flate.STATUS_OK, status, "prefix size %d status = %d", size, status,
		)
		testify.Equal(
			t, size+len(TEST_FIXED_CONTENT), int(count),
			"prefix size %d = (%d, %d)", size, count, status,
		)

		repeat := test_dictionary_repeat()
		compressed = test_stored_prefix(make([]byte, size), repeat)
		count, status = flate.Decode_Into(destination[:], compressed)
		testify.Equal_Values(
			t, flate.STATUS_OK, status,
			"match prefix size %d status = %d", size, status,
		)
		testify.Equal(
			t, size+flate.MATCH_SIZE_MINIMUM, int(count),
			"match prefix size %d = (%d, %d)", size, count, status,
		)

		empty_stored := test_empty_stored()
		compressed = test_stored_prefix(make([]byte, size), empty_stored[:])
		count, status = flate.Decode_Into(destination[:], compressed)
		testify.Equal_Values(
			t, flate.STATUS_OK, status,
			"stored prefix size %d status = %d", size, status,
		)
		testify.Equal(
			t, size, int(count),
			"stored prefix size %d = (%d, %d)", size, count, status,
		)
	}
}

// Short destinations must stop inside a match without partial count lies.
func verify_short_match_destinations(t *testing.T) {
	t.Helper()
	repeat := test_dictionary_repeat()
	for _, size := range []int{
		TEST_DOMAIN_MINIMUM, TEST_DOMAIN_SECOND, TEST_DOMAIN_THIRD,
	} {
		destination := make([]byte, size)
		count, status := flate.Decode_Dictionary_Into(
			destination, repeat, []byte{'x'},
		)
		testify.Equal_Values(
			t, flate.STATUS_OUTPUT_TOO_SMALL, status,
			"match destination %d status = %d", size, status,
		)
		testify.Equal(
			t, size, int(count),
			"match destination %d = (%d, %d)", size, count, status,
		)
	}
	_, status := flate.Decode_Into(nil, repeat)
	testify.Equal_Values(
		t, flate.STATUS_INPUT_INVALID, status,
		"impossible distance status = %d", status,
	)
}

// Empty, truncated, and short stored blocks require distinct results.
func verify_stored_decode_domains(t *testing.T) {
	t.Helper()
	empty_stored := test_empty_stored()
	maximum_destination := make([]byte, flate.BYTE_COUNT_MAXIMUM)
	for _, size := range []int{
		TEST_DOMAIN_MINIMUM,
		TEST_DOMAIN_SECOND,
		TEST_DOMAIN_THIRD,
		flate.BYTE_COUNT_MAXIMUM,
	} {
		stored_count, stored_status := flate.Decode_Into(
			maximum_destination[:size], empty_stored[:],
		)
		testify.Equal_Values(
			t, flate.STATUS_OK, stored_status,
			"stored destination %d status = %d", size, stored_status,
		)
		testify.Zero(
			t, stored_count, "stored destination %d count = %d", size, stored_count,
		)
	}
	stored := test_stored()
	_, status := flate.Decode_Into(nil, stored[:])
	testify.Equal_Values(
		t, flate.STATUS_OUTPUT_TOO_SMALL, status, "stored short status = %d", status,
	)
	_, status = flate.Decode_Into(
		maximum_destination, empty_stored[:TEST_TRUNCATED_STORED_SIZE],
	)
	testify.Equal_Values(
		t, flate.STATUS_INPUT_INVALID, status, "stored invalid status = %d", status,
	)
}

// Test_Dynamic_Domains proves minimum and maximum hostile header declarations.
func Test_Dynamic_Domains(t *testing.T) {
	var destination [TEST_OUTPUT_SIZE]byte
	for _, test_case := range [...]struct {
		Literal_Bits  uint32
		Distance_Bits uint32
		Code_Count    int
		Tail          uint32
	}{
		{0, 0, flate.ENCODED_CODE_COUNT_MINIMUM, 0},
		{0, 0, flate.CODE_COUNT, 0},
		{
			flate.LITERAL_COUNT_MAXIMUM - flate.LITERAL_COUNT_MINIMUM,
			flate.DISTANCE_SYMBOL_COUNT - flate.DISTANCE_COUNT_MINIMUM,
			flate.FIXED_CODE_SIZE,
			1<<flate.REPEAT_ZERO_LONG_EXTRA_BIT_COUNT - 1,
		},
		{0, 0, flate.FIXED_THIRD_CODE_SIZE, flate.REPEAT_PREVIOUS_EXTRA_BIT_COUNT},
	} {
		compressed := test_dynamic_probe(
			bits.BIT_COUNT_16_MAXIMUM,
			test_case.Literal_Bits, test_case.Distance_Bits,
			test_case.Code_Count, test_case.Tail,
		)
		count, status := flate.Decode_Into(destination[:], compressed)
		testify.Greater_Or_Equal(
			t,
			&testify.Greater_Or_Equal_Input[flate.Count]{First: count, Second: 0},
			"dynamic probe count = %d", count,
		)
		testify.Less_Or_Equal(
			t,
			&testify.Less_Or_Equal_Input[flate.Decode_Status]{
				First: status, Second: flate.STATUS_OUTPUT_TOO_SMALL,
			},
			"dynamic probe = (%d, %d)", count, status,
		)
	}
}

// Test_History_Domains proves every retained-history length through one real match.
func Test_History_Domains(t *testing.T) {
	compressed := test_dictionary_repeat()
	dictionary := make([]byte, flate.DICTIONARY_SIZE_MAXIMUM)
	for _, size := range []int{
		TEST_DOMAIN_SECOND, TEST_DOMAIN_THIRD, flate.DICTIONARY_SIZE_MAXIMUM,
	} {
		dictionary[size-1] = byte(size)
		var destination [flate.MATCH_SIZE_MINIMUM]byte
		count, status := flate.Decode_Dictionary_Into(
			destination[:], compressed, dictionary[:size],
		)
		testify.Equal_Values(
			t, flate.STATUS_OK, status, "history size %d status = %d", size, status,
		)
		testify.Equal(
			t, len(destination), int(count), "history size %d count = %d", size, count,
		)
		for _, value := range destination {
			testify.Equal(
				t, byte(size), value,
				"history size %d output = %v", size, destination,
			)
		}
	}
}

// Test_Result_Domains proves operation-specific status and count domains.
func Test_Result_Domains(t *testing.T) {
	var workspace_storage test_workspace
	workspace := test_workspace_value(&workspace_storage)
	var short [TEST_COUNT_DOMAIN_SIZE]byte
	for _, size := range []int{TEST_DOMAIN_SECOND, TEST_DOMAIN_THIRD} {
		count, status := flate.Encode_Into(
			short[:size], workspace, nil, flate.NO_COMPRESSION,
		)
		testify.Equal_Values(
			t, flate.STATUS_OUTPUT_TOO_SMALL, status,
			"short encode %d status = %d", size, status,
		)
		testify.Equal(
			t, size, int(count), "short encode %d = (%d, %d)", size, count, status,
		)
		count, status = flate.Encode_Dictionary_Into(
			short[:size], workspace, nil, []byte{'x'}, flate.NO_COMPRESSION,
		)
		testify.Equal_Values(
			t, flate.STATUS_OUTPUT_TOO_SMALL, status,
			"short dictionary encode %d status = %d", size, status,
		)
		testify.Equal(
			t, size, int(count),
			"short dictionary encode %d = (%d, %d)", size, count, status,
		)
	}

	_, status := flate.Encode_Dictionary_Into(
		nil, workspace, nil, nil, flate.HUFFMAN_ONLY-1,
	)
	testify.Equal_Values(
		t, flate.STATUS_LEVEL_INVALID, status, "dictionary level status = %d", status,
	)
	_, status = flate.Encode_Dictionary_Into(
		nil, flate.Workspace_Unvalidated{}, nil, nil, flate.NO_COMPRESSION,
	)
	testify.Equal_Values(
		t, flate.STATUS_STORAGE_INVALID, status,
		"dictionary storage status = %d", status,
	)

	fixed := test_fixed()
	_, decode_status := flate.Decode_Dictionary_Into(nil, nil, nil)
	testify.Equal_Values(
		t, flate.STATUS_INPUT_INVALID, decode_status,
		"dictionary input status = %d", decode_status,
	)
	_, decode_status = flate.Decode_Dictionary_Into(nil, fixed[:], nil)
	testify.Equal_Values(
		t, flate.STATUS_OUTPUT_TOO_SMALL, decode_status,
		"dictionary output status = %d", decode_status,
	)
	aliased := [TEST_OUTPUT_SIZE]byte{}
	copy(aliased[:], fixed[:])
	_, decode_status = flate.Decode_Dictionary_Into(
		aliased[:], aliased[:len(fixed)], nil,
	)
	testify.Equal_Values(
		t, flate.STATUS_STORAGE_INVALID, decode_status,
		"dictionary storage status = %d", decode_status,
	)
}

// Test_Maximum_Encode proves count and stored-block bounds at one real limit.
func Test_Maximum_Encode(t *testing.T) {
	source := make([]byte, flate.BYTE_COUNT_MAXIMUM)
	destination := make([]byte, flate.BYTE_COUNT_MAXIMUM)
	var workspace_storage test_workspace
	workspace := test_workspace_value(&workspace_storage)
	count, status := flate.Encode_Into(
		destination, workspace, source, flate.NO_COMPRESSION,
	)
	testify.Equal_Values(
		t, flate.STATUS_OUTPUT_TOO_SMALL, status, "maximum encode status = %d", status,
	)
	testify.Equal(
		t, len(destination), int(count), "maximum encode count = %d", count,
	)
	count, status = flate.Encode_Dictionary_Into(
		destination, workspace, source, nil, flate.NO_COMPRESSION,
	)
	testify.Equal_Values(
		t, flate.STATUS_OUTPUT_TOO_SMALL, status,
		"maximum dictionary encode status = %d", status,
	)
	testify.Equal(
		t, len(destination), int(count),
		"maximum dictionary encode count = %d", count,
	)
}

// Test_Encode_Input_Domains drives large source and dictionary through real encoders.
func Test_Encode_Input_Domains(t *testing.T) {
	source := make([]byte, flate.BYTE_COUNT_MAXIMUM)
	var workspace_storage test_workspace
	workspace := test_workspace_value(&workspace_storage)
	for _, level := range []flate.Level_Unvalidated{
		flate.HUFFMAN_ONLY, flate.BEST_SPEED,
	} {
		_, status := flate.Encode_Dictionary_Into(
			nil, workspace, source, source, level,
		)
		testify.Equal_Values(
			t, flate.STATUS_OUTPUT_TOO_SMALL, status,
			"maximum input level %d status = %d", level, status,
		)
	}
	for _, size := range []int{
		TEST_DOMAIN_SECOND, TEST_DOMAIN_THIRD, flate.BYTE_COUNT_MAXIMUM,
	} {
		_, status := flate.Encode_Dictionary_Into(
			nil, workspace, nil, source[:size], flate.NO_COMPRESSION,
		)
		testify.Equal_Values(
			t, flate.STATUS_OUTPUT_TOO_SMALL, status,
			"dictionary size %d status = %d", size, status,
		)
	}
	for _, source_size := range []int{
		TEST_DOMAIN_MINIMUM, TEST_DOMAIN_SECOND, TEST_DOMAIN_THIRD,
	} {
		for _, level := range []flate.Level_Unvalidated{
			flate.NO_COMPRESSION, flate.HUFFMAN_ONLY, flate.BEST_SPEED,
		} {
			var destination [TEST_OUTPUT_SIZE]byte
			_, status := flate.Encode_Into(
				destination[:], workspace, source[:source_size], level,
			)
			testify.Equal_Values(
				t, flate.STATUS_OK, status,
				"source size %d level %d status = %d",
				source_size, level, status,
			)
		}
	}
	for _, dictionary := range [][]byte{
		{TEST_DOMAIN_THIRD},
		{TEST_DOMAIN_SECOND, TEST_DOMAIN_THIRD},
	} {
		var destination [TEST_OUTPUT_SIZE]byte
		_, status := flate.Encode_Dictionary_Into(
			destination[:], workspace,
			[]byte{TEST_DOMAIN_THIRD, TEST_DOMAIN_THIRD, TEST_DOMAIN_THIRD},
			dictionary,
			flate.BEST_SPEED,
		)
		testify.Equal_Values(
			t, flate.STATUS_OK, status,
			"history size %d status = %d", len(dictionary), status,
		)
	}
}

func verify_oversized_bounds(t *testing.T, workspace flate.Workspace_Unvalidated) {
	t.Helper()
	oversized := make([]byte, flate.BYTE_COUNT_MAXIMUM+1)
	_, decode_status := flate.Decode_Into(oversized, nil)
	testify.Equal_Values(
		t, flate.STATUS_STORAGE_INVALID, decode_status,
		"oversized destination status = %d", decode_status,
	)
	_, decode_status = flate.Decode_Into(nil, oversized)
	testify.Equal_Values(
		t, flate.STATUS_STORAGE_INVALID, decode_status,
		"oversized compressed status = %d", decode_status,
	)
	_, decode_status = flate.Decode_Dictionary_Into(nil, nil, oversized)
	testify.Equal_Values(
		t, flate.STATUS_STORAGE_INVALID, decode_status,
		"oversized decode dictionary status = %d", decode_status,
	)
	_, _, prefix_status := flate.Decode_Prefix_Into(oversized, nil)
	testify.Equal_Values(
		t, flate.STATUS_STORAGE_INVALID, prefix_status,
		"oversized prefix destination status = %d", prefix_status,
	)
	_, _, prefix_status = flate.Decode_Prefix_Into(nil, oversized)
	testify.Equal_Values(
		t, flate.STATUS_STORAGE_INVALID, prefix_status,
		"oversized prefix input status = %d", prefix_status,
	)
	_, encode_status := flate.Encode_Into(
		oversized, workspace, nil, flate.NO_COMPRESSION,
	)
	testify.Equal_Values(
		t, flate.STATUS_STORAGE_INVALID, encode_status,
		"oversized encode destination status = %d", encode_status,
	)
	_, encode_status = flate.Encode_Into(
		nil, workspace, oversized, flate.NO_COMPRESSION,
	)
	testify.Equal_Values(
		t, flate.STATUS_STORAGE_INVALID, encode_status,
		"oversized source status = %d", encode_status,
	)
	_, encode_status = flate.Encode_Dictionary_Into(
		nil, workspace, nil, oversized, flate.NO_COMPRESSION,
	)
	testify.Equal_Values(
		t, flate.STATUS_STORAGE_INVALID, encode_status,
		"oversized encode dictionary status = %d", encode_status,
	)
}

func verify_decode_allocation(t *testing.T) {
	t.Helper()
	compressed := test_dynamic()
	dictionary_compressed := test_dictionary_stream()
	dictionary := []byte("common bounded dictionary prefix and repeated phrase")
	var destination [TEST_OUTPUT_SIZE]byte
	aliased := [TEST_OUTPUT_SIZE]byte{}
	copy(aliased[:], compressed[:])
	var observed_count flate.Count
	var observed_decode_status flate.Decode_Status
	testify.Zero_Allocation(t, func() {
		observed_count, observed_decode_status = flate.Decode_Into(
			destination[:], compressed[:],
		)
	})
	testify.Zero_Allocation(t, func() {
		observed_count, observed_decode_status = flate.Decode_Into(
			destination[:TEST_SHORT_SIZE], compressed[:],
		)
	})
	testify.Zero_Allocation(t, func() {
		observed_count, observed_decode_status = flate.Decode_Into(
			destination[:], nil,
		)
	})
	testify.Zero_Allocation(t, func() {
		observed_count, observed_decode_status = flate.Decode_Into(
			aliased[:], aliased[:len(compressed)],
		)
	})
	testify.Zero_Allocation(t, func() {
		observed_count, observed_decode_status = flate.Decode_Dictionary_Into(
			destination[:], dictionary_compressed[:], dictionary,
		)
	})
	testify.Zero_Allocation(t, func() {
		observed_count, observed_decode_status = flate.Decode_Dictionary_Into(
			nil, dictionary_compressed[:], dictionary,
		)
	})
	testify.Greater_Or_Equal(
		t,
		&testify.Greater_Or_Equal_Input[flate.Count]{First: observed_count, Second: 0},
		"decode allocation probe returned impossible count",
	)
	testify.Less_Or_Equal(
		t,
		&testify.Less_Or_Equal_Input[flate.Decode_Status]{
			First: observed_decode_status, Second: flate.STATUS_STORAGE_INVALID,
		},
		"decode allocation probe returned impossible status",
	)
	verify_prefix_allocation(t)
}

func verify_prefix_allocation(t *testing.T) {
	t.Helper()
	compressed := test_dynamic()
	var destination [TEST_OUTPUT_SIZE]byte
	aliased := [TEST_OUTPUT_SIZE]byte{}
	copy(aliased[:], compressed[:])
	var observed_count flate.Count
	var observed_compressed_count flate.Count
	var observed_status flate.Decode_Status
	testify.Zero_Allocation(t, func() {
		observed_count, observed_compressed_count, observed_status =
			flate.Decode_Prefix_Into(destination[:], compressed[:])
	})
	testify.Zero_Allocation(t, func() {
		observed_count, observed_compressed_count, observed_status =
			flate.Decode_Prefix_Into(nil, compressed[:])
	})
	testify.Zero_Allocation(t, func() {
		observed_count, observed_compressed_count, observed_status =
			flate.Decode_Prefix_Into(destination[:], nil)
	})
	testify.Zero_Allocation(t, func() {
		observed_count, observed_compressed_count, observed_status =
			flate.Decode_Prefix_Into(
				aliased[:], aliased[:len(compressed)],
			)
	})
	testify.Greater_Or_Equal(
		t,
		&testify.Greater_Or_Equal_Input[flate.Count]{First: observed_count, Second: 0},
		"prefix allocation probe returned impossible count",
	)
	testify.Greater_Or_Equal(
		t,
		&testify.Greater_Or_Equal_Input[flate.Count]{
			First: observed_compressed_count, Second: 0,
		},
		"prefix allocation probe returned impossible input count",
	)
	testify.Less_Or_Equal(
		t,
		&testify.Less_Or_Equal_Input[flate.Decode_Status]{
			First: observed_status, Second: flate.STATUS_STORAGE_INVALID,
		},
		"prefix allocation probe returned impossible status",
	)
}

func verify_encode_allocation(t *testing.T) {
	t.Helper()
	var encoded [TEST_OUTPUT_SIZE]byte
	var workspace_storage test_workspace
	workspace := test_workspace_value(&workspace_storage)
	dictionary := []byte("common bounded dictionary prefix and repeated phrase")
	var observed_count flate.Count
	var observed_encode_status flate.Encode_Status
	for _, level := range []flate.Level{
		flate.NO_COMPRESSION,
		flate.BEST_SPEED,
		flate.DEFAULT_COMPRESSION,
		flate.BEST_COMPRESSION,
		flate.HUFFMAN_ONLY,
	} {
		testify.Zero_Allocation(t, func() {
			observed_count, observed_encode_status = flate.Encode_Into(
				encoded[:], workspace, []byte(TEST_CONTENT),
				flate.Level_Unvalidated(level),
			)
		})
	}
	testify.Zero_Allocation(t, func() {
		observed_count, observed_encode_status = flate.Encode_Into(
			nil, workspace, nil, flate.NO_COMPRESSION,
		)
	})
	testify.Zero_Allocation(t, func() {
		observed_count, observed_encode_status = flate.Encode_Into(
			nil, workspace, nil, flate.HUFFMAN_ONLY-1,
		)
	})
	testify.Zero_Allocation(t, func() {
		observed_count, observed_encode_status = flate.Encode_Into(
			encoded[:], flate.Workspace_Unvalidated{}, nil, flate.NO_COMPRESSION,
		)
	})
	testify.Zero_Allocation(t, func() {
		observed_count, observed_encode_status = flate.Encode_Dictionary_Into(
			encoded[:], workspace, []byte(TEST_CONTENT), dictionary,
			flate.BEST_COMPRESSION,
		)
	})
	testify.Zero_Allocation(t, func() {
		observed_count, observed_encode_status = flate.Encode_Dictionary_Into(
			nil, workspace, nil, dictionary, flate.NO_COMPRESSION,
		)
	})
	testify.Greater_Or_Equal(
		t,
		&testify.Greater_Or_Equal_Input[flate.Count]{First: observed_count, Second: 0},
		"encode allocation probe returned impossible count",
	)
	testify.Less_Or_Equal(
		t,
		&testify.Less_Or_Equal_Input[flate.Encode_Status]{
			First: observed_encode_status, Second: flate.STATUS_STORAGE_INVALID,
		},
		"encode allocation probe returned impossible status",
	)
}

func verify_workspace_domains() {
	for _, count := range []int{
		TEST_DOMAIN_MINIMUM, TEST_DOMAIN_SECOND, TEST_DOMAIN_THIRD,
	} {
		workspace := flate.Workspace_Unvalidated{
			Heads: make([]int32, count), Previous: make([]int32, count),
		}
		flate.Encode_Into(nil, workspace, nil, flate.NO_COMPRESSION)
	}
	workspace := flate.Workspace_Unvalidated{
		Heads:    make([]int32, flate.HASH_POSITIONS_COUNT_MAXIMUM),
		Previous: make([]int32, flate.HISTORY_POSITIONS_COUNT_MAXIMUM),
	}
	flate.Encode_Into(nil, workspace, nil, flate.NO_COMPRESSION)
}

const TEST_CONTENT = "bounded flate data bounded flate data bounded flate data"
const TEST_FIXED_CONTENT = "fixed huffman payload fixed huffman payload"
const TEST_DOMAIN_MINIMUM = 0
const TEST_DOMAIN_SECOND = TEST_DOMAIN_MINIMUM + 1
const TEST_DOMAIN_THIRD = TEST_DOMAIN_SECOND + 1
const TEST_OUTPUT_SIZE = 1 << bits.BIT_COUNT_8_MAXIMUM
const TEST_PREFIX_SUFFIX_SIZE = 2
const TEST_SHORT_SIZE = 5
const TEST_COUNT_DOMAIN_SIZE = 2
const TEST_STORED_SIZE = 25
const TEST_FIXED_SIZE = 27
const TEST_DYNAMIC_SIZE = 29
const TEST_LEVEL_COUNT = 5
const TEST_DICTIONARY_COMPRESSED_SIZE = 6
const TEST_DYNAMIC_CODE_SIZE_COUNT = 2
const TEST_NO_COMPRESSION_LEVEL = 0
const TEST_BEST_SPEED_LEVEL = TEST_NO_COMPRESSION_LEVEL + 1
const TEST_BEST_COMPRESSION_LEVEL = TEST_BEST_SPEED_LEVEL + bits.BIT_COUNT_8_MAXIMUM
const TEST_DEFAULT_COMPRESSION_LEVEL = TEST_NO_COMPRESSION_LEVEL - 1
const TEST_HUFFMAN_ONLY_LEVEL = TEST_DEFAULT_COMPRESSION_LEVEL - 1
const TEST_STORED_HEADER_BYTE_COUNT = 1
const TEST_STORED_SIZE_LOW_OFFSET = TEST_STORED_HEADER_BYTE_COUNT
const TEST_STORED_SIZE_HIGH_OFFSET = TEST_STORED_SIZE_LOW_OFFSET + 1
const TEST_STORED_INVERSE_LOW_OFFSET = TEST_STORED_SIZE_HIGH_OFFSET + 1
const TEST_STORED_INVERSE_HIGH_OFFSET = TEST_STORED_INVERSE_LOW_OFFSET + 1
const TEST_TRUNCATED_STORED_SIZE = TEST_STORED_HEADER_BYTE_COUNT + 1
const TEST_STORED_PAYLOAD_MAXIMUM = int(flate.STORED_SIZE_MAXIMUM)
const TEST_STORED_WIRE_OVERHEAD = 1 +
	2*(bits.BIT_COUNT_16_MAXIMUM/bits.BIT_COUNT_8_MAXIMUM)
const TEST_MAXIMUM_STORED_BLOCK_COUNT = (flate.BYTE_COUNT_MAXIMUM +
	TEST_STORED_PAYLOAD_MAXIMUM + TEST_STORED_WIRE_OVERHEAD - 1) /
	(TEST_STORED_PAYLOAD_MAXIMUM + TEST_STORED_WIRE_OVERHEAD)
const TEST_MAXIMUM_STORED_PAYLOAD_SIZE = flate.BYTE_COUNT_MAXIMUM -
	TEST_MAXIMUM_STORED_BLOCK_COUNT*TEST_STORED_WIRE_OVERHEAD

type test_workspace struct {
	Heads    [flate.HASH_COUNT]int32
	Previous [flate.WINDOW_SIZE]int32
}

func test_workspace_value(
	storage *test_workspace,
) (workspace flate.Workspace_Unvalidated) {
	return flate.Workspace_Unvalidated{
		Heads: storage.Heads[:], Previous: storage.Previous[:],
	}
}

func test_stored() (compressed [TEST_STORED_SIZE]byte) {
	return [TEST_STORED_SIZE]byte{
		1, 20, 0, 235, 255, 115, 116, 111, 114, 101, 100, 32, 98, 108, 111,
		99, 107, 32, 112, 97, 121, 108, 111, 97, 100,
	}
}

func test_empty_stored() (compressed [TEST_STORED_WIRE_OVERHEAD]byte) {
	size := uint16(bits.WORD_16_MINIMUM)
	inverse := ^size
	return [TEST_STORED_WIRE_OVERHEAD]byte{
		flate.BLOCK_KIND_STORED<<flate.FINAL_BIT_COUNT | flate.FINAL_BLOCK_BIT_VALUE,
		byte(size),
		byte(size >> bits.BIT_COUNT_8_MAXIMUM),
		byte(inverse),
		byte(inverse >> bits.BIT_COUNT_8_MAXIMUM),
	}
}

func test_fixed() (compressed [TEST_FIXED_SIZE]byte) {
	return [TEST_FIXED_SIZE]byte{
		75, 203, 172, 72, 77, 81, 200, 40, 77, 75, 203, 77, 204, 83, 40, 72,
		172, 204, 201, 79, 76, 81, 72, 195, 38, 10, 0,
	}
}

func test_dynamic() (compressed [TEST_DYNAMIC_SIZE]byte) {
	return [TEST_DYNAMIC_SIZE]byte{
		5, 193, 7, 1, 0, 0, 12, 2, 160, 172, 234, 60, 253, 19, 12, 0, 128,
		36, 37, 233, 238, 206, 182, 147, 164, 109, 183, 237, 1,
	}
}

func verify_decode(t *testing.T, compressed []byte, want string) {
	t.Helper()
	var destination [TEST_OUTPUT_SIZE]byte
	count, status := flate.Decode_Into(destination[:], compressed)
	testify.Equal_Values(
		t, flate.STATUS_OK, status, "Decode_Into status = %d", status,
	)
	testify.Equal(
		t, want, string(destination[:count]),
		"Decode_Into output = %q", destination[:count],
	)
}

func test_dictionary_stream() (compressed [TEST_DICTIONARY_COMPRESSED_SIZE]byte) {
	return [TEST_DICTIONARY_COMPRESSED_SIZE]byte{67, 227, 162, 75, 3, 0}
}

func test_dictionary_repeat() (compressed []byte) {
	bits := uint64(0)
	bit_count := uint(0)
	compressed, bits, bit_count = test_append_bits(
		compressed, bits, bit_count,
		uint32(flate.BLOCK_KIND_FIXED<<flate.FINAL_BIT_COUNT|
			flate.FINAL_BLOCK_BIT_VALUE),
		flate.BLOCK_HEADER_BIT_COUNT,
	)
	compressed, bits, bit_count = test_append_fixed_symbol(
		compressed, bits, bit_count, flate.MATCH_SYMBOL_MINIMUM,
	)
	compressed, bits, bit_count = test_append_bits(
		compressed, bits, bit_count, 0, flate.FIXED_DISTANCE_BIT_COUNT,
	)
	compressed, bits, bit_count = test_append_fixed_symbol(
		compressed, bits, bit_count, flate.LITERAL_SYMBOL_END,
	)
	if bit_count > 0 {
		compressed = append(compressed, byte(bits))
	}
	return compressed
}

func test_maximum_stored(t *testing.T) (compressed []byte) {
	t.Helper()
	compressed = make([]byte, flate.BYTE_COUNT_MAXIMUM)
	payload_remainder := TEST_MAXIMUM_STORED_PAYLOAD_SIZE
	position := 0
	for block_index := 0; block_index < TEST_MAXIMUM_STORED_BLOCK_COUNT; block_index++ {
		payload_size := payload_remainder
		if payload_size > TEST_STORED_PAYLOAD_MAXIMUM {
			payload_size = TEST_STORED_PAYLOAD_MAXIMUM
		}
		if block_index == TEST_MAXIMUM_STORED_BLOCK_COUNT-1 {
			compressed[position] = flate.FINAL_BLOCK_BIT_VALUE
		}
		size := uint16(payload_size)
		inverse := ^size
		compressed[position+TEST_STORED_SIZE_LOW_OFFSET] = byte(size)
		compressed[position+TEST_STORED_SIZE_HIGH_OFFSET] = byte(
			size >> bits.BIT_COUNT_8_MAXIMUM,
		)
		compressed[position+TEST_STORED_INVERSE_LOW_OFFSET] = byte(inverse)
		compressed[position+TEST_STORED_INVERSE_HIGH_OFFSET] = byte(
			inverse >> bits.BIT_COUNT_8_MAXIMUM,
		)
		position += TEST_STORED_WIRE_OVERHEAD + payload_size
		payload_remainder -= payload_size
	}
	testify.Equal(
		t, len(compressed), position, "maximum stored stream size mismatch",
	)
	return compressed
}

func test_fixed_zeros(count int) (compressed []byte) {
	bits := uint64(0)
	bit_count := uint(0)
	compressed, bits, bit_count = test_append_fixed_zeros_block(
		compressed, bits, bit_count, count, true,
	)
	if bit_count > 0 {
		compressed = append(compressed, byte(bits))
	}
	return compressed
}

func test_fixed_zeros_then_stored(count int) (compressed []byte) {
	bits := uint64(0)
	bit_count := uint(0)
	compressed, bits, bit_count = test_append_fixed_zeros_block(
		compressed, bits, bit_count, count, false,
	)
	compressed, bits, bit_count = test_append_bits(
		compressed, bits, bit_count,
		uint32(flate.BLOCK_KIND_STORED<<flate.FINAL_BIT_COUNT|
			flate.FINAL_BLOCK_BIT_VALUE),
		flate.BLOCK_HEADER_BIT_COUNT,
	)
	if bit_count > 0 {
		compressed = append(compressed, byte(bits))
	}
	return append(
		compressed,
		byte(flate.BIT_COUNT_MINIMUM),
		byte(flate.BIT_COUNT_MINIMUM),
		flate.BYTE_VALUE_MAXIMUM,
		flate.BYTE_VALUE_MAXIMUM,
	)
}

func test_append_fixed_zeros_block(
	compressed []byte, bits uint64, bit_count uint, count int, final bool,
) (next []byte, next_bits uint64, next_bit_count uint) {
	header := uint32(flate.BLOCK_KIND_FIXED << flate.FINAL_BIT_COUNT)
	if final {
		header |= flate.FINAL_BLOCK_BIT_VALUE
	}
	compressed, bits, bit_count = test_append_bits(
		compressed, bits, bit_count, header, flate.BLOCK_HEADER_BIT_COUNT,
	)
	tail_count := count
	if tail_count > 0 {
		compressed, bits, bit_count = test_append_fixed_symbol(
			compressed, bits, bit_count, 0,
		)
		tail_count--
	}
	for tail_count >= flate.MATCH_SIZE_MAXIMUM {
		compressed, bits, bit_count = test_append_fixed_symbol(
			compressed, bits, bit_count, flate.MATCH_SYMBOL_MAXIMUM,
		)
		compressed, bits, bit_count = test_append_bits(
			compressed, bits, bit_count, 0, flate.MATCH_EXTRA_BIT_COUNT_MAXIMUM,
		)
		tail_count -= flate.MATCH_SIZE_MAXIMUM
	}
	for tail_count > 0 {
		compressed, bits, bit_count = test_append_fixed_symbol(
			compressed, bits, bit_count, 0,
		)
		tail_count--
	}
	compressed, bits, bit_count = test_append_fixed_symbol(
		compressed, bits, bit_count, flate.LITERAL_SYMBOL_END,
	)
	return compressed, bits, bit_count
}

func test_stored_prefix(prefix []byte, suffix []byte) (compressed []byte) {
	size := uint16(len(prefix))
	inverse := ^size
	compressed = append(
		compressed,
		byte(flate.BLOCK_KIND_STORED),
		byte(size),
		byte(size>>bits.BIT_COUNT_8_MAXIMUM),
		byte(inverse),
		byte(inverse>>bits.BIT_COUNT_8_MAXIMUM),
	)
	compressed = append(compressed, prefix...)
	return append(compressed, suffix...)
}

func test_dynamic_probe(
	total_size int,
	literal_bits uint32,
	distance_bits uint32,
	code_count int,
	tail_value uint32,
) (compressed []byte) {
	bits := uint64(0)
	bit_count := uint(0)
	compressed, bits, bit_count = test_append_bits(
		compressed, bits, bit_count,
		uint32(flate.BLOCK_KIND_DYNAMIC<<flate.FINAL_BIT_COUNT|
			flate.FINAL_BLOCK_BIT_VALUE),
		flate.BLOCK_HEADER_BIT_COUNT,
	)
	compressed, bits, bit_count = test_append_bits(
		compressed, bits, bit_count, literal_bits,
		flate.DYNAMIC_ALPHABET_COUNT_BIT_COUNT,
	)
	compressed, bits, bit_count = test_append_bits(
		compressed, bits, bit_count, distance_bits,
		flate.DYNAMIC_ALPHABET_COUNT_BIT_COUNT,
	)
	compressed, bits, bit_count = test_append_bits(
		compressed, bits, bit_count,
		uint32(code_count-flate.ENCODED_CODE_COUNT_MINIMUM),
		flate.DYNAMIC_CODE_COUNT_BIT_COUNT,
	)
	for index := 0; index < code_count; index++ {
		size := uint32(0)
		if index < TEST_DYNAMIC_CODE_SIZE_COUNT {
			size = flate.FINAL_BLOCK_BIT_VALUE
		}
		compressed, bits, bit_count = test_append_bits(
			compressed, bits, bit_count, size, flate.CODE_SIZE_BIT_COUNT,
		)
	}
	if bit_count > 0 {
		bits |= uint64(tail_value) << bit_count
		compressed = append(compressed, byte(bits))
	}
	if total_size < len(compressed) {
		total_size = len(compressed)
	}
	result := make([]byte, total_size)
	copy(result, compressed)
	return result
}

func test_append_fixed_symbol(
	destination []byte, bits uint64, bit_count uint, symbol uint32,
) (next []byte, next_bits uint64, next_bit_count uint) {
	var code uint32
	var width uint
	switch {
	case symbol <= flate.FIXED_LITERAL_FIRST_MAXIMUM:
		code = flate.FIXED_FIRST_CODE_BASE + symbol
		width = flate.FIXED_CODE_SIZE
	case symbol <= flate.FIXED_LITERAL_SECOND_MAXIMUM:
		code = flate.FIXED_SECOND_CODE_BASE + symbol - flate.FIXED_LITERAL_SECOND_MINIMUM
		width = flate.FIXED_SECOND_CODE_SIZE
	case symbol <= flate.FIXED_LITERAL_THIRD_MAXIMUM:
		code = symbol - flate.FIXED_LITERAL_THIRD_MINIMUM
		width = flate.FIXED_THIRD_CODE_SIZE
	default:
		code = flate.FIXED_FOURTH_CODE_BASE + symbol - flate.FIXED_LITERAL_FOURTH_MINIMUM
		width = flate.FIXED_CODE_SIZE
	}
	return test_append_bits(
		destination, bits, bit_count, test_reverse_bits(code, width), width,
	)
}

func test_append_bits(
	destination []byte, bits uint64, bit_count uint, value uint32, count uint,
) (next []byte, next_bits uint64, next_bit_count uint) {
	bits |= uint64(value) << bit_count
	bit_count += count
	for bit_count >= flate.BYTE_BIT_COUNT {
		destination = append(destination, byte(bits))
		bits >>= flate.BYTE_BIT_COUNT
		bit_count -= flate.BYTE_BIT_COUNT
	}
	return destination, bits, bit_count
}

func test_reverse_bits(value uint32, count uint) (reversed uint32) {
	for index := uint(0); index < count; index++ {
		reversed = reversed<<flate.FINAL_BIT_COUNT | value&flate.LOW_BIT_MASK
		value >>= flate.FINAL_BIT_COUNT
	}
	return reversed
}
