package simulation_test

import "testing"

// Test_Main sweeps the seed corpus, driving Main over each seed's filesystem and
// asserting every invariant holds; `go test -fuzz=Fuzz_Main` explores beyond the corpus.
func Test_Main(t *testing.T) {
	for seed := uint64(0); seed < HARNESS_CORPUS; seed++ {
		drive(t, seed)
	}
}
