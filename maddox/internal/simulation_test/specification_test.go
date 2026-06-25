package simulation_test

import "testing"

// The specification tests mirror SPECIFICATION.md: one leaf per heading, in heading
// order, each driving internal.Main through the shared scenario driver. They assert no
// invariant trips; the coverage recorder judges the boundaries the fuzz target reaches.

// Test_Witness_Pipeline drives a full benchmark run — sampling, statistics, comparison,
// and rendering — through Main from the base scenario.
func Test_Witness_Pipeline(t *testing.T) {
	drive(decode_scenario(build(base_scenario())))
}

// Test_Witness_Failure drives a command that exits non-zero, exercising the abort and
// captured-stderr paths a successful run never reaches.
func Test_Witness_Failure(t *testing.T) {
	drive(decode_scenario(with(func(s *scenario) {
		s.Exit = 1
		s.Fail_Every = 1
		s.Stderr_Bytes = 4
	})))
}
