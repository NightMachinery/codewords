import type { MemoryCaptureModel } from './gameplay';

export type MemoryCaptureColorName = 'blue' | 'red' | 'civilian' | 'black' | 'hidden';

interface CapturePalette {
  fill: string;
  stroke: string;
  text: string;
}

const neutralInk = '#f4f0ea';
const mutedInk = '#c9c3b8';

export function memoryCaptureFilename(roomId: string): string {
  const safe = roomId.trim().replace(/[^a-zA-Z0-9-]+/g, '-').replace(/^-+|-+$/g, '').slice(0, 64);
  return `codewords-memory-${safe || 'board'}.png`;
}

export function cardCaptureColors(color: MemoryCaptureColorName): CapturePalette {
  return {
    blue: { fill: '#244fbc', stroke: '#8fb2ff', text: '#f5f8ff' },
    red: { fill: '#b72d36', stroke: '#ff9aa2', text: '#fff5f5' },
    civilian: { fill: '#9f7841', stroke: '#ffd79a', text: '#fff8e8' },
    black: { fill: '#15161c', stroke: '#a6a7af', text: '#f4f4f6' },
    hidden: { fill: '#293042', stroke: '#667089', text: '#f1f5f9' },
  }[color];
}

export async function downloadMemoryCapture(model: MemoryCaptureModel, doc: Document = document): Promise<void> {
  const canvas = doc.createElement('canvas');
  canvas.width = 1400;
  canvas.height = 1800;
  const ctx = canvas.getContext('2d');
  if (!ctx) throw new Error('Could not prepare the memory canvas.');
  await drawMemoryCapture(ctx, model, canvas.width, canvas.height);
  const blob = await canvasBlob(canvas);
  const url = URL.createObjectURL(blob);
  try {
    const link = doc.createElement('a');
    link.href = url;
    link.download = memoryCaptureFilename(model.roomId);
    link.rel = 'noopener';
    doc.body.appendChild(link);
    link.click();
    link.remove();
  } finally {
    URL.revokeObjectURL(url);
  }
}

async function drawMemoryCapture(ctx: CanvasRenderingContext2D, model: MemoryCaptureModel, width: number, height: number): Promise<void> {
  ctx.fillStyle = '#111827';
  ctx.fillRect(0, 0, width, height);
  const winnerGradient = ctx.createRadialGradient(width * 0.2, 0, 80, width * 0.35, 0, 900);
  winnerGradient.addColorStop(0, `${model.winner.color}66`);
  winnerGradient.addColorStop(1, '#11182700');
  ctx.fillStyle = winnerGradient;
  ctx.fillRect(0, 0, width, 900);

  drawLabel(ctx, `Room ${model.roomId}`, 96, 120, '#9be8c8', 28, '800', 0.18);
  drawText(ctx, model.title, 96, 230, neutralInk, 96, '900');
  drawText(ctx, model.subtitle, 100, 304, mutedInk, 34, '700');
  drawText(ctx, model.generatedLabel, 100, 356, '#8d95a7', 26, '700');

  drawTeamPanel(ctx, model.winner.name, 'Winners', model.winner.players, model.winner.color, 96, 430, 568, 250);
  drawTeamPanel(ctx, model.loser.name, 'Final rivals', model.loser.players, model.loser.color, 736, 430, 568, 250);

  drawLabel(ctx, 'Final board', 96, 780, '#9be8c8', 26, '900', 0.22);
  await drawBoard(ctx, model, 96, 835);

  drawText(ctx, 'Captured with Codewords', 96, height - 82, '#777f91', 25, '800');
  drawText(ctx, 'A private table memory for the people who played it.', 96, height - 46, '#777f91', 22, '600');
}

async function drawBoard(ctx: CanvasRenderingContext2D, model: MemoryCaptureModel, x: number, y: number): Promise<void> {
  const columns = 5;
  const gap = 18;
  const cardWidth = 224;
  const cardHeight = 142;
  for (const [index, card] of model.cards.entries()) {
    const col = index % columns;
    const row = Math.floor(index / columns);
    const left = x + col * (cardWidth + gap);
    const top = y + row * (cardHeight + gap);
    const palette = cardCaptureColors(card.color as MemoryCaptureColorName);
    roundedRect(ctx, left, top, cardWidth, cardHeight, 26, palette.fill, palette.stroke, 4);
    drawText(ctx, `#${card.badgeNumber}`, left + 18, top + 34, '#d8dee9', 22, '900');
    if (card.contentType === 'image' && card.imageUrl) {
      const img = await loadImage(card.imageUrl).catch(() => null);
      if (img) {
        ctx.save();
        roundedClip(ctx, left + 12, top + 42, cardWidth - 24, cardHeight - 54, 18);
        ctx.drawImage(img, left + 12, top + 42, cardWidth - 24, cardHeight - 54);
        ctx.restore();
        continue;
      }
    }
    drawWrappedText(ctx, card.label, left + 18, top + 78, cardWidth - 36, 30, palette.text, 30, '900');
  }
}

function drawTeamPanel(ctx: CanvasRenderingContext2D, name: string, label: string, players: string[], color: string, x: number, y: number, width: number, height: number): void {
  roundedRect(ctx, x, y, width, height, 32, '#182132', `${color}cc`, 4);
  roundedRect(ctx, x + 26, y + 28, 60, 60, 30, color, `${color}00`, 0);
  drawLabel(ctx, label, x + 108, y + 58, '#a9b4c8', 21, '900', 0.16);
  drawText(ctx, name, x + 108, y + 104, neutralInk, 40, '900');
  const roster = players.length ? players.join(', ') : 'No seated players';
  drawWrappedText(ctx, roster, x + 32, y + 158, width - 64, 30, mutedInk, 27, '700');
}

function roundedRect(ctx: CanvasRenderingContext2D, x: number, y: number, width: number, height: number, radius: number, fill: string, stroke: string, strokeWidth: number): void {
  ctx.beginPath();
  ctx.roundRect(x, y, width, height, radius);
  ctx.fillStyle = fill;
  ctx.fill();
  if (strokeWidth > 0) {
    ctx.lineWidth = strokeWidth;
    ctx.strokeStyle = stroke;
    ctx.stroke();
  }
}

function roundedClip(ctx: CanvasRenderingContext2D, x: number, y: number, width: number, height: number, radius: number): void {
  ctx.beginPath();
  ctx.roundRect(x, y, width, height, radius);
  ctx.clip();
}

function drawText(ctx: CanvasRenderingContext2D, text: string, x: number, y: number, color: string, size: number, weight: string): void {
  ctx.fillStyle = color;
  ctx.font = `${weight} ${size}px Inter, ui-sans-serif, system-ui, sans-serif`;
  ctx.letterSpacing = '0px';
  ctx.fillText(text, x, y);
}

function drawLabel(ctx: CanvasRenderingContext2D, text: string, x: number, y: number, color: string, size: number, weight: string, spacing: number): void {
  ctx.fillStyle = color;
  ctx.font = `${weight} ${size}px Inter, ui-sans-serif, system-ui, sans-serif`;
  ctx.letterSpacing = `${spacing}em`;
  ctx.fillText(text.toUpperCase(), x, y);
  ctx.letterSpacing = '0px';
}

function drawWrappedText(ctx: CanvasRenderingContext2D, text: string, x: number, y: number, maxWidth: number, lineHeight: number, color: string, size: number, weight: string): void {
  ctx.fillStyle = color;
  ctx.font = `${weight} ${size}px Inter, ui-sans-serif, system-ui, sans-serif`;
  const words = text.split(/\s+/).filter(Boolean);
  const lines: string[] = [];
  let line = '';
  for (const word of words) {
    const next = line ? `${line} ${word}` : word;
    if (ctx.measureText(next).width > maxWidth && line) {
      lines.push(line);
      line = word;
    } else {
      line = next;
    }
  }
  if (line) lines.push(line);
  for (const [index, value] of lines.slice(0, 3).entries()) {
    ctx.fillText(value, x, y + index * lineHeight);
  }
}

function loadImage(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.onload = () => resolve(img);
    img.onerror = () => reject(new Error('Could not load image card.'));
    img.src = src;
  });
}

function canvasBlob(canvas: HTMLCanvasElement): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob((blob) => {
      if (blob) resolve(blob);
      else reject(new Error('Could not export the memory image.'));
    }, 'image/png');
  });
}
