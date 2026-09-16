package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestTokenBucketVeryHighLoad fires 10 million requests
// across 10,000 goroutines and 100 users.
//
// This is primarily a concurrency/load test.
// Capacity and refill rate are intentionally huge so that
// rate limiting itself doesn't dominate the test.
func TestTokenBucketVeryHighLoad(t *testing.T) {
	const (
		goroutines        = 10_000
		requestsPerWorker = 1_000
		numberOfUsers     = 100
	)

	rl := NewTokenBucketRateLimiter(
		1_000_000_000, // capacity
		1_000_000,     // refill rate/sec
	)

	var allowed atomic.Int64
	var denied atomic.Int64

	var wg sync.WaitGroup
	wg.Add(goroutines)

	start := time.Now()

	for worker := 0; worker < goroutines; worker++ {
		go func(workerID int) {
			defer wg.Done()
			userID := fmt.Sprintf(
				"user-%d",
				workerID%numberOfUsers,
			)

			for i := 0; i < requestsPerWorker; i++ {
				ok, err := rl.Allow(userID)

				if err != nil {
					t.Errorf("Allow() error: %v", err)
					return
				}

				if ok {
					allowed.Add(1)
				} else {
					denied.Add(1)
				}
			}
		}(worker)
	}

	wg.Wait()

	duration := time.Since(start)

	totalRequests := int64(goroutines * requestsPerWorker)
	rps := float64(totalRequests) / duration.Seconds()

	t.Logf("===================================")
	t.Logf("HIGH LOAD TEST RESULTS")
	t.Logf("===================================")
	t.Logf("Total requests : %d", totalRequests)
	t.Logf("Allowed        : %d", allowed.Load())
	t.Logf("Denied         : %d", denied.Load())
	t.Logf("Users          : %d", numberOfUsers)
	t.Logf("Goroutines     : %d", goroutines)
	t.Logf("Duration       : %s", duration)
	t.Logf("Throughput     : %.0f requests/sec", rps)
	t.Logf("===================================")
}

func BenchmarkTokenBucketSingleUser(b *testing.B) {
	rl := NewTokenBucketRateLimiter(
		1_000_000_000,
		1_000_000,
	)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = rl.Allow("same-user")
		}
	})
}

func BenchmarkTokenBucketManyUsers(b *testing.B) {
	rl := NewTokenBucketRateLimiter(
		1_000_000_000,
		1_000_000,
	)

	const numberOfUsers = 1_000

	// Pre-create users so string formatting isn't part
	// of the measured benchmark.
	users := make([]string, numberOfUsers)

	for i := range users {
		users[i] = fmt.Sprintf("user-%d", i)
	}

	var counter atomic.Uint64

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := counter.Add(1)

			userID := users[id%uint64(len(users))]

			_, _ = rl.Allow(userID)
		}
	})
}