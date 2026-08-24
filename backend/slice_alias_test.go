package main

import (
	"context"
	"sync"
	"testing"
)

func TestOpsSearchAllPagesComplete(t *testing.T) {
	app, _, _ := newTestApp()
	page := 1
	for {
		result, err := app.ops.Search(context.Background(), OpsQuery{Page: page, PageSize: 15})
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Items) == 0 {
			break
		}
		if page > 10 {
			t.Fatal("search did not terminate")
		}
		page++
	}
	// 超出范围的页码应返回空页而不是越界
	outOfRange, err := app.ops.Search(context.Background(), OpsQuery{Page: 99, PageSize: 15})
	if err != nil {
		t.Fatal(err)
	}
	if len(outOfRange.Items) != 0 {
		t.Fatalf("out-of-range page returned %d items, want 0", len(outOfRange.Items))
	}
}

func TestTopMaterialsSnapshot(t *testing.T) {
	app, _, _ := newTestApp()
	first := app.stats.TopMaterials(5)
	capture := make([]MaterialCount, len(first))
	copy(capture, first)
	app.store.samples["S-1099"] = Sample{ID: "S-1099", Material: "urine", Batch: "B-99", Status: "received"}
	_ = app.stats.TopMaterials(5)
	for i := range capture {
		if capture[i] != first[i] {
			t.Fatalf("first TopMaterials result mutated by second call at index %d", i)
		}
	}
}

func TestStatusCountsIsolation(t *testing.T) {
	app, _, _ := newTestApp()
	first := app.stats.StatusSummary()
	capture := make([]StatusCount, len(first))
	copy(capture, first)
	if _, _, changed := app.store.UpdateStatus("S-1001", "in_review"); !changed {
		t.Fatal("expected S-1001 to move to in_review")
	}
	_ = app.stats.StatusSummary()
	for i := range capture {
		if capture[i] != first[i] {
			t.Fatalf("first StatusSummary result mutated by later call at index %d", i)
		}
	}
}

func TestTopMaterialsConcurrentRace(t *testing.T) {
	app, _, _ := newTestApp()
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 50; j++ {
				_ = app.stats.TopMaterials(5)
				_ = app.stats.StatusSummary()
			}
		}()
	}
	close(start)
	wg.Wait()
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
