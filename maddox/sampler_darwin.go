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
	"unsafe"

	"github.com/james-orcales/james-orcales/maddox/internal"
	"github.com/james-orcales/james-orcales/shared/sh"
	"github.com/james-orcales/james-orcales/shared/time"
)

// System_sampler returns the production Sampler, whose Measure spawns each command
// and reads the Apple-Silicon hardware counters through proc_pid_rusage.
func system_sampler() (sampler maddox.Sampler) {
	sampler.Measure = measure_command
	return sampler
}

// Measure_command spawns the command, reads its hardware counters, and reports the
// Run_Result. Wall time is left zero — Main times each run with the injected clock.
func measure_command(command sh.Command) (result maddox.Run_Result) {
	argv_words := command_argv(command)
	envp_words := append(os.Environ(), command.Environment...)
	argv := build_c_array(argv_words)
	envp := build_c_array(envp_words)
	defer free_c_array(argv, len(argv_words))
	defer free_c_array(envp, len(envp_words))

	capture, create_err := os.CreateTemp("", "maddox-stderr-*")
	if create_err != nil {
		result.Exit = spawn_failure_exit
		result.Stderr = []byte("maddox: cannot capture stderr\n")
		return result
	}
	defer os.Remove(capture.Name())
	defer capture.Close()

	counters := C.maddox_measure(argv, envp, C.int(capture.Fd()))
	if counters.spawn_errno != 0 {
		result.Exit = spawn_failure_exit
		result.Stderr = []byte("maddox: cannot spawn " + command.Path + "\n")
		return result
	}

	result.Sample = maddox.Sample{
		RSS_Bytes_Max: int64(counters.peak_footprint),
		CPU_Cycles:    uint64(counters.cycles),
		Instructions:  uint64(counters.instructions),
		CPU_User:      time.Duration(counters.user_ns),
		CPU_System:    time.Duration(counters.system_ns),
	}
	result.Exit = int(counters.exit_code)
	if result.Exit != 0 {
		result.Stderr = read_captured(capture)
	}
	return result
}

// Build_c_array copies words into a NULL-terminated C array of C strings for
// posix_spawnp. Free_c_array releases it.
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
