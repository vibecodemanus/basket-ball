// Lightweight particle system for visual effects — optimized with pool & swap-pop

export interface Particle {
  x: number;
  y: number;
  vx: number;
  vy: number;
  life: number;     // remaining life 0..1
  maxLife: number;   // total life in seconds
  color: string;
  size: number;
  gravity: boolean;
  active: boolean;
}

const CONFETTI_COLORS = ['#FFD700', '#EF4444', '#3B82F6', '#22C55E', '#F97316', '#A855F7'];
const DUST_COLOR = '#C4A060';
const MAX_PARTICLES = 200;
const TRAIL_INTERVAL = 3; // emit trail every N frames

export class ParticleSystem {
  private pool: Particle[] = [];
  private count = 0; // active particle count (all active particles are at indices 0..count-1)
  private trailFrame = 0;

  constructor() {
    // Pre-allocate pool
    for (let i = 0; i < MAX_PARTICLES; i++) {
      this.pool.push({
        x: 0, y: 0, vx: 0, vy: 0,
        life: 0, maxLife: 1, color: '', size: 0,
        gravity: false, active: false,
      });
    }
  }

  private acquire(): Particle | null {
    if (this.count >= MAX_PARTICLES) return null;
    const p = this.pool[this.count];
    p.active = true;
    this.count++;
    return p;
  }

  update(dt: number): void {
    let i = 0;
    while (i < this.count) {
      const p = this.pool[i];
      p.life -= dt / p.maxLife;
      if (p.life <= 0) {
        // Swap-pop: move last active into this slot
        p.active = false;
        this.count--;
        if (i < this.count) {
          // Swap fields instead of array splice
          const last = this.pool[this.count];
          this.pool[i] = last;
          this.pool[this.count] = p;
        }
        continue; // re-check same index (now holds swapped particle)
      }
      p.x += p.vx * dt;
      p.y += p.vy * dt;
      if (p.gravity) {
        p.vy += 600 * dt;
      }
      i++;
    }
  }

  draw(ctx: CanvasRenderingContext2D): void {
    for (let i = 0; i < this.count; i++) {
      const p = this.pool[i];
      ctx.globalAlpha = Math.min(1, p.life * 2);
      ctx.fillStyle = p.color;
      const s = p.size * (0.5 + p.life * 0.5);
      ctx.fillRect(Math.floor(p.x - s / 2), Math.floor(p.y - s / 2), Math.ceil(s), Math.ceil(s));
    }
    if (this.count > 0) ctx.globalAlpha = 1;
  }

  // Score celebration — confetti burst from the hoop
  emitConfetti(x: number, y: number, count = 30): void {
    for (let i = 0; i < count; i++) {
      const p = this.acquire();
      if (!p) break;
      const angle = Math.random() * Math.PI * 2;
      const speed = 80 + Math.random() * 200;
      p.x = x; p.y = y;
      p.vx = Math.cos(angle) * speed;
      p.vy = Math.sin(angle) * speed - 100;
      p.life = 1;
      p.maxLife = 0.8 + Math.random() * 0.8;
      p.color = CONFETTI_COLORS[Math.floor(Math.random() * CONFETTI_COLORS.length)];
      p.size = 3 + Math.random() * 4;
      p.gravity = true;
    }
  }

  // Dust puff when landing
  emitDust(x: number, y: number, count = 6): void {
    for (let i = 0; i < count; i++) {
      const p = this.acquire();
      if (!p) break;
      const angle = -Math.PI / 2 + (Math.random() - 0.5) * Math.PI * 0.8;
      const speed = 20 + Math.random() * 60;
      p.x = x + (Math.random() - 0.5) * 16;
      p.y = y;
      p.vx = Math.cos(angle) * speed;
      p.vy = Math.sin(angle) * speed;
      p.life = 1;
      p.maxLife = 0.3 + Math.random() * 0.2;
      p.color = DUST_COLOR;
      p.size = 2 + Math.random() * 3;
      p.gravity = false;
    }
  }

  // Ball trail when in flight — throttled to every N frames
  emitTrail(x: number, y: number): void {
    this.trailFrame++;
    if (this.trailFrame % TRAIL_INTERVAL !== 0) return;
    const p = this.acquire();
    if (!p) return;
    p.x = x + (Math.random() - 0.5) * 4;
    p.y = y + (Math.random() - 0.5) * 4;
    p.vx = 0; p.vy = 0;
    p.life = 1;
    p.maxLife = 0.2;
    p.color = '#F9731666';
    p.size = 4 + Math.random() * 4;
    p.gravity = false;
  }

  clear(): void {
    for (let i = 0; i < this.count; i++) {
      this.pool[i].active = false;
    }
    this.count = 0;
  }
}
