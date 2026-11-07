package simulation_test

import (
	"testing"

	fuzzstall "local/james-orcales/fuzzstall/internal"
)

// Test_Main_Echo verifies every byte the run hands over reaches this program's output, in
// order, whatever chunk boundary the seed puts in the middle of a line.
func Test_Main_Echo(t *testing.T) {
	for seed := range uint64(HARNESS_CORPUS) {
		outcome := harness_drive(new_harness_model(seed))
		if outcome.Output == outcome.Delivered {
			continue
		}
		t.Fatalf("seed %d echoed %d bytes of the %d handed over",
			seed, len(outcome.Output), len(outcome.Delivered))
	}
}

// Test_Main_Saturation verifies a stretch as long as the window with no new interesting input
// terminates the run and reports 124, and that a run with no such stretch keeps its own exit
// code and is never terminated.
func Test_Main_Saturation(t *testing.T) {
	for seed := range uint64(HARNESS_CORPUS) {
		model := new_harness_model(seed)
		if model.Configuration.Malformed {
			continue
		}
		if model.Configuration.Unstarted {
			continue
		}
		outcome := harness_drive(model)
		stall, started := harness_stall(model)
		saturated := started
		if stall < model.Window {
			saturated = false
		}
		if saturated {
			if outcome.Status_Code != fuzzstall.EXIT_SATURATED {
				t.Fatalf("seed %d: exit = %d, want %d, stall %d over window %d",
					seed, outcome.Status_Code, fuzzstall.EXIT_SATURATED,
					stall, model.Window)
			}
			if !outcome.Terminated {
				t.Fatalf("seed %d: a saturated run must be terminated", seed)
			}
			continue
		}
		if outcome.Status_Code != model.Configuration.Status_Code {
			t.Fatalf("seed %d: exit = %d, want the run's own %d (stall %d, window %d)",
				seed, outcome.Status_Code, model.Configuration.Status_Code,
				stall, model.Window)
		}
		if outcome.Terminated {
			t.Fatalf("seed %d: a run that kept finding inputs stays alive", seed)
		}
	}
}

// Test_Main_Refusal verifies a window that names no span reports 2 and starts nothing, and
// that a run which cannot start at all reports 125.
func Test_Main_Refusal(t *testing.T) {
	usage_seen := false
	failure_seen := false
	for seed := range uint64(HARNESS_CORPUS) {
		model := new_harness_model(seed)
		if model.Configuration.Malformed {
			usage_seen = true
			outcome := harness_drive(model)
			if outcome.Status_Code != fuzzstall.EXIT_USAGE {
				t.Fatalf("seed %d: exit = %d, want %d for a window naming no span",
					seed, outcome.Status_Code, fuzzstall.EXIT_USAGE)
			}
			if outcome.Delivered != "" {
				t.Fatalf("seed %d: a refused window must start no run", seed)
			}
			continue
		}
		if !model.Configuration.Unstarted {
			continue
		}
		failure_seen = true
		outcome := harness_drive(model)
		if outcome.Status_Code != fuzzstall.EXIT_START_FAILURE {
			t.Fatalf("seed %d: exit = %d, want %d for a run that never started",
				seed, outcome.Status_Code, fuzzstall.EXIT_START_FAILURE)
		}
	}
	if !usage_seen {
		t.Fatalf("the sweep reached no window that names a span")
	}
	if !failure_seen {
		t.Fatalf("the sweep reached no run that refused to start")
	}
}

// Fuzz_Main drives Main over a model built from a fuzzer-supplied seed, so the corpus grows
// past the sweep's own seeds while the property it checks stays the same one.
func Fuzz_Main(f *testing.F) {
	f.Add(uint64(0))
	f.Fuzz(func(t *testing.T, seed uint64) {
		outcome := harness_drive(new_harness_model(seed))
		if outcome.Output == outcome.Delivered {
			return
		}
		t.Fatalf("seed %d echoed %d of the %d bytes handed over",
			seed, len(outcome.Output), len(outcome.Delivered))
	})
}
