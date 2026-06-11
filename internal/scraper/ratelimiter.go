package scraper

import (
	"context"
	"golang.org/x/time/rate"
	"sync"
)

type DomainRateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.Mutex
	defaultLimit rate.Limit
	defaultBurst int
}

func NewDomainRateLimiter(requestsPerSecond float64, burst int) *DomainRateLimiter {
	return &DomainRateLimiter{
		limiters:     make(map[string]*rate.Limiter),
		defaultLimit: rate.Limit(requestsPerSecond),
		defaultBurst: burst,
	}
}

func (d *DomainRateLimiter) GetLimiter(domain string) *rate.Limiter {
	d.mu.Lock()
	defer d.mu.Unlock()

	if limiter, exists := d.limiters[domain]; exists {
		return limiter
	}

	limiter := rate.NewLimiter(d.defaultLimit, d.defaultBurst)
	d.limiters[domain] = limiter
	return limiter
}

func (d *DomainRateLimiter) Wait(domain string) error {
	return d.GetLimiter(domain).Wait(context.Background())
}