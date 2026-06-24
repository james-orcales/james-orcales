//! Waters Rust down to a strict subset, mirroring the Go linter in `../lint`.
//!
//! Rules run on the parsed AST (`syn`), except the `mut`-keyword ban, which
//! stays a token pass so it still fires on files that fail to parse. The crate
//! self-hosts its own dialect: no `mut`, no inherent methods (everything is a
//! free function), `pub` struct fields, module-only imports, no references in
//! fields or returns, no lifetime parameters, and trait methods called via UFCS
//! (`syn::spanned::Spanned::span(node)`) since the import rule forbids importing
//! a trait *name* for method syntax. The traversal is a visitor returning owned
//! results — it never returns a borrowed AST node, so it obeys its own ban 2.

use std::fs;
use std::path;

/// A single rule violation at a source location.
pub struct Violation {
    /// 1-based source line.
    pub line: usize,
    /// 0-based source column; rendered as +1 for editor/human convention.
    pub column: usize,
    /// Human-readable diagnostic body (the part after `file:line:col: `).
    pub message: String,
}

/// One `file:line:col: message` line per violation across every `.rs` file
/// under `root`. Empty means clean.
pub fn scan(root: &path::Path) -> Vec<String> {
    rust_files(root).iter().flat_map(|path| scan_file(path)).collect()
}

fn rust_files(root: &path::Path) -> Vec<path::PathBuf> {
    if root.is_file() {
        return match root.extension().is_some_and(|ext| ext == "rs") {
            true => vec![root.to_path_buf()],
            false => Vec::new(),
        };
    }
    match fs::read_dir(root) {
        Ok(entries) => entries
            .filter_map(Result::ok)
            .flat_map(|entry| rust_files(&entry.path()))
            .collect(),
        Err(_) => Vec::new(),
    }
}

fn scan_file(path: &path::Path) -> Vec<String> {
    let display = path.display().to_string();
    let source = match fs::read_to_string(path) {
        Ok(text) => text,
        // Unreadable files are an I/O concern, out of the linter's remit.
        Err(_) => return Vec::new(),
    };
    // The mut pass is independent of syn so it still reports on files whose
    // delimiters balance but whose grammar syn rejects.
    let keyword = keyword_violations(&source);
    match syn::parse_file(&source) {
        // No AST means no whitelist, so every `mut` stands, plus the parse error.
        Err(error) => keyword
            .iter()
            .map(|v| render(&display, v))
            .chain(std::iter::once(format!("{display}: parse error: {error}")))
            .collect(),
        // Drop the `mut` tokens that fall inside a whitelisted primitive function.
        Ok(file) => {
            let exempt = mut_whitelist_spans(&file, path);
            keyword
                .iter()
                .filter(|v| !within_any(v, &exempt))
                .map(|v| render(&display, v))
                .chain(check_file(&file).iter().map(|v| render(&display, v)))
                // Comments live in the raw source, not the `syn` AST (the lexer
                // drops `//` trivia), so this pass takes the text directly.
                .chain(comment_violations(&source).iter().map(|v| render(&display, v)))
                .collect()
        }
    }
}

fn render(path: &str, violation: &Violation) -> String {
    format!("{path}:{}:{}: {}", violation.line, violation.column + 1, violation.message)
}

/// Fans the parsed file out to every AST rule via the item visitor.
fn check_file(file: &syn::File) -> Vec<Violation> {
    let items = &file.items;
    each_item(items, layout_check)
        .into_iter()
        .chain(each_item(items, fields_public_check))
        .chain(each_item(items, polymorphism_check))
        .chain(each_item(items, import_check))
        .chain(each_item(items, item_casing))
        .chain(each_item(items, macro_check))
        .chain(each_item(items, reference_field_check))
        .chain(each_item(items, reference_return_check))
        .chain(each_item(items, lifetime_param_check))
        .chain(each_item(items, function_size_check))
        .chain(entry_point_check(file))
        .collect()
}

// Item visitor: owned results, borrows only inward.

/// Applies `check` to every item under `items`, including items nested in module
/// bodies and function/method bodies. Returns owned results; no reference
/// escapes, so the visitor itself obeys the no-reference-return rule.
fn each_item<R>(items: &[syn::Item], check: impl Fn(&syn::Item) -> Vec<R> + Copy) -> Vec<R> {
    items
        .iter()
        .flat_map(|item| {
            check(item).into_iter().chain(nested_items(item, check)).collect::<Vec<R>>()
        })
        .collect()
}

fn nested_items<R>(item: &syn::Item, check: impl Fn(&syn::Item) -> Vec<R> + Copy) -> Vec<R> {
    match item {
        syn::Item::Mod(item_mod) => match &item_mod.content {
            Some((_, inner)) => each_item(inner, check),
            None => Vec::new(),
        },
        syn::Item::Fn(item_fn) => block_items(&item_fn.block, check),
        syn::Item::Impl(item_impl) => item_impl
            .items
            .iter()
            .flat_map(|impl_item| match impl_item {
                syn::ImplItem::Fn(method) => block_items(&method.block, check),
                _ => Vec::new(),
            })
            .collect(),
        syn::Item::Trait(item_trait) => item_trait
            .items
            .iter()
            .flat_map(|trait_item| match trait_item {
                syn::TraitItem::Fn(method) => match &method.default {
                    Some(block) => block_items(block, check),
                    None => Vec::new(),
                },
                _ => Vec::new(),
            })
            .collect(),
        _ => Vec::new(),
    }
}

fn block_items<R>(block: &syn::Block, check: impl Fn(&syn::Item) -> Vec<R> + Copy) -> Vec<R> {
    block
        .stmts
        .iter()
        .flat_map(|stmt| match stmt {
            syn::Stmt::Item(nested) => each_item(std::slice::from_ref(nested), check),
            // A `let` initializer (and its `else` block) can hold items in a
            // block expression; an expression statement can too (if/match/loop
            // bodies, closures). Both were previously skipped.
            syn::Stmt::Local(local) => match &local.init {
                Some(init) => expr_items(&init.expr, check)
                    .into_iter()
                    .chain(init.diverge.iter().flat_map(|(_, branch)| expr_items(branch, check)))
                    .collect(),
                None => Vec::new(),
            },
            syn::Stmt::Expr(expr, _) => expr_items(expr, check),
            // A macro invocation's body is opaque tokens; syn does not expand it,
            // so any items it would generate are invisible to every AST rule.
            syn::Stmt::Macro(_) => Vec::new(),
        })
        .collect()
}

/// Every item declared anywhere inside an expression — descending into each
/// nested block (`if`/`match`/`loop`/closure/`unsafe`/...) and recursing through
/// sub-expressions. Items can only live in a block, so reaching every block
/// reaches every item; leaf and opaque expressions contribute nothing.
fn expr_items<R>(expr: &syn::Expr, check: impl Fn(&syn::Item) -> Vec<R> + Copy) -> Vec<R> {
    match expr {
        // Block-bearing expressions: each carries one or more blocks whose
        // statements may declare items. The compound and operand expressions —
        // which only nest sub-expressions — are split off below to keep each
        // arm-set within the function-size cap.
        syn::Expr::Block(e) => block_items(&e.block, check),
        syn::Expr::Unsafe(e) => block_items(&e.block, check),
        syn::Expr::Async(e) => block_items(&e.block, check),
        syn::Expr::Const(e) => block_items(&e.block, check),
        syn::Expr::TryBlock(e) => block_items(&e.block, check),
        syn::Expr::Loop(e) => block_items(&e.body, check),
        syn::Expr::While(e) => {
            expr_items(&e.cond, check).into_iter().chain(block_items(&e.body, check)).collect()
        }
        syn::Expr::ForLoop(e) => {
            expr_items(&e.expr, check).into_iter().chain(block_items(&e.body, check)).collect()
        }
        syn::Expr::If(e) => expr_items(&e.cond, check)
            .into_iter()
            .chain(block_items(&e.then_branch, check))
            .chain(e.else_branch.iter().flat_map(|(_, branch)| expr_items(branch, check)))
            .collect(),
        syn::Expr::Match(e) => expr_items(&e.expr, check)
            .into_iter()
            .chain(e.arms.iter().flat_map(|arm| {
                arm.guard
                    .iter()
                    .flat_map(|(_, guard)| expr_items(guard, check))
                    .chain(expr_items(&arm.body, check))
            }))
            .collect(),
        syn::Expr::Closure(e) => expr_items(&e.body, check),
        _ => compound_expr_items(expr, check),
    }
}

/// The operand and container expressions: those that only nest sub-expressions,
/// never a block of their own. Split out of `expr_items` so neither half exceeds
/// the function-size cap; together they cover every block-reaching variant.
fn compound_expr_items<R>(
    expr: &syn::Expr,
    check: impl Fn(&syn::Item) -> Vec<R> + Copy,
) -> Vec<R> {
    match expr {
        syn::Expr::Array(e) => e.elems.iter().flat_map(|x| expr_items(x, check)).collect(),
        syn::Expr::Tuple(e) => e.elems.iter().flat_map(|x| expr_items(x, check)).collect(),
        syn::Expr::Call(e) => expr_items(&e.func, check)
            .into_iter()
            .chain(e.args.iter().flat_map(|x| expr_items(x, check)))
            .collect(),
        syn::Expr::MethodCall(e) => expr_items(&e.receiver, check)
            .into_iter()
            .chain(e.args.iter().flat_map(|x| expr_items(x, check)))
            .collect(),
        syn::Expr::Binary(e) => {
            expr_items(&e.left, check).into_iter().chain(expr_items(&e.right, check)).collect()
        }
        syn::Expr::Assign(e) => {
            expr_items(&e.left, check).into_iter().chain(expr_items(&e.right, check)).collect()
        }
        syn::Expr::Index(e) => {
            expr_items(&e.expr, check).into_iter().chain(expr_items(&e.index, check)).collect()
        }
        syn::Expr::Repeat(e) => {
            expr_items(&e.expr, check).into_iter().chain(expr_items(&e.len, check)).collect()
        }
        syn::Expr::Range(e) => e
            .start
            .iter()
            .flat_map(|x| expr_items(x, check))
            .chain(e.end.iter().flat_map(|x| expr_items(x, check)))
            .collect(),
        syn::Expr::Struct(e) => e
            .fields
            .iter()
            .flat_map(|field| expr_items(&field.expr, check))
            .chain(e.rest.iter().flat_map(|x| expr_items(x, check)))
            .collect(),
        syn::Expr::Unary(e) => expr_items(&e.expr, check),
        syn::Expr::Cast(e) => expr_items(&e.expr, check),
        syn::Expr::Field(e) => expr_items(&e.base, check),
        syn::Expr::Await(e) => expr_items(&e.base, check),
        syn::Expr::Try(e) => expr_items(&e.expr, check),
        syn::Expr::Paren(e) => expr_items(&e.expr, check),
        syn::Expr::Group(e) => expr_items(&e.expr, check),
        syn::Expr::Reference(e) => expr_items(&e.expr, check),
        syn::Expr::Let(e) => expr_items(&e.expr, check),
        syn::Expr::Break(e) => e.expr.iter().flat_map(|x| expr_items(x, check)).collect(),
        syn::Expr::Return(e) => e.expr.iter().flat_map(|x| expr_items(x, check)).collect(),
        syn::Expr::Yield(e) => e.expr.iter().flat_map(|x| expr_items(x, check)).collect(),
        _ => Vec::new(),
    }
}

// Span helpers.

/// Start line/column of any spanned node, via UFCS so we need not import the
/// `Spanned` trait name (the import rule forbids it).
fn at(node: &impl syn::spanned::Spanned) -> (usize, usize) {
    let start = syn::spanned::Spanned::span(node).start();
    (start.line, start.column)
}

fn violation(node: &impl syn::spanned::Spanned, message: String) -> Violation {
    let (line, column) = at(node);
    Violation { line, column, message }
}

// The `mut` pass: a token scan plus an exact-function whitelist.

// The exact functions allowed to write `mut`: the trusted arena/handle
// primitive, whose in-place O(1) growth has no mut-free equivalent. Each entry
// pins ONE function in ONE file — a `mut` is exempt only if it sits inside a
// function whose name matches AND whose file path ends with this exact suffix.
// Name alone is not enough; this is not a file/directory ignore; nothing else
// in the universe may write `mut`.
const MUT_ALLOWED: &[(&str, &str)] = &[
    ("shared_rs/src/arena.rs", "insert"),
    ("shared_rs/src/gen_arena.rs", "insert"),
    ("shared_rs/src/gen_arena.rs", "remove"),
];

fn mut_allowed(path: &path::Path, ident: &syn::Ident) -> bool {
    let name = ident.to_string();
    MUT_ALLOWED
        .iter()
        .any(|(suffix, function)| *function == name.as_str() && path.ends_with(suffix))
}

/// The (start, end) span of every whitelisted function in THIS file, so the
/// `mut` pass can subtract the keyword tokens — including a `mut self` receiver
/// in the signature — that fall inside them.
fn mut_whitelist_spans(
    file: &syn::File,
    path: &path::Path,
) -> Vec<(proc_macro2::LineColumn, proc_macro2::LineColumn)> {
    each_item(&file.items, |item| match item {
        syn::Item::Fn(item_fn) if mut_allowed(path, &item_fn.sig.ident) => {
            let span = syn::spanned::Spanned::span(item_fn);
            vec![(span.start(), span.end())]
        }
        _ => Vec::new(),
    })
}

fn within_any(
    violation: &Violation,
    spans: &[(proc_macro2::LineColumn, proc_macro2::LineColumn)],
) -> bool {
    let at = (violation.line, violation.column);
    spans.iter().any(|(start, end)| at >= (start.line, start.column) && at <= (end.line, end.column))
}

/// Every bare `mut` keyword token. Tokenizing (not text matching) is why a
/// `mut` in a comment or string never counts.
fn keyword_violations(source: &str) -> Vec<Violation> {
    match source.parse::<proc_macro2::TokenStream>() {
        Ok(tokens) => mut_idents(tokens),
        Err(_) => Vec::new(),
    }
}

fn mut_idents(tokens: proc_macro2::TokenStream) -> Vec<Violation> {
    tokens
        .into_iter()
        .flat_map(|token| match token {
            proc_macro2::TokenTree::Ident(ident) if ident == "mut" => {
                let start = ident.span().start();
                vec![Violation {
                    line: start.line,
                    column: start.column,
                    message: "`mut` is banned".to_string(),
                }]
            }
            proc_macro2::TokenTree::Group(group) => mut_idents(group.stream()),
            _ => Vec::new(),
        })
        .collect()
}

// Shared: does a type contain a reference anywhere?

fn type_has_reference(ty: &syn::Type) -> bool {
    match ty {
        syn::Type::Reference(_) => true,
        syn::Type::Path(type_path) => path_has_reference(&type_path.path),
        syn::Type::Tuple(tuple) => tuple.elems.iter().any(type_has_reference),
        syn::Type::Array(array) => type_has_reference(&array.elem),
        syn::Type::Slice(slice) => type_has_reference(&slice.elem),
        syn::Type::Ptr(ptr) => type_has_reference(&ptr.elem),
        syn::Type::Paren(paren) => type_has_reference(&paren.elem),
        syn::Type::Group(group) => type_has_reference(&group.elem),
        // `fn(&A) -> &B`: a reference in the signature, not just angle brackets.
        syn::Type::BareFn(bare) => {
            bare.inputs.iter().any(|arg| type_has_reference(&arg.ty))
                || return_has_reference(&bare.output)
        }
        // `dyn Fn(&T)`, `dyn Iterator<Item = &T>`, and `impl ...` of the same.
        syn::Type::TraitObject(object) => object.bounds.iter().any(bound_has_reference),
        syn::Type::ImplTrait(imp) => imp.bounds.iter().any(bound_has_reference),
        _ => false,
    }
}

fn path_has_reference(path: &syn::Path) -> bool {
    path.segments.iter().any(|segment| match &segment.arguments {
        syn::PathArguments::AngleBracketed(args) => args.args.iter().any(|arg| match arg {
            syn::GenericArgument::Type(inner) => type_has_reference(inner),
            // `Iterator<Item = &T>` — the reference hides in the binding's type.
            syn::GenericArgument::AssocType(assoc) => type_has_reference(&assoc.ty),
            _ => false,
        }),
        // `Fn(&T) -> &U` — parenthesized inputs and return, not angle brackets.
        syn::PathArguments::Parenthesized(args) => {
            args.inputs.iter().any(type_has_reference) || return_has_reference(&args.output)
        }
        syn::PathArguments::None => false,
    })
}

fn return_has_reference(output: &syn::ReturnType) -> bool {
    match output {
        syn::ReturnType::Type(_, ty) => type_has_reference(ty),
        syn::ReturnType::Default => false,
    }
}

fn bound_has_reference(bound: &syn::TypeParamBound) -> bool {
    match bound {
        syn::TypeParamBound::Trait(trait_bound) => path_has_reference(&trait_bound.path),
        _ => false,
    }
}

// Ban 1: reference fields.

/// A struct/enum/union field may not hold a reference (directly or nested), so
/// no type ever needs a lifetime to carry a borrow. Own the value, or store an
/// integer handle into an arena.
fn reference_field_check(item: &syn::Item) -> Vec<Violation> {
    match item {
        syn::Item::Struct(item_struct) => fields_reference(&item_struct.fields),
        syn::Item::Union(item_union) => {
            item_union.fields.named.iter().filter_map(field_reference).collect()
        }
        syn::Item::Enum(item_enum) => item_enum
            .variants
            .iter()
            .flat_map(|variant| fields_reference(&variant.fields))
            .collect(),
        _ => Vec::new(),
    }
}

fn fields_reference(fields: &syn::Fields) -> Vec<Violation> {
    match fields {
        syn::Fields::Named(named) => named.named.iter().filter_map(field_reference).collect(),
        syn::Fields::Unnamed(unnamed) => {
            unnamed.unnamed.iter().filter_map(field_reference).collect()
        }
        syn::Fields::Unit => Vec::new(),
    }
}

fn field_reference(field: &syn::Field) -> Option<Violation> {
    match type_has_reference(&field.ty) {
        true => Some(violation(
            field,
            "reference field banned; own the value or store an integer handle".to_string(),
        )),
        false => None,
    }
}

// Ban 2: reference returns.

/// A function may not return a reference; borrows flow in, never out. Return an
/// owned value, or expose reads via a visitor (`with(&arena, h, |x| ...)`).
fn reference_return_check(item: &syn::Item) -> Vec<Violation> {
    match item {
        syn::Item::Fn(item_fn) => return_reference(&item_fn.sig),
        syn::Item::Impl(item_impl) => item_impl
            .items
            .iter()
            .flat_map(|impl_item| match impl_item {
                syn::ImplItem::Fn(method) => return_reference(&method.sig),
                _ => Vec::new(),
            })
            .collect(),
        syn::Item::Trait(item_trait) => item_trait
            .items
            .iter()
            .flat_map(|trait_item| match trait_item {
                syn::TraitItem::Fn(method) => return_reference(&method.sig),
                _ => Vec::new(),
            })
            .collect(),
        _ => Vec::new(),
    }
}

fn return_reference(sig: &syn::Signature) -> Vec<Violation> {
    match &sig.output {
        syn::ReturnType::Type(_, ty) if type_has_reference(ty) => vec![violation(
            &sig.ident,
            "reference return banned; return an owned value (a visitor or handle for reads)"
                .to_string(),
        )],
        _ => Vec::new(),
    }
}

// Ban 3: lifetime parameters.

/// No `<'a>` may be declared on any item. With references barred from fields and
/// returns, elision covers every remaining case, so a named lifetime is never
/// needed. `'static` is a fixed bound, not a parameter, and is untouched.
fn lifetime_param_check(item: &syn::Item) -> Vec<Violation> {
    match item {
        syn::Item::Fn(item_fn) => lifetime_params(&item_fn.sig.generics),
        syn::Item::Struct(item_struct) => lifetime_params(&item_struct.generics),
        syn::Item::Enum(item_enum) => lifetime_params(&item_enum.generics),
        syn::Item::Union(item_union) => lifetime_params(&item_union.generics),
        syn::Item::Type(item_type) => lifetime_params(&item_type.generics),
        syn::Item::Trait(item_trait) => lifetime_params(&item_trait.generics)
            .into_iter()
            .chain(item_trait.items.iter().flat_map(|trait_item| match trait_item {
                syn::TraitItem::Fn(method) => lifetime_params(&method.sig.generics),
                syn::TraitItem::Type(item_type) => lifetime_params(&item_type.generics),
                _ => Vec::new(),
            }))
            .collect(),
        syn::Item::Impl(item_impl) => lifetime_params(&item_impl.generics)
            .into_iter()
            .chain(item_impl.items.iter().flat_map(|impl_item| match impl_item {
                syn::ImplItem::Fn(method) => lifetime_params(&method.sig.generics),
                syn::ImplItem::Type(item_type) => lifetime_params(&item_type.generics),
                _ => Vec::new(),
            }))
            .collect(),
        _ => Vec::new(),
    }
}

fn lifetime_params(generics: &syn::Generics) -> Vec<Violation> {
    generics
        .params
        .iter()
        .filter_map(|param| match param {
            syn::GenericParam::Lifetime(lifetime_param) => Some(violation(
                &lifetime_param.lifetime,
                "lifetime parameter banned; references cannot be stored or returned, \
                 so none is needed"
                    .to_string(),
            )),
            _ => None,
        })
        .collect()
}

// Ban: macro authoring.

fn macro_check(item: &syn::Item) -> Vec<Violation> {
    match item {
        // `ident: Some` is a `macro_rules! NAME` definition; `None` is an
        // item-position invocation like `lazy_static! { ... }`, which is allowed.
        syn::Item::Macro(item_macro) => item_macro
            .ident
            .as_ref()
            .map(|ident| {
                violation(
                    ident,
                    "macro_rules! definition banned; macros may be called, not authored".to_string(),
                )
            })
            .into_iter()
            .collect(),
        syn::Item::Fn(item_fn) if has_proc_macro_attr(item_fn) => vec![violation(
            &item_fn.sig.ident,
            "proc-macro definition banned; macros may be called, not authored".to_string(),
        )],
        _ => Vec::new(),
    }
}

fn has_proc_macro_attr(item_fn: &syn::ItemFn) -> bool {
    item_fn.attrs.iter().any(|attr| match attr.path().segments.last() {
        Some(segment) => matches!(
            segment.ident.to_string().as_str(),
            "proc_macro" | "proc_macro_derive" | "proc_macro_attribute"
        ),
        None => false,
    })
}

// Ban: user-authored polymorphism.

fn polymorphism_check(item: &syn::Item) -> Vec<Violation> {
    match item {
        syn::Item::Impl(item_impl) if item_impl.trait_.is_none() => vec![violation(
            &*item_impl.self_ty,
            "inherent impl banned; write a free function taking the receiver \
             as its first parameter"
                .to_string(),
        )],
        syn::Item::Trait(item_trait) if trait_has_method(item_trait) => vec![violation(
            &item_trait.ident,
            "trait with methods banned; use free functions (a marker trait \
             with no methods is allowed)"
                .to_string(),
        )],
        _ => Vec::new(),
    }
}

fn trait_has_method(item_trait: &syn::ItemTrait) -> bool {
    item_trait.items.iter().any(|trait_item| matches!(trait_item, syn::TraitItem::Fn(_)))
}

// Ban: non-pub struct/union fields.

fn fields_public_check(item: &syn::Item) -> Vec<Violation> {
    match item {
        syn::Item::Struct(item_struct) => fields_public_of(&item_struct.fields),
        syn::Item::Union(item_union) => {
            item_union.fields.named.iter().filter_map(field_public).collect()
        }
        _ => Vec::new(),
    }
}

fn fields_public_of(fields: &syn::Fields) -> Vec<Violation> {
    match fields {
        syn::Fields::Named(named) => named.named.iter().filter_map(field_public).collect(),
        syn::Fields::Unnamed(unnamed) => unnamed.unnamed.iter().filter_map(field_public).collect(),
        syn::Fields::Unit => Vec::new(),
    }
}

fn field_public(field: &syn::Field) -> Option<Violation> {
    match field.vis {
        syn::Visibility::Public(_) => None,
        _ => Some(violation(field, "struct field must be `pub`".to_string())),
    }
}

// Layout: inline modules.

fn layout_check(item: &syn::Item) -> Vec<Violation> {
    match item {
        syn::Item::Mod(item_mod) if item_mod.content.is_some() && item_mod.ident != "tests" => {
            vec![violation(
                &item_mod.ident,
                format!(
                    "inline module `{0}` banned; split into a file (`mod {0};`); \
                     only `mod tests` may be inline",
                    item_mod.ident
                ),
            )]
        }
        _ => Vec::new(),
    }
}

// Imports: module-only.

fn import_check(item: &syn::Item) -> Vec<Violation> {
    match item {
        // `pub`/`pub(crate) use` re-exports are exempt; only a private `use` binds.
        syn::Item::Use(item_use) if matches!(item_use.vis, syn::Visibility::Inherited) => {
            use_tree(&item_use.tree, None)
        }
        _ => Vec::new(),
    }
}

fn use_tree(tree: &syn::UseTree, parent: Option<&syn::Ident>) -> Vec<Violation> {
    match tree {
        syn::UseTree::Path(use_path) => use_tree(&use_path.tree, Some(&use_path.ident)),
        syn::UseTree::Name(use_name) => leaf_violation(&use_name.ident, parent).into_iter().collect(),
        syn::UseTree::Rename(use_rename) => {
            leaf_violation(&use_rename.rename, parent).into_iter().collect()
        }
        syn::UseTree::Glob(_) => {
            vec![violation(tree, "glob import banned; name the module and qualify".to_string())]
        }
        syn::UseTree::Group(use_group) if use_group.items.len() > 1 => {
            vec![violation(tree, "grouped import banned; one module per `use`".to_string())]
        }
        syn::UseTree::Group(use_group) => {
            use_group.items.iter().flat_map(|inner| use_tree(inner, parent)).collect()
        }
    }
}

fn leaf_violation(ident: &syn::Ident, parent: Option<&syn::Ident>) -> Option<Violation> {
    let name = ident.to_string();
    // A std trait must be importable by name to call its methods.
    if matches!(name.as_str(), "Read" | "Write" | "Seek" | "BufRead") {
        return None;
    }
    // An uppercase-first leaf is a type/trait, i.e. an item, not a module.
    if name.chars().next().is_some_and(char::is_uppercase) {
        return Some(violation(
            ident,
            format!("import a module, not a type/trait: `{name}` (import the module and qualify)"),
        ));
    }
    // A lowercase leaf that is actually a known std free function.
    if is_denylisted_free_function(parent, &name) {
        return Some(violation(
            ident,
            format!("import the module, call qualified: `{name}` is a std free function"),
        ));
    }
    None
}

fn is_denylisted_free_function(parent: Option<&syn::Ident>, name: &str) -> bool {
    match parent.map(syn::Ident::to_string).as_deref() {
        Some("mem") => matches!(name, "swap" | "replace" | "take"),
        Some("cmp") => matches!(name, "min" | "max"),
        Some("ptr") => true,
        _ => false,
    }
}

// Casing: first-char-driven snake_case / Ada_Case.

fn item_casing(item: &syn::Item) -> Vec<Violation> {
    match item {
        syn::Item::Fn(item_fn) => fn_casing(&item_fn.sig, &item_fn.block),
        syn::Item::Struct(item_struct) => check_name(&item_struct.ident, None)
            .into_iter()
            .chain(generics_casing(&item_struct.generics))
            .chain(fields_casing(&item_struct.fields))
            .collect(),
        syn::Item::Enum(item_enum) => check_name(&item_enum.ident, None)
            .into_iter()
            .chain(generics_casing(&item_enum.generics))
            .chain(item_enum.variants.iter().flat_map(|variant| {
                check_name(&variant.ident, None)
                    .into_iter()
                    .chain(fields_casing(&variant.fields))
            }))
            .collect(),
        syn::Item::Union(item_union) => check_name(&item_union.ident, None)
            .into_iter()
            .chain(generics_casing(&item_union.generics))
            .chain(named_casing(&item_union.fields))
            .collect(),
        syn::Item::Type(item_type) => check_name(&item_type.ident, None)
            .into_iter()
            .chain(generics_casing(&item_type.generics))
            .collect(),
        syn::Item::Trait(item_trait) => check_name(&item_trait.ident, None)
            .into_iter()
            .chain(generics_casing(&item_trait.generics))
            .chain(item_trait.items.iter().flat_map(trait_item_casing))
            .collect(),
        syn::Item::Impl(item_impl) => generics_casing(&item_impl.generics)
            .into_iter()
            .chain(item_impl.items.iter().flat_map(impl_item_casing))
            .collect(),
        syn::Item::Const(item_const) => check_name(&item_const.ident, None).into_iter().collect(),
        syn::Item::Static(item_static) => check_name(&item_static.ident, None).into_iter().collect(),
        syn::Item::Mod(item_mod) => check_name(&item_mod.ident, None).into_iter().collect(),
        _ => Vec::new(),
    }
}

// `forced`: None = first-char-driven; Some(true) = require Ada; Some(false) = require snake.
fn check_name(ident: &syn::Ident, forced: Option<bool>) -> Option<Violation> {
    let name = ident.to_string();
    if !name.chars().next().is_some_and(char::is_alphabetic) {
        return None;
    }
    let want_ada = forced.unwrap_or_else(|| name.chars().next().is_some_and(char::is_uppercase));
    let (ok, want) = match want_ada {
        true => (is_ada(&name), "Ada_Case"),
        false => (is_snake(&name), "snake_case"),
    };
    match ok {
        true => None,
        false => Some(violation(ident, format!("name `{name}` must be {want}"))),
    }
}

/// `^[a-z][a-z0-9]*(_[a-z0-9]+)*$`: lowercase-letter start, a lowercase/digit/`_`
/// body, no leading/trailing/doubled underscore.
fn is_snake(name: &str) -> bool {
    let lower_start = name.chars().next().is_some_and(|c| c.is_ascii_lowercase());
    let charset =
        name.chars().all(|c| c.is_ascii_lowercase() || c.is_ascii_digit() || c == '_');
    lower_start && charset && !name.ends_with('_') && !name.contains("__")
}

/// `^(<word>)(_<word>)*$` where a word is `[A-Z][a-z0-9]*` (Capital + lowercase)
/// or `[A-Z][A-Z0-9]*s?` (all-caps acronym with an optional plural `s`). An
/// empty segment — a leading, trailing, or doubled `_` — fails.
fn is_ada(name: &str) -> bool {
    !name.is_empty() && name.split('_').all(is_ada_word)
}

fn is_ada_word(word: &str) -> bool {
    if !word.chars().next().is_some_and(|c| c.is_ascii_uppercase()) {
        return false;
    }
    let tail = &word[1..];
    is_lower_tail(tail) || is_upper_tail(tail)
}

// `[a-z0-9]*` after the leading capital: `Foo`, `Bar2`, `F`.
fn is_lower_tail(tail: &str) -> bool {
    tail.chars().all(|c| c.is_ascii_lowercase() || c.is_ascii_digit())
}

// `[A-Z0-9]*s?` after the leading capital: `API`, `IDs`, `U64`.
fn is_upper_tail(tail: &str) -> bool {
    let core = tail.strip_suffix('s').unwrap_or(tail);
    core.chars().all(|c| c.is_ascii_uppercase() || c.is_ascii_digit())
}

fn fn_casing(sig: &syn::Signature, block: &syn::Block) -> Vec<Violation> {
    sig_casing(sig).into_iter().chain(block_lets_casing(block)).collect()
}

fn sig_casing(sig: &syn::Signature) -> Vec<Violation> {
    check_name(&sig.ident, None)
        .into_iter()
        .chain(generics_casing(&sig.generics))
        .chain(sig.inputs.iter().filter_map(|arg| match arg {
            syn::FnArg::Typed(pat_type) => pat_casing(&pat_type.pat),
            syn::FnArg::Receiver(_) => None,
        }))
        .collect()
}

// Only top-level `let` bindings are checked; bindings inside nested control-flow
// blocks, closures, and match arms are out of scope (deferred).
fn block_lets_casing(block: &syn::Block) -> Vec<Violation> {
    block
        .stmts
        .iter()
        .filter_map(|stmt| match stmt {
            syn::Stmt::Local(local) => pat_casing(&local.pat),
            _ => None,
        })
        .collect()
}

fn generics_casing(generics: &syn::Generics) -> Vec<Violation> {
    generics
        .params
        .iter()
        .filter_map(|param| match param {
            syn::GenericParam::Type(type_param) => check_name(&type_param.ident, Some(true)),
            syn::GenericParam::Const(const_param) => check_name(&const_param.ident, Some(true)),
            syn::GenericParam::Lifetime(lifetime_param) => {
                check_name(&lifetime_param.lifetime.ident, Some(false))
            }
        })
        .collect()
}

fn fields_casing(fields: &syn::Fields) -> Vec<Violation> {
    match fields {
        syn::Fields::Named(named) => named_casing(named),
        // Tuple and unit fields have no name to check.
        _ => Vec::new(),
    }
}

fn named_casing(fields: &syn::FieldsNamed) -> Vec<Violation> {
    fields
        .named
        .iter()
        .filter_map(|field| field.ident.as_ref())
        .filter_map(|ident| check_name(ident, None))
        .collect()
}

fn trait_item_casing(item: &syn::TraitItem) -> Vec<Violation> {
    match item {
        syn::TraitItem::Fn(method) => {
            let signature = sig_casing(&method.sig);
            match &method.default {
                Some(block) => signature.into_iter().chain(block_lets_casing(block)).collect(),
                None => signature,
            }
        }
        syn::TraitItem::Const(item_const) => {
            check_name(&item_const.ident, None).into_iter().collect()
        }
        syn::TraitItem::Type(item_type) => check_name(&item_type.ident, None)
            .into_iter()
            .chain(generics_casing(&item_type.generics))
            .collect(),
        _ => Vec::new(),
    }
}

fn impl_item_casing(item: &syn::ImplItem) -> Vec<Violation> {
    match item {
        syn::ImplItem::Fn(method) => fn_casing(&method.sig, &method.block),
        syn::ImplItem::Const(item_const) => {
            check_name(&item_const.ident, None).into_iter().collect()
        }
        syn::ImplItem::Type(item_type) => check_name(&item_type.ident, None)
            .into_iter()
            .chain(generics_casing(&item_type.generics))
            .collect(),
        _ => Vec::new(),
    }
}

fn pat_casing(pat: &syn::Pat) -> Option<Violation> {
    match pat {
        syn::Pat::Ident(pat_ident) => check_name(&pat_ident.ident, None),
        syn::Pat::Type(pat_type) => pat_casing(&pat_type.pat),
        _ => None,
    }
}

// Size: function body line count.

// A body wider than this hides its control flow; the cap forces a split. Mirrors
// the Go linter's `function_lines_max`.
const FUNCTION_LINES_MAX: usize = 70;

/// A function body spans at most `FUNCTION_LINES_MAX` lines, counted from the
/// opening brace's line to the closing brace's line inclusive. Free functions,
/// trait-impl methods, and trait default methods are all measured. Closures are
/// deferred, matching the casing pass, which also defers closure interiors.
fn function_size_check(item: &syn::Item) -> Vec<Violation> {
    match item {
        syn::Item::Fn(item_fn) => block_size(&item_fn.sig.ident, &item_fn.block),
        syn::Item::Impl(item_impl) => item_impl
            .items
            .iter()
            .flat_map(|impl_item| match impl_item {
                syn::ImplItem::Fn(method) => block_size(&method.sig.ident, &method.block),
                _ => Vec::new(),
            })
            .collect(),
        syn::Item::Trait(item_trait) => item_trait
            .items
            .iter()
            .flat_map(|trait_item| match trait_item {
                syn::TraitItem::Fn(method) => match &method.default {
                    Some(block) => block_size(&method.sig.ident, block),
                    None => Vec::new(),
                },
                _ => Vec::new(),
            })
            .collect(),
        _ => Vec::new(),
    }
}

fn block_size(ident: &syn::Ident, block: &syn::Block) -> Vec<Violation> {
    let span = block.brace_token.span;
    let open_line = span.open().start().line;
    let close_line = span.close().start().line;
    // Brace-to-brace inclusive, matching the Go linter's lbrace..rbrace count.
    let line_count = close_line - open_line + 1;
    match line_count > FUNCTION_LINES_MAX {
        true => vec![violation(
            ident,
            format!("function spans {line_count} lines (max {FUNCTION_LINES_MAX})"),
        )],
        false => Vec::new(),
    }
}

// Layout: entry point first.

/// When a file declares `fn main`, it must be the first function. Non-function
/// items — a `use`, a `struct`, a `const` — may precede it. Only top-level
/// functions count, so this reads `file.items` directly rather than via the
/// item visitor, which would also descend into nested functions. Rust has no
/// `Main`/`TestMain` convention, so `main` is the sole entry point.
fn entry_point_check(file: &syn::File) -> Vec<Violation> {
    file.items
        .iter()
        .filter_map(|item| match item {
            syn::Item::Fn(item_fn) => Some(item_fn),
            _ => None,
        })
        .skip(1)
        .filter(|item_fn| item_fn.sig.ident == "main")
        .map(|item_fn| {
            violation(
                &item_fn.sig.ident,
                "fn main must be the first function in the file".to_string(),
            )
        })
        .collect()
}

// Comments: opening, ending, and spacing.

/// A line comment opens with a space then a capital letter and ends in `.`, `:`,
/// `?`, or `!`. In a group of comments on consecutive lines only the first
/// line's opening and the last line's ending are judged; a trailing (inline)
/// comment is exempt from both, but every line still needs the space.
fn comment_violations(source: &str) -> Vec<Violation> {
    let lines = comment_lines(source);
    lines.chunk_by(|a, b| b.line == a.line + 1).flat_map(comment_group_violations).collect()
}

fn comment_group_violations(group: &[Comment_Line]) -> Vec<Violation> {
    let space: Vec<Violation> = group.iter().filter_map(comment_space_violation).collect();
    // A trailing comment (code precedes the `//`) is exempt from the opening and
    // ending rules; whether the group is trailing is decided by its first line.
    // Lines in a markdown code fence are exempt too — the ``` line and the code
    // it encloses are not prose. The space rule still binds every line.
    let trailing = group.first().is_some_and(|first| first.inline);
    let opening = match trailing || group.first().is_some_and(is_fence) {
        true => Vec::new(),
        false => group.first().and_then(comment_capital_violation).into_iter().collect(),
    };
    let ending = match trailing || last_in_fenced_code(group) {
        true => Vec::new(),
        false => group.last().and_then(comment_terminator_violation).into_iter().collect(),
    };
    space.into_iter().chain(opening).chain(ending).collect()
}

/// A comment line opening a markdown code fence: its prose, after the marker,
/// starts with three backticks (an optional language tag may follow).
fn is_fence(comment: &Comment_Line) -> bool {
    comment_body(&comment.text).starts_with("```")
}

/// Whether a group's last line sits in a code fence — the fence line itself, or
/// any line an earlier fence left open. An odd count of fence lines before the
/// last means the block is still open at it.
fn last_in_fenced_code(group: &[Comment_Line]) -> bool {
    match group.split_last() {
        Some((last, rest)) => rest.iter().filter(|c| is_fence(c)).count() % 2 == 1 || is_fence(last),
        None => false,
    }
}

/// One `//` line comment: its position, whether code precedes it on the line (a
/// trailing comment), and the raw text from `//` to end of line.
struct Comment_Line {
    pub line: usize,
    pub column: usize,
    pub inline: bool,
    pub text: String,
}

fn comment_lines(source: &str) -> Vec<Comment_Line> {
    let spans = literal_spans(source);
    source
        .lines()
        .enumerate()
        .filter_map(|(index, line)| comment_on_line(index + 1, line, &spans))
        .collect()
}

fn comment_on_line(
    line_number: usize,
    line: &str,
    spans: &[(proc_macro2::LineColumn, proc_macro2::LineColumn)],
) -> Option<Comment_Line> {
    let chars: Vec<char> = line.chars().collect();
    // The comment starts at the first `//` not sitting inside a string or char
    // literal, so a `//` in a URL string is not mistaken for a comment.
    let column = (0..chars.len().saturating_sub(1))
        .find(|&i| chars[i] == '/' && chars[i + 1] == '/' && !within_spans(line_number, i, spans))?;
    let inline = chars[..column].iter().any(|c| !c.is_whitespace());
    let text = chars[column..].iter().collect();
    Some(Comment_Line { line: line_number, column, inline, text })
}

fn within_spans(
    line: usize,
    column: usize,
    spans: &[(proc_macro2::LineColumn, proc_macro2::LineColumn)],
) -> bool {
    let at = (line, column);
    spans.iter().any(|(start, end)| {
        at >= (start.line, start.column) && at < (end.line, end.column)
    })
}

/// Spans of the real string/char/number literals, used to mask `//` that lives
/// inside a literal. A doc comment desugars to a `#[doc = "..."]` literal whose
/// span covers the original `///`/`//!` source, which begins with a slash, never
/// a quote — excluding it would hide doc comments from the scan, so those are
/// kept by checking the literal's first source character.
fn literal_spans(source: &str) -> Vec<(proc_macro2::LineColumn, proc_macro2::LineColumn)> {
    match source.parse::<proc_macro2::TokenStream>() {
        Ok(tokens) => literal_spans_in(tokens, source),
        // The caller only reaches this on a file `syn` already parsed, so a
        // re-tokenize failure is unreachable; degrade to no masking regardless.
        Err(_) => Vec::new(),
    }
}

fn literal_spans_in(
    tokens: proc_macro2::TokenStream,
    source: &str,
) -> Vec<(proc_macro2::LineColumn, proc_macro2::LineColumn)> {
    tokens
        .into_iter()
        .flat_map(|token| match token {
            proc_macro2::TokenTree::Literal(literal) => {
                let span = literal.span();
                match char_at(source, span.start()) {
                    Some('/') => Vec::new(),
                    _ => vec![(span.start(), span.end())],
                }
            }
            proc_macro2::TokenTree::Group(group) => literal_spans_in(group.stream(), source),
            _ => Vec::new(),
        })
        .collect()
}

fn char_at(source: &str, at: proc_macro2::LineColumn) -> Option<char> {
    source.lines().nth(at.line - 1).and_then(|line| line.chars().nth(at.column))
}

fn comment_space_violation(comment: &Comment_Line) -> Option<Violation> {
    match has_space_after_marker(&comment.text) {
        true => None,
        false => Some(Violation {
            line: comment.line,
            column: comment.column,
            message: "comment: missing space after `//`".to_string(),
        }),
    }
}

fn comment_capital_violation(comment: &Comment_Line) -> Option<Violation> {
    match comment_body(&comment.text).chars().next() {
        // Empty body, or a non-letter lead (a digit, a path) — nothing to cap.
        None => None,
        Some(first) if !first.is_alphabetic() => None,
        Some(first) if first.is_uppercase() => None,
        Some(_) => Some(Violation {
            line: comment.line,
            column: comment.column,
            message: "comment: should start with capital letter".to_string(),
        }),
    }
}

fn comment_terminator_violation(comment: &Comment_Line) -> Option<Violation> {
    let body = comment_body(&comment.text);
    match body.trim_end_matches([' ', '\t']).chars().last() {
        None => None,
        Some('.' | ':' | '?' | '!') => None,
        Some(_) => Some(Violation {
            line: comment.line,
            column: comment.column,
            message: "comment: should end with `.`, `:`, `?`, or `!`".to_string(),
        }),
    }
}

/// The comment prose: the `//`/`///`/`//!` marker stripped — every leading
/// slash, then an optional `!` — and the result left-trimmed. Owned, since the
/// dialect forbids returning a borrow.
fn comment_body(text: &str) -> String {
    let after_slashes = text.trim_start_matches('/');
    let after_marker = after_slashes.strip_prefix('!').unwrap_or(after_slashes);
    after_marker.trim_start_matches([' ', '\t']).to_string()
}

fn has_space_after_marker(text: &str) -> bool {
    let after_slashes = text.trim_start_matches('/');
    let after_marker = after_slashes.strip_prefix('!').unwrap_or(after_slashes);
    after_marker.is_empty() || after_marker.starts_with([' ', '\t'])
}
