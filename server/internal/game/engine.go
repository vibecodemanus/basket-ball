package game

import (
	"context"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Engine distributes game rooms across CPU-pinned worker goroutines.
// Instead of each room running its own goroutine with its own timer (100+
// goroutines competing for scheduling), the Engine uses exactly NumCPU workers,
// each pinned to an OS thread via LockOSThread, processing rooms in parallel
// batches at a single synchronized 60 Hz tick.
type Engine struct {
	workers []*gameWorker
	next    atomic.Uint64
}

type gameWorker struct {
	mu    sync.Mutex
	rooms []*Room
}

// NewEngine creates a game engine with one worker per CPU core.
func NewEngine(numWorkers int) *Engine {
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}
	e := &Engine{
		workers: make([]*gameWorker, numWorkers),
	}
	for i := range e.workers {
		e.workers[i] = &gameWorker{}
	}
	log.Printf("game engine created with %d workers", numWorkers)
	return e
}

// Start launches all workers. Each worker pins itself to a dedicated OS thread.
func (e *Engine) Start(ctx context.Context) {
	for i, w := range e.workers {
		go w.run(ctx, i)
	}
}

// AddRoom assigns a room to the least-loaded worker (round-robin).
func (e *Engine) AddRoom(r *Room) {
	idx := e.next.Add(1) % uint64(len(e.workers))
	w := e.workers[idx]
	w.mu.Lock()
	w.rooms = append(w.rooms, r)
	w.mu.Unlock()
}

func (w *gameWorker) run(ctx context.Context, id int) {
	// Pin to a dedicated OS thread — eliminates goroutine scheduling jitter
	// and gives each worker consistent CPU cache access.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	const interval = time.Second / TickRate
	next := time.Now().Add(interval)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		now := time.Now()
		if now.Before(next) {
			time.Sleep(next.Sub(now))
		}

		w.tickAll()
		next = next.Add(interval)

		// Skip ahead after a stall (GC, OS scheduling) to avoid burst
		if time.Now().After(next.Add(2 * interval)) {
			next = time.Now().Add(interval)
		}
	}
}

func (w *gameWorker) tickAll() {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Tick all rooms; remove finished ones in-place.
	alive := w.rooms[:0]
	for _, r := range w.rooms {
		if r.TickExternal() {
			alive = append(alive, r)
		} else {
			close(r.done)
		}
	}
	// Clear tail references so GC can collect removed rooms
	for i := len(alive); i < len(w.rooms); i++ {
		w.rooms[i] = nil
	}
	w.rooms = alive
}
