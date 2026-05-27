package gpuinvoke

import (
	"sync"
	"time"
)

// projectRateLimiter implements a per-project sliding window rate limiter.
// Each project gets a fixed number of requests per window (e.g., 10 req/min).
type projectRateLimiter struct {
	mu       sync.Mutex
	windows  map[string]*window
	limit    int
	interval time.Duration
}

type window struct {
	count    int
	resetAt  time.Time
}

func newProjectRateLimiter(limit int, interval time.Duration) *projectRateLimiter {
	return &projectRateLimiter{
		windows:  make(map[string]*window),
		limit:    limit,
		interval: interval,
	}
}

// allow returns true if the request is within the rate limit for the given project.
func (rl *projectRateLimiter) allow(projectID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	w, ok := rl.windows[projectID]
	if !ok || now.After(w.resetAt) {
		rl.windows[projectID] = &window{
			count:   1,
			resetAt: now.Add(rl.interval),
		}
		return true
	}

	if w.count >= rl.limit {
		return false
	}
	w.count++
	return true
}
