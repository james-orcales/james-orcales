# aver flatjson

`shared/encoding/flatjson` imports this package for its internal `Always` assertions.
`shared/simulation/aver/default` imports the encoder to write JSON coverage reports. Thus, the
encoder cannot import the default package.

This package enforces the encoder assertions. It does not own process configuration or a recorder.
It does not add another default package.
