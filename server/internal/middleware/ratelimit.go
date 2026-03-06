package middleware

import (
	"hash/fnv"
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
	lastSeen    time.Time
}

const numShards = 32

type shard struct {
	mu       sync.Mutex
	visitors map[string]*visitor
}

// IPRateLimiter tracks per-IP connection counts and message rates.
// Uses sharded locks so different IPs rarely contend on the same mutex.
type IPRateLimiter struct {
	shards [numShards]shard

	maxConnsPerIP int
	maxVisitors   int // per-shard max
	msgRate       int
	msgWindow     time.Duration
	trustProxy    bool
}

// NewIPRateLimiter creates a rate limiter.
//   - maxConnsPerIP: max simultaneous WebSocket connections per IP
//   - msgRate: max messages allowed per msgWindow
//   - msgWindow: time window for message rate
func NewIPRateLimiter(maxConnsPerIP, msgRate int, msgWindow time.Duration, trustProxy bool) *IPRateLimiter {
	rl := &IPRateLimiter{
		maxConnsPerIP: maxConnsPerIP,
		maxVisitors:   10000 / numShards, // spread across shards
		msgRate:       msgRate,
		msgWindow:     msgWindow,
		trustProxy:    trustProxy,
	}
	for i := range rl.shards {
		rl.shards[i].visitors = make(map[string]*visitor)
	}
	go rl.cleanup()
	return rl
}

// shardFor returns the shard for a given IP using FNV hash.
func (rl *IPRateLimiter) shardFor(ip string) *shard {
	h := fnv.New32a()
	h.Write([]byte(ip))
	return &rl.shards[h.Sum32()%numShards]
}

// ConnectAllowed checks if an IP can open a new connection.
// If allowed, increments the connection count and returns true.
func (rl *IPRateLimiter) ConnectAllowed(ip string) bool {
	s := rl.shardFor(ip)
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	v, ok := s.visitors[ip]
	if !ok {
		// Reject if this shard is at capacity (prevents memory exhaustion)
		if len(s.visitors) >= rl.maxVisitors {
			return false
		}
		s.visitors[ip] = &visitor{
			connections: 1,
			tokens:      rl.msgRate,
			lastRefill:  now,
			lastSeen:    now,
		}
		return true
	}
	if v.connections >= rl.maxConnsPerIP {
		return false
	}
	v.connections++
	v.lastSeen = now
	return true
}

// Disconnect decrements the connection count for an IP.
func (rl *IPRateLimiter) Disconnect(ip string) {
	s := rl.shardFor(ip)
	s.mu.Lock()
	defer s.mu.Unlock()

	v, ok := s.visitors[ip]
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
	s := rl.shardFor(ip)
	s.mu.Lock()
	defer s.mu.Unlock()

	v, ok := s.visitors[ip]
	if !ok {
		// No tracked visitor — allow but create entry
		s.visitors[ip] = &visitor{
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

// cleanup removes stale entries every minute.
// Entries with no active connections and not seen for 2+ minutes are removed.
func (rl *IPRateLimiter) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		for i := range rl.shards {
			s := &rl.shards[i]
			s.mu.Lock()
			for ip, v := range s.visitors {
				if v.connections <= 0 && now.Sub(v.lastSeen) > 2*time.Minute {
					delete(s.visitors, ip)
				}
			}
			s.mu.Unlock()
		}
	}
}

// RealIP extracts the client IP from the request.
// Only trusts X-Forwarded-For when trustProxy is true (server behind Railway/nginx).
func (rl *IPRateLimiter) RealIP(r *http.Request) string {
	if rl.trustProxy {
		// Take the LAST entry before the proxy — rightmost is added by trusted proxy
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			// Use first IP (client IP added by the outermost trusted proxy)
			ip := strings.TrimSpace(parts[0])
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
