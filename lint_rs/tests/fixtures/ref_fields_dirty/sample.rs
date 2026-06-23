struct Holder {
    pub direct: &'static u8,
    pub nested: Vec<&'static u8>,
}
