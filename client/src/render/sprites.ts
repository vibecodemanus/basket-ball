// Pixel art sprite system — generates player sprites as offscreen canvases
// Each sprite is a 16x24 grid of pixels, drawn at 2x scale (32x48 on screen)

const PX = 2; // pixel scale factor
const SW = 16; // sprite width in logical pixels
const SH = 24; // sprite height in logical pixels

// Color palette indices
const _ = 0;  // transparent
const S = 1;  // skin
const H = 2;  // hair
const J = 3;  // jersey (team color)
const P = 4;  // pants/shorts
const K = 5;  // shoes
const E = 6;  // eye white
const D = 7;  // eye pupil
const O = 8;  // outline/dark
const W = 9;  // wristband
const M = 10; // mouth
const N = 11; // jersey number/stripe

// ----- SPRITE DATA (16 wide x 24 tall) -----

const IDLE_R: number[][] = [
  [_,_,_,_,_,_,H,H,H,H,_,_,_,_,_,_],
  [_,_,_,_,_,H,H,H,H,H,H,_,_,_,_,_],
  [_,_,_,_,_,H,H,H,H,H,H,_,_,_,_,_],
  [_,_,_,_,_,S,S,S,S,S,S,_,_,_,_,_],
  [_,_,_,_,_,S,E,D,S,E,D,_,_,_,_,_],
  [_,_,_,_,_,S,S,S,S,S,S,_,_,_,_,_],
  [_,_,_,_,_,S,S,M,S,S,S,_,_,_,_,_],
  [_,_,_,_,_,_,S,S,S,_,_,_,_,_,_,_],
  [_,_,_,_,J,J,J,J,J,J,J,_,_,_,_,_],
  [_,_,_,_,J,J,N,J,N,J,J,_,_,_,_,_],
  [_,_,_,S,J,J,J,J,J,J,J,S,_,_,_,_],
  [_,_,_,S,J,J,J,J,J,J,J,S,_,_,_,_],
  [_,_,_,W,J,J,J,J,J,J,J,W,_,_,_,_],
  [_,_,_,S,_,J,J,J,J,J,_,S,_,_,_,_],
  [_,_,_,_,_,J,J,J,J,J,_,_,_,_,_,_],
  [_,_,_,_,_,P,P,P,P,P,_,_,_,_,_,_],
  [_,_,_,_,_,P,P,P,P,P,_,_,_,_,_,_],
  [_,_,_,_,_,P,P,_,P,P,_,_,_,_,_,_],
  [_,_,_,_,_,P,P,_,P,P,_,_,_,_,_,_],
  [_,_,_,_,_,S,S,_,S,S,_,_,_,_,_,_],
  [_,_,_,_,_,S,S,_,S,S,_,_,_,_,_,_],
  [_,_,_,_,_,K,K,_,K,K,_,_,_,_,_,_],
  [_,_,_,_,_,K,K,_,K,K,_,_,_,_,_,_],
  [_,_,_,_,K,K,K,_,K,K,K,_,_,_,_,_],
];

const RUN_R_1: number[][] = [
  [_,_,_,_,_,_,H,H,H,H,_,_,_,_,_,_],
  [_,_,_,_,_,H,H,H,H,H,H,_,_,_,_,_],
  [_,_,_,_,_,H,H,H,H,H,H,_,_,_,_,_],
  [_,_,_,_,_,S,S,S,S,S,S,_,_,_,_,_],
  [_,_,_,_,_,S,E,D,S,E,D,_,_,_,_,_],
  [_,_,_,_,_,S,S,S,S,S,S,_,_,_,_,_],
  [_,_,_,_,_,S,S,M,S,S,S,_,_,_,_,_],
  [_,_,_,_,_,_,S,S,S,_,_,_,_,_,_,_],
  [_,_,_,_,J,J,J,J,J,J,J,_,_,_,_,_],
  [_,_,_,_,J,J,N,J,N,J,J,_,_,_,_,_],
  [_,_,_,_,J,J,J,J,J,J,J,S,_,_,_,_],
  [_,_,_,S,J,J,J,J,J,J,J,_,_,_,_,_],
  [_,_,_,W,J,J,J,J,J,J,J,_,_,_,_,_],
  [_,_,_,S,_,J,J,J,J,J,_,_,_,_,_,_],
  [_,_,_,_,_,J,J,J,J,J,_,_,_,_,_,_],
  [_,_,_,_,_,P,P,P,P,P,_,_,_,_,_,_],
  [_,_,_,_,_,P,P,P,P,P,_,_,_,_,_,_],
  [_,_,_,_,_,_,P,P,P,_,_,_,_,_,_,_],
  [_,_,_,_,_,P,P,_,_,P,_,_,_,_,_,_],
  [_,_,_,_,P,S,_,_,_,S,P,_,_,_,_,_],
  [_,_,_,_,S,S,_,_,_,_,S,_,_,_,_,_],
  [_,_,_,K,K,_,_,_,_,_,K,K,_,_,_,_],
  [_,_,_,K,K,_,_,_,_,K,K,_,_,_,_,_],
  [_,_,K,K,K,_,_,_,_,K,K,K,_,_,_,_],
];

const RUN_R_2: number[][] = [
  [_,_,_,_,_,_,H,H,H,H,_,_,_,_,_,_],
  [_,_,_,_,_,H,H,H,H,H,H,_,_,_,_,_],
  [_,_,_,_,_,H,H,H,H,H,H,_,_,_,_,_],
  [_,_,_,_,_,S,S,S,S,S,S,_,_,_,_,_],
  [_,_,_,_,_,S,E,D,S,E,D,_,_,_,_,_],
  [_,_,_,_,_,S,S,S,S,S,S,_,_,_,_,_],
  [_,_,_,_,_,S,S,M,S,S,S,_,_,_,_,_],
  [_,_,_,_,_,_,S,S,S,_,_,_,_,_,_,_],
  [_,_,_,_,J,J,J,J,J,J,J,_,_,_,_,_],
  [_,_,_,_,J,J,N,J,N,J,J,_,_,_,_,_],
  [_,_,_,S,J,J,J,J,J,J,J,_,_,_,_,_],
  [_,_,_,_,J,J,J,J,J,J,J,S,_,_,_,_],
  [_,_,_,_,J,J,J,J,J,J,J,W,_,_,_,_],
  [_,_,_,_,_,J,J,J,J,J,_,S,_,_,_,_],
  [_,_,_,_,_,J,J,J,J,J,_,_,_,_,_,_],
  [_,_,_,_,_,P,P,P,P,P,_,_,_,_,_,_],
  [_,_,_,_,_,P,P,P,P,P,_,_,_,_,_,_],
  [_,_,_,_,_,_,P,P,P,_,_,_,_,_,_,_],
  [_,_,_,_,_,P,_,_,P,P,_,_,_,_,_,_],
  [_,_,_,_,P,S,_,_,_,S,P,_,_,_,_,_],
  [_,_,_,_,_,S,_,_,S,S,_,_,_,_,_,_],
  [_,_,_,_,K,K,_,_,K,K,_,_,_,_,_,_],
  [_,_,_,_,_,K,K,K,K,_,_,_,_,_,_,_],
  [_,_,_,_,K,K,K,K,K,K,_,_,_,_,_,_],
];

const JUMP_R: number[][] = [
  [_,_,_,_,_,_,H,H,H,H,_,_,_,_,_,_],
  [_,_,_,_,_,H,H,H,H,H,H,_,_,_,_,_],
  [_,_,_,_,_,H,H,H,H,H,H,_,_,_,_,_],
  [_,_,_,_,_,S,S,S,S,S,S,_,_,_,_,_],
  [_,_,_,_,_,S,E,D,S,E,D,_,_,_,_,_],
  [_,_,_,_,_,S,S,S,S,S,S,_,_,_,_,_],
  [_,_,_,_,_,S,S,S,S,S,S,_,_,_,_,_],
  [_,_,S,_,_,_,S,S,S,_,_,_,S,_,_,_],
  [_,_,S,_,J,J,J,J,J,J,J,_,S,_,_,_],
  [_,_,W,_,J,J,N,J,N,J,J,_,W,_,_,_],
  [_,_,S,_,J,J,J,J,J,J,J,_,S,_,_,_],
  [_,_,_,_,J,J,J,J,J,J,J,_,_,_,_,_],
  [_,_,_,_,J,J,J,J,J,J,J,_,_,_,_,_],
  [_,_,_,_,_,J,J,J,J,J,_,_,_,_,_,_],
  [_,_,_,_,_,J,J,J,J,J,_,_,_,_,_,_],
  [_,_,_,_,_,P,P,P,P,P,_,_,_,_,_,_],
  [_,_,_,_,_,P,P,P,P,P,_,_,_,_,_,_],
  [_,_,_,_,_,P,P,_,P,P,_,_,_,_,_,_],
  [_,_,_,_,_,S,P,_,P,S,_,_,_,_,_,_],
  [_,_,_,_,_,S,S,_,S,S,_,_,_,_,_,_],
  [_,_,_,_,_,K,S,_,S,K,_,_,_,_,_,_],
  [_,_,_,_,_,K,K,_,_,K,_,_,_,_,_,_],
  [_,_,_,_,_,K,_,_,_,K,K,_,_,_,_,_],
  [_,_,_,_,_,_,_,_,_,_,_,_,_,_,_,_],
];

const SHOOT_R: number[][] = [
  [_,_,_,_,_,_,H,H,H,H,_,_,_,_,_,_],
  [_,_,_,_,_,H,H,H,H,H,H,_,_,_,_,_],
  [_,_,_,_,_,H,H,H,H,H,H,_,_,_,_,_],
  [_,_,_,_,_,S,S,S,S,S,S,_,_,_,_,_],
  [_,_,_,_,_,S,E,D,S,E,D,_,_,_,_,_],
  [_,_,_,_,_,S,S,S,S,S,S,_,_,_,_,_],
  [_,_,_,_,_,S,S,S,S,S,S,_,_,_,_,_],
  [_,_,_,_,_,_,S,S,S,_,_,S,S,_,_,_],
  [_,_,_,_,J,J,J,J,J,J,J,_,S,_,_,_],
  [_,_,_,_,J,J,N,J,N,J,J,_,W,_,_,_],
  [_,_,_,_,J,J,J,J,J,J,J,_,S,_,_,_],
  [_,_,_,S,J,J,J,J,J,J,J,_,_,_,_,_],
  [_,_,_,W,J,J,J,J,J,J,J,_,_,_,_,_],
  [_,_,_,S,_,J,J,J,J,J,_,_,_,_,_,_],
  [_,_,_,_,_,J,J,J,J,J,_,_,_,_,_,_],
  [_,_,_,_,_,P,P,P,P,P,_,_,_,_,_,_],
  [_,_,_,_,_,P,P,P,P,P,_,_,_,_,_,_],
  [_,_,_,_,_,P,P,_,P,P,_,_,_,_,_,_],
  [_,_,_,_,_,P,P,_,P,P,_,_,_,_,_,_],
  [_,_,_,_,_,S,S,_,S,S,_,_,_,_,_,_],
  [_,_,_,_,_,S,S,_,S,S,_,_,_,_,_,_],
  [_,_,_,_,_,K,K,_,K,K,_,_,_,_,_,_],
  [_,_,_,_,_,K,K,_,K,K,_,_,_,_,_,_],
  [_,_,_,_,K,K,K,_,K,K,K,_,_,_,_,_],
];

const DRIBBLE_R_1: number[][] = [
  [_,_,_,_,_,_,H,H,H,H,_,_,_,_,_,_],
  [_,_,_,_,_,H,H,H,H,H,H,_,_,_,_,_],
  [_,_,_,_,_,H,H,H,H,H,H,_,_,_,_,_],
  [_,_,_,_,_,S,S,S,S,S,S,_,_,_,_,_],
  [_,_,_,_,_,S,E,D,S,E,D,_,_,_,_,_],
  [_,_,_,_,_,S,S,S,S,S,S,_,_,_,_,_],
  [_,_,_,_,_,S,S,M,S,S,S,_,_,_,_,_],
  [_,_,_,_,_,_,S,S,S,_,_,_,_,_,_,_],
  [_,_,_,_,J,J,J,J,J,J,J,_,_,_,_,_],
  [_,_,_,_,J,J,N,J,N,J,J,_,_,_,_,_],
  [_,_,_,_,J,J,J,J,J,J,J,S,_,_,_,_],
  [_,_,_,S,J,J,J,J,J,J,J,S,_,_,_,_],
  [_,_,_,W,J,J,J,J,J,J,J,W,_,_,_,_],
  [_,_,_,S,_,J,J,J,J,J,_,S,_,_,_,_],
  [_,_,_,_,_,J,J,J,J,J,_,_,_,_,_,_],
  [_,_,_,_,_,P,P,P,P,P,_,_,_,_,_,_],
  [_,_,_,_,_,P,P,P,P,P,_,_,_,_,_,_],
  [_,_,_,_,_,P,P,_,P,P,_,_,_,_,_,_],
  [_,_,_,_,P,P,_,_,_,P,P,_,_,_,_,_],
  [_,_,_,_,S,S,_,_,_,S,S,_,_,_,_,_],
  [_,_,_,_,S,_,_,_,_,_,S,_,_,_,_,_],
  [_,_,_,K,K,_,_,_,_,_,K,K,_,_,_,_],
  [_,_,_,K,K,_,_,_,_,_,K,K,_,_,_,_],
  [_,_,K,K,K,_,_,_,_,K,K,K,_,_,_,_],
];

const DRIBBLE_R_2 = IDLE_R; // alternate with idle

const BLOCK_R: number[][] = [
  [_,_,_,_,_,_,H,H,H,H,_,_,_,_,_,_],
  [_,_,_,_,_,H,H,H,H,H,H,_,_,_,_,_],
  [_,_,_,_,_,H,H,H,H,H,H,_,_,_,_,_],
  [_,_,_,_,_,S,S,S,S,S,S,_,_,_,_,_],
  [_,_,_,_,_,S,E,D,S,E,D,_,_,_,_,_],
  [_,_,_,_,_,S,S,S,S,S,S,_,_,_,_,_],
  [_,_,_,_,_,S,S,S,S,S,S,_,_,_,_,_],
  [_,_,S,S,_,_,S,S,S,_,_,S,S,_,_,_],
  [_,_,S,S,J,J,J,J,J,J,J,S,S,_,_,_],
  [_,_,W,_,J,J,N,J,N,J,J,_,W,_,_,_],
  [_,_,S,_,J,J,J,J,J,J,J,_,S,_,_,_],
  [_,_,_,_,J,J,J,J,J,J,J,_,_,_,_,_],
  [_,_,_,_,J,J,J,J,J,J,J,_,_,_,_,_],
  [_,_,_,_,_,J,J,J,J,J,_,_,_,_,_,_],
  [_,_,_,_,_,J,J,J,J,J,_,_,_,_,_,_],
  [_,_,_,_,_,P,P,P,P,P,_,_,_,_,_,_],
  [_,_,_,_,_,P,P,P,P,P,_,_,_,_,_,_],
  [_,_,_,_,_,P,P,_,P,P,_,_,_,_,_,_],
  [_,_,_,_,_,S,P,_,P,S,_,_,_,_,_,_],
  [_,_,_,_,_,S,S,_,S,S,_,_,_,_,_,_],
  [_,_,_,_,_,K,S,_,S,K,_,_,_,_,_,_],
  [_,_,_,_,_,K,K,_,_,K,_,_,_,_,_,_],
  [_,_,_,_,_,K,_,_,_,K,K,_,_,_,_,_],
  [_,_,_,_,_,_,_,_,_,_,_,_,_,_,_,_],
];

// Team palettes
interface TeamPalette {
  jersey: string;
  jerseyStripe: string;
  pants: string;
  shoes: string;
  skin: string;
  hair: string;
  wristband: string;
}

const PALETTES: TeamPalette[] = [
  { // P1 - Blue team
    jersey: '#3B82F6',
    jerseyStripe: '#2563EB',
    pants: '#1E3A5F',
    shoes: '#1E40AF',
    skin: '#F0C8A0',
    hair: '#4A3728',
    wristband: '#FBBF24',
  },
  { // P2 - Red team
    jersey: '#EF4444',
    jerseyStripe: '#DC2626',
    pants: '#5F1E1E',
    shoes: '#991B1B',
    skin: '#D4A574',
    hair: '#1A1A2E',
    wristband: '#FBBF24',
  },
];

function colorForIndex(idx: number, pal: TeamPalette): string | null {
  switch (idx) {
    case S: return pal.skin;
    case H: return pal.hair;
    case J: return pal.jersey;
    case P: return pal.pants;
    case K: return pal.shoes;
    case E: return '#FFFFFF';
    case D: return '#1A1A2E';
    case O: return '#0F172A';
    case W: return pal.wristband;
    case M: return '#C4856C';
    case N: return pal.jerseyStripe;
    default: return null;
  }
}

function mirrorSprite(sprite: number[][]): number[][] {
  return sprite.map(row => [...row].reverse());
}

function renderSpriteToCanvas(sprite: number[][], pal: TeamPalette): HTMLCanvasElement {
  const c = document.createElement('canvas');
  c.width = SW * PX;
  c.height = SH * PX;
  const ctx = c.getContext('2d')!;

  for (let y = 0; y < SH; y++) {
    for (let x = 0; x < SW; x++) {
      const idx = sprite[y][x];
      const color = colorForIndex(idx, pal);
      if (color) {
        ctx.fillStyle = color;
        ctx.fillRect(x * PX, y * PX, PX, PX);
      }
    }
  }
  return c;
}

export interface SpriteSet {
  idle: HTMLCanvasElement[];     // [right, left]
  run: HTMLCanvasElement[][];    // [right[frame0,frame1], left[frame0,frame1]]
  jump: HTMLCanvasElement[];     // [right, left]
  shoot: HTMLCanvasElement[];    // [right, left]
  dribble: HTMLCanvasElement[][]; // [right[frame0,frame1], left[frame0,frame1]]
  block: HTMLCanvasElement[];    // [right, left]
}

export function buildSpriteSet(teamIdx: number): SpriteSet {
  const pal = PALETTES[teamIdx];

  const idleR = renderSpriteToCanvas(IDLE_R, pal);
  const idleL = renderSpriteToCanvas(mirrorSprite(IDLE_R), pal);

  const run1R = renderSpriteToCanvas(RUN_R_1, pal);
  const run1L = renderSpriteToCanvas(mirrorSprite(RUN_R_1), pal);
  const run2R = renderSpriteToCanvas(RUN_R_2, pal);
  const run2L = renderSpriteToCanvas(mirrorSprite(RUN_R_2), pal);

  const jumpR = renderSpriteToCanvas(JUMP_R, pal);
  const jumpL = renderSpriteToCanvas(mirrorSprite(JUMP_R), pal);

  const shootR = renderSpriteToCanvas(SHOOT_R, pal);
  const shootL = renderSpriteToCanvas(mirrorSprite(SHOOT_R), pal);

  const drib1R = renderSpriteToCanvas(DRIBBLE_R_1, pal);
  const drib1L = renderSpriteToCanvas(mirrorSprite(DRIBBLE_R_1), pal);
  const drib2R = renderSpriteToCanvas(DRIBBLE_R_2, pal);
  const drib2L = renderSpriteToCanvas(mirrorSprite(DRIBBLE_R_2), pal);

  const blockR = renderSpriteToCanvas(BLOCK_R, pal);
  const blockL = renderSpriteToCanvas(mirrorSprite(BLOCK_R), pal);

  return {
    idle: [idleR, idleL],
    run: [[run1R, run2R], [run1L, run2L]],
    jump: [jumpR, jumpL],
    shoot: [shootR, shootL],
    dribble: [[drib1R, drib2R], [drib1L, drib2L]],
    block: [blockR, blockL],
  };
}

export function getSprite(
  set: SpriteSet,
  anim: number,
  facing: -1 | 1,
  tick: number,
): HTMLCanvasElement {
  const dir = facing === 1 ? 0 : 1; // 0=right, 1=left
  const frame = Math.floor(tick / 8) % 2; // alternate every 8 ticks

  switch (anim) {
    case 1: // Run
      return set.run[dir][frame];
    case 2: // Jump
      return set.jump[dir];
    case 3: // Shoot
      return set.shoot[dir];
    case 4: // Dribble
      return set.dribble[dir][frame];
    case 5: // Block
      return set.block[dir];
    default: // Idle
      return set.idle[dir];
  }
}
