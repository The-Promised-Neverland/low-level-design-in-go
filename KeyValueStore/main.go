package main

import (
	"container/heap"
	"errors"
	"sync"
	"time"
)

type Store interface {
	Set(key string, value any, ttl time.Duration) error
	Get(key string) (any, bool)
	Delete(key string) bool
	Exists(key string) bool
	Shutdown()
}

type Entry struct {
	Value     any
	ExpiresAt time.Time
	Expires   bool
}

type KeyValueStore struct {
	mu          sync.RWMutex // Allow read locks for concurrent reads. But for writes
	store       map[string]Entry
	expiryHeap  *ExpiryHeap
	expiryIndex map[string]*ExpiryItem
	done        chan struct{}
	wakeup      chan struct{}
	shutdown    bool
}

func NewKeyValueStore() Store {
	kv := &KeyValueStore{
		store:       make(map[string]Entry),
		expiryHeap:  NewMinPriorityQueue(),
		expiryIndex: make(map[string]*ExpiryItem),
		done:        make(chan struct{}),
		wakeup:      make(chan struct{}, 1),
		shutdown:    false,
	}
	kv.autoCleanup()
	return kv
}

func (kv *KeyValueStore) autoCleanup() {
	go func() {
		for {
			kv.mu.Lock()
			item := kv.expiryHeap.Peek()
			if item == nil {
				kv.mu.Unlock()
				select {
				case <-kv.done:
					return
				case <-kv.wakeup:
					continue
				}
			}
			expiresAt := item.ExpiresAt
			kv.mu.Unlock()
			wait := time.Until(expiresAt)
			if wait <= 0 {
				kv.mu.Lock()
				item = kv.expiryHeap.Peek()
				if item != nil && !time.Now().Before(item.ExpiresAt) {
					item = heap.Pop(kv.expiryHeap).(*ExpiryItem)
					delete(kv.store, item.Key)
					delete(kv.expiryIndex, item.Key)
				}
				kv.mu.Unlock()
				continue
			}
			timer := time.NewTimer(wait)
			select {
			case <-timer.C:
			case <-kv.wakeup:
				timer.Stop()
			case <-kv.done:
				timer.Stop()
				return
			}
		}
	}()
}

func (kv *KeyValueStore) Set(key string, value any, ttl time.Duration) error {
	if ttl < 0 {
		return errors.New("TTL cannot be negative")
	}
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if kv.shutdown {
		return errors.New("Shutting down")
	}
	oldEntry, exists := kv.store[key]
	entry := Entry{
		Value: value,
	}
	if ttl == 0 {
		entry.Expires = false
		if exists && oldEntry.Expires {
			item := kv.expiryIndex[key]
			heap.Remove(kv.expiryHeap, item.Position)
			delete(kv.expiryIndex, key)
			kv.signalCleanup() // trigger the cleaner
		}
		kv.store[key] = entry
		return nil
	}
	entry.Expires = true
	entry.ExpiresAt = time.Now().Add(ttl)
	if exists && oldEntry.Expires {
		item := kv.expiryIndex[key]
		item.ExpiresAt = entry.ExpiresAt
		heap.Fix(kv.expiryHeap, item.Position)
	} else {
		item := &ExpiryItem{
			Key:       key,
			ExpiresAt: entry.ExpiresAt,
		}
		heap.Push(kv.expiryHeap, item)
		kv.expiryIndex[key] = item
	}
	kv.signalCleanup()
	kv.store[key] = entry
	return nil
}

func (kv *KeyValueStore) signalCleanup() {
	select {
	case kv.wakeup <- struct{}{}:
	default:
	}
}
func (kv *KeyValueStore) Get(key string) (any, bool) {
	kv.mu.RLock()
	entry, exists := kv.store[key]
	if kv.shutdown {
		kv.mu.RUnlock()
		return nil, false
	}
	kv.mu.RUnlock()
	if !exists {
		return nil, false
	}
	if !entry.Expires || !entry.ExpiresAt.Before(time.Now()) {
		return entry.Value, true
	}
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if kv.shutdown {
		return nil, false
	}
	entry, exists = kv.store[key]
	if !exists {
		return nil, false
	}
	if entry.Expires && entry.ExpiresAt.Before(time.Now()) {
		delete(kv.store, key)
		if item := kv.expiryIndex[key]; item != nil {
			heap.Remove(kv.expiryHeap, item.Position)
			delete(kv.expiryIndex, key)
			kv.signalCleanup()
		}
		return nil, false
	}
	return entry.Value, true
}

func (kv *KeyValueStore) Delete(key string) bool {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if kv.shutdown {
		return false
	}
	_, exists := kv.store[key]
	if !exists {
		return false
	}
	delete(kv.store, key)
	if item := kv.expiryIndex[key]; item != nil {
		heap.Remove(kv.expiryHeap, item.Position)
		delete(kv.expiryIndex, key)
		kv.signalCleanup()
	}
	return true
}

func (kv *KeyValueStore) Exists(key string) bool {
	kv.mu.RLock()
	entry, exists := kv.store[key]
	if kv.shutdown {
		kv.mu.RUnlock()
		return false
	}
	kv.mu.RUnlock()
	if !exists {
		return false
	}
	if !entry.Expires || !entry.ExpiresAt.Before(time.Now()) {
		return true
	}
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if kv.shutdown {
		return false
	}
	entry, exists = kv.store[key]
	if !exists {
		return false
	}
	if entry.Expires && entry.ExpiresAt.Before(time.Now()) {
		delete(kv.store, key)
		if item := kv.expiryIndex[key]; item != nil {
			heap.Remove(kv.expiryHeap, item.Position)
			delete(kv.expiryIndex, key)
			kv.signalCleanup()
		}
		return false
	}
	return true
}

func (kv *KeyValueStore) Shutdown() {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if kv.shutdown == true {
		return
	}
	kv.shutdown = true
	close(kv.done)
}

func main() {

}
