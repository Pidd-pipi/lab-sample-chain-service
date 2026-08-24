package main

import (
	"net/http"
	"os"
	"sync"
)

const visitedPathCap = 100

var (
	visitedMu    sync.Mutex
	visitedPaths []string
)

func recordVisitedPath(path string) {
	visitedMu.Lock()
	defer visitedMu.Unlock()
	visitedPaths = append(visitedPaths, path)
	if len(visitedPaths) > visitedPathCap {
		visitedPaths = append([]string(nil), visitedPaths[len(visitedPaths)-visitedPathCap:]...)
	}
}

func visitedPathCount() int {
	visitedMu.Lock()
	defer visitedMu.Unlock()
	return len(visitedPaths)
}

func staticHandler(w http.ResponseWriter, r *http.Request) {
	recordVisitedPath(r.URL.Path)
	name := "web/index.html"
	if r.URL.Path == "/app.js" {
		name = "web/app.js"
	}
	data, err := os.ReadFile(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if name == "web/app.js" {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	} else {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	}
	_, _ = w.Write(data)
}
