//go:build linux

package main

/*
#include "sampler_linux.h"
#include <stdlib.h>
*/
import "C"

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"local/james-orcales/maddox/internal"
	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/io"
	"local/james-orcales/shared/time"
	time_system "local/james-orcales/shared/time/default"
)

// System_sampler returns the production Sampler, whose Measure spawns each command
// under perf_event_open and reads its hardware counters.
func system_sampler() (sampler maddox.Sampler) {
	defer func() { maddox.Sampler_Invariants(sampler, "system_sampler.sampler") }()
	sampler.Measure = measure_command
	return sampler
}

// Measure_command resolves the executable, spawns it under the perf counters, and
// reports the Run_Result. PATH is resolved here so the child can execve an absolute
// path with no malloc between fork and exec. The child's wall is timed around the spawn
// with a monotonic clock — Main reads none of its own, only func main's driver ticks.
func measure_command(command io.Process_Request) (result maddox.Run_Result) {
	defer func() { maddox.Run_Result_Invariants(result, "measure_command.result") }()
	// Built outside the timed bracket, so the clock's own setup never counts as wall.
	clock, _ := time_system.New_Operating_System_Clock()
	path, lookup_err := exec.LookPath(command.Path)
	if lookup_err != nil {
		result.Exit = SPAWN_FAILURE_EXIT
		result.Stderr = []byte("maddox: cannot find " + command.Path + "\n")
		return result
	}
	argv_words := command_argv(command)
	envp_words := append(os.Environ(), command.Environment...)
	argv := build_c_array(argv_words)
	envp := build_c_array(envp_words)
	defer free_c_array(argv, Word_Count(len(argv_words)))
	defer free_c_array(envp, Word_Count(len(envp_words)))
	c_path := C.CString(path)
	defer C.free(unsafe.Pointer(c_path))

	capture, create_err := os.CreateTemp("", "maddox-stderr-*")
	if create_err != nil {
		result.Exit = SPAWN_FAILURE_EXIT
		result.Stderr = []byte("maddox: cannot capture stderr\n")
		return result
	}
	defer os.Remove(capture.Name())
	defer capture.Close()

	before := clock.Now_Monotonic()
	counters := C.maddox_measure(c_path, argv, envp, C.int(capture.Fd()))
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
		Wall:             maddox.Metric(wall),
		RSS_Bytes_Max:    maddox.Resident_Bytes(counters.rss_bytes),
		CPU_Cycles:       maddox.Cycle_Count(counters.cycles),
		Instructions:     maddox.Instruction_Count(counters.instructions),
		Cache_References: maddox.Cache_Reference_Count(counters.cache_references),
		Cache_Misses:     maddox.Cache_Miss_Count(counters.cache_misses),
		Branch_Misses:    maddox.Branch_Miss_Count(counters.branch_misses),
		CPU_User:         maddox.User_Time(counters.user_ns),
		CPU_System:       maddox.System_Time(counters.system_ns),
	}
	result.Exit = maddox.Exit_Status(counters.exit_code)
	if result.Exit != 0 {
		result.Stderr = read_captured(capture)
	}
	return result
}

// Build_c_array copies words into a NULL-terminated C array of C strings for the
// exec. Free_c_array releases it.
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

// Acquire_machine_specs reads the host CPU, memory, and OS details from /proc and
// /sys on Linux. Fields that are absent or unreadable are left zero.
func acquire_machine_specs() (specs maddox.Machine_Specs) {
	defer func() { maddox.Machine_Specs_Invariants(specs, "acquire_machine_specs.specs") }()
	specs.CPU_Arch = maddox.Processor_Architecture(runtime.GOARCH)

	model, physical, logical := read_cpuinfo()
	specs.CPU_Model = model
	specs.Physical_Cores = physical
	specs.Logical_Cores = logical
	specs.CPU_Frequency_Hz_Max = read_cpu_frequency_max()
	specs.Cache_L1_Bytes = maddox.Level_1_Cache_Size(read_cache_size(CACHE_LEVEL_1))
	specs.Cache_L2_Bytes = maddox.Level_2_Cache_Size(read_cache_size(CACHE_LEVEL_2))
	specs.Cache_L3_Bytes = maddox.Level_3_Cache_Size(read_cache_size(CACHE_LEVEL_3))
	specs.RAM_Total_Bytes = read_memory_total()
	specs.Storage_Total_Bytes = maddox.Storage_Size(boot_volume_bytes())
	name, version := read_operating_system_release()
	specs.Operating_System_Name = name
	specs.Operating_System_Version = version

	var uname syscall.Utsname
	if syscall.Uname(&uname) == nil {
		specs.Kernel_Version = maddox.Kernel_Version(
			string(utsname_string(Utsname_Field(uname.Release[:]))) + " " +
				string(utsname_string(Utsname_Field(uname.Version[:]))))
	}
	return specs
}

// PROC_FILE_BYTES_MAX bounds how much of a /proc or /sys file is read into the
// fixed buffer, since the unbounded-read ban forbids os.ReadFile and these
// pseudo-files are small (a busy /proc/cpuinfo on a 256-thread box stays well under).
const PROC_FILE_BYTES_MAX = 1 << 20

// Boot_volume_bytes is the root filesystem's total capacity, taken from statfs. It
// is a close proxy for the physical drive capacity — enough to tell a 256GB drive
// from a 512GB one. A failed statfs reads zero.
func boot_volume_bytes() (total maddox.Byte_Size) {
	defer func() { maddox.Byte_Size_Invariants(total, "boot_volume_bytes.total") }()
	var stat syscall.Statfs_t
	if syscall.Statfs("/", &stat) != nil {
		return 0
	}
	return maddox.Byte_Size(uint64(stat.Bsize) * stat.Blocks)
}

// Utsname_Field is one fixed kernel-text field.
type Utsname_Field []int8

// Utsname_Field_Invariants bounds one fixed kernel-text field.
func Utsname_Field_Invariants(field Utsname_Field, namespace invariant.Namespace) {
	invariant.Tree(field, namespace).Range_Int(len(field), BOUND_MIN, BOUND_MAX).Ensure()
}

// Kernel_Part is one release or version field from Utsname.
type Kernel_Part string

// Kernel_Part_Invariants bounds one release or version field.
func Kernel_Part_Invariants(text Kernel_Part, namespace invariant.Namespace) {
	invariant.Tree(text, namespace).Range_Int(len(text), BOUND_MIN, BOUND_MAX).Ensure()
}

// Utsname_string stops at the fixed field's first zero byte.
func utsname_string(field Utsname_Field) (text Kernel_Part) {
	defer func() { Kernel_Part_Invariants(text, "utsname_string.text") }()
	Utsname_Field_Invariants(field, "utsname_string.field")
	builder := strings.Builder{}
	for _, character := range field {
		if character == 0 {
			break
		}
		builder.WriteByte(byte(character))
	}
	return Kernel_Part(builder.String())
}
