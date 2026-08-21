// Package big keeps large arithmetic bounded without surrendering stdlib number semantics to
// hidden slice growth. Values own fixed storage, while operations write caller destinations.
package big

import (
	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
)

// WORD_BIT_COUNT follows target word width instead of restating machine architecture here.
const WORD_BIT_COUNT = bits.WORD_SIZE

// WORD_BYTE_COUNT derives storage accounting from bit and byte widths.
const WORD_BYTE_COUNT = WORD_BIT_COUNT / bits.BIT_COUNT_8_MAXIMUM

// WORD_COUNT_MINIMUM admits zero magnitude without a sentinel word.
const WORD_COUNT_MINIMUM = 0

// WORD_COUNT_MAXIMUM keeps one magnitude inside repository bounded byte storage.
const WORD_COUNT_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM / WORD_BYTE_COUNT

// WORDS_UNVALIDATED_SIZE_MAXIMUM admits one hostile word beyond Int storage.
const WORDS_UNVALIDATED_SIZE_MAXIMUM = WORD_COUNT_MAXIMUM + 1

// RANDOM_WORD_SIZE_MAXIMUM limits one entropy batch to one complete Int storage bound.
const RANDOM_WORD_SIZE_MAXIMUM = WORD_COUNT_MAXIMUM

// RANDOM_WORD_SIZE_UNVALIDATED_MAXIMUM admits one oversized hostile entropy word.
const RANDOM_WORD_SIZE_UNVALIDATED_MAXIMUM = RANDOM_WORD_SIZE_MAXIMUM + 1

// WORD_COUNT_INCREMENT is one extra sign-extension word.
const WORD_COUNT_INCREMENT = 1

// BITWISE_WORD_COUNT_MINIMUM keeps zero's sign explicit during bitwise work.
const BITWISE_WORD_COUNT_MINIMUM = WORD_COUNT_MINIMUM + WORD_COUNT_INCREMENT

// BITWISE_WORD_COUNT_MAXIMUM holds one sign-extension word beyond any Int magnitude.
const BITWISE_WORD_COUNT_MAXIMUM = WORD_COUNT_MAXIMUM + WORD_COUNT_INCREMENT

// INT_RANDOM_VALUE_INDEX selects candidate storage.
const INT_RANDOM_VALUE_INDEX = 0

// INT_RANDOM_INTEGER_COUNT is complete bounded random state.
const INT_RANDOM_INTEGER_COUNT = INT_RANDOM_VALUE_INDEX + 1

// PRIMALITY_REPETITION_COUNT_MINIMUM retains stdlib Baillie-PSW-only mode.
const PRIMALITY_REPETITION_COUNT_MINIMUM = 0

// PRIMALITY_REPETITION_COUNT_MAXIMUM cannot consume more bases than bounded entropy words.
const PRIMALITY_REPETITION_COUNT_MAXIMUM = RANDOM_WORD_SIZE_MAXIMUM

// PRIMALITY_REPETITION_COUNT_UNVALIDATED_MINIMUM admits one hostile negative repetition count.
const PRIMALITY_REPETITION_COUNT_UNVALIDATED_MINIMUM = PRIMALITY_REPETITION_COUNT_MINIMUM -
	WORD_COUNT_INCREMENT

// PRIMALITY_REPETITION_COUNT_UNVALIDATED_MAXIMUM admits one hostile excessive count.
const PRIMALITY_REPETITION_COUNT_UNVALIDATED_MAXIMUM = PRIMALITY_REPETITION_COUNT_MAXIMUM +
	WORD_COUNT_INCREMENT

// PRIMALITY_PARAMETER_MINIMUM matches first Baillie-OEIS method-C parameter.
const PRIMALITY_PARAMETER_MINIMUM = BASE_BINARY + WORD_COUNT_INCREMENT

// PRIMALITY_PARAMETER_MAXIMUM retains stdlib defensive search ceiling.
const PRIMALITY_PARAMETER_MAXIMUM = BASE_DECIMAL * BASE_DECIMAL * BASE_DECIMAL * BASE_DECIMAL

// PRIMALITY_PARAMETER_COUNT_MINIMUM permits one explicit bounded search attempt.
const PRIMALITY_PARAMETER_COUNT_MINIMUM = WORD_COUNT_INCREMENT

// PRIMALITY_PARAMETER_COUNT_MAXIMUM covers complete stdlib parameter interval.
const PRIMALITY_PARAMETER_COUNT_MAXIMUM = PRIMALITY_PARAMETER_MAXIMUM -
	PRIMALITY_PARAMETER_MINIMUM + WORD_COUNT_INCREMENT

// PRIMALITY_PARAMETER_COUNT_UNVALIDATED_MINIMUM admits one missing search attempt.
const PRIMALITY_PARAMETER_COUNT_UNVALIDATED_MINIMUM = PRIMALITY_PARAMETER_COUNT_MINIMUM -
	WORD_COUNT_INCREMENT

// PRIMALITY_PARAMETER_COUNT_UNVALIDATED_MAXIMUM admits one excessive search attempt.
const PRIMALITY_PARAMETER_COUNT_UNVALIDATED_MAXIMUM = PRIMALITY_PARAMETER_COUNT_MAXIMUM +
	WORD_COUNT_INCREMENT

// PRIMALITY_TRIAL_FACTOR_MINIMUM is first odd factor after even rejection.
const PRIMALITY_TRIAL_FACTOR_MINIMUM = BASE_BINARY + WORD_COUNT_INCREMENT

// PRIMALITY_TRIAL_FACTOR_MAXIMUM tests every odd factor inside one small-value word interval.
const PRIMALITY_TRIAL_FACTOR_MAXIMUM = WORD_BIT_COUNT - WORD_COUNT_INCREMENT

// PRIMALITY_TRIAL_LIMB_BIT_COUNT keeps a partial remainder and the next limb inside one word.
const PRIMALITY_TRIAL_LIMB_BIT_COUNT = WORD_BIT_COUNT / BASE_BINARY

// PRIMALITY_TRIAL_LIMB_MASK derives the low trial-division limb from the machine word width.
const PRIMALITY_TRIAL_LIMB_MASK = Word(bits.WORD_64_MAXIMUM) >> PRIMALITY_TRIAL_LIMB_BIT_COUNT

// PRIMALITY_DELTA_OFFSET derives method-C discriminant P squared minus four.
const PRIMALITY_DELTA_OFFSET = BASE_BINARY * BASE_BINARY

// PRIMALITY_VALUE_INDEX preserves tested value through every destructive subalgorithm.
const PRIMALITY_VALUE_INDEX = 0

// PRIMALITY_MINUS_ONE_INDEX stores Miller-Rabin upper residue.
const PRIMALITY_MINUS_ONE_INDEX = PRIMALITY_VALUE_INDEX + 1

// PRIMALITY_BOUND_INDEX stores random-base bound, then Lucas value minus two.
const PRIMALITY_BOUND_INDEX = PRIMALITY_MINUS_ONE_INDEX + 1

// PRIMALITY_ODD_FACTOR_INDEX stores odd Miller-Rabin or Lucas exponent factor.
const PRIMALITY_ODD_FACTOR_INDEX = PRIMALITY_BOUND_INDEX + 1

// PRIMALITY_BASE_INDEX stores current Miller-Rabin base or Lucas parameter.
const PRIMALITY_BASE_INDEX = PRIMALITY_ODD_FACTOR_INDEX + 1

// PRIMALITY_RESULT_INDEX stores Miller-Rabin power or current Lucas value.
const PRIMALITY_RESULT_INDEX = PRIMALITY_BASE_INDEX + 1

// PRIMALITY_NEXT_INDEX stores next Lucas value.
const PRIMALITY_NEXT_INDEX = PRIMALITY_RESULT_INDEX + 1

// PRIMALITY_LEFT_INDEX stores first independent Lucas update.
const PRIMALITY_LEFT_INDEX = PRIMALITY_NEXT_INDEX + 1

// PRIMALITY_RIGHT_INDEX stores second independent Lucas update.
const PRIMALITY_RIGHT_INDEX = PRIMALITY_LEFT_INDEX + 1

// PRIMALITY_DIFFERENCE_INDEX keeps modular subtraction transactional.
const PRIMALITY_DIFFERENCE_INDEX = PRIMALITY_RIGHT_INDEX + 1

// PRIMALITY_ONE_INDEX stores multiplicative identity.
const PRIMALITY_ONE_INDEX = PRIMALITY_DIFFERENCE_INDEX + 1

// PRIMALITY_TWO_INDEX stores forced Miller-Rabin base and Lucas recurrence constant.
const PRIMALITY_TWO_INDEX = PRIMALITY_ONE_INDEX + 1

// PRIMALITY_DELTA_INDEX stores method-C discriminant, then Lucas trailing-one count.
const PRIMALITY_DELTA_INDEX = PRIMALITY_TWO_INDEX + 1

// PRIMALITY_INTEGER_COUNT is complete bounded probable-prime state.
const PRIMALITY_INTEGER_COUNT = PRIMALITY_DELTA_INDEX + 1

// JACOBI_NUMERATOR_INDEX preserves numerator while Euclidean reduction rotates values.
const JACOBI_NUMERATOR_INDEX = 0

// JACOBI_DENOMINATOR_INDEX preserves denominator while Euclidean reduction rotates values.
const JACOBI_DENOMINATOR_INDEX = JACOBI_NUMERATOR_INDEX + 1

// JACOBI_ODD_NUMERATOR_INDEX removes powers of two without destroying remainder state.
const JACOBI_ODD_NUMERATOR_INDEX = JACOBI_DENOMINATOR_INDEX + 1

// JACOBI_QUOTIENT_INDEX keeps discarded Euclidean quotient outside rotating values.
const JACOBI_QUOTIENT_INDEX = JACOBI_ODD_NUMERATOR_INDEX + 1

// JACOBI_INTEGER_COUNT is complete bounded Jacobi state.
const JACOBI_INTEGER_COUNT = JACOBI_QUOTIENT_INDEX + 1

// WORD_INDEX_MAXIMUM is final coordinate inside owned magnitude storage.
const WORD_INDEX_MAXIMUM = WORD_COUNT_MAXIMUM - 1

// WORD_BIT_INDEX_MAXIMUM is final bit coordinate inside one magnitude word.
const WORD_BIT_INDEX_MAXIMUM = WORD_BIT_COUNT - 1

// BIT_COUNT_MINIMUM is zero magnitude width.
const BIT_COUNT_MINIMUM = WORD_COUNT_MINIMUM * WORD_BIT_COUNT

// BIT_COUNT_MAXIMUM derives exact arithmetic width from owned word storage.
const BIT_COUNT_MAXIMUM = WORD_COUNT_MAXIMUM * WORD_BIT_COUNT

// BIT_INDEX_UNVALIDATED_MINIMUM admits one hostile negative index for graceful rejection.
const BIT_INDEX_UNVALIDATED_MINIMUM = BIT_COUNT_MINIMUM - WORD_COUNT_INCREMENT

// BIT_INDEX_MAXIMUM is final coordinate inside one bounded Int.
const BIT_INDEX_MAXIMUM = BIT_COUNT_MAXIMUM - WORD_COUNT_INCREMENT

// BIT_INDEX_UNVALIDATED_MAXIMUM admits one hostile index beyond Int storage.
const BIT_INDEX_UNVALIDATED_MAXIMUM = BIT_COUNT_MAXIMUM

// TRAILING_ZERO_BIT_COUNT_MAXIMUM stops below width because one nonzero bit remains.
const TRAILING_ZERO_BIT_COUNT_MAXIMUM = BIT_COUNT_MAXIMUM - 1

// SIGN_BYTE_COUNT_MAXIMUM is one optional minus byte.
const SIGN_BYTE_COUNT_MAXIMUM = 1

// INT_TEXT_SIZE_MAXIMUM holds worst-case binary digits and one sign byte.
const INT_TEXT_SIZE_MAXIMUM = BIT_COUNT_MAXIMUM + SIGN_BYTE_COUNT_MAXIMUM

// TEXT_UNVALIDATED_SIZE_MAXIMUM admits one hostile byte beyond valid integer text.
const TEXT_UNVALIDATED_SIZE_MAXIMUM = INT_TEXT_SIZE_MAXIMUM + 1

// BASE_UNVALIDATED_MINIMUM admits one hostile base below automatic detection.
const BASE_UNVALIDATED_MINIMUM = -1

// BASE_MINIMUM is smallest positional integer base.
const BASE_MINIMUM = 2

// DECIMAL_DIGIT_COUNT is numeric digit alphabet size.
const DECIMAL_DIGIT_COUNT = 10

// LETTER_DIGIT_COUNT is one ASCII letter-case alphabet size.
const LETTER_DIGIT_COUNT = 26

// BASE_MAXIMUM includes numeric, lowercase, and uppercase digits.
const BASE_MAXIMUM = DECIMAL_DIGIT_COUNT + LETTER_DIGIT_COUNT + LETTER_DIGIT_COUNT

// BASE_UNVALIDATED_MAXIMUM admits one hostile base beyond supported alphabet.
const BASE_UNVALIDATED_MAXIMUM = BASE_MAXIMUM + 1

// BASE_AUTOMATIC selects prefix-based base detection.
const BASE_AUTOMATIC = 0

// BASE_BINARY is smallest positional integer base.
const BASE_BINARY = BASE_MINIMUM

// BASE_OCTAL derives one digit's three binary places.
const BASE_OCTAL = BASE_BINARY * BASE_BINARY * BASE_BINARY

// BASE_HEXADECIMAL derives one digit's four binary places.
const BASE_HEXADECIMAL = BASE_OCTAL * BASE_BINARY

// BASE_HEXADECIMAL_DIGIT_BIT_COUNT is one half-byte positional digit.
const BASE_HEXADECIMAL_DIGIT_BIT_COUNT = bits.BIT_COUNT_8_MAXIMUM / BASE_BINARY

// BASE_DECIMAL follows numeric digit alphabet size.
const BASE_DECIMAL = DECIMAL_DIGIT_COUNT

// BASE_LOWERCASE_MAXIMUM consumes numeric and lowercase alphabets.
const BASE_LOWERCASE_MAXIMUM = DECIMAL_DIGIT_COUNT + LETTER_DIGIT_COUNT

// BASE_PREFIX_BYTE_COUNT includes zero marker and alphabet marker.
const BASE_PREFIX_BYTE_COUNT = SIGN_BYTE_COUNT_MAXIMUM + SIGN_BYTE_COUNT_MAXIMUM

// FLOAT_64_SIGN_BIT_COUNT is IEEE 754 binary64 sign width.
const FLOAT_64_SIGN_BIT_COUNT = 1

// FLOAT_64_EXPONENT_BIT_COUNT is IEEE 754 binary64 exponent width.
const FLOAT_64_EXPONENT_BIT_COUNT = 11

// FLOAT_64_MANTISSA_BIT_COUNT derives binary64 stored significand width.
const FLOAT_64_MANTISSA_BIT_COUNT = bits.BIT_COUNT_64_MAXIMUM -
	FLOAT_64_SIGN_BIT_COUNT - FLOAT_64_EXPONENT_BIT_COUNT

// FLOAT_64_EXPONENT_SHIFT places encoded exponent above mantissa.
const FLOAT_64_EXPONENT_SHIFT = FLOAT_64_MANTISSA_BIT_COUNT

// FLOAT_64_SIGN_SHIFT selects highest binary64 bit.
const FLOAT_64_SIGN_SHIFT = bits.BIT_COUNT_64_MAXIMUM - FLOAT_64_SIGN_BIT_COUNT

// FLOAT_64_EXPONENT_COUNT derives every encoded exponent.
const FLOAT_64_EXPONENT_COUNT = 1 << FLOAT_64_EXPONENT_BIT_COUNT

// FLOAT_64_EXPONENT_MASK selects encoded exponent.
const FLOAT_64_EXPONENT_MASK = FLOAT_64_EXPONENT_COUNT - 1

// FLOAT_64_FINITE_EXPONENT_FIELD_MAXIMUM excludes the reserved nonfinite field.
const FLOAT_64_FINITE_EXPONENT_FIELD_MAXIMUM = FLOAT_64_EXPONENT_MASK -
	WORD_COUNT_INCREMENT

// FLOAT_64_EXPONENT_BIAS derives symmetric finite exponent offset.
const FLOAT_64_EXPONENT_BIAS = FLOAT_64_EXPONENT_COUNT/BASE_BINARY - 1

// FLOAT_64_HIDDEN_MANTISSA_BIT restores normal leading significand bit.
const FLOAT_64_HIDDEN_MANTISSA_BIT = uint64(1) << FLOAT_64_MANTISSA_BIT_COUNT

// FLOAT_64_MANTISSA_MASK selects stored significand bits.
const FLOAT_64_MANTISSA_MASK = FLOAT_64_HIDDEN_MANTISSA_BIT - 1

// FLOAT_64_SIGN_MASK selects encoded sign.
const FLOAT_64_SIGN_MASK = uint64(1) << FLOAT_64_SIGN_SHIFT

// FLOAT_64_SUBNORMAL_EXPONENT is exponent before significand scaling.
const FLOAT_64_SUBNORMAL_EXPONENT = 1 - FLOAT_64_EXPONENT_BIAS

// FLOAT_64_SUBNORMAL_DENOMINATOR_SHIFT_MAXIMUM is smallest subnormal denominator power.
const FLOAT_64_SUBNORMAL_DENOMINATOR_SHIFT_MAXIMUM = FLOAT_64_MANTISSA_BIT_COUNT -
	FLOAT_64_SUBNORMAL_EXPONENT

// FLOAT_64_VALUE_BIT_COUNT_MAXIMUM is largest finite binary64 integer width.
const FLOAT_64_VALUE_BIT_COUNT_MAXIMUM = FLOAT_64_EXPONENT_BIAS + 1

// FLOAT_64_VALUE_MANTISSA_BIT_COUNT includes normal hidden bit.
const FLOAT_64_VALUE_MANTISSA_BIT_COUNT = FLOAT_64_MANTISSA_BIT_COUNT + 1

// FLOAT_64_ROUNDING_MANTISSA_BIT_COUNT includes low rounding bit.
const FLOAT_64_ROUNDING_MANTISSA_BIT_COUNT = FLOAT_64_VALUE_MANTISSA_BIT_COUNT + 1

// FLOAT_64_ROUNDING_MANTISSA_LIMIT is carry beyond rounding quotient.
const FLOAT_64_ROUNDING_MANTISSA_LIMIT = uint64(1) <<
	FLOAT_64_ROUNDING_MANTISSA_BIT_COUNT

// FLOAT_64_VALUE_MANTISSA_LIMIT is carry beyond final significand.
const FLOAT_64_VALUE_MANTISSA_LIMIT = uint64(1) << FLOAT_64_VALUE_MANTISSA_BIT_COUNT

// FLOAT_64_VALUE_MANTISSA_MAXIMUM is largest rounded significand.
const FLOAT_64_VALUE_MANTISSA_MAXIMUM = FLOAT_64_VALUE_MANTISSA_LIMIT - 1

// FLOAT_64_SUBNORMAL_EXPONENT_MINIMUM is smallest representable binary64 exponent.
const FLOAT_64_SUBNORMAL_EXPONENT_MINIMUM = FLOAT_64_SUBNORMAL_EXPONENT -
	FLOAT_64_MANTISSA_BIT_COUNT

// FLOAT_64_EXPONENT_ENCODING_OFFSET maps normalized exponent to encoded field.
const FLOAT_64_EXPONENT_ENCODING_OFFSET = FLOAT_64_EXPONENT_BIAS - 1

// FLOAT_64_POSITIVE_INFINITY_BITS encodes positive overflow.
const FLOAT_64_POSITIVE_INFINITY_BITS = uint64(FLOAT_64_EXPONENT_MASK) <<
	FLOAT_64_EXPONENT_SHIFT

// FLOAT_64_NEGATIVE_INFINITY_BITS encodes negative overflow.
const FLOAT_64_NEGATIVE_INFINITY_BITS = FLOAT_64_SIGN_MASK |
	FLOAT_64_POSITIVE_INFINITY_BITS

// FLOAT_64_VALUE_BITS_MINIMUM is positive zero encoding.
const FLOAT_64_VALUE_BITS_MINIMUM = bits.WORD_64_MINIMUM

// FLOAT_64_VALUE_BITS_MAXIMUM is negative infinity encoding.
const FLOAT_64_VALUE_BITS_MAXIMUM = FLOAT_64_NEGATIVE_INFINITY_BITS

// FLOAT_PRECISION_MINIMUM keeps the zero-value precision ready for later operands.
const FLOAT_PRECISION_MINIMUM = BIT_COUNT_MINIMUM

// FLOAT_PRECISION_MAXIMUM binds every mantissa to one inline Int magnitude.
const FLOAT_PRECISION_MAXIMUM = BIT_COUNT_MAXIMUM

// FLOAT_PRECISION_UNVALIDATED_MAXIMUM admits one hostile precision beyond storage.
const FLOAT_PRECISION_UNVALIDATED_MAXIMUM = FLOAT_PRECISION_MAXIMUM + WORD_COUNT_INCREMENT

// FLOAT_DISCARDED_BIT_COUNT_MAXIMUM leaves one retained bit and one separate rounding bit.
const FLOAT_DISCARDED_BIT_COUNT_MAXIMUM = BIT_INDEX_MAXIMUM - WORD_COUNT_INCREMENT

// FLOAT_EXPONENT_MINIMUM bounds normalized underflow by complete mantissa width.
const FLOAT_EXPONENT_MINIMUM = -BIT_COUNT_MAXIMUM

// FLOAT_EXPONENT_ZERO is the absent scale payload for zero and infinity.
const FLOAT_EXPONENT_ZERO Float_Exponent = 0

// FLOAT_EXPONENT_MAXIMUM bounds normalized overflow by complete mantissa width.
const FLOAT_EXPONENT_MAXIMUM = BIT_COUNT_MAXIMUM

// FLOAT_EXPONENT_UNVALIDATED_MINIMUM admits one hostile exponent below finite range.
const FLOAT_EXPONENT_UNVALIDATED_MINIMUM = FLOAT_EXPONENT_MINIMUM - WORD_COUNT_INCREMENT

// FLOAT_EXPONENT_UNVALIDATED_MAXIMUM admits one hostile exponent above finite range.
const FLOAT_EXPONENT_UNVALIDATED_MAXIMUM = FLOAT_EXPONENT_MAXIMUM + WORD_COUNT_INCREMENT

// FLOAT_LEAST_BIT_EXPONENT_MINIMUM places a full mantissa at minimum normalized exponent.
const FLOAT_LEAST_BIT_EXPONENT_MINIMUM = FLOAT_EXPONENT_MINIMUM - BIT_COUNT_MAXIMUM

// FLOAT_LEAST_BIT_EXPONENT_MAXIMUM places a one-bit mantissa at maximum exponent.
const FLOAT_LEAST_BIT_EXPONENT_MAXIMUM = FLOAT_EXPONENT_MAXIMUM - WORD_COUNT_INCREMENT

// FLOAT_RESULT_ORIGIN_MINIMUM includes the product of two lowest finite bit positions.
const FLOAT_RESULT_ORIGIN_MINIMUM = BASE_BINARY * FLOAT_LEAST_BIT_EXPONENT_MINIMUM

// FLOAT_RESULT_ORIGIN_MAXIMUM includes the product of two highest finite bit positions.
const FLOAT_RESULT_ORIGIN_MAXIMUM = BASE_BINARY * FLOAT_LEAST_BIT_EXPONENT_MAXIMUM

// FLOAT_ADDITION_BIT_COUNT_MAXIMUM spans every finite operand bit position.
const FLOAT_ADDITION_BIT_COUNT_MAXIMUM = FLOAT_LEAST_BIT_EXPONENT_MAXIMUM -
	FLOAT_LEAST_BIT_EXPONENT_MINIMUM + WORD_COUNT_INCREMENT

// FLOAT_ADDITION_WORD_COUNT_MAXIMUM rounds the complete alignment span to words.
const FLOAT_ADDITION_WORD_COUNT_MAXIMUM = (FLOAT_ADDITION_BIT_COUNT_MAXIMUM +
	WORD_BIT_COUNT - WORD_COUNT_INCREMENT) / WORD_BIT_COUNT

// FLOAT_MULTIPLICATION_WORD_COUNT_MAXIMUM holds two complete mantissa widths.
const FLOAT_MULTIPLICATION_WORD_COUNT_MAXIMUM = BASE_BINARY * WORD_COUNT_MAXIMUM

// FLOAT_DIVISION_WORD_COUNT_MAXIMUM holds one shifted remainder carry word.
const FLOAT_DIVISION_WORD_COUNT_MAXIMUM = WORD_COUNT_MAXIMUM + WORD_COUNT_INCREMENT

// FLOAT_DIVISION_WORD_COUNT_MINIMUM retains one significand word and one carry word.
const FLOAT_DIVISION_WORD_COUNT_MINIMUM = WORD_COUNT_INCREMENT + WORD_COUNT_INCREMENT

// FLOAT_DIVISION_DIVISOR_COUNT_INDEX stores the normalized short-divisor width.
const FLOAT_DIVISION_DIVISOR_COUNT_INDEX = WORD_COUNT_MINIMUM

// FLOAT_DIVISION_DIVIDEND_COUNT_INDEX stores the scaled numerator width.
const FLOAT_DIVISION_DIVIDEND_COUNT_INDEX = FLOAT_DIVISION_DIVISOR_COUNT_INDEX +
	WORD_COUNT_INCREMENT

// FLOAT_DIVISION_SHIFT_INDEX stores the guarded numerator scale.
const FLOAT_DIVISION_SHIFT_INDEX = FLOAT_DIVISION_DIVIDEND_COUNT_INDEX +
	WORD_COUNT_INCREMENT

// FLOAT_DIVISION_OFFSET_INDEX stores the quotient word under construction.
const FLOAT_DIVISION_OFFSET_INDEX = FLOAT_DIVISION_SHIFT_INDEX + WORD_COUNT_INCREMENT

// FLOAT_DIVISION_QUOTIENT_COUNT_INDEX stores the normalized result width.
const FLOAT_DIVISION_QUOTIENT_COUNT_INDEX = FLOAT_DIVISION_OFFSET_INDEX + WORD_COUNT_INCREMENT

// FLOAT_DIVISION_INCREMENT_INDEX stores the rounding carry decision.
const FLOAT_DIVISION_INCREMENT_INDEX = FLOAT_DIVISION_QUOTIENT_COUNT_INDEX + WORD_COUNT_INCREMENT

// FLOAT_DIVISION_CONTROL_COUNT is complete word-division scalar state.
const FLOAT_DIVISION_CONTROL_COUNT = FLOAT_DIVISION_INCREMENT_INDEX +
	WORD_COUNT_INCREMENT

// FLOAT_ADDITION_SHIFT_MAXIMUM stops at the opposite end of the alignment span.
const FLOAT_ADDITION_SHIFT_MAXIMUM = FLOAT_ADDITION_BIT_COUNT_MAXIMUM -
	WORD_COUNT_INCREMENT

// FLOAT_ADDITION_ROUNDING_INDEX_MAXIMUM leaves retained and rounding bits in the span.
const FLOAT_ADDITION_ROUNDING_INDEX_MAXIMUM = FLOAT_ADDITION_BIT_COUNT_MAXIMUM - BASE_BINARY

// FLOAT_FORM_ZERO stores either signed zero without a mantissa.
const FLOAT_FORM_ZERO Float_Form = 0

// FLOAT_FORM_FINITE stores one nonzero bounded mantissa.
const FLOAT_FORM_FINITE = FLOAT_FORM_ZERO + 1

// FLOAT_FORM_INFINITY stores either signed infinity without a mantissa.
const FLOAT_FORM_INFINITY = FLOAT_FORM_FINITE + 1

// ROUND_TO_NEAREST_EVEN matches IEEE round-to-nearest ties-to-even.
const ROUND_TO_NEAREST_EVEN Rounding_Mode_Unvalidated = 0

// ROUND_TO_NEAREST_AWAY rounds midpoint magnitude away from zero.
const ROUND_TO_NEAREST_AWAY = ROUND_TO_NEAREST_EVEN + 1

// ROUND_TO_ZERO truncates discarded magnitude.
const ROUND_TO_ZERO = ROUND_TO_NEAREST_AWAY + 1

// ROUND_AWAY_FROM_ZERO increments every inexact magnitude.
const ROUND_AWAY_FROM_ZERO = ROUND_TO_ZERO + 1

// ROUND_TO_NEGATIVE_INFINITY rounds toward the ordered lower value.
const ROUND_TO_NEGATIVE_INFINITY = ROUND_AWAY_FROM_ZERO + 1

// ROUND_TO_POSITIVE_INFINITY rounds toward the ordered higher value.
const ROUND_TO_POSITIVE_INFINITY = ROUND_TO_NEGATIVE_INFINITY + 1

// ROUNDING_MODE_UNVALIDATED_MAXIMUM admits one hostile mode beyond the enum.
const ROUNDING_MODE_UNVALIDATED_MAXIMUM = ROUND_TO_POSITIVE_INFINITY + 1

// ACCURACY_BELOW reports a rounded value below its exact source.
const ACCURACY_BELOW Accuracy = -1

// ACCURACY_EXACT reports no rounding error.
const ACCURACY_EXACT Accuracy = 0

// ACCURACY_ABOVE reports a rounded value above its exact source.
const ACCURACY_ABOVE Accuracy = ACCURACY_EXACT + 1

// RAT_FLOAT_64_EXPONENT_MINIMUM follows smallest component ratio.
const RAT_FLOAT_64_EXPONENT_MINIMUM = 1 - RAT_COMPONENT_TEXT_DIGIT_COUNT_MAXIMUM

// RAT_FLOAT_64_EXPONENT_MAXIMUM includes normalization and rounding carries.
const RAT_FLOAT_64_EXPONENT_MAXIMUM = RAT_COMPONENT_TEXT_DIGIT_COUNT_MAXIMUM + 1

// PARSE_TEXT_INDEX_MAXIMUM skips optional sign and complete base prefix.
const PARSE_TEXT_INDEX_MAXIMUM = SIGN_BYTE_COUNT_MAXIMUM + BASE_PREFIX_BYTE_COUNT

// PARSE_DIGIT_MINIMUM is first digit in supported alphabet.
const PARSE_DIGIT_MINIMUM uint8 = 0

// PARSE_DIGIT_MAXIMUM is final digit in supported alphabet.
const PARSE_DIGIT_MAXIMUM uint8 = BASE_MAXIMUM - 1

// STATUS_OK distinguishes successful zero results from failure.
const STATUS_OK = 0

// STATUS_INPUT_INVALID rejects hostile encoded input before mutation.
const STATUS_INPUT_INVALID = STATUS_OK + 1

// STATUS_VALUE_OVERFLOW rejects exact results beyond fixed storage or machine destination.
const STATUS_VALUE_OVERFLOW = STATUS_INPUT_INVALID + 1

// STATUS_DESTINATION_TOO_SMALL preserves caller storage when exact output cannot fit.
const STATUS_DESTINATION_TOO_SMALL = STATUS_VALUE_OVERFLOW + 1

// STATUS_DIVISOR_ZERO rejects undefined division before result mutation.
const STATUS_DIVISOR_ZERO = STATUS_DESTINATION_TOO_SMALL + 1

// STATUS_DESTINATIONS_OVERLAP rejects two logical results sharing one Int.
const STATUS_DESTINATIONS_OVERLAP = STATUS_DIVISOR_ZERO + 1

// STATUS_RESULT_ABSENT reports a modular result that does not mathematically exist.
const STATUS_RESULT_ABSENT = STATUS_DESTINATIONS_OVERLAP + 1

// STATUS_SOURCE_EXHAUSTED asks caller for another bounded random-word batch.
const STATUS_SOURCE_EXHAUSTED = STATUS_RESULT_ABSENT + 1

// STATUS_SEARCH_EXHAUSTED reports explicit Lucas parameter budget exhaustion.
const STATUS_SEARCH_EXHAUSTED = STATUS_SOURCE_EXHAUSTED + 1

// BYTES_UNVALIDATED_SIZE_MAXIMUM admits one hostile byte beyond validated storage.
const BYTES_UNVALIDATED_SIZE_MAXIMUM = bytes.SLICE_SIZE_MAXIMUM + 1

// INT_GOB_VERSION is the stdlib wire version retained for byte compatibility.
const INT_GOB_VERSION = 1

// INT_GOB_VERSION_SHIFT leaves the low header bit for sign.
const INT_GOB_VERSION_SHIFT = 1

// INT_GOB_NEGATIVE_MASK selects the low header sign bit.
const INT_GOB_NEGATIVE_MASK = 1

// INT_GOB_HEADER_SIZE is the version-and-sign byte before magnitude.
const INT_GOB_HEADER_SIZE = 1

// INT_GOB_SIZE_MAXIMUM holds one complete bounded magnitude and its header.
const INT_GOB_SIZE_MAXIMUM = INT_GOB_HEADER_SIZE + bytes.SLICE_SIZE_MAXIMUM

// INT_GOB_UNVALIDATED_SIZE_MAXIMUM admits one hostile byte beyond a complete encoding.
const INT_GOB_UNVALIDATED_SIZE_MAXIMUM = INT_GOB_SIZE_MAXIMUM + 1

// RAT_COMPONENT_BYTE_SIZE_MAXIMUM derives one component byte bound from owned words.
const RAT_COMPONENT_BYTE_SIZE_MAXIMUM = RAT_WORD_COUNT_MAXIMUM * WORD_BYTE_COUNT

// RAT_GOB_VERSION is stdlib rational wire version.
const RAT_GOB_VERSION = 1

// RAT_GOB_VERSION_SHIFT leaves low header bit for numerator sign.
const RAT_GOB_VERSION_SHIFT = 1

// RAT_GOB_NEGATIVE_MASK selects low header sign bit.
const RAT_GOB_NEGATIVE_MASK = 1

// RAT_GOB_HEADER_SIZE is version-and-sign byte count.
const RAT_GOB_HEADER_SIZE = 1

// RAT_GOB_NUMERATOR_SIZE_FIELD_SIZE holds one big-endian uint32.
const RAT_GOB_NUMERATOR_SIZE_FIELD_SIZE = bits.BIT_COUNT_32_MAXIMUM /
	bits.BIT_COUNT_8_MAXIMUM

// RAT_GOB_PREFIX_SIZE includes header and numerator byte count.
const RAT_GOB_PREFIX_SIZE = RAT_GOB_HEADER_SIZE + RAT_GOB_NUMERATOR_SIZE_FIELD_SIZE

// RAT_GOB_NUMERATOR_SIZE_OFFSET locates numerator byte count after header.
const RAT_GOB_NUMERATOR_SIZE_OFFSET = RAT_GOB_HEADER_SIZE

// RAT_GOB_COMPONENTS_SIZE_MAXIMUM holds maximum numerator and denominator magnitudes.
const RAT_GOB_COMPONENTS_SIZE_MAXIMUM = RAT_COMPONENT_BYTE_SIZE_MAXIMUM +
	RAT_COMPONENT_BYTE_SIZE_MAXIMUM

// RAT_GOB_SIZE_MAXIMUM holds prefix and both maximum rational components.
const RAT_GOB_SIZE_MAXIMUM = RAT_GOB_PREFIX_SIZE + RAT_GOB_COMPONENTS_SIZE_MAXIMUM

// RAT_GOB_UNVALIDATED_SIZE_MAXIMUM admits one hostile byte beyond complete encoding.
const RAT_GOB_UNVALIDATED_SIZE_MAXIMUM = RAT_GOB_SIZE_MAXIMUM + 1

// RAT_GOB_VALUE_INDEX selects transactional decoded rational.
const RAT_GOB_VALUE_INDEX = 0

// RAT_GOB_VALUE_COUNT is complete transactional decode state.
const RAT_GOB_VALUE_COUNT = RAT_GOB_VALUE_INDEX + 1

// SIGN_NEGATIVE uses comparison ordering so sign can feed ordered decisions without conversion.
const SIGN_NEGATIVE Sign = -1

// SIGN_ZERO sits between negative and positive values.
const SIGN_ZERO Sign = SIGN_NEGATIVE + 1

// SIGN_POSITIVE closes signed ordering.
const SIGN_POSITIVE Sign = SIGN_ZERO + 1

// JACOBI_SYMBOL_NEGATIVE matches quadratic nonresidue convention.
const JACOBI_SYMBOL_NEGATIVE Jacobi_Symbol = -1

// JACOBI_SYMBOL_ZERO reports a shared factor.
const JACOBI_SYMBOL_ZERO Jacobi_Symbol = JACOBI_SYMBOL_NEGATIVE + 1

// JACOBI_SYMBOL_POSITIVE matches quadratic residue convention.
const JACOBI_SYMBOL_POSITIVE Jacobi_Symbol = JACOBI_SYMBOL_ZERO + 1

// JACOBI_PARITY_MASK selects parity without division.
const JACOBI_PARITY_MASK Word = BASE_BINARY - WORD_COUNT_INCREMENT

// JACOBI_SUPPLEMENT_MASK selects denominator modulo eight for the two supplement.
const JACOBI_SUPPLEMENT_MASK Word = BASE_OCTAL - WORD_COUNT_INCREMENT

// JACOBI_SUPPLEMENT_RESIDUE_LOW is first modulo-eight residue that flips the two supplement.
const JACOBI_SUPPLEMENT_RESIDUE_LOW Word = BASE_OCTAL/BASE_BINARY - WORD_COUNT_INCREMENT

// JACOBI_SUPPLEMENT_RESIDUE_HIGH is second modulo-eight residue that flips the two supplement.
const JACOBI_SUPPLEMENT_RESIDUE_HIGH Word = BASE_OCTAL - JACOBI_SUPPLEMENT_RESIDUE_LOW

// JACOBI_RECIPROCITY_MASK selects modulo-four residue three for quadratic reciprocity.
const JACOBI_RECIPROCITY_MASK Word = BASE_OCTAL/BASE_BINARY - WORD_COUNT_INCREMENT

// PRIMALITY_TRIAL_COMPOSITE closes testing before expensive probable-prime work.
const PRIMALITY_TRIAL_COMPOSITE Primality_Trial_Result = 0

// PRIMALITY_TRIAL_PRIME closes testing with exact small-value proof.
const PRIMALITY_TRIAL_PRIME Primality_Trial_Result = PRIMALITY_TRIAL_COMPOSITE + 1

// PRIMALITY_TRIAL_UNDETERMINED requires Miller-Rabin and Lucas tests.
const PRIMALITY_TRIAL_UNDETERMINED Primality_Trial_Result = PRIMALITY_TRIAL_PRIME + 1

// LUCAS_UPDATE_PRODUCT computes V(k)V(k+1)-P.
const LUCAS_UPDATE_PRODUCT Lucas_Update = 0

// LUCAS_UPDATE_CURRENT_SQUARE computes V(k) squared minus two.
const LUCAS_UPDATE_CURRENT_SQUARE Lucas_Update = LUCAS_UPDATE_PRODUCT + 1

// LUCAS_UPDATE_NEXT_SQUARE computes V(k+1) squared minus two.
const LUCAS_UPDATE_NEXT_SQUARE Lucas_Update = LUCAS_UPDATE_CURRENT_SQUARE + 1

// ORDER_BEFORE shares sign ordering while retaining comparison identity.
const ORDER_BEFORE Order = Order(SIGN_NEGATIVE)

// ORDER_SAME shares zero position while retaining comparison identity.
const ORDER_SAME Order = Order(SIGN_ZERO)

// ORDER_AFTER shares positive position while retaining comparison identity.
const ORDER_AFTER Order = Order(SIGN_POSITIVE)

// POLARITY_NONNEGATIVE gives zero and positive values one stored polarity.
const POLARITY_NONNEGATIVE Polarity = 0

// POLARITY_NEGATIVE marks nonzero negative values.
const POLARITY_NEGATIVE Polarity = POLARITY_NONNEGATIVE + 1

// BITWISE_OPERATION_AND computes intersection of infinite signed bit strings.
const BITWISE_OPERATION_AND Bitwise_Operation = 0

// BITWISE_OPERATION_AND_NOT clears right bits from left bits.
const BITWISE_OPERATION_AND_NOT Bitwise_Operation = BITWISE_OPERATION_AND + 1

// BITWISE_OPERATION_OR computes union of infinite signed bit strings.
const BITWISE_OPERATION_OR Bitwise_Operation = BITWISE_OPERATION_AND_NOT + 1

// BITWISE_OPERATION_XOR computes difference of infinite signed bit strings.
const BITWISE_OPERATION_XOR Bitwise_Operation = BITWISE_OPERATION_OR + 1

// BIT_CLEAR is one cleared signed bit.
const BIT_CLEAR Bit_Value = 0

// BIT_SET is one set signed bit.
const BIT_SET Bit_Value = BIT_CLEAR + 1

// RAT_OPERATION_ADD selects rational addition.
const RAT_OPERATION_ADD Rat_Operation = 0

// RAT_OPERATION_SUBTRACT selects rational subtraction.
const RAT_OPERATION_SUBTRACT Rat_Operation = RAT_OPERATION_ADD + 1

// RAT_OPERATION_MULTIPLY selects rational multiplication.
const RAT_OPERATION_MULTIPLY Rat_Operation = RAT_OPERATION_SUBTRACT + 1

// RAT_OPERATION_QUOTIENT selects rational division.
const RAT_OPERATION_QUOTIENT Rat_Operation = RAT_OPERATION_MULTIPLY + 1

// EUCLIDEAN_DIVIDEND_INDEX selects current dividend scratch.
const EUCLIDEAN_DIVIDEND_INDEX = 0

// EUCLIDEAN_DIVISOR_INDEX selects current divisor scratch.
const EUCLIDEAN_DIVISOR_INDEX = EUCLIDEAN_DIVIDEND_INDEX + 1

// EUCLIDEAN_REMAINDER_INDEX selects next remainder scratch.
const EUCLIDEAN_REMAINDER_INDEX = EUCLIDEAN_DIVISOR_INDEX + 1

// EUCLIDEAN_INTEGER_COUNT is complete rotating Euclidean state.
const EUCLIDEAN_INTEGER_COUNT = EUCLIDEAN_REMAINDER_INDEX + 1

// SQUARE_ROOT_CURRENT_INDEX selects current upper approximation.
const SQUARE_ROOT_CURRENT_INDEX = 0

// SQUARE_ROOT_QUOTIENT_INDEX selects source divided by current approximation.
const SQUARE_ROOT_QUOTIENT_INDEX = SQUARE_ROOT_CURRENT_INDEX + 1

// SQUARE_ROOT_NEXT_INDEX selects next Newton approximation.
const SQUARE_ROOT_NEXT_INDEX = SQUARE_ROOT_QUOTIENT_INDEX + 1

// SQUARE_ROOT_INTEGER_COUNT is complete Newton state.
const SQUARE_ROOT_INTEGER_COUNT = SQUARE_ROOT_NEXT_INDEX + 1

// EXPONENT_RESULT_INDEX selects accumulated exponentiation result.
const EXPONENT_RESULT_INDEX = 0

// EXPONENT_FACTOR_INDEX selects current squared base.
const EXPONENT_FACTOR_INDEX = EXPONENT_RESULT_INDEX + 1

// EXPONENT_VALUE_INDEX preserves exponent when destination aliases it.
const EXPONENT_VALUE_INDEX = EXPONENT_FACTOR_INDEX + 1

// EXPONENT_INTEGER_COUNT is complete binary exponentiation state.
const EXPONENT_INTEGER_COUNT = EXPONENT_VALUE_INDEX + 1

// MODULAR_GCD_DIVIDEND_INDEX selects the older Euclidean remainder.
const MODULAR_GCD_DIVIDEND_INDEX = 0

// MODULAR_GCD_DIVISOR_INDEX selects the current Euclidean remainder.
const MODULAR_GCD_DIVISOR_INDEX = MODULAR_GCD_DIVIDEND_INDEX + 1

// MODULAR_GCD_REMAINDER_INDEX selects the next Euclidean remainder.
const MODULAR_GCD_REMAINDER_INDEX = MODULAR_GCD_DIVISOR_INDEX + 1

// MODULAR_COEFFICIENT_DIVIDEND_INDEX selects the older modular coefficient.
const MODULAR_COEFFICIENT_DIVIDEND_INDEX = MODULAR_GCD_REMAINDER_INDEX + 1

// MODULAR_COEFFICIENT_DIVISOR_INDEX selects the current modular coefficient.
const MODULAR_COEFFICIENT_DIVISOR_INDEX = MODULAR_COEFFICIENT_DIVIDEND_INDEX + 1

// MODULAR_COEFFICIENT_REMAINDER_INDEX selects the next modular coefficient.
const MODULAR_COEFFICIENT_REMAINDER_INDEX = MODULAR_COEFFICIENT_DIVISOR_INDEX + 1

// MODULAR_QUOTIENT_INDEX selects Euclidean quotient scratch.
const MODULAR_QUOTIENT_INDEX = MODULAR_COEFFICIENT_REMAINDER_INDEX + 1

// MODULAR_MODULUS_INDEX selects the positive modulus copy.
const MODULAR_MODULUS_INDEX = MODULAR_QUOTIENT_INDEX + 1

// MODULAR_MULTIPLICATION_FACTOR_INDEX selects reduced multiplicand scratch.
const MODULAR_MULTIPLICATION_FACTOR_INDEX = MODULAR_MODULUS_INDEX + 1

// MODULAR_MULTIPLICATION_VALUE_INDEX selects multiplier scratch.
const MODULAR_MULTIPLICATION_VALUE_INDEX = MODULAR_MULTIPLICATION_FACTOR_INDEX + 1

// MODULAR_MULTIPLICATION_RESULT_INDEX selects accumulated product scratch.
const MODULAR_MULTIPLICATION_RESULT_INDEX = MODULAR_MULTIPLICATION_VALUE_INDEX + 1

// MODULAR_THRESHOLD_INDEX selects overflow-free addition threshold scratch.
const MODULAR_THRESHOLD_INDEX = MODULAR_MULTIPLICATION_RESULT_INDEX + 1

// MODULAR_INTEGER_COUNT is complete modular arithmetic state.
const MODULAR_INTEGER_COUNT = MODULAR_THRESHOLD_INDEX + 1

// MODULAR_EXPONENT_RESULT_INDEX reuses obsolete Euclidean dividend state.
const MODULAR_EXPONENT_RESULT_INDEX = MODULAR_GCD_DIVIDEND_INDEX

// MODULAR_EXPONENT_FACTOR_INDEX reuses obsolete Euclidean divisor state.
const MODULAR_EXPONENT_FACTOR_INDEX = MODULAR_GCD_DIVISOR_INDEX

// MODULAR_EXPONENT_VALUE_INDEX reuses obsolete Euclidean remainder state.
const MODULAR_EXPONENT_VALUE_INDEX = MODULAR_GCD_REMAINDER_INDEX

// MODULAR_REDUCTION_OPERAND selects the shared exponent or Euclidean operand slot.
const MODULAR_REDUCTION_OPERAND Modular_Reduction_Operation = 0

// MODULAR_REDUCTION_FACTOR selects modular multiplication factor scratch.
const MODULAR_REDUCTION_FACTOR = MODULAR_REDUCTION_OPERAND + 1

// MODULAR_MULTIPLICATION_ACCUMULATE multiplies result by exponent factor.
const MODULAR_MULTIPLICATION_ACCUMULATE Modular_Multiplication_Operation = 0

// MODULAR_MULTIPLICATION_SQUARE squares the exponent factor.
const MODULAR_MULTIPLICATION_SQUARE = MODULAR_MULTIPLICATION_ACCUMULATE + 1

// MODULAR_MULTIPLICATION_COEFFICIENT multiplies Euclidean quotient by its coefficient.
const MODULAR_MULTIPLICATION_COEFFICIENT = MODULAR_MULTIPLICATION_SQUARE + 1

// MODULAR_ADDITION_ACCUMULATE adds factor into product result.
const MODULAR_ADDITION_ACCUMULATE Modular_Addition_Operation = 0

// MODULAR_ADDITION_DOUBLE doubles the current modular factor.
const MODULAR_ADDITION_DOUBLE = MODULAR_ADDITION_ACCUMULATE + 1

// PRODUCT_ACCUMULATOR_INDEX selects current range product.
const PRODUCT_ACCUMULATOR_INDEX = 0

// PRODUCT_FACTOR_INDEX selects current machine-word factor.
const PRODUCT_FACTOR_INDEX = PRODUCT_ACCUMULATOR_INDEX + 1

// PRODUCT_INTEGER_COUNT is complete bounded range-product state.
const PRODUCT_INTEGER_COUNT = PRODUCT_FACTOR_INDEX + 1

// SQUARE_ROOT_DEGREE is root degree and Newton averaging divisor.
const SQUARE_ROOT_DEGREE = 2

// SQUARE_ROOT_AVERAGE_SHIFT divides Newton sum by root degree.
const SQUARE_ROOT_AVERAGE_SHIFT Shift_Count = SQUARE_ROOT_DEGREE - 1

// RAT_NUMERATOR_INDEX selects stored signed numerator.
const RAT_NUMERATOR_INDEX = 0

// RAT_DENOMINATOR_INDEX selects stored positive denominator, with zero meaning one.
const RAT_DENOMINATOR_INDEX = RAT_NUMERATOR_INDEX + 1

// RAT_COMPONENT_COUNT is complete stored rational state.
const RAT_COMPONENT_COUNT = RAT_DENOMINATOR_INDEX + 1

// RAT_WORD_COUNT_MAXIMUM leaves one word between cross-product sums and Int bound.
const RAT_WORD_COUNT_MAXIMUM = WORD_COUNT_MAXIMUM/RAT_COMPONENT_COUNT - WORD_COUNT_INCREMENT

// RAT_COMPONENT_BIT_COUNT_MAXIMUM derives one stored component width from its word capacity.
const RAT_COMPONENT_BIT_COUNT_MAXIMUM = RAT_WORD_COUNT_MAXIMUM * WORD_BIT_COUNT

// RAT_COMPONENT_TEXT_DIGIT_COUNT_MAXIMUM is one component's binary-dominant text bound.
const RAT_COMPONENT_TEXT_DIGIT_COUNT_MAXIMUM = RAT_WORD_COUNT_MAXIMUM * WORD_BIT_COUNT

// DECIMAL_DIGIT_BINARY_LOGARITHM_SHIFT selects proven fixed-point log10(2) precision.
const DECIMAL_DIGIT_BINARY_LOGARITHM_SHIFT = bits.DECIMAL_DIGIT_BINARY_LOGARITHM_SHIFT

// DECIMAL_DIGIT_BINARY_LOGARITHM_SCALE derives fixed-point denominator from its precision.
const DECIMAL_DIGIT_BINARY_LOGARITHM_SCALE = bits.DECIMAL_DIGIT_BINARY_LOGARITHM_SCALE

// DECIMAL_DIGIT_BINARY_LOGARITHM_CEILING is least scale-12 integer above log10(2).
const DECIMAL_DIGIT_BINARY_LOGARITHM_CEILING = bits.DECIMAL_DIGIT_BINARY_LOGARITHM_CEILING

// RAT_COMPONENT_DECIMAL_DIGIT_NUMERATOR scales final significant bit coordinate.
const RAT_COMPONENT_DECIMAL_DIGIT_NUMERATOR = (RAT_COMPONENT_TEXT_DIGIT_COUNT_MAXIMUM - 1) *
	DECIMAL_DIGIT_BINARY_LOGARITHM_CEILING

// RAT_COMPONENT_DECIMAL_DIGIT_QUOTIENT converts scaled binary width to decimal places.
const RAT_COMPONENT_DECIMAL_DIGIT_QUOTIENT = RAT_COMPONENT_DECIMAL_DIGIT_NUMERATOR /
	DECIMAL_DIGIT_BINARY_LOGARITHM_SCALE

// RAT_COMPONENT_DECIMAL_DIGIT_COUNT_MAXIMUM includes leading decimal digit.
const RAT_COMPONENT_DECIMAL_DIGIT_COUNT_MAXIMUM = RAT_COMPONENT_DECIMAL_DIGIT_QUOTIENT + 1

// RAT_PRECISION_MINIMUM permits integer rounding without fractional digits.
const RAT_PRECISION_MINIMUM = 0

// RAT_PRECISION_MAXIMUM keeps decimal scaling within one full Int product.
const RAT_PRECISION_MAXIMUM = RAT_COMPONENT_DECIMAL_DIGIT_COUNT_MAXIMUM

// RAT_PRECISION_UNVALIDATED_MINIMUM admits one hostile negative precision.
const RAT_PRECISION_UNVALIDATED_MINIMUM = RAT_PRECISION_MINIMUM - 1

// RAT_PRECISION_UNVALIDATED_MAXIMUM admits one hostile excessive precision.
const RAT_PRECISION_UNVALIDATED_MAXIMUM = RAT_PRECISION_MAXIMUM + 1

// RAT_FLOAT_PRECISION_COUNT_MINIMUM is integer decimal representation.
const RAT_FLOAT_PRECISION_COUNT_MINIMUM = 0

// RAT_FLOAT_PRECISION_COUNT_MAXIMUM is one less than component binary width.
const RAT_FLOAT_PRECISION_COUNT_MAXIMUM = RAT_COMPONENT_TEXT_DIGIT_COUNT_MAXIMUM - 1

// RAT_DECIMAL_PRIME_FACTOR is odd prime factor of decimal base.
const RAT_DECIMAL_PRIME_FACTOR = BASE_DECIMAL / BASE_BINARY

// RAT_DECIMAL_FACTOR_REMAINDER_MAXIMUM is final remainder below decimal prime factor.
const RAT_DECIMAL_FACTOR_REMAINDER_MAXIMUM = RAT_DECIMAL_PRIME_FACTOR - 1

// DECIMAL_POINT_BYTE_COUNT is one radix separator in fixed decimal text.
const DECIMAL_POINT_BYTE_COUNT = len(".")

// RAT_COMPONENT_DIGIT_COUNT_MINIMUM keeps zero explicit.
const RAT_COMPONENT_DIGIT_COUNT_MINIMUM = SIGN_BYTE_COUNT_MAXIMUM

// RAT_FLOAT_TEXT_DESTINATION_SIZE_MINIMUM holds rounded zero.
const RAT_FLOAT_TEXT_DESTINATION_SIZE_MINIMUM = RAT_TEXT_SIZE_MINIMUM

// RAT_FLOAT_TEXT_DESTINATION_SIZE_MAXIMUM equals complete rational decimal output.
const RAT_FLOAT_TEXT_DESTINATION_SIZE_MAXIMUM = RAT_TEXT_SIZE_MAXIMUM

// RATIONAL_SEPARATOR_BYTE_COUNT is one slash between exact components.
const RATIONAL_SEPARATOR_BYTE_COUNT = 1

// RAT_TEXT_SIZE_MAXIMUM holds two component bounds, numerator sign, and slash.
const RAT_TEXT_SIZE_MAXIMUM = RAT_COMPONENT_DECIMAL_DIGIT_COUNT_MAXIMUM*RAT_COMPONENT_COUNT +
	SIGN_BYTE_COUNT_MAXIMUM + RATIONAL_SEPARATOR_BYTE_COUNT

// RAT_TEXT_SIZE_MINIMUM is compact zero width.
const RAT_TEXT_SIZE_MINIMUM = SIGN_BYTE_COUNT_MAXIMUM

// RAT_FRACTION_TEXT_SIZE_MINIMUM holds zero, slash, and denominator one.
const RAT_FRACTION_TEXT_SIZE_MINIMUM = RAT_TEXT_SIZE_MINIMUM +
	RATIONAL_SEPARATOR_BYTE_COUNT + SIGN_BYTE_COUNT_MAXIMUM

// RAT_PARSE_COMPONENT_SIZE_MAXIMUM includes longest binary prefix.
const RAT_PARSE_COMPONENT_SIZE_MAXIMUM = RAT_COMPONENT_TEXT_DIGIT_COUNT_MAXIMUM +
	BASE_PREFIX_BYTE_COUNT

// RAT_PARSE_COMPONENTS_SIZE_MAXIMUM holds both prefixed components.
const RAT_PARSE_COMPONENTS_SIZE_MAXIMUM = RAT_PARSE_COMPONENT_SIZE_MAXIMUM * RAT_COMPONENT_COUNT

// RAT_PARSE_FRACTION_TEXT_SIZE_MAXIMUM adds sign and slash to both components.
const RAT_PARSE_FRACTION_TEXT_SIZE_MAXIMUM = RAT_PARSE_COMPONENTS_SIZE_MAXIMUM +
	SIGN_BYTE_COUNT_MAXIMUM + RATIONAL_SEPARATOR_BYTE_COUNT

// RAT_PARSE_FRACTION_TEXT_UNVALIDATED_SIZE_MAXIMUM admits one oversized source byte.
const RAT_PARSE_FRACTION_TEXT_UNVALIDATED_SIZE_MAXIMUM = RAT_PARSE_FRACTION_TEXT_SIZE_MAXIMUM + 1

// RAT_PARSE_TEXT_SIZE_MAXIMUM covers the longer integer-text bound and every exact fraction.
const RAT_PARSE_TEXT_SIZE_MAXIMUM = INT_TEXT_SIZE_MAXIMUM

// RAT_PARSE_TEXT_UNVALIDATED_SIZE_MAXIMUM admits one oversized hostile source byte.
const RAT_PARSE_TEXT_UNVALIDATED_SIZE_MAXIMUM = RAT_PARSE_TEXT_SIZE_MAXIMUM + 1

// RAT_PARSE_EXPONENT_MAGNITUDE_MAXIMUM bounds exponent work by complete integer storage.
const RAT_PARSE_EXPONENT_MAGNITUDE_MAXIMUM = BIT_COUNT_MAXIMUM

// RAT_PARSE_EXPONENT_MARKER_BYTE_COUNT is one e, E, p, or P byte.
const RAT_PARSE_EXPONENT_MARKER_BYTE_COUNT = 1

// RAT_PARSE_MANTISSA_SIZE_MINIMUM keeps one actual digit before exponent syntax.
const RAT_PARSE_MANTISSA_SIZE_MINIMUM = SIGN_BYTE_COUNT_MAXIMUM

// RAT_PARSE_EXPONENT_TEXT_SIZE_MAXIMUM leaves one mantissa digit and exponent marker.
const RAT_PARSE_EXPONENT_TEXT_SIZE_MAXIMUM = RAT_PARSE_TEXT_SIZE_MAXIMUM -
	RAT_PARSE_MANTISSA_SIZE_MINIMUM - RAT_PARSE_EXPONENT_MARKER_BYTE_COUNT

// RAT_PARSE_FRACTIONAL_DIGIT_COUNT_MAXIMUM leaves one radix point in bounded source.
const RAT_PARSE_FRACTIONAL_DIGIT_COUNT_MAXIMUM = RAT_PARSE_TEXT_SIZE_MAXIMUM -
	DECIMAL_POINT_BYTE_COUNT

// RAT_PARSE_SEPARATOR_INDEX_ABSENT distinguishes no slash from first byte.
const RAT_PARSE_SEPARATOR_INDEX_ABSENT = -1

// RAT_TEXT_INTEGER_INDEX selects expanded denominator scratch.
const RAT_TEXT_INTEGER_INDEX = 0

// RAT_TEXT_INTEGER_COUNT is complete rational text integer state.
const RAT_TEXT_INTEGER_COUNT = RAT_TEXT_INTEGER_INDEX + 1

// RAT_TEXT_NUMERATOR_INDEX selects numerator conversion scratch.
const RAT_TEXT_NUMERATOR_INDEX = 0

// RAT_TEXT_DENOMINATOR_INDEX selects denominator conversion scratch.
const RAT_TEXT_DENOMINATOR_INDEX = RAT_TEXT_NUMERATOR_INDEX + 1

// RAT_TEXT_WORKSPACE_COUNT is complete component conversion state.
const RAT_TEXT_WORKSPACE_COUNT = RAT_TEXT_DENOMINATOR_INDEX + 1

// RAT_FLOAT_NUMERATOR_INDEX selects nonnegative numerator scratch.
const RAT_FLOAT_NUMERATOR_INDEX = 0

// RAT_FLOAT_DENOMINATOR_INDEX selects expanded positive denominator.
const RAT_FLOAT_DENOMINATOR_INDEX = RAT_FLOAT_NUMERATOR_INDEX + 1

// RAT_FLOAT_INTEGER_INDEX selects integer quotient.
const RAT_FLOAT_INTEGER_INDEX = RAT_FLOAT_DENOMINATOR_INDEX + 1

// RAT_FLOAT_REMAINDER_INDEX selects numerator remainder.
const RAT_FLOAT_REMAINDER_INDEX = RAT_FLOAT_INTEGER_INDEX + 1

// RAT_FLOAT_SCALE_INDEX selects ten raised to requested precision.
const RAT_FLOAT_SCALE_INDEX = RAT_FLOAT_REMAINDER_INDEX + 1

// RAT_FLOAT_SCALED_REMAINDER_INDEX selects remainder times decimal scale.
const RAT_FLOAT_SCALED_REMAINDER_INDEX = RAT_FLOAT_SCALE_INDEX + 1

// RAT_FLOAT_FRACTION_INDEX selects truncated scaled fractional digits.
const RAT_FLOAT_FRACTION_INDEX = RAT_FLOAT_SCALED_REMAINDER_INDEX + 1

// RAT_FLOAT_ROUNDING_REMAINDER_INDEX selects discarded scaled remainder.
const RAT_FLOAT_ROUNDING_REMAINDER_INDEX = RAT_FLOAT_FRACTION_INDEX + 1

// RAT_FLOAT_ROUNDING_DOUBLE_INDEX selects twice discarded remainder.
const RAT_FLOAT_ROUNDING_DOUBLE_INDEX = RAT_FLOAT_ROUNDING_REMAINDER_INDEX + 1

// RAT_FLOAT_UNIT_INDEX selects arithmetic carry one.
const RAT_FLOAT_UNIT_INDEX = RAT_FLOAT_ROUNDING_DOUBLE_INDEX + 1

// RAT_FLOAT_BASE_INDEX selects decimal exponent base.
const RAT_FLOAT_BASE_INDEX = RAT_FLOAT_UNIT_INDEX + 1

// RAT_FLOAT_PRECISION_INDEX selects lifted decimal exponent.
const RAT_FLOAT_PRECISION_INDEX = RAT_FLOAT_BASE_INDEX + 1

// RAT_FLOAT_INTEGER_COUNT is complete fixed-decimal arithmetic state.
const RAT_FLOAT_INTEGER_COUNT = RAT_FLOAT_PRECISION_INDEX + 1

// RAT_FLOAT_TEXT_INTEGER_PART_INDEX selects integer reversed digits.
const RAT_FLOAT_TEXT_INTEGER_PART_INDEX = 0

// RAT_FLOAT_TEXT_FRACTIONAL_PART_INDEX selects fractional reversed digits.
const RAT_FLOAT_TEXT_FRACTIONAL_PART_INDEX = RAT_FLOAT_TEXT_INTEGER_PART_INDEX + 1

// RAT_FLOAT_TEXT_WORKSPACE_COUNT is complete fixed-decimal conversion state.
const RAT_FLOAT_TEXT_WORKSPACE_COUNT = RAT_FLOAT_TEXT_FRACTIONAL_PART_INDEX + 1

// RAT_FLOAT_PRECISION_DENOMINATOR_INDEX selects reduced odd denominator.
const RAT_FLOAT_PRECISION_DENOMINATOR_INDEX = 0

// RAT_FLOAT_PRECISION_INTEGER_COUNT is complete decimal precision analysis state.
const RAT_FLOAT_PRECISION_INTEGER_COUNT = RAT_FLOAT_PRECISION_DENOMINATOR_INDEX + 1

// RAT_FLOAT_64_NUMERATOR_INDEX selects scaled nonnegative numerator.
const RAT_FLOAT_64_NUMERATOR_INDEX = 0

// RAT_FLOAT_64_DENOMINATOR_INDEX selects scaled positive denominator.
const RAT_FLOAT_64_DENOMINATOR_INDEX = RAT_FLOAT_64_NUMERATOR_INDEX + 1

// RAT_FLOAT_64_QUOTIENT_INDEX selects rounding-width quotient.
const RAT_FLOAT_64_QUOTIENT_INDEX = RAT_FLOAT_64_DENOMINATOR_INDEX + 1

// RAT_FLOAT_64_REMAINDER_INDEX selects discarded division remainder.
const RAT_FLOAT_64_REMAINDER_INDEX = RAT_FLOAT_64_QUOTIENT_INDEX + 1

// RAT_FLOAT_64_INTEGER_COUNT is complete binary64 conversion arithmetic state.
const RAT_FLOAT_64_INTEGER_COUNT = RAT_FLOAT_64_REMAINDER_INDEX + 1

// RAT_PARSE_FRACTION_NUMERATOR_INDEX selects parsed signed numerator.
const RAT_PARSE_FRACTION_NUMERATOR_INDEX = 0

// RAT_PARSE_FRACTION_DENOMINATOR_INDEX selects parsed positive denominator.
const RAT_PARSE_FRACTION_DENOMINATOR_INDEX = RAT_PARSE_FRACTION_NUMERATOR_INDEX + 1

// RAT_PARSE_FRACTION_INTEGER_COUNT is complete fraction parse value state.
const RAT_PARSE_FRACTION_INTEGER_COUNT = RAT_PARSE_FRACTION_DENOMINATOR_INDEX + 1

// RAT_PARSE_NUMERATOR_WORKSPACE_INDEX selects numerator parse scratch.
const RAT_PARSE_NUMERATOR_WORKSPACE_INDEX = 0

// RAT_PARSE_DENOMINATOR_WORKSPACE_INDEX selects denominator parse scratch.
const RAT_PARSE_DENOMINATOR_WORKSPACE_INDEX = RAT_PARSE_NUMERATOR_WORKSPACE_INDEX + 1

// RAT_PARSE_WORKSPACE_COUNT is complete component parse scratch state.
const RAT_PARSE_WORKSPACE_COUNT = RAT_PARSE_DENOMINATOR_WORKSPACE_INDEX + 1

// RAT_PARSE_EXPONENT_2_INDEX selects complete binary scaling.
const RAT_PARSE_EXPONENT_2_INDEX = 0

// RAT_PARSE_EXPONENT_5_INDEX selects odd decimal-prime scaling.
const RAT_PARSE_EXPONENT_5_INDEX = RAT_PARSE_EXPONENT_2_INDEX + 1

// RAT_PARSE_EXPONENT_COUNT is complete split scaling state.
const RAT_PARSE_EXPONENT_COUNT = RAT_PARSE_EXPONENT_5_INDEX + 1

// RAT_TEXT_FORM_FRACTION always emits explicit denominator.
const RAT_TEXT_FORM_FRACTION Rat_Text_Form = 0

// RAT_TEXT_FORM_RATIONAL omits denominator one.
const RAT_TEXT_FORM_RATIONAL Rat_Text_Form = RAT_TEXT_FORM_FRACTION + 1

// RAT_RESULT_NUMERATOR_INDEX selects normalized numerator scratch.
const RAT_RESULT_NUMERATOR_INDEX = 0

// RAT_RESULT_DENOMINATOR_INDEX selects normalized denominator scratch.
const RAT_RESULT_DENOMINATOR_INDEX = RAT_RESULT_NUMERATOR_INDEX + 1

// RAT_COMMON_DIVISOR_INDEX selects normalization divisor or second scaled numerator scratch.
const RAT_COMMON_DIVISOR_INDEX = RAT_RESULT_DENOMINATOR_INDEX + 1

// RAT_LEFT_NUMERATOR_INDEX selects left numerator scratch.
const RAT_LEFT_NUMERATOR_INDEX = RAT_COMMON_DIVISOR_INDEX + 1

// RAT_LEFT_DENOMINATOR_INDEX selects left denominator scratch.
const RAT_LEFT_DENOMINATOR_INDEX = RAT_LEFT_NUMERATOR_INDEX + 1

// RAT_RIGHT_NUMERATOR_INDEX selects right numerator scratch.
const RAT_RIGHT_NUMERATOR_INDEX = RAT_LEFT_DENOMINATOR_INDEX + 1

// RAT_RIGHT_DENOMINATOR_INDEX selects right denominator scratch.
const RAT_RIGHT_DENOMINATOR_INDEX = RAT_RIGHT_NUMERATOR_INDEX + 1

// RAT_OPERATION_INTEGER_COUNT is complete rational arithmetic scratch.
const RAT_OPERATION_INTEGER_COUNT = RAT_RIGHT_DENOMINATOR_INDEX + 1

// INT_64_NEGATIVE_MAGNITUDE_MAXIMUM includes minimum int64 magnitude without signed overflow.
const INT_64_NEGATIVE_MAGNITUDE_MAXIMUM = uint64(bits.INTEGER_64_MAXIMUM) + 1

// Validation_Status excludes range failure from operations that never narrow or grow values.
type Validation_Status uint8

// Validation_Status_Invariants keeps pointer and input validation outcomes exact.
func Validation_Status_Invariants(value Validation_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID)).
		Ensure()
}

// Conversion_Status includes narrowing failure without arithmetic overflow semantics.
type Conversion_Status uint8

// Conversion_Status_Invariants keeps machine conversion outcomes exact.
func Conversion_Status_Invariants(value Conversion_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_VALUE_OVERFLOW)).
		Ensure()
}

// Arithmetic_Status separates mathematical overflow from malformed object input.
type Arithmetic_Status uint8

// Arithmetic_Status_Invariants keeps bounded arithmetic outcomes exact.
func Arithmetic_Status_Invariants(value Arithmetic_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_VALUE_OVERFLOW)).
		Ensure()
}

// Parse_Status distinguishes malformed text from bounded magnitude overflow.
type Parse_Status uint8

// Parse_Status_Invariants admits every transactional parse outcome.
func Parse_Status_Invariants(value Parse_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_VALUE_OVERFLOW),
		).
		Ensure()
}

// Rat_Arithmetic_Status reports exact results or bounded component overflow.
type Rat_Arithmetic_Status uint8

// Rat_Arithmetic_Status_Invariants excludes zero-divisor failure from closed arithmetic.
func Rat_Arithmetic_Status_Invariants(
	value Rat_Arithmetic_Status, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_VALUE_OVERFLOW)).
		Ensure()
}

// Rat_Division_Status includes zero denominator beside component overflow.
type Rat_Division_Status uint8

// Rat_Division_Status_Invariants admits every fraction and quotient outcome.
func Rat_Division_Status_Invariants(value Rat_Division_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_VALUE_OVERFLOW),
			uint8(STATUS_DIVISOR_ZERO),
		).
		Ensure()
}

// Destination_Status distinguishes exact output from insufficient caller storage.
type Destination_Status uint8

// Destination_Status_Invariants keeps bounded-output outcomes exact.
func Destination_Status_Invariants(value Destination_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_DESTINATION_TOO_SMALL)).
		Ensure()
}

// Division_Status separates undefined input and output-layout failures.
type Division_Status uint8

// Division_Status_Invariants keeps division outcomes exact.
func Division_Status_Invariants(value Division_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_DIVISOR_ZERO),
			uint8(STATUS_DESTINATIONS_OVERLAP),
		).
		Ensure()
}

// Divisor_Status reports whether one single-result division had a divisor.
type Divisor_Status uint8

// Divisor_Status_Invariants excludes output-overlap failure from single-result operations.
func Divisor_Status_Invariants(value Divisor_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_DIVISOR_ZERO)).
		Ensure()
}

// Modular_Status separates zero modulus from a mathematically absent inverse.
type Modular_Status uint8

// Modular_Status_Invariants admits every bounded modular result.
func Modular_Status_Invariants(value Modular_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_DIVISOR_ZERO),
			uint8(STATUS_RESULT_ABSENT),
		).
		Ensure()
}

// Random_Status separates malformed entropy bounds from a consumed rejected batch.
type Random_Status uint8

// Random_Status_Invariants keeps bounded random outcomes exact.
func Random_Status_Invariants(value Random_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_SOURCE_EXHAUSTED),
		).
		Ensure()
}

// Primality_Status separates malformed limits from entropy and parameter exhaustion.
type Primality_Status uint8

// Primality_Status_Invariants admits every bounded probable-prime outcome.
func Primality_Status_Invariants(value Primality_Status, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(STATUS_OK), uint8(STATUS_INPUT_INVALID),
			uint8(STATUS_SOURCE_EXHAUSTED), uint8(STATUS_SEARCH_EXHAUSTED),
		).
		Ensure()
}

// Primality_Entropy_Status excludes validation failure after public bounds pass.
type Primality_Entropy_Status uint8

// Primality_Entropy_Status_Invariants admits success or exhausted caller entropy.
func Primality_Entropy_Status_Invariants(
	value Primality_Entropy_Status, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_SOURCE_EXHAUSTED)).
		Ensure()
}

// Primality_Search_Status excludes input failure after parameter count validation.
type Primality_Search_Status uint8

// Primality_Search_Status_Invariants admits success or exhausted method-C search.
func Primality_Search_Status_Invariants(
	value Primality_Search_Status, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(STATUS_OK), uint8(STATUS_SEARCH_EXHAUSTED)).
		Ensure()
}

// Boolean gives fit decisions separate coverage identity.
type Boolean bool

// Boolean_Invariants requires both fit outcomes.
func Boolean_Invariants(value Boolean, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "A big-number decision is true.").
		Ensure()
}

// Accuracy orders a rounded result against its exact mathematical source.
type Accuracy int8

// Accuracy_Invariants admits every stdlib rounding relation.
func Accuracy_Invariants(value Accuracy, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Int8(
			int8(value), int8(ACCURACY_BELOW), int8(ACCURACY_EXACT),
			int8(ACCURACY_ABOVE),
		).
		Ensure()
}

// Rounding_Mode_Unvalidated admits one hostile enum value before validation.
type Rounding_Mode_Unvalidated uint8

// Rounding_Mode_Unvalidated_Invariants bounds mode validation to one scalar decision.
func Rounding_Mode_Unvalidated_Invariants(
	value Rounding_Mode_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(ROUND_TO_NEAREST_EVEN),
			uint8(ROUNDING_MODE_UNVALIDATED_MAXIMUM),
		).
		Ensure()
}

// Rounding_Mode stores one validated finite-result direction.
type Rounding_Mode uint8

// Rounding_Mode_Invariants admits the six contiguous stdlib modes.
func Rounding_Mode_Invariants(value Rounding_Mode, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(
			uint8(value), uint8(ROUND_TO_NEAREST_EVEN),
			uint8(ROUND_TO_POSITIVE_INFINITY),
		).
		Ensure()
}

// Float_Form separates signed zero, finite values, and signed infinity.
type Float_Form uint8

// Float_Form_Invariants admits every non-NaN storage form.
func Float_Form_Invariants(value Float_Form, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(FLOAT_FORM_ZERO), uint8(FLOAT_FORM_FINITE),
			uint8(FLOAT_FORM_INFINITY),
		).
		Ensure()
}

// Primality_Trial_Result distinguishes exact early answers from probable-prime work.
type Primality_Trial_Result uint8

// Primality_Trial_Result_Invariants admits complete trial-division state.
func Primality_Trial_Result_Invariants(
	value Primality_Trial_Result, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(PRIMALITY_TRIAL_COMPOSITE),
			uint8(PRIMALITY_TRIAL_PRIME), uint8(PRIMALITY_TRIAL_UNDETERMINED),
		).
		Ensure()
}

// Lucas_Update selects one recurrence product without passing mutable integer indexes.
type Lucas_Update uint8

// Lucas_Update_Invariants admits every recurrence product shape.
func Lucas_Update_Invariants(value Lucas_Update, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(LUCAS_UPDATE_PRODUCT),
			uint8(LUCAS_UPDATE_CURRENT_SQUARE), uint8(LUCAS_UPDATE_NEXT_SQUARE),
		).
		Ensure()
}

// Polarity stores whether one nonzero magnitude is negative.
type Polarity uint8

// Polarity_Invariants excludes arbitrary sign states from Int storage.
func Polarity_Invariants(value Polarity, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(POLARITY_NONNEGATIVE), uint8(POLARITY_NEGATIVE)).
		Ensure()
}

// Bitwise_Operation selects one fixed binary truth table.
type Bitwise_Operation uint8

// Bitwise_Operation_Invariants admits every supported binary operation.
func Bitwise_Operation_Invariants(value Bitwise_Operation, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(BITWISE_OPERATION_AND),
			uint8(BITWISE_OPERATION_AND_NOT), uint8(BITWISE_OPERATION_OR),
			uint8(BITWISE_OPERATION_XOR),
		).
		Ensure()
}

// Modular_Reduction_Operation selects one owned reduction destination.
type Modular_Reduction_Operation uint8

// Modular_Reduction_Operation_Invariants admits both reduction scratch roles.
func Modular_Reduction_Operation_Invariants(
	value Modular_Reduction_Operation, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(MODULAR_REDUCTION_OPERAND),
			uint8(MODULAR_REDUCTION_FACTOR),
		).
		Ensure()
}

// Modular_Multiplication_Operation selects one modular product role.
type Modular_Multiplication_Operation uint8

// Modular_Multiplication_Operation_Invariants admits every modular product role.
func Modular_Multiplication_Operation_Invariants(
	value Modular_Multiplication_Operation, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(MODULAR_MULTIPLICATION_ACCUMULATE),
			uint8(MODULAR_MULTIPLICATION_SQUARE),
			uint8(MODULAR_MULTIPLICATION_COEFFICIENT),
		).
		Ensure()
}

// Modular_Addition_Operation selects accumulation or factor doubling.
type Modular_Addition_Operation uint8

// Modular_Addition_Operation_Invariants admits both modular sum roles.
func Modular_Addition_Operation_Invariants(
	value Modular_Addition_Operation, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(MODULAR_ADDITION_ACCUMULATE),
			uint8(MODULAR_ADDITION_DOUBLE),
		).
		Ensure()
}

// Rat_Operation selects one fixed binary rational algorithm.
type Rat_Operation uint8

// Rat_Operation_Invariants admits complete rational arithmetic surface.
func Rat_Operation_Invariants(value Rat_Operation, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(RAT_OPERATION_ADD), uint8(RAT_OPERATION_SUBTRACT),
			uint8(RAT_OPERATION_MULTIPLY), uint8(RAT_OPERATION_QUOTIENT),
		).
		Ensure()
}

// Rat_Text_Form selects explicit or denominator-eliding exact text.
type Rat_Text_Form uint8

// Rat_Text_Form_Invariants admits both stdlib rational string forms.
func Rat_Text_Form_Invariants(value Rat_Text_Form, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(
			uint8(value), uint8(RAT_TEXT_FORM_FRACTION), uint8(RAT_TEXT_FORM_RATIONAL),
		).
		Ensure()
}

// Sign separates value polarity from comparison outcomes.
type Sign int8

// Sign_Invariants admits only three mathematical signs.
func Sign_Invariants(value Sign, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Int8(int8(value), int8(SIGN_NEGATIVE), int8(SIGN_ZERO), int8(SIGN_POSITIVE)).
		Ensure()
}

// Jacobi_Symbol keeps quadratic residue result separate from general sign values.
type Jacobi_Symbol int8

// Jacobi_Symbol_Invariants admits exactly three mathematical Jacobi outcomes.
func Jacobi_Symbol_Invariants(value Jacobi_Symbol, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Int8(
			int8(value), int8(JACOBI_SYMBOL_NEGATIVE), int8(JACOBI_SYMBOL_ZERO),
			int8(JACOBI_SYMBOL_POSITIVE),
		).
		Ensure()
}

// Order names relative position without accepting arbitrary integers.
type Order int8

// Order_Invariants admits only three comparison outcomes.
func Order_Invariants(value Order, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Int8(int8(value), int8(ORDER_BEFORE), int8(ORDER_SAME), int8(ORDER_AFTER)).
		Ensure()
}

// Base_Unvalidated admits malformed conversion bases before validation.
type Base_Unvalidated int

// Base_Unvalidated_Invariants bounds hostile base validation to one scalar decision.
func Base_Unvalidated_Invariants(value Base_Unvalidated, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), BASE_UNVALIDATED_MINIMUM, BASE_UNVALIDATED_MAXIMUM).
		Ensure()
}

// Base is one validated positional conversion alphabet size.
type Base int

// Base_Invariants binds text conversion to standard-library base range.
func Base_Invariants(value Base, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), BASE_MINIMUM, BASE_MAXIMUM).
		Ensure()
}

// Bit_Count keeps bit widths from entering word-count arithmetic unnoticed.
type Bit_Count int

// Bit_Count_Invariants binds widths to one Int.
func Bit_Count_Invariants(value Bit_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), BIT_COUNT_MINIMUM, BIT_COUNT_MAXIMUM).
		Ensure()
}

// Float_Precision_Unvalidated admits complete machine input before bounded validation.
type Float_Precision_Unvalidated uint

// Float_Precision_Unvalidated_Invariants keeps precision validation constant work.
func Float_Precision_Unvalidated_Invariants(
	value Float_Precision_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint(uint(value), bits.WORD_MINIMUM, bits.WORD_MAXIMUM).
		Ensure()
}

// Float_Precision bounds one mantissa's significant bits.
type Float_Precision uint

// Float_Precision_Invariants binds precision to inline mantissa storage.
func Float_Precision_Invariants(value Float_Precision, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint(
			uint(value), FLOAT_PRECISION_MINIMUM, FLOAT_PRECISION_MAXIMUM,
		).
		Ensure()
}

// Float_Exponent_Unvalidated admits one hostile normalized exponent on each boundary.
type Float_Exponent_Unvalidated int

// Float_Exponent_Unvalidated_Invariants bounds exponent validation before arithmetic.
func Float_Exponent_Unvalidated_Invariants(
	value Float_Exponent_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), FLOAT_EXPONENT_UNVALIDATED_MINIMUM,
			FLOAT_EXPONENT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Float_Exponent is one normalized finite binary exponent.
type Float_Exponent int

// Float_Exponent_Invariants keeps finite scale inside explicit bounds.
func Float_Exponent_Invariants(value Float_Exponent, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), FLOAT_EXPONENT_MINIMUM, FLOAT_EXPONENT_MAXIMUM).
		Ensure()
}

// Float_Active_Precision excludes zero after a numeric setter establishes a mantissa policy.
type Float_Active_Precision uint

// Float_Active_Precision_Invariants binds finite work to positive inline precision.
func Float_Active_Precision_Invariants(
	value Float_Active_Precision, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint(uint(value), WORD_COUNT_INCREMENT, FLOAT_PRECISION_MAXIMUM).
		Ensure()
}

// Float_Rounding_Source_Precision excludes values that cannot lose one retained bit.
type Float_Rounding_Source_Precision uint

// Float_Rounding_Source_Precision_Invariants binds precision before one reduction.
func Float_Rounding_Source_Precision_Invariants(
	value Float_Rounding_Source_Precision, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint(uint(value), BASE_BINARY, FLOAT_PRECISION_MAXIMUM).
		Ensure()
}

// Float_Reduced_Precision is one rounding target below complete mantissa width.
type Float_Reduced_Precision uint

// Float_Reduced_Precision_Invariants excludes zero and impossible full-width reduction.
func Float_Reduced_Precision_Invariants(
	value Float_Reduced_Precision, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint(uint(value), WORD_COUNT_INCREMENT, BIT_INDEX_MAXIMUM).
		Ensure()
}

// Float_Discarded_Bit_Count counts low inspected bits below one retained mantissa bit.
type Float_Discarded_Bit_Count int

// Float_Discarded_Bit_Count_Invariants excludes a complete discarded finite mantissa.
func Float_Discarded_Bit_Count_Invariants(
	value Float_Discarded_Bit_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), BIT_COUNT_MINIMUM, FLOAT_DISCARDED_BIT_COUNT_MAXIMUM,
		).
		Ensure()
}

// Float_Addition_Shift aligns one finite mantissa above the lower operand bit.
type Float_Addition_Shift int

// Float_Addition_Shift_Invariants bounds exact alignment inside caller workspace.
func Float_Addition_Shift_Invariants(
	value Float_Addition_Shift, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), BIT_COUNT_MINIMUM, FLOAT_ADDITION_SHIFT_MAXIMUM,
		).
		Ensure()
}

// Float_Subtraction_Shift aligns the smaller magnitude below one bounded larger mantissa.
type Float_Subtraction_Shift int

// Float_Subtraction_Shift_Invariants uses magnitude ordering to exclude wider separation.
func Float_Subtraction_Shift_Invariants(
	value Float_Subtraction_Shift, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), BIT_COUNT_MINIMUM, BIT_INDEX_MAXIMUM).
		Ensure()
}

// Float_Addition_Bit_Count is one nonzero normalized exact workspace width.
type Float_Addition_Bit_Count int

// Float_Addition_Bit_Count_Invariants binds normalization to the complete alignment span.
func Float_Addition_Bit_Count_Invariants(
	value Float_Addition_Bit_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), WORD_COUNT_INCREMENT, FLOAT_ADDITION_BIT_COUNT_MAXIMUM,
		).
		Ensure()
}

// Float_Addition_Rounding_Index selects a discarded bit below one retained bit.
type Float_Addition_Rounding_Index int

// Float_Addition_Rounding_Index_Invariants bounds sticky inspection below the rounding bit.
func Float_Addition_Rounding_Index_Invariants(
	value Float_Addition_Rounding_Index, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), BIT_COUNT_MINIMUM, FLOAT_ADDITION_ROUNDING_INDEX_MAXIMUM,
		).
		Ensure()
}

// Float_Addition_Active_Discard_Count excludes exact-width addition results.
type Float_Addition_Active_Discard_Count int

// Float_Addition_Active_Discard_Count_Invariants binds one actual precision reduction.
func Float_Addition_Active_Discard_Count_Invariants(
	value Float_Addition_Active_Discard_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), WORD_COUNT_INCREMENT, FLOAT_ADDITION_SHIFT_MAXIMUM,
		).
		Ensure()
}

// Float_Result_Origin is the binary exponent represented by arithmetic workspace bit zero.
type Float_Result_Origin int

// Float_Result_Origin_Invariants covers addition and multiplication least-significant bits.
func Float_Result_Origin_Invariants(
	value Float_Result_Origin, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), FLOAT_RESULT_ORIGIN_MINIMUM,
			FLOAT_RESULT_ORIGIN_MAXIMUM,
		).
		Ensure()
}

// Float_Addition_Active_Word_Count excludes exact cancellation after it is handled.
type Float_Addition_Active_Word_Count int

// Float_Addition_Active_Word_Count_Invariants bounds normalized workspace output.
func Float_Addition_Active_Word_Count_Invariants(
	value Float_Addition_Active_Word_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), WORD_COUNT_INCREMENT, FLOAT_ADDITION_WORD_COUNT_MAXIMUM,
		).
		Ensure()
}

// Float_Division_Active_Word_Count retains one aligned divisor and one carry word.
type Float_Division_Active_Word_Count int

// Float_Division_Active_Word_Count_Invariants bounds normalized quotient work.
func Float_Division_Active_Word_Count_Invariants(
	value Float_Division_Active_Word_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), FLOAT_DIVISION_WORD_COUNT_MINIMUM,
			FLOAT_DIVISION_WORD_COUNT_MAXIMUM,
		).
		Ensure()
}

// Float_Division_Aligned_Bit_Count is the common width of two active significands.
type Float_Division_Aligned_Bit_Count int

// Float_Division_Aligned_Bit_Count_Invariants excludes zero and excess source widths.
func Float_Division_Aligned_Bit_Count_Invariants(
	value Float_Division_Aligned_Bit_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), WORD_COUNT_INCREMENT, FLOAT_PRECISION_MAXIMUM,
		).
		Ensure()
}

// Float_Unequal_Order excludes equality after the exact quotient path handles it.
type Float_Unequal_Order Order

// Float_Unequal_Order_Invariants admits either strict significand ordering.
func Float_Unequal_Order_Invariants(value Float_Unequal_Order, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Int8(int8(value), int8(ORDER_BEFORE), int8(ORDER_AFTER)).
		Ensure()
}

// Float_Division_Word_Control keeps derived counts outside semantic domains.
type Float_Division_Word_Control [FLOAT_DIVISION_CONTROL_COUNT]int

// Float_Division_Word_Control_Invariants fixes complete short-division control storage.
func Float_Division_Word_Control_Invariants(
	value Float_Division_Word_Control, _ invariant.Namespace,
) {
	invariant.Always(
		len(value) == FLOAT_DIVISION_CONTROL_COUNT,
		"Float word division owns both derived word counts.",
	)
}

// Float_Division_Quotient_Words gives destination storage one structural identity.
type Float_Division_Quotient_Words [WORD_COUNT_MAXIMUM]Word

// Float_Division_Quotient_Words_Invariants keeps a complete bounded quotient span.
func Float_Division_Quotient_Words_Invariants(
	value *Float_Division_Quotient_Words, _ invariant.Namespace,
) {
	invariant.Always(
		len(value) == WORD_COUNT_MAXIMUM,
		"Float word division retains every bounded quotient word.",
	)
}

// Inexact_Accuracy excludes exact from one known discarded-result relation.
type Inexact_Accuracy int8

// Inexact_Accuracy_Invariants admits both directions around an exact value.
func Inexact_Accuracy_Invariants(value Inexact_Accuracy, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Int8(int8(value), int8(ACCURACY_BELOW), int8(ACCURACY_ABOVE)).
		Ensure()
}

// Float_Nonfinite_Form excludes finite from zero and infinity construction.
type Float_Nonfinite_Form uint8

// Float_Nonfinite_Form_Invariants admits exactly signed zero and signed infinity.
func Float_Nonfinite_Form_Invariants(
	value Float_Nonfinite_Form, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(FLOAT_FORM_ZERO), uint8(FLOAT_FORM_INFINITY)).
		Ensure()
}

// Byte_Count states how much caller storage one magnitude populated.
type Byte_Count int

// Byte_Count_Invariants binds encoded magnitude size to repository byte storage.
func Byte_Count_Invariants(value Byte_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, bytes.SLICE_SIZE_MAXIMUM).
		Ensure()
}

// Int_Encoding_Count states bytes populated by one gob representation.
type Int_Encoding_Count int

// Int_Encoding_Count_Invariants includes zero because failed writes populate nothing.
func Int_Encoding_Count_Invariants(value Int_Encoding_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, INT_GOB_SIZE_MAXIMUM).
		Ensure()
}

// Text_Count states exact bytes populated by one integer representation.
type Text_Count int

// Text_Count_Invariants binds signed binary worst case to caller text storage.
func Text_Count_Invariants(value Text_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), bytes.SLICE_SIZE_MINIMUM, INT_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Rat_Text_Count states exact bytes populated by one rational representation.
type Rat_Text_Count int

// Rat_Text_Count_Invariants keeps rational text inside derived component storage.
func Rat_Text_Count_Invariants(value Rat_Text_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), RAT_TEXT_SIZE_MINIMUM, RAT_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Rat_Precision_Unvalidated admits one hostile value beyond each decimal precision bound.
type Rat_Precision_Unvalidated int

// Rat_Precision_Unvalidated_Invariants bounds precision validation to one scalar decision.
func Rat_Precision_Unvalidated_Invariants(
	value Rat_Precision_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), RAT_PRECISION_UNVALIDATED_MINIMUM,
			RAT_PRECISION_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Rat_Precision is one validated fixed-decimal fractional digit count.
type Rat_Precision int

// Rat_Precision_Invariants keeps decimal scaling inside full Int storage.
func Rat_Precision_Invariants(value Rat_Precision, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), RAT_PRECISION_MINIMUM, RAT_PRECISION_MAXIMUM).
		Ensure()
}

// Decimal_Place_Count reports finite prefix length after decimal point.
type Decimal_Place_Count int

// Decimal_Place_Count_Invariants binds precision analysis to denominator width.
func Decimal_Place_Count_Invariants(
	value Decimal_Place_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), RAT_FLOAT_PRECISION_COUNT_MINIMUM,
			RAT_FLOAT_PRECISION_COUNT_MAXIMUM,
		).
		Ensure()
}

// Decimal_Factor_Remainder is one remainder after division by decimal odd prime factor.
type Decimal_Factor_Remainder Word

// Decimal_Factor_Remainder_Invariants excludes values at or above divisor.
func Decimal_Factor_Remainder_Invariants(
	value Decimal_Factor_Remainder, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint64(
			uint64(value), bits.WORD_64_MINIMUM, RAT_DECIMAL_FACTOR_REMAINDER_MAXIMUM,
		).
		Ensure()
}

// Rat_Component_Digit_Count bounds one fixed-decimal component representation.
type Rat_Component_Digit_Count int

// Rat_Component_Digit_Count_Invariants binds digits to rational component width.
func Rat_Component_Digit_Count_Invariants(
	value Rat_Component_Digit_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), RAT_COMPONENT_DIGIT_COUNT_MINIMUM,
			RAT_COMPONENT_DECIMAL_DIGIT_COUNT_MAXIMUM,
		).
		Ensure()
}

// Rat_Float_Text_Destination is exact populated fixed-decimal caller storage.
type Rat_Float_Text_Destination []byte

// Rat_Float_Text_Destination_Invariants binds writes to derived rational text bounds.
func Rat_Float_Text_Destination_Invariants(
	value Rat_Float_Text_Destination, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), RAT_FLOAT_TEXT_DESTINATION_SIZE_MINIMUM,
			RAT_FLOAT_TEXT_DESTINATION_SIZE_MAXIMUM,
		).
		Ensure()
}

// Rat_Fraction_Text_Count states bytes populated by explicit fraction representation.
type Rat_Fraction_Text_Count int

// Rat_Fraction_Text_Count_Invariants excludes denominator-elided widths.
func Rat_Fraction_Text_Count_Invariants(
	value Rat_Fraction_Text_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), RAT_FRACTION_TEXT_SIZE_MINIMUM, RAT_TEXT_SIZE_MAXIMUM,
		).
		Ensure()
}

// Text_Digit_Count excludes optional sign from one magnitude representation.
type Text_Digit_Count int

// Text_Digit_Count_Invariants binds zero's one digit through full binary width.
func Text_Digit_Count_Invariants(value Text_Digit_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), SIGN_BYTE_COUNT_MAXIMUM, BIT_COUNT_MAXIMUM).
		Ensure()
}

// Trailing_Zero_Bit_Count states one bounded magnitude's low zero-bit run.
type Trailing_Zero_Bit_Count int

// Trailing_Zero_Bit_Count_Invariants excludes impossible full-width nonzero runs.
func Trailing_Zero_Bit_Count_Invariants(
	value Trailing_Zero_Bit_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), BIT_COUNT_MINIMUM, TRAILING_ZERO_BIT_COUNT_MAXIMUM).
		Ensure()
}

// Shift_Count_Unvalidated admits complete machine input before bounded-work validation.
type Shift_Count_Unvalidated uint

// Shift_Count_Unvalidated_Invariants keeps validation itself constant work.
func Shift_Count_Unvalidated_Invariants(
	value Shift_Count_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint(uint(value), bits.WORD_MINIMUM, bits.WORD_MAXIMUM).
		Ensure()
}

// Shift_Count is one validated distance across bounded Int storage.
type Shift_Count uint

// Shift_Count_Invariants binds shifts to the complete stored bit width.
func Shift_Count_Invariants(value Shift_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint(uint(value), BIT_COUNT_MINIMUM, BIT_COUNT_MAXIMUM).
		Ensure()
}

// Bit_Index_Unvalidated admits one invalid coordinate on each side of bounded storage.
type Bit_Index_Unvalidated int

// Bit_Index_Unvalidated_Invariants keeps hostile index validation constant work.
func Bit_Index_Unvalidated_Invariants(
	value Bit_Index_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), BIT_INDEX_UNVALIDATED_MINIMUM, BIT_INDEX_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Bit_Index selects one stored signed bit.
type Bit_Index int

// Bit_Index_Invariants admits each coordinate below bounded width.
func Bit_Index_Invariants(value Bit_Index, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), BIT_COUNT_MINIMUM, BIT_INDEX_MAXIMUM).
		Ensure()
}

// Bit_Value is one validated binary digit.
type Bit_Value uint8

// Bit_Value_Invariants excludes values outside one bit.
func Bit_Value_Invariants(value Bit_Value, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(BIT_CLEAR), uint8(BIT_SET)).
		Ensure()
}

// Float_64_Bits keeps binary64 encoding distinct from exact rational storage.
type Float_64_Bits uint64

// Float_64_Bits_Invariants covers every binary64 encoding, including nonfinite values.
func Float_64_Bits_Invariants(value Float_64_Bits, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(
			uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM,
		).
		Ensure()
}

// Float_64_Finite_Exponent_Field excludes the IEEE field reserved for infinity and NaN.
type Float_64_Finite_Exponent_Field uint64

// Float_64_Finite_Exponent_Field_Invariants binds decoded finite exponent bits.
func Float_64_Finite_Exponent_Field_Invariants(
	value Float_64_Finite_Exponent_Field, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint64(
			uint64(value), bits.WORD_64_MINIMUM,
			FLOAT_64_FINITE_EXPONENT_FIELD_MAXIMUM,
		).
		Ensure()
}

// Float_64_Mantissa_Field is one decoded IEEE binary64 stored significand.
type Float_64_Mantissa_Field uint64

// Float_64_Mantissa_Field_Invariants binds decoded significand bits to their field mask.
func Float_64_Mantissa_Field_Invariants(
	value Float_64_Mantissa_Field, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, FLOAT_64_MANTISSA_MASK).
		Ensure()
}

// Float_64_Sign is either cleared or the encoded IEEE binary64 sign bit.
type Float_64_Sign uint64

// Float_64_Sign_Invariants admits both binary64 sign encodings.
func Float_64_Sign_Invariants(value Float_64_Sign, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_Uint64(uint64(value), bits.WORD_64_MINIMUM, FLOAT_64_SIGN_MASK).
		Ensure()
}

// Float_64_Value_Bits is one numeric binary64 encoding, including signed infinities.
type Float_64_Value_Bits uint64

// Float_64_Value_Bits_Invariants rejects NaN encodings from rational conversion output.
func Float_64_Value_Bits_Invariants(
	value Float_64_Value_Bits, namespace invariant.Namespace,
) {
	encoded := uint64(value)
	invariant.Tree(value, namespace).
		Range_Uint64(
			uint64(value), FLOAT_64_VALUE_BITS_MINIMUM, FLOAT_64_VALUE_BITS_MAXIMUM,
		).
		Ensure()
	exponent := encoded >> FLOAT_64_EXPONENT_SHIFT & FLOAT_64_EXPONENT_MASK
	mantissa := encoded & FLOAT_64_MANTISSA_MASK
	exponent_difference := exponent ^ FLOAT_64_EXPONENT_MASK
	exponent_difference_nonzero := (exponent_difference | -exponent_difference) >>
		WORD_BIT_INDEX_MAXIMUM
	exponent_is_nonfinite := uint64(bits.CARRY_MAXIMUM) - exponent_difference_nonzero
	mantissa_nonzero := (mantissa | -mantissa) >> WORD_BIT_INDEX_MAXIMUM
	not_a_number := exponent_is_nonfinite * mantissa_nonzero
	invariant.Always(
		not_a_number == bits.WORD_64_MINIMUM,
		"Rational binary64 output never encodes NaN.",
	)
}

// Rat_Float_64_Mantissa is rounded significand before encoded hidden-bit removal.
type Rat_Float_64_Mantissa uint64

// Rat_Float_64_Mantissa_Invariants bounds final binary64 significand.
func Rat_Float_64_Mantissa_Invariants(
	value Rat_Float_64_Mantissa, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint64(
			uint64(value), bits.WORD_64_MINIMUM, FLOAT_64_VALUE_MANTISSA_MAXIMUM,
		).
		Ensure()
}

// Rat_Float_64_Exponent is normalized binary exponent before field encoding.
type Rat_Float_64_Exponent int

// Rat_Float_64_Exponent_Invariants follows complete bounded rational ratio range.
func Rat_Float_64_Exponent_Invariants(
	value Rat_Float_64_Exponent, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), RAT_FLOAT_64_EXPONENT_MINIMUM, RAT_FLOAT_64_EXPONENT_MAXIMUM,
		).
		Ensure()
}

// Int_64 retains conversion identity at public boundaries.
type Int_64 int64

// Int_64_Invariants covers complete signed machine input.
func Int_64_Invariants(value Int_64, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Word_64 retains unsigned conversion identity at public boundaries.
type Word_64 uint64

// Word_64_Invariants covers complete unsigned machine input.
func Word_64_Invariants(value Word_64, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Word keeps magnitude storage distinct from machine conversion values.
type Word uint64

// Word_Invariants covers complete magnitude word domain.
func Word_Invariants(value Word, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), bits.WORD_64_MINIMUM, bits.WORD_64_MAXIMUM).
		Ensure()
}

// Words_Unvalidated admits one hostile word beyond Int storage.
type Words_Unvalidated []Word

// Words_Unvalidated_Invariants bounds hostile magnitude work before validation.
func Words_Unvalidated_Invariants(value Words_Unvalidated, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), WORD_COUNT_MINIMUM, WORDS_UNVALIDATED_SIZE_MAXIMUM).
		Ensure()
}

// Random_Words_Unvalidated is one hostile caller-owned entropy batch.
type Random_Words_Unvalidated []Word

// Random_Words_Unvalidated_Invariants caps random work before source use.
func Random_Words_Unvalidated_Invariants(
	value Random_Words_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM,
			RANDOM_WORD_SIZE_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Random_Words is one validated caller-owned entropy batch.
type Random_Words []Word

// Random_Words_Invariants binds entropy use to one complete Int storage bound.
func Random_Words_Invariants(value Random_Words, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM, RANDOM_WORD_SIZE_MAXIMUM,
		).
		Ensure()
}

// Words is validated caller-owned little-endian magnitude storage.
type Words []Word

// Words_Invariants binds caller storage to one Int magnitude.
func Words_Invariants(value Words, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Ensure()
}

// Word_Count keeps normalized length distinct from word indexes.
type Word_Count int

// Word_Count_Invariants binds normalized length to owned storage.
func Word_Count_Invariants(value Word_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Ensure()
}

// Random_Word_Count reports entropy words consumed from one bounded source batch.
type Random_Word_Count int

// Random_Word_Count_Invariants stays inside validated entropy storage.
func Random_Word_Count_Invariants(value Random_Word_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, RANDOM_WORD_SIZE_MAXIMUM,
		).
		Ensure()
}

// Primality_Repetition_Count_Unvalidated admits one hostile value beyond each valid bound.
type Primality_Repetition_Count_Unvalidated int

// Primality_Repetition_Count_Unvalidated_Invariants bounds validation work.
func Primality_Repetition_Count_Unvalidated_Invariants(
	value Primality_Repetition_Count_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), PRIMALITY_REPETITION_COUNT_UNVALIDATED_MINIMUM,
			PRIMALITY_REPETITION_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Primality_Repetition_Count is requested pseudorandom Miller-Rabin work.
type Primality_Repetition_Count int

// Primality_Repetition_Count_Invariants stays inside one bounded entropy batch.
func Primality_Repetition_Count_Invariants(
	value Primality_Repetition_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), PRIMALITY_REPETITION_COUNT_MINIMUM,
			PRIMALITY_REPETITION_COUNT_MAXIMUM,
		).
		Ensure()
}

// Primality_Parameter_Count_Unvalidated admits missing or excessive Lucas search work.
type Primality_Parameter_Count_Unvalidated int

// Primality_Parameter_Count_Unvalidated_Invariants bounds validation work.
func Primality_Parameter_Count_Unvalidated_Invariants(
	value Primality_Parameter_Count_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), PRIMALITY_PARAMETER_COUNT_UNVALIDATED_MINIMUM,
			PRIMALITY_PARAMETER_COUNT_UNVALIDATED_MAXIMUM,
		).
		Ensure()
}

// Primality_Parameter_Count is explicit bounded Baillie-OEIS search work.
type Primality_Parameter_Count int

// Primality_Parameter_Count_Invariants preserves at least one parameter attempt.
func Primality_Parameter_Count_Invariants(
	value Primality_Parameter_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), PRIMALITY_PARAMETER_COUNT_MINIMUM,
			PRIMALITY_PARAMETER_COUNT_MAXIMUM,
		).
		Ensure()
}

// Primality_Trial_Factor is one bounded odd trial divisor.
type Primality_Trial_Factor Word

// Primality_Trial_Factor_Invariants binds trial work to one machine-word interval.
func Primality_Trial_Factor_Invariants(
	value Primality_Trial_Factor, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Uint64(
			uint64(value), PRIMALITY_TRIAL_FACTOR_MINIMUM,
			PRIMALITY_TRIAL_FACTOR_MAXIMUM,
		).
		Ensure()
	invariant.Always(
		Word(value)&JACOBI_PARITY_MASK != 0,
		"Trial division omits factors already covered by even rejection.",
	)
}

// Primality_Options_Unvalidated keeps every caller work limit explicit.
type Primality_Options_Unvalidated struct {
	// Repetitions bounds caller-entropy Miller-Rabin rounds before base-two round.
	Repetitions Primality_Repetition_Count_Unvalidated
	// Parameter_Count bounds deterministic Baillie-OEIS method-C search.
	Parameter_Count Primality_Parameter_Count_Unvalidated
}

// Primality_Options_Unvalidated_Invariants composes hostile bounded work limits.
func Primality_Options_Unvalidated_Invariants(
	value Primality_Options_Unvalidated, namespace invariant.Namespace,
) {
	Primality_Repetition_Count_Unvalidated_Invariants(value.Repetitions, namespace)
	Primality_Parameter_Count_Unvalidated_Invariants(value.Parameter_Count, namespace)
}

// Primality_Options carries validated bounded work limits.
type Primality_Options struct {
	// Repetitions is validated caller-entropy Miller-Rabin work.
	Repetitions Primality_Repetition_Count
	// Parameter_Count is validated deterministic Lucas search work.
	Parameter_Count Primality_Parameter_Count
}

// Primality_Options_Invariants composes validated probable-prime work limits.
func Primality_Options_Invariants(value Primality_Options, namespace invariant.Namespace) {
	Primality_Repetition_Count_Invariants(value.Repetitions, namespace)
	Primality_Parameter_Count_Invariants(value.Parameter_Count, namespace)
}

// Parse_Character is one hostile source byte before digit validation.
type Parse_Character uint8

// Parse_Character_Invariants covers complete byte input.
func Parse_Character_Invariants(value Parse_Character, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), uint8(bits.WORD_8_MINIMUM), uint8(bits.WORD_8_MAXIMUM)).
		Ensure()
}

// Parse_Digit is one decoded member of supported digit alphabet.
type Parse_Digit uint8

// Parse_Digit_Invariants binds decoded digits to supported alphabet.
func Parse_Digit_Invariants(value Parse_Digit, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint8(uint8(value), PARSE_DIGIT_MINIMUM, PARSE_DIGIT_MAXIMUM).
		Ensure()
}

// Parse_Text_Index skips only optional sign and base prefix.
type Parse_Text_Index int

// Parse_Text_Index_Invariants bounds first digit coordinate.
func Parse_Text_Index_Invariants(value Parse_Text_Index, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_4_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, SIGN_BYTE_COUNT_MAXIMUM,
			BASE_PREFIX_BYTE_COUNT, PARSE_TEXT_INDEX_MAXIMUM,
		).
		Ensure()
}

// Bitwise_Word_Count includes one sign-extension word beyond an Int magnitude.
type Bitwise_Word_Count int

// Bitwise_Word_Count_Invariants bounds temporary two's-complement width.
func Bitwise_Word_Count_Invariants(value Bitwise_Word_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), BITWISE_WORD_COUNT_MINIMUM, BITWISE_WORD_COUNT_MAXIMUM).
		Ensure()
}

// Quotient_Count normalizes division workspace quotient storage.
type Quotient_Count int

// Quotient_Count_Invariants binds scratch quotient length to one Int.
func Quotient_Count_Invariants(value Quotient_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Ensure()
}

// Remainder_Count normalizes division workspace remainder storage.
type Remainder_Count int

// Remainder_Count_Invariants binds scratch remainder length to one Int.
func Remainder_Count_Invariants(value Remainder_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), WORD_COUNT_MINIMUM, WORD_COUNT_MAXIMUM).
		Ensure()
}

// Word_Index keeps storage coordinates separate from normalized lengths.
type Word_Index int

// Word_Index_Invariants admits every storage coordinate.
func Word_Index_Invariants(value Word_Index, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), WORD_COUNT_MINIMUM, WORD_INDEX_MAXIMUM).
		Ensure()
}

// Bytes_Unvalidated admits one hostile byte beyond Int storage for graceful rejection.
type Bytes_Unvalidated []byte

// Bytes_Unvalidated_Invariants bounds hostile work before validation.
func Bytes_Unvalidated_Invariants(value Bytes_Unvalidated, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, BYTES_UNVALIDATED_SIZE_MAXIMUM).
		Ensure()
}

// Bytes is validated unsigned big-endian magnitude input.
type Bytes []byte

// Bytes_Invariants keeps validated input inside one Int magnitude.
func Bytes_Invariants(value Bytes, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, bytes.SLICE_SIZE_MAXIMUM).
		Ensure()
}

// Int_Encoding_Unvalidated admits empty gob input and one hostile oversized input.
type Int_Encoding_Unvalidated []byte

// Int_Encoding_Unvalidated_Invariants bounds hostile decoding before any magnitude read.
func Int_Encoding_Unvalidated_Invariants(
	value Int_Encoding_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM, INT_GOB_UNVALIDATED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Int_Encoding is caller-owned storage for one bounded gob representation.
type Int_Encoding []byte

// Int_Encoding_Invariants bounds output work by one complete gob representation.
func Int_Encoding_Invariants(value Int_Encoding, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, INT_GOB_SIZE_MAXIMUM).
		Ensure()
}

// Rat_Encoding_Count states required bytes for one rational gob representation.
type Rat_Encoding_Count int

// Rat_Encoding_Count_Invariants excludes impossible partial prefix counts.
func Rat_Encoding_Count_Invariants(value Rat_Encoding_Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), RAT_GOB_PREFIX_SIZE, RAT_GOB_SIZE_MAXIMUM).
		Ensure()
}

// Rat_Component_Byte_Count states one encoded component magnitude size.
type Rat_Component_Byte_Count int

// Rat_Component_Byte_Count_Invariants binds one component to rational word capacity.
func Rat_Component_Byte_Count_Invariants(
	value Rat_Component_Byte_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM, RAT_COMPONENT_BYTE_SIZE_MAXIMUM,
		).
		Ensure()
}

// Rat_Encoding_Unvalidated admits empty input and one hostile oversized encoding.
type Rat_Encoding_Unvalidated []byte

// Rat_Encoding_Unvalidated_Invariants bounds hostile rational decode work.
func Rat_Encoding_Unvalidated_Invariants(
	value Rat_Encoding_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM, RAT_GOB_UNVALIDATED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Rat_Encoding is caller-owned storage for one bounded rational gob representation.
type Rat_Encoding []byte

// Rat_Encoding_Invariants binds output work to complete rational wire bound.
func Rat_Encoding_Invariants(value Rat_Encoding, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, RAT_GOB_SIZE_MAXIMUM).
		Ensure()
}

// Rat_Gob_Encoding is one validated nonempty rational wire value.
type Rat_Gob_Encoding []byte

// Rat_Gob_Encoding_Invariants excludes incomplete gob prefixes.
func Rat_Gob_Encoding_Invariants(value Rat_Gob_Encoding, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), RAT_GOB_PREFIX_SIZE, RAT_GOB_SIZE_MAXIMUM).
		Ensure()
}

// Text_Unvalidated admits malformed or one-byte-oversized integer text.
type Text_Unvalidated []byte

// Text_Unvalidated_Invariants bounds hostile parse work before syntax validation.
func Text_Unvalidated_Invariants(value Text_Unvalidated, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM, TEXT_UNVALIDATED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Rat_Parse_Fraction_Text_Unvalidated admits malformed or one-byte-oversized fraction text.
type Rat_Parse_Fraction_Text_Unvalidated []byte

// Rat_Parse_Fraction_Text_Unvalidated_Invariants bounds hostile fraction scanning.
func Rat_Parse_Fraction_Text_Unvalidated_Invariants(
	value Rat_Parse_Fraction_Text_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM,
			RAT_PARSE_FRACTION_TEXT_UNVALIDATED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Rat_Parse_Text_Unvalidated admits malformed or one-byte-oversized rational text.
type Rat_Parse_Text_Unvalidated []byte

// Rat_Parse_Text_Unvalidated_Invariants bounds hostile rational scanning before syntax work.
func Rat_Parse_Text_Unvalidated_Invariants(
	value Rat_Parse_Text_Unvalidated, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM,
			RAT_PARSE_TEXT_UNVALIDATED_SIZE_MAXIMUM,
		).
		Ensure()
}

// Rat_Parse_Text is nonempty rational text inside complete bounded syntax storage.
type Rat_Parse_Text []byte

// Rat_Parse_Text_Invariants excludes sizes rejected at the public parse boundary.
func Rat_Parse_Text_Invariants(value Rat_Parse_Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SIGN_BYTE_COUNT_MAXIMUM, RAT_PARSE_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Rat_Parse_Negative records a leading minus independently from zero magnitude.
type Rat_Parse_Negative bool

// Rat_Parse_Negative_Invariants requires both leading-sign states.
func Rat_Parse_Negative_Invariants(value Rat_Parse_Negative, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "Rational parse text has a leading minus.").
		Ensure()
}

// Rat_Parse_Prefixed records whether a zero-letter radix prefix was consumed.
type Rat_Parse_Prefixed bool

// Rat_Parse_Prefixed_Invariants requires prefixed and decimal-default syntax.
func Rat_Parse_Prefixed_Invariants(value Rat_Parse_Prefixed, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "Rational parse text has an explicit radix prefix.").
		Ensure()
}

// Rat_Parse_Mantissa_Base is one base accepted by stdlib rational floating syntax.
type Rat_Parse_Mantissa_Base uint8

// Rat_Parse_Mantissa_Base_Invariants excludes integer-only bases from radix-point parsing.
func Rat_Parse_Mantissa_Base_Invariants(
	value Rat_Parse_Mantissa_Base, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(BASE_BINARY), uint8(BASE_OCTAL),
			uint8(BASE_DECIMAL), uint8(BASE_HEXADECIMAL),
		).
		Ensure()
}

// Rat_Parse_Source_Index is one coordinate after a nonempty mantissa scan.
type Rat_Parse_Source_Index int

// Rat_Parse_Source_Index_Invariants stays inside validated rational source.
func Rat_Parse_Source_Index_Invariants(
	value Rat_Parse_Source_Index, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), SIGN_BYTE_COUNT_MAXIMUM, RAT_PARSE_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Rat_Parse_Fractional_Digit_Count counts mantissa digits after one radix point.
type Rat_Parse_Fractional_Digit_Count int

// Rat_Parse_Fractional_Digit_Count_Invariants leaves source room for its radix point.
func Rat_Parse_Fractional_Digit_Count_Invariants(
	value Rat_Parse_Fractional_Digit_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			int(value), bytes.SLICE_SIZE_MINIMUM,
			RAT_PARSE_FRACTIONAL_DIGIT_COUNT_MAXIMUM,
		).
		Ensure()
}

// Rat_Parse_Exponent_Text is the complete suffix after one exponent marker.
type Rat_Parse_Exponent_Text []byte

// Rat_Parse_Exponent_Text_Invariants reserves source bytes for mantissa and marker.
func Rat_Parse_Exponent_Text_Invariants(
	value Rat_Parse_Exponent_Text, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), bytes.SLICE_SIZE_MINIMUM,
			RAT_PARSE_EXPONENT_TEXT_SIZE_MAXIMUM,
		).
		Ensure()
}

// Rat_Parse_Exponent preserves the complete signed stdlib exponent grammar.
type Rat_Parse_Exponent int64

// Rat_Parse_Exponent_Invariants rejects no value accepted by signed exponent syntax.
func Rat_Parse_Exponent_Invariants(
	value Rat_Parse_Exponent, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int64(int64(value), bits.INTEGER_64_MINIMUM, bits.INTEGER_64_MAXIMUM).
		Ensure()
}

// Rat_Parse_Exponent_Base selects decimal or binary exponent scaling.
type Rat_Parse_Exponent_Base uint8

// Rat_Parse_Exponent_Base_Invariants matches e and p exponent marker families.
func Rat_Parse_Exponent_Base_Invariants(
	value Rat_Parse_Exponent_Base, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(BASE_BINARY), uint8(BASE_DECIMAL)).
		Ensure()
}

// Rat_Parse_Exponents keeps split powers in fixed caller-independent scalar state.
type Rat_Parse_Exponents [RAT_PARSE_EXPONENT_COUNT]int64

// Rat_Parse_Exponents_Invariants fixes binary and decimal-prime slots.
func Rat_Parse_Exponents_Invariants(value Rat_Parse_Exponents, _ invariant.Namespace) {
	invariant.Always(
		len(value) == RAT_PARSE_EXPONENT_COUNT,
		"Rational parse has exactly two split exponent components.",
	)
}

// Rat_Parse_Power_Component selects numerator or denominator scaling.
type Rat_Parse_Power_Component int

// Rat_Parse_Power_Component_Invariants keeps scaling inside parsed fraction storage.
func Rat_Parse_Power_Component_Invariants(
	value Rat_Parse_Power_Component, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Enum_Int(
			int(value), RAT_PARSE_FRACTION_NUMERATOR_INDEX,
			RAT_PARSE_FRACTION_DENOMINATOR_INDEX,
		).
		Ensure()
}

// Rat_Parse_Power_Count is nonzero bounded repeated prime multiplication work.
type Rat_Parse_Power_Count int64

// Rat_Parse_Power_Count_Invariants binds repetition to complete integer bit capacity.
func Rat_Parse_Power_Count_Invariants(
	value Rat_Parse_Power_Count, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int64(
			int64(value), int64(SIGN_BYTE_COUNT_MAXIMUM),
			int64(RAT_PARSE_EXPONENT_MAGNITUDE_MAXIMUM),
		).
		Ensure()
}

// Rat_Parse_Prefix carries only sign, radix, and the first mantissa coordinate.
type Rat_Parse_Prefix struct {
	// Negative preserves a sign until a nonzero magnitude exists.
	Negative Rat_Parse_Negative
	// Prefixed permits one underscore immediately after an explicit radix marker.
	Prefixed Rat_Parse_Prefixed
	// Base follows explicit prefix syntax or decimal default.
	Base Rat_Parse_Mantissa_Base
	// Start skips optional sign and explicit base prefix.
	Start Parse_Text_Index
}

// Rat_Parse_Prefix_Invariants composes every bounded prefix decision once.
func Rat_Parse_Prefix_Invariants(value Rat_Parse_Prefix, namespace invariant.Namespace) {
	Rat_Parse_Negative_Invariants(value.Negative, namespace)
	Rat_Parse_Prefixed_Invariants(value.Prefixed, namespace)
	Rat_Parse_Mantissa_Base_Invariants(value.Base, namespace)
	Parse_Text_Index_Invariants(value.Start, namespace)
}

// Rat_Parse_Mantissa carries normalized scan state into exponent scaling.
type Rat_Parse_Mantissa struct {
	// Prefix retains sign and base after syntax bytes disappear.
	Prefix Rat_Parse_Prefix
	// Count bounds magnitude words already written into parse scratch.
	Count Word_Count
	// Fractional_Digits determines the radix-point exponent contribution.
	Fractional_Digits Rat_Parse_Fractional_Digit_Count
	// End locates either complete input or one exponent marker.
	End Rat_Parse_Source_Index
}

// Rat_Parse_Mantissa_Invariants composes completed bounded mantissa state.
func Rat_Parse_Mantissa_Invariants(
	value Rat_Parse_Mantissa, namespace invariant.Namespace,
) {
	Rat_Parse_Prefix_Invariants(value.Prefix, namespace)
	Word_Count_Invariants(value.Count, namespace)
	Rat_Parse_Fractional_Digit_Count_Invariants(value.Fractional_Digits, namespace)
	Rat_Parse_Source_Index_Invariants(value.End, namespace)
}

// Parse_Text is nonempty integer text inside complete signed binary bound.
type Parse_Text []byte

// Parse_Text_Invariants binds syntax work to validated input size.
func Parse_Text_Invariants(value Parse_Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), SIGN_BYTE_COUNT_MAXIMUM, INT_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Text is caller-owned integer representation storage.
type Text []byte

// Text_Invariants bounds output work by worst-case signed binary representation.
func Text_Invariants(value Text, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(len(value), bytes.SLICE_SIZE_MINIMUM, INT_TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Int stores magnitude inline because slice capacity would let arithmetic grow behind caller
// back.
type Int struct {
	// Negative stays separate from magnitude so positive and negative bounds remain symmetric.
	Negative Polarity
	// Count excludes high zero words so arithmetic touches only significant storage.
	Count Word_Count
	// Words stay inline so no operation can grow magnitude behind caller back.
	Words [WORD_COUNT_MAXIMUM]Word
}

// Int_Invariants makes normalization load-bearing instead of trusting every arithmetic path.
func Int_Invariants(value *Int, namespace invariant.Namespace) {
	Polarity_Invariants(value.Negative, namespace)
	Word_Count_Invariants(value.Count, namespace)
	normalization_minimum := Word(min(Word_Count(WORD_COUNT_INCREMENT), value.Count))
	invariant.Always(
		value.Words[(value.Count-Word_Count(WORD_COUNT_INCREMENT))&
			Word_Count(WORD_INDEX_MAXIMUM)] >= normalization_minimum,
		"A big integer omits high zero words.")
	invariant.Always(int(value.Negative) <= int(value.Count),
		"A big integer gives zero no negative twin.")
}

// Float_Mantissa gives the nonnegative magnitude one invariant-chain identity.
type Float_Mantissa Int

// Float_Mantissa_Invariants fixes normalization without duplicating Float sign polarity.
func Float_Mantissa_Invariants(value Float_Mantissa, namespace invariant.Namespace) {
	Word_Count_Invariants(value.Count, namespace)
	normalization_index := int(
		(uint(value.Count) + uint(WORD_INDEX_MAXIMUM)) % uint(WORD_COUNT_MAXIMUM),
	)
	normalization_word := uint64(value.Words[normalization_index])
	normalization_nonzero := (normalization_word | -normalization_word) >>
		WORD_BIT_INDEX_MAXIMUM
	invariant.Always(
		int(value.Count)*int(bits.CARRY_MAXIMUM-normalization_nonzero) ==
			WORD_COUNT_MINIMUM,
		"A float mantissa omits high zero words.",
	)
	invariant.Always(value.Negative == POLARITY_NONNEGATIVE,
		"A float stores sign outside its mantissa.")
}

// Float_Active_Mantissa excludes zero inside finite-only helpers.
type Float_Active_Mantissa Float_Mantissa

// Float_Active_Mantissa_Invariants fixes one normalized nonzero inline magnitude.
func Float_Active_Mantissa_Invariants(
	value Float_Active_Mantissa, namespace invariant.Namespace,
) {
	invariant.Tree(value, namespace).
		Range_Int(int(value.Count), WORD_COUNT_INCREMENT, WORD_COUNT_MAXIMUM).
		Ensure()
	invariant.Always(len(value.Words) == WORD_COUNT_MAXIMUM,
		"An active float mantissa owns its complete word bound.")
	normalization_index := int(
		(uint(value.Count) + uint(WORD_INDEX_MAXIMUM)) % uint(WORD_COUNT_MAXIMUM),
	)
	normalization_word := uint64(value.Words[normalization_index])
	invariant.Always(normalization_word != 0,
		"An active float mantissa omits high zero words.")
	invariant.Always(value.Negative == POLARITY_NONNEGATIVE,
		"An active float stores sign outside its mantissa.")
}

// Float stores one bounded binary mantissa and normalized exponent without slice growth.
type Float struct {
	// Precision limits significant mantissa bits while zero preserves later operation policy.
	Precision Float_Precision
	// Mode chooses the direction for every inexact destination write.
	Mode Rounding_Mode
	// Accuracy orders the latest rounded result against its exact source.
	Accuracy Accuracy
	// Form keeps signed zero and infinity outside finite mantissa rules.
	Form Float_Form
	// Negative preserves the IEEE sign bit, including both zeros and infinities.
	Negative Polarity
	// Mantissa stores finite magnitude with its highest significant bit normalized.
	Mantissa Float_Mantissa
	// Exponent places the binary point immediately after the highest significant bit.
	Exponent Float_Exponent
}

// Float_Invariants makes finite normalization and nonfinite storage explicit.
func Float_Invariants(value *Float, namespace invariant.Namespace) {
	Float_Precision_Invariants(value.Precision, namespace)
	Rounding_Mode_Invariants(value.Mode, namespace)
	Accuracy_Invariants(value.Accuracy, namespace)
	Float_Form_Invariants(value.Form, namespace)
	Polarity_Invariants(value.Negative, namespace)
	Float_Mantissa_Invariants(value.Mantissa, namespace)
	Float_Exponent_Invariants(value.Exponent, namespace)
	finite := int(value.Form) & WORD_COUNT_INCREMENT
	count := int(value.Mantissa.Count)
	high_index := int(
		(uint(value.Mantissa.Count) + uint(WORD_INDEX_MAXIMUM)) %
			uint(WORD_COUNT_MAXIMUM),
	)
	high := uint64(value.Mantissa.Words[high_index])
	precision_word_count := (int(value.Precision) + WORD_BIT_INDEX_MAXIMUM) / WORD_BIT_COUNT
	excess_bit_count := count*WORD_BIT_COUNT - int(value.Precision)
	excess_unsigned := uint(excess_bit_count)
	excess_nonzero := (excess_unsigned | -excess_unsigned) >> WORD_BIT_INDEX_MAXIMUM
	excess_nonnegative := uint(excess_bit_count>>WORD_BIT_INDEX_MAXIMUM) +
		uint(bits.CARRY_MAXIMUM)
	excess_positive := excess_nonzero * excess_nonnegative
	unused_bit_count := excess_unsigned * excess_positive
	high_maximum := uint64(bits.WORD_64_MAXIMUM) >> uint(unused_bit_count)
	invariant.Always(finite <= int(value.Precision),
		"A finite float has positive precision.")
	invariant.Always(finite <= count,
		"A finite float has nonzero mantissa.")
	invariant.Always(count*finite <= precision_word_count,
		"A finite float mantissa word count fits its declared precision.")
	invariant.Always(
		uint64(finite)*uint64(excess_positive)*(high&^high_maximum) == 0,
		"A finite float mantissa fits its declared precision.")
	invariant.Always(int(value.Mantissa.Count)*(WORD_COUNT_INCREMENT-finite) ==
		WORD_COUNT_MINIMUM,
		"A nonfinite float stores no mantissa.")
	invariant.Always(int(value.Exponent)*(WORD_COUNT_INCREMENT-finite) == 0,
		"A nonfinite float has no exponent payload.")
}

// Float_Finite narrows helper input after public Float validation proves finite form.
type Float_Finite Float

// Float_Finite_Invariants retains exact finite field domains without impossible nonfinite cases.
func Float_Finite_Invariants(value *Float_Finite, namespace invariant.Namespace) {
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
	invariant.Tree(Polarity(value.Negative), namespace).
		Enum_Uint8(
			uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE),
		).
		Ensure()
	invariant.Tree(Float_Exponent(value.Exponent), namespace).
		Range_Int(
			int(value.Exponent), FLOAT_EXPONENT_MINIMUM, FLOAT_EXPONENT_MAXIMUM,
		).
		Ensure()
	Float_Active_Mantissa_Invariants(Float_Active_Mantissa(value.Mantissa), namespace)
	invariant.Always(value.Form == FLOAT_FORM_FINITE,
		"A finite helper value has finite form.")
}

// Float_Rounding_Source narrows one finite value that must lose at least one bit.
type Float_Rounding_Source Float

// Float_Rounding_Source_Invariants states exact domains after reduction is proven necessary.
func Float_Rounding_Source_Invariants(
	value *Float_Rounding_Source, namespace invariant.Namespace,
) {
	invariant.Tree(Float_Rounding_Source_Precision(value.Precision), namespace).
		Range_Uint(
			uint(value.Precision), BASE_BINARY, FLOAT_PRECISION_MAXIMUM,
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
	invariant.Tree(Polarity(value.Negative), namespace).
		Enum_Uint8(
			uint8(value.Negative), uint8(POLARITY_NONNEGATIVE),
			uint8(POLARITY_NEGATIVE),
		).
		Ensure()
	invariant.Tree(Float_Exponent(value.Exponent), namespace).
		Range_Int(
			int(value.Exponent), FLOAT_EXPONENT_MINIMUM, FLOAT_EXPONENT_MAXIMUM,
		).
		Ensure()
	Float_Active_Mantissa_Invariants(Float_Active_Mantissa(value.Mantissa), namespace)
	invariant.Always(value.Form == FLOAT_FORM_FINITE,
		"A rounding source has finite form.")
}

// Float_Addition_Workspace holds the exact union of every bounded finite bit position.
type Float_Addition_Workspace struct {
	// Result keeps alignment and arithmetic outside aliased caller values.
	Result [FLOAT_ADDITION_WORD_COUNT_MAXIMUM]Word
}

// Float_Addition_Workspace_Invariants binds scratch to the derived exponent and mantissa span.
func Float_Addition_Workspace_Invariants(
	value *Float_Addition_Workspace, namespace invariant.Namespace,
) {
	invariant.Always(
		len(value.Result) == FLOAT_ADDITION_WORD_COUNT_MAXIMUM,
		"A float addition workspace covers every bounded finite bit position.",
	)
}

// Float_Multiplication_Workspace retains the shared result span used by normalization.
type Float_Multiplication_Workspace struct {
	// Result holds the product before bounded precision reduction.
	Result [FLOAT_ADDITION_WORD_COUNT_MAXIMUM]Word
}

// Float_Multiplication_Workspace_Invariants binds products to the common arithmetic span.
func Float_Multiplication_Workspace_Invariants(
	value *Float_Multiplication_Workspace, namespace invariant.Namespace,
) {
	invariant.Always(
		len(value.Result) == FLOAT_ADDITION_WORD_COUNT_MAXIMUM,
		"A float multiplication workspace retains the complete arithmetic result span.",
	)
}

// Float_Division_Workspace holds aligned significands without growing either operand.
type Float_Division_Workspace struct {
	// Remainder carries one extra high bit during binary quotient extraction.
	Remainder [FLOAT_DIVISION_WORD_COUNT_MAXIMUM]Word
	// Divisor keeps the normalized right significand stable across every quotient bit.
	Divisor [FLOAT_DIVISION_WORD_COUNT_MAXIMUM]Word
}

// Float_Division_Workspace_Invariants binds both arrays to one shifted mantissa bound.
func Float_Division_Workspace_Invariants(
	value *Float_Division_Workspace, namespace invariant.Namespace,
) {
	invariant.Always(
		len(value.Remainder) == FLOAT_DIVISION_WORD_COUNT_MAXIMUM,
		"A float division remainder retains one shifted carry word.",
	)
	invariant.Always(
		len(value.Divisor) == FLOAT_DIVISION_WORD_COUNT_MAXIMUM,
		"A float division divisor retains one aligned mantissa.",
	)
}

// Int_Multiplication_Workspace owns product storage so alias-safe multiplication never allocates.
type Int_Multiplication_Workspace struct {
	// Product keeps partial sums separate from every input and destination.
	Product [WORD_COUNT_MAXIMUM]Word
}

// Int_Multiplication_Workspace_Invariants binds scratch storage to one Int magnitude.
func Int_Multiplication_Workspace_Invariants(
	value *Int_Multiplication_Workspace, namespace invariant.Namespace,
) {
	invariant.Always(
		len(value.Product) == WORD_COUNT_MAXIMUM,
		"A multiplication workspace owns one complete product bound.",
	)
}

// Int_Division_Workspace owns both magnitudes so all input and output aliases remain safe.
type Int_Division_Workspace struct {
	// Quotient keeps quotient bits outside every caller Int until division succeeds.
	Quotient [WORD_COUNT_MAXIMUM]Word
	// Remainder keeps each restoring-division prefix outside caller values.
	Remainder [WORD_COUNT_MAXIMUM]Word
	// Quotient_Count normalizes Quotient after its highest set bit.
	Quotient_Count Quotient_Count
	// Remainder_Count normalizes Remainder after every subtraction.
	Remainder_Count Remainder_Count
}

// Int_Division_Workspace_Invariants binds each scratch magnitude to one Int bound.
func Int_Division_Workspace_Invariants(
	value *Int_Division_Workspace, namespace invariant.Namespace,
) {
	Quotient_Count_Invariants(value.Quotient_Count, namespace)
	Remainder_Count_Invariants(value.Remainder_Count, namespace)
	invariant.Always(
		len(value.Quotient) == WORD_COUNT_MAXIMUM,
		"A division workspace owns one complete quotient bound.",
	)
	invariant.Always(
		len(value.Remainder) == WORD_COUNT_MAXIMUM,
		"A division workspace owns one complete remainder bound.",
	)
}

// Int_Bitwise_Workspace owns signed scratch words so aliases never destroy unread inputs.
type Int_Bitwise_Workspace struct {
	// Left stores left input with one explicit sign-extension word.
	Left [BITWISE_WORD_COUNT_MAXIMUM]Word
	// Right stores right input with one explicit sign-extension word.
	Right [BITWISE_WORD_COUNT_MAXIMUM]Word
	// Result keeps signed output outside caller destination until range validation succeeds.
	Result [BITWISE_WORD_COUNT_MAXIMUM]Word
}

// Int_Bitwise_Workspace_Invariants binds every signed scratch value to one fixed bound.
func Int_Bitwise_Workspace_Invariants(
	value *Int_Bitwise_Workspace, namespace invariant.Namespace,
) {
	invariant.Always(
		len(value.Left) == BITWISE_WORD_COUNT_MAXIMUM,
		"A bitwise workspace owns one complete left signed bound.",
	)
	invariant.Always(
		len(value.Right) == BITWISE_WORD_COUNT_MAXIMUM,
		"A bitwise workspace owns one complete right signed bound.",
	)
	invariant.Always(
		len(value.Result) == BITWISE_WORD_COUNT_MAXIMUM,
		"A bitwise workspace owns one complete result signed bound.",
	)
}

// Euclidean_Integers owns rotating integer state without repeated invariant leaf types.
type Euclidean_Integers [EUCLIDEAN_INTEGER_COUNT]Int

// Euclidean_Integers_Invariants fixes complete rotating state capacity.
func Euclidean_Integers_Invariants(value Euclidean_Integers, _ invariant.Namespace) {
	invariant.Always(
		len(value) == EUCLIDEAN_INTEGER_COUNT,
		"Euclidean integer storage has fixed rotating-state capacity.",
	)
}

// Division_Memory is caller-owned restoring scratch reusable by composite algorithms.
type Division_Memory struct {
	// Quotient keeps discarded quotient bits outside algorithm state.
	Quotient [WORD_COUNT_MAXIMUM]Word
	// Remainder keeps current restoring-division prefix.
	Remainder [WORD_COUNT_MAXIMUM]Word
	// Quotient_Count normalizes Quotient after its highest set bit.
	Quotient_Count Quotient_Count
	// Remainder_Count normalizes Remainder after every subtraction.
	Remainder_Count Remainder_Count
}

// Division_Memory_Invariants binds scratch counts and storage to shared division bounds.
func Division_Memory_Invariants(value Division_Memory, namespace invariant.Namespace) {
	Quotient_Count_Invariants(value.Quotient_Count, namespace)
	Remainder_Count_Invariants(value.Remainder_Count, namespace)
	invariant.Always(
		len(value.Quotient) == WORD_COUNT_MAXIMUM,
		"Division memory owns one complete quotient bound.",
	)
	invariant.Always(
		len(value.Remainder) == WORD_COUNT_MAXIMUM,
		"Division memory owns one complete remainder bound.",
	)
}

// Int_Greatest_Common_Divisor_Workspace owns Euclidean state outside caller values.
type Int_Greatest_Common_Divisor_Workspace struct {
	// Integers rotate dividend, divisor, and remainder without local escaping storage.
	Integers Euclidean_Integers
	// Division owns restoring-division scratch.
	Division Division_Memory
}

// Int_Greatest_Common_Divisor_Workspace_Invariants composes fixed integer and division storage.
func Int_Greatest_Common_Divisor_Workspace_Invariants(
	value *Int_Greatest_Common_Divisor_Workspace, namespace invariant.Namespace,
) {
	Euclidean_Integers_Invariants(value.Integers, namespace)
	Division_Memory_Invariants(value.Division, namespace)
}

// Square_Root_Integers owns Newton state without repeated invariant leaf types.
type Square_Root_Integers [SQUARE_ROOT_INTEGER_COUNT]Int

// Square_Root_Integers_Invariants fixes complete Newton state capacity.
func Square_Root_Integers_Invariants(value Square_Root_Integers, _ invariant.Namespace) {
	invariant.Always(
		len(value) == SQUARE_ROOT_INTEGER_COUNT,
		"Square-root integer storage has fixed Newton-state capacity.",
	)
}

// Int_Square_Root_Workspace owns every Newton temporary outside caller values.
type Int_Square_Root_Workspace struct {
	// Integers hold current, quotient, and next approximations.
	Integers Square_Root_Integers
	// Division owns source-by-approximation scratch.
	Division Division_Memory
}

// Int_Square_Root_Workspace_Invariants composes fixed Newton and division storage.
func Int_Square_Root_Workspace_Invariants(
	value *Int_Square_Root_Workspace, namespace invariant.Namespace,
) {
	Square_Root_Integers_Invariants(value.Integers, namespace)
	Division_Memory_Invariants(value.Division, namespace)
}

// Int_Random_Integers owns one candidate without exposing partial entropy in destination.
type Int_Random_Integers [INT_RANDOM_INTEGER_COUNT]Int

// Int_Random_Integers_Invariants fixes one complete candidate slot.
func Int_Random_Integers_Invariants(value Int_Random_Integers, _ invariant.Namespace) {
	invariant.Always(
		len(value) == INT_RANDOM_INTEGER_COUNT,
		"Random integer storage has fixed candidate capacity.",
	)
}

// Int_Random_Workspace owns rejected candidates outside caller destination.
type Int_Random_Workspace struct {
	// Integers keeps each candidate transactional.
	Integers Int_Random_Integers
}

// Int_Random_Workspace_Invariants fixes complete bounded random storage.
func Int_Random_Workspace_Invariants(
	value *Int_Random_Workspace, namespace invariant.Namespace,
) {
	Int_Random_Integers_Invariants(value.Integers, namespace)
}

// Jacobi_Integers owns rotating values without repeating Int invariant leaves.
type Jacobi_Integers [JACOBI_INTEGER_COUNT]Int

// Jacobi_Integers_Invariants fixes complete binary-Jacobi state capacity.
func Jacobi_Integers_Invariants(value Jacobi_Integers, _ invariant.Namespace) {
	invariant.Always(
		len(value) == JACOBI_INTEGER_COUNT,
		"Jacobi integer storage has fixed binary-Euclidean capacity.",
	)
}

// Int_Jacobi_Workspace owns every reduced value outside caller inputs.
type Int_Jacobi_Workspace struct {
	// Integers preserve inputs while modulus and shifts rotate values.
	Integers Jacobi_Integers
	// Division keeps restoring-division state outside Jacobi values.
	Division Division_Memory
}

// Int_Jacobi_Workspace_Invariants composes fixed integer and division storage.
func Int_Jacobi_Workspace_Invariants(
	value *Int_Jacobi_Workspace, namespace invariant.Namespace,
) {
	Jacobi_Integers_Invariants(value.Integers, namespace)
	Division_Memory_Invariants(value.Division, namespace)
}

// Primality_Integers owns every value reused across Miller-Rabin and Lucas phases.
type Primality_Integers [PRIMALITY_INTEGER_COUNT]Int

// Primality_Integers_Invariants fixes complete probable-prime integer capacity.
func Primality_Integers_Invariants(value Primality_Integers, _ invariant.Namespace) {
	invariant.Always(
		len(value) == PRIMALITY_INTEGER_COUNT,
		"Primality integer storage has fixed Miller-Rabin and Lucas capacity.",
	)
}

// Primality_Modular_Memory gives nested modular storage one invariant-chain identity.
type Primality_Modular_Memory Int_Modular_Workspace

// Primality_Modular_Memory_Invariants composes modular state once under primality.
func Primality_Modular_Memory_Invariants(
	value Primality_Modular_Memory, namespace invariant.Namespace,
) {
	invariant.Always(
		len(value.Integers) == MODULAR_INTEGER_COUNT,
		"Primality modular memory owns complete algorithm state.",
	)
	Multiplication_Memory_Invariants(value.Multiplication, namespace)
	Division_Memory_Invariants(value.Division, namespace)
}

// Primality_Random_Memory gives candidate storage one invariant-chain identity.
type Primality_Random_Memory Int_Random_Workspace

// Primality_Random_Memory_Invariants fixes transactional random-base storage.
func Primality_Random_Memory_Invariants(
	value Primality_Random_Memory, namespace invariant.Namespace,
) {
	invariant.Always(
		len(value.Integers) == INT_RANDOM_INTEGER_COUNT,
		"Primality random memory owns one transactional candidate.",
	)
}

// Int_Primality_Workspace owns every temporary and nested arithmetic workspace.
type Int_Primality_Workspace struct {
	// Integers preserve tested value while subalgorithms destroy their own state.
	Integers Primality_Integers
	// Modular owns overflow-free exponentiation and recurrence products.
	Modular Primality_Modular_Memory
	// Random keeps rejected bases outside primality state.
	Random Primality_Random_Memory
	// Jacobi owns binary reduction during Lucas parameter selection.
	Jacobi Jacobi_Integers
}

// Int_Primality_Workspace_Invariants composes complete fixed probable-prime storage.
func Int_Primality_Workspace_Invariants(
	value *Int_Primality_Workspace, namespace invariant.Namespace,
) {
	Primality_Integers_Invariants(value.Integers, namespace)
	Primality_Modular_Memory_Invariants(value.Modular, namespace)
	Primality_Random_Memory_Invariants(value.Random, namespace)
	Jacobi_Integers_Invariants(value.Jacobi, namespace)
}

// Exponent_Integers owns binary exponentiation state without repeated invariant leaf types.
type Exponent_Integers [EXPONENT_INTEGER_COUNT]Int

// Exponent_Integers_Invariants fixes complete binary exponentiation capacity.
func Exponent_Integers_Invariants(value Exponent_Integers, _ invariant.Namespace) {
	invariant.Always(
		len(value) == EXPONENT_INTEGER_COUNT,
		"Exponent integer storage has fixed binary-exponentiation capacity.",
	)
}

// Int_Exponent_Workspace owns every product and operand copy outside caller values.
type Int_Exponent_Workspace struct {
	// Integers hold accumulated result, squared factor, and exponent copy.
	Integers Exponent_Integers
	// Multiplication keeps each next value transactional.
	Multiplication Multiplication_Memory
}

// Int_Exponent_Workspace_Invariants composes fixed integer and product storage.
func Int_Exponent_Workspace_Invariants(
	value *Int_Exponent_Workspace, namespace invariant.Namespace,
) {
	Exponent_Integers_Invariants(value.Integers, namespace)
	Multiplication_Memory_Invariants(value.Multiplication, namespace)
}

// Modular_Integers owns Euclidean, exponentiation, and modular-product state.
type Modular_Integers [MODULAR_INTEGER_COUNT]Int

// Modular_Integers_Invariants fixes complete modular arithmetic capacity.
func Modular_Integers_Invariants(value Modular_Integers, _ invariant.Namespace) {
	invariant.Always(
		len(value) == MODULAR_INTEGER_COUNT,
		"Modular integer storage has fixed algorithm-state capacity.",
	)
}

// Int_Modular_Workspace owns every modular temporary outside caller values.
type Int_Modular_Workspace struct {
	// Integers hold Euclidean coefficients and overflow-free product state.
	Integers Modular_Integers
	// Multiplication avoids bit-serial work when the exact product stays inside the Int bound.
	Multiplication Multiplication_Memory
	// Division owns every quotient and Euclidean remainder calculation.
	Division Division_Memory
}

// Int_Modular_Workspace_Invariants composes fixed integer and division storage.
func Int_Modular_Workspace_Invariants(
	value *Int_Modular_Workspace, namespace invariant.Namespace,
) {
	Modular_Integers_Invariants(value.Integers, namespace)
	Multiplication_Memory_Invariants(value.Multiplication, namespace)
	Division_Memory_Invariants(value.Division, namespace)
}

// Product_Integers owns range-product state without repeated invariant leaf types.
type Product_Integers [PRODUCT_INTEGER_COUNT]Int

// Product_Integers_Invariants fixes complete range-product capacity.
func Product_Integers_Invariants(value Product_Integers, _ invariant.Namespace) {
	invariant.Always(
		len(value) == PRODUCT_INTEGER_COUNT,
		"Product integer storage has fixed range-product capacity.",
	)
}

// Int_Product_Workspace owns accumulator, factor, and multiplication storage.
type Int_Product_Workspace struct {
	// Integers keep partial product outside caller destination.
	Integers Product_Integers
	// Multiplication keeps every next product transactional.
	Multiplication Multiplication_Memory
}

// Int_Product_Workspace_Invariants composes fixed integer and product storage.
func Int_Product_Workspace_Invariants(
	value *Int_Product_Workspace, namespace invariant.Namespace,
) {
	Product_Integers_Invariants(value.Integers, namespace)
	Multiplication_Memory_Invariants(value.Multiplication, namespace)
}

// Int_Text_Words owns one mutable magnitude copy for repeated small-base division.
type Int_Text_Words [WORD_COUNT_MAXIMUM]Word

// Int_Text_Words_Invariants fixes one complete magnitude capacity.
func Int_Text_Words_Invariants(value Int_Text_Words, _ invariant.Namespace) {
	invariant.Always(
		len(value) == WORD_COUNT_MAXIMUM,
		"Integer text word storage has fixed magnitude capacity.",
	)
}

// Int_Text_Digits owns reversed magnitude digits until caller capacity is known.
type Int_Text_Digits [INT_TEXT_SIZE_MAXIMUM]byte

// Int_Text_Digits_Invariants fixes worst-case signed binary capacity.
func Int_Text_Digits_Invariants(value Int_Text_Digits, _ invariant.Namespace) {
	invariant.Always(
		len(value) == INT_TEXT_SIZE_MAXIMUM,
		"Integer text digit storage has fixed signed binary capacity.",
	)
}

// Int_Text_Workspace owns conversion copies so output remains transactional.
type Int_Text_Workspace struct {
	// Words is mutable magnitude during repeated division.
	Words Int_Text_Words
	// Digits keeps least-significant digit first until final copy.
	Digits Int_Text_Digits
}

// Int_Text_Workspace_Invariants composes fixed magnitude and digit storage.
func Int_Text_Workspace_Invariants(
	value *Int_Text_Workspace, namespace invariant.Namespace,
) {
	Int_Text_Words_Invariants(value.Words, namespace)
	Int_Text_Digits_Invariants(value.Digits, namespace)
}

// Rat_Text_Integers owns expanded denominator without repeated Int invariant leaves.
type Rat_Text_Integers [RAT_TEXT_INTEGER_COUNT]Int

// Rat_Text_Integers_Invariants fixes complete rational text integer capacity.
func Rat_Text_Integers_Invariants(value Rat_Text_Integers, _ invariant.Namespace) {
	invariant.Always(
		len(value) == RAT_TEXT_INTEGER_COUNT,
		"Rational text integer storage has fixed denominator capacity.",
	)
}

// Rat_Text_Integer_Workspaces owns independent reversed digits for both components.
type Rat_Text_Integer_Workspaces [RAT_TEXT_WORKSPACE_COUNT]Int_Text_Workspace

// Rat_Text_Integer_Workspaces_Invariants fixes complete component conversion capacity.
func Rat_Text_Integer_Workspaces_Invariants(
	value Rat_Text_Integer_Workspaces, _ invariant.Namespace,
) {
	invariant.Always(
		len(value) == RAT_TEXT_WORKSPACE_COUNT,
		"Rational text conversion storage has fixed component capacity.",
	)
}

// Rat_Text_Workspace owns expanded denominator and both reversed component representations.
type Rat_Text_Workspace struct {
	// Integers expands implicit denominator one outside caller value.
	Integers Rat_Text_Integers
	// Text keeps numerator and denominator digits until caller capacity is known.
	Text Rat_Text_Integer_Workspaces
}

// Rat_Text_Workspace_Invariants composes fixed integer and text storage.
func Rat_Text_Workspace_Invariants(value *Rat_Text_Workspace, namespace invariant.Namespace) {
	Rat_Text_Integers_Invariants(value.Integers, namespace)
	Rat_Text_Integer_Workspaces_Invariants(value.Text, namespace)
}

// Rat_Float_Text_Integers owns every fixed-decimal arithmetic value.
type Rat_Float_Text_Integers [RAT_FLOAT_INTEGER_COUNT]Int

// Rat_Float_Text_Integers_Invariants fixes complete fixed-decimal integer capacity.
func Rat_Float_Text_Integers_Invariants(value Rat_Float_Text_Integers, _ invariant.Namespace) {
	invariant.Always(
		len(value) == RAT_FLOAT_INTEGER_COUNT,
		"Rational fixed-decimal integer storage has fixed capacity.",
	)
}

// Rat_Float_Text_Integer_Workspaces owns reversed integer and fractional digits.
type Rat_Float_Text_Integer_Workspaces [RAT_FLOAT_TEXT_WORKSPACE_COUNT]Int_Text_Workspace

// Rat_Float_Text_Integer_Workspaces_Invariants fixes both decimal conversion stores.
func Rat_Float_Text_Integer_Workspaces_Invariants(
	value Rat_Float_Text_Integer_Workspaces, _ invariant.Namespace,
) {
	invariant.Always(
		len(value) == RAT_FLOAT_TEXT_WORKSPACE_COUNT,
		"Rational fixed-decimal conversion storage has fixed capacity.",
	)
}

// Rat_Float_Multiplication_Memory owns scaled remainder product.
type Rat_Float_Multiplication_Memory Multiplication_Memory

// Rat_Float_Multiplication_Memory_Invariants fixes one complete scaled product.
func Rat_Float_Multiplication_Memory_Invariants(
	value Rat_Float_Multiplication_Memory, _ invariant.Namespace,
) {
	invariant.Always(
		len(value.Product) == WORD_COUNT_MAXIMUM,
		"Rational fixed-decimal multiplication owns one complete product bound.",
	)
}

// Rat_Float_Exponent_Memory owns decimal scale exponentiation state.
type Rat_Float_Exponent_Memory struct {
	// Integers keep scale result, factor, and exponent outside formatting state.
	Integers Exponent_Integers
	// Multiplication keeps every exponentiation product transactional.
	Multiplication Multiplication_Memory
}

// Rat_Float_Exponent_Memory_Invariants composes decimal scale construction state.
func Rat_Float_Exponent_Memory_Invariants(
	value Rat_Float_Exponent_Memory, namespace invariant.Namespace,
) {
	Exponent_Integers_Invariants(value.Integers, namespace)
	Multiplication_Memory_Invariants(value.Multiplication, namespace)
}

// Rat_Float_Text_Workspace owns decimal scaling, division, rounding, and conversion storage.
type Rat_Float_Text_Workspace struct {
	// Integers hold exact arithmetic state outside caller value.
	Integers Rat_Float_Text_Integers
	// Division owns both quotient and remainder calculations.
	Division Division_Memory
	// Multiplication owns scaled remainder product.
	Multiplication Rat_Float_Multiplication_Memory
	// Exponent owns decimal scale construction.
	Exponent Rat_Float_Exponent_Memory
	// Text keeps both reversed decimal representations until destination preflight passes.
	Text Rat_Float_Text_Integer_Workspaces
}

// Rat_Float_Text_Workspace_Invariants composes complete fixed-decimal scratch storage.
func Rat_Float_Text_Workspace_Invariants(
	value *Rat_Float_Text_Workspace, namespace invariant.Namespace,
) {
	Rat_Float_Text_Integers_Invariants(value.Integers, namespace)
	Division_Memory_Invariants(value.Division, namespace)
	Rat_Float_Multiplication_Memory_Invariants(value.Multiplication, namespace)
	Rat_Float_Exponent_Memory_Invariants(value.Exponent, namespace)
	Rat_Float_Text_Integer_Workspaces_Invariants(value.Text, namespace)
}

// Rat_Float_Precision_Integers owns mutable denominator factorization state.
type Rat_Float_Precision_Integers [RAT_FLOAT_PRECISION_INTEGER_COUNT]Int

// Rat_Float_Precision_Integers_Invariants fixes one denominator capacity.
func Rat_Float_Precision_Integers_Invariants(
	value Rat_Float_Precision_Integers, _ invariant.Namespace,
) {
	invariant.Always(
		len(value) == RAT_FLOAT_PRECISION_INTEGER_COUNT,
		"Rational decimal precision storage has fixed denominator capacity.",
	)
}

// Rat_Float_Precision_Workspace owns mutable factorization outside caller rational.
type Rat_Float_Precision_Workspace struct {
	// Integers keep denominator reduction separate from caller value.
	Integers Rat_Float_Precision_Integers
}

// Rat_Float_Precision_Workspace_Invariants composes decimal precision storage.
func Rat_Float_Precision_Workspace_Invariants(
	value *Rat_Float_Precision_Workspace, namespace invariant.Namespace,
) {
	Rat_Float_Precision_Integers_Invariants(value.Integers, namespace)
}

// Rat_Float_64_Integers owns scaled operands and quotient-remainder output.
type Rat_Float_64_Integers [RAT_FLOAT_64_INTEGER_COUNT]Int

// Rat_Float_64_Integers_Invariants fixes complete binary64 conversion capacity.
func Rat_Float_64_Integers_Invariants(
	value Rat_Float_64_Integers, _ invariant.Namespace,
) {
	invariant.Always(
		len(value) == RAT_FLOAT_64_INTEGER_COUNT,
		"Rational binary64 conversion has fixed integer capacity.",
	)
}

// Rat_Float_64_Workspace owns scaling and division outside caller rational.
type Rat_Float_64_Workspace struct {
	// Integers hold scaled operands and division results.
	Integers Rat_Float_64_Integers
	// Division owns rounding quotient and discarded remainder.
	Division Division_Memory
}

// Rat_Float_64_Workspace_Invariants composes complete binary64 conversion storage.
func Rat_Float_64_Workspace_Invariants(
	value *Rat_Float_64_Workspace, namespace invariant.Namespace,
) {
	Rat_Float_64_Integers_Invariants(value.Integers, namespace)
	Division_Memory_Invariants(value.Division, namespace)
}

// Rat_Parse_Fraction_Integers owns parsed components before normalization.
type Rat_Parse_Fraction_Integers [RAT_PARSE_FRACTION_INTEGER_COUNT]Int

// Rat_Parse_Fraction_Integers_Invariants fixes both parsed component slots.
func Rat_Parse_Fraction_Integers_Invariants(
	value Rat_Parse_Fraction_Integers, _ invariant.Namespace,
) {
	invariant.Always(
		len(value) == RAT_PARSE_FRACTION_INTEGER_COUNT,
		"Rational fraction parse has fixed component capacity.",
	)
}

// Rat_Parse_Fraction_Integer_Workspaces owns independent component parse scratch.
type Rat_Parse_Fraction_Integer_Workspaces [RAT_PARSE_WORKSPACE_COUNT]Int_Parse_Workspace

// Rat_Parse_Fraction_Integer_Workspaces_Invariants fixes both parse stores.
func Rat_Parse_Fraction_Integer_Workspaces_Invariants(
	value Rat_Parse_Fraction_Integer_Workspaces, _ invariant.Namespace,
) {
	invariant.Always(
		len(value) == RAT_PARSE_WORKSPACE_COUNT,
		"Rational fraction parse has fixed integer parse capacity.",
	)
}

// Rat_Parse_Rational_Memory owns normalization outside destination rational.
type Rat_Parse_Rational_Memory struct {
	// Integers hold copied components and normalization results.
	Integers Rat_Operation_Integers
	// Greatest_Common owns reduction state.
	Greatest_Common Greatest_Common_Divisor_Memory
	// Multiplication preserves Rat_Workspace representation for safe conversion.
	Multiplication Multiplication_Memory
}

// Rat_Parse_Rational_Memory_Invariants composes rational normalization state.
func Rat_Parse_Rational_Memory_Invariants(
	value Rat_Parse_Rational_Memory, namespace invariant.Namespace,
) {
	Rat_Operation_Integers_Invariants(value.Integers, namespace)
	Greatest_Common_Divisor_Memory_Invariants(value.Greatest_Common, namespace)
	Multiplication_Memory_Invariants(value.Multiplication, namespace)
}

// Rat_Parse_Fraction_Workspace owns component parsing and rational normalization.
type Rat_Parse_Fraction_Workspace struct {
	// Integers hold parsed components until complete success.
	Integers Rat_Parse_Fraction_Integers
	// Parse keeps both component conversions transactional.
	Parse Rat_Parse_Fraction_Integer_Workspaces
	// Rational owns normalization without destination mutation.
	Rational Rat_Parse_Rational_Memory
}

// Rat_Parse_Fraction_Workspace_Invariants composes complete fraction parse storage.
func Rat_Parse_Fraction_Workspace_Invariants(
	value *Rat_Parse_Fraction_Workspace, namespace invariant.Namespace,
) {
	Rat_Parse_Fraction_Integers_Invariants(value.Integers, namespace)
	Rat_Parse_Fraction_Integer_Workspaces_Invariants(value.Parse, namespace)
	Rat_Parse_Rational_Memory_Invariants(value.Rational, namespace)
}

// Rat_Parse_Workspace keeps both syntax branches on one identical caller-owned memory layout.
type Rat_Parse_Workspace Rat_Parse_Fraction_Workspace

// Rat_Parse_Workspace_Invariants preserves the fraction workspace contract across conversion.
func Rat_Parse_Workspace_Invariants(
	value *Rat_Parse_Workspace, namespace invariant.Namespace,
) {
	invariant.Always(
		len(value.Integers) == RAT_PARSE_FRACTION_INTEGER_COUNT,
		"Complete rational parse has fixed component capacity.",
	)
	invariant.Always(
		len(value.Parse) == RAT_PARSE_WORKSPACE_COUNT,
		"Complete rational parse has fixed integer parse capacity.",
	)
	Rat_Parse_Rational_Memory_Invariants(value.Rational, namespace)
}

// RAT_PARSE_WORKSPACE_REFERENCE_INDEX keeps validated parse memory behind one address.
const RAT_PARSE_WORKSPACE_REFERENCE_INDEX = 0

// RAT_PARSE_WORKSPACE_REFERENCE_COUNT fixes the internal boundary to that one address.
const RAT_PARSE_WORKSPACE_REFERENCE_COUNT = RAT_PARSE_WORKSPACE_REFERENCE_INDEX + 1

// Rat_Parse_Workspace_References carries validated parse memory without copying its fields.
type Rat_Parse_Workspace_References [RAT_PARSE_WORKSPACE_REFERENCE_COUNT]*Rat_Parse_Workspace

// Rat_Parse_Workspace_References_Invariants rejects an absent validated workspace.
func Rat_Parse_Workspace_References_Invariants(
	value *Rat_Parse_Workspace_References, _ invariant.Namespace,
) {
	invariant.Always(
		len(value) == RAT_PARSE_WORKSPACE_REFERENCE_COUNT,
		"A rational parse workspace reference has fixed capacity.",
	)
	invariant.Always(
		value[RAT_PARSE_WORKSPACE_REFERENCE_INDEX] != nil,
		"The rational parse workspace exists.",
	)
}

// RAT_PARSE_FRACTION_REFERENCE_INDEX keeps validated fraction memory by address.
const RAT_PARSE_FRACTION_REFERENCE_INDEX = 0

// RAT_PARSE_FRACTION_REFERENCE_COUNT binds the one fraction workspace address.
const RAT_PARSE_FRACTION_REFERENCE_COUNT = RAT_PARSE_FRACTION_REFERENCE_INDEX + 1

// Rat_Parse_Fraction_References carries validated fraction parse memory by address.
type Rat_Parse_Fraction_References [RAT_PARSE_FRACTION_REFERENCE_COUNT]*Rat_Parse_Fraction_Workspace

// Rat_Parse_Fraction_References_Invariants rejects absent fraction memory.
func Rat_Parse_Fraction_References_Invariants(
	value *Rat_Parse_Fraction_References, _ invariant.Namespace,
) {
	invariant.Always(
		len(value) == RAT_PARSE_FRACTION_REFERENCE_COUNT,
		"A fraction parse workspace reference has fixed capacity.",
	)
	invariant.Always(
		value[RAT_PARSE_FRACTION_REFERENCE_INDEX] != nil,
		"The fraction parse workspace exists.",
	)
}

// Int_Parse_Words owns one transactional magnitude under construction.
type Int_Parse_Words [WORD_COUNT_MAXIMUM]Word

// Int_Parse_Words_Invariants fixes one complete parsed magnitude capacity.
func Int_Parse_Words_Invariants(value Int_Parse_Words, _ invariant.Namespace) {
	invariant.Always(
		len(value) == WORD_COUNT_MAXIMUM,
		"Integer parse word storage has fixed magnitude capacity.",
	)
}

// Int_Parse_Workspace owns parse state so destination changes only after complete success.
type Int_Parse_Workspace struct {
	// Words holds magnitude until every source byte passes validation.
	Words Int_Parse_Words
}

// Int_Parse_Workspace_Invariants binds scratch storage to one Int magnitude.
func Int_Parse_Workspace_Invariants(value *Int_Parse_Workspace, namespace invariant.Namespace) {
	Int_Parse_Words_Invariants(value.Words, namespace)
}

// Rat_Integers stores numerator and denominator without repeated invariant leaf types.
type Rat_Integers [RAT_COMPONENT_COUNT]Int

// Rat_Integers_Invariants fixes complete rational component capacity.
func Rat_Integers_Invariants(value Rat_Integers, _ invariant.Namespace) {
	invariant.Always(
		len(value) == RAT_COMPONENT_COUNT,
		"Rational integer storage has fixed component capacity.",
	)
}

// Rat stores a reduced numerator and denominator; zero denominator storage represents one.
type Rat struct {
	// Integers keep both components inline and caller-owned.
	Integers Rat_Integers
}

// Rat_Invariants bounds both components and keeps denominator positive.
func Rat_Invariants(value *Rat, namespace invariant.Namespace) {
	Rat_Integers_Invariants(value.Integers, namespace)
	numerator := &value.Integers[RAT_NUMERATOR_INDEX]
	denominator := &value.Integers[RAT_DENOMINATOR_INDEX]
	invariant.Always(
		uint(numerator.Count) <= uint(RAT_WORD_COUNT_MAXIMUM),
		"A rational numerator stays inside its component bound.",
	)
	invariant.Always(
		int(numerator.Negative) <= int(numerator.Count),
		"A rational numerator gives zero no negative twin.",
	)
	invariant.Always(
		uint(denominator.Count) <= uint(RAT_WORD_COUNT_MAXIMUM),
		"A rational denominator stays inside its component bound.",
	)
	invariant.Always(
		denominator.Negative == POLARITY_NONNEGATIVE,
		"A rational denominator keeps sign in its numerator.",
	)
}

// Rat_Gob_Values owns transactional rational state without nested field invariant identity.
type Rat_Gob_Values [RAT_GOB_VALUE_COUNT]Rat

// Rat_Gob_Values_Invariants fixes one decoded rational capacity.
func Rat_Gob_Values_Invariants(value Rat_Gob_Values, _ invariant.Namespace) {
	invariant.Always(
		len(value) == RAT_GOB_VALUE_COUNT,
		"Rational gob storage owns one transactional decoded value.",
	)
}

// Rat_Gob_Workspace owns decoded components until complete validation succeeds.
type Rat_Gob_Workspace struct {
	// Values keep hostile input away from caller destination until commit.
	Values Rat_Gob_Values
}

// Rat_Gob_Workspace_Invariants binds transactional decode state to one rational.
func Rat_Gob_Workspace_Invariants(value *Rat_Gob_Workspace, namespace invariant.Namespace) {
	Rat_Gob_Values_Invariants(value.Values, namespace)
}

// Rat_Operation_Integers owns all binary arithmetic temporaries.
type Rat_Operation_Integers [RAT_OPERATION_INTEGER_COUNT]Int

// Rat_Operation_Integers_Invariants fixes complete rational arithmetic capacity.
func Rat_Operation_Integers_Invariants(value Rat_Operation_Integers, _ invariant.Namespace) {
	invariant.Always(
		len(value) == RAT_OPERATION_INTEGER_COUNT,
		"Rational operation storage has fixed integer capacity.",
	)
}

// Multiplication_Memory is reusable product scratch for composite algorithms.
type Multiplication_Memory struct {
	// Product keeps multiplication output outside algorithm state until success.
	Product [WORD_COUNT_MAXIMUM]Word
}

// Multiplication_Memory_Invariants fixes one complete product bound.
func Multiplication_Memory_Invariants(value Multiplication_Memory, _ invariant.Namespace) {
	invariant.Always(
		len(value.Product) == WORD_COUNT_MAXIMUM,
		"Multiplication memory owns one complete product bound.",
	)
}

// Greatest_Common_Divisor_Memory is reusable Euclidean scratch for composite algorithms.
type Greatest_Common_Divisor_Memory struct {
	// Integers rotate dividend, divisor, and remainder.
	Integers Euclidean_Integers
	// Division owns restoring-division scratch.
	Division Division_Memory
}

// Greatest_Common_Divisor_Memory_Invariants composes Euclidean integer and division storage.
func Greatest_Common_Divisor_Memory_Invariants(
	value Greatest_Common_Divisor_Memory, namespace invariant.Namespace,
) {
	Euclidean_Integers_Invariants(value.Integers, namespace)
	Division_Memory_Invariants(value.Division, namespace)
}

// Rat_Workspace owns normalization and cross-product storage outside caller values.
type Rat_Workspace struct {
	// Integers hold result components and both decoded operands.
	Integers Rat_Operation_Integers
	// Greatest_Common owns normalization reduction state.
	Greatest_Common Greatest_Common_Divisor_Memory
	// Multiplication owns every cross-product temporary.
	Multiplication Multiplication_Memory
}

// Rat_Workspace_Invariants composes every fixed rational scratch store.
func Rat_Workspace_Invariants(value *Rat_Workspace, namespace invariant.Namespace) {
	Rat_Operation_Integers_Invariants(value.Integers, namespace)
	Greatest_Common_Divisor_Memory_Invariants(value.Greatest_Common, namespace)
	Multiplication_Memory_Invariants(value.Multiplication, namespace)
}

// Float_Set_Precision applies one validated bound and rounds the stored value in place.
func Float_Set_Precision(
	destination *Float, value Float_Precision_Unvalidated,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "float_set_precision.status") }()
	Float_Invariants(destination, "float_set_precision.destination_initial")
	Float_Precision_Unvalidated_Invariants(value, "float_set_precision.value")
	if value > FLOAT_PRECISION_MAXIMUM {
		return STATUS_INPUT_INVALID
	}
	precision := Float_Precision(value)
	if destination.Precision == precision {
		destination.Accuracy = ACCURACY_EXACT
		return STATUS_OK
	}
	float_round(destination, precision)
	return STATUS_OK
}

// Float_Set_Rounding_Mode validates policy before changing later rounding behavior.
func Float_Set_Rounding_Mode(
	destination *Float, value Rounding_Mode_Unvalidated,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "float_set_rounding_mode.status") }()
	Float_Invariants(destination, "float_set_rounding_mode.destination_initial")
	Rounding_Mode_Unvalidated_Invariants(value, "float_set_rounding_mode.value")
	if value > ROUND_TO_POSITIVE_INFINITY {
		return STATUS_INPUT_INVALID
	}
	destination.Mode = Rounding_Mode(value)
	destination.Accuracy = ACCURACY_EXACT
	return STATUS_OK
}

// Float_Precision_Of reports the destination policy even for zero and infinity.
func Float_Precision_Of(value *Float) (precision Float_Precision) {
	defer func() { Float_Precision_Invariants(precision, "float_precision_of.precision") }()
	Float_Invariants(value, "float_precision_of.value")
	return value.Precision
}

// Float_Minimum_Precision reports significant bits needed for exact finite storage.
func Float_Minimum_Precision(value *Float) (precision Float_Precision) {
	defer func() {
		Float_Precision_Invariants(precision, "float_minimum_precision.precision")
	}()
	Float_Invariants(value, "float_minimum_precision.value")
	if value.Form != FLOAT_FORM_FINITE {
		return FLOAT_PRECISION_MINIMUM
	}
	word_count := int(value.Mantissa.Count)
	high := value.Mantissa.Words[word_count-WORD_COUNT_INCREMENT]
	bit_count := (word_count-WORD_COUNT_INCREMENT)*WORD_BIT_COUNT +
		WORD_BIT_COUNT - int(bits.Leading_Zeros_64(bits.Word_64(high)))
	zero_count := BIT_COUNT_MINIMUM
	for value.Mantissa.Words[zero_count/WORD_BIT_COUNT] == 0 {
		zero_count += WORD_BIT_COUNT
	}
	zero_count += int(bits.Trailing_Zeros_64(
		bits.Word_64(value.Mantissa.Words[zero_count/WORD_BIT_COUNT]),
	))
	return Float_Precision(bit_count - zero_count)
}

// Float_Rounding_Mode reports the policy used by later destination writes.
func Float_Rounding_Mode(value *Float) (mode Rounding_Mode) {
	defer func() { Rounding_Mode_Invariants(mode, "float_rounding_mode.mode") }()
	Float_Invariants(value, "float_rounding_mode.value")
	return value.Mode
}

// Float_Accuracy reports the latest rounding relation.
func Float_Accuracy(value *Float) (accuracy Accuracy) {
	defer func() { Accuracy_Invariants(accuracy, "float_accuracy.accuracy") }()
	Float_Invariants(value, "float_accuracy.value")
	return value.Accuracy
}

// Float_Sign reports mathematical sign while merging both stored zeros.
func Float_Sign(value *Float) (sign Sign) {
	defer func() { Sign_Invariants(sign, "float_sign.sign") }()
	Float_Invariants(value, "float_sign.value")
	if value.Form == FLOAT_FORM_ZERO {
		return SIGN_ZERO
	}
	if value.Negative == POLARITY_NEGATIVE {
		return SIGN_NEGATIVE
	}
	return SIGN_POSITIVE
}

// Float_Sign_Bit preserves negative zero and negative infinity.
func Float_Sign_Bit(value *Float) (negative Boolean) {
	defer func() { Boolean_Invariants(negative, "float_sign_bit.negative") }()
	Float_Invariants(value, "float_sign_bit.value")
	return Boolean(value.Negative == POLARITY_NEGATIVE)
}

// Float_Is_Infinite distinguishes finite values and signed zeros from infinities.
func Float_Is_Infinite(value *Float) (infinite Boolean) {
	defer func() { Boolean_Invariants(infinite, "float_is_infinite.infinite") }()
	Float_Invariants(value, "float_is_infinite.value")
	return Boolean(value.Form == FLOAT_FORM_INFINITY)
}

// Float_Is_Integer checks discarded binary places without conversion storage.
func Float_Is_Integer(value *Float) (integer Boolean) {
	defer func() { Boolean_Invariants(integer, "float_is_integer.integer") }()
	Float_Invariants(value, "float_is_integer.value")
	if value.Form != FLOAT_FORM_FINITE {
		return Boolean(value.Form == FLOAT_FORM_ZERO)
	}
	if value.Exponent <= 0 {
		return false
	}
	word_count := int(value.Mantissa.Count)
	if int(value.Exponent) >= word_count*WORD_BIT_COUNT {
		return true
	}
	high := value.Mantissa.Words[word_count-WORD_COUNT_INCREMENT]
	bit_count := (word_count-WORD_COUNT_INCREMENT)*WORD_BIT_COUNT +
		WORD_BIT_COUNT - int(bits.Leading_Zeros_64(bits.Word_64(high)))
	if int(value.Exponent) >= bit_count {
		return true
	}
	discard_count := bit_count - int(value.Exponent)
	discard_word_count := discard_count / WORD_BIT_COUNT
	for word_index := WORD_COUNT_MINIMUM; word_index < discard_word_count; word_index++ {
		if value.Mantissa.Words[word_index] != 0 {
			return false
		}
	}
	partial_count := discard_count % WORD_BIT_COUNT
	if partial_count == BIT_COUNT_MINIMUM {
		return true
	}
	mask := Word(bits.WORD_64_MAXIMUM) >> uint(WORD_BIT_COUNT-partial_count)
	return Boolean(value.Mantissa.Words[discard_word_count]&mask == 0)
}

// Float_Subtract keeps exact native integers off the full alignment workspace.
func Float_Subtract(
	destination *Float, left *Float, right *Float, workspace *Float_Addition_Workspace,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "float_subtract.status") }()
	Float_Invariants(destination, "float_subtract.destination_initial")
	Float_Invariants(left, "float_subtract.left")
	Float_Invariants(right, "float_subtract.right")
	Float_Addition_Workspace_Invariants(workspace, "float_subtract.workspace")
	if left.Form != FLOAT_FORM_FINITE {
		return float_subtract_general(destination, left, right, workspace)
	}
	if right.Form != FLOAT_FORM_FINITE {
		return float_subtract_general(destination, left, right, workspace)
	}
	if left.Negative != POLARITY_NONNEGATIVE {
		return float_subtract_general(destination, left, right, workspace)
	}
	if right.Negative != POLARITY_NONNEGATIVE {
		return float_subtract_general(destination, left, right, workspace)
	}
	if left.Mantissa.Count == Word_Count(BASE_BINARY) {
		if right.Mantissa.Count == Word_Count(BASE_BINARY) {
			references := Float_References{destination, left, right}
			if float_subtract_exact_double_word(&references) {
				return STATUS_OK
			}
		}
	}
	if left.Mantissa.Count != Word_Count(WORD_COUNT_INCREMENT) {
		return float_subtract_general(destination, left, right, workspace)
	}
	if right.Mantissa.Count != Word_Count(WORD_COUNT_INCREMENT) {
		return float_subtract_general(destination, left, right, workspace)
	}
	if left.Exponent != right.Exponent {
		return float_subtract_general(destination, left, right, workspace)
	}
	exponent := int(left.Exponent)
	if uint(exponent-WORD_COUNT_INCREMENT) >= uint(WORD_BIT_INDEX_MAXIMUM) {
		return float_subtract_general(destination, left, right, workspace)
	}
	minimum := Word(bits.CARRY_MAXIMUM) << uint(exponent-WORD_COUNT_INCREMENT)
	left_word := left.Mantissa.Words[WORD_COUNT_MINIMUM]
	right_word := right.Mantissa.Words[WORD_COUNT_MINIMUM]
	if left_word&right_word&minimum == 0 {
		return float_subtract_general(destination, left, right, workspace)
	}
	if left_word|right_word >= minimum<<WORD_COUNT_INCREMENT {
		return float_subtract_general(destination, left, right, workspace)
	}
	if left_word <= right_word {
		return float_subtract_general(destination, left, right, workspace)
	}
	precision := float_add_precision(
		destination.Precision, left.Precision, right.Precision,
	)
	difference := left_word - right_word
	mantissa := &destination.Mantissa
	previous_count := mantissa.Count
	destination.Precision, destination.Accuracy = precision, ACCURACY_EXACT
	destination.Form, destination.Negative = FLOAT_FORM_FINITE, POLARITY_NONNEGATIVE
	mantissa.Words[WORD_COUNT_MINIMUM] = difference
	mantissa.Count, mantissa.Negative = Word_Count(WORD_COUNT_INCREMENT), POLARITY_NONNEGATIVE
	destination.Exponent = Float_Exponent(bits.Bit_Size_64(bits.Word_64(difference)))
	int_clear((*Int)(mantissa), mantissa.Count, previous_count)
	return STATUS_OK
}

// Float_Multiply keeps the full product in caller storage before one destination rounding.
func Float_Multiply(
	destination *Float, left *Float, right *Float,
	workspace *Float_Multiplication_Workspace,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "float_multiply.status") }()
	Float_Invariants(destination, "float_multiply.destination_initial")
	Float_Invariants(left, "float_multiply.left")
	Float_Invariants(right, "float_multiply.right")
	Float_Multiplication_Workspace_Invariants(workspace, "float_multiply.workspace")
	precision := destination.Precision
	if precision == FLOAT_PRECISION_MINIMUM {
		precision = left.Precision
		if right.Precision > precision {
			precision = right.Precision
		}
	}
	if left.Form != FLOAT_FORM_FINITE {
		return float_multiply_general(destination, left, right, workspace, precision)
	}
	if right.Form != FLOAT_FORM_FINITE {
		return float_multiply_general(destination, left, right, workspace, precision)
	}
	references := Float_References{destination, left, right}
	if left.Mantissa.Count == Word_Count(WORD_COUNT_INCREMENT) {
		if right.Mantissa.Count == Word_Count(WORD_COUNT_INCREMENT) {
			if float_multiply_exact_word(&references) {
				return STATUS_OK
			}
		}
	}
	negative := Polarity(uint8(left.Negative) ^ uint8(right.Negative))
	float_multiply_finite(
		destination, (*Float_Finite)(left), (*Float_Finite)(right), workspace,
		Float_Active_Precision(precision), destination.Mode, negative,
	)
	return STATUS_OK
}

// Float_Set_Int stores one bounded integer before applying destination precision.
func Float_Set_Int(destination *Float, value *Int) {
	Float_Invariants(destination, "float_set_int.destination_initial")
	Int_Invariants(value, "float_set_int.value")
	bit_count := Int_Bit_Count(value)
	precision := destination.Precision
	if precision == FLOAT_PRECISION_MINIMUM {
		precision = WORD_BIT_COUNT
		if int(bit_count) > WORD_BIT_COUNT {
			precision = Float_Precision(bit_count)
		}
	}
	if value.Count > WORD_COUNT_MINIMUM {
		if Float_Precision(bit_count) <= precision {
			previous_count := destination.Mantissa.Count
			for index := Word_Count(WORD_COUNT_MINIMUM); index < value.Count; index++ {
				destination.Mantissa.Words[index] = value.Words[index]
			}
			destination.Precision, destination.Accuracy = precision, ACCURACY_EXACT
			destination.Form, destination.Negative = FLOAT_FORM_FINITE, value.Negative
			destination.Mantissa.Count = value.Count
			destination.Mantissa.Negative = POLARITY_NONNEGATIVE
			destination.Exponent = Float_Exponent(bit_count)
			if previous_count > destination.Mantissa.Count {
				int_clear(
					(*Int)(&destination.Mantissa), destination.Mantissa.Count,
					previous_count,
				)
			}
			return
		}
	}
	magnitude := *value
	magnitude.Negative = POLARITY_NONNEGATIVE
	float_magnitude := Float_Mantissa(magnitude)
	float_set_magnitude(
		destination, &float_magnitude, Float_Exponent(bit_count), value.Negative,
		Float_Active_Precision(precision), destination.Mode,
	)
}

// Float_Set_Float_64_Bits decodes every numeric IEEE binary64 value transactionally.
func Float_Set_Float_64_Bits(
	destination *Float, encoding Float_64_Bits,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "float_set_float_64_bits.status") }()
	Float_Invariants(destination, "float_set_float_64_bits.destination_initial")
	Float_64_Bits_Invariants(encoding, "float_set_float_64_bits.encoding")
	encoded := uint64(encoding)
	exponent_field := encoded >> FLOAT_64_EXPONENT_SHIFT & FLOAT_64_EXPONENT_MASK
	mantissa_field := encoded & FLOAT_64_MANTISSA_MASK
	if exponent_field == FLOAT_64_EXPONENT_MASK {
		if mantissa_field != 0 {
			return STATUS_INPUT_INVALID
		}
	}
	negative := Polarity((encoded >> FLOAT_64_SIGN_SHIFT) & uint64(bits.CARRY_MAXIMUM))
	precision := destination.Precision
	if precision == FLOAT_PRECISION_MINIMUM {
		precision = FLOAT_64_VALUE_MANTISSA_BIT_COUNT
	}
	if exponent_field == FLOAT_64_EXPONENT_MASK {
		float_set_nonfinite(
			destination, Float_Nonfinite_Form(FLOAT_FORM_INFINITY), negative, precision,
		)
		return STATUS_OK
	}
	if exponent_field == 0 {
		if mantissa_field == 0 {
			float_set_nonfinite(
				destination, Float_Nonfinite_Form(FLOAT_FORM_ZERO), negative,
				precision,
			)
			return STATUS_OK
		}
	}
	float_set_float_64_finite(
		destination, Float_64_Finite_Exponent_Field(exponent_field),
		Float_64_Mantissa_Field(mantissa_field), negative,
		Float_Active_Precision(precision),
	)
	return STATUS_OK
}

// Float_Set_Infinity stores signed infinity while preserving destination precision and mode.
func Float_Set_Infinity(destination *Float, negative Boolean) {
	Float_Invariants(destination, "float_set_infinity.destination_initial")
	Boolean_Invariants(negative, "float_set_infinity.negative")
	polarity := POLARITY_NONNEGATIVE
	if negative {
		polarity = POLARITY_NEGATIVE
	}
	float_set_nonfinite(
		destination, Float_Nonfinite_Form(FLOAT_FORM_INFINITY), polarity,
		destination.Precision,
	)
}

// Float_Copy copies numeric value and every policy field without rounding.
func Float_Copy(destination *Float, source *Float) {
	Float_Invariants(destination, "float_copy.destination_initial")
	Float_Invariants(source, "float_copy.source")
	float_copy_exact(destination, source)
}

// Float_Mantissa_Exponent separates one finite normalized exponent without allocation.
func Float_Mantissa_Exponent(value *Float, mantissa *Float) (exponent Float_Exponent) {
	defer func() {
		Float_Exponent_Invariants(exponent, "float_mantissa_exponent.exponent")
	}()
	Float_Invariants(value, "float_mantissa_exponent.value")
	Float_Invariants(mantissa, "float_mantissa_exponent.mantissa_initial")
	exponent = value.Exponent
	Float_Copy(mantissa, value)
	if mantissa.Form == FLOAT_FORM_FINITE {
		mantissa.Exponent = 0
	}
	return exponent
}

// Float_Set_Mantissa_Exponent restores one bounded normalized binary scale transactionally.
func Float_Set_Mantissa_Exponent(
	destination *Float, mantissa *Float, value Float_Exponent_Unvalidated,
) (status Validation_Status) {
	defer func() {
		Validation_Status_Invariants(status, "float_set_mantissa_exponent.status")
	}()
	Float_Invariants(destination, "float_set_mantissa_exponent.destination_initial")
	Float_Invariants(mantissa, "float_set_mantissa_exponent.mantissa")
	Float_Exponent_Unvalidated_Invariants(value, "float_set_mantissa_exponent.value")
	exponent, status := Float_Exponent_Validate(value)
	if status != Validation_Status(STATUS_OK) {
		return status
	}
	if mantissa.Form != FLOAT_FORM_FINITE {
		Float_Copy(destination, mantissa)
		return STATUS_OK
	}
	combined := int(mantissa.Exponent) + int(exponent)
	Float_Copy(destination, mantissa)
	if combined < FLOAT_EXPONENT_MINIMUM {
		float_set_nonfinite(
			destination, Float_Nonfinite_Form(FLOAT_FORM_ZERO), mantissa.Negative,
			mantissa.Precision,
		)
		destination.Accuracy = Accuracy(float_accuracy(false, mantissa.Negative))
		return STATUS_OK
	}
	if combined > FLOAT_EXPONENT_MAXIMUM {
		float_set_nonfinite(
			destination, Float_Nonfinite_Form(FLOAT_FORM_INFINITY), mantissa.Negative,
			mantissa.Precision,
		)
		destination.Accuracy = Accuracy(float_accuracy(true, mantissa.Negative))
		return STATUS_OK
	}
	destination.Exponent = Float_Exponent(combined)
	return STATUS_OK
}

// Float_Float_64_Bits rounds one stored value to nearest IEEE binary64 encoding.
func Float_Float_64_Bits(value *Float) (
	encoding Float_64_Value_Bits, accuracy Accuracy,
) {
	defer func() {
		Float_64_Value_Bits_Invariants(encoding, "float_float_64_bits.encoding")
		Accuracy_Invariants(accuracy, "float_float_64_bits.accuracy")
	}()
	Float_Invariants(value, "float_float_64_bits.value")
	sign := uint64(0)
	if value.Negative == POLARITY_NEGATIVE {
		sign = FLOAT_64_SIGN_MASK
	}
	if value.Form == FLOAT_FORM_ZERO {
		return Float_64_Value_Bits(sign), ACCURACY_EXACT
	}
	if value.Form == FLOAT_FORM_INFINITY {
		return Float_64_Value_Bits(sign | FLOAT_64_POSITIVE_INFINITY_BITS), ACCURACY_EXACT
	}
	return float_finite_float_64_bits(
		(*Float_Finite)(value), Float_64_Sign(sign),
	)
}

// Int_Set_Int_64 avoids allocating constructors by writing caller placement.
func Int_Set_Int_64(destination *Int, value Int_64) {
	Int_Invariants(destination, "int_set_int_64.destination")
	Int_64_Invariants(value, "int_set_int_64.value")
	magnitude := uint64(value)
	negative := POLARITY_NONNEGATIVE
	if value < 0 {
		negative = POLARITY_NEGATIVE
		magnitude = uint64(-(value + 1)) + 1
	}
	int_set_word(destination, Word(magnitude), negative)
}

// Int_Set_Uint_64 writes caller placement so lifting one word cannot allocate.
func Int_Set_Uint_64(destination *Int, value Word_64) {
	Int_Invariants(destination, "int_set_uint_64.destination")
	Word_64_Invariants(value, "int_set_uint_64.value")
	int_set_word(destination, Word(value), POLARITY_NONNEGATIVE)
}

// Int_Set_Bytes rejects hostile byte length before destination mutation.
func Int_Set_Bytes(
	destination *Int, source Bytes_Unvalidated,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "int_set_bytes.status") }()
	Int_Invariants(destination, "int_set_bytes.destination")
	Bytes_Unvalidated_Invariants(source, "int_set_bytes.source")
	validated, status := Bytes_Validate(source)
	if status != Validation_Status(STATUS_OK) {
		return status
	}
	int_set_bytes_validated(destination, validated)
	return STATUS_OK
}

func int_set_bytes_validated(destination *Int, source Bytes) {
	Int_Invariants(destination, "int_set_bytes_validated.destination")
	Bytes_Invariants(source, "int_set_bytes_validated.source")
	first := 0
	for first < len(source) {
		if source[first] != 0 {
			break
		}
		first++
	}
	if first == len(source) {
		int_zero(destination, destination.Count)
		return
	}
	byte_count := len(source) - first
	word_count := (byte_count + WORD_BYTE_COUNT - 1) / WORD_BYTE_COUNT
	previous_count := destination.Count
	for word_index := WORD_COUNT_MINIMUM; word_index < word_count; word_index++ {
		word := Word(0)
		source_end := len(source) - word_index*WORD_BYTE_COUNT
		source_index := source_end - WORD_BYTE_COUNT
		if source_index >= first {
			word = Word(source[source_index])
			source_index++
			word = word<<bits.BIT_COUNT_8_MAXIMUM | Word(source[source_index])
			source_index++
			word = word<<bits.BIT_COUNT_8_MAXIMUM | Word(source[source_index])
			source_index++
			word = word<<bits.BIT_COUNT_8_MAXIMUM | Word(source[source_index])
			source_index++
			word = word<<bits.BIT_COUNT_8_MAXIMUM | Word(source[source_index])
			source_index++
			word = word<<bits.BIT_COUNT_8_MAXIMUM | Word(source[source_index])
			source_index++
			word = word<<bits.BIT_COUNT_8_MAXIMUM | Word(source[source_index])
			source_index++
			word = word<<bits.BIT_COUNT_8_MAXIMUM | Word(source[source_index])
		} else {
			source_index = first
			for source_index < source_end {
				word <<= bits.BIT_COUNT_8_MAXIMUM
				word |= Word(source[source_index])
				source_index++
			}
		}
		destination.Words[word_index] = word
	}
	destination.Count = Word_Count(word_count)
	destination.Negative = POLARITY_NONNEGATIVE
	int_clear(destination, destination.Count, previous_count)
}

// Int_Set_Words copies and normalizes caller little-endian magnitude transactionally.
func Int_Set_Words(
	destination *Int, source Words_Unvalidated,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "int_set_words.status") }()
	Int_Invariants(destination, "int_set_words.destination")
	Words_Unvalidated_Invariants(source, "int_set_words.source")
	if len(source) > WORD_COUNT_MAXIMUM {
		return STATUS_INPUT_INVALID
	}
	count := len(source)
	for count > WORD_COUNT_MINIMUM {
		if source[count-WORD_COUNT_INCREMENT] != 0 {
			break
		}
		count--
	}
	previous_count := destination.Count
	for index := WORD_COUNT_MINIMUM; index < count; index++ {
		destination.Words[index] = source[index]
	}
	destination.Count = Word_Count(count)
	destination.Negative = POLARITY_NONNEGATIVE
	int_clear(destination, destination.Count, previous_count)
	return STATUS_OK
}

// Int_Words_Into copies magnitude only after caller storage can hold every significant word.
func Int_Words_Into(
	destination Words, value *Int,
) (count Word_Count, status Destination_Status) {
	defer func() {
		Word_Count_Invariants(count, "int_words_into.count")
		Destination_Status_Invariants(status, "int_words_into.status")
	}()
	Words_Invariants(destination, "int_words_into.destination")
	Int_Invariants(value, "int_words_into.value")
	if len(destination) < int(value.Count) {
		return 0, STATUS_DESTINATION_TOO_SMALL
	}
	for index := WORD_COUNT_MINIMUM; index < int(value.Count); index++ {
		destination[index] = value.Words[index]
	}
	return value.Count, STATUS_OK
}

// Int_Bytes_Into writes minimal big-endian magnitude at caller storage start.
func Int_Bytes_Into(
	destination Bytes, value *Int,
) (count Byte_Count, status Destination_Status) {
	defer func() {
		Byte_Count_Invariants(count, "int_bytes_into.count")
		Destination_Status_Invariants(status, "int_bytes_into.status")
	}()
	Bytes_Invariants(destination, "int_bytes_into.destination")
	Int_Invariants(value, "int_bytes_into.value")
	required := (int(Int_Bit_Count(value)) + bits.BIT_COUNT_8_MAXIMUM - 1) /
		bits.BIT_COUNT_8_MAXIMUM
	if len(destination) < required {
		return 0, STATUS_DESTINATION_TOO_SMALL
	}
	for byte_index := bytes.SLICE_SIZE_MINIMUM; byte_index < required; byte_index++ {
		word_index := byte_index / WORD_BYTE_COUNT
		word_shift := (byte_index % WORD_BYTE_COUNT) * bits.BIT_COUNT_8_MAXIMUM
		destination[required-1-byte_index] = byte(value.Words[word_index] >> word_shift)
	}
	return Byte_Count(required), STATUS_OK
}

// Int_Fill_Bytes writes magnitude at caller storage end after clearing every leading byte.
func Int_Fill_Bytes(destination Bytes, value *Int) (status Destination_Status) {
	defer func() { Destination_Status_Invariants(status, "int_fill_bytes.status") }()
	Bytes_Invariants(destination, "int_fill_bytes.destination")
	Int_Invariants(value, "int_fill_bytes.value")
	required := (int(Int_Bit_Count(value)) + bits.BIT_COUNT_8_MAXIMUM - 1) /
		bits.BIT_COUNT_8_MAXIMUM
	if len(destination) < required {
		return STATUS_DESTINATION_TOO_SMALL
	}
	for index := range destination {
		destination[index] = 0
	}
	for byte_index := bytes.SLICE_SIZE_MINIMUM; byte_index < required; byte_index++ {
		word_index := byte_index / WORD_BYTE_COUNT
		word_shift := (byte_index % WORD_BYTE_COUNT) * bits.BIT_COUNT_8_MAXIMUM
		destination_index := len(destination) - 1 - byte_index
		destination[destination_index] = byte(value.Words[word_index] >> word_shift)
	}
	return STATUS_OK
}

// Int_Gob_Encode_Into preserves stdlib wire compatibility without returning owned storage.
func Int_Gob_Encode_Into(
	destination Int_Encoding, value *Int,
) (count Int_Encoding_Count, status Destination_Status) {
	defer func() {
		Int_Encoding_Count_Invariants(count, "int_gob_encode_into.count")
		Destination_Status_Invariants(status, "int_gob_encode_into.status")
	}()
	Int_Encoding_Invariants(destination, "int_gob_encode_into.destination")
	Int_Invariants(value, "int_gob_encode_into.value")
	magnitude_size := (int(Int_Bit_Count(value)) + bits.BIT_COUNT_8_MAXIMUM - 1) /
		bits.BIT_COUNT_8_MAXIMUM
	required_size := INT_GOB_HEADER_SIZE + magnitude_size
	if len(destination) < required_size {
		return 0, STATUS_DESTINATION_TOO_SMALL
	}
	header := byte(INT_GOB_VERSION << INT_GOB_VERSION_SHIFT)
	if value.Negative == POLARITY_NEGATIVE {
		header |= INT_GOB_NEGATIVE_MASK
	}
	destination[bytes.SLICE_SIZE_MINIMUM] = header
	for byte_index := bytes.SLICE_SIZE_MINIMUM; byte_index < magnitude_size; byte_index++ {
		word_index := byte_index / WORD_BYTE_COUNT
		word_shift := (byte_index % WORD_BYTE_COUNT) * bits.BIT_COUNT_8_MAXIMUM
		destination[required_size-1-byte_index] =
			byte(value.Words[word_index] >> word_shift)
	}
	return Int_Encoding_Count(required_size), STATUS_OK
}

// Int_Gob_Decode rejects unsupported or oversized wire values before destination mutation.
func Int_Gob_Decode(
	destination *Int, source Int_Encoding_Unvalidated,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "int_gob_decode.status") }()
	Int_Invariants(destination, "int_gob_decode.destination")
	Int_Encoding_Unvalidated_Invariants(source, "int_gob_decode.source")
	if len(source) > INT_GOB_SIZE_MAXIMUM {
		return STATUS_INPUT_INVALID
	}
	if len(source) == bytes.SLICE_SIZE_MINIMUM {
		int_zero(destination, destination.Count)
		return STATUS_OK
	}
	header := source[bytes.SLICE_SIZE_MINIMUM]
	version := header >> INT_GOB_VERSION_SHIFT
	if version != INT_GOB_VERSION {
		return STATUS_INPUT_INVALID
	}
	int_set_bytes_validated(destination, Bytes(source[INT_GOB_HEADER_SIZE:]))
	if header&INT_GOB_NEGATIVE_MASK != 0 {
		if destination.Count != WORD_COUNT_MINIMUM {
			destination.Negative = POLARITY_NEGATIVE
		}
	}
	return STATUS_OK
}

// Rat_Gob_Encode_Into preserves stdlib wire bytes through caller-owned storage.
func Rat_Gob_Encode_Into(
	destination Rat_Encoding, value *Rat,
) (count Rat_Encoding_Count, status Destination_Status) {
	defer func() {
		Rat_Encoding_Count_Invariants(count, "rat_gob_encode_into.count")
		Destination_Status_Invariants(status, "rat_gob_encode_into.status")
	}()
	Rat_Encoding_Invariants(destination, "rat_gob_encode_into.destination")
	Rat_Invariants(value, "rat_gob_encode_into.value")
	numerator := &value.Integers[RAT_NUMERATOR_INDEX]
	denominator := &value.Integers[RAT_DENOMINATOR_INDEX]
	numerator_size := (int(Int_Bit_Count(numerator)) + bits.BIT_COUNT_8_MAXIMUM - 1) /
		bits.BIT_COUNT_8_MAXIMUM
	denominator_size := (int(Int_Bit_Count(denominator)) + bits.BIT_COUNT_8_MAXIMUM - 1) /
		bits.BIT_COUNT_8_MAXIMUM
	required_size := RAT_GOB_PREFIX_SIZE + numerator_size + denominator_size
	if len(destination) < required_size {
		return Rat_Encoding_Count(required_size), STATUS_DESTINATION_TOO_SMALL
	}
	header := byte(RAT_GOB_VERSION << RAT_GOB_VERSION_SHIFT)
	if numerator.Negative == POLARITY_NEGATIVE {
		header |= RAT_GOB_NEGATIVE_MASK
	}
	destination[bytes.SLICE_SIZE_MINIMUM] = header
	encoded_size := uint32(numerator_size)
	position := RAT_GOB_NUMERATOR_SIZE_OFFSET
	remaining_size := RAT_GOB_NUMERATOR_SIZE_FIELD_SIZE
	for remaining_size > bytes.SLICE_SIZE_MINIMUM {
		shift := (remaining_size - 1) * bits.BIT_COUNT_8_MAXIMUM
		destination[position] = byte(encoded_size >> shift)
		position++
		remaining_size--
	}
	numerator_end_size := RAT_GOB_PREFIX_SIZE + numerator_size
	_, numerator_status := Int_Bytes_Into(
		Bytes(destination[RAT_GOB_PREFIX_SIZE:numerator_end_size]), numerator,
	)
	_, denominator_status := Int_Bytes_Into(
		Bytes(destination[numerator_end_size:required_size]), denominator,
	)
	invariant.Always(
		numerator_status == Destination_Status(STATUS_OK),
		"Derived rational numerator storage has exact capacity.",
	)
	invariant.Always(
		denominator_status == Destination_Status(STATUS_OK),
		"Derived rational denominator storage has exact capacity.",
	)
	return Rat_Encoding_Count(required_size), STATUS_OK
}

// Rat_Gob_Decode validates wire boundaries before committing decoded components.
func Rat_Gob_Decode(
	destination *Rat,
	source Rat_Encoding_Unvalidated,
	workspace *Rat_Gob_Workspace,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "rat_gob_decode.status") }()
	Rat_Invariants(destination, "rat_gob_decode.destination")
	Rat_Encoding_Unvalidated_Invariants(source, "rat_gob_decode.source")
	Rat_Gob_Workspace_Invariants(workspace, "rat_gob_decode.workspace")
	if len(source) == bytes.SLICE_SIZE_MINIMUM {
		*destination = Rat{}
		return STATUS_OK
	}
	if len(source) < RAT_GOB_PREFIX_SIZE {
		return STATUS_INPUT_INVALID
	}
	if len(source) > RAT_GOB_SIZE_MAXIMUM {
		return STATUS_INPUT_INVALID
	}
	header := source[bytes.SLICE_SIZE_MINIMUM]
	if header>>RAT_GOB_VERSION_SHIFT != RAT_GOB_VERSION {
		return STATUS_INPUT_INVALID
	}
	encoded_size := uint32(0)
	position := RAT_GOB_NUMERATOR_SIZE_OFFSET
	for position < RAT_GOB_PREFIX_SIZE {
		encoded_size <<= bits.BIT_COUNT_8_MAXIMUM
		encoded_size |= uint32(source[position])
		position++
	}
	numerator_size := Rat_Component_Byte_Count(encoded_size)
	if numerator_size > RAT_COMPONENT_BYTE_SIZE_MAXIMUM {
		return STATUS_INPUT_INVALID
	}
	denominator_size := len(source) - RAT_GOB_PREFIX_SIZE - int(numerator_size)
	if denominator_size < bytes.SLICE_SIZE_MINIMUM {
		return STATUS_INPUT_INVALID
	}
	if denominator_size > RAT_COMPONENT_BYTE_SIZE_MAXIMUM {
		return STATUS_INPUT_INVALID
	}
	rat_gob_decode_commit(
		destination, Rat_Gob_Encoding(source), numerator_size,
	)
	return STATUS_OK
}

func rat_gob_decode_commit(
	destination *Rat,
	source Rat_Gob_Encoding,
	numerator_size Rat_Component_Byte_Count,
) {
	Rat_Invariants(destination, "rat_gob_decode_commit.destination")
	Rat_Gob_Encoding_Invariants(source, "rat_gob_decode_commit.source")
	Rat_Component_Byte_Count_Invariants(
		numerator_size, "rat_gob_decode_commit.numerator_size",
	)
	numerator := &destination.Integers[RAT_NUMERATOR_INDEX]
	denominator := &destination.Integers[RAT_DENOMINATOR_INDEX]
	numerator_end_size := RAT_GOB_PREFIX_SIZE + int(numerator_size)
	int_set_bytes_validated(
		numerator, Bytes(source[RAT_GOB_PREFIX_SIZE:numerator_end_size]),
	)
	int_set_bytes_validated(
		denominator, Bytes(source[numerator_end_size:]),
	)
	header := source[bytes.SLICE_SIZE_MINIMUM]
	if header&RAT_GOB_NEGATIVE_MASK != 0 {
		if numerator.Count > WORD_COUNT_MINIMUM {
			numerator.Negative = POLARITY_NEGATIVE
		}
	}
}

// Int_Text_Into converts through caller scratch before touching destination bytes.
func Int_Text_Into(
	destination Text, value *Int, base Base, workspace *Int_Text_Workspace,
) (count Text_Count, status Destination_Status) {
	defer func() {
		Text_Count_Invariants(count, "int_text_into.count")
		Destination_Status_Invariants(status, "int_text_into.status")
	}()
	Text_Invariants(destination, "int_text_into.destination")
	Int_Invariants(value, "int_text_into.value")
	Base_Invariants(base, "int_text_into.base")
	Int_Text_Workspace_Invariants(workspace, "int_text_into.workspace")
	digit_count := int(int_text_digits(value, base, workspace))
	required := digit_count
	if value.Negative == POLARITY_NEGATIVE {
		required += SIGN_BYTE_COUNT_MAXIMUM
	}
	if len(destination) < required {
		return 0, STATUS_DESTINATION_TOO_SMALL
	}
	destination_index := 0
	if value.Negative == POLARITY_NEGATIVE {
		destination[destination_index] = '-'
		destination_index++
	}
	for digit_index := digit_count - 1; digit_index >= 0; digit_index-- {
		destination[destination_index] = workspace.Digits[digit_index]
		destination_index++
	}
	return Text_Count(required), STATUS_OK
}

// Int_Parse commits only complete text whose magnitude fits owned Int storage.
func Int_Parse(
	destination *Int,
	source Text_Unvalidated,
	requested Base_Unvalidated,
	workspace *Int_Parse_Workspace,
) (used Base, status Parse_Status) {
	defer func() {
		Base_Invariants(used, "int_parse.used")
		Parse_Status_Invariants(status, "int_parse.status")
	}()
	Int_Invariants(destination, "int_parse.destination")
	Text_Unvalidated_Invariants(source, "int_parse.source")
	Base_Unvalidated_Invariants(requested, "int_parse.requested")
	Int_Parse_Workspace_Invariants(workspace, "int_parse.workspace")
	used = BASE_MINIMUM
	if len(source) == bytes.SLICE_SIZE_MINIMUM {
		return used, STATUS_INPUT_INVALID
	}
	if len(source) > INT_TEXT_SIZE_MAXIMUM {
		return used, STATUS_INPUT_INVALID
	}
	source_index := bytes.SLICE_SIZE_MINIMUM
	if source[source_index] == '-' {
		source_index++
	} else if source[source_index] == '+' {
		source_index++
	}
	if source_index == len(source) {
		return used, STATUS_INPUT_INVALID
	}
	if requested != BASE_AUTOMATIC {
		validated, validation_status := Base_Validate(requested)
		if validation_status != Validation_Status(STATUS_OK) {
			return used, STATUS_INPUT_INVALID
		}
		used = validated
	} else {
		used = BASE_DECIMAL
		if source[source_index] == '0' {
			used = BASE_OCTAL
			if source_index+SIGN_BYTE_COUNT_MAXIMUM < len(source) {
				switch source[source_index+SIGN_BYTE_COUNT_MAXIMUM] {
				case 'b', 'B':
					used = BASE_BINARY
					source_index += BASE_PREFIX_BYTE_COUNT
				case 'o', 'O':
					used = BASE_OCTAL
					source_index += BASE_PREFIX_BYTE_COUNT
				case 'x', 'X':
					used = BASE_HEXADECIMAL
					source_index += BASE_PREFIX_BYTE_COUNT
				}
			}
		}
	}
	status = int_parse_digits(
		destination, Parse_Text(source), Boolean(requested == BASE_AUTOMATIC), used,
		Parse_Text_Index(source_index), workspace,
	)
	return used, status
}

func int_parse_digits(
	destination *Int,
	source Parse_Text,
	automatic Boolean,
	base Base,
	source_index Parse_Text_Index,
	workspace *Int_Parse_Workspace,
) (status Parse_Status) {
	defer func() { Parse_Status_Invariants(status, "int_parse_digits.status") }()
	Int_Invariants(destination, "int_parse_digits.destination")
	Parse_Text_Invariants(source, "int_parse_digits.source")
	Boolean_Invariants(automatic, "int_parse_digits.automatic")
	Base_Invariants(base, "int_parse_digits.base")
	Parse_Text_Index_Invariants(source_index, "int_parse_digits.source_index")
	Int_Parse_Workspace_Invariants(workspace, "int_parse_digits.workspace")
	sign_count := bytes.SLICE_SIZE_MINIMUM
	if source[bytes.SLICE_SIZE_MINIMUM] == '-' {
		sign_count = SIGN_BYTE_COUNT_MAXIMUM
	} else if source[bytes.SLICE_SIZE_MINIMUM] == '+' {
		sign_count = SIGN_BYTE_COUNT_MAXIMUM
	}
	prefix_consumed := int(source_index) > sign_count
	count := Word_Count(WORD_COUNT_MINIMUM)
	digit_seen := false
	separator_seen := false
	for index := int(source_index); index < len(source); index++ {
		character := source[index]
		if character == '_' {
			if !bool(automatic) {
				return STATUS_INPUT_INVALID
			}
			if separator_seen {
				return STATUS_INPUT_INVALID
			}
			if !digit_seen {
				if !prefix_consumed {
					return STATUS_INPUT_INVALID
				}
			}
			separator_seen = true
			continue
		}
		digit, valid := int_parse_digit(Parse_Character(character), base)
		if !valid {
			return STATUS_INPUT_INVALID
		}
		accumulation_status := int_parse_accumulate(workspace, &count, base, digit)
		if accumulation_status != Arithmetic_Status(STATUS_OK) {
			return Parse_Status(accumulation_status)
		}
		digit_seen = true
		separator_seen = false
	}
	if !digit_seen {
		return STATUS_INPUT_INVALID
	}
	if separator_seen {
		return STATUS_INPUT_INVALID
	}
	previous_count := destination.Count
	for index := WORD_COUNT_MINIMUM; index < int(count); index++ {
		destination.Words[index] = workspace.Words[index]
	}
	destination.Count = count
	destination.Negative = POLARITY_NONNEGATIVE
	if count > WORD_COUNT_MINIMUM {
		if source[bytes.SLICE_SIZE_MINIMUM] == '-' {
			destination.Negative = POLARITY_NEGATIVE
		}
	}
	int_clear(destination, count, previous_count)
	return STATUS_OK
}

func int_parse_digit(
	character Parse_Character, base Base,
) (digit Parse_Digit, valid Boolean) {
	defer func() {
		Parse_Digit_Invariants(digit, "int_parse_digit.digit")
		Boolean_Invariants(valid, "int_parse_digit.valid")
	}()
	Parse_Character_Invariants(character, "int_parse_digit.character")
	Base_Invariants(base, "int_parse_digit.base")
	value := uint64(character)
	decoded := false
	if value >= '0' {
		if value <= '9' {
			digit = Parse_Digit(value - '0')
			decoded = true
		}
	}
	if !decoded {
		if value >= 'a' {
			if value <= 'z' {
				digit = Parse_Digit(value-'a') + DECIMAL_DIGIT_COUNT
				decoded = true
			}
		}
	}
	if !decoded {
		if value >= 'A' {
			if value <= 'Z' {
				digit = Parse_Digit(value - 'A')
				if base <= BASE_LOWERCASE_MAXIMUM {
					digit += DECIMAL_DIGIT_COUNT
				} else {
					digit += BASE_LOWERCASE_MAXIMUM
				}
				decoded = true
			}
		}
	}
	if !decoded {
		return 0, false
	}
	if digit >= Parse_Digit(base) {
		return 0, false
	}
	return digit, true
}

func int_parse_accumulate(
	workspace *Int_Parse_Workspace, count *Word_Count, base Base, digit Parse_Digit,
) (status Arithmetic_Status) {
	defer func() { Arithmetic_Status_Invariants(status, "int_parse_accumulate.status") }()
	Int_Parse_Workspace_Invariants(workspace, "int_parse_accumulate.workspace")
	Word_Count_Invariants(*count, "int_parse_accumulate.count")
	Base_Invariants(base, "int_parse_accumulate.base")
	Parse_Digit_Invariants(digit, "int_parse_accumulate.digit")
	carry := uint64(digit)
	for index := WORD_COUNT_MINIMUM; index < int(*count); index++ {
		word := uint64(workspace.Words[index])
		low_product := (word&TEXT_DIVISION_LIMB_MASK)*uint64(base) + carry
		low := low_product & TEXT_DIVISION_LIMB_MASK
		carry = low_product >> TEXT_DIVISION_LIMB_BIT_COUNT
		high_product := (word>>TEXT_DIVISION_LIMB_BIT_COUNT)*uint64(base) + carry
		high := high_product & TEXT_DIVISION_LIMB_MASK
		carry = high_product >> TEXT_DIVISION_LIMB_BIT_COUNT
		workspace.Words[index] = Word(high<<TEXT_DIVISION_LIMB_BIT_COUNT | low)
	}
	if carry == 0 {
		return STATUS_OK
	}
	if *count == WORD_COUNT_MAXIMUM {
		return STATUS_VALUE_OVERFLOW
	}
	workspace.Words[*count] = Word(carry)
	*count++
	return STATUS_OK
}

func int_text_digits(
	value *Int, base Base, workspace *Int_Text_Workspace,
) (digit_count Text_Digit_Count) {
	defer func() { Text_Digit_Count_Invariants(digit_count, "int_text_digits.digit_count") }()
	Int_Invariants(value, "int_text_digits.value")
	Base_Invariants(base, "int_text_digits.base")
	Int_Text_Workspace_Invariants(workspace, "int_text_digits.workspace")
	if base == BASE_MINIMUM {
		return int_text_binary_digits(value, workspace)
	}
	if base == BASE_DECIMAL {
		return Text_Digit_Count(int_text_decimal_digits(value, workspace))
	}
	for index := WORD_COUNT_MINIMUM; index < int(value.Count); index++ {
		workspace.Words[index] = value.Words[index]
	}
	word_count := int(value.Count)
	if word_count == WORD_COUNT_MINIMUM {
		workspace.Digits[WORD_COUNT_MINIMUM] = '0'
		return SIGN_BYTE_COUNT_MAXIMUM
	}
	divisor := uint64(base)
	for word_count > WORD_COUNT_MINIMUM {
		remainder := uint64(0)
		for index := word_count - 1; index >= WORD_COUNT_MINIMUM; index-- {
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
			if workspace.Words[word_count-1] != 0 {
				break
			}
			word_count--
		}
		digit := byte(remainder)
		if remainder < DECIMAL_DIGIT_COUNT {
			digit += '0'
		} else {
			digit -= DECIMAL_DIGIT_COUNT
			if digit < LETTER_DIGIT_COUNT {
				digit += 'a'
			} else {
				digit -= LETTER_DIGIT_COUNT
				digit += 'A'
			}
		}
		workspace.Digits[digit_count] = digit
		digit_count++
	}
	return digit_count
}

func int_text_binary_digits(
	value *Int, workspace *Int_Text_Workspace,
) (digit_count Text_Digit_Count) {
	defer func() {
		Text_Digit_Count_Invariants(digit_count, "int_text_binary_digits.digit_count")
	}()
	Int_Invariants(value, "int_text_binary_digits.value")
	Int_Text_Workspace_Invariants(workspace, "int_text_binary_digits.workspace")
	bit_count := int(Int_Bit_Count(value))
	if bit_count == BIT_COUNT_MINIMUM {
		workspace.Digits[WORD_COUNT_MINIMUM] = '0'
		return SIGN_BYTE_COUNT_MAXIMUM
	}
	for bit_index := BIT_COUNT_MINIMUM; bit_index < bit_count; bit_index++ {
		word_index := bit_index / WORD_BIT_COUNT
		word_bit_index := uint(bit_index) % WORD_BIT_COUNT
		bit := byte(value.Words[word_index]>>word_bit_index) & byte(bits.CARRY_MAXIMUM)
		workspace.Digits[bit_index] = '0' + bit
	}
	return Text_Digit_Count(bit_count)
}

// Int_Set copies active magnitude because inactive words carry no numeric state.
func Int_Set(destination *Int, source *Int) {
	Int_Invariants(destination, "int_set.destination")
	Int_Invariants(source, "int_set.source")
	if destination == source {
		return
	}
	previous_count := destination.Count
	for index := Word_Count(WORD_COUNT_MINIMUM); index < source.Count; index++ {
		destination.Words[index] = source.Words[index]
	}
	destination.Count = source.Count
	destination.Negative = source.Negative
	if previous_count > destination.Count {
		int_clear(destination, destination.Count, previous_count)
	}
}

// Int_Sign returns typed polarity so callers cannot confuse arbitrary integers with signs.
func Int_Sign(value *Int) (sign Sign) {
	defer func() { Sign_Invariants(sign, "int_sign.sign") }()
	Int_Invariants(value, "int_sign.value")
	if value.Count == WORD_COUNT_MINIMUM {
		return SIGN_ZERO
	}
	if value.Negative == POLARITY_NEGATIVE {
		return SIGN_NEGATIVE
	}
	return SIGN_POSITIVE
}

// Int_Bit_Count exposes magnitude width without exposing mutable internal words.
func Int_Bit_Count(value *Int) (count Bit_Count) {
	defer func() { Bit_Count_Invariants(count, "int_bit_count.count") }()
	Int_Invariants(value, "int_bit_count.value")
	if value.Count == WORD_COUNT_MINIMUM {
		return 0
	}
	high := value.Words[int(value.Count)-1]
	high_count := WORD_BIT_COUNT - int(bits.Leading_Zeros_64(bits.Word_64(high)))
	return Bit_Count((int(value.Count)-1)*WORD_BIT_COUNT + high_count)
}

// Int_Bit reads one infinite two's-complement bit without constructing signed scratch storage.
func Int_Bit(value *Int, index Bit_Index) (bit Bit_Value) {
	defer func() { Bit_Value_Invariants(bit, "int_bit.bit") }()
	Int_Invariants(value, "int_bit.value")
	Bit_Index_Invariants(index, "int_bit.index")
	word_index := int(index) / WORD_BIT_COUNT
	word_bit_index := uint(index) % WORD_BIT_COUNT
	word := Word(0)
	if word_index < int(value.Count) {
		word = value.Words[word_index]
	}
	if value.Negative == POLARITY_NONNEGATIVE {
		return Bit_Value(word>>word_bit_index) & BIT_SET
	}
	borrow_reaches_word := true
	for lower_index := WORD_COUNT_MINIMUM; lower_index < word_index; lower_index++ {
		if value.Words[lower_index] != 0 {
			borrow_reaches_word = false
			break
		}
	}
	if borrow_reaches_word {
		word--
	}
	return Bit_Value(word>>word_bit_index)&BIT_SET ^ BIT_SET
}

// Int_Trailing_Zero_Bit_Count reports magnitude divisibility by powers of two.
func Int_Trailing_Zero_Bit_Count(value *Int) (count Trailing_Zero_Bit_Count) {
	defer func() {
		Trailing_Zero_Bit_Count_Invariants(count, "int_trailing_zero_bit_count.count")
	}()
	Int_Invariants(value, "int_trailing_zero_bit_count.value")
	for index := Word_Index(WORD_COUNT_MINIMUM); index < Word_Index(value.Count); index++ {
		word := value.Words[index]
		if word != 0 {
			return Trailing_Zero_Bit_Count(
				int(index)*WORD_BIT_COUNT +
					int(bits.Trailing_Zeros_64(bits.Word_64(word))),
			)
		}
	}
	return BIT_COUNT_MINIMUM
}

// Int_Is_Int_64 lets caller branch before requesting a narrowing conversion.
func Int_Is_Int_64(value *Int) (fits Boolean) {
	defer func() { Boolean_Invariants(fits, "int_is_int_64.fits") }()
	Int_Invariants(value, "int_is_int_64.value")
	return int_fits_int_64(value)
}

// Int_Int_64 refuses truncation because bounded arithmetic must keep loss explicit.
func Int_Int_64(value *Int) (result Int_64, status Conversion_Status) {
	defer func() {
		Int_64_Invariants(result, "int_int_64.result")
		Conversion_Status_Invariants(status, "int_int_64.status")
	}()
	Int_Invariants(value, "int_int_64.value")
	if !bool(int_fits_int_64(value)) {
		return 0, STATUS_VALUE_OVERFLOW
	}
	if value.Count == WORD_COUNT_MINIMUM {
		return 0, STATUS_OK
	}
	magnitude := uint64(value.Words[WORD_COUNT_MINIMUM])
	if value.Negative == POLARITY_NONNEGATIVE {
		return Int_64(magnitude), STATUS_OK
	}
	if magnitude == INT_64_NEGATIVE_MAGNITUDE_MAXIMUM {
		return Int_64(bits.INTEGER_64_MINIMUM), STATUS_OK
	}
	return Int_64(-int64(magnitude)), STATUS_OK
}

// Int_Is_Uint_64 keeps negative rejection separate from low-word access.
func Int_Is_Uint_64(value *Int) (fits Boolean) {
	defer func() { Boolean_Invariants(fits, "int_is_uint_64.fits") }()
	Int_Invariants(value, "int_is_uint_64.value")
	if value.Negative == POLARITY_NEGATIVE {
		return false
	}
	return Boolean(value.Count <= 1)
}

// Int_Uint_64 refuses sign loss and high-word truncation.
func Int_Uint_64(value *Int) (result Word_64, status Conversion_Status) {
	defer func() {
		Word_64_Invariants(result, "int_uint_64.result")
		Conversion_Status_Invariants(status, "int_uint_64.status")
	}()
	Int_Invariants(value, "int_uint_64.value")
	if value.Negative == POLARITY_NEGATIVE {
		return 0, STATUS_VALUE_OVERFLOW
	}
	if value.Count > 1 {
		return 0, STATUS_VALUE_OVERFLOW
	}
	if value.Count == WORD_COUNT_MINIMUM {
		return 0, STATUS_OK
	}
	return Word_64(value.Words[WORD_COUNT_MINIMUM]), STATUS_OK
}

// Int_Add preflights final carry so overflow cannot partially replace destination.
func Int_Add(destination *Int, left *Int, right *Int) (status Arithmetic_Status) {
	defer func() { Arithmetic_Status_Invariants(status, "int_add.status") }()
	Int_Invariants(destination, "int_add.destination")
	Int_Invariants(left, "int_add.left")
	Int_Invariants(right, "int_add.right")
	if left.Count == Word_Count(BASE_BINARY) {
		if right.Count == Word_Count(BASE_BINARY) {
			if left.Negative == right.Negative {
				left_low := left.Words[WORD_COUNT_MINIMUM]
				left_high := left.Words[WORD_COUNT_INCREMENT]
				right_low := right.Words[WORD_COUNT_MINIMUM]
				right_high := right.Words[WORD_COUNT_INCREMENT]
				low := left_low + right_low
				carry := Word(0)
				if low < left_low {
					carry = Word(bits.CARRY_MAXIMUM)
				}
				high_sum := left_high + right_high
				high := high_sum + carry
				high_carry := Word(0)
				if high_sum < left_high {
					high_carry = Word(bits.CARRY_MAXIMUM)
				}
				if high < high_sum {
					high_carry = Word(bits.CARRY_MAXIMUM)
				}
				previous_count := destination.Count
				destination.Words[WORD_COUNT_MINIMUM] = low
				destination.Words[WORD_COUNT_INCREMENT] = high
				destination.Words[BASE_BINARY] = high_carry
				destination.Count = Word_Count(BASE_BINARY)
				if high_carry != 0 {
					destination.Count += Word_Count(WORD_COUNT_INCREMENT)
				}
				destination.Negative = left.Negative
				int_clear(destination, destination.Count, previous_count)
				return STATUS_OK
			}
		}
	}
	// One-word sums avoid duplicate carry scans while full-width sums retain atomic preflight.
	if left.Count == Word_Count(WORD_COUNT_INCREMENT) {
		if right.Count == Word_Count(WORD_COUNT_INCREMENT) {
			if left.Negative == right.Negative {
				left_word := left.Words[WORD_COUNT_MINIMUM]
				right_word := right.Words[WORD_COUNT_MINIMUM]
				negative := left.Negative
				previous_count := destination.Count
				sum := left_word + right_word
				destination.Words[WORD_COUNT_MINIMUM] = sum
				count := Word_Count(WORD_COUNT_INCREMENT)
				if sum < left_word {
					destination.Words[count] = Word(bits.CARRY_MAXIMUM)
					count++
				}
				destination.Count = count
				destination.Negative = negative
				if previous_count > count {
					int_clear(destination, count, previous_count)
				}
				return STATUS_OK
			}
		}
	}
	return int_sum(destination, left, left.Negative, right, right.Negative)
}

// Int_Multiply keeps partial words in caller workspace so either input may alias destination.
func Int_Multiply(
	destination *Int,
	left *Int,
	right *Int,
	workspace *Int_Multiplication_Workspace,
) (status Arithmetic_Status) {
	defer func() { Arithmetic_Status_Invariants(status, "int_multiply.status") }()
	Int_Invariants(destination, "int_multiply.destination")
	Int_Invariants(left, "int_multiply.left")
	Int_Invariants(right, "int_multiply.right")
	Int_Multiplication_Workspace_Invariants(workspace, "int_multiply.workspace")
	if int_multiply_double_word(&Int_References{destination, left, right}) {
		return STATUS_OK
	}
	if left.Count == Word_Count(WORD_COUNT_INCREMENT) {
		if right.Count == Word_Count(WORD_COUNT_INCREMENT) {
			left_word := left.Words[WORD_COUNT_MINIMUM]
			right_word := right.Words[WORD_COUNT_MINIMUM]
			negative := POLARITY_NONNEGATIVE
			if left.Negative != right.Negative {
				negative = POLARITY_NEGATIVE
			}
			high, low := bits.Multiply_64(
				bits.Word_64(left_word), bits.Multiplier_64(right_word),
			)
			previous_count := destination.Count
			destination.Words[WORD_COUNT_MINIMUM] = Word(low)
			count := Word_Count(WORD_COUNT_INCREMENT)
			if high != 0 {
				destination.Words[count] = Word(high)
				count++
			}
			destination.Count = count
			destination.Negative = negative
			if previous_count > count {
				int_clear(destination, count, previous_count)
			}
			return STATUS_OK
		}
	}
	negative := POLARITY_NONNEGATIVE
	if left.Negative != right.Negative {
		negative = POLARITY_NEGATIVE
	}
	status = int_multiply_words(workspace, left, right)
	if status != Arithmetic_Status(STATUS_OK) {
		return status
	}
	count := left.Count + right.Count
	if count > WORD_COUNT_MAXIMUM {
		count = WORD_COUNT_MAXIMUM
	}
	for count > WORD_COUNT_MINIMUM {
		if workspace.Product[int(count)-1] != 0 {
			break
		}
		count--
	}
	previous_count := destination.Count
	for index := Word_Index(WORD_COUNT_MINIMUM); index < Word_Index(count); index++ {
		destination.Words[index] = workspace.Product[index]
	}
	destination.Count = count
	destination.Negative = negative
	if count == WORD_COUNT_MINIMUM {
		destination.Negative = POLARITY_NONNEGATIVE
	}
	int_clear(destination, count, previous_count)
	return STATUS_OK
}

// Int_Divide_Modulus adjusts truncated scratch results before either caller output changes.
func Int_Divide_Modulus(
	quotient *Int,
	modulus *Int,
	dividend *Int,
	divisor *Int,
	workspace *Int_Division_Workspace,
) (status Division_Status) {
	defer func() { Division_Status_Invariants(status, "int_divide_modulus.status") }()
	Int_Invariants(quotient, "int_divide_modulus.quotient")
	Int_Invariants(modulus, "int_divide_modulus.modulus")
	Int_Invariants(dividend, "int_divide_modulus.dividend")
	Int_Invariants(divisor, "int_divide_modulus.divisor")
	Int_Division_Workspace_Invariants(workspace, "int_divide_modulus.workspace")
	if quotient == modulus {
		return STATUS_DESTINATIONS_OVERLAP
	}
	quotient_negative := POLARITY_NONNEGATIVE
	if dividend.Negative != divisor.Negative {
		quotient_negative = POLARITY_NEGATIVE
	}
	if dividend.Count <= Word_Count(WORD_COUNT_INCREMENT) {
		if divisor.Count == Word_Count(WORD_COUNT_INCREMENT) {
			dividend_word := Word(0)
			if dividend.Count != Word_Count(WORD_COUNT_MINIMUM) {
				dividend_word = dividend.Words[WORD_COUNT_MINIMUM]
			}
			divisor_word := divisor.Words[WORD_COUNT_MINIMUM]
			quotient_word := dividend_word / divisor_word
			modulus_word := dividend_word % divisor_word
			if dividend.Negative == POLARITY_NEGATIVE {
				if modulus_word != 0 {
					quotient_word++
					modulus_word = divisor_word - modulus_word
				}
			}
			workspace.Quotient[WORD_COUNT_MINIMUM] = quotient_word
			workspace.Remainder[WORD_COUNT_MINIMUM] = modulus_word
			workspace.Quotient_Count = Quotient_Count(WORD_COUNT_INCREMENT)
			workspace.Remainder_Count = Remainder_Count(WORD_COUNT_INCREMENT)
			if quotient_word == 0 {
				workspace.Quotient_Count = Quotient_Count(WORD_COUNT_MINIMUM)
			}
			if modulus_word == 0 {
				workspace.Remainder_Count = Remainder_Count(WORD_COUNT_MINIMUM)
			}
			int_set_word(quotient, quotient_word, quotient_negative)
			int_set_word(modulus, modulus_word, POLARITY_NONNEGATIVE)
			return STATUS_OK
		}
	}
	int_divide_magnitudes(workspace, dividend, divisor)
	int_division_euclidean_adjust(workspace, dividend, divisor)
	if divisor.Count == WORD_COUNT_MINIMUM {
		return STATUS_DIVISOR_ZERO
	}
	int_set_division_quotient(quotient, workspace, quotient_negative)
	int_set_division_remainder(modulus, workspace, POLARITY_NONNEGATIVE)
	return STATUS_OK
}

// Int_Binomial reduces each scalar ratio before multiplication so bounded final values fit.
func Int_Binomial(
	destination *Int, n Int_64, k Int_64, workspace *Int_Product_Workspace,
) (status Arithmetic_Status) {
	defer func() { Arithmetic_Status_Invariants(status, "int_binomial.status") }()
	Int_Invariants(destination, "int_binomial.destination")
	Int_64_Invariants(n, "int_binomial.n")
	Int_64_Invariants(k, "int_binomial.k")
	Int_Product_Workspace_Invariants(workspace, "int_binomial.workspace")
	if int_binomial_word(destination, n, k) {
		return STATUS_OK
	}
	accumulator := &workspace.Integers[PRODUCT_ACCUMULATOR_INDEX]
	factor := &workspace.Integers[PRODUCT_FACTOR_INDEX]
	int_zero(accumulator, accumulator.Count)
	int_zero(factor, factor.Count)
	if k > n {
		int_zero(destination, destination.Count)
		return STATUS_OK
	}
	Int_Set_Uint_64(accumulator, Word_64(bits.CARRY_MAXIMUM))
	if k <= 0 {
		Int_Set(destination, accumulator)
		return STATUS_OK
	}
	complement := n - k
	if k > complement {
		k = complement
	}
	multiplication := (*Int_Multiplication_Workspace)(&workspace.Multiplication)
	for index := int64(WORD_COUNT_MINIMUM); index < int64(k); index++ {
		numerator := uint64(int64(n) - index)
		denominator := uint64(index + WORD_COUNT_INCREMENT)
		common_left := numerator
		common_right := denominator
		for common_right != 0 {
			common_left, common_right = common_right, common_left%common_right
		}
		numerator /= common_left
		denominator /= common_left
		if denominator > uint64(bits.CARRY_MAXIMUM) {
			Int_Set_Uint_64(factor, Word_64(denominator))
			int_binomial_divide(workspace)
		}
		Int_Set_Uint_64(factor, Word_64(numerator))
		multiply_status := Int_Multiply(
			accumulator, accumulator, factor, multiplication,
		)
		if multiply_status != Arithmetic_Status(STATUS_OK) {
			return multiply_status
		}
	}
	Int_Set(destination, accumulator)
	return STATUS_OK
}

// Int_Multiply_Range stops at fixed Int overflow before scalar range size can govern work.
func Int_Multiply_Range(
	destination *Int,
	minimum Int_64,
	maximum Int_64,
	workspace *Int_Product_Workspace,
) (status Arithmetic_Status) {
	defer func() { Arithmetic_Status_Invariants(status, "int_multiply_range.status") }()
	Int_Invariants(destination, "int_multiply_range.destination")
	Int_64_Invariants(minimum, "int_multiply_range.minimum")
	Int_64_Invariants(maximum, "int_multiply_range.maximum")
	Int_Product_Workspace_Invariants(workspace, "int_multiply_range.workspace")
	if int_multiply_range_word(destination, minimum, maximum) {
		return STATUS_OK
	}
	accumulator := &workspace.Integers[PRODUCT_ACCUMULATOR_INDEX]
	factor := &workspace.Integers[PRODUCT_FACTOR_INDEX]
	int_zero(accumulator, accumulator.Count)
	int_zero(factor, factor.Count)
	if minimum > maximum {
		Int_Set_Uint_64(accumulator, Word_64(bits.CARRY_MAXIMUM))
		Int_Set(destination, accumulator)
		return STATUS_OK
	}
	if minimum <= 0 {
		if maximum >= 0 {
			int_zero(destination, destination.Count)
			return STATUS_OK
		}
	}
	negative := false
	start := uint64(minimum)
	end := uint64(maximum)
	if maximum < 0 {
		distance := uint64(maximum) - uint64(minimum)
		if distance&uint64(bits.CARRY_MAXIMUM) == 0 {
			negative = true
		}
		start = uint64(-(maximum + 1)) + uint64(bits.CARRY_MAXIMUM)
		end = uint64(-(minimum + 1)) + uint64(bits.CARRY_MAXIMUM)
	}
	Int_Set_Uint_64(accumulator, Word_64(bits.CARRY_MAXIMUM))
	multiplication := (*Int_Multiplication_Workspace)(&workspace.Multiplication)
	for value := start; ; value++ {
		Int_Set_Uint_64(factor, Word_64(value))
		multiply_status := Int_Multiply(
			accumulator, accumulator, factor, multiplication,
		)
		if multiply_status != Arithmetic_Status(STATUS_OK) {
			return multiply_status
		}
		if value == end {
			break
		}
	}
	if negative {
		accumulator.Negative = POLARITY_NEGATIVE
	}
	Int_Set(destination, accumulator)
	return STATUS_OK
}

// Int_Exponent uses binary exponentiation and commits only one bounded exact power.
func Int_Exponent(destination *Int, base *Int, exponent *Int,
	workspace *Int_Exponent_Workspace,
) (status Arithmetic_Status) {
	defer func() { Arithmetic_Status_Invariants(status, "int_exponent.status") }()
	Int_Invariants(destination, "int_exponent.destination")
	Int_Invariants(base, "int_exponent.base")
	Int_Invariants(exponent, "int_exponent.exponent")
	Int_Exponent_Workspace_Invariants(workspace, "int_exponent.workspace")
	if base.Count <= Word_Count(WORD_COUNT_INCREMENT) {
		if int_exponent_word(&Int_References{destination, base, exponent}) {
			return STATUS_OK
		}
	}
	result, factor := &workspace.Integers[EXPONENT_RESULT_INDEX],
		&workspace.Integers[EXPONENT_FACTOR_INDEX]
	Int_Set(factor, base)
	int_zero(result, result.Count)
	Int_Set_Uint_64(result, Word_64(bits.CARRY_MAXIMUM))
	if exponent.Negative == POLARITY_NEGATIVE {
		Int_Set(destination, result)
		return STATUS_OK
	}
	if exponent.Count == WORD_COUNT_MINIMUM {
		Int_Set(destination, result)
		return STATUS_OK
	}
	if factor.Count == WORD_COUNT_MINIMUM {
		Int_Set(destination, factor)
		return STATUS_OK
	}
	if factor.Count == WORD_COUNT_INCREMENT {
		if factor.Words[WORD_COUNT_MINIMUM] == Word(bits.CARRY_MAXIMUM) {
			if factor.Negative == POLARITY_NEGATIVE {
				parity := exponent.Words[WORD_COUNT_MINIMUM] &
					Word(bits.CARRY_MAXIMUM)
				if parity != 0 {
					result.Negative = POLARITY_NEGATIVE
				}
			}
			Int_Set(destination, result)
			return STATUS_OK
		}
	}
	bit_count := int(Int_Bit_Count(exponent))
	multiplication := (*Int_Multiplication_Workspace)(&workspace.Multiplication)
	for bit_index := BIT_COUNT_MINIMUM; bit_index < bit_count; bit_index++ {
		word_index := bit_index / WORD_BIT_COUNT
		word_shift := uint(bit_index) % WORD_BIT_COUNT
		bit := exponent.Words[word_index] >> word_shift
		if bit&Word(bits.CARRY_MAXIMUM) != 0 {
			if int_exponent_accumulate(
				&Int_References{result, result, factor}, multiplication,
			) {
				return STATUS_VALUE_OVERFLOW
			}
		}
		if bit_index+WORD_COUNT_INCREMENT < bit_count {
			if factor.Count == Word_Count(BASE_BINARY) {
				int_square_double_word(&Int_References{factor, factor, factor})
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
	Int_Set(destination, result)
	return STATUS_OK
}

// Int_Modular_Inverse returns the unique nonnegative residue when an inverse exists.
func Int_Modular_Inverse(
	destination *Int, value *Int, modulus *Int, workspace *Int_Modular_Workspace,
) (status Modular_Status) {
	defer func() { Modular_Status_Invariants(status, "int_modular_inverse.status") }()
	Int_Invariants(destination, "int_modular_inverse.destination")
	Int_Invariants(value, "int_modular_inverse.value")
	Int_Invariants(modulus, "int_modular_inverse.modulus")
	Int_Modular_Workspace_Invariants(workspace, "int_modular_inverse.workspace")
	if int_modular_inverse_word(destination, value, modulus) {
		return STATUS_OK
	}
	return int_modular_inverse(destination, value, modulus, workspace)
}

// Int_Modular_Multiply reduces both operands before overflow-free product accumulation.
func Int_Modular_Multiply(
	destination *Int,
	left *Int,
	right *Int,
	modulus *Int,
	workspace *Int_Modular_Workspace,
) (status Divisor_Status) {
	defer func() { Divisor_Status_Invariants(status, "int_modular_multiply_public.status") }()
	Int_Invariants(destination, "int_modular_multiply_public.destination")
	Int_Invariants(left, "int_modular_multiply_public.left")
	Int_Invariants(right, "int_modular_multiply_public.right")
	Int_Invariants(modulus, "int_modular_multiply_public.modulus")
	Int_Modular_Workspace_Invariants(workspace, "int_modular_multiply_public.workspace")
	normalized_modulus := &workspace.Integers[MODULAR_MODULUS_INDEX]
	Int_Set(normalized_modulus, modulus)
	normalized_modulus.Negative = POLARITY_NONNEGATIVE
	if normalized_modulus.Count == WORD_COUNT_MINIMUM {
		return STATUS_DIVISOR_ZERO
	}
	int_modular_reduce(workspace, MODULAR_REDUCTION_OPERAND, left)
	coefficient := &workspace.Integers[MODULAR_COEFFICIENT_DIVIDEND_INDEX]
	factor := &workspace.Integers[MODULAR_EXPONENT_FACTOR_INDEX]
	Int_Set(coefficient, factor)
	int_modular_reduce(workspace, MODULAR_REDUCTION_OPERAND, right)
	result := &workspace.Integers[MODULAR_EXPONENT_RESULT_INDEX]
	Int_Set(result, coefficient)
	int_modular_multiply(workspace, MODULAR_MULTIPLICATION_ACCUMULATE)
	Int_Set(destination, result)
	return STATUS_OK
}

// Int_Modular_Exponent computes stdlib-compatible signed exponentiation modulo magnitude.
func Int_Modular_Exponent(
	destination *Int,
	base *Int,
	exponent *Int,
	modulus *Int,
	workspace *Int_Modular_Workspace,
) (status Modular_Status) {
	defer func() { Modular_Status_Invariants(status, "int_modular_exponent.status") }()
	Int_Invariants(destination, "int_modular_exponent.destination")
	Int_Invariants(base, "int_modular_exponent.base")
	Int_Invariants(exponent, "int_modular_exponent.exponent")
	Int_Invariants(modulus, "int_modular_exponent.modulus")
	Int_Modular_Workspace_Invariants(workspace, "int_modular_exponent.workspace")
	if modulus.Count == Word_Count(WORD_COUNT_INCREMENT) {
		references := Int_References{destination, base, exponent}
		modulus_references := Int_References{modulus, modulus, modulus}
		if int_modular_exponent_word(&references, &modulus_references) {
			return STATUS_OK
		}
	}
	normalized_modulus := &workspace.Integers[MODULAR_MODULUS_INDEX]
	Int_Set(normalized_modulus, modulus)
	normalized_modulus.Negative = POLARITY_NONNEGATIVE
	if normalized_modulus.Count == WORD_COUNT_MINIMUM {
		return STATUS_DIVISOR_ZERO
	}
	factor := &workspace.Integers[MODULAR_EXPONENT_FACTOR_INDEX]
	if exponent.Negative == POLARITY_NEGATIVE {
		inverse_status := int_modular_inverse(factor, base, normalized_modulus, workspace)
		if inverse_status != Modular_Status(STATUS_OK) {
			return inverse_status
		}
	} else {
		int_modular_reduce(workspace, MODULAR_REDUCTION_OPERAND, base)
	}
	int_modular_exponent_commit(workspace, destination, exponent)
	return STATUS_OK
}

func int_modular_inverse(
	destination *Int, value *Int, modulus *Int, workspace *Int_Modular_Workspace,
) (status Modular_Status) {
	defer func() { Modular_Status_Invariants(status, "int_modular_inverse_raw.status") }()
	Int_Invariants(destination, "int_modular_inverse_raw.destination")
	Int_Invariants(value, "int_modular_inverse_raw.value")
	Int_Invariants(modulus, "int_modular_inverse_raw.modulus")
	Int_Modular_Workspace_Invariants(workspace, "int_modular_inverse_raw.workspace")
	normalized_modulus := &workspace.Integers[MODULAR_MODULUS_INDEX]
	Int_Set(normalized_modulus, modulus)
	normalized_modulus.Negative = POLARITY_NONNEGATIVE
	if normalized_modulus.Count == WORD_COUNT_MINIMUM {
		return STATUS_DIVISOR_ZERO
	}
	dividend := &workspace.Integers[MODULAR_GCD_DIVIDEND_INDEX]
	divisor := &workspace.Integers[MODULAR_GCD_DIVISOR_INDEX]
	remainder := &workspace.Integers[MODULAR_GCD_REMAINDER_INDEX]
	coefficient_dividend := &workspace.Integers[MODULAR_COEFFICIENT_DIVIDEND_INDEX]
	coefficient_divisor := &workspace.Integers[MODULAR_COEFFICIENT_DIVISOR_INDEX]
	coefficient_remainder := &workspace.Integers[MODULAR_COEFFICIENT_REMAINDER_INDEX]
	quotient := &workspace.Integers[MODULAR_QUOTIENT_INDEX]
	coefficient_references := Int_References{
		coefficient_remainder, coefficient_dividend, coefficient_divisor,
	}
	Int_Set(dividend, normalized_modulus)
	int_modular_reduce(workspace, MODULAR_REDUCTION_OPERAND, value)
	int_zero(coefficient_dividend, coefficient_dividend.Count)
	Int_Set_Uint_64(coefficient_divisor, Word_64(bits.CARRY_MAXIMUM))
	if normalized_modulus.Count == WORD_COUNT_INCREMENT {
		if normalized_modulus.Words[WORD_COUNT_MINIMUM] == Word(bits.CARRY_MAXIMUM) {
			int_zero(coefficient_divisor, coefficient_divisor.Count)
		}
	}
	coefficient_bounded := Boolean(
		int(normalized_modulus.Count) <=
			(WORD_COUNT_MAXIMUM-WORD_COUNT_INCREMENT)/BASE_BINARY,
	)
	remainder_references := Int_References{remainder, dividend, divisor}
	result_references := Int_References{destination, normalized_modulus, quotient}
	return Modular_Status(int_modular_inverse_euclidean(
		workspace, coefficient_bounded, &result_references,
		&remainder_references, &coefficient_references,
	))
}

func int_modular_reduce(
	workspace *Int_Modular_Workspace, operation Modular_Reduction_Operation, value *Int,
) {
	Int_Modular_Workspace_Invariants(workspace, "int_modular_reduce.workspace")
	Modular_Reduction_Operation_Invariants(operation, "int_modular_reduce.operation")
	Int_Invariants(value, "int_modular_reduce.value")
	destination_index := MODULAR_EXPONENT_FACTOR_INDEX
	if operation == MODULAR_REDUCTION_FACTOR {
		destination_index = MODULAR_MULTIPLICATION_FACTOR_INDEX
	}
	destination := &workspace.Integers[destination_index]
	modulus := &workspace.Integers[MODULAR_MODULUS_INDEX]
	quotient := &workspace.Integers[MODULAR_QUOTIENT_INDEX]
	division := (*Int_Division_Workspace)(&workspace.Division)
	invariant.Always(
		modulus.Count > WORD_COUNT_MINIMUM,
		"Modular reduction receives one nonzero modulus.",
	)
	if value.Negative == POLARITY_NONNEGATIVE {
		if int_compare_absolute(value, modulus) == ORDER_BEFORE {
			Int_Set(destination, value)
			return
		}
	}
	status := Int_Divide_Modulus(quotient, destination, value, modulus, division)
	invariant.Always(
		status == Division_Status(STATUS_OK),
		"Modular reduction owns distinct outputs and a nonzero modulus.",
	)
}

func int_modular_multiply(
	workspace *Int_Modular_Workspace,
	operation Modular_Multiplication_Operation,
) {
	Int_Modular_Workspace_Invariants(workspace, "int_modular_multiply.workspace")
	Modular_Multiplication_Operation_Invariants(operation, "int_modular_multiply.operation")
	result_index, left_index := MODULAR_EXPONENT_RESULT_INDEX, MODULAR_EXPONENT_RESULT_INDEX
	right_index := MODULAR_EXPONENT_FACTOR_INDEX
	if operation == MODULAR_MULTIPLICATION_SQUARE {
		result_index = MODULAR_EXPONENT_FACTOR_INDEX
		left_index = MODULAR_EXPONENT_FACTOR_INDEX
	}
	if operation == MODULAR_MULTIPLICATION_COEFFICIENT {
		result_index = MODULAR_COEFFICIENT_REMAINDER_INDEX
		left_index = MODULAR_QUOTIENT_INDEX
		right_index = MODULAR_COEFFICIENT_DIVISOR_INDEX
	}
	left, right := &workspace.Integers[left_index], &workspace.Integers[right_index]
	destination := &workspace.Integers[result_index]
	if int_modular_multiply_identity(&Int_References{destination, left, right}) {
		return
	}
	modulus := &workspace.Integers[MODULAR_MODULUS_INDEX]
	invariant.Always(
		modulus.Count > WORD_COUNT_MINIMUM,
		"Modular multiplication receives one nonzero modulus.",
	)
	if int(left.Count)+int(right.Count) <= WORD_COUNT_MAXIMUM {
		product := &workspace.Integers[MODULAR_THRESHOLD_INDEX]
		multiplication := (*Int_Multiplication_Workspace)(&workspace.Multiplication)
		status := Int_Multiply(product, left, right, multiplication)
		invariant.Always(
			status == Arithmetic_Status(STATUS_OK),
			"The operand-count sum proves the exact modular product fits.",
		)
		if int_modular_reduce_normalized_double_word(
			&Int_References{destination, product, modulus}) {
			return
		}
		quotient := &workspace.Integers[MODULAR_QUOTIENT_INDEX]
		division := (*Int_Division_Workspace)(&workspace.Division)
		division_status := Int_Divide_Modulus(
			quotient, destination, product, modulus, division,
		)
		invariant.Always(
			division_status == Division_Status(STATUS_OK),
			"The normalized modular divisor remains nonzero.",
		)
		return
	}
	factor_source, multiplier_source := left, right
	if Int_Bit_Count(left) < Int_Bit_Count(right) {
		factor_source, multiplier_source = right, left
	}
	multiplier := &workspace.Integers[MODULAR_MULTIPLICATION_VALUE_INDEX]
	result := &workspace.Integers[MODULAR_MULTIPLICATION_RESULT_INDEX]
	Int_Set(multiplier, multiplier_source)
	int_modular_reduce(workspace, MODULAR_REDUCTION_FACTOR, factor_source)
	int_zero(result, result.Count)
	bit_count := int(Int_Bit_Count(multiplier))
	for bit_index := BIT_COUNT_MINIMUM; bit_index < bit_count; bit_index++ {
		word_index := bit_index / WORD_BIT_COUNT
		word_shift := uint(bit_index) % WORD_BIT_COUNT
		bit := multiplier.Words[word_index] >> word_shift
		if bit&Word(bits.CARRY_MAXIMUM) != 0 {
			int_modular_add(&workspace.Integers, MODULAR_ADDITION_ACCUMULATE)
		}
		if bit_index+WORD_COUNT_INCREMENT < bit_count {
			int_modular_add(&workspace.Integers, MODULAR_ADDITION_DOUBLE)
		}
	}
	Int_Set(destination, result)
}

func int_modular_add(
	integers *Modular_Integers, operation Modular_Addition_Operation,
) {
	Modular_Integers_Invariants(*integers, "int_modular_add.integers")
	Modular_Addition_Operation_Invariants(operation, "int_modular_add.operation")
	destination_index := MODULAR_MULTIPLICATION_RESULT_INDEX
	left_index := MODULAR_MULTIPLICATION_RESULT_INDEX
	if operation == MODULAR_ADDITION_DOUBLE {
		destination_index = MODULAR_MULTIPLICATION_FACTOR_INDEX
		left_index = MODULAR_MULTIPLICATION_FACTOR_INDEX
	}
	destination := &integers[destination_index]
	left := &integers[left_index]
	right := &integers[MODULAR_MULTIPLICATION_FACTOR_INDEX]
	modulus := &integers[MODULAR_MODULUS_INDEX]
	threshold := &integers[MODULAR_THRESHOLD_INDEX]
	invariant.Always(
		left.Negative == POLARITY_NONNEGATIVE,
		"Modular addition receives one nonnegative left residue.",
	)
	invariant.Always(
		right.Negative == POLARITY_NONNEGATIVE,
		"Modular addition receives one nonnegative right residue.",
	)
	status := Int_Subtract(threshold, modulus, right)
	invariant.Always(
		status == Arithmetic_Status(STATUS_OK),
		"Reduced modular addends never exceed the positive modulus.",
	)
	if int_compare_absolute(left, threshold) != ORDER_BEFORE {
		status = Int_Subtract(destination, left, threshold)
	} else {
		status = Int_Add(destination, left, right)
	}
	invariant.Always(
		status == Arithmetic_Status(STATUS_OK),
		"Threshold selection prevents modular addition overflow.",
	)
}

func int_modular_subtract(
	integers *Modular_Integers,
) {
	Modular_Integers_Invariants(*integers, "int_modular_subtract.integers")
	destination := &integers[MODULAR_COEFFICIENT_REMAINDER_INDEX]
	left := &integers[MODULAR_COEFFICIENT_DIVIDEND_INDEX]
	right := &integers[MODULAR_COEFFICIENT_REMAINDER_INDEX]
	modulus := &integers[MODULAR_MODULUS_INDEX]
	threshold := &integers[MODULAR_THRESHOLD_INDEX]
	invariant.Always(
		left.Negative == POLARITY_NONNEGATIVE,
		"Modular subtraction receives one nonnegative left residue.",
	)
	invariant.Always(
		right.Negative == POLARITY_NONNEGATIVE,
		"Modular subtraction receives one nonnegative right residue.",
	)
	status := Arithmetic_Status(STATUS_OK)
	if int_compare_absolute(left, right) != ORDER_BEFORE {
		status = Int_Subtract(destination, left, right)
	} else {
		status = Int_Subtract(threshold, right, left)
		if status == Arithmetic_Status(STATUS_OK) {
			status = Int_Subtract(destination, modulus, threshold)
		}
	}
	invariant.Always(
		status == Arithmetic_Status(STATUS_OK),
		"Reduced modular subtraction stays inside the positive modulus.",
	)
}

// Int_Square_Root applies Newton iteration from a power-of-two upper bound.
func Int_Square_Root(destination *Int, source *Int,
	workspace *Int_Square_Root_Workspace,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "int_square_root.status") }()
	Int_Invariants(destination, "int_square_root.destination")
	Int_Invariants(source, "int_square_root.source")
	Int_Square_Root_Workspace_Invariants(workspace, "int_square_root.workspace")
	if source.Negative == POLARITY_NEGATIVE {
		return STATUS_INPUT_INVALID
	}
	if source.Count == WORD_COUNT_MINIMUM {
		int_zero(destination, destination.Count)
		return STATUS_OK
	}
	if source.Count == Word_Count(WORD_COUNT_INCREMENT) {
		word := uint64(source.Words[WORD_COUNT_MINIMUM])
		bit_count := int(bits.Bit_Size_64(bits.Word_64(word)))
		root := uint64(bits.CARRY_MAXIMUM) << uint(
			(bit_count+SQUARE_ROOT_DEGREE-WORD_COUNT_INCREMENT)/SQUARE_ROOT_DEGREE,
		)
		next := (root + word/root) / SQUARE_ROOT_DEGREE
		for next < root {
			root = next
			next = (root + word/root) / SQUARE_ROOT_DEGREE
		}
		previous_count := destination.Count
		destination.Words[WORD_COUNT_MINIMUM] = Word(root)
		destination.Count = Word_Count(WORD_COUNT_INCREMENT)
		destination.Negative = POLARITY_NONNEGATIVE
		int_clear(destination, destination.Count, previous_count)
		return STATUS_OK
	}
	if source.Count == Word_Count(BASE_BINARY) {
		int_square_root_double_word(destination, (*Int_Double_Word)(source))
		return STATUS_OK
	}
	workspace.Division.Quotient_Count, workspace.Division.Remainder_Count = 0, 0
	current := &workspace.Integers[SQUARE_ROOT_CURRENT_INDEX]
	quotient := &workspace.Integers[SQUARE_ROOT_QUOTIENT_INDEX]
	next := &workspace.Integers[SQUARE_ROOT_NEXT_INDEX]
	division := (*Int_Division_Workspace)(&workspace.Division)
	Int_Set_Uint_64(current, Word_64(bits.CARRY_MAXIMUM))
	bit_count := int(Int_Bit_Count(source))
	initial_shift := Shift_Count(
		(bit_count + SQUARE_ROOT_DEGREE - 1) / SQUARE_ROOT_DEGREE,
	)
	shift_status := Int_Shift_Left(current, current, initial_shift)
	invariant.Always(
		shift_status == Arithmetic_Status(STATUS_OK),
		"Square-root initial power uses at most half one integer bound.",
	)
	next_is_lower := true
	for next_is_lower {
		division_status := Int_Quotient(quotient, source, current, division)
		invariant.Always(
			division_status == Divisor_Status(STATUS_OK),
			"Newton current approximation remains nonzero.",
		)
		add_status := Int_Add(next, current, quotient)
		invariant.Always(
			add_status == Arithmetic_Status(STATUS_OK),
			"Newton square-root average fits one integer bound.",
		)
		Int_Shift_Right(next, next, SQUARE_ROOT_AVERAGE_SHIFT)
		next_is_lower = Int_Compare(next, current) == ORDER_BEFORE
		if next_is_lower {
			Int_Set(current, next)
		}
	}
	Int_Set(destination, current)
	return STATUS_OK
}

// Int_Random maps bounded caller entropy into [0, maximum) without hidden retry loops.
func Int_Random(
	destination *Int,
	maximum *Int,
	source Random_Words_Unvalidated,
	workspace *Int_Random_Workspace,
) (consumed Random_Word_Count, status Random_Status) {
	defer func() {
		Random_Word_Count_Invariants(consumed, "int_random.consumed")
		Random_Status_Invariants(status, "int_random.status")
	}()
	Int_Invariants(destination, "int_random.destination")
	Int_Invariants(maximum, "int_random.maximum")
	Random_Words_Unvalidated_Invariants(source, "int_random.source")
	Int_Random_Workspace_Invariants(workspace, "int_random.workspace")
	if len(source) > RANDOM_WORD_SIZE_MAXIMUM {
		return 0, STATUS_INPUT_INVALID
	}
	if maximum.Negative == POLARITY_NEGATIVE {
		int_zero(destination, destination.Count)
		return 0, STATUS_OK
	}
	if maximum.Count == WORD_COUNT_MINIMUM {
		int_zero(destination, destination.Count)
		return 0, STATUS_OK
	}
	word_count := int(maximum.Count)
	if len(source) < word_count {
		return 0, STATUS_SOURCE_EXHAUSTED
	}
	candidate := &workspace.Integers[INT_RANDOM_VALUE_INDEX]
	candidate.Negative = POLARITY_NONNEGATIVE
	bit_count := int(Int_Bit_Count(maximum))
	high_word_bit_count := bit_count - (word_count-WORD_COUNT_INCREMENT)*WORD_BIT_COUNT
	high_mask := Word(
		uint64(bits.WORD_64_MAXIMUM) >> uint(WORD_BIT_COUNT-high_word_bit_count),
	)
	consumed_count := bytes.SLICE_SIZE_MINIMUM
	for consumed_count+word_count <= len(source) {
		for word_index := WORD_COUNT_MINIMUM; word_index < word_count; word_index++ {
			candidate.Words[word_index] = source[consumed_count+word_index]
		}
		consumed_count += word_count
		candidate.Words[word_count-WORD_COUNT_INCREMENT] &= high_mask
		candidate.Count = maximum.Count
		for candidate.Count > WORD_COUNT_MINIMUM {
			high_index := int(candidate.Count) - WORD_COUNT_INCREMENT
			if candidate.Words[high_index] != 0 {
				break
			}
			candidate.Count--
		}
		if int_compare_absolute(candidate, maximum) == ORDER_BEFORE {
			Int_Set(destination, candidate)
			return Random_Word_Count(consumed_count), STATUS_OK
		}
	}
	return Random_Word_Count(consumed_count), STATUS_SOURCE_EXHAUSTED
}

// Int_Jacobi uses binary reduction because odd denominator makes each step strictly smaller.
func Int_Jacobi(
	numerator *Int, denominator *Int, workspace *Int_Jacobi_Workspace,
) (symbol Jacobi_Symbol, status Validation_Status) {
	defer func() {
		Jacobi_Symbol_Invariants(symbol, "int_jacobi.symbol")
		Validation_Status_Invariants(status, "int_jacobi.status")
	}()
	Int_Invariants(numerator, "int_jacobi.numerator")
	Int_Invariants(denominator, "int_jacobi.denominator")
	Int_Jacobi_Workspace_Invariants(workspace, "int_jacobi.workspace")
	if word_symbol, matched := int_jacobi_word(numerator, denominator); matched {
		return word_symbol, STATUS_OK
	}
	return int_jacobi(numerator, denominator, &workspace.Integers, &workspace.Division)
}

func int_jacobi(numerator *Int, denominator *Int, integers *Jacobi_Integers,
	division_memory *Division_Memory,
) (symbol Jacobi_Symbol, status Validation_Status) {
	defer func() {
		Jacobi_Symbol_Invariants(symbol, "int_jacobi_internal.symbol")
		Validation_Status_Invariants(status, "int_jacobi_internal.status")
	}()
	Int_Invariants(numerator, "int_jacobi_internal.numerator")
	Int_Invariants(denominator, "int_jacobi_internal.denominator")
	Jacobi_Integers_Invariants(*integers, "int_jacobi_internal.integers")
	Division_Memory_Invariants(*division_memory, "int_jacobi_internal.division")
	if denominator.Count == WORD_COUNT_MINIMUM {
		return JACOBI_SYMBOL_ZERO, STATUS_INPUT_INVALID
	}
	if denominator.Words[WORD_COUNT_MINIMUM]&JACOBI_PARITY_MASK == 0 {
		return JACOBI_SYMBOL_ZERO, STATUS_INPUT_INVALID
	}
	current_numerator := &integers[JACOBI_NUMERATOR_INDEX]
	current_denominator := &integers[JACOBI_DENOMINATOR_INDEX]
	odd_numerator := &integers[JACOBI_ODD_NUMERATOR_INDEX]
	quotient := &integers[JACOBI_QUOTIENT_INDEX]
	division := (*Int_Division_Workspace)(division_memory)
	Int_Set(current_numerator, numerator)
	Int_Set(current_denominator, denominator)
	symbol = JACOBI_SYMBOL_POSITIVE
	if current_denominator.Negative == POLARITY_NEGATIVE {
		if current_numerator.Negative == POLARITY_NEGATIVE {
			symbol = JACOBI_SYMBOL_NEGATIVE
		}
		current_denominator.Negative = POLARITY_NONNEGATIVE
	}
	for current_denominator.Count > WORD_COUNT_MINIMUM {
		if current_denominator.Count == WORD_COUNT_INCREMENT {
			if current_denominator.Words[WORD_COUNT_MINIMUM] ==
				Word(WORD_COUNT_INCREMENT) {
				return symbol, STATUS_OK
			}
		}
		division_status := Int_Divide_Modulus(
			quotient, current_numerator, current_numerator,
			current_denominator, division,
		)
		invariant.Always(
			division_status == Division_Status(STATUS_OK),
			"Jacobi denominator stays nonzero and output storage stays distinct.",
		)
		if current_numerator.Count == WORD_COUNT_MINIMUM {
			return JACOBI_SYMBOL_ZERO, STATUS_OK
		}
		zero_count := Int_Trailing_Zero_Bit_Count(current_numerator)
		if Word(zero_count)&JACOBI_PARITY_MASK != 0 {
			denominator_residue := current_denominator.Words[WORD_COUNT_MINIMUM] &
				JACOBI_SUPPLEMENT_MASK
			switch denominator_residue {
			case JACOBI_SUPPLEMENT_RESIDUE_LOW, JACOBI_SUPPLEMENT_RESIDUE_HIGH:
				symbol = -symbol
			}
		}
		Int_Shift_Right(odd_numerator, current_numerator, Shift_Count(zero_count))
		if current_denominator.Words[WORD_COUNT_MINIMUM]&JACOBI_RECIPROCITY_MASK ==
			JACOBI_RECIPROCITY_MASK {
			if odd_numerator.Words[WORD_COUNT_MINIMUM]&JACOBI_RECIPROCITY_MASK ==
				JACOBI_RECIPROCITY_MASK {
				symbol = -symbol
			}
		}
		Int_Set(current_numerator, current_denominator)
		Int_Set(current_denominator, odd_numerator)
		int_zero(odd_numerator, odd_numerator.Count)
	}
	return JACOBI_SYMBOL_ZERO, STATUS_OK
}

// Int_Probably_Prime keeps probabilistic work bounded by caller entropy and explicit limits.
func Int_Probably_Prime(
	value *Int,
	options_unvalidated Primality_Options_Unvalidated,
	source Random_Words_Unvalidated,
	workspace *Int_Primality_Workspace,
) (
	probably_prime Boolean, consumed Random_Word_Count, status Primality_Status,
) {
	defer func() {
		Boolean_Invariants(probably_prime, "int_probably_prime.probably_prime")
		Random_Word_Count_Invariants(consumed, "int_probably_prime.consumed")
		Primality_Status_Invariants(status, "int_probably_prime.status")
	}()
	Int_Invariants(value, "int_probably_prime.value")
	Primality_Options_Unvalidated_Invariants(
		options_unvalidated, "int_probably_prime.options",
	)
	Random_Words_Unvalidated_Invariants(source, "int_probably_prime.source")
	Int_Primality_Workspace_Invariants(workspace, "int_probably_prime.workspace")
	options, validation_status := Primality_Options_Validate(options_unvalidated)
	if validation_status != Validation_Status(STATUS_OK) {
		return false, 0, STATUS_INPUT_INVALID
	}
	if len(source) > RANDOM_WORD_SIZE_MAXIMUM {
		return false, 0, STATUS_INPUT_INVALID
	}
	initial_quotient_count := workspace.Modular.Division.Quotient_Count
	initial_remainder_count := workspace.Modular.Division.Remainder_Count
	*workspace = Int_Primality_Workspace{}
	workspace.Modular.Division.Quotient_Count = initial_quotient_count
	workspace.Modular.Division.Remainder_Count = initial_remainder_count
	workspace.Integers[PRIMALITY_VALUE_INDEX] = *value
	if value.Negative == POLARITY_NEGATIVE {
		return false, 0, STATUS_OK
	}
	trial := int_primality_trial(&workspace.Integers)
	if trial == PRIMALITY_TRIAL_COMPOSITE {
		return false, 0, STATUS_OK
	}
	if trial == PRIMALITY_TRIAL_PRIME {
		return true, 0, STATUS_OK
	}
	miller_prime, miller_consumed, entropy_status := int_primality_miller_rabin(
		&workspace.Integers, (*Int_Modular_Workspace)(&workspace.Modular),
		(*Int_Random_Workspace)(&workspace.Random),
		options.Repetitions, Random_Words(source),
	)
	if entropy_status != Primality_Entropy_Status(STATUS_OK) {
		return false, miller_consumed, STATUS_SOURCE_EXHAUSTED
	}
	if !miller_prime {
		return false, miller_consumed, STATUS_OK
	}
	workspace.Modular.Division.Quotient_Count = initial_quotient_count
	workspace.Modular.Division.Remainder_Count = initial_remainder_count
	lucas_prime, search_status := int_primality_lucas(
		&workspace.Integers, (*Int_Modular_Workspace)(&workspace.Modular),
		&workspace.Jacobi,
		options.Parameter_Count,
	)
	if search_status != Primality_Search_Status(STATUS_OK) {
		return false, miller_consumed, STATUS_SEARCH_EXHAUSTED
	}
	return lucas_prime, miller_consumed, STATUS_OK
}

func int_primality_trial(
	integers *Primality_Integers,
) (result Primality_Trial_Result) {
	defer func() {
		Primality_Trial_Result_Invariants(result, "int_primality_trial.result")
	}()
	Primality_Integers_Invariants(*integers, "int_primality_trial.integers")
	value := &integers[PRIMALITY_VALUE_INDEX]
	if value.Count == WORD_COUNT_MINIMUM {
		return PRIMALITY_TRIAL_COMPOSITE
	}
	if value.Count == WORD_COUNT_INCREMENT {
		word := value.Words[WORD_COUNT_MINIMUM]
		if word < Word(WORD_BIT_COUNT) {
			if word < Word(BASE_BINARY) {
				return PRIMALITY_TRIAL_COMPOSITE
			}
			if word == Word(BASE_BINARY) {
				return PRIMALITY_TRIAL_PRIME
			}
			if word&JACOBI_PARITY_MASK == 0 {
				return PRIMALITY_TRIAL_COMPOSITE
			}
			factor := Word(PRIMALITY_TRIAL_FACTOR_MINIMUM)
			for factor*factor <= word {
				if word%factor == 0 {
					return PRIMALITY_TRIAL_COMPOSITE
				}
				factor += Word(BASE_BINARY)
			}
			return PRIMALITY_TRIAL_PRIME
		}
	}
	if value.Words[WORD_COUNT_MINIMUM]&JACOBI_PARITY_MASK == 0 {
		return PRIMALITY_TRIAL_COMPOSITE
	}
	factor := Primality_Trial_Factor(PRIMALITY_TRIAL_FACTOR_MINIMUM)
	for factor <= PRIMALITY_TRIAL_FACTOR_MAXIMUM {
		if int_primality_divisible(integers, factor) {
			return PRIMALITY_TRIAL_COMPOSITE
		}
		factor += Primality_Trial_Factor(BASE_BINARY)
	}
	return PRIMALITY_TRIAL_UNDETERMINED
}

func int_primality_divisible(
	integers *Primality_Integers, factor Primality_Trial_Factor,
) (divisible Boolean) {
	defer func() { Boolean_Invariants(divisible, "int_primality_divisible.divisible") }()
	Primality_Integers_Invariants(*integers, "int_primality_divisible.integers")
	Primality_Trial_Factor_Invariants(factor, "int_primality_divisible.factor")
	value := &integers[PRIMALITY_VALUE_INDEX]
	remainder := Word(0)
	factor_word := Word(factor)
	word_index := int(value.Count) - WORD_COUNT_INCREMENT
	for word_index >= WORD_COUNT_MINIMUM {
		word := value.Words[word_index]
		remainder <<= PRIMALITY_TRIAL_LIMB_BIT_COUNT
		remainder += word >> PRIMALITY_TRIAL_LIMB_BIT_COUNT
		remainder %= factor_word
		remainder <<= PRIMALITY_TRIAL_LIMB_BIT_COUNT
		remainder += word & PRIMALITY_TRIAL_LIMB_MASK
		remainder %= factor_word
		word_index--
	}
	return Boolean(remainder == 0)
}

func int_primality_miller_rabin(
	integers *Primality_Integers,
	modular *Int_Modular_Workspace,
	random *Int_Random_Workspace,
	repetitions Primality_Repetition_Count,
	source Random_Words,
) (
	prime Boolean, consumed Random_Word_Count, status Primality_Entropy_Status,
) {
	defer func() {
		Boolean_Invariants(prime, "int_primality_miller_rabin.prime")
		Random_Word_Count_Invariants(consumed, "int_primality_miller_rabin.consumed")
		Primality_Entropy_Status_Invariants(status, "int_primality_miller_rabin.status")
	}()
	Primality_Integers_Invariants(*integers, "int_primality_miller_rabin.integers")
	Int_Modular_Workspace_Invariants(modular, "int_primality_miller_rabin.modular")
	Int_Random_Workspace_Invariants(random, "int_primality_miller_rabin.random")
	Primality_Repetition_Count_Invariants(
		repetitions, "int_primality_miller_rabin.repetitions",
	)
	Random_Words_Invariants(source, "int_primality_miller_rabin.source")
	value := &integers[PRIMALITY_VALUE_INDEX]
	minus_one := &integers[PRIMALITY_MINUS_ONE_INDEX]
	bound := &integers[PRIMALITY_BOUND_INDEX]
	odd_factor := &integers[PRIMALITY_ODD_FACTOR_INDEX]
	base := &integers[PRIMALITY_BASE_INDEX]
	one := &integers[PRIMALITY_ONE_INDEX]
	two := &integers[PRIMALITY_TWO_INDEX]
	Int_Set_Uint_64(one, Word_64(bits.CARRY_MAXIMUM))
	Int_Set_Uint_64(two, Word_64(BASE_BINARY))
	minus_status := Int_Subtract(minus_one, value, one)
	invariant.Always(
		minus_status == Arithmetic_Status(STATUS_OK),
		"Probable-prime trial leaves value above one.",
	)
	minus_status = Int_Subtract(bound, minus_one, two)
	invariant.Always(
		minus_status == Arithmetic_Status(STATUS_OK),
		"Probable-prime trial leaves value above three.",
	)
	zero_count := Int_Trailing_Zero_Bit_Count(minus_one)
	Int_Set(odd_factor, minus_one)
	Int_Shift_Right(odd_factor, odd_factor, Shift_Count(zero_count))
	consumed_count := bytes.SLICE_SIZE_MINIMUM
	repetition_limit := int(repetitions)
	repetition := PRIMALITY_REPETITION_COUNT_MINIMUM
	for repetition < repetition_limit {
		base_consumed, random_status := Int_Random(
			base, bound, Random_Words_Unvalidated(source[consumed_count:]), random,
		)
		consumed_count += int(base_consumed)
		if random_status != Random_Status(STATUS_OK) {
			invariant.Always(
				random_status == Random_Status(STATUS_SOURCE_EXHAUSTED),
				"Validated primality entropy can fail only by exhaustion.",
			)
			return false, Random_Word_Count(consumed_count), STATUS_SOURCE_EXHAUSTED
		}
		add_status := Int_Add(base, base, two)
		invariant.Always(
			add_status == Arithmetic_Status(STATUS_OK),
			"Random Miller-Rabin base stays below tested value.",
		)
		if !int_primality_miller_rabin_round(integers, modular) {
			return false, Random_Word_Count(consumed_count), STATUS_OK
		}
		repetition++
	}
	Int_Set(base, two)
	if !int_primality_miller_rabin_round(integers, modular) {
		return false, Random_Word_Count(consumed_count), STATUS_OK
	}
	return true, Random_Word_Count(consumed_count), STATUS_OK
}

func int_primality_miller_rabin_round(
	integers *Primality_Integers, modular *Int_Modular_Workspace,
) (prime Boolean) {
	defer func() { Boolean_Invariants(prime, "int_primality_miller_rabin_round.prime") }()
	Primality_Integers_Invariants(*integers, "int_primality_miller_rabin_round.integers")
	Int_Modular_Workspace_Invariants(modular, "int_primality_miller_rabin_round.modular")
	value := &integers[PRIMALITY_VALUE_INDEX]
	minus_one := &integers[PRIMALITY_MINUS_ONE_INDEX]
	odd_factor := &integers[PRIMALITY_ODD_FACTOR_INDEX]
	base := &integers[PRIMALITY_BASE_INDEX]
	result := &integers[PRIMALITY_RESULT_INDEX]
	one := &integers[PRIMALITY_ONE_INDEX]
	zero_count := Int_Trailing_Zero_Bit_Count(minus_one)
	modular_status := Int_Modular_Exponent(result, base, odd_factor, value, modular)
	invariant.Always(
		modular_status == Modular_Status(STATUS_OK),
		"Miller-Rabin uses positive exponent and nonzero modulus.",
	)
	if Int_Compare(result, one) == ORDER_SAME {
		return true
	}
	if Int_Compare(result, minus_one) == ORDER_SAME {
		return true
	}
	for square_index := WORD_COUNT_INCREMENT; square_index < int(zero_count); square_index++ {
		multiply_status := Int_Modular_Multiply(
			result, result, result, value, modular,
		)
		invariant.Always(
			multiply_status == Divisor_Status(STATUS_OK),
			"Miller-Rabin tested value remains nonzero.",
		)
		if Int_Compare(result, minus_one) == ORDER_SAME {
			return true
		}
		if Int_Compare(result, one) == ORDER_SAME {
			return false
		}
	}
	return false
}

func int_primality_lucas(
	integers *Primality_Integers,
	modular *Int_Modular_Workspace,
	jacobi *Jacobi_Integers,
	parameter_count Primality_Parameter_Count,
) (prime Boolean, status Primality_Search_Status) {
	defer func() {
		Boolean_Invariants(prime, "int_primality_lucas.prime")
		Primality_Search_Status_Invariants(status, "int_primality_lucas.status")
	}()
	Primality_Integers_Invariants(*integers, "int_primality_lucas.integers")
	Int_Modular_Workspace_Invariants(modular, "int_primality_lucas.modular")
	Jacobi_Integers_Invariants(*jacobi, "int_primality_lucas.jacobi")
	Primality_Parameter_Count_Invariants(
		parameter_count, "int_primality_lucas.parameter_count",
	)
	value := &integers[PRIMALITY_VALUE_INDEX]
	Int_Set(&modular.Integers[MODULAR_MODULUS_INDEX], value)
	base := &integers[PRIMALITY_BASE_INDEX]
	delta := &integers[PRIMALITY_DELTA_INDEX]
	one := &integers[PRIMALITY_ONE_INDEX]
	initial_quotient_count := modular.Division.Quotient_Count
	initial_remainder_count := modular.Division.Remainder_Count
	parameter := Word(PRIMALITY_PARAMETER_MINIMUM)
	attempt_limit := int(parameter_count)
	attempt := PRIMALITY_PARAMETER_COUNT_MINIMUM - WORD_COUNT_INCREMENT
	for attempt < attempt_limit {
		delta_word := parameter*parameter - Word(PRIMALITY_DELTA_OFFSET)
		Int_Set_Uint_64(delta, Word_64(delta_word))
		symbol, jacobi_status := int_jacobi(
			delta, value, jacobi, &modular.Division,
		)
		invariant.Always(
			jacobi_status == Validation_Status(STATUS_OK),
			"Lucas receives one positive odd denominator.",
		)
		if symbol == JACOBI_SYMBOL_NEGATIVE {
			Int_Set_Uint_64(base, Word_64(parameter))
			reduction_status := Int_Modular_Multiply(
				base, base, one, value, modular,
			)
			invariant.Always(
				reduction_status == Divisor_Status(STATUS_OK),
				"Lucas parameter reduction uses nonzero tested value.",
			)
			modular.Division.Quotient_Count = initial_quotient_count
			modular.Division.Remainder_Count = initial_remainder_count
			return int_primality_lucas_sequence(integers, modular), STATUS_OK
		}
		if symbol == JACOBI_SYMBOL_ZERO {
			if value.Count == WORD_COUNT_INCREMENT {
				if value.Words[WORD_COUNT_MINIMUM] == parameter+Word(BASE_BINARY) {
					return true, STATUS_OK
				}
			}
			return false, STATUS_OK
		}
		parameter++
		attempt++
	}
	return false, STATUS_SEARCH_EXHAUSTED
}

func int_primality_lucas_sequence(
	integers *Primality_Integers, modular *Int_Modular_Workspace,
) (prime Boolean) {
	defer func() { Boolean_Invariants(prime, "int_primality_lucas_sequence.prime") }()
	Primality_Integers_Invariants(*integers, "int_primality_lucas_sequence.integers")
	Int_Modular_Workspace_Invariants(modular, "int_primality_lucas_sequence.modular")
	value := &integers[PRIMALITY_VALUE_INDEX]
	minus_two := &integers[PRIMALITY_BOUND_INDEX]
	odd_factor := &integers[PRIMALITY_ODD_FACTOR_INDEX]
	base := &integers[PRIMALITY_BASE_INDEX]
	current := &integers[PRIMALITY_RESULT_INDEX]
	next := &integers[PRIMALITY_NEXT_INDEX]
	left := &integers[PRIMALITY_LEFT_INDEX]
	right := &integers[PRIMALITY_RIGHT_INDEX]
	two := &integers[PRIMALITY_TWO_INDEX]
	initial_quotient_count := modular.Division.Quotient_Count
	initial_remainder_count := modular.Division.Remainder_Count
	minus_status := Int_Subtract(minus_two, value, two)
	invariant.Always(
		minus_status == Arithmetic_Status(STATUS_OK),
		"Lucas receives tested value above two.",
	)
	int_primality_lucas_odd_factor(integers)
	trailing_one_count := int(
		integers[PRIMALITY_DELTA_INDEX].Words[WORD_COUNT_MINIMUM],
	)
	Int_Set(current, two)
	Int_Set(next, base)
	bit_count := int(Int_Bit_Count(odd_factor))
	for bit_index := bit_count; bit_index >= BIT_COUNT_MINIMUM; bit_index-- {
		bit := Int_Bit(odd_factor, Bit_Index(bit_index))
		int_primality_lucas_step(integers, modular, bit)
	}
	if Int_Compare(current, two) == ORDER_SAME {
		modular.Division.Quotient_Count = initial_quotient_count
		modular.Division.Remainder_Count = initial_remainder_count
		int_primality_lucas_u_products(integers, modular)
		if Int_Compare(left, right) == ORDER_SAME {
			return true
		}
	}
	if Int_Compare(current, minus_two) == ORDER_SAME {
		modular.Division.Quotient_Count = initial_quotient_count
		modular.Division.Remainder_Count = initial_remainder_count
		int_primality_lucas_u_products(integers, modular)
		if Int_Compare(left, right) == ORDER_SAME {
			return true
		}
	}
	doubling_count := trailing_one_count - WORD_COUNT_INCREMENT
	for doubling_index := BIT_COUNT_MINIMUM; doubling_index < doubling_count; doubling_index++ {
		if current.Count == WORD_COUNT_MINIMUM {
			return true
		}
		if Int_Compare(current, two) == ORDER_SAME {
			return false
		}
		int_primality_lucas_update(integers, modular, LUCAS_UPDATE_CURRENT_SQUARE)
		Int_Set(current, right)
	}
	return false
}

func int_primality_lucas_odd_factor(
	integers *Primality_Integers,
) {
	Primality_Integers_Invariants(*integers, "int_primality_lucas_odd_factor.integers")
	value := &integers[PRIMALITY_VALUE_INDEX]
	odd_factor := &integers[PRIMALITY_ODD_FACTOR_INDEX]
	one := &integers[PRIMALITY_ONE_INDEX]
	count := &integers[PRIMALITY_DELTA_INDEX]
	trailing_one_count := BIT_COUNT_MINIMUM
	trailing_complete := false
	for word_index := WORD_COUNT_MINIMUM; word_index < int(value.Count); word_index++ {
		word := value.Words[word_index]
		for bit_index := BIT_COUNT_MINIMUM; bit_index < WORD_BIT_COUNT; bit_index++ {
			if word>>uint(bit_index)&Word(bits.CARRY_MAXIMUM) == 0 {
				trailing_complete = true
				break
			}
			trailing_one_count++
		}
		if trailing_complete {
			break
		}
	}
	Int_Set(odd_factor, value)
	Int_Shift_Right(odd_factor, odd_factor, Shift_Count(trailing_one_count))
	Int_Set_Uint_64(count, Word_64(trailing_one_count))
	add_status := Int_Add(odd_factor, odd_factor, one)
	invariant.Always(
		add_status == Arithmetic_Status(STATUS_OK),
		"Removing trailing ones leaves room for Lucas odd-factor carry.",
	)
}

func int_primality_lucas_step(
	integers *Primality_Integers, modular *Int_Modular_Workspace, bit Bit_Value,
) {
	Primality_Integers_Invariants(*integers, "int_primality_lucas_step.integers")
	Int_Modular_Workspace_Invariants(modular, "int_primality_lucas_step.modular")
	Bit_Value_Invariants(bit, "int_primality_lucas_step.bit")
	current := &integers[PRIMALITY_RESULT_INDEX]
	next := &integers[PRIMALITY_NEXT_INDEX]
	left := &integers[PRIMALITY_LEFT_INDEX]
	right := &integers[PRIMALITY_RIGHT_INDEX]
	int_primality_lucas_update(integers, modular, LUCAS_UPDATE_PRODUCT)
	if bit == BIT_SET {
		int_primality_lucas_update(integers, modular, LUCAS_UPDATE_NEXT_SQUARE)
		Int_Set(current, left)
		Int_Set(next, right)
		return
	}
	int_primality_lucas_update(integers, modular, LUCAS_UPDATE_CURRENT_SQUARE)
	Int_Set(current, right)
	Int_Set(next, left)
}

func int_primality_lucas_update(
	integers *Primality_Integers,
	modular *Int_Modular_Workspace,
	operation Lucas_Update,
) {
	Primality_Integers_Invariants(*integers, "int_primality_lucas_update.integers")
	Int_Modular_Workspace_Invariants(modular, "int_primality_lucas_update.modular")
	Lucas_Update_Invariants(operation, "int_primality_lucas_update.operation")
	value := &integers[PRIMALITY_VALUE_INDEX]
	current := &integers[PRIMALITY_RESULT_INDEX]
	next := &integers[PRIMALITY_NEXT_INDEX]
	left := current
	right := next
	destination := &integers[PRIMALITY_LEFT_INDEX]
	subtrahend := &integers[PRIMALITY_BASE_INDEX]
	if operation == LUCAS_UPDATE_CURRENT_SQUARE {
		right = current
		destination = &integers[PRIMALITY_RIGHT_INDEX]
		subtrahend = &integers[PRIMALITY_TWO_INDEX]
	}
	if operation == LUCAS_UPDATE_NEXT_SQUARE {
		left = next
		destination = &integers[PRIMALITY_RIGHT_INDEX]
		subtrahend = &integers[PRIMALITY_TWO_INDEX]
	}
	int_primality_modular_multiply(
		&Int_References{destination, left, right}, modular,
	)
	if Int_Compare(destination, subtrahend) != ORDER_BEFORE {
		subtract_status := Int_Subtract(destination, destination, subtrahend)
		invariant.Always(
			subtract_status == Arithmetic_Status(STATUS_OK),
			"Ordered Lucas subtraction remains nonnegative.",
		)
		return
	}
	difference := &integers[PRIMALITY_DIFFERENCE_INDEX]
	subtract_status := Int_Subtract(difference, subtrahend, destination)
	invariant.Always(
		subtract_status == Arithmetic_Status(STATUS_OK),
		"Reduced Lucas subtrahend difference remains bounded.",
	)
	subtract_status = Int_Subtract(destination, value, difference)
	invariant.Always(
		subtract_status == Arithmetic_Status(STATUS_OK),
		"Modular Lucas subtraction remains inside tested value.",
	)
}

func int_primality_lucas_u_products(
	integers *Primality_Integers, modular *Int_Modular_Workspace,
) {
	Primality_Integers_Invariants(*integers, "int_primality_lucas_u_products.integers")
	Int_Modular_Workspace_Invariants(modular, "int_primality_lucas_u_products.modular")
	base := &integers[PRIMALITY_BASE_INDEX]
	current := &integers[PRIMALITY_RESULT_INDEX]
	next := &integers[PRIMALITY_NEXT_INDEX]
	left := &integers[PRIMALITY_LEFT_INDEX]
	right := &integers[PRIMALITY_RIGHT_INDEX]
	two := &integers[PRIMALITY_TWO_INDEX]
	int_primality_modular_multiply(
		&Int_References{left, base, current}, modular,
	)
	int_primality_modular_multiply(
		&Int_References{right, two, next}, modular,
	)
}

// Rat_Set_Int_64 writes one machine integer with implicit denominator one.
func Rat_Set_Int_64(destination *Rat, value Int_64) {
	Rat_Invariants(destination, "rat_set_int_64.destination")
	Int_64_Invariants(value, "rat_set_int_64.value")
	Int_Set_Int_64(&destination.Integers[RAT_NUMERATOR_INDEX], value)
	denominator := &destination.Integers[RAT_DENOMINATOR_INDEX]
	int_zero(denominator, denominator.Count)
}

// Rat_Set_Uint_64 writes one machine word with implicit denominator one.
func Rat_Set_Uint_64(destination *Rat, value Word_64) {
	Rat_Invariants(destination, "rat_set_uint_64.destination")
	Word_64_Invariants(value, "rat_set_uint_64.value")
	Int_Set_Uint_64(&destination.Integers[RAT_NUMERATOR_INDEX], value)
	denominator := &destination.Integers[RAT_DENOMINATOR_INDEX]
	int_zero(denominator, denominator.Count)
}

// Rat_Set_Float_64_Bits stores exact finite binary64 encoding and rejects nonfinite input.
func Rat_Set_Float_64_Bits(
	destination *Rat, encoding Float_64_Bits,
) (status Validation_Status) {
	defer func() { Validation_Status_Invariants(status, "rat_set_float_64_bits.status") }()
	Rat_Invariants(destination, "rat_set_float_64_bits.destination")
	Float_64_Bits_Invariants(encoding, "rat_set_float_64_bits.encoding")
	encoded := uint64(encoding)
	encoded_exponent := int(
		(encoded >> FLOAT_64_EXPONENT_SHIFT) & FLOAT_64_EXPONENT_MASK,
	)
	if encoded_exponent == FLOAT_64_EXPONENT_MASK {
		return STATUS_INPUT_INVALID
	}
	mantissa := encoded & FLOAT_64_MANTISSA_MASK
	exponent := FLOAT_64_SUBNORMAL_EXPONENT
	if encoded_exponent != 0 {
		mantissa |= FLOAT_64_HIDDEN_MANTISSA_BIT
		exponent = encoded_exponent - FLOAT_64_EXPONENT_BIAS
	}
	if mantissa == 0 {
		Rat_Set_Uint_64(destination, Word_64(bits.WORD_64_MINIMUM))
		return STATUS_OK
	}
	shift := FLOAT_64_MANTISSA_BIT_COUNT - exponent
	for mantissa&uint64(bits.CARRY_MAXIMUM) == 0 && shift > 0 {
		mantissa >>= 1
		shift--
	}
	numerator := &destination.Integers[RAT_NUMERATOR_INDEX]
	denominator := &destination.Integers[RAT_DENOMINATOR_INDEX]
	Int_Set_Uint_64(numerator, Word_64(mantissa))
	int_zero(denominator, denominator.Count)
	if shift > 0 {
		Int_Set_Uint_64(denominator, Word_64(bits.CARRY_MAXIMUM))
		shift_status := Int_Shift_Left(denominator, denominator, Shift_Count(shift))
		invariant.Always(
			shift_status == Arithmetic_Status(STATUS_OK),
			"Binary64 denominator width stays inside rational component bound.",
		)
	} else {
		shift_status := Int_Shift_Left(numerator, numerator, Shift_Count(-shift))
		invariant.Always(
			shift_status == Arithmetic_Status(STATUS_OK),
			"Binary64 numerator width stays inside rational component bound.",
		)
	}
	if encoded&FLOAT_64_SIGN_MASK != 0 {
		numerator.Negative = POLARITY_NEGATIVE
	}
	return STATUS_OK
}

// Rat_Set_Int rejects integer components beyond rational cross-product bound.
func Rat_Set_Int(destination *Rat, source *Int) (status Arithmetic_Status) {
	defer func() { Arithmetic_Status_Invariants(status, "rat_set_int.status") }()
	Rat_Invariants(destination, "rat_set_int.destination")
	Int_Invariants(source, "rat_set_int.source")
	if source.Count > RAT_WORD_COUNT_MAXIMUM {
		return STATUS_VALUE_OVERFLOW
	}
	Int_Set(&destination.Integers[RAT_NUMERATOR_INDEX], source)
	denominator := &destination.Integers[RAT_DENOMINATOR_INDEX]
	int_zero(denominator, denominator.Count)
	return STATUS_OK
}

// Rat_Set copies normalized inline components without sharing mutable state.
func Rat_Set(destination *Rat, source *Rat) {
	Rat_Invariants(destination, "rat_set.destination")
	Rat_Invariants(source, "rat_set.source")
	if destination == source {
		return
	}
	Int_Set(
		&destination.Integers[RAT_NUMERATOR_INDEX],
		&source.Integers[RAT_NUMERATOR_INDEX],
	)
	Int_Set(
		&destination.Integers[RAT_DENOMINATOR_INDEX],
		&source.Integers[RAT_DENOMINATOR_INDEX],
	)
}

// Rat_Parse_Fraction parses one exact integer quotient without destination mutation on failure.
func Rat_Parse_Fraction(
	destination *Rat,
	source Rat_Parse_Fraction_Text_Unvalidated,
	workspace *Rat_Parse_Fraction_Workspace,
) (status Parse_Status) {
	defer func() { Parse_Status_Invariants(status, "rat_parse_fraction.status") }()
	Rat_Invariants(destination, "rat_parse_fraction.destination")
	Rat_Parse_Fraction_Text_Unvalidated_Invariants(source, "rat_parse_fraction.source")
	Rat_Parse_Fraction_Workspace_Invariants(workspace, "rat_parse_fraction.workspace")
	if len(source) > RAT_PARSE_FRACTION_TEXT_SIZE_MAXIMUM {
		return STATUS_INPUT_INVALID
	}
	separator_index := RAT_PARSE_SEPARATOR_INDEX_ABSENT
	for index, character := range source {
		if character == '/' {
			if separator_index != RAT_PARSE_SEPARATOR_INDEX_ABSENT {
				return STATUS_INPUT_INVALID
			}
			separator_index = index
		}
	}
	if separator_index <= bytes.SLICE_SIZE_MINIMUM {
		return STATUS_INPUT_INVALID
	}
	denominator_index := separator_index + RATIONAL_SEPARATOR_BYTE_COUNT
	if denominator_index >= len(source) {
		return STATUS_INPUT_INVALID
	}
	switch source[denominator_index] {
	case '-', '+':
		return STATUS_INPUT_INVALID
	}
	numerator := &workspace.Integers[RAT_PARSE_FRACTION_NUMERATOR_INDEX]
	denominator := &workspace.Integers[RAT_PARSE_FRACTION_DENOMINATOR_INDEX]
	_, numerator_status := Int_Parse(
		numerator, Text_Unvalidated(source[:separator_index]), BASE_AUTOMATIC,
		&workspace.Parse[RAT_PARSE_NUMERATOR_WORKSPACE_INDEX],
	)
	if numerator_status != Parse_Status(STATUS_OK) {
		return numerator_status
	}
	_, denominator_status := Int_Parse(
		denominator, Text_Unvalidated(source[denominator_index:]), BASE_AUTOMATIC,
		&workspace.Parse[RAT_PARSE_DENOMINATOR_WORKSPACE_INDEX],
	)
	if denominator_status != Parse_Status(STATUS_OK) {
		return denominator_status
	}
	if denominator.Count == WORD_COUNT_MINIMUM {
		return STATUS_INPUT_INVALID
	}
	rational := (*Rat_Workspace)(&workspace.Rational)
	rational_status := Rat_Set_Fraction(
		destination, numerator, denominator, rational,
	)
	if rational_status == Rat_Division_Status(STATUS_VALUE_OVERFLOW) {
		return STATUS_VALUE_OVERFLOW
	}
	invariant.Always(
		rational_status == Rat_Division_Status(STATUS_OK),
		"Parsed nonzero denominator keeps rational normalization defined.",
	)
	return STATUS_OK
}

// Rat_Parse keeps stdlib rational syntax without permitting unbounded scanning or growth.
func Rat_Parse(
	destination *Rat,
	source Rat_Parse_Text_Unvalidated,
	workspace *Rat_Parse_Workspace,
) (status Parse_Status) {
	defer func() { Parse_Status_Invariants(status, "rat_parse.status") }()
	Rat_Invariants(destination, "rat_parse.destination")
	Rat_Parse_Text_Unvalidated_Invariants(source, "rat_parse.source")
	Rat_Parse_Workspace_Invariants(workspace, "rat_parse.workspace")
	if len(source) == bytes.SLICE_SIZE_MINIMUM {
		return STATUS_INPUT_INVALID
	}
	if len(source) > RAT_PARSE_TEXT_SIZE_MAXIMUM {
		return STATUS_INPUT_INVALID
	}
	for _, character := range source {
		if character != '/' {
			continue
		}
		if len(source) > RAT_PARSE_FRACTION_TEXT_SIZE_MAXIMUM {
			return STATUS_INPUT_INVALID
		}
		fraction := (*Rat_Parse_Fraction_Workspace)(workspace)
		return Rat_Parse_Fraction(
			destination, Rat_Parse_Fraction_Text_Unvalidated(source),
			fraction,
		)
	}
	references := Rat_Parse_Workspace_References{workspace}
	return rat_parse_float(destination, Rat_Parse_Text(source), &references)
}

func rat_parse_float(
	destination *Rat, source Rat_Parse_Text, references *Rat_Parse_Workspace_References,
) (status Parse_Status) {
	defer func() { Parse_Status_Invariants(status, "rat_parse_float.status") }()
	Rat_Invariants(destination, "rat_parse_float.destination")
	Rat_Parse_Text_Invariants(source, "rat_parse_float.source")
	Rat_Parse_Workspace_References_Invariants(references, "rat_parse_float.references")
	workspace := references[RAT_PARSE_WORKSPACE_REFERENCE_INDEX]
	prefix := rat_parse_prefix(source)
	fraction := (*Rat_Parse_Fraction_Workspace)(workspace)
	mantissa, mantissa_status := rat_parse_mantissa(
		source, prefix, &fraction.Parse[RAT_PARSE_NUMERATOR_WORKSPACE_INDEX],
	)
	if mantissa_status == Parse_Status(STATUS_INPUT_INVALID) {
		return STATUS_INPUT_INVALID
	}
	exponent := Rat_Parse_Exponent(0)
	exponent_base := Rat_Parse_Exponent_Base(BASE_DECIMAL)
	if int(mantissa.End) < len(source) {
		character := source[mantissa.End]
		switch character {
		case 'e', 'E':
			exponent_base = Rat_Parse_Exponent_Base(BASE_DECIMAL)
		case 'p', 'P':
			exponent_base = Rat_Parse_Exponent_Base(BASE_BINARY)
		default:
			return STATUS_INPUT_INVALID
		}
		exponent_start := int(mantissa.End) + RAT_PARSE_EXPONENT_MARKER_BYTE_COUNT
		var exponent_status Validation_Status
		exponent, exponent_status = rat_parse_exponent(
			Rat_Parse_Exponent_Text(source[exponent_start:]),
		)
		if exponent_status != Validation_Status(STATUS_OK) {
			return STATUS_INPUT_INVALID
		}
	}
	if mantissa_status == Parse_Status(STATUS_VALUE_OVERFLOW) {
		return STATUS_VALUE_OVERFLOW
	}
	fraction_references := Rat_Parse_Fraction_References{fraction}
	rat_parse_mantissa_commit(&fraction_references, mantissa)
	numerator := &fraction.Integers[RAT_PARSE_FRACTION_NUMERATOR_INDEX]
	exponents, exponent_status := rat_parse_exponents(mantissa, exponent_base, exponent)
	if numerator.Count == WORD_COUNT_MINIMUM {
		Rat_Set_Uint_64(destination, Word_64(bits.WORD_64_MINIMUM))
		return STATUS_OK
	}
	if exponent_status != Arithmetic_Status(STATUS_OK) {
		return Parse_Status(exponent_status)
	}
	denominator := &fraction.Integers[RAT_PARSE_FRACTION_DENOMINATOR_INDEX]
	int_set_word(denominator, Word(bits.CARRY_MAXIMUM), POLARITY_NONNEGATIVE)
	scale_status := rat_parse_scale(&fraction.Integers, exponents)
	if scale_status != Arithmetic_Status(STATUS_OK) {
		return Parse_Status(scale_status)
	}
	rational := (*Rat_Workspace)(&fraction.Rational)
	normalization_status := Rat_Set_Fraction(
		destination, numerator, denominator, rational,
	)
	if normalization_status == Rat_Division_Status(STATUS_VALUE_OVERFLOW) {
		return STATUS_VALUE_OVERFLOW
	}
	invariant.Always(
		normalization_status == Rat_Division_Status(STATUS_OK),
		"Parsed floating denominator remains positive and nonzero.",
	)
	return STATUS_OK
}

func rat_parse_mantissa_commit(
	references *Rat_Parse_Fraction_References, mantissa Rat_Parse_Mantissa,
) {
	Rat_Parse_Fraction_References_Invariants(references, "rat_parse_mantissa_commit.references")
	Rat_Parse_Mantissa_Invariants(mantissa, "rat_parse_mantissa_commit.mantissa")
	workspace := references[RAT_PARSE_FRACTION_REFERENCE_INDEX]
	numerator := &workspace.Integers[RAT_PARSE_FRACTION_NUMERATOR_INDEX]
	words := &workspace.Parse[RAT_PARSE_NUMERATOR_WORKSPACE_INDEX].Words
	for index := WORD_COUNT_MINIMUM; index < int(mantissa.Count); index++ {
		numerator.Words[index] = words[index]
	}
	numerator.Count = mantissa.Count
	numerator.Negative = POLARITY_NONNEGATIVE
	if bool(mantissa.Prefix.Negative) {
		if numerator.Count > WORD_COUNT_MINIMUM {
			numerator.Negative = POLARITY_NEGATIVE
		}
	}
}

func rat_parse_prefix(source Rat_Parse_Text) (prefix Rat_Parse_Prefix) {
	defer func() { Rat_Parse_Prefix_Invariants(prefix, "rat_parse_prefix.prefix") }()
	Rat_Parse_Text_Invariants(source, "rat_parse_prefix.source")
	prefix.Base = Rat_Parse_Mantissa_Base(BASE_DECIMAL)
	source_index := bytes.SLICE_SIZE_MINIMUM
	if source[source_index] == '-' {
		prefix.Negative = true
		source_index += SIGN_BYTE_COUNT_MAXIMUM
	} else if source[source_index] == '+' {
		source_index += SIGN_BYTE_COUNT_MAXIMUM
	}
	if source_index < len(source) {
		prefix.Start = Parse_Text_Index(source_index)
		if source_index+SIGN_BYTE_COUNT_MAXIMUM < len(source) {
			if source[source_index] == '0' {
				marker := source[source_index+SIGN_BYTE_COUNT_MAXIMUM]
				switch marker {
				case 'b', 'B':
					prefix.Base = Rat_Parse_Mantissa_Base(BASE_BINARY)
					prefix.Prefixed = true
				case 'o', 'O':
					prefix.Base = Rat_Parse_Mantissa_Base(BASE_OCTAL)
					prefix.Prefixed = true
				case 'x', 'X':
					prefix.Base = Rat_Parse_Mantissa_Base(BASE_HEXADECIMAL)
					prefix.Prefixed = true
				}
			}
		}
	}
	if prefix.Prefixed {
		prefix.Start += Parse_Text_Index(BASE_PREFIX_BYTE_COUNT)
	}
	return prefix
}

func rat_parse_mantissa(
	source Rat_Parse_Text,
	prefix Rat_Parse_Prefix,
	workspace *Int_Parse_Workspace,
) (mantissa Rat_Parse_Mantissa, status Parse_Status) {
	defer func() {
		Rat_Parse_Mantissa_Invariants(mantissa, "rat_parse_mantissa.mantissa")
		Parse_Status_Invariants(status, "rat_parse_mantissa.status")
	}()
	Rat_Parse_Text_Invariants(source, "rat_parse_mantissa.source")
	Rat_Parse_Prefix_Invariants(prefix, "rat_parse_mantissa.prefix")
	Int_Parse_Workspace_Invariants(workspace, "rat_parse_mantissa.workspace")
	mantissa.Prefix = prefix
	mantissa.End = Rat_Parse_Source_Index(len(source))
	count := Word_Count(WORD_COUNT_MINIMUM)
	digit_seen, radix_seen := false, false
	separator_seen := false
	previous_digit := bool(prefix.Prefixed)
	source_index := int(prefix.Start)
	magnitude_overflow := false
	for source_index < len(source) {
		character := source[source_index]
		digit, valid := int_parse_digit(Parse_Character(character), Base(prefix.Base))
		if valid {
			if !magnitude_overflow {
				accumulation := int_parse_accumulate(
					workspace, &count, Base(prefix.Base), digit,
				)
				magnitude_overflow = accumulation != Arithmetic_Status(STATUS_OK)
			}
			if radix_seen {
				mantissa.Fractional_Digits++
			}
			digit_seen, separator_seen, previous_digit = true, false, true
			source_index++
			continue
		}
		if character == '_' {
			if !previous_digit {
				return mantissa, STATUS_INPUT_INVALID
			}
			if separator_seen {
				return mantissa, STATUS_INPUT_INVALID
			}
			separator_seen, previous_digit = true, false
			source_index++
			continue
		}
		if character == '.' {
			if radix_seen {
				return mantissa, STATUS_INPUT_INVALID
			}
			if separator_seen {
				return mantissa, STATUS_INPUT_INVALID
			}
			radix_seen, previous_digit = true, false
			source_index++
			continue
		}
		break
	}
	if !digit_seen {
		return mantissa, STATUS_INPUT_INVALID
	}
	if separator_seen {
		return mantissa, STATUS_INPUT_INVALID
	}
	mantissa.Count = count
	mantissa.End = Rat_Parse_Source_Index(source_index)
	if magnitude_overflow {
		return mantissa, STATUS_VALUE_OVERFLOW
	}
	return mantissa, STATUS_OK
}

func rat_parse_exponent(
	source Rat_Parse_Exponent_Text,
) (exponent Rat_Parse_Exponent, status Validation_Status) {
	defer func() {
		Rat_Parse_Exponent_Invariants(exponent, "rat_parse_exponent.exponent")
		Validation_Status_Invariants(status, "rat_parse_exponent.status")
	}()
	Rat_Parse_Exponent_Text_Invariants(source, "rat_parse_exponent.source")
	source_index := bytes.SLICE_SIZE_MINIMUM
	negative := false
	if source_index < len(source) {
		if source[source_index] == '-' {
			negative = true
			source_index += SIGN_BYTE_COUNT_MAXIMUM
		} else if source[source_index] == '+' {
			source_index += SIGN_BYTE_COUNT_MAXIMUM
		}
	}
	limit := uint64(bits.INTEGER_64_MAXIMUM)
	if negative {
		limit = INT_64_NEGATIVE_MAGNITUDE_MAXIMUM
	}
	magnitude := uint64(bits.WORD_64_MINIMUM)
	digit_seen, separator_seen := false, false
	for source_index < len(source) {
		character := source[source_index]
		if character == '_' {
			if !digit_seen {
				return 0, STATUS_INPUT_INVALID
			}
			if separator_seen {
				return 0, STATUS_INPUT_INVALID
			}
			separator_seen = true
			source_index++
			continue
		}
		if character < '0' {
			return 0, STATUS_INPUT_INVALID
		}
		if character > '9' {
			return 0, STATUS_INPUT_INVALID
		}
		digit := uint64(character - '0')
		if magnitude > (limit-digit)/uint64(BASE_DECIMAL) {
			return 0, STATUS_INPUT_INVALID
		}
		magnitude = magnitude*uint64(BASE_DECIMAL) + digit
		digit_seen, separator_seen = true, false
		source_index++
	}
	if !digit_seen {
		return 0, STATUS_INPUT_INVALID
	}
	if separator_seen {
		return 0, STATUS_INPUT_INVALID
	}
	if !negative {
		return Rat_Parse_Exponent(magnitude), STATUS_OK
	}
	if magnitude == INT_64_NEGATIVE_MAGNITUDE_MAXIMUM {
		return Rat_Parse_Exponent(bits.INTEGER_64_MINIMUM), STATUS_OK
	}
	return Rat_Parse_Exponent(-int64(magnitude)), STATUS_OK
}

func rat_parse_exponents(
	mantissa Rat_Parse_Mantissa,
	exponent_base Rat_Parse_Exponent_Base,
	exponent Rat_Parse_Exponent,
) (result Rat_Parse_Exponents, status Arithmetic_Status) {
	defer func() {
		Rat_Parse_Exponents_Invariants(result, "rat_parse_exponents.result")
		Arithmetic_Status_Invariants(status, "rat_parse_exponents.status")
	}()
	Rat_Parse_Mantissa_Invariants(mantissa, "rat_parse_exponents.mantissa")
	Rat_Parse_Exponent_Base_Invariants(exponent_base, "rat_parse_exponents.exponent_base")
	Rat_Parse_Exponent_Invariants(exponent, "rat_parse_exponents.exponent")
	fraction := -int64(mantissa.Fractional_Digits)
	switch Base(mantissa.Prefix.Base) {
	case BASE_DECIMAL:
		result[RAT_PARSE_EXPONENT_2_INDEX] = fraction
		result[RAT_PARSE_EXPONENT_5_INDEX] = fraction
	case BASE_BINARY:
		result[RAT_PARSE_EXPONENT_2_INDEX] = fraction
	case BASE_OCTAL, BASE_HEXADECIMAL:
		digit_bits := int64(BIT_COUNT_MINIMUM)
		for radix := Base(mantissa.Prefix.Base); radix > Base(bits.CARRY_MAXIMUM); {
			digit_bits++
			radix /= BASE_BINARY
		}
		result[RAT_PARSE_EXPONENT_2_INDEX] = fraction * digit_bits
	}
	value := int64(exponent)
	if Base(exponent_base) == BASE_DECIMAL {
		if value > 0 {
			if result[RAT_PARSE_EXPONENT_5_INDEX] > bits.INTEGER_64_MAXIMUM-value {
				return result, STATUS_VALUE_OVERFLOW
			}
		}
		if value < 0 {
			if result[RAT_PARSE_EXPONENT_5_INDEX] < bits.INTEGER_64_MINIMUM-value {
				return result, STATUS_VALUE_OVERFLOW
			}
		}
		result[RAT_PARSE_EXPONENT_5_INDEX] += value
	}
	if value > 0 {
		if result[RAT_PARSE_EXPONENT_2_INDEX] > bits.INTEGER_64_MAXIMUM-value {
			return result, STATUS_VALUE_OVERFLOW
		}
	}
	if value < 0 {
		if result[RAT_PARSE_EXPONENT_2_INDEX] < bits.INTEGER_64_MINIMUM-value {
			return result, STATUS_VALUE_OVERFLOW
		}
	}
	result[RAT_PARSE_EXPONENT_2_INDEX] += value
	return result, STATUS_OK
}

func rat_parse_scale(
	integers *Rat_Parse_Fraction_Integers, exponents Rat_Parse_Exponents,
) (status Arithmetic_Status) {
	defer func() { Arithmetic_Status_Invariants(status, "rat_parse_scale.status") }()
	Rat_Parse_Fraction_Integers_Invariants(*integers, "rat_parse_scale.integers")
	Rat_Parse_Exponents_Invariants(exponents, "rat_parse_scale.exponents")
	maximum := int64(RAT_PARSE_EXPONENT_MAGNITUDE_MAXIMUM)
	exponent_2 := exponents[RAT_PARSE_EXPONENT_2_INDEX]
	exponent_5 := exponents[RAT_PARSE_EXPONENT_5_INDEX]
	if exponent_2 < -maximum {
		return STATUS_VALUE_OVERFLOW
	}
	if exponent_2 > maximum {
		return STATUS_VALUE_OVERFLOW
	}
	if exponent_5 < -maximum {
		return STATUS_VALUE_OVERFLOW
	}
	if exponent_5 > maximum {
		return STATUS_VALUE_OVERFLOW
	}
	component := Rat_Parse_Power_Component(RAT_PARSE_FRACTION_NUMERATOR_INDEX)
	if exponent_5 < 0 {
		component = Rat_Parse_Power_Component(RAT_PARSE_FRACTION_DENOMINATOR_INDEX)
		exponent_5 = -exponent_5
	}
	if exponent_5 > 0 {
		status = rat_parse_multiply_five(
			integers, component, Rat_Parse_Power_Count(exponent_5),
		)
		if status != Arithmetic_Status(STATUS_OK) {
			return status
		}
	}
	component = Rat_Parse_Power_Component(RAT_PARSE_FRACTION_NUMERATOR_INDEX)
	if exponent_2 < 0 {
		component = Rat_Parse_Power_Component(RAT_PARSE_FRACTION_DENOMINATOR_INDEX)
		exponent_2 = -exponent_2
	}
	target := &integers[component]
	return Int_Shift_Left(target, target, Shift_Count(exponent_2))
}

func rat_parse_multiply_five(
	integers *Rat_Parse_Fraction_Integers,
	component Rat_Parse_Power_Component,
	count Rat_Parse_Power_Count,
) (status Arithmetic_Status) {
	defer func() { Arithmetic_Status_Invariants(status, "rat_parse_multiply_five.status") }()
	Rat_Parse_Fraction_Integers_Invariants(
		*integers, "rat_parse_multiply_five.integers",
	)
	Rat_Parse_Power_Component_Invariants(component, "rat_parse_multiply_five.component")
	Rat_Parse_Power_Count_Invariants(count, "rat_parse_multiply_five.count")
	value := &integers[component]
	for count > 0 {
		carry := uint64(bits.WORD_64_MINIMUM)
		for index := WORD_COUNT_MINIMUM; index < int(value.Count); index++ {
			word := uint64(value.Words[index])
			low_product := (word&TEXT_DIVISION_LIMB_MASK)*
				uint64(RAT_DECIMAL_PRIME_FACTOR) + carry
			low := low_product & TEXT_DIVISION_LIMB_MASK
			carry = low_product >> TEXT_DIVISION_LIMB_BIT_COUNT
			high_product := (word>>TEXT_DIVISION_LIMB_BIT_COUNT)*
				uint64(RAT_DECIMAL_PRIME_FACTOR) + carry
			high := high_product & TEXT_DIVISION_LIMB_MASK
			carry = high_product >> TEXT_DIVISION_LIMB_BIT_COUNT
			value.Words[index] = Word(high<<TEXT_DIVISION_LIMB_BIT_COUNT | low)
		}
		if carry != 0 {
			if value.Count == WORD_COUNT_MAXIMUM {
				return STATUS_VALUE_OVERFLOW
			}
			value.Words[value.Count] = Word(carry)
			value.Count++
		}
		count--
	}
	return STATUS_OK
}

// Rat_Numerator_Into copies signed numerator into caller integer storage.
func Rat_Numerator_Into(destination *Int, value *Rat) {
	Int_Invariants(destination, "rat_numerator_into.destination")
	Rat_Invariants(value, "rat_numerator_into.value")
	Int_Set(destination, &value.Integers[RAT_NUMERATOR_INDEX])
}

// Rat_Denominator_Into expands implicit denominator one into caller integer storage.
func Rat_Denominator_Into(destination *Int, value *Rat) {
	Int_Invariants(destination, "rat_denominator_into.destination")
	Rat_Invariants(value, "rat_denominator_into.value")
	denominator := &value.Integers[RAT_DENOMINATOR_INDEX]
	if denominator.Count == WORD_COUNT_MINIMUM {
		int_set_word(destination, Word(bits.CARRY_MAXIMUM), POLARITY_NONNEGATIVE)
		return
	}
	Int_Set(destination, denominator)
}

// Rat_Sign returns numerator sign because denominator is positive.
func Rat_Sign(value *Rat) (sign Sign) {
	defer func() { Sign_Invariants(sign, "rat_sign.sign") }()
	Rat_Invariants(value, "rat_sign.value")
	numerator := &value.Integers[RAT_NUMERATOR_INDEX]
	if numerator.Count == WORD_COUNT_MINIMUM {
		return SIGN_ZERO
	}
	if numerator.Negative == POLARITY_NEGATIVE {
		return SIGN_NEGATIVE
	}
	return SIGN_POSITIVE
}

// Rat_Is_Integer reports whether stored denominator represents one.
func Rat_Is_Integer(value *Rat) (integer Boolean) {
	defer func() { Boolean_Invariants(integer, "rat_is_integer.integer") }()
	Rat_Invariants(value, "rat_is_integer.value")
	denominator := &value.Integers[RAT_DENOMINATOR_INDEX]
	if denominator.Count == WORD_COUNT_MINIMUM {
		return true
	}
	if denominator.Count == WORD_COUNT_INCREMENT {
		if denominator.Words[WORD_COUNT_MINIMUM] == Word(bits.CARRY_MAXIMUM) {
			return true
		}
	}
	return false
}

// Rat_Set_Fraction reduces one signed fraction before committing bounded components.
func Rat_Set_Fraction(
	destination *Rat, numerator *Int, denominator *Int, workspace *Rat_Workspace,
) (status Rat_Division_Status) {
	defer func() { Rat_Division_Status_Invariants(status, "rat_set_fraction.status") }()
	Rat_Invariants(destination, "rat_set_fraction.destination")
	Int_Invariants(numerator, "rat_set_fraction.numerator")
	Int_Invariants(denominator, "rat_set_fraction.denominator")
	Rat_Workspace_Invariants(workspace, "rat_set_fraction.workspace")
	result_numerator := &workspace.Integers[RAT_RESULT_NUMERATOR_INDEX]
	result_denominator := &workspace.Integers[RAT_RESULT_DENOMINATOR_INDEX]
	Int_Set(result_numerator, numerator)
	Int_Set(result_denominator, denominator)
	return Rat_Division_Status(rat_normalize(destination, workspace))
}

// Rat_Set_Fraction_64 normalizes two machine integers without local escaping storage.
func Rat_Set_Fraction_64(
	destination *Rat, numerator Int_64, denominator Int_64, workspace *Rat_Workspace,
) (status Divisor_Status) {
	defer func() { Divisor_Status_Invariants(status, "rat_set_fraction_64.status") }()
	Rat_Invariants(destination, "rat_set_fraction_64.destination")
	Int_64_Invariants(numerator, "rat_set_fraction_64.numerator")
	Int_64_Invariants(denominator, "rat_set_fraction_64.denominator")
	Rat_Workspace_Invariants(workspace, "rat_set_fraction_64.workspace")
	result_numerator := &workspace.Integers[RAT_RESULT_NUMERATOR_INDEX]
	result_denominator := &workspace.Integers[RAT_RESULT_DENOMINATOR_INDEX]
	Int_Set_Int_64(result_numerator, numerator)
	Int_Set_Int_64(result_denominator, denominator)
	normalization_status := rat_normalize(destination, workspace)
	invariant.Always(
		normalization_status != Rat_Division_Status(STATUS_VALUE_OVERFLOW),
		"A machine fraction fits every rational component bound.",
	)
	return Divisor_Status(normalization_status)
}

// Rat_Text_Into writes explicit numerator and denominator through caller storage.
func Rat_Text_Into(
	destination Text, value *Rat, workspace *Rat_Text_Workspace,
) (count Rat_Fraction_Text_Count, status Destination_Status) {
	defer func() {
		Rat_Fraction_Text_Count_Invariants(count, "rat_text_into.count")
		Destination_Status_Invariants(status, "rat_text_into.status")
	}()
	Text_Invariants(destination, "rat_text_into.destination")
	Rat_Invariants(value, "rat_text_into.value")
	Rat_Text_Workspace_Invariants(workspace, "rat_text_into.workspace")
	text_count, status := rat_text_into(
		destination, value, workspace, RAT_TEXT_FORM_FRACTION,
	)
	return Rat_Fraction_Text_Count(text_count), status
}

// Rat_Rational_Text_Into omits denominator one without changing noninteger text.
func Rat_Rational_Text_Into(
	destination Text, value *Rat, workspace *Rat_Text_Workspace,
) (count Rat_Text_Count, status Destination_Status) {
	defer func() {
		Rat_Text_Count_Invariants(count, "rat_rational_text_into.count")
		Destination_Status_Invariants(status, "rat_rational_text_into.status")
	}()
	Text_Invariants(destination, "rat_rational_text_into.destination")
	Rat_Invariants(value, "rat_rational_text_into.value")
	Rat_Text_Workspace_Invariants(workspace, "rat_rational_text_into.workspace")
	return rat_text_into(destination, value, workspace, RAT_TEXT_FORM_RATIONAL)
}

// Rat_Float_Text_Into rounds to fixed decimal precision through caller storage.
func Rat_Float_Text_Into(
	destination Text,
	value *Rat,
	precision Rat_Precision,
	workspace *Rat_Float_Text_Workspace,
) (count Rat_Text_Count, status Destination_Status) {
	defer func() {
		Rat_Text_Count_Invariants(count, "rat_float_text_into.count")
		Destination_Status_Invariants(status, "rat_float_text_into.status")
	}()
	Text_Invariants(destination, "rat_float_text_into.destination")
	Rat_Invariants(value, "rat_float_text_into.value")
	Rat_Precision_Invariants(precision, "rat_float_text_into.precision")
	Rat_Float_Text_Workspace_Invariants(workspace, "rat_float_text_into.workspace")
	rat_float_text_components(value, precision, workspace)
	integer_digits := Rat_Component_Digit_Count(int_text_digits(
		&workspace.Integers[RAT_FLOAT_INTEGER_INDEX], BASE_DECIMAL,
		&workspace.Text[RAT_FLOAT_TEXT_INTEGER_PART_INDEX],
	))
	fractional_digits := Rat_Component_Digit_Count(RAT_COMPONENT_DIGIT_COUNT_MINIMUM)
	if precision > RAT_PRECISION_MINIMUM {
		fractional_digits = Rat_Component_Digit_Count(int_text_digits(
			&workspace.Integers[RAT_FLOAT_FRACTION_INDEX], BASE_DECIMAL,
			&workspace.Text[RAT_FLOAT_TEXT_FRACTIONAL_PART_INDEX],
		))
		invariant.Always(
			int(fractional_digits) <= int(precision),
			"Rounded rational fraction stays inside requested decimal precision.",
		)
	}
	required := int(integer_digits)
	if value.Integers[RAT_NUMERATOR_INDEX].Negative == POLARITY_NEGATIVE {
		required += SIGN_BYTE_COUNT_MAXIMUM
	}
	if precision > RAT_PRECISION_MINIMUM {
		required += DECIMAL_POINT_BYTE_COUNT + int(precision)
	}
	invariant.Always(
		required <= RAT_TEXT_SIZE_MAXIMUM,
		"Fixed rational text stays inside two decimal component bounds.",
	)
	if len(destination) < required {
		return Rat_Text_Count(required), STATUS_DESTINATION_TOO_SMALL
	}
	rat_float_text_write(
		Rat_Float_Text_Destination(destination[:required]), value, precision,
		integer_digits, fractional_digits, &workspace.Text,
	)
	return Rat_Text_Count(required), STATUS_OK
}

// Rat_Float_Precision reports non-repeating decimal places and termination.
func Rat_Float_Precision(
	value *Rat, workspace *Rat_Float_Precision_Workspace,
) (places Decimal_Place_Count, exact Boolean) {
	defer func() {
		Decimal_Place_Count_Invariants(places, "rat_float_precision.places")
		Boolean_Invariants(exact, "rat_float_precision.exact")
	}()
	Rat_Invariants(value, "rat_float_precision.value")
	Rat_Float_Precision_Workspace_Invariants(workspace, "rat_float_precision.workspace")
	denominator := &workspace.Integers[RAT_FLOAT_PRECISION_DENOMINATOR_INDEX]
	Rat_Denominator_Into(denominator, value)
	two_places := Int_Trailing_Zero_Bit_Count(denominator)
	invariant.Always(
		int(two_places) <= RAT_FLOAT_PRECISION_COUNT_MAXIMUM,
		"Rational denominator bound contains every binary factor count.",
	)
	Int_Shift_Right(denominator, denominator, Shift_Count(two_places))
	places = Decimal_Place_Count(two_places)
	five_places := Decimal_Place_Count(RAT_FLOAT_PRECISION_COUNT_MINIMUM)
	for denominator.Count != Word_Count(WORD_COUNT_INCREMENT) ||
		denominator.Words[WORD_COUNT_MINIMUM] != Word(bits.CARRY_MAXIMUM) {
		remainder := rat_float_divide_by_decimal_factor(&workspace.Integers)
		if remainder != Decimal_Factor_Remainder(bits.WORD_64_MINIMUM) {
			return places, Boolean(false)
		}
		five_places++
		if five_places > places {
			places = five_places
		}
	}
	return places, Boolean(true)
}

// Rat_Float_64_Bits returns nearest numeric binary64 encoding using round-to-even.
func Rat_Float_64_Bits(
	value *Rat, workspace *Rat_Float_64_Workspace,
) (encoding Float_64_Value_Bits, exact Boolean) {
	defer func() {
		Float_64_Value_Bits_Invariants(encoding, "rat_float_64_bits.encoding")
		Boolean_Invariants(exact, "rat_float_64_bits.exact")
	}()
	Rat_Invariants(value, "rat_float_64_bits.value")
	Rat_Float_64_Workspace_Invariants(workspace, "rat_float_64_bits.workspace")
	numerator_source := &value.Integers[RAT_NUMERATOR_INDEX]
	if numerator_source.Count == WORD_COUNT_MINIMUM {
		return Float_64_Value_Bits(FLOAT_64_VALUE_BITS_MINIMUM), Boolean(true)
	}
	denominator_source := &value.Integers[RAT_DENOMINATOR_INDEX]
	exponent := int(Int_Bit_Count(numerator_source)) - SIGN_BYTE_COUNT_MAXIMUM
	if denominator_source.Count != WORD_COUNT_MINIMUM {
		exponent += SIGN_BYTE_COUNT_MAXIMUM - int(Int_Bit_Count(denominator_source))
	}
	rat_float_64_divide(value, workspace)
	quotient := &workspace.Integers[RAT_FLOAT_64_QUOTIENT_INDEX]
	remainder := &workspace.Integers[RAT_FLOAT_64_REMAINDER_INDEX]
	quotient_word, conversion_status := Int_Uint_64(quotient)
	quotient_fits := conversion_status == Conversion_Status(STATUS_OK)
	invariant.Always(quotient_fits, "Binary64 rounding quotient fits one word.")
	mantissa := uint64(quotient_word)
	have_remainder := remainder.Count != WORD_COUNT_MINIMUM
	if mantissa>>FLOAT_64_ROUNDING_MANTISSA_BIT_COUNT == uint64(bits.CARRY_MAXIMUM) {
		if mantissa&uint64(bits.CARRY_MAXIMUM) != bits.WORD_64_MINIMUM {
			have_remainder = true
		}
		mantissa >>= 1
		exponent++
	}
	invariant.Always(
		mantissa>>FLOAT_64_VALUE_MANTISSA_BIT_COUNT == uint64(bits.CARRY_MAXIMUM),
		"Scaled rational quotient retains one normal leading bit.",
	)
	subnormal := false
	if exponent >= FLOAT_64_SUBNORMAL_EXPONENT_MINIMUM {
		if exponent <= FLOAT_64_SUBNORMAL_EXPONENT {
			subnormal = true
			shift := uint(FLOAT_64_SUBNORMAL_EXPONENT - exponent + 1)
			lost_mask := uint64(1)<<shift - 1
			have_remainder = have_remainder || mantissa&lost_mask != 0
			mantissa >>= shift
			exponent = 2 - FLOAT_64_EXPONENT_BIAS
		}
	}
	exact = Boolean(!have_remainder)
	if mantissa&uint64(bits.CARRY_MAXIMUM) != bits.WORD_64_MINIMUM {
		exact = Boolean(false)
		round_up := have_remainder
		if !round_up {
			if mantissa&uint64(BASE_BINARY) != 0 {
				round_up = true
			}
		}
		if round_up {
			mantissa++
			if mantissa >= FLOAT_64_ROUNDING_MANTISSA_LIMIT {
				mantissa >>= 1
				exponent++
			}
		}
	}
	mantissa >>= 1
	return rat_float_64_encode(
		Rat_Float_64_Mantissa(mantissa), Rat_Float_64_Exponent(exponent),
		Boolean(subnormal), numerator_source.Negative, exact,
	)
}

func rat_float_64_encode(
	mantissa Rat_Float_64_Mantissa,
	exponent Rat_Float_64_Exponent,
	subnormal Boolean,
	negative Polarity,
	exact Boolean,
) (encoding Float_64_Value_Bits, result_exact Boolean) {
	defer func() {
		Float_64_Value_Bits_Invariants(encoding, "rat_float_64_encode.encoding")
		Boolean_Invariants(result_exact, "rat_float_64_encode.result_exact")
	}()
	Rat_Float_64_Mantissa_Invariants(mantissa, "rat_float_64_encode.mantissa")
	Rat_Float_64_Exponent_Invariants(exponent, "rat_float_64_encode.exponent")
	Boolean_Invariants(subnormal, "rat_float_64_encode.subnormal")
	Polarity_Invariants(negative, "rat_float_64_encode.negative")
	Boolean_Invariants(exact, "rat_float_64_encode.exact")
	result_exact = exact
	encoded := uint64(FLOAT_64_VALUE_BITS_MINIMUM)
	if int(exponent) >= FLOAT_64_SUBNORMAL_EXPONENT_MINIMUM {
		encoded_exponent := int(exponent) + FLOAT_64_EXPONENT_ENCODING_OFFSET
		if bool(subnormal) {
			encoded_exponent = 0
			if uint64(mantissa) >= FLOAT_64_HIDDEN_MANTISSA_BIT {
				encoded_exponent++
			}
		}
		if encoded_exponent >= FLOAT_64_EXPONENT_MASK {
			encoded = FLOAT_64_POSITIVE_INFINITY_BITS
			result_exact = Boolean(false)
		} else {
			encoded = uint64(encoded_exponent)<<FLOAT_64_EXPONENT_SHIFT |
				uint64(mantissa)&FLOAT_64_MANTISSA_MASK
		}
	}
	if negative == POLARITY_NEGATIVE {
		encoded |= FLOAT_64_SIGN_MASK
	}
	return Float_64_Value_Bits(encoded), result_exact
}

func rat_float_64_divide(value *Rat, workspace *Rat_Float_64_Workspace) {
	Rat_Invariants(value, "rat_float_64_divide.value")
	Rat_Float_64_Workspace_Invariants(workspace, "rat_float_64_divide.workspace")
	numerator := &workspace.Integers[RAT_FLOAT_64_NUMERATOR_INDEX]
	denominator := &workspace.Integers[RAT_FLOAT_64_DENOMINATOR_INDEX]
	quotient := &workspace.Integers[RAT_FLOAT_64_QUOTIENT_INDEX]
	remainder := &workspace.Integers[RAT_FLOAT_64_REMAINDER_INDEX]
	Int_Absolute(numerator, &value.Integers[RAT_NUMERATOR_INDEX])
	Rat_Denominator_Into(denominator, value)
	exponent := int(Int_Bit_Count(numerator)) - int(Int_Bit_Count(denominator))
	shift := FLOAT_64_ROUNDING_MANTISSA_BIT_COUNT - exponent
	shift_status := Arithmetic_Status(STATUS_OK)
	if shift > 0 {
		shift_status = Int_Shift_Left(numerator, numerator, Shift_Count(shift))
	} else if shift < 0 {
		shift_status = Int_Shift_Left(denominator, denominator, Shift_Count(-shift))
	}
	invariant.Always(
		shift_status == Arithmetic_Status(STATUS_OK),
		"Rational component bound leaves full Int room for binary64 scaling.",
	)
	division := (*Int_Division_Workspace)(&workspace.Division)
	division_status := Int_Quotient_Remainder(
		quotient, remainder, numerator, denominator, division,
	)
	invariant.Always(
		division_status == Division_Status(STATUS_OK),
		"Normalized rational denominator remains nonzero during binary64 conversion.",
	)
}

func rat_float_divide_by_decimal_factor(
	integers *Rat_Float_Precision_Integers,
) (remainder Decimal_Factor_Remainder) {
	defer func() {
		Decimal_Factor_Remainder_Invariants(
			remainder, "rat_float_divide_by_decimal_factor.remainder",
		)
	}()
	Rat_Float_Precision_Integers_Invariants(
		*integers, "rat_float_divide_by_decimal_factor.integers",
	)
	value := &integers[RAT_FLOAT_PRECISION_DENOMINATOR_INDEX]
	invariant.Always(
		value.Negative == POLARITY_NONNEGATIVE,
		"Decimal denominator factorization receives nonnegative magnitude.",
	)
	word_remainder := uint64(bits.WORD_64_MINIMUM)
	for index := int(value.Count) - 1; index >= WORD_COUNT_MINIMUM; index-- {
		word := uint64(value.Words[index])
		high_limb := word >> TEXT_DIVISION_LIMB_BIT_COUNT
		high_dividend := word_remainder<<TEXT_DIVISION_LIMB_BIT_COUNT | high_limb
		quotient_high := high_dividend / RAT_DECIMAL_PRIME_FACTOR
		word_remainder = high_dividend % RAT_DECIMAL_PRIME_FACTOR
		low_limb := word & TEXT_DIVISION_LIMB_MASK
		low_dividend := word_remainder<<TEXT_DIVISION_LIMB_BIT_COUNT | low_limb
		quotient_low := low_dividend / RAT_DECIMAL_PRIME_FACTOR
		word_remainder = low_dividend % RAT_DECIMAL_PRIME_FACTOR
		value.Words[index] = Word(
			quotient_high<<TEXT_DIVISION_LIMB_BIT_COUNT | quotient_low,
		)
	}
	for value.Count > WORD_COUNT_MINIMUM {
		if value.Words[int(value.Count)-WORD_COUNT_INCREMENT] != 0 {
			break
		}
		value.Count--
	}
	return Decimal_Factor_Remainder(word_remainder)
}

func rat_float_text_components(
	value *Rat, precision Rat_Precision, workspace *Rat_Float_Text_Workspace,
) {
	Rat_Invariants(value, "rat_float_text_components.value")
	Rat_Precision_Invariants(precision, "rat_float_text_components.precision")
	Rat_Float_Text_Workspace_Invariants(workspace, "rat_float_text_components.workspace")
	numerator := &workspace.Integers[RAT_FLOAT_NUMERATOR_INDEX]
	denominator := &workspace.Integers[RAT_FLOAT_DENOMINATOR_INDEX]
	integer := &workspace.Integers[RAT_FLOAT_INTEGER_INDEX]
	remainder := &workspace.Integers[RAT_FLOAT_REMAINDER_INDEX]
	scale := &workspace.Integers[RAT_FLOAT_SCALE_INDEX]
	scaled_remainder := &workspace.Integers[RAT_FLOAT_SCALED_REMAINDER_INDEX]
	fraction := &workspace.Integers[RAT_FLOAT_FRACTION_INDEX]
	rounding_remainder := &workspace.Integers[RAT_FLOAT_ROUNDING_REMAINDER_INDEX]
	unit := &workspace.Integers[RAT_FLOAT_UNIT_INDEX]
	base := &workspace.Integers[RAT_FLOAT_BASE_INDEX]
	exponent := &workspace.Integers[RAT_FLOAT_PRECISION_INDEX]
	Int_Absolute(numerator, &value.Integers[RAT_NUMERATOR_INDEX])
	Rat_Denominator_Into(denominator, value)
	Int_Set_Uint_64(unit, Word_64(bits.CARRY_MAXIMUM))
	Int_Set_Uint_64(base, Word_64(BASE_DECIMAL))
	Int_Set_Uint_64(exponent, Word_64(precision))
	exponent_workspace := (*Int_Exponent_Workspace)(&workspace.Exponent)
	exponent_status := Int_Exponent(scale, base, exponent, exponent_workspace)
	invariant.Always(
		exponent_status == Arithmetic_Status(STATUS_OK),
		"Validated rational precision keeps decimal scale inside full Int storage.",
	)
	division := (*Int_Division_Workspace)(&workspace.Division)
	division_status := Int_Quotient_Remainder(
		integer, remainder, numerator, denominator, division,
	)
	invariant.Always(
		division_status == Division_Status(STATUS_OK),
		"Normalized rational denominator is never zero.",
	)
	multiplication := (*Int_Multiplication_Workspace)(&workspace.Multiplication)
	multiply_status := Int_Multiply(scaled_remainder, remainder, scale, multiplication)
	invariant.Always(
		multiply_status == Arithmetic_Status(STATUS_OK),
		"Component and precision bounds keep scaled remainder inside full Int storage.",
	)
	division_status = Int_Quotient_Remainder(
		fraction, rounding_remainder, scaled_remainder, denominator, division,
	)
	invariant.Always(
		division_status == Division_Status(STATUS_OK),
		"Normalized rational denominator remains nonzero during decimal division.",
	)
	rat_float_text_round(&workspace.Integers)
}

func rat_float_text_round(integers *Rat_Float_Text_Integers) {
	Rat_Float_Text_Integers_Invariants(*integers, "rat_float_text_round.integers")
	denominator := &integers[RAT_FLOAT_DENOMINATOR_INDEX]
	integer := &integers[RAT_FLOAT_INTEGER_INDEX]
	scale := &integers[RAT_FLOAT_SCALE_INDEX]
	fraction := &integers[RAT_FLOAT_FRACTION_INDEX]
	rounding_remainder := &integers[RAT_FLOAT_ROUNDING_REMAINDER_INDEX]
	rounding_double := &integers[RAT_FLOAT_ROUNDING_DOUBLE_INDEX]
	unit := &integers[RAT_FLOAT_UNIT_INDEX]
	status := Int_Add(rounding_double, rounding_remainder, rounding_remainder)
	invariant.Always(
		status == Arithmetic_Status(STATUS_OK),
		"Twice a component remainder fits full Int storage.",
	)
	if Int_Compare(rounding_double, denominator) == ORDER_BEFORE {
		return
	}
	status = Int_Add(fraction, fraction, unit)
	invariant.Always(
		status == Arithmetic_Status(STATUS_OK),
		"Rounded fraction fits its decimal scale.",
	)
	order := Int_Compare(fraction, scale)
	invariant.Always(order != ORDER_AFTER, "Rounded fraction never exceeds decimal scale.")
	if order == ORDER_SAME {
		int_zero(fraction, fraction.Count)
		status = Int_Add(integer, integer, unit)
		invariant.Always(
			status == Arithmetic_Status(STATUS_OK),
			"Rational component bound leaves room for rounding carry.",
		)
	}
}

func rat_float_text_write(
	destination Rat_Float_Text_Destination,
	value *Rat,
	precision Rat_Precision,
	integer_digits Rat_Component_Digit_Count,
	fractional_digits Rat_Component_Digit_Count,
	text *Rat_Float_Text_Integer_Workspaces,
) {
	Rat_Float_Text_Destination_Invariants(destination, "rat_float_text_write.destination")
	Rat_Invariants(value, "rat_float_text_write.value")
	Rat_Precision_Invariants(precision, "rat_float_text_write.precision")
	Rat_Component_Digit_Count_Invariants(
		integer_digits, "rat_float_text_write.integer_digits",
	)
	Rat_Component_Digit_Count_Invariants(
		fractional_digits, "rat_float_text_write.fractional_digits",
	)
	Rat_Float_Text_Integer_Workspaces_Invariants(
		*text, "rat_float_text_write.text",
	)
	destination_index := bytes.SLICE_SIZE_MINIMUM
	if value.Integers[RAT_NUMERATOR_INDEX].Negative == POLARITY_NEGATIVE {
		destination[destination_index] = '-'
		destination_index++
	}
	integer_text := &text[RAT_FLOAT_TEXT_INTEGER_PART_INDEX]
	for index := int(integer_digits) - 1; index >= 0; index-- {
		destination[destination_index] = integer_text.Digits[index]
		destination_index++
	}
	if precision == RAT_PRECISION_MINIMUM {
		return
	}
	destination[destination_index] = '.'
	destination_index++
	for index := int(fractional_digits); index < int(precision); index++ {
		destination[destination_index] = '0'
		destination_index++
	}
	fractional_text := &text[RAT_FLOAT_TEXT_FRACTIONAL_PART_INDEX]
	for index := int(fractional_digits) - 1; index >= 0; index-- {
		destination[destination_index] = fractional_text.Digits[index]
		destination_index++
	}
}

func rat_text_into(
	destination Text, value *Rat, workspace *Rat_Text_Workspace, form Rat_Text_Form,
) (count Rat_Text_Count, status Destination_Status) {
	defer func() {
		Rat_Text_Count_Invariants(count, "rat_text_into_internal.count")
		Destination_Status_Invariants(status, "rat_text_into_internal.status")
	}()
	Text_Invariants(destination, "rat_text_into_internal.destination")
	Rat_Invariants(value, "rat_text_into_internal.value")
	Rat_Text_Workspace_Invariants(workspace, "rat_text_into_internal.workspace")
	Rat_Text_Form_Invariants(form, "rat_text_into_internal.form")
	numerator := &value.Integers[RAT_NUMERATOR_INDEX]
	denominator := &workspace.Integers[RAT_TEXT_INTEGER_INDEX]
	Rat_Denominator_Into(denominator, value)
	numerator_digits := int_text_digits(
		numerator, BASE_DECIMAL, &workspace.Text[RAT_TEXT_NUMERATOR_INDEX],
	)
	include_denominator := true
	if form == RAT_TEXT_FORM_RATIONAL {
		include_denominator = !bool(Rat_Is_Integer(value))
	}
	denominator_digits := Text_Digit_Count(0)
	if include_denominator {
		denominator_digits = int_text_digits(
			denominator, BASE_DECIMAL, &workspace.Text[RAT_TEXT_DENOMINATOR_INDEX],
		)
	}
	required := int(numerator_digits)
	if numerator.Negative == POLARITY_NEGATIVE {
		required += SIGN_BYTE_COUNT_MAXIMUM
	}
	if include_denominator {
		required += RATIONAL_SEPARATOR_BYTE_COUNT + int(denominator_digits)
	}
	invariant.Always(
		required <= RAT_TEXT_SIZE_MAXIMUM,
		"Rational decimal text stays below binary-derived component bound.",
	)
	if len(destination) < required {
		return Rat_Text_Count(required), STATUS_DESTINATION_TOO_SMALL
	}
	destination_index := bytes.SLICE_SIZE_MINIMUM
	if numerator.Negative == POLARITY_NEGATIVE {
		destination[destination_index] = '-'
		destination_index++
	}
	for index := int(numerator_digits) - 1; index >= 0; index-- {
		destination[destination_index] =
			workspace.Text[RAT_TEXT_NUMERATOR_INDEX].Digits[index]
		destination_index++
	}
	if include_denominator {
		destination[destination_index] = '/'
		destination_index++
		for index := int(denominator_digits) - 1; index >= 0; index-- {
			destination[destination_index] =
				workspace.Text[RAT_TEXT_DENOMINATOR_INDEX].Digits[index]
			destination_index++
		}
	}
	return Rat_Text_Count(required), STATUS_OK
}

// Rat_Compare compares exact cross-products inside caller workspace.
func Rat_Compare(left *Rat, right *Rat, workspace *Rat_Workspace) (order Order) {
	defer func() { Order_Invariants(order, "rat_compare.order") }()
	Rat_Invariants(left, "rat_compare.left")
	Rat_Invariants(right, "rat_compare.right")
	Rat_Workspace_Invariants(workspace, "rat_compare.workspace")
	rat_load_operands(workspace, left, right)
	left_scaled := &workspace.Integers[RAT_RESULT_NUMERATOR_INDEX]
	right_scaled := &workspace.Integers[RAT_RESULT_DENOMINATOR_INDEX]
	multiplication := (*Int_Multiplication_Workspace)(&workspace.Multiplication)
	status := Int_Multiply(
		left_scaled, &workspace.Integers[RAT_LEFT_NUMERATOR_INDEX],
		&workspace.Integers[RAT_RIGHT_DENOMINATOR_INDEX], multiplication,
	)
	invariant.Always(
		status == Arithmetic_Status(STATUS_OK),
		"A rational comparison left cross-product fits reserved integer storage.",
	)
	status = Int_Multiply(
		right_scaled, &workspace.Integers[RAT_RIGHT_NUMERATOR_INDEX],
		&workspace.Integers[RAT_LEFT_DENOMINATOR_INDEX], multiplication,
	)
	invariant.Always(
		status == Arithmetic_Status(STATUS_OK),
		"A rational comparison right cross-product fits reserved integer storage.",
	)
	return Int_Compare(left_scaled, right_scaled)
}

// Rat_Add computes one reduced exact sum transactionally.
func Rat_Add(
	destination *Rat, left *Rat, right *Rat, workspace *Rat_Workspace,
) (status Rat_Arithmetic_Status) {
	defer func() { Rat_Arithmetic_Status_Invariants(status, "rat_add.status") }()
	Rat_Invariants(destination, "rat_add.destination")
	Rat_Invariants(left, "rat_add.left")
	Rat_Invariants(right, "rat_add.right")
	Rat_Workspace_Invariants(workspace, "rat_add.workspace")
	if rat_add_single_words(destination, left, right) {
		return STATUS_OK
	}
	return Rat_Arithmetic_Status(
		rat_binary(destination, left, right, workspace, RAT_OPERATION_ADD),
	)
}

// Rat_Subtract computes one reduced exact difference transactionally.
func Rat_Subtract(
	destination *Rat, left *Rat, right *Rat, workspace *Rat_Workspace,
) (status Rat_Arithmetic_Status) {
	defer func() { Rat_Arithmetic_Status_Invariants(status, "rat_subtract.status") }()
	Rat_Invariants(destination, "rat_subtract.destination")
	Rat_Invariants(left, "rat_subtract.left")
	Rat_Invariants(right, "rat_subtract.right")
	Rat_Workspace_Invariants(workspace, "rat_subtract.workspace")
	return Rat_Arithmetic_Status(
		rat_binary(destination, left, right, workspace, RAT_OPERATION_SUBTRACT),
	)
}

// Rat_Multiply computes one reduced exact product transactionally.
func Rat_Multiply(
	destination *Rat, left *Rat, right *Rat, workspace *Rat_Workspace,
) (status Rat_Arithmetic_Status) {
	defer func() { Rat_Arithmetic_Status_Invariants(status, "rat_multiply.status") }()
	Rat_Invariants(destination, "rat_multiply.destination")
	Rat_Invariants(left, "rat_multiply.left")
	Rat_Invariants(right, "rat_multiply.right")
	Rat_Workspace_Invariants(workspace, "rat_multiply.workspace")
	return Rat_Arithmetic_Status(
		rat_binary(destination, left, right, workspace, RAT_OPERATION_MULTIPLY),
	)
}

// Rat_Quotient computes one reduced exact quotient and rejects zero right numerator.
func Rat_Quotient(
	destination *Rat, left *Rat, right *Rat, workspace *Rat_Workspace,
) (status Rat_Division_Status) {
	defer func() { Rat_Division_Status_Invariants(status, "rat_quotient.status") }()
	Rat_Invariants(destination, "rat_quotient.destination")
	Rat_Invariants(left, "rat_quotient.left")
	Rat_Invariants(right, "rat_quotient.right")
	Rat_Workspace_Invariants(workspace, "rat_quotient.workspace")
	return Rat_Division_Status(
		rat_binary(destination, left, right, workspace, RAT_OPERATION_QUOTIENT),
	)
}

func rat_binary_products(
	workspace *Rat_Workspace,
	multiplication *Int_Multiplication_Workspace,
	operation Rat_Operation,
) {
	Rat_Workspace_Invariants(workspace, "rat_binary_products.workspace")
	Int_Multiplication_Workspace_Invariants(
		multiplication, "rat_binary_products.multiplication",
	)
	Rat_Operation_Invariants(operation, "rat_binary_products.operation")
	if operation == RAT_OPERATION_ADD {
		rat_add_products(workspace, multiplication)
		return
	}
	if operation == RAT_OPERATION_SUBTRACT {
		rat_subtract_products(workspace, multiplication)
		return
	}
	if operation == RAT_OPERATION_MULTIPLY {
		rat_multiply_products(workspace, multiplication)
		return
	}
	rat_quotient_products(workspace, multiplication)
}

func rat_add_products(workspace *Rat_Workspace, multiplication *Int_Multiplication_Workspace) {
	Rat_Workspace_Invariants(workspace, "rat_add_products.workspace")
	Int_Multiplication_Workspace_Invariants(multiplication, "rat_add_products.multiplication")
	integers := &workspace.Integers
	status := Int_Multiply(
		&integers[RAT_RESULT_NUMERATOR_INDEX], &integers[RAT_LEFT_NUMERATOR_INDEX],
		&integers[RAT_RIGHT_DENOMINATOR_INDEX], multiplication,
	)
	invariant.Always(
		status == Arithmetic_Status(STATUS_OK),
		"A rational sum left cross-product fits reserved integer storage.",
	)
	status = Int_Multiply(
		&integers[RAT_COMMON_DIVISOR_INDEX], &integers[RAT_RIGHT_NUMERATOR_INDEX],
		&integers[RAT_LEFT_DENOMINATOR_INDEX], multiplication,
	)
	invariant.Always(
		status == Arithmetic_Status(STATUS_OK),
		"A rational sum right cross-product fits reserved integer storage.",
	)
	status = Int_Add(
		&integers[RAT_RESULT_NUMERATOR_INDEX], &integers[RAT_RESULT_NUMERATOR_INDEX],
		&integers[RAT_COMMON_DIVISOR_INDEX],
	)
	invariant.Always(
		status == Arithmetic_Status(STATUS_OK),
		"Reserved rational component width holds one cross-product sum.",
	)
	status = Int_Multiply(
		&integers[RAT_RESULT_DENOMINATOR_INDEX], &integers[RAT_LEFT_DENOMINATOR_INDEX],
		&integers[RAT_RIGHT_DENOMINATOR_INDEX], multiplication,
	)
	invariant.Always(
		status == Arithmetic_Status(STATUS_OK),
		"A rational sum denominator product fits reserved integer storage.",
	)
}

func rat_subtract_products(
	workspace *Rat_Workspace, multiplication *Int_Multiplication_Workspace,
) {
	Rat_Workspace_Invariants(workspace, "rat_subtract_products.workspace")
	Int_Multiplication_Workspace_Invariants(
		multiplication, "rat_subtract_products.multiplication",
	)
	integers := &workspace.Integers
	status := Int_Multiply(
		&integers[RAT_RESULT_NUMERATOR_INDEX], &integers[RAT_LEFT_NUMERATOR_INDEX],
		&integers[RAT_RIGHT_DENOMINATOR_INDEX], multiplication,
	)
	invariant.Always(
		status == Arithmetic_Status(STATUS_OK),
		"A rational difference left cross-product fits reserved integer storage.",
	)
	status = Int_Multiply(
		&integers[RAT_COMMON_DIVISOR_INDEX], &integers[RAT_RIGHT_NUMERATOR_INDEX],
		&integers[RAT_LEFT_DENOMINATOR_INDEX], multiplication,
	)
	invariant.Always(
		status == Arithmetic_Status(STATUS_OK),
		"A rational difference right cross-product fits reserved integer storage.",
	)
	status = Int_Subtract(
		&integers[RAT_RESULT_NUMERATOR_INDEX], &integers[RAT_RESULT_NUMERATOR_INDEX],
		&integers[RAT_COMMON_DIVISOR_INDEX],
	)
	invariant.Always(
		status == Arithmetic_Status(STATUS_OK),
		"Reserved rational component width holds one cross-product difference.",
	)
	status = Int_Multiply(
		&integers[RAT_RESULT_DENOMINATOR_INDEX], &integers[RAT_LEFT_DENOMINATOR_INDEX],
		&integers[RAT_RIGHT_DENOMINATOR_INDEX], multiplication,
	)
	invariant.Always(
		status == Arithmetic_Status(STATUS_OK),
		"A rational difference denominator product fits reserved integer storage.",
	)
}

func rat_multiply_products(
	workspace *Rat_Workspace, multiplication *Int_Multiplication_Workspace,
) {
	Rat_Workspace_Invariants(workspace, "rat_multiply_products.workspace")
	Int_Multiplication_Workspace_Invariants(
		multiplication, "rat_multiply_products.multiplication",
	)
	integers := &workspace.Integers
	status := Int_Multiply(
		&integers[RAT_RESULT_NUMERATOR_INDEX], &integers[RAT_LEFT_NUMERATOR_INDEX],
		&integers[RAT_RIGHT_NUMERATOR_INDEX], multiplication,
	)
	invariant.Always(
		status == Arithmetic_Status(STATUS_OK),
		"A rational numerator product fits reserved integer storage.",
	)
	status = Int_Multiply(
		&integers[RAT_RESULT_DENOMINATOR_INDEX], &integers[RAT_LEFT_DENOMINATOR_INDEX],
		&integers[RAT_RIGHT_DENOMINATOR_INDEX], multiplication,
	)
	invariant.Always(
		status == Arithmetic_Status(STATUS_OK),
		"A rational denominator product fits reserved integer storage.",
	)
}

func rat_quotient_products(
	workspace *Rat_Workspace, multiplication *Int_Multiplication_Workspace,
) {
	Rat_Workspace_Invariants(workspace, "rat_quotient_products.workspace")
	Int_Multiplication_Workspace_Invariants(
		multiplication, "rat_quotient_products.multiplication",
	)
	integers := &workspace.Integers
	status := Int_Multiply(
		&integers[RAT_RESULT_NUMERATOR_INDEX], &integers[RAT_LEFT_NUMERATOR_INDEX],
		&integers[RAT_RIGHT_DENOMINATOR_INDEX], multiplication,
	)
	invariant.Always(
		status == Arithmetic_Status(STATUS_OK),
		"A rational quotient numerator product fits reserved integer storage.",
	)
	status = Int_Multiply(
		&integers[RAT_RESULT_DENOMINATOR_INDEX], &integers[RAT_LEFT_DENOMINATOR_INDEX],
		&integers[RAT_RIGHT_NUMERATOR_INDEX], multiplication,
	)
	invariant.Always(
		status == Arithmetic_Status(STATUS_OK),
		"A rational quotient denominator product fits reserved integer storage.",
	)
}

func rat_load_operands(workspace *Rat_Workspace, left *Rat, right *Rat) {
	Rat_Workspace_Invariants(workspace, "rat_load_operands.workspace")
	Rat_Invariants(left, "rat_load_operands.left")
	Rat_Invariants(right, "rat_load_operands.right")
	Int_Set(
		&workspace.Integers[RAT_LEFT_NUMERATOR_INDEX],
		&left.Integers[RAT_NUMERATOR_INDEX],
	)
	Int_Set(
		&workspace.Integers[RAT_RIGHT_NUMERATOR_INDEX],
		&right.Integers[RAT_NUMERATOR_INDEX],
	)
	left_denominator := &workspace.Integers[RAT_LEFT_DENOMINATOR_INDEX]
	Int_Set(left_denominator, &left.Integers[RAT_DENOMINATOR_INDEX])
	if left_denominator.Count == WORD_COUNT_MINIMUM {
		Int_Set_Uint_64(left_denominator, Word_64(bits.CARRY_MAXIMUM))
	}
	right_denominator := &workspace.Integers[RAT_RIGHT_DENOMINATOR_INDEX]
	Int_Set(right_denominator, &right.Integers[RAT_DENOMINATOR_INDEX])
	if right_denominator.Count == WORD_COUNT_MINIMUM {
		Int_Set_Uint_64(right_denominator, Word_64(bits.CARRY_MAXIMUM))
	}
}

func rat_normalize(
	destination *Rat, workspace *Rat_Workspace,
) (status Rat_Division_Status) {
	defer func() { Rat_Division_Status_Invariants(status, "rat_normalize.status") }()
	Rat_Invariants(destination, "rat_normalize.destination")
	Rat_Workspace_Invariants(workspace, "rat_normalize.workspace")
	numerator := &workspace.Integers[RAT_RESULT_NUMERATOR_INDEX]
	denominator := &workspace.Integers[RAT_RESULT_DENOMINATOR_INDEX]
	common_divisor := &workspace.Integers[RAT_COMMON_DIVISOR_INDEX]
	if denominator.Count == WORD_COUNT_MINIMUM {
		return STATUS_DIVISOR_ZERO
	}
	if denominator.Negative == POLARITY_NEGATIVE {
		denominator.Negative = POLARITY_NONNEGATIVE
		if numerator.Count > WORD_COUNT_MINIMUM {
			numerator.Negative = POLARITY_NEGATIVE - numerator.Negative
		}
	}
	if denominator.Count == Word_Count(WORD_COUNT_INCREMENT) {
		if denominator.Words[WORD_COUNT_MINIMUM] == Word(bits.CARRY_MAXIMUM) {
			int_zero(denominator, denominator.Count)
			if numerator.Count > RAT_WORD_COUNT_MAXIMUM {
				return STATUS_VALUE_OVERFLOW
			}
			Int_Set(&destination.Integers[RAT_NUMERATOR_INDEX], numerator)
			Int_Set(&destination.Integers[RAT_DENOMINATOR_INDEX], denominator)
			return STATUS_OK
		}
	}
	greatest_common := (*Int_Greatest_Common_Divisor_Workspace)(
		&workspace.Greatest_Common,
	)
	Int_Greatest_Common_Divisor(
		common_divisor, numerator, denominator, greatest_common,
	)
	// Unit common divisor leaves both components normalized, so division cannot change state.
	if common_divisor.Count == Word_Count(WORD_COUNT_INCREMENT) {
		if common_divisor.Words[WORD_COUNT_MINIMUM] == Word(bits.CARRY_MAXIMUM) {
			if numerator.Count > RAT_WORD_COUNT_MAXIMUM {
				return STATUS_VALUE_OVERFLOW
			}
			if denominator.Count > RAT_WORD_COUNT_MAXIMUM {
				return STATUS_VALUE_OVERFLOW
			}
			Int_Set(&destination.Integers[RAT_NUMERATOR_INDEX], numerator)
			Int_Set(&destination.Integers[RAT_DENOMINATOR_INDEX], denominator)
			return STATUS_OK
		}
	}
	division := (*Int_Division_Workspace)(&workspace.Greatest_Common.Division)
	numerator_status := Int_Quotient(numerator, numerator, common_divisor, division)
	invariant.Always(
		numerator_status == Divisor_Status(STATUS_OK),
		"A fraction common divisor is nonzero before numerator reduction.",
	)
	denominator_status := Int_Quotient(
		denominator, denominator, common_divisor, division,
	)
	invariant.Always(
		denominator_status == Divisor_Status(STATUS_OK),
		"A fraction common divisor is nonzero before denominator reduction.",
	)
	if numerator.Count > RAT_WORD_COUNT_MAXIMUM {
		return STATUS_VALUE_OVERFLOW
	}
	if denominator.Count > RAT_WORD_COUNT_MAXIMUM {
		return STATUS_VALUE_OVERFLOW
	}
	Int_Set(&destination.Integers[RAT_NUMERATOR_INDEX], numerator)
	Int_Set(&destination.Integers[RAT_DENOMINATOR_INDEX], denominator)
	return STATUS_OK
}

// Int_Shift_Left writes high words first so exact receiver aliasing cannot destroy unread input.
func Int_Shift_Left(
	destination *Int, source *Int, count Shift_Count,
) (status Arithmetic_Status) {
	defer func() { Arithmetic_Status_Invariants(status, "int_shift_left.status") }()
	Int_Invariants(destination, "int_shift_left.destination")
	Int_Invariants(source, "int_shift_left.source")
	Shift_Count_Invariants(count, "int_shift_left.count")
	if int_shift_left_small(destination, source, count) {
		return STATUS_OK
	}
	source_count := source.Count
	negative := source.Negative
	if source_count == WORD_COUNT_MINIMUM {
		int_zero(destination, destination.Count)
		return STATUS_OK
	}
	if count == 0 {
		*destination = *source
		return STATUS_OK
	}
	result_bit_count := int(Int_Bit_Count(source))
	result_bit_count += int(count)
	if result_bit_count > BIT_COUNT_MAXIMUM {
		return STATUS_VALUE_OVERFLOW
	}
	word_shift := int(count) / WORD_BIT_COUNT
	bit_shift := uint(count) % WORD_BIT_COUNT
	result_count := Word_Count(
		(result_bit_count + WORD_BIT_COUNT - 1) / WORD_BIT_COUNT,
	)
	previous_count := destination.Count
	destination_index := int(result_count) - 1
	for ; destination_index >= WORD_COUNT_MINIMUM; destination_index-- {
		source_index := destination_index - word_shift
		word := Word(0)
		if source_index >= WORD_COUNT_MINIMUM {
			if source_index < int(source_count) {
				word = source.Words[source_index] << bit_shift
			}
			if bit_shift != 0 {
				if source_index > WORD_COUNT_MINIMUM {
					if source_index <= int(source_count) {
						high_word := source.Words[source_index-1]
						word |= high_word >> (WORD_BIT_COUNT - bit_shift)
					}
				}
			}
		}
		destination.Words[destination_index] = word
	}
	destination.Count = result_count
	destination.Negative = negative
	int_clear(destination, result_count, previous_count)
	return STATUS_OK
}

// Int_Shift_Right rounds negative magnitudes upward so sign-magnitude storage matches arithmetic
// two's-complement shift semantics.
func Int_Shift_Right(destination *Int, source *Int, count Shift_Count) {
	Int_Invariants(destination, "int_shift_right.destination")
	Int_Invariants(source, "int_shift_right.source")
	Shift_Count_Invariants(count, "int_shift_right.count")
	if int_shift_right_small(destination, source, count) {
		return
	}
	source_count, negative, previous_count := source.Count, source.Negative, destination.Count
	if int(count) >= int(Int_Bit_Count(source)) {
		if negative == POLARITY_NEGATIVE {
			int_set_word(destination, 1, POLARITY_NEGATIVE)
			return
		}
		int_zero(destination, previous_count)
		return
	}
	word_shift, bit_shift := int(count)/WORD_BIT_COUNT, uint(count)%WORD_BIT_COUNT
	discarded := false
	for index := WORD_COUNT_MINIMUM; index < word_shift; index++ {
		if source.Words[index] != 0 {
			discarded = true
			break
		}
	}
	if bit_shift != 0 {
		mask := Word((bits.CARRY_MAXIMUM << bit_shift) - bits.CARRY_MAXIMUM)
		if source.Words[word_shift]&mask != 0 {
			discarded = true
		}
	}
	result_count := source_count - Word_Count(word_shift)
	destination_limit := int(result_count)
	destination_index := WORD_COUNT_MINIMUM
	for ; destination_index < destination_limit; destination_index++ {
		source_index := destination_index
		source_index += word_shift
		word := source.Words[source_index] >> bit_shift
		if bit_shift != 0 {
			next_index := source_index + 1
			if next_index < int(source_count) {
				word |= source.Words[next_index] << (WORD_BIT_COUNT - bit_shift)
			}
		}
		destination.Words[destination_index] = word
	}
	if negative == POLARITY_NEGATIVE {
		if discarded {
			carry := bits.Carry_In(bits.CARRY_MAXIMUM)
			for index := WORD_COUNT_MINIMUM; index < int(result_count); index++ {
				word, next := bits.Add_64(
					bits.Word_64(destination.Words[index]), 0, carry,
				)
				destination.Words[index] = Word(word)
				carry = bits.Carry_In(next)
				if carry == 0 {
					break
				}
			}
		}
	}
	for result_count > WORD_COUNT_MINIMUM {
		if destination.Words[int(result_count)-1] != 0 {
			break
		}
		result_count--
	}
	destination.Count = result_count
	destination.Negative = negative
	int_clear(destination, result_count, previous_count)
}

// Int_And intersects infinite two's-complement bit strings.
func Int_And(destination *Int, left *Int, right *Int, workspace *Int_Bitwise_Workspace) {
	Int_Invariants(destination, "int_and.destination")
	Int_Invariants(left, "int_and.left")
	Int_Invariants(right, "int_and.right")
	Int_Bitwise_Workspace_Invariants(workspace, "int_and.workspace")
	if left.Count == Word_Count(WORD_COUNT_MINIMUM) {
		if right.Count == Word_Count(WORD_COUNT_MINIMUM) {
			status := int_bitwise_binary(
				destination, left, right, workspace, BITWISE_OPERATION_AND,
			)
			invariant.Always(
				status == Arithmetic_Status(STATUS_OK),
				"Zero AND zero retains the signed workspace boundary.",
			)
			return
		}
	}
	if left.Negative == POLARITY_NONNEGATIVE {
		if right.Negative == POLARITY_NONNEGATIVE {
			previous_count := destination.Count
			count := min(left.Count, right.Count)
			for index := Word_Count(WORD_COUNT_MINIMUM); index < count; index++ {
				destination.Words[index] = left.Words[index] & right.Words[index]
			}
			for count > Word_Count(WORD_COUNT_MINIMUM) {
				if destination.Words[count-Word_Count(WORD_COUNT_INCREMENT)] != 0 {
					break
				}
				count--
			}
			destination.Count = count
			destination.Negative = POLARITY_NONNEGATIVE
			if previous_count > count {
				int_clear(destination, count, previous_count)
			}
			return
		}
	}
	status := int_bitwise_binary(
		destination, left, right, workspace, BITWISE_OPERATION_AND,
	)
	invariant.Always(
		status == Arithmetic_Status(STATUS_OK),
		"Signed AND cannot construct the absent negative boundary.",
	)
}

// Int_And_Not clears right bits and reports the one absent negative boundary.
func Int_And_Not(
	destination *Int, left *Int, right *Int, workspace *Int_Bitwise_Workspace,
) (status Arithmetic_Status) {
	defer func() { Arithmetic_Status_Invariants(status, "int_and_not.status") }()
	Int_Invariants(destination, "int_and_not.destination")
	Int_Invariants(left, "int_and_not.left")
	Int_Invariants(right, "int_and_not.right")
	Int_Bitwise_Workspace_Invariants(workspace, "int_and_not.workspace")
	if left.Negative == POLARITY_NONNEGATIVE {
		if right.Negative == POLARITY_NONNEGATIVE {
			previous_count := destination.Count
			count := left.Count
			for index := Word_Count(WORD_COUNT_MINIMUM); index < count; index++ {
				right_word := Word(0)
				if index < right.Count {
					right_word = right.Words[index]
				}
				destination.Words[index] = left.Words[index] &^ right_word
			}
			for count > Word_Count(WORD_COUNT_MINIMUM) {
				if destination.Words[count-Word_Count(WORD_COUNT_INCREMENT)] != 0 {
					break
				}
				count--
			}
			destination.Count = count
			destination.Negative = POLARITY_NONNEGATIVE
			if previous_count > count {
				int_clear(destination, count, previous_count)
			}
			return STATUS_OK
		}
	}
	return int_bitwise_binary(
		destination, left, right, workspace, BITWISE_OPERATION_AND_NOT,
	)
}

// Int_Or unites infinite two's-complement bit strings.
func Int_Or(destination *Int, left *Int, right *Int, workspace *Int_Bitwise_Workspace) {
	Int_Invariants(destination, "int_or.destination")
	Int_Invariants(left, "int_or.left")
	Int_Invariants(right, "int_or.right")
	Int_Bitwise_Workspace_Invariants(workspace, "int_or.workspace")
	if left.Negative == POLARITY_NONNEGATIVE {
		if right.Negative == POLARITY_NONNEGATIVE {
			previous_count := destination.Count
			count := max(left.Count, right.Count)
			for index := Word_Count(WORD_COUNT_MINIMUM); index < count; index++ {
				left_word := Word(0)
				if index < left.Count {
					left_word = left.Words[index]
				}
				right_word := Word(0)
				if index < right.Count {
					right_word = right.Words[index]
				}
				destination.Words[index] = left_word | right_word
			}
			destination.Count = count
			destination.Negative = POLARITY_NONNEGATIVE
			if previous_count > count {
				int_clear(destination, count, previous_count)
			}
			return
		}
	}
	status := int_bitwise_binary(
		destination, left, right, workspace, BITWISE_OPERATION_OR,
	)
	invariant.Always(
		status == Arithmetic_Status(STATUS_OK),
		"Signed OR cannot construct the absent negative boundary.",
	)
}

// Int_Xor differs infinite signed bit strings and reports the absent negative boundary.
func Int_Xor(
	destination *Int, left *Int, right *Int, workspace *Int_Bitwise_Workspace,
) (status Arithmetic_Status) {
	defer func() { Arithmetic_Status_Invariants(status, "int_xor.status") }()
	Int_Invariants(destination, "int_xor.destination")
	Int_Invariants(left, "int_xor.left")
	Int_Invariants(right, "int_xor.right")
	Int_Bitwise_Workspace_Invariants(workspace, "int_xor.workspace")
	if left.Negative == POLARITY_NONNEGATIVE {
		if right.Negative == POLARITY_NONNEGATIVE {
			previous_count := destination.Count
			count := max(left.Count, right.Count)
			for index := Word_Count(WORD_COUNT_MINIMUM); index < count; index++ {
				left_word := Word(0)
				if index < left.Count {
					left_word = left.Words[index]
				}
				right_word := Word(0)
				if index < right.Count {
					right_word = right.Words[index]
				}
				destination.Words[index] = left_word ^ right_word
			}
			for count > Word_Count(WORD_COUNT_MINIMUM) {
				if destination.Words[count-Word_Count(WORD_COUNT_INCREMENT)] != 0 {
					break
				}
				count--
			}
			destination.Count = count
			destination.Negative = POLARITY_NONNEGATIVE
			if previous_count > count {
				int_clear(destination, count, previous_count)
			}
			return STATUS_OK
		}
	}
	return int_bitwise_binary(
		destination, left, right, workspace, BITWISE_OPERATION_XOR,
	)
}

// Int_Not complements one infinite signed bit string and reports the absent negative boundary.
func Int_Not(
	destination *Int, source *Int, workspace *Int_Bitwise_Workspace,
) (status Arithmetic_Status) {
	defer func() { Arithmetic_Status_Invariants(status, "int_not.status") }()
	Int_Invariants(destination, "int_not.destination")
	Int_Invariants(source, "int_not.source")
	Int_Bitwise_Workspace_Invariants(workspace, "int_not.workspace")
	if source.Negative == POLARITY_NONNEGATIVE {
		if source.Count > Word_Count(WORD_COUNT_INCREMENT) {
			if source.Count < Word_Count(WORD_COUNT_MAXIMUM) {
				previous_count := destination.Count
				source_count := source.Count
				carry := Word(bits.CARRY_MAXIMUM)
				index := Word_Count(WORD_COUNT_MINIMUM)
				for ; index < source_count; index++ {
					word := source.Words[index] + carry
					destination.Words[index] = word
					carry = Word(0)
					if word == 0 {
						carry = Word(bits.CARRY_MAXIMUM)
					}
				}
				result_count := source_count
				if carry != 0 {
					destination.Words[result_count] = carry
					result_count++
				}
				destination.Count = result_count
				destination.Negative = POLARITY_NEGATIVE
				if previous_count > result_count {
					int_clear(destination, result_count, previous_count)
				}
				return STATUS_OK
			}
		}
	}
	if int_not_word(destination, source) {
		return STATUS_OK
	}
	width := Bitwise_Word_Count(int(source.Count) + WORD_COUNT_INCREMENT)
	Bitwise_Word_Count_Invariants(width, "int_not.width")
	int_bitwise_encode_left(workspace, source, width)
	for index := WORD_COUNT_MINIMUM; index < int(width); index++ {
		workspace.Result[index] = ^workspace.Left[index]
	}
	return int_bitwise_decode(destination, workspace, width)
}

// Int_Set_Bit writes one infinite signed bit and rejects the absent negative boundary.
func Int_Set_Bit(
	destination *Int,
	source *Int,
	index Bit_Index,
	bit Bit_Value,
	workspace *Int_Bitwise_Workspace,
) (status Arithmetic_Status) {
	defer func() { Arithmetic_Status_Invariants(status, "int_set_bit.status") }()
	Int_Invariants(destination, "int_set_bit.destination")
	Int_Invariants(source, "int_set_bit.source")
	Bit_Index_Invariants(index, "int_set_bit.index")
	Bit_Value_Invariants(bit, "int_set_bit.bit")
	Int_Bitwise_Workspace_Invariants(workspace, "int_set_bit.workspace")
	word_index := int(index) / WORD_BIT_COUNT
	if source.Negative == POLARITY_NONNEGATIVE {
		previous_count := destination.Count
		source_count := source.Count
		result_count := Word_Count(word_index + WORD_COUNT_INCREMENT)
		if source_count > result_count {
			result_count = source_count
		}
		source_index := Word_Count(WORD_COUNT_MINIMUM)
		for ; source_index < source_count; source_index++ {
			destination.Words[source_index] = source.Words[source_index]
		}
		for clear_index := source_count; clear_index < result_count; clear_index++ {
			destination.Words[clear_index] = 0
		}
		mask := Word(bits.CARRY_MAXIMUM) << (uint(index) % WORD_BIT_COUNT)
		if bit == BIT_CLEAR {
			destination.Words[word_index] &^= mask
		} else {
			destination.Words[word_index] |= mask
		}
		for result_count > Word_Count(WORD_COUNT_MINIMUM) {
			if destination.Words[result_count-Word_Count(WORD_COUNT_INCREMENT)] != 0 {
				break
			}
			result_count--
		}
		destination.Count = result_count
		destination.Negative = POLARITY_NONNEGATIVE
		if previous_count > result_count {
			int_clear(destination, result_count, previous_count)
		}
		return STATUS_OK
	}
	width := Bitwise_Word_Count(int(source.Count) + WORD_COUNT_INCREMENT)
	index_width := Bitwise_Word_Count(
		word_index + WORD_COUNT_INCREMENT + WORD_COUNT_INCREMENT,
	)
	if index_width > width {
		width = index_width
	}
	int_bitwise_encode_left(workspace, source, width)
	mask := Word(bits.CARRY_MAXIMUM) << (uint(index) % WORD_BIT_COUNT)
	if bit == BIT_CLEAR {
		workspace.Left[word_index] &^= mask
	} else {
		workspace.Left[word_index] |= mask
	}
	for scratch_index := WORD_COUNT_MINIMUM; scratch_index < int(width); scratch_index++ {
		workspace.Result[scratch_index] = workspace.Left[scratch_index]
	}
	return int_bitwise_decode(destination, workspace, width)
}

func int_bitwise_binary(
	destination *Int,
	left *Int,
	right *Int,
	workspace *Int_Bitwise_Workspace,
	operation Bitwise_Operation,
) (status Arithmetic_Status) {
	defer func() { Arithmetic_Status_Invariants(status, "int_bitwise_binary.status") }()
	Int_Invariants(destination, "int_bitwise_binary.destination")
	Int_Invariants(left, "int_bitwise_binary.left")
	Int_Invariants(right, "int_bitwise_binary.right")
	Int_Bitwise_Workspace_Invariants(workspace, "int_bitwise_binary.workspace")
	Bitwise_Operation_Invariants(operation, "int_bitwise_binary.operation")
	width := Bitwise_Word_Count(int(left.Count) + WORD_COUNT_INCREMENT)
	right_width := Bitwise_Word_Count(int(right.Count) + WORD_COUNT_INCREMENT)
	if right_width > width {
		width = right_width
	}
	Bitwise_Word_Count_Invariants(width, "int_bitwise_binary.width")
	int_bitwise_encode_left(workspace, left, width)
	int_bitwise_encode_right(workspace, right, width)
	for index := WORD_COUNT_MINIMUM; index < int(width); index++ {
		switch operation {
		case BITWISE_OPERATION_AND:
			workspace.Result[index] = workspace.Left[index] & workspace.Right[index]
		case BITWISE_OPERATION_AND_NOT:
			workspace.Result[index] = workspace.Left[index] &^ workspace.Right[index]
		case BITWISE_OPERATION_OR:
			workspace.Result[index] = workspace.Left[index] | workspace.Right[index]
		case BITWISE_OPERATION_XOR:
			workspace.Result[index] = workspace.Left[index] ^ workspace.Right[index]
		}
	}
	return int_bitwise_decode(destination, workspace, width)
}

func int_bitwise_encode_left(
	workspace *Int_Bitwise_Workspace, value *Int, width Bitwise_Word_Count,
) {
	Int_Bitwise_Workspace_Invariants(workspace, "int_bitwise_encode_left.workspace")
	Int_Invariants(value, "int_bitwise_encode_left.value")
	Bitwise_Word_Count_Invariants(width, "int_bitwise_encode_left.width")
	for index := WORD_COUNT_MINIMUM; index < int(width); index++ {
		word := Word(0)
		if index < int(value.Count) {
			word = value.Words[index]
		}
		if value.Negative == POLARITY_NEGATIVE {
			word = ^word
		}
		workspace.Left[index] = word
	}
	if value.Negative == POLARITY_NEGATIVE {
		for index := WORD_COUNT_MINIMUM; index < int(width); index++ {
			workspace.Left[index]++
			if workspace.Left[index] != 0 {
				break
			}
		}
	}
}

func int_bitwise_encode_right(
	workspace *Int_Bitwise_Workspace, value *Int, width Bitwise_Word_Count,
) {
	Int_Bitwise_Workspace_Invariants(workspace, "int_bitwise_encode_right.workspace")
	Int_Invariants(value, "int_bitwise_encode_right.value")
	Bitwise_Word_Count_Invariants(width, "int_bitwise_encode_right.width")
	for index := WORD_COUNT_MINIMUM; index < int(width); index++ {
		word := Word(0)
		if index < int(value.Count) {
			word = value.Words[index]
		}
		if value.Negative == POLARITY_NEGATIVE {
			word = ^word
		}
		workspace.Right[index] = word
	}
	if value.Negative == POLARITY_NEGATIVE {
		for index := WORD_COUNT_MINIMUM; index < int(width); index++ {
			workspace.Right[index]++
			if workspace.Right[index] != 0 {
				break
			}
		}
	}
}

func int_bitwise_decode(
	destination *Int, workspace *Int_Bitwise_Workspace, width Bitwise_Word_Count,
) (status Arithmetic_Status) {
	defer func() { Arithmetic_Status_Invariants(status, "int_bitwise_decode.status") }()
	Int_Invariants(destination, "int_bitwise_decode.destination")
	Int_Bitwise_Workspace_Invariants(workspace, "int_bitwise_decode.workspace")
	Bitwise_Word_Count_Invariants(width, "int_bitwise_decode.width")
	high := workspace.Result[int(width)-WORD_COUNT_INCREMENT]
	negative := high>>WORD_BIT_INDEX_MAXIMUM == Word(bits.CARRY_MAXIMUM)
	if negative {
		for index := WORD_COUNT_MINIMUM; index < int(width); index++ {
			workspace.Result[index] = ^workspace.Result[index]
		}
		for index := WORD_COUNT_MINIMUM; index < int(width); index++ {
			workspace.Result[index]++
			if workspace.Result[index] != 0 {
				break
			}
		}
	}
	count := width
	for count > Bitwise_Word_Count(WORD_COUNT_MINIMUM) {
		if workspace.Result[int(count)-WORD_COUNT_INCREMENT] != 0 {
			break
		}
		count--
	}
	if count > Bitwise_Word_Count(WORD_COUNT_MAXIMUM) {
		return STATUS_VALUE_OVERFLOW
	}
	previous_count := destination.Count
	for index := WORD_COUNT_MINIMUM; index < int(count); index++ {
		destination.Words[index] = workspace.Result[index]
	}
	destination.Count = Word_Count(count)
	destination.Negative = POLARITY_NONNEGATIVE
	if negative {
		if destination.Count > WORD_COUNT_MINIMUM {
			destination.Negative = POLARITY_NEGATIVE
		}
	}
	int_clear(destination, destination.Count, previous_count)
	return STATUS_OK
}

func int_set_division_quotient(
	destination *Int, workspace *Int_Division_Workspace, negative Polarity,
) {
	Int_Invariants(destination, "int_set_division_quotient.destination")
	Int_Division_Workspace_Invariants(workspace, "int_set_division_quotient.workspace")
	Polarity_Invariants(negative, "int_set_division_quotient.negative")
	previous_count := destination.Count
	for index := WORD_COUNT_MINIMUM; index < int(workspace.Quotient_Count); index++ {
		destination.Words[index] = workspace.Quotient[index]
	}
	destination.Count = Word_Count(workspace.Quotient_Count)
	destination.Negative = negative
	if destination.Count == WORD_COUNT_MINIMUM {
		destination.Negative = POLARITY_NONNEGATIVE
	}
	int_clear(destination, destination.Count, previous_count)
}

func int_set_division_remainder(
	destination *Int, workspace *Int_Division_Workspace, negative Polarity,
) {
	Int_Invariants(destination, "int_set_division_remainder.destination")
	Int_Division_Workspace_Invariants(workspace, "int_set_division_remainder.workspace")
	Polarity_Invariants(negative, "int_set_division_remainder.negative")
	previous_count := destination.Count
	for index := WORD_COUNT_MINIMUM; index < int(workspace.Remainder_Count); index++ {
		destination.Words[index] = workspace.Remainder[index]
	}
	destination.Count = Word_Count(workspace.Remainder_Count)
	destination.Negative = negative
	if destination.Count == WORD_COUNT_MINIMUM {
		destination.Negative = POLARITY_NONNEGATIVE
	}
	int_clear(destination, destination.Count, previous_count)
}

func int_division_euclidean_adjust(
	workspace *Int_Division_Workspace, dividend *Int, divisor *Int,
) {
	Int_Division_Workspace_Invariants(workspace, "int_division_euclidean_adjust.workspace")
	Int_Invariants(dividend, "int_division_euclidean_adjust.dividend")
	Int_Invariants(divisor, "int_division_euclidean_adjust.divisor")
	if divisor.Count == WORD_COUNT_MINIMUM {
		return
	}
	if dividend.Negative == POLARITY_NONNEGATIVE {
		return
	}
	if workspace.Remainder_Count == WORD_COUNT_MINIMUM {
		return
	}
	borrow := bits.Borrow_In(0)
	for index := WORD_COUNT_MINIMUM; index < int(divisor.Count); index++ {
		subtrahend := Word(0)
		if index < int(workspace.Remainder_Count) {
			subtrahend = workspace.Remainder[index]
		}
		word, next := bits.Subtract_64(
			bits.Word_64(divisor.Words[index]), bits.Subtrahend_64(subtrahend), borrow,
		)
		workspace.Remainder[index] = Word(word)
		borrow = bits.Borrow_In(next)
	}
	workspace.Remainder_Count = Remainder_Count(divisor.Count)
	for workspace.Remainder_Count > WORD_COUNT_MINIMUM {
		high_index := int(workspace.Remainder_Count) - 1
		if workspace.Remainder[high_index] != 0 {
			break
		}
		workspace.Remainder_Count--
	}
	carry := bits.Carry_In(bits.CARRY_MAXIMUM)
	for index := WORD_COUNT_MINIMUM; index < int(workspace.Quotient_Count); index++ {
		word, next := bits.Add_64(bits.Word_64(workspace.Quotient[index]), 0, carry)
		workspace.Quotient[index] = Word(word)
		carry = bits.Carry_In(next)
		if carry == 0 {
			break
		}
	}
	if carry != 0 {
		invariant.Always(
			workspace.Quotient_Count < WORD_COUNT_MAXIMUM,
			"Euclidean quotient adjustment remains inside dividend bound.",
		)
		workspace.Quotient[workspace.Quotient_Count] = Word(carry)
		workspace.Quotient_Count++
	}
}

func int_divide_magnitudes(
	workspace *Int_Division_Workspace, dividend *Int, divisor *Int,
) {
	Int_Division_Workspace_Invariants(workspace, "int_divide_magnitudes.workspace")
	Int_Invariants(dividend, "int_divide_magnitudes.dividend")
	Int_Invariants(divisor, "int_divide_magnitudes.divisor")
	clear_count := max(int(dividend.Count), int(workspace.Quotient_Count),
		int(workspace.Remainder_Count))
	for index := WORD_COUNT_MINIMUM; index < clear_count; index++ {
		workspace.Quotient[index] = 0
		workspace.Remainder[index] = 0
	}
	workspace.Quotient_Count = Quotient_Count(WORD_COUNT_MINIMUM)
	workspace.Remainder_Count = Remainder_Count(WORD_COUNT_MINIMUM)
	if divisor.Count == Word_Count(WORD_COUNT_MINIMUM) {
		return
	}
	if dividend.Count < divisor.Count {
		for index := Word_Count(WORD_COUNT_MINIMUM); index < dividend.Count; index++ {
			workspace.Remainder[index] = dividend.Words[index]
		}
		workspace.Remainder_Count = Remainder_Count(dividend.Count)
		return
	}
	if divisor.Count == Word_Count(WORD_COUNT_INCREMENT) {
		if dividend.Count > Word_Count(WORD_COUNT_INCREMENT) {
			int_divide_by_word(
				(*Int_Division_Empty_Workspace)(workspace),
				(*Int_Division_Multiword_Magnitude)(dividend),
				Int_Division_Nonzero_Word(divisor.Words[WORD_COUNT_MINIMUM]),
			)
			return
		}
	}
	if dividend.Count == divisor.Count {
		if dividend.Count > Word_Count(WORD_COUNT_INCREMENT) {
			int_divide_equal_word_count(
				(*Int_Division_Empty_Workspace)(workspace),
				(*Int_Division_Multiword_Magnitude)(dividend),
				(*Int_Division_Multiword_Magnitude)(divisor),
			)
			return
		}
	}
	if dividend.Count <= Word_Count(BASE_BINARY*BASE_BINARY) {
		invariant.Always(
			divisor.Count >= Word_Count(BASE_BINARY),
			"Earlier word division owns every scalar divisor.",
		)
		invariant.Always(
			dividend.Count > divisor.Count,
			"Earlier equal-width division owns every remaining tie.",
		)
		int_divide_small(
			(*Int_Division_Empty_Workspace)(workspace),
			(*Int_Division_Small_Dividend)(dividend),
			(*Int_Division_Small_Divisor)(divisor),
		)
		return
	}
	bit_position := int(Int_Bit_Count(dividend))
	for bit_position > BIT_COUNT_MINIMUM {
		bit_position--
		int_divide_bit(
			(*Int_Division_Restoring_Workspace)(workspace),
			(*Int_Division_General_Dividend)(dividend),
			(*Int_Division_General_Divisor)(divisor),
			Bit_Index(bit_position),
		)
	}
}

func int_divide_bit(
	workspace *Int_Division_Restoring_Workspace,
	dividend *Int_Division_General_Dividend,
	divisor *Int_Division_General_Divisor,
	bit_position Bit_Index,
) {
	Int_Division_Restoring_Workspace_Invariants(workspace, "int_divide_bit.workspace")
	Int_Division_General_Dividend_Invariants(dividend, "int_divide_bit.dividend")
	Int_Division_General_Divisor_Invariants(divisor, "int_divide_bit.divisor")
	Bit_Index_Invariants(bit_position, "int_divide_bit.bit_position")
	carry := Word(0)
	for index := WORD_COUNT_MINIMUM; index < int(workspace.Remainder_Count); index++ {
		word := workspace.Remainder[index]
		next_carry := word >> WORD_BIT_INDEX_MAXIMUM
		workspace.Remainder[index] = word<<1 | carry
		carry = next_carry
	}
	if carry != 0 {
		workspace.Remainder[workspace.Remainder_Count] = carry
		workspace.Remainder_Count++
	}
	dividend_word_index := int(bit_position) / WORD_BIT_COUNT
	dividend_bit_index := uint(bit_position) % WORD_BIT_COUNT
	dividend_bit := dividend.Words[dividend_word_index] >> dividend_bit_index
	dividend_bit &= Word(bits.CARRY_MAXIMUM)
	if dividend_bit != 0 {
		workspace.Remainder[WORD_COUNT_MINIMUM] |= dividend_bit
		if workspace.Remainder_Count == WORD_COUNT_MINIMUM {
			workspace.Remainder_Count = 1
		}
	}
	order := int_division_order((*Int_Division_Shifted_Workspace)(workspace), divisor)
	if order == ORDER_BEFORE {
		return
	}
	borrow := Word(0)
	for index := WORD_COUNT_MINIMUM; index < int(workspace.Remainder_Count); index++ {
		subtrahend := Word(0)
		if index < int(divisor.Count) {
			subtrahend = divisor.Words[index]
		}
		minuend := workspace.Remainder[index]
		difference := minuend - subtrahend - borrow
		workspace.Remainder[index] = difference
		borrow = ((^minuend & subtrahend) |
			(^(minuend ^ subtrahend) & difference)) >> WORD_BIT_INDEX_MAXIMUM
	}
	for workspace.Remainder_Count > WORD_COUNT_MINIMUM {
		high_index := int(workspace.Remainder_Count) - WORD_COUNT_INCREMENT
		if workspace.Remainder[high_index] != 0 {
			break
		}
		workspace.Remainder_Count--
	}
	quotient_word_index := int(bit_position) / WORD_BIT_COUNT
	quotient_bit_index := uint(bit_position) % WORD_BIT_COUNT
	workspace.Quotient[quotient_word_index] |= Word(bits.CARRY_MAXIMUM) <<
		quotient_bit_index
	quotient_count := Quotient_Count(quotient_word_index + 1)
	if quotient_count > workspace.Quotient_Count {
		workspace.Quotient_Count = quotient_count
	}
}

func int_division_order(
	workspace *Int_Division_Shifted_Workspace, divisor *Int_Division_General_Divisor,
) (order Order) {
	defer func() { Order_Invariants(order, "int_division_order.order") }()
	Int_Division_Shifted_Workspace_Invariants(workspace, "int_division_order.workspace")
	Int_Division_General_Divisor_Invariants(divisor, "int_division_order.divisor")
	divisor_count := Remainder_Count(divisor.Count)
	if workspace.Remainder_Count < divisor_count {
		return ORDER_BEFORE
	}
	if workspace.Remainder_Count > divisor_count {
		return ORDER_AFTER
	}
	for index := int(divisor.Count) - 1; index >= WORD_COUNT_MINIMUM; index-- {
		if workspace.Remainder[index] < divisor.Words[index] {
			return ORDER_BEFORE
		}
		if workspace.Remainder[index] > divisor.Words[index] {
			return ORDER_AFTER
		}
	}
	return ORDER_SAME
}

func int_multiply_words(
	workspace *Int_Multiplication_Workspace, left *Int, right *Int,
) (status Arithmetic_Status) {
	defer func() { Arithmetic_Status_Invariants(status, "int_multiply_words.status") }()
	Int_Multiplication_Workspace_Invariants(workspace, "int_multiply_words.workspace")
	Int_Invariants(left, "int_multiply_words.left")
	Int_Invariants(right, "int_multiply_words.right")
	clear_count := int(left.Count) + int(right.Count)
	if clear_count > WORD_COUNT_MAXIMUM {
		clear_count = WORD_COUNT_MAXIMUM
	}
	for index := WORD_COUNT_MINIMUM; index < clear_count; index++ {
		workspace.Product[index] = 0
	}
	for right_index := WORD_COUNT_MINIMUM; right_index < int(right.Count); right_index++ {
		factor := right.Words[right_index]
		carry := Word(0)
		for left_index := WORD_COUNT_MINIMUM; left_index < int(left.Count); left_index++ {
			position := right_index
			position += left_index
			if position >= WORD_COUNT_MAXIMUM {
				if factor != 0 {
					if left.Words[left_index] != 0 {
						return STATUS_VALUE_OVERFLOW
					}
				}
				if carry != 0 {
					return STATUS_VALUE_OVERFLOW
				}
				continue
			}
			left_word := uint64(left.Words[left_index])
			right_word := uint64(factor)
			word_mask := uint64(bits.WORD_32_MAXIMUM)
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
			low_sum := low + workspace.Product[position]
			stored_carry := Word(0)
			if low_sum < low {
				stored_carry = Word(bits.CARRY_MAXIMUM)
			}
			word := low_sum + carry
			carry_output := Word(0)
			if word < low_sum {
				carry_output = Word(bits.CARRY_MAXIMUM)
			}
			workspace.Product[position] = word
			carry = high + stored_carry + carry_output
		}
		if carry != 0 {
			position := int(right_index) + int(left.Count)
			if position >= WORD_COUNT_MAXIMUM {
				return STATUS_VALUE_OVERFLOW
			}
			workspace.Product[position] = carry
		}
	}
	return STATUS_OK
}

func int_set_word(destination *Int, word Word, negative Polarity) {
	Int_Invariants(destination, "int_set_word.destination")
	Word_Invariants(word, "int_set_word.word")
	Polarity_Invariants(negative, "int_set_word.negative")
	previous_count := destination.Count
	if word == 0 {
		int_zero(destination, previous_count)
		return
	}
	destination.Words[WORD_COUNT_MINIMUM] = word
	destination.Count = 1
	destination.Negative = negative
	if previous_count > destination.Count {
		int_clear(destination, destination.Count, previous_count)
	}
}

func int_zero(destination *Int, previous_count Word_Count) {
	Int_Invariants(destination, "int_zero.destination")
	Word_Count_Invariants(previous_count, "int_zero.previous_count")
	destination.Count = WORD_COUNT_MINIMUM
	destination.Negative = POLARITY_NONNEGATIVE
	if previous_count > Word_Count(WORD_COUNT_MINIMUM) {
		int_clear(destination, WORD_COUNT_MINIMUM, previous_count)
	}
}

func int_clear(destination *Int, start Word_Count, end Word_Count) {
	Int_Invariants(destination, "int_clear.destination")
	Word_Count_Invariants(start, "int_clear.start")
	Word_Count_Invariants(end, "int_clear.end")
	for index := int(start); index < int(end); index++ {
		destination.Words[index] = 0
	}
}

func int_fits_int_64(value *Int) (fits Boolean) {
	defer func() { Boolean_Invariants(fits, "int_fits_int_64.fits") }()
	Int_Invariants(value, "int_fits_int_64.value")
	if value.Count == WORD_COUNT_MINIMUM {
		return true
	}
	if value.Count != 1 {
		return false
	}
	magnitude := uint64(value.Words[WORD_COUNT_MINIMUM])
	if value.Negative == POLARITY_NEGATIVE {
		return Boolean(magnitude <= INT_64_NEGATIVE_MAGNITUDE_MAXIMUM)
	}
	return Boolean(magnitude <= uint64(bits.INTEGER_64_MAXIMUM))
}

func int_compare_absolute(left *Int, right *Int) (order Order) {
	defer func() { Order_Invariants(order, "int_compare_absolute_internal.order") }()
	Int_Invariants(left, "int_compare_absolute_internal.left")
	Int_Invariants(right, "int_compare_absolute_internal.right")
	if left.Count < right.Count {
		return ORDER_BEFORE
	}
	if left.Count > right.Count {
		return ORDER_AFTER
	}
	for index := int(left.Count) - 1; index >= WORD_COUNT_MINIMUM; index-- {
		if left.Words[index] < right.Words[index] {
			return ORDER_BEFORE
		}
		if left.Words[index] > right.Words[index] {
			return ORDER_AFTER
		}
	}
	return ORDER_SAME
}

func int_sum(
	destination *Int,
	left *Int,
	left_negative Polarity,
	right *Int,
	right_negative Polarity,
) (status Arithmetic_Status) {
	defer func() { Arithmetic_Status_Invariants(status, "int_sum.status") }()
	Int_Invariants(destination, "int_sum.destination")
	Int_Invariants(left, "int_sum.left")
	Polarity_Invariants(left_negative, "int_sum.left_negative")
	Int_Invariants(right, "int_sum.right")
	Polarity_Invariants(right_negative, "int_sum.right_negative")
	if left_negative != right_negative {
		int_subtract_magnitudes(
			destination, left, left_negative, right, right_negative,
		)
		status = STATUS_OK
	} else {
		count := left.Count
		if right.Count > count {
			count = right.Count
		}
		carry := Word(0)
		if count == Word_Count(WORD_COUNT_MAXIMUM) {
			for index := Word_Count(WORD_COUNT_MINIMUM); index < count; index++ {
				left_word := Word(0)
				if index < left.Count {
					left_word = left.Words[index]
				}
				right_word := Word(0)
				if index < right.Count {
					right_word = right.Words[index]
				}
				total := left_word + right_word + carry
				carry_bits := (left_word & right_word) |
					((left_word | right_word) &^ total)
				carry = carry_bits >> WORD_BIT_INDEX_MAXIMUM
			}
			if carry != 0 {
				status = STATUS_VALUE_OVERFLOW
			}
		}
		if status == Arithmetic_Status(STATUS_OK) {
			previous_count := destination.Count
			carry = 0
			for index := Word_Count(WORD_COUNT_MINIMUM); index < count; index++ {
				left_word := Word(0)
				if index < left.Count {
					left_word = left.Words[index]
				}
				right_word := Word(0)
				if index < right.Count {
					right_word = right.Words[index]
				}
				total := left_word + right_word + carry
				destination.Words[index] = total
				carry_bits := (left_word & right_word) |
					((left_word | right_word) &^ total)
				carry = carry_bits >> WORD_BIT_INDEX_MAXIMUM
			}
			if carry != 0 {
				destination.Words[count] = carry
				count++
			}
			destination.Count = count
			destination.Negative = left_negative
			if count == WORD_COUNT_MINIMUM {
				destination.Negative = POLARITY_NONNEGATIVE
			}
			int_clear(destination, count, previous_count)
		}
	}
	return status
}

func int_subtract_magnitudes(
	destination *Int,
	left *Int,
	left_negative Polarity,
	right *Int,
	right_negative Polarity,
) {
	Int_Invariants(destination, "int_subtract_magnitudes.destination")
	Int_Invariants(left, "int_subtract_magnitudes.left")
	Polarity_Invariants(left_negative, "int_subtract_magnitudes.left_negative")
	Int_Invariants(right, "int_subtract_magnitudes.right")
	Polarity_Invariants(right_negative, "int_subtract_magnitudes.right_negative")
	order := int_compare_absolute(left, right)
	if order == ORDER_SAME {
		int_zero(destination, destination.Count)
		return
	}
	larger := left
	smaller := right
	negative := left_negative
	if order == ORDER_BEFORE {
		larger = right
		smaller = left
		negative = right_negative
	}
	previous_count := destination.Count
	borrow := Word(0)
	for index := Word_Index(WORD_COUNT_MINIMUM); index < Word_Index(larger.Count); index++ {
		minuend := larger.Words[index]
		subtrahend := Word(0)
		if index < Word_Index(smaller.Count) {
			subtrahend = smaller.Words[index]
		}
		difference := minuend - subtrahend - borrow
		destination.Words[index] = difference
		borrow = ((^minuend & subtrahend) |
			(^(minuend ^ subtrahend) & difference)) >> WORD_BIT_INDEX_MAXIMUM
	}
	count := larger.Count
	for count > WORD_COUNT_MINIMUM {
		if destination.Words[int(count)-1] != 0 {
			break
		}
		count--
	}
	destination.Count = count
	destination.Negative = negative
	if count == WORD_COUNT_MINIMUM {
		destination.Negative = POLARITY_NONNEGATIVE
	}
	int_clear(destination, count, previous_count)
}

func float_set_nonfinite(
	destination *Float, form Float_Nonfinite_Form, negative Polarity,
	precision Float_Precision,
) {
	Float_Invariants(destination, "float_set_nonfinite.destination_initial")
	Float_Nonfinite_Form_Invariants(form, "float_set_nonfinite.form")
	Polarity_Invariants(negative, "float_set_nonfinite.negative")
	Float_Precision_Invariants(precision, "float_set_nonfinite.precision")
	mode := destination.Mode
	previous_count := destination.Mantissa.Count
	destination.Precision, destination.Mode = precision, mode
	destination.Accuracy, destination.Form = ACCURACY_EXACT, Float_Form(form)
	destination.Negative, destination.Exponent = negative, 0
	destination.Mantissa.Count = Word_Count(WORD_COUNT_MINIMUM)
	destination.Mantissa.Negative = POLARITY_NONNEGATIVE
	if previous_count > Word_Count(WORD_COUNT_MINIMUM) {
		int_clear((*Int)(&destination.Mantissa), WORD_COUNT_MINIMUM, previous_count)
	}
}

func float_compare_magnitude(left *Float_Finite, right *Float_Finite) (order Order) {
	defer func() { Order_Invariants(order, "float_compare_magnitude.order") }()
	Float_Finite_Invariants(left, "float_compare_magnitude.left")
	Float_Finite_Invariants(right, "float_compare_magnitude.right")
	if left.Exponent < right.Exponent {
		return ORDER_BEFORE
	}
	if left.Exponent > right.Exponent {
		return ORDER_AFTER
	}
	left_magnitude := Int(left.Mantissa)
	right_magnitude := Int(right.Mantissa)
	left_count := Int_Bit_Count(&left_magnitude)
	right_count := Int_Bit_Count(&right_magnitude)
	if left_count < right_count {
		status := Int_Shift_Left(
			&left_magnitude, &left_magnitude, Shift_Count(right_count-left_count),
		)
		invariant.Always(status == Arithmetic_Status(STATUS_OK),
			"Left normalized comparison shift stops at right mantissa width.")
	} else if right_count < left_count {
		status := Int_Shift_Left(
			&right_magnitude, &right_magnitude, Shift_Count(left_count-right_count),
		)
		invariant.Always(status == Arithmetic_Status(STATUS_OK),
			"Right normalized comparison shift stops at left mantissa width.")
	}
	return Int_Compare_Absolute(&left_magnitude, &right_magnitude)
}

func float_zero_polarity(mode Rounding_Mode) (negative Polarity) {
	defer func() { Polarity_Invariants(negative, "float_zero_polarity.negative") }()
	Rounding_Mode_Invariants(mode, "float_zero_polarity.mode")
	if mode == Rounding_Mode(ROUND_TO_NEGATIVE_INFINITY) {
		return POLARITY_NEGATIVE
	}
	return POLARITY_NONNEGATIVE
}

func float_set_operation_zero(
	destination *Float,
	negative Polarity,
	precision Float_Active_Precision,
	mode Rounding_Mode,
) {
	Float_Invariants(destination, "float_set_operation_zero.destination_initial")
	Polarity_Invariants(negative, "float_set_operation_zero.negative")
	Float_Active_Precision_Invariants(precision, "float_set_operation_zero.precision")
	Rounding_Mode_Invariants(mode, "float_set_operation_zero.mode")
	result := Float{Precision: Float_Precision(precision), Mode: mode}
	float_set_nonfinite(
		&result, Float_Nonfinite_Form(FLOAT_FORM_ZERO), negative,
		Float_Precision(precision),
	)
	*destination = result
}

func float_set_operation_operand(
	destination *Float,
	source *Float,
	precision Float_Precision,
	mode Rounding_Mode,
) {
	Float_Invariants(destination, "float_set_operation_operand.destination_initial")
	Float_Invariants(source, "float_set_operation_operand.source")
	Float_Precision_Invariants(precision, "float_set_operation_operand.precision")
	Rounding_Mode_Invariants(mode, "float_set_operation_operand.mode")
	result := Float{Precision: precision, Mode: mode}
	Float_Set(&result, source)
	*destination = result
}

func float_set_operation_infinity(
	destination *Float,
	negative Polarity,
	precision Float_Precision,
	mode Rounding_Mode,
) {
	Float_Invariants(destination, "float_set_operation_infinity.destination_initial")
	Polarity_Invariants(negative, "float_set_operation_infinity.negative")
	Float_Precision_Invariants(precision, "float_set_operation_infinity.precision")
	Rounding_Mode_Invariants(mode, "float_set_operation_infinity.mode")
	result := Float{Precision: precision, Mode: mode}
	float_set_nonfinite(
		&result, Float_Nonfinite_Form(FLOAT_FORM_INFINITY), negative, precision,
	)
	*destination = result
}

func float_multiply_finite(
	destination *Float,
	left *Float_Finite,
	right *Float_Finite,
	workspace *Float_Multiplication_Workspace,
	precision Float_Active_Precision,
	mode Rounding_Mode,
	negative Polarity,
) {
	Float_Invariants(destination, "float_multiply_finite.destination_initial")
	Float_Finite_Invariants(left, "float_multiply_finite.left")
	Float_Finite_Invariants(right, "float_multiply_finite.right")
	Float_Multiplication_Workspace_Invariants(workspace, "float_multiply_finite.workspace")
	Float_Active_Precision_Invariants(precision, "float_multiply_finite.precision")
	Rounding_Mode_Invariants(mode, "float_multiply_finite.mode")
	Polarity_Invariants(negative, "float_multiply_finite.negative")
	if destination.Precision != FLOAT_PRECISION_MINIMUM {
		if left.Mantissa.Count == Word_Count(BASE_BINARY) {
			if right.Mantissa.Count == Word_Count(BASE_BINARY) {
				references := Float_References{
					destination, (*Float)(left), (*Float)(right),
				}
				if float_multiply_exact_double_word(&references, workspace) {
					return
				}
			}
		}
	}
	result_count := int(left.Mantissa.Count) + int(right.Mantissa.Count)
	for index := WORD_COUNT_MINIMUM; index < result_count; index++ {
		workspace.Result[index] = 0
	}
	float_multiply_words(workspace, left, right)
	left_bits := int(Int_Bit_Count((*Int)(&left.Mantissa)))
	right_bits := int(Int_Bit_Count((*Int)(&right.Mantissa)))
	origin := int(left.Exponent) - left_bits + int(right.Exponent) - right_bits
	addition_workspace := (*Float_Addition_Workspace)(workspace)
	count := float_addition_count(
		addition_workspace, Float_Addition_Result_Word_Count(result_count),
	)
	float_set_addition_result(
		destination, addition_workspace, count, Float_Result_Origin(origin), negative,
		precision, mode,
	)
}

func float_multiply_words(
	workspace *Float_Multiplication_Workspace,
	left *Float_Finite,
	right *Float_Finite,
) {
	Float_Multiplication_Workspace_Invariants(workspace, "float_multiply_words.workspace")
	Float_Finite_Invariants(left, "float_multiply_words.left")
	Float_Finite_Invariants(right, "float_multiply_words.right")
	left_count := int(left.Mantissa.Count)
	right_count := int(right.Mantissa.Count)
	word_mask := uint64(bits.WORD_32_MAXIMUM)
	for right_offset := WORD_COUNT_MINIMUM; right_offset < right_count; right_offset++ {
		right_word := uint64(right.Mantissa.Words[right_offset])
		carry := Word(0)
		for left_offset := WORD_COUNT_MINIMUM; left_offset < left_count; left_offset++ {
			position := right_offset
			position += left_offset
			left_word := uint64(left.Mantissa.Words[left_offset])
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
			position += left_count
			workspace.Result[position] = carry
		}
	}
}

func float_add_finite(
	destination *Float, left *Float_Finite, right *Float_Finite,
	workspace *Float_Addition_Workspace, precision Float_Active_Precision,
	mode Rounding_Mode, right_negative Polarity,
) {
	Float_Invariants(destination, "float_add_finite.destination_initial")
	Float_Finite_Invariants(left, "float_add_finite.left")
	Float_Finite_Invariants(right, "float_add_finite.right")
	Float_Addition_Workspace_Invariants(workspace, "float_add_finite.workspace")
	Float_Active_Precision_Invariants(precision, "float_add_finite.precision")
	Rounding_Mode_Invariants(mode, "float_add_finite.mode")
	Polarity_Invariants(right_negative, "float_add_finite.right_negative")
	left_bits := int(Int_Bit_Count((*Int)(&left.Mantissa)))
	right_bits := int(Int_Bit_Count((*Int)(&right.Mantissa)))
	left_least := int(left.Exponent) - left_bits
	right_least := int(right.Exponent) - right_bits
	origin := min(left_least, right_least)
	left_end := int(left.Mantissa.Count)*WORD_BIT_COUNT + left_least - origin
	right_end := int(right.Mantissa.Count)*WORD_BIT_COUNT + right_least - origin
	result_bit_count := left_end
	if right_end > result_bit_count {
		result_bit_count = right_end
	}
	result_bit_count += WORD_COUNT_INCREMENT
	result_count := (result_bit_count + WORD_BIT_COUNT - WORD_COUNT_INCREMENT) /
		WORD_BIT_COUNT
	if result_count > len(workspace.Result) {
		result_count = len(workspace.Result)
	}
	for index := WORD_COUNT_MINIMUM; index < result_count; index++ {
		workspace.Result[index] = 0
	}
	negative := left.Negative
	if left.Negative == right_negative {
		float_addition_load(workspace, left, Float_Addition_Shift(left_least-origin))
		float_addition_add(workspace, right, Float_Addition_Shift(right_least-origin))
	} else {
		order := float_compare_magnitude(left, right)
		if order == ORDER_SAME {
			float_set_operation_zero(
				destination, float_zero_polarity(mode),
				precision, mode,
			)
			return
		}
		larger := left
		smaller := right
		if order == ORDER_BEFORE {
			larger = right
			smaller = left
			negative = right_negative
		}
		larger_bits := int(Int_Bit_Count((*Int)(&larger.Mantissa)))
		smaller_bits := int(Int_Bit_Count((*Int)(&smaller.Mantissa)))
		larger_least := int(larger.Exponent) - larger_bits
		smaller_least := int(smaller.Exponent) - smaller_bits
		float_addition_load(
			workspace, larger, Float_Addition_Shift(larger_least-origin),
		)
		float_addition_subtract(
			workspace, (*Float_Active_Mantissa)(&smaller.Mantissa),
			Float_Subtraction_Shift(smaller_least-origin),
			Float_Addition_Result_Word_Count(result_count),
		)
	}
	count := float_addition_count(
		workspace, Float_Addition_Result_Word_Count(result_count),
	)
	float_set_addition_result(
		destination, workspace, count, Float_Result_Origin(origin), negative,
		precision, mode,
	)
}

func float_addition_load(
	workspace *Float_Addition_Workspace,
	source *Float_Finite,
	shift Float_Addition_Shift,
) {
	Float_Addition_Workspace_Invariants(workspace, "float_addition_load.workspace")
	Float_Finite_Invariants(source, "float_addition_load.source")
	Float_Addition_Shift_Invariants(shift, "float_addition_load.shift")
	word_shift := int(shift) / WORD_BIT_COUNT
	bit_shift := uint(shift) % WORD_BIT_COUNT
	source_count := int(source.Mantissa.Count)
	for source_index := WORD_COUNT_MINIMUM; source_index < source_count; source_index++ {
		destination_index := word_shift + source_index
		workspace.Result[destination_index] |=
			source.Mantissa.Words[source_index] << bit_shift
		if bit_shift != 0 {
			high_index := destination_index + WORD_COUNT_INCREMENT
			if high_index < len(workspace.Result) {
				workspace.Result[high_index] |=
					source.Mantissa.Words[source_index] >>
						(WORD_BIT_COUNT - bit_shift)
			}
		}
	}
}

func float_addition_add(
	workspace *Float_Addition_Workspace,
	source *Float_Finite,
	shift Float_Addition_Shift,
) {
	Float_Addition_Workspace_Invariants(workspace, "float_addition_add.workspace")
	Float_Finite_Invariants(source, "float_addition_add.source")
	Float_Addition_Shift_Invariants(shift, "float_addition_add.shift")
	word_shift := int(shift) / WORD_BIT_COUNT
	bit_shift := uint(shift) % WORD_BIT_COUNT
	end := word_shift + int(source.Mantissa.Count)
	if bit_shift != 0 {
		end++
	}
	if end > len(workspace.Result) {
		end = len(workspace.Result)
	}
	carry := bits.Carry_In(bits.CARRY_MINIMUM)
	for destination_index := word_shift; destination_index < end; destination_index++ {
		source_index := destination_index - word_shift
		word := Word(0)
		if source_index < int(source.Mantissa.Count) {
			word = source.Mantissa.Words[source_index] << bit_shift
		}
		if bit_shift != 0 {
			if source_index > WORD_COUNT_MINIMUM {
				word |= source.Mantissa.Words[source_index-WORD_COUNT_INCREMENT] >>
					(WORD_BIT_COUNT - bit_shift)
			}
		}
		sum, next := bits.Add_Word(
			bits.Word(workspace.Result[destination_index]),
			bits.Addend_Word(word), carry,
		)
		workspace.Result[destination_index] = Word(sum)
		carry = bits.Carry_In(next)
	}
	for carry != bits.Carry_In(bits.CARRY_MINIMUM) {
		invariant.Always(end < len(workspace.Result),
			"Finite addition carry remains inside the complete bit-position span.")
		sum, next := bits.Add_Word(
			bits.Word(workspace.Result[end]), 0, carry,
		)
		workspace.Result[end] = Word(sum)
		carry = bits.Carry_In(next)
		end++
	}
}

func float_addition_subtract(
	workspace *Float_Addition_Workspace,
	source *Float_Active_Mantissa,
	shift Float_Subtraction_Shift,
	limit Float_Addition_Result_Word_Count,
) {
	Float_Addition_Workspace_Invariants(workspace, "float_addition_subtract.workspace")
	Float_Active_Mantissa_Invariants(*source, "float_addition_subtract.source")
	Float_Subtraction_Shift_Invariants(shift, "float_addition_subtract.shift")
	Float_Addition_Result_Word_Count_Invariants(
		limit, "float_addition_subtract.limit",
	)
	word_shift := int(shift) / WORD_BIT_COUNT
	bit_shift := uint(shift) % WORD_BIT_COUNT
	borrow := bits.Borrow_In(bits.CARRY_MINIMUM)
	result_count := int(limit)
	for destination_index := word_shift; destination_index < result_count; destination_index++ {
		source_index := destination_index - word_shift
		word := Word(0)
		if source_index < int(source.Count) {
			word = source.Words[source_index] << bit_shift
		}
		if bit_shift != 0 {
			if source_index > WORD_COUNT_MINIMUM {
				if source_index <= int(source.Count) {
					source_word_index := source_index - WORD_COUNT_INCREMENT
					word |= source.Words[source_word_index] >>
						(WORD_BIT_COUNT - bit_shift)
				}
			}
		}
		difference, next := bits.Subtract_Word(
			bits.Word(workspace.Result[destination_index]),
			bits.Subtrahend_Word(word), borrow,
		)
		workspace.Result[destination_index] = Word(difference)
		borrow = bits.Borrow_In(next)
	}
	invariant.Always(borrow == bits.Borrow_In(bits.CARRY_MINIMUM),
		"Magnitude ordering keeps aligned subtraction nonnegative.")
}

func float_addition_count(
	workspace *Float_Addition_Workspace,
	limit Float_Addition_Result_Word_Count,
) (count Float_Addition_Active_Word_Count) {
	defer func() {
		Float_Addition_Active_Word_Count_Invariants(count, "float_addition_count.count")
	}()
	Float_Addition_Workspace_Invariants(workspace, "float_addition_count.workspace")
	Float_Addition_Result_Word_Count_Invariants(limit, "float_addition_count.limit")
	count = Float_Addition_Active_Word_Count(limit)
	for workspace.Result[int(count)-WORD_COUNT_INCREMENT] == 0 {
		count--
	}
	return count
}

func float_set_addition_result(
	destination *Float,
	workspace *Float_Addition_Workspace,
	count Float_Addition_Active_Word_Count,
	origin Float_Result_Origin,
	negative Polarity,
	precision Float_Active_Precision,
	mode Rounding_Mode,
) {
	Float_Invariants(destination, "float_set_addition_result.destination_initial")
	Float_Addition_Workspace_Invariants(workspace, "float_set_addition_result.workspace")
	Float_Addition_Active_Word_Count_Invariants(count, "float_set_addition_result.count")
	Float_Result_Origin_Invariants(origin, "float_set_addition_result.origin")
	Polarity_Invariants(negative, "float_set_addition_result.negative")
	Float_Active_Precision_Invariants(precision, "float_set_addition_result.precision")
	Rounding_Mode_Invariants(mode, "float_set_addition_result.mode")
	high := bits.Word_64(workspace.Result[int(count)-WORD_COUNT_INCREMENT])
	bit_count := (int(count)-WORD_COUNT_INCREMENT)*WORD_BIT_COUNT +
		bits.BIT_COUNT_64_MAXIMUM - int(bits.Leading_Zeros_64(high))
	exponent := int(origin) + bit_count
	if exponent < FLOAT_EXPONENT_MINIMUM {
		float_set_nonfinite(
			destination, Float_Nonfinite_Form(FLOAT_FORM_ZERO), negative,
			Float_Precision(precision),
		)
		destination.Mode = mode
		destination.Accuracy = Accuracy(float_accuracy(false, negative))
		return
	}
	if exponent > FLOAT_EXPONENT_MAXIMUM {
		float_set_nonfinite(
			destination, Float_Nonfinite_Form(FLOAT_FORM_INFINITY), negative,
			Float_Precision(precision),
		)
		destination.Mode = mode
		destination.Accuracy = Accuracy(float_accuracy(true, negative))
		return
	}
	float_set_addition_mantissa(
		destination, workspace, Float_Addition_Bit_Count(bit_count),
		Float_Exponent(exponent), negative,
		precision, mode,
	)
}

func float_set_addition_mantissa(
	destination *Float,
	workspace *Float_Addition_Workspace,
	bit_count Float_Addition_Bit_Count,
	exponent Float_Exponent,
	negative Polarity,
	precision Float_Active_Precision,
	mode Rounding_Mode,
) {
	Float_Invariants(destination, "float_set_addition_mantissa.destination_initial")
	Float_Addition_Workspace_Invariants(workspace, "float_set_addition_mantissa.workspace")
	Float_Addition_Bit_Count_Invariants(bit_count, "float_set_addition_mantissa.bit_count")
	Float_Exponent_Invariants(exponent, "float_set_addition_mantissa.exponent")
	Polarity_Invariants(negative, "float_set_addition_mantissa.negative")
	Float_Active_Precision_Invariants(precision, "float_set_addition_mantissa.precision")
	Rounding_Mode_Invariants(mode, "float_set_addition_mantissa.mode")
	retained_count := int(bit_count)
	if retained_count > int(precision) {
		retained_count = int(precision)
	}
	discard_count := int(bit_count) - retained_count
	mantissa_count := (retained_count + WORD_BIT_COUNT - WORD_COUNT_INCREMENT) /
		WORD_BIT_COUNT
	previous_count := destination.Mantissa.Count
	for index := WORD_COUNT_MINIMUM; index < mantissa_count; index++ {
		source_bit := discard_count + index*WORD_BIT_COUNT
		source_index := source_bit / WORD_BIT_COUNT
		source_shift := uint(source_bit % WORD_BIT_COUNT)
		word := workspace.Result[source_index] >> source_shift
		if source_shift != 0 {
			if source_index+WORD_COUNT_INCREMENT < len(workspace.Result) {
				word |= workspace.Result[source_index+WORD_COUNT_INCREMENT] <<
					(WORD_BIT_COUNT - source_shift)
			}
		}
		destination.Mantissa.Words[index] = word
	}
	destination.Precision, destination.Mode = Float_Precision(precision), mode
	destination.Accuracy, destination.Form = ACCURACY_EXACT, FLOAT_FORM_FINITE
	destination.Negative, destination.Exponent = negative, exponent
	destination.Mantissa.Count = Word_Count(mantissa_count)
	destination.Mantissa.Negative = POLARITY_NONNEGATIVE
	if previous_count > destination.Mantissa.Count {
		int_clear(
			(*Int)(&destination.Mantissa), destination.Mantissa.Count, previous_count,
		)
	}
	accuracy := ACCURACY_EXACT
	if discard_count > BIT_COUNT_MINIMUM {
		accuracy = float_round_addition_mantissa(
			(*Float_Active_Mantissa)(&destination.Mantissa), workspace,
			Float_Addition_Active_Discard_Count(discard_count), &exponent,
			negative, precision, mode,
		)
	}
	if exponent > FLOAT_EXPONENT_MAXIMUM {
		float_set_nonfinite(
			destination, Float_Nonfinite_Form(FLOAT_FORM_INFINITY), negative,
			Float_Precision(precision),
		)
		destination.Mode = mode
		destination.Accuracy = Accuracy(float_accuracy(true, negative))
		return
	}
	destination.Accuracy, destination.Exponent = accuracy, exponent
}

func float_round_addition_mantissa(
	mantissa *Float_Active_Mantissa,
	workspace *Float_Addition_Workspace,
	discard_count Float_Addition_Active_Discard_Count,
	exponent *Float_Exponent,
	negative Polarity,
	precision Float_Active_Precision,
	mode Rounding_Mode,
) (accuracy Accuracy) {
	defer func() { Accuracy_Invariants(accuracy, "float_round_addition_mantissa.accuracy") }()
	Float_Active_Mantissa_Invariants(
		*mantissa, "float_round_addition_mantissa.mantissa_initial",
	)
	Float_Addition_Workspace_Invariants(workspace, "float_round_addition_mantissa.workspace")
	Float_Addition_Active_Discard_Count_Invariants(
		discard_count, "float_round_addition_mantissa.discard_count",
	)
	Float_Exponent_Invariants(*exponent, "float_round_addition_mantissa.exponent_initial")
	Polarity_Invariants(negative, "float_round_addition_mantissa.negative")
	Float_Active_Precision_Invariants(precision, "float_round_addition_mantissa.precision")
	Rounding_Mode_Invariants(mode, "float_round_addition_mantissa.mode")
	rounding_index := int(discard_count) - WORD_COUNT_INCREMENT
	rounding_bit := Bit_Value(
		workspace.Result[rounding_index/WORD_BIT_COUNT] >>
			uint(rounding_index%WORD_BIT_COUNT) & 1,
	)
	sticky_bit := float_addition_sticky(
		workspace, Float_Addition_Rounding_Index(rounding_index),
	)
	increment := float_rounding_increment(
		mode, negative, rounding_bit, sticky_bit,
		Bit_Value(mantissa.Words[WORD_COUNT_MINIMUM]&1),
	)
	accuracy = ACCURACY_EXACT
	if rounding_bit != BIT_CLEAR {
		accuracy = Accuracy(float_accuracy(increment, negative))
	} else if sticky_bit != BIT_CLEAR {
		accuracy = Accuracy(float_accuracy(increment, negative))
	}
	if increment {
		float_increment_mantissa(mantissa, precision, exponent)
	}
	return accuracy
}

func float_addition_sticky(
	workspace *Float_Addition_Workspace, rounding_index Float_Addition_Rounding_Index,
) (sticky Bit_Value) {
	defer func() { Bit_Value_Invariants(sticky, "float_addition_sticky.sticky") }()
	Float_Addition_Workspace_Invariants(workspace, "float_addition_sticky.workspace")
	Float_Addition_Rounding_Index_Invariants(
		rounding_index, "float_addition_sticky.rounding_index",
	)
	word_count := int(rounding_index) / WORD_BIT_COUNT
	for index := WORD_COUNT_MINIMUM; index < word_count; index++ {
		if workspace.Result[index] != 0 {
			return BIT_SET
		}
	}
	partial_count := uint(int(rounding_index) % WORD_BIT_COUNT)
	if partial_count == 0 {
		return BIT_CLEAR
	}
	mask := Word(bits.WORD_64_MAXIMUM) >> (WORD_BIT_COUNT - partial_count)
	if workspace.Result[word_count]&mask != 0 {
		return BIT_SET
	}
	return BIT_CLEAR
}

func float_increment_mantissa(
	mantissa *Float_Active_Mantissa,
	precision Float_Active_Precision,
	exponent *Float_Exponent,
) {
	Float_Active_Mantissa_Invariants(*mantissa, "float_increment_mantissa.mantissa_initial")
	Float_Active_Precision_Invariants(precision, "float_increment_mantissa.precision")
	Float_Exponent_Invariants(*exponent, "float_increment_mantissa.exponent_initial")
	carry := bits.Carry_In(bits.CARRY_MAXIMUM)
	for index := WORD_COUNT_MINIMUM; index < int(mantissa.Count); index++ {
		sum, next := bits.Add_Word(bits.Word(mantissa.Words[index]), 0, carry)
		mantissa.Words[index] = Word(sum)
		carry = bits.Carry_In(next)
		if carry == bits.Carry_In(bits.CARRY_MINIMUM) {
			break
		}
	}
	overflow := carry != bits.Carry_In(bits.CARRY_MINIMUM)
	if !overflow {
		overflow = int(Int_Bit_Count((*Int)(mantissa))) > int(precision)
	}
	if overflow {
		for index := range mantissa.Words {
			mantissa.Words[index] = 0
		}
		high_index := (int(precision) - WORD_COUNT_INCREMENT) / WORD_BIT_COUNT
		high_bit := uint((int(precision) - WORD_COUNT_INCREMENT) % WORD_BIT_COUNT)
		mantissa.Words[high_index] = Word(bits.CARRY_MAXIMUM) << high_bit
		mantissa.Count = Word_Count(high_index + WORD_COUNT_INCREMENT)
		(*exponent)++
	}
}

func float_set_magnitude(
	destination *Float,
	magnitude *Float_Mantissa,
	exponent Float_Exponent,
	negative Polarity,
	precision Float_Active_Precision,
	mode Rounding_Mode,
) {
	Float_Invariants(destination, "float_set_magnitude.destination_initial")
	Float_Mantissa_Invariants(*magnitude, "float_set_magnitude.magnitude")
	Float_Exponent_Invariants(exponent, "float_set_magnitude.exponent")
	Polarity_Invariants(negative, "float_set_magnitude.negative")
	Float_Active_Precision_Invariants(precision, "float_set_magnitude.precision")
	Rounding_Mode_Invariants(mode, "float_set_magnitude.mode")
	if magnitude.Count == WORD_COUNT_MINIMUM {
		float_set_nonfinite(
			destination, Float_Nonfinite_Form(FLOAT_FORM_ZERO), negative,
			Float_Precision(precision),
		)
		destination.Mode = mode
		return
	}
	bit_count := Int_Bit_Count((*Int)(magnitude))
	if Float_Precision(bit_count) <= Float_Precision(precision) {
		previous_count := destination.Mantissa.Count
		for index := Word_Count(WORD_COUNT_MINIMUM); index < magnitude.Count; index++ {
			destination.Mantissa.Words[index] = magnitude.Words[index]
		}
		destination.Precision, destination.Mode = Float_Precision(precision), mode
		destination.Accuracy, destination.Form = ACCURACY_EXACT, FLOAT_FORM_FINITE
		destination.Negative, destination.Exponent = negative, exponent
		destination.Mantissa.Count = magnitude.Count
		destination.Mantissa.Negative = POLARITY_NONNEGATIVE
		if previous_count > destination.Mantissa.Count {
			int_clear(
				(*Int)(&destination.Mantissa), destination.Mantissa.Count,
				previous_count,
			)
		}
		return
	}
	result := Float{
		Precision: Float_Precision(bit_count),
		Mode:      mode,
		Accuracy:  ACCURACY_EXACT,
		Form:      FLOAT_FORM_FINITE,
		Negative:  negative,
		Mantissa:  *magnitude,
		Exponent:  exponent,
	}
	result.Mantissa.Negative = POLARITY_NONNEGATIVE
	float_round(&result, Float_Precision(precision))
	*destination = result
}

func float_set_float_64_finite(
	destination *Float,
	exponent_field Float_64_Finite_Exponent_Field,
	mantissa_field Float_64_Mantissa_Field,
	negative Polarity,
	precision Float_Active_Precision,
) {
	Float_Invariants(destination, "float_set_float_64_finite.destination_initial")
	Float_64_Finite_Exponent_Field_Invariants(
		exponent_field, "float_set_float_64_finite.exponent_field",
	)
	Float_64_Mantissa_Field_Invariants(
		mantissa_field, "float_set_float_64_finite.mantissa_field",
	)
	Polarity_Invariants(negative, "float_set_float_64_finite.negative")
	Float_Active_Precision_Invariants(precision, "float_set_float_64_finite.precision")
	mantissa := uint64(mantissa_field)
	exponent := FLOAT_64_SUBNORMAL_EXPONENT - FLOAT_64_MANTISSA_BIT_COUNT
	if exponent_field != 0 {
		mantissa |= FLOAT_64_HIDDEN_MANTISSA_BIT
		exponent = int(exponent_field) - FLOAT_64_EXPONENT_BIAS + WORD_COUNT_INCREMENT
	} else {
		mantissa_bit_count := bits.BIT_COUNT_64_MAXIMUM -
			int(bits.Leading_Zeros_64(bits.Word_64(mantissa)))
		exponent += mantissa_bit_count
	}
	bit_count := Float_Precision(bits.Bit_Size_64(bits.Word_64(mantissa)))
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
	float_magnitude.Words[WORD_COUNT_MINIMUM] = Word(mantissa)
	float_magnitude.Count = Word_Count(WORD_COUNT_INCREMENT)
	float_set_magnitude(
		destination, &float_magnitude, Float_Exponent(exponent), negative,
		precision, destination.Mode,
	)
}

func float_round(destination *Float, precision Float_Precision) {
	Float_Invariants(destination, "float_round.destination_initial")
	Float_Precision_Invariants(precision, "float_round.precision")
	// Validation leaves no failing step, so a capacity-sized transaction has no rollback value.
	destination.Accuracy = ACCURACY_EXACT
	if destination.Form != FLOAT_FORM_FINITE {
		destination.Precision = precision
		return
	}
	if precision == FLOAT_PRECISION_MINIMUM {
		negative := destination.Negative
		float_set_nonfinite(
			destination, Float_Nonfinite_Form(FLOAT_FORM_ZERO), negative, precision,
		)
		destination.Accuracy = Accuracy(float_accuracy(false, negative))
		return
	}
	bit_count := int(Int_Bit_Count((*Int)(&destination.Mantissa)))
	discard_count := bit_count - int(precision)
	if discard_count <= BIT_COUNT_MINIMUM {
		destination.Precision = precision
		return
	}
	rounding_index := discard_count - WORD_COUNT_INCREMENT
	rounding_word := destination.Mantissa.Words[rounding_index/WORD_BIT_COUNT]
	rounding_bit := Bit_Value(rounding_word >> uint(rounding_index%WORD_BIT_COUNT) & 1)
	sticky_bit := BIT_CLEAR
	if float_low_bits_nonzero(
		(*Float_Active_Mantissa)(&destination.Mantissa),
		Float_Discarded_Bit_Count(rounding_index),
	) {
		sticky_bit = BIT_SET
	}
	Int_Shift_Right(
		(*Int)(&destination.Mantissa), (*Int)(&destination.Mantissa),
		Shift_Count(discard_count),
	)
	least_bit := Bit_Value(destination.Mantissa.Words[WORD_COUNT_MINIMUM] & 1)
	increment := float_rounding_increment(
		destination.Mode, destination.Negative, rounding_bit, sticky_bit, least_bit,
	)
	if rounding_bit != BIT_CLEAR {
		destination.Accuracy = Accuracy(float_accuracy(increment, destination.Negative))
	} else if sticky_bit != BIT_CLEAR {
		destination.Accuracy = Accuracy(float_accuracy(increment, destination.Negative))
	}
	float_round_increment(
		(*Float_Rounding_Source)(destination), increment,
		Float_Reduced_Precision(precision),
	)
}

func float_round_increment(
	destination *Float_Rounding_Source,
	increment Boolean,
	precision Float_Reduced_Precision,
) {
	Float_Rounding_Source_Invariants(destination, "float_round_increment.destination")
	Boolean_Invariants(increment, "float_round_increment.increment")
	Float_Reduced_Precision_Invariants(precision, "float_round_increment.precision")
	if increment {
		var one Int
		Int_Set_Uint_64(&one, Word_64(bits.CARRY_MAXIMUM))
		status := Int_Add(
			(*Int)(&destination.Mantissa), (*Int)(&destination.Mantissa), &one,
		)
		invariant.Always(status == Arithmetic_Status(STATUS_OK),
			"Rounding carry fits one bounded mantissa.")
	}
	if int(Int_Bit_Count((*Int)(&destination.Mantissa))) > int(precision) {
		Int_Shift_Right(
			(*Int)(&destination.Mantissa), (*Int)(&destination.Mantissa), 1,
		)
		if destination.Exponent == FLOAT_EXPONENT_MAXIMUM {
			negative := destination.Negative
			float_set_nonfinite(
				(*Float)(destination), Float_Nonfinite_Form(FLOAT_FORM_INFINITY),
				negative, Float_Precision(precision),
			)
			destination.Accuracy = Accuracy(float_accuracy(true, negative))
			return
		}
		destination.Exponent++
	}
	destination.Precision = Float_Precision(precision)
}

func float_low_bits_nonzero(
	value *Float_Active_Mantissa, count Float_Discarded_Bit_Count,
) (nonzero Boolean) {
	defer func() { Boolean_Invariants(nonzero, "float_low_bits_nonzero.nonzero") }()
	Float_Active_Mantissa_Invariants(*value, "float_low_bits_nonzero.value")
	Float_Discarded_Bit_Count_Invariants(count, "float_low_bits_nonzero.count")
	word_count := int(count) / WORD_BIT_COUNT
	for word_index := WORD_COUNT_MINIMUM; word_index < word_count; word_index++ {
		if value.Words[word_index] != 0 {
			return true
		}
	}
	partial_count := int(count) % WORD_BIT_COUNT
	if partial_count == BIT_COUNT_MINIMUM {
		return false
	}
	mask := Word(bits.WORD_64_MAXIMUM) >> uint(WORD_BIT_COUNT-partial_count)
	return Boolean(value.Words[word_count]&mask != 0)
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
