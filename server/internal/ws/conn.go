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
	limiter  *middleware.IPRateLimiter
}

func NewConn(ws *websocket.Conn, id string, ip string, limiter *middleware.IPRateLimiter) *Conn {
	return &Conn{
		ws:      ws,
		sendCh:  make(chan []byte, 64),
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

func (c *Conn) ReadLoop(ctx context.Context) <-chan Message {
	ch := make(chan Message, 64)
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
			ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := c.ws.Write(ctx2, websocket.MessageText, data)
			cancel()
			if err != nil {
				log.Printf("conn %s: write error: %v", c.ID, err)
				c.Close()
				return
			}
		case <-c.done:
			return
		case <-ctx.Done():
			return
		}
	}
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
