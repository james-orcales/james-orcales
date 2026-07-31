#include "sampler_linux.h"

#include <errno.h>
#include <fcntl.h>
#include <linux/perf_event.h>
#include <stdlib.h>
#include <string.h>
#include <sys/ioctl.h>
#include <sys/resource.h>
#include <sys/syscall.h>
#include <sys/wait.h>
#include <unistd.h>

// The C bridge sets perf_event_attr bitfields because cgo cannot address them.
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

// An unavailable performance counter is a sparse metric, not a failed sample.
static unsigned long long maddox_perf_read(int fd) {
	unsigned long long value = 0;
	if (fd < 0) return 0;
	if (read(fd, &value, sizeof(value)) != (ssize_t)sizeof(value)) return 0;
	return value;
}

// One C call keeps perf setup and fork on one OS thread. The child uses only
// async-signal-safe functions before execve, which is necessary in the Go process.
maddox_measurement maddox_measure(char *path, char **argv, char **envp, int stderr_fd) {
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
	// Linux reports ru_maxrss in kibibytes, while Maddox reports bytes.
	out.rss_bytes = (unsigned long long)usage.ru_maxrss * 1024ULL;
	return out;
}
