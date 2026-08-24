package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

type sampleStatusRequest struct {
	Status string `json:"status"`
}

type sampleHandler struct {
	store *SampleStore
	audit *SampleAudit
}

func (h sampleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/samples" {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"samples": h.store.List()})
		return
	}
	if !strings.HasPrefix(r.URL.Path, "/api/samples/") {
		writeError(w, http.StatusNotFound, "route not found")
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/samples/"), "/")
	if len(parts) < 1 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "route not found")
		return
	}
	if r.Method == http.MethodGet && len(parts) == 1 {
		sample, ok := h.store.Get(parts[0])
		if !ok {
			writeError(w, http.StatusNotFound, "sample not found")
			return
		}
		writeJSON(w, http.StatusOK, sample)
		return
	}
	if r.Method == http.MethodGet && len(parts) == 2 && parts[1] == "audit" {
		writeJSON(w, http.StatusOK, map[string]any{"events": h.audit.For(parts[0])})
		return
	}
	if r.Method != http.MethodPost || len(parts) != 2 || parts[1] != "status" {
		writeError(w, http.StatusNotFound, "route not found")
		return
	}
	var request sampleStatusRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := validateSampleStatus(request.Status); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	sample, exists, changed := h.store.UpdateStatus(parts[0], request.Status)
	if !exists {
		writeError(w, http.StatusNotFound, "sample not found")
		return
	}
	if !changed {
		writeError(w, http.StatusConflict, "invalid status transition")
		return
	}
	h.audit.Add(parts[0], "status_update", opsActorFromRequest(r), request.Status)
	writeJSON(w, http.StatusOK, sample)
}

type promoteRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func promoteHandler(batch *SampleBatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var req promoteRequest
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.From == "" {
			writeError(w, http.StatusBadRequest, "from status is required")
			return
		}
		result, _ := batch.Promote(r.Context(), req.From, req.To)
		writeJSON(w, http.StatusOK, result)
	}
}
