package main

import (
	"fmt"
	"net/http"
)

func newServer(store *SampleStore) *http.ServeMux {
	return newSampleApp(store).routes()
}

type sampleApp struct {
	store  *SampleStore
	audit  *SampleAudit
	batch  *SampleBatcher
	stats  *SampleStats
	review *ReviewStore
	export *SampleExporter
	ops    *OpsService
}

func newSampleApp(store *SampleStore) *sampleApp {
	audit := newSampleAudit()
	return &sampleApp{
		store:  store,
		audit:  audit,
		batch:  newSampleBatcher(store, audit),
		stats:  newSampleStats(store),
		review: newReviewStore(),
		export: newSampleExporter(store),
		ops:    newOpsService(seedOpsRecords()),
	}
}

func (a *sampleApp) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)
	mux.Handle("/api/samples", sampleHandler{store: a.store, audit: a.audit})
	mux.Handle("/api/samples/", sampleHandler{store: a.store, audit: a.audit})
	mux.HandleFunc("/api/samples/promote", promoteHandler(a.batch))
	mux.HandleFunc("/api/samples/stats", statsHandler(a.stats))
	mux.HandleFunc("/api/samples/review", reviewActionHandler(a.review, a.store))
	mux.HandleFunc("/api/samples/reviews", reviewListHandler(a.review))
	mux.HandleFunc("/api/samples/reviews/pending", reviewPendingHandler(a.review))
	mux.HandleFunc("/api/samples/export", exportHandler(a.export))
	gate := newOperatorGate(loadConfig().RequireOperator)
	opsAPI := newOpsAPIHandler(a.ops, gate)
	mux.Handle("/ops/records", opsAPI)
	mux.Handle("/ops/records/", opsAPI)
	mux.Handle("/ops/snapshot", opsAPI)
	mux.Handle("/ops/rules", opsAPI)
	mux.HandleFunc("/", staticHandler)
	return mux
}

func seedOpsRecords() []OpsRecord {
	priorities := []OpsPriority{OpsPriorityLow, OpsPriorityNormal, OpsPriorityHigh, OpsPriorityCritical}
	statuses := []OpsStatus{OpsStatusQueued, OpsStatusActive, OpsStatusPaused, OpsStatusClosed}
	owners := []string{"li", "wang", "zhang", "chen"}
	subjects := []string{
		"reagent calibration", "cold chain inspection", "freezer defrost check",
		"pipette certification", "incubator temp audit", "batch label verify",
		"cleanroom air sample", "media sterility check", "autoclave cycle log",
		"sample freezer audit",
	}
	records := make([]OpsRecord, 0, 40)
	for i := 1; i <= 40; i++ {
		records = append(records, OpsRecord{
			ID:        fmt.Sprintf("ops-%03d", i),
			Subject:   fmt.Sprintf("%s %d", subjects[(i-1)%len(subjects)], i),
			Owner:     owners[(i-1)%len(owners)],
			Status:    statuses[(i-1)%len(statuses)],
			Priority:  priorities[(i-1)%len(priorities)],
			Revision:  1,
			Labels:    map[string]string{"site": fmt.Sprintf("lab-%d", (i-1)%5+1)},
			CreatedAt: "2026-08-10T08:00:00Z",
			UpdatedAt: "2026-08-10T08:00:00Z",
		})
	}
	return records
}
