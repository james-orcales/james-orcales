package sh_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/james-orcales/james-orcales/golang_snacks/invariant"
	"github.com/james-orcales/james-orcales/golang_snacks/sh"
	"github.com/james-orcales/james-orcales/golang_snacks/snap"
)

func TestMain(m *testing.M) {
	invariant.RunTestMain(m)
}

func TestSpawn(t *testing.T) {
	sh.Spawn("echo", "test")
}

func TestPushDir(t *testing.T) {
	before := sh.WorkingDirectory()
	if filepath.Base(before) != "sh" {
		panic("Working directory must be the package directory")
	}
	popDir := sh.PushDir("../../")
	current := filepath.Base(sh.WorkingDirectory())
	if !snap.Init(`bot-management-solution`).IsEqual(current) {
		t.Error("PushDir didn't change into the project root")
	}
	popDir()
	after := sh.WorkingDirectory()
	if before != after {
		t.Error("PushDir didn't revert the original directory")
	}
}

func TestPipe(t *testing.T) {
	output := sh.Pipe("echo", "hello world")
	snap.Init(`hello world`).Expect(t, output)
}

func TestPipeWithErr(t *testing.T) {
	stdout, _ := sh.PipeWithErr("echo", "hello")
	snap.Init(`hello
`).Expect(t, stdout)
}

func TestFpipe(t *testing.T) {
	var buf bytes.Buffer
	sh.Fpipe(&buf, "echo", "test output")
	snap.Init(`test output
`).Expect(t, buf.String())
}

func TestFpipeWithErr(t *testing.T) {
	var stdout, stderr bytes.Buffer
	sh.FpipeWithErr(sh.FpipeWithErrInput{Stdout: &stdout, Stderr: &stderr}, "echo", "stdout content")
	snap.Init(`stdout content
`).Expect(t, stdout.String())
}

func TestSpawnRaw_EnvVars(t *testing.T) {
	snap.BatchExpect(t, func(args []string) any {
		var buf bytes.Buffer
		sh.SpawnRaw(sh.SpawnRawInput{Stdout: &buf}, args...)
		return buf.String()
	}, []snap.Entry[[]string]{
		{Name: "single env var", Input: []string{"TEST_VAR=hello", "sh", "-c", "echo $TEST_VAR"}, Snapshot: snap.Init(`hello
`)},
		{Name: "multiple env vars", Input: []string{"A=1", "B=2", "sh", "-c", "echo $A $B"}, Snapshot: snap.Init(`1 2
`)},
		{
			Name:  "env var with multiple equals",
			Input: []string{"VAR=value=with=equals", "sh", "-c", "echo $VAR"},
			Snapshot: snap.Init(`value=with=equals
`),
		},
		{Name: "env var with empty value", Input: []string{"EMPTY=", "sh", "-c", "echo \"${EMPTY}x\""}, Snapshot: snap.Init(`x
`)},
		{Name: "no env vars", Input: []string{"echo", "test"}, Snapshot: snap.Init(`test
`)},
		{Name: "env vars with command args", Input: []string{"X=10", "Y=20", "sh", "-c", "echo $X-$Y"}, Snapshot: snap.Init(`10-20
`)},
		{Name: "command arg with equals", Input: []string{"echo", "key=value"}, Snapshot: snap.Init(`key=value
`)},
	})
}

func TestSpawnRaw_WorkingDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	var buf bytes.Buffer
	sh.SpawnRaw(sh.SpawnRawInput{WorkingDirectory: tmpDir, Stdout: &buf}, "pwd")
	resolved, _ := filepath.EvalSymlinks(tmpDir)
	snap.Init(resolved+"\n").Expect(t, buf.String())
}

func TestPanics(t *testing.T) {
	snap.BatchExpectPanic(t, func(fn func()) {
		fn()
	}, []snap.Entry[func()]{
		{
			Name:     "SpawnRaw no args",
			Input:    func() { sh.SpawnRaw(sh.SpawnRawInput{}) },
			Snapshot: snap.Init(`scripting.spawn requires at least one argument`),
		},
		{
			Name:     "SpawnRaw only env vars",
			Input:    func() { sh.SpawnRaw(sh.SpawnRawInput{}, "A=1", "B=2") },
			Snapshot: snap.Init(`sh.SpawnRaw was not provided an executable`),
		},
		{Name: "Spawn no args", Input: func() { sh.Spawn() }, Snapshot: snap.Init(`scripting.spawn requires at least one argument`)},
		{Name: "Pipe no args", Input: func() { sh.Pipe() }, Snapshot: snap.Init(`scripting.spawn requires at least one argument`)},
		{
			Name:     "PipeWithErr no args",
			Input:    func() { sh.PipeWithErr() },
			Snapshot: snap.Init(`scripting.spawn requires at least one argument`),
		},
		{
			Name:     "Fpipe no args",
			Input:    func() { var buf bytes.Buffer; sh.Fpipe(&buf) },
			Snapshot: snap.Init(`scripting.spawn requires at least one argument`),
		},
		{
			Name:     "FpipeWithErr no args",
			Input:    func() { var buf bytes.Buffer; sh.FpipeWithErr(sh.FpipeWithErrInput{Stdout: &buf, Stderr: &buf}) },
			Snapshot: snap.Init(`scripting.spawn requires at least one argument`),
		},
		{
			Name:     "PushDir invalid dir",
			Input:    func() { sh.PushDir("/nonexistent/directory/that/does/not/exist") },
			Snapshot: snap.Init(`chdir /nonexistent/directory/that/does/not/exist: no such file or directory`),
		},
	})
}

func TestSpawnRaw_FailingCommand(t *testing.T) {
	var buf bytes.Buffer
	ok := sh.SpawnRaw(sh.SpawnRawInput{Stdout: &buf, Stderr: &buf}, "sh", "-c", "exit 1")
	if ok {
		t.Error("SpawnRaw should return false for failing command")
	}
}

func TestWorkingDirectory(t *testing.T) {
	wd := sh.WorkingDirectory()
	if wd == "" {
		t.Error("WorkingDirectory returned empty string")
	}
	if !filepath.IsAbs(wd) {
		t.Error("WorkingDirectory should return absolute path")
	}
}

func TestWhich(t *testing.T) {
	path := sh.Which("sh")
	if path == "" {
		t.Error("Which should find sh")
	}
	if !filepath.IsAbs(path) {
		t.Error("Which should return absolute path")
	}
}

func TestWhich_NotFound(t *testing.T) {
	path := sh.Which("nonexistent-command-12345")
	snap.Init(``).Expect(t, path)
}

func TestMakeDirAll(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "a", "b", "c")
	err := sh.MakeDirAll(newDir)
	if err != nil {
		t.Errorf("MakeDirAll failed: %v", err)
	}
	info, err := os.Stat(newDir)
	if err != nil {
		t.Errorf("Created directory doesn't exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("Created path is not a directory")
	}
}

func TestLines(t *testing.T) {
	t.Run("zero returns nil", func(t *testing.T) {
		result := sh.Lines("a\nb\nc", 0)
		snap.Init(`<nil>`).Expect(t, formatSlice(result))
	})

	t.Run("positive n splits into n parts", func(t *testing.T) {
		result := sh.Lines("a\nb\nc\nd", 2)
		snap.Init(`[a b
c
d]`).Expect(t, formatSlice(result))
	})

	t.Run("positive n equal to line count", func(t *testing.T) {
		result := sh.Lines("a\nb\nc", 3)
		snap.Init(`[a b c]`).Expect(t, formatSlice(result))
	})

	t.Run("single line with positive n", func(t *testing.T) {
		result := sh.Lines("hello", 1)
		snap.Init(`[hello]`).Expect(t, formatSlice(result))
	})

	t.Run("negative n returns last lines when enough lines exist", func(t *testing.T) {
		result := sh.Lines("a\nb\nc\nd\ne", -2)
		snap.Init(`[d e]`).Expect(t, formatSlice(result))
	})

	t.Run("negative n returns all lines when fewer lines exist", func(t *testing.T) {
		result := sh.Lines("a\nb", -5)
		snap.Init(`[a b]`).Expect(t, formatSlice(result))
	})
}

func formatSlice(s []string) (out string) {
	if s == nil {
		return "<nil>"
	}
	return "[" + joinStrings(s, " ") + "]"
}

func joinStrings(s []string, sep string) (out string) {
	if len(s) == 0 {
		return ""
	}
	result := s[0]
	for i := 1; i < len(s); i++ {
		result += sep + s[i]
	}
	return result
}
