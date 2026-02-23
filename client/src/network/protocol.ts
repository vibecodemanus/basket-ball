// Message type IDs — mirrors server/internal/ws/message.go
export const MsgPlayerInput = 0x01;
export const MsgJoinQueue = 0x02;
export const MsgPing = 0x04;

export const MsgGameState = 0x81;
export const MsgGameStart = 0x82;
export const MsgGameOver = 0x83;
export const MsgScored = 0x84;
export const MsgPong = 0x86;
export const MsgPlayerDisconnected = 0x87;

export interface Message {
  type: number;
  tick: number;
  payload: unknown;
}

export interface PlayerInputPayload {
  moveX: number;
  jump: boolean;
  shoot: boolean;
}

export interface GameStartPayload {
  playerIndex: number;
  names: [string, string];
}

export const enum GamePhase {
  Waiting = 0,
  Countdown = 1,
  Playing = 2,
  Scored = 3,
  GameOver = 4,
}

export const enum AnimState {
  Idle = 0,
  Run = 1,
  Jump = 2,
  Shoot = 3,
  Dribble = 4,
  Block = 5,
}

export interface PlayerState {
  x: number;
  y: number;
  vx: number;
  vy: number;
  facing: -1 | 1;
  anim: AnimState;
  grounded: boolean;
  hasBall: boolean;
}

export interface BallState {
  x: number;
  y: number;
  vx: number;
  vy: number;
  owner: -1 | 0 | 1;
  inFlight: boolean;
}

export interface GameStatePayload {
  tick: number;
  phase: GamePhase;
  phaseTimer: number;
  players: [PlayerState, PlayerState];
  ball: BallState;
  score: [number, number];
  shotClock: number;
  gameClock: number;
  winner: number; // -1=tie, 0=p1, 1=p2
}

export interface GameOverPayload {
  winner: number;
  score: [number, number];
}

export interface ScoredPayload {
  scorerIndex: number;
  points: number;
  newScore: [number, number];
}

export interface PlayerDisconnectedPayload {
  playerIndex: number;
}
