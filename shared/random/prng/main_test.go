package prng_test

import (
	"testing"

	"local/james-orcales/shared/invariant/default"
)

// TestMain makes bounded random domains fail when public tests miss a witness.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}
