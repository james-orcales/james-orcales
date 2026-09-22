//go:build darwin

package path

import "local/james-orcales/shared/math/bits"

// PATH_SIZE_MAXIMUM leaves Darwin PATH_MAX terminating NUL outside pathname bytes.
const PATH_SIZE_MAXIMUM = bits.KIBIBYTE_BYTES - PATH_TERMINATOR_BYTES
