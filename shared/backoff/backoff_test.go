package backoff_test

import (
	"local/james-orcales/shared/backoff"
	"local/james-orcales/shared/io"
	"local/james-orcales/shared/time"
)

// Retry_Result captures what a driven Retry delivered plus the virtual time it took.
type Retry_Result struct {
	// Value is the result the operation ultimately returned.
	Value int
	// Error is the error Retry delivered, or nil on success.
	Error error
	// Elapsed is the virtual time the retries consumed on the io clock.
	Elapsed time.Duration
}

// Drives Retry on a fresh seeded sim loop, pumping until it finishes, and reports the
// delivered outcome with the virtual time the waits consumed.
func run_retry(
	seed uint64, policy backoff.Policy, tries uint, operation backoff.Operation[int],
) (outcome Retry_Result) {
	loop, driver, clock := io.New_Sim(seed)
	started := clock.Now_Monotonic()
	done_called := false
	backoff.Retry[int](&backoff.Retry_Input{
		Timer:     &loop,
		Policy:    policy,
		Tries_Max: tries,
	}, operation, func(value int, err error) {
		// Read the clock here, inside the completing callback: it reports the exact
		// Ready_At, whereas a read after Run_Until returns has already ticked past it.
		outcome.Value = value
		outcome.Error = err
		outcome.Elapsed = time.Duration(clock.Now_Monotonic() - started)
		done_called = true
	})
	driver.Run_Until(func() (finished bool) { return done_called }, io.FOREVER)
	return outcome
}
