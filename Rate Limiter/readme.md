# Token Bucket Rate Limiter

A thread-safe, in-memory **Token Bucket Rate Limiter** implemented in Go.

The implementation maintains an independent token bucket for each user and supports concurrent requests without unnecessarily blocking requests belonging to different users.

## Problem Statement

Design and implement an in-memory rate limiter using the **Token Bucket algorithm**.

The rate limiter should determine whether a request from a given user is allowed based on the number of tokens currently available in that user's bucket.

Each request consumes one token. Tokens are replenished over time according to a configurable refill rate.

## Requirements

* Maintain an independent token bucket for every user.
* Support configurable bucket capacity.
* Support configurable token refill rate.
* A new bucket starts at full capacity.
* Each successful request consumes one token.
* Reject requests when fewer than one token is available.
* Refill tokens based on the elapsed time since the previous refill.
* Never allow the number of tokens to exceed the configured capacity.
* Support fractional tokens for accurate time-based refilling.
* Be safe under concurrent access from multiple goroutines.
* Requests for different users should not unnecessarily block each other.
* Safely handle concurrent creation of a bucket for the same user.

## Interface

```go
type RateLimiter interface {
    Allow(key string) (bool, error)
}
```

`Allow` returns:

* `true` when the request is allowed.
* `false` when the user's rate limit has been exceeded.

## Example

Given:

```text
Capacity   = 5 tokens
RefillRate = 2 tokens/second
```

A new user starts with:

```text
[● ● ● ● ●]
 5 tokens
```

After five requests:

```text
Request 1 → ALLOWED
Request 2 → ALLOWED
Request 3 → ALLOWED
Request 4 → ALLOWED
Request 5 → ALLOWED
Request 6 → REJECTED
```

After one second, approximately two tokens become available again:

```text
[● ● ○ ○ ○]
```

The next two requests can therefore be accepted.

## Concurrency

Each user has an independent bucket:

```text
Alice ─────┐
Alice ─────┤
Bob ───────┤
Carol ─────┼──> RateLimiter.Allow()
Bob ───────┤
Alice ─────┤
Dave ──────┘
```

The bucket map is protected separately from individual buckets.

This allows requests for different users to proceed concurrently instead of serializing every request behind a single global mutex.

## Synchronization Strategy

Two synchronization levels are used:

```text
Bucket Map
    │
    └── sync.RWMutex
          │
          ├── Alice → Bucket Mutex
          ├── Bob   → Bucket Mutex
          ├── Carol → Bucket Mutex
          └── Dave  → Bucket Mutex
```

The map-level `RWMutex` protects bucket lookup and creation.

Each bucket has its own mutex protecting:

* Current token count
* Last refill timestamp
* Token refill calculation
* Token consumption

As a result, two concurrent requests for the **same user** are serialized, while requests for **different users** can execute concurrently.

## Complexity

Average bucket lookup:

```text
O(1)
```

Rate-limit calculation:

```text
O(1)
```

Therefore, each `Allow()` operation runs in approximately:

```text
O(1)
```

with memory usage proportional to the number of unique users:

```text
O(number of users)
```
