//! A value-threaded port of the `blip_buf` band-limited synthesizer (BLEP), from the
//! vendored `blip_buf` crate. rboy mutated the sample accumulator in place through
//! `&mut self`; here the buffer threads by value. The one hot random-write —
//! `add_delta` accumulating a 16-tap kernel from `BL_STEP` at a fixed-point offset —
//! becomes a spliced 16-element window rebuild (the immutability tax); `read_samples`
//! integrates via a `successors` fold. The fixed-point math and the i16 output are
//! ported verbatim, so the samples are bit-identical to rboy's.

use std::iter;

const MAX_RATIO: u64 = 1 << 20;
const PRE_SHIFT: u32 = 32;
const TIME_BITS: u32 = PRE_SHIFT + 20;
const TIME_UNIT: u64 = 1u64 << TIME_BITS;
const BASS_SHIFT: u32 = 9;
const END_FRAME_EXTRA: usize = 2;
const HALF_WIDTH: usize = 8;
const BUF_EXTRA: usize = HALF_WIDTH * 2 + END_FRAME_EXTRA;
const PHASE_BITS: u32 = 5;
const PHASE_COUNT: usize = 1 << PHASE_BITS;
const DELTA_BITS: u32 = 15;
const DELTA_UNIT: i32 = 1 << DELTA_BITS;
const FRAC_BITS: u32 = TIME_BITS - PRE_SHIFT;

/// A resampling buffer: input clocks in, output samples out, threaded by value.
#[derive(Clone, Debug)]
pub struct Blip_Buf {
    pub factor: u64,
    pub offset: u64,
    pub integrator: i32,
    pub avail: usize,
    pub samples: Vec<i32>,
}

/// A buffer holding at most `sample_count` samples, at the maximum clock/sample ratio.
pub fn new(sample_count: usize) -> Blip_Buf {
    let factor = TIME_UNIT / MAX_RATIO;
    Blip_Buf { factor, offset: factor / 2, integrator: 0, avail: 0, samples: vec![0; sample_count + BUF_EXTRA] }
}

/// Sets the input clock and output sample rates (the factor is rounded up, as rboy).
pub fn set_rates(blip: Blip_Buf, clock_rate: f64, sample_rate: f64) -> Blip_Buf {
    let factor = (TIME_UNIT as f64) * sample_rate / clock_rate;
    Blip_Buf { factor: factor.ceil() as u64, ..blip }
}

/// Empties the buffer.
pub fn clear(blip: Blip_Buf) -> Blip_Buf {
    Blip_Buf { offset: blip.factor / 2, avail: 0, integrator: 0, samples: vec![0; blip.samples.len()], ..blip }
}

/// Number of buffered samples available for reading.
pub fn samples_avail(blip: &Blip_Buf) -> usize {
    blip.avail
}

/// Adds a delta (amplitude change) into the buffer at `clock_time`, splicing the
/// 16-tap band-limited step kernel into the sample window at the computed offset.
pub fn add_delta(blip: Blip_Buf, clock_time: u32, delta: i32) -> Blip_Buf {
    let fixed = (((clock_time as u64) * blip.factor + blip.offset) >> PRE_SHIFT) as usize;
    let out_index = blip.avail + (fixed >> FRAC_BITS);
    let phase_shift = FRAC_BITS - PHASE_BITS;
    let phase = (fixed >> phase_shift) & (PHASE_COUNT - 1);
    let phase_rev = PHASE_COUNT - phase;
    let interp = ((fixed >> (phase_shift - DELTA_BITS)) & (DELTA_UNIT as usize - 1)) as i32;
    let delta2 = (delta * interp) >> DELTA_BITS;
    let delta1 = delta - delta2;
    // Immutability tax: rboy's O(16) in-place accumulate becomes an O(n) rebuild of
    // the sample buffer, splicing in the modified 16-element window.
    let window: Vec<i32> = (0..16)
        .map(|i| kernel_tap(&blip.samples, out_index, i, phase, phase_rev, delta1, delta2))
        .collect();
    Blip_Buf { samples: splice(&blip.samples, out_index, &window), ..blip }
}

// One tap of the 16-element kernel window: the left half (0..8) uses the phase pair,
// the right half (8..16) the reversed phase pair, matching rboy's two loops.
fn kernel_tap(samples: &[i32], out_index: usize, i: usize, phase: usize, phase_rev: usize, delta1: i32, delta2: i32) -> i32 {
    let base = samples[out_index + i];
    match i < 8 {
        true => base + BL_STEP[phase][i] * delta1 + BL_STEP[phase + 1][i] * delta2,
        false => {
            let j = 7 - (i - 8);
            base + BL_STEP[phase_rev][j] * delta1 + BL_STEP[phase_rev - 1][j] * delta2
        }
    }
}

/// Ends the time frame at `clock_duration`, making its clocks available as samples
/// and rebasing the offset.
pub fn end_frame(blip: Blip_Buf, clock_duration: u32) -> Blip_Buf {
    let off = (clock_duration as u64) * blip.factor + blip.offset;
    Blip_Buf { avail: blip.avail + (off >> TIME_BITS) as usize, offset: off & (TIME_UNIT - 1), ..blip }
}

/// Reads up to `buf_len` samples as i16, integrating and removing them from the
/// buffer. Returns the drained buffer and the samples.
pub fn read_samples(blip: Blip_Buf, buf_len: usize) -> (Blip_Buf, Vec<i16>) {
    let count = buf_len.min(blip.avail);
    match count == 0 {
        true => (blip, Vec::new()),
        false => integrate(blip, count),
    }
}

// Integrates `count` deltas into i16 samples: a running sum threaded through a
// `successors` fold (mut-free), then removes the consumed samples.
fn integrate(blip: Blip_Buf, count: usize) -> (Blip_Buf, Vec<i16>) {
    let sums: Vec<(usize, i32)> = iter::successors(Some((0usize, blip.integrator)), |&(i, sum)| match i < count {
        true => {
            let sample = clamp_i16(sum >> DELTA_BITS);
            Some((i + 1, sum + blip.samples[i] - (sample << (DELTA_BITS - BASS_SHIFT))))
        }
        false => None,
    })
    .collect();
    let out: Vec<i16> = sums[..count].iter().map(|&(_, sum)| clamp_i16(sum >> DELTA_BITS) as i16).collect();
    (remove_samples(Blip_Buf { integrator: sums[count].1, ..blip }, count), out)
}

// Shifts the retained samples to the front and zeroes the freed tail (rboy's
// memmove+memset), keeping the buffer length fixed.
fn remove_samples(blip: Blip_Buf, count: usize) -> Blip_Buf {
    let remain = (blip.avail + BUF_EXTRA).saturating_sub(count);
    let shifted: Vec<i32> = blip.samples[count..count + remain]
        .iter()
        .copied()
        .chain(iter::repeat_n(0, blip.samples.len() - remain))
        .collect();
    Blip_Buf { avail: blip.avail.saturating_sub(count), samples: shifted, ..blip }
}

// Returns `samples` with a window replaced at `start` (the mut-free buffer rebuild).
fn splice(samples: &[i32], start: usize, window: &[i32]) -> Vec<i32> {
    [&samples[..start], window, &samples[start + window.len()..]].concat()
}

// Clamps to the signed 16-bit range.
fn clamp_i16(n: i32) -> i32 {
    n.clamp(i16::MIN as i32, i16::MAX as i32)
}

// The 33x8 band-limited step response table, ported verbatim from `blip_buf`.
const BL_STEP: [[i32; 8]; 33] = [
    [43, -115, 350, -488, 1136, -914, 5861, 21022],
    [44, -118, 348, -473, 1076, -799, 5274, 21001],
    [45, -121, 344, -454, 1011, -677, 4706, 20936],
    [46, -122, 336, -431, 942, -549, 4156, 20829],
    [47, -123, 327, -404, 868, -418, 3629, 20679],
    [47, -122, 316, -375, 792, -285, 3124, 20488],
    [47, -120, 303, -344, 714, -151, 2644, 20256],
    [46, -117, 289, -310, 634, -17, 2188, 19985],
    [46, -114, 273, -275, 553, 117, 1758, 19675],
    [44, -108, 255, -237, 471, 247, 1356, 19327],
    [43, -103, 237, -199, 390, 373, 981, 18944],
    [42, -98, 218, -160, 310, 495, 633, 18527],
    [40, -91, 198, -121, 231, 611, 314, 18078],
    [38, -84, 178, -81, 153, 722, 22, 17599],
    [36, -76, 157, -43, 80, 824, -241, 17092],
    [34, -68, 135, -3, 8, 919, -476, 16558],
    [32, -61, 115, 34, -60, 1006, -683, 16001],
    [29, -52, 94, 70, -123, 1083, -862, 15422],
    [27, -44, 73, 106, -184, 1152, -1015, 14824],
    [25, -36, 53, 139, -239, 1211, -1142, 14210],
    [22, -27, 34, 170, -290, 1261, -1244, 13582],
    [20, -20, 16, 199, -335, 1301, -1322, 12942],
    [18, -12, -3, 226, -375, 1331, -1376, 12293],
    [15, -4, -19, 250, -410, 1351, -1408, 11638],
    [13, 3, -35, 272, -439, 1361, -1419, 10979],
    [11, 9, -49, 292, -464, 1362, -1410, 10319],
    [9, 16, -63, 309, -483, 1354, -1383, 9660],
    [7, 22, -75, 322, -496, 1337, -1339, 9005],
    [6, 26, -85, 333, -504, 1312, -1280, 8355],
    [4, 31, -94, 341, -507, 1278, -1205, 7713],
    [3, 35, -102, 347, -506, 1238, -1119, 7082],
    [1, 40, -110, 350, -499, 1190, -1021, 6464],
    [0, 43, -115, 350, -488, 1136, -914, 5861],
];
