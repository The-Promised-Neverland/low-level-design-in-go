package main

import (
	"fmt"
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
}

// RATE LIMITER, Token Bucket rate limiter
type TokenBucketRateLimiter struct {
	Capacity   float64            // maximum tokens in bucket
	RefillRate float64            // tokens added per second
	cache      map[string]*Bucket // in-memory cache
	mu         sync.Mutex         // mutex for map
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
	rl.mu.Lock()
	defer rl.mu.Unlock()
	// User is first time user
	if _, ok := rl.cache[userId]; !ok {
		rl.cache[userId] = &Bucket{
			Tokens:     rl.Capacity,
			LastRefill: time.Now(),
		}
	}
	// Current bucket snapshot
	bucket := rl.cache[userId]
	now := time.Now()

	// User bucket refilled as per seconds elapsed
	secondsElapsed := now.Sub(bucket.LastRefill).Seconds()
	tokensToFillIn := rl.RefillRate * secondsElapsed
	bucket.Tokens = minFunction(rl.Capacity, tokensToFillIn+bucket.Tokens)
	bucket.LastRefill = now

	// If not tokens available
	if bucket.Tokens < 1 {
		return false, nil
	}

	// Deduct the token use
	bucket.Tokens--

	return true, nil
}

func main() {
	rl := NewTokenBucketRateLimiter(100, 5)
	userID := "user-123"
	// Fire 105 requests immediately.
	// Capacity is 100, so roughly the first 100 should succeed.
	for i := 1; i <= 105; i++ {
		go func() {
			allowed, _ := rl.Allow(userID)
			fmt.Printf(
				"Request %d: allowed=%v, tokens=%.2f\n",
				i,
				allowed,
				rl.cache[userID].Tokens,
			)
		}()
	}
	fmt.Println("\nWaiting 2 seconds...")
	time.Sleep(2 * time.Second)
	fmt.Println("\nSending another 12 requests:")
	for i := 1; i <= 12; i++ {
		go func() {
			allowed, _ := rl.Allow(userID)
			fmt.Printf(
				"Request %d: allowed=%v, tokens=%.2f\n",
				i,
				allowed,
				rl.cache[userID].Tokens,
			)
		}()
	}
	time.Sleep(10 * time.Second)

}
