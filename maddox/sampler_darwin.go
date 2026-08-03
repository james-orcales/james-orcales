//go:build darwin && arm64

package main

/*
#include "sampler_darwin.h"
#include <stdlib.h>
*/
import "C"

import (
	"os"
	"runtime"
	"syscall"
	"unsafe"

	"local/james-orcales/maddox/internal"
	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/io"
	"local/james-orcales/shared/time"
	time_system "local/james-orcales/shared/time/default"
)

// System_sampler returns the production Sampler, whose Measure spawns each command
// and reads the Apple-Silicon hardware counters through proc_pid_rusage.
func system_sampler() (sampler maddox.Sampler) {
	defer func() { maddox.Sampler_Invariants(sampler, "system_sampler.sampler") }()
	sampler.Measure = measure_command
	return sampler
}

// Measure_command spawns the command, reads its hardware counters, and reports the
// Run_Result, timing the child's wall around the spawn with a monotonic clock. Main reads
// no clock of its own — only func main's driver advances one — so wall is measured here.
func measure_command(command io.Process_Request) (result maddox.Run_Result) {
	defer func() { maddox.Run_Result_Invariants(result, "measure_command.result") }()
	// Built outside the timed bracket, so the clock's own setup never counts as wall.
	clock, _ := time_system.New_Operating_System_Clock()
	argv_words := command_argv(command)
	envp_words := append(os.Environ(), command.Environment...)
	argv := build_c_array(argv_words)
	envp := build_c_array(envp_words)
	defer free_c_array(argv, Word_Count(len(argv_words)))
	defer free_c_array(envp, Word_Count(len(envp_words)))

	capture, create_err := os.CreateTemp("", "maddox-stderr-*")
	if create_err != nil {
		result.Exit = SPAWN_FAILURE_EXIT
		result.Stderr = []byte("maddox: cannot capture stderr\n")
		return result
	}
	defer os.Remove(capture.Name())
	defer capture.Close()

	before := clock.Now_Monotonic()
	counters := C.maddox_measure(argv, envp, C.int(capture.Fd()))
	after := clock.Now_Monotonic()
	wall := time.Duration(after - before)
	// The completion stamp drives the budget stopwatch, spanning the gaps between runs.
	result.Completed_At = maddox.Completion_Moment(after)
	if counters.spawn_errno != 0 {
		result.Exit = SPAWN_FAILURE_EXIT
		result.Stderr = []byte("maddox: cannot spawn " + command.Path + "\n")
		return result
	}

	result.Sample = maddox.Sample{
		Wall:          maddox.Metric(wall),
		RSS_Bytes_Max: maddox.Resident_Bytes(counters.peak_footprint),
		CPU_Cycles:    maddox.Cycle_Count(counters.cycles),
		Instructions:  maddox.Instruction_Count(counters.instructions),
		CPU_User:      maddox.User_Time(counters.user_ns),
		CPU_System:    maddox.System_Time(counters.system_ns),
	}
	result.Exit = maddox.Exit_Status(counters.exit_code)
	if result.Exit != 0 {
		result.Stderr = read_captured(capture)
	}
	return result
}

// Build_c_array copies words into a NULL-terminated C array of C strings for
// posix_spawnp. Free_c_array releases it.
func build_c_array(words Argv) (array **C.char) {
	Argv_Invariants(words, "build_c_array.words")
	pointer_size := C.size_t(unsafe.Sizeof((*C.char)(nil)))
	block := C.malloc(C.size_t(len(words)+1) * pointer_size)
	view := unsafe.Slice((**C.char)(block), len(words)+1)
	for index, word := range words {
		view[index] = C.CString(word)
	}
	view[len(words)] = nil
	return (**C.char)(block)
}

// Free_c_array releases the word_count C strings Build_c_array allocated and the
// array holding them; the trailing NULL is not a C string.
func free_c_array(array **C.char, word_count Word_Count) {
	Word_Count_Invariants(word_count, "free_c_array.word_count")
	view := unsafe.Slice(array, int(word_count)+1)
	for index := 0; index < int(word_count); index++ {
		C.free(unsafe.Pointer(view[index]))
	}
	C.free(unsafe.Pointer(array))
}

// Acquire_machine_specs reads the host CPU, memory, and OS details via Darwin
// sysctls. Fields the kernel does not expose — like L3 cache on Apple Silicon —
// are left zero and omitted from the report via omitempty.
func acquire_machine_specs() (specs maddox.Machine_Specs) {
	defer func() { maddox.Machine_Specs_Invariants(specs, "acquire_machine_specs.specs") }()
	model, _ := syscall.Sysctl("machdep.cpu.brand_string")
	specs.CPU_Model = maddox.Processor_Model(model)
	specs.CPU_Arch = maddox.Processor_Architecture(runtime.GOARCH)

	physical, _ := syscall.SysctlUint32("hw.physicalcpu")
	specs.Physical_Cores = maddox.Physical_Core_Count(physical)
	logical, _ := syscall.SysctlUint32("hw.logicalcpu")
	specs.Logical_Cores = maddox.Logical_Core_Count(logical)

	// Apple Silicon exposes P-cores at perflevel0 and E-cores at perflevel1.
	p_cores, p_err := syscall.SysctlUint32("hw.perflevel0.physicalcpu")
	e_cores, e_err := syscall.SysctlUint32("hw.perflevel1.physicalcpu")
	if p_err == nil {
		if e_err == nil {
			specs.Performance_Cores = maddox.Performance_Core_Count(p_cores)
			specs.Efficiency_Cores = maddox.Efficiency_Core_Count(e_cores)
		}
	}

	// Frequency and caches are 64-bit sysctls, read through cgo since SysctlUint32
	// truncates and the stdlib offers no raw form.
	specs.CPU_Frequency_Hz_Max = maddox.Processor_Frequency(
		sysctl_uint64("hw.perflevel0.cpufrequency_max"))
	specs.Cache_L1_Bytes = maddox.Level_1_Cache_Size(
		sysctl_uint64("hw.perflevel0.l1dcachesize"))
	specs.Cache_L2_Bytes = maddox.Level_2_Cache_Size(
		sysctl_uint64("hw.perflevel0.l2cachesize"))
	// L3 is absent on Apple Silicon; an absent key reads zero and is omitted.
	specs.Cache_L3_Bytes = maddox.Level_3_Cache_Size(sysctl_uint64("hw.l3cachesize"))
	specs.RAM_Total_Bytes = maddox.Memory_Size(sysctl_uint64("hw.memsize"))
	specs.Storage_Total_Bytes = maddox.Storage_Size(boot_volume_bytes())

	specs.Operating_System_Name = "macOS"
	version, _ := syscall.Sysctl("kern.osproductversion")
	specs.Operating_System_Version = maddox.Operating_System_Version(version)
	kernel, _ := syscall.Sysctl("kern.osrelease")
	specs.Kernel_Version = maddox.Kernel_Version("Darwin " + kernel)
	return specs
}

// Sysctl_Key is the name of a sysctl reading, e.g. "hw.memsize".
type Sysctl_Key string

// Sysctl_Key_Invariants bounds the key's length.
func Sysctl_Key_Invariants(name Sysctl_Key, namespace invariant.Namespace) {
	invariant.Tree(name, namespace).Range_Int(len(name), BOUND_MIN, BOUND_MAX).Ensure()
}

// SYSCTL_VALUE_MIN is the smallest unsigned sysctl scalar.
const SYSCTL_VALUE_MIN uint64 = 0

// SYSCTL_VALUE_MAX is the largest unsigned sysctl scalar.
const SYSCTL_VALUE_MAX uint64 = 1<<64 - 1

// Sysctl_Value is one raw unsigned scalar from sysctlbyname.
type Sysctl_Value uint64

// Sysctl_Value_Invariants accepts the complete unsigned sysctl scalar domain.
func Sysctl_Value_Invariants(value Sysctl_Value, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Uint64(uint64(value), SYSCTL_VALUE_MIN, SYSCTL_VALUE_MAX).
		Ensure()
}

// Sysctl_uint64 reads a 64-bit sysctl by name, marshaling the Go string across the
// cgo boundary and freeing the C copy after the call.
func sysctl_uint64(name Sysctl_Key) (value Sysctl_Value) {
	defer func() { Sysctl_Value_Invariants(value, "sysctl_uint64.value") }()
	Sysctl_Key_Invariants(name, "sysctl_uint64.name")
	cname := C.CString(string(name))
	defer C.free(unsafe.Pointer(cname))
	return Sysctl_Value(C.maddox_sysctl_uint64(cname))
}

// Boot_volume_bytes is the boot filesystem's total capacity, taken from statfs on
// the root. On APFS this reports the shared container size, a close proxy for the
// physical SSD capacity — enough to tell a 256GB drive from a 512GB one. A failed
// statfs reads zero.
func boot_volume_bytes() (total maddox.Byte_Size) {
	defer func() { maddox.Byte_Size_Invariants(total, "boot_volume_bytes.total") }()
	var stat syscall.Statfs_t
	if syscall.Statfs("/", &stat) != nil {
		return 0
	}
	return maddox.Byte_Size(uint64(stat.Bsize) * stat.Blocks)
}
