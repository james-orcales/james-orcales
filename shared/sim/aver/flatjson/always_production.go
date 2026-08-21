//go:build (invariant_disable_coverage || prd || prod || production) && !invariant_noop

package aver_flatjson

import "local/james-orcales/shared/sim/aver"

// Always preserves lazy production failure formatting without adding recorder ownership.
func Always[T ~bool](condition T, message string) {
	if !condition {
		panic(aver.Assertion_Failure{
			Identity: message,
			Reason:   "  Always — condition was false",
			Value:    condition,
		})
	}
}
