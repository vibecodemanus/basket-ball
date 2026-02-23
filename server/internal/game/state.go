package game

// Physics & court constants
const (
	TickRate   = 60
	DT         = 1.0 / float32(TickRate)
	Gravity    = float32(1800.0)
	PlayerSpeedWithBall = float32(300.0)
	DefenderSpeedBoost  = float32(350.0)
	JumpVelocity         = float32(-780.0)
	DefenderJumpVelocity = float32(-880.0)
	AirControlMult       = float32(0.5)

	CourtWidth  = float32(960)
	CourtHeight = float32(450)
	FloorY      = float32(380)

	PlayerWidth  = float32(32)
	PlayerHeight = float32(48)

	BallRadius = float32(12)

	HoopLeftX  = float32(80)
	HoopRightX = float32(880)
	HoopY      = float32(180)
	RimWidth   = float32(48)
	BackboardHeight = float32(80)

	RestitutionRim       = float32(0.6)
	RestitutionBackboard = float32(0.4)
	RestitutionFloor     = float32(0.5)

	MaxShootForce = float32(1200.0)
	MinShootForce = float32(300.0)

	ShotClockSecs = float32(24)
	GameDuration  = float32(120)

	CountdownSecs   = float32(3)
	ScoredPauseSecs = float32(2)

	// Phase 8: Defense mechanics
	BlockRange        = float32(50)
	DeflectSpeedMult  = float32(0.5)

	// Phase 12: Steal mechanic
	StealRange         = float32(40)  // proximity for steal attempt
	StealChance        = 0.5          // 50% success probability
	StealCooldownTicks = uint8(45)    // ~0.75 sec anti-spam cooldown

	// 3-point line: distance from hoop center
	ThreePointRadius = float32(150)
)

type GamePhase uint8

const (
	PhaseWaiting   GamePhase = iota
	PhaseCountdown
	PhasePlaying
	PhaseScored
	PhaseGameOver
)

type AnimState uint8

const (
	AnimIdle    AnimState = iota
	AnimRun
	AnimJump
	AnimShoot
	AnimDribble
	AnimBlock
)

type PlayerState struct {
	X         float32   `json:"x"`
	Y         float32   `json:"y"`
	VX        float32   `json:"vx"`
	VY        float32   `json:"vy"`
	Facing    int8      `json:"facing"`
	Anim      AnimState `json:"anim"`
	Grounded      bool      `json:"grounded"`
	HasBall       bool      `json:"hasBall"`
	StealCooldown uint8     `json:"-"` // ticks until next steal attempt allowed (not sent to client)
	PickupDelay   uint8     `json:"-"` // ticks this player can't pick up the ball (after losing it)
}

type BallState struct {
	X              float32 `json:"x"`
	Y              float32 `json:"y"`
	VX             float32 `json:"vx"`
	VY             float32 `json:"vy"`
	Owner          int8    `json:"owner"`    // -1=free, 0=player0, 1=player1
	InFlight       bool    `json:"inFlight"`
	PickupCooldown uint8   `json:"-"` // ticks before ball can be picked up (not sent to client)
	ShooterIdx     int8    `json:"-"` // who shot this ball (-1=nobody) — shooter can't collide with own shot
	ShotAgeTicks   uint8   `json:"-"` // ticks since shot was taken
	ShotOriginX    float32 `json:"-"` // x-position where shot was taken (for 3-pt detection)
}

type GameState struct {
	Tick       uint32         `json:"tick"`
	Phase      GamePhase      `json:"phase"`
	PhaseTimer float32        `json:"phaseTimer"` // countdown/scored pause timer
	Players    [2]PlayerState `json:"players"`
	Ball       BallState      `json:"ball"`
	Score      [2]uint8       `json:"score"`
	ShotClock  float32        `json:"shotClock"`
	GameClock  float32        `json:"gameClock"`
	Winner     int8           `json:"winner"` // -1=tie, 0=p1, 1=p2 (only set in GameOver)
}

type PlayerInput struct {
	MoveX int8   `json:"moveX"`
	Jump  bool   `json:"jump"`
	Shoot bool   `json:"shoot"`
	Tick  uint32 `json:"tick"`
}
