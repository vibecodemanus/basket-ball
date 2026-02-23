package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/vladimirvolkov/basketball/server/internal/game"
	"github.com/vladimirvolkov/basketball/server/internal/ws"
)

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
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "../client/dist"
	}

	manager := &GameManager{}
	hub := ws.NewHub(manager)
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
		Addr:    ":" + port,
		Handler: mux,
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
