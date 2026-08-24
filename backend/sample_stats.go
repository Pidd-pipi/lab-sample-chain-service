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

var materialScratch []MaterialCount
var statusScratch []StatusCount

func (st *SampleStats) TopMaterials(n int) []MaterialCount {
	counts := map[string]int{}
	for _, sample := range st.store.List() {
		counts[sample.Material]++
	}
	materialScratch = materialScratch[:0]
	for material, count := range counts {
		materialScratch = append(materialScratch, MaterialCount{Material: material, Count: count})
	}
	sort.Slice(materialScratch, func(i, j int) bool {
		if materialScratch[i].Count != materialScratch[j].Count {
			return materialScratch[i].Count > materialScratch[j].Count
		}
		return materialScratch[i].Material < materialScratch[j].Material
	})
	if n >= 0 && n < len(materialScratch) {
		materialScratch = materialScratch[:n]
	}
	return materialScratch
}

func (st *SampleStats) StatusSummary() []StatusCount {
	counts := map[string]int{}
	for _, sample := range st.store.List() {
		counts[sample.Status]++
	}
	statusScratch = statusScratch[:0]
	for status, count := range counts {
		statusScratch = append(statusScratch, StatusCount{Status: status, Count: count})
	}
	sort.Slice(statusScratch, func(i, j int) bool { return statusScratch[i].Status < statusScratch[j].Status })
	return statusScratch
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
