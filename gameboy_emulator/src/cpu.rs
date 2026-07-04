//! The Sharp LR35902 core. rboy stepped `&mut self`; here the whole machine (`Cpu`
//! owns `Mmu`) is threaded by value: every instruction consumes the `Cpu` and
//! returns the new `Cpu` plus its M-cycle count. rboy's two ~1100-line opcode
//! `match`es are re-derived into small block/row free functions that reuse `alu`,
//! `register`, and `mmu` — the regular blocks (LD r,r', ALU A,r, the CB groups) are
//! computed from the operand/op bits rather than enumerated, the irregular rows are
//! one function per 16 opcodes so each body stays under the 70-line limit.

use crate::alu;
use crate::gbmode;
use crate::mbc;
use crate::mmu;
use crate::register;

/// The processor and the bus it drives, plus the interrupt-master flag and the
/// one-instruction DI/EI delay counters and the HALT state.
#[derive(Clone, Debug)]
pub struct Cpu {
    pub reg: register::Registers,
    pub mmu: mmu::Mmu,
    pub halted: bool,
    pub halt_bug: bool,
    pub ime: bool,
    pub setdi: u32,
    pub setei: u32,
}

/// Builds the machine for a cartridge and console mode, post-boot register state
/// included. Fails if the cartridge cannot run in the requested mode.
pub fn new(cart: mbc::Mbc, mode: gbmode::Gb_Mode) -> Result<Cpu, String> {
    assemble(mmu::new(cart, mode)?)
}

/// Builds the machine for CGB hardware, the mode chosen from the cartridge header.
pub fn new_cgb(cart: mbc::Mbc) -> Result<Cpu, String> {
    assemble(mmu::new_cgb(cart)?)
}

/// Injects the current wall-clock time into the machine (for an MBC3 RTC). The
/// composition root calls this before running so the core never reads a clock.
pub fn set_clock(cpu: Cpu, now: u64) -> Cpu {
    Cpu { mmu: mmu::set_clock(cpu.mmu, now), ..cpu }
}

/// Connects the Game Boy Printer to the machine's serial port.
pub fn attach_printer(cpu: Cpu) -> Cpu {
    Cpu { mmu: mmu::attach_printer(cpu.mmu), ..cpu }
}

// Wraps a freshly built bus in the initial CPU state (post-boot registers for its
// console mode, interrupts enabled).
fn assemble(bus: mmu::Mmu) -> Result<Cpu, String> {
    Ok(Cpu {
        reg: register::new(bus.gbmode),
        mmu: bus,
        halted: false,
        halt_bug: false,
        ime: true,
        setdi: 0,
        setei: 0,
    })
}

/// Runs one step (interrupt dispatch or one instruction), then advances the
/// peripherals by the elapsed T-cycles, returning the machine and the PPU tick count.
pub fn do_cycle(cpu: Cpu) -> (Cpu, u32) {
    let (stepped, mcycles) = step(cpu);
    let (advanced, gputicks) = mmu::do_cycle(stepped.mmu, mcycles * 4);
    (Cpu { mmu: advanced, ..stepped }, gputicks)
}

// One CPU step: apply the pending DI/EI edge, service an interrupt if one is due,
// otherwise execute an instruction (a halted CPU burns one idle M-cycle).
fn step(cpu: Cpu) -> (Cpu, u32) {
    let cpu = update_ime(cpu);
    let (cpu, serviced) = handle_interrupt(cpu);
    match serviced {
        0 => match cpu.halted {
            true => (cpu, 1),
            false => execute(cpu),
        },
        cycles => (cpu, cycles),
    }
}

// Counts the DI/EI delay down, flipping IME when a counter reaches its edge, so a
// DI/EI takes effect one instruction late (rboy's `updateime`).
fn update_ime(cpu: Cpu) -> Cpu {
    let (setdi, ime_after_di) = match cpu.setdi {
        2 => (1, cpu.ime),
        1 => (0, false),
        _ => (0, cpu.ime),
    };
    let (setei, ime) = match cpu.setei {
        2 => (1, ime_after_di),
        1 => (0, true),
        _ => (0, ime_after_di),
    };
    Cpu { setdi, setei, ime, ..cpu }
}

// Services the highest-priority pending-and-enabled interrupt when IME allows, or
// just wakes from HALT. Returns the added M-cycles (4 when a vector is taken).
fn handle_interrupt(cpu: Cpu) -> (Cpu, u32) {
    match !cpu.ime && !cpu.halted {
        true => (cpu, 0),
        false => {
            let triggered = cpu.mmu.inte & cpu.mmu.intf & 0x1F;
            match triggered == 0 {
                true => (cpu, 0),
                false => dispatch_interrupt(Cpu { halted: false, ..cpu }, triggered),
            }
        }
    }
}

// Takes the interrupt vector for the lowest set bit: clears IME and that IF bit,
// pushes PC, and jumps to 0x40 + n*8.
fn dispatch_interrupt(cpu: Cpu, triggered: u8) -> (Cpu, u32) {
    match cpu.ime {
        false => (cpu, 0),
        true => {
            let n = triggered.trailing_zeros();
            let cleared = Cpu {
                ime: false,
                mmu: mmu::Mmu { intf: cpu.mmu.intf & !(1u8 << n), ..cpu.mmu },
                ..cpu
            };
            let vector = 0x0040 | ((n as u16) << 3);
            (set_pc(push_pc(cleared), vector), 4)
        }
    }
}

// Reads the byte at PC and advances PC, unless the HALT bug is pending (then PC is
// read but not incremented, so the next byte is fetched twice).
fn fetch_byte(cpu: Cpu) -> (Cpu, u8) {
    let byte = mmu::read_byte(&cpu.mmu, cpu.reg.pc);
    match cpu.halt_bug {
        true => (Cpu { halt_bug: false, ..cpu }, byte),
        false => {
            let pc = cpu.reg.pc.wrapping_add(1);
            (set_pc(cpu, pc), byte)
        }
    }
}

// Reads the little-endian word at PC and advances PC by two.
fn fetch_word(cpu: Cpu) -> (Cpu, u16) {
    let word = mmu::read_wide(&cpu.mmu, cpu.reg.pc);
    let pc = cpu.reg.pc.wrapping_add(2);
    (set_pc(cpu, pc), word)
}

// Returns the machine with PC set.
fn set_pc(cpu: Cpu, pc: u16) -> Cpu {
    Cpu { reg: register::Registers { pc, ..cpu.reg }, ..cpu }
}

// Pushes a word: predecrements SP by two and writes it there.
fn push_stack(cpu: Cpu, value: u16) -> Cpu {
    let sp = cpu.reg.sp.wrapping_sub(2);
    let mmu = mmu::write_wide(cpu.mmu, sp, value);
    Cpu { reg: register::Registers { sp, ..cpu.reg }, mmu, ..cpu }
}

// Pops a word from SP and postincrements SP by two.
fn pop_stack(cpu: Cpu) -> (Cpu, u16) {
    let value = mmu::read_wide(&cpu.mmu, cpu.reg.sp);
    (Cpu { reg: register::Registers { sp: cpu.reg.sp.wrapping_add(2), ..cpu.reg }, ..cpu }, value)
}

// Reads an 8-bit operand slot (0=B..5=L, 6=(HL), 7=A) — pure, since even (HL) only
// reads memory.
fn read_operand(cpu: &Cpu, code: u8) -> u8 {
    match code {
        0 => cpu.reg.b,
        1 => cpu.reg.c,
        2 => cpu.reg.d,
        3 => cpu.reg.e,
        4 => cpu.reg.h,
        5 => cpu.reg.l,
        6 => mmu::read_byte(&cpu.mmu, register::hl(cpu.reg)),
        _ => cpu.reg.a,
    }
}

// Writes an 8-bit operand slot, storing to memory for the (HL) slot.
fn with_operand(cpu: Cpu, code: u8, value: u8) -> Cpu {
    match code {
        0 => Cpu { reg: register::Registers { b: value, ..cpu.reg }, ..cpu },
        1 => Cpu { reg: register::Registers { c: value, ..cpu.reg }, ..cpu },
        2 => Cpu { reg: register::Registers { d: value, ..cpu.reg }, ..cpu },
        3 => Cpu { reg: register::Registers { e: value, ..cpu.reg }, ..cpu },
        4 => Cpu { reg: register::Registers { h: value, ..cpu.reg }, ..cpu },
        5 => Cpu { reg: register::Registers { l: value, ..cpu.reg }, ..cpu },
        6 => Cpu { mmu: mmu::write_byte(cpu.mmu, register::hl(cpu.reg), value), ..cpu },
        _ => Cpu { reg: register::Registers { a: value, ..cpu.reg }, ..cpu },
    }
}

// Whether the flag condition (0=NZ, 1=Z, 2=NC, 3=C) holds.
fn condition_met(cpu: &Cpu, code: u8) -> bool {
    match code {
        0 => !register::get_flag(cpu.reg, register::Cpu_Flag::Z),
        1 => register::get_flag(cpu.reg, register::Cpu_Flag::Z),
        2 => !register::get_flag(cpu.reg, register::Cpu_Flag::C),
        _ => register::get_flag(cpu.reg, register::Cpu_Flag::C),
    }
}

// Decodes and runs one instruction, splitting the map into the four quadrants.
fn execute(cpu: Cpu) -> (Cpu, u32) {
    let (cpu, opcode) = fetch_byte(cpu);
    match opcode {
        0xCB => execute_cb(cpu),
        _ => match opcode >> 6 {
            0 => block_misc(cpu, opcode),
            1 => block_ld(cpu, opcode),
            2 => block_alu(cpu, opcode),
            _ => block_control(cpu, opcode),
        },
    }
}

// The 0x40-0x7F LD r,r' block, with HALT occupying 0x76.
fn block_ld(cpu: Cpu, opcode: u8) -> (Cpu, u32) {
    match opcode == 0x76 {
        true => halt(cpu),
        false => {
            let src = opcode & 0x07;
            let dst = (opcode >> 3) & 0x07;
            let value = read_operand(&cpu, src);
            (with_operand(cpu, dst, value), if src == 6 || dst == 6 { 2 } else { 1 })
        }
    }
}

// The 0x80-0xBF ALU-on-A block: the op is bits 3-5, the operand slot bits 0-2.
fn block_alu(cpu: Cpu, opcode: u8) -> (Cpu, u32) {
    let operand = read_operand(&cpu, opcode & 0x07);
    let reg = apply_alu(cpu.reg, (opcode >> 3) & 0x07, operand);
    (Cpu { reg, ..cpu }, if opcode & 0x07 == 6 { 2 } else { 1 })
}

// Applies one of the eight A-accumulator ALU ops.
fn apply_alu(reg: register::Registers, op: u8, operand: u8) -> register::Registers {
    match op {
        0 => alu::add(reg, operand, false),
        1 => alu::add(reg, operand, true),
        2 => alu::sub(reg, operand, false),
        3 => alu::sub(reg, operand, true),
        4 => alu::and(reg, operand),
        5 => alu::xor(reg, operand),
        6 => alu::or(reg, operand),
        _ => alu::cp(reg, operand),
    }
}

// HALT: stop until an interrupt, arming the HALT bug when one is already pending
// with IME off.
fn halt(cpu: Cpu) -> (Cpu, u32) {
    let bug = cpu.mmu.intf & cpu.mmu.inte & 0x1F != 0;
    (Cpu { halted: true, halt_bug: bug, ..cpu }, 1)
}

// A read-modify-write increment of an operand slot (INC r / INC (HL)).
fn inc_operand(cpu: Cpu, code: u8) -> (Cpu, u32) {
    let (reg, result) = alu::inc(cpu.reg, read_operand(&cpu, code));
    (with_operand(Cpu { reg, ..cpu }, code, result), if code == 6 { 3 } else { 1 })
}

// A read-modify-write decrement of an operand slot (DEC r / DEC (HL)).
fn dec_operand(cpu: Cpu, code: u8) -> (Cpu, u32) {
    let (reg, result) = alu::dec(cpu.reg, read_operand(&cpu, code));
    (with_operand(Cpu { reg, ..cpu }, code, result), if code == 6 { 3 } else { 1 })
}

// Loads an immediate byte into an operand slot (LD r,d8 / LD (HL),d8).
fn ld_immediate(cpu: Cpu, code: u8) -> (Cpu, u32) {
    let (cpu, value) = fetch_byte(cpu);
    (with_operand(cpu, code, value), if code == 6 { 3 } else { 2 })
}

// The accumulator rotates (RLCA/RRCA/RLA/RRA): like the CB rotate but Z is forced
// clear.
fn rotate_accumulator(cpu: Cpu, reg: register::Registers, result: u8) -> (Cpu, u32) {
    let cleared = register::set_flag(register::Registers { a: result, ..reg }, register::Cpu_Flag::Z, false);
    (Cpu { reg: cleared, ..cpu }, 1)
}

// A relative jump: PC += signed immediate.
fn cpu_jr(cpu: Cpu) -> Cpu {
    let (cpu, byte) = fetch_byte(cpu);
    let offset = byte as i8 as i32;
    let target = (cpu.reg.pc as i32 + offset) as u16;
    set_pc(cpu, target)
}

// Conditional relative jump, consuming the offset byte either way.
fn jr_conditional(cpu: Cpu, code: u8) -> (Cpu, u32) {
    match condition_met(&cpu, code) {
        true => (cpu_jr(cpu), 3),
        false => {
            let (cpu, _) = fetch_byte(cpu);
            (cpu, 2)
        }
    }
}

// Conditional absolute jump, always consuming the address word.
fn jp_conditional(cpu: Cpu, code: u8) -> (Cpu, u32) {
    let (cpu, address) = fetch_word(cpu);
    match condition_met(&cpu, code) {
        true => (set_pc(cpu, address), 4),
        false => (cpu, 3),
    }
}

// Conditional call, always consuming the address word.
fn call_conditional(cpu: Cpu, code: u8) -> (Cpu, u32) {
    let (cpu, address) = fetch_word(cpu);
    match condition_met(&cpu, code) {
        true => (set_pc(push_pc(cpu), address), 6),
        false => (cpu, 3),
    }
}

// Conditional return.
fn ret_conditional(cpu: Cpu, code: u8) -> (Cpu, u32) {
    match condition_met(&cpu, code) {
        true => {
            let (cpu, address) = pop_stack(cpu);
            (set_pc(cpu, address), 5)
        }
        false => (cpu, 2),
    }
}

// A restart: push PC and jump to the fixed vector.
fn restart(cpu: Cpu, vector: u16) -> (Cpu, u32) {
    (set_pc(push_pc(cpu), vector), 4)
}

// Pushes the return address (PC), read before the push consumes the machine — the
// call-argument order would otherwise move `cpu` before reading `cpu.reg.pc`.
fn push_pc(cpu: Cpu) -> Cpu {
    let pc = cpu.reg.pc;
    push_stack(cpu, pc)
}

// PUSH rr: read the pair before the push consumes the machine.
fn push_bc(cpu: Cpu) -> (Cpu, u32) {
    let value = register::bc(cpu.reg);
    (push_stack(cpu, value), 4)
}

fn push_de(cpu: Cpu) -> (Cpu, u32) {
    let value = register::de(cpu.reg);
    (push_stack(cpu, value), 4)
}

fn push_hl(cpu: Cpu) -> (Cpu, u32) {
    let value = register::hl(cpu.reg);
    (push_stack(cpu, value), 4)
}

fn push_af(cpu: Cpu) -> (Cpu, u32) {
    let value = register::af(cpu.reg);
    (push_stack(cpu, value), 4)
}

// The 0x00-0x3F block: 16-bit loads, INC/DEC, immediates, the A-rotates, JR, and
// the DAA/CPL/SCF/CCF group; split by row to bound each function.
fn block_misc(cpu: Cpu, opcode: u8) -> (Cpu, u32) {
    match opcode >> 4 {
        0 => block_misc_0(cpu, opcode),
        1 => block_misc_1(cpu, opcode),
        2 => block_misc_2(cpu, opcode),
        _ => block_misc_3(cpu, opcode),
    }
}

fn block_misc_0(cpu: Cpu, opcode: u8) -> (Cpu, u32) {
    match opcode {
        0x00 => (cpu, 1),
        0x01 => load_pair(cpu, register::set_bc),
        0x02 => (Cpu { mmu: mmu::write_byte(cpu.mmu, register::bc(cpu.reg), cpu.reg.a), ..cpu }, 2),
        0x03 => (Cpu { reg: register::set_bc(cpu.reg, register::bc(cpu.reg).wrapping_add(1)), ..cpu }, 2),
        0x04 => inc_operand(cpu, 0),
        0x05 => dec_operand(cpu, 0),
        0x06 => ld_immediate(cpu, 0),
        0x07 => {
            let (reg, result) = alu::rlc(cpu.reg, cpu.reg.a);
            rotate_accumulator(cpu, reg, result)
        }
        0x08 => store_sp(cpu),
        0x09 => (Cpu { reg: alu::add16(cpu.reg, register::bc(cpu.reg)), ..cpu }, 2),
        0x0A => {
            let address = register::bc(cpu.reg);
            (load_a(cpu, address), 2)
        }
        0x0B => (Cpu { reg: register::set_bc(cpu.reg, register::bc(cpu.reg).wrapping_sub(1)), ..cpu }, 2),
        0x0C => inc_operand(cpu, 1),
        0x0D => dec_operand(cpu, 1),
        0x0E => ld_immediate(cpu, 1),
        _ => {
            let (reg, result) = alu::rrc(cpu.reg, cpu.reg.a);
            rotate_accumulator(cpu, reg, result)
        }
    }
}

fn block_misc_1(cpu: Cpu, opcode: u8) -> (Cpu, u32) {
    match opcode {
        // STOP: on CGB it performs a pending double-speed switch, else a no-op.
        0x10 => (Cpu { mmu: mmu::switch_speed(cpu.mmu), ..cpu }, 1),
        0x11 => load_pair(cpu, register::set_de),
        0x12 => (Cpu { mmu: mmu::write_byte(cpu.mmu, register::de(cpu.reg), cpu.reg.a), ..cpu }, 2),
        0x13 => (Cpu { reg: register::set_de(cpu.reg, register::de(cpu.reg).wrapping_add(1)), ..cpu }, 2),
        0x14 => inc_operand(cpu, 2),
        0x15 => dec_operand(cpu, 2),
        0x16 => ld_immediate(cpu, 2),
        0x17 => {
            let (reg, result) = alu::rl(cpu.reg, cpu.reg.a);
            rotate_accumulator(cpu, reg, result)
        }
        0x18 => (cpu_jr(cpu), 3),
        0x19 => (Cpu { reg: alu::add16(cpu.reg, register::de(cpu.reg)), ..cpu }, 2),
        0x1A => {
            let address = register::de(cpu.reg);
            (load_a(cpu, address), 2)
        }
        0x1B => (Cpu { reg: register::set_de(cpu.reg, register::de(cpu.reg).wrapping_sub(1)), ..cpu }, 2),
        0x1C => inc_operand(cpu, 3),
        0x1D => dec_operand(cpu, 3),
        0x1E => ld_immediate(cpu, 3),
        _ => {
            let (reg, result) = alu::rr(cpu.reg, cpu.reg.a);
            rotate_accumulator(cpu, reg, result)
        }
    }
}

fn block_misc_2(cpu: Cpu, opcode: u8) -> (Cpu, u32) {
    match opcode {
        0x20 => jr_conditional(cpu, 0),
        0x21 => load_pair(cpu, register::set_hl),
        0x22 => store_hl_move(cpu, true),
        0x23 => (Cpu { reg: register::set_hl(cpu.reg, register::hl(cpu.reg).wrapping_add(1)), ..cpu }, 2),
        0x24 => inc_operand(cpu, 4),
        0x25 => dec_operand(cpu, 4),
        0x26 => ld_immediate(cpu, 4),
        0x27 => (Cpu { reg: alu::daa(cpu.reg), ..cpu }, 1),
        0x28 => jr_conditional(cpu, 1),
        0x29 => (Cpu { reg: alu::add16(cpu.reg, register::hl(cpu.reg)), ..cpu }, 2),
        0x2A => load_hl_move(cpu, true),
        0x2B => (Cpu { reg: register::set_hl(cpu.reg, register::hl(cpu.reg).wrapping_sub(1)), ..cpu }, 2),
        0x2C => inc_operand(cpu, 5),
        0x2D => dec_operand(cpu, 5),
        0x2E => ld_immediate(cpu, 5),
        _ => complement(cpu),
    }
}

fn block_misc_3(cpu: Cpu, opcode: u8) -> (Cpu, u32) {
    match opcode {
        0x30 => jr_conditional(cpu, 2),
        0x31 => {
            let (cpu, value) = fetch_word(cpu);
            (Cpu { reg: register::Registers { sp: value, ..cpu.reg }, ..cpu }, 3)
        }
        0x32 => store_hl_move(cpu, false),
        0x33 => (Cpu { reg: register::Registers { sp: cpu.reg.sp.wrapping_add(1), ..cpu.reg }, ..cpu }, 2),
        0x34 => inc_operand(cpu, 6),
        0x35 => dec_operand(cpu, 6),
        0x36 => ld_immediate(cpu, 6),
        0x37 => set_carry(cpu, true),
        0x38 => jr_conditional(cpu, 3),
        0x39 => (Cpu { reg: alu::add16(cpu.reg, cpu.reg.sp), ..cpu }, 2),
        0x3A => load_hl_move(cpu, false),
        0x3B => (Cpu { reg: register::Registers { sp: cpu.reg.sp.wrapping_sub(1), ..cpu.reg }, ..cpu }, 2),
        0x3C => inc_operand(cpu, 7),
        0x3D => dec_operand(cpu, 7),
        0x3E => ld_immediate(cpu, 7),
        _ => set_carry(cpu, false),
    }
}

// LD rr,d16: load an immediate word into a register pair.
fn load_pair(cpu: Cpu, setter: fn(register::Registers, u16) -> register::Registers) -> (Cpu, u32) {
    let (cpu, value) = fetch_word(cpu);
    (Cpu { reg: setter(cpu.reg, value), ..cpu }, 3)
}

// LD A,(rr): load A from the addressed byte.
fn load_a(cpu: Cpu, address: u16) -> Cpu {
    let value = mmu::read_byte(&cpu.mmu, address);
    Cpu { reg: register::Registers { a: value, ..cpu.reg }, ..cpu }
}

// LD (a16),SP: store SP at an immediate address.
fn store_sp(cpu: Cpu) -> (Cpu, u32) {
    let (cpu, address) = fetch_word(cpu);
    (Cpu { mmu: mmu::write_wide(cpu.mmu, address, cpu.reg.sp), ..cpu }, 5)
}

// LD (HL+),A / LD (HL-),A: store A at (HL), then step HL.
fn store_hl_move(cpu: Cpu, increment: bool) -> (Cpu, u32) {
    let address = register::hl(cpu.reg);
    let stepped = match increment {
        true => register::hli(cpu.reg).0,
        false => register::hld(cpu.reg).0,
    };
    (Cpu { mmu: mmu::write_byte(cpu.mmu, address, cpu.reg.a), reg: stepped, ..cpu }, 2)
}

// LD A,(HL+) / LD A,(HL-): load A from (HL), then step HL.
fn load_hl_move(cpu: Cpu, increment: bool) -> (Cpu, u32) {
    let value = mmu::read_byte(&cpu.mmu, register::hl(cpu.reg));
    let stepped = match increment {
        true => register::hli(cpu.reg).0,
        false => register::hld(cpu.reg).0,
    };
    (Cpu { reg: register::Registers { a: value, ..stepped }, ..cpu }, 2)
}

// CPL: complement A and set N and H.
fn complement(cpu: Cpu) -> (Cpu, u32) {
    let reg = register::Registers { a: !cpu.reg.a, ..cpu.reg };
    let reg = register::set_flag(reg, register::Cpu_Flag::H, true);
    (Cpu { reg: register::set_flag(reg, register::Cpu_Flag::N, true), ..cpu }, 1)
}

// SCF/CCF: set or complement the carry, clearing N and H.
fn set_carry(cpu: Cpu, force_set: bool) -> (Cpu, u32) {
    let carry = match force_set {
        true => true,
        false => !register::get_flag(cpu.reg, register::Cpu_Flag::C),
    };
    let reg = register::set_flag(cpu.reg, register::Cpu_Flag::C, carry);
    let reg = register::set_flag(reg, register::Cpu_Flag::N, false);
    (Cpu { reg: register::set_flag(reg, register::Cpu_Flag::H, false), ..cpu }, 1)
}

// The 0xC0-0xFF control block: returns, jumps, calls, stack ops, RST, the I/O
// loads, and the immediate ALU ops; split by row.
fn block_control(cpu: Cpu, opcode: u8) -> (Cpu, u32) {
    match opcode >> 4 {
        0xC => block_control_c(cpu, opcode),
        0xD => block_control_d(cpu, opcode),
        0xE => block_control_e(cpu, opcode),
        _ => block_control_f(cpu, opcode),
    }
}

fn block_control_c(cpu: Cpu, opcode: u8) -> (Cpu, u32) {
    match opcode {
        0xC0 => ret_conditional(cpu, 0),
        0xC1 => pop_pair(cpu, register::set_bc),
        0xC2 => jp_conditional(cpu, 0),
        0xC3 => {
            let (cpu, address) = fetch_word(cpu);
            (set_pc(cpu, address), 4)
        }
        0xC4 => call_conditional(cpu, 0),
        0xC5 => push_bc(cpu),
        0xC6 => alu_immediate(cpu, 0),
        0xC7 => restart(cpu, 0x00),
        0xC8 => ret_conditional(cpu, 1),
        0xC9 => {
            let (cpu, address) = pop_stack(cpu);
            (set_pc(cpu, address), 4)
        }
        0xCA => jp_conditional(cpu, 1),
        0xCC => call_conditional(cpu, 1),
        0xCD => {
            let (cpu, address) = fetch_word(cpu);
            (set_pc(push_pc(cpu), address), 6)
        }
        0xCE => alu_immediate(cpu, 1),
        _ => restart(cpu, 0x08),
    }
}

fn block_control_d(cpu: Cpu, opcode: u8) -> (Cpu, u32) {
    match opcode {
        0xD0 => ret_conditional(cpu, 2),
        0xD1 => pop_pair(cpu, register::set_de),
        0xD2 => jp_conditional(cpu, 2),
        0xD4 => call_conditional(cpu, 2),
        0xD5 => push_de(cpu),
        0xD6 => alu_immediate(cpu, 2),
        0xD7 => restart(cpu, 0x10),
        0xD8 => ret_conditional(cpu, 3),
        0xD9 => return_from_interrupt(cpu),
        0xDA => jp_conditional(cpu, 3),
        0xDC => call_conditional(cpu, 3),
        0xDE => alu_immediate(cpu, 3),
        0xDF => restart(cpu, 0x18),
        _ => panic!("Instruction {:02X} is not implemented", opcode),
    }
}

fn block_control_e(cpu: Cpu, opcode: u8) -> (Cpu, u32) {
    match opcode {
        0xE0 => store_high_immediate(cpu),
        0xE1 => pop_pair(cpu, register::set_hl),
        0xE2 => (Cpu { mmu: mmu::write_byte(cpu.mmu, 0xFF00 | cpu.reg.c as u16, cpu.reg.a), ..cpu }, 2),
        0xE5 => push_hl(cpu),
        0xE6 => alu_immediate(cpu, 4),
        0xE7 => restart(cpu, 0x20),
        0xE8 => add_sp_immediate(cpu),
        0xE9 => {
            let target = register::hl(cpu.reg);
            (set_pc(cpu, target), 1)
        }
        0xEA => store_absolute(cpu),
        0xEE => alu_immediate(cpu, 5),
        0xEF => restart(cpu, 0x28),
        _ => panic!("Instruction {:02X} is not implemented", opcode),
    }
}

fn block_control_f(cpu: Cpu, opcode: u8) -> (Cpu, u32) {
    match opcode {
        0xF0 => load_high_immediate(cpu),
        0xF1 => pop_pair(cpu, register::set_af),
        0xF2 => {
            let address = 0xFF00 | cpu.reg.c as u16;
            (load_a(cpu, address), 2)
        }
        0xF3 => (Cpu { setdi: 2, ..cpu }, 1),
        0xF5 => push_af(cpu),
        0xF6 => alu_immediate(cpu, 6),
        0xF7 => restart(cpu, 0x30),
        0xF8 => load_hl_sp_immediate(cpu),
        0xF9 => (Cpu { reg: register::Registers { sp: register::hl(cpu.reg), ..cpu.reg }, ..cpu }, 2),
        0xFA => load_absolute(cpu),
        0xFB => (Cpu { setei: 2, ..cpu }, 1),
        0xFE => alu_immediate(cpu, 7),
        0xFF => restart(cpu, 0x38),
        _ => panic!("Instruction {:02X} is not implemented", opcode),
    }
}

// POP rr into a register pair (POP AF masks F's low nibble via `set_af`).
fn pop_pair(cpu: Cpu, setter: fn(register::Registers, u16) -> register::Registers) -> (Cpu, u32) {
    let (cpu, value) = pop_stack(cpu);
    (Cpu { reg: setter(cpu.reg, value), ..cpu }, 3)
}

// RETI: return and re-enable interrupts (delayed one step through `setei`).
fn return_from_interrupt(cpu: Cpu) -> (Cpu, u32) {
    let (cpu, address) = pop_stack(cpu);
    (Cpu { setei: 1, ..set_pc(cpu, address) }, 4)
}

// The immediate-operand ALU ops (op A,d8).
fn alu_immediate(cpu: Cpu, op: u8) -> (Cpu, u32) {
    let (cpu, value) = fetch_byte(cpu);
    (Cpu { reg: apply_alu(cpu.reg, op, value), ..cpu }, 2)
}

// LDH (a8),A: store A to the high page at an immediate offset.
fn store_high_immediate(cpu: Cpu) -> (Cpu, u32) {
    let (cpu, offset) = fetch_byte(cpu);
    (Cpu { mmu: mmu::write_byte(cpu.mmu, 0xFF00 | offset as u16, cpu.reg.a), ..cpu }, 3)
}

// LDH A,(a8): load A from the high page at an immediate offset.
fn load_high_immediate(cpu: Cpu) -> (Cpu, u32) {
    let (cpu, offset) = fetch_byte(cpu);
    (load_a(cpu, 0xFF00 | offset as u16), 3)
}

// LD (a16),A: store A at an immediate absolute address.
fn store_absolute(cpu: Cpu) -> (Cpu, u32) {
    let (cpu, address) = fetch_word(cpu);
    (Cpu { mmu: mmu::write_byte(cpu.mmu, address, cpu.reg.a), ..cpu }, 4)
}

// LD A,(a16): load A from an immediate absolute address.
fn load_absolute(cpu: Cpu) -> (Cpu, u32) {
    let (cpu, address) = fetch_word(cpu);
    (load_a(cpu, address), 4)
}

// ADD SP,r8: add a signed immediate to SP.
fn add_sp_immediate(cpu: Cpu) -> (Cpu, u32) {
    let (cpu, byte) = fetch_byte(cpu);
    let (reg, result) = alu::add16imm(cpu.reg, cpu.reg.sp, byte as i8 as i16 as u16);
    (Cpu { reg: register::Registers { sp: result, ..reg }, ..cpu }, 4)
}

// LD HL,SP+r8: HL gets SP plus a signed immediate.
fn load_hl_sp_immediate(cpu: Cpu) -> (Cpu, u32) {
    let (cpu, byte) = fetch_byte(cpu);
    let (reg, result) = alu::add16imm(cpu.reg, cpu.reg.sp, byte as i8 as i16 as u16);
    (Cpu { reg: register::set_hl(reg, result), ..cpu }, 3)
}

// The CB-prefixed page: rotates/shifts (0x00-0x3F), BIT (0x40-0x7F), RES
// (0x80-0xBF), SET (0xC0-0xFF).
fn execute_cb(cpu: Cpu) -> (Cpu, u32) {
    let (cpu, opcode) = fetch_byte(cpu);
    let code = opcode & 0x07;
    let value = read_operand(&cpu, code);
    let bit = (opcode >> 3) & 0x07;
    match opcode >> 6 {
        0 => cb_shift(cpu, bit, code, value),
        1 => (Cpu { reg: alu::bit(cpu.reg, value, bit), ..cpu }, if code == 6 { 3 } else { 2 }),
        2 => (with_operand(cpu, code, value & !(1 << bit)), if code == 6 { 4 } else { 2 }),
        _ => (with_operand(cpu, code, value | (1 << bit)), if code == 6 { 4 } else { 2 }),
    }
}

// A CB rotate/shift: compute the result and flags, write the slot back.
fn cb_shift(cpu: Cpu, op: u8, code: u8, value: u8) -> (Cpu, u32) {
    let (reg, result) = apply_shift(cpu.reg, op, value);
    (with_operand(Cpu { reg, ..cpu }, code, result), if code == 6 { 4 } else { 2 })
}

// The eight CB rotate/shift ops.
fn apply_shift(reg: register::Registers, op: u8, value: u8) -> (register::Registers, u8) {
    match op {
        0 => alu::rlc(reg, value),
        1 => alu::rrc(reg, value),
        2 => alu::rl(reg, value),
        3 => alu::rr(reg, value),
        4 => alu::sla(reg, value),
        5 => alu::sra(reg, value),
        6 => alu::swap(reg, value),
        _ => alu::srl(reg, value),
    }
}
