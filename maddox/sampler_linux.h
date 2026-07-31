#ifndef MADDOX_SAMPLER_LINUX_H
#define MADDOX_SAMPLER_LINUX_H

// The Go bridge needs a plain value that cgo can copy without platform objects.
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

maddox_measurement maddox_measure(char *path, char **argv, char **envp, int stderr_fd);

#endif
