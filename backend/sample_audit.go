package main

import (
	"fmt"
	"sync"
	"time"
)

const sampleAuditCap = 1000

type SampleAuditEvent struct {
	ID       string `json:"id"`
	SampleID string `json:"sample_id"`
	Action   string `json:"action"`
	Actor    string `json:"actor"`
	At       string `json:"at"`
	Note     string `json:"note"`
}

type SampleAudit struct {
	mu     sync.RWMutex
	events []SampleAuditEvent
	next   int
}

func newSampleAudit() *SampleAudit {
	return &SampleAudit{events: []SampleAuditEvent{}}
}

func (a *SampleAudit) Add(sampleID, action, actor, note string) SampleAuditEvent {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.next++
	event := SampleAuditEvent{
		ID:       fmt.Sprintf("sa-%06d", a.next),
		SampleID: sampleID,
		Action:   action,
		Actor:    actor,
		At:       time.Now().UTC().Format(time.RFC3339Nano),
		Note:     note,
	}
	a.events = append(a.events, event)
	if len(a.events) > sampleAuditCap {
		a.events = append([]SampleAuditEvent(nil), a.events[len(a.events)-sampleAuditCap:]...)
	}
	return event
}

func (a *SampleAudit) For(sampleID string) []SampleAuditEvent {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]SampleAuditEvent, 0)
	for _, event := range a.events {
		if event.SampleID == sampleID {
			out = append(out, event)
		}
	}
	return out
}

func (a *SampleAudit) Recent(limit int) []SampleAuditEvent {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if limit < 1 {
		limit = 20
	}
	if limit > len(a.events) {
		limit = len(a.events)
	}
	out := make([]SampleAuditEvent, limit)
	copy(out, a.events[len(a.events)-limit:])
	return out
}

func (a *SampleAudit) Count() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.events)
}
