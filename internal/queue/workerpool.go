package queue

import (
	"context"
	"runtime"
	"sync"
)

type WorkerPool struct {
	jobs       chan Job
	maxWorkers int
	processor  func(ctx context.Context, job Job) error
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup

	stats struct {
		submitted int
		completed int
		failed    int
	}
}

func NewWorkerPool(
	maxWorkers int,
	processor func(ctx context.Context, job Job) error) *WorkerPool {
		if maxWorkers <= 0 {
			maxWorkers = runtime.NumCPU()
		}

		ctx, cancel := context.WithCancel(context.Background())

		return &WorkerPool{
			maxWorkers: maxWorkers,
			processor: processor,

			jobs: make(chan Job, 200), // MAX BUFFER SIZE 200
			ctx: ctx,
			cancel: cancel,
		}
}

