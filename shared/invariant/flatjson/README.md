# invariant flatjson

`shared/encoding/flatjson` imports this package for its internal `Always` assertions.
`shared/invariant/default` imports that encoder to write JSON coverage reports. Therefore, the
encoder cannot import the default package.

This package keeps encoder assertion enforcement without owning process configuration, a recorder,
or another default.
