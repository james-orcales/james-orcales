//! Append-only arena: value storage addressed by a bare integer [`Handle`].
//! `insert` appends and returns a handle; there is no removal, so a handle can
//! never go stale, and the handle is just an index. Smaller per element than the
//! generational arena (no slot generation), at the cost of never reclaiming a
//! slot — use the `gen_arena` library when you need removal.
//!
//! The handle's integer width is generic (`u8`/`u16`/`u32`/`u64`/`usize`, see
//! [`Index_Integer`]), defaulting to `u32` so existing callers that never name
//! a width are unaffected. `insert` panics if the arena has already grown past
//! what the chosen width can address — pick a wider one rather than silently
//! colliding two elements onto the same handle.
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

use std::marker;

/// Unsigned integer types usable as an arena's handle width. A marker trait
/// (no methods — the dialect bans trait methods): real behavior comes from
/// the `Copy` supertrait and the `TryFrom<usize>`/`usize: TryFrom<Self>`
/// bounds each function declares where it actually needs a conversion. No
/// signed integers — a negative index never makes sense.
pub trait Index_Integer: Copy {}
impl Index_Integer for u8 {}
impl Index_Integer for u16 {}
impl Index_Integer for u32 {}
impl Index_Integer for u64 {}
impl Index_Integer for usize {}

/// Handle into an [`Arena`]: a bare slot index, valid for the arena's lifetime.
#[derive(Copy, Clone, PartialEq, Eq, Hash, Debug)]
pub struct Handle<I: Index_Integer = u32>(pub I);

/// Append-only value storage: `insert` appends and returns a [`Handle`]. The
/// `phantom` field ties the handle width `I` to this arena at the type
/// level (an `Arena<T, u8>` and its handles can't be mixed with another
/// arena's `u16` handles) — it carries no runtime data.
#[derive(Clone, Debug)]
pub struct Arena<T, I: Index_Integer = u32> {
    pub items: Vec<T>,
    pub phantom: marker::PhantomData<I>,
}

pub fn new<T, I: Index_Integer>() -> Arena<T, I> {
    Arena { items: Vec::new(), phantom: marker::PhantomData }
}

/// Appends `value` and returns its handle. Panics if the arena has already
/// grown past what `I` can represent — the caller picked too narrow a width,
/// and silently wrapping would collide two elements onto the same handle.
pub fn insert<T, I>(mut arena: Arena<T, I>, value: T) -> (Arena<T, I>, Handle<I>)
where
    I: Index_Integer + TryFrom<usize>,
{
    let index = I::try_from(arena.items.len())
        .unwrap_or_else(|_| panic!("arena index exceeds its representable range"));
    arena.items.push(value);
    (arena, Handle(index))
}

/// Overwrites the value at `handle` in place. Returns `(arena, false)`
/// unchanged if the handle is out of bounds — there is no generation to go
/// stale, since nothing is ever removed.
pub fn update<T, I>(mut arena: Arena<T, I>, handle: Handle<I>, value: T) -> (Arena<T, I>, bool)
where
    I: Index_Integer,
    usize: TryFrom<I>,
{
    match usize::try_from(handle.0) {
        Ok(index) if index < arena.items.len() => {
            arena.items[index] = value;
            (arena, true)
        }
        _ => (arena, false),
    }
}

pub fn with<T, I, Result_Type>(
    arena: &Arena<T, I>,
    handle: Handle<I>,
    reader: impl FnOnce(&T) -> Result_Type,
) -> Option<Result_Type>
where
    I: Index_Integer,
    usize: TryFrom<I>,
{
    let index = usize::try_from(handle.0).ok()?;
    arena.items.get(index).map(reader)
}

pub fn len<T, I: Index_Integer>(arena: &Arena<T, I>) -> usize {
    arena.items.len()
}

pub fn is_empty<T, I: Index_Integer>(arena: &Arena<T, I>) -> bool {
    arena.items.is_empty()
}
