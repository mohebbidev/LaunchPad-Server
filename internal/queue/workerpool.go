package queue

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"
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
		processor:  processor,

		jobs:   make(chan Job, 200), // MAX BUFFER SIZE 200
		ctx:    ctx,
		cancel: cancel,
	}
}

func (wp *WorkerPool) Start() {
	for i := 0; i < wp.maxWorkers; i++ {
		wp.wg.Add(1)
		go wp.Worker(i)
	}
}

func (wp *WorkerPool) Worker(id int) {
	defer wp.wg.Done()

	for {
		select {
		case <-wp.ctx.Done():
			log.Printf("WORKER CANCELLEd هی: %v", id)
			return

		case job, ok := <-wp.jobs:
			if !ok {
				log.Printf("[WorkerPool] Worker %d shutting down (channel closed)", id)
				return
			}

			wp.Process(id, job)
		}
	}
}

func (wp *WorkerPool) Process(workerId int, job Job) {
	// start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err := wp.processor(ctx, job)

	if err != nil {
		wp.stats.failed++

		// retry 3 times
		if job.RetryCount < 3 {
			job.RetryCount++

			backoff := time.Duration(job.RetryCount) * 2 * time.Second

			time.Sleep(backoff)
			wp.Submit(job)
		} else {
			log.Printf("job %v fucked up all retries", job.ID)
		}

	} else {
		wp.stats.completed++
	}

	wp.stats.submitted++
}

func (wp *WorkerPool) Submit(job Job) error {
	select {
	case wp.jobs <- job:
		return nil
	default:
		return fmt.Errorf("BUFFER FULLLLLLLLLLLLLLLLL")
	}
}


func (wp *WorkerPool) ShutDown() {
	wp.cancel()
	close(wp.jobs)
}
func (wp *WorkerPool) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"queue_size":    len(wp.jobs),
		"active_workers": wp.maxWorkers,
		"submitted":     wp.stats.submitted,
		"completed":     wp.stats.completed,
		"failed":        wp.stats.failed,
	}
}