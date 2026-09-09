package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestHighRPMConcurrentSubmissions(t *testing.T) {
	const (
		workers         = 50
		queueBuffer     = 5000
		producers       = 100
		jobsPerProducer = 1000
	)

	q := NewInMemoryJobQueue(queueBuffer, workers)

	fastRetry := RetryOptions{
		MaxRetries: 1,
		Backoff:    0,
		Jitter:     0,
	}

	q.Start()

	var (
		wg        sync.WaitGroup
		submitted atomic.Int64
		failed    atomic.Int64
	)

	start := time.Now()

	for producerID := 0; producerID < producers; producerID++ {
		wg.Add(1)

		go func(pid int) {
			defer wg.Done()

			for i := 0; i < jobsPerProducer; i++ {
				job := Job{
					jobID: fmt.Sprintf("producer-%d-job-%d", pid, i),
					Payload: struct {
						Producer int
						Index    int
					}{
						Producer: pid,
						Index:    i,
					},
					Options: &fastRetry,
				}

				if err := q.Submit(job); err != nil {
					failed.Add(1)
					continue
				}

				submitted.Add(1)
			}
		}(producerID)
	}

	wg.Wait()

	submissionDuration := time.Since(start)

	// Shutdown waits for workers to finish draining the queue.
	q.Shutdown()

	totalDuration := time.Since(start)

	expected := int64(producers * jobsPerProducer)

	rps := float64(submitted.Load()) / submissionDuration.Seconds()
	rpm := rps * 60

	t.Logf("Expected jobs:       %d", expected)
	t.Logf("Submitted jobs:      %d", submitted.Load())
	t.Logf("Submit failed:       %d", failed.Load())
	t.Logf("Submission duration: %v", submissionDuration)
	t.Logf("Total duration:      %v", totalDuration)
	t.Logf("Approx RPS:          %.2f", rps)
	t.Logf("Approx RPM:          %.2f", rpm)

	if submitted.Load() != expected {
		t.Fatalf(
			"expected %d submitted jobs, got %d",
			expected,
			submitted.Load(),
		)
	}

	if failed.Load() != 0 {
		t.Fatalf(
			"expected 0 submission failures, got %d",
			failed.Load(),
		)
	}
}