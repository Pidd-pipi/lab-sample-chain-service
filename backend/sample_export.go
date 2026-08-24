package main

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

const exportLogCap = 100

var (
	exportLogMu sync.Mutex
	exportLog   [][]string
)

type SampleExporter struct {
	store *SampleStore
}

func newSampleExporter(store *SampleStore) *SampleExporter {
	return &SampleExporter{store: store}
}

func (e *SampleExporter) Rows() [][]string {
	samples := e.store.List()
	rows := make([][]string, 0, len(samples)+1)
	rows = append(rows, []string{"id", "material", "batch", "status"})
	for _, sample := range samples {
		rows = append(rows, []string{sample.ID, sample.Material, sample.Batch, sample.Status})
	}
	return rows
}

func (e *SampleExporter) CSV() []byte {
	rows := e.Rows()
	var buf bytes.Buffer
	for _, row := range rows {
		buf.WriteString(strings.Join(row, ","))
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}

func (e *SampleExporter) RecordExport(rows int) {
	exportLogMu.Lock()
	defer exportLogMu.Unlock()
	exportLog = append(exportLog, []string{strconv.Itoa(rows)})
	if len(exportLog) > exportLogCap {
		exportLog = exportLog[len(exportLog)-exportLogCap:]
	}
}

func exportHandler(e *SampleExporter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		rows := e.Rows()
		data := e.CSV()
		e.RecordExport(len(rows))
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=samples.csv")
		_, _ = w.Write(data)
	}
}
