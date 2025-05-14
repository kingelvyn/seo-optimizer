package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type RateLimiter struct {
	tokens         map[string]float64
	lastRefill     map[string]time.Time
	mu             sync.Mutex
	rate           float64  // tokens per second
	bucketSize     float64  // maximum tokens
	refillInterval time.Duration
}

func NewRateLimiter(rate float64, bucketSize float64) *RateLimiter {
	return &RateLimiter{
		tokens:         make(map[string]float64),
		lastRefill:     make(map[string]time.Time),
		rate:           rate,
		bucketSize:     bucketSize,
		refillInterval: time.Second,
	}
}

func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		rl.mu.Lock()
		now := time.Now()

		// Initialize if first request
		if _, exists := rl.lastRefill[ip]; !exists {
			rl.tokens[ip] = rl.bucketSize
			rl.lastRefill[ip] = now
		}

		// Refill tokens based on time elapsed
		elapsed := now.Sub(rl.lastRefill[ip])
		newTokens := float64(elapsed) / float64(rl.refillInterval) * rl.rate
		rl.tokens[ip] = min(rl.bucketSize, rl.tokens[ip]+newTokens)
		rl.lastRefill[ip] = now

		// Check if we have enough tokens
		if rl.tokens[ip] < 1 {
			rl.mu.Unlock()
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded. Please try again later.",
			})
			c.Abort()
			return
		}

		// Consume one token
		rl.tokens[ip]--
		rl.mu.Unlock()

		c.Next()
	}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
} 