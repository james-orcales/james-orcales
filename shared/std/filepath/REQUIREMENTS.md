
# Clean

Clean uses host path separator while preserving standard lexical dot and dot-dot rules. Unix host
uses slash. Caller owns written result.

# Local Paths

Is_Local accepts nonempty relative paths that Clean cannot move above evaluation root. Localize
accepts valid UTF-8 slash-separated filesystem paths and writes host form. Invalid path returns
Error_Invalid_Path.

# Slash Conversion

To_Slash and From_Slash write caller storage. Unix host copies unchanged bytes.

# Components

Split_List writes caller path slots around host list separator. Split, Extension, Base, Directory,
and Volume_Name preserve standard host component behavior. Unix Volume_Name stays empty.

# Join

Join inserts host separator between elements and cleans result.

# Absolute And Relative

Absolute receives working directory explicitly, so ambient process state stays injected. Relative
writes path from cleaned base to cleaned target. Incompatible roots return stable errors.

# Match

Match uses standard shell grammar and host separator. Malformed pattern returns Error_Bad_Pattern.

# Glob

Glob composes injected no-follow status and asynchronous directory reads. Caller owns candidate,
entry, and record storage. The runner queues continuation but never owns or drives IO.

# Walk

Walk and Walk_Directory visit lexically sorted paths in preorder through injected storage.
Symbolic links are visited but never followed. Skip_Directory and Skip_All preserve standard
control behavior. Caller owns queue, child, entry, and record storage.

# Symbolic Links

Eval_Symlinks_Into resolves relative, absolute, and chained links through injected no-follow
status and Read_Link. Caller owns destination and both scratch buffers. Link traversal stops after
255 links so a malicious cycle stays bounded.

# Bounds

Darwin path holds at most 1,023 bytes; Linux path holds at most 4,095. Kernel NUL stays outside.
Path lists and filesystem traversal collections use separate bounds.
Overflow leaks no partial output. Directory record storage uses injected nbio block bound.

# Allocation

Every exported operation performs zero heap allocation. Written forms use caller storage. Other
forms return scalars or input views.
