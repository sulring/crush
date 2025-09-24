package gorust

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Stage[T, R any] interface {
	Process(ctx context.Context, input T) (R, error)
	Name() string
}

type Pipeline[T any] struct {
	stages []Stage[any, any]
	mu     sync.RWMutex
	opts   PipelineOptions
}

type PipelineOptions struct {
	MaxConcurrency int
	Timeout        time.Duration
	RetryAttempts  int
	RetryDelay     time.Duration
}

type PipelineResult[T any] struct {
	Output T
	Error  error
	Stage  string
	Took   time.Duration
}

func NewPipeline[T any](opts PipelineOptions) *Pipeline[T] {
	if opts.MaxConcurrency <= 0 {
		opts.MaxConcurrency = 10
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.RetryAttempts <= 0 {
		opts.RetryAttempts = 3
	}
	if opts.RetryDelay <= 0 {
		opts.RetryDelay = time.Second
	}

	return &Pipeline[T]{
		stages: make([]Stage[any, any], 0),
		opts:   opts,
	}
}

func (p *Pipeline[T]) AddStage(stage Stage[any, any]) *Pipeline[T] {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stages = append(p.stages, stage)
	return p
}

func (p *Pipeline[T]) Execute(ctx context.Context, input T) <-chan PipelineResult[any] {
	results := make(chan PipelineResult[any], len(p.stages))
	
	go func() {
		defer close(results)
		
		current := input
		for _, stage := range p.stages {
			select {
			case <-ctx.Done():
				results <- PipelineResult[any]{
					Error: ctx.Err(),
					Stage: stage.Name(),
				}
				return
			default:
				result := p.executeStageWithRetry(ctx, stage, current)
				results <- result
				
				if result.Error != nil {
					return
				}
				current = result.Output
			}
		}
	}()
	
	return results
}

func (p *Pipeline[T]) executeStageWithRetry(ctx context.Context, stage Stage[any, any], input any) PipelineResult[any] {
	var lastErr error
	start := time.Now()
	
	for attempt := 0; attempt < p.opts.RetryAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return PipelineResult[any]{
					Error: ctx.Err(),
					Stage: stage.Name(),
					Took:  time.Since(start),
				}
			case <-time.After(p.opts.RetryDelay):
			}
		}
		
		stageCtx, cancel := context.WithTimeout(ctx, p.opts.Timeout)
		output, err := stage.Process(stageCtx, input)
		cancel()
		
		if err == nil {
			return PipelineResult[any]{
				Output: output,
				Stage:  stage.Name(),
				Took:   time.Since(start),
			}
		}
		
		lastErr = err
	}
	
	return PipelineResult[any]{
		Error: fmt.Errorf("stage %s failed after %d attempts: %w", stage.Name(), p.opts.RetryAttempts, lastErr),
		Stage: stage.Name(),
		Took:  time.Since(start),
	}
}

type TransformStage[T, R any] struct {
	name      string
	transform func(context.Context, T) (R, error)
}

func NewTransformStage[T, R any](name string, transform func(context.Context, T) (R, error)) *TransformStage[T, R] {
	return &TransformStage[T, R]{
		name:      name,
		transform: transform,
	}
}

func (s *TransformStage[T, R]) Name() string {
	return s.name
}

func (s *TransformStage[T, R]) Process(ctx context.Context, input T) (R, error) {
	return s.transform(ctx, input)
}

type FilterStage[T any] struct {
	name      string
	predicate func(context.Context, T) (bool, error)
}

func NewFilterStage[T any](name string, predicate func(context.Context, T) (bool, error)) *FilterStage[T] {
	return &FilterStage[T]{
		name:      name,
		predicate: predicate,
	}
}

func (s *FilterStage[T]) Name() string {
	return s.name
}

func (s *FilterStage[T]) Process(ctx context.Context, input T) (T, error) {
	keep, err := s.predicate(ctx, input)
	if err != nil {
		var zero T
		return zero, err
	}
	
	if !keep {
		var zero T
		return zero, fmt.Errorf("item filtered out")
	}
	
	return input, nil
}

type BatchProcessor[T, R any] struct {
	name      string
	batchSize int
	processor func(context.Context, []T) ([]R, error)
	buffer    []T
	mu        sync.Mutex
}

func NewBatchProcessor[T, R any](name string, batchSize int, processor func(context.Context, []T) ([]R, error)) *BatchProcessor[T, R] {
	return &BatchProcessor[T, R]{
		name:      name,
		batchSize: batchSize,
		processor: processor,
		buffer:    make([]T, 0, batchSize),
	}
}

func (b *BatchProcessor[T, R]) Name() string {
	return b.name
}

func (b *BatchProcessor[T, R]) Process(ctx context.Context, input T) ([]R, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.buffer = append(b.buffer, input)
	
	if len(b.buffer) >= b.batchSize {
		batch := make([]T, len(b.buffer))
		copy(batch, b.buffer)
		b.buffer = b.buffer[:0]
		
		return b.processor(ctx, batch)
	}
	
	return nil, nil
}

func (b *BatchProcessor[T, R]) Flush(ctx context.Context) ([]R, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if len(b.buffer) == 0 {
		return nil, nil
	}
	
	batch := make([]T, len(b.buffer))
	copy(batch, b.buffer)
	b.buffer = b.buffer[:0]
	
	return b.processor(ctx, batch)
}