package big

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
)

// Float_Gob_Version excludes unsupported wire versions before decoding fields.
type Float_Gob_Version byte

// Float_Gob_Version_Invariants preserves validated version identity.
func Float_Gob_Version_Invariants(value Float_Gob_Version, _ aver.Namespace) {
	aver.Always(uint8(value) == FLOAT_GOB_VERSION, "Float gob header version is supported.")
}

// Float_Parse_References limits scanning to source, cursor state, and mantissa storage.
type Float_Parse_References struct {
	// Source retains bounded input without copying during scanning.
	Source Float_Parse_Source
	// Control shares cursor updates between prefix, mantissa, and exponent scans.
	Control Float_Parse_Control
	// Parse retains accumulated words outside caller destination.
	Parse Int_Parse_Workspace_Handle
}

// Float_Parse_References_Invariants checks only state consumed by syntax scanning.
func Float_Parse_References_Invariants(value Float_Parse_References, namespace aver.Namespace) {
	Float_Parse_Source_Invariants(value.Source, namespace)
	Float_Parse_Control_Invariants(value.Control, namespace)
	Int_Parse_Workspace_Handle_Invariants(value.Parse, namespace)
}

// Rat_Operation_Integers preserves operands until normalized results can commit.
type Rat_Operation_Integers struct {
	// Result_Numerator retains cross-products outside caller numerator.
	Result_Numerator Rat_Result_Numerator
	// Result_Denominator retains cross-products outside caller denominator.
	Result_Denominator Rat_Result_Denominator
	// Common_Divisor reuses consumed cross-product scratch during reduction.
	Common_Divisor Rat_Common_Divisor
	// Left_Numerator preserves left operand during aliased writes.
	Left_Numerator Rat_Left_Numerator
	// Left_Denominator expands implicit unit before cross-products.
	Left_Denominator Rat_Left_Denominator
	// Right_Numerator preserves right operand during aliased writes.
	Right_Numerator Rat_Right_Numerator
	// Right_Denominator expands implicit unit independently of left operand.
	Right_Denominator Rat_Right_Denominator
}

// Rat_Operation_Integers_Invariants checks each independent arithmetic temporary.
func Rat_Operation_Integers_Invariants(value Rat_Operation_Integers, namespace aver.Namespace) {
	Rat_Result_Numerator_Invariants(value.Result_Numerator, namespace)
	Rat_Result_Denominator_Invariants(value.Result_Denominator, namespace)
	Rat_Common_Divisor_Invariants(value.Common_Divisor, namespace)
	Rat_Left_Numerator_Invariants(value.Left_Numerator, namespace)
	Rat_Left_Denominator_Invariants(value.Left_Denominator, namespace)
	Rat_Right_Numerator_Invariants(value.Right_Numerator, namespace)
	Rat_Right_Denominator_Invariants(value.Right_Denominator, namespace)
}

// Rat_Operation_Integers_Handle preserves cross-product metadata in caller storage.
type Rat_Operation_Integers_Handle *Rat_Operation_Integers

// Rat_Operation_Integers_Handle_Invariants composes present arithmetic temporaries.
func Rat_Operation_Integers_Handle_Invariants(
	value Rat_Operation_Integers_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Rat_Operation_Integers_Invariants(*value, namespace)
}

// Rat_Loaded_Integers distinguishes loaded operands from arbitrary reusable scratch.
type Rat_Loaded_Integers Rat_Operation_Integers

// Rat_Loaded_Integers_Invariants preserves expanded denominator bounds during products.
func Rat_Loaded_Integers_Invariants(value Rat_Loaded_Integers, namespace aver.Namespace) {
	Rat_Result_Numerator_Invariants(value.Result_Numerator, namespace)
	Rat_Result_Denominator_Invariants(value.Result_Denominator, namespace)
	Rat_Common_Divisor_Invariants(value.Common_Divisor, namespace)
	Rat_Loaded_Left_Numerator_Invariants(
		Rat_Loaded_Left_Numerator(value.Left_Numerator), namespace,
	)
	Rat_Loaded_Left_Denominator_Invariants(
		Rat_Loaded_Left_Denominator(value.Left_Denominator), namespace,
	)
	Rat_Loaded_Right_Numerator_Invariants(
		Rat_Loaded_Right_Numerator(value.Right_Numerator), namespace,
	)
	Rat_Loaded_Right_Denominator_Invariants(
		Rat_Loaded_Right_Denominator(value.Right_Denominator), namespace,
	)
}

// Rat_Loaded_Integers_Handle retains result writes in caller-owned arithmetic storage.
type Rat_Loaded_Integers_Handle *Rat_Loaded_Integers

// Rat_Loaded_Integers_Handle_Invariants checks present loaded arithmetic state.
func Rat_Loaded_Integers_Handle_Invariants(
	value Rat_Loaded_Integers_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Rat_Loaded_Integers_Invariants(*value, namespace)
}

// Rat_Loaded_Left_Numerator retains the rational component bound after loading.
type Rat_Loaded_Left_Numerator Rat_Left_Numerator

// Rat_Loaded_Left_Numerator_Invariants preserves normalized signed left operands.
func Rat_Loaded_Left_Numerator_Invariants(
	value Rat_Loaded_Left_Numerator, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, RAT_WORD_COUNT_MAXIMUM).
		Enum_Uint8(uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE)).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Loaded left numerator retains full scratch storage.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Loaded left numerator omits high zero words.")
	aver.Always(int(value.Negative) <= int(value.Count),
		"Loaded left numerator excludes negative zero.")
}

// Rat_Loaded_Right_Numerator retains the rational component bound after loading.
type Rat_Loaded_Right_Numerator Rat_Right_Numerator

// Rat_Loaded_Right_Numerator_Invariants preserves normalized signed right operands.
func Rat_Loaded_Right_Numerator_Invariants(
	value Rat_Loaded_Right_Numerator, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, RAT_WORD_COUNT_MAXIMUM).
		Enum_Uint8(uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE)).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Loaded right numerator retains full scratch storage.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Loaded right numerator omits high zero words.")
	aver.Always(int(value.Negative) <= int(value.Count),
		"Loaded right numerator excludes negative zero.")
}

// Rat_Loaded_Left_Denominator excludes implicit units expanded while loading.
type Rat_Loaded_Left_Denominator Rat_Left_Denominator

// Rat_Loaded_Left_Denominator_Invariants preserves positive bounded left factors.
func Rat_Loaded_Left_Denominator_Invariants(
	value Rat_Loaded_Left_Denominator, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_INCREMENT, RAT_WORD_COUNT_MAXIMUM).
		Ensure()
	aver.Always(value.Negative == POLARITY_NONNEGATIVE,
		"Loaded left denominator is positive.")
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Loaded left denominator retains full scratch storage.")
	aver.Always(int_high_word(value.Words[:value.Count]) != 0,
		"Loaded left denominator has no high zero words.")
}

// Rat_Loaded_Right_Denominator excludes implicit units expanded while loading.
type Rat_Loaded_Right_Denominator Rat_Right_Denominator

// Rat_Loaded_Right_Denominator_Invariants preserves positive bounded right factors.
func Rat_Loaded_Right_Denominator_Invariants(
	value Rat_Loaded_Right_Denominator, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_INCREMENT, RAT_WORD_COUNT_MAXIMUM).
		Ensure()
	aver.Always(value.Negative == POLARITY_NONNEGATIVE,
		"Loaded right denominator is positive.")
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Loaded right denominator retains full scratch storage.")
	aver.Always(int_high_word(value.Words[:value.Count]) != 0,
		"Loaded right denominator has no high zero words.")
}

// Rat_Result_Numerator retains signed cross-product sums before normalization.
type Rat_Result_Numerator Int

// Rat_Result_Numerator_Invariants preserves normalized signed scratch magnitude.
func Rat_Result_Numerator_Invariants(value Rat_Result_Numerator, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Enum_Uint8(uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE)).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Rational result numerator retains full caller storage.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Rational result numerator omits high zero words.")
	aver.Always(int(value.Negative) <= int(value.Count),
		"Rational result numerator excludes negative zero.")
}

// Rat_Result_Denominator retains signed quotient products before sign normalization.
type Rat_Result_Denominator Int

// Rat_Result_Denominator_Invariants preserves normalized signed scratch magnitude.
func Rat_Result_Denominator_Invariants(value Rat_Result_Denominator, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Enum_Uint8(uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE)).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Rational result denominator retains full caller storage.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Rational result denominator omits high zero words.")
	aver.Always(int(value.Negative) <= int(value.Count),
		"Rational result denominator excludes negative zero.")
}

// Rat_Common_Divisor retains signed cross-products before reduction reuses storage.
type Rat_Common_Divisor Int

// Rat_Common_Divisor_Invariants preserves both arithmetic and reduction phases.
func Rat_Common_Divisor_Invariants(value Rat_Common_Divisor, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Enum_Uint8(uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE)).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Rational common divisor retains full caller storage.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Rational common divisor omits high zero words.")
	aver.Always(int(value.Negative) <= int(value.Count),
		"Rational common divisor excludes negative zero.")
}

// Rat_Left_Numerator preserves signed left operand outside caller storage.
type Rat_Left_Numerator Int

// Rat_Left_Numerator_Invariants preserves normalized signed scratch magnitude.
func Rat_Left_Numerator_Invariants(value Rat_Left_Numerator, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Enum_Uint8(uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE)).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Rational left numerator retains full caller storage.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Rational left numerator omits high zero words.")
	aver.Always(int(value.Negative) <= int(value.Count),
		"Rational left numerator excludes negative zero.")
}

// Rat_Left_Denominator preserves expanded left divisor outside caller storage.
type Rat_Left_Denominator Int

// Rat_Left_Denominator_Invariants preserves normalized signed scratch magnitude.
func Rat_Left_Denominator_Invariants(value Rat_Left_Denominator, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Enum_Uint8(uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE)).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Rational left denominator retains full caller storage.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Rational left denominator omits high zero words.")
	aver.Always(int(value.Negative) <= int(value.Count),
		"Rational left denominator excludes negative zero.")
}

// Rat_Right_Numerator preserves signed right operand outside caller storage.
type Rat_Right_Numerator Int

// Rat_Right_Numerator_Invariants preserves normalized signed scratch magnitude.
func Rat_Right_Numerator_Invariants(value Rat_Right_Numerator, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Enum_Uint8(uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE)).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Rational right numerator retains full caller storage.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Rational right numerator omits high zero words.")
	aver.Always(int(value.Negative) <= int(value.Count),
		"Rational right numerator excludes negative zero.")
}

// Rat_Right_Denominator preserves expanded right divisor outside caller storage.
type Rat_Right_Denominator Int

// Rat_Right_Denominator_Invariants preserves normalized signed scratch magnitude.
func Rat_Right_Denominator_Invariants(value Rat_Right_Denominator, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Enum_Uint8(uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE)).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Rational right denominator retains full caller storage.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Rational right denominator omits high zero words.")
	aver.Always(int(value.Negative) <= int(value.Count),
		"Rational right denominator excludes negative zero.")
}

// Float_Text_Values preserves source and rounded values outside caller float.
type Float_Text_Values struct {
	// Source preserves caller value through decimal expansion.
	Source Float_Text_Source
	// Rounded keeps hexadecimal precision changes outside source.
	Rounded Float_Text_Rounded
}

// Float_Text_Values_Invariants fixes both independent rounding slots.
func Float_Text_Values_Invariants(value Float_Text_Values, namespace aver.Namespace) {
	Float_Text_Source_Invariants(value.Source, namespace)
	Float_Text_Rounded_Invariants(value.Rounded, namespace)
}

// Float_Text_Source retains a copied float independently of mutable text work.
type Float_Text_Source Float

// Float_Text_Source_Invariants preserves float metadata across conversion phases.
func Float_Text_Source_Invariants(value Float_Text_Source, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Precision), FLOAT_PRECISION_MINIMUM, FLOAT_PRECISION_MAXIMUM).
		Range_Uint8(uint8(value.Mode), uint8(ROUND_TO_NEAREST_EVEN),
			uint8(ROUND_TO_POSITIVE_INFINITY)).
		Enum_3_Int8(int8(value.Accuracy), int8(ACCURACY_BELOW), int8(ACCURACY_EXACT),
			int8(ACCURACY_ABOVE)).
		Enum_3_Uint8(uint8(value.Form), uint8(FLOAT_FORM_ZERO), uint8(FLOAT_FORM_FINITE),
			uint8(FLOAT_FORM_INFINITY)).
		Enum_Uint8(uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE)).
		Range_Int(int(value.Exponent), FLOAT_EXPONENT_MINIMUM, FLOAT_EXPONENT_MAXIMUM).
		Ensure()
	Float_Text_Source_Mantissa_Invariants(Float_Text_Source_Mantissa(value.Mantissa), namespace)
}

// Float_Text_Source_Mantissa keeps source normalization separate from rounded output.
type Float_Text_Source_Mantissa Float_Mantissa

// Float_Text_Source_Mantissa_Invariants retains normalized magnitude in full caller storage.
func Float_Text_Source_Mantissa_Invariants(
	value Float_Text_Source_Mantissa, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Float text source mantissa has complete scratch storage.")
	aver.Always(value.Negative == POLARITY_NONNEGATIVE,
		"Float text source mantissa excludes sign payload.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Float text source mantissa omits high zero words.")
}

// Float_Text_Rounded retains hexadecimal rounding outside source.
type Float_Text_Rounded Float

// Float_Text_Rounded_Invariants preserves float metadata across precision changes.
func Float_Text_Rounded_Invariants(value Float_Text_Rounded, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Precision), FLOAT_PRECISION_MINIMUM, FLOAT_PRECISION_MAXIMUM).
		Range_Uint8(uint8(value.Mode), uint8(ROUND_TO_NEAREST_EVEN),
			uint8(ROUND_TO_POSITIVE_INFINITY)).
		Enum_3_Int8(int8(value.Accuracy), int8(ACCURACY_BELOW), int8(ACCURACY_EXACT),
			int8(ACCURACY_ABOVE)).
		Enum_3_Uint8(uint8(value.Form), uint8(FLOAT_FORM_ZERO), uint8(FLOAT_FORM_FINITE),
			uint8(FLOAT_FORM_INFINITY)).
		Enum_Uint8(uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE)).
		Range_Int(int(value.Exponent), FLOAT_EXPONENT_MINIMUM, FLOAT_EXPONENT_MAXIMUM).
		Ensure()
	Float_Text_Rounded_Mantissa_Invariants(
		Float_Text_Rounded_Mantissa(value.Mantissa), namespace,
	)
}

// Float_Text_Rounded_Mantissa keeps rounded normalization separate from source.
type Float_Text_Rounded_Mantissa Float_Mantissa

// Float_Text_Rounded_Mantissa_Invariants retains normalized magnitude in full caller storage.
func Float_Text_Rounded_Mantissa_Invariants(
	value Float_Text_Rounded_Mantissa, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Float text rounded mantissa has complete scratch storage.")
	aver.Always(value.Negative == POLARITY_NONNEGATIVE,
		"Float text rounded mantissa excludes sign payload.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Float text rounded mantissa omits high zero words.")
}

// Float_Text_Decimals keeps exact value separate from shortest-rounding bounds.
type Float_Text_Decimals struct {
	// Value retains exact digits while midpoint bounds are rounded.
	Value Float_Text_Decimal
	// Lower preserves lower midpoint independently of exact value.
	Lower Float_Text_Decimal_Lower
	// Upper preserves upper midpoint independently of lower midpoint.
	Upper Float_Text_Decimal_Upper
}

// Float_Text_Decimals_Invariants fixes exact value and both boundary slots.
func Float_Text_Decimals_Invariants(value Float_Text_Decimals, namespace aver.Namespace) {
	Float_Text_Decimal_Invariants(value.Value, namespace)
	Float_Text_Decimal_Lower_Invariants(value.Lower, namespace)
	Float_Text_Decimal_Upper_Invariants(value.Upper, namespace)
}

// Float_Text_Decimal_Lower retains lower midpoint in independent digit storage.
type Float_Text_Decimal_Lower Float_Text_Decimal

// Float_Text_Decimal_Lower_Invariants preserves complete storage and lower point state.
func Float_Text_Decimal_Lower_Invariants(
	value Float_Text_Decimal_Lower, namespace aver.Namespace,
) {
	aver.Always(len(value.Digits) == FLOAT_DECIMAL_DIGIT_COUNT_MAXIMUM,
		"Lower decimal midpoint retains complete digit storage.")
	Float_Text_Decimal_Lower_Control_Invariants(
		Float_Text_Decimal_Lower_Control(value.Control), namespace,
	)
}

// Float_Text_Decimal_Lower_Control keeps lower midpoint state distinct from exact value.
type Float_Text_Decimal_Lower_Control Float_Text_Decimal_Control

// Float_Text_Decimal_Lower_Control_Invariants bounds lower midpoint count and point.
func Float_Text_Decimal_Lower_Control_Invariants(
	value Float_Text_Decimal_Lower_Control, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, FLOAT_DECIMAL_DIGIT_COUNT_MAXIMUM).
		Range_Int(int(value.Exponent), FLOAT_TEXT_DECIMAL_EXPONENT_MINIMUM,
			FLOAT_TEXT_DECIMAL_EXPONENT_MAXIMUM).
		Ensure()
}

// Float_Text_Decimal_Upper retains upper midpoint in independent digit storage.
type Float_Text_Decimal_Upper Float_Text_Decimal

// Float_Text_Decimal_Upper_Invariants preserves complete storage and upper point state.
func Float_Text_Decimal_Upper_Invariants(
	value Float_Text_Decimal_Upper, namespace aver.Namespace,
) {
	aver.Always(len(value.Digits) == FLOAT_DECIMAL_DIGIT_COUNT_MAXIMUM,
		"Upper decimal midpoint retains complete digit storage.")
	Float_Text_Decimal_Upper_Control_Invariants(
		Float_Text_Decimal_Upper_Control(value.Control), namespace,
	)
}

// Float_Text_Decimal_Upper_Control keeps upper midpoint state distinct from lower midpoint.
type Float_Text_Decimal_Upper_Control Float_Text_Decimal_Control

// Float_Text_Decimal_Upper_Control_Invariants bounds upper midpoint count and point.
func Float_Text_Decimal_Upper_Control_Invariants(
	value Float_Text_Decimal_Upper_Control, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, FLOAT_DECIMAL_DIGIT_COUNT_MAXIMUM).
		Range_Int(int(value.Exponent), FLOAT_TEXT_DECIMAL_EXPONENT_MINIMUM,
			FLOAT_TEXT_DECIMAL_EXPONENT_MAXIMUM).
		Ensure()
}

// Float_Text_Decimal_Handle permits one selected decimal to retain mutations.
type Float_Text_Decimal_Handle *Float_Text_Decimal

// Float_Text_Decimal_Handle_Invariants checks selected decimal without copying its digits.
func Float_Text_Decimal_Handle_Invariants(
	value Float_Text_Decimal_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Text_Decimal_Invariants(*value, namespace)
}

// Float_Text_Decimals_Handle preserves all midpoint metadata in caller storage.
type Float_Text_Decimals_Handle *Float_Text_Decimals

// Float_Text_Decimals_Handle_Invariants composes present decimal storage.
func Float_Text_Decimals_Handle_Invariants(
	value Float_Text_Decimals_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Text_Decimals_Invariants(*value, namespace)
}

// Float_Text_Digits includes derived shortest-format precision beyond explicit requests.
type Float_Text_Digits int

// Float_Text_Digits_Invariants bounds work by complete output capacity.
func Float_Text_Digits_Invariants(value Float_Text_Digits, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), FLOAT_TEXT_PRECISION_MINIMUM, FLOAT_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Float_Text_Digits_Handle preserves precision updates without unrelated formatting state.
type Float_Text_Digits_Handle *Float_Text_Digits

// Float_Text_Digits_Handle_Invariants composes present precision storage.
func Float_Text_Digits_Handle_Invariants(value Float_Text_Digits_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Float_Text_Digits_Invariants(*value, namespace)
}

// Float_Text_Exponent_Digits preserves padding independently of exponent magnitude.
type Float_Text_Exponent_Digits int

// Float_Text_Exponent_Digits_Invariants admits unset, binary, and decimal padding.
func Float_Text_Exponent_Digits_Invariants(
	value Float_Text_Exponent_Digits, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Int(int(value), WORD_COUNT_MINIMUM, WORD_COUNT_INCREMENT, BASE_BINARY).
		Ensure()
}

// Float_Text_Decimal_Shift limits exact division to one machine-sized step.
type Float_Text_Decimal_Shift int

// Float_Text_Decimal_Shift_Invariants includes untouched scratch before division.
func Float_Text_Decimal_Shift_Invariants(
	value Float_Text_Decimal_Shift, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), WORD_COUNT_MINIMUM, FLOAT_DECIMAL_SHIFT_MAXIMUM).
		Ensure()
}

// FLOAT_TEXT_BINARY_SHIFT_MINIMUM includes one lower-midpoint guard bit.
const FLOAT_TEXT_BINARY_SHIFT_MINIMUM = -FLOAT_DECIMAL_RIGHT_SHIFT_MAXIMUM

// Float_Text_Binary_Shift preserves midpoint scale independently of decimal shifts.
type Float_Text_Binary_Shift int

// Float_Text_Binary_Shift_Invariants bounds precision-adjusted exponent scale.
func Float_Text_Binary_Shift_Invariants(
	value Float_Text_Binary_Shift, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), FLOAT_TEXT_BINARY_SHIFT_MINIMUM, FLOAT_EXPONENT_MAXIMUM).
		Ensure()
}

// Float_Text_Retained_Digits permits rounding before the first significant digit.
type Float_Text_Retained_Digits int

// Float_Text_Retained_Digits_Invariants bounds the signed rounding boundary.
func Float_Text_Retained_Digits_Invariants(
	value Float_Text_Retained_Digits, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), FLOAT_TEXT_BINARY_SHIFT_MINIMUM, FLOAT_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Float_Text_Control preserves independent formatting decisions without positional slots.
type Float_Text_Control struct {
	// Output_Count prevents later stages from overwriting prior output.
	Output_Count Float_Text_Count
	// Precision carries requested or derived decimal precision across stages.
	Precision Float_Text_Digits
	// Exponent_Digits preserves format-specific exponent padding.
	Exponent_Digits Float_Text_Exponent_Digits
	// Format preserves validated representation selection.
	Format Float_Text_Format
}

// Float_Text_Control_Invariants bounds every independent formatting decision.
func Float_Text_Control_Invariants(value Float_Text_Control, namespace aver.Namespace) {
	Float_Text_Count_Invariants(value.Output_Count, namespace)
	Float_Text_Digits_Invariants(value.Precision, namespace)
	Float_Text_Exponent_Digits_Invariants(value.Exponent_Digits, namespace)
	Float_Text_Format_Invariants(value.Format, namespace)
}

// Float_Text_Control_Handle preserves control mutations in caller-owned storage.
type Float_Text_Control_Handle *Float_Text_Control

// Float_Text_Control_Handle_Invariants composes present control storage.
func Float_Text_Control_Handle_Invariants(
	value Float_Text_Control_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Text_Control_Invariants(*value, namespace)
}

// Float_Text_References excludes integer formatting scratch from decimal rounding.
type Float_Text_References struct {
	// Source retains exact binary value while decimal bounds change.
	Source Float_Finite
	// Decimals keeps midpoint mutations visible to caller.
	Decimals Float_Text_Decimals_Handle
	// Magnitude supplies independent storage for each midpoint expansion.
	Magnitude Float_Text_Magnitude_Handle
}

// Float_Text_References_Invariants binds only storage used by midpoint rounding.
func Float_Text_References_Invariants(value Float_Text_References, namespace aver.Namespace) {
	Float_Finite_Invariants(value.Source, namespace)
	Float_Text_Decimals_Handle_Invariants(value.Decimals, namespace)
	Float_Text_Magnitude_Handle_Invariants(value.Magnitude, namespace)
}

// Float_Text_Decimal preserves digits independently of count and decimal-point state.
type Float_Text_Decimal struct {
	// Digits retains exact expansion before rounding discards low digits.
	Digits Float_Text_Decimal_Digits
	// Control preserves active count and decimal point across shifts.
	Control Float_Text_Decimal_Control
}

// Float_Text_Decimal_Invariants requires complete scratch before decimal expansion.
func Float_Text_Decimal_Invariants(value Float_Text_Decimal, namespace aver.Namespace) {
	Float_Text_Decimal_Digits_Invariants(value.Digits, namespace)
	Float_Text_Decimal_Control_Invariants(value.Control, namespace)
}

// Float_Text_Decimal_Digits bounds exact expansion without growing caller storage.
type Float_Text_Decimal_Digits []byte

// Float_Text_Decimal_Digits_Invariants retains every possible exact digit.
func Float_Text_Decimal_Digits_Invariants(value Float_Text_Decimal_Digits, _ aver.Namespace) {
	aver.Always(len(value) == FLOAT_DECIMAL_DIGIT_COUNT_MAXIMUM,
		"Float text decimal digits retain complete expansion capacity.")
}

// Float_Text_Decimal_Control retains count and decimal point outside digit storage.
type Float_Text_Decimal_Control struct {
	// Count excludes discarded trailing zeros from rounding work.
	Count Float_Text_Decimal_Count
	// Exponent preserves decimal point while shifts change digit count.
	Exponent Float_Text_Decimal_Exponent
}

// Float_Text_Decimal_Control_Invariants fixes complete decimal cursor storage.
func Float_Text_Decimal_Control_Invariants(
	value Float_Text_Decimal_Control, namespace aver.Namespace,
) {
	Float_Text_Decimal_Count_Invariants(value.Count, namespace)
	Float_Text_Decimal_Exponent_Invariants(value.Exponent, namespace)
}

// Float_Text_Decimal_Count bounds active expansion independently of binary mantissa width.
type Float_Text_Decimal_Count int

// Float_Text_Decimal_Count_Invariants includes empty decimal before expansion.
func Float_Text_Decimal_Count_Invariants(value Float_Text_Decimal_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), WORD_COUNT_MINIMUM, FLOAT_DECIMAL_DIGIT_COUNT_MAXIMUM).
		Ensure()
}

// FLOAT_TEXT_DECIMAL_EXPONENT_MINIMUM includes every bounded right shift.
const FLOAT_TEXT_DECIMAL_EXPONENT_MINIMUM = -FLOAT_DECIMAL_RIGHT_SHIFT_MAXIMUM

// FLOAT_TEXT_DECIMAL_EXPONENT_MAXIMUM leaves room for rounding carry.
const FLOAT_TEXT_DECIMAL_EXPONENT_MAXIMUM = FLOAT_DECIMAL_INTEGER_DIGIT_COUNT_MAXIMUM +
	WORD_COUNT_INCREMENT

// Float_Text_Decimal_Exponent locates decimal point independently of retained digits.
type Float_Text_Decimal_Exponent int

// Float_Text_Decimal_Exponent_Invariants includes every shift and one rounding carry.
func Float_Text_Decimal_Exponent_Invariants(
	value Float_Text_Decimal_Exponent, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), FLOAT_TEXT_DECIMAL_EXPONENT_MINIMUM,
			FLOAT_TEXT_DECIMAL_EXPONENT_MAXIMUM).
		Ensure()
}

// Float_Division_Word_Control preserves scalar state across quotient phases.
type Float_Division_Word_Control struct {
	// Divisor_Count selects short division only after width validation.
	Divisor_Count Float_Division_Divisor_Count
	// Dividend_Count bounds scaled numerator traversal.
	Dividend_Count Float_Division_Dividend_Count
	// Shift becomes result precision after scaling consumes it.
	Shift Float_Division_Scale
	// Offset becomes prior destination width after quotient construction consumes it.
	Offset Float_Division_Offset
	// Quotient_Count excludes leading zeros before destination commit.
	Quotient_Count Float_Division_Quotient_Count
	// Increment retains rounding decision until mantissa commit.
	Increment Float_Division_Increment
}

// Float_Division_Word_Control_Invariants keeps each phase inside bounded scratch.
func Float_Division_Word_Control_Invariants(
	value Float_Division_Word_Control, namespace aver.Namespace,
) {
	Float_Division_Divisor_Count_Invariants(value.Divisor_Count, namespace)
	Float_Division_Dividend_Count_Invariants(value.Dividend_Count, namespace)
	Float_Division_Scale_Invariants(value.Shift, namespace)
	Float_Division_Offset_Invariants(value.Offset, namespace)
	Float_Division_Quotient_Count_Invariants(value.Quotient_Count, namespace)
	Float_Division_Increment_Invariants(value.Increment, namespace)
}

// Float_Division_Word_Control_Handle shares scalar updates between division phases.
type Float_Division_Word_Control_Handle *Float_Division_Word_Control

// Float_Division_Word_Control_Handle_Invariants requires caller-owned control state.
func Float_Division_Word_Control_Handle_Invariants(
	value Float_Division_Word_Control_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Division_Word_Control_Invariants(*value, namespace)
}

// Float_Division_Divisor_Count admits zero before short-divisor selection.
type Float_Division_Divisor_Count int

// Float_Division_Divisor_Count_Invariants excludes multiword fallback widths.
func Float_Division_Divisor_Count_Invariants(
	value Float_Division_Divisor_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_3_Int(int(value), WORD_COUNT_MINIMUM, WORD_COUNT_INCREMENT, BASE_BINARY).
		Ensure()
}

// Float_Division_Short_Count excludes failed selection before quotient generation.
type Float_Division_Short_Count int

// Float_Division_Short_Count_Invariants permits only supported divisor widths.
func Float_Division_Short_Count_Invariants(
	value Float_Division_Short_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Enum_Int(int(value), WORD_COUNT_INCREMENT, BASE_BINARY).
		Ensure()
}

// Float_Division_Dividend_Count retains scaled numerator width before normalization.
type Float_Division_Dividend_Count int

// Float_Division_Dividend_Count_Invariants reserves carry outside active numerator.
func Float_Division_Dividend_Count_Invariants(
	value Float_Division_Dividend_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Ensure()
}

// Float_Division_Scale retains guarded shift, then final precision.
type Float_Division_Scale int

// Float_Division_Scale_Invariants admits initial zero and every representable bit scale.
func Float_Division_Scale_Invariants(value Float_Division_Scale, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), FLOAT_PRECISION_MINIMUM, FLOAT_PRECISION_MAXIMUM).
		Ensure()
}

// Float_Division_Offset retains quotient position, then prior destination width.
type Float_Division_Offset int

// Float_Division_Offset_Invariants admits complete prior width at commit.
func Float_Division_Offset_Invariants(value Float_Division_Offset, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Ensure()
}

// Float_Division_Quotient_Count retains normalized output width until commit.
type Float_Division_Quotient_Count int

// Float_Division_Quotient_Count_Invariants admits initial zero before rounding.
func Float_Division_Quotient_Count_Invariants(
	value Float_Division_Quotient_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Ensure()
}

// Float_Division_Increment retains one rounding carry without widening count state.
type Float_Division_Increment int

// Float_Division_Increment_Invariants admits exactly both carry decisions.
func Float_Division_Increment_Invariants(value Float_Division_Increment, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Int(int(value), WORD_COUNT_MINIMUM, WORD_COUNT_INCREMENT).
		Ensure()
}

// Float_Rat_Values owns intermediate floats outside caller values.
type Float_Rat_Values struct {
	// Numerator retains exact signed magnitude before division.
	Numerator Float_Rat_Numerator
	// Denominator retains divisor independently of numerator rounding.
	Denominator Float_Rat_Denominator
	// Result preserves caller destination until conversion succeeds.
	Result Float_Rat_Result
}

// Float_Rat_Values_Invariants fixes complete rational conversion storage.
func Float_Rat_Values_Invariants(value Float_Rat_Values, namespace aver.Namespace) {
	Float_Rat_Numerator_Invariants(value.Numerator, namespace)
	Float_Rat_Denominator_Invariants(value.Denominator, namespace)
	Float_Rat_Result_Invariants(value.Result, namespace)
}

// Float_Rat_Numerator retains mutable numerator scratch across conversions.
type Float_Rat_Numerator Float

// Float_Rat_Numerator_Invariants checks capacity before operand setup validates float state.
func Float_Rat_Numerator_Invariants(value Float_Rat_Numerator, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Precision), FLOAT_PRECISION_MINIMUM, FLOAT_PRECISION_MAXIMUM).
		Range_Uint8(uint8(value.Mode), uint8(ROUND_TO_NEAREST_EVEN),
			uint8(ROUND_TO_POSITIVE_INFINITY)).
		Enum_3_Int8(int8(value.Accuracy), int8(ACCURACY_BELOW), int8(ACCURACY_EXACT),
			int8(ACCURACY_ABOVE)).
		Enum_3_Uint8(uint8(value.Form), uint8(FLOAT_FORM_ZERO), uint8(FLOAT_FORM_FINITE),
			uint8(FLOAT_FORM_INFINITY)).
		Enum_Uint8(uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE)).
		Range_Int(int(value.Exponent), FLOAT_EXPONENT_MINIMUM, FLOAT_EXPONENT_MAXIMUM).
		Ensure()
	Float_Rat_Numerator_Mantissa_Invariants(
		Float_Rat_Numerator_Mantissa(value.Mantissa), namespace,
	)
}

// Float_Rat_Denominator retains mutable divisor scratch across conversions.
type Float_Rat_Denominator Float

// Float_Rat_Denominator_Invariants checks capacity before operand setup validates float state.
func Float_Rat_Denominator_Invariants(value Float_Rat_Denominator, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Precision), FLOAT_PRECISION_MINIMUM, FLOAT_PRECISION_MAXIMUM).
		Range_Uint8(uint8(value.Mode), uint8(ROUND_TO_NEAREST_EVEN),
			uint8(ROUND_TO_POSITIVE_INFINITY)).
		Enum_3_Int8(int8(value.Accuracy), int8(ACCURACY_BELOW), int8(ACCURACY_EXACT),
			int8(ACCURACY_ABOVE)).
		Enum_3_Uint8(uint8(value.Form), uint8(FLOAT_FORM_ZERO), uint8(FLOAT_FORM_FINITE),
			uint8(FLOAT_FORM_INFINITY)).
		Enum_Uint8(uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE)).
		Range_Int(int(value.Exponent), FLOAT_EXPONENT_MINIMUM, FLOAT_EXPONENT_MAXIMUM).
		Ensure()
	Float_Rat_Denominator_Mantissa_Invariants(
		Float_Rat_Denominator_Mantissa(value.Mantissa), namespace,
	)
}

// Float_Rat_Result retains transactional output outside caller destination.
type Float_Rat_Result Float

// Float_Rat_Result_Invariants checks storage before conversion replaces result metadata.
func Float_Rat_Result_Invariants(value Float_Rat_Result, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Precision), FLOAT_PRECISION_MINIMUM, FLOAT_PRECISION_MAXIMUM).
		Range_Uint8(uint8(value.Mode), uint8(ROUND_TO_NEAREST_EVEN),
			uint8(ROUND_TO_POSITIVE_INFINITY)).
		Enum_3_Int8(int8(value.Accuracy), int8(ACCURACY_BELOW), int8(ACCURACY_EXACT),
			int8(ACCURACY_ABOVE)).
		Enum_3_Uint8(uint8(value.Form), uint8(FLOAT_FORM_ZERO), uint8(FLOAT_FORM_FINITE),
			uint8(FLOAT_FORM_INFINITY)).
		Enum_Uint8(uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE)).
		Range_Int(int(value.Exponent), FLOAT_EXPONENT_MINIMUM, FLOAT_EXPONENT_MAXIMUM).
		Ensure()
	Float_Rat_Result_Mantissa_Invariants(Float_Rat_Result_Mantissa(value.Mantissa), namespace)
}

// Float_Rat_Numerator_Mantissa keeps numerator normalization distinct from divisor state.
type Float_Rat_Numerator_Mantissa Float_Mantissa

// Float_Rat_Numerator_Mantissa_Invariants bounds active magnitude in complete caller storage.
func Float_Rat_Numerator_Mantissa_Invariants(
	value Float_Rat_Numerator_Mantissa, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Rational numerator mantissa has complete scratch storage.")
	aver.Always(value.Negative == POLARITY_NONNEGATIVE,
		"Rational numerator mantissa excludes sign payload.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Rational numerator mantissa omits high zero words.")
}

// Float_Rat_Denominator_Mantissa keeps divisor normalization distinct from numerator state.
type Float_Rat_Denominator_Mantissa Float_Mantissa

// Float_Rat_Denominator_Mantissa_Invariants bounds active magnitude in complete caller storage.
func Float_Rat_Denominator_Mantissa_Invariants(
	value Float_Rat_Denominator_Mantissa, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Rational denominator mantissa has complete scratch storage.")
	aver.Always(value.Negative == POLARITY_NONNEGATIVE,
		"Rational denominator mantissa excludes sign payload.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Rational denominator mantissa omits high zero words.")
}

// Float_Rat_Result_Mantissa keeps result normalization distinct from either operand.
type Float_Rat_Result_Mantissa Float_Mantissa

// Float_Rat_Result_Mantissa_Invariants bounds active magnitude in complete caller storage.
func Float_Rat_Result_Mantissa_Invariants(
	value Float_Rat_Result_Mantissa, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Rational result mantissa has complete scratch storage.")
	aver.Always(value.Negative == POLARITY_NONNEGATIVE,
		"Rational result mantissa excludes sign payload.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Rational result mantissa omits high zero words.")
}

// Rat_Float_32_Integers owns scaled operands and quotient-remainder output.
type Rat_Float_32_Integers struct {
	// Numerator preserves caller magnitude during scaling.
	Numerator Int
	// Denominator preserves caller divisor during scaling.
	Denominator Rat_Float_32_Denominator
	// Quotient retains guard bits before rounding.
	Quotient Rat_Float_32_Quotient
	// Remainder distinguishes exact values from discarded fractions.
	Remainder Rat_Float_32_Remainder
}

// Rat_Float_32_Integers_Invariants fixes complete binary32 conversion capacity.
func Rat_Float_32_Integers_Invariants(
	value Rat_Float_32_Integers, namespace aver.Namespace,
) {
	Int_Invariants(value.Numerator, namespace)
	Rat_Float_32_Denominator_Invariants(value.Denominator, namespace)
	Rat_Float_32_Quotient_Invariants(value.Quotient, namespace)
	Rat_Float_32_Remainder_Invariants(value.Remainder, namespace)
	aver.Always(len(value.Numerator.Words) == WORD_COUNT_MAXIMUM,
		"Rational binary32 numerator has complete scaling storage.")
}

// Rat_Float_32_Denominator preserves scaled divisor outside caller rational.
type Rat_Float_32_Denominator Int

// Rat_Float_32_Denominator_Invariants permits initial zero before divisor setup.
func Rat_Float_32_Denominator_Invariants(
	value Rat_Float_32_Denominator, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Binary32 scaled denominator retains complete caller storage.")
	aver.Always(value.Negative == POLARITY_NONNEGATIVE,
		"Binary32 scaled denominator excludes negative divisors.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Binary32 scaled denominator omits high zero words.")
}

// Rat_Float_32_Quotient retains guard bits before final rounding.
type Rat_Float_32_Quotient Int

// Rat_Float_32_Quotient_Invariants preserves normalized nonnegative division output.
func Rat_Float_32_Quotient_Invariants(
	value Rat_Float_32_Quotient, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Binary32 rounding quotient retains complete caller storage.")
	aver.Always(value.Negative == POLARITY_NONNEGATIVE,
		"Binary32 rounding quotient leaves sign on caller rational.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Binary32 rounding quotient omits high zero words.")
}

// Rat_Float_32_Remainder distinguishes exact division from discarded fraction.
type Rat_Float_32_Remainder Int

// Rat_Float_32_Remainder_Invariants preserves normalized nonnegative division residue.
func Rat_Float_32_Remainder_Invariants(
	value Rat_Float_32_Remainder, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Binary32 discarded remainder retains complete caller storage.")
	aver.Always(value.Negative == POLARITY_NONNEGATIVE,
		"Binary32 discarded remainder excludes negative residues.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Binary32 discarded remainder omits high zero words.")
}

// Rat_Float_64_Denominator preserves scaled divisor outside caller rational.
type Rat_Float_64_Denominator Int

// Rat_Float_64_Denominator_Invariants permits initial zero before divisor setup.
func Rat_Float_64_Denominator_Invariants(
	value Rat_Float_64_Denominator, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Binary64 scaled denominator retains complete caller storage.")
	aver.Always(value.Negative == POLARITY_NONNEGATIVE,
		"Binary64 scaled denominator excludes negative divisors.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Binary64 scaled denominator omits high zero words.")
}

// Rat_Float_64_Quotient retains guard bits before final rounding.
type Rat_Float_64_Quotient Int

// Rat_Float_64_Quotient_Invariants preserves normalized nonnegative division output.
func Rat_Float_64_Quotient_Invariants(
	value Rat_Float_64_Quotient, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Binary64 rounding quotient retains complete caller storage.")
	aver.Always(value.Negative == POLARITY_NONNEGATIVE,
		"Binary64 rounding quotient leaves sign on caller rational.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Binary64 rounding quotient omits high zero words.")
}

// Rat_Float_64_Remainder distinguishes exact division from discarded fraction.
type Rat_Float_64_Remainder Int

// Rat_Float_64_Remainder_Invariants preserves normalized nonnegative division residue.
func Rat_Float_64_Remainder_Invariants(
	value Rat_Float_64_Remainder, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Binary64 discarded remainder retains complete caller storage.")
	aver.Always(value.Negative == POLARITY_NONNEGATIVE,
		"Binary64 discarded remainder excludes negative residues.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Binary64 discarded remainder omits high zero words.")
}

// Exponent_Factor preserves squared base independently of accumulated result.
type Exponent_Factor Int

// Exponent_Factor_Invariants preserves full normalized signed integer domain.
func Exponent_Factor_Invariants(value Exponent_Factor, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Enum_Uint8(
			uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE),
		).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Exponent factor reserves complete base and squared-product storage.")
	aver.Always(int(value.Count) <= len(value.Words),
		"An exponent factor retains active magnitude in caller storage.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"An exponent factor omits high zero words.")
	aver.Always(int(value.Negative) <= int(value.Count),
		"An exponent factor gives zero no negative twin.")
}

// Product_Factor stays nonnegative because range sign belongs to accumulator.
type Product_Factor Int

// Product_Factor_Invariants bounds scalar factors and reduced binomial denominators.
func Product_Factor_Invariants(value Product_Factor, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_INCREMENT).
		Range_Int(len(value.Words), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Ensure()
	aver.Always(value.Negative == POLARITY_NONNEGATIVE,
		"A product factor leaves sign on accumulator.")
	aver.Always(int(value.Count) <= len(value.Words),
		"A product factor retains its active word in caller storage.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"A product factor omits high zero words.")
}

// Int_Division_Trial_High retains normalized numerator carry without sharing divisor state.
type Int_Division_Trial_High Word

// Int_Division_Trial_High_Invariants preserves full carry-word precision.
func Int_Division_Trial_High_Invariants(value Int_Division_Trial_High, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Int_Division_Trial_Next retains numerator precision below carry word.
type Int_Division_Trial_Next Word

// Int_Division_Trial_Next_Invariants preserves every numerator bit.
func Int_Division_Trial_Next_Invariants(value Int_Division_Trial_Next, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Int_Division_Trial_Third retains low comparison bits for estimate correction.
type Int_Division_Trial_Third Word

// Int_Division_Trial_Third_Invariants preserves every correction bit.
func Int_Division_Trial_Third_Invariants(
	value Int_Division_Trial_Third, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Empty magnitude represents zero without requiring backing storage for observation.
func int_high_word(words Words) (word Word) {
	defer func() { Word_Invariants(word, "int_high_word.word") }()
	Words_Invariants(words, "int_high_word.words")
	if len(words) == WORD_COUNT_MINIMUM {
		return 0
	}
	return words[len(words)-WORD_COUNT_INCREMENT]
}

// Both discarded bits determine exactness; increment and sign determine error direction.
func float_rounding_accuracy(
	rounding_bit Bit_Value, sticky_bit Bit_Value, increment Boolean, negative Polarity,
) (accuracy Accuracy) {
	defer func() { Accuracy_Invariants(accuracy, "float_rounding_accuracy.accuracy") }()
	Bit_Value_Invariants(rounding_bit, "float_rounding_accuracy.rounding_bit")
	Bit_Value_Invariants(sticky_bit, "float_rounding_accuracy.sticky_bit")
	Boolean_Invariants(increment, "float_rounding_accuracy.increment")
	Polarity_Invariants(negative, "float_rounding_accuracy.negative")
	if rounding_bit != BIT_CLEAR {
		return Accuracy(float_accuracy(increment, negative))
	}
	if sticky_bit != BIT_CLEAR {
		return Accuracy(float_accuracy(increment, negative))
	}
	return ACCURACY_EXACT
}

// Int_Division_Normalized_Word requires leading bit for quotient estimation.
type Int_Division_Normalized_Word Word

// Int_Division_Normalized_Word_Invariants excludes unnormalized divisor prefixes.
func Int_Division_Normalized_Word_Invariants(
	value Int_Division_Normalized_Word, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(
			uint64(value), INT_DIVISION_NORMALIZED_WORD_MINIMUM, bits.WORD_64_MAXIMUM,
		).
		Ensure()
}

// Previous outputs and incoming dividend can differ in width; clear their union before reuse.
func int_division_clear(workspace Int_Division_Workspace_Handle, count Word_Count) {
	Int_Division_Workspace_Handle_Invariants(workspace, "int_division_clear.workspace")
	Word_Count_Invariants(count, "int_division_clear.count")
	clear_count := max(int(count), int(workspace.Quotient_Count),
		int(workspace.Remainder_Count))
	for index := WORD_COUNT_MINIMUM; index < clear_count; index++ {
		workspace.Quotient[index] = 0
		workspace.Remainder[index] = 0
	}
	workspace.Quotient_Count = Quotient_Count(WORD_COUNT_MINIMUM)
	workspace.Remainder_Count = Remainder_Count(WORD_COUNT_MINIMUM)
}

// Float_Text_Output preserves transactional formatting without implicit buffer growth.
type Float_Text_Output []byte

// Float_Text_Output_Invariants requires complete output capacity before formatting.
func Float_Text_Output_Invariants(value Float_Text_Output, _ aver.Namespace) {
	aver.Always(len(value) == FLOAT_TEXT_SIZE_MAXIMUM,
		"Float text owns complete transactional output.")
}

// Float_Parse_Source isolates parsing from caller input mutation.
type Float_Parse_Source []byte

// Float_Parse_Source_Invariants requires complete bounded source storage.
func Float_Parse_Source_Invariants(value Float_Parse_Source, _ aver.Namespace) {
	aver.Always(len(value) == FLOAT_PARSE_TEXT_SIZE_MAXIMUM,
		"Float parse owns complete validated source storage.")
}

// Float_Parse_Control keeps scanner state mutable across parsing helpers.
type Float_Parse_Control []int64

// Float_Parse_Control_Invariants retains every scanner slot throughout parsing.
func Float_Parse_Control_Invariants(value Float_Parse_Control, _ aver.Namespace) {
	aver.Always(len(value) == FLOAT_PARSE_CONTROL_COUNT,
		"Float parse owns complete structural scanner state.")
}

func rat_add_single_words(
	destination Rat_Handle, left Rat_Handle, right Rat_Handle,
) (matched Boolean) {
	defer func() { Boolean_Invariants(matched, "rat_add_single_words.matched") }()
	Rat_Handle_Invariants(destination, "rat_add_single_words.destination")
	Rat_Handle_Invariants(left, "rat_add_single_words.left")
	Rat_Handle_Invariants(right, "rat_add_single_words.right")
	left_numerator, right_numerator :=
		&left.Integers.Numerator, &right.Integers.Numerator
	left_denominator, right_denominator :=
		(*Int)(&left.Integers.Denominator), (*Int)(&right.Integers.Denominator)
	switch {
	case left_numerator.Count > Word_Count(WORD_COUNT_INCREMENT):
		return false
	case right_numerator.Count > Word_Count(WORD_COUNT_INCREMENT):
		return false
	case left_denominator.Count > Word_Count(WORD_COUNT_INCREMENT):
		return false
	case right_denominator.Count > Word_Count(WORD_COUNT_INCREMENT):
		return false
	}
	left_numerator_word := Word(0)
	if left_numerator.Count != Word_Count(WORD_COUNT_MINIMUM) {
		left_numerator_word = left_numerator.Words[WORD_COUNT_MINIMUM]
	}
	right_numerator_word := Word(0)
	if right_numerator.Count != Word_Count(WORD_COUNT_MINIMUM) {
		right_numerator_word = right_numerator.Words[WORD_COUNT_MINIMUM]
	}
	left_denominator_word := Word(bits.CARRY_MAXIMUM)
	if left_denominator.Count != Word_Count(WORD_COUNT_MINIMUM) {
		left_denominator_word = left_denominator.Words[WORD_COUNT_MINIMUM]
	}
	right_denominator_word := Word(bits.CARRY_MAXIMUM)
	if right_denominator.Count != Word_Count(WORD_COUNT_MINIMUM) {
		right_denominator_word = right_denominator.Words[WORD_COUNT_MINIMUM]
	}
	switch {
	case left_numerator_word > Word(bits.WORD_32_MAXIMUM):
		return false
	case right_numerator_word > Word(bits.WORD_32_MAXIMUM):
		return false
	case left_denominator_word > Word(bits.WORD_32_MAXIMUM):
		return false
	case right_denominator_word > Word(bits.WORD_32_MAXIMUM):
		return false
	}
	left_scaled := left_numerator_word * right_denominator_word
	right_scaled := right_numerator_word * left_denominator_word
	numerator, negative, valid := rat_add_signed_words(
		Half_Word_Product(left_scaled), left_numerator.Negative,
		Half_Word_Product(right_scaled), right_numerator.Negative,
	)
	if !valid {
		return false
	}
	denominator := left_denominator_word * right_denominator_word
	common_divisor := denominator
	for remainder := numerator; remainder != 0; {
		common_divisor, remainder = remainder, common_divisor%remainder
	}
	numerator /= common_divisor
	denominator /= common_divisor
	int_set_word(&destination.Integers.Numerator, numerator, negative)
	denominator_destination := (*Int)(&destination.Integers.Denominator)
	if denominator == Word(bits.CARRY_MAXIMUM) {
		int_zero(denominator_destination, denominator_destination.Count)
	} else {
		int_set_word(denominator_destination, denominator, POLARITY_NONNEGATIVE)
	}
	return true
}

// Two half words form the widest native product accepted by the rational fast path.
const HALF_WORD_PRODUCT_MAXIMUM = Word(bits.WORD_32_MAXIMUM) * Word(bits.WORD_32_MAXIMUM)

// Half_Word_Product is one native product of two bounded half words.
type Half_Word_Product Word

// Half_Word_Product_Invariants narrows a machine word to its reachable product interval.
func Half_Word_Product_Invariants(
	value Half_Word_Product, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(
			uint64(value), uint64(bits.CARRY_MINIMUM),
			uint64(HALF_WORD_PRODUCT_MAXIMUM),
		).
		Ensure()
}

func rat_add_signed_words(
	left Half_Word_Product, left_negative Polarity,
	right Half_Word_Product, right_negative Polarity,
) (sum Word, negative Polarity, valid Boolean) {
	defer func() {
		Word_Invariants(sum, "rat_add_signed_words.sum")
		Polarity_Invariants(negative, "rat_add_signed_words.negative")
		Boolean_Invariants(valid, "rat_add_signed_words.valid")
	}()
	Half_Word_Product_Invariants(left, "rat_add_signed_words.left")
	Polarity_Invariants(left_negative, "rat_add_signed_words.left_negative")
	Half_Word_Product_Invariants(right, "rat_add_signed_words.right")
	Polarity_Invariants(right_negative, "rat_add_signed_words.right_negative")
	left_word := Word(left)
	right_word := Word(right)
	if left_negative == right_negative {
		sum = left_word + right_word
		if sum < left_word {
			return 0, POLARITY_NONNEGATIVE, false
		}
		return sum, left_negative, true
	}
	if left_word >= right_word {
		return left_word - right_word, left_negative, true
	}
	return right_word - left_word, right_negative, true
}

// INT_DIVISION_WORD_REMAINDER_MAXIMUM is below even largest native divisor.
const INT_DIVISION_WORD_REMAINDER_MAXIMUM = bits.WORD_64_MAXIMUM - bits.CARRY_MAXIMUM

// Int_Division_Word_Remainder excludes largest word because remainder is below divisor.
type Int_Division_Word_Remainder Word

// Int_Division_Word_Remainder_Invariants bounds residual magnitude after native division.
func Int_Division_Word_Remainder_Invariants(
	value Int_Division_Word_Remainder, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Uint64(
			uint64(value), bits.CARRY_MINIMUM, INT_DIVISION_WORD_REMAINDER_MAXIMUM,
		).
		Ensure()
}

// Int_Division_Word_Result_Handle keeps division outputs in caller-owned scalar storage.
type Int_Division_Word_Result_Handle *Int_Division_Word_Result

// Int_Division_Word_Result_Handle_Invariants composes output storage when present.
func Int_Division_Word_Result_Handle_Invariants(
	value Int_Division_Word_Result_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Int_Division_Word_Result_Invariants(*value, namespace)
}

// Rat_Parse_Binary_Exponent preserves signed syntax until magnitude validation.
type Rat_Parse_Binary_Exponent int64

// Rat_Parse_Binary_Exponent_Invariants permits every signed exponent before scaling.
func Rat_Parse_Binary_Exponent_Invariants(
	value Rat_Parse_Binary_Exponent, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Rat_Parse_Five_Exponent preserves decimal prime scaling until magnitude validation.
type Rat_Parse_Five_Exponent int64

// Rat_Parse_Five_Exponent_Invariants permits every signed exponent before scaling.
func Rat_Parse_Five_Exponent_Invariants(
	value Rat_Parse_Five_Exponent, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

func float_rounding_increment(
	mode Rounding_Mode,
	negative Polarity,
	rounding_bit Bit_Value,
	sticky_bit Bit_Value,
	least_bit Bit_Value,
) (increment Boolean) {
	defer func() { Boolean_Invariants(increment, "float_rounding_increment.increment") }()
	Rounding_Mode_Invariants(mode, "float_rounding_increment.mode")
	Polarity_Invariants(negative, "float_rounding_increment.negative")
	Bit_Value_Invariants(rounding_bit, "float_rounding_increment.rounding_bit")
	Bit_Value_Invariants(sticky_bit, "float_rounding_increment.sticky_bit")
	Bit_Value_Invariants(least_bit, "float_rounding_increment.least_bit")
	switch mode {
	case Rounding_Mode(ROUND_TO_NEAREST_EVEN):
		return Boolean(rounding_bit != BIT_CLEAR &&
			(sticky_bit != BIT_CLEAR || least_bit != BIT_CLEAR))
	case Rounding_Mode(ROUND_TO_NEAREST_AWAY):
		return Boolean(rounding_bit != BIT_CLEAR)
	case Rounding_Mode(ROUND_TO_ZERO):
		return false
	case Rounding_Mode(ROUND_AWAY_FROM_ZERO):
		return true
	case Rounding_Mode(ROUND_TO_NEGATIVE_INFINITY):
		return Boolean(negative == POLARITY_NEGATIVE)
	case Rounding_Mode(ROUND_TO_POSITIVE_INFINITY):
		return Boolean(negative == POLARITY_NONNEGATIVE)
	}
	return false
}

// Int_Double_Word is one normalized nonnegative two-word magnitude.
type Int_Double_Word Int

// Int_Double_Word_Invariants states exact shape selected by public square-root dispatch.
func Int_Double_Word_Invariants(value Int_Double_Word, _ aver.Namespace) {
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"A double-word integer retains complete caller storage.")
	aver.Always(
		value.Negative == POLARITY_NONNEGATIVE,
		"A double-word integer is nonnegative.",
	)
	aver.Always(
		value.Count == Word_Count(BASE_BINARY),
		"A double-word integer owns exactly two active words.",
	)
	aver.Always(
		value.Words[WORD_COUNT_INCREMENT] != 0,
		"A double-word integer retains its high word.",
	)
}

// Two_Word_Magnitude admits caller storage beyond its two active words.
type Two_Word_Magnitude Int

// Two_Word_Magnitude_Invariants separates active width from storage capacity.
func Two_Word_Magnitude_Invariants(value Two_Word_Magnitude, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value.Words), BASE_BINARY, WORD_COUNT_MAXIMUM).
		Ensure()
	aver.Always(value.Count == Word_Count(BASE_BINARY),
		"Two-word Euclidean operands have exactly two active words.")
	aver.Always(value.Negative == POLARITY_NONNEGATIVE,
		"Two-word Euclidean operands have positive magnitude.")
	aver.Always(value.Words[WORD_COUNT_INCREMENT] != 0,
		"Two-word Euclidean operands retain a nonzero high word.")
}

// Two_Word_Magnitude_Handle preserves caller-owned Euclidean operands.
type Two_Word_Magnitude_Handle *Two_Word_Magnitude

// Two_Word_Magnitude_Handle_Invariants checks the present operand shape.
func Two_Word_Magnitude_Handle_Invariants(
	value Two_Word_Magnitude_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Two_Word_Magnitude_Invariants(*value, namespace)
}

// Euclidean_Dividend keeps the rotating dividend distinct in one invariant chain.
type Euclidean_Dividend Int

// Euclidean_Dividend_Invariants preserves complete normalized Int storage.
func Euclidean_Dividend_Invariants(value Euclidean_Dividend, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Enum_Uint8(
			uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE),
		).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Euclidean dividend storage is complete.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	index := (value.Count - Word_Count(WORD_COUNT_INCREMENT)) & Word_Count(WORD_INDEX_MAXIMUM)
	aver.Always(value.Words[index] >= minimum, "A Euclidean dividend is normalized.")
	aver.Always(
		int(value.Negative) <= int(value.Count),
		"A Euclidean dividend gives zero no negative twin.",
	)
}

// Euclidean_Divisor keeps the rotating divisor distinct in one invariant chain.
type Euclidean_Divisor Int

// Euclidean_Divisor_Invariants preserves complete normalized Int storage.
func Euclidean_Divisor_Invariants(value Euclidean_Divisor, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Enum_Uint8(
			uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE),
		).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Euclidean divisor storage is complete.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	index := (value.Count - Word_Count(WORD_COUNT_INCREMENT)) & Word_Count(WORD_INDEX_MAXIMUM)
	aver.Always(value.Words[index] >= minimum, "A Euclidean divisor is normalized.")
	aver.Always(
		int(value.Negative) <= int(value.Count),
		"A Euclidean divisor gives zero no negative twin.",
	)
}

// Euclidean_Remainder keeps the rotating remainder distinct in one invariant chain.
type Euclidean_Remainder Int

// Euclidean_Remainder_Invariants preserves complete normalized Int storage.
func Euclidean_Remainder_Invariants(value Euclidean_Remainder, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Enum_Uint8(
			uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE),
		).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Euclidean remainder storage is complete.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	index := (value.Count - Word_Count(WORD_COUNT_INCREMENT)) & Word_Count(WORD_INDEX_MAXIMUM)
	aver.Always(value.Words[index] >= minimum, "A Euclidean remainder is normalized.")
	aver.Always(
		int(value.Negative) <= int(value.Count),
		"A Euclidean remainder gives zero no negative twin.",
	)
}

// Int_Destination keeps destination state distinct from both operands in one assertion chain.
type Int_Destination Int

// Int_Destination_Invariants preserves complete normalized destination storage.
func Int_Destination_Invariants(value Int_Destination, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Enum_Uint8(
			uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE),
		).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM, "Int destination storage is complete.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	index := (value.Count - Word_Count(WORD_COUNT_INCREMENT)) & Word_Count(WORD_INDEX_MAXIMUM)
	aver.Always(value.Words[index] >= minimum, "An Int destination is normalized.")
	aver.Always(
		int(value.Negative) <= int(value.Count),
		"An Int destination gives zero no negative twin.",
	)
}

// Int_Left keeps the left operand distinct from other references in one assertion chain.
type Int_Left Int

// Int_Left_Invariants preserves complete normalized left-operand storage.
func Int_Left_Invariants(value Int_Left, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Enum_Uint8(
			uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE),
		).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM, "Left Int storage is complete.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	index := (value.Count - Word_Count(WORD_COUNT_INCREMENT)) & Word_Count(WORD_INDEX_MAXIMUM)
	aver.Always(value.Words[index] >= minimum, "A left Int operand is normalized.")
	aver.Always(
		int(value.Negative) <= int(value.Count),
		"A left Int operand gives zero no negative twin.",
	)
}

// Int_Right keeps the right operand distinct from other references in one assertion chain.
type Int_Right Int

// Int_Right_Invariants preserves complete normalized right-operand storage.
func Int_Right_Invariants(value Int_Right, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Enum_Uint8(
			uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE),
		).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM, "Right Int storage is complete.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	index := (value.Count - Word_Count(WORD_COUNT_INCREMENT)) & Word_Count(WORD_INDEX_MAXIMUM)
	aver.Always(value.Words[index] >= minimum, "A right Int operand is normalized.")
	aver.Always(
		int(value.Negative) <= int(value.Count),
		"A right Int operand gives zero no negative twin.",
	)
}

// Word_Count_Handle keeps parse progress in caller storage.
type Word_Count_Handle *Word_Count

// Word_Count_Handle_Invariants checks present count storage.
func Word_Count_Handle_Invariants(value Word_Count_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Word_Count_Invariants(*value, namespace)
}

// Division_Memory_Handle preserves mutable division scratch across calls.
type Division_Memory_Handle *Division_Memory

// Division_Memory_Handle_Invariants checks present division scratch.
func Division_Memory_Handle_Invariants(
	value Division_Memory_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Division_Memory_Invariants(*value, namespace)
}

// Word_Scale_Handle preserves mutable alignment across reduction steps.
type Word_Scale_Handle *Word_Scale

// Word_Scale_Handle_Invariants checks present alignment storage.
func Word_Scale_Handle_Invariants(value Word_Scale_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Word_Scale_Invariants(*value, namespace)
}

// Float_Exponent_Handle preserves scale changes caused by rounding carry.
type Float_Exponent_Handle *Float_Exponent

// Float_Exponent_Handle_Invariants checks present exponent storage.
func Float_Exponent_Handle_Invariants(value Float_Exponent_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Float_Exponent_Invariants(*value, namespace)
}

// Int_Double_Word_Handle preserves source storage during square-root extraction.
type Int_Double_Word_Handle *Int_Double_Word

// Int_Double_Word_Handle_Invariants checks present double-word storage.
func Int_Double_Word_Handle_Invariants(value Int_Double_Word_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Int_Double_Word_Invariants(*value, namespace)
}

// Float_Active_Mantissa_Handle preserves mutable magnitude during rounding.
type Float_Active_Mantissa_Handle *Float_Active_Mantissa

// Float_Active_Mantissa_Handle_Invariants checks present active magnitude.
func Float_Active_Mantissa_Handle_Invariants(
	value Float_Active_Mantissa_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Active_Mantissa_Invariants(*value, namespace)
}

// Float_Mantissa_Handle preserves magnitude storage across conversion.
type Float_Mantissa_Handle *Float_Mantissa

// Float_Mantissa_Handle_Invariants checks present magnitude storage.
func Float_Mantissa_Handle_Invariants(value Float_Mantissa_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Float_Mantissa_Invariants(*value, namespace)
}

// Int_Division_Empty_Workspace_Handle preserves reset scratch through word division.
type Int_Division_Empty_Workspace_Handle *Int_Division_Empty_Workspace

// Int_Division_Empty_Workspace_Handle_Invariants checks present reset scratch.
func Int_Division_Empty_Workspace_Handle_Invariants(
	value Int_Division_Empty_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Int_Division_Empty_Workspace_Invariants(*value, namespace)
}

// Positive_Magnitude excludes zero and sign handling completed before magnitude arithmetic.
type Positive_Magnitude Int

// Positive_Magnitude_Invariants keeps positive normalized values in complete scratch storage.
func Positive_Magnitude_Invariants(value Positive_Magnitude, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_INCREMENT, WORD_COUNT_MAXIMUM).
		Ensure()
	aver.Always(value.Negative == POLARITY_NONNEGATIVE,
		"Positive magnitude has no negative sign.")
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Positive magnitude scratch contains every bounded word.")
	high_index := (value.Count - Word_Count(WORD_COUNT_INCREMENT)) &
		Word_Count(WORD_INDEX_MAXIMUM)
	aver.Always(value.Words[high_index] != 0,
		"Positive magnitude omits high zero words.")
}

// Positive_Magnitude_Handle permits in-place magnitude arithmetic without copying storage.
type Positive_Magnitude_Handle *Positive_Magnitude

// Positive_Magnitude_Handle_Invariants checks present positive arithmetic scratch.
func Positive_Magnitude_Handle_Invariants(
	value Positive_Magnitude_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Positive_Magnitude_Invariants(*value, namespace)
}

// Int_Division_Multiword_Magnitude_Handle preserves source magnitude during division.
type Int_Division_Multiword_Magnitude_Handle *Int_Division_Multiword_Magnitude

// Int_Division_Multiword_Magnitude_Handle_Invariants checks present multiword source.
func Int_Division_Multiword_Magnitude_Handle_Invariants(
	value Int_Division_Multiword_Magnitude_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Int_Division_Multiword_Magnitude_Invariants(*value, namespace)
}

// Int_Division_General_Divisor_Handle preserves divisor across restoring steps.
type Int_Division_General_Divisor_Handle *Int_Division_General_Divisor

// Int_Division_General_Divisor_Handle_Invariants checks present restoring divisor.
func Int_Division_General_Divisor_Handle_Invariants(
	value Int_Division_General_Divisor_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Int_Division_General_Divisor_Invariants(*value, namespace)
}

// Int_Division_Restoring_Workspace_Handle preserves partial quotient across steps.
type Int_Division_Restoring_Workspace_Handle *Int_Division_Restoring_Workspace

// Int_Division_Restoring_Workspace_Handle_Invariants checks present restoring scratch.
func Int_Division_Restoring_Workspace_Handle_Invariants(
	value Int_Division_Restoring_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Int_Division_Restoring_Workspace_Invariants(*value, namespace)
}

// Int_Division_Small_Dividend_Handle preserves source through small division.
type Int_Division_Small_Dividend_Handle *Int_Division_Small_Dividend

// Int_Division_Small_Dividend_Handle_Invariants checks present small source.
func Int_Division_Small_Dividend_Handle_Invariants(
	value Int_Division_Small_Dividend_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Int_Division_Small_Dividend_Invariants(*value, namespace)
}

// Int_Division_Small_Divisor_Handle preserves divisor through small division.
type Int_Division_Small_Divisor_Handle *Int_Division_Small_Divisor

// Int_Division_Small_Divisor_Handle_Invariants checks present small divisor.
func Int_Division_Small_Divisor_Handle_Invariants(
	value Int_Division_Small_Divisor_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Int_Division_Small_Divisor_Invariants(*value, namespace)
}

// Int_Division_Small_Trial_Handle preserves normalized trial operands across estimation.
type Int_Division_Small_Trial_Handle *Int_Division_Small_Trial

// Int_Division_Small_Trial_Handle_Invariants checks present trial storage.
func Int_Division_Small_Trial_Handle_Invariants(
	value Int_Division_Small_Trial_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Int_Division_Small_Trial_Invariants(*value, namespace)
}

// Int_Division_Estimate_Handle preserves estimate output across helper calls.
type Int_Division_Estimate_Handle *Int_Division_Estimate

// Int_Division_Estimate_Handle_Invariants checks present estimate storage.
func Int_Division_Estimate_Handle_Invariants(
	value Int_Division_Estimate_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Int_Division_Estimate_Invariants(*value, namespace)
}

// Int_Handle keeps caller-owned integer storage explicit.
type Int_Handle *Int

// Int_Handle_Invariants composes present caller storage.
func Int_Handle_Invariants(value Int_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Int_Invariants(*value, namespace)
}

// Float_Handle keeps caller-owned float storage explicit.
type Float_Handle *Float

// Float_Handle_Invariants composes present caller storage.
func Float_Handle_Invariants(value Float_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Float_Invariants(*value, namespace)
}

// Float_Finite_Handle keeps proven-finite storage explicit.
type Float_Finite_Handle *Float_Finite

// Float_Finite_Handle_Invariants composes present finite storage.
func Float_Finite_Handle_Invariants(value Float_Finite_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Float_Finite_Invariants(*value, namespace)
}

// Float_Rounding_Source_Handle keeps finite reduction storage explicit.
type Float_Rounding_Source_Handle *Float_Rounding_Source

// Float_Rounding_Source_Handle_Invariants composes present reduction storage.
func Float_Rounding_Source_Handle_Invariants(
	value Float_Rounding_Source_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Rounding_Source_Invariants(*value, namespace)
}

// Float_Addition_Workspace_Handle keeps addition scratch explicit.
type Float_Addition_Workspace_Handle *Float_Addition_Workspace

// Float_Addition_Workspace_Handle_Invariants composes present addition scratch.
func Float_Addition_Workspace_Handle_Invariants(
	value Float_Addition_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Addition_Workspace_Invariants(*value, namespace)
}

// Float_Multiplication_Workspace_Handle keeps multiplication scratch explicit.
type Float_Multiplication_Workspace_Handle *Float_Multiplication_Workspace

// Float_Multiplication_Workspace_Handle_Invariants composes present multiplication scratch.
func Float_Multiplication_Workspace_Handle_Invariants(
	value Float_Multiplication_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Multiplication_Workspace_Invariants(*value, namespace)
}

// Float_Division_Workspace_Handle keeps division scratch explicit.
type Float_Division_Workspace_Handle *Float_Division_Workspace

// Float_Division_Workspace_Handle_Invariants composes present division scratch.
func Float_Division_Workspace_Handle_Invariants(
	value Float_Division_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Division_Workspace_Invariants(*value, namespace)
}

// Int_Multiplication_Workspace_Handle keeps multiplication scratch explicit.
type Int_Multiplication_Workspace_Handle *Int_Multiplication_Workspace

// Int_Multiplication_Workspace_Handle_Invariants composes present multiplication scratch.
func Int_Multiplication_Workspace_Handle_Invariants(
	value Int_Multiplication_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Int_Multiplication_Workspace_Invariants(*value, namespace)
}

// Int_Division_Workspace_Handle keeps division scratch explicit.
type Int_Division_Workspace_Handle *Int_Division_Workspace

// Int_Division_Workspace_Handle_Invariants composes present division scratch.
func Int_Division_Workspace_Handle_Invariants(
	value Int_Division_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Int_Division_Workspace_Invariants(*value, namespace)
}

// Int_Bitwise_Workspace_Handle keeps bitwise scratch explicit.
type Int_Bitwise_Workspace_Handle *Int_Bitwise_Workspace

// Int_Bitwise_Workspace_Handle_Invariants composes present bitwise scratch.
func Int_Bitwise_Workspace_Handle_Invariants(
	value Int_Bitwise_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Int_Bitwise_Workspace_Invariants(*value, namespace)
}

// Int_Greatest_Common_Divisor_Workspace_Handle keeps Euclidean scratch explicit.
type Int_Greatest_Common_Divisor_Workspace_Handle *Int_Greatest_Common_Divisor_Workspace

// Int_Greatest_Common_Divisor_Workspace_Handle_Invariants composes present scratch.
func Int_Greatest_Common_Divisor_Workspace_Handle_Invariants(
	value Int_Greatest_Common_Divisor_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Int_Greatest_Common_Divisor_Workspace_Invariants(*value, namespace)
}

// Int_Square_Root_Workspace_Handle keeps Newton scratch explicit.
type Int_Square_Root_Workspace_Handle *Int_Square_Root_Workspace

// Int_Square_Root_Workspace_Handle_Invariants composes present scratch.
func Int_Square_Root_Workspace_Handle_Invariants(
	value Int_Square_Root_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Int_Square_Root_Workspace_Invariants(*value, namespace)
}

// Int_Random_Workspace_Handle keeps candidate scratch explicit.
type Int_Random_Workspace_Handle *Int_Random_Workspace

// Int_Random_Workspace_Handle_Invariants composes present scratch.
func Int_Random_Workspace_Handle_Invariants(
	value Int_Random_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Int_Random_Workspace_Invariants(*value, namespace)
}

// Int_Jacobi_Workspace_Handle keeps reduction scratch explicit.
type Int_Jacobi_Workspace_Handle *Int_Jacobi_Workspace

// Int_Jacobi_Workspace_Handle_Invariants composes present scratch.
func Int_Jacobi_Workspace_Handle_Invariants(
	value Int_Jacobi_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Int_Jacobi_Workspace_Invariants(*value, namespace)
}

// Int_Primality_Workspace_Handle keeps probable-prime scratch explicit.
type Int_Primality_Workspace_Handle *Int_Primality_Workspace

// Int_Primality_Workspace_Handle_Invariants composes present scratch.
func Int_Primality_Workspace_Handle_Invariants(
	value Int_Primality_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Int_Primality_Workspace_Invariants(*value, namespace)
}

// Int_Exponent_Workspace_Handle keeps exponentiation scratch explicit.
type Int_Exponent_Workspace_Handle *Int_Exponent_Workspace

// Int_Exponent_Workspace_Handle_Invariants composes present scratch.
func Int_Exponent_Workspace_Handle_Invariants(
	value Int_Exponent_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Int_Exponent_Workspace_Invariants(*value, namespace)
}

// Int_Modular_Workspace_Handle keeps modular scratch explicit.
type Int_Modular_Workspace_Handle *Int_Modular_Workspace

// Int_Modular_Workspace_Handle_Invariants composes present scratch.
func Int_Modular_Workspace_Handle_Invariants(
	value Int_Modular_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Int_Modular_Workspace_Invariants(*value, namespace)
}

// Int_Product_Workspace_Handle keeps product scratch explicit.
type Int_Product_Workspace_Handle *Int_Product_Workspace

// Int_Product_Workspace_Handle_Invariants composes present scratch.
func Int_Product_Workspace_Handle_Invariants(
	value Int_Product_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Int_Product_Workspace_Invariants(*value, namespace)
}

// Int_Text_Workspace_Handle keeps text scratch explicit.
type Int_Text_Workspace_Handle *Int_Text_Workspace

// Int_Text_Workspace_Handle_Invariants composes present scratch.
func Int_Text_Workspace_Handle_Invariants(
	value Int_Text_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Int_Text_Workspace_Invariants(*value, namespace)
}

// Rat_Text_Workspace_Handle keeps rational text scratch explicit.
type Rat_Text_Workspace_Handle *Rat_Text_Workspace

// Rat_Text_Workspace_Handle_Invariants composes present scratch.
func Rat_Text_Workspace_Handle_Invariants(
	value Rat_Text_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Rat_Text_Workspace_Invariants(*value, namespace)
}

// Rat_Float_Text_Workspace_Handle keeps decimal text scratch explicit.
type Rat_Float_Text_Workspace_Handle *Rat_Float_Text_Workspace

// Rat_Float_Text_Workspace_Handle_Invariants composes present scratch.
func Rat_Float_Text_Workspace_Handle_Invariants(
	value Rat_Float_Text_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Rat_Float_Text_Workspace_Invariants(*value, namespace)
}

// Rat_Float_Precision_Workspace_Handle keeps precision scratch explicit.
type Rat_Float_Precision_Workspace_Handle *Rat_Float_Precision_Workspace

// Rat_Float_Precision_Workspace_Handle_Invariants composes present scratch.
func Rat_Float_Precision_Workspace_Handle_Invariants(
	value Rat_Float_Precision_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Rat_Float_Precision_Workspace_Invariants(*value, namespace)
}

// Rat_Float_64_Workspace_Handle keeps binary64 scratch explicit.
type Rat_Float_64_Workspace_Handle *Rat_Float_64_Workspace

// Rat_Float_64_Workspace_Handle_Invariants composes present scratch.
func Rat_Float_64_Workspace_Handle_Invariants(
	value Rat_Float_64_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Rat_Float_64_Workspace_Invariants(*value, namespace)
}

// Rat_Parse_Fraction_Workspace_Handle keeps fraction parse scratch explicit.
type Rat_Parse_Fraction_Workspace_Handle *Rat_Parse_Fraction_Workspace

// Rat_Parse_Fraction_Workspace_Handle_Invariants composes present scratch.
func Rat_Parse_Fraction_Workspace_Handle_Invariants(
	value Rat_Parse_Fraction_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Rat_Parse_Fraction_Workspace_Invariants(*value, namespace)
}

// Rat_Parse_Workspace_Handle keeps rational parse scratch explicit.
type Rat_Parse_Workspace_Handle *Rat_Parse_Workspace

// Rat_Parse_Workspace_Handle_Invariants composes present scratch.
func Rat_Parse_Workspace_Handle_Invariants(
	value Rat_Parse_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Rat_Parse_Workspace_Invariants(*value, namespace)
}

// Int_Parse_Workspace_Handle keeps integer parse scratch explicit.
type Int_Parse_Workspace_Handle *Int_Parse_Workspace

// Int_Parse_Workspace_Handle_Invariants composes present scratch.
func Int_Parse_Workspace_Handle_Invariants(
	value Int_Parse_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Int_Parse_Workspace_Invariants(*value, namespace)
}

// Rat_Handle keeps caller-owned rational storage explicit.
type Rat_Handle *Rat

// Rat_Handle_Invariants composes present rational storage.
func Rat_Handle_Invariants(value Rat_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Rat_Invariants(*value, namespace)
}

// Rat_Workspace_Handle keeps rational arithmetic scratch explicit.
type Rat_Workspace_Handle *Rat_Workspace

// Rat_Workspace_Handle_Invariants composes present scratch.
func Rat_Workspace_Handle_Invariants(value Rat_Workspace_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Rat_Workspace_Invariants(*value, namespace)
}

// Rat_Float_32_Workspace_Handle keeps binary32 scratch explicit.
type Rat_Float_32_Workspace_Handle *Rat_Float_32_Workspace

// Rat_Float_32_Workspace_Handle_Invariants composes present scratch.
func Rat_Float_32_Workspace_Handle_Invariants(
	value Rat_Float_32_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Rat_Float_32_Workspace_Invariants(*value, namespace)
}

// Int_Modular_Square_Root_Workspace_Handle keeps Tonelli-Shanks scratch explicit.
type Int_Modular_Square_Root_Workspace_Handle *Int_Modular_Square_Root_Workspace

// Int_Modular_Square_Root_Workspace_Handle_Invariants composes present scratch.
func Int_Modular_Square_Root_Workspace_Handle_Invariants(
	value Int_Modular_Square_Root_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Int_Modular_Square_Root_Workspace_Invariants(*value, namespace)
}

// Float_Text_Workspace_Handle keeps formatting scratch explicit.
type Float_Text_Workspace_Handle *Float_Text_Workspace

// Float_Text_Workspace_Handle_Invariants composes present scratch.
func Float_Text_Workspace_Handle_Invariants(
	value Float_Text_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Text_Workspace_Invariants(*value, namespace)
}

// Float_Parse_Workspace_Handle keeps parse scratch explicit.
type Float_Parse_Workspace_Handle *Float_Parse_Workspace

// Float_Parse_Workspace_Handle_Invariants composes present scratch.
func Float_Parse_Workspace_Handle_Invariants(
	value Float_Parse_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Parse_Workspace_Invariants(*value, namespace)
}

// Float_Parse_Factors excludes source bytes and syntax scratch after scanning.
type Float_Parse_Factors struct {
	// Control carries scanned signs and exponents into exact conversion.
	Control Float_Parse_Control
	// Words preserves scanned mantissa before factor construction mutates integers.
	Words Int_Parse_Words
	// Integers keeps exact factors independent from rounded operands.
	Integers Rat_Parse_Fraction_Integers_Handle
	// Values preserves quotient mutations in caller storage.
	Values Float_Parse_Values_Handle
	// Division retains aligned quotient scratch across conversion.
	Division Float_Parse_Division_Workspace_Handle
}

// Float_Parse_Factors_Invariants composes only post-scan conversion dependencies.
func Float_Parse_Factors_Invariants(value Float_Parse_Factors, namespace aver.Namespace) {
	Float_Parse_Control_Invariants(value.Control, namespace)
	Int_Parse_Words_Invariants(value.Words, namespace)
	Rat_Parse_Fraction_Integers_Handle_Invariants(value.Integers, namespace)
	Float_Parse_Values_Handle_Invariants(value.Values, namespace)
	Float_Parse_Division_Workspace_Handle_Invariants(value.Division, namespace)
}

// Float_Parse_Values separates initialized parse result from arbitrary operand scratch.
type Float_Parse_Values Float_Rat_Values

// Float_Parse_Values_Invariants preserves operand storage and initialized result policy.
func Float_Parse_Values_Invariants(value Float_Parse_Values, namespace aver.Namespace) {
	Float_Rat_Numerator_Invariants(value.Numerator, namespace)
	Float_Rat_Denominator_Invariants(value.Denominator, namespace)
	Float_Parse_Result_Invariants(Float_Parse_Result(value.Result), namespace)
}

// Float_Parse_Values_Handle keeps conversion mutations in caller storage.
type Float_Parse_Values_Handle *Float_Parse_Values

// Float_Parse_Values_Handle_Invariants composes present post-scan state.
func Float_Parse_Values_Handle_Invariants(
	value Float_Parse_Values_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Parse_Values_Invariants(*value, namespace)
}

// Float_Parse_Result carries destination policy before exact factors are converted.
type Float_Parse_Result Float

// Float_Parse_Result_Invariants records initialization performed before syntax scanning.
func Float_Parse_Result_Invariants(value Float_Parse_Result, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint(uint(value.Precision), WORD_COUNT_INCREMENT, FLOAT_PRECISION_MAXIMUM).
		Range_Uint8(uint8(value.Mode), uint8(ROUND_TO_NEAREST_EVEN),
			uint8(ROUND_TO_POSITIVE_INFINITY)).
		Ensure()
	aver.Always(value.Form == FLOAT_FORM_ZERO, "Parse result starts with zero form.")
	aver.Always(value.Accuracy == ACCURACY_EXACT, "Parse result starts with exact accuracy.")
	aver.Always(value.Negative == POLARITY_NONNEGATIVE, "Parse result starts nonnegative.")
	aver.Always(value.Exponent == FLOAT_EXPONENT_ZERO,
		"Parse result starts with zero exponent.")
	Float_Parse_Result_Mantissa_Invariants(
		Float_Parse_Result_Mantissa(value.Mantissa), namespace,
	)
}

// Float_Parse_Result_Mantissa retains full storage after zero initialization.
type Float_Parse_Result_Mantissa Float_Mantissa

// Float_Parse_Result_Mantissa_Invariants excludes stale magnitude and sign metadata.
func Float_Parse_Result_Mantissa_Invariants(value Float_Parse_Result_Mantissa, _ aver.Namespace) {
	aver.Always(value.Count == WORD_COUNT_MINIMUM, "Parse result starts with empty mantissa.")
	aver.Always(value.Negative == POLARITY_NONNEGATIVE,
		"Parse result starts with nonnegative mantissa.")
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Parse result retains complete caller mantissa storage.")
}

// Float_Rat_Values_Handle preserves independent operand and result metadata.
type Float_Rat_Values_Handle *Float_Rat_Values

// Float_Rat_Values_Handle_Invariants composes present conversion storage.
func Float_Rat_Values_Handle_Invariants(value Float_Rat_Values_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Float_Rat_Values_Invariants(*value, namespace)
}

// Float_Parse_Division_Workspace_Handle retains parse-specific alignment constraints.
type Float_Parse_Division_Workspace_Handle *Float_Parse_Division_Workspace

// Float_Parse_Division_Workspace_Handle_Invariants composes present quotient storage.
func Float_Parse_Division_Workspace_Handle_Invariants(
	value Float_Parse_Division_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Parse_Division_Workspace_Invariants(*value, namespace)
}

// Float_Nonnegative_Finite_Handle keeps square-root source proof explicit.
type Float_Nonnegative_Finite_Handle *Float_Nonnegative_Finite

// Float_Nonnegative_Finite_Handle_Invariants composes present source.
func Float_Nonnegative_Finite_Handle_Invariants(
	value Float_Nonnegative_Finite_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Nonnegative_Finite_Invariants(*value, namespace)
}

// Rat_Integers_Handle preserves component metadata in caller-owned storage.
type Rat_Integers_Handle *Rat_Integers

// Rat_Integers_Handle_Invariants composes present components.
func Rat_Integers_Handle_Invariants(value Rat_Integers_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Rat_Integers_Invariants(*value, namespace)
}

// Float_Rat_Workspace_Handle keeps rational conversion scratch explicit.
type Float_Rat_Workspace_Handle *Float_Rat_Workspace

// Float_Rat_Workspace_Handle_Invariants composes present scratch.
func Float_Rat_Workspace_Handle_Invariants(
	value Float_Rat_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Rat_Workspace_Invariants(*value, namespace)
}

// Float_Square_Root_Workspace_Handle keeps restoring-root scratch explicit.
type Float_Square_Root_Workspace_Handle *Float_Square_Root_Workspace

// Float_Square_Root_Workspace_Handle_Invariants composes present scratch.
func Float_Square_Root_Workspace_Handle_Invariants(
	value Float_Square_Root_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Square_Root_Workspace_Invariants(*value, namespace)
}

// Float_Gob_Workspace_Handle keeps wire scratch explicit.
type Float_Gob_Workspace_Handle *Float_Gob_Workspace

// Float_Gob_Workspace_Handle_Invariants composes present scratch.
func Float_Gob_Workspace_Handle_Invariants(
	value Float_Gob_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Gob_Workspace_Invariants(*value, namespace)
}

// Float_Arithmetic_Words holds complete aligned float arithmetic storage.
type Float_Arithmetic_Words []Word

// Float_Arithmetic_Words_Invariants fixes caller storage to exact arithmetic width.
func Float_Arithmetic_Words_Invariants(value Float_Arithmetic_Words, _ aver.Namespace) {
	aver.Always(len(value) == FLOAT_ADDITION_WORD_COUNT_MAXIMUM,
		"Float arithmetic storage has exact width.")
}

// Float_Division_Remainder holds shifted dividend storage.
type Float_Division_Remainder []Word

// Float_Division_Remainder_Invariants fixes one carry-width dividend.
func Float_Division_Remainder_Invariants(value Float_Division_Remainder, _ aver.Namespace) {
	aver.Always(len(value) == FLOAT_DIVISION_WORD_COUNT_MAXIMUM,
		"Float division remainder has exact width.")
}

// Float_Division_Divisor holds aligned divisor storage.
type Float_Division_Divisor []Word

// Float_Division_Divisor_Invariants fixes one carry-width divisor.
func Float_Division_Divisor_Invariants(value Float_Division_Divisor, _ aver.Namespace) {
	aver.Always(len(value) == FLOAT_DIVISION_WORD_COUNT_MAXIMUM,
		"Float division divisor has exact width.")
}

// Double_Words excludes unrelated metadata from two-word arithmetic.
type Double_Words struct {
	// Low preserves bits below carry boundary.
	Low Word
	// High excludes single-word operands already handled by caller.
	High Float_Active_Word
}

// Double_Words_Invariants preserves normalized two-word magnitude.
func Double_Words_Invariants(value Double_Words, namespace aver.Namespace) {
	Word_Invariants(value.Low, namespace)
	Float_Active_Word_Invariants(value.High, namespace)
}

func float_accuracy(increment Boolean, negative Polarity) (accuracy Inexact_Accuracy) {
	defer func() {
		Inexact_Accuracy_Invariants(accuracy, "float_accuracy_internal.accuracy")
	}()
	Boolean_Invariants(increment, "float_accuracy_internal.increment")
	Polarity_Invariants(negative, "float_accuracy_internal.negative")
	if increment == Boolean(negative == POLARITY_NEGATIVE) {
		return Inexact_Accuracy(ACCURACY_BELOW)
	}
	return Inexact_Accuracy(ACCURACY_ABOVE)
}

// Int_Product_Words holds one bounded integer product.
type Int_Product_Words []Word

// Int_Product_Words_Invariants fixes complete product storage.
func Int_Product_Words_Invariants(value Int_Product_Words, _ aver.Namespace) {
	aver.Always(len(value) == WORD_COUNT_MAXIMUM,
		"Integer product storage has exact width.")
}

// Int_Division_Quotient_Words holds one bounded quotient.
type Int_Division_Quotient_Words []Word

// Int_Division_Quotient_Words_Invariants fixes complete quotient storage.
func Int_Division_Quotient_Words_Invariants(value Int_Division_Quotient_Words, _ aver.Namespace) {
	aver.Always(len(value) == WORD_COUNT_MAXIMUM,
		"Integer quotient storage has exact width.")
}

// Int_Division_Remainder_Words holds one bounded remainder.
type Int_Division_Remainder_Words []Word

// Int_Division_Remainder_Words_Invariants fixes complete remainder storage.
func Int_Division_Remainder_Words_Invariants(value Int_Division_Remainder_Words, _ aver.Namespace) {
	aver.Always(len(value) == WORD_COUNT_MAXIMUM,
		"Integer remainder storage has exact width.")
}

// Int_Bitwise_Left_Words holds sign-extended left input.
type Int_Bitwise_Left_Words []Word

// Int_Bitwise_Left_Words_Invariants fixes complete signed width.
func Int_Bitwise_Left_Words_Invariants(value Int_Bitwise_Left_Words, _ aver.Namespace) {
	aver.Always(len(value) == BITWISE_WORD_COUNT_MAXIMUM,
		"Left bitwise storage has exact width.")
}

// Int_Bitwise_Right_Words holds sign-extended right input.
type Int_Bitwise_Right_Words []Word

// Int_Bitwise_Right_Words_Invariants fixes complete signed width.
func Int_Bitwise_Right_Words_Invariants(value Int_Bitwise_Right_Words, _ aver.Namespace) {
	aver.Always(len(value) == BITWISE_WORD_COUNT_MAXIMUM,
		"Right bitwise storage has exact width.")
}

// Int_Bitwise_Result_Words holds transactional signed output.
type Int_Bitwise_Result_Words []Word

// Int_Bitwise_Result_Words_Invariants fixes complete signed width.
func Int_Bitwise_Result_Words_Invariants(value Int_Bitwise_Result_Words, _ aver.Namespace) {
	aver.Always(len(value) == BITWISE_WORD_COUNT_MAXIMUM,
		"Result bitwise storage has exact width.")
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
type Double_Word_Products struct {
	// Low retains low-word multiplication before carry propagation.
	Low Double_Word_Product
	// Cross_Left preserves low-by-high multiplication independently.
	Cross_Left Double_Word_Cross_Left
	// Cross_Right preserves high-by-low multiplication independently.
	Cross_Right Double_Word_Cross_Right
	// High retains high-word multiplication before carry propagation.
	High Double_Word_Product_High
}

// Double_Word_Products_Invariants fixes exact two-by-two multiplication storage.
func Double_Word_Products_Invariants(value Double_Word_Products, namespace aver.Namespace) {
	Double_Word_Product_Invariants(value.Low, namespace)
	Double_Word_Cross_Left_Invariants(value.Cross_Left, namespace)
	Double_Word_Cross_Right_Invariants(value.Cross_Right, namespace)
	Double_Word_Product_High_Invariants(value.High, namespace)
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
type Modular_Dividend_Limbs []uint32

// Modular_Dividend_Limbs_Invariants binds reduction scratch to its formula capacity.
func Modular_Dividend_Limbs_Invariants(
	value Modular_Dividend_Limbs, _ aver.Namespace,
) {
	aver.Always(
		len(value) == MODULAR_DIVIDEND_LIMB_COUNT,
		"Normalized modular dividend scratch has formula capacity.",
	)
}

// Modular_Divisor_Limbs holds every limb of one two-word modulus.
type Modular_Divisor_Limbs struct {
	// Low retains least-significant divisor bits during subtraction.
	Low bits.Word_32
	// Next retains upper half of low divisor word.
	Next Modular_Divisor_Next
	// Third retains lower half of high divisor word.
	Third Modular_Divisor_Third
	// High provides normalized quotient-estimation divisor.
	High Modular_Divisor_High
}

// Modular_Divisor_Limbs_Invariants binds normalized divisor scratch to two words.
func Modular_Divisor_Limbs_Invariants(
	value Modular_Divisor_Limbs, namespace aver.Namespace,
) {
	bits.Word_32_Invariants(value.Low, namespace)
	Modular_Divisor_Next_Invariants(value.Next, namespace)
	Modular_Divisor_Third_Invariants(value.Third, namespace)
	Modular_Divisor_High_Invariants(value.High, namespace)
}

func int_modular_reduce_normalized_double_word(
	references Int_References,
) (matched Boolean) {
	defer func() {
		Boolean_Invariants(matched, "int_modular_reduce_normalized_double_word.matched")
	}()
	Int_References_Invariants(
		references, "int_modular_reduce_normalized_double_word.references",
	)
	destination := (*Int)((*Int_Destination)(references.Destination))
	product := (*Int)((*Int_Left)(references.Left))
	modulus := (*Int)((*Int_Right)(references.Right))
	if modulus.Count != Word_Count(BASE_BINARY) {
		return false
	}
	if product.Count > Word_Count(BASE_BINARY*BASE_BINARY) {
		return false
	}
	if modulus.Words[WORD_COUNT_MINIMUM] == Word(bits.WORD_64_MAXIMUM) {
		if modulus.Words[WORD_COUNT_INCREMENT] == MODULAR_MERSENNE_HIGH_WORD {
			int_modular_reduce_mersenne_double_word(
				references.Destination, references.Left)
			return true
		}
	}
	var dividend_storage [MODULAR_DIVIDEND_LIMB_COUNT]uint32
	dividend := Modular_Dividend_Limbs(dividend_storage[:])
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
	modulus_low := uint64(modulus.Words[WORD_COUNT_MINIMUM])
	modulus_high := uint64(modulus.Words[WORD_COUNT_INCREMENT])
	if modulus_high>>bits.BIT_COUNT_32_MAXIMUM <= limb_mask>>bits.CARRY_MAXIMUM {
		return false
	}
	int_modular_reduce_normalized_limbs(dividend, Modular_Divisor_Limbs{
		Low:   bits.Word_32(modulus_low & limb_mask),
		Next:  Modular_Divisor_Next(modulus_low >> bits.BIT_COUNT_32_MAXIMUM),
		Third: Modular_Divisor_Third(modulus_high & limb_mask),
		High:  Modular_Divisor_High(modulus_high >> bits.BIT_COUNT_32_MAXIMUM),
	})
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
	destination_reference Int_Destination_Reference, product_reference Int_Left_Reference,
) {
	Int_Destination_Reference_Invariants(
		destination_reference, "int_modular_reduce_mersenne.destination")
	Int_Left_Reference_Invariants(product_reference, "int_modular_reduce_mersenne.product")
	destination := (*Int)((*Int_Destination)(destination_reference))
	product := (*Int)((*Int_Left)(product_reference))
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
}

func int_modular_multiply_identity(references Int_References) (matched Boolean) {
	defer func() { Boolean_Invariants(matched, "int_modular_multiply_identity.matched") }()
	Int_References_Invariants(references, "int_modular_multiply_identity.references")
	destination := (*Int)((*Int_Destination)(references.Destination))
	left := (*Int)((*Int_Left)(references.Left))
	right := (*Int)((*Int_Right)(references.Right))
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

func int_subtract_positive_double_word(references Int_References) (matched Boolean) {
	defer func() {
		Boolean_Invariants(matched, "int_subtract_positive_double_word.matched")
	}()
	Int_References_Invariants(references, "int_subtract_positive_double_word.references")
	destination := (*Int)((*Int_Destination)(references.Destination))
	left := (*Int)((*Int_Left)(references.Left))
	right := (*Int)((*Int_Right)(references.Right))
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

func int_multiply_double_word(references Int_References) (matched Boolean) {
	defer func() { Boolean_Invariants(matched, "int_multiply_double_word.matched") }()
	Int_References_Invariants(references, "int_multiply_double_word.references")
	destination := (*Int)((*Int_Destination)(references.Destination))
	left := (*Int)((*Int_Left)(references.Left))
	right := (*Int)((*Int_Right)(references.Right))
	if left.Count != Word_Count(BASE_BINARY) {
		return false
	}
	if right.Count != Word_Count(BASE_BINARY) {
		return false
	}
	products := int_multiply_double_word_products(references.Left, references.Right)
	high_low, low_low := products.Low.High, products.Low.Low
	high_cross_left, low_cross_left := products.Cross_Left.High, products.Cross_Left.Low
	high_cross_right, low_cross_right := products.Cross_Right.High, products.Cross_Right.Low
	high_high, low_high := products.High.High, products.High.Low
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
	left Int_Left_Reference, right Int_Right_Reference,
) (products Double_Word_Products) {
	defer func() {
		Double_Word_Products_Invariants(
			products, "int_multiply_double_word_products.products",
		)
	}()
	Int_Left_Reference_Invariants(left, "int_multiply_double_word_products.left")
	Int_Right_Reference_Invariants(right, "int_multiply_double_word_products.right")
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
	products.Low.Low = Word(left_word * right_word)
	products.Low.High = Word_Product_High(high)
	right_word = uint64(right.Words[WORD_COUNT_INCREMENT])
	right_low, right_high = right_word&uint64(bits.WORD_32_MAXIMUM),
		right_word>>bits.BIT_COUNT_32_MAXIMUM
	partial = left_low * right_low
	middle_first = left_high*right_low + partial>>bits.BIT_COUNT_32_MAXIMUM
	middle_second = left_low*right_high + middle_first&uint64(bits.WORD_32_MAXIMUM)
	high = left_high*right_high + middle_first>>bits.BIT_COUNT_32_MAXIMUM +
		middle_second>>bits.BIT_COUNT_32_MAXIMUM
	products.Cross_Left.Low = Word(left_word * right_word)
	products.Cross_Left.High = Word_Product_High(high)
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
	products.Cross_Right.Low = Word(left_word * right_word)
	products.Cross_Right.High = Word_Product_High(high)
	right_word = uint64(right.Words[WORD_COUNT_INCREMENT])
	right_low, right_high = right_word&uint64(bits.WORD_32_MAXIMUM),
		right_word>>bits.BIT_COUNT_32_MAXIMUM
	partial = left_low * right_low
	middle_first = left_high*right_low + partial>>bits.BIT_COUNT_32_MAXIMUM
	middle_second = left_low*right_high + middle_first&uint64(bits.WORD_32_MAXIMUM)
	high = left_high*right_high + middle_first>>bits.BIT_COUNT_32_MAXIMUM +
		middle_second>>bits.BIT_COUNT_32_MAXIMUM
	products.High.Low = Word(left_word * right_word)
	products.High.High = Word_Product_High(high)
	return products
}

// Rat_Absolute copies value and clears numerator sign.
func Rat_Absolute(destination Rat_Handle, source Rat_Handle) {
	Rat_Handle_Invariants(destination, "rat_absolute.destination")
	Rat_Handle_Invariants(source, "rat_absolute.source")
	numerator := &source.Integers.Numerator
	denominator := (*Int)(&source.Integers.Denominator)
	if numerator.Count == Word_Count(BASE_BINARY) {
		if denominator.Count == Word_Count(BASE_BINARY) {
			rat_copy_double_words(destination,
				Double_Words{Low: numerator.Words[WORD_COUNT_MINIMUM],
					High: Float_Active_Word(
						numerator.Words[WORD_COUNT_INCREMENT])},
				Double_Words{Low: denominator.Words[WORD_COUNT_MINIMUM],
					High: Float_Active_Word(
						denominator.Words[WORD_COUNT_INCREMENT])},
				POLARITY_NONNEGATIVE,
			)
			return
		}
	}
	Rat_Set(destination, source)
	destination.Integers.Numerator.Negative = POLARITY_NONNEGATIVE
}

// Rat_Negate copies value and flips only nonzero numerator sign.
func Rat_Negate(destination Rat_Handle, source Rat_Handle) {
	Rat_Handle_Invariants(destination, "rat_negate.destination")
	Rat_Handle_Invariants(source, "rat_negate.source")
	negative := POLARITY_NONNEGATIVE
	if source.Integers.Numerator.Count > WORD_COUNT_MINIMUM {
		negative = POLARITY_NEGATIVE - source.Integers.Numerator.Negative
	}
	numerator := &source.Integers.Numerator
	denominator := (*Int)(&source.Integers.Denominator)
	if numerator.Count == Word_Count(BASE_BINARY) {
		if denominator.Count == Word_Count(BASE_BINARY) {
			rat_copy_double_words(destination,
				Double_Words{Low: numerator.Words[WORD_COUNT_MINIMUM],
					High: Float_Active_Word(
						numerator.Words[WORD_COUNT_INCREMENT])},
				Double_Words{Low: denominator.Words[WORD_COUNT_MINIMUM],
					High: Float_Active_Word(
						denominator.Words[WORD_COUNT_INCREMENT])},
				negative,
			)
			return
		}
	}
	Rat_Set(destination, source)
	destination.Integers.Numerator.Negative = negative
}

func rat_copy_double_words(
	destination Rat_Handle, numerator Double_Words, denominator Double_Words, negative Polarity,
) {
	Rat_Handle_Invariants(destination, "rat_copy_double_words.destination")
	Double_Words_Invariants(numerator, "rat_copy_double_words.numerator")
	Double_Words_Invariants(denominator, "rat_copy_double_words.denominator")
	Polarity_Invariants(negative, "rat_copy_double_words.negative")
	destination_numerator := &destination.Integers.Numerator
	destination_denominator := (*Int)(&destination.Integers.Denominator)
	previous_numerator_count := destination_numerator.Count
	previous_denominator_count := destination_denominator.Count
	destination_numerator.Words[WORD_COUNT_MINIMUM] = numerator.Low
	destination_numerator.Words[WORD_COUNT_INCREMENT] = Word(numerator.High)
	destination_numerator.Count = Word_Count(BASE_BINARY)
	destination_numerator.Negative = negative
	destination_denominator.Words[WORD_COUNT_MINIMUM] = denominator.Low
	destination_denominator.Words[WORD_COUNT_INCREMENT] = Word(denominator.High)
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

func float_rat_set_zero(destination Rat_Handle) {
	Rat_Handle_Invariants(destination, "float_rat_set_zero.destination_initial")
	numerator := &destination.Integers.Numerator
	denominator := (*Int)(&destination.Integers.Denominator)
	int_zero(numerator, numerator.Count)
	int_zero(denominator, denominator.Count)
}

func rat_binary(
	destination Rat_Handle,
	left Rat_Handle,
	right Rat_Handle,
	workspace Rat_Workspace_Handle,
	operation Rat_Operation,
) (status Rat_Division_Status) {
	defer func() { Rat_Division_Status_Invariants(status, "rat_binary.status") }()
	Rat_Handle_Invariants(destination, "rat_binary.destination")
	Rat_Handle_Invariants(left, "rat_binary.left")
	Rat_Handle_Invariants(right, "rat_binary.right")
	Rat_Workspace_Handle_Invariants(workspace, "rat_binary.workspace")
	Rat_Operation_Invariants(operation, "rat_binary.operation")
	rat_load_operands(workspace, left, right)
	if operation == RAT_OPERATION_QUOTIENT {
		if workspace.Integers.Right_Numerator.Count == WORD_COUNT_MINIMUM {
			return STATUS_DIVISOR_ZERO
		}
	}
	integers := (*Rat_Loaded_Integers)(&workspace.Integers)
	result_normalized := rat_binary_result_normalized(
		integers, &workspace.Greatest_Common, operation,
	)
	multiplication := (*Int_Multiplication_Workspace)(&workspace.Multiplication)
	rat_binary_products(integers, multiplication, operation)
	if result_normalized {
		numerator := (*Int)(&workspace.Integers.Result_Numerator)
		denominator := (*Int)(&workspace.Integers.Result_Denominator)
		if numerator.Count > RAT_WORD_COUNT_MAXIMUM {
			return STATUS_VALUE_OVERFLOW
		}
		if denominator.Count > RAT_WORD_COUNT_MAXIMUM {
			return STATUS_VALUE_OVERFLOW
		}
		rat_commit_normalized(destination, numerator, denominator)
		return STATUS_OK
	}
	return rat_normalize(destination, workspace)
}

func rat_binary_result_normalized(
	integers Rat_Loaded_Integers_Handle, memory Greatest_Common_Divisor_Memory_Handle,
	operation Rat_Operation,
) (normalized Boolean) {
	defer func() { Boolean_Invariants(normalized, "rat_binary_result_normalized.normalized") }()
	Rat_Loaded_Integers_Handle_Invariants(integers, "rat_binary_result_normalized.integers")
	Greatest_Common_Divisor_Memory_Handle_Invariants(
		memory, "rat_binary_result_normalized.memory",
	)
	Rat_Operation_Invariants(operation, "rat_binary_result_normalized.operation")
	if operation == RAT_OPERATION_MULTIPLY {
		return false
	}
	greatest_common := (*Int_Greatest_Common_Divisor_Workspace)(
		(*Greatest_Common_Divisor_Memory)(memory),
	)
	left := (*Int)(&integers.Left_Denominator)
	right := (*Int)(&integers.Right_Denominator)
	if operation == RAT_OPERATION_QUOTIENT {
		left = (*Int)(&integers.Left_Numerator)
		right = (*Int)(&integers.Right_Numerator)
	}
	Int_Greatest_Common_Divisor(
		(*Int)(&integers.Common_Divisor), left, right, greatest_common,
	)
	common_divisor := (*Int)(&integers.Common_Divisor)
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
		common_divisor, (*Int)(&integers.Right_Denominator),
		(*Int)(&integers.Left_Denominator), greatest_common,
	)
	if common_divisor.Count != Word_Count(WORD_COUNT_INCREMENT) {
		return false
	}
	return common_divisor.Words[WORD_COUNT_MINIMUM] == Word(bits.CARRY_MAXIMUM)
}

// Greatest_Common_Divisor_Memory_Handle preserves reusable Euclidean scratch mutations.
type Greatest_Common_Divisor_Memory_Handle *Greatest_Common_Divisor_Memory

// Greatest_Common_Divisor_Memory_Handle_Invariants composes present reduction storage.
func Greatest_Common_Divisor_Memory_Handle_Invariants(
	value Greatest_Common_Divisor_Memory_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Greatest_Common_Divisor_Memory_Invariants(*value, namespace)
}

func int_exponent_accumulate(
	references Int_References, workspace Int_Multiplication_Workspace_Handle,
) (overflow Boolean) {
	defer func() { Boolean_Invariants(overflow, "int_exponent_accumulate.overflow") }()
	Int_References_Invariants(references, "int_exponent_accumulate.references")
	Int_Multiplication_Workspace_Handle_Invariants(
		workspace, "int_exponent_accumulate.workspace")
	destination := (*Int)((*Int_Destination)(references.Destination))
	factor := (*Int)((*Int_Right)(references.Right))
	if destination.Count == Word_Count(WORD_COUNT_INCREMENT) {
		if destination.Words[WORD_COUNT_MINIMUM] == Word(bits.CARRY_MAXIMUM) {
			Int_Set(destination, factor)
			return false
		}
	}
	status := Int_Multiply(destination, destination, factor, workspace)
	return status != Arithmetic_Status(STATUS_OK)
}

func int_square_double_word(destination Int_Handle) {
	Int_Handle_Invariants(destination, "int_square_double_word.destination")
	aver.Always(
		destination.Count == Word_Count(BASE_BINARY),
		"The symmetric square receives exactly two words.",
	)
	products := int_multiply_double_word_products(
		Int_Left_Reference((*Int_Left)((*Int)(destination))),
		Int_Right_Reference((*Int_Right)((*Int)(destination))),
	)
	low_square_low, low_square_high := products.Low.Low, products.Low.High
	cross_low, cross_high := products.Cross_Left.Low, products.Cross_Left.High
	high_square_low, high_square_high := products.High.Low, products.High.High
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

func int_square_root_double_word(destination Int_Handle, source Int_Double_Word_Handle) {
	Int_Handle_Invariants(destination, "int_square_root_double_word.destination_initial")
	Int_Double_Word_Handle_Invariants(source, "int_square_root_double_word.source")
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
	references Int_References, workspace Int_Modular_Workspace_Handle,
) {
	Int_References_Invariants(references, "int_primality_modular_multiply.references")
	Int_Modular_Workspace_Handle_Invariants(
		workspace, "int_primality_modular_multiply.workspace",
	)
	destination := (*Int)((*Int_Destination)(references.Destination))
	left := (*Int)((*Int_Left)(references.Left))
	right := (*Int)((*Int_Right)(references.Right))
	result := &workspace.Integers[MODULAR_EXPONENT_RESULT_INDEX]
	factor := &workspace.Integers[MODULAR_EXPONENT_FACTOR_INDEX]
	// Primality recurrences retain residues, so public re-reduction is redundant.
	Int_Set(result, left)
	Int_Set(factor, right)
	int_modular_multiply(workspace, MODULAR_MULTIPLICATION_ACCUMULATE)
	Int_Set(destination, result)
}

// Int_Compare keeps sign handling above magnitude ordering so zero needs no special magnitude.
func Int_Compare(left Int_Handle, right Int_Handle) (order Order) {
	defer func() { Order_Invariants(order, "int_compare.order") }()
	Int_Handle_Invariants(left, "int_compare.left")
	Int_Handle_Invariants(right, "int_compare.right")
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
func Int_Compare_Absolute(left Int_Handle, right Int_Handle) (order Order) {
	defer func() { Order_Invariants(order, "int_compare_absolute.order") }()
	Int_Handle_Invariants(left, "int_compare_absolute.left")
	Int_Handle_Invariants(right, "int_compare_absolute.right")
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
	destination Rat_Handle, numerator Int_Handle, denominator Int_Handle,
) {
	Rat_Handle_Invariants(destination, "rat_commit_normalized.destination")
	Int_Handle_Invariants(numerator, "rat_commit_normalized.numerator")
	Int_Handle_Invariants(denominator, "rat_commit_normalized.denominator")
	aver.Always(
		denominator.Count > WORD_COUNT_MINIMUM,
		"A normalized rational operation retains one nonzero denominator.",
	)
	aver.Always(
		denominator.Negative == POLARITY_NONNEGATIVE,
		"Normalized rational operands produce one nonnegative denominator.",
	)
	Int_Set(&destination.Integers.Numerator, numerator)
	Int_Set((*Int)(&destination.Integers.Denominator), denominator)
}

func int_modular_exponent_commit(
	workspace Int_Modular_Workspace_Handle, destination Int_Handle, exponent Int_Handle,
) {
	Int_Modular_Workspace_Handle_Invariants(workspace, "int_modular_exponent_commit.workspace")
	Int_Handle_Invariants(destination, "int_modular_exponent_commit.destination")
	Int_Handle_Invariants(exponent, "int_modular_exponent_commit.exponent")
	modulus := &workspace.Integers[MODULAR_MODULUS_INDEX]
	aver.Always(
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
	dividend Modular_Dividend_Limbs, divisor Modular_Divisor_Limbs,
) {
	Modular_Dividend_Limbs_Invariants(
		dividend, "int_modular_reduce_normalized_limbs.dividend",
	)
	Modular_Divisor_Limbs_Invariants(
		divisor, "int_modular_reduce_normalized_limbs.divisor",
	)
	limbs := [...]uint32{
		uint32(divisor.Low), uint32(divisor.Next),
		uint32(divisor.Third), uint32(divisor.High),
	}
	limb_mask := uint64(bits.WORD_32_MAXIMUM)
	divisor_count := MODULAR_DOUBLE_WORD_LIMB_COUNT
	for offset := divisor_count; offset >= WORD_COUNT_MINIMUM; offset-- {
		high_index := int(offset) + MODULAR_DIVISOR_LIMB_END_INDEX
		numerator := uint64(dividend[high_index])<<bits.BIT_COUNT_32_MAXIMUM |
			uint64(dividend[high_index-WORD_COUNT_INCREMENT])
		divisor_high := uint64(divisor.High)
		quotient := numerator / divisor_high
		remainder := numerator % divisor_high
		if quotient > limb_mask {
			quotient = limb_mask
			remainder = numerator - quotient*divisor_high
		}
		for remainder <= limb_mask {
			trial := remainder<<bits.BIT_COUNT_32_MAXIMUM |
				uint64(dividend[high_index-BASE_BINARY])
			divisor_next := uint64(divisor.Third)
			if quotient*divisor_next <= trial {
				break
			}
			quotient--
			remainder += divisor_high
		}
		borrow := uint64(0)
		for index := WORD_COUNT_MINIMUM; index < divisor_count; index++ {
			product_limb := quotient*uint64(limbs[index]) + borrow
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
					uint64(limbs[index]) + carry
				dividend[dividend_index] = uint32(sum)
				carry = sum >> bits.BIT_COUNT_32_MAXIMUM
			}
			dividend[high_index] += uint32(carry)
		}
	}
}

func int_modular_inverse_euclidean(
	workspace Int_Modular_Workspace_Handle,
	bounded Boolean,
	result_references Int_References,
	remainder_references Int_References,
	coefficient_references Int_References,
) (status Modular_Inverse_Result_Status) {
	defer func() {
		Modular_Inverse_Result_Status_Invariants(
			status, "int_modular_inverse_euclidean.status",
		)
	}()
	Int_Modular_Workspace_Handle_Invariants(
		workspace, "int_modular_inverse_euclidean.workspace",
	)
	Boolean_Invariants(bounded, "int_modular_inverse_euclidean.bounded")
	Int_References_Invariants(result_references, "int_modular_inverse_euclidean.result")
	Int_References_Invariants(
		remainder_references, "int_modular_inverse_euclidean.remainders",
	)
	Int_References_Invariants(
		coefficient_references, "int_modular_inverse_euclidean.coefficient",
	)
	if bounded {
		modulus := (*Int)((*Int_Left)(result_references.Left))
		if modulus.Count <= Word_Count(BASE_BINARY) {
			return int_modular_inverse_euclidean_double_word(
				workspace, result_references, remainder_references,
				coefficient_references,
			)
		}
	}
	return int_modular_inverse_euclidean_general(
		workspace, bounded, result_references.Destination,
	)
}

func int_modular_inverse_euclidean_double_word(
	workspace Int_Modular_Workspace_Handle,
	result_references Int_References,
	remainder_references Int_References,
	coefficient_references Int_References,
) (status Modular_Inverse_Result_Status) {
	defer func() {
		Modular_Inverse_Result_Status_Invariants(
			status, "int_modular_inverse_euclidean_double_word.status",
		)
	}()
	Int_Modular_Workspace_Handle_Invariants(workspace,
		"int_modular_inverse_euclidean_double_word.workspace")
	Int_References_Invariants(
		result_references, "int_modular_inverse_euclidean_double_word.result",
	)
	Int_References_Invariants(
		remainder_references, "int_modular_inverse_euclidean_double_word.remainders",
	)
	Int_References_Invariants(
		coefficient_references, "int_modular_inverse_euclidean_double_word.coefficient",
	)
	remainder := (*Int)((*Int_Destination)(remainder_references.Destination))
	dividend := (*Int)((*Int_Left)(remainder_references.Left))
	divisor := (*Int)((*Int_Right)(remainder_references.Right))
	for divisor.Count > Word_Count(WORD_COUNT_MINIMUM) {
		if divisor.Count == Word_Count(BASE_BINARY) {
			int_modular_inverse_divide_double_word(
				result_references.Right, remainder_references.Destination,
				(*Two_Word_Magnitude)(dividend), (*Two_Word_Magnitude)(divisor),
			)
		} else if dividend.Count == Word_Count(WORD_COUNT_INCREMENT) {
			int_modular_inverse_divide_word_dividend(
				result_references.Right, remainder_references.Destination,
				Int_Division_Nonzero_Word(dividend.Words[WORD_COUNT_MINIMUM]),
				Int_Division_Nonzero_Word(divisor.Words[WORD_COUNT_MINIMUM]),
			)
		} else if dividend.Words[WORD_COUNT_INCREMENT] < divisor.Words[WORD_COUNT_MINIMUM] {
			int_modular_inverse_divide_double_word_dividend(
				result_references.Right, remainder_references.Destination,
				Double_Words{Low: dividend.Words[WORD_COUNT_MINIMUM],
					High: Float_Active_Word(
						dividend.Words[WORD_COUNT_INCREMENT])},
				Int_Division_Nonzero_Word(divisor.Words[WORD_COUNT_MINIMUM]),
			)
		} else {
			int_modular_inverse_divide(
				(*Int_Division_Workspace)(&workspace.Division),
				result_references.Right, remainder_references,
			)
		}
		quotient := (*Int)((*Int_Right)(result_references.Right))
		if quotient.Count == Word_Count(WORD_COUNT_INCREMENT) {
			int_modular_inverse_coefficient_double_word(
				Int_Division_Nonzero_Word(quotient.Words[WORD_COUNT_MINIMUM]),
				coefficient_references,
			)
		} else {
			int_modular_inverse_coefficient_bounded(
				(*Int_Multiplication_Workspace)(&workspace.Multiplication),
				result_references.Right, coefficient_references, true,
			)
		}
		dividend, divisor, remainder = divisor, remainder, dividend
		remainder_references = int_references_rotate(remainder_references)
		coefficient_references = int_references_rotate(coefficient_references)
	}
	commit_references := Int_References{
		Destination: result_references.Destination,
		Left:        result_references.Left,
		Right:       Int_Right_Reference((*Int_Right)(dividend)),
	}
	return int_modular_inverse_commit(true, commit_references, coefficient_references)
}

func int_modular_inverse_divide(
	workspace Int_Division_Workspace_Handle,
	quotient_reference Int_Right_Reference,
	remainder_references Int_References,
) {
	Int_Division_Workspace_Handle_Invariants(
		workspace, "int_modular_inverse_divide.workspace",
	)
	Int_Right_Reference_Invariants(
		quotient_reference, "int_modular_inverse_divide.quotient",
	)
	Int_References_Invariants(
		remainder_references, "int_modular_inverse_divide.remainders",
	)
	quotient := (*Int)((*Int_Right)(quotient_reference))
	remainder := (*Int)((*Int_Destination)(remainder_references.Destination))
	dividend := (*Int)((*Int_Left)(remainder_references.Left))
	divisor := (*Int)((*Int_Right)(remainder_references.Right))
	if divisor.Count == Word_Count(BASE_BINARY) {
		if dividend.Count == Word_Count(BASE_BINARY) {
			int_modular_inverse_divide_double_word(
				quotient_reference, remainder_references.Destination,
				(*Two_Word_Magnitude)(dividend), (*Two_Word_Magnitude)(divisor),
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
	aver.Always(
		status == Division_Status(STATUS_OK),
		"Modular Euclid divides only by its nonzero current remainder.",
	)
}

func int_modular_inverse_divide_word_dividend(
	quotient_reference Int_Right_Reference, remainder_reference Int_Destination_Reference,
	dividend Int_Division_Nonzero_Word, divisor Int_Division_Nonzero_Word,
) {
	Int_Right_Reference_Invariants(
		quotient_reference, "int_modular_inverse_divide_word_dividend.quotient",
	)
	Int_Destination_Reference_Invariants(
		remainder_reference, "int_modular_inverse_divide_word_dividend.remainder",
	)
	Int_Division_Nonzero_Word_Invariants(
		dividend, "int_modular_inverse_divide_word_dividend.dividend",
	)
	Int_Division_Nonzero_Word_Invariants(
		divisor, "int_modular_inverse_divide_word_dividend.divisor",
	)
	quotient := (*Int)((*Int_Right)(quotient_reference))
	remainder := (*Int)((*Int_Destination)(remainder_reference))
	quotient.Words[WORD_COUNT_MINIMUM] = Word(dividend / divisor)
	quotient.Words[WORD_COUNT_INCREMENT] = 0
	quotient.Count = Word_Count(WORD_COUNT_INCREMENT)
	quotient.Negative = POLARITY_NONNEGATIVE
	remainder_word := Word(dividend % divisor)
	remainder.Words[WORD_COUNT_MINIMUM] = remainder_word
	remainder.Words[WORD_COUNT_INCREMENT] = 0
	remainder.Count = Word_Count(WORD_COUNT_INCREMENT)
	if remainder_word == 0 {
		remainder.Count = Word_Count(WORD_COUNT_MINIMUM)
	}
	remainder.Negative = POLARITY_NONNEGATIVE
}

func int_modular_inverse_divide_double_word_dividend(
	quotient_reference Int_Right_Reference,
	remainder_reference Int_Destination_Reference,
	dividend Double_Words,
	divisor Int_Division_Nonzero_Word,
) {
	Int_Right_Reference_Invariants(
		quotient_reference, "int_modular_inverse_divide_double_word_dividend.quotient",
	)
	Int_Destination_Reference_Invariants(
		remainder_reference, "int_modular_inverse_divide_double_word_dividend.remainder",
	)
	Double_Words_Invariants(
		dividend, "int_modular_inverse_divide_double_word_dividend.dividend",
	)
	Int_Division_Nonzero_Word_Invariants(
		divisor, "int_modular_inverse_divide_double_word_dividend.divisor",
	)
	quotient := (*Int)((*Int_Right)(quotient_reference))
	remainder := (*Int)((*Int_Destination)(remainder_reference))
	result, rest := bits.Divide_64(
		bits.Dividend_High_64(dividend.High),
		bits.Dividend_Low_64(dividend.Low),
		bits.Divisor_64(divisor),
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
}

func int_modular_inverse_divide_double_word(
	quotient_reference Int_Right_Reference, remainder_reference Int_Destination_Reference,
	dividend Two_Word_Magnitude_Handle, divisor Two_Word_Magnitude_Handle,
) {
	Int_Right_Reference_Invariants(
		quotient_reference, "int_modular_inverse_divide_double_word.quotient",
	)
	Int_Destination_Reference_Invariants(
		remainder_reference, "int_modular_inverse_divide_double_word.remainder",
	)
	Two_Word_Magnitude_Handle_Invariants(
		dividend, "int_modular_inverse_divide_double_word.dividend")
	Two_Word_Magnitude_Handle_Invariants(
		divisor, "int_modular_inverse_divide_double_word.divisor")
	remainder := (*Int)((*Int_Destination)(remainder_reference))
	dividend_high := dividend.Words[WORD_COUNT_INCREMENT]
	divisor_high := divisor.Words[WORD_COUNT_INCREMENT]
	dividend_words := Double_Words{
		Low: dividend.Words[WORD_COUNT_MINIMUM], High: Float_Active_Word(dividend_high),
	}
	divisor_words := Double_Words{
		Low: divisor.Words[WORD_COUNT_MINIMUM], High: Float_Active_Word(divisor_high),
	}
	factor := Int_Division_Quotient_Word(WORD_COUNT_INCREMENT)
	if divisor_high != Word(bits.WORD_64_MAXIMUM) {
		factor = Int_Division_Quotient_Word(max(
			dividend_high/(divisor_high+Word(WORD_COUNT_INCREMENT)),
			Word(WORD_COUNT_INCREMENT),
		))
	}
	int_modular_inverse_divide_double_word_commit(
		quotient_reference, remainder_reference,
		dividend_words, divisor_words, factor,
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
		(*Int_Division_Multiword_Magnitude)((*Two_Word_Magnitude)(dividend)),
		(*Int_Division_Multiword_Magnitude)((*Two_Word_Magnitude)(divisor)),
	)
	int_modular_inverse_divide_double_word_commit(
		quotient_reference, remainder_reference,
		dividend_words, divisor_words, factor,
	)
}

func int_modular_inverse_divide_double_word_commit(
	quotient_reference Int_Right_Reference,
	remainder_reference Int_Destination_Reference,
	dividend Double_Words, divisor Double_Words,
	factor Int_Division_Quotient_Word,
) {
	Int_Right_Reference_Invariants(
		quotient_reference, "int_modular_inverse_divide_double_word_commit.quotient",
	)
	Int_Destination_Reference_Invariants(
		remainder_reference, "int_modular_inverse_divide_double_word_commit.remainder",
	)
	Double_Words_Invariants(dividend, "int_modular_inverse_divide_double_word_commit.dividend")
	Double_Words_Invariants(divisor, "int_modular_inverse_divide_double_word_commit.divisor")
	Int_Division_Quotient_Word_Invariants(
		factor, "int_modular_inverse_divide_double_word_commit.factor",
	)
	quotient := (*Int)((*Int_Right)(quotient_reference))
	remainder := (*Int)((*Int_Destination)(remainder_reference))
	left, right := uint64(factor), uint64(divisor.Low)
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
	product_high := uint64(factor)*uint64(divisor.High) +
		product_low_high
	minuend := uint64(dividend.Low)
	difference_low := minuend - product_low
	borrow := ((^minuend & product_low) |
		(^(minuend ^ product_low) & difference_low)) >> WORD_BIT_INDEX_MAXIMUM
	difference_high := uint64(dividend.High) -
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
	quotient Int_Division_Nonzero_Word, coefficient_references Int_References,
) {
	Int_Division_Nonzero_Word_Invariants(
		quotient, "int_modular_inverse_coefficient_double_word.quotient",
	)
	Int_References_Invariants(
		coefficient_references, "int_modular_inverse_coefficient_double_word.coefficient",
	)
	destination := (*Int)((*Int_Destination)(coefficient_references.Destination))
	dividend := (*Int)((*Int_Left)(coefficient_references.Left))
	coefficient := (*Int)((*Int_Right)(coefficient_references.Right))
	factor := uint64(quotient)
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

func int_binomial_divide(
	accumulator Positive_Magnitude_Handle, divisor Int_Division_Nonzero_Word,
) {
	Positive_Magnitude_Handle_Invariants(accumulator, "int_binomial_divide.accumulator")
	Int_Division_Nonzero_Word_Invariants(divisor, "int_binomial_divide.divisor")
	denominator := uint64(divisor)
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
	aver.Always(
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
	value Decimal_Text_Digit_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), SIGN_BYTE_COUNT_MAXIMUM, DECIMAL_TEXT_DIGIT_COUNT_MAXIMUM,
		).
		Ensure()
}

// Float_Reference prevents a second invariant chain for one already checked Float.
type Float_Reference *Float

// Float_Reference_Invariants rejects an absent checked Float.
func Float_Reference_Invariants(value Float_Reference, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Float_Invariants(*value, namespace)
}

// Float_Storage_Reference carries one writable Float slot without asserting old scratch state.
type Float_Storage_Reference *Float

// Float_Storage_Reference_Invariants rejects absent writable Float storage.
func Float_Storage_Reference_Invariants(
	value Float_Storage_Reference, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Invariants(*value, namespace)
}

// Float_Square_Root_Workspace_Reference carries already-validated restoring storage.
type Float_Square_Root_Workspace_Reference *Float_Square_Root_Workspace

// Float_Square_Root_Workspace_Reference_Invariants rejects absent restoring storage.
func Float_Square_Root_Workspace_Reference_Invariants(
	value Float_Square_Root_Workspace_Reference, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Square_Root_Workspace_Invariants(*value, namespace)
}

// Float_Square_Root_Pair_Reference carries one already-validated radix-four digit.
type Float_Square_Root_Pair_Reference *Float_Square_Root_Pair

// Float_Square_Root_Pair_Reference_Invariants rejects an absent radix-four digit.
func Float_Square_Root_Pair_Reference_Invariants(
	value Float_Square_Root_Pair_Reference, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Square_Root_Pair_Invariants(*value, namespace)
}

// Float_Square_Root_Count_Reference carries one already-validated active word count.
type Float_Square_Root_Count_Reference *Float_Square_Root_Active_Word_Count

// Float_Square_Root_Count_Reference_Invariants rejects an absent active word count.
func Float_Square_Root_Count_Reference_Invariants(
	value Float_Square_Root_Count_Reference, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Square_Root_Active_Word_Count_Invariants(*value, namespace)
}

// Float_Gob_Encoding_Reference preserves validated bounds across the finite fast path.
type Float_Gob_Encoding_Reference *Float_Gob_Encoding

// Float_Gob_Encoding_Reference_Invariants rejects absent checked encoding storage.
func Float_Gob_Encoding_Reference_Invariants(
	value Float_Gob_Encoding_Reference, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Gob_Encoding_Invariants(*value, namespace)
}

func float_gob_encode_exact_double_word(
	destination_reference Float_Gob_Encoding_Reference, value_reference Float_Reference,
) (encoded Boolean) {
	defer func() {
		Boolean_Invariants(encoded, "float_gob_encode_exact_double_word.encoded")
	}()
	Float_Gob_Encoding_Reference_Invariants(
		destination_reference, "float_gob_encode_exact_double_word.destination",
	)
	Float_Reference_Invariants(value_reference, "float_gob_encode_exact_double_word.value")
	destination := *destination_reference
	value := (*Float)(value_reference)
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
	value Int_Handle, workspace Int_Text_Workspace_Handle,
) (digit_count Decimal_Text_Digit_Count) {
	defer func() {
		Decimal_Text_Digit_Count_Invariants(
			digit_count, "int_text_decimal_digits.digit_count",
		)
	}()
	Int_Handle_Invariants(value, "int_text_decimal_digits.value")
	Int_Text_Workspace_Handle_Invariants(workspace, "int_text_decimal_digits.workspace")
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
	destination_reference Float_Storage_Reference, source_reference Float_Reference,
) {
	Float_Storage_Reference_Invariants(
		destination_reference, "float_text_source_set.destination",
	)
	Float_Reference_Invariants(source_reference, "float_text_source_set.source")
	destination := (*Float)(destination_reference)
	source := (*Float)(source_reference)
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
	workspace Float_Text_Workspace_Handle, value_reference Float_Reference,
	format Float_Text_Format,
	precision Float_Text_Precision,
) (encoded Boolean) {
	defer func() {
		Boolean_Invariants(encoded, "float_text_general_integer.encoded")
	}()
	Float_Text_Workspace_Handle_Invariants(workspace, "float_text_general_integer.workspace")
	Float_Reference_Invariants(value_reference, "float_text_general_integer.value")
	Float_Text_Format_Invariants(format, "float_text_general_integer.format")
	Float_Text_Precision_Invariants(precision, "float_text_general_integer.precision")
	value := (*Float)(value_reference)
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
	text := &workspace.Text.Significand
	digit_count := int(int_text_decimal_digits(mantissa, text))
	if digit_count-WORD_COUNT_INCREMENT < int(precision) {
		return false
	}
	index := WORD_COUNT_MINIMUM
	if value.Negative == POLARITY_NEGATIVE {
		workspace.Output[index] = '-'
		index++
	}
	workspace.Control.Output_Count = Float_Text_Count(index)
	decimal := &workspace.Decimals.Value
	decimal.Control = Float_Text_Decimal_Control{}
	decimal.Control.Exponent = Float_Text_Decimal_Exponent(digit_count)
	low_zero_count := WORD_COUNT_MINIMUM
	for low_zero_count < digit_count {
		if text.Digits[low_zero_count] != '0' {
			break
		}
		low_zero_count++
	}
	significant_count := digit_count - low_zero_count
	decimal.Control.Count = Float_Text_Decimal_Count(significant_count)
	completed_count := WORD_COUNT_MINIMUM
	for completed_count < significant_count {
		remaining_count := digit_count - completed_count
		source_index := remaining_count - WORD_COUNT_INCREMENT
		decimal.Digits[completed_count] = text.Digits[source_index]
		completed_count++
	}
	float_text_decimal_general(
		workspace.Output, workspace.Control, decimal, &workspace.Integers.Exponent,
		(*Int_Text_Workspace)(&workspace.Text.Exponent), false,
	)
	return true
}

// Float_Text_Into writes one complete bounded stdlib representation transactionally.
func Float_Text_Into(
	destination Text, value Float_Handle, format_unvalidated Float_Text_Format_Unvalidated,
	precision_unvalidated Float_Text_Precision_Unvalidated,
	workspace Float_Text_Workspace_Handle,
) (count Float_Text_Count, status Float_Text_Status) {
	defer func() {
		Float_Text_Count_Invariants(count, "float_text_into.count")
		Float_Text_Status_Invariants(status, "float_text_into.status")
	}()
	Text_Invariants(destination, "float_text_into.destination")
	Float_Handle_Invariants(value, "float_text_into.value")
	Float_Text_Format_Unvalidated_Invariants(format_unvalidated, "float_text_into.format")
	Float_Text_Precision_Unvalidated_Invariants(
		precision_unvalidated, "float_text_into.precision",
	)
	Float_Text_Workspace_Handle_Invariants(workspace, "float_text_into.workspace")
	format, validation := Float_Text_Format_Validate(format_unvalidated)
	if validation != Validation_Status(STATUS_OK) {
		return 0, STATUS_INPUT_INVALID
	}
	precision, validation := Float_Text_Precision_Validate(precision_unvalidated)
	if validation != Validation_Status(STATUS_OK) {
		return 0, STATUS_INPUT_INVALID
	}
	workspace.Control.Precision = Float_Text_Digits(precision)
	workspace.Control.Format = format
	value_reference := Float_Reference(value)
	if !float_text_general_integer(workspace, value_reference, format, precision) {
		destination_reference := Float_Storage_Reference(
			(*Float)(&workspace.Values.Source),
		)
		float_text_source_set(destination_reference, value_reference)
		if format == FLOAT_TEXT_KIND_BINARY {
			float_text_binary(
				workspace.Output, workspace.Control, workspace.Values.Source,
				&workspace.Integers, workspace.Text,
			)
		}
		if format == FLOAT_TEXT_KIND_HEXADECIMAL_FRACTION {
			float_text_hexadecimal_fraction(
				workspace.Output, workspace.Control, workspace.Values.Source,
				&workspace.Integers, workspace.Text,
			)
		}
		if format == FLOAT_TEXT_KIND_HEXADECIMAL {
			float_text_hexadecimal(
				workspace.Output, workspace.Control, workspace.Values.Source,
				&workspace.Values.Rounded, &workspace.Integers, workspace.Text,
			)
		}
		if format >= FLOAT_TEXT_KIND_DECIMAL_EXPONENT {
			float_text_decimal(workspace)
		}
	}
	count = Float_Text_Count(workspace.Control.Output_Count)
	if len(destination) < int(count) {
		return count, STATUS_DESTINATION_TOO_SMALL
	}
	copy(destination[:count], workspace.Output[:count])
	return count, STATUS_OK
}

func float_square_root_digit_one(
	workspace_reference Float_Square_Root_Workspace_Reference,
	pair_reference Float_Square_Root_Pair_Reference,
) {
	Float_Square_Root_Workspace_Reference_Invariants(
		workspace_reference, "float_square_root_digit_one.workspace",
	)
	Float_Square_Root_Pair_Reference_Invariants(
		pair_reference, "float_square_root_digit_one.pair",
	)
	workspace := (*Float_Square_Root_Workspace)(workspace_reference)
	pair := (*Float_Square_Root_Pair)(pair_reference)
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
	workspace_reference Float_Square_Root_Workspace_Reference,
	pair_reference Float_Square_Root_Pair_Reference,
) {
	Float_Square_Root_Workspace_Reference_Invariants(
		workspace_reference, "float_square_root_digit_two.workspace",
	)
	Float_Square_Root_Pair_Reference_Invariants(
		pair_reference, "float_square_root_digit_two.pair",
	)
	workspace := (*Float_Square_Root_Workspace)(workspace_reference)
	pair := (*Float_Square_Root_Pair)(pair_reference)
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
	workspace_reference Float_Square_Root_Workspace_Reference,
	pair_reference Float_Square_Root_Pair_Reference,
) {
	Float_Square_Root_Workspace_Reference_Invariants(
		workspace_reference, "float_square_root_digit_three.workspace",
	)
	Float_Square_Root_Pair_Reference_Invariants(
		pair_reference, "float_square_root_digit_three.pair",
	)
	workspace := (*Float_Square_Root_Workspace)(workspace_reference)
	pair := (*Float_Square_Root_Pair)(pair_reference)
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
	workspace_reference Float_Square_Root_Workspace_Reference,
	count_reference Float_Square_Root_Count_Reference,
	pair_reference Float_Square_Root_Pair_Reference,
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
	workspace := (*Float_Square_Root_Workspace)(workspace_reference)
	active_count := int(*count_reference)
	pair := (*Float_Square_Root_Pair)(pair_reference)
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

// Normalized_Word excludes leading zero bits when fixed-width arithmetic relies on alignment.
type Normalized_Word Word

// Normalized_Word_Invariants requires the same leading-bit bound as normalized division.
func Normalized_Word_Invariants(value Normalized_Word, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(
			uint64(value), INT_DIVISION_NORMALIZED_WORD_MINIMUM, bits.WORD_64_MAXIMUM,
		).
		Ensure()
}

func float_square_root_triple_word_one(
	workspace_reference Float_Square_Root_Workspace_Reference,
	high_word Normalized_Word, low_word Word,
) {
	Float_Square_Root_Workspace_Reference_Invariants(
		workspace_reference, "float_square_root_triple_word_one.workspace",
	)
	Normalized_Word_Invariants(high_word, "float_square_root_triple_word_one.high_word")
	Word_Invariants(low_word, "float_square_root_triple_word_one.low_word")
	workspace := (*Float_Square_Root_Workspace)(workspace_reference)
	root_word := Word(BIT_CLEAR)
	remainder_word := Word(BIT_CLEAR)
	candidate_word := Word(BIT_CLEAR)
	remaining_count := WORD_BIT_COUNT
	for remaining_count > BIT_COUNT_MINIMUM {
		remaining_count -= FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT
		pair := Word(high_word) >> uint(remaining_count) &
			Word(FLOAT_SQUARE_ROOT_PAIR_MAXIMUM)
		remainder_word = remainder_word<<FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT | pair
		candidate_word = root_word<<FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT | Word(BIT_SET)
		if remainder_word >= candidate_word {
			remainder_word -= candidate_word
			root_word = root_word<<WORD_COUNT_INCREMENT | Word(BIT_SET)
		} else {
			root_word <<= WORD_COUNT_INCREMENT
		}
	}
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
	workspace_reference Float_Square_Root_Workspace_Reference,
	low_word Word,
) {
	Float_Square_Root_Workspace_Reference_Invariants(
		workspace_reference, "float_square_root_triple_word_two.workspace",
	)
	Word_Invariants(low_word, "float_square_root_triple_word_two.low_word")
	workspace := (*Float_Square_Root_Workspace)(workspace_reference)
	root_low := workspace.Root[WORD_COUNT_MINIMUM]
	root_high := Word(BIT_CLEAR)
	remainder_low := workspace.Remainder[WORD_COUNT_MINIMUM]
	remainder_high := Word(BIT_CLEAR)
	candidate_low := Word(BIT_CLEAR)
	candidate_high := Word(BIT_CLEAR)
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
	workspace_reference Float_Square_Root_Workspace_Reference,
) {
	Float_Square_Root_Workspace_Reference_Invariants(
		workspace_reference, "float_square_root_triple_word_three.workspace",
	)
	workspace := (*Float_Square_Root_Workspace)(workspace_reference)
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
	workspace_reference Float_Square_Root_Workspace_Reference,
	high_word Normalized_Word, low_word Word,
) {
	Float_Square_Root_Workspace_Reference_Invariants(
		workspace_reference, "float_square_root_triple_word.workspace",
	)
	Normalized_Word_Invariants(high_word, "float_square_root_triple_word.high_word")
	Word_Invariants(low_word, "float_square_root_triple_word.low_word")
	float_square_root_triple_word_one(workspace_reference, high_word, low_word)
	float_square_root_triple_word_two(workspace_reference, low_word)
	float_square_root_triple_word_three(workspace_reference)
	active_count := Float_Square_Root_Active_Word_Count(BASE_BINARY * BASE_BINARY)
	pair := Float_Square_Root_Pair(BIT_CLEAR)
	count_reference := Float_Square_Root_Count_Reference(&active_count)
	pair_reference := Float_Square_Root_Pair_Reference(&pair)
	final_count := FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT + FLOAT_SQUARE_ROOT_GUARD_BIT_COUNT
	for completed_count := BIT_COUNT_MINIMUM; completed_count < final_count; completed_count++ {
		float_square_root_digit_many(
			workspace_reference, count_reference, pair_reference,
		)
	}
}

func float_square_root_triple_word_match(
	source Float_Nonnegative_Finite_Handle, workspace Float_Square_Root_Workspace_Handle,
	precision Float_Active_Precision,
) (matched Boolean) {
	defer func() {
		Boolean_Invariants(matched, "float_square_root_triple_word_match.matched")
	}()
	Float_Nonnegative_Finite_Handle_Invariants(
		source, "float_square_root_triple_word_match.source",
	)
	Float_Square_Root_Workspace_Handle_Invariants(
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
	workspace_reference := Float_Square_Root_Workspace_Reference(workspace)
	float_square_root_triple_word(
		workspace_reference, Normalized_Word(high_word),
		source.Mantissa.Words[WORD_COUNT_MINIMUM],
	)
	return true
}

// Square_Root_Quotient retains source divided by current approximation.
type Square_Root_Quotient Int

// Square_Root_Quotient_Invariants preserves normalized nonnegative Newton state.
func Square_Root_Quotient_Invariants(value Square_Root_Quotient, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Newton quotient retains complete division output storage.")
	aver.Always(value.Negative == POLARITY_NONNEGATIVE,
		"Newton quotient has no negative magnitude.")
	aver.Always(int(value.Count) <= len(value.Words),
		"Newton quotient retains active magnitude in caller storage.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Newton quotient omits high zero words.")
}

// Square_Root_Approximation retains next Newton average outside current state.
type Square_Root_Approximation Int

// Square_Root_Approximation_Invariants preserves normalized nonnegative averages.
func Square_Root_Approximation_Invariants(
	value Square_Root_Approximation, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Newton average retains complete addition output storage.")
	aver.Always(value.Negative == POLARITY_NONNEGATIVE,
		"Newton average has no negative magnitude.")
	aver.Always(int(value.Count) <= len(value.Words),
		"Newton average retains active magnitude in caller storage.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Newton average omits high zero words.")
}

// Float_Text_Magnitude_Words retains midpoint precision plus one guard bit.
type Float_Text_Magnitude_Words []Word

// Float_Text_Magnitude_Words_Invariants fixes complete midpoint capacity.
func Float_Text_Magnitude_Words_Invariants(value Float_Text_Magnitude_Words, _ aver.Namespace) {
	aver.Always(len(value) == FLOAT_TEXT_MAGNITUDE_WORD_COUNT_MAXIMUM,
		"Float text midpoint retains one guard bit.")
}

// Float_Text_Magnitude_Count bounds midpoint traversal independently of capacity.
type Float_Text_Magnitude_Count int

// Float_Text_Magnitude_Count_Invariants includes empty midpoint initialization.
func Float_Text_Magnitude_Count_Invariants(
	value Float_Text_Magnitude_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), WORD_COUNT_MINIMUM, FLOAT_TEXT_MAGNITUDE_WORD_COUNT_MAXIMUM).
		Ensure()
}

// Float_Text_Magnitude keeps shortest-rounding midpoint outside caller value.
type Float_Text_Magnitude struct {
	// Words reserves guard precision beyond public magnitude capacity.
	Words Float_Text_Magnitude_Words
	// Count excludes unused storage from midpoint arithmetic.
	Count Float_Text_Magnitude_Count
}

// Float_Text_Magnitude_Invariants composes bounded midpoint storage and traversal.
func Float_Text_Magnitude_Invariants(value Float_Text_Magnitude, namespace aver.Namespace) {
	Float_Text_Magnitude_Words_Invariants(value.Words, namespace)
	Float_Text_Magnitude_Count_Invariants(value.Count, namespace)
}

// Float_Text_Magnitude_Handle preserves midpoint count updates in caller storage.
type Float_Text_Magnitude_Handle *Float_Text_Magnitude

// Float_Text_Magnitude_Handle_Invariants validates selected midpoint storage.
func Float_Text_Magnitude_Handle_Invariants(
	value Float_Text_Magnitude_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Text_Magnitude_Invariants(*value, namespace)
}

// Rat_Parse_Denominator_Memory isolates second component parsing from numerator scratch.
type Rat_Parse_Denominator_Memory Int_Parse_Workspace

// Rat_Parse_Denominator_Memory_Invariants fixes denominator parse capacity independently.
func Rat_Parse_Denominator_Memory_Invariants(
	value Rat_Parse_Denominator_Memory, _ aver.Namespace,
) {
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Rational denominator parsing retains complete magnitude scratch.")
}

// Rat_Parse_Denominator isolates denominator sign and magnitude before normalization.
type Rat_Parse_Denominator Int

// Rat_Parse_Denominator_Invariants preserves complete normalized component state.
func Rat_Parse_Denominator_Invariants(value Rat_Parse_Denominator, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Enum_Uint8(
			uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE),
		).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Parsed denominator reserves full capacity for exact scaling.")
	aver.Always(int(value.Count) <= len(value.Words),
		"Parsed denominator retains active magnitude in caller storage.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Parsed denominator omits high zero words.")
	aver.Always(int(value.Negative) <= int(value.Count),
		"Parsed denominator gives zero no negative twin.")
}

// Rat_Parse_Fraction_Integers_Handle preserves caller component ownership during scaling.
type Rat_Parse_Fraction_Integers_Handle *Rat_Parse_Fraction_Integers

// Rat_Parse_Fraction_Integers_Handle_Invariants composes present component storage.
func Rat_Parse_Fraction_Integers_Handle_Invariants(
	value Rat_Parse_Fraction_Integers_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Rat_Parse_Fraction_Integers_Invariants(*value, namespace)
}

// Float_Text_Exponent_Handle preserves exponent metadata alongside caller words.
type Float_Text_Exponent_Handle *Float_Text_Exponent

// Float_Text_Exponent_Handle_Invariants composes present exponent storage.
func Float_Text_Exponent_Handle_Invariants(
	value Float_Text_Exponent_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Text_Exponent_Invariants(*value, namespace)
}

// Float_Text_Exponent keeps signed machine exponent outside significand storage.
type Float_Text_Exponent Int

// Float_Text_Exponent_Invariants bounds scalar exponent magnitude and storage.
func Float_Text_Exponent_Invariants(value Float_Text_Exponent, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_INCREMENT).
		Range_Int(len(value.Words), WORD_COUNT_INCREMENT, WORD_COUNT_MAXIMUM).
		Enum_Uint8(
			uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE),
		).
		Ensure()
	aver.Always(int(value.Count) <= len(value.Words),
		"Float text exponent retains active magnitude in caller storage.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Float text exponent omits high zero words.")
	aver.Always(int(value.Negative) <= int(value.Count),
		"Float text exponent gives zero no negative twin.")
}

// Float_Text_Integers separates significand and exponent conversion scratch.
type Float_Text_Integers struct {
	// Significand preserves magnitude through destructive digit conversion.
	Significand Int
	// Exponent keeps signed power independent of significand formatting.
	Exponent Float_Text_Exponent
}

// Float_Text_Integers_Invariants composes independent conversion state.
func Float_Text_Integers_Invariants(value Float_Text_Integers, namespace aver.Namespace) {
	Int_Invariants(value.Significand, namespace)
	Float_Text_Exponent_Invariants(value.Exponent, namespace)
}

func float_text_infinity(
	output Float_Text_Output, count Float_Text_Count_Handle, negative Polarity,
) {
	Float_Text_Output_Invariants(output, "float_text_infinity.output")
	Float_Text_Count_Handle_Invariants(count, "float_text_infinity.count")
	Polarity_Invariants(negative, "float_text_infinity.negative")
	index := bytes.SLICE_SIZE_MINIMUM
	if negative == POLARITY_NEGATIVE {
		output[index] = '-'
	} else {
		output[index] = '+'
	}
	index++
	output[index] = 'I'
	output[index+WORD_COUNT_INCREMENT] = 'n'
	output[index+BASE_BINARY] = 'f'
	*count =
		Float_Text_Count(index + BASE_BINARY + WORD_COUNT_INCREMENT)
}

func float_text_sign(
	output Float_Text_Output, count Float_Text_Count_Handle, negative Polarity,
) {
	Float_Text_Output_Invariants(output, "float_text_sign.output")
	Float_Text_Count_Handle_Invariants(count, "float_text_sign.count")
	Polarity_Invariants(negative, "float_text_sign.negative")
	*count = bytes.SLICE_SIZE_MINIMUM
	if negative == POLARITY_NEGATIVE {
		output[WORD_COUNT_MINIMUM] = '-'
		*count = SIGN_BYTE_COUNT_MAXIMUM
	}
}

// Float_Text_Count_Handle lets writers update length without unrelated formatting state.
type Float_Text_Count_Handle *Float_Text_Count

// Float_Text_Count_Handle_Invariants composes present output length storage.
func Float_Text_Count_Handle_Invariants(value Float_Text_Count_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Float_Text_Count_Invariants(*value, namespace)
}

// Float_Text_Rounded_Handle preserves rounding metadata in caller storage.
type Float_Text_Rounded_Handle *Float_Text_Rounded

// Float_Text_Rounded_Handle_Invariants composes present rounding storage.
func Float_Text_Rounded_Handle_Invariants(
	value Float_Text_Rounded_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Text_Rounded_Invariants(*value, namespace)
}

// Float_Text_Integers_Handle preserves scratch metadata across formatting calls.
type Float_Text_Integers_Handle *Float_Text_Integers

// Float_Text_Integers_Handle_Invariants composes present conversion storage.
func Float_Text_Integers_Handle_Invariants(
	value Float_Text_Integers_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Float_Text_Integers_Invariants(*value, namespace)
}

// Float_Text_Exponent_Memory prevents exponent formatting from consuming significand digits.
type Float_Text_Exponent_Memory Int_Text_Workspace

// Float_Text_Exponent_Memory_Invariants preserves complete independent conversion storage.
func Float_Text_Exponent_Memory_Invariants(value Float_Text_Exponent_Memory, _ aver.Namespace) {
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Float exponent text retains complete magnitude scratch.")
	aver.Always(len(value.Digits) == INT_TEXT_SIZE_MAXIMUM,
		"Float exponent text retains complete digit scratch.")
}

// Float_Text_Integer_Workspaces preserves independent significand and exponent output.
type Float_Text_Integer_Workspaces struct {
	// Significand retains digits while exponent formatting runs.
	Significand Int_Text_Workspace
	// Exponent prevents exponent conversion from overwriting significand digits.
	Exponent Float_Text_Exponent_Memory
}

// Float_Text_Integer_Workspaces_Invariants composes independent conversion stores.
func Float_Text_Integer_Workspaces_Invariants(
	value Float_Text_Integer_Workspaces, namespace aver.Namespace,
) {
	Int_Text_Workspace_Invariants(value.Significand, namespace)
	Float_Text_Exponent_Memory_Invariants(value.Exponent, namespace)
}

// Rat_Text_Denominator_Memory isolates denominator formatting from numerator digits.
type Rat_Text_Denominator_Memory Int_Text_Workspace

// Rat_Text_Denominator_Memory_Invariants preserves independent component storage.
func Rat_Text_Denominator_Memory_Invariants(value Rat_Text_Denominator_Memory, _ aver.Namespace) {
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Rational denominator text retains complete magnitude scratch.")
	aver.Always(len(value.Digits) == INT_TEXT_SIZE_MAXIMUM,
		"Rational denominator text retains complete digit scratch.")
}

// Rat_Float_Text_Fraction_Memory isolates fractional digit conversion from whole digits.
type Rat_Float_Text_Fraction_Memory Int_Text_Workspace

// Rat_Float_Text_Fraction_Memory_Invariants preserves independent fractional conversion storage.
func Rat_Float_Text_Fraction_Memory_Invariants(
	value Rat_Float_Text_Fraction_Memory, _ aver.Namespace,
) {
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Rational fractional text retains complete magnitude scratch.")
	aver.Always(len(value.Digits) == INT_TEXT_SIZE_MAXIMUM,
		"Rational fractional text retains complete digit scratch.")
}

// Active_Double_Word_Magnitude_Handle preserves in-place Euclidean reduction.
type Active_Double_Word_Magnitude_Handle *Active_Double_Word_Magnitude

// Active_Double_Word_Magnitude_Handle_Invariants composes present magnitude state.
func Active_Double_Word_Magnitude_Handle_Invariants(
	value Active_Double_Word_Magnitude_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Active_Double_Word_Magnitude_Invariants(*value, namespace)
}

// Double_Word_High keeps high magnitude bits independent of low-word invariant identity.
type Double_Word_High Word

// Double_Word_High_Invariants admits every high-word bit pattern.
func Double_Word_High_Invariants(value Double_Word_High, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Modular_Divisor_Next preserves second limb independently of low limb checks.
type Modular_Divisor_Next uint32

// Modular_Divisor_Next_Invariants admits every second-limb bit pattern.
func Modular_Divisor_Next_Invariants(value Modular_Divisor_Next, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, bits.WORD_32_MAXIMUM).
		Ensure()
}

// Modular_Divisor_Third preserves quotient-correction limb independently.
type Modular_Divisor_Third uint32

// Modular_Divisor_Third_Invariants admits every correction-limb bit pattern.
func Modular_Divisor_Third_Invariants(value Modular_Divisor_Third, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, bits.WORD_32_MAXIMUM).
		Ensure()
}

// MODULAR_DIVISOR_HIGH_MINIMUM requires normalized highest bit.
const MODULAR_DIVISOR_HIGH_MINIMUM = bits.WORD_32_MAXIMUM/BASE_BINARY + WORD_COUNT_INCREMENT

// Modular_Divisor_High excludes unnormalized quotient-estimation divisors.
type Modular_Divisor_High uint32

// Modular_Divisor_High_Invariants requires highest divisor bit set.
func Modular_Divisor_High_Invariants(value Modular_Divisor_High, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint32(
			uint32(value), MODULAR_DIVISOR_HIGH_MINIMUM,
			bits.WORD_32_MAXIMUM,
		).
		Ensure()
}

// WORD_PRODUCT_HIGH_MAXIMUM follows from (2^64-1)^2 = (2^64-2)*2^64+1.
const WORD_PRODUCT_HIGH_MAXIMUM = bits.WORD_64_MAXIMUM - WORD_COUNT_INCREMENT

// Word_Product_High excludes the unreachable all-ones high half of multiplication.
type Word_Product_High Word

// Word_Product_High_Invariants bounds an exact product before carry accumulation.
func Word_Product_High_Invariants(value Word_Product_High, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, WORD_PRODUCT_HIGH_MAXIMUM).
		Ensure()
}

// Double_Word_Product preserves both halves of one machine-word product.
type Double_Word_Product struct {
	// Low retains bits below carry boundary.
	Low Word
	// High retains bits above carry boundary.
	High Word_Product_High
}

// Double_Word_Product_Invariants composes independent low and high word domains.
func Double_Word_Product_Invariants(value Double_Word_Product, namespace aver.Namespace) {
	Word_Invariants(value.Low, namespace)
	Word_Product_High_Invariants(value.High, namespace)
}

// Double_Word_Cross_Left isolates low-by-high product assertion identity.
type Double_Word_Cross_Left Double_Word_Product

// Double_Word_Cross_Left_Invariants admits every complete cross product.
func Double_Word_Cross_Left_Invariants(value Double_Word_Cross_Left, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Low), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.High), bits.WORD_64_MINIMUM, WORD_PRODUCT_HIGH_MAXIMUM).
		Ensure()
}

// Double_Word_Cross_Right isolates high-by-low product assertion identity.
type Double_Word_Cross_Right Double_Word_Product

// Double_Word_Cross_Right_Invariants admits every complete cross product.
func Double_Word_Cross_Right_Invariants(value Double_Word_Cross_Right, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Low), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.High), bits.WORD_64_MINIMUM, WORD_PRODUCT_HIGH_MAXIMUM).
		Ensure()
}

// Double_Word_Product_High isolates high-by-high product assertion identity.
type Double_Word_Product_High Double_Word_Product

// Double_Word_Product_High_Invariants admits every complete high-word product.
func Double_Word_Product_High_Invariants(value Double_Word_Product_High, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value.Low), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Range_Uint64(uint64(value.High), bits.WORD_64_MINIMUM, WORD_PRODUCT_HIGH_MAXIMUM).
		Ensure()
}

// Jacobi_Denominator isolates rotating denominator state.
type Jacobi_Denominator Int

// Jacobi_Denominator_Invariants preserves normalized signed magnitude.
func Jacobi_Denominator_Invariants(value Jacobi_Denominator, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Enum_Uint8(
			uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE),
		).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Jacobi denominator reserves full capacity for copied operands.")
	aver.Always(int(value.Count) <= len(value.Words),
		"Jacobi denominator retains active magnitude in caller storage.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Jacobi denominator omits high zero words.")
	aver.Always(int(value.Negative) <= int(value.Count),
		"Jacobi denominator gives zero no negative twin.")
}

// Jacobi_Odd isolates shifted numerator state.
type Jacobi_Odd Int

// Jacobi_Odd_Invariants preserves normalized nonnegative reduction state.
func Jacobi_Odd_Invariants(value Jacobi_Odd, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Jacobi odd numerator reserves full capacity for shifted operands.")
	aver.Always(value.Negative == POLARITY_NONNEGATIVE,
		"Jacobi odd numerator has nonnegative magnitude.")
	aver.Always(int(value.Count) <= len(value.Words),
		"Jacobi odd numerator retains active magnitude in caller storage.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Jacobi odd numerator omits high zero words.")
}

// Jacobi_Quotient isolates discarded signed division output.
type Jacobi_Quotient Int

// Jacobi_Quotient_Invariants preserves normalized signed quotient storage.
func Jacobi_Quotient_Invariants(value Jacobi_Quotient, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Enum_Uint8(
			uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE),
		).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Jacobi quotient reserves full capacity for division output.")
	aver.Always(int(value.Count) <= len(value.Words),
		"Jacobi quotient retains active magnitude in caller storage.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Jacobi quotient omits high zero words.")
	aver.Always(int(value.Negative) <= int(value.Count),
		"Jacobi quotient gives zero no negative twin.")
}

// Jacobi_Integers_Handle preserves rotating state across nested arithmetic.
type Jacobi_Integers_Handle *Jacobi_Integers

// Jacobi_Integers_Handle_Invariants composes present Jacobi state.
func Jacobi_Integers_Handle_Invariants(value Jacobi_Integers_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Jacobi_Integers_Invariants(*value, namespace)
}

// Jacobi_Numerator separates Jacobi state from nested primality integer storage.
type Jacobi_Numerator Int

// Jacobi_Numerator_Invariants preserves normalized signed numerator storage.
func Jacobi_Numerator_Invariants(value Jacobi_Numerator, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Enum_Uint8(
			uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE),
		).
		Ensure()
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Jacobi numerator reserves full capacity for copied operands.")
	aver.Always(int(value.Count) <= len(value.Words),
		"Jacobi numerator retains active magnitude in caller storage.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Jacobi numerator omits high zero words.")
	aver.Always(int(value.Negative) <= int(value.Count),
		"Jacobi numerator gives zero no negative twin.")
}

// Jacobi_Unit records the terminal coprime denominator.
type Jacobi_Unit Jacobi_Denominator

// Jacobi_Unit_Invariants excludes denominator states that cannot yield a nonzero symbol.
func Jacobi_Unit_Invariants(value Jacobi_Unit, _ aver.Namespace) {
	aver.Always(value.Count == WORD_COUNT_INCREMENT,
		"Coprime Jacobi termination leaves a one-word denominator.")
	aver.Always(value.Negative == POLARITY_NONNEGATIVE,
		"Coprime Jacobi termination leaves a positive denominator.")
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Terminal Jacobi denominator retains complete caller storage.")
	aver.Always(value.Words[WORD_COUNT_MINIMUM] == Word(WORD_COUNT_INCREMENT),
		"Coprime Jacobi termination leaves denominator one.")
}

// Jacobi_Empty records the shifted temporary after operand rotation.
type Jacobi_Empty Jacobi_Odd

// Jacobi_Empty_Invariants preserves the cleared temporary at coprime termination.
func Jacobi_Empty_Invariants(value Jacobi_Empty, _ aver.Namespace) {
	aver.Always(value.Count == WORD_COUNT_MINIMUM,
		"Coprime Jacobi termination follows clearing the shifted temporary.")
	aver.Always(value.Negative == POLARITY_NONNEGATIVE,
		"Cleared Jacobi temporary carries no negative sign.")
	aver.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Cleared Jacobi temporary retains complete caller storage.")
}

// Coprime_Jacobi_Integers excludes early unit-modulus and zero-symbol exits.
type Coprime_Jacobi_Integers Jacobi_Integers

// Coprime_Jacobi_Integers_Invariants binds the successful nontrivial Jacobi phase.
func Coprime_Jacobi_Integers_Invariants(
	value Coprime_Jacobi_Integers, namespace aver.Namespace,
) {
	Positive_Magnitude_Invariants(Positive_Magnitude(value.Numerator), namespace)
	Jacobi_Unit_Invariants(Jacobi_Unit(value.Denominator), namespace)
	Jacobi_Empty_Invariants(Jacobi_Empty(value.Odd), namespace)
	Jacobi_Quotient_Invariants(value.Quotient, namespace)
}

// Validated_Square_Root_Workspace retains successful nonresidue validation state.
type Validated_Square_Root_Workspace Int_Modular_Square_Root_Workspace

// Validated_Square_Root_Workspace_Invariants keeps post-validation bounds separate.
func Validated_Square_Root_Workspace_Invariants(
	value Validated_Square_Root_Workspace, namespace aver.Namespace,
) {
	aver.Always(len(value.Integers) == MODULAR_SQUARE_ROOT_INTEGER_COUNT,
		"Validated modular square-root state retains every integer slot.")
	Modular_Square_Root_Modular_Memory_Invariants(value.Modular, namespace)
	Coprime_Jacobi_Integers_Invariants(Coprime_Jacobi_Integers(value.Jacobi), namespace)
}

// Validated_Square_Root_Workspace_Handle preserves caller-owned post-validation scratch.
type Validated_Square_Root_Workspace_Handle *Validated_Square_Root_Workspace

// Validated_Square_Root_Workspace_Handle_Invariants checks present phase state.
func Validated_Square_Root_Workspace_Handle_Invariants(
	value Validated_Square_Root_Workspace_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Validated_Square_Root_Workspace_Invariants(*value, namespace)
}

// Rat_Denominator preserves implicit one without allocating zero-value storage.
type Rat_Denominator Int

// Rat_Denominator_Invariants bounds nonnegative normalized rational component storage.
func Rat_Denominator_Invariants(value Rat_Denominator, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_MINIMUM, RAT_WORD_COUNT_MAXIMUM).
		Range_Int(len(value.Words), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Ensure()
	aver.Always(value.Negative == POLARITY_NONNEGATIVE,
		"Stored rational denominator is nonnegative.")
	aver.Always(int(value.Count) <= len(value.Words),
		"Stored rational denominator retains active magnitude in caller storage.")
	minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	aver.Always(int_high_word(value.Words[:value.Count]) >= minimum,
		"Stored rational denominator omits high zero words.")
}

func int_references_rotate(references Int_References) (rotated Int_References) {
	defer func() { Int_References_Invariants(rotated, "int_references_rotate.rotated") }()
	Int_References_Invariants(references, "int_references_rotate.references")
	rotated.Destination = Int_Destination_Reference(
		(*Int_Destination)((*Int)((*Int_Left)(references.Left))),
	)
	rotated.Left = Int_Left_Reference((*Int_Left)((*Int)((*Int_Right)(references.Right))))
	rotated.Right = Int_Right_Reference(
		(*Int_Right)((*Int)((*Int_Destination)(references.Destination))),
	)
	return rotated
}

func int_modular_multiply_binary(integers Modular_Integers) {
	Modular_Integers_Invariants(integers, "int_modular_multiply_binary.integers")
	multiplier := &integers[MODULAR_MULTIPLICATION_VALUE_INDEX]
	bit_count := int(Int_Bit_Count(multiplier))
	for bit_index := BIT_COUNT_MINIMUM; bit_index < bit_count; bit_index++ {
		word_index := bit_index / WORD_BIT_COUNT
		word_shift := uint(bit_index) % WORD_BIT_COUNT
		bit := multiplier.Words[word_index] >> word_shift
		if bit&Word(bits.CARRY_MAXIMUM) != 0 {
			int_modular_add(integers, MODULAR_ADDITION_ACCUMULATE)
		}
		if bit_index+WORD_COUNT_INCREMENT < bit_count {
			int_modular_add(integers, MODULAR_ADDITION_DOUBLE)
		}
	}
}

func int_exponent_binary(
	exponent Int_Handle, workspace Int_Exponent_Workspace_Handle,
) (status Arithmetic_Status) {
	defer func() { Arithmetic_Status_Invariants(status, "int_exponent_binary.status") }()
	Int_Handle_Invariants(exponent, "int_exponent_binary.exponent")
	Int_Exponent_Workspace_Handle_Invariants(workspace, "int_exponent_binary.workspace")
	result := &workspace.Integers.Result
	factor := (*Int)(&workspace.Integers.Factor)
	bit_count := int(Int_Bit_Count(exponent))
	multiplication := (*Int_Multiplication_Workspace)(&workspace.Multiplication)
	for bit_index := BIT_COUNT_MINIMUM; bit_index < bit_count; bit_index++ {
		word_index := bit_index / WORD_BIT_COUNT
		word_shift := uint(bit_index) % WORD_BIT_COUNT
		bit := exponent.Words[word_index] >> word_shift
		if bit&Word(bits.CARRY_MAXIMUM) != 0 {
			if int_exponent_accumulate(
				Int_References{
					Destination: Int_Destination_Reference(
						(*Int_Destination)(result),
					),
					Left:  Int_Left_Reference((*Int_Left)(result)),
					Right: Int_Right_Reference((*Int_Right)(factor)),
				}, multiplication,
			) {
				return STATUS_VALUE_OVERFLOW
			}
		}
		if bit_index+WORD_COUNT_INCREMENT < bit_count {
			if factor.Count == Word_Count(BASE_BINARY) {
				int_square_double_word(factor)
			} else {
				multiply_status := Int_Multiply(
					factor, factor, factor, multiplication,
				)
				if multiply_status != Arithmetic_Status(STATUS_OK) {
					return multiply_status
				}
			}
		}
	}
	return STATUS_OK
}
