package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func addReceivedSamples(app *sampleApp, n int) {
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("S-20%02d", i)
		app.store.samples[id] = Sample{ID: id, Material: "serum", Batch: "B-90", Status: "received"}
	}
}

func TestBatchPromoteLargeCompletes(t *testing.T) {
	app, _, _ := newTestApp()
	addReceivedSamples(app, 6)
	result, err := app.batch.Promote(context.Background(), "received", "in_review")
	if err != nil {
		t.Fatalf("promote returned error: %v", err)
	}
	if result.Promoted != 13 {
		t.Fatalf("promoted %d samples, want 13", result.Promoted)
	}
}

func TestBatchPromoteCancelPreservesError(t *testing.T) {
	app, _, _ := newTestApp()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := app.batch.Promote(ctx, "received", "in_review")
	if err == nil {
		t.Fatal("expected cancellation error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestBatchPromoteHTTPErrorStatus(t *testing.T) {
	app, _, _ := newTestApp()
	addReceivedSamples(app, 6)
	req := httptest.NewRequest(http.MethodPost, "/api/samples/promote",
		strings.NewReader(`{"from":"received","to":"in_review"}`))
	ctx, cancel := context.WithCancel(req.Context())
	cancel()
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	promoteHandler(app.batch).ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatal("promote handler returned 200 on canceled request")
	}
}

func TestBatchPromoteInvalidTargetRejected(t *testing.T) {
	app, _, _ := newTestApp()
	server := httptest.NewServer(app.routes())
	defer server.Close()
	body := `{"from":"received","to":"bogus"}`
	resp, err := http.Post(server.URL+"/api/samples/promote", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid target: got %d, want 400", resp.StatusCode)
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

func TestBatchPromoteAuditsSamples(t *testing.T) {
	app, _, audit := newTestApp()
	addReceivedSamples(app, 6)
	_, err := app.batch.Promote(context.Background(), "received", "in_review")
	if err != nil {
		t.Fatal(err)
	}
	if got := audit.Count(); got < 13 {
		t.Fatalf("promote audited %d events, want at least 13", got)
	}
}

