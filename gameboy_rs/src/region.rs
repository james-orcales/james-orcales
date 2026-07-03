//! A paged, allocate-once memory region backed by the blessed `gen_arena`. rboy poked
//! a byte into a flat array in O(1); the dialect bans in-place mutation, so R1
//! (`memory.rs`) rebuilt the whole region on every write, O(n). Here the region is a
//! fixed grid of `PAGE`-byte pages, each a slot in a `gen_arena` inserted once at
//! construction (the only allocation). A write rebuilds just the one page on the stack
//! (mut-free, no heap) and overwrites its slot via `gen_arena::update`, so the tax
//! shrinks from the whole region to a single page. It is byte-identical to R1: reads
//! return the same bytes, and the arena's move-consuming API means the pre-write state
//! is never aliased, so reusing the backing store is unobservable.

use shared_rs::gen_arena;
use std::array;

// Page size in bytes. A power of two so address -> (page, offset) is shift/mask.
// Balances the per-write page rebuild against the per-read arena visit; tunable.
const PAGE: usize = 64;
const PAGE_MASK: usize = PAGE - 1;
const PAGE_SHIFT: u32 = PAGE.trailing_zeros();

/// A memory region as a grid of fixed-size pages in a generational arena. Every page
/// is inserted at construction and never removed, so page `p` lives permanently at
/// slot index `p`, generation 0.
#[derive(Clone, Debug)]
pub struct Region {
    pub pages: gen_arena::Arena<[u8; PAGE]>,
}

/// Builds a region addressing `size` bytes (rounded up to a whole page), every byte
/// set to `value`. The page inserts here are the region's only heap allocation for its
/// whole lifetime.
pub fn filled(size: usize, value: u8) -> Region {
    let page_count = size.div_ceil(PAGE);
    let pages: gen_arena::Arena<[u8; PAGE]> =
        (0..page_count).fold(gen_arena::new(), |arena, _| gen_arena::insert(arena, [value; PAGE]).0);
    Region { pages }
}

/// Builds a zeroed region addressing `size` bytes.
pub fn new(size: usize) -> Region {
    filled(size, 0)
}

/// Rebuilds a region from a flat byte image (save-state restore, battery load).
pub fn from_bytes(bytes: &[u8]) -> Region {
    write_slice(new(bytes.len()), 0, bytes)
}

/// The whole region as a flat byte image (framebuffer output, save-state encode). Off
/// the hot path — reads every byte through the arena visitor.
pub fn to_vec(region: &Region) -> Vec<u8> {
    (0..gen_arena::len(&region.pages) * PAGE).map(|addr| read(region, addr)).collect()
}

/// The byte at `addr`. Panics on an out-of-range address, matching a bare `region[addr]`.
pub fn read(region: &Region, addr: usize) -> u8 {
    gen_arena::with(&region.pages, page_handle(addr >> PAGE_SHIFT), |page| page[addr & PAGE_MASK]).unwrap()
}

/// Returns `region` with the byte at `addr` replaced, rebuilding only that byte's page.
pub fn write(region: Region, addr: usize, value: u8) -> Region {
    let handle = page_handle(addr >> PAGE_SHIFT);
    let off = addr & PAGE_MASK;
    let rebuilt: [u8; PAGE] = gen_arena::with(&region.pages, handle, |page| {
        array::from_fn(|i| match i == off {
            true => value,
            false => page[i],
        })
    })
    .unwrap();
    let (pages, _) = gen_arena::update(region.pages, handle, rebuilt);
    Region { pages }
}

/// Returns `region` with `bytes` written starting at `start`, rebuilding only the
/// pages the range touches. The batched form for OAM DMA, HDMA rows, and scanlines.
pub fn write_slice(region: Region, start: usize, bytes: &[u8]) -> Region {
    match bytes.is_empty() {
        true => region,
        false => {
            let first = start >> PAGE_SHIFT;
            let last = (start + bytes.len() - 1) >> PAGE_SHIFT;
            (first..=last).fold(region, |acc, page| write_page(acc, page, start, bytes))
        }
    }
}

// Rebuilds one page, taking each byte from `bytes` where the range covers it and from
// the old page elsewhere.
fn write_page(region: Region, page: usize, start: usize, bytes: &[u8]) -> Region {
    let handle = page_handle(page);
    let base = page << PAGE_SHIFT;
    let end = start + bytes.len();
    let rebuilt: [u8; PAGE] = gen_arena::with(&region.pages, handle, |old| {
        array::from_fn(|i| match base + i >= start && base + i < end {
            true => bytes[base + i - start],
            false => old[i],
        })
    })
    .unwrap();
    let (pages, _) = gen_arena::update(region.pages, handle, rebuilt);
    Region { pages }
}

// The permanent handle for page `p`: slot `p`, generation 0 (pages are never removed).
fn page_handle(page: usize) -> gen_arena::Handle {
    gen_arena::Handle { index: page as u32, generation: 0 }
}

#[cfg(test)]
mod tests {
    use crate::region;

    #[test]
    fn write_then_read_round_trips() {
        let filled = region::write(region::write(region::new(0x8000), 0x1234, 0xAB), 0x0001, 0x01);
        assert_eq!(region::read(&filled, 0x1234), 0xAB);
        assert_eq!(region::read(&filled, 0x0001), 0x01);
        assert_eq!(region::read(&filled, 0x1235), 0x00);
    }

    #[test]
    fn write_slice_spans_pages() {
        let bytes: Vec<u8> = (0..300u32).map(|i| (i & 0xFF) as u8).collect();
        let filled = region::write_slice(region::new(0x400), 200, &bytes);
        assert_eq!(region::read(&filled, 200), bytes[0]);
        assert_eq!(region::read(&filled, 255), bytes[55]);
        assert_eq!(region::read(&filled, 256), bytes[56]);
        assert_eq!(region::read(&filled, 499), bytes[299]);
        assert_eq!(region::read(&filled, 199), 0);
    }

    #[test]
    fn to_vec_and_from_bytes_round_trip() {
        let filled = region::write(region::write(region::new(0x400), 5, 0x42), 1000, 0x99);
        let bytes = region::to_vec(&filled);
        assert_eq!(bytes.len(), 0x400);
        assert_eq!(bytes[5], 0x42);
        assert_eq!(bytes[1000], 0x99);
        assert_eq!(region::to_vec(&region::from_bytes(&bytes)), bytes);
    }
}
