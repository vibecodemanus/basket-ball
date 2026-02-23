import { Game } from '../game/game';
import {
  COURT_WIDTH, COURT_HEIGHT, FLOOR_Y,
  PLAYER_WIDTH, PLAYER_HEIGHT, BALL_RADIUS,
  HOOP_LEFT_X, HOOP_RIGHT_X, HOOP_Y,
  RIM_WIDTH, BACKBOARD_HEIGHT,
} from '../game/court';
import { drawRect, drawCircle, drawCircleOutline, drawLine, drawText, drawRectOutline } from './draw';
import { PlayerState, BallState, AnimState, GamePhase } from '../network/protocol';
import { SpriteSet, buildSpriteSet, getSprite } from './sprites';
import { ParticleSystem } from './particles';
import { TouchController } from '../game/touch';

const PLAYER_COLORS = ['#3B82F6', '#EF4444']; // blue, red
const BALL_COLOR = '#F97316';
const COURT_FLOOR_COLOR = '#C4853C';
const COURT_FLOOR_DARK = '#B07430';
const SKY_COLOR = '#1E293B';
const BACKBOARD_COLOR = '#E2E8F0';
const RIM_COLOR = '#DC2626';
const NET_COLOR = '#CBD5E1';

export class Renderer {
  private ctx: CanvasRenderingContext2D;
  private canvas: HTMLCanvasElement;
  private spriteSets: SpriteSet[];
  private particles: ParticleSystem;
  private prevGrounded: [boolean, boolean] = [true, true];
  private ballAngle = 0;
  private lastTime = 0;
  private prevPhase: number = -1;

  // Cached static background (court + hoops) — drawn once, reused every frame
  private bgCache: HTMLCanvasElement;

  constructor(canvas: HTMLCanvasElement) {
    this.canvas = canvas;
    this.ctx = canvas.getContext('2d')!;
    this.ctx.imageSmoothingEnabled = false;

    this.spriteSets = [buildSpriteSet(0), buildSpriteSet(1)];
    this.particles = new ParticleSystem();

    // Render static background once to off-screen canvas
    this.bgCache = document.createElement('canvas');
    this.bgCache.width = COURT_WIDTH;
    this.bgCache.height = COURT_HEIGHT;
    const bgCtx = this.bgCache.getContext('2d')!;
    bgCtx.imageSmoothingEnabled = false;
    this.renderStaticBg(bgCtx);
  }

  render(game: Game): void {
    const ctx = this.ctx;
    const now = performance.now();
    const dt = Math.min(0.05, (now - this.lastTime) / 1000);
    this.lastTime = now;

    // Blit cached static background (replaces clearRect + drawBackground + drawCourt + drawHoops)
    ctx.drawImage(this.bgCache, 0, 0);

    // Update particles
    this.particles.update(dt);

    if (game.state) {
      // Clear particles on phase transition to Countdown (new game starting)
      if (game.state.phase === GamePhase.Countdown && this.prevPhase !== GamePhase.Countdown) {
        this.particles.clear();
      }
      this.prevPhase = game.state.phase;

      // Ball trail particles
      if (game.state.ball.inFlight) {
        this.particles.emitTrail(game.state.ball.x, game.state.ball.y);
      }

      // Landing dust detection
      for (let i = 0; i < 2; i++) {
        const p = game.state.players[i];
        if (p.grounded && !this.prevGrounded[i]) {
          this.particles.emitDust(p.x, p.y + PLAYER_HEIGHT / 2);
        }
        this.prevGrounded[i] = p.grounded;
      }

      // Draw particles behind players
      this.particles.draw(ctx);

      this.drawPlayers(game.state.players, game.playerIndex, game.playerNames, game.state.tick, now);
      this.drawBall(game.state.ball, dt);

      this.drawHUD(game);

      // Phase-specific overlays
      if (game.isCountdown()) {
        this.drawCountdown(game.state.phaseTimer);
      }

      if (game.isScoredPause()) {
        this.drawScoredOverlay(game);
      }

      if (game.isGameOver()) {
        this.drawGameOver(game, now);
      }
    }

    if (!game.connected) {
      this.drawWaiting(now);
    }

    if (game.opponentDisconnected && !game.isGameOver()) {
      this.drawDisconnected(game);
    }

    // Score flash (during playing phase after a score)
    if (game.lastScoreFlash && !game.isScoredPause()) {
      const elapsed = now - game.lastScoreFlash.time;
      if (elapsed < 1500) {
        const alpha = 1 - elapsed / 1500;
        ctx.globalAlpha = alpha;
        drawText(ctx, 'SCORE!', COURT_WIDTH / 2, COURT_HEIGHT / 2 - 40, '#FFD700', 36, 'center');
        ctx.globalAlpha = 1;
      }
    }

    // Touch controls overlay (drawn last, on top of everything)
    const touch = game.getTouchController();
    if (touch.isEnabled()) {
      if (window.innerHeight > window.innerWidth) {
        this.drawRotatePrompt();
      } else {
        this.drawTouchControls(touch);
      }
    }
  }

  emitScoreConfetti(hoopX: number): void {
    this.particles.emitConfetti(hoopX, HOOP_Y, 40);
  }

  // ── Static background: rendered once into off-screen canvas ──
  private renderStaticBg(ctx: CanvasRenderingContext2D): void {
    const THREE_PT_RADIUS = 150;

    // Sky background
    drawRect(ctx, 0, 0, COURT_WIDTH, COURT_HEIGHT, SKY_COLOR);
    ctx.fillStyle = 'rgba(0,0,0,0.15)';
    ctx.fillRect(0, 0, COURT_WIDTH, 60);

    // Floor with wood plank effect
    for (let y = FLOOR_Y; y < COURT_HEIGHT; y += 8) {
      const stripe = Math.floor((y - FLOOR_Y) / 8);
      const color = (stripe % 2 === 0) ? COURT_FLOOR_COLOR : COURT_FLOOR_DARK;
      drawRect(ctx, 0, y, COURT_WIDTH, 8, color);

      ctx.strokeStyle = 'rgba(0,0,0,0.08)';
      ctx.lineWidth = 1;
      for (let x = 30; x < COURT_WIDTH; x += 60) {
        const offset = (stripe % 2) * 30;
        ctx.beginPath();
        ctx.moveTo(x + offset, y);
        ctx.lineTo(x + offset, y + 8);
        ctx.stroke();
      }
    }

    // Court edge line
    drawLine(ctx, 0, FLOOR_Y, COURT_WIDTH, FLOOR_Y, '#8B6914', 2);

    // Center line on floor
    ctx.strokeStyle = 'rgba(255, 255, 255, 0.45)';
    ctx.lineWidth = 3;
    ctx.beginPath();
    ctx.moveTo(COURT_WIDTH / 2, FLOOR_Y);
    ctx.lineTo(COURT_WIDTH / 2, COURT_HEIGHT);
    ctx.stroke();

    // 3-point lines on floor (vertical marks)
    const threeLeft = HOOP_LEFT_X + THREE_PT_RADIUS;
    const threeRight = HOOP_RIGHT_X - THREE_PT_RADIUS;
    ctx.strokeStyle = 'rgba(255, 255, 255, 0.35)';
    ctx.lineWidth = 3;
    ctx.setLineDash([4, 4]);
    ctx.beginPath();
    ctx.moveTo(threeLeft, FLOOR_Y);
    ctx.lineTo(threeLeft, COURT_HEIGHT);
    ctx.stroke();
    ctx.beginPath();
    ctx.moveTo(threeRight, FLOOR_Y);
    ctx.lineTo(threeRight, COURT_HEIGHT);
    ctx.stroke();
    ctx.setLineDash([]);

    // Hoops
    this.renderStaticHoop(ctx, HOOP_LEFT_X, true);
    this.renderStaticHoop(ctx, HOOP_RIGHT_X, false);
  }

  private renderStaticHoop(ctx: CanvasRenderingContext2D, centerX: number, isLeft: boolean): void {
    const rimLeft = centerX - RIM_WIDTH / 2;
    const rimRight = centerX + RIM_WIDTH / 2;

    // Backboard
    const bbX = isLeft ? rimLeft - 6 : rimRight + 2;
    drawRect(ctx, bbX, HOOP_Y - BACKBOARD_HEIGHT / 2, 4, BACKBOARD_HEIGHT, BACKBOARD_COLOR);
    drawRectOutline(ctx, bbX, HOOP_Y - BACKBOARD_HEIGHT / 2, 4, BACKBOARD_HEIGHT, '#94A3B8', 1);
    const sqSize = 16;
    drawRectOutline(ctx, centerX - sqSize / 2, HOOP_Y - sqSize / 2, sqSize, sqSize, '#DC262666', 1);

    // Rim
    drawLine(ctx, rimLeft, HOOP_Y, rimRight, HOOP_Y, RIM_COLOR, 3);
    drawCircle(ctx, rimLeft, HOOP_Y, 3, RIM_COLOR);
    drawCircle(ctx, rimRight, HOOP_Y, 3, RIM_COLOR);

    // Net (diamond mesh) — batched into single path per row
    const netBottom = HOOP_Y + 28;
    const cols = 6;
    const rows = 4;
    ctx.strokeStyle = NET_COLOR;
    ctx.lineWidth = 1;
    for (let r = 0; r < rows; r++) {
      const t0 = r / rows;
      const t1 = (r + 1) / rows;
      const y0 = HOOP_Y + (netBottom - HOOP_Y) * t0;
      const y1 = HOOP_Y + (netBottom - HOOP_Y) * t1;
      const shrink0 = r * 2;
      const shrink1 = (r + 1) * 2;
      const w0 = (rimRight - rimLeft) - shrink0 * 2;
      const w1 = (rimRight - rimLeft) - shrink1 * 2;

      // Batch all net lines in this row into one path
      ctx.beginPath();
      for (let c = 0; c <= cols; c++) {
        const x0 = rimLeft + shrink0 + w0 * (c / cols);
        const x1a = rimLeft + shrink1 + w1 * (Math.max(0, c - 0.5) / cols);
        const x1b = rimLeft + shrink1 + w1 * (Math.min(cols, c + 0.5) / cols);
        ctx.moveTo(x0, y0);
        ctx.lineTo(x1a, y1);
        ctx.moveTo(x0, y0);
        ctx.lineTo(x1b, y1);
      }
      ctx.stroke(); // single stroke per row instead of per-line
    }

    // Pole
    const poleX = isLeft ? rimLeft - 6 : rimRight + 4;
    drawRect(ctx, poleX, HOOP_Y + BACKBOARD_HEIGHT / 2, 2, FLOOR_Y - HOOP_Y - BACKBOARD_HEIGHT / 2, '#64748B');
  }

  // ── Dynamic elements ──

  private drawPlayers(players: [PlayerState, PlayerState], localIdx: number, names: [string, string], tick: number, now: number): void {
    const ctx = this.ctx;

    for (let i = 0; i < 2; i++) {
      const p = players[i];
      const sprite = getSprite(this.spriteSets[i], p.anim, p.facing, tick);

      const x = Math.floor(p.x - PLAYER_WIDTH / 2);
      const y = Math.floor(p.y - PLAYER_HEIGHT / 2);

      // Shadow on floor
      ctx.globalAlpha = 0.3;
      const shadowScale = Math.max(0.3, 1 - (FLOOR_Y - (p.y + PLAYER_HEIGHT / 2)) / 200);
      const shadowW = PLAYER_WIDTH * shadowScale;
      ctx.fillStyle = '#000';
      ctx.beginPath();
      ctx.ellipse(Math.floor(p.x), FLOOR_Y, shadowW / 2, 3, 0, 0, Math.PI * 2);
      ctx.fill();
      ctx.globalAlpha = 1;

      // Sprite
      ctx.drawImage(sprite, x, y);

      // Local player highlight
      if (i === localIdx) {
        ctx.strokeStyle = '#FFD700';
        ctx.lineWidth = 2;
        ctx.strokeRect(x - 1, y - 1, PLAYER_WIDTH + 2, PLAYER_HEIGHT + 2);

        const arrowY = y - 12;
        const bob = Math.sin(now / 300) * 2;
        ctx.fillStyle = '#FFD700';
        ctx.beginPath();
        ctx.moveTo(p.x, arrowY + bob + 6);
        ctx.lineTo(p.x - 5, arrowY + bob);
        ctx.lineTo(p.x + 5, arrowY + bob);
        ctx.closePath();
        ctx.fill();
      }

      // Name tag
      drawText(ctx, names[i], p.x, y - 4, PLAYER_COLORS[i], 10, 'center');
    }
  }

  private drawBall(ball: BallState, dt: number): void {
    const ctx = this.ctx;
    const bx = Math.floor(ball.x);
    const by = Math.floor(ball.y);

    // Update rotation
    if (ball.inFlight || ball.owner === -1) {
      const speed = Math.sqrt(ball.vx * ball.vx + ball.vy * ball.vy);
      this.ballAngle += (speed * dt * 0.01);
    }

    // Shadow
    if (ball.owner === -1 || ball.inFlight) {
      ctx.globalAlpha = 0.25;
      const distFromFloor = FLOOR_Y - by;
      const shadowScale = Math.max(0.3, 1 - distFromFloor / 300);
      ctx.fillStyle = '#000';
      ctx.beginPath();
      ctx.ellipse(bx, FLOOR_Y, BALL_RADIUS * shadowScale, 2, 0, 0, Math.PI * 2);
      ctx.fill();
      ctx.globalAlpha = 1;
    }

    // Ball body with rotation
    ctx.save();
    ctx.translate(bx, by);
    ctx.rotate(this.ballAngle);

    // Fill + outline in one path
    ctx.beginPath();
    ctx.arc(0, 0, BALL_RADIUS, 0, Math.PI * 2);
    ctx.fillStyle = BALL_COLOR;
    ctx.fill();
    ctx.strokeStyle = '#C2410C';
    ctx.lineWidth = 2;
    ctx.stroke();

    // Seams — batched into single path
    ctx.strokeStyle = '#C2410C';
    ctx.lineWidth = 1;
    ctx.beginPath();
    ctx.moveTo(-BALL_RADIUS + 3, 0);
    ctx.lineTo(BALL_RADIUS - 3, 0);
    ctx.moveTo(0, -BALL_RADIUS + 3);
    ctx.lineTo(0, BALL_RADIUS - 3);
    ctx.stroke();
    // Curved seams
    ctx.beginPath();
    ctx.arc(0, 0, BALL_RADIUS * 0.55, -0.4, 0.4);
    ctx.moveTo(0, 0); // disconnect arcs
    ctx.arc(0, 0, BALL_RADIUS * 0.55, Math.PI - 0.4, Math.PI + 0.4);
    ctx.stroke();

    // Highlight
    ctx.fillStyle = 'rgba(255,255,255,0.25)';
    ctx.beginPath();
    ctx.arc(-3, -3, 3, 0, Math.PI * 2);
    ctx.fill();

    ctx.restore();
  }

  private drawHUD(game: Game): void {
    if (!game.state) return;
    const ctx = this.ctx;
    const s = game.state;

    // Score panel background
    drawRect(ctx, COURT_WIDTH / 2 - 90, 4, 180, 74, 'rgba(0, 0, 0, 0.55)');

    // Player nicknames (left and right of panel)
    const names = game.playerNames;
    drawText(ctx, names[0], COURT_WIDTH / 2 - 98, 30, PLAYER_COLORS[0], 13, 'right');
    drawText(ctx, names[1], COURT_WIDTH / 2 + 98, 30, PLAYER_COLORS[1], 13, 'left');

    // Score (main row)
    const scoreText = `${s.score[0]}  —  ${s.score[1]}`;
    drawText(ctx, scoreText, COURT_WIDTH / 2, 34, '#FFF', 28, 'center');

    // Game clock
    const mins = Math.floor(Math.max(0, s.gameClock) / 60);
    const secs = Math.floor(Math.max(0, s.gameClock) % 60);
    const clockText = `${mins}:${secs.toString().padStart(2, '0')}`;
    const clockColor = s.gameClock <= 30 ? '#EF4444' : '#94A3B8';
    drawText(ctx, clockText, COURT_WIDTH / 2, 58, clockColor, 12, 'center');

    // Shot clock
    if (s.phase === GamePhase.Playing) {
      const shotSecs = Math.ceil(s.shotClock);
      const shotColor = s.shotClock <= 5 ? '#EF4444' : s.shotClock <= 10 ? '#FBBF24' : '#94A3B8';
      const shotSize = s.shotClock <= 5 ? 16 : 13;
      drawText(ctx, `${shotSecs}`, COURT_WIDTH / 2, 74, shotColor, shotSize, 'center');
    }

    // Controls hint (fades out) — skip on touch devices (controls are visible)
    if (s.phase === GamePhase.Playing && s.tick < 300 && !game.getTouchController().isEnabled()) {
      ctx.globalAlpha = Math.max(0, 1 - s.tick / 300);
      drawText(ctx, 'A/D: Move  W: Jump  Space: Shoot', COURT_WIDTH / 2, COURT_HEIGHT - 10, '#64748B', 10, 'center');
      ctx.globalAlpha = 1;
    }
  }

  private drawCountdown(timer: number): void {
    const ctx = this.ctx;

    ctx.fillStyle = 'rgba(0, 0, 0, 0.5)';
    ctx.fillRect(0, 0, COURT_WIDTH, COURT_HEIGHT);

    const count = Math.ceil(timer);
    const frac = timer - Math.floor(timer);

    if (count > 0) {
      const scale = 1 + frac * 0.5;
      const alpha = 0.5 + frac * 0.5;

      ctx.save();
      ctx.globalAlpha = alpha;
      ctx.translate(COURT_WIDTH / 2, COURT_HEIGHT / 2 - 20);
      ctx.scale(scale, scale);
      drawText(ctx, `${count}`, 0, 0, '#FFD700', 72, 'center');
      ctx.restore();
      ctx.globalAlpha = 1;

      drawText(ctx, 'GET READY!', COURT_WIDTH / 2, COURT_HEIGHT / 2 + 50, '#94A3B8', 18, 'center');
    } else {
      drawText(ctx, 'GO!', COURT_WIDTH / 2, COURT_HEIGHT / 2, '#22C55E', 64, 'center');
    }
  }

  private drawScoredOverlay(game: Game): void {
    if (!game.state || !game.lastScoreFlash) return;
    const ctx = this.ctx;
    const timer = game.state.phaseTimer;
    const scorer = game.lastScoreFlash.scorer;
    const pts = game.lastScoreFlash.points;

    const alpha = Math.min(0.6, timer / 1.5);
    ctx.fillStyle = `rgba(0, 0, 0, ${alpha})`;
    ctx.fillRect(0, 0, COURT_WIDTH, COURT_HEIGHT);

    const scorerName = game.playerNames[scorer];
    const scorerColor = PLAYER_COLORS[scorer];

    const ptsColor = pts >= 3 ? '#A855F7' : '#FFD700';
    drawText(ctx, `+${pts}`, COURT_WIDTH / 2, COURT_HEIGHT / 2 - 50, ptsColor, 48, 'center');
    drawText(ctx, `${scorerName} SCORES!`, COURT_WIDTH / 2, COURT_HEIGHT / 2, scorerColor, 28, 'center');

    const s = game.state;
    drawText(ctx, `${s.score[0]} — ${s.score[1]}`, COURT_WIDTH / 2, COURT_HEIGHT / 2 + 40, '#FFF', 22, 'center');
  }

  private drawGameOver(game: Game, now: number): void {
    const ctx = this.ctx;

    ctx.fillStyle = 'rgba(0, 0, 0, 0.8)';
    ctx.fillRect(0, 0, COURT_WIDTH, COURT_HEIGHT);

    drawText(ctx, 'GAME OVER', COURT_WIDTH / 2, 100, '#FFD700', 42, 'center');

    if (game.state) {
      const s = game.state;
      const winner = s.winner;

      drawText(ctx, `${s.score[0]}`, COURT_WIDTH / 2 - 80, 180, PLAYER_COLORS[0], 56, 'center');
      drawText(ctx, '—', COURT_WIDTH / 2, 180, '#64748B', 32, 'center');
      drawText(ctx, `${s.score[1]}`, COURT_WIDTH / 2 + 80, 180, PLAYER_COLORS[1], 56, 'center');

      drawText(ctx, game.playerNames[0], COURT_WIDTH / 2 - 80, 210, PLAYER_COLORS[0], 14, 'center');
      drawText(ctx, game.playerNames[1], COURT_WIDTH / 2 + 80, 210, PLAYER_COLORS[1], 14, 'center');

      if (winner >= 0) {
        const winnerName = game.playerNames[winner];
        const winnerColor = PLAYER_COLORS[winner];
        const isLocalWinner = winner === game.playerIndex;
        const msg = isLocalWinner ? 'YOU WIN!' : 'YOU LOSE';
        const msgColor = isLocalWinner ? '#22C55E' : '#EF4444';

        drawText(ctx, msg, COURT_WIDTH / 2, 260, msgColor, 36, 'center');
        drawText(ctx, `${winnerName} wins!`, COURT_WIDTH / 2, 290, winnerColor, 16, 'center');
      } else {
        drawText(ctx, "IT'S A TIE!", COURT_WIDTH / 2, 260, '#FBBF24', 36, 'center');
      }
    }

    const pulse = 0.5 + Math.sin(now / 500) * 0.5;
    ctx.globalAlpha = 0.6 + pulse * 0.4;
    const restartHint = game.getTouchController().isEnabled() ? 'Tap to play again' : 'Press ENTER to play again';
    drawText(ctx, restartHint, COURT_WIDTH / 2, 350, '#94A3B8', 16, 'center');
    ctx.globalAlpha = 1;
  }

  private drawDisconnected(game?: Game): void {
    const ctx = this.ctx;
    ctx.fillStyle = 'rgba(0, 0, 0, 0.6)';
    ctx.fillRect(0, 0, COURT_WIDTH, COURT_HEIGHT);
    drawText(ctx, 'OPPONENT LEFT', COURT_WIDTH / 2, COURT_HEIGHT / 2 - 10, '#EF4444', 32, 'center');
    const hint = game?.getTouchController().isEnabled() ? 'Tap to find a new match' : 'Press ENTER to find a new match';
    drawText(ctx, hint, COURT_WIDTH / 2, COURT_HEIGHT / 2 + 25, '#94A3B8', 14, 'center');
  }

  private drawWaiting(now: number): void {
    const ctx = this.ctx;
    ctx.fillStyle = 'rgba(0, 0, 0, 0.7)';
    ctx.fillRect(0, 0, COURT_WIDTH, COURT_HEIGHT);

    drawText(ctx, 'PIXEL', COURT_WIDTH / 2, COURT_HEIGHT / 2 - 70, '#FFD700', 40, 'center');
    drawText(ctx, 'BASKETBALL', COURT_WIDTH / 2, COURT_HEIGHT / 2 - 30, '#FFD700', 40, 'center');

    const ballY = COURT_HEIGHT / 2 + 10;
    const bounce = Math.sin(now / 300) * 5;
    drawCircle(ctx, COURT_WIDTH / 2, ballY + bounce, 10, BALL_COLOR);
    drawCircleOutline(ctx, COURT_WIDTH / 2, ballY + bounce, 10, '#C2410C', 1);

    const dots = '.'.repeat(Math.floor(now / 500) % 4);
    drawText(ctx, `Waiting for opponent${dots}`, COURT_WIDTH / 2, COURT_HEIGHT / 2 + 50, '#94A3B8', 16, 'center');

    drawText(ctx, 'A/D: Move  |  W: Jump  |  Space: Shoot', COURT_WIDTH / 2, COURT_HEIGHT / 2 + 85, '#475569', 11, 'center');
  }

  // ── Touch controls overlay ──

  private drawTouchControls(touch: TouchController): void {
    const ctx = this.ctx;
    const js = touch.getJoystickState();
    const sb = touch.getShootButtonState();

    // ── Joystick ──
    // Outer ring
    ctx.globalAlpha = js.active ? 0.35 : 0.15;
    ctx.beginPath();
    ctx.arc(js.centerX, js.centerY, js.radius, 0, Math.PI * 2);
    ctx.fillStyle = 'rgba(255,255,255,0.1)';
    ctx.fill();
    ctx.strokeStyle = '#FFFFFF';
    ctx.lineWidth = 2;
    ctx.stroke();

    // Direction arrows
    ctx.globalAlpha = js.active ? 0.5 : 0.25;
    drawText(ctx, '\u25C0', js.centerX - js.radius + 12, js.centerY + 5, '#FFF', 14, 'center');
    drawText(ctx, '\u25B6', js.centerX + js.radius - 12, js.centerY + 5, '#FFF', 14, 'center');
    drawText(ctx, '\u25B2', js.centerX, js.centerY - js.radius + 16, '#FFF', 12, 'center');

    // Knob
    const knobR = js.knobRadius;
    const knobDrawX = js.centerX + js.knobX * (js.radius - knobR);
    const knobDrawY = js.centerY + js.knobY * (js.radius - knobR);
    ctx.globalAlpha = js.active ? 0.6 : 0.25;
    ctx.beginPath();
    ctx.arc(knobDrawX, knobDrawY, knobR, 0, Math.PI * 2);
    ctx.fillStyle = '#FFFFFF';
    ctx.fill();

    // ── Shoot button ──
    ctx.globalAlpha = sb.pressed ? 0.5 : 0.2;
    ctx.beginPath();
    ctx.arc(sb.centerX, sb.centerY, sb.radius, 0, Math.PI * 2);
    ctx.fillStyle = sb.pressed ? '#F97316' : 'rgba(255,255,255,0.1)';
    ctx.fill();
    ctx.strokeStyle = '#F97316';
    ctx.lineWidth = 2;
    ctx.stroke();

    ctx.globalAlpha = sb.pressed ? 0.8 : 0.35;
    drawText(ctx, 'SHOOT', sb.centerX, sb.centerY + 5, '#FFF', 13, 'center');

    ctx.globalAlpha = 1;
  }

  private drawRotatePrompt(): void {
    const ctx = this.ctx;
    ctx.fillStyle = 'rgba(0, 0, 0, 0.92)';
    ctx.fillRect(0, 0, COURT_WIDTH, COURT_HEIGHT);

    drawText(ctx, '\u21BB', COURT_WIDTH / 2, COURT_HEIGHT / 2 - 40, '#FFD700', 48, 'center');
    drawText(ctx, 'ROTATE YOUR DEVICE', COURT_WIDTH / 2, COURT_HEIGHT / 2 + 10, '#FFD700', 22, 'center');
    drawText(ctx, 'Landscape mode required', COURT_WIDTH / 2, COURT_HEIGHT / 2 + 40, '#94A3B8', 14, 'center');
  }
}
