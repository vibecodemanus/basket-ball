import { GameStatePayload, PlayerState, BallState } from '../network/protocol';

/**
 * Interpolation buffer: smooths server state updates.
 *
 * Only interpolates X (horizontal movement) — this is where jitter
 * is most visible. Y position, ball, and all state flags use server
 * values directly to keep landing, shooting, and scoring responsive.
 */

const LERP_SPEED = 25; // horizontal catch-up speed (per second)
const SNAP_THRESHOLD_X = 50; // px — if horizontal delta > this, snap

function lerpNum(a: number, b: number, t: number): number {
  return a + (b - a) * t;
}

export class Interpolator {
  /** The interpolated state that the renderer reads */
  private display: GameStatePayload | null = null;

  /**
   * Called when a new server state arrives.
   */
  pushServerState(state: GameStatePayload): void {
    if (!this.display) {
      // First state — clone and use directly
      this.display = JSON.parse(JSON.stringify(state));
      return;
    }

    // Save current display X positions before overwriting
    const prevX0 = this.display.players[0].x;
    const prevX1 = this.display.players[1].x;
    const prevBallX = this.display.ball.x;

    // Overwrite everything with server state
    const d = this.display;
    d.tick = state.tick;
    d.phase = state.phase;
    d.phaseTimer = state.phaseTimer;
    d.score = [state.score[0], state.score[1]];
    d.shotClock = state.shotClock;
    d.gameClock = state.gameClock;
    d.winner = state.winner;

    // Players: copy all fields directly (Y, anim, grounded, etc.)
    for (let i = 0; i < 2; i++) {
      const src = state.players[i];
      const dst = d.players[i];
      dst.y = src.y;
      dst.vx = src.vx;
      dst.vy = src.vy;
      dst.facing = src.facing;
      dst.anim = src.anim;
      dst.grounded = src.grounded;
      dst.hasBall = src.hasBall;
    }

    // Ball: copy all fields directly (Y, velocity, owner, etc.)
    const bs = state.ball;
    const bd = d.ball;
    bd.y = bs.y;
    bd.vx = bs.vx;
    bd.vy = bs.vy;
    bd.owner = bs.owner;
    bd.inFlight = bs.inFlight;

    // Store target X — lerp happens in getDisplayState()
    (d as any)._targetX0 = state.players[0].x;
    (d as any)._targetX1 = state.players[1].x;
    (d as any)._targetBallX = state.ball.x;

    // Keep current interpolated X (will lerp toward target)
    d.players[0].x = prevX0;
    d.players[1].x = prevX1;
    d.ball.x = prevBallX;
  }

  /**
   * Returns the smoothed state for rendering.
   * Only X positions are interpolated — everything else is server-authoritative.
   */
  getDisplayState(dt: number): GameStatePayload | null {
    if (!this.display) return null;

    const d = this.display as any;
    const tX0 = d._targetX0 as number | undefined;
    const tX1 = d._targetX1 as number | undefined;
    const tBallX = d._targetBallX as number | undefined;

    if (tX0 === undefined) return this.display;

    const t = Math.min(1, LERP_SPEED * dt);

    // Lerp player X positions
    const dx0 = Math.abs(this.display.players[0].x - tX0);
    this.display.players[0].x = dx0 > SNAP_THRESHOLD_X ? tX0 : lerpNum(this.display.players[0].x, tX0, t);

    const dx1 = Math.abs(this.display.players[1].x - tX1!);
    this.display.players[1].x = dx1 > SNAP_THRESHOLD_X ? tX1! : lerpNum(this.display.players[1].x, tX1!, t);

    // Lerp ball X (only when free/in-flight, snap when held by player)
    if (this.display.ball.owner >= 0) {
      this.display.ball.x = tBallX!;
    } else {
      const dBx = Math.abs(this.display.ball.x - tBallX!);
      this.display.ball.x = dBx > SNAP_THRESHOLD_X ? tBallX! : lerpNum(this.display.ball.x, tBallX!, t);
    }

    return this.display;
  }

  /** Reset on disconnect / new game */
  reset(): void {
    this.display = null;
  }
}
