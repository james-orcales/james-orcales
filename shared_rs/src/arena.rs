//! Append-only arena: value storage addressed by a bare integer [`Handle`].
//! `insert` appends and returns a handle; there is no removal, so a handle can
//! never go stale, and the handle is just an index. Smaller per element than the
//! generational arena (no slot generation), at the cost of never reclaiming a
//! slot — use the `gen_arena` library when you need removal.
//!
//! Trusted primitive: `insert` is the one function the linter whitelists for
//! `mut` (the in-place O(1) append). Everything else is mut-free; reads use a
//! visitor closure since the dialect bans `-> &T` returns.
//!
//! ```
//! use shared_rs::arena;
//! let store: arena::Arena<i32> = arena::new();
//! let (store, h) = arena::insert(store, 7);
//! assert_eq!(arena::with(&store, h, |value| *value), Some(7));
//! ```

/// Handle into an [`Arena`]: a bare slot index, valid for the arena's lifetime.
#[derive(Copy, Clone, PartialEq, Eq, Hash, Debug)]
pub struct Handle(pub u32);

/// Append-only value storage: `insert` appends and returns a [`Handle`].
#[derive(Clone, Debug)]
pub struct Arena<T> {
    pub items: Vec<T>,
}

pub fn new<T>() -> Arena<T> {
    Arena { items: Vec::new() }
}

pub fn insert<T>(mut arena: Arena<T>, value: T) -> (Arena<T>, Handle) {
    let handle = Handle(arena.items.len() as u32);
    arena.items.push(value);
    (arena, handle)
}

pub fn with<T, Result_Type>(
    arena: &Arena<T>,
    handle: Handle,
    reader: impl FnOnce(&T) -> Result_Type,
) -> Option<Result_Type> {
    arena.items.get(handle.0 as usize).map(reader)
}

pub fn len<T>(arena: &Arena<T>) -> usize {
    arena.items.len()
}

pub fn is_empty<T>(arena: &Arena<T>) -> bool {
    arena.items.is_empty()
}
