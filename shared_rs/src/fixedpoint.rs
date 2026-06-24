//! Base-two fixed-point arithmetic backed by `i64`: the deterministic stand-in
//! for `f64` where results must reproduce bit-for-bit across platforms (IEEE-754
//! diverges; an integer scaled by a fixed power of two does not). The scale is a
//! power of two so the rescale a multiply or divide needs is a bit shift, not a
//! base-ten hardware divide. Behavior is free functions over the value, never
//! methods, with the value acted on first.
//!
//! The Go original's JSON `Marshaler`/`Unmarshaler` methods become the free
//! functions `to_decimal`/`from_decimal` here, since the dialect bans methods;
//! automatic struct serialization would need a serde dependency the workspace
//! does not vendor.
//!
//! Add and subtract are the native `i64` operators on the `.0` field — the shared
//! [`SCALE`] already aligns them — so only the scale-cancelling `multiply` and
//! `divide` are functions here.
//!
//! ```
//! use shared_rs::fixedpoint;
//! let product = fixedpoint::multiply(fixedpoint::from_integer(6), fixedpoint::from_integer(7));
//! assert_eq!(fixedpoint::whole(product), 42);
//! ```

/// A base-two fixed-point value: the real number is `.0` divided by [`SCALE`].
#[derive(Copy, Clone, PartialEq, Eq, Hash, Debug, PartialOrd, Ord)]
pub struct Number(pub i64);

// How many low bits of a `Number` hold the fraction; the rest hold the integer
// part. Twenty bits gives ~9.5e-7 precision over a ±8.8e12 range, matching the
// magnitudes the deterministic callers work in.
const FRACTIONAL_BITS: u32 = 20;

/// The count of fixed-point units in one whole — two raised to the fractional
/// bit count. A `Number`'s real value is its stored integer divided by this.
pub const SCALE: i64 = 1 << FRACTIONAL_BITS;

/// Lifts a whole number into fixed-point.
pub fn from_integer(value: i64) -> Number {
    Number(value * SCALE)
}

/// Truncates a value toward zero to the integer it contains.
pub fn whole(value: Number) -> i64 {
    value.0 / SCALE
}

/// Whether a value carries no fractional part.
pub fn is_integer(value: Number) -> bool {
    value.0 % SCALE == 0
}

/// The fixed-point product: the 128-bit product of the factors shifted back down
/// by the scale, so the scale cancels with no divide and no premature overflow.
pub fn multiply(a: Number, b: Number) -> Number {
    let magnitude = multiply_shifted(a.0.unsigned_abs(), b.0.unsigned_abs());
    signed(magnitude, (a.0 < 0) != (b.0 < 0))
}

/// The fixed-point quotient: the dividend scaled up through a 128-bit
/// intermediate, then one divide by the divisor. A zero divisor yields zero.
pub fn divide(dividend: Number, divisor: Number) -> Number {
    match divisor.0 == 0 {
        true => Number(0),
        false => {
            let magnitude = shift_divide(dividend.0.unsigned_abs(), divisor.0.unsigned_abs());
            signed(magnitude, (dividend.0 < 0) != (divisor.0 < 0))
        }
    }
}

/// The sine of an angle measured in whole turns, reduced to one period and
/// approximated by Bhaskara's rational formula for `sin(pi*theta)` — so no
/// irrational pi enters and the quarter-turn extremes land exactly on ±1.
pub fn sine_turns(turns: Number) -> Number {
    // Reduce to one period, [0, SCALE) units of a turn.
    let reduced = turns.0.rem_euclid(SCALE);
    // The second half-turn mirrors the first, negated.
    let negative = reduced >= SCALE / 2;
    let fraction = match negative {
        true => reduced - SCALE / 2,
        false => reduced,
    };
    // Theta in [0,1] is twice the half-period fraction; theta*(1-theta) drives
    // Bhaskara's 16p / (5 - 4p) approximation of sin(pi*theta).
    let theta = fraction * 2;
    let product = multiply(Number(theta), Number(SCALE - theta));
    let numerator = Number(16 * product.0);
    let denominator = Number(from_integer(5).0 - 4 * product.0);
    let magnitude = divide(numerator, denominator);
    match negative {
        true => Number(-magnitude.0),
        false => magnitude,
    }
}

// `left * right` rescaled by the scale: the 128-bit product shifted down by the
// fractional bits — a shift where a base-ten scale would need a divide. Callers
// pass factors whose true product fits `i64`, so the shifted-away high bits are
// zero.
fn multiply_shifted(left: u64, right: u64) -> u64 {
    ((u128::from(left) * u128::from(right)) >> FRACTIONAL_BITS) as u64
}

// `(numerator << fractional_bits) / denominator` through a 128-bit intermediate,
// so the scaled-up numerator keeps full precision without overflowing. Callers
// pass a nonzero denominator and a result that fits `i64`.
fn shift_divide(numerator: u64, denominator: u64) -> u64 {
    ((u128::from(numerator) << FRACTIONAL_BITS) / u128::from(denominator)) as u64
}

// Reapplies a sign stripped for the unsigned 128-bit math.
fn signed(magnitude: u64, negative: bool) -> Number {
    match negative {
        true => Number((magnitude as i64).wrapping_neg()),
        false => Number(magnitude as i64),
    }
}

/// A dimensionless fixed-point multiplier: the real factor is `.0` over [`SCALE`].
/// Kept distinct from [`Number`] so `apply` takes one of each.
#[derive(Copy, Clone, PartialEq, Eq, Hash, Debug, PartialOrd, Ord)]
pub struct Ratio(pub i64);

/// Lifts the quotient `numerator / denominator` into fixed-point through a 128-bit
/// intermediate, so the scaled-up numerator keeps full precision. A zero
/// denominator yields zero, not a divide by zero.
pub fn from_ratio(numerator: i64, denominator: i64) -> Number {
    match denominator == 0 {
        true => Number(0),
        false => {
            let magnitude = shift_divide(numerator.unsigned_abs(), denominator.unsigned_abs());
            signed(magnitude, (numerator < 0) != (denominator < 0))
        }
    }
}

/// Scales a value by a dimensionless ratio.
pub fn apply(value: Number, ratio: Ratio) -> Number {
    let magnitude = multiply_shifted(value.0.unsigned_abs(), ratio.0.unsigned_abs());
    signed(magnitude, (value.0 < 0) != (ratio.0 < 0))
}

/// The fixed-point square root of a fixed-point value. A negative input has no
/// real root and yields zero.
pub fn square_root(value: Number) -> Number {
    match value.0 < 0 {
        true => Number(0),
        // The root of value/Scale, scaled back up, is the root of value*Scale; the
        // product can exceed i64, so it is rooted as a 128-bit radicand.
        false => Number(integer_root(u128::from(value.0 as u64) * SCALE as u128) as i64),
    }
}

/// The fixed-point square root of a plain integer, for a sum of squares whose
/// magnitude would overflow if first lifted into fixed-point. A negative input
/// yields zero.
pub fn square_root_scaled(value: i64) -> Number {
    match value < 0 {
        true => Number(0),
        // The root of value, scaled up by Scale, is the root of value*Scale*Scale.
        false => {
            let radicand = u128::from(value as u64) * SCALE as u128 * SCALE as u128;
            Number(integer_root(radicand) as i64)
        }
    }
}

/// The floor of the square root of a 128-bit radicand — the integer primitive the
/// fixed-point roots build on, and the escape hatch for a sum of squares too large
/// to lift into fixed-point first.
pub fn integer_root(radicand: u128) -> u64 {
    match radicand == 0 {
        true => 0,
        false => {
            // Seed at two raised to half the bit length: an overestimate Newton's
            // method drives down to the floor in a handful of steps.
            let seed = 1u128 << (128 - radicand.leading_zeros()).div_ceil(2);
            integer_root_step(radicand, seed, 64)
        }
    }
}

// One Newton step toward the floor root, recursing until the estimate stops
// decreasing or the step budget (a safety bound never reached in practice) runs
// out. Mut-free, so the iteration is recursion, not a mutating loop.
fn integer_root_step(radicand: u128, estimate: u128, remaining: u32) -> u64 {
    match remaining == 0 {
        true => estimate as u64,
        false => {
            let next = (estimate + radicand / estimate) / 2;
            match next >= estimate {
                true => estimate as u64,
                false => integer_root_step(radicand, next, remaining - 1),
            }
        }
    }
}

/// Renders a value as decimal text with `digits` fractional digits, the dropped
/// remainder rounded half away from zero.
pub fn format(value: Number, digits: u32) -> String {
    let scaled = decimal_scaled(value.0.unsigned_abs(), digits);
    let negative = value.0 < 0 && scaled != 0;
    let power = power_of_ten(digits);
    let whole_text = (scaled / power).to_string();
    let body = match digits > 0 {
        true => format!("{whole_text}.{:0width$}", scaled % power, width = digits as usize),
        false => whole_text,
    };
    match negative {
        true => format!("-{body}"),
        false => body,
    }
}

/// Renders a value as a bare decimal: an integer when whole, else six fractional
/// digits with trailing zeros trimmed — the form a JSON number would take.
pub fn to_decimal(number: Number) -> String {
    match is_integer(number) {
        true => whole(number).to_string(),
        false => trim_trailing_zeros(format(number, 6)),
    }
}

/// Parses a bare decimal onto the fixed-point grid, rounding the fraction. `None`
/// when the integer part is not a valid number.
pub fn from_decimal(text: &str) -> Option<Number> {
    let (negative, body) = match text.strip_prefix('-') {
        Some(rest) => (true, rest),
        None => (false, text),
    };
    let (whole_text, fraction_text) = match body.split_once('.') {
        Some((whole_part, fraction)) => (whole_part, fraction),
        None => (body, ""),
    };
    let whole_value: i64 = whole_text.parse().ok()?;
    let magnitude = whole_value * SCALE + fraction_to_units(fraction_text);
    Some(Number(match negative {
        true => -magnitude,
        false => magnitude,
    }))
}

// Rounds magnitude * 10^digits / Scale to the magnitude as an integer with
// `digits` decimal places, the divide by the power-of-two Scale done as a
// rounding add and a shift.
fn decimal_scaled(magnitude: u64, digits: u32) -> u64 {
    let product = u128::from(magnitude) * u128::from(power_of_ten(digits));
    ((product + (1u128 << (FRACTIONAL_BITS - 1))) >> FRACTIONAL_BITS) as u64
}

// Ten raised to a small non-negative exponent.
fn power_of_ten(exponent: u32) -> u64 {
    (0..exponent).fold(1u64, |result, _| result * 10)
}

// Drops trailing zeros, and a now-trailing point, leaving the minimal exact
// decimal.
fn trim_trailing_zeros(text: String) -> String {
    text.trim_end_matches('0').trim_end_matches('.').to_string()
}

// Reads decimal fraction digits as a count of fixed-point units, rounding. Digits
// past the ninth are dropped, far beyond the grid's resolution.
fn fraction_to_units(fraction_text: &str) -> i64 {
    let digits: String = fraction_text.chars().take(9).collect();
    match digits.parse::<i64>() {
        Ok(parsed) => {
            let power = power_of_ten(digits.len() as u32) as i64;
            ((parsed << FRACTIONAL_BITS) + power / 2) / power
        }
        Err(_) => 0,
    }
}
