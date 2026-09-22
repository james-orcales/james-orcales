package uuid_test

import "testing"

func Test_Generator(t *testing.T) {
  test_operating_system_generator_smoke(t)
}

func Test_Allocation(t *testing.T) {
  test_operating_system_generator_zero_allocation(t)
}

func Test_Replay(t *testing.T) {
  test_operating_system_generator_accepts_replay_boundaries(t)
}
