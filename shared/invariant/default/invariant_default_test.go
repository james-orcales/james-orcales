//go:build !noassert

package invariant_test

import (
	"math"
	"testing"

	core "local/james-orcales/shared/invariant"
	invariant "local/james-orcales/shared/invariant/default"
)

// Test_Primitive_Invariants_Use_Assertions_Builders prevents primitive sugar bypassing Ensure.
func Test_Primitive_Invariants_Use_Assertions_Builders(t *testing.T) {
	recorder := &core.Recorder{}
	previous := invariant.Default
	invariant.Default = recorder
	defer func() { invariant.Default = previous }()
	invariant.Int_Invariants(1, "int")
	invariant.Int8_Invariants(1, "int8")
	invariant.Int16_Invariants(1, "int16")
	invariant.Int32_Invariants(1, "int32")
	invariant.Int64_Invariants(1, "int64")
	invariant.Uint_Invariants(1, "uint")
	invariant.Uint8_Invariants(1, "uint8")
	invariant.Uint16_Invariants(1, "uint16")
	invariant.Uint32_Invariants(1, "uint32")
	invariant.Uint64_Invariants(1, "uint64")
	invariant.Float64_Invariants(math.Inf(1), "float")
	invariant.Float32_Invariants(float32(math.Inf(1)), "float32")
	invariant.Boolean_Invariants(true, "bool")
}

// Test_Default_Assertions_Uses_Injected_Recorder protects composition-tier dependency injection.
func Test_Default_Assertions_Uses_Injected_Recorder(t *testing.T) {
	recorder := &core.Recorder{}
	previous := invariant.Default
	invariant.Default = recorder
	defer func() { invariant.Default = previous }()
	message := panic_text_default(func() {
		invariant.Assertions("range").Range_Int(3, 0, 2).Ensure()
	})
	if message == "" {
		t.Fatal("default builder did not enforce through the injected recorder")
	}
}

func panic_text_default(action func()) (message string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			message = recovered.(string)
		}
	}()
	action()
	return ""
}
