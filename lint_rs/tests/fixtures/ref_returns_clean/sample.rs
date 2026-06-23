fn value() -> u8 {
    0
}

fn visit<Result_Type>(reader: impl FnOnce(&u8) -> Result_Type) -> Option<Result_Type> {
    let _ = reader;
    None
}
