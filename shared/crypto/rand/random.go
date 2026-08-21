// Package random fills bounded caller storage from an injected cryptographic generator.
package random

import (
	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/random/csprng"
)

// DESTINATION_SIZE_MINIMUM admits an empty entropy request.
const DESTINATION_SIZE_MINIMUM = csprng.SINK_MIN

// DESTINATION_SIZE_MAXIMUM preserves the injected generator's work bound.
const DESTINATION_SIZE_MAXIMUM = csprng.SINK_MAX

// COUNT_MINIMUM reports empty output.
const COUNT_MINIMUM Count = Count(DESTINATION_SIZE_MINIMUM)

// COUNT_MAXIMUM reports one complete largest draw.
const COUNT_MAXIMUM Count = Count(DESTINATION_SIZE_MAXIMUM)

// Generator is a nonnil caller-owned cryptographic source handle.
type Generator *csprng.Generator

// Generator_Invariants composes the injected source state after proving storage exists.
func Generator_Invariants(value Generator, namespace invariant.Namespace) {
	invariant.Always(value != nil, "A random Generator has caller-owned state.")
	csprng.Generator_Invariants(*value, namespace)
}

// Destination is bounded caller-owned entropy storage.
type Destination []byte

// Destination_Invariants preserves the source generator's bounded sink domain.
func Destination_Invariants(value Destination, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(
			len(value), DESTINATION_SIZE_MINIMUM, DESTINATION_SIZE_MAXIMUM,
		).
		Ensure()
}

// Count is bytes filled by one complete draw.
type Count int

// Count_Invariants spans empty through largest bounded output.
func Count_Invariants(value Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), int(COUNT_MINIMUM), int(COUNT_MAXIMUM)).
		Ensure()
}

// Read keeps operating-system choice outside the library by requiring caller-owned state.
func Read(generator Generator, destination Destination) (count Count) {
	defer func() { Count_Invariants(count, "Read.count") }()
	Generator_Invariants(generator, "Read.generator.input")
	Destination_Invariants(destination, "Read.destination")
	if generator == nil {
		panic("rand: generator is nil")
	}
	if len(destination) > DESTINATION_SIZE_MAXIMUM {
		panic("rand: destination exceeds bound")
	}
	filled, read_error := (*csprng.Generator)(generator).Read(destination)
	invariant.Always(read_error == nil, "Injected cryptographic entropy cannot fail.")
	if read_error != nil {
		panic("rand: generator read failed")
	}
	invariant.Always(filled == len(destination), "Injected entropy fills complete storage.")
	if filled != len(destination) {
		panic("rand: generator returned partial output")
	}
	Generator_Invariants(generator, "Read.generator.output")
	return Count(filled)
}
