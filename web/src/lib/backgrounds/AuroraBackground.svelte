<script lang="ts">
  import { onMount } from 'svelte';
  import * as THREE from 'three';

  import { auroraFragmentShader, auroraVertexShader } from './auroraShaders';
  import { auroraPaletteFor, themeMode, type ThemeId } from '../theme';

  type Props = {
    intensity?: number;
    speed?: number;
    theme?: ThemeId;
    class?: string;
  };

  let { intensity = 0.74, speed = 0.16, theme = 'dark', class: className = '' }: Props = $props();

  let host: HTMLDivElement;
  let canvas: HTMLCanvasElement;
  let webglSupported = $state(false);
  // Color uniforms are updated reactively by the $effect below when `theme` changes.
  const colorUniforms = {
    uSkyTop: { value: new THREE.Vector3() },
    uSkyMid: { value: new THREE.Vector3() },
    uSkyLow: { value: new THREE.Vector3() },
    uRibbonA: { value: new THREE.Vector3() },
    uRibbonB: { value: new THREE.Vector3() },
    uRibbonC: { value: new THREE.Vector3() },
    uLight: { value: 0 },
  };

  $effect(() => {
    const palette = auroraPaletteFor(theme);
    colorUniforms.uSkyTop.value.set(...palette.skyTop);
    colorUniforms.uSkyMid.value.set(...palette.skyMid);
    colorUniforms.uSkyLow.value.set(...palette.skyLow);
    colorUniforms.uRibbonA.value.set(...palette.ribbonA);
    colorUniforms.uRibbonB.value.set(...palette.ribbonB);
    colorUniforms.uRibbonC.value.set(...palette.ribbonC);
    colorUniforms.uLight.value = themeMode(theme) === 'light' ? 1 : 0;
  });

  onMount(() => {
    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');
    const forceMotion = true;

    const uniforms = {
      uTime: { value: 0 },
      uResolution: { value: new THREE.Vector2(1, 1) },
      uIntensity: { value: intensity },
      uSpeed: { value: speed },
      uMouse: { value: new THREE.Vector2(0.5, 0.5) },
      ...colorUniforms,
    };

    let renderer: THREE.WebGLRenderer;

    try {
      renderer = new THREE.WebGLRenderer({
        canvas,
        alpha: true,
        antialias: false,
        powerPreference: 'high-performance',
      });
    } catch {
      return;
    }

    webglSupported = true;
    renderer.setClearColor(0x000000, 0);
    renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));

    const scene = new THREE.Scene();
    const camera = new THREE.OrthographicCamera(-1, 1, 1, -1, 0, 1);
    const geometry = new THREE.PlaneGeometry(2, 2);
    const material = new THREE.ShaderMaterial({
      vertexShader: auroraVertexShader,
      fragmentShader: auroraFragmentShader,
      uniforms,
      depthWrite: false,
      depthTest: false,
    });
    const mesh = new THREE.Mesh(geometry, material);
    scene.add(mesh);

    let frameId = 0;
    let lastTime = performance.now();

    const resize = () => {
      const rect = host.getBoundingClientRect();
      const width = Math.max(1, Math.floor(rect.width));
      const height = Math.max(1, Math.floor(rect.height));
      renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
      renderer.setSize(width, height, false);
      uniforms.uResolution.value.set(width, height).multiplyScalar(renderer.getPixelRatio());
      renderer.render(scene, camera);
    };

    const render = (now: number) => {
      frameId = requestAnimationFrame(render);

      if (document.hidden) {
        lastTime = now;
        return;
      }

      const deltaSeconds = Math.min(0.05, (now - lastTime) / 1000);
      lastTime = now;
      uniforms.uTime.value += deltaSeconds;
      uniforms.uIntensity.value = intensity;
      uniforms.uSpeed.value = speed;
      renderer.render(scene, camera);
    };

    const handleMouseMove = (event: PointerEvent) => {
      const rect = host.getBoundingClientRect();
      uniforms.uMouse.value.set(
        (event.clientX - rect.left) / Math.max(rect.width, 1),
        1 - (event.clientY - rect.top) / Math.max(rect.height, 1),
      );
    };

    const handleVisibilityChange = () => {
      if (!document.hidden) {
        lastTime = performance.now();
        renderer.render(scene, camera);
      }
    };

    const handleReducedMotionChange = () => {
      if (! forceMotion && reducedMotion.matches) {
        cancelAnimationFrame(frameId);
        frameId = 0;
        uniforms.uTime.value = 3.8;
        renderer.render(scene, camera);
      } else if (frameId === 0) {
        lastTime = performance.now();
        frameId = requestAnimationFrame(render);
      }
    };

    const resizeObserver = new ResizeObserver(resize);
    resizeObserver.observe(host);
    window.addEventListener('resize', resize);
    host.addEventListener('pointermove', handleMouseMove, { passive: true });
    document.addEventListener('visibilitychange', handleVisibilityChange);
    reducedMotion.addEventListener('change', handleReducedMotionChange);

    resize();

    if (! forceMotion && reducedMotion.matches) {
      uniforms.uTime.value = 3.8;
      renderer.render(scene, camera);
    } else {
      frameId = requestAnimationFrame(render);
    }

    return () => {
      cancelAnimationFrame(frameId);
      resizeObserver.disconnect();
      window.removeEventListener('resize', resize);
      host.removeEventListener('pointermove', handleMouseMove);
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      reducedMotion.removeEventListener('change', handleReducedMotionChange);
      scene.remove(mesh);
      geometry.dispose();
      material.dispose();
      renderer.dispose();
    };
  });
</script>

<div
  bind:this={host}
  class={["aurora-background", className, webglSupported && "webgl-supported"]}
  aria-hidden="true"
>
  <div class="aurora-fallback"></div>
  <canvas bind:this={canvas}></canvas>
</div>

<style>
  .aurora-background {
    position: absolute;
    inset: 0;
    overflow: hidden;
    pointer-events: none;
    /* Base color while the WebGL canvas fades in; overridden per theme below. */
    background: oklch(7% 0.026 260);
    isolation: isolate;
  }

  .aurora-fallback,
  canvas {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
  }

  /*
   * The CSS fallback (and base color) mirror the shader palettes so the hero stays
   * cohesive before/without WebGL. Keyed off [data-theme] on <html>.
   */
  .aurora-fallback {
    background:
      radial-gradient(circle at 50% 118%, oklch(21% 0.035 258 / 0.78), transparent 46%),
      radial-gradient(ellipse at 28% 18%, oklch(76% 0.16 160 / 0.18), transparent 34%),
      radial-gradient(ellipse at 74% 24%, oklch(66% 0.13 220 / 0.15), transparent 36%),
      radial-gradient(ellipse at 52% 4%, oklch(62% 0.13 304 / 0.11), transparent 30%),
      linear-gradient(180deg, oklch(7% 0.03 260), oklch(12% 0.035 242) 58%, oklch(6% 0.025 265));
  }

  .aurora-fallback::after {
    content: '';
    position: absolute;
    inset: 0;
    background: linear-gradient(180deg, transparent, oklch(5% 0.024 260 / 0.58) 78%);
  }

  :global([data-theme='light']) .aurora-background {
    background: oklch(95% 0.012 250);
  }

  :global([data-theme='light']) .aurora-fallback {
    background:
      radial-gradient(ellipse at 50% 92%, oklch(92% 0.16 72 / 0.58), transparent 42%),
      radial-gradient(ellipse at 40% 74%, oklch(78% 0.22 42 / 0.34), transparent 30%),
      radial-gradient(ellipse at 62% 68%, oklch(72% 0.22 25 / 0.28), transparent 28%),
      radial-gradient(ellipse at 52% 50%, oklch(98% 0.08 88 / 0.24), transparent 25%),
      linear-gradient(180deg, oklch(97% 0.025 78), oklch(94% 0.04 72) 58%, oklch(90% 0.07 58));
  }

  :global([data-theme='light']) .aurora-fallback::after {
    background: linear-gradient(180deg, transparent 42%, oklch(88% 0.12 52 / 0.44) 88%);
  }

  :global([data-theme='solarized-light']) .aurora-background {
    background: oklch(96% 0.025 90);
  }

  :global([data-theme='solarized-light']) .aurora-fallback {
    background:
      radial-gradient(ellipse at 50% 92%, oklch(88% 0.15 76 / 0.58), transparent 42%),
      radial-gradient(ellipse at 40% 74%, oklch(74% 0.2 42 / 0.32), transparent 30%),
      radial-gradient(ellipse at 62% 68%, oklch(66% 0.19 22 / 0.26), transparent 28%),
      radial-gradient(ellipse at 52% 50%, oklch(97% 0.07 88 / 0.2), transparent 25%),
      linear-gradient(180deg, oklch(98% 0.02 90), oklch(94% 0.04 82) 58%, oklch(90% 0.06 64));
  }

  :global([data-theme='solarized-light']) .aurora-fallback::after {
    background: linear-gradient(180deg, transparent 42%, oklch(86% 0.11 52 / 0.42) 88%);
  }

  :global([data-theme='matrix']) .aurora-background {
    background: oklch(8% 0.04 152);
  }

  :global([data-theme='matrix']) .aurora-fallback {
    background:
      radial-gradient(circle at 50% 118%, oklch(24% 0.08 152 / 0.78), transparent 46%),
      radial-gradient(ellipse at 28% 18%, oklch(80% 0.2 152 / 0.18), transparent 34%),
      radial-gradient(ellipse at 74% 24%, oklch(70% 0.18 158 / 0.15), transparent 36%),
      linear-gradient(180deg, oklch(8% 0.04 152), oklch(13% 0.06 152) 58%, oklch(7% 0.035 152));
  }

  :global([data-theme='matrix']) .aurora-fallback::after {
    background: linear-gradient(180deg, transparent, oklch(6% 0.03 152 / 0.58) 78%);
  }

  canvas {
    display: block;
    opacity: 0;
    transition: opacity 800ms ease;
  }

  .webgl-supported canvas {
    opacity: 1;
  }

  @media (prefers-reduced-motion: reduce) {
    canvas {
      transition: none;
    }
  }
</style>
