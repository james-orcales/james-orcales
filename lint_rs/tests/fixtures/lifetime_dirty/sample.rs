fn unused_lifetime<'a>() {}

struct Phantom<'a> {
    pub value: u8,
}
