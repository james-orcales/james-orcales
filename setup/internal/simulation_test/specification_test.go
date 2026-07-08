package simulation_test

import "testing"

// Test_Mirror sweeps the seed corpus, driving the mirror over each seed's filesystem and
// asserting every invariant holds; `go test -fuzz=Fuzz_Main` explores beyond the corpus.
func Test_Mirror(t *testing.T) {
	for seed := uint64(0); seed < HARNESS_CORPUS; seed++ {
		drive(t, seed)
	}
}
