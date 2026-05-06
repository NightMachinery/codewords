<script lang="ts">
  import { onDestroy, onMount, tick } from 'svelte';

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

  function scheduleFit() {
    if (frame) window.cancelAnimationFrame(frame);
    frame = window.requestAnimationFrame(async () => {
      frame = 0;
      await tick();
      fitLabel();
    });
  }

  function fitLabel() {
    if (!box || !label) return;
    const width = box.clientWidth;
    const height = box.clientHeight;
    if (width <= 0 || height <= 0) return;

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

    fontSize = Math.max(minimum, Math.floor(low * 10) / 10);
    label.style.fontSize = `${fontSize}px`;
  }

  onMount(() => {
    resizeObserver = new ResizeObserver(scheduleFit);
    resizeObserver.observe(box);
    scheduleFit();
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

<div bind:this={box} class="absolute inset-1.5 grid min-h-0 min-w-0 place-items-center overflow-visible [container-type:inline-size]">
  <span bind:this={label} class={classes} style={`font-size: ${fontSize}px; overflow: visible;`} dir="auto">
    {#each segments as segment, segmentIndex}
      {segment}{#if segmentIndex < segments.length - 1}<wbr />{/if}
    {/each}
  </span>
</div>
