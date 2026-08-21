// Package zlib decodes zlib streams into fixed caller storage.
package zlib

import "local/james-orcales/shared/sim/aver/default"

// DEFLATE_METHOD is zlib compression-method identifier.
const DEFLATE_METHOD = 8

// WINDOW_SIZE_MAXIMUM bounds legal DEFLATE backward distance.
const WINDOW_SIZE_MAXIMUM = 32_768

// CODE_SIZE_MAXIMUM bounds DEFLATE Huffman code bits.
const CODE_SIZE_MAXIMUM = 15

// LITERAL_COUNT_MAXIMUM bounds literal and match alphabet.
const LITERAL_COUNT_MAXIMUM = 286

// DISTANCE_COUNT_MAXIMUM excludes two reserved distance symbols.
const DISTANCE_COUNT_MAXIMUM = 30

// FIXED_DISTANCE_COUNT includes reserved symbols encoded by fixed tree.
const FIXED_DISTANCE_COUNT = 32

// CODE_COUNT bounds code-size alphabet.
const CODE_COUNT = 19

// HUFFMAN_SYMBOL_COUNT_MAXIMUM fits largest fixed alphabet.
const HUFFMAN_SYMBOL_COUNT_MAXIMUM = 288

// ADLER_MODULUS defines zlib checksum arithmetic.
const ADLER_MODULUS = 65_521

// BYTE_COUNT_MAXIMUM matches the caller's per-stream resource boundary.
const BYTE_COUNT_MAXIMUM = 64 * 1024 * 1024

// BYTE_COUNT_MINIMUM is an empty input, destination, or result.
const BYTE_COUNT_MINIMUM = 0

// COMPRESSED_PAYLOAD_SIZE_MAXIMUM excludes the two-byte zlib header.
const COMPRESSED_PAYLOAD_SIZE_MAXIMUM = BYTE_COUNT_MAXIMUM - 2

// BIT_COUNT_MAXIMUM is the unread remainder after one bounded read.
const BIT_COUNT_MAXIMUM = 7

// BIT_BUFFER_MAXIMUM is the largest unread remainder after one bounded read.
const BIT_BUFFER_MAXIMUM = 1<<BIT_COUNT_MAXIMUM - 1

// BIT_VALUE_MAXIMUM is the largest value one bounded read returns.
const BIT_VALUE_MAXIMUM = 1<<16 - 1

// CODE_SIZE_POSITION_MINIMUM is the first encoded code-size position.
const CODE_SIZE_POSITION_MINIMUM = 0

// CODE_SIZE_POSITION_MAXIMUM is the final encoded code-size position.
const CODE_SIZE_POSITION_MAXIMUM = CODE_COUNT - 1

// HUFFMAN_SYMBOL_MINIMUM is the first alphabet member.
const HUFFMAN_SYMBOL_MINIMUM = 0

// HUFFMAN_SYMBOL_MAXIMUM is the final member of the largest alphabet.
const HUFFMAN_SYMBOL_MAXIMUM = HUFFMAN_SYMBOL_COUNT_MAXIMUM - 1

// MATCH_SIZE_MINIMUM is the shortest DEFLATE match.
const MATCH_SIZE_MINIMUM = 3

// MATCH_SIZE_MAXIMUM is the longest DEFLATE match.
const MATCH_SIZE_MAXIMUM = 258

// MATCH_SYMBOL_MINIMUM is the first length symbol.
const MATCH_SYMBOL_MINIMUM = 257

// MATCH_SYMBOL_MAXIMUM is the final length symbol.
const MATCH_SYMBOL_MAXIMUM = 285

// DISTANCE_SYMBOL_MINIMUM is the nearest distance symbol.
const DISTANCE_SYMBOL_MINIMUM = 0

// DISTANCE_SYMBOL_MAXIMUM is the final legal distance symbol.
const DISTANCE_SYMBOL_MAXIMUM = DISTANCE_COUNT_MAXIMUM - 1

// DISTANCE_MINIMUM is the nearest prior byte.
const DISTANCE_MINIMUM = 1

// DISTANCE_BASE_MAXIMUM is the largest distance before its encoded suffix.
const DISTANCE_BASE_MAXIMUM = 24_577

// EXTRA_BIT_COUNT_MINIMUM is a symbol with no suffix.
const EXTRA_BIT_COUNT_MINIMUM = 0

// EXTRA_BIT_COUNT_MAXIMUM is the widest DEFLATE suffix.
const EXTRA_BIT_COUNT_MAXIMUM = 13

// MATCH_EXTRA_BIT_COUNT_MAXIMUM is the widest match suffix.
const MATCH_EXTRA_BIT_COUNT_MAXIMUM = 5

// BIT_READ_COUNT_MINIMUM is a symbol with no encoded suffix.
const BIT_READ_COUNT_MINIMUM = 0

// BIT_READ_COUNT_MAXIMUM is the widest reader request.
const BIT_READ_COUNT_MAXIMUM = 16

// BIT_VALUE_MINIMUM is an all-zero reader result.
const BIT_VALUE_MINIMUM = 0

// CODE_SIZES_MINIMUM admits a one-symbol alphabet.
const CODE_SIZES_MINIMUM = 1

// CODE_SIZES_MAXIMUM is the largest fixed alphabet.
const CODE_SIZES_MAXIMUM = HUFFMAN_SYMBOL_COUNT_MAXIMUM

// DYNAMIC_SIZES_MINIMUM is the smallest combined dynamic alphabet.
const DYNAMIC_SIZES_MINIMUM = 257 + 1

// DYNAMIC_SIZES_MAXIMUM is the largest combined dynamic alphabet.
const DYNAMIC_SIZES_MAXIMUM = LITERAL_COUNT_MAXIMUM + DISTANCE_COUNT_MAXIMUM

// LITERAL_COUNT_MINIMUM is the smallest dynamic literal alphabet.
const LITERAL_COUNT_MINIMUM = 257

// DISTANCE_COUNT_MINIMUM is the smallest dynamic distance alphabet.
const DISTANCE_COUNT_MINIMUM = 1

// COMPRESSED_PAYLOAD_SIZE_MINIMUM is exhausted encoded input.
const COMPRESSED_PAYLOAD_SIZE_MINIMUM = 0

// BIT_BUFFER_MINIMUM is an empty bit remainder.
const BIT_BUFFER_MINIMUM = 0

// BIT_COUNT_MINIMUM is an empty bit remainder.
const BIT_COUNT_MINIMUM = 0

// CHECKSUM_SIZE is Adler-32 storage width.
const CHECKSUM_SIZE = 4

// BLOCK_SOURCE_SIZE_MINIMUM is the checksum-only tail after one block header.
const BLOCK_SOURCE_SIZE_MINIMUM = 3

// BLOCK_SOURCE_SIZE_MAXIMUM follows the wrapper and first block byte.
const BLOCK_SOURCE_SIZE_MAXIMUM = BYTE_COUNT_MAXIMUM - 3

// DYNAMIC_SIZE_SOURCE_SIZE_MINIMUM admits a truncated dynamic-size payload.
const DYNAMIC_SIZE_SOURCE_SIZE_MINIMUM = 0

// DYNAMIC_SIZE_SOURCE_SIZE_MAXIMUM follows the shortest dynamic header.
const DYNAMIC_SIZE_SOURCE_SIZE_MAXIMUM = BYTE_COUNT_MAXIMUM - 6

// BLOCK_BIT_BUFFER_MINIMUM is an all-zero block header tail.
const BLOCK_BIT_BUFFER_MINIMUM = 0

// BLOCK_BIT_BUFFER_MAXIMUM is the five-bit block header tail.
const BLOCK_BIT_BUFFER_MAXIMUM = 31

// BLOCK_BIT_COUNT is unread block-header width.
const BLOCK_BIT_COUNT = 5

// STORED_COUNT_MAXIMUM is one stored block's unsigned length.
const STORED_COUNT_MAXIMUM = 1<<16 - 1

// Status reports decode result without interface boxing or owned error text.
type Status uint8

// Status_Invariants states every decode result.
func Status_Invariants(value Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_OUTPUT_TOO_SMALL),
		).
		Ensure()
}

// STATUS_OK means compressed input filled destination successfully.
const STATUS_OK Status = 0

// STATUS_INPUT_INVALID means compressed input violated zlib or DEFLATE format.
const STATUS_INPUT_INVALID Status = 1

// STATUS_OUTPUT_TOO_SMALL means decoded output exceeded destination.
const STATUS_OUTPUT_TOO_SMALL Status = 2

// Destination is caller-owned decoded storage.
type Destination []byte

// Destination_Invariants bounds one decoded stream.
func Destination_Invariants(value Destination, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_MAXIMUM).
		Ensure()
}

// Compressed is caller-owned zlib input.
type Compressed []byte

// Compressed_Invariants bounds one encoded stream.
func Compressed_Invariants(value Compressed, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_MAXIMUM).
		Ensure()
}

// Count is decoded byte count.
type Count int

// Count_Invariants bounds one decoded stream count.
func Count_Invariants(value Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_MAXIMUM).
		Ensure()
}

// Stored_Destination is caller storage remaining for one stored block.
type Stored_Destination []byte

// Stored_Destination_Invariants bounds remaining caller storage.
func Stored_Destination_Invariants(
	value Stored_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(len(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_MAXIMUM).
		Ensure()
}

// Stored_Count is one stored block's decoded byte count.
type Stored_Count int

// Stored_Count_Invariants bounds one stored block's unsigned length.
func Stored_Count_Invariants(value Stored_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), BYTE_COUNT_MINIMUM, STORED_COUNT_MAXIMUM).
		Ensure()
}

// Boolean is one decoder report.
type Boolean bool

// Boolean_Invariants states both report values.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The decoder report is true.").
		Ensure()
}

// Code_Size_Position is one position in encoded code-size order.
type Code_Size_Position int

// Code_Size_Position_Invariants bounds encoded code-size order.
func Code_Size_Position_Invariants(value Code_Size_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), CODE_SIZE_POSITION_MINIMUM, CODE_SIZE_POSITION_MAXIMUM).
		Ensure()
}

// Ordered_Code_Size_Position is one canonical code-size position.
type Ordered_Code_Size_Position int

// Ordered_Code_Size_Position_Invariants bounds canonical code-size order.
func Ordered_Code_Size_Position_Invariants(
	value Ordered_Code_Size_Position, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), CODE_SIZE_POSITION_MINIMUM, CODE_SIZE_POSITION_MAXIMUM).
		Ensure()
}

// Huffman_Symbol is one decoded alphabet member.
type Huffman_Symbol uint16

// Huffman_Symbol_Invariants bounds the largest DEFLATE alphabet.
func Huffman_Symbol_Invariants(value Huffman_Symbol, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), HUFFMAN_SYMBOL_MINIMUM, HUFFMAN_SYMBOL_MAXIMUM).
		Ensure()
}

// Match_Symbol is one DEFLATE length symbol.
type Match_Symbol uint16

// Match_Symbol_Invariants bounds the length alphabet.
func Match_Symbol_Invariants(value Match_Symbol, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), MATCH_SYMBOL_MINIMUM, MATCH_SYMBOL_MAXIMUM).
		Ensure()
}

// Distance_Symbol is one DEFLATE distance symbol.
type Distance_Symbol uint16

// Distance_Symbol_Invariants bounds the legal distance alphabet.
func Distance_Symbol_Invariants(value Distance_Symbol, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint16(uint16(value), DISTANCE_SYMBOL_MINIMUM, DISTANCE_SYMBOL_MAXIMUM).
		Ensure()
}

// Match_Size is one DEFLATE match byte count.
type Match_Size int

// Match_Size_Invariants bounds one DEFLATE match.
func Match_Size_Invariants(value Match_Size, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), MATCH_SIZE_MINIMUM, MATCH_SIZE_MAXIMUM).
		Ensure()
}

// Distance_Base is one DEFLATE backward distance before its encoded suffix.
type Distance_Base int

// Distance_Base_Invariants bounds one DEFLATE distance base.
func Distance_Base_Invariants(value Distance_Base, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), DISTANCE_MINIMUM, DISTANCE_BASE_MAXIMUM).
		Ensure()
}

// Extra_Bit_Count is one encoded match or distance suffix width.
type Extra_Bit_Count uint

// Extra_Bit_Count_Invariants bounds a DEFLATE suffix width.
func Extra_Bit_Count_Invariants(value Extra_Bit_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint(uint(value), EXTRA_BIT_COUNT_MINIMUM, EXTRA_BIT_COUNT_MAXIMUM).
		Ensure()
}

// Match_Extra_Bit_Count is one encoded match suffix width.
type Match_Extra_Bit_Count uint

// Match_Extra_Bit_Count_Invariants bounds a match suffix width.
func Match_Extra_Bit_Count_Invariants(
	value Match_Extra_Bit_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint(
			uint(value), EXTRA_BIT_COUNT_MINIMUM, MATCH_EXTRA_BIT_COUNT_MAXIMUM,
		).
		Ensure()
}

// Bit_Read_Count is one bounded reader request.
type Bit_Read_Count uint

// Bit_Read_Count_Invariants bounds one reader request.
func Bit_Read_Count_Invariants(value Bit_Read_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint(uint(value), BIT_READ_COUNT_MINIMUM, BIT_READ_COUNT_MAXIMUM).
		Ensure()
}

// Bit_Value is one bounded reader result.
type Bit_Value uint32

// Bit_Value_Invariants bounds one reader result.
func Bit_Value_Invariants(value Bit_Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), BIT_VALUE_MINIMUM, BIT_VALUE_MAXIMUM).
		Ensure()
}

// Code_Sizes is one canonical alphabet's code widths.
type Code_Sizes []uint8

// Code_Sizes_Invariants bounds the largest canonical alphabet.
func Code_Sizes_Invariants(value Code_Sizes, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), CODE_SIZES_MINIMUM, CODE_SIZES_MAXIMUM).
		Ensure()
}

// Dynamic_Sizes is one dynamic block's literal and distance widths.
type Dynamic_Sizes []uint8

// Dynamic_Sizes_Invariants bounds a dynamic block's combined alphabets.
func Dynamic_Sizes_Invariants(value Dynamic_Sizes, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), DYNAMIC_SIZES_MINIMUM, DYNAMIC_SIZES_MAXIMUM).
		Ensure()
}

// Literal_Count is one dynamic literal alphabet size.
type Literal_Count int

// Literal_Count_Invariants bounds one dynamic literal alphabet.
func Literal_Count_Invariants(value Literal_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), LITERAL_COUNT_MINIMUM, LITERAL_COUNT_MAXIMUM).
		Ensure()
}

// Distance_Count is one dynamic distance alphabet size.
type Distance_Count int

// Distance_Count_Invariants bounds one dynamic distance alphabet.
func Distance_Count_Invariants(value Distance_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), DISTANCE_COUNT_MINIMUM, DISTANCE_COUNT_MAXIMUM).
		Ensure()
}

// Checksum is one Adler-32 value in network byte order.
type Checksum [CHECKSUM_SIZE]byte

// Checksum_Invariants fixes Adler-32 storage width.
func Checksum_Invariants(value Checksum, namespace aver.Namespace) {
	aver.Always(
		len(value) == CHECKSUM_SIZE,
		"An Adler checksum occupies exactly four bytes.",
	)
}

// Compressed_Payload is zlib input after its wrapper header.
type Compressed_Payload []byte

// Compressed_Payload_Invariants bounds remaining encoded bytes.
func Compressed_Payload_Invariants(
	value Compressed_Payload, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), COMPRESSED_PAYLOAD_SIZE_MINIMUM,
			COMPRESSED_PAYLOAD_SIZE_MAXIMUM,
		).
		Ensure()
}

// Bit_Buffer holds unread low-order bits.
type Bit_Buffer uint64

// Bit_Buffer_Invariants bounds the remainder after one reader operation.
func Bit_Buffer_Invariants(value Bit_Buffer, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), BIT_BUFFER_MINIMUM, BIT_BUFFER_MAXIMUM).
		Ensure()
}

// Bit_Count is unread bit count.
type Bit_Count uint

// Bit_Count_Invariants bounds the remainder after one reader operation.
func Bit_Count_Invariants(value Bit_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint(uint(value), BIT_COUNT_MINIMUM, BIT_COUNT_MAXIMUM).
		Ensure()
}

// Bit_Reader holds bounded input cursor.
type Bit_Reader struct {
	// Source remains caller-owned compressed bytes.
	Source Compressed_Payload
	// Bits retains unread low-order DEFLATE bits.
	Bits Bit_Buffer
	// Bits_Count bounds valid low-order bits.
	Bits_Count Bit_Count
}

// Bit_Reader_Invariants composes bounded reader storage.
func Bit_Reader_Invariants(value *Bit_Reader, namespace aver.Namespace) {
	Compressed_Payload_Invariants(value.Source, namespace)
	Bit_Buffer_Invariants(value.Bits, namespace)
	Bit_Count_Invariants(value.Bits_Count, namespace)
}

// Stored_Reader is reader state at one stored block boundary.
type Stored_Reader Bit_Reader

// Stored_Reader_Invariants keeps stored-block state inside reader storage.
func Stored_Reader_Invariants(value Stored_Reader, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value.Source), BLOCK_SOURCE_SIZE_MINIMUM, BLOCK_SOURCE_SIZE_MAXIMUM,
		).
		Range_Uint64(uint64(value.Bits), BIT_BUFFER_MINIMUM, BIT_BUFFER_MAXIMUM).
		Range_Uint(uint(value.Bits_Count), BIT_COUNT_MINIMUM, BIT_COUNT_MAXIMUM).
		Ensure()
}

// Dynamic_Reader is reader state while building dynamic alphabets.
type Dynamic_Reader Bit_Reader

// Dynamic_Reader_Invariants keeps dynamic-tree state inside reader storage.
func Dynamic_Reader_Invariants(value Dynamic_Reader, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value.Source), BLOCK_SOURCE_SIZE_MINIMUM, BLOCK_SOURCE_SIZE_MAXIMUM,
		).
		Range_Uint64(uint64(value.Bits), BIT_BUFFER_MINIMUM, BIT_BUFFER_MAXIMUM).
		Range_Uint(uint(value.Bits_Count), BIT_COUNT_MINIMUM, BIT_COUNT_MAXIMUM).
		Ensure()
}

// Dynamic_Size_Reader is reader state after one dynamic header.
type Dynamic_Size_Reader Bit_Reader

// Dynamic_Size_Reader_Invariants bounds state entering dynamic alphabet widths.
func Dynamic_Size_Reader_Invariants(
	value Dynamic_Size_Reader, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value.Source), DYNAMIC_SIZE_SOURCE_SIZE_MINIMUM,
			DYNAMIC_SIZE_SOURCE_SIZE_MAXIMUM,
		).
		Range_Uint64(uint64(value.Bits), BIT_BUFFER_MINIMUM, BIT_BUFFER_MAXIMUM).
		Range_Uint(uint(value.Bits_Count), BIT_COUNT_MINIMUM, BIT_COUNT_MAXIMUM).
		Ensure()
}

// Huffman_Reader is reader state while decoding one Huffman block.
type Huffman_Reader Bit_Reader

// Huffman_Reader_Invariants keeps Huffman state inside reader storage.
func Huffman_Reader_Invariants(value Huffman_Reader, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value.Source), COMPRESSED_PAYLOAD_SIZE_MINIMUM,
			BLOCK_SOURCE_SIZE_MAXIMUM,
		).
		Range_Uint64(uint64(value.Bits), BIT_BUFFER_MINIMUM, BIT_BUFFER_MAXIMUM).
		Range_Uint(uint(value.Bits_Count), BIT_COUNT_MINIMUM, BIT_COUNT_MAXIMUM).
		Ensure()
}

// Symbol_Reader is reader state after a block header.
type Symbol_Reader Bit_Reader

// Symbol_Reader_Invariants bounds state entering one symbol read.
func Symbol_Reader_Invariants(
	value Symbol_Reader, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value.Source), COMPRESSED_PAYLOAD_SIZE_MINIMUM,
			BLOCK_SOURCE_SIZE_MAXIMUM,
		).
		Range_Uint64(uint64(value.Bits), BIT_BUFFER_MINIMUM, BIT_BUFFER_MAXIMUM).
		Range_Uint(uint(value.Bits_Count), BIT_COUNT_MINIMUM, BIT_COUNT_MAXIMUM).
		Ensure()
}

// Huffman_Decoder holds canonical code tables in fixed storage.
type Huffman_Decoder struct {
	// Counts groups canonical symbols by bit count.
	Counts [CODE_SIZE_MAXIMUM + 1]uint16
	// Symbols stores canonical order without owned slices.
	Symbols [HUFFMAN_SYMBOL_COUNT_MAXIMUM]uint16
}

// Huffman_Decoder_Invariants bounds the fixed canonical tables.
func Huffman_Decoder_Invariants(
	value *Huffman_Decoder, namespace aver.Namespace,
) {
	aver.Always(
		value.Counts[0] <= HUFFMAN_SYMBOL_COUNT_MAXIMUM,
		"A Huffman decoder counts no more symbols than its fixed table holds.",
	)
}

func code_size_index(
	index Code_Size_Position,
) (ordered_index Ordered_Code_Size_Position) {
	defer func() {
		Ordered_Code_Size_Position_Invariants(
			ordered_index, "code_size_index.ordered_index",
		)
	}()
	Code_Size_Position_Invariants(index, "code_size_index.index")
	switch index {
	case 0:
		return 16
	case 1:
		return 17
	case 2:
		return 18
	case 3:
		return 0
	case 4:
		return 8
	case 5:
		return 7
	case 6:
		return 9
	case 7:
		return 6
	case 8:
		return 10
	case 9:
		return 5
	case 10:
		return 11
	case 11:
		return 4
	case 12:
		return 12
	case 13:
		return 3
	case 14:
		return 13
	case 15:
		return 2
	case 16:
		return 14
	case 17:
		return 1
	default:
		return 15
	}
}

func match_size(
	symbol Match_Symbol,
) (size Match_Size, extra_count Match_Extra_Bit_Count) {
	defer func() {
		Match_Size_Invariants(size, "match_size.size")
		Match_Extra_Bit_Count_Invariants(extra_count, "match_size.extra_count")
	}()
	Match_Symbol_Invariants(symbol, "match_size.symbol")
	value := int(symbol)
	switch {
	case value < 265:
		return Match_Size(value - 254), 0
	case value < 269:
		return Match_Size(value*2 - 519), 1
	case value < 273:
		return Match_Size(value*4 - 1057), 2
	case value < 277:
		return Match_Size(value*8 - 2149), 3
	case value < 281:
		return Match_Size(value*16 - 4365), 4
	case value < 285:
		return Match_Size(value*32 - 8861), 5
	default:
		return 258, 0
	}
}

func distance_size(
	symbol Distance_Symbol,
) (distance Distance_Base, extra_count Extra_Bit_Count) {
	defer func() {
		Distance_Base_Invariants(distance, "distance_size.distance")
		Extra_Bit_Count_Invariants(extra_count, "distance_size.extra_count")
	}()
	Distance_Symbol_Invariants(symbol, "distance_size.symbol")
	value := int(symbol)
	if value < 4 {
		return Distance_Base(value + 1), 0
	}
	extra_count = Extra_Bit_Count(uint(value-2) >> 1)
	distance = Distance_Base(1<<(extra_count+1) + 1)
	distance += Distance_Base((value & 1) << extra_count)
	return distance, extra_count
}

// Decode_Into keeps full output as DEFLATE history, removing separate window storage.
func Decode_Into(
	destination Destination, compressed Compressed,
) (count Count, status Status) {
	defer func() {
		Count_Invariants(count, "Decode_Into.count")
		Status_Invariants(status, "Decode_Into.status")
	}()
	Destination_Invariants(destination, "Decode_Into.destination")
	Compressed_Invariants(compressed, "Decode_Into.compressed")
	if !header_valid(compressed) {
		return 0, STATUS_INPUT_INVALID
	}

	reader := Bit_Reader{Source: Compressed_Payload(compressed[2:])}
	final := Bit_Value(0)
	for final == 0 {
		final_value, available := bit_reader_read(&reader, 1)
		if !available {
			return count, STATUS_INPUT_INVALID
		}
		final = final_value
		block_kind, available := bit_reader_read(&reader, 2)
		if !available {
			return count, STATUS_INPUT_INVALID
		}

		switch block_kind {
		case 0:
			stored_count := Stored_Count(0)
			stored_count, status = decode_stored_block(
				Stored_Destination(destination[count:]), (*Stored_Reader)(&reader),
			)
			count += Count(stored_count)
		case 1:
			var literal_decoder, distance_decoder Huffman_Decoder
			fixed_decoders(&literal_decoder, &distance_decoder)
			count, status = decode_huffman_block(
				destination, count, (*Huffman_Reader)(&reader),
				&literal_decoder, &distance_decoder,
			)
		case 2:
			var literal_decoder, distance_decoder Huffman_Decoder
			if !dynamic_decoders(
				(*Dynamic_Reader)(&reader), &literal_decoder, &distance_decoder,
			) {
				return count, STATUS_INPUT_INVALID
			}
			count, status = decode_huffman_block(
				destination, count, (*Huffman_Reader)(&reader),
				&literal_decoder, &distance_decoder,
			)
		default:
			return count, STATUS_INPUT_INVALID
		}
		if status != STATUS_OK {
			return count, status
		}
	}

	reader.Bits = 0
	reader.Bits_Count = 0
	if len(reader.Source) != 4 {
		return count, STATUS_INPUT_INVALID
	}
	want := [CHECKSUM_SIZE]byte{
		reader.Source[0], reader.Source[1], reader.Source[2], reader.Source[3],
	}
	if adler32(destination[:count]) != want {
		return count, STATUS_INPUT_INVALID
	}
	return count, STATUS_OK
}

func header_valid(compressed Compressed) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "header_valid.valid") }()
	Compressed_Invariants(compressed, "header_valid.compressed")
	if len(compressed) < 6 {
		return false
	}
	method_and_window := compressed[0]
	flags := compressed[1]
	header := uint16(method_and_window)<<8 | uint16(flags)
	return method_and_window&0x0f == DEFLATE_METHOD &&
		method_and_window>>4 <= 7 &&
		header%31 == 0 &&
		flags&0x20 == 0
}

func decode_stored_block(
	destination Stored_Destination,
	reader *Stored_Reader,
) (count Stored_Count, status Status) {
	defer func() {
		Stored_Count_Invariants(count, "decode_stored_block.count")
		Status_Invariants(status, "decode_stored_block.status")
	}()
	Stored_Destination_Invariants(destination, "decode_stored_block.destination")
	Stored_Reader_Invariants(*reader, "decode_stored_block.reader")
	reader.Bits = 0
	reader.Bits_Count = 0
	base_reader := (*Bit_Reader)(reader)
	size, available := bit_reader_read(base_reader, 16)
	if !available {
		return count, STATUS_INPUT_INVALID
	}
	inverse, available := bit_reader_read(base_reader, 16)
	if !available {
		return count, STATUS_INPUT_INVALID
	}
	if uint16(inverse) != ^uint16(size) {
		return count, STATUS_INPUT_INVALID
	}
	for byte_index := 0; byte_index < int(size); byte_index++ {
		value, present := bit_reader_read(base_reader, 8)
		if !present {
			return count, STATUS_INPUT_INVALID
		}
		if int(count) == len(destination) {
			return count, STATUS_OUTPUT_TOO_SMALL
		}
		destination[count] = byte(value)
		count++
	}
	return count, STATUS_OK
}

func fixed_decoders(
	literal_decoder *Huffman_Decoder, distance_decoder *Huffman_Decoder,
) {
	Huffman_Decoder_Invariants(literal_decoder, "fixed_decoders.literal_decoder")
	Huffman_Decoder_Invariants(distance_decoder, "fixed_decoders.distance_decoder")
	var literal_sizes [HUFFMAN_SYMBOL_COUNT_MAXIMUM]uint8
	for index := 0; index <= 143; index++ {
		literal_sizes[index] = 8
	}
	for index := 144; index <= 255; index++ {
		literal_sizes[index] = 9
	}
	for index := 256; index <= 279; index++ {
		literal_sizes[index] = 7
	}
	for index := 280; index < len(literal_sizes); index++ {
		literal_sizes[index] = 8
	}
	var distance_sizes [FIXED_DISTANCE_COUNT]uint8
	for index := range distance_sizes {
		distance_sizes[index] = 5
	}
	literal_valid := huffman_build(literal_decoder, literal_sizes[:])
	aver.Always(bool(literal_valid), "The fixed literal table is canonical.")
	distance_valid := huffman_build(distance_decoder, distance_sizes[:])
	aver.Always(bool(distance_valid), "The fixed distance table is canonical.")
}

func dynamic_decoders(
	reader *Dynamic_Reader,
	literal_decoder *Huffman_Decoder,
	distance_decoder *Huffman_Decoder,
) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "dynamic_decoders.valid") }()
	Dynamic_Reader_Invariants(*reader, "dynamic_decoders.reader")
	Huffman_Decoder_Invariants(literal_decoder, "dynamic_decoders.literal_decoder")
	Huffman_Decoder_Invariants(distance_decoder, "dynamic_decoders.distance_decoder")
	var code_decoder Huffman_Decoder
	literal_count, distance_count, header_ok := dynamic_header(reader, &code_decoder)
	if !header_ok {
		return false
	}
	var sizes [LITERAL_COUNT_MAXIMUM + DISTANCE_COUNT_MAXIMUM]uint8
	total_count := int(literal_count) + int(distance_count)
	if !dynamic_sizes(
		(*Dynamic_Size_Reader)(reader), &code_decoder, sizes[:total_count],
	) {
		return false
	}
	if sizes[256] == 0 {
		return false
	}
	if !huffman_build(literal_decoder, sizes[:int(literal_count)]) {
		return false
	}
	return huffman_build(distance_decoder, sizes[int(literal_count):total_count])
}

func dynamic_header(
	reader *Dynamic_Reader,
	code_decoder *Huffman_Decoder,
) (literal_count Literal_Count, distance_count Distance_Count, valid Boolean) {
	defer func() {
		Literal_Count_Invariants(literal_count, "dynamic_header.literal_count")
		Distance_Count_Invariants(distance_count, "dynamic_header.distance_count")
		Boolean_Invariants(valid, "dynamic_header.valid")
	}()
	Dynamic_Reader_Invariants(*reader, "dynamic_header.reader")
	Huffman_Decoder_Invariants(code_decoder, "dynamic_header.code_decoder")
	base_reader := (*Bit_Reader)(reader)
	literal_count = LITERAL_COUNT_MINIMUM
	distance_count = DISTANCE_COUNT_MINIMUM
	literal_count_bits, available := bit_reader_read(base_reader, 5)
	if !available {
		return LITERAL_COUNT_MINIMUM, DISTANCE_COUNT_MINIMUM, false
	}
	distance_count_bits, available := bit_reader_read(base_reader, 5)
	if !available {
		return LITERAL_COUNT_MINIMUM, DISTANCE_COUNT_MINIMUM, false
	}
	code_count_bits, available := bit_reader_read(base_reader, 4)
	if !available {
		return LITERAL_COUNT_MINIMUM, DISTANCE_COUNT_MINIMUM, false
	}
	literal_count = Literal_Count(literal_count_bits) + 257
	distance_count = Distance_Count(distance_count_bits) + 1
	encoded_code_count := int(code_count_bits) + 4
	if literal_count > LITERAL_COUNT_MAXIMUM {
		return LITERAL_COUNT_MINIMUM, DISTANCE_COUNT_MINIMUM, false
	}
	if distance_count > DISTANCE_COUNT_MAXIMUM {
		return LITERAL_COUNT_MINIMUM, DISTANCE_COUNT_MINIMUM, false
	}
	var code_sizes [CODE_COUNT]uint8
	for index := 0; index < encoded_code_count; index++ {
		value, present := bit_reader_read(base_reader, 3)
		if !present {
			return LITERAL_COUNT_MINIMUM, DISTANCE_COUNT_MINIMUM, false
		}
		code_sizes[code_size_index(Code_Size_Position(index))] = uint8(value)
	}
	if !huffman_build(code_decoder, code_sizes[:]) {
		return LITERAL_COUNT_MINIMUM, DISTANCE_COUNT_MINIMUM, false
	}
	if code_decoder.Counts[0] == CODE_COUNT {
		return LITERAL_COUNT_MINIMUM, DISTANCE_COUNT_MINIMUM, false
	}
	return literal_count, distance_count, true
}

func dynamic_sizes(
	reader *Dynamic_Size_Reader,
	code_decoder *Huffman_Decoder,
	sizes Dynamic_Sizes,
) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "dynamic_sizes.valid") }()
	Dynamic_Size_Reader_Invariants(*reader, "dynamic_sizes.reader")
	Huffman_Decoder_Invariants(code_decoder, "dynamic_sizes.code_decoder")
	Dynamic_Sizes_Invariants(sizes, "dynamic_sizes.sizes")
	base_reader := (*Bit_Reader)(reader)
	for index := 0; index < len(sizes); {
		symbol, present := huffman_read(
			(*Symbol_Reader)(base_reader), code_decoder,
		)
		if !present {
			return false
		}
		if symbol < 16 {
			sizes[index] = uint8(symbol)
			index++
			continue
		}
		if symbol == 16 {
			if index == 0 {
				return false
			}
			extra, extra_present := bit_reader_read(base_reader, 2)
			if !extra_present {
				return false
			}
			repeat_count := int(extra) + 3
			if index+repeat_count > len(sizes) {
				return false
			}
			value := sizes[index-1]
			for repeat_index := 0; repeat_index < repeat_count; repeat_index++ {
				sizes[index] = value
				index++
			}
			continue
		}
		if symbol == 17 {
			extra, extra_present := bit_reader_read(base_reader, 3)
			if !extra_present {
				return false
			}
			repeat_count := int(extra) + 3
			if index+repeat_count > len(sizes) {
				return false
			}
			index += repeat_count
			continue
		}
		if symbol == 18 {
			extra, extra_present := bit_reader_read(base_reader, 7)
			if !extra_present {
				return false
			}
			repeat_count := int(extra) + 11
			if index+repeat_count > len(sizes) {
				return false
			}
			index += repeat_count
			continue
		}
		return false
	}
	return true
}

func decode_huffman_block(
	destination Destination, count Count, reader *Huffman_Reader,
	literal_decoder *Huffman_Decoder, distance_decoder *Huffman_Decoder,
) (next_count Count, status Status) {
	defer func() {
		Count_Invariants(next_count, "decode_huffman_block.next_count")
		Status_Invariants(status, "decode_huffman_block.status")
	}()
	Destination_Invariants(destination, "decode_huffman_block.destination")
	Count_Invariants(count, "decode_huffman_block.count")
	Huffman_Reader_Invariants(*reader, "decode_huffman_block.reader")
	Huffman_Decoder_Invariants(literal_decoder, "decode_huffman_block.literal_decoder")
	Huffman_Decoder_Invariants(distance_decoder, "decode_huffman_block.distance_decoder")
	base_reader := (*Bit_Reader)(reader)
	for more := true; more; {
		symbol, present := huffman_read((*Symbol_Reader)(base_reader), literal_decoder)
		if !present {
			return count, STATUS_INPUT_INVALID
		}
		if symbol < 256 {
			if int(count) == len(destination) {
				return count, STATUS_OUTPUT_TOO_SMALL
			}
			destination[count] = byte(symbol)
			count++
			continue
		}
		if symbol == 256 {
			return count, STATUS_OK
		}
		if symbol > 285 {
			return count, STATUS_INPUT_INVALID
		}
		match_count, extra_count := match_size(Match_Symbol(symbol))
		extra, available := bit_reader_read(base_reader, Bit_Read_Count(extra_count))
		if !available {
			return count, STATUS_INPUT_INVALID
		}
		match_count += Match_Size(extra)
		distance_symbol, present := huffman_read(
			(*Symbol_Reader)(base_reader), distance_decoder)
		if !present {
			return count, STATUS_INPUT_INVALID
		}
		if distance_symbol >= DISTANCE_COUNT_MAXIMUM {
			return count, STATUS_INPUT_INVALID
		}
		distance_base, distance_extra_count := distance_size(
			Distance_Symbol(distance_symbol))
		distance_extra, distance_available := bit_reader_read(
			base_reader, Bit_Read_Count(distance_extra_count),
		)
		if !distance_available {
			return count, STATUS_INPUT_INVALID
		}
		distance := int(distance_base) + int(distance_extra)
		if distance > int(count) {
			return count, STATUS_INPUT_INVALID
		}
		if distance > WINDOW_SIZE_MAXIMUM {
			return count, STATUS_INPUT_INVALID
		}
		for copied_index := 0; copied_index < int(match_count); copied_index++ {
			if int(count) == len(destination) {
				return count, STATUS_OUTPUT_TOO_SMALL
			}
			destination[count] = destination[int(count)-distance]
			count++
		}
	}
	return count, STATUS_INPUT_INVALID
}

func huffman_build(
	decoder *Huffman_Decoder, sizes Code_Sizes,
) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "huffman_build.valid") }()
	Huffman_Decoder_Invariants(decoder, "huffman_build.decoder")
	Code_Sizes_Invariants(sizes, "huffman_build.sizes")
	clear(decoder.Counts[:])
	clear(decoder.Symbols[:])
	decoder_size := 0
	for _, size := range sizes {
		if size > CODE_SIZE_MAXIMUM {
			return false
		}
		decoder.Counts[size]++
		if size != 0 {
			decoder_size++
		}
	}
	if decoder_size == 0 {
		return true
	}

	space := 1
	for index := 1; index <= CODE_SIZE_MAXIMUM; index++ {
		space = space*2 - int(decoder.Counts[index])
		if space < 0 {
			return false
		}
	}
	if space != 0 {
		if decoder_size != 1 {
			return false
		}
		if decoder.Counts[1] != 1 {
			return false
		}
	}

	var offsets [CODE_SIZE_MAXIMUM + 1]uint16
	for index := 1; index < CODE_SIZE_MAXIMUM; index++ {
		offsets[index+1] = offsets[index] + decoder.Counts[index]
	}
	for symbol, size := range sizes {
		if size == 0 {
			continue
		}
		position := offsets[size]
		decoder.Symbols[position] = uint16(symbol)
		offsets[size]++
	}
	return true
}

func huffman_read(
	reader *Symbol_Reader,
	decoder *Huffman_Decoder,
) (symbol Huffman_Symbol, present Boolean) {
	defer func() {
		Huffman_Symbol_Invariants(symbol, "huffman_read.symbol")
		Boolean_Invariants(present, "huffman_read.present")
	}()
	Symbol_Reader_Invariants(*reader, "huffman_read.reader")
	Huffman_Decoder_Invariants(decoder, "huffman_read.decoder")
	base_reader := (*Bit_Reader)(reader)
	var code uint32
	var first uint32
	var symbol_index uint32
	for size_index := 1; size_index <= CODE_SIZE_MAXIMUM; size_index++ {
		bit, available := bit_reader_read(base_reader, 1)
		if !available {
			return 0, false
		}
		code |= uint32(bit)
		count := uint32(decoder.Counts[size_index])
		if code < first+count {
			return Huffman_Symbol(decoder.Symbols[symbol_index+code-first]), true
		}
		symbol_index += count
		first = (first + count) << 1
		code <<= 1
	}
	return 0, false
}

func bit_reader_read(
	reader *Bit_Reader, count Bit_Read_Count,
) (value Bit_Value, available Boolean) {
	defer func() {
		Bit_Value_Invariants(value, "bit_reader_read.value")
		Boolean_Invariants(available, "bit_reader_read.available")
	}()
	Bit_Reader_Invariants(reader, "bit_reader_read.reader")
	Bit_Read_Count_Invariants(count, "bit_reader_read.count")
	if count == 0 {
		return 0, true
	}
	for reader.Bits_Count < Bit_Count(count) {
		if len(reader.Source) == 0 {
			return 0, false
		}
		reader.Bits |= Bit_Buffer(uint64(reader.Source[0]) << reader.Bits_Count)
		reader.Source = reader.Source[1:]
		reader.Bits_Count += 8
	}
	mask := uint64(1<<count) - 1
	value = Bit_Value(uint64(reader.Bits) & mask)
	reader.Bits >>= count
	reader.Bits_Count -= Bit_Count(count)
	return value, true
}

func adler32(value Destination) (checksum Checksum) {
	defer func() { Checksum_Invariants(checksum, "adler32.checksum") }()
	Destination_Invariants(value, "adler32.value")
	first := uint32(1)
	second := uint32(0)
	for _, one := range value {
		first = (first + uint32(one)) % ADLER_MODULUS
		second = (second + first) % ADLER_MODULUS
	}
	combined := second<<16 | first
	return Checksum{
		byte(combined >> 24), byte(combined >> 16), byte(combined >> 8), byte(combined),
	}
}
