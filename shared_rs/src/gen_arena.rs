//! Generational arena: value storage addressed by a versioned [`Handle`], with
//! removal and slot reuse. Each slot carries a generation; `remove` bumps it, so
//! a handle to a removed occupant is rejected after the slot is reused (no
//! use-after-free / ABA). Use the `arena` library if you never remove and want
//! the smaller footprint.
//!
//! The handle's integer width is generic (`u8`/`u16`/`u32`/`u64`/`usize`, see
//! [`arena::Index_Integer`]), defaulting to `u32` so existing callers that
//! never name a width are unaffected. `insert` panics if the arena has already
//! grown past what the chosen width can address. A narrow generation width
//! also narrows the stale-handle protection: after `2^width` reuses of one
//! slot, the generation wraps and a very old handle can alias the current
//! occupant again — real, but irrelevant at the default `u32` (four billion
//! reuses of one slot).
//!
//! Trusted primitive: `insert` and `remove` are the functions the linter
//! whitelists for `mut` (the in-place O(1) growth). Everything else is mut-free;
//! reads use a visitor closure since the dialect bans `-> &T` returns.
//!
//! ```
//! use shared_rs::gen_arena;
//! let store: gen_arena::Arena<i32> = gen_arena::new();
//! let (store, h) = gen_arena::insert(store, 8);
//! let (store, removed) = gen_arena::remove(store, h);
//! assert_eq!(removed, Some(8));
//! assert_eq!(gen_arena::with(&store, h, |value| *value), None); // stale rejected
//! ```

use crate::arena;
use std::marker;
use std::mem;

/// Handle into an [`Arena`]. It pairs the slot index with the generation the slot
/// had when the handle was minted; a removal bumps the slot's generation so any
/// handle minted before that removal no longer matches and is rejected.
#[derive(Copy, Clone, PartialEq, Eq, Hash, Debug)]
pub struct Handle<I: arena::Index_Integer = u32> {
    pub index: I,
    pub generation: I,
}

/// The contents of a slot: a live value or a hole in the free list.
#[derive(Clone, Debug)]
pub enum Entry<T, I: arena::Index_Integer = u32> {
    Occupied(T),
    /// A freed slot. `next_free` is the index of the next free slot, forming a
    /// singly linked stack of holes rooted at [`Arena::free_head`].
    Free { next_free: Option<I> },
}

/// A slot tracks its current generation alongside its entry. The generation is
/// bumped on every removal so reused slots reject handles from prior occupants.
#[derive(Clone, Debug)]
pub struct Slot<T, I: arena::Index_Integer = u32> {
    pub generation: I,
    pub entry: Entry<T, I>,
}

/// Generational value storage with removal and slot reuse. Freed slots are
/// recycled via a free list; each slot carries a generation so a handle to a
/// prior occupant is rejected after reuse.
#[derive(Clone, Debug)]
pub struct Arena<T, I: arena::Index_Integer = u32> {
    pub slots: Vec<Slot<T, I>>,
    /// Index of the most recently freed slot, or `None` if no slot is free.
    pub free_head: Option<I>,
    /// Count of occupied slots (not the length of `slots`).
    pub len: usize,
    pub phantom: marker::PhantomData<I>,
}

pub fn new<T, I: arena::Index_Integer>() -> Arena<T, I> {
    Arena { slots: Vec::new(), free_head: None, len: 0, phantom: marker::PhantomData }
}

/// Insert `value`, reusing a freed slot if one is available (preserving its
/// current generation) and otherwise appending a fresh slot at generation 0.
/// Panics if the arena has already grown past what `I` can represent.
pub fn insert<T, I>(mut arena: Arena<T, I>, value: T) -> (Arena<T, I>, Handle<I>)
where
    I: arena::Index_Integer + TryFrom<usize>,
    usize: TryFrom<I>,
{
    arena.len += 1;
    match arena.free_head {
        Some(index) => {
            let index_usize =
                usize::try_from(index).unwrap_or_else(|_| unreachable!("a free_head index always fits usize"));
            let slot = &mut arena.slots[index_usize];
            // Unlink this slot from the free list before overwriting it; its
            // `next_free` becomes the new head.
            let next_free = match slot.entry {
                Entry::Free { next_free } => next_free,
                // The free list only ever links Free slots together.
                Entry::Occupied(_) => unreachable!("free_head pointed at an occupied slot"),
            };
            slot.entry = Entry::Occupied(value);
            arena.free_head = next_free;
            let handle = Handle { index, generation: slot.generation };
            (arena, handle)
        }
        None => {
            let index = I::try_from(arena.slots.len())
                .unwrap_or_else(|_| panic!("arena index exceeds its representable range"));
            let generation = zero::<I>();
            arena.slots.push(Slot { generation, entry: Entry::Occupied(value) });
            let handle = Handle { index, generation };
            (arena, handle)
        }
    }
}

/// The zero value of `I`, used as a fresh slot's starting generation.
/// Converting `0usize` into any of the supported unsigned widths never fails.
fn zero<I: arena::Index_Integer + TryFrom<usize>>() -> I {
    I::try_from(0).unwrap_or_else(|_| unreachable!("zero fits in any unsigned integer width"))
}

/// Bumps a generation by one, wrapping to zero at `I`'s max — the only way
/// converting `generation + 1` back into `I` can fail is when `generation`
/// was already `I`'s max, in which case wrapping to zero is exactly the
/// correct modular-arithmetic result.
fn bump_generation<I>(generation: I) -> I
where
    I: arena::Index_Integer + TryFrom<usize>,
    usize: TryFrom<I>,
{
    let as_usize =
        usize::try_from(generation).unwrap_or_else(|_| unreachable!("a stored generation always fits usize"));
    I::try_from(as_usize + 1).unwrap_or_else(|_| zero())
}

/// Remove the value addressed by `handle` if the handle is still valid, push the
/// slot onto the free list, and bump its generation so every existing handle to
/// that slot becomes stale. Returns `(arena, None)` unchanged if the handle is
/// out of bounds, points at a freed slot, or has a stale generation.
pub fn remove<T, I>(mut arena: Arena<T, I>, handle: Handle<I>) -> (Arena<T, I>, Option<T>)
where
    I: arena::Index_Integer + TryFrom<usize> + PartialEq,
    usize: TryFrom<I>,
{
    let index = match usize::try_from(handle.index) {
        Ok(index) => index,
        Err(_) => return (arena, None),
    };
    let valid = match arena.slots.get(index) {
        Some(slot) => slot.generation == handle.generation && matches!(slot.entry, Entry::Occupied(_)),
        None => false,
    };
    if !valid {
        return (arena, None);
    }

    let slot = &mut arena.slots[index];
    // Bumping the generation is what defeats the ABA problem: a future occupant
    // of this slot will have a higher generation, so `handle` no longer matches.
    slot.generation = bump_generation(slot.generation);
    let old_entry = mem::replace(&mut slot.entry, Entry::Free { next_free: arena.free_head });
    arena.free_head = Some(handle.index);
    arena.len -= 1;

    let value = match old_entry {
        Entry::Occupied(value) => value,
        // We checked `Occupied` above and hold the only mutable borrow since.
        Entry::Free { .. } => unreachable!("validated slot was not occupied"),
    };
    (arena, Some(value))
}

/// Overwrites the value behind `handle` in place, keeping the handle valid —
/// unlike `remove` followed by `insert`, this does not bump the slot's
/// generation, since the occupant is being replaced, not vacated. Returns
/// `(arena, false)` unchanged if the handle is out of bounds, points at a
/// freed slot, or has a stale generation.
pub fn update<T, I>(mut arena: Arena<T, I>, handle: Handle<I>, value: T) -> (Arena<T, I>, bool)
where
    I: arena::Index_Integer + PartialEq,
    usize: TryFrom<I>,
{
    let index = match usize::try_from(handle.index) {
        Ok(index) => index,
        Err(_) => return (arena, false),
    };
    let valid = match arena.slots.get(index) {
        Some(slot) => slot.generation == handle.generation && matches!(slot.entry, Entry::Occupied(_)),
        None => false,
    };
    if !valid {
        return (arena, false);
    }
    arena.slots[index].entry = Entry::Occupied(value);
    (arena, true)
}

/// Read the value behind `handle`, handing the borrow to `reader` and returning
/// its owned result. `None` when the slot is empty OR the generation does not
/// match (the stale-handle protection). A visitor, not `-> &T`, because the
/// dialect bans reference returns.
pub fn with<T, I, Result_Type>(
    arena: &Arena<T, I>,
    handle: Handle<I>,
    reader: impl FnOnce(&T) -> Result_Type,
) -> Option<Result_Type>
where
    I: arena::Index_Integer + PartialEq,
    usize: TryFrom<I>,
{
    let index = usize::try_from(handle.index).ok()?;
    let slot = arena.slots.get(index)?;
    if slot.generation != handle.generation {
        return None;
    }
    match &slot.entry {
        Entry::Occupied(value) => Some(reader(value)),
        Entry::Free { .. } => None,
    }
}

pub fn len<T, I: arena::Index_Integer>(arena: &Arena<T, I>) -> usize {
    arena.len
}

pub fn is_empty<T, I: arena::Index_Integer>(arena: &Arena<T, I>) -> bool {
    arena.len == 0
}
