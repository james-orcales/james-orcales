//! Source lines of code counter — a faithful Rust-dialect twin of the Go `sloc`
//! and its OCaml port. A generic per-line scanner, configured by a `Language`,
//! partitions every physical line into code / comment / blank; a tree walk
//! classifies recognized files across scoped threads; a renderer prints an
//! aligned table or JSON. The whole library is one module, mirroring the
//! one-file Go and OCaml originals.

pub mod sloc;
