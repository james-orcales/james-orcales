package snap_test

import (
	"strings"
	"testing"

	"github.com/james-orcales/james-orcales/golang_snacks/snap"
)

func TestBatchExpect(t *testing.T) {
	snap.BatchExpect(t, func(input string) any {
		return strings.ToUpper(input)
	}, []snap.Entry[string]{
		{"uppercase", "hello", snap.Init(`HELLO`)},
		{"with spaces", "hello world", snap.Init(`HELLO WORLD`)},
		{"empty string", "", snap.Init(``)},
	})
}

func TestBatchExpectPanic(t *testing.T) {
	snap.BatchExpectPanic(t, func(msg string) {
		panic(msg)
	}, []snap.Entry[string]{
		{"custom panic", "test panic", snap.Init(`test panic`)},
		{"another panic", "another message", snap.Init(`another message`)},
	})
}
