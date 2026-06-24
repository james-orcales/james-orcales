fn uses_std() {
    let _ = std::iter::once(0u8);
}

fn uses_syn(item: syn::Expr) {
    let _ = item;
}

fn uses_crate() {
    let _ = crate::CONST;
}
