package middleware

import (
	"sync"
	"testing"
	"time"
)

func TestRateLimiter_Allow_Basic(t *testing.T) {
	rl := NewRateLimiter(10, 10) // 10/s, burst 10

	// First 10 requests should be allowed (burst)
	for i := 0; i < 10; i++ {
		if !rl.allow("192.168.1.1") {
			t.Errorf("request %d should be allowed", i)
		}
	}

	// 11th should be denied
	if rl.allow("192.168.1.1") {
		t.Error("request 11 should be denied")
	}
}

func TestRateLimiter_Allow_DifferentIPs(t *testing.T) {
	rl := NewRateLimiter(10, 10)

	// Different IPs should have separate buckets
	if !rl.allow("192.168.1.1") {
		t.Error("IP1 should be allowed")
	}
	if !rl.allow("192.168.1.2") {
		t.Error("IP2 should be allowed")
	}
}

func TestRateLimiter_Allow_Refill(t *testing.T) {
	rl := NewRateLimiter(100, 10) // 100/s, burst 10

	// Exhaust burst
	for i := 0; i < 10; i++ {
		rl.allow("10.0.0.1")
	}
	if rl.allow("10.0.0.1") {
		t.Error("should be denied after burst")
	}

	// Wait for refill (100ms = 10 tokens at 100/s)
	time.Sleep(120 * time.Millisecond)

	if !rl.allow("10.0.0.1") {
		t.Error("should be allowed after refill")
	}
}

func TestRateLimiter_Concurrent(t *testing.T) {
	rl := NewRateLimiter(1000, 100)
	var wg sync.WaitGroup

	// 100 goroutines, each making 10 requests
	allowed := 0
	var mu sync.Mutex

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				if rl.allow("10.0.0.1") {
					mu.Lock()
					allowed++
					mu.Unlock()
				}
			}
		}()
	}
	wg.Wait()

	// Should allow roughly burst (100) + some refill
	if allowed < 100 {
		t.Errorf("allowed %d, expected at least 100", allowed)
	}
	if allowed > 200 {
		t.Errorf("allowed %d, expected at most 200", allowed)
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	rl := NewRateLimiter(10, 10)
	rl.cleanup = 50 * time.Millisecond // fast cleanup for testing

	rl.allow("old-ip")
	time.Sleep(100 * time.Millisecond)

	rl.mu.Lock()
	_, exists := rl.visitors["old-ip"]
	rl.mu.Unlock()

	if exists {
		t.Error("old visitor should be cleaned up")
	}
}
