import {
  GameStatePayload,
  GameStartPayload,
  GameOverPayload,
  PlayerState,
  BallState,
  GamePhase,
  MsgGameState,
  MsgGameStart,
  MsgGameOver,
  MsgPlayerInput,
  MsgPlayerDisconnected,
  MsgScored,
  Message,
  ScoredPayload,
} from '../network/protocol';
import { GameSocket } from '../network/socket';
import { InputManager } from './input';
import { TouchController } from './touch';

export class Game {
  socket: GameSocket;
  input: InputManager;
  playerIndex: number = -1;
  state: GameStatePayload | null = null;
  connected: boolean = false;
  opponentDisconnected: boolean = false;
  lastScoreFlash: { scorer: number; points: number; time: number } | null = null;
  gameOverData: GameOverPayload | null = null;
  onScore: ((scorerIndex: number) => void) | null = null;
  private prevMoveX = 0;
  private prevJump = false;
  private prevShoot = false;

  constructor(socket: GameSocket, canvas: HTMLCanvasElement) {
    this.socket = socket;
    this.input = new InputManager(canvas, () => this.isGameOver() || this.opponentDisconnected);

    socket.onMessage((msg) => this.handleMessage(msg));
    socket.onClose(() => this.resetState());
  }

  resetState(): void {
    this.connected = false;
    this.state = null;
    this.playerIndex = -1;
    this.gameOverData = null;
    this.lastScoreFlash = null;
    // Don't reset opponentDisconnected here — it's reset on new GameStart
  }

  private handleMessage(msg: Message): void {
    switch (msg.type) {
      case MsgGameStart: {
        const payload = msg.payload as GameStartPayload;
        this.playerIndex = payload.playerIndex;
        this.connected = true;
        this.gameOverData = null;
        this.opponentDisconnected = false;
        console.log(`Game started! You are player ${this.playerIndex}`);
        break;
      }
      case MsgGameState: {
        this.state = msg.payload as GameStatePayload;
        break;
      }
      case MsgScored: {
        const scored = msg.payload as ScoredPayload;
        this.lastScoreFlash = { scorer: scored.scorerIndex, points: scored.points, time: performance.now() };
        if (this.onScore) this.onScore(scored.scorerIndex);
        break;
      }
      case MsgGameOver: {
        this.gameOverData = msg.payload as GameOverPayload;
        console.log('Game over!', this.gameOverData);
        break;
      }
      case MsgPlayerDisconnected: {
        this.opponentDisconnected = true;
        break;
      }
    }
  }

  update(): void {
    if (!this.connected || !this.state || this.playerIndex < 0) return;

    // Don't send input during non-playing phases
    if (this.state.phase !== GamePhase.Playing) return;

    const input = this.input.getInput();

    // Only send if input changed (saves ~90% of network messages)
    if (input.moveX === this.prevMoveX && input.jump === this.prevJump && input.shoot === this.prevShoot) {
      return;
    }
    this.prevMoveX = input.moveX;
    this.prevJump = input.jump;
    this.prevShoot = input.shoot;

    const msg: Message = {
      type: MsgPlayerInput,
      tick: this.state.tick,
      payload: input,
    };
    this.socket.send(msg);
  }

  getLocalPlayer(): PlayerState | null {
    if (!this.state || this.playerIndex < 0) return null;
    return this.state.players[this.playerIndex];
  }

  getRemotePlayer(): PlayerState | null {
    if (!this.state || this.playerIndex < 0) return null;
    return this.state.players[1 - this.playerIndex];
  }

  getBall(): BallState | null {
    if (!this.state) return null;
    return this.state.ball;
  }

  isGameOver(): boolean {
    return this.state?.phase === GamePhase.GameOver || this.gameOverData !== null;
  }

  isCountdown(): boolean {
    return this.state?.phase === GamePhase.Countdown;
  }

  isScoredPause(): boolean {
    return this.state?.phase === GamePhase.Scored;
  }

  getTouchController(): TouchController {
    return this.input.getTouchController();
  }
}
