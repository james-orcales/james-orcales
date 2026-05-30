package main


import "core:fmt"
import "core:io"
import "core:os"
import "core:bytes"
import "core:strconv"
import "core:reflect"
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
        if err := posix.inet_pton(.INET, "127.0.0.1", rawptr(&address.sin_addr)); err != .SUCCESS {
                fmt.panicf("%w", err)
        }


        SOL_SOCKET, SO_REUSEADDR :: 0xffff, 0x0004
        syscall(.setsockopt,  socket, SOL_SOCKET, SO_REUSEADDR, uintptr(new(i32)), size_of(rawptr))
        syscall(.bind,        socket, uintptr(&address), size_of(address))
        addr_len: uint = size_of(address) // annoying as FUUUUUUUU
        syscall(.getsockname, socket, uintptr(&address), uintptr(&addr_len)) 
        syscall(.listen,      socket, MAX_PENDING_CONNECTIONS)


        fmt.printfln("listening on 127.0.0.1:%d", address.sin_port)
        buf: [KiB * 16]byte
        for {
                client := os.Handle(syscallv(.accept, socket, 0, 0))
                stream := os.stream_from_handle(client)
                defer os.close(client)


                n, err := io.read(stream, buf[:])
                if err != nil {
                        return
                }
                req := http_parse_request(buf[:n])


                fmt.printf("%s %q\n", req.method, req.uri)
                #partial switch req.method {
                case .GET:
                        switch string(req.uri) {
                        case:
                                fmt.wprint(stream, "HTTP/1.0 404 Not Found\r\nContent-Type: text/html\r\nContent-Length: 16\r\n\r\nPage not found!\n")
                        case "/":
                                fmt.wprintf(stream, "HTTP/1.0 200 OK\r\nContent-Type: text/html\r\nContent-Length: 7\r\n\r\nHello!\n")
                        case "/my-super-secret-page":
                                lua_doc := #load("../vendor/lua-5.1.5/doc/manual.html")
                                fmt.wprintf(stream, "HTTP/1.0 200 OK\r\nContent-Type: text/html\r\nContent-Length: %d\r\n\r\n%s", len(lua_doc), lua_doc)
                        }
                case .HEAD:
                        switch string(req.uri) {
                        case:
                                fmt.wprint(stream, "HTTP/1.0 404 Not Found\r\nContent-Type: text/html\r\nContent-Length: 16\r\n\r\n")
                        case "/":
                                fmt.wprintf(stream, "HTTP/1.0 200 OK\r\nContent-Type: text/html\r\nContent-Length: 7\r\n\r\n")
                        case "/my-super-secret-page":
                                lua_doc := #load("../vendor/lua-5.1.5/doc/manual.html")
                                fmt.wprintf(stream, "HTTP/1.0 200 OK\r\nContent-Type: text/html\r\nContent-Length: %d\r\n\r\n", len(lua_doc))
                        }
                case:
                                fmt.wprint(stream, "HTTP/1.0 501 Not Implemented\r\n\r\n")
                                continue
                }
        }
}


CRLF  :: []byte{'\r', '\n'}
DOUBLE_CRLF :: []byte{'\r', '\n', '\r', '\n'}
KiB :: 1024
http_parse_request :: proc(msg: []byte) -> (req: http_Request) {
        defer {
                if req.version != "HTTP/1.0" {
                        fmt.println("WARNING: unsupported http version", string(req.version[:]))
                }
        }
        head := msg[:bytes.index(msg, DOUBLE_CRLF) + len(CRLF)]
        body := len(msg) > len(head) + len(CRLF) ? msg[len(head)+len(CRLF):] : nil


        http_parse_token :: proc(buf: ^[]byte, delimiter_substr: ..byte) -> (token: []byte) {
                if len(buf) == 0 || len(delimiter_substr) == 0 {
                        return nil
                }
                right := bytes.index(buf^, delimiter_substr)
                if right == -1 {
                        return buf^
                } 
                token = buf[:right]
                buf^ = buf[right+len(delimiter_substr):]
                return token
        }


        req.method = reflect.enum_from_name(http_Method, string(http_parse_token(&head, ' '))) or_else panic("unsupported method")
        req.uri    = http_parse_token(&head, ' ')
        copy(req.version[:], http_parse_token(&head, ..CRLF))


        for {
                key := http_parse_token(&head, ':', ' ')
                val := http_parse_token(&head, ..CRLF)
                for _, i in key {
                        switch key[i] {
                        case:           key[i] = key[i]
                        case 'A'..='Z': key[i] = key[i] + ('a' - 'A')
                        }
                }
                req.headers[string(key)] = val
                if len(head) > 0 {
                } else {
                        break
                }
        }


        if len(body) > 0 {
                req.body = body[:(strconv.atoi(string(req.headers["content-length"])))]
        }
        return req
}
http_dump_request :: proc(req: http_Request) {
        req := req
        fmt.println("HTTP Request:")
        fmt.println("\tMethod:  ", req.method)
        fmt.println("\tURI:     ", string(req.uri))
        fmt.println("\tVersion: ", string(req.version[:]))


        fmt.println("\tHeaders:")
        for key, val in req.headers {
                fmt.println("\t\t", key, ": ", string(val))
        }


        if len(req.body) > 0 {
                fmt.println("\tBody: ", string(req.body))
        } else {
                fmt.println("\tBody: <empty>")
        }
}
http_Request :: struct {
        // request line
        method:  http_Method,
        uri:     []byte                `fmt:"q,n"`,
        version: [len("HTTP/1.0")]byte `fmt:"q,n"`,
        headers: map[string][]byte     `fmt:"q,n"`,
        body: []byte                   `fmt:"q,n"`,
        // There are many reasons to track the length separately from content-length. It doesn't matter for us.
        // body_len: int
}
// Those prefixed with underscores are unsupported.
http_Method :: enum {
        GET,
        HEAD,
        _POST,
        _PUT,
        _DELETE,
        _CONNECT,
        _OPTIONS,
        _TRACE,
        _PATCH,
}
