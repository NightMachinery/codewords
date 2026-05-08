<script lang="ts">
  import { onDestroy, onMount, tick } from 'svelte';
  import { conservativeFitCardWordSize, fitCardWordBoxClasses, fitCardWordLabelStyle } from './gameplay';

  interface Props {
    segments: string[];
    classes: string;
  }

  let { segments, classes }: Props = $props();

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

  function fitLabel(): boolean {
    if (!box || !label) return false;
    const width = box.clientWidth;
    const height = box.clientHeight;
    if (width <= 0 || height <= 0) return false;

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

    fontSize = conservativeFitCardWordSize(low);
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
