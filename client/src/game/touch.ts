// Mobile touch controls: virtual joystick + shoot button

import { PlayerInputPayload } from '../network/protocol';

// ── Detection ──
export function isTouchDevice(): boolean {
  return 'ontouchstart' in window || navigator.maxTouchPoints > 0;
}

// ── Coordinate mapping: screen → canvas 960×450 ──
function screenToCanvas(
  canvas: HTMLCanvasElement, clientX: number, clientY: number,
): { x: number; y: number } {
  const rect = canvas.getBoundingClientRect();
  return {
    x: (clientX - rect.left) * (canvas.width / rect.width),
    y: (clientY - rect.top) * (canvas.height / rect.height),
  };
}

// ── Layout constants (in canvas logical coords 960×450) ──
const JOYSTICK_X = 110;
const JOYSTICK_Y = 340;
const JOYSTICK_RADIUS = 55;
const JOYSTICK_KNOB_R = 22;
const JOYSTICK_DEAD = 0.16; // normalized dead zone (0..1)
const JUMP_THRESHOLD = -30;  // canvas-px upward delta to trigger jump

const SHOOT_X = 885;
const SHOOT_Y = 340;
const SHOOT_R = 44;

const HALF_X = 480; // left/right half boundary

const PLAY_AGAIN_ZONE = { x: 330, y: 280, w: 300, h: 100 };

// ── Joystick state ──
interface Stick {
  active: boolean;
  touchId: number;
  knobX: number; // -1..1 normalized
  knobY: number;
  startY: number; // canvas Y where touch began
  jumpFired: boolean;
}

// ── Shoot button state ──
interface ShootBtn {
  active: boolean;
  touchId: number;
  pressed: boolean;
}

// ── Public state for renderer ──
export interface JoystickVisual {
  centerX: number; centerY: number; radius: number;
  knobX: number; knobY: number; knobRadius: number; active: boolean;
}
export interface ShootBtnVisual {
  centerX: number; centerY: number; radius: number; pressed: boolean;
}

export class TouchController {
  private canvas: HTMLCanvasElement;
  private enabled: boolean;
  private stick: Stick;
  private shoot: ShootBtn;
  private playAgainHandler: (() => void) | null = null;
  private isRestartable: () => boolean;
  // After all touches released, emit one neutral frame before returning null
  private pendingNeutral = false;

  constructor(canvas: HTMLCanvasElement, isRestartable: () => boolean) {
    this.canvas = canvas;
    this.enabled = isTouchDevice();
    this.isRestartable = isRestartable;

    this.stick = { active: false, touchId: -1, knobX: 0, knobY: 0, startY: 0, jumpFired: false };
    this.shoot = { active: false, touchId: -1, pressed: false };

    if (this.enabled) {
      canvas.addEventListener('touchstart', (e) => this.onTouchStart(e), { passive: false });
      canvas.addEventListener('touchmove', (e) => this.onTouchMove(e), { passive: false });
      canvas.addEventListener('touchend', (e) => this.onTouchEnd(e));
      canvas.addEventListener('touchcancel', (e) => this.onTouchEnd(e));
    }
  }

  // ── Public API ──

  isEnabled(): boolean { return this.enabled; }

  /** Returns touch input, or null when no touch active (let keyboard take over) */
  getInput(): PlayerInputPayload | null {
    if (!this.enabled) return null;

    // After all touches released: one neutral frame, then null
    if (!this.stick.active && !this.shoot.active) {
      if (this.pendingNeutral) {
        this.pendingNeutral = false;
        return { moveX: 0, jump: false, shoot: false };
      }
      return null;
    }

    // Compute discrete moveX from stick
    let moveX = 0;
    if (this.stick.active) {
      if (this.stick.knobX > JOYSTICK_DEAD) moveX = 1;
      else if (this.stick.knobX < -JOYSTICK_DEAD) moveX = -1;
    }

    // Jump: one-shot from swipe-up
    const jump = this.stick.active && this.stick.jumpFired;
    if (jump) this.stick.jumpFired = false; // consume

    return { moveX, jump, shoot: this.shoot.pressed };
  }

  getJoystickState(): JoystickVisual {
    return {
      centerX: JOYSTICK_X, centerY: JOYSTICK_Y, radius: JOYSTICK_RADIUS,
      knobX: this.stick.knobX, knobY: this.stick.knobY, knobRadius: JOYSTICK_KNOB_R,
      active: this.stick.active,
    };
  }

  getShootButtonState(): ShootBtnVisual {
    return {
      centerX: SHOOT_X, centerY: SHOOT_Y, radius: SHOOT_R,
      pressed: this.shoot.pressed,
    };
  }

  setPlayAgainHandler(handler: () => void): void {
    this.playAgainHandler = handler;
  }

  // ── Touch handlers ──

  private onTouchStart(e: TouchEvent): void {
    e.preventDefault();
    for (let i = 0; i < e.changedTouches.length; i++) {
      const t = e.changedTouches[i];
      const pos = screenToCanvas(this.canvas, t.clientX, t.clientY);

      // Play Again tap
      if (this.isRestartable()) {
        const z = PLAY_AGAIN_ZONE;
        if (pos.x >= z.x && pos.x <= z.x + z.w && pos.y >= z.y && pos.y <= z.y + z.h) {
          if (this.playAgainHandler) this.playAgainHandler();
          return;
        }
      }

      // Joystick — left half of screen, generous hit area
      if (!this.stick.active && pos.x < HALF_X) {
        const dx = pos.x - JOYSTICK_X;
        const dy = pos.y - JOYSTICK_Y;
        const dist = Math.sqrt(dx * dx + dy * dy);
        if (dist < JOYSTICK_RADIUS * 2) { // generous touch area
          this.stick.active = true;
          this.stick.touchId = t.identifier;
          this.stick.startY = pos.y;
          this.stick.jumpFired = false;
          this.updateKnob(pos.x, pos.y);
          this.pendingNeutral = true;
          continue;
        }
      }

      // Shoot button — right half of screen
      if (!this.shoot.active && pos.x >= HALF_X) {
        const dx = pos.x - SHOOT_X;
        const dy = pos.y - SHOOT_Y;
        const dist = Math.sqrt(dx * dx + dy * dy);
        if (dist < SHOOT_R * 2) { // generous touch area
          this.shoot.active = true;
          this.shoot.touchId = t.identifier;
          this.shoot.pressed = true;
          this.pendingNeutral = true;
          continue;
        }
      }
    }
  }

  private onTouchMove(e: TouchEvent): void {
    e.preventDefault();
    for (let i = 0; i < e.changedTouches.length; i++) {
      const t = e.changedTouches[i];
      if (this.stick.active && t.identifier === this.stick.touchId) {
        const pos = screenToCanvas(this.canvas, t.clientX, t.clientY);
        this.updateKnob(pos.x, pos.y);

        // Jump detection: upward swipe from start position
        if (!this.stick.jumpFired && (pos.y - this.stick.startY) < JUMP_THRESHOLD) {
          this.stick.jumpFired = true;
        }
      }
    }
  }

  private onTouchEnd(e: TouchEvent): void {
    for (let i = 0; i < e.changedTouches.length; i++) {
      const t = e.changedTouches[i];
      if (this.stick.active && t.identifier === this.stick.touchId) {
        this.stick.active = false;
        this.stick.touchId = -1;
        this.stick.knobX = 0;
        this.stick.knobY = 0;
        this.stick.jumpFired = false;
      }
      if (this.shoot.active && t.identifier === this.shoot.touchId) {
        this.shoot.active = false;
        this.shoot.touchId = -1;
        this.shoot.pressed = false;
      }
    }
  }

  private updateKnob(px: number, py: number): void {
    let dx = px - JOYSTICK_X;
    let dy = py - JOYSTICK_Y;
    const dist = Math.sqrt(dx * dx + dy * dy);
    if (dist > JOYSTICK_RADIUS) {
      dx = (dx / dist) * JOYSTICK_RADIUS;
      dy = (dy / dist) * JOYSTICK_RADIUS;
    }
    this.stick.knobX = dx / JOYSTICK_RADIUS;  // -1..1
    this.stick.knobY = dy / JOYSTICK_RADIUS;
  }
}
