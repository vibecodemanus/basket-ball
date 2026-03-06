package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
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
	hub        *ws.Hub
	tournament *game.Tournament
	engine     *game.Engine
}

func (gm *GameManager) CreateRoom(p1, p2 *ws.Conn) {
	room := game.NewRoom(p1, p2)
	room.Start(context.Background())
	gm.engine.AddRoom(room)
	go func() {
		<-room.Done()
		gm.hub.RoomEnded()
	}()
}

func (gm *GameManager) CreateTournamentRoom(p1, p2 *ws.Conn) {
	room := game.NewTournamentRoom(p1, p2, gm.tournament)
	room.Start(context.Background())
	gm.engine.AddRoom(room)
	go func() {
		<-room.Done()
		gm.hub.RoomEnded()
	}()
}

func main() {
	// Use all available CPU cores for game loop parallelism
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Reduce GC frequency — default GOGC=100 triggers too often at 60 Hz × 100 rooms,
	// causing stop-the-world pauses that lag ALL connections simultaneously.
	// GOGC=400 lets heap grow 4× before collecting, trading memory for lower latency.
	debug.SetGCPercent(400)

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

	// Trust X-Forwarded-For only when behind a reverse proxy (Railway, nginx)
	trustProxy := os.Getenv("TRUST_PROXY") == "true"

	// Create rate limiter: max 200 conns/IP (100 rooms × 2 players), 300 msgs/sec/IP
	// Higher msg rate avoids dropping legitimate input at 60 Hz from multiple connections
	limiter := middleware.NewIPRateLimiter(200, 300, time.Second, trustProxy)

	// Multi-core game engine: one worker per CPU core, each pinned to an OS thread.
	// All game rooms are distributed across workers and ticked in parallel at 60 Hz.
	engine := game.NewEngine(runtime.NumCPU())
	engine.Start(context.Background())

	tournament := game.NewTournament()
	manager := &GameManager{tournament: tournament, engine: engine}
	hub := ws.NewHub(manager, limiter, originPatterns, tournament)
	manager.hub = hub

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.HandleWS)

	// Health / stats endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		stats := hub.Stats()
		json.NewEncoder(w).Encode(stats)
	})

	// Tournament leaderboard endpoint
	mux.HandleFunc("/tournament/leaderboard", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		sortBy := r.URL.Query().Get("sort")
		limit := 20

		var entries []game.LeaderboardEntry
		if sortBy == "points" {
			entries = tournament.LeaderboardByPoints(limit)
		} else {
			entries = tournament.LeaderboardByWins(limit)
		}
		json.NewEncoder(w).Encode(entries)
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
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      0, // disable for long-lived WebSocket connections
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
