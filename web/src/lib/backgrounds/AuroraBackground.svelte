<script lang="ts">
  import { onMount } from 'svelte';
  import * as THREE from 'three';

  import { auroraFragmentShader, auroraVertexShader } from './auroraShaders';

  type Props = {
    intensity?: number;
    speed?: number;
    class?: string;
  };

  let { intensity = 0.74, speed = 0.16, class: className = '' }: Props = $props();

  let host: HTMLDivElement;
  let canvas: HTMLCanvasElement;
  let webglSupported = $state(false);

  onMount(() => {
    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');
    const uniforms = {
      uTime: { value: 0 },
      uResolution: { value: new THREE.Vector2(1, 1) },
      uIntensity: { value: intensity },
      uSpeed: { value: speed },
      uMouse: { value: new THREE.Vector2(0.5, 0.5) },
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
      if (reducedMotion.matches) {
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

    if (reducedMotion.matches) {
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
