<script lang="ts">
  import { onDestroy, onMount, tick } from 'svelte';
  import { conservativeFitCardWordSize, fitCardWordBoxClasses, fitCardWordLabelStyle } from './gameplay';

  interface Props {
    segments: string[];
    classes: string;
    shrinkPx?: number;
    avoidTopLeftBadge?: boolean;
  }

  let { segments, classes, shrinkPx = 0, avoidTopLeftBadge = false }: Props = $props();

  // Height of the top-left number badge, measured from the word box's top edge,
  // for the widest badge (three digits, e.g. #100). When a centred label would
  // reach into this band we reserve it symmetrically (top and bottom) so the
  // label shrinks just enough to clear the badge while staying centred.
  const badgeBandPx = 14;

  let box: HTMLDivElement;
  let label: HTMLSpanElement;
  let resizeObserver: ResizeObserver | undefined;
  let frame = 0;
  let fontSize = $state(16);
  let fitReady = $state(false);

  function scheduleFit() {
    fitReady = false;
    if (frame) window.cancelAnimationFrame(frame);
    frame = window.requestAnimationFrame(async () => {
      frame = 0;
      await tick();
      fitReady = fitLabel();
    });
  }

  function searchFontSize(width: number, height: number): number {
    const minimum = 8;
    const maximum = Math.min(42, Math.max(14, width * 0.34, height * 0.5));
    let low = minimum;
    let high = maximum;

    for (let attempt = 0; attempt < 9; attempt += 1) {
      const candidate = (low + high) / 2;
      label.style.fontSize = `${candidate}px`;
      const fitsWidth = label.scrollWidth <= width + 1;
      const fitsHeight = label.scrollHeight <= height + 1;
      if (fitsWidth && fitsHeight) {
        low = candidate;
      } else {
        high = candidate;
      }
    }

    return low;
  }

  // With the fitted size applied, would the centred label's top line sit inside
  // the badge band? The label is centred in the box, so its top edge is
  // (boxHeight - labelHeight) / 2; if that is above the badge band, the first
  // line overlaps the badge.
  function centredLabelEntersBadgeBand(boxHeight: number): boolean {
    return (boxHeight - label.scrollHeight) / 2 < badgeBandPx;
  }

  function fitLabel(): boolean {
    if (!box || !label) return false;
    const width = box.clientWidth;
    const height = box.clientHeight;
    if (width <= 0 || height <= 0) return false;

    // Fit using the full box first. On a roomy card the centred label clears the
    // badge on its own and uses the whole height — no adjustment needed.
    let low = searchFontSize(width, height);
    // searchFontSize leaves the label at its last probe size; apply the result
    // so the badge-band check measures the label at the size we will keep.
    label.style.fontSize = `${low}px`;

    // Otherwise reserve the badge band symmetrically (top and bottom) and re-fit
    // into the shorter height. The label stays vertically centred but shrinks
    // just enough that its first line drops below the badge. Reserving both ends
    // keeps it centred; reserving only the top would push it downward.
    if (avoidTopLeftBadge && centredLabelEntersBadgeBand(height)) {
      const clearedHeight = height - 2 * badgeBandPx;
      if (clearedHeight > 0) {
        low = Math.min(low, searchFontSize(width, clearedHeight));
      }
    }

    fontSize = conservativeFitCardWordSize(low, shrinkPx);
    label.style.fontSize = `${fontSize}px`;
    return true;
  }

  onMount(() => {
    resizeObserver = new ResizeObserver(scheduleFit);
    resizeObserver.observe(box);
    scheduleFit();
    void document.fonts?.ready.then(scheduleFit).catch(() => undefined);
  });

  onDestroy(() => {
    if (frame) window.cancelAnimationFrame(frame);
    resizeObserver?.disconnect();
  });

  $effect(() => {
    segments;
    classes;
    shrinkPx;
    avoidTopLeftBadge;
    scheduleFit();
  });
</script>

<div bind:this={box} data-fit-card-word data-fit-ready={fitReady ? 'true' : 'false'} class={fitCardWordBoxClasses()}>
  <span bind:this={label} class={classes} style={fitCardWordLabelStyle(fontSize)} dir="auto">
    {#each segments as segment, segmentIndex (segmentIndex)}
      {segment}{#if segmentIndex < segments.length - 1}<wbr />{/if}
    {/each}
  </span>
</div>
