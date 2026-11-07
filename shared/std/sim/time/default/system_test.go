package time_test

import "testing"

import library "local/james-orcales/shared/std/sim/time"
import "local/james-orcales/shared/std/sim/time/default"
import "local/james-orcales/shared/std/testify"

// Test_Operating_System_Smoke check host clock never read monotonic backward, and report
// positive wall-clock time. Real time is not deterministic, thus this is smoke test, not
// snapshot.
func verify_operating_system_smoke(t *testing.T) {
  host := time.New_Operating_System_Clock()
  first := time.Clock_Now_Monotonic(host)
  second := time.Clock_Now_Monotonic(host)
  testify.True(t, !(second < first), first, second)
  testify.Positive(t, time.Clock_Now_Realtime(host))
}

// Host-backed readers need the same heap proof as deterministic readers.
func verify_operating_system_allocation(t *testing.T) {
  host := time.New_Operating_System_Clock()
  t.Run("New_Operating_System_Clock", func(t *testing.T) {
    testify.Zero_Allocation(t, func() { allocation_clock(&host) })
  })
  t.Run("Now_Monotonic", func(t *testing.T) {
    verify_monotonic_allocation(t, host)
  })
  t.Run("Now_Realtime", func(t *testing.T) {
    verify_realtime_allocation(t, host)
  })
}

func verify_monotonic_allocation(t *testing.T, host library.Clock) {
  var moment int64
  testify.Zero_Allocation(t, func() {
    allocation_monotonic(&moment, host)
  })
  testify.Positive(t, moment)
}

func verify_realtime_allocation(t *testing.T, host library.Clock) {
  var moment int64
  testify.Zero_Allocation(t, func() {
    allocation_realtime(&moment, host)
  })
  testify.Positive(t, moment)
}

func allocation_clock(host *library.Clock) {
  *host = time.New_Operating_System_Clock()
}

func allocation_monotonic(moment *int64, host library.Clock) {
  *moment = int64(time.Clock_Now_Monotonic(host))
}

func allocation_realtime(moment *int64, host library.Clock) {
  *moment = int64(time.Clock_Now_Realtime(host))
}
