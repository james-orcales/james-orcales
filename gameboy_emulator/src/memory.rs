//! Whole-value updates for the machine's random-access byte regions (WRAM, VRAM,
//! OAM, HRAM, cartridge RAM, framebuffer). The dialect bans `mut`, so a byte cannot
//! be stored in place; each write rebuilds the region. Concentrating the rebuild
//! here keeps the immutability tax measurable in one spot: the arena primitives only
//! append or remove whole objects (no in-place `set`), and structural sharing across
//! versions needs the banned `Rc`/`Arc`, so an obviously-correct whole-region copy
//! is the honest representation.

/// Returns `region` with the byte at `index` replaced by `value`.
pub fn write(region: &[u8], index: usize, value: u8) -> Vec<u8> {
    write_slice(region, index, &[value])
}

/// Returns `region` with `bytes` written starting at `start`, overwriting the bytes
/// already there. The batched form for block copies (OAM DMA, HDMA rows, a rendered
/// scanline) so a run of writes costs one rebuild, not one per byte.
pub fn write_slice(region: &[u8], start: usize, bytes: &[u8]) -> Vec<u8> {
    // Immutability tax: an O(k) in-place block store becomes an O(n) rebuild of the
    // whole region, since a mut-free array has no in-place set and sharing the
    // untouched span across versions would need the banned `Rc`/`Arc`.
    [&region[..start], bytes, &region[start + bytes.len()..]].concat()
}

#[cfg(test)]
mod tests {
    use crate::memory;

    #[test]
    fn write_replaces_one_byte_by_value() {
        let region = vec![1u8, 2, 3, 4];
        let updated = memory::write(&region, 2, 99);
        assert_eq!(updated, vec![1, 2, 99, 4]);
        // Value semantics: the source region is untouched, never aliased.
        assert_eq!(region, vec![1, 2, 3, 4]);
    }

    #[test]
    fn write_slice_overwrites_a_run() {
        let region = vec![0u8; 6];
        let updated = memory::write_slice(&region, 2, &[7, 8, 9]);
        assert_eq!(updated, vec![0, 0, 7, 8, 9, 0]);
    }

    #[test]
    fn write_slice_at_end_leaves_empty_tail() {
        let region = vec![1u8, 2, 3];
        let updated = memory::write_slice(&region, 1, &[8, 9]);
        assert_eq!(updated, vec![1, 8, 9]);
    }
}
