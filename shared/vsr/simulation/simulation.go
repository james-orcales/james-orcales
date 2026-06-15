// Package simulation is a deterministic, seeded fuzzer for the vsr protocol core. It drives a
// cluster of replicas through message loss, reordering, duplication, delay, partition, and
// crash-restart on a virtual clock, asserting the whole-cluster safety properties after every
// delivery and reproducing any run exactly from its seed.
//
// It lives apart from vsr for two reasons. The cross-cluster properties it checks — agreement and
// single-primary — cannot be evaluated from one replica's own state, so they belong outside the
// step function. And it parallelizes across seeds, which needs goroutines the deterministic vsr
// package forbids even in its tests; a separate, non-deterministic package is what unlocks that.
package simulation
