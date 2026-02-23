package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type visitor struct {
	connections int
	tokens      int
	lastRefill  time.Time
}

// IPRateLimiter tracks per-IP connection counts and message rates.
type IPRateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor

	maxConnsPerIP int
	msgRate       int
	msgWindow     time.Duration
}

// NewIPRateLimiter creates a rate limiter.
//   - maxConnsPerIP: max simultaneous WebSocket connections per IP
//   - msgRate: max messages allowed per msgWindow
//   - msgWindow: time window for message rate
func NewIPRateLimiter(maxConnsPerIP, msgRate int, msgWindow time.Duration) *IPRateLimiter {
	rl := &IPRateLimiter{
		visitors:      make(map[string]*visitor),
		maxConnsPerIP: maxConnsPerIP,
		msgRate:       msgRate,
		msgWindow:     msgWindow,
	}
	go rl.cleanup()
	return rl
}

// ConnectAllowed checks if an IP can open a new connection.
// If allowed, increments the connection count and returns true.
func (rl *IPRateLimiter) ConnectAllowed(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, ok := rl.visitors[ip]
	if !ok {
		rl.visitors[ip] = &visitor{
			connections: 1,
			tokens:      rl.msgRate,
			lastRefill:  time.Now(),
		}
		return true
	}
	if v.connections >= rl.maxConnsPerIP {
		return false
	}
	v.connections++
	return true
}

// Disconnect decrements the connection count for an IP.
func (rl *IPRateLimiter) Disconnect(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, ok := rl.visitors[ip]
	if !ok {
		return
	}
	v.connections--
	if v.connections < 0 {
		v.connections = 0
	}
}

// MessageAllowed checks if a message from this IP is within rate limits.
// Uses token bucket: refills msgRate tokens per msgWindow.
func (rl *IPRateLimiter) MessageAllowed(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, ok := rl.visitors[ip]
	if !ok {
		// No tracked visitor — allow but create entry
		rl.visitors[ip] = &visitor{
			tokens:     rl.msgRate - 1,
			lastRefill: time.Now(),
		}
		return true
	}

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(v.lastRefill)
	if elapsed >= rl.msgWindow {
		windows := int(elapsed / rl.msgWindow)
		v.tokens += windows * rl.msgRate
		if v.tokens > rl.msgRate {
			v.tokens = rl.msgRate
		}
		v.lastRefill = v.lastRefill.Add(time.Duration(windows) * rl.msgWindow)
	}

	if v.tokens <= 0 {
		return false
	}
	v.tokens--
	return true
}

// cleanup removes stale entries (no connections) every 5 minutes.
func (rl *IPRateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if v.connections <= 0 {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// RealIP extracts the client IP from the request.
// Checks X-Forwarded-For (for reverse proxies like Fly.io) then RemoteAddr.
func RealIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if comma := strings.Index(xff, ","); comma > 0 {
			return strings.TrimSpace(xff[:comma])
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
