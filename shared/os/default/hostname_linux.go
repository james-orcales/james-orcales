//go:build linux

package os

import (
	"runtime"
	"syscall"
	"unsafe"

	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/strconv"
	"local/james-orcales/shared/strings"
)

// Linux exposes execve through its native syscall table.
func system_execve(workspace_pointer unsafe.Pointer) (err error) {
	workspace := (*Self_Exec_Workspace)(workspace_pointer)
	_, _, operation_errno := syscall.RawSyscall(
		syscall.SYS_EXECVE,
		uintptr(unsafe.Pointer(&(*workspace.Path)[0])),
		uintptr(unsafe.Pointer(&(*workspace.Argument_Pointers)[0])),
		uintptr(unsafe.Pointer(&(*workspace.Environment_Pointers)[0])),
	)
	if operation_errno != 0 {
		return operation_errno
	}
	return nil
}

// MACHINE_FILE_BYTES_MAXIMUM bounds one procfs or sysfs read. These pseudo-files report size
// zero, thus File_Read_Into cannot bound them, and a busy /proc/cpuinfo stays well under this.
const MACHINE_FILE_BYTES_MAXIMUM = bits.MEBIBYTE_BYTES

// MACHINE_CORE_IDENTIFIER_COUNT_MAXIMUM bounds the distinct core ids kept while counting physical
// cores. A host past it saturates the count rather than grow the set.
const MACHINE_CORE_IDENTIFIER_COUNT_MAXIMUM = PROCESSOR_CORE_COUNT_MAXIMUM

// MACHINE_HERTZ_PER_KILOHERTZ scales the sysfs kilohertz frequency.
const MACHINE_HERTZ_PER_KILOHERTZ = 1000

// MACHINE_FILE_PATH_BYTES_MINIMUM is the shortest path read here.
const MACHINE_FILE_PATH_BYTES_MINIMUM = len("/proc/meminfo")

// Machine_File_Path names one procfs, sysfs, or release file.
type Machine_File_Path string

// Machine_File_Path_Invariants bounds the path text.
func Machine_File_Path_Invariants(value Machine_File_Path, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), MACHINE_FILE_PATH_BYTES_MINIMUM, strings.TEXT_SIZE_MAXIMUM).
		Ensure()
}

// Machine_File_Text is one whole pseudo-file.
type Machine_File_Text string

// Machine_File_Text_Invariants bounds the file to its read storage.
func Machine_File_Text_Invariants(value Machine_File_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), MACHINE_VALUE_ABSENT, MACHINE_FILE_BYTES_MAXIMUM).
		Ensure()
}

// Machine_Number is one decimal figure a pseudo-file states.
type Machine_Number uint64

// Machine_Number_Invariants admits every figure a size, count, or rate reaches.
func Machine_Number_Invariants(value Machine_Number, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), MACHINE_VALUE_ABSENT, STORAGE_BYTES_MAXIMUM).
		Ensure()
}

// Machine_Read describes the host through procfs, sysfs, /etc/os-release, statfs, and uname.
func Machine_Read() (machine Machine) {
	defer func() { Machine_Invariants(machine, "Machine_Read.machine") }()
	machine.Processor_Architecture = Processor_Architecture(runtime.GOARCH)
	machine.Processor_Model, machine.Physical_Core_Count, machine.Logical_Core_Count =
		machine_processors(machine_file("/proc/cpuinfo"))
	kilohertz := machine_number(
		machine_file("/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq"))
	machine.Processor_Frequency = Processor_Frequency(MACHINE_HERTZ_PER_KILOHERTZ * kilohertz)
	machine.Level_1_Cache_Size = Level_1_Cache_Size(
		machine_cache("/sys/devices/system/cpu/cpu0/cache/index0/size"))
	machine.Level_2_Cache_Size = Level_2_Cache_Size(
		machine_cache("/sys/devices/system/cpu/cpu0/cache/index1/size"))
	machine.Level_3_Cache_Size = Level_3_Cache_Size(
		machine_cache("/sys/devices/system/cpu/cpu0/cache/index2/size"))
	machine.Physical_Memory_Size = machine_memory(machine_file("/proc/meminfo"))
	machine.Storage_Size = machine_storage()
	machine.Operating_System_Name, machine.Operating_System_Version = machine_release(
		machine_file("/etc/os-release"))
	names := syscall.Utsname{}
	if syscall.Uname(&names) == nil {
		machine.Kernel_Version = Kernel_Version(machine_utsname(names.Release[:]) + " " +
			machine_utsname(names.Version[:]))
	}
	return machine
}

// Read one pseudo-file until end of file. An unreadable file is empty text.
func machine_file(path Machine_File_Path) (content Machine_File_Text) {
	defer func() { Machine_File_Text_Invariants(content, "machine_file.content") }()
	Machine_File_Path_Invariants(path, "machine_file.path")
	file, open_err := syscall.Open(string(path), syscall.O_RDONLY|syscall.O_CLOEXEC, 0)
	if open_err != nil {
		return ""
	}
	defer syscall.Close(file)
	storage := make([]byte, MACHINE_FILE_BYTES_MAXIMUM)
	filled := 0
	// Each pass advances by at least one byte or ends, thus storage bounds the loop.
	for pass_index := 0; pass_index < len(storage); pass_index++ {
		if filled >= len(storage) {
			break
		}
		count, read_err := syscall.Read(file, storage[filled:])
		if read_err != nil {
			break
		}
		if count == 0 {
			break
		}
		filled += count
	}
	return Machine_File_Text(storage[:filled])
}

// Count logical cores and distinct core ids from /proc/cpuinfo, which has no summary. A host whose
// cpuinfo names no core id counts each logical core as physical.
func machine_processors(cpuinfo Machine_File_Text) (
	model Processor_Model, physical Physical_Core_Count, logical Logical_Core_Count,
) {
	defer func() {
		Processor_Model_Invariants(model, "machine_processors.model")
		Physical_Core_Count_Invariants(physical, "machine_processors.physical")
		Logical_Core_Count_Invariants(logical, "machine_processors.logical")
	}()
	Machine_File_Text_Invariants(cpuinfo, "machine_processors.cpuinfo")
	identifiers := make(Core_Identifiers, 0, MACHINE_CORE_IDENTIFIER_COUNT_MAXIMUM)
	start_index := 0
	for end_index := 0; end_index <= len(cpuinfo); end_index++ {
		if end_index < len(cpuinfo) {
			if cpuinfo[end_index] != '\n' {
				continue
			}
		}
		key, value := machine_field(cpuinfo[start_index:end_index], ":")
		start_index = end_index + 1
		switch key {
		case "model name":
			if model == "" {
				model = Processor_Model(value)
			}
		case "processor":
			logical++
		case "core id":
			if !machine_identifier_known(identifiers, value) {
				if len(identifiers) < cap(identifiers) {
					identifiers = append(identifiers, value)
				}
			}
		}
	}
	physical = Physical_Core_Count(len(identifiers))
	if physical == 0 {
		physical = Physical_Core_Count(logical)
	}
	return model, physical, logical
}

// Core_Identifiers are the distinct core ids counted so far.
type Core_Identifiers []strings.Text

// Core_Identifiers_Invariants bounds the set to its storage.
func Core_Identifiers_Invariants(value Core_Identifiers, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), MACHINE_VALUE_ABSENT, MACHINE_CORE_IDENTIFIER_COUNT_MAXIMUM).
		Ensure()
}

// Report whether identifier is already counted.
func machine_identifier_known(
	identifiers Core_Identifiers, identifier strings.Text,
) (known strings.Boolean) {
	defer func() { strings.Boolean_Invariants(known, "machine_identifier_known.known") }()
	Core_Identifiers_Invariants(identifiers, "machine_identifier_known.identifiers")
	strings.Text_Invariants(identifier, "machine_identifier_known.identifier")
	for _, candidate := range identifiers {
		if candidate == identifier {
			return true
		}
	}
	return false
}

// Split one line at its first separator into a trimmed key and value. A line with no separator,
// or one past the shared text bound, has neither.
func machine_field(
	line Machine_File_Text, separator strings.Text,
) (key strings.Text, value strings.Text) {
	defer func() {
		strings.Text_Invariants(key, "machine_field.key")
		strings.Text_Invariants(value, "machine_field.value")
	}()
	Machine_File_Text_Invariants(line, "machine_field.line")
	strings.Text_Invariants(separator, "machine_field.separator")
	if len(line) > strings.TEXT_SIZE_MAXIMUM {
		return "", ""
	}
	before, after, found := strings.Cut(strings.Text(line), separator)
	if !found {
		return "", ""
	}
	return strings.Trim_Space(before), strings.Trim_Space(after)
}

// Parse one whole decimal figure, or zero when the text is not one.
func machine_number(text Machine_File_Text) (value Machine_Number) {
	defer func() { Machine_Number_Invariants(value, "machine_number.value") }()
	Machine_File_Text_Invariants(text, "machine_number.text")
	if len(text) > strconv.TEXT_SIZE_MAXIMUM {
		return 0
	}
	trimmed := strings.Trim_Space(strings.Text(text))
	parsed, parse_err := strconv.Parse_Unsigned_Integer(
		strconv.Text(trimmed), strconv.DECIMAL_BASE, strconv.BIT_SIZE_MAXIMUM)
	if parse_err != nil {
		return 0
	}
	return Machine_Number(parsed)
}

// Convert one sysfs cache size such as "48K" to bytes, or zero when absent or malformed.
func machine_cache(path Machine_File_Path) (size Machine_Number) {
	defer func() { Machine_Number_Invariants(size, "machine_cache.size") }()
	Machine_File_Path_Invariants(path, "machine_cache.path")
	text := strings.Trim_Space(strings.Text(machine_file(path)))
	if len(text) == 0 {
		return 0
	}
	switch text[len(text)-1] {
	case 'K':
		return bits.KIBIBYTE_BYTES * machine_number(Machine_File_Text(text[:len(text)-1]))
	case 'M':
		return bits.MEBIBYTE_BYTES * machine_number(Machine_File_Text(text[:len(text)-1]))
	}
	return machine_number(Machine_File_Text(text))
}

// Convert the MemTotal kibibyte figure of /proc/meminfo to bytes.
func machine_memory(meminfo Machine_File_Text) (size Physical_Memory_Size) {
	defer func() { Physical_Memory_Size_Invariants(size, "machine_memory.size") }()
	Machine_File_Text_Invariants(meminfo, "machine_memory.meminfo")
	start_index := 0
	for end_index := 0; end_index <= len(meminfo); end_index++ {
		if end_index < len(meminfo) {
			if meminfo[end_index] != '\n' {
				continue
			}
		}
		key, value := machine_field(meminfo[start_index:end_index], ":")
		start_index = end_index + 1
		if key != "MemTotal" {
			continue
		}
		figure, _, _ := strings.Cut(value, " ")
		return Physical_Memory_Size(
			bits.KIBIBYTE_BYTES * machine_number(Machine_File_Text(figure)))
	}
	return 0
}

// Read NAME and VERSION_ID from /etc/os-release. A host without it is plain Linux.
func machine_release(release Machine_File_Text) (
	name Operating_System_Name, version Operating_System_Version,
) {
	defer func() {
		Operating_System_Name_Invariants(name, "machine_release.name")
		Operating_System_Version_Invariants(version, "machine_release.version")
	}()
	Machine_File_Text_Invariants(release, "machine_release.release")
	start_index := 0
	for end_index := 0; end_index <= len(release); end_index++ {
		if end_index < len(release) {
			if release[end_index] != '\n' {
				continue
			}
		}
		key, value := machine_field(release[start_index:end_index], "=")
		start_index = end_index + 1
		switch key {
		case "NAME":
			name = Operating_System_Name(strings.Trim(value, "\""))
		case "VERSION_ID":
			version = Operating_System_Version(strings.Trim(value, "\""))
		}
	}
	if name == "" {
		name = "Linux"
	}
	return name, version
}

// Utsname_Field is one fixed zero-padded uname field.
type Utsname_Field []int8

// Utsname_Field_Invariants holds the field to the kernel's fixed width.
func Utsname_Field_Invariants(value Utsname_Field, namespace aver.Namespace) {
	aver.Always(len(value) == UTSNAME_FIELD_BYTES, "A uname field has the kernel's width.")
}

// UTSNAME_FIELD_BYTES is the width of every Linux uname field.
const UTSNAME_FIELD_BYTES = len(syscall.Utsname{}.Release)

// Read one uname field up to its first zero byte.
func machine_utsname(field Utsname_Field) (text Kernel_Version) {
	defer func() { Kernel_Version_Invariants(text, "machine_utsname.text") }()
	Utsname_Field_Invariants(field, "machine_utsname.field")
	characters := make([]byte, 0, len(field))
	for _, character := range field {
		if character == 0 {
			break
		}
		characters = append(characters, byte(character))
	}
	return Kernel_Version(characters)
}
