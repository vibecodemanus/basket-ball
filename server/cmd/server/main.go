package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/vladimirvolkov/basketball/server/internal/game"
	"github.com/vladimirvolkov/basketball/server/internal/middleware"
	"github.com/vladimirvolkov/basketball/server/internal/ws"
)

// securityHeaders wraps a handler with common security response headers.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; connect-src 'self' ws: wss:; img-src 'self' data:")
		next.ServeHTTP(w, r)
	})
}

type GameManager struct {
	hub *ws.Hub
}

func (gm *GameManager) CreateRoom(p1, p2 *ws.Conn) {
	room := game.NewRoom(p1, p2)
	room.Start(context.Background())
	go func() {
		<-room.Done()
		gm.hub.RoomEnded()
	}()
}

func main() {
	// Write logs to stdout so Railway doesn't mark them as errors
	log.SetOutput(os.Stdout)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "../client/dist"
	}

	// Parse allowed origins for WebSocket from environment
	var originPatterns []string
	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		originPatterns = strings.Split(origins, ",")
	}

	// Create rate limiter: max 4 conns/IP, 120 msgs/sec/IP
	limiter := middleware.NewIPRateLimiter(4, 120, time.Second)

	manager := &GameManager{}
	hub := ws.NewHub(manager, limiter, originPatterns)
	manager.hub = hub

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.HandleWS)

	// Health / stats endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		stats := hub.Stats()
		json.NewEncoder(w).Encode(stats)
	})

	// Static files with no-cache headers (prevents stale JS in browser)
	fs := http.FileServer(http.Dir(staticDir))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		fs.ServeHTTP(w, r)
	}))

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 16, // 64KB
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down...")
		server.Close()
	}()

	log.Printf("Pixel Basketball server starting on :%s", port)
	log.Printf("serving static files from %s", staticDir)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
	log.Println("server stopped")
}
