// Package filepath keeps host paths bounded. Ambient filesystem access stays injected elsewhere.
package filepath

import (
	"errors"
	"unsafe"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/path"
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/sim/nbio"
	"local/james-orcales/shared/unicode/utf8"
)

// Error_Path_Absent gives injected Status same missing-path result on every backend.
var Error_Path_Absent = errors.New("filepath: path does not exist")

// Error_Not_Directory rejects traversal through ordinary file.
var Error_Not_Directory = errors.New("filepath: path component is not directory")

// Error_Too_Many_Links bounds malicious symbolic-link cycles.
var Error_Too_Many_Links = errors.New("filepath: too many symbolic links")

// Skip_Directory tells walk to omit directory descendants. On file it omits remaining siblings,
// matching standard library contract.
var Skip_Directory = errors.New("filepath: skip directory")

// Skip_All tells walk to finish successfully without visiting remaining paths.
var Skip_All = errors.New("filepath: skip all")

// SYMBOLIC_LINK_COUNT_MAXIMUM shares the standard library traversal guard with its
// authoritative unsigned-byte bound.
const SYMBOLIC_LINK_COUNT_MAXIMUM = 1<<bits.BIT_COUNT_8_MAXIMUM - NONEMPTY_SIZE_MINIMUM

// SYMBOLIC_LINK_COUNT_MINIMUM is resolution before any link.
const SYMBOLIC_LINK_COUNT_MINIMUM = 0

// SYMBOLIC_LINK_COUNT_BOUNDARY is the first rejected link traversal count.
const SYMBOLIC_LINK_COUNT_BOUNDARY = SYMBOLIC_LINK_COUNT_MAXIMUM + NONEMPTY_SIZE_MINIMUM

// FILESYSTEM_PATH_COUNT_MINIMUM keeps zero-value runner valid before initialization.
const FILESYSTEM_PATH_COUNT_MINIMUM = 0

// FILESYSTEM_PATH_COUNT_MAXIMUM gives one traversal slot to each pathname boundary.
const FILESYSTEM_PATH_COUNT_MAXIMUM = PATH_SIZE_MAXIMUM + NONEMPTY_SIZE_MINIMUM

// PATH_PARENT_SIZE_MAXIMUM leaves separator and one-byte base inside path bound.
const PATH_PARENT_SIZE_MAXIMUM = PATH_SIZE_MAXIMUM - 2*NONEMPTY_SIZE_MINIMUM

// Path_Storage holds caller-owned fixed-capacity path slots. Populated slots are resliced to
// result length; backing storage stays caller-owned.
type Path_Storage []Slice

// Path_Storage_Invariants bounds traversal width and every path slot.
func Path_Storage_Invariants(value Path_Storage, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), FILESYSTEM_PATH_COUNT_MINIMUM, FILESYSTEM_PATH_COUNT_MAXIMUM).
		Ensure()
}

// Parent_Path is directory prefix returned from one bounded path.
type Parent_Path Slice

// Parent_Path_Invariants spans empty prefix through largest possible parent.
func Parent_Path_Invariants(value Parent_Path, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), PATH_SIZE_MINIMUM, PATH_PARENT_SIZE_MAXIMUM).
		Ensure()
}

// Glob_Current_Paths owns candidates consumed by active component.
type Glob_Current_Paths Path_Storage

// Glob_Current_Paths_Invariants bounds active candidate collection.
func Glob_Current_Paths_Invariants(value Glob_Current_Paths, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), FILESYSTEM_PATH_COUNT_MINIMUM, FILESYSTEM_PATH_COUNT_MAXIMUM).
		Ensure()
}

// Glob_Next_Paths owns candidates produced by active component.
type Glob_Next_Paths Path_Storage

// Glob_Next_Paths_Invariants bounds produced candidate collection.
func Glob_Next_Paths_Invariants(value Glob_Next_Paths, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), FILESYSTEM_PATH_COUNT_MINIMUM, FILESYSTEM_PATH_COUNT_MAXIMUM).
		Ensure()
}

// Glob_Current_Count bounds populated active candidates.
type Glob_Current_Count int

// Glob_Current_Count_Invariants spans empty through caller candidate bound.
func Glob_Current_Count_Invariants(value Glob_Current_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), FILESYSTEM_PATH_COUNT_MINIMUM, FILESYSTEM_PATH_COUNT_MAXIMUM).
		Ensure()
}

// Glob_Next_Count bounds populated produced candidates.
type Glob_Next_Count int

// Glob_Next_Count_Invariants spans empty through caller candidate bound.
func Glob_Next_Count_Invariants(value Glob_Next_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), FILESYSTEM_PATH_COUNT_MINIMUM, FILESYSTEM_PATH_COUNT_MAXIMUM).
		Ensure()
}

// Glob_Candidate_Index identifies candidate or boundary after final candidate.
type Glob_Candidate_Index int

// Glob_Candidate_Index_Invariants spans first candidate through collection boundary.
func Glob_Candidate_Index_Invariants(
	value Glob_Candidate_Index, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), FILESYSTEM_PATH_COUNT_MINIMUM, FILESYSTEM_PATH_COUNT_MAXIMUM).
		Ensure()
}

// Glob_Segment_Start identifies active component first byte.
type Glob_Segment_Start int

// Glob_Segment_Start_Invariants spans pattern boundaries.
func Glob_Segment_Start_Invariants(value Glob_Segment_Start, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), PATH_SIZE_MINIMUM, PATH_SIZE_MAXIMUM).
		Ensure()
}

// Glob_Segment_End identifies boundary after active component.
type Glob_Segment_End int

// Glob_Segment_End_Invariants spans pattern boundaries.
func Glob_Segment_End_Invariants(value Glob_Segment_End, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), PATH_SIZE_MINIMUM, PATH_SIZE_MAXIMUM).
		Ensure()
}

// Glob_Segment_Final separates intermediate and final component.
type Glob_Segment_Final bool

// Glob_Segment_Final_Invariants proves both component positions occur.
func Glob_Segment_Final_Invariants(value Glob_Segment_Final, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Glob component is final.").
		Ensure()
}

// Glob_Segment_Meta separates literal and pattern component.
type Glob_Segment_Meta bool

// Glob_Segment_Meta_Invariants proves both component forms occur.
func Glob_Segment_Meta_Invariants(value Glob_Segment_Meta, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Glob component contains pattern syntax.").
		Ensure()
}

// Glob_Work_Ready separates pending IO from queued continuation.
type Glob_Work_Ready bool

// Glob_Work_Ready_Invariants proves both async lifecycle states occur.
func Glob_Work_Ready_Invariants(value Glob_Work_Ready, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Glob callback queued continuation.").
		Ensure()
}

// Glob_Done separates active and terminal runner.
type Glob_Done bool

// Glob_Done_Invariants proves active and terminal runner states occur.
func Glob_Done_Invariants(value Glob_Done, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Glob runner stopped.").
		Ensure()
}

// Directory_Entries gives retained nbio entry slots defined bounded type.
type Directory_Entries []nbio.Directory_Entry

// Directory_Entries_Invariants requires bounded positive pass capacity.
func Directory_Entries_Invariants(value Directory_Entries, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), FILESYSTEM_PATH_COUNT_MINIMUM, FILESYSTEM_PATH_COUNT_MAXIMUM).
		Ensure()
}

// Directory_Buffer gives backend record storage defined bounded type.
type Directory_Buffer []byte

// Directory_Buffer_Invariants bounds one backend directory pass.
func Directory_Buffer_Invariants(value Directory_Buffer, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			len(value), FILESYSTEM_PATH_COUNT_MINIMUM,
			nbio.DIRECTORY_BUFFER_SIZE_MAXIMUM,
		).
		Ensure()
}

// Glob_Memory states all retained collection ownership.
type Glob_Memory struct {
	// Current holds candidates for current pattern component and final matches.
	Current Glob_Current_Paths
	// Next holds candidates produced by current component.
	Next Glob_Next_Paths
	// Entries receives one bounded directory pass.
	Entries Directory_Entries
	// Directory_Buffer receives backend-specific directory records.
	Directory_Buffer Directory_Buffer
}

// Glob_Memory_Invariants states each retained collection has usable bounded storage.
func Glob_Memory_Invariants(value Glob_Memory, namespace aver.Namespace) {
	Glob_Current_Paths_Invariants(value.Current, namespace)
	Glob_Next_Paths_Invariants(value.Next, namespace)
	Directory_Entries_Invariants(value.Entries, namespace)
	Directory_Buffer_Invariants(value.Directory_Buffer, namespace)
	aver.Always(len(value.Current) > 0, "Glob has current candidate capacity.")
	aver.Always(len(value.Next) > 0, "Glob has next candidate capacity.")
	aver.Always(len(value.Entries) > 0, "Glob has directory-entry capacity.")
	aver.Always(len(value.Directory_Buffer) > 0,
		"Glob has directory-record capacity.")
}

// Glob_Phase identifies directory operation whose completion is queued.
type Glob_Phase uint8

// Glob_Phase_Invariants accepts exactly idle, open, read, and close states.
func Glob_Phase_Invariants(value Glob_Phase, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(GLOB_PHASE_IDLE), uint8(GLOB_PHASE_OPEN),
			uint8(GLOB_PHASE_READ), uint8(GLOB_PHASE_CLOSE),
		).
		Ensure()
}

// GLOB_PHASE_IDLE means no injected operation is submitted.
const GLOB_PHASE_IDLE Glob_Phase = 0

// GLOB_PHASE_OPEN means directory open is submitted.
const GLOB_PHASE_OPEN Glob_Phase = GLOB_PHASE_IDLE + NONEMPTY_SIZE_MINIMUM

// GLOB_PHASE_READ means directory entry read is submitted.
const GLOB_PHASE_READ Glob_Phase = GLOB_PHASE_OPEN + NONEMPTY_SIZE_MINIMUM

// GLOB_PHASE_CLOSE means directory close is submitted.
const GLOB_PHASE_CLOSE Glob_Phase = GLOB_PHASE_READ + NONEMPTY_SIZE_MINIMUM

// Glob_Runner composes injected async directory primitives without owning driver. Completion is
// first so static callback can recover caller-owned runner without closure allocation.
type Glob_Runner struct {
	// Completion stays first so static callback recovers runner with no retained closure.
	Completion nbio.Completion
	// Loop submits storage work while composition root alone owns driver.
	Loop nbio.IO
	// Pattern remains borrowed until runner stops.
	Pattern Text
	// Current owns candidates for active component.
	Current Glob_Current_Paths
	// Next owns candidates produced for following component.
	Next Glob_Next_Paths
	// Entries receives one directory pass.
	Entries Directory_Entries
	// Directory_Buffer receives backend records for one pass.
	Directory_Buffer Directory_Buffer
	// Current_Count bounds populated current candidates.
	Current_Count Glob_Current_Count
	// Next_Count bounds populated next candidates.
	Next_Count Glob_Next_Count
	// Candidate_Index identifies active current candidate.
	Candidate_Index Glob_Candidate_Index
	// Segment_Start identifies active component first byte.
	Segment_Start Glob_Segment_Start
	// Segment_End identifies boundary after active component.
	Segment_End Glob_Segment_End
	// Segment_Final reports active component ends pattern.
	Segment_Final Glob_Segment_Final
	// Segment_Meta reports active component needs directory matching.
	Segment_Meta Glob_Segment_Meta
	// Directory is descriptor held between open and close completions.
	Directory nbio.File
	// Phase identifies submitted directory operation.
	Phase Glob_Phase
	// Work_Ready reports callback queued continuation.
	Work_Ready Glob_Work_Ready
	// Done reports terminal state.
	Done Glob_Done
	// Result retains terminal composition error.
	Result error
}

// Glob_Runner_Invariants states complete runner scalar domains.
func Glob_Runner_Invariants(value Glob_Runner, namespace aver.Namespace) {
	nbio.IO_Invariants(value.Loop, namespace)
	Text_Invariants(value.Pattern, namespace)
	Glob_Current_Paths_Invariants(value.Current, namespace)
	Glob_Next_Paths_Invariants(value.Next, namespace)
	Directory_Entries_Invariants(value.Entries, namespace)
	Directory_Buffer_Invariants(value.Directory_Buffer, namespace)
	Glob_Current_Count_Invariants(value.Current_Count, namespace)
	Glob_Next_Count_Invariants(value.Next_Count, namespace)
	Glob_Candidate_Index_Invariants(value.Candidate_Index, namespace)
	Glob_Segment_Start_Invariants(value.Segment_Start, namespace)
	Glob_Segment_End_Invariants(value.Segment_End, namespace)
	Glob_Segment_Final_Invariants(value.Segment_Final, namespace)
	Glob_Segment_Meta_Invariants(value.Segment_Meta, namespace)
	Glob_Phase_Invariants(value.Phase, namespace)
	Glob_Work_Ready_Invariants(value.Work_Ready, namespace)
	Glob_Done_Invariants(value.Done, namespace)
}

// Glob_Runner_Handle keeps caller-owned glob state nonnil.
type Glob_Runner_Handle *Glob_Runner

// Glob_Runner_Handle_Invariants states complete runner scalar domains.
func Glob_Runner_Handle_Invariants(value Glob_Runner_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Glob_Runner_Invariants(*value, namespace)
}

// Glob_Runner_Init validates pattern and binds injected storage. It never drives loop.
func Glob_Runner_Init(
	runner Glob_Runner_Handle, loop nbio.IO, pattern Text, memory Glob_Memory,
) (err error) {
	Glob_Runner_Handle_Invariants(runner, "glob_runner_init.runner")
	nbio.IO_Invariants(loop, "glob_runner_init.loop")
	Text_Invariants(pattern, "glob_runner_init.pattern")
	Glob_Memory_Invariants(memory, "glob_runner_init.memory")
	path_storage_slots_validate(Path_Storage(memory.Current))
	path_storage_slots_validate(Path_Storage(memory.Next))
	aver.Always(len(memory.Current) == len(memory.Next),
		"Glob candidate generations have equal capacity.")
	directory_buffer_validate(memory.Directory_Buffer)
	*runner = Glob_Runner{
		Loop: loop, Pattern: pattern, Current: memory.Current, Next: memory.Next,
		Entries: memory.Entries, Directory_Buffer: memory.Directory_Buffer,
	}
	if _, match_err := Match(pattern, ""); match_err != nil {
		runner.Done = true
		runner.Result = Error_Bad_Pattern
		return Error_Bad_Pattern
	}
	if !path_has_meta(pattern) {
		runner.Done = true
		if pattern == "" {
			return nil
		}
		status, status_err := nbio.Storage_Status(loop.Storage, string(pattern))
		if status_err != nil {
			return nil
		}
		if !status.Exists {
			return nil
		}
		path, copy_err := path_storage_write_text(
			Path_Storage(runner.Current), 0, pattern,
		)
		if copy_err != nil {
			runner.Result = copy_err
			return copy_err
		}
		runner.Current[0] = path
		runner.Current_Count = 1
		return nil
	}
	if len(pattern) > 0 {
		if pattern[0] == SEPARATOR {
			root, copy_err := path_storage_write(
				Path_Storage(runner.Current), 0, Slice{SEPARATOR},
			)
			if copy_err != nil {
				runner.Done = true
				runner.Result = copy_err
				return copy_err
			}
			runner.Current[0] = root
		} else {
			runner.Current[0] = runner.Current[0][:0]
		}
	} else {
		runner.Current[0] = runner.Current[0][:0]
	}
	runner.Current_Count = 1
	glob_segment_advance(runner)
	return nil
}

// Glob_Runner_Rearm executes queued continuation or synchronous lexical work. False means root
// must drive pending IO, or runner stopped.
func Glob_Runner_Rearm(runner Glob_Runner_Handle) (rearm Boolean) {
	defer func() { Boolean_Invariants(rearm, "glob_runner_rearm.rearm") }()
	Glob_Runner_Handle_Invariants(runner, "glob_runner_rearm.runner")
	if runner.Done {
		return false
	}
	if runner.Work_Ready {
		runner.Work_Ready = false
		return glob_completion_apply(runner)
	}
	if int(runner.Candidate_Index) >= int(runner.Current_Count) {
		glob_generation_finish(runner)
		return !Boolean(runner.Done)
	}
	if runner.Segment_Meta {
		return glob_meta_begin(runner)
	}
	return glob_literal_apply(runner)
}

// Glob_Runner_Work_Queued reports callback stored one continuation for root.
func Glob_Runner_Work_Queued(runner Glob_Runner_Handle) (queued Boolean) {
	defer func() { Boolean_Invariants(queued, "glob_runner_work_queued.queued") }()
	Glob_Runner_Handle_Invariants(runner, "glob_runner_work_queued.runner")
	return Boolean(runner.Work_Ready)
}

// Glob_Runner_Stopped reports terminal success or failure.
func Glob_Runner_Stopped(runner Glob_Runner_Handle) (stopped Boolean) {
	defer func() { Boolean_Invariants(stopped, "glob_runner_stopped.stopped") }()
	Glob_Runner_Handle_Invariants(runner, "glob_runner_stopped.runner")
	return Boolean(runner.Done)
}

// Glob_Runner_Status reports terminal error. Filesystem lookup errors are ignored by Glob.
func Glob_Runner_Status(runner Glob_Runner_Handle) (err error) {
	Glob_Runner_Handle_Invariants(runner, "glob_runner_status.runner")
	return runner.Result
}

// Glob_Runner_Matches returns caller-owned lexically sorted populated slots.
func Glob_Runner_Matches(runner Glob_Runner_Handle) (matches Path_Storage) {
	defer func() { Path_Storage_Invariants(matches, "glob_runner_matches.matches") }()
	Glob_Runner_Handle_Invariants(runner, "glob_runner_matches.runner")
	aver.Always(runner.Done, "Glob result is read after runner stops.")
	return Path_Storage(runner.Current[:runner.Current_Count])
}

func glob_completion(completion nbio.Completion_Handle) {
	// First-field ownership avoids closure allocation and a self-pointer escape.
	runner := Glob_Runner_Handle(unsafe.Pointer(completion))
	runner.Work_Ready = true
}

func glob_completion_apply(runner Glob_Runner_Handle) (rearm Boolean) {
	defer func() { Boolean_Invariants(rearm, "glob_completion_apply.rearm") }()
	Glob_Runner_Handle_Invariants(runner, "glob_completion_apply.runner")
	aver.Always(
		runner.Phase != GLOB_PHASE_IDLE,
		"Filepath Glob completion has pending operation.",
	)
	switch runner.Phase {
	case GLOB_PHASE_OPEN:
		if runner.Completion.Error != nil {
			runner.Candidate_Index++
			runner.Phase = GLOB_PHASE_IDLE
			return true
		}
		runner.Directory = nbio.File(runner.Completion.Data)
		glob_directory_read(runner)
		return false
	case GLOB_PHASE_READ:
		if runner.Completion.Error != nil {
			glob_directory_close(runner)
			return false
		}
		if runner.Completion.Data == 0 {
			glob_directory_close(runner)
			return false
		}
		glob_entries_apply(runner, Path_Count(runner.Completion.Data))
		if runner.Result != nil {
			glob_directory_close(runner)
			return false
		}
		glob_directory_read(runner)
		return false
	case GLOB_PHASE_CLOSE:
		runner.Candidate_Index++
		runner.Phase = GLOB_PHASE_IDLE
		return true
	}
	return false
}

func glob_meta_begin(runner Glob_Runner_Handle) (rearm Boolean) {
	defer func() { Boolean_Invariants(rearm, "glob_meta_begin.rearm") }()
	Glob_Runner_Handle_Invariants(runner, "glob_meta_begin.runner")
	candidate := runner.Current[runner.Candidate_Index]
	directory_text := path_slice_text(candidate)
	if len(candidate) == 0 {
		directory_text = "."
	}
	status, status_err := nbio.Storage_Status(runner.Loop.Storage, string(directory_text))
	if status_err != nil {
		runner.Candidate_Index++
		return true
	}
	if !status.Exists {
		runner.Candidate_Index++
		return true
	}
	if !nbio.File_Mode_Is_Directory(status.Mode) {
		runner.Candidate_Index++
		return true
	}
	runner.Phase = GLOB_PHASE_OPEN
	runner.Work_Ready = false
	nbio.Storage_Open_At(
		runner.Loop.Storage, &runner.Completion, nbio.DIRECTORY_CURRENT,
		string(directory_text), nbio.Open_At_Options{Access: nbio.OPEN_READ_ONLY},
		nbio.Callback(glob_completion),
	)
	return false
}

func glob_directory_read(runner Glob_Runner_Handle) {
	Glob_Runner_Handle_Invariants(runner, "glob_directory_read.runner")
	runner.Phase = GLOB_PHASE_READ
	runner.Work_Ready = false
	nbio.Storage_Get_Directory_Entries(
		runner.Loop.Storage, &runner.Completion, runner.Directory,
		runner.Directory_Buffer, runner.Entries, nbio.Callback(glob_completion),
	)
}

func glob_directory_close(runner Glob_Runner_Handle) {
	Glob_Runner_Handle_Invariants(runner, "glob_directory_close.runner")
	runner.Phase = GLOB_PHASE_CLOSE
	runner.Work_Ready = false
	nbio.IO_Close(
		runner.Loop, &runner.Completion, runner.Directory,
		nbio.Callback(glob_completion),
	)
}

func glob_entries_apply(runner Glob_Runner_Handle, count Path_Count) {
	Glob_Runner_Handle_Invariants(runner, "glob_entries_apply.runner")
	Path_Count_Invariants(count, "glob_entries_apply.count")
	aver.Always(count > 0, "Glob directory completion has entries.")
	aver.Always(int(count) <= len(runner.Entries),
		"Glob directory completion count fits caller entries.")
	segment := runner.Pattern[runner.Segment_Start:runner.Segment_End]
	for index := 0; index < int(count); index++ {
		entry := runner.Entries[index]
		matched, match_err := Match(segment, Text(entry.Name))
		if match_err != nil {
			runner.Result = Error_Bad_Pattern
			return
		}
		if !matched {
			continue
		}
		joined, join_err := path_storage_join(
			Path_Storage(runner.Next), Path_Count(runner.Next_Count),
			runner.Current[runner.Candidate_Index], Text(entry.Name),
		)
		if join_err != nil {
			runner.Result = join_err
			return
		}
		runner.Next[runner.Next_Count] = joined
		runner.Next_Count++
	}
}

func glob_literal_apply(runner Glob_Runner_Handle) (rearm Boolean) {
	defer func() { Boolean_Invariants(rearm, "glob_literal_apply.rearm") }()
	Glob_Runner_Handle_Invariants(runner, "glob_literal_apply.runner")
	segment := runner.Pattern[runner.Segment_Start:runner.Segment_End]
	joined, join_err := path_storage_join(
		Path_Storage(runner.Next), Path_Count(runner.Next_Count),
		runner.Current[runner.Candidate_Index], segment,
	)
	runner.Candidate_Index++
	if join_err != nil {
		runner.Result = join_err
		runner.Done = true
		return false
	}
	status, status_err := nbio.Storage_Status(
		runner.Loop.Storage, string(path_slice_text(joined)),
	)
	if status_err != nil {
		return true
	}
	if !status.Exists {
		return true
	}
	if !runner.Segment_Final {
		if !nbio.File_Mode_Is_Directory(status.Mode) {
			return true
		}
	}
	runner.Next[runner.Next_Count] = joined
	runner.Next_Count++
	return true
}

func glob_generation_finish(runner Glob_Runner_Handle) {
	Glob_Runner_Handle_Invariants(runner, "glob_generation_finish.runner")
	if runner.Result != nil {
		runner.Done = true
		return
	}
	old_current := runner.Current
	runner.Current = Glob_Current_Paths(runner.Next)
	runner.Next = Glob_Next_Paths(old_current)
	runner.Current_Count = Glob_Current_Count(runner.Next_Count)
	runner.Next_Count = 0
	runner.Candidate_Index = 0
	if runner.Segment_Final {
		path_storage_sort(Path_Storage(runner.Current[:runner.Current_Count]))
		runner.Done = true
		return
	}
	if runner.Current_Count == 0 {
		path_storage_sort(Path_Storage(runner.Current[:runner.Current_Count]))
		runner.Done = true
		return
	}
	glob_segment_advance(runner)
}

func glob_segment_advance(runner Glob_Runner_Handle) {
	Glob_Runner_Handle_Invariants(runner, "glob_segment_advance.runner")
	start := int(runner.Segment_End)
	for start < len(runner.Pattern) && runner.Pattern[start] == SEPARATOR {
		start++
	}
	end := start
	for end < len(runner.Pattern) && runner.Pattern[end] != SEPARATOR {
		end++
	}
	next := end
	for next < len(runner.Pattern) && runner.Pattern[next] == SEPARATOR {
		next++
	}
	runner.Segment_Start = Glob_Segment_Start(start)
	runner.Segment_End = Glob_Segment_End(end)
	runner.Segment_Final = Glob_Segment_Final(next == len(runner.Pattern))
	runner.Segment_Meta = Glob_Segment_Meta(path_has_meta(runner.Pattern[start:end]))
}

func path_has_meta(value Text) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "path_has_meta.yes") }()
	Text_Invariants(value, "path_has_meta.value")
	for index := range value {
		switch value[index] {
		case '*', '?', '[', '\\':
			return true
		}
	}
	return false
}

// Walk_Entry is link-aware metadata passed beside caller-owned path view.
type Walk_Entry struct {
	// Name borrows final component from visited path.
	Name bytes.Slice
	// Status carries no-follow injected metadata.
	Status nbio.File_Status
}

// Walk_Entry_Invariants bounds borrowed name.
func Walk_Entry_Invariants(value Walk_Entry, namespace aver.Namespace) {
	bytes.Slice_Invariants(value.Name, namespace)
}

// Walk_Visitor_State keeps callback state concrete without interface or generic ownership.
type Walk_Visitor_State struct {
	// Pointer borrows caller state for walk lifetime.
	Pointer unsafe.Pointer
}

// Walk_Visitor_State_Invariants admits zero runner storage before initialization.
func Walk_Visitor_State_Invariants(value Walk_Visitor_State, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(value.Pointer != nil, "Walk visitor state is bound.").
		Ensure()
}

// Walk_Function receives explicit state so runner retains no allocating closure.
type Walk_Function func(
	state Walk_Visitor_State, path bytes.Slice, entry Walk_Entry, err error,
) (visit_err error)

// Walk_Queue_Paths owns paths awaiting visit.
type Walk_Queue_Paths Path_Storage

// Walk_Queue_Paths_Invariants bounds queued path collection.
func Walk_Queue_Paths_Invariants(value Walk_Queue_Paths, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), FILESYSTEM_PATH_COUNT_MINIMUM, FILESYSTEM_PATH_COUNT_MAXIMUM).
		Ensure()
}

// Walk_Child_Paths owns one directory children until sort.
type Walk_Child_Paths Path_Storage

// Walk_Child_Paths_Invariants bounds active child collection.
func Walk_Child_Paths_Invariants(value Walk_Child_Paths, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), FILESYSTEM_PATH_COUNT_MINIMUM, FILESYSTEM_PATH_COUNT_MAXIMUM).
		Ensure()
}

// Walk_Queue_Count bounds populated queued paths.
type Walk_Queue_Count int

// Walk_Queue_Count_Invariants spans empty through caller path bound.
func Walk_Queue_Count_Invariants(value Walk_Queue_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), FILESYSTEM_PATH_COUNT_MINIMUM, FILESYSTEM_PATH_COUNT_MAXIMUM).
		Ensure()
}

// Walk_Child_Count bounds populated active children.
type Walk_Child_Count int

// Walk_Child_Count_Invariants spans empty through caller path bound.
func Walk_Child_Count_Invariants(value Walk_Child_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), FILESYSTEM_PATH_COUNT_MINIMUM, FILESYSTEM_PATH_COUNT_MAXIMUM).
		Ensure()
}

// Walk_Work_Ready separates pending IO from queued continuation.
type Walk_Work_Ready bool

// Walk_Work_Ready_Invariants proves both async lifecycle states occur.
func Walk_Work_Ready_Invariants(value Walk_Work_Ready, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Walk callback queued continuation.").
		Ensure()
}

// Walk_Done separates active and terminal runner.
type Walk_Done bool

// Walk_Done_Invariants proves active and terminal runner states occur.
func Walk_Done_Invariants(value Walk_Done, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Walk runner stopped.").
		Ensure()
}

// Walk_Memory states queued paths, one directory children, and backend directory pass storage.
type Walk_Memory struct {
	// Queue owns paths still awaiting visit.
	Queue Walk_Queue_Paths
	// Children owns one directory children until whole collection sorts.
	Children Walk_Child_Paths
	// Entries receives one directory pass.
	Entries Directory_Entries
	// Directory_Buffer receives backend records for one pass.
	Directory_Buffer Directory_Buffer
}

// Walk_Memory_Invariants states each retained collection has usable bounded storage.
func Walk_Memory_Invariants(value Walk_Memory, namespace aver.Namespace) {
	Walk_Queue_Paths_Invariants(value.Queue, namespace)
	Walk_Child_Paths_Invariants(value.Children, namespace)
	Directory_Entries_Invariants(value.Entries, namespace)
	Directory_Buffer_Invariants(value.Directory_Buffer, namespace)
	aver.Always(len(value.Queue) > 0, "Walk has queued path capacity.")
	aver.Always(len(value.Children) > 0, "Walk has child path capacity.")
	aver.Always(len(value.Entries) > 0, "Walk has directory-entry capacity.")
	aver.Always(len(value.Directory_Buffer) > 0,
		"Walk has directory-record capacity.")
}

// Walk_Phase identifies directory operation whose completion is queued.
type Walk_Phase uint8

// Walk_Phase_Invariants accepts exactly idle, open, read, and close states.
func Walk_Phase_Invariants(value Walk_Phase, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(
			uint8(value), uint8(WALK_PHASE_IDLE), uint8(WALK_PHASE_OPEN),
			uint8(WALK_PHASE_READ), uint8(WALK_PHASE_CLOSE),
		).
		Ensure()
}

// WALK_PHASE_IDLE means no injected operation is submitted.
const WALK_PHASE_IDLE Walk_Phase = 0

// WALK_PHASE_OPEN means directory open is submitted.
const WALK_PHASE_OPEN Walk_Phase = WALK_PHASE_IDLE + NONEMPTY_SIZE_MINIMUM

// WALK_PHASE_READ means directory entry read is submitted.
const WALK_PHASE_READ Walk_Phase = WALK_PHASE_OPEN + NONEMPTY_SIZE_MINIMUM

// WALK_PHASE_CLOSE means directory close is submitted.
const WALK_PHASE_CLOSE Walk_Phase = WALK_PHASE_READ + NONEMPTY_SIZE_MINIMUM

// Walk_Runner walks injected filesystem without driver ownership. Completion stays first for
// same intrusive static-callback ownership as Glob_Runner.
type Walk_Runner struct {
	// Completion stays first so static callback recovers runner with no retained closure.
	Completion nbio.Completion
	// Loop submits storage work while composition root alone owns driver.
	Loop nbio.IO
	// Visitor_State remains explicit callback state.
	Visitor_State Walk_Visitor_State
	// Visitor observes each path and filters errors.
	Visitor Walk_Function
	// Queue owns paths still awaiting visit.
	Queue Walk_Queue_Paths
	// Children owns active directory children until whole collection sorts.
	Children Walk_Child_Paths
	// Entries receives one directory pass.
	Entries Directory_Entries
	// Directory_Buffer receives backend records for one pass.
	Directory_Buffer Directory_Buffer
	// Queue_Count bounds populated queued paths.
	Queue_Count Walk_Queue_Count
	// Children_Count bounds populated active children.
	Children_Count Walk_Child_Count
	// Directory_Path retains active directory across async completions.
	Directory_Path Slice
	// Directory_Status retains first visitor metadata for error callback.
	Directory_Status nbio.File_Status
	// Directory is descriptor held between open and close completions.
	Directory nbio.File
	// Phase identifies submitted directory operation.
	Phase Walk_Phase
	// Work_Ready reports callback queued continuation.
	Work_Ready Walk_Work_Ready
	// Done reports terminal state.
	Done Walk_Done
	// Result retains terminal visitor or filesystem error.
	Result error
}

// Walk_Runner_Invariants states complete runner scalar domains.
func Walk_Runner_Invariants(value Walk_Runner, namespace aver.Namespace) {
	nbio.IO_Invariants(value.Loop, namespace)
	Walk_Visitor_State_Invariants(value.Visitor_State, namespace)
	Walk_Queue_Paths_Invariants(value.Queue, namespace)
	Walk_Child_Paths_Invariants(value.Children, namespace)
	Directory_Entries_Invariants(value.Entries, namespace)
	Directory_Buffer_Invariants(value.Directory_Buffer, namespace)
	Walk_Queue_Count_Invariants(value.Queue_Count, namespace)
	Walk_Child_Count_Invariants(value.Children_Count, namespace)
	Slice_Invariants(value.Directory_Path, namespace)
	Walk_Phase_Invariants(value.Phase, namespace)
	Walk_Work_Ready_Invariants(value.Work_Ready, namespace)
	Walk_Done_Invariants(value.Done, namespace)
}

// Walk_Runner_Storage_Handle admits zero caller storage before initialization.
type Walk_Runner_Storage_Handle *Walk_Runner

// Walk_Runner_Storage_Handle_Invariants requires writable caller storage.
func Walk_Runner_Storage_Handle_Invariants(
	value Walk_Runner_Storage_Handle, namespace aver.Namespace,
) {
	if value == nil {
		return
	}
	Walk_Runner_Invariants(*value, namespace)
}

// Walk_Runner_Handle keeps initialized caller-owned walk state nonnil.
type Walk_Runner_Handle *Walk_Runner

// Walk_Runner_Handle_Invariants states complete runner scalar domains.
func Walk_Runner_Handle_Invariants(value Walk_Runner_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Walk_Runner_Invariants(*value, namespace)
}

// Walk_Runner_Init binds standard Walk behavior to injected filesystem.
func Walk_Runner_Init(
	runner Walk_Runner_Storage_Handle, loop nbio.IO, root Text,
	visitor_state Walk_Visitor_State, visitor Walk_Function, memory Walk_Memory,
) (err error) {
	Walk_Runner_Storage_Handle_Invariants(runner, "walk_runner_init.runner")
	nbio.IO_Invariants(loop, "walk_runner_init.loop")
	Text_Invariants(root, "walk_runner_init.root")
	Walk_Visitor_State_Invariants(visitor_state, "walk_runner_init.visitor_state")
	Walk_Memory_Invariants(memory, "walk_runner_init.memory")
	return walk_runner_init(runner, loop, root, visitor_state, visitor, memory)
}

// Walk_Directory_Runner_Init binds standard WalkDir behavior. Current injected entry already
// carries kind, while callback contract remains same link-aware Walk_Entry.
func Walk_Directory_Runner_Init(
	runner Walk_Runner_Storage_Handle, loop nbio.IO, root Text,
	visitor_state Walk_Visitor_State, visitor Walk_Function, memory Walk_Memory,
) (err error) {
	Walk_Runner_Storage_Handle_Invariants(runner, "walk_directory_runner_init.runner")
	nbio.IO_Invariants(loop, "walk_directory_runner_init.loop")
	Text_Invariants(root, "walk_directory_runner_init.root")
	Walk_Visitor_State_Invariants(
		visitor_state, "walk_directory_runner_init.visitor_state",
	)
	Walk_Memory_Invariants(memory, "walk_directory_runner_init.memory")
	return walk_runner_init(runner, loop, root, visitor_state, visitor, memory)
}

func walk_runner_init(
	runner Walk_Runner_Storage_Handle, loop nbio.IO, root Text,
	visitor_state Walk_Visitor_State, visitor Walk_Function, memory Walk_Memory,
) (err error) {
	Walk_Runner_Storage_Handle_Invariants(runner, "walk_runner_init_internal.runner")
	nbio.IO_Invariants(loop, "walk_runner_init_internal.loop")
	Text_Invariants(root, "walk_runner_init_internal.root")
	Walk_Visitor_State_Invariants(visitor_state, "walk_runner_init_internal.visitor_state")
	Walk_Memory_Invariants(memory, "walk_runner_init_internal.memory")
	aver.Always(
		visitor_state.Pointer != nil, "Walk initialization requires visitor state.",
	)
	aver.Always(visitor != nil, "A Walk_Runner has visitor.")
	path_storage_slots_validate(Path_Storage(memory.Queue))
	path_storage_slots_validate(Path_Storage(memory.Children))
	directory_buffer_validate(memory.Directory_Buffer)
	*runner = Walk_Runner{
		Loop: loop, Visitor_State: visitor_state, Visitor: visitor,
		Queue: memory.Queue, Children: memory.Children, Entries: memory.Entries,
		Directory_Buffer: memory.Directory_Buffer,
	}
	path, copy_err := path_storage_write_text(Path_Storage(runner.Queue), 0, root)
	if copy_err != nil {
		runner.Done = true
		runner.Result = copy_err
		return copy_err
	}
	runner.Queue[0] = path
	runner.Queue_Count = 1
	return nil
}

// Walk_Runner_Rearm executes one continuation or synchronous visit.
func Walk_Runner_Rearm(runner Walk_Runner_Handle) (rearm Boolean) {
	defer func() { Boolean_Invariants(rearm, "walk_runner_rearm.rearm") }()
	Walk_Runner_Handle_Invariants(runner, "walk_runner_rearm.runner")
	if runner.Done {
		return false
	}
	if runner.Work_Ready {
		runner.Work_Ready = false
		return walk_completion_apply(runner)
	}
	if runner.Queue_Count == 0 {
		runner.Done = true
		return false
	}
	return walk_visit_next(runner)
}

// Walk_Runner_Work_Queued reports callback stored one continuation.
func Walk_Runner_Work_Queued(runner Walk_Runner_Handle) (queued Boolean) {
	defer func() { Boolean_Invariants(queued, "walk_runner_work_queued.queued") }()
	Walk_Runner_Handle_Invariants(runner, "walk_runner_work_queued.runner")
	return Boolean(runner.Work_Ready)
}

// Walk_Runner_Stopped reports walk terminal state.
func Walk_Runner_Stopped(runner Walk_Runner_Handle) (stopped Boolean) {
	defer func() { Boolean_Invariants(stopped, "walk_runner_stopped.stopped") }()
	Walk_Runner_Handle_Invariants(runner, "walk_runner_stopped.runner")
	return Boolean(runner.Done)
}

// Walk_Runner_Status reports terminal visitor or filesystem error.
func Walk_Runner_Status(runner Walk_Runner_Handle) (err error) {
	Walk_Runner_Handle_Invariants(runner, "walk_runner_status.runner")
	return runner.Result
}

func walk_io_complete(completion nbio.Completion_Handle) {
	runner := Walk_Runner_Handle(unsafe.Pointer(completion))
	runner.Work_Ready = true
}

func walk_visit_next(runner Walk_Runner_Handle) (rearm Boolean) {
	defer func() { Boolean_Invariants(rearm, "walk_visit_next.rearm") }()
	Walk_Runner_Handle_Invariants(runner, "walk_visit_next.runner")
	runner.Queue_Count--
	path := runner.Queue[runner.Queue_Count]
	path_text := string(path_slice_text(path))
	status, status_err := nbio.Storage_Status(runner.Loop.Storage, path_text)
	if status_err != nil {
		visit_err := runner.Visitor(
			runner.Visitor_State, bytes.Slice(path), Walk_Entry{}, status_err,
		)
		return walk_visit_result(runner, path, status, visit_err)
	}
	if !status.Exists {
		if status_err == nil {
			status_err = Error_Path_Absent
		}
		visit_err := runner.Visitor(
			runner.Visitor_State, bytes.Slice(path), Walk_Entry{}, status_err,
		)
		return walk_visit_result(runner, path, status, visit_err)
	}
	entry := Walk_Entry{Name: bytes.Slice(path_base(path)), Status: status}
	visit_err := runner.Visitor(runner.Visitor_State, bytes.Slice(path), entry, nil)
	if visit_err != nil {
		return walk_visit_result(runner, path, status, visit_err)
	}
	if !nbio.File_Mode_Is_Directory(status.Mode) {
		return true
	}
	runner.Directory_Path = path
	runner.Directory_Status = status
	runner.Children_Count = 0
	runner.Phase = WALK_PHASE_OPEN
	runner.Work_Ready = false
	nbio.Storage_Open_At(
		runner.Loop.Storage, &runner.Completion, nbio.DIRECTORY_CURRENT,
		string(path_slice_text(path)), nbio.Open_At_Options{Access: nbio.OPEN_READ_ONLY},
		nbio.Callback(walk_io_complete),
	)
	return false
}

func walk_visit_result(
	runner Walk_Runner_Handle, visited Slice, status nbio.File_Status, visit_err error,
) (rearm Boolean) {
	defer func() { Boolean_Invariants(rearm, "walk_visit_result.rearm") }()
	Walk_Runner_Handle_Invariants(runner, "walk_visit_result.runner")
	Slice_Invariants(visited, "walk_visit_result.visited")
	if visit_err == nil {
		return true
	}
	if visit_err == Skip_All {
		runner.Done = true
		return false
	}
	if visit_err == Skip_Directory {
		if !nbio.File_Mode_Is_Directory(status.Mode) {
			walk_queue_siblings_remove(runner, visited)
		}
		return true
	}
	runner.Result = visit_err
	runner.Done = true
	return false
}

func walk_completion_apply(runner Walk_Runner_Handle) (rearm Boolean) {
	defer func() { Boolean_Invariants(rearm, "walk_completion_apply.rearm") }()
	Walk_Runner_Handle_Invariants(runner, "walk_completion_apply.runner")
	aver.Always(
		runner.Phase != WALK_PHASE_IDLE,
		"Filepath Walk completion has pending operation.",
	)
	switch runner.Phase {
	case WALK_PHASE_OPEN:
		if runner.Completion.Error != nil {
			return walk_directory_error(runner, runner.Completion.Error)
		}
		runner.Directory = nbio.File(runner.Completion.Data)
		walk_directory_read(runner)
		return false
	case WALK_PHASE_READ:
		if runner.Completion.Error != nil {
			walk_directory_close(runner)
			runner.Result = runner.Completion.Error
			return false
		}
		if runner.Completion.Data == 0 {
			walk_directory_close(runner)
			return false
		}
		walk_entries_collect(runner, Path_Count(runner.Completion.Data))
		if runner.Result != nil {
			walk_directory_close(runner)
			return false
		}
		walk_directory_read(runner)
		return false
	case WALK_PHASE_CLOSE:
		if runner.Result != nil {
			return walk_directory_error(runner, runner.Result)
		}
		walk_children_push(runner)
		runner.Phase = WALK_PHASE_IDLE
		return !Boolean(runner.Done)
	}
	return false
}

func walk_directory_read(runner Walk_Runner_Handle) {
	Walk_Runner_Handle_Invariants(runner, "walk_directory_read.runner")
	runner.Phase = WALK_PHASE_READ
	runner.Work_Ready = false
	nbio.Storage_Get_Directory_Entries(
		runner.Loop.Storage, &runner.Completion, runner.Directory,
		runner.Directory_Buffer, runner.Entries, nbio.Callback(walk_io_complete),
	)
}

func walk_directory_close(runner Walk_Runner_Handle) {
	Walk_Runner_Handle_Invariants(runner, "walk_directory_close.runner")
	runner.Phase = WALK_PHASE_CLOSE
	runner.Work_Ready = false
	nbio.IO_Close(
		runner.Loop, &runner.Completion, runner.Directory, nbio.Callback(walk_io_complete),
	)
}

func walk_directory_error(
	runner Walk_Runner_Handle, directory_err error,
) (rearm Boolean) {
	defer func() { Boolean_Invariants(rearm, "walk_directory_error.rearm") }()
	Walk_Runner_Handle_Invariants(runner, "walk_directory_error.runner")
	path := runner.Directory_Path
	entry := Walk_Entry{
		Name: bytes.Slice(path_base(path)), Status: runner.Directory_Status,
	}
	visit_err := runner.Visitor(
		runner.Visitor_State, bytes.Slice(path), entry, directory_err,
	)
	runner.Result = nil
	runner.Phase = WALK_PHASE_IDLE
	return walk_visit_result(runner, path, runner.Directory_Status, visit_err)
}

func walk_entries_collect(runner Walk_Runner_Handle, count Path_Count) {
	Walk_Runner_Handle_Invariants(runner, "walk_entries_collect.runner")
	Path_Count_Invariants(count, "walk_entries_collect.count")
	aver.Always(count > 0, "Walk directory completion has entries.")
	aver.Always(int(count) <= len(runner.Entries),
		"Walk directory completion count fits caller entries.")
	parent := runner.Directory_Path
	for index := 0; index < int(count); index++ {
		child, join_err := path_storage_join(
			Path_Storage(runner.Children), Path_Count(runner.Children_Count), parent,
			Text(runner.Entries[index].Name),
		)
		if join_err != nil {
			runner.Result = join_err
			return
		}
		runner.Children[runner.Children_Count] = child
		runner.Children_Count++
	}
}

func walk_children_push(runner Walk_Runner_Handle) {
	Walk_Runner_Handle_Invariants(runner, "walk_children_push.runner")
	path_storage_sort(Path_Storage(runner.Children[:runner.Children_Count]))
	for index := runner.Children_Count; index > 0; index-- {
		if int(runner.Queue_Count) == len(runner.Queue) {
			runner.Result = Error_Result_Too_Large
			runner.Done = true
			return
		}
		path, copy_err := path_storage_write(
			Path_Storage(runner.Queue), Path_Count(runner.Queue_Count),
			runner.Children[index-1],
		)
		if copy_err != nil {
			runner.Result = copy_err
			runner.Done = true
			return
		}
		runner.Queue[runner.Queue_Count] = path
		runner.Queue_Count++
	}
	runner.Children_Count = 0
}

func walk_queue_siblings_remove(
	runner Walk_Runner_Handle, visited Slice,
) {
	Walk_Runner_Handle_Invariants(runner, "walk_queue_siblings_remove.runner")
	Slice_Invariants(visited, "walk_queue_siblings_remove.visited")
	parent := path_parent(visited)
	for runner.Queue_Count > 0 {
		candidate := runner.Queue[runner.Queue_Count-1]
		if !bytes.Equal(
			bytes.Slice(path_parent(candidate)), bytes.Slice(parent),
		) {
			return
		}
		runner.Queue_Count--
	}
}

// Eval_Symlinks_Into resolves final links through injected synchronous metadata readers. Two
// caller scratch buffers preserve remainder and link target without hidden heap ownership.
func Eval_Symlinks_Into(
	storage nbio.Storage, destination Slice, remainder Slice,
	link_target Slice, value Text,
) (count Boundary, err error) {
	defer func() {
		Boundary_Invariants(count, "eval_symlinks_into.count")
	}()
	Slice_Invariants(destination, "eval_symlinks_into.destination")
	Slice_Invariants(remainder, "eval_symlinks_into.remainder")
	Slice_Invariants(link_target, "eval_symlinks_into.link_target")
	Text_Invariants(value, "eval_symlinks_into.value")
	aver.Always(len(destination) == PATH_SIZE_MAXIMUM,
		"Eval_Symlinks destination has exact path capacity.")
	aver.Always(len(remainder) == PATH_SIZE_MAXIMUM,
		"Eval_Symlinks remainder storage has exact path capacity.")
	aver.Always(len(link_target) == PATH_SIZE_MAXIMUM,
		"Eval_Symlinks link storage has exact target capacity.")
	return eval_symlinks_apply(storage, destination, remainder, link_target, value)
}

// Eval_Link_Count bounds followed symbolic links.
type Eval_Link_Count int

// Eval_Link_Count_Invariants spans first lookup through rejected cycle boundary.
func Eval_Link_Count_Invariants(value Eval_Link_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(
			int(value), SYMBOLIC_LINK_COUNT_MINIMUM, SYMBOLIC_LINK_COUNT_BOUNDARY,
		).
		Ensure()
}

// Eval_Done separates active and terminal resolution.
type Eval_Done bool

// Eval_Done_Invariants proves both resolution lifecycle states occur.
func Eval_Done_Invariants(value Eval_Done, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "Symbolic-link resolution stopped.").
		Ensure()
}

// Eval_Destination holds resolved components.
type Eval_Destination bytes.Slice

// Eval_Destination_Invariants requires exact caller path storage.
func Eval_Destination_Invariants(value Eval_Destination, _ aver.Namespace) {
	aver.Always(len(value) == PATH_SIZE_MAXIMUM,
		"Eval destination has exact path capacity.")
}

// Eval_Remainder holds unresolved components.
type Eval_Remainder bytes.Slice

// Eval_Remainder_Invariants requires exact caller path storage.
func Eval_Remainder_Invariants(value Eval_Remainder, _ aver.Namespace) {
	aver.Always(len(value) == PATH_SIZE_MAXIMUM,
		"Eval remainder has exact path capacity.")
}

// Eval_Link_Target receives one injected readlink result.
type Eval_Link_Target bytes.Slice

// Eval_Link_Target_Invariants requires exact caller path storage.
func Eval_Link_Target_Invariants(value Eval_Link_Target, _ aver.Namespace) {
	aver.Always(len(value) == PATH_SIZE_MAXIMUM,
		"Eval link target has exact path capacity.")
}

// Eval_Remainder_Count bounds unresolved bytes.
type Eval_Remainder_Count bytes.Boundary

// Eval_Remainder_Count_Invariants spans empty through full path.
func Eval_Remainder_Count_Invariants(
	value Eval_Remainder_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), PATH_SIZE_MINIMUM, PATH_SIZE_MAXIMUM).
		Ensure()
}

// Eval_Destination_Count bounds resolved bytes.
type Eval_Destination_Count bytes.Boundary

// Eval_Destination_Count_Invariants spans empty through full path.
func Eval_Destination_Count_Invariants(
	value Eval_Destination_Count, namespace aver.Namespace,
) {
	aver.Tree(value, namespace).
		Range_Int(int(value), PATH_SIZE_MINIMUM, PATH_SIZE_MAXIMUM).
		Ensure()
}

// Eval_Start identifies unresolved component first byte.
type Eval_Start bytes.Boundary

// Eval_Start_Invariants spans path boundaries.
func Eval_Start_Invariants(value Eval_Start, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), PATH_SIZE_MINIMUM, PATH_SIZE_MAXIMUM).
		Ensure()
}

// Eval_End identifies boundary after unresolved component.
type Eval_End bytes.Boundary

// Eval_End_Invariants spans path boundaries.
func Eval_End_Invariants(value Eval_End, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), PATH_SIZE_MINIMUM, PATH_SIZE_MAXIMUM).
		Ensure()
}

// Eval_State retains one caller-owned symbolic-link resolution.
type Eval_State struct {
	// Storage provides no-follow metadata and link targets.
	Storage nbio.Storage
	// Destination holds resolved path.
	Destination Eval_Destination
	// Remainder holds unresolved path.
	Remainder Eval_Remainder
	// Link_Target receives one link target.
	Link_Target Eval_Link_Target
	// Remainder_Count bounds unresolved bytes.
	Remainder_Count Eval_Remainder_Count
	// Destination_Count bounds resolved bytes.
	Destination_Count Eval_Destination_Count
	// Start identifies current component first byte.
	Start Eval_Start
	// End identifies boundary after current component.
	End Eval_End
	// Link_Count bounds followed symbolic links.
	Link_Count Eval_Link_Count
	// Done reports terminal resolution.
	Done Eval_Done
	// Result retains terminal error.
	Result error
}

// Eval_State_Invariants states retained buffer and scalar bounds.
func Eval_State_Invariants(value Eval_State, namespace aver.Namespace) {
	Eval_Destination_Invariants(value.Destination, namespace)
	Eval_Remainder_Invariants(value.Remainder, namespace)
	Eval_Link_Target_Invariants(value.Link_Target, namespace)
	Eval_Remainder_Count_Invariants(value.Remainder_Count, namespace)
	Eval_Destination_Count_Invariants(value.Destination_Count, namespace)
	Eval_Start_Invariants(value.Start, namespace)
	Eval_End_Invariants(value.End, namespace)
	Eval_Link_Count_Invariants(value.Link_Count, namespace)
	Eval_Done_Invariants(value.Done, namespace)
}

// Eval_State_Handle keeps active caller-owned symbolic-link state nonnil.
type Eval_State_Handle *Eval_State

// Eval_State_Handle_Invariants states retained buffer and scalar bounds.
func Eval_State_Handle_Invariants(value Eval_State_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Eval_State_Invariants(*value, namespace)
}

func eval_symlinks_apply(
	storage nbio.Storage, destination Slice, remainder Slice,
	link_target Slice, value Text,
) (count Boundary, err error) {
	defer func() {
		Boundary_Invariants(count, "eval_symlinks_apply.count")
	}()
	Slice_Invariants(destination, "eval_symlinks_apply.destination")
	Slice_Invariants(remainder, "eval_symlinks_apply.remainder")
	Slice_Invariants(link_target, "eval_symlinks_apply.link_target")
	Text_Invariants(value, "eval_symlinks_apply.value")
	state := Eval_State{
		Storage: storage, Destination: Eval_Destination(destination),
		Remainder: Eval_Remainder(remainder), Link_Target: Eval_Link_Target(link_target),
		Remainder_Count: Eval_Remainder_Count(copy(remainder, value)),
	}
	if len(value) > 0 {
		if value[0] == SEPARATOR {
			state.Destination[0] = SEPARATOR
			state.Destination_Count = 1
		}
	}
	for !state.Done {
		eval_state_step(&state)
	}
	if state.Result != nil {
		return 0, state.Result
	}
	clean_count := Clean_Into(
		bytes.Slice(state.Remainder),
		path_slice_text(Slice(state.Destination[:state.Destination_Count])),
	)
	copy(state.Destination, state.Remainder[:clean_count])
	return Boundary(clean_count), nil
}

func eval_state_step(state Eval_State_Handle) {
	Eval_State_Handle_Invariants(state, "eval_state_step.state")
	eval_component_find(state)
	if state.Done {
		return
	}
	if eval_component_dot(state) {
		state.Start = Eval_Start(state.End)
		return
	}
	if eval_component_dot_dot(state) {
		eval_parent_apply(state)
		state.Start = Eval_Start(state.End)
		return
	}
	eval_component_append(state)
	if state.Result != nil {
		state.Done = true
		return
	}
	status, status_err := nbio.Storage_Status(
		state.Storage,
		string(path_slice_text(Slice(
			state.Destination[:state.Destination_Count],
		))),
	)
	if status_err != nil {
		state.Result = status_err
		state.Done = true
		return
	}
	eval_status_apply(state, status)
}

func eval_component_find(state Eval_State_Handle) {
	Eval_State_Handle_Invariants(state, "eval_component_find.state")
	for int(state.Start) < int(state.Remainder_Count) {
		if state.Remainder[state.Start] != SEPARATOR {
			break
		}
		state.Start++
	}
	state.End = Eval_End(state.Start)
	for int(state.End) < int(state.Remainder_Count) {
		if state.Remainder[state.End] == SEPARATOR {
			break
		}
		state.End++
	}
	if int(state.End) == int(state.Start) {
		state.Done = true
	}
}

func eval_status_apply(state Eval_State_Handle, status nbio.File_Status) {
	Eval_State_Handle_Invariants(state, "eval_status_apply.state")
	if !status.Exists {
		state.Result = Error_Path_Absent
		state.Done = true
		return
	}
	if !nbio.File_Mode_Is_Symbolic_Link(status.Mode) {
		if !nbio.File_Mode_Is_Directory(status.Mode) {
			if eval_has_component(state) {
				state.Result = Error_Not_Directory
				state.Done = true
				return
			}
		}
		state.Start = Eval_Start(state.End)
		return
	}
	state.Link_Count++
	if state.Link_Count > SYMBOLIC_LINK_COUNT_MAXIMUM {
		state.Result = Error_Too_Many_Links
		state.Done = true
		return
	}
	eval_link_apply(state)
	state.Start = 0
}

func eval_link_apply(state Eval_State_Handle) {
	Eval_State_Handle_Invariants(state, "eval_link_apply.state")
	target_count, read_err := nbio.Storage_Read_Link(
		state.Storage,
		string(path_slice_text(Slice(
			state.Destination[:state.Destination_Count],
		))),
		state.Link_Target,
	)
	if read_err != nil {
		state.Result = read_err
		state.Done = true
		return
	}
	rest_count := int(state.Remainder_Count) - int(state.End)
	if target_count+rest_count > PATH_SIZE_MAXIMUM {
		state.Result = Error_Result_Too_Large
		state.Done = true
		return
	}
	copy(
		state.Remainder[target_count:target_count+rest_count],
		state.Remainder[state.End:state.Remainder_Count],
	)
	copy(state.Remainder[:target_count], state.Link_Target[:target_count])
	state.Remainder_Count = Eval_Remainder_Count(target_count + rest_count)
	eval_component_remove(state)
	if target_count > 0 {
		if state.Remainder[0] == SEPARATOR {
			state.Destination[0] = SEPARATOR
			state.Destination_Count = 1
		}
	}
}

func eval_component_dot(state Eval_State_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "eval_component_dot.yes") }()
	Eval_State_Handle_Invariants(state, "eval_component_dot.state")
	component := state.Remainder[state.Start:state.End]
	if len(component) != 1 {
		return false
	}
	return Boolean(component[0] == '.')
}

func eval_component_dot_dot(state Eval_State_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "eval_component_dot_dot.yes") }()
	Eval_State_Handle_Invariants(state, "eval_component_dot_dot.state")
	component := state.Remainder[state.Start:state.End]
	if len(component) != 2 {
		return false
	}
	if component[0] != '.' {
		return false
	}
	return Boolean(component[1] == '.')
}

func eval_component_append(state Eval_State_Handle) {
	Eval_State_Handle_Invariants(state, "eval_component_append.state")
	component := state.Remainder[state.Start:state.End]
	separator := state.Destination_Count > 0
	if separator {
		separator = state.Destination[state.Destination_Count-1] != SEPARATOR
	}
	needed_count := len(component)
	if separator {
		needed_count++
	}
	if int(state.Destination_Count)+needed_count > PATH_SIZE_MAXIMUM {
		state.Result = Error_Result_Too_Large
		return
	}
	if separator {
		state.Destination[state.Destination_Count] = SEPARATOR
		state.Destination_Count++
	}
	state.Destination_Count += Eval_Destination_Count(
		copy(state.Destination[state.Destination_Count:], component),
	)
}

func eval_component_remove(state Eval_State_Handle) {
	Eval_State_Handle_Invariants(state, "eval_component_remove.state")
	if state.Destination_Count == 0 {
		return
	}
	end_count := state.Destination_Count
	if end_count > 1 {
		if state.Destination[end_count-1] == SEPARATOR {
			end_count--
		}
	}
	for end_count > 0 {
		if state.Destination[end_count-1] == SEPARATOR {
			break
		}
		end_count--
	}
	if end_count == 1 {
		if state.Destination[0] == SEPARATOR {
			state.Destination_Count = 1
			return
		}
	}
	if end_count > 0 {
		state.Destination_Count = end_count - 1
		return
	}
	state.Destination_Count = 0
}

func eval_parent_apply(state Eval_State_Handle) {
	Eval_State_Handle_Invariants(state, "eval_parent_apply.state")
	if state.Destination_Count == 0 {
		state.Destination[0] = '.'
		state.Destination[1] = '.'
		state.Destination_Count = 2
		return
	}
	base := path_base(Slice(state.Destination[:state.Destination_Count]))
	if len(base) == 2 {
		if base[0] == '.' {
			if base[1] == '.' {
				if state.Destination_Count+3 > PATH_SIZE_MAXIMUM {
					return
				}
				state.Destination[state.Destination_Count] = SEPARATOR
				state.Destination[state.Destination_Count+1] = '.'
				state.Destination[state.Destination_Count+2] = '.'
				state.Destination_Count += 3
				return
			}
		}
	}
	eval_component_remove(state)
}

func eval_has_component(state Eval_State_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "eval_has_component.yes") }()
	Eval_State_Handle_Invariants(state, "eval_has_component.state")
	for index := int(state.End); index < int(state.Remainder_Count); index++ {
		if state.Remainder[index] != SEPARATOR {
			return true
		}
	}
	return false
}

func path_storage_write(
	storage Path_Storage, index Path_Count, value Slice,
) (written Slice, err error) {
	defer func() { Slice_Invariants(written, "path_storage_write.written") }()
	Path_Storage_Invariants(storage, "path_storage_write.storage")
	Path_Count_Invariants(index, "path_storage_write.index")
	Slice_Invariants(value, "path_storage_write.value")
	if int(index) >= len(storage) {
		return nil, Error_Result_Too_Large
	}
	if len(value) > cap(storage[index]) {
		return nil, Error_Result_Too_Large
	}
	if len(value) > PATH_SIZE_MAXIMUM {
		return nil, Error_Result_Too_Large
	}
	written = Slice(storage[index][:len(value)])
	copy(written, value)
	return written, nil
}

func path_storage_write_text(
	storage Path_Storage, index Path_Count, value Text,
) (written Slice, err error) {
	defer func() {
		Slice_Invariants(written, "path_storage_write_text.written")
	}()
	Path_Storage_Invariants(storage, "path_storage_write_text.storage")
	Path_Count_Invariants(index, "path_storage_write_text.index")
	Text_Invariants(value, "path_storage_write_text.value")
	if int(index) >= len(storage) {
		return nil, Error_Result_Too_Large
	}
	if len(value) > cap(storage[index]) {
		return nil, Error_Result_Too_Large
	}
	if len(value) > PATH_SIZE_MAXIMUM {
		return nil, Error_Result_Too_Large
	}
	written = Slice(storage[index][:len(value)])
	copy(written, value)
	return written, nil
}

func path_storage_slots_validate(storage Path_Storage) {
	Path_Storage_Invariants(storage, "path_storage_slots_validate.storage")
	for index := range storage {
		aver.Always(cap(storage[index]) <= PATH_SIZE_MAXIMUM,
			"A filesystem path slot stays inside path bound.")
	}
}

func directory_buffer_validate(buffer Directory_Buffer) {
	Directory_Buffer_Invariants(buffer, "directory_buffer_validate.buffer")
	aver.Always(len(buffer) >= nbio.DIRECTORY_BUFFER_SIZE_MINIMUM,
		"Filesystem runner holds at least one directory-record byte.")
}

func path_storage_join(
	storage Path_Storage, index Path_Count, parent Slice, name Text,
) (written Slice, err error) {
	defer func() { Slice_Invariants(written, "path_storage_join.written") }()
	Path_Storage_Invariants(storage, "path_storage_join.storage")
	Path_Count_Invariants(index, "path_storage_join.index")
	Slice_Invariants(parent, "path_storage_join.parent")
	Text_Invariants(name, "path_storage_join.name")
	if int(index) >= len(storage) {
		return nil, Error_Result_Too_Large
	}
	separator := len(parent) > 0 && parent[len(parent)-1] != SEPARATOR
	size := len(parent) + len(name)
	if separator {
		size++
	}
	if size > cap(storage[index]) {
		return nil, Error_Result_Too_Large
	}
	if size > PATH_SIZE_MAXIMUM {
		return nil, Error_Result_Too_Large
	}
	written = Slice(storage[index][:size])
	position := copy(written, parent)
	if separator {
		written[position] = SEPARATOR
		position++
	}
	copy(written[position:], name)
	return written, nil
}

func path_storage_sort(storage Path_Storage) {
	Path_Storage_Invariants(storage, "path_storage_sort.storage")
	for index := 1; index < len(storage); index++ {
		value := storage[index]
		position := index
		for position > 0 && bytes.Compare(
			bytes.Slice(value), bytes.Slice(storage[position-1]),
		) < 0 {
			storage[position] = storage[position-1]
			position--
		}
		storage[position] = value
	}
}

func path_slice_text(value Slice) (text Text) {
	defer func() { Text_Invariants(text, "path_slice_text.text") }()
	Slice_Invariants(value, "path_slice_text.value")
	if len(value) == 0 {
		return ""
	}
	// IO borrows dynamic caller path until completion. String view preserves same ownership;
	// copy would allocate and split lifecycle from submitted bytes.
	return Text(unsafe.String(unsafe.SliceData(value), len(value)))
}

func path_base(value Slice) (base Slice) {
	defer func() { Slice_Invariants(base, "path_base.base") }()
	Slice_Invariants(value, "path_base.value")
	if len(value) == 0 {
		return value
	}
	end_count := len(value)
	for end_count > 1 && value[end_count-1] == SEPARATOR {
		end_count--
	}
	start := end_count
	for start > 0 && value[start-1] != SEPARATOR {
		start--
	}
	return value[start:end_count]
}

func path_parent(value Slice) (parent Parent_Path) {
	defer func() { Parent_Path_Invariants(parent, "path_parent.parent") }()
	Slice_Invariants(value, "path_parent.value")
	if len(value) == 0 {
		return Parent_Path(value)
	}
	end_count := len(value)
	for end_count > 1 && value[end_count-1] == SEPARATOR {
		end_count--
	}
	for end_count > 0 && value[end_count-1] != SEPARATOR {
		end_count--
	}
	for end_count > 1 && value[end_count-1] == SEPARATOR {
		end_count--
	}
	return Parent_Path(value[:end_count])
}

// SEPARATOR stays fixed because current host contract targets Unix.
const SEPARATOR = '/'

// LIST_SEPARATOR stays fixed because current host contract targets Unix.
const LIST_SEPARATOR = ':'

// PATH_SIZE_MINIMUM keeps empty input valid.
const PATH_SIZE_MINIMUM = bytes.TEXT_SIZE_MINIMUM

// PATH_SIZE_MAXIMUM shares host pathname bound with slash path operations.
const PATH_SIZE_MAXIMUM = path.PATH_SIZE_MAXIMUM

// Text gives host pathname limits to generic text representation.
type Text bytes.Text

// Text_Invariants applies host pathname limit to text representation.
func Text_Invariants(value Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), PATH_SIZE_MINIMUM, PATH_SIZE_MAXIMUM).
		Ensure()
}

// Slice gives host pathname limits to generic byte representation.
type Slice bytes.Slice

// Slice_Invariants applies host pathname limit to byte representation.
func Slice_Invariants(value Slice, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), PATH_SIZE_MINIMUM, PATH_SIZE_MAXIMUM).
		Ensure()
}

// Boundary gives host pathname limits to generic written byte count.
type Boundary bytes.Boundary

// Boundary_Invariants applies host pathname limit to written byte count.
func Boundary_Invariants(value Boundary, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), PATH_SIZE_MINIMUM, PATH_SIZE_MAXIMUM).
		Ensure()
}

// NONEMPTY_SIZE_MINIMUM accounts for mandatory dot or path byte.
const NONEMPTY_SIZE_MINIMUM = path.NONEMPTY_SIZE_MINIMUM

// DIRECTORY_SIZE_MAXIMUM subtracts final element or trailing separator.
const DIRECTORY_SIZE_MAXIMUM = path.DIRECTORY_SIZE_MAXIMUM

// ELEMENT_COUNT_MINIMUM keeps zero-element Join_Into valid.
const ELEMENT_COUNT_MINIMUM = path.ELEMENT_COUNT_MINIMUM

// ELEMENT_COUNT_MAXIMUM bounds empty Join_Into work.
const ELEMENT_COUNT_MAXIMUM = path.ELEMENT_COUNT_MAXIMUM

// PATH_COUNT_MINIMUM keeps empty path-list result valid.
const PATH_COUNT_MINIMUM = 0

// PATH_COUNT_MAXIMUM includes one empty field around every path byte.
const PATH_COUNT_MAXIMUM = PATH_SIZE_MAXIMUM + NONEMPTY_SIZE_MINIMUM

// VOLUME_SIZE stays zero because Unix paths carry no volume prefix.
const VOLUME_SIZE = 0

// DOT_COUNT stays one because relative identity is one dot.
const DOT_COUNT = NONEMPTY_SIZE_MINIMUM

// START_MAXIMUM leaves one differing byte after segment start.
const START_MAXIMUM = PATH_SIZE_MAXIMUM - NONEMPTY_SIZE_MINIMUM

// UP_COUNT_MAXIMUM follows densest one-byte elements separated by slashes.
const UP_COUNT_MAXIMUM = (PATH_SIZE_MAXIMUM + NONEMPTY_SIZE_MINIMUM) /
	(2 * NONEMPTY_SIZE_MINIMUM)

// PART_TAIL_MAXIMUM subtracts separator accepted at first input byte.
const PART_TAIL_MAXIMUM = PATH_SIZE_MAXIMUM - NONEMPTY_SIZE_MINIMUM

// Error_Bad_Pattern keeps malformed shell grammar error stable.
var Error_Bad_Pattern = errors.New("syntax error in pattern")

// Error_Invalid_Path keeps malicious slash-path rejection stable.
var Error_Invalid_Path = errors.New("filepath: invalid path")

// Error_Result_Too_Large separates valid composition overflow from malformed input.
var Error_Result_Too_Large = errors.New("filepath: result outside bounds")

// Error_Root_Mismatch prevents relative output across incompatible roots.
var Error_Root_Mismatch = errors.New("filepath: roots do not match")

// Error_Base_Above_Root prevents lexical answer when base already escaped evaluation root.
var Error_Base_Above_Root = errors.New("filepath: base begins above root")

// Error_Working_Directory rejects injected state that cannot anchor absolute output.
var Error_Working_Directory = errors.New("filepath: working directory is not absolute")

// Boolean separates filepath decision coverage from unrelated booleans.
type Boolean bool

// Boolean_Invariants proves both filepath decision states occur.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "A filepath decision is true.").
		Ensure()
}

// Elements bounds Join_Into work even when every element is empty.
type Elements []bytes.Text

// Elements_Invariants prevents empty elements from causing unbounded work.
func Elements_Invariants(value Elements, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), ELEMENT_COUNT_MINIMUM, ELEMENT_COUNT_MAXIMUM).
		Ensure()
}

// Paths makes Split_List_Into result ownership explicit so no result slice allocates.
type Paths []bytes.Text

// Paths_Invariants prevents malicious caller slot traversal from becoming unbounded.
func Paths_Invariants(value Paths, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), PATH_COUNT_MINIMUM, PATH_COUNT_MAXIMUM).
		Ensure()
}

// Path_Count separates populated slots from destination capacity.
type Path_Count int

// Path_Count_Invariants proves empty through separator-dense list.
func Path_Count_Invariants(value Path_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), PATH_COUNT_MINIMUM, PATH_COUNT_MAXIMUM).
		Ensure()
}

// Nonempty_Count excludes zero because empty lexical result becomes dot.
type Nonempty_Count int

// Nonempty_Count_Invariants proves mandatory byte through path bound.
func Nonempty_Count_Invariants(value Nonempty_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), NONEMPTY_SIZE_MINIMUM, PATH_SIZE_MAXIMUM).
		Ensure()
}

// Directory_Count excludes zero because cleaned directory always contains dot or root.
type Directory_Count int

// Directory_Count_Invariants excludes full size because directory loses final element or slash.
func Directory_Count_Invariants(value Directory_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), NONEMPTY_SIZE_MINIMUM, DIRECTORY_SIZE_MAXIMUM).
		Ensure()
}

// Volume keeps Unix no-volume fact typed instead of broad text.
type Volume string

// Volume_Invariants proves platform absence exactly instead of accepting broad Text range.
func Volume_Invariants(value Volume, namespace aver.Namespace) {
	aver.Always(len(value) == VOLUME_SIZE, "A Unix volume is empty.")
}

// Dot_Count keeps fixed relative identity size typed.
type Dot_Count int

// Dot_Count_Invariants prevents fixed output from claiming full path range.
func Dot_Count_Invariants(value Dot_Count, namespace aver.Namespace) {
	aver.Always(int(value) == DOT_COUNT, "A relative identity has one byte.")
}

// Base_Text excludes zero because standard base result is never empty.
type Base_Text string

// Base_Text_Invariants proves mandatory dot through full final element.
func Base_Text_Invariants(value Base_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), NONEMPTY_SIZE_MINIMUM, PATH_SIZE_MAXIMUM).
		Ensure()
}

// Nonempty_Path preserves Clean proof before optional identity normalization.
type Nonempty_Path []byte

// Nonempty_Path_Invariants proves Clean mandatory dot output.
func Nonempty_Path_Invariants(value Nonempty_Path, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), NONEMPTY_SIZE_MINIMUM, PATH_SIZE_MAXIMUM).
		Ensure()
}

// Cleaned_Path permits empty only after relative algorithm removes dot identity.
type Cleaned_Path []byte

// Cleaned_Path_Invariants proves normalized identity stays inside path bound.
func Cleaned_Path_Invariants(value Cleaned_Path, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), PATH_SIZE_MINIMUM, PATH_SIZE_MAXIMUM).
		Ensure()
}

// Start excludes final boundary because relative comparison already handled equal paths.
type Start int

// Start_Invariants proves at least one differing byte follows boundary.
func Start_Invariants(value Start, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), PATH_SIZE_MINIMUM, START_MAXIMUM).
		Ensure()
}

// Up_Count bounds unmatched base elements by densest valid packing.
type Up_Count int

// Up_Count_Invariants proves shortest one-byte element packing bound.
func Up_Count_Invariants(value Up_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), PATH_SIZE_MINIMUM, UP_COUNT_MAXIMUM).
		Ensure()
}

// Nonempty_Text excludes loop state after component scan exhausts path.
type Nonempty_Text string

// Nonempty_Text_Invariants proves scan enters only active tails.
func Nonempty_Text_Invariants(value Nonempty_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), NONEMPTY_SIZE_MINIMUM, PATH_SIZE_MAXIMUM).
		Ensure()
}

// Part_Text permits empty because malicious repeated separators need validation.
type Part_Text string

// Part_Text_Invariants proves malicious empty component through full path component.
func Part_Text_Invariants(value Part_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), PATH_SIZE_MINIMUM, PATH_SIZE_MAXIMUM).
		Ensure()
}

// Part_Tail preserves proof that one separator was consumed.
type Part_Tail string

// Part_Tail_Invariants subtracts consumed separator before longest tail.
func Part_Tail_Invariants(value Part_Tail, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), PATH_SIZE_MINIMUM, PART_TAIL_MAXIMUM).
		Ensure()
}

// Clean_Into uses caller storage because returning cleaned host text would allocate.
func Clean_Into(destination bytes.Slice, value Text) (count Nonempty_Count) {
	defer func() { Nonempty_Count_Invariants(count, "clean_into.count") }()
	bytes.Slice_Invariants(destination, "clean_into.destination")
	Text_Invariants(value, "clean_into.value")
	return Nonempty_Count(path.Clean_Into(destination, path.Text(value)))
}

// Is_Local stays lexical so containment check needs no filesystem access.
func Is_Local(value Text) (local Boolean) {
	defer func() { Boolean_Invariants(local, "is_local.local") }()
	Text_Invariants(value, "is_local.value")
	if value == "" {
		return false
	}
	if value[0] == SEPARATOR {
		return false
	}
	depth := 0
	start := 0
	for end := 0; end <= len(value); end++ {
		if end < len(value) {
			if value[end] != SEPARATOR {
				continue
			}
		}
		part := value[start:end]
		if part == ".." {
			if depth == 0 {
				return false
			}
			depth--
		} else if part != "" {
			if part != "." {
				depth++
			}
		}
		start = end + 1
	}
	return true
}

// Localize_Into validates before copy because input is malicious and caller must see no partial
// path.
func Localize_Into(
	destination bytes.Slice, value Text,
) (count Boundary, err error) {
	defer func() { Boundary_Invariants(count, "localize_into.count") }()
	bytes.Slice_Invariants(destination, "localize_into.destination")
	Text_Invariants(value, "localize_into.value")
	if !valid_path(value) {
		return 0, Error_Invalid_Path
	}
	aver.Always(
		len(destination) >= len(value),
		"Filepath localize destination holds complete result.",
	)
	copy(destination, value)
	return Boundary(len(value)), nil
}

// To_Slash_Into uses caller storage because converted text may not share source bytes.
func To_Slash_Into(destination bytes.Slice, value Text) (count Boundary) {
	defer func() { Boundary_Invariants(count, "to_slash_into.count") }()
	bytes.Slice_Invariants(destination, "to_slash_into.destination")
	Text_Invariants(value, "to_slash_into.value")
	aver.Always(
		len(destination) >= len(value),
		"Filepath slash destination holds complete result.",
	)
	copy(destination, value)
	return Boundary(len(value))
}

// From_Slash_Into uses caller storage because converted text may not share source bytes.
func From_Slash_Into(destination bytes.Slice, value Text) (count Boundary) {
	defer func() { Boundary_Invariants(count, "from_slash_into.count") }()
	bytes.Slice_Invariants(destination, "from_slash_into.destination")
	Text_Invariants(value, "from_slash_into.value")
	aver.Always(
		len(destination) >= len(value),
		"Filepath host separator destination holds complete result.",
	)
	copy(destination, value)
	return Boundary(len(value))
}

// Split_List_Into uses caller slots because returned slice ownership would allocate.
func Split_List_Into(destination Paths, value Text) (count Path_Count) {
	defer func() { Path_Count_Invariants(count, "split_list_into.count") }()
	Paths_Invariants(destination, "split_list_into.destination")
	Text_Invariants(value, "split_list_into.value")
	if value == "" {
		return 0
	}
	count = 1
	for index := range len(value) {
		if value[index] == LIST_SEPARATOR {
			count++
		}
	}
	aver.Always(
		len(destination) >= int(count),
		"Filepath list destination holds every path.",
	)
	start := 0
	position := 0
	for index := range len(value) {
		if value[index] == LIST_SEPARATOR {
			destination[position] = bytes.Text(value[start:index])
			position++
			start = index + 1
		}
	}
	destination[position] = bytes.Text(value[start:])
	return count
}

// Split returns input views because component copies would allocate.
func Split(value Text) (directory Text, file Text) {
	defer func() {
		Text_Invariants(directory, "split.directory")
		Text_Invariants(file, "split.file")
	}()
	Text_Invariants(value, "split.value")
	path_directory, path_file := path.Split(path.Text(value))
	return Text(path_directory), Text(path_file)
}

// Join_Into uses caller storage because joined output has no source view.
func Join_Into(destination bytes.Slice, elements Elements) (count Boundary) {
	defer func() { Boundary_Invariants(count, "join_into.count") }()
	bytes.Slice_Invariants(destination, "join_into.destination")
	Elements_Invariants(elements, "join_into.elements")
	for _, element := range elements {
		Text_Invariants(Text(element), "join_into.element")
	}
	return Boundary(path.Join_Into(destination, path.Elements(elements)))
}

// Extension returns source view because suffix already exists in input.
func Extension(value Text) (extension Text) {
	defer func() { Text_Invariants(extension, "extension.extension") }()
	Text_Invariants(value, "extension.value")
	return Text(path.Extension(path.Text(value)))
}

// Is_Absolute stays lexical so no filesystem dependency enters filepath.
func Is_Absolute(value Text) (absolute Boolean) {
	defer func() { Boolean_Invariants(absolute, "is_absolute.absolute") }()
	Text_Invariants(value, "is_absolute.value")
	return Boolean(path.Is_Absolute(path.Text(value)))
}

// Absolute_Into takes working directory explicitly because ambient process state breaks injection.
func Absolute_Into(
	destination bytes.Slice, working_directory Text, value Text,
) (count Boundary, err error) {
	defer func() { Boundary_Invariants(count, "absolute_into.count") }()
	bytes.Slice_Invariants(destination, "absolute_into.destination")
	Text_Invariants(working_directory, "absolute_into.working_directory")
	Text_Invariants(value, "absolute_into.value")
	if Is_Absolute(value) {
		// Absolute input needs no working directory validity beyond bounded representation.
		return Boundary(Clean_Into(destination, value)), nil
	}
	if !Is_Absolute(working_directory) {
		return 0, Error_Working_Directory
	}
	if value == "" {
		return Boundary(Clean_Into(destination, working_directory)), nil
	}
	if len(working_directory)+NONEMPTY_SIZE_MINIMUM+len(value) > PATH_SIZE_MAXIMUM {
		return 0, Error_Result_Too_Large
	}
	joined := Elements{bytes.Text(working_directory), bytes.Text(value)}
	return Join_Into(destination, joined), nil
}

// Relative_Into uses caller storage because relative output has no source view.
func Relative_Into(
	destination bytes.Slice, base Text, target Text,
) (count Boundary, err error) {
	defer func() { Boundary_Invariants(count, "relative_into.count") }()
	bytes.Slice_Invariants(destination, "relative_into.destination")
	Text_Invariants(base, "relative_into.base")
	Text_Invariants(target, "relative_into.target")
	var base_storage [PATH_SIZE_MAXIMUM]byte
	base_count := Clean_Into(base_storage[:], base)
	var target_storage [PATH_SIZE_MAXIMUM]byte
	target_count := Clean_Into(target_storage[:], target)
	// Independent cleaned copies keep comparison stable while source forms differ lexically.
	base_clean := base_storage[:base_count]
	target_clean := target_storage[:target_count]
	if bytes.Equal(base_clean, target_clean) {
		return Boundary(write_dot(destination)), nil
	}
	if rooted(Nonempty_Path(base_clean)) != rooted(Nonempty_Path(target_clean)) {
		return 0, Error_Root_Mismatch
	}
	if bytes.Equal(base_clean, []byte(".")) {
		base_clean = base_clean[:0]
	}
	if bytes.Equal(target_clean, []byte(".")) {
		target_clean = target_clean[:0]
	}
	base_start, target_start := common_path_boundary(
		Cleaned_Path(base_clean), Cleaned_Path(target_clean),
	)
	if segment_is_dot_dot(Cleaned_Path(base_clean[base_start:])) {
		return 0, Error_Base_Above_Root
	}
	return write_relative(
		destination,
		Cleaned_Path(base_clean[base_start:]), Cleaned_Path(target_clean[target_start:]),
	)
}

// Has_Prefix preserves historical standard behavior for compatibility despite lexical weakness.
func Has_Prefix(value Text, prefix Text) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "has_prefix.yes") }()
	Text_Invariants(value, "has_prefix.value")
	Text_Invariants(prefix, "has_prefix.prefix")
	if len(prefix) > len(value) {
		return false
	}
	return Boolean(value[:len(prefix)] == prefix)
}

// Base returns source view where possible because component copies would allocate.
func Base(value Text) (base Base_Text) {
	defer func() { Base_Text_Invariants(base, "base.base") }()
	Text_Invariants(value, "base.value")
	return Base_Text(path.Base(path.Text(value)))
}

// Directory_Into uses caller storage because cleaning cannot always return input view.
func Directory_Into(destination bytes.Slice, value Text) (count Directory_Count) {
	defer func() { Directory_Count_Invariants(count, "directory_into.count") }()
	bytes.Slice_Invariants(destination, "directory_into.destination")
	Text_Invariants(value, "directory_into.value")
	return Directory_Count(path.Directory_Into(destination, path.Text(value)))
}

// Volume_Name returns typed empty view because Unix carries no volume prefix.
func Volume_Name(value Text) (volume Volume) {
	defer func() { Volume_Invariants(volume, "volume_name.volume") }()
	Text_Invariants(value, "volume_name.value")
	return ""
}

// Match reuses bounded slash matcher because Unix host separator equals slash.
func Match(pattern Text, name Text) (matched Boolean, err error) {
	defer func() { Boolean_Invariants(matched, "match.matched") }()
	Text_Invariants(pattern, "match.pattern")
	Text_Invariants(name, "match.name")
	path_matched, path_err := path.Match(path.Text(pattern), path.Text(name))
	if path_err != nil {
		return false, Error_Bad_Pattern
	}
	return Boolean(path_matched), nil
}

func next_part(value Nonempty_Text) (part Part_Text, rest Part_Tail) {
	defer func() {
		Part_Text_Invariants(part, "next_part.part")
		Part_Tail_Invariants(rest, "next_part.rest")
	}()
	Nonempty_Text_Invariants(value, "next_part.value")
	for index := range len(value) {
		if value[index] == SEPARATOR {
			return Part_Text(value[:index]), Part_Tail(value[index+1:])
		}
	}
	return Part_Text(value), ""
}

func valid_path(value Text) (valid Boolean) {
	defer func() { Boolean_Invariants(valid, "valid_path.valid") }()
	Text_Invariants(value, "valid_path.value")
	if !utf8.Valid_Text(utf8.Text(value)) {
		// Slash-path validity follows io/fs contract; invalid UTF-8 must never reach host
		// path.
		return false
	}
	if value == "." {
		return true
	}
	if value == "" {
		return false
	}
	if value[0] == SEPARATOR {
		return false
	}
	for tail := value; tail != ""; {
		part, rest := next_part(Nonempty_Text(tail))
		if part == "" {
			return false
		}
		if part == "." {
			return false
		}
		if part == ".." {
			return false
		}
		for index := range len(part) {
			if part[index] == 0 {
				return false
			}
		}
		tail = Text(rest)
	}
	return value[len(value)-1] != SEPARATOR
}

func rooted(value Nonempty_Path) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "rooted.yes") }()
	Nonempty_Path_Invariants(value, "rooted.value")
	return len(value) > 0 && value[0] == SEPARATOR
}

func common_path_boundary(base Cleaned_Path, target Cleaned_Path) (
	base_start Start, target_start Start,
) {
	defer func() {
		Start_Invariants(base_start, "common_path_boundary.base_start")
		Start_Invariants(target_start, "common_path_boundary.target_start")
	}()
	Cleaned_Path_Invariants(base, "common_path_boundary.base")
	Cleaned_Path_Invariants(target, "common_path_boundary.target")
	base_index := Boundary(0)
	target_index := Boundary(0)
	for int(base_index) <= len(base) {
		base_end := next_separator(base, Start(base_index))
		target_end := next_separator(target, Start(target_index))
		if !bytes.Equal(
			bytes.Slice(base[base_index:base_end]),
			bytes.Slice(target[target_index:target_end]),
		) {
			return Start(base_index), Start(target_index)
		}
		if int(base_end) < len(base) {
			base_end++
		}
		if int(target_end) < len(target) {
			target_end++
		}
		base_index = base_end
		target_index = target_end
		if int(base_index) == len(base) {
			return Start(base_index), Start(target_index)
		}
		if int(target_index) == len(target) {
			return Start(base_index), Start(target_index)
		}
	}
	return Start(base_index), Start(target_index)
}

func next_separator(value Cleaned_Path, start Start) (end Boundary) {
	defer func() { Boundary_Invariants(end, "next_separator.end") }()
	Cleaned_Path_Invariants(value, "next_separator.value")
	Start_Invariants(start, "next_separator.start")
	for end = Boundary(start); int(end) < len(value); end++ {
		if value[end] == SEPARATOR {
			return end
		}
	}
	return end
}

func segment_is_dot_dot(value Cleaned_Path) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "segment_is_dot_dot.yes") }()
	Cleaned_Path_Invariants(value, "segment_is_dot_dot.value")
	if len(value) < 2 {
		return false
	}
	if value[0] != '.' {
		return false
	}
	if value[1] != '.' {
		return false
	}
	return len(value) == 2 || value[2] == SEPARATOR
}

func write_dot(destination bytes.Slice) (count Dot_Count) {
	defer func() { Dot_Count_Invariants(count, "write_dot.count") }()
	bytes.Slice_Invariants(destination, "write_dot.destination")
	aver.Always(
		len(destination) >= NONEMPTY_SIZE_MINIMUM,
		"Filepath dot destination holds one byte.",
	)
	destination[0] = '.'
	return NONEMPTY_SIZE_MINIMUM
}

func write_relative(
	destination bytes.Slice, base Cleaned_Path, target Cleaned_Path,
) (count Boundary, err error) {
	defer func() { Boundary_Invariants(count, "write_relative.count") }()
	bytes.Slice_Invariants(destination, "write_relative.destination")
	Cleaned_Path_Invariants(base, "write_relative.base")
	Cleaned_Path_Invariants(target, "write_relative.target")
	up_count := relative_up_count(base)
	result_size := 0
	if up_count > 0 {
		result_size = int(up_count)*2 + int(up_count-1)
	}
	if len(target) > 0 {
		if up_count > 0 {
			result_size++
		}
		result_size += len(target)
	}
	if result_size > PATH_SIZE_MAXIMUM {
		return 0, Error_Result_Too_Large
	}
	// Size check precedes every write so failure cannot leak partial relative path.
	aver.Always(
		len(destination) >= result_size,
		"Filepath relative destination holds complete result.",
	)
	position := 0
	for index := range up_count {
		if index > 0 {
			destination[position] = SEPARATOR
			position++
		}
		destination[position] = '.'
		destination[position+1] = '.'
		position += 2
	}
	if len(target) > 0 {
		if position > 0 {
			destination[position] = SEPARATOR
			position++
		}
		position += copy(destination[position:], target)
	}
	return Boundary(position), nil
}

func relative_up_count(base Cleaned_Path) (count Up_Count) {
	defer func() { Up_Count_Invariants(count, "relative_up_count.count") }()
	Cleaned_Path_Invariants(base, "relative_up_count.base")
	if len(base) == 0 {
		return 0
	}
	count = 1
	for index := range len(base) {
		if base[index] == SEPARATOR {
			count++
		}
	}
	return count
}
