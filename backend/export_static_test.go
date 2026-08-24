package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestExportHistoryCapped(t *testing.T) {
	exportLogMu.Lock()
	exportLog = nil
	exportLogMu.Unlock()
	exporter := newSampleExporter(newSampleStore())
	for i := 0; i < 200; i++ {
		exporter.RecordExport(11)
	}
	exportLogMu.Lock()
	count := len(exportLog)
	exportLogMu.Unlock()
	if count > exportLogCap {
		t.Fatalf("export log grew to %d, want capped at %d", count, exportLogCap)
	}
}

func TestExportCSVDetached(t *testing.T) {
	store := newSampleStore()
	exporter := newSampleExporter(store)
	first := exporter.CSV()
	expected := append([]byte(nil), first...)
	sample := store.samples["S-1001"]
	sample.Material = "water"
	store.samples["S-1001"] = sample
	_ = exporter.CSV()
	if !bytes.Equal(first, expected) {
		t.Fatal("first CSV content mutated by a later export")
	}
}

func TestStaticVisitLogCapped(t *testing.T) {
	visitedMu.Lock()
	visitedPaths = nil
	visitedMu.Unlock()
	for i := 0; i < 300; i++ {
		staticHandler(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	}
	if count := visitedPathCount(); count > visitedPathCap {
		t.Fatalf("visited path log grew to %d, want capped at %d", count, visitedPathCap)
	}
}

func TestStaticPageParallelLoads(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(staticHandler))
	defer server.Close()
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 30; j++ {
				_, _ = http.Get(server.URL + "/")
				_, _ = http.Get(server.URL + "/app.js")
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
