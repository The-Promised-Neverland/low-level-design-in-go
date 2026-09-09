package main

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestLRUHeavyLoad(t *testing.T) {
	const (
		capacity   = 100_000
		goroutines = 1000
		opsEach    = 20_000
	)

	cache := NewLRUCache(capacity)

	var wg sync.WaitGroup
	start := time.Now()

	for workerID := 0; workerID < goroutines; workerID++ {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			for i := 0; i < opsEach; i++ {
				key := fmt.Sprintf(
					"worker-%d-key-%d",
					id,
					i%200_000,
				)

				// ~33% reads, ~67% writes
				if i%3 == 0 {
					cache.Get(key)
				} else {
					cache.Put(key, "value")
				}
			}
		}(workerID)
	}

	wg.Wait()

	duration := time.Since(start)
	totalOps := int64(goroutines * opsEach)
	opsPerSecond := float64(totalOps) / duration.Seconds()

	finalSize := len(cache.nodeCache)

	if finalSize > capacity {
		t.Fatalf(
			"cache exceeded capacity: got %d, max %d",
			finalSize,
			capacity,
		)
	}

	fmt.Printf(
		"[PASS] HeavyLoad | ops=%d | duration=%v | ops/sec=%.0f | finalSize=%d\n",
		totalOps,
		duration,
		opsPerSecond,
		finalSize,
	)
}

func TestLRUHeavyConcurrentWrites(t *testing.T) {
	const (
		capacity   = 100_000
		goroutines = 1000
		putsEach   = 20_000
	)

	cache := NewLRUCache(capacity)

	var wg sync.WaitGroup
	start := time.Now()

	for workerID := 0; workerID < goroutines; workerID++ {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			for i := 0; i < putsEach; i++ {
				key := fmt.Sprintf(
					"writer-%d-key-%d",
					id,
					i,
				)

				cache.Put(key, "value")
			}
		}(workerID)
	}

	wg.Wait()

	duration := time.Since(start)
	totalOps := int64(goroutines * putsEach)
	opsPerSecond := float64(totalOps) / duration.Seconds()

	finalSize := len(cache.nodeCache)

	if finalSize != capacity {
		t.Fatalf(
			"expected final cache size %d, got %d",
			capacity,
			finalSize,
		)
	}

	fmt.Printf(
		"[PASS] HeavyConcurrentWrites | ops=%d | duration=%v | ops/sec=%.0f | finalSize=%d\n",
		totalOps,
		duration,
		opsPerSecond,
		finalSize,
	)
}

func TestLRUHeavyContentionSameKeys(t *testing.T) {
	const (
		capacity   = 1000
		goroutines = 2000
		opsEach    = 10_000
		keySpace   = 100
	)

	cache := NewLRUCache(capacity)

	for i := 0; i < keySpace; i++ {
		cache.Put(
			fmt.Sprintf("hot-key-%d", i),
			"value",
		)
	}

	var wg sync.WaitGroup
	start := time.Now()

	for workerID := 0; workerID < goroutines; workerID++ {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			for i := 0; i < opsEach; i++ {
				key := fmt.Sprintf(
					"hot-key-%d",
					(i+id)%keySpace,
				)

				if i%2 == 0 {
					cache.Get(key)
				} else {
					cache.Put(
						key,
						fmt.Sprintf("value-%d-%d", id, i),
					)
				}
			}
		}(workerID)
	}

	wg.Wait()

	duration := time.Since(start)
	totalOps := int64(goroutines * opsEach)
	opsPerSecond := float64(totalOps) / duration.Seconds()

	finalSize := len(cache.nodeCache)

	if finalSize > capacity {
		t.Fatalf(
			"cache exceeded capacity: got %d, max %d",
			finalSize,
			capacity,
		)
	}

	fmt.Printf(
		"[PASS] HeavyContentionSameKeys | ops=%d | duration=%v | ops/sec=%.0f | finalSize=%d\n",
		totalOps,
		duration,
		opsPerSecond,
		finalSize,
	)
}

func TestLRULinkedListIntegrityAfterHeavyLoad(t *testing.T) {
	const (
		capacity = 10_000
		workers  = 200
		opsEach  = 5000
	)

	cache := NewLRUCache(capacity)

	var wg sync.WaitGroup
	start := time.Now()

	for workerID := 0; workerID < workers; workerID++ {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			for i := 0; i < opsEach; i++ {
				key := fmt.Sprintf(
					"worker-%d-key-%d",
					id,
					i,
				)

				cache.Put(key, "value")

				if i%5 == 0 {
					cache.Get(key)
				}
			}
		}(workerID)
	}

	wg.Wait()

	duration := time.Since(start)

	cache.mu.Lock()
	defer cache.mu.Unlock()

	visited := make(map[*LRUNode]bool)

	current := cache.MFUNode
	count := 0

	for current != nil {
		if visited[current] {
			t.Fatalf(
				"cycle detected in linked list at key %s",
				current.key,
			)
		}

		visited[current] = true
		count++

		if current.nextBlock != nil &&
			current.nextBlock.prevBlock != current {
			t.Fatalf(
				"broken prev pointer at key %s",
				current.key,
			)
		}

		if current.prevBlock != nil &&
			current.prevBlock.nextBlock != current {
			t.Fatalf(
				"broken next pointer at key %s",
				current.key,
			)
		}

		current = current.nextBlock
	}

	mapSize := len(cache.nodeCache)

	if count != mapSize {
		t.Fatalf(
			"linked list has %d nodes but map has %d entries",
			count,
			mapSize,
		)
	}

	if count > cache.capacity {
		t.Fatalf(
			"linked list exceeded capacity: %d > %d",
			count,
			cache.capacity,
		)
	}

	if count > 0 {
		if cache.MFUNode == nil {
			t.Fatal("MFUNode is nil even though cache is not empty")
		}

		if cache.LRUNode == nil {
			t.Fatal("LRUNode is nil even though cache is not empty")
		}
	}

	if cache.MFUNode != nil && cache.MFUNode.prevBlock != nil {
		t.Fatal("MFUNode.prevBlock should be nil")
	}

	if cache.LRUNode != nil && cache.LRUNode.nextBlock != nil {
		t.Fatal("LRUNode.nextBlock should be nil")
	}

	fmt.Printf(
		"[PASS] LinkedListIntegrity | duration=%v | nodes=%d | mapSize=%d | capacity=%d\n",
		duration,
		count,
		mapSize,
		cache.capacity,
	)
}
