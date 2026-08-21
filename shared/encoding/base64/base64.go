// Package base64 implements bounded RFC 4648 base64 on caller-owned storage.
package base64

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
)

// ALPHABET_SIZE follows the base64 radix.
const ALPHABET_SIZE = bits.BIT_COUNT_64_MAXIMUM

// ALPHABET_FINAL_INDEX lets bit extraction avoid a detached mask.
const ALPHABET_FINAL_INDEX = ALPHABET_SIZE - 1

// ENCODED_SYMBOL_BIT_COUNT is the RFC 4648 base64 symbol width.
const ENCODED_SYMBOL_BIT_COUNT = 6

// ENCODED_GROUP_SIZE follows the four base64 symbols representing one complete quantum.
const ENCODED_GROUP_SIZE = bits.BIT_COUNT_32_MAXIMUM / bits.BIT_COUNT_8_MAXIMUM

// DECODED_GROUP_SIZE follows the three source bytes represented by one complete quantum.
const DECODED_GROUP_SIZE = ENCODED_GROUP_SIZE - 1

// DECODE_MAP_SIZE covers every possible encoded byte without signedness branching.
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

// DECODED_WRITE_DESTINATION_SIZE_MINIMUM admits the shortest valid terminal quantum.
const DECODED_WRITE_DESTINATION_SIZE_MINIMUM = SIZE_MINIMUM + 1

// DECODED_TAIL_SIZE_FINAL reaches the largest incomplete decoded group.
const DECODED_TAIL_SIZE_FINAL = DECODED_GROUP_SIZE - 1

// ENCODED_TAIL_SIZE_ONE derives the shortest useful terminal symbol count.
const ENCODED_TAIL_SIZE_ONE = (DECODED_WRITE_DESTINATION_SIZE_MINIMUM*
	bits.BIT_COUNT_8_MAXIMUM + ENCODED_SYMBOL_BIT_COUNT - 1) / ENCODED_SYMBOL_BIT_COUNT

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

// STRICT_STORAGE_INDEX holds the canonical-terminal-bit requirement.
const STRICT_STORAGE_INDEX = SIZE_MINIMUM

// STRICT_STORAGE_SIZE admits no unrelated state.
const STRICT_STORAGE_SIZE = STRICT_STORAGE_INDEX + 1

// STRICT_DISABLED is the zero-value permissive marker.
const STRICT_DISABLED byte = 0

// STRICT_ENABLED is the only strict marker.
const STRICT_ENABLED = STRICT_DISABLED + 1

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

// Alphabet holds every base64 digit exactly once when valid.
type Alphabet [ALPHABET_SIZE]byte

// Alphabet_Invariants keeps content validation in New_Encoding while proving fixed storage.
func Alphabet_Invariants(value Alphabet, _ invariant.Namespace) {
	invariant.Always(
		len(value) == ALPHABET_SIZE,
		"A base64 alphabet has one byte for every symbol.",
	)
}

// Padding includes the wire byte domain and invalid int16 values for status-based validation.
type Padding int16

// Padding_Invariants keeps unvalidated padding in its complete concrete scalar domain.
func Padding_Invariants(value Padding, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int16(
			int16(value), bits.INTEGER_16_MINIMUM, bits.INTEGER_16_MAXIMUM,
		).
		Ensure()
}

// Padding_Storage retains optional padding without an invalid scalar field state.
type Padding_Storage [PADDING_STORAGE_SIZE]byte

// Padding_Storage_Invariants proves fixed representation; operations validate its marker.
func Padding_Storage_Invariants(value Padding_Storage, _ invariant.Namespace) {
	invariant.Always(
		len(value) == PADDING_STORAGE_SIZE,
		"Padding storage retains value and presence.",
	)
}

// Strict_Storage retains strictness without a broad scalar field domain.
type Strict_Storage [STRICT_STORAGE_SIZE]byte

// Strict_Storage_Invariants proves fixed representation; operations validate its marker.
func Strict_Storage_Invariants(value Strict_Storage, _ invariant.Namespace) {
	invariant.Always(
		len(value) == STRICT_STORAGE_SIZE,
		"Strict storage retains one canonical-bit marker.",
	)
}

// Decode_Map gives one fixed lookup entry to every possible input byte.
type Decode_Map [DECODE_MAP_SIZE]uint8

// Decode_Map_Invariants proves the fixed lookup size without imposing content validity.
func Decode_Map_Invariants(value Decode_Map, _ invariant.Namespace) {
	invariant.Always(
		len(value) == DECODE_MAP_SIZE,
		"A decode map covers every possible input byte.",
	)
}

// Encoding_Valid reports complete validation without an interface result.
type Encoding_Valid bool

// Encoding_Valid_Invariants states its complete concrete boolean domain.
func Encoding_Valid_Invariants(value Encoding_Valid, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "An encoding configuration is valid.").
		Ensure()
}

// Encoding owns alphabet, lookup, padding, and strictness by value.
type Encoding struct {
	// Alphabet remains visible because the linter forbids hidden struct state.
	Alphabet Alphabet
	// Decode_Map lets validation prove that lookup and alphabet still agree.
	Decode_Map Decode_Map
	// Padding retains absence without consuming a valid byte sentinel.
	Padding Padding_Storage
	// Strict retains canonical-bit policy without a mutable boolean field.
	Strict Strict_Storage
}

// Encoding_Invariants composes storage while validity stays an operation result.
func Encoding_Invariants(value Encoding, namespace invariant.Namespace) {
	Alphabet_Invariants(value.Alphabet, namespace)
	Decode_Map_Invariants(value.Decode_Map, namespace)
	Padding_Storage_Invariants(value.Padding, namespace)
	Strict_Storage_Invariants(value.Strict, namespace)
}

// Source is decoded input whose padded encoding fits the package boundary.
type Source []byte

// Source_Invariants enforces the worst-case padded source bound.
func Source_Invariants(value Source, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Encoded is encoded input or writable encoded destination.
type Encoded []byte

// Encoded_Invariants keeps wire storage within repository byte boundary.
func Encoded_Invariants(value Encoded, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Decoded is writable decoded destination.
type Decoded []byte

// Decoded_Invariants bounds output to the maximum admitted encoded input.
func Decoded_Invariants(value Decoded, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SIZE_MINIMUM, DECODED_SIZE_MAXIMUM).
		Ensure()
}

// Decoded_Write_Destination is a suffix able to receive at least one decoded byte.
type Decoded_Write_Destination []byte

// Decoded_Write_Destination_Invariants excludes storage unable to receive a quantum.
func Decoded_Write_Destination_Invariants(
	value Decoded_Write_Destination, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), DECODED_WRITE_DESTINATION_SIZE_MINIMUM,
			DECODED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Source_Count is a decoded byte count accepted by size calculation.
type Source_Count int

// Source_Count_Invariants follows Source's complete length domain.
func Source_Count_Invariants(value Source_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), SIZE_MINIMUM, SOURCE_SIZE_MAXIMUM).
		Ensure()
}

// Encoded_Count is one complete padded or valid unpadded representation size.
type Encoded_Count int

// Encoded_Count_Invariants excludes the shortest impossible representation.
func Encoded_Count_Invariants(value Encoded_Count, namespace invariant.Namespace) {
	remainder := int(value) % ENCODED_GROUP_SIZE
	valid := remainder == SIZE_MINIMUM || remainder == ENCODED_TAIL_SIZE_ONE ||
		remainder == ENCODED_TAIL_SIZE_FINAL
	invariant.Always(
		valid,
		"Encoded output ends after a complete decoded-byte boundary.",
	)
	invariant.Tree(value, namespace).
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
	value Encoded_Input_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
}

// Decoded_Count is a decoded capacity or successful write count.
type Decoded_Count int

// Decoded_Count_Invariants follows Decoded's complete length domain.
func Decoded_Count_Invariants(value Decoded_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), SIZE_MINIMUM, DECODED_SIZE_MAXIMUM).
		Ensure()
}

// Quantum stores one encoded group before validation publishes bytes.
type Quantum [ENCODED_GROUP_SIZE]uint8

// Quantum_Invariants proves fixed storage for the current encoded group.
func Quantum_Invariants(value Quantum, _ invariant.Namespace) {
	invariant.Always(
		len(value) == ENCODED_GROUP_SIZE,
		"A quantum has one slot for every encoded symbol.",
	)
}

// Configuration_Status reports atomic alphabet construction.
type Configuration_Status uint8

// Configuration_Status_Invariants lists constructor outcomes.
func Configuration_Status_Invariants(
	value Configuration_Status, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_ALPHABET_INVALID),
			uint8(STATUS_PADDING_INVALID),
		).
		Ensure()
}

// Padding_Status reports immutable padding replacement.
type Padding_Status uint8

// Padding_Status_Invariants lists replacement outcomes.
func Padding_Status_Invariants(value Padding_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_ENCODING_INVALID),
			uint8(STATUS_PADDING_INVALID),
		).
		Ensure()
}

// Strict_Status reports immutable strictness selection.
type Strict_Status uint8

// Strict_Status_Invariants lists strictness outcomes.
func Strict_Status_Invariants(value Strict_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_ENCODING_INVALID),
		).
		Ensure()
}

// Size_Status reports whether a configuration admits size calculation.
type Size_Status uint8

// Size_Status_Invariants lists size outcomes.
func Size_Status_Invariants(value Size_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_ENCODING_INVALID),
		).
		Ensure()
}

// Encode_Status reports caller storage and configuration failures.
type Encode_Status uint8

// Encode_Status_Invariants lists encoder outcomes.
func Encode_Status_Invariants(value Encode_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_OUTPUT_TOO_SMALL),
			uint8(STATUS_STORAGE_INVALID), uint8(STATUS_ENCODING_INVALID),
		).
		Ensure()
}

// Decode_Status adds malicious wire-input failure to encoder outcomes.
type Decode_Status uint8

// Decode_Status_Invariants covers its contiguous five-value outcome domain.
func Decode_Status_Invariants(value Decode_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_ENCODING_INVALID),
		).
		Ensure()
}

// Decode_Data_Status is the subset reachable after configuration and storage validation.
type Decode_Data_Status uint8

// Decode_Data_Status_Invariants lists wire parser outcomes.
func Decode_Data_Status_Invariants(
	value Decode_Data_Status, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
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
	copy(alphabet[:], "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/")
	encoding, status := New_Encoding(alphabet, STANDARD_PADDING)
	invariant.Always(status == STATUS_OK, "Standard base64 alphabet is valid.")
	return encoding
}

// URL_Encoding rebuilds by value so no caller can mutate shared package state.
func URL_Encoding() (encoding Encoding) {
	defer func() { Encoding_Invariants(encoding, "URL_Encoding.encoding") }()
	var alphabet Alphabet
	copy(alphabet[:], "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_")
	encoding, status := New_Encoding(alphabet, STANDARD_PADDING)
	invariant.Always(status == STATUS_OK, "URL base64 alphabet is valid.")
	return encoding
}

// Raw_Standard_Encoding derives unpadded standard configuration without shared state.
func Raw_Standard_Encoding() (encoding Encoding) {
	defer func() { Encoding_Invariants(encoding, "Raw_Standard_Encoding.encoding") }()
	encoding, status := With_Padding(Standard_Encoding(), NO_PADDING)
	invariant.Always(status == STATUS_OK, "Raw standard base64 configuration is valid.")
	return encoding
}

// Raw_URL_Encoding derives unpadded URL configuration without shared state.
func Raw_URL_Encoding() (encoding Encoding) {
	defer func() { Encoding_Invariants(encoding, "Raw_URL_Encoding.encoding") }()
	encoding, status := With_Padding(URL_Encoding(), NO_PADDING)
	invariant.Always(status == STATUS_OK, "Raw URL base64 configuration is valid.")
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
	configured.Strict = encoding.Strict
	return configured, STATUS_OK
}

// Strict_Encoding validates a copy before requiring canonical terminal bits.
func Strict_Encoding(
	encoding Encoding,
) (configured Encoding, status Strict_Status) {
	defer func() {
		Encoding_Invariants(configured, "Strict_Encoding.configured")
		Strict_Status_Invariants(status, "Strict_Encoding.status")
	}()
	Encoding_Invariants(encoding, "Strict_Encoding.encoding")
	if !bool(encoding_valid(encoding)) {
		return Encoding{}, STATUS_ENCODING_INVALID
	}
	configured = encoding
	configured.Strict[STRICT_STORAGE_INDEX] = STRICT_ENABLED
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
	padding := NO_PADDING
	if encoding.Padding[PADDING_STORAGE_ENABLED_INDEX] == PADDING_STORAGE_ENABLED {
		padding = Padding(encoding.Padding[PADDING_STORAGE_VALUE_INDEX])
	}
	return encoded_size_unchecked(source_count, padding), STATUS_OK
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
	padding := NO_PADDING
	if encoding.Padding[PADDING_STORAGE_ENABLED_INDEX] == PADDING_STORAGE_ENABLED {
		padding = Padding(encoding.Padding[PADDING_STORAGE_VALUE_INDEX])
	}
	return decoded_size_unchecked(encoded_count, padding), STATUS_OK
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
	padding := NO_PADDING
	if encoding.Padding[PADDING_STORAGE_ENABLED_INDEX] == PADDING_STORAGE_ENABLED {
		padding = Padding(encoding.Padding[PADDING_STORAGE_VALUE_INDEX])
	}
	required := encoded_size_unchecked(Source_Count(len(source)), padding)
	if len(destination) < int(required) {
		return 0, STATUS_OUTPUT_TOO_SMALL
	}
	return encode_unchecked(destination, source, encoding), STATUS_OK
}

// Decode_Into rejects malformed wire bytes without newline scratch storage.
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
	decoded_count, data_status := decode_unchecked(destination, source, encoding)
	return decoded_count, Decode_Status(data_status)
}

func encoding_valid(encoding Encoding) (valid Encoding_Valid) {
	defer func() { Encoding_Valid_Invariants(valid, "encoding_valid.valid") }()
	Encoding_Invariants(encoding, "encoding_valid.encoding")
	padding_enabled := encoding.Padding[PADDING_STORAGE_ENABLED_INDEX]
	if padding_enabled > PADDING_STORAGE_ENABLED {
		return false
	}
	if padding_enabled == PADDING_STORAGE_DISABLED {
		if encoding.Padding[PADDING_STORAGE_VALUE_INDEX] != byte(NO_PADDING+1) {
			return false
		}
	}
	strict := encoding.Strict[STRICT_STORAGE_INDEX]
	if strict > STRICT_ENABLED {
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

func encoded_size_unchecked[Padding_Value ~int16](
	source_count Source_Count, padding Padding_Value,
) (count Encoded_Count) {
	defer func() { Encoded_Count_Invariants(count, "encoded_size_unchecked.count") }()
	Source_Count_Invariants(source_count, "encoded_size_unchecked.source_count")
	if Padding(padding) == NO_PADDING {
		full_count := int(source_count) / DECODED_GROUP_SIZE * ENCODED_GROUP_SIZE
		tail_bit_count := int(source_count) % DECODED_GROUP_SIZE * bits.BIT_COUNT_8_MAXIMUM
		tail_count := (tail_bit_count + ENCODED_SYMBOL_BIT_COUNT - 1) /
			ENCODED_SYMBOL_BIT_COUNT
		return Encoded_Count(full_count + tail_count)
	}
	return Encoded_Count((int(source_count) + DECODED_GROUP_SIZE - 1) /
		DECODED_GROUP_SIZE * ENCODED_GROUP_SIZE)
}

func decoded_size_unchecked[Padding_Value ~int16](
	encoded_count Encoded_Input_Count, padding Padding_Value,
) (count Decoded_Count) {
	defer func() { Decoded_Count_Invariants(count, "decoded_size_unchecked.count") }()
	Encoded_Input_Count_Invariants(
		encoded_count, "decoded_size_unchecked.encoded_count",
	)
	full_count := int(encoded_count) / ENCODED_GROUP_SIZE * DECODED_GROUP_SIZE
	if Padding(padding) != NO_PADDING {
		return Decoded_Count(full_count)
	}
	tail_count := int(encoded_count) % ENCODED_GROUP_SIZE * ENCODED_SYMBOL_BIT_COUNT /
		bits.BIT_COUNT_8_MAXIMUM
	return Decoded_Count(full_count + tail_count)
}

func encode_unchecked(
	destination Encoded, source Source, encoding Encoding,
) (count Encoded_Count) {
	defer func() { Encoded_Count_Invariants(count, "encode_unchecked.count") }()
	Encoded_Invariants(destination, "encode_unchecked.destination")
	Source_Invariants(source, "encode_unchecked.source")
	Encoding_Invariants(encoding, "encode_unchecked.encoding")
	position := 0
	bit_count := 0
	var value uint64
	for _, source_byte := range source {
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
	required := encoded_size_unchecked(Source_Count(len(source)), padding)
	for position < int(required) {
		destination[position] = byte(padding)
		position++
	}
	return Encoded_Count(position)
}

func decode_unchecked(
	destination Decoded, source Encoded, encoding Encoding,
) (count Decoded_Count, status Decode_Data_Status) {
	defer func() {
		Decoded_Count_Invariants(count, "decode_unchecked.count")
		Decode_Data_Status_Invariants(status, "decode_unchecked.status")
	}()
	Decoded_Invariants(destination, "decode_unchecked.destination")
	Encoded_Invariants(source, "decode_unchecked.source")
	Encoding_Invariants(encoding, "decode_unchecked.encoding")
	var quantum Quantum
	quantum_count := 0
	padding_count := 0
	padding := NO_PADDING
	if encoding.Padding[PADDING_STORAGE_ENABLED_INDEX] == PADDING_STORAGE_ENABLED {
		padding = Padding(encoding.Padding[PADDING_STORAGE_VALUE_INDEX])
	}
	ended := false
	for _, source_byte := range source {
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
			if quantum_count+padding_count > ENCODED_GROUP_SIZE {
				return count, STATUS_INPUT_INVALID
			}
			if quantum_count+padding_count == ENCODED_GROUP_SIZE {
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
		decoded_write(tail, quantum, quantum_count)
		count += DECODED_GROUP_SIZE
		quantum_count = 0
	}
	return decode_tail(
		destination, quantum, quantum_count, padding_count, encoding, int(count),
	)
}

func decode_tail[Quantum_Count ~int, Padding_Count ~int, Prefix_Count ~int](
	destination Decoded,
	quantum Quantum,
	quantum_count Quantum_Count,
	padding_count Padding_Count,
	encoding Encoding,
	prefix_count Prefix_Count,
) (count Decoded_Count, status Decode_Data_Status) {
	defer func() {
		Decoded_Count_Invariants(count, "decode_tail.count")
		Decode_Data_Status_Invariants(status, "decode_tail.status")
	}()
	Decoded_Invariants(destination, "decode_tail.destination")
	Quantum_Invariants(quantum, "decode_tail.quantum")
	Encoding_Invariants(encoding, "decode_tail.encoding")
	count = Decoded_Count(prefix_count)
	invariant.Always(int(quantum_count) >= SIZE_MINIMUM, "Quantum count is nonnegative.")
	invariant.Always(
		int(quantum_count) <= ENCODED_GROUP_SIZE,
		"Quantum count stays inside fixed storage.",
	)
	invariant.Always(int(padding_count) >= SIZE_MINIMUM, "Padding count is nonnegative.")
	invariant.Always(
		int(padding_count) <= ENCODED_GROUP_SIZE,
		"Padding count stays inside fixed storage.",
	)
	invariant.Always(int(prefix_count) >= SIZE_MINIMUM, "Decoded prefix is nonnegative.")
	invariant.Always(
		int(prefix_count) <= DECODED_SIZE_MAXIMUM,
		"Decoded prefix stays inside destination storage.",
	)
	invariant.Always(
		int(prefix_count)%DECODED_GROUP_SIZE == 0,
		"Decoded prefix contains only complete groups.",
	)
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
	if encoding.Strict[STRICT_STORAGE_INDEX] == STRICT_ENABLED {
		unused_bit_count := int(quantum_count)*ENCODED_SYMBOL_BIT_COUNT -
			decoded_count*bits.BIT_COUNT_8_MAXIMUM
		unused_mask := uint8(PADDING_STORAGE_ENABLED)<<unused_bit_count -
			PADDING_STORAGE_ENABLED
		if quantum[int(quantum_count)-1]&unused_mask != 0 {
			return count, STATUS_INPUT_INVALID
		}
	}
	if len(destination)-int(count) < decoded_count {
		return count, STATUS_OUTPUT_TOO_SMALL
	}
	tail := Decoded_Write_Destination(destination[int(count):])
	decoded_write(tail, quantum, quantum_count)
	return count + Decoded_Count(decoded_count), STATUS_OK
}

func decoded_write[Count ~int](
	destination Decoded_Write_Destination,
	quantum Quantum,
	symbol_count Count,
) {
	Decoded_Write_Destination_Invariants(destination, "decoded_write.destination")
	Quantum_Invariants(quantum, "decoded_write.quantum")
	invariant.Always(int(symbol_count) > 0, "Decoded quantum contains output bits.")
	invariant.Always(
		int(symbol_count) <= ENCODED_GROUP_SIZE,
		"Decoded quantum stays within fixed symbol storage.",
	)
	var value uint32
	for index := range int(symbol_count) {
		value = value<<ENCODED_SYMBOL_BIT_COUNT | uint32(quantum[index])
	}
	bit_count := int(symbol_count) * ENCODED_SYMBOL_BIT_COUNT
	decoded_size := bit_count / bits.BIT_COUNT_8_MAXIMUM
	for index := range decoded_size {
		bit_count -= bits.BIT_COUNT_8_MAXIMUM
		destination[index] = byte(value >> bit_count)
	}
}
