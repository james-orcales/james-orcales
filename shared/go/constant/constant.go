// Package constant folds the value of a Go constant expression exactly. It exists because a
// linter judges a bound, a capacity, and an enum member by the value the source states, and an
// exact value is a ratio and never a float. Every value is fixed storage the caller holds.
package constant

import (
	"local/james-orcales/shared/math/integer"
	"local/james-orcales/shared/math/rational"
	"local/james-orcales/shared/sim/aver/default"
)

// KIND_UNKNOWN names a value the grammar could not fold.
const KIND_UNKNOWN Kind = 0

// KIND_BOOLEAN names a truth.
const KIND_BOOLEAN Kind = 1

// KIND_TEXT names a string value.
const KIND_TEXT Kind = 2

// KIND_INT names a whole number.
const KIND_INT Kind = 3

// KIND_FLOAT names a number that is no whole one.
const KIND_FLOAT Kind = 4

// KIND_COMPLEX names a pair of numbers.
const KIND_COMPLEX Kind = 5

// KIND_MINIMUM is the unknown kind, the smallest kind.
const KIND_MINIMUM = uint8(KIND_UNKNOWN)

// KIND_MAXIMUM is the complex kind, the largest kind.
const KIND_MAXIMUM = uint8(KIND_COMPLEX)

// UNARY_PLUS leaves a number as it stands.
const UNARY_PLUS Unary = 0

// UNARY_MINUS reverses the sign of a number.
const UNARY_MINUS Unary = 1

// UNARY_COMPLEMENT flips every bit of a whole number.
const UNARY_COMPLEMENT Unary = 2

// UNARY_NOT reverses a truth.
const UNARY_NOT Unary = 3

// UNARY_MINIMUM is the first unary operation.
const UNARY_MINIMUM = uint8(UNARY_PLUS)

// UNARY_MAXIMUM is the final unary operation.
const UNARY_MAXIMUM = uint8(UNARY_NOT)

// BINARY_ADD sums two values.
const BINARY_ADD Binary = 0

// BINARY_SUBTRACT takes one value from another.
const BINARY_SUBTRACT Binary = 1

// BINARY_MULTIPLY forms the product of two values.
const BINARY_MULTIPLY Binary = 2

// BINARY_QUOTIENT divides one value by another.
const BINARY_QUOTIENT Binary = 3

// BINARY_REMAINDER reads what a whole division leaves.
const BINARY_REMAINDER Binary = 4

// BINARY_AND meets the bits of two whole numbers.
const BINARY_AND Binary = 5

// BINARY_OR joins the bits of two whole numbers.
const BINARY_OR Binary = 6

// BINARY_EXCLUSIVE_OR keeps the bits that stand in one whole number alone.
const BINARY_EXCLUSIVE_OR Binary = 7

// BINARY_AND_NOT clears the bits of one whole number that stand in another.
const BINARY_AND_NOT Binary = 8

// BINARY_MINIMUM is the first binary operation.
const BINARY_MINIMUM = uint8(BINARY_ADD)

// BINARY_MAXIMUM is the final binary operation.
const BINARY_MAXIMUM = uint8(BINARY_AND_NOT)

// SHIFT_UP moves the bits of a whole number toward the sign.
const SHIFT_UP Shift_Operation = 0

// SHIFT_DOWN moves the bits of a whole number away from the sign.
const SHIFT_DOWN Shift_Operation = 1

// TEXT_SIZE_MINIMUM admits the empty string value.
const TEXT_SIZE_MINIMUM = 0

// TEXT_SIZE_MAXIMUM caps a string value at the source a scanner admits.
const TEXT_SIZE_MAXIMUM = 1048576

// FORM_SIZE_MINIMUM admits no output.
const FORM_SIZE_MINIMUM = 0

// FORM_SIZE_MAXIMUM holds the longest written form. A string value is the widest thing a value
// carries, thus the form of any other kind fits wherever a string value fits.
const FORM_SIZE_MAXIMUM = TEXT_SIZE_MAXIMUM

// NUMBER_FORM_SIZE_MAXIMUM holds the longest number form: two ratios, the sign between them, the
// mark that names the second one, and the parentheses around the pair.
const NUMBER_FORM_SIZE_MAXIMUM = rational.TEXT_SIZE_MAXIMUM*2 + 6

// IMAGINARY_MARK is the byte a literal ends in to name the part that stands off the number line.
const IMAGINARY_MARK = 'i'

// KIND_SLOT_COUNT is the slot count the kind of one value spends.
const KIND_SLOT_COUNT = 1

// KIND_SLOT is where a value holds its kind.
const KIND_SLOT = 0

// TEXT_SLOT_COUNT is the slot count the text of one value spends.
const TEXT_SLOT_COUNT = 1

// TEXT_SLOT is where a value holds the source its text views.
const TEXT_SLOT = 0

// WORD_SIZE_NONE admits no word, which is what storage too small for one reads back.
const WORD_SIZE_NONE Word_Count = 0

// WORD_SIZE_TRUE is the byte count the word of a true value spans.
const WORD_SIZE_TRUE Word_Count = 4

// WORD_SIZE_FALSE is the byte count the word of a false value spans.
const WORD_SIZE_FALSE Word_Count = 5

// Kind names what a value holds.
type Kind uint8

// Kind_Invariants states every kind a value can wear.
func Kind_Invariants(value Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), KIND_MINIMUM, KIND_MAXIMUM).
		Ensure()
}

// Unary names an operation over one value.
type Unary uint8

// Unary_Invariants states every operation over one value.
func Unary_Invariants(value Unary, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(uint8(value), uint8(UNARY_PLUS), uint8(UNARY_MINUS),
			uint8(UNARY_COMPLEMENT), uint8(UNARY_NOT)).
		Ensure()
}

// Binary names an operation over two values.
type Binary uint8

// Binary_Invariants states every operation over two values.
func Binary_Invariants(value Binary, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), BINARY_MINIMUM, BINARY_MAXIMUM).
		Ensure()
}

// Shift_Operation names the direction one shift moves.
type Shift_Operation uint8

// Shift_Operation_Invariants states the two directions a shift moves.
func Shift_Operation_Invariants(value Shift_Operation, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(SHIFT_UP), uint8(SHIFT_DOWN)).
		Ensure()
}

// Boolean is a true or false report about one value.
type Boolean bool

// Boolean_Invariants states both value reports as obligations.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The constant report is true.").
		Ensure()
}

// Text is a string value, a view of the source that states it.
type Text string

// Text_Invariants caps a string value at the source a scanner admits.
func Text_Invariants(value Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), TEXT_SIZE_MINIMUM, TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Form is caller storage one value writes itself into.
type Form []byte

// Form_Invariants states the storage the longest written form needs.
func Form_Invariants(value Form, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), FORM_SIZE_MINIMUM, FORM_SIZE_MAXIMUM).
		Ensure()
}

// Form_Count is the byte count one written form spans.
type Form_Count int

// Form_Count_Invariants states the byte count of the longest written form.
func Form_Count_Invariants(value Form_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), FORM_SIZE_MINIMUM, FORM_SIZE_MAXIMUM).
		Ensure()
}

// NUMBER_ADD names the sum of two numbers.
const NUMBER_ADD = Number_Binary(BINARY_ADD)

// NUMBER_SUBTRACT names the difference of two numbers.
const NUMBER_SUBTRACT = Number_Binary(BINARY_SUBTRACT)

// NUMBER_MULTIPLY names the product of two numbers.
const NUMBER_MULTIPLY = Number_Binary(BINARY_MULTIPLY)

// NUMBER_QUOTIENT names the quotient of two numbers.
const NUMBER_QUOTIENT = Number_Binary(BINARY_QUOTIENT)

// Whole_Binary names an operation two whole numbers alone admit.
type Whole_Binary uint8

// Whole_Binary_Invariants states every operation two whole numbers alone admit.
func Whole_Binary_Invariants(value Whole_Binary, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(BINARY_REMAINDER), uint8(BINARY_AND_NOT)).
		Ensure()
}

// Number_Binary names an arithmetic operation two numbers admit.
type Number_Binary uint8

// Number_Binary_Invariants states every arithmetic operation two numbers admit.
func Number_Binary_Invariants(value Number_Binary, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(uint8(value), uint8(BINARY_ADD), uint8(BINARY_SUBTRACT),
			uint8(BINARY_MULTIPLY), uint8(BINARY_QUOTIENT)).
		Ensure()
}

// Sum_Binary names an operation that folds two pairs one part at a time.
type Sum_Binary uint8

// Sum_Binary_Invariants states both operations that fold a pair part by part.
func Sum_Binary_Invariants(value Sum_Binary, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(BINARY_ADD), uint8(BINARY_SUBTRACT)).
		Ensure()
}

// Word_Count is the byte count the word of a truth spans.
type Word_Count int

// Word_Count_Invariants states every size the word of a truth spans.
func Word_Count_Invariants(value Word_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Int(int(value), int(WORD_SIZE_NONE), int(WORD_SIZE_TRUE),
			int(WORD_SIZE_FALSE)).
		Ensure()
}

// Real is the part of a number that stands on the number line.
type Real rational.Rational

// Real_Invariants states the one sign bit a real part carries.
func Real_Invariants(value Real, namespace aver.Namespace) {
	aver.Always(
		value.Numerator.Limbs[integer.LIMB_INDEX_MAXIMUM]>>integer.SIGN_BIT_INDEX <= 1,
		"A real part carries its sign in the one top bit of its final limb.",
	)
}

// Imaginary is the part of a number that stands off the number line.
type Imaginary rational.Rational

// Imaginary_Invariants states the one sign bit an imaginary part carries.
func Imaginary_Invariants(value Imaginary, namespace aver.Namespace) {
	aver.Always(
		value.Numerator.Limbs[integer.LIMB_INDEX_MAXIMUM]>>integer.SIGN_BIT_INDEX <= 1,
		"An imaginary part carries its sign in the one top bit of its final limb.",
	)
}

// Real_Storage makes numeric union payload a composed field rather than inherited scalar state.
type Real_Storage struct {
	// Value stays wrapped because validated union transport inherits this field.
	Value Real
}

// Real_Storage_Invariants composes number-line payload.
func Real_Storage_Invariants(value Real_Storage, namespace aver.Namespace) {
	Real_Invariants(value.Value, namespace)
}

// Imaginary_Storage makes complex union payload a composed field rather than inherited scalar
// state.
type Imaginary_Storage struct {
	// Value stays wrapped because validated union transport inherits this field.
	Value Imaginary
}

// Imaginary_Storage_Invariants composes off-line payload.
func Imaginary_Storage_Invariants(value Imaginary_Storage, namespace aver.Namespace) {
	Imaginary_Invariants(value.Value, namespace)
}

// Kind_Slot keeps discriminator storage addressable without hiding one member behind a loop.
type Kind_Slot struct {
	// Value stands alone because one discriminator needs no indexed storage.
	Value Kind
}

// Kind_Slot_Invariants gives raw union storage complete kind obligations.
func Kind_Slot_Invariants(value Kind_Slot, namespace aver.Namespace) {
	Kind_Invariants(value.Value, namespace)
}

// Text_Slot keeps borrowed source storage addressable without hiding one member behind a loop.
type Text_Slot struct {
	// Value stands alone because one borrowed source needs no indexed storage.
	Value Text
}

// Text_Slot_Invariants gives raw union storage complete text obligations.
func Text_Slot_Invariants(value Text_Slot, namespace aver.Namespace) {
	Text_Invariants(value.Value, namespace)
}

// Value_Fields keeps raw union fields separate from validated value transport.
type Value_Fields struct {
	// Kinds holds what the value holds. It is a slot rather than a scalar, because a scalar
	// field owes each body that reads a value an observation of every kind, and a body that
	// builds one kind can never make it. Read the slot through kind_of.
	Kinds Kind_Slot
	// Texts holds a value of the text kind, a view of the source that states it. It is a slot
	// for the reason the kind is one. Read the slot through view_of.
	Texts Text_Slot
	// Real holds the number of an int, float, or complex kind. It also holds a truth as one or
	// zero, because a store that carries a number already carries everything a truth needs.
	Real Real_Storage
	// Imaginary holds the second number of a complex kind.
	Imaginary Imaginary_Storage
}

// Value_Fields_Invariants composes raw union storage before active-kind validation.
func Value_Fields_Invariants(value Value_Fields, namespace aver.Namespace) {
	Kind_Slot_Invariants(value.Kinds, namespace)
	Text_Slot_Invariants(value.Texts, namespace)
	Real_Storage_Invariants(value.Real, namespace)
	Imaginary_Storage_Invariants(value.Imaginary, namespace)
}

// Kind_Stored prevents inactive discriminator domains from leaking through every value use.
type Kind_Stored interface{}

// Kind_Stored_Invariants fixes validated discriminator representation.
func Kind_Stored_Invariants(value Kind_Stored, _ aver.Namespace) {
	_, valid := value.(Kind_Slot)
	aver.Always(valid == (value != nil), "Value kind has expected storage type.")
}

// Text_Stored prevents inactive text domains from leaking through every value use.
type Text_Stored interface{}

// Text_Stored_Invariants fixes validated text representation.
func Text_Stored_Invariants(value Text_Stored, _ aver.Namespace) {
	_, valid := value.(Text_Slot)
	aver.Always(valid == (value != nil), "Value text has expected storage type.")
}

// Value is one folded constant. A whole number and a number share their storage, because a whole
// number is a ratio over one and the kind alone tells a caller which it read.
type Value Value_Fields

// Value_Invariants keeps inactive union fields from widening each operation's proof domain.
func Value_Invariants(value Value, namespace aver.Namespace) {
	Kind_Stored_Invariants(Kind_Stored(value.Kinds), namespace)
	Text_Stored_Invariants(Text_Stored(value.Texts), namespace)
	Real_Storage_Invariants(value.Real, namespace)
	Imaginary_Storage_Invariants(value.Imaginary, namespace)
}

// Kind_Of reads the kind one value wears.
func Kind_Of(value Value) (kind Kind) {
	defer func() { Kind_Invariants(kind, "kind_of.kind") }()
	Value_Invariants(value, "kind_of.value")
	return value.Kinds.Value
}

// Reads the source a string value views.
func view_of(value Value) (text Text) {
	defer func() { Text_Invariants(text, "view_of.text") }()
	Value_Invariants(value, "view_of.value")
	return value.Texts.Value
}

// Make_Unknown builds the value the grammar could not fold. Every operation on it yields another
// unknown, thus one unfolded operand never becomes a wrong answer.
func Make_Unknown() (result Value) {
	defer func() { Value_Invariants(result, "make_unknown.result") }()
	return Value{
		Kinds:     Kind_Slot{Value: KIND_UNKNOWN},
		Texts:     Text_Slot{Value: ""},
		Real:      Real_Storage{Value: Real(rational.Zero())},
		Imaginary: Imaginary_Storage{Value: Imaginary(rational.Zero())},
	}
}

// Make_Boolean builds a truth.
func Make_Boolean(truth Boolean) (result Value) {
	defer func() { Value_Invariants(result, "make_boolean.result") }()
	Boolean_Invariants(truth, "make_boolean.truth")
	result = Make_Unknown()
	result.Kinds.Value = KIND_BOOLEAN
	if bool(truth) {
		result.Real.Value = Real(rational.One())
	}
	return result
}

// Reads the truth a value of the bool kind carries.
func truth_of(value Value) (truth Boolean) {
	defer func() { Boolean_Invariants(truth, "truth_of.truth") }()
	Value_Invariants(value, "truth_of.value")
	return !Boolean(rational.Is_Zero(rational.Rational(value.Real.Value)))
}

// Make_Text builds a string value from a view of the source that states it.
func Make_Text(text Text) (result Value) {
	defer func() { Value_Invariants(result, "make_text.result") }()
	Text_Invariants(text, "make_text.text")
	result = Make_Unknown()
	result.Kinds.Value = KIND_TEXT
	result.Texts.Value = text
	return result
}

// Make_Int_64 builds a whole number from a machine integer.
func Make_Int_64(value Int_64) (result Value) {
	defer func() { Value_Invariants(result, "make_int_64.result") }()
	Int_64_Invariants(value, "make_int_64.value")
	result = Make_Unknown()
	result.Kinds.Value = KIND_INT
	result.Real.Value = Real(rational.From_Integer(whole_of_int_64(value)))
	return result
}

// Lifts a machine integer into a numerator, carrying its sign into every limb above it. The
// result is a numerator rather than an integer, because a numerator root states no limb domain
// and a lifted value pins every upper limb to the fill.
func whole_of_int_64(value Int_64) (whole rational.Numerator) {
	defer func() { rational.Numerator_Invariants(whole, "whole_of_int_64.whole") }()
	Int_64_Invariants(value, "whole_of_int_64.value")
	return rational.Numerator(integer.From_Int_64(integer.Int_64(value)))
}

// Int_64 is a machine integer a whole number lifts from or lowers to.
type Int_64 int64

// Int_64_Invariants states the complete machine integer domain.
func Int_64_Invariants(value Int_64, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int64(int64(value), integer.INT_64_MINIMUM, integer.INT_64_MAXIMUM).
		Ensure()
}

// Make_Ratio builds a number from a numerator and a denominator. A ratio over one names a whole
// number, thus the kind follows the value and never the call.
func Make_Ratio(numerator Int_64, denominator Int_64) (result Value) {
	defer func() { Value_Invariants(result, "make_ratio.result") }()
	Int_64_Invariants(numerator, "make_ratio.numerator")
	Int_64_Invariants(denominator, "make_ratio.denominator")
	ratio, ok := rational.From_Ratio(
		whole_of_int_64(numerator), rational.Denominator(whole_of_int_64(denominator)))
	if !bool(ok) {
		return Make_Unknown()
	}
	return from_rational(ratio)
}

// Builds a value from a ratio and names the kind the ratio wears.
func from_rational(value rational.Rational) (result Value) {
	defer func() { Value_Invariants(result, "from_rational.result") }()
	rational.Rational_Invariants(value, "from_rational.value")
	result = Make_Unknown()
	result.Kinds.Value = KIND_FLOAT
	if bool(rational.Is_Whole(value)) {
		result.Kinds.Value = KIND_INT
	}
	result.Real.Value = Real(value)
	return result
}

// Make_From_Literal reads a Go integer literal and names the number it spells. A literal that
// ends in the imaginary mark folds to a pair whose real part is zero. A literal the grammar
// cannot read folds to an unknown rather than to a wrong value.
func Make_From_Literal(text Text) (result Value) {
	defer func() { Value_Invariants(result, "make_from_literal.result") }()
	Text_Invariants(text, "make_from_literal.text")
	digits := text
	imaginary := Boolean(false)
	if len(text) > 0 {
		if text[len(text)-1] == IMAGINARY_MARK {
			digits = text[:len(text)-1]
			imaginary = true
		}
	}
	if len(digits) > integer.TEXT_SIZE_MAXIMUM {
		return Make_Unknown()
	}
	whole, ok := integer.From_Text(integer.Text(digits))
	if !bool(ok) {
		return Make_Unknown()
	}
	folded := rational.From_Integer(rational.Numerator(whole))
	if !bool(imaginary) {
		return from_rational(folded)
	}
	return make_complex(rational.Zero(), folded)
}

// Make_Complex builds a pair of numbers from a real value and an imaginary value. A part that
// holds no number of its own folds the pair to an unknown.
func Make_Complex(real_part Value, imaginary_part Value) (result Value) {
	defer func() { Value_Invariants(result, "make_complex_value.result") }()
	Value_Invariants(real_part, "make_complex_value.real_part")
	Value_Invariants(imaginary_part, "make_complex_value.imaginary_part")
	if !bool(is_real(Kind_Of(real_part))) {
		return Make_Unknown()
	}
	if !bool(is_real(Kind_Of(imaginary_part))) {
		return Make_Unknown()
	}
	return make_complex(
		rational.Rational(real_part.Real.Value),
		rational.Rational(imaginary_part.Real.Value))
}

// Builds a pair of numbers from two ratios. The kind stays a pair even where the imaginary part
// is zero, because the kind states what the source wrote and never what it happened to fold to.
func make_complex(
	real_part rational.Rational, imaginary_part rational.Rational,
) (result Value) {
	defer func() { Value_Invariants(result, "make_complex.result") }()
	rational.Rational_Invariants(real_part, "make_complex.real_part")
	rational.Rational_Invariants(imaginary_part, "make_complex.imaginary_part")
	result = Make_Unknown()
	result.Kinds.Value = KIND_COMPLEX
	result.Real.Value = Real(real_part)
	result.Imaginary.Value = Imaginary(imaginary_part)
	return result
}

// Read distinguishes available values from rejected conversions.
type Read Boolean

// Read_Invariants states both conversion outcomes.
func Read_Invariants(value Read, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Conversion yields a value.").
		Ensure()
}

// Boolean_Result keeps truth and conversion outcome at one output boundary.
type Boolean_Result struct {
	// Value may be false after either a valid read or refusal.
	Value Boolean
	// OK distinguishes valid false from refused conversion.
	OK Read
}

// Boolean_Result_Invariants composes one truth conversion.
func Boolean_Result_Invariants(value Boolean_Result, namespace aver.Namespace) {
	Boolean_Invariants(value.Value, namespace)
	Read_Invariants(value.OK, namespace)
}

// Text_Result keeps text and conversion outcome at one output boundary.
type Text_Result struct {
	// Value may be empty after either a valid read or refusal.
	Value Text
	// OK distinguishes valid empty text from refused conversion.
	OK Read
}

// Text_Result_Invariants composes one text conversion.
func Text_Result_Invariants(value Text_Result, namespace aver.Namespace) {
	Text_Invariants(value.Value, namespace)
	Read_Invariants(value.OK, namespace)
}

// Int_64_Result keeps integer and conversion outcome at one output boundary.
type Int_64_Result struct {
	// Value may be zero after either a valid read or refusal.
	Value Int_64
	// OK distinguishes valid zero from refused conversion.
	OK Read
}

// Int_64_Result_Invariants composes one integer conversion.
func Int_64_Result_Invariants(value Int_64_Result, namespace aver.Namespace) {
	Int_64_Invariants(value.Value, namespace)
	Read_Invariants(value.OK, namespace)
}

// Boolean_Value reads a truth back and reports whether the kind allowed it.
func Boolean_Value(value Value) (result Boolean_Result) {
	defer func() { Boolean_Result_Invariants(result, "boolean_value.result") }()
	Value_Invariants(value, "boolean_value.value")
	if Kind_Of(value) != KIND_BOOLEAN {
		return Boolean_Result{Value: false, OK: false}
	}
	return Boolean_Result{Value: truth_of(value), OK: true}
}

// Text_Value reads a string value back and reports whether the kind allowed it.
func Text_Value(value Value) (result Text_Result) {
	defer func() { Text_Result_Invariants(result, "text_value.result") }()
	Value_Invariants(value, "text_value.value")
	if Kind_Of(value) != KIND_TEXT {
		return Text_Result{Value: "", OK: false}
	}
	return Text_Result{Value: view_of(value), OK: true}
}

// Int_64_Value reads a whole number back and reports whether the kind and the width allowed it.
func Int_64_Value(value Value) (result Int_64_Result) {
	defer func() { Int_64_Result_Invariants(result, "int_64_value.result") }()
	Value_Invariants(value, "int_64_value.value")
	if Kind_Of(value) != KIND_INT {
		return Int_64_Result{Value: 0, OK: false}
	}
	whole, whole_ok := rational.Whole(rational.Rational(value.Real.Value))
	if !bool(whole_ok) {
		return Int_64_Result{Value: 0, OK: false}
	}
	lowered, fits := integer.To_Int_64(integer.Integer(whole))
	return Int_64_Result{Value: Int_64(lowered), OK: Read(fits)}
}

// Unary_Operation applies one operation to one value. A kind the operation does not admit folds
// to an unknown, thus a caller reads a value only where Go would have one.
func Unary_Operation(operation Unary, value Value) (result Value) {
	defer func() { Value_Invariants(result, "unary_operation.result") }()
	Unary_Invariants(operation, "unary_operation.operation")
	Value_Invariants(value, "unary_operation.value")
	switch operation {
	case UNARY_NOT:
		if Kind_Of(value) != KIND_BOOLEAN {
			return Make_Unknown()
		}
		return Make_Boolean(!truth_of(value))
	case UNARY_PLUS:
		if is_number(Kind_Of(value)) {
			return value
		}
		return Make_Unknown()
	case UNARY_COMPLEMENT:
		if Kind_Of(value) != KIND_INT {
			return Make_Unknown()
		}
		whole, ok := rational.Whole(rational.Rational(value.Real.Value))
		if !bool(ok) {
			return Make_Unknown()
		}
		return from_rational(rational.From_Integer(
			rational.Numerator(integer.Not(integer.Integer(whole)))))
	}
	if !bool(is_number(Kind_Of(value))) {
		return Make_Unknown()
	}
	flipped, ok := rational.Negate(rational.Rational(value.Real.Value))
	if !bool(ok) {
		return Make_Unknown()
	}
	if Kind_Of(value) != KIND_COMPLEX {
		return from_rational(flipped)
	}
	turned, turned_ok := rational.Negate(rational.Rational(value.Imaginary.Value))
	if !bool(turned_ok) {
		return Make_Unknown()
	}
	return make_complex(flipped, turned)
}

// Reports whether a kind holds a number, a pair of numbers included.
func is_number(kind Kind) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_number.yes") }()
	Kind_Invariants(kind, "is_number.kind")
	if bool(is_real(kind)) {
		return true
	}
	return kind == KIND_COMPLEX
}

// Reports whether a kind holds a number that stands on the number line alone.
func is_real(kind Kind) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "is_real.yes") }()
	Kind_Invariants(kind, "is_real.kind")
	if kind == KIND_INT {
		return true
	}
	return kind == KIND_FLOAT
}

// Binary_Operation applies one operation to two values. A pair of kinds the operation does not
// admit folds to an unknown.
func Binary_Operation(left Value, operation Binary, right Value) (result Value) {
	defer func() { Value_Invariants(result, "binary_operation.result") }()
	Value_Invariants(left, "binary_operation.left")
	Binary_Invariants(operation, "binary_operation.operation")
	Value_Invariants(right, "binary_operation.right")
	switch operation {
	case BINARY_AND, BINARY_OR, BINARY_EXCLUSIVE_OR, BINARY_AND_NOT, BINARY_REMAINDER:
		return whole_operation(left, Whole_Binary(operation), right)
	}
	if operation == BINARY_ADD {
		if Kind_Of(left) == KIND_TEXT {
			return text_operation(left, right)
		}
	}
	return number_operation(left, Number_Binary(operation), right)
}

// Joins two string values. The result views no source of its own, thus it folds to an unknown
// and a caller that wants the joined text builds it from the two views it already holds.
func text_operation(left Value, right Value) (result Value) {
	defer func() { Value_Invariants(result, "text_operation.result") }()
	Value_Invariants(left, "text_operation.left")
	Value_Invariants(right, "text_operation.right")
	if Kind_Of(right) != KIND_TEXT {
		return Make_Unknown()
	}
	if len(view_of(left)) == 0 {
		return right
	}
	if len(view_of(right)) == 0 {
		return left
	}
	return Make_Unknown()
}

// Applies one bit or remainder operation to two whole numbers.
func whole_operation(left Value, operation Whole_Binary, right Value) (result Value) {
	defer func() { Value_Invariants(result, "whole_operation.result") }()
	Value_Invariants(left, "whole_operation.left")
	Whole_Binary_Invariants(operation, "whole_operation.operation")
	Value_Invariants(right, "whole_operation.right")
	if Kind_Of(left) != KIND_INT {
		return Make_Unknown()
	}
	if Kind_Of(right) != KIND_INT {
		return Make_Unknown()
	}
	above, above_ok := rational.Whole(rational.Rational(left.Real.Value))
	below, below_ok := rational.Whole(rational.Rational(right.Real.Value))
	if !bool(above_ok) {
		return Make_Unknown()
	}
	if !bool(below_ok) {
		return Make_Unknown()
	}
	first := integer.Integer(above)
	second := integer.Integer(below)
	switch operation {
	case Whole_Binary(BINARY_AND):
		return whole_value(rational.Numerator(integer.And(first, second)))
	case Whole_Binary(BINARY_OR):
		return whole_value(rational.Numerator(integer.Or(first, second)))
	case Whole_Binary(BINARY_EXCLUSIVE_OR):
		return whole_value(rational.Numerator(integer.Exclusive_Or(first, second)))
	case Whole_Binary(BINARY_AND_NOT):
		return whole_value(rational.Numerator(integer.And_Not(first, second)))
	}
	_, rest, divided := integer.Divide(first, second)
	if !bool(divided) {
		return Make_Unknown()
	}
	return whole_value(rational.Numerator(rest))
}

// Builds a whole number value from a numerator. A numerator rather than an integer, because an
// integer root states every limb domain and a folded whole number reaches few of its edges.
func whole_value(value rational.Numerator) (result Value) {
	defer func() { Value_Invariants(result, "whole_value.result") }()
	rational.Numerator_Invariants(value, "whole_value.value")
	return from_rational(rational.From_Integer(value))
}

// Applies one arithmetic operation to two numbers. A quotient of two whole numbers stays whole,
// which is what Go states for an untyped constant division of integers.
func number_operation(left Value, operation Number_Binary, right Value) (result Value) {
	defer func() { Value_Invariants(result, "number_operation.result") }()
	Value_Invariants(left, "number_operation.left")
	Number_Binary_Invariants(operation, "number_operation.operation")
	Value_Invariants(right, "number_operation.right")
	if !bool(is_number(Kind_Of(left))) {
		return Make_Unknown()
	}
	if !bool(is_number(Kind_Of(right))) {
		return Make_Unknown()
	}
	if Kind_Of(left) == KIND_COMPLEX {
		return complex_operation(left, operation, right)
	}
	if Kind_Of(right) == KIND_COMPLEX {
		return complex_operation(left, operation, right)
	}
	first := rational.Rational(left.Real.Value)
	second := rational.Rational(right.Real.Value)
	whole := Kind_Of(left) == KIND_INT
	if Kind_Of(right) != KIND_INT {
		whole = false
	}
	folded := apply_ratio(first, operation, second)
	if !bool(folded.OK) {
		return Make_Unknown()
	}
	if operation != Number_Binary(BINARY_QUOTIENT) {
		return from_rational(folded.Value)
	}
	if !whole {
		return from_rational(folded.Value)
	}
	truncated, truncated_ok := rational.Whole(folded.Value)
	if !bool(truncated_ok) {
		return Make_Unknown()
	}
	return whole_value(truncated)
}

// Ratio_Result keeps exact value and bounded arithmetic outcome together.
type Ratio_Result struct {
	// Value remains zero when arithmetic exceeds fixed storage.
	Value rational.Rational
	// OK distinguishes valid zero from bounded refusal.
	OK Read
}

// Ratio_Result_Invariants composes one exact arithmetic outcome.
func Ratio_Result_Invariants(value Ratio_Result, namespace aver.Namespace) {
	rational.Rational_Invariants(value.Value, namespace)
	Read_Invariants(value.OK, namespace)
}

// Applies one arithmetic operation to two ratios.
func apply_ratio(
	left rational.Rational, operation Number_Binary, right rational.Rational,
) (result Ratio_Result) {
	defer func() { Ratio_Result_Invariants(result, "apply_ratio.result") }()
	rational.Rational_Invariants(left, "apply_ratio.left")
	Number_Binary_Invariants(operation, "apply_ratio.operation")
	rational.Rational_Invariants(right, "apply_ratio.right")
	switch operation {
	case Number_Binary(BINARY_ADD):
		folded, held := rational.Add(left, right)
		return Ratio_Result{Value: folded, OK: Read(held)}
	case Number_Binary(BINARY_SUBTRACT):
		folded, held := rational.Subtract(left, right)
		return Ratio_Result{Value: folded, OK: Read(held)}
	case Number_Binary(BINARY_MULTIPLY):
		folded, held := rational.Multiply(left, right)
		return Ratio_Result{Value: folded, OK: Read(held)}
	}
	folded, held := rational.Divide(left, right)
	return Ratio_Result{Value: folded, OK: Read(held)}
}

// Applies one arithmetic operation to two pairs of numbers. A number that stands on the number
// line carries a zero imaginary part, thus one path serves a pair and a plain number alike.
func complex_operation(left Value, operation Number_Binary, right Value) (result Value) {
	defer func() { Value_Invariants(result, "complex_operation.result") }()
	Value_Invariants(left, "complex_operation.left")
	Number_Binary_Invariants(operation, "complex_operation.operation")
	Value_Invariants(right, "complex_operation.right")
	switch operation {
	case Number_Binary(BINARY_ADD), Number_Binary(BINARY_SUBTRACT):
		return add_complex(left, Sum_Binary(operation), right)
	case Number_Binary(BINARY_MULTIPLY):
		return multiply_complex(left, right)
	}
	return divide_complex(left, right)
}

// Adds or subtracts two pairs of numbers, one part at a time.
func add_complex(left Value, operation Sum_Binary, right Value) (result Value) {
	defer func() { Value_Invariants(result, "add_complex.result") }()
	Value_Invariants(left, "add_complex.left")
	Sum_Binary_Invariants(operation, "add_complex.operation")
	Value_Invariants(right, "add_complex.right")
	real_part := apply_ratio(rational.Rational(left.Real.Value),
		Number_Binary(operation), rational.Rational(right.Real.Value))
	if !bool(real_part.OK) {
		return Make_Unknown()
	}
	imaginary_part := apply_ratio(rational.Rational(left.Imaginary.Value),
		Number_Binary(operation), rational.Rational(right.Imaginary.Value))
	if !bool(imaginary_part.OK) {
		return Make_Unknown()
	}
	return make_complex(real_part.Value, imaginary_part.Value)
}

// Multiplies two pairs of numbers. The imaginary parts multiply to a real term of the other sign,
// which is why the real part of the product subtracts.
func multiply_complex(left Value, right Value) (result Value) {
	defer func() { Value_Invariants(result, "multiply_complex.result") }()
	Value_Invariants(left, "multiply_complex.left")
	Value_Invariants(right, "multiply_complex.right")
	across := apply_ratio(
		rational.Rational(left.Real.Value), NUMBER_MULTIPLY,
		rational.Rational(right.Real.Value))
	turned := apply_ratio(
		rational.Rational(left.Imaginary.Value), NUMBER_MULTIPLY,
		rational.Rational(right.Imaginary.Value))
	upward := apply_ratio(
		rational.Rational(left.Imaginary.Value), NUMBER_MULTIPLY,
		rational.Rational(right.Real.Value))
	downward := apply_ratio(
		rational.Rational(left.Real.Value), NUMBER_MULTIPLY,
		rational.Rational(right.Imaginary.Value))
	held := across.OK && turned.OK && upward.OK && downward.OK
	if !bool(held) {
		return Make_Unknown()
	}
	real_part := apply_ratio(across.Value, NUMBER_SUBTRACT, turned.Value)
	imaginary_part := apply_ratio(upward.Value, NUMBER_ADD, downward.Value)
	if !bool(real_part.OK) {
		return Make_Unknown()
	}
	if !bool(imaginary_part.OK) {
		return Make_Unknown()
	}
	return make_complex(real_part.Value, imaginary_part.Value)
}

// Divides two pairs of numbers. A product with the conjugate leaves a real divisor, thus the
// division is two divisions by one ratio.
func divide_complex(left Value, right Value) (result Value) {
	defer func() { Value_Invariants(result, "divide_complex.result") }()
	Value_Invariants(left, "divide_complex.left")
	Value_Invariants(right, "divide_complex.right")
	scale := square_sum(right)
	if !bool(scale.OK) {
		return Make_Unknown()
	}
	product := multiply_complex(left, conjugate(right))
	if Kind_Of(product) != KIND_COMPLEX {
		return Make_Unknown()
	}
	real_part := apply_ratio(
		rational.Rational(product.Real.Value), NUMBER_QUOTIENT, scale.Value)
	imaginary_part := apply_ratio(
		rational.Rational(product.Imaginary.Value), NUMBER_QUOTIENT, scale.Value)
	if !bool(real_part.OK) {
		return Make_Unknown()
	}
	if !bool(imaginary_part.OK) {
		return Make_Unknown()
	}
	return make_complex(real_part.Value, imaginary_part.Value)
}

// Sums the squares of the two parts of a pair, which is the divisor a division by a pair leaves.
func square_sum(value Value) (result Ratio_Result) {
	defer func() { Ratio_Result_Invariants(result, "square_sum.result") }()
	Value_Invariants(value, "square_sum.value")
	real_square := apply_ratio(
		rational.Rational(value.Real.Value), NUMBER_MULTIPLY,
		rational.Rational(value.Real.Value))
	imaginary_square := apply_ratio(
		rational.Rational(value.Imaginary.Value), NUMBER_MULTIPLY,
		rational.Rational(value.Imaginary.Value))
	held := real_square.OK && imaginary_square.OK
	if !bool(held) {
		return Ratio_Result{Value: rational.Zero(), OK: false}
	}
	summed := apply_ratio(real_square.Value, NUMBER_ADD, imaginary_square.Value)
	if !bool(summed.OK) {
		return Ratio_Result{Value: rational.Zero(), OK: false}
	}
	if bool(rational.Is_Zero(summed.Value)) {
		return Ratio_Result{Value: rational.Zero(), OK: false}
	}
	return summed
}

// Reverses the imaginary part of a pair and leaves the real part alone.
func conjugate(value Value) (result Value) {
	defer func() { Value_Invariants(result, "conjugate.result") }()
	Value_Invariants(value, "conjugate.value")
	turned, ok := rational.Negate(rational.Rational(value.Imaginary.Value))
	if !bool(ok) {
		return Make_Unknown()
	}
	return make_complex(rational.Rational(value.Real.Value), turned)
}

// Comparison_Result keeps ordering and comparability at one output boundary.
type Comparison_Result struct {
	// Order remains same when values admit no comparison.
	Order integer.Order
	// OK distinguishes equality from values that admit no comparison.
	OK Read
}

// Comparison_Result_Invariants composes one comparison outcome.
func Comparison_Result_Invariants(value Comparison_Result, namespace aver.Namespace) {
	integer.Order_Invariants(value.Order, namespace)
	Read_Invariants(value.OK, namespace)
}

// Compare reads two values of one kind and reports the order Go states. Two values of different
// kinds compare as no order at all.
func Compare(left Value, right Value) (result Comparison_Result) {
	defer func() { Comparison_Result_Invariants(result, "compare.result") }()
	Value_Invariants(left, "compare.left")
	Value_Invariants(right, "compare.right")
	if bool(is_number(Kind_Of(left))) {
		if bool(is_number(Kind_Of(right))) {
			return compare_number(left, right)
		}
	}
	if Kind_Of(left) != Kind_Of(right) {
		return Comparison_Result{Order: integer.ORDER_SAME, OK: false}
	}
	switch Kind_Of(left) {
	case KIND_BOOLEAN:
		if truth_of(left) == truth_of(right) {
			return Comparison_Result{Order: integer.ORDER_SAME, OK: true}
		}
		if bool(truth_of(left)) {
			return Comparison_Result{Order: integer.ORDER_AFTER, OK: true}
		}
		return Comparison_Result{Order: integer.ORDER_BEFORE, OK: true}
	case KIND_TEXT:
		return Comparison_Result{
			Order: compare_text(view_of(left), view_of(right)), OK: true,
		}
	}
	return Comparison_Result{Order: integer.ORDER_SAME, OK: false}
}

// Reads the order of two numbers. A pair orders by its real part and then by its imaginary part,
// thus two pairs that read the same order hold the same two numbers, which is the one comparison
// Go states for a pair.
func compare_number(left Value, right Value) (result Comparison_Result) {
	defer func() { Comparison_Result_Invariants(result, "compare_number.result") }()
	Value_Invariants(left, "compare_number.left")
	Value_Invariants(right, "compare_number.right")
	folded, held := rational.Compare(
		rational.Rational(left.Real.Value), rational.Rational(right.Real.Value))
	if !bool(held) {
		return Comparison_Result{Order: integer.ORDER_SAME, OK: false}
	}
	if folded != integer.ORDER_SAME {
		return Comparison_Result{Order: folded, OK: true}
	}
	turned, turned_held := rational.Compare(
		rational.Rational(left.Imaginary.Value), rational.Rational(right.Imaginary.Value))
	return Comparison_Result{Order: turned, OK: Read(turned_held)}
}

// Reads the order of two string values, byte by byte.
func compare_text(left Text, right Text) (order integer.Order) {
	defer func() { integer.Order_Invariants(order, "compare_text.order") }()
	Text_Invariants(left, "compare_text.left")
	Text_Invariants(right, "compare_text.right")
	for index := range len(left) {
		if index == len(right) {
			return integer.ORDER_AFTER
		}
		if left[index] == right[index] {
			continue
		}
		if left[index] < right[index] {
			return integer.ORDER_BEFORE
		}
		return integer.ORDER_AFTER
	}
	if len(left) == len(right) {
		return integer.ORDER_SAME
	}
	return integer.ORDER_BEFORE
}

// Shift moves a whole number by a count. A shift of anything else, or one that leaves the width,
// folds to an unknown.
func Shift(
	value Value, operation Shift_Operation, count integer.Shift_Count,
) (result Value) {
	defer func() { Value_Invariants(result, "shift.result") }()
	Value_Invariants(value, "shift.value")
	Shift_Operation_Invariants(operation, "shift.operation")
	integer.Shift_Count_Invariants(count, "shift.count")
	if Kind_Of(value) != KIND_INT {
		return Make_Unknown()
	}
	whole, ok := rational.Whole(rational.Rational(value.Real.Value))
	if !bool(ok) {
		return Make_Unknown()
	}
	if operation == SHIFT_DOWN {
		return whole_value(rational.Numerator(
			integer.Shift_Right(integer.Integer(whole), count)))
	}
	moved, held := integer.Shift_Left(integer.Integer(whole), count)
	if !bool(held) {
		return Make_Unknown()
	}
	return whole_value(rational.Numerator(moved))
}

// Into_Text writes a value into caller storage and returns the byte count. An unknown writes
// nothing, because no form states a value that was never folded.
func Into_Text(destination Form, value Value) (count Form_Count) {
	defer func() { Form_Count_Invariants(count, "into_text.count") }()
	Form_Invariants(destination, "into_text.destination")
	Value_Invariants(value, "into_text.value")
	switch Kind_Of(value) {
	case KIND_UNKNOWN:
		return FORM_SIZE_MINIMUM
	case KIND_BOOLEAN:
		return Form_Count(write_truth(destination, truth_of(value)))
	case KIND_TEXT:
		if len(destination) < len(view_of(value)) {
			return FORM_SIZE_MINIMUM
		}
		return Form_Count(copy(destination, view_of(value)))
	}
	var real_storage [rational.TEXT_SIZE_MAXIMUM]byte
	var imaginary_storage [rational.TEXT_SIZE_MAXIMUM]byte
	above := rational.Into_Text(real_storage[:], rational.Rational(value.Real.Value))
	if above == 0 {
		return FORM_SIZE_MINIMUM
	}
	if Kind_Of(value) != KIND_COMPLEX {
		if len(destination) < int(above) {
			return FORM_SIZE_MINIMUM
		}
		return Form_Count(copy(destination, real_storage[:above]))
	}
	below := rational.Into_Text(
		imaginary_storage[:], rational.Rational(value.Imaginary.Value))
	if below == 0 {
		return FORM_SIZE_MINIMUM
	}
	var storage [NUMBER_FORM_SIZE_MAXIMUM]byte
	written := copy(storage[:], "(")
	written = written + copy(storage[written:], real_storage[:above])
	written = written + copy(storage[written:], " + ")
	written = written + copy(storage[written:], imaginary_storage[:below])
	written = written + copy(storage[written:], "i)")
	if len(destination) < written {
		return FORM_SIZE_MINIMUM
	}
	return Form_Count(copy(destination, storage[:written]))
}

// Writes the word one truth reads as. The count wears a type of its own, because a word is one of
// three sizes and a form of any other size is one a caller never reads back.
func write_truth(destination Form, truth Boolean) (count Word_Count) {
	defer func() { Word_Count_Invariants(count, "write_truth.count") }()
	Form_Invariants(destination, "write_truth.destination")
	Boolean_Invariants(truth, "write_truth.truth")
	word := "false"
	if bool(truth) {
		word = "true"
	}
	if len(destination) < len(word) {
		return WORD_SIZE_NONE
	}
	return Word_Count(copy(destination, word))
}
