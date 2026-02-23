import { GameSocket } from './network/socket';
import { Game } from './game/game';
import { Renderer } from './render/renderer';
import { COURT_WIDTH, COURT_HEIGHT, HOOP_LEFT_X, HOOP_RIGHT_X } from './game/court';
import { isTouchDevice } from './game/touch';

// ── Nickname management ──

const NICKNAME_KEY = 'nickname';

function getSavedNickname(): string | null {
  return localStorage.getItem(NICKNAME_KEY);
}

function saveNickname(name: string): void {
  localStorage.setItem(NICKNAME_KEY, name);
}

function validateNickname(raw: string): string | null {
  const trimmed = raw.trim();
  if (trimmed.length < 2) return null;
  if (trimmed.length > 12) return null;
  // Allow letters, digits, underscore, dash, spaces, cyrillic
  if (!/^[a-zA-Z0-9_\- \u0400-\u04FF]+$/.test(trimmed)) return null;
  return trimmed;
}

// ── Overlay control ──

const overlay = document.getElementById('nickname-overlay')!;
const nicknameInput = document.getElementById('nickname-input') as HTMLInputElement;
const nicknameOk = document.getElementById('nickname-ok')!;
const nicknameError = document.getElementById('nickname-error')!;

function hideOverlay(): void {
  overlay.classList.add('hidden');
}

function showOverlay(): void {
  overlay.classList.remove('hidden');
  nicknameInput.focus();
}

function submitNickname(): void {
  const valid = validateNickname(nicknameInput.value);
  if (!valid) {
    nicknameError.textContent = '2-12 characters (letters, digits, _)';
    return;
  }
  nicknameError.textContent = '';
  saveNickname(valid);
  hideOverlay();
  startGame(valid);
}

nicknameOk.addEventListener('click', submitNickname);
nicknameInput.addEventListener('keydown', (e) => {
  if (e.key === 'Enter') submitNickname();
});

// ── Game startup ──

const canvas = document.getElementById('game') as HTMLCanvasElement;
canvas.width = COURT_WIDTH;
canvas.height = COURT_HEIGHT;

function resizeCanvas() {
  const maxW = window.innerWidth;
  const maxH = window.innerHeight;
  const scale = Math.min(maxW / COURT_WIDTH, maxH / COURT_HEIGHT);
  canvas.style.width = `${Math.floor(COURT_WIDTH * scale)}px`;
  canvas.style.height = `${Math.floor(COURT_HEIGHT * scale)}px`;
}

function startGame(nickname: string): void {
  // Show canvas
  canvas.style.display = 'block';
  resizeCanvas();
  window.addEventListener('resize', resizeCanvas);

  // Try to lock landscape orientation on mobile
  if (isTouchDevice() && screen.orientation?.lock) {
    screen.orientation.lock('landscape').catch(() => {});
  }

  // Determine WebSocket URL
  const wsProtocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
  const wsUrl = `${wsProtocol}//${location.host}/ws`;

  const socket = new GameSocket(wsUrl, nickname);
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

  window.addEventListener('keydown', (e) => {
    if (e.code === 'Enter') triggerPlayAgain();
  });

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
}

// ── Init: check saved nickname ──

const saved = getSavedNickname();
if (saved) {
  hideOverlay();
  startGame(saved);
} else {
  showOverlay();
}
