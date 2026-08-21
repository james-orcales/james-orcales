//go:build invariant_noop && !invariant_disable_coverage && !prd && !prod && !production

package invariant_flatjson

// Always stays inert because invariant_noop exists only to measure assertion overhead.
func Always[T ~bool](condition T, message string) {
	return
}
