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
    for (const uniform of ['uTime', 'uResolution', 'uIntensity', 'uSpeed', 'uMouse', 'uShaderVariant']) {
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

  it('preserves the current clean fire shader as a named variant', () => {
    expect(shaderSource).toContain('float flameTongue');
    expect(shaderSource).toContain('vec3 renderCleanFire');
    expect(shaderSource).toContain('whiteHotCore');
    expect(shaderSource).toContain('risingEmbers');
  });

  it('adds a separate realistic campfire shader variant', () => {
    expect(shaderSource).toContain('float campfireBody');
    expect(shaderSource).toContain('float flameLick');
    expect(shaderSource).toContain('float sparkField');
    expect(shaderSource).toContain('float smokeVeil');
    expect(shaderSource).toContain('vec3 renderCampfire');
  });

  it('routes aurora, clean fire, and campfire by the active theme variant', () => {
    expect(shaderSource).toContain('uShaderVariant < 0.5');
    expect(shaderSource).toContain('renderCleanFire');
    expect(shaderSource).toContain('renderCampfire');
    expect(componentSource).toContain('shaderVariantUniformFor');
  });
});
