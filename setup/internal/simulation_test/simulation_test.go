// Package simulation_test drives setup.Main over the filesystem New_Sim fabricates from each
// seed and asserts the mirror's properties hold for any seed: convergence and idempotency,
// pruning of an ignored subtree, and rewriting only files that differ. New_Sim draws a
// generic tree; this harness imposes the source, destination, and ignored meaning the mirror
// needs, so the library stays free of any consumer's layout.
package simulation_test

import (
	"path/filepath"
	"strings"
	"testing"

	setup "local/james-orcales/setup/internal"
	sysio "local/james-orcales/shared/io"
	"local/james-orcales/shared/prng"
	systime "local/james-orcales/shared/time"
)

// The Run_Until cap for the harness's own writes, in virtual time: the write and close
// finish in a few grains, so this ample bound only bites a bug that stalls one.
const HARNESS_DEADLINE = systime.MICROSECOND

// The destination directory, relative to the source root, pruned from the source walk so the
// mirror never copies its own output back into itself.
const HARNESS_DESTINATION = "dest"

// A top-level entry marked ignored, so the harness can prove the mirror prunes it.
const HARNESS_IGNORED = "e0"

// Content written over a destination file to force a difference no random source file
// realistically matches, so a later run rewrites exactly the mutated files.
const HARNESS_POISON = "harness-poison-differs-from-any-generated-file"

// The number of seeds the corpus adds, so `go test` alone sweeps a range before `-fuzz`.
const HARNESS_CORPUS = 256

// Fuzz_Main drives the mirror over the filesystem New_Sim draws from each seed.
func Fuzz_Main(f *testing.F) {
	for seed := uint64(0); seed < HARNESS_CORPUS; seed++ {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, seed uint64) {
		drive(t, seed)
	})
}

// Drives setup.Main over New_Sim(seed) and asserts every mirror invariant: a second run over
// the freshly mirrored tree writes nothing (convergence, idempotency, no rewrite of an equal
// file), the ignored subtree never reaches the destination (prune), and after some
// destination files are made to differ, a run rewrites exactly those (overwrite).
func drive(t *testing.T, seed uint64) {
	loop, driver, _ := sysio.New_Sim(seed)
	system := setup.File_System{Loop: loop, Run_Until: driver.Run_Until}
	input := &setup.Main_Input{
		File_System:           system,
		Source_Directory:      "/",
		Destination_Directory: filepath.Join("/", HARNESS_DESTINATION),
		Operating_System:      "linux",
		Run_Command:           harness_run_command,
		Is_Ignored:            harness_ignore,
	}
	// The first run must succeed for every well-formed seed — with no faults injected, Main
	// fails only on a Plan or write error, neither reachable here. If that ever changes this
	// catches it instead of skipping the seed blind. When faults land it becomes an explicit,
	// reason-carrying, mostly-false skip — never a silent return.
	if setup.Main(input) != 0 {
		t.Fatal("the first mirror run over a fault-free tree failed")
	}
	writes := harness_count_writes(&system)
	input.File_System = system
	if setup.Main(input) != 0 {
		t.Fatal("the second mirror run failed")
	}
	if *writes != 0 {
		t.Fatalf("a converged mirror rewrote %d files on a repeat run", *writes)
	}
	harness_assert_pruned(t, &system)
	mutated := harness_mutate(&system, seed)
	*writes = 0
	if setup.Main(input) != 0 {
		t.Fatal("the mirror run after mutation failed")
	}
	if *writes != mutated {
		t.Fatalf("rewrote %d files, want the %d made to differ", *writes, mutated)
	}
}

// A no-op external-program runner: the mirror simulation asserts filesystem properties, not
// the macos-defaults path, so the injected runner does nothing.
func harness_run_command(name string, arguments []string) (err error) {
	return nil
}

// Classifies a batch of source paths, returning the set that is the destination root or the
// ignored subtree — pruned so the mirror neither copies into itself nor syncs the ignored
// entry. Batched to match the Is_Ignored seam the mirror now calls once per tree level.
func harness_ignore(relatives []string) (ignored map[string]bool) {
	ignored = map[string]bool{}
	for _, relative := range relatives {
		if harness_ignores_one(relative) {
			ignored[relative] = true
		}
	}
	return ignored
}

// Reports whether one source path is the destination root or the ignored subtree, the
// per-path rule the harness's own walk shares with the batched predicate.
func harness_ignores_one(relative string) (ignored bool) {
	if relative == HARNESS_DESTINATION {
		return true
	}
	if strings.HasPrefix(relative, HARNESS_DESTINATION+"/") {
		return true
	}
	if relative == HARNESS_IGNORED {
		return true
	}
	return strings.HasPrefix(relative, HARNESS_IGNORED+"/")
}

// Wraps the loop's Create to count the files a mirror run writes, returning the live count.
func harness_count_writes(system *setup.File_System) (writes *int) {
	count := 0
	create := system.Loop.Create
	system.Loop.Create = func(path string) (file sysio.File, err error) {
		count++
		return create(path)
	}
	return &count
}

// Asserts the ignored subtree never reached the destination.
func harness_assert_pruned(t *testing.T, system *setup.File_System) {
	status, _ := system.Loop.Status(filepath.Join("/", HARNESS_DESTINATION, HARNESS_IGNORED))
	if status.Exists {
		t.Fatalf("the ignored subtree %q was synced to the destination", HARNESS_IGNORED)
	}
}

// Overwrites a seed-chosen subset of the destination's mirrored files with poison so they
// differ from their source, returning how many were changed.
func harness_mutate(system *setup.File_System, seed uint64) (count int) {
	generator := prng.New(seed)
	for _, relative := range harness_source_files(system) {
		if prng.Generator_Boolean(&generator) {
			continue
		}
		harness_overwrite(system, filepath.Join("/", HARNESS_DESTINATION, relative))
		count++
	}
	return count
}

// Lists the non-ignored source files under the root, walking the tree through Read_Directory.
func harness_source_files(system *setup.File_System) (relatives []string) {
	relatives = []string{}
	worklist := []string{"."}
	for len(worklist) > 0 {
		directory := worklist[len(worklist)-1]
		worklist = worklist[:len(worklist)-1]
		entries, err := system.Loop.Read_Directory(filepath.Join("/", directory))
		if err != nil {
			continue
		}
		for _, entry := range entries {
			relative := filepath.Join(directory, entry.Name)
			if harness_ignores_one(relative) {
				continue
			}
			if entry.Is_Directory {
				worklist = append(worklist, relative)
				continue
			}
			relatives = append(relatives, relative)
		}
	}
	return relatives
}

// Overwrites path with the poison content through the loop, driving the write and the close
// to completion.
func harness_overwrite(system *setup.File_System, path string) {
	file, create_err := system.Loop.Create(path)
	if create_err != nil {
		return
	}
	var write_completion sysio.Completion
	written := false
	system.Loop.Write(&write_completion, func(_ *sysio.Completion, _ int, _ error) {
		written = true
	}, file, []byte(HARNESS_POISON), 0)
	system.Run_Until(func() (finished bool) { return written }, HARNESS_DEADLINE)
	var close_completion sysio.Completion
	closed := false
	system.Loop.Close(&close_completion, func(_ *sysio.Completion, _ error) {
		closed = true
	}, file)
	system.Run_Until(func() (finished bool) { return closed }, HARNESS_DEADLINE)
}
