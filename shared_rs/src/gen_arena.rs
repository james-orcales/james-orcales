//! Generational arena: value storage addressed by a versioned [`Handle`], with
//! removal and slot reuse. Each slot carries a generation; `remove` bumps it, so
//! a handle to a removed occupant is rejected after the slot is reused (no
//! use-after-free / ABA). Use the `arena` library if you never remove and want
//! the smaller footprint.
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

/// Handle into an [`Arena`]. It pairs the slot index with the generation the slot
/// had when the handle was minted; a removal bumps the slot's generation so any
/// handle minted before that removal no longer matches and is rejected.
#[derive(Copy, Clone, PartialEq, Eq, Hash, Debug)]
pub struct Handle {
    pub index: u32,
    pub generation: u32,
}

/// The contents of a slot: a live value or a hole in the free list.
#[derive(Clone, Debug)]
pub enum Entry<T> {
    Occupied(T),
    /// A freed slot. `next_free` is the index of the next free slot, forming a
    /// singly linked stack of holes rooted at [`Arena::free_head`].
    Free { next_free: Option<u32> },
}

/// A slot tracks its current generation alongside its entry. The generation is
/// bumped on every removal so reused slots reject handles from prior occupants.
#[derive(Clone, Debug)]
pub struct Slot<T> {
    pub generation: u32,
    pub entry: Entry<T>,
}

/// Generational value storage with removal and slot reuse. Freed slots are
/// recycled via a free list; each slot carries a generation so a handle to a
/// prior occupant is rejected after reuse.
#[derive(Clone, Debug)]
pub struct Arena<T> {
    pub slots: Vec<Slot<T>>,
    /// Index of the most recently freed slot, or `None` if no slot is free.
    pub free_head: Option<u32>,
    /// Count of occupied slots (not the length of `slots`).
    pub len: usize,
}

pub fn new<T>() -> Arena<T> {
    Arena { slots: Vec::new(), free_head: None, len: 0 }
}

/// Insert `value`, reusing a freed slot if one is available (preserving its
/// current generation) and otherwise appending a fresh slot at generation 0.
pub fn insert<T>(mut arena: Arena<T>, value: T) -> (Arena<T>, Handle) {
    arena.len += 1;
    match arena.free_head {
        Some(index) => {
            let slot = &mut arena.slots[index as usize];
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
            let index = arena.slots.len() as u32;
            arena.slots.push(Slot { generation: 0, entry: Entry::Occupied(value) });
            let handle = Handle { index, generation: 0 };
            (arena, handle)
        }
    }
}

/// Remove the value addressed by `handle` if the handle is still valid, push the
/// slot onto the free list, and bump its generation so every existing handle to
/// that slot becomes stale. Returns `(arena, None)` unchanged if the handle is
/// out of bounds, points at a freed slot, or has a stale generation.
pub fn remove<T>(mut arena: Arena<T>, handle: Handle) -> (Arena<T>, Option<T>) {
    let index = handle.index as usize;
    let valid = match arena.slots.get(index) {
        Some(slot) => {
            slot.generation == handle.generation && matches!(slot.entry, Entry::Occupied(_))
        }
        None => false,
    };
    if !valid {
        return (arena, None);
    }

    let slot = &mut arena.slots[index];
    // Bumping the generation is what defeats the ABA problem: a future occupant
    // of this slot will have a higher generation, so `handle` no longer matches.
    slot.generation += 1;
    let old_entry = std::mem::replace(&mut slot.entry, Entry::Free { next_free: arena.free_head });
    arena.free_head = Some(handle.index);
    arena.len -= 1;

    let value = match old_entry {
        Entry::Occupied(value) => value,
        // We checked `Occupied` above and hold the only mutable borrow since.
        Entry::Free { .. } => unreachable!("validated slot was not occupied"),
    };
    (arena, Some(value))
}

/// Read the value behind `handle`, handing the borrow to `reader` and returning
/// its owned result. `None` when the slot is empty OR the generation does not
/// match (the stale-handle protection). A visitor, not `-> &T`, because the
/// dialect bans reference returns.
pub fn with<T, Result_Type>(
    arena: &Arena<T>,
    handle: Handle,
    reader: impl FnOnce(&T) -> Result_Type,
) -> Option<Result_Type> {
    let slot = arena.slots.get(handle.index as usize)?;
    if slot.generation != handle.generation {
        return None;
    }
    match &slot.entry {
        Entry::Occupied(value) => Some(reader(value)),
        Entry::Free { .. } => None,
    }
}

pub fn len<T>(arena: &Arena<T>) -> usize {
    arena.len
}

pub fn is_empty<T>(arena: &Arena<T>) -> bool {
    arena.len == 0
}
