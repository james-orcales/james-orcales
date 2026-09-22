package time_test

import "testing"

func Test_Operating_System_Smoke(t *testing.T) {
  verify_operating_system_smoke(t)
}

func Test_Operating_System_API_Heap_Allocation(t *testing.T) {
  verify_operating_system_allocation(t)
}
