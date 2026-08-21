
# Build

A Build is the caller-owned state one read of a build constraint runs through. It answers one
question: does this file stand in the build the caller names. The caller owns the name, the
constraint text, and the state, thus one read allocates nothing and no read reaches a file system.

# Target

A Target names the operating system, the architecture, and the tags one build states. The caller
writes them into the target it owns, thus this package states no default of its own and reads no
environment.

# Names

A name closing at a known operating system or a known architecture stands in that build alone, one
closing at both stands where both hold, and one closing at the word of a test file answers for the
name ahead of it. Everything ahead of the first underscore of the name says nothing at all.

# Constraints

A constraint states one truth of tags: a tag holds where the target names it, the exclamation
reverses one, the two signs join and meet them, and parentheses group them. The word unix holds on
every operating system Go calls one, and a tag the target never names holds nothing.

# Imports

Every import one file states names this module or the standard library; a path whose first element
holds a period names a host, thus a third party, and a file stating one stands in no build. A head
whose quotes or whose block never close states no truth, and one file states at most 256 imports.

# Systems

Nineteen operating systems and twenty-four architectures are known, which is the list the toolchain
itself states. A word this list never names is a tag like any other, thus a name closing at one
says nothing about the system it stands in.

# Directories

A Directory_Runner reads one directory through the storage the caller injects and keeps the names
of the Go files that stand in the build: the name states the systems, the constraint the file opens
with states the tags, and the imports it states name no third party.

# Refusals

A constraint the reader cannot read answers false and reports it: an empty text, a sign standing
where a tag belongs, a bracket that never closes, and a text past the widest one. A refused read
states no truth, thus a caller that reads one keeps the file out of every build.

# Bounds

One constraint spans at most 4,096 bytes, holds at most 64 tags, and nests at most 32 brackets. One
target holds at most 32 tags of its own.

# Allocation

Every read performs zero heap allocation. The caller owns the target, the state, and the text each
read stands on.
