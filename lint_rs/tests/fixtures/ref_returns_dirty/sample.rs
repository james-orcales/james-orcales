fn borrow() -> &'static u8 {
    &0
}

fn maybe() -> Option<&'static u8> {
    None
}
