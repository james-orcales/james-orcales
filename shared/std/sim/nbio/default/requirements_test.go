package nbio

import "testing"

func Test_Storage(t *testing.T) {
  t.Run("File_Chain", check_file_chain)
  t.Run("Write_Cursor", check_write_cursor)
  t.Run("Pipe", check_pipe)
}

func Test_Network(t *testing.T) {
  t.Run("Socket", check_socket)
  t.Run("Receive_Timeout", check_receive_timeout)
  t.Run("Send_In_Connect", check_send_in_connect)
}

func Test_Timeline(t *testing.T) {
  t.Run("Timeout", check_timeout)
  t.Run("Callback_Released", check_callback_released)
  t.Run("Reuse", check_reuse)
  t.Run("Reentrancy", check_reentrancy)
}

func Test_Events(t *testing.T) {
  t.Run("Event", check_event)
  t.Run("Concurrent_Rearm", check_concurrent_rearm)
}

func Test_Processes(t *testing.T) {
  t.Run("Spawn", check_spawn)
  t.Run("Deadline", check_spawn_deadline)
  t.Run("Output_Too_Large", check_output_too_large)
  t.Run("Reaps_After_Deadline", check_reaps_after_deadline)
}

func Test_Allocations(t *testing.T) {
  t.Run("Constructor", check_constructor_allocation)
  t.Run("Storage", check_storage_allocation)
  t.Run("Network_Synchronous", check_network_synchronous_allocation)
  t.Run("Network_Asynchronous", check_network_asynchronous_allocation)
  t.Run("Spawn", check_spawn_allocation)
  t.Run("Spawn_Capture", check_spawn_capture_allocation)
}
