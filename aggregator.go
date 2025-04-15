package agg

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type Job struct {
	ID       string
	Interval time.Duration
	LastRun  time.Time
	Execute  func(ctx context.Context) error
}

const (
	defaultQueueSize  = 100
	schedulerInterval = 30 * time.Second
)

type Aggregator struct {
	jobs    map[string]*Job
	queue   chan *Job
	workers int
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
}

func New(workers int) *Aggregator {
	ctx, cancel := context.WithCancel(context.Background())
	return &Aggregator{
		jobs:    make(map[string]*Job),
		queue:   make(chan *Job, defaultQueueSize),
		workers: workers,
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (a *Aggregator) AddJob(id string, interval time.Duration, execute func(ctx context.Context) error, lastrun time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.jobs[id] = &Job{
		ID:       id,
		Interval: interval,
		Execute:  execute,
		LastRun:  lastrun,
	}
}

func (a *Aggregator) Start() {
	for i := 0; i < a.workers; i++ {
		go a.worker()
	}
	go a.scheduler()
}

func (a *Aggregator) Stop() {
	a.cancel()
	close(a.queue)
}

func (a *Aggregator) worker() {
	for {
		select {
		case <-a.ctx.Done():
			return
		case job := <-a.queue:
			if err := job.Execute(a.ctx); err != nil {
				slog.Error("job execution failed",
					"id", job.ID,
					"error", err)
			} else {
				slog.Info("job completed successfully", "id", job.ID)
			}
		}
	}
}

func (a *Aggregator) scheduler() {
	ticker := time.NewTicker(schedulerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			jobs := a.getJobsToRun(now)

			for _, job := range jobs {
				select {
				case a.queue <- job:
					a.updateJobLastRun(job.ID, now)
				default:
					slog.Warn("queue is full, skipping job", "id", job.ID)
				}
			}
		}
	}
}

func (a *Aggregator) getJobsToRun(now time.Time) []*Job {
	a.mu.Lock()
	defer a.mu.Unlock()

	var jobs []*Job
	for _, job := range a.jobs {
		if now.Sub(job.LastRun) >= job.Interval {
			jobs = append(jobs, job)
		}
	}
	return jobs
}

func (a *Aggregator) updateJobLastRun(id string, now time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if job, exists := a.jobs[id]; exists {
		job.LastRun = now
	}
}
