
# Dispatch

### Unknown Verb

An unrecognized invoking-link name writes a usage error and returns the usage
exit code.

# Wire

### Round Trip

Encoding a value and decoding the result reproduces the same value, so the
binary format flowing between sibling verbs is lossless.

# Json

### Round Trip

Parsing a JSON document and emitting the result reproduces the same compact
text, preserving object key order.
