//go:build darwin

package os

import (
	"os"
	"syscall"
)

func system_executable() (path string, err error) {
	return os.Executable()
}

// The runtime owns Args, so a caller must not receive its slice.
func system_arguments(destination []string) (count int) {
	return copy(destination, os.Args)
}

func system_argument_count() (count int) {
	return len(os.Args)
}

// Reads the machine name from the kern.hostname sysctl. Go exports no Gethostname, and Darwin
// has no gethostname trap of its own: libc reads this same sysctl, so the sysctl is the syscall.
func system_hostname() (name string, err error) {
	return syscall.Sysctl("kern.hostname")
}
