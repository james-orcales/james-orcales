//! Black-box: the dependency-injected time source. One `Clock` is read with
//! `now_monotonic`/`now_realtime` and advanced with `tick`/`sleep`, whether it is
//! the deterministic virtual backend or the operating-system one — the calling
//! code never branches on which. Skew (uniform/linear/step/periodic) bends
//! realtime away from monotonic; the operating-system backend is checked loosely.

use shared_rs::time;

#[test]
fn duration_constants_scale_in_nanoseconds() {
    assert_eq!(time::SECOND.0, 1_000_000_000);
    assert_eq!(time::WEEK.0, 7 * 24 * 60 * 60 * 1_000_000_000);
}

#[test]
fn since_and_add_are_inverse() {
    let earlier = time::Moment(100);
    let later = time::add(earlier, time::Duration(50));
    assert_eq!(later, time::Moment(150));
    assert_eq!(time::since(later, earlier), time::Duration(50));
}

#[test]
fn virtual_monotonic_advances_only_on_tick() {
    let clock = time::new_virtual(time::SECOND, time::Moment(0), time::Skew::Uniform);
    assert_eq!(time::now_monotonic(&clock), time::Moment(0));
    let clock = time::tick(time::tick(clock));
    assert_eq!(time::now_monotonic(&clock), time::Moment(2 * time::SECOND.0));
}

#[test]
fn sleep_advances_virtual_time() {
    let clock = time::new_virtual(time::SECOND, time::Moment(0), time::Skew::Uniform);
    let clock = time::sleep(clock, time::Duration(3 * time::SECOND.0));
    assert_eq!(time::now_monotonic(&clock), time::Moment(3 * time::SECOND.0));
}

#[test]
fn realtime_is_epoch_plus_elapsed_without_skew() {
    let clock = time::tick(time::new_virtual(time::SECOND, time::Moment(1_000), time::Skew::Uniform));
    assert_eq!(time::now_realtime(&clock), time::Moment(1_000 + time::SECOND.0));
}

#[test]
fn uniform_skew_is_no_deviation() {
    assert_eq!(time::skew_at(&time::Skew::Uniform, 7), time::Duration(0));
}

#[test]
fn linear_skew_is_drift_per_tick_plus_offset() {
    let skew = time::Skew::Linear { drift_per_tick: time::Duration(2), initial_offset: time::Duration(5) };
    assert_eq!(time::skew_at(&skew, 10), time::Duration(25));
}

#[test]
fn step_skew_jumps_after_its_onset_tick() {
    let skew = time::Skew::Step { jump: time::Duration(100), onset_tick: 4 };
    assert_eq!(time::skew_at(&skew, 3), time::Duration(0));
    assert_eq!(time::skew_at(&skew, 5), time::Duration(100));
}

#[test]
fn periodic_skew_peaks_at_a_quarter_period() {
    // A quarter into the period the sine is exactly 1, so the skew is exactly the amplitude.
    let skew = time::Skew::Periodic { amplitude: time::Duration(1_000), period: 4 };
    assert_eq!(time::skew_at(&skew, 1), time::Duration(1_000));
    // A zero period is degenerate and yields no skew rather than dividing by zero.
    let degenerate = time::Skew::Periodic { amplitude: time::Duration(1_000), period: 0 };
    assert_eq!(time::skew_at(&degenerate, 1), time::Duration(0));
}

#[test]
fn skew_subtracts_from_realtime_but_not_monotonic() {
    let skew = time::Skew::Periodic { amplitude: time::Duration(1_000), period: 4 };
    let clock = time::tick(time::new_virtual(time::SECOND, time::Moment(0), skew));
    assert_eq!(time::now_monotonic(&clock), time::Moment(time::SECOND.0));
    assert_eq!(time::now_realtime(&clock), time::Moment(time::SECOND.0 - 1_000));
}

#[test]
fn one_api_reads_both_backends() {
    // The same read functions serve a virtual clock and an operating-system clock;
    // the caller does not branch on which `Clock` it holds.
    let virtual_clock = time::new_virtual(time::SECOND, time::Moment(0), time::Skew::Uniform);
    let operating_system_clock = time::new_operating_system();
    let _virtual_reading = time::now_monotonic(&virtual_clock);
    let _operating_system_reading = time::now_monotonic(&operating_system_clock);
}

#[test]
fn operating_system_realtime_is_after_the_year_2020() {
    let clock = time::new_operating_system();
    let year_2020_nanoseconds = 1_577_836_800_000_000_000;
    assert!(time::now_realtime(&clock).0 > year_2020_nanoseconds);
}

#[test]
fn operating_system_monotonic_never_regresses_below_its_epoch() {
    let clock = time::new_operating_system();
    assert!(time::now_monotonic(&clock).0 >= 0);
    // Ticking an operating-system clock is a no-op; it still reads.
    let clock = time::tick(clock);
    assert!(time::now_monotonic(&clock).0 >= 0);
}
