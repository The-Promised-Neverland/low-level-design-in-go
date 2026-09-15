# Thread-Safe In-Memory Key-Value Store with TTL

## Problem Statement

Design and implement a **thread-safe in-memory Key-Value Store in Go** with support for optional key expiration using TTL (Time To Live).

Clients should be able to store, retrieve, update, and delete key-value pairs concurrently.

A key may optionally have a TTL. Once the TTL expires, the key must no longer be considered available.

## Requirements

- Store all key-value pairs completely in memory.
- Support multiple concurrent readers and writers.
- Support setting a key with an optional TTL.
- A TTL of `0` means the key never expires.
- `Get` must never return an expired value.
- Setting an existing key must replace its value.
- Setting an existing key must reset its TTL.
- Keys must be explicitly deletable.
- The store must support checking whether a key currently exists.
- Expired keys should eventually be removed from memory automatically.
- Expiration cleanup must happen in the background.
- Cleanup should not repeatedly scan the entire key space to find expired keys.
- The implementation must not create one goroutine or timer per key.
- Updating or deleting a key while expiration cleanup is running must be safe.
- An old expiration event must never delete a newer value of the same key.
- `Shutdown` must stop accepting new operations.
- `Shutdown` must safely stop all background processing.
- Calling `Shutdown` multiple times must be safe.
- The implementation must not have data races, deadlocks, or goroutine leaks.

## Interface

```go
type Store interface {
    Set(key string, value any, ttl time.Duration) error
    Get(key string) (any, bool)
    Delete(key string) bool
    Exists(key string) bool
    Shutdown()
}
```

## Example

A key can be stored without expiration:

```text
SET user:123 "Abhijit" TTL=0

             |
             v

      +---------------+
      |   KV STORE    |
      +---------------+
      |               |
      | user:123      |
      | "Abhijit"     |
      |               |
      | TTL: NEVER    |
      +---------------+

GET user:123
      |
      v
  "Abhijit"
```

A key can also be stored with a TTL:

```text
SET session:ABC "data" TTL=5s

             |
             v

      +---------------+
      |   KV STORE    |
      +---------------+
      |               |
      | session:ABC   |
      | "data"        |
      |               |
      | TTL: 5s       |
      +---------------+
```

Before expiration:

```text
GET session:ABC
      |
      v
   "data"
```

After 5 seconds:

```text
GET session:ABC
      |
      v
  NOT FOUND
```

An expired key must never be returned, even if the background cleanup process has not physically removed it from memory yet.

## Updating TTL

Setting an existing key replaces both its value and its TTL.

For example:

```text
SET token "A" TTL=5s

        |
        | 2 seconds later
        v

SET token "B" TTL=30s
```

The latest value is now:

```text
token -> "B"
TTL   -> 30s
```

The expiration associated with the previous value must not remove the newer value.

```text
Old expiration becomes due
          |
          v
      token = "B"
          |
          v
       SURVIVES
```

## Concurrent Access

Multiple goroutines may operate on the store simultaneously.

```text
                 +---------------+
                 |   KV STORE    |
                 +---------------+
                  ^    ^    ^    ^
                  |    |    |    |
                 GET  SET  DEL  GET
                  |    |    |    |
                 G1   G2   G3   G4

                       ^
                       |
                 TTL CLEANUP
```

Normal client operations and background expiration may therefore happen concurrently.

The store must remain consistent and thread-safe under this contention.

## Slow Cleanup

Expiration has two separate concerns:

1. An expired key must no longer be returned to callers.
2. Expired entries should eventually be physically removed from memory.

Therefore, even if background cleanup has not yet processed an expired key:

```text
Current time > expiration time

            |
            v

GET key
   |
   v
NOT FOUND
```

The cleanup mechanism is responsible for eventually reclaiming the memory occupied by expired entries.

## Shutdown

Calling `Shutdown()` should transition the store into a stopped state.

```text
              Shutdown()
                  |
                  v
          +---------------+
          |   KV STORE    |
          |   SHUTDOWN    |
          +---------------+
                  |
                  v
          Background work
              terminates
```

After shutdown:

- New operations must not be accepted.
- Background expiration processing must terminate safely.
- No background goroutines should remain running.
- Repeated calls to `Shutdown()` must be safe.

## Scope

This implementation provides a **single-process, in-memory key-value store with TTL expiration**.

The following are outside the scope of this problem:

- Persistence to disk
- Replication
- Distributed storage
- Network APIs
- Transactions
- Authentication
- LRU/LFU eviction
- Memory-size-based eviction
- Clustering
- High availability
- Recovery after process failure

All stored data is lost when the process terminates.

The choice of data structures, expiration strategy, synchronization mechanism, and background cleanup design are part of the LLD exercise.