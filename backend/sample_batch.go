package main

import (
	"context"
	"errors"
	"fmt"
)

// ErrBatchInvalidTransition 表示批量推进请求的 from -> to 状态转移不被允许。
var ErrBatchInvalidTransition = errors.New("batch promote transition is not allowed")

// BatchResult 描述一次批量推进的汇总结果。
type BatchResult struct {
	Promoted int      `json:"promoted"`
	Skipped  int      `json:"skipped"`
	Failed   int      `json:"failed"`
	IDs      []string `json:"ids"`
}

type SampleBatcher struct {
	store *SampleStore
	audit *SampleAudit
}

func newSampleBatcher(store *SampleStore, audit *SampleAudit) *SampleBatcher {
	return &SampleBatcher{store: store, audit: audit}
}

// Promote 把处于 from 状态的样品批量推进到 to 状态。
//
// 全程顺序执行：每个样品的状态变更单独落库并立刻写审计，避免整批失败时丢失
// 已提交的记录与审计事件。处理每个样品前都会检查 ctx，客户端中途取消会在
// 下一次处理前停止，已落库的部分结果仍随 result 返回，调用方据此返回非 200
// 响应并记录日志，而不是把部分成功伪装成 200。
func (b *SampleBatcher) Promote(ctx context.Context, from, to string) (BatchResult, error) {
	var result BatchResult
	if from == "" || to == "" {
		return result, fmt.Errorf("%w: from and to status are required", ErrBatchInvalidTransition)
	}
	if !sampleTransitions[from][to] {
		return result, fmt.Errorf("%w: %s -> %s", ErrBatchInvalidTransition, from, to)
	}
	samples := b.store.List()
	for _, sample := range samples {
		// 每个样品处理前检查取消，避免客户端断开后还继续推进后续样品。
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if sample.Status != from {
			result.Skipped++
			continue
		}
		updated, exists, changed := b.store.UpdateStatus(sample.ID, to)
		if !exists {
			result.Failed++
			continue
		}
		if !changed {
			result.Skipped++
			continue
		}
		// 单条审计紧跟单条落库，批量中途失败时已推进的样品仍可在审计中查到。
		b.audit.Add(sample.ID, "promote", "system", "batch promote")
		result.Promoted++
		result.IDs = append(result.IDs, updated.ID)
	}
	return result, nil
}
