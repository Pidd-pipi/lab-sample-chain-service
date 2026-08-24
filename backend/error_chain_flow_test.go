package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpsGetUnknownRecord404(t *testing.T) {
	app, _, _ := newTestApp()
	server := httptest.NewServer(app.routes())
	defer server.Close()
	resp, err := http.Get(server.URL + "/ops/records/ops-999")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("GET missing record: got %d, want 404", resp.StatusCode)
	}
}

func TestOpsMoveMissingRecord404(t *testing.T) {
	app, _, _ := newTestApp()
	server := httptest.NewServer(app.routes())
	defer server.Close()
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/ops/records/ops-999/transition",
		strings.NewReader(`{"expected":1,"target":"active","actor":"li"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Operator", "li")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("POST transition missing record: got %d, want 404", resp.StatusCode)
	}
}

func TestOpsStoreNotFoundChain(t *testing.T) {
	store := newOpsStore(seedOpsRecords())
	_, err := store.Get(context.Background(), "ops-999")
	if err == nil {
		t.Fatal("expected error for missing record")
	}
	if !errors.Is(err, ErrOpsNotFound) {
		t.Fatalf("errors.Is(err, ErrOpsNotFound) = false, err=%v", err)
	}
}

func TestOpsStoreConflictChain(t *testing.T) {
	store := newOpsStore(seedOpsRecords())
	record, err := store.Get(context.Background(), "ops-001")
	if err != nil {
		t.Fatal(err)
	}
	err = store.Update(context.Background(), record, record.Revision+5)
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !errors.Is(err, ErrOpsConflict) {
		t.Fatalf("errors.Is(err, ErrOpsConflict) = false, err=%v", err)
	}
}

func TestOpsCodeClassifies(t *testing.T) {
	store := newOpsStore(seedOpsRecords())
	_, notFound := store.Get(context.Background(), "ops-999")
	if code := opsCode(notFound); code != "not_found" {
		t.Fatalf("opsCode(notFound) = %q, want not_found", code)
	}
	record, _ := store.Get(context.Background(), "ops-001")
	conflict := store.Update(context.Background(), record, record.Revision+5)
	if code := opsCode(conflict); code != "conflict" {
		t.Fatalf("opsCode(conflict) = %q, want conflict", code)
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

func TestOpsStoreUpdateNotFoundChain(t *testing.T) {
	store := newOpsStore(seedOpsRecords())
	missing := OpsRecord{ID: "ops-999", Subject: "x", Owner: "li", Status: OpsStatusQueued, Priority: OpsPriorityHigh, Labels: map[string]string{"site": "lab-1"}}
	err := store.Update(context.Background(), missing, 0)
	if err == nil {
		t.Fatal("expected not-found error")
	}
	if !errors.Is(err, ErrOpsNotFound) {
		t.Fatalf("errors.Is(err, ErrOpsNotFound) = false, err=%v", err)
	}
}
