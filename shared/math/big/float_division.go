// Package big keeps large arithmetic bounded without hidden allocation.
package big

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
)

// INT_DIVISION_GENERAL_DIVISOR_WORD_COUNT_MAXIMUM leaves one higher dividend word.
const INT_DIVISION_GENERAL_DIVISOR_WORD_COUNT_MAXIMUM = WORD_COUNT_MAXIMUM -
	WORD_COUNT_INCREMENT

// Result construction reuses preparation fields after quotient extraction consumes them.
const FLOAT_DIVISION_RESULT_PRECISION_INDEX = FLOAT_DIVISION_SHIFT_INDEX

// FLOAT_DIVISION_RESULT_PREVIOUS_COUNT_INDEX reuses the consumed quotient offset.
const FLOAT_DIVISION_RESULT_PREVIOUS_COUNT_INDEX = FLOAT_DIVISION_OFFSET_INDEX

// Int_Division_Empty_Workspace is the reset state required by word division.
type Int_Division_Empty_Workspace Int_Division_Workspace

// Int_Division_Empty_Workspace_Invariants excludes stale scratch from word algorithms.
func Int_Division_Empty_Workspace_Invariants(
	value Int_Division_Empty_Workspace, _ aver.Namespace,
) {
	aver.Always(len(value.Quotient) == WORD_COUNT_MAXIMUM,
		"Empty quotient storage is complete.")
	aver.Always(len(value.Remainder) == WORD_COUNT_MAXIMUM,
		"Empty remainder storage is complete.")
	aver.Always(value.Quotient_Count == Quotient_Count(WORD_COUNT_MINIMUM),
		"Word division quotient starts empty.")
	aver.Always(value.Remainder_Count == Remainder_Count(WORD_COUNT_MINIMUM),
		"Word division remainder starts empty.")
}

// Int_Division_Multiword_Magnitude excludes scalar paths handled before word division.
type Int_Division_Multiword_Magnitude Int

// Int_Division_Multiword_Magnitude_Invariants keeps normalized inline multiword storage.
func Int_Division_Multiword_Magnitude_Invariants(
	value Int_Division_Multiword_Magnitude, namespace aver.Namespace,
) {
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM, "Multiword storage remains complete.")
	aver.Tree(Polarity(value.Negative), namespace).
		Enum_Uint8(
			uint8(value.Negative),
			uint8(POLARITY_NONNEGATIVE), uint8(POLARITY_NEGATIVE),
		).
		Ensure()
	aver.Tree(Word_Count(value.Count), namespace).
		Range_Int(
			int(value.Count),
			WORD_COUNT_INCREMENT+WORD_COUNT_INCREMENT,
			WORD_COUNT_MAXIMUM,
		).
		Ensure()
	high_index := (value.Count - Word_Count(WORD_COUNT_INCREMENT)) &
		Word_Count(WORD_INDEX_MAXIMUM)
	aver.Always(
		value.Words[high_index] != 0,
		"A multiword division magnitude omits high zero words.",
	)
}

// Int_Division_Nonzero_Word is the exact divisor domain of word division.
type Int_Division_Nonzero_Word Word

// Int_Division_Nonzero_Word_Invariants rejects the zero divisor handled by the caller.
func Int_Division_Nonzero_Word_Invariants(
	value Int_Division_Nonzero_Word, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(
			uint64(value), uint64(bits.CARRY_MAXIMUM), uint64(bits.WORD_64_MAXIMUM),
		).
		Ensure()
}

// Int_Division_Quotient_Word excludes zero after an equal-width dividend wins comparison.
type Int_Division_Quotient_Word Word

// Int_Division_Quotient_Word_Invariants covers one complete nonzero quotient word.
func Int_Division_Quotient_Word_Invariants(
	value Int_Division_Quotient_Word, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(
			uint64(value), uint64(bits.CARRY_MAXIMUM), uint64(bits.WORD_64_MAXIMUM),
		).
		Ensure()
}

// Int_Division_General_Divisor leaves at least one higher dividend word.
type Int_Division_General_Divisor Int

// Int_Division_General_Divisor_Invariants states the restoring-path divisor domain.
func Int_Division_General_Divisor_Invariants(
	value Int_Division_General_Divisor, namespace aver.Namespace,
) {
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM, "General divisor storage complete.")
	aver.Tree(Polarity(value.Negative), namespace).
		Enum_Uint8(
			uint8(value.Negative),
			uint8(POLARITY_NONNEGATIVE), uint8(POLARITY_NEGATIVE),
		).
		Ensure()
	aver.Tree(Word_Count(value.Count), namespace).
		Range_Int(
			int(value.Count),
			WORD_COUNT_INCREMENT+WORD_COUNT_INCREMENT,
			INT_DIVISION_GENERAL_DIVISOR_WORD_COUNT_MAXIMUM,
		).
		Ensure()
	high_index := (value.Count - Word_Count(WORD_COUNT_INCREMENT)) &
		Word_Count(WORD_INDEX_MAXIMUM)
	aver.Always(
		value.Words[high_index] != 0,
		"A general division divisor omits high zero words.",
	)
}

// Int_Division_Small_Dividend is the bounded Knuth path above equal-width division.
type Int_Division_Small_Dividend Int

// Int_Division_Small_Dividend_Invariants states every reachable small dividend width.
func Int_Division_Small_Dividend_Invariants(
	value Int_Division_Small_Dividend, namespace aver.Namespace,
) {
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM, "Small dividend storage complete.")
	aver.Tree(Polarity(value.Negative), namespace).
		Enum_Uint8(
			uint8(value.Negative),
			uint8(POLARITY_NONNEGATIVE), uint8(POLARITY_NEGATIVE),
		).
		Ensure()
	aver.Tree(Word_Count(value.Count), namespace).
		Enum_Int(
			int(value.Count),
			WORD_COUNT_INCREMENT+WORD_COUNT_INCREMENT+WORD_COUNT_INCREMENT,
			BASE_BINARY*BASE_BINARY,
		).
		Ensure()
	high_index := value.Count - Word_Count(WORD_COUNT_INCREMENT)
	aver.Always(
		value.Words[high_index] != 0,
		"A small division dividend omits high zero words.",
	)
}

// Int_Division_Small_Divisor is the multiword divisor below a small dividend.
type Int_Division_Small_Divisor Int

// Int_Division_Small_Divisor_Invariants states every reachable small divisor width.
func Int_Division_Small_Divisor_Invariants(
	value Int_Division_Small_Divisor, namespace aver.Namespace,
) {
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM, "Small divisor storage complete.")
	aver.Tree(Polarity(value.Negative), namespace).
		Enum_Uint8(
			uint8(value.Negative),
			uint8(POLARITY_NONNEGATIVE), uint8(POLARITY_NEGATIVE),
		).
		Ensure()
	aver.Tree(Word_Count(value.Count), namespace).
		Enum_Int(
			int(value.Count), WORD_COUNT_INCREMENT+WORD_COUNT_INCREMENT,
			BASE_BINARY*BASE_BINARY-WORD_COUNT_INCREMENT,
		).
		Ensure()
	high_index := value.Count - Word_Count(WORD_COUNT_INCREMENT)
	aver.Always(
		value.Words[high_index] != 0,
		"A small division divisor omits high zero words.",
	)
}

// Int_Division_Restoring_Workspace is state before one quotient bit enters.
type Int_Division_Restoring_Workspace Int_Division_Workspace

// Int_Division_Restoring_Workspace_Invariants excludes an impossible full-width quotient.
func Int_Division_Restoring_Workspace_Invariants(
	value Int_Division_Restoring_Workspace, namespace aver.Namespace,
) {
	aver.Always(len(value.Quotient) == WORD_COUNT_MAXIMUM,
		"Restoring quotient storage is complete.")
	aver.Always(len(value.Remainder) == WORD_COUNT_MAXIMUM,
		"Restoring remainder storage is complete.")
	aver.Tree(Quotient_Count(value.Quotient_Count), namespace).
		Range_Int(
			int(value.Quotient_Count), WORD_COUNT_MINIMUM,
			INT_DIVISION_GENERAL_DIVISOR_WORD_COUNT_MAXIMUM,
		).
		Ensure()
	aver.Tree(Remainder_Count(value.Remainder_Count), namespace).
		Range_Int(
			int(value.Remainder_Count), WORD_COUNT_MINIMUM,
			INT_DIVISION_GENERAL_DIVISOR_WORD_COUNT_MAXIMUM,
		).
		Ensure()
}

func int_divide_by_word(
	workspace Int_Division_Empty_Workspace_Handle,
	dividend Int_Division_Multiword_Magnitude_Handle,
	divisor Int_Division_Nonzero_Word,
) {
	Int_Division_Empty_Workspace_Handle_Invariants(workspace, "int_divide_by_word.workspace")
	Int_Division_Multiword_Magnitude_Handle_Invariants(
		dividend, "int_divide_by_word.dividend",
	)
	Int_Division_Nonzero_Word_Invariants(divisor, "int_divide_by_word.divisor")
	remainder := Word(0)
	high_index := int(dividend.Count) - WORD_COUNT_INCREMENT
	for index := high_index; index >= WORD_COUNT_MINIMUM; index-- {
		quotient, rest := bits.Divide_64(
			bits.Dividend_High_64(remainder),
			bits.Dividend_Low_64(dividend.Words[index]),
			bits.Divisor_64(divisor),
		)
		workspace.Quotient[index] = Word(quotient)
		remainder = Word(rest)
	}
	workspace.Quotient_Count = Quotient_Count(dividend.Count)
	for workspace.Quotient_Count > Quotient_Count(WORD_COUNT_MINIMUM) {
		high := int(workspace.Quotient_Count) - WORD_COUNT_INCREMENT
		if workspace.Quotient[high] != 0 {
			break
		}
		workspace.Quotient_Count--
	}
	if remainder != 0 {
		workspace.Remainder[WORD_COUNT_MINIMUM] = remainder
		workspace.Remainder_Count = Remainder_Count(WORD_COUNT_INCREMENT)
	}
}

func int_divide_equal_word_count(
	workspace Int_Division_Empty_Workspace_Handle,
	dividend Int_Division_Multiword_Magnitude_Handle,
	divisor Int_Division_Multiword_Magnitude_Handle,
) {
	Int_Division_Empty_Workspace_Handle_Invariants(
		workspace, "int_divide_equal_word_count.workspace",
	)
	Int_Division_Multiword_Magnitude_Handle_Invariants(
		dividend, "int_divide_equal_word_count.dividend",
	)
	Int_Division_Multiword_Magnitude_Handle_Invariants(
		divisor, "int_divide_equal_word_count.divisor",
	)
	aver.Always(
		dividend.Count == divisor.Count,
		"Equal-width division receives equal normalized word counts.",
	)
	dividend_integer := (*Int)((*Int_Division_Multiword_Magnitude)(dividend))
	divisor_integer := (*Int)((*Int_Division_Multiword_Magnitude)(divisor))
	order := int_compare_absolute(dividend_integer, divisor_integer)
	if order != ORDER_AFTER {
		if order == ORDER_SAME {
			workspace.Quotient[WORD_COUNT_MINIMUM] = Word(bits.CARRY_MAXIMUM)
			workspace.Quotient_Count = Quotient_Count(WORD_COUNT_INCREMENT)
			return
		}
		for index := Word_Count(WORD_COUNT_MINIMUM); index < dividend.Count; index++ {
			workspace.Remainder[index] = dividend.Words[index]
		}
		workspace.Remainder_Count = Remainder_Count(dividend.Count)
		return
	}
	quotient := int_divide_equal_word_count_quotient(dividend, divisor)
	int_divide_equal_word_count_commit(workspace, dividend, divisor, quotient)
}

func int_divide_equal_word_count_quotient(
	dividend Int_Division_Multiword_Magnitude_Handle,
	divisor Int_Division_Multiword_Magnitude_Handle,
) (quotient Int_Division_Quotient_Word) {
	defer func() {
		Int_Division_Quotient_Word_Invariants(
			quotient, "int_divide_equal_word_count_quotient.quotient",
		)
	}()
	Int_Division_Multiword_Magnitude_Handle_Invariants(
		dividend, "int_divide_equal_word_count_quotient.dividend",
	)
	Int_Division_Multiword_Magnitude_Handle_Invariants(
		divisor, "int_divide_equal_word_count_quotient.divisor",
	)
	dividend_integer := (*Int)((*Int_Division_Multiword_Magnitude)(dividend))
	divisor_integer := (*Int)((*Int_Division_Multiword_Magnitude)(divisor))
	order := int_compare_absolute(dividend_integer, divisor_integer)
	aver.Always(order == ORDER_AFTER, "A quotient estimate receives a larger dividend.")
	high_index := int(divisor.Count) - WORD_COUNT_INCREMENT
	next_index := high_index - WORD_COUNT_INCREMENT
	shift := uint(bits.Leading_Zeros_64(bits.Word_64(divisor.Words[high_index])))
	divisor_high := divisor.Words[high_index]
	dividend_high := Word(0)
	dividend_next := dividend.Words[high_index]
	if shift != 0 {
		divisor_high = divisor_high<<shift |
			divisor.Words[next_index]>>(WORD_BIT_COUNT-shift)
		dividend_high = dividend.Words[high_index] >> (WORD_BIT_COUNT - shift)
		dividend_next = dividend.Words[high_index]<<shift |
			dividend.Words[next_index]>>(WORD_BIT_COUNT-shift)
	}
	estimate, rest := bits.Divide_64(
		bits.Dividend_High_64(dividend_high), bits.Dividend_Low_64(dividend_next),
		bits.Divisor_64(divisor_high),
	)
	divisor_next := divisor.Words[next_index] << shift
	trial_next := dividend.Words[next_index] << shift
	if shift != 0 {
		if next_index > WORD_COUNT_MINIMUM {
			divisor_next |= divisor.Words[next_index-WORD_COUNT_INCREMENT] >>
				(WORD_BIT_COUNT - shift)
			trial_next |= dividend.Words[next_index-WORD_COUNT_INCREMENT] >>
				(WORD_BIT_COUNT - shift)
		}
	}
	for correction := WORD_COUNT_MINIMUM; correction < BASE_BINARY; correction++ {
		product_high, product_low := bits.Multiply_64(
			bits.Word_64(estimate), bits.Multiplier_64(divisor_next),
		)
		if uint64(product_high) < uint64(rest) {
			break
		}
		if uint64(product_high) == uint64(rest) {
			if uint64(product_low) <= uint64(trial_next) {
				break
			}
		}
		estimate--
		previous := uint64(rest)
		rest = bits.Division_Remainder_64(previous + uint64(divisor_high))
		if uint64(rest) < previous {
			break
		}
	}
	return Int_Division_Quotient_Word(estimate)
}

func int_divide_equal_word_count_commit(
	workspace Int_Division_Empty_Workspace_Handle,
	dividend Int_Division_Multiword_Magnitude_Handle,
	divisor Int_Division_Multiword_Magnitude_Handle,
	quotient Int_Division_Quotient_Word,
) {
	Int_Division_Empty_Workspace_Handle_Invariants(
		workspace, "int_divide_equal_word_count_commit.workspace",
	)
	Int_Division_Multiword_Magnitude_Handle_Invariants(
		dividend, "int_divide_equal_word_count_commit.dividend",
	)
	Int_Division_Multiword_Magnitude_Handle_Invariants(
		divisor, "int_divide_equal_word_count_commit.divisor",
	)
	Int_Division_Quotient_Word_Invariants(
		quotient, "int_divide_equal_word_count_commit.quotient",
	)
	carry, borrow := uint64(0), uint64(0)
	for index := WORD_COUNT_MINIMUM; index < int(divisor.Count); index++ {
		left, right := uint64(quotient), uint64(divisor.Words[index])
		left_low := left & TEXT_DIVISION_LIMB_MASK
		right_low := right & TEXT_DIVISION_LIMB_MASK
		left_high := left >> TEXT_DIVISION_LIMB_BIT_COUNT
		right_high := right >> TEXT_DIVISION_LIMB_BIT_COUNT
		partial := left_low * right_low
		middle_first := left_high*right_low + partial>>TEXT_DIVISION_LIMB_BIT_COUNT
		middle_second := left_low*right_high + middle_first&TEXT_DIVISION_LIMB_MASK
		product_high := left_high*right_high +
			middle_first>>TEXT_DIVISION_LIMB_BIT_COUNT +
			middle_second>>TEXT_DIVISION_LIMB_BIT_COUNT
		product_low := left*right + carry
		if product_low < carry {
			product_high++
		}
		minuend := uint64(dividend.Words[index])
		difference := minuend - product_low - borrow
		workspace.Remainder[index] = Word(difference)
		borrow = ((^minuend & product_low) |
			(^(minuend ^ product_low) & difference)) >> WORD_BIT_INDEX_MAXIMUM
		carry = product_high
	}
	aver.Always(
		carry+borrow == 0,
		"Normalized quotient estimate cannot exceed equal-width dividend.",
	)
	workspace.Remainder_Count = Remainder_Count(dividend.Count)
	for workspace.Remainder_Count > Remainder_Count(WORD_COUNT_MINIMUM) {
		high := int(workspace.Remainder_Count) - WORD_COUNT_INCREMENT
		if workspace.Remainder[high] != 0 {
			break
		}
		workspace.Remainder_Count--
	}
	workspace.Quotient[WORD_COUNT_MINIMUM] = Word(quotient)
	workspace.Quotient_Count = Quotient_Count(WORD_COUNT_INCREMENT)
}

// Int_Division_Small_Trial_Dividend groups the three normalized estimate words.
type Int_Division_Small_Trial_Dividend struct {
	// High retains carry from normalized numerator prefix.
	High Int_Division_Trial_High
	// Next completes double-word quotient numerator.
	Next Int_Division_Trial_Next
	// Third distinguishes estimates sharing leading numerator words.
	Third Int_Division_Trial_Third
}

// Int_Division_Small_Trial_Dividend_Invariants fixes the complete estimate numerator.
func Int_Division_Small_Trial_Dividend_Invariants(
	value Int_Division_Small_Trial_Dividend, namespace aver.Namespace,
) {
	Int_Division_Trial_High_Invariants(value.High, namespace)
	Int_Division_Trial_Next_Invariants(value.Next, namespace)
	Int_Division_Trial_Third_Invariants(value.Third, namespace)
}

// INT_DIVISION_NORMALIZED_WORD_MINIMUM keeps leading divisor bit set for estimation.
const INT_DIVISION_NORMALIZED_WORD_MINIMUM = bits.CARRY_MAXIMUM << WORD_BIT_INDEX_MAXIMUM

// Int_Division_Small_Trial_Divisor groups the two normalized correction words.
type Int_Division_Small_Trial_Divisor struct {
	// High establishes normalized quotient scale.
	High Int_Division_Normalized_Word
	// Next supplies correction precision beyond leading word.
	Next Word
}

// Int_Division_Small_Trial_Divisor_Invariants fixes and normalizes the divisor prefix.
func Int_Division_Small_Trial_Divisor_Invariants(
	value Int_Division_Small_Trial_Divisor, namespace aver.Namespace,
) {
	Int_Division_Normalized_Word_Invariants(value.High, namespace)
	Word_Invariants(value.Next, namespace)
}

// Int_Division_Small_Trial holds normalized leading words for one quotient estimate.
type Int_Division_Small_Trial struct {
	// Dividend keeps normalization outside caller storage.
	Dividend Int_Division_Small_Trial_Dividend
	// Divisor keeps correction below one full normalized copy.
	Divisor Int_Division_Small_Trial_Divisor
}

// Int_Division_Small_Trial_Invariants binds every trial field to one machine word.
func Int_Division_Small_Trial_Invariants(
	value Int_Division_Small_Trial, namespace aver.Namespace,
) {
	Int_Division_Small_Trial_Dividend_Invariants(value.Dividend, namespace)
	Int_Division_Small_Trial_Divisor_Invariants(value.Divisor, namespace)
}

// Int_Division_Small_Offset is a quotient word offset inside four dividend words.
type Int_Division_Small_Offset int

// Int_Division_Small_Offset_Invariants states every reachable small quotient offset.
func Int_Division_Small_Offset_Invariants(
	value Int_Division_Small_Offset, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Int(
			int(value), WORD_COUNT_MINIMUM, WORD_COUNT_INCREMENT, BASE_BINARY,
		).
		Ensure()
}

func int_divide_small_trial(
	workspace Int_Division_Empty_Workspace_Handle,
	divisor Int_Division_Small_Divisor_Handle,
	offset Int_Division_Small_Offset,
) (trial Int_Division_Small_Trial) {
	defer func() {
		Int_Division_Small_Trial_Invariants(trial, "int_divide_small_trial.trial")
	}()
	Int_Division_Empty_Workspace_Handle_Invariants(
		workspace, "int_divide_small_trial.workspace",
	)
	Int_Division_Small_Divisor_Handle_Invariants(divisor, "int_divide_small_trial.divisor")
	Int_Division_Small_Offset_Invariants(offset, "int_divide_small_trial.offset")
	divisor_count := int(divisor.Count)
	high_index := int(offset) + divisor_count
	next_index, third_index := high_index-WORD_COUNT_INCREMENT,
		high_index-WORD_COUNT_INCREMENT-WORD_COUNT_INCREMENT
	trial.Dividend.High = Int_Division_Trial_High(workspace.Remainder[high_index])
	trial.Dividend.Next = Int_Division_Trial_Next(workspace.Remainder[next_index])
	trial.Dividend.Third = Int_Division_Trial_Third(workspace.Remainder[third_index])
	trial.Divisor.High =
		Int_Division_Normalized_Word(divisor.Words[divisor_count-WORD_COUNT_INCREMENT])
	trial.Divisor.Next =
		divisor.Words[divisor_count-BASE_BINARY]
	shift := uint(bits.Leading_Zeros_64(bits.Word_64(
		trial.Divisor.High,
	)))
	if shift == 0 {
		return trial
	}
	trial.Dividend.High = trial.Dividend.High<<shift |
		Int_Division_Trial_High(trial.Dividend.Next>>(WORD_BIT_COUNT-shift))
	trial.Dividend.Next = trial.Dividend.Next<<shift |
		Int_Division_Trial_Next(trial.Dividend.Third>>(WORD_BIT_COUNT-shift))
	trial.Dividend.Third <<= shift
	lower_index := third_index - WORD_COUNT_INCREMENT
	if lower_index >= WORD_COUNT_MINIMUM {
		trial.Dividend.Third |= Int_Division_Trial_Third(
			workspace.Remainder[lower_index] >> (WORD_BIT_COUNT - shift),
		)
	}
	trial.Divisor.High = trial.Divisor.High<<shift |
		Int_Division_Normalized_Word(trial.Divisor.Next>>(WORD_BIT_COUNT-shift))
	trial.Divisor.Next <<= shift
	lower_index = divisor_count - BASE_BINARY - WORD_COUNT_INCREMENT
	if lower_index >= WORD_COUNT_MINIMUM {
		trial.Divisor.Next |=
			divisor.Words[lower_index] >> (WORD_BIT_COUNT - shift)
	}
	return trial
}

func int_divide_small_estimate(trial Int_Division_Small_Trial) (estimate Word) {
	defer func() { Word_Invariants(estimate, "int_divide_small_estimate.estimate") }()
	Int_Division_Small_Trial_Invariants(trial, "int_divide_small_estimate.trial")
	trial_word := uint64(bits.WORD_64_MAXIMUM)
	dividend_high := Word(trial.Dividend.High)
	dividend_next := Word(trial.Dividend.Next)
	divisor_high := Word(trial.Divisor.High)
	remainder := uint64(dividend_next + divisor_high)
	remainder_overflow := remainder < uint64(dividend_next)
	if dividend_high < divisor_high {
		quotient, rest := int_divide_small_word_quotient(
			Int_Division_Word_Remainder(dividend_high), dividend_next,
			trial.Divisor.High)
		trial_word = uint64(quotient)
		remainder = uint64(rest)
		remainder_overflow = false
	}
	correction := WORD_COUNT_MINIMUM
	for correction < BASE_BINARY {
		if remainder_overflow {
			break
		}
		left := trial_word
		right := uint64(trial.Divisor.Next)
		left_low, right_low := left&TEXT_DIVISION_LIMB_MASK,
			right&TEXT_DIVISION_LIMB_MASK
		left_high, right_high := left>>TEXT_DIVISION_LIMB_BIT_COUNT,
			right>>TEXT_DIVISION_LIMB_BIT_COUNT
		partial, product_low := left_low*right_low, left*right
		middle_first := left_high*right_low + partial>>TEXT_DIVISION_LIMB_BIT_COUNT
		middle_second := left_low*right_high + middle_first&TEXT_DIVISION_LIMB_MASK
		product_high := left_high*right_high +
			middle_first>>TEXT_DIVISION_LIMB_BIT_COUNT +
			middle_second>>TEXT_DIVISION_LIMB_BIT_COUNT
		if product_high < remainder {
			break
		}
		if product_high == remainder {
			if product_low <= uint64(trial.Dividend.Third) {
				break
			}
		}
		trial_word--
		previous := remainder
		remainder += uint64(divisor_high)
		remainder_overflow = remainder < previous
		correction++
	}
	return Word(trial_word)
}

func int_divide_small_word_quotient(
	high_word Int_Division_Word_Remainder, low_word Word,
	divisor_word Int_Division_Normalized_Word,
) (quotient Word, rest Int_Division_Word_Remainder) {
	defer func() {
		Word_Invariants(quotient, "int_divide_small_word_quotient.quotient")
		Int_Division_Word_Remainder_Invariants(rest, "int_divide_small_word_quotient.rest")
	}()
	Int_Division_Word_Remainder_Invariants(
		high_word, "int_divide_small_word_quotient.high_word")
	Word_Invariants(low_word, "int_divide_small_word_quotient.low_word")
	Int_Division_Normalized_Word_Invariants(
		divisor_word, "int_divide_small_word_quotient.divisor_word")
	high := uint64(high_word)
	low := uint64(low_word)
	divisor := uint64(divisor_word)
	if high == uint64(WORD_COUNT_MINIMUM) {
		return Word(low / divisor), Int_Division_Word_Remainder(low % divisor)
	}
	limb_base := uint64(bits.CARRY_MAXIMUM) << TEXT_DIVISION_LIMB_BIT_COUNT
	limb_mask := uint64(TEXT_DIVISION_LIMB_MASK)
	divisor_high, divisor_low := divisor>>TEXT_DIVISION_LIMB_BIT_COUNT,
		divisor&limb_mask
	middle, bottom := low>>TEXT_DIVISION_LIMB_BIT_COUNT, low&limb_mask
	first := high / divisor_high
	partial := high - first*divisor_high
	for correction := WORD_COUNT_MINIMUM; correction < BASE_BINARY; correction++ {
		too_large := first >= limb_base
		if first*divisor_low > limb_base*partial+middle {
			too_large = true
		}
		if !too_large {
			break
		}
		first--
		partial += divisor_high
		if partial >= limb_base {
			break
		}
	}
	joined := high*limb_base + middle - first*divisor
	second := joined / divisor_high
	partial = joined - second*divisor_high
	for correction := WORD_COUNT_MINIMUM; correction < BASE_BINARY; correction++ {
		too_large := second >= limb_base
		if second*divisor_low > limb_base*partial+bottom {
			too_large = true
		}
		if !too_large {
			break
		}
		second--
		partial += divisor_high
		if partial >= limb_base {
			break
		}
	}
	remainder := joined*limb_base + bottom - second*divisor
	return Word(first*limb_base + second), Int_Division_Word_Remainder(remainder)
}

func int_divide_small_subtract(
	workspace Int_Division_Empty_Workspace_Handle,
	divisor Int_Division_Small_Divisor_Handle,
	offset Int_Division_Small_Offset,
	estimate Word,
) (adjusted Word) {
	defer func() { Word_Invariants(adjusted, "int_divide_small_subtract.adjusted") }()
	Int_Division_Empty_Workspace_Handle_Invariants(
		workspace, "int_divide_small_subtract.workspace",
	)
	Int_Division_Small_Divisor_Handle_Invariants(
		divisor, "int_divide_small_subtract.divisor",
	)
	Int_Division_Small_Offset_Invariants(offset, "int_divide_small_subtract.offset")
	Word_Invariants(estimate, "int_divide_small_subtract.estimate")
	carry, borrow := uint64(0), uint64(0)
	divisor_index := WORD_COUNT_MINIMUM
	for divisor_index < int(divisor.Count) {
		left, right := uint64(estimate), uint64(divisor.Words[divisor_index])
		left_low, right_low := left&TEXT_DIVISION_LIMB_MASK,
			right&TEXT_DIVISION_LIMB_MASK
		left_high, right_high := left>>TEXT_DIVISION_LIMB_BIT_COUNT,
			right>>TEXT_DIVISION_LIMB_BIT_COUNT
		partial, product_low := left_low*right_low, left*right
		middle_first := left_high*right_low + partial>>TEXT_DIVISION_LIMB_BIT_COUNT
		middle_second := left_low*right_high + middle_first&TEXT_DIVISION_LIMB_MASK
		product_high := left_high*right_high +
			middle_first>>TEXT_DIVISION_LIMB_BIT_COUNT +
			middle_second>>TEXT_DIVISION_LIMB_BIT_COUNT
		product_low += carry
		if product_low < carry {
			product_high++
		}
		position := int(offset)
		position += divisor_index
		minuend := uint64(workspace.Remainder[position])
		difference := minuend - product_low - borrow
		workspace.Remainder[position] = Word(difference)
		borrow = ((^minuend & product_low) |
			(^(minuend ^ product_low) & difference)) >> WORD_BIT_INDEX_MAXIMUM
		carry = product_high
		divisor_index++
	}
	high_position := int(offset)
	high_position += int(divisor.Count)
	minuend := uint64(workspace.Remainder[high_position])
	difference := minuend - carry - borrow
	workspace.Remainder[high_position] = Word(difference)
	borrow = ((^minuend & carry) |
		(^(minuend ^ carry) & difference)) >> WORD_BIT_INDEX_MAXIMUM
	if borrow == 0 {
		return estimate
	}
	adjusted = estimate - Word(bits.CARRY_MAXIMUM)
	carry = 0
	divisor_index = WORD_COUNT_MINIMUM
	for divisor_index < int(divisor.Count) {
		position := int(offset)
		position += divisor_index
		left := uint64(workspace.Remainder[position])
		right := uint64(divisor.Words[divisor_index])
		sum := left + right + carry
		workspace.Remainder[position] = Word(sum)
		carry = ((left & right) | ((left | right) &^ sum)) >>
			WORD_BIT_INDEX_MAXIMUM
		divisor_index++
	}
	workspace.Remainder[high_position] += Word(carry)
	return adjusted
}

func int_divide_small(
	workspace Int_Division_Empty_Workspace_Handle,
	dividend Int_Division_Small_Dividend_Handle,
	divisor Int_Division_Small_Divisor_Handle,
) {
	Int_Division_Empty_Workspace_Handle_Invariants(workspace, "int_divide_small.workspace")
	Int_Division_Small_Dividend_Handle_Invariants(dividend, "int_divide_small.dividend")
	Int_Division_Small_Divisor_Handle_Invariants(divisor, "int_divide_small.divisor")
	aver.Always(
		dividend.Count > divisor.Count,
		"Small division receives at least one quotient position.",
	)
	dividend_count := int(dividend.Count)
	divisor_count := int(divisor.Count)
	for index := WORD_COUNT_MINIMUM; index < dividend_count; index++ {
		workspace.Remainder[index] = dividend.Words[index]
	}
	workspace.Remainder[dividend_count] = 0
	quotient_count := dividend_count - divisor_count + WORD_COUNT_INCREMENT
	quotient_offset := quotient_count - WORD_COUNT_INCREMENT
	for quotient_offset >= 0 {
		word_offset := Int_Division_Small_Offset(quotient_offset)
		trial := int_divide_small_trial(workspace, divisor, word_offset)
		estimate := int_divide_small_estimate(trial)
		workspace.Quotient[quotient_offset] = int_divide_small_subtract(
			workspace, divisor, word_offset, estimate,
		)
		quotient_offset--
	}
	workspace.Quotient_Count = Quotient_Count(quotient_count)
	for workspace.Quotient_Count > Quotient_Count(WORD_COUNT_MINIMUM) {
		high := int(workspace.Quotient_Count) - WORD_COUNT_INCREMENT
		if workspace.Quotient[high] != 0 {
			break
		}
		workspace.Quotient_Count--
	}
	workspace.Remainder_Count = Remainder_Count(divisor_count)
	for workspace.Remainder_Count > Remainder_Count(WORD_COUNT_MINIMUM) {
		high := int(workspace.Remainder_Count) - WORD_COUNT_INCREMENT
		if workspace.Remainder[high] != 0 {
			break
		}
		workspace.Remainder_Count--
	}
}

func int_binomial_word(destination Int_Handle, n Int_64, k Int_64) (matched Boolean) {
	defer func() { Boolean_Invariants(matched, "int_binomial_word.matched") }()
	Int_Handle_Invariants(destination, "int_binomial_word.destination")
	Int_64_Invariants(n, "int_binomial_word.n")
	Int_64_Invariants(k, "int_binomial_word.k")
	if n < 0 {
		return false
	}
	if k < 0 {
		int_set_word(destination, Word(bits.CARRY_MAXIMUM), POLARITY_NONNEGATIVE)
		return true
	}
	if k > n {
		int_zero(destination, destination.Count)
		return true
	}
	if k > n-k {
		k = n - k
	}
	result := uint64(bits.CARRY_MAXIMUM)
	for index := int64(WORD_COUNT_INCREMENT); index <= int64(k); index++ {
		factor := uint64(int64(n) - int64(k) + index)
		if result != 0 {
			if factor > uint64(bits.WORD_64_MAXIMUM)/result {
				return false
			}
		}
		result = result * factor / uint64(index)
	}
	int_set_word(destination, Word(result), POLARITY_NONNEGATIVE)
	return true
}

func int_multiply_range_word(
	destination Int_Handle, minimum Int_64, maximum Int_64,
) (matched Boolean) {
	defer func() { Boolean_Invariants(matched, "int_multiply_range_word.matched") }()
	Int_Handle_Invariants(destination, "int_multiply_range_word.destination")
	Int_64_Invariants(minimum, "int_multiply_range_word.minimum")
	Int_64_Invariants(maximum, "int_multiply_range_word.maximum")
	if minimum < WORD_COUNT_INCREMENT {
		return false
	}
	if maximum < minimum {
		int_set_word(destination, Word(bits.CARRY_MAXIMUM), POLARITY_NONNEGATIVE)
		return true
	}
	result := uint64(bits.CARRY_MAXIMUM)
	for value := int64(minimum); value <= int64(maximum); value++ {
		if result != 0 {
			if uint64(value) > uint64(bits.WORD_64_MAXIMUM)/result {
				return false
			}
		}
		result *= uint64(value)
	}
	int_set_word(destination, Word(result), POLARITY_NONNEGATIVE)
	return true
}

func int_exponent_word(
	destination Int_Handle, base Word, base_negative Polarity, exponent Int_Handle,
) (matched Boolean) {
	defer func() { Boolean_Invariants(matched, "int_exponent_word.matched") }()
	Int_Handle_Invariants(destination, "int_exponent_word.destination")
	Word_Invariants(base, "int_exponent_word.base")
	Polarity_Invariants(base_negative, "int_exponent_word.base_negative")
	Int_Handle_Invariants(exponent, "int_exponent_word.exponent")
	if exponent.Count > Word_Count(WORD_COUNT_INCREMENT) {
		return false
	}
	if exponent.Negative == POLARITY_NEGATIVE {
		return false
	}
	exponent_word := uint64(0)
	if exponent.Count != Word_Count(WORD_COUNT_MINIMUM) {
		exponent_word = uint64(exponent.Words[WORD_COUNT_MINIMUM])
	}
	result, factor := uint64(bits.CARRY_MAXIMUM), uint64(base)
	for exponent_word != 0 {
		if exponent_word&uint64(bits.CARRY_MAXIMUM) != 0 {
			if result != 0 {
				if factor > uint64(bits.WORD_64_MAXIMUM)/result {
					return false
				}
			}
			result *= factor
		}
		exponent_word >>= WORD_COUNT_INCREMENT
		if exponent_word != 0 {
			if factor != 0 {
				if factor > uint64(bits.WORD_64_MAXIMUM)/factor {
					return false
				}
			}
			factor *= factor
		}
	}
	negative := POLARITY_NONNEGATIVE
	if exponent.Count != Word_Count(WORD_COUNT_MINIMUM) {
		if exponent.Words[WORD_COUNT_MINIMUM]&Word(bits.CARRY_MAXIMUM) != 0 {
			negative = base_negative
		}
	}
	int_set_word(destination, Word(result), negative)
	return true
}

func int_modular_exponent_word(
	destination, base, exponent Int_Handle, modulus Int_Division_Nonzero_Word,
) (matched Boolean) {
	defer func() { Boolean_Invariants(matched, "int_modular_exponent_word.matched") }()
	Int_Handle_Invariants(destination, "int_modular_exponent_word.destination")
	Int_Handle_Invariants(base, "int_modular_exponent_word.base")
	Int_Handle_Invariants(exponent, "int_modular_exponent_word.exponent")
	Int_Division_Nonzero_Word_Invariants(
		modulus, "int_modular_exponent_word.modulus",
	)
	if base.Count > Word_Count(WORD_COUNT_INCREMENT) {
		return false
	}
	if exponent.Count > Word_Count(WORD_COUNT_INCREMENT) {
		return false
	}
	if base.Negative == POLARITY_NEGATIVE {
		return false
	}
	if exponent.Negative == POLARITY_NEGATIVE {
		return false
	}
	modulus_word := uint64(modulus)
	base_word := uint64(0)
	if base.Count != Word_Count(WORD_COUNT_MINIMUM) {
		base_word = uint64(base.Words[WORD_COUNT_MINIMUM])
	}
	exponent_word := uint64(0)
	if exponent.Count != Word_Count(WORD_COUNT_MINIMUM) {
		exponent_word = uint64(exponent.Words[WORD_COUNT_MINIMUM])
	}
	result, factor := uint64(bits.CARRY_MAXIMUM)%modulus_word, base_word%modulus_word
	for exponent_word != 0 {
		if exponent_word&uint64(bits.CARRY_MAXIMUM) != 0 {
			if result != 0 {
				if factor > uint64(bits.WORD_64_MAXIMUM)/result {
					return false
				}
			}
			result = result * factor % modulus_word
		}
		exponent_word >>= WORD_COUNT_INCREMENT
		if exponent_word != 0 {
			if factor != 0 {
				if factor > uint64(bits.WORD_64_MAXIMUM)/factor {
					return false
				}
			}
			factor = factor * factor % modulus_word
		}
	}
	int_set_word(destination, Word(result), POLARITY_NONNEGATIVE)
	return true
}

func int_modular_inverse_word(
	destination Int_Handle, value Int_Handle, modulus Int_Handle,
) (matched Boolean) {
	defer func() { Boolean_Invariants(matched, "int_modular_inverse_word.matched") }()
	Int_Handle_Invariants(destination, "int_modular_inverse_word.destination")
	Int_Handle_Invariants(value, "int_modular_inverse_word.value")
	Int_Handle_Invariants(modulus, "int_modular_inverse_word.modulus")
	if value.Count != Word_Count(WORD_COUNT_INCREMENT) {
		return false
	}
	if modulus.Count != Word_Count(WORD_COUNT_INCREMENT) {
		return false
	}
	modulus_word := modulus.Words[WORD_COUNT_MINIMUM]
	if modulus_word > Word(bits.INTEGER_32_MAXIMUM) {
		return false
	}
	value_word := int64(value.Words[WORD_COUNT_MINIMUM])
	if value.Negative == POLARITY_NEGATIVE {
		value_word = -value_word
	}
	modulus_signed := int64(modulus_word)
	old_remainder, remainder := modulus_signed, value_word%modulus_signed
	if remainder < 0 {
		remainder += modulus_signed
	}
	old_coefficient, coefficient := int64(0), int64(1)
	for remainder != 0 {
		quotient := old_remainder / remainder
		old_remainder, remainder = remainder, old_remainder-quotient*remainder
		old_coefficient, coefficient =
			coefficient, old_coefficient-quotient*coefficient
	}
	if old_remainder != int64(bits.CARRY_MAXIMUM) {
		return false
	}
	if old_coefficient < 0 {
		old_coefficient += modulus_signed
	}
	int_set_word(destination, Word(old_coefficient), POLARITY_NONNEGATIVE)
	return true
}

// Modular_Inverse_Result_Status excludes the zero modulus rejected by its caller.
type Modular_Inverse_Result_Status uint8

// Modular_Inverse_Result_Status_Invariants admits a result or mathematical absence.
func Modular_Inverse_Result_Status_Invariants(
	value Modular_Inverse_Result_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_RESULT_ABSENT)).
		Ensure()
}

func int_modular_inverse_coefficient_bounded(
	multiplication Int_Multiplication_Workspace_Handle,
	quotient_value Bounded_Quotient,
	coefficient_references Coefficient_References,
	double_word Boolean,
) {
	Int_Multiplication_Workspace_Handle_Invariants(
		multiplication, "int_modular_inverse_coefficient_bounded.multiplication",
	)
	Bounded_Quotient_Invariants(
		quotient_value, "int_modular_inverse_coefficient_bounded.quotient",
	)
	Coefficient_References_Invariants(
		coefficient_references, "int_modular_inverse_coefficient_bounded.coefficient",
	)
	Boolean_Invariants(double_word, "int_modular_inverse_coefficient_bounded.double_word")
	quotient := (*Int)(&quotient_value)
	coefficient_remainder := (*Int)((*Int_Destination)(coefficient_references.Destination))
	coefficient_dividend := (*Int)((*Coefficient)(coefficient_references.Left))
	coefficient_divisor := (*Int)((*Nonzero_Coefficient)(coefficient_references.Right))
	if quotient.Count == Word_Count(WORD_COUNT_INCREMENT) {
		if double_word {
			int_modular_inverse_coefficient_double_word(
				Int_Division_Nonzero_Word(quotient.Words[WORD_COUNT_MINIMUM]),
				coefficient_references.Destination,
				Short_Coefficient(*coefficient_dividend),
				Nonzero_Short_Coefficient(*coefficient_divisor),
			)
			return
		}
		int_modular_inverse_coefficient_word(
			Int_Division_Nonzero_Word(quotient.Words[WORD_COUNT_MINIMUM]),
			coefficient_references,
		)
		return
	}
	multiplication_status := Int_Multiply(
		coefficient_remainder, quotient, coefficient_divisor, multiplication,
	)
	aver.Always(
		multiplication_status == Arithmetic_Status(STATUS_OK),
		"The bounded coefficient product reserves one subtraction carry word.",
	)
	subtraction_status := Int_Subtract(
		coefficient_remainder, coefficient_dividend, coefficient_remainder,
	)
	aver.Always(
		subtraction_status == Arithmetic_Status(STATUS_OK),
		"Extended Euclid keeps every bounded coefficient below its modulus.",
	)
}

func int_modular_inverse_coefficient_word(
	quotient Int_Division_Nonzero_Word, coefficient_references Coefficient_References,
) {
	Int_Division_Nonzero_Word_Invariants(
		quotient, "int_modular_inverse_coefficient_word.quotient",
	)
	Coefficient_References_Invariants(
		coefficient_references, "int_modular_inverse_coefficient_word.coefficient",
	)
	destination := (*Int)((*Int_Destination)(coefficient_references.Destination))
	dividend := (*Int)((*Coefficient)(coefficient_references.Left))
	coefficient := (*Int)((*Nonzero_Coefficient)(coefficient_references.Right))
	previous_count := destination.Count
	factor := Word(quotient)
	carry := Word(0)
	count := max(coefficient.Count, dividend.Count)
	for index := Word_Count(WORD_COUNT_MINIMUM); index < count; index++ {
		left := uint64(0)
		if index < coefficient.Count {
			left = uint64(coefficient.Words[index])
		}
		right := uint64(factor)
		word_mask := uint64(bits.WORD_32_MAXIMUM)
		partial := (left & word_mask) * (right & word_mask)
		middle_first := (left>>bits.BIT_COUNT_32_MAXIMUM)*(right&word_mask) +
			partial>>bits.BIT_COUNT_32_MAXIMUM
		middle_second := (left&word_mask)*(right>>bits.BIT_COUNT_32_MAXIMUM) +
			middle_first&word_mask
		high := (left>>bits.BIT_COUNT_32_MAXIMUM)*
			(right>>bits.BIT_COUNT_32_MAXIMUM) +
			middle_first>>bits.BIT_COUNT_32_MAXIMUM +
			middle_second>>bits.BIT_COUNT_32_MAXIMUM
		low := left * right
		sum_product := low + uint64(carry)
		carry_output := Word(bits.CARRY_MINIMUM)
		if sum_product < low {
			carry_output = Word(bits.CARRY_MAXIMUM)
		}
		addend := uint64(0)
		if index < dividend.Count {
			addend = uint64(dividend.Words[index])
		}
		sum := sum_product + addend
		if sum < sum_product {
			carry_output++
		}
		destination.Words[index] = Word(sum)
		carry = Word(high) + carry_output
	}
	if carry != 0 {
		destination.Words[count] = carry
		count++
	}
	destination.Count = count
	destination.Negative = POLARITY_NEGATIVE
	if coefficient.Negative == POLARITY_NEGATIVE {
		destination.Negative = POLARITY_NONNEGATIVE
	}
	int_clear(destination, count, previous_count)
}

func int_modular_inverse_commit(
	bounded Boolean,
	destination Int_Handle,
	modulus_reference Positive_Magnitude_Handle, dividend_reference Positive_Magnitude_Handle,
	coefficient_reference Int_Left_Reference,
) (status Modular_Inverse_Result_Status) {
	defer func() {
		Modular_Inverse_Result_Status_Invariants(
			status, "int_modular_inverse_commit.status",
		)
	}()
	Boolean_Invariants(bounded, "int_modular_inverse_commit.bounded")
	Int_Handle_Invariants(destination, "int_modular_inverse_commit.destination")
	Positive_Magnitude_Handle_Invariants(
		modulus_reference, "int_modular_inverse_commit.modulus")
	Positive_Magnitude_Handle_Invariants(
		dividend_reference, "int_modular_inverse_commit.dividend")
	Int_Left_Reference_Invariants(
		coefficient_reference, "int_modular_inverse_commit.coefficient")
	modulus := (*Int)((*Positive_Magnitude)(modulus_reference))
	dividend := (*Int)((*Positive_Magnitude)(dividend_reference))
	coefficient := (*Int)((*Int_Left)(coefficient_reference))
	if dividend.Count != WORD_COUNT_INCREMENT {
		return STATUS_RESULT_ABSENT
	}
	if dividend.Words[WORD_COUNT_MINIMUM] != Word(bits.CARRY_MAXIMUM) {
		return STATUS_RESULT_ABSENT
	}
	if bounded {
		if coefficient.Negative == POLARITY_NEGATIVE {
			addition := Int_Add(coefficient, coefficient, modulus)
			aver.Always(
				addition == Arithmetic_Status(STATUS_OK),
				"A bounded Bezout coefficient magnitude is below its modulus.",
			)
		}
	}
	Int_Set(destination, coefficient)
	return STATUS_OK
}

func int_modular_inverse_euclidean_general(
	workspace Int_Modular_Workspace_Handle,
	bounded Boolean,
	destination Int_Handle,
) (status Modular_Inverse_Result_Status) {
	defer func() {
		Modular_Inverse_Result_Status_Invariants(
			status, "int_modular_inverse_euclidean_general.status",
		)
	}()
	Int_Modular_Workspace_Handle_Invariants(
		workspace, "int_modular_inverse_euclidean_general.workspace",
	)
	Boolean_Invariants(bounded, "int_modular_inverse_euclidean_general.bounded")
	Int_Handle_Invariants(
		destination, "int_modular_inverse_euclidean_general.destination",
	)
	modulus := &workspace.Integers[MODULAR_MODULUS_INDEX]
	quotient_reference := Int_Right_Reference(
		(*Int_Right)(&workspace.Integers[MODULAR_QUOTIENT_INDEX]))
	remainder := &workspace.Integers[MODULAR_GCD_REMAINDER_INDEX]
	dividend := &workspace.Integers[MODULAR_GCD_DIVIDEND_INDEX]
	divisor := &workspace.Integers[MODULAR_GCD_DIVISOR_INDEX]
	coefficient_remainder := &workspace.Integers[MODULAR_COEFFICIENT_REMAINDER_INDEX]
	coefficient_dividend := &workspace.Integers[MODULAR_COEFFICIENT_DIVIDEND_INDEX]
	coefficient_divisor := &workspace.Integers[MODULAR_COEFFICIENT_DIVISOR_INDEX]
	coefficient_references := Coefficient_References{
		Destination: Int_Destination_Reference((*Int_Destination)(coefficient_remainder)),
		Left:        (*Coefficient)(coefficient_dividend),
		Right:       (*Nonzero_Coefficient)(coefficient_divisor),
	}
	multiplication := (*Int_Multiplication_Workspace)(&workspace.Multiplication)
	for divisor.Count > WORD_COUNT_MINIMUM {
		int_modular_inverse_divide(
			(*Int_Division_Workspace)(&workspace.Division),
			quotient_reference, (*Int_Destination)(remainder),
			(*Positive_Magnitude)(dividend), (*Positive_Magnitude)(divisor),
		)
		if !bounded {
			int_modular_multiply(workspace, MODULAR_MULTIPLICATION_COEFFICIENT)
			int_modular_subtract(workspace.Integers)
		} else {
			int_modular_inverse_coefficient_bounded(
				multiplication, Bounded_Quotient(*quotient_reference),
				coefficient_references,
				Boolean(modulus.Count <= Word_Count(BASE_BINARY)),
			)
		}
		if bounded {
			dividend, divisor, remainder = divisor, remainder, dividend
			spare, previous, current := int_references_rotate(
				(*Nonzero_Coefficient)((*Int_Destination)(
					coefficient_references.Destination)),
				coefficient_references.Left, coefficient_references.Right)
			coefficient_references = Coefficient_References{
				Destination: (*Int_Destination)((*Coefficient)(spare)),
				Left:        (*Coefficient)((*Nonzero_Coefficient)(previous)),
				Right:       current,
			}
			coefficient_dividend = (*Int)((*Nonzero_Coefficient)(previous))
		} else {
			Int_Set(dividend, divisor)
			Int_Set(divisor, remainder)
			int_zero(remainder, remainder.Count)
			Int_Set(coefficient_dividend, coefficient_divisor)
			Int_Set(coefficient_divisor, coefficient_remainder)
			int_zero(coefficient_remainder, coefficient_remainder.Count)
		}
	}
	return int_modular_inverse_commit(
		bounded, destination, (*Positive_Magnitude)(modulus),
		(*Positive_Magnitude)(dividend), (*Int_Left)(coefficient_dividend))
}

func int_jacobi_word(
	numerator Int_Handle, denominator Int_Handle,
) (symbol Jacobi_Symbol, matched Boolean) {
	defer func() {
		Jacobi_Symbol_Invariants(symbol, "int_jacobi_word.symbol")
		Boolean_Invariants(matched, "int_jacobi_word.matched")
	}()
	Int_Handle_Invariants(numerator, "int_jacobi_word.numerator")
	Int_Handle_Invariants(denominator, "int_jacobi_word.denominator")
	if numerator.Count > Word_Count(WORD_COUNT_INCREMENT) {
		return 0, false
	}
	if denominator.Count != Word_Count(WORD_COUNT_INCREMENT) {
		return 0, false
	}
	if numerator.Negative == POLARITY_NEGATIVE {
		return 0, false
	}
	if denominator.Negative == POLARITY_NEGATIVE {
		return 0, false
	}
	denominator_word := uint64(denominator.Words[WORD_COUNT_MINIMUM])
	if denominator_word&uint64(bits.CARRY_MAXIMUM) == 0 {
		return 0, false
	}
	numerator_word := uint64(0)
	if numerator.Count != Word_Count(WORD_COUNT_MINIMUM) {
		numerator_word = uint64(numerator.Words[WORD_COUNT_MINIMUM])
	}
	result := int(bits.CARRY_MAXIMUM)
	numerator_word %= denominator_word
	for numerator_word != 0 {
		for numerator_word&uint64(bits.CARRY_MAXIMUM) == 0 {
			numerator_word >>= WORD_COUNT_INCREMENT
			residue := denominator_word & 7
			if residue == 3 {
				result = -result
			} else if residue == 5 {
				result = -result
			}
		}
		numerator_word, denominator_word = denominator_word, numerator_word
		if numerator_word&3 == 3 {
			if denominator_word&3 == 3 {
				result = -result
			}
		}
		numerator_word %= denominator_word
	}
	if denominator_word != uint64(bits.CARRY_MAXIMUM) {
		return JACOBI_SYMBOL_ZERO, true
	}
	return Jacobi_Symbol(result), true
}

// Bytes_Validate centralizes hostile byte rejection before arithmetic reads input.
func Bytes_Validate(value Bytes_Unvalidated) (validated Bytes, status Validation_Status) {
	defer func() {
		Bytes_Invariants(validated, "bytes_validate.validated")
		Validation_Status_Invariants(status, "bytes_validate.status")
	}()
	Bytes_Unvalidated_Invariants(value, "bytes_validate.value")
	if len(value) > bytes.SLICE_SIZE_MAXIMUM {
		return nil, STATUS_INPUT_INVALID
	}
	return Bytes(value), STATUS_OK
}

// Shift_Count_Validate rejects a distance larger than every bounded Int magnitude.
func Shift_Count_Validate(
	value Shift_Count_Unvalidated,
) (validated Shift_Count, status Validation_Status) {
	defer func() {
		Shift_Count_Invariants(validated, "shift_count_validate.validated")
		Validation_Status_Invariants(status, "shift_count_validate.status")
	}()
	Shift_Count_Unvalidated_Invariants(value, "shift_count_validate.value")
	if value > BIT_COUNT_MAXIMUM {
		return 0, STATUS_INPUT_INVALID
	}
	return Shift_Count(value), STATUS_OK
}

// Bit_Index_Validate rejects coordinates outside one bounded signed value.
func Bit_Index_Validate(
	value Bit_Index_Unvalidated,
) (validated Bit_Index, status Validation_Status) {
	defer func() {
		Bit_Index_Invariants(validated, "bit_index_validate.validated")
		Validation_Status_Invariants(status, "bit_index_validate.status")
	}()
	Bit_Index_Unvalidated_Invariants(value, "bit_index_validate.value")
	if value < BIT_COUNT_MINIMUM {
		return 0, STATUS_INPUT_INVALID
	}
	if value > BIT_INDEX_MAXIMUM {
		return 0, STATUS_INPUT_INVALID
	}
	return Bit_Index(value), STATUS_OK
}

// Base_Validate rejects unsupported alphabets before text conversion.
func Base_Validate(value Base_Unvalidated) (base Base, status Validation_Status) {
	defer func() {
		Base_Invariants(base, "base_validate.base")
		Validation_Status_Invariants(status, "base_validate.status")
	}()
	Base_Unvalidated_Invariants(value, "base_validate.value")
	if value < BASE_MINIMUM {
		return BASE_MINIMUM, STATUS_INPUT_INVALID
	}
	if value > BASE_MAXIMUM {
		return BASE_MINIMUM, STATUS_INPUT_INVALID
	}
	return Base(value), STATUS_OK
}

// Primality_Options_Validate rejects work limits before any arithmetic begins.
func Primality_Options_Validate(
	value Primality_Options_Unvalidated,
) (validated Primality_Options, status Validation_Status) {
	defer func() {
		Primality_Options_Invariants(validated, "primality_options_validate.validated")
		Validation_Status_Invariants(status, "primality_options_validate.status")
	}()
	Primality_Options_Unvalidated_Invariants(value, "primality_options_validate.value")
	validated.Parameter_Count = PRIMALITY_PARAMETER_COUNT_MINIMUM
	if value.Repetitions < PRIMALITY_REPETITION_COUNT_MINIMUM {
		return validated, STATUS_INPUT_INVALID
	}
	if value.Repetitions > PRIMALITY_REPETITION_COUNT_MAXIMUM {
		return validated, STATUS_INPUT_INVALID
	}
	if value.Parameter_Count < PRIMALITY_PARAMETER_COUNT_MINIMUM {
		return validated, STATUS_INPUT_INVALID
	}
	if value.Parameter_Count > PRIMALITY_PARAMETER_COUNT_MAXIMUM {
		return validated, STATUS_INPUT_INVALID
	}
	validated.Repetitions = Primality_Repetition_Count(value.Repetitions)
	validated.Parameter_Count = Primality_Parameter_Count(value.Parameter_Count)
	return validated, STATUS_OK
}

// Float_Precision_Validate rejects mantissas wider than inline Int storage.
func Float_Precision_Validate(
	value Float_Precision_Unvalidated,
) (precision Float_Precision, status Validation_Status) {
	defer func() {
		Float_Precision_Invariants(precision, "float_precision_validate.precision")
		Validation_Status_Invariants(status, "float_precision_validate.status")
	}()
	Float_Precision_Unvalidated_Invariants(value, "float_precision_validate.value")
	if value > FLOAT_PRECISION_UNVALIDATED_MAXIMUM-WORD_COUNT_INCREMENT {
		return FLOAT_PRECISION_MINIMUM, STATUS_INPUT_INVALID
	}
	return Float_Precision(value), STATUS_OK
}

// Rounding_Mode_Validate rejects enum values before a float policy changes.
func Rounding_Mode_Validate(
	value Rounding_Mode_Unvalidated,
) (mode Rounding_Mode, status Validation_Status) {
	defer func() {
		Rounding_Mode_Invariants(mode, "rounding_mode_validate.mode")
		Validation_Status_Invariants(status, "rounding_mode_validate.status")
	}()
	Rounding_Mode_Unvalidated_Invariants(value, "rounding_mode_validate.value")
	if value > ROUND_TO_POSITIVE_INFINITY {
		return Rounding_Mode(ROUND_TO_NEAREST_EVEN), STATUS_INPUT_INVALID
	}
	return Rounding_Mode(value), STATUS_OK
}

// Float_Exponent_Validate rejects scale work outside explicit finite range.
func Float_Exponent_Validate(
	value Float_Exponent_Unvalidated,
) (exponent Float_Exponent, status Validation_Status) {
	defer func() {
		Float_Exponent_Invariants(exponent, "float_exponent_validate.exponent")
		Validation_Status_Invariants(status, "float_exponent_validate.status")
	}()
	Float_Exponent_Unvalidated_Invariants(value, "float_exponent_validate.value")
	if value < FLOAT_EXPONENT_MINIMUM {
		return 0, STATUS_INPUT_INVALID
	}
	if value > FLOAT_EXPONENT_MAXIMUM {
		return 0, STATUS_INPUT_INVALID
	}
	return Float_Exponent(value), STATUS_OK
}

// Rat_Precision_Validate rejects decimal work beyond one bounded rational component.
func Rat_Precision_Validate(
	value Rat_Precision_Unvalidated,
) (precision Rat_Precision, status Validation_Status) {
	defer func() {
		Rat_Precision_Invariants(precision, "rat_precision_validate.precision")
		Validation_Status_Invariants(status, "rat_precision_validate.status")
	}()
	Rat_Precision_Unvalidated_Invariants(value, "rat_precision_validate.value")
	if value < RAT_PRECISION_MINIMUM {
		return RAT_PRECISION_MINIMUM, STATUS_INPUT_INVALID
	}
	if value > RAT_PRECISION_MAXIMUM {
		return RAT_PRECISION_MINIMUM, STATUS_INPUT_INVALID
	}
	return Rat_Precision(value), STATUS_OK
}

func float_copy_exact(destination Float_Handle, source Float_Handle) {
	Float_Handle_Invariants(destination, "float_copy_exact.destination_initial")
	Float_Handle_Invariants(source, "float_copy_exact.source")
	if destination == source {
		return
	}
	previous_count := destination.Mantissa.Count
	destination.Precision = source.Precision
	destination.Mode = source.Mode
	destination.Accuracy = source.Accuracy
	destination.Form = source.Form
	destination.Negative = source.Negative
	destination.Exponent = source.Exponent
	for index := Word_Count(WORD_COUNT_MINIMUM); index < source.Mantissa.Count; index++ {
		destination.Mantissa.Words[index] = source.Mantissa.Words[index]
	}
	destination.Mantissa.Count = source.Mantissa.Count
	destination.Mantissa.Negative = POLARITY_NONNEGATIVE
	if previous_count > destination.Mantissa.Count {
		int_clear(
			(*Int)(&destination.Mantissa), destination.Mantissa.Count, previous_count,
		)
	}
}

// Float_Set copies numeric value through destination precision and rounding mode.
func Float_Set(destination Float_Handle, source Float_Handle) {
	Float_Handle_Invariants(destination, "float_set.destination_initial")
	Float_Handle_Invariants(source, "float_set.source")
	if destination == source {
		destination.Accuracy = ACCURACY_EXACT
		return
	}
	precision := destination.Precision
	if precision == FLOAT_PRECISION_MINIMUM {
		precision = source.Precision
	}
	mode := destination.Mode
	if source.Form != FLOAT_FORM_FINITE {
		float_set_nonfinite(
			destination, Float_Nonfinite_Form(source.Form), source.Negative, precision,
		)
		destination.Mode = mode
		return
	}
	if source.Precision <= precision {
		float_copy_exact(destination, source)
		destination.Precision = precision
		destination.Mode = mode
		destination.Accuracy = ACCURACY_EXACT
		return
	}
	float_set_magnitude(
		destination, &source.Mantissa, source.Exponent,
		source.Negative, Float_Active_Precision(precision), mode,
	)
}

// Rat_Inverse swaps normalized components through caller workspace so exact aliasing is safe.
func Rat_Inverse(
	destination Rat_Handle, source Rat_Handle, workspace Rat_Workspace_Handle,
) (status Divisor_Status) {
	defer func() { Divisor_Status_Invariants(status, "rat_inverse.status") }()
	Rat_Handle_Invariants(destination, "rat_inverse.destination")
	Rat_Handle_Invariants(source, "rat_inverse.source")
	Rat_Workspace_Handle_Invariants(workspace, "rat_inverse.workspace")
	source_numerator := &source.Integers.Numerator
	if source_numerator.Count == WORD_COUNT_MINIMUM {
		return STATUS_DIVISOR_ZERO
	}
	source_denominator := (*Int)(&source.Integers.Denominator)
	if source_numerator.Count == Word_Count(BASE_BINARY) {
		if source_denominator.Count == Word_Count(BASE_BINARY) {
			rat_copy_double_words(
				(*Rat_Numerator_Storage)(&destination.Integers.Numerator),
				(*Rat_Denominator_Storage)(&destination.Integers.Denominator),
				Double_Words{Low: source_denominator.Words[WORD_COUNT_MINIMUM],
					High: Float_Active_Word(
						source_denominator.Words[WORD_COUNT_INCREMENT])},
				Double_Words{Low: source_numerator.Words[WORD_COUNT_MINIMUM],
					High: Float_Active_Word(
						source_numerator.Words[WORD_COUNT_INCREMENT])},
				source_numerator.Negative,
			)
			return STATUS_OK
		}
	}
	if source_numerator.Count != Word_Count(WORD_COUNT_INCREMENT) {
		rat_inverse_many(
			(*Rat_Nonempty)((*Rat)(destination)), (*Nonzero_Rat)((*Rat)(source)),
			&workspace.Integers.Result_Numerator,
			&workspace.Integers.Result_Denominator,
		)
		return STATUS_OK
	}
	if source_denominator.Count > Word_Count(WORD_COUNT_INCREMENT) {
		rat_inverse_many(
			(*Rat_Nonempty)((*Rat)(destination)), (*Nonzero_Rat)((*Rat)(source)),
			&workspace.Integers.Result_Numerator,
			&workspace.Integers.Result_Denominator,
		)
		return STATUS_OK
	}
	numerator_word := Word(bits.CARRY_MAXIMUM)
	if source_denominator.Count != Word_Count(WORD_COUNT_MINIMUM) {
		numerator_word = source_denominator.Words[WORD_COUNT_MINIMUM]
	}
	denominator_word := source_numerator.Words[WORD_COUNT_MINIMUM]
	destination_numerator := (*Int)(&destination.Integers.Numerator)
	destination_denominator := (*Int)(&destination.Integers.Denominator)
	previous_numerator_count := destination_numerator.Count
	previous_denominator_count := destination_denominator.Count
	destination_numerator.Words[WORD_COUNT_MINIMUM] = numerator_word
	destination_numerator.Count = Word_Count(WORD_COUNT_INCREMENT)
	destination_numerator.Negative = source_numerator.Negative
	if denominator_word == Word(bits.CARRY_MAXIMUM) {
		destination_denominator.Count = Word_Count(WORD_COUNT_MINIMUM)
		destination_denominator.Negative = POLARITY_NONNEGATIVE
	} else {
		destination_denominator.Words[WORD_COUNT_MINIMUM] = denominator_word
		destination_denominator.Count = Word_Count(WORD_COUNT_INCREMENT)
		destination_denominator.Negative = POLARITY_NONNEGATIVE
	}
	int_clear(destination_numerator, destination_numerator.Count, previous_numerator_count)
	int_clear(
		destination_denominator, destination_denominator.Count,
		previous_denominator_count,
	)
	return STATUS_OK
}

func rat_inverse_many(
	reference Rat_Nonempty_Handle, source Nonzero_Rat_Handle,
	numerator_storage Rat_Result_Numerator_Handle,
	denominator_storage Rat_Result_Denominator_Handle,
) {
	Rat_Nonempty_Handle_Invariants(reference, "rat_inverse_many.destination")
	Nonzero_Rat_Handle_Invariants(source, "rat_inverse_many.source")
	Rat_Result_Numerator_Handle_Invariants(
		numerator_storage, "rat_inverse_many.result_numerator")
	Rat_Result_Denominator_Handle_Invariants(
		denominator_storage, "rat_inverse_many.result_denominator")
	destination := (*Rat)((*Rat_Nonempty)(reference))
	result_numerator := (*Int)((*Rat_Result_Numerator)(numerator_storage))
	result_denominator := (*Int)((*Rat_Result_Denominator)(denominator_storage))
	source_numerator := (*Int)(&source.Integers.Numerator)
	source_denominator := (*Int)(&source.Integers.Denominator)
	if (*Rat)(destination) != (*Rat)((*Nonzero_Rat)(source)) {
		destination_numerator := (*Int)(&destination.Integers.Numerator)
		if source_denominator.Count == WORD_COUNT_MINIMUM {
			int_set_word(
				destination_numerator, Word(bits.CARRY_MAXIMUM),
				source_numerator.Negative,
			)
		} else {
			Int_Set(destination_numerator, source_denominator)
			destination_numerator.Negative = source_numerator.Negative
		}
		destination_denominator := (*Int)(&destination.Integers.Denominator)
		if source_numerator.Count == Word_Count(WORD_COUNT_INCREMENT) {
			if source_numerator.Words[WORD_COUNT_MINIMUM] == Word(bits.CARRY_MAXIMUM) {
				int_zero(destination_denominator, destination_denominator.Count)
				return
			}
		}
		Int_Set(destination_denominator, source_numerator)
		destination_denominator.Negative = POLARITY_NONNEGATIVE
		return
	}
	Int_Set(result_numerator, (*Int)(&source.Integers.Denominator))
	if result_numerator.Count == WORD_COUNT_MINIMUM {
		Int_Set_Uint_64(result_numerator, Word_64(bits.CARRY_MAXIMUM))
	}
	result_numerator.Negative = source_numerator.Negative
	Int_Set(result_denominator, source_numerator)
	result_denominator.Negative = POLARITY_NONNEGATIVE
	if result_denominator.Count == WORD_COUNT_INCREMENT {
		if result_denominator.Words[WORD_COUNT_MINIMUM] == Word(bits.CARRY_MAXIMUM) {
			int_zero(result_denominator, result_denominator.Count)
		}
	}
	Int_Set((*Int)(&destination.Integers.Numerator), result_numerator)
	Int_Set((*Int)(&destination.Integers.Denominator), result_denominator)
	return
}

// Int_Absolute writes a distinct value so callers choose whether aliasing destroys source sign.
func Int_Absolute(destination Int_Handle, source Int_Handle) {
	Int_Handle_Invariants(destination, "int_absolute.destination")
	Int_Handle_Invariants(source, "int_absolute.source")
	if destination != source {
		previous_count := destination.Count
		if source.Count == Word_Count(BASE_BINARY) {
			destination.Words[WORD_COUNT_MINIMUM] = source.Words[WORD_COUNT_MINIMUM]
			destination.Words[WORD_COUNT_INCREMENT] = source.Words[WORD_COUNT_INCREMENT]
		} else {
			for index := Word_Count(WORD_COUNT_MINIMUM); index < source.Count; index++ {
				destination.Words[index] = source.Words[index]
			}
		}
		destination.Count = source.Count
		destination.Negative = POLARITY_NONNEGATIVE
		if previous_count > destination.Count {
			int_clear(destination, destination.Count, previous_count)
		}
	}
	destination.Negative = POLARITY_NONNEGATIVE
}

// Int_Negate toggles sign only for nonzero magnitude so zero retains one representation.
func Int_Negate(destination Int_Handle, source Int_Handle) {
	Int_Handle_Invariants(destination, "int_negate.destination")
	Int_Handle_Invariants(source, "int_negate.source")
	negative := POLARITY_NONNEGATIVE
	if source.Count != WORD_COUNT_MINIMUM {
		negative = POLARITY_NEGATIVE - source.Negative
	}
	if destination != source {
		if source.Count == Word_Count(WORD_COUNT_INCREMENT) {
			previous_count := destination.Count
			destination.Words[WORD_COUNT_MINIMUM] = source.Words[WORD_COUNT_MINIMUM]
			destination.Count = source.Count
			destination.Negative = negative
			if previous_count > destination.Count {
				int_clear(destination, destination.Count, previous_count)
			}
			return
		}
		if source.Count == Word_Count(BASE_BINARY) {
			low := source.Words[WORD_COUNT_MINIMUM]
			high := source.Words[WORD_COUNT_INCREMENT]
			previous_count := destination.Count
			destination.Words[WORD_COUNT_MINIMUM] = low
			destination.Words[WORD_COUNT_INCREMENT] = high
			destination.Count = Word_Count(BASE_BINARY)
			destination.Negative = negative
			if previous_count > destination.Count {
				int_clear(destination, destination.Count, previous_count)
			}
			return
		}
	}
	if destination != source {
		previous_count := destination.Count
		for index := Word_Count(WORD_COUNT_MINIMUM); index < source.Count; index++ {
			destination.Words[index] = source.Words[index]
		}
		destination.Count = source.Count
		destination.Negative = negative
		if previous_count > destination.Count {
			int_clear(destination, destination.Count, previous_count)
		}
	}
	destination.Negative = negative
}

// Int_Subtract reuses sign-magnitude addition without constructing a full negated temporary.
func Int_Subtract(destination, left, right Int_Handle) (status Arithmetic_Status) {
	defer func() { Arithmetic_Status_Invariants(status, "int_subtract.status") }()
	Int_Handle_Invariants(destination, "int_subtract.destination")
	Int_Handle_Invariants(left, "int_subtract.left")
	Int_Handle_Invariants(right, "int_subtract.right")
	if int_subtract_positive_double_word(destination, left, right) {
		return STATUS_OK
	}
	if left.Negative == POLARITY_NONNEGATIVE {
		if right.Negative == POLARITY_NONNEGATIVE {
			if left.Count == Word_Count(WORD_COUNT_INCREMENT) {
				if right.Count == Word_Count(WORD_COUNT_INCREMENT) {
					left_word := Word(0)
					if left.Count != Word_Count(WORD_COUNT_MINIMUM) {
						left_word = left.Words[WORD_COUNT_MINIMUM]
					}
					right_word := Word(0)
					if right.Count != Word_Count(WORD_COUNT_MINIMUM) {
						right_word = right.Words[WORD_COUNT_MINIMUM]
					}
					word := left_word - right_word
					negative := POLARITY_NONNEGATIVE
					if left_word < right_word {
						word = right_word - left_word
						negative = POLARITY_NEGATIVE
					}
					previous_count := destination.Count
					destination.Count = Word_Count(WORD_COUNT_MINIMUM)
					destination.Negative = POLARITY_NONNEGATIVE
					if word != 0 {
						destination.Words[WORD_COUNT_MINIMUM] = word
						destination.Count = Word_Count(WORD_COUNT_INCREMENT)
						destination.Negative = negative
					}
					if previous_count > destination.Count {
						int_clear(
							destination, destination.Count,
							previous_count,
						)
					}
					return STATUS_OK
				}
			}
		}
	}
	return int_sum(
		destination, left, left.Negative, right, POLARITY_NEGATIVE-right.Negative,
	)
}

func int_shift_left_small(
	destination, source Int_Handle, count Shift_Count,
) (matched Boolean) {
	defer func() { Boolean_Invariants(matched, "int_shift_left_small.matched") }()
	Int_Handle_Invariants(destination, "int_shift_left_small.destination")
	Int_Handle_Invariants(source, "int_shift_left_small.source")
	Shift_Count_Invariants(count, "int_shift_left_small.count")
	if source.Count == Word_Count(BASE_BINARY) {
		if count == 0 {
			return false
		}
		if count >= Shift_Count(WORD_BIT_COUNT) {
			return false
		}
		shift := uint(count)
		low := source.Words[WORD_COUNT_MINIMUM]
		high := source.Words[WORD_COUNT_INCREMENT]
		negative := source.Negative
		previous_count := destination.Count
		carry := high >> (WORD_BIT_COUNT - shift)
		destination.Words[WORD_COUNT_INCREMENT] =
			high<<shift | low>>(WORD_BIT_COUNT-shift)
		destination.Words[WORD_COUNT_MINIMUM] = low << shift
		destination.Count = Word_Count(BASE_BINARY)
		if carry != 0 {
			destination.Words[BASE_BINARY] = carry
			destination.Count += Word_Count(WORD_COUNT_INCREMENT)
		}
		destination.Negative = negative
		int_clear(destination, destination.Count, previous_count)
		return true
	}
	if source.Count != Word_Count(WORD_COUNT_INCREMENT) {
		return false
	}
	if count >= Shift_Count(WORD_BIT_COUNT) {
		return false
	}
	source_word := source.Words[WORD_COUNT_MINIMUM]
	if source_word > Word(bits.WORD_64_MAXIMUM)>>uint(count) {
		return false
	}
	previous_count := destination.Count
	destination.Words[WORD_COUNT_MINIMUM] = source_word << uint(count)
	destination.Count = Word_Count(WORD_COUNT_INCREMENT)
	destination.Negative = source.Negative
	if previous_count > destination.Count {
		int_clear(destination, destination.Count, previous_count)
	}
	return true
}

func int_shift_right_small(
	destination, source Int_Handle, count Shift_Count,
) (matched Boolean) {
	defer func() { Boolean_Invariants(matched, "int_shift_right_small.matched") }()
	Int_Handle_Invariants(destination, "int_shift_right_small.destination")
	Int_Handle_Invariants(source, "int_shift_right_small.source")
	Shift_Count_Invariants(count, "int_shift_right_small.count")
	if source.Count == Word_Count(BASE_BINARY) {
		if count == 0 {
			return false
		}
		if count >= Shift_Count(WORD_BIT_COUNT) {
			return false
		}
		shift := uint(count)
		low, high := source.Words[WORD_COUNT_MINIMUM], source.Words[WORD_COUNT_INCREMENT]
		negative, previous_count := source.Negative, destination.Count
		result_low := low>>shift | high<<(WORD_BIT_COUNT-shift)
		result_high := high >> shift
		mask := Word((bits.CARRY_MAXIMUM << shift) - bits.CARRY_MAXIMUM)
		if negative == POLARITY_NEGATIVE {
			if low&mask != 0 {
				result_low++
				if result_low == 0 {
					result_high++
				}
			}
		}
		destination.Words[WORD_COUNT_MINIMUM] = result_low
		destination.Words[WORD_COUNT_INCREMENT] = result_high
		destination.Count = Word_Count(BASE_BINARY)
		if result_high == 0 {
			destination.Count = Word_Count(WORD_COUNT_INCREMENT)
		}
		destination.Negative = negative
		int_clear(destination, destination.Count, previous_count)
		return true
	}
	if source.Count > Word_Count(WORD_COUNT_INCREMENT) {
		return false
	}
	previous_count := destination.Count
	if source.Count == Word_Count(WORD_COUNT_MINIMUM) {
		int_zero(destination, previous_count)
		return true
	}
	word := Word(0)
	if count < Shift_Count(WORD_BIT_COUNT) {
		source_word := source.Words[WORD_COUNT_MINIMUM]
		word = source_word >> uint(count)
		if source.Negative == POLARITY_NEGATIVE {
			if source_word != word<<uint(count) {
				word++
			}
		}
	}
	if word == 0 {
		if source.Negative == POLARITY_NEGATIVE {
			word = Word(bits.CARRY_MAXIMUM)
		} else {
			int_zero(destination, previous_count)
			return true
		}
	}
	destination.Words[WORD_COUNT_MINIMUM] = word
	destination.Count = Word_Count(WORD_COUNT_INCREMENT)
	destination.Negative = source.Negative
	if previous_count > destination.Count {
		int_clear(destination, destination.Count, previous_count)
	}
	return true
}

func int_not_word(destination Int_Handle, source Int_Handle) (matched Boolean) {
	defer func() { Boolean_Invariants(matched, "int_not_word.matched") }()
	Int_Handle_Invariants(destination, "int_not_word.destination")
	Int_Handle_Invariants(source, "int_not_word.source")
	if source.Count > Word_Count(WORD_COUNT_INCREMENT) {
		return false
	}
	// Zero and negative words retain the generic sign-extension boundary witnesses.
	if source.Count == Word_Count(WORD_COUNT_MINIMUM) {
		return false
	}
	if source.Negative == POLARITY_NEGATIVE {
		return false
	}
	word := Word(0)
	if source.Count != Word_Count(WORD_COUNT_MINIMUM) {
		word = source.Words[WORD_COUNT_MINIMUM]
	}
	negative := POLARITY_NONNEGATIVE
	if source.Negative == POLARITY_NONNEGATIVE {
		if word == Word(bits.WORD_64_MAXIMUM) {
			previous_count := destination.Count
			destination.Words[WORD_COUNT_MINIMUM] = Word(bits.WORD_64_MINIMUM)
			destination.Words[WORD_COUNT_INCREMENT] = Word(bits.CARRY_MAXIMUM)
			destination.Count = Word_Count(WORD_COUNT_INCREMENT + WORD_COUNT_INCREMENT)
			destination.Negative = POLARITY_NEGATIVE
			if previous_count > destination.Count {
				int_clear(destination, destination.Count, previous_count)
			}
			return true
		}
		word++
		negative = POLARITY_NEGATIVE
	} else {
		word--
	}
	previous_count := destination.Count
	destination.Count = Word_Count(WORD_COUNT_MINIMUM)
	destination.Negative = POLARITY_NONNEGATIVE
	if word != 0 {
		destination.Words[WORD_COUNT_MINIMUM] = word
		destination.Count = Word_Count(WORD_COUNT_INCREMENT)
		destination.Negative = negative
	}
	if previous_count > destination.Count {
		int_clear(destination, destination.Count, previous_count)
	}
	return true
}

// Float_Absolute writes separately so exact receiver aliasing needs no special path.
func Float_Absolute(destination Float_Handle, source Float_Handle) {
	Float_Handle_Invariants(destination, "float_absolute.destination_initial")
	Float_Handle_Invariants(source, "float_absolute.source")
	if destination != source {
		previous_count := destination.Mantissa.Count
		source_count := source.Mantissa.Count
		destination.Precision = source.Precision
		destination.Mode = source.Mode
		destination.Form = source.Form
		destination.Exponent = source.Exponent
		if source_count == Word_Count(BASE_BINARY) {
			destination.Mantissa.Words[WORD_COUNT_MINIMUM] =
				source.Mantissa.Words[WORD_COUNT_MINIMUM]
			destination.Mantissa.Words[WORD_COUNT_INCREMENT] =
				source.Mantissa.Words[WORD_COUNT_INCREMENT]
		} else {
			for index := Word_Count(WORD_COUNT_MINIMUM); index < source_count; index++ {
				destination.Mantissa.Words[index] = source.Mantissa.Words[index]
			}
		}
		destination.Mantissa.Count = source.Mantissa.Count
		destination.Mantissa.Negative = POLARITY_NONNEGATIVE
		if previous_count > destination.Mantissa.Count {
			int_clear(
				(*Int)(&destination.Mantissa), destination.Mantissa.Count,
				previous_count,
			)
		}
	}
	destination.Negative = POLARITY_NONNEGATIVE
	destination.Accuracy = ACCURACY_EXACT
}

// Float_Set_Uint_64 stores one exact machine magnitude before destination rounding.
func Float_Set_Uint_64(destination Float_Handle, value Word_64) {
	Float_Handle_Invariants(destination, "float_set_uint_64.destination_initial")
	Word_64_Invariants(value, "float_set_uint_64.value")
	precision := destination.Precision
	if precision == FLOAT_PRECISION_MINIMUM {
		precision = WORD_BIT_COUNT
	}
	word := Word(value)
	bit_count := Float_Precision(bits.Bit_Size_64(bits.Word_64(word)))
	if word == 0 {
		float_set_nonfinite(
			destination, Float_Nonfinite_Form(FLOAT_FORM_ZERO),
			POLARITY_NONNEGATIVE, precision,
		)
		return
	}
	if bit_count > precision {
		var magnitude Float_Mantissa
		var magnitude_words [WORD_COUNT_MAXIMUM]Word
		magnitude.Words = magnitude_words[:]
		magnitude.Words[WORD_COUNT_MINIMUM] = word
		magnitude.Count = Word_Count(WORD_COUNT_INCREMENT)
		float_set_magnitude(
			destination, &magnitude, Float_Exponent(bit_count),
			POLARITY_NONNEGATIVE, Float_Active_Precision(precision), destination.Mode,
		)
		return
	}
	previous_count := destination.Mantissa.Count
	destination.Precision, destination.Accuracy = precision, ACCURACY_EXACT
	destination.Form, destination.Negative = FLOAT_FORM_FINITE, POLARITY_NONNEGATIVE
	destination.Mantissa.Words[WORD_COUNT_MINIMUM] = word
	destination.Mantissa.Count = Word_Count(WORD_COUNT_INCREMENT)
	destination.Mantissa.Negative = POLARITY_NONNEGATIVE
	destination.Exponent = Float_Exponent(bit_count)
	if previous_count > Word_Count(WORD_COUNT_INCREMENT) {
		int_clear(
			(*Int)(&destination.Mantissa), Word_Count(WORD_COUNT_INCREMENT),
			previous_count,
		)
	}
}

// Float_Set_Int_64 stores one exact signed machine value before destination rounding.
func Float_Set_Int_64(destination Float_Handle, value Int_64) {
	Float_Handle_Invariants(destination, "float_set_int_64.destination_initial")
	Int_64_Invariants(value, "float_set_int_64.value")
	negative := POLARITY_NONNEGATIVE
	magnitude := Word(value)
	if value < 0 {
		negative = POLARITY_NEGATIVE
		magnitude = Word(-(value + Int_64(bits.CARRY_MAXIMUM))) +
			Word(bits.CARRY_MAXIMUM)
	}
	precision := destination.Precision
	if precision == FLOAT_PRECISION_MINIMUM {
		precision = WORD_BIT_COUNT
	}
	bit_count := Float_Precision(bits.Bit_Size_64(bits.Word_64(magnitude)))
	if magnitude == 0 {
		float_set_nonfinite(
			destination, Float_Nonfinite_Form(FLOAT_FORM_ZERO), negative, precision,
		)
		return
	}
	if bit_count > precision {
		var magnitude_words [WORD_COUNT_MAXIMUM]Word
		var float_magnitude Float_Mantissa
		float_magnitude.Words = magnitude_words[:]
		float_magnitude.Words[WORD_COUNT_MINIMUM] = magnitude
		float_magnitude.Count = Word_Count(WORD_COUNT_INCREMENT)
		float_set_magnitude(
			destination, &float_magnitude, Float_Exponent(bit_count), negative,
			Float_Active_Precision(precision), destination.Mode,
		)
		return
	}
	previous_count := destination.Mantissa.Count
	destination.Precision, destination.Accuracy = precision, ACCURACY_EXACT
	destination.Form, destination.Negative = FLOAT_FORM_FINITE, negative
	destination.Mantissa.Words[WORD_COUNT_MINIMUM] = magnitude
	destination.Mantissa.Count = Word_Count(WORD_COUNT_INCREMENT)
	destination.Mantissa.Negative = POLARITY_NONNEGATIVE
	destination.Exponent = Float_Exponent(bit_count)
	if previous_count > Word_Count(WORD_COUNT_INCREMENT) {
		int_clear(
			(*Int)(&destination.Mantissa), Word_Count(WORD_COUNT_INCREMENT),
			previous_count,
		)
	}
}

// Float_Negate preserves the IEEE distinction between positive and negative zero.
func Float_Negate(destination Float_Handle, source Float_Handle) {
	Float_Handle_Invariants(destination, "float_negate.destination_initial")
	Float_Handle_Invariants(source, "float_negate.source")
	if destination != source {
		previous_count := destination.Mantissa.Count
		source_count := source.Mantissa.Count
		destination.Precision = source.Precision
		destination.Mode = source.Mode
		destination.Form = source.Form
		destination.Exponent = source.Exponent
		if source_count == Word_Count(BASE_BINARY) {
			destination.Mantissa.Words[WORD_COUNT_MINIMUM] =
				source.Mantissa.Words[WORD_COUNT_MINIMUM]
			destination.Mantissa.Words[WORD_COUNT_INCREMENT] =
				source.Mantissa.Words[WORD_COUNT_INCREMENT]
		} else {
			for index := Word_Count(WORD_COUNT_MINIMUM); index < source_count; index++ {
				destination.Mantissa.Words[index] = source.Mantissa.Words[index]
			}
		}
		destination.Mantissa.Count = source.Mantissa.Count
		destination.Mantissa.Negative = POLARITY_NONNEGATIVE
		if previous_count > destination.Mantissa.Count {
			int_clear(
				(*Int)(&destination.Mantissa), destination.Mantissa.Count,
				previous_count,
			)
		}
	}
	destination.Negative = POLARITY_NEGATIVE - source.Negative
	destination.Accuracy = ACCURACY_EXACT
}

// Float_Compare orders numeric values while treating both signed zeros as equal.
func Float_Compare(left Float_Handle, right Float_Handle) (order Order) {
	defer func() { Order_Invariants(order, "float_compare.order") }()
	Float_Handle_Invariants(left, "float_compare.left")
	Float_Handle_Invariants(right, "float_compare.right")
	if left.Form != FLOAT_FORM_FINITE {
		return float_compare_general(left, right)
	}
	if right.Form != FLOAT_FORM_FINITE {
		return float_compare_general(left, right)
	}
	if left.Negative != right.Negative {
		return float_compare_general(left, right)
	}
	if left.Exponent > right.Exponent {
		if left.Negative == POLARITY_NEGATIVE {
			return ORDER_BEFORE
		}
		return ORDER_AFTER
	}
	if left.Exponent < right.Exponent {
		if left.Negative == POLARITY_NEGATIVE {
			return ORDER_AFTER
		}
		return ORDER_BEFORE
	}
	if left.Mantissa.Count != Word_Count(WORD_COUNT_INCREMENT) {
		return float_compare_general(left, right)
	}
	if right.Mantissa.Count != Word_Count(WORD_COUNT_INCREMENT) {
		return float_compare_general(left, right)
	}
	left_word := left.Mantissa.Words[WORD_COUNT_MINIMUM]
	right_word := right.Mantissa.Words[WORD_COUNT_MINIMUM]
	exponent := int(left.Exponent)
	if uint(exponent-WORD_COUNT_INCREMENT) < uint(WORD_BIT_INDEX_MAXIMUM) {
		minimum := Word(bits.CARRY_MAXIMUM) << uint(exponent-WORD_COUNT_INCREMENT)
		if left_word&right_word&minimum == 0 {
			return float_compare_general(left, right)
		}
		if left_word|right_word >= minimum<<WORD_COUNT_INCREMENT {
			return float_compare_general(left, right)
		}
	} else if exponent != WORD_BIT_COUNT {
		return float_compare_general(left, right)
	}
	if left_word < right_word {
		order = ORDER_BEFORE
	} else if left_word > right_word {
		order = ORDER_AFTER
	}
	if left.Negative == POLARITY_NEGATIVE {
		order = -order
	}
	return order
}

func float_compare_general(left Float_Handle, right Float_Handle) (order Order) {
	defer func() { Order_Invariants(order, "float_compare_general.order") }()
	Float_Handle_Invariants(left, "float_compare_general.left")
	Float_Handle_Invariants(right, "float_compare_general.right")
	left_sign := SIGN_ZERO
	if left.Form != FLOAT_FORM_ZERO {
		left_sign = SIGN_POSITIVE
		if left.Negative == POLARITY_NEGATIVE {
			left_sign = SIGN_NEGATIVE
		}
	}
	right_sign := SIGN_ZERO
	if right.Form != FLOAT_FORM_ZERO {
		right_sign = SIGN_POSITIVE
		if right.Negative == POLARITY_NEGATIVE {
			right_sign = SIGN_NEGATIVE
		}
	}
	if left_sign < right_sign {
		return ORDER_BEFORE
	}
	if left_sign > right_sign {
		return ORDER_AFTER
	}
	if left_sign == SIGN_ZERO {
		return ORDER_SAME
	}
	if left.Form == FLOAT_FORM_INFINITY {
		if right.Form == FLOAT_FORM_INFINITY {
			return ORDER_SAME
		}
		order = ORDER_AFTER
	} else if right.Form == FLOAT_FORM_INFINITY {
		order = ORDER_BEFORE
	} else {
		order = float_compare_magnitude(
			Positive_Int(left.Mantissa), Positive_Int(right.Mantissa),
			left.Exponent, right.Exponent)
	}
	if left_sign == SIGN_NEGATIVE {
		order = -order
	}
	return order
}

func float_multiply_exact_word(
	destination Float_Handle, left Float_Active_Word, right Float_Active_Word,
	left_exponent Float_Exponent, right_exponent Float_Exponent,
	precision Float_Active_Precision, negative Polarity,
) (matched Boolean) {
	defer func() { Boolean_Invariants(matched, "float_multiply_exact_word.matched") }()
	Float_Handle_Invariants(destination, "float_multiply_exact_word.destination_initial")
	Float_Active_Word_Invariants(left, "float_multiply_exact_word.left")
	Float_Active_Word_Invariants(right, "float_multiply_exact_word.right")
	Float_Exponent_Invariants(left_exponent, "float_multiply_exact_word.left_exponent")
	Float_Exponent_Invariants(right_exponent, "float_multiply_exact_word.right_exponent")
	Float_Active_Precision_Invariants(precision, "float_multiply_exact_word.precision")
	Polarity_Invariants(negative, "float_multiply_exact_word.negative")
	if uint(left_exponent-WORD_COUNT_INCREMENT) >= uint(WORD_BIT_INDEX_MAXIMUM) {
		return false
	}
	if uint(right_exponent-WORD_COUNT_INCREMENT) >= uint(WORD_BIT_INDEX_MAXIMUM) {
		return false
	}
	left_minimum := Word(bits.CARRY_MAXIMUM) << uint(left_exponent-WORD_COUNT_INCREMENT)
	right_minimum := Word(bits.CARRY_MAXIMUM) << uint(right_exponent-WORD_COUNT_INCREMENT)
	left_word := Word(left)
	right_word := Word(right)
	if left_word&left_minimum == 0 {
		return false
	}
	if left_word >= left_minimum<<WORD_COUNT_INCREMENT {
		return false
	}
	if right_word&right_minimum == 0 {
		return false
	}
	if right_word >= right_minimum<<WORD_COUNT_INCREMENT {
		return false
	}
	high, low := bits.Multiply_64(
		bits.Word_64(left_word), bits.Multiplier_64(right_word),
	)
	if high != 0 {
		return false
	}
	bit_count := Float_Precision(bits.Bit_Size_64(bits.Word_64(low)))
	if bit_count > Float_Precision(precision) {
		return false
	}
	mantissa := &destination.Mantissa
	previous_count := mantissa.Count
	destination.Precision, destination.Accuracy = Float_Precision(precision), ACCURACY_EXACT
	destination.Form = FLOAT_FORM_FINITE
	destination.Negative = negative
	mantissa.Words[WORD_COUNT_MINIMUM] = Word(low)
	mantissa.Count, mantissa.Negative = Word_Count(WORD_COUNT_INCREMENT), POLARITY_NONNEGATIVE
	destination.Exponent = Float_Exponent(bit_count)
	int_clear((*Int)(mantissa), mantissa.Count, previous_count)
	return true
}

func float_multiply_general(
	destination Float_Handle, left Float_Handle, right Float_Handle,
	workspace Float_Multiplication_Workspace_Handle, precision Float_Precision,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "float_multiply_general.status") }()
	Float_Handle_Invariants(destination, "float_multiply_general.destination_initial")
	Float_Handle_Invariants(left, "float_multiply_general.left")
	Float_Handle_Invariants(right, "float_multiply_general.right")
	Float_Multiplication_Workspace_Handle_Invariants(
		workspace, "float_multiply_general.workspace")
	Float_Precision_Invariants(precision, "float_multiply_general.precision")
	mode := destination.Mode
	negative := POLARITY_NONNEGATIVE
	if left.Negative != right.Negative {
		negative = POLARITY_NEGATIVE
	}
	if left.Form == FLOAT_FORM_ZERO {
		if right.Form == FLOAT_FORM_INFINITY {
			return STATUS_INPUT_INVALID
		}
		zero := *left
		zero.Negative = negative
		float_set_operation_operand(destination, &zero, precision, mode)
		return STATUS_OK
	}
	if right.Form == FLOAT_FORM_ZERO {
		if left.Form == FLOAT_FORM_INFINITY {
			return STATUS_INPUT_INVALID
		}
		zero := *right
		zero.Negative = negative
		float_set_operation_operand(destination, &zero, precision, mode)
		return STATUS_OK
	}
	if left.Form == FLOAT_FORM_INFINITY {
		float_set_operation_infinity(destination, negative, precision, mode)
		return STATUS_OK
	}
	if right.Form == FLOAT_FORM_INFINITY {
		float_set_operation_infinity(destination, negative, precision, mode)
		return STATUS_OK
	}
	float_multiply_finite(
		destination, (*Float_Finite)((*Float)(left)),
		(*Float_Finite)((*Float)(right)), workspace,
		Float_Active_Precision(precision), mode, negative,
	)
	return STATUS_OK
}

// Float_Active_Word_Count excludes zero after finite validation proves a mantissa.
type Float_Active_Word_Count int

// Float_Active_Word_Count_Invariants preserves the finite mantissa count proof.
func Float_Active_Word_Count_Invariants(
	value Float_Active_Word_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), WORD_COUNT_INCREMENT, WORD_COUNT_MAXIMUM).
		Ensure()
}

// Float_Active_Word excludes zero after one-word normalization is proven.
type Float_Active_Word Word

// Float_Active_Word_Invariants preserves the normalized one-word proof.
func Float_Active_Word_Invariants(value Float_Active_Word, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(
			uint64(value), uint64(bits.CARRY_MAXIMUM),
			uint64(bits.WORD_64_MAXIMUM),
		).
		Ensure()
}

// Float_Add_Exact_Sum excludes one because two active magnitudes cannot produce it.
type Float_Add_Exact_Sum Word

// Float_Add_Exact_Sum_Invariants excludes the sole unreachable native sum.
func Float_Add_Exact_Sum_Invariants(
	value Float_Add_Exact_Sum, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Uint64(
			uint64(value), uint64(bits.CARRY_MINIMUM), uint64(bits.WORD_64_MAXIMUM),
			uint64(bits.CARRY_MAXIMUM), uint64(bits.CARRY_MAXIMUM),
			uint64(bits.CARRY_MAXIMUM),
		).
		Ensure()
}

// Float_Add_Exact_Exponent is zero on no match or one native-word bit width.
type Float_Add_Exact_Exponent int

// Float_Add_Exact_Exponent_Invariants bounds no-match and native exact results.
func Float_Add_Exact_Exponent_Invariants(
	value Float_Add_Exact_Exponent, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Holed_Int(
			int(value), BIT_COUNT_MINIMUM, WORD_BIT_COUNT,
			WORD_COUNT_INCREMENT, WORD_COUNT_INCREMENT,
			WORD_COUNT_INCREMENT, WORD_COUNT_INCREMENT,
		).
		Ensure()
}

func float_add_precision(
	destination Float_Precision, left Float_Precision, right Float_Precision,
) (precision Float_Precision) {
	defer func() { Float_Precision_Invariants(precision, "float_add_precision.precision") }()
	Float_Precision_Invariants(destination, "float_add_precision.destination")
	Float_Precision_Invariants(left, "float_add_precision.left")
	Float_Precision_Invariants(right, "float_add_precision.right")
	precision = destination
	if precision != FLOAT_PRECISION_MINIMUM {
		return precision
	}
	precision = left
	if right > precision {
		precision = right
	}
	return precision
}

// Float_Add keeps the exact alignment span in caller storage so aliases cannot destroy operands.
func Float_Add(
	destination, left, right Float_Handle, workspace Float_Addition_Workspace_Handle,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "float_add.status") }()
	Float_Handle_Invariants(destination, "float_add.destination_initial")
	Float_Handle_Invariants(left, "float_add.left")
	Float_Handle_Invariants(right, "float_add.right")
	Float_Addition_Workspace_Handle_Invariants(workspace, "float_add.workspace")
	precision := float_add_precision(
		destination.Precision, left.Precision, right.Precision,
	)
	if left.Form == FLOAT_FORM_FINITE {
		if right.Form == FLOAT_FORM_FINITE {
			if float_add_exact(destination, Float_Finite(*left), Float_Finite(*right),
				Float_Active_Precision(precision)) {
				return STATUS_OK
			}
			sum, exponent, exact := float_add_exact_words(
				Float_Active_Word_Count(left.Mantissa.Count),
				left.Mantissa.Words[WORD_COUNT_MINIMUM], left.Exponent,
				left.Negative,
				Float_Active_Word_Count(right.Mantissa.Count),
				right.Mantissa.Words[WORD_COUNT_MINIMUM], right.Exponent,
				right.Negative, Float_Active_Precision(precision),
			)
			if exact {
				previous_count := destination.Mantissa.Count
				destination.Precision = precision
				destination.Accuracy = ACCURACY_EXACT
				destination.Form = FLOAT_FORM_FINITE
				destination.Negative = left.Negative
				destination.Mantissa.Words[WORD_COUNT_MINIMUM] = Word(sum)
				destination.Mantissa.Count = Word_Count(WORD_COUNT_INCREMENT)
				destination.Mantissa.Negative = POLARITY_NONNEGATIVE
				destination.Exponent = Float_Exponent(exponent)
				if previous_count > Word_Count(WORD_COUNT_INCREMENT) {
					int_clear(
						(*Int)(&destination.Mantissa),
						Word_Count(WORD_COUNT_INCREMENT), previous_count,
					)
				}
				return STATUS_OK
			}
		}
	}
	return float_add_with_right_polarity(
		destination, left, right, right.Negative, workspace,
	)
}

func float_add_exact(
	destination Float_Handle, left Float_Finite, right Float_Finite,
	precision Float_Active_Precision,
) (matched Boolean) {
	defer func() { Boolean_Invariants(matched, "float_add_exact.matched") }()
	Float_Handle_Invariants(destination, "float_add_exact.destination_initial")
	Float_Finite_Invariants(left, "float_add_exact.left")
	Float_Finite_Invariants(right, "float_add_exact.right")
	Float_Active_Precision_Invariants(precision, "float_add_exact.precision")
	if left.Mantissa.Count != Word_Count(BASE_BINARY) {
		return false
	}
	if right.Mantissa.Count != Word_Count(BASE_BINARY) {
		return false
	}
	if left.Negative != right.Negative {
		return false
	}
	left_words := Double_Words{Low: left.Mantissa.Words[WORD_COUNT_MINIMUM],
		High: Float_Active_Word(left.Mantissa.Words[WORD_COUNT_INCREMENT])}
	right_words := Double_Words{Low: right.Mantissa.Words[WORD_COUNT_MINIMUM],
		High: Float_Active_Word(right.Mantissa.Words[WORD_COUNT_INCREMENT])}
	return float_add_exact_double_word(destination, left_words, right_words,
		left.Exponent, right.Exponent, precision, left.Negative,
	)
}

func float_add_exact_double_word(
	destination Float_Handle, left Double_Words, right Double_Words,
	left_exponent Float_Exponent, right_exponent Float_Exponent,
	precision Float_Active_Precision, negative Polarity,
) (matched Boolean) {
	defer func() {
		Boolean_Invariants(matched, "float_add_exact_double_word.matched")
	}()
	Float_Handle_Invariants(destination, "float_add_exact_double_word.destination_initial")
	Double_Words_Invariants(left, "float_add_exact_double_word.left")
	Double_Words_Invariants(right, "float_add_exact_double_word.right")
	Float_Exponent_Invariants(left_exponent, "float_add_exact_double_word.left_exponent")
	Float_Exponent_Invariants(right_exponent, "float_add_exact_double_word.right_exponent")
	Float_Active_Precision_Invariants(precision, "float_add_exact_double_word.precision")
	Polarity_Invariants(negative, "float_add_exact_double_word.negative")
	if !float_exact_double_word_inputs(
		left_exponent, right_exponent, left.High, right.High,
	) {
		return false
	}
	left_low := left.Low
	right_low := right.Low
	low := left_low + right_low
	carry := Word(0)
	if low < left_low {
		carry = Word(bits.CARRY_MAXIMUM)
	}
	left_high := Word(left.High)
	right_high := Word(right.High)
	high := left_high + right_high + carry
	carry = ((left_high & right_high) | ((left_high | right_high) &^ high)) >>
		WORD_BIT_INDEX_MAXIMUM
	count := Word_Count(BASE_BINARY)
	exponent := WORD_BIT_COUNT + int(bits.Bit_Size_64(bits.Word_64(high)))
	if carry != 0 {
		count++
		exponent = BASE_BINARY*WORD_BIT_COUNT + WORD_COUNT_INCREMENT
	}
	if int(precision) < exponent {
		return false
	}
	mantissa := &destination.Mantissa
	previous_count := mantissa.Count
	mantissa.Words[WORD_COUNT_MINIMUM] = low
	mantissa.Words[WORD_COUNT_INCREMENT] = high
	if carry != 0 {
		mantissa.Words[BASE_BINARY] = carry
	}
	mantissa.Count, mantissa.Negative = count, POLARITY_NONNEGATIVE
	destination.Precision, destination.Accuracy = Float_Precision(precision), ACCURACY_EXACT
	destination.Form, destination.Negative = FLOAT_FORM_FINITE, negative
	destination.Exponent = Float_Exponent(exponent)
	if previous_count > count {
		int_clear((*Int)(mantissa), count, previous_count)
	}
	return true
}

func float_subtract_exact_double_word(
	destination Float_Handle, left Double_Words, right Double_Words,
	left_exponent Float_Exponent, right_exponent Float_Exponent,
	precision Float_Active_Precision,
) (matched Boolean) {
	defer func() {
		Boolean_Invariants(matched, "float_subtract_exact_double_word.matched")
	}()
	Float_Handle_Invariants(destination, "float_subtract_exact_double_word.destination_initial")
	Double_Words_Invariants(left, "float_subtract_exact_double_word.left")
	Double_Words_Invariants(right, "float_subtract_exact_double_word.right")
	Float_Exponent_Invariants(left_exponent, "float_subtract_exact_double_word.left_exponent")
	Float_Exponent_Invariants(right_exponent, "float_subtract_exact_double_word.right_exponent")
	Float_Active_Precision_Invariants(precision, "float_subtract_exact_double_word.precision")
	if !float_exact_double_word_inputs(
		left_exponent, right_exponent, left.High, right.High,
	) {
		return false
	}
	maximum_exponent := max(left_exponent, right_exponent)
	if precision < Float_Active_Precision(maximum_exponent) {
		return false
	}
	left_high := Word(left.High)
	right_high := Word(right.High)
	left_low := left.Low
	right_low := right.Low
	if left_high < right_high {
		return false
	}
	if left_high == right_high {
		if left_low <= right_low {
			return false
		}
	}
	low := left_low - right_low
	borrow := Word(0)
	if low > left_low {
		borrow = Word(bits.CARRY_MAXIMUM)
	}
	high := left_high - right_high - borrow
	mantissa := &destination.Mantissa
	previous_count := mantissa.Count
	mantissa.Words[WORD_COUNT_MINIMUM] = low
	mantissa.Words[WORD_COUNT_INCREMENT] = high
	mantissa.Count = Word_Count(WORD_COUNT_INCREMENT + WORD_COUNT_INCREMENT)
	if high == 0 {
		mantissa.Count = Word_Count(WORD_COUNT_INCREMENT)
	}
	mantissa.Negative = POLARITY_NONNEGATIVE
	destination.Precision, destination.Accuracy = Float_Precision(precision), ACCURACY_EXACT
	destination.Form, destination.Negative = FLOAT_FORM_FINITE, POLARITY_NONNEGATIVE
	result_high := mantissa.Words[mantissa.Count-Word_Count(WORD_COUNT_INCREMENT)]
	destination.Exponent = Float_Exponent(
		(int(mantissa.Count)-WORD_COUNT_INCREMENT)*WORD_BIT_COUNT +
			int(bits.Bit_Size_64(bits.Word_64(result_high))),
	)
	if previous_count > mantissa.Count {
		int_clear((*Int)(mantissa), mantissa.Count, previous_count)
	}
	return true
}

func float_exact_double_word_inputs(
	left_exponent Float_Exponent, right_exponent Float_Exponent,
	left_high Float_Active_Word, right_high Float_Active_Word,
) (matched Boolean) {
	defer func() {
		Boolean_Invariants(matched, "float_exact_double_word_inputs.matched")
	}()
	Float_Exponent_Invariants(left_exponent, "float_exact_double_word_inputs.left_exponent")
	Float_Exponent_Invariants(right_exponent, "float_exact_double_word_inputs.right_exponent")
	Float_Active_Word_Invariants(left_high, "float_exact_double_word_inputs.left_high")
	Float_Active_Word_Invariants(right_high, "float_exact_double_word_inputs.right_high")
	left_bit_index := int(left_exponent) - WORD_BIT_COUNT - WORD_COUNT_INCREMENT
	right_bit_index := int(right_exponent) - WORD_BIT_COUNT - WORD_COUNT_INCREMENT
	if uint(left_bit_index) >= uint(WORD_BIT_COUNT) {
		return false
	}
	if uint(right_bit_index) >= uint(WORD_BIT_COUNT) {
		return false
	}
	left_minimum := Float_Active_Word(bits.CARRY_MAXIMUM) << uint(left_bit_index)
	right_minimum := Float_Active_Word(bits.CARRY_MAXIMUM) << uint(right_bit_index)
	if left_high&left_minimum == 0 {
		return false
	}
	if right_high&right_minimum == 0 {
		return false
	}
	if left_bit_index < WORD_BIT_INDEX_MAXIMUM {
		if left_high >= left_minimum<<WORD_COUNT_INCREMENT {
			return false
		}
	}
	if right_bit_index < WORD_BIT_INDEX_MAXIMUM {
		if right_high >= right_minimum<<WORD_COUNT_INCREMENT {
			return false
		}
	}
	return true
}

func float_multiply_exact_double_word(
	reference Float_Active_Handle, left Double_Words, right Double_Words,
	left_exponent Float_Exponent, right_exponent Float_Exponent, negative Polarity,
	workspace Float_Multiplication_Workspace_Handle,
) (matched Boolean) {
	defer func() {
		Boolean_Invariants(matched, "float_multiply_exact_double_word.matched")
	}()
	Float_Active_Handle_Invariants(
		reference, "float_multiply_exact_double_word.destination_initial")
	Double_Words_Invariants(left, "float_multiply_exact_double_word.left")
	Double_Words_Invariants(right, "float_multiply_exact_double_word.right")
	Float_Exponent_Invariants(left_exponent, "float_multiply_exact_double_word.left_exponent")
	Float_Exponent_Invariants(right_exponent, "float_multiply_exact_double_word.right_exponent")
	Polarity_Invariants(negative, "float_multiply_exact_double_word.negative")
	Float_Multiplication_Workspace_Handle_Invariants(
		workspace, "float_multiply_exact_double_word.workspace",
	)
	destination := (*Float)((*Float_Active)(reference))
	if !float_exact_double_word_inputs(
		left_exponent, right_exponent, left.High, right.High,
	) {
		return false
	}
	float_multiply_double_words(left, right, workspace)
	bit_count := int(left_exponent) + int(right_exponent) - WORD_COUNT_INCREMENT
	upper_bit_index := bit_count
	upper_word := workspace.Result[upper_bit_index/WORD_BIT_COUNT]
	if upper_word>>uint(upper_bit_index%WORD_BIT_COUNT)&1 != 0 {
		bit_count++
	}
	precision := Float_Active_Precision(destination.Precision)
	if bit_count > int(precision) {
		if int(precision)%WORD_BIT_COUNT == BIT_COUNT_MINIMUM {
			float_set_aligned_double_word_product(
				(*Float_Three_Word_Aligned)((*Float)(destination)), workspace,
				Double_Word_Product_Bit_Count(bit_count), negative,
			)
			return true
		}
	}
	addition_workspace := (*Float_Addition_Workspace)(
		(*Float_Multiplication_Workspace)(workspace))
	float_set_addition_mantissa(
		(*Float_Nonempty)((*Float)(destination)), addition_workspace,
		Float_Addition_Bit_Count(bit_count), Float_Exponent(bit_count),
		negative, precision, destination.Mode,
	)
	return true
}

func float_multiply_double_words(
	left Double_Words, right Double_Words,
	workspace Float_Multiplication_Workspace_Handle,
) {
	Double_Words_Invariants(left, "float_multiply_double_words.left")
	Double_Words_Invariants(right, "float_multiply_double_words.right")
	Float_Multiplication_Workspace_Handle_Invariants(
		workspace, "float_multiply_double_words.workspace",
	)
	result_count := BASE_BINARY * BASE_BINARY
	for index := WORD_COUNT_MINIMUM; index < result_count; index++ {
		workspace.Result[index] = 0
	}
	word_mask := uint64(bits.WORD_32_MAXIMUM)
	for right_offset := WORD_COUNT_MINIMUM; right_offset < BASE_BINARY; right_offset++ {
		right_word := uint64(right.Low)
		if right_offset != WORD_COUNT_MINIMUM {
			right_word = uint64(right.High)
		}
		carry := Word(0)
		for left_offset := WORD_COUNT_MINIMUM; left_offset < BASE_BINARY; left_offset++ {
			position := right_offset
			position += left_offset
			left_word := uint64(left.Low)
			if left_offset != WORD_COUNT_MINIMUM {
				left_word = uint64(left.High)
			}
			partial := (left_word & word_mask) * (right_word & word_mask)
			middle_first := (left_word>>bits.BIT_COUNT_32_MAXIMUM)*
				(right_word&word_mask) + partial>>bits.BIT_COUNT_32_MAXIMUM
			middle_second := (left_word & word_mask) *
				(right_word >> bits.BIT_COUNT_32_MAXIMUM)
			middle_second += middle_first & word_mask
			high := Word(
				(left_word>>bits.BIT_COUNT_32_MAXIMUM)*
					(right_word>>bits.BIT_COUNT_32_MAXIMUM) +
					middle_first>>bits.BIT_COUNT_32_MAXIMUM +
					middle_second>>bits.BIT_COUNT_32_MAXIMUM,
			)
			low := Word(left_word * right_word)
			low_sum := low + workspace.Result[position]
			stored_carry := Word(0)
			if low_sum < low {
				stored_carry = Word(bits.CARRY_MAXIMUM)
			}
			word := low_sum + carry
			carry_output := Word(0)
			if word < low_sum {
				carry_output = Word(bits.CARRY_MAXIMUM)
			}
			workspace.Result[position] = word
			carry = high + stored_carry + carry_output
		}
		if carry != 0 {
			position := right_offset
			position += BASE_BINARY
			workspace.Result[position] = carry
		}
	}
}

// DOUBLE_WORD_PRODUCT_BIT_COUNT_MINIMUM follows from two normalized two-word factors.
const DOUBLE_WORD_PRODUCT_BIT_COUNT_MINIMUM = BASE_BINARY*WORD_BIT_COUNT + WORD_COUNT_INCREMENT

// DOUBLE_WORD_PRODUCT_BIT_COUNT_MAXIMUM reserves both complete factors in their product.
const DOUBLE_WORD_PRODUCT_BIT_COUNT_MAXIMUM = BASE_BINARY * BASE_BINARY * WORD_BIT_COUNT

// Double_Word_Product_Bit_Count excludes exponents unrelated to exact integer products.
type Double_Word_Product_Bit_Count int

// Double_Word_Product_Bit_Count_Invariants binds the product of two normalized two-word integers.
func Double_Word_Product_Bit_Count_Invariants(
	value Double_Word_Product_Bit_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), DOUBLE_WORD_PRODUCT_BIT_COUNT_MINIMUM,
			DOUBLE_WORD_PRODUCT_BIT_COUNT_MAXIMUM).
		Ensure()
}

func float_set_aligned_double_word_product(
	destination_reference Float_Three_Word_Aligned_Handle,
	workspace Float_Multiplication_Workspace_Handle,
	exponent Double_Word_Product_Bit_Count, negative Polarity,
) {
	Float_Three_Word_Aligned_Handle_Invariants(
		destination_reference, "float_set_aligned_double_word_product.destination_initial",
	)
	Float_Multiplication_Workspace_Handle_Invariants(
		workspace, "float_set_aligned_double_word_product.workspace",
	)
	Double_Word_Product_Bit_Count_Invariants(
		exponent, "float_set_aligned_double_word_product.exponent")
	Polarity_Invariants(negative, "float_set_aligned_double_word_product.negative")
	destination := (*Float)((*Float_Three_Word_Aligned)(destination_reference))
	// Validate destination before staging the product's new exponent and sign.
	destination.Exponent, destination.Negative = Float_Exponent(exponent), negative
	precision := Float_Active_Precision(destination.Precision)
	discard_count := int(destination.Exponent) - int(precision)
	mantissa_count := int(precision) / WORD_BIT_COUNT
	previous_count := destination.Mantissa.Count
	for index := WORD_COUNT_MINIMUM; index < mantissa_count; index++ {
		source_bit := discard_count + index*WORD_BIT_COUNT
		source_index := source_bit / WORD_BIT_COUNT
		source_shift := uint(source_bit % WORD_BIT_COUNT)
		word := workspace.Result[source_index] >> source_shift
		if source_shift != 0 {
			word |= workspace.Result[source_index+WORD_COUNT_INCREMENT] <<
				(WORD_BIT_COUNT - source_shift)
		}
		destination.Mantissa.Words[index] = word
	}
	rounding_index := discard_count - WORD_COUNT_INCREMENT
	rounding_word := rounding_index / WORD_BIT_COUNT
	rounding_shift := uint(rounding_index % WORD_BIT_COUNT)
	rounding_bit := Bit_Value(workspace.Result[rounding_word] >> rounding_shift & 1)
	sticky_bit := BIT_CLEAR
	for index := WORD_COUNT_MINIMUM; index < rounding_word; index++ {
		if workspace.Result[index] != 0 {
			sticky_bit = BIT_SET
		}
	}
	if rounding_shift != 0 {
		mask := Word(bits.WORD_64_MAXIMUM) >> (WORD_BIT_COUNT - rounding_shift)
		if workspace.Result[rounding_word]&mask != 0 {
			sticky_bit = BIT_SET
		}
	}
	mantissa := (*Float_Active_Mantissa)(&destination.Mantissa)
	mantissa.Count, mantissa.Negative = Word_Count(mantissa_count), POLARITY_NONNEGATIVE
	increment := float_rounding_increment(
		destination.Mode, destination.Negative, rounding_bit, sticky_bit,
		Bit_Value(mantissa.Words[WORD_COUNT_MINIMUM]&1),
	)
	accuracy := ACCURACY_EXACT
	if rounding_bit|sticky_bit != BIT_CLEAR {
		accuracy = ACCURACY_ABOVE
		if increment == Boolean(destination.Negative == POLARITY_NEGATIVE) {
			accuracy = ACCURACY_BELOW
		}
	}
	if increment {
		float_increment_mantissa(
			(*Positive_Int)(mantissa), precision, &destination.Exponent)
	}
	destination.Accuracy, destination.Form = accuracy, FLOAT_FORM_FINITE
	if previous_count > mantissa.Count {
		int_clear((*Int)(&destination.Mantissa), mantissa.Count, previous_count)
	}
}

func float_subtract_general(
	destination, left, right Float_Handle, workspace Float_Addition_Workspace_Handle,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "float_subtract_general.status") }()
	Float_Handle_Invariants(destination, "float_subtract_general.destination_initial")
	Float_Handle_Invariants(left, "float_subtract_general.left")
	Float_Handle_Invariants(right, "float_subtract_general.right")
	Float_Addition_Workspace_Handle_Invariants(workspace, "float_subtract_general.workspace")
	return float_add_with_right_polarity(
		destination, left, right, POLARITY_NEGATIVE-right.Negative, workspace,
	)
}

func float_add_with_right_polarity(
	destination, left, right Float_Handle, right_negative Polarity,
	workspace Float_Addition_Workspace_Handle,
) (status Validation_Status) {
	defer func() {
		Validation_Status_Invariants(status, "float_add_with_right_polarity.status")
	}()
	Float_Handle_Invariants(destination, "float_add_with_right_polarity.destination_initial")
	Float_Handle_Invariants(left, "float_add_with_right_polarity.left")
	Float_Handle_Invariants(right, "float_add_with_right_polarity.right")
	Polarity_Invariants(right_negative, "float_add_with_right_polarity.right_negative")
	Float_Addition_Workspace_Handle_Invariants(
		workspace, "float_add_with_right_polarity.workspace",
	)
	precision := float_add_precision(
		destination.Precision, left.Precision, right.Precision,
	)
	mode := destination.Mode
	if left.Form == FLOAT_FORM_FINITE {
		if right.Form == FLOAT_FORM_FINITE {
			float_add_finite_polarity(
				destination, (*Float_Finite)((*Float)(left)),
				(*Float_Finite)((*Float)(right)),
				workspace,
				Float_Active_Precision(precision), mode, right_negative,
			)
			return STATUS_OK
		}
	}
	return float_add_nonfinite(
		destination, left, right, right_negative, precision, mode,
	)
}

func float_add_nonfinite(
	destination, left, right Float_Handle, right_negative Polarity,
	precision Float_Precision, mode Rounding_Mode,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "float_add_nonfinite.status") }()
	Float_Handle_Invariants(destination, "float_add_nonfinite.destination_initial")
	Float_Handle_Invariants(left, "float_add_nonfinite.left")
	Float_Handle_Invariants(right, "float_add_nonfinite.right")
	Polarity_Invariants(right_negative, "float_add_nonfinite.right_negative")
	Float_Precision_Invariants(precision, "float_add_nonfinite.precision")
	Rounding_Mode_Invariants(mode, "float_add_nonfinite.mode")
	if left.Form == FLOAT_FORM_ZERO {
		if right.Form == FLOAT_FORM_ZERO {
			negative := left.Negative
			if negative != right_negative {
				negative = float_zero_polarity(mode)
			}
			float_set_nonfinite(
				destination, Float_Nonfinite_Form(FLOAT_FORM_ZERO), negative,
				precision,
			)
			return STATUS_OK
		}
		if right.Form == FLOAT_FORM_INFINITY {
			float_set_nonfinite(
				destination, Float_Nonfinite_Form(FLOAT_FORM_INFINITY),
				right_negative, precision,
			)
			return STATUS_OK
		}
		float_set_magnitude(
			destination, &right.Mantissa, right.Exponent, right_negative,
			Float_Active_Precision(precision), mode,
		)
		return STATUS_OK
	}
	if right.Form == FLOAT_FORM_ZERO {
		float_set_operation_operand(destination, left, precision, mode)
		return STATUS_OK
	}
	if left.Form == FLOAT_FORM_INFINITY {
		if right.Form == FLOAT_FORM_INFINITY {
			if left.Negative != right_negative {
				return STATUS_INPUT_INVALID
			}
		}
		float_set_nonfinite(
			destination, Float_Nonfinite_Form(FLOAT_FORM_INFINITY), left.Negative,
			precision,
		)
		return STATUS_OK
	}
	float_set_nonfinite(
		destination, Float_Nonfinite_Form(FLOAT_FORM_INFINITY), right_negative,
		precision,
	)
	return STATUS_OK
}

func float_add_finite_polarity(
	destination Float_Handle, left Float_Finite_Handle, right Float_Finite_Handle,
	workspace Float_Addition_Workspace_Handle, precision Float_Active_Precision,
	mode Rounding_Mode, right_negative Polarity,
) {
	Float_Handle_Invariants(destination, "float_add_finite_polarity.destination_initial")
	Float_Finite_Handle_Invariants(left, "float_add_finite_polarity.left")
	Float_Finite_Handle_Invariants(right, "float_add_finite_polarity.right")
	Float_Addition_Workspace_Handle_Invariants(
		workspace, "float_add_finite_polarity.workspace",
	)
	Float_Active_Precision_Invariants(precision, "float_add_finite_polarity.precision")
	Rounding_Mode_Invariants(mode, "float_add_finite_polarity.mode")
	Polarity_Invariants(right_negative, "float_add_finite_polarity.right_negative")
	sum_word, result_exponent, exact := float_add_exact_words(
		Float_Active_Word_Count(left.Mantissa.Count),
		left.Mantissa.Words[WORD_COUNT_MINIMUM], left.Exponent, left.Negative,
		Float_Active_Word_Count(right.Mantissa.Count),
		right.Mantissa.Words[WORD_COUNT_MINIMUM], right.Exponent, right_negative,
		Float_Active_Precision(precision),
	)
	if exact {
		previous_count := destination.Mantissa.Count
		destination.Precision = Float_Precision(precision)
		destination.Accuracy = ACCURACY_EXACT
		destination.Form = FLOAT_FORM_FINITE
		destination.Negative = left.Negative
		destination.Mantissa.Negative, destination.Mantissa.Count =
			POLARITY_NONNEGATIVE, Word_Count(WORD_COUNT_INCREMENT)
		destination.Mantissa.Words[WORD_COUNT_MINIMUM] = Word(sum_word)
		destination.Exponent = Float_Exponent(result_exponent)
		if previous_count > destination.Mantissa.Count {
			int_clear(
				(*Int)(&destination.Mantissa), destination.Mantissa.Count,
				previous_count,
			)
		}
		return
	}
	float_add_finite(
		destination, left, right, workspace, precision, mode, right_negative,
	)
}

func float_add_exact_words(
	left_count Float_Active_Word_Count, left_word Word,
	left_exponent Float_Exponent, left_negative Polarity,
	right_count Float_Active_Word_Count, right_word Word,
	right_exponent Float_Exponent, right_negative Polarity,
	precision Float_Active_Precision,
) (sum Float_Add_Exact_Sum, exponent Float_Add_Exact_Exponent, matched Boolean) {
	defer func() {
		Float_Add_Exact_Sum_Invariants(sum, "float_add_exact_words.sum")
		Float_Add_Exact_Exponent_Invariants(exponent, "float_add_exact_words.exponent")
		Boolean_Invariants(matched, "float_add_exact_words.matched")
	}()
	Float_Active_Word_Count_Invariants(left_count, "float_add_exact_words.left_count")
	Word_Invariants(left_word, "float_add_exact_words.left_word")
	Float_Exponent_Invariants(left_exponent, "float_add_exact_words.left_exponent")
	Polarity_Invariants(left_negative, "float_add_exact_words.left_negative")
	Float_Active_Word_Count_Invariants(right_count, "float_add_exact_words.right_count")
	Word_Invariants(right_word, "float_add_exact_words.right_word")
	Float_Exponent_Invariants(right_exponent, "float_add_exact_words.right_exponent")
	Polarity_Invariants(right_negative, "float_add_exact_words.right_negative")
	Float_Active_Precision_Invariants(precision, "float_add_exact_words.precision")
	if left_negative != right_negative {
		return 0, 0, false
	}
	if left_count != Float_Active_Word_Count(WORD_COUNT_INCREMENT) {
		return 0, 0, false
	}
	if right_count != Float_Active_Word_Count(WORD_COUNT_INCREMENT) {
		return 0, 0, false
	}
	if !float_exact_word(Float_Active_Word(left_word), left_exponent) {
		return 0, 0, false
	}
	if !float_exact_word(Float_Active_Word(right_word), right_exponent) {
		return 0, 0, false
	}
	word_sum := left_word + right_word
	if word_sum < left_word {
		return 0, 0, false
	}
	result_exponent := left_exponent
	if right_exponent > result_exponent {
		result_exponent = right_exponent
	}
	if result_exponent < Float_Exponent(WORD_BIT_COUNT) {
		if word_sum >= Word(bits.CARRY_MAXIMUM)<<uint(result_exponent) {
			result_exponent++
		}
	}
	if result_exponent > Float_Exponent(precision) {
		return 0, 0, false
	}
	return Float_Add_Exact_Sum(word_sum), Float_Add_Exact_Exponent(result_exponent), true
}

func float_exact_word(word Float_Active_Word, exponent Float_Exponent) (exact Boolean) {
	defer func() { Boolean_Invariants(exact, "float_exact_word.exact") }()
	Float_Active_Word_Invariants(word, "float_exact_word.word")
	Float_Exponent_Invariants(exponent, "float_exact_word.exponent")
	if exponent < Float_Exponent(WORD_COUNT_INCREMENT) {
		return false
	}
	if exponent > Float_Exponent(WORD_BIT_COUNT) {
		return false
	}
	minimum := Float_Active_Word(bits.CARRY_MAXIMUM) << uint(
		int(exponent)-WORD_COUNT_INCREMENT,
	)
	if word < minimum {
		return false
	}
	if exponent == Float_Exponent(WORD_BIT_COUNT) {
		return true
	}
	return Boolean(word < minimum<<WORD_COUNT_INCREMENT)
}

// FLOAT_RAT_NUMERATOR_INDEX stores the signed rational component as one float operand.
const FLOAT_RAT_NUMERATOR_INDEX = RAT_NUMERATOR_INDEX

// FLOAT_RAT_DENOMINATOR_INDEX stores the positive rational component as one float operand.
const FLOAT_RAT_DENOMINATOR_INDEX = RAT_DENOMINATOR_INDEX

// FLOAT_RAT_RESULT_INDEX keeps the configured destination outside both operands.
const FLOAT_RAT_RESULT_INDEX = RAT_COMPONENT_COUNT

// FLOAT_RAT_VALUE_COUNT includes both operands and one alias-safe result.
const FLOAT_RAT_VALUE_COUNT = FLOAT_RAT_RESULT_INDEX + WORD_COUNT_INCREMENT

// FLOAT_SQUARE_ROOT_WORD_COUNT_MAXIMUM retains one guard bit beyond Float precision.
const FLOAT_SQUARE_ROOT_WORD_COUNT_MAXIMUM = WORD_COUNT_MAXIMUM + WORD_COUNT_INCREMENT

// FLOAT_SQUARE_ROOT_GUARD_BIT_COUNT extracts one bit for rounding direction.
const FLOAT_SQUARE_ROOT_GUARD_BIT_COUNT = WORD_COUNT_INCREMENT

// FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT consumes one radix-four digit per root bit.
const FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT = BASE_BINARY

// FLOAT_SQUARE_ROOT_PAIR_COUNT follows every pattern in one radix-four digit.
const FLOAT_SQUARE_ROOT_PAIR_COUNT = WORD_COUNT_INCREMENT << FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT

// FLOAT_SQUARE_ROOT_PAIR_MAXIMUM is one complete radix-four source digit.
const FLOAT_SQUARE_ROOT_PAIR_MAXIMUM = FLOAT_SQUARE_ROOT_PAIR_COUNT - WORD_COUNT_INCREMENT

// FLOAT_SQUARE_ROOT_EXPONENT_MINIMUM halves the least finite source exponent.
const FLOAT_SQUARE_ROOT_EXPONENT_MINIMUM = FLOAT_EXPONENT_MINIMUM / BASE_BINARY

// FLOAT_SQUARE_ROOT_EXPONENT_MAXIMUM halves the greatest finite source exponent.
const FLOAT_SQUARE_ROOT_EXPONENT_MAXIMUM = FLOAT_EXPONENT_MAXIMUM / BASE_BINARY

// FLOAT_SQUARE_ROOT_SOURCE_BIT_INDEX_MAXIMUM remains after the shortest odd-root prefix.
const FLOAT_SQUARE_ROOT_SOURCE_BIT_INDEX_MAXIMUM = BIT_INDEX_MAXIMUM -
	FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT - WORD_COUNT_INCREMENT

// FLOAT_32_SIGN_BIT_COUNT reserves IEEE sign outside exponent and mantissa.
const FLOAT_32_SIGN_BIT_COUNT = WORD_COUNT_INCREMENT

// FLOAT_32_EXPONENT_BIT_COUNT follows binary32 exponent storage width.
const FLOAT_32_EXPONENT_BIT_COUNT = bits.BIT_COUNT_8_MAXIMUM

// FLOAT_32_MANTISSA_BIT_COUNT consumes every remaining binary32 bit.
const FLOAT_32_MANTISSA_BIT_COUNT = bits.BIT_COUNT_32_MAXIMUM -
	FLOAT_32_SIGN_BIT_COUNT - FLOAT_32_EXPONENT_BIT_COUNT

// FLOAT_32_EXPONENT_SHIFT places exponent above stored mantissa.
const FLOAT_32_EXPONENT_SHIFT = FLOAT_32_MANTISSA_BIT_COUNT

// FLOAT_32_SIGN_SHIFT leaves sign in highest binary32 position.
const FLOAT_32_SIGN_SHIFT = bits.BIT_COUNT_32_MAXIMUM - FLOAT_32_SIGN_BIT_COUNT

// FLOAT_32_EXPONENT_COUNT derives complete encoded exponent domain.
const FLOAT_32_EXPONENT_COUNT = WORD_COUNT_INCREMENT << FLOAT_32_EXPONENT_BIT_COUNT

// FLOAT_32_EXPONENT_MASK selects every encoded exponent bit.
const FLOAT_32_EXPONENT_MASK = FLOAT_32_EXPONENT_COUNT - WORD_COUNT_INCREMENT

// FLOAT_32_FINITE_EXPONENT_FIELD_MAXIMUM excludes reserved nonfinite field.
const FLOAT_32_FINITE_EXPONENT_FIELD_MAXIMUM = FLOAT_32_EXPONENT_MASK -
	WORD_COUNT_INCREMENT

// FLOAT_32_EXPONENT_BIAS centers normal binary32 exponent range.
const FLOAT_32_EXPONENT_BIAS = FLOAT_32_EXPONENT_COUNT/BASE_BINARY -
	WORD_COUNT_INCREMENT

// FLOAT_32_HIDDEN_MANTISSA_BIT restores omitted normal leading bit.
const FLOAT_32_HIDDEN_MANTISSA_BIT = uint32(bits.CARRY_MAXIMUM) <<
	FLOAT_32_MANTISSA_BIT_COUNT

// FLOAT_32_MANTISSA_MASK selects stored significand without hidden bit.
const FLOAT_32_MANTISSA_MASK = FLOAT_32_HIDDEN_MANTISSA_BIT -
	uint32(bits.CARRY_MAXIMUM)

// FLOAT_32_SIGN_MASK selects encoded binary32 sign.
const FLOAT_32_SIGN_MASK = uint32(bits.CARRY_MAXIMUM) << FLOAT_32_SIGN_SHIFT

// FLOAT_32_SUBNORMAL_EXPONENT keeps subnormal scale below normal range.
const FLOAT_32_SUBNORMAL_EXPONENT = WORD_COUNT_INCREMENT - FLOAT_32_EXPONENT_BIAS

// FLOAT_32_SUBNORMAL_DENOMINATOR_SHIFT_MAXIMUM reaches smallest subnormal.
const FLOAT_32_SUBNORMAL_DENOMINATOR_SHIFT_MAXIMUM = FLOAT_32_MANTISSA_BIT_COUNT -
	FLOAT_32_SUBNORMAL_EXPONENT

// FLOAT_32_VALUE_BIT_COUNT_MAXIMUM bounds finite binary32 magnitude.
const FLOAT_32_VALUE_BIT_COUNT_MAXIMUM = FLOAT_32_EXPONENT_BIAS + WORD_COUNT_INCREMENT

// FLOAT_32_VALUE_MANTISSA_BIT_COUNT restores normal hidden bit.
const FLOAT_32_VALUE_MANTISSA_BIT_COUNT = FLOAT_32_MANTISSA_BIT_COUNT +
	WORD_COUNT_INCREMENT

// FLOAT_32_ROUNDING_MANTISSA_BIT_COUNT retains one low rounding bit.
const FLOAT_32_ROUNDING_MANTISSA_BIT_COUNT = FLOAT_32_VALUE_MANTISSA_BIT_COUNT +
	WORD_COUNT_INCREMENT

// FLOAT_32_ROUNDING_MANTISSA_LIMIT is the carry beyond rounding quotient.
const FLOAT_32_ROUNDING_MANTISSA_LIMIT = uint32(bits.CARRY_MAXIMUM) <<
	FLOAT_32_ROUNDING_MANTISSA_BIT_COUNT

// FLOAT_32_ROUNDING_MANTISSA_MAXIMUM is the greatest guarded significand.
const FLOAT_32_ROUNDING_MANTISSA_MAXIMUM = FLOAT_32_ROUNDING_MANTISSA_LIMIT -
	uint32(bits.CARRY_MAXIMUM)

// FLOAT_32_ROUNDING_MANTISSA_MINIMUM is the half-subnormal tie.
const FLOAT_32_ROUNDING_MANTISSA_MINIMUM = uint32(bits.CARRY_MAXIMUM)

// FLOAT_32_VALUE_MANTISSA_LIMIT is the carry beyond final significand.
const FLOAT_32_VALUE_MANTISSA_LIMIT = uint32(bits.CARRY_MAXIMUM) <<
	FLOAT_32_VALUE_MANTISSA_BIT_COUNT

// FLOAT_32_VALUE_MANTISSA_MAXIMUM is the greatest rounded significand.
const FLOAT_32_VALUE_MANTISSA_MAXIMUM = FLOAT_32_VALUE_MANTISSA_LIMIT -
	uint32(bits.CARRY_MAXIMUM)

// FLOAT_32_SUBNORMAL_EXPONENT_MINIMUM fixes smallest binary32 power.
const FLOAT_32_SUBNORMAL_EXPONENT_MINIMUM = FLOAT_32_SUBNORMAL_EXPONENT -
	FLOAT_32_MANTISSA_BIT_COUNT

// FLOAT_32_EXPONENT_ENCODING_OFFSET maps normalized exponent to encoded field.
const FLOAT_32_EXPONENT_ENCODING_OFFSET = FLOAT_32_EXPONENT_BIAS -
	WORD_COUNT_INCREMENT

// FLOAT_32_POSITIVE_INFINITY_BITS reserves nonfinite exponent with clear mantissa.
const FLOAT_32_POSITIVE_INFINITY_BITS = uint32(FLOAT_32_EXPONENT_MASK) <<
	FLOAT_32_EXPONENT_SHIFT

// FLOAT_32_NEGATIVE_INFINITY_BITS adds sign without changing nonfinite identity.
const FLOAT_32_NEGATIVE_INFINITY_BITS = FLOAT_32_SIGN_MASK |
	FLOAT_32_POSITIVE_INFINITY_BITS

// FLOAT_32_VALUE_BITS_MINIMUM keeps positive zero as first numeric encoding.
const FLOAT_32_VALUE_BITS_MINIMUM = bits.WORD_32_MINIMUM

// FLOAT_32_VALUE_BITS_MAXIMUM stops before negative NaN encodings.
const FLOAT_32_VALUE_BITS_MAXIMUM = FLOAT_32_NEGATIVE_INFINITY_BITS

// RAT_FLOAT_32_EXPONENT_MINIMUM follows the smallest bounded component ratio.
const RAT_FLOAT_32_EXPONENT_MINIMUM = WORD_COUNT_INCREMENT -
	RAT_COMPONENT_TEXT_DIGIT_COUNT_MAXIMUM

// RAT_FLOAT_32_EXPONENT_MAXIMUM includes normalization and rounding carries.
const RAT_FLOAT_32_EXPONENT_MAXIMUM = RAT_COMPONENT_TEXT_DIGIT_COUNT_MAXIMUM +
	WORD_COUNT_INCREMENT

// RAT_FLOAT_32_ROUNDING_EXPONENT_MAXIMUM leaves final carry to round output.
const RAT_FLOAT_32_ROUNDING_EXPONENT_MAXIMUM = RAT_FLOAT_32_EXPONENT_MAXIMUM -
	WORD_COUNT_INCREMENT

// RAT_FLOAT_32_NUMERATOR_INDEX selects scaled nonnegative numerator.
const RAT_FLOAT_32_NUMERATOR_INDEX = WORD_COUNT_MINIMUM

// RAT_FLOAT_32_DENOMINATOR_INDEX selects scaled positive denominator.
const RAT_FLOAT_32_DENOMINATOR_INDEX = RAT_FLOAT_32_NUMERATOR_INDEX +
	WORD_COUNT_INCREMENT

// RAT_FLOAT_32_QUOTIENT_INDEX selects rounding-width quotient.
const RAT_FLOAT_32_QUOTIENT_INDEX = RAT_FLOAT_32_DENOMINATOR_INDEX +
	WORD_COUNT_INCREMENT

// RAT_FLOAT_32_REMAINDER_INDEX selects discarded division remainder.
const RAT_FLOAT_32_REMAINDER_INDEX = RAT_FLOAT_32_QUOTIENT_INDEX +
	WORD_COUNT_INCREMENT

// RAT_FLOAT_32_INTEGER_COUNT is complete binary32 conversion state.
const RAT_FLOAT_32_INTEGER_COUNT = RAT_FLOAT_32_REMAINDER_INDEX +
	WORD_COUNT_INCREMENT

// INT_FLOAT_64_VALUE_BITS_MINIMUM is positive zero.
const INT_FLOAT_64_VALUE_BITS_MINIMUM = FLOAT_64_VALUE_BITS_MINIMUM

// INT_FLOAT_64_VALUE_BITS_MAXIMUM is negative infinity.
const INT_FLOAT_64_VALUE_BITS_MAXIMUM = FLOAT_64_VALUE_BITS_MAXIMUM

// INT_FLOAT_64_SUBNORMAL_HOLE_1 removes the first impossible scalar witness.
const INT_FLOAT_64_SUBNORMAL_HOLE_1 = INT_FLOAT_64_VALUE_BITS_MINIMUM +
	WORD_COUNT_INCREMENT

// INT_FLOAT_64_SUBNORMAL_HOLE_2 removes the second impossible scalar witness.
const INT_FLOAT_64_SUBNORMAL_HOLE_2 = INT_FLOAT_64_SUBNORMAL_HOLE_1 +
	WORD_COUNT_INCREMENT

// INT_FLOAT_64_SUBNORMAL_HOLE_3 keeps canonical range witnesses production-reachable.
const INT_FLOAT_64_SUBNORMAL_HOLE_3 = INT_FLOAT_64_SUBNORMAL_HOLE_2 +
	WORD_COUNT_INCREMENT

// MODULAR_SQUARE_ROOT_MODULUS_INDEX preserves validated positive odd modulus.
const MODULAR_SQUARE_ROOT_MODULUS_INDEX = WORD_COUNT_MINIMUM

// MODULAR_SQUARE_ROOT_RESIDUE_INDEX stores nonnegative reduced source.
const MODULAR_SQUARE_ROOT_RESIDUE_INDEX = MODULAR_SQUARE_ROOT_MODULUS_INDEX +
	WORD_COUNT_INCREMENT

// MODULAR_SQUARE_ROOT_NONRESIDUE_INDEX preserves caller-supplied search result.
const MODULAR_SQUARE_ROOT_NONRESIDUE_INDEX = MODULAR_SQUARE_ROOT_RESIDUE_INDEX +
	WORD_COUNT_INCREMENT

// MODULAR_SQUARE_ROOT_ODD_FACTOR_INDEX stores odd factor of modulus minus one.
const MODULAR_SQUARE_ROOT_ODD_FACTOR_INDEX = MODULAR_SQUARE_ROOT_NONRESIDUE_INDEX +
	WORD_COUNT_INCREMENT

// MODULAR_SQUARE_ROOT_EXPONENT_INDEX stores each derived exponent.
const MODULAR_SQUARE_ROOT_EXPONENT_INDEX = MODULAR_SQUARE_ROOT_ODD_FACTOR_INDEX +
	WORD_COUNT_INCREMENT

// MODULAR_SQUARE_ROOT_RESULT_INDEX stores the transactional root candidate.
const MODULAR_SQUARE_ROOT_RESULT_INDEX = MODULAR_SQUARE_ROOT_EXPONENT_INDEX +
	WORD_COUNT_INCREMENT

// MODULAR_SQUARE_ROOT_POWER_INDEX stores source raised to odd factor.
const MODULAR_SQUARE_ROOT_POWER_INDEX = MODULAR_SQUARE_ROOT_RESULT_INDEX +
	WORD_COUNT_INCREMENT

// MODULAR_SQUARE_ROOT_GENERATOR_INDEX stores nonresidue subgroup generator.
const MODULAR_SQUARE_ROOT_GENERATOR_INDEX = MODULAR_SQUARE_ROOT_POWER_INDEX +
	WORD_COUNT_INCREMENT

// MODULAR_SQUARE_ROOT_FACTOR_INDEX stores each order-reduction factor.
const MODULAR_SQUARE_ROOT_FACTOR_INDEX = MODULAR_SQUARE_ROOT_GENERATOR_INDEX +
	WORD_COUNT_INCREMENT

// MODULAR_SQUARE_ROOT_ONE_INDEX stores multiplicative identity.
const MODULAR_SQUARE_ROOT_ONE_INDEX = MODULAR_SQUARE_ROOT_FACTOR_INDEX +
	WORD_COUNT_INCREMENT

// MODULAR_SQUARE_ROOT_CHECK_INDEX verifies a root before destination mutation.
const MODULAR_SQUARE_ROOT_CHECK_INDEX = MODULAR_SQUARE_ROOT_ONE_INDEX +
	WORD_COUNT_INCREMENT

// MODULAR_SQUARE_ROOT_INTEGER_COUNT is complete Tonelli-Shanks state.
const MODULAR_SQUARE_ROOT_INTEGER_COUNT = MODULAR_SQUARE_ROOT_CHECK_INDEX +
	WORD_COUNT_INCREMENT

// FLOAT_TEXT_FORMAT_BINARY emits decimal significand with binary exponent.
const FLOAT_TEXT_FORMAT_BINARY = 'b'

// FLOAT_TEXT_FORMAT_HEXADECIMAL_FRACTION emits normalized hexadecimal fraction.
const FLOAT_TEXT_FORMAT_HEXADECIMAL_FRACTION = 'p'

// FLOAT_TEXT_FORMAT_HEXADECIMAL emits normalized hexadecimal significand.
const FLOAT_TEXT_FORMAT_HEXADECIMAL = 'x'

// FLOAT_TEXT_FORMAT_DECIMAL_EXPONENT emits lowercase decimal scientific notation.
const FLOAT_TEXT_FORMAT_DECIMAL_EXPONENT = 'e'

// FLOAT_TEXT_FORMAT_DECIMAL_EXPONENT_UPPER emits uppercase decimal scientific notation.
const FLOAT_TEXT_FORMAT_DECIMAL_EXPONENT_UPPER = 'E'

// FLOAT_TEXT_FORMAT_DECIMAL_FIXED emits decimal notation without exponent.
const FLOAT_TEXT_FORMAT_DECIMAL_FIXED = 'f'

// FLOAT_TEXT_FORMAT_DECIMAL_GENERAL selects lowercase compact decimal notation.
const FLOAT_TEXT_FORMAT_DECIMAL_GENERAL = 'g'

// FLOAT_TEXT_FORMAT_DECIMAL_GENERAL_UPPER selects uppercase compact decimal notation.
const FLOAT_TEXT_FORMAT_DECIMAL_GENERAL_UPPER = 'G'

// FLOAT_TEXT_KIND_BINARY is first validated representation kind.
const FLOAT_TEXT_KIND_BINARY Float_Text_Format = 0

// FLOAT_TEXT_KIND_HEXADECIMAL_FRACTION follows binary kind.
const FLOAT_TEXT_KIND_HEXADECIMAL_FRACTION = FLOAT_TEXT_KIND_BINARY + WORD_COUNT_INCREMENT

// FLOAT_TEXT_KIND_HEXADECIMAL follows fractional hexadecimal kind.
const FLOAT_TEXT_KIND_HEXADECIMAL = FLOAT_TEXT_KIND_HEXADECIMAL_FRACTION +
	WORD_COUNT_INCREMENT

// FLOAT_TEXT_KIND_DECIMAL_EXPONENT follows power-of-two kinds.
const FLOAT_TEXT_KIND_DECIMAL_EXPONENT = FLOAT_TEXT_KIND_HEXADECIMAL + WORD_COUNT_INCREMENT

// FLOAT_TEXT_KIND_DECIMAL_EXPONENT_UPPER follows lowercase scientific notation.
const FLOAT_TEXT_KIND_DECIMAL_EXPONENT_UPPER = FLOAT_TEXT_KIND_DECIMAL_EXPONENT +
	WORD_COUNT_INCREMENT

// FLOAT_TEXT_KIND_DECIMAL_FIXED follows scientific notation kinds.
const FLOAT_TEXT_KIND_DECIMAL_FIXED = FLOAT_TEXT_KIND_DECIMAL_EXPONENT_UPPER +
	WORD_COUNT_INCREMENT

// FLOAT_TEXT_KIND_DECIMAL_GENERAL follows fixed decimal kind.
const FLOAT_TEXT_KIND_DECIMAL_GENERAL = FLOAT_TEXT_KIND_DECIMAL_FIXED + WORD_COUNT_INCREMENT

// FLOAT_TEXT_KIND_DECIMAL_GENERAL_UPPER is final validated representation kind.
const FLOAT_TEXT_KIND_DECIMAL_GENERAL_UPPER = FLOAT_TEXT_KIND_DECIMAL_GENERAL +
	WORD_COUNT_INCREMENT

// FLOAT_TEXT_PRECISION_MINIMUM requests shortest exact identifying text.
const FLOAT_TEXT_PRECISION_MINIMUM = -WORD_COUNT_INCREMENT

// FLOAT_TEXT_PRECISION_MAXIMUM keeps hexadecimal rounding inside Float precision.
const FLOAT_TEXT_PRECISION_MAXIMUM = (FLOAT_PRECISION_MAXIMUM - WORD_COUNT_INCREMENT) /
	BASE_HEXADECIMAL_DIGIT_BIT_COUNT

// FLOAT_TEXT_PRECISION_UNVALIDATED_MINIMUM admits one hostile low precision.
const FLOAT_TEXT_PRECISION_UNVALIDATED_MINIMUM = FLOAT_TEXT_PRECISION_MINIMUM -
	WORD_COUNT_INCREMENT

// FLOAT_TEXT_PRECISION_UNVALIDATED_MAXIMUM admits one hostile high precision.
const FLOAT_TEXT_PRECISION_UNVALIDATED_MAXIMUM = FLOAT_TEXT_PRECISION_MAXIMUM +
	WORD_COUNT_INCREMENT

// FLOAT_TEXT_INTEGER_DIGIT_COUNT_MAXIMUM bounds full-width decimal significand.
const FLOAT_TEXT_INTEGER_DIGIT_COUNT_MAXIMUM = BIT_COUNT_MAXIMUM*
	DECIMAL_DIGIT_BINARY_LOGARITHM_CEILING/
	DECIMAL_DIGIT_BINARY_LOGARITHM_SCALE + WORD_COUNT_INCREMENT

// Half-ulp spacing bounds decimal places needed to distinguish the smallest finite values.
const FLOAT_TEXT_FRACTION_DIGIT_COUNT_MAXIMUM = FLOAT_DECIMAL_RIGHT_SHIFT_MAXIMUM*
	DECIMAL_DIGIT_BINARY_LOGARITHM_CEILING/DECIMAL_DIGIT_BINARY_LOGARITHM_SCALE +
	WORD_COUNT_INCREMENT

// Explicit fixed precision combines the largest integer part with all requested places.
const FLOAT_TEXT_EXPLICIT_SIZE_MAXIMUM = SIGN_BYTE_COUNT_MAXIMUM +
	FLOAT_TEXT_INTEGER_DIGIT_COUNT_MAXIMUM + DECIMAL_POINT_BYTE_COUNT +
	FLOAT_TEXT_PRECISION_MAXIMUM

// Shortest tiny values need more fixed bytes than maximum explicit fractional precision.
const FLOAT_TEXT_SIZE_MAXIMUM = SIGN_BYTE_COUNT_MAXIMUM + WORD_COUNT_INCREMENT +
	DECIMAL_POINT_BYTE_COUNT + FLOAT_TEXT_FRACTION_DIGIT_COUNT_MAXIMUM

// FLOAT_DECIMAL_DIGIT_NUMERATOR_MAXIMUM keeps the logarithm formula integral.
const FLOAT_DECIMAL_DIGIT_NUMERATOR_MAXIMUM = BIT_COUNT_MAXIMUM *
	DECIMAL_DIGIT_BINARY_LOGARITHM_CEILING

// FLOAT_DECIMAL_INTEGER_DIGIT_COUNT_MAXIMUM includes one shortest-bound guard bit.
const FLOAT_DECIMAL_INTEGER_DIGIT_COUNT_MAXIMUM = FLOAT_DECIMAL_DIGIT_NUMERATOR_MAXIMUM/
	DECIMAL_DIGIT_BINARY_LOGARITHM_SCALE + WORD_COUNT_INCREMENT

// FLOAT_DECIMAL_RIGHT_SHIFT_MAXIMUM includes one shortest-bound half-ulp bit.
const FLOAT_DECIMAL_RIGHT_SHIFT_MAXIMUM = FLOAT_PRECISION_MAXIMUM -
	FLOAT_EXPONENT_MINIMUM + WORD_COUNT_INCREMENT

// Midpoint numerator is below 10^q / 2^-emin, where q includes the half-ulp guard.
const FLOAT_DECIMAL_DIGIT_COUNT_MAXIMUM = FLOAT_DECIMAL_RIGHT_SHIFT_MAXIMUM -
	(-FLOAT_EXPONENT_MINIMUM*DECIMAL_DIGIT_BINARY_LOGARITHM_CEILING)/
		DECIMAL_DIGIT_BINARY_LOGARITHM_SCALE

// FLOAT_PARSE_TEXT_SIZE_MAXIMUM keeps syntax work inside the shared numeric text bound.
const FLOAT_PARSE_TEXT_SIZE_MAXIMUM = RAT_PARSE_TEXT_SIZE_MAXIMUM

// FLOAT_PARSE_TEXT_UNVALIDATED_SIZE_MAXIMUM admits one hostile oversized source.
const FLOAT_PARSE_TEXT_UNVALIDATED_SIZE_MAXIMUM = FLOAT_PARSE_TEXT_SIZE_MAXIMUM +
	WORD_COUNT_INCREMENT

// FLOAT_PARSE_BASE_AUTOMATIC defers radix choice to a source prefix.
const FLOAT_PARSE_BASE_AUTOMATIC Float_Parse_Base = 0

// FLOAT_PARSE_BASE_BINARY identifies radix two after validation or prefix scanning.
const FLOAT_PARSE_BASE_BINARY = FLOAT_PARSE_BASE_AUTOMATIC + WORD_COUNT_INCREMENT

// FLOAT_PARSE_BASE_OCTAL identifies radix eight after validation or prefix scanning.
const FLOAT_PARSE_BASE_OCTAL = FLOAT_PARSE_BASE_BINARY + WORD_COUNT_INCREMENT

// FLOAT_PARSE_BASE_DECIMAL identifies radix ten after validation or default scanning.
const FLOAT_PARSE_BASE_DECIMAL = FLOAT_PARSE_BASE_OCTAL + WORD_COUNT_INCREMENT

// FLOAT_PARSE_BASE_HEXADECIMAL identifies radix sixteen after validation or prefix scanning.
const FLOAT_PARSE_BASE_HEXADECIMAL = FLOAT_PARSE_BASE_DECIMAL + WORD_COUNT_INCREMENT

// FLOAT_PARSE_SOURCE_COUNT_INDEX stores validated caller text size.
const FLOAT_PARSE_SOURCE_COUNT_INDEX = 0

// FLOAT_PARSE_BASE_INDEX stores validated requested base kind.
const FLOAT_PARSE_BASE_INDEX = FLOAT_PARSE_SOURCE_COUNT_INDEX + WORD_COUNT_INCREMENT

// FLOAT_PARSE_SEPARATOR_INDEX stores whether automatic syntax admits underscores.
const FLOAT_PARSE_SEPARATOR_INDEX = FLOAT_PARSE_BASE_INDEX + WORD_COUNT_INCREMENT

// FLOAT_PARSE_NEGATIVE_INDEX stores the leading sign independently from zero.
const FLOAT_PARSE_NEGATIVE_INDEX = FLOAT_PARSE_SEPARATOR_INDEX + WORD_COUNT_INCREMENT

// FLOAT_PARSE_PREFIXED_INDEX stores whether automatic radix syntax consumed a prefix.
const FLOAT_PARSE_PREFIXED_INDEX = FLOAT_PARSE_NEGATIVE_INDEX + WORD_COUNT_INCREMENT

// FLOAT_PARSE_RADIX_INDEX stores resolved numeric radix.
const FLOAT_PARSE_RADIX_INDEX = FLOAT_PARSE_PREFIXED_INDEX + WORD_COUNT_INCREMENT

// FLOAT_PARSE_START_INDEX stores first mantissa byte.
const FLOAT_PARSE_START_INDEX = FLOAT_PARSE_RADIX_INDEX + WORD_COUNT_INCREMENT

// FLOAT_PARSE_MANTISSA_COUNT_INDEX stores parsed magnitude words.
const FLOAT_PARSE_MANTISSA_COUNT_INDEX = FLOAT_PARSE_START_INDEX + WORD_COUNT_INCREMENT

// FLOAT_PARSE_FRACTIONAL_DIGIT_COUNT_INDEX stores digits after the radix point.
const FLOAT_PARSE_FRACTIONAL_DIGIT_COUNT_INDEX = FLOAT_PARSE_MANTISSA_COUNT_INDEX +
	WORD_COUNT_INCREMENT

// FLOAT_PARSE_END_INDEX stores first byte after the mantissa.
const FLOAT_PARSE_END_INDEX = FLOAT_PARSE_FRACTIONAL_DIGIT_COUNT_INDEX +
	WORD_COUNT_INCREMENT

// FLOAT_PARSE_EXPONENT_INDEX stores the complete signed exponent.
const FLOAT_PARSE_EXPONENT_INDEX = FLOAT_PARSE_END_INDEX + WORD_COUNT_INCREMENT

// FLOAT_PARSE_EXPONENT_BASE_INDEX stores decimal or binary exponent scaling.
const FLOAT_PARSE_EXPONENT_BASE_INDEX = FLOAT_PARSE_EXPONENT_INDEX + WORD_COUNT_INCREMENT

// FLOAT_PARSE_EXPONENT_START_INDEX stores first exponent sign or digit byte.
const FLOAT_PARSE_EXPONENT_START_INDEX = FLOAT_PARSE_EXPONENT_BASE_INDEX +
	WORD_COUNT_INCREMENT

// FLOAT_PARSE_EXPONENT_2_INDEX stores complete binary scaling.
const FLOAT_PARSE_EXPONENT_2_INDEX = FLOAT_PARSE_EXPONENT_START_INDEX + WORD_COUNT_INCREMENT

// FLOAT_PARSE_EXPONENT_5_INDEX stores complete decimal-prime scaling.
const FLOAT_PARSE_EXPONENT_5_INDEX = FLOAT_PARSE_EXPONENT_2_INDEX + WORD_COUNT_INCREMENT

// FLOAT_PARSE_CONTROL_COUNT is complete structural scanner state.
const FLOAT_PARSE_CONTROL_COUNT = FLOAT_PARSE_EXPONENT_5_INDEX + WORD_COUNT_INCREMENT

// FLOAT_DECIMAL_SHIFT_MAXIMUM keeps one decimal digit product inside a Word.
const FLOAT_DECIMAL_SHIFT_MAXIMUM = WORD_BIT_COUNT - BASE_HEXADECIMAL_DIGIT_BIT_COUNT

// FLOAT_DECIMAL_VALUE_INDEX selects exact source decimal.
const FLOAT_DECIMAL_VALUE_INDEX = WORD_COUNT_MINIMUM

// FLOAT_DECIMAL_LOWER_INDEX selects shortest-rounding lower midpoint.
const FLOAT_DECIMAL_LOWER_INDEX = FLOAT_DECIMAL_VALUE_INDEX + WORD_COUNT_INCREMENT

// FLOAT_DECIMAL_UPPER_INDEX selects shortest-rounding upper midpoint.
const FLOAT_DECIMAL_UPPER_INDEX = FLOAT_DECIMAL_LOWER_INDEX + WORD_COUNT_INCREMENT

// FLOAT_DECIMAL_COUNT owns exact value and both shortest bounds.
const FLOAT_DECIMAL_COUNT = FLOAT_DECIMAL_UPPER_INDEX + WORD_COUNT_INCREMENT

// FLOAT_DECIMAL_DIGIT_COUNT_INDEX selects populated exact decimal digits.
const FLOAT_DECIMAL_DIGIT_COUNT_INDEX = WORD_COUNT_MINIMUM

// FLOAT_DECIMAL_EXPONENT_INDEX selects decimal point position.
const FLOAT_DECIMAL_EXPONENT_INDEX = FLOAT_DECIMAL_DIGIT_COUNT_INDEX + WORD_COUNT_INCREMENT

// FLOAT_DECIMAL_CONTROL_COUNT owns decimal length and point position.
const FLOAT_DECIMAL_CONTROL_COUNT = FLOAT_DECIMAL_EXPONENT_INDEX + WORD_COUNT_INCREMENT

// FLOAT_TEXT_MAGNITUDE_WORD_COUNT_MAXIMUM includes one shortest-bound guard bit.
const FLOAT_TEXT_MAGNITUDE_WORD_COUNT_MAXIMUM = WORD_COUNT_MAXIMUM + WORD_COUNT_INCREMENT

// FLOAT_TEXT_SOURCE_INDEX selects preserved caller value.
const FLOAT_TEXT_SOURCE_INDEX = WORD_COUNT_MINIMUM

// FLOAT_TEXT_ROUNDED_INDEX selects rounded hexadecimal copy.
const FLOAT_TEXT_ROUNDED_INDEX = FLOAT_TEXT_SOURCE_INDEX + WORD_COUNT_INCREMENT

// FLOAT_TEXT_VALUE_COUNT owns source and rounded values.
const FLOAT_TEXT_VALUE_COUNT = FLOAT_TEXT_ROUNDED_INDEX + WORD_COUNT_INCREMENT

// FLOAT_TEXT_OUTPUT_COUNT_INDEX selects populated transactional byte count.
const FLOAT_TEXT_OUTPUT_COUNT_INDEX = WORD_COUNT_MINIMUM

// FLOAT_TEXT_PRECISION_INDEX selects validated requested digit count.
const FLOAT_TEXT_PRECISION_INDEX = FLOAT_TEXT_OUTPUT_COUNT_INDEX + WORD_COUNT_INCREMENT

// FLOAT_TEXT_EXPONENT_MINIMUM_DIGIT_COUNT_INDEX selects exponent padding policy.
const FLOAT_TEXT_EXPONENT_MINIMUM_DIGIT_COUNT_INDEX = FLOAT_TEXT_PRECISION_INDEX +
	WORD_COUNT_INCREMENT

// FLOAT_TEXT_FORMAT_INDEX selects validated representation kind.
const FLOAT_TEXT_FORMAT_INDEX = FLOAT_TEXT_EXPONENT_MINIMUM_DIGIT_COUNT_INDEX +
	WORD_COUNT_INCREMENT

// FLOAT_TEXT_DECIMAL_SHIFT_INDEX selects one bounded decimal right shift.
const FLOAT_TEXT_DECIMAL_SHIFT_INDEX = FLOAT_TEXT_FORMAT_INDEX + WORD_COUNT_INCREMENT

// FLOAT_TEXT_BINARY_SHIFT_INDEX selects exact magnitude power of two.
const FLOAT_TEXT_BINARY_SHIFT_INDEX = FLOAT_TEXT_DECIMAL_SHIFT_INDEX + WORD_COUNT_INCREMENT

// FLOAT_TEXT_ROUND_DIGIT_COUNT_INDEX selects retained significant decimal digits.
const FLOAT_TEXT_ROUND_DIGIT_COUNT_INDEX = FLOAT_TEXT_BINARY_SHIFT_INDEX +
	WORD_COUNT_INCREMENT

// FLOAT_TEXT_CONTROL_COUNT owns complete scalar conversion state.
const FLOAT_TEXT_CONTROL_COUNT = FLOAT_TEXT_ROUND_DIGIT_COUNT_INDEX +
	WORD_COUNT_INCREMENT

// FLOAT_GOB_VERSION retains stdlib wire compatibility.
const FLOAT_GOB_VERSION = WORD_COUNT_INCREMENT

// FLOAT_GOB_NEGATIVE_BIT_COUNT reserves low attribute sign bit.
const FLOAT_GOB_NEGATIVE_BIT_COUNT = WORD_COUNT_INCREMENT

// FLOAT_GOB_FORM_BIT_COUNT retains all Float forms and one invalid wire value.
const FLOAT_GOB_FORM_BIT_COUNT = BASE_BINARY

// FLOAT_GOB_ACCURACY_BIT_COUNT retains three accuracies and one invalid wire value.
const FLOAT_GOB_ACCURACY_BIT_COUNT = BASE_BINARY

// FLOAT_GOB_MODE_BIT_COUNT consumes remaining attribute byte bits.
const FLOAT_GOB_MODE_BIT_COUNT = bits.BIT_COUNT_8_MAXIMUM -
	FLOAT_GOB_NEGATIVE_BIT_COUNT - FLOAT_GOB_FORM_BIT_COUNT -
	FLOAT_GOB_ACCURACY_BIT_COUNT

// FLOAT_GOB_FORM_SHIFT leaves low sign bit intact.
const FLOAT_GOB_FORM_SHIFT = FLOAT_GOB_NEGATIVE_BIT_COUNT

// FLOAT_GOB_ACCURACY_SHIFT places accuracy above form.
const FLOAT_GOB_ACCURACY_SHIFT = FLOAT_GOB_FORM_SHIFT + FLOAT_GOB_FORM_BIT_COUNT

// FLOAT_GOB_MODE_SHIFT places mode in highest attribute bits.
const FLOAT_GOB_MODE_SHIFT = FLOAT_GOB_ACCURACY_SHIFT + FLOAT_GOB_ACCURACY_BIT_COUNT

// FLOAT_GOB_NEGATIVE_MASK selects low sign bit.
const FLOAT_GOB_NEGATIVE_MASK = byte(bits.CARRY_MAXIMUM)

// FLOAT_GOB_FORM_MASK selects encoded form.
const FLOAT_GOB_FORM_MASK = byte(
	WORD_COUNT_INCREMENT<<FLOAT_GOB_FORM_BIT_COUNT - WORD_COUNT_INCREMENT,
)

// FLOAT_GOB_ACCURACY_MASK selects encoded accuracy offset.
const FLOAT_GOB_ACCURACY_MASK = byte(
	WORD_COUNT_INCREMENT<<FLOAT_GOB_ACCURACY_BIT_COUNT - WORD_COUNT_INCREMENT,
)

// FLOAT_GOB_MODE_MASK selects encoded rounding mode.
const FLOAT_GOB_MODE_MASK = byte(
	WORD_COUNT_INCREMENT<<FLOAT_GOB_MODE_BIT_COUNT - WORD_COUNT_INCREMENT,
)

// FLOAT_GOB_VERSION_OFFSET locates wire version.
const FLOAT_GOB_VERSION_OFFSET = WORD_COUNT_MINIMUM

// FLOAT_GOB_ATTRIBUTES_OFFSET follows version byte.
const FLOAT_GOB_ATTRIBUTES_OFFSET = FLOAT_GOB_VERSION_OFFSET + WORD_COUNT_INCREMENT

// FLOAT_GOB_PRECISION_OFFSET follows packed attributes.
const FLOAT_GOB_PRECISION_OFFSET = FLOAT_GOB_ATTRIBUTES_OFFSET + WORD_COUNT_INCREMENT

// FLOAT_GOB_FIELD_SIZE stores one big-endian uint32.
const FLOAT_GOB_FIELD_SIZE = bits.BIT_COUNT_32_MAXIMUM / bits.BIT_COUNT_8_MAXIMUM

// FLOAT_GOB_HEADER_SIZE includes version, attributes, and precision.
const FLOAT_GOB_HEADER_SIZE = FLOAT_GOB_PRECISION_OFFSET + FLOAT_GOB_FIELD_SIZE

// FLOAT_GOB_EXPONENT_OFFSET follows common header.
const FLOAT_GOB_EXPONENT_OFFSET = FLOAT_GOB_HEADER_SIZE

// FLOAT_GOB_FINITE_PREFIX_SIZE includes finite exponent.
const FLOAT_GOB_FINITE_PREFIX_SIZE = FLOAT_GOB_EXPONENT_OFFSET + FLOAT_GOB_FIELD_SIZE

// FLOAT_GOB_FINITE_ENCODING_SIZE_MINIMUM includes one complete mantissa word.
const FLOAT_GOB_FINITE_ENCODING_SIZE_MINIMUM = FLOAT_GOB_FINITE_PREFIX_SIZE +
	BASE_BINARY*FLOAT_GOB_FIELD_SIZE

// FLOAT_GOB_MANTISSA_OFFSET follows finite prefix.
const FLOAT_GOB_MANTISSA_OFFSET = FLOAT_GOB_FINITE_PREFIX_SIZE

// FLOAT_GOB_SIZE_MAXIMUM includes one full bounded mantissa.
const FLOAT_GOB_SIZE_MAXIMUM = FLOAT_GOB_FINITE_PREFIX_SIZE +
	WORD_COUNT_MAXIMUM*WORD_BYTE_COUNT

// FLOAT_GOB_UNVALIDATED_SIZE_MAXIMUM admits one hostile excess byte.
const FLOAT_GOB_UNVALIDATED_SIZE_MAXIMUM = FLOAT_GOB_SIZE_MAXIMUM +
	WORD_COUNT_INCREMENT

// FLOAT_GOB_MANTISSA_INDEX selects alignment and decode storage.
const FLOAT_GOB_MANTISSA_INDEX = WORD_COUNT_MINIMUM

// FLOAT_GOB_MANTISSA_COUNT owns one transactional magnitude.
const FLOAT_GOB_MANTISSA_COUNT = FLOAT_GOB_MANTISSA_INDEX + WORD_COUNT_INCREMENT

// FLOAT_GOB_MANTISSA_SIZE_MINIMUM includes shortest common header shortfall.
const FLOAT_GOB_MANTISSA_SIZE_MINIMUM = -FLOAT_GOB_FIELD_SIZE

// FLOAT_GOB_MANTISSA_SIZE_MAXIMUM fills complete bounded wire payload.
const FLOAT_GOB_MANTISSA_SIZE_MAXIMUM = FLOAT_GOB_SIZE_MAXIMUM -
	FLOAT_GOB_MANTISSA_OFFSET

// Float_Gob_Encoding keeps caller output inside complete wire bound.
type Float_Gob_Encoding []byte

// Float_Gob_Encoding_Invariants bounds caller destination before writes.
func Float_Gob_Encoding_Invariants(
	value Float_Gob_Encoding, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), WORD_COUNT_MINIMUM, FLOAT_GOB_SIZE_MAXIMUM,
		).
		Ensure()
}

// Float_Gob_Encoding_Unvalidated admits one hostile excess input.
type Float_Gob_Encoding_Unvalidated []byte

// Float_Gob_Encoding_Unvalidated_Invariants bounds validation work.
func Float_Gob_Encoding_Unvalidated_Invariants(
	value Float_Gob_Encoding_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), WORD_COUNT_MINIMUM,
			FLOAT_GOB_UNVALIDATED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Float_Gob_Finite_Encoding holds one structurally complete finite payload.
type Float_Gob_Finite_Encoding []byte

// Float_Gob_Finite_Encoding_Invariants binds payload length and alignment after validation.
func Float_Gob_Finite_Encoding_Invariants(
	value Float_Gob_Finite_Encoding, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), FLOAT_GOB_FINITE_ENCODING_SIZE_MINIMUM,
			FLOAT_GOB_SIZE_MAXIMUM,
		).
		Ensure()
	aver.Always(
		(len(value)-FLOAT_GOB_MANTISSA_OFFSET)%WORD_BYTE_COUNT == WORD_COUNT_MINIMUM,
		"A finite float gob payload holds complete mantissa words.",
	)
	high_bit_shift := bits.BIT_COUNT_8_MAXIMUM - WORD_COUNT_INCREMENT
	aver.Always(
		value[FLOAT_GOB_MANTISSA_OFFSET]>>high_bit_shift == byte(bits.CARRY_MAXIMUM),
		"A finite float gob payload has a normalized mantissa.",
	)
}

// Float_Gob_Header_Encoding prevents short source reads after common header validation.
type Float_Gob_Header_Encoding struct {
	// Version preserves the boundary's supported-version proof.
	Version Float_Gob_Version
	// Mode retains unsupported wire policies for rejection.
	Mode Float_Gob_Mode
	// Accuracy retains the reserved wire state for rejection.
	Accuracy Float_Gob_Accuracy
	// Form excludes the finite branch already dispatched by the boundary.
	Form Float_Gob_Nonfinite_Form
	// Negative preserves either sign independently of form validation.
	Negative Polarity
	// Precision_High prevents unchecked high bits from being discarded.
	Precision_High Float_Gob_Precision_High
	// Precision_Upper retains high precision bits for overflow rejection.
	Precision_Upper Float_Gob_Precision_Upper
	// Precision_Lower retains low precision bits for complete validation.
	Precision_Lower Float_Gob_Precision_Lower
	// Precision_Low preserves the wire value before range validation.
	Precision_Low Float_Gob_Precision_Low
}

// Float_Gob_Header_Encoding_Invariants fixes common wire header width.
func Float_Gob_Header_Encoding_Invariants(
	value Float_Gob_Header_Encoding, namespace aver.Namespace,
) {
	Float_Gob_Version_Invariants(value.Version, namespace)
	Float_Gob_Mode_Invariants(value.Mode, namespace)
	Float_Gob_Accuracy_Invariants(value.Accuracy, namespace)
	Float_Gob_Nonfinite_Form_Invariants(value.Form, namespace)
	Polarity_Invariants(value.Negative, namespace)
	Float_Gob_Precision_High_Invariants(value.Precision_High, namespace)
	Float_Gob_Precision_Upper_Invariants(value.Precision_Upper, namespace)
	Float_Gob_Precision_Lower_Invariants(value.Precision_Lower, namespace)
	Float_Gob_Precision_Low_Invariants(value.Precision_Low, namespace)
}

// Float_Gob_Precision_High retains the most significant unvalidated precision byte.
type Float_Gob_Precision_High byte

// Float_Gob_Precision_High_Invariants permits the complete wire domain.
func Float_Gob_Precision_High_Invariants(value Float_Gob_Precision_High, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), bits.WORD_8_MINIMUM, bits.WORD_8_MAXIMUM).
		Ensure()
}

// Float_Gob_Precision_Upper retains the second unvalidated precision byte.
type Float_Gob_Precision_Upper byte

// Float_Gob_Precision_Upper_Invariants permits the complete wire domain.
func Float_Gob_Precision_Upper_Invariants(
	value Float_Gob_Precision_Upper, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), bits.WORD_8_MINIMUM, bits.WORD_8_MAXIMUM).
		Ensure()
}

// Float_Gob_Precision_Lower retains the third unvalidated precision byte.
type Float_Gob_Precision_Lower byte

// Float_Gob_Precision_Lower_Invariants permits the complete wire domain.
func Float_Gob_Precision_Lower_Invariants(
	value Float_Gob_Precision_Lower, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), bits.WORD_8_MINIMUM, bits.WORD_8_MAXIMUM).
		Ensure()
}

// Float_Gob_Precision_Low retains the least significant unvalidated precision byte.
type Float_Gob_Precision_Low byte

// Float_Gob_Precision_Low_Invariants permits the complete wire domain.
func Float_Gob_Precision_Low_Invariants(value Float_Gob_Precision_Low, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), bits.WORD_8_MINIMUM, bits.WORD_8_MAXIMUM).
		Ensure()
}

// Float_Gob_Mantissa_Size includes malformed finite header shortfalls.
type Float_Gob_Mantissa_Size int

// Float_Gob_Mantissa_Size_Invariants bounds structural validation work.
func Float_Gob_Mantissa_Size_Invariants(
	value Float_Gob_Mantissa_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), FLOAT_GOB_MANTISSA_SIZE_MINIMUM,
			FLOAT_GOB_MANTISSA_SIZE_MAXIMUM,
		).
		Ensure()
}

// Float_Gob_Aligned_Mantissa_Size excludes incomplete and absent mantissa words.
type Float_Gob_Aligned_Mantissa_Size int

// Float_Gob_Aligned_Mantissa_Size_Invariants binds one complete word-aligned payload.
func Float_Gob_Aligned_Mantissa_Size_Invariants(
	value Float_Gob_Aligned_Mantissa_Size, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), WORD_BYTE_COUNT, FLOAT_GOB_MANTISSA_SIZE_MAXIMUM).
		Ensure()
	aver.Always(
		int(value)%WORD_BYTE_COUNT == WORD_COUNT_MINIMUM,
		"A validated float gob mantissa holds complete words.",
	)
}

// Float_Gob_Header holds validated policy before numeric payload decoding.
type Float_Gob_Header struct {
	// Precision stays separate until finite mantissa validates.
	Precision Float_Precision
	// Mode must survive exact decode when destination has no policy.
	Mode Rounding_Mode
	// Accuracy preserves wire operation history until destination rounding.
	Accuracy Accuracy
	// Form decides whether finite payload must exist.
	Form Float_Form
	// Negative preserves sign for zero and infinity too.
	Negative Polarity
}

// Float_Gob_Header_Invariants composes only validated Float policy fields.
func Float_Gob_Header_Invariants(
	value Float_Gob_Header, namespace aver.Namespace,
) {
	Float_Precision_Invariants(value.Precision, namespace)
	Rounding_Mode_Invariants(value.Mode, namespace)
	Accuracy_Invariants(value.Accuracy, namespace)
	Float_Form_Invariants(value.Form, namespace)
	Polarity_Invariants(value.Negative, namespace)
}

// Float_Gob_Finite_Header excludes policy-free finite encodings before payload work.
type Float_Gob_Finite_Header Float_Gob_Header

// Float_Gob_Finite_Header_Invariants states the finite payload policy domain.
func Float_Gob_Finite_Header_Invariants(
	value Float_Gob_Finite_Header, namespace aver.Namespace,
) {
	aver.Tree(Float_Active_Precision(value.Precision), namespace).
		Range_Uint(
			uint(value.Precision), WORD_COUNT_INCREMENT, FLOAT_PRECISION_MAXIMUM,
		).
		Ensure()
	aver.Tree(Rounding_Mode(value.Mode), namespace).
		Range_Uint8(
			uint8(value.Mode), uint8(ROUND_TO_NEAREST_EVEN),
			uint8(ROUND_TO_POSITIVE_INFINITY),
		).
		Ensure()
	aver.Tree(Accuracy(value.Accuracy), namespace).
		Enum_3_Int8(
			int8(value.Accuracy), int8(ACCURACY_BELOW), int8(ACCURACY_EXACT),
			int8(ACCURACY_ABOVE),
		).
		Ensure()
	aver.Always(
		value.Form == FLOAT_FORM_FINITE,
		"A finite gob header owns a mantissa payload.",
	)
	aver.Tree(Polarity(value.Negative), namespace).
		Enum_Uint8(
			uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE),
		).
		Ensure()
}

// Float_Gob_Nonfinite_Header excludes the mantissa fields absent from zero and infinity.
type Float_Gob_Nonfinite_Header Float_Gob_Header

// Float_Gob_Nonfinite_Header_Invariants states complete zero and infinity wire policy.
func Float_Gob_Nonfinite_Header_Invariants(
	value Float_Gob_Nonfinite_Header, namespace aver.Namespace,
) {
	aver.Tree(Float_Precision(value.Precision), namespace).
		Range_Uint(
			uint(value.Precision), FLOAT_PRECISION_MINIMUM,
			FLOAT_PRECISION_MAXIMUM,
		).
		Ensure()
	aver.Tree(Rounding_Mode(value.Mode), namespace).
		Range_Uint8(
			uint8(value.Mode), uint8(ROUND_TO_NEAREST_EVEN),
			uint8(ROUND_TO_POSITIVE_INFINITY),
		).
		Ensure()
	aver.Tree(Accuracy(value.Accuracy), namespace).
		Enum_3_Int8(
			int8(value.Accuracy), int8(ACCURACY_BELOW), int8(ACCURACY_EXACT),
			int8(ACCURACY_ABOVE),
		).
		Ensure()
	aver.Tree(Float_Form(value.Form), namespace).
		Enum_Uint8(
			uint8(value.Form), uint8(FLOAT_FORM_ZERO), uint8(FLOAT_FORM_INFINITY),
		).
		Ensure()
	aver.Tree(Polarity(value.Negative), namespace).
		Enum_Uint8(
			uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE),
		).
		Ensure()
}

// Float_Gob_Mantissas keeps one Int without duplicating Int invariant chains.
type Float_Gob_Mantissas struct {
	// Value keeps hostile mantissa input transactional until validation succeeds.
	Value Int
}

// Float_Gob_Mantissas_Invariants fixes transactional storage count.
func Float_Gob_Mantissas_Invariants(
	value Float_Gob_Mantissas, namespace aver.Namespace,
) {
	Int_Invariants(value.Value, namespace)
}

// Float_32_Bits keeps raw binary32 input distinct from numeric-only output.
type Float_32_Bits uint32

// Float_32_Bits_Invariants admits every binary32 encoding, including NaN.
func Float_32_Bits_Invariants(value Float_32_Bits, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, bits.WORD_32_MAXIMUM).
		Ensure()
}

// Float_32_Finite_Exponent_Field prevents NaN and infinity from entering finite decoding.
type Float_32_Finite_Exponent_Field uint32

// Float_32_Finite_Exponent_Field_Invariants bounds finite encoded exponent.
func Float_32_Finite_Exponent_Field_Invariants(
	value Float_32_Finite_Exponent_Field, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint32(
			uint32(value), bits.WORD_32_MINIMUM,
			FLOAT_32_FINITE_EXPONENT_FIELD_MAXIMUM,
		).
		Ensure()
}

// Float_32_Mantissa_Field excludes bits owned by exponent and sign.
type Float_32_Mantissa_Field uint32

// Float_32_Mantissa_Field_Invariants binds stored significand width.
func Float_32_Mantissa_Field_Invariants(
	value Float_32_Mantissa_Field, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint32(
			uint32(value), bits.WORD_32_MINIMUM, FLOAT_32_MANTISSA_MASK,
		).
		Ensure()
}

// Float_32_Sign keeps only clear or encoded binary32 sign.
type Float_32_Sign uint32

// Float_32_Sign_Invariants rejects mantissa or exponent bits.
func Float_32_Sign_Invariants(value Float_32_Sign, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint32(uint32(value), bits.WORD_32_MINIMUM, FLOAT_32_SIGN_MASK).
		Ensure()
}

// Float_32_Value_Bits excludes NaN while retaining signed infinity and zero.
type Float_32_Value_Bits uint32

// Float_32_Value_Bits_Invariants rejects every NaN payload.
func Float_32_Value_Bits_Invariants(
	value Float_32_Value_Bits, namespace aver.Namespace,
) {
	encoded := uint32(value)
	aver.Tree(value, namespace).
		Range_Uint32(
			uint32(value), FLOAT_32_VALUE_BITS_MINIMUM,
			FLOAT_32_VALUE_BITS_MAXIMUM,
		).
		Ensure()
	exponent := encoded >> FLOAT_32_EXPONENT_SHIFT & FLOAT_32_EXPONENT_MASK
	mantissa := encoded & FLOAT_32_MANTISSA_MASK
	exponent_difference := exponent ^ FLOAT_32_EXPONENT_MASK
	exponent_difference_nonzero := (exponent_difference | -exponent_difference) >>
		(bits.BIT_COUNT_32_MAXIMUM - WORD_COUNT_INCREMENT)
	exponent_is_nonfinite := uint32(bits.CARRY_MAXIMUM) - exponent_difference_nonzero
	mantissa_nonzero := (mantissa | -mantissa) >>
		(bits.BIT_COUNT_32_MAXIMUM - WORD_COUNT_INCREMENT)
	not_a_number := exponent_is_nonfinite * mantissa_nonzero
	aver.Always(
		not_a_number == bits.WORD_32_MINIMUM,
		"Numeric binary32 output never encodes NaN.",
	)
}

// Int_Float_64_Value_Bits is one integral binary64 result or signed infinity.
type Int_Float_64_Value_Bits uint64

// Int_Float_64_Value_Bits_Invariants excludes negative zero, subnormal, and NaN.
func Int_Float_64_Value_Bits_Invariants(
	value Int_Float_64_Value_Bits, namespace aver.Namespace,
) {
	encoded := uint64(value)
	aver.Tree(value, namespace).
		Range_Holed_Uint64(
			uint64(value), INT_FLOAT_64_VALUE_BITS_MINIMUM,
			INT_FLOAT_64_VALUE_BITS_MAXIMUM,
			INT_FLOAT_64_SUBNORMAL_HOLE_1, INT_FLOAT_64_SUBNORMAL_HOLE_2,
			INT_FLOAT_64_SUBNORMAL_HOLE_3,
		).
		Ensure()
	exponent := encoded >> FLOAT_64_EXPONENT_SHIFT & FLOAT_64_EXPONENT_MASK
	mantissa := encoded & FLOAT_64_MANTISSA_MASK
	sign := encoded & FLOAT_64_SIGN_MASK
	Int_Float_64_Sign_Invariants(Int_Float_64_Sign(sign), namespace)
	Int_Float_64_Mantissa_Field_Invariants(
		Int_Float_64_Mantissa_Field(mantissa), namespace,
	)
	nonzero := (encoded | -encoded) >> WORD_BIT_INDEX_MAXIMUM
	exponent_nonzero := (exponent | -exponent) >> WORD_BIT_INDEX_MAXIMUM
	aver.Always(
		nonzero <= exponent_nonzero,
		"Integral binary64 output gives zero no negative or subnormal twin.",
	)
	exponent_difference := exponent ^ FLOAT_64_EXPONENT_MASK
	exponent_difference_nonzero := (exponent_difference | -exponent_difference) >>
		WORD_BIT_INDEX_MAXIMUM
	exponent_is_nonfinite := uint64(bits.CARRY_MAXIMUM) - exponent_difference_nonzero
	mantissa_nonzero := (mantissa | -mantissa) >> WORD_BIT_INDEX_MAXIMUM
	aver.Always(
		exponent_is_nonfinite*mantissa_nonzero == bits.WORD_64_MINIMUM,
		"Integral binary64 output reserves the nonfinite field for infinity.",
	)
	zero := uint64(bits.CARRY_MAXIMUM) - nonzero
	mapped_exponent := exponent*nonzero + uint64(FLOAT_64_EXPONENT_BIAS)*zero
	Int_Float_64_Exponent_Field_Invariants(
		Int_Float_64_Exponent_Field(mapped_exponent), namespace,
	)
}

// Int_Float_64_Exponent_Field excludes fractional normal exponents.
type Int_Float_64_Exponent_Field uint64

// Int_Float_64_Exponent_Field_Invariants starts at the encoding for one.
func Int_Float_64_Exponent_Field_Invariants(
	value Int_Float_64_Exponent_Field, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(
			uint64(value), FLOAT_64_EXPONENT_BIAS,
			FLOAT_64_EXPONENT_MASK,
		).
		Ensure()
}

// Int_Float_64_Mantissa_Field is one stored integral significand field.
type Int_Float_64_Mantissa_Field uint64

// Int_Float_64_Mantissa_Field_Invariants admits every binary64 mantissa field.
func Int_Float_64_Mantissa_Field_Invariants(
	value Int_Float_64_Mantissa_Field, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(
			uint64(value), bits.WORD_64_MINIMUM,
			FLOAT_64_MANTISSA_MASK,
		).
		Ensure()
}

// Int_Float_64_Sign is either cleared or the binary64 sign bit.
type Int_Float_64_Sign uint64

// Int_Float_64_Sign_Invariants admits both integer polarities.
func Int_Float_64_Sign_Invariants(
	value Int_Float_64_Sign, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint64(
			uint64(value), bits.WORD_64_MINIMUM, FLOAT_64_SIGN_MASK,
		).
		Ensure()
}

// Rat_Float_32_Mantissa is the rounded significand before hidden-bit removal.
type Rat_Float_32_Mantissa uint32

// Rat_Float_32_Mantissa_Invariants bounds the final binary32 significand.
func Rat_Float_32_Mantissa_Invariants(
	value Rat_Float_32_Mantissa, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint32(
			uint32(value), bits.WORD_32_MINIMUM,
			FLOAT_32_VALUE_MANTISSA_MAXIMUM,
		).
		Ensure()
}

// Rat_Float_32_Rounding_Mantissa retains one discarded low bit.
type Rat_Float_32_Rounding_Mantissa uint32

// Rat_Float_32_Rounding_Mantissa_Invariants bounds the guarded significand.
func Rat_Float_32_Rounding_Mantissa_Invariants(
	value Rat_Float_32_Rounding_Mantissa, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint32(
			uint32(value), FLOAT_32_ROUNDING_MANTISSA_MINIMUM,
			FLOAT_32_ROUNDING_MANTISSA_MAXIMUM,
		).
		Ensure()
}

// Rat_Float_32_Exponent is the normalized binary exponent before field encoding.
type Rat_Float_32_Exponent int

// Rat_Float_32_Exponent_Invariants follows the complete bounded rational ratio.
func Rat_Float_32_Exponent_Invariants(
	value Rat_Float_32_Exponent, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), RAT_FLOAT_32_EXPONENT_MINIMUM,
			RAT_FLOAT_32_EXPONENT_MAXIMUM,
		).
		Ensure()
}

// Rat_Float_32_Rounding_Exponent excludes the final rounding carry.
type Rat_Float_32_Rounding_Exponent int

// Rat_Float_32_Rounding_Exponent_Invariants binds pre-round rational scale.
func Rat_Float_32_Rounding_Exponent_Invariants(
	value Rat_Float_32_Rounding_Exponent, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), RAT_FLOAT_32_EXPONENT_MINIMUM,
			RAT_FLOAT_32_ROUNDING_EXPONENT_MAXIMUM,
		).
		Ensure()
}

// Rat_Float_32_Workspace owns scaling and division outside caller rational.
type Rat_Float_32_Workspace struct {
	// Integers hold scaled operands and division results.
	Integers Rat_Float_32_Integers
	// Division owns rounding quotient and discarded remainder.
	Division Division_Memory
}

// Rat_Float_32_Workspace_Invariants composes complete binary32 conversion storage.
func Rat_Float_32_Workspace_Invariants(
	value Rat_Float_32_Workspace, namespace aver.Namespace,
) {
	Rat_Float_32_Integers_Invariants(value.Integers, namespace)
	Division_Memory_Invariants(value.Division, namespace)
}

// Modular_Square_Root_Status separates malformed bounds from mathematical absence.
type Modular_Square_Root_Status uint8

// Modular_Square_Root_Status_Invariants admits every transactional root outcome.
func Modular_Square_Root_Status_Invariants(
	value Modular_Square_Root_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_RESULT_ABSENT),
		).
		Ensure()
}

// Modular_Square_Root_Result_Status excludes validation after public input passes.
type Modular_Square_Root_Result_Status uint8

// Modular_Square_Root_Result_Status_Invariants admits root or mathematical absence.
func Modular_Square_Root_Result_Status_Invariants(
	value Modular_Square_Root_Result_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_RESULT_ABSENT),
		).
		Ensure()
}

// Modular_Square_Root_Integers owns complete Tonelli-Shanks state.
type Modular_Square_Root_Integers []Int

// Modular_Square_Root_Integers_Invariants fixes bounded algorithm capacity.
func Modular_Square_Root_Integers_Invariants(
	value Modular_Square_Root_Integers, _ aver.Namespace,
) {
	aver.Always(
		len(value) == MODULAR_SQUARE_ROOT_INTEGER_COUNT,
		"Modular square-root integer storage has fixed Tonelli-Shanks capacity.",
	)
}

// Modular_Square_Root_Modular_Memory gives nested arithmetic one chain identity.
type Modular_Square_Root_Modular_Memory Int_Modular_Workspace

// Modular_Square_Root_Modular_Memory_Invariants composes modular scratch once.
func Modular_Square_Root_Modular_Memory_Invariants(
	value Modular_Square_Root_Modular_Memory, namespace aver.Namespace,
) {
	aver.Always(
		len(value.Integers) == MODULAR_INTEGER_COUNT,
		"Modular square-root arithmetic owns complete modular integer state.",
	)
	Multiplication_Memory_Invariants(value.Multiplication, namespace)
	Division_Memory_Invariants(value.Division, namespace)
}

// Int_Modular_Square_Root_Workspace owns root, modular, and Jacobi scratch.
type Int_Modular_Square_Root_Workspace struct {
	// Integers preserve public inputs while nested workspaces mutate.
	Integers Modular_Square_Root_Integers
	// Modular owns every bounded multiplication and exponentiation temporary.
	Modular Modular_Square_Root_Modular_Memory
	// Jacobi reuses modular division while validating quadratic characters.
	Jacobi Jacobi_Integers
}

// Int_Modular_Square_Root_Workspace_Invariants composes complete bounded storage.
func Int_Modular_Square_Root_Workspace_Invariants(
	value Int_Modular_Square_Root_Workspace, namespace aver.Namespace,
) {
	Modular_Square_Root_Integers_Invariants(value.Integers, namespace)
	Modular_Square_Root_Modular_Memory_Invariants(value.Modular, namespace)
	Jacobi_Integers_Invariants(value.Jacobi, namespace)
}

// Float_Text_Format_Unvalidated admits every hostile format byte.
type Float_Text_Format_Unvalidated byte

// Float_Text_Format_Unvalidated_Invariants keeps format validation constant work.
func Float_Text_Format_Unvalidated_Invariants(
	value Float_Text_Format_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), bits.WORD_8_MINIMUM, bits.WORD_8_MAXIMUM).
		Ensure()
}

// Float_Text_Format is one validated representation kind.
type Float_Text_Format uint8

// Float_Text_Format_Invariants rejects every unsupported format byte.
func Float_Text_Format_Invariants(
	value Float_Text_Format, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(FLOAT_TEXT_KIND_BINARY),
			uint8(FLOAT_TEXT_KIND_DECIMAL_GENERAL_UPPER),
		).
		Ensure()
}

// Float_Text_Precision_Unvalidated admits one hostile value beyond each bound.
type Float_Text_Precision_Unvalidated int

// Float_Text_Precision_Unvalidated_Invariants bounds validation to one decision.
func Float_Text_Precision_Unvalidated_Invariants(
	value Float_Text_Precision_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), FLOAT_TEXT_PRECISION_UNVALIDATED_MINIMUM,
			FLOAT_TEXT_PRECISION_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Float_Text_Precision is shortest mode or one bounded digit count.
type Float_Text_Precision int

// Float_Text_Precision_Invariants binds text work and hexadecimal rounding.
func Float_Text_Precision_Invariants(
	value Float_Text_Precision, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), FLOAT_TEXT_PRECISION_MINIMUM,
			FLOAT_TEXT_PRECISION_MAXIMUM,
		).
		Ensure()
}

// Float_Text_Count reports exact required or populated caller bytes.
type Float_Text_Count int

// Float_Text_Count_Invariants includes zero for malformed format or precision.
func Float_Text_Count_Invariants(
	value Float_Text_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, FLOAT_TEXT_SIZE_MAXIMUM,
		).
		Ensure()
}

// Float_Text_Status separates malformed policy from short caller storage.
type Float_Text_Status uint8

// Float_Text_Status_Invariants admits every transactional text outcome.
func Float_Text_Status_Invariants(
	value Float_Text_Status, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_DESTINATION_TOO_SMALL),
		).
		Ensure()
}

// Float_Text_Workspace owns every conversion and transactional output byte.
type Float_Text_Workspace struct {
	// Integers hold aligned significand and signed exponent.
	Integers Float_Text_Integers
	// Values retain hexadecimal rounding outside caller value.
	Values Float_Text_Values
	// Text converts significand and exponent independently.
	Text Float_Text_Integer_Workspaces
	// Control holds output cursor, precision, and exponent padding.
	Control Float_Text_Control_Handle
	// Decimals hold exact value and shortest-rounding bounds.
	Decimals Float_Text_Decimals
	// Magnitude holds one precision-plus-guard shortest midpoint.
	Magnitude Float_Text_Magnitude
	// Output prevents partial caller mutation on short storage.
	Output Float_Text_Output
}

// Float_Text_Workspace_Invariants composes complete bounded text storage.
func Float_Text_Workspace_Invariants(
	value Float_Text_Workspace, namespace aver.Namespace,
) {
	Float_Text_Integers_Invariants(value.Integers, namespace)
	Float_Text_Values_Invariants(value.Values, namespace)
	Float_Text_Integer_Workspaces_Invariants(value.Text, namespace)
	Float_Text_Control_Handle_Invariants(value.Control, namespace)
	Float_Text_Decimals_Invariants(value.Decimals, namespace)
	Float_Text_Magnitude_Invariants(value.Magnitude, namespace)
	Float_Text_Output_Invariants(value.Output, namespace)
}

// Float_Parse_Base is one validated or resolved stdlib Float radix kind.
type Float_Parse_Base uint8

// Float_Parse_Base_Invariants keeps automatic and four numeric radix kinds exhaustive.
func Float_Parse_Base_Invariants(
	value Float_Parse_Base, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(FLOAT_PARSE_BASE_AUTOMATIC),
			uint8(FLOAT_PARSE_BASE_HEXADECIMAL),
		).
		Ensure()
}

// Float_Parse_Text_Unvalidated admits malformed text and one oversized hostile source.
type Float_Parse_Text_Unvalidated []byte

// Float_Parse_Text_Unvalidated_Invariants bounds scanning before any source copy.
func Float_Parse_Text_Unvalidated_Invariants(
	value Float_Parse_Text_Unvalidated, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM,
			FLOAT_PARSE_TEXT_UNVALIDATED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Float_Parse_Integer_Workspace gives mantissa storage one invariant identity.
type Float_Parse_Integer_Workspace Int_Parse_Workspace

// Float_Parse_Integer_Workspace_Invariants fixes full-width mantissa scratch.
func Float_Parse_Integer_Workspace_Invariants(
	value Float_Parse_Integer_Workspace, _ aver.Namespace,
) {
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Float parse mantissa retains one complete Int magnitude.")
}

// Float_Parse_Division_Workspace gives quotient storage one invariant identity.
type Float_Parse_Division_Workspace Float_Division_Workspace

// Float_Parse_Division_Workspace_Invariants fixes exact-factor quotient storage.
func Float_Parse_Division_Workspace_Invariants(
	value Float_Parse_Division_Workspace, _ aver.Namespace,
) {
	aver.Always(len(value.Remainder) == FLOAT_DIVISION_WORD_COUNT_MAXIMUM,
		"Float parse quotient retains one shifted remainder.")
	aver.Always(len(value.Divisor) == FLOAT_DIVISION_WORD_COUNT_MAXIMUM,
		"Float parse quotient retains one aligned divisor.")
}

// Float_Parse_Workspace owns source, exact factors, quotient, and syntax scratch.
type Float_Parse_Workspace struct {
	// Source keeps helper reads inside validated caller-independent storage.
	Source Float_Parse_Source
	// Control keeps source size and policy out of semantic invariant domains.
	Control Float_Parse_Control
	// Parse owns transactional full-width mantissa accumulation.
	Parse Float_Parse_Integer_Workspace
	// Integers own exact numerator and denominator factors.
	Integers Rat_Parse_Fraction_Integers
	// Values own exact operands and the rounded result.
	Values Float_Rat_Values
	// Division owns one correctly rounded exact-factor quotient.
	Division Float_Parse_Division_Workspace
}

// Float_Parse_Workspace_Invariants composes only always-valid structural storage.
func Float_Parse_Workspace_Invariants(
	value Float_Parse_Workspace, namespace aver.Namespace,
) {
	Float_Parse_Source_Invariants(value.Source, namespace)
	Float_Parse_Control_Invariants(value.Control, namespace)
	Float_Parse_Integer_Workspace_Invariants(value.Parse, namespace)
	Rat_Parse_Fraction_Integers_Invariants(value.Integers, namespace)
	Float_Rat_Values_Invariants(value.Values, namespace)
	Float_Parse_Division_Workspace_Invariants(value.Division, namespace)
}

// Float_Rat_Status separates nonfinite input from bounded component overflow.
type Float_Rat_Status uint8

// Float_Rat_Status_Invariants admits every transactional rational conversion result.
func Float_Rat_Status_Invariants(value Float_Rat_Status, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_VALUE_OVERFLOW),
		).
		Ensure()
}

// Float_Square_Root_Source_Bit_Index includes exhaustion beside valid mantissa bits.
type Float_Square_Root_Source_Bit_Index int

// Float_Square_Root_Source_Bit_Index_Invariants bounds one descending source cursor.
func Float_Square_Root_Source_Bit_Index_Invariants(
	value Float_Square_Root_Source_Bit_Index, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), BIT_INDEX_UNVALIDATED_MINIMUM,
			FLOAT_SQUARE_ROOT_SOURCE_BIT_INDEX_MAXIMUM,
		).
		Ensure()
}

// Float_Square_Root_Active_Word_Count bounds precision-plus-guard scratch access.
type Float_Square_Root_Active_Word_Count int

// Float_Square_Root_Active_Word_Count_Invariants keeps one nonempty active prefix.
func Float_Square_Root_Active_Word_Count_Invariants(
	value Float_Square_Root_Active_Word_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), WORD_COUNT_INCREMENT, FLOAT_SQUARE_ROOT_WORD_COUNT_MAXIMUM,
		).
		Ensure()
}

// Float_Square_Root_Pair is one radix-four source digit.
type Float_Square_Root_Pair Word

// Float_Square_Root_Pair_Invariants admits every two-bit digit.
func Float_Square_Root_Pair_Invariants(
	value Float_Square_Root_Pair, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_4_Uint64(
			uint64(value), uint64(BIT_CLEAR), uint64(BIT_SET),
			uint64(BASE_BINARY),
			uint64(FLOAT_SQUARE_ROOT_PAIR_MAXIMUM),
		).
		Ensure()
}

// Float_Binary_Precision is one IEEE destination significand width.
type Float_Binary_Precision uint

// Float_Binary_Precision_Invariants admits binary32 through binary64 precision.
func Float_Binary_Precision_Invariants(
	value Float_Binary_Precision, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint(
			uint(value), WORD_COUNT_INCREMENT, FLOAT_64_VALUE_MANTISSA_BIT_COUNT,
		).
		Ensure()
}

// Float_Binary_Source_Exponent excludes exponents handled before IEEE rounding.
type Float_Binary_Source_Exponent int

// Float_Binary_Source_Exponent_Invariants reserves overflow carry for the result.
func Float_Binary_Source_Exponent_Invariants(
	value Float_Binary_Source_Exponent, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), FLOAT_BINARY_EXPONENT_MINIMUM,
			FLOAT_64_VALUE_BIT_COUNT_MAXIMUM).
		Ensure()
}

// Float_Binary_Mantissa is one nonzero binary32-or-binary64 significand.
type Float_Binary_Mantissa Word

// Float_Binary_Mantissa_Invariants keeps reduction inside the wider IEEE significand.
func Float_Binary_Mantissa_Invariants(
	value Float_Binary_Mantissa, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(
			uint64(value), uint64(bits.CARRY_MAXIMUM),
			FLOAT_64_VALUE_MANTISSA_MAXIMUM,
		).
		Ensure()
}

// FLOAT_BINARY_EXPONENT_MINIMUM follows the first value above handled tie underflow.
const FLOAT_BINARY_EXPONENT_MINIMUM = FLOAT_64_SUBNORMAL_EXPONENT_MINIMUM +
	WORD_COUNT_INCREMENT

// FLOAT_BINARY_EXPONENT_MAXIMUM admits the one carry that rounds to infinity.
const FLOAT_BINARY_EXPONENT_MAXIMUM = FLOAT_64_VALUE_BIT_COUNT_MAXIMUM +
	WORD_COUNT_INCREMENT

// Float_Binary_Exponent is one exponent after IEEE rounding can carry once.
type Float_Binary_Exponent int

// Float_Binary_Exponent_Invariants keeps handled underflow and one overflow carry.
func Float_Binary_Exponent_Invariants(
	value Float_Binary_Exponent, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), FLOAT_BINARY_EXPONENT_MINIMUM,
			FLOAT_BINARY_EXPONENT_MAXIMUM,
		).
		Ensure()
}

// Float_Binary_Result is one scalar significand after IEEE nearest-even reduction.
type Float_Binary_Result struct {
	// Mantissa holds the normalized IEEE significand before field packing.
	Mantissa Float_Binary_Mantissa
	// Exponent holds the Float exponent paired with Mantissa.
	Exponent Float_Binary_Exponent
	// Accuracy reports which side of the source the packed value occupies.
	Accuracy Accuracy
}

// Float_Binary_Result_Invariants keeps the scalar finite result normalized.
func Float_Binary_Result_Invariants(
	value Float_Binary_Result, namespace aver.Namespace,
) {
	Float_Binary_Mantissa_Invariants(value.Mantissa, namespace)
	Float_Binary_Exponent_Invariants(value.Exponent, namespace)
	Accuracy_Invariants(value.Accuracy, namespace)
}

// Float_Square_Root_Exponent is the normalized exact root exponent before rounding.
type Float_Square_Root_Exponent int

// Float_Square_Root_Exponent_Invariants binds root scale to halved source bounds.
func Float_Square_Root_Exponent_Invariants(
	value Float_Square_Root_Exponent, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), FLOAT_SQUARE_ROOT_EXPONENT_MINIMUM,
			FLOAT_SQUARE_ROOT_EXPONENT_MAXIMUM,
		).
		Ensure()
}

// Float_Nonnegative_Finite narrows square-root input after sign validation.
type Float_Nonnegative_Finite Float

// Float_Nonnegative_Finite_Invariants retains every finite field except negative polarity.
func Float_Nonnegative_Finite_Invariants(
	value Float_Nonnegative_Finite, namespace aver.Namespace,
) {
	aver.Tree(Float_Active_Precision(value.Precision), namespace).
		Range_Uint(
			uint(value.Precision), WORD_COUNT_INCREMENT, FLOAT_PRECISION_MAXIMUM,
		).
		Ensure()
	aver.Tree(Rounding_Mode(value.Mode), namespace).
		Range_Uint8(
			uint8(value.Mode), uint8(ROUND_TO_NEAREST_EVEN),
			uint8(ROUND_TO_POSITIVE_INFINITY),
		).
		Ensure()
	aver.Tree(Accuracy(value.Accuracy), namespace).
		Enum_3_Int8(
			int8(value.Accuracy), int8(ACCURACY_BELOW), int8(ACCURACY_EXACT),
			int8(ACCURACY_ABOVE),
		).
		Ensure()
	aver.Tree(Float_Exponent(value.Exponent), namespace).
		Range_Int(
			int(value.Exponent), FLOAT_EXPONENT_MINIMUM, FLOAT_EXPONENT_MAXIMUM,
		).
		Ensure()
	Positive_Int_Invariants(Positive_Int(value.Mantissa), namespace)
	aver.Always(value.Form == FLOAT_FORM_FINITE,
		"A nonnegative finite helper value has finite form.")
	aver.Always(value.Negative == POLARITY_NONNEGATIVE,
		"A square-root source is nonnegative.")
}

// Float_Rat_Division_Workspace gives nested quotient memory one invariant identity.
type Float_Rat_Division_Workspace Float_Division_Workspace

// Float_Rat_Division_Workspace_Invariants fixes both quotient magnitudes.
func Float_Rat_Division_Workspace_Invariants(
	value Float_Rat_Division_Workspace, _ aver.Namespace,
) {
	aver.Always(
		len(value.Remainder) == FLOAT_DIVISION_WORD_COUNT_MAXIMUM,
		"Float rational quotient remainder retains its carry word.",
	)
	aver.Always(
		len(value.Divisor) == FLOAT_DIVISION_WORD_COUNT_MAXIMUM,
		"Float rational quotient divisor retains its aligned mantissa.",
	)
}

// Float_Rat_Integers retains full mantissa before rational component validation.
type Float_Rat_Integers struct {
	// Numerator must hold source precision even when conversion overflows.
	Numerator Int
	// Denominator grows only after component bounds have passed.
	Denominator Rat_Denominator
}

// Float_Rat_Integers_Invariants retains separate mantissa and denominator bounds.
func Float_Rat_Integers_Invariants(value Float_Rat_Integers, namespace aver.Namespace) {
	Int_Invariants(value.Numerator, namespace)
	Rat_Denominator_Invariants(value.Denominator, namespace)
}

// Float_Rat_Workspace owns rational operands, result, and quotient scratch.
type Float_Rat_Workspace struct {
	// Values keep operands and result alias-safe.
	Values Float_Rat_Values
	// Integers keep exact binary rational components.
	Integers Float_Rat_Integers
	// Division owns quotient extraction for rational-to-float conversion.
	Division Float_Rat_Division_Workspace
}

// Float_Rat_Workspace_Invariants binds all conversion storage to derived bounds.
func Float_Rat_Workspace_Invariants(
	value Float_Rat_Workspace, namespace aver.Namespace,
) {
	Float_Rat_Values_Invariants(value.Values, namespace)
	Float_Rat_Integers_Invariants(value.Integers, namespace)
	Float_Rat_Division_Workspace_Invariants(value.Division, namespace)
}

// Float_Square_Root_Root stores one guard-width restoring-sqrt result.
type Float_Square_Root_Root []Word

// Float_Square_Root_Root_Invariants fixes the precision-plus-guard capacity.
func Float_Square_Root_Root_Invariants(
	value Float_Square_Root_Root, _ aver.Namespace,
) {
	aver.Always(
		len(value) == FLOAT_SQUARE_ROOT_WORD_COUNT_MAXIMUM,
		"Float square-root result storage retains one precision guard bit.",
	)
}

// Float_Square_Root_Remainder stores restoring state beside the root identity.
type Float_Square_Root_Remainder []Word

// Float_Square_Root_Remainder_Invariants fixes the precision-plus-guard capacity.
func Float_Square_Root_Remainder_Invariants(
	value Float_Square_Root_Remainder, _ aver.Namespace,
) {
	aver.Always(
		len(value) == FLOAT_SQUARE_ROOT_WORD_COUNT_MAXIMUM,
		"Float square-root remainder storage retains one precision guard bit.",
	)
}

// Float_Square_Root_Candidate stores the current restoring subtraction magnitude.
type Float_Square_Root_Candidate []Word

// Float_Square_Root_Candidate_Invariants fixes the precision-plus-guard capacity.
func Float_Square_Root_Candidate_Invariants(
	value Float_Square_Root_Candidate, _ aver.Namespace,
) {
	aver.Always(
		len(value) == FLOAT_SQUARE_ROOT_WORD_COUNT_MAXIMUM,
		"Float square-root candidate storage retains one precision guard bit.",
	)
}

// Float_Square_Root_Workspace owns the restoring root and remainder.
type Float_Square_Root_Workspace struct {
	// Root accumulates one rounded result and its guard bit.
	Root Float_Square_Root_Root
	// Remainder preserves every discarded square-root digit.
	Remainder Float_Square_Root_Remainder
	// Candidate keeps restoring subtraction outside the root.
	Candidate Float_Square_Root_Candidate
}

// Float_Square_Root_Workspace_Invariants binds both magnitudes to one derived width.
func Float_Square_Root_Workspace_Invariants(
	value Float_Square_Root_Workspace, namespace aver.Namespace,
) {
	Float_Square_Root_Root_Invariants(value.Root, namespace)
	Float_Square_Root_Remainder_Invariants(value.Remainder, namespace)
	Float_Square_Root_Candidate_Invariants(value.Candidate, namespace)
}

// Float_Gob_Workspace keeps word alignment outside source Float.
type Float_Gob_Workspace struct {
	// Mantissa holds aligned wire magnitude or transactional decoded magnitude.
	Mantissas Float_Gob_Mantissas
}

// Float_Gob_Workspace_Invariants composes one bounded mantissa.
func Float_Gob_Workspace_Invariants(
	value Float_Gob_Workspace, namespace aver.Namespace,
) {
	Float_Gob_Mantissas_Invariants(value.Mantissas, namespace)
}

// Float_Gob_Encode_Into keeps stdlib wire alignment in caller workspace.
func Float_Gob_Encode_Into(
	destination Float_Gob_Encoding,
	value Float_Handle,
	workspace Float_Gob_Workspace_Handle,
) (mantissa_count Word_Count, status Destination_Status) {
	defer func() {
		Word_Count_Invariants(mantissa_count, "float_gob_encode_into.mantissa_count")
		Destination_Status_Invariants(status, "float_gob_encode_into.status")
	}()
	Float_Gob_Encoding_Invariants(destination, "float_gob_encode_into.destination")
	Float_Handle_Invariants(value, "float_gob_encode_into.value")
	Float_Gob_Workspace_Handle_Invariants(workspace, "float_gob_encode_into.workspace")
	required := FLOAT_GOB_HEADER_SIZE
	if value.Form == FLOAT_FORM_FINITE {
		mantissa_count = value.Mantissa.Count
		required = FLOAT_GOB_FINITE_PREFIX_SIZE +
			int(mantissa_count)*WORD_BYTE_COUNT
	}
	if len(destination) < required {
		return mantissa_count, STATUS_DESTINATION_TOO_SMALL
	}
	destination[FLOAT_GOB_VERSION_OFFSET] = byte(FLOAT_GOB_VERSION)
	attributes := byte(value.Mode)<<FLOAT_GOB_MODE_SHIFT |
		byte(int8(value.Accuracy)+int8(WORD_COUNT_INCREMENT))<<
			FLOAT_GOB_ACCURACY_SHIFT |
		byte(value.Form)<<FLOAT_GOB_FORM_SHIFT
	if value.Negative == POLARITY_NEGATIVE {
		attributes |= FLOAT_GOB_NEGATIVE_MASK
	}
	destination[FLOAT_GOB_ATTRIBUTES_OFFSET] = attributes
	precision_end_offset := FLOAT_GOB_PRECISION_OFFSET + FLOAT_GOB_FIELD_SIZE
	binary.Put_Uint_32(
		binary.Bytes(destination[FLOAT_GOB_PRECISION_OFFSET:precision_end_offset]),
		binary.Word_32(value.Precision), binary.BIG_ENDIAN,
	)
	if value.Form != FLOAT_FORM_FINITE {
		return mantissa_count, STATUS_OK
	}
	exponent_end_offset := FLOAT_GOB_EXPONENT_OFFSET + FLOAT_GOB_FIELD_SIZE
	binary.Put_Uint_32(
		binary.Bytes(destination[FLOAT_GOB_EXPONENT_OFFSET:exponent_end_offset]),
		binary.Word_32(uint32(int32(value.Exponent))), binary.BIG_ENDIAN,
	)
	if value.Mantissa.Count == Word_Count(BASE_BINARY) {
		if float_gob_encode_exact_double_word(
			Double_Word_Encoding(destination[FLOAT_GOB_MANTISSA_OFFSET:required]),
			Double_Words{
				Low:  value.Mantissa.Words[WORD_COUNT_MINIMUM],
				High: Float_Active_Word(value.Mantissa.Words[WORD_COUNT_INCREMENT]),
			}, value.Exponent) {
			return mantissa_count, STATUS_OK
		}
	}
	mantissa := &workspace.Mantissas.Value
	Int_Set(mantissa, (*Int)(&value.Mantissa))
	bit_count := int(Int_Bit_Count(mantissa))
	aligned_bit_count := int(mantissa_count) * WORD_BIT_COUNT
	shift := Shift_Count(aligned_bit_count - bit_count)
	shift_status := Int_Shift_Left(mantissa, mantissa, shift)
	aver.Always(
		shift_status == Arithmetic_Status(STATUS_OK),
		"Word alignment cannot exceed existing mantissa word count.",
	)
	fill_status := Int_Fill_Bytes(
		Bytes(destination[FLOAT_GOB_MANTISSA_OFFSET:required]),
		mantissa,
	)
	aver.Always(
		fill_status == Destination_Status(STATUS_OK),
		"Derived wire mantissa storage has exact capacity.",
	)
	return mantissa_count, STATUS_OK
}

// Float_Gob_Decode validates complete wire state before destination mutation.
func Float_Gob_Decode(
	destination Float_Handle,
	source Float_Gob_Encoding_Unvalidated,
	workspace Float_Gob_Workspace_Handle,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "float_gob_decode.status") }()
	Float_Handle_Invariants(destination, "float_gob_decode.destination_initial")
	Float_Gob_Encoding_Unvalidated_Invariants(source, "float_gob_decode.source")
	Float_Gob_Workspace_Handle_Invariants(workspace, "float_gob_decode.workspace")
	if len(source) > FLOAT_GOB_SIZE_MAXIMUM {
		return STATUS_INPUT_INVALID
	}
	if len(source) == WORD_COUNT_MINIMUM {
		// Empty encoding resets value while retaining caller-owned storage.
		words := destination.Mantissa.Words
		clear(words)
		*destination = Float{Mantissa: Float_Mantissa{Words: words}}
		return STATUS_OK
	}
	if len(source) < FLOAT_GOB_HEADER_SIZE {
		return STATUS_INPUT_INVALID
	}
	if source[FLOAT_GOB_VERSION_OFFSET] != byte(FLOAT_GOB_VERSION) {
		return STATUS_INPUT_INVALID
	}
	attributes := source[FLOAT_GOB_ATTRIBUTES_OFFSET]
	form := Float_Form(attributes >> FLOAT_GOB_FORM_SHIFT & FLOAT_GOB_FORM_MASK)
	if form == FLOAT_FORM_FINITE {
		if len(source) < FLOAT_GOB_FINITE_ENCODING_SIZE_MINIMUM {
			return STATUS_INPUT_INVALID
		}
		mantissa_size := Float_Gob_Mantissa_Size(len(source) - FLOAT_GOB_MANTISSA_OFFSET)
		if int(mantissa_size)%WORD_BYTE_COUNT != WORD_COUNT_MINIMUM {
			return STATUS_INPUT_INVALID
		}
		high_bit_shift := bits.BIT_COUNT_8_MAXIMUM - WORD_COUNT_INCREMENT
		if source[FLOAT_GOB_MANTISSA_OFFSET]>>high_bit_shift != byte(bits.CARRY_MAXIMUM) {
			return STATUS_INPUT_INVALID
		}
		return float_gob_decode_finite_encoding(
			destination, Float_Gob_Finite_Encoding(source), workspace,
			Float_Gob_Aligned_Mantissa_Size(mantissa_size),
		)
	}
	encoded_header := Float_Gob_Header_Encoding{
		Version: Float_Gob_Version(source[FLOAT_GOB_VERSION_OFFSET]),
		Mode:    Float_Gob_Mode(attributes >> FLOAT_GOB_MODE_SHIFT & FLOAT_GOB_MODE_MASK),
		Accuracy: Float_Gob_Accuracy(attributes >> FLOAT_GOB_ACCURACY_SHIFT &
			FLOAT_GOB_ACCURACY_MASK),
		Form:            Float_Gob_Nonfinite_Form(form),
		Negative:        Polarity(attributes & FLOAT_GOB_NEGATIVE_MASK),
		Precision_High:  Float_Gob_Precision_High(source[2]),
		Precision_Upper: Float_Gob_Precision_Upper(source[3]),
		Precision_Lower: Float_Gob_Precision_Lower(source[4]),
		Precision_Low:   Float_Gob_Precision_Low(source[5]),
	}
	header, validation := float_gob_decode_header(encoded_header)
	if validation != STATUS_OK {
		return STATUS_INPUT_INVALID
	}
	float_gob_decode_nonfinite_commit(
		destination, header,
	)
	return STATUS_OK
}

func float_gob_decode_finite_encoding(
	destination Float_Handle,
	source Float_Gob_Finite_Encoding,
	workspace Float_Gob_Workspace_Handle,
	mantissa_size Float_Gob_Aligned_Mantissa_Size,
) (status Validation_Status) {
	defer func() {
		Validation_Status_Invariants(status, "float_gob_decode_finite_encoding.status")
	}()
	Float_Handle_Invariants(destination, "float_gob_decode_finite_encoding.destination_initial")
	Float_Gob_Finite_Encoding_Invariants(
		source, "float_gob_decode_finite_encoding.source",
	)
	Float_Gob_Workspace_Handle_Invariants(
		workspace, "float_gob_decode_finite_encoding.workspace",
	)
	Float_Gob_Aligned_Mantissa_Size_Invariants(
		mantissa_size, "float_gob_decode_finite_encoding.size",
	)
	header, exponent, header_status := float_gob_decode_finite_header(source)
	if header_status != STATUS_OK {
		return STATUS_INPUT_INVALID
	}
	mantissa_count := int(mantissa_size) / WORD_BYTE_COUNT
	precision := Float_Active_Precision(header.Precision)
	aligned_bit_count := Bit_Count(mantissa_count * WORD_BIT_COUNT)
	if destination.Precision != header.Precision {
		return float_gob_decode_finite_general(
			destination, source, (*Nonempty_Int)(&workspace.Mantissas.Value),
			header, exponent, mantissa_size,
		)
	}
	if aligned_bit_count > Bit_Count(precision) {
		return float_gob_decode_finite_general(
			destination, source, (*Nonempty_Int)(&workspace.Mantissas.Value),
			header, exponent, mantissa_size,
		)
	}
	previous_count := destination.Mantissa.Count
	for index := WORD_COUNT_MINIMUM; index < mantissa_count; index++ {
		source_end := len(source) - index*WORD_BYTE_COUNT
		source_start := source_end - WORD_BYTE_COUNT
		word := Word(0)
		for source_index := source_start; source_index < source_end; source_index++ {
			word = word<<bits.BIT_COUNT_8_MAXIMUM | Word(source[source_index])
		}
		destination.Mantissa.Words[index] = word
	}
	destination.Accuracy, destination.Form = ACCURACY_EXACT, FLOAT_FORM_FINITE
	destination.Negative, destination.Exponent = header.Negative, exponent
	destination.Mantissa.Count = Word_Count(mantissa_count)
	destination.Mantissa.Negative = POLARITY_NONNEGATIVE
	int_clear((*Int)(&destination.Mantissa), destination.Mantissa.Count, previous_count)
	return STATUS_OK
}

func float_gob_decode_finite_general(
	destination Float_Handle,
	source Float_Gob_Finite_Encoding,
	mantissa_reference Nonempty_Int_Handle,
	header Float_Gob_Finite_Header,
	exponent Float_Exponent,
	mantissa_size Float_Gob_Aligned_Mantissa_Size,
) (status Validation_Status) {
	defer func() {
		Validation_Status_Invariants(status, "float_gob_decode_finite_general.status")
	}()
	Float_Handle_Invariants(destination, "float_gob_decode_finite_general.destination_initial")
	Float_Gob_Finite_Encoding_Invariants(source, "float_gob_decode_finite_general.source")
	Nonempty_Int_Handle_Invariants(
		mantissa_reference, "float_gob_decode_finite_general.workspace",
	)
	Float_Gob_Finite_Header_Invariants(header, "float_gob_decode_finite_general.header")
	Float_Exponent_Invariants(exponent, "float_gob_decode_finite_general.exponent")
	Float_Gob_Aligned_Mantissa_Size_Invariants(
		mantissa_size, "float_gob_decode_finite_general.size",
	)
	mantissa := (*Int)((*Nonempty_Int)(mantissa_reference))
	mantissa_count := int(mantissa_size) / WORD_BYTE_COUNT
	previous_count := mantissa.Count
	for index := WORD_COUNT_MINIMUM; index < mantissa_count; index++ {
		source_end := len(source) - index*WORD_BYTE_COUNT
		source_start := source_end - WORD_BYTE_COUNT
		word := Word(0)
		for source_index := source_start; source_index < source_end; source_index++ {
			word = word<<bits.BIT_COUNT_8_MAXIMUM | Word(source[source_index])
		}
		mantissa.Words[index] = word
	}
	mantissa.Count = Word_Count(mantissa_count)
	mantissa.Negative = POLARITY_NONNEGATIVE
	int_clear(mantissa, mantissa.Count, previous_count)
	precision := Float_Active_Precision(header.Precision)
	status = float_gob_decode_finite((*Positive_Int)(mantissa), precision)
	if status != STATUS_OK {
		return status
	}
	float_gob_decode_finite_commit(
		destination, header, exponent, (*Positive_Int)(mantissa),
	)
	return STATUS_OK
}

func float_gob_decode_finite_header(
	source Float_Gob_Finite_Encoding,
) (header Float_Gob_Finite_Header, exponent Float_Exponent, status Validation_Status) {
	defer func() {
		Float_Gob_Finite_Header_Invariants(header, "float_gob_decode_finite_header.header")
		Float_Exponent_Invariants(exponent, "float_gob_decode_finite_header.exponent")
		Validation_Status_Invariants(status, "float_gob_decode_finite_header.status")
	}()
	Float_Gob_Finite_Encoding_Invariants(source, "float_gob_decode_finite_header.source")
	header.Precision = Float_Precision(WORD_COUNT_INCREMENT)
	header.Form = FLOAT_FORM_FINITE
	attributes := source[FLOAT_GOB_ATTRIBUTES_OFFSET]
	mode := Rounding_Mode(attributes >> FLOAT_GOB_MODE_SHIFT & FLOAT_GOB_MODE_MASK)
	if mode > Rounding_Mode(ROUND_TO_POSITIVE_INFINITY) {
		return header, exponent, STATUS_INPUT_INVALID
	}
	accuracy := Accuracy(int8(attributes>>FLOAT_GOB_ACCURACY_SHIFT&
		FLOAT_GOB_ACCURACY_MASK) - int8(WORD_COUNT_INCREMENT))
	if accuracy < ACCURACY_BELOW {
		return header, exponent, STATUS_INPUT_INVALID
	}
	if accuracy > ACCURACY_ABOVE {
		return header, exponent, STATUS_INPUT_INVALID
	}
	precision_end_offset := FLOAT_GOB_PRECISION_OFFSET + FLOAT_GOB_FIELD_SIZE
	precision_value := binary.Word_32(0)
	for index := FLOAT_GOB_PRECISION_OFFSET; index < precision_end_offset; index++ {
		precision_value = precision_value<<bits.BIT_COUNT_8_MAXIMUM |
			binary.Word_32(source[index])
	}
	if precision_value == binary.Word_32(FLOAT_PRECISION_MINIMUM) {
		return header, exponent, STATUS_INPUT_INVALID
	}
	if precision_value > binary.Word_32(FLOAT_PRECISION_MAXIMUM) {
		return header, exponent, STATUS_INPUT_INVALID
	}
	exponent_end_offset := FLOAT_GOB_EXPONENT_OFFSET + FLOAT_GOB_FIELD_SIZE
	encoded_exponent := binary.Word_32(0)
	for index := FLOAT_GOB_EXPONENT_OFFSET; index < exponent_end_offset; index++ {
		encoded_exponent = encoded_exponent<<bits.BIT_COUNT_8_MAXIMUM |
			binary.Word_32(source[index])
	}
	exponent_value := int32(encoded_exponent)
	if int(exponent_value) < FLOAT_EXPONENT_MINIMUM {
		return header, exponent, STATUS_INPUT_INVALID
	}
	if int(exponent_value) > FLOAT_EXPONENT_MAXIMUM {
		return header, exponent, STATUS_INPUT_INVALID
	}
	header = Float_Gob_Finite_Header{
		Precision: Float_Precision(precision_value), Mode: mode, Accuracy: accuracy,
		Form: FLOAT_FORM_FINITE, Negative: Polarity(attributes & FLOAT_GOB_NEGATIVE_MASK),
	}
	return header, Float_Exponent(exponent_value), STATUS_OK
}

func float_gob_decode_header(
	encoded Float_Gob_Header_Encoding,
) (header Float_Gob_Nonfinite_Header, status Validation_Status) {
	defer func() {
		Float_Gob_Nonfinite_Header_Invariants(header, "float_gob_decode_header.header")
		Validation_Status_Invariants(status, "float_gob_decode_header.status")
	}()
	Float_Gob_Header_Encoding_Invariants(
		encoded, "float_gob_decode_header.encoded",
	)
	mode_value := byte(encoded.Mode)
	if mode_value > byte(ROUNDING_MODE_UNVALIDATED_MAXIMUM) {
		return Float_Gob_Nonfinite_Header{}, STATUS_INPUT_INVALID
	}
	mode, validation := Rounding_Mode_Validate(Rounding_Mode_Unvalidated(mode_value))
	if validation != STATUS_OK {
		return Float_Gob_Nonfinite_Header{}, STATUS_INPUT_INVALID
	}
	accuracy_value := int8(encoded.Accuracy) - int8(WORD_COUNT_INCREMENT)
	if accuracy_value < int8(ACCURACY_BELOW) {
		return Float_Gob_Nonfinite_Header{}, STATUS_INPUT_INVALID
	}
	if accuracy_value > int8(ACCURACY_ABOVE) {
		return Float_Gob_Nonfinite_Header{}, STATUS_INPUT_INVALID
	}
	form := Float_Form(encoded.Form)
	if form > FLOAT_FORM_INFINITY {
		return Float_Gob_Nonfinite_Header{}, STATUS_INPUT_INVALID
	}
	precision_bytes := [FLOAT_GOB_FIELD_SIZE]byte{
		byte(encoded.Precision_High), byte(encoded.Precision_Upper),
		byte(encoded.Precision_Lower), byte(encoded.Precision_Low),
	}
	precision_value := binary.Uint_32(
		binary.Bytes(precision_bytes[:]),
		binary.BIG_ENDIAN,
	)
	if uint32(precision_value) > uint32(FLOAT_PRECISION_UNVALIDATED_MAXIMUM) {
		return Float_Gob_Nonfinite_Header{}, STATUS_INPUT_INVALID
	}
	precision, validation := Float_Precision_Validate(
		Float_Precision_Unvalidated(precision_value),
	)
	if validation != STATUS_OK {
		return Float_Gob_Nonfinite_Header{}, STATUS_INPUT_INVALID
	}
	return Float_Gob_Nonfinite_Header{
		Precision: precision,
		Mode:      mode,
		Accuracy:  Accuracy(accuracy_value),
		Form:      form,
		Negative:  encoded.Negative,
	}, STATUS_OK
}

func float_gob_decode_nonfinite_commit(
	destination Float_Handle, header Float_Gob_Nonfinite_Header,
) {
	Float_Handle_Invariants(destination, "float_gob_nonfinite_commit.destination_initial")
	Float_Gob_Nonfinite_Header_Invariants(
		header, "float_gob_decode_nonfinite_commit.header",
	)
	precision, mode, accuracy := header.Precision, header.Mode, header.Accuracy
	if destination.Precision != FLOAT_PRECISION_MINIMUM {
		precision, mode, accuracy = destination.Precision, destination.Mode, ACCURACY_EXACT
	}
	previous_count := destination.Mantissa.Count
	destination.Precision, destination.Mode = precision, mode
	destination.Accuracy, destination.Form = accuracy, header.Form
	destination.Negative, destination.Exponent = header.Negative, FLOAT_EXPONENT_ZERO
	int_zero((*Int)(&destination.Mantissa), previous_count)
}

func float_gob_decode_finite_commit(
	destination Float_Handle,
	header Float_Gob_Finite_Header,
	exponent Float_Exponent,
	mantissa Positive_Int_Handle,
) {
	Float_Handle_Invariants(destination, "float_gob_decode_finite_commit.destination_initial")
	Float_Gob_Finite_Header_Invariants(
		header, "float_gob_decode_finite_commit.header",
	)
	Float_Exponent_Invariants(exponent, "float_gob_decode_finite_commit.exponent")
	Positive_Int_Handle_Invariants(
		mantissa, "float_gob_decode_finite_commit.mantissa",
	)
	precision, mode, accuracy := header.Precision, header.Mode, header.Accuracy
	if destination.Precision != FLOAT_PRECISION_MINIMUM {
		precision, mode, accuracy = destination.Precision, destination.Mode, ACCURACY_EXACT
		integer := (*Int)((*Positive_Int)(mantissa))
		if Float_Precision(Int_Bit_Count(integer)) > precision {
			result := Float{
				Precision: header.Precision, Mode: mode, Accuracy: header.Accuracy,
				Form: FLOAT_FORM_FINITE, Negative: header.Negative,
				Mantissa: Float_Mantissa(*mantissa), Exponent: exponent,
			}
			var result_words [WORD_COUNT_MAXIMUM]Word
			result.Mantissa.Words = result_words[:]
			copy(
				result.Mantissa.Words[:mantissa.Count],
				mantissa.Words[:mantissa.Count],
			)
			validation := Float_Set_Precision(
				&result, Float_Precision_Unvalidated(precision),
			)
			aver.Always(
				validation == Validation_Status(STATUS_OK),
				"Decoded precision came from validated receiver policy.",
			)
			float_copy_exact(destination, &result)
			return
		}
	}
	previous_count := destination.Mantissa.Count
	destination.Precision, destination.Mode = precision, mode
	destination.Accuracy, destination.Form = accuracy, FLOAT_FORM_FINITE
	destination.Negative, destination.Exponent = header.Negative, exponent
	for index := Word_Count(WORD_COUNT_MINIMUM); index < mantissa.Count; index++ {
		destination.Mantissa.Words[index] = mantissa.Words[index]
	}
	destination.Mantissa.Count = mantissa.Count
	destination.Mantissa.Negative = POLARITY_NONNEGATIVE
	if previous_count > mantissa.Count {
		int_clear((*Int)(&destination.Mantissa), mantissa.Count, previous_count)
	}
}

func float_gob_decode_finite(
	mantissa Positive_Int_Handle, precision Float_Active_Precision,
) (status Validation_Status) {
	defer func() {
		Positive_Int_Handle_Invariants(
			mantissa, "float_gob_decode_finite.mantissa_result",
		)
		Validation_Status_Invariants(status, "float_gob_decode_finite.status")
	}()
	Positive_Int_Handle_Invariants(mantissa, "float_gob_decode_finite.mantissa")
	Float_Active_Precision_Invariants(precision, "float_gob_decode_finite.precision")
	integer := (*Int)((*Positive_Int)(mantissa))
	aligned_bit_count := Bit_Count(int(integer.Count) * WORD_BIT_COUNT)
	if aligned_bit_count <= Bit_Count(precision) {
		return STATUS_OK
	}
	zero_count := Int_Trailing_Zero_Bit_Count(integer)
	Int_Shift_Right(integer, integer, Shift_Count(zero_count))
	if Int_Bit_Count(integer) > Bit_Count(precision) {
		return STATUS_INPUT_INVALID
	}
	return STATUS_OK
}

// Int_Set_Float_64_Bits truncates finite binary64 toward zero.
func Int_Set_Float_64_Bits(
	destination Int_Handle, encoding Float_64_Bits,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "int_set_float_64_bits.status") }()
	Int_Handle_Invariants(destination, "int_set_float_64_bits.destination")
	Float_64_Bits_Invariants(encoding, "int_set_float_64_bits.encoding")
	encoded := uint64(encoding)
	exponent_field := encoded >> FLOAT_64_EXPONENT_SHIFT & FLOAT_64_EXPONENT_MASK
	if exponent_field == FLOAT_64_EXPONENT_MASK {
		return STATUS_INPUT_INVALID
	}
	if exponent_field == bits.WORD_64_MINIMUM {
		int_zero(destination, destination.Count)
		return STATUS_OK
	}
	exponent := int(exponent_field) - FLOAT_64_EXPONENT_BIAS
	if exponent < WORD_COUNT_MINIMUM {
		int_zero(destination, destination.Count)
		return STATUS_OK
	}
	mantissa := encoded&FLOAT_64_MANTISSA_MASK | FLOAT_64_HIDDEN_MANTISSA_BIT
	Int_Set_Uint_64(destination, Word_64(mantissa))
	shift := exponent - FLOAT_64_MANTISSA_BIT_COUNT
	if shift < WORD_COUNT_MINIMUM {
		Int_Shift_Right(destination, destination, Shift_Count(-shift))
	} else if shift > WORD_COUNT_MINIMUM {
		shift_status := Int_Shift_Left(destination, destination, Shift_Count(shift))
		aver.Always(
			shift_status == Arithmetic_Status(STATUS_OK),
			"Finite binary64 integer magnitude fits bounded Int storage.",
		)
	}
	if encoded&FLOAT_64_SIGN_MASK != bits.WORD_64_MINIMUM {
		if destination.Count != WORD_COUNT_MINIMUM {
			destination.Negative = POLARITY_NEGATIVE
		}
	}
	return STATUS_OK
}

// Int_Float_64_Bits returns nearest binary64 using round-to-even.
func Int_Float_64_Bits(
	value Int_Handle,
) (encoding Int_Float_64_Value_Bits, accuracy Accuracy) {
	defer func() {
		Int_Float_64_Value_Bits_Invariants(encoding, "int_float_64_bits.encoding")
		Accuracy_Invariants(accuracy, "int_float_64_bits.accuracy")
	}()
	Int_Handle_Invariants(value, "int_float_64_bits.value")
	// Conversion scratch never escapes through scalar encoding result.
	var storage [WORD_COUNT_MAXIMUM]Word
	var number Float
	number.Mantissa.Words = storage[:]
	Float_Set_Int(&number, value)
	result, accuracy := Float_Float_64_Bits(&number)
	return Int_Float_64_Value_Bits(result), accuracy
}

// Int_Modular_Square_Root uses caller-supplied nonresidue to keep search bounded.
func Int_Modular_Square_Root(
	destination Int_Handle,
	value Int_Handle,
	modulus Int_Handle,
	nonresidue Int_Handle,
	workspace Int_Modular_Square_Root_Workspace_Handle,
) (status Modular_Square_Root_Status) {
	defer func() {
		Modular_Square_Root_Status_Invariants(
			status, "int_modular_square_root.status",
		)
	}()
	Int_Handle_Invariants(destination, "int_modular_square_root.destination")
	Int_Handle_Invariants(value, "int_modular_square_root.value")
	Int_Handle_Invariants(modulus, "int_modular_square_root.modulus")
	Int_Handle_Invariants(nonresidue, "int_modular_square_root.nonresidue")
	Int_Modular_Square_Root_Workspace_Handle_Invariants(
		workspace, "int_modular_square_root.workspace",
	)
	if !int_modular_square_root_modulus_valid(modulus) {
		return STATUS_INPUT_INVALID
	}
	integers := workspace.Integers
	modulus_copy := &integers[MODULAR_SQUARE_ROOT_MODULUS_INDEX]
	nonresidue_copy := &integers[MODULAR_SQUARE_ROOT_NONRESIDUE_INDEX]
	Int_Set(modulus_copy, modulus)
	Int_Set(nonresidue_copy, nonresidue)
	initial_quotient_count := workspace.Modular.Division.Quotient_Count
	initial_remainder_count := workspace.Modular.Division.Remainder_Count
	symbol, validation := int_jacobi(
		nonresidue_copy, modulus_copy, &workspace.Jacobi, &workspace.Modular.Division,
	)
	workspace.Modular.Division.Quotient_Count = initial_quotient_count
	workspace.Modular.Division.Remainder_Count = initial_remainder_count
	if validation != Validation_Status(STATUS_OK) {
		return STATUS_INPUT_INVALID
	}
	if symbol != JACOBI_SYMBOL_NEGATIVE {
		return STATUS_INPUT_INVALID
	}
	return Modular_Square_Root_Status(
		int_modular_square_root_reduce(
			destination, value, (*Validated_Square_Root_Workspace)(
				(*Int_Modular_Square_Root_Workspace)(workspace))),
	)
}

func int_modular_square_root_modulus_valid(modulus Int_Handle) (valid Boolean) {
	defer func() {
		Boolean_Invariants(valid, "int_modular_square_root_modulus_valid.valid")
	}()
	Int_Handle_Invariants(modulus, "int_modular_square_root_modulus_valid.modulus")
	if modulus.Negative == POLARITY_NEGATIVE {
		return false
	}
	if modulus.Count != WORD_COUNT_INCREMENT {
		if modulus.Count == WORD_COUNT_MINIMUM {
			return false
		}
		return Boolean(modulus.Words[WORD_COUNT_MINIMUM]&JACOBI_PARITY_MASK != 0)
	}
	return Boolean(
		modulus.Words[WORD_COUNT_MINIMUM] > Word(WORD_COUNT_INCREMENT) &&
			modulus.Words[WORD_COUNT_MINIMUM]&JACOBI_PARITY_MASK != 0,
	)
}

func int_modular_square_root_reduce(
	destination Int_Handle, value Int_Handle,
	workspace Validated_Square_Root_Workspace_Handle,
) (status Modular_Square_Root_Result_Status) {
	defer func() {
		Modular_Square_Root_Result_Status_Invariants(
			status, "int_modular_square_root_reduce.status",
		)
	}()
	Int_Handle_Invariants(destination, "int_modular_square_root_reduce.destination")
	Int_Handle_Invariants(value, "int_modular_square_root_reduce.value")
	Validated_Square_Root_Workspace_Handle_Invariants(
		workspace, "int_modular_square_root_reduce.workspace",
	)
	integers := workspace.Integers
	modulus := &integers[MODULAR_SQUARE_ROOT_MODULUS_INDEX]
	residue := &integers[MODULAR_SQUARE_ROOT_RESIDUE_INDEX]
	one := &integers[MODULAR_SQUARE_ROOT_ONE_INDEX]
	Int_Set_Uint_64(one, Word_64(WORD_COUNT_INCREMENT))
	initial_quotient_count := workspace.Modular.Division.Quotient_Count
	initial_remainder_count := workspace.Modular.Division.Remainder_Count
	reduction := Int_Modular_Multiply(
		residue, value, one, modulus,
		(*Int_Modular_Workspace)(&workspace.Modular),
	)
	workspace.Modular.Division.Quotient_Count = initial_quotient_count
	workspace.Modular.Division.Remainder_Count = initial_remainder_count
	aver.Always(
		reduction == Divisor_Status(STATUS_OK),
		"Modular square-root reduction uses validated nonzero modulus.",
	)
	symbol, validation := int_jacobi(
		residue, modulus, &workspace.Jacobi, &workspace.Modular.Division,
	)
	workspace.Modular.Division.Quotient_Count = initial_quotient_count
	workspace.Modular.Division.Remainder_Count = initial_remainder_count
	aver.Always(
		validation == Validation_Status(STATUS_OK),
		"Modular square-root residue uses validated positive odd modulus.",
	)
	if symbol == JACOBI_SYMBOL_NEGATIVE {
		return STATUS_RESULT_ABSENT
	}
	if symbol == JACOBI_SYMBOL_ZERO {
		int_zero(destination, destination.Count)
		return STATUS_OK
	}
	return int_modular_square_root_start(
		destination, workspace.Integers, (*Int_Modular_Workspace)(&workspace.Modular),
	)
}

func int_modular_square_root_start(
	destination Int_Handle, integers Modular_Square_Root_Integers,
	modular Int_Modular_Workspace_Handle,
) (status Modular_Square_Root_Result_Status) {
	defer func() {
		Modular_Square_Root_Result_Status_Invariants(
			status, "int_modular_square_root_start.status",
		)
	}()
	Int_Handle_Invariants(destination, "int_modular_square_root_start.destination")
	Modular_Square_Root_Integers_Invariants(
		integers, "int_modular_square_root_start.integers",
	)
	Int_Modular_Workspace_Handle_Invariants(
		modular, "int_modular_square_root_start.modular",
	)
	modulus := &integers[MODULAR_SQUARE_ROOT_MODULUS_INDEX]
	odd_factor := &integers[MODULAR_SQUARE_ROOT_ODD_FACTOR_INDEX]
	exponent := &integers[MODULAR_SQUARE_ROOT_EXPONENT_INDEX]
	one := &integers[MODULAR_SQUARE_ROOT_ONE_INDEX]
	subtraction := Int_Subtract(odd_factor, modulus, one)
	aver.Always(
		subtraction == Arithmetic_Status(STATUS_OK),
		"Validated modular square-root modulus exceeds one.",
	)
	order := Int_Trailing_Zero_Bit_Count(odd_factor)
	Int_Shift_Right(odd_factor, odd_factor, Shift_Count(order))
	addition := Int_Add(exponent, odd_factor, one)
	aver.Always(
		addition == Arithmetic_Status(STATUS_OK),
		"Odd factor plus one cannot exceed validated modulus.",
	)
	Int_Shift_Right(exponent, exponent, Shift_Count(WORD_COUNT_INCREMENT))
	if order == Trailing_Zero_Bit_Count(WORD_COUNT_INCREMENT) {
		return int_modular_square_root_order_one(destination, integers, modular)
	}
	int_modular_square_root_powers(integers, modular)
	return int_modular_square_root_iterate(destination, integers, modular)
}

func int_modular_square_root_order_one(
	destination Int_Handle, integers Modular_Square_Root_Integers,
	modular Int_Modular_Workspace_Handle,
) (status Modular_Square_Root_Result_Status) {
	defer func() {
		Modular_Square_Root_Result_Status_Invariants(
			status, "int_modular_square_root_order_one.status",
		)
	}()
	Int_Handle_Invariants(destination, "int_modular_square_root_order_one.destination")
	Modular_Square_Root_Integers_Invariants(
		integers, "int_modular_square_root_order_one.integers",
	)
	Int_Modular_Workspace_Handle_Invariants(
		modular, "int_modular_square_root_order_one.modular",
	)
	modulus := &integers[MODULAR_SQUARE_ROOT_MODULUS_INDEX]
	residue := &integers[MODULAR_SQUARE_ROOT_RESIDUE_INDEX]
	exponent := &integers[MODULAR_SQUARE_ROOT_EXPONENT_INDEX]
	result := &integers[MODULAR_SQUARE_ROOT_RESULT_INDEX]
	initial_quotient_count := modular.Division.Quotient_Count
	initial_remainder_count := modular.Division.Remainder_Count
	exponentiation := Int_Modular_Exponent(result, residue, exponent, modulus, modular)
	modular.Division.Quotient_Count = initial_quotient_count
	modular.Division.Remainder_Count = initial_remainder_count
	aver.Always(
		exponentiation == Modular_Status(STATUS_OK),
		"A positive root exponent and nonzero modulus make exponentiation defined.",
	)
	return int_modular_square_root_commit(destination, integers, modular)
}

func int_modular_square_root_powers(
	integers Modular_Square_Root_Integers, modular Int_Modular_Workspace_Handle,
) {
	Modular_Square_Root_Integers_Invariants(
		integers, "int_modular_square_root_powers.integers",
	)
	Int_Modular_Workspace_Handle_Invariants(
		modular, "int_modular_square_root_powers.modular",
	)
	modulus := &integers[MODULAR_SQUARE_ROOT_MODULUS_INDEX]
	residue := &integers[MODULAR_SQUARE_ROOT_RESIDUE_INDEX]
	nonresidue := &integers[MODULAR_SQUARE_ROOT_NONRESIDUE_INDEX]
	odd_factor := &integers[MODULAR_SQUARE_ROOT_ODD_FACTOR_INDEX]
	exponent := &integers[MODULAR_SQUARE_ROOT_EXPONENT_INDEX]
	result := &integers[MODULAR_SQUARE_ROOT_RESULT_INDEX]
	power := &integers[MODULAR_SQUARE_ROOT_POWER_INDEX]
	generator := &integers[MODULAR_SQUARE_ROOT_GENERATOR_INDEX]
	initial_quotient_count := modular.Division.Quotient_Count
	initial_remainder_count := modular.Division.Remainder_Count
	status := Int_Modular_Exponent(result, residue, exponent, modulus, modular)
	aver.Always(
		status == Modular_Status(STATUS_OK),
		"Tonelli-Shanks root exponent is nonnegative and modulus is nonzero.",
	)
	status = Int_Modular_Exponent(power, residue, odd_factor, modulus, modular)
	aver.Always(
		status == Modular_Status(STATUS_OK),
		"Tonelli-Shanks power exponent is nonnegative and modulus is nonzero.",
	)
	status = Int_Modular_Exponent(generator, nonresidue, odd_factor, modulus, modular)
	modular.Division.Quotient_Count = initial_quotient_count
	modular.Division.Remainder_Count = initial_remainder_count
	aver.Always(
		status == Modular_Status(STATUS_OK),
		"Tonelli-Shanks generator exponent is nonnegative and modulus is nonzero.",
	)
}

func int_modular_square_root_square_factor(
	factor_storage Positive_Magnitude_Handle, modulus_storage Positive_Magnitude_Handle,
	modular Int_Modular_Workspace_Handle,
) {
	Positive_Magnitude_Handle_Invariants(
		factor_storage, "int_modular_square_root_square_factor.factor")
	Positive_Magnitude_Handle_Invariants(
		modulus_storage, "int_modular_square_root_square_factor.modulus")
	Int_Modular_Workspace_Handle_Invariants(
		modular, "int_modular_square_root_square_factor.modular",
	)
	initial_quotient_count := modular.Division.Quotient_Count
	initial_remainder_count := modular.Division.Remainder_Count
	factor := (*Int)((*Positive_Magnitude)(factor_storage))
	modulus := (*Int)((*Positive_Magnitude)(modulus_storage))
	status := Int_Modular_Multiply(factor, factor, factor, modulus, modular)
	modular.Division.Quotient_Count = initial_quotient_count
	modular.Division.Remainder_Count = initial_remainder_count
	aver.Always(
		status == Divisor_Status(STATUS_OK),
		"Tonelli-Shanks factor square uses nonzero modulus.",
	)
}

func int_modular_square_root_advance(
	integers Modular_Square_Root_Integers, modular Int_Modular_Workspace_Handle,
) {
	Modular_Square_Root_Integers_Invariants(
		integers, "int_modular_square_root_advance.integers",
	)
	Int_Modular_Workspace_Handle_Invariants(
		modular, "int_modular_square_root_advance.modular",
	)
	modulus := &integers[MODULAR_SQUARE_ROOT_MODULUS_INDEX]
	exponent := &integers[MODULAR_SQUARE_ROOT_EXPONENT_INDEX]
	result := &integers[MODULAR_SQUARE_ROOT_RESULT_INDEX]
	power := &integers[MODULAR_SQUARE_ROOT_POWER_INDEX]
	generator := &integers[MODULAR_SQUARE_ROOT_GENERATOR_INDEX]
	factor := &integers[MODULAR_SQUARE_ROOT_FACTOR_INDEX]
	initial_quotient_count := modular.Division.Quotient_Count
	initial_remainder_count := modular.Division.Remainder_Count
	status := Int_Modular_Exponent(factor, generator, exponent, modulus, modular)
	aver.Always(
		status == Modular_Status(STATUS_OK),
		"Tonelli-Shanks factor exponent uses nonnegative exponent and nonzero modulus.",
	)
	product_status := Int_Modular_Multiply(generator, factor, factor, modulus, modular)
	aver.Always(product_status == Divisor_Status(STATUS_OK),
		"Tonelli-Shanks generator square uses nonzero modulus.")
	product_status = Int_Modular_Multiply(result, result, factor, modulus, modular)
	aver.Always(product_status == Divisor_Status(STATUS_OK),
		"Tonelli-Shanks root product uses nonzero modulus.")
	product_status = Int_Modular_Multiply(power, power, generator, modulus, modular)
	modular.Division.Quotient_Count = initial_quotient_count
	modular.Division.Remainder_Count = initial_remainder_count
	aver.Always(product_status == Divisor_Status(STATUS_OK),
		"Tonelli-Shanks power product uses nonzero modulus.")
}

func int_modular_square_root_iterate(
	destination Int_Handle, integers Modular_Square_Root_Integers,
	modular Int_Modular_Workspace_Handle,
) (status Modular_Square_Root_Result_Status) {
	defer func() {
		Modular_Square_Root_Result_Status_Invariants(
			status, "int_modular_square_root_iterate.status",
		)
	}()
	Int_Handle_Invariants(destination, "int_modular_square_root_iterate.destination")
	Modular_Square_Root_Integers_Invariants(
		integers, "int_modular_square_root_iterate.integers",
	)
	Int_Modular_Workspace_Handle_Invariants(
		modular, "int_modular_square_root_iterate.modular",
	)
	modulus := &integers[MODULAR_SQUARE_ROOT_MODULUS_INDEX]
	exponent := &integers[MODULAR_SQUARE_ROOT_EXPONENT_INDEX]
	power := &integers[MODULAR_SQUARE_ROOT_POWER_INDEX]
	factor := &integers[MODULAR_SQUARE_ROOT_FACTOR_INDEX]
	one := &integers[MODULAR_SQUARE_ROOT_ONE_INDEX]
	subtraction := Int_Subtract(exponent, modulus, one)
	aver.Always(
		subtraction == Arithmetic_Status(STATUS_OK),
		"Validated modular square-root modulus exceeds one during iteration.",
	)
	limit := Int_Trailing_Zero_Bit_Count(exponent)
	order := int(limit)
	for attempt := BIT_COUNT_MINIMUM; attempt < int(limit); attempt++ {
		Int_Set(factor, power)
		distance := BIT_COUNT_MINIMUM
		for distance < order && Int_Compare(factor, one) != ORDER_SAME {
			// Jacobi-one residue is a unit; modular powers cannot become zero.
			int_modular_square_root_square_factor(
				(*Positive_Magnitude)(factor), (*Positive_Magnitude)(modulus),
				modular)
			distance++
		}
		if Int_Compare(factor, one) != ORDER_SAME {
			return STATUS_RESULT_ABSENT
		}
		if distance == BIT_COUNT_MINIMUM {
			return int_modular_square_root_commit(destination, integers, modular)
		}
		if distance >= order {
			return STATUS_RESULT_ABSENT
		}
		Int_Set_Uint_64(exponent, Word_64(WORD_COUNT_INCREMENT))
		shift := order - distance - WORD_COUNT_INCREMENT
		shift_status := Int_Shift_Left(exponent, exponent, Shift_Count(shift))
		aver.Always(
			shift_status == Arithmetic_Status(STATUS_OK),
			"Tonelli-Shanks subgroup exponent fits modulus bit width.",
		)
		int_modular_square_root_advance(integers, modular)
		order = distance
	}
	return STATUS_RESULT_ABSENT
}

func int_modular_square_root_commit(
	destination Int_Handle, integers Modular_Square_Root_Integers,
	modular Int_Modular_Workspace_Handle,
) (status Modular_Square_Root_Result_Status) {
	defer func() {
		Modular_Square_Root_Result_Status_Invariants(
			status, "int_modular_square_root_commit.status",
		)
	}()
	Int_Handle_Invariants(destination, "int_modular_square_root_commit.destination")
	Modular_Square_Root_Integers_Invariants(
		integers, "int_modular_square_root_commit.integers",
	)
	Int_Modular_Workspace_Handle_Invariants(modular, "int_modular_square_root_commit.modular")
	modulus := &integers[MODULAR_SQUARE_ROOT_MODULUS_INDEX]
	residue := &integers[MODULAR_SQUARE_ROOT_RESIDUE_INDEX]
	result := &integers[MODULAR_SQUARE_ROOT_RESULT_INDEX]
	check := &integers[MODULAR_SQUARE_ROOT_CHECK_INDEX]
	initial_quotient_count := modular.Division.Quotient_Count
	initial_remainder_count := modular.Division.Remainder_Count
	product_status := Int_Modular_Multiply(check, result, result, modulus, modular)
	modular.Division.Quotient_Count = initial_quotient_count
	modular.Division.Remainder_Count = initial_remainder_count
	aver.Always(
		product_status == Divisor_Status(STATUS_OK),
		"A modular square-root check uses one validated nonzero modulus.",
	)
	if Int_Compare(check, residue) != ORDER_SAME {
		return STATUS_RESULT_ABSENT
	}
	Int_Set(destination, result)
	return STATUS_OK
}

// Float_Text_Format_Validate rejects unsupported formatting work before conversion.
func Float_Text_Format_Validate(
	value Float_Text_Format_Unvalidated,
) (format Float_Text_Format, status Validation_Status) {
	defer func() {
		Float_Text_Format_Invariants(format, "float_text_format_validate.format")
		Validation_Status_Invariants(status, "float_text_format_validate.status")
	}()
	Float_Text_Format_Unvalidated_Invariants(
		value, "float_text_format_validate.value",
	)
	format = FLOAT_TEXT_KIND_BINARY
	switch value {
	case FLOAT_TEXT_FORMAT_BINARY:
		return FLOAT_TEXT_KIND_BINARY, STATUS_OK
	case FLOAT_TEXT_FORMAT_HEXADECIMAL_FRACTION:
		return FLOAT_TEXT_KIND_HEXADECIMAL_FRACTION, STATUS_OK
	case FLOAT_TEXT_FORMAT_HEXADECIMAL:
		return FLOAT_TEXT_KIND_HEXADECIMAL, STATUS_OK
	case FLOAT_TEXT_FORMAT_DECIMAL_EXPONENT:
		return FLOAT_TEXT_KIND_DECIMAL_EXPONENT, STATUS_OK
	case FLOAT_TEXT_FORMAT_DECIMAL_EXPONENT_UPPER:
		return FLOAT_TEXT_KIND_DECIMAL_EXPONENT_UPPER, STATUS_OK
	case FLOAT_TEXT_FORMAT_DECIMAL_FIXED:
		return FLOAT_TEXT_KIND_DECIMAL_FIXED, STATUS_OK
	case FLOAT_TEXT_FORMAT_DECIMAL_GENERAL:
		return FLOAT_TEXT_KIND_DECIMAL_GENERAL, STATUS_OK
	case FLOAT_TEXT_FORMAT_DECIMAL_GENERAL_UPPER:
		return FLOAT_TEXT_KIND_DECIMAL_GENERAL_UPPER, STATUS_OK
	default:
		return format, STATUS_INPUT_INVALID
	}
}

// Float_Text_Precision_Validate bounds requested digits before conversion.
func Float_Text_Precision_Validate(
	value Float_Text_Precision_Unvalidated,
) (precision Float_Text_Precision, status Validation_Status) {
	defer func() {
		Float_Text_Precision_Invariants(
			precision, "float_text_precision_validate.precision",
		)
		Validation_Status_Invariants(status, "float_text_precision_validate.status")
	}()
	Float_Text_Precision_Unvalidated_Invariants(
		value, "float_text_precision_validate.value",
	)
	if value < FLOAT_TEXT_PRECISION_MINIMUM {
		return FLOAT_TEXT_PRECISION_MINIMUM, STATUS_INPUT_INVALID
	}
	if value > FLOAT_TEXT_PRECISION_MAXIMUM {
		return FLOAT_TEXT_PRECISION_MINIMUM, STATUS_INPUT_INVALID
	}
	return Float_Text_Precision(value), STATUS_OK
}

func float_text_binary(
	output Float_Text_Output, count Float_Text_Count_Handle, value Float_Text_Source,
	integers Float_Text_Integers_Handle, texts Float_Text_Integer_Workspaces,
) {
	Float_Text_Output_Invariants(output, "float_text_binary.output")
	Float_Text_Count_Handle_Invariants(count, "float_text_binary.count")
	Float_Text_Source_Invariants(value, "float_text_binary.value")
	Float_Text_Integers_Handle_Invariants(integers, "float_text_binary.integers")
	Float_Text_Integer_Workspaces_Invariants(texts, "float_text_binary.texts")
	float_text_sign(output, count, value.Negative)
	if value.Form == FLOAT_FORM_INFINITY {
		float_text_infinity(output, value.Negative)
		*count = Float_Text_Count(FLOAT_TEXT_INFINITY_SIZE)
		return
	}
	output_index := *count
	if value.Form == FLOAT_FORM_ZERO {
		output[output_index] = '0'
		*count = output_index + WORD_COUNT_INCREMENT
		return
	}
	integer := &integers.Significand
	// Text scratch must not retain shared words from source mantissa.
	Int_Set(integer, (*Int)(&value.Mantissa))
	bit_count := int(Int_Bit_Count(integer))
	shift := int(value.Precision) - bit_count
	shift_status := Int_Shift_Left(integer, integer, Shift_Count(shift))
	aver.Always(
		shift_status == Arithmetic_Status(STATUS_OK),
		"Binary Float text aligns only inside active precision.",
	)
	text := &texts.Significand
	digit_count := int(int_text_digits(integer, BASE_DECIMAL, text))
	for index := digit_count - WORD_COUNT_INCREMENT; index >= WORD_COUNT_MINIMUM; index-- {
		output[output_index] = text.Digits[index]
		output_index++
	}
	exponent := (*Int)(&integers.Exponent)
	Int_Set_Int_64(
		exponent, Int_64(int(value.Exponent)-int(value.Precision)),
	)
	*count = Float_Text_Count(float_text_write_exponent(
		output, Float_Text_Significand_Count(output_index), integers.Exponent,
		(*Int_Text_Workspace)(&texts.Exponent), 'p', SIGN_BYTE_COUNT_MAXIMUM,
	))
}

func float_text_hexadecimal_fraction(
	output Float_Text_Output, count Float_Text_Count_Handle, value Float_Text_Source,
	integers Float_Text_Integers_Handle, texts Float_Text_Integer_Workspaces,
) {
	Float_Text_Output_Invariants(output, "float_text_hexadecimal_fraction.output")
	Float_Text_Count_Handle_Invariants(count, "float_text_hexadecimal_fraction.count")
	Float_Text_Source_Invariants(value, "float_text_hexadecimal_fraction.value")
	Float_Text_Integers_Handle_Invariants(integers, "float_text_hexadecimal_fraction.integers")
	Float_Text_Integer_Workspaces_Invariants(texts, "float_text_hexadecimal_fraction.texts")
	float_text_sign(output, count, value.Negative)
	if value.Form == FLOAT_FORM_INFINITY {
		float_text_infinity(output, value.Negative)
		*count = Float_Text_Count(FLOAT_TEXT_INFINITY_SIZE)
		return
	}
	output_index := *count
	if value.Form == FLOAT_FORM_ZERO {
		output[output_index] = '0'
		*count = output_index + WORD_COUNT_INCREMENT
		return
	}
	integer := &integers.Significand
	Int_Set(integer, (*Int)(&value.Mantissa))
	bit_count := int(Int_Bit_Count(integer))
	shift := (BASE_HEXADECIMAL_DIGIT_BIT_COUNT -
		bit_count%BASE_HEXADECIMAL_DIGIT_BIT_COUNT) %
		BASE_HEXADECIMAL_DIGIT_BIT_COUNT
	shift_status := Int_Shift_Left(integer, integer, Shift_Count(shift))
	aver.Always(
		shift_status == Arithmetic_Status(STATUS_OK),
		"Hexadecimal fraction alignment adds fewer than one digit.",
	)
	text := &texts.Significand
	digit_count := int(int_text_digits(integer, BASE_HEXADECIMAL, text))
	low_index := WORD_COUNT_MINIMUM
	for low_index+WORD_COUNT_INCREMENT < digit_count {
		if text.Digits[low_index] != '0' {
			break
		}
		low_index++
	}
	output[output_index] = '0'
	output[output_index+WORD_COUNT_INCREMENT] = 'x'
	output[output_index+BASE_PREFIX_BYTE_COUNT] = '.'
	output_index += Float_Text_Count(BASE_PREFIX_BYTE_COUNT + DECIMAL_POINT_BYTE_COUNT)
	for index := digit_count - WORD_COUNT_INCREMENT; index >= low_index; index-- {
		output[output_index] = text.Digits[index]
		output_index++
	}
	exponent := (*Int)(&integers.Exponent)
	Int_Set_Int_64(exponent, Int_64(value.Exponent))
	*count = Float_Text_Count(float_text_write_exponent(
		output, Float_Text_Significand_Count(output_index), integers.Exponent,
		(*Int_Text_Workspace)(&texts.Exponent), 'p', SIGN_BYTE_COUNT_MAXIMUM,
	))
}

func float_text_hexadecimal(
	output Float_Text_Output, count Float_Text_Count_Handle, precision Float_Text_Precision,
	source Float_Text_Source,
	storage Float_Text_Rounded_Handle, integers Float_Text_Integers_Handle,
	text Float_Text_Integer_Workspaces,
) {
	Float_Text_Output_Invariants(output, "float_text_hexadecimal.output")
	Float_Text_Count_Handle_Invariants(count, "float_text_hexadecimal.count")
	Float_Text_Precision_Invariants(precision, "float_text_hexadecimal.precision")
	Float_Text_Source_Invariants(source, "float_text_hexadecimal.source")
	Float_Text_Rounded_Handle_Invariants(storage, "float_text_hexadecimal.storage")
	Float_Text_Integers_Handle_Invariants(integers, "float_text_hexadecimal.integers")
	Float_Text_Integer_Workspaces_Invariants(text, "float_text_hexadecimal.text")
	value := (*Float)(&source)
	float_text_sign(output, count, value.Negative)
	if value.Form == FLOAT_FORM_INFINITY {
		float_text_infinity(output, value.Negative)
		*count = Float_Text_Count(FLOAT_TEXT_INFINITY_SIZE)
		return
	}
	if value.Form == FLOAT_FORM_ZERO {
		zero_count := float_text_hexadecimal_zero(output, value.Negative, precision)
		*count = Float_Text_Count(zero_count)
		return
	}
	rounded := (*Float)((*Float_Text_Rounded)(storage))
	Float_Copy(rounded, value)
	target_precision := WORD_COUNT_INCREMENT +
		BASE_HEXADECIMAL_DIGIT_BIT_COUNT*int(precision)
	if precision == FLOAT_TEXT_PRECISION_MINIMUM {
		minimum := int(Float_Minimum_Precision(value))
		target_precision = WORD_COUNT_INCREMENT +
			(minimum-WORD_COUNT_INCREMENT+BASE_HEXADECIMAL_DIGIT_BIT_COUNT-
				WORD_COUNT_INCREMENT)/BASE_HEXADECIMAL_DIGIT_BIT_COUNT*
				BASE_HEXADECIMAL_DIGIT_BIT_COUNT
	}
	validation := Float_Set_Precision(
		rounded, Float_Precision_Unvalidated(target_precision),
	)
	aver.Always(
		validation == Validation_Status(STATUS_OK),
		"Hexadecimal text target precision was bounded before conversion.",
	)
	integer := &integers.Significand
	// Later formats reuse significand scratch without owning rounded float metadata.
	Int_Set(integer, (*Int)(&rounded.Mantissa))
	bit_count := int(Int_Bit_Count(integer))
	shift_status := Int_Shift_Left(
		integer, integer, Shift_Count(target_precision-bit_count),
	)
	aver.Always(
		shift_status == Arithmetic_Status(STATUS_OK),
		"Rounded hexadecimal significand fits target precision.",
	)
	Int_Set_Int_64(
		(*Int)(&integers.Exponent),
		Int_64(int(rounded.Exponent)-WORD_COUNT_INCREMENT),
	)
	*count = Float_Text_Count(float_text_hexadecimal_finite(
		output, value.Negative, Positive_Int(integers.Significand), text))
	*count = Float_Text_Count(float_text_write_exponent(
		output, Float_Text_Significand_Count(*count), integers.Exponent,
		(*Int_Text_Workspace)(&text.Exponent), 'p', BASE_BINARY))
}

func float_text_hexadecimal_finite(
	output Float_Text_Output, negative Polarity,
	mantissa Positive_Int, text Float_Text_Integer_Workspaces,
) (count Float_Text_Hexadecimal_Count) {
	defer func() {
		Float_Text_Hexadecimal_Count_Invariants(
			count, "float_text_hexadecimal_finite.count")
	}()
	Float_Text_Output_Invariants(output, "float_text_hexadecimal_finite.output")
	Polarity_Invariants(negative, "float_text_hexadecimal_finite.negative")
	Positive_Int_Invariants(mantissa, "float_text_hexadecimal_finite.mantissa")
	Float_Text_Integer_Workspaces_Invariants(text, "float_text_hexadecimal_finite.text")
	integer := (*Int)(&mantissa)
	significand := &text.Significand
	digit_count := int(int_text_digits(integer, BASE_HEXADECIMAL, significand))
	index := WORD_COUNT_MINIMUM
	if negative == POLARITY_NEGATIVE {
		index = SIGN_BYTE_COUNT_MAXIMUM
	}
	output[index] = '0'
	output[index+WORD_COUNT_INCREMENT] = 'x'
	index += BASE_PREFIX_BYTE_COUNT
	output[index] = significand.Digits[digit_count-WORD_COUNT_INCREMENT]
	index++
	if digit_count > WORD_COUNT_INCREMENT {
		output[index] = '.'
		index++
		for digit_index := digit_count - BASE_BINARY; digit_index >= 0; digit_index-- {
			output[index] = significand.Digits[digit_index]
			index++
		}
	}
	return Float_Text_Hexadecimal_Count(index)
}

func float_text_hexadecimal_zero(
	output Float_Text_Output, negative Polarity, precision Float_Text_Precision,
) (count Float_Text_Hexadecimal_Zero_Count) {
	defer func() {
		Float_Text_Hexadecimal_Zero_Count_Invariants(
			count, "float_text_hexadecimal_zero.count",
		)
	}()
	Float_Text_Output_Invariants(output, "float_text_hexadecimal_zero.output")
	Polarity_Invariants(negative, "float_text_hexadecimal_zero.negative")
	Float_Text_Precision_Invariants(precision, "float_text_hexadecimal_zero.precision")
	index := bytes.SLICE_SIZE_MINIMUM
	if negative == POLARITY_NEGATIVE {
		index = SIGN_BYTE_COUNT_MAXIMUM
	}
	output[index] = '0'
	output[index+WORD_COUNT_INCREMENT] = 'x'
	output[index+BASE_PREFIX_BYTE_COUNT] = '0'
	index += BASE_PREFIX_BYTE_COUNT + WORD_COUNT_INCREMENT
	if precision > 0 {
		output[index] = '.'
		index++
		for digit := WORD_COUNT_MINIMUM; digit < int(precision); digit++ {
			output[index] = '0'
			index++
		}
	}
	// Zero exponent needs no integer scratch or decimal conversion.
	output[index] = 'p'
	output[index+WORD_COUNT_INCREMENT] = '+'
	output[index+BASE_BINARY] = '0'
	output[index+BASE_BINARY+WORD_COUNT_INCREMENT] = '0'
	return Float_Text_Hexadecimal_Zero_Count(index + BASE_BINARY + BASE_BINARY)
}

func float_text_decimal(
	output Float_Text_Output, count Float_Text_Count_Handle, source Float_Text_Source,
	format Float_Text_Decimal_Format, digits Float_Text_Precision,
	integers Float_Text_Integers_Handle, text Float_Text_Integer_Workspaces,
	decimals Float_Text_Decimals_Handle, magnitude Float_Text_Magnitude_Handle,
) {
	Float_Text_Output_Invariants(output, "float_text_decimal.output")
	Float_Text_Count_Handle_Invariants(count, "float_text_decimal.count")
	Float_Text_Source_Invariants(source, "float_text_decimal.source")
	Float_Text_Decimal_Format_Invariants(format, "float_text_decimal.format")
	Float_Text_Precision_Invariants(digits, "float_text_decimal.digits")
	Float_Text_Integers_Handle_Invariants(integers, "float_text_decimal.integers")
	Float_Text_Integer_Workspaces_Invariants(text, "float_text_decimal.text")
	Float_Text_Decimals_Handle_Invariants(decimals, "float_text_decimal.decimals")
	Float_Text_Magnitude_Handle_Invariants(magnitude, "float_text_decimal.magnitude")
	value, decimal := (*Float)(&source), &decimals.Value
	float_text_sign(output, count, value.Negative)
	if value.Form == FLOAT_FORM_INFINITY {
		float_text_infinity(output, value.Negative)
		*count = Float_Text_Count(FLOAT_TEXT_INFINITY_SIZE)
		return
	}
	float_decimal_initialize(Float_Text_Source_Mantissa(value.Mantissa), value.Exponent,
		decimal, &integers.Significand, &text.Significand)
	bounded := (*Float_Text_Value_Decimal)(decimal)
	precision := int(digits)
	shortest := Boolean(precision == FLOAT_TEXT_PRECISION_MINIMUM)
	if shortest {
		if decimal.Control.Count != 0 {
			float_decimal_round_shortest(Float_Text_References{
				Mantissa:  Float_Active_Mantissa(source.Mantissa),
				Precision: Float_Active_Precision(source.Precision),
				Exponent:  source.Exponent,
				Decimals: (*Float_Text_Nonzero_Decimals)(
					(*Float_Text_Decimals)(decimals)),
				Magnitude: magnitude,
			})
		}
	}
	digit_count, exponent := int(decimal.Control.Count), int(decimal.Control.Exponent)
	if format <= Float_Text_Decimal_Format(FLOAT_TEXT_KIND_DECIMAL_EXPONENT_UPPER) {
		retained := max(digit_count-WORD_COUNT_INCREMENT, WORD_COUNT_MINIMUM)
		if !shortest {
			retained = precision
			float_decimal_round(
				bounded, Float_Text_Retained_Digits(precision+WORD_COUNT_INCREMENT))
		}
		upper := format == Float_Text_Decimal_Format(FLOAT_TEXT_KIND_DECIMAL_EXPONENT_UPPER)
		*count = Float_Text_Count(float_text_decimal_exponent(
			output, Sign_Byte_Count(*count),
			Float_Text_Significand_Digits(retained), Boolean(upper),
			Float_Text_Short_Decimal(*decimal), &integers.Exponent,
			(*Int_Text_Workspace)(&text.Exponent)))
		return
	}
	if format == Float_Text_Decimal_Format(FLOAT_TEXT_KIND_DECIMAL_FIXED) {
		if shortest {
			precision = max(digit_count-exponent, WORD_COUNT_MINIMUM)
		} else {
			float_decimal_round(bounded, Float_Text_Retained_Digits(exponent+precision))
		}
		*count = Float_Text_Count(float_text_decimal_fixed(
			output, Sign_Byte_Count(*count), Float_Text_Digits(precision), bounded))
		return
	}
	if !shortest {
		float_decimal_round(bounded,
			Float_Text_Retained_Digits(max(precision, WORD_COUNT_INCREMENT)))
	}
	*count = Float_Text_Count(float_text_decimal_general(
		output, Sign_Byte_Count(*count), digits,
		Boolean(format == Float_Text_Decimal_Format(FLOAT_TEXT_KIND_DECIMAL_GENERAL_UPPER)),
		(*Float_Text_Short_Decimal)(decimal), &integers.Exponent,
		(*Int_Text_Workspace)(&text.Exponent), shortest))
}

func float_text_decimal_general(
	output Float_Text_Output, prefix Sign_Byte_Count, digits Float_Text_Precision,
	upper Boolean,
	reference Float_Text_Short_Decimal_Handle,
	integer Float_Text_Exponent_Handle, text Int_Text_Workspace_Handle, shortest Boolean,
) (count Float_Text_General_Count) {
	defer func() {
		Float_Text_General_Count_Invariants(count, "float_text_decimal_general.count")
	}()
	Float_Text_Output_Invariants(output, "float_text_decimal_general.output")
	Sign_Byte_Count_Invariants(prefix, "float_text_decimal_general.prefix")
	Float_Text_Precision_Invariants(digits, "float_text_decimal_general.digits")
	Boolean_Invariants(upper, "float_text_decimal_general.upper")
	Float_Text_Short_Decimal_Handle_Invariants(reference, "float_text_decimal_general.decimal")
	Float_Text_Exponent_Handle_Invariants(integer, "float_text_decimal_general.integer")
	Int_Text_Workspace_Handle_Invariants(text, "float_text_decimal_general.text")
	Boolean_Invariants(shortest, "float_text_decimal_general.shortest")
	decimal := (*Float_Text_Decimal)((*Float_Text_Short_Decimal)(reference))
	precision := int(digits)
	if shortest {
		precision = int(decimal.Control.Count)
	} else {
		if precision == 0 {
			precision = WORD_COUNT_INCREMENT
		}
	}
	digit_count := int(decimal.Control.Count)
	exponent := int(decimal.Control.Exponent)
	exponent_precision := precision
	if exponent_precision > digit_count {
		if digit_count >= exponent {
			exponent_precision = digit_count
		}
	}
	if shortest {
		exponent_precision = BASE_BINARY + BASE_HEXADECIMAL_DIGIT_BIT_COUNT
	}
	decimal_exponent := exponent - WORD_COUNT_INCREMENT
	if decimal_exponent < -BASE_HEXADECIMAL_DIGIT_BIT_COUNT {
		return Float_Text_General_Count(float_text_decimal_exponent(
			output, prefix,
			Float_Text_Significand_Digits(precision-WORD_COUNT_INCREMENT),
			upper, Float_Text_Short_Decimal(*decimal), integer, text))
	}
	if decimal_exponent >= exponent_precision {
		return Float_Text_General_Count(float_text_decimal_exponent(
			output, prefix,
			Float_Text_Significand_Digits(precision-WORD_COUNT_INCREMENT),
			upper, Float_Text_Short_Decimal(*decimal), integer, text))
	}
	if precision > exponent {
		precision = digit_count
	}
	precision -= exponent
	if precision < 0 {
		precision = 0
	}
	return Float_Text_General_Count(float_text_decimal_fixed(
		output, prefix, Float_Text_Digits(precision), (*Float_Text_Value_Decimal)(decimal)))
}

func float_text_decimal_exponent(
	output Float_Text_Output, prefix Sign_Byte_Count,
	digits Float_Text_Significand_Digits,
	upper Boolean, decimal Float_Text_Short_Decimal,
	integer Float_Text_Exponent_Handle, text Int_Text_Workspace_Handle,
) (count Float_Text_Scientific_Count) {
	defer func() {
		Float_Text_Scientific_Count_Invariants(count, "float_text_decimal_exponent.count")
	}()
	Float_Text_Output_Invariants(output, "float_text_decimal_exponent.output")
	Sign_Byte_Count_Invariants(prefix, "float_text_decimal_exponent.prefix")
	Float_Text_Significand_Digits_Invariants(digits, "float_text_decimal_exponent.digits")
	Boolean_Invariants(upper, "float_text_decimal_exponent.upper")
	Float_Text_Short_Decimal_Invariants(decimal, "float_text_decimal_exponent.decimal")
	Float_Text_Exponent_Handle_Invariants(integer, "float_text_decimal_exponent.integer")
	Int_Text_Workspace_Handle_Invariants(text, "float_text_decimal_exponent.text")
	digit_count := int(decimal.Control.Count)
	precision := int(digits)
	index := Float_Text_Count(prefix)
	digit := byte('0')
	if digit_count > 0 {
		digit = decimal.Digits[WORD_COUNT_MINIMUM]
	}
	output[index] = digit
	index++
	if precision > 0 {
		output[index] = '.'
		index++
		for digit_index := WORD_COUNT_INCREMENT; digit_index <= precision; digit_index++ {
			digit = '0'
			if digit_index < digit_count {
				digit = decimal.Digits[digit_index]
			}
			output[index] = digit
			index++
		}
	}
	exponent := bytes.SLICE_SIZE_MINIMUM
	if digit_count > 0 {
		exponent = int(decimal.Control.Exponent) - WORD_COUNT_INCREMENT
	}
	Int_Set_Int_64((*Int)((*Float_Text_Exponent)(integer)), Int_64(exponent))
	marker := Float_Text_Exponent_Marker('e')
	if upper {
		marker = 'E'
	}
	return Float_Text_Scientific_Count(float_text_write_exponent(
		output, Float_Text_Significand_Count(index), *integer, text, marker, BASE_BINARY))
}

func float_text_decimal_fixed(
	output Float_Text_Output, prefix Sign_Byte_Count, digits Float_Text_Digits,
	decimal Float_Text_Value_Decimal_Handle,
) (count Float_Text_Nonempty_Count) {
	defer func() {
		Float_Text_Nonempty_Count_Invariants(count, "float_text_decimal_fixed.count")
	}()
	Float_Text_Output_Invariants(output, "float_text_decimal_fixed.output")
	Sign_Byte_Count_Invariants(prefix, "float_text_decimal_fixed.prefix")
	Float_Text_Digits_Invariants(digits, "float_text_decimal_fixed.digits")
	Float_Text_Value_Decimal_Handle_Invariants(decimal, "float_text_decimal_fixed.decimal")
	digit_count := int(decimal.Control.Count)
	exponent := int(decimal.Control.Exponent)
	precision := int(digits)
	index := int(prefix)
	if exponent > 0 {
		copied := digit_count
		if copied > exponent {
			copied = exponent
		}
		for digit_index := WORD_COUNT_MINIMUM; digit_index < copied; digit_index++ {
			output[index] = decimal.Digits[digit_index]
			index++
		}
		for digit_index := copied; digit_index < exponent; digit_index++ {
			output[index] = '0'
			index++
		}
	} else {
		output[index] = '0'
		index++
	}
	if precision > 0 {
		output[index] = '.'
		index++
		for digit_index := WORD_COUNT_MINIMUM; digit_index < precision; digit_index++ {
			source_index := exponent + digit_index
			digit := byte('0')
			if source_index >= WORD_COUNT_MINIMUM {
				if source_index < digit_count {
					digit = decimal.Digits[source_index]
				}
			}
			output[index] = digit
			index++
		}
	}
	aver.Always(
		int(index) <= FLOAT_TEXT_SIZE_MAXIMUM,
		"Fixed Float text fits its exact derived output bound.",
	)
	return Float_Text_Nonempty_Count(index)
}

func float_decimal_initialize(
	mantissa Float_Text_Source_Mantissa, exponent Float_Exponent,
	decimal Float_Text_Decimal_Handle, integer Int_Handle,
	text Int_Text_Workspace_Handle,
) {
	Float_Text_Source_Mantissa_Invariants(mantissa, "float_decimal_initialize.mantissa")
	Float_Exponent_Invariants(exponent, "float_decimal_initialize.exponent")
	Float_Text_Decimal_Handle_Invariants(decimal, "float_decimal_initialize.decimal")
	Int_Handle_Invariants(integer, "float_decimal_initialize.integer")
	Int_Text_Workspace_Handle_Invariants(text, "float_decimal_initialize.text")
	decimal.Control = Float_Text_Decimal_Control{}
	if mantissa.Count == WORD_COUNT_MINIMUM {
		return
	}
	// Decimal shifts must not mutate source mantissa through shared words.
	Int_Set(integer, (*Int)(&mantissa))
	bit_count := int(Int_Bit_Count(integer))
	shift := int(exponent) - bit_count
	if shift < 0 {
		zero_count := int(Int_Trailing_Zero_Bit_Count(integer))
		strip := -shift
		if strip > zero_count {
			strip = zero_count
		}
		Int_Shift_Right(integer, integer, Shift_Count(strip))
		shift += strip
	}
	if shift > 0 {
		shift_status := Int_Shift_Left(integer, integer, Shift_Count(shift))
		aver.Always(
			shift_status == Arithmetic_Status(STATUS_OK),
			"Positive Float decimal shift cannot exceed normalized exponent.",
		)
		shift = 0
	}
	digit_count := int(int_text_digits(integer, BASE_DECIMAL, text))
	decimal.Control.Count = Float_Text_Decimal_Count(digit_count)
	decimal.Control.Exponent = Float_Text_Decimal_Exponent(digit_count)
	source_index := digit_count - WORD_COUNT_INCREMENT
	destination_index := WORD_COUNT_MINIMUM
	for destination_index < digit_count {
		decimal.Digits[destination_index] = text.Digits[source_index]
		source_index--
		destination_index++
	}
	float_decimal_trim(decimal)
	nonzero := (*Float_Text_Shift_Decimal)((*Float_Text_Decimal)(decimal))
	for shift < -FLOAT_DECIMAL_SHIFT_MAXIMUM {
		float_decimal_shift_right(nonzero, FLOAT_DECIMAL_SHIFT_MAXIMUM)
		shift += FLOAT_DECIMAL_SHIFT_MAXIMUM
	}
	if shift < 0 {
		float_decimal_shift_right(nonzero, Float_Text_Decimal_Shift(-shift))
	}
}

func float_decimal_shift_right(
	reference Float_Text_Shift_Decimal_Handle, scale Float_Text_Decimal_Shift,
) {
	Float_Text_Shift_Decimal_Handle_Invariants(reference, "float_decimal_shift_right.decimal")
	Float_Text_Decimal_Shift_Invariants(scale, "float_decimal_shift_right.scale")
	decimal := (*Float_Text_Decimal)((*Float_Text_Shift_Decimal)(reference))
	shift := uint(scale)
	count := int(decimal.Control.Count)
	read_index := WORD_COUNT_MINIMUM
	number := Word(bits.WORD_64_MINIMUM)
	for number>>shift == 0 && read_index < count {
		number = number*Word(BASE_DECIMAL) + Word(decimal.Digits[read_index]-'0')
		read_index++
	}
	if number == 0 {
		decimal.Control.Count = bytes.SLICE_SIZE_MINIMUM
		return
	}
	for number>>shift == 0 {
		read_index++
		number *= Word(BASE_DECIMAL)
	}
	decimal.Control.Exponent += Float_Text_Decimal_Exponent(WORD_COUNT_INCREMENT - read_index)
	write_index := WORD_COUNT_MINIMUM
	mask := Word(bits.CARRY_MAXIMUM)<<shift - Word(bits.CARRY_MAXIMUM)
	for read_index < count {
		digit := number >> shift
		number &= mask
		decimal.Digits[write_index] = byte(digit) + '0'
		write_index++
		number = number*Word(BASE_DECIMAL) + Word(decimal.Digits[read_index]-'0')
		read_index++
	}
	for number > 0 && write_index < count {
		digit := number >> shift
		number &= mask
		decimal.Digits[write_index] = byte(digit) + '0'
		write_index++
		number *= Word(BASE_DECIMAL)
	}
	for number > 0 {
		aver.Always(
			write_index < FLOAT_DECIMAL_DIGIT_COUNT_MAXIMUM,
			"Exact Float decimal right shift fits derived digit capacity.",
		)
		digit := number >> shift
		number &= mask
		decimal.Digits[write_index] = byte(digit) + '0'
		write_index++
		number *= Word(BASE_DECIMAL)
	}
	decimal.Control.Count = Float_Text_Decimal_Count(write_index)
	float_decimal_trim(decimal)
}

func float_decimal_round(
	reference Float_Text_Value_Decimal_Handle, digits Float_Text_Retained_Digits,
) {
	Float_Text_Value_Decimal_Handle_Invariants(reference, "float_decimal_round.decimal")
	Float_Text_Retained_Digits_Invariants(digits, "float_decimal_round.digits")
	decimal := (*Float_Text_Decimal)((*Float_Text_Value_Decimal)(reference))
	retained := int(digits)
	count := int(decimal.Control.Count)
	if retained < 0 {
		return
	}
	if retained >= count {
		return
	}
	digit := decimal.Digits[retained]
	round_up := digit >= '5'
	if digit == '5' {
		if retained+WORD_COUNT_INCREMENT == count {
			round_up = false
			if retained > 0 {
				round_up = (decimal.Digits[retained-WORD_COUNT_INCREMENT]-'0')&
					byte(bits.CARRY_MAXIMUM) != 0
			}
		}
	}
	if round_up {
		decimal.Control.Count = Float_Text_Decimal_Count(float_decimal_round_up(
			decimal.Digits, &decimal.Control.Exponent,
			Float_Text_Rounding_Count(retained)))
		return
	}
	decimal.Control.Count = Float_Text_Decimal_Count(retained)
	float_decimal_trim(decimal)
}

func float_decimal_round_up(
	digits Float_Text_Decimal_Digits, exponent Float_Text_Decimal_Exponent_Handle,
	retained Float_Text_Rounding_Count,
) (count Float_Text_Rounded_Count) {
	defer func() {
		Float_Text_Rounded_Count_Invariants(count, "float_decimal_round_up.count")
	}()
	Float_Text_Decimal_Digits_Invariants(digits, "float_decimal_round_up.digits")
	Float_Text_Decimal_Exponent_Handle_Invariants(exponent, "float_decimal_round_up.exponent")
	Float_Text_Rounding_Count_Invariants(retained, "float_decimal_round_up.retained")
	for index := int(retained) - WORD_COUNT_INCREMENT; index >= WORD_COUNT_MINIMUM; index-- {
		if digits[index] < '9' {
			digits[index]++
			// Increment leaves a nonzero final digit; trimming cannot remove it.
			return Float_Text_Rounded_Count(index + WORD_COUNT_INCREMENT)
		}
	}
	digits[WORD_COUNT_MINIMUM] = '1'
	*exponent++
	return WORD_COUNT_INCREMENT
}

func float_decimal_trim(decimal Float_Text_Decimal_Handle) {
	Float_Text_Decimal_Handle_Invariants(decimal, "float_decimal_trim.decimal")
	count := int(decimal.Control.Count)
	for count > 0 {
		if decimal.Digits[count-WORD_COUNT_INCREMENT] != '0' {
			break
		}
		count--
	}
	decimal.Control.Count = Float_Text_Decimal_Count(count)
}

func float_decimal_round_shortest(workspace Float_Text_References) {
	Float_Text_References_Invariants(workspace, "float_decimal_round_shortest.workspace")
	decimal := &workspace.Decimals.Value
	scale := Float_Text_Binary_Shift(
		int(workspace.Exponent) - int(workspace.Precision) - WORD_COUNT_INCREMENT)
	lower, upper := &workspace.Decimals.Lower, &workspace.Decimals.Upper
	magnitude := (*Float_Text_Magnitude)(workspace.Magnitude)
	nonzero := (*Float_Text_Nonzero_Magnitude)(magnitude)
	float_text_midpoint_load(workspace.Mantissa, workspace.Precision, workspace.Magnitude)
	inclusive := workspace.Magnitude.Words[WORD_COUNT_MINIMUM]&Word(BASE_BINARY) == 0
	for index := WORD_COUNT_MINIMUM; index < int(workspace.Magnitude.Count); index++ {
		previous := workspace.Magnitude.Words[index]
		workspace.Magnitude.Words[index]--
		if previous != 0 {
			break
		}
	}
	float_decimal_initialize_magnitude((*Float_Text_Decimal)(lower), nonzero, scale)
	float_text_midpoint_load(workspace.Mantissa, workspace.Precision, workspace.Magnitude)
	workspace.Magnitude.Words[WORD_COUNT_MINIMUM]++
	float_decimal_initialize_magnitude((*Float_Text_Decimal)(upper), nonzero, scale)
	lower_count := int(lower.Control.Count)
	upper_count := int(upper.Control.Count)
	for index := WORD_COUNT_MINIMUM; index < int(decimal.Control.Count); index++ {
		digit := decimal.Digits[index]
		lower_digit := byte('0')
		if index < lower_count {
			lower_digit = lower.Digits[index]
		}
		upper_digit := byte('0')
		if index < upper_count {
			upper_digit = upper.Digits[index]
		}
		down := lower_digit != digit
		if !down {
			if inclusive {
				if index+WORD_COUNT_INCREMENT == lower_count {
					down = true
				}
			}
		}
		up := digit != upper_digit
		if up {
			greater := inclusive
			if digit+WORD_COUNT_INCREMENT < upper_digit {
				greater = true
			}
			if index+WORD_COUNT_INCREMENT < upper_count {
				greater = true
			}
			up = greater
		}
		retained := Float_Text_Retained_Digits(index + WORD_COUNT_INCREMENT)
		if down {
			if up {
				float_decimal_round((*Float_Text_Value_Decimal)(decimal), retained)
				return
			}
			decimal.Control.Count = Float_Text_Decimal_Count(retained)
			float_decimal_trim(decimal)
			return
		}
		if up {
			decimal.Control.Count = Float_Text_Decimal_Count(float_decimal_round_up(
				decimal.Digits, &decimal.Control.Exponent,
				Float_Text_Rounding_Count(retained)))
			return
		}
	}
}

func float_text_midpoint_load(
	value Float_Active_Mantissa, precision Float_Active_Precision,
	magnitude Float_Text_Magnitude_Handle,
) {
	Float_Active_Mantissa_Invariants(value, "float_text_midpoint_load.value")
	Float_Active_Precision_Invariants(precision, "float_text_midpoint_load.precision")
	Float_Text_Magnitude_Handle_Invariants(magnitude, "float_text_midpoint_load.magnitude")
	mantissa := (*Int)(&value)
	bit_count := int(Int_Bit_Count(mantissa))
	guarded_bit_count := int(precision) + WORD_COUNT_INCREMENT
	shift_count := guarded_bit_count - bit_count
	clear(magnitude.Words)
	word_shift := shift_count / WORD_BIT_COUNT
	bit_shift := uint(shift_count % WORD_BIT_COUNT)
	source_index := WORD_COUNT_MINIMUM
	for source_index < int(mantissa.Count) {
		destination_index := source_index + word_shift
		word := mantissa.Words[source_index]
		magnitude.Words[destination_index] |= word << bit_shift
		if bit_shift != 0 {
			magnitude.Words[destination_index+WORD_COUNT_INCREMENT] |=
				word >> (WORD_BIT_COUNT - bit_shift)
		}
		source_index++
	}
	word_count := (guarded_bit_count + WORD_BIT_COUNT - WORD_COUNT_INCREMENT) /
		WORD_BIT_COUNT
	magnitude.Count = Float_Text_Magnitude_Count(word_count)
}

func float_decimal_initialize_magnitude(
	decimal Float_Text_Decimal_Handle, magnitude Float_Text_Nonzero_Magnitude_Handle,
	scale Float_Text_Binary_Shift,
) {
	Float_Text_Decimal_Handle_Invariants(decimal, "float_decimal_initialize_magnitude.decimal")
	Float_Text_Nonzero_Magnitude_Handle_Invariants(
		magnitude, "float_decimal_initialize_magnitude.magnitude",
	)
	Float_Text_Binary_Shift_Invariants(scale, "float_decimal_initialize_magnitude.scale")
	decimal.Control = Float_Text_Decimal_Control{}
	shift := int(scale)
	if shift > 0 {
		float_text_magnitude_shift_left(
			(*Float_Text_Shift_Magnitude)((*Float_Text_Nonzero_Magnitude)(magnitude)),
			Float_Text_Left_Shift(shift))
		shift = 0
	}
	word_count := int(magnitude.Count)
	digit_count := bytes.SLICE_SIZE_MINIMUM
	for word_count > WORD_COUNT_MINIMUM {
		remainder := uint64(bits.WORD_64_MINIMUM)
		word_index := word_count - WORD_COUNT_INCREMENT
		for word_index >= WORD_COUNT_MINIMUM {
			word := uint64(magnitude.Words[word_index])
			high_limb := word >> TEXT_DIVISION_LIMB_BIT_COUNT
			high_dividend := remainder<<TEXT_DIVISION_LIMB_BIT_COUNT | high_limb
			quotient_high := high_dividend / BASE_DECIMAL
			remainder = high_dividend % BASE_DECIMAL
			low_limb := word & TEXT_DIVISION_LIMB_MASK
			low_dividend := remainder<<TEXT_DIVISION_LIMB_BIT_COUNT | low_limb
			quotient_low := low_dividend / BASE_DECIMAL
			remainder = low_dividend % BASE_DECIMAL
			magnitude.Words[word_index] = Word(
				quotient_high<<TEXT_DIVISION_LIMB_BIT_COUNT | quotient_low,
			)
			word_index--
		}
		for word_count > WORD_COUNT_MINIMUM {
			if magnitude.Words[word_count-WORD_COUNT_INCREMENT] != 0 {
				break
			}
			word_count--
		}
		aver.Always(
			digit_count < FLOAT_DECIMAL_INTEGER_DIGIT_COUNT_MAXIMUM,
			"Shortest-bound magnitude fits derived decimal integer digits.",
		)
		decimal.Digits[digit_count] = byte(remainder) + '0'
		digit_count++
	}
	low_index := WORD_COUNT_MINIMUM
	high_index := digit_count - WORD_COUNT_INCREMENT
	for low_index < high_index {
		decimal.Digits[low_index], decimal.Digits[high_index] =
			decimal.Digits[high_index], decimal.Digits[low_index]
		low_index++
		high_index--
	}
	decimal.Control.Count = Float_Text_Decimal_Count(digit_count)
	decimal.Control.Exponent = Float_Text_Decimal_Exponent(digit_count)
	float_decimal_trim(decimal)
	nonzero := (*Float_Text_Shift_Decimal)((*Float_Text_Decimal)(decimal))
	for shift < -FLOAT_DECIMAL_SHIFT_MAXIMUM {
		float_decimal_shift_right(nonzero, FLOAT_DECIMAL_SHIFT_MAXIMUM)
		shift += FLOAT_DECIMAL_SHIFT_MAXIMUM
	}
	if shift < 0 {
		float_decimal_shift_right(nonzero, Float_Text_Decimal_Shift(-shift))
	}
}

func float_text_magnitude_shift_left(
	magnitude Float_Text_Shift_Magnitude_Handle, shift Float_Text_Left_Shift,
) {
	Float_Text_Shift_Magnitude_Handle_Invariants(
		magnitude, "float_text_magnitude_shift_left.magnitude",
	)
	Float_Text_Left_Shift_Invariants(shift, "float_text_magnitude_shift_left.shift")
	source_count := int(magnitude.Count)
	high_word := magnitude.Words[source_count-WORD_COUNT_INCREMENT]
	high_bit_count := WORD_BIT_COUNT -
		int(bits.Leading_Zeros_64(bits.Word_64(high_word)))
	result_bit_count := (source_count-WORD_COUNT_INCREMENT)*WORD_BIT_COUNT +
		high_bit_count + int(shift)
	result_count := (result_bit_count + WORD_BIT_COUNT - WORD_COUNT_INCREMENT) /
		WORD_BIT_COUNT
	word_shift := int(shift) / WORD_BIT_COUNT
	bit_shift := uint(shift % WORD_BIT_COUNT)
	destination_index := result_count - WORD_COUNT_INCREMENT
	for destination_index >= WORD_COUNT_MINIMUM {
		source_index := destination_index - word_shift
		word := Word(bits.WORD_64_MINIMUM)
		if source_index >= WORD_COUNT_MINIMUM {
			if source_index < source_count {
				word = magnitude.Words[source_index] << bit_shift
			}
			if bit_shift != 0 {
				if source_index > WORD_COUNT_MINIMUM {
					if source_index <= source_count {
						previous_index := source_index -
							WORD_COUNT_INCREMENT
						previous := magnitude.Words[previous_index]
						word |= previous >>
							(WORD_BIT_COUNT - bit_shift)
					}
				}
			}
		}
		magnitude.Words[destination_index] = word
		destination_index--
	}
	magnitude.Count = Float_Text_Magnitude_Count(result_count)
}

// Float_Parse_Base_Validate rejects every radix outside stdlib Float syntax.
func Float_Parse_Base_Validate(
	value Base_Unvalidated,
) (base Float_Parse_Base, status Validation_Status) {
	defer func() {
		Float_Parse_Base_Invariants(base, "float_parse_base_validate.base")
		Validation_Status_Invariants(status, "float_parse_base_validate.status")
	}()
	Base_Unvalidated_Invariants(value, "float_parse_base_validate.value")
	switch value {
	case BASE_AUTOMATIC:
		return FLOAT_PARSE_BASE_AUTOMATIC, STATUS_OK
	case BASE_BINARY:
		return FLOAT_PARSE_BASE_BINARY, STATUS_OK
	case BASE_OCTAL:
		return FLOAT_PARSE_BASE_OCTAL, STATUS_OK
	case BASE_DECIMAL:
		return FLOAT_PARSE_BASE_DECIMAL, STATUS_OK
	case BASE_HEXADECIMAL:
		return FLOAT_PARSE_BASE_HEXADECIMAL, STATUS_OK
	default:
		return FLOAT_PARSE_BASE_AUTOMATIC, STATUS_INPUT_INVALID
	}
}

// Float_Parse preserves stdlib numeric syntax within explicit exact-factor bounds.
func Float_Parse(
	destination Float_Handle,
	source Float_Parse_Text_Unvalidated,
	base_unvalidated Base_Unvalidated,
	workspace Float_Parse_Workspace_Handle,
) (base Float_Parse_Base, status Parse_Status) {
	defer func() {
		Float_Parse_Base_Invariants(base, "float_parse.base")
		Parse_Status_Invariants(status, "float_parse.status")
	}()
	Float_Handle_Invariants(destination, "float_parse.destination_initial")
	Float_Parse_Text_Unvalidated_Invariants(source, "float_parse.source")
	Base_Unvalidated_Invariants(base_unvalidated, "float_parse.base_unvalidated")
	Float_Parse_Workspace_Handle_Invariants(workspace, "float_parse.workspace")
	validated_base, validation := Float_Parse_Base_Validate(base_unvalidated)
	if validation != Validation_Status(STATUS_OK) {
		return FLOAT_PARSE_BASE_AUTOMATIC, STATUS_INPUT_INVALID
	}
	if len(source) == bytes.SLICE_SIZE_MINIMUM {
		return FLOAT_PARSE_BASE_AUTOMATIC, STATUS_INPUT_INVALID
	}
	if len(source) > FLOAT_PARSE_TEXT_SIZE_MAXIMUM {
		return FLOAT_PARSE_BASE_AUTOMATIC, STATUS_INPUT_INVALID
	}
	workspace.Control[FLOAT_PARSE_SOURCE_COUNT_INDEX] = int64(len(source))
	workspace.Control[FLOAT_PARSE_BASE_INDEX] = int64(validated_base)
	workspace.Control[FLOAT_PARSE_SEPARATOR_INDEX] = bytes.SLICE_SIZE_MINIMUM
	if validated_base == FLOAT_PARSE_BASE_AUTOMATIC {
		workspace.Control[FLOAT_PARSE_SEPARATOR_INDEX] = WORD_COUNT_INCREMENT
	}
	copy(workspace.Source[:len(source)], source)
	precision := destination.Precision
	if precision == FLOAT_PRECISION_MINIMUM {
		precision = WORD_BIT_COUNT
	}
	result := (*Float)(&workspace.Values.Result)
	result.Mode = destination.Mode
	float_set_nonfinite(result, Float_Nonfinite_Form(FLOAT_FORM_ZERO),
		POLARITY_NONNEGATIVE, precision,
	)
	matched, negative := float_parse_infinity(workspace.Source, workspace.Control)
	if matched {
		float_set_nonfinite(
			result, Float_Nonfinite_Form(FLOAT_FORM_INFINITY), negative, precision,
		)
		float_copy_exact(destination, result)
		return FLOAT_PARSE_BASE_AUTOMATIC, STATUS_OK
	}
	scan_status := float_parse_scan(Float_Parse_References{
		Source: workspace.Source, Control: workspace.Control,
		Parse: (*Int_Parse_Workspace)(&workspace.Parse),
	})
	base = FLOAT_PARSE_BASE_DECIMAL
	switch Base(workspace.Control[FLOAT_PARSE_RADIX_INDEX]) {
	case BASE_BINARY:
		base = FLOAT_PARSE_BASE_BINARY
	case BASE_OCTAL:
		base = FLOAT_PARSE_BASE_OCTAL
	case BASE_HEXADECIMAL:
		base = FLOAT_PARSE_BASE_HEXADECIMAL
	}
	if scan_status != Parse_Status(STATUS_OK) {
		return base, scan_status
	}
	arithmetic := float_parse_finite(Float_Parse_Factors{
		Control: workspace.Control, Words: workspace.Parse.Words,
		Integers: &workspace.Integers, Values: (*Float_Parse_Values)(&workspace.Values),
		Division: &workspace.Division,
	})
	if arithmetic != Arithmetic_Status(STATUS_OK) {
		return base, STATUS_VALUE_OVERFLOW
	}
	float_copy_exact(destination, result)
	return base, STATUS_OK
}

func float_parse_finite(
	workspace Float_Parse_Factors,
) (status Arithmetic_Status) {
	defer func() { Arithmetic_Status_Invariants(status, "float_parse_finite.status") }()
	Float_Parse_Factors_Invariants(workspace, "float_parse_finite.workspace")
	prefix := Rat_Parse_Prefix{
		Negative: Rat_Parse_Negative(
			workspace.Control[FLOAT_PARSE_NEGATIVE_INDEX] != 0,
		),
		Prefixed: Rat_Parse_Prefixed(
			workspace.Control[FLOAT_PARSE_PREFIXED_INDEX] != 0,
		),
		Base:  Rat_Parse_Mantissa_Base(workspace.Control[FLOAT_PARSE_RADIX_INDEX]),
		Start: Parse_Text_Index(workspace.Control[FLOAT_PARSE_START_INDEX]),
	}
	mantissa := Rat_Parse_Mantissa{
		Prefix: prefix,
		Count: Word_Count(
			workspace.Control[FLOAT_PARSE_MANTISSA_COUNT_INDEX],
		),
		Fractional_Digits: Rat_Parse_Fractional_Digit_Count(
			workspace.Control[FLOAT_PARSE_FRACTIONAL_DIGIT_COUNT_INDEX],
		),
		End: Rat_Parse_Source_Index(workspace.Control[FLOAT_PARSE_END_INDEX]),
	}
	result := (*Float)(&workspace.Values.Result)
	if mantissa.Count == Word_Count(WORD_COUNT_MINIMUM) {
		negative := POLARITY_NONNEGATIVE
		if bool(prefix.Negative) {
			negative = POLARITY_NEGATIVE
		}
		float_set_nonfinite(
			result, Float_Nonfinite_Form(FLOAT_FORM_ZERO), negative,
			result.Precision,
		)
		return STATUS_OK
	}
	exponents, arithmetic := rat_parse_exponents(
		mantissa,
		Rat_Parse_Exponent_Base(
			workspace.Control[FLOAT_PARSE_EXPONENT_BASE_INDEX],
		),
		Rat_Parse_Exponent(workspace.Control[FLOAT_PARSE_EXPONENT_INDEX]),
	)
	if arithmetic != Arithmetic_Status(STATUS_OK) {
		return arithmetic
	}
	workspace.Control[FLOAT_PARSE_EXPONENT_2_INDEX] =
		int64(exponents.Binary)
	workspace.Control[FLOAT_PARSE_EXPONENT_5_INDEX] =
		int64(exponents.Five)
	return float_parse_quotient(Float_Parse_Quotient{
		Control: workspace.Control, Words: workspace.Words,
		Numerator:   (*Nonempty_Int)(&workspace.Integers.Numerator),
		Denominator: &workspace.Integers.Denominator,
		Values:      workspace.Values, Division: workspace.Division,
	})
}

func float_parse_quotient(
	workspace Float_Parse_Quotient,
) (status Arithmetic_Status) {
	defer func() { Arithmetic_Status_Invariants(status, "float_parse_quotient.status") }()
	Float_Parse_Quotient_Invariants(workspace, "float_parse_quotient.workspace")
	numerator := (*Int)((*Nonempty_Int)(workspace.Numerator))
	denominator := (*Int)((*Rat_Parse_Denominator)(workspace.Denominator))
	mantissa_count := int(workspace.Control[FLOAT_PARSE_MANTISSA_COUNT_INDEX])
	previous_count := numerator.Count
	for index := WORD_COUNT_MINIMUM; index < mantissa_count; index++ {
		numerator.Words[index] = workspace.Words[index]
	}
	numerator.Count = Word_Count(mantissa_count)
	numerator.Negative = POLARITY_NONNEGATIVE
	int_clear(numerator, numerator.Count, previous_count)
	if workspace.Control[FLOAT_PARSE_NEGATIVE_INDEX] != 0 {
		numerator.Negative = POLARITY_NEGATIVE
	}
	int_set_word(denominator, Word(bits.CARRY_MAXIMUM), POLARITY_NONNEGATIVE)
	exponent_5 := workspace.Control[FLOAT_PARSE_EXPONENT_5_INDEX]
	if exponent_5 < -int64(RAT_PARSE_EXPONENT_MAGNITUDE_MAXIMUM) {
		return STATUS_VALUE_OVERFLOW
	}
	if exponent_5 > int64(RAT_PARSE_EXPONENT_MAGNITUDE_MAXIMUM) {
		return STATUS_VALUE_OVERFLOW
	}
	value := numerator
	if exponent_5 < 0 {
		value = denominator
		exponent_5 = -exponent_5
	}
	if exponent_5 > 0 {
		arithmetic := rat_parse_multiply_five(
			(*Nonzero_Int)(value), Rat_Parse_Power_Count(exponent_5),
		)
		if arithmetic != Arithmetic_Status(STATUS_OK) {
			return STATUS_VALUE_OVERFLOW
		}
	}
	left := (*Float)(&workspace.Values.Numerator)
	right := (*Float)(&workspace.Values.Denominator)
	// Reset conversion policy without discarding caller-owned mantissa buffers.
	*left = Float{Mantissa: Float_Mantissa{Words: left.Mantissa.Words}}
	*right = Float{Mantissa: Float_Mantissa{Words: right.Mantissa.Words}}
	Float_Set_Int(left, numerator)
	Float_Set_Int(right, denominator)
	result := (*Float)(&workspace.Values.Result)
	quotient_status := Float_Quotient(
		result, left, right,
		(*Float_Division_Workspace)((*Float_Parse_Division_Workspace)(workspace.Division)),
	)
	aver.Always(
		quotient_status == Validation_Status(STATUS_OK),
		"Float parse exact denominator is positive and nonzero.",
	)
	aver.Always(result.Form == FLOAT_FORM_FINITE,
		"Full-width exact factors keep the unscaled quotient finite.")
	exponent_2 := workspace.Control[FLOAT_PARSE_EXPONENT_2_INDEX]
	minimum_shift := int64(FLOAT_EXPONENT_MINIMUM) - int64(result.Exponent)
	maximum_shift := int64(FLOAT_EXPONENT_MAXIMUM) - int64(result.Exponent)
	if exponent_2 < minimum_shift {
		return STATUS_VALUE_OVERFLOW
	}
	if exponent_2 > maximum_shift {
		return STATUS_VALUE_OVERFLOW
	}
	result.Exponent = Float_Exponent(int64(result.Exponent) + exponent_2)
	return STATUS_OK
}

func float_parse_infinity(
	source Float_Parse_Source, control Float_Parse_Control,
) (matched Boolean, negative Polarity) {
	defer func() {
		Boolean_Invariants(matched, "float_parse_infinity.matched")
		Polarity_Invariants(negative, "float_parse_infinity.negative")
	}()
	Float_Parse_Source_Invariants(source, "float_parse_infinity.source")
	Float_Parse_Control_Invariants(control, "float_parse_infinity.control")
	end_index := int(control[FLOAT_PARSE_SOURCE_COUNT_INDEX])
	index := bytes.SLICE_SIZE_MINIMUM
	negative = POLARITY_NONNEGATIVE
	if source[index] == '-' {
		negative = POLARITY_NEGATIVE
		index++
	} else if source[index] == '+' {
		index++
	}
	remaining_count := end_index - index
	if remaining_count != len("Inf") {
		return false, negative
	}
	matched = Boolean(source[index] == 'I')
	if matched {
		matched = Boolean(source[index+WORD_COUNT_INCREMENT] == 'n')
		if matched {
			matched = Boolean(source[index+BASE_BINARY] == 'f')
		}
	}
	if !matched {
		matched = Boolean(source[index] == 'i')
		if matched {
			matched = Boolean(source[index+WORD_COUNT_INCREMENT] == 'n')
			if matched {
				matched = Boolean(source[index+BASE_BINARY] == 'f')
			}
		}
	}
	return matched, negative
}

func float_parse_scan(
	workspace Float_Parse_References,
) (status Parse_Status) {
	defer func() { Parse_Status_Invariants(status, "float_parse_scan.status") }()
	Float_Parse_References_Invariants(workspace, "float_parse_scan.workspace")
	float_parse_scan_prefix(workspace.Source, workspace.Control)
	mantissa_status := float_parse_scan_mantissa(workspace)
	if mantissa_status == Parse_Status(STATUS_INPUT_INVALID) {
		return mantissa_status
	}
	workspace.Control[FLOAT_PARSE_EXPONENT_INDEX] = 0
	workspace.Control[FLOAT_PARSE_EXPONENT_BASE_INDEX] = BASE_DECIMAL
	end_index := int(workspace.Control[FLOAT_PARSE_END_INDEX])
	source_count := int(workspace.Control[FLOAT_PARSE_SOURCE_COUNT_INDEX])
	if end_index < source_count {
		character := workspace.Source[end_index]
		switch character {
		case 'e', 'E':
			workspace.Control[FLOAT_PARSE_EXPONENT_BASE_INDEX] = BASE_DECIMAL
		case 'p', 'P':
			workspace.Control[FLOAT_PARSE_EXPONENT_BASE_INDEX] = BASE_BINARY
		default:
			return STATUS_INPUT_INVALID
		}
		workspace.Control[FLOAT_PARSE_EXPONENT_START_INDEX] = int64(
			end_index + RAT_PARSE_EXPONENT_MARKER_BYTE_COUNT,
		)
		validation := float_parse_scan_exponent(workspace.Source, workspace.Control)
		if validation != Validation_Status(STATUS_OK) {
			return STATUS_INPUT_INVALID
		}
	}
	return mantissa_status
}

func float_parse_scan_prefix(storage Float_Parse_Source, control Float_Parse_Control) {
	Float_Parse_Source_Invariants(storage, "float_parse_scan_prefix.storage")
	Float_Parse_Control_Invariants(control, "float_parse_scan_prefix.control")
	end_index := int(control[FLOAT_PARSE_SOURCE_COUNT_INDEX])
	source := storage[:end_index]
	requested := Float_Parse_Base(control[FLOAT_PARSE_BASE_INDEX])
	prefix := Rat_Parse_Prefix{Base: Rat_Parse_Mantissa_Base(BASE_DECIMAL)}
	index := bytes.SLICE_SIZE_MINIMUM
	if source[index] == '-' {
		prefix.Negative = true
		index++
	} else if source[index] == '+' {
		index++
	}
	prefix.Start = Parse_Text_Index(index)
	if requested == FLOAT_PARSE_BASE_AUTOMATIC {
		prefix = rat_parse_prefix(Rat_Parse_Text(source))
	} else {
		switch requested {
		case FLOAT_PARSE_BASE_BINARY:
			prefix.Base = Rat_Parse_Mantissa_Base(BASE_BINARY)
		case FLOAT_PARSE_BASE_OCTAL:
			prefix.Base = Rat_Parse_Mantissa_Base(BASE_OCTAL)
		case FLOAT_PARSE_BASE_DECIMAL:
			prefix.Base = Rat_Parse_Mantissa_Base(BASE_DECIMAL)
		case FLOAT_PARSE_BASE_HEXADECIMAL:
			prefix.Base = Rat_Parse_Mantissa_Base(BASE_HEXADECIMAL)
		}
	}
	control[FLOAT_PARSE_NEGATIVE_INDEX] = 0
	if prefix.Negative {
		control[FLOAT_PARSE_NEGATIVE_INDEX] = WORD_COUNT_INCREMENT
	}
	control[FLOAT_PARSE_PREFIXED_INDEX] = 0
	if prefix.Prefixed {
		control[FLOAT_PARSE_PREFIXED_INDEX] = WORD_COUNT_INCREMENT
	}
	control[FLOAT_PARSE_RADIX_INDEX] = int64(prefix.Base)
	control[FLOAT_PARSE_START_INDEX] = int64(prefix.Start)
}

func float_parse_scan_mantissa(workspace Float_Parse_References) (status Parse_Status) {
	defer func() { Parse_Status_Invariants(status, "float_parse_scan_mantissa.status") }()
	Float_Parse_References_Invariants(workspace, "float_parse_scan_mantissa.workspace")
	// Reuse caller-owned parse storage across scans.
	clear(workspace.Parse.Words)
	end_index := int(workspace.Control[FLOAT_PARSE_SOURCE_COUNT_INDEX])
	radix := Base(workspace.Control[FLOAT_PARSE_RADIX_INDEX])
	word_count := Word_Count(WORD_COUNT_MINIMUM)
	digit_seen, radix_seen, separator_seen, magnitude_overflow :=
		false, false, false, false
	previous_digit := workspace.Control[FLOAT_PARSE_PREFIXED_INDEX] != 0
	index := int(workspace.Control[FLOAT_PARSE_START_INDEX])
	separators := workspace.Control[FLOAT_PARSE_SEPARATOR_INDEX] != 0
	fractional_digit_count := int64(bytes.SLICE_SIZE_MINIMUM)
	parse := workspace.Parse
	for index < end_index {
		character := workspace.Source[index]
		digit, valid := int_parse_digit(Parse_Character(character), radix)
		if valid {
			if !magnitude_overflow {
				arithmetic := int_parse_accumulate(parse, &word_count, radix, digit)
				magnitude_overflow = arithmetic != Arithmetic_Status(STATUS_OK)
			}
			if radix_seen {
				fractional_digit_count++
			}
			digit_seen, separator_seen, previous_digit = true, false, true
			index++
			continue
		}
		if character == '_' {
			if !separators {
				return STATUS_INPUT_INVALID
			}
			if !previous_digit {
				return STATUS_INPUT_INVALID
			}
			if separator_seen {
				return STATUS_INPUT_INVALID
			}
			separator_seen, previous_digit = true, false
			index++
			continue
		}
		if character == '.' {
			if radix_seen {
				return STATUS_INPUT_INVALID
			}
			if separator_seen {
				return STATUS_INPUT_INVALID
			}
			radix_seen, previous_digit = true, false
			index++
			continue
		}
		break
	}
	workspace.Control[FLOAT_PARSE_MANTISSA_COUNT_INDEX] = int64(word_count)
	workspace.Control[FLOAT_PARSE_FRACTIONAL_DIGIT_COUNT_INDEX] =
		fractional_digit_count
	workspace.Control[FLOAT_PARSE_END_INDEX] = int64(index)
	return float_parse_scan_mantissa_status(
		Boolean(digit_seen), Boolean(separator_seen), Boolean(magnitude_overflow),
	)
}

func float_parse_scan_mantissa_status(
	digit_seen Boolean, separator_seen Boolean, magnitude_overflow Boolean,
) (status Parse_Status) {
	defer func() {
		Parse_Status_Invariants(status, "float_parse_scan_mantissa_status.status")
	}()
	Boolean_Invariants(digit_seen, "float_parse_scan_mantissa_status.digit_seen")
	Boolean_Invariants(separator_seen, "float_parse_scan_mantissa_status.separator_seen")
	Boolean_Invariants(
		magnitude_overflow, "float_parse_scan_mantissa_status.magnitude_overflow",
	)
	if !digit_seen {
		return STATUS_INPUT_INVALID
	}
	if separator_seen {
		return STATUS_INPUT_INVALID
	}
	if magnitude_overflow {
		return STATUS_VALUE_OVERFLOW
	}
	return STATUS_OK
}

func float_parse_scan_exponent(
	source Float_Parse_Source, control Float_Parse_Control,
) (status Validation_Status) {
	defer func() {
		Validation_Status_Invariants(status, "float_parse_scan_exponent.status")
	}()
	Float_Parse_Source_Invariants(source, "float_parse_scan_exponent.source")
	Float_Parse_Control_Invariants(control, "float_parse_scan_exponent.control")
	end_index := int(control[FLOAT_PARSE_SOURCE_COUNT_INDEX])
	index := int(control[FLOAT_PARSE_EXPONENT_START_INDEX])
	negative := false
	if index < end_index {
		if source[index] == '-' {
			negative = true
			index++
		} else if source[index] == '+' {
			index++
		}
	}
	limit := uint64(bits.INTEGER_64_MAXIMUM)
	if negative {
		limit = INT_64_NEGATIVE_MAGNITUDE_MAXIMUM
	}
	magnitude := uint64(bits.WORD_64_MINIMUM)
	digit_seen, separator_seen := false, false
	separators := control[FLOAT_PARSE_SEPARATOR_INDEX] != 0
	for index < end_index {
		character := source[index]
		if character == '_' {
			if !separators {
				return STATUS_INPUT_INVALID
			}
			if !digit_seen {
				return STATUS_INPUT_INVALID
			}
			if separator_seen {
				return STATUS_INPUT_INVALID
			}
			separator_seen = true
			index++
			continue
		}
		digit_value, valid := int_parse_digit(Parse_Character(character), BASE_DECIMAL)
		if !valid {
			return STATUS_INPUT_INVALID
		}
		digit := uint64(digit_value)
		if magnitude > (limit-digit)/uint64(BASE_DECIMAL) {
			return STATUS_INPUT_INVALID
		}
		magnitude = magnitude*uint64(BASE_DECIMAL) + digit
		digit_seen, separator_seen = true, false
		index++
	}
	if !digit_seen {
		return STATUS_INPUT_INVALID
	}
	if separator_seen {
		return STATUS_INPUT_INVALID
	}
	exponent := int64(magnitude)
	if negative {
		if magnitude == INT_64_NEGATIVE_MAGNITUDE_MAXIMUM {
			exponent = bits.INTEGER_64_MINIMUM
		} else {
			exponent = -int64(magnitude)
		}
	}
	control[FLOAT_PARSE_EXPONENT_INDEX] = exponent
	return STATUS_OK
}

// Carry reservation makes two words the smallest addition normalization span.
const FLOAT_ADDITION_RESULT_WORD_COUNT_MINIMUM = WORD_COUNT_INCREMENT + WORD_COUNT_INCREMENT

// Float_Addition_Result_Word_Count includes carry storage before high zeros are removed.
type Float_Addition_Result_Word_Count int

// Float_Addition_Result_Word_Count_Invariants preserves that reserved carry word.
func Float_Addition_Result_Word_Count_Invariants(
	value Float_Addition_Result_Word_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), FLOAT_ADDITION_RESULT_WORD_COUNT_MINIMUM,
			FLOAT_ADDITION_WORD_COUNT_MAXIMUM,
		).
		Ensure()
}

// Rat_Set_Float_32_Bits stores exact finite binary32 and rejects nonfinite input.
func Rat_Set_Float_32_Bits(
	destination Rat_Handle, encoding Float_32_Bits,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "rat_set_float_32_bits.status") }()
	Rat_Handle_Invariants(destination, "rat_set_float_32_bits.destination")
	Float_32_Bits_Invariants(encoding, "rat_set_float_32_bits.encoding")
	encoded := uint32(encoding)
	encoded_exponent := encoded >> FLOAT_32_EXPONENT_SHIFT & FLOAT_32_EXPONENT_MASK
	if encoded_exponent == FLOAT_32_EXPONENT_MASK {
		return STATUS_INPUT_INVALID
	}
	mantissa := encoded & FLOAT_32_MANTISSA_MASK
	exponent := FLOAT_32_SUBNORMAL_EXPONENT
	if encoded_exponent != bits.WORD_32_MINIMUM {
		mantissa |= FLOAT_32_HIDDEN_MANTISSA_BIT
		exponent = int(encoded_exponent) - FLOAT_32_EXPONENT_BIAS
	}
	if mantissa == bits.WORD_32_MINIMUM {
		Rat_Set_Uint_64(destination, Word_64(bits.WORD_64_MINIMUM))
		return STATUS_OK
	}
	shift := FLOAT_32_MANTISSA_BIT_COUNT - exponent
	for mantissa&uint32(bits.CARRY_MAXIMUM) == bits.WORD_32_MINIMUM && shift > 0 {
		mantissa >>= 1
		shift--
	}
	numerator := (*Int)(&destination.Integers.Numerator)
	denominator := (*Int)(&destination.Integers.Denominator)
	Int_Set_Uint_64(numerator, Word_64(mantissa))
	// Preserve caller storage for denominator construction and later reuse.
	Int_Set_Uint_64(denominator, Word_64(bits.WORD_64_MINIMUM))
	if shift > 0 {
		Int_Set_Uint_64(denominator, Word_64(bits.CARRY_MAXIMUM))
		shift_status := Int_Shift_Left(denominator, denominator, Shift_Count(shift))
		aver.Always(
			shift_status == Arithmetic_Status(STATUS_OK),
			"Binary32 denominator fits rational component storage.",
		)
	} else {
		shift_status := Int_Shift_Left(numerator, numerator, Shift_Count(-shift))
		aver.Always(
			shift_status == Arithmetic_Status(STATUS_OK),
			"Binary32 numerator fits rational component storage.",
		)
	}
	if encoded&FLOAT_32_SIGN_MASK != bits.WORD_32_MINIMUM {
		numerator.Negative = POLARITY_NEGATIVE
	}
	return STATUS_OK
}

// Rat_Float_32_Bits returns nearest numeric binary32 using round-to-even.
func Rat_Float_32_Bits(
	value Rat_Handle, workspace Rat_Float_32_Workspace_Handle,
) (encoding Float_32_Value_Bits, exact Boolean) {
	defer func() {
		Float_32_Value_Bits_Invariants(encoding, "rat_float_32_bits.encoding")
		Boolean_Invariants(exact, "rat_float_32_bits.exact")
	}()
	Rat_Handle_Invariants(value, "rat_float_32_bits.value")
	Rat_Float_32_Workspace_Handle_Invariants(workspace, "rat_float_32_bits.workspace")
	numerator_source := (*Int)(&value.Integers.Numerator)
	if numerator_source.Count == WORD_COUNT_MINIMUM {
		return Float_32_Value_Bits(FLOAT_32_VALUE_BITS_MINIMUM), Boolean(true)
	}
	denominator_source := (*Int)(&value.Integers.Denominator)
	exponent := int(Int_Bit_Count(numerator_source)) - WORD_COUNT_INCREMENT
	if denominator_source.Count != WORD_COUNT_MINIMUM {
		exponent += WORD_COUNT_INCREMENT - int(Int_Bit_Count(denominator_source))
	}
	rat_float_32_divide((*Nonzero_Rat)((*Rat)(value)), workspace)
	quotient := (*Int)(&workspace.Integers.Quotient)
	remainder := (*Int)(&workspace.Integers.Remainder)
	quotient_word, conversion_status := Int_Uint_64(quotient)
	quotient_fits := conversion_status == Conversion_Status(STATUS_OK) &&
		uint64(quotient_word) <= uint64(bits.WORD_32_MAXIMUM)
	aver.Always(quotient_fits, "Binary32 rounding quotient fits one word.")
	mantissa := uint32(quotient_word)
	have_remainder := remainder.Count != WORD_COUNT_MINIMUM
	if mantissa>>FLOAT_32_ROUNDING_MANTISSA_BIT_COUNT == uint32(bits.CARRY_MAXIMUM) {
		if mantissa&uint32(bits.CARRY_MAXIMUM) != bits.WORD_32_MINIMUM {
			have_remainder = true
		}
		mantissa >>= 1
		exponent++
	}
	aver.Always(
		mantissa>>FLOAT_32_VALUE_MANTISSA_BIT_COUNT == uint32(bits.CARRY_MAXIMUM),
		"Scaled binary32 rational quotient retains one normal leading bit.",
	)
	subnormal := false
	if exponent >= FLOAT_32_SUBNORMAL_EXPONENT_MINIMUM {
		if exponent <= FLOAT_32_SUBNORMAL_EXPONENT {
			subnormal = true
			shift := uint(FLOAT_32_SUBNORMAL_EXPONENT - exponent + WORD_COUNT_INCREMENT)
			lost_mask := uint32(bits.CARRY_MAXIMUM)<<shift - uint32(bits.CARRY_MAXIMUM)
			lost := mantissa&lost_mask != bits.WORD_32_MINIMUM
			have_remainder = have_remainder || lost
			mantissa >>= shift
			exponent = BASE_BINARY - FLOAT_32_EXPONENT_BIAS
		}
	}
	exact = Boolean(!have_remainder)
	rounded, exact, rounded_exponent := rat_float_32_round(
		Rat_Float_32_Rounding_Mantissa(mantissa), exact, Boolean(have_remainder),
		Rat_Float_32_Rounding_Exponent(exponent),
	)
	return rat_float_32_encode(
		rounded, rounded_exponent,
		Boolean(subnormal), numerator_source.Negative, exact,
	)
}

func rat_float_32_round(
	mantissa Rat_Float_32_Rounding_Mantissa,
	exact Boolean,
	have_remainder Boolean,
	exponent Rat_Float_32_Rounding_Exponent,
) (
	result Rat_Float_32_Mantissa,
	result_exact Boolean,
	result_exponent Rat_Float_32_Exponent,
) {
	defer func() {
		Rat_Float_32_Mantissa_Invariants(result, "rat_float_32_round.result")
		Boolean_Invariants(result_exact, "rat_float_32_round.result_exact")
		Rat_Float_32_Exponent_Invariants(
			result_exponent, "rat_float_32_round.result_exponent",
		)
	}()
	Rat_Float_32_Rounding_Mantissa_Invariants(mantissa, "rat_float_32_round.mantissa")
	Boolean_Invariants(exact, "rat_float_32_round.exact")
	Boolean_Invariants(have_remainder, "rat_float_32_round.have_remainder")
	Rat_Float_32_Rounding_Exponent_Invariants(exponent, "rat_float_32_round.exponent")
	rounded := uint32(mantissa)
	result_exact = exact
	result_exponent = Rat_Float_32_Exponent(exponent)
	if rounded&uint32(bits.CARRY_MAXIMUM) == bits.WORD_32_MINIMUM {
		result = Rat_Float_32_Mantissa(rounded >> WORD_COUNT_INCREMENT)
		return result, result_exact, result_exponent
	}
	result_exact = Boolean(false)
	round_up := bool(have_remainder) ||
		rounded&uint32(BASE_BINARY) != bits.WORD_32_MINIMUM
	if round_up {
		rounded++
		if rounded >= FLOAT_32_ROUNDING_MANTISSA_LIMIT {
			rounded >>= 1
			result_exponent++
		}
	}
	result = Rat_Float_32_Mantissa(rounded >> WORD_COUNT_INCREMENT)
	return result, result_exact, result_exponent
}

func rat_float_32_encode(
	mantissa Rat_Float_32_Mantissa,
	exponent Rat_Float_32_Exponent,
	subnormal Boolean,
	negative Polarity,
	exact Boolean,
) (encoding Float_32_Value_Bits, result_exact Boolean) {
	defer func() {
		Float_32_Value_Bits_Invariants(encoding, "rat_float_32_encode.encoding")
		Boolean_Invariants(result_exact, "rat_float_32_encode.result_exact")
	}()
	Rat_Float_32_Mantissa_Invariants(mantissa, "rat_float_32_encode.mantissa")
	Rat_Float_32_Exponent_Invariants(exponent, "rat_float_32_encode.exponent")
	Boolean_Invariants(subnormal, "rat_float_32_encode.subnormal")
	Polarity_Invariants(negative, "rat_float_32_encode.negative")
	Boolean_Invariants(exact, "rat_float_32_encode.exact")
	result_exact = exact
	encoded := uint32(FLOAT_32_VALUE_BITS_MINIMUM)
	if int(exponent) >= FLOAT_32_SUBNORMAL_EXPONENT_MINIMUM {
		encoded_exponent := int(exponent) + FLOAT_32_EXPONENT_ENCODING_OFFSET
		if bool(subnormal) {
			encoded_exponent = WORD_COUNT_MINIMUM
			if uint32(mantissa) >= FLOAT_32_HIDDEN_MANTISSA_BIT {
				encoded_exponent++
			}
		}
		if encoded_exponent >= FLOAT_32_EXPONENT_MASK {
			encoded = FLOAT_32_POSITIVE_INFINITY_BITS
			result_exact = Boolean(false)
		} else {
			encoded = uint32(encoded_exponent)<<FLOAT_32_EXPONENT_SHIFT |
				uint32(mantissa)&FLOAT_32_MANTISSA_MASK
		}
	}
	if negative == POLARITY_NEGATIVE {
		encoded |= FLOAT_32_SIGN_MASK
	}
	return Float_32_Value_Bits(encoded), result_exact
}

func rat_float_32_divide(value Nonzero_Rat_Handle, workspace Rat_Float_32_Workspace_Handle) {
	Nonzero_Rat_Handle_Invariants(value, "rat_float_32_divide.value")
	Rat_Float_32_Workspace_Handle_Invariants(workspace, "rat_float_32_divide.workspace")
	numerator := (*Int)(&workspace.Integers.Numerator)
	denominator := (*Int)(&workspace.Integers.Denominator)
	quotient := (*Int)(&workspace.Integers.Quotient)
	remainder := (*Int)(&workspace.Integers.Remainder)
	Int_Absolute(numerator, (*Int)(&value.Integers.Numerator))
	Rat_Denominator_Into((*Nonempty_Int)(denominator), (*Rat)((*Nonzero_Rat)(value)))
	exponent := int(Int_Bit_Count(numerator)) - int(Int_Bit_Count(denominator))
	shift := FLOAT_32_ROUNDING_MANTISSA_BIT_COUNT - exponent
	shift_status := Arithmetic_Status(STATUS_OK)
	if shift > WORD_COUNT_MINIMUM {
		shift_status = Int_Shift_Left(numerator, numerator, Shift_Count(shift))
	} else if shift < WORD_COUNT_MINIMUM {
		shift_status = Int_Shift_Left(denominator, denominator, Shift_Count(-shift))
	}
	aver.Always(
		shift_status == Arithmetic_Status(STATUS_OK),
		"Rational component bound leaves full Int room for binary32 scaling.",
	)
	division := (*Int_Division_Workspace)(&workspace.Division)
	division_status := Int_Quotient_Remainder(
		quotient, remainder, numerator, denominator, division,
	)
	aver.Always(
		division_status == Division_Status(STATUS_OK),
		"Normalized rational denominator remains nonzero during binary32 conversion.",
	)
}

// Float_Set_Float_32_Bits avoids native float conversion and its hidden rounding.
func Float_Set_Float_32_Bits(
	destination Float_Handle, encoding Float_32_Bits,
) (status Validation_Status) {
	defer func() {
		Validation_Status_Invariants(status, "float_set_float_32_bits.status")
	}()
	Float_Handle_Invariants(destination, "float_set_float_32_bits.destination_initial")
	Float_32_Bits_Invariants(encoding, "float_set_float_32_bits.encoding")
	encoded := uint32(encoding)
	exponent_field := encoded >> FLOAT_32_EXPONENT_SHIFT & FLOAT_32_EXPONENT_MASK
	mantissa_field := encoded & FLOAT_32_MANTISSA_MASK
	if exponent_field == FLOAT_32_EXPONENT_MASK {
		if mantissa_field != bits.WORD_32_MINIMUM {
			return STATUS_INPUT_INVALID
		}
	}
	negative := Polarity(
		(encoded >> FLOAT_32_SIGN_SHIFT) & uint32(bits.CARRY_MAXIMUM),
	)
	precision := destination.Precision
	if precision == FLOAT_PRECISION_MINIMUM {
		precision = FLOAT_32_VALUE_MANTISSA_BIT_COUNT
	}
	if exponent_field == FLOAT_32_EXPONENT_MASK {
		float_set_nonfinite(
			destination, Float_Nonfinite_Form(FLOAT_FORM_INFINITY), negative, precision,
		)
		return STATUS_OK
	}
	if exponent_field == bits.WORD_32_MINIMUM {
		if mantissa_field == bits.WORD_32_MINIMUM {
			float_set_nonfinite(
				destination, Float_Nonfinite_Form(FLOAT_FORM_ZERO), negative,
				precision,
			)
			return STATUS_OK
		}
	}
	float_set_float_32_finite(
		(*Float_Nonempty)((*Float)(destination)),
		Float_32_Finite_Exponent_Field(exponent_field),
		Float_32_Mantissa_Field(mantissa_field), negative,
		Float_Active_Precision(precision),
	)
	return STATUS_OK
}

// Float_Float_32_Bits prevents host floating-point policy from changing IEEE rounding.
func Float_Float_32_Bits(value Float_Handle) (
	encoding Float_32_Value_Bits, accuracy Accuracy,
) {
	defer func() {
		Float_32_Value_Bits_Invariants(encoding, "float_float_32_bits.encoding")
		Accuracy_Invariants(accuracy, "float_float_32_bits.accuracy")
	}()
	Float_Handle_Invariants(value, "float_float_32_bits.value")
	sign := uint32(bits.WORD_32_MINIMUM)
	if value.Negative == POLARITY_NEGATIVE {
		sign = FLOAT_32_SIGN_MASK
	}
	if value.Form == FLOAT_FORM_ZERO {
		return Float_32_Value_Bits(sign), ACCURACY_EXACT
	}
	if value.Form == FLOAT_FORM_INFINITY {
		return Float_32_Value_Bits(sign | FLOAT_32_POSITIVE_INFINITY_BITS),
			ACCURACY_EXACT
	}
	return float_finite_float_32_bits(
		(*Float_Finite)((*Float)(value)), Float_32_Sign(sign))
}

func float_binary_round(
	value Positive_Int, exponent Float_Binary_Source_Exponent, negative Polarity,
	precision Float_Binary_Precision,
) (result Float_Binary_Result) {
	defer func() {
		Float_Binary_Result_Invariants(result, "float_binary_round.result")
	}()
	Positive_Int_Invariants(value, "float_binary_round.value")
	Float_Binary_Source_Exponent_Invariants(exponent, "float_binary_round.exponent")
	Polarity_Invariants(negative, "float_binary_round.negative")
	Float_Binary_Precision_Invariants(precision, "float_binary_round.precision")
	result.Exponent = Float_Binary_Exponent(exponent)
	result.Accuracy = ACCURACY_EXACT
	source_count := int(value.Count)
	high := value.Words[source_count-WORD_COUNT_INCREMENT]
	bit_count := (source_count-WORD_COUNT_INCREMENT)*WORD_BIT_COUNT +
		int(bits.Bit_Size_64(bits.Word_64(high)))
	discard_count := bit_count - int(precision)
	if discard_count <= BIT_COUNT_MINIMUM {
		result.Mantissa = Float_Binary_Mantissa(
			value.Words[WORD_COUNT_MINIMUM],
		)
		return result
	}
	word_shift := discard_count / WORD_BIT_COUNT
	bit_shift := uint(discard_count % WORD_BIT_COUNT)
	result.Mantissa = Float_Binary_Mantissa(
		value.Words[word_shift] >> bit_shift,
	)
	next_index := word_shift + WORD_COUNT_INCREMENT
	if bit_shift != BIT_COUNT_MINIMUM {
		if next_index < int(value.Count) {
			result.Mantissa |= Float_Binary_Mantissa(
				value.Words[next_index] << (WORD_BIT_COUNT - bit_shift),
			)
		}
	}
	rounding_index := discard_count - WORD_COUNT_INCREMENT
	rounding_word := value.Words[rounding_index/WORD_BIT_COUNT]
	rounding_bit := Bit_Value(
		rounding_word >> uint(rounding_index%WORD_BIT_COUNT) & Word(BIT_SET),
	)
	sticky_bit := BIT_CLEAR
	if float_low_bits_nonzero(
		&value,
		Float_Discarded_Bit_Count(rounding_index),
	) {
		sticky_bit = BIT_SET
	}
	increment := Boolean(false)
	if rounding_bit != BIT_CLEAR {
		if sticky_bit != BIT_CLEAR {
			increment = true
		} else if result.Mantissa&Float_Binary_Mantissa(BIT_SET) != 0 {
			increment = true
		}
		result.Accuracy = Accuracy(float_accuracy(increment, negative))
	} else if sticky_bit != BIT_CLEAR {
		result.Accuracy = Accuracy(float_accuracy(increment, negative))
	}
	if increment {
		result.Mantissa++
	}
	if int(bits.Bit_Size_64(bits.Word_64(result.Mantissa))) > int(precision) {
		result.Mantissa >>= WORD_COUNT_INCREMENT
		result.Exponent++
	}
	return result
}

func float_set_float_32_finite(
	reference Float_Nonempty_Handle,
	exponent_field Float_32_Finite_Exponent_Field,
	mantissa_field Float_32_Mantissa_Field,
	negative Polarity,
	precision Float_Active_Precision,
) {
	Float_Nonempty_Handle_Invariants(reference, "float_set_float_32_finite.destination_initial")
	Float_32_Finite_Exponent_Field_Invariants(
		exponent_field, "float_set_float_32_finite.exponent_field",
	)
	Float_32_Mantissa_Field_Invariants(
		mantissa_field, "float_set_float_32_finite.mantissa_field",
	)
	Polarity_Invariants(negative, "float_set_float_32_finite.negative")
	Float_Active_Precision_Invariants(
		precision, "float_set_float_32_finite.precision",
	)
	destination := (*Float)((*Float_Nonempty)(reference))
	mantissa := uint32(mantissa_field)
	exponent := FLOAT_32_SUBNORMAL_EXPONENT - FLOAT_32_MANTISSA_BIT_COUNT
	if exponent_field != Float_32_Finite_Exponent_Field(bits.WORD_32_MINIMUM) {
		mantissa |= FLOAT_32_HIDDEN_MANTISSA_BIT
		exponent = int(exponent_field) - FLOAT_32_EXPONENT_BIAS +
			WORD_COUNT_INCREMENT
	} else {
		mantissa_bit_count := bits.BIT_COUNT_32_MAXIMUM -
			int(bits.Leading_Zeros_32(bits.Word_32(mantissa)))
		exponent += mantissa_bit_count
	}
	bit_count := Float_Precision(bits.Bit_Size_32(bits.Word_32(mantissa)))
	if bit_count <= Float_Precision(precision) {
		previous_count := destination.Mantissa.Count
		destination.Precision = Float_Precision(precision)
		destination.Accuracy, destination.Form = ACCURACY_EXACT, FLOAT_FORM_FINITE
		destination.Negative = negative
		destination.Mantissa.Words[WORD_COUNT_MINIMUM] = Word(mantissa)
		destination.Mantissa.Count = Word_Count(WORD_COUNT_INCREMENT)
		destination.Mantissa.Negative = POLARITY_NONNEGATIVE
		destination.Exponent = Float_Exponent(exponent)
		if previous_count > Word_Count(WORD_COUNT_INCREMENT) {
			int_clear(
				(*Int)(&destination.Mantissa), Word_Count(WORD_COUNT_INCREMENT),
				previous_count,
			)
		}
		return
	}
	var float_magnitude Float_Mantissa
	var magnitude_words [WORD_COUNT_MAXIMUM]Word
	float_magnitude.Words = magnitude_words[:]
	float_magnitude.Words[WORD_COUNT_MINIMUM] = Word(mantissa)
	float_magnitude.Count = Word_Count(WORD_COUNT_INCREMENT)
	float_set_magnitude(
		destination, &float_magnitude, Float_Exponent(exponent), negative,
		precision, destination.Mode,
	)
}

func float_finite_float_64_bits(
	value Float_Finite_Handle, sign Float_64_Sign,
) (encoding Float_64_Value_Bits, accuracy Accuracy) {
	defer func() {
		Float_64_Value_Bits_Invariants(encoding, "float_finite_float_64_bits.encoding")
		Accuracy_Invariants(accuracy, "float_finite_float_64_bits.accuracy")
	}()
	Float_Finite_Handle_Invariants(value, "float_finite_float_64_bits.value")
	Float_64_Sign_Invariants(sign, "float_finite_float_64_bits.sign")
	if value.Exponent > FLOAT_64_VALUE_BIT_COUNT_MAXIMUM {
		return Float_64_Value_Bits(uint64(sign) | FLOAT_64_POSITIVE_INFINITY_BITS),
			Accuracy(float_accuracy(true, value.Negative))
	}
	normal_minimum := FLOAT_64_SUBNORMAL_EXPONENT + WORD_COUNT_INCREMENT
	precision := FLOAT_64_VALUE_MANTISSA_BIT_COUNT
	if int(value.Exponent) < normal_minimum {
		precision = int(value.Exponent) + FLOAT_64_SUBNORMAL_DENOMINATOR_SHIFT_MAXIMUM
		if precision <= FLOAT_PRECISION_MINIMUM {
			if precision == FLOAT_PRECISION_MINIMUM {
				minimum_precision := Float_Minimum_Precision(
					(*Float)((*Float_Finite)(value)))
				if minimum_precision != WORD_COUNT_INCREMENT {
					return Float_64_Value_Bits(
							uint64(sign) | bits.CARRY_MAXIMUM,
						),
						Accuracy(float_accuracy(true, value.Negative))
				}
			}
			return Float_64_Value_Bits(sign),
				Accuracy(float_accuracy(false, value.Negative))
		}
	}
	rounded := float_binary_round(
		Positive_Int(value.Mantissa), Float_Binary_Source_Exponent(value.Exponent),
		value.Negative, Float_Binary_Precision(precision),
	)
	if rounded.Exponent > FLOAT_64_VALUE_BIT_COUNT_MAXIMUM {
		return Float_64_Value_Bits(uint64(sign) | FLOAT_64_POSITIVE_INFINITY_BITS),
			rounded.Accuracy
	}
	aver.Always(rounded.Exponent > FLOAT_64_SUBNORMAL_EXPONENT-
		FLOAT_64_MANTISSA_BIT_COUNT,
		"Rounded binary64 exponent stays above tie-underflow handling.")
	aver.Always(rounded.Exponent <= FLOAT_64_VALUE_BIT_COUNT_MAXIMUM,
		"Rounded binary64 exponent stays below overflow handling.")
	bit_count := int(bits.Bit_Size_64(bits.Word_64(rounded.Mantissa)))
	mantissa := uint64(rounded.Mantissa)
	if int(rounded.Exponent) < normal_minimum {
		precision = int(rounded.Exponent) + FLOAT_64_SUBNORMAL_DENOMINATOR_SHIFT_MAXIMUM
		mantissa <<= uint(precision - bit_count)
		return Float_64_Value_Bits(uint64(sign) | mantissa), rounded.Accuracy
	}
	mantissa <<= uint(FLOAT_64_VALUE_MANTISSA_BIT_COUNT - bit_count)
	exponent_field := uint64(
		int(rounded.Exponent) - WORD_COUNT_INCREMENT + FLOAT_64_EXPONENT_BIAS,
	)
	encoded := uint64(sign) | exponent_field<<FLOAT_64_EXPONENT_SHIFT |
		mantissa&FLOAT_64_MANTISSA_MASK
	return Float_64_Value_Bits(encoded), rounded.Accuracy
}

func float_finite_float_32_bits(
	value Float_Finite_Handle, sign Float_32_Sign,
) (encoding Float_32_Value_Bits, accuracy Accuracy) {
	defer func() {
		Float_32_Value_Bits_Invariants(
			encoding, "float_finite_float_32_bits.encoding",
		)
		Accuracy_Invariants(accuracy, "float_finite_float_32_bits.accuracy")
	}()
	Float_Finite_Handle_Invariants(value, "float_finite_float_32_bits.value")
	Float_32_Sign_Invariants(sign, "float_finite_float_32_bits.sign")
	if value.Exponent > FLOAT_32_VALUE_BIT_COUNT_MAXIMUM {
		return Float_32_Value_Bits(
			uint32(sign) | FLOAT_32_POSITIVE_INFINITY_BITS,
		), Accuracy(float_accuracy(true, value.Negative))
	}
	normal_minimum := FLOAT_32_SUBNORMAL_EXPONENT + WORD_COUNT_INCREMENT
	precision := FLOAT_32_VALUE_MANTISSA_BIT_COUNT
	if int(value.Exponent) < normal_minimum {
		precision = int(value.Exponent) + FLOAT_32_SUBNORMAL_DENOMINATOR_SHIFT_MAXIMUM
		if precision <= FLOAT_PRECISION_MINIMUM {
			if precision == FLOAT_PRECISION_MINIMUM {
				minimum_precision := Float_Minimum_Precision(
					(*Float)((*Float_Finite)(value)))
				if minimum_precision != WORD_COUNT_INCREMENT {
					return Float_32_Value_Bits(
						uint32(sign) | uint32(bits.CARRY_MAXIMUM),
					), Accuracy(float_accuracy(true, value.Negative))
				}
			}
			return Float_32_Value_Bits(sign),
				Accuracy(float_accuracy(false, value.Negative))
		}
	}
	rounded := float_binary_round(
		Positive_Int(value.Mantissa), Float_Binary_Source_Exponent(value.Exponent),
		value.Negative, Float_Binary_Precision(precision),
	)
	if rounded.Exponent > FLOAT_32_VALUE_BIT_COUNT_MAXIMUM {
		return Float_32_Value_Bits(
			uint32(sign) | FLOAT_32_POSITIVE_INFINITY_BITS,
		), rounded.Accuracy
	}
	aver.Always(
		rounded.Exponent > FLOAT_32_SUBNORMAL_EXPONENT-
			FLOAT_32_MANTISSA_BIT_COUNT,
		"Rounded binary32 exponent stays above tie underflow handling.",
	)
	aver.Always(
		rounded.Exponent <= FLOAT_32_VALUE_BIT_COUNT_MAXIMUM,
		"Rounded binary32 exponent stays below overflow handling.",
	)
	bit_count := int(bits.Bit_Size_64(bits.Word_64(rounded.Mantissa)))
	mantissa := uint32(rounded.Mantissa)
	if int(rounded.Exponent) < normal_minimum {
		precision = int(rounded.Exponent) + FLOAT_32_SUBNORMAL_DENOMINATOR_SHIFT_MAXIMUM
		mantissa <<= uint(precision - bit_count)
		return Float_32_Value_Bits(uint32(sign) | mantissa), rounded.Accuracy
	}
	mantissa <<= uint(FLOAT_32_VALUE_MANTISSA_BIT_COUNT - bit_count)
	exponent_field := uint32(
		int(rounded.Exponent) - WORD_COUNT_INCREMENT + FLOAT_32_EXPONENT_BIAS,
	)
	encoded := uint32(sign) | exponent_field<<FLOAT_32_EXPONENT_SHIFT |
		mantissa&FLOAT_32_MANTISSA_MASK
	return Float_32_Value_Bits(encoded), rounded.Accuracy
}

// Float_Quotient extracts one bounded binary expansion instead of growing a rational result.
func Float_Quotient(
	destination Float_Handle,
	dividend Float_Handle,
	divisor Float_Handle,
	workspace Float_Division_Workspace_Handle,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "float_quotient.status") }()
	Float_Handle_Invariants(destination, "float_quotient.destination_initial")
	Float_Handle_Invariants(dividend, "float_quotient.dividend")
	Float_Handle_Invariants(divisor, "float_quotient.divisor")
	Float_Division_Workspace_Handle_Invariants(workspace, "float_quotient.workspace")
	precision := destination.Precision
	if precision == FLOAT_PRECISION_MINIMUM {
		precision = dividend.Precision
		if divisor.Precision > precision {
			precision = divisor.Precision
		}
	}
	if dividend.Form != FLOAT_FORM_FINITE {
		return float_quotient_general(destination, dividend, divisor, workspace, precision)
	}
	if divisor.Form != FLOAT_FORM_FINITE {
		return float_quotient_general(destination, dividend, divisor, workspace, precision)
	}
	if dividend.Mantissa.Count != Word_Count(WORD_COUNT_INCREMENT) {
		return float_quotient_general(destination, dividend, divisor, workspace, precision)
	}
	if divisor.Mantissa.Count != Word_Count(WORD_COUNT_INCREMENT) {
		return float_quotient_general(destination, dividend, divisor, workspace, precision)
	}
	dividend_exponent := int(dividend.Exponent)
	divisor_exponent := int(divisor.Exponent)
	if uint(dividend_exponent-WORD_COUNT_INCREMENT) >= uint(WORD_BIT_INDEX_MAXIMUM) {
		return float_quotient_general(destination, dividend, divisor, workspace, precision)
	}
	if uint(divisor_exponent-WORD_COUNT_INCREMENT) >= uint(WORD_BIT_INDEX_MAXIMUM) {
		return float_quotient_general(destination, dividend, divisor, workspace, precision)
	}
	dividend_minimum := Word(bits.CARRY_MAXIMUM) <<
		uint(dividend_exponent-WORD_COUNT_INCREMENT)
	divisor_minimum := Word(bits.CARRY_MAXIMUM) <<
		uint(divisor_exponent-WORD_COUNT_INCREMENT)
	dividend_word := dividend.Mantissa.Words[WORD_COUNT_MINIMUM]
	divisor_word := divisor.Mantissa.Words[WORD_COUNT_MINIMUM]
	if dividend_word&dividend_minimum == 0 {
		return float_quotient_general(destination, dividend, divisor, workspace, precision)
	}
	if dividend_word >= dividend_minimum<<WORD_COUNT_INCREMENT {
		return float_quotient_general(destination, dividend, divisor, workspace, precision)
	}
	if divisor_word&divisor_minimum == 0 {
		return float_quotient_general(destination, dividend, divisor, workspace, precision)
	}
	if divisor_word >= divisor_minimum<<WORD_COUNT_INCREMENT {
		return float_quotient_general(destination, dividend, divisor, workspace, precision)
	}
	quotient := dividend_word / divisor_word
	if dividend_word%divisor_word != 0 {
		return float_quotient_general(destination, dividend, divisor, workspace, precision)
	}
	bit_count := Float_Precision(bits.Bit_Size_64(bits.Word_64(quotient)))
	if bit_count > precision {
		return float_quotient_general(destination, dividend, divisor, workspace, precision)
	}
	mantissa := &destination.Mantissa
	previous_count := mantissa.Count
	destination.Precision, destination.Accuracy = precision, ACCURACY_EXACT
	destination.Form = FLOAT_FORM_FINITE
	destination.Negative = Polarity(uint8(dividend.Negative) ^ uint8(divisor.Negative))
	mantissa.Words[WORD_COUNT_MINIMUM] = quotient
	mantissa.Count, mantissa.Negative = Word_Count(WORD_COUNT_INCREMENT), POLARITY_NONNEGATIVE
	destination.Exponent = Float_Exponent(bit_count)
	int_clear((*Int)(mantissa), mantissa.Count, previous_count)
	return STATUS_OK
}

func float_quotient_general(
	destination, dividend, divisor Float_Handle,
	workspace Float_Division_Workspace_Handle, precision Float_Precision,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "float_quotient_general.status") }()
	Float_Handle_Invariants(destination, "float_quotient_general.destination_initial")
	Float_Handle_Invariants(dividend, "float_quotient_general.dividend")
	Float_Handle_Invariants(divisor, "float_quotient_general.divisor")
	Float_Division_Workspace_Handle_Invariants(
		workspace, "float_quotient_general.workspace")
	Float_Precision_Invariants(precision, "float_quotient_general.precision")
	mode := destination.Mode
	negative := POLARITY_NONNEGATIVE
	if dividend.Negative != divisor.Negative {
		negative = POLARITY_NEGATIVE
	}
	if divisor.Form == FLOAT_FORM_ZERO {
		if dividend.Form == FLOAT_FORM_ZERO {
			return STATUS_INPUT_INVALID
		}
		float_set_operation_infinity(destination, negative, precision, mode)
		return STATUS_OK
	}
	if dividend.Form == FLOAT_FORM_INFINITY {
		if divisor.Form == FLOAT_FORM_INFINITY {
			return STATUS_INPUT_INVALID
		}
		float_set_operation_infinity(destination, negative, precision, mode)
		return STATUS_OK
	}
	if divisor.Form == FLOAT_FORM_INFINITY {
		zero := *divisor
		zero.Form = FLOAT_FORM_ZERO
		zero.Negative = negative
		float_set_operation_operand(destination, &zero, precision, mode)
		return STATUS_OK
	}
	if dividend.Form == FLOAT_FORM_ZERO {
		zero := *dividend
		zero.Negative = negative
		float_set_operation_operand(destination, &zero, precision, mode)
		return STATUS_OK
	}
	float_divide_finite(
		destination, (*Float_Finite)((*Float)(dividend)),
		(*Float_Finite)((*Float)(divisor)), workspace,
		Float_Active_Precision(precision), mode, negative,
	)
	return STATUS_OK
}

// Float_Int_Into truncates one finite value toward zero into bounded caller storage.
func Float_Int_Into(
	destination Int_Handle, value Float_Handle,
) (accuracy Accuracy, status Conversion_Status) {
	defer func() {
		Accuracy_Invariants(accuracy, "float_int_into.accuracy")
		Conversion_Status_Invariants(status, "float_int_into.status")
	}()
	Int_Handle_Invariants(destination, "float_int_into.destination_initial")
	Float_Handle_Invariants(value, "float_int_into.value")
	if value.Form == FLOAT_FORM_INFINITY {
		return ACCURACY_EXACT, STATUS_VALUE_OVERFLOW
	}
	if value.Form == FLOAT_FORM_ZERO {
		int_zero(destination, destination.Count)
		return ACCURACY_EXACT, STATUS_OK
	}
	if value.Exponent <= FLOAT_EXPONENT_ZERO {
		int_zero(destination, destination.Count)
		return Accuracy(float_accuracy(false, value.Negative)), STATUS_OK
	}
	if value.Exponent > BIT_COUNT_MAXIMUM {
		return ACCURACY_EXACT, STATUS_VALUE_OVERFLOW
	}
	bit_count := int(Int_Bit_Count((*Int)(&value.Mantissa)))
	shift := int(value.Exponent) - bit_count
	if shift == BIT_COUNT_MINIMUM {
		previous_count := destination.Count
		for index := Word_Count(WORD_COUNT_MINIMUM); index < value.Mantissa.Count; index++ {
			destination.Words[index] = value.Mantissa.Words[index]
		}
		destination.Count, destination.Negative = value.Mantissa.Count, value.Negative
		if previous_count > destination.Count {
			int_clear(destination, destination.Count, previous_count)
		}
		return ACCURACY_EXACT, STATUS_OK
	}
	// Truncation must preserve source mantissa and transactional destination writes.
	var result_words [WORD_COUNT_MAXIMUM]Word
	result := Int{Words: result_words[:]}
	Int_Set(&result, (*Int)(&value.Mantissa))
	if shift < 0 {
		discard_count := -shift
		inexact := float_low_bits_nonzero(
			(*Positive_Int)(&value.Mantissa),
			Float_Discarded_Bit_Count(discard_count),
		)
		validated, validation := Shift_Count_Validate(
			Shift_Count_Unvalidated(discard_count),
		)
		if validation != STATUS_OK {
			return ACCURACY_EXACT, STATUS_VALUE_OVERFLOW
		}
		Int_Shift_Right(&result, &result, validated)
		if inexact {
			accuracy = Accuracy(float_accuracy(false, value.Negative))
		}
	} else if shift > 0 {
		validated, validation := Shift_Count_Validate(Shift_Count_Unvalidated(shift))
		if validation != STATUS_OK {
			return ACCURACY_EXACT, STATUS_VALUE_OVERFLOW
		}
		arithmetic := Int_Shift_Left(&result, &result, validated)
		if arithmetic != STATUS_OK {
			return ACCURACY_EXACT, STATUS_VALUE_OVERFLOW
		}
	}
	if result.Count != WORD_COUNT_MINIMUM {
		result.Negative = value.Negative
	}
	*destination = result
	return accuracy, STATUS_OK
}

// Float_Int_64 truncates through bounded Int conversion and reports narrowing failure.
func Float_Int_64(
	value Float_Handle,
) (result Int_64, accuracy Accuracy, status Conversion_Status) {
	defer func() {
		Int_64_Invariants(result, "float_int_64.result")
		Accuracy_Invariants(accuracy, "float_int_64.accuracy")
		Conversion_Status_Invariants(status, "float_int_64.status")
	}()
	Float_Handle_Invariants(value, "float_int_64.value")
	if value.Form == FLOAT_FORM_FINITE {
		if value.Mantissa.Count == Word_Count(WORD_COUNT_INCREMENT) {
			magnitude := uint64(value.Mantissa.Words[WORD_COUNT_MINIMUM])
			bit_count := Float_Exponent(bits.Bit_Size_64(bits.Word_64(magnitude)))
			if value.Exponent == bit_count {
				if value.Negative == POLARITY_NEGATIVE {
					if magnitude > INT_64_NEGATIVE_MAGNITUDE_MAXIMUM {
						return 0, ACCURACY_EXACT, STATUS_VALUE_OVERFLOW
					}
					if magnitude == INT_64_NEGATIVE_MAGNITUDE_MAXIMUM {
						result = Int_64(bits.INTEGER_64_MINIMUM)
						return result, ACCURACY_EXACT, STATUS_OK
					}
					return Int_64(-int64(magnitude)), ACCURACY_EXACT, STATUS_OK
				}
				if magnitude > uint64(bits.INTEGER_64_MAXIMUM) {
					return 0, ACCURACY_EXACT, STATUS_VALUE_OVERFLOW
				}
				return Int_64(magnitude), ACCURACY_EXACT, STATUS_OK
			}
		}
	}
	var integer Int
	accuracy, status = Float_Int_Into(&integer, value)
	if status != STATUS_OK {
		return 0, accuracy, status
	}
	result, status = Int_Int_64(&integer)
	return result, accuracy, status
}

// Float_Uint_64 truncates through bounded Int conversion without hiding sign loss.
func Float_Uint_64(
	value Float_Handle,
) (result Word_64, accuracy Accuracy, status Conversion_Status) {
	defer func() {
		Word_64_Invariants(result, "float_uint_64.result")
		Accuracy_Invariants(accuracy, "float_uint_64.accuracy")
		Conversion_Status_Invariants(status, "float_uint_64.status")
	}()
	Float_Handle_Invariants(value, "float_uint_64.value")
	if value.Form == FLOAT_FORM_FINITE {
		if value.Mantissa.Count == Word_Count(WORD_COUNT_INCREMENT) {
			magnitude := value.Mantissa.Words[WORD_COUNT_MINIMUM]
			bit_count := Float_Exponent(bits.Bit_Size_64(bits.Word_64(magnitude)))
			if value.Exponent == bit_count {
				if value.Negative == POLARITY_NEGATIVE {
					return 0, ACCURACY_EXACT, STATUS_VALUE_OVERFLOW
				}
				return Word_64(magnitude), ACCURACY_EXACT, STATUS_OK
			}
		}
	}
	var integer Int
	accuracy, status = Float_Int_Into(&integer, value)
	if status != STATUS_OK {
		return 0, accuracy, status
	}
	result, status = Int_Uint_64(&integer)
	return result, accuracy, status
}

// Float_Set_Rat rounds one exact rational through caller-owned quotient storage.
func Float_Set_Rat(
	destination Float_Handle, source Rat_Handle, workspace Float_Rat_Workspace_Handle,
) {
	Float_Handle_Invariants(destination, "float_set_rat.destination_initial")
	Rat_Handle_Invariants(source, "float_set_rat.source")
	Float_Rat_Workspace_Handle_Invariants(workspace, "float_set_rat.workspace")
	precision := destination.Precision
	mode := destination.Mode
	numerator := (*Float)(&workspace.Values.Numerator)
	denominator := (*Float)(&workspace.Values.Denominator)
	result := (*Float)(&workspace.Values.Result)
	// Exact factors must not inherit precision or rounding from reused scratch.
	*numerator = Float{Mantissa: Float_Mantissa{Words: numerator.Mantissa.Words}}
	*denominator = Float{Mantissa: Float_Mantissa{Words: denominator.Mantissa.Words}}
	Float_Set_Int(numerator, (*Int)(&source.Integers.Numerator))
	stored_denominator := (*Int)(&source.Integers.Denominator)
	if stored_denominator.Count == WORD_COUNT_MINIMUM {
		Float_Set_Uint_64(denominator, Word_64(WORD_COUNT_INCREMENT))
	} else {
		Float_Set_Int(denominator, stored_denominator)
	}
	result.Mode = mode
	float_set_nonfinite(
		result, Float_Nonfinite_Form(FLOAT_FORM_ZERO), POLARITY_NONNEGATIVE,
		precision,
	)
	Float_Quotient(
		result, numerator, denominator,
		(*Float_Division_Workspace)(&workspace.Division),
	)
	float_copy_exact(destination, result)
}

// Float_Rat_Into writes one exact binary rational or leaves the destination unchanged.
func Float_Rat_Into(
	destination Rat_Handle, value Float_Handle, workspace Float_Rat_Workspace_Handle,
) (status Float_Rat_Status) {
	defer func() { Float_Rat_Status_Invariants(status, "float_rat_into.status") }()
	Rat_Handle_Invariants(destination, "float_rat_into.destination_initial")
	Float_Handle_Invariants(value, "float_rat_into.value")
	Float_Rat_Workspace_Handle_Invariants(workspace, "float_rat_into.workspace")
	if value.Form == FLOAT_FORM_INFINITY {
		return STATUS_INPUT_INVALID
	}
	if value.Form == FLOAT_FORM_ZERO {
		float_rat_set_zero(destination)
		return STATUS_OK
	}
	bit_count := BIT_COUNT_MINIMUM
	if value.Mantissa.Count == Word_Count(BASE_BINARY) {
		bit_index := int(value.Exponent) - WORD_BIT_COUNT - WORD_COUNT_INCREMENT
		if uint(bit_index) < uint(WORD_BIT_COUNT) {
			minimum := Word(bits.CARRY_MAXIMUM) << uint(bit_index)
			high := value.Mantissa.Words[WORD_COUNT_INCREMENT]
			if high&minimum != 0 {
				if bit_index == WORD_BIT_INDEX_MAXIMUM {
					bit_count = int(value.Exponent)
				} else if high < minimum<<WORD_COUNT_INCREMENT {
					bit_count = int(value.Exponent)
				}
			}
		}
	}
	if bit_count == BIT_COUNT_MINIMUM {
		bit_count = int(Int_Bit_Count((*Int)(&value.Mantissa)))
	}
	if int(value.Exponent) == bit_count {
		if value.Mantissa.Count > RAT_WORD_COUNT_MAXIMUM {
			return STATUS_VALUE_OVERFLOW
		}
		numerator := (*Int)(&destination.Integers.Numerator)
		previous_count := numerator.Count
		for index := Word_Count(WORD_COUNT_MINIMUM); index < value.Mantissa.Count; index++ {
			numerator.Words[index] = value.Mantissa.Words[index]
		}
		numerator.Count, numerator.Negative = value.Mantissa.Count, value.Negative
		if previous_count > numerator.Count {
			int_clear(numerator, numerator.Count, previous_count)
		}
		denominator := (*Int)(&destination.Integers.Denominator)
		int_zero(denominator, denominator.Count)
		return STATUS_OK
	}
	general_status := float_rat_general(
		destination, (*Positive_Int)(&value.Mantissa), value.Exponent,
		value.Negative, (*Float_Rat_Nonempty_Integers)(&workspace.Integers),
	)
	return Float_Rat_Status(general_status)
}

func float_rat_general(
	destination Rat_Handle, mantissa Positive_Int_Handle, exponent Float_Exponent,
	negative Polarity, integers Float_Rat_Integers_Handle,
) (status Arithmetic_Status) {
	defer func() { Arithmetic_Status_Invariants(status, "float_rat_general.status") }()
	Rat_Handle_Invariants(destination, "float_rat_general.destination_initial")
	Positive_Int_Handle_Invariants(mantissa, "float_rat_general.mantissa")
	Float_Exponent_Invariants(exponent, "float_rat_general.exponent")
	Polarity_Invariants(negative, "float_rat_general.negative")
	Float_Rat_Integers_Handle_Invariants(integers, "float_rat_general.integers")
	numerator := &integers.Numerator
	denominator := (*Int)(&integers.Denominator)
	Int_Set(numerator, (*Int)((*Positive_Int)(mantissa)))
	bit_count := int(Int_Bit_Count(numerator))
	zero_count := int(Int_Trailing_Zero_Bit_Count(numerator))
	reduced_bit_count := bit_count - zero_count
	power := int(exponent) - bit_count + zero_count
	numerator_bit_count := reduced_bit_count
	if power > 0 {
		numerator_bit_count += power
	}
	denominator_bit_count := WORD_COUNT_INCREMENT
	if power < 0 {
		denominator_bit_count -= power
	}
	if numerator_bit_count > RAT_COMPONENT_BIT_COUNT_MAXIMUM {
		return STATUS_VALUE_OVERFLOW
	}
	if denominator_bit_count > RAT_COMPONENT_BIT_COUNT_MAXIMUM {
		return STATUS_VALUE_OVERFLOW
	}
	// Integer results must not retain a denominator from prior scratch use.
	if power >= 0 {
		int_zero(denominator, denominator.Count)
	}
	if zero_count > 0 {
		shift, validation := Shift_Count_Validate(Shift_Count_Unvalidated(zero_count))
		if validation != STATUS_OK {
			return STATUS_VALUE_OVERFLOW
		}
		Int_Shift_Right(numerator, numerator, shift)
	}
	if power > 0 {
		shift, validation := Shift_Count_Validate(Shift_Count_Unvalidated(power))
		if validation != STATUS_OK {
			return STATUS_VALUE_OVERFLOW
		}
		if Int_Shift_Left(numerator, numerator, shift) != STATUS_OK {
			return STATUS_VALUE_OVERFLOW
		}
	} else if power < 0 {
		Int_Set_Uint_64(denominator, Word_64(WORD_COUNT_INCREMENT))
		shift, validation := Shift_Count_Validate(Shift_Count_Unvalidated(-power))
		if validation != STATUS_OK {
			return STATUS_VALUE_OVERFLOW
		}
		if Int_Shift_Left(denominator, denominator, shift) != STATUS_OK {
			return STATUS_VALUE_OVERFLOW
		}
	}
	numerator.Negative = negative
	// Reusing scratch must not alter previously returned rationals.
	Int_Set((*Int)(&destination.Integers.Numerator), numerator)
	Int_Set((*Int)(&destination.Integers.Denominator), denominator)
	return STATUS_OK
}

// Float_Square_Root rounds one nonnegative root through caller-owned restoring state.
func Float_Square_Root(
	destination Float_Handle, source Float_Handle,
	workspace Float_Square_Root_Workspace_Handle,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "float_square_root.status") }()
	Float_Handle_Invariants(destination, "float_square_root.destination_initial")
	Float_Handle_Invariants(source, "float_square_root.source")
	Float_Square_Root_Workspace_Handle_Invariants(workspace, "float_square_root.workspace")
	if source.Negative == POLARITY_NEGATIVE {
		if source.Form != FLOAT_FORM_ZERO {
			return STATUS_INPUT_INVALID
		}
	}
	precision := destination.Precision
	if precision == FLOAT_PRECISION_MINIMUM {
		precision = source.Precision
	}
	if source.Form != FLOAT_FORM_FINITE {
		float_square_root_general(destination, source, workspace, precision)
		return STATUS_OK
	}
	if source.Mantissa.Count != Word_Count(WORD_COUNT_INCREMENT) {
		float_square_root_general(destination, source, workspace, precision)
		return STATUS_OK
	}
	exponent := int(source.Exponent)
	if uint(exponent-WORD_COUNT_INCREMENT) >= uint(WORD_BIT_INDEX_MAXIMUM) {
		float_square_root_general(destination, source, workspace, precision)
		return STATUS_OK
	}
	minimum := Word(bits.CARRY_MAXIMUM) << uint(exponent-WORD_COUNT_INCREMENT)
	word := source.Mantissa.Words[WORD_COUNT_MINIMUM]
	if word&minimum == 0 {
		float_square_root_general(destination, source, workspace, precision)
		return STATUS_OK
	}
	if word >= minimum<<WORD_COUNT_INCREMENT {
		float_square_root_general(destination, source, workspace, precision)
		return STATUS_OK
	}
	root_bit_count := (exponent + SQUARE_ROOT_DEGREE - WORD_COUNT_INCREMENT) /
		SQUARE_ROOT_DEGREE
	root := Word(bits.CARRY_MAXIMUM) << uint(root_bit_count)
	next := (root + word/root) / SQUARE_ROOT_DEGREE
	for next < root {
		root = next
		next = (root + word/root) / SQUARE_ROOT_DEGREE
	}
	if root*root != word {
		float_square_root_general(destination, source, workspace, precision)
		return STATUS_OK
	}
	mantissa := &destination.Mantissa
	previous_count := mantissa.Count
	destination.Precision, destination.Accuracy = precision, ACCURACY_EXACT
	destination.Form, destination.Negative = FLOAT_FORM_FINITE, POLARITY_NONNEGATIVE
	mantissa.Words[WORD_COUNT_MINIMUM] = root
	mantissa.Count, mantissa.Negative = Word_Count(WORD_COUNT_INCREMENT), POLARITY_NONNEGATIVE
	destination.Exponent = Float_Exponent(bits.Bit_Size_64(bits.Word_64(root)))
	int_clear((*Int)(mantissa), mantissa.Count, previous_count)
	return STATUS_OK
}

func float_square_root_general(
	destination Float_Handle, source Float_Handle,
	workspace Float_Square_Root_Workspace_Handle,
	precision Float_Precision,
) {
	Float_Handle_Invariants(destination, "float_square_root_general.destination_initial")
	Float_Handle_Invariants(source, "float_square_root_general.source")
	Float_Square_Root_Workspace_Handle_Invariants(
		workspace, "float_square_root_general.workspace",
	)
	Float_Precision_Invariants(precision, "float_square_root_general.precision")
	mode := destination.Mode
	if source.Form == FLOAT_FORM_ZERO {
		float_set_operation_operand(destination, source, precision, mode)
		return
	}
	if source.Form == FLOAT_FORM_INFINITY {
		float_set_operation_infinity(
			destination, POLARITY_NONNEGATIVE, precision, mode,
		)
		return
	}
	float_square_root_finite(
		destination, (*Float_Nonnegative_Finite)((*Float)(source)), workspace,
		Float_Active_Precision(precision), mode,
	)
}

func float_square_root_finite(
	destination Float_Handle,
	source Float_Nonnegative_Finite_Handle,
	workspace Float_Square_Root_Workspace_Handle,
	precision Float_Active_Precision,
	mode Rounding_Mode,
) {
	Float_Handle_Invariants(destination, "float_square_root_finite.destination_initial")
	Float_Nonnegative_Finite_Handle_Invariants(source, "float_square_root_finite.source")
	Float_Square_Root_Workspace_Handle_Invariants(
		workspace, "float_square_root_finite.workspace",
	)
	Float_Active_Precision_Invariants(precision, "float_square_root_finite.precision")
	Rounding_Mode_Invariants(mode, "float_square_root_finite.mode")
	rounding_bit, sticky_bit := float_square_root_extract(source, workspace, precision)
	exponent := int(source.Exponent) / BASE_BINARY
	if source.Exponent > FLOAT_EXPONENT_ZERO {
		if int(source.Exponent)%BASE_BINARY != BIT_COUNT_MINIMUM {
			exponent++
		}
	}
	mantissa_count := (int(precision) + WORD_BIT_INDEX_MAXIMUM) / WORD_BIT_COUNT
	var mantissa_words [WORD_COUNT_MAXIMUM]Word
	var mantissa Float_Mantissa
	mantissa.Words = mantissa_words[:]
	mantissa.Count = Word_Count(mantissa_count)
	for index := WORD_COUNT_MINIMUM; index < mantissa_count; index++ {
		mantissa.Words[index] = workspace.Root[index]
	}
	increment := float_rounding_increment(
		mode, POLARITY_NONNEGATIVE, rounding_bit, sticky_bit,
		Bit_Value(mantissa.Words[WORD_COUNT_MINIMUM]&Word(BIT_SET)),
	)
	accuracy := ACCURACY_EXACT
	if rounding_bit != BIT_CLEAR {
		accuracy = Accuracy(float_accuracy(increment, POLARITY_NONNEGATIVE))
	} else if sticky_bit != BIT_CLEAR {
		accuracy = Accuracy(float_accuracy(increment, POLARITY_NONNEGATIVE))
	}
	float_square_root_commit(
		destination, (*Float_Active_Mantissa)(&mantissa),
		Float_Square_Root_Exponent(exponent), precision, mode, accuracy, increment,
	)
}

func float_square_root_extract(
	source Float_Nonnegative_Finite_Handle,
	workspace Float_Square_Root_Workspace_Handle,
	precision Float_Active_Precision,
) (rounding_bit Bit_Value, sticky Bit_Value) {
	defer func() {
		Bit_Value_Invariants(
			rounding_bit, "float_square_root_extract.rounding_bit",
		)
		Bit_Value_Invariants(sticky, "float_square_root_extract.sticky")
	}()
	Float_Nonnegative_Finite_Handle_Invariants(source, "float_square_root_extract.source")
	Float_Square_Root_Workspace_Handle_Invariants(
		workspace, "float_square_root_extract.workspace",
	)
	Float_Active_Precision_Invariants(precision, "float_square_root_extract.precision")
	active_count := Float_Square_Root_Active_Word_Count(
		(int(precision) + FLOAT_SQUARE_ROOT_GUARD_BIT_COUNT +
			WORD_BIT_INDEX_MAXIMUM) / WORD_BIT_COUNT,
	)
	source_index := float_square_root_digits(source, workspace, precision, active_count)
	rounding_bit = Bit_Value(workspace.Root[WORD_COUNT_MINIMUM] & Word(BIT_SET))
	sticky = float_square_root_sticky(source, workspace, source_index, active_count)
	float_square_root_remove_guard(workspace, precision)
	return rounding_bit, sticky
}

func float_square_root_digits(
	source Float_Nonnegative_Finite_Handle,
	workspace Float_Square_Root_Workspace_Handle,
	precision Float_Active_Precision,
	workspace_count Float_Square_Root_Active_Word_Count,
) (source_index Float_Square_Root_Source_Bit_Index) {
	defer func() {
		Float_Square_Root_Source_Bit_Index_Invariants(
			source_index, "float_square_root_digits.source_index",
		)
	}()
	Float_Nonnegative_Finite_Handle_Invariants(source, "float_square_root_digits.source")
	Float_Square_Root_Workspace_Handle_Invariants(
		workspace, "float_square_root_digits.workspace",
	)
	Float_Active_Precision_Invariants(precision, "float_square_root_digits.precision")
	Float_Square_Root_Active_Word_Count_Invariants(
		workspace_count, "float_square_root_digits.workspace_count",
	)
	float_square_root_clear(workspace, workspace_count)
	if float_square_root_triple_word_match(source, workspace, precision) {
		return BIT_INDEX_UNVALIDATED_MINIMUM
	}
	cursor := int(Int_Bit_Count((*Int)(&source.Mantissa))) - WORD_COUNT_INCREMENT
	prefix_clear := int(source.Exponent)%BASE_BINARY != BIT_COUNT_MINIMUM
	digit_count := int(precision) + FLOAT_SQUARE_ROOT_GUARD_BIT_COUNT
	for digit_index := WORD_COUNT_MINIMUM; digit_index < digit_count; digit_index++ {
		pair := Float_Square_Root_Pair(BIT_CLEAR)
		pair_limit := FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT
		source_exhausted := false
		for pair_index := WORD_COUNT_MINIMUM; pair_index < pair_limit; pair_index++ {
			pair <<= WORD_COUNT_INCREMENT
			if prefix_clear {
				prefix_clear = false
				continue
			}
			if cursor >= BIT_COUNT_MINIMUM {
				word_index := cursor / WORD_BIT_COUNT
				bit_index := uint(cursor % WORD_BIT_COUNT)
				bit := source.Mantissa.Words[word_index] >> bit_index
				pair |= Float_Square_Root_Pair(bit & Word(BIT_SET))
				cursor--
				if cursor == BIT_INDEX_UNVALIDATED_MINIMUM {
					source_exhausted = true
				}
			}
		}
		active_count := Float_Square_Root_Active_Word_Count(
			(digit_index + FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT +
				WORD_COUNT_INCREMENT + WORD_BIT_INDEX_MAXIMUM) / WORD_BIT_COUNT,
		)
		float_square_root_digit(workspace, active_count, pair)
		if source_exhausted {
			last_digit_index := digit_count - WORD_COUNT_INCREMENT
			suffix_count := last_digit_index - digit_index
			if suffix_count > BIT_COUNT_MINIMUM {
				if float_square_root_complete(
					workspace, Float_Active_Precision(suffix_count),
					workspace_count,
				) {
					break
				}
			}
		}
	}
	return Float_Square_Root_Source_Bit_Index(cursor)
}

func float_square_root_digit(
	workspace Float_Square_Root_Workspace_Handle,
	active_count Float_Square_Root_Active_Word_Count,
	pair Float_Square_Root_Pair,
) {
	Float_Square_Root_Workspace_Handle_Invariants(
		workspace, "float_square_root_digit.workspace",
	)
	Float_Square_Root_Active_Word_Count_Invariants(
		active_count, "float_square_root_digit.active_count",
	)
	Float_Square_Root_Pair_Invariants(pair, "float_square_root_digit.pair")
	workspace_reference := Float_Square_Root_Workspace_Reference(workspace)
	pair_reference := Float_Square_Root_Pair_Reference(&pair)
	if active_count == WORD_COUNT_INCREMENT {
		float_square_root_digit_one(workspace_reference, pair_reference)
		return
	}
	if active_count == BASE_BINARY {
		float_square_root_digit_two(workspace_reference, pair_reference)
		return
	}
	if active_count == BASE_BINARY+WORD_COUNT_INCREMENT {
		float_square_root_digit_three(workspace_reference, pair_reference)
		return
	}
	float_square_root_digit_many(
		workspace_reference, Float_Square_Root_Multiword_Count(active_count),
		pair_reference,
	)
}

func float_square_root_complete(
	workspace Float_Square_Root_Workspace_Handle,
	count Float_Active_Precision,
	active_count Float_Square_Root_Active_Word_Count,
) (complete Boolean) {
	defer func() {
		Boolean_Invariants(complete, "float_square_root_complete.complete")
	}()
	Float_Square_Root_Workspace_Handle_Invariants(
		workspace, "float_square_root_complete.workspace",
	)
	Float_Active_Precision_Invariants(count, "float_square_root_complete.count")
	Float_Square_Root_Active_Word_Count_Invariants(
		active_count, "float_square_root_complete.active_count",
	)
	for index := WORD_COUNT_MINIMUM; index < int(active_count); index++ {
		if workspace.Remainder[index] != 0 {
			return false
		}
	}
	word_shift := int(count) / WORD_BIT_COUNT
	bit_shift := uint(int(count) % WORD_BIT_COUNT)
	minimum := WORD_COUNT_MINIMUM
	last_index := int(active_count) - WORD_COUNT_INCREMENT
	for destination_index := last_index; destination_index >= minimum; destination_index-- {
		source_index := destination_index - word_shift
		word := Word(BIT_CLEAR)
		if source_index >= minimum {
			word = workspace.Root[source_index] << bit_shift
			if bit_shift != BIT_COUNT_MINIMUM {
				if source_index > WORD_COUNT_MINIMUM {
					word |= workspace.Root[source_index-WORD_COUNT_INCREMENT] >>
						(WORD_BIT_COUNT - bit_shift)
				}
			}
		}
		workspace.Root[destination_index] = word
	}
	return true
}

func float_square_root_clear(
	workspace Float_Square_Root_Workspace_Handle,
	active_count Float_Square_Root_Active_Word_Count,
) {
	Float_Square_Root_Workspace_Handle_Invariants(
		workspace, "float_square_root_clear.workspace",
	)
	Float_Square_Root_Active_Word_Count_Invariants(
		active_count, "float_square_root_clear.active_count",
	)
	for index := WORD_COUNT_MINIMUM; index < int(active_count); index++ {
		workspace.Root[index] = 0
		workspace.Remainder[index] = 0
		workspace.Candidate[index] = 0
	}
}

func float_square_root_sticky(
	source Float_Nonnegative_Finite_Handle,
	workspace Float_Square_Root_Workspace_Handle,
	source_index Float_Square_Root_Source_Bit_Index,
	active_count Float_Square_Root_Active_Word_Count,
) (sticky Bit_Value) {
	defer func() { Bit_Value_Invariants(sticky, "float_square_root_sticky.sticky") }()
	Float_Nonnegative_Finite_Handle_Invariants(source, "float_square_root_sticky.source")
	Float_Square_Root_Workspace_Handle_Invariants(
		workspace, "float_square_root_sticky.workspace",
	)
	Float_Square_Root_Source_Bit_Index_Invariants(
		source_index, "float_square_root_sticky.source_index",
	)
	Float_Square_Root_Active_Word_Count_Invariants(
		active_count, "float_square_root_sticky.active_count",
	)
	if source_index >= BIT_COUNT_MINIMUM {
		if float_low_bits_nonzero(
			(*Positive_Int)(&source.Mantissa),
			Float_Discarded_Bit_Count(
				int(source_index)+WORD_COUNT_INCREMENT,
			),
		) {
			return BIT_SET
		}
	}
	for index := WORD_COUNT_MINIMUM; index < int(active_count); index++ {
		if workspace.Remainder[index] != 0 {
			return BIT_SET
		}
	}
	return BIT_CLEAR
}

func float_square_root_remove_guard(
	workspace Float_Square_Root_Workspace_Handle, precision Float_Active_Precision,
) {
	Float_Square_Root_Workspace_Handle_Invariants(
		workspace, "float_square_root_remove_guard.workspace",
	)
	Float_Active_Precision_Invariants(precision, "float_square_root_remove_guard.precision")
	count := (int(precision) + FLOAT_SQUARE_ROOT_GUARD_BIT_COUNT +
		WORD_BIT_INDEX_MAXIMUM) / WORD_BIT_COUNT
	carry := Word(BIT_CLEAR)
	for index := count - WORD_COUNT_INCREMENT; index >= WORD_COUNT_MINIMUM; index-- {
		next := workspace.Root[index] << WORD_BIT_INDEX_MAXIMUM
		workspace.Root[index] = workspace.Root[index]>>WORD_COUNT_INCREMENT | carry
		carry = next
	}
}

func float_square_root_commit(
	destination Float_Handle,
	mantissa Float_Active_Mantissa_Handle,
	exponent Float_Square_Root_Exponent,
	precision Float_Active_Precision,
	mode Rounding_Mode,
	accuracy Accuracy,
	increment Boolean,
) {
	Float_Handle_Invariants(destination, "float_square_root_commit.destination_initial")
	Float_Active_Mantissa_Handle_Invariants(mantissa, "float_square_root_commit.mantissa")
	Float_Square_Root_Exponent_Invariants(exponent, "float_square_root_commit.exponent")
	Float_Active_Precision_Invariants(precision, "float_square_root_commit.precision")
	Rounding_Mode_Invariants(mode, "float_square_root_commit.mode")
	Accuracy_Invariants(accuracy, "float_square_root_commit.accuracy")
	Boolean_Invariants(increment, "float_square_root_commit.increment")
	result_exponent := Float_Exponent(exponent)
	if increment {
		float_increment_mantissa(
			(*Positive_Int)((*Float_Active_Mantissa)(mantissa)),
			precision, &result_exponent,
		)
	}
	*destination = Float{
		Precision: Float_Precision(precision), Mode: mode, Accuracy: accuracy,
		Form: FLOAT_FORM_FINITE, Mantissa: Float_Mantissa(*mantissa),
		Exponent: result_exponent,
	}
}

func float_divide_finite(
	destination Float_Handle,
	dividend Float_Finite_Handle,
	divisor Float_Finite_Handle,
	workspace Float_Division_Workspace_Handle,
	precision Float_Active_Precision,
	mode Rounding_Mode,
	negative Polarity,
) {
	Float_Handle_Invariants(destination, "float_divide_finite.destination_initial")
	Float_Finite_Handle_Invariants(dividend, "float_divide_finite.dividend")
	Float_Finite_Handle_Invariants(divisor, "float_divide_finite.divisor")
	Float_Division_Workspace_Handle_Invariants(workspace, "float_divide_finite.workspace")
	Float_Active_Precision_Invariants(precision, "float_divide_finite.precision")
	Rounding_Mode_Invariants(mode, "float_divide_finite.mode")
	Polarity_Invariants(negative, "float_divide_finite.negative")
	dividend_count := int(dividend.Mantissa.Count)
	dividend_high := dividend.Mantissa.Words[dividend_count-WORD_COUNT_INCREMENT]
	dividend_bits := (dividend_count-WORD_COUNT_INCREMENT)*WORD_BIT_COUNT +
		int(bits.Bit_Size_64(bits.Word_64(dividend_high)))
	divisor_count := int(divisor.Mantissa.Count)
	divisor_high := divisor.Mantissa.Words[divisor_count-WORD_COUNT_INCREMENT]
	divisor_bits := (divisor_count-WORD_COUNT_INCREMENT)*WORD_BIT_COUNT +
		int(bits.Bit_Size_64(bits.Word_64(divisor_high)))
	aligned_bits := dividend_bits
	if divisor_bits > aligned_bits {
		aligned_bits = divisor_bits
	}
	count := Float_Division_Active_Word_Count(
		(aligned_bits+WORD_BIT_COUNT-WORD_COUNT_INCREMENT)/WORD_BIT_COUNT +
			WORD_COUNT_INCREMENT,
	)
	for index := WORD_COUNT_MINIMUM; index < int(count); index++ {
		workspace.Remainder[index] = 0
		workspace.Divisor[index] = 0
	}
	float_division_load(
		workspace, dividend, divisor, Float_Division_Aligned_Bit_Count(aligned_bits),
	)
	order := float_division_compare(workspace, count)
	exponent := int(dividend.Exponent) - int(divisor.Exponent)
	if order != ORDER_BEFORE {
		exponent++
	}
	if exponent < FLOAT_EXPONENT_MINIMUM {
		float_set_division_bound(
			destination, Float_Nonfinite_Form(FLOAT_FORM_ZERO), negative, precision,
			mode, false,
		)
		return
	}
	if exponent > FLOAT_EXPONENT_MAXIMUM {
		float_set_division_bound(
			destination, Float_Nonfinite_Form(FLOAT_FORM_INFINITY), negative,
			precision, mode, true,
		)
		return
	}
	if order == ORDER_SAME {
		var mantissa Float_Mantissa
		var mantissa_words [WORD_COUNT_INCREMENT]Word
		mantissa.Words = mantissa_words[:]
		mantissa.Count = WORD_COUNT_INCREMENT
		mantissa.Words[WORD_COUNT_MINIMUM] = Word(bits.CARRY_MAXIMUM)
		result := Float{
			Precision: Float_Precision(precision), Mode: mode, Accuracy: ACCURACY_EXACT,
			Form: FLOAT_FORM_FINITE, Negative: negative, Mantissa: mantissa,
			Exponent: Float_Exponent(exponent),
		}
		Float_Copy(destination, &result)
		return
	}
	float_set_quotient_mantissa(
		destination, workspace, count, Float_Unequal_Order(order),
		Float_Exponent(exponent), negative,
		precision, mode,
	)
}

func float_division_load(
	workspace Float_Division_Workspace_Handle,
	dividend Float_Finite_Handle,
	divisor Float_Finite_Handle,
	aligned_bits Float_Division_Aligned_Bit_Count,
) {
	Float_Division_Workspace_Handle_Invariants(workspace, "float_division_load.workspace")
	Float_Finite_Handle_Invariants(dividend, "float_division_load.dividend")
	Float_Finite_Handle_Invariants(divisor, "float_division_load.divisor")
	Float_Division_Aligned_Bit_Count_Invariants(
		aligned_bits, "float_division_load.aligned_bits",
	)
	dividend_count := int(dividend.Mantissa.Count)
	dividend_high := dividend.Mantissa.Words[dividend_count-WORD_COUNT_INCREMENT]
	dividend_bits := (dividend_count-WORD_COUNT_INCREMENT)*WORD_BIT_COUNT +
		int(bits.Bit_Size_64(bits.Word_64(dividend_high)))
	divisor_count := int(divisor.Mantissa.Count)
	divisor_high := divisor.Mantissa.Words[divisor_count-WORD_COUNT_INCREMENT]
	divisor_bits := (divisor_count-WORD_COUNT_INCREMENT)*WORD_BIT_COUNT +
		int(bits.Bit_Size_64(bits.Word_64(divisor_high)))
	dividend_shift := int(aligned_bits) - dividend_bits
	divisor_shift := int(aligned_bits) - divisor_bits
	dividend_word_shift := dividend_shift / WORD_BIT_COUNT
	dividend_bit_shift := uint(dividend_shift % WORD_BIT_COUNT)
	for index := WORD_COUNT_MINIMUM; index < int(dividend.Mantissa.Count); index++ {
		destination_index := dividend_word_shift + index
		word := dividend.Mantissa.Words[index]
		workspace.Remainder[destination_index] |= word << dividend_bit_shift
		if dividend_bit_shift != 0 {
			workspace.Remainder[destination_index+WORD_COUNT_INCREMENT] |=
				word >> (WORD_BIT_COUNT - dividend_bit_shift)
		}
	}
	divisor_word_shift := divisor_shift / WORD_BIT_COUNT
	divisor_bit_shift := uint(divisor_shift % WORD_BIT_COUNT)
	for index := WORD_COUNT_MINIMUM; index < int(divisor.Mantissa.Count); index++ {
		destination_index := divisor_word_shift + index
		word := divisor.Mantissa.Words[index]
		workspace.Divisor[destination_index] |= word << divisor_bit_shift
		if divisor_bit_shift != 0 {
			workspace.Divisor[destination_index+WORD_COUNT_INCREMENT] |=
				word >> (WORD_BIT_COUNT - divisor_bit_shift)
		}
	}
}

func float_division_compare(
	workspace Float_Division_Workspace_Handle, count Float_Division_Active_Word_Count,
) (order Order) {
	defer func() { Order_Invariants(order, "float_division_compare.order") }()
	Float_Division_Workspace_Handle_Invariants(workspace, "float_division_compare.workspace")
	Float_Division_Active_Word_Count_Invariants(count, "float_division_compare.count")
	for index := int(count) - WORD_COUNT_INCREMENT; index >= WORD_COUNT_MINIMUM; index-- {
		if workspace.Remainder[index] < workspace.Divisor[index] {
			return ORDER_BEFORE
		}
		if workspace.Remainder[index] > workspace.Divisor[index] {
			return ORDER_AFTER
		}
	}
	return ORDER_SAME
}

func float_division_subtract(
	workspace Float_Division_Workspace_Handle, count Float_Division_Active_Word_Count,
) {
	Float_Division_Workspace_Handle_Invariants(workspace, "float_division_subtract.workspace")
	Float_Division_Active_Word_Count_Invariants(count, "float_division_subtract.count")
	borrow := bits.Borrow_In(bits.CARRY_MINIMUM)
	for index := WORD_COUNT_MINIMUM; index < int(count); index++ {
		difference, next := bits.Subtract_Word(
			bits.Word(workspace.Remainder[index]),
			bits.Subtrahend_Word(workspace.Divisor[index]), borrow,
		)
		workspace.Remainder[index] = Word(difference)
		borrow = bits.Borrow_In(next)
	}
	aver.Always(borrow == bits.Borrow_In(bits.CARRY_MINIMUM),
		"A quotient digit subtracts only after magnitude comparison.")
}

func float_division_digit(
	workspace Float_Division_Workspace_Handle, count Float_Division_Active_Word_Count,
) (digit Bit_Value) {
	defer func() { Bit_Value_Invariants(digit, "float_division_digit.digit") }()
	Float_Division_Workspace_Handle_Invariants(workspace, "float_division_digit.workspace")
	Float_Division_Active_Word_Count_Invariants(count, "float_division_digit.count")
	carry := Word(0)
	for index := WORD_COUNT_MINIMUM; index < int(count); index++ {
		next := workspace.Remainder[index] >> WORD_BIT_INDEX_MAXIMUM
		workspace.Remainder[index] = workspace.Remainder[index]<<1 | carry
		carry = next
	}
	if float_division_compare(workspace, count) == ORDER_BEFORE {
		return BIT_CLEAR
	}
	float_division_subtract(workspace, count)
	return BIT_SET
}

func float_set_division_bound(
	destination Float_Handle,
	form Float_Nonfinite_Form,
	negative Polarity,
	precision Float_Active_Precision,
	mode Rounding_Mode,
	increment Boolean,
) {
	Float_Handle_Invariants(destination, "float_set_division_bound.destination_initial")
	Float_Nonfinite_Form_Invariants(form, "float_set_division_bound.form")
	Polarity_Invariants(negative, "float_set_division_bound.negative")
	Float_Active_Precision_Invariants(precision, "float_set_division_bound.precision")
	Rounding_Mode_Invariants(mode, "float_set_division_bound.mode")
	Boolean_Invariants(increment, "float_set_division_bound.increment")
	result := Float{Precision: Float_Precision(precision), Mode: mode}
	result.Mantissa.Words = destination.Mantissa.Words
	float_set_nonfinite(
		&result, form, negative, Float_Precision(precision),
	)
	result.Accuracy = Accuracy(float_accuracy(increment, negative))
	*destination = result
}

func float_set_quotient_mantissa(
	destination Float_Handle, workspace Float_Division_Workspace_Handle,
	count Float_Division_Active_Word_Count,
	order Float_Unequal_Order,
	exponent Float_Exponent,
	negative Polarity,
	precision Float_Active_Precision,
	mode Rounding_Mode,
) {
	Float_Handle_Invariants(destination, "float_set_quotient_mantissa.destination_initial")
	Float_Division_Workspace_Handle_Invariants(
		workspace, "float_set_quotient_mantissa.workspace")
	Float_Division_Active_Word_Count_Invariants(count, "float_set_quotient_mantissa.count")
	Float_Unequal_Order_Invariants(order, "float_set_quotient_mantissa.order")
	Float_Exponent_Invariants(exponent, "float_set_quotient_mantissa.exponent")
	Polarity_Invariants(negative, "float_set_quotient_mantissa.negative")
	Float_Active_Precision_Invariants(precision, "float_set_quotient_mantissa.precision")
	Rounding_Mode_Invariants(mode, "float_set_quotient_mantissa.mode")
	if float_set_quotient_words(
		destination, workspace, count, order, exponent, negative, precision, mode,
	) {
		return
	}
	mantissa_count := (int(precision) + WORD_BIT_COUNT - WORD_COUNT_INCREMENT) /
		WORD_BIT_COUNT
	var mantissa Float_Mantissa
	var mantissa_words [WORD_COUNT_MAXIMUM]Word
	mantissa.Words = mantissa_words[:]
	mantissa.Count = Word_Count(mantissa_count)
	quotient_index := WORD_COUNT_MINIMUM
	if order != Float_Unequal_Order(ORDER_BEFORE) {
		bit_index := int(precision) - WORD_COUNT_INCREMENT
		mantissa.Words[bit_index/WORD_BIT_COUNT] |=
			Word(bits.CARRY_MAXIMUM) << uint(bit_index%WORD_BIT_COUNT)
		float_division_subtract(workspace, count)
		quotient_index++
	}
	for quotient_index < int(precision) {
		if float_division_digit(workspace, count) == BIT_SET {
			bit_index := int(precision) - quotient_index - WORD_COUNT_INCREMENT
			mantissa.Words[bit_index/WORD_BIT_COUNT] |=
				Word(bits.CARRY_MAXIMUM) << uint(bit_index%WORD_BIT_COUNT)
		}
		quotient_index++
	}
	rounding_bit := float_division_digit(workspace, count)
	sticky_bit := BIT_CLEAR
	if float_division_remainder_nonzero(workspace, count) {
		sticky_bit = BIT_SET
	}
	increment := float_rounding_increment(
		mode, negative, rounding_bit, sticky_bit,
		Bit_Value(mantissa.Words[WORD_COUNT_MINIMUM]&1),
	)
	accuracy := float_rounding_accuracy(rounding_bit, sticky_bit, increment, negative)
	if increment {
		float_increment_mantissa(
			(*Positive_Int)(&mantissa), precision, &exponent,
		)
	}
	if exponent > FLOAT_EXPONENT_MAXIMUM {
		float_set_division_bound(
			destination, Float_Nonfinite_Form(FLOAT_FORM_INFINITY), negative,
			precision, mode, true,
		)
		return
	}
	result := Float{
		Precision: Float_Precision(precision), Mode: mode, Accuracy: accuracy,
		Form: FLOAT_FORM_FINITE, Negative: negative, Mantissa: mantissa,
		Exponent: exponent,
	}
	Float_Copy(destination, &result)
}

func float_set_quotient_words(
	destination Float_Handle,
	workspace Float_Division_Workspace_Handle,
	count Float_Division_Active_Word_Count,
	order Float_Unequal_Order,
	exponent Float_Exponent,
	negative Polarity,
	precision Float_Active_Precision,
	mode Rounding_Mode,
) (matched Boolean) {
	defer func() { Boolean_Invariants(matched, "float_set_quotient_words.matched") }()
	Float_Handle_Invariants(destination, "float_set_quotient_words.destination_initial")
	Float_Division_Workspace_Handle_Invariants(
		workspace, "float_set_quotient_words.workspace")
	Float_Division_Active_Word_Count_Invariants(count, "float_set_quotient_words.count")
	Float_Unequal_Order_Invariants(order, "float_set_quotient_words.order")
	Float_Exponent_Invariants(exponent, "float_set_quotient_words.exponent")
	Polarity_Invariants(negative, "float_set_quotient_words.negative")
	Float_Active_Precision_Invariants(precision, "float_set_quotient_words.precision")
	Rounding_Mode_Invariants(mode, "float_set_quotient_words.mode")
	divisor, dividend, scale :=
		float_division_word_prepare(workspace, count, order, precision)
	if divisor == WORD_COUNT_MINIMUM {
		return false
	}
	float_division_word_scale(
		workspace, Float_Division_Short_Count(divisor), Float_Active_Word_Count(dividend),
		Float_Division_Active_Scale(scale),
	)
	result := (*Float_Division_Destination)((*Float)(destination))
	// Guard-width quotient must not invalidate destination before commit.
	var quotient_storage [WORD_COUNT_MAXIMUM]Word
	quotient := Float_Division_Quotient_Words(quotient_storage[:])
	float_division_word_quotient(workspace, quotient,
		Float_Division_Short_Count(divisor), Float_Active_Word_Count(dividend),
	)
	float_set_quotient_words_commit(
		result, workspace, quotient, Float_Division_Short_Count(divisor),
		Float_Active_Word_Count(dividend), Float_Division_Precision(precision),
		exponent, negative, mode,
	)
	return true
}

func float_division_word_prepare(
	workspace Float_Division_Workspace_Handle,
	count Float_Division_Active_Word_Count,
	order Float_Unequal_Order,
	precision Float_Active_Precision,
) (divisor Float_Division_Divisor_Count,
	dividend Float_Division_Dividend_Count, scale Float_Division_Scale,
) {
	defer func() {
		Float_Division_Divisor_Count_Invariants(
			divisor, "float_division_word_prepare.divisor",
		)
		Float_Division_Dividend_Count_Invariants(
			dividend, "float_division_word_prepare.dividend",
		)
		Float_Division_Scale_Invariants(scale, "float_division_word_prepare.scale")
	}()
	Float_Division_Workspace_Handle_Invariants(
		workspace, "float_division_word_prepare.workspace")
	Float_Division_Active_Word_Count_Invariants(
		count, "float_division_word_prepare.count",
	)
	Float_Unequal_Order_Invariants(order, "float_division_word_prepare.order")
	Float_Active_Precision_Invariants(precision, "float_division_word_prepare.precision")
	divisor_count := int(count) - WORD_COUNT_INCREMENT
	for divisor_count > WORD_COUNT_MINIMUM {
		if workspace.Divisor[divisor_count-WORD_COUNT_INCREMENT] != 0 {
			break
		}
		divisor_count--
	}
	if divisor_count > BASE_BINARY {
		return 0, 0, 0
	}
	aligned_bits := (divisor_count-WORD_COUNT_INCREMENT)*WORD_BIT_COUNT +
		int(bits.Bit_Size_64(bits.Word_64(
			workspace.Divisor[divisor_count-WORD_COUNT_INCREMENT],
		)))
	shift := int(precision)
	if order == Float_Unequal_Order(ORDER_BEFORE) {
		shift++
	}
	if aligned_bits+shift > BIT_COUNT_MAXIMUM {
		return 0, 0, 0
	}
	dividend_count := (aligned_bits + shift + WORD_BIT_INDEX_MAXIMUM) / WORD_BIT_COUNT
	return Float_Division_Divisor_Count(divisor_count),
		Float_Division_Dividend_Count(dividend_count), Float_Division_Scale(shift)
}

func float_division_word_scale(
	workspace Float_Division_Workspace_Handle,
	divisor Float_Division_Short_Count,
	dividend Float_Active_Word_Count,
	scale Float_Division_Active_Scale,
) {
	Float_Division_Workspace_Handle_Invariants(workspace, "float_division_word_scale.workspace")
	Float_Division_Short_Count_Invariants(divisor, "float_division_word_scale.divisor")
	Float_Active_Word_Count_Invariants(dividend, "float_division_word_scale.dividend")
	Float_Division_Active_Scale_Invariants(scale, "float_division_word_scale.scale")
	divisor_count := int(divisor)
	dividend_count := int(dividend)
	shift := int(scale)
	word_shift, bit_shift := shift/WORD_BIT_COUNT, uint(shift%WORD_BIT_COUNT)
	destination_index := dividend_count - WORD_COUNT_INCREMENT
	for destination_index >= WORD_COUNT_MINIMUM {
		source_index := destination_index - word_shift
		word := Word(0)
		if source_index >= WORD_COUNT_MINIMUM {
			if source_index < divisor_count {
				word = workspace.Remainder[source_index] << bit_shift
			}
			if bit_shift != 0 {
				if source_index > WORD_COUNT_MINIMUM {
					if source_index <= divisor_count {
						lower_index := source_index - WORD_COUNT_INCREMENT
						word |= workspace.Remainder[lower_index] >>
							(WORD_BIT_COUNT - bit_shift)
					}
				}
			}
		}
		workspace.Remainder[destination_index] = word
		destination_index--
	}
	workspace.Remainder[dividend_count] = 0
}

func float_set_quotient_words_commit(
	destination Float_Division_Destination_Handle,
	workspace Float_Division_Workspace_Handle,
	quotient_words Float_Division_Quotient_Words,
	divisor Float_Division_Short_Count,
	dividend Float_Active_Word_Count,
	precision_value Float_Division_Precision,
	exponent Float_Exponent,
	negative Polarity,
	mode Rounding_Mode,
) {
	Float_Division_Destination_Handle_Invariants(
		destination, "float_set_quotient_words_commit.destination",
	)
	Float_Division_Workspace_Handle_Invariants(
		workspace, "float_set_quotient_words_commit.workspace",
	)
	Float_Division_Quotient_Words_Invariants(
		quotient_words, "float_set_quotient_words_commit.quotient",
	)
	Float_Division_Short_Count_Invariants(divisor, "float_set_quotient_words_commit.divisor")
	Float_Active_Word_Count_Invariants(dividend, "float_set_quotient_words_commit.dividend")
	Float_Division_Precision_Invariants(
		precision_value, "float_set_quotient_words_commit.precision")
	Float_Exponent_Invariants(exponent, "float_set_quotient_words_commit.exponent")
	Polarity_Invariants(negative, "float_set_quotient_words_commit.negative")
	Rounding_Mode_Invariants(mode, "float_set_quotient_words_commit.mode")
	precision := Float_Active_Precision(precision_value)
	result := (*Float)((*Float_Division_Destination)(destination))
	previous_count := result.Mantissa.Count
	count, increment, accuracy := float_division_word_round(
		workspace, quotient_words, divisor, dividend, negative, mode,
	)
	result.Accuracy = accuracy
	quotient_count := int(count)
	copy(result.Mantissa.Words[:quotient_count], quotient_words[:quotient_count])
	result.Precision, result.Mode = Float_Precision(precision), mode
	result.Form, result.Negative = FLOAT_FORM_FINITE, negative
	result.Exponent = exponent
	result.Mantissa.Count = Word_Count(quotient_count)
	result.Mantissa.Negative = POLARITY_NONNEGATIVE
	if increment {
		float_increment_mantissa(
			(*Positive_Int)(&result.Mantissa), precision,
			&result.Exponent,
		)
	}
	if result.Exponent > FLOAT_EXPONENT_MAXIMUM {
		float_set_division_bound(
			result, Float_Nonfinite_Form(FLOAT_FORM_INFINITY), negative,
			precision, mode, true,
		)
		return
	}
	if previous_count > result.Mantissa.Count {
		int_clear(
			(*Int)(&result.Mantissa), result.Mantissa.Count, previous_count,
		)
	}
}

func float_division_word_round(
	workspace Float_Division_Workspace_Handle,
	quotient_words Float_Division_Quotient_Words,
	divisor Float_Division_Short_Count,
	dividend Float_Active_Word_Count,
	negative Polarity,
	mode Rounding_Mode,
) (count Float_Active_Word_Count, rounded Boolean, accuracy Accuracy) {
	defer func() {
		Float_Active_Word_Count_Invariants(count, "float_division_word_round.count")
		Boolean_Invariants(rounded, "float_division_word_round.rounded")
		Accuracy_Invariants(accuracy, "float_division_word_round.accuracy")
	}()
	Float_Division_Workspace_Handle_Invariants(
		workspace, "float_division_word_round.workspace")
	Float_Division_Quotient_Words_Invariants(
		quotient_words, "float_division_word_round.quotient",
	)
	Float_Division_Short_Count_Invariants(divisor, "float_division_word_round.divisor")
	Float_Active_Word_Count_Invariants(dividend, "float_division_word_round.dividend")
	Polarity_Invariants(negative, "float_division_word_round.negative")
	Rounding_Mode_Invariants(mode, "float_division_word_round.mode")
	dividend_count := int(dividend)
	divisor_count := int(divisor)
	quotient_count := dividend_count - divisor_count + WORD_COUNT_INCREMENT
	for quotient_count > WORD_COUNT_MINIMUM {
		if quotient_words[quotient_count-WORD_COUNT_INCREMENT] != 0 {
			break
		}
		quotient_count--
	}
	aver.Always(
		quotient_count > WORD_COUNT_MINIMUM,
		"Normalized finite division retains one guarded quotient bit.",
	)
	rounding_bit := Bit_Value(quotient_words[WORD_COUNT_MINIMUM] & Word(BIT_SET))
	for index := WORD_COUNT_MINIMUM; index < quotient_count; index++ {
		word := quotient_words[index] >> WORD_COUNT_INCREMENT
		if index+WORD_COUNT_INCREMENT < quotient_count {
			word |= quotient_words[index+WORD_COUNT_INCREMENT] << WORD_BIT_INDEX_MAXIMUM
		}
		quotient_words[index] = word
	}
	for quotient_count > WORD_COUNT_MINIMUM {
		if quotient_words[quotient_count-WORD_COUNT_INCREMENT] != 0 {
			break
		}
		quotient_count--
	}
	sticky_bit := BIT_CLEAR
	for index := WORD_COUNT_MINIMUM; index < divisor_count; index++ {
		if workspace.Remainder[index] != 0 {
			sticky_bit = BIT_SET
			break
		}
	}
	increment := float_rounding_increment(
		mode, negative, rounding_bit, sticky_bit,
		Bit_Value(quotient_words[WORD_COUNT_MINIMUM]&Word(BIT_SET)),
	)
	accuracy = ACCURACY_EXACT
	if rounding_bit != BIT_CLEAR {
		accuracy = Accuracy(float_accuracy(increment, negative))
	} else if sticky_bit != BIT_CLEAR {
		accuracy = Accuracy(float_accuracy(increment, negative))
	}
	return Float_Active_Word_Count(quotient_count), increment, accuracy
}

func float_division_word_quotient(
	workspace Float_Division_Workspace_Handle,
	quotient Float_Division_Quotient_Words,
	divisor Float_Division_Short_Count,
	dividend Float_Active_Word_Count,
) {
	Float_Division_Workspace_Handle_Invariants(
		workspace, "float_division_word_quotient.workspace")
	Float_Division_Quotient_Words_Invariants(
		quotient, "float_division_word_quotient.quotient",
	)
	Float_Division_Short_Count_Invariants(
		divisor, "float_division_word_quotient.divisor",
	)
	Float_Active_Word_Count_Invariants(
		dividend, "float_division_word_quotient.dividend",
	)
	divisor_count := int(divisor)
	dividend_count := int(dividend)
	aver.Always(
		dividend_count >= divisor_count,
		"Guard scaling leaves at least one quotient word.",
	)
	quotient_count := dividend_count - divisor_count + WORD_COUNT_INCREMENT
	for index := WORD_COUNT_MINIMUM; index < quotient_count; index++ {
		quotient[index] = 0
	}
	if divisor_count == WORD_COUNT_INCREMENT {
		float_division_one_word_quotient(
			workspace, quotient, dividend,
		)
		return
	}
	float_division_two_word_quotient(
		workspace, quotient, Float_Division_Multiword_Count(dividend),
	)
}

func float_division_one_word_quotient(
	workspace Float_Division_Workspace_Handle,
	quotient Float_Division_Quotient_Words,
	count Float_Active_Word_Count,
) {
	Float_Division_Workspace_Handle_Invariants(
		workspace, "float_division_one_word_quotient.workspace",
	)
	Float_Division_Quotient_Words_Invariants(
		quotient, "float_division_one_word_quotient.quotient",
	)
	Float_Active_Word_Count_Invariants(
		count, "float_division_one_word_quotient.count",
	)
	remainder := Word(0)
	index := int(count) - WORD_COUNT_INCREMENT
	for index >= WORD_COUNT_MINIMUM {
		word, rest := bits.Divide_64(
			bits.Dividend_High_64(remainder),
			bits.Dividend_Low_64(workspace.Remainder[index]),
			bits.Divisor_64(workspace.Divisor[WORD_COUNT_MINIMUM]),
		)
		quotient[index] = Word(word)
		remainder = Word(rest)
		index--
	}
	workspace.Remainder[WORD_COUNT_MINIMUM] = remainder
}

func float_division_two_word_quotient(
	workspace Float_Division_Workspace_Handle,
	quotient Float_Division_Quotient_Words,
	count Float_Division_Multiword_Count,
) {
	Float_Division_Workspace_Handle_Invariants(
		workspace, "float_division_two_word_quotient.workspace",
	)
	Float_Division_Quotient_Words_Invariants(
		quotient, "float_division_two_word_quotient.quotient",
	)
	Float_Division_Multiword_Count_Invariants(
		count, "float_division_two_word_quotient.count",
	)
	float_division_two_word_normalize(workspace, count)
	quotient_count := int(count) - BASE_BINARY + WORD_COUNT_INCREMENT
	offset := quotient_count - WORD_COUNT_INCREMENT
	for offset >= WORD_COUNT_MINIMUM {
		high_index := offset
		high_index += BASE_BINARY
		next_index := high_index - WORD_COUNT_INCREMENT
		third_index := next_index - WORD_COUNT_INCREMENT
		trial := Int_Division_Small_Trial{
			Dividend: Int_Division_Small_Trial_Dividend{
				High: Int_Division_Trial_High(workspace.Remainder[high_index]),
				Next: Int_Division_Trial_Next(workspace.Remainder[next_index]),
				Third: Int_Division_Trial_Third(
					workspace.Remainder[third_index],
				),
			},
			Divisor: Int_Division_Small_Trial_Divisor{
				High: Int_Division_Normalized_Word(
					workspace.Divisor[WORD_COUNT_INCREMENT],
				),
				Next: workspace.Divisor[WORD_COUNT_MINIMUM],
			},
		}
		estimate := int_divide_small_estimate(trial)
		float_division_word_subtract(
			workspace, quotient, Float_Division_Offset(offset), estimate,
		)
		offset--
	}
}

func float_division_two_word_normalize(
	workspace Float_Division_Workspace_Handle,
	count Float_Division_Multiword_Count,
) {
	Float_Division_Workspace_Handle_Invariants(
		workspace, "float_division_two_word_normalize.workspace",
	)
	Float_Division_Multiword_Count_Invariants(
		count, "float_division_two_word_normalize.count",
	)
	shift := uint(bits.Leading_Zeros_64(bits.Word_64(
		workspace.Divisor[WORD_COUNT_INCREMENT],
	)))
	if shift == 0 {
		return
	}
	carry := Word(0)
	dividend_count := int(count)
	for index := WORD_COUNT_MINIMUM; index < dividend_count; index++ {
		word := workspace.Remainder[index]
		workspace.Remainder[index] = word<<shift | carry
		carry = word >> (WORD_BIT_COUNT - shift)
	}
	workspace.Remainder[dividend_count] = carry
	low := workspace.Divisor[WORD_COUNT_MINIMUM]
	workspace.Divisor[WORD_COUNT_INCREMENT] =
		workspace.Divisor[WORD_COUNT_INCREMENT]<<shift |
			low>>(WORD_BIT_COUNT-shift)
	workspace.Divisor[WORD_COUNT_MINIMUM] = low << shift
}

func float_division_word_subtract(
	workspace Float_Division_Workspace_Handle,
	quotient Float_Division_Quotient_Words,
	start Float_Division_Offset,
	estimate Word,
) {
	Float_Division_Workspace_Handle_Invariants(
		workspace, "float_division_word_subtract.workspace")
	Float_Division_Quotient_Words_Invariants(
		quotient, "float_division_word_subtract.quotient",
	)
	Float_Division_Offset_Invariants(
		start, "float_division_word_subtract.start",
	)
	Word_Invariants(estimate, "float_division_word_subtract.estimate")
	offset := int(start)
	carry, borrow := uint64(0), uint64(0)
	for divisor_index := WORD_COUNT_MINIMUM; divisor_index < BASE_BINARY; divisor_index++ {
		left, right := uint64(estimate), uint64(workspace.Divisor[divisor_index])
		left_low, right_low := left&TEXT_DIVISION_LIMB_MASK,
			right&TEXT_DIVISION_LIMB_MASK
		left_high, right_high := left>>TEXT_DIVISION_LIMB_BIT_COUNT,
			right>>TEXT_DIVISION_LIMB_BIT_COUNT
		partial, product_low := left_low*right_low, left*right
		middle_first := left_high*right_low + partial>>TEXT_DIVISION_LIMB_BIT_COUNT
		middle_second := left_low*right_high + middle_first&TEXT_DIVISION_LIMB_MASK
		product_high := left_high*right_high +
			middle_first>>TEXT_DIVISION_LIMB_BIT_COUNT +
			middle_second>>TEXT_DIVISION_LIMB_BIT_COUNT
		product_low += carry
		if product_low < carry {
			product_high++
		}
		position := offset
		position += divisor_index
		minuend := uint64(workspace.Remainder[position])
		difference := minuend - product_low - borrow
		workspace.Remainder[position] = Word(difference)
		borrow = ((^minuend & product_low) |
			(^(minuend ^ product_low) & difference)) >> WORD_BIT_INDEX_MAXIMUM
		carry = product_high
	}
	high_position := offset
	high_position += BASE_BINARY
	minuend := uint64(workspace.Remainder[high_position])
	difference := minuend - carry - borrow
	workspace.Remainder[high_position] = Word(difference)
	borrow = ((^minuend & carry) |
		(^(minuend ^ carry) & difference)) >> WORD_BIT_INDEX_MAXIMUM
	if borrow == 0 {
		quotient[offset] = estimate
		return
	}
	quotient[offset] = estimate - Word(bits.CARRY_MAXIMUM)
	carry = 0
	for divisor_index := WORD_COUNT_MINIMUM; divisor_index < BASE_BINARY; divisor_index++ {
		position := offset
		position += divisor_index
		left := uint64(workspace.Remainder[position])
		right := uint64(workspace.Divisor[divisor_index])
		sum := left + right + carry
		workspace.Remainder[position] = Word(sum)
		carry = ((left & right) | ((left | right) &^ sum)) >> WORD_BIT_INDEX_MAXIMUM
	}
	workspace.Remainder[high_position] += Word(carry)
}

func float_division_remainder_nonzero(
	workspace Float_Division_Workspace_Handle, count Float_Division_Active_Word_Count,
) (nonzero Boolean) {
	defer func() {
		Boolean_Invariants(nonzero, "float_division_remainder_nonzero.nonzero")
	}()
	Float_Division_Workspace_Handle_Invariants(
		workspace, "float_division_remainder_nonzero.workspace",
	)
	Float_Division_Active_Word_Count_Invariants(
		count, "float_division_remainder_nonzero.count",
	)
	for index := WORD_COUNT_MINIMUM; index < int(count); index++ {
		if workspace.Remainder[index] != 0 {
			return true
		}
	}
	return false
}

// Int_Quotient_Remainder implements truncated division without exposing partial results.
func Int_Quotient_Remainder(
	quotient Int_Handle,
	remainder Int_Handle,
	dividend Int_Handle,
	divisor Int_Handle,
	workspace Int_Division_Workspace_Handle,
) (status Division_Status) {
	defer func() {
		Division_Status_Invariants(status, "int_quotient_remainder.status")
	}()
	Int_Handle_Invariants(quotient, "int_quotient_remainder.quotient")
	Int_Handle_Invariants(remainder, "int_quotient_remainder.remainder")
	Int_Handle_Invariants(dividend, "int_quotient_remainder.dividend")
	Int_Handle_Invariants(divisor, "int_quotient_remainder.divisor")
	Int_Division_Workspace_Handle_Invariants(workspace, "int_quotient_remainder.workspace")
	if quotient == remainder {
		return STATUS_DESTINATIONS_OVERLAP
	}
	quotient_negative := POLARITY_NONNEGATIVE
	if dividend.Negative != divisor.Negative {
		quotient_negative = POLARITY_NEGATIVE
	}
	remainder_negative := dividend.Negative
	// One-word division produces both outputs atomically without restoring-division scratch.
	if dividend.Count <= Word_Count(WORD_COUNT_INCREMENT) {
		if divisor.Count == Word_Count(WORD_COUNT_INCREMENT) {
			dividend_word := Word(0)
			if dividend.Count != Word_Count(WORD_COUNT_MINIMUM) {
				dividend_word = dividend.Words[WORD_COUNT_MINIMUM]
			}
			divisor_word := divisor.Words[WORD_COUNT_MINIMUM]
			quotient_word := dividend_word / divisor_word
			remainder_word := dividend_word % divisor_word
			workspace.Quotient[WORD_COUNT_MINIMUM] = quotient_word
			workspace.Remainder[WORD_COUNT_MINIMUM] = remainder_word
			workspace.Quotient_Count = Quotient_Count(WORD_COUNT_INCREMENT)
			workspace.Remainder_Count = Remainder_Count(WORD_COUNT_INCREMENT)
			if quotient_word == 0 {
				workspace.Quotient_Count = Quotient_Count(WORD_COUNT_MINIMUM)
			}
			if remainder_word == 0 {
				workspace.Remainder_Count = Remainder_Count(WORD_COUNT_MINIMUM)
			}
			int_set_word(quotient, quotient_word, quotient_negative)
			int_set_word(remainder, remainder_word, remainder_negative)
			return STATUS_OK
		}
	}
	int_divide_magnitudes(workspace, dividend, divisor)
	if divisor.Count == WORD_COUNT_MINIMUM {
		return STATUS_DIVISOR_ZERO
	}
	int_set_division_quotient(quotient, workspace, quotient_negative)
	int_set_division_remainder(remainder, workspace, remainder_negative)
	return STATUS_OK
}

// Int_Quotient writes truncated quotient without manufacturing an unused remainder Int.
func Int_Quotient(
	destination Int_Handle, dividend Int_Handle, divisor Int_Handle,
	workspace Int_Division_Workspace_Handle,
) (status Divisor_Status) {
	defer func() { Divisor_Status_Invariants(status, "int_quotient.status") }()
	Int_Handle_Invariants(destination, "int_quotient.destination")
	Int_Handle_Invariants(dividend, "int_quotient.dividend")
	Int_Handle_Invariants(divisor, "int_quotient.divisor")
	Int_Division_Workspace_Handle_Invariants(workspace, "int_quotient.workspace")
	if divisor.Count == Word_Count(WORD_COUNT_MINIMUM) {
		return STATUS_DIVISOR_ZERO
	}
	// One-word division is already atomic, so restoring-division scratch adds no safety.
	if dividend.Count <= Word_Count(WORD_COUNT_INCREMENT) {
		if divisor.Count == Word_Count(WORD_COUNT_INCREMENT) {
			dividend_word := Word(0)
			if dividend.Count != Word_Count(WORD_COUNT_MINIMUM) {
				dividend_word = dividend.Words[WORD_COUNT_MINIMUM]
			}
			divisor_word := divisor.Words[WORD_COUNT_MINIMUM]
			negative := POLARITY_NONNEGATIVE
			if dividend.Negative != divisor.Negative {
				negative = POLARITY_NEGATIVE
			}
			word := dividend_word / divisor_word
			previous_count := destination.Count
			destination.Count = Word_Count(WORD_COUNT_MINIMUM)
			destination.Negative = POLARITY_NONNEGATIVE
			if word != 0 {
				destination.Words[WORD_COUNT_MINIMUM] = word
				destination.Count = Word_Count(WORD_COUNT_INCREMENT)
				destination.Negative = negative
			}
			if previous_count > destination.Count {
				int_clear(destination, destination.Count, previous_count)
			}
			return STATUS_OK
		}
	}
	negative := POLARITY_NONNEGATIVE
	if dividend.Negative != divisor.Negative {
		negative = POLARITY_NEGATIVE
	}
	int_divide_magnitudes(workspace, dividend, divisor)
	if divisor.Count == WORD_COUNT_MINIMUM {
		return STATUS_DIVISOR_ZERO
	}
	int_set_division_quotient(destination, workspace, negative)
	return STATUS_OK
}

// Int_Remainder writes truncated remainder with dividend sign.
func Int_Remainder(
	destination Int_Handle, dividend Int_Handle, divisor Int_Handle,
	workspace Int_Division_Workspace_Handle,
) (status Divisor_Status) {
	defer func() { Divisor_Status_Invariants(status, "int_remainder.status") }()
	Int_Handle_Invariants(destination, "int_remainder.destination")
	Int_Handle_Invariants(dividend, "int_remainder.dividend")
	Int_Handle_Invariants(divisor, "int_remainder.divisor")
	Int_Division_Workspace_Handle_Invariants(workspace, "int_remainder.workspace")
	if divisor.Count == Word_Count(WORD_COUNT_MINIMUM) {
		return STATUS_DIVISOR_ZERO
	}
	// One-word remainder is atomic and does not need restoring-division scratch.
	if dividend.Count <= Word_Count(WORD_COUNT_INCREMENT) {
		if divisor.Count == Word_Count(WORD_COUNT_INCREMENT) {
			dividend_word := Word(0)
			if dividend.Count != Word_Count(WORD_COUNT_MINIMUM) {
				dividend_word = dividend.Words[WORD_COUNT_MINIMUM]
			}
			negative := dividend.Negative
			word := dividend_word % divisor.Words[WORD_COUNT_MINIMUM]
			previous_count := destination.Count
			destination.Count = Word_Count(WORD_COUNT_MINIMUM)
			destination.Negative = POLARITY_NONNEGATIVE
			if word != 0 {
				destination.Words[WORD_COUNT_MINIMUM] = word
				destination.Count = Word_Count(WORD_COUNT_INCREMENT)
				destination.Negative = negative
			}
			if previous_count > destination.Count {
				int_clear(destination, destination.Count, previous_count)
			}
			return STATUS_OK
		}
	}
	int_divide_magnitudes(workspace, dividend, divisor)
	if divisor.Count == WORD_COUNT_MINIMUM {
		return STATUS_DIVISOR_ZERO
	}
	int_set_division_remainder(destination, workspace, dividend.Negative)
	return STATUS_OK
}

// DOUBLE_WORD_MAGNITUDE_WORD_COUNT binds the scalar path to its exact capacity.
const DOUBLE_WORD_MAGNITUDE_WORD_COUNT = WORD_COUNT_INCREMENT + WORD_COUNT_INCREMENT

// INT_REFERENCE_DESTINATION_INDEX keeps the mutable result separate from both inputs.
const INT_REFERENCE_DESTINATION_INDEX = 0

// INT_REFERENCE_LEFT_INDEX preserves the public operand order inside helper boundaries.
const INT_REFERENCE_LEFT_INDEX = INT_REFERENCE_DESTINATION_INDEX + 1

// INT_REFERENCE_RIGHT_INDEX preserves the second public operand without another parameter type.
const INT_REFERENCE_RIGHT_INDEX = INT_REFERENCE_LEFT_INDEX + 1

// INT_REFERENCE_COUNT binds the complete validated boundary crossing.
const INT_REFERENCE_COUNT = INT_REFERENCE_RIGHT_INDEX + 1

// Active_Double_Word_Magnitude holds one nonzero magnitude without repeated field types.
type Active_Double_Word_Magnitude struct {
	// Low preserves low bits independently of high-word normalization.
	Low Word
	// High retains overflow beyond one machine word.
	High Double_Word_High
}

// Active_Double_Word_Magnitude_Invariants fixes both words and excludes zero.
func Active_Double_Word_Magnitude_Invariants(
	value Active_Double_Word_Magnitude, namespace aver.Namespace,
) {
	Word_Invariants(value.Low, namespace)
	Double_Word_High_Invariants(value.High, namespace)
	aver.Always(
		value.Low|Word(value.High) != 0,
		"An active double word is nonzero.",
	)
}

// Int_Destination_Reference prevents destination identity from merging with operand identity.
type Int_Destination_Reference *Int_Destination

// Int_Destination_Reference_Invariants rejects absent destination storage.
func Int_Destination_Reference_Invariants(
	value Int_Destination_Reference, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Int_Destination_Invariants(*value, namespace)
}

// Int_Left_Reference prevents left-operand identity from merging with other roles.
type Int_Left_Reference *Int_Left

// Int_Left_Reference_Invariants rejects an absent left operand.
func Int_Left_Reference_Invariants(value Int_Left_Reference, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Int_Left_Invariants(*value, namespace)
}

// Int_Right_Reference prevents right-operand identity from merging with other roles.
type Int_Right_Reference *Int_Right

// Int_Right_Reference_Invariants rejects an absent right operand.
func Int_Right_Reference_Invariants(value Int_Right_Reference, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Int_Right_Invariants(*value, namespace)
}

// Int_References carries validated Int addresses through internal helper boundaries.
type Int_References struct {
	// Destination stays distinct when caller storage aliases an operand.
	Destination Int_Destination_Reference
	// Left stays distinct when caller storage aliases the destination.
	Left Int_Left_Reference
	// Right stays distinct when caller storage aliases another role.
	Right Int_Right_Reference
}

// Int_References_Invariants rejects absent addresses without repeating Int in one chain.
func Int_References_Invariants(value Int_References, namespace aver.Namespace) {
	Int_Destination_Reference_Invariants(value.Destination, namespace)
	Int_Left_Reference_Invariants(value.Left, namespace)
	Int_Right_Reference_Invariants(value.Right, namespace)
}

// Euclidean_Index selects one of the two live magnitudes; the third slot remains spare.
type Euclidean_Index int

// Euclidean_Index_Invariants excludes the inactive remainder slot from binary reduction.
func Euclidean_Index_Invariants(value Euclidean_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Int(int(value), EUCLIDEAN_DIVIDEND_INDEX, EUCLIDEAN_DIVISOR_INDEX).
		Ensure()
}

// Word_Shift counts complete-word alignment without admitting the absent end index.
type Word_Shift Word_Count

// Word_Shift_Invariants derives its maximum from the last writable word.
func Word_Shift_Invariants(value Word_Shift, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), WORD_COUNT_MINIMUM, WORD_INDEX_MAXIMUM).
		Ensure()
}

// Word_Factor is one nonzero scalar quotient approximation.
type Word_Factor Word

// Word_Factor_Invariants excludes the no-progress factor.
func Word_Factor_Invariants(value Word_Factor, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(
			uint64(value), uint64(bits.CARRY_MAXIMUM), uint64(bits.WORD_64_MAXIMUM),
		).
		Ensure()
}

// Word_Scale is one word factor shifted by complete words.
type Word_Scale struct {
	// Shift carries whole-word alignment so Factor remains one machine word.
	Shift Word_Shift
	// Factor skips repeated subtraction without retaining a full quotient.
	Factor Word_Factor
}

// Word_Scale_Invariants fixes the bounded shift and scalar factor.
func Word_Scale_Invariants(value Word_Scale, namespace aver.Namespace) {
	Word_Shift_Invariants(value.Shift, namespace)
	Word_Factor_Invariants(value.Factor, namespace)
}

// Int_Greatest_Common_Divisor removes powers of two before subtraction because bit-at-a-time
// division spends work producing a quotient that normalization never consumes.
func Int_Greatest_Common_Divisor(
	destination Int_Handle,
	left Int_Handle,
	right Int_Handle,
	workspace Int_Greatest_Common_Divisor_Workspace_Handle,
) {
	Int_Handle_Invariants(destination, "int_greatest_common_divisor.destination")
	Int_Handle_Invariants(left, "int_greatest_common_divisor.left")
	Int_Handle_Invariants(right, "int_greatest_common_divisor.right")
	Int_Greatest_Common_Divisor_Workspace_Handle_Invariants(
		workspace, "int_greatest_common_divisor.workspace",
	)
	references := Int_References{
		Destination: Int_Destination_Reference((*Int_Destination)((*Int)(destination))),
		Left:        Int_Left_Reference((*Int_Left)((*Int)(left))),
		Right:       Int_Right_Reference((*Int_Right)((*Int)(right))),
	}
	if left.Count == WORD_COUNT_MINIMUM {
		Int_Set(destination, right)
		destination.Negative = POLARITY_NONNEGATIVE
		return
	}
	if right.Count == WORD_COUNT_MINIMUM {
		Int_Set(destination, left)
		destination.Negative = POLARITY_NONNEGATIVE
		return
	}
	if left.Count <= Word_Count(WORD_COUNT_INCREMENT) {
		if right.Count <= Word_Count(WORD_COUNT_INCREMENT) {
			dividend_word := left.Words[WORD_COUNT_MINIMUM]
			divisor_word := right.Words[WORD_COUNT_MINIMUM]
			for divisor_word != 0 {
				remainder := dividend_word % divisor_word
				dividend_word, divisor_word = divisor_word, remainder
			}
			int_set_word(destination, dividend_word, POLARITY_NONNEGATIVE)
			return
		}
	}
	two_word_count := Word_Count(WORD_COUNT_INCREMENT + WORD_COUNT_INCREMENT)
	if left.Count <= two_word_count {
		if right.Count <= two_word_count {
			left_value := Active_Double_Word_Magnitude{
				Low: left.Words[WORD_COUNT_MINIMUM],
			}
			if left.Count == two_word_count {
				left_value.High = Double_Word_High(left.Words[WORD_COUNT_INCREMENT])
			}
			right_value := Active_Double_Word_Magnitude{
				Low: right.Words[WORD_COUNT_MINIMUM],
			}
			if right.Count == two_word_count {
				right_value.High =
					Double_Word_High(right.Words[WORD_COUNT_INCREMENT])
			}
			int_greatest_common_divisor_double_word(
				references.Destination, left_value, right_value)
			return
		}
	}
	int_greatest_common_divisor_multiword(
		references.Destination, (*Nonzero_Int)((*Int)(left)), (*Nonzero_Int)((*Int)(right)),
		(*Int_Left)((*Int)(&workspace.Integers.Dividend)),
		(*Int_Right)((*Int)(&workspace.Integers.Divisor)),
	)
}

func int_greatest_common_divisor_double_word(
	destination Int_Destination_Reference, left_value, right_value Active_Double_Word_Magnitude,
) {
	Int_Destination_Reference_Invariants(destination, "int_gcd_double_word.destination")
	Active_Double_Word_Magnitude_Invariants(left_value, "int_gcd_double_word.left")
	Active_Double_Word_Magnitude_Invariants(right_value, "int_gcd_double_word.right")
	for left_value.Low != right_value.Low || left_value.High != right_value.High {
		double_word_magnitude_reduce(&left_value, &right_value)
	}
	double_word_magnitude_commit(destination, left_value)
}

func double_word_magnitude_reduce(
	left Active_Double_Word_Magnitude_Handle, right Active_Double_Word_Magnitude_Handle,
) {
	Active_Double_Word_Magnitude_Handle_Invariants(left, "double_word_reduce.left")
	Active_Double_Word_Magnitude_Handle_Invariants(right, "double_word_reduce.right")
	larger, smaller := left, right
	if left.High < right.High {
		larger, smaller = right, left
	} else if left.High == right.High {
		if left.Low < right.Low {
			larger, smaller = right, left
		}
	}
	double_word_magnitude_remainder(larger, *smaller)
}

func double_word_magnitude_remainder(
	larger Active_Double_Word_Magnitude_Handle, smaller Active_Double_Word_Magnitude,
) {
	Active_Double_Word_Magnitude_Handle_Invariants(larger, "double_word_remainder.larger")
	Active_Double_Word_Magnitude_Invariants(smaller, "double_word_remainder.smaller")
	smaller_word := Word(smaller.Low)
	if smaller.High == 0 {
		high_remainder := Word(larger.High) % smaller_word
		_, remainder := bits.Divide_64(
			bits.Dividend_High_64(high_remainder),
			bits.Dividend_Low_64(larger.Low),
			bits.Divisor_64(smaller.Low),
		)
		larger.Low = Word(remainder)
		larger.High = 0
		if larger.Low == 0 {
			larger.Low = smaller_word
			larger.High = smaller.High
		}
		return
	}
	quotient := Word(WORD_COUNT_INCREMENT)
	if smaller.High != Double_Word_High(bits.WORD_64_MAXIMUM) {
		quotient = Word(larger.High) / (Word(smaller.High) + Word(WORD_COUNT_INCREMENT))
		quotient = max(quotient, Word(WORD_COUNT_INCREMENT))
	}
	word_32_mask := Word(bits.WORD_32_MAXIMUM)
	smaller_low := smaller_word & word_32_mask
	smaller_high := smaller_word >> bits.BIT_COUNT_32_MAXIMUM
	quotient_low := quotient & word_32_mask
	quotient_high := quotient >> bits.BIT_COUNT_32_MAXIMUM
	product_low_part := smaller_low * quotient_low
	middle := smaller_high*quotient_low +
		product_low_part>>bits.BIT_COUNT_32_MAXIMUM
	middle_low := middle & word_32_mask
	middle_high := middle >> bits.BIT_COUNT_32_MAXIMUM
	middle_low += smaller_low * quotient_high
	product_low_high := smaller_high*quotient_high +
		middle_high + middle_low>>bits.BIT_COUNT_32_MAXIMUM
	product_low := smaller_word * quotient
	product_high := product_low_high + Word(smaller.High)*quotient
	minuend := larger.Low
	difference := minuend - product_low
	borrow := ((^minuend & product_low) |
		(^(minuend ^ product_low) & difference)) >> WORD_BIT_INDEX_MAXIMUM
	larger.Low = difference
	larger.High = Double_Word_High(Word(larger.High) - product_high - borrow)
}

func double_word_magnitude_commit(
	destination_reference Int_Destination_Reference,
	value Active_Double_Word_Magnitude,
) {
	Int_Destination_Reference_Invariants(
		destination_reference, "double_word_commit.destination",
	)
	Active_Double_Word_Magnitude_Invariants(value, "double_word_commit.value")
	destination := (*Int)((*Int_Destination)(destination_reference))
	previous_count := destination.Count
	destination.Words[WORD_COUNT_MINIMUM] = value.Low
	destination.Count = Word_Count(WORD_COUNT_INCREMENT)
	if value.High != 0 {
		destination.Words[WORD_COUNT_INCREMENT] = Word(value.High)
		destination.Count += Word_Count(WORD_COUNT_INCREMENT)
	}
	destination.Negative = POLARITY_NONNEGATIVE
	if previous_count > destination.Count {
		int_clear(destination, destination.Count, previous_count)
	}
}

func int_greatest_common_divisor_multiword(
	destination_reference Int_Destination_Reference, left, right Nonzero_Int_Handle,
	dividend_reference Int_Left_Reference, divisor_reference Int_Right_Reference,
) {
	Int_Destination_Reference_Invariants(destination_reference, "int_gcd_multiword.destination")
	Nonzero_Int_Handle_Invariants(left, "int_gcd_multiword.left")
	Nonzero_Int_Handle_Invariants(right, "int_gcd_multiword.right")
	Int_Left_Reference_Invariants(dividend_reference, "int_gcd_multiword.dividend")
	Int_Right_Reference_Invariants(divisor_reference, "int_gcd_multiword.divisor")
	destination := (*Int)((*Int_Destination)(destination_reference))
	dividend := (*Int)((*Int_Left)(dividend_reference))
	divisor := (*Int)((*Int_Right)(divisor_reference))
	Int_Absolute(dividend, (*Int)((*Nonzero_Int)(left)))
	Int_Absolute(divisor, (*Int)((*Nonzero_Int)(right)))
	dividend_zero_count := Int_Trailing_Zero_Bit_Count(dividend)
	divisor_zero_count := Int_Trailing_Zero_Bit_Count(divisor)
	common_zero_count := min(dividend_zero_count, divisor_zero_count)
	Int_Shift_Right(dividend, dividend, Shift_Count(dividend_zero_count))
	Int_Shift_Right(divisor, divisor, Shift_Count(divisor_zero_count))
	scale := Word_Scale{Factor: Word_Factor(WORD_COUNT_INCREMENT)}
	for order := ORDER_BEFORE; order != ORDER_SAME; {
		order = ORDER_SAME
		if dividend.Count < divisor.Count {
			order = ORDER_BEFORE
		} else if dividend.Count > divisor.Count {
			order = ORDER_AFTER
		} else {
			index := int(dividend.Count) - WORD_COUNT_INCREMENT
			for ; index >= 0; index-- {
				if dividend.Words[index] < divisor.Words[index] {
					order = ORDER_BEFORE
					break
				}
				if dividend.Words[index] > divisor.Words[index] {
					order = ORDER_AFTER
					break
				}
			}
		}
		if order == ORDER_SAME {
			break
		}
		larger := (*Positive_Magnitude)(dividend)
		smaller := (*Positive_Magnitude)(divisor)
		if order == ORDER_BEFORE {
			larger, smaller = smaller, larger
		}
		// Once the odd GCD is one, only the saved common power of two remains.
		if smaller.Count == Word_Count(WORD_COUNT_INCREMENT) {
			if smaller.Words[WORD_COUNT_MINIMUM] == Word(WORD_COUNT_INCREMENT) {
				Int_Set(dividend, (*Int)(smaller))
				break
			}
		}
		int_greatest_common_divisor_scale(larger, smaller, &scale)
		int_greatest_common_divisor_subtract(larger, smaller, &scale)
		if larger.Count == WORD_COUNT_MINIMUM {
			Int_Set((*Int)(larger), (*Int)(smaller))
			break
		}
		int_greatest_common_divisor_make_odd(larger)
	}
	if common_zero_count == BIT_COUNT_MINIMUM {
		Int_Set(destination, dividend)
		return
	}
	status := Int_Shift_Left(destination, dividend, Shift_Count(common_zero_count))
	aver.Always(
		status == Arithmetic_Status(STATUS_OK),
		"Restoring a common input factor cannot exceed either input magnitude.",
	)
}

func int_greatest_common_divisor_scale(
	larger Positive_Magnitude_Handle,
	smaller Positive_Magnitude_Handle,
	scale Word_Scale_Handle,
) {
	Positive_Magnitude_Handle_Invariants(larger, "int_gcd_scale.larger")
	Positive_Magnitude_Handle_Invariants(smaller, "int_gcd_scale.smaller")
	Word_Scale_Handle_Invariants(scale, "int_gcd_scale.scale")
	scale.Shift = Word_Shift(WORD_COUNT_MINIMUM)
	scale.Factor = Word_Factor(WORD_COUNT_INCREMENT)
	larger_high_index := int(larger.Count) - WORD_COUNT_INCREMENT
	larger_high := larger.Words[larger_high_index]
	smaller_high := smaller.Words[int(smaller.Count)-WORD_COUNT_INCREMENT]
	if larger.Count == smaller.Count {
		if smaller_high != Word(bits.WORD_64_MAXIMUM) {
			scale.Factor = Word_Factor(larger_high /
				(smaller_high + Word(WORD_COUNT_INCREMENT)))
			scale.Factor = max(scale.Factor, Word_Factor(WORD_COUNT_INCREMENT))
		}
		return
	}
	scale.Shift = Word_Shift(larger.Count - smaller.Count)
	scale.Factor = 0
	if smaller_high != Word(bits.WORD_64_MAXIMUM) {
		denominator := smaller_high + Word(WORD_COUNT_INCREMENT)
		scale.Factor = Word_Factor(larger_high / denominator)
	}
	if scale.Factor != 0 {
		return
	}
	scale.Shift--
	larger_low := larger.Words[larger_high_index-WORD_COUNT_INCREMENT]
	if smaller_high == Word(bits.WORD_64_MAXIMUM) {
		scale.Factor = Word_Factor(larger_high)
		return
	}
	denominator := smaller_high + Word(WORD_COUNT_INCREMENT)
	quotient, _ := bits.Divide_64(
		bits.Dividend_High_64(larger_high),
		bits.Dividend_Low_64(larger_low),
		bits.Divisor_64(denominator),
	)
	scale.Factor = max(Word_Factor(quotient), Word_Factor(WORD_COUNT_INCREMENT))
}

func int_greatest_common_divisor_subtract(
	larger Positive_Magnitude_Handle,
	smaller Positive_Magnitude_Handle,
	scale Word_Scale_Handle,
) {
	Positive_Magnitude_Handle_Invariants(larger, "int_gcd_subtract.larger")
	Positive_Magnitude_Handle_Invariants(smaller, "int_gcd_subtract.smaller")
	Word_Scale_Handle_Invariants(scale, "int_gcd_subtract.scale")
	borrow := Word(0)
	carry := Word(0)
	for index := WORD_COUNT_MINIMUM; index < int(smaller.Count); index++ {
		subtrahend := smaller.Words[index]
		if scale.Factor != Word_Factor(WORD_COUNT_INCREMENT) {
			word_32_mask := Word(bits.WORD_32_MAXIMUM)
			multiplicand_low := smaller.Words[index] & word_32_mask
			multiplicand_high := smaller.Words[index] >> bits.BIT_COUNT_32_MAXIMUM
			factor_low := Word(scale.Factor) & word_32_mask
			factor_high := Word(scale.Factor) >> bits.BIT_COUNT_32_MAXIMUM
			product_low_part := multiplicand_low * factor_low
			middle := multiplicand_high*factor_low +
				product_low_part>>bits.BIT_COUNT_32_MAXIMUM
			middle_low := middle & word_32_mask
			middle_high := middle >> bits.BIT_COUNT_32_MAXIMUM
			middle_low += multiplicand_low * factor_high
			product_high := multiplicand_high*factor_high +
				middle_high + middle_low>>bits.BIT_COUNT_32_MAXIMUM
			product_low := smaller.Words[index] * Word(scale.Factor)
			subtrahend = product_low + carry
			carry_overflow := Word(0)
			if subtrahend < product_low {
				carry_overflow = Word(WORD_COUNT_INCREMENT)
			}
			carry = product_high + carry_overflow
		}
		destination_index := index + int(scale.Shift)
		minuend := larger.Words[destination_index]
		difference := minuend - subtrahend - borrow
		larger.Words[destination_index] = difference
		borrow = ((^minuend & subtrahend) |
			(^(minuend ^ subtrahend) & difference)) >> WORD_BIT_INDEX_MAXIMUM
	}
	carry_index := int(smaller.Count) + int(scale.Shift)
	if carry_index < int(larger.Count) {
		minuend := larger.Words[carry_index]
		difference := minuend - carry - borrow
		larger.Words[carry_index] = difference
		borrow = ((^minuend & carry) |
			(^(minuend ^ carry) & difference)) >> WORD_BIT_INDEX_MAXIMUM
	}
	aver.Always(
		borrow == 0,
		"A conservative quotient keeps the scaled divisor below the dividend.",
	)
	for larger.Count > WORD_COUNT_MINIMUM {
		if larger.Words[int(larger.Count)-WORD_COUNT_INCREMENT] != 0 {
			break
		}
		larger.Count--
	}
}

func int_greatest_common_divisor_make_odd(larger Positive_Magnitude_Handle) {
	Positive_Magnitude_Handle_Invariants(larger, "int_gcd_make_odd.larger")
	zero_word_count := WORD_COUNT_MINIMUM
	for larger.Words[zero_word_count] == 0 {
		zero_word_count++
	}
	zero_bit_count := BIT_COUNT_MINIMUM
	residue := larger.Words[zero_word_count]
	for residue&Word(bits.CARRY_MAXIMUM) == 0 {
		zero_bit_count++
		residue >>= uint(bits.CARRY_MAXIMUM)
	}
	shifted_count := larger.Count - Word_Count(zero_word_count)
	for index := WORD_COUNT_MINIMUM; index < int(shifted_count); index++ {
		source_index := index + zero_word_count
		word := larger.Words[source_index] >> uint(zero_bit_count)
		if zero_bit_count != BIT_COUNT_MINIMUM {
			next_index := source_index + WORD_COUNT_INCREMENT
			if next_index < int(larger.Count) {
				word |= larger.Words[next_index] <<
					uint(WORD_BIT_COUNT-zero_bit_count)
			}
		}
		larger.Words[index] = word
	}
	for shifted_count > WORD_COUNT_MINIMUM {
		if larger.Words[int(shifted_count)-WORD_COUNT_INCREMENT] != 0 {
			break
		}
		shifted_count--
	}
	larger.Count = shifted_count
}
