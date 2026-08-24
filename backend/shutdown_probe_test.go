package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestShutdownTimeoutHonored(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block
	}))
	defer srv.Close()
	defer close(block)
	go func() {
		_, _ = http.Get(srv.URL)
	}()
	time.Sleep(50 * time.Millisecond)
	done := make(chan error, 1)
	go func() {
		done <- shutdownWithTimeout(srv.Config, 200*time.Millisecond)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("shutdownWithTimeout did not honor the configured timeout")
	}
}

func TestConfigParseShutdownTimeout(t *testing.T) {
	t.Setenv("SHUTDOWN_TIMEOUT_SECONDS", "2")
	config := loadConfig()
	if config.ShutdownTimeoutSeconds != 2 {
		t.Fatalf("ShutdownTimeoutSeconds = %d, want 2", config.ShutdownTimeoutSeconds)
	}
}

func TestHealthProbeLogBounded(t *testing.T) {
	healthProbesMu.Lock()
	healthProbes = nil
	healthProbesMu.Unlock()
	for i := 0; i < 300; i++ {
		healthHandler(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz", nil))
	}
	healthProbesMu.Lock()
	count := len(healthProbes)
	healthProbesMu.Unlock()
	if count > healthProbeCap {
		t.Fatalf("health probe log grew to %d, want capped at %d", count, healthProbeCap)
	}
}

func TestRequestLogCapped(t *testing.T) {
	requestLogMu.Lock()
	requestLog = nil
	requestLogMu.Unlock()
	for i := 0; i < 300; i++ {
		recordRequestLog("GET /api/samples")
	}
	requestLogMu.Lock()
	count := len(requestLog)
	requestLogMu.Unlock()
	if count > requestLogCap {
		t.Fatalf("request log grew to %d, want capped at %d", count, requestLogCap)
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
