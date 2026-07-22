package simulation_test

import "testing"

// The specification tests mirror SPECIFICATION.md: one leaf per heading, in heading
// order, each driving internal.Main through the shared scenario driver. They assert no
// invariant trips; the coverage recorder judges the boundaries the fuzz target reaches.

// Test_Witness_Tree drives a walked directory through classification and both
// renderers from one Main call.
func Test_Witness_Tree(t *testing.T) {
	drive(decode_scenario(build(base_scenario())))
	drive(decode_scenario(with(func(one *scenario) { one.Show_Files = true })))
	drive(decode_scenario(with(func(one *scenario) { one.Json = true })))
}

// Test_Witness_Languages counts one file per recognized extension and bare name, so
// every language constructor runs.
func Test_Witness_Languages(t *testing.T) {
	drive(decode_scenario(with(func(one *scenario) { one.Recipe = 2 })))
	drive(decode_scenario(with(func(one *scenario) {
		one.Recipe = 2
		one.Show_Files = true
	})))
}

// Test_Witness_Boundaries counts a line at the scan window, a tree at the file bound,
// and a table whose every column reaches its width.
func Test_Witness_Boundaries(t *testing.T) {
	drive(decode_scenario(with(func(one *scenario) {
		one.Recipe = 3
		one.Line_Bytes = 4096
	})))
	drive(decode_scenario(with(func(one *scenario) {
		one.Recipe = 10
		one.Show_Files = true
	})))
}

// Test_Witness_Failure drives the stat failure, the read failure, and the trees whose
// files are dropped as binary, oversized, or unrecognized.
func Test_Witness_Failure(t *testing.T) {
	drive(decode_scenario(with(func(one *scenario) { one.Stat_Error = true })))
	drive(decode_scenario(with(func(one *scenario) {
		one.Explicit = true
		one.Read_Error = true
		one.Root_Bytes = 7
	})))
	drive(decode_scenario(with(func(one *scenario) { one.Recipe = 7 })))
	drive(decode_scenario(with(func(one *scenario) { one.Recipe = 11 })))
	drive(decode_scenario(with(func(one *scenario) { one.Recipe = 12 })))
}
