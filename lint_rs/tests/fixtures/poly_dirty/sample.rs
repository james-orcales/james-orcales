struct Money {
    pub cents: u64,
}

impl Money {
    fn cents(&self) -> u64 {
        self.cents
    }
}

trait Currency {
    fn cents(&self) -> u64;
}
