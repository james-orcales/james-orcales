// Package base32 implements bounded RFC 4648 base32 on caller-owned storage.
package base32

import (
	"unsafe"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
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

// PADDING_CODE_DISABLED reserves zero for absence.
const PADDING_CODE_DISABLED uint16 = 0

// PADDING_CODE_OFFSET shifts every wire byte above absence.
const PADDING_CODE_OFFSET uint16 = PADDING_CODE_DISABLED + 1

// PADDING_CODE_MAXIMUM reserves zero above every wire byte.
const PADDING_CODE_MAXIMUM uint16 = uint16(bits.WORD_8_MAXIMUM) + PADDING_CODE_OFFSET

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

// QUANTUM_INDEX_MAXIMUM is final explicit base32 symbol slot.
const QUANTUM_INDEX_MAXIMUM = ENCODED_GROUP_SIZE - 1

// QUANTUM_VALUE_MAXIMUM holds eight five-bit symbols.
const QUANTUM_VALUE_MAXIMUM = 1<<(ENCODED_GROUP_SIZE*ENCODED_SYMBOL_BIT_COUNT) - 1

// Alphabet holds caller configuration without mutable shared storage.
type Alphabet string

// Alphabet_Invariants bounds hostile constructor input before validation scans it.
func Alphabet_Invariants(value Alphabet, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SIZE_MINIMUM, ENCODED_SIZE_MAXIMUM).
		Ensure()
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

// Encoding_Valid reports complete validation without an interface result.
type Encoding_Valid bool

// Encoding_Valid_Invariants states its complete concrete boolean domain.
func Encoding_Valid_Invariants(value Encoding_Valid, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "An encoding configuration is valid.").
		Ensure()
}

// Encoding_Alphabet_Storage marks the first byte of immutable alphabet storage.
type Encoding_Alphabet_Storage struct{}

// Encoding_Alphabet_Storage_Invariants has no state beyond its address.
func Encoding_Alphabet_Storage_Invariants(
	value Encoding_Alphabet_Storage, _ aver.Namespace,
) {
	aver.Always(
		value == (Encoding_Alphabet_Storage{}),
		"Alphabet storage marker has no payload.",
	)
}

// Encoding_Alphabet holds the first byte of immutable wire symbols.
type Encoding_Alphabet *Encoding_Alphabet_Storage

// Encoding_Alphabet_Invariants composes present alphabet storage.
func Encoding_Alphabet_Invariants(value Encoding_Alphabet, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Encoding_Alphabet_Storage_Invariants(*value, namespace)
}

// Encoding_Padding holds absent or offset wire padding without raw boundary syntax.
type Encoding_Padding interface{}

// Decoded_Symbol holds one alphabet index or INVALID_SYMBOL.
type Decoded_Symbol uint8

// Decoded_Symbol_Invariants includes every index plus invalid sentinel.
func Decoded_Symbol_Invariants(value Decoded_Symbol, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(SIZE_MINIMUM), uint8(INVALID_SYMBOL)).
		Ensure()
}

// Encoding owns immutable alphabet and validated padding by value.
type Encoding struct {
	// Alphabet retains immutable wire symbols by address.
	Alphabet Encoding_Alphabet
	// Padding reserves zero for NO_PADDING and offsets one wire byte.
	Padding Encoding_Padding
}

// Encoding_Invariants composes immutable configuration storage.
func Encoding_Invariants(value Encoding, namespace aver.Namespace) {
	padding, padding_valid := value.Padding.(uint16)
	Encoding_Alphabet_Invariants(value.Alphabet, namespace)
	aver.Always(
		padding_valid == (value.Padding != nil),
		"Encoding padding storage has expected type.",
	)
	aver.Always(
		padding <= PADDING_CODE_MAXIMUM,
		"Padding code reserves zero above every wire byte.",
	)
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

// Quantum_Symbol is one decoded five-bit alphabet index.
type Quantum_Symbol uint8

// Quantum_Symbol_Invariants keeps accumulated symbols inside alphabet.
func Quantum_Symbol_Invariants(value Quantum_Symbol, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), SIZE_MINIMUM, ALPHABET_FINAL_INDEX).
		Ensure()
}

// Quantum_Index selects one explicit symbol field.
type Quantum_Index int

// Quantum_Index_Invariants keeps dynamic traversal inside one quantum.
func Quantum_Index_Invariants(value Quantum_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), SIZE_MINIMUM, QUANTUM_INDEX_MAXIMUM).
		Ensure()
}

// Quantum packs one encoded group without fixed-array API.
type Quantum uint64

// Quantum_Invariants proves fixed storage for the current encoded group.
func Quantum_Invariants(value Quantum, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), SIZE_MINIMUM, QUANTUM_VALUE_MAXIMUM).
		Ensure()
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
	if len(alphabet) != ALPHABET_SIZE {
		return Encoding{}, STATUS_ALPHABET_INVALID
	}
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
	for index := range len(alphabet) {
		symbol := alphabet[index]
		if symbol == CARRIAGE_RETURN {
			return Encoding{}, STATUS_ALPHABET_INVALID
		}
		if symbol == LINE_FEED {
			return Encoding{}, STATUS_ALPHABET_INVALID
		}
		for previous_index := 0; previous_index < index; previous_index++ {
			if alphabet[previous_index] == symbol {
				return Encoding{}, STATUS_ALPHABET_INVALID
			}
		}
	}
	if padding != NO_PADDING {
		for index := range len(alphabet) {
			if alphabet[index] == byte(padding) {
				return Encoding{}, STATUS_PADDING_INVALID
			}
		}
	}
	encoding.Alphabet = Encoding_Alphabet(unsafe.Pointer(unsafe.StringData(string(alphabet))))
	encoding.Padding = uint16(PADDING_CODE_DISABLED)
	if padding != NO_PADDING {
		encoding.Padding = uint16(padding) + PADDING_CODE_OFFSET
	}
	return encoding, STATUS_OK
}

// Standard_Encoding rebuilds by value so no caller can mutate shared package state.
func Standard_Encoding() (encoding Encoding) {
	defer func() { Encoding_Invariants(encoding, "Standard_Encoding.encoding") }()
	alphabet := Alphabet("ABCDEFGHIJKLMNOPQRSTUVWXYZ234567")
	encoding, status := New_Encoding(alphabet, STANDARD_PADDING)
	aver.Always(status == STATUS_OK, "Standard base32 alphabet is valid.")
	return encoding
}

// Hexadecimal_Encoding rebuilds by value so no caller can mutate shared package state.
func Hexadecimal_Encoding() (encoding Encoding) {
	defer func() { Encoding_Invariants(encoding, "Hexadecimal_Encoding.encoding") }()
	alphabet := Alphabet("0123456789ABCDEFGHIJKLMNOPQRSTUV")
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
	alphabet := Alphabet(unsafe.String(
		(*byte)(unsafe.Pointer(encoding.Alphabet)), ALPHABET_SIZE))
	if padding != NO_PADDING {
		for index := range len(alphabet) {
			if alphabet[index] == byte(padding) {
				return Encoding{}, STATUS_PADDING_INVALID
			}
		}
	}
	configured = encoding
	configured.Padding = uint16(PADDING_CODE_DISABLED)
	if padding != NO_PADDING {
		configured.Padding = uint16(padding) + PADDING_CODE_OFFSET
	}
	if !bool(encoding_valid(configured)) {
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
	padding_enabled := Padding_Enabled(encoding.Padding != PADDING_CODE_DISABLED)
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
	padding_enabled := Padding_Enabled(encoding.Padding != PADDING_CODE_DISABLED)
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
	padding_enabled := Padding_Enabled(encoding.Padding != PADDING_CODE_DISABLED)
	required := encoded_size_unchecked(Source_Count(len(source)), padding_enabled)
	if len(destination) < int(required) {
		return 0, STATUS_OUTPUT_TOO_SMALL
	}
	return encode_unchecked(destination, source, encoding, required), STATUS_OK
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
	decoded_count, data_status := decode_unchecked(destination, source, encoding)
	return decoded_count, Decode_Status(data_status)
}

func encoding_valid(encoding Encoding) (valid Encoding_Valid) {
	defer func() { Encoding_Valid_Invariants(valid, "encoding_valid.valid") }()
	Encoding_Invariants(encoding, "encoding_valid.encoding")
	if encoding.Alphabet == nil {
		return false
	}
	alphabet := Alphabet(unsafe.String(
		(*byte)(unsafe.Pointer(encoding.Alphabet)), ALPHABET_SIZE))
	padding_code, _ := encoding.Padding.(uint16)
	padding := NO_PADDING
	if padding_code != PADDING_CODE_DISABLED {
		padding = Padding(padding_code - PADDING_CODE_OFFSET)
	}
	if padding == Padding(CARRIAGE_RETURN) {
		return false
	}
	if padding == Padding(LINE_FEED) {
		return false
	}
	for index := range len(alphabet) {
		symbol := alphabet[index]
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
		for previous_index := 0; previous_index < index; previous_index++ {
			if alphabet[previous_index] == symbol {
				return false
			}
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
	destination Encoded, source Source, encoding Encoding, required Encoded_Count,
) (count Encoded_Count) {
	defer func() { Encoded_Count_Invariants(count, "encode_unchecked.count") }()
	Encoded_Invariants(destination, "encode_unchecked.destination")
	Source_Invariants(source, "encode_unchecked.source")
	Encoding_Invariants(encoding, "encode_unchecked.encoding")
	Encoded_Count_Invariants(required, "encode_unchecked.required")
	alphabet := Alphabet(unsafe.String(
		(*byte)(unsafe.Pointer(encoding.Alphabet)), ALPHABET_SIZE))
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
			destination[position] = alphabet[symbol]
			position++
		}
	}
	if bit_count > 0 {
		symbol := value << (ENCODED_SYMBOL_BIT_COUNT - bit_count)
		symbol &= uint64(ALPHABET_FINAL_INDEX)
		destination[position] = alphabet[symbol]
		position++
	}
	padding := NO_PADDING
	padding_code, _ := encoding.Padding.(uint16)
	if padding_code != PADDING_CODE_DISABLED {
		padding = Padding(padding_code - PADDING_CODE_OFFSET)
	}
	for position < int(required) {
		destination[position] = byte(padding)
		position++
	}
	return required
}

func encode_bulk_unchecked(
	destination Encoded, source Source, encoding Encoding,
) (group_count Bulk_Group_Count) {
	defer func() {
		Bulk_Group_Count_Invariants(group_count, "encode_bulk_unchecked.group_count")
	}()
	Encoded_Invariants(destination, "encode_bulk_unchecked.destination")
	Source_Invariants(source, "encode_bulk_unchecked.source")
	Encoding_Invariants(encoding, "encode_bulk_unchecked.encoding")
	alphabet := Alphabet(unsafe.String(
		(*byte)(unsafe.Pointer(encoding.Alphabet)), ALPHABET_SIZE))
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
			alphabet[value>>SHIFT_FIRST&ALPHABET_FINAL_INDEX]
		destination_block[OFFSET_SECOND] =
			alphabet[value>>SHIFT_SECOND&ALPHABET_FINAL_INDEX]
		destination_block[OFFSET_THIRD] =
			alphabet[value>>SHIFT_THIRD&ALPHABET_FINAL_INDEX]
		destination_block[OFFSET_FOURTH] =
			alphabet[value>>SHIFT_FOURTH&ALPHABET_FINAL_INDEX]
		destination_block[OFFSET_FINAL] =
			alphabet[value>>SHIFT_FIFTH&ALPHABET_FINAL_INDEX]
		destination_block[OFFSET_FINAL+OFFSET_SECOND] =
			alphabet[value>>SHIFT_SIXTH&ALPHABET_FINAL_INDEX]
		destination_block[OFFSET_FINAL+OFFSET_THIRD] =
			alphabet[value>>SHIFT_SEVENTH&ALPHABET_FINAL_INDEX]
		destination_block[ENCODED_GROUP_SIZE-OFFSET_SECOND] =
			alphabet[value&ALPHABET_FINAL_INDEX]
		source_position += DECODED_GROUP_SIZE
		destination_position += ENCODED_GROUP_SIZE
		group_count++
	}
	return group_count
}

func decode_bulk_unchecked(
	destination Decoded, source Encoded, encoding Encoding,
) (group_count Bulk_Group_Count) {
	defer func() {
		Bulk_Group_Count_Invariants(group_count, "decode_bulk_unchecked.group_count")
	}()
	Decoded_Invariants(destination, "decode_bulk_unchecked.destination")
	Encoded_Invariants(source, "decode_bulk_unchecked.source")
	Encoding_Invariants(encoding, "decode_bulk_unchecked.encoding")
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
		first := decode_symbol(encoding, bytes.Byte(block[OFFSET_FIRST]))
		second := decode_symbol(encoding, bytes.Byte(block[OFFSET_SECOND]))
		third := decode_symbol(encoding, bytes.Byte(block[OFFSET_THIRD]))
		fourth := decode_symbol(encoding, bytes.Byte(block[OFFSET_FOURTH]))
		fifth := decode_symbol(encoding, bytes.Byte(block[OFFSET_FIFTH]))
		sixth := decode_symbol(encoding, bytes.Byte(block[OFFSET_SIXTH]))
		seventh := decode_symbol(encoding, bytes.Byte(block[OFFSET_SEVENTH]))
		final := decode_symbol(encoding, bytes.Byte(block[OFFSET_FINAL]))
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
	destination Decoded, source Encoded, encoding Encoding,
) (count Decoded_Count, status Decode_Data_Status) {
	defer func() {
		Decoded_Count_Invariants(count, "decode_unchecked.count")
		Decode_Data_Status_Invariants(status, "decode_unchecked.status")
	}()
	Decoded_Invariants(destination, "decode_unchecked.destination")
	Encoded_Invariants(source, "decode_unchecked.source")
	Encoding_Invariants(encoding, "decode_unchecked.encoding")
	group_count := decode_bulk_unchecked(destination, source, encoding)
	source_position := int(group_count) * ENCODED_GROUP_SIZE
	count = Decoded_Count(group_count) * DECODED_GROUP_SIZE
	quantum, ended := Quantum(0), false
	quantum_count := 0
	padding_count := Padding_Count(0)
	padding := NO_PADDING
	padding_code, _ := encoding.Padding.(uint16)
	if padding_code != PADDING_CODE_DISABLED {
		padding = Padding(padding_code - PADDING_CODE_OFFSET)
	}
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
		symbol := decode_symbol(encoding, bytes.Byte(source_byte))
		if symbol == INVALID_SYMBOL {
			return count, STATUS_INPUT_INVALID
		}
		quantum = quantum_with_symbol(
			quantum, Quantum_Index(quantum_count), Quantum_Symbol(symbol),
		)
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
	encoding Encoding,
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
	Encoding_Invariants(encoding, "decode_tail.encoding")
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
		padding_code, _ := encoding.Padding.(uint16)
		if padding_code != PADDING_CODE_DISABLED {
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

func decode_symbol(
	encoding Encoding, source_byte bytes.Byte,
) (symbol Decoded_Symbol) {
	defer func() { Decoded_Symbol_Invariants(symbol, "decode_symbol.symbol") }()
	Encoding_Invariants(encoding, "decode_symbol.encoding")
	bytes.Byte_Invariants(source_byte, "decode_symbol.source_byte")
	alphabet := Alphabet(unsafe.String(
		(*byte)(unsafe.Pointer(encoding.Alphabet)), ALPHABET_SIZE))
	for index := range len(alphabet) {
		if alphabet[index] == byte(source_byte) {
			return Decoded_Symbol(index)
		}
	}
	return INVALID_SYMBOL
}

func quantum_with_symbol(
	quantum Quantum, index Quantum_Index, symbol Quantum_Symbol,
) (updated Quantum) {
	defer func() { Quantum_Invariants(updated, "quantum_with_symbol.updated") }()
	Quantum_Invariants(quantum, "quantum_with_symbol.quantum")
	Quantum_Index_Invariants(index, "quantum_with_symbol.index")
	Quantum_Symbol_Invariants(symbol, "quantum_with_symbol.symbol")
	shift := (QUANTUM_INDEX_MAXIMUM - int(index)) * ENCODED_SYMBOL_BIT_COUNT
	mask := Quantum(ALPHABET_FINAL_INDEX) << shift
	return quantum&^mask | Quantum(symbol)<<shift
}

func quantum_symbol(quantum Quantum, index Quantum_Index) (symbol Quantum_Symbol) {
	defer func() { Quantum_Symbol_Invariants(symbol, "quantum_symbol.symbol") }()
	Quantum_Invariants(quantum, "quantum_symbol.quantum")
	Quantum_Index_Invariants(index, "quantum_symbol.index")
	shift := (QUANTUM_INDEX_MAXIMUM - int(index)) * ENCODED_SYMBOL_BIT_COUNT
	return Quantum_Symbol(quantum >> shift & ALPHABET_FINAL_INDEX)
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
		value = value<<ENCODED_SYMBOL_BIT_COUNT |
			uint64(quantum_symbol(quantum, Quantum_Index(index)))
	}
	bit_count := int(symbol_count) * ENCODED_SYMBOL_BIT_COUNT
	decoded_size := bit_count / bits.BIT_COUNT_8_MAXIMUM
	for index := range decoded_size {
		bit_count -= bits.BIT_COUNT_8_MAXIMUM
		destination[index] = byte(value >> bit_count)
	}
}
