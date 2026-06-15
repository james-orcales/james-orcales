package vsr_test

import (
	"testing"

	invariant "github.com/james-orcales/james-orcales/shared/invariant/default"
	"github.com/james-orcales/james-orcales/shared/time"
	"github.com/james-orcales/james-orcales/shared/vsr"
)

// TestMain runs the suite through the invariant harness so that, under a plain `go test`, every
// Always reached must hold — a coverage gap or a violated assertion fails the run. The seeded
// fuzzer and the cluster-wide invariants live in the sibling simulation package, which can spawn
// the goroutines parallel seeds need (this deterministic package forbids them).
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}

// Folds one message into replica at time zero and returns the step output.
func receive(replica *vsr.Replica, message vsr.Message) (output vsr.Step_Output) {
	return vsr.Replica_Receive(&vsr.Replica_Receive_Input{Replica: replica, Message: message})
}

// Advances replica to now and returns the step output.
func tick(replica *vsr.Replica, now time.Moment) (output vsr.Step_Output) {
	return vsr.Replica_Tick(&vsr.Replica_Tick_Input{Replica: replica, Now: now})
}

// Returns the first message of the given kind, reporting whether one was found.
func first_message(
	messages []vsr.Message, kind vsr.Message_Kind,
) (found vsr.Message, ok bool) {
	for _, message := range messages {
		if message.Kind == kind {
			return message, true
		}
	}
	return found, false
}

// Returns the messages of the given kind keyed by their To field, so a test can assert which
// recipients a step addressed a kind to without an inline collection loop per assertion.
func messages_of_kind(
	messages []vsr.Message, kind vsr.Message_Kind,
) (by_recipient map[vsr.Replica_Identifier]vsr.Message) {
	by_recipient = map[vsr.Replica_Identifier]vsr.Message{}
	for _, message := range messages {
		if message.Kind == kind {
			by_recipient[message.To] = message
		}
	}
	return by_recipient
}

// A normal replica 1 at epoch 1, view 2, on a 3-node group: the fixture the epoch-precedence matrix
// reuses so each cell starts from the same (epoch, view) baseline.
func epoch_1_view_2_replica() (replica vsr.Replica) {
	return vsr.Replica{
		Identifier: 1, Configuration: vsr.Configuration{0, 1, 2},
		Epoch: 1, View: 2, Status: vsr.Status_Normal,
	}
}
