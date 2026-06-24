struct Holder {
    // Reference hidden in a parenthesized Fn argument.
    pub callback: Box<dyn Fn(&u8)>,
    // Reference hidden in a bare fn type.
    pub func: fn(&u8) -> u8,
}

// Reference hidden in an associated-type binding.
fn make() -> impl Iterator<Item = &'static u8> {
    [].into_iter()
}
