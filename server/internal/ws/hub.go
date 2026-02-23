package ws

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sync"
	"sync/atomic"
	"unicode/utf8"

	"github.com/coder/websocket"
	"github.com/vladimirvolkov/basketball/server/internal/middleware"
)

// nicknameRe allows letters, digits, underscore, dash, spaces, cyrillic.
var nicknameRe = regexp.MustCompile(`^[a-zA-Z0-9_\- \x{0400}-\x{04FF}]+$`)

const maxActiveRooms = 100

// sanitizeNickname validates and cleans a nickname.
// Strips invalid chars, enforces 2-12 rune length, ensures valid UTF-8.
func sanitizeNickname(raw string) string {
	if !utf8.ValidString(raw) {
		return "Player"
	}
	// Strip characters not matching the allowed set
	cleaned := []rune{}
	for _, r := range raw {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-' || r == ' ' ||
			(r >= 0x0400 && r <= 0x04FF) {
			cleaned = append(cleaned, r)
		}
	}
	if len(cleaned) < 2 {
		return "Player"
	}
	if len(cleaned) > 12 {
		cleaned = cleaned[:12]
	}
	return string(cleaned)
}

type RoomCreator interface {
	CreateRoom(p1, p2 *Conn)
}

// HubStats holds live server metrics.
type HubStats struct {
	ActiveRooms      int64  `json:"activeRooms"`
	TotalConnections uint64 `json:"totalConnections"`
	WaitingPlayers   int    `json:"waitingPlayers"`
}

type Hub struct {
	mu      sync.Mutex
	waiting *Conn
	creator RoomCreator
	nextID  atomic.Uint64

	activeRooms      atomic.Int64
	totalConnections atomic.Uint64

	limiter        *middleware.IPRateLimiter
	originPatterns []string
}

func NewHub(creator RoomCreator, limiter *middleware.IPRateLimiter, originPatterns []string) *Hub {
	return &Hub{
		creator:        creator,
		limiter:        limiter,
		originPatterns: originPatterns,
	}
}

// Stats returns a snapshot of current server metrics.
func (h *Hub) Stats() HubStats {
	h.mu.Lock()
	w := 0
	if h.waiting != nil {
		w = 1
	}
	h.mu.Unlock()
	return HubStats{
		ActiveRooms:      h.activeRooms.Load(),
		TotalConnections: h.totalConnections.Load(),
		WaitingPlayers:   w,
	}
}

// RoomEnded decrements the active room counter. Call when a room goroutine exits.
func (h *Hub) RoomEnded() {
	h.activeRooms.Add(-1)
}

func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	// Rate limit: check per-IP connection limit
	ip := h.limiter.RealIP(r)
	if h.limiter != nil && !h.limiter.ConnectAllowed(ip) {
		http.Error(w, "too many connections", http.StatusTooManyRequests)
		return
	}

	acceptOpts := &websocket.AcceptOptions{}
	if len(h.originPatterns) > 0 {
		acceptOpts.OriginPatterns = h.originPatterns
	}

	ws, err := websocket.Accept(w, r, acceptOpts)
	if err != nil {
		if h.limiter != nil {
			h.limiter.Disconnect(ip)
		}
		log.Printf("ws accept error: %v", err)
		return
	}

	// Limit incoming message size (game messages are <500 bytes)
	ws.SetReadLimit(1024)

	h.totalConnections.Add(1)
	id := fmt.Sprintf("player-%d", h.nextID.Add(1))
	conn := NewConn(ws, id, ip, h.limiter)

	// Parse and sanitize nickname from query parameter
	nickname := sanitizeNickname(r.URL.Query().Get("name"))
	conn.Nickname = nickname
	log.Printf("new connection: %s [%s] from %s (total: %d)", id, nickname, ip, h.totalConnections.Load())

	// Use background context so connection lives beyond HTTP handler
	go conn.WriteLoop(context.Background())

	// Decrement rate limiter on disconnect
	go func() {
		<-conn.Done()
		if h.limiter != nil {
			h.limiter.Disconnect(ip)
		}
	}()

	h.tryMatch(conn)

	// Block until the connection is closed — keeps HTTP handler alive
	// which keeps the underlying TCP connection open for WebSocket
	<-conn.Done()
	log.Printf("connection closed: %s", id)
}

func (h *Hub) tryMatch(conn *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.waiting == nil {
		h.waiting = conn
		log.Printf("%s waiting for opponent", conn.ID)

		// If this player disconnects while waiting, clean up
		go func() {
			<-conn.Done()
			h.mu.Lock()
			if h.waiting == conn {
				h.waiting = nil
				log.Printf("%s disconnected while waiting", conn.ID)
			}
			h.mu.Unlock()
		}()
		return
	}

	// Limit active rooms to prevent resource exhaustion
	if h.activeRooms.Load() >= maxActiveRooms {
		log.Printf("max rooms reached, rejecting %s", conn.ID)
		go func() {
			conn.ws.Close(websocket.StatusTryAgainLater, "server full")
		}()
		return
	}

	// If duplicate nickname, append "(2)" — trim to stay within 12 runes
	if h.waiting.Nickname == conn.Nickname {
		suffix := "(2)"
		runes := []rune(conn.Nickname)
		maxBase := 12 - len([]rune(suffix))
		if len(runes) > maxBase {
			runes = runes[:maxBase]
		}
		conn.Nickname = string(runes) + suffix
		log.Printf("renamed duplicate nickname to %q for %s", conn.Nickname, conn.ID)
	}

	opponent := h.waiting
	h.waiting = nil

	h.activeRooms.Add(1)
	log.Printf("matched %s [%s] vs %s [%s] (rooms: %d)", opponent.ID, opponent.Nickname, conn.ID, conn.Nickname, h.activeRooms.Load())
	h.creator.CreateRoom(opponent, conn)
}
