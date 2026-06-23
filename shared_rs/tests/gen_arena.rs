use shared_rs::gen_arena;

#[test]
fn insert_then_with_returns_value() {
    let store: gen_arena::Arena<&str> = gen_arena::new();
    let (store, h) = gen_arena::insert(store, "hello");
    assert_eq!(gen_arena::with(&store, h, |value| *value), Some("hello"));
}

#[test]
fn remove_returns_value_and_then_with_is_none() {
    let store: gen_arena::Arena<i32> = gen_arena::new();
    let (store, h) = gen_arena::insert(store, 99);
    let (store, removed) = gen_arena::remove(store, h);
    assert_eq!(removed, Some(99));
    assert_eq!(gen_arena::with(&store, h, |value| *value), None);
}

/// After a slot is removed and reused, the OLD handle still resolves to `None`
/// even though its index now holds live data — the stale-handle / ABA protection
/// the plain arena cannot offer.
#[test]
fn stale_handle_does_not_alias_reused_slot() {
    let store: gen_arena::Arena<&str> = gen_arena::new();
    let (store, stale) = gen_arena::insert(store, "old");
    let (store, _removed) = gen_arena::remove(store, stale);

    let (store, fresh) = gen_arena::insert(store, "new");
    assert_eq!(fresh.index, stale.index);
    assert_ne!(fresh.generation, stale.generation);
    assert_eq!(gen_arena::with(&store, stale, |value| *value), None);
    assert_eq!(gen_arena::with(&store, fresh, |value| *value), Some("new"));
}

#[test]
fn len_decrements_on_remove() {
    let store: gen_arena::Arena<u8> = gen_arena::new();
    let (store, a) = gen_arena::insert(store, 10);
    let (store, _b) = gen_arena::insert(store, 20);
    assert_eq!(gen_arena::len(&store), 2);
    let (store, _) = gen_arena::remove(store, a);
    assert_eq!(gen_arena::len(&store), 1);
    assert!(!gen_arena::is_empty(&store));
}
