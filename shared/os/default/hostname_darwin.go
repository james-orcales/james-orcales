//go:build darwin

package os

import "syscall"

// Reads the machine name from the kern.hostname sysctl. Go exports no Gethostname, and Darwin
// has no gethostname trap of its own: libc reads this same sysctl, so the sysctl is the syscall.
func system_hostname() (name string, err error) {
	return syscall.Sysctl("kern.hostname")
}
