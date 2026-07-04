//! The serial link port. rboy handed each transmitted byte to a `SerialCallback`
//! trait object; the dialect bans method-bearing traits, so instead the peer is a
//! `Serial_Target` enum. The default `Capture` appends the byte to an owned `output`
//! log that the headless runner reads back — the blargg pass/fail oracle (the ROMs
//! stream "Passed" through here). `Print` attaches the Game Boy Printer, consuming
//! the byte and handing back its status. Writing SC with the transfer-start and
//! internal-clock bits both set (0x81) transmits SB. An unconnected `Capture` link
//! receives nothing back, so SB is left unchanged and no serial interrupt is raised
//! (rboy's callback returning `None`).

use crate::printer;

/// The peer on the other end of the serial cable: a byte sink that records the
/// stream, or the Game Boy Printer that answers each byte with a status.
#[derive(Clone, Debug)]
pub enum Serial_Target {
    Capture,
    Print(printer::Printer),
}

/// The serial data (SB) and control (SC) registers, the pending interrupt byte, the
/// accumulated transmitted bytes, and the connected peer.
#[derive(Clone, Debug)]
pub struct Serial {
    pub data: u8,
    pub control: u8,
    pub interrupt: u8,
    pub output: Vec<u8>,
    pub target: Serial_Target,
}

pub fn new() -> Serial {
    Serial { data: 0, control: 0, interrupt: 0, output: Vec::new(), target: Serial_Target::Capture }
}

/// Connects the Game Boy Printer to the link port; the composition root calls this
/// when it wants to capture printed images instead of the raw byte stream.
pub fn attach_printer(serial: Serial) -> Serial {
    Serial { target: Serial_Target::Print(printer::new()), ..serial }
}

pub fn read_byte(serial: &Serial, address: u16) -> u8 {
    match address {
        0xFF01 => serial.data,
        // SC: only bits 7 and 0 are meaningful; the middle bits read as 1.
        0xFF02 => serial.control | 0b0111_1110,
        _ => panic!("Serial does not handle read {:04X}", address),
    }
}

pub fn write_byte(serial: Serial, address: u16, value: u8) -> Serial {
    match address {
        0xFF01 => Serial { data: value, ..serial },
        0xFF02 => transmit(serial, value),
        _ => panic!("Serial does not handle write {:04X}", address),
    }
}

// A write to SC. When it requests a transfer on the internal clock (bits 7 and 0),
// the current SB byte goes to the connected peer; otherwise only SC changes.
fn transmit(serial: Serial, value: u8) -> Serial {
    match value & 0x81 == 0x81 {
        true => send_to_target(serial, value),
        false => Serial { control: value, ..serial },
    }
}

// Delivers SB to the peer. `Capture` logs it and leaves SB and the interrupt alone
// (an idle open line); `Print` feeds the printer, takes back its status into SB, and
// raises the transfer-complete interrupt with the transfer-start bit cleared.
fn send_to_target(serial: Serial, value: u8) -> Serial {
    match serial.target {
        Serial_Target::Capture => Serial {
            control: value,
            output: [serial.output, vec![serial.data]].concat(),
            target: Serial_Target::Capture,
            ..serial
        },
        Serial_Target::Print(peer) => {
            let (printer, response) = printer::send(peer, serial.data);
            Serial {
                control: value & !0x80,
                data: response,
                interrupt: serial.interrupt | 0x08,
                target: Serial_Target::Print(printer),
                ..serial
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use crate::serial;

    #[test]
    fn transmit_appends_sb_to_output() {
        let link = serial::write_byte(serial::new(), 0xFF01, b'P');
        let link = serial::write_byte(link, 0xFF02, 0x81);
        assert_eq!(link.output, vec![b'P']);
    }

    #[test]
    fn sc_without_start_bit_does_not_transmit() {
        let link = serial::write_byte(serial::new(), 0xFF01, b'X');
        // Internal clock requested but no transfer-start bit, so nothing is sent.
        let link = serial::write_byte(link, 0xFF02, 0x01);
        assert!(link.output.is_empty());
    }

    #[test]
    fn sc_read_forces_the_unused_bits_high() {
        let link = serial::write_byte(serial::new(), 0xFF02, 0x80);
        assert_eq!(serial::read_byte(&link, 0xFF02), 0xFE);
    }

    #[test]
    fn attached_printer_consumes_bytes_and_raises_interrupt() {
        let link = serial::attach_printer(serial::new());
        let link = serial::write_byte(serial::write_byte(link, 0xFF01, 0x88), 0xFF02, 0x81);
        // The printer consumes the byte, so nothing is logged to the capture output,
        // and the serial transfer-complete interrupt is raised.
        assert!(link.output.is_empty());
        assert_eq!(link.interrupt, 0x08);
    }
}
