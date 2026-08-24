package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

var opsCreateCounter uint64

func opsCreateSequence() int {
	return int(atomic.AddUint64(&opsCreateCounter, 1))
}

type opsGate interface {
	Allow(r *http.Request) bool
}

type operatorGate struct {
	requireOperator bool
}

func (g *operatorGate) Allow(r *http.Request) bool {
	if g == nil {
		return true
	}
	if !g.requireOperator {
		return true
	}
	return strings.TrimSpace(r.Header.Get("X-Operator")) != ""
}

func newOperatorGate(require bool) opsGate {
	if require {
		return &operatorGate{requireOperator: true}
	}
	return &operatorGate{requireOperator: false}
}

func opsEnterpriseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		w.Header().Set("X-Operations-Domain", opsDomainName)
		if strings.TrimSpace(r.Header.Get("X-Request-ID")) == "" {
			w.Header().Set("X-Operations-Request", "generated")
		} else {
			w.Header().Set("X-Operations-Request", "provided")
		}
		defer func() { w.Header().Set("X-Operations-Latency-Ms", formatOpsInt(int(time.Since(start).Milliseconds()))) }()
		next.ServeHTTP(w, r)
	})
}
func formatOpsInt(value int) string {
	if value == 0 {
		return "0"
	}
	out := ""
	for value > 0 {
		out = string(rune('0'+value%10)) + out
		value /= 10
	}
	return out
}
func opsJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func opsAllowed(method string, allowed ...string) bool {
	for _, candidate := range allowed {
		if method == candidate {
			return true
		}
	}
	return false
}
func opsPathID(path, prefix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	return strings.Trim(strings.TrimPrefix(path, prefix), "/")
}
func opsActorFromRequest(r *http.Request) string {
	value := strings.TrimSpace(r.Header.Get("X-Operator"))
	if value == "" {
		return "web"
	}
	return value
}
func opsNoStore(w http.ResponseWriter)    { w.Header().Set("Cache-Control", "no-store") }
func opsRequestID(r *http.Request) string { return r.Header.Get("X-Request-ID") }
func enrichOpsLabels(record OpsRecord) OpsRecord {
	record = normalizeOpsRecord(record)
	record.Labels["source"] = "http"
	return record
}
