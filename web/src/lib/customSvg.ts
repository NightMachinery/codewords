import spy from '../../../assets/SVG/spy.svg?raw';
import representative from '../../../assets/SVG/representative.svg?raw';
import blueCard from '../../../assets/SVG/blue-card.svg?raw';
import redCard from '../../../assets/SVG/red-card.svg?raw';
import civilianCard from '../../../assets/SVG/civilian-card.svg?raw';
import assassinCard from '../../../assets/SVG/assassin-card.svg?raw';

export const customSvg = {
  spy,
  representative,
  blueCard,
  redCard,
  civilianCard,
  assassinCard,
} as const;


export function svgAssetMarkup(svg: string, label = '', classes = 'h-4 w-4'): string {
  const safeClasses = escapeAttribute(classes);
  const aria = label ? `role="img" aria-label="${escapeAttribute(label)}"` : 'aria-hidden="true"';
  return svg.replace(/<svg\b([^>]*)>/i, `<svg$1 class="${safeClasses}" ${aria} focusable="false">`);
}

function escapeAttribute(value: string): string {
  return value.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
