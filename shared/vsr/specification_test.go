package vsr_test

import (
	"testing"

	"github.com/james-orcales/james-orcales/shared/vsr"
)

// Test_Normal_Operation_Append: a client command received by the primary becomes one log entry
// tagged with the current view, and op advances to equal the log length.
func Test_Normal_Operation_Append(t *testing.T) {
	primary := vsr.Replica{Identifier: 0, Configuration: vsr.Configuration{0, 1, 2}}
	vsr.Replica_Receive(&vsr.Replica_Receive_Input{
		Replica: &primary,
		Message: vsr.Message{
			Kind: vsr.Message_Kind_Request, To: 0, Command: []byte("set x=1"),
		},
	})
	if primary.Op != 1 {
		t.Fatalf("expected op 1, got %d", primary.Op)
	}
	if len(primary.Log) != 1 {
		t.Fatalf("expected log length 1, got %d", len(primary.Log))
	}
	if string(primary.Log[0].Command) != "set x=1" {
		t.Errorf("expected command %q, got %q", "set x=1", primary.Log[0].Command)
	}
	if primary.Log[0].View != primary.View {
		t.Errorf("expected entry tagged with view %d, got %d",
			primary.View, primary.Log[0].View)
	}
}

// Test_Normal_Operation_Prepare: appending on the primary broadcasts a Prepare to every other
// replica, each carrying the new op, the entry, and the primary's commit number.
func Test_Normal_Operation_Prepare(t *testing.T) {
	primary := vsr.Replica{Identifier: 0, Configuration: vsr.Configuration{0, 1, 2}}
	output := vsr.Replica_Receive(&vsr.Replica_Receive_Input{
		Replica: &primary,
		Message: vsr.Message{
			Kind: vsr.Message_Kind_Request, To: 0, Command: []byte("set x=1"),
		},
	})
	if len(output.Messages) != 2 {
		t.Fatalf("expected 2 prepares, got %d", len(output.Messages))
	}
	recipients := map[vsr.Replica_Identifier]bool{}
	for _, message := range output.Messages {
		if message.Kind != vsr.Message_Kind_Prepare {
			t.Errorf("expected a Prepare, got kind %d", message.Kind)
		}
		if message.From != 0 {
			t.Errorf("expected from primary 0, got %d", message.From)
		}
		if message.Op != 1 {
			t.Errorf("expected op 1, got %d", message.Op)
		}
		if string(message.Entries[0].Command) != "set x=1" {
			t.Errorf("expected entry command %q, got %q",
				"set x=1", message.Entries[0].Command)
		}
		if message.Commit != primary.Commit {
			t.Errorf("expected commit %d, got %d", primary.Commit, message.Commit)
		}
		recipients[message.To] = true
	}
	if recipients[0] {
		t.Errorf("expected no prepare to the primary 0, got %v", recipients)
	}
	if !recipients[1] {
		t.Errorf("expected a prepare to backup 1, got %v", recipients)
	}
	if !recipients[2] {
		t.Errorf("expected a prepare to backup 2, got %v", recipients)
	}
}

// Test_Normal_Operation_Prepare_Ok: a backup applies a Prepare for the op right after its last one
// and acks with a single Prepare_Ok to the primary; a Prepare that skips an op is not applied.
func Test_Normal_Operation_Prepare_Ok(t *testing.T) {
	backup := vsr.Replica{Identifier: 1, Configuration: vsr.Configuration{0, 1, 2}}
	prepare := vsr.Message{
		Kind: vsr.Message_Kind_Prepare, From: 0, To: 1, View: 0,
		Op: 1, Entries: []vsr.Log_Entry{{View: 0, Command: []byte("set x=1")}},
	}
	output := vsr.Replica_Receive(&vsr.Replica_Receive_Input{
		Replica: &backup, Message: prepare,
	})
	if backup.Op != 1 {
		t.Fatalf("expected the prepared op applied, got op %d", backup.Op)
	}
	if len(backup.Log) != 1 {
		t.Fatalf("expected the prepared entry applied, got log length %d", len(backup.Log))
	}
	if len(output.Messages) != 1 {
		t.Fatalf("expected one Prepare_Ok, got %d messages", len(output.Messages))
	}
	ack := output.Messages[0]
	if ack.Kind != vsr.Message_Kind_Prepare_Ok {
		t.Errorf("expected a Prepare_Ok, got %+v", ack)
	}
	if ack.From != 1 {
		t.Errorf("expected the ack from backup 1, got %+v", ack)
	}
	if ack.To != 0 {
		t.Errorf("expected the ack to primary 0, got %+v", ack)
	}
	if ack.Op != 1 {
		t.Errorf("expected the ack for op 1, got %+v", ack)
	}

	// A Prepare that skips an op (the backup is at op 1, this is op 3) is not applied.
	skip := vsr.Message{
		Kind: vsr.Message_Kind_Prepare, From: 0, To: 1, View: 0,
		Op: 3, Entries: []vsr.Log_Entry{{View: 0, Command: []byte("set z=9")}},
	}
	output = vsr.Replica_Receive(&vsr.Replica_Receive_Input{Replica: &backup, Message: skip})
	if backup.Op != 1 {
		t.Errorf("expected a skipped op to be ignored, op advanced to %d", backup.Op)
	}
	if _, acked := first_message(output.Messages, vsr.Message_Kind_Prepare_Ok); acked {
		t.Errorf("expected no Prepare_Ok for a skipped op, got %+v", output.Messages)
	}
}

// Test_Normal_Operation_Commit: on an N=3 cluster the primary commits an op once a single backup
// acks it — the primary counts itself, so two distinct replicas form the quorum — and surfaces
// the entry; a duplicate ack from the same backup commits nothing further.
func Test_Normal_Operation_Commit(t *testing.T) {
	primary := vsr.Replica{Identifier: 0, Configuration: vsr.Configuration{0, 1, 2}}
	vsr.Replica_Receive(&vsr.Replica_Receive_Input{
		Replica: &primary,
		Message: vsr.Message{
			Kind: vsr.Message_Kind_Request, To: 0, Command: []byte("set x=1"),
		},
	})

	output := vsr.Replica_Receive(&vsr.Replica_Receive_Input{
		Replica: &primary,
		Message: vsr.Message{
			Kind: vsr.Message_Kind_Prepare_Ok, From: 1, To: 0, View: 0, Op: 1,
		},
	})
	if primary.Commit != 1 {
		t.Fatalf("expected commit 1 after a quorum, got %d", primary.Commit)
	}
	if len(output.Committed) != 1 {
		t.Fatalf("expected the committed entry surfaced, got %+v", output.Committed)
	}
	if string(output.Committed[0].Command) != "set x=1" {
		t.Fatalf("expected the committed entry surfaced, got %+v", output.Committed)
	}

	output = vsr.Replica_Receive(&vsr.Replica_Receive_Input{
		Replica: &primary,
		Message: vsr.Message{
			Kind: vsr.Message_Kind_Prepare_Ok, From: 1, To: 0, View: 0, Op: 1,
		},
	})
	if primary.Commit != 1 {
		t.Errorf("expected commit to stay 1 on a duplicate ack, got %d", primary.Commit)
	}
	if len(output.Committed) != 0 {
		t.Errorf("expected nothing newly committed on a duplicate ack, got %+v",
			output.Committed)
	}
}

// Test_Normal_Operation_Commit_Broadcast: an idle primary advertises its commit number with a
// Commit to every backup only once its heartbeat deadline passes, then on the next cadence.
func Test_Normal_Operation_Commit_Broadcast(t *testing.T) {
	primary := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 0, Configuration: vsr.Configuration{0, 1, 2},
		Heartbeat: 10, Timeout: 100, Now: 0,
	})
	// Commit op 1 so the heartbeat has a non-zero commit number to carry.
	vsr.Replica_Receive(&vsr.Replica_Receive_Input{
		Replica: &primary,
		Message: vsr.Message{
			Kind: vsr.Message_Kind_Request, To: 0, Command: []byte("set x=1"),
		},
	})
	vsr.Replica_Receive(&vsr.Replica_Receive_Input{
		Replica: &primary,
		Message: vsr.Message{
			Kind: vsr.Message_Kind_Prepare_Ok, From: 1, To: 0, View: 0, Op: 1,
		},
	})

	early := vsr.Replica_Tick(&vsr.Replica_Tick_Input{Replica: &primary, Now: 5})
	if len(early.Messages) != 0 {
		t.Fatalf("expected no broadcast before the heartbeat deadline, got %d",
			len(early.Messages))
	}
	output := vsr.Replica_Tick(&vsr.Replica_Tick_Input{Replica: &primary, Now: 10})
	if len(output.Messages) != 2 {
		t.Fatalf("expected a Commit to each backup at the deadline, got %d",
			len(output.Messages))
	}
	for _, message := range output.Messages {
		if message.Kind != vsr.Message_Kind_Commit {
			t.Errorf("expected a Commit, got %+v", message)
		}
		if message.From != 0 {
			t.Errorf("expected the Commit from 0, got %+v", message)
		}
		if message.Commit != 1 {
			t.Errorf("expected the Commit to carry commit 1, got %+v", message)
		}
	}
	if output.Timer.Kind != vsr.Timer_Kind_Commit {
		t.Errorf("expected the heartbeat timer re-armed, got %+v", output.Timer)
	}
	if output.Timer.Deadline != 20 {
		t.Errorf("expected the heartbeat re-armed to 20, got %+v", output.Timer)
	}
	late := vsr.Replica_Tick(&vsr.Replica_Tick_Input{Replica: &primary, Now: 15})
	if len(late.Messages) != 0 {
		t.Errorf("expected silence before the next cadence, got %d", len(late.Messages))
	}
}

// Test_Normal_Operation_Commit_Apply: a backup told of a higher commit number advances its own and
// surfaces every newly-committed entry in op order, never past the ops it actually holds.
func Test_Normal_Operation_Commit_Apply(t *testing.T) {
	backup := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 1, Configuration: vsr.Configuration{0, 1, 2},
		Heartbeat: 10, Timeout: 100, Now: 0,
	})
	for op, command := range []string{"a", "b"} {
		vsr.Replica_Receive(&vsr.Replica_Receive_Input{
			Replica: &backup,
			Message: vsr.Message{
				Kind: vsr.Message_Kind_Prepare, From: 0, To: 1, View: 0,
				Op:      vsr.Op(op + 1),
				Entries: []vsr.Log_Entry{{View: 0, Command: []byte(command)}},
			},
		})
	}
	if backup.Commit != 0 {
		t.Fatalf("expected no commit before the primary advertises one, got %d",
			backup.Commit)
	}

	// The advertised commit exceeds the backup's op count: it may only apply what it holds.
	output := vsr.Replica_Receive(&vsr.Replica_Receive_Input{
		Replica: &backup,
		Message: vsr.Message{
			Kind: vsr.Message_Kind_Commit, From: 0, To: 1, View: 0, Commit: 5,
		},
	})
	if backup.Commit != 2 {
		t.Fatalf("expected commit capped at op 2, got %d", backup.Commit)
	}
	if len(output.Committed) != 2 {
		t.Fatalf("expected entries a,b surfaced in order, got %+v", output.Committed)
	}
	if string(output.Committed[0].Command) != "a" {
		t.Fatalf("expected entries a,b surfaced in order, got %+v", output.Committed)
	}
	if string(output.Committed[1].Command) != "b" {
		t.Fatalf("expected entries a,b surfaced in order, got %+v", output.Committed)
	}
}

// Test_Normal_Operation_Duplicate_Request: a Request whose request-number is not greater than the
// client-table's recorded number is not re-appended. When it equals the most recent executed
// request, the cached result is re-sent as a Reply (a client that lost its reply, §4.5); a
// strictly stale request is dropped silently.
func Test_Normal_Operation_Duplicate_Request(t *testing.T) {
	calls := 0
	primary := vsr.Replica{
		Identifier:    0,
		Configuration: vsr.Configuration{0, 1, 2},
		State_Machine: vsr.State_Machine{Execute: func(
			command []byte, prediction []byte,
		) (result []byte) {
			calls++
			return []byte("ok")
		}},
	}
	// Append and commit op 1 for client 7, request 3, so it becomes executed.
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Client: 7, Request_Number: 3,
		Command: []byte("set x=1"),
	})
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Prepare_Ok, From: 1, To: 0, View: 0, Op: 1,
	})
	if calls != 1 {
		t.Fatalf("expected one execution after commit, got %d", calls)
	}

	// A duplicate of the executed request: not re-appended, but the cached reply is re-sent.
	output := receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Client: 7, Request_Number: 3,
		Command: []byte("set x=1"),
	})
	if primary.Op != 1 {
		t.Fatalf("expected no re-append on a duplicate, op is %d", primary.Op)
	}
	if calls != 1 {
		t.Fatalf("expected no re-execution on a duplicate, calls is %d", calls)
	}
	reply, ok := first_message(output.Replies, vsr.Message_Kind_Reply)
	if !ok {
		t.Fatalf("expected the cached reply re-sent, got %+v", output.Replies)
	}
	if reply.Client != 7 {
		t.Errorf("expected the reply to client 7, got %+v", reply)
	}
	if reply.Request_Number != 3 {
		t.Errorf("expected the reply for request 3, got %+v", reply)
	}
	if string(reply.Result) != "ok" {
		t.Errorf("expected the cached result %q, got %q", "ok", reply.Result)
	}

	// A strictly stale request (request 2 < 3) is dropped silently: no append, no reply.
	output = receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Client: 7, Request_Number: 2,
		Command: []byte("old"),
	})
	if primary.Op != 1 {
		t.Errorf("expected no re-append on a stale request, op is %d", primary.Op)
	}
	if len(output.Replies) != 0 {
		t.Errorf("expected no reply for a strictly stale request, got %+v", output.Replies)
	}
}

// Test_Normal_Operation_Inflight_Request: a Request equal to the client-table's current
// not-yet-executed request is dropped — no new entry, no Reply — since the inflight op will
// reply when it commits.
func Test_Normal_Operation_Inflight_Request(t *testing.T) {
	primary := vsr.Replica{
		Identifier:    0,
		Configuration: vsr.Configuration{0, 1, 2},
		State_Machine: vsr.State_Machine{Execute: func(
			command []byte, prediction []byte,
		) (result []byte) {
			return []byte("ok")
		}},
	}
	// Append op 1 for client 7, request 3, but do NOT commit it: it stays in flight.
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Client: 7, Request_Number: 3,
		Command: []byte("set x=1"),
	})
	if primary.Op != 1 {
		t.Fatalf("expected the request appended, op is %d", primary.Op)
	}

	// A re-send of the still-in-flight request: dropped, no second entry, no reply.
	output := receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Client: 7, Request_Number: 3,
		Command: []byte("set x=1"),
	})
	if primary.Op != 1 {
		t.Fatalf("expected no re-append for an in-flight request, op is %d", primary.Op)
	}
	if len(output.Replies) != 0 {
		t.Errorf("expected no reply for an in-flight request, got %+v", output.Replies)
	}
	if len(output.Messages) != 0 {
		t.Errorf("expected no Prepare for an in-flight request, got %+v", output.Messages)
	}
}

// Test_Execution_Up_Call: advancing the commit number calls Execute once per op in op order, and
// never twice for the same op.
func Test_Execution_Up_Call(t *testing.T) {
	var executed [][]byte
	primary := vsr.Replica{
		Identifier:    0,
		Configuration: vsr.Configuration{0, 1, 2},
		State_Machine: vsr.State_Machine{Execute: func(
			command []byte, prediction []byte,
		) (result []byte) {
			executed = append(executed, command)
			return command
		}},
	}
	for request, command := range []string{"a", "b"} {
		receive(&primary, vsr.Message{
			Kind: vsr.Message_Kind_Request, To: 0, Client: 7,
			Request_Number: vsr.Request_Number(request + 1), Command: []byte(command),
		})
	}
	// One ack commits op 1 (quorum of two with the primary); a later ack for op 2 commits it.
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Prepare_Ok, From: 1, To: 0, View: 0, Op: 1,
	})
	if len(executed) != 1 {
		t.Fatalf("expected exactly op 1 executed, got %+v", executed)
	}
	if string(executed[0]) != "a" {
		t.Fatalf("expected op 1 = a executed first, got %+v", executed)
	}
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Prepare_Ok, From: 1, To: 0, View: 0, Op: 2,
	})
	if len(executed) != 2 {
		t.Fatalf("expected op 2 executed, never twice, got %+v", executed)
	}
	if string(executed[1]) != "b" {
		t.Fatalf("expected op 2 = b executed second, got %+v", executed)
	}

	// A duplicate ack must commit nothing further and so execute nothing further.
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Prepare_Ok, From: 1, To: 0, View: 0, Op: 2,
	})
	if len(executed) != 2 {
		t.Errorf("expected no re-execution on a duplicate ack, got %+v", executed)
	}
}

// Test_Execution_Reply: the primary, on committing a client's op, emits a Reply to the client
// carrying the view, request-number, and Execute result; a backup committing the same op emits no
// Reply.
func Test_Execution_Reply(t *testing.T) {
	primary := vsr.Replica{
		Identifier:    0,
		Configuration: vsr.Configuration{0, 1, 2},
		State_Machine: vsr.State_Machine{Execute: func(
			command []byte, prediction []byte,
		) (result []byte) {
			return []byte("result-" + string(command))
		}},
	}
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Client: 9, Request_Number: 1,
		Command: []byte("x"),
	})
	output := receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Prepare_Ok, From: 1, To: 0, View: 0, Op: 1,
	})
	reply, ok := first_message(output.Replies, vsr.Message_Kind_Reply)
	if !ok {
		t.Fatalf("expected a Reply from the primary on commit, got %+v", output.Replies)
	}
	if reply.Client != 9 {
		t.Errorf("expected the reply addressed to client 9, got %+v", reply)
	}
	if reply.Request_Number != 1 {
		t.Errorf("expected the reply for request 1, got %+v", reply)
	}
	if reply.View != 0 {
		t.Errorf("expected the reply to carry view 0, got %+v", reply)
	}
	if string(reply.Result) != "result-x" {
		t.Errorf("expected the Execute result carried, got %q", reply.Result)
	}

	// A backup commits the same op via a Commit heartbeat but emits no Reply.
	backup := vsr.Replica{
		Identifier:    1,
		Configuration: vsr.Configuration{0, 1, 2},
		State_Machine: vsr.State_Machine{Execute: func(
			command []byte, prediction []byte,
		) (result []byte) {
			return []byte("result-" + string(command))
		}},
	}
	receive(&backup, vsr.Message{
		Kind: vsr.Message_Kind_Prepare, From: 0, To: 1, View: 0, Op: 1,
		Entries: []vsr.Log_Entry{{
			View: 0, Command: []byte("x"), Client: 9, Request_Number: 1,
		}},
	})
	output = receive(&backup, vsr.Message{
		Kind: vsr.Message_Kind_Commit, From: 0, To: 1, View: 0, Commit: 1,
	})
	if backup.Commit != 1 {
		t.Fatalf("expected the backup to commit op 1, got %d", backup.Commit)
	}
	if len(output.Replies) != 0 {
		t.Errorf("expected a backup to emit no Reply, got %+v", output.Replies)
	}
}

// Test_Execution_Client_Table: committing a client's op records {request-number, result, executed}
// in the client-table on every replica — here a backup, so a new primary inherits dedup state.
func Test_Execution_Client_Table(t *testing.T) {
	backup := vsr.Replica{
		Identifier:    1,
		Configuration: vsr.Configuration{0, 1, 2},
		State_Machine: vsr.State_Machine{Execute: func(
			command []byte, prediction []byte,
		) (result []byte) {
			return []byte("done")
		}},
	}
	receive(&backup, vsr.Message{
		Kind: vsr.Message_Kind_Prepare, From: 0, To: 1, View: 0, Op: 1,
		Entries: []vsr.Log_Entry{{
			View: 0, Command: []byte("x"), Client: 5, Request_Number: 8,
		}},
	})
	receive(&backup, vsr.Message{
		Kind: vsr.Message_Kind_Commit, From: 0, To: 1, View: 0, Commit: 1,
	})
	record, ok := backup.Client_Table[5]
	if !ok {
		t.Fatalf("expected a record for client 5, got %+v", backup.Client_Table)
	}
	if record.Request_Number != 8 {
		t.Errorf("expected the record's request-number 8, got %+v", record)
	}
	if !record.Executed {
		t.Errorf("expected the record marked executed, got %+v", record)
	}
	if string(record.Result) != "done" {
		t.Errorf("expected the record's cached result %q, got %q", "done", record.Result)
	}
}

// Test_Prediction_Local_Predict: with only Predict configured, the primary stamps each new entry's
// Prediction from Predict before broadcasting the Prepare; a backup adopts the entry's Prediction
// unchanged and never calls Predict.
func Test_Prediction_Local_Predict(t *testing.T) {
	primary := vsr.Replica{
		Identifier:    0,
		Configuration: vsr.Configuration{0, 1, 2},
		State_Machine: vsr.State_Machine{
			Execute: func(command []byte, prediction []byte) (result []byte) {
				return prediction
			},
			Predict: func(command []byte) (prediction []byte) {
				return []byte("predicted-" + string(command))
			},
		},
	}
	output := receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Command: []byte("now"),
	})
	if string(primary.Log[0].Prediction) != "predicted-now" {
		t.Fatalf("expected the entry stamped with the prediction, got %q",
			primary.Log[0].Prediction)
	}
	prepare, ok := first_message(output.Messages, vsr.Message_Kind_Prepare)
	if !ok {
		t.Fatalf("expected a Prepare, got %+v", output.Messages)
	}
	if string(prepare.Entries[0].Prediction) != "predicted-now" {
		t.Fatalf("expected the Prepare to carry the prediction, got %q",
			prepare.Entries[0].Prediction)
	}

	// A backup applies the Prepare and adopts the entry's prediction verbatim, never Predict.
	predicted := false
	backup := vsr.Replica{
		Identifier:    1,
		Configuration: vsr.Configuration{0, 1, 2},
		State_Machine: vsr.State_Machine{
			Execute: func(command []byte, prediction []byte) (result []byte) {
				return prediction
			},
			Predict: func(command []byte) (prediction []byte) {
				predicted = true
				return []byte("backup-must-not-predict")
			},
		},
	}
	receive(&backup, prepare)
	if string(backup.Log[0].Prediction) != "predicted-now" {
		t.Fatalf("expected the backup to adopt the prediction unchanged, got %q",
			backup.Log[0].Prediction)
	}
	if predicted {
		t.Errorf("expected the backup never to call Predict, but it did")
	}
}

// Test_Prediction_Predicted_Execution: Execute receives the entry's stored Prediction, so the
// primary and a backup both execute the op against the identical predetermined value.
func Test_Prediction_Predicted_Execution(t *testing.T) {
	var primary_prediction []byte
	primary := vsr.Replica{
		Identifier:    0,
		Configuration: vsr.Configuration{0, 1, 2},
		State_Machine: vsr.State_Machine{
			Execute: func(command []byte, prediction []byte) (result []byte) {
				primary_prediction = prediction
				return prediction
			},
			Predict: func(command []byte) (prediction []byte) {
				return []byte("ts-42")
			},
		},
	}
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Client: 9, Request_Number: 1,
		Command: []byte("stamp"),
	})
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Prepare_Ok, From: 1, To: 0, View: 0, Op: 1,
	})
	if string(primary_prediction) != "ts-42" {
		t.Fatalf("expected the primary's Execute to receive the prediction, got %q",
			primary_prediction)
	}

	// A backup executes the same op via Prepare + Commit and must see the identical bytes.
	var backup_prediction []byte
	backup := vsr.Replica{
		Identifier:    1,
		Configuration: vsr.Configuration{0, 1, 2},
		State_Machine: vsr.State_Machine{
			Execute: func(command []byte, prediction []byte) (result []byte) {
				backup_prediction = prediction
				return prediction
			},
			Predict: func(command []byte) (prediction []byte) {
				return []byte("backup-different")
			},
		},
	}
	receive(&backup, vsr.Message{
		Kind: vsr.Message_Kind_Prepare, From: 0, To: 1, View: 0, Op: 1,
		Entries: []vsr.Log_Entry{{
			View: 0, Command: []byte("stamp"), Client: 9, Request_Number: 1,
			Prediction: []byte("ts-42"),
		}},
	})
	receive(&backup, vsr.Message{
		Kind: vsr.Message_Kind_Commit, From: 0, To: 1, View: 0, Commit: 1,
	})
	if string(backup_prediction) != "ts-42" {
		t.Fatalf("expected the backup's Execute to receive the same prediction, got %q",
			backup_prediction)
	}
	if string(primary_prediction) != string(backup_prediction) {
		t.Errorf("expected identical predictions, primary %q backup %q",
			primary_prediction, backup_prediction)
	}
}

// Test_Prediction_Pre_Step_Request: with Combine configured, a client Request does not append; the
// primary broadcasts a Predict_Request carrying the command and client/request-number instead.
func Test_Prediction_Pre_Step_Request(t *testing.T) {
	primary := vsr.Replica{
		Identifier:    0,
		Configuration: vsr.Configuration{0, 1, 2},
		State_Machine: vsr.State_Machine{
			Execute: func(command []byte, prediction []byte) (result []byte) {
				return prediction
			},
			Predict: func(command []byte) (prediction []byte) {
				return []byte("p")
			},
			Combine: func(
				command []byte, responses [][]byte, own []byte,
			) (prediction []byte) {
				return own
			},
		},
	}
	output := receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Client: 9, Request_Number: 1,
		Command: []byte("stamp"),
	})
	if primary.Op != 0 {
		t.Fatalf("expected no append before the pre-step round, op is %d", primary.Op)
	}
	if _, prepared := first_message(output.Messages, vsr.Message_Kind_Prepare); prepared {
		t.Fatalf("expected no Prepare before the pre-step round, got %+v", output.Messages)
	}
	request, ok := first_message(output.Messages, vsr.Message_Kind_Predict_Request)
	if !ok {
		t.Fatalf("expected a Predict_Request, got %+v", output.Messages)
	}
	if request.From != 0 {
		t.Errorf("expected the Predict_Request from primary 0, got %+v", request)
	}
	if string(request.Command) != "stamp" {
		t.Errorf("expected the Predict_Request to carry the command, got %+v", request)
	}
	if request.Client != 9 {
		t.Errorf("expected the Predict_Request to carry client 9, got %+v", request)
	}
	if request.Request_Number != 1 {
		t.Errorf("expected the Predict_Request to carry request 1, got %+v", request)
	}
}

// Test_Prediction_Pre_Step_Combine: the primary, once it holds Predict_Response from f backups,
// calls Combine and only then appends and broadcasts the Prepare, the entry carrying the combined
// Prediction. With N=3, f=1: no Prepare before the first response, a Prepare after it.
func Test_Prediction_Pre_Step_Combine(t *testing.T) {
	primary := vsr.Replica{
		Identifier:    0,
		Configuration: vsr.Configuration{0, 1, 2},
		State_Machine: vsr.State_Machine{
			Execute: func(command []byte, prediction []byte) (result []byte) {
				return prediction
			},
			Predict: func(command []byte) (prediction []byte) {
				return []byte("own")
			},
			Combine: func(
				command []byte, responses [][]byte, own []byte,
			) (prediction []byte) {
				combined := string(own)
				for _, response := range responses {
					combined += "+" + string(response)
				}
				return []byte(combined)
			},
		},
	}
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Client: 9, Request_Number: 1,
		Command: []byte("stamp"),
	})

	// The f-th (here first) Predict_Response triggers the combine, append, and Prepare.
	output := receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Predict_Response, From: 1, To: 0, View: 0,
		Client: 9, Request_Number: 1, Result: []byte("backup-1"),
	})
	if primary.Op != 1 {
		t.Fatalf("expected the entry appended after f responses, op is %d", primary.Op)
	}
	if string(primary.Log[0].Prediction) != "own+backup-1" {
		t.Fatalf("expected the combined prediction stored, got %q",
			primary.Log[0].Prediction)
	}
	prepare, ok := first_message(output.Messages, vsr.Message_Kind_Prepare)
	if !ok {
		t.Fatalf("expected a Prepare after the pre-step round, got %+v", output.Messages)
	}
	if string(prepare.Entries[0].Prediction) != "own+backup-1" {
		t.Fatalf("expected the Prepare to carry the combined prediction, got %q",
			prepare.Entries[0].Prediction)
	}
}

// Test_Checkpoint_Take: as the commit number crosses a multiple of the checkpoint interval, the
// replica records that op as Checkpoint_Op and captures the application state through Snapshot.
func Test_Checkpoint_Take(t *testing.T) {
	snapshots := 0
	primary := vsr.Replica{
		Identifier:          0,
		Configuration:       vsr.Configuration{0, 1, 2},
		Checkpoint_Interval: 2,
		Log_Retain:          2,
		State_Machine: vsr.State_Machine{
			Execute: func(command []byte, prediction []byte) (result []byte) {
				return command
			},
			Snapshot: func() (state []byte) {
				snapshots++
				return []byte("snap")
			},
			Restore: func(state []byte) {},
		},
	}
	// Commit op 1: below the interval, no checkpoint yet.
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Client: 7, Request_Number: 1,
		Command: []byte("a"),
	})
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Prepare_Ok, From: 1, To: 0, View: 0, Op: 1,
	})
	if primary.Checkpoint_Op != 0 {
		t.Fatalf("expected no checkpoint before the interval, op %d", primary.Checkpoint_Op)
	}
	if snapshots != 0 {
		t.Fatalf("expected Snapshot not called before the interval, got %d", snapshots)
	}

	// Commit op 2: crosses the interval of 2, so the checkpoint is taken at op 2.
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Client: 7, Request_Number: 2,
		Command: []byte("b"),
	})
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Prepare_Ok, From: 1, To: 0, View: 0, Op: 2,
	})
	if primary.Checkpoint_Op != 2 {
		t.Fatalf("expected the checkpoint taken at op 2, got op %d", primary.Checkpoint_Op)
	}
	if snapshots != 1 {
		t.Fatalf("expected Snapshot called once at the checkpoint, got %d", snapshots)
	}
	if string(primary.Checkpoint_State) != "snap" {
		t.Errorf("expected the captured state stored, got %q", primary.Checkpoint_State)
	}
}

// Test_Checkpoint_Compact: after a checkpoint the replica drops every log entry at or below
// Checkpoint_Op minus the retained suffix, advancing Log_Start, never above the commit number.
func Test_Checkpoint_Compact(t *testing.T) {
	primary := vsr.Replica{
		Identifier:          0,
		Configuration:       vsr.Configuration{0, 1, 2},
		Checkpoint_Interval: 4,
		Log_Retain:          2,
		State_Machine: vsr.State_Machine{
			Execute: func(command []byte, prediction []byte) (result []byte) {
				return command
			},
			Snapshot: func() (state []byte) { return []byte("s") },
			Restore:  func(state []byte) {},
		},
	}
	// Commit ops 1..4, crossing the interval of 4 at op 4. With Log_Retain 2, entries at op <=
	// 4-2 == 2 are dropped, so ops 3 and 4 survive and Log_Start advances to 2.
	for op := 1; op <= 4; op++ {
		receive(&primary, vsr.Message{
			Kind: vsr.Message_Kind_Request, To: 0, Client: 7,
			Request_Number: vsr.Request_Number(op),
			Command:        []byte{byte('a' + op - 1)},
		})
		receive(&primary, vsr.Message{
			Kind: vsr.Message_Kind_Prepare_Ok, From: 1, To: 0, View: 0, Op: vsr.Op(op),
		})
	}
	if primary.Checkpoint_Op != 4 {
		t.Fatalf("expected the checkpoint at op 4, got op %d", primary.Checkpoint_Op)
	}
	if primary.Log_Start != 2 {
		t.Fatalf("expected Log_Start advanced to 2, got %d", primary.Log_Start)
	}
	if len(primary.Log) != 2 {
		t.Fatalf("expected the suffix of 2 retained, got log length %d", len(primary.Log))
	}
	if string(primary.Log[0].Command) != "c" {
		t.Errorf("expected op 3 = c first in the suffix, got %q", primary.Log[0].Command)
	}
	if string(primary.Log[1].Command) != "d" {
		t.Errorf("expected op 4 = d last in the suffix, got %q", primary.Log[1].Command)
	}
	if primary.Op != 4 {
		t.Errorf("expected op unchanged at 4 by compaction, got %d", primary.Op)
	}
}

// Test_Checkpoint_Offset_Index: after a prefix is compacted, op k is stored at index k - Log_Start
// - 1, and Op always equals Log_Start + len(Log). The primary, after compaction, still re-drives
// the surviving ops as Prepares carrying their correct op-numbers and entries.
func Test_Checkpoint_Offset_Index(t *testing.T) {
	primary := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 0, Configuration: vsr.Configuration{0, 1, 2},
		Heartbeat: 10, Timeout: 100, Now: 0,
		Checkpoint_Interval: 4, Log_Retain: 2,
		State_Machine: vsr.State_Machine{
			Execute: func(command []byte, prediction []byte) (result []byte) {
				return command
			},
			Snapshot: func() (state []byte) { return []byte("s") },
			Restore:  func(state []byte) {},
		},
	})
	for op := 1; op <= 4; op++ {
		receive(&primary, vsr.Message{
			Kind: vsr.Message_Kind_Request, To: 0, Client: 7,
			Request_Number: vsr.Request_Number(op),
			Command:        []byte{byte('a' + op - 1)},
		})
		receive(&primary, vsr.Message{
			Kind: vsr.Message_Kind_Prepare_Ok, From: 1, To: 0, View: 0, Op: vsr.Op(op),
		})
	}
	if vsr.Op(int(primary.Log_Start)+len(primary.Log)) != primary.Op {
		t.Fatalf("expected Op == Log_Start + len(Log), got op %d start %d len %d",
			primary.Op, primary.Log_Start, len(primary.Log))
	}
	// Op k maps to Log[k - Log_Start - 1]: op 3 at index 0 and op 4 at index 1 after the prefix
	// compacted to Log_Start 2.
	if string(primary.Log[3-primary.Log_Start-1].Command) != "c" {
		t.Errorf("expected op 3 at index 3-Log_Start-1, got %q",
			primary.Log[3-primary.Log_Start-1].Command)
	}
	if string(primary.Log[4-primary.Log_Start-1].Command) != "d" {
		t.Errorf("expected op 4 at index 4-Log_Start-1, got %q",
			primary.Log[4-primary.Log_Start-1].Command)
	}
	// Append an uncommitted op 5, then heartbeat: the re-drive reads op 5 through the offset
	// index and ships a Prepare carrying op-number 5 and entry e, proving the index serves a
	// live op above a compacted prefix rather than panicking.
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Client: 7, Request_Number: 5,
		Command: []byte("e"),
	})
	output := tick(&primary, 10)
	prepare, ok := first_message(output.Messages, vsr.Message_Kind_Prepare)
	if !ok {
		t.Fatalf("expected the heartbeat to re-drive op 5, got %+v", output.Messages)
	}
	if prepare.Op != 5 {
		t.Errorf("expected the re-drive at op 5, got %d", prepare.Op)
	}
	if string(prepare.Entries[0].Command) != "e" {
		t.Errorf("expected op 5 entry e re-driven through the offset index, got %q",
			prepare.Entries[0].Command)
	}
}

// Test_View_Change_Timeout: a backup that hears nothing from the primary past its deadline enters
// Status_View_Change and broadcasts a Start_View_Change for the next view.
func Test_View_Change_Timeout(t *testing.T) {
	backup := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 1, Configuration: vsr.Configuration{0, 1, 2},
		Heartbeat: 10, Timeout: 100, Now: 0,
	})
	output := vsr.Replica_Tick(&vsr.Replica_Tick_Input{Replica: &backup, Now: 100})
	if backup.Status != vsr.Status_View_Change {
		t.Fatalf("expected Status_View_Change, got %d", backup.Status)
	}
	if backup.View != 1 {
		t.Fatalf("expected view advanced to 1, got %d", backup.View)
	}
	if len(output.Messages) != 2 {
		t.Fatalf("expected Start_View_Change to the other two replicas, got %d",
			len(output.Messages))
	}
	recipients := map[vsr.Replica_Identifier]bool{}
	for _, message := range output.Messages {
		if message.Kind != vsr.Message_Kind_Start_View_Change {
			t.Errorf("expected a Start_View_Change, got %+v", message)
		}
		if message.From != 1 {
			t.Errorf("expected the Start_View_Change from 1, got %+v", message)
		}
		if message.View != 1 {
			t.Errorf("expected the Start_View_Change for view 1, got %+v", message)
		}
		recipients[message.To] = true
	}
	if !recipients[0] {
		t.Errorf("expected a Start_View_Change to 0, got %v", recipients)
	}
	if !recipients[2] {
		t.Errorf("expected a Start_View_Change to 2, got %v", recipients)
	}
}

// Test_View_Change_Start_View_Change: a replica that sees a Start_View_Change for a view above its
// own advances to that view, enters Status_View_Change, and rebroadcasts its own. N=5, so a single
// vote does not yet form a quorum and trigger a Do_View_Change.
func Test_View_Change_Start_View_Change(t *testing.T) {
	replica := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 2, Configuration: vsr.Configuration{0, 1, 2, 3, 4},
		Heartbeat: 10, Timeout: 100, Now: 0,
	})
	output := receive(&replica, vsr.Message{
		Kind: vsr.Message_Kind_Start_View_Change, From: 1, To: 2, View: 1,
	})
	if replica.Status != vsr.Status_View_Change {
		t.Fatalf("expected view-change, got status %d", replica.Status)
	}
	if replica.View != 1 {
		t.Fatalf("expected view 1, got view %d", replica.View)
	}
	broadcast, ok := first_message(output.Messages, vsr.Message_Kind_Start_View_Change)
	if !ok {
		t.Fatalf("expected own Start_View_Change(view 1), got %+v ok=%v", broadcast, ok)
	}
	if broadcast.From != 2 {
		t.Fatalf("expected own Start_View_Change from 2, got %+v", broadcast)
	}
	if broadcast.View != 1 {
		t.Fatalf("expected own Start_View_Change for view 1, got %+v", broadcast)
	}
	if _, sent := first_message(output.Messages, vsr.Message_Kind_Do_View_Change); sent {
		t.Error("expected no Do_View_Change before a quorum on a 5-node cluster")
	}
}

// Test_View_Change_Quorum: once a replica has Start_View_Change from a quorum of distinct replicas
// it sends a Do_View_Change — carrying a bounded log suffix, its op, and last-normal-view — to
// the new view's primary.
func Test_View_Change_Quorum(t *testing.T) {
	replica := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 2, Configuration: vsr.Configuration{0, 1, 2},
		Heartbeat: 10, Timeout: 100, Now: 0,
	})
	receive(&replica, vsr.Message{
		Kind: vsr.Message_Kind_Prepare, From: 0, To: 2, View: 0,
		Op: 1, Entries: []vsr.Log_Entry{{View: 0, Command: []byte("a")}},
	})
	// Self enters view-change for view 1, counting its own vote.
	tick(&replica, 100)
	output := receive(&replica, vsr.Message{
		Kind: vsr.Message_Kind_Start_View_Change, From: 0, To: 2, View: 1,
	})

	do_view_change, ok := first_message(output.Messages, vsr.Message_Kind_Do_View_Change)
	if !ok {
		t.Fatalf("expected a Do_View_Change at quorum, got %+v", output.Messages)
	}
	if do_view_change.To != 1 {
		t.Errorf("expected Do_View_Change to new primary 1, got %+v", do_view_change)
	}
	if do_view_change.From != 2 {
		t.Errorf("expected Do_View_Change from 2, got %+v", do_view_change)
	}
	if do_view_change.View != 1 {
		t.Errorf("expected Do_View_Change for view 1, got %+v", do_view_change)
	}
	if do_view_change.Last_Normal_View != 0 {
		t.Errorf("expected last-normal-view 0 carried, got %+v", do_view_change)
	}
	if do_view_change.Op != 1 {
		t.Errorf("expected the reporter's op 1 carried, got %+v", do_view_change)
	}
	if len(do_view_change.Log_Suffix) != 1 {
		t.Errorf("expected the replica's log suffix carried, got %+v", do_view_change)
	}
	if string(do_view_change.Log_Suffix[0].Command) != "a" {
		t.Errorf("expected the replica's log suffix carried, got %+v", do_view_change)
	}
}

// Test_View_Change_Do_View_Change: the new primary installs the view only once it holds a quorum of
// Do_View_Change; fewer leave it in Status_View_Change with no Start_View, with N=5, quorum 3.
func Test_View_Change_Do_View_Change(t *testing.T) {
	primary := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 0, Configuration: vsr.Configuration{0, 1, 2, 3, 4},
		Heartbeat: 10, Timeout: 100, Now: 0,
	})
	// Drive the new primary into view-change for view 5 (5 mod 5 == 0) via a Start_View_Change
	// quorum, which folds in its own Do_View_Change.
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Start_View_Change, From: 1, To: 0, View: 5,
	})
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Start_View_Change, From: 2, To: 0, View: 5,
	})

	output := receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Do_View_Change, From: 1, To: 0, View: 5,
	})
	if _, installed := first_message(output.Messages, vsr.Message_Kind_Start_View); installed {
		t.Fatal("expected no Start_View before a quorum of Do_View_Change")
	}
	if primary.Status != vsr.Status_View_Change {
		t.Fatalf("expected still in view-change, got status %d", primary.Status)
	}

	output = receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Do_View_Change, From: 2, To: 0, View: 5,
	})
	if _, installed := first_message(output.Messages, vsr.Message_Kind_Start_View); !installed {
		t.Fatalf("expected Start_View once the quorum is met, got %+v", output.Messages)
	}
	if primary.Status != vsr.Status_Normal {
		t.Errorf("expected normal after install, got status %d", primary.Status)
	}
	if primary.View != 5 {
		t.Errorf("expected view 5 after install, got view %d", primary.View)
	}
}

// Test_View_Change_Log_Selection: the new primary adopts the log with the largest last-normal-view,
// never merely the longest — a shorter log from a later view wins over a longer log from an
// earlier one.
func Test_View_Change_Log_Selection(t *testing.T) {
	primary := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 0, Configuration: vsr.Configuration{0, 1, 2, 3, 4},
		Heartbeat: 10, Timeout: 100, Now: 0,
	})
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Start_View_Change, From: 1, To: 0, View: 5,
	})
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Start_View_Change, From: 2, To: 0, View: 5,
	})

	// A short log from a later view (the correct winner) and a long log from an earlier view
	// (the trap for "longest wins"), each reported as a bounded suffix plus its op.
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Do_View_Change, From: 1, To: 0, View: 5,
		Last_Normal_View: 4, Op: 1,
		Log_Suffix: []vsr.Log_Entry{{View: 4, Command: []byte("x")}},
	})
	output := receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Do_View_Change, From: 2, To: 0, View: 5,
		Last_Normal_View: 3, Op: 3, Log_Suffix: []vsr.Log_Entry{
			{View: 3, Command: []byte("p")},
			{View: 3, Command: []byte("q")},
			{View: 3, Command: []byte("r")},
		},
	})
	if primary.Status != vsr.Status_Normal {
		t.Fatalf("expected install after the quorum, got status %d", primary.Status)
	}
	if primary.Op != 1 {
		t.Fatalf("expected the later-view log [x], got op %d log %+v",
			primary.Op, primary.Log)
	}
	if len(primary.Log) != 1 {
		t.Fatalf("expected the later-view log [x], got log %+v", primary.Log)
	}
	if string(primary.Log[0].Command) != "x" {
		t.Fatalf("expected the later-view log [x], got log %+v", primary.Log)
	}
	start_view, _ := first_message(output.Messages, vsr.Message_Kind_Start_View)
	if len(start_view.Log) != 1 {
		t.Errorf("expected Start_View to carry [x], got %+v", start_view.Log)
	}
	if string(start_view.Log[0].Command) != "x" {
		t.Errorf("expected Start_View to carry [x], got %+v", start_view.Log)
	}
}

// Test_View_Change_Suffix_Report: a Do_View_Change carries only a bounded log suffix — the last
// View_Change_Suffix entries — together with the reporter's op, last-normal-view, and commit, not
// its full log; a reporter whose op exceeds the suffix length reports fewer entries than it holds.
func Test_View_Change_Suffix_Report(t *testing.T) {
	replica := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 2, Configuration: vsr.Configuration{0, 1, 2},
		Heartbeat: 10, Timeout: 100, Now: 0,
	})
	// Build a log longer than the suffix bound so the report must drop entries.
	for op, command := range []string{"a", "b", "c"} {
		receive(&replica, vsr.Message{
			Kind: vsr.Message_Kind_Prepare, From: 0, To: 2, View: 0,
			Op:      vsr.Op(op + 1),
			Entries: []vsr.Log_Entry{{View: 0, Command: []byte(command)}},
		})
	}
	tick(&replica, 100) // Self enters view-change for view 1, counting its own vote.
	output := receive(&replica, vsr.Message{
		Kind: vsr.Message_Kind_Start_View_Change, From: 0, To: 2, View: 1,
	})
	report, ok := first_message(output.Messages, vsr.Message_Kind_Do_View_Change)
	if !ok {
		t.Fatalf("expected a Do_View_Change at quorum, got %+v", output.Messages)
	}
	if report.Op != 3 {
		t.Fatalf("expected the reporter's op 3 carried, got %d", report.Op)
	}
	if len(report.Log) != 0 {
		t.Errorf("expected no full log carried, got %+v", report.Log)
	}
	if len(report.Log_Suffix) != int(vsr.View_Change_Suffix) {
		t.Fatalf("expected a suffix of %d entries, got %+v",
			vsr.View_Change_Suffix, report.Log_Suffix)
	}
	// The suffix is the LAST View_Change_Suffix entries, b and c, not a, b.
	if string(report.Log_Suffix[0].Command) != "b" {
		t.Errorf("expected the suffix to begin at op 2 = b, got %q",
			report.Log_Suffix[0].Command)
	}
	if string(report.Log_Suffix[1].Command) != "c" {
		t.Errorf("expected the suffix to end at op 3 = c, got %q",
			report.Log_Suffix[1].Command)
	}
	// The op exceeds the suffix length: the report omits entries the reporter still holds.
	if int(report.Op) <= len(report.Log_Suffix) {
		t.Errorf("expected op %d to exceed the suffix length %d",
			report.Op, len(report.Log_Suffix))
	}
}

// Test_View_Change_Suffix_Fetch: when the selected reporter's op exceeds what the new primary can
// reconstruct from its own committed prefix plus the received suffix, the new primary fetches the
// rest by state transfer — a Get_State to the reporter — rather than installing a truncated log.
func Test_View_Change_Suffix_Fetch(t *testing.T) {
	primary := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 1, Configuration: vsr.Configuration{0, 1, 2},
		Heartbeat: 10, Timeout: 100, Now: 0,
	})
	// Drive replica 1 (primary of view 1) into view-change for view 1; it folds in its own
	// empty report.
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Start_View_Change, From: 0, To: 1, View: 1,
	})
	// The winner reports op 5 but a suffix of only its last two entries (ops 4, 5), and the new
	// primary holds nothing below them: ops 1..3 are missing and uncommitted on the primary, so
	// the suffix cannot be spliced onto a trusted prefix.
	output := receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Do_View_Change, From: 0, To: 1, View: 1,
		Last_Normal_View: 0, Op: 5, Commit: 3,
		Log_Suffix: []vsr.Log_Entry{
			{View: 0, Command: []byte("d")}, {View: 0, Command: []byte("e")},
		},
	})
	if _, installed := first_message(output.Messages, vsr.Message_Kind_Start_View); installed {
		t.Fatal("expected no Start_View when the selected log cannot be reconstructed")
	}
	request, ok := first_message(output.Messages, vsr.Message_Kind_Get_State)
	if !ok {
		t.Fatalf("expected a Get_State to fetch the rest, got %+v", output.Messages)
	}
	if request.To != 0 {
		t.Errorf("expected the Get_State to the selected reporter 0, got %d", request.To)
	}
	if request.View != 1 {
		t.Errorf("expected the Get_State in the new view 1, got %d", request.View)
	}
}

// Test_View_Change_Deferred_Install: the new primary stays in Status_View_Change and broadcasts no
// Start_View until the awaited state arrives; only then does it install — with op at least the
// selected reporter's op so no committed op is dropped — return to normal, and emit Start_View
// exactly once.
func Test_View_Change_Deferred_Install(t *testing.T) {
	primary := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 1, Configuration: vsr.Configuration{0, 1, 2},
		Heartbeat: 10, Timeout: 100, Now: 0,
	})
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Start_View_Change, From: 0, To: 1, View: 1,
	})
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Do_View_Change, From: 0, To: 1, View: 1,
		Last_Normal_View: 0, Op: 5, Commit: 3,
		Log_Suffix: []vsr.Log_Entry{
			{View: 0, Command: []byte("d")}, {View: 0, Command: []byte("e")},
		},
	})
	// The fetch is outstanding: the install is deferred, so the primary is still changing views
	// and has emitted no Start_View.
	if primary.Status != vsr.Status_View_Change {
		t.Fatalf("expected the primary still in view-change while fetching, got status %d",
			primary.Status)
	}

	// A stray New_State from a non-selected replica must not prematurely install.
	stray := receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_New_State, From: 2, To: 1, View: 1, Op: 5, Commit: 3,
		Log: []vsr.Log_Entry{
			{View: 0, Command: []byte("p")}, {View: 0, Command: []byte("q")},
			{View: 0, Command: []byte("r")}, {View: 0, Command: []byte("d")},
			{View: 0, Command: []byte("e")},
		},
	})
	if _, installed := first_message(stray.Messages, vsr.Message_Kind_Start_View); installed {
		t.Fatal("expected no Start_View from a stray New_State while still awaiting")
	}
	if primary.Status != vsr.Status_View_Change {
		t.Fatalf("expected a stray New_State ignored, status is %d", primary.Status)
	}

	// The awaited New_State arrives from the selected reporter 0, carrying the complete log.
	output := receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_New_State, From: 0, To: 1, View: 1, Op: 5, Commit: 3,
		Log: []vsr.Log_Entry{
			{View: 0, Command: []byte("a")}, {View: 0, Command: []byte("b")},
			{View: 0, Command: []byte("c")}, {View: 0, Command: []byte("d")},
			{View: 0, Command: []byte("e")},
		},
	})
	if primary.Status != vsr.Status_Normal {
		t.Fatalf("expected install once the awaited state arrives, status is %d",
			primary.Status)
	}
	if primary.Op < 5 {
		t.Fatalf("expected the installed op >= the reporter's op 5, got %d", primary.Op)
	}
	// Exactly one Start_View per backup, and no more — the deferred install emits it once.
	starts := 0
	for _, message := range output.Messages {
		if message.Kind == vsr.Message_Kind_Start_View {
			starts++
		}
	}
	if starts != len(primary.Configuration)-1 {
		t.Errorf("expected one Start_View per backup and no more, got %d", starts)
	}
}

// Test_View_Change_Start_View: on install the new primary returns to normal in the new view and
// broadcasts a Start_View carrying the selected log to every backup.
func Test_View_Change_Start_View(t *testing.T) {
	primary := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 1, Configuration: vsr.Configuration{0, 1, 2},
		Heartbeat: 10, Timeout: 100, Now: 0,
	})
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Start_View_Change, From: 0, To: 1, View: 1,
	})
	output := receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Do_View_Change, From: 0, To: 1, View: 1,
		Last_Normal_View: 0, Op: 1, Commit: 1,
		Log_Suffix: []vsr.Log_Entry{{View: 0, Command: []byte("a")}},
	})
	if primary.Status != vsr.Status_Normal {
		t.Fatalf("expected normal in view 1, got status %d", primary.Status)
	}
	if primary.View != 1 {
		t.Fatalf("expected view 1, got view %d", primary.View)
	}
	recipients := map[vsr.Replica_Identifier]bool{}
	for _, message := range output.Messages {
		if message.Kind != vsr.Message_Kind_Start_View {
			continue
		}
		if message.From != 1 {
			t.Errorf("expected Start_View from 1, got %+v", message)
		}
		if message.View != 1 {
			t.Errorf("expected Start_View for view 1, got %+v", message)
		}
		recipients[message.To] = true
	}
	if !recipients[0] {
		t.Errorf("expected a Start_View to backup 0, got %v", recipients)
	}
	if !recipients[2] {
		t.Errorf("expected a Start_View to backup 2, got %v", recipients)
	}
}

// Test_View_Change_Adoption: a backup that receives a Start_View adopts its log, view, op, and
// commit, and returns to normal.
func Test_View_Change_Adoption(t *testing.T) {
	backup := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 2, Configuration: vsr.Configuration{0, 1, 2},
		Heartbeat: 10, Timeout: 100, Now: 0,
	})
	receive(&backup, vsr.Message{
		Kind: vsr.Message_Kind_Start_View, From: 1, To: 2, View: 1, Commit: 1, Op: 2,
		Log: []vsr.Log_Entry{
			{View: 0, Command: []byte("a")}, {View: 1, Command: []byte("b")},
		},
	})
	if backup.Status != vsr.Status_Normal {
		t.Fatalf("expected normal in view 1, got status %d", backup.Status)
	}
	if backup.View != 1 {
		t.Fatalf("expected view 1, got view %d", backup.View)
	}
	if backup.Op != 2 {
		t.Fatalf("expected adopted op 2, got op %d", backup.Op)
	}
	if len(backup.Log) != 2 {
		t.Fatalf("expected adopted log of 2, got log %d", len(backup.Log))
	}
	if backup.Commit != 1 {
		t.Fatalf("expected adopted commit 1, got commit %d", backup.Commit)
	}
	if string(backup.Log[1].Command) != "b" {
		t.Errorf("expected adopted entry b at op 2, got %q", backup.Log[1].Command)
	}
}

// Test_View_Change_Commit_Preservation: an entry committed in an earlier view is present in the log
// the new primary installs, so no acknowledged command is lost across the change.
func Test_View_Change_Commit_Preservation(t *testing.T) {
	primary := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 0, Configuration: vsr.Configuration{0, 1, 2},
		Heartbeat: 10, Timeout: 100, Now: 0,
	})
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Start_View_Change, From: 1, To: 0, View: 3,
	})
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Do_View_Change, From: 2, To: 0, View: 3,
		Last_Normal_View: 1, Op: 2, Commit: 1,
		Log_Suffix: []vsr.Log_Entry{
			{View: 1, Command: []byte("a")}, {View: 1, Command: []byte("b")},
		},
	})
	if primary.Status != vsr.Status_Normal {
		t.Fatalf("expected install, got status %d", primary.Status)
	}
	if len(primary.Log) < 1 {
		t.Fatalf("expected the committed entry a preserved at op 1, got %+v", primary.Log)
	}
	if string(primary.Log[0].Command) != "a" {
		t.Fatalf("expected the committed entry a preserved at op 1, got %+v", primary.Log)
	}
	if primary.Commit < 1 {
		t.Errorf("expected the new primary to know op 1 committed, got commit %d",
			primary.Commit)
	}
}

// Test_Recovery_Nonce: a recovering replica enters Status_Recovery and broadcasts a Recovery
// carrying its fresh nonce to the rest of the cluster.
func Test_Recovery_Nonce(t *testing.T) {
	replica := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 1, Configuration: vsr.Configuration{0, 1, 2},
		Heartbeat: 10, Timeout: 100, Now: 0,
	})
	output := vsr.Replica_Recover(&vsr.Replica_Recover_Input{
		Replica: &replica, Nonce: 42, Now: 0,
	})
	if replica.Status != vsr.Status_Recovery {
		t.Fatalf("expected Status_Recovery, got %d", replica.Status)
	}
	if len(output.Messages) != 2 {
		t.Fatalf("expected Recovery to the other two replicas, got %d",
			len(output.Messages))
	}
	for _, message := range output.Messages {
		if message.Kind != vsr.Message_Kind_Recovery {
			t.Errorf("expected a Recovery, got %+v", message)
		}
		if message.From != 1 {
			t.Errorf("expected the Recovery from 1, got %+v", message)
		}
		if message.Nonce != 42 {
			t.Errorf("expected the Recovery to carry nonce 42, got %+v", message)
		}
	}
}

// Test_Recovery_Response: a normal primary answers a Recovery with its log, op, and commit; a
// normal backup answers with only its view. Both echo the nonce.
func Test_Recovery_Response(t *testing.T) {
	primary := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 0, Configuration: vsr.Configuration{0, 1, 2},
		Heartbeat: 10, Timeout: 100, Now: 0,
	})
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Command: []byte("x"),
	})
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Prepare_Ok, From: 1, To: 0, View: 0, Op: 1,
	})

	output := receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Recovery, From: 1, To: 0, Nonce: 42,
	})
	response, ok := first_message(output.Messages, vsr.Message_Kind_Recovery_Response)
	if !ok {
		t.Fatalf("expected Recovery_Response(nonce 42) from 0 to 1, got %+v ok=%v",
			response, ok)
	}
	if response.To != 1 {
		t.Fatalf("expected the response to 1, got %+v", response)
	}
	if response.From != 0 {
		t.Fatalf("expected the response from 0, got %+v", response)
	}
	if response.Nonce != 42 {
		t.Fatalf("expected the response to echo nonce 42, got %+v", response)
	}
	if response.View != 0 {
		t.Fatalf("expected the response to report view 0, got %+v", response)
	}
	if len(response.Log) != 1 {
		t.Errorf("expected the primary to report its log, got %+v", response)
	}
	if string(response.Log[0].Command) != "x" {
		t.Errorf("expected the primary to report its log, got %+v", response)
	}
	if response.Op != 1 {
		t.Errorf("expected the primary to report op 1, got %+v", response)
	}
	if response.Commit != 1 {
		t.Errorf("expected the primary to report commit 1, got %+v", response)
	}

	backup := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 2, Configuration: vsr.Configuration{0, 1, 2},
		Heartbeat: 10, Timeout: 100, Now: 0,
	})
	output = receive(&backup, vsr.Message{
		Kind: vsr.Message_Kind_Recovery, From: 1, To: 2, Nonce: 7,
	})
	response, _ = first_message(output.Messages, vsr.Message_Kind_Recovery_Response)
	if response.From != 2 {
		t.Errorf("expected a backup to answer from 2, got %+v", response)
	}
	if response.Nonce != 7 {
		t.Errorf("expected a backup to echo nonce 7, got %+v", response)
	}
	if len(response.Log) != 0 {
		t.Errorf("expected a backup to answer with no log, got %+v", response)
	}
}

// Test_Recovery_Stale_Rejection: a Recovery_Response whose nonce differs from the in-flight one is
// ignored, so a delayed response from an earlier attempt cannot complete the current one.
func Test_Recovery_Stale_Rejection(t *testing.T) {
	replica := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 1, Configuration: vsr.Configuration{0, 1, 2},
		Heartbeat: 10, Timeout: 100, Now: 0,
	})
	vsr.Replica_Recover(&vsr.Replica_Recover_Input{Replica: &replica, Nonce: 42, Now: 0})
	receive(&replica, vsr.Message{
		Kind: vsr.Message_Kind_Recovery_Response, From: 0, To: 1, Nonce: 42, View: 0,
		Op: 1, Commit: 1, Log: []vsr.Log_Entry{{View: 0, Command: []byte("x")}},
	})
	// A stale response from the second replica would, if wrongly counted, form a quorum that
	// includes the primary and complete recovery.
	receive(&replica, vsr.Message{
		Kind: vsr.Message_Kind_Recovery_Response, From: 2, To: 1, Nonce: 99, View: 0,
	})
	if replica.Status != vsr.Status_Recovery {
		t.Fatalf("expected a stale-nonce response to be ignored, status is %d",
			replica.Status)
	}
}

// Test_Recovery_Quorum: recovery completes only with responses from a quorum of distinct replicas
// that includes the primary of the highest view among them.
func Test_Recovery_Quorum(t *testing.T) {
	replica := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 2, Configuration: vsr.Configuration{0, 1, 2, 3, 4},
		Heartbeat: 10, Timeout: 100, Now: 0,
	})
	vsr.Replica_Recover(&vsr.Replica_Recover_Input{Replica: &replica, Nonce: 42, Now: 0})
	// A quorum of backups (3 of 5) replies, but none is the primary of view 0.
	for _, identifier := range []vsr.Replica_Identifier{1, 3, 4} {
		receive(&replica, vsr.Message{
			Kind: vsr.Message_Kind_Recovery_Response, From: identifier, To: 2,
			Nonce: 42, View: 0,
		})
	}
	if replica.Status != vsr.Status_Recovery {
		t.Fatalf("expected to wait for the primary's response, status is %d",
			replica.Status)
	}
	receive(&replica, vsr.Message{
		Kind: vsr.Message_Kind_Recovery_Response, From: 0, To: 2, Nonce: 42, View: 0,
		Op: 1, Commit: 1, Log: []vsr.Log_Entry{{View: 0, Command: []byte("x")}},
	})
	if replica.Status != vsr.Status_Normal {
		t.Fatalf("expected recovery to complete once the primary replies, status is %d",
			replica.Status)
	}
}

// Test_Recovery_Rejoin: a recovered replica adopts the primary's log, op, and commit and returns to
// normal in that view.
func Test_Recovery_Rejoin(t *testing.T) {
	replica := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 1, Configuration: vsr.Configuration{0, 1, 2},
		Heartbeat: 10, Timeout: 100, Now: 0,
	})
	vsr.Replica_Recover(&vsr.Replica_Recover_Input{Replica: &replica, Nonce: 42, Now: 0})
	receive(&replica, vsr.Message{
		Kind: vsr.Message_Kind_Recovery_Response, From: 2, To: 1, Nonce: 42, View: 0,
	})
	receive(&replica, vsr.Message{
		Kind: vsr.Message_Kind_Recovery_Response, From: 0, To: 1, Nonce: 42, View: 0,
		Op: 2, Commit: 1, Log: []vsr.Log_Entry{
			{View: 0, Command: []byte("a")}, {View: 0, Command: []byte("b")},
		},
	})
	if replica.Status != vsr.Status_Normal {
		t.Fatalf("expected normal in view 0, got status %d", replica.Status)
	}
	if replica.View != 0 {
		t.Fatalf("expected view 0, got view %d", replica.View)
	}
	if replica.Op != 2 {
		t.Fatalf("expected adopted op 2, got op %d", replica.Op)
	}
	if len(replica.Log) != 2 {
		t.Fatalf("expected adopted log of 2, got log %d", len(replica.Log))
	}
	if replica.Commit != 1 {
		t.Fatalf("expected adopted commit 1, got commit %d", replica.Commit)
	}
	if string(replica.Log[1].Command) != "b" {
		t.Errorf("expected adopted entry b, got %q", replica.Log[1].Command)
	}
}

// Test_Recovery_Quiescence: a recovering replica ignores a Start_View_Change and a Start_View, so
// its wiped log never enters a view change; it rejoins only through Recovery_Response.
func Test_Recovery_Quiescence(t *testing.T) {
	replica := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 1, Configuration: vsr.Configuration{0, 1, 2},
		Heartbeat: 10, Timeout: 100, Now: 0,
	})
	vsr.Replica_Recover(&vsr.Replica_Recover_Input{Replica: &replica, Nonce: 7, Now: 0})

	output := receive(&replica, vsr.Message{
		Kind: vsr.Message_Kind_Start_View_Change, From: 2, To: 1, View: 1,
	})
	if replica.Status != vsr.Status_Recovery {
		t.Fatalf("expected a recovering replica to ignore Start_View_Change, status %d",
			replica.Status)
	}
	if len(output.Messages) != 0 {
		t.Errorf("recovering replica must stay quiescent, it sent %+v", output.Messages)
	}

	receive(&replica, vsr.Message{
		Kind: vsr.Message_Kind_Start_View, From: 1, To: 1, View: 1, Op: 1,
		Log: []vsr.Log_Entry{{View: 1, Command: []byte("z")}},
	})
	if replica.Status != vsr.Status_Recovery {
		t.Fatalf("recovering replica must ignore Start_View, status %d", replica.Status)
	}
}

// Test_Recovery_Checkpoint_Advertise: a replica recovering warm keeps its checkpoint and advertises
// its Checkpoint_Op in the Recovery it broadcasts, so the answering primary knows which prefix it
// already holds.
func Test_Recovery_Checkpoint_Advertise(t *testing.T) {
	replica := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 1, Configuration: vsr.Configuration{0, 1, 2},
		Heartbeat: 10, Timeout: 100, Now: 0,
		Checkpoint_Interval: 4, Log_Retain: 2,
		State_Machine: vsr.State_Machine{
			Execute: func(command []byte, prediction []byte) (result []byte) {
				return command
			},
			Snapshot: func() (state []byte) { return []byte("s") },
			Restore:  func(state []byte) {},
		},
	})
	// Stand the replica up with a checkpoint at op 4 (as if it had taken one before crashing).
	replica.Checkpoint_Op = 4
	replica.Checkpoint_State = []byte("disk-snap")
	replica.Log_Start = 2

	output := vsr.Replica_Recover(&vsr.Replica_Recover_Input{
		Replica: &replica, Nonce: 42, Now: 0, Keep_Checkpoint: true,
	})
	recovery, ok := first_message(output.Messages, vsr.Message_Kind_Recovery)
	if !ok {
		t.Fatalf("expected a Recovery broadcast, got %+v", output.Messages)
	}
	if recovery.Checkpoint_Op != 4 {
		t.Errorf("expected the Recovery to advertise Checkpoint_Op 4, got %d",
			recovery.Checkpoint_Op)
	}
}

// Test_Recovery_Checkpoint_Restore: a replica recovering warm restores its application state from
// the surviving checkpoint, keeps Log_Start at the checkpoint op, and replays only the
// un-checkpointed suffix the primary ships, rather than re-executing the whole log from empty.
func Test_Recovery_Checkpoint_Restore(t *testing.T) {
	var restored []byte
	var executed []string
	replica := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 1, Configuration: vsr.Configuration{0, 1, 2},
		Heartbeat: 10, Timeout: 100, Now: 0,
		Checkpoint_Interval: 4, Log_Retain: 2,
		State_Machine: vsr.State_Machine{
			Execute: func(command []byte, prediction []byte) (result []byte) {
				executed = append(executed, string(command))
				return command
			},
			Snapshot: func() (state []byte) { return []byte("s") },
			Restore: func(state []byte) {
				restored = state
			},
		},
	})
	replica.Checkpoint_Op = 4
	replica.Checkpoint_State = []byte("disk-snap")
	replica.Log_Start = 2
	vsr.Replica_Recover(&vsr.Replica_Recover_Input{
		Replica: &replica, Nonce: 42, Now: 0, Keep_Checkpoint: true,
	})

	// The backup answers so a quorum forms; the primary ships the suffix from the checkpoint op
	// forward, op 5 and op 6, with commit 6.
	receive(&replica, vsr.Message{
		Kind: vsr.Message_Kind_Recovery_Response, From: 2, To: 1, Nonce: 42, View: 0,
	})
	receive(&replica, vsr.Message{
		Kind: vsr.Message_Kind_Recovery_Response, From: 0, To: 1, Nonce: 42, View: 0,
		Op: 6, Commit: 6, Checkpoint_Op: 4, Log: []vsr.Log_Entry{
			{View: 0, Command: []byte("e")}, {View: 0, Command: []byte("f")},
		},
	})
	if replica.Status != vsr.Status_Normal {
		t.Fatalf("expected normal after recovery, got status %d", replica.Status)
	}
	if string(restored) != "disk-snap" {
		t.Fatalf("expected Restore from the surviving checkpoint, got %q", restored)
	}
	if replica.Log_Start != 4 {
		t.Fatalf("expected Log_Start kept at checkpoint op 4, got %d", replica.Log_Start)
	}
	if replica.Op != 6 {
		t.Fatalf("expected op 6 adopted, got %d", replica.Op)
	}
	// Only the un-checkpointed suffix (ops 5, 6) is replayed, never the whole log from empty.
	if len(executed) != 2 {
		t.Fatalf("expected only the suffix replayed, got %v", executed)
	}
	if executed[0] != "e" {
		t.Errorf("expected op 5 = e replayed first, got %v", executed)
	}
	if executed[1] != "f" {
		t.Errorf("expected op 6 = f replayed second, got %v", executed)
	}
}

// Test_State_Transfer_Behind_View: a normal replica that hears a later view drops its uncommitted
// tail, adopts the view, and sends a Get_State to the sender.
func Test_State_Transfer_Behind_View(t *testing.T) {
	backup := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 1, Configuration: vsr.Configuration{0, 1, 2},
		Heartbeat: 10, Timeout: 100, Now: 0,
	})
	receive(&backup, vsr.Message{
		Kind: vsr.Message_Kind_Prepare, From: 0, To: 1, View: 0,
		Op: 1, Entries: []vsr.Log_Entry{{View: 0, Command: []byte("a")}},
	})
	receive(&backup, vsr.Message{
		Kind: vsr.Message_Kind_Prepare, From: 0, To: 1, View: 0,
		Op: 2, Commit: 1, Entries: []vsr.Log_Entry{{View: 0, Command: []byte("b")}},
	})
	output := receive(&backup, vsr.Message{
		Kind: vsr.Message_Kind_Prepare, From: 2, To: 1, View: 2,
		Op: 5, Entries: []vsr.Log_Entry{{View: 2, Command: []byte("x")}},
	})
	if backup.View != 2 {
		t.Fatalf("expected the later view 2 adopted, got %d", backup.View)
	}
	if backup.Op != 1 {
		t.Fatalf("expected the uncommitted tail dropped to commit, got op %d", backup.Op)
	}
	if len(backup.Log) != 1 {
		t.Fatalf("expected the log dropped to one entry, got %d", len(backup.Log))
	}
	if string(backup.Log[0].Command) != "a" {
		t.Errorf("expected the committed entry a retained, got %q", backup.Log[0].Command)
	}
	request, ok := first_message(output.Messages, vsr.Message_Kind_Get_State)
	if !ok {
		t.Fatalf("expected a Get_State, got %+v", output.Messages)
	}
	if request.To != 2 {
		t.Errorf("expected the Get_State to the sender 2, got %d", request.To)
	}
	if request.View != 2 {
		t.Errorf("expected the Get_State in view 2, got %d", request.View)
	}
}

// Test_State_Transfer_Gap: a normal replica that receives a Prepare beyond its next op sends a
// Get_State for the missing entries instead of acking.
func Test_State_Transfer_Gap(t *testing.T) {
	backup := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 1, Configuration: vsr.Configuration{0, 1, 2},
		Heartbeat: 10, Timeout: 100, Now: 0,
	})
	receive(&backup, vsr.Message{
		Kind: vsr.Message_Kind_Prepare, From: 0, To: 1, View: 0,
		Op: 1, Entries: []vsr.Log_Entry{{View: 0, Command: []byte("a")}},
	})
	output := receive(&backup, vsr.Message{
		Kind: vsr.Message_Kind_Prepare, From: 0, To: 1, View: 0,
		Op: 3, Entries: []vsr.Log_Entry{{View: 0, Command: []byte("c")}},
	})
	if backup.Op != 1 {
		t.Fatalf("expected the gap not applied, op %d", backup.Op)
	}
	request, ok := first_message(output.Messages, vsr.Message_Kind_Get_State)
	if !ok {
		t.Fatalf("expected a Get_State, got %+v", output.Messages)
	}
	if request.To != 0 {
		t.Errorf("expected the Get_State to the primary 0, got %d", request.To)
	}
	if request.View != 0 {
		t.Errorf("expected the Get_State in view 0, got %d", request.View)
	}
	if _, acked := first_message(output.Messages, vsr.Message_Kind_Prepare_Ok); acked {
		t.Errorf("expected no ack on a gap, got %+v", output.Messages)
	}
}

// Test_State_Transfer_Response: a normal replica answers a Get_State with its log, op, and commit.
func Test_State_Transfer_Response(t *testing.T) {
	primary := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 0, Configuration: vsr.Configuration{0, 1, 2},
		Heartbeat: 10, Timeout: 100, Now: 0,
	})
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Request_Number: 1, Command: []byte("a"),
	})
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Prepare_Ok, From: 1, To: 0, View: 0, Op: 1,
	})
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Request_Number: 2, Command: []byte("b"),
	})

	output := receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Get_State, From: 1, To: 0, View: 0, Op: 0,
	})
	state, ok := first_message(output.Messages, vsr.Message_Kind_New_State)
	if !ok {
		t.Fatalf("expected a New_State, got %+v", output.Messages)
	}
	if state.To != 1 {
		t.Errorf("expected the New_State to 1, got %d", state.To)
	}
	if state.View != 0 {
		t.Errorf("expected the New_State in view 0, got %d", state.View)
	}
	if state.Op != 2 {
		t.Errorf("expected the New_State op 2, got %d", state.Op)
	}
	if state.Commit != 1 {
		t.Errorf("expected the New_State commit 1, got %d", state.Commit)
	}
	if len(state.Log) != 2 {
		t.Fatalf("expected the responder's full log of 2, got %d", len(state.Log))
	}
	if string(state.Log[1].Command) != "b" {
		t.Errorf("expected the responder's log entry b, got %q", state.Log[1].Command)
	}
}

// Test_State_Transfer_Apply: a replica that receives a New_State adopts the log, op, commit, and
// view, and resumes normal operation.
func Test_State_Transfer_Apply(t *testing.T) {
	replica := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 1, Configuration: vsr.Configuration{0, 1, 2},
		Heartbeat: 10, Timeout: 100, Now: 0,
	})
	receive(&replica, vsr.Message{
		Kind: vsr.Message_Kind_New_State, From: 0, To: 1, View: 2, Op: 3, Commit: 2,
		Log: []vsr.Log_Entry{
			{View: 0, Command: []byte("a")},
			{View: 0, Command: []byte("b")},
			{View: 2, Command: []byte("c")},
		},
	})
	if replica.Status != vsr.Status_Normal {
		t.Fatalf("expected normal status, got %d", replica.Status)
	}
	if replica.View != 2 {
		t.Fatalf("expected view 2, got %d", replica.View)
	}
	if replica.Op != 3 {
		t.Fatalf("expected adopted op 3, got %d", replica.Op)
	}
	if len(replica.Log) != 3 {
		t.Fatalf("expected adopted log of 3, got %d", len(replica.Log))
	}
	if replica.Commit != 2 {
		t.Fatalf("expected adopted commit 2, got %d", replica.Commit)
	}
	if string(replica.Log[2].Command) != "c" {
		t.Errorf("expected adopted entry c, got %q", replica.Log[2].Command)
	}
}

// Test_State_Transfer_Checkpoint_Gap: a Get_State for an op the responder has garbage-collected is
// answered with a New_State carrying the responder's checkpoint and the log from the checkpoint
// forward, since the requested prefix no longer exists to ship.
func Test_State_Transfer_Checkpoint_Gap(t *testing.T) {
	primary := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 0, Configuration: vsr.Configuration{0, 1, 2},
		Heartbeat: 10, Timeout: 100, Now: 0,
		Checkpoint_Interval: 4, Log_Retain: 2,
		State_Machine: vsr.State_Machine{
			Execute: func(command []byte, prediction []byte) (result []byte) {
				return command
			},
			Snapshot: func() (state []byte) { return []byte("snap-at-4") },
			Restore:  func(state []byte) {},
		},
	})
	// Commit ops 1..4 so the checkpoint lands at op 4 and the prefix (ops 1, 2) is compacted.
	for op := 1; op <= 4; op++ {
		receive(&primary, vsr.Message{
			Kind: vsr.Message_Kind_Request, To: 0, Client: 7,
			Request_Number: vsr.Request_Number(op),
			Command:        []byte{byte('a' + op - 1)},
		})
		receive(&primary, vsr.Message{
			Kind: vsr.Message_Kind_Prepare_Ok, From: 1, To: 0, View: 0, Op: vsr.Op(op),
		})
	}
	if primary.Log_Start != 2 {
		t.Fatalf("expected the prefix compacted to Log_Start 2, got %d", primary.Log_Start)
	}

	// A requester asks for op 1 — below Log_Start, already GC'd. The responder cannot ship
	// it, so it answers with the checkpoint plus the suffix from the checkpoint op forward.
	output := receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Get_State, From: 1, To: 0, View: 0, Op: 1,
	})
	state, ok := first_message(output.Messages, vsr.Message_Kind_New_State)
	if !ok {
		t.Fatalf("expected a New_State, got %+v", output.Messages)
	}
	if state.Checkpoint_Op != 4 {
		t.Errorf("expected New_State to carry Checkpoint_Op 4, got %d", state.Checkpoint_Op)
	}
	if string(state.Checkpoint_State) != "snap-at-4" {
		t.Errorf("expected the checkpoint state shipped, got %q", state.Checkpoint_State)
	}
	if state.Op != 4 {
		t.Errorf("expected the New_State op 4, got %d", state.Op)
	}
	// The log carries the suffix from the checkpoint op forward — here only op 4, since the
	// checkpoint already covers ops 1..4 and Log_Retain kept ops 3, 4 but op 3 is below the
	// checkpoint op so the shipped suffix starts after the checkpoint op.
	if len(state.Log) != 0 {
		t.Errorf("expected no entries beyond the checkpoint op, got %+v", state.Log)
	}
}

// Test_State_Transfer_Checkpoint_Apply: a replica that receives a New_State carrying a checkpoint
// restores from it, sets Log_Start to the checkpoint op, adopts the suffix and the view the
// checkpoint was taken in, and resumes normal operation.
func Test_State_Transfer_Checkpoint_Apply(t *testing.T) {
	var restored []byte
	var executed []string
	replica := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 1, Configuration: vsr.Configuration{0, 1, 2},
		Heartbeat: 10, Timeout: 100, Now: 0,
		Checkpoint_Interval: 4, Log_Retain: 2,
		State_Machine: vsr.State_Machine{
			Execute: func(command []byte, prediction []byte) (result []byte) {
				executed = append(executed, string(command))
				return command
			},
			Snapshot: func() (state []byte) { return []byte("s") },
			Restore: func(state []byte) {
				restored = state
			},
		},
	})
	receive(&replica, vsr.Message{
		Kind: vsr.Message_Kind_New_State, From: 0, To: 1, View: 3, Op: 6, Commit: 6,
		Checkpoint_Op: 4, Checkpoint_State: []byte("snap-at-4"),
		Log: []vsr.Log_Entry{
			{View: 3, Command: []byte("e")}, {View: 3, Command: []byte("f")},
		},
	})
	if replica.Status != vsr.Status_Normal {
		t.Fatalf("expected normal status, got %d", replica.Status)
	}
	if string(restored) != "snap-at-4" {
		t.Fatalf("expected Restore from the checkpoint, got %q", restored)
	}
	if replica.View != 3 {
		t.Fatalf("expected the checkpoint's view 3 adopted, got %d", replica.View)
	}
	if replica.Log_Start != 4 {
		t.Fatalf("expected Log_Start set to the checkpoint op 4, got %d", replica.Log_Start)
	}
	if replica.Op != 6 {
		t.Fatalf("expected op 6 adopted, got %d", replica.Op)
	}
	if replica.Commit != 6 {
		t.Fatalf("expected commit 6 adopted, got %d", replica.Commit)
	}
	if len(replica.Log) != 2 {
		t.Fatalf("expected the suffix of 2 adopted, got %d", len(replica.Log))
	}
	// Only the committed ops after the checkpoint are replayed — ops 5, 6 — not the prefix
	// the checkpoint already covers.
	if len(executed) != 2 {
		t.Fatalf("expected only the post-checkpoint suffix executed, got %v", executed)
	}
	if executed[0] != "e" {
		t.Errorf("expected op 5 = e executed first, got %v", executed)
	}
	if executed[1] != "f" {
		t.Errorf("expected op 6 = f executed second, got %v", executed)
	}
}

// Test_Reconfiguration_Request: the primary accepts a Reconfiguration only when its epoch matches,
// the request-number is fresh, and the new configuration has at least three members; it appends the
// request, broadcasts a Prepare carrying the new configuration, and then refuses further client
// requests, the reconfiguration being the epoch's last op.
func Test_Reconfiguration_Request(t *testing.T) {
	primary := vsr.Replica{Identifier: 0, Configuration: vsr.Configuration{0, 1, 2}}
	output := receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Reconfiguration, To: 0, Client: 99, Request_Number: 1,
		Epoch: 0, New_Configuration: vsr.Configuration{0, 1, 2, 3, 4},
	})
	if primary.Op != 1 {
		t.Fatalf("expected the reconfiguration appended at op 1, got %d", primary.Op)
	}
	prepare, ok := first_message(output.Messages, vsr.Message_Kind_Prepare)
	if !ok {
		t.Fatalf("expected a Prepare for the reconfiguration, got %+v", output.Messages)
	}
	if len(prepare.Entries[0].New_Configuration) != 5 {
		t.Fatalf("expected the Prepare entry to carry the new configuration, got %+v",
			prepare.Entries[0])
	}

	// A new configuration with fewer than three members is rejected: no append.
	primary2 := vsr.Replica{Identifier: 0, Configuration: vsr.Configuration{0, 1, 2}}
	receive(&primary2, vsr.Message{
		Kind: vsr.Message_Kind_Reconfiguration, To: 0, Client: 99, Request_Number: 1,
		Epoch: 0, New_Configuration: vsr.Configuration{0, 1},
	})
	if primary2.Op != 0 {
		t.Fatalf("expected a too-small new configuration rejected, op %d", primary2.Op)
	}

	// After accepting the reconfiguration the primary stops accepting client requests.
	dropped := receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Client: 7, Request_Number: 1,
		Command: []byte("after"),
	})
	if primary.Op != 1 {
		t.Fatalf("expected no client request accepted after a reconfiguration, op %d",
			primary.Op)
	}
	if len(dropped.Messages) != 0 {
		t.Errorf("expected no Prepare for a post-reconfiguration request, got %+v",
			dropped.Messages)
	}
}

// Test_Reconfiguration_Threshold: the deciding quorum is recomputed for the membership that owns
// the decision, not inherited. A reconfiguration that GROWS the group 3->5 still commits in the OLD
// group at its quorum of 2 (the primary plus one ack), not the new group's 3, since the old group
// owns the reconfiguration op until the epoch advances.
func Test_Reconfiguration_Threshold(t *testing.T) {
	primary := vsr.Replica{Identifier: 0, Configuration: vsr.Configuration{0, 1, 2}}
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Reconfiguration, To: 0, Client: 99, Request_Number: 1,
		Epoch: 0, New_Configuration: vsr.Configuration{0, 1, 2, 3, 4},
	})
	// One ack from the old group (a quorum of two with the primary) commits the
	// reconfiguration, even though the new group of five would need three. The epoch increments
	// on that commit.
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Prepare_Ok, From: 1, To: 0, View: 0, Epoch: 0, Op: 1,
	})
	if primary.Commit != 1 {
		t.Fatalf("expected the reconfiguration committed at the old group's quorum, "+
			"commit %d", primary.Commit)
	}
	if primary.Epoch != 1 {
		t.Fatalf("expected the epoch incremented on the reconfiguration commit, epoch %d",
			primary.Epoch)
	}
}

// Test_Reconfiguration_Epoch_Increment: when the primary holds a quorum of Prepare_Ok for the
// reconfiguration op it increments its epoch, sends a Commit to the other old replicas, sends a
// Start_Epoch to the added nodes (in the new configuration but not the old), and executes every
// client op ordered before the reconfiguration that it had not executed.
func Test_Reconfiguration_Epoch_Increment(t *testing.T) {
	var executed []string
	primary := vsr.Replica{
		Identifier:    0,
		Configuration: vsr.Configuration{0, 1, 2},
		State_Machine: vsr.State_Machine{Execute: func(
			command []byte, prediction []byte,
		) (result []byte) {
			executed = append(executed, string(command))
			return command
		}},
	}
	// A client op precedes the reconfiguration; it must execute when the reconfiguration
	// commits.
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Client: 7, Request_Number: 1,
		Command: []byte("work"),
	})
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Reconfiguration, To: 0, Client: 99, Request_Number: 1,
		Epoch: 0, New_Configuration: vsr.Configuration{0, 1, 2, 3, 4},
	})
	// Ack op 1 (the client op) then op 2 (the reconfiguration); the reconfiguration commit
	// drives the handoff.
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Prepare_Ok, From: 1, To: 0, View: 0, Epoch: 0, Op: 1,
	})
	output := receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Prepare_Ok, From: 1, To: 0, View: 0, Epoch: 0, Op: 2,
	})
	if primary.Epoch != 1 {
		t.Fatalf("expected epoch incremented to 1, got %d", primary.Epoch)
	}
	if len(executed) != 1 {
		t.Fatalf("expected the prior client op executed, got %v", executed)
	}
	if executed[0] != "work" {
		t.Fatalf("expected the prior client op executed, got %v", executed)
	}
	// A Commit to the other old replicas (1 and 2) at the old epoch so they self-advance.
	commits := messages_of_kind(output.Messages, vsr.Message_Kind_Commit)
	if _, ok := commits[1]; !ok {
		t.Errorf("expected a Commit to old replica 1, got %v", commits)
	}
	if _, ok := commits[2]; !ok {
		t.Errorf("expected a Commit to old replica 2, got %v", commits)
	}
	// A Start_Epoch to each added node (3 and 4), carrying both configurations and the start
	// op.
	starts := messages_of_kind(output.Messages, vsr.Message_Kind_Start_Epoch)
	if len(starts) != 2 {
		t.Fatalf("expected a Start_Epoch to added nodes 3 and 4, got %v", starts)
	}
	added := starts[3]
	if added.Epoch != 1 {
		t.Errorf("expected the Start_Epoch to carry epoch 1, got %d", added.Epoch)
	}
	if added.Op != 2 {
		t.Errorf("expected the Start_Epoch to carry the start op 2, got %d", added.Op)
	}
	if len(added.New_Configuration) != 5 {
		t.Errorf("expected the Start_Epoch to carry the new configuration, got %+v", added)
	}
	if len(added.Old_Configuration) != 3 {
		t.Errorf("expected the Start_Epoch to carry the old configuration, got %+v", added)
	}
}

// Test_Reconfiguration_Start_Epoch: a brand-new replica that receives a Start_Epoch records the old
// and new configurations, the new epoch and start op, resets to view 0, enters Transitioning, and —
// its log being empty and short of the epoch start — sends Get_State to catch up from the old
// group.
func Test_Reconfiguration_Start_Epoch(t *testing.T) {
	added := vsr.Replica{Identifier: 3, Configuration: vsr.Configuration{0, 1, 2, 3, 4}}
	output := receive(&added, vsr.Message{
		Kind: vsr.Message_Kind_Start_Epoch, From: 0, To: 3, View: 0, Epoch: 1, Op: 2,
		Old_Configuration: vsr.Configuration{0, 1, 2},
		New_Configuration: vsr.Configuration{0, 1, 2, 3, 4},
	})
	if added.Epoch != 1 {
		t.Fatalf("expected epoch 1 adopted, got %d", added.Epoch)
	}
	if added.Status != vsr.Status_Transition {
		t.Fatalf("expected Status_Transitioning, got %d", added.Status)
	}
	if added.View != 0 {
		t.Fatalf("expected view reset to 0, got %d", added.View)
	}
	if added.Epoch_Start_Op != 2 {
		t.Fatalf("expected the epoch-start op 2 recorded, got %d", added.Epoch_Start_Op)
	}
	if len(added.Old_Configuration) != 3 {
		t.Fatalf("expected the old configuration recorded, got %+v",
			added.Old_Configuration)
	}
	if len(added.Configuration) != 5 {
		t.Fatalf("expected the new configuration adopted, got %+v", added.Configuration)
	}
	// The new node is short of the epoch start, so it asks the old replicas for state.
	transfer, ok := first_message(output.Messages, vsr.Message_Kind_Get_State)
	if !ok {
		t.Fatalf("expected a Get_State to catch up, got %+v", output.Messages)
	}
	if transfer.Epoch != 1 {
		t.Errorf("expected the Get_State stamped with the new epoch 1, got %d",
			transfer.Epoch)
	}
	sources := map[vsr.Replica_Identifier]bool{}
	for _, message := range output.Messages {
		if message.Kind == vsr.Message_Kind_Get_State {
			sources[message.To] = true
		}
	}
	if !sources[0] {
		t.Errorf("expected a Get_State to an old replica, got %v", sources)
	}
}

// Test_Reconfiguration_New_Group_Catch_Up: a brand-new replica with an empty log catches up to the
// start of the epoch through state transfer — restoring a CHECKPOINT rather than replaying from
// zero — and only then returns to Normal in the new epoch.
func Test_Reconfiguration_New_Group_Catch_Up(t *testing.T) {
	var restored []byte
	var executed []string
	added := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 3, Configuration: vsr.Configuration{0, 1, 2, 3, 4},
		Heartbeat: 10, Timeout: 100, Now: 0,
		Checkpoint_Interval: 4, Log_Retain: 2,
		State_Machine: vsr.State_Machine{
			Execute: func(command []byte, prediction []byte) (result []byte) {
				executed = append(executed, string(command))
				return command
			},
			Snapshot: func() (state []byte) { return []byte("s") },
			Restore:  func(state []byte) { restored = state },
		},
	})
	// Learn the epoch: epoch 1 begins at op 6, old group {0,1,2}, new group {0,1,2,3,4}.
	receive(&added, vsr.Message{
		Kind: vsr.Message_Kind_Start_Epoch, From: 0, To: 3, View: 0, Epoch: 1, Op: 6,
		Old_Configuration: vsr.Configuration{0, 1, 2},
		New_Configuration: vsr.Configuration{0, 1, 2, 3, 4},
	})
	if added.Status != vsr.Status_Transition {
		t.Fatalf("expected Transitioning while catching up, got %d", added.Status)
	}
	// An old replica answers the catch-up with a checkpoint at op 4 and the suffix (ops 5, 6),
	// stamped with the new epoch. The new node, empty, must Restore the checkpoint then replay.
	receive(&added, vsr.Message{
		Kind: vsr.Message_Kind_New_State, From: 0, To: 3, View: 0, Epoch: 1,
		Op: 6, Commit: 6,
		Checkpoint_Op: 4, Checkpoint_State: []byte("snap-at-4"),
		Log: []vsr.Log_Entry{
			{View: 0, Command: []byte("e")}, {View: 0, Command: []byte("f")},
		},
	})
	if added.Status != vsr.Status_Normal {
		t.Fatalf("expected Normal once caught up to the epoch start, got %d", added.Status)
	}
	if string(restored) != "snap-at-4" {
		t.Fatalf("expected the checkpoint restored, not a replay from zero, got %q",
			restored)
	}
	if added.Log_Start != 4 {
		t.Fatalf("expected Log_Start at the checkpoint op 4, got %d", added.Log_Start)
	}
	if added.Op != 6 {
		t.Fatalf("expected op 6 adopted, got %d", added.Op)
	}
	if added.Commit != 6 {
		t.Fatalf("expected commit 6, got %d", added.Commit)
	}
	// Only the post-checkpoint suffix is replayed (ops 5, 6), never the GC'd prefix.
	if len(executed) != 2 {
		t.Fatalf("expected only the suffix replayed, got %v", executed)
	}
}

// Test_Reconfiguration_Epoch_Started: once a new-group member is caught up to the epoch start it
// returns to Normal, sends Epoch_Started to the replaced replicas (in the old group but not the
// new), and answers a duplicate Start_Epoch arriving after completion with another Epoch_Started.
func Test_Reconfiguration_Epoch_Started(t *testing.T) {
	// A swap: old group {0,1,2}, new group {0,1,3}; replica 3 is added and replica 2 replaced.
	added := vsr.Replica{Identifier: 3, Configuration: vsr.Configuration{0, 1, 3}}
	// Learn the epoch at start op 1, then catch up to it in one New_State so it completes.
	receive(&added, vsr.Message{
		Kind: vsr.Message_Kind_Start_Epoch, From: 0, To: 3, View: 0, Epoch: 1, Op: 1,
		Old_Configuration: vsr.Configuration{0, 1, 2},
		New_Configuration: vsr.Configuration{0, 1, 3},
	})
	output := receive(&added, vsr.Message{
		Kind: vsr.Message_Kind_New_State, From: 0, To: 3, View: 0, Epoch: 1,
		Op: 1, Commit: 1,
		Log: []vsr.Log_Entry{{View: 0, Command: []byte("a")}},
	})
	if added.Status != vsr.Status_Normal {
		t.Fatalf("expected Normal once caught up, got %d", added.Status)
	}
	started, ok := first_message(output.Messages, vsr.Message_Kind_Epoch_Started)
	if !ok {
		t.Fatalf("expected an Epoch_Started to the replaced replica, got %+v",
			output.Messages)
	}
	if started.To != 2 {
		t.Errorf("expected the Epoch_Started to the replaced replica 2, got %d", started.To)
	}
	if started.Epoch != 1 {
		t.Errorf("expected the Epoch_Started to carry epoch 1, got %d", started.Epoch)
	}
	if started.From != 3 {
		t.Errorf("expected the Epoch_Started from 3, got %d", started.From)
	}

	// A duplicate Start_Epoch arriving after completion is answered with Epoch_Started.
	dup := receive(&added, vsr.Message{
		Kind: vsr.Message_Kind_Start_Epoch, From: 2, To: 3, View: 0, Epoch: 1, Op: 1,
		Old_Configuration: vsr.Configuration{0, 1, 2},
		New_Configuration: vsr.Configuration{0, 1, 3},
	})
	ack, ok := first_message(dup.Messages, vsr.Message_Kind_Epoch_Started)
	if !ok {
		t.Fatalf("expected a duplicate Start_Epoch answered with Epoch_Started, got %+v",
			dup.Messages)
	}
	if ack.To != 2 {
		t.Errorf("expected the re-acknowledgement to the sender 2, got %d", ack.To)
	}
}

// Test_Reconfiguration_Old_Group_Shutdown: a replica being replaced, on executing the committed
// reconfiguration, enters Transitioning with its old membership in Old_Configuration and the new
// one adopted; it then serves until it holds f'+1 Epoch_Started — the new group's quorum — and
// shuts down.
func Test_Reconfiguration_Old_Group_Shutdown(t *testing.T) {
	// A swap removing replica 2: old group {0,1,2}, new group {0,1,3}, so f'=1 and f'+1=2.
	replaced := vsr.Replica{Identifier: 2, Configuration: vsr.Configuration{0, 1, 2}}
	// The reconfiguration sits at op 1 in its log; a Commit then commits and executes it.
	receive(&replaced, vsr.Message{
		Kind: vsr.Message_Kind_Prepare, From: 0, To: 2, View: 0, Epoch: 0, Op: 1,
		Entries: []vsr.Log_Entry{{
			View: 0, Client: 99, Request_Number: 1,
			New_Configuration: vsr.Configuration{0, 1, 3},
		}},
	})
	receive(&replaced, vsr.Message{
		Kind: vsr.Message_Kind_Commit, From: 0, To: 2, View: 0, Epoch: 0, Commit: 1,
	})
	if replaced.Status != vsr.Status_Transition {
		t.Fatalf("expected Transitioning after executing the reconfiguration, got %d",
			replaced.Status)
	}
	if replaced.Epoch != 1 {
		t.Fatalf("expected epoch advanced to 1, got %d", replaced.Epoch)
	}
	if len(replaced.Old_Configuration) != 3 {
		t.Fatalf("expected the old membership kept in Old_Configuration, got %+v",
			replaced.Old_Configuration)
	}
	if len(replaced.Configuration) != 3 {
		t.Fatalf("expected the new configuration adopted, got %+v", replaced.Configuration)
	}
	if replaced.Configuration[2] != 3 {
		t.Fatalf("expected the new configuration adopted, got %+v", replaced.Configuration)
	}

	// Retire only once EVERY active new-group member ({0,1,3}) is up, not merely a quorum: the
	// ADDED member 3 must confirm too, so the new group is fault-tolerant before the handoff
	// completes. A quorum of just the continuing members {0,1} would leave 3 stranded; a later
	// crash then drops the group below quorum with no driver to bring 3 up, the reconfiguration
	// wedge the simulator surfaced (seeds 8185/20229/22678/9437).
	receive(&replaced, vsr.Message{
		Kind: vsr.Message_Kind_Epoch_Started, From: 0, To: 2, View: 0, Epoch: 1,
	})
	if replaced.Status != vsr.Status_Transition {
		t.Fatalf("expected still serving with one member up, got %d", replaced.Status)
	}
	// Continuing members 0 and 1 are up (a quorum), but added member 3 is not: still serving.
	receive(&replaced, vsr.Message{
		Kind: vsr.Message_Kind_Epoch_Started, From: 1, To: 2, View: 0, Epoch: 1,
	})
	if replaced.Status != vsr.Status_Transition {
		t.Fatalf("expected still serving until member 3 is up, got %d", replaced.Status)
	}
	// The added member 3 confirms: every active new-group member is up, so the handoff is done.
	receive(&replaced, vsr.Message{
		Kind: vsr.Message_Kind_Epoch_Started, From: 3, To: 2, View: 0, Epoch: 1,
	})
	if replaced.Status != vsr.Status_Shutdown {
		t.Fatalf("expected Shutdown once every active new-group member is up, got %d",
			replaced.Status)
	}
}

// Test_Reconfiguration_Epoch_Precedence: epoch dominates view. A message at a LOWER epoch is
// dropped and the sender informed with a New_Epoch, whatever its view; a message at a HIGHER epoch
// is adopted even when its view is lower; only at an EQUAL epoch does the view comparison apply.
// The test sweeps the {<,=,>} epoch by {<,=,>} view matrix.
func Test_Reconfiguration_Epoch_Precedence(t *testing.T) {
	// A LOWER epoch (0 < 1) is rejected at every view, with a New_Epoch back to the sender.
	for _, view := range []vsr.View{1, 2, 3} { // Below, equal, above the replica's view 2.
		replica := epoch_1_view_2_replica()
		output := receive(&replica, vsr.Message{
			Kind: vsr.Message_Kind_Prepare, From: 0, To: 1, Epoch: 0, View: view,
			Op: 1, Entries: []vsr.Log_Entry{{Command: []byte("x")}},
		})
		if replica.Op != 0 {
			t.Fatalf("lower epoch at view %d should be dropped, op %d",
				view, replica.Op)
		}
		redirect, ok := first_message(output.Messages, vsr.Message_Kind_New_Epoch)
		if !ok {
			t.Fatalf("lower epoch at view %d should send New_Epoch, got %+v",
				view, output.Messages)
		}
		if redirect.To != 0 {
			t.Errorf("expected the New_Epoch back to the sender 0, got %d", redirect.To)
		}
		if len(redirect.New_Configuration) == 0 {
			t.Errorf("expected the New_Epoch to carry the configuration, got %+v",
				redirect)
		}
	}
	// A HIGHER epoch (2 > 1) is adopted even at a LOWER view (0 < 2): the Start_Epoch carrier
	// moves the replica to epoch 2 and view 0, proving epoch dominates view.
	higher := epoch_1_view_2_replica()
	receive(&higher, vsr.Message{
		Kind: vsr.Message_Kind_Start_Epoch, From: 0, To: 1, Epoch: 2, View: 0, Op: 0,
		Old_Configuration: vsr.Configuration{0, 1, 2},
		New_Configuration: vsr.Configuration{0, 1, 2, 3, 4},
	})
	if higher.Epoch != 2 {
		t.Fatalf("higher epoch at a lower view should be adopted, epoch %d", higher.Epoch)
	}
	if higher.View != 0 {
		t.Fatalf("expected the new epoch to start in view 0, got %d", higher.View)
	}
	// At an EQUAL epoch the view comparison applies: a lower view is dropped, an equal view is
	// processed normally.
	equal_low := epoch_1_view_2_replica()
	receive(&equal_low, vsr.Message{
		Kind: vsr.Message_Kind_Prepare, From: 0, To: 1, Epoch: 1, View: 1,
		Op: 1, Entries: []vsr.Log_Entry{{Command: []byte("x")}},
	})
	if equal_low.Op != 0 {
		t.Fatalf("equal epoch, lower view should be dropped, op %d", equal_low.Op)
	}
	equal_same := epoch_1_view_2_replica()
	output := receive(&equal_same, vsr.Message{
		Kind: vsr.Message_Kind_Prepare, From: 0, To: 1, Epoch: 1, View: 2,
		Op: 1, Entries: []vsr.Log_Entry{{Command: []byte("x")}},
	})
	if equal_same.Op != 1 {
		t.Fatalf("equal epoch and view should be processed, op %d", equal_same.Op)
	}
	if _, ok := first_message(output.Messages, vsr.Message_Kind_Prepare_Ok); !ok {
		t.Errorf("equal epoch and view should ack, got %+v", output.Messages)
	}
}

// Test_Reconfiguration_View_Change_Across_Epoch: when a view change installs a log whose topmost
// entry is a COMMITTED reconfiguration, the new primary re-drives the epoch handoff — it sends
// Start_Epoch to the added nodes — and stops accepting client requests, the reconfiguration being
// the epoch's last op (§7.2, §8.3).
func Test_Reconfiguration_View_Change_Across_Epoch(t *testing.T) {
	// Replica 1 becomes primary of view 1 on a 3-node group; the reconfiguration grows to 5.
	primary := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 1, Configuration: vsr.Configuration{0, 1, 2},
		Heartbeat: 10, Timeout: 100, Now: 0,
	})
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Start_View_Change, From: 0, To: 1, View: 1, Epoch: 0,
	})
	// The winner reports a committed reconfiguration at op 1 (its topmost), growing to
	// {0,1,2,3,4}.
	output := receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Do_View_Change, From: 0, To: 1, View: 1, Epoch: 0,
		Last_Normal_View: 0, Op: 1, Commit: 1,
		Log_Suffix: []vsr.Log_Entry{{
			View: 0, Client: 99, Request_Number: 1,
			New_Configuration: vsr.Configuration{0, 1, 2, 3, 4},
		}},
	})
	// Installing the committed reconfiguration advances the epoch and re-drives the handoff.
	if primary.Epoch != 1 {
		t.Fatalf("expected the committed reconfiguration to advance the epoch, got %d",
			primary.Epoch)
	}
	starts := map[vsr.Replica_Identifier]bool{}
	for _, message := range output.Messages {
		if message.Kind == vsr.Message_Kind_Start_Epoch {
			starts[message.To] = true
		}
	}
	if !starts[3] {
		t.Fatalf("expected Start_Epoch to added node 3, got %v", starts)
	}
	if !starts[4] {
		t.Fatalf("expected Start_Epoch to added node 4, got %v", starts)
	}
}

// Test_Reconfiguration_Client_Redirect: an old replica that receives a client request stamped with
// a stale epoch replies New_Epoch carrying the current view and the new configuration, so the
// client can find the group that moved on (§7.4). The reply is addressed to the client, not the
// sender.
func Test_Reconfiguration_Client_Redirect(t *testing.T) {
	// A replica that has advanced to epoch 2, now serving the new configuration {0,1,3}.
	replica := vsr.Replica{
		Identifier: 0, Configuration: vsr.Configuration{0, 1, 3},
		Epoch: 2, View: 1, Status: vsr.Status_Normal,
	}
	output := receive(&replica, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Epoch: 0, Client: 77, Request_Number: 5,
		Command: []byte("stale"),
	})
	if replica.Op != 0 {
		t.Fatalf("expected the stale-epoch request not appended, op %d", replica.Op)
	}
	redirect, ok := first_message(output.Replies, vsr.Message_Kind_New_Epoch)
	if !ok {
		t.Fatalf("expected a New_Epoch redirect to the client, got %+v", output.Replies)
	}
	if redirect.Client != 77 {
		t.Errorf("expected the redirect addressed to client 77, got %d", redirect.Client)
	}
	if redirect.View != 1 {
		t.Errorf("expected the redirect to carry the current view 1, got %d", redirect.View)
	}
	if len(redirect.New_Configuration) != 3 {
		t.Errorf("expected the redirect to carry the new configuration, got %+v", redirect)
	}
	if redirect.New_Configuration[2] != 3 {
		t.Errorf("expected the redirect to carry the new configuration, got %+v", redirect)
	}
}

// Test_Batching_Flush_Immediate: an idle primary — its commit number caught up to its op — accepts
// a request and broadcasts a Prepare carrying that single entry at once, so batching costs no
// latency when the primary is not loaded.
func Test_Batching_Flush_Immediate(t *testing.T) {
	primary := vsr.Replica{
		Identifier: 0, Configuration: vsr.Configuration{0, 1, 2}, Batch_Max: 4,
	}
	output := receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Client: 7, Request_Number: 1,
		Command: []byte("a"),
	})
	if primary.Op != 1 {
		t.Fatalf("expected the idle primary to append at once, got op %d", primary.Op)
	}
	prepare, ok := first_message(output.Messages, vsr.Message_Kind_Prepare)
	if !ok {
		t.Fatalf("expected an immediate Prepare, got %+v", output.Messages)
	}
	if len(prepare.Entries) != 1 {
		t.Fatalf("expected an immediate batch of one, got %d entries", len(prepare.Entries))
	}
	if prepare.Op != 1 {
		t.Errorf("expected the Prepare at op 1, got %d", prepare.Op)
	}
	if string(prepare.Entries[0].Command) != "a" {
		t.Errorf("expected the entry command a, got %q", prepare.Entries[0].Command)
	}
}

// Test_Batching_Flush_Batched: a request that arrives while a round is in flight is buffered, not
// sent; when the in-flight round commits, the primary appends the buffered requests as consecutive
// ops and broadcasts them in one Prepare (§6.2).
func Test_Batching_Flush_Batched(t *testing.T) {
	primary := vsr.Replica{
		Identifier: 0, Configuration: vsr.Configuration{0, 1, 2}, Batch_Max: 4,
	}
	// Request from client 7: the primary is idle, so it flushes op 1 immediately.
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Client: 7, Request_Number: 1,
		Command: []byte("a"),
	})
	// Requests from clients 8 and 9 arrive while op 1 is in flight: buffered, no Prepare
	// emitted.
	out2 := receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Client: 8, Request_Number: 1,
		Command: []byte("b"),
	})
	if _, sent := first_message(out2.Messages, vsr.Message_Kind_Prepare); sent {
		t.Fatalf("expected the request buffered while busy, got a Prepare")
	}
	out3 := receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Client: 9, Request_Number: 1,
		Command: []byte("c"),
	})
	if _, sent := first_message(out3.Messages, vsr.Message_Kind_Prepare); sent {
		t.Fatalf("expected the request buffered while busy, got a Prepare")
	}
	if primary.Op != 1 {
		t.Fatalf("expected buffered requests not yet appended, got op %d", primary.Op)
	}
	// Op 1 commits on a quorum; the buffered batch flushes as one Prepare for ops 2..3.
	output := receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Prepare_Ok, From: 1, To: 0, View: 0, Op: 1,
	})
	prepare, ok := first_message(output.Messages, vsr.Message_Kind_Prepare)
	if !ok {
		t.Fatalf("expected the buffered batch flushed on commit, got %+v", output.Messages)
	}
	if len(prepare.Entries) != 2 {
		t.Fatalf("expected a batch of two, got %d entries", len(prepare.Entries))
	}
	if prepare.Op != 3 {
		t.Errorf("expected the batch's highest op 3, got %d", prepare.Op)
	}
	if primary.Op != 3 {
		t.Errorf("expected both buffered ops appended, got op %d", primary.Op)
	}
	if string(prepare.Entries[0].Command) != "b" {
		t.Errorf(
			"expected op 2 entry b first in the batch, got %q",
			prepare.Entries[0].Command)
	}
	if string(prepare.Entries[1].Command) != "c" {
		t.Errorf(
			"expected op 3 entry c last in the batch, got %q",
			prepare.Entries[1].Command)
	}
}

// Test_Batching_Batch_Cap: a busy primary flushes once the buffer reaches Batch_Max even before the
// in-flight round commits, bounding a single Prepare's size.
func Test_Batching_Batch_Cap(t *testing.T) {
	primary := vsr.Replica{
		Identifier: 0, Configuration: vsr.Configuration{0, 1, 2}, Batch_Max: 2,
	}
	// Op 1 flushes immediately (idle).
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Client: 7, Request_Number: 1,
		Command: []byte("a"),
	})
	// One request buffered: below the cap of 2, no flush.
	out2 := receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Client: 8, Request_Number: 1,
		Command: []byte("b"),
	})
	if _, sent := first_message(out2.Messages, vsr.Message_Kind_Prepare); sent {
		t.Fatalf("expected no flush below the cap, got a Prepare")
	}
	// The second buffered request reaches the cap of 2, flushing the batch though op 1 is
	// unacked.
	out3 := receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Client: 9, Request_Number: 1,
		Command: []byte("c"),
	})
	prepare, ok := first_message(out3.Messages, vsr.Message_Kind_Prepare)
	if !ok {
		t.Fatalf("expected a flush at the cap, got %+v", out3.Messages)
	}
	if len(prepare.Entries) != 2 {
		t.Fatalf("expected the batch capped at 2, got %d entries", len(prepare.Entries))
	}
	if prepare.Op != 3 {
		t.Errorf("expected the capped batch's highest op 3, got %d", prepare.Op)
	}
	if primary.Commit != 0 {
		t.Errorf(
			"expected nothing committed while flushing at the cap, got %d",
			primary.Commit)
	}
}

// Test_Batching_Batch_Prepare_Ok: a backup applies every entry of a batched Prepare and returns one
// Prepare_Ok per newly-appended op, so the primary tallies an explicit acknowledgement for each; a
// primary commits the whole batch once a quorum has acknowledged every op in it.
func Test_Batching_Batch_Prepare_Ok(t *testing.T) {
	backup := vsr.Replica{Identifier: 1, Configuration: vsr.Configuration{0, 1, 2}}
	output := receive(&backup, vsr.Message{
		Kind: vsr.Message_Kind_Prepare, From: 0, To: 1, View: 0, Op: 3,
		Entries: []vsr.Log_Entry{
			{View: 0, Command: []byte("a")},
			{View: 0, Command: []byte("b")},
			{View: 0, Command: []byte("c")},
		},
	})
	if backup.Op != 3 {
		t.Fatalf("expected all three batch entries appended, got op %d", backup.Op)
	}
	if len(backup.Log) != 3 {
		t.Fatalf("expected a three-entry log, got %d", len(backup.Log))
	}
	acked := map[vsr.Op]bool{}
	for _, message := range output.Messages {
		if message.Kind == vsr.Message_Kind_Prepare_Ok {
			acked[message.Op] = true
		}
	}
	if !acked[1] {
		t.Fatalf("expected one Prepare_Ok per batch op (1,2,3), got acks for %v", acked)
	}
	if !acked[2] {
		t.Fatalf("expected one Prepare_Ok per batch op (1,2,3), got acks for %v", acked)
	}
	if !acked[3] {
		t.Fatalf("expected one Prepare_Ok per batch op (1,2,3), got acks for %v", acked)
	}
	if len(acked) != 3 {
		t.Fatalf("expected one Prepare_Ok per batch op (1,2,3), got acks for %v", acked)
	}

	// A primary holding a three-op batch commits all of it once a backup acknowledges every op.
	primary := vsr.Replica{
		Identifier: 0, Configuration: vsr.Configuration{0, 1, 2}, Op: 3,
		Log: []vsr.Log_Entry{
			{View: 0, Command: []byte("a")},
			{View: 0, Command: []byte("b")},
			{View: 0, Command: []byte("c")},
		},
	}
	for op := vsr.Op(1); op <= 3; op++ {
		receive(&primary, vsr.Message{
			Kind: vsr.Message_Kind_Prepare_Ok, From: 1, To: 0, View: 0, Op: op,
		})
	}
	if primary.Commit != 3 {
		t.Fatalf(
			"expected the batch committed once every op is acknowledged, got commit %d",
			primary.Commit)
	}
}

// Test_Batching_Batch_Reconfiguration_Singleton: a Reconfiguration first flushes any buffered
// client requests, so they are ordered ahead of it, then is appended alone as the epoch's last op
// — never inside a batch — after which the primary accepts no further client requests.
func Test_Batching_Batch_Reconfiguration_Singleton(t *testing.T) {
	primary := vsr.Replica{
		Identifier: 0, Configuration: vsr.Configuration{0, 1, 2}, Batch_Max: 8,
	}
	// Op 1 flushes immediately (idle); a second request buffers while op 1 is in flight.
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Client: 7, Request_Number: 1,
		Command: []byte("a"),
	})
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Client: 8, Request_Number: 1,
		Command: []byte("b"),
	})
	if primary.Op != 1 {
		t.Fatalf("expected the second request buffered, got op %d", primary.Op)
	}
	// The reconfiguration flushes the buffered client request as op 2, then appends alone as op
	// 3.
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Reconfiguration, To: 0, Client: 99, Request_Number: 1,
		New_Configuration: vsr.Configuration{0, 1, 2, 3, 4},
	})
	if primary.Op != 3 {
		t.Fatalf(
			"expected op 2 (flushed client) then op 3 (reconfiguration), got op %d",
			primary.Op)
	}
	last := primary.Log[len(primary.Log)-1]
	if len(last.New_Configuration) != 5 {
		t.Fatalf("expected the reconfiguration as the last op, got %+v", last)
	}
	ahead := primary.Log[len(primary.Log)-2]
	if len(ahead.New_Configuration) != 0 {
		t.Fatalf(
			"expected the client request ordered ahead of the reconfiguration, got %+v",
			ahead)
	}
	// No further client requests are accepted once the reconfiguration is the epoch's last op.
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Client: 8, Request_Number: 2,
		Command: []byte("c"),
	})
	if primary.Op != 3 {
		t.Fatalf(
			"expected client requests refused after the reconfiguration, got op %d",
			primary.Op)
	}
}

// Test_Standby_Roles: the leading Active_Count members are active and execute committed ops; the
// members after them are standbys and do not. With Active_Count 2 on {0,1,2}, replica 1 executes a
// committed op and replica 2 does not.
func Test_Standby_Roles(t *testing.T) {
	executed := map[vsr.Replica_Identifier]bool{}
	for _, identifier := range []vsr.Replica_Identifier{1, 2} {
		mark := identifier
		replica := vsr.Replica{
			Identifier:    identifier,
			Configuration: vsr.Configuration{0, 1, 2},
			Active_Count:  2,
			State_Machine: vsr.State_Machine{
				Execute: func(cmd, pred []byte) (result []byte) {
					executed[mark] = true
					return cmd
				},
			},
		}
		receive(&replica, vsr.Message{
			Kind: vsr.Message_Kind_Prepare, From: 0, To: identifier, View: 0, Op: 1,
			Entries: []vsr.Log_Entry{{Command: []byte("x")}},
		})
		receive(&replica, vsr.Message{
			Kind: vsr.Message_Kind_Commit, From: 0, To: identifier, View: 0, Commit: 1,
		})
	}
	if !executed[1] {
		t.Errorf("expected active replica 1 to execute the committed op")
	}
	if executed[2] {
		t.Errorf("expected standby replica 2 not to execute")
	}
}

// Test_Standby_Primary_Always_Active: the primary is Configuration[View mod Active_Count], so a
// standby is never primary. Replica 2 (a standby, Active_Count 2) refuses a client Request even at
// view 2, where without standbys it would be the primary; active replica 0 is the primary there.
func Test_Standby_Primary_Always_Active(t *testing.T) {
	standby := vsr.Replica{
		Identifier: 2, Configuration: vsr.Configuration{0, 1, 2}, Active_Count: 2, View: 2,
		Status: vsr.Status_Normal,
	}
	receive(&standby, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 2, View: 2, Client: 7, Request_Number: 1,
		Command: []byte("x"),
	})
	if standby.Op != 0 {
		t.Fatalf("expected the standby to refuse the request (never primary), op %d",
			standby.Op)
	}
	primary := vsr.Replica{
		Identifier: 0, Configuration: vsr.Configuration{0, 1, 2}, Active_Count: 2, View: 2,
		Status: vsr.Status_Normal,
	}
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, View: 2, Client: 7, Request_Number: 1,
		Command: []byte("x"),
	})
	if primary.Op != 1 {
		t.Fatalf("expected active replica 0 to be the primary of view 2, op %d", primary.Op)
	}
}

// Test_Standby_No_Execution: a standby advances its commit number and retains the committed log,
// but makes no Execute up-call, emits no Reply, and takes no checkpoint of its own.
func Test_Standby_No_Execution(t *testing.T) {
	executed := false
	snapshotted := false
	standby := vsr.Replica{
		Identifier: 2, Configuration: vsr.Configuration{0, 1, 2}, Active_Count: 2,
		Checkpoint_Interval: 1,
		State_Machine: vsr.State_Machine{
			Execute: func(cmd, pred []byte) (result []byte) {
				executed = true
				return cmd
			},
			Snapshot: func() (snapshot []byte) { snapshotted = true; return nil },
		},
	}
	receive(&standby, vsr.Message{
		Kind: vsr.Message_Kind_Prepare, From: 0, To: 2, View: 0, Op: 1,
		Entries: []vsr.Log_Entry{{Command: []byte("x")}},
	})
	output := receive(&standby, vsr.Message{
		Kind: vsr.Message_Kind_Commit, From: 0, To: 2, View: 0, Commit: 1,
	})
	if standby.Commit != 1 {
		t.Fatalf("expected the standby to advance its commit number, got %d",
			standby.Commit)
	}
	if len(standby.Log) != 1 {
		t.Fatalf("expected the standby to retain the committed log, got %d",
			len(standby.Log))
	}
	if executed {
		t.Errorf("expected no Execute up-call on a standby")
	}
	if snapshotted {
		t.Errorf("expected no checkpoint taken on a standby")
	}
	if len(output.Replies) != 0 {
		t.Errorf("expected no Reply from a standby, got %+v", output.Replies)
	}
}

// Test_Standby_No_Vote: a standby that receives a Prepare appends the entry and returns a
// Prepare_Ok, so it is never counted toward a commit quorum (TigerBeetle's non-voting-standby
// design).
func Test_Standby_No_Vote(t *testing.T) {
	standby := vsr.Replica{
		Identifier: 2, Configuration: vsr.Configuration{0, 1, 2}, Active_Count: 2,
	}
	output := receive(&standby, vsr.Message{
		Kind: vsr.Message_Kind_Prepare, From: 0, To: 2, View: 0, Op: 1,
		Entries: []vsr.Log_Entry{{Command: []byte("x")}},
	})
	if standby.Op != 1 {
		t.Fatalf("expected the standby to append the entry to stay current, op %d",
			standby.Op)
	}
	if _, voted := first_message(output.Messages, vsr.Message_Kind_Prepare_Ok); voted {
		t.Errorf("expected a standby to cast no Prepare_Ok, got %+v", output.Messages)
	}
}

// Test_Standby_Follow: a standby applies a Prepare and Commit to track the cluster, but on a
// timeout it never starts a view change — it is not responsible for liveness and casts no
// view-change vote (TigerBeetle's non-voting-standby design); the voting backups alone fail a
// primary over.
func Test_Standby_Follow(t *testing.T) {
	standby := vsr.New_Replica(&vsr.New_Replica_Input{
		Identifier: 2, Configuration: vsr.Configuration{0, 1, 2}, Active_Count: 2,
		Heartbeat: 10, Timeout: 100, Now: 0,
	})
	receive(&standby, vsr.Message{
		Kind: vsr.Message_Kind_Prepare, From: 0, To: 2, View: 0, Op: 1,
		Entries: []vsr.Log_Entry{{Command: []byte("x")}},
	})
	receive(&standby, vsr.Message{
		Kind: vsr.Message_Kind_Commit, From: 0, To: 2, View: 0, Commit: 1,
	})
	if standby.Commit != 1 {
		t.Fatalf("expected the standby to track the commit number, got %d", standby.Commit)
	}
	// The primary falls silent; the standby's failure-detector deadline passes. It must NOT
	// start a view change.
	output := tick(&standby, 100)
	if standby.Status != vsr.Status_Normal {
		t.Fatalf("expected the standby to stay normal, not enter a view change, status %d",
			standby.Status)
	}
	_, started := first_message(output.Messages, vsr.Message_Kind_Start_View_Change)
	if started {
		t.Errorf("expected a standby to start no view change on timeout, got %+v",
			output.Messages)
	}
}

// Test_Standby_Promotion: a standby a reconfiguration moves into the voting set materializes the
// application state it never built — it replays its committed log (TigerBeetle's state sync) and
// serves as an active replica. Replica 3 follows as a standby in {0,1,2,3,4} (active {0,1,2}),
// holding two committed ops it never executed; a Start_Epoch into {0,1,3} makes it active, and it
// executes both as it completes the epoch.
func Test_Standby_Promotion(t *testing.T) {
	executed := []string{}
	standby := vsr.Replica{
		Identifier:    3,
		Configuration: vsr.Configuration{0, 1, 2, 3, 4},
		Active_Count:  3,
		Status:        vsr.Status_Normal,
		Op:            2,
		Commit:        2,
		// Followed as a standby: commit advanced, but nothing was executed.
		Executed: 2,
		Log: []vsr.Log_Entry{
			{View: 0, Command: []byte("a"), Client: 7, Request_Number: 1},
			{View: 0, Command: []byte("b"), Client: 8, Request_Number: 1},
		},
		State_Machine: vsr.State_Machine{
			Execute: func(command, prediction []byte) (result []byte) {
				executed = append(executed, string(command))
				return command
			},
		},
	}
	receive(&standby, vsr.Message{
		Kind: vsr.Message_Kind_Start_Epoch, From: 0, To: 3, Epoch: 1, View: 0, Op: 2,
		Old_Configuration: vsr.Configuration{0, 1, 2, 3, 4},
		New_Configuration: vsr.Configuration{0, 1, 3},
		New_Active_Count:  3,
	})
	if standby.Status != vsr.Status_Normal {
		t.Fatalf("expected the promoted standby to complete the epoch as normal, status %d",
			standby.Status)
	}
	if len(executed) != 2 {
		t.Fatalf("expected the promotion to replay both committed ops, executed %v",
			executed)
	}
	if executed[0] != "a" {
		t.Fatalf("expected the promotion to replay both committed ops, executed %v",
			executed)
	}
	if executed[1] != "b" {
		t.Fatalf("expected the promotion to replay both committed ops, executed %v",
			executed)
	}
}

// Test_Read_Through_Consensus: a read-only request is an ordinary command — the primary appends it,
// commits it on a quorum, the state machine executes it, and the primary replies, exactly as for a
// write. The protocol does not distinguish a read; reads run through consensus, TigerBeetle's
// design (it has no lease-based fast-read path).
func Test_Read_Through_Consensus(t *testing.T) {
	primary := vsr.Replica{
		Identifier: 0, Configuration: vsr.Configuration{0, 1, 2},
		State_Machine: vsr.State_Machine{
			Execute: func(command, prediction []byte) (result []byte) {
				return []byte("answered-" + string(command))
			},
		},
	}
	// A read-only command arrives like any request and is appended to the log.
	receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Request, To: 0, Client: 7, Request_Number: 1,
		Command: []byte("read-x"),
	})
	if primary.Op != 1 {
		t.Fatalf("expected the read appended through the log, op %d", primary.Op)
	}
	// A quorum ack commits it; the state machine executes it and the primary replies.
	output := receive(&primary, vsr.Message{
		Kind: vsr.Message_Kind_Prepare_Ok, From: 1, To: 0, View: 0, Op: 1,
	})
	if primary.Commit != 1 {
		t.Fatalf("expected the read committed on a quorum, commit %d", primary.Commit)
	}
	reply, ok := first_message(output.Replies, vsr.Message_Kind_Reply)
	if !ok {
		t.Fatalf("expected a Reply for the committed read, got %+v", output.Replies)
	}
	if string(reply.Result) != "answered-read-x" {
		t.Errorf("expected the state machine's read result, got %q", reply.Result)
	}
}
