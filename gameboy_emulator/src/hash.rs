//! FNV-1a over a byte buffer: the deterministic framebuffer fingerprint the headless
//! runner and the differential harness compare. Hand-rolled so the crate needs no
//! external hashing dependency (the workspace builds offline from a vendored tree).
//! A fold, since the dialect has no mutable accumulator.

// The 64-bit FNV-1a offset basis and prime, fixed by the algorithm's definition.
const OFFSET_BASIS: u64 = 0xcbf2_9ce4_8422_2325;
const PRIME: u64 = 0x0000_0100_0000_01b3;

/// The 64-bit FNV-1a hash of `bytes`.
pub fn fnv1a(bytes: &[u8]) -> u64 {
    bytes.iter().fold(OFFSET_BASIS, |hash, &byte| (hash ^ byte as u64).wrapping_mul(PRIME))
}
