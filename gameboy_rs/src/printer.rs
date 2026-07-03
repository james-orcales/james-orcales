//! The Game Boy Printer, a serial peripheral, ported from rboy's `GbPrinter`. rboy
//! drove the packet state machine through `&mut self` and wrote each finished print
//! to a `.pgm` file; here the state threads by value and a finished print is
//! appended to `prints` (the 160xN grey image) rather than written to disk, so the
//! emulation core stays side-effect free. `send` processes one serial byte and
//! returns the status byte the console reads back. The printer is not exercised by
//! the differential corpus (no test ROM prints, and rboy's headless Device does not
//! attach it), so it is a feature-parity port, not a differentially-verified one.

use crate::memory;

const DATA_SIZE: usize = 0x280 * 9;
const PACKET_SIZE: usize = 0x400;

/// The printer's packet-protocol state, receive buffers, and finished prints.
#[derive(Clone, Debug)]
pub struct Printer {
    pub status: u8,
    pub state: u32,
    pub data: Vec<u8>,
    pub packet: Vec<u8>,
    pub count: usize,
    pub datacount: usize,
    pub datasize: usize,
    pub result: u8,
    pub printcount: u8,
    pub prints: Vec<Vec<u8>>,
}

pub fn new() -> Printer {
    Printer {
        status: 0,
        state: 0,
        data: vec![0; DATA_SIZE],
        packet: vec![0; PACKET_SIZE],
        count: 0,
        datacount: 0,
        datasize: 0,
        result: 0,
        printcount: 0,
        prints: Vec::new(),
    }
}

/// The PGM (P5) file bytes for one finished print (160 pixels wide, 4 grey levels),
/// for the composition root to write to disk — the core produces the bytes, the host
/// performs the I/O.
pub fn pgm(image: &[u8]) -> Vec<u8> {
    let height = image.len() / 160;
    [format!("P5 160 {height} 3\n").into_bytes(), image.to_vec()].concat()
}

/// Feeds one serial byte to the printer, returning the printer and the status byte
/// the console reads back on the next transfer.
pub fn send(printer: Printer, value: u8) -> (Printer, u8) {
    let stored =
        Printer { packet: memory::write(&printer.packet, printer.count, value), count: printer.count + 1, ..printer };
    let stepped = step(stored, value);
    let result = stepped.result;
    (stepped, result)
}

// Advances the packet state machine one byte: magic bytes 0x88 0x33, a header, the
// data block, then the checksum and two status-handshake bytes.
fn step(printer: Printer, value: u8) -> Printer {
    match printer.state {
        0 => match value == 0x88 {
            true => Printer { state: 1, ..printer },
            false => reset(printer),
        },
        1 => match value == 0x33 {
            true => Printer { state: 2, ..printer },
            false => reset(printer),
        },
        2 => step_header(printer),
        3 => step_data(printer),
        4 => Printer { state: 5, ..printer },
        5 => step_command(printer),
        6 => Printer { result: 0x81, state: 7, ..printer },
        7 => Printer { result: printer.status, state: 0, count: 0, ..printer },
        _ => reset(printer),
    }
}

// After the six header bytes, latch the data length and pick the data or checksum
// state.
fn step_header(printer: Printer) -> Printer {
    match printer.count == 6 {
        false => printer,
        true => {
            let datasize = printer.packet[4] as usize + ((printer.packet[5] as usize) << 8);
            let state = match datasize > 0 {
                true => 3,
                false => 4,
            };
            Printer { datasize, state, ..printer }
        }
    }
}

// Waits for the whole data block, then moves to the checksum.
fn step_data(printer: Printer) -> Printer {
    match printer.count == printer.datasize + 6 {
        true => Printer { state: 4, ..printer },
        false => printer,
    }
}

// On a valid checksum, run the packet's command.
fn step_command(printer: Printer) -> Printer {
    let commanded = match check_crc(&printer) {
        true => command(printer),
        false => printer,
    };
    Printer { state: 6, ..commanded }
}

// Resets to the idle state between packets.
fn reset(printer: Printer) -> Printer {
    Printer { state: 0, datasize: 0, datacount: 0, count: 0, status: 0, result: 0, ..printer }
}

// Verifies the 16-bit additive checksum over the packet body.
fn check_crc(printer: &Printer) -> bool {
    let crc = (2..(6 + printer.datasize)).fold(0u16, |sum, i| sum.wrapping_add(printer.packet[i] as u16));
    let message = (printer.packet[6 + printer.datasize] as u16)
        .wrapping_add((printer.packet[7 + printer.datasize] as u16) << 8);
    crc == message
}

// Dispatches a packet command: 0x01 init, 0x02 print, 0x04 transfer data.
fn command(printer: Printer) -> Printer {
    match printer.packet[2] {
        0x01 => Printer { datacount: 0, status: 0, ..printer },
        0x02 => render(printer),
        0x04 => receive(printer),
        _ => printer,
    }
}

// Appends the packet's data block (RLE-decompressed when packet[3] is set) to the
// image buffer.
fn receive(printer: Printer) -> Printer {
    let block = match printer.packet[3] != 0 {
        true => decompress(&printer.packet, 6, printer.datasize, Vec::new()),
        false => printer.packet[6..6 + printer.datasize].to_vec(),
    };
    let data = memory::write_slice(&printer.data, printer.datacount, &block);
    Printer { data, datacount: printer.datacount + block.len(), ..printer }
}

// Decompresses the packet's RLE data body into `out`: a control byte with bit 7 set
// repeats the next byte, otherwise it copies a literal run.
fn decompress(packet: &[u8], dataidx: usize, datasize: usize, out: Vec<u8>) -> Vec<u8> {
    match dataidx - 6 < datasize {
        false => out,
        true => {
            let control = packet[dataidx];
            match control & 0x80 != 0 {
                true => {
                    let run = ((control & 0x7F) + 2) as usize;
                    let next = [out, vec![packet[dataidx + 1]; run]].concat();
                    decompress(packet, dataidx + 2, datasize, next)
                }
                false => {
                    let run = (control + 1) as usize;
                    let next = [out, packet[dataidx + 1..dataidx + 1 + run].to_vec()].concat();
                    decompress(packet, dataidx + 1 + run, datasize, next)
                }
            }
        }
    }
}

// Renders the received tile data into a 160xN grey image and appends it to `prints`.
fn render(printer: Printer) -> Printer {
    let height = printer.datacount / 40;
    match height == 0 {
        true => Printer { printcount: printer.printcount + 1, ..printer },
        false => {
            let palette = printer_palette(printer.packet[8]);
            let image: Vec<u8> = (0..height)
                .flat_map(|y| (0..160usize).map(move |x| (y, x)))
                .map(|(y, x)| palette[pixel_color(&printer.data, y, x)] as usize as u8)
                .collect();
            Printer { printcount: printer.printcount + 1, prints: [printer.prints, vec![image]].concat(), ..printer }
        }
    }
}

// The four grey levels from the print palette byte, brightest to darkest inverted.
fn printer_palette(palbyte: u8) -> [u8; 4] {
    [0, 1, 2, 3].map(|i| 3 - ((palbyte >> (2 * i)) & 3))
}

// The 2-bit colour index of one printed pixel from the tile bit-planes.
fn pixel_color(data: &[u8], y: usize, x: usize) -> usize {
    let tile = ((y >> 3) * 20) + (x >> 3);
    let offset = tile * 16 + (y & 7) * 2;
    let bit = 7 - (x & 7);
    (((data[offset] >> bit) & 1) | (((data[offset + 1] >> bit) << 1) & 2)) as usize
}
