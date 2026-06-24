use std::iter;
use syn;

fn uses_std() {
    let _ = iter::once(0u8);
}

fn uses_syn(item: syn::Expr) {
    let _ = item;
}
