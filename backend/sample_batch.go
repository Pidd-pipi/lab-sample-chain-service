package main

import (
	"context"
	"errors"
	"fmt"
)

var ErrLeaseExhausted = errors.New("batch lease pool exhausted")

type BatchResult struct {
	Promoted int      `json:"promoted"`
	Skipped  int      `json:"skipped"`
	Failed   int      `json:"failed"`
	IDs      []string `json:"ids"`
}

type SampleBatcher struct {
	store  *SampleStore
	audit  *SampleAudit
	tokens chan struct{}
}

const batchLeaseLimit = 8

func newSampleBatcher(store *SampleStore, audit *SampleAudit) *SampleBatcher {
	return &SampleBatcher{store: store, audit: audit, tokens: make(chan struct{}, batchLeaseLimit)}
}

func (b *SampleBatcher) acquire() (release func(), ok bool) {
	select {
	case b.tokens <- struct{}{}:
		var once bool
		return func() {
			if !once {
				once = true
				<-b.tokens
			}
		}, true
	default:
		return nil, false
	}
}

func (b *SampleBatcher) finalize(result BatchResult) error {
	if result.Promoted == 0 {
		return nil
	}
	for _, id := range result.IDs {
		b.audit.Add(id, "promote", "system", "batch promote")
	}
	return nil
}

// Promote 把处于 from 状态的样品批量推进到 to 状态。
func (b *SampleBatcher) Promote(ctx context.Context, from, to string) (result BatchResult, err error) {
	defer func() { err = b.finalize(result) }()
	samples := b.store.List()
	for _, sample := range samples {
		if sample.Status != from {
			result.Skipped++
			continue
		}
		if err := ctx.Err(); err != nil {
			return result, err
		}
		release, ok := b.acquire()
		if !ok {
			err = fmt.Errorf("%w: pool exhausted after %d samples", ErrLeaseExhausted, result.Promoted)
			return result, err
		}
		defer release()
		updated, exists, changed := b.store.UpdateStatus(sample.ID, to)
		if !exists {
			result.Failed++
			continue
		}
		if !changed {
			result.Skipped++
			continue
		}
		result.Promoted++
		result.IDs = append(result.IDs, updated.ID)
	}
	return result, nil
}
