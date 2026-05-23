package queue

import (
	"context"
	"sync"
	"time"

	"github.com/lrstanley/go-ytdlp"

	"neodlp/internal/downloader"
)

// Job represents a single download task in the queue.
type Job struct {
	URL   string
	Opts  downloader.Options
	Index int
}

// ProgressUpdate wraps a ytdlp progress update with the originating job index.
type ProgressUpdate struct {
	JobIndex int
	Progress ytdlp.ProgressUpdate
}

// Result captures the outcome of a completed job.
type Result struct {
	Job        Job
	Err        error
	StartedAt  time.Time
	FinishedAt time.Time
}

// Queue orchestrates concurrent downloads with a bounded worker pool.
type Queue struct {
	jobs          []Job
	maxConcurrent int
	mu            sync.Mutex
}

// NewQueue creates a new download queue with the given concurrency limit.
func NewQueue(maxConcurrent int) *Queue {
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	return &Queue{
		maxConcurrent: maxConcurrent,
	}
}

// Add enqueues a download job. Must be called before Run.
func (q *Queue) Add(url string, opts downloader.Options) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.jobs = append(q.jobs, Job{
		URL:   url,
		Opts:  opts,
		Index: len(q.jobs),
	})
}

// Run starts the worker pool and processes all enqueued jobs.
// The progressFn callback is invoked from worker goroutines with per-job progress updates.
// Returns results for every job after all workers finish.
func (q *Queue) Run(ctx context.Context, progressFn func(update ProgressUpdate)) []Result {
	q.mu.Lock()
	jobs := make([]Job, len(q.jobs))
	copy(jobs, q.jobs)
	q.mu.Unlock()

	if len(jobs) == 0 {
		return nil
	}

	jobCh := make(chan Job, len(jobs))
	for _, j := range jobs {
		jobCh <- j
	}
	close(jobCh)

	resultCh := make(chan Result, len(jobs))

	workers := q.maxConcurrent
	if workers > len(jobs) {
		workers = len(jobs)
	}

	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for job := range jobCh {
				// Check if context is already cancelled
				if ctx.Err() != nil {
					resultCh <- Result{
						Job:        job,
						Err:        ctx.Err(),
						StartedAt:  time.Now(),
						FinishedAt: time.Now(),
					}
					continue
				}

				started := time.Now()

				// Create a per-job progress callback that tags updates with the job index
				var jobProgressFn ytdlp.ProgressCallbackFunc
				if progressFn != nil {
					idx := job.Index
					jobProgressFn = func(prog ytdlp.ProgressUpdate) {
						progressFn(ProgressUpdate{
							JobIndex: idx,
							Progress: prog,
						})
					}
				}

				_, err := downloader.DownloadWithProgress(
					ctx,
					[]string{job.URL},
					job.Opts,
					jobProgressFn,
				)

				resultCh <- Result{
					Job:        job,
					Err:        err,
					StartedAt:  started,
					FinishedAt: time.Now(),
				}
			}
		}()
	}

	// Wait for all workers in a separate goroutine, then close results channel
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	var results []Result
	for r := range resultCh {
		results = append(results, r)
	}

	return results
}
