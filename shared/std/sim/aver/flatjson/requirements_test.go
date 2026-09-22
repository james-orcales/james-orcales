//go:build !invariant_disable_coverage && !prd && !prod && !production && !invariant_noop

package aver_flatjson

import "testing"

// Test_Always verifies that the encoder assertion seam retains failure enforcement.
func Test_Always(t *testing.T) {
  const MESSAGE = "The flat JSON encoder assertion failed."
  Always(true, MESSAGE)
  defer func() {
    check_recovered(t, recover())
  }()
  Always(false, MESSAGE)
}

func check_recovered(t *testing.T, recovered any) {
  if recovered == nil {
    t.Fatal("a false encoder assertion did not panic")
  }
}
