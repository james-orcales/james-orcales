package ecdsa_test

import (
	"testing"

	"local/james-orcales/shared/simulation/aver/default"
)

// TestMain makes public operations prove their invariant paths.
func TestMain(m *testing.M) {
	aver.Run_Test_Main(m)
}
