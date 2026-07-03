//! The console model the emulator runs as. `Classic` is the original DMG; `Color`
//! is the Game Boy Color; `Color_As_Classic` is a CGB running a DMG cartridge in
//! compatibility mode. `Gb_Speed` is the CGB double-speed toggle, whose discriminant
//! doubles as the CPU/GPU tick divider (1 at single speed, 2 at double).

#[derive(PartialEq, Eq, Copy, Clone, Debug)]
pub enum Gb_Mode {
    Classic,
    Color,
    Color_As_Classic,
}

#[derive(PartialEq, Eq, Copy, Clone, Debug)]
pub enum Gb_Speed {
    Single = 1,
    Double = 2,
}
