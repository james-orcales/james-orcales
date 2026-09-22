package os_test

import "testing"

func Test_Host(t *testing.T) {
  test_operating_system_smoke(t)
  test_operating_system_environment(t)
  test_operating_system_arguments_live(t)
  test_operating_system_environment_live(t)
  test_operating_system_arguments_copy(t)
  test_operating_system_self_exec_environment(t)
}

func Test_Exit(t *testing.T) {
  test_operating_system_exit(t)
}

func Test_Bounds(t *testing.T) {
  test_operating_system_destination_capacity(t)
  test_operating_system_invariant_boundaries(t)
  test_operating_system_memory(t)
}

func Test_Allocation(t *testing.T) {
  test_operating_system_heap_allocation(t)
  test_operating_system_memory_heap_allocation(t)
}

func Test_Files(t *testing.T) {
  test_file_round_trip(t)
  test_file_read_into(t)
  test_file_read_absent(t)
  test_standard_error_write(t)
  test_descriptor_terminal(t)
}

func Test_Machine(t *testing.T) {
  test_machine_read(t)
}
