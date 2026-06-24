//! Black-box: the deterministic fixed-point arithmetic core — integer lift and
//! truncate, scale-cancelling multiply/divide, and Bhaskara sine whose quarter
//! turns land exactly on ±1.

use shared_rs::fixedpoint;

#[test]
fn integer_lift_and_truncate_round_trip() {
    assert_eq!(fixedpoint::whole(fixedpoint::from_integer(7)), 7);
    assert_eq!(fixedpoint::whole(fixedpoint::from_integer(-3)), -3);
    assert!(fixedpoint::is_integer(fixedpoint::from_integer(5)));
}

#[test]
fn multiply_cancels_the_scale() {
    let product = fixedpoint::multiply(fixedpoint::from_integer(6), fixedpoint::from_integer(7));
    assert_eq!(fixedpoint::whole(product), 42);
}

#[test]
fn divide_cancels_the_scale_and_guards_zero() {
    let quotient = fixedpoint::divide(fixedpoint::from_integer(20), fixedpoint::from_integer(4));
    assert_eq!(fixedpoint::whole(quotient), 5);
    assert_eq!(
        fixedpoint::divide(fixedpoint::from_integer(1), fixedpoint::from_integer(0)),
        fixedpoint::Number(0),
    );
}

#[test]
fn sine_turns_hits_the_quarter_extremes_exactly() {
    let scale = fixedpoint::SCALE;
    assert_eq!(fixedpoint::sine_turns(fixedpoint::Number(0)), fixedpoint::Number(0));
    assert_eq!(fixedpoint::sine_turns(fixedpoint::Number(scale / 4)), fixedpoint::Number(scale));
    assert_eq!(fixedpoint::sine_turns(fixedpoint::Number(scale / 2)), fixedpoint::Number(0));
    assert_eq!(
        fixedpoint::sine_turns(fixedpoint::Number(3 * scale / 4)),
        fixedpoint::Number(-scale),
    );
}

#[test]
fn from_ratio_lifts_a_quotient_and_guards_zero() {
    assert_eq!(fixedpoint::from_ratio(7, 2), fixedpoint::Number(fixedpoint::from_integer(7).0 / 2));
    assert_eq!(fixedpoint::from_ratio(7, 0), fixedpoint::Number(0));
}

#[test]
fn apply_scales_by_a_dimensionless_ratio() {
    let half = fixedpoint::Ratio(fixedpoint::SCALE / 2);
    assert_eq!(fixedpoint::apply(fixedpoint::from_integer(10), half), fixedpoint::from_integer(5));
}

#[test]
fn square_root_is_exact_on_squares_and_floors_otherwise() {
    assert_eq!(fixedpoint::square_root(fixedpoint::from_integer(4)), fixedpoint::from_integer(2));
    let root_two = fixedpoint::square_root(fixedpoint::from_integer(2));
    let squared = fixedpoint::multiply(root_two, root_two);
    assert!((squared.0 - fixedpoint::from_integer(2).0).abs() <= 16, "got {squared:?}");
}

#[test]
fn square_root_scaled_roots_a_plain_integer() {
    let root = fixedpoint::square_root_scaled(250);
    let squared = fixedpoint::multiply(root, root);
    assert!((fixedpoint::whole(squared) - 250).abs() <= 1, "got {squared:?}");
}

#[test]
fn integer_root_floors_a_128_bit_radicand() {
    assert_eq!(fixedpoint::integer_root(144), 12);
    assert_eq!(fixedpoint::integer_root(143), 11);
    let big = 3u128 << 40;
    assert_eq!(fixedpoint::integer_root(big * big), 3 << 40);
}

#[test]
fn format_renders_decimals_with_half_away_rounding() {
    let half = fixedpoint::Number(fixedpoint::from_integer(1).0 / 2);
    let three_halves = fixedpoint::Number(fixedpoint::from_integer(3).0 / 2);
    assert_eq!(fixedpoint::format(half, 2), "0.50");
    assert_eq!(fixedpoint::format(three_halves, 2), "1.50");
    assert_eq!(fixedpoint::format(fixedpoint::Number(-three_halves.0), 2), "-1.50");
    let almost_one =
        fixedpoint::Number(fixedpoint::from_integer(1).0 - fixedpoint::from_integer(1).0 / 1000);
    assert_eq!(fixedpoint::format(almost_one, 2), "1.00");
}

#[test]
fn decimal_serialization_round_trips_and_drops_trailing_zeros() {
    let two_and_half = fixedpoint::Number(fixedpoint::from_integer(5).0 / 2);
    assert_eq!(fixedpoint::to_decimal(two_and_half), "2.5");
    assert_eq!(fixedpoint::to_decimal(fixedpoint::from_integer(3)), "3");
    assert_eq!(fixedpoint::from_decimal("2.5"), Some(two_and_half));
    assert_eq!(fixedpoint::from_decimal("-3"), Some(fixedpoint::from_integer(-3)));
    assert_eq!(fixedpoint::from_decimal("nope"), None);
}
