package game

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/vladimirvolkov/basketball/server/internal/ws"
)

type Room struct {
	conns     [2]*ws.Conn
	nicknames [2]string
	state     GameState
	inputs    [2]PlayerInput
	inputMu   sync.Mutex
	cancel    context.CancelFunc
	done      chan struct{}
}

func NewRoom(p1, p2 *ws.Conn) *Room {
	r := &Room{
		conns:     [2]*ws.Conn{p1, p2},
		nicknames: [2]string{p1.Nickname, p2.Nickname},
	}
	r.state = GameState{
		Phase:      PhaseCountdown,
		PhaseTimer: CountdownSecs,
		Players: [2]PlayerState{
			NewPlayer(240, FloorY-PlayerHeight/2, 1),
			NewPlayer(720, FloorY-PlayerHeight/2, -1),
		},
		Ball:      NewBall(),
		ShotClock: ShotClockSecs,
		GameClock: GameDuration,
		Winner:    -1,
	}
	return r
}

func (r *Room) Start(ctx context.Context) {
	ctx, r.cancel = context.WithCancel(ctx)
	r.done = make(chan struct{})

	// Send GameStart to both players (includes both nicknames)
	for i, c := range r.conns {
		msg, _ := ws.NewMessage(ws.MsgGameStart, 0, ws.GameStartPayload{
			PlayerIndex: uint8(i),
			Names:       r.nicknames,
		})
		c.Send(msg)
	}

	// Start read loops
	for i, c := range r.conns {
		go r.readLoop(ctx, c, i)
	}

	// Start game loop (closes done channel on exit)
	go func() {
		r.gameLoop(ctx)
		close(r.done)
	}()
}

// Done returns a channel that closes when the room's game loop exits.
func (r *Room) Done() <-chan struct{} {
	return r.done
}

func (r *Room) readLoop(ctx context.Context, conn *ws.Conn, playerIdx int) {
	msgs := conn.ReadLoop(ctx)
	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				log.Printf("player %d disconnected", playerIdx)
				r.handleDisconnect(playerIdx)
				return
			}
			r.handleMessage(playerIdx, msg)
		case <-ctx.Done():
			return
		}
	}
}

func (r *Room) handleMessage(playerIdx int, msg ws.Message) {
	switch msg.Type {
	case ws.MsgPlayerInput:
		var input PlayerInput
		if err := json.Unmarshal(msg.Payload, &input); err != nil {
			return
		}
		// Clamp moveX to valid range [-1, 1]
		if input.MoveX < -1 {
			input.MoveX = -1
		}
		if input.MoveX > 1 {
			input.MoveX = 1
		}
		r.inputMu.Lock()
		r.inputs[playerIdx] = input
		r.inputMu.Unlock()

	case ws.MsgPing:
		var ping ws.PingPayload
		if err := json.Unmarshal(msg.Payload, &ping); err != nil {
			return
		}
		pong, _ := ws.NewMessage(ws.MsgPong, r.state.Tick, ws.PongPayload{
			ClientTime: ping.ClientTime,
			ServerTime: uint64(time.Now().UnixMilli()),
		})
		r.conns[playerIdx].Send(pong)

	case ws.MsgJoinQueue:
		// "Play Again" request — not implemented yet at server level
	}
}

func (r *Room) handleDisconnect(playerIdx int) {
	other := 1 - playerIdx
	msg, _ := ws.NewMessage(ws.MsgPlayerDisconnected, r.state.Tick, ws.PlayerDisconnectedPayload{
		PlayerIndex: uint8(playerIdx),
	})
	r.conns[other].Send(msg)
	r.cancel()
}

func (r *Room) gameLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second / TickRate)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.tick()
		case <-ctx.Done():
			return
		}
	}
}

func (r *Room) tick() {
	s := &r.state
	s.Tick++

	switch s.Phase {
	case PhaseCountdown:
		r.tickCountdown()
	case PhasePlaying:
		r.tickPlaying()
	case PhaseScored:
		r.tickScored()
	case PhaseGameOver:
		// No more updates; room stays alive for clients to see final state
	}

	r.broadcastState()
}

func (r *Room) tickCountdown() {
	s := &r.state
	s.PhaseTimer -= DT
	if s.PhaseTimer <= 0 {
		s.Phase = PhasePlaying
		s.PhaseTimer = 0
	}
}

func (r *Room) tickPlaying() {
	s := &r.state

	// Read inputs; consume one-shot actions (jump, shoot) but keep movement (moveX)
	r.inputMu.Lock()
	inputs := r.inputs
	for i := range r.inputs {
		r.inputs[i].Jump = false
		r.inputs[i].Shoot = false
	}
	r.inputMu.Unlock()

	// Apply inputs
	for i := range s.Players {
		ApplyInput(&s.Players[i], inputs[i])

		// Handle shooting — accuracy depends on position
		if inputs[i].Shoot && s.Players[i].HasBall {
			// Don't allow shooting from behind opponent's backboard
			canShoot := true
			if i == 0 && s.Players[i].X > RightHoop.BackboardX {
				canShoot = false
			}
			if i == 1 && s.Players[i].X < LeftHoop.BackboardX {
				canShoot = false
			}

			if canShoot {
				// Check for block by opponent
				otherIdx := 1 - i
				blocker := &s.Players[otherIdx]
				blocked := TryBlockShot(&s.Ball, &s.Players[i], int8(i), blocker)
				if !blocked {
					ShootBall(&s.Ball, &s.Players[i], int8(i))
				}
			}
		}
	}

	// Step physics
	for i := range s.Players {
		StepPlayer(&s.Players[i])
	}

	prevBallY := s.Ball.Y
	StepBall(&s.Ball, &s.Players)

	// Check scoring against both hoops
	if CheckBallHoop(&s.Ball, &RightHoop, prevBallY) {
		r.scored(0)
		return
	}
	if CheckBallHoop(&s.Ball, &LeftHoop, prevBallY) {
		r.scored(1)
		return
	}

	// Shot clock
	s.ShotClock -= DT
	if s.ShotClock <= 0 {
		r.shotClockViolation()
	}

	// Game clock
	s.GameClock -= DT
	if s.GameClock <= 0 {
		s.GameClock = 0
		r.gameOver()
	}
}

func (r *Room) tickScored() {
	s := &r.state
	s.PhaseTimer -= DT
	if s.PhaseTimer <= 0 {
		s.Phase = PhasePlaying
		s.PhaseTimer = 0
	}
}

func (r *Room) scored(playerIdx int) {
	s := &r.state

	// Determine points: 3 if shot from behind 3-point line, else 2
	var points uint8 = 2
	shotX := s.Ball.ShotOriginX
	if playerIdx == 0 {
		// P0 scores on right hoop (x=720). 3-point line at 720-150=570.
		if shotX < HoopRightX-ThreePointRadius {
			points = 3
		}
	} else {
		// P1 scores on left hoop (x=80). 3-point line at 80+150=230.
		if shotX > HoopLeftX+ThreePointRadius {
			points = 3
		}
	}
	s.Score[playerIdx] += points

	log.Printf("SCORED: player %d +%d pts (shot from x=%.1f)", playerIdx, points, shotX)

	// Enter scored pause phase
	s.Phase = PhaseScored
	s.PhaseTimer = ScoredPauseSecs

	// Reset ball ownership to other player
	otherIdx := 1 - playerIdx
	s.Ball = NewBall()
	s.Ball.Owner = int8(otherIdx)
	s.Players[otherIdx].HasBall = true
	s.Players[playerIdx].HasBall = false

	// Reset positions
	s.Players[0].X = 240
	s.Players[0].Y = FloorY - PlayerHeight/2
	s.Players[0].VX = 0
	s.Players[0].VY = 0
	s.Players[0].Anim = AnimIdle
	s.Players[1].X = 720
	s.Players[1].Y = FloorY - PlayerHeight/2
	s.Players[1].VX = 0
	s.Players[1].VY = 0
	s.Players[1].Anim = AnimIdle

	// Reset shot clock
	s.ShotClock = ShotClockSecs

	for _, c := range r.conns {
		msg, _ := ws.NewMessage(ws.MsgScored, s.Tick, struct {
			ScorerIndex uint8    `json:"scorerIndex"`
			Points      uint8    `json:"points"`
			NewScore    [2]uint8 `json:"newScore"`
		}{
			ScorerIndex: uint8(playerIdx),
			Points:      points,
			NewScore:    s.Score,
		})
		c.Send(msg)
	}
}

func (r *Room) shotClockViolation() {
	s := &r.state
	s.ShotClock = ShotClockSecs

	// Determine who had possession and give ball to the other player
	currentOwner := -1
	for i, p := range s.Players {
		if p.HasBall {
			currentOwner = i
		}
	}
	if currentOwner == -1 && s.Ball.Owner >= 0 {
		currentOwner = int(s.Ball.Owner)
	}

	// Turnover: give ball to the other player
	var newOwner int
	if currentOwner >= 0 {
		newOwner = 1 - currentOwner
	} else {
		newOwner = 0 // default to player 0
	}

	// Reset ball
	s.Ball = NewBall()
	s.Ball.Owner = int8(newOwner)

	// Reset player states
	s.Players[0].HasBall = newOwner == 0
	s.Players[1].HasBall = newOwner == 1
	s.Players[0].X = 240
	s.Players[0].Y = FloorY - PlayerHeight/2
	s.Players[0].VX = 0
	s.Players[0].VY = 0
	s.Players[1].X = 720
	s.Players[1].Y = FloorY - PlayerHeight/2
	s.Players[1].VX = 0
	s.Players[1].VY = 0
}

func (r *Room) gameOver() {
	s := &r.state
	s.Phase = PhaseGameOver
	s.PhaseTimer = 0

	// Determine winner
	if s.Score[0] > s.Score[1] {
		s.Winner = 0
	} else if s.Score[1] > s.Score[0] {
		s.Winner = 1
	} else {
		s.Winner = -1 // tie
	}

	// Send game over message
	for _, c := range r.conns {
		msg, _ := ws.NewMessage(ws.MsgGameOver, s.Tick, struct {
			Winner int8     `json:"winner"`
			Score  [2]uint8 `json:"score"`
		}{
			Winner: s.Winner,
			Score:  s.Score,
		})
		c.Send(msg)
	}
}

func (r *Room) broadcastState() {
	msg, err := ws.NewMessage(ws.MsgGameState, r.state.Tick, r.state)
	if err != nil {
		log.Printf("failed to encode state: %v", err)
		return
	}
	for _, c := range r.conns {
		c.Send(msg)
	}
}
