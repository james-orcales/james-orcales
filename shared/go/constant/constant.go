// Package constant folds the value of a Go constant expression exactly. It exists because a
// linter judges a bound, a capacity, and an enum member by the value the source states, and an
// exact value is a ratio and never a float. Every value is fixed storage the caller holds.
package constant

import "local/james-orcales/shared/math/integer"
import "local/james-orcales/shared/math/rational"
import "local/james-orcales/shared/sim/aver/default"

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
    Enum_4_Uint8(
    uint8(value),
    uint8(UNARY_PLUS),
    uint8(UNARY_MINUS),
    uint8(UNARY_COMPLEMENT),
    uint8(UNARY_NOT),
  ).
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
    Range_Uint8(
    uint8(value),
    uint8(BINARY_REMAINDER),
    uint8(BINARY_AND_NOT),
  ).
    Ensure()
}

// Number_Binary names an arithmetic operation two numbers admit.
type Number_Binary uint8

// Number_Binary_Invariants states every arithmetic operation two numbers admit.
func Number_Binary_Invariants(value Number_Binary, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_4_Uint8(
    uint8(value),
    uint8(BINARY_ADD),
    uint8(BINARY_SUBTRACT),
    uint8(BINARY_MULTIPLY),
    uint8(BINARY_QUOTIENT),
  ).
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
    Enum_3_Int(
    int(value),
    int(WORD_SIZE_NONE),
    int(WORD_SIZE_TRUE),
    int(WORD_SIZE_FALSE),
  ).
    Ensure()
}

// Real is the part of a number that stands on the number line.
type Real rational.Rational

// Real_Invariants requires positive denominator so stored sign stays in numerator.
func Real_Invariants(value Real, namespace aver.Namespace) {
  aver.Always(
    value.Denominator.Limb_7>>integer.SIGN_BIT_INDEX == 0,
    "A real part denominator has no sign bit.",
  )
  aver.Always(
    integer.Limbs(value.Denominator) != (integer.Limbs{}),
    "A real part denominator is nonzero.",
  )
}

// Imaginary is the part of a number that stands off the number line.
type Imaginary rational.Rational

// Imaginary_Invariants requires positive denominator so stored sign stays in numerator.
func Imaginary_Invariants(value Imaginary, namespace aver.Namespace) {
  aver.Always(
    value.Denominator.Limb_7>>integer.SIGN_BIT_INDEX == 0,
    "An imaginary part denominator has no sign bit.",
  )
  aver.Always(
    integer.Limbs(value.Denominator) != (integer.Limbs{}),
    "An imaginary part denominator is nonzero.",
  )
}

// Real_Storage makes numeric union payload a composed field rather than inherited scalar state.
type Real_Storage struct {
  // Value stays wrapped because validated union transport inherits this field.
  Value Real
}

// Real_Storage_Invariants composes number-line payload.
func Real_Storage_Invariants(value *Real_Storage, namespace aver.Namespace) {
  Real_Invariants(value.Value, namespace)
}

// Imaginary_Storage makes complex union payload a composed field rather than inherited scalar
// state.
type Imaginary_Storage struct {
  // Value stays wrapped because validated union transport inherits this field.
  Value Imaginary
}

// Imaginary_Storage_Invariants composes off-line payload.
func Imaginary_Storage_Invariants(value *Imaginary_Storage, namespace aver.Namespace) {
  Imaginary_Invariants(value.Value, namespace)
}

// Kind_Slot keeps discriminator storage addressable without hiding one member behind a loop.
type Kind_Slot struct {
  // Value stands alone because one discriminator needs no indexed storage.
  Value Kind
}

// Kind_Slot_Invariants gives raw union storage complete kind obligations.
func Kind_Slot_Invariants(value *Kind_Slot, namespace aver.Namespace) {
  Kind_Invariants(value.Value, namespace)
}

// Text_Slot keeps borrowed source storage addressable without hiding one member behind a loop.
type Text_Slot struct {
  // Value stands alone because one borrowed source needs no indexed storage.
  Value Text
}

// Text_Slot_Invariants gives raw union storage complete text obligations.
func Text_Slot_Invariants(value *Text_Slot, namespace aver.Namespace) {
  Text_Invariants(value.Value, namespace)
}

// Value is one folded constant. A whole number and a number share their storage, because a whole
// number is a ratio over one and the kind alone tells a caller which it read.
type Value struct {
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

// Value_Invariants composes raw union storage before active-kind validation.
func Value_Invariants(value *Value, namespace aver.Namespace) {
  Kind_Slot_Invariants(&value.Kinds, namespace)
  Text_Slot_Invariants(&value.Texts, namespace)
  Real_Storage_Invariants(&value.Real, namespace)
  Imaginary_Storage_Invariants(&value.Imaginary, namespace)
}

// Kind_Of reads the kind one value wears.
func Kind_Of(value Value) (kind Kind) {
  defer func() { Kind_Invariants(kind, "kind_of.kind") }()
  Value_Invariants(&value, "kind_of.value")
  return value.Kinds.Value
}

// Reads the source a string value views.
func view_of(value Text_Slot) (text Text) {
  defer func() { Text_Invariants(text, "view_of.text") }()
  Text_Slot_Invariants(&value, "view_of.value")
  return value.Value
}

// Zero_Pair is the pair both numbers of which stand at zero, which is what a value of no number
// carries.
func Zero_Pair() (pair Pair) {
  defer func() { Pair_Invariants(&pair, "zero_pair.pair") }()
  rational.Zero((*rational.Rational)(&pair.Real.Value))
  rational.Zero((*rational.Rational)(&pair.Imaginary.Value))
  return pair
}

// Make builds one folded constant from the kind it wears and the storage it carries. It is the
// one place a value is built, thus every kind and every text stands at one namespace. A builder
// of its own would owe the kinds its own kind can never hold.
func Make(kind Kind, text Text, numbers Pair) (result Value) {
  defer func() { Value_Invariants(&result, "make.result") }()
  Kind_Invariants(kind, "make.kind")
  Text_Invariants(text, "make.text")
  Pair_Invariants(&numbers, "make.numbers")
  result.Kinds.Value = kind
  result.Texts.Value = text
  result.Real = numbers.Real
  result.Imaginary = numbers.Imaginary
  return result
}

// Truth_Pair holds one truth as the number one or the number zero. A store that carries a number
// already carries everything a truth needs, thus a truth spends no storage of its own. The caller
// names the kind at Make, because a builder that named it here would owe every other kind.
func Truth_Pair(truth Boolean) (numbers Pair) {
  defer func() { Pair_Invariants(&numbers, "truth_pair.numbers") }()
  Boolean_Invariants(truth, "truth_pair.truth")
  numbers = Zero_Pair()
  if bool(truth) {
    rational.One((*rational.Rational)(&numbers.Real.Value))
  }
  return numbers
}

// Reads the truth a value of the bool kind carries.
func truth_of(value Pair) (truth Boolean) {
  defer func() { Boolean_Invariants(truth, "truth_of.truth") }()
  Pair_Invariants(&value, "truth_of.value")
  return !Boolean(rational.Is_Zero(rational.Rational(value.Real.Value)))
}

// Int_64_Pair holds one machine integer as a ratio over one. The caller names the kind at Make,
// for the reason Truth_Pair states.
func Int_64_Pair(value Int_64) (numbers Pair) {
  defer func() { Pair_Invariants(&numbers, "int_64_pair.numbers") }()
  Int_64_Invariants(value, "int_64_pair.value")
  numbers = Zero_Pair()
  var whole integer.Integer
  integer.From_Int_64(&whole, integer.Int_64(value))
  rational.From_Integer(
    (*rational.Rational)(&numbers.Real.Value), rational.Numerator(whole.Limbs),
  )
  return numbers
}

// Int_64 is a machine integer a whole number lifts from or lowers to.
type Int_64 int64

// Int_64_Invariants states the complete machine integer domain.
func Int_64_Invariants(value Int_64, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Range_Int64(
    int64(value),
    integer.INT_64_MINIMUM,
    integer.INT_64_MAXIMUM,
  ).
    Ensure()
}

// Make_Ratio builds a number from a numerator and a denominator. A ratio over one names a whole
// number, thus the kind follows the value and never the call.
func Make_Ratio(numerator Int_64, denominator Int_64) (result Ratio_Fold) {
  defer func() { Ratio_Fold_Invariants(&result, "make_ratio.result") }()
  Int_64_Invariants(numerator, "make_ratio.numerator")
  Int_64_Invariants(denominator, "make_ratio.denominator")
  var above, below integer.Integer
  integer.From_Int_64(&above, integer.Int_64(numerator))
  integer.From_Int_64(&below, integer.Int_64(denominator))
  var ratio Real
  ok := rational.From_Ratio(
    (*rational.Rational)(&ratio),
    rational.Numerator(above.Limbs),
    rational.Denominator(below.Limbs),
  )
  if !bool(ok) {
    return Ratio_Fold{Kind: Ratio_Kind(KIND_INT), Numbers: Zero_Pair(), OK: false}
  }
  folded := from_rational(ratio)
  return Ratio_Fold{Kind: folded.Kind, Numbers: folded.Numbers, OK: true}
}

// Ratio_Kind is the kind one ratio wears. A ratio is whole or it is not, and it is never a truth,
// a text, or a pair, thus a step that folds one owes those kinds nothing.
type Ratio_Kind Kind

// Ratio_Kind_Invariants states both kinds a ratio wears.
func Ratio_Kind_Invariants(value Ratio_Kind, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Uint8(uint8(value), uint8(KIND_INT), uint8(KIND_FLOAT)).
    Ensure()
}

// Ratio_Fold is what one ratio folds to: the kind the ratio wears, the number it holds, and
// whether the fold held. Make alone widens it to a value.
type Ratio_Fold struct {
  // Kind rests at the whole kind where the fold did not hold.
  Kind Ratio_Kind
  // Numbers stands at zero where the fold did not hold.
  Numbers Pair
  // OK distinguishes a folded ratio from storage that overran.
  OK Read
}

// Ratio_Fold_Invariants composes one ratio fold.
func Ratio_Fold_Invariants(value *Ratio_Fold, namespace aver.Namespace) {
  Ratio_Kind_Invariants(value.Kind, namespace)
  Pair_Invariants(&value.Numbers, namespace)
  Read_Invariants(value.OK, namespace)
}

// Ratio is one folded ratio: the kind it wears and the number it holds. It carries no outcome,
// because naming the kind of a ratio already in hand cannot refuse.
type Ratio struct {
  // Kind follows the value and never the call.
  Kind Ratio_Kind
  // Numbers holds the ratio in its real part.
  Numbers Pair
}

// Ratio_Invariants composes one folded ratio.
func Ratio_Invariants(value *Ratio, namespace aver.Namespace) {
  Ratio_Kind_Invariants(value.Kind, namespace)
  Pair_Invariants(&value.Numbers, namespace)
}

// Folds one ratio and names the kind the ratio wears.
func from_rational(value Real) (result Ratio) {
  defer func() { Ratio_Invariants(&result, "from_rational.result") }()
  Real_Invariants(value, "from_rational.value")
  kind := Ratio_Kind(KIND_FLOAT)
  if bool(rational.Is_Whole(rational.Rational(value))) {
    kind = Ratio_Kind(KIND_INT)
  }
  numbers := Zero_Pair()
  numbers.Real.Value = value
  return Ratio{Kind: kind, Numbers: numbers}
}

// Literal_Kind is the kind one Go integer literal names. Its digits spell a whole number, and the
// imaginary mark makes a pair of it, thus a literal names no other kind.
type Literal_Kind Kind

// Literal_Kind_Invariants states both kinds one literal names.
func Literal_Kind_Invariants(value Literal_Kind, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_Uint8(uint8(value), uint8(KIND_INT), uint8(KIND_COMPLEX)).
    Ensure()
}

// Literal_Fold is what one Go integer literal folds to.
type Literal_Fold struct {
  // Kind rests at the whole kind where the fold did not hold.
  Kind Literal_Kind
  // Numbers stands at zero where the fold did not hold.
  Numbers Pair
  // OK distinguishes a folded literal from text the grammar refuses.
  OK Read
}

// Literal_Fold_Invariants composes one literal fold.
func Literal_Fold_Invariants(value *Literal_Fold, namespace aver.Namespace) {
  Literal_Kind_Invariants(value.Kind, namespace)
  Pair_Invariants(&value.Numbers, namespace)
  Read_Invariants(value.OK, namespace)
}

// Make_From_Literal reads a Go integer literal and names the number it spells. A literal that
// ends in the imaginary mark folds to a pair whose real part is zero. A literal the grammar
// cannot read folds to an unknown rather than to a wrong value.
func Make_From_Literal(text Text) (result Literal_Fold) {
  defer func() { Literal_Fold_Invariants(&result, "make_from_literal.result") }()
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
    return Literal_Fold{Kind: Literal_Kind(KIND_INT), Numbers: Zero_Pair(), OK: false}
  }
  var whole integer.Integer
  ok := integer.From_Text(&whole, integer.Text(digits))
  if !bool(ok) {
    return Literal_Fold{Kind: Literal_Kind(KIND_INT), Numbers: Zero_Pair(), OK: false}
  }
  var folded Real
  rational.From_Integer((*rational.Rational)(&folded), rational.Numerator(whole.Limbs))
  if !bool(imaginary) {
    numbers := Zero_Pair()
    numbers.Real.Value = folded
    return Literal_Fold{
      Kind: Literal_Kind(KIND_INT), Numbers: numbers, OK: true,
    }
  }
  var zero Real
  rational.Zero((*rational.Rational)(&zero))
  return Literal_Fold{
    Kind:    Literal_Kind(KIND_COMPLEX),
    Numbers: Make_Complex(zero, folded),
    OK:      true,
  }
}

// Make_Complex builds a pair of numbers from two ratios. It names no kind, because the kind
// states what the source wrote and never what the pair happened to fold to, thus the caller names
// it at Make even where the imaginary part is zero.
func Make_Complex(
  real_part Real, imaginary_part Real,
) (result Pair) {
  defer func() { Pair_Invariants(&result, "make_complex.result") }()
  Real_Invariants(real_part, "make_complex.real_part")
  Real_Invariants(imaginary_part, "make_complex.imaginary_part")
  result = Zero_Pair()
  result.Real.Value = real_part
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
func Boolean_Result_Invariants(value *Boolean_Result, namespace aver.Namespace) {
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
func Text_Result_Invariants(value *Text_Result, namespace aver.Namespace) {
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
func Int_64_Result_Invariants(value *Int_64_Result, namespace aver.Namespace) {
  Int_64_Invariants(value.Value, namespace)
  Read_Invariants(value.OK, namespace)
}

// Boolean_Value reads a truth back and reports whether the kind allowed it.
func Boolean_Value(value Value) (result Boolean_Result) {
  defer func() { Boolean_Result_Invariants(&result, "boolean_value.result") }()
  Value_Invariants(&value, "boolean_value.value")
  if Kind_Of(value) != KIND_BOOLEAN {
    return Boolean_Result{Value: false, OK: false}
  }
  return Boolean_Result{Value: truth_of(Pair_Of(value)), OK: true}
}

// Text_Value reads a string value back and reports whether the kind allowed it.
func Text_Value(value Value) (result Text_Result) {
  defer func() { Text_Result_Invariants(&result, "text_value.result") }()
  Value_Invariants(&value, "text_value.value")
  if Kind_Of(value) != KIND_TEXT {
    return Text_Result{Value: "", OK: false}
  }
  return Text_Result{Value: view_of(value.Texts), OK: true}
}

// Int_64_Value reads a whole number back and reports whether the kind and the width allowed it.
func Int_64_Value(value Value) (result Int_64_Result) {
  defer func() { Int_64_Result_Invariants(&result, "int_64_value.result") }()
  Value_Invariants(&value, "int_64_value.value")
  if Kind_Of(value) != KIND_INT {
    return Int_64_Result{Value: 0, OK: false}
  }
  var whole rational.Numerator
  whole_ok := rational.Whole(&whole, rational.Rational(value.Real.Value))
  if !bool(whole_ok) {
    return Int_64_Result{Value: 0, OK: false}
  }
  lowered, fits := integer.To_Int_64(integer.Integer{Limbs: integer.Limbs(whole)})
  return Int_64_Result{Value: Int_64(lowered), OK: Read(fits)}
}

// Unary_Kind is the kind one unary operation names. No unary operation of Go names a text, thus
// the fold of one owes the text kind nothing.
type Unary_Kind Kind

// Unary_Kind_Invariants states every kind one unary operation names.
func Unary_Kind_Invariants(value Unary_Kind, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_4_Uint8(
    uint8(value),
    uint8(KIND_BOOLEAN),
    uint8(KIND_INT),
    uint8(KIND_FLOAT),
    uint8(KIND_COMPLEX),
  ).
    Ensure()
}

// Unary_Fold is what one unary operation folds to. It names no text, thus the caller hands Make
// an empty view.
type Unary_Fold struct {
  // Kind rests at the whole kind where the fold did not hold.
  Kind Unary_Kind
  // Numbers stands at zero where the fold did not hold.
  Numbers Pair
  // OK distinguishes a folded value from a kind the operation does not admit.
  OK Read
}

// Unary_Fold_Invariants composes one unary fold.
func Unary_Fold_Invariants(value *Unary_Fold, namespace aver.Namespace) {
  Unary_Kind_Invariants(value.Kind, namespace)
  Pair_Invariants(&value.Numbers, namespace)
  Read_Invariants(value.OK, namespace)
}

// Unary_Operation applies one operation to one value. A kind the operation does not admit folds
// to no value, thus a caller reads one only where Go would have one.
// Flips every bit of one whole number. The caller proves the value wears the whole kind, and a
// whole number divides by its denominator of one, thus the division here cannot refuse.
func complement(value Real) (result Pair) {
  defer func() { Pair_Invariants(&result, "complement.result") }()
  Real_Invariants(value, "complement.value")
  var whole rational.Numerator
  ok := rational.Whole(&whole, rational.Rational(value))
  aver.Always(bool(ok), "A whole number divides by its denominator.")
  var flipped integer.Integer
  integer.Not(&flipped, integer.Integer{Limbs: integer.Limbs(whole)})
  return whole_value(rational.Numerator(flipped.Limbs))
}

// Unary_Operation applies one operation to one value. A kind the operation does not admit folds
// to no value, thus a caller reads one only where Go would have one.
func Unary_Operation(operation Unary, value Value) (result Unary_Fold) {
  defer func() { Unary_Fold_Invariants(&result, "unary_operation.result") }()
  Unary_Invariants(operation, "unary_operation.operation")
  Value_Invariants(&value, "unary_operation.value")
  // The refusal rests at the whole kind, and stands in one place for the reason
  // Binary_Operation states.
  refused := Unary_Fold{Kind: Unary_Kind(KIND_INT), Numbers: Zero_Pair(), OK: false}
  switch operation {
  case UNARY_NOT:
    if Kind_Of(value) != KIND_BOOLEAN {
      return refused
    }
    return Unary_Fold{
      Kind:    Unary_Kind(KIND_BOOLEAN),
      Numbers: Truth_Pair(!truth_of(Pair_Of(value))),
      OK:      true,
    }
  case UNARY_PLUS:
    if is_number(Kind_Of(value)) {
      return Unary_Fold{
        Kind:    Unary_Kind(Kind_Of(value)),
        Numbers: Pair_Of(value),
        OK:      true,
      }
    }
    return refused
  case UNARY_COMPLEMENT:
    if Kind_Of(value) != KIND_INT {
      return refused
    }
    return Unary_Fold{
      Kind:    Unary_Kind(KIND_INT),
      Numbers: complement(value.Real.Value),
      OK:      true,
    }
  }
  if !bool(is_number(Kind_Of(value))) {
    return refused
  }
  var flipped Real
  ok := rational.Negate((*rational.Rational)(&flipped), rational.Rational(value.Real.Value))
  if !bool(ok) {
    return refused
  }
  if Kind_Of(value) != KIND_COMPLEX {
    ratio := from_rational(flipped)
    return Unary_Fold{
      Kind: Unary_Kind(ratio.Kind), Numbers: ratio.Numbers, OK: true,
    }
  }
  var turned Real
  turned_ok := rational.Negate(
    (*rational.Rational)(&turned),
    rational.Rational(value.Imaginary.Value),
  )
  if !bool(turned_ok) {
    return refused
  }
  return Unary_Fold{
    Kind:    Unary_Kind(KIND_COMPLEX),
    Numbers: Make_Complex(flipped, turned),
    OK:      true,
  }
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
// Binary_Kind is the kind one binary operation names. No binary operation of Go names a truth,
// because an order is what a comparison answers and Compare answers it, thus the fold of one owes
// the truth kind nothing.
type Binary_Kind Kind

// Binary_Kind_Invariants states every kind one binary operation names.
func Binary_Kind_Invariants(value Binary_Kind, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_4_Uint8(
    uint8(value),
    uint8(KIND_TEXT),
    uint8(KIND_INT),
    uint8(KIND_FLOAT),
    uint8(KIND_COMPLEX),
  ).
    Ensure()
}

// Binary_Fold is what one binary operation folds to. A join of two string values keeps a view and
// no numbers, and every other operation keeps numbers and no view.
type Binary_Fold struct {
  // Kind rests at the whole kind where the fold did not hold.
  Kind Binary_Kind
  // View stands empty except where the fold names the text kind.
  View Text
  // Numbers stands at zero where the fold did not hold.
  Numbers Pair
  // OK distinguishes a folded value from a kind the operation does not admit.
  OK Read
}

// Binary_Fold_Invariants composes one binary fold.
func Binary_Fold_Invariants(value *Binary_Fold, namespace aver.Namespace) {
  Binary_Kind_Invariants(value.Kind, namespace)
  Text_Invariants(value.View, namespace)
  Pair_Invariants(&value.Numbers, namespace)
  Read_Invariants(value.OK, namespace)
}

// Binary_Operation applies one operation to two values. A pair of kinds the operation does not
// admit folds to no value, thus a caller reads one only where Go would have one.
func Binary_Operation(
  left Value, operation Binary, right Value,
) (result Binary_Fold) {
  defer func() { Binary_Fold_Invariants(&result, "binary_operation.result") }()
  Value_Invariants(&left, "binary_operation.left")
  Binary_Invariants(operation, "binary_operation.operation")
  Value_Invariants(&right, "binary_operation.right")
  // The refusal rests at the whole kind. It stands in one place, because a body that spells
  // it at each turn hides the three paths this step holds.
  refused := Binary_Fold{
    Kind: Binary_Kind(KIND_INT), View: "", Numbers: Zero_Pair(), OK: false,
  }
  switch operation {
  case BINARY_AND, BINARY_OR, BINARY_EXCLUSIVE_OR, BINARY_AND_NOT, BINARY_REMAINDER:
    // Only two whole numbers admit a bit or a remainder operation, thus the kinds
    // stand here and the step below takes the numbers alone.
    if Kind_Of(left) != KIND_INT {
      return refused
    }
    if Kind_Of(right) != KIND_INT {
      return refused
    }
    folded := whole_operation(
      Pair_Of(left), Whole_Binary(operation), Pair_Of(right),
    )
    if !bool(folded.OK) {
      return refused
    }
    return Binary_Fold{
      Kind: Binary_Kind(KIND_INT), View: "", Numbers: folded.Numbers, OK: true,
    }
  }
  if operation == BINARY_ADD {
    if Kind_Of(left) == KIND_TEXT {
      if Kind_Of(right) != KIND_TEXT {
        return refused
      }
      joined := text_operation(left.Texts, right.Texts)
      if !bool(joined.OK) {
        return refused
      }
      return Binary_Fold{
        Kind:    Binary_Kind(KIND_TEXT),
        View:    joined.View,
        Numbers: Zero_Pair(),
        OK:      true,
      }
    }
  }
  // Only a number admits an arithmetic operation, thus the kinds stand here and the step
  // below takes the numbers and the narrow kind each side wears.
  if !bool(is_number(Kind_Of(left))) {
    return refused
  }
  if !bool(is_number(Kind_Of(right))) {
    return refused
  }
  numbered := number_operation(
    Number_Operand{Kind: Number_Kind(Kind_Of(left)), Numbers: Pair_Of(left)},
    Number_Binary(operation),
    Number_Operand{Kind: Number_Kind(Kind_Of(right)), Numbers: Pair_Of(right)},
  )
  if !bool(numbered.OK) {
    return refused
  }
  return Binary_Fold{
    Kind:    Binary_Kind(numbered.Kind),
    View:    "",
    Numbers: numbered.Numbers,
    OK:      true,
  }
}

// Joins two string values. The result views no source of its own, thus it folds to an unknown
// and a caller that wants the joined text builds it from the two views it already holds.
// Text_Fold is what one join of two string values folds to: the view it kept, and whether it kept
// one. The fold names no kind, thus the caller names it at Make.
type Text_Fold struct {
  // View stands empty where the join kept none.
  View Text
  // OK distinguishes a kept view from a join that needed storage of its own.
  OK Read
}

// Text_Fold_Invariants composes one text join.
func Text_Fold_Invariants(value *Text_Fold, namespace aver.Namespace) {
  Text_Invariants(value.View, namespace)
  Read_Invariants(value.OK, namespace)
}

func text_operation(left Text_Slot, right Text_Slot) (result Text_Fold) {
  defer func() { Text_Fold_Invariants(&result, "text_operation.result") }()
  Text_Slot_Invariants(&left, "text_operation.left")
  Text_Slot_Invariants(&right, "text_operation.right")
  if len(view_of(left)) == 0 {
    return Text_Fold{View: view_of(right), OK: true}
  }
  if len(view_of(right)) == 0 {
    return Text_Fold{View: view_of(left), OK: true}
  }
  return Text_Fold{View: "", OK: false}
}

// Applies one bit or remainder operation to two whole numbers.
func whole_operation(left Pair, operation Whole_Binary, right Pair) (result Pair_Fold) {
  defer func() { Pair_Fold_Invariants(&result, "whole_operation.result") }()
  Pair_Invariants(&left, "whole_operation.left")
  Whole_Binary_Invariants(operation, "whole_operation.operation")
  Pair_Invariants(&right, "whole_operation.right")
  var above rational.Numerator
  above_ok := rational.Whole(&above, rational.Rational(left.Real.Value))
  var below rational.Numerator
  below_ok := rational.Whole(&below, rational.Rational(right.Real.Value))
  if !bool(above_ok) {
    return Pair_Fold{Numbers: Zero_Pair(), OK: false}
  }
  if !bool(below_ok) {
    return Pair_Fold{Numbers: Zero_Pair(), OK: false}
  }
  first := integer.Integer{Limbs: integer.Limbs(above)}
  second := integer.Integer{Limbs: integer.Limbs(below)}
  switch operation {
  case Whole_Binary(BINARY_AND):
    var folded integer.Integer
    integer.And(&folded, first, second)
    return Pair_Fold{Numbers: whole_value(rational.Numerator(folded.Limbs)), OK: true}
  case Whole_Binary(BINARY_OR):
    var folded integer.Integer
    integer.Or(&folded, first, second)
    return Pair_Fold{Numbers: whole_value(rational.Numerator(folded.Limbs)), OK: true}
  case Whole_Binary(BINARY_EXCLUSIVE_OR):
    var folded integer.Integer
    integer.Exclusive_Or(&folded, first, second)
    return Pair_Fold{Numbers: whole_value(rational.Numerator(folded.Limbs)), OK: true}
  case Whole_Binary(BINARY_AND_NOT):
    var folded integer.Integer
    integer.And_Not(&folded, first, second)
    return Pair_Fold{Numbers: whole_value(rational.Numerator(folded.Limbs)), OK: true}
  }
  var quotient, rest integer.Integer
  divided := integer.Divide(&quotient, &rest, first, second)
  if !bool(divided) {
    return Pair_Fold{Numbers: Zero_Pair(), OK: false}
  }
  return Pair_Fold{Numbers: whole_value(rational.Numerator(rest.Limbs)), OK: true}
}

// Whole arithmetic already owns limbs; ratio conversion must preserve every high word.
func whole_value(value rational.Numerator) (result Pair) {
  defer func() { Pair_Invariants(&result, "whole_value.result") }()
  rational.Numerator_Invariants(value, "whole_value.value")
  var folded Real
  rational.From_Integer((*rational.Rational)(&folded), value)
  result = Zero_Pair()
  result.Real.Value = folded
  return result
}

// Applies one arithmetic operation to two numbers. A quotient of two whole numbers stays whole,
// which is what Go states for an untyped constant division of integers.
// Number_Kind is the kind one number wears: a whole number, a number that is no whole one, or a
// pair. A number is never a truth, a text, or an unknown, thus a step over numbers owes those
// kinds nothing.
type Number_Kind Kind

// Number_Kind_Invariants states every kind a number wears.
func Number_Kind_Invariants(value Number_Kind, namespace aver.Namespace) {
  aver.Tree(value, namespace).
    Enum_3_Uint8(
    uint8(value),
    uint8(KIND_INT),
    uint8(KIND_FLOAT),
    uint8(KIND_COMPLEX),
  ).
    Ensure()
}

// Number_Operand is one side of an operation over numbers: the kind it wears and the numbers it
// holds. The caller proves the kind is a number before it builds one.
type Number_Operand struct {
  // Kind decides whether the pair path or the ratio path serves the operation.
  Kind Number_Kind
  // Numbers holds the real part and, for a pair, the imaginary part.
  Numbers Pair
}

// Number_Operand_Invariants composes one operand of an operation over numbers.
func Number_Operand_Invariants(value *Number_Operand, namespace aver.Namespace) {
  Number_Kind_Invariants(value.Kind, namespace)
  Pair_Invariants(&value.Numbers, namespace)
}

// Number_Fold is what one operation over numbers folds to.
type Number_Fold struct {
  // Kind rests at the whole kind where the fold did not hold.
  Kind Number_Kind
  // Numbers stands at zero where the fold did not hold.
  Numbers Pair
  // OK distinguishes a folded number from storage that overran.
  OK Read
}

// Number_Fold_Invariants composes one fold of an operation over numbers.
func Number_Fold_Invariants(value *Number_Fold, namespace aver.Namespace) {
  Number_Kind_Invariants(value.Kind, namespace)
  Pair_Invariants(&value.Numbers, namespace)
  Read_Invariants(value.OK, namespace)
}

func number_operation(
  left Number_Operand,
  operation Number_Binary,
  right Number_Operand,
) (result Number_Fold) {
  defer func() { Number_Fold_Invariants(&result, "number_operation.result") }()
  Number_Operand_Invariants(&left, "number_operation.left")
  Number_Binary_Invariants(operation, "number_operation.operation")
  Number_Operand_Invariants(&right, "number_operation.right")
  // A number that stands on the number line carries a zero imaginary part, thus one pair
  // path serves a pair and a plain number alike once either side wears the pair kind.
  paired := left.Kind == Number_Kind(KIND_COMPLEX)
  if right.Kind == Number_Kind(KIND_COMPLEX) {
    paired = true
  }
  if bool(Boolean(paired)) {
    folded := complex_operation(left.Numbers, operation, right.Numbers)
    if !bool(folded.OK) {
      return Number_Fold{
        Kind: Number_Kind(KIND_INT), Numbers: Zero_Pair(), OK: false,
      }
    }
    return Number_Fold{
      Kind: Number_Kind(KIND_COMPLEX), Numbers: folded.Numbers, OK: true,
    }
  }
  first := left.Numbers.Real.Value
  second := right.Numbers.Real.Value
  whole := left.Kind == Number_Kind(KIND_INT)
  if right.Kind != Number_Kind(KIND_INT) {
    whole = false
  }
  folded := apply_ratio(first, operation, second)
  if !bool(folded.OK) {
    return Number_Fold{
      Kind: Number_Kind(KIND_INT), Numbers: Zero_Pair(), OK: false,
    }
  }
  if operation != Number_Binary(BINARY_QUOTIENT) {
    ratio := from_rational(folded.Value)
    return Number_Fold{
      Kind: Number_Kind(ratio.Kind), Numbers: ratio.Numbers, OK: true,
    }
  }
  if !whole {
    ratio := from_rational(folded.Value)
    return Number_Fold{
      Kind: Number_Kind(ratio.Kind), Numbers: ratio.Numbers, OK: true,
    }
  }
  var truncated rational.Numerator
  truncated_ok := rational.Whole(&truncated, rational.Rational(folded.Value))
  if !bool(truncated_ok) {
    return Number_Fold{
      Kind: Number_Kind(KIND_INT), Numbers: Zero_Pair(), OK: false,
    }
  }
  return Number_Fold{
    Kind: Number_Kind(KIND_INT), Numbers: whole_value(truncated), OK: true,
  }
}

// Ratio_Result keeps exact value and bounded arithmetic outcome together.
type Ratio_Result struct {
  // Value remains zero when arithmetic exceeds fixed storage.
  Value Real
  // OK distinguishes valid zero from bounded refusal.
  OK Read
}

// Ratio_Result_Invariants composes one exact arithmetic outcome.
func Ratio_Result_Invariants(value *Ratio_Result, namespace aver.Namespace) {
  Real_Invariants(value.Value, namespace)
  Read_Invariants(value.OK, namespace)
}

// Applies one arithmetic operation to two ratios.
func apply_ratio(
  left Real, operation Number_Binary, right Real,
) (result Ratio_Result) {
  defer func() { Ratio_Result_Invariants(&result, "apply_ratio.result") }()
  Real_Invariants(left, "apply_ratio.left")
  Number_Binary_Invariants(operation, "apply_ratio.operation")
  Real_Invariants(right, "apply_ratio.right")
  switch operation {
  case Number_Binary(BINARY_ADD):
    var folded Real
    held := rational.Add(
      (*rational.Rational)(&folded),
      rational.Rational(left),
      rational.Rational(right),
    )
    return Ratio_Result{Value: folded, OK: Read(held)}
  case Number_Binary(BINARY_SUBTRACT):
    var folded Real
    held := rational.Subtract(
      (*rational.Rational)(&folded),
      rational.Rational(left),
      rational.Rational(right),
    )
    return Ratio_Result{Value: folded, OK: Read(held)}
  case Number_Binary(BINARY_MULTIPLY):
    var folded Real
    held := rational.Multiply(
      (*rational.Rational)(&folded),
      rational.Rational(left),
      rational.Rational(right),
    )
    return Ratio_Result{Value: folded, OK: Read(held)}
  }
  var folded Real
  held := rational.Divide(
    (*rational.Rational)(&folded),
    rational.Rational(left),
    rational.Rational(right),
  )
  return Ratio_Result{Value: folded, OK: Read(held)}
}

// Applies one arithmetic operation to two pairs of numbers. A number that stands on the number
// line carries a zero imaginary part, thus one path serves a pair and a plain number alike.
// Pair is the two numbers one complex operation works over. Such an operation reads no kind and
// no text, thus it takes the pair and never the whole value, and its body owes the kind domain
// and the text domain nothing.
type Pair struct {
  // Real holds the number that stands on the number line.
  Real Real_Storage
  // Imaginary holds the number that stands off it.
  Imaginary Imaginary_Storage
}

// Pair_Invariants composes the two numbers of one pair.
func Pair_Invariants(value *Pair, namespace aver.Namespace) {
  Real_Storage_Invariants(&value.Real, namespace)
  Imaginary_Storage_Invariants(&value.Imaginary, namespace)
}

// Reads the two numbers one value holds.
func Pair_Of(value Value) (pair Pair) {
  defer func() { Pair_Invariants(&pair, "pair_of.pair") }()
  Value_Invariants(&value, "pair_of.value")
  return Pair{Real: value.Real, Imaginary: value.Imaginary}
}

// Pair_Fold is what one complex step folds to: the two numbers, and whether the step held. A step
// whose storage overran folds to no pair, and the caller alone names the unknown. The fold
// carries no kind, thus a step that makes one kind owes the other kinds nothing.
type Pair_Fold struct {
  // Numbers stands at zero where the step did not hold.
  Numbers Pair
  // OK distinguishes a folded pair from storage that overran.
  OK Read
}

// Pair_Fold_Invariants composes one complex fold.
func Pair_Fold_Invariants(value *Pair_Fold, namespace aver.Namespace) {
  Pair_Invariants(&value.Numbers, namespace)
  Read_Invariants(value.OK, namespace)
}

func complex_operation(left Pair, operation Number_Binary, right Pair) (result Pair_Fold) {
  defer func() { Pair_Fold_Invariants(&result, "complex_operation.result") }()
  Pair_Invariants(&left, "complex_operation.left")
  Number_Binary_Invariants(operation, "complex_operation.operation")
  Pair_Invariants(&right, "complex_operation.right")
  switch operation {
  case Number_Binary(BINARY_ADD), Number_Binary(BINARY_SUBTRACT):
    return add_complex(left, Sum_Binary(operation), right)
  case Number_Binary(BINARY_MULTIPLY):
    return multiply_complex(left, right)
  }
  return divide_complex(left, right)
}

// Adds or subtracts two pairs of numbers, one part at a time.
func add_complex(left Pair, operation Sum_Binary, right Pair) (result Pair_Fold) {
  defer func() { Pair_Fold_Invariants(&result, "add_complex.result") }()
  Pair_Invariants(&left, "add_complex.left")
  Sum_Binary_Invariants(operation, "add_complex.operation")
  Pair_Invariants(&right, "add_complex.right")
  real_part := apply_ratio(left.Real.Value,
    Number_Binary(operation), right.Real.Value)
  if !bool(real_part.OK) {
    return Pair_Fold{Numbers: Zero_Pair(), OK: false}
  }
  imaginary_part := apply_ratio(
    Real(left.Imaginary.Value),
    Number_Binary(operation),
    Real(right.Imaginary.Value),
  )
  if !bool(imaginary_part.OK) {
    return Pair_Fold{Numbers: Zero_Pair(), OK: false}
  }
  return Pair_Fold{Numbers: Make_Complex(real_part.Value, imaginary_part.Value), OK: true}
}

// Multiplies two pairs of numbers. The imaginary parts multiply to a real term of the other sign,
// which is why the real part of the product subtracts.
func multiply_complex(left Pair, right Pair) (result Pair_Fold) {
  defer func() { Pair_Fold_Invariants(&result, "multiply_complex.result") }()
  Pair_Invariants(&left, "multiply_complex.left")
  Pair_Invariants(&right, "multiply_complex.right")
  across := apply_ratio(
    left.Real.Value, NUMBER_MULTIPLY,
    right.Real.Value,
  )
  turned := apply_ratio(
    Real(left.Imaginary.Value), NUMBER_MULTIPLY,
    Real(right.Imaginary.Value),
  )
  upward := apply_ratio(
    Real(left.Imaginary.Value), NUMBER_MULTIPLY,
    right.Real.Value,
  )
  downward := apply_ratio(
    left.Real.Value, NUMBER_MULTIPLY,
    Real(right.Imaginary.Value),
  )
  held := across.OK
  if !turned.OK {
    held = false
  }
  if !upward.OK {
    held = false
  }
  if !downward.OK {
    held = false
  }
  if !bool(held) {
    return Pair_Fold{Numbers: Zero_Pair(), OK: false}
  }
  real_part := apply_ratio(across.Value, NUMBER_SUBTRACT, turned.Value)
  imaginary_part := apply_ratio(upward.Value, NUMBER_ADD, downward.Value)
  if !bool(real_part.OK) {
    return Pair_Fold{Numbers: Zero_Pair(), OK: false}
  }
  if !bool(imaginary_part.OK) {
    return Pair_Fold{Numbers: Zero_Pair(), OK: false}
  }
  return Pair_Fold{Numbers: Make_Complex(real_part.Value, imaginary_part.Value), OK: true}
}

// Divides two pairs of numbers. A product with the conjugate leaves a real divisor, thus the
// division is two divisions by one ratio.
func divide_complex(left Pair, right Pair) (result Pair_Fold) {
  defer func() { Pair_Fold_Invariants(&result, "divide_complex.result") }()
  Pair_Invariants(&left, "divide_complex.left")
  Pair_Invariants(&right, "divide_complex.right")
  // The conjugate stands ahead of the divisor, because the one imaginary part no reversal
  // admits is the most negative whole number, and its square leaves the limbs too. A
  // divisor read first would answer for that part and the reversal would never be asked.
  turned := conjugate(right)
  if !bool(turned.OK) {
    return Pair_Fold{Numbers: Zero_Pair(), OK: false}
  }
  scale := square_sum(right)
  if !bool(scale.OK) {
    return Pair_Fold{Numbers: Zero_Pair(), OK: false}
  }
  product := multiply_complex(left, turned.Numbers)
  if !bool(product.OK) {
    return Pair_Fold{Numbers: Zero_Pair(), OK: false}
  }
  real_part := apply_ratio(
    product.Numbers.Real.Value, NUMBER_QUOTIENT, scale.Value,
  )
  imaginary_part := apply_ratio(
    Real(product.Numbers.Imaginary.Value), NUMBER_QUOTIENT, scale.Value,
  )
  if !bool(real_part.OK) {
    return Pair_Fold{Numbers: Zero_Pair(), OK: false}
  }
  if !bool(imaginary_part.OK) {
    return Pair_Fold{Numbers: Zero_Pair(), OK: false}
  }
  return Pair_Fold{Numbers: Make_Complex(real_part.Value, imaginary_part.Value), OK: true}
}

// Sums the squares of the two parts of a pair, which is the divisor a division by a pair leaves.
func square_sum(value Pair) (result Ratio_Result) {
  defer func() { Ratio_Result_Invariants(&result, "square_sum.result") }()
  Pair_Invariants(&value, "square_sum.value")
  real_square := apply_ratio(
    value.Real.Value, NUMBER_MULTIPLY,
    value.Real.Value,
  )
  imaginary_square := apply_ratio(
    Real(value.Imaginary.Value),
    NUMBER_MULTIPLY,
    Real(value.Imaginary.Value),
  )
  held := real_square.OK
  if !imaginary_square.OK {
    held = false
  }
  if !bool(held) {
    rational.Zero((*rational.Rational)(&result.Value))
    return result
  }
  summed := apply_ratio(real_square.Value, NUMBER_ADD, imaginary_square.Value)
  if !bool(summed.OK) {
    rational.Zero((*rational.Rational)(&result.Value))
    return result
  }
  if bool(rational.Is_Zero(rational.Rational(summed.Value))) {
    rational.Zero((*rational.Rational)(&result.Value))
    return result
  }
  return summed
}

// Reverses the imaginary part of a pair and leaves the real part alone.
func conjugate(value Pair) (result Pair_Fold) {
  defer func() { Pair_Fold_Invariants(&result, "conjugate.result") }()
  Pair_Invariants(&value, "conjugate.value")
  var turned Real
  ok := rational.Negate(
    (*rational.Rational)(&turned), rational.Rational(value.Imaginary.Value),
  )
  if !bool(ok) {
    return Pair_Fold{Numbers: Zero_Pair(), OK: false}
  }
  return Pair_Fold{Numbers: Make_Complex(value.Real.Value, turned), OK: true}
}

// Comparison_Result keeps ordering and comparability at one output boundary.
type Comparison_Result struct {
  // Order remains same when values admit no comparison.
  Order integer.Order
  // OK distinguishes equality from values that admit no comparison.
  OK Read
}

// Comparison_Result_Invariants composes one comparison outcome.
func Comparison_Result_Invariants(value *Comparison_Result, namespace aver.Namespace) {
  integer.Order_Invariants(value.Order, namespace)
  Read_Invariants(value.OK, namespace)
}

// Compare reads two values of one kind and reports the order Go states. Two values of different
// kinds compare as no order at all.
func Compare(left Value, right Value) (result Comparison_Result) {
  defer func() { Comparison_Result_Invariants(&result, "compare.result") }()
  Value_Invariants(&left, "compare.left")
  Value_Invariants(&right, "compare.right")
  if bool(is_number(Kind_Of(left))) {
    if bool(is_number(Kind_Of(right))) {
      return compare_number(Pair_Of(left), Pair_Of(right))
    }
  }
  if Kind_Of(left) != Kind_Of(right) {
    return Comparison_Result{Order: integer.ORDER_SAME, OK: false}
  }
  switch Kind_Of(left) {
  case KIND_BOOLEAN:
    if truth_of(Pair_Of(left)) == truth_of(Pair_Of(right)) {
      return Comparison_Result{Order: integer.ORDER_SAME, OK: true}
    }
    if bool(truth_of(Pair_Of(left))) {
      return Comparison_Result{Order: integer.ORDER_AFTER, OK: true}
    }
    return Comparison_Result{Order: integer.ORDER_BEFORE, OK: true}
  case KIND_TEXT:
    return Comparison_Result{
      Order: compare_text(view_of(left.Texts), view_of(right.Texts)),
      OK:    true,
    }
  }
  return Comparison_Result{Order: integer.ORDER_SAME, OK: false}
}

// Reads the order of two numbers. A pair orders by its real part and then by its imaginary part,
// thus two pairs that read the same order hold the same two numbers, which is the one comparison
// Go states for a pair.
func compare_number(left Pair, right Pair) (result Comparison_Result) {
  defer func() { Comparison_Result_Invariants(&result, "compare_number.result") }()
  Pair_Invariants(&left, "compare_number.left")
  Pair_Invariants(&right, "compare_number.right")
  folded, held := rational.Compare(
    rational.Rational(left.Real.Value),
    rational.Rational(right.Real.Value),
  )
  if !bool(held) {
    return Comparison_Result{Order: integer.ORDER_SAME, OK: false}
  }
  if folded != integer.ORDER_SAME {
    return Comparison_Result{Order: folded, OK: true}
  }
  turned, turned_held := rational.Compare(
    rational.Rational(left.Imaginary.Value),
    rational.Rational(right.Imaginary.Value),
  )
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
  value Pair, operation Shift_Operation, count integer.Shift_Count,
) (result Pair_Fold) {
  defer func() { Pair_Fold_Invariants(&result, "shift.result") }()
  Pair_Invariants(&value, "shift.value")
  Shift_Operation_Invariants(operation, "shift.operation")
  integer.Shift_Count_Invariants(count, "shift.count")
  var whole rational.Numerator
  ok := rational.Whole(&whole, rational.Rational(value.Real.Value))
  if !bool(ok) {
    return Pair_Fold{Numbers: Zero_Pair(), OK: false}
  }
  if operation == SHIFT_DOWN {
    var moved integer.Integer
    integer.Shift_Right(&moved, integer.Integer{Limbs: integer.Limbs(whole)}, count)
    return Pair_Fold{Numbers: whole_value(rational.Numerator(moved.Limbs)), OK: true}
  }
  var moved integer.Integer
  held := integer.Shift_Left(&moved, integer.Integer{Limbs: integer.Limbs(whole)}, count)
  if !bool(held) {
    return Pair_Fold{Numbers: Zero_Pair(), OK: false}
  }
  return Pair_Fold{Numbers: whole_value(rational.Numerator(moved.Limbs)), OK: true}
}

// Into_Text writes a value into caller storage and returns the byte count. An unknown writes
// nothing, because no form states a value that was never folded.
func Into_Text(destination Form, value Value) (count Form_Count) {
  defer func() { Form_Count_Invariants(count, "into_text.count") }()
  Form_Invariants(destination, "into_text.destination")
  Value_Invariants(&value, "into_text.value")
  switch Kind_Of(value) {
  case KIND_UNKNOWN:
    return FORM_SIZE_MINIMUM
  case KIND_BOOLEAN:
    return Form_Count(write_truth(destination, truth_of(Pair_Of(value))))
  case KIND_TEXT:
    if len(destination) < len(view_of(value.Texts)) {
      return FORM_SIZE_MINIMUM
    }
    return Form_Count(copy(destination, view_of(value.Texts)))
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
  below := rational.Into_Text(imaginary_storage[:], rational.Rational(value.Imaginary.Value))
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
