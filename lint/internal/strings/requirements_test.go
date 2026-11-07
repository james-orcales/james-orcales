package strings_test

import (
	"testing"

	"local/james-orcales/lint/internal/strings"
)

// Test_Search verifies scalar search operations.
func Test_Search(t *testing.T) {
	t.Parallel()
	if !strings.Contains("alpha", "ph") {
		t.Fatal("Contains must find substring")
	}
	if strings.Index("alpha", "ph") != 2 {
		t.Fatal("Index must report substring start")
	}
	if strings.Count("banana", "an") != 2 {
		t.Fatal("Count must report non-overlapping matches")
	}
	if !strings.Has_Prefix("alpha", "al") {
		t.Fatal("prefix must match")
	}
	if !strings.Has_Suffix("alpha", "ha") {
		t.Fatal("prefix and suffix must match")
	}
}

// Test_Split_And_Transform verifies storage-owning transformations.
func Test_Split_And_Transform(t *testing.T) {
	t.Parallel()
	parts := strings.Split("a,b,c", ",")
	if len(parts) != 3 {
		t.Fatalf("Split = %#v", parts)
	}
	if parts[1] != "b" {
		t.Fatalf("Split = %#v", parts)
	}
	if strings.Join(parts, "-") != "a-b-c" {
		t.Fatal("Join must retain order")
	}
	if strings.Trim_Space(" \tvalue\n") != "value" {
		t.Fatal("Trim_Space must remove Unicode space")
	}
	if strings.Trim_Space("\u2003value\u2003") != "value" {
		t.Fatal("Trim_Space must retain Unicode fallback")
	}
	unicode_fields := strings.Fields("left\u2003right")
	if len(unicode_fields) != 2 {
		t.Fatalf("Fields = %#v", unicode_fields)
	}
	if strings.To_Upper("a世") != "A世" {
		t.Fatal("upper case mapping must retain Unicode text")
	}
	if strings.To_Lower("A世") != "a世" {
		t.Fatal("case mapping must retain Unicode text")
	}
	if strings.To_Upper("ä") != "Ä" {
		t.Fatal("upper case mapping must retain Unicode fallback")
	}
	if strings.To_Lower("Ä") != "ä" {
		t.Fatal("lower case mapping must retain Unicode fallback")
	}
	if strings.Replace_All("a-b-a", "a", "x") != "x-b-x" {
		t.Fatal("Replace_All must replace each match")
	}
}

// Test_Builder verifies bounded writer and explicit writes share content.
func Test_Builder(t *testing.T) {
	t.Parallel()
	var builder strings.Builder
	if _, err := builder.Write([]byte("a")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	strings.Builder_Write_Text(&builder, "b")
	strings.Builder_Write_Byte(&builder, 'c')
	if builder.String() != "abc" {
		t.Fatalf("String = %q", builder.String())
	}
}

// Test_Unchanged_ASCII_Case_Conversion_Allocation prevents hot-path copies.
func Test_Unchanged_ASCII_Case_Conversion_Allocation(t *testing.T) {
	result := ""
	lower_allocations := testing.AllocsPerRun(100, func() {
		result = strings.To_Lower("already_lower")
	})
	if result != "already_lower" {
		t.Fatalf("To_Lower = %q", result)
	}
	if lower_allocations != 0 {
		t.Fatalf("To_Lower allocations = %f", lower_allocations)
	}
	upper_allocations := testing.AllocsPerRun(100, func() {
		result = strings.To_Upper("ALREADY_UPPER")
	})
	if result != "ALREADY_UPPER" {
		t.Fatalf("To_Upper = %q", result)
	}
	if upper_allocations != 0 {
		t.Fatalf("To_Upper allocations = %f", upper_allocations)
	}
}
