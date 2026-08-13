//go:build unix

package nbio

import (
	"encoding/binary"
	"errors"
	"net"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	invariant "local/james-orcales/shared/invariant/default"
	io "local/james-orcales/shared/simulation/nbio"
	sysos "local/james-orcales/shared/simulation/os"
	"local/james-orcales/shared/simulation/time"
)

// A pipe has exactly two ends: the read end and the write end.
const PIPE_ENDS = 2

// Caps the EINTR retries of one reap. A signal can interrupt wait4, but only a broken kernel
// interrupts it repeatedly, so a bound turns that into a reported error rather than a spin.
const PROCESS_REAP_RETRIES_MAX = 16

// Bounds one readdir pass into a fixed buffer, so a large directory is read in repeated
// passes rather than one unbounded allocation.
const DIRECTORY_READ_BYTES = 8192

// Caps the number of readdir passes so a pathological directory errors rather than looping
// unbounded; 4096 passes of directory_read_bytes cover hundreds of thousands of entries.
const DIRECTORY_READ_PASSES_MAX = 4096

// Reads up to len(buffer) bytes from file at offset via the pread syscall — the raw
// positioned read TigerBeetle's posix backend uses.
func read_at(file io.File, buffer []byte, offset int64) (count int, err error) {
	return syscall.Pread(int(file), buffer, offset)
}

// Writes buffer to file at offset via the pwrite syscall.
func write_at(file io.File, buffer []byte, offset int64) (count int, err error) {
	return syscall.Pwrite(int(file), buffer, offset)
}

// Opens the file at path for reading via the open syscall, returning its descriptor.
func file_open(path string) (descriptor int, err error) {
	return syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC, 0)
}

// Creates or truncates path for writing via the open syscall, returning its descriptor.
func file_create(path string) (descriptor int, err error) {
	return syscall.Open(
		path, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_TRUNC|syscall.O_CLOEXEC, 0o644,
	)
}

// Reports whether path exists, whether it is a directory, and its byte size via lstat. An
// absent path is Exists false with a nil error, so a caller distinguishes "not there" from a
// real stat failure.
func file_status(path string) (status io.File_Status, err error) {
	metadata := syscall.Stat_t{}
	stat_err := syscall.Lstat(path, &metadata)
	if stat_err == syscall.ENOENT {
		return io.File_Status{}, nil
	}
	if stat_err != nil {
		return io.File_Status{}, stat_err
	}
	return io.File_Status{
		Exists:       true,
		Is_Directory: metadata.Mode&syscall.S_IFMT == syscall.S_IFDIR,
		Is_Regular:   metadata.Mode&syscall.S_IFMT == syscall.S_IFREG,
		Size:         metadata.Size,
	}, nil
}

// The mode a created directory takes: readable and traversable by all, writable by the owner.
const DIRECTORY_MODE = 0o755

// Creates path and every missing parent. Each component is created in turn, and an existing
// directory converges rather than failing, so a repeated call is not an error. The walk is
// bounded by the path length, and the final Status check rejects a path whose last component
// exists as something other than a directory — the one case a converging mkdir would hide.
func directory_make(path string) (err error) {
	if path == "" {
		return syscall.ENOENT
	}
	for index := 1; index <= len(path); index++ {
		if index < len(path) {
			if path[index] != '/' {
				continue
			}
		}
		mkdir_err := syscall.Mkdir(path[:index], DIRECTORY_MODE)
		if mkdir_err == nil {
			continue
		}
		if mkdir_err != syscall.EEXIST {
			return mkdir_err
		}
	}
	status, status_err := file_status(path)
	if status_err != nil {
		return status_err
	}
	if !status.Is_Directory {
		return syscall.ENOTDIR
	}
	return nil
}

// Reads one pass of a directory's raw entries into buffer and returns the children it names.
// The record layout is per-platform, so the walk casts to syscall.Dirent and lets the platform
// struct supply the offsets rather than naming a byte position. The pass loop, the descriptor
// lifetime, and the accumulation across passes all compose above the surface in Read_Directory.
func file_directory_pass(
	descriptor int, buffer []byte,
) (entries []io.Directory_Entry, err error) {
	count, read_err := platform_directory_read(descriptor, buffer)
	if read_err != nil {
		return nil, read_err
	}
	entries = []io.Directory_Entry{}
	for offset := 0; offset < count; {
		record := (*syscall.Dirent)(unsafe.Pointer(&buffer[offset]))
		record_bytes := int(record.Reclen)
		invariant.Always(record_bytes > 0, "A directory record spans at least one byte.")
		invariant.Always(offset+record_bytes <= count,
			"A directory record ends inside the bytes the kernel returned.")
		entry, keep, entry_err := file_directory_entry(descriptor, record, record_bytes)
		if entry_err != nil {
			return nil, entry_err
		}
		if keep {
			entries = append(entries, entry)
		}
		offset += record_bytes
	}
	return entries, nil
}

// Decodes one directory record. keep is false for the two self-referencing entries and for a
// record the filesystem already deleted, which is what syscall.ParseDirent dropped before this
// walk replaced it.
func file_directory_entry(
	descriptor int, record *syscall.Dirent, record_bytes int,
) (entry io.Directory_Entry, keep bool, err error) {
	if platform_directory_absent(record) {
		return io.Directory_Entry{}, false, nil
	}
	name := file_directory_name(record, record_bytes)
	if name == "." {
		return io.Directory_Entry{}, false, nil
	}
	if name == ".." {
		return io.Directory_Entry{}, false, nil
	}
	directory_bit, kind_err := file_directory_kind(descriptor, record, name)
	if kind_err != nil {
		return io.Directory_Entry{}, false, kind_err
	}
	return io.Directory_Entry{Name: name, Is_Directory: directory_bit}, true, nil
}

// Returns the name a directory record holds. The kernel terminates the name with a NUL byte
// inside the record, so the terminator bounds it on both platforms. Darwin also counts the name
// bytes in a field of its own, but Linux does not, and the terminator serves both.
func file_directory_name(record *syscall.Dirent, record_bytes int) (name string) {
	start := int(unsafe.Offsetof(record.Name))
	invariant.Always(record_bytes > start, "A directory record holds at least one name byte.")
	bytes := unsafe.Slice((*byte)(unsafe.Pointer(&record.Name[0])), record_bytes-start)
	for index, value := range bytes {
		if value == 0 {
			return string(bytes[:index])
		}
	}
	return string(bytes)
}

// Reports whether a directory record names a directory. The kernel already wrote the kind into
// the record, so the common case costs no syscall at all. Only a filesystem that leaves the
// field empty — XFS made with ftype=0, or NFS without READDIRPLUS — costs one fstatat.
func file_directory_kind(
	descriptor int, record *syscall.Dirent, name string,
) (directory_bit bool, err error) {
	if record.Type == syscall.DT_UNKNOWN {
		return file_status_at(descriptor, name)
	}
	return record.Type == syscall.DT_DIR, nil
}

// Reports whether name, resolved against the open directory, is itself a directory. Go exports
// no Fstatat on this platform, so the call goes by trap number the way Open_At and Mkdir_At do.
func file_status_at(directory int, name string) (directory_bit bool, err error) {
	bytes, convert_err := syscall.BytePtrFromString(name)
	if convert_err != nil {
		return false, convert_err
	}
	metadata := syscall.Stat_t{}
	_, _, errno := syscall.Syscall6(
		PLATFORM_STAT_AT_CALL, uintptr(directory), uintptr(unsafe.Pointer(bytes)),
		uintptr(unsafe.Pointer(&metadata)), PLATFORM_SYMBOLIC_LINK_NO_FOLLOW, 0, 0,
	)
	if errno != 0 {
		return false, errno
	}
	return metadata.Mode&syscall.S_IFMT == syscall.S_IFDIR, nil
}

// Reads up to len(buffer) bytes from a pipe. A pipe is not seekable, so this is plain read
// rather than the pread read_at uses; again is true when the writer has produced nothing yet,
// and a zero count with no error is the writer's end closing.
func pipe_read(descriptor int, buffer []byte) (count int, again bool, err error) {
	count, err = syscall.Read(descriptor, buffer)
	if err != nil {
		return 0, socket_again(err), err
	}
	return count, false, nil
}

// Writes up to len(buffer) bytes to a pipe; again is true when the pipe buffer is full and the
// operation must stay armed.
func pipe_write(descriptor int, buffer []byte) (count int, again bool, err error) {
	count, err = syscall.Write(descriptor, buffer)
	if err != nil {
		return 0, socket_again(err), socket_send_translate(err)
	}
	return count, false, nil
}

// Creates a pipe and returns its read and write ends. Neither end is configured: the caller
// keeps one end and hands the other to the child, and only the kept end takes close-on-exec and
// non-blocking mode. The two ends are separate open file descriptions, so configuring one does
// not change what the child sees on the other.
func pipe_open() (read int, write int, err error) {
	descriptors := [PIPE_ENDS]int{}
	if pipe_err := syscall.Pipe(descriptors[:]); pipe_err != nil {
		return -1, -1, pipe_err
	}
	return descriptors[0], descriptors[1], nil
}

// Configures the end of a pipe the loop keeps: close-on-exec so a later spawn does not inherit
// it, and non-blocking so the loop's eager attempt reports EAGAIN instead of waiting.
func pipe_retain(descriptor int) (err error) {
	if flag_err := descriptor_close_on_exec(descriptor); flag_err != nil {
		return flag_err
	}
	return syscall.SetNonblock(descriptor, true)
}

// Sets FD_CLOEXEC on descriptor. Both platforms carry SYS_FCNTL, so one raw call covers them.
func descriptor_close_on_exec(descriptor int) (err error) {
	_, _, errno := syscall.Syscall(
		syscall.SYS_FCNTL, uintptr(descriptor), uintptr(syscall.F_SETFD),
		uintptr(syscall.FD_CLOEXEC),
	)
	if errno != 0 {
		return errno
	}
	return nil
}

// Resolves an executable name the way a shell does, because syscall.StartProcess requires a
// path while callers pass bare names such as sh or fc-cache. A name holding a slash is used as
// given. This mirrors os/exec's own fallback: reject a directory, then accept any execute bit,
// and let StartProcess report EACCES when the mode bits promise more than the file allows.
func executable_path(name string, search string) (path string, err error) {
	if name == "" {
		return "", syscall.ENOENT
	}
	if strings.Contains(name, "/") {
		return name, executable_check(name)
	}
	for _, directory := range strings.Split(search, ":") {
		if directory == "" {
			directory = "."
		}
		candidate := filepath.Join(directory, name)
		if executable_check(candidate) == nil {
			return candidate, nil
		}
	}
	return "", syscall.ENOENT
}

// Reports whether path names a regular file carrying an execute bit.
func executable_check(path string) (err error) {
	metadata := syscall.Stat_t{}
	if stat_err := syscall.Stat(path, &metadata); stat_err != nil {
		return stat_err
	}
	if metadata.Mode&syscall.S_IFMT == syscall.S_IFDIR {
		return syscall.EISDIR
	}
	if metadata.Mode&0o111 == 0 {
		return syscall.EACCES
	}
	return nil
}

// Reaps an exited child, returning its exit code and resource accounting. WNOHANG returns at
// once because the caller runs this only after the kernel reported the exit.
func process_reap(identifier int) (exit int, usage sysos.Process_Usage, err error) {
	status := syscall.WaitStatus(0)
	rusage := syscall.Rusage{}
	reaped := false
	for retry_index := 0; retry_index < PROCESS_REAP_RETRIES_MAX; retry_index++ {
		_, wait_err := syscall.Wait4(identifier, &status, syscall.WNOHANG, &rusage)
		if wait_err == syscall.EINTR {
			continue
		}
		if wait_err != nil {
			return 0, sysos.Process_Usage{}, wait_err
		}
		reaped = true
		break
	}
	if !reaped {
		return 0, sysos.Process_Usage{}, syscall.EINTR
	}
	usage.CPU_User = time.Duration(
		rusage.Utime.Sec*1_000_000_000 + int64(rusage.Utime.Usec)*1_000)
	usage.CPU_System = time.Duration(
		rusage.Stime.Sec*1_000_000_000 + int64(rusage.Stime.Usec)*1_000)
	usage.RSS_Bytes_Max = process_rss_bytes(int64(rusage.Maxrss))
	return process_exit_code(status), usage, nil
}

// Reports the portable exit code: a signalled child reports 128 plus its signal, matching the
// convention every shell uses and the value os.ProcessState.ExitCode reported before.
func process_exit_code(status syscall.WaitStatus) (exit int) {
	if status.Signaled() {
		return 128 + int(status.Signal())
	}
	return status.ExitStatus()
}

// Reports the remote IP address of descriptor via getpeername; a non-IP peer yields the
// empty address with no error.
func socket_peer_address(descriptor int) (address string, err error) {
	storage := [SOCKET_ADDRESS_BYTES]byte{}
	size := uint32(SOCKET_ADDRESS_BYTES)
	_, _, errno := syscall.RawSyscall(
		syscall.SYS_GETPEERNAME, uintptr(descriptor),
		uintptr(unsafe.Pointer(&storage[0])), uintptr(unsafe.Pointer(&size)),
	)
	if errno != 0 {
		return "", errno
	}
	peer, decode_err := socket_address_decode(&storage, size)
	// A peer that is neither IPv4 nor IPv6 is not an error here, only an absent address.
	if decode_err == syscall.EAFNOSUPPORT {
		return "", nil
	}
	if decode_err != nil {
		return "", decode_err
	}
	if peer.Family == io.FAMILY_IPV4 {
		return net.IP(peer.IP[:io.IPV4_ADDRESS_BYTES]).String(), nil
	}
	return net.IP(peer.IP[:]).String(), nil
}

// Reports whether err is the non-blocking "try again" signal that keeps an operation
// armed rather than completing it.
func socket_again(err error) (again bool) {
	if err == syscall.EAGAIN {
		return true
	}
	return err == syscall.EWOULDBLOCK
}

// The wire size of a sockaddr_in, which bounds the bytes the kernel reads for an IPv4 address.
const SOCKET_ADDRESS_IPV4_BYTES = 16

// The wire size of a sockaddr_in6, which is also the full storage size a decode may receive.
const SOCKET_ADDRESS_IPV6_BYTES = 28

// Writes address into storage as a sockaddr and returns the byte count the kernel reads. The
// kernel takes exactly these bytes, so this replaces syscall.Sockaddr: that interface costs an
// allocation and a type switch to build a struct the wrapper converts straight back to this
// same layout. The port and the address bytes sit at the same offsets on both platforms, so
// only the two header bytes need the platform.
func socket_address_encode(
	address io.Address, storage *[SOCKET_ADDRESS_BYTES]byte,
) (size uint32, err error) {
	for index := range storage {
		storage[index] = 0
	}
	if address.Family == io.FAMILY_IPV4 {
		platform_address_header(storage, syscall.AF_INET, SOCKET_ADDRESS_IPV4_BYTES)
		binary.BigEndian.PutUint16(storage[2:4], address.Port)
		copy(storage[4:8], address.IP[:io.IPV4_ADDRESS_BYTES])
		return SOCKET_ADDRESS_IPV4_BYTES, nil
	}
	if address.Family == io.FAMILY_IPV6 {
		platform_address_header(storage, syscall.AF_INET6, SOCKET_ADDRESS_IPV6_BYTES)
		binary.BigEndian.PutUint16(storage[2:4], address.Port)
		copy(storage[8:24], address.IP[:])
		return SOCKET_ADDRESS_IPV6_BYTES, nil
	}
	return 0, syscall.EAFNOSUPPORT
}

// Reads the sockaddr the kernel wrote into storage. size is what the kernel reported it filled,
// so a truncated answer fails rather than decodes whatever the zeroed remainder happens to say.
func socket_address_decode(
	storage *[SOCKET_ADDRESS_BYTES]byte, size uint32,
) (address io.Address, err error) {
	family := platform_address_family(storage)
	port := binary.BigEndian.Uint16(storage[2:4])
	if family == syscall.AF_INET {
		if size < SOCKET_ADDRESS_IPV4_BYTES {
			return io.Address{}, syscall.EINVAL
		}
		ip := [io.IPV4_ADDRESS_BYTES]byte{}
		copy(ip[:], storage[4:8])
		return io.Address_I_Pv4(ip, port), nil
	}
	if family == syscall.AF_INET6 {
		if size < SOCKET_ADDRESS_IPV6_BYTES {
			return io.Address{}, syscall.EINVAL
		}
		ip := [io.IPV6_ADDRESS_BYTES]byte{}
		copy(ip[:], storage[8:24])
		return io.Address_I_Pv6(ip, port), nil
	}
	return io.Address{}, syscall.EAFNOSUPPORT
}

// Maps the backend-independent family to the platform's socket family constant. Each platform's
// syscall package carries its own value, so one function serves both.
func socket_family(family io.Address_Family) (system int) {
	if family == io.FAMILY_IPV6 {
		return syscall.AF_INET6
	}
	return syscall.AF_INET
}

// Resolve turns a host into the IPv4 literal socket_address_encode requires: an IP literal passes
// through unchanged, a name is looked up and its first IPv4 returned. The encode takes only an
// explicit family so no lookup runs on the dial path — a caller resolves here first and hands
// Connect an address, never a name. It is
// the prod Resolver injected into the shared http client, failing closed when the host has no IPv4.
func Resolve(host string) (address string, err error) {
	if net.ParseIP(host) != nil {
		return host, nil
	}
	found, lookup_err := net.LookupIP(host)
	if lookup_err != nil {
		return "", lookup_err
	}
	for index := range found {
		if four := found[index].To4(); four != nil {
			return four.String(), nil
		}
	}
	return "", errors.New("io: no IPv4 address for host " + host)
}

// Accepts one pending connection on listener, returning a non-blocking connected
// descriptor; again is true when none is ready yet.
func socket_accept(listener int) (descriptor int, again bool, err error) {
	descriptor, _, err = syscall.Accept(listener)
	if err != nil {
		return -1, socket_again(err), err
	}
	non_block_err := syscall.SetNonblock(descriptor, true)
	if non_block_err != nil {
		syscall.Close(descriptor)
		return -1, false, non_block_err
	}
	configure_err := socket_accept_configure(descriptor)
	if configure_err != nil {
		syscall.Close(descriptor)
		return -1, false, configure_err
	}
	return descriptor, false, nil
}

// Socket_Connect_Start_Input keeps the caller-owned descriptor and typed address together.
type Socket_Connect_Start_Input struct {
	// Descriptor preserves caller ownership during the connect syscall.
	Descriptor int
	// Address prevents a DNS name from entering the syscall boundary.
	Address io.Address
}

// Begins connecting a caller-owned descriptor; an in-progress handshake completes later.
func socket_connect_start(input *Socket_Connect_Start_Input) (err error) {
	connect_err := socket_connect_call(input.Descriptor, input.Address)
	if connect_err == nil {
		return nil
	}
	if connect_err == syscall.EINPROGRESS {
		return nil
	}
	return socket_connect_translate(connect_err)
}

// Issues one connect. Every socket here is non-blocking, so the call returns EINPROGRESS at
// once when the handshake has further to go and never waits, which is what lets it go raw.
func socket_connect_call(descriptor int, address io.Address) (err error) {
	storage := [SOCKET_ADDRESS_BYTES]byte{}
	size, encode_err := socket_address_encode(address, &storage)
	if encode_err != nil {
		return encode_err
	}
	_, _, errno := syscall.RawSyscall(
		syscall.SYS_CONNECT, uintptr(descriptor),
		uintptr(unsafe.Pointer(&storage[0])), uintptr(size),
	)
	if errno != 0 {
		return errno
	}
	return nil
}

// Returns the pending error on descriptor after a connect completes, or nil when the
// handshake succeeded.
func socket_connect_error(descriptor int) (err error) {
	value, get_err := syscall.GetsockoptInt(descriptor, syscall.SOL_SOCKET, syscall.SO_ERROR)
	if get_err != nil {
		return get_err
	}
	if value != 0 {
		return socket_connect_translate(syscall.Errno(value))
	}
	return nil
}

// Sets one portable socket option. It is the setsockopt primitive: every caller-selected socket
// setting arrives here, and the platform-mandatory ones stay in socket_open.
func socket_option_set(descriptor int, option io.Socket_Option, value int) (err error) {
	// A platform answers first, because one portable option can need a different name or a
	// privileged variant there. The shared table covers what both platforms spell the same.
	handled, platform_err := platform_option_set(descriptor, option, value)
	if handled {
		return platform_err
	}
	level, name, known := socket_option_name(option)
	if !known {
		return syscall.EINVAL
	}
	return syscall.SetsockoptInt(descriptor, level, name, value)
}

// Maps a portable socket option to its level and name.
func socket_option_name(option io.Socket_Option) (level int, name int, known bool) {
	switch option {
	case io.SOCKET_OPTION_RECEIVE_BUFFER:
		return syscall.SOL_SOCKET, syscall.SO_RCVBUF, true
	case io.SOCKET_OPTION_SEND_BUFFER:
		return syscall.SOL_SOCKET, syscall.SO_SNDBUF, true
	case io.SOCKET_OPTION_KEEPALIVE:
		return syscall.SOL_SOCKET, syscall.SO_KEEPALIVE, true
	case io.SOCKET_OPTION_REUSE_ADDRESS:
		return syscall.SOL_SOCKET, syscall.SO_REUSEADDR, true
	case io.SOCKET_OPTION_NO_DELAY:
		return syscall.IPPROTO_TCP, syscall.TCP_NODELAY, true
	}
	return 0, 0, false
}

// Gives a socket its local address, the bind primitive. bind only records the address in the
// kernel, so it cannot block and the raw call skips the scheduler handoff that syscall.Syscall
// performs for a call that can.
func socket_bind(descriptor int, address io.Address) (err error) {
	storage := [SOCKET_ADDRESS_BYTES]byte{}
	size, encode_err := socket_address_encode(address, &storage)
	if encode_err != nil {
		return encode_err
	}
	_, _, errno := syscall.RawSyscall(
		syscall.SYS_BIND, uintptr(descriptor),
		uintptr(unsafe.Pointer(&storage[0])), uintptr(size),
	)
	if errno != 0 {
		return errno
	}
	return nil
}

// Marks a bound socket as accepting, the listen primitive.
func socket_listen_mark(descriptor int, backlog uint32) (err error) {
	if backlog == 0 {
		return syscall.EINVAL
	}
	return syscall.Listen(descriptor, int(backlog))
}

// Reports a socket's own address, the getsockname primitive. The call only reads kernel state,
// so it cannot block and goes raw.
func socket_name(descriptor int) (address io.Address, err error) {
	storage := [SOCKET_ADDRESS_BYTES]byte{}
	size := uint32(SOCKET_ADDRESS_BYTES)
	_, _, errno := syscall.RawSyscall(
		syscall.SYS_GETSOCKNAME, uintptr(descriptor),
		uintptr(unsafe.Pointer(&storage[0])), uintptr(unsafe.Pointer(&size)),
	)
	if errno != 0 {
		return io.Address{}, errno
	}
	return socket_address_decode(&storage, size)
}

// Translates a filesystem errno to the backend-independent sentinel. Darwin completes a
// filesystem operation in its eager submit and never reaches the Linux result translation, so
// the mapping lives here where both platforms use it.
func file_translate(err error) (translated error) {
	if err == syscall.EEXIST {
		return io.Path_Exists
	}
	return err
}

// Translates platform-specific connect refusal to the backend-independent sentinel.
func socket_connect_translate(err error) (translated error) {
	if err == syscall.ECONNREFUSED {
		return io.Connection_Refused
	}
	return err
}

// Reads up to len(buffer) bytes from descriptor; again is true when no data is ready
// yet, and a zero count with no error marks a closed peer.
func socket_receive(descriptor int, buffer []byte) (count int, again bool, err error) {
	count, err = syscall.Read(descriptor, buffer)
	if err != nil {
		return 0, socket_again(err), err
	}
	return count, false, nil
}

// Writes up to len(buffer) bytes to descriptor; again is true when the kernel buffer
// is full and the operation must stay armed.
func socket_send(descriptor int, buffer []byte) (count int, again bool, err error) {
	count, err = syscall.Write(descriptor, buffer)
	if err != nil {
		return 0, socket_again(err), err
	}
	return count, false, nil
}

// Attempts one synchronous send, returning sent false for would-block or another send failure.
func socket_send_now(descriptor int, buffer []byte) (count int, sent bool) {
	count, again, err := socket_send(descriptor, buffer)
	if again {
		return 0, false
	}
	if err != nil {
		return 0, false
	}
	return count, true
}

// Shuts down one or both connected-socket directions synchronously.
func socket_shutdown(descriptor int, how io.Shutdown_How) (err error) {
	system_how := syscall.SHUT_RD
	if how == io.SHUTDOWN_SEND {
		system_how = syscall.SHUT_WR
	}
	if how == io.SHUTDOWN_BOTH {
		system_how = syscall.SHUT_RDWR
	}
	err = syscall.Shutdown(descriptor, system_how)
	if err == syscall.ENOTCONN {
		return io.Socket_Not_Connected
	}
	return socket_send_translate(err)
}

// Translates send-side broken-pipe errors to the portable result.
func socket_send_translate(err error) (translated error) {
	if err == syscall.EPIPE {
		return io.Broken_Pipe
	}
	return err
}

// Releases descriptor.
func socket_close(descriptor int) (err error) {
	return syscall.Close(descriptor)
}
