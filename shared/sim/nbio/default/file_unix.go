//go:build unix

package nbio

import (
	"syscall"
	"unsafe"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/encoding/binary"
	"local/james-orcales/shared/filepath"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/sim/nbio"
	"local/james-orcales/shared/sim/time"
)

// Pipe has exactly two ends: read end and write end.
const PIPE_ENDS = 2

// Cap EINTR retries of one reap. Signal can interrupt wait4, but only broken kernel interrupt
// it over and over, thus bound turn that into reported error, not spin.
const PROCESS_REAP_RETRIES_MAX = 16

// EXECUTABLE_SEARCH_TEXT_BYTES_MAXIMUM bounds the environment text one spawn scans.
const EXECUTABLE_SEARCH_TEXT_BYTES_MAXIMUM = 4 * bits.KIBIBYTE_BYTES

// DIRECTORY_RECORD_SIZE_MINIMUM is shortest record on both supported 64-bit Unix ABIs.
const DIRECTORY_RECORD_SIZE_MINIMUM = 3 * bits.BIT_COUNT_8_MAXIMUM

// Read up to len(buffer) bytes from file at offset through pread syscall — raw positioned read.
func read_at(file nbio.File, buffer []byte, offset int64) (count int, err error) {
	return syscall.Pread(int(file), buffer, offset)
}

// Write buffer to file at offset through pwrite syscall.
func write_at(file nbio.File, buffer []byte, offset int64) (count int, err error) {
	return syscall.Pwrite(int(file), buffer, offset)
}

// Open file at path for read through open syscall. Return its descriptor.
func file_open(path string) (descriptor int, err error) {
	return syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC, 0)
}

// FILE_PERMISSIONS is what a made file take: readable by all, writable by owner. Kernel may
// still reduce it through process umask.
const FILE_PERMISSIONS nbio.File_Permissions = 0o644

// Make or truncate path for write through open syscall. Return its descriptor.
func file_create(path string) (descriptor int, err error) {
	return syscall.Open(
		path, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_TRUNC|syscall.O_CLOEXEC,
		uint32(nbio.File_Permissions_To_POSIX(FILE_PERMISSIONS)),
	)
}

// Report whether path exist, its portable mode, and its byte size, through lstat. Absent path is
// Exists false with nil error, thus caller tell "not there" apart from real stat failure. Whole
// kernel mode word is decoded, not three booleans: discarding permissions here is what left
// callers with no portable metadata to forward.
func file_status(path string) (status nbio.File_Status, err error) {
	metadata, stat_err := file_metadata_at(platform_current_directory(), path)
	if stat_err == syscall.ENOENT {
		return nbio.File_Status{}, nil
	}
	if stat_err != nil {
		return nbio.File_Status{}, stat_err
	}
	return nbio.File_Status{
		Exists: true,
		Mode:   nbio.File_Mode(nbio.File_Mode_From_POSIX(uint16(metadata.Mode))),
		Size:   metadata.Size,
	}, nil
}

// Read symbolic-link target without following it. Caller storage bounds kernel answer.
func file_read_link(path string, destination []byte) (count int, err error) {
	path_bytes := [OPERATING_SYSTEM_PATH_BYTES_MAXIMUM]byte{}
	path_pointer, path_err := file_path_pointer(path, path_bytes[:])
	if path_err != nil {
		return 0, path_err
	}
	read_count, _, errno := syscall.Syscall(
		PLATFORM_READ_LINK_CALL, uintptr(unsafe.Pointer(path_pointer)),
		uintptr(unsafe.Pointer(unsafe.SliceData(destination))), uintptr(len(destination)),
	)
	if errno == syscall.EINVAL {
		return 0, nbio.Not_Symbolic_Link
	}
	if errno != 0 {
		return 0, errno
	}
	return int(read_count), nil
}

// DIRECTORY_PERMISSIONS is what a made directory take: readable and traversable by all, writable
// by owner. Kernel may still reduce it through process umask.
const DIRECTORY_PERMISSIONS nbio.File_Permissions = 0o755

// Make path and every missing parent. Each component is made in turn, and existing directory
// converge, not fail, thus repeated call is not error. Walk is bounded by path length, and
// final Status check reject path whose last component exist as something other than directory —
// one case converging mkdir would hide.
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
		mkdir_err := syscall.Mkdir(
			path[:index], uint32(nbio.File_Permissions_To_POSIX(DIRECTORY_PERMISSIONS)),
		)
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
	if !nbio.File_Mode_Is_Directory(status.Mode) {
		return syscall.ENOTDIR
	}
	return nil
}

// Read one pass of directory raw entries into buffer and return children it name. Record layout
// is per-platform, thus walk cast to syscall.Dirent and let platform struct supply offsets
// rather than name byte position. Pass loop, descriptor lifetime, and accumulation across
// passes all compose above surface in Read_Directory.
func file_directory_pass(
	descriptor int, buffer []byte, entries []nbio.Directory_Entry,
) (entry_count int, err error) {
	buffer_size_maximum := len(entries) * DIRECTORY_RECORD_SIZE_MINIMUM
	if len(buffer) > buffer_size_maximum {
		buffer = buffer[:buffer_size_maximum]
	}
	count, read_err := platform_directory_read(descriptor, buffer)
	if read_err != nil {
		return 0, read_err
	}
	for offset := 0; offset < count; {
		record := (*syscall.Dirent)(unsafe.Pointer(&buffer[offset]))
		record_bytes := int(record.Reclen)
		aver.Always(record_bytes > 0, "A directory record spans at least one byte.")
		aver.Always(offset+record_bytes <= count,
			"A directory record ends inside the bytes the kernel returned.")
		entry, keep, entry_err := file_directory_entry(descriptor, record, record_bytes)
		if entry_err != nil {
			return 0, entry_err
		}
		if keep {
			aver.Always(entry_count < len(entries),
				"Caller-owned directory entry storage holds one complete pass.")
			entries[entry_count] = entry
			entry_count++
		}
		offset += record_bytes
	}
	return entry_count, nil
}

// Decode one directory record. keep is false for two self-referencing entries, and for record
// filesystem already deleted. That is what syscall.ParseDirent dropped before this walk
// replaced it.
func file_directory_entry(
	descriptor int, record *syscall.Dirent, record_bytes int,
) (entry nbio.Directory_Entry, keep bool, err error) {
	if platform_directory_absent(record) {
		return nbio.Directory_Entry{}, false, nil
	}
	name := file_directory_name(record, record_bytes)
	if name == "." {
		return nbio.Directory_Entry{}, false, nil
	}
	if name == ".." {
		return nbio.Directory_Entry{}, false, nil
	}
	directory_bit, kind_err := file_directory_kind(descriptor, record, name)
	if kind_err != nil {
		return nbio.Directory_Entry{}, false, kind_err
	}
	return nbio.Directory_Entry{Name: name, Is_Directory: directory_bit}, true, nil
}

// Return name a directory record hold. Kernel terminate name with NUL byte inside record, thus
// terminator bound it on both platform. Darwin also count name bytes in field of its own, but
// Linux does not, and terminator serve both.
func file_directory_name(record *syscall.Dirent, record_bytes int) (name string) {
	start := int(unsafe.Offsetof(record.Name))
	aver.Always(record_bytes > start, "A directory record holds at least one name byte.")
	bytes := unsafe.Slice((*byte)(unsafe.Pointer(&record.Name[0])), record_bytes-start)
	for index, value := range bytes {
		if value == 0 {
			return unsafe.String(unsafe.SliceData(bytes), index)
		}
	}
	return unsafe.String(unsafe.SliceData(bytes), len(bytes))
}

// Report whether directory record name directory. Kernel already wrote kind into record, thus
// common case cost no syscall at all. Only filesystem that leave field empty — XFS made with
// ftype=0, or NFS without READDIRPLUS — cost one fstatat.
func file_directory_kind(
	descriptor int, record *syscall.Dirent, name string,
) (directory_bit bool, err error) {
	if record.Type == syscall.DT_UNKNOWN {
		return file_status_at(descriptor, name)
	}
	return record.Type == syscall.DT_DIR, nil
}

// Report whether name, resolved against open directory, is itself directory. Go export no
// Fstatat on this platform, thus call go by trap number, same way Open_At and Mkdir_At do.
func file_status_at(directory int, name string) (directory_bit bool, err error) {
	metadata, metadata_err := file_metadata_at(directory, name)
	if metadata_err != nil {
		return false, metadata_err
	}
	return metadata.Mode&syscall.S_IFMT == syscall.S_IFDIR, nil
}

func file_metadata_at(directory int, path string) (metadata syscall.Stat_t, err error) {
	bytes := [OPERATING_SYSTEM_PATH_BYTES_MAXIMUM]byte{}
	pointer, convert_err := file_path_pointer(path, bytes[:])
	if convert_err != nil {
		return syscall.Stat_t{}, convert_err
	}
	_, _, errno := syscall.Syscall6(
		PLATFORM_STAT_AT_CALL, uintptr(directory), uintptr(unsafe.Pointer(pointer)),
		uintptr(unsafe.Pointer(&metadata)), PLATFORM_SYMBOLIC_LINK_NO_FOLLOW, 0, 0,
	)
	if errno != 0 {
		return syscall.Stat_t{}, errno
	}
	return metadata, nil
}

func file_path_pointer(path string, storage []byte) (pointer *byte, err error) {
	if len(path) >= len(storage) {
		return nil, syscall.ENAMETOOLONG
	}
	for index := 0; index < len(path); index++ {
		if path[index] == 0 {
			return nil, syscall.EINVAL
		}
		storage[index] = path[index]
	}
	return &storage[0], nil
}

// Read up to len(buffer) bytes from pipe. Pipe is not seekable, thus this is plain read, not
// pread read_at use. again is true when writer produced nothing yet, and zero count with no
// error is writer end closing.
func pipe_read(descriptor int, buffer []byte) (count int, again bool, err error) {
	count, err = syscall.Read(descriptor, buffer)
	if err != nil {
		return 0, socket_again(err), err
	}
	return count, false, nil
}

// Write up to len(buffer) bytes to pipe. again is true when pipe buffer is full and operation
// must stay armed.
func pipe_write(descriptor int, buffer []byte) (count int, again bool, err error) {
	count, err = syscall.Write(descriptor, buffer)
	if err != nil {
		return 0, socket_again(err), socket_send_translate(err)
	}
	return count, false, nil
}

// Make pipe and return its read and write ends. Neither end is configured: caller keep one end
// and hand other to child, and only kept end take close-on-exec and non-blocking mode. Two ends
// are separate open file descriptions, thus config of one does not change what child see on
// other.
func pipe_open() (read int, write int, err error) {
	descriptors := [PIPE_ENDS]int{}
	if pipe_err := syscall.Pipe(descriptors[:]); pipe_err != nil {
		return -1, -1, pipe_err
	}
	return descriptors[0], descriptors[1], nil
}

// Config end of pipe loop keep: close-on-exec, thus later spawn does not inherit it, and
// non-blocking, thus loop eager attempt report EAGAIN instead of wait.
func pipe_retain(descriptor int) (err error) {
	if flag_err := descriptor_close_on_exec(descriptor); flag_err != nil {
		return flag_err
	}
	return syscall.SetNonblock(descriptor, true)
}

// Set FD_CLOEXEC on descriptor. Both platform carry SYS_FCNTL, thus one raw call cover them.
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

// Resolve executable name same way shell do, because syscall.StartProcess require path while
// caller pass bare name such as sh or fc-cache. Name holding slash is used as given. This
// mirror os/exec own fallback: reject directory, then accept any execute bit, and let
// StartProcess report EACCES when mode bits promise more than file allow.
func executable_path(name string, search string) (path string, err error) {
	if name == "" {
		return "", syscall.ENOENT
	}
	if len(name) > EXECUTABLE_SEARCH_TEXT_BYTES_MAXIMUM {
		return "", syscall.ENAMETOOLONG
	}
	if len(search) > EXECUTABLE_SEARCH_TEXT_BYTES_MAXIMUM {
		return "", syscall.ENAMETOOLONG
	}
	for index := range name {
		if name[index] == '/' {
			return name, executable_check(name)
		}
	}
	start := 0
	var candidate_storage [filepath.PATH_SIZE_MAXIMUM]byte
	for position := 0; position <= len(search); position++ {
		if position < len(search) {
			if search[position] != ':' {
				continue
			}
		}
		directory := search[start:position]
		if directory == "" {
			directory = "."
		}
		if len(directory)+1+len(name) > len(candidate_storage) {
			start = position + 1
			continue
		}
		count := filepath.Join_Into(
			bytes.Slice(candidate_storage[:]),
			filepath.Elements{bytes.Text(directory), bytes.Text(name)},
		)
		// A copy, not a view: a view over this array moves the whole array to heap, and
		// syscall.Stat below copies the path again regardless, thus no path here is
		// allocation-free.
		candidate := string(candidate_storage[:count])
		if executable_check(candidate) == nil {
			return candidate, nil
		}
		start = position + 1
	}
	return "", syscall.ENOENT
}

// Report whether path name regular file carrying execute bit.
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

// Reap exited child. Return its exit code and resource accounting. WNOHANG return at once,
// because caller run this only after kernel reported exit.
func process_reap(identifier int) (exit int, usage nbio.Process_Usage, err error) {
	status := syscall.WaitStatus(0)
	rusage := syscall.Rusage{}
	reaped := false
	for retry_index := 0; retry_index < PROCESS_REAP_RETRIES_MAX; retry_index++ {
		_, wait_err := syscall.Wait4(identifier, &status, syscall.WNOHANG, &rusage)
		if wait_err == syscall.EINTR {
			continue
		}
		if wait_err != nil {
			return 0, nbio.Process_Usage{}, wait_err
		}
		reaped = true
		break
	}
	if !reaped {
		return 0, nbio.Process_Usage{}, syscall.EINTR
	}
	usage.CPU_User = time.Duration(
		rusage.Utime.Sec*1_000_000_000 + int64(rusage.Utime.Usec)*1_000)
	usage.CPU_System = time.Duration(
		rusage.Stime.Sec*1_000_000_000 + int64(rusage.Stime.Usec)*1_000)
	usage.RSS_Bytes_Max = process_rss_bytes(int64(rusage.Maxrss))
	return process_exit_code(status), usage, nil
}

// Report portable exit code: signalled child report 128 plus its signal, same as convention
// every shell use, and value os.ProcessState.ExitCode reported before.
func process_exit_code(status syscall.WaitStatus) (exit int) {
	if status.Signaled() {
		return 128 + int(status.Signal())
	}
	return status.ExitStatus()
}

// Report remote address of descriptor through getpeername. Non-IP peer yields zero address.
func socket_peer_address(
	descriptor int, destination *nbio.Address,
) (err error) {
	storage := [SOCKET_ADDRESS_BYTES]byte{}
	size := uint32(SOCKET_ADDRESS_BYTES)
	_, _, errno := syscall.RawSyscall(
		syscall.SYS_GETPEERNAME, uintptr(descriptor),
		uintptr(unsafe.Pointer(&storage[0])), uintptr(unsafe.Pointer(&size)),
	)
	if errno != 0 {
		return errno
	}
	decode_err := socket_address_decode(storage[:], size, destination)
	// Peer that is neither IPv4 nor IPv6 is not error here, only absent address.
	if decode_err == syscall.EAFNOSUPPORT {
		return socket_address_clear(destination)
	}
	if decode_err != nil {
		return decode_err
	}
	return nil
}

// Report whether err is non-blocking "try again" signal that keep operation armed, not complete
// it.
func socket_again(err error) (again bool) {
	if err == syscall.EAGAIN {
		return true
	}
	return err == syscall.EWOULDBLOCK
}

// Wire size of sockaddr_in, which bound bytes kernel read for IPv4 address.
const SOCKET_ADDRESS_IPV4_BYTES = 16

// Wire size of sockaddr_in6, which is also full storage size decode may receive.
const SOCKET_ADDRESS_IPV6_BYTES = 28

// Write address into storage as sockaddr and return byte count kernel read. Kernel take exactly
// these bytes, thus this replace syscall.Sockaddr: that interface cost allocation and type
// switch to build struct wrapper convert straight back to this same layout. Port and address
// bytes sit at same offsets on both platform, thus only two header bytes need platform.
func socket_address_encode(
	address nbio.Address, storage []byte,
) (size uint32, err error) {
	if len(storage) < SOCKET_ADDRESS_BYTES {
		return 0, syscall.EINVAL
	}
	for index := range storage {
		storage[index] = 0
	}
	if address.Family == nbio.FAMILY_IPV4 {
		platform_address_header(storage, syscall.AF_INET, SOCKET_ADDRESS_IPV4_BYTES)
		binary.Put_Uint_16(
			binary.Bytes(storage[2:4]), binary.Word_16(address.Port), binary.BIG_ENDIAN,
		)
		copy(storage[4:8], address.IP[:nbio.IPV4_ADDRESS_BYTES])
		return SOCKET_ADDRESS_IPV4_BYTES, nil
	}
	if address.Family == nbio.FAMILY_IPV6 {
		platform_address_header(storage, syscall.AF_INET6, SOCKET_ADDRESS_IPV6_BYTES)
		binary.Put_Uint_16(
			binary.Bytes(storage[2:4]), binary.Word_16(address.Port), binary.BIG_ENDIAN,
		)
		copy(storage[8:24], address.IP[:])
		return SOCKET_ADDRESS_IPV6_BYTES, nil
	}
	return 0, syscall.EAFNOSUPPORT
}

// Read sockaddr kernel wrote into storage. size is what kernel reported it filled, thus
// truncated answer fail, not decode whatever zeroed remainder happen to say.
func socket_address_decode(
	storage []byte, size uint32, destination *nbio.Address,
) (err error) {
	if len(storage) < SOCKET_ADDRESS_BYTES {
		return syscall.EINVAL
	}
	address_storage, address_err := socket_address_destination(destination)
	if address_err != nil {
		return address_err
	}
	family := platform_address_family(storage)
	port := uint16(binary.Uint_16(binary.Bytes(storage[2:4]), binary.BIG_ENDIAN))
	if family == syscall.AF_INET {
		if size < SOCKET_ADDRESS_IPV4_BYTES {
			return syscall.EINVAL
		}
		ip := address_storage[:nbio.IPV4_ADDRESS_BYTES:nbio.IPV4_ADDRESS_BYTES]
		copy(ip, storage[4:8])
		*destination = nbio.Address{
			Family: nbio.FAMILY_IPV4,
			IP:     address_storage[:nbio.IPV4_ADDRESS_BYTES:nbio.IPV6_ADDRESS_BYTES],
			Port:   port,
		}
		return nil
	}
	if family == syscall.AF_INET6 {
		if size < SOCKET_ADDRESS_IPV6_BYTES {
			return syscall.EINVAL
		}
		ip := address_storage[:nbio.IPV6_ADDRESS_BYTES:nbio.IPV6_ADDRESS_BYTES]
		copy(ip, storage[8:24])
		*destination = nbio.Address_IPV6(ip, port)
		return nil
	}
	return syscall.EAFNOSUPPORT
}

// Capacity, not current length, lets one destination receive IPv4 after IPv6 and vice versa.
func socket_address_destination(
	destination *nbio.Address,
) (storage []byte, err error) {
	if destination == nil {
		return nil, syscall.EINVAL
	}
	if cap(destination.IP) < nbio.IPV6_ADDRESS_BYTES {
		return nil, syscall.EINVAL
	}
	storage = destination.IP[:nbio.IPV6_ADDRESS_BYTES:nbio.IPV6_ADDRESS_BYTES]
	*destination = nbio.Address{IP: storage[:0:nbio.IPV6_ADDRESS_BYTES]}
	return storage, nil
}

func socket_address_clear(destination *nbio.Address) (err error) {
	_, err = socket_address_destination(destination)
	return err
}

// Map backend-independent family to platform socket family constant. Each platform syscall
// package carry its own value, thus one function serve both.
func socket_family(family nbio.Address_Family) (system int) {
	if family == nbio.FAMILY_IPV6 {
		return syscall.AF_INET6
	}
	return syscall.AF_INET
}

// Accept one pending connection on listener. Return non-blocking connected descriptor. again is
// true when none is ready yet.
func socket_accept(listener int) (descriptor int, again bool, err error) {
	result, _, errno := syscall.RawSyscall(syscall.SYS_ACCEPT, uintptr(listener), 0, 0)
	if errno != 0 {
		return -1, socket_again(errno), errno
	}
	descriptor = int(result)
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

// Socket_Connect_Start_Input keep caller-owned descriptor and typed address together.
type Socket_Connect_Start_Input struct {
	// Descriptor keep caller ownership during connect syscall.
	Descriptor int
	// Address stop DNS name from entry into syscall boundary.
	Address nbio.Address
}

// Begin connect of caller-owned descriptor. In-progress handshake complete later.
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

// Issue one connect. Every socket here is non-blocking, thus call return EINPROGRESS at once
// when handshake has further to go, and never wait. That is what let it go raw.
func socket_connect_call(descriptor int, address nbio.Address) (err error) {
	storage := [SOCKET_ADDRESS_BYTES]byte{}
	size, encode_err := socket_address_encode(address, storage[:])
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

// Return pending error on descriptor after connect complete, or nil when handshake succeeded.
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

// A distinct primitive per transport prevents a TCP profile from reaching a UDP descriptor.
func socket_open_tcp(
	family nbio.Address_Family, options nbio.TCP_Options,
) (descriptor int, err error) {
	descriptor, err = socket_open_tcp_raw(family)
	if err != nil {
		return -1, err
	}
	option_err := socket_tcp_options_set(descriptor, options)
	if option_err != nil {
		syscall.Close(descriptor)
		return -1, option_err
	}
	return descriptor, nil
}

// A distinct primitive per transport prevents a UDP profile from omitting its generic limits.
func socket_open_udp(
	family nbio.Address_Family, options nbio.UDP_Options,
) (descriptor int, err error) {
	descriptor, err = socket_open_udp_raw(family)
	if err != nil {
		return -1, err
	}
	option_err := socket_udp_options_set(descriptor, options)
	if option_err != nil {
		syscall.Close(descriptor)
		return -1, option_err
	}
	return descriptor, nil
}

// Apply all TCP limits before ownership can leave this package.
func socket_tcp_options_set(descriptor int, options nbio.TCP_Options) (err error) {
	generic_err := socket_options_set(
		descriptor,
		options.Receive_Buffer_Bytes,
		options.Send_Buffer_Bytes,
		options.Receive_Low_Water_Bytes,
		options.Linger_Timeout,
	)
	if generic_err != nil {
		return generic_err
	}
	if options.Maximum_Segment_Bytes > nbio.TCP_MAXIMUM_SEGMENT_BYTES_MAXIMUM {
		return syscall.EINVAL
	}
	if options.Maximum_Segment_Bytes == 0 {
		return syscall.EINVAL
	}
	maximum_segment := platform_tcp_maximum_segment_clamp(options.Maximum_Segment_Bytes)
	maximum_segment_err := syscall.SetsockoptInt(
		descriptor, syscall.IPPROTO_TCP, syscall.TCP_MAXSEG, int(maximum_segment))
	if maximum_segment_err != nil {
		return maximum_segment_err
	}
	if !socket_option_integer_valid(options.Not_Sent_Low_Water_Bytes) {
		return syscall.EINVAL
	}
	not_sent_err := syscall.SetsockoptInt(
		descriptor, syscall.IPPROTO_TCP, SOCKET_TCP_NOT_SENT_LOW_WATER,
		int(options.Not_Sent_Low_Water_Bytes))
	if not_sent_err != nil {
		return not_sent_err
	}
	keepalive_err := socket_keepalive_set(descriptor, options.Keepalive)
	if keepalive_err != nil {
		return keepalive_err
	}
	if !options.No_Delay {
		return nil
	}
	return syscall.SetsockoptInt(
		descriptor, syscall.IPPROTO_TCP, syscall.TCP_NODELAY, 1)
}

// Apply all UDP limits before ownership can leave this package.
func socket_udp_options_set(descriptor int, options nbio.UDP_Options) (err error) {
	return socket_options_set(
		descriptor,
		options.Receive_Buffer_Bytes,
		options.Send_Buffer_Bytes,
		options.Receive_Low_Water_Bytes,
		options.Linger_Timeout,
	)
}

// Apply all generic limits before ownership can leave this package.
func socket_options_set(
	descriptor int,
	receive_buffer_bytes uint32,
	send_buffer_bytes uint32,
	receive_low_water_bytes uint32,
	linger_timeout time.Duration,
) (err error) {
	if !socket_option_integer_valid(receive_buffer_bytes) {
		return syscall.EINVAL
	}
	receive_buffer_err := platform_receive_buffer_set(descriptor, int(receive_buffer_bytes))
	if receive_buffer_err != nil {
		return receive_buffer_err
	}
	if !socket_option_integer_valid(send_buffer_bytes) {
		return syscall.EINVAL
	}
	send_buffer_err := platform_send_buffer_set(descriptor, int(send_buffer_bytes))
	if send_buffer_err != nil {
		return send_buffer_err
	}
	if !socket_option_integer_valid(receive_low_water_bytes) {
		return syscall.EINVAL
	}
	receive_low_water_err := syscall.SetsockoptInt(
		descriptor, syscall.SOL_SOCKET, syscall.SO_RCVLOWAT,
		int(receive_low_water_bytes))
	if receive_low_water_err != nil {
		return receive_low_water_err
	}
	if !socket_linger_valid(linger_timeout) {
		return syscall.EINVAL
	}
	linger := syscall.Linger{
		Onoff: 1, Linger: int32(linger_timeout / time.SECOND),
	}
	linger_err := syscall.SetsockoptLinger(
		descriptor, syscall.SOL_SOCKET, syscall.SO_LINGER, &linger)
	if linger_err != nil {
		return linger_err
	}
	return nil
}

// The syscall takes a positive signed integer, thus zero cannot enable a limit.
func socket_option_integer_valid(value uint32) (valid bool) {
	return value > 0 && value <= nbio.SOCKET_OPTION_INTEGER_MAXIMUM
}

// Linger has a signed whole-second field on both supported kernels.
func socket_linger_valid(value time.Duration) (valid bool) {
	if value <= 0 {
		return false
	}
	if value%time.SECOND != 0 {
		return false
	}
	return value/time.SECOND <= time.Duration(nbio.SOCKET_OPTION_INTEGER_MAXIMUM)
}

// Direct ownership requires all keepalive limits before this function enables the option.
func socket_keepalive_set(descriptor int, keepalive nbio.TCP_Keepalive) (err error) {
	if !socket_keepalive_duration_valid(keepalive.Idle) {
		return syscall.EINVAL
	}
	if !socket_keepalive_duration_valid(keepalive.Interval) {
		return syscall.EINVAL
	}
	if keepalive.Probe_Count == 0 {
		return syscall.EINVAL
	}
	if keepalive.Probe_Count > nbio.TCP_KEEPALIVE_PROBE_COUNT_MAXIMUM {
		return syscall.EINVAL
	}
	enable_err := syscall.SetsockoptInt(
		descriptor, syscall.SOL_SOCKET, syscall.SO_KEEPALIVE, 1)
	if enable_err != nil {
		return enable_err
	}
	idle_err := platform_keepalive_idle_set(
		descriptor, int(keepalive.Idle/time.SECOND))
	if idle_err != nil {
		return idle_err
	}
	interval_err := platform_keepalive_interval_set(
		descriptor, int(keepalive.Interval/time.SECOND))
	if interval_err != nil {
		return interval_err
	}
	count_err := platform_keepalive_count_set(descriptor, int(keepalive.Probe_Count))
	if count_err != nil {
		return count_err
	}
	return nil
}

// The platform setters take signed whole seconds, so no rounding or sign change is permitted.
func socket_keepalive_duration_valid(value time.Duration) (valid bool) {
	if value <= 0 {
		return false
	}
	if value%time.SECOND != 0 {
		return false
	}
	return value/time.SECOND <= time.Duration(nbio.SOCKET_OPTION_INTEGER_MAXIMUM)
}

// Give socket its local address. Address reuse must precede bind, thus every bound socket has
// the same restart behavior and no caller can omit the option.
func socket_bind(descriptor int, address nbio.Address) (err error) {
	storage := [SOCKET_ADDRESS_BYTES]byte{}
	size, encode_err := socket_address_encode(address, storage[:])
	if encode_err != nil {
		return encode_err
	}
	reuse_err := syscall.SetsockoptInt(
		descriptor, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	if reuse_err != nil {
		return reuse_err
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

// Mark bound socket accepting, listen primitive.
func socket_listen_mark(descriptor int, backlog uint32) (err error) {
	if backlog == 0 {
		return syscall.EINVAL
	}
	return syscall.Listen(descriptor, int(backlog))
}

// Report socket own address, getsockname primitive. Call only read kernel state, thus it cannot
// block and go raw.
func socket_name(
	descriptor int, destination *nbio.Address,
) (err error) {
	storage := [SOCKET_ADDRESS_BYTES]byte{}
	size := uint32(SOCKET_ADDRESS_BYTES)
	_, _, errno := syscall.RawSyscall(
		syscall.SYS_GETSOCKNAME, uintptr(descriptor),
		uintptr(unsafe.Pointer(&storage[0])), uintptr(unsafe.Pointer(&size)),
	)
	if errno != 0 {
		return errno
	}
	return socket_address_decode(storage[:], size, destination)
}

// Translate filesystem errno to backend-independent sentinel. Darwin complete filesystem
// operation in its eager submit and never reach Linux result translation, thus mapping live here
// where both platform use it.
func file_translate(err error) (translated error) {
	if err == syscall.EEXIST {
		return nbio.Path_Exists
	}
	return err
}

// Translate platform-specific connect refusal to backend-independent sentinel.
func socket_connect_translate(err error) (translated error) {
	if err == syscall.ECONNREFUSED {
		return nbio.Connection_Refused
	}
	return err
}

// Read up to len(buffer) bytes from descriptor. again is true when no data is ready yet, and
// zero count with no error mark closed peer.
func socket_receive(descriptor int, buffer []byte) (count int, again bool, err error) {
	count, err = syscall.Read(descriptor, buffer)
	if err != nil {
		return 0, socket_again(err), err
	}
	return count, false, nil
}

// Write up to len(buffer) bytes to descriptor. again is true when kernel buffer is full and
// operation must stay armed.
func socket_send(descriptor int, buffer []byte) (count int, again bool, err error) {
	count, err = syscall.Write(descriptor, buffer)
	if err != nil {
		return 0, socket_again(err), err
	}
	return count, false, nil
}

// Shut down one or both connected-socket direction, synchronously.
func socket_shutdown(descriptor int, how nbio.Shutdown_How) (err error) {
	system_how := syscall.SHUT_RD
	if how == nbio.SHUTDOWN_SEND {
		system_how = syscall.SHUT_WR
	}
	if how == nbio.SHUTDOWN_BOTH {
		system_how = syscall.SHUT_RDWR
	}
	err = syscall.Shutdown(descriptor, system_how)
	if err == syscall.ENOTCONN {
		return nbio.Socket_Not_Connected
	}
	return socket_send_translate(err)
}

// Translate send-side broken-pipe error to portable result.
func socket_send_translate(err error) (translated error) {
	if err == syscall.EPIPE {
		return nbio.Broken_Pipe
	}
	return err
}

// Release descriptor.
func socket_close(descriptor int) (err error) {
	return syscall.Close(descriptor)
}
