//go:build prd

package gzip_test

import (
	"runtime"
	"testing"

	"local/james-orcales/shared/compress/gzip"
)

// Benchmark_Cold_Allocation prevents lazy process-global state from hiding behind allocation-test
// warmup. Benchmark mode also keeps invariant coverage bookkeeping outside this production check.
func Benchmark_Cold_Allocation(b *testing.B) {
	if b.N != 1 {
		b.Fatalf("cold allocation benchmark iterations = %d, want 1", b.N)
	}
	previous_processor_count := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous_processor_count)

	var destination [TEST_DESTINATION_SIZE]byte
	var workspace_storage test_workspace
	workspace := test_workspace_value(&workspace_storage)
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	_, status := gzip.Encode_Into(
		destination[:], workspace, []byte(TEST_HELLO_CONTENT),
		gzip.Header_Unvalidated{}, gzip.DEFAULT_COMPRESSION,
	)
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	if status != gzip.STATUS_OK {
		b.Fatalf("cold Encode_Into status = %d, want STATUS_OK", status)
	}
	if after.Mallocs != before.Mallocs {
		b.Fatalf("cold Encode_Into allocations = %d, want 0", after.Mallocs-before.Mallocs)
	}
}
