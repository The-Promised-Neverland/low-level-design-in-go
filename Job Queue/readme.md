LLD Problem: In-Memory Concurrent Job Queue

Design and implement a thread-safe, in-memory Job Queue / Worker Pool in Go.

The system should allow producers to submit jobs into a queue. A configurable number of workers should consume and process these jobs concurrently.

Requirements:

1. Define a Job containing:
   - ID
   - Payload
   - Retry count or any other fields you think are necessary.

2. The Job Queue should expose roughly the following API:

   type JobQueue interface {
       Submit(job Job) error
       Start()
       Shutdown()
   }

3. Submit(job):
   - Adds a job to the queue.
   - Multiple goroutines may call Submit concurrently.
   - Once shutdown begins, new jobs must not be accepted.

4. Worker Pool:
   - Number of workers should be configurable.
   - Workers run concurrently.
   - Each submitted job must be processed by exactly one worker.
   - A worker should pick the next available job after finishing its current job.

5. Job Processing:
   - Simulate job processing using a function such as:

       Process(job Job) error

   - Processing may succeed or fail.

6. Retry:
   - If processing fails, retry the job.
   - Maximum retry count: 3.
   - A retry must not result in multiple workers processing the same attempt simultaneously.

7. Dead Letter Queue (DLQ):
   - If a job continues to fail after the maximum retries, move it to a DLQ.
   - Failed jobs should be inspectable later.

8. Graceful Shutdown:
   - Stop accepting new jobs.
   - Finish processing all jobs that were already accepted.
   - Include jobs that are currently being processed or waiting for retry.
   - Workers should exit cleanly.
   - Shutdown() should return only after all accepted work has either succeeded or moved to the DLQ.

9. Thread Safety:
   - The system must work correctly when multiple producers and workers operate concurrently.
   - Avoid race conditions, deadlocks, and goroutine leaks.

10. In-memory only:
    - Do not use Redis, Kafka, RabbitMQ, databases, or external queues.

Expected properties:

- Concurrent job processing.
- Configurable worker count.
- Each job handled by only one worker at a time.
- Retry up to 3 times.
- DLQ after retry exhaustion.
- Graceful shutdown.
- Thread-safe implementation.

Example:

Submit A
Submit B
Submit C
Submit D

Worker-1 -> A
Worker-2 -> B
Worker-3 -> C

Worker-2 finishes B
Worker-2 -> D

Worker-1 fails A
A -> Retry #1

Worker-3 finishes C

Worker-2 may pick up retry A

A fails 3 times
A -> DLQ

Think about:

- Which Go concurrency primitives should be used?
- Buffered vs unbuffered channels?
- How will workers know when to stop?
- How will Shutdown() know all accepted jobs are finished?
- Where should failed jobs be requeued?
- How do you prevent sending to a closed channel?
- Who should own/close the channels?
- How do you avoid losing retry jobs during shutdown?