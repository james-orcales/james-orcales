//go:build linux

package path

import "local/james-orcales/shared/math/bits"

// PATH_SIZE_MAXIMUM leaves Linux PATH_MAX terminating NUL outside pathname bytes.
const PATH_SIZE_MAXIMUM = 4*bits.KIBIBYTE_BYTES - PATH_TERMINATOR_BYTES
