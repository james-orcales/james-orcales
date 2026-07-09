
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

# Csv

### Round Trip

Parsing CSV and emitting the result reproduces the same text; the header row
names the columns and each later row becomes a record.

# Verbs

### Sort By

sort-by orders a list of records by a field, stably, comparing numbers as exact
rationals.

### Group By

group-by buckets a list of records into a record of field-value to sublist, in
first-seen order.

### Distinct

distinct drops duplicate list elements, keeping the first of each by value
equality.

### Count

count returns the number of list items or record fields as a number.
