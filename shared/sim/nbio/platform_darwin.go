//go:build darwin

package nbio

// Platform_IO is empty on Darwin, because Darwin backend has no statx operation.
type Platform_IO struct{}

// Wire no platform-only simulator operation on Darwin.
func sim_wire_platform(_ *Sim, loop *IO) {
	loop.Platform_IO = Platform_IO{}
}

func sim_statx_operation_complete(_ *Sim_Operation) (data int, err error) {
	panic("io: Darwin has no statx operation")
}
