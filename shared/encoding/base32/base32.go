// Package base32 implements bounded RFC 4648 base32 on caller-owned storage.
package base32

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/simulation/aver/default"
)

// ALPHABET_SIZE follows the base32 radix.
const ALPHABET_SIZE = bits.BIT_COUNT_32_MAXIMUM

// ALPHABET_FINAL_INDEX lets boundary code avoid a detached final position.
const ALPHABET_FINAL_INDEX = ALPHABET_SIZE - 1

// ENCODED_SYMBOL_BIT_COUNT is the RFC 4648 base32 symbol width.
const ENCODED_SYMBOL_BIT_COUNT = 5

// DECODED_GROUP_SIZE follows the five source bytes represented by one complete quantum.
const DECODED_GROUP_SIZE = ENCODED_SYMBOL_BIT_COUNT

// ENCODED_GROUP_SIZE follows the eight base32 symbols representing one complete quantum.
const ENCODED_GROUP_SIZE = bits.BIT_COUNT_8_MAXIMUM

// DECODE_MAP_SIZE covers every possible encoded byte without branching on signedness.
const DECODE_MAP_SIZE = 1 << bits.BIT_COUNT_8_MAXIMUM

// INVALID_SYMBOL lies directly beyond every alphabet index.
const INVALID_SYMBOL = ALPHABET_SIZE

// SIZE_MINIMUM admits empty source and output.
const SIZE_MINIMUM = bytes.SLICE_SIZE_MINIMUM

// ENCODED_SIZE_MAXIMUM follows the repository byte-slice boundary.
const ENCODED_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM

// SOURCE_SIZE_MAXIMUM is the largest complete source-group count fitting encoded storage.
const SOURCE_SIZE_MAXIMUM = ENCODED_SIZE_MAXIMUM /
	ENCODED_GROUP_SIZE * DECODED_GROUP_SIZE

// DECODED_SIZE_MAXIMUM follows the largest output possible from bounded encoded input.
const DECODED_SIZE_MAXIMUM = SOURCE_SIZE_MAXIMUM

// BULK_GROUP_COUNT_MAXIMUM keeps paired source and destination advances inside bounds.
const BULK_GROUP_COUNT_MAXIMUM = ENCODED_SIZE_MAXIMUM / ENCODED_GROUP_SIZE

// DECODED_WRITE_DESTINATION_SIZE_MINIMUM admits the shortest valid terminal quantum.
const DECODED_WRITE_DESTINATION_SIZE_MINIMUM = SIZE_MINIMUM + 1

// DECODED_TAIL_SIZE_TWO advances the terminal source domain by one byte.
const DECODED_TAIL_SIZE_TWO = DECODED_WRITE_DESTINATION_SIZE_MINIMUM + 1

// DECODED_TAIL_SIZE_THREE advances the terminal source domain by one byte.
const DECODED_TAIL_SIZE_THREE = DECODED_TAIL_SIZE_TWO + 1

// DECODED_TAIL_SIZE_FINAL reaches the largest incomplete decoded group.
const DECODED_TAIL_SIZE_FINAL = DECODED_GROUP_SIZE - 1

// ENCODED_TAIL_SIZE_ONE derives the shortest useful terminal symbol count.
const ENCODED_TAIL_SIZE_ONE = (DECODED_WRITE_DESTINATION_SIZE_MINIMUM*
	bits.BIT_COUNT_8_MAXIMUM + ENCODED_SYMBOL_BIT_COUNT - 1) / ENCODED_SYMBOL_BIT_COUNT

// TAIL_QUANTUM_COUNT_MAXIMUM excludes one complete encoded group.
const TAIL_QUANTUM_COUNT_MAXIMUM = ENCODED_GROUP_SIZE - 1

// ENCODED_TAIL_SIZE_TWO derives the next useful terminal symbol count.
const ENCODED_TAIL_SIZE_TWO = (DECODED_TAIL_SIZE_TWO*bits.BIT_COUNT_8_MAXIMUM +
	ENCODED_SYMBOL_BIT_COUNT - 1) / ENCODED_SYMBOL_BIT_COUNT

// ENCODED_TAIL_SIZE_THREE derives the next useful terminal symbol count.
const ENCODED_TAIL_SIZE_THREE = (DECODED_TAIL_SIZE_THREE*bits.BIT_COUNT_8_MAXIMUM +
	ENCODED_SYMBOL_BIT_COUNT - 1) / ENCODED_SYMBOL_BIT_COUNT

// ENCODED_TAIL_SIZE_FINAL derives the largest incomplete encoded quantum.
const ENCODED_TAIL_SIZE_FINAL = (DECODED_TAIL_SIZE_FINAL*bits.BIT_COUNT_8_MAXIMUM +
	ENCODED_SYMBOL_BIT_COUNT - 1) / ENCODED_SYMBOL_BIT_COUNT

// ENCODED_COUNT_IMPOSSIBLE_MINIMUM cannot contain one complete decoded byte.
const ENCODED_COUNT_IMPOSSIBLE_MINIMUM = SIZE_MINIMUM + 1

// NO_PADDING disables final quantum padding.
const NO_PADDING Padding = -1

// STANDARD_PADDING is the RFC 4648 padding byte.
const STANDARD_PADDING Padding = '='

// PADDING_BYTE_MAXIMUM is the largest padding value representable on the wire.
const PADDING_BYTE_MAXIMUM Padding = Padding(bits.WORD_8_MAXIMUM)

// PADDING_STORAGE_VALUE_INDEX retains the optional wire byte.
const PADDING_STORAGE_VALUE_INDEX = SIZE_MINIMUM

// PADDING_STORAGE_ENABLED_INDEX distinguishes NO_PADDING from a zero padding byte.
const PADDING_STORAGE_ENABLED_INDEX = PADDING_STORAGE_VALUE_INDEX + 1

// PADDING_STORAGE_SIZE includes value and presence without a broad scalar field.
const PADDING_STORAGE_SIZE = PADDING_STORAGE_ENABLED_INDEX + 1

// PADDING_STORAGE_DISABLED is the zero-value absence marker.
const PADDING_STORAGE_DISABLED byte = 0

// PADDING_STORAGE_ENABLED is the only presence marker.
const PADDING_STORAGE_ENABLED = PADDING_STORAGE_DISABLED + 1

// CARRIAGE_RETURN remains ignored only while decoding.
const CARRIAGE_RETURN byte = '\r'

// LINE_FEED remains ignored only while decoding.
const LINE_FEED byte = '\n'

// STATUS_OK means the operation completed.
const STATUS_OK = 0

// STATUS_INPUT_INVALID means encoded bytes violate the selected grammar.
const STATUS_INPUT_INVALID = STATUS_OK + 1

// STATUS_OUTPUT_TOO_SMALL means caller storage cannot hold the next complete result.
const STATUS_OUTPUT_TOO_SMALL = STATUS_INPUT_INVALID + 1

// STATUS_STORAGE_INVALID means source and destination overlap.
const STATUS_STORAGE_INVALID = STATUS_OUTPUT_TOO_SMALL + 1

// STATUS_ENCODING_INVALID means a zero or failed configuration reached an operation.
const STATUS_ENCODING_INVALID = STATUS_STORAGE_INVALID + 1

// STATUS_ALPHABET_INVALID means alphabet bytes are duplicated or include a newline.
const STATUS_ALPHABET_INVALID = STATUS_ENCODING_INVALID + 1

// STATUS_PADDING_INVALID means padding is not one distinct wire byte or NO_PADDING.
const STATUS_PADDING_INVALID = STATUS_ALPHABET_INVALID + 1

// Alphabet holds every base32 digit exactly once when valid.
type Alphabet [ALPHABET_SIZE]byte

// Alphabet_Invariants keeps content validation in New_Encoding while proving fixed storage.
func Alphabet_Invariants(value Alphabet, _ aver.Namespace) {
	aver.Always(
		len(value) == ALPHABET_SIZE,
		"A base32 alphabet has one byte for every symbol.",
	)
}

// Padding includes the wire byte domain and invalid int16 values for status-based validation.
type Padding int16

// Padding_Invariants keeps unvalidated padding in its complete concrete scalar domain.
func Padding_Invariants(value Padding, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int16(
			int16(value), bits.INTEGER_16_MINIMUM, bits.INTEGER_16_MAXIMUM,
		).
		Ensure()
}

// Padding_Enabled keeps size formulas independent from wire-byte identity.
type Padding_Enabled bool

// Padding_Enabled_Invariants covers padded and unpadded configurations.
func Padding_Enabled_Invariants(value Padding_Enabled, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Padding is enabled.").
		Ensure()
}

// Padding_Storage retains optional padding without an invalid scalar field state.
type Padding_Storage [PADDING_STORAGE_SIZE]byte

// Padding_Storage_Invariants proves fixed representation; operations validate its marker.
func Padding_Storage_Invariants(value Padding_Storage, _ aver.Namespace) {
	aver.Always(
		len(value) == PADDING_STORAGE_SIZE,
		"Padding storage retains value and presence.",
	)
}

// Decode_Map gives one fixed lookup entry to every possible input byte.
type Decode_Map [DECODE_MAP_SIZE]uint8

// Decode_Map_Invariants proves the fixed lookup size without imposing content validity.
func Decode_Map_Invariants(value Decode_Map, _ aver.Namespace) {
	aver.Always(
		len(value) == DECODE_MAP_SIZE,
		"A decode map covers every possible input byte.",
	)
}

// Encoding_Valid reports complete validation without an interface result.
type Encoding_Valid bool

// Encoding_Valid_Invariants states its complete concrete boolean domain.
func Encoding_Valid_Invariants(value Encoding_Valid, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "An encoding configuration is valid.").
		Ensure()
}

// Encoding owns one alphabet lookup by value.
type Encoding struct {
	// Alphabet remains visible because the linter forbids hidden struct state.
	Alphabet Alphabet
	// Decode_Map lets validation prove that lookup and alphabet still agree.
	Decode_Map Decode_Map
	// Padding retains absence without consuming a valid byte sentinel.
	Padding Padding_Storage
}

// Encoding_Invariants composes storage while validity stays an operation result.
func Encoding_Invariants(value Encoding, namespace aver.Namespace) {
	Alphabet_Invariants(value.Alphabet, namespace)
	Decode_Map_Invariants(value.Decode_Map, namespace)
	Padding_Storage_Invariants(value.Padding, namespace)
}

// Source is decoded input whose padded encoding fits the package boundary.
type Source []byte

// Source_Invariants enforces the worst-case padded source bound.
func Source_Invariants(value Source, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Encoded is encoded input or writable encoded destination.
type Encoded []byte

// Encoded_Invariants keeps wire storage within repository byte boundary.
func Encoded_Invariants(value Encoded, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Decoded is writable decoded destination.
type Decoded []byte

// Decoded_Invariants bounds output to the maximum admitted encoded input.
func Decoded_Invariants(value Decoded, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SIZE_MINIMUM, DECODED_SIZE_MAXIMUM).
		Ensure()
}

// Decoded_Write_Destination is a suffix able to receive at least one decoded byte.
type Decoded_Write_Destination []byte

// Decoded_Write_Destination_Invariants excludes storage unable to receive a quantum.
func Decoded_Write_Destination_Invariants(
	value Decoded_Write_Destination, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), DECODED_WRITE_DESTINATION_SIZE_MINIMUM,
			DECODED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Source_Count is a decoded byte count accepted by size calculation.
type Source_Count int

// Source_Count_Invariants follows Source's complete length domain.
func Source_Count_Invariants(value Source_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Encoded_Count is one complete padded or valid unpadded representation size.
type Encoded_Count int

// Encoded_Count_Invariants excludes the shortest impossible representation.
func Encoded_Count_Invariants(value Encoded_Count, namespace aver.Namespace) {
	remainder := int(value) % ENCODED_GROUP_SIZE
	valid := remainder == SIZE_MINIMUM || remainder == ENCODED_TAIL_SIZE_ONE ||
		remainder == ENCODED_TAIL_SIZE_TWO || remainder == ENCODED_TAIL_SIZE_THREE ||
		remainder == ENCODED_TAIL_SIZE_FINAL
	aver.Always(
		valid,
		"Encoded output ends after a complete decoded-byte boundary.",
	)
	aver.Tree(value, namespace).
		Range_Holed_Int(
			int(value), SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM,
			ENCODED_COUNT_IMPOSSIBLE_MINIMUM, ENCODED_COUNT_IMPOSSIBLE_MINIMUM,
			ENCODED_COUNT_IMPOSSIBLE_MINIMUM, ENCODED_COUNT_IMPOSSIBLE_MINIMUM,
		).
		Ensure()
}

// Encoded_Input_Count is any bounded wire-storage length before grammar validation.
type Encoded_Input_Count int

// Encoded_Input_Count_Invariants follows Encoded's complete raw length domain.
func Encoded_Input_Count_Invariants(
	value Encoded_Input_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Decoded_Count is a decoded capacity or successful write count.
type Decoded_Count int

// Decoded_Count_Invariants follows Decoded's complete length domain.
func Decoded_Count_Invariants(value Decoded_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), SIZE_MINIMUM, DECODED_SIZE_MAXIMUM).
		Ensure()
}

// Bulk_Group_Count keeps paired source and destination advances one checked value.
type Bulk_Group_Count int

// Bulk_Group_Count_Invariants covers every complete bulk group.
func Bulk_Group_Count_Invariants(
	value Bulk_Group_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), SIZE_MINIMUM, BULK_GROUP_COUNT_MAXIMUM).
		Ensure()
}

// Tail_Quantum_Count excludes complete groups already published by decode loop.
type Tail_Quantum_Count int

// Tail_Quantum_Count_Invariants keeps only incomplete quantum state.
func Tail_Quantum_Count_Invariants(
	value Tail_Quantum_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), SIZE_MINIMUM, TAIL_QUANTUM_COUNT_MAXIMUM).
		Ensure()
}

// Padding_Count keeps terminal padding inside one quantum.
type Padding_Count int

// Padding_Count_Invariants permits no padding through one complete quantum.
func Padding_Count_Invariants(value Padding_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), SIZE_MINIMUM, ENCODED_GROUP_SIZE).
		Ensure()
}

// Decoded_Write_Symbol_Count excludes symbol counts that cannot produce one byte.
type Decoded_Write_Symbol_Count int

// Decoded_Write_Symbol_Count_Invariants bounds one publishable quantum.
func Decoded_Write_Symbol_Count_Invariants(
	value Decoded_Write_Symbol_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), ENCODED_TAIL_SIZE_ONE, ENCODED_GROUP_SIZE).
		Ensure()
}

// Decoded_Prefix_Count excludes byte counts that cannot hold complete groups.
type Decoded_Prefix_Count int

// Decoded_Prefix_Count_Invariants keeps published prefixes group-aligned.
func Decoded_Prefix_Count_Invariants(
	value Decoded_Prefix_Count, namespace aver.Namespace,
) {
	aver.Always(
		int(value)%DECODED_GROUP_SIZE == SIZE_MINIMUM,
		"Decoded prefix contains only complete groups.",
	)
	aver.Tree(value, namespace).
		Range_Holed_Int(
			int(value), SIZE_MINIMUM, DECODED_SIZE_MAXIMUM,
			SIZE_MINIMUM+1, SIZE_MINIMUM+2, SIZE_MINIMUM+2, SIZE_MINIMUM+2,
		).
		Ensure()
}

// Quantum stores one encoded group before validation publishes bytes.
type Quantum [ENCODED_GROUP_SIZE]uint8

// Quantum_Invariants proves fixed storage for the current encoded group.
func Quantum_Invariants(value Quantum, _ aver.Namespace) {
	aver.Always(
		len(value) == ENCODED_GROUP_SIZE,
		"A quantum has one slot for every encoded symbol.",
	)
}

// Configuration_Status reports atomic alphabet construction.
type Configuration_Status uint8

// Configuration_Status_Invariants lists constructor outcomes.
func Configuration_Status_Invariants(
	value Configuration_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_ALPHABET_INVALID),
			uint8(STATUS_PADDING_INVALID),
		).
		Ensure()
}

// Padding_Status reports immutable padding replacement.
type Padding_Status uint8

// Padding_Status_Invariants lists replacement outcomes.
func Padding_Status_Invariants(value Padding_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_ENCODING_INVALID),
			uint8(STATUS_PADDING_INVALID),
		).
		Ensure()
}

// Size_Status reports whether a configuration admits size calculation.
type Size_Status uint8

// Size_Status_Invariants lists size outcomes.
func Size_Status_Invariants(value Size_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_ENCODING_INVALID),
		).
		Ensure()
}

// Encode_Status reports caller storage and configuration failures.
type Encode_Status uint8

// Encode_Status_Invariants lists encoder outcomes.
func Encode_Status_Invariants(value Encode_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_OUTPUT_TOO_SMALL),
			uint8(STATUS_STORAGE_INVALID), uint8(STATUS_ENCODING_INVALID),
		).
		Ensure()
}

// Decode_Status adds malicious wire-input failure to encoder outcomes.
type Decode_Status uint8

// Decode_Status_Invariants covers its contiguous five-value outcome domain.
func Decode_Status_Invariants(value Decode_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_ENCODING_INVALID),
		).
		Ensure()
}

// Decode_Data_Status is the subset reachable after configuration and storage validation.
type Decode_Data_Status uint8

// Decode_Data_Status_Invariants lists wire parser outcomes.
func Decode_Data_Status_Invariants(
	value Decode_Data_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_OUTPUT_TOO_SMALL),
		).
		Ensure()
}

// New_Encoding validates the complete configuration before publishing lookup state.
func New_Encoding(
	alphabet Alphabet, padding Padding,
) (encoding Encoding, status Configuration_Status) {
	defer func() {
		Encoding_Invariants(encoding, "New_Encoding.encoding")
		Configuration_Status_Invariants(status, "New_Encoding.status")
	}()
	Alphabet_Invariants(alphabet, "New_Encoding.alphabet")
	Padding_Invariants(padding, "New_Encoding.padding")
	if padding < NO_PADDING {
		return Encoding{}, STATUS_PADDING_INVALID
	}
	if padding > PADDING_BYTE_MAXIMUM {
		return Encoding{}, STATUS_PADDING_INVALID
	}
	if padding == Padding(CARRIAGE_RETURN) {
		return Encoding{}, STATUS_PADDING_INVALID
	}
	if padding == Padding(LINE_FEED) {
		return Encoding{}, STATUS_PADDING_INVALID
	}
	for index := range encoding.Decode_Map {
		encoding.Decode_Map[index] = INVALID_SYMBOL
	}
	for index, symbol := range alphabet {
		if symbol == CARRIAGE_RETURN {
			return Encoding{}, STATUS_ALPHABET_INVALID
		}
		if symbol == LINE_FEED {
			return Encoding{}, STATUS_ALPHABET_INVALID
		}
		if encoding.Decode_Map[symbol] != INVALID_SYMBOL {
			return Encoding{}, STATUS_ALPHABET_INVALID
		}
		encoding.Decode_Map[symbol] = uint8(index)
	}
	if padding != NO_PADDING {
		if encoding.Decode_Map[byte(padding)] != INVALID_SYMBOL {
			return Encoding{}, STATUS_PADDING_INVALID
		}
	}
	encoding.Alphabet = alphabet
	if padding != NO_PADDING {
		encoding.Padding[PADDING_STORAGE_VALUE_INDEX] = byte(padding)
		encoding.Padding[PADDING_STORAGE_ENABLED_INDEX] = PADDING_STORAGE_ENABLED
	}
	return encoding, STATUS_OK
}

// Standard_Encoding rebuilds by value so no caller can mutate shared package state.
func Standard_Encoding() (encoding Encoding) {
	defer func() { Encoding_Invariants(encoding, "Standard_Encoding.encoding") }()
	var alphabet Alphabet
	copy(alphabet[:], "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567")
	encoding, status := New_Encoding(alphabet, STANDARD_PADDING)
	aver.Always(status == STATUS_OK, "Standard base32 alphabet is valid.")
	return encoding
}

// Hexadecimal_Encoding rebuilds by value so no caller can mutate shared package state.
func Hexadecimal_Encoding() (encoding Encoding) {
	defer func() { Encoding_Invariants(encoding, "Hexadecimal_Encoding.encoding") }()
	var alphabet Alphabet
	copy(alphabet[:], "0123456789ABCDEFGHIJKLMNOPQRSTUV")
	encoding, status := New_Encoding(alphabet, STANDARD_PADDING)
	aver.Always(status == STATUS_OK, "Hex base32 alphabet is valid.")
	return encoding
}

// With_Padding validates a copy so failed replacement cannot damage the input configuration.
func With_Padding(
	encoding Encoding, padding Padding,
) (configured Encoding, status Padding_Status) {
	defer func() {
		Encoding_Invariants(configured, "With_Padding.configured")
		Padding_Status_Invariants(status, "With_Padding.status")
	}()
	Encoding_Invariants(encoding, "With_Padding.encoding")
	Padding_Invariants(padding, "With_Padding.padding")
	if !bool(encoding_valid(encoding)) {
		return Encoding{}, STATUS_ENCODING_INVALID
	}
	configured, configuration_status := New_Encoding(encoding.Alphabet, padding)
	if configuration_status != STATUS_OK {
		return Encoding{}, STATUS_PADDING_INVALID
	}
	return configured, STATUS_OK
}

// Encoded_Size reports exact output because padding belongs to immutable configuration.
func Encoded_Size(
	encoding Encoding, source_count Source_Count,
) (count Encoded_Count, status Size_Status) {
	defer func() {
		Encoded_Count_Invariants(count, "Encoded_Size.count")
		Size_Status_Invariants(status, "Encoded_Size.status")
	}()
	Encoding_Invariants(encoding, "Encoded_Size.encoding")
	Source_Count_Invariants(source_count, "Encoded_Size.source_count")
	if !bool(encoding_valid(encoding)) {
		return 0, STATUS_ENCODING_INVALID
	}
	padding_enabled := Padding_Enabled(
		encoding.Padding[PADDING_STORAGE_ENABLED_INDEX] == PADDING_STORAGE_ENABLED,
	)
	return encoded_size_unchecked(source_count, padding_enabled), STATUS_OK
}

// Decoded_Size_Maximum remains safe when newline and padding later reduce actual output.
func Decoded_Size_Maximum(
	encoding Encoding, encoded_count Encoded_Input_Count,
) (count Decoded_Count, status Size_Status) {
	defer func() {
		Decoded_Count_Invariants(count, "Decoded_Size_Maximum.count")
		Size_Status_Invariants(status, "Decoded_Size_Maximum.status")
	}()
	Encoding_Invariants(encoding, "Decoded_Size_Maximum.encoding")
	Encoded_Input_Count_Invariants(
		encoded_count, "Decoded_Size_Maximum.encoded_count",
	)
	if !bool(encoding_valid(encoding)) {
		return 0, STATUS_ENCODING_INVALID
	}
	padding_enabled := Padding_Enabled(
		encoding.Padding[PADDING_STORAGE_ENABLED_INDEX] == PADDING_STORAGE_ENABLED,
	)
	return decoded_size_unchecked(encoded_count, padding_enabled), STATUS_OK
}

// Encode_Into checks every refusal before the first caller byte changes.
func Encode_Into(
	destination Encoded, source Source, encoding Encoding,
) (count Encoded_Count, status Encode_Status) {
	defer func() {
		Encoded_Count_Invariants(count, "Encode_Into.count")
		Encode_Status_Invariants(status, "Encode_Into.status")
	}()
	Encoded_Invariants(destination, "Encode_Into.destination")
	Source_Invariants(source, "Encode_Into.source")
	Encoding_Invariants(encoding, "Encode_Into.encoding")
	if !bool(encoding_valid(encoding)) {
		return 0, STATUS_ENCODING_INVALID
	}
	if bool(bytes.Overlap(bytes.Slice(destination), bytes.Slice(source))) {
		return 0, STATUS_STORAGE_INVALID
	}
	padding_enabled := Padding_Enabled(
		encoding.Padding[PADDING_STORAGE_ENABLED_INDEX] == PADDING_STORAGE_ENABLED,
	)
	required := encoded_size_unchecked(Source_Count(len(source)), padding_enabled)
	if len(destination) < int(required) {
		return 0, STATUS_OUTPUT_TOO_SMALL
	}
	return encode_unchecked(destination, source, &encoding, required), STATUS_OK
}

// Decode_Into rejects malformed wire bytes without allocating newline scratch storage.
func Decode_Into(
	destination Decoded, source Encoded, encoding Encoding,
) (count Decoded_Count, status Decode_Status) {
	defer func() {
		Decoded_Count_Invariants(count, "Decode_Into.count")
		Decode_Status_Invariants(status, "Decode_Into.status")
	}()
	Decoded_Invariants(destination, "Decode_Into.destination")
	Encoded_Invariants(source, "Decode_Into.source")
	Encoding_Invariants(encoding, "Decode_Into.encoding")
	if !bool(encoding_valid(encoding)) {
		return 0, STATUS_ENCODING_INVALID
	}
	if bool(bytes.Overlap(bytes.Slice(destination), bytes.Slice(source))) {
		return 0, STATUS_STORAGE_INVALID
	}
	decoded_count, data_status := decode_unchecked(destination, source, &encoding)
	return decoded_count, Decode_Status(data_status)
}

// Exact standard storage avoids the custom-map scan without trusting caller mutation.
func standard_encoding_valid(encoding *Encoding) (valid Encoding_Valid) {
	defer func() { Encoding_Valid_Invariants(valid, "standard_encoding_valid.valid") }()
	Encoding_Invariants(*encoding, "standard_encoding_valid.encoding")
	const INVALID_1 = string(rune(INVALID_SYMBOL))
	const INVALID_2 = INVALID_1 + INVALID_1
	const INVALID_4 = INVALID_2 + INVALID_2
	const INVALID_8 = INVALID_4 + INVALID_4
	const INVALID_16 = INVALID_8 + INVALID_8
	const INVALID_32 = INVALID_16 + INVALID_16
	const INVALID_64 = INVALID_32 + INVALID_32
	const INVALID_128 = INVALID_64 + INVALID_64
	const INVALID_PREFIX = INVALID_32 + INVALID_16 + INVALID_2
	const DIGITS = "\x1a\x1b\x1c\x1d\x1e\x1f"
	const INVALID_MIDDLE = INVALID_8 + INVALID_1
	const UPPER = "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c" +
		"\x0d\x0e\x0f\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19"
	const INVALID_FINAL = INVALID_128 + INVALID_32 + INVALID_4 + INVALID_1
	const STANDARD_ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
	const STANDARD_DECODE_MAP = INVALID_PREFIX + DIGITS + INVALID_MIDDLE + UPPER +
		INVALID_FINAL
	return Encoding_Valid(string(encoding.Alphabet[:]) == STANDARD_ALPHABET &&
		string(encoding.Decode_Map[:]) == STANDARD_DECODE_MAP &&
		encoding.Padding[PADDING_STORAGE_VALUE_INDEX] == byte(STANDARD_PADDING) &&
		encoding.Padding[PADDING_STORAGE_ENABLED_INDEX] == PADDING_STORAGE_ENABLED)
}

func encoding_valid(encoding Encoding) (valid Encoding_Valid) {
	defer func() { Encoding_Valid_Invariants(valid, "encoding_valid.valid") }()
	Encoding_Invariants(encoding, "encoding_valid.encoding")
	if bool(standard_encoding_valid(&encoding)) {
		return true
	}
	padding_enabled := encoding.Padding[PADDING_STORAGE_ENABLED_INDEX]
	if padding_enabled > PADDING_STORAGE_ENABLED {
		return false
	}
	padding := NO_PADDING
	if padding_enabled == PADDING_STORAGE_ENABLED {
		padding = Padding(encoding.Padding[PADDING_STORAGE_VALUE_INDEX])
	}
	if padding == Padding(CARRIAGE_RETURN) {
		return false
	}
	if padding == Padding(LINE_FEED) {
		return false
	}
	for index, symbol := range encoding.Alphabet {
		if symbol == CARRIAGE_RETURN {
			return false
		}
		if symbol == LINE_FEED {
			return false
		}
		if padding != NO_PADDING {
			if symbol == byte(padding) {
				return false
			}
		}
		if encoding.Decode_Map[symbol] != uint8(index) {
			return false
		}
	}
	for encoded_byte, symbol := range encoding.Decode_Map {
		if symbol == INVALID_SYMBOL {
			continue
		}
		if int(symbol) >= ALPHABET_SIZE {
			return false
		}
		if encoding.Alphabet[symbol] != byte(encoded_byte) {
			return false
		}
	}
	return true
}

func encoded_size_unchecked(
	source_count Source_Count, padding_enabled Padding_Enabled,
) (count Encoded_Count) {
	defer func() { Encoded_Count_Invariants(count, "encoded_size_unchecked.count") }()
	Source_Count_Invariants(source_count, "encoded_size_unchecked.source_count")
	Padding_Enabled_Invariants(padding_enabled, "encoded_size_unchecked.padding_enabled")
	if !bool(padding_enabled) {
		full_count := int(source_count) / DECODED_GROUP_SIZE * ENCODED_GROUP_SIZE
		tail_bit_count := int(source_count) % DECODED_GROUP_SIZE * bits.BIT_COUNT_8_MAXIMUM
		tail_count := (tail_bit_count + ENCODED_SYMBOL_BIT_COUNT - 1) /
			ENCODED_SYMBOL_BIT_COUNT
		return Encoded_Count(full_count + tail_count)
	}
	return Encoded_Count((int(source_count) + DECODED_GROUP_SIZE - 1) /
		DECODED_GROUP_SIZE * ENCODED_GROUP_SIZE)
}

func decoded_size_unchecked(
	encoded_count Encoded_Input_Count, padding_enabled Padding_Enabled,
) (count Decoded_Count) {
	defer func() { Decoded_Count_Invariants(count, "decoded_size_unchecked.count") }()
	Encoded_Input_Count_Invariants(
		encoded_count, "decoded_size_unchecked.encoded_count",
	)
	Padding_Enabled_Invariants(padding_enabled, "decoded_size_unchecked.padding_enabled")
	full_count := int(encoded_count) / ENCODED_GROUP_SIZE * DECODED_GROUP_SIZE
	if bool(padding_enabled) {
		return Decoded_Count(full_count)
	}
	tail_count := int(encoded_count) % ENCODED_GROUP_SIZE * ENCODED_SYMBOL_BIT_COUNT /
		bits.BIT_COUNT_8_MAXIMUM
	return Decoded_Count(full_count + tail_count)
}

func encode_unchecked(
	destination Encoded, source Source, encoding *Encoding, required Encoded_Count,
) (count Encoded_Count) {
	defer func() { Encoded_Count_Invariants(count, "encode_unchecked.count") }()
	Encoded_Invariants(destination, "encode_unchecked.destination")
	Source_Invariants(source, "encode_unchecked.source")
	Encoding_Invariants(*encoding, "encode_unchecked.encoding")
	Encoded_Count_Invariants(required, "encode_unchecked.required")
	group_count := encode_bulk_unchecked(destination, source, encoding)
	source_position := int(group_count) * DECODED_GROUP_SIZE
	position := int(group_count) * ENCODED_GROUP_SIZE
	bit_count := 0
	var value uint64
	for _, source_byte := range source[source_position:] {
		value = value<<bits.BIT_COUNT_8_MAXIMUM | uint64(source_byte)
		bit_count += bits.BIT_COUNT_8_MAXIMUM
		for bit_count >= ENCODED_SYMBOL_BIT_COUNT {
			bit_count -= ENCODED_SYMBOL_BIT_COUNT
			symbol := value >> bit_count & uint64(ALPHABET_FINAL_INDEX)
			destination[position] = encoding.Alphabet[symbol]
			position++
		}
	}
	if bit_count > 0 {
		symbol := value << (ENCODED_SYMBOL_BIT_COUNT - bit_count)
		symbol &= uint64(ALPHABET_FINAL_INDEX)
		destination[position] = encoding.Alphabet[symbol]
		position++
	}
	padding := NO_PADDING
	if encoding.Padding[PADDING_STORAGE_ENABLED_INDEX] == PADDING_STORAGE_ENABLED {
		padding = Padding(encoding.Padding[PADDING_STORAGE_VALUE_INDEX])
	}
	for position < int(required) {
		destination[position] = byte(padding)
		position++
	}
	return required
}

func encode_bulk_unchecked(
	destination Encoded, source Source, encoding *Encoding,
) (group_count Bulk_Group_Count) {
	defer func() {
		Bulk_Group_Count_Invariants(group_count, "encode_bulk_unchecked.group_count")
	}()
	Encoded_Invariants(destination, "encode_bulk_unchecked.destination")
	Source_Invariants(source, "encode_bulk_unchecked.source")
	Encoding_Invariants(*encoding, "encode_bulk_unchecked.encoding")
	const OFFSET_FIRST = SIZE_MINIMUM
	const OFFSET_SECOND = DECODED_GROUP_SIZE / DECODED_GROUP_SIZE
	const OFFSET_THIRD = OFFSET_SECOND + OFFSET_SECOND
	const OFFSET_FOURTH = OFFSET_THIRD + OFFSET_SECOND
	const OFFSET_FINAL = OFFSET_FOURTH + OFFSET_SECOND
	const GROUP_BIT_COUNT = DECODED_GROUP_SIZE * bits.BIT_COUNT_8_MAXIMUM
	const SHIFT_FIRST = GROUP_BIT_COUNT - ENCODED_SYMBOL_BIT_COUNT
	const SHIFT_SECOND = SHIFT_FIRST - ENCODED_SYMBOL_BIT_COUNT
	const SHIFT_THIRD = SHIFT_SECOND - ENCODED_SYMBOL_BIT_COUNT
	const SHIFT_FOURTH = SHIFT_THIRD - ENCODED_SYMBOL_BIT_COUNT
	const SHIFT_FIFTH = SHIFT_FOURTH - ENCODED_SYMBOL_BIT_COUNT
	const SHIFT_SIXTH = SHIFT_FIFTH - ENCODED_SYMBOL_BIT_COUNT
	const SHIFT_SEVENTH = SHIFT_SIXTH - ENCODED_SYMBOL_BIT_COUNT
	source_position := SIZE_MINIMUM
	destination_position := SIZE_MINIMUM
	full_end_position := len(source) / DECODED_GROUP_SIZE * DECODED_GROUP_SIZE
	for source_position < full_end_position {
		source_block := source[source_position : source_position+DECODED_GROUP_SIZE]
		destination_end_position := destination_position + ENCODED_GROUP_SIZE
		destination_block := destination[destination_position:destination_end_position]
		value := uint64(source_block[OFFSET_FIRST]) <<
			(OFFSET_FINAL * bits.BIT_COUNT_8_MAXIMUM)
		value |= uint64(source_block[OFFSET_SECOND]) <<
			(OFFSET_FOURTH * bits.BIT_COUNT_8_MAXIMUM)
		value |= uint64(source_block[OFFSET_THIRD]) <<
			(OFFSET_THIRD * bits.BIT_COUNT_8_MAXIMUM)
		value |= uint64(source_block[OFFSET_FOURTH]) <<
			(OFFSET_SECOND * bits.BIT_COUNT_8_MAXIMUM)
		value |= uint64(source_block[OFFSET_FINAL])
		destination_block[OFFSET_FIRST] =
			encoding.Alphabet[value>>SHIFT_FIRST&ALPHABET_FINAL_INDEX]
		destination_block[OFFSET_SECOND] =
			encoding.Alphabet[value>>SHIFT_SECOND&ALPHABET_FINAL_INDEX]
		destination_block[OFFSET_THIRD] =
			encoding.Alphabet[value>>SHIFT_THIRD&ALPHABET_FINAL_INDEX]
		destination_block[OFFSET_FOURTH] =
			encoding.Alphabet[value>>SHIFT_FOURTH&ALPHABET_FINAL_INDEX]
		destination_block[OFFSET_FINAL] =
			encoding.Alphabet[value>>SHIFT_FIFTH&ALPHABET_FINAL_INDEX]
		destination_block[OFFSET_FINAL+OFFSET_SECOND] =
			encoding.Alphabet[value>>SHIFT_SIXTH&ALPHABET_FINAL_INDEX]
		destination_block[OFFSET_FINAL+OFFSET_THIRD] =
			encoding.Alphabet[value>>SHIFT_SEVENTH&ALPHABET_FINAL_INDEX]
		destination_block[ENCODED_GROUP_SIZE-OFFSET_SECOND] =
			encoding.Alphabet[value&ALPHABET_FINAL_INDEX]
		source_position += DECODED_GROUP_SIZE
		destination_position += ENCODED_GROUP_SIZE
		group_count++
	}
	return group_count
}

func decode_bulk_unchecked(
	destination Decoded, source Encoded, encoding *Encoding,
) (group_count Bulk_Group_Count) {
	defer func() {
		Bulk_Group_Count_Invariants(group_count, "decode_bulk_unchecked.group_count")
	}()
	Decoded_Invariants(destination, "decode_bulk_unchecked.destination")
	Encoded_Invariants(source, "decode_bulk_unchecked.source")
	Encoding_Invariants(*encoding, "decode_bulk_unchecked.encoding")
	const OFFSET_FIRST = SIZE_MINIMUM
	const OFFSET_SECOND = DECODED_GROUP_SIZE / DECODED_GROUP_SIZE
	const OFFSET_THIRD = OFFSET_SECOND + OFFSET_SECOND
	const OFFSET_FOURTH = OFFSET_THIRD + OFFSET_SECOND
	const OFFSET_FIFTH = OFFSET_FOURTH + OFFSET_SECOND
	const OFFSET_SIXTH = OFFSET_FIFTH + OFFSET_SECOND
	const OFFSET_SEVENTH = OFFSET_SIXTH + OFFSET_SECOND
	const OFFSET_FINAL = OFFSET_SEVENTH + OFFSET_SECOND
	const GROUP_BIT_COUNT = DECODED_GROUP_SIZE * bits.BIT_COUNT_8_MAXIMUM
	const SHIFT_FIRST = GROUP_BIT_COUNT - ENCODED_SYMBOL_BIT_COUNT
	const SHIFT_SECOND = SHIFT_FIRST - ENCODED_SYMBOL_BIT_COUNT
	const SHIFT_THIRD = SHIFT_SECOND - ENCODED_SYMBOL_BIT_COUNT
	const SHIFT_FOURTH = SHIFT_THIRD - ENCODED_SYMBOL_BIT_COUNT
	const SHIFT_FIFTH = SHIFT_FOURTH - ENCODED_SYMBOL_BIT_COUNT
	const SHIFT_SIXTH = SHIFT_FIFTH - ENCODED_SYMBOL_BIT_COUNT
	const SHIFT_SEVENTH = SHIFT_SIXTH - ENCODED_SYMBOL_BIT_COUNT
	source_position := SIZE_MINIMUM
	destination_position := SIZE_MINIMUM
	for len(source)-source_position >= ENCODED_GROUP_SIZE {
		if len(destination)-destination_position < DECODED_GROUP_SIZE {
			break
		}
		block := source[source_position : source_position+ENCODED_GROUP_SIZE]
		first := encoding.Decode_Map[block[OFFSET_FIRST]]
		second := encoding.Decode_Map[block[OFFSET_SECOND]]
		third := encoding.Decode_Map[block[OFFSET_THIRD]]
		fourth := encoding.Decode_Map[block[OFFSET_FOURTH]]
		fifth := encoding.Decode_Map[block[OFFSET_FIFTH]]
		sixth := encoding.Decode_Map[block[OFFSET_SIXTH]]
		seventh := encoding.Decode_Map[block[OFFSET_SEVENTH]]
		final := encoding.Decode_Map[block[OFFSET_FINAL]]
		if first|second|third|fourth|fifth|sixth|seventh|final >= INVALID_SYMBOL {
			break
		}
		value := uint64(first)<<SHIFT_FIRST |
			uint64(second)<<SHIFT_SECOND |
			uint64(third)<<SHIFT_THIRD |
			uint64(fourth)<<SHIFT_FOURTH |
			uint64(fifth)<<SHIFT_FIFTH |
			uint64(sixth)<<SHIFT_SIXTH |
			uint64(seventh)<<SHIFT_SEVENTH |
			uint64(final)
		word := (*[DECODED_GROUP_SIZE]byte)(destination[destination_position:])
		*word = [DECODED_GROUP_SIZE]byte{
			byte(value >> (OFFSET_FIFTH * bits.BIT_COUNT_8_MAXIMUM)),
			byte(value >> (OFFSET_FOURTH * bits.BIT_COUNT_8_MAXIMUM)),
			byte(value >> (OFFSET_THIRD * bits.BIT_COUNT_8_MAXIMUM)),
			byte(value >> (OFFSET_SECOND * bits.BIT_COUNT_8_MAXIMUM)),
			byte(value),
		}
		source_position += ENCODED_GROUP_SIZE
		destination_position += DECODED_GROUP_SIZE
		group_count++
	}
	return group_count
}

func decode_unchecked(
	destination Decoded, source Encoded, encoding *Encoding,
) (count Decoded_Count, status Decode_Data_Status) {
	defer func() {
		Decoded_Count_Invariants(count, "decode_unchecked.count")
		Decode_Data_Status_Invariants(status, "decode_unchecked.status")
	}()
	Decoded_Invariants(destination, "decode_unchecked.destination")
	Encoded_Invariants(source, "decode_unchecked.source")
	Encoding_Invariants(*encoding, "decode_unchecked.encoding")
	group_count := decode_bulk_unchecked(destination, source, encoding)
	source_position := int(group_count) * ENCODED_GROUP_SIZE
	count = Decoded_Count(group_count) * DECODED_GROUP_SIZE
	var quantum Quantum
	quantum_count := 0
	padding_count := Padding_Count(0)
	padding := NO_PADDING
	if encoding.Padding[PADDING_STORAGE_ENABLED_INDEX] == PADDING_STORAGE_ENABLED {
		padding = Padding(encoding.Padding[PADDING_STORAGE_VALUE_INDEX])
	}
	ended := false
	for _, source_byte := range source[source_position:] {
		if source_byte == CARRIAGE_RETURN {
			continue
		}
		if source_byte == LINE_FEED {
			continue
		}
		if ended {
			return count, STATUS_INPUT_INVALID
		}
		is_padding := padding != NO_PADDING
		if is_padding {
			is_padding = source_byte == byte(padding)
		}
		if is_padding {
			padding_count++
			if int(quantum_count)+int(padding_count) > ENCODED_GROUP_SIZE {
				return count, STATUS_INPUT_INVALID
			}
			if int(quantum_count)+int(padding_count) == ENCODED_GROUP_SIZE {
				ended = true
			}
			continue
		}
		if padding_count != 0 {
			return count, STATUS_INPUT_INVALID
		}
		symbol := encoding.Decode_Map[source_byte]
		if symbol == INVALID_SYMBOL {
			return count, STATUS_INPUT_INVALID
		}
		quantum[quantum_count] = symbol
		quantum_count++
		if quantum_count != ENCODED_GROUP_SIZE {
			continue
		}
		if len(destination)-int(count) < DECODED_GROUP_SIZE {
			return count, STATUS_OUTPUT_TOO_SMALL
		}
		tail := Decoded_Write_Destination(destination[int(count):])
		decoded_write(tail, quantum, Decoded_Write_Symbol_Count(quantum_count))
		count += DECODED_GROUP_SIZE
		quantum_count = 0
	}
	return decode_tail(
		destination, quantum, Tail_Quantum_Count(quantum_count), padding_count,
		encoding, Decoded_Prefix_Count(count),
	)
}

func decode_tail(
	destination Decoded,
	quantum Quantum,
	quantum_count Tail_Quantum_Count,
	padding_count Padding_Count,
	encoding *Encoding,
	prefix_count Decoded_Prefix_Count,
) (count Decoded_Count, status Decode_Data_Status) {
	defer func() {
		Decoded_Count_Invariants(count, "decode_tail.count")
		Decode_Data_Status_Invariants(status, "decode_tail.status")
	}()
	Decoded_Invariants(destination, "decode_tail.destination")
	Quantum_Invariants(quantum, "decode_tail.quantum")
	Tail_Quantum_Count_Invariants(quantum_count, "decode_tail.quantum_count")
	Padding_Count_Invariants(padding_count, "decode_tail.padding_count")
	Encoding_Invariants(*encoding, "decode_tail.encoding")
	Decoded_Prefix_Count_Invariants(prefix_count, "decode_tail.prefix_count")
	count = Decoded_Count(prefix_count)
	if padding_count != 0 {
		if int(quantum_count) < ENCODED_TAIL_SIZE_ONE {
			return count, STATUS_INPUT_INVALID
		}
		if int(quantum_count)+int(padding_count) != ENCODED_GROUP_SIZE {
			return count, STATUS_INPUT_INVALID
		}
	} else {
		if quantum_count == 0 {
			return count, STATUS_OK
		}
		padding_enabled := encoding.Padding[PADDING_STORAGE_ENABLED_INDEX]
		if padding_enabled == PADDING_STORAGE_ENABLED {
			return count, STATUS_INPUT_INVALID
		}
	}
	decoded_count := int(quantum_count) * ENCODED_SYMBOL_BIT_COUNT /
		bits.BIT_COUNT_8_MAXIMUM
	required_symbol_count := (decoded_count*bits.BIT_COUNT_8_MAXIMUM +
		ENCODED_SYMBOL_BIT_COUNT - 1) / ENCODED_SYMBOL_BIT_COUNT
	if int(quantum_count) != required_symbol_count {
		return count, STATUS_INPUT_INVALID
	}
	if len(destination)-int(count) < decoded_count {
		return count, STATUS_OUTPUT_TOO_SMALL
	}
	tail := Decoded_Write_Destination(destination[int(count):])
	decoded_write(tail, quantum, Decoded_Write_Symbol_Count(quantum_count))
	return count + Decoded_Count(decoded_count), STATUS_OK
}

func decoded_write(
	destination Decoded_Write_Destination,
	quantum Quantum,
	symbol_count Decoded_Write_Symbol_Count,
) {
	Decoded_Write_Destination_Invariants(destination, "decoded_write.destination")
	Quantum_Invariants(quantum, "decoded_write.quantum")
	Decoded_Write_Symbol_Count_Invariants(symbol_count, "decoded_write.symbol_count")
	var value uint64
	for index := range int(symbol_count) {
		value = value<<ENCODED_SYMBOL_BIT_COUNT | uint64(quantum[index])
	}
	bit_count := int(symbol_count) * ENCODED_SYMBOL_BIT_COUNT
	decoded_size := bit_count / bits.BIT_COUNT_8_MAXIMUM
	for index := range decoded_size {
		bit_count -= bits.BIT_COUNT_8_MAXIMUM
		destination[index] = byte(value >> bit_count)
	}
}
