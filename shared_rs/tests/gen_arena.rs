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

#[test]
fn update_changes_the_value_read_back_through_the_same_handle() {
    let store: gen_arena::Arena<&str> = gen_arena::new();
    let (store, h) = gen_arena::insert(store, "old");
    let (store, updated) = gen_arena::update(store, h, "new");
    assert!(updated);
    assert_eq!(gen_arena::with(&store, h, |value| *value), Some("new"));
}

#[test]
fn update_on_a_stale_handle_is_a_no_op() {
    let store: gen_arena::Arena<&str> = gen_arena::new();
    let (store, stale) = gen_arena::insert(store, "old");
    let (store, _removed) = gen_arena::remove(store, stale);
    let (store, _fresh) = gen_arena::insert(store, "new");

    let (store, updated) = gen_arena::update(store, stale, "clobber");
    assert!(!updated);
    assert_eq!(gen_arena::with(&store, stale, |value| *value), None);
}

#[test]
fn update_on_an_out_of_bounds_handle_is_a_no_op() {
    let store: gen_arena::Arena<i32> = gen_arena::new();
    let far = gen_arena::Handle { index: 7, generation: 0 };
    let (store, updated) = gen_arena::update(store, far, 1);
    assert!(!updated);
    assert!(gen_arena::is_empty(&store));
}

#[test]
fn narrow_width_insert_with_update_round_trip() {
    let store: gen_arena::Arena<&str, u8> = gen_arena::new();
    let (store, a) = gen_arena::insert(store, "alpha");
    let (store, b) = gen_arena::insert(store, "beta");
    let (store, updated) = gen_arena::update(store, a, "gamma");
    assert!(updated);
    assert_eq!(gen_arena::with(&store, a, |value| *value), Some("gamma"));
    assert_eq!(gen_arena::with(&store, b, |value| *value), Some("beta"));
}

#[test]
fn narrow_width_remove_and_reinsert_reuses_the_slot() {
    let store: gen_arena::Arena<&str, u8> = gen_arena::new();
    let (store, stale) = gen_arena::insert(store, "old");
    let (store, _removed) = gen_arena::remove(store, stale);
    let (store, fresh) = gen_arena::insert(store, "new");
    assert_eq!(fresh.index, stale.index);
    assert_ne!(fresh.generation, stale.generation);
    assert_eq!(gen_arena::with(&store, stale, |value| *value), None);
    assert_eq!(gen_arena::with(&store, fresh, |value| *value), Some("new"));
}

#[test]
fn narrow_width_generation_wraps_without_panicking_and_still_rejects_a_midway_handle() {
    let mut store: gen_arena::Arena<i32, u8> = gen_arena::new();
    let (next_store, first_handle) = gen_arena::insert(store, 0);
    store = next_store;
    let mut handle = first_handle;
    let mut midway = first_handle;
    for i in 1..=256u32 {
        let (removed_store, _value) = gen_arena::remove(store, handle);
        let (inserted_store, new_handle) = gen_arena::insert(removed_store, i as i32);
        store = inserted_store;
        handle = new_handle;
        if i == 100 {
            midway = handle;
        }
    }
    // 256 remove-then-reinsert cycles on the same slot wrap u8's generation
    // (0..=255, then back to 0) without panicking. A handle captured partway
    // through (generation 100) no longer matches the slot's current
    // (wrapped) generation and is correctly rejected.
    assert_eq!(gen_arena::with(&store, midway, |value| *value), None);
    // The known, accepted tradeoff of a narrow width: `first_handle`'s
    // generation (0) now coincidentally matches the wrapped-around current
    // generation, so it aliases the live occupant — this is not prevented,
    // only documented (see the module doc). At the default u32 width this
    // never happens in practice (4 billion reuses of one slot).
    assert_eq!(gen_arena::with(&store, first_handle, |value| *value), Some(256));
}

#[test]
#[should_panic]
fn narrow_width_insert_panics_past_representable_range() {
    let mut store: gen_arena::Arena<u32, u8> = gen_arena::new();
    for i in 0..257u32 {
        let (next_store, _handle) = gen_arena::insert(store, i);
        store = next_store;
    }
}
