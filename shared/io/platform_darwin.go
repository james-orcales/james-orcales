//go:build darwin

package io

// Platform_IO is empty on Darwin because TigerBeetle's Darwin backend has no statx operation.
type Platform_IO struct{}

// Wires no platform-only simulator operations on Darwin.
func sim_wire_platform(state *sim, loop *IO) {
	loop.Platform_IO = Platform_IO{}
}
