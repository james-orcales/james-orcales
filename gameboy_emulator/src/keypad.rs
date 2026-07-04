//! The joypad matrix. rboy mutated `&mut self`; here reads are pure and writes and
//! key events return a new `Keypad`. Bits 4-5 of the register select the direction
//! and/or button row; a pressed key clears its bit in the matching row, and a
//! falling edge on the low nibble raises the joypad interrupt (0x10).

/// The eight console keys, split across the two selectable rows.
#[derive(Copy, Clone, Debug)]
pub enum Keypad_Key {
    Right,
    Left,
    Up,
    Down,
    A,
    B,
    Select,
    Start,
}

/// The two row latches (direction, button), the register readback, and the pending
/// interrupt byte.
#[derive(Copy, Clone, Debug)]
pub struct Keypad {
    pub row0: u8,
    pub row1: u8,
    pub data: u8,
    pub interrupt: u8,
}

pub fn new() -> Keypad {
    Keypad { row0: 0x0F, row1: 0x0F, data: 0xFF, interrupt: 0 }
}

pub fn read_byte(keypad: &Keypad) -> u8 {
    keypad.data
}

pub fn write_byte(keypad: Keypad, value: u8) -> Keypad {
    // Only the row-select bits (4-5) are writable; the rest is recomputed.
    refresh(Keypad { data: (keypad.data & 0xCF) | (value & 0x30), ..keypad })
}

/// Presses a key, clearing its row bit, then refreshes the readback.
pub fn keydown(keypad: Keypad, key: Keypad_Key) -> Keypad {
    refresh(press(keypad, key))
}

/// Releases a key, setting its row bit, then refreshes the readback.
pub fn keyup(keypad: Keypad, key: Keypad_Key) -> Keypad {
    refresh(release(keypad, key))
}

// Recomputes the low nibble from the selected rows and raises the joypad interrupt
// on a high-to-low transition of any selected line.
fn refresh(keypad: Keypad) -> Keypad {
    let old_values = keypad.data & 0xF;
    let new_values = selected(keypad);
    let interrupt = keypad.interrupt
        | match old_values == 0xF && new_values != 0xF {
            true => 0x10,
            false => 0,
        };
    Keypad { data: (keypad.data & 0xF0) | new_values, interrupt, ..keypad }
}

// The low nibble as read: 0xF masked by each row the register currently selects.
fn selected(keypad: Keypad) -> u8 {
    let from_row0 = match keypad.data & 0x10 == 0x00 {
        true => keypad.row0,
        false => 0xF,
    };
    let from_row1 = match keypad.data & 0x20 == 0x00 {
        true => keypad.row1,
        false => 0xF,
    };
    0xF & from_row0 & from_row1
}

// Clears the pressed key's bit in its row.
fn press(keypad: Keypad, key: Keypad_Key) -> Keypad {
    match key {
        Keypad_Key::Right => Keypad { row0: keypad.row0 & !(1 << 0), ..keypad },
        Keypad_Key::Left => Keypad { row0: keypad.row0 & !(1 << 1), ..keypad },
        Keypad_Key::Up => Keypad { row0: keypad.row0 & !(1 << 2), ..keypad },
        Keypad_Key::Down => Keypad { row0: keypad.row0 & !(1 << 3), ..keypad },
        Keypad_Key::A => Keypad { row1: keypad.row1 & !(1 << 0), ..keypad },
        Keypad_Key::B => Keypad { row1: keypad.row1 & !(1 << 1), ..keypad },
        Keypad_Key::Select => Keypad { row1: keypad.row1 & !(1 << 2), ..keypad },
        Keypad_Key::Start => Keypad { row1: keypad.row1 & !(1 << 3), ..keypad },
    }
}

// Sets the released key's bit in its row.
fn release(keypad: Keypad, key: Keypad_Key) -> Keypad {
    match key {
        Keypad_Key::Right => Keypad { row0: keypad.row0 | (1 << 0), ..keypad },
        Keypad_Key::Left => Keypad { row0: keypad.row0 | (1 << 1), ..keypad },
        Keypad_Key::Up => Keypad { row0: keypad.row0 | (1 << 2), ..keypad },
        Keypad_Key::Down => Keypad { row0: keypad.row0 | (1 << 3), ..keypad },
        Keypad_Key::A => Keypad { row1: keypad.row1 | (1 << 0), ..keypad },
        Keypad_Key::B => Keypad { row1: keypad.row1 | (1 << 1), ..keypad },
        Keypad_Key::Select => Keypad { row1: keypad.row1 | (1 << 2), ..keypad },
        Keypad_Key::Start => Keypad { row1: keypad.row1 | (1 << 3), ..keypad },
    }
}

#[cfg(test)]
mod tests {
    use crate::keypad;

    #[test]
    fn button_keys_report_on_the_button_row() {
        let buttons =
            [keypad::Keypad_Key::A, keypad::Keypad_Key::B, keypad::Keypad_Key::Select, keypad::Keypad_Key::Start];
        (0..buttons.len()).for_each(|i| {
            let pad = keypad::keydown(keypad::new(), buttons[i]);
            // Both rows selected: the pressed button bit is low.
            let pad = keypad::write_byte(pad, 0x00);
            assert_eq!(keypad::read_byte(&pad), 0xCF & !(1 << i));
            // Selecting only the direction row hides the button.
            let pad = keypad::write_byte(pad, 0x10);
            assert_eq!(keypad::read_byte(&pad), 0xDF & !(1 << i));
            let pad = keypad::write_byte(pad, 0x20);
            assert_eq!(keypad::read_byte(&pad), 0xEF);
        });
    }

    #[test]
    fn direction_keys_report_on_the_direction_row() {
        let directions =
            [keypad::Keypad_Key::Right, keypad::Keypad_Key::Left, keypad::Keypad_Key::Up, keypad::Keypad_Key::Down];
        (0..directions.len()).for_each(|i| {
            let pad = keypad::keydown(keypad::new(), directions[i]);
            let pad = keypad::write_byte(pad, 0x00);
            assert_eq!(keypad::read_byte(&pad), 0xCF & !(1 << i));
            let pad = keypad::write_byte(pad, 0x20);
            assert_eq!(keypad::read_byte(&pad), 0xEF & !(1 << i));
        });
    }
}
