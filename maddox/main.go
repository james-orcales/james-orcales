// Package main binds the operating-system dependencies for Maddox.
package main

import (
	"os"

	maddox "local/james-orcales/maddox/internal"
)

// BOUND_MIN leaves zero as a witnessed interior shape for root-owned values.
const BOUND_MIN = -1

// BOUND_MAX caps root-owned command and platform buffers.
const BOUND_MAX = 1 << 16

func main() {
	os.Exit(int(maddox.Main(maddox.Main_Input{
		Arguments:    maddox.Arguments(os.Args),
		Sampler:      system_sampler(),
		Output:       os.Stdout,
		Error_Output: os.Stderr,
		Output_Is_Terminal: maddox.Output_Terminal_Status(
			is_terminal(os.Stdout)),
		Error_Output_Is_Terminal: maddox.Error_Output_Terminal_Status(
			is_terminal(os.Stderr)),
		Machine: acquire_machine_specs(),
	})))
}

// Is_terminal reports whether the file is a character device, so color and progress
// are suppressed when the stream is piped or redirected to a file.
func is_terminal(file *os.File) (terminal maddox.Terminal_Status) {
	defer func() { maddox.Terminal_Status_Invariants(terminal, "is_terminal.terminal") }()
	stat, stat_err := file.Stat()
	if stat_err != nil {
		return maddox.Terminal_Status(maddox.TERMINAL_STATUS_NOT_TERMINAL)
	}
	if stat.Mode()&os.ModeCharDevice != 0 {
		return maddox.Terminal_Status(maddox.TERMINAL_STATUS_TERMINAL)
	}
	return maddox.Terminal_Status(maddox.TERMINAL_STATUS_NOT_TERMINAL)
}
