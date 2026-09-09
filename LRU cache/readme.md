LLD Problem: Thread-Safe In-Memory LRU Cache

Functional Requirements
Design an in-memory cache using the LRU (Least Recently Used) eviction policy.
The cache should have a configurable maximum capacity.
    Support the following operations:
    Get(key string) (value any, found bool)
    Put(key string, value any)
    Get(key):
Return the value if the key exists.
Return found = false if the key does not exist.
Accessing a key should mark it as the most recently used item.
Put(key, value):
Insert a new key-value pair.
If the key already exists, update its value and mark it as most recently used.
If inserting a new item exceeds the cache capacity, evict the least recently used item.
The cache must be thread-safe and support concurrent Get() and Put() calls.

Non-Functional Requirements
Get() should have O(1) average time complexity.
Put() should have O(1) average time complexity.
Finding and removing the least recently used item should be O(1).
The implementation should be entirely in-memory.
The cache should not depend on external systems such as Redis or a database.
Example
Capacity = 3

Put("A", 10)
Put("B", 20)
Put("C", 30)

Cache:
A, B, C

Get("A")

LRU order:
B -> C -> A
^         ^
LRU       MRU

Put("D", 40)

"B" is evicted because it is the least recently used.

LRU order:
C -> A -> D
^         ^
LRU       MRU
Expected Interface
type Cache interface {
	Get(key string) (any, bool)
	Put(key string, value any)
}
Design Constraint

A hash map provides O(1) lookup, but by itself cannot efficiently determine which item was least recently used.

Design the appropriate combination of data structures so that lookup, insertion, update, and eviction all remain O(1).