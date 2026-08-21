//go:build !invariant_disable_coverage && !prd && !prod && !production && !invariant_noop

package aver_flatjson

import "local/james-orcales/shared/sim/aver"

// Always enforces encoder assertions locally because recording would create false obligations.
func Always[T ~bool](condition T, message string) {
	if !condition {
		panic(aver.ASSERTION_FAILURE_MESSAGE_PREFIX + message +
			"  Always — condition was false: false")
	}
}
