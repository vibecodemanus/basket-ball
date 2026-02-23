import { GameSocket } from './network/socket';
import { Game } from './game/game';
import { Renderer } from './render/renderer';
import { COURT_WIDTH, COURT_HEIGHT, HOOP_LEFT_X, HOOP_RIGHT_X } from './game/court';
import { isTouchDevice } from './game/touch';

// Determine WebSocket URL
const wsProtocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
const wsUrl = `${wsProtocol}//${location.host}/ws`;

// Create canvas
const canvas = document.getElementById('game') as HTMLCanvasElement;
canvas.width = COURT_WIDTH;
canvas.height = COURT_HEIGHT;

// Scale canvas to fit window while maintaining aspect ratio
function resizeCanvas() {
  const maxW = window.innerWidth;
  const maxH = window.innerHeight;
  const scale = Math.min(maxW / COURT_WIDTH, maxH / COURT_HEIGHT);
  canvas.style.width = `${Math.floor(COURT_WIDTH * scale)}px`;
  canvas.style.height = `${Math.floor(COURT_HEIGHT * scale)}px`;
}
resizeCanvas();
window.addEventListener('resize', resizeCanvas);

// Try to lock landscape orientation on mobile
if (isTouchDevice() && screen.orientation?.lock) {
  screen.orientation.lock('landscape').catch(() => {});
}

// Connect
const socket = new GameSocket(wsUrl);
const game = new Game(socket, canvas);
const renderer = new Renderer(canvas);

// Expose game for debugging
(window as any).__game = game;

// Wire up score confetti
game.onScore = (scorerIdx: number) => {
  const hoopX = scorerIdx === 0 ? HOOP_RIGHT_X : HOOP_LEFT_X;
  renderer.emitScoreConfetti(hoopX);
};

setTimeout(() => socket.connect(), 500);

// ── Play Again (keyboard + touch) ──
function triggerPlayAgain(): void {
  if (game.isGameOver() || game.opponentDisconnected) {
    socket.disconnect();
    game.opponentDisconnected = false;
    setTimeout(() => socket.connect(), 300);
  }
}

// Keyboard: Enter
window.addEventListener('keydown', (e) => {
  if (e.code === 'Enter') triggerPlayAgain();
});

// Touch: tap zone handled by TouchController
game.getTouchController().setPlayAgainHandler(triggerPlayAgain);

// ── Game loop ──
const INPUT_RATE = 1000 / 60;
let lastInputTime = 0;

function loop(timestamp: number): void {
  if (timestamp - lastInputTime >= INPUT_RATE) {
    game.update();
    lastInputTime = timestamp;
  }

  renderer.render(game);
  requestAnimationFrame(loop);
}

requestAnimationFrame(loop);
