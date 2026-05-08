import type { MemoryCaptureModel } from './gameplay';

export type MemoryCaptureColorName = 'blue' | 'red' | 'civilian' | 'black' | 'hidden';
export type MemoryCaptureExporter = (node: HTMLElement) => Promise<Blob | null>;

interface CapturePalette {
  fill: string;
  stroke: string;
  text: string;
}

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

export async function downloadMemoryCapture(
  model: Pick<MemoryCaptureModel, 'roomId'>,
  node: HTMLElement,
  doc: Document = document,
  exporter: MemoryCaptureExporter = exportMemoryNodeBlob
): Promise<void> {
  const blob = await exporter(node);
  if (!blob) throw new Error('Could not export the memory image.');
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

async function exportMemoryNodeBlob(node: HTMLElement): Promise<Blob | null> {
  const { toBlob } = await import('html-to-image');
  return toBlob(node, {
    cacheBust: true,
    pixelRatio: 1,
    backgroundColor: '#07111f',
    width: node.scrollWidth,
    height: node.scrollHeight,
    style: {
      transform: 'none',
    },
  });
}
