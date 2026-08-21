package big

import (
	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
)

// Int_Double_Word is one normalized nonnegative two-word magnitude.
type Int_Double_Word Int

// Int_Double_Word_Invariants states exact shape selected by public square-root dispatch.
func Int_Double_Word_Invariants(value *Int_Double_Word, _ invariant.Namespace) {
	invariant.Always(
		value.Negative == POLARITY_NONNEGATIVE,
		"A double-word integer is nonnegative.",
	)
	invariant.Always(
		value.Count == Word_Count(BASE_BINARY),
		"A double-word integer owns exactly two active words.",
	)
	invariant.Always(
		value.Words[WORD_COUNT_INCREMENT] != 0,
		"A double-word integer retains its high word.",
	)
}

// MODULAR_DOUBLE_WORD_LIMB_COUNT expands two machine words into safe half-word limbs.
const MODULAR_DOUBLE_WORD_LIMB_COUNT = BASE_BINARY * TEXT_DIVISION_WORD_LIMB_COUNT

// DOUBLE_WORD_PRODUCT_LOW_INDEX selects low by low multiplication.
const DOUBLE_WORD_PRODUCT_LOW_INDEX = WORD_COUNT_MINIMUM

// DOUBLE_WORD_PRODUCT_CROSS_LEFT_INDEX selects low by high multiplication.
const DOUBLE_WORD_PRODUCT_CROSS_LEFT_INDEX = DOUBLE_WORD_PRODUCT_LOW_INDEX +
	WORD_COUNT_INCREMENT

// DOUBLE_WORD_PRODUCT_CROSS_RIGHT_INDEX selects high by low multiplication.
const DOUBLE_WORD_PRODUCT_CROSS_RIGHT_INDEX = DOUBLE_WORD_PRODUCT_CROSS_LEFT_INDEX +
	WORD_COUNT_INCREMENT

// DOUBLE_WORD_PRODUCT_HIGH_INDEX selects high by high multiplication.
const DOUBLE_WORD_PRODUCT_HIGH_INDEX = DOUBLE_WORD_PRODUCT_CROSS_RIGHT_INDEX +
	WORD_COUNT_INCREMENT

// DOUBLE_WORD_PRODUCT_COUNT covers every pair of two input words.
const DOUBLE_WORD_PRODUCT_COUNT = DOUBLE_WORD_PRODUCT_HIGH_INDEX + WORD_COUNT_INCREMENT

// Double_Word_Products holds four complete word products.
type Double_Word_Products [DOUBLE_WORD_PRODUCT_COUNT]struct {
	Low  Word
	High Word
}

// Double_Word_Products_Invariants fixes exact two-by-two multiplication storage.
func Double_Word_Products_Invariants(value *Double_Word_Products, _ invariant.Namespace) {
	invariant.Always(
		len(value) == DOUBLE_WORD_PRODUCT_COUNT,
		"Double-word multiplication owns every word pair product.",
	)
}

// MODULAR_PRODUCT_LIMB_COUNT reserves every limb of a two-by-two-word product.
const MODULAR_PRODUCT_LIMB_COUNT = BASE_BINARY * MODULAR_DOUBLE_WORD_LIMB_COUNT

// MODULAR_DIVIDEND_LIMB_COUNT includes the leading Knuth carry limb.
const MODULAR_DIVIDEND_LIMB_COUNT = MODULAR_PRODUCT_LIMB_COUNT + WORD_COUNT_INCREMENT

// MODULAR_DIVISOR_LIMB_END_INDEX is the exclusive divisor limb position.
const MODULAR_DIVISOR_LIMB_END_INDEX = MODULAR_DOUBLE_WORD_LIMB_COUNT

// MODULAR_REMAINDER_HIGH_LIMB_INDEX selects the high half of the high remainder word.
const MODULAR_REMAINDER_HIGH_LIMB_INDEX = BASE_BINARY + WORD_COUNT_INCREMENT

// MODULAR_MERSENNE_HIGH_WORD derives the high word of one less than the double-word sign bit.
const MODULAR_MERSENNE_HIGH_WORD = Word(bits.WORD_64_MAXIMUM >> WORD_COUNT_INCREMENT)

// Modular_Dividend_Limbs reserves one Knuth carry limb above the complete product.
type Modular_Dividend_Limbs [MODULAR_DIVIDEND_LIMB_COUNT]uint32

// Modular_Dividend_Limbs_Invariants binds reduction scratch to its formula capacity.
func Modular_Dividend_Limbs_Invariants(
	value *Modular_Dividend_Limbs, _ invariant.Namespace,
) {
	invariant.Always(
		len(value) == MODULAR_DIVIDEND_LIMB_COUNT,
		"Normalized modular dividend scratch has formula capacity.",
	)
}

// Modular_Divisor_Limbs holds every limb of one two-word modulus.
type Modular_Divisor_Limbs [MODULAR_DOUBLE_WORD_LIMB_COUNT]uint32

// Modular_Divisor_Limbs_Invariants binds normalized divisor scratch to two words.
func Modular_Divisor_Limbs_Invariants(
	value *Modular_Divisor_Limbs, _ invariant.Namespace,
) {
	invariant.Always(
		len(value) == MODULAR_DOUBLE_WORD_LIMB_COUNT,
		"Normalized modular divisor scratch has formula capacity.",
	)
}

func int_modular_reduce_normalized_double_word(
	references *Int_References,
) (matched Boolean) {
	defer func() {
		Boolean_Invariants(matched, "int_modular_reduce_normalized_double_word.matched")
	}()
	Int_References_Invariants(
		references, "int_modular_reduce_normalized_double_word.references",
	)
	destination := references[INT_REFERENCE_DESTINATION_INDEX]
	product := references[INT_REFERENCE_LEFT_INDEX]
	modulus := references[INT_REFERENCE_RIGHT_INDEX]
	if modulus.Count != Word_Count(BASE_BINARY) {
		return false
	}
	if product.Count > Word_Count(BASE_BINARY*BASE_BINARY) {
		return false
	}
	if int_modular_reduce_mersenne_double_word(references) {
		return true
	}
	var dividend Modular_Dividend_Limbs
	var divisor Modular_Divisor_Limbs
	limb_mask := uint64(bits.WORD_32_MAXIMUM)
	for index := WORD_COUNT_MINIMUM; index < MODULAR_PRODUCT_LIMB_COUNT; index++ {
		word_index := index / TEXT_DIVISION_WORD_LIMB_COUNT
		if word_index < int(product.Count) {
			shift := uint(index%TEXT_DIVISION_WORD_LIMB_COUNT) *
				bits.BIT_COUNT_32_MAXIMUM
			dividend[index] = uint32(
				uint64(product.Words[word_index]) >> shift & limb_mask,
			)
		}
	}
	for index := WORD_COUNT_MINIMUM; index < MODULAR_DOUBLE_WORD_LIMB_COUNT; index++ {
		word_index := index / TEXT_DIVISION_WORD_LIMB_COUNT
		shift := uint(index%TEXT_DIVISION_WORD_LIMB_COUNT) *
			bits.BIT_COUNT_32_MAXIMUM
		divisor[index] = uint32(
			uint64(modulus.Words[word_index]) >> shift & limb_mask,
		)
	}
	if uint64(divisor[MODULAR_DOUBLE_WORD_LIMB_COUNT-WORD_COUNT_INCREMENT]) <=
		limb_mask>>bits.CARRY_MAXIMUM {
		return false
	}
	int_modular_reduce_normalized_limbs(&dividend, &divisor)
	previous_count := destination.Count
	destination.Words[WORD_COUNT_MINIMUM] = Word(uint64(dividend[WORD_COUNT_INCREMENT])<<
		bits.BIT_COUNT_32_MAXIMUM | uint64(dividend[WORD_COUNT_MINIMUM]))
	destination.Words[WORD_COUNT_INCREMENT] = Word(
		uint64(dividend[MODULAR_REMAINDER_HIGH_LIMB_INDEX])<<bits.BIT_COUNT_32_MAXIMUM |
			uint64(dividend[BASE_BINARY]),
	)
	destination.Count = Word_Count(BASE_BINARY)
	if destination.Words[WORD_COUNT_INCREMENT] == 0 {
		destination.Count = Word_Count(WORD_COUNT_INCREMENT)
		if destination.Words[WORD_COUNT_MINIMUM] == 0 {
			destination.Count = Word_Count(WORD_COUNT_MINIMUM)
		}
	}
	destination.Negative = POLARITY_NONNEGATIVE
	int_clear(destination, destination.Count, previous_count)
	return true
}

func int_modular_reduce_mersenne_double_word(
	references *Int_References,
) (matched Boolean) {
	defer func() { Boolean_Invariants(matched, "int_modular_reduce_mersenne.matched") }()
	Int_References_Invariants(references, "int_modular_reduce_mersenne.references")
	destination := references[INT_REFERENCE_DESTINATION_INDEX]
	product := references[INT_REFERENCE_LEFT_INDEX]
	modulus := references[INT_REFERENCE_RIGHT_INDEX]
	if modulus.Count != Word_Count(BASE_BINARY) {
		return false
	}
	if modulus.Words[WORD_COUNT_MINIMUM] != Word(bits.WORD_64_MAXIMUM) {
		return false
	}
	if modulus.Words[WORD_COUNT_INCREMENT] != MODULAR_MERSENNE_HIGH_WORD {
		return false
	}
	if product.Count > Word_Count(BASE_BINARY*BASE_BINARY) {
		return false
	}
	var words [BASE_BINARY * BASE_BINARY]Word
	for index := Word_Count(WORD_COUNT_MINIMUM); index < product.Count; index++ {
		words[index] = product.Words[index]
	}
	low := words[WORD_COUNT_MINIMUM]
	high := words[WORD_COUNT_INCREMENT] & MODULAR_MERSENNE_HIGH_WORD
	fold_low := words[WORD_COUNT_INCREMENT] >> WORD_BIT_INDEX_MAXIMUM
	fold_low |= words[BASE_BINARY] << WORD_COUNT_INCREMENT
	fold_high := words[BASE_BINARY] >> WORD_BIT_INDEX_MAXIMUM
	fold_high |= words[BASE_BINARY+WORD_COUNT_INCREMENT] << WORD_COUNT_INCREMENT
	sum_low := low + fold_low
	carry := Word(0)
	if sum_low < low {
		carry = Word(bits.CARRY_MAXIMUM)
	}
	sum_high := high + fold_high + carry
	fold := sum_high >> WORD_BIT_INDEX_MAXIMUM
	sum_high &= MODULAR_MERSENNE_HIGH_WORD
	if fold != 0 {
		previous := sum_low
		sum_low += fold
		if sum_low < previous {
			sum_high++
		}
	}
	fold = sum_high >> WORD_BIT_INDEX_MAXIMUM
	if fold != 0 {
		sum_high &= MODULAR_MERSENNE_HIGH_WORD
		sum_low += fold
	}
	if sum_low == Word(bits.WORD_64_MAXIMUM) {
		if sum_high == MODULAR_MERSENNE_HIGH_WORD {
			sum_low, sum_high = 0, 0
		}
	}
	previous_count := destination.Count
	destination.Words[WORD_COUNT_MINIMUM] = sum_low
	destination.Words[WORD_COUNT_INCREMENT] = sum_high
	destination.Count = Word_Count(BASE_BINARY)
	if sum_high == 0 {
		destination.Count = Word_Count(WORD_COUNT_INCREMENT)
		if sum_low == 0 {
			destination.Count = Word_Count(WORD_COUNT_MINIMUM)
		}
	}
	destination.Negative = POLARITY_NONNEGATIVE
	int_clear(destination, destination.Count, previous_count)
	return true
}

func int_modular_multiply_identity(references *Int_References) (matched Boolean) {
	defer func() { Boolean_Invariants(matched, "int_modular_multiply_identity.matched") }()
	Int_References_Invariants(references, "int_modular_multiply_identity.references")
	destination := references[INT_REFERENCE_DESTINATION_INDEX]
	left := references[INT_REFERENCE_LEFT_INDEX]
	right := references[INT_REFERENCE_RIGHT_INDEX]
	if left.Count == Word_Count(WORD_COUNT_INCREMENT) {
		if left.Words[WORD_COUNT_MINIMUM] == Word(bits.CARRY_MAXIMUM) {
			if left.Negative == POLARITY_NONNEGATIVE {
				Int_Set(destination, right)
				return true
			}
		}
	}
	if right.Count == Word_Count(WORD_COUNT_INCREMENT) {
		if right.Words[WORD_COUNT_MINIMUM] == Word(bits.CARRY_MAXIMUM) {
			if right.Negative == POLARITY_NONNEGATIVE {
				Int_Set(destination, left)
				return true
			}
		}
	}
	return false
}

func int_subtract_positive_double_word(references *Int_References) (matched Boolean) {
	defer func() {
		Boolean_Invariants(matched, "int_subtract_positive_double_word.matched")
	}()
	Int_References_Invariants(references, "int_subtract_positive_double_word.references")
	destination := references[INT_REFERENCE_DESTINATION_INDEX]
	left := references[INT_REFERENCE_LEFT_INDEX]
	right := references[INT_REFERENCE_RIGHT_INDEX]
	if left.Count != Word_Count(BASE_BINARY) {
		return false
	}
	if right.Count != Word_Count(BASE_BINARY) {
		return false
	}
	if left.Negative != POLARITY_NONNEGATIVE {
		return false
	}
	if right.Negative != POLARITY_NONNEGATIVE {
		return false
	}
	low_left, high_left := left.Words[WORD_COUNT_MINIMUM], left.Words[WORD_COUNT_INCREMENT]
	low_right, high_right := right.Words[WORD_COUNT_MINIMUM], right.Words[WORD_COUNT_INCREMENT]
	negative := POLARITY_NONNEGATIVE
	if high_left < high_right {
		low_left, low_right = low_right, low_left
		high_left, high_right = high_right, high_left
		negative = POLARITY_NEGATIVE
	} else if high_left == high_right {
		if low_left < low_right {
			low_left, low_right = low_right, low_left
			negative = POLARITY_NEGATIVE
		}
	}
	low := low_left - low_right
	borrow := Word(0)
	if low_left < low_right {
		borrow = Word(bits.CARRY_MAXIMUM)
	}
	high := high_left - high_right - borrow
	previous_count := destination.Count
	destination.Words[WORD_COUNT_MINIMUM] = low
	destination.Words[WORD_COUNT_INCREMENT] = high
	destination.Count = Word_Count(BASE_BINARY)
	if high == 0 {
		destination.Count = Word_Count(WORD_COUNT_INCREMENT)
		if low == 0 {
			destination.Count = Word_Count(WORD_COUNT_MINIMUM)
		}
	}
	destination.Negative = negative
	int_clear(destination, destination.Count, previous_count)
	return true
}

func int_multiply_double_word(references *Int_References) (matched Boolean) {
	defer func() { Boolean_Invariants(matched, "int_multiply_double_word.matched") }()
	Int_References_Invariants(references, "int_multiply_double_word.references")
	destination := references[INT_REFERENCE_DESTINATION_INDEX]
	left := references[INT_REFERENCE_LEFT_INDEX]
	right := references[INT_REFERENCE_RIGHT_INDEX]
	if left.Count != Word_Count(BASE_BINARY) {
		return false
	}
	if right.Count != Word_Count(BASE_BINARY) {
		return false
	}
	var products Double_Word_Products
	int_multiply_double_word_products(&products, references)
	high_low, low_low := products[DOUBLE_WORD_PRODUCT_LOW_INDEX].High,
		products[DOUBLE_WORD_PRODUCT_LOW_INDEX].Low
	high_cross_left, low_cross_left := products[DOUBLE_WORD_PRODUCT_CROSS_LEFT_INDEX].High,
		products[DOUBLE_WORD_PRODUCT_CROSS_LEFT_INDEX].Low
	high_cross_right, low_cross_right := products[DOUBLE_WORD_PRODUCT_CROSS_RIGHT_INDEX].High,
		products[DOUBLE_WORD_PRODUCT_CROSS_RIGHT_INDEX].Low
	high_high, low_high := products[DOUBLE_WORD_PRODUCT_HIGH_INDEX].High,
		products[DOUBLE_WORD_PRODUCT_HIGH_INDEX].Low
	word_1 := uint64(high_low) + uint64(low_cross_left)
	carry_1 := uint64(0)
	if word_1 < uint64(high_low) {
		carry_1++
	}
	previous := word_1
	word_1 += uint64(low_cross_right)
	if word_1 < previous {
		carry_1++
	}
	word_2 := uint64(high_cross_left) + uint64(high_cross_right)
	carry_2 := uint64(0)
	if word_2 < uint64(high_cross_left) {
		carry_2++
	}
	previous = word_2
	word_2 += uint64(low_high)
	if word_2 < previous {
		carry_2++
	}
	previous = word_2
	word_2 += carry_1
	if word_2 < previous {
		carry_2++
	}
	previous_count := destination.Count
	destination.Words[WORD_COUNT_MINIMUM] = Word(low_low)
	destination.Words[WORD_COUNT_INCREMENT] = Word(word_1)
	destination.Words[BASE_BINARY] = Word(word_2)
	destination.Words[BASE_BINARY+WORD_COUNT_INCREMENT] = Word(uint64(high_high) + carry_2)
	destination.Count = Word_Count(BASE_BINARY * BASE_BINARY)
	for destination.Count > Word_Count(WORD_COUNT_MINIMUM) {
		if destination.Words[destination.Count-Word_Count(WORD_COUNT_INCREMENT)] != 0 {
			break
		}
		destination.Count--
	}
	destination.Negative = POLARITY_NONNEGATIVE
	if left.Negative != right.Negative {
		destination.Negative = POLARITY_NEGATIVE
	}
	int_clear(destination, destination.Count, previous_count)
	return true
}

func int_multiply_double_word_products(
	products *Double_Word_Products, references *Int_References,
) {
	Double_Word_Products_Invariants(products, "int_multiply_double_word_products.products")
	Int_References_Invariants(references, "int_multiply_double_word_products.references")
	left := references[INT_REFERENCE_LEFT_INDEX]
	right := references[INT_REFERENCE_RIGHT_INDEX]
	left_word, right_word := uint64(left.Words[WORD_COUNT_MINIMUM]),
		uint64(right.Words[WORD_COUNT_MINIMUM])
	left_low, right_low := left_word&uint64(bits.WORD_32_MAXIMUM),
		right_word&uint64(bits.WORD_32_MAXIMUM)
	left_high, right_high := left_word>>bits.BIT_COUNT_32_MAXIMUM,
		right_word>>bits.BIT_COUNT_32_MAXIMUM
	partial := left_low * right_low
	middle_first := left_high*right_low + partial>>bits.BIT_COUNT_32_MAXIMUM
	middle_second := left_low*right_high + middle_first&uint64(bits.WORD_32_MAXIMUM)
	high := left_high*right_high + middle_first>>bits.BIT_COUNT_32_MAXIMUM +
		middle_second>>bits.BIT_COUNT_32_MAXIMUM
	products[DOUBLE_WORD_PRODUCT_LOW_INDEX].Low = Word(left_word * right_word)
	products[DOUBLE_WORD_PRODUCT_LOW_INDEX].High = Word(high)
	right_word = uint64(right.Words[WORD_COUNT_INCREMENT])
	right_low, right_high = right_word&uint64(bits.WORD_32_MAXIMUM),
		right_word>>bits.BIT_COUNT_32_MAXIMUM
	partial = left_low * right_low
	middle_first = left_high*right_low + partial>>bits.BIT_COUNT_32_MAXIMUM
	middle_second = left_low*right_high + middle_first&uint64(bits.WORD_32_MAXIMUM)
	high = left_high*right_high + middle_first>>bits.BIT_COUNT_32_MAXIMUM +
		middle_second>>bits.BIT_COUNT_32_MAXIMUM
	products[DOUBLE_WORD_PRODUCT_CROSS_LEFT_INDEX].Low = Word(left_word * right_word)
	products[DOUBLE_WORD_PRODUCT_CROSS_LEFT_INDEX].High = Word(high)
	left_word = uint64(left.Words[WORD_COUNT_INCREMENT])
	right_word = uint64(right.Words[WORD_COUNT_MINIMUM])
	left_low, right_low = left_word&uint64(bits.WORD_32_MAXIMUM),
		right_word&uint64(bits.WORD_32_MAXIMUM)
	left_high, right_high = left_word>>bits.BIT_COUNT_32_MAXIMUM,
		right_word>>bits.BIT_COUNT_32_MAXIMUM
	partial = left_low * right_low
	middle_first = left_high*right_low + partial>>bits.BIT_COUNT_32_MAXIMUM
	middle_second = left_low*right_high + middle_first&uint64(bits.WORD_32_MAXIMUM)
	high = left_high*right_high + middle_first>>bits.BIT_COUNT_32_MAXIMUM +
		middle_second>>bits.BIT_COUNT_32_MAXIMUM
	products[DOUBLE_WORD_PRODUCT_CROSS_RIGHT_INDEX].Low = Word(left_word * right_word)
	products[DOUBLE_WORD_PRODUCT_CROSS_RIGHT_INDEX].High = Word(high)
	right_word = uint64(right.Words[WORD_COUNT_INCREMENT])
	right_low, right_high = right_word&uint64(bits.WORD_32_MAXIMUM),
		right_word>>bits.BIT_COUNT_32_MAXIMUM
	partial = left_low * right_low
	middle_first = left_high*right_low + partial>>bits.BIT_COUNT_32_MAXIMUM
	middle_second = left_low*right_high + middle_first&uint64(bits.WORD_32_MAXIMUM)
	high = left_high*right_high + middle_first>>bits.BIT_COUNT_32_MAXIMUM +
		middle_second>>bits.BIT_COUNT_32_MAXIMUM
	products[DOUBLE_WORD_PRODUCT_HIGH_INDEX].Low = Word(left_word * right_word)
	products[DOUBLE_WORD_PRODUCT_HIGH_INDEX].High = Word(high)
}

// Rat_Absolute copies value and clears numerator sign.
func Rat_Absolute(destination *Rat, source *Rat) {
	Rat_Invariants(destination, "rat_absolute.destination")
	Rat_Invariants(source, "rat_absolute.source")
	numerator := &source.Integers[RAT_NUMERATOR_INDEX]
	denominator := &source.Integers[RAT_DENOMINATOR_INDEX]
	if numerator.Count == Word_Count(BASE_BINARY) {
		if denominator.Count == Word_Count(BASE_BINARY) {
			rat_copy_double_words(destination, source)
			destination.Integers[RAT_NUMERATOR_INDEX].Negative = POLARITY_NONNEGATIVE
			return
		}
	}
	Rat_Set(destination, source)
	destination.Integers[RAT_NUMERATOR_INDEX].Negative = POLARITY_NONNEGATIVE
}

// Rat_Negate copies value and flips only nonzero numerator sign.
func Rat_Negate(destination *Rat, source *Rat) {
	Rat_Invariants(destination, "rat_negate.destination")
	Rat_Invariants(source, "rat_negate.source")
	negative := POLARITY_NONNEGATIVE
	if source.Integers[RAT_NUMERATOR_INDEX].Count > WORD_COUNT_MINIMUM {
		negative = POLARITY_NEGATIVE - source.Integers[RAT_NUMERATOR_INDEX].Negative
	}
	numerator := &source.Integers[RAT_NUMERATOR_INDEX]
	denominator := &source.Integers[RAT_DENOMINATOR_INDEX]
	if numerator.Count == Word_Count(BASE_BINARY) {
		if denominator.Count == Word_Count(BASE_BINARY) {
			rat_copy_double_words(destination, source)
			destination.Integers[RAT_NUMERATOR_INDEX].Negative = negative
			return
		}
	}
	Rat_Set(destination, source)
	destination.Integers[RAT_NUMERATOR_INDEX].Negative = negative
}

func rat_copy_double_words(destination *Rat, source *Rat) {
	Rat_Invariants(destination, "rat_copy_double_words.destination")
	Rat_Invariants(source, "rat_copy_double_words.source")
	source_numerator := &source.Integers[RAT_NUMERATOR_INDEX]
	source_denominator := &source.Integers[RAT_DENOMINATOR_INDEX]
	// Public selection makes this fixed copy safe without a second shape branch.
	numerator_low := source_numerator.Words[WORD_COUNT_MINIMUM]
	numerator_high := source_numerator.Words[WORD_COUNT_INCREMENT]
	denominator_low := source_denominator.Words[WORD_COUNT_MINIMUM]
	denominator_high := source_denominator.Words[WORD_COUNT_INCREMENT]
	destination_numerator := &destination.Integers[RAT_NUMERATOR_INDEX]
	destination_denominator := &destination.Integers[RAT_DENOMINATOR_INDEX]
	previous_numerator_count := destination_numerator.Count
	previous_denominator_count := destination_denominator.Count
	destination_numerator.Words[WORD_COUNT_MINIMUM] = numerator_low
	destination_numerator.Words[WORD_COUNT_INCREMENT] = numerator_high
	destination_numerator.Count = Word_Count(BASE_BINARY)
	destination_numerator.Negative = source_numerator.Negative
	destination_denominator.Words[WORD_COUNT_MINIMUM] = denominator_low
	destination_denominator.Words[WORD_COUNT_INCREMENT] = denominator_high
	destination_denominator.Count = Word_Count(BASE_BINARY)
	destination_denominator.Negative = POLARITY_NONNEGATIVE
	if previous_numerator_count > destination_numerator.Count {
		int_clear(
			destination_numerator, destination_numerator.Count,
			previous_numerator_count,
		)
	}
	if previous_denominator_count > destination_denominator.Count {
		int_clear(
			destination_denominator, destination_denominator.Count,
			previous_denominator_count,
		)
	}
}

func rat_inverse_double_words(destination *Rat, source *Rat) {
	Rat_Invariants(destination, "rat_inverse_double_words.destination")
	Rat_Invariants(source, "rat_inverse_double_words.source")
	source_numerator := &source.Integers[RAT_NUMERATOR_INDEX]
	source_denominator := &source.Integers[RAT_DENOMINATOR_INDEX]
	// Rat_Inverse selects the exact shape before alias-safe scalar loads begin.
	numerator_low := source_denominator.Words[WORD_COUNT_MINIMUM]
	numerator_high := source_denominator.Words[WORD_COUNT_INCREMENT]
	denominator_low := source_numerator.Words[WORD_COUNT_MINIMUM]
	denominator_high := source_numerator.Words[WORD_COUNT_INCREMENT]
	negative := source_numerator.Negative
	destination_numerator := &destination.Integers[RAT_NUMERATOR_INDEX]
	destination_denominator := &destination.Integers[RAT_DENOMINATOR_INDEX]
	previous_numerator_count := destination_numerator.Count
	previous_denominator_count := destination_denominator.Count
	destination_numerator.Words[WORD_COUNT_MINIMUM] = numerator_low
	destination_numerator.Words[WORD_COUNT_INCREMENT] = numerator_high
	destination_numerator.Count = Word_Count(BASE_BINARY)
	destination_numerator.Negative = negative
	destination_denominator.Words[WORD_COUNT_MINIMUM] = denominator_low
	destination_denominator.Words[WORD_COUNT_INCREMENT] = denominator_high
	destination_denominator.Count = Word_Count(BASE_BINARY)
	destination_denominator.Negative = POLARITY_NONNEGATIVE
	if previous_numerator_count > destination_numerator.Count {
		int_clear(
			destination_numerator, destination_numerator.Count,
			previous_numerator_count,
		)
	}
	if previous_denominator_count > destination_denominator.Count {
		int_clear(
			destination_denominator, destination_denominator.Count,
			previous_denominator_count,
		)
	}
}

func rat_binary(
	destination *Rat,
	left *Rat,
	right *Rat,
	workspace *Rat_Workspace,
	operation Rat_Operation,
) (status Rat_Division_Status) {
	defer func() { Rat_Division_Status_Invariants(status, "rat_binary.status") }()
	Rat_Invariants(destination, "rat_binary.destination")
	Rat_Invariants(left, "rat_binary.left")
	Rat_Invariants(right, "rat_binary.right")
	Rat_Workspace_Invariants(workspace, "rat_binary.workspace")
	Rat_Operation_Invariants(operation, "rat_binary.operation")
	rat_load_operands(workspace, left, right)
	if operation == RAT_OPERATION_QUOTIENT {
		if workspace.Integers[RAT_RIGHT_NUMERATOR_INDEX].Count == WORD_COUNT_MINIMUM {
			return STATUS_DIVISOR_ZERO
		}
	}
	result_normalized := rat_binary_result_normalized(workspace, operation)
	multiplication := (*Int_Multiplication_Workspace)(&workspace.Multiplication)
	rat_binary_products(workspace, multiplication, operation)
	if result_normalized {
		numerator := &workspace.Integers[RAT_RESULT_NUMERATOR_INDEX]
		denominator := &workspace.Integers[RAT_RESULT_DENOMINATOR_INDEX]
		if numerator.Count > RAT_WORD_COUNT_MAXIMUM {
			return STATUS_VALUE_OVERFLOW
		}
		if denominator.Count > RAT_WORD_COUNT_MAXIMUM {
			return STATUS_VALUE_OVERFLOW
		}
		rat_commit_normalized(destination, workspace)
		return STATUS_OK
	}
	return rat_normalize(destination, workspace)
}

func rat_binary_result_normalized(
	workspace *Rat_Workspace, operation Rat_Operation,
) (normalized Boolean) {
	defer func() { Boolean_Invariants(normalized, "rat_binary_result_normalized.normalized") }()
	Rat_Workspace_Invariants(workspace, "rat_binary_result_normalized.workspace")
	Rat_Operation_Invariants(operation, "rat_binary_result_normalized.operation")
	if operation == RAT_OPERATION_MULTIPLY {
		return false
	}
	integers := &workspace.Integers
	greatest_common := (*Int_Greatest_Common_Divisor_Workspace)(
		&workspace.Greatest_Common,
	)
	left_index := RAT_LEFT_DENOMINATOR_INDEX
	right_index := RAT_RIGHT_DENOMINATOR_INDEX
	if operation == RAT_OPERATION_QUOTIENT {
		left_index = RAT_LEFT_NUMERATOR_INDEX
		right_index = RAT_RIGHT_NUMERATOR_INDEX
	}
	Int_Greatest_Common_Divisor(
		&integers[RAT_COMMON_DIVISOR_INDEX], &integers[left_index],
		&integers[right_index], greatest_common,
	)
	common_divisor := &integers[RAT_COMMON_DIVISOR_INDEX]
	if common_divisor.Count != Word_Count(WORD_COUNT_INCREMENT) {
		return false
	}
	if common_divisor.Words[WORD_COUNT_MINIMUM] != Word(bits.CARRY_MAXIMUM) {
		return false
	}
	if operation != RAT_OPERATION_QUOTIENT {
		return true
	}
	Int_Greatest_Common_Divisor(
		common_divisor, &integers[RAT_RIGHT_DENOMINATOR_INDEX],
		&integers[RAT_LEFT_DENOMINATOR_INDEX], greatest_common,
	)
	if common_divisor.Count != Word_Count(WORD_COUNT_INCREMENT) {
		return false
	}
	return common_divisor.Words[WORD_COUNT_MINIMUM] == Word(bits.CARRY_MAXIMUM)
}

func int_exponent_accumulate(
	references *Int_References, workspace *Int_Multiplication_Workspace,
) (overflow Boolean) {
	defer func() { Boolean_Invariants(overflow, "int_exponent_accumulate.overflow") }()
	Int_References_Invariants(references, "int_exponent_accumulate.references")
	Int_Multiplication_Workspace_Invariants(workspace, "int_exponent_accumulate.workspace")
	destination := references[INT_REFERENCE_DESTINATION_INDEX]
	factor := references[INT_REFERENCE_RIGHT_INDEX]
	if destination.Count == Word_Count(WORD_COUNT_INCREMENT) {
		if destination.Words[WORD_COUNT_MINIMUM] == Word(bits.CARRY_MAXIMUM) {
			Int_Set(destination, factor)
			return false
		}
	}
	status := Int_Multiply(destination, destination, factor, workspace)
	return status != Arithmetic_Status(STATUS_OK)
}

func int_square_double_word(references *Int_References) {
	Int_References_Invariants(references, "int_square_double_word.references")
	destination := references[INT_REFERENCE_DESTINATION_INDEX]
	factor := references[INT_REFERENCE_RIGHT_INDEX]
	invariant.Always(
		factor.Count == Word_Count(BASE_BINARY),
		"The symmetric square receives exactly two words.",
	)
	var products Double_Word_Products
	int_multiply_double_word_products(&products, references)
	low_square_low, low_square_high := products[DOUBLE_WORD_PRODUCT_LOW_INDEX].Low,
		products[DOUBLE_WORD_PRODUCT_LOW_INDEX].High
	cross_low, cross_high := products[DOUBLE_WORD_PRODUCT_CROSS_LEFT_INDEX].Low,
		products[DOUBLE_WORD_PRODUCT_CROSS_LEFT_INDEX].High
	high_square_low, high_square_high := products[DOUBLE_WORD_PRODUCT_HIGH_INDEX].Low,
		products[DOUBLE_WORD_PRODUCT_HIGH_INDEX].High
	word_1 := Word(low_square_high) + Word(cross_low)
	carry_1 := Word(0)
	if word_1 < Word(low_square_high) {
		carry_1++
	}
	previous := word_1
	word_1 += Word(cross_low)
	if word_1 < previous {
		carry_1++
	}
	word_2 := Word(high_square_low) + Word(cross_high)
	carry_2 := Word(0)
	if word_2 < Word(high_square_low) {
		carry_2++
	}
	previous = word_2
	word_2 += Word(cross_high)
	if word_2 < previous {
		carry_2++
	}
	previous = word_2
	word_2 += carry_1
	if word_2 < previous {
		carry_2++
	}
	previous_count := destination.Count
	destination.Words[WORD_COUNT_MINIMUM] = Word(low_square_low)
	destination.Words[WORD_COUNT_INCREMENT] = word_1
	destination.Words[BASE_BINARY] = word_2
	destination.Words[BASE_BINARY+WORD_COUNT_INCREMENT] = Word(high_square_high) + carry_2
	destination.Count = Word_Count(BASE_BINARY * BASE_BINARY)
	for destination.Words[destination.Count-Word_Count(WORD_COUNT_INCREMENT)] == 0 {
		destination.Count--
	}
	destination.Negative = POLARITY_NONNEGATIVE
	if previous_count > destination.Count {
		int_clear(destination, destination.Count, previous_count)
	}
}

func int_square_root_double_word(destination *Int, source *Int_Double_Word) {
	Int_Invariants(destination, "int_square_root_double_word.destination_initial")
	Int_Double_Word_Invariants(source, "int_square_root_double_word.source")
	high := uint64(source.Words[WORD_COUNT_INCREMENT])
	low := uint64(source.Words[WORD_COUNT_MINIMUM])
	maximum := uint64(bits.WORD_64_MAXIMUM)
	root := maximum
	if high != maximum {
		bit_count := WORD_BIT_COUNT + int(bits.Bit_Size_64(bits.Word_64(high)))
		root_shift := (bit_count + SQUARE_ROOT_DEGREE - WORD_COUNT_INCREMENT) /
			SQUARE_ROOT_DEGREE
		if root_shift < WORD_BIT_COUNT {
			root = uint64(bits.CARRY_MAXIMUM) << uint(root_shift)
		}
		converged := false
		for !converged {
			quotient, _ := bits.Divide_64(
				bits.Dividend_High_64(high), bits.Dividend_Low_64(low),
				bits.Divisor_64(root),
			)
			quotient_word := uint64(quotient)
			next := root>>uint(bits.CARRY_MAXIMUM) +
				quotient_word>>uint(bits.CARRY_MAXIMUM) +
				root&quotient_word&uint64(bits.CARRY_MAXIMUM)
			if next >= root {
				converged = true
			} else {
				root = next
			}
		}
	}
	previous_count := destination.Count
	destination.Words[WORD_COUNT_MINIMUM] = Word(root)
	destination.Count = Word_Count(WORD_COUNT_INCREMENT)
	destination.Negative = POLARITY_NONNEGATIVE
	if previous_count > destination.Count {
		int_clear(destination, destination.Count, previous_count)
	}
}

func int_primality_modular_multiply(
	references *Int_References, workspace *Int_Modular_Workspace,
) {
	Int_References_Invariants(references, "int_primality_modular_multiply.references")
	Int_Modular_Workspace_Invariants(workspace, "int_primality_modular_multiply.workspace")
	destination := references[INT_REFERENCE_DESTINATION_INDEX]
	left := references[INT_REFERENCE_LEFT_INDEX]
	right := references[INT_REFERENCE_RIGHT_INDEX]
	result := &workspace.Integers[MODULAR_EXPONENT_RESULT_INDEX]
	factor := &workspace.Integers[MODULAR_EXPONENT_FACTOR_INDEX]
	// Primality recurrences retain residues, so public re-reduction is redundant.
	Int_Set(result, left)
	Int_Set(factor, right)
	int_modular_multiply(workspace, MODULAR_MULTIPLICATION_ACCUMULATE)
	Int_Set(destination, result)
}

// Int_Compare keeps sign handling above magnitude ordering so zero needs no special magnitude.
func Int_Compare(left *Int, right *Int) (order Order) {
	defer func() { Order_Invariants(order, "int_compare.order") }()
	Int_Invariants(left, "int_compare.left")
	Int_Invariants(right, "int_compare.right")
	if left.Negative != right.Negative {
		if left.Negative == POLARITY_NEGATIVE {
			return ORDER_BEFORE
		}
		return ORDER_AFTER
	}
	if left.Count == Word_Count(BASE_BINARY) {
		if right.Count == Word_Count(BASE_BINARY) {
			if left.Words[WORD_COUNT_INCREMENT] >
				right.Words[WORD_COUNT_INCREMENT] {
				order = ORDER_AFTER
			} else if left.Words[WORD_COUNT_INCREMENT] <
				right.Words[WORD_COUNT_INCREMENT] {
				order = ORDER_BEFORE
			} else if left.Words[WORD_COUNT_MINIMUM] > right.Words[WORD_COUNT_MINIMUM] {
				order = ORDER_AFTER
			} else if left.Words[WORD_COUNT_MINIMUM] < right.Words[WORD_COUNT_MINIMUM] {
				order = ORDER_BEFORE
			}
			if left.Negative == POLARITY_NEGATIVE {
				order = -order
			}
			return order
		}
	}
	if left.Count == Word_Count(WORD_COUNT_INCREMENT) {
		if right.Count == Word_Count(WORD_COUNT_INCREMENT) {
			if left.Words[WORD_COUNT_MINIMUM] < right.Words[WORD_COUNT_MINIMUM] {
				order = ORDER_BEFORE
			} else if left.Words[WORD_COUNT_MINIMUM] > right.Words[WORD_COUNT_MINIMUM] {
				order = ORDER_AFTER
			}
			if left.Negative == POLARITY_NEGATIVE {
				order = -order
			}
			return order
		}
	}
	if left.Count < right.Count {
		order = ORDER_BEFORE
	} else if left.Count > right.Count {
		order = ORDER_AFTER
	} else {
		index := int(left.Count) - WORD_COUNT_INCREMENT
		for ; index >= WORD_COUNT_MINIMUM; index-- {
			if left.Words[index] < right.Words[index] {
				order = ORDER_BEFORE
				break
			}
			if left.Words[index] > right.Words[index] {
				order = ORDER_AFTER
				break
			}
		}
	}
	if left.Negative == POLARITY_NEGATIVE {
		order = -order
	}
	return order
}

// Int_Compare_Absolute avoids temporary copies merely to clear signs.
func Int_Compare_Absolute(left *Int, right *Int) (order Order) {
	defer func() { Order_Invariants(order, "int_compare_absolute.order") }()
	Int_Invariants(left, "int_compare_absolute.left")
	Int_Invariants(right, "int_compare_absolute.right")
	if left.Count == Word_Count(BASE_BINARY) {
		if right.Count == Word_Count(BASE_BINARY) {
			if left.Words[WORD_COUNT_INCREMENT] > right.Words[WORD_COUNT_INCREMENT] {
				return ORDER_AFTER
			}
			if left.Words[WORD_COUNT_INCREMENT] < right.Words[WORD_COUNT_INCREMENT] {
				return ORDER_BEFORE
			}
			if left.Words[WORD_COUNT_MINIMUM] > right.Words[WORD_COUNT_MINIMUM] {
				return ORDER_AFTER
			}
			if left.Words[WORD_COUNT_MINIMUM] < right.Words[WORD_COUNT_MINIMUM] {
				return ORDER_BEFORE
			}
			return ORDER_SAME
		}
	}
	if left.Count == Word_Count(WORD_COUNT_INCREMENT) {
		if right.Count == Word_Count(WORD_COUNT_INCREMENT) {
			if left.Words[WORD_COUNT_MINIMUM] < right.Words[WORD_COUNT_MINIMUM] {
				return ORDER_BEFORE
			}
			if left.Words[WORD_COUNT_MINIMUM] > right.Words[WORD_COUNT_MINIMUM] {
				return ORDER_AFTER
			}
			return ORDER_SAME
		}
	}
	if left.Count < right.Count {
		return ORDER_BEFORE
	}
	if left.Count > right.Count {
		return ORDER_AFTER
	}
	index := int(left.Count) - WORD_COUNT_INCREMENT
	for ; index >= WORD_COUNT_MINIMUM; index-- {
		if left.Words[index] < right.Words[index] {
			return ORDER_BEFORE
		}
		if left.Words[index] > right.Words[index] {
			return ORDER_AFTER
		}
	}
	return ORDER_SAME
}

func rat_commit_normalized(
	destination *Rat, workspace *Rat_Workspace,
) {
	Rat_Invariants(destination, "rat_commit_normalized.destination")
	Rat_Workspace_Invariants(workspace, "rat_commit_normalized.workspace")
	numerator := &workspace.Integers[RAT_RESULT_NUMERATOR_INDEX]
	denominator := &workspace.Integers[RAT_RESULT_DENOMINATOR_INDEX]
	invariant.Always(
		denominator.Count > WORD_COUNT_MINIMUM,
		"A normalized rational operation retains one nonzero denominator.",
	)
	invariant.Always(
		denominator.Negative == POLARITY_NONNEGATIVE,
		"Normalized rational operands produce one nonnegative denominator.",
	)
	Int_Set(&destination.Integers[RAT_NUMERATOR_INDEX], numerator)
	Int_Set(&destination.Integers[RAT_DENOMINATOR_INDEX], denominator)
}

func int_modular_exponent_commit(
	workspace *Int_Modular_Workspace, destination *Int, exponent *Int,
) {
	Int_Modular_Workspace_Invariants(workspace, "int_modular_exponent_commit.workspace")
	Int_Invariants(destination, "int_modular_exponent_commit.destination")
	Int_Invariants(exponent, "int_modular_exponent_commit.exponent")
	modulus := &workspace.Integers[MODULAR_MODULUS_INDEX]
	invariant.Always(
		modulus.Count > WORD_COUNT_MINIMUM,
		"Modular exponentiation receives one nonzero normalized modulus.",
	)
	result := &workspace.Integers[MODULAR_EXPONENT_RESULT_INDEX]
	Int_Set_Uint_64(result, Word_64(bits.CARRY_MAXIMUM))
	if modulus.Count == WORD_COUNT_INCREMENT {
		if modulus.Words[WORD_COUNT_MINIMUM] == Word(bits.CARRY_MAXIMUM) {
			int_zero(result, result.Count)
			Int_Set(destination, result)
			return
		}
	}
	if exponent.Count <= Word_Count(WORD_COUNT_INCREMENT) {
		exponent_word := Word(0)
		if exponent.Count != Word_Count(WORD_COUNT_MINIMUM) {
			exponent_word = exponent.Words[WORD_COUNT_MINIMUM]
		}
		for exponent_word != 0 {
			if exponent_word&Word(bits.CARRY_MAXIMUM) != 0 {
				int_modular_multiply(workspace, MODULAR_MULTIPLICATION_ACCUMULATE)
			}
			exponent_word >>= WORD_COUNT_INCREMENT
			if exponent_word != 0 {
				int_modular_multiply(workspace, MODULAR_MULTIPLICATION_SQUARE)
			}
		}
		Int_Set(destination, result)
		return
	}
	exponent_value := &workspace.Integers[MODULAR_EXPONENT_VALUE_INDEX]
	Int_Set(exponent_value, exponent)
	exponent_value.Negative = POLARITY_NONNEGATIVE
	bit_count := int(Int_Bit_Count(exponent_value))
	for bit_index := BIT_COUNT_MINIMUM; bit_index < bit_count; bit_index++ {
		word_index := bit_index / WORD_BIT_COUNT
		word_shift := uint(bit_index) % WORD_BIT_COUNT
		bit := exponent_value.Words[word_index] >> word_shift
		if bit&Word(bits.CARRY_MAXIMUM) != 0 {
			int_modular_multiply(workspace, MODULAR_MULTIPLICATION_ACCUMULATE)
		}
		if bit_index+WORD_COUNT_INCREMENT < bit_count {
			int_modular_multiply(workspace, MODULAR_MULTIPLICATION_SQUARE)
		}
	}
	Int_Set(destination, result)
}

func int_modular_reduce_normalized_limbs(
	dividend *Modular_Dividend_Limbs, divisor *Modular_Divisor_Limbs,
) {
	Modular_Dividend_Limbs_Invariants(
		dividend, "int_modular_reduce_normalized_limbs.dividend",
	)
	Modular_Divisor_Limbs_Invariants(
		divisor, "int_modular_reduce_normalized_limbs.divisor",
	)
	limb_mask := uint64(bits.WORD_32_MAXIMUM)
	divisor_count := MODULAR_DOUBLE_WORD_LIMB_COUNT
	for offset := divisor_count; offset >= WORD_COUNT_MINIMUM; offset-- {
		high_index := int(offset) + MODULAR_DIVISOR_LIMB_END_INDEX
		numerator := uint64(dividend[high_index])<<bits.BIT_COUNT_32_MAXIMUM |
			uint64(dividend[high_index-WORD_COUNT_INCREMENT])
		divisor_high := uint64(divisor[MODULAR_DOUBLE_WORD_LIMB_COUNT-WORD_COUNT_INCREMENT])
		quotient := numerator / divisor_high
		remainder := numerator % divisor_high
		if quotient > limb_mask {
			quotient = limb_mask
			remainder = numerator - quotient*divisor_high
		}
		for remainder <= limb_mask {
			trial := remainder<<bits.BIT_COUNT_32_MAXIMUM |
				uint64(dividend[high_index-BASE_BINARY])
			divisor_next := uint64(divisor[MODULAR_DOUBLE_WORD_LIMB_COUNT-BASE_BINARY])
			if quotient*divisor_next <= trial {
				break
			}
			quotient--
			remainder += divisor_high
		}
		borrow := uint64(0)
		for index := WORD_COUNT_MINIMUM; index < divisor_count; index++ {
			product_limb := quotient*uint64(divisor[index]) + borrow
			dividend_index := int(offset) + index
			minuend := uint64(dividend[dividend_index])
			dividend[dividend_index] = uint32(minuend - product_limb&limb_mask)
			borrow = product_limb >> bits.BIT_COUNT_32_MAXIMUM
			if minuend < product_limb&limb_mask {
				borrow++
			}
		}
		minuend := uint64(dividend[high_index])
		dividend[high_index] = uint32(minuend - borrow)
		if minuend < borrow {
			carry := uint64(0)
			for index := WORD_COUNT_MINIMUM; index < divisor_count; index++ {
				dividend_index := int(offset) + index
				sum := uint64(dividend[dividend_index]) +
					uint64(divisor[index]) + carry
				dividend[dividend_index] = uint32(sum)
				carry = sum >> bits.BIT_COUNT_32_MAXIMUM
			}
			dividend[high_index] += uint32(carry)
		}
	}
}

func int_modular_inverse_euclidean(
	workspace *Int_Modular_Workspace,
	bounded Boolean,
	result_references *Int_References,
	remainder_references *Int_References,
	coefficient_references *Int_References,
) (status Modular_Inverse_Result_Status) {
	defer func() {
		Modular_Inverse_Result_Status_Invariants(
			status, "int_modular_inverse_euclidean.status",
		)
	}()
	Int_Modular_Workspace_Invariants(workspace, "int_modular_inverse_euclidean.workspace")
	Boolean_Invariants(bounded, "int_modular_inverse_euclidean.bounded")
	Int_References_Invariants(result_references, "int_modular_inverse_euclidean.result")
	Int_References_Invariants(
		remainder_references, "int_modular_inverse_euclidean.remainders",
	)
	Int_References_Invariants(
		coefficient_references, "int_modular_inverse_euclidean.coefficient",
	)
	if bounded {
		modulus := result_references[INT_REFERENCE_LEFT_INDEX]
		if modulus.Count <= Word_Count(BASE_BINARY) {
			return int_modular_inverse_euclidean_double_word(
				workspace, result_references, remainder_references,
				coefficient_references,
			)
		}
	}
	return int_modular_inverse_euclidean_general(
		workspace, bounded, result_references, remainder_references,
		coefficient_references,
	)
}

func int_modular_inverse_euclidean_double_word(
	workspace *Int_Modular_Workspace,
	result_references *Int_References,
	remainder_references *Int_References,
	coefficient_references *Int_References,
) (status Modular_Inverse_Result_Status) {
	defer func() {
		Modular_Inverse_Result_Status_Invariants(
			status, "int_modular_inverse_euclidean_double_word.status",
		)
	}()
	Int_Modular_Workspace_Invariants(
		workspace, "int_modular_inverse_euclidean_double_word.workspace",
	)
	Int_References_Invariants(
		result_references, "int_modular_inverse_euclidean_double_word.result",
	)
	Int_References_Invariants(
		remainder_references, "int_modular_inverse_euclidean_double_word.remainders",
	)
	Int_References_Invariants(
		coefficient_references, "int_modular_inverse_euclidean_double_word.coefficient",
	)
	remainder := remainder_references[INT_REFERENCE_DESTINATION_INDEX]
	dividend := remainder_references[INT_REFERENCE_LEFT_INDEX]
	divisor := remainder_references[INT_REFERENCE_RIGHT_INDEX]
	coefficient_remainder := coefficient_references[INT_REFERENCE_DESTINATION_INDEX]
	coefficient_dividend := coefficient_references[INT_REFERENCE_LEFT_INDEX]
	coefficient_divisor := coefficient_references[INT_REFERENCE_RIGHT_INDEX]
	for divisor.Count > Word_Count(WORD_COUNT_MINIMUM) {
		if divisor.Count == Word_Count(BASE_BINARY) {
			int_modular_inverse_divide_double_word(
				result_references, remainder_references,
			)
		} else if dividend.Count == Word_Count(WORD_COUNT_INCREMENT) {
			int_modular_inverse_divide_word_dividend(
				result_references, remainder_references,
			)
		} else {
			int_modular_inverse_divide_double_word_dividend(
				(*Int_Division_Workspace)(&workspace.Division), result_references,
				remainder_references,
			)
		}
		quotient := result_references[INT_REFERENCE_RIGHT_INDEX]
		if quotient.Count == Word_Count(WORD_COUNT_INCREMENT) {
			int_modular_inverse_coefficient_double_word(
				result_references, coefficient_references,
			)
		} else {
			int_modular_inverse_coefficient_bounded(
				(*Int_Multiplication_Workspace)(&workspace.Multiplication),
				result_references, coefficient_references,
			)
		}
		dividend, divisor, remainder = divisor, remainder, dividend
		*remainder_references = Int_References{remainder, dividend, divisor}
		coefficient_dividend, coefficient_divisor, coefficient_remainder =
			coefficient_divisor, coefficient_remainder, coefficient_dividend
		*coefficient_references = Int_References{
			coefficient_remainder, coefficient_dividend, coefficient_divisor,
		}
	}
	destination := result_references[INT_REFERENCE_DESTINATION_INDEX]
	modulus := result_references[INT_REFERENCE_LEFT_INDEX]
	commit_references := Int_References{destination, modulus, dividend}
	return int_modular_inverse_commit(
		true, &commit_references, coefficient_references,
	)
}

func int_modular_inverse_divide(
	workspace *Int_Division_Workspace,
	result_references *Int_References,
	remainder_references *Int_References,
) {
	Int_Division_Workspace_Invariants(
		workspace, "int_modular_inverse_divide.workspace",
	)
	Int_References_Invariants(
		result_references, "int_modular_inverse_divide.result",
	)
	Int_References_Invariants(
		remainder_references, "int_modular_inverse_divide.remainders",
	)
	quotient := result_references[INT_REFERENCE_RIGHT_INDEX]
	remainder := remainder_references[INT_REFERENCE_DESTINATION_INDEX]
	dividend := remainder_references[INT_REFERENCE_LEFT_INDEX]
	divisor := remainder_references[INT_REFERENCE_RIGHT_INDEX]
	if divisor.Count == Word_Count(BASE_BINARY) {
		if dividend.Count == Word_Count(BASE_BINARY) {
			int_modular_inverse_divide_double_word(
				result_references, remainder_references,
			)
			return
		}
	}
	if divisor.Count == Word_Count(WORD_COUNT_INCREMENT) {
		divisor_word := divisor.Words[WORD_COUNT_MINIMUM]
		if dividend.Count == Word_Count(WORD_COUNT_INCREMENT) {
			int_set_word(quotient, dividend.Words[WORD_COUNT_MINIMUM]/divisor_word,
				POLARITY_NONNEGATIVE)
			int_set_word(remainder, dividend.Words[WORD_COUNT_MINIMUM]%divisor_word,
				POLARITY_NONNEGATIVE)
			return
		}
		if dividend.Count == Word_Count(BASE_BINARY) {
			high := dividend.Words[WORD_COUNT_INCREMENT]
			if high < divisor_word {
				result, rest := bits.Divide_64(
					bits.Dividend_High_64(high),
					bits.Dividend_Low_64(dividend.Words[WORD_COUNT_MINIMUM]),
					bits.Divisor_64(divisor_word),
				)
				int_set_word(quotient, Word(result), POLARITY_NONNEGATIVE)
				int_set_word(remainder, Word(rest), POLARITY_NONNEGATIVE)
				return
			}
		}
	}
	status := Int_Quotient_Remainder(
		quotient, remainder, dividend, divisor, workspace,
	)
	invariant.Always(
		status == Division_Status(STATUS_OK),
		"Modular Euclid divides only by its nonzero current remainder.",
	)
}

func int_modular_inverse_divide_word_dividend(
	result_references *Int_References, remainder_references *Int_References,
) {
	Int_References_Invariants(
		result_references, "int_modular_inverse_divide_word_dividend.result",
	)
	Int_References_Invariants(
		remainder_references, "int_modular_inverse_divide_word_dividend.remainders",
	)
	quotient := result_references[INT_REFERENCE_RIGHT_INDEX]
	remainder := remainder_references[INT_REFERENCE_DESTINATION_INDEX]
	dividend := remainder_references[INT_REFERENCE_LEFT_INDEX]
	divisor := remainder_references[INT_REFERENCE_RIGHT_INDEX]
	divisor_word := divisor.Words[WORD_COUNT_MINIMUM]
	quotient.Words[WORD_COUNT_MINIMUM] =
		dividend.Words[WORD_COUNT_MINIMUM] / divisor_word
	quotient.Words[WORD_COUNT_INCREMENT] = 0
	quotient.Count = Word_Count(WORD_COUNT_INCREMENT)
	quotient.Negative = POLARITY_NONNEGATIVE
	remainder_word := dividend.Words[WORD_COUNT_MINIMUM] % divisor_word
	remainder.Words[WORD_COUNT_MINIMUM] = remainder_word
	remainder.Words[WORD_COUNT_INCREMENT] = 0
	remainder.Count = Word_Count(WORD_COUNT_INCREMENT)
	if remainder_word == 0 {
		remainder.Count = Word_Count(WORD_COUNT_MINIMUM)
	}
	remainder.Negative = POLARITY_NONNEGATIVE
}

func int_modular_inverse_divide_double_word_dividend(
	workspace *Int_Division_Workspace,
	result_references *Int_References,
	remainder_references *Int_References,
) {
	Int_Division_Workspace_Invariants(
		workspace, "int_modular_inverse_divide_double_word_dividend.workspace",
	)
	Int_References_Invariants(
		result_references, "int_modular_inverse_divide_double_word_dividend.result",
	)
	Int_References_Invariants(
		remainder_references,
		"int_modular_inverse_divide_double_word_dividend.remainders",
	)
	quotient := result_references[INT_REFERENCE_RIGHT_INDEX]
	remainder := remainder_references[INT_REFERENCE_DESTINATION_INDEX]
	dividend := remainder_references[INT_REFERENCE_LEFT_INDEX]
	divisor := remainder_references[INT_REFERENCE_RIGHT_INDEX]
	divisor_word := divisor.Words[WORD_COUNT_MINIMUM]
	high := dividend.Words[WORD_COUNT_INCREMENT]
	if high < divisor_word {
		result, rest := bits.Divide_64(
			bits.Dividend_High_64(high),
			bits.Dividend_Low_64(dividend.Words[WORD_COUNT_MINIMUM]),
			bits.Divisor_64(divisor_word),
		)
		quotient.Words[WORD_COUNT_MINIMUM] = Word(result)
		quotient.Words[WORD_COUNT_INCREMENT] = 0
		quotient.Count = Word_Count(WORD_COUNT_INCREMENT)
		quotient.Negative = POLARITY_NONNEGATIVE
		remainder.Words[WORD_COUNT_MINIMUM] = Word(rest)
		remainder.Words[WORD_COUNT_INCREMENT] = 0
		remainder.Count = Word_Count(WORD_COUNT_INCREMENT)
		if rest == 0 {
			remainder.Count = Word_Count(WORD_COUNT_MINIMUM)
		}
		remainder.Negative = POLARITY_NONNEGATIVE
		return
	}
	status := Int_Quotient_Remainder(
		quotient, remainder, dividend, divisor, workspace,
	)
	invariant.Always(
		status == Division_Status(STATUS_OK),
		"The scalar modular divisor remains nonzero.",
	)
}

func int_modular_inverse_divide_double_word(
	result_references *Int_References, remainder_references *Int_References,
) {
	Int_References_Invariants(
		result_references, "int_modular_inverse_divide_double_word.result",
	)
	Int_References_Invariants(
		remainder_references, "int_modular_inverse_divide_double_word.remainders",
	)
	remainder := remainder_references[INT_REFERENCE_DESTINATION_INDEX]
	dividend := remainder_references[INT_REFERENCE_LEFT_INDEX]
	divisor := remainder_references[INT_REFERENCE_RIGHT_INDEX]
	dividend_high := dividend.Words[WORD_COUNT_INCREMENT]
	divisor_high := divisor.Words[WORD_COUNT_INCREMENT]
	factor := Int_Division_Quotient_Word(WORD_COUNT_INCREMENT)
	if divisor_high != Word(bits.WORD_64_MAXIMUM) {
		factor = Int_Division_Quotient_Word(max(
			dividend_high/(divisor_high+Word(WORD_COUNT_INCREMENT)),
			Word(WORD_COUNT_INCREMENT),
		))
	}
	int_modular_inverse_divide_double_word_commit(
		result_references, remainder_references, factor,
	)
	if remainder.Count < Word_Count(BASE_BINARY) {
		return
	}
	if remainder.Words[WORD_COUNT_INCREMENT] < divisor_high {
		return
	}
	if remainder.Words[WORD_COUNT_INCREMENT] == divisor_high {
		if remainder.Words[WORD_COUNT_MINIMUM] < divisor.Words[WORD_COUNT_MINIMUM] {
			return
		}
	}
	factor = int_divide_equal_word_count_quotient(
		(*Int_Division_Multiword_Magnitude)(dividend),
		(*Int_Division_Multiword_Magnitude)(divisor),
	)
	int_modular_inverse_divide_double_word_commit(
		result_references, remainder_references, factor,
	)
}

func int_modular_inverse_divide_double_word_commit(
	result_references *Int_References,
	remainder_references *Int_References,
	factor Int_Division_Quotient_Word,
) {
	Int_References_Invariants(
		result_references, "int_modular_inverse_divide_double_word_commit.result",
	)
	Int_References_Invariants(
		remainder_references, "int_modular_inverse_divide_double_word_commit.remainders",
	)
	Int_Division_Quotient_Word_Invariants(
		factor, "int_modular_inverse_divide_double_word_commit.factor",
	)
	quotient := result_references[INT_REFERENCE_RIGHT_INDEX]
	remainder := remainder_references[INT_REFERENCE_DESTINATION_INDEX]
	dividend := remainder_references[INT_REFERENCE_LEFT_INDEX]
	divisor := remainder_references[INT_REFERENCE_RIGHT_INDEX]
	left, right := uint64(factor), uint64(divisor.Words[WORD_COUNT_MINIMUM])
	left_low, right_low := left&TEXT_DIVISION_LIMB_MASK,
		right&TEXT_DIVISION_LIMB_MASK
	left_high, right_high := left>>TEXT_DIVISION_LIMB_BIT_COUNT,
		right>>TEXT_DIVISION_LIMB_BIT_COUNT
	partial := left_low * right_low
	middle_first := left_high*right_low + partial>>TEXT_DIVISION_LIMB_BIT_COUNT
	middle_second := left_low*right_high + middle_first&TEXT_DIVISION_LIMB_MASK
	product_low_high := left_high*right_high +
		middle_first>>TEXT_DIVISION_LIMB_BIT_COUNT +
		middle_second>>TEXT_DIVISION_LIMB_BIT_COUNT
	product_low := left * right
	product_high := uint64(factor)*uint64(divisor.Words[WORD_COUNT_INCREMENT]) +
		product_low_high
	minuend := uint64(dividend.Words[WORD_COUNT_MINIMUM])
	difference_low := minuend - product_low
	borrow := ((^minuend & product_low) |
		(^(minuend ^ product_low) & difference_low)) >> WORD_BIT_INDEX_MAXIMUM
	difference_high := uint64(dividend.Words[WORD_COUNT_INCREMENT]) -
		product_high - borrow
	previous_quotient_count := quotient.Count
	quotient.Words[WORD_COUNT_MINIMUM] = Word(factor)
	quotient.Words[WORD_COUNT_INCREMENT] = 0
	quotient.Count = Word_Count(WORD_COUNT_INCREMENT)
	quotient.Negative = POLARITY_NONNEGATIVE
	if previous_quotient_count > Word_Count(BASE_BINARY) {
		int_clear(quotient, quotient.Count, previous_quotient_count)
	}
	previous_count := remainder.Count
	remainder.Words[WORD_COUNT_MINIMUM] = Word(difference_low)
	remainder.Words[WORD_COUNT_INCREMENT] = Word(difference_high)
	remainder.Count = Word_Count(BASE_BINARY)
	if difference_high == 0 {
		remainder.Count = Word_Count(WORD_COUNT_INCREMENT)
		if difference_low == 0 {
			remainder.Count = Word_Count(WORD_COUNT_MINIMUM)
		}
	}
	remainder.Negative = POLARITY_NONNEGATIVE
	if previous_count > Word_Count(BASE_BINARY) {
		int_clear(remainder, remainder.Count, previous_count)
	}
}

func int_modular_inverse_coefficient_double_word(
	result_references *Int_References, coefficient_references *Int_References,
) {
	Int_References_Invariants(
		result_references, "int_modular_inverse_coefficient_double_word.result",
	)
	Int_References_Invariants(
		coefficient_references, "int_modular_inverse_coefficient_double_word.coefficient",
	)
	quotient := result_references[INT_REFERENCE_RIGHT_INDEX]
	destination := coefficient_references[INT_REFERENCE_DESTINATION_INDEX]
	dividend := coefficient_references[INT_REFERENCE_LEFT_INDEX]
	coefficient := coefficient_references[INT_REFERENCE_RIGHT_INDEX]
	factor := uint64(quotient.Words[WORD_COUNT_MINIMUM])
	coefficient_low := uint64(0)
	coefficient_high := uint64(0)
	if coefficient.Count > Word_Count(WORD_COUNT_MINIMUM) {
		coefficient_low = uint64(coefficient.Words[WORD_COUNT_MINIMUM])
	}
	if coefficient.Count > Word_Count(WORD_COUNT_INCREMENT) {
		coefficient_high = uint64(coefficient.Words[WORD_COUNT_INCREMENT])
	}
	left_low := coefficient_low & TEXT_DIVISION_LIMB_MASK
	left_high := coefficient_low >> TEXT_DIVISION_LIMB_BIT_COUNT
	partial := left_low * factor
	middle := left_high*factor + partial>>TEXT_DIVISION_LIMB_BIT_COUNT
	product_high := middle >> TEXT_DIVISION_LIMB_BIT_COUNT
	if factor > TEXT_DIVISION_LIMB_MASK {
		right_low := factor & TEXT_DIVISION_LIMB_MASK
		right_high := factor >> TEXT_DIVISION_LIMB_BIT_COUNT
		partial = left_low * right_low
		middle = left_high*right_low + partial>>TEXT_DIVISION_LIMB_BIT_COUNT
		middle_second := left_low*right_high + middle&TEXT_DIVISION_LIMB_MASK
		product_high = left_high*right_high + middle>>TEXT_DIVISION_LIMB_BIT_COUNT +
			middle_second>>TEXT_DIVISION_LIMB_BIT_COUNT
	}
	product_low := coefficient_low * factor
	dividend_low := uint64(0)
	dividend_high := uint64(0)
	if dividend.Count > Word_Count(WORD_COUNT_MINIMUM) {
		dividend_low = uint64(dividend.Words[WORD_COUNT_MINIMUM])
	}
	if dividend.Count > Word_Count(WORD_COUNT_INCREMENT) {
		dividend_high = uint64(dividend.Words[WORD_COUNT_INCREMENT])
	}
	result_low := product_low + dividend_low
	carry := uint64(0)
	if result_low < product_low {
		carry = uint64(WORD_COUNT_INCREMENT)
	}
	result_high := coefficient_high*factor + product_high + dividend_high + carry
	previous_count := destination.Count
	destination.Words[WORD_COUNT_MINIMUM] = Word(result_low)
	destination.Words[WORD_COUNT_INCREMENT] = Word(result_high)
	destination.Count = Word_Count(BASE_BINARY)
	if result_high == 0 {
		destination.Count = Word_Count(WORD_COUNT_INCREMENT)
		if result_low == 0 {
			destination.Count = Word_Count(WORD_COUNT_MINIMUM)
		}
	}
	destination.Negative = POLARITY_NEGATIVE
	if coefficient.Negative == POLARITY_NEGATIVE {
		destination.Negative = POLARITY_NONNEGATIVE
	}
	if previous_count > Word_Count(BASE_BINARY) {
		int_clear(destination, destination.Count, previous_count)
	}
}

func int_binomial_divide(workspace *Int_Product_Workspace) {
	Int_Product_Workspace_Invariants(workspace, "int_binomial_divide.workspace")
	accumulator := &workspace.Integers[PRODUCT_ACCUMULATOR_INDEX]
	divisor := &workspace.Integers[PRODUCT_FACTOR_INDEX]
	invariant.Always(
		divisor.Count == WORD_COUNT_INCREMENT,
		"Reduced binomial denominator fits one nonzero word.",
	)
	denominator := uint64(divisor.Words[WORD_COUNT_MINIMUM])
	remainder := uint64(0)
	if denominator <= uint64(bits.WORD_32_MAXIMUM) {
		limb_mask := uint64(bits.WORD_32_MAXIMUM)
		for word_index := int(accumulator.Count) - 1; word_index >= 0; word_index-- {
			word := uint64(accumulator.Words[word_index])
			high_dividend := remainder<<bits.BIT_COUNT_32_MAXIMUM |
				word>>bits.BIT_COUNT_32_MAXIMUM
			quotient_high := high_dividend / denominator
			remainder = high_dividend % denominator
			low_dividend := remainder<<bits.BIT_COUNT_32_MAXIMUM |
				word&limb_mask
			quotient_low := low_dividend / denominator
			remainder = low_dividend % denominator
			accumulator.Words[word_index] = Word(
				quotient_high<<bits.BIT_COUNT_32_MAXIMUM | quotient_low,
			)
		}
	} else {
		for word_index := int(accumulator.Count) - 1; word_index >= 0; word_index-- {
			word := uint64(accumulator.Words[word_index])
			quotient := uint64(0)
			for bit_index := WORD_BIT_INDEX_MAXIMUM; bit_index >= 0; bit_index-- {
				// Symmetry bounds divisor; shifting keeps every bit.
				remainder <<= bits.CARRY_MAXIMUM
				remainder |= word >> uint(bit_index) & uint64(bits.CARRY_MAXIMUM)
				if remainder >= denominator {
					remainder -= denominator
					quotient |= uint64(bits.CARRY_MAXIMUM) << uint(bit_index)
				}
			}
			accumulator.Words[word_index] = Word(quotient)
		}
	}
	invariant.Always(
		remainder == 0,
		"Reduced binomial denominator divides prior coefficient exactly.",
	)
	for accumulator.Count > WORD_COUNT_MINIMUM {
		high_index := int(accumulator.Count) - WORD_COUNT_INCREMENT
		if accumulator.Words[high_index] != 0 {
			break
		}
		accumulator.Count--
	}
}

// TEXT_DIVISION_WORD_LIMB_COUNT splits one word into two safe division limbs.
const TEXT_DIVISION_WORD_LIMB_COUNT = 2

// TEXT_DIVISION_LIMB_BIT_COUNT keeps remainder-plus-limb division inside one word.
const TEXT_DIVISION_LIMB_BIT_COUNT = WORD_BIT_COUNT / TEXT_DIVISION_WORD_LIMB_COUNT

// TEXT_DIVISION_LIMB_MASK selects one low half-word limb.
const TEXT_DIVISION_LIMB_MASK = uint64(bits.WORD_64_MAXIMUM) >> TEXT_DIVISION_LIMB_BIT_COUNT

// TEXT_DECIMAL_CHUNK_DIGIT_COUNT is the decimal width below one division limb.
const TEXT_DECIMAL_CHUNK_DIGIT_COUNT = TEXT_DIVISION_LIMB_BIT_COUNT *
	DECIMAL_DIGIT_BINARY_LOGARITHM_CEILING / DECIMAL_DIGIT_BINARY_LOGARITHM_SCALE

// TEXT_DECIMAL_CHUNK_DIVISOR removes that complete decimal width per division pass.
const TEXT_DECIMAL_CHUNK_DIVISOR = BASE_DECIMAL * BASE_DECIMAL * BASE_DECIMAL *
	BASE_DECIMAL * BASE_DECIMAL * BASE_DECIMAL * BASE_DECIMAL * BASE_DECIMAL * BASE_DECIMAL

// FLOAT_SQUARE_ROOT_TRIPLE_WORD_PRECISION selects the common three-word result width.
const FLOAT_SQUARE_ROOT_TRIPLE_WORD_PRECISION = (BASE_BINARY + WORD_COUNT_INCREMENT) *
	WORD_BIT_COUNT

// DECIMAL_TEXT_DIGIT_COUNT_MAXIMUM bounds one full-width magnitude in base ten.
const DECIMAL_TEXT_DIGIT_COUNT_MAXIMUM = BIT_COUNT_MAXIMUM*
	DECIMAL_DIGIT_BINARY_LOGARITHM_CEILING/DECIMAL_DIGIT_BINARY_LOGARITHM_SCALE +
	WORD_COUNT_INCREMENT

// Decimal_Text_Digit_Count counts one nonnegative decimal magnitude representation.
type Decimal_Text_Digit_Count int

// Decimal_Text_Digit_Count_Invariants binds zero text through full-width decimal text.
func Decimal_Text_Digit_Count_Invariants(
	value Decimal_Text_Digit_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), SIGN_BYTE_COUNT_MAXIMUM, DECIMAL_TEXT_DIGIT_COUNT_MAXIMUM,
		).
		Ensure()
}

// Float_Reference prevents a second invariant chain for one already checked Float.
type Float_Reference [WORD_COUNT_INCREMENT]*Float

// Float_Reference_Invariants rejects an absent checked Float.
func Float_Reference_Invariants(value *Float_Reference, _ invariant.Namespace) {
	invariant.Always(len(value) == WORD_COUNT_INCREMENT, "The Float reference is complete.")
	invariant.Always(value[WORD_COUNT_MINIMUM] != nil, "The Float reference exists.")
}

// Float_Storage_Reference carries one writable Float slot without asserting old scratch state.
type Float_Storage_Reference [WORD_COUNT_INCREMENT]*Float

// Float_Storage_Reference_Invariants rejects absent writable Float storage.
func Float_Storage_Reference_Invariants(
	value *Float_Storage_Reference, _ invariant.Namespace,
) {
	invariant.Always(len(value) == WORD_COUNT_INCREMENT, "The Float storage is complete.")
	invariant.Always(value[WORD_COUNT_MINIMUM] != nil, "The Float storage exists.")
}

// Float_Square_Root_Workspace_Reference carries already-validated restoring storage.
type Float_Square_Root_Workspace_Reference [WORD_COUNT_INCREMENT]*Float_Square_Root_Workspace

// Float_Square_Root_Workspace_Reference_Invariants rejects absent restoring storage.
func Float_Square_Root_Workspace_Reference_Invariants(
	value *Float_Square_Root_Workspace_Reference, _ invariant.Namespace,
) {
	invariant.Always(len(value) == WORD_COUNT_INCREMENT, "The square-root storage is complete.")
	invariant.Always(value[WORD_COUNT_MINIMUM] != nil, "The square-root storage exists.")
}

// Float_Square_Root_Pair_Reference carries one already-validated radix-four digit.
type Float_Square_Root_Pair_Reference [WORD_COUNT_INCREMENT]*Float_Square_Root_Pair

// Float_Square_Root_Pair_Reference_Invariants rejects an absent radix-four digit.
func Float_Square_Root_Pair_Reference_Invariants(
	value *Float_Square_Root_Pair_Reference, _ invariant.Namespace,
) {
	invariant.Always(len(value) == WORD_COUNT_INCREMENT, "The square-root pair is complete.")
	invariant.Always(value[WORD_COUNT_MINIMUM] != nil, "The square-root pair exists.")
}

// Float_Square_Root_Count_Reference carries one already-validated active word count.
type Float_Square_Root_Count_Reference [WORD_COUNT_INCREMENT]*Float_Square_Root_Active_Word_Count

// Float_Square_Root_Count_Reference_Invariants rejects an absent active word count.
func Float_Square_Root_Count_Reference_Invariants(
	value *Float_Square_Root_Count_Reference, _ invariant.Namespace,
) {
	invariant.Always(len(value) == WORD_COUNT_INCREMENT, "The square-root count is complete.")
	invariant.Always(value[WORD_COUNT_MINIMUM] != nil, "The square-root count exists.")
}

// Float_Gob_Encoding_Reference preserves validated bounds across the finite fast path.
type Float_Gob_Encoding_Reference [WORD_COUNT_INCREMENT]*Float_Gob_Encoding

// Float_Gob_Encoding_Reference_Invariants rejects absent checked encoding storage.
func Float_Gob_Encoding_Reference_Invariants(
	value *Float_Gob_Encoding_Reference, _ invariant.Namespace,
) {
	invariant.Always(
		len(value) == WORD_COUNT_INCREMENT,
		"The float Gob encoding reference is complete.",
	)
	invariant.Always(
		value[WORD_COUNT_MINIMUM] != nil,
		"The float Gob encoding reference exists.",
	)
}

func float_gob_encode_exact_double_word(
	destination_reference *Float_Gob_Encoding_Reference, value_reference *Float_Reference,
) (encoded Boolean) {
	defer func() {
		Boolean_Invariants(encoded, "float_gob_encode_exact_double_word.encoded")
	}()
	Float_Gob_Encoding_Reference_Invariants(
		destination_reference, "float_gob_encode_exact_double_word.destination",
	)
	Float_Reference_Invariants(value_reference, "float_gob_encode_exact_double_word.value")
	destination := *destination_reference[WORD_COUNT_MINIMUM]
	value := value_reference[WORD_COUNT_MINIMUM]
	if value.Mantissa.Count != Word_Count(BASE_BINARY) {
		return false
	}
	bit_index := int(value.Exponent) - WORD_BIT_COUNT - WORD_COUNT_INCREMENT
	if uint(bit_index) >= uint(WORD_BIT_COUNT) {
		return false
	}
	minimum := Word(bits.CARRY_MAXIMUM) << uint(bit_index)
	high := value.Mantissa.Words[WORD_COUNT_INCREMENT]
	if high&minimum == 0 {
		return false
	}
	if bit_index < WORD_BIT_INDEX_MAXIMUM {
		if high >= minimum<<WORD_COUNT_INCREMENT {
			return false
		}
	}
	low := value.Mantissa.Words[WORD_COUNT_MINIMUM]
	shift := uint(BASE_BINARY*WORD_BIT_COUNT - int(value.Exponent))
	if shift != 0 {
		high = high<<shift | low>>(WORD_BIT_COUNT-shift)
		low <<= shift
	}
	low_position := FLOAT_GOB_MANTISSA_OFFSET
	low_position += WORD_BYTE_COUNT
	for byte_count := WORD_BYTE_COUNT; byte_count > 0; byte_count-- {
		completed_byte_count := WORD_BYTE_COUNT - byte_count
		byte_shift := uint(
			(byte_count - WORD_COUNT_INCREMENT) * bits.BIT_COUNT_8_MAXIMUM,
		)
		high_position := FLOAT_GOB_MANTISSA_OFFSET
		high_position += completed_byte_count
		word_position := low_position
		word_position += completed_byte_count
		destination[high_position] = byte(high >> byte_shift)
		destination[word_position] = byte(low >> byte_shift)
	}
	return true
}

func int_text_decimal_digits(
	value *Int, workspace *Int_Text_Workspace,
) (digit_count Decimal_Text_Digit_Count) {
	defer func() {
		Decimal_Text_Digit_Count_Invariants(
			digit_count, "int_text_decimal_digits.digit_count",
		)
	}()
	Int_Invariants(value, "int_text_decimal_digits.value")
	Int_Text_Workspace_Invariants(workspace, "int_text_decimal_digits.workspace")
	for index := WORD_COUNT_MINIMUM; index < int(value.Count); index++ {
		workspace.Words[index] = value.Words[index]
	}
	word_count := int(value.Count)
	if word_count == WORD_COUNT_MINIMUM {
		workspace.Digits[WORD_COUNT_MINIMUM] = '0'
		return SIGN_BYTE_COUNT_MAXIMUM
	}
	divisor := uint64(TEXT_DECIMAL_CHUNK_DIVISOR)
	for word_count > WORD_COUNT_MINIMUM {
		high_index := word_count - WORD_COUNT_INCREMENT
		high_word := uint64(workspace.Words[high_index])
		workspace.Words[high_index] = Word(high_word / divisor)
		remainder := high_word % divisor
		for index := high_index - WORD_COUNT_INCREMENT; index >= 0; index-- {
			word := uint64(workspace.Words[index])
			high_limb := word >> TEXT_DIVISION_LIMB_BIT_COUNT
			high_dividend := remainder<<TEXT_DIVISION_LIMB_BIT_COUNT | high_limb
			quotient_high := high_dividend / divisor
			remainder = high_dividend % divisor
			low_limb := word & TEXT_DIVISION_LIMB_MASK
			low_dividend := remainder<<TEXT_DIVISION_LIMB_BIT_COUNT | low_limb
			quotient_low := low_dividend / divisor
			remainder = low_dividend % divisor
			workspace.Words[index] = Word(
				quotient_high<<TEXT_DIVISION_LIMB_BIT_COUNT | quotient_low,
			)
		}
		for word_count > WORD_COUNT_MINIMUM {
			if workspace.Words[word_count-WORD_COUNT_INCREMENT] != 0 {
				break
			}
			word_count--
		}
		minimum := TEXT_DECIMAL_CHUNK_DIGIT_COUNT
		if word_count == WORD_COUNT_MINIMUM {
			minimum = SIGN_BYTE_COUNT_MAXIMUM
		}
		chunk_count := WORD_COUNT_MINIMUM
		for remainder > 0 {
			workspace.Digits[digit_count] = byte(remainder%BASE_DECIMAL) + '0'
			digit_count++
			chunk_count++
			remainder /= BASE_DECIMAL
		}
		for chunk_count < minimum {
			workspace.Digits[digit_count] = '0'
			digit_count++
			chunk_count++
		}
	}
	return digit_count
}

func float_text_source_set(
	destination_reference *Float_Storage_Reference, source_reference *Float_Reference,
) {
	Float_Storage_Reference_Invariants(
		destination_reference, "float_text_source_set.destination",
	)
	Float_Reference_Invariants(source_reference, "float_text_source_set.source")
	destination := destination_reference[WORD_COUNT_MINIMUM]
	source := source_reference[WORD_COUNT_MINIMUM]
	destination.Precision = source.Precision
	destination.Mode = source.Mode
	destination.Accuracy = source.Accuracy
	destination.Form = source.Form
	destination.Negative = source.Negative
	destination.Mantissa.Negative = source.Mantissa.Negative
	destination.Mantissa.Count = source.Mantissa.Count
	for index := WORD_COUNT_MINIMUM; index < int(source.Mantissa.Count); index++ {
		destination.Mantissa.Words[index] = source.Mantissa.Words[index]
	}
	destination.Exponent = source.Exponent
}

func float_text_general_integer(
	workspace *Float_Text_Workspace, value_reference *Float_Reference, format Float_Text_Format,
	precision Float_Text_Precision,
) (encoded Boolean) {
	defer func() {
		Boolean_Invariants(encoded, "float_text_general_integer.encoded")
	}()
	Float_Text_Workspace_Invariants(workspace, "float_text_general_integer.workspace")
	Float_Reference_Invariants(value_reference, "float_text_general_integer.value")
	Float_Text_Format_Invariants(format, "float_text_general_integer.format")
	Float_Text_Precision_Invariants(precision, "float_text_general_integer.precision")
	value := value_reference[WORD_COUNT_MINIMUM]
	if format < FLOAT_TEXT_KIND_DECIMAL_GENERAL {
		return false
	}
	if precision < SIGN_BYTE_COUNT_MAXIMUM {
		return false
	}
	if value.Form != FLOAT_FORM_FINITE {
		return false
	}
	if value.Mantissa.Count == WORD_COUNT_MINIMUM {
		return false
	}
	mantissa := (*Int)(&value.Mantissa)
	if int(value.Exponent) != int(Int_Bit_Count(mantissa)) {
		return false
	}
	text := &workspace.Text[FLOAT_TEXT_MANTISSA_WORKSPACE_INDEX]
	digit_count := int(int_text_decimal_digits(mantissa, text))
	if digit_count-WORD_COUNT_INCREMENT < int(precision) {
		return false
	}
	index := WORD_COUNT_MINIMUM
	if value.Negative == POLARITY_NEGATIVE {
		workspace.Output[index] = '-'
		index++
	}
	workspace.Control[FLOAT_TEXT_OUTPUT_COUNT_INDEX] = index
	workspace.Control[FLOAT_TEXT_DECIMAL_INDEX] = FLOAT_DECIMAL_VALUE_INDEX
	decimal := &workspace.Decimals[FLOAT_DECIMAL_VALUE_INDEX]
	decimal.Control = [FLOAT_DECIMAL_CONTROL_COUNT]int{}
	decimal.Control[FLOAT_DECIMAL_EXPONENT_INDEX] = digit_count
	low_zero_count := WORD_COUNT_MINIMUM
	for low_zero_count < digit_count {
		if text.Digits[low_zero_count] != '0' {
			break
		}
		low_zero_count++
	}
	significant_count := digit_count - low_zero_count
	decimal.Control[FLOAT_DECIMAL_DIGIT_COUNT_INDEX] = significant_count
	completed_count := WORD_COUNT_MINIMUM
	for completed_count < significant_count {
		remaining_count := digit_count - completed_count
		source_index := remaining_count - WORD_COUNT_INCREMENT
		decimal.Digits[completed_count] = text.Digits[source_index]
		completed_count++
	}
	float_text_decimal_general(workspace, false)
	return true
}

// Float_Text_Into writes one complete bounded stdlib representation transactionally.
func Float_Text_Into(
	destination Text, value *Float, format_unvalidated Float_Text_Format_Unvalidated,
	precision_unvalidated Float_Text_Precision_Unvalidated,
	workspace *Float_Text_Workspace,
) (count Float_Text_Count, status Float_Text_Status) {
	defer func() {
		Float_Text_Count_Invariants(count, "float_text_into.count")
		Float_Text_Status_Invariants(status, "float_text_into.status")
	}()
	Text_Invariants(destination, "float_text_into.destination")
	Float_Invariants(value, "float_text_into.value")
	Float_Text_Format_Unvalidated_Invariants(format_unvalidated, "float_text_into.format")
	Float_Text_Precision_Unvalidated_Invariants(
		precision_unvalidated, "float_text_into.precision",
	)
	Float_Text_Workspace_Invariants(workspace, "float_text_into.workspace")
	format, validation := Float_Text_Format_Validate(format_unvalidated)
	if validation != Validation_Status(STATUS_OK) {
		return 0, STATUS_INPUT_INVALID
	}
	precision, validation := Float_Text_Precision_Validate(precision_unvalidated)
	if validation != Validation_Status(STATUS_OK) {
		return 0, STATUS_INPUT_INVALID
	}
	workspace.Control[FLOAT_TEXT_PRECISION_INDEX] = int(precision)
	workspace.Control[FLOAT_TEXT_FORMAT_INDEX] = int(format)
	value_reference := Float_Reference{value}
	if !float_text_general_integer(workspace, &value_reference, format, precision) {
		destination_reference := Float_Storage_Reference{
			&workspace.Values[FLOAT_TEXT_SOURCE_INDEX],
		}
		float_text_source_set(&destination_reference, &value_reference)
		if format == FLOAT_TEXT_KIND_BINARY {
			float_text_binary(workspace)
		}
		if format == FLOAT_TEXT_KIND_HEXADECIMAL_FRACTION {
			float_text_hexadecimal_fraction(workspace)
		}
		if format == FLOAT_TEXT_KIND_HEXADECIMAL {
			float_text_hexadecimal(workspace)
		}
		if format >= FLOAT_TEXT_KIND_DECIMAL_EXPONENT {
			float_text_decimal(workspace)
		}
	}
	count = Float_Text_Count(workspace.Control[FLOAT_TEXT_OUTPUT_COUNT_INDEX])
	if len(destination) < int(count) {
		return count, STATUS_DESTINATION_TOO_SMALL
	}
	copy(destination[:count], workspace.Output[:count])
	return count, STATUS_OK
}

func float_square_root_digit_one(
	workspace_reference *Float_Square_Root_Workspace_Reference,
	pair_reference *Float_Square_Root_Pair_Reference,
) {
	Float_Square_Root_Workspace_Reference_Invariants(
		workspace_reference, "float_square_root_digit_one.workspace",
	)
	Float_Square_Root_Pair_Reference_Invariants(
		pair_reference, "float_square_root_digit_one.pair",
	)
	workspace := workspace_reference[WORD_COUNT_MINIMUM]
	pair := pair_reference[WORD_COUNT_MINIMUM]
	root_word := workspace.Root[WORD_COUNT_MINIMUM]
	remainder_word := workspace.Remainder[WORD_COUNT_MINIMUM] <<
		FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT
	remainder_word |= Word(*pair)
	candidate_word := root_word<<FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT | Word(BIT_SET)
	workspace.Candidate[WORD_COUNT_MINIMUM] = candidate_word
	if remainder_word >= candidate_word {
		remainder_word -= candidate_word
		root_word = root_word<<WORD_COUNT_INCREMENT | Word(BIT_SET)
	} else {
		root_word <<= WORD_COUNT_INCREMENT
	}
	workspace.Remainder[WORD_COUNT_MINIMUM] = remainder_word
	workspace.Root[WORD_COUNT_MINIMUM] = root_word
}

func float_square_root_digit_two(
	workspace_reference *Float_Square_Root_Workspace_Reference,
	pair_reference *Float_Square_Root_Pair_Reference,
) {
	Float_Square_Root_Workspace_Reference_Invariants(
		workspace_reference, "float_square_root_digit_two.workspace",
	)
	Float_Square_Root_Pair_Reference_Invariants(
		pair_reference, "float_square_root_digit_two.pair",
	)
	workspace := workspace_reference[WORD_COUNT_MINIMUM]
	pair := pair_reference[WORD_COUNT_MINIMUM]
	root_low := workspace.Root[WORD_COUNT_MINIMUM]
	root_high := workspace.Root[WORD_COUNT_INCREMENT]
	root_low_carry := root_low >> WORD_BIT_INDEX_MAXIMUM
	remainder_low_initial := workspace.Remainder[WORD_COUNT_MINIMUM]
	remainder_low := remainder_low_initial<<FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT |
		Word(*pair)
	remainder_high := workspace.Remainder[WORD_COUNT_INCREMENT]<<
		FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT |
		remainder_low_initial>>(WORD_BIT_COUNT-FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT)
	candidate_low := root_low<<FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT | Word(BIT_SET)
	candidate_high := root_high<<FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT |
		root_low>>(WORD_BIT_COUNT-FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT)
	workspace.Candidate[WORD_COUNT_MINIMUM] = candidate_low
	workspace.Candidate[WORD_COUNT_INCREMENT] = candidate_high
	subtract := remainder_high > candidate_high
	if remainder_high == candidate_high {
		subtract = remainder_low >= candidate_low
	}
	if subtract {
		borrow := Word(bits.CARRY_MINIMUM)
		if remainder_low < candidate_low {
			borrow = Word(bits.CARRY_MAXIMUM)
		}
		remainder_low -= candidate_low
		remainder_high = remainder_high - candidate_high - borrow
		root_low = root_low<<WORD_COUNT_INCREMENT | Word(BIT_SET)
	} else {
		root_low <<= WORD_COUNT_INCREMENT
	}
	workspace.Remainder[WORD_COUNT_MINIMUM] = remainder_low
	workspace.Remainder[WORD_COUNT_INCREMENT] = remainder_high
	workspace.Root[WORD_COUNT_MINIMUM] = root_low
	workspace.Root[WORD_COUNT_INCREMENT] =
		root_high<<WORD_COUNT_INCREMENT | root_low_carry
}

func float_square_root_digit_three(
	workspace_reference *Float_Square_Root_Workspace_Reference,
	pair_reference *Float_Square_Root_Pair_Reference,
) {
	Float_Square_Root_Workspace_Reference_Invariants(
		workspace_reference, "float_square_root_digit_three.workspace",
	)
	Float_Square_Root_Pair_Reference_Invariants(
		pair_reference, "float_square_root_digit_three.pair",
	)
	workspace := workspace_reference[WORD_COUNT_MINIMUM]
	pair := pair_reference[WORD_COUNT_MINIMUM]
	root_low := workspace.Root[WORD_COUNT_MINIMUM]
	root_middle := workspace.Root[WORD_COUNT_INCREMENT]
	root_high := workspace.Root[BASE_BINARY]
	root_low_carry := root_low >> WORD_BIT_INDEX_MAXIMUM
	root_middle_carry := root_middle >> WORD_BIT_INDEX_MAXIMUM
	remainder_low_initial := workspace.Remainder[WORD_COUNT_MINIMUM]
	remainder_middle_initial := workspace.Remainder[WORD_COUNT_INCREMENT]
	remainder_low := remainder_low_initial<<FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT |
		Word(*pair)
	remainder_middle := remainder_middle_initial<<FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT |
		remainder_low_initial>>(WORD_BIT_COUNT-FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT)
	remainder_high := workspace.Remainder[BASE_BINARY]<<
		FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT |
		remainder_middle_initial>>(WORD_BIT_COUNT-FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT)
	candidate_low := root_low<<FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT | Word(BIT_SET)
	candidate_middle := root_middle<<FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT |
		root_low>>(WORD_BIT_COUNT-FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT)
	candidate_high := root_high<<FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT |
		root_middle>>(WORD_BIT_COUNT-FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT)
	workspace.Candidate[WORD_COUNT_MINIMUM] = candidate_low
	workspace.Candidate[WORD_COUNT_INCREMENT] = candidate_middle
	workspace.Candidate[BASE_BINARY] = candidate_high
	subtract := remainder_high > candidate_high
	if remainder_high == candidate_high {
		subtract = remainder_middle > candidate_middle
		if remainder_middle == candidate_middle {
			subtract = remainder_low >= candidate_low
		}
	}
	if subtract {
		difference_low := remainder_low - candidate_low
		borrow := ((^remainder_low & candidate_low) |
			(^(remainder_low ^ candidate_low) & difference_low)) >>
			WORD_BIT_INDEX_MAXIMUM
		difference_middle := remainder_middle - candidate_middle - borrow
		borrow = ((^remainder_middle & candidate_middle) |
			(^(remainder_middle ^ candidate_middle) & difference_middle)) >>
			WORD_BIT_INDEX_MAXIMUM
		remainder_low = difference_low
		remainder_middle = difference_middle
		remainder_high = remainder_high - candidate_high - borrow
		root_low = root_low<<WORD_COUNT_INCREMENT | Word(BIT_SET)
	} else {
		root_low <<= WORD_COUNT_INCREMENT
	}
	workspace.Remainder[WORD_COUNT_MINIMUM] = remainder_low
	workspace.Remainder[WORD_COUNT_INCREMENT] = remainder_middle
	workspace.Remainder[BASE_BINARY] = remainder_high
	workspace.Root[WORD_COUNT_MINIMUM] = root_low
	workspace.Root[WORD_COUNT_INCREMENT] = root_middle<<WORD_COUNT_INCREMENT |
		root_low_carry
	workspace.Root[BASE_BINARY] = root_high<<WORD_COUNT_INCREMENT |
		root_middle_carry
}

func float_square_root_digit_many(
	workspace_reference *Float_Square_Root_Workspace_Reference,
	count_reference *Float_Square_Root_Count_Reference,
	pair_reference *Float_Square_Root_Pair_Reference,
) {
	Float_Square_Root_Workspace_Reference_Invariants(
		workspace_reference, "float_square_root_digit_many.workspace",
	)
	Float_Square_Root_Count_Reference_Invariants(
		count_reference, "float_square_root_digit_many.count",
	)
	Float_Square_Root_Pair_Reference_Invariants(
		pair_reference, "float_square_root_digit_many.pair",
	)
	workspace := workspace_reference[WORD_COUNT_MINIMUM]
	active_count := int(*count_reference[WORD_COUNT_MINIMUM])
	pair := pair_reference[WORD_COUNT_MINIMUM]
	remainder_carry := Word(BIT_CLEAR)
	candidate_carry := Word(BIT_CLEAR)
	for index := WORD_COUNT_MINIMUM; index < active_count; index++ {
		next := workspace.Remainder[index] >>
			(WORD_BIT_COUNT - FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT)
		workspace.Remainder[index] =
			workspace.Remainder[index]<<FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT |
				remainder_carry
		remainder_carry = next
		workspace.Candidate[index] =
			workspace.Root[index]<<FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT | candidate_carry
		candidate_carry = workspace.Root[index] >>
			(WORD_BIT_COUNT - FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT)
	}
	workspace.Remainder[WORD_COUNT_MINIMUM] |= Word(*pair)
	workspace.Candidate[WORD_COUNT_MINIMUM] |= Word(BIT_SET)
	active_last_index := active_count - WORD_COUNT_INCREMENT
	subtract := true
	for index := active_last_index; index >= WORD_COUNT_MINIMUM; index-- {
		if workspace.Remainder[index] != workspace.Candidate[index] {
			subtract = workspace.Remainder[index] > workspace.Candidate[index]
			break
		}
	}
	borrow := Word(bits.CARRY_MINIMUM)
	carry := Word(BIT_CLEAR)
	if subtract {
		for index := WORD_COUNT_MINIMUM; index <= active_last_index; index++ {
			left := workspace.Remainder[index]
			right := workspace.Candidate[index]
			difference := left - right - borrow
			borrow = ((^left & right) | (^(left ^ right) & difference)) >>
				WORD_BIT_INDEX_MAXIMUM
			workspace.Remainder[index] = difference
			next := workspace.Root[index] >> WORD_BIT_INDEX_MAXIMUM
			workspace.Root[index] = workspace.Root[index]<<WORD_COUNT_INCREMENT | carry
			carry = next
		}
		workspace.Root[WORD_COUNT_MINIMUM] |= Word(BIT_SET)
		return
	}
	for index := WORD_COUNT_MINIMUM; index <= active_last_index; index++ {
		next := workspace.Root[index] >> WORD_BIT_INDEX_MAXIMUM
		workspace.Root[index] = workspace.Root[index]<<WORD_COUNT_INCREMENT | carry
		carry = next
	}
}

func float_square_root_triple_word_one(
	workspace_reference *Float_Square_Root_Workspace_Reference,
	source_reference *Float_Reference,
) {
	Float_Square_Root_Workspace_Reference_Invariants(
		workspace_reference, "float_square_root_triple_word_one.workspace",
	)
	Float_Reference_Invariants(source_reference, "float_square_root_triple_word_one.source")
	workspace := workspace_reference[WORD_COUNT_MINIMUM]
	source := source_reference[WORD_COUNT_MINIMUM]
	root_word := Word(BIT_CLEAR)
	remainder_word := Word(BIT_CLEAR)
	candidate_word := Word(BIT_CLEAR)
	high_word := source.Mantissa.Words[WORD_COUNT_INCREMENT]
	remaining_count := WORD_BIT_COUNT
	for remaining_count > BIT_COUNT_MINIMUM {
		remaining_count -= FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT
		pair := high_word >> uint(remaining_count) & Word(FLOAT_SQUARE_ROOT_PAIR_MAXIMUM)
		remainder_word = remainder_word<<FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT | pair
		candidate_word = root_word<<FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT | Word(BIT_SET)
		if remainder_word >= candidate_word {
			remainder_word -= candidate_word
			root_word = root_word<<WORD_COUNT_INCREMENT | Word(BIT_SET)
		} else {
			root_word <<= WORD_COUNT_INCREMENT
		}
	}
	low_word := source.Mantissa.Words[WORD_COUNT_MINIMUM]
	remaining_count = WORD_BIT_COUNT
	for remaining_count > BASE_BINARY*FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT {
		remaining_count -= FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT
		pair := low_word >> uint(remaining_count) & Word(FLOAT_SQUARE_ROOT_PAIR_MAXIMUM)
		remainder_word = remainder_word<<FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT | pair
		candidate_word = root_word<<FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT | Word(BIT_SET)
		if remainder_word >= candidate_word {
			remainder_word -= candidate_word
			root_word = root_word<<WORD_COUNT_INCREMENT | Word(BIT_SET)
		} else {
			root_word <<= WORD_COUNT_INCREMENT
		}
	}
	workspace.Root[WORD_COUNT_MINIMUM] = root_word
	workspace.Remainder[WORD_COUNT_MINIMUM] = remainder_word
	workspace.Candidate[WORD_COUNT_MINIMUM] = candidate_word
}

func float_square_root_triple_word_two(
	workspace_reference *Float_Square_Root_Workspace_Reference,
	source_reference *Float_Reference,
) {
	Float_Square_Root_Workspace_Reference_Invariants(
		workspace_reference, "float_square_root_triple_word_two.workspace",
	)
	Float_Reference_Invariants(source_reference, "float_square_root_triple_word_two.source")
	workspace := workspace_reference[WORD_COUNT_MINIMUM]
	source := source_reference[WORD_COUNT_MINIMUM]
	root_low := workspace.Root[WORD_COUNT_MINIMUM]
	root_high := Word(BIT_CLEAR)
	remainder_low := workspace.Remainder[WORD_COUNT_MINIMUM]
	remainder_high := Word(BIT_CLEAR)
	candidate_low := Word(BIT_CLEAR)
	candidate_high := Word(BIT_CLEAR)
	low_word := source.Mantissa.Words[WORD_COUNT_MINIMUM]
	remaining_pair_count := BASE_BINARY
	for count := BIT_COUNT_MINIMUM; count < WORD_BIT_COUNT; count++ {
		pair := Word(BIT_CLEAR)
		if remaining_pair_count > WORD_COUNT_MINIMUM {
			remaining_pair_count--
			shift_count := remaining_pair_count * FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT
			pair = low_word >> uint(shift_count) & Word(FLOAT_SQUARE_ROOT_PAIR_MAXIMUM)
		}
		root_low_carry := root_low >> WORD_BIT_INDEX_MAXIMUM
		remainder_low_carry := remainder_low >>
			(WORD_BIT_COUNT - FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT)
		remainder_low = remainder_low<<FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT | pair
		remainder_high = remainder_high<<FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT |
			remainder_low_carry
		candidate_low = root_low<<FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT | Word(BIT_SET)
		candidate_high = root_high<<FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT |
			root_low>>(WORD_BIT_COUNT-FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT)
		subtract := remainder_high > candidate_high
		if remainder_high == candidate_high {
			subtract = remainder_low >= candidate_low
		}
		if subtract {
			borrow := Word(bits.CARRY_MINIMUM)
			if remainder_low < candidate_low {
				borrow = Word(bits.CARRY_MAXIMUM)
			}
			remainder_low -= candidate_low
			remainder_high = remainder_high - candidate_high - borrow
			root_low = root_low<<WORD_COUNT_INCREMENT | Word(BIT_SET)
		} else {
			root_low <<= WORD_COUNT_INCREMENT
		}
		root_high = root_high<<WORD_COUNT_INCREMENT | root_low_carry
	}
	workspace.Root[WORD_COUNT_MINIMUM] = root_low
	workspace.Root[WORD_COUNT_INCREMENT] = root_high
	workspace.Remainder[WORD_COUNT_MINIMUM] = remainder_low
	workspace.Remainder[WORD_COUNT_INCREMENT] = remainder_high
	workspace.Candidate[WORD_COUNT_MINIMUM] = candidate_low
	workspace.Candidate[WORD_COUNT_INCREMENT] = candidate_high
}

func float_square_root_triple_word_three(
	workspace_reference *Float_Square_Root_Workspace_Reference,
) {
	Float_Square_Root_Workspace_Reference_Invariants(
		workspace_reference, "float_square_root_triple_word_three.workspace",
	)
	workspace := workspace_reference[WORD_COUNT_MINIMUM]
	root_low := workspace.Root[WORD_COUNT_MINIMUM]
	root_middle := workspace.Root[WORD_COUNT_INCREMENT]
	root_high := Word(BIT_CLEAR)
	remainder_low := workspace.Remainder[WORD_COUNT_MINIMUM]
	remainder_middle := workspace.Remainder[WORD_COUNT_INCREMENT]
	remainder_high := Word(BIT_CLEAR)
	candidate_low := Word(BIT_CLEAR)
	candidate_middle := Word(BIT_CLEAR)
	candidate_high := Word(BIT_CLEAR)
	for count := BIT_COUNT_MINIMUM; count < WORD_BIT_COUNT; count++ {
		root_low_carry := root_low >> WORD_BIT_INDEX_MAXIMUM
		root_middle_carry := root_middle >> WORD_BIT_INDEX_MAXIMUM
		remainder_low_carry := remainder_low >>
			(WORD_BIT_COUNT - FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT)
		remainder_middle_carry := remainder_middle >>
			(WORD_BIT_COUNT - FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT)
		remainder_low <<= FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT
		remainder_middle = remainder_middle<<FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT |
			remainder_low_carry
		remainder_high = remainder_high<<FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT |
			remainder_middle_carry
		candidate_low = root_low<<FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT | Word(BIT_SET)
		candidate_middle = root_middle<<FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT |
			root_low>>(WORD_BIT_COUNT-FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT)
		candidate_high = root_high<<FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT |
			root_middle>>(WORD_BIT_COUNT-FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT)
		subtract := remainder_high > candidate_high
		if remainder_high == candidate_high {
			subtract = remainder_middle > candidate_middle
			if remainder_middle == candidate_middle {
				subtract = remainder_low >= candidate_low
			}
		}
		if subtract {
			difference_low := remainder_low - candidate_low
			borrow := ((^remainder_low & candidate_low) |
				(^(remainder_low ^ candidate_low) & difference_low)) >>
				WORD_BIT_INDEX_MAXIMUM
			difference_middle := remainder_middle - candidate_middle - borrow
			borrow = ((^remainder_middle & candidate_middle) |
				(^(remainder_middle ^ candidate_middle) & difference_middle)) >>
				WORD_BIT_INDEX_MAXIMUM
			remainder_low = difference_low
			remainder_middle = difference_middle
			remainder_high = remainder_high - candidate_high - borrow
			root_low = root_low<<WORD_COUNT_INCREMENT | Word(BIT_SET)
		} else {
			root_low <<= WORD_COUNT_INCREMENT
		}
		root_middle = root_middle<<WORD_COUNT_INCREMENT | root_low_carry
		root_high = root_high<<WORD_COUNT_INCREMENT | root_middle_carry
	}
	workspace.Root[WORD_COUNT_MINIMUM] = root_low
	workspace.Root[WORD_COUNT_INCREMENT] = root_middle
	workspace.Root[BASE_BINARY] = root_high
	workspace.Remainder[WORD_COUNT_MINIMUM] = remainder_low
	workspace.Remainder[WORD_COUNT_INCREMENT] = remainder_middle
	workspace.Remainder[BASE_BINARY] = remainder_high
	workspace.Candidate[WORD_COUNT_MINIMUM] = candidate_low
	workspace.Candidate[WORD_COUNT_INCREMENT] = candidate_middle
	workspace.Candidate[BASE_BINARY] = candidate_high
}

func float_square_root_triple_word(
	workspace_reference *Float_Square_Root_Workspace_Reference,
	source_reference *Float_Reference,
) {
	Float_Square_Root_Workspace_Reference_Invariants(
		workspace_reference, "float_square_root_triple_word.workspace",
	)
	Float_Reference_Invariants(source_reference, "float_square_root_triple_word.source")
	float_square_root_triple_word_one(workspace_reference, source_reference)
	float_square_root_triple_word_two(workspace_reference, source_reference)
	float_square_root_triple_word_three(workspace_reference)
	active_count := Float_Square_Root_Active_Word_Count(BASE_BINARY * BASE_BINARY)
	pair := Float_Square_Root_Pair(BIT_CLEAR)
	count_reference := Float_Square_Root_Count_Reference{&active_count}
	pair_reference := Float_Square_Root_Pair_Reference{&pair}
	final_count := FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT + FLOAT_SQUARE_ROOT_GUARD_BIT_COUNT
	for completed_count := BIT_COUNT_MINIMUM; completed_count < final_count; completed_count++ {
		float_square_root_digit_many(
			workspace_reference, &count_reference, &pair_reference,
		)
	}
}

func float_square_root_triple_word_match(
	source *Float_Nonnegative_Finite, workspace *Float_Square_Root_Workspace,
	precision Float_Active_Precision,
) (matched Boolean) {
	defer func() {
		Boolean_Invariants(matched, "float_square_root_triple_word_match.matched")
	}()
	Float_Nonnegative_Finite_Invariants(
		source, "float_square_root_triple_word_match.source",
	)
	Float_Square_Root_Workspace_Invariants(
		workspace, "float_square_root_triple_word_match.workspace",
	)
	Float_Active_Precision_Invariants(
		precision, "float_square_root_triple_word_match.precision",
	)
	if precision != FLOAT_SQUARE_ROOT_TRIPLE_WORD_PRECISION {
		return false
	}
	if source.Mantissa.Count != BASE_BINARY {
		return false
	}
	if int(source.Exponent)%BASE_BINARY != BIT_COUNT_MINIMUM {
		return false
	}
	high_word := source.Mantissa.Words[WORD_COUNT_INCREMENT]
	if high_word>>WORD_BIT_INDEX_MAXIMUM != Word(BIT_SET) {
		return false
	}
	workspace_reference := Float_Square_Root_Workspace_Reference{workspace}
	source_reference := Float_Reference{(*Float)(source)}
	float_square_root_triple_word(&workspace_reference, &source_reference)
	return true
}
