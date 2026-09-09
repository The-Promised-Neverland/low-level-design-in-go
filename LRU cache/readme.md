# Thread-Safe In-Memory LRU Cache

A thread-safe, in-memory cache implemented in Go using the **Least Recently Used (LRU)** eviction policy.

## Problem Statement

Design and implement an in-memory cache with a configurable maximum capacity.

When the cache reaches capacity, inserting a new item should automatically evict the **least recently used** item.

Both reads and writes affect the usage order of cached entries.

## Functional Requirements

The cache should support:

```go
type Cache interface {
    Get(key string) (any, bool)
    Put(key string, value any)
}
```

### Get

```go
Get(key string) (value any, found bool)
```

* Return the value if the key exists.
* Return `found = false` if the key does not exist.
* Accessing an existing key should mark it as the **most recently used** item.

### Put

```go
Put(key string, value any)
```

* Insert a new key-value pair.
* If the key already exists:

  * Update its value.
  * Mark it as the most recently used item.
* If inserting a new item exceeds the configured capacity:

  * Remove the least recently used item.
  * Insert the new item as the most recently used item.

## Non-Functional Requirements

* `Get()` should have **O(1)** average time complexity.
* `Put()` should have **O(1)** average time complexity.
* Finding the least recently used item should be **O(1)**.
* Removing the least recently used item should be **O(1)**.
* The cache must be thread-safe.
* Multiple goroutines should be able to call `Get()` and `Put()` safely.
* The implementation should be entirely in memory.
* No external systems such as Redis or a database should be required.

## Example

Consider a cache with:

```text
Capacity = 3
```

Insert three entries:

```text
Put("A", 10)
Put("B", 20)
Put("C", 30)
```

The usage order is:

```text
LRU               MRU
 ↓                 ↓

 A  <->  B  <->  C
```

Now access `A`:

```text
Get("A")
```

Since `A` was just accessed, it becomes the most recently used item:

```text
LRU               MRU
 ↓                 ↓

 B  <->  C  <->  A
```

Now insert another entry:

```text
Put("D", 40)
```

The cache is already at capacity.

`B` is currently the least recently used item, so it is evicted.

The resulting cache becomes:

```text
LRU               MRU
 ↓                 ↓

 C  <->  A  <->  D
```

## Design Challenge

A hash map provides average **O(1)** key lookup:

```text
key -> cache entry
```

However, a hash map alone cannot efficiently determine which entry was least recently used.

Similarly, a linked list can maintain usage order efficiently, but searching for a key in a linked list would require **O(n)** time.

The implementation therefore needs an appropriate combination of data structures that provides:

```text
Lookup       → O(1)
Insert       → O(1)
Update       → O(1)
Re-ranking   → O(1)
Eviction     → O(1)
```

## Data Structure

A common approach combines:

* A **hash map** for direct key lookup.
* A **doubly linked list** for maintaining usage order.

Conceptually:

```text
Hash Map
────────────────────────

"A" ─────────────┐
"B" ────────┐    │
"C" ───┐    │    │
       │    │    │
       ▼    ▼    ▼

Doubly Linked List
────────────────────────

MRU                           LRU
 ↓                             ↓

 C  <->  B  <->  A
```

The hash map provides direct access to a node, while the doubly linked list allows that node to be removed and moved to the most recently used position in constant time.

## Concurrency

The cache may receive concurrent operations from multiple goroutines:

```text
Goroutine 1 ── Get("A") ──┐
Goroutine 2 ── Put("B") ──┤
Goroutine 3 ── Get("C") ──┼──> LRU Cache
Goroutine 4 ── Put("D") ──┤
Goroutine 5 ── Get("A") ──┘
```

Synchronization must ensure that the hash map and linked-list structure always remain consistent.

A partially updated linked list or map could otherwise lead to:

* Incorrect eviction.
* Broken `next` / `prev` pointers.
* Lost cache entries.
* Duplicate nodes.
* Linked-list corruption.

## Complexity

| Operation             | Average Complexity |
| --------------------- | -----------------: |
| Get                   |               O(1) |
| Put                   |               O(1) |
| Update existing entry |               O(1) |
| Move entry to MRU     |               O(1) |
| Find LRU entry        |               O(1) |
| Evict LRU entry       |               O(1) |

Memory usage is:

```text
O(capacity)
```

since the cache stores at most the configured number of entries.
