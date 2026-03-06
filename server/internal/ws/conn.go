package ws

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/vladimirvolkov/basketball/server/internal/middleware"
)

type Conn struct {
	ws       *websocket.Conn
	sendCh   chan []byte
	done     chan struct{}
	once     sync.Once
	ID       string
	Nickname string
	IP       string
	Mode     string // "" for regular, "tournament" for tournament
	limiter  *middleware.IPRateLimiter
}

func NewConn(ws *websocket.Conn, id string, ip string, limiter *middleware.IPRateLimiter) *Conn {
	return &Conn{
		ws:      ws,
		sendCh:  make(chan []byte, 512),
		done:    make(chan struct{}),
		ID:      id,
		IP:      ip,
		limiter: limiter,
	}
}

func (c *Conn) Send(msg Message) {
	data, err := Encode(msg)
	if err != nil {
		log.Printf("conn %s: encode error: %v", c.ID, err)
		return
	}
	select {
	case c.sendCh <- data:
	default:
		log.Printf("conn %s: send buffer full, dropping message", c.ID)
	}
}

// SendRaw sends pre-encoded bytes directly, skipping per-connection encoding.
// Use for broadcast messages that are encoded once and sent to multiple connections.
func (c *Conn) SendRaw(data []byte) {
	select {
	case c.sendCh <- data:
	default:
		log.Printf("conn %s: send buffer full, dropping message", c.ID)
	}
}

func (c *Conn) ReadLoop(ctx context.Context) <-chan Message {
	ch := make(chan Message, 128)
	go func() {
		defer close(ch)
		for {
			_, data, err := c.ws.Read(ctx)
			if err != nil {
				log.Printf("conn %s: read error: %v", c.ID, err)
				c.Close()
				return
			}
			// Per-IP message rate limiting
			if c.limiter != nil && !c.limiter.MessageAllowed(c.IP) {
				continue // drop message silently, don't disconnect
			}
			msg, err := Decode(data)
			if err != nil {
				log.Printf("conn %s: decode error: %v", c.ID, err)
				continue
			}
			select {
			case ch <- msg:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch
}

func (c *Conn) WriteLoop(ctx context.Context) {
	for {
		select {
		case data := <-c.sendCh:
			if err := c.writeWithTimeout(ctx, data); err != nil {
				return
			}
			// Drain queued messages in burst to catch up after any brief delay.
			// This reduces syscall overhead by writing back-to-back without
			// re-entering the select.
			if err := c.drainSendCh(ctx); err != nil {
				return
			}
		case <-c.done:
			return
		case <-ctx.Done():
			return
		}
	}
}

// drainSendCh writes up to 8 queued messages without re-entering the select loop.
func (c *Conn) drainSendCh(ctx context.Context) error {
	for i := 0; i < 8; i++ {
		select {
		case data := <-c.sendCh:
			if err := c.writeWithTimeout(ctx, data); err != nil {
				return err
			}
		default:
			return nil
		}
	}
	return nil
}

// writeWithTimeout writes a single message with a short deadline.
func (c *Conn) writeWithTimeout(ctx context.Context, data []byte) error {
	ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
	err := c.ws.Write(ctx2, websocket.MessageText, data)
	cancel()
	if err != nil {
		log.Printf("conn %s: write error: %v", c.ID, err)
		c.Close()
		return err
	}
	return nil
}

func (c *Conn) Close() {
	c.once.Do(func() {
		close(c.done)
		c.ws.Close(websocket.StatusNormalClosure, "")
	})
}

func (c *Conn) Done() <-chan struct{} {
	return c.done
}
