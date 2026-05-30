# odin_http_server_from_scratch

Implement an http server from scratch using syscalls. Written in Odin and the implementation is specific to MacOS. Avoid using the Odin stdlib syscall wrappers.

## Quickstart

```
bin/darwin/lua bootstrap.lua

bin/darwin/lua odin run src/tcp.odin -file
nc -E 127.0.0.1:<port>

bin/darwin/lua odin run src/http.odin -file
curl --http1.0 --include 127.0.0.1:<port>
curl --http1.0 --include 127.0.0.1:<port>/wazoo
curl --http1.0 --include 127.0.0.1:<port>/my-super-secret-page
# Lastly, visit in your browser: 127.0.0.1:<port>/my-super-secret-page

```

## List libc macro definitions (MacOS):

`$ cd "$(xcrun --show-sdk-path)/usr/include/`

```
clang -E -dM "$(xcrun --show-sdk-path)/usr/include/sys/socket.h" | $EDITOR
clang -E -dM "$(xcrun --show-sdk-path)/usr/include/sys/types.h"  | $EDITOR
clang -E -dM "$(xcrun --show-sdk-path)/usr/include/netinet/in.h" | $EDITOR
```

## Syscall stdlib wrappers:

```
core:net
core:sys/darwin/sys_socket.odin
core:sys/posix/*
```

