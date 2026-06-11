package main


import "core:fmt"
import "core:io"
import "core:os"
import "core:sys/darwin"
import "core:sys/posix"
import "base:intrinsics"


main :: proc() {
        MAX_PENDING_CONNECTIONS :: 128 
        PF_INET :: 2
        SOCK_STREAM :: 1
        // <netinet/in.h>
        // You can also use zero to indicate the default protocol for any socket type.
        IPPROTO_TCP :: 6


        // Syscalls can either return status code (success/failure) or a semantically different value (file descriptor) on success.
        // tryv and syscallv propagate those values to the caller
        @(require_results)
        tryv :: proc(e: uintptr, loc := #caller_location) -> uintptr {
                if e == ~uintptr(0) {
                        err := posix.errno()
                        fmt.panicf("syscall failed: %v", err, loc = loc)
                }
                return e
        }
        try :: proc(e: uintptr, loc := #caller_location) {
                if err := os.Platform_Error(e); err != nil {
                        fmt.panicf("syscall failed: %v", err, loc = loc)
                }
                return
        }
        // Why proc groups? -> You can't use variadics here
        // Add more as you need.
        syscall :: proc {
                syscall2,
                syscall3,
                syscall4,
                syscall5,
        }
        @(require_results)
        syscallv :: proc {
                syscallv2,
                syscallv3,
        }
        dscn :: darwin.System_Call_Number
        isc  :: intrinsics.syscall
        syscall2  :: proc(sc: dscn, a, b:          uintptr, loc := #caller_location) { try(isc(darwin.unix_offset_syscall(sc), a, b          ), loc) }
        syscall3  :: proc(sc: dscn, a, b, c:       uintptr, loc := #caller_location) { try(isc(darwin.unix_offset_syscall(sc), a, b, c       ), loc) }
        syscall4  :: proc(sc: dscn, a, b, c, d:    uintptr, loc := #caller_location) { try(isc(darwin.unix_offset_syscall(sc), a, b, c, d    ), loc) }
        syscall5  :: proc(sc: dscn, a, b, c, d, e: uintptr, loc := #caller_location) { try(isc(darwin.unix_offset_syscall(sc), a, b, c, d, e ), loc) }
        syscallv2 :: proc(sc: dscn, a, b:    uintptr, loc := #caller_location) -> uintptr { return tryv(isc(darwin.unix_offset_syscall(sc), a, b   ), loc) }
        syscallv3 :: proc(sc: dscn, a, b, c: uintptr, loc := #caller_location) -> uintptr { return tryv(isc(darwin.unix_offset_syscall(sc), a, b, c), loc) }


        socket := syscallv(.socket, PF_INET, SOCK_STREAM, IPPROTO_TCP)
        address := posix.sockaddr_in{
                sin_len    = size_of(posix.sockaddr_in),
                sin_family = .INET,
                sin_port   = 0, // let the OS pick
        }
        if err := posix.inet_pton(.INET, "127.0.0.1", rawptr(&address.sin_addr), size_of(address)); err != .SUCCESS {
                fmt.panicf("%w", err)
        }


        SOL_SOCKET, SO_REUSEADDR :: 0xffff, 0x0004
        syscall(.setsockopt,  socket, SOL_SOCKET, SO_REUSEADDR, uintptr(new(i32)), size_of(rawptr))
        syscall(.bind,        socket, uintptr(&address), size_of(address))
        addr_len: uint = size_of(address) // annoying as FUUUUUUUU
        syscall(.getsockname, socket, uintptr(&address), uintptr(&addr_len)) 
        syscall(.listen,      socket, MAX_PENDING_CONNECTIONS)


        fmt.printfln("listening on 127.0.0.1:%d", address.sin_port)
        KiB :: 1024
        buf: [KiB * 16]byte
        client := os.Handle(syscallv(.accept, socket, 0, 0))
        stream := os.stream_from_handle(client)
        defer os.close(client)
        for {
                n, err := io.read(stream, buf[:])
                if err != nil {
                        return
                }
                fmt.wprint(stream, "You said:", string(buf[:n]))
        }
}
