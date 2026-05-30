//go:build sim || sim.fault_injection

package sim

import (
	"errors"
	"fmt"
	"os"

	"github.com/james-orcales/james-orcales/shared/invariant"
	"github.com/james-orcales/james-orcales/shared/xdebug"
)

const (
	FaultErrorPrefix = "Fault Injected: "

	Kilobyte = 1 * 1000
	Megabyte = Kilobyte * 1000
	Gigabyte = Megabyte * 1000
	Terabyte = Gigabyte * 1000

	Kibibyte = 1 * 1024
	Mebibyte = Kibibyte * 1024
	Gibibyte = Mebibyte * 1024
	Tebibyte = Gibibyte * 1024
)

// Fault probabilities are grouped by subsystems/devices. This allows targeted fault injection to
// model cascading or correlated failures. For example, progressively increase disk IO faults to emulate
// a degrading SSD while leaving other IO paths mostly healthy.
var (
	FaultChanceGeneric float32 = 0.10

	FaultChanceFatal            float32 = 0.00
	FaultChancePanic            float32 = 0.01
	FaultChanceAssertionFailure float32 = 0.01

	FaultChanceIOGeneric float32 = 0.10
	FaultChanceIODisk    float32 = 0.10
	FaultChanceIONetwork float32 = 0.10

	FaultChanceLatency      float32  = 0.10
	FaultSeverityLatencyMin Duration = 100 * Microsecond
	FaultSeverityLatencyMax Duration = 5 * Second

	FaultChanceMemorySpike   float32 = 0.05
	FaultSeverityMemorySpike int     = 50 * Megabyte
)

// Err randomly returns an error based on FaultChanceGeneric.
//
// Usage:
//
//	err := fallible()
//	if sim.Err(&err) != nil {
//		handleError(err)
//	}
func Err(err *error) error {
	return ErrN(FaultChanceGeneric, err)
}

func ErrN(chance float32, err *error) error {
	if err == nil && determine(chance) {
		*err = errors.New(FaultErrorPrefix + "Generic error")
	}
	return *err
}

func True() bool {
	return TrueN(FaultChanceGeneric)
}

func TrueN(chance float32) bool {
	return determine(chance)
}

// Fatal is disabled by default.
func Fatal() {
	FatalN(FaultChanceFatal)
}

func FatalN(chance float32) {
	if determine(chance) {
		xdebug.FprintStackTrace(os.Stderr)
		fmt.Fprintln(os.Stderr, FaultErrorPrefix+"Fatal")
		os.Exit(1)
	}
}

func Panic() {
	PanicN(FaultChancePanic)
}

func PanicN(chance float32) {
	if determine(chance) {
		panic(FaultErrorPrefix + "Panic")
	}
}

func AssertionFailure() {
	AssertionFailureN(FaultChanceAssertionFailure)
}

func AssertionFailureN(chance float32) {
	if !invariant.AssertionFailureIsFatal && determine(chance) {
		invariant.Ensure(false, "Fault Injected")
	}
}

func IOErr(err *error) error {
	return IOErrN(FaultChanceGeneric, err)
}

func IOErrN(chance float32, err *error) error {
	if err == nil && determine(chance) {
		*err = errors.New(FaultErrorPrefix + "IO error (Generic)")
	}
	return *err
}

func IODiskErr(err *error) error {
	return IODiskErrN(FaultChanceIODisk, err)
}

func IODiskErrN(chance float32, err *error) error {
	if err == nil && determine(chance) {
		*err = errors.New(FaultErrorPrefix + "IO error (Disk)")
	}
	return *err
}

// TODO: Add latency
func IONetworkErr(err *error) error {
	return IONetworkErrN(FaultChanceIONetwork, err)
}

func IONetworkErrN(chance float32, err *error) error {
	if err == nil && determine(chance) {
		*err = errors.New(FaultErrorPrefix + "IO error (Network)")
	}
	return *err
}

func Latency() {
	LatencyN(FaultChanceLatency, FaultSeverityLatencyMin, FaultSeverityLatencyMax)
}

func LatencyN(chance float32, lo, hi Duration) {
	if determine(chance) {
		sim.Propel(random(lo, hi))
	}
}

func MemorySpike(release <-chan struct{}) {
	MemorySpikeN(FaultChanceMemorySpike, FaultMemorySpikeBytes, release)
}

// Usage:
//
//	release := make(chan struct{})
//	go simulation.MemorySpike(50*1024*1024, release) // 50 MB spike
//	// do work under memory pressure
//	close(release) // release the memory
//
// TODO: Verify if this gets optimized away
func MemorySpikeN(chance float32, n int, release <-chan struct{}) {
	if determine(chance) {
		garbage := make([]byte, n)
		<-release
		garbage[0] = 42
	}
}
