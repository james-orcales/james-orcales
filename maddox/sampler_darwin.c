#include "sampler_darwin.h"

#include <fcntl.h>
#include <libproc.h>
#include <signal.h>
#include <spawn.h>
#include <stdlib.h>
#include <string.h>
#include <sys/resource.h>
#include <sys/sysctl.h>
#include <sys/types.h>
#include <sys/wait.h>

// The stdlib has no 64-bit sysctl reader, and a 32-bit read truncates host sizes.
uint64_t maddox_sysctl_uint64(const char *name) {
	uint64_t value = 0;
	size_t size = sizeof(value);
	sysctlbyname(name, &value, &size, NULL, 0);
	return value;
}

// posix_spawnp is safe in a multithreaded Go process. The wait keeps the child as a
// zombie until proc_pid_rusage reads the counters, then waitpid reaps the child.
maddox_measurement maddox_measure(char **argv, char **envp, int stderr_fd) {
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
