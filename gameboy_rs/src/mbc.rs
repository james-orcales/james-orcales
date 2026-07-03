//! The cartridge and its memory-bank controller, replacing rboy's `Box<dyn MBC>`
//! trait object with an enum plus free functions that `match`. Each variant owns its
//! ROM, RAM, and bank registers as a value: a read is a pure function of the
//! cartridge, and a write consumes the cartridge and returns the new one. The
//! banking arithmetic is ported directly from rboy's per-MBC `readrom`/`writerom`.
//! rboy's `&'static str` error type can't be returned here (the dialect bans
//! reference returns), so load errors are owned `String`s.

use crate::memory;

/// A cartridge and its bank controller. rboy dispatched these through a trait; here
/// the variant is the tag and behavior lives in the free functions below.
#[derive(Clone, Debug)]
pub enum Mbc {
    Mbc0(Mbc0),
    Mbc1(Mbc1),
    Mbc2(Mbc2),
    Mbc3(Mbc3),
    Mbc5(Mbc5),
}

/// A rom-only cartridge: 32 KiB mapped straight through, no banking, no RAM.
#[derive(Clone, Debug)]
pub struct Mbc0 {
    pub rom: Vec<u8>,
}

/// An MBC1 cartridge: up to 2 MiB ROM and 32 KiB RAM behind 5-bit/2-bit bank
/// registers and a mode select.
#[derive(Clone, Debug)]
pub struct Mbc1 {
    pub rom: Vec<u8>,
    pub ram: Vec<u8>,
    pub ram_on: bool,
    pub ram_updated: bool,
    pub banking_mode: u8,
    pub rombank: usize,
    pub rambank: usize,
    pub has_battery: bool,
    pub rombanks: usize,
    pub rambanks: usize,
}

/// An MBC2 cartridge: built-in 512 x 4-bit RAM, a single 4-bit ROM-bank register.
#[derive(Clone, Debug)]
pub struct Mbc2 {
    pub rom: Vec<u8>,
    pub ram: Vec<u8>,
    pub ram_on: bool,
    pub ram_updated: bool,
    pub rombank: usize,
    pub has_battery: bool,
    pub rombanks: usize,
}

/// An MBC3 cartridge: up to 2 MiB ROM, 32 KiB RAM, and an optional real-time clock
/// (driven here by a fixed injected clock for determinism).
#[derive(Clone, Debug)]
pub struct Mbc3 {
    pub rom: Vec<u8>,
    pub ram: Vec<u8>,
    pub rombank: usize,
    pub rambank: usize,
    pub rambanks: usize,
    pub selectrtc: bool,
    pub ram_on: bool,
    pub ram_updated: bool,
    pub has_battery: bool,
    pub rtc_ram: [u8; 5],
    pub rtc_ram_latch: [u8; 5],
    pub rtc_zero: Option<u64>,
    // The current wall-clock time in Unix seconds, injected by the composition root
    // (real time in production, a fixed value for a deterministic run). The core
    // never reads the clock itself, so it stays a pure function of this input.
    pub now: u64,
}

/// An MBC5 cartridge: up to 8 MiB ROM (9-bit bank) and 128 KiB RAM.
#[derive(Clone, Debug)]
pub struct Mbc5 {
    pub rom: Vec<u8>,
    pub ram: Vec<u8>,
    pub rombank: usize,
    pub rambank: usize,
    pub ram_on: bool,
    pub ram_updated: bool,
    pub has_battery: bool,
    pub rombanks: usize,
    pub rambanks: usize,
}

/// Builds the cartridge for a ROM image, dispatching on the header's cartridge-type
/// byte (0x147). Mirrors rboy's `get_mbc`.
pub fn get_mbc(data: Vec<u8>, skip_checksum: bool) -> Result<Mbc, String> {
    if data.len() < 0x150 {
        return Err("Rom size to small".to_string());
    }
    if !skip_checksum {
        check_checksum(&data)?;
    }
    match data[0x147] {
        0x00 => Ok(Mbc::Mbc0(Mbc0 { rom: data })),
        0x01..=0x03 => Ok(Mbc::Mbc1(new_mbc1(data))),
        0x05..=0x06 => Ok(Mbc::Mbc2(new_mbc2(data))),
        0x0F..=0x13 => Ok(Mbc::Mbc3(new_mbc3(data))),
        0x19..=0x1E => Ok(Mbc::Mbc5(new_mbc5(data))),
        _ => Err("Unsupported MBC type".to_string()),
    }
}

// The header checksum: subtract each title-region byte and one, running byte by
// byte, and compare against 0x14D.
fn check_checksum(data: &[u8]) -> Result<(), String> {
    let value = (0x134..0x14D).fold(0u8, |sum, i| sum.wrapping_sub(data[i]).wrapping_sub(1));
    match data[0x14D] == value {
        true => Ok(()),
        false => Err("Cartridge checksum is invalid".to_string()),
    }
}

// The MBC1 initial state: battery and RAM size come from the header, ROM bank
// starts at 1, RAM starts disabled.
fn new_mbc1(data: Vec<u8>) -> Mbc1 {
    let (has_battery, rambanks) = match data[0x147] {
        0x03 => (true, ram_banks(data[0x149])),
        0x02 => (false, ram_banks(data[0x149])),
        _ => (false, 0),
    };
    let rombanks = rom_banks(data[0x148]);
    Mbc1 {
        rom: data,
        ram: vec![0u8; rambanks * 0x2000],
        ram_on: false,
        ram_updated: false,
        banking_mode: 0,
        rombank: 1,
        rambank: 0,
        has_battery,
        rombanks,
        rambanks,
    }
}

// RAM bank count from the header's ram-size code (0x149), in whole 8 KiB banks.
fn ram_banks(code: u8) -> usize {
    match code {
        1 | 2 => 1,
        3 => 4,
        4 => 16,
        5 => 8,
        _ => 0,
    }
}

// ROM bank count from the header's rom-size code (0x148): 2 << code up to code 8.
fn rom_banks(code: u8) -> usize {
    match code <= 8 {
        true => 2 << code,
        false => 0,
    }
}

/// Reads a ROM byte (0x0000-0x7FFF) through the active banking.
pub fn read_rom(mbc: &Mbc, address: u16) -> u8 {
    match mbc {
        Mbc::Mbc0(cart) => cart.rom[address as usize],
        Mbc::Mbc1(cart) => mbc1_read_rom(cart, address),
        Mbc::Mbc2(cart) => mbc2_read_rom(cart, address),
        Mbc::Mbc3(cart) => mbc3_read_rom(cart, address),
        Mbc::Mbc5(cart) => mbc5_read_rom(cart, address),
    }
}

/// Reads a cartridge-RAM byte (0xA000-0xBFFF), or open-bus when RAM is disabled.
pub fn read_ram(mbc: &Mbc, address: u16) -> u8 {
    match mbc {
        Mbc::Mbc0(_) => 0,
        Mbc::Mbc1(cart) => mbc1_read_ram(cart, address),
        Mbc::Mbc2(cart) => mbc2_read_ram(cart, address),
        Mbc::Mbc3(cart) => mbc3_read_ram(cart, address),
        Mbc::Mbc5(cart) => mbc5_read_ram(cart, address),
    }
}

/// Writes to the ROM region, which the controller decodes as a bank/mode command.
pub fn write_rom(mbc: Mbc, address: u16, value: u8) -> Mbc {
    match mbc {
        Mbc::Mbc0(cart) => Mbc::Mbc0(cart),
        Mbc::Mbc1(cart) => Mbc::Mbc1(mbc1_write_rom(cart, address, value)),
        Mbc::Mbc2(cart) => Mbc::Mbc2(mbc2_write_rom(cart, address, value)),
        Mbc::Mbc3(cart) => Mbc::Mbc3(mbc3_write_rom(cart, address, value)),
        Mbc::Mbc5(cart) => Mbc::Mbc5(mbc5_write_rom(cart, address, value)),
    }
}

/// Writes a cartridge-RAM byte, ignored when RAM is disabled.
pub fn write_ram(mbc: Mbc, address: u16, value: u8) -> Mbc {
    match mbc {
        Mbc::Mbc0(cart) => Mbc::Mbc0(cart),
        Mbc::Mbc1(cart) => Mbc::Mbc1(mbc1_write_ram(cart, address, value)),
        Mbc::Mbc2(cart) => Mbc::Mbc2(mbc2_write_ram(cart, address, value)),
        Mbc::Mbc3(cart) => Mbc::Mbc3(mbc3_write_ram(cart, address, value)),
        Mbc::Mbc5(cart) => Mbc::Mbc5(mbc5_write_ram(cart, address, value)),
    }
}

// The MBC1 ROM read: the low 16 KiB is the fixed bank (bank 0, or the high bank
// bits in advanced mode); the high 16 KiB is the selected bank.
fn mbc1_read_rom(cart: &Mbc1, address: u16) -> u8 {
    let bank = match address < 0x4000 {
        true => match cart.banking_mode == 0 {
            true => 0,
            false => cart.rombank & 0xE0,
        },
        false => cart.rombank,
    };
    let index = bank * 0x4000 | ((address as usize) & 0x3FFF);
    cart.rom.get(index).copied().unwrap_or(0xFF)
}

// The MBC1 RAM read: open-bus (0xFF) when disabled; bank chosen only in advanced
// mode.
fn mbc1_read_ram(cart: &Mbc1, address: u16) -> u8 {
    match cart.ram_on {
        false => 0xFF,
        true => {
            let rambank = match cart.banking_mode == 1 {
                true => cart.rambank,
                false => 0,
            };
            cart.ram[(rambank * 0x2000) | ((address & 0x1FFF) as usize)]
        }
    }
}

// The MBC1 bank/mode command decode over the four ROM sub-ranges.
fn mbc1_write_rom(cart: Mbc1, address: u16, value: u8) -> Mbc1 {
    match address {
        0x0000..=0x1FFF => Mbc1 { ram_on: value & 0xF == 0xA, ..cart },
        0x2000..=0x3FFF => {
            // A written bank of 0 reads back as 1 (bank 0 is never selectable here).
            let lower = match (value as usize) & 0x1F {
                0 => 1,
                n => n,
            };
            Mbc1 { rombank: ((cart.rombank & 0x60) | lower) % cart.rombanks, ..cart }
        }
        0x4000..=0x5FFF => mbc1_bank_upper(cart, value),
        0x6000..=0x7FFF => Mbc1 { banking_mode: value & 0x01, ..cart },
        _ => panic!("Could not write to {:04X} (MBC1)", address),
    }
}

// The 0x4000-0x5FFF register: the ROM bank's high bits on large ROMs, the RAM bank
// on multi-bank RAM.
fn mbc1_bank_upper(cart: Mbc1, value: u8) -> Mbc1 {
    let rombank = match cart.rombanks > 0x20 {
        true => cart.rombank & 0x1F | (((value as usize & 0x03) % (cart.rombanks >> 5)) << 5),
        false => cart.rombank,
    };
    let rambank = match cart.rambanks > 1 {
        true => (value as usize) & 0x03,
        false => cart.rambank,
    };
    Mbc1 { rombank, rambank, ..cart }
}

// The MBC1 RAM write: dropped when RAM is off or the address is past the fitted
// RAM; otherwise the region is rebuilt with the byte in place.
fn mbc1_write_ram(cart: Mbc1, address: u16, value: u8) -> Mbc1 {
    match cart.ram_on {
        false => cart,
        true => {
            let rambank = match cart.banking_mode == 1 {
                true => cart.rambank,
                false => 0,
            };
            let index = (rambank * 0x2000) | ((address & 0x1FFF) as usize);
            match index < cart.ram.len() {
                true => Mbc1 {
                    ram: memory::write(&cart.ram, index, value),
                    ram_updated: true,
                    ..cart
                },
                false => cart,
            }
        }
    }
}

// The MBC2 initial state: a 512 x 4-bit built-in RAM, ROM bank starting at 1.
fn new_mbc2(data: Vec<u8>) -> Mbc2 {
    let has_battery = data[0x147] == 0x06;
    let rombanks = rom_banks(data[0x148]);
    Mbc2 { rom: data, ram: vec![0; 512], ram_on: false, ram_updated: false, rombank: 1, has_battery, rombanks }
}

fn mbc2_read_rom(cart: &Mbc2, address: u16) -> u8 {
    let bank = match address < 0x4000 {
        true => 0,
        false => cart.rombank,
    };
    cart.rom.get(bank * 0x4000 | ((address as usize) & 0x3FFF)).copied().unwrap_or(0xFF)
}

// MBC2 RAM is 4-bit, so the high nibble always reads as 1.
fn mbc2_read_ram(cart: &Mbc2, address: u16) -> u8 {
    match cart.ram_on {
        false => 0xFF,
        true => cart.ram[(address as usize) & 0x1FF] | 0xF0,
    }
}

// MBC2 folds RAM-enable and ROM-bank into one range, chosen by address bit 8.
fn mbc2_write_rom(cart: Mbc2, address: u16, value: u8) -> Mbc2 {
    match address {
        0x0000..=0x3FFF => match address & 0x100 == 0 {
            true => Mbc2 { ram_on: value & 0xF == 0xA, ..cart },
            false => {
                let bank = match (value as usize) & 0x0F {
                    0 => 1,
                    n => n,
                } % cart.rombanks;
                Mbc2 { rombank: bank, ..cart }
            }
        },
        _ => cart,
    }
}

fn mbc2_write_ram(cart: Mbc2, address: u16, value: u8) -> Mbc2 {
    match cart.ram_on {
        false => cart,
        true => Mbc2 {
            ram: memory::write(&cart.ram, (address as usize) & 0x1FF, value | 0xF0),
            ram_updated: true,
            ..cart
        },
    }
}

// The MBC5 initial state: bank registers span 9 bits of ROM and 4 bits of RAM.
fn new_mbc5(data: Vec<u8>) -> Mbc5 {
    let subtype = data[0x147];
    let has_battery = matches!(subtype, 0x1B | 0x1E);
    let rambanks = match subtype {
        0x1A | 0x1B | 0x1D | 0x1E => ram_banks(data[0x149]),
        _ => 0,
    };
    let rombanks = rom_banks(data[0x148]);
    Mbc5 {
        rom: data,
        ram: vec![0; 0x2000 * rambanks],
        rombank: 1,
        rambank: 0,
        ram_on: false,
        ram_updated: false,
        has_battery,
        rombanks,
        rambanks,
    }
}

fn mbc5_read_rom(cart: &Mbc5, address: u16) -> u8 {
    let index = match address < 0x4000 {
        true => address as usize,
        false => cart.rombank * 0x4000 | ((address as usize) & 0x3FFF),
    };
    cart.rom.get(index).copied().unwrap_or(0)
}

fn mbc5_read_ram(cart: &Mbc5, address: u16) -> u8 {
    match cart.ram_on {
        false => 0,
        true => cart.ram[cart.rambank * 0x2000 | ((address as usize) & 0x1FFF)],
    }
}

// MBC5 splits the ROM bank into a low byte (0x2000) and a 9th bit (0x3000).
fn mbc5_write_rom(cart: Mbc5, address: u16, value: u8) -> Mbc5 {
    match address {
        0x0000..=0x1FFF => Mbc5 { ram_on: value & 0x0F == 0x0A, ..cart },
        0x2000..=0x2FFF => Mbc5 { rombank: ((cart.rombank & 0x100) | (value as usize)) % cart.rombanks, ..cart },
        0x3000..=0x3FFF => {
            Mbc5 { rombank: ((cart.rombank & 0x0FF) | (((value & 0x1) as usize) << 8)) % cart.rombanks, ..cart }
        }
        0x4000..=0x5FFF => Mbc5 { rambank: ((value & 0x0F) as usize) % cart.rambanks, ..cart },
        0x6000..=0x7FFF => cart,
        _ => panic!("Could not write to {:04X} (MBC5)", address),
    }
}

fn mbc5_write_ram(cart: Mbc5, address: u16, value: u8) -> Mbc5 {
    match cart.ram_on {
        false => cart,
        true => Mbc5 {
            ram: memory::write(&cart.ram, cart.rambank * 0x2000 | ((address as usize) & 0x1FFF), value),
            ram_updated: true,
            ..cart
        },
    }
}

// The MBC3 initial state, including the optional RTC (subtypes 0x0F/0x10).
fn new_mbc3(data: Vec<u8>) -> Mbc3 {
    let subtype = data[0x147];
    let has_battery = matches!(subtype, 0x0F | 0x10 | 0x13);
    let rambanks = match subtype {
        0x10 | 0x12 | 0x13 => ram_banks(data[0x149]),
        _ => 0,
    };
    let rtc_zero = match subtype {
        0x0F | 0x10 => Some(0),
        _ => None,
    };
    Mbc3 {
        rom: data,
        ram: vec![0; rambanks * 0x2000],
        rombank: 1,
        rambank: 0,
        rambanks,
        selectrtc: false,
        ram_on: false,
        ram_updated: false,
        has_battery,
        rtc_ram: [0; 5],
        rtc_ram_latch: [0; 5],
        rtc_zero,
        now: 0,
    }
}

fn mbc3_read_rom(cart: &Mbc3, address: u16) -> u8 {
    let index = match address < 0x4000 {
        true => address as usize,
        false => cart.rombank * 0x4000 | ((address as usize) & 0x3FFF),
    };
    cart.rom.get(index).copied().unwrap_or(0xFF)
}

// MBC3 RAM reads return either banked RAM or the latched RTC register.
fn mbc3_read_ram(cart: &Mbc3, address: u16) -> u8 {
    match cart.ram_on {
        false => 0xFF,
        true => match (!cart.selectrtc && cart.rambank < cart.rambanks, cart.selectrtc && cart.rambank < 5) {
            (true, _) => cart.ram[cart.rambank * 0x2000 | ((address as usize) & 0x1FFF)],
            (_, true) => cart.rtc_ram_latch[cart.rambank],
            _ => 0xFF,
        },
    }
}

// MBC3 decodes RAM-enable, ROM bank, RAM/RTC select, and the RTC latch pulse.
fn mbc3_write_rom(cart: Mbc3, address: u16, value: u8) -> Mbc3 {
    match address {
        0x0000..=0x1FFF => Mbc3 { ram_on: value & 0x0F == 0x0A, ..cart },
        0x2000..=0x3FFF => Mbc3 {
            rombank: match value & 0x7F {
                0 => 1,
                n => n as usize,
            },
            ..cart
        },
        0x4000..=0x5FFF => Mbc3 { selectrtc: value & 0x8 == 0x8, rambank: (value & 0x7) as usize, ..cart },
        0x6000..=0x7FFF => mbc3_latch(cart),
        _ => panic!("Could not write to {:04X} (MBC3)", address),
    }
}

fn mbc3_write_ram(cart: Mbc3, address: u16, value: u8) -> Mbc3 {
    match cart.ram_on {
        false => cart,
        true => match (!cart.selectrtc && cart.rambank < cart.rambanks, cart.selectrtc && cart.rambank < 5) {
            (true, _) => Mbc3 {
                ram: memory::write(&cart.ram, cart.rambank * 0x2000 | ((address as usize) & 0x1FFF), value),
                ram_updated: true,
                ..cart
            },
            (_, true) => mbc3_write_rtc(cart, value),
            _ => cart,
        },
    }
}

// Latches the live RTC into the readable register set.
fn mbc3_latch(cart: Mbc3) -> Mbc3 {
    let calculated = mbc3_calc_rtc(cart);
    Mbc3 { rtc_ram_latch: calculated.rtc_ram, ..calculated }
}

// Writes one RTC register (masked per register) and re-bases the clock.
fn mbc3_write_rtc(cart: Mbc3, value: u8) -> Mbc3 {
    let calculated = mbc3_calc_rtc(cart);
    let mask = match calculated.rambank {
        0 | 1 => 0x3F,
        2 => 0x1F,
        4 => 0xC1,
        _ => 0xFF,
    };
    let bank = calculated.rambank;
    let rtc_ram = set_rtc(calculated.rtc_ram, bank, value & mask);
    mbc3_calc_zero(Mbc3 { rtc_ram, ram_updated: true, ..calculated })
}

// Recomputes the live RTC registers from the fixed clock, unless the RTC is halted.
fn mbc3_calc_rtc(cart: Mbc3) -> Mbc3 {
    match cart.rtc_ram[4] & 0x40 == 0x40 {
        true => cart,
        false => match cart.rtc_zero {
            None => cart,
            Some(zero) => mbc3_recompute(cart, zero),
        },
    }
}

// Fills the RTC registers from the seconds elapsed since the clock's zero point.
fn mbc3_recompute(cart: Mbc3, zero: u64) -> Mbc3 {
    match mbc3_compute_difftime(&cart) == Some(zero) {
        true => cart,
        false => {
            let difftime = cart.now.saturating_sub(zero);
            let days = difftime / (3600 * 24);
            let high = (cart.rtc_ram[4] & 0xFE) | (((days >> 8) & 0x01) as u8);
            let rtc_ram = [
                (difftime % 60) as u8,
                ((difftime / 60) % 60) as u8,
                ((difftime / 3600) % 24) as u8,
                days as u8,
                match days >= 512 {
                    true => high | 0x80,
                    false => high,
                },
            ];
            let updated = Mbc3 { rtc_ram, ..cart };
            match days >= 512 {
                true => mbc3_calc_zero(updated),
                false => updated,
            }
        }
    }
}

// The clock's zero point implied by the current registers (rboy's compute_difftime).
fn mbc3_compute_difftime(cart: &Mbc3) -> Option<u64> {
    match cart.rtc_zero {
        None => None,
        Some(_) => {
            let days = ((cart.rtc_ram[4] as u64 & 0x1) << 8) | (cart.rtc_ram[3] as u64);
            let elapsed = cart.rtc_ram[0] as u64
                + (cart.rtc_ram[1] as u64) * 60
                + (cart.rtc_ram[2] as u64) * 3600
                + days * 3600 * 24;
            Some(cart.now.saturating_sub(elapsed))
        }
    }
}

// Re-bases the clock's zero point from the current registers.
fn mbc3_calc_zero(cart: Mbc3) -> Mbc3 {
    let zero = mbc3_compute_difftime(&cart);
    Mbc3 { rtc_zero: zero, ..cart }
}

// Replaces one RTC register, keeping the rest.
fn set_rtc(rtc: [u8; 5], index: usize, value: u8) -> [u8; 5] {
    [0, 1, 2, 3, 4].map(|i| match i == index {
        true => value,
        false => rtc[i],
    })
}

/// Injects the current wall-clock time (Unix seconds) into the cartridge, consumed
/// only by an MBC3 real-time clock. The composition root supplies it (real time in
/// production, a fixed value for a deterministic run) so the core never reads a clock.
pub fn set_clock(mbc: Mbc, now: u64) -> Mbc {
    match mbc {
        Mbc::Mbc3(cart) => Mbc::Mbc3(Mbc3 { now, ..cart }),
        other => other,
    }
}

/// Whether the cartridge has battery-backed RAM worth persisting.
pub fn is_battery_backed(mbc: &Mbc) -> bool {
    match mbc {
        Mbc::Mbc0(_) => false,
        Mbc::Mbc1(cart) => cart.has_battery,
        Mbc::Mbc2(cart) => cart.has_battery,
        Mbc::Mbc3(cart) => cart.has_battery,
        Mbc::Mbc5(cart) => cart.has_battery,
    }
}

/// The battery-backed RAM contents to persist. MBC3 prepends the 8-byte RTC base so
/// the clock survives too (rboy's `dumpram`). The host writes these bytes to a file.
pub fn dump_ram(mbc: &Mbc) -> Vec<u8> {
    match mbc {
        Mbc::Mbc0(_) => Vec::new(),
        Mbc::Mbc1(cart) => cart.ram.clone(),
        Mbc::Mbc2(cart) => cart.ram.clone(),
        Mbc::Mbc3(cart) => [cart.rtc_zero.unwrap_or(0).to_be_bytes().to_vec(), cart.ram.clone()].concat(),
        Mbc::Mbc5(cart) => cart.ram.clone(),
    }
}

/// Restores battery-backed RAM (and the MBC3 RTC base) from persisted bytes; a
/// length mismatch leaves the cartridge unchanged (the host read a stale save).
pub fn load_ram(mbc: Mbc, data: &[u8]) -> Mbc {
    match mbc {
        Mbc::Mbc1(cart) if data.len() == cart.ram.len() => Mbc::Mbc1(Mbc1 { ram: data.to_vec(), ..cart }),
        Mbc::Mbc2(cart) if data.len() == cart.ram.len() => Mbc::Mbc2(Mbc2 { ram: data.to_vec(), ..cart }),
        Mbc::Mbc5(cart) if data.len() == cart.ram.len() => Mbc::Mbc5(Mbc5 { ram: data.to_vec(), ..cart }),
        Mbc::Mbc3(cart) if data.len() == 8 + cart.ram.len() => load_mbc3_ram(cart, data),
        other => other,
    }
}

// Splits the persisted MBC3 blob into its 8-byte RTC base and the RAM body.
fn load_mbc3_ram(cart: Mbc3, data: &[u8]) -> Mbc {
    let rtc = u64::from_be_bytes(data[..8].try_into().unwrap_or([0; 8]));
    let rtc_zero = cart.rtc_zero.map(|_| rtc);
    Mbc::Mbc3(Mbc3 { rtc_zero, ram: data[8..].to_vec(), ..cart })
}

#[cfg(test)]
mod tests {
    use crate::mbc;

    // A blank 32 KiB ROM whose header marks it MBC1 with 4 ROM banks and no RAM.
    fn rom_mbc1() -> Vec<u8> {
        // 0x147 cart type, 0x148 rom-size code (1 => 4 banks), 0x149 ram-size code.
        (0..0x8000usize)
            .map(|i| match i {
                0x147 => 0x01u8,
                0x148 => 0x01,
                _ => 0,
            })
            .collect()
    }

    #[test]
    fn get_mbc_selects_mbc1_and_sizes_banks() {
        let cart = mbc::get_mbc(rom_mbc1(), true).unwrap();
        match cart {
            mbc::Mbc::Mbc1(inner) => {
                assert_eq!(inner.rombanks, 4);
                assert_eq!(inner.rombank, 1);
                assert_eq!(inner.ram_on, false);
            }
            _ => panic!("expected an MBC1 cartridge"),
        }
    }

    #[test]
    fn rom_bank_zero_selects_bank_one() {
        // Writing 0 to the ROM-bank register selects bank 1, never bank 0.
        let cart = mbc::write_rom(mbc::get_mbc(rom_mbc1(), true).unwrap(), 0x2000, 0x00);
        match cart {
            mbc::Mbc::Mbc1(inner) => assert_eq!(inner.rombank, 1),
            _ => panic!("expected an MBC1 cartridge"),
        }
    }

    #[test]
    fn ram_enable_then_write_read_round_trip() {
        // Cart type 0x03 is MBC1+RAM+BATTERY; ram-size code 3 gives 4 banks.
        let rom: Vec<u8> = (0..0x8000usize)
            .map(|i| match i {
                0x147 => 0x03u8,
                0x148 => 0x01,
                0x149 => 0x03,
                _ => 0,
            })
            .collect();
        let cart = mbc::get_mbc(rom, true).unwrap();
        // 0x0A to 0x0000-0x1FFF enables RAM; then a byte round-trips through it.
        let cart = mbc::write_rom(cart, 0x0000, 0x0A);
        let cart = mbc::write_ram(cart, 0xA000, 0x42);
        assert_eq!(mbc::read_ram(&cart, 0xA000), 0x42);
    }

    #[test]
    fn read_rom_low_bank_is_fixed_bank_zero() {
        let rom: Vec<u8> = (0..0x8000usize)
            .map(|i| match i {
                0x147 => 0x01u8,
                0x148 => 0x01,
                0x0010 => 0xAB,
                _ => 0,
            })
            .collect();
        let cart = mbc::get_mbc(rom, true).unwrap();
        // 0x0010 is in the fixed low bank, so it reads the raw ROM byte.
        assert_eq!(mbc::read_rom(&cart, 0x0010), 0xAB);
    }

    #[test]
    fn disabled_ram_reads_open_bus() {
        let cart = mbc::get_mbc(rom_mbc1(), true).unwrap();
        // With RAM off (and none fitted), reads return 0xFF.
        assert_eq!(mbc::read_ram(&cart, 0xA000), 0xFF);
    }
}
