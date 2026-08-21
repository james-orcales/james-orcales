//go:build darwin

package os

import (
	"runtime"
	"syscall"
	"unsafe"

	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/slices"
)

// Darwin exposes execve through its BSD syscall table.
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

// SYSCTL_WORD_BYTES is one 64-bit sysctl scalar.
const SYSCTL_WORD_BYTES = 8

// Sysctl_Name is one sysctl key, such as hw.memsize.
type Sysctl_Name string

// Sysctl_Name_Invariants holds a key to the dotted names the kernel publishes.
func Sysctl_Name_Invariants(value Sysctl_Name, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), SYSCTL_NAME_BYTES_MINIMUM, SYSCTL_NAME_BYTES_MAXIMUM).
		Ensure()
}

// SYSCTL_NAME_BYTES_MINIMUM is the shortest key read here, hw.l3cachesize's peers included.
const SYSCTL_NAME_BYTES_MINIMUM = len("hw.memsize")

// SYSCTL_NAME_BYTES_MAXIMUM is the kernel's CTL_MAXNAME-bounded name text.
const SYSCTL_NAME_BYTES_MAXIMUM = 256

// Sysctl_Text is one text sysctl value.
type Sysctl_Text string

// Sysctl_Text_Invariants bounds text the kernel describes itself with.
func Sysctl_Text_Invariants(value Sysctl_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), slices.COUNT_MINIMUM, slices.COUNT_MAXIMUM).
		Ensure()
}

// Sysctl_Raw is the byte string syscall.Sysctl returns for a scalar key.
type Sysctl_Raw string

// Sysctl_Raw_Invariants admits every width a scalar arrives in, one trimmed byte included.
func Sysctl_Raw_Invariants(value Sysctl_Raw, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), MACHINE_VALUE_ABSENT, SYSCTL_WORD_BYTES).
		Ensure()
}

// Sysctl_Word is one scalar sysctl value.
type Sysctl_Word uint64

// Sysctl_Word_Invariants admits the whole scalar domain: sizes, counts, and rates share it.
func Sysctl_Word_Invariants(value Sysctl_Word, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint64(uint64(value), MACHINE_VALUE_ABSENT, STORAGE_BYTES_MAXIMUM).
		Ensure()
}

// Machine_Read describes the host through sysctl. Apple Silicon has no third-level cache, thus
// that key is absent and reads zero.
func Machine_Read() (machine Machine) {
	defer func() { Machine_Invariants(machine, "Machine_Read.machine") }()
	machine.Processor_Model = Processor_Model(machine_sysctl_text("machdep.cpu.brand_string"))
	machine.Processor_Architecture = Processor_Architecture(runtime.GOARCH)
	machine.Physical_Core_Count = Physical_Core_Count(machine_sysctl_word("hw.physicalcpu"))
	machine.Logical_Core_Count = Logical_Core_Count(machine_sysctl_word("hw.logicalcpu"))
	// Apple Silicon names performance cores perflevel0 and efficiency cores perflevel1. A
	// homogeneous processor has neither, thus both stay zero.
	machine.Performance_Core_Count = Performance_Core_Count(
		machine_sysctl_word("hw.perflevel0.physicalcpu"))
	machine.Efficiency_Core_Count = Efficiency_Core_Count(
		machine_sysctl_word("hw.perflevel1.physicalcpu"))
	machine.Processor_Frequency = Processor_Frequency(
		machine_sysctl_word("hw.perflevel0.cpufrequency_max"))
	machine.Level_1_Cache_Size = Level_1_Cache_Size(
		machine_sysctl_word("hw.perflevel0.l1dcachesize"))
	machine.Level_2_Cache_Size = Level_2_Cache_Size(
		machine_sysctl_word("hw.perflevel0.l2cachesize"))
	machine.Level_3_Cache_Size = Level_3_Cache_Size(machine_sysctl_word("hw.l3cachesize"))
	machine.Physical_Memory_Size = Physical_Memory_Size(machine_sysctl_word("hw.memsize"))
	machine.Storage_Size = machine_storage()
	machine.Operating_System_Name = "macOS"
	machine.Operating_System_Version = Operating_System_Version(
		machine_sysctl_text("kern.osproductversion"))
	machine.Kernel_Version = Kernel_Version("Darwin " + machine_sysctl_text("kern.osrelease"))
	return machine
}

// Read one text sysctl, or empty text when the key is absent.
func machine_sysctl_text(name Sysctl_Name) (text Sysctl_Text) {
	defer func() { Sysctl_Text_Invariants(text, "machine_sysctl_text.text") }()
	Sysctl_Name_Invariants(name, "machine_sysctl_text.name")
	value, sysctl_err := syscall.Sysctl(string(name))
	if sysctl_err != nil {
		return ""
	}
	return Sysctl_Text(value)
}

// Read one scalar sysctl of 32 or 64 bits, or zero when the key is absent. Sysctl trims one
// trailing zero byte as if the value were text, thus the width is restored with zeros before
// decoding.
func machine_sysctl_word(name Sysctl_Name) (value Sysctl_Word) {
	defer func() { Sysctl_Word_Invariants(value, "machine_sysctl_word.value") }()
	Sysctl_Name_Invariants(name, "machine_sysctl_word.name")
	raw, sysctl_err := syscall.Sysctl(string(name))
	if sysctl_err != nil {
		return 0
	}
	if len(raw) > SYSCTL_WORD_BYTES {
		return 0
	}
	Sysctl_Raw_Invariants(Sysctl_Raw(raw), "machine_sysctl_word.raw")
	storage := [SYSCTL_WORD_BYTES]byte{}
	copy(storage[:], raw)
	return Sysctl_Word(binary.Uint_64(binary.Bytes(storage[:]), binary.LITTLE_ENDIAN))
}
