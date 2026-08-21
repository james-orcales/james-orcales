
# Name

Name_Validate accepts ASCII host labels holding 1 through 63 bytes and wire names holding at most
255 bytes. Final dot is accepted. Empty labels, non-ASCII, and edge hyphens are invalid.

# Resolver

Resolver submits recursive A or AAAA question to injected DNS server through nbio. Dependencies and
storage stay caller-owned. Resolver never drives timeline and permits only one active operation.
UDP runs first. Truncation retries through TCP. Whole operation has one finite timeout.

# Response

Response must match transaction, question, type, and Internet class. Parser follows bounded CNAME
chain and accepts only address records owned by final name. Duplicate addresses collapse.
Malformed data, DNS failures, missing addresses, and short result storage report distinct errors.

# Bounds

DNS formulas derive every limit from protocol field widths. Workspace owns maximum query and
response arrays plus three expanded-name scratch arrays. Resolver retains named scalar counts and
state only. Caller result cannot exceed smallest-address-record capacity of maximum message.

# Allocation

Validation, initialization, query construction, parsing, and inline resolution allocate no heap.
Deferred completion uses same retained caller-owned state and allocates no heap.
