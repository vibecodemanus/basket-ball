package ws

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/coder/websocket"
)

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
}

func NewHub(creator RoomCreator) *Hub {
	return &Hub{creator: creator}
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
	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("ws accept error: %v", err)
		return
	}

	h.totalConnections.Add(1)
	id := fmt.Sprintf("player-%d", h.nextID.Add(1))
	conn := NewConn(ws, id)

	// Parse nickname from query parameter
	nickname := r.URL.Query().Get("name")
	if nickname == "" {
		nickname = "Player"
	}
	// Limit to 12 runes
	runes := []rune(nickname)
	if len(runes) > 12 {
		nickname = string(runes[:12])
	}
	conn.Nickname = nickname
	log.Printf("new connection: %s [%s] (total: %d)", id, nickname, h.totalConnections.Load())

	// Use background context so connection lives beyond HTTP handler
	go conn.WriteLoop(context.Background())

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

	// Reject duplicate nickname
	if h.waiting.Nickname == conn.Nickname {
		log.Printf("rejecting duplicate nickname %q from %s", conn.Nickname, conn.ID)
		go func() {
			conn.ws.Close(websocket.StatusPolicyViolation, "duplicate nickname")
		}()
		return
	}

	opponent := h.waiting
	h.waiting = nil

	h.activeRooms.Add(1)
	log.Printf("matched %s [%s] vs %s [%s] (rooms: %d)", opponent.ID, opponent.Nickname, conn.ID, conn.Nickname, h.activeRooms.Load())
	h.creator.CreateRoom(opponent, conn)
}
