//! Scanline rendering for both DMG and CGB, ported from rboy's `renderscan` /
//! `draw_bg` / `draw_sprites`. rboy wrote pixels one at a time through `&mut self`;
//! here a whole scanline is built as a value — a `Vec` of (RGB triple, priority) —
//! then spliced into the framebuffer once. A pixel is an `[u8; 3]` so the monochrome
//! path (three equal grey bytes) and the colour path (a Gambatte-corrected CGB
//! colour) share one representation. rboy's in-place sprite sort becomes a `BTreeMap`
//! keyed by reversed X (DMG) or reversed OAM index (CGB), reproducing its exact
//! front-to-back draw order without mutation.

use crate::gbmode;
use crate::gpu;
use crate::region;
use std::cmp;
use std::collections;

/// A background pixel's relationship to sprites: colour 0 (always behind), the CGB
/// background-priority flag, or a normal opaque pixel.
#[derive(Copy, Clone, PartialEq, Eq, Debug)]
pub enum Prio_Type {
    Color0,
    Prio_Flag,
    Normal,
}

// The CGB tilemap attribute byte, or the DMG defaults (no attributes).
#[derive(Copy, Clone)]
struct Attr {
    pub palnr: usize,
    pub vram1: bool,
    pub xflip: bool,
    pub yflip: bool,
    pub prio: bool,
}

/// Renders the current scanline into the framebuffer, advancing the window's line
/// counter. The first frame after the LCD switches on is not drawn (rboy quirk).
pub fn render_scan(gpu: gpu::Gpu) -> gpu::Gpu {
    match gpu.first_frame {
        true => gpu,
        false => draw_line(gpu),
    }
}

// Builds the background then overlays sprites, and splices the finished scanline in.
fn draw_line(gpu: gpu::Gpu) -> gpu::Gpu {
    let (winy_line, wy_pos) = window_line(&gpu);
    let advanced = gpu::Gpu { wy_pos, ..gpu };
    let row = sprite_row(&advanced, bg_row(&advanced, winy_line));
    let bytes = expand_row(&row);
    let base = advanced.line as usize * gpu::SCREEN_W * 3;
    gpu::Gpu { data: region::write_slice(advanced.data, base, &bytes), ..advanced }
}

// The window's source line for this scanline (or -1 when inactive) and the updated
// window line counter, which advances once per scanline the window is active.
fn window_line(gpu: &gpu::Gpu) -> (i32, i32) {
    let wx_trigger = gpu.winx <= 166;
    match gpu.win_on && gpu.wy_trigger && wx_trigger {
        true => (gpu.wy_pos + 1, gpu.wy_pos + 1),
        false => (-1, gpu.wy_pos),
    }
}

// The background/window colour and priority for every column of the scanline.
fn bg_row(gpu: &gpu::Gpu, winy_line: i32) -> Vec<([u8; 3], Prio_Type)> {
    (0..gpu::SCREEN_W).map(|x| bg_pixel(gpu, x, winy_line)).collect()
}

// One column: the window pixel when the window covers it, else the background pixel.
fn bg_pixel(gpu: &gpu::Gpu, x: usize, winy_line: i32) -> ([u8; 3], Prio_Type) {
    let winx = -((gpu.winx as i32) - 7) + (x as i32);
    match winy_line >= 0 && winx >= 0 {
        true => tile_pixel(gpu, gpu.win_tilemap, (winy_line as u16) >> 3 & 31, (winx as u16) >> 3, (winy_line as u16) & 7, (winx as u8) & 7),
        false => bg_only_pixel(gpu, x),
    }
}

// The background pixel for a column, or white when the background is disabled (CGB
// always draws the background).
fn bg_only_pixel(gpu: &gpu::Gpu, x: usize) -> ([u8; 3], Prio_Type) {
    match gpu.gbmode == gbmode::Gb_Mode::Color || gpu.lcdc0 {
        false => ([255; 3], Prio_Type::Normal),
        true => {
            let bgx = gpu.scx as u32 + x as u32;
            let bgy = gpu.scy.wrapping_add(gpu.line);
            tile_pixel(gpu, gpu.bg_tilemap, (bgy as u16) >> 3 & 31, (bgx as u16) >> 3 & 31, (bgy as u16) & 7, (bgx as u8) & 7)
        }
    }
}

// One background/window pixel: look up the tile and its CGB attributes, fetch the
// bit-planes (honouring the bank and Y flip), and read the palette.
fn tile_pixel(gpu: &gpu::Gpu, tilemap: u16, tiley: u16, tilex: u16, pixely: u16, pixelx: u8) -> ([u8; 3], Prio_Type) {
    let mapaddr = tilemap + tiley * 32 + tilex;
    let attr = tile_attributes(gpu, mapaddr);
    let base = tile_base_address(gpu, read_vram0(gpu, mapaddr), pixely, attr.yflip);
    let (low, high) = tile_bytes(gpu, base, attr.vram1);
    let bit = match attr.xflip {
        true => pixelx,
        false => 7 - pixelx,
    } as u32;
    let colnr = color_number(low, high, bit);
    (bg_color(gpu, attr.palnr, colnr), bg_priority(colnr, attr.prio))
}

// The tilemap attribute byte from VRAM bank 1 in CGB mode, or the DMG defaults.
fn tile_attributes(gpu: &gpu::Gpu, mapaddr: u16) -> Attr {
    match gpu.gbmode == gbmode::Gb_Mode::Color {
        false => Attr { palnr: 0, vram1: false, xflip: false, yflip: false, prio: false },
        true => {
            let flags = read_vram1(gpu, mapaddr) as usize;
            Attr {
                palnr: flags & 0x07,
                vram1: flags & (1 << 3) != 0,
                xflip: flags & (1 << 5) != 0,
                yflip: flags & (1 << 6) != 0,
                prio: flags & (1 << 7) != 0,
            }
        }
    }
}

// The address of the tile's bit-plane pair for this row, honouring the Y flip and the
// signed vs unsigned tile numbering.
fn tile_base_address(gpu: &gpu::Gpu, tilenr: u8, pixely: u16, yflip: bool) -> u16 {
    let offset = match gpu.tilebase == 0x8000 {
        true => tilenr as u16,
        false => (tilenr as i8 as i16 + 128) as u16,
    };
    let tileaddress = gpu.tilebase + offset * 16;
    match yflip {
        true => tileaddress + (14 - pixely * 2),
        false => tileaddress + pixely * 2,
    }
}

// The two bit-planes at an address from VRAM bank 0 or bank 1.
fn tile_bytes(gpu: &gpu::Gpu, base: u16, vram1: bool) -> (u8, u8) {
    match vram1 {
        true => (read_vram1(gpu, base), read_vram1(gpu, base + 1)),
        false => (read_vram0(gpu, base), read_vram0(gpu, base + 1)),
    }
}

// A background pixel's priority tag: colour 0 is always behind sprites.
fn bg_priority(colnr: usize, prio: bool) -> Prio_Type {
    match colnr == 0 {
        true => Prio_Type::Color0,
        false => match prio {
            true => Prio_Type::Prio_Flag,
            false => Prio_Type::Normal,
        },
    }
}

// A background pixel's RGB: the CGB palette (corrected) or the DMG grey level.
fn bg_color(gpu: &gpu::Gpu, palnr: usize, colnr: usize) -> [u8; 3] {
    match gpu.gbmode == gbmode::Gb_Mode::Color {
        true => cgb_rgb(gpu.cbgpal[palnr][colnr]),
        false => [gpu.palb[colnr]; 3],
    }
}

// Gambatte's BGR555-to-RGB888 correction, matching rboy's `setrgb`.
fn cgb_rgb(color: [u8; 3]) -> [u8; 3] {
    let (r, g, b) = (color[0] as u32, color[1] as u32, color[2] as u32);
    [((r * 13 + g * 2 + b) >> 1) as u8, ((g * 3 + b) << 1) as u8, ((r * 3 + g * 2 + b * 11) >> 1) as u8]
}

// Overlays the visible sprites onto the background row (nothing when sprites are off).
fn sprite_row(gpu: &gpu::Gpu, row: Vec<([u8; 3], Prio_Type)>) -> Vec<([u8; 3], Prio_Type)> {
    match gpu.sprite_on {
        false => row,
        true => collect_sprites(gpu).into_iter().fold(row, |row, sprite| blit_sprite(gpu, row, sprite)),
    }
}

// The up-to-ten sprites on this line, in rboy's front-to-back draw order. The
// BTreeMap key reproduces the DMG (X then index) or CGB (index only) sort without a
// `&mut` in-place sort.
fn collect_sprites(gpu: &gpu::Gpu) -> Vec<(i32, i32, usize)> {
    let line = gpu.line as i32;
    let size = gpu.sprite_size as i32;
    let by_index = gpu.gbmode == gbmode::Gb_Mode::Color;
    let visible = (0..40usize).filter_map(|i| {
        let sprite_y = gpu.voam[i * 4] as i32 - 16;
        match line >= sprite_y && line < sprite_y + size {
            true => Some((gpu.voam[i * 4 + 1] as i32 - 8, sprite_y, i)),
            false => None,
        }
    });
    let ordered: collections::BTreeMap<(cmp::Reverse<i32>, cmp::Reverse<usize>), (i32, i32, usize)> = visible
        .take(10)
        .map(|s| ((cmp::Reverse(if by_index { 0 } else { s.0 }), cmp::Reverse(s.2)), s))
        .collect();
    ordered.into_values().collect()
}

// Draws one 8-pixel-wide sprite row onto the scanline.
fn blit_sprite(gpu: &gpu::Gpu, row: Vec<([u8; 3], Prio_Type)>, sprite: (i32, i32, usize)) -> Vec<([u8; 3], Prio_Type)> {
    let (sprite_x, sprite_y, i) = sprite;
    match sprite_x < -7 || sprite_x >= gpu::SCREEN_W as i32 {
        true => row,
        false => {
            let flags = gpu.voam[i * 4 + 3];
            let (low, high) = sprite_bytes(gpu, i, sprite_y, flags);
            (0..8i32).fold(row, |row, x| {
                let bit = match flags & 0x20 != 0 {
                    true => x as u32,
                    false => (7 - x) as u32,
                };
                blit_pixel(gpu, row, sprite_x + x, color_number(low, high, bit), flags)
            })
        }
    }
}

// The two bit-planes for a sprite's slice of this scanline, honouring the Y flip and
// (in CGB) the tile's VRAM bank.
fn sprite_bytes(gpu: &gpu::Gpu, i: usize, sprite_y: i32, flags: u8) -> (u8, u8) {
    let size = gpu.sprite_size as i32;
    let tiley = match flags & 0x40 != 0 {
        true => (size - 1 - (gpu.line as i32 - sprite_y)) as u16,
        false => (gpu.line as i32 - sprite_y) as u16,
    };
    let tilenum = (gpu.voam[i * 4 + 2] & (if size == 16 { 0xFE } else { 0xFF })) as u16;
    let address = 0x8000 + tilenum * 16 + tiley * 2;
    tile_bytes(gpu, address, gpu.gbmode == gbmode::Gb_Mode::Color && flags & (1 << 3) != 0)
}

// Writes one sprite pixel, skipping transparent and priority-blocked pixels.
fn blit_pixel(gpu: &gpu::Gpu, row: Vec<([u8; 3], Prio_Type)>, column: i32, colnr: usize, flags: u8) -> Vec<([u8; 3], Prio_Type)> {
    let onscreen = column >= 0 && column < gpu::SCREEN_W as i32;
    match !onscreen || colnr == 0 || blocked(gpu, &row, column, flags) {
        true => row,
        false => set_pixel(row, column as usize, sprite_color(gpu, flags, colnr)),
    }
}

// Whether a sprite pixel is hidden by the background, per the DMG or CGB priority
// rules.
fn blocked(gpu: &gpu::Gpu, row: &[([u8; 3], Prio_Type)], column: i32, flags: u8) -> bool {
    let prio = row[column as usize].1;
    let belowbg = flags & 0x80 != 0;
    match gpu.gbmode == gbmode::Gb_Mode::Color {
        true => gpu.lcdc0 && (prio == Prio_Type::Prio_Flag || (belowbg && prio != Prio_Type::Color0)),
        false => belowbg && prio != Prio_Type::Color0,
    }
}

// A sprite pixel's RGB: the CGB sprite palette (corrected) or DMG OBP0/OBP1 grey.
fn sprite_color(gpu: &gpu::Gpu, flags: u8, colnr: usize) -> [u8; 3] {
    match gpu.gbmode == gbmode::Gb_Mode::Color {
        true => cgb_rgb(gpu.csprit[(flags & 0x07) as usize][colnr]),
        false => match flags & 0x10 != 0 {
            true => [gpu.pal1[colnr]; 3],
            false => [gpu.pal0[colnr]; 3],
        },
    }
}

// Replaces one column's colour, keeping its priority tag.
fn set_pixel(row: Vec<([u8; 3], Prio_Type)>, index: usize, color: [u8; 3]) -> Vec<([u8; 3], Prio_Type)> {
    row.iter()
        .enumerate()
        .map(|(j, &(existing, prio))| match j == index {
            true => (color, prio),
            false => (existing, prio),
        })
        .collect()
}

// The 2-bit colour number from two bit-planes at a given bit position.
fn color_number(low: u8, high: u8, bit: u32) -> usize {
    (if low & (1 << bit) != 0 { 1 } else { 0 }) | (if high & (1 << bit) != 0 { 2 } else { 0 })
}

// A read from VRAM bank 0.
fn read_vram0(gpu: &gpu::Gpu, address: u16) -> u8 {
    gpu.vram[address as usize & 0x1FFF]
}

// A read from VRAM bank 1 (CGB tile attributes and bank-1 tiles).
fn read_vram1(gpu: &gpu::Gpu, address: u16) -> u8 {
    gpu.vram[0x2000 + (address as usize & 0x1FFF)]
}

// Flattens a 160-entry RGB row into the framebuffer's bytes.
fn expand_row(row: &[([u8; 3], Prio_Type)]) -> Vec<u8> {
    row.iter().flat_map(|&(rgb, _)| rgb).collect()
}
