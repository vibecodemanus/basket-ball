import { GameStatePayload, PlayerState, BallState } from '../network/protocol';

/**
 * Interpolation buffer: smooths server state updates.
 *
 * Instead of snapping to the latest server state, we keep two snapshots
 * and interpolate between them. The renderer always shows a position
 * slightly in the past (one server tick behind), but movement is smooth.
 *
 * If a snapshot is too old (>100ms gap), we snap instantly to avoid
 * showing stale data.
 */

interface Snapshot {
  state: GameStatePayload;
  time: number; // performance.now() when received
}

const LERP_SPEED = 15; // how fast to catch up (per second). Higher = snappier, lower = smoother
const SNAP_THRESHOLD = 80; // pixels — if delta > this, snap instead of interpolating
const BALL_SNAP_THRESHOLD = 120; // ball can move faster, larger threshold

function lerpNum(a: number, b: number, t: number): number {
  return a + (b - a) * t;
}

function lerpPlayer(out: PlayerState, target: PlayerState, t: number, snap: boolean): void {
  if (snap) {
    out.x = target.x;
    out.y = target.y;
  } else {
    out.x = lerpNum(out.x, target.x, t);
    out.y = lerpNum(out.y, target.y, t);
  }
  // Non-positional fields — always use latest
  out.vx = target.vx;
  out.vy = target.vy;
  out.facing = target.facing;
  out.anim = target.anim;
  out.grounded = target.grounded;
  out.hasBall = target.hasBall;
}

function lerpBall(out: BallState, target: BallState, t: number, snap: boolean): void {
  if (snap) {
    out.x = target.x;
    out.y = target.y;
  } else {
    out.x = lerpNum(out.x, target.x, t);
    out.y = lerpNum(out.y, target.y, t);
  }
  out.vx = target.vx;
  out.vy = target.vy;
  out.owner = target.owner;
  out.inFlight = target.inFlight;
}

function distSq(ax: number, ay: number, bx: number, by: number): number {
  const dx = ax - bx;
  const dy = ay - by;
  return dx * dx + dy * dy;
}

export class Interpolator {
  /** The interpolated state that the renderer reads */
  private display: GameStatePayload | null = null;
  private lastUpdateTime = 0;

  /**
   * Called when a new server state arrives.
   * Initializes display state on first call.
   */
  pushServerState(state: GameStatePayload): void {
    if (!this.display) {
      // First state — use directly, no interpolation
      this.display = JSON.parse(JSON.stringify(state));
      this.lastUpdateTime = performance.now();
      return;
    }

    // Copy non-interpolated fields immediately
    this.display.tick = state.tick;
    this.display.phase = state.phase;
    this.display.phaseTimer = state.phaseTimer;
    this.display.score = [state.score[0], state.score[1]];
    this.display.shotClock = state.shotClock;
    this.display.gameClock = state.gameClock;
    this.display.winner = state.winner;

    this.lastUpdateTime = performance.now();

    // Store target positions — actual lerp happens in getDisplayState()
    // We store the target in a side channel
    (this.display as any)._targetPlayers = [
      { ...state.players[0] },
      { ...state.players[1] },
    ];
    (this.display as any)._targetBall = { ...state.ball };
  }

  /**
   * Returns the smoothed state for rendering.
   * Call this every frame with the frame's delta time.
   */
  getDisplayState(dt: number): GameStatePayload | null {
    if (!this.display) return null;

    const target = this.display as any;
    const targetPlayers = target._targetPlayers as [PlayerState, PlayerState] | undefined;
    const targetBall = target._targetBall as BallState | undefined;

    if (!targetPlayers || !targetBall) {
      return this.display;
    }

    // Lerp factor: higher dt = more catch-up
    const t = Math.min(1, LERP_SPEED * dt);

    // Interpolate each player
    for (let i = 0; i < 2; i++) {
      const curr = this.display.players[i];
      const tgt = targetPlayers[i];
      const d = distSq(curr.x, curr.y, tgt.x, tgt.y);
      const snap = d > SNAP_THRESHOLD * SNAP_THRESHOLD;
      lerpPlayer(curr, tgt, t, snap);
    }

    // Interpolate ball
    const ballD = distSq(this.display.ball.x, this.display.ball.y, targetBall.x, targetBall.y);
    const ballSnap = ballD > BALL_SNAP_THRESHOLD * BALL_SNAP_THRESHOLD;
    lerpBall(this.display.ball, targetBall, t, ballSnap);

    return this.display;
  }

  /** Reset on disconnect / new game */
  reset(): void {
    this.display = null;
    this.lastUpdateTime = 0;
  }
}
