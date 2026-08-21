
# Clean

Clean removes repeated slashes, dot elements, and cancellable dot-dot elements. Root stops dot-dot
backtracking. Empty result becomes one dot. Only root keeps trailing slash.

# Components

Split returns directory and file views around final slash. Extension returns suffix from final dot
in final element. Base returns final element after trailing slash removal. Directory cleans bytes
before final element. Is_Absolute reports leading slash.

# Join

Join ignores empty elements, inserts slash between retained elements, then applies Clean.

# Match

Match implements Go path shell grammar. Star never crosses slash. Question mark consumes one UTF-8
character except slash. Classes support negation, escaped members, and inclusive ranges. Malformed
patterns return Error_Bad_Pattern.

# Bounds

Darwin path holds at most 1,023 bytes; Linux path holds at most 4,095 bytes. Each limit leaves one
PATH_MAX byte for kernel NUL. Join accepts at most path limit plus one elements. Oversized malicious
input or short destination panics before partial output escapes.

# Allocation

Every exported operation performs zero heap allocation. Clean, Join, and Directory write caller
storage and return byte count. Other operations return scalars or input views.
