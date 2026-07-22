//go:build noassert

// This file is the silent half of the compile-time dual described in enforcement.go: the same
// entry points, the same signatures, and bodies that do nothing. Selecting it with `-tags noassert`
// yields a binary in which no condition is enforced and no coverage is recorded.
//
// The tag is for measuring what enforcement costs and for a build that has already been measured
// and cannot afford it. It is not a release default: the framework's contract is that enforcement
// fires in every run mode, and a binary built this way answers "would this have held?" with
// silence. Nothing gates the tag, so choosing it is choosing to ship unchecked.

package invariant

// Always is the one guard that fires without a chain, so its silence is the tag's whole meaning.
func Recorder_Always[T ~bool](recorder *Recorder, condition T, message string) {}

func Recorder_Sometimes[T ~bool](
	recorder *Recorder, identifier string, condition T, message string,
) {
}

func Recorder_Range[T Integer](
	recorder *Recorder, identifier string, value T, minimum T, maximum T, excluded ...T,
) {
}

func Recorder_Enum[T Integer](recorder *Recorder, identifier string, value T, members ...T) {}
