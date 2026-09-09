package main

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func assertInvariant(t *testing.T, pool *Pool) {
	t.Helper()

	pool.mu.Lock()
	defer pool.mu.Unlock()

	if pool.totalConnections < 0 {
		t.Fatalf(
			"totalConnections became negative: %d",
			pool.totalConnections,
		)
	}

	if pool.totalConnections > pool.maxConnections {
		t.Fatalf(
			"maxConnections violated: total=%d max=%d",
			pool.totalConnections,
			pool.maxConnections,
		)
	}

	if len(pool.connections) != pool.totalConnections {
		t.Fatalf(
			"registry invariant broken: registry=%d total=%d",
			len(pool.connections),
			pool.totalConnections,
		)
	}

	if len(pool.idleConnChan) > pool.idleConnections {
		t.Fatalf(
			"idle limit violated: idle=%d maxIdle=%d",
			len(pool.idleConnChan),
			pool.idleConnections,
		)
	}
}

func testPool(
	t *testing.T,
	maxConnections int,
	idleConnections int,
	idleTimeout time.Duration,
) *Pool {
	t.Helper()

	cp := NewConnectionPool(
		maxConnections,
		idleTimeout,
		idleConnections,
	)

	pool, ok := cp.(*Pool)
	if !ok {
		t.Fatal("NewConnectionPool did not return *Pool")
	}

	return pool
}

// ============================================================
// CHAOS TEST 1 — ABSOLUTE SATURATION
// ============================================================

func TestChaos1_AbsoluteSaturation(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())

	const (
		maxConnections  = 32
		idleConnections = 16

		workers             = 10000
		operationsPerWorker = 500
		failurePercent      = 7
	)

	pool := testPool(
		t,
		maxConnections,
		idleConnections,
		30*time.Second,
	)

	var wg sync.WaitGroup

	var success atomic.Int64
	var acquireFailures atomic.Int64
	var poisoned atomic.Int64

	var active atomic.Int64
	var peak atomic.Int64

	wg.Add(workers)

	start := time.Now()

	for worker := 0; worker < workers; worker++ {
		go func(workerID int) {
			defer wg.Done()

			rng := rand.New(
				rand.NewSource(
					time.Now().UnixNano() +
						int64(workerID)*7919,
				),
			)

			for i := 0; i < operationsPerWorker; i++ {
				ctx, cancel := context.WithTimeout(
					context.Background(),
					20*time.Second,
				)

				conn, err := pool.Acquire(ctx)

				cancel()

				if err != nil {
					acquireFailures.Add(1)
					continue
				}

				current := active.Add(1)

				for {
					oldPeak := peak.Load()

					if current <= oldPeak {
						break
					}

					if peak.CompareAndSwap(
						oldPeak,
						current,
					) {
						break
					}
				}

				// IMPORTANT:
				// Hold the connection long enough to FORCE saturation.
				//
				// This makes thousands of goroutines queue behind
				// maxConnections.
				time.Sleep(
					time.Duration(
						rng.Intn(1500)+500,
					) * time.Microsecond,
				)

				if rng.Intn(100) < failurePercent {
					pool.mu.Lock()
					conn.Health = StatusUnhealthy
					pool.mu.Unlock()

					poisoned.Add(1)
				}

				active.Add(-1)
				success.Add(1)

				if err := pool.Release(conn); err != nil {
					t.Errorf(
						"worker=%d Release failed: %v",
						workerID,
						err,
					)
					return
				}
			}
		}(worker)
	}

	wg.Wait()

	duration := time.Since(start)

	assertInvariant(t, pool)

	if peak.Load() > maxConnections {
		t.Fatalf(
			"connection limit violated: peak=%d max=%d",
			peak.Load(),
			maxConnections,
		)
	}

	fmt.Println()
	fmt.Println("[CHAOS 1] ABSOLUTE SATURATION")
	fmt.Println("────────────────────────────────────────────────────────────")
	fmt.Printf("CPU cores             : %d\n", runtime.NumCPU())
	fmt.Printf("GOMAXPROCS            : %d\n", runtime.GOMAXPROCS(0))
	fmt.Printf("Goroutines            : %d\n", workers)
	fmt.Printf("Operations requested  : %d\n", workers*operationsPerWorker)
	fmt.Printf("Successful operations : %d\n", success.Load())
	fmt.Printf("Acquire failures      : %d\n", acquireFailures.Load())
	fmt.Printf("Connections poisoned  : %d\n", poisoned.Load())
	fmt.Printf("Peak borrowed         : %d\n", peak.Load())
	fmt.Printf("Maximum allowed       : %d\n", maxConnections)
	fmt.Printf("Duration              : %v\n", duration)

	if peak.Load() != maxConnections {
		t.Fatalf(
			"pool never fully saturated: peak=%d expected=%d",
			peak.Load(),
			maxConnections,
		)
	}

	pool.Shutdown()

	fmt.Println("RESULT                : PASS")
}

// ============================================================
// CHAOS TEST 2 — DIFFERENT CONTEXT PER GOROUTINE
// ============================================================

func TestChaos2_PerGoroutineContextTimeouts(t *testing.T) {
	const (
		maxConnections = 16
		workers        = 10000
	)

	pool := testPool(
		t,
		maxConnections,
		8,
		time.Minute,
	)

	var wg sync.WaitGroup

	var acquired atomic.Int64
	var timedOut atomic.Int64
	var otherErrors atomic.Int64

	wg.Add(workers)

	startGate := make(chan struct{})

	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()

			rng := rand.New(
				rand.NewSource(
					time.Now().UnixNano() +
						int64(id)*3571,
				),
			)

			<-startGate

			// Every goroutine gets its OWN deadline.
			//
			// Anywhere from 1ms to 200ms.
			timeout :=
				time.Duration(
					rng.Intn(200)+1,
				) * time.Millisecond

			ctx, cancel := context.WithTimeout(
				context.Background(),
				timeout,
			)
			defer cancel()

			conn, err := pool.Acquire(ctx)

			if err != nil {
				if ctx.Err() != nil {
					timedOut.Add(1)
				} else {
					otherErrors.Add(1)
				}

				return
			}

			acquired.Add(1)

			// Hold long enough that many short contexts expire.
			time.Sleep(
				time.Duration(
					rng.Intn(5)+2,
				) * time.Millisecond,
			)

			if err := pool.Release(conn); err != nil {
				t.Errorf(
					"worker=%d release failed: %v",
					id,
					err,
				)
			}
		}(i)
	}

	// Release all goroutines at exactly the same time.
	close(startGate)

	wg.Wait()

	assertInvariant(t, pool)

	fmt.Println()
	fmt.Println("[CHAOS 2] PER-GOROUTINE CONTEXTS")
	fmt.Println("────────────────────────────────────────────────────────────")
	fmt.Printf("Goroutines            : %d\n", workers)
	fmt.Printf("Acquired              : %d\n", acquired.Load())
	fmt.Printf("Context expirations   : %d\n", timedOut.Load())
	fmt.Printf("Other errors          : %d\n", otherErrors.Load())

	if acquired.Load() == 0 {
		t.Fatal("nobody acquired a connection")
	}

	if timedOut.Load() == 0 {
		t.Fatal(
			"no goroutine timed out; context contention was not strong enough",
		)
	}

	if acquired.Load()+timedOut.Load()+otherErrors.Load() != workers {
		t.Fatalf(
			"result mismatch: total=%d workers=%d",
			acquired.Load()+
				timedOut.Load()+
				otherErrors.Load(),
			workers,
		)
	}

	pool.Shutdown()

	fmt.Println("RESULT                : PASS")
}

// ============================================================
// CHAOS TEST 3 — RANDOM SHUTDOWN UNDER FULL LOAD
// ============================================================

func TestChaos3_RandomShutdownUnderLoad(t *testing.T) {
	const (
		maxConnections  = 32
		idleConnections = 16

		workers = 15000
	)

	pool := testPool(
		t,
		maxConnections,
		idleConnections,
		time.Minute,
	)

	var wg sync.WaitGroup

	var acquireSuccess atomic.Int64
	var acquireErrors atomic.Int64

	var releaseSuccess atomic.Int64
	var releaseErrors atomic.Int64

	var startedBeforeShutdown atomic.Int64

	var shutdownStarted atomic.Bool

	startGate := make(chan struct{})

	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()

			rng := rand.New(
				rand.NewSource(
					time.Now().UnixNano() +
						int64(id)*1237,
				),
			)

			<-startGate

			if !shutdownStarted.Load() {
				startedBeforeShutdown.Add(1)
			}

			// Every goroutine also has its own context.
			timeout :=
				time.Duration(
					rng.Intn(1000)+50,
				) * time.Millisecond

			ctx, cancel := context.WithTimeout(
				context.Background(),
				timeout,
			)
			defer cancel()

			conn, err := pool.Acquire(ctx)

			if err != nil {
				acquireErrors.Add(1)
				return
			}

			acquireSuccess.Add(1)

			// Simulate work.
			time.Sleep(
				time.Duration(
					rng.Intn(5)+1,
				) * time.Millisecond,
			)

			// Some connections fail while being used.
			if rng.Intn(100) < 10 {
				pool.mu.Lock()
				conn.Health = StatusUnhealthy
				pool.mu.Unlock()
			}

			err = pool.Release(conn)

			if err != nil {
				releaseErrors.Add(1)
				return
			}

			releaseSuccess.Add(1)
		}(i)
	}

	close(startGate)

	// Random shutdown injection.
	rng := rand.New(
		rand.NewSource(time.Now().UnixNano()),
	)

	shutdownDelay :=
		time.Duration(
			rng.Intn(50)+5,
		) * time.Millisecond

	time.Sleep(shutdownDelay)

	shutdownStarted.Store(true)

	shutdownDone := make(chan struct{})

	go func() {
		pool.Shutdown()
		close(shutdownDone)
	}()

	wg.Wait()

	select {
	case <-shutdownDone:
	case <-time.After(10 * time.Second):
		t.Fatal(
			"Shutdown did not finish after workers completed",
		)
	}

	fmt.Println()
	fmt.Println("[CHAOS 3] RANDOM SHUTDOWN UNDER LOAD")
	fmt.Println("────────────────────────────────────────────────────────────")
	fmt.Printf("Workers               : %d\n", workers)
	fmt.Printf("Shutdown delay        : %v\n", shutdownDelay)
	fmt.Printf(
		"Started pre-shutdown  : %d\n",
		startedBeforeShutdown.Load(),
	)
	fmt.Printf(
		"Acquire successes     : %d\n",
		acquireSuccess.Load(),
	)
	fmt.Printf(
		"Acquire errors        : %d\n",
		acquireErrors.Load(),
	)
	fmt.Printf(
		"Release successes     : %d\n",
		releaseSuccess.Load(),
	)
	fmt.Printf(
		"Release errors        : %d\n",
		releaseErrors.Load(),
	)

	if acquireSuccess.Load()+acquireErrors.Load() != workers {
		t.Fatalf(
			"workers disappeared: success=%d errors=%d expected=%d",
			acquireSuccess.Load(),
			acquireErrors.Load(),
			workers,
		)
	}

	// Everyone who successfully acquired must have attempted Release.
	if releaseSuccess.Load()+releaseErrors.Load() !=
		acquireSuccess.Load() {

		t.Fatalf(
			"borrowed connection accounting mismatch: acquired=%d releases=%d",
			acquireSuccess.Load(),
			releaseSuccess.Load()+
				releaseErrors.Load(),
		)
	}

	pool.mu.Lock()

	total := pool.totalConnections
	registry := len(pool.connections)
	closed := pool.closed

	pool.mu.Unlock()

	fmt.Printf("Pool closed           : %v\n", closed)
	fmt.Printf("Connections remaining : %d\n", total)
	fmt.Printf("Registry remaining    : %d\n", registry)

	if !closed {
		t.Fatal("pool should be closed")
	}

	if total != 0 {
		t.Fatalf(
			"connections leaked after shutdown: %d",
			total,
		)
	}

	if registry != 0 {
		t.Fatalf(
			"registry leaked after shutdown: %d",
			registry,
		)
	}

	// Final proof: no Acquire after shutdown.
	conn, err := pool.Acquire(context.Background())

	if err == nil {
		t.Fatalf(
			"Acquire succeeded after shutdown: %v",
			conn,
		)
	}

	fmt.Println("RESULT                : PASS")
}
