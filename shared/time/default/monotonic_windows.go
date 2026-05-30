//go:build windows

package time

// Panics: the system clock is not yet supported on Windows. The faithful backend
// is QueryPerformanceCounter (it counts across suspend); this stub keeps the
// package building on Windows until that lands.
func monotonic_nanoseconds() (nanoseconds int64) {
	panic("time: the system clock is not yet supported on Windows")
}
