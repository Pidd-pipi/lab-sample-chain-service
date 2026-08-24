package main

import (
	"net/http"
	"sort"
)

type StatusCount struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

type MaterialCount struct {
	Material string `json:"material"`
	Count    int    `json:"count"`
}

type SampleStats struct {
	store *SampleStore
}

func newSampleStats(store *SampleStore) *SampleStats {
	return &SampleStats{store: store}
}

func (st *SampleStats) TopMaterials(n int) []MaterialCount {
	counts := map[string]int{}
	for _, sample := range st.store.List() {
		counts[sample.Material]++
	}
	out := make([]MaterialCount, 0, len(counts))
	for material, count := range counts {
		out = append(out, MaterialCount{Material: material, Count: count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Material < out[j].Material
	})
	if n >= 0 && n < len(out) {
		out = out[:n]
	}
	return out
}

func (st *SampleStats) StatusSummary() []StatusCount {
	counts := map[string]int{}
	for _, sample := range st.store.List() {
		counts[sample.Status]++
	}
	out := make([]StatusCount, 0, len(counts))
	for status, count := range counts {
		out = append(out, StatusCount{Status: status, Count: count})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Status < out[j].Status })
	return out
}

func statsHandler(st *SampleStats) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"top_materials": st.TopMaterials(5),
			"by_status":     st.StatusSummary(),
		})
	}
}
