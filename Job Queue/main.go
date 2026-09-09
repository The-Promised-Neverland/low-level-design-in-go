package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type JobQueue interface {
	Submit(job Job) error
	RedriveDLQ()
	Start()
	Shutdown()
}

var DefaultRetryOptions = RetryOptions{
	MaxRetries: 3,
	Backoff:    time.Second,
	Jitter:     0,
}

type RetryOptions struct {
	MaxRetries int
	Backoff    time.Duration
	Jitter     time.Duration
}

type Job struct {
	jobID    string
	Payload  interface{}
	Attempts int
	Options  *RetryOptions
}

type InMemoryJobQueue struct {
	bufferSize  int
	jobs        chan Job
	workerCount int
	wg          sync.WaitGroup
	mu          sync.RWMutex
	dlqMu       sync.Mutex
	shutdown    bool
	dlq         []Job
}

func NewInMemoryJobQueue(queueBuffer int, workerCount int) *InMemoryJobQueue {
	return &InMemoryJobQueue{
		bufferSize:  queueBuffer,
		jobs:        make(chan Job, queueBuffer),
		workerCount: workerCount,
		dlq:         make([]Job, 0),
		shutdown:    true,
	}
}

func (q *InMemoryJobQueue) storeInDLQ(job Job) {
	q.dlqMu.Lock()
	defer q.dlqMu.Unlock()
	q.dlq = append(q.dlq, job)
	fmt.Printf("JobID: %s stored in DLQ\n", job.jobID)
}

func RetryJob(job Job) bool {
	if job.Options == nil {
		job.Options = &DefaultRetryOptions
	}
	for i := 0; i < job.Options.MaxRetries; i++ {
		delay := job.Options.Backoff
		if job.Options.Jitter > 0 {
			delay += time.Duration(rand.Int63n(int64(job.Options.Jitter)))
		}
		time.Sleep(delay)
		if rand.Intn(2) == 0 {
			fmt.Printf("Job %s succeeded on attempt %d\n", job.jobID, i+1)
			return true
		}
		fmt.Printf("Job %s failed on attempt %d\n", job.jobID, i+1)
	}
	return false
}

func (q *InMemoryJobQueue) processJob(job Job) {
	if RetryJob(job) {
		fmt.Printf("JobID: %s processed successfully\n", job.jobID)
	} else {
		fmt.Printf("JobID: %s failed after retries, storing in DLQ\n", job.jobID)
		q.storeInDLQ(job)
	}
}

func (q *InMemoryJobQueue) worker(workerId int) {
	defer q.wg.Done()
	fmt.Printf("Starting Worker %d\n", workerId)
	for job := range q.jobs {
		q.processJob(job)
	}
	fmt.Printf("Worker %d stopped\n", workerId)
}

func (q *InMemoryJobQueue) initiateWorkerPool() {
	for i := 0; i < q.workerCount; i++ {
		q.wg.Add(1)
		go q.worker(i + 1)
	}
	fmt.Println("Workers started")
}

func (q *InMemoryJobQueue) Submit(job Job) error {
	q.mu.RLock()
	defer q.mu.RUnlock()
	if q.shutdown == true {
		return fmt.Errorf("queue has been shutdown; jobs cannot be accepted")
	}
	q.jobs <- job
	return nil
}

func (q *InMemoryJobQueue) Shutdown() {
	q.mu.Lock()
	if q.shutdown == true {
		q.mu.Unlock()
		return
	}
	q.shutdown = true
	close(q.jobs)
	q.mu.Unlock()
	q.wg.Wait()
	fmt.Println("Worker pool is initiated for shutdown.")
}

func (q *InMemoryJobQueue) Start() {
	q.mu.Lock()
	if !q.shutdown {
		q.mu.Unlock()
		return
	}
	q.shutdown = false
	q.jobs = make(chan Job, q.bufferSize)
	q.mu.Unlock()
	q.initiateWorkerPool()
}

func (q *InMemoryJobQueue) RedriveDLQ() {
	q.mu.RLock()
	if q.shutdown {
		q.mu.RUnlock()
		fmt.Println("Cannot redrive DLQ. Queue is shutting down.")
		return
	}
	q.mu.RUnlock()
	q.dlqMu.Lock()
	dlq := append([]Job(nil), q.dlq...)
	q.dlq = nil
	q.dlqMu.Unlock()
	go func() {
		for _, job := range dlq {
			if err := q.Submit(job); err != nil {
				q.storeInDLQ(job)
				return
			}
		}
	}()
	fmt.Println("DLQ has been instructed to re-drive the jobs.")
}

func main() {

}
