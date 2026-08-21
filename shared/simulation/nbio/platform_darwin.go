//go:build darwin

package nbio

// Platform_IO is empty on Darwin, because Darwin backend has no statx operation.
type Platform_IO struct{}

// Wire no platform-only simulator operation on Darwin.
func sim_wire_platform(state *Sim, loop *IO) {
	loop.Platform_IO = Platform_IO{}
}
