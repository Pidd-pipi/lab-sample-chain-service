package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

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

func TestConcurrentStatusTransitionNoLostUpdate(t *testing.T) {
	app, _, audit := newTestApp()
	server := httptest.NewServer(app.routes())
	defer server.Close()
	const workers = 8
	start := make(chan struct{})
	results := make([]int, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			resp, err := http.Post(server.URL+"/api/samples/S-1001/status", "application/json", strings.NewReader(`{"status":"in_review"}`))
			if err != nil {
				results[idx] = -1
				return
			}
			resp.Body.Close()
			results[idx] = resp.StatusCode
		}(i)
	}
	close(start)
	wg.Wait()
	success := 0
	for _, code := range results {
		switch code {
		case http.StatusOK:
			success++
		case http.StatusConflict:
		default:
			t.Fatalf("unexpected status code %d", code)
		}
	}
	if success != 1 {
		t.Fatalf("got %d successful transitions, want exactly 1 (lost update)", success)
	}
	if audit.Count() != 1 {
		t.Fatalf("audit has %d events, want 1 for the single success", audit.Count())
	}
}

func TestAuditConcurrentAppendRace(t *testing.T) {
	_, _, audit := newTestApp()
	const workers = 8
	const perWorker = 50
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-start
			for j := 0; j < perWorker; j++ {
				audit.Add("S-1001", "status_update", "tester", "note")
			}
		}(i)
	}
	close(start)
	wg.Wait()
	if got := audit.Count(); got != workers*perWorker {
		t.Fatalf("audit count = %d, want %d", got, workers*perWorker)
	}
}

func TestAuditRecentReadWriteRace(t *testing.T) {
	_, _, audit := newTestApp()
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 100; i++ {
			audit.Add("S-1002", "status_update", "tester", "note")
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 100; i++ {
			_ = audit.Recent(10)
			_ = audit.For("S-1002")
		}
	}()
	close(start)
	wg.Wait()
}

func TestAuditMemoryBounded(t *testing.T) {
	_, _, audit := newTestApp()
	for i := 0; i < 1500; i++ {
		audit.Add("S-1001", "status_update", "tester", "note")
	}
	if got := audit.Count(); got > sampleAuditCap {
		t.Fatalf("audit grew to %d events, want capped at %d", got, sampleAuditCap)
	}
}
