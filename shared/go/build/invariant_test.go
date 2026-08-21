package build

import (
	"testing"
	"unsafe"

	"local/james-orcales/shared/sim/nbio"
	"local/james-orcales/shared/testify"
)

// States one loop whose halves carry one backend, which is the loop a read stands on.
func storage_loop(state unsafe.Pointer) (loop nbio.IO) {
	return nbio.IO{
		Storage: nbio.Storage{State: state},
		Network: nbio.Network{State: state},
	}
}

// Test_Storage_Widths drives every step of a directory read through storage of the widths the
// package admits. A step reaching one width states nothing about the step beside it, thus every
// entry point stands in the sweep and the widths are the ones the domains name: none, one, two,
// and the widest.
func Test_Storage_Widths(t *testing.T) {
	t.Parallel()
	slots := []int{
		STORAGE_SIZE_MINIMUM, STORAGE_SIZE_MINIMUM + 1, STORAGE_SIZE_MINIMUM + 2,
		ENTRY_COUNT_MAXIMUM,
	}
	records := []int{
		STORAGE_SIZE_MINIMUM, STORAGE_SIZE_MINIMUM + 1, STORAGE_SIZE_MINIMUM + 2,
		nbio.DIRECTORY_BUFFER_SIZE_MAXIMUM,
	}
	bytes := []int{
		STORAGE_SIZE_MINIMUM, STORAGE_SIZE_MINIMUM + 1, STORAGE_SIZE_MINIMUM + 2,
		STORAGE_SIZE_MAXIMUM,
	}
	phases := []Count{
		PHASE_IDLE, PHASE_OPEN_DIRECTORY, PHASE_READ_ENTRIES, PHASE_CLOSE_DIRECTORY,
	}
	for index := range slots {
		memory := storage_widths(slots[index], records[index], bytes[index])
		one := Target{}
		runner := Directory_Runner{
			Loop:   Loop(storage_loop(unsafe.Pointer(&one))),
			Target: &one, Reader: memory.Reader, Entries: memory.Entries,
			Records: memory.Records, Names: memory.Names, Bytes: memory.Bytes,
			Header: memory.Header,
		}
		runner.Counts[DIRECTORY_COUNT_PHASE] = phases[index]
		storage_step_widths(runner)
		storage_init_widths(runner, memory)
	}
	// An open meets the storage it writes into, thus one standing nowhere at all stands read
	// as well as one the caller owns.
	none := Target{}
	storage_boundary(func() {
		Directory_Runner_Init(
			nil, storage_loop(unsafe.Pointer(&none)), &none, "",
			storage_widths(0, 0, 0),
		)
	})
	// A read hands its names back after it stops, thus the widest run of names stands on a
	// runner that stopped and never on one mid-read.
	one := Target{}
	filled := Directory_Runner{
		Loop:   Loop(storage_loop(unsafe.Pointer(&one))),
		Target: &one, Reader: new(Build),
		Names: make(Name_Storage, ENTRY_COUNT_MAXIMUM),
	}
	filled.Counts[DIRECTORY_COUNT_NAME] = ENTRY_COUNT_MAXIMUM
	filled.Flags[DIRECTORY_FLAG_STOPPED] = true
	storage_boundary(func() { Directory_Runner_Names(&filled) })
	testify.True(t, true, "every step of a read stands on every width")
}

// States one run of storage of the widths named, which the sweep hands every step.
func storage_widths(slots int, records int, bytes int) (memory Directory_Memory) {
	return Directory_Memory{
		Entries: make(Entry_Storage, slots),
		Records: make(Record_Storage, records),
		Names:   make(Name_Storage, slots),
		Bytes:   make(Name_Bytes, bytes),
		Header:  make(Header_Storage, bytes),
		Reader:  new(Build),
	}
}

// Runs every step of a read on one runner. Each step reads a copy, because a step the width of the
// storage stops states nothing the step beside it stands on.
func storage_step_widths(runner Directory_Runner) {
	storage_boundary(func() { value := runner; Directory_Runner_Rearm(&value) })
	storage_boundary(func() { value := runner; Directory_Runner_Work_Queued(&value) })
	storage_boundary(func() { value := runner; Directory_Runner_Stopped(&value) })
	storage_boundary(func() { value := runner; Directory_Runner_Status(&value) })
	storage_boundary(func() { value := runner; Directory_Runner_Names(&value) })
	storage_boundary(func() { value := runner; directory_completion_apply(&value) })
	storage_boundary(func() { value := runner; open_directory_apply(&value) })
	storage_boundary(func() { value := runner; read_entries_apply(&value) })
	storage_boundary(func() { value := runner; open_file_apply(&value) })
	storage_boundary(func() { value := runner; read_header_apply(&value) })
	storage_boundary(func() { value := runner; hold_name(&value) })
	storage_boundary(func() { value := runner; read_entries(&value) })
	storage_boundary(func() { value := runner; open_next_file(&value) })
	storage_boundary(func() { value := runner; names_source(&value) })
}

// Opens one read on the widths named, which is the width the caller states rather than the width
// the runner already holds.
func storage_init_widths(runner Directory_Runner, memory Directory_Memory) {
	one := Target{}
	storage_boundary(func() {
		value := runner
		Directory_Runner_Init(
			&value, storage_loop(unsafe.Pointer(&one)), &one, "", memory,
		)
	})
}

// Runs one step and keeps the refusal it states. A width the step refuses is a width the domains
// admit, thus the refusal is the answer the sweep asks for and never a failure of the test.
func storage_boundary(step func()) {
	defer func() { recover() }()
	step()
}
