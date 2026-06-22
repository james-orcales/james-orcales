struct Money {
    pub cents: u64,
}

impl From<u64> for Money {
    fn from(value: u64) -> Money {
        Money { cents: value }
    }
}

trait Marker {}

#[derive(Clone, Debug)]
struct Point {
    pub x: u64,
    pub y: u64,
}
