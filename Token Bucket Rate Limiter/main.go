package main

import (
	"sync"
	"time"
)

type RateLimiter interface {
	Allow(key string) (bool, error)
}

// A token bucket needs to know at least:
type Bucket struct {
	Tokens     float64
	LastRefill time.Time
	bmu        sync.Mutex // mutex for bucket
}

// RATE LIMITER, Token Bucket rate limiter
type TokenBucketRateLimiter struct {
	Capacity   float64            // maximum tokens in bucket
	RefillRate float64            // tokens added per second
	cache      map[string]*Bucket // in-memory cache
	rmu        sync.RWMutex       // mutex for map
}

func NewTokenBucketRateLimiter(capacity float64, refillRate float64) *TokenBucketRateLimiter {
	return &TokenBucketRateLimiter{
		Capacity:   capacity,
		RefillRate: refillRate,
		cache:      make(map[string]*Bucket),
	}
}

func minFunction(a float64, b float64) float64 {
	if a > b {
		return b
	}
	return a
}
func (rl *TokenBucketRateLimiter) Allow(userId string) (bool, error) {
	// Find bucket
	rl.rmu.RLock()
	bucket, exists := rl.cache[userId]
	rl.rmu.RUnlock()
	// Create
	if !exists {
		rl.rmu.Lock()
		// Check again after acquiring write lock.
		bucket, exists = rl.cache[userId]
		if !exists {
			bucket = &Bucket{
				Tokens:     rl.Capacity,
				LastRefill: time.Now(),
			}
			rl.cache[userId] = bucket
		}
		rl.rmu.Unlock()
	}
	// Bucket retrieved
	bucket.bmu.Lock()
	defer bucket.bmu.Unlock()
	now := time.Now()
	secondsElapsed := now.Sub(bucket.LastRefill).Seconds()
	tokensToFillIn := rl.RefillRate * secondsElapsed
	bucket.Tokens = minFunction(rl.Capacity, tokensToFillIn+bucket.Tokens)
	bucket.LastRefill = now
	// No tokens available
	if bucket.Tokens < 1 {
		return false, nil
	}
	bucket.Tokens--
	return true, nil
}

func main() {
}
