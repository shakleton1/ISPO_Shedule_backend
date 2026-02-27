package httpapi

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"ispo-schedule/internal/config"
)

type rateLimitStore struct {
	mu          sync.Mutex
	clients     map[string]*rateLimitClient
	clientTTL   time.Duration
	nextCleanup time.Time
}

type rateLimitClient struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func newRateLimitStore(clientTTL time.Duration) *rateLimitStore {
	return &rateLimitStore{
		clients:     map[string]*rateLimitClient{},
		clientTTL:   clientTTL,
		nextCleanup: time.Now().Add(clientTTL),
	}
}

func (s *rateLimitStore) get(key string, rps float64, burst int) *rate.Limiter {
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	if now.After(s.nextCleanup) {
		for k, c := range s.clients {
			if now.Sub(c.lastSeen) > s.clientTTL {
				delete(s.clients, k)
			}
		}
		s.nextCleanup = now.Add(s.clientTTL)
	}

	c := s.clients[key]
	if c == nil {
		c = &rateLimitClient{limiter: rate.NewLimiter(rate.Limit(rps), burst)}
		s.clients[key] = c
	}
	c.lastSeen = now
	return c.limiter
}

func rateLimitMiddleware(store *rateLimitStore, rule config.RateLimitRuleConfig, bucketName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rule.Enabled {
			c.Next()
			return
		}
		if rule.RPS <= 0 || rule.Burst <= 0 {
			c.Next()
			return
		}

		key := bucketName + "|" + c.ClientIP()
		lim := store.get(key, rule.RPS, rule.Burst)
		if !lim.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}
