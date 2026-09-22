//go:build arm64be || ppc64 || s390x || sparc64

package binary

// NATIVE_BYTE_ORDER stays separate because NATIVE_ENDIAN retains own public identity.
const NATIVE_BYTE_ORDER = BIG_ENDIAN
