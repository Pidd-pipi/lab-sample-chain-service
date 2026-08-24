package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReviewDecideAdvancesSample(t *testing.T) {
	app, _, _ := newTestApp()
	server := httptest.NewServer(app.routes())
	defer server.Close()
	submit := `{"sample_id":"S-1003","action":"submit","reviewer":"li","comment":"checking"}`
	resp, err := http.Post(server.URL+"/api/samples/review", "application/json", strings.NewReader(submit))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("submit: got %d", resp.StatusCode)
	}
	decide := `{"sample_id":"S-1003","action":"approve","reviewer":"li","comment":"ok"}`
	resp, err = http.Post(server.URL+"/api/samples/review", "application/json", strings.NewReader(decide))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("approve: got %d", resp.StatusCode)
	}
	sampleResp, err := http.Get(server.URL + "/api/samples/S-1003")
	if err != nil {
		t.Fatal(err)
	}
	defer sampleResp.Body.Close()
	var sample Sample
	if err := json.NewDecoder(sampleResp.Body).Decode(&sample); err != nil {
		t.Fatal(err)
	}
	if sample.Status != "released" {
		t.Fatalf("sample status after approval = %s, want released", sample.Status)
	}
	review, ok := app.review.Get("S-1003")
	if !ok {
		t.Fatal("review record missing")
	}
	if review.Status != ReviewApproved {
		t.Fatalf("review status = %s, want approved", review.Status)
	}
}

func TestReviewPendingListIncludesInProgress(t *testing.T) {
	app, _, _ := newTestApp()
	server := httptest.NewServer(app.routes())
	defer server.Close()
	submit := `{"sample_id":"S-1003","action":"submit","reviewer":"li","comment":"checking"}`
	resp, err := http.Post(server.URL+"/api/samples/review", "application/json", strings.NewReader(submit))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("submit: got %d", resp.StatusCode)
	}
	pendingResp, err := http.Get(server.URL + "/api/samples/reviews/pending")
	if err != nil {
		t.Fatal(err)
	}
	defer pendingResp.Body.Close()
	var body struct {
		Pending []SampleReview `json:"pending"`
	}
	if err := json.NewDecoder(pendingResp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, review := range body.Pending {
		if review.SampleID == "S-1003" {
			found = true
		}
	}
	if !found {
		t.Fatal("in-progress review S-1003 missing from pending list")
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

func TestReceivedSkipReviewRejected(t *testing.T) {
	app, _, _ := newTestApp()
	server := httptest.NewServer(app.routes())
	defer server.Close()
	resp, err := http.Post(server.URL+"/api/samples/S-1001/status", "application/json",
		strings.NewReader(`{"status":"released"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode < http.StatusBadRequest {
		t.Fatalf("received->released jump: got %d, want 4xx", resp.StatusCode)
	}
}

func TestReviewSubmitAdvancesSample(t *testing.T) {
	app, _, _ := newTestApp()
	server := httptest.NewServer(app.routes())
	defer server.Close()
	body := `{"sample_id":"S-1003","action":"submit","reviewer":"li","comment":"checking"}`
	resp, err := http.Post(server.URL+"/api/samples/review", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("submit: got %d", resp.StatusCode)
	}
	sampleResp, err := http.Get(server.URL + "/api/samples/S-1003")
	if err != nil {
		t.Fatal(err)
	}
	defer sampleResp.Body.Close()
	var sample Sample
	if err := json.NewDecoder(sampleResp.Body).Decode(&sample); err != nil {
		t.Fatal(err)
	}
	if sample.Status != "in_review" {
		t.Fatalf("sample status after submit = %s, want in_review", sample.Status)
	}
}
