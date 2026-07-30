//go:build darwin && arm64

package main

/*
#include <spawn.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <signal.h>
#include <libproc.h>
#include <sys/resource.h>
#include <fcntl.h>
#include <stdlib.h>
#include <string.h>
#include <sys/sysctl.h>
#include <stdint.h>

// maddox_sysctl_uint64 reads a uint64 sysctl by name via sysctlbyname, the only
// way to read the 64-bit hw.* keys (frequency, cache, memory) the Go stdlib's
// SysctlUint32 cannot reach. Returns 0 on any error, so an absent key reads zero.
static uint64_t maddox_sysctl_uint64(const char *name) {
	uint64_t value = 0;
	size_t size = sizeof(value);
	sysctlbyname(name, &value, &size, NULL, 0);
	return value;
}

// maddox_measurement is what one spawned-and-measured run reports back: whether
// the spawn itself failed, the child's exit code, and the proc_pid_rusage counters
// the kernel populates from the Apple-Silicon PMU.
typedef struct {
	int spawn_errno;
	int exit_code;
	unsigned long long cycles;
	unsigned long long instructions;
	unsigned long long user_ns;
	unsigned long long system_ns;
	unsigned long long peak_footprint;
} maddox_measurement;

// maddox_measure runs argv to completion and reports its counters. posix_spawnp is
// used over fork because forking a multithreaded Go runtime is unsafe; waitid with
// WNOWAIT detects the exit WITHOUT reaping, so proc_pid_rusage can still read the
// (now zombie) process before waitpid finally reaps it — the macOS analogue of
// reading Linux perf fds after the child dies. Child stdout is discarded; child
// stderr is redirected to stderr_fd so a failing run's diagnostics can be shown.
static maddox_measurement maddox_measure(char **argv, char **envp, int stderr_fd) {
	maddox_measurement out;
	memset(&out, 0, sizeof(out));

	posix_spawn_file_actions_t actions;
	posix_spawn_file_actions_init(&actions);
	posix_spawn_file_actions_addopen(&actions, 1, "/dev/null", O_WRONLY, 0);
	posix_spawn_file_actions_adddup2(&actions, stderr_fd, 2);

	pid_t pid;
	int rc = posix_spawnp(&pid, argv[0], &actions, NULL, argv, envp);
	posix_spawn_file_actions_destroy(&actions);
	if (rc != 0) {
		out.spawn_errno = rc;
		return out;
	}

	siginfo_t info;
	memset(&info, 0, sizeof(info));
	waitid(P_PID, pid, &info, WEXITED | WNOWAIT);

	struct rusage_info_v4 ri;
	memset(&ri, 0, sizeof(ri));
	if (proc_pid_rusage(pid, RUSAGE_INFO_V4, (rusage_info_t *)&ri) == 0) {
		out.cycles = ri.ri_cycles;
		out.instructions = ri.ri_instructions;
		out.user_ns = ri.ri_user_time;
		out.system_ns = ri.ri_system_time;
		out.peak_footprint = ri.ri_lifetime_max_phys_footprint;
	}

	int status = 0;
	waitpid(pid, &status, 0);
	if (WIFEXITED(status)) {
		out.exit_code = WEXITSTATUS(status);
	} else {
		out.exit_code = -1;
	}
	return out;
}
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
	time_default "local/james-orcales/shared/time/default"
)

// System_sampler returns the production Sampler, whose Measure spawns each command
// and reads the Apple-Silicon hardware counters through proc_pid_rusage.
func system_sampler() (sampler maddox.Sampler) {
	defer func() { maddox.Sampler_Invariants("system_sampler.sampler", sampler) }()
	sampler.Measure = measure_command
	return sampler
}

// Measure_command spawns the command, reads its hardware counters, and reports the
// Run_Result, timing the child's wall around the spawn with a monotonic clock. Main reads
// no clock of its own — only func main's driver advances one — so wall is measured here.
func measure_command(command io.Process_Request) (result maddox.Run_Result) {
	defer func() { maddox.Run_Result_Invariants("measure_command.result", result) }()
	// Built outside the timed bracket, so the clock's own setup never counts as wall.
	clock, _ := time_default.New_Operating_System_Clock()
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
	result.Completed_At = after
	if counters.spawn_errno != 0 {
		result.Exit = SPAWN_FAILURE_EXIT
		result.Stderr = []byte("maddox: cannot spawn " + command.Path + "\n")
		return result
	}

	result.Sample = maddox.Sample{
		Wall:          maddox.Wall_Time(wall),
		RSS_Bytes_Max: maddox.RSS_Bytes_Max(counters.peak_footprint),
		CPU_Cycles:    maddox.CPU_Cycles(counters.cycles),
		Instructions:  maddox.Instructions(counters.instructions),
		CPU_User:      maddox.CPU_User_Time(counters.user_ns),
		CPU_System:    maddox.CPU_System_Time(counters.system_ns),
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
	Argv_Invariants("build_c_array.words", words)
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
	Word_Count_Invariants("free_c_array.word_count", word_count)
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
	defer func() { maddox.Machine_Specs_Invariants("acquire_machine_specs.specs", specs) }()
	model, _ := syscall.Sysctl("machdep.cpu.brand_string")
	specs.CPU_Model = maddox.CPU_Model(model)
	specs.CPU_Arch = maddox.CPU_Arch(runtime.GOARCH)

	physical, _ := syscall.SysctlUint32("hw.physicalcpu")
	specs.Physical_Cores = maddox.Physical_Cores(physical)
	logical, _ := syscall.SysctlUint32("hw.logicalcpu")
	specs.Logical_Cores = maddox.Logical_Cores(logical)

	// Apple Silicon exposes P-cores at perflevel0 and E-cores at perflevel1.
	p_cores, p_err := syscall.SysctlUint32("hw.perflevel0.physicalcpu")
	e_cores, e_err := syscall.SysctlUint32("hw.perflevel1.physicalcpu")
	if p_err == nil {
		if e_err == nil {
			specs.Performance_Cores = maddox.Performance_Cores(p_cores)
			specs.Efficiency_Cores = maddox.Efficiency_Cores(e_cores)
		}
	}

	// Frequency and caches are 64-bit sysctls, read through cgo since SysctlUint32
	// truncates and the stdlib offers no raw form.
	specs.CPU_Frequency_Hz_Max = maddox.Hertz(sysctl_uint64("hw.perflevel0.cpufrequency_max"))
	specs.Cache_L1_Bytes = maddox.Cache_L1_Bytes(sysctl_uint64("hw.perflevel0.l1dcachesize"))
	specs.Cache_L2_Bytes = maddox.Cache_L2_Bytes(sysctl_uint64("hw.perflevel0.l2cachesize"))
	// L3 is absent on Apple Silicon; an absent key reads zero and is omitted.
	specs.Cache_L3_Bytes = maddox.Cache_L3_Bytes(sysctl_uint64("hw.l3cachesize"))
	specs.RAM_Total_Bytes = maddox.RAM_Total_Bytes(sysctl_uint64("hw.memsize"))
	specs.Storage_Total_Bytes = boot_volume_bytes()

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
func Sysctl_Key_Invariants(identifier string, name Sysctl_Key) {
	invariant.Range(identifier, len(name), BOUND_MIN, BOUND_MAX)
}

// SYSCTL_VALUE_MAX mirrors the library's byte-size ceiling: every sysctl this sampler
// reads feeds a machine-spec field whose own guard caps at or below it, so a reading
// past this bound would have died downstream anyway — the guard only moves that
// refusal to the read.
const SYSCTL_VALUE_MAX = 1 << 53

// Sysctl_Value is one 64-bit sysctl reading; an absent key reads zero.
type Sysctl_Value uint64

// Sysctl_Value_Invariants bounds the reading to the machine-spec ceiling its
// consumers enforce.
func Sysctl_Value_Invariants(identifier string, value Sysctl_Value) {
	invariant.Range(identifier, uint64(value), 0, SYSCTL_VALUE_MAX)
}

// Sysctl_uint64 reads a 64-bit sysctl by name, marshaling the Go string across the
// cgo boundary and freeing the C copy after the call.
func sysctl_uint64(name Sysctl_Key) (value Sysctl_Value) {
	defer func() { Sysctl_Value_Invariants("sysctl_uint64.value", value) }()
	Sysctl_Key_Invariants("sysctl_uint64.name", name)
	cname := C.CString(string(name))
	defer C.free(unsafe.Pointer(cname))
	return Sysctl_Value(C.maddox_sysctl_uint64(cname))
}

// Boot_volume_bytes is the boot filesystem's total capacity, taken from statfs on
// the root. On APFS this reports the shared container size, a close proxy for the
// physical SSD capacity — enough to tell a 256GB drive from a 512GB one. A failed
// statfs reads zero.
func boot_volume_bytes() (total maddox.Storage_Total_Bytes) {
	defer func() {
		maddox.Storage_Total_Bytes_Invariants("boot_volume_bytes.total", total)
	}()
	var stat syscall.Statfs_t
	if syscall.Statfs("/", &stat) != nil {
		return 0
	}
	return maddox.Storage_Total_Bytes(uint64(stat.Bsize) * stat.Blocks)
}
