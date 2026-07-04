//! The CPU register file and its flag bits. rboy exposed these as `&mut self`
//! methods on `Registers`; here the state is a plain value and every mutator is a
//! free function that returns a new `Registers`. The `f` register's low nibble is
//! always clear on real hardware, so the setters mask it off.

use crate::gbmode;

/// The eight 8-bit registers plus the 16-bit program counter and stack pointer.
/// `f` holds the flag bits (see `Cpu_Flag`); its low nibble is always zero.
#[derive(Copy, Clone, Debug, PartialEq, Eq)]
pub struct Registers {
    pub a: u8,
    pub f: u8,
    pub b: u8,
    pub c: u8,
    pub d: u8,
    pub e: u8,
    pub h: u8,
    pub l: u8,
    pub pc: u16,
    pub sp: u16,
}

/// A single flag bit in the `f` register, its discriminant the bit mask.
#[derive(Copy, Clone, Debug, PartialEq, Eq)]
pub enum Cpu_Flag {
    C = 0b0001_0000,
    H = 0b0010_0000,
    N = 0b0100_0000,
    Z = 0b1000_0000,
}

/// The post-boot register file for a given console model, matching the values the
/// boot ROM leaves behind (rboy's `Registers::new`).
pub fn new(mode: gbmode::Gb_Mode) -> Registers {
    match mode {
        gbmode::Gb_Mode::Classic => Registers {
            a: 0x01,
            f: Cpu_Flag::C as u8 | Cpu_Flag::H as u8 | Cpu_Flag::Z as u8,
            b: 0x00, c: 0x13, d: 0x00, e: 0xD8, h: 0x01, l: 0x4D, pc: 0x0100, sp: 0xFFFE,
        },
        gbmode::Gb_Mode::Color_As_Classic => Registers {
            a: 0x11, f: Cpu_Flag::Z as u8,
            b: 0x00, c: 0x00, d: 0x00, e: 0x08, h: 0x00, l: 0x7C, pc: 0x0100, sp: 0xFFFE,
        },
        gbmode::Gb_Mode::Color => Registers {
            a: 0x11, f: Cpu_Flag::Z as u8,
            b: 0x00, c: 0x00, d: 0xFF, e: 0x56, h: 0x00, l: 0x0D, pc: 0x0100, sp: 0xFFFE,
        },
    }
}

/// The AF pair, F masked to its high nibble (the low nibble reads as zero).
pub fn af(reg: Registers) -> u16 {
    ((reg.a as u16) << 8) | ((reg.f & 0xF0) as u16)
}

pub fn bc(reg: Registers) -> u16 {
    ((reg.b as u16) << 8) | (reg.c as u16)
}

pub fn de(reg: Registers) -> u16 {
    ((reg.d as u16) << 8) | (reg.e as u16)
}

pub fn hl(reg: Registers) -> u16 {
    ((reg.h as u16) << 8) | (reg.l as u16)
}

/// Writes AF, masking F's low nibble off the way hardware does.
pub fn set_af(reg: Registers, value: u16) -> Registers {
    Registers { a: (value >> 8) as u8, f: (value & 0x00F0) as u8, ..reg }
}

pub fn set_bc(reg: Registers, value: u16) -> Registers {
    Registers { b: (value >> 8) as u8, c: (value & 0x00FF) as u8, ..reg }
}

pub fn set_de(reg: Registers, value: u16) -> Registers {
    Registers { d: (value >> 8) as u8, e: (value & 0x00FF) as u8, ..reg }
}

pub fn set_hl(reg: Registers, value: u16) -> Registers {
    Registers { h: (value >> 8) as u8, l: (value & 0x00FF) as u8, ..reg }
}

/// Returns the pre-decrement HL and the register file with HL decremented (the
/// `LD (HL-),…` addressing mode). `wrapping_sub` reproduces the hardware wrap that
/// rboy gets from release-mode overflow, so the value is correct in debug too.
pub fn hld(reg: Registers) -> (Registers, u16) {
    let value = hl(reg);
    (set_hl(reg, value.wrapping_sub(1)), value)
}

/// Returns the pre-increment HL and the register file with HL incremented.
pub fn hli(reg: Registers) -> (Registers, u16) {
    let value = hl(reg);
    (set_hl(reg, value.wrapping_add(1)), value)
}

/// Sets or clears one flag bit, keeping F's low nibble clear.
pub fn set_flag(reg: Registers, flag: Cpu_Flag, set: bool) -> Registers {
    let mask = flag as u8;
    let updated = match set {
        true => reg.f | mask,
        false => reg.f & !mask,
    };
    Registers { f: updated & 0xF0, ..reg }
}

/// Reports whether one flag bit is set.
pub fn get_flag(reg: Registers, flag: Cpu_Flag) -> bool {
    reg.f & (flag as u8) > 0
}

#[cfg(test)]
mod tests {
    use crate::gbmode;
    use crate::register;

    #[test]
    fn wide_registers() {
        let base = register::new(gbmode::Gb_Mode::Classic);
        let reg = register::Registers {
            a: 0x12,
            f: 0x23 & 0xF0,
            b: 0x34,
            c: 0x45,
            d: 0x56,
            e: 0x67,
            h: 0x78,
            l: 0x89,
            ..base
        };
        assert_eq!(register::af(reg), 0x1220);
        assert_eq!(register::bc(reg), 0x3445);
        assert_eq!(register::de(reg), 0x5667);
        assert_eq!(register::hl(reg), 0x7889);

        let reg = register::set_hl(
            register::set_de(register::set_bc(register::set_af(reg, 0x1111), 0x1111), 0x1111),
            0x1111,
        );
        // Writing 0x1111 to AF reads back as 0x1110, since set_af clears F's low nibble.
        assert_eq!(register::af(reg), 0x1110);
        assert_eq!(register::bc(reg), 0x1111);
        assert_eq!(register::de(reg), 0x1111);
        assert_eq!(register::hl(reg), 0x1111);
    }

    #[test]
    fn flags_round_trip() {
        let base = register::new(gbmode::Gb_Mode::Classic);
        // The freshly built register file already has a clear low nibble.
        assert_eq!(base.f & 0x0F, 0);
        let reg = register::Registers { f: 0x00, ..base };
        let masks =
            [register::Cpu_Flag::C, register::Cpu_Flag::H, register::Cpu_Flag::N, register::Cpu_Flag::Z];
        masks.iter().for_each(|&mask| {
            assert!(!register::get_flag(reg, mask));
            let set = register::set_flag(reg, mask, true);
            assert!(register::get_flag(set, mask));
            let cleared = register::set_flag(set, mask, false);
            assert!(!register::get_flag(cleared, mask));
        });
    }

    #[test]
    fn hl_increment_and_decrement() {
        let reg = register::set_hl(register::new(gbmode::Gb_Mode::Classic), 0x1234);
        let (reg, first) = register::hld(reg);
        assert_eq!(first, 0x1234);
        let (reg, second) = register::hld(reg);
        assert_eq!(second, 0x1233);
        let (reg, third) = register::hli(reg);
        assert_eq!(third, 0x1232);
        let (reg, fourth) = register::hli(reg);
        assert_eq!(fourth, 0x1233);
        assert_eq!(register::hl(reg), 0x1234);
    }
}
