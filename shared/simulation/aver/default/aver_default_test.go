//go:build !invariant_disable_coverage && !prd && !prod && !production && !invariant_noop

package aver_test

import (
	"testing"

	core "local/james-orcales/shared/simulation/aver"
	aver "local/james-orcales/shared/simulation/aver/default"
)

// Fixture_Subject stands in for a bundle subject where the test drives the builder directly.
type Fixture_Subject int

// Test_Default_Assertions_Uses_Injected_Recorder protects composition-tier dependency injection.
func Test_Default_Assertions_Uses_Injected_Recorder(t *testing.T) {
	recorder := &core.Recorder{}
	previous := aver.Default
	aver.Default = recorder
	defer func() { aver.Default = previous }()
	message := panic_text_default(func() {
		aver.Tree(Fixture_Subject(0), "range").Range_Int(3, 0, 2).Ensure()
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
