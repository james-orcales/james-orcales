// Package main is the timeout command: the thin composition root over the internal
// library tier, and the one place allowed to bind a real process. The child is started
// with os/exec rather than through shared/io, whose Spawn keeps the process handle to
// itself and so offers no way to kill a child that outlives its deadline — the one thing
// this program exists to do.
package main

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	systime "local/james-orcales/shared/time"
	timeout "local/james-orcales/timeout/internal"
)

// SIGNAL_EXIT_BASE is the shell's convention for a process killed by a signal: 128 plus
// the signal number. A signalled process carries no exit status of its own, so this is the
// only code there is to report for one.
const SIGNAL_EXIT_BASE = 128

func main() {
	os.Exit(timeout.Main(&timeout.Main_Input{
		Arguments:    os.Args,
		Error_Output: os.Stderr,
		Start:        main_start,
	}))
}

// Starts path with arguments as a transparent child: it inherits this process's standard
// input, output, and error, so a command run under timeout behaves as it would run alone.
// The wait runs on its own goroutine and parks the exit code in a buffered channel, so the
// deadline race never blocks it and Terminate can still collect that code after killing.
func main_start(path string, arguments []string) (child timeout.Child, err error) {
	command := exec.Command(path, arguments...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	// The child leads a process group of its own so the deadline reaches the whole tree it
	// spawns, not just the process this program can see: killing a shell script alone
	// leaves the commands it launched running, which is the case a deadline exists for.
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	start_err := command.Start()
	if start_err != nil {
		return timeout.Child{}, start_err
	}
	// A group led by the child is numbered by the child's own pid.
	group := command.Process.Pid
	exit := make(chan int, 1)
	go func() {
		command.Wait()
		exit <- main_exit_code(command.ProcessState)
	}()
	main_forward(group)
	return timeout.Child{
		Wait_Until: main_wait_until(exit),
		Terminate:  main_terminate(group, exit),
	}, nil
}

// Forwards an interrupt or a termination aimed at this program on to the child's group. A
// child in its own group is out of the terminal's foreground group, so without this a
// Ctrl-C would kill the wrapper alone and leave the very tree it was bounding orphaned and
// running. This program outlives the signal on purpose: its answer is the child's fate.
func main_forward(group int) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals,
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	go func() {
		for received := range signals {
			forwarded, ok := received.(syscall.Signal)
			if !ok {
				continue
			}
			syscall.Kill(-group, forwarded)
		}
	}()
}

// Returns the bounded wait: the child's exit code when it arrives before the deadline, or
// nothing and a false report once the deadline elapses first.
func main_wait_until(
	exit chan int,
) (wait func(deadline systime.Duration) (status_code int, exited bool)) {
	return func(deadline systime.Duration) (status_code int, exited bool) {
		alarm := time.NewTimer(time.Duration(deadline))
		defer alarm.Stop()
		select {
		case code := <-exit:
			return code, true
		case <-alarm.C:
			return 0, false
		}
	}
}

// Returns the terminator: it kills the child's whole process group and waits for the exit
// its wait goroutine posts, so the child is reaped here rather than orphaned as this
// program exits.
func main_terminate(group int, exit chan int) (terminate func()) {
	return func() {
		// A negated group id is kill(2)'s spelling of "every process in this group". The
		// group outlives its leader, so this still reaches the grandchildren even when the
		// child itself has already exited.
		//
		// SIGKILL, not SIGTERM: a deadline a child may decline is not a deadline. Nothing
		// downstream waits on a graceful stop — the caller asked for the command to be
		// over by now.
		syscall.Kill(-group, syscall.SIGKILL)
		<-exit
	}
}

// Reports the exit code of a finished process. A process killed by a signal has no exit
// status of its own and takes the shell's 128+signal form instead; a process whose state
// cannot be read at all yields no outcome to report and so answers with the wrapper's own
// failure code.
func main_exit_code(state *os.ProcessState) (status_code int) {
	if state == nil {
		return timeout.EXIT_START_FAILURE
	}
	code := state.ExitCode()
	if code >= 0 {
		return code
	}
	status, ok := state.Sys().(syscall.WaitStatus)
	if !ok {
		return timeout.EXIT_START_FAILURE
	}
	return SIGNAL_EXIT_BASE + int(status.Signal())
}
