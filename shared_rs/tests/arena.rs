use shared_rs::arena;

#[test]
fn insert_then_with_returns_each_value() {
    let store: arena::Arena<&str> = arena::new();
    let (store, a) = arena::insert(store, "alpha");
    let (store, b) = arena::insert(store, "beta");

    assert_eq!(arena::with(&store, a, |value| *value), Some("alpha"));
    assert_eq!(arena::with(&store, b, |value| *value), Some("beta"));
    assert_eq!(arena::len(&store), 2);
}

#[test]
fn len_tracks_inserts() {
    let store: arena::Arena<u8> = arena::new();
    assert!(arena::is_empty(&store));
    let (store, _) = arena::insert(store, 1);
    let (store, _) = arena::insert(store, 2);
    assert_eq!(arena::len(&store), 2);
    assert!(!arena::is_empty(&store));
}

#[test]
fn with_out_of_range_returns_none() {
    let store: arena::Arena<i32> = arena::new();
    let (store, _h) = arena::insert(store, 42);
    // A handle minted from a longer arena indexes past this one's end.
    let (longer, far) = {
        let (a, _) = arena::insert(arena::new(), 1);
        let (a, _) = arena::insert(a, 2);
        arena::insert(a, 3)
    };
    let _ = longer;
    assert_eq!(arena::with(&store, far, |value| *value), None);
}

#[test]
fn update_changes_the_value_read_back_through_the_same_handle() {
    let store: arena::Arena<&str> = arena::new();
    let (store, h) = arena::insert(store, "old");
    let (store, updated) = arena::update(store, h, "new");
    assert!(updated);
    assert_eq!(arena::with(&store, h, |value| *value), Some("new"));
}

#[test]
fn update_on_an_out_of_bounds_handle_is_a_no_op() {
    let store: arena::Arena<i32> = arena::new();
    let far = arena::Handle(7);
    let (store, updated) = arena::update(store, far, 1);
    assert!(!updated);
    assert!(arena::is_empty(&store));
}

#[test]
fn narrow_width_insert_with_update_round_trip() {
    let store: arena::Arena<&str, u8> = arena::new();
    let (store, a) = arena::insert(store, "alpha");
    let (store, b) = arena::insert(store, "beta");
    let (store, updated) = arena::update(store, a, "gamma");
    assert!(updated);
    assert_eq!(arena::with(&store, a, |value| *value), Some("gamma"));
    assert_eq!(arena::with(&store, b, |value| *value), Some("beta"));
    assert_eq!(arena::len(&store), 2);
}

#[test]
fn narrow_width_fills_every_representable_index() {
    // u8 can represent 256 distinct indices (0..=255): the 256th insert (index
    // 255) must still succeed.
    let mut store: arena::Arena<u32, u8> = arena::new();
    let mut last = arena::Handle(0);
    for i in 0..256u32 {
        let (next_store, handle) = arena::insert(store, i);
        store = next_store;
        last = handle;
    }
    assert_eq!(arena::len(&store), 256);
    assert_eq!(arena::with(&store, last, |value| *value), Some(255));
}

#[test]
#[should_panic]
fn narrow_width_insert_panics_past_representable_range() {
    // The 257th insert needs index 256, which does not fit in a u8.
    let mut store: arena::Arena<u32, u8> = arena::new();
    for i in 0..257u32 {
        let (next_store, _handle) = arena::insert(store, i);
        store = next_store;
    }
}
