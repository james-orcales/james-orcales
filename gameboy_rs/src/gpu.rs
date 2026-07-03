//! The PPU: timing state machine, LCD registers, and VRAM/OAM storage. rboy drove
//! this through `&mut self`; here reads are pure and every write or tick returns a
//! new `Gpu`. Two dialect-driven representation choices: VRAM, OAM, and the
//! framebuffer are `Vec<u8>` rather than rboy's inline `[u8; N]` arrays, because a
//! value-threaded struct is moved on every tick and moving a 16 KiB inline array is
//! a memcpy — a heap `Vec` moves as a pointer, with writes paying the rebuild tax in
//! `memory`. And `bgprio` is a render-local, not a field, since it is scratch.
//!
//! Pixel rendering (`render_scan`) lands in M3 and CGB palette RAM in M5; the CPU
//! and serial milestones need only the timing here (LY, mode changes, the VBlank,
//! STAT, and LYC interrupts), so the framebuffer is not yet drawn.

use crate::gbmode;
use crate::gpu_render;
use crate::memory;
use crate::region;
use std::array;

/// The visible screen dimensions; the framebuffer is `SCREEN_W * SCREEN_H * 3` RGB.
pub const SCREEN_W: usize = 160;
pub const SCREEN_H: usize = 144;

const VRAM_SIZE: usize = 0x4000;
const VOAM_SIZE: usize = 0xA0;

/// The PPU state: timing, the decoded LCDC/STAT control bits, scroll and palette
/// registers, VRAM/OAM, the framebuffer, and the pending interrupt byte.
#[derive(Clone, Debug)]
pub struct Gpu {
    pub mode: u8,
    pub modeclock: u32,
    pub line: u8,
    pub lyc: u8,
    pub lcd_on: bool,
    pub win_tilemap: u16,
    pub win_on: bool,
    pub tilebase: u16,
    pub bg_tilemap: u16,
    pub sprite_size: u32,
    pub sprite_on: bool,
    pub lcdc0: bool,
    pub lyc_inte: bool,
    pub m0_inte: bool,
    pub m1_inte: bool,
    pub m2_inte: bool,
    pub scy: u8,
    pub scx: u8,
    pub winy: u8,
    pub winx: u8,
    pub wy_trigger: bool,
    pub wy_pos: i32,
    pub palbr: u8,
    pub pal0r: u8,
    pub pal1r: u8,
    pub palb: [u8; 4],
    pub pal0: [u8; 4],
    pub pal1: [u8; 4],
    pub vram: Vec<u8>,
    pub voam: Vec<u8>,
    pub vrambank: usize,
    pub cbgpal_inc: bool,
    pub cbgpal_ind: u8,
    // Boxed so the per-cycle Gpu reconstruction moves an 8-byte pointer, not the 96
    // inline palette bytes that only a palette-register write ever changes.
    pub cbgpal: Box<[[[u8; 3]; 4]; 8]>,
    pub csprit_inc: bool,
    pub csprit_ind: u8,
    pub csprit: Box<[[[u8; 3]; 4]; 8]>,
    pub data: region::Region,
    pub updated: bool,
    pub interrupt: u8,
    pub gbmode: gbmode::Gb_Mode,
    pub hblanking: bool,
    pub first_frame: bool,
}

pub fn new() -> Gpu {
    Gpu {
        mode: 0,
        modeclock: 0,
        line: 0,
        lyc: 0,
        lcd_on: false,
        win_tilemap: 0x9C00,
        win_on: false,
        tilebase: 0x8000,
        bg_tilemap: 0x9C00,
        sprite_size: 8,
        sprite_on: false,
        lcdc0: false,
        lyc_inte: false,
        m0_inte: false,
        m1_inte: false,
        m2_inte: false,
        scy: 0,
        scx: 0,
        winy: 0,
        winx: 0,
        wy_trigger: false,
        wy_pos: -1,
        palbr: 0,
        pal0r: 0,
        pal1r: 1,
        palb: [0; 4],
        pal0: [0; 4],
        pal1: [0; 4],
        vram: vec![0; VRAM_SIZE],
        voam: vec![0; VOAM_SIZE],
        vrambank: 0,
        cbgpal_inc: false,
        cbgpal_ind: 0,
        cbgpal: Box::new([[[0; 3]; 4]; 8]),
        csprit_inc: false,
        csprit_ind: 0,
        csprit: Box::new([[[0; 3]; 4]; 8]),
        data: region::new(SCREEN_W * SCREEN_H * 3),
        updated: false,
        interrupt: 0,
        gbmode: gbmode::Gb_Mode::Classic,
        hblanking: false,
        first_frame: false,
    }
}

/// Advances the PPU by `ticks` T-cycles. With the LCD off nothing moves; otherwise
/// the ticks are stepped in <=80 chunks so a mode transition is checked at least
/// once per chunk (rboy's `while` becomes recursion).
pub fn do_cycle(gpu: Gpu, ticks: u32) -> Gpu {
    match gpu.lcd_on {
        false => gpu,
        true => step_ticks(Gpu { hblanking: false, ..gpu }, ticks),
    }
}

// Recurses over the tick budget, each step advancing at most 80 T-cycles.
fn step_ticks(gpu: Gpu, ticksleft: u32) -> Gpu {
    match ticksleft {
        0 => gpu,
        _ => {
            let chunk = ticksleft.min(80);
            step_ticks(advance(gpu, chunk), ticksleft - chunk)
        }
    }
}

// One <=80-tick step: bump the mode clock, roll to the next line at 456, then pick
// the drawing mode for a visible line.
fn advance(gpu: Gpu, chunk: u32) -> Gpu {
    let clocked = Gpu { modeclock: gpu.modeclock + chunk, ..gpu };
    let lined = match clocked.modeclock >= 456 {
        true => enter_line(clocked),
        false => clocked,
    };
    match lined.line < 144 {
        true => visible_mode(lined),
        false => lined,
    }
}

// Wraps the mode clock past a full 456-tick line, advances LY (mod 154), checks the
// LYC coincidence, and enters VBlank at line 144.
fn enter_line(gpu: Gpu) -> Gpu {
    let advanced =
        check_interrupt_lyc(Gpu { modeclock: gpu.modeclock - 456, line: (gpu.line + 1) % 154, ..gpu });
    match advanced.line >= 144 && advanced.mode != 1 {
        true => change_mode(advanced, 1),
        false => advanced,
    }
}

// Selects OAM-scan (2), pixel-transfer (3), or HBlank (0) from the mode clock on a
// visible line.
fn visible_mode(gpu: Gpu) -> Gpu {
    match gpu.modeclock {
        clock if clock <= 80 => ensure_mode(gpu, 2),
        clock if clock <= 252 => ensure_mode(gpu, 3),
        _ => ensure_mode(gpu, 0),
    }
}

// Switches to `mode` only if it differs, so the mode-entry side effects fire once.
fn ensure_mode(gpu: Gpu, mode: u8) -> Gpu {
    match gpu.mode != mode {
        true => change_mode(gpu, mode),
        false => gpu,
    }
}

// Enters a PPU mode and runs its side effects: HBlank renders the line and opens the
// HDMA window; VBlank raises 0x01 and flags the frame; each mode may raise STAT.
fn change_mode(gpu: Gpu, mode: u8) -> Gpu {
    let switched = Gpu { mode, ..gpu };
    match mode {
        0 => {
            let rendered = gpu_render::render_scan(Gpu { hblanking: true, ..switched });
            let condition = rendered.m0_inte;
            raise_stat_if(rendered, condition)
        }
        1 => {
            let vblank = Gpu {
                wy_trigger: false,
                updated: true,
                first_frame: false,
                interrupt: switched.interrupt | 0x01,
                ..switched
            };
            let condition = vblank.m1_inte;
            raise_stat_if(vblank, condition)
        }
        2 => {
            let condition = switched.m2_inte;
            raise_stat_if(switched, condition)
        }
        _ => window_trigger(switched),
    }
}

// Mode 3 entry: latch the window's vertical start when the window first becomes
// active on this line. Mode 3 never raises STAT.
fn window_trigger(gpu: Gpu) -> Gpu {
    match gpu.win_on && !gpu.wy_trigger && gpu.line == gpu.winy {
        true => Gpu { wy_trigger: true, wy_pos: -1, ..gpu },
        false => gpu,
    }
}

// Raises the STAT interrupt (0x02) when the mode-entry condition holds.
fn raise_stat_if(gpu: Gpu, condition: bool) -> Gpu {
    match condition {
        true => Gpu { interrupt: gpu.interrupt | 0x02, ..gpu },
        false => gpu,
    }
}

// Raises STAT (0x02) on an LY == LYC coincidence when that interrupt is enabled.
fn check_interrupt_lyc(gpu: Gpu) -> Gpu {
    match gpu.lyc_inte && gpu.line == gpu.lyc {
        true => Gpu { interrupt: gpu.interrupt | 0x02, ..gpu },
        false => gpu,
    }
}

/// Whether the PPU is in HBlank, the window the CGB HDMA copies a row in.
pub fn may_hdma(gpu: &Gpu) -> bool {
    gpu.hblanking
}

pub fn read_byte(gpu: &Gpu, address: u16) -> u8 {
    match address {
        0x8000..=0x9FFF => gpu.vram[(gpu.vrambank * 0x2000) | (address as usize & 0x1FFF)],
        0xFE00..=0xFE9F => gpu.voam[address as usize - 0xFE00],
        _ => read_register(gpu, address),
    }
}

// The LCD control/status/scroll/palette register reads (0xFF40-0xFF6B).
fn read_register(gpu: &Gpu, address: u16) -> u8 {
    match address {
        0xFF40 => compose_lcdc(gpu),
        0xFF41 => compose_stat(gpu),
        0xFF42 => gpu.scy,
        0xFF43 => gpu.scx,
        0xFF44 => gpu.line,
        0xFF45 => gpu.lyc,
        // The DMA register (0xFF46) is write-only; rboy reads it back as 0.
        0xFF46 => 0,
        0xFF47 => gpu.palbr,
        0xFF48 => gpu.pal0r,
        0xFF49 => gpu.pal1r,
        0xFF4A => gpu.winy,
        0xFF4B => gpu.winx,
        // CGB-only registers read as open bus in DMG mode.
        0xFF4F..=0xFF6B if gpu.gbmode != gbmode::Gb_Mode::Color => 0xFF,
        0xFF4F => gpu.vrambank as u8 | 0xFE,
        0xFF68 => 0x40 | gpu.cbgpal_ind | (if gpu.cbgpal_inc { 0x80 } else { 0 }),
        0xFF69 => read_palette(&gpu.cbgpal, gpu.cbgpal_ind),
        0xFF6A => 0x40 | gpu.csprit_ind | (if gpu.csprit_inc { 0x80 } else { 0 }),
        0xFF6B => read_palette(&gpu.csprit, gpu.csprit_ind),
        _ => 0xFF,
    }
}

// A CGB palette-RAM read: the low byte gives red and the low green bits, the high
// byte the high green bits and blue (BGR555 split across two bytes).
fn read_palette(pal: &[[[u8; 3]; 4]; 8], index: u8) -> u8 {
    let palnum = (index >> 3) as usize;
    let colnum = ((index >> 1) & 0x3) as usize;
    match index & 0x01 == 0x00 {
        true => pal[palnum][colnum][0] | ((pal[palnum][colnum][1] & 0x07) << 5),
        false => ((pal[palnum][colnum][1] & 0x18) >> 3) | (pal[palnum][colnum][2] << 2),
    }
}

// Reassembles the LCDC byte (0xFF40) from the decoded control bits.
fn compose_lcdc(gpu: &Gpu) -> u8 {
    (if gpu.lcd_on { 0x80 } else { 0 })
        | (if gpu.win_tilemap == 0x9C00 { 0x40 } else { 0 })
        | (if gpu.win_on { 0x20 } else { 0 })
        | (if gpu.tilebase == 0x8000 { 0x10 } else { 0 })
        | (if gpu.bg_tilemap == 0x9C00 { 0x08 } else { 0 })
        | (if gpu.sprite_size == 16 { 0x04 } else { 0 })
        | (if gpu.sprite_on { 0x02 } else { 0 })
        | (if gpu.lcdc0 { 0x01 } else { 0 })
}

// Reassembles the STAT byte (0xFF41): bit 7 always set, the STAT-source enables, the
// LYC coincidence flag, and the current mode.
fn compose_stat(gpu: &Gpu) -> u8 {
    0x80 | (if gpu.lyc_inte { 0x40 } else { 0 })
        | (if gpu.m2_inte { 0x20 } else { 0 })
        | (if gpu.m1_inte { 0x10 } else { 0 })
        | (if gpu.m0_inte { 0x08 } else { 0 })
        | (if gpu.line == gpu.lyc { 0x04 } else { 0 })
        | gpu.mode
}

pub fn write_byte(gpu: Gpu, address: u16, value: u8) -> Gpu {
    match address {
        0x8000..=0x9FFF => {
            let index = (gpu.vrambank * 0x2000) | (address as usize & 0x1FFF);
            Gpu { vram: memory::write(&gpu.vram, index, value), ..gpu }
        }
        0xFE00..=0xFE9F => {
            Gpu { voam: memory::write(&gpu.voam, address as usize - 0xFE00, value), ..gpu }
        }
        _ => write_register(gpu, address, value),
    }
}

// The LCD register writes (0xFF40-0xFF6B).
fn write_register(gpu: Gpu, address: u16, value: u8) -> Gpu {
    match address {
        0xFF40 => write_lcdc(gpu, value),
        0xFF41 => Gpu {
            lyc_inte: value & 0x40 == 0x40,
            m2_inte: value & 0x20 == 0x20,
            m1_inte: value & 0x10 == 0x10,
            m0_inte: value & 0x08 == 0x08,
            ..gpu
        },
        0xFF42 => Gpu { scy: value, ..gpu },
        0xFF43 => Gpu { scx: value, ..gpu },
        0xFF45 => check_interrupt_lyc(Gpu { lyc: value, ..gpu }),
        0xFF46 => panic!("0xFF46 should be handled by the MMU"),
        0xFF47 => update_pal(Gpu { palbr: value, ..gpu }),
        0xFF48 => update_pal(Gpu { pal0r: value, ..gpu }),
        0xFF49 => update_pal(Gpu { pal1r: value, ..gpu }),
        0xFF4A => Gpu { winy: value, ..gpu },
        0xFF4B => Gpu { winx: value, ..gpu },
        // CGB-only registers are inert in DMG mode.
        0xFF4F..=0xFF6B if gpu.gbmode != gbmode::Gb_Mode::Color => gpu,
        0xFF4F => Gpu { vrambank: (value & 0x01) as usize, ..gpu },
        0xFF68 => Gpu { cbgpal_ind: value & 0x3F, cbgpal_inc: value & 0x80 == 0x80, ..gpu },
        0xFF69 => write_cbgpal(gpu, value),
        0xFF6A => Gpu { csprit_ind: value & 0x3F, csprit_inc: value & 0x80 == 0x80, ..gpu },
        0xFF6B => write_csprit(gpu, value),
        _ => gpu,
    }
}

// Writes the CGB background palette RAM at the current index, auto-incrementing it
// when the auto-increment bit is set.
fn write_cbgpal(gpu: Gpu, value: u8) -> Gpu {
    let cbgpal = Box::new(write_palette(*gpu.cbgpal, gpu.cbgpal_ind, value));
    let cbgpal_ind = match gpu.cbgpal_inc {
        true => (gpu.cbgpal_ind + 1) & 0x3F,
        false => gpu.cbgpal_ind,
    };
    Gpu { cbgpal, cbgpal_ind, ..gpu }
}

// Writes the CGB sprite palette RAM at the current index, auto-incrementing it when
// the auto-increment bit is set.
fn write_csprit(gpu: Gpu, value: u8) -> Gpu {
    let csprit = Box::new(write_palette(*gpu.csprit, gpu.csprit_ind, value));
    let csprit_ind = match gpu.csprit_inc {
        true => (gpu.csprit_ind + 1) & 0x3F,
        false => gpu.csprit_ind,
    };
    Gpu { csprit, csprit_ind, ..gpu }
}

// Returns the palette array with one BGR555 component byte merged into its colour.
fn write_palette(pal: [[[u8; 3]; 4]; 8], index: u8, value: u8) -> [[[u8; 3]; 4]; 8] {
    let palnum = (index >> 3) as usize;
    let colnum = ((index >> 1) & 0x03) as usize;
    let old = pal[palnum][colnum];
    let triple = match index & 0x01 == 0x00 {
        true => [value & 0x1F, (old[1] & 0x18) | (value >> 5), old[2]],
        false => [old[0], (old[1] & 0x07) | ((value & 0x3) << 3), (value >> 2) & 0x1F],
    };
    array::from_fn(|p| {
        array::from_fn(|c| match p == palnum && c == colnum {
            true => triple,
            false => pal[p][c],
        })
    })
}

// The LCDC write decodes the control bits; toggling the LCD off clears the screen
// and resets timing, toggling it on restarts at OAM scan.
fn write_lcdc(gpu: Gpu, value: u8) -> Gpu {
    let was_on = gpu.lcd_on;
    let decoded = Gpu {
        lcd_on: value & 0x80 == 0x80,
        win_tilemap: if value & 0x40 == 0x40 { 0x9C00 } else { 0x9800 },
        win_on: value & 0x20 == 0x20,
        tilebase: if value & 0x10 == 0x10 { 0x8000 } else { 0x8800 },
        bg_tilemap: if value & 0x08 == 0x08 { 0x9C00 } else { 0x9800 },
        sprite_size: if value & 0x04 == 0x04 { 16 } else { 8 },
        sprite_on: value & 0x02 == 0x02,
        lcdc0: value & 0x01 == 0x01,
        ..gpu
    };
    match (was_on, decoded.lcd_on) {
        (true, false) => clear_screen(Gpu {
            modeclock: 0,
            line: 0,
            mode: 0,
            wy_trigger: false,
            first_frame: true,
            ..decoded
        }),
        (false, true) => Gpu { modeclock: 4, ..change_mode(decoded, 2) },
        _ => decoded,
    }
}

// Blanks the framebuffer to white when the LCD switches off.
fn clear_screen(gpu: Gpu) -> Gpu {
    Gpu { data: region::filled(SCREEN_W * SCREEN_H * 3, 255), updated: true, ..gpu }
}

// Recomputes the three DMG palettes from their register bytes.
fn update_pal(gpu: Gpu) -> Gpu {
    Gpu {
        palb: monochrome_palette(gpu.palbr),
        pal0: monochrome_palette(gpu.pal0r),
        pal1: monochrome_palette(gpu.pal1r),
        ..gpu
    }
}

// The four grey levels a DMG palette register selects, brightest to darkest.
fn monochrome_palette(value: u8) -> [u8; 4] {
    [0, 1, 2, 3].map(|index| match (value >> (2 * index)) & 0x03 {
        0 => 255,
        1 => 192,
        2 => 96,
        _ => 0,
    })
}

#[cfg(test)]
mod tests {
    use crate::gpu;

    // Turns the LCD on (LCDC bit 7) so the timing machine runs.
    fn lcd_on() -> gpu::Gpu {
        gpu::write_byte(gpu::new(), 0xFF40, 0x80)
    }

    #[test]
    fn line_advances_after_a_full_scanline() {
        // A scanline is 456 T-cycles; the LCD-on reset leaves modeclock at 4.
        let ppu = gpu::do_cycle(lcd_on(), 456);
        assert_eq!(ppu.line, 1);
    }

    #[test]
    fn vblank_interrupt_raised_at_line_144() {
        // Step 144 whole scanlines to reach the VBlank line.
        let ppu = (0..144).fold(lcd_on(), |ppu, _| gpu::do_cycle(ppu, 456));
        assert_eq!(ppu.line, 144);
        assert_eq!(ppu.interrupt & 0x01, 0x01);
        assert!(ppu.updated);
    }

    #[test]
    fn dmg_palette_maps_indices_to_grey_levels() {
        // BGP 0xE4 = 11_10_01_00 maps colours 0..3 to 255,192,96,0.
        let ppu = gpu::write_byte(gpu::new(), 0xFF47, 0xE4);
        assert_eq!(ppu.palb, [255, 192, 96, 0]);
    }

    #[test]
    fn vram_write_then_read_round_trip() {
        let ppu = gpu::write_byte(gpu::new(), 0x8000, 0xAB);
        assert_eq!(gpu::read_byte(&ppu, 0x8000), 0xAB);
    }
}
