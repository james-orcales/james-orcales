package bzip2

import "testing"

// Test_Bit_And_Table_Boundaries protects bit and canonical-table edges.
func Test_Bit_And_Table_Boundaries(t *testing.T) {
	reader := Bit_Reader{Bits: BIT_BUFFER_MAXIMUM, Bits_Count: BIT_COUNT_MAXIMUM}
	value, available := bit_reader_read(Bit_Reader_Handle(&reader), 2)
	if !available {
		t.Fatal("two-bit read is unavailable")
	}
	if value != 3 {
		t.Fatalf("two-bit read = (%d, %t)", value, available)
	}
	reader = Bit_Reader{Source: Bit_Source{255, 255, 255, 255, 255, 255}}
	value, available = bit_reader_read(Bit_Reader_Handle(&reader), BIT_READ_COUNT_MAXIMUM)
	if !available {
		t.Fatal("maximum bit read is unavailable")
	}
	if value != BIT_VALUE_MAXIMUM {
		t.Fatalf("maximum bit read = (%d, %t)", value, available)
	}

	decoder := test_huffman_decoder()
	sizes := make(Code_Sizes, SYMBOL_COUNT_MAXIMUM)
	if huffman_build(decoder, sizes) {
		t.Fatal("zero-width maximum alphabet is invalid")
	}
	decoder.Counts[1] = 1
	decoder.Symbols[0] = POSITION_MAXIMUM + 2
	symbol_reader := Payload_Reader{Bits_Count: 1}
	symbol, present := huffman_read(Payload_Reader_Handle(&symbol_reader), decoder)
	if !present {
		t.Fatal("maximum symbol is absent")
	}
	if symbol != POSITION_MAXIMUM+2 {
		t.Fatalf("maximum symbol read = (%d, %t)", symbol, present)
	}
	state := test_block_state()
	for index := TREE_INDEX_MINIMUM; index <= TREE_INDEX_MAXIMUM; index++ {
		decoder = huffman_decoder(state.Trees, Tree_Index(index))
		if len(decoder.Counts) != HUFFMAN_COUNT_SIZE {
			t.Fatalf("decoder %d count storage = %d", index, len(decoder.Counts))
		}
	}
}

// Test_Symbol_And_Tree_Boundaries protects bitmap and tree-header domains.
func Test_Symbol_And_Tree_Boundaries(t *testing.T) {
	for _, bits := range []Bit_Buffer{0, 1, 2, BIT_BUFFER_MAXIMUM} {
		reader := Symbol_Reader{Bits: bits, Bits_Count: SYMBOL_BIT_COUNT}
		state := test_block_state()
		_, valid := block_symbols(Symbol_Reader_Handle(&reader), state)
		if valid {
			t.Fatalf("truncated symbol bitmap with bits %d is valid", bits)
		}
	}
	reader := Symbol_Reader{
		Source: Bit_Source(test_bytes(34, 255)),
		Bits:   BIT_BUFFER_MAXIMUM, Bits_Count: SYMBOL_BIT_COUNT,
	}
	state := test_block_state()
	count, valid := block_symbols(Symbol_Reader_Handle(&reader), state)
	if !valid {
		t.Fatal("full symbol bitmap is invalid")
	}
	if count != BYTE_VALUE_COUNT {
		t.Fatalf("full symbol bitmap = (%d, %t)", count, valid)
	}

	for _, bits := range []Bit_Buffer{0, 1, 2, BIT_BUFFER_MAXIMUM} {
		tree_reader := Tree_Reader{Bits: bits, Bits_Count: SYMBOL_BIT_COUNT}
		_, _, _, tree_valid := block_trees(Tree_Reader_Handle(&tree_reader), state, 1)
		if tree_valid {
			t.Fatalf("truncated tree header with bits %d is valid", bits)
		}
	}
	tree_reader := Tree_Reader{
		Source: Bit_Source{255, 255}, Bits: 111, Bits_Count: SYMBOL_BIT_COUNT,
	}
	tree_count, selector_count, selectors, tree_valid := block_trees(
		Tree_Reader_Handle(&tree_reader), state, 1,
	)
	if tree_valid {
		t.Fatal("truncated maximum tree header is valid")
	}
	if tree_count != HUFFMAN_TREE_COUNT_MAXIMUM {
		t.Fatalf("maximum tree count = %d", tree_count)
	}
	if selector_count != SELECTOR_COUNT_MAXIMUM {
		t.Fatalf("maximum selector count = %d", selector_count)
	}
	if selectors.Bits != 31 {
		t.Fatalf("maximum tree header = (%d, %d, %d, %t)",
			tree_count, selector_count, selectors.Bits, tree_valid)
	}
	tree_reader = Tree_Reader{Bits_Count: SYMBOL_BIT_COUNT}
	block_trees(Tree_Reader_Handle(&tree_reader), state, BYTE_VALUE_COUNT)
}

// Test_Selector_Boundaries protects unary selector and move-to-front edges.
func Test_Selector_Boundaries(t *testing.T) {
	order := Selector_Order{
		First: 0, Second: 1, Third: 2, Fourth: 3, Fifth: 4, Sixth: 5,
	}
	reader := Selector_Reader{Bits: 62, Bits_Count: 6}
	tree, present := selector_read(
		Selector_Reader_Handle(&reader), Selector_Order_Handle(&order),
		HUFFMAN_TREE_COUNT_MAXIMUM,
	)
	if !present {
		t.Fatal("maximum selector is absent")
	}
	if tree != TREE_INDEX_MAXIMUM {
		t.Fatalf("maximum selector = (%d, %t)", tree, present)
	}
	selector_read_cases := []Selector_Reader{
		{},
		{Bits: 1, Bits_Count: 1},
		{Bits: 2, Bits_Count: 2},
		{Bits: BIT_BUFFER_MAXIMUM, Bits_Count: BIT_COUNT_MAXIMUM},
	}
	for index := range selector_read_cases {
		selector_read(
			Selector_Reader_Handle(&selector_read_cases[index]),
			Selector_Order_Handle(&order), TREE_COUNT_MINIMUM,
		)
	}
	order = Selector_Order{First: 0, Second: 1, Third: 2, Fourth: 3, Fifth: 4, Sixth: 5}
	reader = Selector_Reader{Bits: 6, Bits_Count: 3}
	tree, present = selector_read(
		Selector_Reader_Handle(&reader), Selector_Order_Handle(&order),
		HUFFMAN_TREE_COUNT_MAXIMUM,
	)
	if !present {
		t.Fatal("two-position selector is absent")
	}
	if tree != 2 {
		t.Fatalf("two-position selector = %d", tree)
	}
	test_tree_selection_bounds(t)
	test_move_to_front_edges(t)
}

func test_tree_selection_bounds(t *testing.T) {
	t.Helper()
	order := Selector_Order{
		First: 0, Second: 1, Third: 2, Fourth: 3, Fifth: 4, Sixth: 5,
	}
	selection := Tree_Selection{
		Selector_Index: SELECTOR_COUNT_MAXIMUM,
		Decoded_Count:  GROUP_COUNT_MINIMUM,
		Current_Tree:   TREE_INDEX_MAXIMUM,
	}
	for size := 0; size <= 2; size++ {
		candidate := Selector_Reader{
			Source: Bit_Source(make([]byte, size)),
			Bits:   Bit_Buffer(size), Bits_Count: Bit_Count(size),
		}
		if !block_tree_select(
			Selector_Reader_Handle(&candidate), Selector_Order_Handle(&order),
			HUFFMAN_TREE_COUNT_MAXIMUM, SELECTOR_COUNT_MAXIMUM,
			Tree_Selection_Handle(&selection),
		) {
			t.Fatalf("unused selector state %d is invalid", size)
		}
	}
	selection.Current_Tree = 2
	maximum_reader := Selector_Reader{
		Bits: BIT_BUFFER_MAXIMUM, Bits_Count: BIT_COUNT_MAXIMUM,
	}
	if !block_tree_select(
		Selector_Reader_Handle(&maximum_reader), Selector_Order_Handle(&order),
		HUFFMAN_TREE_COUNT_MAXIMUM, SELECTOR_COUNT_MAXIMUM,
		Tree_Selection_Handle(&selection),
	) {
		t.Fatal("unused maximum selector reader is invalid")
	}
	selection.Decoded_Count = SELECTOR_GROUP_SIZE
	reader := Selector_Reader{}
	if block_tree_select(
		Selector_Reader_Handle(&reader), Selector_Order_Handle(&order),
		HUFFMAN_TREE_COUNT_MAXIMUM, SELECTOR_COUNT_MAXIMUM,
		Tree_Selection_Handle(&selection),
	) {
		t.Fatal("exhausted maximum selector list is valid")
	}
	selectors := Selector_List_Reader{Bits: 31, Bits_Count: SELECTOR_BIT_COUNT}
	if selectors_skip(
		Selector_List_Reader_Handle(&selectors),
		HUFFMAN_TREE_COUNT_MAXIMUM, SELECTOR_COUNT_MAXIMUM,
	) {
		t.Fatal("truncated maximum selector list is valid")
	}
	test_selector_order_members(t)
}

func test_selector_order_members(t *testing.T) {
	t.Helper()
	for member := TREE_INDEX_MINIMUM; member <= TREE_INDEX_MAXIMUM; member++ {
		bound := Tree_Index(member)
		bounded_order := Selector_Order{
			First: First_Selector_Tree(bound), Second: Second_Selector_Tree(bound),
			Third: Third_Selector_Tree(bound), Fourth: Fourth_Selector_Tree(bound),
			Fifth: Fifth_Selector_Tree(bound), Sixth: Sixth_Selector_Tree(bound),
		}
		bounded_reader := Selector_Reader{Bits_Count: 1}
		bounded_selection := Tree_Selection{Decoded_Count: SELECTOR_GROUP_SIZE}
		if !block_tree_select(
			Selector_Reader_Handle(&bounded_reader),
			Selector_Order_Handle(&bounded_order), HUFFMAN_TREE_COUNT_MAXIMUM,
			SELECTOR_COUNT_MINIMUM, Tree_Selection_Handle(&bounded_selection),
		) {
			t.Fatalf("selector order bound %d is invalid", bound)
		}
	}
}

func test_move_to_front_edges(t *testing.T) {
	t.Helper()
	minimum_order := Move_Order{0}
	if move_to_front_decode(minimum_order, 0) != 0 {
		t.Fatal("minimum move-to-front order changed value")
	}
	maximum_order := make(Move_Order, BYTE_VALUE_COUNT)
	for index := range maximum_order {
		maximum_order[index] = byte(index)
	}
	if move_to_front_decode(maximum_order, POSITION_MAXIMUM) != BYTE_VALUE_MAXIMUM {
		t.Fatal("maximum move-to-front position changed value")
	}
}

// Test_Block_Boundaries protects decoded block and emission edges.
func Test_Block_Boundaries(t *testing.T) {
	encoded := test_repeat_block(TRANSFORM_COUNT_MAXIMUM, FIRST_POSITION_MAXIMUM)
	storage := make(Block_Storage, TRANSFORM_COUNT_MAXIMUM)
	count, first, _, valid := decode_block(
		Block_Compressed(encoded), storage, TRANSFORM_COUNT_MAXIMUM,
	)
	if !valid {
		t.Fatal("maximum block is invalid")
	}
	if count != TRANSFORM_COUNT_MAXIMUM {
		t.Fatalf("maximum block count = %d", count)
	}
	if first != FIRST_POSITION_MAXIMUM {
		t.Fatalf("maximum block = (%d, %d, %t)", count, first, valid)
	}
	small_encoded := test_repeat_block(3, 2)
	_, small_first, _, small_valid := decode_block(
		Block_Compressed(small_encoded), storage, TRANSFORM_COUNT_MAXIMUM,
	)
	if !small_valid {
		t.Fatal("two-position block is invalid")
	}
	if small_first != 2 {
		t.Fatalf("two-position block first = %d", small_first)
	}
	test_block_remainder_edges(t, storage)

	state := test_block_state()
	if !huffman_build(huffman_decoder(state.Trees, 0), Code_Sizes{1, 2, 2}) {
		t.Fatal("three-symbol payload table is invalid")
	}
	payload := Payload_Reader{Source: Bit_Source(test_repeat_payload(TRANSFORM_COUNT_MAXIMUM))}
	selectors := Selector_List_Reader{Bits_Count: SELECTOR_BIT_COUNT}
	count, valid = block_payload(
		Payload_Reader_Handle(&payload), Selector_List_Reader_Handle(&selectors),
		storage, TRANSFORM_COUNT_MAXIMUM, state,
		1, TREE_COUNT_MINIMUM, SELECTOR_COUNT_MINIMUM,
	)
	if !valid {
		t.Fatal("maximum payload is invalid")
	}
	if count != TRANSFORM_COUNT_MAXIMUM {
		t.Fatalf("maximum payload = (%d, %t)", count, valid)
	}
	test_block_payload_edges(t, storage)
	test_block_repeat_edges(t, storage)
	test_transform_edges(t)
	test_emit_edges(t)
}

func test_block_payload_edges(t *testing.T, storage Block_Storage) {
	t.Helper()
	state := test_block_state()
	readers := []Payload_Reader{
		{},
		{Bits: 2, Bits_Count: 2},
		{Bits: BIT_BUFFER_MAXIMUM, Bits_Count: BIT_COUNT_MAXIMUM},
	}
	selectors := []Selector_List_Reader{
		{Bits: 31, Bits_Count: SELECTOR_BIT_COUNT},
		{Source: Bit_Source{0}, Bits_Count: SELECTOR_BIT_COUNT},
		{Source: Bit_Source{0, 0}, Bits_Count: SELECTOR_BIT_COUNT},
	}
	for index := range readers {
		_, valid := block_payload(
			Payload_Reader_Handle(&readers[index]),
			Selector_List_Reader_Handle(&selectors[index]),
			storage, BLOCK_SIZE_UNIT, state,
			BYTE_VALUE_COUNT, HUFFMAN_TREE_COUNT_MAXIMUM, SELECTOR_COUNT_MAXIMUM,
		)
		if valid {
			t.Fatalf("invalid payload edge %d is valid", index)
		}
	}

	decoder_readers := []Decoder_Reader{
		{},
		{Bits: 2, Bits_Count: 2},
		{Bits: BIT_BUFFER_MAXIMUM, Bits_Count: BIT_COUNT_MAXIMUM},
	}
	for index := range decoder_readers {
		if block_tree_decoders(
			Decoder_Reader_Handle(&decoder_readers[index]), state,
			HUFFMAN_TREE_COUNT_MAXIMUM, SYMBOL_COUNT_MAXIMUM,
		) {
			t.Fatalf("invalid decoder edge %d is valid", index)
		}
	}
}

func test_block_remainder_edges(t *testing.T, storage Block_Storage) {
	t.Helper()
	maximum_found := false
	one_found := false
	for count := 1; count <= BYTE_VALUE_COUNT; count++ {
		encoded := test_repeat_block_ones(count, 0)
		_, _, remainder, valid := decode_block(
			Block_Compressed(encoded), storage, TRANSFORM_COUNT_MAXIMUM,
		)
		if !valid {
			t.Fatalf("padding probe %d is invalid", count)
		}
		if remainder.Bits == BIT_BUFFER_MAXIMUM {
			maximum_found = true
		}
		if remainder.Bits == 1 {
			one_found = true
		}
	}
	if !maximum_found {
		t.Fatal("no decoded block left the maximum bit remainder")
	}
	if !one_found {
		t.Fatal("no decoded block left a one-bit remainder")
	}

	trailer_readers := []Trailer_Reader{
		{Bits: 2, Bits_Count: 2},
		{Bits: BIT_BUFFER_MAXIMUM, Bits_Count: BIT_COUNT_MAXIMUM},
	}
	for index := range trailer_readers {
		if decode_trailer(Trailer_Reader_Handle(&trailer_readers[index]), 0) {
			t.Fatalf("truncated trailer edge %d is valid", index)
		}
	}
}

func test_block_repeat_edges(t *testing.T, storage Block_Storage) {
	t.Helper()
	var counts [BYTE_VALUE_COUNT]uint32
	next, valid := block_repeat(
		storage, TRANSFORM_COUNT_MAXIMUM, TRANSFORM_COUNT_MAXIMUM,
		REPEAT_COUNT_MINIMUM, 0, counts[:],
	)
	if !valid {
		t.Fatal("maximum repeat position is invalid")
	}
	if next != TRANSFORM_COUNT_MAXIMUM {
		t.Fatalf("maximum repeat position = (%d, %t)", next, valid)
	}
	for _, value := range []Byte_Value{1, 2, BYTE_VALUE_MAXIMUM} {
		next, valid = block_repeat(storage, 0, TRANSFORM_COUNT_MAXIMUM, 1, value, counts[:])
		if !valid {
			t.Fatalf("byte %d repeat is invalid", value)
		}
		if next != 1 {
			t.Fatalf("byte %d repeat = (%d, %t)", value, next, valid)
		}
	}
	next, valid = block_repeat(
		storage, 0, TRANSFORM_COUNT_MAXIMUM, REPEAT_COUNT_MAXIMUM, 0, counts[:],
	)
	if valid {
		t.Fatal("oversized repeat is valid")
	}
	if next != 0 {
		t.Fatalf("oversized repeat = (%d, %t)", next, valid)
	}
}

func test_transform_edges(t *testing.T) {
	t.Helper()
	maximum := make(Block, TRANSFORM_COUNT_MAXIMUM)
	var maximum_counts [BYTE_VALUE_COUNT]uint32
	maximum_counts[0] = TRANSFORM_COUNT_MAXIMUM
	first := inverse_transform(maximum, FIRST_POSITION_MAXIMUM, maximum_counts[:])
	if first != FIRST_POSITION_MAXIMUM {
		t.Fatalf("maximum inverse position = %d", first)
	}
	destination := make(Destination, TRANSFORM_COUNT_MAXIMUM)
	next, _, status := emit_block(destination, 0, maximum, first)
	if status != EMIT_STATUS_OK {
		t.Fatalf("maximum transform emission status = %d", status)
	}
	if next != 720_000 {
		t.Fatalf("maximum transform emission count = %d", next)
	}
	small := make(Block, 3)
	var small_counts [BYTE_VALUE_COUNT]uint32
	small_counts[0] = 3
	if inverse_transform(small, 2, small_counts[:]) != 2 {
		t.Fatal("two-position inverse transform changed position")
	}
}

func test_emit_edges(t *testing.T) {
	t.Helper()
	maximum_destination := make(Destination, BYTE_COUNT_MAXIMUM)
	next, _, status := emit_block(
		maximum_destination, BYTE_COUNT_MAXIMUM, Block{0}, 0,
	)
	if status != EMIT_STATUS_OUTPUT_TOO_SMALL {
		t.Fatalf("maximum emission status = %d", status)
	}
	if next != BYTE_COUNT_MAXIMUM {
		t.Fatalf("maximum emission count = (%d, %d)", next, status)
	}
	for _, count := range []Count{1, 2} {
		next, _, status = emit_block(maximum_destination, count, Block{0}, 0)
		if status != EMIT_STATUS_OK {
			t.Fatalf("emission count %d status = %d", count, status)
		}
		if next != count+1 {
			t.Fatalf("emission count %d = (%d, %d)", count, next, status)
		}
	}
	invalid := Block{2 << 8, 0}
	_, _, status = emit_block(maximum_destination, 0, invalid, 0)
	if status != EMIT_STATUS_INPUT_INVALID {
		t.Fatalf("invalid transform status = %d", status)
	}
	_, _, status = emit_block(maximum_destination, 0, Block{0, 0, 0}, 2)
	if status != EMIT_STATUS_OK {
		t.Fatalf("two-position emission status = %d", status)
	}
}

type test_bit_writer struct {
	// Bytes stays separate so partial bits never enter decoder input.
	Bytes []byte
	// Value keeps partial bits out of completed octets.
	Value byte
	// Count prevents final padding from becoming encoded data.
	Count uint
}

func test_bit_write(writer *test_bit_writer, value uint64, count uint) {
	for bit_count := count; bit_count > 0; bit_count-- {
		writer.Value = writer.Value<<1 | byte(value>>(bit_count-1)&1)
		writer.Count++
		if writer.Count == 8 {
			writer.Bytes = append(writer.Bytes, writer.Value)
			writer.Value = 0
			writer.Count = 0
		}
	}
}

func test_bit_finish(writer *test_bit_writer) (bytes []byte) {
	if writer.Count > 0 {
		writer.Bytes = append(writer.Bytes, writer.Value<<(8-writer.Count))
	}
	return writer.Bytes
}

func test_bit_finish_ones(writer *test_bit_writer) (bytes []byte) {
	if writer.Count > 0 {
		padding := byte(1<<(8-writer.Count)) - 1
		writer.Bytes = append(writer.Bytes, writer.Value<<(8-writer.Count)|padding)
	}
	return writer.Bytes
}

func test_repeat_block(count int, position uint32) (compressed []byte) {
	var writer test_bit_writer
	test_write_repeat_block(&writer, count, position)
	return test_bit_finish(&writer)
}

func test_repeat_block_ones(count int, position uint32) (compressed []byte) {
	var writer test_bit_writer
	test_write_repeat_block(&writer, count, position)
	return test_bit_finish_ones(&writer)
}

func test_write_repeat_block(writer *test_bit_writer, count int, position uint32) {
	test_bit_write(writer, 0, 1)
	test_bit_write(writer, uint64(position), 24)
	test_bit_write(writer, 1<<15, 16)
	test_bit_write(writer, 1<<15, 16)
	test_bit_write(writer, TREE_COUNT_MINIMUM, 3)
	test_bit_write(writer, SELECTOR_COUNT_MINIMUM, 15)
	test_bit_write(writer, 0, 1)
	for tree_index := 0; tree_index < TREE_COUNT_MINIMUM; tree_index++ {
		test_write_three_symbol_tree(writer)
	}
	test_write_repeat_payload(writer, count)
}

func test_repeat_payload(count int) (compressed []byte) {
	var writer test_bit_writer
	test_write_repeat_payload(&writer, count)
	return test_bit_finish(&writer)
}

func test_write_three_symbol_tree(writer *test_bit_writer) {
	test_bit_write(writer, 1, 5)
	test_bit_write(writer, 0, 1)
	test_bit_write(writer, 1, 1)
	test_bit_write(writer, 0, 1)
	test_bit_write(writer, 0, 1)
	test_bit_write(writer, 0, 1)
}

func test_write_repeat_payload(writer *test_bit_writer, count int) {
	magnitude := 1
	for magnitude<<1 <= count+1 {
		magnitude <<= 1
	}
	digits := count + 1 - magnitude
	for power := 1; power < magnitude; power <<= 1 {
		if digits&power == 0 {
			test_bit_write(writer, 0, 1)
		} else {
			test_bit_write(writer, 2, 2)
		}
	}
	test_bit_write(writer, 3, 2)
}

func test_bytes(size int, value byte) (bytes []byte) {
	bytes = make([]byte, size)
	for index := range bytes {
		bytes[index] = value
	}
	return bytes
}

func test_huffman_decoder() (decoder Huffman_Decoder) {
	return Huffman_Decoder{
		Counts:  make(Huffman_Counts, HUFFMAN_COUNT_SIZE),
		Symbols: make(Huffman_Symbols, SYMBOL_COUNT_MAXIMUM),
	}
}

func test_block_state() (state Block_State) {
	return Block_State{
		Character_Count: make(Character_Counts, BYTE_VALUE_COUNT),
		Trees:           make(Huffman_Storage, HUFFMAN_STORAGE_SIZE),
		Symbols:         make(Block_Symbols, BYTE_VALUE_COUNT),
		Code_Sizes:      make(Block_Code_Sizes, SYMBOL_COUNT_MAXIMUM),
	}
}
