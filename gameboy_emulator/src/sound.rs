//! The full APU, ported from rboy's `sound.rs`: four channels and a 512 Hz frame
//! sequencer feeding a `blip_buf` band-limited synthesizer, plus the stereo mix.
//! rboy drove it through `&mut self` and advanced it lazily on register access; here
//! everything threads by value and is advanced **eagerly** in `do_cycle`, using
//! rboy's exact frame-chunk boundaries and absolute blip clock times. That is
//! byte-identical because execute-then-tick makes register writes take effect at the
//! instruction boundary in both models, `blip_buf` deltas are position-addressed and
//! additive (so run-ordering is irrelevant), and audio materializes only at
//! `do_output` — where both models have emitted every delta up to `time`. So reads
//! stay pure. The synthesized f32 samples accumulate in `audio_left`/`audio_right`
//! instead of a device sink.

use crate::blip_buf;
use crate::gbmode;
use crate::memory;

// The four square-wave duty patterns (rboy's WAVE_PATTERN).
const WAVE_PATTERN: [[i32; 8]; 4] = [
    [-1, -1, -1, -1, 1, -1, -1, -1],
    [-1, -1, -1, -1, 1, 1, -1, -1],
    [-1, -1, 1, 1, 1, 1, -1, -1],
    [1, 1, 1, 1, -1, -1, 1, 1],
];
const CLOCKS_PER_SECOND: u32 = 1 << 22;
const CLOCKS_PER_FRAME: u32 = CLOCKS_PER_SECOND / 512;
const OUTPUT_SAMPLE_COUNT: usize = 2000;
const SAMPLE_RATE: u32 = 44100;
const SWEEP_DELAY_ZERO_PERIOD: u8 = 8;
const WAVE_INITIAL_DELAY: u32 = 4;

/// The volume envelope shared by the two square channels and the noise channel.
#[derive(Copy, Clone, Debug)]
pub struct Volume_Envelope {
    pub period: u8,
    pub goes_up: bool,
    pub delay: u8,
    pub initial_volume: u8,
    pub volume: u8,
}

fn envelope_new() -> Volume_Envelope {
    Volume_Envelope { period: 0, goes_up: false, delay: 0, initial_volume: 0, volume: 0 }
}

fn envelope_read(env: &Volume_Envelope) -> u8 {
    ((env.initial_volume & 0xF) << 4) | (if env.goes_up { 0x08 } else { 0 }) | (env.period & 0x7)
}

fn envelope_write(env: Volume_Envelope, address: u16, value: u8) -> Volume_Envelope {
    match address {
        0xFF12 | 0xFF17 | 0xFF21 => Volume_Envelope {
            period: value & 0x7,
            goes_up: value & 0x8 == 0x8,
            initial_volume: value >> 4,
            volume: value >> 4,
            ..env
        },
        0xFF14 | 0xFF19 | 0xFF23 if value & 0x80 == 0x80 => {
            Volume_Envelope { delay: env.period, volume: env.initial_volume, ..env }
        }
        _ => env,
    }
}

fn envelope_step(env: Volume_Envelope) -> Volume_Envelope {
    match env.delay {
        d if d > 1 => Volume_Envelope { delay: d - 1, ..env },
        1 => {
            let volume = match (env.goes_up, env.volume) {
                (true, v) if v < 15 => v + 1,
                (false, v) if v > 0 => v - 1,
                (_, v) => v,
            };
            Volume_Envelope { delay: env.period, volume, ..env }
        }
        _ => env,
    }
}

/// A channel's length counter, which silences the channel when it reaches zero.
#[derive(Copy, Clone, Debug)]
pub struct Length_Counter {
    pub enabled: bool,
    pub value: u16,
    pub max: u16,
}

fn length_new(max: u16) -> Length_Counter {
    Length_Counter { enabled: false, value: 0, max }
}

fn length_is_active(length: &Length_Counter) -> bool {
    length.value > 0
}

fn length_extra_step(frame_step: u8) -> bool {
    frame_step % 2 == 1
}

fn length_step(length: Length_Counter) -> Length_Counter {
    match length.enabled && length.value > 0 {
        true => Length_Counter { value: length.value - 1, ..length },
        false => length,
    }
}

fn length_enable(length: Length_Counter, enable: bool, frame_step: u8) -> Length_Counter {
    let was_enabled = length.enabled;
    let enabled = Length_Counter { enabled: enable, ..length };
    match !was_enabled && length_extra_step(frame_step) {
        true => length_step(enabled),
        false => enabled,
    }
}

fn length_set(length: Length_Counter, minus_value: u8) -> Length_Counter {
    Length_Counter { value: length.max - minus_value as u16, ..length }
}

fn length_trigger(length: Length_Counter, frame_step: u8) -> Length_Counter {
    match length.value == 0 {
        true => {
            let full = Length_Counter { value: length.max, ..length };
            match length_extra_step(frame_step) {
                true => length_step(full),
                false => full,
            }
        }
        false => length,
    }
}

/// A square-wave channel (channel 1 has the frequency sweep, channel 2 does not).
#[derive(Clone, Debug)]
pub struct Square_Channel {
    pub active: bool,
    pub dac_enabled: bool,
    pub duty: u8,
    pub phase: u8,
    pub length: Length_Counter,
    pub frequency: u16,
    pub period: u32,
    pub last_amp: i32,
    pub delay: u32,
    pub has_sweep: bool,
    pub sweep_enabled: bool,
    pub sweep_frequency: u16,
    pub sweep_delay: u8,
    pub sweep_period: u8,
    pub sweep_shift: u8,
    pub sweep_negate: bool,
    pub sweep_did_negate: bool,
    pub volume_envelope: Volume_Envelope,
    pub blip: blip_buf::Blip_Buf,
}

fn square_new(with_sweep: bool) -> Square_Channel {
    Square_Channel {
        active: false,
        dac_enabled: false,
        duty: 1,
        phase: 1,
        length: length_new(64),
        frequency: 0,
        period: 2048,
        last_amp: 0,
        delay: 0,
        has_sweep: with_sweep,
        sweep_enabled: false,
        sweep_frequency: 0,
        sweep_delay: 0,
        sweep_period: 0,
        sweep_shift: 0,
        sweep_negate: false,
        sweep_did_negate: false,
        volume_envelope: envelope_new(),
        blip: create_blip(),
    }
}

fn square_read(channel: &Square_Channel, address: u16) -> u8 {
    match address {
        0xFF10 => {
            0x80 | ((channel.sweep_period & 0x7) << 4)
                | (if channel.sweep_negate { 0x8 } else { 0 })
                | (channel.sweep_shift & 0x7)
        }
        0xFF11 | 0xFF16 => ((channel.duty & 3) << 6) | 0x3F,
        0xFF12 | 0xFF17 => envelope_read(&channel.volume_envelope),
        0xFF13 | 0xFF18 => 0xFF,
        0xFF14 | 0xFF19 => 0x80 | (if channel.length.enabled { 0x40 } else { 0 }) | 0x3F,
        _ => 0xFF,
    }
}

fn square_write(channel: Square_Channel, address: u16, value: u8, frame_step: u8) -> Square_Channel {
    let updated = match address {
        0xFF10 => square_write_sweep(channel, value),
        0xFF11 | 0xFF16 => Square_Channel { duty: value >> 6, length: length_set(channel.length, value & 0x3F), ..channel },
        0xFF12 | 0xFF17 => {
            let dac = value & 0xF8 != 0;
            Square_Channel { dac_enabled: dac, active: channel.active && dac, ..channel }
        }
        0xFF13 | 0xFF18 => square_set_period(Square_Channel { frequency: (channel.frequency & 0x0700) | (value as u16), ..channel }),
        0xFF14 | 0xFF19 => square_write_high(channel, value, frame_step),
        _ => channel,
    };
    Square_Channel { volume_envelope: envelope_write(updated.volume_envelope, address, value), ..updated }
}

// Recomputes the square period from the frequency: `(2048-freq)*4`, or 0 above range.
fn square_set_period(channel: Square_Channel) -> Square_Channel {
    let period = match channel.frequency > 2047 {
        true => 0,
        false => (2048 - channel.frequency as u32) * 4,
    };
    Square_Channel { period, ..channel }
}

// NR10 sweep write: a negate-to-positive edge after a negate calc silences the channel.
fn square_write_sweep(channel: Square_Channel, value: u8) -> Square_Channel {
    let negate = value & 0x8 == 0x8;
    let silence = channel.sweep_negate && !negate && channel.sweep_did_negate;
    Square_Channel {
        sweep_period: (value >> 4) & 0x7,
        sweep_shift: value & 0x7,
        sweep_negate: negate,
        sweep_did_negate: false,
        active: channel.active && !silence,
        ..channel
    }
}

// NR14/NR24: frequency high bits, length enable, and (on trigger) length reload,
// DAC-gated active, and sweep arm.
fn square_write_high(channel: Square_Channel, value: u8, frame_step: u8) -> Square_Channel {
    let frequency = (channel.frequency & 0x00FF) | (((value & 0x07) as u16) << 8);
    let length = length_enable(channel.length, value & 0x40 == 0x40, frame_step);
    let gated = square_set_period(Square_Channel {
        frequency,
        length,
        active: channel.active && length_is_active(&length),
        ..channel
    });
    match value & 0x80 == 0x80 {
        false => gated,
        true => square_trigger(gated, frame_step),
    }
}

fn square_trigger(channel: Square_Channel, frame_step: u8) -> Square_Channel {
    let triggered = Square_Channel {
        active: channel.active || channel.dac_enabled,
        length: length_trigger(channel.length, frame_step),
        ..channel
    };
    match triggered.has_sweep {
        false => triggered,
        true => square_start_sweep(triggered),
    }
}

fn square_start_sweep(channel: Square_Channel) -> Square_Channel {
    let armed = Square_Channel {
        sweep_frequency: channel.frequency,
        sweep_delay: match channel.sweep_period != 0 {
            true => channel.sweep_period,
            false => SWEEP_DELAY_ZERO_PERIOD,
        },
        sweep_enabled: channel.sweep_period > 0 || channel.sweep_shift > 0,
        ..channel
    };
    match armed.sweep_shift > 0 {
        true => sweep_calculate(armed).0,
        false => armed,
    }
}

// One sweep frequency calculation: clears active on overflow, records the negate.
fn sweep_calculate(channel: Square_Channel) -> (Square_Channel, u16) {
    let offset = channel.sweep_frequency >> channel.sweep_shift;
    let newfreq = match channel.sweep_negate {
        true => channel.sweep_frequency.wrapping_sub(offset),
        false => channel.sweep_frequency.wrapping_add(offset),
    };
    let recorded = Square_Channel { sweep_did_negate: channel.sweep_did_negate || channel.sweep_negate, ..channel };
    (Square_Channel { active: recorded.active && newfreq <= 2047, ..recorded }, newfreq)
}

fn square_step_length(channel: Square_Channel) -> Square_Channel {
    let length = length_step(channel.length);
    Square_Channel { length, active: channel.active && length_is_active(&length), ..channel }
}

fn square_step_sweep(channel: Square_Channel) -> Square_Channel {
    match channel.sweep_delay > 1 {
        true => Square_Channel { sweep_delay: channel.sweep_delay - 1, ..channel },
        false => match channel.sweep_period == 0 {
            true => Square_Channel { sweep_delay: SWEEP_DELAY_ZERO_PERIOD, ..channel },
            false => square_apply_sweep(Square_Channel { sweep_delay: channel.sweep_period, ..channel }),
        },
    }
}

fn square_apply_sweep(channel: Square_Channel) -> Square_Channel {
    match channel.sweep_enabled {
        false => channel,
        true => {
            let (checked, newfreq) = sweep_calculate(channel);
            match newfreq <= 2047 && checked.sweep_shift != 0 {
                true => sweep_calculate(square_set_period(Square_Channel {
                    sweep_frequency: newfreq,
                    frequency: newfreq,
                    ..checked
                }))
                .0,
                false => checked,
            }
        }
    }
}

// Synthesizes the square waveform over [start, end): emits a blip delta whenever the
// duty-modulated amplitude changes, advancing the phase (rboy's SquareChannel::run).
fn square_run(channel: Square_Channel, start: u32, end: u32) -> Square_Channel {
    match !channel.active || channel.period == 0 {
        true => square_silence(channel, start),
        false => {
            let volume = channel.volume_envelope.volume as i32;
            let from = start + channel.delay;
            square_walk(channel, from, end, volume)
        }
    }
}

fn square_silence(channel: Square_Channel, start: u32) -> Square_Channel {
    match channel.last_amp != 0 {
        true => Square_Channel { blip: blip_buf::add_delta(channel.blip, start, -channel.last_amp), last_amp: 0, delay: 0, ..channel },
        false => channel,
    }
}

fn square_walk(channel: Square_Channel, time: u32, end: u32, volume: i32) -> Square_Channel {
    match time < end {
        false => Square_Channel { delay: time - end, ..channel },
        true => {
            let amp = volume * WAVE_PATTERN[channel.duty as usize][channel.phase as usize];
            let next_time = time + channel.period;
            let next_phase = (channel.phase + 1) % 8;
            let (blip, last_amp) = emit(channel.blip, channel.last_amp, time, amp);
            square_walk(Square_Channel { blip, last_amp, phase: next_phase, ..channel }, next_time, end, volume)
        }
    }
}

// Emits a blip delta when the amplitude changed, returning the new blip and last_amp.
fn emit(blip: blip_buf::Blip_Buf, last_amp: i32, time: u32, amp: i32) -> (blip_buf::Blip_Buf, i32) {
    match amp != last_amp {
        true => (blip_buf::add_delta(blip, time, amp - last_amp), amp),
        false => (blip, last_amp),
    }
}

/// The programmable-wave channel (channel 3), owning its 16-byte wave RAM.
#[derive(Clone, Debug)]
pub struct Wave_Channel {
    pub active: bool,
    pub dac_enabled: bool,
    pub length: Length_Counter,
    pub frequency: u16,
    pub period: u32,
    pub last_amp: i32,
    pub delay: u32,
    pub volume_shift: u8,
    pub waveram: Vec<u8>,
    pub current_wave: u8,
    pub sample_recently_accessed: bool,
    pub dmg_mode: bool,
    pub blip: blip_buf::Blip_Buf,
}

fn wave_new(dmg_mode: bool) -> Wave_Channel {
    Wave_Channel {
        active: false,
        dac_enabled: false,
        length: length_new(256),
        frequency: 0,
        period: 2048,
        last_amp: 0,
        delay: 0,
        volume_shift: 0,
        waveram: vec![0; 16],
        current_wave: 0,
        sample_recently_accessed: false,
        dmg_mode,
        blip: create_blip(),
    }
}

fn wave_read(channel: &Wave_Channel, address: u16) -> u8 {
    match address {
        0xFF1A => (if channel.dac_enabled { 0x80 } else { 0 }) | 0x7F,
        0xFF1B => 0xFF,
        0xFF1C => 0x80 | ((channel.volume_shift & 0b11) << 5) | 0x1F,
        0xFF1D => 0xFF,
        0xFF1E => 0x80 | (if channel.length.enabled { 0x40 } else { 0 }) | 0x3F,
        0xFF30..=0xFF3F => wave_read_ram(channel, address),
        _ => 0xFF,
    }
}

// While inactive the wave RAM is directly addressable; while active only the byte at
// the current playback position is (and in DMG only within the access window).
fn wave_read_ram(channel: &Wave_Channel, address: u16) -> u8 {
    match channel.active {
        false => channel.waveram[address as usize - 0xFF30],
        true => match !channel.dmg_mode || channel.sample_recently_accessed {
            true => channel.waveram[(channel.current_wave >> 1) as usize],
            false => 0xFF,
        },
    }
}

fn wave_write(channel: Wave_Channel, address: u16, value: u8, frame_step: u8) -> Wave_Channel {
    match address {
        0xFF1A => {
            let dac = value & 0x80 == 0x80;
            Wave_Channel { dac_enabled: dac, active: channel.active && dac, ..channel }
        }
        0xFF1B => Wave_Channel { length: length_set(channel.length, value), ..channel },
        0xFF1C => Wave_Channel { volume_shift: (value >> 5) & 0b11, ..channel },
        0xFF1D => wave_set_period(Wave_Channel { frequency: (channel.frequency & 0x0700) | (value as u16), ..channel }),
        0xFF1E => wave_write_high(channel, value, frame_step),
        0xFF30..=0xFF3F => wave_write_ram(channel, address, value),
        _ => channel,
    }
}

// Recomputes the wave period: `(2048-freq)*2`, or 0 above range (note: > 2048).
fn wave_set_period(channel: Wave_Channel) -> Wave_Channel {
    let period = match channel.frequency > 2048 {
        true => 0,
        false => (2048 - channel.frequency as u32) * 2,
    };
    Wave_Channel { period, ..channel }
}

fn wave_write_high(channel: Wave_Channel, value: u8, frame_step: u8) -> Wave_Channel {
    let frequency = (channel.frequency & 0x00FF) | (((value & 0b111) as u16) << 8);
    let length = length_enable(channel.length, value & 0x40 == 0x40, frame_step);
    let gated = wave_set_period(Wave_Channel {
        frequency,
        length,
        active: channel.active && length_is_active(&length),
        ..channel
    });
    match value & 0x80 == 0x80 {
        false => gated,
        true => wave_trigger(gated, frame_step),
    }
}

// Wave trigger: DMG wave-RAM corruption, length reload, reset playback position with
// the initial delay, DAC-gated active.
fn wave_trigger(channel: Wave_Channel, frame_step: u8) -> Wave_Channel {
    let corrupted = dmg_maybe_corrupt_waveram(channel);
    Wave_Channel {
        length: length_trigger(corrupted.length, frame_step),
        current_wave: 0,
        delay: corrupted.period + WAVE_INITIAL_DELAY,
        active: corrupted.dac_enabled,
        ..corrupted
    }
}

// The DMG wave-RAM corruption on a trigger while the channel is mid-sample (delay 0).
fn dmg_maybe_corrupt_waveram(channel: Wave_Channel) -> Wave_Channel {
    match !channel.dmg_mode || !channel.active || channel.delay != 0 {
        true => channel,
        false => {
            let byteindex = ((channel.current_wave + 1) % 32) as usize >> 1;
            let waveram = match byteindex < 4 {
                true => memory::write(&channel.waveram, 0, channel.waveram[byteindex]),
                false => corrupt_block(&channel.waveram, byteindex & 0b1100),
            };
            Wave_Channel { waveram, ..channel }
        }
    }
}

// Copies a 4-byte source block to the front of wave RAM (the wide corruption case).
fn corrupt_block(waveram: &[u8], blockstart: usize) -> Vec<u8> {
    memory::write_slice(waveram, 0, &waveram[blockstart..blockstart + 4])
}

fn wave_write_ram(channel: Wave_Channel, address: u16, value: u8) -> Wave_Channel {
    match channel.active {
        false => Wave_Channel { waveram: memory::write(&channel.waveram, address as usize - 0xFF30, value), ..channel },
        true => match !channel.dmg_mode || channel.sample_recently_accessed {
            true => Wave_Channel { waveram: memory::write(&channel.waveram, (channel.current_wave >> 1) as usize, value), ..channel },
            false => channel,
        },
    }
}

fn wave_step_length(channel: Wave_Channel) -> Wave_Channel {
    let length = length_step(channel.length);
    Wave_Channel { length, active: channel.active && length_is_active(&length), ..channel }
}

// Synthesizes the wave output over [start, end): emits a blip delta whenever the
// sample changes, advancing the playback position and the DMG access window.
fn wave_run(channel: Wave_Channel, start: u32, end: u32) -> Wave_Channel {
    let reset = Wave_Channel { sample_recently_accessed: false, ..channel };
    match !reset.active || reset.period == 0 {
        true => match reset.last_amp != 0 {
            true => Wave_Channel { blip: blip_buf::add_delta(reset.blip, start, -reset.last_amp), last_amp: 0, delay: 0, ..reset },
            false => reset,
        },
        false => {
            let from = start + reset.delay;
            wave_walk(reset, from, end)
        }
    }
}

fn wave_walk(channel: Wave_Channel, time: u32, end: u32) -> Wave_Channel {
    match time < end {
        false => Wave_Channel { delay: time - end, ..channel },
        true => {
            let shift = wave_volshift(channel.volume_shift);
            let wavebyte = channel.waveram[(channel.current_wave >> 1) as usize];
            let sample = match channel.current_wave % 2 == 0 {
                true => wavebyte >> 4,
                false => wavebyte & 0xF,
            };
            let amp = (((sample as i32) << 2) >> shift) as i32;
            let next_time = time + channel.period;
            let next_wave = (channel.current_wave + 1) % 32;
            let recently = channel.sample_recently_accessed || time >= end.saturating_sub(2);
            let (blip, last_amp) = emit(channel.blip, channel.last_amp, time, amp);
            wave_walk(Wave_Channel { blip, last_amp, current_wave: next_wave, sample_recently_accessed: recently, ..channel }, next_time, end)
        }
    }
}

// The wave output is emitted at 4x amplitude; this maps the volume shift to the
// right-shift that mutes (0), or scales for 100%/50%/25% (rboy's volshift).
fn wave_volshift(volume_shift: u8) -> u32 {
    match volume_shift {
        0 => 6,
        1 => 0,
        2 => 1,
        _ => 2,
    }
}

/// The noise channel (channel 4).
#[derive(Clone, Debug)]
pub struct Noise_Channel {
    pub active: bool,
    pub dac_enabled: bool,
    pub reg_ff22: u8,
    pub length: Length_Counter,
    pub volume_envelope: Volume_Envelope,
    pub period: u32,
    pub shift_width: u8,
    pub state: u16,
    pub delay: u32,
    pub last_amp: i32,
    pub blip: blip_buf::Blip_Buf,
}

fn noise_new() -> Noise_Channel {
    Noise_Channel {
        active: false,
        dac_enabled: false,
        reg_ff22: 0,
        length: length_new(64),
        volume_envelope: envelope_new(),
        period: 2048,
        shift_width: 14,
        state: 1,
        delay: 0,
        last_amp: 0,
        blip: create_blip(),
    }
}

fn noise_read(channel: &Noise_Channel, address: u16) -> u8 {
    match address {
        0xFF20 => 0xFF,
        0xFF21 => envelope_read(&channel.volume_envelope),
        0xFF22 => channel.reg_ff22,
        0xFF23 => 0x80 | (if channel.length.enabled { 0x40 } else { 0 }) | 0x3F,
        _ => 0xFF,
    }
}

fn noise_write(channel: Noise_Channel, address: u16, value: u8, frame_step: u8) -> Noise_Channel {
    let updated = match address {
        0xFF20 => Noise_Channel { length: length_set(channel.length, value & 0x3F), ..channel },
        0xFF21 => {
            let dac = value & 0xF8 != 0;
            Noise_Channel { dac_enabled: dac, active: channel.active && dac, ..channel }
        }
        0xFF22 => noise_write_control(channel, value),
        0xFF23 => noise_write_high(channel, value, frame_step),
        _ => channel,
    };
    Noise_Channel { volume_envelope: envelope_write(updated.volume_envelope, address, value), ..updated }
}

// NR43: the LFSR width and the clock divider/shift that set the period.
fn noise_write_control(channel: Noise_Channel, value: u8) -> Noise_Channel {
    let shift_width = match value & 8 == 8 {
        true => 6,
        false => 14,
    };
    let freq_div = match value & 7 {
        0 => 8,
        n => n as u32 * 16,
    };
    Noise_Channel { reg_ff22: value, shift_width, period: freq_div << (value >> 4), ..channel }
}

fn noise_write_high(channel: Noise_Channel, value: u8, frame_step: u8) -> Noise_Channel {
    let length = length_enable(channel.length, value & 0x40 == 0x40, frame_step);
    let gated = Noise_Channel { length, active: channel.active && length_is_active(&length), ..channel };
    match value & 0x80 == 0x80 {
        false => gated,
        true => Noise_Channel {
            length: length_trigger(gated.length, frame_step),
            state: 0xFF,
            delay: 0,
            active: gated.dac_enabled,
            ..gated
        },
    }
}

fn noise_step_length(channel: Noise_Channel) -> Noise_Channel {
    let length = length_step(channel.length);
    Noise_Channel { length, active: channel.active && length_is_active(&length), ..channel }
}

// Synthesizes the noise output over [start, end): clocks the LFSR each period,
// emitting a blip delta on each polarity change (rboy's NoiseChannel::run).
fn noise_run(channel: Noise_Channel, start: u32, end: u32) -> Noise_Channel {
    match !channel.active {
        true => match channel.last_amp != 0 {
            true => Noise_Channel { blip: blip_buf::add_delta(channel.blip, start, -channel.last_amp), last_amp: 0, delay: 0, ..channel },
            false => channel,
        },
        false => {
            let from = start + channel.delay;
            noise_walk(channel, from, end)
        }
    }
}

fn noise_walk(channel: Noise_Channel, time: u32, end: u32) -> Noise_Channel {
    match time < end {
        false => Noise_Channel { delay: time - end, ..channel },
        true => {
            let oldstate = channel.state;
            let shifted = oldstate << 1;
            let bit = ((oldstate >> channel.shift_width) ^ (shifted >> channel.shift_width)) & 1;
            let volume = channel.volume_envelope.volume as i32;
            let amp = match (oldstate >> channel.shift_width) & 1 {
                0 => -volume,
                _ => volume,
            };
            let next_time = time + channel.period;
            let (blip, last_amp) = emit(channel.blip, channel.last_amp, time, amp);
            noise_walk(Noise_Channel { blip, last_amp, state: shifted | bit, ..channel }, next_time, end)
        }
    }
}

/// The whole APU: the four channels, the frame sequencer, the master control
/// registers, and the accumulated stereo audio.
#[derive(Clone, Debug)]
pub struct Sound {
    pub on: bool,
    pub time: u32,
    pub prev_time: u32,
    pub next_time: u32,
    pub output_period: u32,
    pub frame_step: u8,
    pub channel1: Square_Channel,
    pub channel2: Square_Channel,
    pub channel3: Wave_Channel,
    pub channel4: Noise_Channel,
    pub volume_left: u8,
    pub volume_right: u8,
    pub reg_vin_to_so: u8,
    pub reg_ff25: u8,
    pub dmg_mode: bool,
    pub audio_left: Vec<f32>,
    pub audio_right: Vec<f32>,
}

// A blip buffer sized and rated for the emulator (matches rboy's create_blipbuf).
fn create_blip() -> blip_buf::Blip_Buf {
    blip_buf::set_rates(blip_buf::new(OUTPUT_SAMPLE_COUNT + 1), CLOCKS_PER_SECOND as f64, SAMPLE_RATE as f64)
}

/// Builds the APU for a console mode, its post-boot state.
pub fn new(dmg_mode: bool) -> Sound {
    Sound {
        on: false,
        time: 0,
        prev_time: 0,
        next_time: CLOCKS_PER_FRAME,
        output_period: ((OUTPUT_SAMPLE_COUNT as u64 * CLOCKS_PER_SECOND as u64) / SAMPLE_RATE as u64) as u32,
        frame_step: 0,
        channel1: square_new(true),
        channel2: square_new(false),
        channel3: wave_new(dmg_mode),
        channel4: noise_new(),
        volume_left: 7,
        volume_right: 7,
        reg_vin_to_so: 0,
        reg_ff25: 0,
        dmg_mode,
        audio_left: Vec::new(),
        audio_right: Vec::new(),
    }
}

/// The console mode's APU: DMG or CGB (they differ only in wave-RAM access rules).
pub fn new_for(mode: gbmode::Gb_Mode) -> Sound {
    new(mode == gbmode::Gb_Mode::Classic)
}

pub fn read_byte(sound: &Sound, address: u16) -> u8 {
    match address {
        0xFF10..=0xFF14 => square_read(&sound.channel1, address),
        0xFF16..=0xFF19 => square_read(&sound.channel2, address),
        0xFF1A..=0xFF1E => wave_read(&sound.channel3, address),
        0xFF20..=0xFF23 => noise_read(&sound.channel4, address),
        0xFF24 => ((sound.volume_right & 7) << 4) | (sound.volume_left & 7) | sound.reg_vin_to_so,
        0xFF25 => sound.reg_ff25,
        0xFF26 => read_status(sound),
        0xFF30..=0xFF3F => wave_read(&sound.channel3, address),
        _ => 0xFF,
    }
}

// NR52: power bit, the three unused-read-1 bits, and the four channel-active flags.
fn read_status(sound: &Sound) -> u8 {
    (if sound.on { 0x80 } else { 0 })
        | 0x70
        | (if sound.channel4.active { 0x8 } else { 0 })
        | (if sound.channel3.active { 0x4 } else { 0 })
        | (if sound.channel2.active { 0x2 } else { 0 })
        | (if sound.channel1.active { 0x1 } else { 0 })
}

pub fn write_byte(sound: Sound, address: u16, value: u8) -> Sound {
    match sound.on {
        false => write_while_off(sound, address, value),
        true => write_register(sound, address, value),
    }
}

// While the APU is off, only NR52 and (in DMG) the length registers accept writes.
fn write_while_off(sound: Sound, address: u16, value: u8) -> Sound {
    let lengthed = match sound.dmg_mode {
        false => sound,
        true => match address {
            0xFF11 => Sound { channel1: square_write(sound.channel1, address, value & 0x3F, sound.frame_step), ..sound },
            0xFF16 => Sound { channel2: square_write(sound.channel2, address, value & 0x3F, sound.frame_step), ..sound },
            0xFF1B => write_register(sound, address, value),
            0xFF20 => Sound { channel4: noise_write(sound.channel4, address, value & 0x3F, sound.frame_step), ..sound },
            _ => sound,
        },
    };
    match address == 0xFF26 {
        true => write_register(lengthed, address, value),
        false => lengthed,
    }
}

// Dispatches a register write to the addressed channel or control register.
fn write_register(sound: Sound, address: u16, value: u8) -> Sound {
    let step = sound.frame_step;
    match address {
        0xFF10..=0xFF14 => Sound { channel1: square_write(sound.channel1, address, value, step), ..sound },
        0xFF16..=0xFF19 => Sound { channel2: square_write(sound.channel2, address, value, step), ..sound },
        0xFF1A..=0xFF1E => Sound { channel3: wave_write(sound.channel3, address, value, step), ..sound },
        0xFF20..=0xFF23 => Sound { channel4: noise_write(sound.channel4, address, value, step), ..sound },
        0xFF24 => Sound { volume_left: value & 0x7, volume_right: (value >> 4) & 0x7, reg_vin_to_so: value & 0x88, ..sound },
        0xFF25 => Sound { reg_ff25: value, ..sound },
        0xFF26 => write_power(sound, value),
        0xFF30..=0xFF3F => Sound { channel3: wave_write(sound.channel3, address, value, step), ..sound },
        _ => sound,
    }
}

// NR52 power: turning off zeroes every register; turning on resets the frame step.
fn write_power(sound: Sound, value: u8) -> Sound {
    let turn_on = value & 0x80 == 0x80;
    match (sound.on, turn_on) {
        (true, false) => Sound { on: false, ..power_off(sound) },
        (false, true) => Sound { on: true, frame_step: 0, ..sound },
        _ => Sound { on: turn_on, ..sound },
    }
}

// Clears 0xFF10-0xFF25 by writing zero to each, the way rboy does on power-off.
fn power_off(sound: Sound) -> Sound {
    (0xFF10..=0xFF25u16).fold(Sound { on: true, ..sound }, |acc, address| write_register(acc, address, 0))
}

/// Advances the APU by `cycles`, running the channels and frame sequencer up to the
/// new time, and flushing the synthesized samples once per output period.
pub fn do_cycle(sound: Sound, cycles: u32) -> Sound {
    match sound.on {
        false => sound,
        true => {
            let ran = run(Sound { time: sound.time + cycles, ..sound });
            match ran.time >= ran.output_period {
                true => do_output(ran),
                false => ran,
            }
        }
    }
}

// Runs the channels over each whole frame chunk (stepping the sequencer at the
// boundary), then the remaining partial chunk — rboy's `run()` loop, mut-free.
fn run(sound: Sound) -> Sound {
    match sound.next_time <= sound.time {
        true => {
            let boundary = sound.next_time;
            let start = sound.prev_time;
            let ran = run_channels(sound, start, boundary);
            let stepped = step_sequencer(ran);
            run(Sound {
                frame_step: (stepped.frame_step + 1) % 8,
                prev_time: boundary,
                next_time: boundary + CLOCKS_PER_FRAME,
                ..stepped
            })
        }
        false => run_tail(sound),
    }
}

fn run_tail(sound: Sound) -> Sound {
    match sound.prev_time != sound.time {
        true => {
            let time = sound.time;
            let start = sound.prev_time;
            Sound { prev_time: time, ..run_channels(sound, start, time) }
        }
        false => sound,
    }
}

// Advances all four channels over one time chunk.
fn run_channels(sound: Sound, start: u32, end: u32) -> Sound {
    Sound {
        channel1: square_run(sound.channel1, start, end),
        channel2: square_run(sound.channel2, start, end),
        channel3: wave_run(sound.channel3, start, end),
        channel4: noise_run(sound.channel4, start, end),
        ..sound
    }
}

// One frame-sequencer tick: length on even steps, sweep on step%4==2, envelope on 7.
fn step_sequencer(sound: Sound) -> Sound {
    let lengthed = match sound.frame_step % 2 == 0 {
        true => step_lengths(sound),
        false => sound,
    };
    let swept = match lengthed.frame_step % 4 == 2 {
        true => Sound { channel1: square_step_sweep(lengthed.channel1), ..lengthed },
        false => lengthed,
    };
    match swept.frame_step == 7 {
        true => step_envelopes(swept),
        false => swept,
    }
}

fn step_lengths(sound: Sound) -> Sound {
    Sound {
        channel1: square_step_length(sound.channel1),
        channel2: square_step_length(sound.channel2),
        channel3: wave_step_length(sound.channel3),
        channel4: noise_step_length(sound.channel4),
        ..sound
    }
}

fn step_envelopes(sound: Sound) -> Sound {
    Sound {
        channel1: Square_Channel { volume_envelope: envelope_step(sound.channel1.volume_envelope), ..sound.channel1 },
        channel2: Square_Channel { volume_envelope: envelope_step(sound.channel2.volume_envelope), ..sound.channel2 },
        channel4: Noise_Channel { volume_envelope: envelope_step(sound.channel4.volume_envelope), ..sound.channel4 },
        ..sound
    }
}

// Ends the blip frame on every channel, mixes the samples out, and rebases the clock.
fn do_output(sound: Sound) -> Sound {
    let time = sound.time;
    let mixed = mix_buffers(end_frames(sound, time));
    Sound { time: 0, prev_time: 0, next_time: mixed.next_time - time, ..mixed }
}

fn end_frames(sound: Sound, time: u32) -> Sound {
    Sound {
        channel1: Square_Channel { blip: blip_buf::end_frame(sound.channel1.blip, time), ..sound.channel1 },
        channel2: Square_Channel { blip: blip_buf::end_frame(sound.channel2.blip, time), ..sound.channel2 },
        channel3: Wave_Channel { blip: blip_buf::end_frame(sound.channel3.blip, time), ..sound.channel3 },
        channel4: Noise_Channel { blip: blip_buf::end_frame(sound.channel4.blip, time), ..sound.channel4 },
        ..sound
    }
}

// Reads the band-limited samples from each channel, applies the master volume and
// per-channel panning, and appends the stereo f32 result to the audio buffers.
fn mix_buffers(sound: Sound) -> Sound {
    let count = blip_buf::samples_avail(&sound.channel1.blip);
    let (blip1, s1) = blip_buf::read_samples(sound.channel1.blip, count);
    let (blip2, s2) = blip_buf::read_samples(sound.channel2.blip, count);
    let (blip3, s3) = blip_buf::read_samples(sound.channel3.blip, count);
    let (blip4, s4) = blip_buf::read_samples(sound.channel4.blip, count);
    let left_vol = (sound.volume_left as f32 / 7.0) * (1.0 / 15.0) * 0.25;
    let right_vol = (sound.volume_right as f32 / 7.0) * (1.0 / 15.0) * 0.25;
    let mask = sound.reg_ff25;
    let left: Vec<f32> = (0..count).map(|i| mix_sample(mask >> 4, &s1, &s2, &s3, &s4, i, left_vol)).collect();
    let right: Vec<f32> = (0..count).map(|i| mix_sample(mask, &s1, &s2, &s3, &s4, i, right_vol)).collect();
    Sound {
        channel1: Square_Channel { blip: blip1, ..sound.channel1 },
        channel2: Square_Channel { blip: blip2, ..sound.channel2 },
        channel3: Wave_Channel { blip: blip3, ..sound.channel3 },
        channel4: Noise_Channel { blip: blip4, ..sound.channel4 },
        audio_left: [sound.audio_left, left].concat(),
        audio_right: [sound.audio_right, right].concat(),
        ..sound
    }
}

// One mixed sample for one stereo side: the enabled channels summed in channel order
// (the wave channel at 4x amplitude is scaled back by 4), matching rboy's accumulate.
fn mix_sample(pan: u8, s1: &[i16], s2: &[i16], s3: &[i16], s4: &[i16], i: usize, vol: f32) -> f32 {
    (if pan & 0x01 != 0 { s1[i] as f32 * vol } else { 0.0 })
        + (if pan & 0x02 != 0 { s2[i] as f32 * vol } else { 0.0 })
        + (if pan & 0x04 != 0 { (s3[i] as f32 / 4.0) * vol } else { 0.0 })
        + (if pan & 0x08 != 0 { s4[i] as f32 * vol } else { 0.0 })
}
