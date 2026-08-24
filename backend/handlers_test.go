package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSampleHTTPFlow(t *testing.T) {
	server := httptest.NewServer(newServer(newSampleStore()))
	defer server.Close()
	checks := []struct {
		path string
		want int
	}{
		{"/healthz", http.StatusOK}, {"/api/samples", http.StatusOK}, {"/", http.StatusOK}, {"/app.js", http.StatusOK},
	}
	for _, check := range checks {
		response, err := http.Get(server.URL + check.path)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != check.want {
			t.Fatalf("GET %s: got %d, want %d", check.path, response.StatusCode, check.want)
		}
	}
	response, err := http.Post(server.URL+"/api/samples/S-1001/status", "application/json", strings.NewReader(`{"status":"in_review"}`))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("POST success: got %d", response.StatusCode)
	}
	response, err = http.Post(server.URL+"/api/samples/S-1001/status", "application/json", strings.NewReader(`{"status":"released"}`))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("POST second success: got %d", response.StatusCode)
	}
	response, err = http.Post(server.URL+"/api/samples/S-1001/status", "application/json", strings.NewReader(`{"status":"in_review"}`))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("POST invalid transition: got %d", response.StatusCode)
	}
	response, err = http.Post(server.URL+"/api/samples/missing/status", "application/json", strings.NewReader(`{"status":"released"}`))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("POST missing: got %d", response.StatusCode)
	}
}
