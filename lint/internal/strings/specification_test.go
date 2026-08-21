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
	if strings.To_Upper("a世") != "A世" {
		t.Fatal("upper case mapping must retain Unicode text")
	}
	if strings.To_Lower("A世") != "a世" {
		t.Fatal("case mapping must retain Unicode text")
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
