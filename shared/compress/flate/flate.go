// Package flate encodes and decodes raw DEFLATE streams in fixed caller storage.
package flate

import (
	"unsafe"

	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
)

// BYTE_COUNT_MEBIBYTE_EXPONENT states caller-storage policy as binary scale.
const BYTE_COUNT_MEBIBYTE_EXPONENT = 6

// BYTE_COUNT_MEBIBYTE_COUNT states caller-storage policy in IEC units.
const BYTE_COUNT_MEBIBYTE_COUNT = 1 << BYTE_COUNT_MEBIBYTE_EXPONENT

// BYTE_COUNT_MAXIMUM caps every byte slice accepted by package entry points.
const BYTE_COUNT_MAXIMUM = BYTE_COUNT_MEBIBYTE_COUNT * bits.MEBIBYTE_BYTES

// BYTE_COUNT_UNVALIDATED_MAXIMUM admits one rejected boundary witness.
const BYTE_COUNT_UNVALIDATED_MAXIMUM = BYTE_COUNT_MAXIMUM + 1

// WINDOW_BIT_COUNT is RFC 1951 history-window exponent.
const WINDOW_BIT_COUNT = bits.BIT_COUNT_16_MAXIMUM - 1

// WINDOW_KIBIBYTE_COUNT states RFC 1951 history window in IEC units.
const WINDOW_KIBIBYTE_COUNT = (1 << WINDOW_BIT_COUNT) / bits.KIBIBYTE_BYTES

// DICTIONARY_SIZE_MAXIMUM matches RFC 1951 history window.
const DICTIONARY_SIZE_MAXIMUM = WINDOW_KIBIBYTE_COUNT * bits.KIBIBYTE_BYTES

// NO_COMPRESSION emits stored blocks.
const NO_COMPRESSION = 0

// BEST_SPEED spends least bounded search work.
const BEST_SPEED = NO_COMPRESSION + 1

// COMPRESSION_LEVEL_COUNT is standard searched-level count.
const COMPRESSION_LEVEL_COUNT = 9

// BEST_COMPRESSION spends most bounded search work.
const BEST_COMPRESSION = BEST_SPEED + COMPRESSION_LEVEL_COUNT - 1

// DEFAULT_COMPRESSION selects DEFAULT_COMPRESSION_LEVEL search.
const DEFAULT_COMPRESSION = NO_COMPRESSION - 1

// HUFFMAN_ONLY disables repeated-span search.
const HUFFMAN_ONLY = DEFAULT_COMPRESSION - 1

// COMPRESSION_LEVEL_2 is second searched compression level.
const COMPRESSION_LEVEL_2 = BEST_SPEED + 1

// COMPRESSION_LEVEL_3 is third searched compression level.
const COMPRESSION_LEVEL_3 = COMPRESSION_LEVEL_2 + 1

// COMPRESSION_LEVEL_4 is fourth searched compression level.
const COMPRESSION_LEVEL_4 = COMPRESSION_LEVEL_3 + 1

// COMPRESSION_LEVEL_5 is fifth searched compression level.
const COMPRESSION_LEVEL_5 = COMPRESSION_LEVEL_4 + 1

// COMPRESSION_LEVEL_6 is sixth searched compression level.
const COMPRESSION_LEVEL_6 = COMPRESSION_LEVEL_5 + 1

// COMPRESSION_LEVEL_7 is seventh searched compression level.
const COMPRESSION_LEVEL_7 = COMPRESSION_LEVEL_6 + 1

// COMPRESSION_LEVEL_8 is eighth searched compression level.
const COMPRESSION_LEVEL_8 = COMPRESSION_LEVEL_7 + 1

// DEFAULT_COMPRESSION_LEVEL is standard upper-middle search effort.
const DEFAULT_COMPRESSION_LEVEL = COMPRESSION_LEVEL_6

// WINDOW_SIZE is largest legal backward distance.
const WINDOW_SIZE = DICTIONARY_SIZE_MAXIMUM

// WINDOW_MASK maps absolute positions into caller workspace.
const WINDOW_MASK = WINDOW_SIZE - 1

// HASH_BIT_COUNT bounds prefix lookup width.
const HASH_BIT_COUNT = bits.BIT_COUNT_16_MAXIMUM

// HASH_COUNT bounds prefix lookup table.
const HASH_COUNT = 1 << HASH_BIT_COUNT

// HASH_MASK maps prefix hash into caller workspace.
const HASH_MASK = HASH_COUNT - 1

// HASH_INPUT_BYTE_COUNT is prefix width consumed by match lookup.
const HASH_INPUT_BYTE_COUNT = MATCH_SIZE_MINIMUM

// HASH_MULTIPLIER is fixed multiplicative coefficient for prefix lookup.
const HASH_MULTIPLIER uint32 = 0x1e35a7bd

// HASH_PRODUCT_BIT_COUNT is width of prefix hash product.
const HASH_PRODUCT_BIT_COUNT = bits.BIT_COUNT_32_MAXIMUM

// MATCH_SIZE_MINIMUM is shortest RFC 1951 repeated span.
const MATCH_SIZE_MINIMUM = 3

// MATCH_SIZE_MAXIMUM is longest RFC 1951 repeated span.
const MATCH_SIZE_MAXIMUM = 1<<bits.BIT_COUNT_8_MAXIMUM - 1 + MATCH_SIZE_MINIMUM

// MATCH_COPY_COUNT_MAXIMUM leaves the shortest complete match writable.
const MATCH_COPY_COUNT_MAXIMUM = BYTE_COUNT_MAXIMUM - MATCH_SIZE_MINIMUM

// MATCH_SYMBOL_MINIMUM is first RFC 1951 length symbol.
const MATCH_SYMBOL_MINIMUM = LITERAL_SYMBOL_END + 1

// MATCH_SYMBOL_COUNT is RFC 1951 length-code count.
const MATCH_SYMBOL_COUNT = 29

// MATCH_DIRECT_SYMBOL_COUNT is prefix with no length suffix.
const MATCH_DIRECT_SYMBOL_COUNT = 8

// MATCH_DIRECT_SIZE_MAXIMUM is final length carried wholly by its symbol.
const MATCH_DIRECT_SIZE_MAXIMUM = MATCH_SIZE_MINIMUM + MATCH_DIRECT_SYMBOL_COUNT - 1

// MATCH_EXTRA_SYMBOL_MINIMUM starts grouped length suffixes.
const MATCH_EXTRA_SYMBOL_MINIMUM = MATCH_SYMBOL_MINIMUM + MATCH_DIRECT_SYMBOL_COUNT

// MATCH_EXTRA_SYMBOL_GROUP_SIZE is symbols sharing one suffix width.
const MATCH_EXTRA_SYMBOL_GROUP_SIZE = 1 << MATCH_EXTRA_BASE_SHIFT

// MATCH_EXTRA_GROUP_MINIMUM converts the first grouped suffix width once.
const MATCH_EXTRA_GROUP_MINIMUM = int(GROUPED_EXTRA_BIT_COUNT_MINIMUM)

// MATCH_EXTRA_BASE_SHIFT places first grouped length range after direct lengths.
const MATCH_EXTRA_BASE_SHIFT = 2

// MATCH_SYMBOL_MAXIMUM is final RFC 1951 length symbol.
const MATCH_SYMBOL_MAXIMUM = MATCH_SYMBOL_MINIMUM + MATCH_SYMBOL_COUNT - 1

// DISTANCE_MINIMUM is nearest legal history byte.
const DISTANCE_MINIMUM = 1

// DISTANCE_MAXIMUM is farthest legal history byte.
const DISTANCE_MAXIMUM = WINDOW_SIZE

// DISTANCE_BASE_MAXIMUM leaves final suffix to reach window end.
const DISTANCE_BASE_MAXIMUM = DISTANCE_MAXIMUM - (1 << DISTANCE_EXTRA_BIT_COUNT_MAXIMUM) + 1

// STORED_SIZE_MAXIMUM is largest unsigned stored block.
const STORED_SIZE_MAXIMUM = int(bits.WORD_16_MAXIMUM)

// LITERAL_SYMBOL_END terminates one Huffman block.
const LITERAL_SYMBOL_END = 1 << bits.BIT_COUNT_8_MAXIMUM

// LITERAL_SYMBOL_MAXIMUM excludes reserved literal symbols.
const LITERAL_SYMBOL_MAXIMUM = MATCH_SYMBOL_MAXIMUM

// DISTANCE_SYMBOL_RESERVED_COUNT is fixed-tree suffix absent from dynamic streams.
const DISTANCE_SYMBOL_RESERVED_COUNT = 2

// DISTANCE_DIRECT_SYMBOL_COUNT is prefix with no distance suffix.
const DISTANCE_DIRECT_SYMBOL_COUNT = BINARY_RADIX * BINARY_RADIX

// DISTANCE_EXTRA_SYMBOL_OFFSET aligns grouped distance symbols with suffix widths.
const DISTANCE_EXTRA_SYMBOL_OFFSET = DISTANCE_SYMBOL_RESERVED_COUNT

// DISTANCE_GROUP_SHIFT divides paired distance symbols.
const DISTANCE_GROUP_SHIFT = FINAL_BIT_COUNT

// GROUPED_EXTRA_BIT_COUNT_MINIMUM starts length and distance grouped suffixes.
const GROUPED_EXTRA_BIT_COUNT_MINIMUM = BIT_COUNT_MINIMUM + 1

// DISTANCE_SYMBOL_COUNT excludes two reserved distance symbols.
const DISTANCE_SYMBOL_COUNT = FIXED_DISTANCE_COUNT - DISTANCE_SYMBOL_RESERVED_COUNT

// DISTANCE_SYMBOL_MINIMUM is nearest distance alphabet member.
const DISTANCE_SYMBOL_MINIMUM uint16 = bits.WORD_16_MINIMUM

// DISTANCE_SYMBOL_MAXIMUM is final legal distance alphabet member.
const DISTANCE_SYMBOL_MAXIMUM = DISTANCE_SYMBOL_COUNT - 1

// CODE_SIZE_MAXIMUM is largest RFC 1951 Huffman code width.
const CODE_SIZE_MAXIMUM = bits.BIT_COUNT_16_MAXIMUM - 1

// LITERAL_COUNT_MAXIMUM is largest dynamic literal alphabet.
const LITERAL_COUNT_MAXIMUM = LITERAL_SYMBOL_MAXIMUM + 1

// FIXED_LITERAL_RESERVED_COUNT is fixed literal-tree suffix absent from dynamic streams.
const FIXED_LITERAL_RESERVED_COUNT = DISTANCE_SYMBOL_RESERVED_COUNT

// FIXED_LITERAL_COUNT includes reserved fixed-tree symbols.
const FIXED_LITERAL_COUNT = LITERAL_COUNT_MAXIMUM + FIXED_LITERAL_RESERVED_COUNT

// FIXED_DISTANCE_COUNT includes reserved fixed-tree symbols.
const FIXED_DISTANCE_COUNT = bits.BIT_COUNT_32_MAXIMUM

// FIXED_LITERAL_FIRST_COUNT is RFC fixed-tree first range width.
const FIXED_LITERAL_FIRST_COUNT = 144

// FIXED_LITERAL_FIRST_MAXIMUM is final symbol using FIXED_CODE_SIZE range.
const FIXED_LITERAL_FIRST_MAXIMUM = FIXED_LITERAL_FIRST_COUNT - 1

// FIXED_LITERAL_SECOND_MINIMUM follows first fixed-tree range.
const FIXED_LITERAL_SECOND_MINIMUM = FIXED_LITERAL_FIRST_MAXIMUM + 1

// FIXED_LITERAL_SECOND_MAXIMUM is final byte-valued literal.
const FIXED_LITERAL_SECOND_MAXIMUM = LITERAL_SYMBOL_END - 1

// FIXED_LITERAL_SECOND_COUNT is the nine-bit fixed-tree range width.
const FIXED_LITERAL_SECOND_COUNT = FIXED_LITERAL_SECOND_MAXIMUM - FIXED_LITERAL_SECOND_MINIMUM + 1

// FIXED_LITERAL_THIRD_MINIMUM is end-of-block symbol.
const FIXED_LITERAL_THIRD_MINIMUM = LITERAL_SYMBOL_END

// FIXED_LITERAL_THIRD_COUNT is RFC fixed-tree third range width.
const FIXED_LITERAL_THIRD_COUNT = 24

// FIXED_LITERAL_THIRD_MAXIMUM is final symbol using FIXED_THIRD_CODE_SIZE range.
const FIXED_LITERAL_THIRD_MAXIMUM = FIXED_LITERAL_THIRD_MINIMUM + FIXED_LITERAL_THIRD_COUNT - 1

// FIXED_LITERAL_FOURTH_MINIMUM follows third fixed-tree range.
const FIXED_LITERAL_FOURTH_MINIMUM = FIXED_LITERAL_THIRD_MAXIMUM + 1

// FIXED_LITERAL_FOURTH_COUNT is the reserved eight-bit fixed-tree suffix width.
const FIXED_LITERAL_FOURTH_COUNT = FIXED_LITERAL_COUNT - FIXED_LITERAL_FOURTH_MINIMUM

// FIXED_CODE_SIZE is common first and fourth fixed-tree width.
const FIXED_CODE_SIZE = BYTE_BIT_COUNT

// FIXED_SECOND_CODE_SIZE is second fixed-tree width.
const FIXED_SECOND_CODE_SIZE = FIXED_CODE_SIZE + 1

// FIXED_THIRD_CODE_SIZE is third fixed-tree width.
const FIXED_THIRD_CODE_SIZE = FIXED_CODE_SIZE - 1

// HUFFMAN_LOOKUP_CODE_SIZE covers every fixed literal code in one indexed read.
const HUFFMAN_LOOKUP_CODE_SIZE = FIXED_SECOND_CODE_SIZE

// HUFFMAN_LOOKUP_COUNT covers each prefix of lookup width.
const HUFFMAN_LOOKUP_COUNT = 1 << HUFFMAN_LOOKUP_CODE_SIZE

// HUFFMAN_LOOKUP_MASK selects one lookup-width prefix.
const HUFFMAN_LOOKUP_MASK = HUFFMAN_LOOKUP_COUNT - 1

// FIXED_PREFIX_MINIMUM is the first fixed lookup prefix.
const FIXED_PREFIX_MINIMUM Fixed_Prefix = Fixed_Prefix(bits.WORD_16_MINIMUM)

// FIXED_PREFIX_MAXIMUM is the last fixed lookup prefix.
const FIXED_PREFIX_MAXIMUM Fixed_Prefix = Fixed_Prefix(HUFFMAN_LOOKUP_MASK)

// FIXED_DECODE_SIZE_MINIMUM is the shortest fixed literal code.
const FIXED_DECODE_SIZE_MINIMUM Fixed_Decode_Size = Fixed_Decode_Size(FIXED_THIRD_CODE_SIZE)

// FIXED_DECODE_SIZE_MAXIMUM is the longest fixed literal code.
const FIXED_DECODE_SIZE_MAXIMUM Fixed_Decode_Size = Fixed_Decode_Size(FIXED_SECOND_CODE_SIZE)

// FIXED_THIRD_PREFIX_SHIFT removes suffix bits beyond the shortest fixed code.
const FIXED_THIRD_PREFIX_SHIFT = HUFFMAN_LOOKUP_CODE_SIZE - FIXED_THIRD_CODE_SIZE

// FIXED_FIRST_PREFIX_SHIFT removes the suffix bit beyond an eight-bit fixed code.
const FIXED_FIRST_PREFIX_SHIFT = HUFFMAN_LOOKUP_CODE_SIZE - FIXED_CODE_SIZE

// FIXED_FIRST_CODE_BASE follows the shorter seven-bit range in canonical order.
const FIXED_FIRST_CODE_BASE = FIXED_LITERAL_THIRD_COUNT << FINAL_BIT_COUNT

// FIXED_FOURTH_CODE_BASE follows the first eight-bit range in canonical order.
const FIXED_FOURTH_CODE_BASE = FIXED_FIRST_CODE_BASE + FIXED_LITERAL_FIRST_COUNT

// FIXED_SECOND_CODE_BASE follows both eight-bit ranges in canonical order.
const FIXED_SECOND_CODE_BASE = FIXED_FOURTH_CODE_BASE*BINARY_RADIX +
	FIXED_LITERAL_FOURTH_COUNT*BINARY_RADIX

// BIT_REVERSE_SINGLE_MASK selects alternating bits across one 16-bit word.
const BIT_REVERSE_SINGLE_MASK = bits.WORD_16_MAXIMUM / (1<<BINARY_RADIX - 1)

// BIT_REVERSE_PAIR_MASK selects alternating two-bit groups across one 16-bit word.
const BIT_REVERSE_PAIR_MASK = bits.WORD_16_MAXIMUM / (1<<BINARY_RADIX + 1)

// BIT_REVERSE_BYTE_SINGLE_MASK narrows the shared 16-bit formula to one byte.
const BIT_REVERSE_BYTE_SINGLE_MASK = uint8(BIT_REVERSE_SINGLE_MASK & uint16(bits.WORD_8_MAXIMUM))

// BIT_REVERSE_BYTE_PAIR_MASK narrows the shared 16-bit formula to one byte.
const BIT_REVERSE_BYTE_PAIR_MASK = uint8(BIT_REVERSE_PAIR_MASK & uint16(bits.WORD_8_MAXIMUM))

// BIT_REVERSE_NIBBLE_BIT_COUNT names the final byte reversal group.
const BIT_REVERSE_NIBBLE_BIT_COUNT = BINARY_RADIX * BINARY_RADIX

// BIT_REVERSE_NIBBLE_MASK keeps the four reversed low distance bits.
const BIT_REVERSE_NIBBLE_MASK = bits.WORD_8_MAXIMUM >> BIT_REVERSE_NIBBLE_BIT_COUNT

// CODE_SIZE_ORDER_PREFIX_COUNT holds repeat symbols and zero code size.
const CODE_SIZE_ORDER_PREFIX_COUNT = REPEAT_SYMBOL_COUNT + 1

// CODE_SIZE_ORDER_PAIR_COUNT holds alternating sizes around FIXED_CODE_SIZE.
const CODE_SIZE_ORDER_PAIR_COUNT = FIXED_CODE_SIZE - 1

// CODE_SIZE_ORDER_SUFFIX_COUNT holds CODE_SIZE_MAXIMUM.
const CODE_SIZE_ORDER_SUFFIX_COUNT = 1

// CODE_COUNT is code-size alphabet width.
const CODE_COUNT = CODE_SIZE_ORDER_PREFIX_COUNT +
	CODE_SIZE_ORDER_PAIR_COUNT*BINARY_RADIX + CODE_SIZE_ORDER_SUFFIX_COUNT

// ENCODED_CODE_COUNT_MINIMUM is mandatory dynamic header prefix.
const ENCODED_CODE_COUNT_MINIMUM = 4

// ENCODED_CODE_COUNT_MAXIMUM is complete code-size alphabet.
const ENCODED_CODE_COUNT_MAXIMUM = CODE_COUNT

// DYNAMIC_SIZES_MINIMUM is smallest combined literal and distance alphabet.
const DYNAMIC_SIZES_MINIMUM = LITERAL_COUNT_MINIMUM + DISTANCE_COUNT_MINIMUM

// DYNAMIC_SIZES_MAXIMUM is largest combined literal and distance alphabet.
const DYNAMIC_SIZES_MAXIMUM = LITERAL_COUNT_MAXIMUM + DISTANCE_SYMBOL_COUNT

// DYNAMIC_POSITION_MAXIMUM is final populated dynamic alphabet slot.
const DYNAMIC_POSITION_MAXIMUM = DYNAMIC_SIZES_MAXIMUM - 1

// LITERAL_COUNT_MINIMUM is mandatory dynamic literal alphabet.
const LITERAL_COUNT_MINIMUM = MATCH_SYMBOL_MINIMUM

// DISTANCE_COUNT_MINIMUM is mandatory dynamic distance alphabet.
const DISTANCE_COUNT_MINIMUM = 1

// POSITION_MAXIMUM includes dictionary prefix before maximum source.
const POSITION_MAXIMUM = BYTE_COUNT_MAXIMUM + DICTIONARY_SIZE_MAXIMUM

// VIRTUAL_BYTE_POSITION_MAXIMUM is final byte in maximum history and source.
const VIRTUAL_BYTE_POSITION_MAXIMUM = POSITION_MAXIMUM - 1

// SEQUENCE_POSITION_MAXIMUM leaves three readable bytes.
const SEQUENCE_POSITION_MAXIMUM = POSITION_MAXIMUM - MATCH_SIZE_MINIMUM

// MATCH_POSITION_MAXIMUM leaves a later sequence start.
const MATCH_POSITION_MAXIMUM = SEQUENCE_POSITION_MAXIMUM - 1

// BIT_COUNT_MAXIMUM bounds one RFC 1951 read or write request.
const BIT_COUNT_MAXIMUM = bits.BIT_COUNT_16_MAXIMUM

// BIT_BUFFER_MAXIMUM bounds retained low-order bits between byte transfers.
const BIT_BUFFER_MAXIMUM = bits.WORD_16_MAXIMUM

// SYMBOL_MAXIMUM includes reserved fixed-tree members.
const SYMBOL_MAXIMUM = FIXED_LITERAL_COUNT - 1

// CHAIN_COUNT_MAXIMUM matches BEST_COMPRESSION search effort.
const CHAIN_COUNT_MAXIMUM = CHAIN_COUNT_LEVEL_8 * BINARY_RADIX * BINARY_RADIX

// BYTE_COUNT_MINIMUM admits empty caller storage.
const BYTE_COUNT_MINIMUM = bits.BIT_COUNT_MINIMUM

// POSITION_MINIMUM starts every virtual history coordinate.
const POSITION_MINIMUM = bits.BIT_COUNT_MINIMUM

// BIT_COUNT_MINIMUM admits fields without suffix bits.
const BIT_COUNT_MINIMUM uint8 = bits.BIT_COUNT_MINIMUM

// BIT_BUFFER_MINIMUM is empty retained bit state.
const BIT_BUFFER_MINIMUM uint64 = bits.WORD_64_MINIMUM

// BIT_VALUE_MINIMUM is all-zero bit field.
const BIT_VALUE_MINIMUM uint32 = bits.WORD_32_MINIMUM

// BIT_VALUE_MAXIMUM is widest bounded bit field.
const BIT_VALUE_MAXIMUM uint32 = uint32(bits.WORD_16_MAXIMUM)

// BYTE_BIT_COUNT is width of one emitted or loaded byte.
const BYTE_BIT_COUNT = bits.BIT_COUNT_8_MAXIMUM

// STORED_SIZE_BIT_COUNT is width of one stored-block size field.
const STORED_SIZE_BIT_COUNT = bits.BIT_COUNT_16_MAXIMUM

// DYNAMIC_ALPHABET_COUNT_BIT_COUNT is width of literal and distance count fields.
const DYNAMIC_ALPHABET_COUNT_BIT_COUNT = 5

// DYNAMIC_CODE_COUNT_BIT_COUNT is width of encoded code-size count field.
const DYNAMIC_CODE_COUNT_BIT_COUNT = 4

// CODE_SIZE_BIT_COUNT is width of one encoded code size.
const CODE_SIZE_BIT_COUNT = 3

// FIXED_DISTANCE_BIT_COUNT is width of every fixed distance symbol.
const FIXED_DISTANCE_BIT_COUNT = DYNAMIC_ALPHABET_COUNT_BIT_COUNT

// FIXED_MATCH_SIZE_FIELD consumes the length suffix first.
const FIXED_MATCH_SIZE_FIELD = bits.BIT_COUNT_MINIMUM

// FIXED_MATCH_DISTANCE_SYMBOL_FIELD follows the length suffix.
const FIXED_MATCH_DISTANCE_SYMBOL_FIELD = FIXED_MATCH_SIZE_FIELD + FINAL_BIT_COUNT

// FIXED_MATCH_DISTANCE_FIELD follows the fixed distance symbol.
const FIXED_MATCH_DISTANCE_FIELD = FIXED_MATCH_DISTANCE_SYMBOL_FIELD + FINAL_BIT_COUNT

// FIXED_MATCH_FIELD_COUNT bounds one length-distance tuple.
const FIXED_MATCH_FIELD_COUNT = FIXED_MATCH_DISTANCE_FIELD + FINAL_BIT_COUNT

// BYTE_VALUE_MINIMUM is all-zero byte.
const BYTE_VALUE_MINIMUM uint8 = bits.WORD_8_MINIMUM

// BYTE_VALUE_MAXIMUM is all-one byte.
const BYTE_VALUE_MAXIMUM uint8 = bits.WORD_8_MAXIMUM

// SYMBOL_MINIMUM is first literal alphabet member.
const SYMBOL_MINIMUM uint16 = bits.WORD_16_MINIMUM

// CODE_SIZES_MINIMUM admits one-symbol Huffman alphabet.
const CODE_SIZES_MINIMUM = 1

// CODE_SIZES_MAXIMUM is largest fixed Huffman alphabet.
const CODE_SIZES_MAXIMUM = FIXED_LITERAL_COUNT

// SEARCH_SOURCE_SIZE_MINIMUM admits first searched byte.
const SEARCH_SOURCE_SIZE_MINIMUM = 1

// LATER_SEQUENCE_POSITION_MINIMUM follows candidate zero.
const LATER_SEQUENCE_POSITION_MINIMUM = 1

// SYMBOL_COUNT_MINIMUM admits empty canonical alphabet.
const SYMBOL_COUNT_MINIMUM = bits.BIT_COUNT_MINIMUM

// SYMBOL_COUNT_MAXIMUM fills fixed canonical table.
const SYMBOL_COUNT_MAXIMUM = FIXED_LITERAL_COUNT

// MATCHING_SIZE_MINIMUM admits immediate mismatch.
const MATCHING_SIZE_MINIMUM = bits.BIT_COUNT_MINIMUM

// MATCHING_SIZE_MAXIMUM stops at RFC match bound.
const MATCHING_SIZE_MAXIMUM = MATCH_SIZE_MAXIMUM

// DYNAMIC_CURSOR_MINIMUM starts combined alphabet.
const DYNAMIC_CURSOR_MINIMUM = bits.BIT_COUNT_MINIMUM

// MATCH_EXTRA_VALUE_MINIMUM is empty length suffix.
const MATCH_EXTRA_VALUE_MINIMUM uint32 = bits.WORD_32_MINIMUM

// MATCH_EXTRA_VALUE_MAXIMUM fills widest length suffix.
const MATCH_EXTRA_VALUE_MAXIMUM = 1<<MATCH_EXTRA_BIT_COUNT_MAXIMUM - 1

// DISTANCE_EXTRA_VALUE_MINIMUM is empty distance suffix.
const DISTANCE_EXTRA_VALUE_MINIMUM uint32 = bits.WORD_32_MINIMUM

// DISTANCE_EXTRA_VALUE_MAXIMUM fills widest distance suffix.
const DISTANCE_EXTRA_VALUE_MAXIMUM = 1<<DISTANCE_EXTRA_BIT_COUNT_MAXIMUM - 1

// STORED_SOURCE_SIZE_MINIMUM admits empty stored block.
const STORED_SOURCE_SIZE_MINIMUM = bits.BIT_COUNT_MINIMUM

// STORED_SOURCE_SIZE_MAXIMUM fills unsigned stored length.
const STORED_SOURCE_SIZE_MAXIMUM = STORED_SIZE_MAXIMUM

// MATCH_EXTRA_BIT_COUNT_MAXIMUM is widest length suffix.
const MATCH_EXTRA_BIT_COUNT_MAXIMUM = 5

// DISTANCE_EXTRA_BIT_COUNT_MAXIMUM is widest distance suffix.
const DISTANCE_EXTRA_BIT_COUNT_MAXIMUM = 13

// REPEAT_SYMBOL_MINIMUM starts dynamic code-size repeat alphabet.
const REPEAT_SYMBOL_MINIMUM = CODE_SIZE_MAXIMUM + 1

// REPEAT_SYMBOL_MIDDLE is zero-repeat short form.
const REPEAT_SYMBOL_MIDDLE = REPEAT_SYMBOL_MINIMUM + 1

// REPEAT_SYMBOL_MAXIMUM is zero-repeat long form.
const REPEAT_SYMBOL_MAXIMUM = REPEAT_SYMBOL_MIDDLE + 1

// REPEAT_SYMBOL_COUNT is dynamic code-size repeat alphabet width.
const REPEAT_SYMBOL_COUNT = REPEAT_SYMBOL_MAXIMUM - REPEAT_SYMBOL_MINIMUM + 1

// REPEAT_PREVIOUS_BASE is minimum previous-size repetition.
const REPEAT_PREVIOUS_BASE = 3

// REPEAT_PREVIOUS_EXTRA_BIT_COUNT encodes previous-size repetition suffix.
const REPEAT_PREVIOUS_EXTRA_BIT_COUNT = 2

// REPEAT_ZERO_SHORT_BASE is minimum short zero repetition.
const REPEAT_ZERO_SHORT_BASE = REPEAT_PREVIOUS_BASE

// REPEAT_ZERO_SHORT_EXTRA_BIT_COUNT encodes short zero repetition suffix.
const REPEAT_ZERO_SHORT_EXTRA_BIT_COUNT = 3

// REPEAT_ZERO_LONG_BASE follows maximum short zero repetition.
const REPEAT_ZERO_LONG_BASE = REPEAT_ZERO_SHORT_BASE + (1 << REPEAT_ZERO_SHORT_EXTRA_BIT_COUNT)

// REPEAT_ZERO_LONG_EXTRA_BIT_COUNT encodes long zero repetition suffix.
const REPEAT_ZERO_LONG_EXTRA_BIT_COUNT = 7

// REPEAT_COUNT_MINIMUM is shortest dynamic code-size repetition.
const REPEAT_COUNT_MINIMUM = REPEAT_PREVIOUS_BASE

// REPEAT_COUNT_MAXIMUM is longest dynamic code-size repetition.
const REPEAT_COUNT_MAXIMUM = REPEAT_ZERO_LONG_BASE + (1 << REPEAT_ZERO_LONG_EXTRA_BIT_COUNT) - 1

// FINAL_BIT_COUNT encodes final-block flag.
const FINAL_BIT_COUNT = 1

// FINAL_BLOCK_BIT_VALUE marks final block.
const FINAL_BLOCK_BIT_VALUE = 1

// BLOCK_KIND_BIT_COUNT encodes RFC 1951 block selector.
const BLOCK_KIND_BIT_COUNT = 2

// BLOCK_HEADER_BIT_COUNT combines final flag and block selector.
const BLOCK_HEADER_BIT_COUNT = FINAL_BIT_COUNT + BLOCK_KIND_BIT_COUNT

// BINARY_RADIX is cardinality of one bit.
const BINARY_RADIX = 2

// LOW_BIT_MASK selects least-significant bit.
const LOW_BIT_MASK = BINARY_RADIX - 1

// BLOCK_KIND_STORED selects byte-aligned payload.
const BLOCK_KIND_STORED = bits.BIT_COUNT_MINIMUM

// BLOCK_KIND_FIXED selects fixed Huffman tables.
const BLOCK_KIND_FIXED = BLOCK_KIND_STORED + 1

// BLOCK_KIND_DYNAMIC selects encoded Huffman tables.
const BLOCK_KIND_DYNAMIC = BLOCK_KIND_FIXED + 1

// BLOCK_KIND_RESERVED is invalid RFC 1951 selector.
const BLOCK_KIND_RESERVED = BLOCK_KIND_DYNAMIC + 1

// CHAIN_COUNT_MINIMUM is level-one prefix search effort.
const CHAIN_COUNT_MINIMUM = 4

// CHAIN_COUNT_LEVEL_2 doubles minimum search effort.
const CHAIN_COUNT_LEVEL_2 = CHAIN_COUNT_MINIMUM * BINARY_RADIX

// CHAIN_COUNT_LEVEL_3 quadruples level-two search effort.
const CHAIN_COUNT_LEVEL_3 = CHAIN_COUNT_LEVEL_2 * BINARY_RADIX * BINARY_RADIX

// CHAIN_COUNT_LEVEL_4 halves level-three search effort.
const CHAIN_COUNT_LEVEL_4 = CHAIN_COUNT_LEVEL_3 / BINARY_RADIX

// CHAIN_COUNT_LEVEL_5 restores level-three search effort.
const CHAIN_COUNT_LEVEL_5 = CHAIN_COUNT_LEVEL_3

// CHAIN_COUNT_LEVEL_6 quadruples level-five search effort.
const CHAIN_COUNT_LEVEL_6 = CHAIN_COUNT_LEVEL_5 * BINARY_RADIX * BINARY_RADIX

// CHAIN_COUNT_LEVEL_7 doubles level-six search effort.
const CHAIN_COUNT_LEVEL_7 = CHAIN_COUNT_LEVEL_6 * BINARY_RADIX

// CHAIN_COUNT_LEVEL_8 quadruples level-seven search effort.
const CHAIN_COUNT_LEVEL_8 = CHAIN_COUNT_LEVEL_7 * BINARY_RADIX * BINARY_RADIX

// HASH_MINIMUM is first prefix bucket.
const HASH_MINIMUM = bits.BIT_COUNT_MINIMUM

// HASH_MAXIMUM is final prefix bucket.
const HASH_MAXIMUM = HASH_COUNT - 1

// HASH_POSITIONS_COUNT_MINIMUM admits missing hostile workspace.
const HASH_POSITIONS_COUNT_MINIMUM = bits.BIT_COUNT_MINIMUM

// HASH_POSITIONS_COUNT_MAXIMUM admits one-slot oversized hostile workspace.
const HASH_POSITIONS_COUNT_MAXIMUM = HASH_COUNT + 1

// HISTORY_POSITIONS_COUNT_MINIMUM admits missing hostile workspace.
const HISTORY_POSITIONS_COUNT_MINIMUM = bits.BIT_COUNT_MINIMUM

// HISTORY_POSITIONS_COUNT_MAXIMUM admits one-slot oversized hostile workspace.
const HISTORY_POSITIONS_COUNT_MAXIMUM = WINDOW_SIZE + 1

// HISTORY_SIZE_MINIMUM admits no preset dictionary.
const HISTORY_SIZE_MINIMUM = bits.BIT_COUNT_MINIMUM

// HISTORY_SIZE_MAXIMUM matches RFC 1951 window.
const HISTORY_SIZE_MAXIMUM = DICTIONARY_SIZE_MAXIMUM

// STATUS_MINIMUM is successful operation value.
const STATUS_MINIMUM = 0

// STATUS_MAXIMUM is invalid-storage operation value.
const STATUS_MAXIMUM uint8 = STATUS_STORAGE_INVALID

// WRITER_STATE_READY admits bounded writes.
const WRITER_STATE_READY = false

// WRITER_STATE_EXHAUSTED stops writes after caller storage ends.
const WRITER_STATE_EXHAUSTED = true

// PENDING_BITS_MAXIMUM fills seven retained low-order bits.
const PENDING_BITS_MAXIMUM uint64 = 1<<PENDING_BIT_COUNT_MAXIMUM - 1

// PENDING_BIT_COUNT_MAXIMUM is final partial-byte width.
const PENDING_BIT_COUNT_MAXIMUM uint8 = bits.BIT_COUNT_8_MAXIMUM - 1

// HUFFMAN_COUNT_MINIMUM admits unused code widths.
const HUFFMAN_COUNT_MINIMUM uint16 = bits.WORD_16_MINIMUM

// HUFFMAN_COUNT_MAXIMUM bounds largest fixed alphabet.
const HUFFMAN_COUNT_MAXIMUM uint16 = uint16(FIXED_LITERAL_COUNT)

// LEVEL_UNVALIDATED_MINIMUM covers hostile int8 input.
const LEVEL_UNVALIDATED_MINIMUM int8 = bits.INTEGER_8_MINIMUM

// LEVEL_UNVALIDATED_MAXIMUM covers hostile int8 input.
const LEVEL_UNVALIDATED_MAXIMUM int8 = bits.INTEGER_8_MAXIMUM

// Level_Unvalidated is caller compression level before domain check.
type Level_Unvalidated int8

// Level_Unvalidated_Invariants keeps hostile level inside explicit wire type.
func Level_Unvalidated_Invariants(
	value Level_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int8(
			int8(value), LEVEL_UNVALIDATED_MINIMUM, LEVEL_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Level selects standard DEFLATE effort contract.
type Level int8

// Level_Invariants bounds standard compression levels.
func Level_Invariants(value Level, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int8(int8(value), int8(HUFFMAN_ONLY), int8(BEST_COMPRESSION)).
		Ensure()
}

// Compression_Level excludes modes handled before prefix search.
type Compression_Level uint8

// Compression_Level_Invariants bounds standard searching levels.
func Compression_Level_Invariants(
	value Compression_Level, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), BEST_SPEED, BEST_COMPRESSION).
		Ensure()
}

// Encode_Status excludes decoder-only malformed-input result.
type Encode_Status uint8

// Encode_Status_Invariants states every encoder outcome.
func Encode_Status_Invariants(
	value Encode_Status, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), STATUS_OK, STATUS_OUTPUT_TOO_SMALL,
			STATUS_LEVEL_INVALID, STATUS_STORAGE_INVALID,
		).
		Ensure()
}

// Decode_Status excludes encoder-only invalid-level result.
type Decode_Status uint8

// Decode_Status_Invariants states every decoder outcome.
func Decode_Status_Invariants(
	value Decode_Status, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), STATUS_OK, STATUS_INPUT_INVALID,
			STATUS_OUTPUT_TOO_SMALL, STATUS_STORAGE_INVALID,
		).
		Ensure()
}

// Block_Status excludes storage checks completed at public boundary.
type Block_Status uint8

// Block_Status_Invariants states every payload outcome.
func Block_Status_Invariants(
	value Block_Status, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), STATUS_OK, STATUS_INPUT_INVALID,
			STATUS_OUTPUT_TOO_SMALL,
		).
		Ensure()
}

// STATUS_OK means operation consumed valid input and completed output.
const STATUS_OK = STATUS_MINIMUM

// STATUS_INPUT_INVALID means compressed bytes violate RFC 1951.
const STATUS_INPUT_INVALID = STATUS_OK + 1

// STATUS_OUTPUT_TOO_SMALL means caller destination ended before operation.
const STATUS_OUTPUT_TOO_SMALL = STATUS_INPUT_INVALID + 1

// STATUS_LEVEL_INVALID means compression level falls outside standard values.
const STATUS_LEVEL_INVALID = STATUS_OUTPUT_TOO_SMALL + 1

// STATUS_STORAGE_INVALID means storage exceeds bounds, aliases input, or omits workspace.
const STATUS_STORAGE_INVALID = STATUS_LEVEL_INVALID + 1

// Count reports complete bytes written.
type Count int

// Count_Invariants bounds complete output bytes.
func Count_Invariants(value Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_MAXIMUM).
		Ensure()
}

// Destination_Unvalidated keeps hostile output size outside decoder state.
type Destination_Unvalidated []byte

// Destination_Unvalidated_Invariants bounds rejection before conversion.
func Destination_Unvalidated_Invariants(
	value Destination_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Source_Unvalidated keeps hostile source size outside encoder state.
type Source_Unvalidated []byte

// Source_Unvalidated_Invariants bounds rejection before conversion.
func Source_Unvalidated_Invariants(
	value Source_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Compressed_Unvalidated keeps hostile wire size outside decoder state.
type Compressed_Unvalidated []byte

// Compressed_Unvalidated_Invariants bounds rejection before conversion.
func Compressed_Unvalidated_Invariants(
	value Compressed_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Dictionary_Unvalidated keeps hostile history size outside codec state.
type Dictionary_Unvalidated []byte

// Dictionary_Unvalidated_Invariants bounds rejection before conversion.
func Dictionary_Unvalidated_Invariants(
	value Dictionary_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Destination is caller-owned output storage.
type Destination []byte

// Destination_Invariants bounds caller output storage.
func Destination_Invariants(value Destination, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_MAXIMUM).
		Ensure()
}

// Match_Destination excludes storage too short for any complete match.
type Match_Destination []byte

// Match_Destination_Invariants states the complete-copy storage domain.
func Match_Destination_Invariants(
	value Match_Destination, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), MATCH_SIZE_MINIMUM, BYTE_COUNT_MAXIMUM).
		Ensure()
}

// Source is caller-owned uncompressed input.
type Source []byte

// Source_Invariants bounds caller uncompressed input.
func Source_Invariants(value Source, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_MAXIMUM).
		Ensure()
}

// Search_Source excludes empty input handled before match search.
type Search_Source []byte

// Search_Source_Invariants gives every search at least one source byte.
func Search_Source_Invariants(value Search_Source, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), SEARCH_SOURCE_SIZE_MINIMUM, BYTE_COUNT_MAXIMUM,
		).
		Ensure()
}

// Compressed is caller-owned raw DEFLATE input.
type Compressed []byte

// Compressed_Invariants bounds caller compressed input.
func Compressed_Invariants(value Compressed, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_MAXIMUM).
		Ensure()
}

// Dictionary is caller-owned preset history.
type Dictionary []byte

// Dictionary_Invariants bounds caller preset history input.
func Dictionary_Invariants(value Dictionary, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_MAXIMUM).
		Ensure()
}

// History is validated final preset dictionary window.
type History []byte

// History_Invariants bounds decoder-visible preset history.
func History_Invariants(value History, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), HISTORY_SIZE_MINIMUM, HISTORY_SIZE_MAXIMUM).
		Ensure()
}

// Bytes is generic bounded storage used only for alias validation.
type Bytes []byte

// Bytes_Invariants bounds alias-validation storage.
func Bytes_Invariants(value Bytes, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_MAXIMUM).
		Ensure()
}

// Hash_Positions is exact prefix lookup storage.
type Hash_Positions []int32

// Hash_Positions_Invariants states exact lookup capacity.
func Hash_Positions_Invariants(value Hash_Positions, namespace invariant.Namespace) {
	invariant.Always(
		len(value) == HASH_COUNT,
		"Hash positions have one slot for every prefix bucket.",
	)
}

// History_Positions is exact collision-chain storage.
type History_Positions []int32

// History_Positions_Invariants states exact history capacity.
func History_Positions_Invariants(value History_Positions, namespace invariant.Namespace) {
	invariant.Always(
		len(value) == WINDOW_SIZE,
		"History positions have one slot for every window byte.",
	)
}

// Hash_Positions_Unvalidated is caller-supplied prefix lookup storage.
type Hash_Positions_Unvalidated []int32

// Hash_Positions_Unvalidated_Invariants bounds hostile caller storage.
func Hash_Positions_Unvalidated_Invariants(
	value Hash_Positions_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), HASH_POSITIONS_COUNT_MINIMUM,
			HASH_POSITIONS_COUNT_MAXIMUM,
		).
		Ensure()
}

// History_Positions_Unvalidated is caller-supplied collision-chain storage.
type History_Positions_Unvalidated []int32

// History_Positions_Unvalidated_Invariants bounds hostile caller storage.
func History_Positions_Unvalidated_Invariants(
	value History_Positions_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), HISTORY_POSITIONS_COUNT_MINIMUM,
			HISTORY_POSITIONS_COUNT_MAXIMUM,
		).
		Ensure()
}

// Workspace_Unvalidated is caller-owned encoder storage before size checks.
type Workspace_Unvalidated struct {
	// Heads must supply one slot per prefix hash after validation.
	Heads Hash_Positions_Unvalidated
	// Previous must supply one slot per window position after validation.
	Previous History_Positions_Unvalidated
}

// Workspace_Unvalidated_Invariants composes bounded hostile storage.
func Workspace_Unvalidated_Invariants(
	value Workspace_Unvalidated, namespace invariant.Namespace,
) {
	Hash_Positions_Unvalidated_Invariants(value.Heads, namespace)
	History_Positions_Unvalidated_Invariants(value.Previous, namespace)
}

// Workspace is validated exact encoder storage.
type Workspace struct {
	// Heads indexes current three-byte prefixes.
	Heads Hash_Positions
	// Previous links candidates inside one history window.
	Previous History_Positions
}

// Workspace_Invariants composes exact caller storage dimensions.
func Workspace_Invariants(value *Workspace, namespace invariant.Namespace) {
	Hash_Positions_Invariants(value.Heads, namespace)
	History_Positions_Invariants(value.Previous, namespace)
}

// Virtual_Byte_Position indexes dictionary followed by source.
type Virtual_Byte_Position int

// Virtual_Byte_Position_Invariants excludes one-past-end coordinate.
func Virtual_Byte_Position_Invariants(
	value Virtual_Byte_Position, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), POSITION_MINIMUM, VIRTUAL_BYTE_POSITION_MAXIMUM,
		).
		Ensure()
}

// Sequence_Position leaves three bytes for prefix hashing.
type Sequence_Position int

// Sequence_Position_Invariants proves every hash read remains present.
func Sequence_Position_Invariants(
	value Sequence_Position, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), POSITION_MINIMUM, SEQUENCE_POSITION_MAXIMUM).
		Ensure()
}

// Later_Sequence_Position follows at least one candidate byte.
type Later_Sequence_Position int

// Later_Sequence_Position_Invariants excludes impossible first-position matches.
func Later_Sequence_Position_Invariants(
	value Later_Sequence_Position, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), LATER_SEQUENCE_POSITION_MINIMUM,
			SEQUENCE_POSITION_MAXIMUM,
		).
		Ensure()
}

// Match_Position is strictly before one sequence position.
type Match_Position int

// Match_Position_Invariants bounds a prior hash-chain candidate.
func Match_Position_Invariants(
	value Match_Position, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), POSITION_MINIMUM, MATCH_POSITION_MAXIMUM).
		Ensure()
}

// Match keeps absence separate from legal three-byte minimum.
type Match struct {
	// Position remains zero when no candidate reaches legal match size.
	Position Match_Position
	// Size remains legal while Present selects absence.
	Size Match_Size
	// Present rejects short prefix-hash collisions.
	Present Boolean
}

// Match_Invariants prevents short hash collisions becoming encoded matches.
func Match_Invariants(value Match, namespace invariant.Namespace) {
	Match_Position_Invariants(value.Position, namespace)
	Match_Size_Invariants(value.Size, namespace)
	Boolean_Invariants(value.Present, namespace)
}

// Match_Size is one legal repeated byte count.
type Match_Size int

// Match_Size_Invariants rejects short and oversized matches.
func Match_Size_Invariants(value Match_Size, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), MATCH_SIZE_MINIMUM, MATCH_SIZE_MAXIMUM).
		Ensure()
}

// Match_Count leaves room for at least the shortest legal match.
type Match_Count int

// Match_Count_Invariants excludes positions that require a partial copy.
func Match_Count_Invariants(value Match_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), BYTE_COUNT_MINIMUM, MATCH_COPY_COUNT_MAXIMUM).
		Ensure()
}

// Matching_Size admits candidate prefixes shorter than a legal match.
type Matching_Size int

// Matching_Size_Invariants bounds comparison work to one match.
func Matching_Size_Invariants(value Matching_Size, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), MATCHING_SIZE_MINIMUM, MATCHING_SIZE_MAXIMUM,
		).
		Ensure()
}

// Distance is one complete legal backward distance.
type Distance int

// Distance_Invariants keeps history lookup inside RFC window.
func Distance_Invariants(value Distance, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), DISTANCE_MINIMUM, DISTANCE_MAXIMUM).
		Ensure()
}

// Distance_Base is distance before encoded suffix.
type Distance_Base int

// Distance_Base_Invariants bounds table-derived distance base.
func Distance_Base_Invariants(value Distance_Base, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), DISTANCE_MINIMUM, DISTANCE_BASE_MAXIMUM).
		Ensure()
}

// Encoded_Code_Count is dynamic header code-size prefix length.
type Encoded_Code_Count int

// Encoded_Code_Count_Invariants bounds encoded code-size order.
func Encoded_Code_Count_Invariants(
	value Encoded_Code_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), ENCODED_CODE_COUNT_MINIMUM, ENCODED_CODE_COUNT_MAXIMUM,
		).
		Ensure()
}

// Literal_Count is one dynamic literal alphabet width.
type Literal_Count int

// Literal_Count_Invariants enforces RFC dynamic header domain.
func Literal_Count_Invariants(value Literal_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), LITERAL_COUNT_MINIMUM, LITERAL_COUNT_MAXIMUM).
		Ensure()
}

// Distance_Count is one dynamic distance alphabet width.
type Distance_Count int

// Distance_Count_Invariants enforces RFC dynamic header domain.
func Distance_Count_Invariants(
	value Distance_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), DISTANCE_COUNT_MINIMUM, DISTANCE_SYMBOL_COUNT).
		Ensure()
}

// Dynamic_Position is one populated combined alphabet slot.
type Dynamic_Position int

// Dynamic_Position_Invariants prevents combined alphabet escape.
func Dynamic_Position_Invariants(
	value Dynamic_Position, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), DYNAMIC_CURSOR_MINIMUM, DYNAMIC_POSITION_MAXIMUM,
		).
		Ensure()
}

// Dynamic_Cursor admits one-past-end completion.
type Dynamic_Cursor int

// Dynamic_Cursor_Invariants bounds progress through combined alphabet.
func Dynamic_Cursor_Invariants(
	value Dynamic_Cursor, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), DYNAMIC_CURSOR_MINIMUM, DYNAMIC_SIZES_MAXIMUM,
		).
		Ensure()
}

// Repeat_Count is one decoded dynamic code-size run.
type Repeat_Count int

// Repeat_Count_Invariants bounds all three repeat symbols.
func Repeat_Count_Invariants(value Repeat_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), REPEAT_COUNT_MINIMUM, REPEAT_COUNT_MAXIMUM).
		Ensure()
}

// Bit_Count is one bounded bit width.
type Bit_Count uint8

// Bit_Count_Invariants bounds one read or write request.
func Bit_Count_Invariants(value Bit_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), BIT_COUNT_MINIMUM, BIT_COUNT_MAXIMUM).
		Ensure()
}

// Match_Extra_Bit_Count is one length suffix width.
type Match_Extra_Bit_Count uint8

// Match_Extra_Bit_Count_Invariants bounds length suffix reads and writes.
func Match_Extra_Bit_Count_Invariants(
	value Match_Extra_Bit_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), BIT_COUNT_MINIMUM, MATCH_EXTRA_BIT_COUNT_MAXIMUM).
		Ensure()
}

// Distance_Extra_Bit_Count is one distance suffix width.
type Distance_Extra_Bit_Count uint8

// Distance_Extra_Bit_Count_Invariants bounds distance suffix reads and writes.
func Distance_Extra_Bit_Count_Invariants(
	value Distance_Extra_Bit_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint8(
			uint8(value), BIT_COUNT_MINIMUM, DISTANCE_EXTRA_BIT_COUNT_MAXIMUM,
		).
		Ensure()
}

// Bit_Value is one bounded bit field.
type Bit_Value uint32

// Bit_Value_Invariants bounds one read or write field.
func Bit_Value_Invariants(value Bit_Value, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint32(uint32(value), BIT_VALUE_MINIMUM, BIT_VALUE_MAXIMUM).
		Ensure()
}

// Match_Extra_Value is one decoded or encoded length suffix.
type Match_Extra_Value uint32

// Match_Extra_Value_Invariants bounds five suffix bits.
func Match_Extra_Value_Invariants(
	value Match_Extra_Value, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint32(
			uint32(value), MATCH_EXTRA_VALUE_MINIMUM, MATCH_EXTRA_VALUE_MAXIMUM,
		).
		Ensure()
}

// Distance_Extra_Value is one decoded or encoded distance suffix.
type Distance_Extra_Value uint32

// Distance_Extra_Value_Invariants bounds thirteen suffix bits.
func Distance_Extra_Value_Invariants(
	value Distance_Extra_Value, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint32(
			uint32(value), DISTANCE_EXTRA_VALUE_MINIMUM,
			DISTANCE_EXTRA_VALUE_MAXIMUM,
		).
		Ensure()
}

// Byte_Value is one byte crossing bit boundary.
type Byte_Value byte

// Byte_Value_Invariants bounds full byte domain.
func Byte_Value_Invariants(value Byte_Value, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), BYTE_VALUE_MINIMUM, BYTE_VALUE_MAXIMUM).
		Ensure()
}

// Boolean is one branch report.
type Boolean bool

// Boolean_Invariants states both branch results.
func Boolean_Invariants(value Boolean, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "The DEFLATE branch report is true.").
		Ensure()
}

// Symbol is one Huffman alphabet member.
type Symbol uint16

// Symbol_Invariants bounds largest fixed alphabet.
func Symbol_Invariants(value Symbol, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint16(uint16(value), SYMBOL_MINIMUM, SYMBOL_MAXIMUM).
		Ensure()
}

// Fixed_Symbol excludes two reserved literal alphabet members.
type Fixed_Symbol uint16

// Fixed_Symbol_Invariants bounds every encoder-emitted fixed symbol.
func Fixed_Symbol_Invariants(value Fixed_Symbol, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint16(uint16(value), SYMBOL_MINIMUM, LITERAL_SYMBOL_MAXIMUM).
		Ensure()
}

// Fixed_Prefix is one complete nine-bit fixed-tree lookup key.
type Fixed_Prefix uint16

// Fixed_Prefix_Invariants bounds every possible fixed-tree lookup key.
func Fixed_Prefix_Invariants(value Fixed_Prefix, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint16(
			uint16(value), uint16(FIXED_PREFIX_MINIMUM), uint16(FIXED_PREFIX_MAXIMUM),
		).
		Ensure()
}

// Fixed_Decode_Size is one fixed literal code width.
type Fixed_Decode_Size uint8

// Fixed_Decode_Size_Invariants bounds the three fixed literal code widths.
func Fixed_Decode_Size_Invariants(value Fixed_Decode_Size, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value),
			uint8(FIXED_DECODE_SIZE_MINIMUM), uint8(FIXED_CODE_SIZE),
			uint8(FIXED_DECODE_SIZE_MAXIMUM),
		).
		Ensure()
}

// Fixed_Decoded_Symbol keeps one symbol paired with its consumed prefix width.
type Fixed_Decoded_Symbol struct {
	// Symbol stays paired so cursor movement cannot use another prefix's value.
	Symbol Symbol
	// Size stays paired so cursor movement cannot use another symbol's width.
	Size Fixed_Decode_Size
}

// Fixed_Decoded_Symbol_Invariants keeps the symbol and its cursor movement bounded.
func Fixed_Decoded_Symbol_Invariants(
	value Fixed_Decoded_Symbol, namespace invariant.Namespace,
) {
	Symbol_Invariants(value.Symbol, namespace)
	Fixed_Decode_Size_Invariants(value.Size, namespace)
}

// Match_Symbol is one RFC length alphabet member.
type Match_Symbol uint16

// Match_Symbol_Invariants excludes literals and reserved symbols.
func Match_Symbol_Invariants(value Match_Symbol, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint16(uint16(value), MATCH_SYMBOL_MINIMUM, MATCH_SYMBOL_MAXIMUM).
		Ensure()
}

// Distance_Symbol is one legal RFC distance alphabet member.
type Distance_Symbol uint16

// Distance_Symbol_Invariants excludes reserved fixed-tree members.
func Distance_Symbol_Invariants(
	value Distance_Symbol, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint16(
			uint16(value), DISTANCE_SYMBOL_MINIMUM, DISTANCE_SYMBOL_MAXIMUM,
		).
		Ensure()
}

// Repeat_Symbol is one dynamic code-size run member.
type Repeat_Symbol uint16

// Repeat_Symbol_Invariants states all legal repeat operations.
func Repeat_Symbol_Invariants(value Repeat_Symbol, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Uint16(
			uint16(value), REPEAT_SYMBOL_MINIMUM, REPEAT_SYMBOL_MIDDLE,
			REPEAT_SYMBOL_MAXIMUM,
		).
		Ensure()
}

// Code_Sizes holds one caller-local Huffman width alphabet.
type Code_Sizes []uint8

// Code_Sizes_Invariants bounds one Huffman alphabet.
func Code_Sizes_Invariants(value Code_Sizes, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), CODE_SIZES_MINIMUM, CODE_SIZES_MAXIMUM).
		Ensure()
}

// Code_Size_Alphabet always spans the RFC permutation table.
type Code_Size_Alphabet []uint8

// Code_Size_Alphabet_Invariants prevents permutation indexing outside its table.
func Code_Size_Alphabet_Invariants(
	value Code_Size_Alphabet, namespace invariant.Namespace,
) {
	invariant.Always(
		len(value) == CODE_COUNT,
		"Dynamic code sizes retain every permutation slot.",
	)
}

// Dynamic_Sizes is one combined dynamic literal and distance alphabet.
type Dynamic_Sizes []uint8

// Dynamic_Sizes_Invariants bounds both header-declared alphabets together.
func Dynamic_Sizes_Invariants(value Dynamic_Sizes, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), DYNAMIC_SIZES_MINIMUM, DYNAMIC_SIZES_MAXIMUM).
		Ensure()
}

// Stored_Source is one unsigned stored block payload.
type Stored_Source []byte

// Stored_Source_Invariants prevents uint16 length truncation.
func Stored_Source_Invariants(value Stored_Source, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), STORED_SOURCE_SIZE_MINIMUM, STORED_SOURCE_SIZE_MAXIMUM,
		).
		Ensure()
}

// Block_Kind is one RFC 1951 block selector.
type Block_Kind uint8

// Block_Kind_Invariants states stored, fixed, dynamic, and reserved selectors.
func Block_Kind_Invariants(value Block_Kind, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), BLOCK_KIND_STORED, BLOCK_KIND_FIXED,
			BLOCK_KIND_DYNAMIC, BLOCK_KIND_RESERVED,
		).
		Ensure()
}

// Chain_Count bounds one prefix search effort.
type Chain_Count int

// Chain_Count_Invariants bounds level-selected search effort.
func Chain_Count_Invariants(value Chain_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), CHAIN_COUNT_MINIMUM, CHAIN_COUNT_MAXIMUM).
		Ensure()
}

// Hash is one caller-workspace prefix bucket.
type Hash int

// Hash_Invariants bounds prefix table index.
func Hash_Invariants(value Hash, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), HASH_MINIMUM, HASH_MAXIMUM).
		Ensure()
}

// Bit_Storage is caller storage held by one bit cursor.
type Bit_Storage []byte

// Bit_Storage_Invariants enforces package byte boundary on whole storage.
func Bit_Storage_Invariants(value Bit_Storage, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_MAXIMUM).
		Ensure()
}

// Byte_Position is one bit-cursor byte offset.
type Byte_Position int

// Byte_Position_Invariants rejects negative and oversized cursors.
func Byte_Position_Invariants(
	value Byte_Position, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), BYTE_COUNT_MINIMUM, BYTE_COUNT_MAXIMUM).
		Ensure()
}

// Pending_Bits retains one partial byte between cursor operations.
type Pending_Bits uint64

// Pending_Bits_Invariants keeps pending storage inside bounded scratch word.
func Pending_Bits_Invariants(
	value Pending_Bits, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), BIT_BUFFER_MINIMUM, PENDING_BITS_MAXIMUM).
		Ensure()
}

// Pending_Bit_Count is unread or unwritten partial-byte width.
type Pending_Bit_Count uint8

// Pending_Bit_Count_Invariants keeps normalized cursor state below one byte.
func Pending_Bit_Count_Invariants(
	value Pending_Bit_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint8(
			uint8(value), BIT_COUNT_MINIMUM, PENDING_BIT_COUNT_MAXIMUM,
		).
		Ensure()
}

// Writer_State stops work after caller destination fills.
type Writer_State bool

// Writer_State_Invariants rejects states outside ready and exhausted.
func Writer_State_Invariants(value Writer_State, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "The writer exhausted caller storage.").
		Ensure()
}

// Bit_Writer keeps one bounded output cursor.
type Bit_Writer struct {
	// Destination remains caller-owned during partial writes.
	Destination Bit_Storage
	// Position counts complete emitted bytes.
	Position Byte_Position
	// Bits retains low-order pending output bits.
	Bits Pending_Bits
	// Bits_Count tracks pending output width.
	Bits_Count Pending_Bit_Count
	// State stops mutation after caller storage ends.
	State Writer_State
}

// Bit_Writer_Invariants composes one output cursor.
func Bit_Writer_Invariants(value *Bit_Writer, namespace invariant.Namespace) {
	Bit_Storage_Invariants(value.Destination, namespace)
	Byte_Position_Invariants(value.Position, namespace)
	Pending_Bits_Invariants(value.Bits, namespace)
	Pending_Bit_Count_Invariants(value.Bits_Count, namespace)
	Writer_State_Invariants(value.State, namespace)
	invariant.Always(
		int(value.Position) <= len(value.Destination),
		"Writer position does not cross caller destination.",
	)
	invariant.Always(
		value.Bits_Count <= Pending_Bit_Count(PENDING_BIT_COUNT_MAXIMUM),
		"Writer flushes every complete pending byte.",
	)
	invariant.Always(
		uint64(value.Bits) < uint64(1)<<value.Bits_Count,
		"Writer retains only declared low-order bits.",
	)
}

// Bit_Reader keeps one bounded input cursor.
type Bit_Reader struct {
	// Source stays caller-owned during parsing.
	Source Bit_Storage
	// Position counts loaded source bytes.
	Position Byte_Position
	// Bits retains unread low-order input bits.
	Bits Pending_Bits
	// Bits_Count tracks unread input width.
	Bits_Count Pending_Bit_Count
}

// Bit_Reader_Invariants composes one input cursor.
func Bit_Reader_Invariants(value *Bit_Reader, namespace invariant.Namespace) {
	Bit_Storage_Invariants(value.Source, namespace)
	Byte_Position_Invariants(value.Position, namespace)
	Pending_Bits_Invariants(value.Bits, namespace)
	Pending_Bit_Count_Invariants(value.Bits_Count, namespace)
	invariant.Always(
		int(value.Position) <= len(value.Source),
		"Reader position does not cross compressed input.",
	)
	invariant.Always(
		value.Bits_Count <= Pending_Bit_Count(PENDING_BIT_COUNT_MAXIMUM),
		"Reader retains fewer than one loaded byte between requests.",
	)
	invariant.Always(
		uint64(value.Bits) < uint64(1)<<value.Bits_Count,
		"Reader retains only declared low-order bits.",
	)
}

// Fixed_Bit_Cursor keeps only mutable state so immutable storage stays outside each handoff.
type Fixed_Bit_Cursor struct {
	// Bits keeps unread input out of the smaller public cursor.
	Bits Bit_Value
	// Count proves which low-order bits remain live.
	Count Bit_Count
	// Position prevents reads beyond hostile compressed input.
	Position Byte_Position
}

// Fixed_Bit_Cursor_Invariants composes the mutable fixed-match bit register.
func Fixed_Bit_Cursor_Invariants(value Fixed_Bit_Cursor, namespace invariant.Namespace) {
	Bit_Value_Invariants(value.Bits, namespace)
	Bit_Count_Invariants(value.Count, namespace)
	Byte_Position_Invariants(value.Position, namespace)
}

// Fixed_Match holds one fixed length-distance result.
type Fixed_Match struct {
	// Size carries the decoded length into the bounded copy.
	Size Match_Size
	// Distance carries the decoded offset into the bounded copy.
	Distance Distance
	// Valid rejects truncated and reserved encodings before copying.
	Valid Boolean
}

// Fixed_Match_Invariants composes one fixed length-distance result.
func Fixed_Match_Invariants(value *Fixed_Match, namespace invariant.Namespace) {
	Match_Size_Invariants(value.Size, namespace)
	Distance_Invariants(value.Distance, namespace)
	Boolean_Invariants(value.Valid, namespace)
}

// Canonical_Symbol_Count bounds populated canonical symbol prefix.
type Canonical_Symbol_Count int

// Canonical_Symbol_Count_Invariants prevents canonical table escape.
func Canonical_Symbol_Count_Invariants(
	value Canonical_Symbol_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), SYMBOL_COUNT_MINIMUM, SYMBOL_COUNT_MAXIMUM).
		Ensure()
}

// Canonical_Code_Size bounds populated canonical code width.
type Canonical_Code_Size uint8

// Canonical_Code_Size_Invariants prevents canonical count-table escape.
func Canonical_Code_Size_Invariants(
	value Canonical_Code_Size, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), BIT_COUNT_MINIMUM, CODE_SIZE_MAXIMUM).
		Ensure()
}

// Huffman_Decoder stores canonical counts and symbols without owned slices.
type Huffman_Decoder struct {
	// Counts maps each bit width to alphabet population.
	Counts [CODE_SIZE_MAXIMUM + 1]uint16
	// Symbols stores canonical alphabet order.
	Symbols [FIXED_LITERAL_COUNT]uint16
	// Lookup_Symbols maps short reversed prefixes directly to symbols.
	Lookup_Symbols [HUFFMAN_LOOKUP_COUNT]uint16
	// Lookup_Sizes distinguishes populated prefixes and gives consumed widths.
	Lookup_Sizes [HUFFMAN_LOOKUP_COUNT]uint8
	// Symbol_Count bounds populated Symbols prefix.
	Symbol_Count Canonical_Symbol_Count
	// Maximum_Code_Size stops reads at populated widths.
	Maximum_Code_Size Canonical_Code_Size
}

// Huffman_Decoder_Invariants composes bounds used for every table access.
func Huffman_Decoder_Invariants(value *Huffman_Decoder, namespace invariant.Namespace) {
	Canonical_Symbol_Count_Invariants(value.Symbol_Count, namespace)
	Canonical_Code_Size_Invariants(value.Maximum_Code_Size, namespace)
	invariant.Always(
		(value.Symbol_Count == 0) == (value.Maximum_Code_Size == 0),
		"Empty canonical alphabet has no maximum code size.",
	)
}

// Encode_Into writes one raw DEFLATE stream without preset history.
func Encode_Into(
	destination Destination_Unvalidated,
	workspace Workspace_Unvalidated,
	source Source_Unvalidated,
	level_unvalidated Level_Unvalidated,
) (count Count, status Encode_Status) {
	defer func() {
		Count_Invariants(count, "Encode_Into.count")
		Encode_Status_Invariants(status, "Encode_Into.status")
	}()
	Destination_Unvalidated_Invariants(destination, "Encode_Into.destination")
	Workspace_Unvalidated_Invariants(workspace, "Encode_Into.workspace")
	Source_Unvalidated_Invariants(source, "Encode_Into.source")
	Level_Unvalidated_Invariants(level_unvalidated, "Encode_Into.level_unvalidated")
	return Encode_Dictionary_Into(
		destination, workspace, source, nil, level_unvalidated,
	)
}

// Encode_Dictionary_Into writes one raw DEFLATE stream with preset history.
func Encode_Dictionary_Into(
	destination_unvalidated Destination_Unvalidated,
	workspace_unvalidated Workspace_Unvalidated,
	source_unvalidated Source_Unvalidated,
	dictionary_unvalidated Dictionary_Unvalidated,
	level_unvalidated Level_Unvalidated,
) (count Count, status Encode_Status) {
	defer func() {
		Count_Invariants(count, "Encode_Dictionary_Into.count")
		Encode_Status_Invariants(status, "Encode_Dictionary_Into.status")
	}()
	Destination_Unvalidated_Invariants(
		destination_unvalidated, "Encode_Dictionary_Into.destination",
	)
	Workspace_Unvalidated_Invariants(
		workspace_unvalidated, "Encode_Dictionary_Into.workspace_unvalidated",
	)
	Source_Unvalidated_Invariants(
		source_unvalidated, "Encode_Dictionary_Into.source",
	)
	Dictionary_Unvalidated_Invariants(
		dictionary_unvalidated, "Encode_Dictionary_Into.dictionary",
	)
	Level_Unvalidated_Invariants(
		level_unvalidated, "Encode_Dictionary_Into.level_unvalidated",
	)
	level, level_valid := level_validate(level_unvalidated)
	if !level_valid {
		return 0, STATUS_LEVEL_INVALID
	}
	if !workspace_valid(workspace_unvalidated) {
		return 0, STATUS_STORAGE_INVALID
	}
	workspace := Workspace{
		Heads:    Hash_Positions(workspace_unvalidated.Heads),
		Previous: History_Positions(workspace_unvalidated.Previous),
	}
	if !encode_storage_valid(
		destination_unvalidated, source_unvalidated, dictionary_unvalidated,
	) {
		return 0, STATUS_STORAGE_INVALID
	}
	destination := Destination(destination_unvalidated)
	source := Source(source_unvalidated)
	dictionary := Dictionary(dictionary_unvalidated)
	writer := Bit_Writer{Destination: Bit_Storage(destination)}
	switch level {
	case NO_COMPRESSION:
		encode_stored(&writer, source)
	case HUFFMAN_ONLY:
		encode_literals(&writer, source)
	default:
		compression_level := Compression_Level(level)
		if level == DEFAULT_COMPRESSION {
			compression_level = DEFAULT_COMPRESSION_LEVEL
		}
		history := dictionary_tail(dictionary)
		encode_fixed(&writer, &workspace, source, history, compression_level)
	}
	bit_writer_finish(&writer)
	if writer.State == WRITER_STATE_EXHAUSTED {
		return Count(writer.Position), STATUS_OUTPUT_TOO_SMALL
	}
	return Count(writer.Position), STATUS_OK
}

func level_validate(
	value Level_Unvalidated,
) (level Level, valid Boolean) {
	defer func() {
		Level_Invariants(level, "level_validate.level")
		Boolean_Invariants(valid, "level_validate.valid")
	}()
	Level_Unvalidated_Invariants(value, "level_validate.value")
	if value == HUFFMAN_ONLY {
		return Level(value), true
	}
	if value == DEFAULT_COMPRESSION {
		return Level(value), true
	}
	if value < NO_COMPRESSION {
		return NO_COMPRESSION, false
	}
	if value > BEST_COMPRESSION {
		return NO_COMPRESSION, false
	}
	return Level(value), true
}

func encode_storage_valid(
	destination Destination_Unvalidated,
	source Source_Unvalidated,
	dictionary Dictionary_Unvalidated,
) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "encode_storage_valid.valid") }()
	Destination_Unvalidated_Invariants(
		destination, "encode_storage_valid.destination",
	)
	Source_Unvalidated_Invariants(source, "encode_storage_valid.source")
	Dictionary_Unvalidated_Invariants(
		dictionary, "encode_storage_valid.dictionary",
	)
	if len(destination) > BYTE_COUNT_MAXIMUM {
		return false
	}
	if len(source) > BYTE_COUNT_MAXIMUM {
		return false
	}
	if len(dictionary) > BYTE_COUNT_MAXIMUM {
		return false
	}
	if byte_slices_overlap(Bytes(destination), Bytes(source)) {
		return false
	}
	return !byte_slices_overlap(Bytes(destination), Bytes(dictionary))
}

func workspace_valid(value Workspace_Unvalidated) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "workspace_valid.valid") }()
	Workspace_Unvalidated_Invariants(value, "workspace_valid.value")
	if len(value.Heads) != HASH_COUNT {
		return false
	}
	if len(value.Previous) != WINDOW_SIZE {
		return false
	}
	return true
}

func decode_storage_valid(
	destination Destination_Unvalidated,
	compressed Compressed_Unvalidated,
	dictionary Dictionary_Unvalidated,
) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "decode_storage_valid.valid") }()
	Destination_Unvalidated_Invariants(
		destination, "decode_storage_valid.destination",
	)
	Compressed_Unvalidated_Invariants(
		compressed, "decode_storage_valid.compressed",
	)
	Dictionary_Unvalidated_Invariants(
		dictionary, "decode_storage_valid.dictionary",
	)
	if len(destination) > BYTE_COUNT_MAXIMUM {
		return false
	}
	if len(compressed) > BYTE_COUNT_MAXIMUM {
		return false
	}
	if len(dictionary) > BYTE_COUNT_MAXIMUM {
		return false
	}
	if byte_slices_overlap(Bytes(destination), Bytes(compressed)) {
		return false
	}
	return !byte_slices_overlap(Bytes(destination), Bytes(dictionary))
}

func byte_slices_overlap(first Bytes, second Bytes) (overlap Boolean) {
	defer func() { Boolean_Invariants(overlap, "byte_slices_overlap.overlap") }()
	Bytes_Invariants(first, "byte_slices_overlap.first")
	Bytes_Invariants(second, "byte_slices_overlap.second")
	// Shared/bytes owns a smaller domain than compression storage.
	if len(first) == BYTE_COUNT_MINIMUM {
		return false
	}
	if len(second) == BYTE_COUNT_MINIMUM {
		return false
	}
	first_address := uintptr(unsafe.Pointer(unsafe.SliceData(first)))
	second_address := uintptr(unsafe.Pointer(unsafe.SliceData(second)))
	if first_address < second_address {
		return Boolean(second_address-first_address < uintptr(len(first)))
	}
	return Boolean(first_address-second_address < uintptr(len(second)))
}

func dictionary_tail(dictionary Dictionary) (tail History) {
	defer func() { History_Invariants(tail, "dictionary_tail.tail") }()
	Dictionary_Invariants(dictionary, "dictionary_tail.dictionary")
	if len(dictionary) > DICTIONARY_SIZE_MAXIMUM {
		return History(dictionary[len(dictionary)-DICTIONARY_SIZE_MAXIMUM:])
	}
	return History(dictionary)
}

func encode_stored(writer *Bit_Writer, source Source) {
	Bit_Writer_Invariants(writer, "encode_stored.writer")
	Source_Invariants(source, "encode_stored.source")
	if len(source) == 0 {
		stored_block(writer, nil, true)
		return
	}
	for position := 0; position < len(source); {
		end := position + STORED_SIZE_MAXIMUM
		if end > len(source) {
			end = len(source)
		}
		stored_block(
			writer,
			Stored_Source(source[position:end]), Boolean(end == len(source)),
		)
		if writer.State == WRITER_STATE_EXHAUSTED {
			return
		}
		position = end
	}
}

func stored_block(
	writer *Bit_Writer, source Stored_Source, final Boolean,
) {
	Bit_Writer_Invariants(writer, "stored_block.writer")
	Stored_Source_Invariants(source, "stored_block.source")
	Boolean_Invariants(final, "stored_block.final")
	final_bit := Bit_Value(0)
	if final {
		final_bit = FINAL_BLOCK_BIT_VALUE
	}
	bit_writer_write_bits(writer, final_bit, FINAL_BIT_COUNT)
	bit_writer_write_bits(writer, BLOCK_KIND_STORED, BLOCK_KIND_BIT_COUNT)
	bit_writer_align(writer)
	size := uint16(len(source))
	bit_writer_write_byte(writer, Byte_Value(size))
	bit_writer_write_byte(writer, Byte_Value(size>>BYTE_BIT_COUNT))
	inverse := ^size
	bit_writer_write_byte(writer, Byte_Value(inverse))
	bit_writer_write_byte(writer, Byte_Value(inverse>>BYTE_BIT_COUNT))
	bit_writer_write_bytes(writer, source)
}

func encode_literals(writer *Bit_Writer, source Source) {
	Bit_Writer_Invariants(writer, "encode_literals.writer")
	Source_Invariants(source, "encode_literals.source")
	bit_writer_write_bits(
		writer,
		Bit_Value(BLOCK_KIND_FIXED<<FINAL_BIT_COUNT|FINAL_BLOCK_BIT_VALUE),
		BLOCK_HEADER_BIT_COUNT,
	)
	for _, value := range source {
		write_fixed_symbol(writer, Fixed_Symbol(value))
		if writer.State == WRITER_STATE_EXHAUSTED {
			return
		}
	}
	write_fixed_symbol(writer, LITERAL_SYMBOL_END)
}

func encode_fixed(
	writer *Bit_Writer,
	workspace *Workspace,
	source Source,
	dictionary History,
	level Compression_Level,
) {
	Bit_Writer_Invariants(writer, "encode_fixed.writer")
	Workspace_Invariants(workspace, "encode_fixed.workspace")
	Source_Invariants(source, "encode_fixed.source")
	History_Invariants(dictionary, "encode_fixed.dictionary")
	Compression_Level_Invariants(level, "encode_fixed.level")
	if level == BEST_SPEED {
		if len(dictionary) == HISTORY_SIZE_MINIMUM {
			if len(source) >= MATCH_SIZE_MINIMUM {
				encode_fixed_best_speed(writer, workspace.Heads, source)
				return
			}
		}
	}
	clear(workspace.Heads[:])
	// Each reachable previous link is overwritten before its head publishes it.
	bit_writer_write_bits(
		writer,
		Bit_Value(BLOCK_KIND_FIXED<<FINAL_BIT_COUNT|FINAL_BLOCK_BIT_VALUE),
		BLOCK_HEADER_BIT_COUNT,
	)
	workspace_seed(workspace, dictionary)
	for source_position := 0; source_position < len(source); {
		if writer.State == WRITER_STATE_EXHAUSTED {
			break
		}
		virtual_position := Sequence_Position(len(dictionary) + source_position)
		match := match_search(
			workspace, dictionary, Search_Source(source), virtual_position, level,
		)
		if !match.Present {
			write_fixed_symbol(writer, Fixed_Symbol(source[source_position]))
			workspace_insert(workspace, dictionary, source, virtual_position)
			source_position++
			continue
		}
		match_symbol, match_extra, match_suffix_count := match_code(
			match.Size,
		)
		write_fixed_symbol(writer, Fixed_Symbol(match_symbol))
		bit_writer_write_bits(
			writer,
			Bit_Value(match_extra), Bit_Count(match_suffix_count),
		)
		distance_symbol, distance_extra, distance_extra_count := distance_code(
			Distance(int(virtual_position) - int(match.Position)),
		)
		write_fixed_distance(writer, distance_symbol)
		bit_writer_write_bits(
			writer,
			Bit_Value(distance_extra), Bit_Count(distance_extra_count),
		)
		for inserted_index := 0; inserted_index < int(match.Size); inserted_index++ {
			workspace_insert(
				workspace, dictionary, source,
				virtual_position+Sequence_Position(inserted_index),
			)
		}
		source_position += int(match.Size)
	}
	write_fixed_symbol(writer, LITERAL_SYMBOL_END)
}

// Keeps already-validated values inside one loop because repeated generic assertion boundaries
// survive an inert build and hide codec cost.
func encode_fixed_best_speed(
	writer *Bit_Writer, heads Hash_Positions, source Source,
) {
	Bit_Writer_Invariants(writer, "encode_fixed_best_speed.writer")
	Hash_Positions_Invariants(heads, "encode_fixed_best_speed.heads")
	Source_Invariants(source, "encode_fixed_best_speed.source")
	if writer.State == WRITER_STATE_EXHAUSTED {
		return
	}
	bit_writer_write_bits(
		writer,
		Bit_Value(BLOCK_KIND_FIXED<<FINAL_BIT_COUNT|FINAL_BLOCK_BIT_VALUE),
		BLOCK_HEADER_BIT_COUNT,
	)
	position := BYTE_COUNT_MINIMUM
	for position+MATCH_SIZE_MINIMUM <= len(source) {
		hash := sequence_hash(
			nil, source, Sequence_Position(position),
		)
		candidate_link := heads[hash]
		heads[hash] = int32(position + DISTANCE_MINIMUM)
		if candidate_link != POSITION_MINIMUM {
			candidate := int(candidate_link) - DISTANCE_MINIMUM
			distance := position - candidate
			// A prior call may leave a position. A proven earlier coordinate
			// names bytes in this call. Full comparison makes it authoritative.
			if candidate >= POSITION_MINIMUM {
				if candidate < position {
					if distance <= WINDOW_SIZE {
						maximum := len(source) - position
						if maximum > MATCH_SIZE_MAXIMUM {
							maximum = MATCH_SIZE_MAXIMUM
						}
						size := matching_size(
							nil, source, Match_Position(candidate),
							Later_Sequence_Position(position),
							Match_Size(maximum),
						)
						if size >= MATCH_SIZE_MINIMUM {
							write_fixed_match(
								writer, Match_Size(size),
								Distance(distance),
							)
							position += int(size)
							continue
						}
					}
				}
			}
		}
		write_fixed_symbol(writer, Fixed_Symbol(source[position]))
		if writer.State == WRITER_STATE_EXHAUSTED {
			return
		}
		position++
	}
	for position < len(source) {
		write_fixed_symbol(writer, Fixed_Symbol(source[position]))
		if writer.State == WRITER_STATE_EXHAUSTED {
			return
		}
		position++
	}
	write_fixed_symbol(writer, LITERAL_SYMBOL_END)
}

// Writes one already-proven match without re-entering assertion wrappers for each code field.
func write_fixed_match(writer *Bit_Writer, size Match_Size, distance Distance) {
	Bit_Writer_Invariants(writer, "write_fixed_match.writer")
	Match_Size_Invariants(size, "write_fixed_match.size")
	Distance_Invariants(distance, "write_fixed_match.distance")
	match_symbol := Match_Symbol(MATCH_SYMBOL_MINIMUM)
	match_extra := Match_Extra_Value(MATCH_EXTRA_VALUE_MINIMUM)
	match_extra_count := Match_Extra_Bit_Count(BIT_COUNT_MINIMUM)
	if size <= MATCH_DIRECT_SIZE_MAXIMUM {
		match_symbol += Match_Symbol(size - MATCH_SIZE_MINIMUM)
	} else if size == MATCH_SIZE_MAXIMUM {
		match_symbol = MATCH_SYMBOL_MAXIMUM
	} else {
		match_extra_count = Match_Extra_Bit_Count(GROUPED_EXTRA_BIT_COUNT_MINIMUM)
		for size >= Match_Size(
			MATCH_SIZE_MINIMUM+
				1<<(match_extra_count+MATCH_EXTRA_BASE_SHIFT+FINAL_BIT_COUNT),
		) {
			match_extra_count++
		}
		group_base := Match_Size(
			MATCH_SIZE_MINIMUM + 1<<(match_extra_count+MATCH_EXTRA_BASE_SHIFT),
		)
		group_position := (size - group_base) >> match_extra_count
		group_count := match_extra_count -
			Match_Extra_Bit_Count(GROUPED_EXTRA_BIT_COUNT_MINIMUM)
		match_symbol = MATCH_EXTRA_SYMBOL_MINIMUM + Match_Symbol(
			group_count*MATCH_EXTRA_SYMBOL_GROUP_SIZE,
		) + Match_Symbol(group_position)
		match_base := group_base + Match_Size(group_position<<match_extra_count)
		match_extra = Match_Extra_Value(size - match_base)
	}
	write_fixed_symbol(writer, Fixed_Symbol(match_symbol))
	bit_writer_write_bits(
		writer, Bit_Value(match_extra), Bit_Count(match_extra_count),
	)
	distance_symbol := Distance_Symbol(distance - DISTANCE_MINIMUM)
	distance_extra := Distance_Extra_Value(DISTANCE_EXTRA_VALUE_MINIMUM)
	distance_extra_count := Distance_Extra_Bit_Count(BIT_COUNT_MINIMUM)
	if distance >= DISTANCE_DIRECT_SYMBOL_COUNT+DISTANCE_MINIMUM {
		distance_extra_count = Distance_Extra_Bit_Count(GROUPED_EXTRA_BIT_COUNT_MINIMUM)
		for distance > Distance(1<<(distance_extra_count+BINARY_RADIX)) {
			distance_extra_count++
		}
		distance_group_base := Distance(1<<(distance_extra_count+1) + DISTANCE_MINIMUM)
		distance_group_position := (distance - distance_group_base) >> distance_extra_count
		distance_symbol = Distance_Symbol(
			distance_extra_count*BINARY_RADIX+DISTANCE_EXTRA_SYMBOL_OFFSET,
		) + Distance_Symbol(distance_group_position)
		distance_base := distance_group_base +
			Distance(distance_group_position<<distance_extra_count)
		distance_extra = Distance_Extra_Value(distance - distance_base)
	}
	write_fixed_distance(writer, distance_symbol)
	bit_writer_write_bits(
		writer, Bit_Value(distance_extra), Bit_Count(distance_extra_count),
	)
}

func workspace_seed(workspace *Workspace, dictionary History) {
	Workspace_Invariants(workspace, "workspace_seed.workspace")
	History_Invariants(dictionary, "workspace_seed.dictionary")
	position := Sequence_Position(0)
	for int(position)+MATCH_SIZE_MINIMUM <= len(dictionary) {
		workspace_insert(workspace, dictionary, nil, position)
		position++
	}
}

func workspace_insert(
	workspace *Workspace,
	dictionary History,
	source Source,
	position Sequence_Position,
) {
	Workspace_Invariants(workspace, "workspace_insert.workspace")
	History_Invariants(dictionary, "workspace_insert.dictionary")
	Source_Invariants(source, "workspace_insert.source")
	Sequence_Position_Invariants(position, "workspace_insert.position")
	if int(position)+MATCH_SIZE_MINIMUM > len(dictionary)+len(source) {
		return
	}
	hash := sequence_hash(dictionary, Source(source), position)
	workspace.Previous[int(position)&WINDOW_MASK] = workspace.Heads[hash]
	workspace.Heads[hash] = int32(position + 1)
}

func match_search(
	workspace *Workspace,
	dictionary History,
	source Search_Source,
	position Sequence_Position,
	level Compression_Level,
) (match Match) {
	defer func() { Match_Invariants(match, "match_search.match") }()
	Workspace_Invariants(workspace, "match_search.workspace")
	History_Invariants(dictionary, "match_search.dictionary")
	Search_Source_Invariants(source, "match_search.source")
	Sequence_Position_Invariants(position, "match_search.position")
	Compression_Level_Invariants(level, "match_search.level")
	match.Size = MATCH_SIZE_MINIMUM
	if int(position)+MATCH_SIZE_MINIMUM > len(dictionary)+len(source) {
		return match
	}
	hash := sequence_hash(dictionary, Source(source), position)
	candidate_link := workspace.Heads[hash]
	limit := chain_limit(level)
	maximum := Match_Size(len(dictionary) + len(source) - int(position))
	if maximum > MATCH_SIZE_MAXIMUM {
		maximum = MATCH_SIZE_MAXIMUM
	}
	for searched_count := 0; candidate_link != 0; searched_count++ {
		if searched_count >= int(limit) {
			break
		}
		candidate := Match_Position(candidate_link - 1)
		distance := int(position) - int(candidate)
		if distance <= 0 {
			break
		}
		if distance > WINDOW_SIZE {
			break
		}
		size := matching_size(
			dictionary, Source(source), candidate,
			Later_Sequence_Position(position), maximum,
		)
		if size < MATCH_SIZE_MINIMUM {
			candidate_link = workspace.Previous[int(candidate)&WINDOW_MASK]
			continue
		}
		if match.Present {
			if size <= Matching_Size(match.Size) {
				candidate_link = workspace.Previous[int(candidate)&WINDOW_MASK]
				continue
			}
		}
		match.Position = candidate
		match.Size = Match_Size(size)
		match.Present = true
		if size == Matching_Size(maximum) {
			break
		}
		candidate_link = workspace.Previous[int(candidate)&WINDOW_MASK]
	}
	return match
}

func chain_limit(level Compression_Level) (count Chain_Count) {
	defer func() { Chain_Count_Invariants(count, "chain_limit.count") }()
	Compression_Level_Invariants(level, "chain_limit.level")
	switch level {
	case BEST_SPEED:
		return CHAIN_COUNT_MINIMUM
	case COMPRESSION_LEVEL_2:
		return CHAIN_COUNT_LEVEL_2
	case COMPRESSION_LEVEL_3:
		return CHAIN_COUNT_LEVEL_3
	case COMPRESSION_LEVEL_4:
		return CHAIN_COUNT_LEVEL_4
	case COMPRESSION_LEVEL_5:
		return CHAIN_COUNT_LEVEL_5
	case COMPRESSION_LEVEL_6:
		return CHAIN_COUNT_LEVEL_6
	case COMPRESSION_LEVEL_7:
		return CHAIN_COUNT_LEVEL_7
	case COMPRESSION_LEVEL_8:
		return CHAIN_COUNT_LEVEL_8
	default:
		return CHAIN_COUNT_MAXIMUM
	}
}

func matching_size(
	dictionary History,
	source Source,
	first Match_Position,
	second Later_Sequence_Position,
	maximum Match_Size,
) (count Matching_Size) {
	defer func() { Matching_Size_Invariants(count, "matching_size.count") }()
	History_Invariants(dictionary, "matching_size.dictionary")
	Source_Invariants(source, "matching_size.source")
	Match_Position_Invariants(first, "matching_size.first")
	Later_Sequence_Position_Invariants(second, "matching_size.second")
	Match_Size_Invariants(maximum, "matching_size.maximum")
	maximum_count := Matching_Size(maximum)
	if len(dictionary) == HISTORY_SIZE_MINIMUM {
		first_position := int(first)
		second_position := int(second)
		maximum_size := int(maximum)
		if string(source[first_position:first_position+maximum_size]) ==
			string(source[second_position:second_position+maximum_size]) {
			return maximum_count
		}
		for count < maximum_count {
			first_value := source[first_position+int(count)]
			second_value := source[second_position+int(count)]
			if first_value != second_value {
				return count
			}
			count++
		}
		return count
	}
	for index := Matching_Size(MATCHING_SIZE_MINIMUM); index < maximum_count; index++ {
		first_position := Virtual_Byte_Position(first) + Virtual_Byte_Position(index)
		second_position := Virtual_Byte_Position(second) + Virtual_Byte_Position(index)
		if history_byte(dictionary, source, first_position) != history_byte(
			dictionary, source, second_position,
		) {
			return index
		}
	}
	return maximum_count
}

func history_byte(
	dictionary History, source Source, position Virtual_Byte_Position,
) (value Byte_Value) {
	defer func() { Byte_Value_Invariants(value, "history_byte.value") }()
	History_Invariants(dictionary, "history_byte.dictionary")
	Source_Invariants(source, "history_byte.source")
	Virtual_Byte_Position_Invariants(position, "history_byte.position")
	if int(position) < len(dictionary) {
		return Byte_Value(dictionary[position])
	}
	return Byte_Value(source[int(position)-len(dictionary)])
}

func sequence_hash(
	dictionary History, source Source, position Sequence_Position,
) (hash Hash) {
	defer func() { Hash_Invariants(hash, "sequence_hash.hash") }()
	History_Invariants(dictionary, "sequence_hash.dictionary")
	Source_Invariants(source, "sequence_hash.source")
	Sequence_Position_Invariants(position, "sequence_hash.position")
	value := bits.WORD_32_MINIMUM
	if len(dictionary) == HISTORY_SIZE_MINIMUM {
		source_position := int(position)
		for index := BYTE_COUNT_MINIMUM; index < HASH_INPUT_BYTE_COUNT; index++ {
			value <<= BYTE_BIT_COUNT
			value |= uint32(source[source_position+index])
		}
		return Hash(
			int((value*HASH_MULTIPLIER)>>(HASH_PRODUCT_BIT_COUNT-HASH_BIT_COUNT)) &
				HASH_MASK,
		)
	}
	for index := BYTE_COUNT_MINIMUM; index < HASH_INPUT_BYTE_COUNT; index++ {
		value <<= BYTE_BIT_COUNT
		value |= uint32(history_byte(
			dictionary, source,
			Virtual_Byte_Position(int(position)+index),
		))
	}
	return Hash(
		int((value*HASH_MULTIPLIER)>>(HASH_PRODUCT_BIT_COUNT-HASH_BIT_COUNT)) & HASH_MASK,
	)
}

func match_code(
	size Match_Size,
) (
	symbol Match_Symbol,
	extra Match_Extra_Value,
	extra_count Match_Extra_Bit_Count,
) {
	defer func() {
		Match_Symbol_Invariants(symbol, "match_code.symbol")
		Match_Extra_Value_Invariants(extra, "match_code.extra")
		Match_Extra_Bit_Count_Invariants(extra_count, "match_code.extra_count")
	}()
	Match_Size_Invariants(size, "match_code.size")
	minimum := Match_Symbol(MATCH_SYMBOL_MINIMUM)
	for candidate := minimum; candidate < MATCH_SYMBOL_MAXIMUM; candidate++ {
		base, candidate_extra_count := match_code_values(candidate)
		next_base, _ := match_code_values(candidate + 1)
		if size < next_base {
			return candidate, Match_Extra_Value(size - base), candidate_extra_count
		}
	}
	base, extra_count := match_code_values(MATCH_SYMBOL_MAXIMUM)
	return MATCH_SYMBOL_MAXIMUM, Match_Extra_Value(size - base), extra_count
}

func distance_code(
	distance Distance,
) (
	symbol Distance_Symbol,
	extra Distance_Extra_Value,
	extra_count Distance_Extra_Bit_Count,
) {
	defer func() {
		Distance_Symbol_Invariants(symbol, "distance_code.symbol")
		Distance_Extra_Value_Invariants(extra, "distance_code.extra")
		Distance_Extra_Bit_Count_Invariants(
			extra_count, "distance_code.extra_count",
		)
	}()
	Distance_Invariants(distance, "distance_code.distance")
	symbol = Distance_Symbol(DISTANCE_SYMBOL_MINIMUM)
	for ; symbol < DISTANCE_SYMBOL_MAXIMUM; symbol++ {
		base, candidate_extra_count := distance_code_values(symbol)
		next_base, _ := distance_code_values(symbol + 1)
		if distance < Distance(next_base) {
			return symbol, Distance_Extra_Value(
				distance - Distance(base),
			), candidate_extra_count
		}
	}
	base, extra_count := distance_code_values(DISTANCE_SYMBOL_MAXIMUM)
	return DISTANCE_SYMBOL_MAXIMUM,
		Distance_Extra_Value(distance - Distance(base)), extra_count
}

func write_fixed_symbol(writer *Bit_Writer, symbol Fixed_Symbol) {
	Bit_Writer_Invariants(writer, "fixed_symbol_write.writer")
	Fixed_Symbol_Invariants(symbol, "fixed_symbol_write.symbol")
	var code Fixed_Symbol
	var size Bit_Count
	switch {
	case symbol <= FIXED_LITERAL_FIRST_MAXIMUM:
		code = FIXED_FIRST_CODE_BASE + symbol
		size = FIXED_CODE_SIZE
	case symbol <= FIXED_LITERAL_SECOND_MAXIMUM:
		code = FIXED_SECOND_CODE_BASE + symbol - FIXED_LITERAL_SECOND_MINIMUM
		size = FIXED_SECOND_CODE_SIZE
	case symbol <= FIXED_LITERAL_THIRD_MAXIMUM:
		code = symbol - FIXED_LITERAL_THIRD_MINIMUM
		size = FIXED_THIRD_CODE_SIZE
	default:
		code = FIXED_FOURTH_CODE_BASE + symbol - FIXED_LITERAL_FOURTH_MINIMUM
		size = FIXED_CODE_SIZE
	}
	bit_writer_write_bits(writer, reverse_low_bits(Bit_Value(code), size), size)
}

func write_fixed_distance(writer *Bit_Writer, symbol Distance_Symbol) {
	Bit_Writer_Invariants(writer, "fixed_distance_write.writer")
	Distance_Symbol_Invariants(symbol, "fixed_distance_write.symbol")
	bit_writer_write_bits(
		writer,
		reverse_low_bits(Bit_Value(symbol), FIXED_DISTANCE_BIT_COUNT),
		FIXED_DISTANCE_BIT_COUNT,
	)
}

func reverse_low_bits(value Bit_Value, count Bit_Count) (result Bit_Value) {
	defer func() { Bit_Value_Invariants(result, "reverse_low_bits.result") }()
	Bit_Value_Invariants(value, "reverse_low_bits.value")
	Bit_Count_Invariants(count, "reverse_low_bits.count")
	result = 0
	for index := Bit_Count(0); index < count; index++ {
		result = result<<FINAL_BIT_COUNT | value&LOW_BIT_MASK
		value >>= FINAL_BIT_COUNT
	}
	return result
}

func bit_writer_write_bits(
	writer *Bit_Writer, value Bit_Value, count Bit_Count,
) {
	Bit_Writer_Invariants(writer, "bit_writer_write.writer")
	Bit_Value_Invariants(value, "bit_writer_write.value")
	Bit_Count_Invariants(count, "bit_writer_write.count")
	if writer.State == WRITER_STATE_EXHAUSTED {
		return
	}
	writer.Bits |= Pending_Bits(uint64(value) << writer.Bits_Count)
	writer.Bits_Count += Pending_Bit_Count(count)
	for writer.Bits_Count >= BYTE_BIT_COUNT {
		if int(writer.Position) == len(writer.Destination) {
			writer.Bits = 0
			writer.Bits_Count = 0
			writer.State = WRITER_STATE_EXHAUSTED
			return
		}
		writer.Destination[writer.Position] = byte(writer.Bits)
		writer.Position++
		writer.Bits >>= BYTE_BIT_COUNT
		writer.Bits_Count -= BYTE_BIT_COUNT
	}
}

func bit_writer_align(writer *Bit_Writer) {
	Bit_Writer_Invariants(writer, "bit_writer_align.writer")
	if writer.Bits_Count > 0 {
		bit_writer_write_byte(writer, Byte_Value(writer.Bits))
		writer.Bits = 0
		writer.Bits_Count = 0
	}
}

func bit_writer_finish(writer *Bit_Writer) {
	Bit_Writer_Invariants(writer, "bit_writer_finish.writer")
	bit_writer_align(writer)
}

func bit_writer_write_byte(writer *Bit_Writer, value Byte_Value) {
	Bit_Writer_Invariants(writer, "bit_writer_byte.writer")
	Byte_Value_Invariants(value, "bit_writer_byte.value")
	if writer.State == WRITER_STATE_EXHAUSTED {
		return
	}
	if int(writer.Position) == len(writer.Destination) {
		writer.State = WRITER_STATE_EXHAUSTED
		return
	}
	writer.Destination[writer.Position] = byte(value)
	writer.Position++
}

func bit_writer_write_bytes(writer *Bit_Writer, source Stored_Source) {
	Bit_Writer_Invariants(writer, "bit_writer_bytes.writer")
	Stored_Source_Invariants(source, "bit_writer_bytes.source")
	if writer.State == WRITER_STATE_EXHAUSTED {
		return
	}
	available := len(writer.Destination) - int(writer.Position)
	if available < len(source) {
		copy(writer.Destination[writer.Position:], source[:available])
		writer.Position += Byte_Position(available)
		writer.State = WRITER_STATE_EXHAUSTED
		return
	}
	copy(writer.Destination[writer.Position:], source)
	writer.Position += Byte_Position(len(source))
}

// Decode_Prefix_Into reports consumption so framing layers retain their footer.
func Decode_Prefix_Into(
	destination_unvalidated Destination_Unvalidated,
	compressed_unvalidated Compressed_Unvalidated,
) (
	count Count,
	compressed_count Count,
	status Decode_Status,
) {
	defer func() {
		Count_Invariants(count, "Decode_Prefix_Into.count")
		Count_Invariants(
			compressed_count, "Decode_Prefix_Into.compressed_count",
		)
		Decode_Status_Invariants(status, "Decode_Prefix_Into.status")
	}()
	Destination_Unvalidated_Invariants(
		destination_unvalidated, "Decode_Prefix_Into.destination",
	)
	Compressed_Unvalidated_Invariants(
		compressed_unvalidated, "Decode_Prefix_Into.compressed",
	)
	if !decode_storage_valid(
		destination_unvalidated, compressed_unvalidated, nil,
	) {
		return 0, 0, STATUS_STORAGE_INVALID
	}
	count, compressed_count, block_status := decode_prefix(
		Destination(destination_unvalidated),
		Compressed(compressed_unvalidated), nil,
	)
	return count, compressed_count, Decode_Status(block_status)
}

// Decode_Into reads one raw DEFLATE stream without preset history.
func Decode_Into(
	destination_unvalidated Destination_Unvalidated,
	compressed_unvalidated Compressed_Unvalidated,
) (count Count, status Decode_Status) {
	defer func() {
		Count_Invariants(count, "Decode_Into.count")
		Decode_Status_Invariants(status, "Decode_Into.status")
	}()
	Destination_Unvalidated_Invariants(destination_unvalidated, "Decode_Into.destination")
	Compressed_Unvalidated_Invariants(compressed_unvalidated, "Decode_Into.compressed")
	return Decode_Dictionary_Into(
		destination_unvalidated, compressed_unvalidated, nil,
	)
}

// Decode_Dictionary_Into reads one raw DEFLATE stream with preset history.
func Decode_Dictionary_Into(
	destination_unvalidated Destination_Unvalidated,
	compressed_unvalidated Compressed_Unvalidated,
	dictionary_unvalidated Dictionary_Unvalidated,
) (count Count, status Decode_Status) {
	defer func() {
		Count_Invariants(count, "Decode_Dictionary_Into.count")
		Decode_Status_Invariants(status, "Decode_Dictionary_Into.status")
	}()
	Destination_Unvalidated_Invariants(
		destination_unvalidated, "Decode_Dictionary_Into.destination",
	)
	Compressed_Unvalidated_Invariants(
		compressed_unvalidated, "Decode_Dictionary_Into.compressed",
	)
	Dictionary_Unvalidated_Invariants(
		dictionary_unvalidated, "Decode_Dictionary_Into.dictionary",
	)
	if len(dictionary_unvalidated) == 0 {
		prefix_count, compressed_count, prefix_status := Decode_Prefix_Into(
			destination_unvalidated, compressed_unvalidated,
		)
		if prefix_status != STATUS_OK {
			return prefix_count, prefix_status
		}
		if int(compressed_count) != len(compressed_unvalidated) {
			return prefix_count, STATUS_INPUT_INVALID
		}
		return prefix_count, STATUS_OK
	}
	if !decode_storage_valid(
		destination_unvalidated, compressed_unvalidated, dictionary_unvalidated,
	) {
		return 0, STATUS_STORAGE_INVALID
	}
	destination := Destination(destination_unvalidated)
	compressed := Compressed(compressed_unvalidated)
	dictionary := Dictionary(dictionary_unvalidated)
	count, compressed_count, block_status := decode_prefix(
		destination, compressed, dictionary,
	)
	if block_status != STATUS_OK {
		return count, Decode_Status(block_status)
	}
	if int(compressed_count) != len(compressed) {
		return count, STATUS_INPUT_INVALID
	}
	return count, STATUS_OK
}

func decode_prefix(
	destination Destination,
	compressed Compressed,
	dictionary Dictionary,
) (
	count Count,
	compressed_count Count,
	status Block_Status,
) {
	defer func() {
		Count_Invariants(count, "decode_prefix.count")
		Count_Invariants(compressed_count, "decode_prefix.compressed_count")
		Block_Status_Invariants(status, "decode_prefix.status")
	}()
	Destination_Invariants(destination, "decode_prefix.destination")
	Compressed_Invariants(compressed, "decode_prefix.compressed")
	Dictionary_Invariants(dictionary, "decode_prefix.dictionary")
	if len(compressed) == 0 {
		return 0, 0, STATUS_INPUT_INVALID
	}
	history := dictionary_tail(dictionary)
	reader := Bit_Reader{Source: Bit_Storage(compressed)}
	final := Bit_Value(0)
	for final == 0 {
		var available Boolean
		final, available = bit_reader_read(&reader, FINAL_BIT_COUNT)
		if !available {
			return count, Count(reader.Position), STATUS_INPUT_INVALID
		}
		block_kind, available := bit_reader_read(&reader, BLOCK_KIND_BIT_COUNT)
		if !available {
			return count, Count(reader.Position), STATUS_INPUT_INVALID
		}
		next_count, block_status := decode_block(
			&reader,
			destination, history, count, Block_Kind(block_kind),
		)
		count = next_count
		if block_status != STATUS_OK {
			return count, Count(reader.Position), block_status
		}
	}
	return count, Count(reader.Position), STATUS_OK
}

func decode_block(
	reader *Bit_Reader,
	destination Destination, dictionary History, count Count,
	block_kind Block_Kind,
) (next_count Count, status Block_Status) {
	defer func() {
		Count_Invariants(next_count, "decode_block.next_count")
		Block_Status_Invariants(status, "decode_block.status")
	}()
	Bit_Reader_Invariants(reader, "decode_block.reader")
	Destination_Invariants(destination, "decode_block.destination")
	History_Invariants(dictionary, "decode_block.dictionary")
	Count_Invariants(count, "decode_block.count")
	Block_Kind_Invariants(block_kind, "decode_block.block_kind")
	switch block_kind {
	case BLOCK_KIND_STORED:
		return decode_stored(reader, destination, count)
	case BLOCK_KIND_FIXED:
		var literal_decoder Huffman_Decoder
		var distance_decoder Huffman_Decoder
		return decode_huffman(
			reader, destination, dictionary, count,
			&literal_decoder, &distance_decoder, true,
		)
	case BLOCK_KIND_DYNAMIC:
		var literal_decoder Huffman_Decoder
		var distance_decoder Huffman_Decoder
		valid := dynamic_decoders(
			reader, &literal_decoder, &distance_decoder,
		)
		if !valid {
			return count, STATUS_INPUT_INVALID
		}
		return decode_huffman(
			reader,
			destination, dictionary, count, &literal_decoder, &distance_decoder, false,
		)
	default:
		return count, STATUS_INPUT_INVALID
	}
}

func decode_stored(
	reader *Bit_Reader,
	destination Destination, count Count,
) (next_count Count, status Block_Status) {
	defer func() {
		Count_Invariants(next_count, "decode_stored.next_count")
		Block_Status_Invariants(status, "decode_stored.status")
	}()
	Bit_Reader_Invariants(reader, "decode_stored.reader")
	Destination_Invariants(destination, "decode_stored.destination")
	Count_Invariants(count, "decode_stored.count")
	bit_reader_align(reader)
	size, available := bit_reader_read(reader, STORED_SIZE_BIT_COUNT)
	if !available {
		return count, STATUS_INPUT_INVALID
	}
	inverse, available := bit_reader_read(reader, STORED_SIZE_BIT_COUNT)
	if !available {
		return count, STATUS_INPUT_INVALID
	}
	if uint16(inverse) != ^uint16(size) {
		return count, STATUS_INPUT_INVALID
	}
	stored_count := int(size)
	remaining_source := len(reader.Source) - int(reader.Position)
	if remaining_source < stored_count {
		return count, STATUS_INPUT_INVALID
	}
	remaining_destination := len(destination) - int(count)
	copied_count := stored_count
	if remaining_destination < copied_count {
		copied_count = remaining_destination
	}
	copy(
		destination[int(count):int(count)+copied_count],
		reader.Source[int(reader.Position):int(reader.Position)+copied_count],
	)
	reader.Position += Byte_Position(copied_count)
	count += Count(copied_count)
	if copied_count != stored_count {
		return count, STATUS_OUTPUT_TOO_SMALL
	}
	return count, STATUS_OK
}

func dynamic_decoders(
	reader *Bit_Reader,
	literal_decoder *Huffman_Decoder,
	distance_decoder *Huffman_Decoder,
) (
	valid Boolean,
) {
	defer func() { Boolean_Invariants(valid, "dynamic_decoders.valid") }()
	Bit_Reader_Invariants(reader, "dynamic_decoders.reader")
	Huffman_Decoder_Invariants(
		literal_decoder, "dynamic_decoders.literal_decoder",
	)
	Huffman_Decoder_Invariants(
		distance_decoder, "dynamic_decoders.distance_decoder",
	)
	literal_bits, available := bit_reader_read(reader, DYNAMIC_ALPHABET_COUNT_BIT_COUNT)
	if !available {
		return false
	}
	distance_bits, available := bit_reader_read(reader, DYNAMIC_ALPHABET_COUNT_BIT_COUNT)
	if !available {
		return false
	}
	code_bits, available := bit_reader_read(reader, DYNAMIC_CODE_COUNT_BIT_COUNT)
	if !available {
		return false
	}
	literal_count := Literal_Count(literal_bits + LITERAL_COUNT_MINIMUM)
	distance_count := Distance_Count(distance_bits + DISTANCE_COUNT_MINIMUM)
	if literal_count > LITERAL_COUNT_MAXIMUM {
		return false
	}
	if distance_count > DISTANCE_SYMBOL_COUNT {
		return false
	}
	var code_sizes [CODE_COUNT]uint8
	if !dynamic_code_sizes(
		reader,
		Code_Size_Alphabet(code_sizes[:]),
		Encoded_Code_Count(code_bits+ENCODED_CODE_COUNT_MINIMUM),
	) {
		return false
	}
	var code_decoder Huffman_Decoder
	if !huffman_build(&code_decoder, Code_Sizes(code_sizes[:])) {
		return false
	}
	if code_decoder.Symbol_Count == 0 {
		return false
	}
	var sizes [LITERAL_COUNT_MAXIMUM + DISTANCE_SYMBOL_COUNT]uint8
	total_count := int(literal_count) + int(distance_count)
	if !dynamic_sizes(
		reader,
		&code_decoder, Dynamic_Sizes(sizes[:total_count]),
	) {
		return false
	}
	if sizes[LITERAL_SYMBOL_END] == 0 {
		return false
	}
	if !huffman_build(literal_decoder, Code_Sizes(sizes[:int(literal_count)])) {
		return false
	}
	valid = huffman_build(
		distance_decoder,
		Code_Sizes(sizes[int(literal_count):total_count]),
	)
	return valid
}

func dynamic_code_sizes(
	reader *Bit_Reader,
	sizes Code_Size_Alphabet,
	encoded_count Encoded_Code_Count,
) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "dynamic_code_sizes.valid") }()
	Bit_Reader_Invariants(reader, "dynamic_code_sizes.reader")
	Code_Size_Alphabet_Invariants(sizes, "dynamic_code_sizes.sizes")
	Encoded_Code_Count_Invariants(
		encoded_count, "dynamic_code_sizes.encoded_count",
	)
	var order [CODE_COUNT]uint8
	for index := 0; index < REPEAT_SYMBOL_COUNT; index++ {
		order[index] = uint8(REPEAT_SYMBOL_MINIMUM + index)
	}
	order[REPEAT_SYMBOL_COUNT] = BIT_COUNT_MINIMUM
	for code_size := FIXED_CODE_SIZE; code_size < CODE_SIZE_MAXIMUM; code_size++ {
		position := CODE_SIZE_ORDER_PREFIX_COUNT +
			(code_size-FIXED_CODE_SIZE)*BINARY_RADIX
		order[position] = uint8(code_size)
		order[position+FINAL_BIT_COUNT] = uint8(CODE_SIZE_MAXIMUM - code_size)
	}
	order[CODE_COUNT-CODE_SIZE_ORDER_SUFFIX_COUNT] = CODE_SIZE_MAXIMUM
	for position_index := 0; position_index < int(encoded_count); position_index++ {
		value, available := bit_reader_read(reader, CODE_SIZE_BIT_COUNT)
		if !available {
			return false
		}
		sizes[order[position_index]] = uint8(value)
	}
	return true
}

func dynamic_sizes(
	reader *Bit_Reader,
	decoder *Huffman_Decoder, sizes Dynamic_Sizes,
) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "dynamic_sizes.valid") }()
	Bit_Reader_Invariants(reader, "dynamic_sizes.reader")
	Huffman_Decoder_Invariants(decoder, "dynamic_sizes.decoder")
	Dynamic_Sizes_Invariants(sizes, "dynamic_sizes.sizes")
	for cursor := Dynamic_Cursor(0); int(cursor) < len(sizes); {
		position := Dynamic_Position(cursor)
		symbol, available := huffman_read(reader, decoder)
		if !available {
			return false
		}
		if symbol < REPEAT_SYMBOL_MINIMUM {
			sizes[position] = uint8(symbol)
			cursor++
			continue
		}
		if symbol > REPEAT_SYMBOL_MAXIMUM {
			return false
		}
		next, repeated_valid := repeated_size(
			reader,
			sizes, position, Repeat_Symbol(symbol),
		)
		if !repeated_valid {
			return false
		}
		cursor = next
	}
	return true
}

func repeated_size(
	reader *Bit_Reader,
	sizes Dynamic_Sizes,
	position Dynamic_Position,
	symbol Repeat_Symbol,
) (next Dynamic_Cursor, valid Boolean) {
	defer func() {
		Dynamic_Cursor_Invariants(next, "repeated_size.next")
		Boolean_Invariants(valid, "repeated_size.valid")
	}()
	Bit_Reader_Invariants(reader, "repeated_size.reader")
	Dynamic_Sizes_Invariants(sizes, "repeated_size.sizes")
	Dynamic_Position_Invariants(position, "repeated_size.position")
	Repeat_Symbol_Invariants(symbol, "repeated_size.symbol")
	var value uint8
	var repeat_base Repeat_Count
	var extra_count Bit_Count
	switch symbol {
	case REPEAT_SYMBOL_MINIMUM:
		if position == 0 {
			return Dynamic_Cursor(position), false
		}
		value = sizes[position-1]
		repeat_base = REPEAT_PREVIOUS_BASE
		extra_count = REPEAT_PREVIOUS_EXTRA_BIT_COUNT
	case REPEAT_SYMBOL_MIDDLE:
		repeat_base = REPEAT_ZERO_SHORT_BASE
		extra_count = REPEAT_ZERO_SHORT_EXTRA_BIT_COUNT
	case REPEAT_SYMBOL_MAXIMUM:
		repeat_base = REPEAT_ZERO_LONG_BASE
		extra_count = REPEAT_ZERO_LONG_EXTRA_BIT_COUNT
	default:
		return Dynamic_Cursor(position), false
	}
	extra, available := bit_reader_read(reader, extra_count)
	if !available {
		return Dynamic_Cursor(position), false
	}
	repeat_count := repeat_base + Repeat_Count(extra)
	Repeat_Count_Invariants(repeat_count, "repeated_size.repeat_count")
	end := int(position) + int(repeat_count)
	if end > len(sizes) {
		return Dynamic_Cursor(position), false
	}
	for repeated_index := 0; repeated_index < int(repeat_count); repeated_index++ {
		sizes[position+Dynamic_Position(repeated_index)] = value
	}
	return Dynamic_Cursor(end), true
}

func decode_fixed(
	reader *Bit_Reader,
	destination Destination, dictionary History, count Count,
) (next_count Count, status Block_Status) {
	defer func() {
		Count_Invariants(next_count, "decode_fixed.next_count")
		Block_Status_Invariants(status, "decode_fixed.status")
	}()
	Bit_Reader_Invariants(reader, "decode_fixed.reader")
	Destination_Invariants(destination, "decode_fixed.destination")
	History_Invariants(dictionary, "decode_fixed.dictionary")
	Count_Invariants(count, "decode_fixed.count")
	return decode_fixed_raw(reader, destination, dictionary, count)
}

// Keeps the already-validated fixed-block cursor in registers until the block finishes.
func decode_fixed_raw(
	reader *Bit_Reader, destination Destination, dictionary History, count Count,
) (next_count Count, status Block_Status) {
	defer func() {
		Count_Invariants(next_count, "decode_fixed_raw.next_count")
		Block_Status_Invariants(status, "decode_fixed_raw.status")
	}()
	Bit_Reader_Invariants(reader, "decode_fixed_raw.reader")
	Destination_Invariants(destination, "decode_fixed_raw.destination")
	History_Invariants(dictionary, "decode_fixed_raw.dictionary")
	Count_Invariants(count, "decode_fixed_raw.count")
	cursor := Fixed_Bit_Cursor{
		Bits: Bit_Value(reader.Bits), Count: Bit_Count(reader.Bits_Count),
		Position: reader.Position,
	}
	defer bit_reader_store_raw(reader, &cursor)
	for more := Boolean(true); more; {
		for cursor.Count < HUFFMAN_LOOKUP_CODE_SIZE {
			if int(cursor.Position) == len(reader.Source) {
				break
			}
			cursor.Bits |= Bit_Value(reader.Source[cursor.Position]) << cursor.Count
			cursor.Position++
			cursor.Count += BYTE_BIT_COUNT
		}
		if cursor.Count < FIXED_THIRD_CODE_SIZE {
			return count, STATUS_INPUT_INVALID
		}
		prefix := Fixed_Prefix(cursor.Bits & HUFFMAN_LOOKUP_MASK)
		decoded := fixed_huffman_symbol_fast(prefix)
		if cursor.Count < Bit_Count(decoded.Size) {
			return count, STATUS_INPUT_INVALID
		}
		cursor.Bits >>= decoded.Size
		cursor.Count -= Bit_Count(decoded.Size)
		switch symbol := decoded.Symbol; {
		case symbol < LITERAL_SYMBOL_END:
			if int(count) == len(destination) {
				return count, STATUS_OUTPUT_TOO_SMALL
			}
			destination[count] = byte(symbol)
			count++
			continue
		case symbol == LITERAL_SYMBOL_END:
			return count, STATUS_OK
		case symbol > LITERAL_SYMBOL_MAXIMUM:
			return count, STATUS_INPUT_INVALID
		}
		match := Fixed_Match{
			Size: MATCH_SIZE_MINIMUM, Distance: DISTANCE_MINIMUM, Valid: true,
		}
		fixed_match_read(reader.Source, &cursor, Match_Symbol(decoded.Symbol), &match)
		if !match.Valid {
			return count, STATUS_INPUT_INVALID
		}
		if int(match.Distance) > len(dictionary)+int(count) {
			return count, STATUS_INPUT_INVALID
		}
		if match.Distance > WINDOW_SIZE {
			return count, STATUS_INPUT_INVALID
		}
		size, distance := match.Size, match.Distance
		if int(size) > len(destination)-int(count) {
			count, _ = match_copy(destination, dictionary, count, size, distance)
			return count, STATUS_OUTPUT_TOO_SMALL
		}
		output := Match_Destination(destination)
		match_copy_complete(output, dictionary, Match_Count(count), size, distance)
		count += Count(size)
	}
	return count, STATUS_INPUT_INVALID
}

func fixed_match_read(
	source Bit_Storage, cursor *Fixed_Bit_Cursor,
	symbol Match_Symbol, match *Fixed_Match,
) {
	Bit_Storage_Invariants(source, "fixed_match_read.source")
	Fixed_Bit_Cursor_Invariants(*cursor, "fixed_match_read.cursor")
	Match_Symbol_Invariants(symbol, "fixed_match_read.symbol")
	Fixed_Match_Invariants(match, "fixed_match_read.match")
	extra_count := Match_Extra_Bit_Count(BIT_COUNT_MINIMUM)
	relative := int(symbol) - MATCH_SYMBOL_MINIMUM
	match.Size = Match_Size(MATCH_SIZE_MINIMUM + relative)
	if symbol == MATCH_SYMBOL_MAXIMUM {
		match.Size = MATCH_SIZE_MAXIMUM
	} else if symbol >= MATCH_EXTRA_SYMBOL_MINIMUM {
		relative = int(symbol) - MATCH_EXTRA_SYMBOL_MINIMUM
		group := relative / MATCH_EXTRA_SYMBOL_GROUP_SIZE
		extra_count = Match_Extra_Bit_Count(group + MATCH_EXTRA_GROUP_MINIMUM)
		group_position := relative % MATCH_EXTRA_SYMBOL_GROUP_SIZE
		base := MATCH_SIZE_MINIMUM + 1<<(extra_count+MATCH_EXTRA_BASE_SHIFT)
		match.Size = Match_Size(base + group_position<<extra_count)
	}
	required := Bit_Count(extra_count)
	for field := FIXED_MATCH_SIZE_FIELD; field < FIXED_MATCH_FIELD_COUNT; field++ {
		for cursor.Count < required {
			if int(cursor.Position) == len(source) {
				match.Valid = false
				return
			}
			cursor.Bits |= Bit_Value(source[cursor.Position]) << cursor.Count
			cursor.Position++
			cursor.Count += BYTE_BIT_COUNT
		}
		mask := Bit_Value(LOW_BIT_MASK)<<required - Bit_Value(LOW_BIT_MASK)
		value := cursor.Bits & mask
		cursor.Bits >>= required
		cursor.Count -= required
		switch field {
		case FIXED_MATCH_SIZE_FIELD:
			match.Size += Match_Size(value)
			required = FIXED_DISTANCE_BIT_COUNT
		case FIXED_MATCH_DISTANCE_SYMBOL_FIELD:
			residue := uint8(value)
			reverse_mask := BIT_REVERSE_BYTE_SINGLE_MASK
			low := residue & reverse_mask
			residue = residue>>FINAL_BIT_COUNT&reverse_mask | low<<FINAL_BIT_COUNT
			reverse_mask = BIT_REVERSE_BYTE_PAIR_MASK
			low = residue & reverse_mask
			residue = residue>>BINARY_RADIX&reverse_mask | low<<BINARY_RADIX
			lower := residue & BIT_REVERSE_NIBBLE_MASK
			distance_symbol := Distance_Symbol(
				lower<<FINAL_BIT_COUNT | residue>>PENDING_BIT_COUNT_MAXIMUM,
			)
			if distance_symbol >= DISTANCE_SYMBOL_COUNT {
				match.Valid = false
				return
			}
			if distance_symbol < DISTANCE_DIRECT_SYMBOL_COUNT {
				match.Distance = Distance(distance_symbol + DISTANCE_MINIMUM)
				return
			}
			relative := int(distance_symbol) - DISTANCE_EXTRA_SYMBOL_OFFSET
			extra_count := Distance_Extra_Bit_Count(relative >> DISTANCE_GROUP_SHIFT)
			distance_group_base := 1 << (extra_count + FINAL_BIT_COUNT)
			match.Distance = Distance(distance_group_base + DISTANCE_MINIMUM)
			group_position := distance_symbol & LOW_BIT_MASK
			match.Distance += Distance(group_position << extra_count)
			required = Bit_Count(extra_count)
		case FIXED_MATCH_DISTANCE_FIELD:
			match.Distance += Distance(value)
		}
	}
}

// Restores the public cursor form by returning one wholly unread byte to source position.
func bit_reader_store_raw(
	reader *Bit_Reader, cursor *Fixed_Bit_Cursor,
) {
	Bit_Reader_Invariants(reader, "bit_reader_store_raw.reader")
	Fixed_Bit_Cursor_Invariants(*cursor, "bit_reader_store_raw.cursor")
	invariant.Always(
		int(cursor.Position) <= len(reader.Source),
		"Stored fixed cursor position does not cross compressed input.",
	)
	for cursor.Count > Bit_Count(PENDING_BIT_COUNT_MAXIMUM) {
		cursor.Count -= BYTE_BIT_COUNT
		cursor.Bits &= Bit_Value(LOW_BIT_MASK)<<cursor.Count - Bit_Value(LOW_BIT_MASK)
		cursor.Position--
	}
	reader.Bits = Pending_Bits(cursor.Bits)
	reader.Bits_Count = Pending_Bit_Count(cursor.Count)
	reader.Position = cursor.Position
}

func decode_huffman(
	reader *Bit_Reader,
	destination Destination, dictionary History, count Count,
	literal_decoder *Huffman_Decoder,
	distance_decoder *Huffman_Decoder,
	fixed Boolean,
) (next_count Count, status Block_Status) {
	defer func() {
		Count_Invariants(next_count, "decode_huffman.next_count")
		Block_Status_Invariants(status, "decode_huffman.status")
	}()
	Bit_Reader_Invariants(reader, "decode_huffman.reader")
	Destination_Invariants(destination, "decode_huffman.destination")
	History_Invariants(dictionary, "decode_huffman.dictionary")
	Count_Invariants(count, "decode_huffman.count")
	Huffman_Decoder_Invariants(
		literal_decoder, "decode_huffman.literal_decoder",
	)
	Huffman_Decoder_Invariants(
		distance_decoder, "decode_huffman.distance_decoder",
	)
	Boolean_Invariants(fixed, "decode_huffman.fixed")
	if fixed {
		return decode_fixed(reader, destination, dictionary, count)
	}
	for more := Boolean(true); more; {
		symbol, available := huffman_read(reader, literal_decoder)
		if !available {
			return count, STATUS_INPUT_INVALID
		}
		if symbol < LITERAL_SYMBOL_END {
			if int(count) == len(destination) {
				return count, STATUS_OUTPUT_TOO_SMALL
			}
			destination[count] = byte(symbol)
			count++
			continue
		}
		if symbol == LITERAL_SYMBOL_END {
			return count, STATUS_OK
		}
		if symbol > LITERAL_SYMBOL_MAXIMUM {
			return count, STATUS_INPUT_INVALID
		}
		var match_status Block_Status
		count, match_status = decode_match(
			reader,
			destination, dictionary, count, Match_Symbol(symbol), distance_decoder,
		)
		if match_status != STATUS_OK {
			return count, match_status
		}
	}
	return count, STATUS_INPUT_INVALID
}

func decode_match(
	reader *Bit_Reader,
	destination Destination, dictionary History, count Count,
	symbol Match_Symbol,
	distance_decoder *Huffman_Decoder,
) (next_count Count, status Block_Status) {
	defer func() {
		Count_Invariants(next_count, "decode_match.next_count")
		Block_Status_Invariants(status, "decode_match.status")
	}()
	Bit_Reader_Invariants(reader, "decode_match.reader")
	Destination_Invariants(destination, "decode_match.destination")
	History_Invariants(dictionary, "decode_match.dictionary")
	Count_Invariants(count, "decode_match.count")
	Match_Symbol_Invariants(symbol, "decode_match.symbol")
	Huffman_Decoder_Invariants(distance_decoder, "decode_match.distance_decoder")
	size_base, size_extra_count := decoded_match_code(symbol)
	size_extra, available := bit_reader_read(reader, Bit_Count(size_extra_count))
	if !available {
		return count, STATUS_INPUT_INVALID
	}
	distance_symbol, available := huffman_read(reader, distance_decoder)
	if !available {
		return count, STATUS_INPUT_INVALID
	}
	if distance_symbol >= DISTANCE_SYMBOL_COUNT {
		return count, STATUS_INPUT_INVALID
	}
	distance_base, distance_extra_count := decoded_distance_code(
		Distance_Symbol(distance_symbol),
	)
	distance_extra, available := bit_reader_read(
		reader,
		Bit_Count(distance_extra_count),
	)
	if !available {
		return count, STATUS_INPUT_INVALID
	}
	size := size_base + Match_Size(size_extra)
	distance := Distance(distance_base) + Distance(distance_extra)
	if int(distance) > len(dictionary)+int(count) {
		return count, STATUS_INPUT_INVALID
	}
	if distance > WINDOW_SIZE {
		return count, STATUS_INPUT_INVALID
	}
	next_count, copied := match_copy(
		destination, dictionary, count, size, distance,
	)
	if !copied {
		return next_count, STATUS_OUTPUT_TOO_SMALL
	}
	return next_count, STATUS_OK
}

func match_copy(
	destination Destination,
	dictionary History,
	count Count,
	size Match_Size,
	distance Distance,
) (next_count Count, copied Boolean) {
	defer func() {
		Count_Invariants(next_count, "match_copy.next_count")
		Boolean_Invariants(copied, "match_copy.copied")
	}()
	Destination_Invariants(destination, "match_copy.destination")
	History_Invariants(dictionary, "match_copy.dictionary")
	Count_Invariants(count, "match_copy.count")
	Match_Size_Invariants(size, "match_copy.size")
	Distance_Invariants(distance, "match_copy.distance")
	// Existing output can seed geometric copies because each new prefix doubles valid history.
	if len(dictionary) == HISTORY_SIZE_MINIMUM {
		available_count := len(destination) - int(count)
		copy_count := int(size)
		copied = true
		if copy_count > available_count {
			copy_count = available_count
			copied = false
		}
		if copy_count == BYTE_COUNT_MINIMUM {
			return count, copied
		}
		output := destination[int(count) : int(count)+copy_count]
		seed_count := int(distance)
		if seed_count > copy_count {
			seed_count = copy_count
		}
		source_start := int(count) - int(distance)
		copy(output[:seed_count], destination[source_start:source_start+seed_count])
		written_count := seed_count
		for written_count < copy_count {
			written_count += copy(output[written_count:], output[:written_count])
		}
		return Count(int(count) + copy_count), copied
	}
	for copied_index := 0; copied_index < int(size); copied_index++ {
		if int(count) == len(destination) {
			return count, false
		}
		history_position := len(dictionary) + int(count) - int(distance)
		if history_position < len(dictionary) {
			destination[count] = dictionary[history_position]
		} else {
			destination[count] = destination[history_position-len(dictionary)]
		}
		count++
	}
	return count, true
}

func match_copy_complete(
	destination Match_Destination,
	dictionary History,
	count Match_Count,
	size Match_Size,
	distance Distance,
) {
	Match_Destination_Invariants(destination, "match_copy_complete.destination")
	History_Invariants(dictionary, "match_copy_complete.dictionary")
	Match_Count_Invariants(count, "match_copy_complete.count")
	Match_Size_Invariants(size, "match_copy_complete.size")
	Distance_Invariants(distance, "match_copy_complete.distance")
	invariant.Always(
		int(count)+int(size) <= len(destination),
		"Complete match remains inside destination.",
	)
	invariant.Always(
		int(distance) <= len(dictionary)+int(count),
		"Complete match distance remains inside available history.",
	)
	if len(dictionary) == HISTORY_SIZE_MINIMUM {
		output := destination[int(count) : int(count)+int(size)]
		seed_count := int(distance)
		if seed_count > len(output) {
			seed_count = len(output)
		}
		source_start := int(count) - int(distance)
		copy(output[:seed_count], destination[source_start:source_start+seed_count])
		written_count := seed_count
		for written_count < len(output) {
			written_count += copy(output[written_count:], output[:written_count])
		}
	} else {
		for copied_index := 0; copied_index < int(size); copied_index++ {
			destination_position := int(count) + copied_index
			history_position := len(dictionary) + destination_position - int(distance)
			if history_position < len(dictionary) {
				destination[destination_position] = dictionary[history_position]
			} else {
				destination[destination_position] =
					destination[history_position-len(dictionary)]
			}
		}
	}
}

func decoded_match_code(
	symbol Match_Symbol,
) (size Match_Size, extra_count Match_Extra_Bit_Count) {
	defer func() {
		Match_Size_Invariants(size, "decoded_match_code.size")
		Match_Extra_Bit_Count_Invariants(
			extra_count, "decoded_match_code.extra_count",
		)
	}()
	Match_Symbol_Invariants(symbol, "decoded_match_code.symbol")
	return match_code_values(symbol)
}

func match_code_values(
	symbol Match_Symbol,
) (size Match_Size, extra_count Match_Extra_Bit_Count) {
	defer func() {
		Match_Size_Invariants(size, "match_code_values.size")
		Match_Extra_Bit_Count_Invariants(
			extra_count, "match_code_values.extra_count",
		)
	}()
	Match_Symbol_Invariants(symbol, "match_code_values.symbol")
	if symbol < MATCH_EXTRA_SYMBOL_MINIMUM {
		return Match_Size(
			int(symbol) - (MATCH_SYMBOL_MINIMUM - MATCH_SIZE_MINIMUM),
		), Match_Extra_Bit_Count(BIT_COUNT_MINIMUM)
	}
	if symbol == MATCH_SYMBOL_MAXIMUM {
		return MATCH_SIZE_MAXIMUM, Match_Extra_Bit_Count(BIT_COUNT_MINIMUM)
	}
	relative := int(symbol) - MATCH_EXTRA_SYMBOL_MINIMUM
	extra_count = Match_Extra_Bit_Count(
		relative/MATCH_EXTRA_SYMBOL_GROUP_SIZE + 1,
	)
	group_position := relative % MATCH_EXTRA_SYMBOL_GROUP_SIZE
	base := MATCH_SIZE_MINIMUM + 1<<(extra_count+MATCH_EXTRA_BASE_SHIFT)
	base += group_position << extra_count
	return Match_Size(base), extra_count
}

func decoded_distance_code(
	symbol Distance_Symbol,
) (distance Distance_Base, extra_count Distance_Extra_Bit_Count) {
	defer func() {
		Distance_Base_Invariants(distance, "decoded_distance_code.distance")
		Distance_Extra_Bit_Count_Invariants(
			extra_count, "decoded_distance_code.extra_count",
		)
	}()
	Distance_Symbol_Invariants(symbol, "decoded_distance_code.symbol")
	return distance_code_values(symbol)
}

func distance_code_values(
	symbol Distance_Symbol,
) (distance Distance_Base, extra_count Distance_Extra_Bit_Count) {
	defer func() {
		Distance_Base_Invariants(distance, "distance_code_values.distance")
		Distance_Extra_Bit_Count_Invariants(
			extra_count, "distance_code_values.extra_count",
		)
	}()
	Distance_Symbol_Invariants(symbol, "distance_code_values.symbol")
	if symbol < DISTANCE_DIRECT_SYMBOL_COUNT {
		return Distance_Base(symbol + DISTANCE_MINIMUM),
			Distance_Extra_Bit_Count(BIT_COUNT_MINIMUM)
	}
	extra_count = Distance_Extra_Bit_Count(
		(int(symbol) - DISTANCE_EXTRA_SYMBOL_OFFSET) >> DISTANCE_GROUP_SHIFT,
	)
	distance = Distance_Base(1<<(extra_count+1) + DISTANCE_MINIMUM)
	distance += Distance_Base((symbol & LOW_BIT_MASK) << extra_count)
	return distance, extra_count
}

func huffman_build(
	decoder *Huffman_Decoder, sizes Code_Sizes,
) (valid Boolean) {
	defer func() {
		Boolean_Invariants(valid, "huffman_build.valid")
		Huffman_Decoder_Invariants(decoder, "huffman_build.decoder")
	}()
	Huffman_Decoder_Invariants(decoder, "huffman_build.input_decoder")
	Code_Sizes_Invariants(sizes, "huffman_build.sizes")
	clear(decoder.Counts[:])
	clear(decoder.Symbols[:])
	clear(decoder.Lookup_Symbols[:])
	clear(decoder.Lookup_Sizes[:])
	decoder.Symbol_Count = 0
	decoder.Maximum_Code_Size = 0
	symbol_count := 0
	for _, size := range sizes {
		if size > CODE_SIZE_MAXIMUM {
			return false
		}
		decoder.Counts[size]++
		if size != 0 {
			symbol_count++
			if Canonical_Code_Size(size) > decoder.Maximum_Code_Size {
				decoder.Maximum_Code_Size = Canonical_Code_Size(size)
			}
		}
	}
	decoder.Symbol_Count = Canonical_Symbol_Count(symbol_count)
	if symbol_count == 0 {
		return true
	}
	space := 1
	for size := CODE_SIZES_MINIMUM; size <= CODE_SIZE_MAXIMUM; size++ {
		space = space*BINARY_RADIX - int(decoder.Counts[size])
		if space < 0 {
			return false
		}
	}
	if space != 0 {
		if symbol_count != 1 {
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
		if size != 0 {
			decoder.Symbols[offsets[size]] = uint16(symbol)
			offsets[size]++
		}
	}
	huffman_lookup_build(
		&decoder.Counts, &decoder.Lookup_Symbols, &decoder.Lookup_Sizes, sizes,
	)
	return true
}

func huffman_lookup_build(
	counts *[CODE_SIZE_MAXIMUM + 1]uint16,
	lookup_symbols *[HUFFMAN_LOOKUP_COUNT]uint16,
	lookup_sizes *[HUFFMAN_LOOKUP_COUNT]uint8,
	code_sizes Code_Sizes,
) {
	Code_Sizes_Invariants(code_sizes, "huffman_lookup_build.code_sizes")
	var codes [CODE_SIZE_MAXIMUM + 1]uint16
	code := bits.WORD_16_MINIMUM
	for code_size := CODE_SIZES_MINIMUM; code_size <= CODE_SIZE_MAXIMUM; code_size++ {
		previous_size := code_size - CODE_SIZES_MINIMUM
		code = (code + counts[previous_size]) << FINAL_BIT_COUNT
		codes[code_size] = code
	}
	for symbol, code_size := range code_sizes {
		if code_size == BIT_COUNT_MINIMUM {
			continue
		}
		canonical := codes[code_size]
		codes[code_size]++
		if code_size > HUFFMAN_LOOKUP_CODE_SIZE {
			continue
		}
		reversed := bits.Reverse_16(bits.Word_16(canonical)) >>
			(bits.BIT_COUNT_16_MAXIMUM - code_size)
		remaining_bit_count := HUFFMAN_LOOKUP_CODE_SIZE - int(code_size)
		suffix_count := 1 << remaining_bit_count
		for suffix_index := POSITION_MINIMUM; suffix_index < suffix_count; suffix_index++ {
			position := int(reversed) | suffix_index<<code_size
			lookup_symbols[position] = uint16(symbol)
			lookup_sizes[position] = code_size
		}
	}
}

func fixed_huffman_symbol_fast(
	prefix Fixed_Prefix,
) (decoded Fixed_Decoded_Symbol) {
	defer func() {
		Fixed_Decoded_Symbol_Invariants(decoded, "fixed_huffman_symbol_fast.decoded")
	}()
	Fixed_Prefix_Invariants(prefix, "fixed_huffman_symbol_fast.prefix")
	residue := uint8(prefix)
	mask := BIT_REVERSE_BYTE_SINGLE_MASK
	low := residue & mask
	residue = residue>>FINAL_BIT_COUNT&mask | low<<FINAL_BIT_COUNT
	mask = BIT_REVERSE_BYTE_PAIR_MASK
	low = residue & mask
	residue = residue>>BINARY_RADIX&mask | low<<BINARY_RADIX
	residue = residue>>BIT_REVERSE_NIBBLE_BIT_COUNT |
		residue<<BIT_REVERSE_NIBBLE_BIT_COUNT
	canonical := Fixed_Prefix(residue)<<FINAL_BIT_COUNT |
		prefix>>bits.BIT_COUNT_8_MAXIMUM
	code := canonical >> FIXED_THIRD_PREFIX_SHIFT
	if code < Fixed_Prefix(FIXED_LITERAL_THIRD_COUNT) {
		decoded.Symbol = Symbol(FIXED_LITERAL_THIRD_MINIMUM) + Symbol(code)
		decoded.Size = Fixed_Decode_Size(FIXED_THIRD_CODE_SIZE)
	} else {
		code = canonical >> FIXED_FIRST_PREFIX_SHIFT
		if code < Fixed_Prefix(FIXED_SECOND_CODE_BASE>>FINAL_BIT_COUNT) {
			position := code - Fixed_Prefix(FIXED_FIRST_CODE_BASE)
			if position < Fixed_Prefix(FIXED_LITERAL_FIRST_COUNT) {
				decoded.Symbol = Symbol(position)
			} else {
				decoded.Symbol = Symbol(FIXED_LITERAL_FOURTH_MINIMUM) +
					Symbol(position-Fixed_Prefix(FIXED_LITERAL_FIRST_COUNT))
			}
			decoded.Size = Fixed_Decode_Size(FIXED_CODE_SIZE)
		} else {
			decoded.Symbol = Symbol(FIXED_LITERAL_SECOND_MINIMUM) +
				Symbol(canonical-Fixed_Prefix(FIXED_SECOND_CODE_BASE))
			decoded.Size = Fixed_Decode_Size(FIXED_SECOND_CODE_SIZE)
		}
	}
	return decoded
}

func huffman_read(
	reader *Bit_Reader, decoder *Huffman_Decoder,
) (symbol Symbol, available Boolean) {
	defer func() {
		Symbol_Invariants(symbol, "huffman_read.symbol")
		Boolean_Invariants(available, "huffman_read.available")
	}()
	Bit_Reader_Invariants(reader, "huffman_read.reader")
	Huffman_Decoder_Invariants(decoder, "huffman_read.decoder")
	bit_buffer := Bit_Value(reader.Bits)
	bit_count := Bit_Count(reader.Bits_Count)
	source_position := reader.Position
	for bit_count < Bit_Count(HUFFMAN_LOOKUP_CODE_SIZE) {
		if int(source_position) == len(reader.Source) {
			break
		}
		bit_buffer |= Bit_Value(reader.Source[source_position]) << bit_count
		source_position++
		bit_count += Bit_Count(BYTE_BIT_COUNT)
	}
	if bit_count >= Bit_Count(HUFFMAN_LOOKUP_CODE_SIZE) {
		lookup_position := int(bit_buffer & Bit_Value(HUFFMAN_LOOKUP_MASK))
		lookup_size := decoder.Lookup_Sizes[lookup_position]
		remaining_count := bit_count - Bit_Count(lookup_size)
		if lookup_size != BIT_COUNT_MINIMUM {
			if remaining_count <= Bit_Count(PENDING_BIT_COUNT_MAXIMUM) {
				reader.Bits = Pending_Bits(bit_buffer >> lookup_size)
				reader.Bits_Count = Pending_Bit_Count(remaining_count)
				reader.Position = source_position
				return Symbol(decoder.Lookup_Symbols[lookup_position]), true
			}
		}
	}
	code := Bit_Value(0)
	first := Bit_Value(0)
	symbol_index := Bit_Value(0)
	for index := 1; index <= int(decoder.Maximum_Code_Size); index++ {
		bit, bit_available := bit_reader_read(reader, Bit_Count(FINAL_BIT_COUNT))
		if !bit_available {
			return 0, false
		}
		code |= bit
		count := Bit_Value(decoder.Counts[index])
		if code < first+count {
			symbol_position := symbol_index + code - first
			if symbol_position >= Bit_Value(decoder.Symbol_Count) {
				return 0, false
			}
			return Symbol(decoder.Symbols[symbol_position]), true
		}
		symbol_index += count
		first = (first + count) << FINAL_BIT_COUNT
		code <<= FINAL_BIT_COUNT
	}
	return 0, false
}

func bit_reader_read(
	reader *Bit_Reader,
	count Bit_Count,
) (value Bit_Value, available Boolean) {
	defer func() {
		Bit_Value_Invariants(value, "bit_reader_read.value")
		Boolean_Invariants(available, "bit_reader_read.available")
	}()
	Bit_Reader_Invariants(reader, "bit_reader_read.reader")
	Bit_Count_Invariants(count, "bit_reader_read.count")
	if count == 0 {
		return 0, true
	}
	for Bit_Count(reader.Bits_Count) < count {
		if int(reader.Position) == len(reader.Source) {
			return 0, false
		}
		reader.Bits |= Pending_Bits(
			uint64(reader.Source[reader.Position]) << reader.Bits_Count,
		)
		reader.Position++
		reader.Bits_Count += BYTE_BIT_COUNT
	}
	mask := uint64(1<<count) - 1
	value = Bit_Value(uint64(reader.Bits) & mask)
	reader.Bits >>= count
	reader.Bits_Count -= Pending_Bit_Count(count)
	return value, true
}

func bit_reader_align(reader *Bit_Reader) {
	Bit_Reader_Invariants(reader, "bit_reader_align.reader")
	reader.Bits = 0
	reader.Bits_Count = 0
}
