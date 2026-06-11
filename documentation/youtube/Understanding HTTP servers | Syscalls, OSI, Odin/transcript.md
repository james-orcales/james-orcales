# Building an HTTP Server from Scratch in Odin

Okay, today we'll be building an HTTP server from scratch. It's written in a low-level language,
Odin, which has manual memory management. We'll discuss the OSI model, operating systems, and how
that relates to the HTTP protocol.

First, I'll show you our HTTP server and run it. It's running on localhost, listening on port 49185.
Let's go to Safari and paste that in. Hello. If we try a different path — `wazoo` — page not found.
If we try my super secret page, we get the Lua 5.1 manual. I'll show you the logs later. For now,
let's discuss the OSI model before we go into the implementation. Actually, we'll also discuss
operating systems.

## The OSI model

The OSI model is a seven-layer framework that describes how data moves throughout the network. You
have the physical layer, the data link layer, the network layer, the transport layer, the session
layer, the presentation layer, and the application layer.

I'm not going to assume you have absolutely no idea about the OSI model — I'm just going to give you
a general understanding of how it relates to HTTP servers. I created a diagram here. I'm too lazy to
create a PowerPoint presentation, and I'm too lazy to do editing, so we're going old school. I
mapped the five W's — well, it's the four W's and "how" — to the seven layers. (How am I going to
hold this? I probably should have planned this better.)

### Physical layer — "What?"

It answers the question: what? What is our data? When it comes to software, we're talking about
digital information, but we need to store that information in a physical medium. So what is our data
in that physical medium — is it voltages, radio waves, or light pulses? The physical layer deals
with the NIC; when it comes to the hardware, that's the physical medium. NIC — what does that mean?
That's your network interface card. It could be your Ethernet port or your Wi-Fi card.

### Data link layer — "Who?"

This answers: who? This deals with MAC addresses. So what are MAC addresses? Your NIC is given a
globally unique identifier by your manufacturer — that's the MAC address. The point is that, in your
local area network, your device can be identified. So for example, your Wi-Fi router: if you go to
the administrator page, you can see the MAC addresses of all devices connected to it — which is
pretty funny, because that way you can find out if a neighbor is riding along on your Wi-Fi
connection. And that answers the question of who is connected to the local area network.

### Network layer — "Where?"

Where is the data going? This deals with IP addresses — Internet Protocol. This is similar to the
data link layer, but the difference is the scope of it, and the fact that it's logical — it's pure
software. So MAC addresses are part of the hardware; they're assigned by the manufacturer. IP
addresses are logical addresses, and what this means is they enable hierarchical routing — and this
allows it to scale globally to what we call the internet.

When I say hierarchical routing, that means your router and other routers only need to know routes
to other routers. Whereas if we relied solely on MAC addresses to identify everything, then each
router would need to take into account a million — a billion — devices in the world. So in other
words, IP addresses abstract away other networks, and they allow us to route data to other networks.
Whereas MAC addresses — this is local only.

### Transport layer — "How?"

The transport layer answers: how is the data delivered? This deals with TCP — Transmission Control
Protocol. This ensures reliable and in-order delivery of data between two parties. So if the network
layer deals with where the data is going, the transport layer deals with how the data is going to
get there, to its destination.

### The network stack vs. the user-space stack

These four layers are what we call the network stack in the OSI model. The top three are the
user-space stack, and everything there is built on top of the transport layer.

In our implementation — what I mean by implementing an HTTP server from scratch — the lowest that
we'll go is the TCP server. Because technically, if we're being vague, you could also implement an
HTTP server from scratch starting with creating Wi-Fi cards. So we'll only create, first, a TCP
server, and then we'll build an HTTP server on top of that.

So what do I mean by the user-space stack building on top of the TCP/transport layer? The transport
layer is how we send raw data — but that means nothing through the wire; that's just bits and bytes,
it has no meaning. So what we build on top of that is what gives this data meaning.

### Session layer

First of all, session — that's where HTTP comes in, where you'd have long-lived connections. Now,
this is a TCP server, so there are also connections here, right? So why wouldn't this be in the
session layer? The difference is that the session layer gives semantic context to the long-lived
connection. So for example, you're in a multiplayer video game lobby. Are you in the loading screen,
in the champ-select screen, in the match itself, or in the final result? You can communicate that
using HTTP. Whereas with TCP, all you can really tell is: do you have a connection with another host
on some other network? And then you'd also have TLS and SSH here — this is for HTTPS, encrypted
connections, and SSH.

### Presentation layer

This deals with how the data is formatted. This covers encoding, compression, and also encryption.
So this would be your tar archives, your JSON files, your protobufs, your Huffman encodings. And
it's important to know that, in this diagram, there's a clear separation between the layers, but in
implementation the applications would do a little bit of multiple layers. So for example, SSH or TLS
covers both presentation and session. And for NIC cards, they can do some work for the data link
layer on the firmware itself.

### Application layer

The application layer is the topmost layer. This is where we refer to the DNS servers or the
browsers. So your browser is just making the HTTP requests for you. When you go to a website, it
goes way down to the transport layer, opens a TCP socket to that server, creates the HTTP message,
and then communicates. DNS is what gives websites their human-readable output. So let's go back to
the website here — you see, I connected to my website through an IP address, and then there's a
port. That's how servers work. But for your user-facing website, you'd have a user-readable name
like Google. So that's the job of DNS servers: they give these IP addresses names.

(Okay, I'm going to cut that out — I just switched to the terminal and it accidentally showed you
the logs for the website. Let's go back.)

## A closer look at IP and TCP

I actually forgot to mention something — actually, two things. I want to discuss more about IP and
TCP.

Like I said, TCP sits on top of IP. The IP/network layer deals with sending data where it needs to
go. It doesn't care about whether that data gets there successfully, or whether its integrity is
preserved — that's the primary responsibility of the transport layer. When data packets leave this
IP address (this host) and reach their destination (another host on another network, a.k.a. another
IP address), TCP takes over.

### Ports

Now, since this is another host, multiple processes/applications can listen on it, so you need some
way to differentiate those multiple applications. That's where ports come in. So port 49163
represents my server; another port would represent your Postgres server; another would represent
some other service; and there are going to be ports dedicated to HTTP connections, TCP, yada yada
yada. This combination of the IP address and the port is a unique identifier, and that's what
identifies the process that listens on this host when data arrives at that destination and initiates
a connection.

### The TCP handshake

There's this thing that happens — it's called the TCP handshake. Here's the client; here's the
server. The client sends an initial message — a synchronization message: **SYN**. The server must
reply with **SYN-ACK** (synchronize-acknowledge). Finally, the client must respond with a final
**ACK**. Now, if they don't receive any of these messages successfully, then we don't have a
successful connection.

So that's the first part of TCP: ensuring there's a reliable connection between two parties. Then,
for ensuring the integrity of data, it does checksums on the data — and multiple layers of the stack
    do checksums, so there's continuous verification of the integrity throughout the stack. And to
        ensure that data is received in order, TCP uses sequence numbers on the packets, so that
        when packets come out of order — and they will — TCP can reconstruct the message again. That
        covers the OSI model.

## Operating systems and syscalls

Now let's move on to operating systems. I've mentioned syscalls, so let's talk about what they are
and what their role is in operating systems. I'll have to draw another diagram.

An operating system abstracts away the hardware. It virtualizes it to logical counterparts, so that
multiple processes can share this single device and they only have to worry about their single piece
of the pie. When I say virtualizing the hardware, I mean virtual memory — so that each process only
has its own address space. That's why, in a low-level language with manual memory management like C,
when you go out of bounds in your array, you're accessing memory outside of your assigned address
space — and that's why you get a segmentation fault.

### The kernel

I drew a diagram here to represent the stack. You have the hardware, you have your kernel (the
operating system), and then the user space. But the kernel is part of the operating system; it's the
core part, actually, and it's what directly interfaces with the hardware.

When I say hardware, I mean your actual computer: your NIC, your keyboard, your hard drive, your
CPU, your RAM. OS — that's going to be Linux, macOS, Windows. User applications: your web browser,
your photo editor, your video editor, your screen recorder, your video games. Technically, you can
consider a large part of the operating system as user-space applications too, like CLI utilities,
but you consider them part of the operating system.

Now, the kernel, since it's the one that directly interfaces with the hardware, is the most
privileged part of your system. I'll give you a fun fact, if you're not updated: there's a huge
controversy in online video games because they use kernel-level anti-cheat. Looking at you,
Valorant/Riot. Riot uses an anti-cheat that lives in the kernel, so it can monitor all activity on
your hardware. You don't have to be a software developer to understand how crazy that is, just from
a privacy perspective. But if you're a software developer, you understand that even without the
privacy concerns, that's a crazy thing to do — because you're taking code from a video game company
and putting it in the kernel. And all software has bugs; their game is full of them. I don't want
that, man. Anyway, rant over.

### Syscalls

When a user application does I/O operations — like listening for user input from the keyboard, or
listening to a socket, interacting with the NIC — it needs to talk to the hardware, a.k.a. through
the kernel. To formalize this process, the kernel exposes OS primitives called syscalls. Syscalls
are how a user-space application can drop down to kernel-level privileges and specifically request
certain services from it.

To create our TCP server, we need to use a couple of syscalls: `socket`, `bind`, `listen`, and
`accept`. `socket` is what creates the socket — the logical socket. Then you bind it to an IP
address plus a port, and then you start listening for connections there. And then, once a new
connection comes in, you need to accept it. `accept` returns a socket descriptor, and then you treat
that like a normal file descriptor — you can read and write data to it. And that's how you do
communication using TCP sockets, a.k.a. TCP servers.

Okay, I think you have enough context now on the whole network stack, so we can go to the
implementation.

## Implementing a TCP server

Switch — go to Ghostty and `vim tcp.odin`. Let's start. So here, first I just have some wrapper
functions, because I'm using the `intrinsics` package instead of the core library here in Odin,
since we're using the syscalls directly.

```odin
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
```

### The socket syscall

Our first syscall is going to be the `socket` syscall. You can create different types of sockets
here — like I mentioned earlier, there's UDP and Unix sockets. It takes three arguments, and I
hardcoded the arguments here. So, two questions: how did you figure out what arguments this call
takes, and how do you know the values to put there?

Well, we can read the man page for `socket`, and you'll see that it takes three arguments — domain,
type, protocol — and they're all integers. When you read the description, it indicates the values
that are valid there. For domain, these are the values. And what did I use? I used `PF_INET`, and I
set it to 2.

```odin
PF_INET :: 2
SOCK_STREAM :: 1
// <netinet/in.h>
// You can also use zero to indicate the default protocol for any socket type.
IPPROTO_TCP :: 6

socket := syscallv(.socket, PF_INET, SOCK_STREAM, IPPROTO_TCP)
```

### Finding the right constants

So how did I figure out that `PF_INET` is 2? You need to reference your system libs. I have a
command here. I'm on Darwin, so mine is going to be quite different from Linux. We go here — this
outputs all the macro definitions in that specific header file of libc. Let's search for `PF_INET`.
Okay, it's defined as `AF_INET`. So let's search for `AF_INET` now. Okay, `AF_INET` is set to 2. And
that's how we get the value of 2 for `PF_INET`. Then you do the same thing for all the other values.

### `bind` and the address struct

Moving on, you'll figure out that you need to do `bind`, so you can bind your IP address to the
socket. If you look at the man page for `bind`, here you go: it takes in an integer, which is the
socket descriptor. (Oh — I forgot to show you that these man pages also show you the return value.
So the `socket` syscall returns -1 if an error occurs, and a descriptor on success.)

Going back to `bind`: it's a bit claustrophobic. It takes in a `sockaddr` as a second argument. And
what I actually use here is `sockaddr_in` — I'll explain why. So if we go back here — instead of
`socket.h`, we need to go to `types.h`, and search for `sockaddr`, the type. Here you go, I found
it.

So the `sockaddr` struct takes in a 14-byte character array as its third field. This tells you that
it's actually just a generic struct, and that different implementations would have a more specific
struct or type that can break down this 14-byte character array into separate fields. And this is
something you just have to know — maybe you can ask ChatGPT. That's what I did. Instead, you'll use
the specific struct, `sockaddr_in` — this is in `netinet/in.h`. And this is the actual type of
struct that you need to pass into `bind`.

That's why, here in `tcp.odin`, I started creating the address after the socket. Here I just used
the standard library — I couldn't be bothered to actually create the struct using the C types in
Odin. And to finish the initialization of this struct, you need to use this function, `inet_pton` —
presentation-to-number. It converts your string representation of the IP address into its integer
equivalent. You see here, there's the `sin_addr` field of the address struct, and it mutates this
through an out-parameter.

```odin
address := posix.sockaddr_in{
	sin_len    = size_of(posix.sockaddr_in),
	sin_family = .INET,
	sin_port   = 0, // let the OS pick
}
if err := posix.inet_pton(.INET, "127.0.0.1", rawptr(&address.sin_addr), size_of(address)); err != .SUCCESS {
	fmt.panicf("%w", err)
}
```

### Reusing the address, and `getsockname`

Next, we'll bind the socket to the address we just created. And you'll see that before I bound it, I
set an option on the socket descriptor to reuse the address. What this does is: now, when we
immediately restart our server after shutting it down, we can use the address again, instead of
there being a 10- or 15-second delay — which would be annoying for local development.

And then you get the socket name. Earlier I had the port set to 0, which lets the OS pick. If you do
this, you need to do `getsockname`, so the address can be filled in with the proper port that the OS
chose, so you can print it later on.

```odin
SOL_SOCKET, SO_REUSEADDR :: 0xffff, 0x0004
syscall(.setsockopt,  socket, SOL_SOCKET, SO_REUSEADDR, uintptr(new(i32)), size_of(rawptr))
syscall(.bind,        socket, uintptr(&address), size_of(address))
addr_len: uint = size_of(address) // annoying as FUUUUUUUU
syscall(.getsockname, socket, uintptr(&address), uintptr(&addr_len))
```

### `listen` and `accept`

And then you listen. This is going to start accepting connections in the background — so this is
multi-threaded by default. And then, to actually get the socket descriptor for individual
connections, you'd have to do the `accept` syscall here. Then we convert it into an OS handle, and
that's our file descriptor (a.k.a. socket descriptor). And then we just interact with it normally
using the I/O library here. And you also have to remember to close it. And if you're handling
multiple connections, you'd have to do this `accept` in a for loop as well, and then do non-blocking
I/O.

```odin
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
```

### Demo with netcat

So let's run this — actually, this is the first time I'm running the TCP file. And then we create a
client. So `nc` — that's netcat. `nc --help`... this is way too much. Okay, `nc` means netcat. We
connect to our server on port 6969, and now everything I type here will get sent to our server. So
`arsc` — you sent `arsc`, what did we do? We read the data and then printed it out. "Hello. Can you
hear me?" Okay, that's enough of the demo.

## Implementing the HTTP server

Now let's go on to the HTTP server. The HTTP server is just a format built on top of TCP. So you see
that TCP is just sending the raw data — it's up to us to give structure or semantic meaning to that
data. So here we need to start creating HTTP messages.

### The HTTP message format

Now we need to go back to Safari and reference the HTTP spec. Here you go, HTTP 1.0. Let's go to
HTTP message. There you go. This is the format that we need to follow if you want to create a valid
HTTP message. So there's a request line, then your headers, then the body. And these are all
separated by a CRLF — a carriage-return line-feed, which is these two special escape bytes, `\r\n`.

And then you just read on through this. First you'd have the request line: you'd have the method and
then the request URI. And you really just follow this to the T. You can read my code for the
implementation.

```odin
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
```

### Parsing the request

Let's go back to the code. See here, we take in the data for the message from the client, we read
it, and then we start parsing it. If it's a GET request, we do this — we switch on the request URI.
There you go. Where did I start parsing it? Here we go: `http_parse_request`. If we go to this
function, `http_parse_request`, there you go — I start parsing the HTTP message. I'm not going to go
through all of this, because it's really just about reading the spec and reading the code.

```odin
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
```

### Building the response

And then, to reply, we just create a valid HTTP message too. So, going back to Safari to create a
response: it has a similar shape to the request message. There you go. And I just hardcoded it here.
You create your response line — "Not Found" is the phrase — and then Content-Type, Content-Length,
and then here is your actual body, separated by two carriage-return line-feeds.

So you saw earlier that when I went to my super secret page, we loaded the Lua documentation. So
here, I have the Lua file — the Lua repository, vendored — and I just loaded that HTML file at
compile time, and then wrote that out to the socket descriptor.

```odin
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
```

So again, when we do `odin run http.odin`, let's first try curl. Let's curl it: `127.0.0.1:49200`.
Hello. If we try `wazoo`, page not found — you see, if it's not an actual URI that I support, I
return page not found. And then my super secret page — you see? So it's just literal bytes that
we're sending to each other, and the web browser's job is to render that into a GUI.

### HTTP 1.0 vs 1.1

So you see here we're getting warnings that the client is using HTTP 1.1. I have notes here, let's
check them again. curl... wait, I didn't put it here. Well, anyway, there's a way to make this use
HTTP 1.0. Let's see if I can remember it. `--http1.0` — here we go. Okay, we passed the 1.0 flag,
and now we're not getting the warning. See here, I have an `http_warn`: if the version of the
request is not 1.0, we log.

So now when we go to the browser, Safari, and we go to this again — and what was the port? It's
49200, easy to remember. And then we go to my super secret page. I'm not going to type this out...
Oh, what? Oh, it's because of my extra slash. There you go. If you look at the logs — yeah, we got a
bunch of warnings, because the web browser always uses HTTP 1.1.

### Path cleaning

And I want you to notice — let's go back to Safari — when we had double slashes here, it did not
clean the path automatically so that it just becomes one slash. Nope. So it becomes a page not
found. So that's one of the things your HTTP frameworks do under the hood: they clean the path —
they turn consecutive slashes into a single slash.

## Wrapping up

So that is how to create an HTTP server from scratch. It's only 200 lines of code. And if you want
to do a TCP server from scratch, it's only 90 lines of code. The link to the repository is in the
description. I hope you learned something today. That's it. Oh, and don't forget to drink water.
