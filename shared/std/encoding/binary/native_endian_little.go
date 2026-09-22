//go:build amd64 || arm64 || loong64 || ppc64le || riscv64 || wasm

package binary

// NATIVE_BYTE_ORDER stays separate because NATIVE_ENDIAN retains own public identity.
const NATIVE_BYTE_ORDER = LITTLE_ENDIAN
