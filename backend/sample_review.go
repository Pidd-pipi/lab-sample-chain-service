package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"
)

type ReviewStatus string

const (
	ReviewPending    ReviewStatus = "pending"
	ReviewInProgress ReviewStatus = "in_progress"
	ReviewApproved   ReviewStatus = "approved"
	ReviewRejected   ReviewStatus = "rejected"
)

var (
	ErrReviewNotFound   = errors.New("review record not found")
	ErrReviewTransition = errors.New("review status transition is not allowed")
)

var reviewTransitions = map[ReviewStatus]map[ReviewStatus]bool{
	ReviewPending:    {ReviewInProgress: true},
	ReviewInProgress: {ReviewApproved: true, ReviewRejected: true},
	ReviewApproved:   {},
	ReviewRejected:   {},
}

type SampleReview struct {
	SampleID  string       `json:"sample_id"`
	Status    ReviewStatus `json:"status"`
	Reviewer  string       `json:"reviewer"`
	Comment   string       `json:"comment"`
	UpdatedAt string       `json:"updated_at"`
}

type ReviewStore struct {
	mu      sync.RWMutex
	reviews map[string]SampleReview
}

func newReviewStore() *ReviewStore {
	return &ReviewStore{reviews: map[string]SampleReview{
		"S-1001": {SampleID: "S-1001", Status: ReviewPending, UpdatedAt: "2026-08-20T09:00:00Z"},
		"S-1002": {SampleID: "S-1002", Status: ReviewInProgress, Reviewer: "wang", Comment: "reviewing", UpdatedAt: "2026-08-21T10:30:00Z"},
		"S-1003": {SampleID: "S-1003", Status: ReviewPending, UpdatedAt: "2026-08-21T11:00:00Z"},
		"S-1004": {SampleID: "S-1004", Status: ReviewApproved, Reviewer: "li", Comment: "ok", UpdatedAt: "2026-08-22T08:15:00Z"},
	}}
}

func (rs *ReviewStore) Get(sampleID string) (SampleReview, bool) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	review, ok := rs.reviews[sampleID]
	return review, ok
}

func (rs *ReviewStore) List() []SampleReview {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	out := make([]SampleReview, 0, len(rs.reviews))
	for _, review := range rs.reviews {
		out = append(out, review)
	}
	return out
}

func (rs *ReviewStore) PendingReviews() []SampleReview {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	out := make([]SampleReview, 0)
	for _, review := range rs.reviews {
		if review.Status == ReviewPending || review.Status == ReviewInProgress {
			out = append(out, review)
		}
	}
	return out
}

func (rs *ReviewStore) Submit(sampleID, reviewer, comment string, store *SampleStore) (SampleReview, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	review, ok := rs.reviews[sampleID]
	if !ok {
		return SampleReview{}, ErrReviewNotFound
	}
	if !reviewTransitions[review.Status][ReviewInProgress] {
		return SampleReview{}, ErrReviewTransition
	}
	// Push the sample into the review pipeline so its status tracks the review.
	if store != nil {
		if _, exists, changed := store.UpdateStatus(sampleID, "in_review"); !exists || !changed {
			return SampleReview{}, ErrReviewTransition
		}
	}
	review.Status = ReviewInProgress
	review.Reviewer = reviewer
	review.Comment = comment
	review.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	rs.reviews[sampleID] = review
	return review, nil
}

func (rs *ReviewStore) Decide(sampleID string, approve bool, reviewer, comment string, store *SampleStore) (SampleReview, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	review, ok := rs.reviews[sampleID]
	if !ok {
		return SampleReview{}, ErrReviewNotFound
	}
	target := ReviewRejected
	if approve {
		target = ReviewApproved
	}
	if !reviewTransitions[review.Status][target] {
		return SampleReview{}, ErrReviewTransition
	}
	// Carry the decision over to the sample so its status tracks the review outcome.
	if store != nil {
		sampleStatus := "rejected"
		if approve {
			sampleStatus = "released"
		}
		if _, exists, changed := store.UpdateStatus(sampleID, sampleStatus); !exists || !changed {
			return SampleReview{}, ErrReviewTransition
		}
	}
	review.Status = target
	review.Reviewer = reviewer
	review.Comment = comment
	review.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	rs.reviews[sampleID] = review
	return review, nil
}

type reviewActionRequest struct {
	SampleID string `json:"sample_id"`
	Action   string `json:"action"`
	Reviewer string `json:"reviewer"`
	Comment  string `json:"comment"`
}

func reviewActionHandler(rs *ReviewStore, store *SampleStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var req reviewActionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.SampleID == "" {
			writeError(w, http.StatusBadRequest, "sample_id is required")
			return
		}
		switch req.Action {
		case "submit":
			review, err := rs.Submit(req.SampleID, req.Reviewer, req.Comment, store)
			if err != nil {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, review)
		case "approve", "reject":
			review, err := rs.Decide(req.SampleID, req.Action == "approve", req.Reviewer, req.Comment, store)
			if err != nil {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, review)
		default:
			writeError(w, http.StatusBadRequest, "unknown action")
		}
	}
}

func reviewListHandler(rs *ReviewStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"reviews": rs.List()})
	}
}

func reviewPendingHandler(rs *ReviewStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"pending": rs.PendingReviews()})
	}
}
