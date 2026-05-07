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
    hidden: { fill: '#111827', stroke: '#334155', text: '#f1f5f9' },
  }[color];
}

export async function downloadMemoryCapture(model: MemoryCaptureModel, doc: Document = document): Promise<void> {
  const canvas = doc.createElement('canvas');
  canvas.width = 1400;
  canvas.height = memoryCaptureHeight(model);
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

function memoryCaptureHeight(model: MemoryCaptureModel): number {
  const rows = Math.ceil(model.cards.length / 5);
  const boardHeight = rows * 156 + Math.max(0, rows - 1) * 18;
  return Math.max(1280, 520 + boardHeight + (model.roastLine ? 90 : 0));
}

async function drawMemoryCapture(ctx: CanvasRenderingContext2D, model: MemoryCaptureModel, width: number, height: number): Promise<void> {
  drawAurora(ctx, model.roomId, width, height);
  drawText(ctx, model.title, 96, 140, neutralInk, 84, '900');
  drawText(ctx, model.generatedLabel, 100, 190, '#aeb8c9', 24, '700');
  if (model.roastLine) {
    drawWrappedText(ctx, model.roastLine, 100, 260, width - 200, 42, '#f7d483', 38, '900', 2);
  }

  const teamTop = model.roastLine ? 330 : 260;
  drawTeamPanel(ctx, model.winner.name, 'Winners', model.winner.players, model.winner.color, 96, teamTop, 568, 190);
  drawTeamPanel(ctx, model.loser.name, 'Rivals', model.loser.players, model.loser.color, 736, teamTop, 568, 190);
  await drawBoard(ctx, model, 96, teamTop + 265);
}

function drawAurora(ctx: CanvasRenderingContext2D, seed: string, width: number, height: number): void {
  const hash = hashString(seed || 'board');
  const base = ctx.createLinearGradient(0, 0, width, height);
  base.addColorStop(0, '#07111f');
  base.addColorStop(0.5, '#0f172a');
  base.addColorStop(1, '#07101a');
  ctx.fillStyle = base;
  ctx.fillRect(0, 0, width, height);

  for (let i = 0; i < 7; i += 1) {
    const hue = (hash + i * 47) % 360;
    const x = ((hash >>> (i % 16)) % width) / width;
    const y = 0.08 + i * 0.08;
    const gradient = ctx.createRadialGradient(width * x, height * y, 40, width * x, height * y, 680 + i * 55);
    gradient.addColorStop(0, `hsla(${hue}, 86%, 66%, ${0.22 - i * 0.015})`);
    gradient.addColorStop(0.45, `hsla(${(hue + 80) % 360}, 80%, 58%, 0.08)`);
    gradient.addColorStop(1, 'rgba(7, 17, 31, 0)');
    ctx.fillStyle = gradient;
    ctx.fillRect(0, 0, width, height);
  }

  ctx.strokeStyle = 'rgba(148, 163, 184, 0.08)';
  ctx.lineWidth = 2;
  for (let line = 0; line < 16; line += 1) {
    ctx.beginPath();
    const y = 120 + line * 58;
    ctx.moveTo(0, y);
    for (let x = 0; x <= width; x += 80) {
      ctx.lineTo(x, y + Math.sin((x + hash + line * 31) / 120) * 28);
    }
    ctx.stroke();
  }
}

async function drawBoard(ctx: CanvasRenderingContext2D, model: MemoryCaptureModel, x: number, y: number): Promise<void> {
  const columns = 5;
  const gap = 18;
  const cardWidth = 224;
  const cardHeight = 156;
  for (const [index, card] of model.cards.entries()) {
    const col = index % columns;
    const row = Math.floor(index / columns);
    const left = x + col * (cardWidth + gap);
    const top = y + row * (cardHeight + gap);
    const palette = cardCaptureColors(card.color as MemoryCaptureColorName);
    roundedRect(ctx, left, top, cardWidth, cardHeight, 24, palette.fill, palette.stroke, card.color === 'hidden' ? 2 : 4);
    let drewImage = false;
    if (card.contentType === 'image' && card.imageUrl) {
      const img = await loadImage(card.imageUrl).catch(() => null);
      if (img) {
        ctx.save();
        roundedClip(ctx, left + 10, top + 10, cardWidth - 20, cardHeight - 20, 18);
        ctx.drawImage(img, left + 10, top + 10, cardWidth - 20, cardHeight - 20);
        ctx.restore();
        drewImage = true;
      }
    }
    if (!drewImage) {
      drawWrappedText(ctx, card.label, left + 18, top + 88, cardWidth - 36, 32, palette.text, 31, '900', 3);
    }
    if (card.isLastSelected) {
      roundedRect(ctx, left, top, cardWidth, cardHeight, 24, 'rgba(0,0,0,0)', '#a7f3d0', 6);
      if (card.color !== 'hidden') roundedRect(ctx, left + 6, top + 6, cardWidth - 12, cardHeight - 12, 18, 'rgba(0,0,0,0)', palette.stroke, 12);
    }
    if (model.showNumberBadges) {
      drawBadge(ctx, `#${card.badgeNumber}`, left, top);
    }
  }
}

function drawBadge(ctx: CanvasRenderingContext2D, text: string, x: number, y: number): void {
  ctx.save();
  ctx.beginPath();
  ctx.roundRect(x, y, 56, 36, [20, 0, 18, 0]);
  ctx.fillStyle = 'rgba(2, 6, 23, 0.84)';
  ctx.fill();
  drawText(ctx, text, x + 12, y + 25, '#f1f5f9', 22, '900');
  ctx.restore();
}

function drawTeamPanel(ctx: CanvasRenderingContext2D, name: string, label: string, players: string[], color: string, x: number, y: number, width: number, height: number): void {
  roundedRect(ctx, x, y, width, height, 30, 'rgba(15, 23, 42, 0.78)', `${color}cc`, 4);
  roundedRect(ctx, x + 24, y + 28, 48, 48, 24, color, `${color}00`, 0);
  drawLabel(ctx, label, x + 94, y + 54, '#a9b4c8', 19, '900', 0.16);
  drawText(ctx, name, x + 94, y + 98, neutralInk, 36, '900');
  const roster = players.length ? players.join(', ') : 'No seated players';
  drawWrappedText(ctx, roster, x + 28, y + 145, width - 56, 28, mutedInk, 25, '700', 2);
}

function roundedRect(ctx: CanvasRenderingContext2D, x: number, y: number, width: number, height: number, radius: number | number[], fill: string, stroke: string, strokeWidth: number): void {
  ctx.beginPath();
  ctx.roundRect(x, y, width, height, radius);
  if (fill !== 'rgba(0,0,0,0)') {
    ctx.fillStyle = fill;
    ctx.fill();
  }
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

function drawWrappedText(ctx: CanvasRenderingContext2D, text: string, x: number, y: number, maxWidth: number, lineHeight: number, color: string, size: number, weight: string, maxLines = 3): void {
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
  for (const [index, value] of lines.slice(0, maxLines).entries()) {
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

function hashString(value: string): number {
  let hash = 2166136261;
  for (let index = 0; index < value.length; index += 1) {
    hash ^= value.charCodeAt(index);
    hash = Math.imul(hash, 16777619);
  }
  return hash >>> 0;
}
