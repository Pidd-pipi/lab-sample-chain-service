package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

const healthProbeCap = 200

var (
	healthProbesMu sync.Mutex
	healthProbes   []time.Time
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	healthProbesMu.Lock()
	healthProbes = append(healthProbes, time.Now())
	healthProbesMu.Unlock()
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "lab-sample-chain"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
