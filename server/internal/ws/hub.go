package ws

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sync"
	"sync/atomic"
	"time"
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

// deduplicateNickname appends "(2)" if nicknames match.
func deduplicateNickname(existing, incoming string) string {
	if existing != incoming {
		return incoming
	}
	suffix := "(2)"
	runes := []rune(incoming)
	maxBase := 12 - len([]rune(suffix))
	if len(runes) > maxBase {
		runes = runes[:maxBase]
	}
	return string(runes) + suffix
}

// TournamentMatcher provides tournament history lookups (breaks import cycle with game package).
type TournamentMatcher interface {
	HavePlayedBefore(nick1, nick2 string) bool
	TimesPlayed(nick1, nick2 string) int
}

type RoomCreator interface {
	CreateRoom(p1, p2 *Conn)
	CreateTournamentRoom(p1, p2 *Conn)
}

// HubStats holds live server metrics.
type HubStats struct {
	ActiveRooms         int64  `json:"activeRooms"`
	TotalConnections    uint64 `json:"totalConnections"`
	WaitingPlayers      int    `json:"waitingPlayers"`
	TournamentQueueSize int    `json:"tournamentQueueSize"`
}

type tournamentEntry struct {
	conn     *Conn
	joinedAt time.Time
}

type Hub struct {
	mu      sync.Mutex
	waiting *Conn
	creator RoomCreator
	nextID  atomic.Uint64

	// Tournament
	tournamentQueue []*tournamentEntry
	tournament      TournamentMatcher

	activeRooms      atomic.Int64
	totalConnections atomic.Uint64

	limiter        *middleware.IPRateLimiter
	originPatterns []string
}

func NewHub(creator RoomCreator, limiter *middleware.IPRateLimiter, originPatterns []string, tournament TournamentMatcher) *Hub {
	return &Hub{
		creator:        creator,
		limiter:        limiter,
		originPatterns: originPatterns,
		tournament:     tournament,
	}
}

// Stats returns a snapshot of current server metrics.
func (h *Hub) Stats() HubStats {
	h.mu.Lock()
	w := 0
	if h.waiting != nil {
		w = 1
	}
	tq := len(h.tournamentQueue)
	h.mu.Unlock()
	return HubStats{
		ActiveRooms:         h.activeRooms.Load(),
		TotalConnections:    h.totalConnections.Load(),
		WaitingPlayers:      w,
		TournamentQueueSize: tq,
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

	// Parse game mode
	mode := r.URL.Query().Get("mode")
	conn.Mode = mode

	log.Printf("new connection: %s [%s] mode=%s from %s (total: %d)", id, nickname, mode, ip, h.totalConnections.Load())

	// Use background context so connection lives beyond HTTP handler
	go conn.WriteLoop(context.Background())

	// Decrement rate limiter on disconnect
	go func() {
		<-conn.Done()
		if h.limiter != nil {
			h.limiter.Disconnect(ip)
		}
	}()

	// Route to appropriate matchmaking
	if mode == "tournament" {
		h.tryTournamentMatch(conn)
	} else {
		h.tryMatch(conn)
	}

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

	conn.Nickname = deduplicateNickname(h.waiting.Nickname, conn.Nickname)

	opponent := h.waiting
	h.waiting = nil

	h.activeRooms.Add(1)
	log.Printf("matched %s [%s] vs %s [%s] (rooms: %d)", opponent.ID, opponent.Nickname, conn.ID, conn.Nickname, h.activeRooms.Load())
	h.creator.CreateRoom(opponent, conn)
}

// ── Tournament matchmaking ──

func (h *Hub) tryTournamentMatch(conn *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Try to find an opponent this player hasn't played before
	for i, candidate := range h.tournamentQueue {
		if candidate.conn == conn {
			continue
		}
		if !h.tournament.HavePlayedBefore(conn.Nickname, candidate.conn.Nickname) {
			// New opponent found — match them
			h.tournamentQueue = append(h.tournamentQueue[:i], h.tournamentQueue[i+1:]...)
			h.startTournamentRoom(candidate.conn, conn)
			return
		}
	}

	// No new opponent available — add to queue
	entry := &tournamentEntry{conn: conn, joinedAt: time.Now()}
	h.tournamentQueue = append(h.tournamentQueue, entry)
	log.Printf("%s [%s] waiting in tournament queue (size: %d)", conn.ID, conn.Nickname, len(h.tournamentQueue))

	// Clean up on disconnect
	go func() {
		<-conn.Done()
		h.mu.Lock()
		for i, e := range h.tournamentQueue {
			if e.conn == conn {
				h.tournamentQueue = append(h.tournamentQueue[:i], h.tournamentQueue[i+1:]...)
				log.Printf("%s disconnected from tournament queue", conn.ID)
				break
			}
		}
		h.mu.Unlock()
	}()

	// 20-second fallback: allow rematch if no new opponent appears
	go func() {
		timer := time.NewTimer(20 * time.Second)
		defer timer.Stop()

		select {
		case <-timer.C:
			h.tryTournamentRematch(conn)
		case <-conn.Done():
			return
		}
	}()
}

func (h *Hub) tryTournamentRematch(conn *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Find conn in queue
	connIdx := -1
	for i, e := range h.tournamentQueue {
		if e.conn == conn {
			connIdx = i
			break
		}
	}
	if connIdx == -1 {
		return // already matched or disconnected
	}

	// Find any other player in queue, preferring least-played opponent
	bestIdx := -1
	bestCount := int(^uint(0) >> 1) // max int
	for i, candidate := range h.tournamentQueue {
		if i == connIdx {
			continue
		}
		count := h.tournament.TimesPlayed(conn.Nickname, candidate.conn.Nickname)
		if count < bestCount {
			bestCount = count
			bestIdx = i
		}
	}

	if bestIdx == -1 {
		return // still alone in queue
	}

	opponent := h.tournamentQueue[bestIdx]

	// Remove both from queue (higher index first to avoid shifting issues)
	if bestIdx > connIdx {
		h.tournamentQueue = append(h.tournamentQueue[:bestIdx], h.tournamentQueue[bestIdx+1:]...)
		h.tournamentQueue = append(h.tournamentQueue[:connIdx], h.tournamentQueue[connIdx+1:]...)
	} else {
		h.tournamentQueue = append(h.tournamentQueue[:connIdx], h.tournamentQueue[connIdx+1:]...)
		h.tournamentQueue = append(h.tournamentQueue[:bestIdx], h.tournamentQueue[bestIdx+1:]...)
	}

	h.startTournamentRoom(opponent.conn, conn)
}

func (h *Hub) startTournamentRoom(p1, p2 *Conn) {
	if h.activeRooms.Load() >= maxActiveRooms {
		log.Printf("max rooms reached, rejecting tournament match %s vs %s", p1.ID, p2.ID)
		go func() {
			p1.ws.Close(websocket.StatusTryAgainLater, "server full")
			p2.ws.Close(websocket.StatusTryAgainLater, "server full")
		}()
		return
	}

	p2.Nickname = deduplicateNickname(p1.Nickname, p2.Nickname)

	h.activeRooms.Add(1)
	log.Printf("tournament matched %s [%s] vs %s [%s] (rooms: %d)", p1.ID, p1.Nickname, p2.ID, p2.Nickname, h.activeRooms.Load())
	h.creator.CreateTournamentRoom(p1, p2)
}
