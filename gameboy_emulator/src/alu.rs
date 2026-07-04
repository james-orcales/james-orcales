//! The arithmetic/logic/shift core, ported from rboy's `alu_*` methods. rboy read
//! and wrote `self.reg.a` and the flags in place; here every operation takes the
//! register file by value and returns the updated file (plus a result byte for the
//! ops whose result the caller stores elsewhere). `wrapping_*` makes the hardware
//! wrap explicit so results are identical in debug and release, matching rboy's
//! release-mode overflow behavior.

use crate::register;

// Sets all four flags at once. Ops that leave a flag untouched pass its current
// value straight back through (read from `reg`), so this one helper serves every
// case without a separate partial-update path.
fn flags(reg: register::Registers, z: bool, n: bool, h: bool, c: bool) -> register::Registers {
    let reg = register::set_flag(reg, register::Cpu_Flag::Z, z);
    let reg = register::set_flag(reg, register::Cpu_Flag::N, n);
    let reg = register::set_flag(reg, register::Cpu_Flag::H, h);
    register::set_flag(reg, register::Cpu_Flag::C, c)
}

/// `A = A + b (+ carry)`; sets Z, clears N, sets H/C from the nibble/byte carries.
pub fn add(reg: register::Registers, b: u8, use_carry: bool) -> register::Registers {
    let carry = carry_in(reg, use_carry);
    let a = reg.a;
    let result = a.wrapping_add(b).wrapping_add(carry);
    let reg = flags(
        reg,
        result == 0,
        false,
        (a & 0xF) + (b & 0xF) + carry > 0xF,
        (a as u16) + (b as u16) + (carry as u16) > 0xFF,
    );
    register::Registers { a: result, ..reg }
}

/// `A = A - b (- carry)`; sets N and Z, sets H/C from the nibble/byte borrows.
pub fn sub(reg: register::Registers, b: u8, use_carry: bool) -> register::Registers {
    let carry = carry_in(reg, use_carry);
    let a = reg.a;
    let result = a.wrapping_sub(b).wrapping_sub(carry);
    let reg = flags(
        reg,
        result == 0,
        true,
        (a & 0x0F) < (b & 0x0F) + carry,
        (a as u16) < (b as u16) + (carry as u16),
    );
    register::Registers { a: result, ..reg }
}

// The carry bit to fold into an add/sub, or zero when the op ignores carry.
fn carry_in(reg: register::Registers, use_carry: bool) -> u8 {
    match use_carry && register::get_flag(reg, register::Cpu_Flag::C) {
        true => 1,
        false => 0,
    }
}

/// `A = A & b`; Z from the result, H set, N and C clear.
pub fn and(reg: register::Registers, b: u8) -> register::Registers {
    let result = reg.a & b;
    register::Registers { a: result, ..flags(reg, result == 0, false, true, false) }
}

/// `A = A | b`; Z from the result, all other flags clear.
pub fn or(reg: register::Registers, b: u8) -> register::Registers {
    let result = reg.a | b;
    register::Registers { a: result, ..flags(reg, result == 0, false, false, false) }
}

/// `A = A ^ b`; Z from the result, all other flags clear.
pub fn xor(reg: register::Registers, b: u8) -> register::Registers {
    let result = reg.a ^ b;
    register::Registers { a: result, ..flags(reg, result == 0, false, false, false) }
}

/// Compare: the flags of `A - b` with A left unchanged.
pub fn cp(reg: register::Registers, b: u8) -> register::Registers {
    register::Registers { a: reg.a, ..sub(reg, b, false) }
}

/// `value + 1`; sets Z/H, clears N, leaves C. Returns the flags and the result.
pub fn inc(reg: register::Registers, value: u8) -> (register::Registers, u8) {
    let result = value.wrapping_add(1);
    let carry = register::get_flag(reg, register::Cpu_Flag::C);
    (flags(reg, result == 0, false, (value & 0x0F) + 1 > 0x0F, carry), result)
}

/// `value - 1`; sets Z/N, H on a nibble borrow, leaves C. Returns flags and result.
pub fn dec(reg: register::Registers, value: u8) -> (register::Registers, u8) {
    let result = value.wrapping_sub(1);
    let carry = register::get_flag(reg, register::Cpu_Flag::C);
    (flags(reg, result == 0, true, value & 0x0F == 0, carry), result)
}

/// `HL = HL + b`; clears N, sets H/C from the 12-bit/16-bit carries, leaves Z.
pub fn add16(reg: register::Registers, b: u16) -> register::Registers {
    let a = register::hl(reg);
    let zero = register::get_flag(reg, register::Cpu_Flag::Z);
    let reg = flags(reg, zero, false, (a & 0x0FFF) + (b & 0x0FFF) > 0x0FFF, a > 0xFFFF - b);
    register::set_hl(reg, a.wrapping_add(b))
}

/// `a + signed(b)` for `ADD SP,r8` / `LD HL,SP+r8`. Clears Z and N; H/C come from
/// the low nibble/byte. `b` is the already sign-extended immediate. Returns the
/// flags and the sum (the caller stores it in SP or HL).
pub fn add16imm(reg: register::Registers, a: u16, b: u16) -> (register::Registers, u16) {
    let reg = flags(
        reg,
        false,
        false,
        (a & 0x000F) + (b & 0x000F) > 0x000F,
        (a & 0x00FF) + (b & 0x00FF) > 0x00FF,
    );
    (reg, a.wrapping_add(b))
}

/// Swaps the nibbles of `value`; Z from the result, all other flags clear.
pub fn swap(reg: register::Registers, value: u8) -> (register::Registers, u8) {
    let result = value.rotate_left(4);
    (flags(reg, value == 0, false, false, false), result)
}

// The shift/rotate flag update rboy shares across the shift ops: clears H/N, Z
// from the result, C from the shifted-out bit.
fn shift_flags(reg: register::Registers, result: u8, carry: bool) -> register::Registers {
    flags(reg, result == 0, false, false, carry)
}

/// Rotate left, bit 7 into both carry and bit 0.
pub fn rlc(reg: register::Registers, value: u8) -> (register::Registers, u8) {
    let carry = value & 0x80 == 0x80;
    let result = (value << 1) | u8::from(carry);
    (shift_flags(reg, result, carry), result)
}

/// Rotate left through carry: old carry into bit 0, bit 7 out to carry.
pub fn rl(reg: register::Registers, value: u8) -> (register::Registers, u8) {
    let carry = value & 0x80 == 0x80;
    let result = (value << 1) | u8::from(register::get_flag(reg, register::Cpu_Flag::C));
    (shift_flags(reg, result, carry), result)
}

/// Rotate right, bit 0 into both carry and bit 7.
pub fn rrc(reg: register::Registers, value: u8) -> (register::Registers, u8) {
    let carry = value & 0x01 == 0x01;
    let result = (value >> 1) | (if carry { 0x80 } else { 0 });
    (shift_flags(reg, result, carry), result)
}

/// Rotate right through carry: old carry into bit 7, bit 0 out to carry.
pub fn rr(reg: register::Registers, value: u8) -> (register::Registers, u8) {
    let carry = value & 0x01 == 0x01;
    let bit7 = if register::get_flag(reg, register::Cpu_Flag::C) { 0x80 } else { 0 };
    let result = (value >> 1) | bit7;
    (shift_flags(reg, result, carry), result)
}

/// Arithmetic/logical shift left; bit 7 out to carry, bit 0 cleared.
pub fn sla(reg: register::Registers, value: u8) -> (register::Registers, u8) {
    let carry = value & 0x80 == 0x80;
    let result = value << 1;
    (shift_flags(reg, result, carry), result)
}

/// Arithmetic shift right; bit 7 (the sign) preserved, bit 0 out to carry.
pub fn sra(reg: register::Registers, value: u8) -> (register::Registers, u8) {
    let carry = value & 0x01 == 0x01;
    let result = (value >> 1) | (value & 0x80);
    (shift_flags(reg, result, carry), result)
}

/// Logical shift right; bit 7 cleared, bit 0 out to carry.
pub fn srl(reg: register::Registers, value: u8) -> (register::Registers, u8) {
    let carry = value & 0x01 == 0x01;
    let result = value >> 1;
    (shift_flags(reg, result, carry), result)
}

/// Tests bit `n` of `value`: Z set when the bit is clear, H set, N clear, C left.
pub fn bit(reg: register::Registers, value: u8, n: u8) -> register::Registers {
    let zero = value & (1 << (n as u32)) == 0;
    let carry = register::get_flag(reg, register::Cpu_Flag::C);
    flags(reg, zero, false, true, carry)
}

/// Decimal-adjust A after a BCD add/subtract; sets C/Z, clears H (rboy's `alu_daa`).
pub fn daa(reg: register::Registers) -> register::Registers {
    let a = reg.a;
    let negative = register::get_flag(reg, register::Cpu_Flag::N);
    let carry = register::get_flag(reg, register::Cpu_Flag::C);
    let half = register::get_flag(reg, register::Cpu_Flag::H);
    // The correction: 0x60 for a high-nibble/carry overflow, 0x06 for the low one.
    let base = (if carry { 0x60u8 } else { 0 }) | (if half { 0x06 } else { 0 });
    let adjust = match negative {
        true => base,
        false => base | high_adjust(a) | low_adjust(a),
    };
    let result = match negative {
        true => a.wrapping_sub(adjust),
        false => a.wrapping_add(adjust),
    };
    let reg = flags(reg, result == 0, negative, false, adjust >= 0x60);
    register::Registers { a: result, ..reg }
}

// The high-nibble DAA correction for an add whose value exceeds 0x99.
fn high_adjust(a: u8) -> u8 {
    match a > 0x99 {
        true => 0x60,
        false => 0,
    }
}

// The low-nibble DAA correction for an add whose low nibble exceeds 9.
fn low_adjust(a: u8) -> u8 {
    match a & 0x0F > 0x09 {
        true => 0x06,
        false => 0,
    }
}

#[cfg(test)]
mod tests {
    use crate::alu;
    use crate::gbmode;
    use crate::register;

    // A register file with a chosen accumulator and flag byte, boot values elsewhere.
    fn state(a: u8, f: u8) -> register::Registers {
        register::Registers { a, f, ..register::new(gbmode::Gb_Mode::Classic) }
    }

    // The (Z, N, H, C) flags as booleans, in the order the tests assert them.
    fn flag_bits(reg: register::Registers) -> (bool, bool, bool, bool) {
        (
            register::get_flag(reg, register::Cpu_Flag::Z),
            register::get_flag(reg, register::Cpu_Flag::N),
            register::get_flag(reg, register::Cpu_Flag::H),
            register::get_flag(reg, register::Cpu_Flag::C),
        )
    }

    #[test]
    fn add_sets_zero_half_and_carry() {
        // 0x3A + 0xC6 = 0x00, with a half-carry and a carry out.
        let reg = alu::add(state(0x3A, 0), 0xC6, false);
        assert_eq!(reg.a, 0x00);
        assert_eq!(flag_bits(reg), (true, false, true, true));
    }

    #[test]
    fn adc_includes_carry_in() {
        // With C set, 0x00 + 0x00 + 1 = 0x01 and no flags survive.
        let reg = alu::add(state(0x00, register::Cpu_Flag::C as u8), 0x00, true);
        assert_eq!(reg.a, 0x01);
        assert_eq!(flag_bits(reg), (false, false, false, false));
    }

    #[test]
    fn sub_sets_borrow_flags() {
        // 0x3E - 0x40 = 0xFE, a full borrow (C) with N set and no half-borrow.
        let reg = alu::sub(state(0x3E, 0), 0x40, false);
        assert_eq!(reg.a, 0xFE);
        assert_eq!(flag_bits(reg), (false, true, false, true));
    }

    #[test]
    fn cp_preserves_a() {
        // Compare is a subtract that keeps A and only sets the flags.
        let reg = alu::cp(state(0x3E, 0), 0x40);
        assert_eq!(reg.a, 0x3E);
        assert_eq!(flag_bits(reg), (false, true, false, true));
    }

    #[test]
    fn inc_wraps_and_sets_half_carry() {
        // 0xFF + 1 wraps to 0x00: Z and H set, N clear, C untouched (stays set).
        let (reg, result) = alu::inc(state(0x00, register::Cpu_Flag::C as u8), 0xFF);
        assert_eq!(result, 0x00);
        assert_eq!(flag_bits(reg), (true, false, true, true));
    }

    #[test]
    fn dec_sets_half_borrow_at_nibble_boundary() {
        // 0x10 - 1 borrows across the nibble: H and N set, not zero, C untouched.
        let (reg, result) = alu::dec(state(0x00, 0), 0x10);
        assert_eq!(result, 0x0F);
        assert_eq!(flag_bits(reg), (false, true, true, false));
    }

    #[test]
    fn swap_exchanges_nibbles() {
        let (reg, result) = alu::swap(state(0, 0), 0xAB);
        assert_eq!(result, 0xBA);
        assert_eq!(flag_bits(reg), (false, false, false, false));
    }

    #[test]
    fn daa_after_addition() {
        // 0x45 + 0x38 = 0x7D; DAA corrects it to the BCD result 0x83.
        let reg = alu::daa(alu::add(state(0x45, 0), 0x38, false));
        assert_eq!(reg.a, 0x83);
        assert!(!register::get_flag(reg, register::Cpu_Flag::C));
    }

    #[test]
    fn rlc_rotates_left_through_bit7() {
        // 0x85 = 1000_0101 rotates to 0000_1011, bit7 into carry and bit0.
        let (reg, result) = alu::rlc(state(0, 0), 0x85);
        assert_eq!(result, 0x0B);
        assert_eq!(flag_bits(reg), (false, false, false, true));
    }

    #[test]
    fn rr_rotates_carry_into_bit7() {
        // Carry into bit7, the old bit0 out to carry: 0x01 -> 0x80.
        let (reg, result) = alu::rr(state(0, register::Cpu_Flag::C as u8), 0x01);
        assert_eq!(result, 0x80);
        assert_eq!(flag_bits(reg), (false, false, false, true));
    }

    #[test]
    fn bit_sets_zero_when_bit_clear() {
        // Bit 7 of 0x00 is clear so Z is set; H set, N clear, C preserved.
        let reg = alu::bit(state(0, register::Cpu_Flag::C as u8), 0x00, 7);
        assert_eq!(flag_bits(reg), (true, false, true, true));
    }

    #[test]
    fn add16_sets_half_and_carry_not_zero() {
        // 0xFFFF + 1 wraps HL to 0; H and C set, Z left as it was (set here).
        let reg = register::set_hl(state(0, register::Cpu_Flag::Z as u8), 0xFFFF);
        let reg = alu::add16(reg, 0x0001);
        assert_eq!(register::hl(reg), 0x0000);
        assert_eq!(flag_bits(reg), (true, false, true, true));
    }

    #[test]
    fn add16imm_flags_from_low_bytes() {
        // 0x0008 + 0x0008 = 0x0010: nibble carry sets H, no byte carry so C clear.
        let (reg, result) = alu::add16imm(state(0, 0), 0x0008, 0x0008);
        assert_eq!(result, 0x0010);
        assert_eq!(flag_bits(reg), (false, false, true, false));
    }

    #[test]
    fn add16imm_handles_negative_immediate() {
        // A sign-extended -1 (0xFFFF) added to 0 gives 0xFFFF with no carries.
        let (reg, result) = alu::add16imm(state(0, 0), 0x0000, 0xFFFF);
        assert_eq!(result, 0xFFFF);
        assert_eq!(flag_bits(reg), (false, false, false, false));
    }
}
