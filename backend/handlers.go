package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
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
		result, err := batch.Promote(r.Context(), req.From, req.To)
		if err != nil {
			// 批量推进可能在中途失败或被取消，但已落库的样品不会回滚：
			// 既要记日志、返回非 200，也要把已经推进的 result 带回去，方便排查。
			requestID := w.Header().Get("X-Request-ID")
			log.Printf("batch promote failed request_id=%s from=%s to=%s promoted=%d failed=%d err=%v",
				requestID, req.From, req.To, result.Promoted, result.Failed, err)
			status := http.StatusInternalServerError
			switch {
			case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
				status = 499
			case errors.Is(err, ErrBatchInvalidTransition):
				status = http.StatusBadRequest
			}
			writeJSON(w, status, map[string]any{
				"error":  err.Error(),
				"result": result,
			})
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}
