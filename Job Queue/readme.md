# In-Memory Concurrent Job Queue

A thread-safe, in-memory **Job Queue / Worker Pool** implemented in Go.

The system allows multiple producers to submit jobs while a configurable pool of workers processes them concurrently. Failed jobs can be retried and eventually moved to a **Dead Letter Queue (DLQ)** when retries are exhausted.

## Problem Statement

Design and implement an in-memory concurrent job processing system.

The queue should accept jobs from multiple goroutines, distribute them across multiple workers, retry failed jobs, support a dead letter queue, and shut down gracefully without losing already accepted work.

## Functional Requirements

### Job

A job should contain the information required for processing and retry handling.

For example:

```go
type Job struct {
    ID       string
    Payload  any
    Attempts int
}
```

Additional fields may be introduced as required.

## Expected Interface

```go
type JobQueue interface {
    Submit(job Job) error
    Start()
    Shutdown()
}
```

An implementation may expose additional operations such as inspecting or redriving the DLQ.

## Submit

```go
Submit(job Job) error
```

`Submit` should:

* Add a job to the queue.
* Be safe when called concurrently by multiple goroutines.
* Reject new jobs once shutdown has started.
* Ensure an accepted job is eventually either processed successfully or moved to the DLQ.

## Worker Pool

The number of workers should be configurable.

Each worker continuously waits for available jobs:

```text
                 Job Queue
                    │
          ┌─────────┼─────────┐
          │         │         │
          ▼         ▼         ▼

      Worker-1  Worker-2  Worker-3
```

Workers should:

* Run concurrently.
* Process one job at a time.
* Ensure each individual attempt is handled by only one worker.
* Pick another available job after completing the current one.
* Exit cleanly during shutdown.

## Job Processing

Job execution can be represented by a function such as:

```go
func Process(job Job) error
```

Processing may either succeed or fail.

```text
Job
 │
 ▼
Process()
 │
 ├── Success ──> Complete
 │
 └── Failure ──> Retry
```

## Retry

If processing fails, the job should be retried.

The default maximum retry count is:

```text
3 retries
```

A retry may optionally support:

* Retry delay
* Backoff
* Jitter
* Per-job retry configuration

Only one worker should process a particular attempt at any given time.

Conceptually:

```text
Attempt 1
    │
    └── FAIL
         │
         ▼
     Attempt 2
         │
         └── FAIL
              │
              ▼
          Attempt 3
              │
              └── FAIL
                   │
                   ▼
                  DLQ
```

## Dead Letter Queue

Jobs that continue to fail after exhausting their retry policy should be moved to a **Dead Letter Queue**.

```text
Processing
    │
    ├── Success ───────> Done
    │
    └── Retry exhausted
             │
             ▼
            DLQ
```

The DLQ should keep failed jobs available for later inspection.

An implementation may also support redriving DLQ jobs back into the active queue.

## Example

Suppose four jobs are submitted:

```text
Submit A
Submit B
Submit C
Submit D
```

With three workers:

```text
                Queue
             [ A B C D ]
                 │
        ┌────────┼────────┐
        ▼        ▼        ▼

    Worker-1  Worker-2  Worker-3
       A         B         C
```

If Worker-2 finishes first:

```text
Worker-2
   │
   └── B completed
          │
          ▼
          D
```

If `A` fails:

```text
Worker-1 -> A -> FAIL
                 │
                 ▼
              Retry A
```

A worker can later process the retry:

```text
Worker-2 -> Retry A
```

If `A` continues failing until its retry limit is exhausted:

```text
A
│
├── Attempt 1 -> FAIL
├── Attempt 2 -> FAIL
├── Attempt 3 -> FAIL
│
▼
DLQ
```

## Concurrency

Multiple producers may submit jobs concurrently while multiple workers consume them:

```text
Producer-1 ──┐
Producer-2 ──┤
Producer-3 ──┼──> Job Queue
Producer-4 ──┘
                 │
        ┌────────┼────────┐
        ▼        ▼        ▼
    Worker-1  Worker-2  Worker-3
```

The implementation must remain correct under concurrent:

* Job submissions
* Job consumption
* Retry processing
* DLQ writes
* Startup and shutdown

## Graceful Shutdown

`Shutdown()` should stop accepting new jobs while allowing already accepted work to finish.

The expected sequence is:

```text
Shutdown()
    │
    ▼
Reject new submissions
    │
    ▼
Finish queued jobs
    │
    ▼
Finish jobs currently being processed
    │
    ▼
Finish pending retries
    │
    ▼
Success or DLQ
    │
    ▼
Workers exit
    │
    ▼
Shutdown() returns
```

Shutdown must avoid:

* Dropping accepted jobs
* Sending jobs to a closed channel
* Closing a channel while producers are still sending
* Leaving workers blocked forever
* Goroutine leaks
* Deadlocks

## Thread Safety

The queue must support multiple producers and workers concurrently without:

* Data races
* Duplicate processing of the same attempt
* Lost jobs
* Corrupted DLQ state
* Deadlocks
* Goroutine leaks
* `send on closed channel` panics

Synchronization should be designed carefully around queue lifecycle and shared state.

## In-Memory Constraint

The entire system must operate within the Go process.

Do not use:

```text
Redis
Kafka
RabbitMQ
SQS
Databases
External message brokers
```

All queued jobs and DLQ entries are lost when the process terminates.

## Expected Properties

The final implementation should provide:

| Property                       | Expected Behavior        |
| ------------------------------ | ------------------------ |
| Concurrent producers           | Supported                |
| Concurrent workers             | Supported                |
| Configurable worker count      | Supported                |
| Exactly one worker per attempt | Required                 |
| Retry                          | Up to configured maximum |
| DLQ                            | Failed jobs retained     |
| Graceful shutdown              | Required                 |
| Thread safety                  | Required                 |
| External dependencies          | None                     |

## Design Considerations

Important questions to consider while designing the system:

* Which Go concurrency primitives are appropriate?
* Should the job channel be buffered or unbuffered?
* What should determine the queue buffer size?
* How should workers know when they need to exit?
* Who owns the job channel?
* Who is responsible for closing it?
* How should `Shutdown()` know that all accepted jobs are complete?
* How should retries be scheduled?
* Should retries happen inside the same worker or be requeued?
* How can retry jobs be preserved during shutdown?
* How do you prevent `Submit()` from sending to a closed channel?
* How should lifecycle state be synchronized?
* Should DLQ synchronization be independent from queue lifecycle synchronization?
* How should a DLQ redrive behave if shutdown begins midway through the operation?
