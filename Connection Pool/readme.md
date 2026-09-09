# Thread-Safe In-Memory Connection Pool

Design and implement a thread-safe, in-memory **Connection Pool** in Go.

The pool manages a limited number of reusable connections. Multiple goroutines should be able to acquire connections concurrently, use them, and return them to the pool.

Connections should be reused whenever possible instead of creating a new connection for every request.

## Problem Statement

Creating a new connection for every request can be expensive.

A connection pool maintains a collection of reusable connections:

```text
Acquire → Use Connection → Release → Reuse
```

The pool should create connections when required, reuse idle connections, enforce a maximum number of open connections, clean up connections that remain idle for too long, and correctly handle concurrent callers.

## Functional Requirements

### Connection

For this exercise, a connection can be simulated using a simple structure:

```go
type Connection struct {
    ID      int
    Healthy bool
}
```

You may add additional fields required by your design.

No real database or network connection is required.

## Expected Interface

```go
type ConnectionPool interface {
    Acquire(ctx context.Context) (*Connection, error)
    Release(conn *Connection) error
    Shutdown()
}
```

## Pool Configuration

The pool should support configuration such as:

```text
MaxConnections
IdleTimeout
```

Optionally, the design may also support:

```text
MinConnections
```

`MaxConnections` defines the maximum number of connections that may exist at the same time.

`IdleTimeout` defines how long an unused connection may remain in the pool before it is closed.

## Acquire

```go
Acquire(ctx context.Context) (*Connection, error)
```

When a caller requests a connection:

### Idle connection available

If an idle connection already exists, reuse it.

```text
Idle Pool

[C1] [C2] [C3]

Acquire()
   │
   ▼
  C1
```

The connection becomes borrowed and must no longer be considered idle.

### No idle connection, pool below maximum

If there are no idle connections but the number of existing connections is below `MaxConnections`, create a new connection.

```text
MaxConnections   = 10
TotalConnections = 7
IdleConnections  = 0

Acquire()
   │
   ▼
Create C8
   │
   ▼
Return C8
```

### No idle connection, pool at maximum

If the pool has reached `MaxConnections`, a new connection must not be created.

The caller should wait until another goroutine releases a connection.

```text
MaxConnections   = 3
TotalConnections = 3
IdleConnections  = 0

G1 → using C1
G2 → using C2
G3 → using C3

G4 → Acquire()
       │
       ▼
      WAIT
       │
       │
G2 → Release(C2)
       │
       ▼
G4 receives C2
```

The waiting goroutine should block efficiently rather than repeatedly checking the pool in a busy loop.

## Context Cancellation

`Acquire()` must respect the supplied `context.Context`.

For example:

```go
ctx, cancel := context.WithTimeout(
    context.Background(),
    2*time.Second,
)
defer cancel()

conn, err := pool.Acquire(ctx)
```

If no connection becomes available before the context expires:

```text
Acquire()
   │
   ▼
WAIT
   │
   ├── connection available → return connection
   │
   └── context cancelled    → return error
```

A caller should never remain blocked indefinitely after its context has been cancelled.

## Release

```go
Release(conn *Connection) error
```

When a caller finishes using a connection, it should return it to the pool.

```text
Acquire C1
    │
    ▼
Use C1
    │
    ▼
Release C1
    │
    ▼
Idle Pool
```

A healthy released connection should become available for another caller.

## Connection Reuse

The pool should prefer reusing existing idle connections instead of unnecessarily creating new ones.

Example:

```text
Acquire()
   │
   └── Create C1

Release(C1)
   │
   ▼

Idle: [C1]

Acquire()
   │
   ▼

Reuse C1
```

A new `C2` should not be created when `C1` is already available for reuse.

## Idle Connection Timeout

Connections should not remain idle forever.

When a connection has remained unused longer than `IdleTimeout`, it should be closed and removed from the pool.

Example:

```text
IdleTimeout = 30 seconds

C1 released
     │
     ▼
Idle
     │
     │ 30+ seconds unused
     ▼
Close C1
     │
     ▼
Remove from pool
```

Closing an idle connection should reduce the total number of existing connections.

A later `Acquire()` may create a new connection if the pool is below `MaxConnections`.

## Unhealthy Connections

A connection may become unhealthy while being used.

For example:

```go
conn.Healthy = false
```

When an unhealthy connection is released, it should not be returned to the idle pool.

Instead:

```text
Release(C1)
    │
    ▼
Healthy?
 │
 ├── YES → return to idle pool
 │
 └── NO  → close/discard connection
```

Discarding an unhealthy connection should reduce the total number of existing connections.

## Double Release

The pool should protect itself against accidentally releasing the same connection more than once.

For example:

```go
conn, _ := pool.Acquire(ctx)

pool.Release(conn)

pool.Release(conn) // invalid
```

The second release should return an error rather than placing the same connection into the idle pool twice.

Otherwise, the same physical connection could eventually be handed to two callers simultaneously.

## Concurrency

Multiple goroutines may interact with the pool concurrently.

```text
G1 ── Acquire ──┐
G2 ── Acquire ──┤
G3 ── Release ──┼──> Connection Pool
G4 ── Acquire ──┤
G5 ── Release ──┘
```

The implementation must correctly synchronize shared state.

It must prevent situations such as:

```text
MaxConnections = 10
Total           = 9

G1 checks total → 9
G2 checks total → 9
G3 checks total → 9

G1 creates connection
G2 creates connection
G3 creates connection

Total → 12   ❌
```

The maximum connection limit must never be exceeded.

## Idle vs Borrowed Connections

The implementation should distinguish between:

```text
Total Connections
       │
       ├── Idle
       │
       └── Borrowed
```

Therefore:

```text
TotalConnections = IdleConnections + BorrowedConnections
```

For example:

```text
MaxConnections   = 10

TotalConnections = 10
IdleConnections  = 3
Borrowed         = 7
```

The existence of only three idle connections does not mean the pool may create seven more connections.

## Shutdown

```go
Shutdown()
```

Shutdown should stop the pool from accepting new acquisitions.

Idle connections should be closed.

Connections currently borrowed by callers may be returned later, but they should not be placed back into the reusable idle pool after shutdown.

Conceptually:

```text
Shutdown()
    │
    ▼
Reject new Acquire()
    │
    ▼
Close idle connections
    │
    ▼
Borrowed connections return
    │
    ▼
Close/discard them
```

Any goroutines currently waiting inside `Acquire()` should also be able to stop waiting when the pool shuts down.

The implementation must avoid:

* Deadlocks
* Goroutine leaks
* Double release
* Exceeding `MaxConnections`
* Returning the same connection to multiple callers
* Reusing unhealthy connections
* Waiting forever after context cancellation
* Sending to closed channels
* Closing connections while they are still legitimately borrowed

## Example

Suppose:

```text
MaxConnections = 3
```

Three goroutines acquire connections:

```text
G1 → Acquire → C1
G2 → Acquire → C2
G3 → Acquire → C3
```

The pool is now:

```text
Total    = 3
Borrowed = 3
Idle     = 0
```

Another goroutine requests a connection:

```text
G4 → Acquire
      │
      ▼
     WAIT
```

Then:

```text
G2 → Release(C2)
```

`C2` becomes available and can immediately be reused:

```text
G4 ← C2
```

Later:

```text
G4 → Release(C2)
```

If nobody acquires `C2` before the configured idle timeout:

```text
C2
 │
 │ idle too long
 ▼
CLOSE
```

The pool now has room to create another connection when necessary.

## Expected Properties

| Property                       | Expected Behavior |
| ------------------------------ | ----------------- |
| Connection reuse               | Required          |
| Configurable maximum           | Required          |
| Concurrent Acquire             | Supported         |
| Concurrent Release             | Supported         |
| Context-aware waiting          | Required          |
| Maximum connection enforcement | Required          |
| Idle timeout                   | Required          |
| Unhealthy connection removal   | Required          |
| Double-release protection      | Required          |
| Graceful shutdown              | Required          |
| Thread safety                  | Required          |
| External dependencies          | None              |

## Design Considerations

Think carefully about:

* Which state should be protected by a mutex?
* Can a channel represent available connections?
* What should the capacity of that channel be?
* How should `TotalConnections` be tracked?
* How do you atomically check `total < max` and create a connection?
* How should waiting callers be notified when a connection is released?
* How should `Acquire()` wait without busy spinning?
* How should `Acquire()` simultaneously wait for a connection, context cancellation, or shutdown?
* How should the pool track when a connection became idle?
* Who is responsible for cleaning up expired idle connections?
* What happens when an unhealthy connection is released while other callers are waiting?
* How do you prevent the same connection from being released twice?
* Should the available-connections channel ever be closed?
* How do you prevent `Release()` from sending to a closed channel?
* How should waiting callers react when `Shutdown()` begins?
* How do you guarantee that `MaxConnections` is never exceeded under heavy concurrency?

The final implementation should remain correct even when hundreds or thousands of goroutines acquire and release connections concurrently.
