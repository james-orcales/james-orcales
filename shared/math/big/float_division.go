// Package big keeps large arithmetic bounded without hidden allocation.
package big

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
)

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

// FLOAT_TEXT_SIZE_MAXIMUM is negative fixed text at maximum integer digits and precision.
const FLOAT_TEXT_SIZE_MAXIMUM = SIGN_BYTE_COUNT_MAXIMUM +
	FLOAT_TEXT_INTEGER_DIGIT_COUNT_MAXIMUM + DECIMAL_POINT_BYTE_COUNT +
	FLOAT_TEXT_PRECISION_MAXIMUM

// FLOAT_DECIMAL_DIGIT_NUMERATOR_MAXIMUM keeps the logarithm formula integral.
const FLOAT_DECIMAL_DIGIT_NUMERATOR_MAXIMUM = BIT_COUNT_MAXIMUM *
	DECIMAL_DIGIT_BINARY_LOGARITHM_CEILING

// FLOAT_DECIMAL_INTEGER_DIGIT_COUNT_MAXIMUM includes one shortest-bound guard bit.
const FLOAT_DECIMAL_INTEGER_DIGIT_COUNT_MAXIMUM = FLOAT_DECIMAL_DIGIT_NUMERATOR_MAXIMUM/
	DECIMAL_DIGIT_BINARY_LOGARITHM_SCALE + WORD_COUNT_INCREMENT

// FLOAT_DECIMAL_RIGHT_SHIFT_MAXIMUM includes one shortest-bound half-ulp bit.
const FLOAT_DECIMAL_RIGHT_SHIFT_MAXIMUM = FLOAT_PRECISION_MAXIMUM -
	FLOAT_EXPONENT_MINIMUM + WORD_COUNT_INCREMENT

// FLOAT_DECIMAL_DIGIT_COUNT_MAXIMUM holds exact mantissa times every denominator five.
const FLOAT_DECIMAL_DIGIT_COUNT_MAXIMUM = FLOAT_DECIMAL_INTEGER_DIGIT_COUNT_MAXIMUM +
	FLOAT_DECIMAL_RIGHT_SHIFT_MAXIMUM

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

// FLOAT_TEXT_MAGNITUDE_WORD_COUNT_INDEX selects populated extended words.
const FLOAT_TEXT_MAGNITUDE_WORD_COUNT_INDEX = WORD_COUNT_MINIMUM

// FLOAT_TEXT_MAGNITUDE_CONTROL_COUNT owns one extended word count.
const FLOAT_TEXT_MAGNITUDE_CONTROL_COUNT = FLOAT_TEXT_MAGNITUDE_WORD_COUNT_INDEX +
	WORD_COUNT_INCREMENT

// FLOAT_TEXT_INTEGER_INDEX selects normalized significand scratch.
const FLOAT_TEXT_INTEGER_INDEX = WORD_COUNT_MINIMUM

// FLOAT_TEXT_EXPONENT_INDEX selects signed exponent conversion scratch.
const FLOAT_TEXT_EXPONENT_INDEX = FLOAT_TEXT_INTEGER_INDEX + WORD_COUNT_INCREMENT

// FLOAT_TEXT_INTEGER_COUNT owns significand and exponent conversion values.
const FLOAT_TEXT_INTEGER_COUNT = FLOAT_TEXT_EXPONENT_INDEX + WORD_COUNT_INCREMENT

// FLOAT_TEXT_SOURCE_INDEX selects preserved caller value.
const FLOAT_TEXT_SOURCE_INDEX = WORD_COUNT_MINIMUM

// FLOAT_TEXT_ROUNDED_INDEX selects rounded hexadecimal copy.
const FLOAT_TEXT_ROUNDED_INDEX = FLOAT_TEXT_SOURCE_INDEX + WORD_COUNT_INCREMENT

// FLOAT_TEXT_VALUE_COUNT owns source and rounded values.
const FLOAT_TEXT_VALUE_COUNT = FLOAT_TEXT_ROUNDED_INDEX + WORD_COUNT_INCREMENT

// FLOAT_TEXT_MANTISSA_WORKSPACE_INDEX selects significand digit conversion.
const FLOAT_TEXT_MANTISSA_WORKSPACE_INDEX = WORD_COUNT_MINIMUM

// FLOAT_TEXT_EXPONENT_WORKSPACE_INDEX selects exponent digit conversion.
const FLOAT_TEXT_EXPONENT_WORKSPACE_INDEX = FLOAT_TEXT_MANTISSA_WORKSPACE_INDEX +
	WORD_COUNT_INCREMENT

// FLOAT_TEXT_INTEGER_WORKSPACE_COUNT owns both independent digit conversions.
const FLOAT_TEXT_INTEGER_WORKSPACE_COUNT = FLOAT_TEXT_EXPONENT_WORKSPACE_INDEX +
	WORD_COUNT_INCREMENT

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

// FLOAT_TEXT_DECIMAL_INDEX selects one exact decimal workspace value.
const FLOAT_TEXT_DECIMAL_INDEX = FLOAT_TEXT_FORMAT_INDEX + WORD_COUNT_INCREMENT

// FLOAT_TEXT_DECIMAL_SHIFT_INDEX selects one bounded decimal right shift.
const FLOAT_TEXT_DECIMAL_SHIFT_INDEX = FLOAT_TEXT_DECIMAL_INDEX + WORD_COUNT_INCREMENT

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
	value Float_Gob_Encoding, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), WORD_COUNT_MINIMUM, FLOAT_GOB_SIZE_MAXIMUM,
		).
		Ensure()
}

// Float_Gob_Encoding_Unvalidated admits one hostile excess input.
type Float_Gob_Encoding_Unvalidated []byte

// Float_Gob_Encoding_Unvalidated_Invariants bounds validation work.
func Float_Gob_Encoding_Unvalidated_Invariants(
	value Float_Gob_Encoding_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), WORD_COUNT_MINIMUM,
			FLOAT_GOB_UNVALIDATED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Float_Gob_Header_Encoding prevents short source reads after common header validation.
type Float_Gob_Header_Encoding [FLOAT_GOB_HEADER_SIZE]byte

// Float_Gob_Header_Encoding_Invariants fixes common wire header width.
func Float_Gob_Header_Encoding_Invariants(
	value Float_Gob_Header_Encoding, _ invariant.Namespace,
) {
	invariant.Always(
		len(value) == FLOAT_GOB_HEADER_SIZE,
		"Float gob header has fixed wire width.",
	)
}

// Float_Gob_Mantissa_Size includes malformed finite header shortfalls.
type Float_Gob_Mantissa_Size int

// Float_Gob_Mantissa_Size_Invariants bounds structural validation work.
func Float_Gob_Mantissa_Size_Invariants(
	value Float_Gob_Mantissa_Size, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), FLOAT_GOB_MANTISSA_SIZE_MINIMUM,
			FLOAT_GOB_MANTISSA_SIZE_MAXIMUM,
		).
		Ensure()
}

// Float_Gob_Exponent_Encoding prevents partial finite exponent reads.
type Float_Gob_Exponent_Encoding [FLOAT_GOB_FIELD_SIZE]byte

// Float_Gob_Exponent_Encoding_Invariants fixes finite exponent field width.
func Float_Gob_Exponent_Encoding_Invariants(
	value Float_Gob_Exponent_Encoding, _ invariant.Namespace,
) {
	invariant.Always(
		len(value) == FLOAT_GOB_FIELD_SIZE,
		"Float gob exponent has fixed wire width.",
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
	value Float_Gob_Header, namespace invariant.Namespace,
) {
	Float_Precision_Invariants(value.Precision, namespace)
	Rounding_Mode_Invariants(value.Mode, namespace)
	Accuracy_Invariants(value.Accuracy, namespace)
	Float_Form_Invariants(value.Form, namespace)
	Polarity_Invariants(value.Negative, namespace)
}

// Float_Gob_Mantissas keeps one Int without duplicating Int invariant chains.
type Float_Gob_Mantissas [FLOAT_GOB_MANTISSA_COUNT]Int

// Float_Gob_Mantissas_Invariants fixes transactional storage count.
func Float_Gob_Mantissas_Invariants(
	value Float_Gob_Mantissas, _ invariant.Namespace,
) {
	invariant.Always(
		len(value) == FLOAT_GOB_MANTISSA_COUNT,
		"Float gob workspace owns one mantissa.",
	)
}

// Float_32_Bits keeps raw binary32 input distinct from numeric-only output.
type Float_32_Bits uint32

// Float_32_Bits_Invariants admits every binary32 encoding, including NaN.
func Float_32_Bits_Invariants(value Float_32_Bits, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint32(uint32(value), bits.WORD_32_MINIMUM, bits.WORD_32_MAXIMUM).
		Ensure()
}

// Float_32_Finite_Exponent_Field prevents NaN and infinity from entering finite decoding.
type Float_32_Finite_Exponent_Field uint32

// Float_32_Finite_Exponent_Field_Invariants bounds finite encoded exponent.
func Float_32_Finite_Exponent_Field_Invariants(
	value Float_32_Finite_Exponent_Field, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
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
	value Float_32_Mantissa_Field, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint32(
			uint32(value), bits.WORD_32_MINIMUM, FLOAT_32_MANTISSA_MASK,
		).
		Ensure()
}

// Float_32_Sign keeps only clear or encoded binary32 sign.
type Float_32_Sign uint32

// Float_32_Sign_Invariants rejects mantissa or exponent bits.
func Float_32_Sign_Invariants(value Float_32_Sign, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint32(uint32(value), bits.WORD_32_MINIMUM, FLOAT_32_SIGN_MASK).
		Ensure()
}

// Float_32_Value_Bits excludes NaN while retaining signed infinity and zero.
type Float_32_Value_Bits uint32

// Float_32_Value_Bits_Invariants rejects every NaN payload.
func Float_32_Value_Bits_Invariants(
	value Float_32_Value_Bits, namespace invariant.Namespace,
) {
	encoded := uint32(value)
	invariant.Tree(value, namespace).
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
	invariant.Always(
		not_a_number == bits.WORD_32_MINIMUM,
		"Numeric binary32 output never encodes NaN.",
	)
}

// Int_Float_64_Value_Bits is one integral binary64 result or signed infinity.
type Int_Float_64_Value_Bits uint64

// Int_Float_64_Value_Bits_Invariants excludes negative zero, subnormal, and NaN.
func Int_Float_64_Value_Bits_Invariants(
	value Int_Float_64_Value_Bits, namespace invariant.Namespace,
) {
	encoded := uint64(value)
	invariant.Tree(value, namespace).
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
	invariant.Always(
		nonzero <= exponent_nonzero,
		"Integral binary64 output gives zero no negative or subnormal twin.",
	)
	exponent_difference := exponent ^ FLOAT_64_EXPONENT_MASK
	exponent_difference_nonzero := (exponent_difference | -exponent_difference) >>
		WORD_BIT_INDEX_MAXIMUM
	exponent_is_nonfinite := uint64(bits.CARRY_MAXIMUM) - exponent_difference_nonzero
	mantissa_nonzero := (mantissa | -mantissa) >> WORD_BIT_INDEX_MAXIMUM
	invariant.Always(
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
	value Int_Float_64_Exponent_Field, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
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
	value Int_Float_64_Mantissa_Field, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
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
	value Int_Float_64_Sign, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_Uint64(
			uint64(value), bits.WORD_64_MINIMUM, FLOAT_64_SIGN_MASK,
		).
		Ensure()
}

// Rat_Float_32_Mantissa is the rounded significand before hidden-bit removal.
type Rat_Float_32_Mantissa uint32

// Rat_Float_32_Mantissa_Invariants bounds the final binary32 significand.
func Rat_Float_32_Mantissa_Invariants(
	value Rat_Float_32_Mantissa, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
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
	value Rat_Float_32_Rounding_Mantissa, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
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
	value Rat_Float_32_Exponent, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
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
	value Rat_Float_32_Rounding_Exponent, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), RAT_FLOAT_32_EXPONENT_MINIMUM,
			RAT_FLOAT_32_ROUNDING_EXPONENT_MAXIMUM,
		).
		Ensure()
}

// Rat_Float_32_Integers owns scaled operands and quotient-remainder output.
type Rat_Float_32_Integers [RAT_FLOAT_32_INTEGER_COUNT]Int

// Rat_Float_32_Integers_Invariants fixes complete binary32 conversion capacity.
func Rat_Float_32_Integers_Invariants(
	value Rat_Float_32_Integers, _ invariant.Namespace,
) {
	invariant.Always(
		len(value) == RAT_FLOAT_32_INTEGER_COUNT,
		"Rational binary32 conversion has fixed integer capacity.",
	)
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
	value *Rat_Float_32_Workspace, namespace invariant.Namespace,
) {
	Rat_Float_32_Integers_Invariants(value.Integers, namespace)
	Division_Memory_Invariants(value.Division, namespace)
}

// Modular_Square_Root_Status separates malformed bounds from mathematical absence.
type Modular_Square_Root_Status uint8

// Modular_Square_Root_Status_Invariants admits every transactional root outcome.
func Modular_Square_Root_Status_Invariants(
	value Modular_Square_Root_Status, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
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
	value Modular_Square_Root_Result_Status, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_RESULT_ABSENT),
		).
		Ensure()
}

// Modular_Square_Root_Integers owns complete Tonelli-Shanks state.
type Modular_Square_Root_Integers [MODULAR_SQUARE_ROOT_INTEGER_COUNT]Int

// Modular_Square_Root_Integers_Invariants fixes bounded algorithm capacity.
func Modular_Square_Root_Integers_Invariants(
	value Modular_Square_Root_Integers, _ invariant.Namespace,
) {
	invariant.Always(
		len(value) == MODULAR_SQUARE_ROOT_INTEGER_COUNT,
		"Modular square-root integer storage has fixed Tonelli-Shanks capacity.",
	)
}

// Modular_Square_Root_Modular_Memory gives nested arithmetic one chain identity.
type Modular_Square_Root_Modular_Memory Int_Modular_Workspace

// Modular_Square_Root_Modular_Memory_Invariants composes modular scratch once.
func Modular_Square_Root_Modular_Memory_Invariants(
	value Modular_Square_Root_Modular_Memory, namespace invariant.Namespace,
) {
	invariant.Always(
		len(value.Integers) == MODULAR_INTEGER_COUNT,
		"Modular square-root arithmetic owns complete modular integer state.",
	)
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
	value *Int_Modular_Square_Root_Workspace, namespace invariant.Namespace,
) {
	Modular_Square_Root_Integers_Invariants(value.Integers, namespace)
	Modular_Square_Root_Modular_Memory_Invariants(value.Modular, namespace)
	Jacobi_Integers_Invariants(value.Jacobi, namespace)
}

// Float_Text_Format_Unvalidated admits every hostile format byte.
type Float_Text_Format_Unvalidated byte

// Float_Text_Format_Unvalidated_Invariants keeps format validation constant work.
func Float_Text_Format_Unvalidated_Invariants(
	value Float_Text_Format_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), bits.WORD_8_MINIMUM, bits.WORD_8_MAXIMUM).
		Ensure()
}

// Float_Text_Format is one validated representation kind.
type Float_Text_Format uint8

// Float_Text_Format_Invariants rejects every unsupported format byte.
func Float_Text_Format_Invariants(
	value Float_Text_Format, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
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
	value Float_Text_Precision_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
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
	value Float_Text_Precision, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
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
	value Float_Text_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, FLOAT_TEXT_SIZE_MAXIMUM,
		).
		Ensure()
}

// Float_Text_Status separates malformed policy from short caller storage.
type Float_Text_Status uint8

// Float_Text_Status_Invariants admits every transactional text outcome.
func Float_Text_Status_Invariants(
	value Float_Text_Status, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_DESTINATION_TOO_SMALL),
		).
		Ensure()
}

// Float_Text_Workspace owns every conversion and transactional output byte.
type Float_Text_Workspace struct {
	// Integers hold aligned significand and signed exponent.
	Integers [FLOAT_TEXT_INTEGER_COUNT]Int
	// Values retain hexadecimal rounding outside caller value.
	Values [FLOAT_TEXT_VALUE_COUNT]Float
	// Text converts significand and exponent independently.
	Text [FLOAT_TEXT_INTEGER_WORKSPACE_COUNT]Int_Text_Workspace
	// Control holds output cursor, precision, and exponent padding.
	Control [FLOAT_TEXT_CONTROL_COUNT]int
	// Decimals hold exact value and shortest-rounding bounds.
	Decimals [FLOAT_DECIMAL_COUNT]struct {
		Digits  [FLOAT_DECIMAL_DIGIT_COUNT_MAXIMUM]byte
		Control [FLOAT_DECIMAL_CONTROL_COUNT]int
	}
	// Magnitude holds one precision-plus-guard shortest midpoint.
	Magnitude struct {
		Words   [FLOAT_TEXT_MAGNITUDE_WORD_COUNT_MAXIMUM]Word
		Control [FLOAT_TEXT_MAGNITUDE_CONTROL_COUNT]int
	}
	// Output prevents partial caller mutation on short storage.
	Output [FLOAT_TEXT_SIZE_MAXIMUM]byte
}

// Float_Text_Workspace_Invariants composes complete bounded text storage.
func Float_Text_Workspace_Invariants(
	value *Float_Text_Workspace, _ invariant.Namespace,
) {
	invariant.Always(len(value.Integers) == FLOAT_TEXT_INTEGER_COUNT,
		"Float text owns complete integer conversion storage.")
	invariant.Always(len(value.Values) == FLOAT_TEXT_VALUE_COUNT,
		"Float text owns source and rounded values.")
	invariant.Always(len(value.Text) == FLOAT_TEXT_INTEGER_WORKSPACE_COUNT,
		"Float text owns both integer text workspaces.")
	invariant.Always(len(value.Control) == FLOAT_TEXT_CONTROL_COUNT,
		"Float text owns complete scalar control storage.")
	invariant.Always(len(value.Decimals) == FLOAT_DECIMAL_COUNT,
		"Float text owns value and both shortest bounds.")
	invariant.Always(
		len(value.Decimals[FLOAT_DECIMAL_VALUE_INDEX].Digits) ==
			FLOAT_DECIMAL_DIGIT_COUNT_MAXIMUM,
		"Float text value decimal retains every exact digit.",
	)
	invariant.Always(
		len(value.Decimals[FLOAT_DECIMAL_LOWER_INDEX].Digits) ==
			FLOAT_DECIMAL_DIGIT_COUNT_MAXIMUM,
		"Float text lower decimal retains every exact digit.",
	)
	invariant.Always(
		len(value.Decimals[FLOAT_DECIMAL_UPPER_INDEX].Digits) ==
			FLOAT_DECIMAL_DIGIT_COUNT_MAXIMUM,
		"Float text upper decimal retains every exact digit.",
	)
	invariant.Always(
		len(value.Decimals[FLOAT_DECIMAL_VALUE_INDEX].Control) ==
			FLOAT_DECIMAL_CONTROL_COUNT,
		"Float text value decimal retains count and point state.",
	)
	invariant.Always(
		len(value.Decimals[FLOAT_DECIMAL_LOWER_INDEX].Control) ==
			FLOAT_DECIMAL_CONTROL_COUNT,
		"Float text lower decimal retains count and point state.",
	)
	invariant.Always(
		len(value.Decimals[FLOAT_DECIMAL_UPPER_INDEX].Control) ==
			FLOAT_DECIMAL_CONTROL_COUNT,
		"Float text upper decimal retains count and point state.",
	)
	invariant.Always(len(value.Magnitude.Words) == FLOAT_TEXT_MAGNITUDE_WORD_COUNT_MAXIMUM,
		"Float text midpoint retains one guard bit.")
	invariant.Always(len(value.Magnitude.Control) == FLOAT_TEXT_MAGNITUDE_CONTROL_COUNT,
		"Float text midpoint retains its active word count.")
	invariant.Always(len(value.Output) == FLOAT_TEXT_SIZE_MAXIMUM,
		"Float text owns complete transactional output.")
}

// Float_Parse_Base is one validated or resolved stdlib Float radix kind.
type Float_Parse_Base uint8

// Float_Parse_Base_Invariants keeps automatic and four numeric radix kinds exhaustive.
func Float_Parse_Base_Invariants(
	value Float_Parse_Base, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
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
	value Float_Parse_Text_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
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
	value Float_Parse_Integer_Workspace, _ invariant.Namespace,
) {
	invariant.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"Float parse mantissa retains one complete Int magnitude.")
}

// Float_Parse_Division_Workspace gives quotient storage one invariant identity.
type Float_Parse_Division_Workspace Float_Division_Workspace

// Float_Parse_Division_Workspace_Invariants fixes exact-factor quotient storage.
func Float_Parse_Division_Workspace_Invariants(
	value Float_Parse_Division_Workspace, _ invariant.Namespace,
) {
	invariant.Always(len(value.Remainder) == FLOAT_DIVISION_WORD_COUNT_MAXIMUM,
		"Float parse quotient retains one shifted remainder.")
	invariant.Always(len(value.Divisor) == FLOAT_DIVISION_WORD_COUNT_MAXIMUM,
		"Float parse quotient retains one aligned divisor.")
}

// Float_Parse_Workspace owns source, exact factors, quotient, and syntax scratch.
type Float_Parse_Workspace struct {
	// Source keeps helper reads inside validated caller-independent storage.
	Source [FLOAT_PARSE_TEXT_SIZE_MAXIMUM]byte
	// Control keeps source size and policy out of semantic invariant domains.
	Control [FLOAT_PARSE_CONTROL_COUNT]int64
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
	value *Float_Parse_Workspace, namespace invariant.Namespace,
) {
	invariant.Always(len(value.Source) == FLOAT_PARSE_TEXT_SIZE_MAXIMUM,
		"Float parse owns complete validated source storage.")
	invariant.Always(len(value.Control) == FLOAT_PARSE_CONTROL_COUNT,
		"Float parse owns complete structural scanner state.")
	Float_Parse_Integer_Workspace_Invariants(value.Parse, namespace)
	Rat_Parse_Fraction_Integers_Invariants(value.Integers, namespace)
	Float_Rat_Values_Invariants(value.Values, namespace)
	Float_Parse_Division_Workspace_Invariants(value.Division, namespace)
}

// Float_Rat_Status separates nonfinite input from bounded component overflow.
type Float_Rat_Status uint8

// Float_Rat_Status_Invariants admits every transactional rational conversion result.
func Float_Rat_Status_Invariants(value Float_Rat_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
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
	value Float_Square_Root_Source_Bit_Index, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), BIT_INDEX_UNVALIDATED_MINIMUM,
			FLOAT_SQUARE_ROOT_SOURCE_BIT_INDEX_MAXIMUM,
		).
		Ensure()
}

// Float_Square_Root_Exponent is the normalized exact root exponent before rounding.
type Float_Square_Root_Exponent int

// Float_Square_Root_Exponent_Invariants binds root scale to halved source bounds.
func Float_Square_Root_Exponent_Invariants(
	value Float_Square_Root_Exponent, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
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
	value *Float_Nonnegative_Finite, namespace invariant.Namespace,
) {
	invariant.Tree(Float_Active_Precision(value.Precision), namespace).
		Range_Uint(
			uint(value.Precision), WORD_COUNT_INCREMENT, FLOAT_PRECISION_MAXIMUM,
		).
		Ensure()
	invariant.Tree(Rounding_Mode(value.Mode), namespace).
		Range_Uint8(
			uint8(value.Mode), uint8(ROUND_TO_NEAREST_EVEN),
			uint8(ROUND_TO_POSITIVE_INFINITY),
		).
		Ensure()
	invariant.Tree(Accuracy(value.Accuracy), namespace).
		Enum_3_Int8(
			int8(value.Accuracy), int8(ACCURACY_BELOW), int8(ACCURACY_EXACT),
			int8(ACCURACY_ABOVE),
		).
		Ensure()
	invariant.Tree(Float_Exponent(value.Exponent), namespace).
		Range_Int(
			int(value.Exponent), FLOAT_EXPONENT_MINIMUM, FLOAT_EXPONENT_MAXIMUM,
		).
		Ensure()
	Float_Active_Mantissa_Invariants(Float_Active_Mantissa(value.Mantissa), namespace)
	invariant.Always(value.Form == FLOAT_FORM_FINITE,
		"A nonnegative finite helper value has finite form.")
	invariant.Always(value.Negative == POLARITY_NONNEGATIVE,
		"A square-root source is nonnegative.")
}

// Float_Rat_Values owns all intermediate floats without giving them input semantics.
type Float_Rat_Values [FLOAT_RAT_VALUE_COUNT]Float

// Float_Rat_Values_Invariants fixes complete rational conversion storage.
func Float_Rat_Values_Invariants(value Float_Rat_Values, _ invariant.Namespace) {
	invariant.Always(
		len(value) == FLOAT_RAT_VALUE_COUNT,
		"Float rational conversion keeps both operands and one result.",
	)
}

// Float_Rat_Division_Workspace gives nested quotient memory one invariant identity.
type Float_Rat_Division_Workspace Float_Division_Workspace

// Float_Rat_Division_Workspace_Invariants fixes both quotient magnitudes.
func Float_Rat_Division_Workspace_Invariants(
	value Float_Rat_Division_Workspace, _ invariant.Namespace,
) {
	invariant.Always(
		len(value.Remainder) == FLOAT_DIVISION_WORD_COUNT_MAXIMUM,
		"Float rational quotient remainder retains its carry word.",
	)
	invariant.Always(
		len(value.Divisor) == FLOAT_DIVISION_WORD_COUNT_MAXIMUM,
		"Float rational quotient divisor retains its aligned mantissa.",
	)
}

// Float_Rat_Workspace owns rational operands, result, and quotient scratch.
type Float_Rat_Workspace struct {
	// Values keep operands and result alias-safe.
	Values Float_Rat_Values
	// Integers keep exact binary rational components.
	Integers Rat_Integers
	// Division owns quotient extraction for rational-to-float conversion.
	Division Float_Rat_Division_Workspace
}

// Float_Rat_Workspace_Invariants binds all conversion storage to derived bounds.
func Float_Rat_Workspace_Invariants(
	value *Float_Rat_Workspace, namespace invariant.Namespace,
) {
	Float_Rat_Values_Invariants(value.Values, namespace)
	Rat_Integers_Invariants(value.Integers, namespace)
	Float_Rat_Division_Workspace_Invariants(value.Division, namespace)
}

// Float_Square_Root_Root stores one guard-width restoring-sqrt result.
type Float_Square_Root_Root [FLOAT_SQUARE_ROOT_WORD_COUNT_MAXIMUM]Word

// Float_Square_Root_Root_Invariants fixes the precision-plus-guard capacity.
func Float_Square_Root_Root_Invariants(
	value Float_Square_Root_Root, _ invariant.Namespace,
) {
	invariant.Always(
		len(value) == FLOAT_SQUARE_ROOT_WORD_COUNT_MAXIMUM,
		"Float square-root result storage retains one precision guard bit.",
	)
}

// Float_Square_Root_Remainder stores restoring state beside the root identity.
type Float_Square_Root_Remainder [FLOAT_SQUARE_ROOT_WORD_COUNT_MAXIMUM]Word

// Float_Square_Root_Remainder_Invariants fixes the precision-plus-guard capacity.
func Float_Square_Root_Remainder_Invariants(
	value Float_Square_Root_Remainder, _ invariant.Namespace,
) {
	invariant.Always(
		len(value) == FLOAT_SQUARE_ROOT_WORD_COUNT_MAXIMUM,
		"Float square-root remainder storage retains one precision guard bit.",
	)
}

// Float_Square_Root_Candidate stores the current restoring subtraction magnitude.
type Float_Square_Root_Candidate [FLOAT_SQUARE_ROOT_WORD_COUNT_MAXIMUM]Word

// Float_Square_Root_Candidate_Invariants fixes the precision-plus-guard capacity.
func Float_Square_Root_Candidate_Invariants(
	value Float_Square_Root_Candidate, _ invariant.Namespace,
) {
	invariant.Always(
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
	value *Float_Square_Root_Workspace, namespace invariant.Namespace,
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
	value *Float_Gob_Workspace, namespace invariant.Namespace,
) {
	Float_Gob_Mantissas_Invariants(value.Mantissas, namespace)
}

// Float_Gob_Encode_Into keeps stdlib wire alignment in caller workspace.
func Float_Gob_Encode_Into(
	destination Float_Gob_Encoding,
	value *Float,
	workspace *Float_Gob_Workspace,
) (mantissa_count Word_Count, status Destination_Status) {
	defer func() {
		Word_Count_Invariants(mantissa_count, "float_gob_encode_into.mantissa_count")
		Destination_Status_Invariants(status, "float_gob_encode_into.status")
	}()
	Float_Gob_Encoding_Invariants(destination, "float_gob_encode_into.destination")
	Float_Invariants(value, "float_gob_encode_into.value")
	Float_Gob_Workspace_Invariants(workspace, "float_gob_encode_into.workspace")
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
	mantissa := &workspace.Mantissas[FLOAT_GOB_MANTISSA_INDEX]
	*mantissa = Int(value.Mantissa)
	bit_count := int(Int_Bit_Count(mantissa))
	aligned_bit_count := int(mantissa_count) * WORD_BIT_COUNT
	shift := Shift_Count(aligned_bit_count - bit_count)
	shift_status := Int_Shift_Left(mantissa, mantissa, shift)
	invariant.Always(
		shift_status == Arithmetic_Status(STATUS_OK),
		"Word alignment cannot exceed existing mantissa word count.",
	)
	fill_status := Int_Fill_Bytes(
		Bytes(destination[FLOAT_GOB_MANTISSA_OFFSET:required]),
		mantissa,
	)
	invariant.Always(
		fill_status == Destination_Status(STATUS_OK),
		"Derived wire mantissa storage has exact capacity.",
	)
	return mantissa_count, STATUS_OK
}

// Float_Gob_Decode validates complete wire state before destination mutation.
func Float_Gob_Decode(
	destination *Float,
	source Float_Gob_Encoding_Unvalidated,
	workspace *Float_Gob_Workspace,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "float_gob_decode.status") }()
	Float_Invariants(destination, "float_gob_decode.destination_initial")
	Float_Gob_Encoding_Unvalidated_Invariants(source, "float_gob_decode.source")
	Float_Gob_Workspace_Invariants(workspace, "float_gob_decode.workspace")
	if len(source) > FLOAT_GOB_SIZE_MAXIMUM {
		return STATUS_INPUT_INVALID
	}
	if len(source) == WORD_COUNT_MINIMUM {
		*destination = Float{}
		return STATUS_OK
	}
	if len(source) < FLOAT_GOB_HEADER_SIZE {
		return STATUS_INPUT_INVALID
	}
	if source[FLOAT_GOB_VERSION_OFFSET] != byte(FLOAT_GOB_VERSION) {
		return STATUS_INPUT_INVALID
	}
	var encoded_header Float_Gob_Header_Encoding
	copy(encoded_header[:], source[:FLOAT_GOB_HEADER_SIZE])
	header, validation := float_gob_decode_header(encoded_header)
	if validation != STATUS_OK {
		return STATUS_INPUT_INVALID
	}
	result := *destination
	if float_gob_result_set(&result, header) != STATUS_OK {
		return STATUS_INPUT_INVALID
	}
	if header.Form == FLOAT_FORM_FINITE {
		mantissa_size := Float_Gob_Mantissa_Size(
			len(source) - FLOAT_GOB_MANTISSA_OFFSET,
		)
		high_bit_shift := bits.BIT_COUNT_8_MAXIMUM - WORD_COUNT_INCREMENT
		high_bit_set := Boolean(false)
		if mantissa_size > Float_Gob_Mantissa_Size(WORD_COUNT_MINIMUM) {
			high_bit_set = Boolean(
				source[FLOAT_GOB_MANTISSA_OFFSET]>>high_bit_shift ==
					byte(bits.CARRY_MAXIMUM),
			)
		}
		structure_status := float_gob_mantissa_validate(mantissa_size, high_bit_set)
		if structure_status != STATUS_OK {
			return STATUS_INPUT_INVALID
		}
		var encoded_exponent Float_Gob_Exponent_Encoding
		copy(encoded_exponent[:], source[FLOAT_GOB_EXPONENT_OFFSET:])
		exponent, exponent_status := float_gob_decode_exponent(encoded_exponent)
		if exponent_status != STATUS_OK {
			return STATUS_INPUT_INVALID
		}
		mantissa := &workspace.Mantissas[FLOAT_GOB_MANTISSA_INDEX]
		mantissa_status := Int_Set_Bytes(
			mantissa, Bytes_Unvalidated(source[FLOAT_GOB_MANTISSA_OFFSET:]),
		)
		invariant.Always(
			mantissa_status == Validation_Status(STATUS_OK),
			"Validated wire bound leaves one complete mantissa.",
		)
		precision := Float_Active_Precision(result.Precision)
		status = float_gob_decode_finite((*Float_Active_Mantissa)(mantissa), precision)
		if status != STATUS_OK {
			return status
		}
		result.Mantissa = Float_Mantissa(*mantissa)
		result.Exponent = exponent
	}
	Float_Invariants(&result, "float_gob_decode.result")
	float_gob_decode_commit(destination, &result)
	return STATUS_OK
}

func float_gob_mantissa_validate(
	size Float_Gob_Mantissa_Size, high_bit_set Boolean,
) (status Validation_Status) {
	defer func() {
		Validation_Status_Invariants(status, "float_gob_mantissa_validate.status")
	}()
	Float_Gob_Mantissa_Size_Invariants(size, "float_gob_mantissa_validate.size")
	Boolean_Invariants(high_bit_set, "float_gob_mantissa_validate.high_bit_set")
	if int(size) < WORD_BYTE_COUNT {
		return STATUS_INPUT_INVALID
	}
	if int(size)%WORD_BYTE_COUNT != WORD_COUNT_MINIMUM {
		return STATUS_INPUT_INVALID
	}
	if !high_bit_set {
		return STATUS_INPUT_INVALID
	}
	return STATUS_OK
}

func float_gob_decode_exponent(
	encoded Float_Gob_Exponent_Encoding,
) (exponent Float_Exponent, status Validation_Status) {
	defer func() {
		Float_Exponent_Invariants(exponent, "float_gob_decode_exponent.exponent")
		Validation_Status_Invariants(status, "float_gob_decode_exponent.status")
	}()
	Float_Gob_Exponent_Encoding_Invariants(
		encoded, "float_gob_decode_exponent.encoded",
	)
	value := binary.Uint_32(binary.Bytes(encoded[:]), binary.BIG_ENDIAN)
	return Float_Exponent_Validate(Float_Exponent_Unvalidated(int32(value)))
}

func float_gob_decode_header(
	encoded Float_Gob_Header_Encoding,
) (header Float_Gob_Header, status Validation_Status) {
	defer func() {
		Float_Gob_Header_Invariants(header, "float_gob_decode_header.header")
		Validation_Status_Invariants(status, "float_gob_decode_header.status")
	}()
	Float_Gob_Header_Encoding_Invariants(
		encoded, "float_gob_decode_header.encoded",
	)
	attributes := encoded[FLOAT_GOB_ATTRIBUTES_OFFSET]
	mode_value := attributes >> FLOAT_GOB_MODE_SHIFT & FLOAT_GOB_MODE_MASK
	mode, validation := Rounding_Mode_Validate(Rounding_Mode_Unvalidated(mode_value))
	if validation != STATUS_OK {
		return Float_Gob_Header{}, STATUS_INPUT_INVALID
	}
	accuracy_value := int8(attributes>>FLOAT_GOB_ACCURACY_SHIFT&
		FLOAT_GOB_ACCURACY_MASK) - int8(WORD_COUNT_INCREMENT)
	if accuracy_value < int8(ACCURACY_BELOW) {
		return Float_Gob_Header{}, STATUS_INPUT_INVALID
	}
	if accuracy_value > int8(ACCURACY_ABOVE) {
		return Float_Gob_Header{}, STATUS_INPUT_INVALID
	}
	form := Float_Form(attributes >> FLOAT_GOB_FORM_SHIFT & FLOAT_GOB_FORM_MASK)
	if form > FLOAT_FORM_INFINITY {
		return Float_Gob_Header{}, STATUS_INPUT_INVALID
	}
	precision_end_offset := FLOAT_GOB_PRECISION_OFFSET + FLOAT_GOB_FIELD_SIZE
	precision_value := binary.Uint_32(
		binary.Bytes(encoded[FLOAT_GOB_PRECISION_OFFSET:precision_end_offset]),
		binary.BIG_ENDIAN,
	)
	if uint32(precision_value) > uint32(FLOAT_PRECISION_UNVALIDATED_MAXIMUM) {
		return Float_Gob_Header{}, STATUS_INPUT_INVALID
	}
	precision, validation := Float_Precision_Validate(
		Float_Precision_Unvalidated(precision_value),
	)
	if validation != STATUS_OK {
		return Float_Gob_Header{}, STATUS_INPUT_INVALID
	}
	return Float_Gob_Header{
		Precision: precision,
		Mode:      mode,
		Accuracy:  Accuracy(accuracy_value),
		Form:      form,
		Negative:  Polarity(attributes & FLOAT_GOB_NEGATIVE_MASK),
	}, STATUS_OK
}

func float_gob_result_set(
	result *Float, header Float_Gob_Header,
) (status Validation_Status) {
	defer func() {
		Validation_Status_Invariants(status, "float_gob_result_set.status")
	}()
	Float_Invariants(result, "float_gob_result_set.result_initial")
	Float_Gob_Header_Invariants(header, "float_gob_result_set.header")
	if header.Form == FLOAT_FORM_FINITE {
		if header.Precision == FLOAT_PRECISION_MINIMUM {
			return STATUS_INPUT_INVALID
		}
	}
	next := Float{
		Precision: header.Precision,
		Mode:      header.Mode,
		Accuracy:  header.Accuracy,
		Form:      header.Form,
		Negative:  header.Negative,
	}
	if header.Form == FLOAT_FORM_FINITE {
		Int_Set_Uint_64((*Int)(&next.Mantissa), Word_64(WORD_COUNT_INCREMENT))
	}
	*result = next
	return STATUS_OK
}

func float_gob_decode_commit(destination *Float, result *Float) {
	Float_Invariants(destination, "float_gob_decode_commit.destination_initial")
	Float_Invariants(result, "float_gob_decode_commit.result")
	if destination.Precision != FLOAT_PRECISION_MINIMUM {
		result.Mode = destination.Mode
		validation := Float_Set_Precision(
			result, Float_Precision_Unvalidated(destination.Precision),
		)
		invariant.Always(
			validation == Validation_Status(STATUS_OK),
			"Stored destination precision is already validated.",
		)
	}
	*destination = *result
}

func float_gob_decode_finite(
	mantissa *Float_Active_Mantissa, precision Float_Active_Precision,
) (status Validation_Status) {
	defer func() {
		Float_Active_Mantissa_Invariants(
			*mantissa, "float_gob_decode_finite.mantissa_result",
		)
		Validation_Status_Invariants(status, "float_gob_decode_finite.status")
	}()
	Float_Active_Mantissa_Invariants(*mantissa, "float_gob_decode_finite.mantissa")
	Float_Active_Precision_Invariants(precision, "float_gob_decode_finite.precision")
	integer := (*Int)(mantissa)
	zero_count := Int_Trailing_Zero_Bit_Count(integer)
	Int_Shift_Right(integer, integer, Shift_Count(zero_count))
	if Int_Bit_Count(integer) > Bit_Count(precision) {
		return STATUS_INPUT_INVALID
	}
	return STATUS_OK
}

// Int_Set_Float_64_Bits truncates finite binary64 toward zero.
func Int_Set_Float_64_Bits(
	destination *Int, encoding Float_64_Bits,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "int_set_float_64_bits.status") }()
	Int_Invariants(destination, "int_set_float_64_bits.destination")
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
		invariant.Always(
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
	value *Int,
) (encoding Int_Float_64_Value_Bits, accuracy Accuracy) {
	defer func() {
		Int_Float_64_Value_Bits_Invariants(encoding, "int_float_64_bits.encoding")
		Accuracy_Invariants(accuracy, "int_float_64_bits.accuracy")
	}()
	Int_Invariants(value, "int_float_64_bits.value")
	var number Float
	Float_Set_Int(&number, value)
	result, accuracy := Float_Float_64_Bits(&number)
	return Int_Float_64_Value_Bits(result), accuracy
}

// Int_Modular_Square_Root uses caller-supplied nonresidue to keep search bounded.
func Int_Modular_Square_Root(
	destination *Int,
	value *Int,
	modulus *Int,
	nonresidue *Int,
	workspace *Int_Modular_Square_Root_Workspace,
) (status Modular_Square_Root_Status) {
	defer func() {
		Modular_Square_Root_Status_Invariants(
			status, "int_modular_square_root.status",
		)
	}()
	Int_Invariants(destination, "int_modular_square_root.destination")
	Int_Invariants(value, "int_modular_square_root.value")
	Int_Invariants(modulus, "int_modular_square_root.modulus")
	Int_Invariants(nonresidue, "int_modular_square_root.nonresidue")
	Int_Modular_Square_Root_Workspace_Invariants(
		workspace, "int_modular_square_root.workspace",
	)
	integers := &workspace.Integers
	modulus_copy := &integers[MODULAR_SQUARE_ROOT_MODULUS_INDEX]
	nonresidue_copy := &integers[MODULAR_SQUARE_ROOT_NONRESIDUE_INDEX]
	*modulus_copy = *modulus
	*nonresidue_copy = *nonresidue
	if !int_modular_square_root_modulus_valid(modulus_copy) {
		return STATUS_INPUT_INVALID
	}
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
		int_modular_square_root_reduce(destination, value, workspace),
	)
}

func int_modular_square_root_modulus_valid(modulus *Int) (valid Boolean) {
	defer func() {
		Boolean_Invariants(valid, "int_modular_square_root_modulus_valid.valid")
	}()
	Int_Invariants(modulus, "int_modular_square_root_modulus_valid.modulus")
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
	destination *Int, value *Int, workspace *Int_Modular_Square_Root_Workspace,
) (status Modular_Square_Root_Result_Status) {
	defer func() {
		Modular_Square_Root_Result_Status_Invariants(
			status, "int_modular_square_root_reduce.status",
		)
	}()
	Int_Invariants(destination, "int_modular_square_root_reduce.destination")
	Int_Invariants(value, "int_modular_square_root_reduce.value")
	Int_Modular_Square_Root_Workspace_Invariants(
		workspace, "int_modular_square_root_reduce.workspace",
	)
	integers := &workspace.Integers
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
	invariant.Always(
		reduction == Divisor_Status(STATUS_OK),
		"Modular square-root reduction uses validated nonzero modulus.",
	)
	symbol, validation := int_jacobi(
		residue, modulus, &workspace.Jacobi, &workspace.Modular.Division,
	)
	workspace.Modular.Division.Quotient_Count = initial_quotient_count
	workspace.Modular.Division.Remainder_Count = initial_remainder_count
	invariant.Always(
		validation == Validation_Status(STATUS_OK),
		"Modular square-root residue uses validated positive odd modulus.",
	)
	if symbol == JACOBI_SYMBOL_NEGATIVE {
		return STATUS_RESULT_ABSENT
	}
	if symbol == JACOBI_SYMBOL_ZERO {
		*destination = Int{}
		return STATUS_OK
	}
	return int_modular_square_root_start(destination, workspace)
}

func int_modular_square_root_start(
	destination *Int, workspace *Int_Modular_Square_Root_Workspace,
) (status Modular_Square_Root_Result_Status) {
	defer func() {
		Modular_Square_Root_Result_Status_Invariants(
			status, "int_modular_square_root_start.status",
		)
	}()
	Int_Invariants(destination, "int_modular_square_root_start.destination")
	Int_Modular_Square_Root_Workspace_Invariants(
		workspace, "int_modular_square_root_start.workspace",
	)
	integers := &workspace.Integers
	modulus := &integers[MODULAR_SQUARE_ROOT_MODULUS_INDEX]
	odd_factor := &integers[MODULAR_SQUARE_ROOT_ODD_FACTOR_INDEX]
	exponent := &integers[MODULAR_SQUARE_ROOT_EXPONENT_INDEX]
	one := &integers[MODULAR_SQUARE_ROOT_ONE_INDEX]
	subtraction := Int_Subtract(odd_factor, modulus, one)
	invariant.Always(
		subtraction == Arithmetic_Status(STATUS_OK),
		"Validated modular square-root modulus exceeds one.",
	)
	order := Int_Trailing_Zero_Bit_Count(odd_factor)
	Int_Shift_Right(odd_factor, odd_factor, Shift_Count(order))
	addition := Int_Add(exponent, odd_factor, one)
	invariant.Always(
		addition == Arithmetic_Status(STATUS_OK),
		"Odd factor plus one cannot exceed validated modulus.",
	)
	Int_Shift_Right(exponent, exponent, Shift_Count(WORD_COUNT_INCREMENT))
	int_modular_square_root_powers(workspace)
	return int_modular_square_root_iterate(destination, workspace)
}

func int_modular_square_root_powers(workspace *Int_Modular_Square_Root_Workspace) {
	Int_Modular_Square_Root_Workspace_Invariants(
		workspace, "int_modular_square_root_powers.workspace",
	)
	integers := &workspace.Integers
	modulus := &integers[MODULAR_SQUARE_ROOT_MODULUS_INDEX]
	residue := &integers[MODULAR_SQUARE_ROOT_RESIDUE_INDEX]
	nonresidue := &integers[MODULAR_SQUARE_ROOT_NONRESIDUE_INDEX]
	odd_factor := &integers[MODULAR_SQUARE_ROOT_ODD_FACTOR_INDEX]
	exponent := &integers[MODULAR_SQUARE_ROOT_EXPONENT_INDEX]
	result := &integers[MODULAR_SQUARE_ROOT_RESULT_INDEX]
	power := &integers[MODULAR_SQUARE_ROOT_POWER_INDEX]
	generator := &integers[MODULAR_SQUARE_ROOT_GENERATOR_INDEX]
	initial_quotient_count := workspace.Modular.Division.Quotient_Count
	initial_remainder_count := workspace.Modular.Division.Remainder_Count
	status := Int_Modular_Exponent(
		result, residue, exponent, modulus,
		(*Int_Modular_Workspace)(&workspace.Modular),
	)
	invariant.Always(
		status == Modular_Status(STATUS_OK),
		"Tonelli-Shanks root exponent is nonnegative and modulus is nonzero.",
	)
	status = Int_Modular_Exponent(
		power, residue, odd_factor, modulus,
		(*Int_Modular_Workspace)(&workspace.Modular),
	)
	invariant.Always(
		status == Modular_Status(STATUS_OK),
		"Tonelli-Shanks power exponent is nonnegative and modulus is nonzero.",
	)
	status = Int_Modular_Exponent(
		generator, nonresidue, odd_factor, modulus,
		(*Int_Modular_Workspace)(&workspace.Modular),
	)
	workspace.Modular.Division.Quotient_Count = initial_quotient_count
	workspace.Modular.Division.Remainder_Count = initial_remainder_count
	invariant.Always(
		status == Modular_Status(STATUS_OK),
		"Tonelli-Shanks generator exponent is nonnegative and modulus is nonzero.",
	)
}

func int_modular_square_root_square_factor(
	workspace *Int_Modular_Square_Root_Workspace,
) {
	Int_Modular_Square_Root_Workspace_Invariants(
		workspace, "int_modular_square_root_square_factor.workspace",
	)
	integers := &workspace.Integers
	modulus := &integers[MODULAR_SQUARE_ROOT_MODULUS_INDEX]
	factor := &integers[MODULAR_SQUARE_ROOT_FACTOR_INDEX]
	initial_quotient_count := workspace.Modular.Division.Quotient_Count
	initial_remainder_count := workspace.Modular.Division.Remainder_Count
	status := Int_Modular_Multiply(
		factor, factor, factor, modulus,
		(*Int_Modular_Workspace)(&workspace.Modular),
	)
	workspace.Modular.Division.Quotient_Count = initial_quotient_count
	workspace.Modular.Division.Remainder_Count = initial_remainder_count
	invariant.Always(
		status == Divisor_Status(STATUS_OK),
		"Tonelli-Shanks factor square uses nonzero modulus.",
	)
}

func int_modular_square_root_advance(workspace *Int_Modular_Square_Root_Workspace) {
	Int_Modular_Square_Root_Workspace_Invariants(
		workspace, "int_modular_square_root_advance.workspace",
	)
	integers := &workspace.Integers
	modulus := &integers[MODULAR_SQUARE_ROOT_MODULUS_INDEX]
	exponent := &integers[MODULAR_SQUARE_ROOT_EXPONENT_INDEX]
	result := &integers[MODULAR_SQUARE_ROOT_RESULT_INDEX]
	power := &integers[MODULAR_SQUARE_ROOT_POWER_INDEX]
	generator := &integers[MODULAR_SQUARE_ROOT_GENERATOR_INDEX]
	factor := &integers[MODULAR_SQUARE_ROOT_FACTOR_INDEX]
	modular := (*Int_Modular_Workspace)(&workspace.Modular)
	initial_quotient_count := workspace.Modular.Division.Quotient_Count
	initial_remainder_count := workspace.Modular.Division.Remainder_Count
	status := Int_Modular_Exponent(factor, generator, exponent, modulus, modular)
	invariant.Always(
		status == Modular_Status(STATUS_OK),
		"Tonelli-Shanks factor exponent uses nonnegative exponent and nonzero modulus.",
	)
	product_status := Int_Modular_Multiply(generator, factor, factor, modulus, modular)
	invariant.Always(product_status == Divisor_Status(STATUS_OK),
		"Tonelli-Shanks generator square uses nonzero modulus.")
	product_status = Int_Modular_Multiply(result, result, factor, modulus, modular)
	invariant.Always(product_status == Divisor_Status(STATUS_OK),
		"Tonelli-Shanks root product uses nonzero modulus.")
	product_status = Int_Modular_Multiply(power, power, generator, modulus, modular)
	workspace.Modular.Division.Quotient_Count = initial_quotient_count
	workspace.Modular.Division.Remainder_Count = initial_remainder_count
	invariant.Always(product_status == Divisor_Status(STATUS_OK),
		"Tonelli-Shanks power product uses nonzero modulus.")
}

func int_modular_square_root_iterate(
	destination *Int, workspace *Int_Modular_Square_Root_Workspace,
) (status Modular_Square_Root_Result_Status) {
	defer func() {
		Modular_Square_Root_Result_Status_Invariants(
			status, "int_modular_square_root_iterate.status",
		)
	}()
	Int_Invariants(destination, "int_modular_square_root_iterate.destination")
	Int_Modular_Square_Root_Workspace_Invariants(
		workspace, "int_modular_square_root_iterate.workspace",
	)
	integers := &workspace.Integers
	modulus := &integers[MODULAR_SQUARE_ROOT_MODULUS_INDEX]
	exponent := &integers[MODULAR_SQUARE_ROOT_EXPONENT_INDEX]
	result := &integers[MODULAR_SQUARE_ROOT_RESULT_INDEX]
	power := &integers[MODULAR_SQUARE_ROOT_POWER_INDEX]
	factor := &integers[MODULAR_SQUARE_ROOT_FACTOR_INDEX]
	one := &integers[MODULAR_SQUARE_ROOT_ONE_INDEX]
	subtraction := Int_Subtract(exponent, modulus, one)
	invariant.Always(
		subtraction == Arithmetic_Status(STATUS_OK),
		"Validated modular square-root modulus exceeds one during iteration.",
	)
	limit := Int_Trailing_Zero_Bit_Count(exponent)
	order := int(limit)
	for attempt := BIT_COUNT_MINIMUM; attempt < int(limit); attempt++ {
		*factor = *power
		distance := BIT_COUNT_MINIMUM
		for distance < order && Int_Compare(factor, one) != ORDER_SAME {
			int_modular_square_root_square_factor(workspace)
			distance++
		}
		if Int_Compare(factor, one) != ORDER_SAME {
			return STATUS_RESULT_ABSENT
		}
		if distance == BIT_COUNT_MINIMUM {
			residue := &integers[MODULAR_SQUARE_ROOT_RESIDUE_INDEX]
			check := &integers[MODULAR_SQUARE_ROOT_CHECK_INDEX]
			initial_quotient_count := workspace.Modular.Division.Quotient_Count
			initial_remainder_count := workspace.Modular.Division.Remainder_Count
			product_status := Int_Modular_Multiply(
				check, result, result, modulus,
				(*Int_Modular_Workspace)(&workspace.Modular),
			)
			workspace.Modular.Division.Quotient_Count = initial_quotient_count
			workspace.Modular.Division.Remainder_Count = initial_remainder_count
			invariant.Always(product_status == Divisor_Status(STATUS_OK),
				"Tonelli-Shanks root check uses nonzero modulus.")
			if Int_Compare(check, residue) != ORDER_SAME {
				return STATUS_RESULT_ABSENT
			}
			*destination = *result
			return STATUS_OK
		}
		if distance >= order {
			return STATUS_RESULT_ABSENT
		}
		Int_Set_Uint_64(exponent, Word_64(WORD_COUNT_INCREMENT))
		shift := order - distance - WORD_COUNT_INCREMENT
		shift_status := Int_Shift_Left(exponent, exponent, Shift_Count(shift))
		invariant.Always(
			shift_status == Arithmetic_Status(STATUS_OK),
			"Tonelli-Shanks subgroup exponent fits modulus bit width.",
		)
		int_modular_square_root_advance(workspace)
		order = distance
	}
	return STATUS_RESULT_ABSENT
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

// Float_Text_Into writes one complete bounded stdlib representation transactionally.
func Float_Text_Into(
	destination Text,
	value *Float,
	format_unvalidated Float_Text_Format_Unvalidated,
	precision_unvalidated Float_Text_Precision_Unvalidated,
	workspace *Float_Text_Workspace,
) (count Float_Text_Count, status Float_Text_Status) {
	defer func() {
		Float_Text_Count_Invariants(count, "float_text_into.count")
		Float_Text_Status_Invariants(status, "float_text_into.status")
	}()
	Text_Invariants(destination, "float_text_into.destination")
	Float_Invariants(value, "float_text_into.value")
	Float_Text_Format_Unvalidated_Invariants(
		format_unvalidated, "float_text_into.format",
	)
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
	workspace.Values[FLOAT_TEXT_SOURCE_INDEX] = *value
	workspace.Control[FLOAT_TEXT_PRECISION_INDEX] = int(precision)
	workspace.Control[FLOAT_TEXT_FORMAT_INDEX] = int(format)
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
	count = Float_Text_Count(workspace.Control[FLOAT_TEXT_OUTPUT_COUNT_INDEX])
	if len(destination) < int(count) {
		return count, STATUS_DESTINATION_TOO_SMALL
	}
	copy(destination[:count], workspace.Output[:count])
	return count, STATUS_OK
}

func float_text_binary(workspace *Float_Text_Workspace) {
	Float_Text_Workspace_Invariants(workspace, "float_text_binary.workspace")
	value := &workspace.Values[FLOAT_TEXT_SOURCE_INDEX]
	float_text_sign(workspace)
	if value.Form == FLOAT_FORM_INFINITY {
		float_text_infinity(workspace)
		return
	}
	output_index := workspace.Control[FLOAT_TEXT_OUTPUT_COUNT_INDEX]
	if value.Form == FLOAT_FORM_ZERO {
		workspace.Output[output_index] = '0'
		workspace.Control[FLOAT_TEXT_OUTPUT_COUNT_INDEX] =
			output_index + WORD_COUNT_INCREMENT
		return
	}
	integer := &workspace.Integers[FLOAT_TEXT_INTEGER_INDEX]
	*integer = Int(value.Mantissa)
	bit_count := int(Int_Bit_Count(integer))
	shift := int(value.Precision) - bit_count
	shift_status := Int_Shift_Left(integer, integer, Shift_Count(shift))
	invariant.Always(
		shift_status == Arithmetic_Status(STATUS_OK),
		"Binary Float text aligns only inside active precision.",
	)
	text := &workspace.Text[FLOAT_TEXT_MANTISSA_WORKSPACE_INDEX]
	digit_count := int(int_text_digits(integer, BASE_DECIMAL, text))
	for index := digit_count - WORD_COUNT_INCREMENT; index >= WORD_COUNT_MINIMUM; index-- {
		workspace.Output[output_index] = text.Digits[index]
		output_index++
	}
	exponent := &workspace.Integers[FLOAT_TEXT_EXPONENT_INDEX]
	Int_Set_Int_64(
		exponent, Int_64(int(value.Exponent)-int(value.Precision)),
	)
	workspace.Control[FLOAT_TEXT_OUTPUT_COUNT_INDEX] = output_index
	workspace.Control[FLOAT_TEXT_EXPONENT_MINIMUM_DIGIT_COUNT_INDEX] =
		SIGN_BYTE_COUNT_MAXIMUM
	float_text_write_exponent(workspace)
}

func float_text_hexadecimal_fraction(workspace *Float_Text_Workspace) {
	Float_Text_Workspace_Invariants(
		workspace, "float_text_hexadecimal_fraction.workspace",
	)
	value := &workspace.Values[FLOAT_TEXT_SOURCE_INDEX]
	float_text_sign(workspace)
	if value.Form == FLOAT_FORM_INFINITY {
		float_text_infinity(workspace)
		return
	}
	output_index := workspace.Control[FLOAT_TEXT_OUTPUT_COUNT_INDEX]
	if value.Form == FLOAT_FORM_ZERO {
		workspace.Output[output_index] = '0'
		workspace.Control[FLOAT_TEXT_OUTPUT_COUNT_INDEX] =
			output_index + WORD_COUNT_INCREMENT
		return
	}
	integer := &workspace.Integers[FLOAT_TEXT_INTEGER_INDEX]
	*integer = Int(value.Mantissa)
	bit_count := int(Int_Bit_Count(integer))
	shift := (BASE_HEXADECIMAL_DIGIT_BIT_COUNT -
		bit_count%BASE_HEXADECIMAL_DIGIT_BIT_COUNT) %
		BASE_HEXADECIMAL_DIGIT_BIT_COUNT
	shift_status := Int_Shift_Left(integer, integer, Shift_Count(shift))
	invariant.Always(
		shift_status == Arithmetic_Status(STATUS_OK),
		"Hexadecimal fraction alignment adds fewer than one digit.",
	)
	text := &workspace.Text[FLOAT_TEXT_MANTISSA_WORKSPACE_INDEX]
	digit_count := int(int_text_digits(integer, BASE_HEXADECIMAL, text))
	low_index := WORD_COUNT_MINIMUM
	for low_index+WORD_COUNT_INCREMENT < digit_count {
		if text.Digits[low_index] != '0' {
			break
		}
		low_index++
	}
	workspace.Output[output_index] = '0'
	workspace.Output[output_index+WORD_COUNT_INCREMENT] = 'x'
	workspace.Output[output_index+BASE_PREFIX_BYTE_COUNT] = '.'
	output_index += BASE_PREFIX_BYTE_COUNT + DECIMAL_POINT_BYTE_COUNT
	for index := digit_count - WORD_COUNT_INCREMENT; index >= low_index; index-- {
		workspace.Output[output_index] = text.Digits[index]
		output_index++
	}
	exponent := &workspace.Integers[FLOAT_TEXT_EXPONENT_INDEX]
	Int_Set_Int_64(exponent, Int_64(value.Exponent))
	workspace.Control[FLOAT_TEXT_OUTPUT_COUNT_INDEX] = output_index
	workspace.Control[FLOAT_TEXT_EXPONENT_MINIMUM_DIGIT_COUNT_INDEX] =
		SIGN_BYTE_COUNT_MAXIMUM
	float_text_write_exponent(workspace)
}

func float_text_hexadecimal(workspace *Float_Text_Workspace) {
	Float_Text_Workspace_Invariants(workspace, "float_text_hexadecimal.workspace")
	value := &workspace.Values[FLOAT_TEXT_SOURCE_INDEX]
	precision := Float_Text_Precision(workspace.Control[FLOAT_TEXT_PRECISION_INDEX])
	float_text_sign(workspace)
	if value.Form == FLOAT_FORM_INFINITY {
		float_text_infinity(workspace)
		return
	}
	if value.Form == FLOAT_FORM_ZERO {
		float_text_hexadecimal_zero(workspace)
		return
	}
	rounded := &workspace.Values[FLOAT_TEXT_ROUNDED_INDEX]
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
	invariant.Always(
		validation == Validation_Status(STATUS_OK),
		"Hexadecimal text target precision was bounded before conversion.",
	)
	integer := &workspace.Integers[FLOAT_TEXT_INTEGER_INDEX]
	*integer = Int(rounded.Mantissa)
	bit_count := int(Int_Bit_Count(integer))
	shift_status := Int_Shift_Left(
		integer, integer, Shift_Count(target_precision-bit_count),
	)
	invariant.Always(
		shift_status == Arithmetic_Status(STATUS_OK),
		"Rounded hexadecimal significand fits target precision.",
	)
	float_text_hexadecimal_finite(workspace)
}

func float_text_hexadecimal_finite(workspace *Float_Text_Workspace) {
	Float_Text_Workspace_Invariants(
		workspace, "float_text_hexadecimal_finite.workspace",
	)
	value := &workspace.Values[FLOAT_TEXT_ROUNDED_INDEX]
	integer := &workspace.Integers[FLOAT_TEXT_INTEGER_INDEX]
	text := &workspace.Text[FLOAT_TEXT_MANTISSA_WORKSPACE_INDEX]
	digit_count := int(int_text_digits(integer, BASE_HEXADECIMAL, text))
	index := workspace.Control[FLOAT_TEXT_OUTPUT_COUNT_INDEX]
	workspace.Output[index] = '0'
	workspace.Output[index+WORD_COUNT_INCREMENT] = 'x'
	index += BASE_PREFIX_BYTE_COUNT
	workspace.Output[index] = text.Digits[digit_count-WORD_COUNT_INCREMENT]
	index++
	if digit_count > WORD_COUNT_INCREMENT {
		workspace.Output[index] = '.'
		index++
		for digit_index := digit_count - BASE_BINARY; digit_index >= 0; digit_index-- {
			workspace.Output[index] = text.Digits[digit_index]
			index++
		}
	}
	exponent := &workspace.Integers[FLOAT_TEXT_EXPONENT_INDEX]
	Int_Set_Int_64(exponent, Int_64(int(value.Exponent)-WORD_COUNT_INCREMENT))
	workspace.Control[FLOAT_TEXT_OUTPUT_COUNT_INDEX] = index
	workspace.Control[FLOAT_TEXT_EXPONENT_MINIMUM_DIGIT_COUNT_INDEX] = BASE_BINARY
	float_text_write_exponent(workspace)
}

func float_text_hexadecimal_zero(workspace *Float_Text_Workspace) {
	Float_Text_Workspace_Invariants(
		workspace, "float_text_hexadecimal_zero.workspace",
	)
	precision := Float_Text_Precision(workspace.Control[FLOAT_TEXT_PRECISION_INDEX])
	index := workspace.Control[FLOAT_TEXT_OUTPUT_COUNT_INDEX]
	workspace.Output[index] = '0'
	workspace.Output[index+WORD_COUNT_INCREMENT] = 'x'
	workspace.Output[index+BASE_PREFIX_BYTE_COUNT] = '0'
	index += BASE_PREFIX_BYTE_COUNT + WORD_COUNT_INCREMENT
	if precision > 0 {
		workspace.Output[index] = '.'
		index++
		for digit := WORD_COUNT_MINIMUM; digit < int(precision); digit++ {
			workspace.Output[index] = '0'
			index++
		}
	}
	Int_Set_Uint_64(
		&workspace.Integers[FLOAT_TEXT_EXPONENT_INDEX],
		Word_64(bytes.SLICE_SIZE_MINIMUM),
	)
	workspace.Control[FLOAT_TEXT_OUTPUT_COUNT_INDEX] = index
	workspace.Control[FLOAT_TEXT_EXPONENT_MINIMUM_DIGIT_COUNT_INDEX] = BASE_BINARY
	float_text_write_exponent(workspace)
}

func float_text_sign(workspace *Float_Text_Workspace) {
	Float_Text_Workspace_Invariants(workspace, "float_text_sign.workspace")
	value := &workspace.Values[FLOAT_TEXT_SOURCE_INDEX]
	workspace.Control[FLOAT_TEXT_OUTPUT_COUNT_INDEX] = bytes.SLICE_SIZE_MINIMUM
	if value.Negative == POLARITY_NEGATIVE {
		workspace.Output[WORD_COUNT_MINIMUM] = '-'
		workspace.Control[FLOAT_TEXT_OUTPUT_COUNT_INDEX] = SIGN_BYTE_COUNT_MAXIMUM
	}
}

func float_text_infinity(workspace *Float_Text_Workspace) {
	Float_Text_Workspace_Invariants(workspace, "float_text_infinity.workspace")
	value := &workspace.Values[FLOAT_TEXT_SOURCE_INDEX]
	index := bytes.SLICE_SIZE_MINIMUM
	if value.Negative == POLARITY_NEGATIVE {
		workspace.Output[index] = '-'
	} else {
		workspace.Output[index] = '+'
	}
	index++
	workspace.Output[index] = 'I'
	workspace.Output[index+WORD_COUNT_INCREMENT] = 'n'
	workspace.Output[index+BASE_BINARY] = 'f'
	workspace.Control[FLOAT_TEXT_OUTPUT_COUNT_INDEX] =
		index + BASE_BINARY + WORD_COUNT_INCREMENT
}

func float_text_write_exponent(workspace *Float_Text_Workspace) {
	Float_Text_Workspace_Invariants(workspace, "float_text_write_exponent.workspace")
	index := workspace.Control[FLOAT_TEXT_OUTPUT_COUNT_INDEX]
	format := Float_Text_Format(workspace.Control[FLOAT_TEXT_FORMAT_INDEX])
	marker := byte('p')
	if format == FLOAT_TEXT_KIND_DECIMAL_EXPONENT {
		marker = 'e'
	}
	if format == FLOAT_TEXT_KIND_DECIMAL_GENERAL {
		marker = 'e'
	}
	if format == FLOAT_TEXT_KIND_DECIMAL_EXPONENT_UPPER {
		marker = 'E'
	}
	if format == FLOAT_TEXT_KIND_DECIMAL_GENERAL_UPPER {
		marker = 'E'
	}
	workspace.Output[index] = marker
	index++
	exponent := &workspace.Integers[FLOAT_TEXT_EXPONENT_INDEX]
	if exponent.Negative == POLARITY_NEGATIVE {
		workspace.Output[index] = '-'
	} else {
		workspace.Output[index] = '+'
	}
	index++
	text := &workspace.Text[FLOAT_TEXT_EXPONENT_WORKSPACE_INDEX]
	digit_count := int(int_text_digits(exponent, BASE_DECIMAL, text))
	minimum_digits := workspace.Control[FLOAT_TEXT_EXPONENT_MINIMUM_DIGIT_COUNT_INDEX]
	for digit_count < minimum_digits {
		workspace.Output[index] = '0'
		index++
		minimum_digits--
	}
	for digit_index := digit_count - WORD_COUNT_INCREMENT; digit_index >= 0; digit_index-- {
		workspace.Output[index] = text.Digits[digit_index]
		index++
	}
	workspace.Control[FLOAT_TEXT_OUTPUT_COUNT_INDEX] = index
}

func float_text_decimal(workspace *Float_Text_Workspace) {
	Float_Text_Workspace_Invariants(workspace, "float_text_decimal.workspace")
	value := &workspace.Values[FLOAT_TEXT_SOURCE_INDEX]
	float_text_sign(workspace)
	if value.Form == FLOAT_FORM_INFINITY {
		float_text_infinity(workspace)
		return
	}
	float_decimal_initialize(workspace)
	format := Float_Text_Format(workspace.Control[FLOAT_TEXT_FORMAT_INDEX])
	precision := workspace.Control[FLOAT_TEXT_PRECISION_INDEX]
	shortest := Boolean(precision == FLOAT_TEXT_PRECISION_MINIMUM)
	if shortest {
		float_decimal_round_shortest(workspace)
	}
	decimal := &workspace.Decimals[FLOAT_DECIMAL_VALUE_INDEX]
	digit_count := decimal.Control[FLOAT_DECIMAL_DIGIT_COUNT_INDEX]
	exponent := decimal.Control[FLOAT_DECIMAL_EXPONENT_INDEX]
	if format == FLOAT_TEXT_KIND_DECIMAL_EXPONENT {
		float_text_decimal_prepare_exponent(workspace, shortest)
		float_text_decimal_exponent(workspace)
		return
	}
	if format == FLOAT_TEXT_KIND_DECIMAL_EXPONENT_UPPER {
		float_text_decimal_prepare_exponent(workspace, shortest)
		float_text_decimal_exponent(workspace)
		return
	}
	if format == FLOAT_TEXT_KIND_DECIMAL_FIXED {
		if shortest {
			precision = digit_count - exponent
			if precision < 0 {
				precision = 0
			}
			workspace.Control[FLOAT_TEXT_PRECISION_INDEX] = precision
		} else {
			workspace.Control[FLOAT_TEXT_ROUND_DIGIT_COUNT_INDEX] = exponent + precision
			float_decimal_round(workspace)
		}
		float_text_decimal_fixed(workspace)
		return
	}
	float_text_decimal_general(workspace, shortest)
}

func float_text_decimal_prepare_exponent(
	workspace *Float_Text_Workspace, shortest Boolean,
) {
	Float_Text_Workspace_Invariants(
		workspace, "float_text_decimal_prepare_exponent.workspace",
	)
	Boolean_Invariants(shortest, "float_text_decimal_prepare_exponent.shortest")
	decimal := &workspace.Decimals[FLOAT_DECIMAL_VALUE_INDEX]
	precision := workspace.Control[FLOAT_TEXT_PRECISION_INDEX]
	if shortest {
		precision = decimal.Control[FLOAT_DECIMAL_DIGIT_COUNT_INDEX] -
			WORD_COUNT_INCREMENT
		if precision < 0 {
			precision = 0
		}
		workspace.Control[FLOAT_TEXT_PRECISION_INDEX] = precision
		return
	}
	workspace.Control[FLOAT_TEXT_ROUND_DIGIT_COUNT_INDEX] =
		precision + WORD_COUNT_INCREMENT
	float_decimal_round(workspace)
}

func float_text_decimal_general(
	workspace *Float_Text_Workspace, shortest Boolean,
) {
	Float_Text_Workspace_Invariants(
		workspace, "float_text_decimal_general.workspace",
	)
	Boolean_Invariants(shortest, "float_text_decimal_general.shortest")
	decimal := &workspace.Decimals[FLOAT_DECIMAL_VALUE_INDEX]
	precision := workspace.Control[FLOAT_TEXT_PRECISION_INDEX]
	if shortest {
		precision = decimal.Control[FLOAT_DECIMAL_DIGIT_COUNT_INDEX]
		workspace.Control[FLOAT_TEXT_PRECISION_INDEX] = precision
	} else {
		if precision == 0 {
			precision = WORD_COUNT_INCREMENT
			workspace.Control[FLOAT_TEXT_PRECISION_INDEX] = precision
		}
		workspace.Control[FLOAT_TEXT_ROUND_DIGIT_COUNT_INDEX] = precision
		float_decimal_round(workspace)
	}
	digit_count := decimal.Control[FLOAT_DECIMAL_DIGIT_COUNT_INDEX]
	exponent := decimal.Control[FLOAT_DECIMAL_EXPONENT_INDEX]
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
		workspace.Control[FLOAT_TEXT_PRECISION_INDEX] = precision - WORD_COUNT_INCREMENT
		float_text_decimal_exponent(workspace)
		return
	}
	if decimal_exponent >= exponent_precision {
		workspace.Control[FLOAT_TEXT_PRECISION_INDEX] = precision - WORD_COUNT_INCREMENT
		float_text_decimal_exponent(workspace)
		return
	}
	if precision > exponent {
		precision = digit_count
	}
	precision -= exponent
	if precision < 0 {
		precision = 0
	}
	workspace.Control[FLOAT_TEXT_PRECISION_INDEX] = precision
	float_text_decimal_fixed(workspace)
}

func float_text_decimal_exponent(workspace *Float_Text_Workspace) {
	Float_Text_Workspace_Invariants(
		workspace, "float_text_decimal_exponent.workspace",
	)
	decimal := &workspace.Decimals[FLOAT_DECIMAL_VALUE_INDEX]
	digit_count := decimal.Control[FLOAT_DECIMAL_DIGIT_COUNT_INDEX]
	precision := workspace.Control[FLOAT_TEXT_PRECISION_INDEX]
	index := workspace.Control[FLOAT_TEXT_OUTPUT_COUNT_INDEX]
	digit := byte('0')
	if digit_count > 0 {
		digit = decimal.Digits[WORD_COUNT_MINIMUM]
	}
	workspace.Output[index] = digit
	index++
	if precision > 0 {
		workspace.Output[index] = '.'
		index++
		for digit_index := WORD_COUNT_INCREMENT; digit_index <= precision; digit_index++ {
			digit = '0'
			if digit_index < digit_count {
				digit = decimal.Digits[digit_index]
			}
			workspace.Output[index] = digit
			index++
		}
	}
	exponent := bytes.SLICE_SIZE_MINIMUM
	if digit_count > 0 {
		exponent = decimal.Control[FLOAT_DECIMAL_EXPONENT_INDEX] - WORD_COUNT_INCREMENT
	}
	Int_Set_Int_64(
		&workspace.Integers[FLOAT_TEXT_EXPONENT_INDEX], Int_64(exponent),
	)
	workspace.Control[FLOAT_TEXT_OUTPUT_COUNT_INDEX] = index
	workspace.Control[FLOAT_TEXT_EXPONENT_MINIMUM_DIGIT_COUNT_INDEX] = BASE_BINARY
	float_text_write_exponent(workspace)
}

func float_text_decimal_fixed(workspace *Float_Text_Workspace) {
	Float_Text_Workspace_Invariants(
		workspace, "float_text_decimal_fixed.workspace",
	)
	decimal := &workspace.Decimals[FLOAT_DECIMAL_VALUE_INDEX]
	digit_count := decimal.Control[FLOAT_DECIMAL_DIGIT_COUNT_INDEX]
	exponent := decimal.Control[FLOAT_DECIMAL_EXPONENT_INDEX]
	precision := workspace.Control[FLOAT_TEXT_PRECISION_INDEX]
	index := workspace.Control[FLOAT_TEXT_OUTPUT_COUNT_INDEX]
	if exponent > 0 {
		copied := digit_count
		if copied > exponent {
			copied = exponent
		}
		for digit_index := WORD_COUNT_MINIMUM; digit_index < copied; digit_index++ {
			workspace.Output[index] = decimal.Digits[digit_index]
			index++
		}
		for digit_index := copied; digit_index < exponent; digit_index++ {
			workspace.Output[index] = '0'
			index++
		}
	} else {
		workspace.Output[index] = '0'
		index++
	}
	if precision > 0 {
		workspace.Output[index] = '.'
		index++
		for digit_index := WORD_COUNT_MINIMUM; digit_index < precision; digit_index++ {
			source_index := exponent + digit_index
			digit := byte('0')
			if source_index >= WORD_COUNT_MINIMUM {
				if source_index < digit_count {
					digit = decimal.Digits[source_index]
				}
			}
			workspace.Output[index] = digit
			index++
		}
	}
	invariant.Always(
		index <= FLOAT_TEXT_SIZE_MAXIMUM,
		"Fixed Float text fits its exact derived output bound.",
	)
	workspace.Control[FLOAT_TEXT_OUTPUT_COUNT_INDEX] = index
}

func float_decimal_initialize(workspace *Float_Text_Workspace) {
	Float_Text_Workspace_Invariants(workspace, "float_decimal_initialize.workspace")
	workspace.Control[FLOAT_TEXT_DECIMAL_INDEX] = FLOAT_DECIMAL_VALUE_INDEX
	decimal := &workspace.Decimals[FLOAT_DECIMAL_VALUE_INDEX]
	decimal.Control = [FLOAT_DECIMAL_CONTROL_COUNT]int{}
	value := &workspace.Values[FLOAT_TEXT_SOURCE_INDEX]
	if value.Form == FLOAT_FORM_ZERO {
		return
	}
	integer := &workspace.Integers[FLOAT_TEXT_INTEGER_INDEX]
	*integer = Int(value.Mantissa)
	bit_count := int(Int_Bit_Count(integer))
	shift := int(value.Exponent) - bit_count
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
		invariant.Always(
			shift_status == Arithmetic_Status(STATUS_OK),
			"Positive Float decimal shift cannot exceed normalized exponent.",
		)
		shift = 0
	}
	text := &workspace.Text[FLOAT_TEXT_MANTISSA_WORKSPACE_INDEX]
	digit_count := int(int_text_digits(integer, BASE_DECIMAL, text))
	decimal.Control[FLOAT_DECIMAL_DIGIT_COUNT_INDEX] = digit_count
	decimal.Control[FLOAT_DECIMAL_EXPONENT_INDEX] = digit_count
	source_index := digit_count - WORD_COUNT_INCREMENT
	destination_index := WORD_COUNT_MINIMUM
	for destination_index < digit_count {
		decimal.Digits[destination_index] = text.Digits[source_index]
		source_index--
		destination_index++
	}
	float_decimal_trim(workspace)
	for shift < -FLOAT_DECIMAL_SHIFT_MAXIMUM {
		workspace.Control[FLOAT_TEXT_DECIMAL_SHIFT_INDEX] = FLOAT_DECIMAL_SHIFT_MAXIMUM
		float_decimal_shift_right(workspace)
		shift += FLOAT_DECIMAL_SHIFT_MAXIMUM
	}
	if shift < 0 {
		workspace.Control[FLOAT_TEXT_DECIMAL_SHIFT_INDEX] = -shift
		float_decimal_shift_right(workspace)
	}
}

func float_decimal_shift_right(workspace *Float_Text_Workspace) {
	Float_Text_Workspace_Invariants(workspace, "float_decimal_shift_right.workspace")
	decimal_index := workspace.Control[FLOAT_TEXT_DECIMAL_INDEX]
	decimal := &workspace.Decimals[decimal_index]
	shift := uint(workspace.Control[FLOAT_TEXT_DECIMAL_SHIFT_INDEX])
	count := decimal.Control[FLOAT_DECIMAL_DIGIT_COUNT_INDEX]
	read_index := WORD_COUNT_MINIMUM
	number := Word(bits.WORD_64_MINIMUM)
	for number>>shift == 0 && read_index < count {
		number = number*Word(BASE_DECIMAL) + Word(decimal.Digits[read_index]-'0')
		read_index++
	}
	if number == 0 {
		decimal.Control[FLOAT_DECIMAL_DIGIT_COUNT_INDEX] = bytes.SLICE_SIZE_MINIMUM
		return
	}
	for number>>shift == 0 {
		read_index++
		number *= Word(BASE_DECIMAL)
	}
	decimal.Control[FLOAT_DECIMAL_EXPONENT_INDEX] +=
		WORD_COUNT_INCREMENT - read_index
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
		invariant.Always(
			write_index < FLOAT_DECIMAL_DIGIT_COUNT_MAXIMUM,
			"Exact Float decimal right shift fits derived digit capacity.",
		)
		digit := number >> shift
		number &= mask
		decimal.Digits[write_index] = byte(digit) + '0'
		write_index++
		number *= Word(BASE_DECIMAL)
	}
	decimal.Control[FLOAT_DECIMAL_DIGIT_COUNT_INDEX] = write_index
	float_decimal_trim(workspace)
}

func float_decimal_round(workspace *Float_Text_Workspace) {
	Float_Text_Workspace_Invariants(workspace, "float_decimal_round.workspace")
	decimal_index := workspace.Control[FLOAT_TEXT_DECIMAL_INDEX]
	decimal := &workspace.Decimals[decimal_index]
	retained := workspace.Control[FLOAT_TEXT_ROUND_DIGIT_COUNT_INDEX]
	count := decimal.Control[FLOAT_DECIMAL_DIGIT_COUNT_INDEX]
	if retained < 0 {
		return
	}
	if retained >= count {
		return
	}
	if float_decimal_should_round_up(workspace) {
		float_decimal_round_up(workspace)
		return
	}
	decimal.Control[FLOAT_DECIMAL_DIGIT_COUNT_INDEX] = retained
	float_decimal_trim(workspace)
}

func float_decimal_should_round_up(
	workspace *Float_Text_Workspace,
) (round_up Boolean) {
	defer func() {
		Boolean_Invariants(round_up, "float_decimal_should_round_up.round_up")
	}()
	Float_Text_Workspace_Invariants(
		workspace, "float_decimal_should_round_up.workspace",
	)
	decimal_index := workspace.Control[FLOAT_TEXT_DECIMAL_INDEX]
	decimal := &workspace.Decimals[decimal_index]
	retained := workspace.Control[FLOAT_TEXT_ROUND_DIGIT_COUNT_INDEX]
	count := decimal.Control[FLOAT_DECIMAL_DIGIT_COUNT_INDEX]
	digit := decimal.Digits[retained]
	if digit == '5' {
		if retained+WORD_COUNT_INCREMENT == count {
			if retained == 0 {
				return false
			}
			return Boolean((decimal.Digits[retained-WORD_COUNT_INCREMENT]-'0')&
				byte(bits.CARRY_MAXIMUM) != 0)
		}
	}
	return Boolean(digit >= '5')
}

func float_decimal_round_up(workspace *Float_Text_Workspace) {
	Float_Text_Workspace_Invariants(workspace, "float_decimal_round_up.workspace")
	decimal_index := workspace.Control[FLOAT_TEXT_DECIMAL_INDEX]
	decimal := &workspace.Decimals[decimal_index]
	retained := workspace.Control[FLOAT_TEXT_ROUND_DIGIT_COUNT_INDEX]
	for index := retained - WORD_COUNT_INCREMENT; index >= WORD_COUNT_MINIMUM; index-- {
		if decimal.Digits[index] < '9' {
			decimal.Digits[index]++
			decimal.Control[FLOAT_DECIMAL_DIGIT_COUNT_INDEX] = index +
				WORD_COUNT_INCREMENT
			float_decimal_trim(workspace)
			return
		}
	}
	decimal.Digits[WORD_COUNT_MINIMUM] = '1'
	decimal.Control[FLOAT_DECIMAL_DIGIT_COUNT_INDEX] = WORD_COUNT_INCREMENT
	decimal.Control[FLOAT_DECIMAL_EXPONENT_INDEX]++
}

func float_decimal_trim(workspace *Float_Text_Workspace) {
	Float_Text_Workspace_Invariants(workspace, "float_decimal_trim.workspace")
	decimal_index := workspace.Control[FLOAT_TEXT_DECIMAL_INDEX]
	decimal := &workspace.Decimals[decimal_index]
	count := decimal.Control[FLOAT_DECIMAL_DIGIT_COUNT_INDEX]
	for count > 0 {
		if decimal.Digits[count-WORD_COUNT_INCREMENT] != '0' {
			break
		}
		count--
	}
	decimal.Control[FLOAT_DECIMAL_DIGIT_COUNT_INDEX] = count
}

func float_decimal_round_shortest(workspace *Float_Text_Workspace) {
	Float_Text_Workspace_Invariants(workspace, "float_decimal_round_shortest.workspace")
	value_decimal := &workspace.Decimals[FLOAT_DECIMAL_VALUE_INDEX]
	if value_decimal.Control[FLOAT_DECIMAL_DIGIT_COUNT_INDEX] == 0 {
		return
	}
	float_text_midpoint_load(workspace)
	inclusive := workspace.Magnitude.Words[WORD_COUNT_MINIMUM]&Word(BASE_BINARY) == 0
	workspace.Magnitude.Words[WORD_COUNT_MINIMUM]--
	workspace.Control[FLOAT_TEXT_DECIMAL_INDEX] = FLOAT_DECIMAL_LOWER_INDEX
	float_decimal_initialize_magnitude(workspace)
	float_text_midpoint_load(workspace)
	workspace.Magnitude.Words[WORD_COUNT_MINIMUM]++
	workspace.Control[FLOAT_TEXT_DECIMAL_INDEX] = FLOAT_DECIMAL_UPPER_INDEX
	float_decimal_initialize_magnitude(workspace)
	lower := &workspace.Decimals[FLOAT_DECIMAL_LOWER_INDEX]
	upper := &workspace.Decimals[FLOAT_DECIMAL_UPPER_INDEX]
	lower_count := lower.Control[FLOAT_DECIMAL_DIGIT_COUNT_INDEX]
	upper_count := upper.Control[FLOAT_DECIMAL_DIGIT_COUNT_INDEX]
	value_count := value_decimal.Control[FLOAT_DECIMAL_DIGIT_COUNT_INDEX]
	workspace.Control[FLOAT_TEXT_DECIMAL_INDEX] = FLOAT_DECIMAL_VALUE_INDEX
	for digit_index := WORD_COUNT_MINIMUM; digit_index < value_count; digit_index++ {
		digit := value_decimal.Digits[digit_index]
		lower_digit := byte('0')
		if digit_index < lower_count {
			lower_digit = lower.Digits[digit_index]
		}
		upper_digit := byte('0')
		if digit_index < upper_count {
			upper_digit = upper.Digits[digit_index]
		}
		down := lower_digit != digit
		if !down {
			if inclusive {
				if digit_index+WORD_COUNT_INCREMENT == lower_count {
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
			if digit_index+WORD_COUNT_INCREMENT < upper_count {
				greater = true
			}
			up = greater
		}
		retained := digit_index + WORD_COUNT_INCREMENT
		workspace.Control[FLOAT_TEXT_ROUND_DIGIT_COUNT_INDEX] = retained
		if down {
			if up {
				float_decimal_round(workspace)
				return
			}
			value_decimal.Control[FLOAT_DECIMAL_DIGIT_COUNT_INDEX] = retained
			float_decimal_trim(workspace)
			return
		}
		if up {
			float_decimal_round_up(workspace)
			return
		}
	}
}

func float_text_midpoint_load(workspace *Float_Text_Workspace) {
	Float_Text_Workspace_Invariants(workspace, "float_text_midpoint_load.workspace")
	value := &workspace.Values[FLOAT_TEXT_SOURCE_INDEX]
	mantissa := (*Int)(&value.Mantissa)
	bit_count := int(Int_Bit_Count(mantissa))
	guarded_bit_count := int(value.Precision) + WORD_COUNT_INCREMENT
	shift_count := guarded_bit_count - bit_count
	magnitude := &workspace.Magnitude
	magnitude.Words = [FLOAT_TEXT_MAGNITUDE_WORD_COUNT_MAXIMUM]Word{}
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
	magnitude.Control[FLOAT_TEXT_MAGNITUDE_WORD_COUNT_INDEX] = word_count
	workspace.Control[FLOAT_TEXT_BINARY_SHIFT_INDEX] =
		int(value.Exponent) - guarded_bit_count
}

func float_decimal_initialize_magnitude(workspace *Float_Text_Workspace) {
	Float_Text_Workspace_Invariants(
		workspace, "float_decimal_initialize_magnitude.workspace",
	)
	decimal_index := workspace.Control[FLOAT_TEXT_DECIMAL_INDEX]
	decimal := &workspace.Decimals[decimal_index]
	decimal.Control = [FLOAT_DECIMAL_CONTROL_COUNT]int{}
	shift := workspace.Control[FLOAT_TEXT_BINARY_SHIFT_INDEX]
	if shift > 0 {
		float_text_magnitude_shift_left(workspace)
		shift = 0
	}
	magnitude := &workspace.Magnitude
	word_count := magnitude.Control[FLOAT_TEXT_MAGNITUDE_WORD_COUNT_INDEX]
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
		invariant.Always(
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
	decimal.Control[FLOAT_DECIMAL_DIGIT_COUNT_INDEX] = digit_count
	decimal.Control[FLOAT_DECIMAL_EXPONENT_INDEX] = digit_count
	float_decimal_trim(workspace)
	for shift < -FLOAT_DECIMAL_SHIFT_MAXIMUM {
		workspace.Control[FLOAT_TEXT_DECIMAL_SHIFT_INDEX] = FLOAT_DECIMAL_SHIFT_MAXIMUM
		float_decimal_shift_right(workspace)
		shift += FLOAT_DECIMAL_SHIFT_MAXIMUM
	}
	if shift < 0 {
		workspace.Control[FLOAT_TEXT_DECIMAL_SHIFT_INDEX] = -shift
		float_decimal_shift_right(workspace)
	}
}

func float_text_magnitude_shift_left(workspace *Float_Text_Workspace) {
	Float_Text_Workspace_Invariants(
		workspace, "float_text_magnitude_shift_left.workspace",
	)
	magnitude := &workspace.Magnitude
	shift := workspace.Control[FLOAT_TEXT_BINARY_SHIFT_INDEX]
	source_count := magnitude.Control[FLOAT_TEXT_MAGNITUDE_WORD_COUNT_INDEX]
	high_word := magnitude.Words[source_count-WORD_COUNT_INCREMENT]
	high_bit_count := WORD_BIT_COUNT -
		int(bits.Leading_Zeros_64(bits.Word_64(high_word)))
	result_bit_count := (source_count-WORD_COUNT_INCREMENT)*WORD_BIT_COUNT +
		high_bit_count + shift
	result_count := (result_bit_count + WORD_BIT_COUNT - WORD_COUNT_INCREMENT) /
		WORD_BIT_COUNT
	word_shift := shift / WORD_BIT_COUNT
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
	magnitude.Control[FLOAT_TEXT_MAGNITUDE_WORD_COUNT_INDEX] = result_count
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
	destination *Float,
	source Float_Parse_Text_Unvalidated,
	base_unvalidated Base_Unvalidated,
	workspace *Float_Parse_Workspace,
) (base Float_Parse_Base, status Parse_Status) {
	defer func() {
		Float_Parse_Base_Invariants(base, "float_parse.base")
		Parse_Status_Invariants(status, "float_parse.status")
	}()
	Float_Invariants(destination, "float_parse.destination_initial")
	Float_Parse_Text_Unvalidated_Invariants(source, "float_parse.source")
	Base_Unvalidated_Invariants(base_unvalidated, "float_parse.base_unvalidated")
	Float_Parse_Workspace_Invariants(workspace, "float_parse.workspace")
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
	result := &workspace.Values[FLOAT_RAT_RESULT_INDEX]
	*result = Float{Precision: precision, Mode: destination.Mode}
	if float_parse_infinity(workspace) {
		*destination = *result
		return FLOAT_PARSE_BASE_AUTOMATIC, STATUS_OK
	}
	scan_status := float_parse_scan(workspace)
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
	arithmetic := float_parse_finite(workspace)
	if arithmetic != Arithmetic_Status(STATUS_OK) {
		return base, STATUS_VALUE_OVERFLOW
	}
	*destination = *result
	return base, STATUS_OK
}

func float_parse_finite(
	workspace *Float_Parse_Workspace,
) (status Arithmetic_Status) {
	defer func() { Arithmetic_Status_Invariants(status, "float_parse_finite.status") }()
	Float_Parse_Workspace_Invariants(workspace, "float_parse_finite.workspace")
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
	result := &workspace.Values[FLOAT_RAT_RESULT_INDEX]
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
		exponents[RAT_PARSE_EXPONENT_2_INDEX]
	workspace.Control[FLOAT_PARSE_EXPONENT_5_INDEX] =
		exponents[RAT_PARSE_EXPONENT_5_INDEX]
	return float_parse_quotient(workspace)
}

func float_parse_quotient(
	workspace *Float_Parse_Workspace,
) (status Arithmetic_Status) {
	defer func() { Arithmetic_Status_Invariants(status, "float_parse_quotient.status") }()
	Float_Parse_Workspace_Invariants(workspace, "float_parse_quotient.workspace")
	workspace.Integers = Rat_Parse_Fraction_Integers{}
	numerator := &workspace.Integers[RAT_PARSE_FRACTION_NUMERATOR_INDEX]
	denominator := &workspace.Integers[RAT_PARSE_FRACTION_DENOMINATOR_INDEX]
	mantissa_count := int(workspace.Control[FLOAT_PARSE_MANTISSA_COUNT_INDEX])
	for index := WORD_COUNT_MINIMUM; index < mantissa_count; index++ {
		numerator.Words[index] = workspace.Parse.Words[index]
	}
	numerator.Count = Word_Count(mantissa_count)
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
	component := Rat_Parse_Power_Component(RAT_PARSE_FRACTION_NUMERATOR_INDEX)
	if exponent_5 < 0 {
		component = Rat_Parse_Power_Component(RAT_PARSE_FRACTION_DENOMINATOR_INDEX)
		exponent_5 = -exponent_5
	}
	if exponent_5 > 0 {
		arithmetic := rat_parse_multiply_five(
			&workspace.Integers, component, Rat_Parse_Power_Count(exponent_5),
		)
		if arithmetic != Arithmetic_Status(STATUS_OK) {
			return STATUS_VALUE_OVERFLOW
		}
	}
	left := &workspace.Values[FLOAT_RAT_NUMERATOR_INDEX]
	right := &workspace.Values[FLOAT_RAT_DENOMINATOR_INDEX]
	*left = Float{}
	*right = Float{}
	Float_Set_Int(left, numerator)
	Float_Set_Int(right, denominator)
	result := &workspace.Values[FLOAT_RAT_RESULT_INDEX]
	quotient_status := Float_Quotient(
		result, left, right,
		(*Float_Division_Workspace)(&workspace.Division),
	)
	invariant.Always(
		quotient_status == Validation_Status(STATUS_OK),
		"Float parse exact denominator is positive and nonzero.",
	)
	invariant.Always(result.Form == FLOAT_FORM_FINITE,
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

func float_parse_infinity(workspace *Float_Parse_Workspace) (matched Boolean) {
	defer func() { Boolean_Invariants(matched, "float_parse_infinity.matched") }()
	Float_Parse_Workspace_Invariants(workspace, "float_parse_infinity.workspace")
	end_index := int(workspace.Control[FLOAT_PARSE_SOURCE_COUNT_INDEX])
	index := bytes.SLICE_SIZE_MINIMUM
	negative := POLARITY_NONNEGATIVE
	if workspace.Source[index] == '-' {
		negative = POLARITY_NEGATIVE
		index++
	} else if workspace.Source[index] == '+' {
		index++
	}
	remaining_count := end_index - index
	if remaining_count != len("Inf") {
		return false
	}
	matched = Boolean(workspace.Source[index] == 'I')
	if matched {
		matched = Boolean(workspace.Source[index+WORD_COUNT_INCREMENT] == 'n')
		if matched {
			matched = Boolean(workspace.Source[index+BASE_BINARY] == 'f')
		}
	}
	if !matched {
		matched = Boolean(workspace.Source[index] == 'i')
		if matched {
			matched = Boolean(workspace.Source[index+WORD_COUNT_INCREMENT] == 'n')
			if matched {
				matched = Boolean(workspace.Source[index+BASE_BINARY] == 'f')
			}
		}
	}
	if !matched {
		return matched
	}
	result := &workspace.Values[FLOAT_RAT_RESULT_INDEX]
	float_set_nonfinite(
		result, Float_Nonfinite_Form(FLOAT_FORM_INFINITY), negative,
		result.Precision,
	)
	return matched
}

func float_parse_scan(
	workspace *Float_Parse_Workspace,
) (status Parse_Status) {
	defer func() { Parse_Status_Invariants(status, "float_parse_scan.status") }()
	Float_Parse_Workspace_Invariants(workspace, "float_parse_scan.workspace")
	float_parse_scan_prefix(workspace)
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
		validation := float_parse_scan_exponent(workspace)
		if validation != Validation_Status(STATUS_OK) {
			return STATUS_INPUT_INVALID
		}
	}
	return mantissa_status
}

func float_parse_scan_prefix(workspace *Float_Parse_Workspace) {
	Float_Parse_Workspace_Invariants(workspace, "float_parse_scan_prefix.workspace")
	end_index := int(workspace.Control[FLOAT_PARSE_SOURCE_COUNT_INDEX])
	source := workspace.Source[:end_index]
	requested := Float_Parse_Base(workspace.Control[FLOAT_PARSE_BASE_INDEX])
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
	workspace.Control[FLOAT_PARSE_NEGATIVE_INDEX] = 0
	if prefix.Negative {
		workspace.Control[FLOAT_PARSE_NEGATIVE_INDEX] = WORD_COUNT_INCREMENT
	}
	workspace.Control[FLOAT_PARSE_PREFIXED_INDEX] = 0
	if prefix.Prefixed {
		workspace.Control[FLOAT_PARSE_PREFIXED_INDEX] = WORD_COUNT_INCREMENT
	}
	workspace.Control[FLOAT_PARSE_RADIX_INDEX] = int64(prefix.Base)
	workspace.Control[FLOAT_PARSE_START_INDEX] = int64(prefix.Start)
}

func float_parse_scan_mantissa(workspace *Float_Parse_Workspace) (status Parse_Status) {
	defer func() { Parse_Status_Invariants(status, "float_parse_scan_mantissa.status") }()
	Float_Parse_Workspace_Invariants(workspace, "float_parse_scan_mantissa.workspace")
	workspace.Parse = Float_Parse_Integer_Workspace{}
	end_index := int(workspace.Control[FLOAT_PARSE_SOURCE_COUNT_INDEX])
	radix := Base(workspace.Control[FLOAT_PARSE_RADIX_INDEX])
	word_count := Word_Count(WORD_COUNT_MINIMUM)
	digit_seen, radix_seen, separator_seen, magnitude_overflow :=
		false, false, false, false
	previous_digit := workspace.Control[FLOAT_PARSE_PREFIXED_INDEX] != 0
	index := int(workspace.Control[FLOAT_PARSE_START_INDEX])
	separators := workspace.Control[FLOAT_PARSE_SEPARATOR_INDEX] != 0
	fractional_digit_count := int64(bytes.SLICE_SIZE_MINIMUM)
	parse := (*Int_Parse_Workspace)(&workspace.Parse)
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
	workspace *Float_Parse_Workspace,
) (status Validation_Status) {
	defer func() {
		Validation_Status_Invariants(status, "float_parse_scan_exponent.status")
	}()
	Float_Parse_Workspace_Invariants(workspace, "float_parse_scan_exponent.workspace")
	end_index := int(workspace.Control[FLOAT_PARSE_SOURCE_COUNT_INDEX])
	index := int(workspace.Control[FLOAT_PARSE_EXPONENT_START_INDEX])
	negative := false
	if index < end_index {
		if workspace.Source[index] == '-' {
			negative = true
			index++
		} else if workspace.Source[index] == '+' {
			index++
		}
	}
	limit := uint64(bits.INTEGER_64_MAXIMUM)
	if negative {
		limit = INT_64_NEGATIVE_MAGNITUDE_MAXIMUM
	}
	magnitude := uint64(bits.WORD_64_MINIMUM)
	digit_seen, separator_seen := false, false
	separators := workspace.Control[FLOAT_PARSE_SEPARATOR_INDEX] != 0
	for index < end_index {
		character := workspace.Source[index]
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
		if character < '0' {
			return STATUS_INPUT_INVALID
		}
		if character > '9' {
			return STATUS_INPUT_INVALID
		}
		digit := uint64(character - '0')
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
	workspace.Control[FLOAT_PARSE_EXPONENT_INDEX] = exponent
	return STATUS_OK
}

// Rat_Set_Float_32_Bits stores exact finite binary32 and rejects nonfinite input.
func Rat_Set_Float_32_Bits(
	destination *Rat, encoding Float_32_Bits,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "rat_set_float_32_bits.status") }()
	Rat_Invariants(destination, "rat_set_float_32_bits.destination")
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
	numerator := &destination.Integers[RAT_NUMERATOR_INDEX]
	denominator := &destination.Integers[RAT_DENOMINATOR_INDEX]
	Int_Set_Uint_64(numerator, Word_64(mantissa))
	*denominator = Int{}
	if shift > 0 {
		Int_Set_Uint_64(denominator, Word_64(bits.CARRY_MAXIMUM))
		shift_status := Int_Shift_Left(denominator, denominator, Shift_Count(shift))
		invariant.Always(
			shift_status == Arithmetic_Status(STATUS_OK),
			"Binary32 denominator fits rational component storage.",
		)
	} else {
		shift_status := Int_Shift_Left(numerator, numerator, Shift_Count(-shift))
		invariant.Always(
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
	value *Rat, workspace *Rat_Float_32_Workspace,
) (encoding Float_32_Value_Bits, exact Boolean) {
	defer func() {
		Float_32_Value_Bits_Invariants(encoding, "rat_float_32_bits.encoding")
		Boolean_Invariants(exact, "rat_float_32_bits.exact")
	}()
	Rat_Invariants(value, "rat_float_32_bits.value")
	Rat_Float_32_Workspace_Invariants(workspace, "rat_float_32_bits.workspace")
	numerator_source := &value.Integers[RAT_NUMERATOR_INDEX]
	if numerator_source.Count == WORD_COUNT_MINIMUM {
		return Float_32_Value_Bits(FLOAT_32_VALUE_BITS_MINIMUM), Boolean(true)
	}
	denominator_source := &value.Integers[RAT_DENOMINATOR_INDEX]
	exponent := int(Int_Bit_Count(numerator_source)) - WORD_COUNT_INCREMENT
	if denominator_source.Count != WORD_COUNT_MINIMUM {
		exponent += WORD_COUNT_INCREMENT - int(Int_Bit_Count(denominator_source))
	}
	rat_float_32_divide(value, workspace)
	quotient := &workspace.Integers[RAT_FLOAT_32_QUOTIENT_INDEX]
	remainder := &workspace.Integers[RAT_FLOAT_32_REMAINDER_INDEX]
	quotient_word, conversion_status := Int_Uint_64(quotient)
	quotient_fits := conversion_status == Conversion_Status(STATUS_OK) &&
		uint64(quotient_word) <= uint64(bits.WORD_32_MAXIMUM)
	invariant.Always(quotient_fits, "Binary32 rounding quotient fits one word.")
	mantissa := uint32(quotient_word)
	have_remainder := remainder.Count != WORD_COUNT_MINIMUM
	if mantissa>>FLOAT_32_ROUNDING_MANTISSA_BIT_COUNT == uint32(bits.CARRY_MAXIMUM) {
		if mantissa&uint32(bits.CARRY_MAXIMUM) != bits.WORD_32_MINIMUM {
			have_remainder = true
		}
		mantissa >>= 1
		exponent++
	}
	invariant.Always(
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

func rat_float_32_divide(value *Rat, workspace *Rat_Float_32_Workspace) {
	Rat_Invariants(value, "rat_float_32_divide.value")
	Rat_Float_32_Workspace_Invariants(workspace, "rat_float_32_divide.workspace")
	workspace.Integers = Rat_Float_32_Integers{}
	numerator := &workspace.Integers[RAT_FLOAT_32_NUMERATOR_INDEX]
	denominator := &workspace.Integers[RAT_FLOAT_32_DENOMINATOR_INDEX]
	quotient := &workspace.Integers[RAT_FLOAT_32_QUOTIENT_INDEX]
	remainder := &workspace.Integers[RAT_FLOAT_32_REMAINDER_INDEX]
	Int_Absolute(numerator, &value.Integers[RAT_NUMERATOR_INDEX])
	Rat_Denominator_Into(denominator, value)
	exponent := int(Int_Bit_Count(numerator)) - int(Int_Bit_Count(denominator))
	shift := FLOAT_32_ROUNDING_MANTISSA_BIT_COUNT - exponent
	shift_status := Arithmetic_Status(STATUS_OK)
	if shift > WORD_COUNT_MINIMUM {
		shift_status = Int_Shift_Left(numerator, numerator, Shift_Count(shift))
	} else if shift < WORD_COUNT_MINIMUM {
		shift_status = Int_Shift_Left(denominator, denominator, Shift_Count(-shift))
	}
	invariant.Always(
		shift_status == Arithmetic_Status(STATUS_OK),
		"Rational component bound leaves full Int room for binary32 scaling.",
	)
	division := (*Int_Division_Workspace)(&workspace.Division)
	division_status := Int_Quotient_Remainder(
		quotient, remainder, numerator, denominator, division,
	)
	invariant.Always(
		division_status == Division_Status(STATUS_OK),
		"Normalized rational denominator remains nonzero during binary32 conversion.",
	)
}

// Float_Set_Float_32_Bits avoids native float conversion and its hidden rounding.
func Float_Set_Float_32_Bits(
	destination *Float, encoding Float_32_Bits,
) (status Validation_Status) {
	defer func() {
		Validation_Status_Invariants(status, "float_set_float_32_bits.status")
	}()
	Float_Invariants(destination, "float_set_float_32_bits.destination_initial")
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
		destination, Float_32_Finite_Exponent_Field(exponent_field),
		Float_32_Mantissa_Field(mantissa_field), negative,
		Float_Active_Precision(precision),
	)
	return STATUS_OK
}

// Float_Float_32_Bits prevents host floating-point policy from changing IEEE rounding.
func Float_Float_32_Bits(value *Float) (
	encoding Float_32_Value_Bits, accuracy Accuracy,
) {
	defer func() {
		Float_32_Value_Bits_Invariants(encoding, "float_float_32_bits.encoding")
		Accuracy_Invariants(accuracy, "float_float_32_bits.accuracy")
	}()
	Float_Invariants(value, "float_float_32_bits.value")
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
	return float_finite_float_32_bits((*Float_Finite)(value), Float_32_Sign(sign))
}

func float_set_float_32_finite(
	destination *Float,
	exponent_field Float_32_Finite_Exponent_Field,
	mantissa_field Float_32_Mantissa_Field,
	negative Polarity,
	precision Float_Active_Precision,
) {
	Float_Invariants(destination, "float_set_float_32_finite.destination_initial")
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
	var magnitude Int
	Int_Set_Uint_64(&magnitude, Word_64(mantissa))
	float_magnitude := Float_Mantissa(magnitude)
	float_set_magnitude(
		destination, &float_magnitude, Float_Exponent(exponent), negative,
		precision, destination.Mode,
	)
}

func float_finite_float_32_bits(
	value *Float_Finite, sign Float_32_Sign,
) (encoding Float_32_Value_Bits, accuracy Accuracy) {
	defer func() {
		Float_32_Value_Bits_Invariants(
			encoding, "float_finite_float_32_bits.encoding",
		)
		Accuracy_Invariants(accuracy, "float_finite_float_32_bits.accuracy")
	}()
	Float_Finite_Invariants(value, "float_finite_float_32_bits.value")
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
				minimum_precision := Float_Minimum_Precision((*Float)(value))
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
	rounded := Float(*value)
	rounded.Mode = Rounding_Mode(ROUND_TO_NEAREST_EVEN)
	float_round(&rounded, Float_Precision(precision))
	if rounded.Form == FLOAT_FORM_INFINITY {
		return Float_32_Value_Bits(
			uint32(sign) | FLOAT_32_POSITIVE_INFINITY_BITS,
		), rounded.Accuracy
	}
	invariant.Always(
		rounded.Mantissa.Count == WORD_COUNT_INCREMENT,
		"Rounded binary32 mantissa fits one machine word.",
	)
	invariant.Always(
		rounded.Exponent > FLOAT_32_SUBNORMAL_EXPONENT-FLOAT_32_MANTISSA_BIT_COUNT,
		"Rounded binary32 exponent stays above tie underflow handling.",
	)
	invariant.Always(
		rounded.Exponent <= FLOAT_32_VALUE_BIT_COUNT_MAXIMUM,
		"Rounded binary32 exponent stays below overflow handling.",
	)
	bit_count := int(Int_Bit_Count((*Int)(&rounded.Mantissa)))
	mantissa := uint32(rounded.Mantissa.Words[WORD_COUNT_MINIMUM])
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
	destination *Float,
	dividend *Float,
	divisor *Float,
	workspace *Float_Division_Workspace,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "float_quotient.status") }()
	Float_Invariants(destination, "float_quotient.destination_initial")
	Float_Invariants(dividend, "float_quotient.dividend")
	Float_Invariants(divisor, "float_quotient.divisor")
	Float_Division_Workspace_Invariants(workspace, "float_quotient.workspace")
	precision := destination.Precision
	if precision == FLOAT_PRECISION_MINIMUM {
		precision = dividend.Precision
		if divisor.Precision > precision {
			precision = divisor.Precision
		}
	}
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
		destination, (*Float_Finite)(dividend), (*Float_Finite)(divisor), workspace,
		Float_Active_Precision(precision), mode, negative,
	)
	return STATUS_OK
}

// Float_Int_Into truncates one finite value toward zero into bounded caller storage.
func Float_Int_Into(
	destination *Int, value *Float,
) (accuracy Accuracy, status Conversion_Status) {
	defer func() {
		Accuracy_Invariants(accuracy, "float_int_into.accuracy")
		Conversion_Status_Invariants(status, "float_int_into.status")
	}()
	Int_Invariants(destination, "float_int_into.destination_initial")
	Float_Invariants(value, "float_int_into.value")
	if value.Form == FLOAT_FORM_INFINITY {
		return ACCURACY_EXACT, STATUS_VALUE_OVERFLOW
	}
	if value.Form == FLOAT_FORM_ZERO {
		*destination = Int{}
		return ACCURACY_EXACT, STATUS_OK
	}
	if value.Exponent <= FLOAT_EXPONENT_ZERO {
		*destination = Int{}
		return Accuracy(float_accuracy(false, value.Negative)), STATUS_OK
	}
	if value.Exponent > BIT_COUNT_MAXIMUM {
		return ACCURACY_EXACT, STATUS_VALUE_OVERFLOW
	}
	bit_count := int(Int_Bit_Count((*Int)(&value.Mantissa)))
	shift := int(value.Exponent) - bit_count
	result := Int(value.Mantissa)
	if shift < 0 {
		discard_count := -shift
		inexact := float_low_bits_nonzero(
			(*Float_Active_Mantissa)(&value.Mantissa),
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
	value *Float,
) (result Int_64, accuracy Accuracy, status Conversion_Status) {
	defer func() {
		Int_64_Invariants(result, "float_int_64.result")
		Accuracy_Invariants(accuracy, "float_int_64.accuracy")
		Conversion_Status_Invariants(status, "float_int_64.status")
	}()
	Float_Invariants(value, "float_int_64.value")
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
	value *Float,
) (result Word_64, accuracy Accuracy, status Conversion_Status) {
	defer func() {
		Word_64_Invariants(result, "float_uint_64.result")
		Accuracy_Invariants(accuracy, "float_uint_64.accuracy")
		Conversion_Status_Invariants(status, "float_uint_64.status")
	}()
	Float_Invariants(value, "float_uint_64.value")
	var integer Int
	accuracy, status = Float_Int_Into(&integer, value)
	if status != STATUS_OK {
		return 0, accuracy, status
	}
	result, status = Int_Uint_64(&integer)
	return result, accuracy, status
}

// Float_Set_Rat rounds one exact rational through caller-owned quotient storage.
func Float_Set_Rat(destination *Float, source *Rat, workspace *Float_Rat_Workspace) {
	Float_Invariants(destination, "float_set_rat.destination_initial")
	Rat_Invariants(source, "float_set_rat.source")
	Float_Rat_Workspace_Invariants(workspace, "float_set_rat.workspace")
	precision := destination.Precision
	mode := destination.Mode
	workspace.Values = Float_Rat_Values{}
	numerator := &workspace.Values[FLOAT_RAT_NUMERATOR_INDEX]
	denominator := &workspace.Values[FLOAT_RAT_DENOMINATOR_INDEX]
	result := &workspace.Values[FLOAT_RAT_RESULT_INDEX]
	Float_Set_Int(numerator, &source.Integers[RAT_NUMERATOR_INDEX])
	stored_denominator := &source.Integers[RAT_DENOMINATOR_INDEX]
	if stored_denominator.Count == WORD_COUNT_MINIMUM {
		Float_Set_Uint_64(denominator, Word_64(WORD_COUNT_INCREMENT))
	} else {
		Float_Set_Int(denominator, stored_denominator)
	}
	result.Precision = precision
	result.Mode = mode
	Float_Quotient(
		result, numerator, denominator,
		(*Float_Division_Workspace)(&workspace.Division),
	)
	*destination = *result
}

// Float_Rat_Into writes one exact binary rational or leaves the destination unchanged.
func Float_Rat_Into(
	destination *Rat, value *Float, workspace *Float_Rat_Workspace,
) (status Float_Rat_Status) {
	defer func() { Float_Rat_Status_Invariants(status, "float_rat_into.status") }()
	Rat_Invariants(destination, "float_rat_into.destination_initial")
	Float_Invariants(value, "float_rat_into.value")
	Float_Rat_Workspace_Invariants(workspace, "float_rat_into.workspace")
	if value.Form == FLOAT_FORM_INFINITY {
		return STATUS_INPUT_INVALID
	}
	if value.Form == FLOAT_FORM_ZERO {
		*destination = Rat{}
		return STATUS_OK
	}
	workspace.Integers = Rat_Integers{}
	numerator := &workspace.Integers[RAT_NUMERATOR_INDEX]
	denominator := &workspace.Integers[RAT_DENOMINATOR_INDEX]
	*numerator = Int(value.Mantissa)
	bit_count := int(Int_Bit_Count(numerator))
	zero_count := int(Int_Trailing_Zero_Bit_Count(numerator))
	reduced_bit_count := bit_count - zero_count
	power := int(value.Exponent) - bit_count + zero_count
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
	numerator.Negative = value.Negative
	result := Rat{}
	result.Integers[RAT_NUMERATOR_INDEX] = *numerator
	result.Integers[RAT_DENOMINATOR_INDEX] = *denominator
	*destination = result
	return STATUS_OK
}

// Float_Square_Root rounds one nonnegative root through caller-owned restoring state.
func Float_Square_Root(
	destination *Float, source *Float, workspace *Float_Square_Root_Workspace,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "float_square_root.status") }()
	Float_Invariants(destination, "float_square_root.destination_initial")
	Float_Invariants(source, "float_square_root.source")
	Float_Square_Root_Workspace_Invariants(workspace, "float_square_root.workspace")
	if source.Negative == POLARITY_NEGATIVE {
		if source.Form != FLOAT_FORM_ZERO {
			return STATUS_INPUT_INVALID
		}
	}
	precision := destination.Precision
	if precision == FLOAT_PRECISION_MINIMUM {
		precision = source.Precision
	}
	mode := destination.Mode
	if source.Form == FLOAT_FORM_ZERO {
		float_set_operation_operand(destination, source, precision, mode)
		return STATUS_OK
	}
	if source.Form == FLOAT_FORM_INFINITY {
		float_set_operation_infinity(
			destination, POLARITY_NONNEGATIVE, precision, mode,
		)
		return STATUS_OK
	}
	float_square_root_finite(
		destination, (*Float_Nonnegative_Finite)(source), workspace,
		Float_Active_Precision(precision), mode,
	)
	return STATUS_OK
}

func float_square_root_finite(
	destination *Float,
	source *Float_Nonnegative_Finite,
	workspace *Float_Square_Root_Workspace,
	precision Float_Active_Precision,
	mode Rounding_Mode,
) {
	Float_Invariants(destination, "float_square_root_finite.destination_initial")
	Float_Nonnegative_Finite_Invariants(source, "float_square_root_finite.source")
	Float_Square_Root_Workspace_Invariants(
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
	var mantissa Float_Mantissa
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
	source *Float_Nonnegative_Finite,
	workspace *Float_Square_Root_Workspace,
	precision Float_Active_Precision,
) (rounding_bit Bit_Value, sticky Bit_Value) {
	defer func() {
		Bit_Value_Invariants(
			rounding_bit, "float_square_root_extract.rounding_bit",
		)
		Bit_Value_Invariants(sticky, "float_square_root_extract.sticky")
	}()
	Float_Nonnegative_Finite_Invariants(source, "float_square_root_extract.source")
	Float_Square_Root_Workspace_Invariants(
		workspace, "float_square_root_extract.workspace",
	)
	Float_Active_Precision_Invariants(precision, "float_square_root_extract.precision")
	source_index := float_square_root_digits(source, workspace, precision)
	rounding_bit = Bit_Value(workspace.Root[WORD_COUNT_MINIMUM] & Word(BIT_SET))
	sticky = float_square_root_sticky(source, workspace, source_index)
	float_square_root_remove_guard(workspace, precision)
	return rounding_bit, sticky
}

func float_square_root_digits(
	source *Float_Nonnegative_Finite,
	workspace *Float_Square_Root_Workspace,
	precision Float_Active_Precision,
) (source_index Float_Square_Root_Source_Bit_Index) {
	defer func() {
		Float_Square_Root_Source_Bit_Index_Invariants(
			source_index, "float_square_root_digits.source_index",
		)
	}()
	Float_Nonnegative_Finite_Invariants(source, "float_square_root_digits.source")
	Float_Square_Root_Workspace_Invariants(
		workspace, "float_square_root_digits.workspace",
	)
	Float_Active_Precision_Invariants(precision, "float_square_root_digits.precision")
	float_square_root_clear(workspace)
	cursor := int(Int_Bit_Count((*Int)(&source.Mantissa))) - WORD_COUNT_INCREMENT
	prefix_clear := int(source.Exponent)%BASE_BINARY != BIT_COUNT_MINIMUM
	digit_count := int(precision) + FLOAT_SQUARE_ROOT_GUARD_BIT_COUNT
	for digit_index := WORD_COUNT_MINIMUM; digit_index < digit_count; digit_index++ {
		pair := Word(BIT_CLEAR)
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
				pair |= source.Mantissa.Words[word_index] >> bit_index &
					Word(BIT_SET)
				cursor--
				if cursor == BIT_INDEX_UNVALIDATED_MINIMUM {
					source_exhausted = true
				}
			}
		}
		active_count := (digit_index + FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT +
			WORD_BIT_INDEX_MAXIMUM) / WORD_BIT_COUNT
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
				workspace.Root[index]<<FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT |
					candidate_carry
			candidate_carry = workspace.Root[index] >>
				(WORD_BIT_COUNT - FLOAT_SQUARE_ROOT_PAIR_BIT_COUNT)
		}
		workspace.Remainder[WORD_COUNT_MINIMUM] |= pair
		workspace.Candidate[WORD_COUNT_MINIMUM] |= Word(BIT_SET)
		float_square_root_restore(workspace)
		if source_exhausted {
			last_digit_index := digit_count - WORD_COUNT_INCREMENT
			suffix_count := last_digit_index - digit_index
			if suffix_count > BIT_COUNT_MINIMUM {
				if float_square_root_complete(
					workspace, Float_Active_Precision(suffix_count),
				) {
					break
				}
			}
		}
	}
	return Float_Square_Root_Source_Bit_Index(cursor)
}

func float_square_root_restore(workspace *Float_Square_Root_Workspace) {
	Float_Square_Root_Workspace_Invariants(
		workspace, "float_square_root_restore.workspace",
	)
	active_last_index := WORD_COUNT_MINIMUM
	last_index := len(workspace.Root) - WORD_COUNT_INCREMENT
	for index := last_index; index > WORD_COUNT_MINIMUM; index-- {
		if workspace.Root[index] != 0 {
			active_last_index = index
			break
		}
		if workspace.Remainder[index] != 0 {
			active_last_index = index
			break
		}
		if workspace.Candidate[index] != 0 {
			active_last_index = index
			break
		}
	}
	subtract := true
	for index := active_last_index; index >= WORD_COUNT_MINIMUM; index-- {
		if workspace.Remainder[index] != workspace.Candidate[index] {
			subtract = workspace.Remainder[index] > workspace.Candidate[index]
			break
		}
	}
	if subtract {
		borrow := bits.Borrow_In(bits.CARRY_MINIMUM)
		for index := WORD_COUNT_MINIMUM; index <= active_last_index; index++ {
			difference, next := bits.Subtract_Word(
				bits.Word(workspace.Remainder[index]),
				bits.Subtrahend_Word(workspace.Candidate[index]), borrow,
			)
			workspace.Remainder[index] = Word(difference)
			borrow = bits.Borrow_In(next)
		}
	}
	carry := Word(BIT_CLEAR)
	for index := WORD_COUNT_MINIMUM; index <= active_last_index; index++ {
		next := workspace.Root[index] >> WORD_BIT_INDEX_MAXIMUM
		workspace.Root[index] =
			workspace.Root[index]<<WORD_COUNT_INCREMENT | carry
		carry = next
	}
	if subtract {
		workspace.Root[WORD_COUNT_MINIMUM] |= Word(BIT_SET)
	}
}

func float_square_root_complete(
	workspace *Float_Square_Root_Workspace, count Float_Active_Precision,
) (complete Boolean) {
	defer func() {
		Boolean_Invariants(complete, "float_square_root_complete.complete")
	}()
	Float_Square_Root_Workspace_Invariants(
		workspace, "float_square_root_complete.workspace",
	)
	Float_Active_Precision_Invariants(count, "float_square_root_complete.count")
	for index := WORD_COUNT_MINIMUM; index < len(workspace.Remainder); index++ {
		if workspace.Remainder[index] != 0 {
			return false
		}
	}
	word_shift := int(count) / WORD_BIT_COUNT
	bit_shift := uint(int(count) % WORD_BIT_COUNT)
	minimum := WORD_COUNT_MINIMUM
	last_index := len(workspace.Root) - WORD_COUNT_INCREMENT
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

func float_square_root_clear(workspace *Float_Square_Root_Workspace) {
	Float_Square_Root_Workspace_Invariants(
		workspace, "float_square_root_clear.workspace",
	)
	workspace.Root = Float_Square_Root_Root{}
	workspace.Remainder = Float_Square_Root_Remainder{}
	workspace.Candidate = Float_Square_Root_Candidate{}
}

func float_square_root_sticky(
	source *Float_Nonnegative_Finite,
	workspace *Float_Square_Root_Workspace,
	source_index Float_Square_Root_Source_Bit_Index,
) (sticky Bit_Value) {
	defer func() { Bit_Value_Invariants(sticky, "float_square_root_sticky.sticky") }()
	Float_Nonnegative_Finite_Invariants(source, "float_square_root_sticky.source")
	Float_Square_Root_Workspace_Invariants(
		workspace, "float_square_root_sticky.workspace",
	)
	Float_Square_Root_Source_Bit_Index_Invariants(
		source_index, "float_square_root_sticky.source_index",
	)
	if source_index >= BIT_COUNT_MINIMUM {
		if float_low_bits_nonzero(
			(*Float_Active_Mantissa)(&source.Mantissa),
			Float_Discarded_Bit_Count(
				int(source_index)+WORD_COUNT_INCREMENT,
			),
		) {
			return BIT_SET
		}
	}
	for _, word := range workspace.Remainder {
		if word != 0 {
			return BIT_SET
		}
	}
	return BIT_CLEAR
}

func float_square_root_remove_guard(
	workspace *Float_Square_Root_Workspace, precision Float_Active_Precision,
) {
	Float_Square_Root_Workspace_Invariants(
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
	destination *Float,
	mantissa *Float_Active_Mantissa,
	exponent Float_Square_Root_Exponent,
	precision Float_Active_Precision,
	mode Rounding_Mode,
	accuracy Accuracy,
	increment Boolean,
) {
	Float_Invariants(destination, "float_square_root_commit.destination_initial")
	Float_Active_Mantissa_Invariants(*mantissa, "float_square_root_commit.mantissa")
	Float_Square_Root_Exponent_Invariants(exponent, "float_square_root_commit.exponent")
	Float_Active_Precision_Invariants(precision, "float_square_root_commit.precision")
	Rounding_Mode_Invariants(mode, "float_square_root_commit.mode")
	Accuracy_Invariants(accuracy, "float_square_root_commit.accuracy")
	Boolean_Invariants(increment, "float_square_root_commit.increment")
	result_exponent := Float_Exponent(exponent)
	if increment {
		float_increment_mantissa(
			mantissa, precision, &result_exponent,
		)
	}
	*destination = Float{
		Precision: Float_Precision(precision), Mode: mode, Accuracy: accuracy,
		Form: FLOAT_FORM_FINITE, Mantissa: Float_Mantissa(*mantissa),
		Exponent: result_exponent,
	}
}

func float_divide_finite(
	destination *Float,
	dividend *Float_Finite,
	divisor *Float_Finite,
	workspace *Float_Division_Workspace,
	precision Float_Active_Precision,
	mode Rounding_Mode,
	negative Polarity,
) {
	Float_Invariants(destination, "float_divide_finite.destination_initial")
	Float_Finite_Invariants(dividend, "float_divide_finite.dividend")
	Float_Finite_Invariants(divisor, "float_divide_finite.divisor")
	Float_Division_Workspace_Invariants(workspace, "float_divide_finite.workspace")
	Float_Active_Precision_Invariants(precision, "float_divide_finite.precision")
	Rounding_Mode_Invariants(mode, "float_divide_finite.mode")
	Polarity_Invariants(negative, "float_divide_finite.negative")
	dividend_bits := int(Int_Bit_Count((*Int)(&dividend.Mantissa)))
	divisor_bits := int(Int_Bit_Count((*Int)(&divisor.Mantissa)))
	aligned_bits := dividend_bits
	if divisor_bits > aligned_bits {
		aligned_bits = divisor_bits
	}
	count := Float_Division_Active_Word_Count(
		(aligned_bits+WORD_BIT_COUNT-WORD_COUNT_INCREMENT)/WORD_BIT_COUNT +
			WORD_COUNT_INCREMENT,
	)
	for index := range workspace.Remainder {
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
		mantissa.Count = WORD_COUNT_INCREMENT
		mantissa.Words[WORD_COUNT_MINIMUM] = Word(bits.CARRY_MAXIMUM)
		*destination = Float{
			Precision: Float_Precision(precision), Mode: mode, Accuracy: ACCURACY_EXACT,
			Form: FLOAT_FORM_FINITE, Negative: negative, Mantissa: mantissa,
			Exponent: Float_Exponent(exponent),
		}
		return
	}
	float_set_quotient_mantissa(
		destination, workspace, count, Float_Unequal_Order(order),
		Float_Exponent(exponent), negative,
		precision, mode,
	)
}

func float_division_load(
	workspace *Float_Division_Workspace,
	dividend *Float_Finite,
	divisor *Float_Finite,
	aligned_bits Float_Division_Aligned_Bit_Count,
) {
	Float_Division_Workspace_Invariants(workspace, "float_division_load.workspace")
	Float_Finite_Invariants(dividend, "float_division_load.dividend")
	Float_Finite_Invariants(divisor, "float_division_load.divisor")
	Float_Division_Aligned_Bit_Count_Invariants(
		aligned_bits, "float_division_load.aligned_bits",
	)
	dividend_bits := int(Int_Bit_Count((*Int)(&dividend.Mantissa)))
	divisor_bits := int(Int_Bit_Count((*Int)(&divisor.Mantissa)))
	dividend_shift := int(aligned_bits) - dividend_bits
	divisor_shift := int(aligned_bits) - divisor_bits
	for index := WORD_COUNT_MINIMUM; index < int(dividend.Mantissa.Count); index++ {
		word_shift := dividend_shift / WORD_BIT_COUNT
		bit_shift := uint(dividend_shift % WORD_BIT_COUNT)
		destination_index := word_shift + index
		word := dividend.Mantissa.Words[index]
		workspace.Remainder[destination_index] |= word << bit_shift
		if bit_shift != 0 {
			workspace.Remainder[destination_index+WORD_COUNT_INCREMENT] |=
				word >> (WORD_BIT_COUNT - bit_shift)
		}
	}
	for index := WORD_COUNT_MINIMUM; index < int(divisor.Mantissa.Count); index++ {
		word_shift := divisor_shift / WORD_BIT_COUNT
		bit_shift := uint(divisor_shift % WORD_BIT_COUNT)
		destination_index := word_shift + index
		word := divisor.Mantissa.Words[index]
		workspace.Divisor[destination_index] |= word << bit_shift
		if bit_shift != 0 {
			workspace.Divisor[destination_index+WORD_COUNT_INCREMENT] |=
				word >> (WORD_BIT_COUNT - bit_shift)
		}
	}
}

func float_division_compare(
	workspace *Float_Division_Workspace, count Float_Division_Active_Word_Count,
) (order Order) {
	defer func() { Order_Invariants(order, "float_division_compare.order") }()
	Float_Division_Workspace_Invariants(workspace, "float_division_compare.workspace")
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
	workspace *Float_Division_Workspace, count Float_Division_Active_Word_Count,
) {
	Float_Division_Workspace_Invariants(workspace, "float_division_subtract.workspace")
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
	invariant.Always(borrow == bits.Borrow_In(bits.CARRY_MINIMUM),
		"A quotient digit subtracts only after magnitude comparison.")
}

func float_division_digit(
	workspace *Float_Division_Workspace, count Float_Division_Active_Word_Count,
) (digit Bit_Value) {
	defer func() { Bit_Value_Invariants(digit, "float_division_digit.digit") }()
	Float_Division_Workspace_Invariants(workspace, "float_division_digit.workspace")
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
	destination *Float,
	form Float_Nonfinite_Form,
	negative Polarity,
	precision Float_Active_Precision,
	mode Rounding_Mode,
	increment Boolean,
) {
	Float_Invariants(destination, "float_set_division_bound.destination_initial")
	Float_Nonfinite_Form_Invariants(form, "float_set_division_bound.form")
	Polarity_Invariants(negative, "float_set_division_bound.negative")
	Float_Active_Precision_Invariants(precision, "float_set_division_bound.precision")
	Rounding_Mode_Invariants(mode, "float_set_division_bound.mode")
	Boolean_Invariants(increment, "float_set_division_bound.increment")
	result := Float{Precision: Float_Precision(precision), Mode: mode}
	float_set_nonfinite(
		&result, form, negative, Float_Precision(precision),
	)
	result.Accuracy = Accuracy(float_accuracy(increment, negative))
	*destination = result
}

func float_set_quotient_mantissa(
	destination *Float,
	workspace *Float_Division_Workspace,
	count Float_Division_Active_Word_Count,
	order Float_Unequal_Order,
	exponent Float_Exponent,
	negative Polarity,
	precision Float_Active_Precision,
	mode Rounding_Mode,
) {
	Float_Invariants(destination, "float_set_quotient_mantissa.destination_initial")
	Float_Division_Workspace_Invariants(workspace, "float_set_quotient_mantissa.workspace")
	Float_Division_Active_Word_Count_Invariants(count, "float_set_quotient_mantissa.count")
	Float_Unequal_Order_Invariants(order, "float_set_quotient_mantissa.order")
	Float_Exponent_Invariants(exponent, "float_set_quotient_mantissa.exponent")
	Polarity_Invariants(negative, "float_set_quotient_mantissa.negative")
	Float_Active_Precision_Invariants(precision, "float_set_quotient_mantissa.precision")
	Rounding_Mode_Invariants(mode, "float_set_quotient_mantissa.mode")
	mantissa_count := (int(precision) + WORD_BIT_COUNT - WORD_COUNT_INCREMENT) /
		WORD_BIT_COUNT
	var mantissa Float_Mantissa
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
	accuracy := ACCURACY_EXACT
	if rounding_bit != BIT_CLEAR {
		accuracy = Accuracy(float_accuracy(increment, negative))
	} else if sticky_bit != BIT_CLEAR {
		accuracy = Accuracy(float_accuracy(increment, negative))
	}
	if increment {
		float_increment_mantissa(
			(*Float_Active_Mantissa)(&mantissa), precision, &exponent,
		)
	}
	if exponent > FLOAT_EXPONENT_MAXIMUM {
		float_set_division_bound(
			destination, Float_Nonfinite_Form(FLOAT_FORM_INFINITY), negative,
			precision, mode, true,
		)
		return
	}
	*destination = Float{
		Precision: Float_Precision(precision), Mode: mode, Accuracy: accuracy,
		Form: FLOAT_FORM_FINITE, Negative: negative, Mantissa: mantissa,
		Exponent: exponent,
	}
}

func float_division_remainder_nonzero(
	workspace *Float_Division_Workspace, count Float_Division_Active_Word_Count,
) (nonzero Boolean) {
	defer func() {
		Boolean_Invariants(nonzero, "float_division_remainder_nonzero.nonzero")
	}()
	Float_Division_Workspace_Invariants(
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
