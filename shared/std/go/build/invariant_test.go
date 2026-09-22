package build

import "testing"
import "unsafe"

import "local/james-orcales/shared/sim/nbio"
import "local/james-orcales/shared/testify"

// States one loop whose halves carry one backend, which is the loop a read stands on.
func storage_loop(state unsafe.Pointer) (loop nbio.IO) {
  return nbio.IO{
    Storage: nbio.Storage{State: state},
    Network: nbio.Network{State: state},
  }
}

// Provides every invariant entry point the complete caller-owned tag storage it needs.
func target_state() (held Target) {
  return Target{Tags: Target_Tag_Storage{Values: make(Target_Tags, TAG_COUNT_MAXIMUM)}}
}

// Test_Storage_Widths drives every step of a directory read through storage of the widths the
// package admits. A step reaching one width states nothing about the step beside it, thus every
// entry point stands in the sweep and the widths are the ones the domains name: none, one, two,
// and the widest.
func Test_Storage_Widths(t *testing.T) {
  t.Parallel()
  slots := []int{
    STORAGE_SIZE_MINIMUM,
    STORAGE_SIZE_MINIMUM + 1,
    STORAGE_SIZE_MINIMUM + 2,
    ENTRY_COUNT_MAXIMUM,
  }
  records := []int{
    STORAGE_SIZE_MINIMUM,
    STORAGE_SIZE_MINIMUM + 1,
    STORAGE_SIZE_MINIMUM + 2,
    nbio.DIRECTORY_BUFFER_SIZE_MAXIMUM,
  }
  bytes := []int{
    STORAGE_SIZE_MINIMUM,
    STORAGE_SIZE_MINIMUM + 1,
    STORAGE_SIZE_MINIMUM + 2,
    STORAGE_SIZE_MAXIMUM,
  }
  phases := []Phase{
    PHASE_IDLE, PHASE_OPEN_DIRECTORY, PHASE_READ_ENTRIES, PHASE_CLOSE_DIRECTORY,
  }
  tags := []int{0, 1, 2, TAG_COUNT_MAXIMUM}
  for index := range slots {
    memory := storage_widths(slots[index], records[index], bytes[index])
    one := target_state()
    one.Counts.Tag = Tag_Count(tags[index])
    runner := Directory_Runner{
      Pass_Fields: Pass_Fields{
        Loop:    Loop(storage_loop(unsafe.Pointer(&one))),
        Entries: memory.Entries,
        Records: memory.Records,
        Names:   memory.Names,
        Bytes:   memory.Bytes,
        Header:  memory.Header,
      },
    }
    runner.Counts.Phase = phases[index]
    runner.Counts.Name = Name_Count(slots[index])
    runner.Counts.Entry = Entry_Index(slots[index])
    runner.Counts.Entry_Total = Entry_Total(slots[index])
    runner.Counts.Written = Written_Count(bytes[index])
    storage_step_widths(runner, &one)
    storage_init_widths(runner, memory)
    // Every step reads the flags the callback and the stop wrote, thus each step stands
    // on a runner that holds both as well as on one that holds neither.
    runner.Flags.Stopped = true
    runner.Flags.Queued = true
    storage_step_widths(runner, &one)
    storage_init_widths(runner, memory)
  }
  // An open meets the storage it writes into, thus one standing nowhere at all stands read
  // as well as one the caller owns.
  none := target_state()
  storage_boundary(func() {
    Directory_Runner_Init(
      nil, storage_loop(unsafe.Pointer(&none)), "",
      storage_widths(0, 0, 0),
    )
  })
  // A read hands its names back after it stops, thus the widest run of names stands on a
  // runner that stopped and never on one mid-read.
  one := target_state()
  filled := Directory_Runner{
    Pass_Fields: Pass_Fields{
      Loop:  Loop(storage_loop(unsafe.Pointer(&one))),
      Names: make(Name_Storage, ENTRY_COUNT_MAXIMUM),
    },
  }
  filled.Counts.Name = ENTRY_COUNT_MAXIMUM
  filled.Flags.Stopped = true
  storage_boundary(func() { Directory_Runner_Names(&filled.Pass_Fields) })
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
  }
}

// Runs every step of a read on one runner. Each step reads a copy, because a step the width of the
// storage stops states nothing the step beside it stands on.
func storage_step_widths(runner Directory_Runner, one Target_Handle) {
  storage_boundary(func() {
    Directory_Runner_Rearm(storage_copy(runner), one)
  })
  storage_boundary(func() {
    Directory_Runner_Work_Queued(&storage_copy(runner).Pass_Fields)
  })
  storage_boundary(func() { Directory_Runner_Stopped(&storage_copy(runner).Pass_Fields) })
  storage_boundary(func() { Directory_Runner_Status(&storage_copy(runner).Pass_Fields) })
  storage_boundary(func() { Directory_Runner_Names(&storage_copy(runner).Pass_Fields) })
  storage_boundary(func() {
    directory_completion_apply(storage_copy(runner), one)
  })
  storage_boundary(func() { open_directory_apply(&storage_copy(runner).Pass_Fields) })
  storage_boundary(func() { read_entries_apply(&storage_copy(runner).Pass_Fields) })
  storage_boundary(func() { open_file_apply(&storage_copy(runner).Pass_Fields) })
  storage_boundary(func() { read_header_apply(storage_copy(runner), one) })
  storage_boundary(func() { hold_name(&storage_copy(runner).Pass_Fields) })
  storage_boundary(func() { read_entries(&storage_copy(runner).Pass_Fields) })
  storage_boundary(func() { open_next_file(storage_copy(runner), one) })
  storage_boundary(func() { names_source(storage_copy(runner), one) })
}

func storage_copy(value Directory_Runner) (copy *Directory_Runner) {
  return &value
}

// Opens one read on the widths named, which is the width the caller states rather than the width
// the runner already holds.
func storage_init_widths(runner Directory_Runner, memory Directory_Memory) {
  one := target_state()
  storage_boundary(func() {
    Directory_Runner_Init(
      storage_copy(runner), storage_loop(unsafe.Pointer(&one)), "", memory,
    )
  })
}

// Runs one step and keeps the refusal it states. A width the step refuses is a width the domains
// admit, thus the refusal is the answer the sweep asks for and never a failure of the test.
func storage_boundary(step func()) {
  defer func() { recover() }()
  step()
}

// States each place a name scan names, from the first byte to the last the constraint text
// admits. The two places stand inside that one scan, thus the scan alone can state them.
func Test_Name_Place_Answers(t *testing.T) {
  text := ""
  for range TEXT_SIZE_MAXIMUM {
    text = text + "x"
  }
  wide := Constraint_Text(text)
  testify.Equal(
    t,
    Opening_Place(TEXT_SIZE_MAXIMUM),
    take_opening(wide),
    "a text holding no underscore opens at its end",
  )
  divider := text[:TEXT_SIZE_MAXIMUM-1] + "_"
  testify.Equal(
    t,
    Field_Place(TEXT_SIZE_MAXIMUM),
    take_field(Stem(divider)),
    "a stem closing at an underscore opens its field at its end",
  )
  testify.Equal(
    t,
    Opening_Place(1),
    take_opening(Constraint_Text("a_")),
    "a text of one letter opens at its second byte",
  )
  testify.Equal(
    t,
    Field_Place(2),
    take_field(Stem("a_")),
    "a stem of one letter opens its field at its third byte",
  )
  testify.Equal(
    t,
    Field_Place(1),
    take_field(Stem("_")),
    "a stem of one underscore opens its field behind it",
  )
}
