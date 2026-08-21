//go:build aver_registration

package aver

import (
	"slices"
	"testing"
)

// Compiled custom tag must survive source registration.
func Test_Default_Custom_Build_Tag(t *testing.T) {
	context := running_build_context()
	if !slices.Contains(context.BuildTags, "aver_registration") {
		t.Fatalf("compiled tag missing: %v", context.BuildTags)
	}
}
