package main

import (
	"context"
	"testing"
	"time"
)

func TestOpsContextParentDeadline(t *testing.T) {
	parentDeadline := time.Now().Add(time.Second)
	parent, cancel := context.WithDeadline(context.Background(), parentDeadline)
	defer cancel()
	ctx, cancel2 := opsContext(parent, 10*time.Second)
	defer cancel2()
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("derived context has no deadline")
	}
	if deadline.After(parentDeadline) {
		t.Fatalf("derived deadline %v is later than parent deadline %v", deadline, parentDeadline)
	}
}

func TestOpsDelayRespectsCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)
	start := time.Now()
	err := opsDelay(ctx, 2*time.Second)
	elapsed := time.Since(start)
	if elapsed > time.Second {
		t.Fatalf("opsDelay ignored cancellation, took %v", elapsed)
	}
	if err != context.Canceled {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestOpsTransitionCancelledNotApplied(t *testing.T) {
	service := newOpsService(seedOpsRecords())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	record, err := service.Transition(ctx, "ops-001", 1, OpsStatusActive, "li")
	if err == nil {
		t.Fatalf("transition succeeded after cancel, record status=%s", record.Status)
	}
	got, _ := service.Get(context.Background(), "ops-001")
	if got.Status != OpsStatusQueued {
		t.Fatalf("record status changed to %s after canceled transition", got.Status)
	}
}

func TestOpsAddCancelledNotStored(t *testing.T) {
	service := newOpsService(seedOpsRecords())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	record := OpsRecord{
		ID:       "ops-new",
		Subject:  "reagent check",
		Owner:    "li",
		Status:   OpsStatusQueued,
		Priority: OpsPriorityHigh,
		Labels:   map[string]string{"site": "lab-1"},
	}
	_, err := service.Create(ctx, record)
	if err == nil {
		t.Fatal("create succeeded after cancel")
	}
}

func newTestApp() (*sampleApp, *SampleStore, *SampleAudit) {
	store := newSampleStore()
	audit := newSampleAudit()
	app := &sampleApp{
		store:  store,
		audit:  audit,
		batch:  newSampleBatcher(store, audit),
		stats:  newSampleStats(store),
		review: newReviewStore(),
		export: newSampleExporter(store),
		ops:    newOpsService(seedOpsRecords()),
	}
	return app, store, audit
}
