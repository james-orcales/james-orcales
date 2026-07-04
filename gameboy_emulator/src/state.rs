//! Save states: a std-only serialization of the whole machine to bytes and back, so
//! the composition root can persist and restore a snapshot without a serialization
//! dependency in the core (rboy used serde + ciborium; here it is hand-rolled over
//! std). The format is gameboy_emulator's own — round-trippable, not cross-emulator. Each
//! type encodes to a `Vec<u8>` composed by `concat` (never a byte-at-a-time push, to
//! avoid an O(n^2) build); decoding threads a byte offset and fails to `None` on any
//! truncation, so a short or corrupt blob is rejected rather than panicking.

use crate::blip_buf;
use crate::cpu;
use crate::gbmode;
use crate::gpu;
use crate::keypad;
use crate::mbc;
use crate::mmu;
use crate::region;
use crate::register;
use crate::serial;
use crate::sound;
use crate::timer;
use std::array;

/// Serializes the whole machine to a byte blob.
pub fn save(machine: &cpu::Cpu) -> Vec<u8> {
    [
        encode_registers(&machine.reg),
        encode_mmu(&machine.mmu),
        e_bool(machine.halted),
        e_bool(machine.halt_bug),
        e_bool(machine.ime),
        e_u32(machine.setdi),
        e_u32(machine.setei),
    ]
    .concat()
}

/// Restores a machine from a byte blob, or `None` if it is truncated/corrupt.
pub fn load(bytes: &[u8]) -> Option<cpu::Cpu> {
    let (reg, o) = decode_registers(bytes, 0)?;
    let (mmu, o) = decode_mmu(bytes, o)?;
    let (halted, o) = d_bool(bytes, o)?;
    let (halt_bug, o) = d_bool(bytes, o)?;
    let (ime, o) = d_bool(bytes, o)?;
    let (setdi, o) = d_u32(bytes, o)?;
    let (setei, _) = d_u32(bytes, o)?;
    Some(cpu::Cpu { reg, mmu, halted, halt_bug, ime, setdi, setei })
}

// Scalar encoders: each returns its own bytes; parents compose them with `concat`.

fn e_u8(v: u8) -> Vec<u8> {
    vec![v]
}

fn e_u16(v: u16) -> Vec<u8> {
    v.to_le_bytes().to_vec()
}

fn e_u32(v: u32) -> Vec<u8> {
    v.to_le_bytes().to_vec()
}

fn e_u64(v: u64) -> Vec<u8> {
    v.to_le_bytes().to_vec()
}

fn e_i32(v: i32) -> Vec<u8> {
    v.to_le_bytes().to_vec()
}

fn e_usize(v: usize) -> Vec<u8> {
    (v as u64).to_le_bytes().to_vec()
}

fn e_bool(v: bool) -> Vec<u8> {
    vec![v as u8]
}

// A length-prefixed byte slice.
fn e_bytes(v: &[u8]) -> Vec<u8> {
    [e_u32(v.len() as u32), v.to_vec()].concat()
}

fn e_i32s(v: &[i32]) -> Vec<u8> {
    [e_u32(v.len() as u32), v.iter().flat_map(|x| x.to_le_bytes()).collect()].concat()
}

fn e_f32s(v: &[f32]) -> Vec<u8> {
    [e_u32(v.len() as u32), v.iter().flat_map(|x| x.to_le_bytes()).collect()].concat()
}

// Scalar decoders: each returns the value and the advanced offset.

fn d_u8(b: &[u8], o: usize) -> Option<(u8, usize)> {
    b.get(o).map(|&v| (v, o + 1))
}

fn d_u16(b: &[u8], o: usize) -> Option<(u16, usize)> {
    b.get(o..o + 2).map(|s| (u16::from_le_bytes(s.try_into().unwrap()), o + 2))
}

fn d_u32(b: &[u8], o: usize) -> Option<(u32, usize)> {
    b.get(o..o + 4).map(|s| (u32::from_le_bytes(s.try_into().unwrap()), o + 4))
}

fn d_u64(b: &[u8], o: usize) -> Option<(u64, usize)> {
    b.get(o..o + 8).map(|s| (u64::from_le_bytes(s.try_into().unwrap()), o + 8))
}

fn d_i32(b: &[u8], o: usize) -> Option<(i32, usize)> {
    b.get(o..o + 4).map(|s| (i32::from_le_bytes(s.try_into().unwrap()), o + 4))
}

fn d_usize(b: &[u8], o: usize) -> Option<(usize, usize)> {
    d_u64(b, o).map(|(v, o)| (v as usize, o))
}

fn d_bool(b: &[u8], o: usize) -> Option<(bool, usize)> {
    d_u8(b, o).map(|(v, o)| (v != 0, o))
}

fn d_bytes(b: &[u8], o: usize) -> Option<(Vec<u8>, usize)> {
    let (len, o) = d_u32(b, o)?;
    let end = o + len as usize;
    b.get(o..end).map(|s| (s.to_vec(), end))
}

fn d_i32s(b: &[u8], o: usize) -> Option<(Vec<i32>, usize)> {
    let (len, o) = d_u32(b, o)?;
    let end = o + len as usize * 4;
    let slice = b.get(o..end)?;
    Some((slice.chunks_exact(4).map(|c| i32::from_le_bytes(c.try_into().unwrap())).collect(), end))
}

fn d_f32s(b: &[u8], o: usize) -> Option<(Vec<f32>, usize)> {
    let (len, o) = d_u32(b, o)?;
    let end = o + len as usize * 4;
    let slice = b.get(o..end)?;
    Some((slice.chunks_exact(4).map(|c| f32::from_le_bytes(c.try_into().unwrap())).collect(), end))
}

// Fixed-size arrays, options, and enums.

fn d_arr3(b: &[u8], o: usize) -> Option<([u8; 3], usize)> {
    b.get(o..o + 3).map(|s| ([s[0], s[1], s[2]], o + 3))
}

fn d_arr4(b: &[u8], o: usize) -> Option<([u8; 4], usize)> {
    b.get(o..o + 4).map(|s| ([s[0], s[1], s[2], s[3]], o + 4))
}

fn d_arr5(b: &[u8], o: usize) -> Option<([u8; 5], usize)> {
    b.get(o..o + 5).map(|s| ([s[0], s[1], s[2], s[3], s[4]], o + 5))
}

fn e_palette(pal: &[[[u8; 3]; 4]; 8]) -> Vec<u8> {
    pal.iter().flatten().flatten().copied().collect()
}

fn d_palette(b: &[u8], o: usize) -> Option<([[[u8; 3]; 4]; 8], usize)> {
    let s = b.get(o..o + 96)?;
    Some((array::from_fn(|p| array::from_fn(|c| array::from_fn(|k| s[p * 12 + c * 3 + k]))), o + 96))
}

fn e_opt_u64(v: Option<u64>) -> Vec<u8> {
    match v {
        Some(x) => [e_u8(1), e_u64(x)].concat(),
        None => e_u8(0),
    }
}

fn d_opt_u64(b: &[u8], o: usize) -> Option<(Option<u64>, usize)> {
    let (tag, o) = d_u8(b, o)?;
    match tag {
        0 => Some((None, o)),
        _ => d_u64(b, o).map(|(x, o)| (Some(x), o)),
    }
}

fn e_mode(mode: gbmode::Gb_Mode) -> Vec<u8> {
    e_u8(match mode {
        gbmode::Gb_Mode::Classic => 0,
        gbmode::Gb_Mode::Color => 1,
        gbmode::Gb_Mode::Color_As_Classic => 2,
    })
}

fn d_mode(b: &[u8], o: usize) -> Option<(gbmode::Gb_Mode, usize)> {
    d_u8(b, o).map(|(v, o)| {
        (
            match v {
                1 => gbmode::Gb_Mode::Color,
                2 => gbmode::Gb_Mode::Color_As_Classic,
                _ => gbmode::Gb_Mode::Classic,
            },
            o,
        )
    })
}

fn e_speed(speed: gbmode::Gb_Speed) -> Vec<u8> {
    e_u8(match speed {
        gbmode::Gb_Speed::Single => 1,
        gbmode::Gb_Speed::Double => 2,
    })
}

fn d_speed(b: &[u8], o: usize) -> Option<(gbmode::Gb_Speed, usize)> {
    d_u8(b, o).map(|(v, o)| {
        (
            match v {
                2 => gbmode::Gb_Speed::Double,
                _ => gbmode::Gb_Speed::Single,
            },
            o,
        )
    })
}

fn e_dma(dma: mmu::Dma_Type) -> Vec<u8> {
    e_u8(match dma {
        mmu::Dma_Type::No_Dma => 0,
        mmu::Dma_Type::Gdma => 1,
        mmu::Dma_Type::Hdma => 2,
    })
}

fn d_dma(b: &[u8], o: usize) -> Option<(mmu::Dma_Type, usize)> {
    d_u8(b, o).map(|(v, o)| {
        (
            match v {
                1 => mmu::Dma_Type::Gdma,
                2 => mmu::Dma_Type::Hdma,
                _ => mmu::Dma_Type::No_Dma,
            },
            o,
        )
    })
}

// Registers.

fn encode_registers(reg: &register::Registers) -> Vec<u8> {
    [
        e_u8(reg.a), e_u8(reg.f), e_u8(reg.b), e_u8(reg.c), e_u8(reg.d), e_u8(reg.e),
        e_u8(reg.h), e_u8(reg.l), e_u16(reg.pc), e_u16(reg.sp),
    ]
    .concat()
}

fn decode_registers(b: &[u8], o: usize) -> Option<(register::Registers, usize)> {
    let (a, o) = d_u8(b, o)?;
    let (f, o) = d_u8(b, o)?;
    let (bc, o) = d_u8(b, o)?;
    let (c, o) = d_u8(b, o)?;
    let (d, o) = d_u8(b, o)?;
    let (e, o) = d_u8(b, o)?;
    let (h, o) = d_u8(b, o)?;
    let (l, o) = d_u8(b, o)?;
    let (pc, o) = d_u16(b, o)?;
    let (sp, o) = d_u16(b, o)?;
    Some((register::Registers { a, f, b: bc, c, d, e, h, l, pc, sp }, o))
}

// Envelope, length counter, blip buffer.

fn encode_envelope(env: &sound::Volume_Envelope) -> Vec<u8> {
    [e_u8(env.period), e_bool(env.goes_up), e_u8(env.delay), e_u8(env.initial_volume), e_u8(env.volume)].concat()
}

fn decode_envelope(b: &[u8], o: usize) -> Option<(sound::Volume_Envelope, usize)> {
    let (period, o) = d_u8(b, o)?;
    let (goes_up, o) = d_bool(b, o)?;
    let (delay, o) = d_u8(b, o)?;
    let (initial_volume, o) = d_u8(b, o)?;
    let (volume, o) = d_u8(b, o)?;
    Some((sound::Volume_Envelope { period, goes_up, delay, initial_volume, volume }, o))
}

fn encode_length(length: &sound::Length_Counter) -> Vec<u8> {
    [e_bool(length.enabled), e_u16(length.value), e_u16(length.max)].concat()
}

fn decode_length(b: &[u8], o: usize) -> Option<(sound::Length_Counter, usize)> {
    let (enabled, o) = d_bool(b, o)?;
    let (value, o) = d_u16(b, o)?;
    let (max, o) = d_u16(b, o)?;
    Some((sound::Length_Counter { enabled, value, max }, o))
}

fn encode_blip(blip: &blip_buf::Blip_Buf) -> Vec<u8> {
    [e_u64(blip.factor), e_u64(blip.offset), e_i32(blip.integrator), e_usize(blip.avail), e_i32s(&blip.samples)].concat()
}

fn decode_blip(b: &[u8], o: usize) -> Option<(blip_buf::Blip_Buf, usize)> {
    let (factor, o) = d_u64(b, o)?;
    let (offset, o) = d_u64(b, o)?;
    let (integrator, o) = d_i32(b, o)?;
    let (avail, o) = d_usize(b, o)?;
    let (samples, o) = d_i32s(b, o)?;
    Some((blip_buf::Blip_Buf { factor, offset, integrator, avail, samples }, o))
}

// Channels.

fn encode_square(channel: &sound::Square_Channel) -> Vec<u8> {
    [
        e_bool(channel.active), e_bool(channel.dac_enabled), e_u8(channel.duty), e_u8(channel.phase),
        encode_length(&channel.length), e_u16(channel.frequency), e_u32(channel.period), e_i32(channel.last_amp),
        e_u32(channel.delay), e_bool(channel.has_sweep), e_bool(channel.sweep_enabled), e_u16(channel.sweep_frequency),
        e_u8(channel.sweep_delay), e_u8(channel.sweep_period), e_u8(channel.sweep_shift), e_bool(channel.sweep_negate),
        e_bool(channel.sweep_did_negate), encode_envelope(&channel.volume_envelope), encode_blip(&channel.blip),
    ]
    .concat()
}

fn decode_square(b: &[u8], o: usize) -> Option<(sound::Square_Channel, usize)> {
    let (active, o) = d_bool(b, o)?;
    let (dac_enabled, o) = d_bool(b, o)?;
    let (duty, o) = d_u8(b, o)?;
    let (phase, o) = d_u8(b, o)?;
    let (length, o) = decode_length(b, o)?;
    let (frequency, o) = d_u16(b, o)?;
    let (period, o) = d_u32(b, o)?;
    let (last_amp, o) = d_i32(b, o)?;
    let (delay, o) = d_u32(b, o)?;
    let (has_sweep, o) = d_bool(b, o)?;
    let (sweep_enabled, o) = d_bool(b, o)?;
    let (sweep_frequency, o) = d_u16(b, o)?;
    let (sweep_delay, o) = d_u8(b, o)?;
    let (sweep_period, o) = d_u8(b, o)?;
    let (sweep_shift, o) = d_u8(b, o)?;
    let (sweep_negate, o) = d_bool(b, o)?;
    let (sweep_did_negate, o) = d_bool(b, o)?;
    let (volume_envelope, o) = decode_envelope(b, o)?;
    let (blip, o) = decode_blip(b, o)?;
    Some((
        sound::Square_Channel {
            active, dac_enabled, duty, phase, length, frequency, period, last_amp, delay, has_sweep,
            sweep_enabled, sweep_frequency, sweep_delay, sweep_period, sweep_shift, sweep_negate,
            sweep_did_negate, volume_envelope, blip,
        },
        o,
    ))
}

fn encode_wave(channel: &sound::Wave_Channel) -> Vec<u8> {
    [
        e_bool(channel.active), e_bool(channel.dac_enabled), encode_length(&channel.length), e_u16(channel.frequency),
        e_u32(channel.period), e_i32(channel.last_amp), e_u32(channel.delay), e_u8(channel.volume_shift),
        e_bytes(&channel.waveram), e_u8(channel.current_wave), e_bool(channel.sample_recently_accessed),
        e_bool(channel.dmg_mode), encode_blip(&channel.blip),
    ]
    .concat()
}

fn decode_wave(b: &[u8], o: usize) -> Option<(sound::Wave_Channel, usize)> {
    let (active, o) = d_bool(b, o)?;
    let (dac_enabled, o) = d_bool(b, o)?;
    let (length, o) = decode_length(b, o)?;
    let (frequency, o) = d_u16(b, o)?;
    let (period, o) = d_u32(b, o)?;
    let (last_amp, o) = d_i32(b, o)?;
    let (delay, o) = d_u32(b, o)?;
    let (volume_shift, o) = d_u8(b, o)?;
    let (waveram, o) = d_bytes(b, o)?;
    let (current_wave, o) = d_u8(b, o)?;
    let (sample_recently_accessed, o) = d_bool(b, o)?;
    let (dmg_mode, o) = d_bool(b, o)?;
    let (blip, o) = decode_blip(b, o)?;
    Some((
        sound::Wave_Channel {
            active, dac_enabled, length, frequency, period, last_amp, delay, volume_shift, waveram,
            current_wave, sample_recently_accessed, dmg_mode, blip,
        },
        o,
    ))
}

fn encode_noise(channel: &sound::Noise_Channel) -> Vec<u8> {
    [
        e_bool(channel.active), e_bool(channel.dac_enabled), e_u8(channel.reg_ff22), encode_length(&channel.length),
        encode_envelope(&channel.volume_envelope), e_u32(channel.period), e_u8(channel.shift_width),
        e_u16(channel.state), e_u32(channel.delay), e_i32(channel.last_amp), encode_blip(&channel.blip),
    ]
    .concat()
}

fn decode_noise(b: &[u8], o: usize) -> Option<(sound::Noise_Channel, usize)> {
    let (active, o) = d_bool(b, o)?;
    let (dac_enabled, o) = d_bool(b, o)?;
    let (reg_ff22, o) = d_u8(b, o)?;
    let (length, o) = decode_length(b, o)?;
    let (volume_envelope, o) = decode_envelope(b, o)?;
    let (period, o) = d_u32(b, o)?;
    let (shift_width, o) = d_u8(b, o)?;
    let (state, o) = d_u16(b, o)?;
    let (delay, o) = d_u32(b, o)?;
    let (last_amp, o) = d_i32(b, o)?;
    let (blip, o) = decode_blip(b, o)?;
    Some((
        sound::Noise_Channel {
            active, dac_enabled, reg_ff22, length, volume_envelope, period, shift_width, state, delay, last_amp, blip,
        },
        o,
    ))
}

// Sound.

fn encode_sound(s: &sound::Sound) -> Vec<u8> {
    [
        e_bool(s.on), e_u32(s.time), e_u32(s.prev_time), e_u32(s.next_time), e_u32(s.output_period), e_u8(s.frame_step),
        encode_square(&s.channel1), encode_square(&s.channel2), encode_wave(&s.channel3), encode_noise(&s.channel4),
        e_u8(s.volume_left), e_u8(s.volume_right), e_u8(s.reg_vin_to_so), e_u8(s.reg_ff25), e_bool(s.dmg_mode),
        e_f32s(&s.audio_left), e_f32s(&s.audio_right),
    ]
    .concat()
}

fn decode_sound(b: &[u8], o: usize) -> Option<(sound::Sound, usize)> {
    let (on, o) = d_bool(b, o)?;
    let (time, o) = d_u32(b, o)?;
    let (prev_time, o) = d_u32(b, o)?;
    let (next_time, o) = d_u32(b, o)?;
    let (output_period, o) = d_u32(b, o)?;
    let (frame_step, o) = d_u8(b, o)?;
    let (channel1, o) = decode_square(b, o)?;
    let (channel2, o) = decode_square(b, o)?;
    let (channel3, o) = decode_wave(b, o)?;
    let (channel4, o) = decode_noise(b, o)?;
    let (volume_left, o) = d_u8(b, o)?;
    let (volume_right, o) = d_u8(b, o)?;
    let (reg_vin_to_so, o) = d_u8(b, o)?;
    let (reg_ff25, o) = d_u8(b, o)?;
    let (dmg_mode, o) = d_bool(b, o)?;
    let (audio_left, o) = d_f32s(b, o)?;
    let (audio_right, o) = d_f32s(b, o)?;
    Some((
        sound::Sound {
            on, time, prev_time, next_time, output_period, frame_step, channel1, channel2, channel3, channel4,
            volume_left, volume_right, reg_vin_to_so, reg_ff25, dmg_mode, audio_left, audio_right,
        },
        o,
    ))
}

// Serial, timer, keypad.

fn encode_serial(s: &serial::Serial) -> Vec<u8> {
    [e_u8(s.data), e_u8(s.control), e_u8(s.interrupt), e_bytes(&s.output)].concat()
}

fn decode_serial(b: &[u8], o: usize) -> Option<(serial::Serial, usize)> {
    let (data, o) = d_u8(b, o)?;
    let (control, o) = d_u8(b, o)?;
    let (interrupt, o) = d_u8(b, o)?;
    let (output, o) = d_bytes(b, o)?;
    // The peer is not part of a snapshot (rboy does not serialize its serial callback
    // either); a restored link starts as an idle capture line.
    Some((serial::Serial { data, control, interrupt, output, target: serial::Serial_Target::Capture }, o))
}

fn encode_timer(t: &timer::Timer) -> Vec<u8> {
    [
        e_u8(t.divider), e_u8(t.counter), e_u8(t.modulo), e_bool(t.enabled), e_u32(t.step), e_u32(t.internalcnt),
        e_u32(t.internaldiv), e_u8(t.interrupt),
    ]
    .concat()
}

fn decode_timer(b: &[u8], o: usize) -> Option<(timer::Timer, usize)> {
    let (divider, o) = d_u8(b, o)?;
    let (counter, o) = d_u8(b, o)?;
    let (modulo, o) = d_u8(b, o)?;
    let (enabled, o) = d_bool(b, o)?;
    let (step, o) = d_u32(b, o)?;
    let (internalcnt, o) = d_u32(b, o)?;
    let (internaldiv, o) = d_u32(b, o)?;
    let (interrupt, o) = d_u8(b, o)?;
    Some((timer::Timer { divider, counter, modulo, enabled, step, internalcnt, internaldiv, interrupt }, o))
}

fn encode_keypad(k: &keypad::Keypad) -> Vec<u8> {
    [e_u8(k.row0), e_u8(k.row1), e_u8(k.data), e_u8(k.interrupt)].concat()
}

fn decode_keypad(b: &[u8], o: usize) -> Option<(keypad::Keypad, usize)> {
    let (row0, o) = d_u8(b, o)?;
    let (row1, o) = d_u8(b, o)?;
    let (data, o) = d_u8(b, o)?;
    let (interrupt, o) = d_u8(b, o)?;
    Some((keypad::Keypad { row0, row1, data, interrupt }, o))
}

// GPU, split so each decode stays under the function-size limit.

fn encode_gpu(g: &gpu::Gpu) -> Vec<u8> {
    [
        e_u8(g.mode), e_u32(g.modeclock), e_u8(g.line), e_u8(g.lyc), e_bool(g.lcd_on), e_u16(g.win_tilemap),
        e_bool(g.win_on), e_u16(g.tilebase), e_u16(g.bg_tilemap), e_u32(g.sprite_size), e_bool(g.sprite_on),
        e_bool(g.lcdc0), e_bool(g.lyc_inte), e_bool(g.m0_inte), e_bool(g.m1_inte), e_bool(g.m2_inte), e_u8(g.scy),
        e_u8(g.scx), e_u8(g.winy), e_u8(g.winx), e_bool(g.wy_trigger), e_i32(g.wy_pos), e_u8(g.palbr), e_u8(g.pal0r),
        e_u8(g.pal1r), g.palb.to_vec(), g.pal0.to_vec(), g.pal1.to_vec(), e_bytes(&g.vram), e_bytes(&g.voam),
        e_usize(g.vrambank), e_bool(g.cbgpal_inc), e_u8(g.cbgpal_ind), e_palette(&g.cbgpal), e_bool(g.csprit_inc),
        e_u8(g.csprit_ind), e_palette(&g.csprit), e_bytes(&region::to_vec(&g.data)), e_bool(g.updated), e_u8(g.interrupt),
        e_mode(g.gbmode), e_bool(g.hblanking), e_bool(g.first_frame),
    ]
    .concat()
}

fn decode_gpu(b: &[u8], o: usize) -> Option<(gpu::Gpu, usize)> {
    let (mode, o) = d_u8(b, o)?;
    let (modeclock, o) = d_u32(b, o)?;
    let (line, o) = d_u8(b, o)?;
    let (lyc, o) = d_u8(b, o)?;
    let (lcd_on, o) = d_bool(b, o)?;
    let (win_tilemap, o) = d_u16(b, o)?;
    let (win_on, o) = d_bool(b, o)?;
    let (tilebase, o) = d_u16(b, o)?;
    let (bg_tilemap, o) = d_u16(b, o)?;
    let (sprite_size, o) = d_u32(b, o)?;
    let (sprite_on, o) = d_bool(b, o)?;
    let (lcdc0, o) = d_bool(b, o)?;
    let (lyc_inte, o) = d_bool(b, o)?;
    let (m0_inte, o) = d_bool(b, o)?;
    let (m1_inte, o) = d_bool(b, o)?;
    let (m2_inte, o) = d_bool(b, o)?;
    let head = gpu::Gpu {
        mode, modeclock, line, lyc, lcd_on, win_tilemap, win_on, tilebase, bg_tilemap, sprite_size, sprite_on,
        lcdc0, lyc_inte, m0_inte, m1_inte, m2_inte, ..gpu::new()
    };
    decode_gpu_tail(b, o, head)
}

fn decode_gpu_tail(b: &[u8], o: usize, head: gpu::Gpu) -> Option<(gpu::Gpu, usize)> {
    let (scy, o) = d_u8(b, o)?;
    let (scx, o) = d_u8(b, o)?;
    let (winy, o) = d_u8(b, o)?;
    let (winx, o) = d_u8(b, o)?;
    let (wy_trigger, o) = d_bool(b, o)?;
    let (wy_pos, o) = d_i32(b, o)?;
    let (palbr, o) = d_u8(b, o)?;
    let (pal0r, o) = d_u8(b, o)?;
    let (pal1r, o) = d_u8(b, o)?;
    let (palb, o) = d_arr4(b, o)?;
    let (pal0, o) = d_arr4(b, o)?;
    let (pal1, o) = d_arr4(b, o)?;
    let (vram, o) = d_bytes(b, o)?;
    let (voam, o) = d_bytes(b, o)?;
    let (vrambank, o) = d_usize(b, o)?;
    let (cbgpal_inc, o) = d_bool(b, o)?;
    let (cbgpal_ind, o) = d_u8(b, o)?;
    let (cbgpal, o) = d_palette(b, o)?;
    let (csprit_inc, o) = d_bool(b, o)?;
    let (csprit_ind, o) = d_u8(b, o)?;
    let (csprit, o) = d_palette(b, o)?;
    let (data_bytes, o) = d_bytes(b, o)?;
    let (updated, o) = d_bool(b, o)?;
    let (interrupt, o) = d_u8(b, o)?;
    let (gbmode, o) = d_mode(b, o)?;
    let (hblanking, o) = d_bool(b, o)?;
    let (first_frame, o) = d_bool(b, o)?;
    Some((
        gpu::Gpu {
            scy, scx, winy, winx, wy_trigger, wy_pos, palbr, pal0r, pal1r, palb, pal0, pal1, vram, voam, vrambank,
            cbgpal_inc, cbgpal_ind, cbgpal: Box::new(cbgpal), csprit_inc, csprit_ind, csprit: Box::new(csprit),
            data: region::from_bytes(&data_bytes),
            updated, interrupt, gbmode, hblanking, first_frame, ..head
        },
        o,
    ))
}

// MBC.

fn encode_mbc(mbc: &mbc::Mbc) -> Vec<u8> {
    match mbc {
        mbc::Mbc::Mbc0(c) => [e_u8(0), e_bytes(&c.rom)].concat(),
        mbc::Mbc::Mbc1(c) => [
            e_u8(1), e_bytes(&c.rom), e_bytes(&c.ram), e_bool(c.ram_on), e_bool(c.ram_updated), e_u8(c.banking_mode),
            e_usize(c.rombank), e_usize(c.rambank), e_bool(c.has_battery), e_usize(c.rombanks), e_usize(c.rambanks),
        ]
        .concat(),
        mbc::Mbc::Mbc2(c) => [
            e_u8(2), e_bytes(&c.rom), e_bytes(&c.ram), e_bool(c.ram_on), e_bool(c.ram_updated), e_usize(c.rombank),
            e_bool(c.has_battery), e_usize(c.rombanks),
        ]
        .concat(),
        mbc::Mbc::Mbc3(c) => [
            e_u8(3), e_bytes(&c.rom), e_bytes(&c.ram), e_usize(c.rombank), e_usize(c.rambank), e_usize(c.rambanks),
            e_bool(c.selectrtc), e_bool(c.ram_on), e_bool(c.ram_updated), e_bool(c.has_battery), c.rtc_ram.to_vec(),
            c.rtc_ram_latch.to_vec(), e_opt_u64(c.rtc_zero), e_u64(c.now),
        ]
        .concat(),
        mbc::Mbc::Mbc5(c) => [
            e_u8(5), e_bytes(&c.rom), e_bytes(&c.ram), e_usize(c.rombank), e_usize(c.rambank), e_bool(c.ram_on),
            e_bool(c.ram_updated), e_bool(c.has_battery), e_usize(c.rombanks), e_usize(c.rambanks),
        ]
        .concat(),
    }
}

fn decode_mbc(b: &[u8], o: usize) -> Option<(mbc::Mbc, usize)> {
    let (tag, o) = d_u8(b, o)?;
    match tag {
        0 => d_bytes(b, o).map(|(rom, o)| (mbc::Mbc::Mbc0(mbc::Mbc0 { rom }), o)),
        1 => decode_mbc1(b, o),
        2 => decode_mbc2(b, o),
        3 => decode_mbc3(b, o),
        _ => decode_mbc5(b, o),
    }
}

fn decode_mbc1(b: &[u8], o: usize) -> Option<(mbc::Mbc, usize)> {
    let (rom, o) = d_bytes(b, o)?;
    let (ram, o) = d_bytes(b, o)?;
    let (ram_on, o) = d_bool(b, o)?;
    let (ram_updated, o) = d_bool(b, o)?;
    let (banking_mode, o) = d_u8(b, o)?;
    let (rombank, o) = d_usize(b, o)?;
    let (rambank, o) = d_usize(b, o)?;
    let (has_battery, o) = d_bool(b, o)?;
    let (rombanks, o) = d_usize(b, o)?;
    let (rambanks, o) = d_usize(b, o)?;
    Some((
        mbc::Mbc::Mbc1(mbc::Mbc1 {
            rom, ram, ram_on, ram_updated, banking_mode, rombank, rambank, has_battery, rombanks, rambanks,
        }),
        o,
    ))
}

fn decode_mbc2(b: &[u8], o: usize) -> Option<(mbc::Mbc, usize)> {
    let (rom, o) = d_bytes(b, o)?;
    let (ram, o) = d_bytes(b, o)?;
    let (ram_on, o) = d_bool(b, o)?;
    let (ram_updated, o) = d_bool(b, o)?;
    let (rombank, o) = d_usize(b, o)?;
    let (has_battery, o) = d_bool(b, o)?;
    let (rombanks, o) = d_usize(b, o)?;
    Some((mbc::Mbc::Mbc2(mbc::Mbc2 { rom, ram, ram_on, ram_updated, rombank, has_battery, rombanks }), o))
}

fn decode_mbc3(b: &[u8], o: usize) -> Option<(mbc::Mbc, usize)> {
    let (rom, o) = d_bytes(b, o)?;
    let (ram, o) = d_bytes(b, o)?;
    let (rombank, o) = d_usize(b, o)?;
    let (rambank, o) = d_usize(b, o)?;
    let (rambanks, o) = d_usize(b, o)?;
    let (selectrtc, o) = d_bool(b, o)?;
    let (ram_on, o) = d_bool(b, o)?;
    let (ram_updated, o) = d_bool(b, o)?;
    let (has_battery, o) = d_bool(b, o)?;
    let (rtc_ram, o) = d_arr5(b, o)?;
    let (rtc_ram_latch, o) = d_arr5(b, o)?;
    let (rtc_zero, o) = d_opt_u64(b, o)?;
    let (now, o) = d_u64(b, o)?;
    Some((
        mbc::Mbc::Mbc3(mbc::Mbc3 {
            rom, ram, rombank, rambank, rambanks, selectrtc, ram_on, ram_updated, has_battery, rtc_ram,
            rtc_ram_latch, rtc_zero, now,
        }),
        o,
    ))
}

fn decode_mbc5(b: &[u8], o: usize) -> Option<(mbc::Mbc, usize)> {
    let (rom, o) = d_bytes(b, o)?;
    let (ram, o) = d_bytes(b, o)?;
    let (rombank, o) = d_usize(b, o)?;
    let (rambank, o) = d_usize(b, o)?;
    let (ram_on, o) = d_bool(b, o)?;
    let (ram_updated, o) = d_bool(b, o)?;
    let (has_battery, o) = d_bool(b, o)?;
    let (rombanks, o) = d_usize(b, o)?;
    let (rambanks, o) = d_usize(b, o)?;
    Some((
        mbc::Mbc::Mbc5(mbc::Mbc5 {
            rom, ram, rombank, rambank, ram_on, ram_updated, has_battery, rombanks, rambanks,
        }),
        o,
    ))
}

// MMU.

fn encode_mmu(m: &mmu::Mmu) -> Vec<u8> {
    [
        e_bytes(&region::to_vec(&m.wram)), e_bytes(&m.zram), e_bytes(&m.hdma), e_u8(m.inte), e_u8(m.intf), encode_serial(&m.serial),
        encode_timer(&m.timer), encode_keypad(&m.keypad), encode_gpu(&m.gpu), encode_sound(&m.sound),
        e_dma(m.hdma_status), e_u16(m.hdma_src), e_u16(m.hdma_dst), e_u8(m.hdma_len), e_usize(m.wrambank),
        encode_mbc(&m.mbc), e_mode(m.gbmode), e_speed(m.gbspeed), e_bool(m.speed_switch_req),
        m.undocumented_cgb_regs.to_vec(),
    ]
    .concat()
}

fn decode_mmu(b: &[u8], o: usize) -> Option<(mmu::Mmu, usize)> {
    let (wram_bytes, o) = d_bytes(b, o)?;
    let (zram, o) = d_bytes(b, o)?;
    let (hdma, o) = d_bytes(b, o)?;
    let (inte, o) = d_u8(b, o)?;
    let (intf, o) = d_u8(b, o)?;
    let (serial, o) = decode_serial(b, o)?;
    let (timer, o) = decode_timer(b, o)?;
    let (keypad, o) = decode_keypad(b, o)?;
    let (gpu, o) = decode_gpu(b, o)?;
    let (sound, o) = decode_sound(b, o)?;
    let (hdma_status, o) = d_dma(b, o)?;
    let (hdma_src, o) = d_u16(b, o)?;
    let (hdma_dst, o) = d_u16(b, o)?;
    let (hdma_len, o) = d_u8(b, o)?;
    let (wrambank, o) = d_usize(b, o)?;
    let (mbc, o) = decode_mbc(b, o)?;
    let (gbmode, o) = d_mode(b, o)?;
    let (gbspeed, o) = d_speed(b, o)?;
    let (speed_switch_req, o) = d_bool(b, o)?;
    let (regs, o) = d_arr3(b, o)?;
    Some((
        mmu::Mmu {
            wram: region::from_bytes(&wram_bytes), zram, hdma, inte, intf, serial, timer, keypad, gpu, sound: Box::new(sound), hdma_status, hdma_src, hdma_dst,
            hdma_len, wrambank, mbc: Box::new(mbc), gbmode, gbspeed, speed_switch_req, undocumented_cgb_regs: regs,
        },
        o,
    ))
}

#[cfg(test)]
mod tests {
    use crate::cpu;
    use crate::device;
    use crate::gbmode;
    use crate::mbc;
    use crate::state;

    // A machine advanced a little so its state is not all default. The ROM is an
    // infinite `JR -2` at the entry point, so the CPU idles there while the timer and
    // PPU advance, rather than executing into random-initialized WRAM.
    fn machine() -> cpu::Cpu {
        let rom: Vec<u8> = (0..0x8000usize)
            .map(|i| match i {
                0x100 => 0x18,
                0x101 => 0xFE,
                _ => 0,
            })
            .collect();
        let built = cpu::new(mbc::get_mbc(rom, true).unwrap(), gbmode::Gb_Mode::Classic).unwrap();
        device::run(built, 200000)
    }

    #[test]
    fn save_load_round_trips() {
        let bytes = state::save(&machine());
        let restored = state::load(&bytes).unwrap();
        // Re-serializing the restored machine yields the identical bytes.
        assert_eq!(state::save(&restored), bytes);
    }

    #[test]
    fn truncated_state_is_rejected() {
        let bytes = state::save(&machine());
        assert!(state::load(&bytes[..bytes.len() / 2]).is_none());
    }
}
