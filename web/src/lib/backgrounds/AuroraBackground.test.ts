/// <reference types="vite/client" />
import { describe, expect, it } from 'vitest';

import componentSource from './AuroraBackground.svelte?raw';
import shaderSource from './auroraShaders.ts?raw';

describe('AuroraBackground WebGL component contract', () => {
  it('initializes exactly one fullscreen Three.js shader scene from synchronous onMount', () => {
    expect(componentSource).toContain("import { onMount } from 'svelte'");
    expect(componentSource).toContain("from 'three'");
    expect(componentSource).toContain('new THREE.WebGLRenderer');
    expect(componentSource).toContain('new THREE.Scene');
    expect(componentSource).toContain('new THREE.OrthographicCamera');
    expect(componentSource).toContain('new THREE.PlaneGeometry(2, 2)');
    expect(componentSource).toContain('new THREE.ShaderMaterial');
    expect(componentSource).not.toContain('async () =>');
  });

  it('declares the aurora shader uniforms required for runtime tuning', () => {
    for (const uniform of ['uTime', 'uResolution', 'uIntensity', 'uSpeed', 'uMouse']) {
      expect(componentSource).toContain(uniform);
      expect(shaderSource).toContain(uniform);
    }
  });

  it('handles responsive sizing, reduced motion, hidden tabs, and teardown', () => {
    expect(componentSource).toContain('Math.min(window.devicePixelRatio || 1, 2)');
    expect(componentSource).toContain('ResizeObserver');
    expect(componentSource).toContain('prefers-reduced-motion: reduce');
    expect(componentSource).toContain('document.hidden');
    expect(componentSource).toContain('cancelAnimationFrame');
    expect(componentSource).toContain('removeEventListener');
    expect(componentSource).toContain('geometry.dispose()');
    expect(componentSource).toContain('material.dispose()');
    expect(componentSource).toContain('renderer.dispose()');
  });

  it('keeps the WebGL canvas decorative and backed by a CSS fallback', () => {
    expect(componentSource).toContain('aria-hidden="true"');
    expect(componentSource).toContain('pointer-events: none');
    expect(componentSource).toContain('aurora-fallback');
    expect(componentSource).toContain('webgl-supported');
  });
});
