//! The DIV/TIMA timer. rboy advanced the counters in place through `&mut self`;
//! here reads are pure and each write or tick returns a new `Timer`. `do_cycle`
//! returns the timer with its interrupt byte (0x04) possibly raised; the MMU folds
//! that into IF and clears it. rboy's `while` accumulators become divide/modulo and
//! a bounded `fold`, since the dialect has no mutating loop.

/// The timer registers plus the sub-register tick accumulators.
#[derive(Copy, Clone, Debug)]
pub struct Timer {
    pub divider: u8,
    pub counter: u8,
    pub modulo: u8,
    pub enabled: bool,
    pub step: u32,
    pub internalcnt: u32,
    pub internaldiv: u32,
    pub interrupt: u8,
}

pub fn new() -> Timer {
    Timer {
        divider: 0,
        counter: 0,
        modulo: 0,
        enabled: false,
        step: 1024,
        internalcnt: 0,
        internaldiv: 0,
        interrupt: 0,
    }
}

pub fn read_byte(timer: &Timer, address: u16) -> u8 {
    match address {
        0xFF04 => timer.divider,
        0xFF05 => timer.counter,
        0xFF06 => timer.modulo,
        // TAC: the unused high bits read as 1, plus the enable bit and the step code.
        0xFF07 => 0xF8 | (if timer.enabled { 0x4 } else { 0 }) | step_code(timer.step),
        _ => panic!("Timer does not handle read {:04X}", address),
    }
}

// The two-bit TAC clock-select code for a given period.
fn step_code(step: u32) -> u8 {
    match step {
        16 => 1,
        64 => 2,
        256 => 3,
        _ => 0,
    }
}

pub fn write_byte(timer: Timer, address: u16, value: u8) -> Timer {
    match address {
        // Any write to DIV resets it to zero.
        0xFF04 => Timer { divider: 0, ..timer },
        0xFF05 => Timer { counter: value, ..timer },
        0xFF06 => Timer { modulo: value, ..timer },
        0xFF07 => Timer { enabled: value & 0x4 != 0, step: step_period(value), ..timer },
        _ => panic!("Timer does not handle write {:04X}", address),
    }
}

// The TAC clock period selected by the low two bits.
fn step_period(value: u8) -> u32 {
    match value & 0x3 {
        1 => 16,
        2 => 64,
        3 => 256,
        _ => 1024,
    }
}

/// Advances the timer by `ticks` T-cycles: DIV rolls every 256, and when enabled
/// TIMA rolls every `step`, reloading from `modulo` and raising 0x04 on overflow.
pub fn do_cycle(timer: Timer, ticks: u32) -> Timer {
    let divsum = timer.internaldiv + ticks;
    // A call's tick count is small, so divsum/256 is 0 or 1 and the cast never truncates.
    let stepped = Timer {
        divider: timer.divider.wrapping_add((divsum / 256) as u8),
        internaldiv: divsum % 256,
        ..timer
    };
    match timer.enabled {
        false => stepped,
        true => tick_counter(stepped, ticks),
    }
}

// Advances TIMA by the whole number of `step`-periods `ticks` covers, folding each
// increment so an overflow reloads `modulo` and remembers to raise the interrupt.
fn tick_counter(timer: Timer, ticks: u32) -> Timer {
    let total = timer.internalcnt + ticks;
    let increments = total / timer.step;
    let (counter, overflowed) = (0..increments).fold((timer.counter, false), |(value, irq), _| {
        match value.wrapping_add(1) {
            0 => (timer.modulo, true),
            next => (next, irq),
        }
    });
    Timer {
        counter,
        internalcnt: total % timer.step,
        interrupt: timer.interrupt | if overflowed { 0x04 } else { 0 },
        ..timer
    }
}

#[cfg(test)]
mod tests {
    use crate::timer;

    #[test]
    fn divider_rolls_every_256_ticks() {
        let clock = timer::do_cycle(timer::new(), 256);
        assert_eq!(clock.divider, 1);
        // 255 more is not yet another full period.
        let clock = timer::do_cycle(clock, 255);
        assert_eq!(clock.divider, 1);
        let clock = timer::do_cycle(clock, 1);
        assert_eq!(clock.divider, 2);
    }

    #[test]
    fn counter_overflow_reloads_modulo_and_raises_interrupt() {
        // Enable at the fastest step (16), preload TIMA at 0xFF, modulo 0xAB.
        let clock = timer::write_byte(timer::new(), 0xFF06, 0xAB);
        let clock = timer::write_byte(clock, 0xFF05, 0xFF);
        let clock = timer::write_byte(clock, 0xFF07, 0x05);
        // One 16-tick period increments 0xFF to 0x00, so it reloads and interrupts.
        let clock = timer::do_cycle(clock, 16);
        assert_eq!(clock.counter, 0xAB);
        assert_eq!(clock.interrupt & 0x04, 0x04);
    }

    #[test]
    fn tac_read_reports_enable_and_step() {
        // Enable with step code 2 (period 64): 0xF8 | 0x04 | 0x02 = 0xFE.
        let clock = timer::write_byte(timer::new(), 0xFF07, 0x06);
        assert_eq!(timer::read_byte(&clock, 0xFF07), 0xFE);
    }
}
