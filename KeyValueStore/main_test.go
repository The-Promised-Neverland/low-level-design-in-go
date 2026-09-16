package main

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ============================================================
// Helpers
// ============================================================

func newTestStore(t *testing.T) *KeyValueStore {
	t.Helper()

	s := NewKeyValueStore()
	kv := s.(*KeyValueStore)

	t.Cleanup(func() {
		kv.Shutdown()
	})

	return kv
}

func assertInternalConsistency(t *testing.T, kv *KeyValueStore) {
	t.Helper()

	kv.mu.RLock()
	defer kv.mu.RUnlock()

	// Every expiryIndex entry must:
	// 1. exist in store
	// 2. be expiring
	// 3. point to a valid heap position
	// 4. point to itself in the heap
	for key, item := range kv.expiryIndex {
		entry, exists := kv.store[key]
		if !exists {
			t.Fatalf("expiryIndex contains %q but store does not", key)
		}

		if !entry.Expires {
			t.Fatalf("expiryIndex contains non-expiring key %q", key)
		}

		if item.Position < 0 || item.Position >= kv.expiryHeap.Len() {
			t.Fatalf(
				"invalid heap position for %q: %d",
				key,
				item.Position,
			)
		}

		if kv.expiryHeap.store[item.Position] != item {
			t.Fatalf(
				"expiryIndex item for %q does not match heap position %d",
				key,
				item.Position,
			)
		}

		if item.Key != key {
			t.Fatalf(
				"expiryIndex key mismatch: map key=%q item key=%q",
				key,
				item.Key,
			)
		}
	}

	// Every heap item must exist in expiryIndex.
	for i, item := range kv.expiryHeap.store {
		if item == nil {
			t.Fatalf("nil heap item at position %d", i)
		}

		if item.Position != i {
			t.Fatalf(
				"heap position mismatch for %q: Position=%d actual=%d",
				item.Key,
				item.Position,
				i,
			)
		}

		indexedItem, exists := kv.expiryIndex[item.Key]
		if !exists {
			t.Fatalf(
				"heap contains %q but expiryIndex does not",
				item.Key,
			)
		}

		if indexedItem != item {
			t.Fatalf(
				"heap/index pointer mismatch for %q",
				item.Key,
			)
		}
	}

	if kv.expiryHeap.Len() != len(kv.expiryIndex) {
		t.Fatalf(
			"heap/index size mismatch: heap=%d index=%d",
			kv.expiryHeap.Len(),
			len(kv.expiryIndex),
		)
	}

	// Verify min-heap property.
	for i := 0; i < kv.expiryHeap.Len(); i++ {
		left := 2*i + 1
		right := 2*i + 2

		if left < kv.expiryHeap.Len() &&
			kv.expiryHeap.Less(left, i) {
			t.Fatalf(
				"heap violation: child %d expires before parent %d",
				left,
				i,
			)
		}

		if right < kv.expiryHeap.Len() &&
			kv.expiryHeap.Less(right, i) {
			t.Fatalf(
				"heap violation: child %d expires before parent %d",
				right,
				i,
			)
		}
	}
}

// ============================================================
// Basic behavior
// ============================================================

func TestSetGet(t *testing.T) {
	kv := newTestStore(t)

	if err := kv.Set("name", "Abhijit", 0); err != nil {
		t.Fatal(err)
	}

	value, ok := kv.Get("name")

	if !ok {
		t.Fatal("expected key to exist")
	}

	if value != "Abhijit" {
		t.Fatalf("expected Abhijit, got %v", value)
	}
}

func TestGetMissingKey(t *testing.T) {
	kv := newTestStore(t)

	value, ok := kv.Get("missing")

	if ok {
		t.Fatal("missing key unexpectedly exists")
	}

	if value != nil {
		t.Fatalf("expected nil, got %v", value)
	}
}

func TestSetOverwriteValue(t *testing.T) {
	kv := newTestStore(t)

	_ = kv.Set("x", "old", 0)
	_ = kv.Set("x", "new", 0)

	value, ok := kv.Get("x")

	if !ok || value != "new" {
		t.Fatalf("expected new, got %v, exists=%v", value, ok)
	}
}

func TestDifferentValueTypes(t *testing.T) {
	kv := newTestStore(t)

	_ = kv.Set("string", "hello", 0)
	_ = kv.Set("int", 100, 0)
	_ = kv.Set("bool", true, 0)

	if v, _ := kv.Get("string"); v != "hello" {
		t.Fatalf("unexpected string: %v", v)
	}

	if v, _ := kv.Get("int"); v != 100 {
		t.Fatalf("unexpected int: %v", v)
	}

	if v, _ := kv.Get("bool"); v != true {
		t.Fatalf("unexpected bool: %v", v)
	}
}

func TestEmptyKey(t *testing.T) {
	kv := newTestStore(t)

	if err := kv.Set("", "value", 0); err != nil {
		t.Fatal(err)
	}

	value, ok := kv.Get("")

	if !ok || value != "value" {
		t.Fatalf("empty key failed")
	}
}

func TestNilValue(t *testing.T) {
	kv := newTestStore(t)

	if err := kv.Set("nil", nil, 0); err != nil {
		t.Fatal(err)
	}

	value, ok := kv.Get("nil")

	if !ok {
		t.Fatal("key containing nil should still exist")
	}

	if value != nil {
		t.Fatalf("expected nil value, got %v", value)
	}
}

// ============================================================
// TTL validation
// ============================================================

func TestNegativeTTL(t *testing.T) {
	kv := newTestStore(t)

	err := kv.Set("x", "value", -time.Second)

	if err == nil {
		t.Fatal("negative TTL should fail")
	}

	if kv.Exists("x") {
		t.Fatal("failed Set should not create key")
	}
}

func TestZeroTTLMeansNoExpiration(t *testing.T) {
	kv := newTestStore(t)

	_ = kv.Set("x", "value", 0)

	time.Sleep(50 * time.Millisecond)

	value, ok := kv.Get("x")

	if !ok || value != "value" {
		t.Fatal("TTL=0 key unexpectedly expired")
	}
}

// ============================================================
// TTL expiration
// ============================================================

func TestTTLExpires(t *testing.T) {
	kv := newTestStore(t)

	_ = kv.Set("x", "value", 30*time.Millisecond)

	if !kv.Exists("x") {
		t.Fatal("key should initially exist")
	}

	time.Sleep(70 * time.Millisecond)

	if kv.Exists("x") {
		t.Fatal("key should have expired")
	}
}

func TestGetExpiredKey(t *testing.T) {
	kv := newTestStore(t)

	_ = kv.Set("x", "value", 20*time.Millisecond)

	time.Sleep(50 * time.Millisecond)

	value, ok := kv.Get("x")

	if ok {
		t.Fatal("expired key returned as existing")
	}

	if value != nil {
		t.Fatalf("expected nil, got %v", value)
	}
}

// ============================================================
// TTL mutation
// ============================================================

func TestExtendTTL(t *testing.T) {
	kv := newTestStore(t)

	_ = kv.Set("x", "v1", 50*time.Millisecond)

	time.Sleep(20 * time.Millisecond)

	_ = kv.Set("x", "v2", 200*time.Millisecond)

	time.Sleep(70 * time.Millisecond)

	value, ok := kv.Get("x")

	if !ok {
		t.Fatal("key expired according to old TTL")
	}

	if value != "v2" {
		t.Fatalf("expected v2, got %v", value)
	}

	time.Sleep(170 * time.Millisecond)

	if kv.Exists("x") {
		t.Fatal("key did not expire according to extended TTL")
	}
}

func TestShortenTTL(t *testing.T) {
	kv := newTestStore(t)

	_ = kv.Set("x", "value", time.Hour)

	_ = kv.Set("x", "value", 30*time.Millisecond)

	time.Sleep(70 * time.Millisecond)

	if kv.Exists("x") {
		t.Fatal("shortened TTL was not respected")
	}
}

func TestExpiringToPermanent(t *testing.T) {
	kv := newTestStore(t)

	_ = kv.Set("x", "temporary", 30*time.Millisecond)

	_ = kv.Set("x", "permanent", 0)

	time.Sleep(70 * time.Millisecond)

	value, ok := kv.Get("x")

	if !ok {
		t.Fatal("key should have become permanent")
	}

	if value != "permanent" {
		t.Fatalf("expected permanent, got %v", value)
	}

	assertInternalConsistency(t, kv)
}

func TestPermanentToExpiring(t *testing.T) {
	kv := newTestStore(t)

	_ = kv.Set("x", "permanent", 0)

	_ = kv.Set("x", "temporary", 30*time.Millisecond)

	time.Sleep(70 * time.Millisecond)

	if kv.Exists("x") {
		t.Fatal("key should have expired")
	}

	assertInternalConsistency(t, kv)
}

// ============================================================
// Delete
// ============================================================

func TestDeletePermanentKey(t *testing.T) {
	kv := newTestStore(t)

	_ = kv.Set("x", "value", 0)

	if !kv.Delete("x") {
		t.Fatal("Delete should return true")
	}

	if kv.Exists("x") {
		t.Fatal("deleted key still exists")
	}
}

func TestDeleteExpiringKey(t *testing.T) {
	kv := newTestStore(t)

	_ = kv.Set("x", "value", time.Hour)

	if !kv.Delete("x") {
		t.Fatal("Delete failed")
	}

	assertInternalConsistency(t, kv)

	kv.mu.RLock()
	defer kv.mu.RUnlock()

	if kv.expiryHeap.Len() != 0 {
		t.Fatalf("heap should be empty, got %d", kv.expiryHeap.Len())
	}

	if len(kv.expiryIndex) != 0 {
		t.Fatalf("expiryIndex should be empty")
	}
}

func TestDeleteMissingKey(t *testing.T) {
	kv := newTestStore(t)

	if kv.Delete("missing") {
		t.Fatal("Delete should return false for missing key")
	}
}

// ============================================================
// Heap behavior
// ============================================================

func TestHeapOrdersByEarliestExpiry(t *testing.T) {
	kv := newTestStore(t)

	_ = kv.Set("hour", 1, time.Hour)
	_ = kv.Set("minute", 2, time.Minute)
	_ = kv.Set("half-hour", 3, 30*time.Minute)

	kv.mu.RLock()
	root := kv.expiryHeap.Peek()
	kv.mu.RUnlock()

	if root == nil {
		t.Fatal("heap unexpectedly empty")
	}

	if root.Key != "minute" {
		t.Fatalf("expected minute at root, got %s", root.Key)
	}

	assertInternalConsistency(t, kv)
}

func TestRepeatedTTLUpdateDoesNotDuplicateHeapItem(t *testing.T) {
	kv := newTestStore(t)

	for i := 1; i <= 10_000; i++ {
		ttl := time.Duration(i) * time.Second

		if err := kv.Set("x", i, ttl); err != nil {
			t.Fatal(err)
		}
	}

	kv.mu.RLock()

	heapSize := kv.expiryHeap.Len()
	indexSize := len(kv.expiryIndex)

	kv.mu.RUnlock()

	if heapSize != 1 {
		t.Fatalf("expected 1 heap item, got %d", heapSize)
	}

	if indexSize != 1 {
		t.Fatalf("expected 1 expiryIndex item, got %d", indexSize)
	}

	assertInternalConsistency(t, kv)
}

func TestHeapFixMovesItemTowardRoot(t *testing.T) {
	kv := newTestStore(t)

	_ = kv.Set("a", "a", time.Hour)
	_ = kv.Set("b", "b", 2*time.Hour)
	_ = kv.Set("c", "c", 3*time.Hour)

	// c suddenly becomes earliest.
	_ = kv.Set("c", "c", time.Second)

	kv.mu.RLock()
	root := kv.expiryHeap.Peek()
	kv.mu.RUnlock()

	if root == nil || root.Key != "c" {
		t.Fatalf("expected c at root, got %+v", root)
	}

	assertInternalConsistency(t, kv)
}

func TestHeapFixMovesItemAwayFromRoot(t *testing.T) {
	kv := newTestStore(t)

	_ = kv.Set("a", "a", time.Minute)
	_ = kv.Set("b", "b", 2*time.Minute)
	_ = kv.Set("c", "c", 3*time.Minute)

	// a used to be earliest. Push it far into future.
	_ = kv.Set("a", "a", 4*time.Hour)

	kv.mu.RLock()
	root := kv.expiryHeap.Peek()
	kv.mu.RUnlock()

	if root == nil || root.Key != "b" {
		t.Fatalf("expected b at root, got %+v", root)
	}

	assertInternalConsistency(t, kv)
}

// ============================================================
// Active cleanup
// ============================================================

func TestAutoCleanupPhysicallyDeletesKey(t *testing.T) {
	kv := newTestStore(t)

	_ = kv.Set("x", "value", 20*time.Millisecond)

	time.Sleep(70 * time.Millisecond)

	// Deliberately inspect store directly.
	// Do NOT call Get/Exists first because that would trigger lazy cleanup.

	kv.mu.RLock()
	_, storeExists := kv.store["x"]
	_, indexExists := kv.expiryIndex["x"]
	heapSize := kv.expiryHeap.Len()
	kv.mu.RUnlock()

	if storeExists {
		t.Fatal("auto cleanup did not remove store entry")
	}

	if indexExists {
		t.Fatal("auto cleanup did not remove expiryIndex entry")
	}

	if heapSize != 0 {
		t.Fatalf("expected empty heap, got %d", heapSize)
	}
}

func TestCleanerReactsToEarlierExpiry(t *testing.T) {
	kv := newTestStore(t)

	// Cleaner should initially sleep for this.
	_ = kv.Set("slow", "slow", time.Hour)

	time.Sleep(10 * time.Millisecond)

	// Cleaner must wake and recalculate.
	_ = kv.Set("fast", "fast", 30*time.Millisecond)

	time.Sleep(70 * time.Millisecond)

	if kv.Exists("fast") {
		t.Fatal("cleaner failed to react to earlier expiry")
	}

	if !kv.Exists("slow") {
		t.Fatal("slow key should still exist")
	}
}

func TestManyKeysExpire(t *testing.T) {
	kv := newTestStore(t)

	const count = 1000

	for i := 0; i < count; i++ {
		key := fmt.Sprintf("key-%d", i)
		_ = kv.Set(key, i, 30*time.Millisecond)
	}

	time.Sleep(150 * time.Millisecond)

	kv.mu.RLock()
	storeSize := len(kv.store)
	heapSize := kv.expiryHeap.Len()
	indexSize := len(kv.expiryIndex)
	kv.mu.RUnlock()

	if storeSize != 0 {
		t.Fatalf("expected empty store, got %d", storeSize)
	}

	if heapSize != 0 {
		t.Fatalf("expected empty heap, got %d", heapSize)
	}

	if indexSize != 0 {
		t.Fatalf("expected empty expiryIndex, got %d", indexSize)
	}
}

// ============================================================
// Shutdown
// ============================================================

func TestShutdown(t *testing.T) {
	kv := NewKeyValueStore()

	_ = kv.Set("x", "value", 0)

	kv.Shutdown()

	if err := kv.Set("new", "value", 0); err == nil {
		t.Fatal("Set should fail after shutdown")
	}

	if _, ok := kv.Get("x"); ok {
		t.Fatal("Get should fail after shutdown")
	}

	if kv.Exists("x") {
		t.Fatal("Exists should fail after shutdown")
	}

	if kv.Delete("x") {
		t.Fatal("Delete should fail after shutdown")
	}
}

func TestShutdownIsIdempotent(t *testing.T) {
	kv := NewKeyValueStore()

	kv.Shutdown()
	kv.Shutdown()
	kv.Shutdown()
}

func TestConcurrentShutdown(t *testing.T) {
	kv := NewKeyValueStore()

	const goroutines = 1000

	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			kv.Shutdown()
		}()
	}

	wg.Wait()
}

// ============================================================
// Concurrency
// ============================================================

func TestConcurrentDifferentKeys(t *testing.T) {
	kv := newTestStore(t)

	const workers = 1000

	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			key := fmt.Sprintf("key-%d", i)

			if err := kv.Set(key, i, time.Hour); err != nil {
				t.Errorf("Set(%s): %v", key, err)
			}

			value, ok := kv.Get(key)

			if !ok {
				t.Errorf("%s disappeared", key)
				return
			}

			if value != i {
				t.Errorf(
					"%s: expected %d, got %v",
					key,
					i,
					value,
				)
			}
		}(i)
	}

	wg.Wait()

	assertInternalConsistency(t, kv)
}

func TestConcurrentSameKey(t *testing.T) {
	kv := newTestStore(t)

	const workers = 5000

	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			ttl := time.Duration((i%100)+1) * time.Second

			if err := kv.Set("hot-key", i, ttl); err != nil {
				t.Errorf("Set failed: %v", err)
			}
		}(i)
	}

	wg.Wait()

	assertInternalConsistency(t, kv)

	kv.mu.RLock()
	heapSize := kv.expiryHeap.Len()
	indexSize := len(kv.expiryIndex)
	storeSize := len(kv.store)
	kv.mu.RUnlock()

	if heapSize != 1 {
		t.Fatalf("duplicate heap items: %d", heapSize)
	}

	if indexSize != 1 {
		t.Fatalf("expected one expiryIndex entry, got %d", indexSize)
	}

	if storeSize != 1 {
		t.Fatalf("expected one store entry, got %d", storeSize)
	}
}

func TestConcurrentReadWriteDelete(t *testing.T) {
	kv := newTestStore(t)

	const workers = 200
	const operations = 2000

	var wg sync.WaitGroup

	for worker := 0; worker < workers; worker++ {
		wg.Add(1)

		go func(worker int) {
			defer wg.Done()

			for i := 0; i < operations; i++ {
				key := fmt.Sprintf("key-%d", i%100)

				switch (worker + i) % 4 {
				case 0:
					_ = kv.Set(key, worker, time.Second)

				case 1:
					kv.Get(key)

				case 2:
					kv.Exists(key)

				case 3:
					kv.Delete(key)
				}
			}
		}(worker)
	}

	wg.Wait()

	assertInternalConsistency(t, kv)
}

// ============================================================
// Concurrent TTL transitions
// ============================================================

func TestConcurrentTTLTransitions(t *testing.T) {
	kv := newTestStore(t)

	const workers = 1000

	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			switch i % 3 {
			case 0:
				_ = kv.Set("x", i, 0)

			case 1:
				_ = kv.Set("x", i, time.Hour)

			case 2:
				_ = kv.Set("x", i, 2*time.Hour)
			}
		}(i)
	}

	wg.Wait()

	assertInternalConsistency(t, kv)

	kv.mu.RLock()
	defer kv.mu.RUnlock()

	if len(kv.store) != 1 {
		t.Fatalf("expected one store entry, got %d", len(kv.store))
	}

	if len(kv.expiryIndex) > 1 {
		t.Fatalf(
			"duplicate expiryIndex entries: %d",
			len(kv.expiryIndex),
		)
	}

	if kv.expiryHeap.Len() > 1 {
		t.Fatalf(
			"duplicate heap entries: %d",
			kv.expiryHeap.Len(),
		)
	}
}

// ============================================================
// TORTURE TEST
// ============================================================

func TestTorture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping torture test in short mode")
	}

	kv := newTestStore(t)

	const (
		workers       = 1000
		opsPerWorker  = 20_000
		numberOfKeys  = 1000
	)

	var (
		wg          sync.WaitGroup
		setCount    atomic.Int64
		getCount    atomic.Int64
		existsCount atomic.Int64
		deleteCount atomic.Int64
	)

	start := time.Now()

	for worker := 0; worker < workers; worker++ {
		wg.Add(1)

		go func(worker int) {
			defer wg.Done()

			// Each goroutine gets its own RNG.
			r := rand.New(
				rand.NewSource(int64(worker + 1)),
			)

			for i := 0; i < opsPerWorker; i++ {
				key := fmt.Sprintf(
					"key-%d",
					r.Intn(numberOfKeys),
				)

				operation := r.Intn(100)

				switch {
				// 35% Set
				case operation < 35:
					var ttl time.Duration

					switch r.Intn(4) {
					case 0:
						// Permanent
						ttl = 0

					case 1:
						// Very short TTL
						ttl = time.Duration(
							r.Intn(50)+1,
						) * time.Millisecond

					case 2:
						// Medium TTL
						ttl = time.Duration(
							r.Intn(500)+50,
						) * time.Millisecond

					default:
						// Long enough that it normally
						// survives the torture test.
						ttl = time.Minute
					}

					_ = kv.Set(
						key,
						fmt.Sprintf("%d-%d", worker, i),
						ttl,
					)

					setCount.Add(1)

				// 30% Get
				case operation < 65:
					kv.Get(key)
					getCount.Add(1)

				// 20% Exists
				case operation < 85:
					kv.Exists(key)
					existsCount.Add(1)

				// 15% Delete
				default:
					kv.Delete(key)
					deleteCount.Add(1)
				}

				// Occasionally hammer one single hot key.
				if i%100 == 0 {
					_ = kv.Set(
						"HOT-KEY",
						worker,
						time.Duration(r.Intn(1000)+1)*
							time.Millisecond,
					)
				}
			}
		}(worker)
	}

	wg.Wait()

	duration := time.Since(start)

	assertInternalConsistency(t, kv)

	total :=
		setCount.Load() +
			getCount.Load() +
			existsCount.Load() +
			deleteCount.Load()

	t.Logf("========================================")
	t.Logf("TORTURE TEST COMPLETE")
	t.Logf("========================================")
	t.Logf("Workers       : %d", workers)
	t.Logf("Operations    : %d", total)
	t.Logf("Sets          : %d", setCount.Load())
	t.Logf("Gets          : %d", getCount.Load())
	t.Logf("Exists        : %d", existsCount.Load())
	t.Logf("Deletes       : %d", deleteCount.Load())
	t.Logf("Duration      : %v", duration)
	t.Logf(
		"Throughput    : %.0f ops/sec",
		float64(total)/duration.Seconds(),
	)

	kv.mu.RLock()

	t.Logf("Store entries : %d", len(kv.store))
	t.Logf("Heap entries  : %d", kv.expiryHeap.Len())
	t.Logf("Expiry index  : %d", len(kv.expiryIndex))

	kv.mu.RUnlock()

	t.Logf("========================================")
}