macro_rules! my_macro {
    () => {};
}

#[proc_macro]
pub fn expand(input: TokenStream) -> TokenStream {
    input
}
