package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpsAddWithoutLabelsNoPanic(t *testing.T) {
	app, _, _ := newTestApp()
	server := httptest.NewServer(app.routes())
	defer server.Close()
	body := `{"id":"ops-t1","subject":"quick check","owner":"li","priority":"high"}`
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/ops/records", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Operator", "li")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusInternalServerError {
		t.Fatal("create without labels panicked with 500")
	}
}

func TestOpsWriteRequiresOperator(t *testing.T) {
	app, _, _ := newTestApp()
	server := httptest.NewServer(app.routes())
	defer server.Close()
	body := `{"id":"ops-t2","subject":"quick check","owner":"li","priority":"high","labels":{"site":"lab-1"}}`
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/ops/records", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("create without operator header: got %d, want 403", resp.StatusCode)
	}
}

func TestOpsAddWithoutOwnerRejected(t *testing.T) {
	server := httptest.NewServer(newServer(newSampleStore()))
	defer server.Close()
	body := `{"id":"ops-t3","subject":"quick check","priority":"high","labels":{"site":"lab-1"}}`
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/ops/records", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Operator", "li")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("create without owner: got %d, want 403", resp.StatusCode)
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

func TestOpsRulesEndpointListsRules(t *testing.T) {
	app, _, _ := newTestApp()
	server := httptest.NewServer(app.routes())
	defer server.Close()
	resp, err := http.Get(server.URL + "/ops/rules")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /ops/rules: got %d", resp.StatusCode)
	}
	var body struct {
		Rules []OpsRule `json:"rules"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Rules) < 100 {
		t.Fatalf("rules list has %d entries, want at least 100", len(body.Rules))
	}
}
