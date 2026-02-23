// Primitive drawing helpers

export function drawRect(
  ctx: CanvasRenderingContext2D,
  x: number, y: number, w: number, h: number,
  color: string,
): void {
  ctx.fillStyle = color;
  ctx.fillRect(Math.floor(x), Math.floor(y), w, h);
}

export function drawRectOutline(
  ctx: CanvasRenderingContext2D,
  x: number, y: number, w: number, h: number,
  color: string, lineWidth = 1,
): void {
  ctx.strokeStyle = color;
  ctx.lineWidth = lineWidth;
  ctx.strokeRect(Math.floor(x), Math.floor(y), w, h);
}

export function drawCircle(
  ctx: CanvasRenderingContext2D,
  x: number, y: number, radius: number,
  color: string,
): void {
  ctx.fillStyle = color;
  ctx.beginPath();
  ctx.arc(Math.floor(x), Math.floor(y), radius, 0, Math.PI * 2);
  ctx.fill();
}

export function drawCircleOutline(
  ctx: CanvasRenderingContext2D,
  x: number, y: number, radius: number,
  color: string, lineWidth = 2,
): void {
  ctx.strokeStyle = color;
  ctx.lineWidth = lineWidth;
  ctx.beginPath();
  ctx.arc(Math.floor(x), Math.floor(y), radius, 0, Math.PI * 2);
  ctx.stroke();
}

export function drawLine(
  ctx: CanvasRenderingContext2D,
  x1: number, y1: number, x2: number, y2: number,
  color: string, lineWidth = 2,
): void {
  ctx.strokeStyle = color;
  ctx.lineWidth = lineWidth;
  ctx.beginPath();
  ctx.moveTo(Math.floor(x1), Math.floor(y1));
  ctx.lineTo(Math.floor(x2), Math.floor(y2));
  ctx.stroke();
}

export function drawText(
  ctx: CanvasRenderingContext2D,
  text: string, x: number, y: number,
  color: string, size = 16, align: CanvasTextAlign = 'left',
): void {
  ctx.font = `bold ${size}px monospace`;
  ctx.textAlign = align;
  const px = Math.floor(x);
  const py = Math.floor(y);
  // Dark outline for clean pixel-art look (eliminates anti-alias artifacts)
  ctx.strokeStyle = '#000';
  ctx.lineWidth = 3;
  ctx.lineJoin = 'round';
  ctx.strokeText(text, px, py);
  // Main text fill
  ctx.fillStyle = color;
  ctx.fillText(text, px, py);
}
