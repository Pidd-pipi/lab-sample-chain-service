package main

import (
	"testing"
	"time"
)

func TestStateHistoryInvalidNotRecorded(t *testing.T) {
	machine := newOpsStateMachine()
	err := machine.Move(OpsStatusQueued, OpsStatusPaused, "attempt")
	if err == nil {
		t.Fatal("expected invalid transition error")
	}
	if got := len(machine.History()); got != 0 {
		t.Fatalf("invalid transition recorded %d history entries, want 0", got)
	}
}

func TestStateHistoryNoopNotRecorded(t *testing.T) {
	machine := newOpsStateMachine()
	if err := machine.Move(OpsStatusActive, OpsStatusActive, "noop"); err != nil {
		t.Fatal(err)
	}
	if got := len(machine.History()); got != 0 {
		t.Fatalf("no-op transition recorded %d history entries, want 0", got)
	}
}

func TestStateHistoryRecordsValidMoves(t *testing.T) {
	machine := newOpsStateMachine()
	if err := machine.Move(OpsStatusQueued, OpsStatusActive, "start"); err != nil {
		t.Fatal(err)
	}
	history := machine.History()
	if len(history) != 1 {
		t.Fatalf("valid transition recorded %d entries, want exactly 1", len(history))
	}
	if history[0].From != OpsStatusQueued || history[0].To != OpsStatusActive {
		t.Fatalf("unexpected history entry %+v", history[0])
	}
}

func TestStateHistorySnapshotIsolation(t *testing.T) {
	machine := newOpsStateMachine()
	_ = machine.Move(OpsStatusQueued, OpsStatusActive, "start")
	_ = machine.Move(OpsStatusActive, OpsStatusPaused, "pause")
	snapshot := machine.History()
	machine.Reset()
	_ = machine.Move(OpsStatusQueued, OpsStatusClosed, "close")
	if len(snapshot) != 2 || snapshot[0].To != OpsStatusActive || snapshot[1].To != OpsStatusPaused {
		t.Fatalf("history snapshot was mutated by later operations: %+v", snapshot)
	}
}

func TestStateMachineResetEmpties(t *testing.T) {
	machine := newOpsStateMachine()
	_ = machine.Move(OpsStatusQueued, OpsStatusActive, "start")
	machine.Reset()
	if got := len(machine.History()); got != 0 {
		t.Fatalf("history after Reset = %d, want 0", got)
	}
}

func TestAuditSinceNanoPrecision(t *testing.T) {
	audit := &OpsAudit{events: []OpsEvent{
		{ID: "evt-1", RecordID: "ops-001", Type: "created", Actor: "li", At: "2026-08-24T08:00:00.123456789Z"},
		{ID: "evt-2", RecordID: "ops-002", Type: "created", Actor: "wang", At: "2026-08-24T09:00:00Z"},
	}}
	start, err := time.Parse(time.RFC3339Nano, "2026-08-24T07:59:00Z")
	if err != nil {
		t.Fatal(err)
	}
	events := audit.Since(start)
	if len(events) != 2 {
		t.Fatalf("Since(start) returned %d events, want 2 (nanosecond event must be included)", len(events))
	}
}

func TestAuditSinceExcludesPast(t *testing.T) {
	audit := &OpsAudit{events: []OpsEvent{
		{ID: "evt-1", RecordID: "ops-001", Type: "created", Actor: "li", At: "2026-08-24T07:00:00Z"},
		{ID: "evt-2", RecordID: "ops-002", Type: "created", Actor: "wang", At: "2026-08-24T09:00:00Z"},
	}}
	start, err := time.Parse(time.RFC3339Nano, "2026-08-24T08:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	events := audit.Since(start)
	if len(events) != 1 {
		t.Fatalf("Since(start) returned %d events, want 1 (events before start excluded)", len(events))
	}
	if events[0].RecordID != "ops-002" {
		t.Fatalf("unexpected event %s", events[0].RecordID)
	}
}

func TestAuditClearEmpties(t *testing.T) {
	audit := newOpsAudit()
	audit.Add("ops-001", "created", "li")
	audit.Add("ops-002", "created", "wang")
	audit.Clear()
	if got := audit.Count(); got != 0 {
		t.Fatalf("audit count after Clear = %d, want 0", got)
	}
}
