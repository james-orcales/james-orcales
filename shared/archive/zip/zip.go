// Package zip decodes bounded classic ZIP entries into caller-owned storage.
package zip

import (
	"encoding/binary"
	"hash/crc32"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
)

// STATUS_MINIMUM anchors output coverage at successful decode.
const STATUS_MINIMUM = STATUS_OK

// STATUS_MAXIMUM closes output coverage at unsupported codec.
const STATUS_MAXIMUM = STATUS_METHOD_UNSUPPORTED

// HUFFMAN_BIT_COUNT_MAXIMUM rejects overlong canonical codes before table writes.
const HUFFMAN_BIT_COUNT_MAXIMUM = 15

// HUFFMAN_SYMBOL_COUNT_MAXIMUM fixes full literal table on stack.
const HUFFMAN_SYMBOL_COUNT_MAXIMUM = 288

// Status reports bounded decode result without allocating error state.
type Status uint8

// STATUS_OK keeps success distinct from every rejected input.
const STATUS_OK Status = 0

// STATUS_INPUT_INVALID prevents malformed metadata from reaching decompression.
const STATUS_INPUT_INVALID Status = 1

// STATUS_ENTRY_NOT_FOUND distinguishes valid archives from absent members.
const STATUS_ENTRY_NOT_FOUND Status = 2

// STATUS_OUTPUT_TOO_SMALL protects caller-owned output bounds.
const STATUS_OUTPUT_TOO_SMALL Status = 3

// STATUS_METHOD_UNSUPPORTED rejects codecs outside bounded implementation.
const STATUS_METHOD_UNSUPPORTED Status = 4

// Status_Invariants lists every decode outcome.
func Status_Invariants(value Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(STATUS_MINIMUM), uint8(STATUS_MAXIMUM)).
		Ensure()
}

// Central_Entry keeps validated directory metadata together during one decode.
type Central_Entry struct {
	// Flags must agree with local metadata before compressed bytes become trusted.
	Flags uint16
	// Method selects only bounded codecs implemented here.
	Method uint16
	// Checksum catches both corrupt metadata and corrupt decoded bytes.
	Checksum uint32
	// Compressed_Size bounds source slicing before codec entry.
	Compressed_Size uint32
	// Uncompressed_Size must fit destination before any write.
	Uncompressed_Size uint32
	// Local_Offset keeps directory pointers inside classic ZIP address space.
	Local_Offset uint32
	// Name_Start retains caller storage instead of copying member names.
	Name_Start int
	// Name_End closes retained member-name view.
	Name_End int
}

// Bit_Reader keeps one bounded DEFLATE cursor.
type Bit_Reader struct {
	// Source stays caller-owned throughout one decode.
	Source []byte
	// Position prevents DEFLATE reads crossing compressed member boundary.
	Position int
	// Value holds only bounded unread bits from Source.
	Value uint64
	// Count distinguishes valid zero bits from empty bit state.
	Count uint8
}

// Huffman holds bounded canonical code metadata on stack.
type Huffman struct {
	// Counts bound every canonical code size without dynamic tables.
	Counts [HUFFMAN_BIT_COUNT_MAXIMUM + 1]uint16
	// Symbols retain canonical order in fixed stack storage.
	Symbols [HUFFMAN_SYMBOL_COUNT_MAXIMUM]uint16
}

// Decode_Into finds one member by base-name suffix and writes decoded bytes into destination.
func Decode_Into(
	source bytes.Slice, entry_suffix bytes.Text, destination bytes.Slice,
) (count bytes.Boundary, status Status) {
	defer func() { Status_Invariants(status, "decode_into.status") }()
	if !valid_input(source, entry_suffix, destination) {
		return 0, STATUS_INPUT_INVALID
	}
	end_position, entry_count, central_size, central_offset, found :=
		directory_end(source)
	if !found {
		return 0, STATUS_INPUT_INVALID
	}
	if uint64(central_size) > uint64(end_position) {
		return 0, STATUS_INPUT_INVALID
	}
	central_start := end_position - int(central_size)
	if uint64(central_offset) > uint64(central_start) {
		return 0, STATUS_INPUT_INVALID
	}
	base_offset := central_start - int(central_offset)
	position := central_start
	for entry_index := uint16(0); entry_index < entry_count; entry_index++ {
		entry, next, valid := read_central_entry(source, position, end_position)
		if !valid {
			return 0, STATUS_INPUT_INVALID
		}
		position = next
		name := source[entry.Name_Start:entry.Name_End]
		if name_has_suffix(name, entry_suffix) {
			return decode_entry(
				source, destination, base_offset, central_start, entry,
			)
		}
	}
	if position != end_position {
		return 0, STATUS_INPUT_INVALID
	}
	return 0, STATUS_ENTRY_NOT_FOUND
}

func valid_input(
	source bytes.Slice, entry_suffix bytes.Text, destination bytes.Slice,
) (valid bool) {
	const DIRECTORY_END_SIZE = 22
	if len(source) < DIRECTORY_END_SIZE {
		return false
	}
	if len(source) > bytes.SLICE_SIZE_MAXIMUM {
		return false
	}
	if len(entry_suffix) == 0 {
		return false
	}
	if len(entry_suffix) > bytes.TEXT_SIZE_MAXIMUM {
		return false
	}
	for position := range entry_suffix {
		if entry_suffix[position] == 0 {
			return false
		}
		if entry_suffix[position] == '/' {
			return false
		}
	}
	return len(destination) <= bytes.SLICE_SIZE_MAXIMUM
}

func directory_end(
	source []byte,
) (
	position int,
	entry_count uint16,
	central_size uint32,
	central_offset uint32,
	found bool,
) {
	const DIRECTORY_END_SIGNATURE uint32 = 0x06054b50
	const DIRECTORY_END_SIZE = 22
	const DIRECTORY_END_DISK_POSITION = 4
	const DIRECTORY_END_CENTRAL_DISK_POSITION = 6
	const DIRECTORY_END_DISK_ENTRY_COUNT_POSITION = 8
	const DIRECTORY_END_ENTRY_COUNT_POSITION = 10
	const DIRECTORY_END_CENTRAL_SIZE_POSITION = 12
	const DIRECTORY_END_CENTRAL_OFFSET_POSITION = 16
	const DIRECTORY_END_COMMENT_SIZE_POSITION = 20
	const DIRECTORY_END_SEARCH_SIZE_MAXIMUM = DIRECTORY_END_SIZE + int(bits.WORD_16_MAXIMUM)
	minimum := len(source) - DIRECTORY_END_SEARCH_SIZE_MAXIMUM
	if minimum < 0 {
		minimum = 0
	}
	for candidate := len(source) - DIRECTORY_END_SIZE; candidate >= minimum; candidate-- {
		if binary.LittleEndian.Uint32(source[candidate:]) != DIRECTORY_END_SIGNATURE {
			continue
		}
		comment_size := int(binary.LittleEndian.Uint16(
			source[candidate+DIRECTORY_END_COMMENT_SIZE_POSITION:],
		))
		if candidate+DIRECTORY_END_SIZE+comment_size != len(source) {
			continue
		}
		if binary.LittleEndian.Uint16(
			source[candidate+DIRECTORY_END_DISK_POSITION:],
		) != 0 {
			return 0, 0, 0, 0, false
		}
		if binary.LittleEndian.Uint16(
			source[candidate+DIRECTORY_END_CENTRAL_DISK_POSITION:],
		) != 0 {
			return 0, 0, 0, 0, false
		}
		entry_count = binary.LittleEndian.Uint16(
			source[candidate+DIRECTORY_END_ENTRY_COUNT_POSITION:],
		)
		if binary.LittleEndian.Uint16(
			source[candidate+DIRECTORY_END_DISK_ENTRY_COUNT_POSITION:],
		) != entry_count {
			return 0, 0, 0, 0, false
		}
		return candidate, entry_count,
			binary.LittleEndian.Uint32(
				source[candidate+DIRECTORY_END_CENTRAL_SIZE_POSITION:],
			),
			binary.LittleEndian.Uint32(
				source[candidate+DIRECTORY_END_CENTRAL_OFFSET_POSITION:],
			), true
	}
	return 0, 0, 0, 0, false
}

func read_central_entry(
	source []byte, position int, central_end int,
) (entry Central_Entry, next int, valid bool) {
	const CENTRAL_HEADER_SIGNATURE uint32 = 0x02014b50
	const CENTRAL_HEADER_SIZE = 46
	const CENTRAL_FLAGS_POSITION = 8
	const CENTRAL_METHOD_POSITION = 10
	const CENTRAL_CHECKSUM_POSITION = 16
	const CENTRAL_COMPRESSED_SIZE_POSITION = 20
	const CENTRAL_UNCOMPRESSED_SIZE_POSITION = 24
	const CENTRAL_NAME_SIZE_POSITION = 28
	const CENTRAL_EXTRA_SIZE_POSITION = 30
	const CENTRAL_COMMENT_SIZE_POSITION = 32
	const CENTRAL_DISK_POSITION = 34
	const CENTRAL_LOCAL_OFFSET_POSITION = 42
	if !available(position, CENTRAL_HEADER_SIZE, central_end) {
		return Central_Entry{}, 0, false
	}
	header := source[position : position+CENTRAL_HEADER_SIZE]
	if binary.LittleEndian.Uint32(header) != CENTRAL_HEADER_SIGNATURE {
		return Central_Entry{}, 0, false
	}
	if binary.LittleEndian.Uint16(header[CENTRAL_DISK_POSITION:]) != 0 {
		return Central_Entry{}, 0, false
	}
	name_size := int(binary.LittleEndian.Uint16(
		header[CENTRAL_NAME_SIZE_POSITION:],
	))
	if name_size > bytes.SLICE_SIZE_MAXIMUM {
		return Central_Entry{}, 0, false
	}
	extra_size := int(binary.LittleEndian.Uint16(
		header[CENTRAL_EXTRA_SIZE_POSITION:],
	))
	comment_size := int(binary.LittleEndian.Uint16(
		header[CENTRAL_COMMENT_SIZE_POSITION:],
	))
	entry_size := CENTRAL_HEADER_SIZE + name_size + extra_size + comment_size
	if !available(position, entry_size, central_end) {
		return Central_Entry{}, 0, false
	}
	entry = Central_Entry{
		Flags: binary.LittleEndian.Uint16(
			header[CENTRAL_FLAGS_POSITION:],
		),
		Method: binary.LittleEndian.Uint16(
			header[CENTRAL_METHOD_POSITION:],
		),
		Checksum: binary.LittleEndian.Uint32(
			header[CENTRAL_CHECKSUM_POSITION:],
		),
		Compressed_Size: binary.LittleEndian.Uint32(
			header[CENTRAL_COMPRESSED_SIZE_POSITION:],
		),
		Uncompressed_Size: binary.LittleEndian.Uint32(
			header[CENTRAL_UNCOMPRESSED_SIZE_POSITION:],
		),
		Local_Offset: binary.LittleEndian.Uint32(
			header[CENTRAL_LOCAL_OFFSET_POSITION:],
		),
		Name_Start: position + CENTRAL_HEADER_SIZE,
		Name_End:   position + CENTRAL_HEADER_SIZE + name_size,
	}
	return entry, position + entry_size, true
}

func name_has_suffix(name []byte, suffix bytes.Text) (matches bool) {
	base_start := 0
	for position := range name {
		if name[position] == '/' {
			base_start = position + 1
		}
	}
	base_size := len(name) - base_start
	if len(suffix) > base_size {
		return false
	}
	suffix_start := len(name) - len(suffix)
	for position := range suffix {
		if name[suffix_start+position] != suffix[position] {
			return false
		}
	}
	return true
}

func decode_entry(
	source []byte,
	destination []byte,
	base_offset int,
	central_start int,
	entry Central_Entry,
) (count bytes.Boundary, status Status) {
	const FLAG_UTF8 uint16 = 1 << 11
	const FLAG_DEFLATE_OPTIONS uint16 = 3 << 1
	const FLAG_DATA_DESCRIPTOR uint16 = 1 << 3
	const FLAGS_COMMON = FLAG_UTF8 | FLAG_DATA_DESCRIPTOR
	const FLAGS_DEFLATE = FLAGS_COMMON | FLAG_DEFLATE_OPTIONS
	const METHOD_STORE uint16 = 0
	const METHOD_DEFLATE uint16 = 8
	if entry.Method != METHOD_STORE {
		if entry.Method != METHOD_DEFLATE {
			return 0, STATUS_METHOD_UNSUPPORTED
		}
	}
	allowed_flags := FLAGS_COMMON
	if entry.Method == METHOD_DEFLATE {
		allowed_flags = FLAGS_DEFLATE
	}
	if entry.Flags&^allowed_flags != 0 {
		return 0, STATUS_INPUT_INVALID
	}
	if uint64(entry.Uncompressed_Size) > uint64(len(destination)) {
		return 0, STATUS_OUTPUT_TOO_SMALL
	}
	data_position, valid := local_data_position(
		source, base_offset, central_start, entry,
	)
	if !valid {
		return 0, STATUS_INPUT_INVALID
	}
	if uint64(entry.Compressed_Size) > uint64(central_start-data_position) {
		return 0, STATUS_INPUT_INVALID
	}
	compressed_end := data_position + int(entry.Compressed_Size)
	compressed := source[data_position:compressed_end]
	output := destination[:int(entry.Uncompressed_Size)]
	switch entry.Method {
	case METHOD_STORE:
		if entry.Compressed_Size != entry.Uncompressed_Size {
			return 0, STATUS_INPUT_INVALID
		}
		copy(output, compressed)
	case METHOD_DEFLATE:
		if !deflate_into(compressed, output) {
			return 0, STATUS_INPUT_INVALID
		}
	}
	if crc32.ChecksumIEEE(output) != entry.Checksum {
		return 0, STATUS_INPUT_INVALID
	}
	return bytes.Boundary(entry.Uncompressed_Size), STATUS_OK
}

func local_data_position(
	source []byte,
	base_offset int,
	central_start int,
	entry Central_Entry,
) (data_position int, valid bool) {
	const LOCAL_HEADER_SIGNATURE uint32 = 0x04034b50
	const LOCAL_HEADER_SIZE = 30
	const LOCAL_FLAGS_POSITION = 6
	const LOCAL_METHOD_POSITION = 8
	const LOCAL_CHECKSUM_POSITION = 14
	const LOCAL_COMPRESSED_SIZE_POSITION = 18
	const LOCAL_UNCOMPRESSED_SIZE_POSITION = 22
	const LOCAL_NAME_SIZE_POSITION = 26
	const LOCAL_EXTRA_SIZE_POSITION = 28
	const FLAG_DATA_DESCRIPTOR uint16 = 1 << 3
	local_position_64 := uint64(base_offset) + uint64(entry.Local_Offset)
	if local_position_64 > uint64(central_start) {
		return 0, false
	}
	local_position := int(local_position_64)
	if !available(local_position, LOCAL_HEADER_SIZE, central_start) {
		return 0, false
	}
	header := source[local_position : local_position+LOCAL_HEADER_SIZE]
	if binary.LittleEndian.Uint32(header) != LOCAL_HEADER_SIGNATURE {
		return 0, false
	}
	local_flags := binary.LittleEndian.Uint16(header[LOCAL_FLAGS_POSITION:])
	if local_flags != entry.Flags {
		return 0, false
	}
	if binary.LittleEndian.Uint16(header[LOCAL_METHOD_POSITION:]) != entry.Method {
		return 0, false
	}
	name_size := int(binary.LittleEndian.Uint16(header[LOCAL_NAME_SIZE_POSITION:]))
	if name_size > bytes.SLICE_SIZE_MAXIMUM {
		return 0, false
	}
	extra_size := int(binary.LittleEndian.Uint16(header[LOCAL_EXTRA_SIZE_POSITION:]))
	data_position = local_position + LOCAL_HEADER_SIZE + name_size + extra_size
	if !available(local_position, LOCAL_HEADER_SIZE+name_size+extra_size, central_start) {
		return 0, false
	}
	local_name_start := local_position + LOCAL_HEADER_SIZE
	local_name := source[local_name_start : local_name_start+name_size]
	central_name := source[entry.Name_Start:entry.Name_End]
	if !bytes.Equal(bytes.Slice(local_name), bytes.Slice(central_name)) {
		return 0, false
	}
	if entry.Flags&FLAG_DATA_DESCRIPTOR == 0 {
		if binary.LittleEndian.Uint32(header[LOCAL_CHECKSUM_POSITION:]) !=
			entry.Checksum {
			return 0, false
		}
		if binary.LittleEndian.Uint32(header[LOCAL_COMPRESSED_SIZE_POSITION:]) !=
			entry.Compressed_Size {
			return 0, false
		}
		if binary.LittleEndian.Uint32(header[LOCAL_UNCOMPRESSED_SIZE_POSITION:]) !=
			entry.Uncompressed_Size {
			return 0, false
		}
	}
	return data_position, true
}

func available(position int, size int, end int) (yes bool) {
	if position < 0 {
		return false
	}
	if size < 0 {
		return false
	}
	if end < 0 {
		return false
	}
	if position > end {
		return false
	}
	return size <= end-position
}

func deflate_into(source []byte, destination []byte) (valid bool) {
	reader := Bit_Reader{Source: source}
	output_position := 0
	final := uint32(0)
	for final == 0 {
		var read bool
		final, read = read_bits(&reader, 1)
		if !read {
			return false
		}
		block_type, read := read_bits(&reader, 2)
		if !read {
			return false
		}
		switch block_type {
		case 0:
			if !stored_block(&reader, destination, &output_position) {
				return false
			}
		case 1:
			var literal Huffman
			var distance Huffman
			if !fixed_tables(&literal, &distance) {
				return false
			}
			if !compressed_block(
				&reader, destination, &output_position, &literal, &distance,
			) {
				return false
			}
		case 2:
			var literal Huffman
			var distance Huffman
			if !dynamic_tables(&reader, &literal, &distance) {
				return false
			}
			if !compressed_block(
				&reader, destination, &output_position, &literal, &distance,
			) {
				return false
			}
		default:
			return false
		}
	}
	return output_position == len(destination) && reader.Position == len(source)
}

func read_bits(reader *Bit_Reader, bit_count uint8) (value uint32, valid bool) {
	for reader.Count < bit_count {
		if reader.Position >= len(reader.Source) {
			return 0, false
		}
		reader.Value |= uint64(reader.Source[reader.Position]) << reader.Count
		reader.Position++
		reader.Count += uint8(bits.BIT_COUNT_8_MAXIMUM)
	}
	mask := uint64(1)<<bit_count - 1
	value = uint32(reader.Value & mask)
	reader.Value >>= bit_count
	reader.Count -= bit_count
	return value, true
}

func align_byte(reader *Bit_Reader) {
	reader.Value = 0
	reader.Count = 0
}

func stored_block(
	reader *Bit_Reader, destination []byte, output_position *int,
) (valid bool) {
	align_byte(reader)
	if !available(reader.Position, 4, len(reader.Source)) {
		return false
	}
	size := int(binary.LittleEndian.Uint16(reader.Source[reader.Position:]))
	complement := binary.LittleEndian.Uint16(reader.Source[reader.Position+2:])
	if uint16(size)^complement != bits.WORD_16_MAXIMUM {
		return false
	}
	reader.Position += 4
	if !available(reader.Position, size, len(reader.Source)) {
		return false
	}
	if !available(*output_position, size, len(destination)) {
		return false
	}
	copy(
		destination[*output_position:*output_position+size],
		reader.Source[reader.Position:reader.Position+size],
	)
	reader.Position += size
	*output_position += size
	return true
}

func fixed_tables(literal *Huffman, distance *Huffman) (valid bool) {
	const DISTANCE_SYMBOL_COUNT_MAXIMUM = 32
	var literal_sizes [HUFFMAN_SYMBOL_COUNT_MAXIMUM]uint8
	for symbol_index := 0; symbol_index <= 143; symbol_index++ {
		literal_sizes[symbol_index] = 8
	}
	for symbol_index := 144; symbol_index <= 255; symbol_index++ {
		literal_sizes[symbol_index] = 9
	}
	for symbol_index := 256; symbol_index <= 279; symbol_index++ {
		literal_sizes[symbol_index] = 7
	}
	for symbol_index := 280; symbol_index < len(literal_sizes); symbol_index++ {
		literal_sizes[symbol_index] = 8
	}
	var distance_sizes [DISTANCE_SYMBOL_COUNT_MAXIMUM]uint8
	for symbol_index := range distance_sizes {
		distance_sizes[symbol_index] = 5
	}
	if !build_huffman(literal, literal_sizes[:]) {
		return false
	}
	return build_huffman(distance, distance_sizes[:])
}

func dynamic_tables(
	reader *Bit_Reader, literal *Huffman, distance *Huffman,
) (valid bool) {
	const DYNAMIC_SYMBOL_COUNT_MAXIMUM = 318
	literal_count, distance_count, code_size_count, read := dynamic_header(reader)
	if !read {
		return false
	}
	var code_size_table Huffman
	if !dynamic_code_table(reader, code_size_count, &code_size_table) {
		return false
	}
	var sizes [DYNAMIC_SYMBOL_COUNT_MAXIMUM]uint8
	total_count := literal_count + distance_count
	position := 0
	for position < total_count {
		symbol, decoded := decode_huffman(reader, &code_size_table)
		if !decoded {
			return false
		}
		switch {
		case symbol <= 15:
			sizes[position] = uint8(symbol)
			position++
		case symbol == 16:
			if position == 0 {
				return false
			}
			repeat_delta, one_read := read_bits(reader, 2)
			if !one_read {
				return false
			}
			repeat := int(repeat_delta) + 3
			if position+repeat > total_count {
				return false
			}
			previous := sizes[position-1]
			for index := 0; index < repeat; index++ {
				sizes[position] = previous
				position++
			}
		case symbol == 17:
			repeat_delta, one_read := read_bits(reader, 3)
			if !one_read {
				return false
			}
			repeat := int(repeat_delta) + 3
			if position+repeat > total_count {
				return false
			}
			position += repeat
		case symbol == 18:
			repeat_delta, one_read := read_bits(reader, 7)
			if !one_read {
				return false
			}
			repeat := int(repeat_delta) + 11
			if position+repeat > total_count {
				return false
			}
			position += repeat
		default:
			return false
		}
	}
	return build_dynamic_tables(
		literal, distance, sizes[:], literal_count, distance_count,
	)
}

func build_dynamic_tables(
	literal *Huffman,
	distance *Huffman,
	sizes []uint8,
	literal_count int,
	distance_count int,
) (valid bool) {
	const END_BLOCK_SYMBOL = 256
	if sizes[END_BLOCK_SYMBOL] == 0 {
		return false
	}
	if !build_huffman(literal, sizes[:literal_count]) {
		return false
	}
	return build_huffman(
		distance, sizes[literal_count:literal_count+distance_count],
	)
}

func dynamic_header(
	reader *Bit_Reader,
) (literal_count int, distance_count int, code_size_count int, valid bool) {
	const LITERAL_SYMBOL_COUNT_MINIMUM = 257
	const DISTANCE_SYMBOL_COUNT_MINIMUM = 1
	literal_delta, read := read_bits(reader, 5)
	if !read {
		return 0, 0, 0, false
	}
	distance_delta, read := read_bits(reader, 5)
	if !read {
		return 0, 0, 0, false
	}
	code_size_delta, read := read_bits(reader, 4)
	if !read {
		return 0, 0, 0, false
	}
	return int(literal_delta) + LITERAL_SYMBOL_COUNT_MINIMUM,
		int(distance_delta) + DISTANCE_SYMBOL_COUNT_MINIMUM,
		int(code_size_delta) + 4, true
}

func dynamic_code_table(
	reader *Bit_Reader, code_size_count int, table *Huffman,
) (valid bool) {
	const CODE_SIZE_SYMBOL_COUNT = 19
	var code_sizes [CODE_SIZE_SYMBOL_COUNT]uint8
	code_size_order := [...]uint8{
		16, 17, 18, 0, 8, 7, 9, 6, 10, 5, 11, 4, 12, 3, 13, 2, 14, 1, 15,
	}
	for index := 0; index < code_size_count; index++ {
		code_size, read := read_bits(reader, 3)
		if !read {
			return false
		}
		code_sizes[code_size_order[index]] = uint8(code_size)
	}
	return build_huffman(table, code_sizes[:])
}

func build_huffman(table *Huffman, sizes []uint8) (valid bool) {
	for index := range table.Counts {
		table.Counts[index] = 0
	}
	for _, size := range sizes {
		if size > HUFFMAN_BIT_COUNT_MAXIMUM {
			return false
		}
		table.Counts[size]++
	}
	if table.Counts[0] == uint16(len(sizes)) {
		return false
	}
	left := 1
	for size_index := 1; size_index <= HUFFMAN_BIT_COUNT_MAXIMUM; size_index++ {
		left <<= 1
		left -= int(table.Counts[size_index])
		if left < 0 {
			return false
		}
	}
	var offsets [HUFFMAN_BIT_COUNT_MAXIMUM + 1]uint16
	for size_index := 1; size_index < HUFFMAN_BIT_COUNT_MAXIMUM; size_index++ {
		offsets[size_index+1] = offsets[size_index] + table.Counts[size_index]
	}
	for symbol_index, size := range sizes {
		if size == 0 {
			continue
		}
		position := offsets[size]
		if int(position) >= len(table.Symbols) {
			return false
		}
		table.Symbols[position] = uint16(symbol_index)
		offsets[size]++
	}
	return true
}

func decode_huffman(
	reader *Bit_Reader, table *Huffman,
) (symbol uint16, valid bool) {
	code := uint32(0)
	first := uint32(0)
	index := 0
	for size := 1; size <= HUFFMAN_BIT_COUNT_MAXIMUM; size++ {
		bit, read := read_bits(reader, 1)
		if !read {
			return 0, false
		}
		code |= bit
		count := uint32(table.Counts[size])
		if code >= first {
			if code < first+count {
				position := index + int(code-first)
				if position >= len(table.Symbols) {
					return 0, false
				}
				return table.Symbols[position], true
			}
		}
		index += int(count)
		first = (first + count) << 1
		code <<= 1
	}
	return 0, false
}

func compressed_block(
	reader *Bit_Reader,
	destination []byte,
	output_position *int,
	literal *Huffman,
	distance *Huffman,
) (valid bool) {
	const END_BLOCK_SYMBOL = 256
	const SPAN_SYMBOL_MINIMUM = 257
	const SPAN_SYMBOL_MAXIMUM = 285
	const DISTANCE_SYMBOL_MAXIMUM = 29
	for symbol_count := 0; symbol_count <= len(destination); symbol_count++ {
		symbol, decoded := decode_huffman(reader, literal)
		if !decoded {
			return false
		}
		if symbol < END_BLOCK_SYMBOL {
			if *output_position >= len(destination) {
				return false
			}
			destination[*output_position] = byte(symbol)
			*output_position++
			continue
		}
		if symbol == END_BLOCK_SYMBOL {
			return true
		}
		if symbol < SPAN_SYMBOL_MINIMUM {
			return false
		}
		if symbol > SPAN_SYMBOL_MAXIMUM {
			return false
		}
		span_index := int(symbol - SPAN_SYMBOL_MINIMUM)
		span_size, span_bit_count := span_properties(span_index)
		extra_span, read := read_bits(reader, span_bit_count)
		if !read {
			return false
		}
		span_size += int(extra_span)
		distance_symbol, decoded := decode_huffman(reader, distance)
		if !decoded {
			return false
		}
		if distance_symbol > DISTANCE_SYMBOL_MAXIMUM {
			return false
		}
		distance_index := int(distance_symbol)
		copy_distance, distance_bit_count := distance_properties(distance_index)
		extra_distance, read := read_bits(reader, distance_bit_count)
		if !read {
			return false
		}
		copy_distance += int(extra_distance)
		if copy_distance > *output_position {
			return false
		}
		if !available(*output_position, span_size, len(destination)) {
			return false
		}
		for index := 0; index < span_size; index++ {
			destination[*output_position] =
				destination[*output_position-copy_distance]
			*output_position++
		}
	}
	return false
}

func span_properties(index int) (base int, bit_count uint8) {
	bases := [...]uint16{
		3, 4, 5, 6, 7, 8, 9, 10,
		11, 13, 15, 17, 19, 23, 27, 31,
		35, 43, 51, 59, 67, 83, 99, 115,
		131, 163, 195, 227, 258,
	}
	bit_counts := [...]uint8{
		0, 0, 0, 0, 0, 0, 0, 0,
		1, 1, 1, 1, 2, 2, 2, 2,
		3, 3, 3, 3, 4, 4, 4, 4,
		5, 5, 5, 5, 0,
	}
	return int(bases[index]), bit_counts[index]
}

func distance_properties(index int) (base int, bit_count uint8) {
	bases := [...]uint16{
		1, 2, 3, 4, 5, 7, 9, 13,
		17, 25, 33, 49, 65, 97, 129, 193,
		257, 385, 513, 769, 1025, 1537, 2049, 3073,
		4097, 6145, 8193, 12289, 16385, 24577,
	}
	bit_counts := [...]uint8{
		0, 0, 0, 0, 1, 1, 2, 2,
		3, 3, 4, 4, 5, 5, 6, 6,
		7, 7, 8, 8, 9, 9, 10, 10,
		11, 11, 12, 12, 13, 13,
	}
	return int(bases[index]), bit_counts[index]
}
