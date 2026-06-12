//go:build linux

package main

/*
#include <linux/perf_event.h>
#include <sys/ioctl.h>
#include <sys/wait.h>
#include <sys/resource.h>
#include <sys/syscall.h>
#include <unistd.h>
#include <fcntl.h>
#include <errno.h>
#include <stdlib.h>
#include <string.h>

// maddox_measurement is what one spawned-and-measured run reports back: whether the
// spawn failed, the child's exit code, the five perf hardware counters, and the
// getrusage fields. Only plain integer fields, so cgo maps it without trouble.
typedef struct {
	int spawn_errno;
	int exit_code;
	unsigned long long cycles;
	unsigned long long instructions;
	unsigned long long cache_references;
	unsigned long long cache_misses;
	unsigned long long branch_misses;
	unsigned long long user_ns;
	unsigned long long system_ns;
	unsigned long long rss_bytes;
} maddox_measurement;

// maddox_perf_open opens one user-space hardware counter, grouped under group_fd
// (-1 makes it the group leader). disabled + enable_on_exec leave it off until the
// child execs; inherit makes the child's events count toward this fd; exclude_kernel
// and exclude_hv keep it to user space, matching poop. The perf_event_attr bitfields
// are set here in C, since cgo cannot set C bitfields from Go.
static int maddox_perf_open(unsigned long config, int group_fd) {
	struct perf_event_attr attr;
	memset(&attr, 0, sizeof(attr));
	attr.type = PERF_TYPE_HARDWARE;
	attr.size = sizeof(attr);
	attr.config = config;
	attr.disabled = 1;
	attr.exclude_kernel = 1;
	attr.exclude_hv = 1;
	attr.inherit = 1;
	attr.enable_on_exec = 1;
	return (int)syscall(__NR_perf_event_open, &attr, 0, -1, group_fd, PERF_FLAG_FD_CLOEXEC);
}

// maddox_perf_read reads one counter's accumulated value; a closed or failed fd
// reads as zero.
static unsigned long long maddox_perf_read(int fd) {
	unsigned long long value = 0;
	if (fd < 0) return 0;
	if (read(fd, &value, sizeof(value)) != (ssize_t)sizeof(value)) return 0;
	return value;
}

// maddox_measure spawns path via a real fork+exec so the child inherits the perf
// counters (inherit + enable_on_exec attribute its user-space events). Between fork
// and exec the child runs only async-signal-safe calls — dup2 and execve of an
// absolute path, the Go side having resolved PATH — so it is safe even though the Go
// runtime is multithreaded. The whole function runs in one cgo call on one OS thread,
// so the perf open and the fork share a thread and the child is a descendant of the
// monitored task. If the PMU is unavailable (perf_event_paranoid / no CAP_PERFMON),
// the counters stay zero and the run still reports wall, cpu, and rss.
static maddox_measurement maddox_measure(char *path, char **argv, char **envp, int stderr_fd) {
	maddox_measurement out;
	memset(&out, 0, sizeof(out));

	int devnull = open("/dev/null", O_RDWR | O_CLOEXEC);

	int cycles_fd = maddox_perf_open(PERF_COUNT_HW_CPU_CYCLES, -1);
	int instructions_fd = -1;
	int cache_references_fd = -1;
	int cache_misses_fd = -1;
	int branch_misses_fd = -1;
	if (cycles_fd != -1) {
		instructions_fd = maddox_perf_open(PERF_COUNT_HW_INSTRUCTIONS, cycles_fd);
		cache_references_fd = maddox_perf_open(PERF_COUNT_HW_CACHE_REFERENCES, cycles_fd);
		cache_misses_fd = maddox_perf_open(PERF_COUNT_HW_CACHE_MISSES, cycles_fd);
		branch_misses_fd = maddox_perf_open(PERF_COUNT_HW_BRANCH_MISSES, cycles_fd);
		ioctl(cycles_fd, PERF_EVENT_IOC_RESET, PERF_IOC_FLAG_GROUP);
		ioctl(cycles_fd, PERF_EVENT_IOC_DISABLE, PERF_IOC_FLAG_GROUP);
	}

	pid_t pid = fork();
	if (pid == -1) {
		out.spawn_errno = errno;
		return out;
	}
	if (pid == 0) {
		if (devnull != -1) {
			dup2(devnull, 0);
			dup2(devnull, 1);
		}
		dup2(stderr_fd, 2);
		execve(path, argv, envp);
		_exit(127);
	}

	int status = 0;
	struct rusage usage;
	memset(&usage, 0, sizeof(usage));
	wait4(pid, &status, 0, &usage);

	if (cycles_fd != -1) {
		ioctl(cycles_fd, PERF_EVENT_IOC_DISABLE, PERF_IOC_FLAG_GROUP);
		out.cycles = maddox_perf_read(cycles_fd);
		out.instructions = maddox_perf_read(instructions_fd);
		out.cache_references = maddox_perf_read(cache_references_fd);
		out.cache_misses = maddox_perf_read(cache_misses_fd);
		out.branch_misses = maddox_perf_read(branch_misses_fd);
		close(cycles_fd);
		close(instructions_fd);
		close(cache_references_fd);
		close(cache_misses_fd);
		close(branch_misses_fd);
	}
	if (devnull != -1) close(devnull);

	if (WIFEXITED(status)) {
		out.exit_code = WEXITSTATUS(status);
	} else {
		out.exit_code = -1;
	}
	out.user_ns = (unsigned long long)usage.ru_utime.tv_sec * 1000000000ULL +
		(unsigned long long)usage.ru_utime.tv_usec * 1000ULL;
	out.system_ns = (unsigned long long)usage.ru_stime.tv_sec * 1000000000ULL +
		(unsigned long long)usage.ru_stime.tv_usec * 1000ULL;
	// Linux getrusage reports ru_maxrss in kibibytes; normalize to bytes.
	out.rss_bytes = (unsigned long long)usage.ru_maxrss * 1024ULL;
	return out;
}
*/
import "C"

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/james-orcales/james-orcales/maddox/internal"
	"github.com/james-orcales/james-orcales/shared/sh"
	"github.com/james-orcales/james-orcales/shared/time"
)

// System_sampler returns the production Sampler, whose Measure spawns each command
// under perf_event_open and reads its hardware counters.
func system_sampler() (sampler maddox.Sampler) {
	sampler.Measure = measure_command
	return sampler
}

// Measure_command resolves the executable, spawns it under the perf counters, and
// reports the Run_Result. PATH is resolved here so the child can execve an absolute
// path with no malloc between fork and exec. Wall time is left zero — Main times each
// run with the injected clock.
func measure_command(command sh.Command) (result maddox.Run_Result) {
	path, lookup_err := exec.LookPath(command.Path)
	if lookup_err != nil {
		result.Exit = spawn_failure_exit
		result.Stderr = []byte("maddox: cannot find " + command.Path + "\n")
		return result
	}
	argv_words := command_argv(command)
	envp_words := append(os.Environ(), command.Environment...)
	argv := build_c_array(argv_words)
	envp := build_c_array(envp_words)
	defer free_c_array(argv, len(argv_words))
	defer free_c_array(envp, len(envp_words))
	c_path := C.CString(path)
	defer C.free(unsafe.Pointer(c_path))

	capture, create_err := os.CreateTemp("", "maddox-stderr-*")
	if create_err != nil {
		result.Exit = spawn_failure_exit
		result.Stderr = []byte("maddox: cannot capture stderr\n")
		return result
	}
	defer os.Remove(capture.Name())
	defer capture.Close()

	counters := C.maddox_measure(c_path, argv, envp, C.int(capture.Fd()))
	if counters.spawn_errno != 0 {
		result.Exit = spawn_failure_exit
		result.Stderr = []byte("maddox: cannot spawn " + command.Path + "\n")
		return result
	}

	result.Sample = maddox.Sample{
		RSS_Bytes_Max:    int64(counters.rss_bytes),
		CPU_Cycles:       uint64(counters.cycles),
		Instructions:     uint64(counters.instructions),
		Cache_References: uint64(counters.cache_references),
		Cache_Misses:     uint64(counters.cache_misses),
		Branch_Misses:    uint64(counters.branch_misses),
		CPU_User:         time.Duration(counters.user_ns),
		CPU_System:       time.Duration(counters.system_ns),
	}
	result.Exit = int(counters.exit_code)
	if result.Exit != 0 {
		result.Stderr = read_captured(capture)
	}
	return result
}

// Build_c_array copies words into a NULL-terminated C array of C strings for the
// exec. Free_c_array releases it.
func build_c_array(words []string) (array **C.char) {
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
func free_c_array(array **C.char, word_count int) {
	view := unsafe.Slice(array, word_count+1)
	for index := 0; index < word_count; index++ {
		C.free(unsafe.Pointer(view[index]))
	}
	C.free(unsafe.Pointer(array))
}

// Acquire_machine_specs reads the host CPU, memory, and OS details from /proc and
// /sys on Linux. Fields that are absent or unreadable are left zero.
func acquire_machine_specs() (specs maddox.Machine_Specs) {
	specs.CPU_Arch = runtime.GOARCH

	specs.CPU_Model, specs.Physical_Cores, specs.Logical_Cores = read_cpuinfo()
	specs.CPU_Frequency_Hz_Max = read_cpu_frequency_max()
	specs.Cache_L1_Bytes = read_cache_size(1)
	specs.Cache_L2_Bytes = read_cache_size(2)
	specs.Cache_L3_Bytes = read_cache_size(3)
	specs.RAM_Total_Bytes = read_memory_total()
	specs.Storage_Total_Bytes = boot_volume_bytes()
	specs.Operating_System_Name, specs.Operating_System_Version =
		read_operating_system_release()

	var uname syscall.Utsname
	if syscall.Uname(&uname) == nil {
		specs.Kernel_Version = utsname_string(uname.Release[:]) +
			" " + utsname_string(uname.Version[:])
	}
	return specs
}

// Proc_file_bytes_max bounds how much of a /proc or /sys file is read into the
// fixed buffer, since the unbounded-read ban forbids os.ReadFile and these
// pseudo-files are small (a busy /proc/cpuinfo on a 256-thread box stays well under).
const proc_file_bytes_max = 1 << 20

// Read_proc_file reads up to proc_file_bytes_max bytes of a pseudo-file into a fixed
// buffer, the bounded-read pattern read_captured uses. A missing or unreadable file
// reads empty, so every caller treats absence as "field unknown".
func read_proc_file(path string) (content string) {
	file, open_err := os.Open(path)
	if open_err != nil {
		return ""
	}
	defer file.Close()
	buffer := make([]byte, proc_file_bytes_max)
	total := 0
	for total < len(buffer) {
		n, read_err := file.Read(buffer[total:])
		total += n
		if read_err != nil {
			break
		}
	}
	return string(buffer[:total])
}

// Boot_volume_bytes is the root filesystem's total capacity, taken from statfs. It
// is a close proxy for the physical drive capacity — enough to tell a 256GB drive
// from a 512GB one. A failed statfs reads zero.
func boot_volume_bytes() (total uint64) {
	var stat syscall.Statfs_t
	if syscall.Statfs("/", &stat) != nil {
		return 0
	}
	return uint64(stat.Bsize) * stat.Blocks
}

// Read_cpuinfo parses /proc/cpuinfo for the CPU model string, the physical-core
// count (unique "core id" values), and the logical-core count (total "processor"
// entries).
func read_cpuinfo() (model string, physical int, logical int) {
	core_ids := map[string]struct{}{}
	for _, line := range strings.Split(read_proc_file("/proc/cpuinfo"), "\n") {
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch key {
		case "model name":
			if model == "" {
				model = value
			}
		case "processor":
			logical++
		case "core id":
			core_ids[value] = struct{}{}
		}
	}
	// Some ARM kernels omit "core id"; fall back to the logical count there.
	if len(core_ids) > 0 {
		physical = len(core_ids)
	} else {
		physical = logical
	}
	return model, physical, logical
}

// Read_cpu_frequency_max reads the maximum CPU frequency from the cpufreq driver
// for CPU 0; most Linux CPUs expose this even without the governor active.
func read_cpu_frequency_max() (hz uint64) {
	text := strings.TrimSpace(
		read_proc_file("/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq"))
	// The cpufreq driver reports the frequency in kHz.
	khz, parse_err := strconv.ParseUint(text, 10, 64)
	if parse_err != nil {
		return 0
	}
	return khz * 1000
}

// Read_cache_size reads the size of CPU 0's cache at the given level from the sysfs
// cache directory. The size string carries a unit suffix (K, M).
func read_cache_size(level int) (size uint64) {
	path := "/sys/devices/system/cpu/cpu0/cache/index" +
		strconv.Itoa(level-1) + "/size"
	text := strings.TrimSpace(read_proc_file(path))
	if len(text) == 0 {
		return 0
	}
	suffix := text[len(text)-1]
	digits := text[:len(text)-1]
	value, parse_err := strconv.ParseUint(digits, 10, 64)
	if parse_err != nil {
		return 0
	}
	switch suffix {
	case 'K':
		return value * 1024
	case 'M':
		return value * 1024 * 1024
	}
	return value
}

// Read_memory_total parses MemTotal from /proc/meminfo, returning bytes.
func read_memory_total() (total uint64) {
	for _, line := range strings.Split(read_proc_file("/proc/meminfo"), "\n") {
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		// A MemTotal line reads "MemTotal:    16384000 kB", in kibibytes.
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0
		}
		kb, parse_err := strconv.ParseUint(fields[1], 10, 64)
		if parse_err != nil {
			return 0
		}
		return kb * 1024
	}
	return 0
}

// Read_operating_system_release parses /etc/os-release for the OS name and version.
func read_operating_system_release() (name string, version string) {
	content := read_proc_file("/etc/os-release")
	if content == "" {
		return "Linux", ""
	}
	for _, line := range strings.Split(content, "\n") {
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		value = strings.Trim(value, `"`)
		switch key {
		case "NAME":
			name = value
		case "VERSION_ID":
			version = value
		}
	}
	if name == "" {
		name = "Linux"
	}
	return name, version
}

// Utsname_string converts a fixed-size Utsname field to a Go string, stopping at
// the first zero byte. The element type differs by platform (int8 vs uint8).
func utsname_string[T int8 | uint8](field []T) (text string) {
	builder := strings.Builder{}
	for _, b := range field {
		if b == 0 {
			break
		}
		builder.WriteByte(byte(b))
	}
	return builder.String()
}
