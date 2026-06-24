//! A dependency-injected time source modeled on TigerBeetle's `vsr.Time`. One
//! [`Clock`] is injected; calling code reads it with [`now_monotonic`] /
//! [`now_realtime`] and advances it with [`tick`] / [`sleep`] and never knows
//! whether it holds the deterministic virtual backend (a simulation) or the host
//! operating system (production). The Go original is a vtable of closures sharing
//! a mutated tick counter; the dialect forbids that shared mutation, so the
//! backend is an enum and advancing is consume-and-return — `tick` takes a
//! `Clock` and returns the advanced one — rather than a closure mutating in place.
//!
//! ```
//! use shared_rs::time;
//! let clock = time::new_virtual(time::SECOND, time::Moment(0), time::Skew::Uniform);
//! assert_eq!(time::now_monotonic(&clock), time::Moment(0));
//! let clock = time::tick(clock);
//! assert_eq!(time::now_monotonic(&clock), time::Moment(time::SECOND.0));
//! ```

use crate::fixedpoint;
use std::thread;
use std::time as wallclock;

/// A clock reading in nanoseconds since an arbitrary, clock-specific epoch. Only
/// the difference between two `Moment`s from the SAME clock is meaningful; a
/// monotonic and a realtime `Moment` are not comparable.
#[derive(Copy, Clone, PartialEq, Eq, Hash, Debug, PartialOrd, Ord)]
pub struct Moment(pub i64);

/// A span of nanoseconds.
#[derive(Copy, Clone, PartialEq, Eq, Hash, Debug, PartialOrd, Ord)]
pub struct Duration(pub i64);

/// The unit a [`Duration`] counts in.
pub const NANOSECOND: Duration = Duration(1);
/// A thousand nanoseconds.
pub const MICROSECOND: Duration = Duration(NANOSECOND.0 * 1000);
/// A thousand microseconds.
pub const MILLISECOND: Duration = Duration(MICROSECOND.0 * 1000);
/// A thousand milliseconds.
pub const SECOND: Duration = Duration(MILLISECOND.0 * 1000);
/// Sixty seconds.
pub const MINUTE: Duration = Duration(SECOND.0 * 60);
/// Sixty minutes.
pub const HOUR: Duration = Duration(MINUTE.0 * 60);
/// Twenty-four hours.
pub const DAY: Duration = Duration(HOUR.0 * 24);
/// Seven days.
pub const WEEK: Duration = Duration(DAY.0 * 7);

/// The `Moment` a span past another.
pub fn add(moment: Moment, span: Duration) -> Moment {
    Moment(moment.0 + span.0)
}

/// The span from `earlier` to `later`. Meaningful only when both come from the
/// same clock.
pub fn since(later: Moment, earlier: Moment) -> Duration {
    Duration(later.0 - earlier.0)
}

/// The injected time source. Both backends share one read/advance API, so calling
/// code is written against `Clock` and never branches on which it holds — the Go
/// `Clock` vtable, expressed as an enum the dialect can build without the mutable
/// closures the original used.
#[derive(Copy, Clone, Debug)]
pub enum Clock {
    /// The deterministic, system-free backend: time advances only on `tick`/`sleep`.
    Virtual(Virtual_Clock),
    /// The operating-system backend, holding the monotonic reference point captured
    /// at creation (Rust's `Instant` exposes only elapsed time, not an absolute).
    Operating_System(wallclock::Instant),
}

/// A fresh virtual clock at tick zero, wrapped as a [`Clock`]. Pass `Skew::Uniform`
/// for a perfect clock.
pub fn new_virtual(resolution: Duration, epoch: Moment, skew: Skew) -> Clock {
    Clock::Virtual(Virtual_Clock { resolution, epoch, skew, ticks: 0 })
}

/// A clock backed by the host operating system, wrapped as a [`Clock`].
pub fn new_operating_system() -> Clock {
    Clock::Operating_System(wallclock::Instant::now())
}

/// The monotonic reading — never regresses, so use it for elapsed time, timeouts,
/// and latency. The virtual backend reads tick count times resolution; the
/// operating-system backend reads nanoseconds elapsed since the reference point.
pub fn now_monotonic(clock: &Clock) -> Moment {
    match clock {
        Clock::Virtual(virtual_clock) => {
            Moment(virtual_clock.ticks * virtual_clock.resolution.0)
        }
        Clock::Operating_System(epoch) => Moment(epoch.elapsed().as_nanos() as i64),
    }
}

/// The wall-clock reading — it can jump, so use it only for calendar timestamps,
/// never for elapsed time. The virtual backend is epoch plus elapsed less any
/// modeled skew; the operating-system backend reads nanoseconds since the Unix
/// epoch.
pub fn now_realtime(clock: &Clock) -> Moment {
    match clock {
        Clock::Virtual(virtual_clock) => virtual_realtime(virtual_clock),
        Clock::Operating_System(_) => operating_system_realtime(),
    }
}

/// Advances the clock by one resolution, returning the advanced clock. Ticking the
/// operating-system backend is a no-op — it advances on its own.
pub fn tick(clock: Clock) -> Clock {
    match clock {
        Clock::Virtual(virtual_clock) => {
            Clock::Virtual(Virtual_Clock { ticks: virtual_clock.ticks + 1, ..virtual_clock })
        }
        Clock::Operating_System(epoch) => Clock::Operating_System(epoch),
    }
}

/// Waits out the span and returns the clock. The virtual backend advances its own
/// time instead of waiting (a simulation never blocks); the operating-system
/// backend blocks the thread for the real duration.
pub fn sleep(clock: Clock, duration: Duration) -> Clock {
    match clock {
        Clock::Virtual(virtual_clock) => Clock::Virtual(Virtual_Clock {
            ticks: virtual_clock.ticks + duration.0 / virtual_clock.resolution.0,
            ..virtual_clock
        }),
        Clock::Operating_System(epoch) => {
            operating_system_sleep(duration);
            Clock::Operating_System(epoch)
        }
    }
}

/// The deterministic backend of a [`Clock`]: time advances only when [`tick`] or
/// [`sleep`] is called, so a simulation reaches a future `Moment` by ticking
/// rather than by waiting on a wall clock.
#[derive(Copy, Clone, Debug)]
pub struct Virtual_Clock {
    /// How far the monotonic clock advances on each tick — the grain of a
    /// simulated oscillator.
    pub resolution: Duration,
    /// The wall-clock origin: realtime at tick zero, before any skew.
    pub epoch: Moment,
    /// How realtime bends away from true elapsed time; `Skew::Uniform` is a perfect
    /// clock.
    pub skew: Skew,
    /// The current tick count; reads are a pure function of it.
    pub ticks: i64,
}

fn virtual_realtime(clock: &Virtual_Clock) -> Moment {
    // `Skew::Uniform` contributes zero, so realtime is just epoch plus elapsed.
    let now = Moment(clock.epoch.0 + clock.ticks * clock.resolution.0);
    Moment(now.0 - skew_at(&clock.skew, clock.ticks).0)
}

fn operating_system_realtime() -> Moment {
    let since_epoch = wallclock::SystemTime::now()
        .duration_since(wallclock::SystemTime::UNIX_EPOCH)
        .unwrap_or(wallclock::Duration::ZERO);
    Moment(since_epoch.as_nanos() as i64)
}

fn operating_system_sleep(duration: Duration) {
    // A non-positive span is not a wait; only a real one blocks the thread.
    if duration.0 > 0 {
        thread::sleep(wallclock::Duration::from_nanos(duration.0 as u64));
    }
}

/// How a simulated wall clock deviates from true elapsed time. Each model carries
/// its own coefficients, so an invalid pairing cannot be expressed.
#[derive(Copy, Clone, Debug)]
pub enum Skew {
    /// A perfect clock: realtime tracks elapsed time with no deviation.
    Uniform,
    /// Constant drift: `drift_per_tick` nanoseconds each tick, plus a fixed
    /// `initial_offset`.
    Linear { drift_per_tick: Duration, initial_offset: Duration },
    /// A sinusoidal wobble of `amplitude` over a `period` measured in ticks.
    Periodic { amplitude: Duration, period: i64 },
    /// A discontinuous `jump` after `onset_tick` ticks — an NTP correction or
    /// operator clock change.
    Step { jump: Duration, onset_tick: i64 },
}

/// The skew this model produces at the given tick count.
pub fn skew_at(skew: &Skew, ticks: i64) -> Duration {
    match skew {
        Skew::Uniform => Duration(0),
        Skew::Linear { drift_per_tick, initial_offset } => {
            Duration(ticks * drift_per_tick.0 + initial_offset.0)
        }
        Skew::Step { jump, onset_tick } => match ticks > *onset_tick {
            true => *jump,
            false => Duration(0),
        },
        Skew::Periodic { amplitude, period } => periodic_skew(*amplitude, *period, ticks),
    }
}

fn periodic_skew(amplitude: Duration, period: i64, ticks: i64) -> Duration {
    // A zero period is a degenerate sinusoid; report no skew rather than divide by
    // zero. The phase is reduced to one period before the fixed-point lift, so a
    // long tick count cannot overflow the scaled numerator.
    match period == 0 {
        true => Duration(0),
        false => {
            let phase = ticks % period;
            let turns = fixedpoint::divide(
                fixedpoint::from_integer(phase),
                fixedpoint::from_integer(period),
            );
            let wobble = fixedpoint::multiply(
                fixedpoint::from_integer(amplitude.0),
                fixedpoint::sine_turns(turns),
            );
            Duration(fixedpoint::whole(wobble))
        }
    }
}
