
# Overflow File

Coverage overflow writes all supplied gap records to a JSON file and returns its path.

# JSON Output Records

JSON output represents branch and reachability gaps using a stable flat schema and ends with a newline.

# JSON Output Selection

An unset output environment variable and `table` select table output; `json` selects JSON.
An unsupported value produces a configuration diagnostic.

# JSON Output Failure

JSON output reports a failed write to its caller.
