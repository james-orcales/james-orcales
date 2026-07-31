//go:build linux && cgo

package main

import (
	"os"
	"strconv"
	"strings"

	"local/james-orcales/maddox/internal"
	invariant "local/james-orcales/shared/invariant/default"
)

// PROC_CONTENT_MAX sits above the fixed read buffer.
const PROC_CONTENT_MAX = 1 << 21

// Proc_Path is one Linux host-data path.
type Proc_Path string

// Proc_Path_Invariants bounds one Linux host-data path.
func Proc_Path_Invariants(path Proc_Path, namespace invariant.Namespace) {
	invariant.Tree(path, namespace).Range_Int(len(path), BOUND_MIN, BOUND_MAX).Ensure()
}

// Proc_Content is bounded content from procfs, sysfs, or an operating-system file.
type Proc_Content string

// Proc_Content_Invariants bounds one Linux host-data value.
func Proc_Content_Invariants(content Proc_Content, namespace invariant.Namespace) {
	invariant.Tree(content, namespace).
		Range_Int(len(content), BOUND_MIN, PROC_CONTENT_MAX).
		Ensure()
}

// Read_proc_file uses a fixed buffer because Linux pseudo-files do not have stable sizes.
func read_proc_file(path Proc_Path) (content Proc_Content) {
	defer func() { Proc_Content_Invariants(content, "read_proc_file.content") }()
	Proc_Path_Invariants(path, "read_proc_file.path")
	file, open_err := os.Open(string(path))
	if open_err != nil {
		return ""
	}
	defer file.Close()
	buffer := make([]byte, PROC_FILE_BYTES_MAX)
	total := 0
	for total < len(buffer) {
		count, read_err := file.Read(buffer[total:])
		total += count
		if read_err != nil {
			break
		}
	}
	return Proc_Content(buffer[:total])
}

// Read_cpuinfo derives the core counts from identifiers because procfs has no summary.
func read_cpuinfo() (
	model maddox.Processor_Model,
	physical maddox.Physical_Core_Count,
	logical maddox.Logical_Core_Count,
) {
	defer func() {
		maddox.Processor_Model_Invariants(model, "read_cpuinfo.model")
		maddox.Physical_Core_Count_Invariants(physical, "read_cpuinfo.physical")
		maddox.Logical_Core_Count_Invariants(logical, "read_cpuinfo.logical")
	}()
	core_identifiers := map[string]struct{}{}
	for _, line := range strings.Split(string(read_proc_file("/proc/cpuinfo")), "\n") {
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch key {
		case "model name":
			if model == "" {
				model = maddox.Processor_Model(value)
			}
		case "processor":
			logical++
		case "core id":
			core_identifiers[value] = struct{}{}
		}
	}
	physical = maddox.Physical_Core_Count(len(core_identifiers))
	if physical == 0 {
		physical = maddox.Physical_Core_Count(logical)
	}
	return model, physical, logical
}

// Read_cpu_frequency_max converts the sysfs kilohertz value to hertz.
func read_cpu_frequency_max() (frequency maddox.Processor_Frequency) {
	defer func() {
		maddox.Processor_Frequency_Invariants(frequency, "read_cpu_frequency_max.frequency")
	}()
	content := read_proc_file("/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq")
	kilohertz, parse_err := strconv.ParseUint(strings.TrimSpace(string(content)), 10, 64)
	if parse_err != nil {
		return 0
	}
	return maddox.Processor_Frequency(kilohertz * 1000)
}

// CACHE_LEVEL_1 is the first cache index.
const CACHE_LEVEL_1 Cache_Level = 1

// CACHE_LEVEL_2 is the second cache index.
const CACHE_LEVEL_2 Cache_Level = 2

// CACHE_LEVEL_3 is the third cache index.
const CACHE_LEVEL_3 Cache_Level = 3

// Cache_Level is one CPU cache level that Maddox reports.
type Cache_Level int

// Cache_Level_Invariants limits cache probes to the reported levels.
func Cache_Level_Invariants(level Cache_Level, namespace invariant.Namespace) {
	invariant.Tree(level, namespace).
		Enum_3_Int(int(level), int(CACHE_LEVEL_1), int(CACHE_LEVEL_2), int(CACHE_LEVEL_3)).
		Ensure()
}

// Read_cache_size converts one sysfs cache-size value to bytes.
func read_cache_size(level Cache_Level) (size maddox.Byte_Size) {
	defer func() { maddox.Byte_Size_Invariants(size, "read_cache_size.size") }()
	Cache_Level_Invariants(level, "read_cache_size.level")
	path := "/sys/devices/system/cpu/cpu0/cache/index" +
		strconv.Itoa(int(level)-1) + "/size"
	text := strings.TrimSpace(string(read_proc_file(Proc_Path(path))))
	if len(text) == 0 {
		return 0
	}
	value, parse_err := strconv.ParseUint(text[:len(text)-1], 10, 64)
	if parse_err != nil {
		return 0
	}
	if text[len(text)-1] == 'K' {
		value *= 1024
	}
	if text[len(text)-1] == 'M' {
		value *= 1024 * 1024
	}
	return maddox.Byte_Size(value)
}

// Read_memory_total converts the procfs kibibyte count to bytes.
func read_memory_total() (total maddox.Memory_Size) {
	defer func() { maddox.Memory_Size_Invariants(total, "read_memory_total.total") }()
	for _, line := range strings.Split(string(read_proc_file("/proc/meminfo")), "\n") {
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0
		}
		value, parse_err := strconv.ParseUint(fields[1], 10, 64)
		if parse_err != nil {
			return 0
		}
		return maddox.Memory_Size(value * 1024)
	}
	return 0
}

// Read_operating_system_release keeps missing release data representable as empty text.
func read_operating_system_release() (
	name maddox.Operating_System_Name,
	version maddox.Operating_System_Version,
) {
	defer func() {
		maddox.Operating_System_Name_Invariants(name, "read_operating_system_release.name")
		maddox.Operating_System_Version_Invariants(
			version, "read_operating_system_release.version")
	}()
	content := read_proc_file("/etc/os-release")
	for _, line := range strings.Split(string(content), "\n") {
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		value = strings.Trim(value, `"`)
		if key == "NAME" {
			name = maddox.Operating_System_Name(value)
		}
		if key == "VERSION_ID" {
			version = maddox.Operating_System_Version(value)
		}
	}
	if name == "" {
		name = "Linux"
	}
	return name, version
}
