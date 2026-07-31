#ifndef MADDOX_SAMPLER_DARWIN_H
#define MADDOX_SAMPLER_DARWIN_H

#include <stdint.h>

// The Go bridge needs a plain value that cgo can copy without platform objects.
typedef struct {
	int spawn_errno;
	int exit_code;
	unsigned long long cycles;
	unsigned long long instructions;
	unsigned long long user_ns;
	unsigned long long system_ns;
	unsigned long long peak_footprint;
} maddox_measurement;

uint64_t maddox_sysctl_uint64(const char *name);
maddox_measurement maddox_measure(char **argv, char **envp, int stderr_fd);

#endif
