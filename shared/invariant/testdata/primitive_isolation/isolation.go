// Package primitive_isolation exists because namespace isolation needs two statically registered
// callsites before runtime coverage can try to cross-credit them.
package primitive_isolation

import invariant "local/james-orcales/shared/invariant/default"

func first(value int8) {
	invariant.Int8_Invariants(value, "primitive.first")
}

func second(value int8) {
	invariant.Int8_Invariants(value, "primitive.second")
}
