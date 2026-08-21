//go:build linux

package os

import (
	"os"
	"syscall"
)

func system_executable() (path string, err error) {
	return os.Executable()
}

// The runtime owns Args, so a caller must not receive its slice.
func system_arguments() (arguments []string) {
	arguments = make([]string, len(os.Args))
	copy(arguments, os.Args)
	return arguments
}

// Reads the machine name from uname. Go exports no Gethostname, and Linux has no gethostname
// trap of its own: libc reads the nodename field of uname, so uname is the syscall.
func system_hostname() (name string, err error) {
	identity := syscall.Utsname{}
	if uname_err := syscall.Uname(&identity); uname_err != nil {
		return "", uname_err
	}
	// Nodename is a fixed array the kernel NUL-terminates, so the name ends at the first zero
	// rather than at the end of the array.
	bytes := make([]byte, 0, len(identity.Nodename))
	for _, value := range identity.Nodename {
		if value == 0 {
			break
		}
		bytes = append(bytes, byte(value))
	}
	return string(bytes), nil
}
