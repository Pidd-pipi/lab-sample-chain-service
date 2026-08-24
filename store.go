package main

import "sync"

type SampleStore struct {
	mu      sync.RWMutex
	samples map[string]Sample
}

func newSampleStore() *SampleStore {
	return &SampleStore{samples: map[string]Sample{
		"S-1001": {ID: "S-1001", Material: "serum", Batch: "B-77", Status: "received"},
		"S-1002": {ID: "S-1002", Material: "buffer", Batch: "B-78", Status: "in_review"},
		"S-1003": {ID: "S-1003", Material: "plasma", Batch: "B-79", Status: "received"},
		"S-1004": {ID: "S-1004", Material: "urine", Batch: "B-79", Status: "in_review"},
		"S-1005": {ID: "S-1005", Material: "serum", Batch: "B-80", Status: "received"},
		"S-1006": {ID: "S-1006", Material: "buffer", Batch: "B-80", Status: "received"},
		"S-1007": {ID: "S-1007", Material: "plasma", Batch: "B-81", Status: "in_review"},
		"S-1008": {ID: "S-1008", Material: "urine", Batch: "B-81", Status: "received"},
		"S-1009": {ID: "S-1009", Material: "serum", Batch: "B-82", Status: "received"},
		"S-1010": {ID: "S-1010", Material: "buffer", Batch: "B-82", Status: "received"},
	}}
}

func (s *SampleStore) Get(id string) (Sample, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sample, ok := s.samples[id]
	return sample, ok
}

func (s *SampleStore) List() []Sample {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Sample, 0, len(s.samples))
	for _, sample := range s.samples {
		result = append(result, sample)
	}
	return result
}

func (s *SampleStore) UpdateStatus(id, status string) (Sample, bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sample, exists := s.samples[id]
	if !exists {
		return Sample{}, false, false
	}
	if !sampleTransitions[sample.Status][status] {
		return sample, true, false
	}
	sample.Status = status
	s.samples[id] = sample
	return sample, true, true
}
