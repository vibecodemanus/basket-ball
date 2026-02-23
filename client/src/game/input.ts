import { PlayerInputPayload } from '../network/protocol';
import { TouchController } from './touch';

export class InputManager {
  private keys: Set<string> = new Set();
  private touch: TouchController;

  constructor(canvas: HTMLCanvasElement, isRestartable: () => boolean) {
    window.addEventListener('keydown', (e) => { this.keys.add(e.code); });
    window.addEventListener('keyup', (e) => { this.keys.delete(e.code); });

    this.touch = new TouchController(canvas, isRestartable);
  }

  getInput(): PlayerInputPayload {
    // Touch input takes priority when active
    const touchInput = this.touch.getInput();
    if (touchInput) return touchInput;

    // Keyboard fallback
    let moveX = 0;
    if (this.keys.has('ArrowLeft') || this.keys.has('KeyA')) moveX -= 1;
    if (this.keys.has('ArrowRight') || this.keys.has('KeyD')) moveX += 1;

    const jump = this.keys.has('ArrowUp') || this.keys.has('KeyW');
    const shoot = this.keys.has('Space');

    return { moveX, jump, shoot };
  }

  getTouchController(): TouchController { return this.touch; }
}
