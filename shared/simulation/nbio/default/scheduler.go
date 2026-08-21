//go:build darwin || (linux && amd64)

package nbio

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/path"
	"local/james-orcales/shared/simulation/nbio"
	"local/james-orcales/shared/simulation/time"
)

// Operating_System_Signal_Channel keeps the standard signal type out of the simulation OS file.
type Operating_System_Signal_Channel struct {
	// Channel stays opaque to the file that also imports the simulated OS package.
	Channel chan os.Signal
}

func operating_system_signal_channel(depth_count int) (channel Operating_System_Signal_Channel) {
	channel.Channel = make(chan os.Signal, depth_count)
	return channel
}

func operating_system_signal_notify(
	channel Operating_System_Signal_Channel, system syscall.Signal,
) {
	signal.Notify(channel.Channel, system)
}

func operating_system_signal_stop(channel Operating_System_Signal_Channel) {
	signal.Stop(channel.Channel)
}

func operating_system_signal_receive(
	channel Operating_System_Signal_Channel,
) (received syscall.Signal, found bool) {
	select {
	case value := <-channel.Channel:
		system, valid := value.(syscall.Signal)
		if !valid {
			panic("io: signal notifier returned an unknown signal type")
		}
		return system, true
	default:
		return 0, false
	}
}

// This bound hold largest socket address of either supported platform.
const SOCKET_ADDRESS_BYTES = 28

// Operating system operation kind is one kernel operation tag. Platform backend
// translate tag direct to kqueue readiness plus syscall on Darwin, or to matching io_uring
// opcode on Linux.
type Operating_System_Operation_Kind int

// OPERATING_SYSTEM_OPERATION_ACCEPT select platform accept operation.
const OPERATING_SYSTEM_OPERATION_ACCEPT Operating_System_Operation_Kind = 0

// OPERATING_SYSTEM_OPERATION_CLOSE select platform close operation.
const OPERATING_SYSTEM_OPERATION_CLOSE Operating_System_Operation_Kind = 1

// OPERATING_SYSTEM_OPERATION_CONNECT select platform connect operation.
const OPERATING_SYSTEM_OPERATION_CONNECT Operating_System_Operation_Kind = 2

// OPERATING_SYSTEM_OPERATION_READ select platform file-read operation.
const OPERATING_SYSTEM_OPERATION_READ Operating_System_Operation_Kind = 3

// OPERATING_SYSTEM_OPERATION_RECEIVE select platform socket-receive operation.
const OPERATING_SYSTEM_OPERATION_RECEIVE Operating_System_Operation_Kind = 4

// OPERATING_SYSTEM_OPERATION_SEND select platform socket-send operation.
const OPERATING_SYSTEM_OPERATION_SEND Operating_System_Operation_Kind = 5

// OPERATING_SYSTEM_OPERATION_TIMEOUT select platform timeout operation.
const OPERATING_SYSTEM_OPERATION_TIMEOUT Operating_System_Operation_Kind = 6

// OPERATING_SYSTEM_OPERATION_WRITE select platform file-write operation.
const OPERATING_SYSTEM_OPERATION_WRITE Operating_System_Operation_Kind = 7

// OPERATING_SYSTEM_OPERATION_FSYNC select platform fsync operation.
const OPERATING_SYSTEM_OPERATION_FSYNC Operating_System_Operation_Kind = 8

// OPERATING_SYSTEM_OPERATION_OPEN_AT select platform Open_At operation.
const OPERATING_SYSTEM_OPERATION_OPEN_AT Operating_System_Operation_Kind = 9

// OPERATING_SYSTEM_OPERATION_EVENT select platform event operation.
const OPERATING_SYSTEM_OPERATION_EVENT Operating_System_Operation_Kind = 10

// OPERATING_SYSTEM_OPERATION_STATX select Linux-only statx operation.
const OPERATING_SYSTEM_OPERATION_STATX Operating_System_Operation_Kind = 11

// OPERATING_SYSTEM_OPERATION_BOUNDED_DEADLINE identify internal linked deadline.
const OPERATING_SYSTEM_OPERATION_BOUNDED_DEADLINE Operating_System_Operation_Kind = 12

// OPERATING_SYSTEM_OPERATION_PROCESS_EXIT wait for one spawned child to exit. Darwin arm
// EVFILT_PROC/NOTE_EXIT kevent on process identifier, Linux poll child pidfd, thus neither
// platform block thread in wait4.
const OPERATING_SYSTEM_OPERATION_PROCESS_EXIT Operating_System_Operation_Kind = 13

// OPERATING_SYSTEM_OPERATION_PIPE_READ read one buffer from pipe. It is distinct from READ
// because pipe is not seekable: READ issue pread on Darwin and carry file offset on Linux, and
// both reject pipe with ESPIPE.
const OPERATING_SYSTEM_OPERATION_PIPE_READ Operating_System_Operation_Kind = 14

// OPERATING_SYSTEM_OPERATION_PIPE_WRITE write one buffer to pipe, counterpart of PIPE_READ.
const OPERATING_SYSTEM_OPERATION_PIPE_WRITE Operating_System_Operation_Kind = 15

// OPERATING_SYSTEM_OPERATION_MKDIR_AT make one directory. It is mkdirat primitive, thus parent
// must exist and existing path is error.
const OPERATING_SYSTEM_OPERATION_MKDIR_AT Operating_System_Operation_Kind = 16

// Kernel timespec is stable UAPI timespec layout of Linux. It live with operation, thus io_uring
// timeout never point at stack storage while it is in kernel.
type Kernel_Timespec struct {
	// Seconds keep Linux UAPI layout.
	Seconds int64
	// Nanoseconds keep Linux UAPI layout.
	Nanoseconds int64
}

// Kernel path buffer includes terminating NUL excluded from caller pathname bound.
const OPERATING_SYSTEM_PATH_BYTES_MAXIMUM = path.PATH_SIZE_MAXIMUM + path.PATH_TERMINATOR_BYTES

// Operating system operation is Go counterpart of one in-flight kernel operation.
// Identifier is written to kernel user_data instead of Go pointer: registry own operation until
// kernel retire that integer identifier.
type Operating_System_Operation struct {
	Platform_Operation
	// Used separates registered or prepared work from caller-owned capacity.
	Used bool
	// Identifier correlate operation without Go pointer in kernel.
	Identifier uint64
	// Completion keep caller identity until callback delivery.
	Completion *nbio.Completion
	// Kind select platform operation.
	Kind Operating_System_Operation_Kind
	// Descriptor keep caller ownership during asynchronous kernel use.
	Descriptor int
	// Buffer hold operation memory until kernel retire it.
	Buffer []byte
	// Offset keep requested file position.
	Offset uint64
	// Address hold typed socket address until submission.
	Address nbio.Address
	// File_Path holds zero-terminated path memory inline until kernel retirement.
	File_Path [OPERATING_SYSTEM_PATH_BYTES_MAXIMUM]byte
	// File_Path_Count includes the terminal zero and rejects absent path initialization.
	File_Path_Count int
	// Open_Options hold Open_At behavior until submission.
	Open_Options nbio.Open_At_Options
	// Event_Value correlate synthetic event without Go pointer.
	Event_Value uint64
	// Process_Identifier name spawned child a PROCESS_EXIT operation wait for. Darwin use it
	// as kevent Ident, and both platform reap with it.
	Process_Identifier int
	// Initiated stop Darwin from issue of connect twice after readiness.
	Initiated bool
	// Timespec hold timeout memory until kernel retire it.
	Timespec Kernel_Timespec
	// Deadline bound operation on Darwin.
	Deadline time.Monotonic_Moment
	// Deadline_Span hold linked-timeout memory until Linux retire it.
	Deadline_Span Kernel_Timespec
	// Bounded join one Linux operation to its linked deadline.
	Bounded *Operating_System_Bounded_Operation
	// Bounded_State keeps Linux linked retirement inside caller-owned operation storage.
	Bounded_State Operating_System_Bounded_Operation
	// Internal_Completion gives a linked deadline stable storage without heap ownership.
	Internal_Completion nbio.Completion
	// Socket_Address hold sockaddr memory until kernel retire it.
	Socket_Address [SOCKET_ADDRESS_BYTES]byte
	// Socket_Address_Size tell kernel which sockaddr bytes are valid.
	Socket_Address_Size uint32
	// Deliver keeps callback delivery behind scheduler retirement, after Completion holds the
	// kernel result.
	Deliver nbio.Callback
	// Pinner stop Go runtime from move of kernel-owned memory.
	Pinner runtime.Pinner
	// Pinned stop duplicate unpin operation.
	Pinned bool
}

// Operating system bounded operation join primary operation and its internal Linux link timeout
// before it expose either result. Darwin use Deadline direct and never allocate join.
type Operating_System_Bounded_Operation struct {
	// Operation hold primary operation until both linked SQEs retire.
	Operation *Operating_System_Operation
	// Deadline_Operation hold internal timeout until both linked SQEs retire.
	Deadline_Operation *Operating_System_Operation
	// Operation_Result keep primary result until linked timeout retire.
	Operation_Result int32
	// Deadline_Result keep timeout result until primary operation retire.
	Deadline_Result int32
	// Operation_Completed stop duplicate primary retirement.
	Operation_Completed bool
	// Deadline_Completed stop duplicate deadline retirement.
	Deadline_Completed bool
}

// Operating system operation submit give operation its kernel correlation identifier and hand
// it to platform scheduler. Submission failure is delivered through same completed queue as
// ordinary kernel result.
func operating_system_operation_submit(
	state *Operating_System, operation *Operating_System_Operation,
) {
	operating_system_operation_register(state, operation)
	err := platform_submit(state, operation)
	if err != nil {
		result := 0
		if err == nbio.Deadline_Exceeded {
			result = operating_system_timeout_result(operation)
		}
		operating_system_operation_complete(state, operation, result, err)
	}
}

// Operating system operation register allocate generation token written to kernel userdata.
func operating_system_operation_register(
	state *Operating_System, operation *Operating_System_Operation,
) {
	stable_event_identifier := false
	if operation.Kind == OPERATING_SYSTEM_OPERATION_EVENT {
		operation.Identifier = operation.Completion.Kernel_Identifier
		stable_event_identifier = operation.Identifier != 0
	}
	if operation.Identifier == 0 {
		state.Next_Identifier++
		operation.Identifier = state.Next_Identifier
	}
	// Trigger thread reads event token while loop rearms listener. Reused token needs no write,
	// since even same-value write races that read.
	if !stable_event_identifier {
		operation.Completion.Kernel_Identifier = operation.Identifier
	}
	invariant.Always(len(state.Operations) < cap(state.Operations),
		"The caller-owned operation registry has capacity before submission.")
	state.Operations = append(state.Operations, operation)
}

// Caller slots keep operation addresses stable while kernel retains them.
func operating_system_operation_acquire(
	state *Operating_System, value Operating_System_Operation,
) (operation *Operating_System_Operation) {
	for index := range state.Operation_Memory {
		if state.Operation_Memory[index].Used {
			continue
		}
		value.Used = true
		state.Operation_Memory[index] = value
		return &state.Operation_Memory[index]
	}
	invariant.Always(false, "The caller-owned operation pool has capacity before submission.")
	return nil
}

// Kernel identifiers are opaque generations, so bounded linear storage needs explicit lookup.
func operating_system_operation_find(
	state *Operating_System, identifier uint64,
) (operation *Operating_System_Operation) {
	for _, candidate := range state.Operations {
		if candidate.Identifier == identifier {
			return candidate
		}
	}
	return nil
}

// Registry removal happens before callback publication so resubmission sees retired ownership.
func operating_system_operation_unregister(
	state *Operating_System, operation *Operating_System_Operation,
) {
	found := false
	kept := state.Operations[:0]
	for _, candidate := range state.Operations {
		if candidate == operation {
			found = true
			continue
		}
		kept = append(kept, candidate)
	}
	state.Operations = kept
	invariant.Always(found, "A completed operation was registered exactly once.")
}

func operating_system_operation_release(operation *Operating_System_Operation) {
	pinner := operation.Pinner
	*operation = Operating_System_Operation{Pinner: pinner}
}

func operating_system_operation_path_set(
	operation *Operating_System_Operation, path string,
) (err error) {
	if len(path) >= len(operation.File_Path) {
		return syscall.ENAMETOOLONG
	}
	for index := 0; index < len(path); index++ {
		if path[index] == 0 {
			return syscall.EINVAL
		}
		operation.File_Path[index] = path[index]
	}
	operation.File_Path_Count = len(path) + 1
	return nil
}

func operating_system_operation_submit_path(
	state *Operating_System, operation *Operating_System_Operation, path string,
) {
	path_err := operating_system_operation_path_set(operation, path)
	if path_err == nil {
		operating_system_operation_submit(state, operation)
		return
	}
	operation.Completion.Data = 0
	operation.Completion.Error = path_err
	operation.Completion.Callback = operation.Deliver
	operating_system_completion_add(state, operation.Completion)
	operating_system_operation_release(operation)
}

func operating_system_completion_add(
	state *Operating_System, completion *nbio.Completion,
) {
	invariant.Always(len(state.Completed) < cap(state.Completed),
		"The caller-owned completion queue has capacity before publication.")
	state.Completed = append(state.Completed, completion)
}

// Operating system operation complete retire registry entry and every pinned address before it
// expose result to application code. Retire come before deliver, thus callback that resubmit
// same completion find registry already clear.
func operating_system_operation_complete(
	state *Operating_System, operation *Operating_System_Operation,
	result int, err error,
) {
	operating_system_operation_unregister(state, operation)
	if operation.Pinned {
		operation.Pinner.Unpin()
		operation.Pinned = false
	}
	operating_system_operation_account(state, operation, result, err)
	operation.Completion.Data = result
	operation.Completion.Error = err
	operation.Completion.Callback = operation.Deliver
	operating_system_completion_add(state, operation.Completion)
	operating_system_operation_release(operation)
}

// Operating system operation account update descriptor ownership only after kernel complete
// operation that make or release descriptor.
func operating_system_operation_account(
	state *Operating_System, operation *Operating_System_Operation,
	result int, err error,
) {
	if err == nil {
		if operation.Kind == OPERATING_SYSTEM_OPERATION_ACCEPT {
			operating_system_descriptor_add(state, result)
		}
	}
	if err == nil {
		if operation.Kind == OPERATING_SYSTEM_OPERATION_OPEN_AT {
			operating_system_descriptor_add(state, result)
		}
	}
	if err == nil {
		if operation.Kind == OPERATING_SYSTEM_OPERATION_CLOSE {
			operating_system_descriptor_remove(state, operation.Descriptor)
		}
	}
}

func operating_system_descriptor_add(state *Operating_System, descriptor int) {
	for index := range state.Raw_Open {
		if state.Raw_Open[index].Used {
			continue
		}
		state.Raw_Open[index] = Operating_System_Descriptor{
			Used: true, Descriptor: descriptor,
		}
		return
	}
	invariant.Always(false, "The caller-owned descriptor census has capacity before ownership.")
}

func operating_system_descriptor_remove(state *Operating_System, descriptor int) {
	found := false
	for index := range state.Raw_Open {
		if !state.Raw_Open[index].Used {
			continue
		}
		if state.Raw_Open[index].Descriptor != descriptor {
			continue
		}
		state.Raw_Open[index] = Operating_System_Descriptor{}
		found = true
		break
	}
	invariant.Always(found, "A released descriptor belonged to the backend.")
}

func operating_system_descriptor_count(state *Operating_System) (count int) {
	for index := range state.Raw_Open {
		if state.Raw_Open[index].Used {
			count++
		}
	}
	return count
}

// Operating system translate result apply portable result variants nbio expose. Other
// errno values stay raw operating-system errors.
func operating_system_translate_result(
	operation *Operating_System_Operation, result int32,
) (count int, err error) {
	if result >= 0 {
		return int(result), nil
	}
	errno := syscall.Errno(-result)
	if errno == syscall.ECONNREFUSED {
		return 0, nbio.Connection_Refused
	}
	if errno == syscall.EPIPE {
		return 0, nbio.Broken_Pipe
	}
	if errno == syscall.ENOTCONN {
		return 0, nbio.Socket_Not_Connected
	}
	if errno == syscall.ECANCELED {
		return 0, nbio.Canceled
	}
	if errno == syscall.EEXIST {
		return 0, nbio.Path_Exists
	}
	if operation.Kind == OPERATING_SYSTEM_OPERATION_TIMEOUT {
		if errno == syscall.ETIME {
			return 0, nil
		}
	}
	if operation.Kind == OPERATING_SYSTEM_OPERATION_CLOSE {
		if errno == syscall.EINTR {
			return 0, nil
		}
	}
	return 0, errno
}

// Operating system retryable result report io_uring results backend resubmit, not deliver.
// EINTR apply to every kernel operation except close. File read and write also retry EAGAIN.
func operating_system_retryable_result(
	operation *Operating_System_Operation, result int32,
) (retry bool) {
	if result >= 0 {
		return false
	}
	errno := syscall.Errno(-result)
	if errno == syscall.EINTR {
		return operation.Kind != OPERATING_SYSTEM_OPERATION_CLOSE
	}
	if errno != syscall.EAGAIN {
		return false
	}
	if operation.Kind == OPERATING_SYSTEM_OPERATION_READ {
		return true
	}
	if operation.Kind == OPERATING_SYSTEM_OPERATION_PIPE_READ {
		return true
	}
	if operation.Kind == OPERATING_SYSTEM_OPERATION_PIPE_WRITE {
		return true
	}
	return operation.Kind == OPERATING_SYSTEM_OPERATION_WRITE
}

// Byte-transfer callbacks use zero on timeout, while descriptor-producing callbacks use -1.
func operating_system_timeout_result(
	operation *Operating_System_Operation,
) (result int) {
	if operation.Kind == OPERATING_SYSTEM_OPERATION_RECEIVE {
		return 0
	}
	if operation.Kind == OPERATING_SYSTEM_OPERATION_SEND {
		return 0
	}
	if operation.Kind == OPERATING_SYSTEM_OPERATION_READ {
		return 0
	}
	if operation.Kind == OPERATING_SYSTEM_OPERATION_WRITE {
		return 0
	}
	if operation.Kind == OPERATING_SYSTEM_OPERATION_FSYNC {
		return 0
	}
	return -1
}

// Operating system timeout span convert positive duration to stable kernel layout.
func operating_system_timeout_span(duration time.Duration) (span Kernel_Timespec) {
	nanoseconds := int64(duration)
	span.Seconds = nanoseconds / 1_000_000_000
	span.Nanoseconds = nanoseconds % 1_000_000_000
	return span
}
