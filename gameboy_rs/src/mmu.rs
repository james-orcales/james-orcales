//! The memory bus: address decode, WRAM/HRAM, the interrupt registers, and the
//! per-cycle fan-out to the peripherals. rboy drove this through `&mut self`; here
//! reads are pure functions of the bus and every write or tick consumes the bus and
//! returns the new one. WRAM/HRAM are `Vec<u8>` (moved as a pointer when the bus is
//! threaded, unlike rboy's inline arrays), with writes paying the `memory` rebuild
//! tax. The CGB VRAM-DMA and speed-switch machinery is decoded but inert in DMG
//! (the guards route it to open bus); its transfer logic lands in M5.

use crate::gbmode;
use crate::gpu;
use crate::keypad;
use crate::mbc;
use crate::memory;
use crate::region;
use crate::serial;
use crate::sound;
use crate::timer;
use std::iter;

const WRAM_SIZE: usize = 0x8000;
const ZRAM_SIZE: usize = 0x7F;

/// The kind of CGB VRAM DMA in progress: none, general-purpose (all at once), or
/// HBlank (a row per HBlank).
#[derive(Copy, Clone, PartialEq, Eq, Debug)]
pub enum Dma_Type {
    No_Dma,
    Gdma,
    Hdma,
}

/// The whole bus: work RAM, high RAM, interrupt enable/flag, the peripherals, the
/// cartridge, and the CGB banking/DMA/speed state.
#[derive(Clone, Debug)]
pub struct Mmu {
    pub wram: region::Region,
    pub zram: Vec<u8>,
    pub hdma: Vec<u8>,
    pub inte: u8,
    pub intf: u8,
    pub serial: serial::Serial,
    pub timer: timer::Timer,
    pub keypad: keypad::Keypad,
    pub gpu: gpu::Gpu,
    pub sound: sound::Sound,
    pub hdma_status: Dma_Type,
    pub hdma_src: u16,
    pub hdma_dst: u16,
    pub hdma_len: u8,
    pub wrambank: usize,
    pub mbc: mbc::Mbc,
    pub gbmode: gbmode::Gb_Mode,
    pub gbspeed: gbmode::Gb_Speed,
    pub speed_switch_req: bool,
    pub undocumented_cgb_regs: [u8; 3],
}

/// Builds the bus for a cartridge and console mode, seeding WRAM the way rboy does
/// and writing the post-boot register values. Rejects a CGB-only cartridge asked to
/// run in Classic mode.
pub fn new(cart: mbc::Mbc, mode: gbmode::Gb_Mode) -> Result<Mmu, String> {
    let bus = Mmu {
        wram: region::from_bytes(&random_wram(42, WRAM_SIZE)),
        zram: vec![0; ZRAM_SIZE],
        hdma: vec![0; 4],
        inte: 0,
        intf: 0,
        serial: serial::new(),
        timer: timer::new(),
        keypad: keypad::new(),
        gpu: gpu::Gpu { gbmode: mode, ..gpu::new() },
        sound: sound::new_for(mode),
        hdma_status: Dma_Type::No_Dma,
        hdma_src: 0,
        hdma_dst: 0,
        hdma_len: 0xFF,
        wrambank: 1,
        mbc: cart,
        gbmode: mode,
        gbspeed: gbmode::Gb_Speed::Single,
        speed_switch_req: false,
        undocumented_cgb_regs: [0; 3],
    };
    match mode == gbmode::Gb_Mode::Classic && mbc::read_rom(&bus.mbc, 0x0143) == 0xC0 {
        true => Err("This game does not work in Classic mode".to_string()),
        false => {
            // The headless APU (rboy's too) is created after the boot register
            // writes and never sees them, so it is reset to fresh here to match.
            let initialized = set_initial(bus);
            Ok(Mmu { sound: sound::new_for(mode), ..initialized })
        }
    }
}

/// Builds the bus for CGB hardware, choosing full Color or DMG-compatibility mode
/// from the cartridge's CGB flag (header byte 0x143), the way rboy's `new_cgb` does.
pub fn new_cgb(cart: mbc::Mbc) -> Result<Mmu, String> {
    let mode = match mbc::read_rom(&cart, 0x0143) & 0x80 == 0x80 {
        true => gbmode::Gb_Mode::Color,
        false => gbmode::Gb_Mode::Color_As_Classic,
    };
    new(cart, mode)
}

// The LCG WRAM seed rboy uses, reproduced exactly so uninitialized reads match:
// each byte is bits 30..23 of the running state. `successors` replaces the loop.
fn random_wram(seed: u32, size: usize) -> Vec<u8> {
    iter::successors(Some(seed), |&x| Some(x.wrapping_mul(1103515245).wrapping_add(12345)))
        .skip(1)
        .take(size)
        .map(|x| ((x >> 23) & 0xFF) as u8)
        .collect()
}

// The post-boot I/O register values the boot ROM leaves behind, applied through the
// normal write path (sound writes are inert until the APU lands in M6).
fn set_initial(bus: Mmu) -> Mmu {
    // The exact post-boot write sequence rboy applies, including its duplicate 0xFF16.
    let writes: [(u16, u8); 31] = [
        (0xFF05, 0), (0xFF06, 0), (0xFF07, 0), (0xFF10, 0x80), (0xFF11, 0xBF),
        (0xFF12, 0xF3), (0xFF14, 0xBF), (0xFF16, 0x3F), (0xFF16, 0x3F), (0xFF17, 0),
        (0xFF19, 0xBF), (0xFF1A, 0x7F), (0xFF1B, 0xFF), (0xFF1C, 0x9F), (0xFF1E, 0xFF),
        (0xFF20, 0xFF), (0xFF21, 0), (0xFF22, 0), (0xFF23, 0xBF), (0xFF24, 0x77),
        (0xFF25, 0xF3), (0xFF26, 0xF1), (0xFF40, 0x91), (0xFF42, 0), (0xFF43, 0),
        (0xFF45, 0), (0xFF47, 0xFC), (0xFF48, 0xFF), (0xFF49, 0xFF), (0xFF4A, 0),
        (0xFF4B, 0),
    ];
    writes.iter().fold(bus, |acc, &(address, value)| write_byte(acc, address, value))
}

pub fn read_byte(mmu: &Mmu, address: u16) -> u8 {
    match address {
        0x0000..=0x7FFF => mbc::read_rom(&mmu.mbc, address),
        0x8000..=0x9FFF => gpu::read_byte(&mmu.gpu, address),
        0xA000..=0xBFFF => mbc::read_ram(&mmu.mbc, address),
        0xC000..=0xCFFF | 0xE000..=0xEFFF => region::read(&mmu.wram, address as usize & 0x0FFF),
        0xD000..=0xDFFF | 0xF000..=0xFDFF => {
            region::read(&mmu.wram, (mmu.wrambank * 0x1000) | (address as usize & 0x0FFF))
        }
        0xFE00..=0xFE9F => gpu::read_byte(&mmu.gpu, address),
        0xFEA0..=0xFEFF => 0xFF,
        0xFF00..=0xFF7F => read_io(mmu, address),
        0xFF80..=0xFFFE => mmu.zram[address as usize & 0x007F],
        0xFFFF => mmu.inte,
    }
}

// The I/O register reads (0xFF00-0xFF7F), mirroring rboy's arm order so the CGB
// guards and DMG open-bus fallbacks match byte-for-byte.
fn read_io(mmu: &Mmu, address: u16) -> u8 {
    match address {
        0xFF00 => keypad::read_byte(&mmu.keypad),
        0xFF01..=0xFF02 => serial::read_byte(&mmu.serial, address),
        0xFF04..=0xFF07 => timer::read_byte(&mmu.timer, address),
        0xFF0F => mmu.intf | 0b1110_0000,
        0xFF10..=0xFF3F => sound::read_byte(&mmu.sound, address),
        0xFF4D | 0xFF4F | 0xFF51..=0xFF55 | 0xFF6C | 0xFF70
            if mmu.gbmode != gbmode::Gb_Mode::Color =>
        {
            0xFF
        }
        0xFF72..=0xFF73 | 0xFF75..=0xFF77 if mmu.gbmode == gbmode::Gb_Mode::Classic => 0xFF,
        0xFF4D => read_speed(mmu),
        0xFF40..=0xFF4F => gpu::read_byte(&mmu.gpu, address),
        0xFF51..=0xFF55 => hdma_read(mmu, address),
        0xFF68..=0xFF6B => gpu::read_byte(&mmu.gpu, address),
        0xFF70 => mmu.wrambank as u8,
        0xFF72..=0xFF73 => mmu.undocumented_cgb_regs[address as usize - 0xFF72],
        0xFF75 => mmu.undocumented_cgb_regs[2] | 0b1000_1111,
        0xFF76..=0xFF77 => 0x00,
        _ => 0xFF,
    }
}

pub fn read_wide(mmu: &Mmu, address: u16) -> u16 {
    (read_byte(mmu, address) as u16) | ((read_byte(mmu, address.wrapping_add(1)) as u16) << 8)
}

pub fn write_byte(mmu: Mmu, address: u16, value: u8) -> Mmu {
    match address {
        0x0000..=0x7FFF => Mmu { mbc: mbc::write_rom(mmu.mbc, address, value), ..mmu },
        0x8000..=0x9FFF => Mmu { gpu: gpu::write_byte(mmu.gpu, address, value), ..mmu },
        0xA000..=0xBFFF => Mmu { mbc: mbc::write_ram(mmu.mbc, address, value), ..mmu },
        0xC000..=0xCFFF | 0xE000..=0xEFFF => {
            Mmu { wram: region::write(mmu.wram, address as usize & 0x0FFF, value), ..mmu }
        }
        0xD000..=0xDFFF | 0xF000..=0xFDFF => {
            let index = (mmu.wrambank * 0x1000) | (address as usize & 0x0FFF);
            Mmu { wram: region::write(mmu.wram, index, value), ..mmu }
        }
        0xFE00..=0xFE9F => Mmu { gpu: gpu::write_byte(mmu.gpu, address, value), ..mmu },
        0xFEA0..=0xFEFF => mmu,
        0xFF00..=0xFF7F => write_io(mmu, address, value),
        0xFF80..=0xFFFE => {
            Mmu { zram: memory::write(&mmu.zram, address as usize & 0x007F, value), ..mmu }
        }
        0xFFFF => Mmu { inte: value, ..mmu },
    }
}

// The I/O register writes (0xFF00-0xFF7F), mirroring rboy's arm order.
fn write_io(mmu: Mmu, address: u16, value: u8) -> Mmu {
    match address {
        0xFF00 => Mmu { keypad: keypad::write_byte(mmu.keypad, value), ..mmu },
        0xFF01..=0xFF02 => Mmu { serial: serial::write_byte(mmu.serial, address, value), ..mmu },
        0xFF04..=0xFF07 => Mmu { timer: timer::write_byte(mmu.timer, address, value), ..mmu },
        0xFF10..=0xFF3F => Mmu { sound: sound::write_byte(mmu.sound, address, value), ..mmu },
        0xFF46 => oamdma(mmu, value),
        0xFF4D | 0xFF4F | 0xFF51..=0xFF55 | 0xFF6C | 0xFF70 | 0xFF76..=0xFF77
            if mmu.gbmode != gbmode::Gb_Mode::Color =>
        {
            mmu
        }
        0xFF72..=0xFF73 | 0xFF75..=0xFF77 if mmu.gbmode == gbmode::Gb_Mode::Classic => mmu,
        0xFF4D => request_speed_switch(mmu, value),
        0xFF40..=0xFF4F => Mmu { gpu: gpu::write_byte(mmu.gpu, address, value), ..mmu },
        0xFF51..=0xFF55 => hdma_write(mmu, address, value),
        0xFF68..=0xFF6B => Mmu { gpu: gpu::write_byte(mmu.gpu, address, value), ..mmu },
        0xFF0F => Mmu { intf: value, ..mmu },
        0xFF70 => Mmu { wrambank: wrambank_select(value), ..mmu },
        0xFF72 => Mmu { undocumented_cgb_regs: set_undoc(mmu.undocumented_cgb_regs, 0, value), ..mmu },
        0xFF73 => Mmu { undocumented_cgb_regs: set_undoc(mmu.undocumented_cgb_regs, 1, value), ..mmu },
        0xFF75 => Mmu { undocumented_cgb_regs: set_undoc(mmu.undocumented_cgb_regs, 2, value), ..mmu },
        _ => mmu,
    }
}

pub fn write_wide(mmu: Mmu, address: u16, value: u16) -> Mmu {
    let low = write_byte(mmu, address, (value & 0xFF) as u8);
    write_byte(low, address.wrapping_add(1), (value >> 8) as u8)
}

/// Advances every peripheral by `ticks`, folding their raised interrupts into IF and
/// clearing each source, and returns the bus plus the PPU tick count the caller
/// accumulates toward a frame.
pub fn do_cycle(mmu: Mmu, ticks: u32) -> (Mmu, u32) {
    let cpudivider = mmu.gbspeed as u32;
    let (dmad, vramticks) = perform_vramdma(mmu);
    let gputicks = ticks / cpudivider + vramticks;
    let cputicks = ticks + vramticks * cpudivider;
    let timed = tick_timer(dmad, cputicks);
    let keyed = collect_keypad(timed);
    let drawn = tick_gpu(keyed, gputicks);
    let sounded = Mmu { sound: sound::do_cycle(drawn.sound, gputicks), ..drawn };
    (collect_serial(sounded), gputicks)
}

// Ticks the timer and merges its overflow interrupt into IF.
fn tick_timer(mmu: Mmu, cputicks: u32) -> Mmu {
    let timer = timer::do_cycle(mmu.timer, cputicks);
    Mmu { intf: mmu.intf | timer.interrupt, timer: timer::Timer { interrupt: 0, ..timer }, ..mmu }
}

// Merges any pending joypad interrupt into IF.
fn collect_keypad(mmu: Mmu) -> Mmu {
    Mmu {
        intf: mmu.intf | mmu.keypad.interrupt,
        keypad: keypad::Keypad { interrupt: 0, ..mmu.keypad },
        ..mmu
    }
}

// Ticks the PPU and merges its VBlank/STAT interrupts into IF.
fn tick_gpu(mmu: Mmu, gputicks: u32) -> Mmu {
    let gpu = gpu::do_cycle(mmu.gpu, gputicks);
    Mmu { intf: mmu.intf | gpu.interrupt, gpu: gpu::Gpu { interrupt: 0, ..gpu }, ..mmu }
}

// Merges any pending serial interrupt into IF.
fn collect_serial(mmu: Mmu) -> Mmu {
    Mmu {
        intf: mmu.intf | mmu.serial.interrupt,
        serial: serial::Serial { interrupt: 0, ..mmu.serial },
        ..mmu
    }
}

// OAM DMA: copy 0xA0 bytes from the page `value << 8` into OAM. rboy did 160 byte
// writes; here the source block is read once and spliced into OAM in a single
// rebuild (the batched tax path), which yields the identical OAM contents.
fn oamdma(mmu: Mmu, value: u8) -> Mmu {
    let base = (value as u16) << 8;
    let block: Vec<u8> = (0..0xA0u16).map(|i| read_byte(&mmu, base + i)).collect();
    let voam = memory::write_slice(&mmu.gpu.voam, 0, &block);
    Mmu { gpu: gpu::Gpu { voam, ..mmu.gpu }, ..mmu }
}

// The KEY1 speed register read (CGB): current speed in bit 7, pending switch in bit 0.
fn read_speed(mmu: &Mmu) -> u8 {
    0b0111_1110
        | (if mmu.gbspeed == gbmode::Gb_Speed::Double { 0x80 } else { 0 })
        | (if mmu.speed_switch_req { 1 } else { 0 })
}

// A KEY1 write requesting a speed switch (CGB); the switch itself happens on STOP.
fn request_speed_switch(mmu: Mmu, value: u8) -> Mmu {
    match value & 0x1 == 0x1 {
        true => Mmu { speed_switch_req: true, ..mmu },
        false => mmu,
    }
}

// The HDMA registers read: source/dest bytes, and at 0xFF55 the remaining length
// plus an active flag in bit 7.
fn hdma_read(mmu: &Mmu, address: u16) -> u8 {
    match address {
        0xFF51..=0xFF54 => mmu.hdma[(address - 0xFF51) as usize],
        0xFF55 => mmu.hdma_len | if mmu.hdma_status == Dma_Type::No_Dma { 0x80 } else { 0 },
        _ => panic!("Address {:04X} is not an HDMA register", address),
    }
}

/// Injects the current wall-clock time into the cartridge (for an MBC3 RTC); the
/// composition root supplies it so the bus reads no clock itself.
pub fn set_clock(mmu: Mmu, now: u64) -> Mmu {
    Mmu { mbc: mbc::set_clock(mmu.mbc, now), ..mmu }
}

/// Connects the Game Boy Printer to the serial port, so a printing ROM's output is
/// captured as images instead of the raw byte stream.
pub fn attach_printer(mmu: Mmu) -> Mmu {
    Mmu { serial: serial::attach_printer(mmu.serial), ..mmu }
}

/// Toggles CGB double speed when a switch was requested (executed by STOP).
pub fn switch_speed(mmu: Mmu) -> Mmu {
    match mmu.speed_switch_req {
        true => Mmu { gbspeed: toggle_speed(mmu.gbspeed), speed_switch_req: false, ..mmu },
        false => Mmu { speed_switch_req: false, ..mmu },
    }
}

fn toggle_speed(speed: gbmode::Gb_Speed) -> gbmode::Gb_Speed {
    match speed {
        gbmode::Gb_Speed::Double => gbmode::Gb_Speed::Single,
        gbmode::Gb_Speed::Single => gbmode::Gb_Speed::Double,
    }
}

// The HDMA registers write and transfer kickoff (CGB). DMG never reaches this
// (0xFF51-0xFF55 writes are guarded to inert above).
fn hdma_write(mmu: Mmu, address: u16, value: u8) -> Mmu {
    match address {
        0xFF51 => Mmu { hdma: memory::write(&mmu.hdma, 0, value), ..mmu },
        0xFF52 => Mmu { hdma: memory::write(&mmu.hdma, 1, value & 0xF0), ..mmu },
        0xFF53 => Mmu { hdma: memory::write(&mmu.hdma, 2, value & 0x1F), ..mmu },
        0xFF54 => Mmu { hdma: memory::write(&mmu.hdma, 3, value & 0xF0), ..mmu },
        0xFF55 => hdma_start(mmu, value),
        _ => panic!("Address {:04X} is not an HDMA register", address),
    }
}

// A 0xFF55 write: cancel an active HBlank DMA when bit 7 is clear, otherwise latch
// the source/destination and start a general-purpose (bit 7 clear) or HBlank DMA.
fn hdma_start(mmu: Mmu, value: u8) -> Mmu {
    match mmu.hdma_status == Dma_Type::Hdma {
        true => match value & 0x80 == 0 {
            true => Mmu { hdma_status: Dma_Type::No_Dma, ..mmu },
            false => mmu,
        },
        false => {
            let src = ((mmu.hdma[0] as u16) << 8) | (mmu.hdma[1] as u16);
            let dst = ((mmu.hdma[2] as u16) << 8) | (mmu.hdma[3] as u16) | 0x8000;
            match src <= 0x7FF0 || (0xA000..=0xDFF0).contains(&src) {
                false => panic!("HDMA transfer with illegal start address {:04X}", src),
                true => Mmu {
                    hdma_src: src,
                    hdma_dst: dst,
                    hdma_len: value & 0x7F,
                    hdma_status: match value & 0x80 == 0x80 {
                        true => Dma_Type::Hdma,
                        false => Dma_Type::Gdma,
                    },
                    ..mmu
                },
            }
        }
    }
}

// The CGB VRAM DMA step run at the top of each cycle. Inert in DMG (status is always
// No_Dma).
fn perform_vramdma(mmu: Mmu) -> (Mmu, u32) {
    match mmu.hdma_status {
        Dma_Type::No_Dma => (mmu, 0),
        Dma_Type::Gdma => perform_gdma(mmu),
        Dma_Type::Hdma => perform_hdma(mmu),
    }
}

// General-purpose DMA copies every remaining row at once.
fn perform_gdma(mmu: Mmu) -> (Mmu, u32) {
    let len = mmu.hdma_len as u32 + 1;
    let copied = (0..len).fold(mmu, |acc, _| perform_vramdma_row(acc));
    (Mmu { hdma_status: Dma_Type::No_Dma, ..copied }, len * 8)
}

// HBlank DMA copies one row per HBlank, ending when the length wraps past zero.
fn perform_hdma(mmu: Mmu) -> (Mmu, u32) {
    match gpu::may_hdma(&mmu.gpu) {
        false => (mmu, 0),
        true => {
            let copied = perform_vramdma_row(mmu);
            match copied.hdma_len == 0x7F {
                true => (Mmu { hdma_status: Dma_Type::No_Dma, ..copied }, 8),
                false => (copied, 8),
            }
        }
    }
}

// Copies one 16-byte row from the source into VRAM and advances the pointers.
fn perform_vramdma_row(mmu: Mmu) -> Mmu {
    let block: Vec<u8> = (0..0x10u16).map(|j| read_byte(&mmu, mmu.hdma_src + j)).collect();
    let index = (mmu.gpu.vrambank * 0x2000) | ((mmu.hdma_dst as usize) & 0x1FFF);
    let vram = memory::write_slice(&mmu.gpu.vram, index, &block);
    let hdma_len = match mmu.hdma_len {
        0 => 0x7F,
        n => n - 1,
    };
    Mmu {
        gpu: gpu::Gpu { vram, ..mmu.gpu },
        hdma_src: mmu.hdma_src + 0x10,
        hdma_dst: mmu.hdma_dst + 0x10,
        hdma_len,
        ..mmu
    }
}

// The WRAM bank a 0xFF70 write selects, treating 0 as bank 1.
fn wrambank_select(value: u8) -> usize {
    match value & 0x7 {
        0 => 1,
        n => n as usize,
    }
}

// Returns the undocumented-register array with one entry replaced.
fn set_undoc(regs: [u8; 3], index: usize, value: u8) -> [u8; 3] {
    [0, 1, 2].map(|i| if i == index { value } else { regs[i] })
}

#[cfg(test)]
mod tests {
    use crate::gbmode;
    use crate::mbc;
    use crate::mmu;

    // A blank MBC0 ROM sized so the header checks pass.
    fn dmg_bus() -> mmu::Mmu {
        let rom = vec![0u8; 0x8000];
        mmu::new(mbc::get_mbc(rom, true).unwrap(), gbmode::Gb_Mode::Classic).unwrap()
    }

    #[test]
    fn work_ram_round_trips_and_echoes() {
        let bus = mmu::write_byte(dmg_bus(), 0xC000, 0x5A);
        assert_eq!(mmu::read_byte(&bus, 0xC000), 0x5A);
        // 0xE000 echoes the low work-RAM bank.
        assert_eq!(mmu::read_byte(&bus, 0xE000), 0x5A);
    }

    #[test]
    fn high_ram_round_trips() {
        let bus = mmu::write_byte(dmg_bus(), 0xFF80, 0x99);
        assert_eq!(mmu::read_byte(&bus, 0xFF80), 0x99);
    }

    #[test]
    fn interrupt_enable_register_round_trips() {
        let bus = mmu::write_byte(dmg_bus(), 0xFFFF, 0x1F);
        assert_eq!(mmu::read_byte(&bus, 0xFFFF), 0x1F);
    }

    #[test]
    fn oam_dma_copies_a_page_into_oam() {
        // Seed a work-RAM page, trigger DMA from it, then read OAM back.
        let bus = (0..0xA0u16).fold(dmg_bus(), |acc, i| mmu::write_byte(acc, 0xC000 + i, i as u8));
        let bus = mmu::write_byte(bus, 0xFF46, 0xC0);
        assert_eq!(mmu::read_byte(&bus, 0xFE00), 0x00);
        assert_eq!(mmu::read_byte(&bus, 0xFE9F), 0x9F);
    }
}
