package main

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	testTopics = 8
)

var topics = []string{
	"orders",
	"payments",
	"users",
	"notifications",
	"audit",
	"transactions",
	"settlements",
	"reports",
}

// ============================================================
// MAIN TEST SUITE
// ============================================================

func TestPubSubBroker(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())

	fmt.Println()
	fmt.Println("PUB/SUB BROKER TEST SUITE")
	fmt.Println("════════════════════════════════════════════════════════════")

	t.Run("01_BasicCorrectness", testBasicCorrectness)
	t.Run("02_TopicIsolation", testTopicIsolation)
	t.Run("03_SlowSubscriber", testSlowSubscriber)
	t.Run("04_UnsubscribeSafety", testUnsubscribeSafety)
	t.Run("05_ExtremePublishLoad", testExtremePublishLoad)
	t.Run("06_ConcurrentChurn", testConcurrentChurn)
	t.Run("07_PublishUnsubscribeRace", testPublishUnsubscribeRace)
	t.Run("08_ShutdownChaos", testShutdownChaos)

	fmt.Println()
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Println("ALL PUB/SUB BROKER TESTS PASSED")
	fmt.Println("════════════════════════════════════════════════════════════")
}

func printPass(name string) {
	fmt.Printf("[PASS] %s\n", name)
}

// ============================================================
// 1. BASIC CORRECTNESS
// ============================================================

func testBasicCorrectness(t *testing.T) {
	b := NewPubSubBroker()

	const subscriberCount = 100

	subs := make([]*Subscriber, 0, subscriberCount)

	for i := 0; i < subscriberCount; i++ {
		sub, err := b.Subscribe("orders")
		if err != nil {
			t.Fatal(err)
		}

		subs = append(subs, sub)
	}

	msg := Message{
		ID:   "MSG-001",
		Data: "Order Created",
	}

	if err := b.Publish("orders", msg); err != nil {
		t.Fatal(err)
	}

	for i, sub := range subs {
		select {
		case received := <-sub.Messages():
			if received.ID != msg.ID {
				t.Fatalf(
					"subscriber %d received incorrect message",
					i,
				)
			}

		case <-time.After(time.Second):
			t.Fatalf(
				"subscriber %d did not receive message",
				i,
			)
		}
	}

	b.Shutdown()

	printPass("Basic broadcast to 100 subscribers")
}

// ============================================================
// 2. TOPIC ISOLATION
// ============================================================

func testTopicIsolation(t *testing.T) {
	b := NewPubSubBroker()

	orderSub, _ := b.Subscribe("orders")
	paymentSub, _ := b.Subscribe("payments")

	err := b.Publish(
		"orders",
		Message{ID: "ORDER-1"},
	)

	if err != nil {
		t.Fatal(err)
	}

	select {
	case msg := <-orderSub.Messages():
		if msg.ID != "ORDER-1" {
			t.Fatal("wrong message received")

		}

	case <-time.After(time.Second):
		t.Fatal("orders subscriber received nothing")
	}

	select {
	case msg := <-paymentSub.Messages():
		t.Fatalf(
			"payments subscriber incorrectly received: %+v",
			msg,
		)

	default:
	}

	b.Shutdown()

	printPass("Topic isolation")
}

// ============================================================
// 3. SLOW SUBSCRIBER
//
// Fill one subscriber's entire 1000-message buffer.
// Publishing another message MUST NOT block.
// ============================================================

func testSlowSubscriber(t *testing.T) {
	b := NewPubSubBroker()

	slow, _ := b.Subscribe("orders")
	fast, _ := b.Subscribe("orders")

	// Keep fast subscriber drained.
	var fastConsumed atomic.Int64

	done := make(chan struct{})

	go func() {
		for range fast.Messages() {
			fastConsumed.Add(1)
		}

		close(done)
	}()

	// Slow subscriber NEVER reads.
	for i := 0; i < 1000; i++ {
		err := b.Publish(
			"orders",
			Message{
				ID: fmt.Sprintf("MSG-%d", i),
			},
		)

		if err != nil {
			t.Fatal(err)
		}
	}

	if len(slow.ch) != 1000 {
		t.Fatalf(
			"expected slow buffer = 1000, got %d",
			len(slow.ch),
		)
	}

	start := time.Now()

	err := b.Publish(
		"orders",
		Message{ID: "BUFFER-FULL"},
	)

	duration := time.Since(start)

	if err != nil {
		t.Fatal(err)
	}

	if duration > 100*time.Millisecond {
		t.Fatalf(
			"slow subscriber blocked Publish for %v",
			duration,
		)
	}

	// Fast subscriber should still be receiving.
	deadline := time.Now().Add(time.Second)

	for fastConsumed.Load() < 1001 {
		if time.Now().After(deadline) {
			t.Fatalf(
				"fast subscriber only received %d messages",
				fastConsumed.Load(),
			)
		}

		runtime.Gosched()
	}

	b.Shutdown()

	<-done

	printPass("Slow subscriber does not block fast subscriber")
}

// ============================================================
// 4. UNSUBSCRIBE SAFETY
// ============================================================

func testUnsubscribeSafety(t *testing.T) {
	b := NewPubSubBroker()

	sub, err := b.Subscribe("orders")
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 100; i++ {
		_ = b.Publish(
			"orders",
			Message{
				ID: fmt.Sprintf("MSG-%d", i),
			},
		)
	}

	if err := b.Unsubscribe("orders", sub); err != nil {
		t.Fatal(err)
	}

	// Closed channel still contains buffered messages.
	count := 0

	for range sub.Messages() {
		count++
	}

	if count != 100 {
		t.Fatalf(
			"expected 100 buffered messages, got %d",
			count,
		)
	}

	// Double unsubscribe must return an error, not panic.
	if err := b.Unsubscribe("orders", sub); err == nil {
		t.Fatal("expected double unsubscribe error")
	}

	b.Shutdown()

	printPass("Unsubscribe + buffered drain + double unsubscribe")
}

// ============================================================
// 5. EXTREME PURE PUBLISH LOAD
//
// IMPORTANT:
// There is NO shutdown during this workload.
//
// Therefore all 10,000,000 Publish() calls should execute.
//
// This is the actual raw-load benchmark.
// ============================================================

func testExtremePublishLoad(t *testing.T) {
	b := NewPubSubBroker()

	const (
		subscriberCount      = 100
		publisherWorkers     = 40_000
		messagesPerPublisher = 250
	)

	subs := make([]*Subscriber, 0, subscriberCount)

	for i := 0; i < subscriberCount; i++ {
		sub, err := b.Subscribe("load")

		if err != nil {
			t.Fatal(err)
		}

		subs = append(subs, sub)
	}

	// Drain all subscribers continuously.
	var consumed atomic.Int64

	var consumerWG sync.WaitGroup
	consumerWG.Add(len(subs))

	for _, sub := range subs {
		go func(s *Subscriber) {
			defer consumerWG.Done()

			for range s.Messages() {
				consumed.Add(1)
			}
		}(sub)
	}

	var successful atomic.Int64
	var failed atomic.Int64

	var active atomic.Int64
	var peak atomic.Int64

	startGate := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(publisherWorkers)

	for worker := 0; worker < publisherWorkers; worker++ {
		workerID := worker

		go func() {
			defer wg.Done()

			<-startGate

			current := active.Add(1)

			for {
				old := peak.Load()

				if current <= old ||
					peak.CompareAndSwap(old, current) {
					break
				}
			}

			defer active.Add(-1)

			for i := 0; i < messagesPerPublisher; i++ {
				err := b.Publish(
					"load",
					Message{
						ID: fmt.Sprintf(
							"P%d-M%d",
							workerID,
							i,
						),
					},
				)

				if err != nil {
					failed.Add(1)
					continue
				}

				successful.Add(1)

				if i%25 == 0 {
					runtime.Gosched()
				}
			}
		}()
	}

	totalPlanned :=
		int64(publisherWorkers) *
			int64(messagesPerPublisher)

	start := time.Now()

	close(startGate)

	wg.Wait()

	duration := time.Since(start)

	// Shutdown only AFTER publishers finished.
	b.Shutdown()

	consumerWG.Wait()

	if failed.Load() != 0 {
		t.Fatalf(
			"unexpected publish failures: %d",
			failed.Load(),
		)
	}

	if successful.Load() != totalPlanned {
		t.Fatalf(
			"expected %d publishes, got %d",
			totalPlanned,
			successful.Load(),
		)
	}

	fmt.Println()
	fmt.Println("EXTREME PURE PUBLISH LOAD")
	fmt.Println("────────────────────────────────────────────────────────────")
	fmt.Printf("CPU cores                 : %d\n", runtime.NumCPU())
	fmt.Printf("GOMAXPROCS                : %d\n", runtime.GOMAXPROCS(0))
	fmt.Printf("Publisher goroutines      : %d\n", publisherWorkers)
	fmt.Printf("Subscribers               : %d\n", subscriberCount)
	fmt.Printf("Messages / publisher      : %d\n", messagesPerPublisher)
	fmt.Printf("Publish calls planned     : %d\n", totalPlanned)
	fmt.Printf("Publish calls completed   : %d\n", successful.Load())
	fmt.Printf("Publish failures          : %d\n", failed.Load())
	fmt.Printf("Subscriber deliveries     : %d\n", consumed.Load())
	fmt.Printf("Peak publishers active    : %d\n", peak.Load())
	fmt.Printf("Duration                  : %v\n", duration)

	if duration > 0 {
		fmt.Printf(
			"Publish throughput        : %.0f calls/sec\n",
			float64(successful.Load())/
				duration.Seconds(),
		)
	}

	fmt.Println("RESULT                    : PASS")

	printPass("10M publish load")
}

// ============================================================
// 6. MASSIVE SUBSCRIBE / UNSUBSCRIBE CHURN
//
// No shutdown interference.
// Every worker must successfully subscribe and unsubscribe.
// ============================================================

func testConcurrentChurn(t *testing.T) {
	b := NewPubSubBroker()

	const workers = 100_000

	var subscribed atomic.Int64
	var unsubscribed atomic.Int64
	var failures atomic.Int64

	startGate := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(workers)

	start := time.Now()

	for i := 0; i < workers; i++ {
		workerID := i

		go func() {
			defer wg.Done()

			<-startGate

			topic := topics[workerID%len(topics)]

			sub, err := b.Subscribe(topic)

			if err != nil {
				failures.Add(1)
				return
			}

			subscribed.Add(1)

			runtime.Gosched()

			err = b.Unsubscribe(topic, sub)

			if err != nil {
				failures.Add(1)
				return
			}

			unsubscribed.Add(1)
		}()
	}

	close(startGate)

	wg.Wait()

	duration := time.Since(start)

	if failures.Load() != 0 {
		t.Fatalf(
			"churn failures: %d",
			failures.Load(),
		)
	}

	if subscribed.Load() != workers {
		t.Fatalf(
			"expected %d subscriptions, got %d",
			workers,
			subscribed.Load(),
		)
	}

	if unsubscribed.Load() != workers {
		t.Fatalf(
			"expected %d unsubscriptions, got %d",
			workers,
			unsubscribed.Load(),
		)
	}

	b.mu.RLock()
	remainingTopics := len(b.topics)
	b.mu.RUnlock()

	if remainingTopics != 0 {
		t.Fatalf(
			"%d topics remain after churn",
			remainingTopics,
		)
	}

	b.Shutdown()

	fmt.Println()
	fmt.Println("EXTREME SUBSCRIBE / UNSUBSCRIBE CHURN")
	fmt.Println("────────────────────────────────────────────────────────────")
	fmt.Printf("Concurrent workers        : %d\n", workers)
	fmt.Printf("Subscriptions             : %d\n", subscribed.Load())
	fmt.Printf("Unsubscriptions           : %d\n", unsubscribed.Load())
	fmt.Printf("Failures                  : %d\n", failures.Load())
	fmt.Printf("Topics remaining          : %d\n", remainingTopics)
	fmt.Printf("Duration                  : %v\n", duration)
	fmt.Println("RESULT                    : PASS")

	printPass("100K subscription churn")
}

// ============================================================
// 7. PUBLISH / UNSUBSCRIBE RACE
//
// Publishers continuously snapshot subscribers while thousands
// of subscribers are concurrently removed and closed.
//
// Main objective:
//      NO "send on closed channel"
//      NO concurrent map panic
//      NO deadlock
// ============================================================

func testPublishUnsubscribeRace(t *testing.T) {
	b := NewPubSubBroker()

	const (
		subscriberCount  = 20_000
		publisherWorkers = 10_000
		publishesEach    = 50
	)

	subs := make([]*Subscriber, 0, subscriberCount)

	for i := 0; i < subscriberCount; i++ {
		sub, err := b.Subscribe("race")

		if err != nil {
			t.Fatal(err)
		}

		subs = append(subs, sub)
	}

	startGate := make(chan struct{})

	var publisherWG sync.WaitGroup
	var unsubscribeWG sync.WaitGroup

	var publishes atomic.Int64
	var unsubscribeSuccess atomic.Int64
	var unsubscribeFailure atomic.Int64

	publisherWG.Add(publisherWorkers)

	for worker := 0; worker < publisherWorkers; worker++ {
		go func(id int) {
			defer publisherWG.Done()

			<-startGate

			for i := 0; i < publishesEach; i++ {
				if err := b.Publish(
					"race",
					Message{
						ID: fmt.Sprintf(
							"P%d-%d",
							id,
							i,
						),
					},
				); err == nil {
					publishes.Add(1)
				}

				if i%5 == 0 {
					runtime.Gosched()
				}
			}
		}(worker)
	}

	unsubscribeWG.Add(len(subs))

	for _, subscriber := range subs {
		sub := subscriber

		go func() {
			defer unsubscribeWG.Done()

			<-startGate

			runtime.Gosched()

			if err := b.Unsubscribe("race", sub); err != nil {
				unsubscribeFailure.Add(1)
			} else {
				unsubscribeSuccess.Add(1)
			}
		}()
	}

	start := time.Now()

	close(startGate)

	publisherWG.Wait()
	unsubscribeWG.Wait()

	duration := time.Since(start)

	/*
		Some Unsubscribe calls can legitimately fail.

		Why?

		The last subscriber removes the topic itself:

		    delete(b.topics, topic)

		Another Unsubscribe arriving afterwards sees:
		    "topic does not exist"

		That's consistent with our current API.
	*/

	b.Shutdown()

	fmt.Println()
	fmt.Println("PUBLISH / UNSUBSCRIBE RACE")
	fmt.Println("────────────────────────────────────────────────────────────")
	fmt.Printf("Subscribers               : %d\n", subscriberCount)
	fmt.Printf("Publisher goroutines      : %d\n", publisherWorkers)
	fmt.Printf("Successful publish calls  : %d\n", publishes.Load())
	fmt.Printf("Unsubscribe success       : %d\n", unsubscribeSuccess.Load())
	fmt.Printf("Unsubscribe errors        : %d\n", unsubscribeFailure.Load())
	fmt.Printf("Duration                  : %v\n", duration)
	fmt.Println("Panics                    : 0")
	fmt.Println("RESULT                    : PASS")

	printPass("Publish/unsubscribe race")
}

// ============================================================
// 8. SHUTDOWN CHAOS
//
// This test intentionally shuts the broker down while:
//   - publishers are running
//   - subscribers are being created
//   - subscribers are being removed
//   - consumers are reading
//   - 10,000 Shutdown() calls race
//
// Unlike the pure-load test, rejection is EXPECTED here.
// ============================================================

func testShutdownChaos(t *testing.T) {
	b := NewPubSubBroker()

	const (
		initialSubscribers = 10_000

		publisherWorkers = 30_000
		churnWorkers     = 30_000
		shutdownWorkers  = 10_000

		publishesEach = 250
	)

	initialSubs := make([]*Subscriber, 0, initialSubscribers)

	for i := 0; i < initialSubscribers; i++ {
		sub, err := b.Subscribe(
			topics[i%len(topics)],
		)

		if err != nil {
			t.Fatal(err)
		}

		initialSubs = append(initialSubs, sub)
	}

	// Consume only half.
	//
	// Other half are deliberately slow consumers.
	const activeConsumers = initialSubscribers / 2

	var consumed atomic.Int64

	var consumerWG sync.WaitGroup
	consumerWG.Add(activeConsumers)

	for i := 0; i < activeConsumers; i++ {
		sub := initialSubs[i]

		go func() {
			defer consumerWG.Done()

			for range sub.Messages() {
				consumed.Add(1)
			}
		}()
	}

	var publishSuccess atomic.Int64
	var publishRejected atomic.Int64

	var subscribeSuccess atomic.Int64
	var subscribeRejected atomic.Int64

	var unsubscribeSuccess atomic.Int64
	var unsubscribeFailed atomic.Int64

	var shutdownReturned atomic.Int64

	var activeWorkers atomic.Int64
	var peakWorkers atomic.Int64

	workerStart := func() {
		current := activeWorkers.Add(1)

		for {
			peak := peakWorkers.Load()

			if current <= peak {
				return
			}

			if peakWorkers.CompareAndSwap(
				peak,
				current,
			) {
				return
			}
		}
	}

	workerDone := func() {
		activeWorkers.Add(-1)
	}

	startGate := make(chan struct{})
	shutdownGate := make(chan struct{})

	var publisherWG sync.WaitGroup
	var churnWG sync.WaitGroup
	var shutdownWG sync.WaitGroup

	// --------------------------------------------------------
	// Publishers
	// --------------------------------------------------------

	publisherWG.Add(publisherWorkers)

	for worker := 0; worker < publisherWorkers; worker++ {
		workerID := worker

		go func() {
			defer publisherWG.Done()

			<-startGate

			workerStart()
			defer workerDone()

			for i := 0; i < publishesEach; i++ {
				err := b.Publish(
					topics[(workerID+i)%len(topics)],
					Message{
						ID: fmt.Sprintf(
							"P%d-M%d",
							workerID,
							i,
						),
					},
				)

				if err != nil {
					publishRejected.Add(1)
					return
				}

				publishSuccess.Add(1)

				if i%10 == 0 {
					runtime.Gosched()
				}
			}
		}()
	}

	// --------------------------------------------------------
	// Subscriber churn
	// --------------------------------------------------------

	churnWG.Add(churnWorkers)

	for worker := 0; worker < churnWorkers; worker++ {
		workerID := worker

		go func() {
			defer churnWG.Done()

			<-startGate

			workerStart()
			defer workerDone()

			topic := topics[workerID%len(topics)]

			sub, err := b.Subscribe(topic)

			if err != nil {
				subscribeRejected.Add(1)
				return
			}

			subscribeSuccess.Add(1)

			runtime.Gosched()

			err = b.Unsubscribe(
				topic,
				sub,
			)

			if err != nil {
				unsubscribeFailed.Add(1)
				return
			}

			unsubscribeSuccess.Add(1)
		}()
	}

	// --------------------------------------------------------
	// Shutdown army
	// --------------------------------------------------------

	shutdownWG.Add(shutdownWorkers)

	for i := 0; i < shutdownWorkers; i++ {
		go func() {
			defer shutdownWG.Done()

			<-shutdownGate

			workerStart()
			defer workerDone()

			b.Shutdown()

			shutdownReturned.Add(1)
		}()
	}

	goroutinesBefore := runtime.NumGoroutine()

	start := time.Now()

	close(startGate)

	/*
		Let actual broker work happen first.

		We intentionally do NOT use a fixed 100ms sleep here.

		Instead, wait until some meaningful number of successful
		operations has occurred before triggering shutdown.
	*/

	deadline := time.Now().Add(5 * time.Second)

	for {
		workDone :=
			publishSuccess.Load() >= 100_000 ||
				subscribeSuccess.Load() >= 5_000

		if workDone {
			break
		}

		if time.Now().After(deadline) {
			break
		}

		runtime.Gosched()
	}

	close(shutdownGate)

	publisherWG.Wait()
	churnWG.Wait()
	shutdownWG.Wait()
	consumerWG.Wait()

	duration := time.Since(start)

	// --------------------------------------------------------
	// Final invariants
	// --------------------------------------------------------

	b.mu.RLock()

	isShutdown := b.shutdown
	remainingTopics := len(b.topics)

	b.mu.RUnlock()

	if !isShutdown {
		t.Fatal("broker not marked shutdown")
	}

	if remainingTopics != 0 {
		t.Fatalf(
			"expected zero topics after shutdown, got %d",
			remainingTopics,
		)
	}

	if shutdownReturned.Load() != shutdownWorkers {
		t.Fatalf(
			"expected %d Shutdown calls to return, got %d",
			shutdownWorkers,
			shutdownReturned.Load(),
		)
	}

	if _, err := b.Subscribe("dead"); err == nil {
		t.Fatal("Subscribe succeeded after shutdown")
	}

	if err := b.Publish(
		"dead",
		Message{ID: "DEAD"},
	); err == nil {
		t.Fatal("Publish succeeded after shutdown")
	}

	// Shutdown remains idempotent.
	for i := 0; i < 1000; i++ {
		b.Shutdown()
	}

	runtime.GC()
	time.Sleep(200 * time.Millisecond)

	goroutinesAfter := runtime.NumGoroutine()

	fmt.Println()
	fmt.Println("EXTREME SHUTDOWN CHAOS")
	fmt.Println("────────────────────────────────────────────────────────────")
	fmt.Printf("CPU cores                 : %d\n", runtime.NumCPU())
	fmt.Printf("Initial subscribers       : %d\n", initialSubscribers)
	fmt.Printf("Active consumers          : %d\n", activeConsumers)
	fmt.Printf("Publisher goroutines      : %d\n", publisherWorkers)
	fmt.Printf("Churn goroutines          : %d\n", churnWorkers)
	fmt.Printf("Shutdown goroutines       : %d\n", shutdownWorkers)
	fmt.Printf("Peak active workers       : %d\n", peakWorkers.Load())

	fmt.Println()
	fmt.Println("OPERATIONS")
	fmt.Println("────────────────────────────────────────────────────────────")
	fmt.Printf("Successful publishes      : %d\n", publishSuccess.Load())
	fmt.Printf("Publishers rejected       : %d\n", publishRejected.Load())
	fmt.Printf("Messages consumed         : %d\n", consumed.Load())
	fmt.Printf("Subscribe success         : %d\n", subscribeSuccess.Load())
	fmt.Printf("Subscribe rejected        : %d\n", subscribeRejected.Load())
	fmt.Printf("Unsubscribe success       : %d\n", unsubscribeSuccess.Load())
	fmt.Printf("Unsubscribe errors        : %d\n", unsubscribeFailed.Load())

	fmt.Println()
	fmt.Println("SHUTDOWN")
	fmt.Println("────────────────────────────────────────────────────────────")
	fmt.Printf("Shutdown callers          : %d\n", shutdownWorkers)
	fmt.Printf("Shutdown returned         : %d\n", shutdownReturned.Load())
	fmt.Printf("Broker shutdown           : %v\n", isShutdown)
	fmt.Printf("Topics remaining          : %d\n", remainingTopics)

	fmt.Println()
	fmt.Println("GOROUTINES")
	fmt.Println("────────────────────────────────────────────────────────────")
	fmt.Printf("Before workload           : %d\n", goroutinesBefore)
	fmt.Printf("After cleanup             : %d\n", goroutinesAfter)

	fmt.Println()
	fmt.Println("PERFORMANCE")
	fmt.Println("────────────────────────────────────────────────────────────")
	fmt.Printf("Duration                  : %v\n", duration)

	fmt.Println()
	fmt.Println("RESULT                    : PASS")

	printPass("Shutdown chaos")
}