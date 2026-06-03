import { describe, expect, it } from 'vitest';
import glslangFactory from '@webgpu/glslang';

import {
  auroraFragmentShaders,
  auroraVertexShader,
} from './auroraShaders';
import type { AuroraShaderVariant } from '../theme';

type Glslang = {
  compileGLSL(source: string, stage: 'vertex' | 'fragment', genDebug: boolean): Uint32Array;
};

const glslang = glslangFactory() as unknown as Glslang;

function toSpirvVertexShader(source: string): string {
  return source
    .replace('varying vec2 vUv;', 'layout(location = 0) out vec2 vUv;')
    .replace(
      'void main()',
      [
        'layout(location = 0) in vec3 position;',
        'layout(location = 1) in vec2 uv;',
        'void main()',
      ].join('\n'),
    )
    .replace(/^/, '#version 310 es\n');
}

function toSpirvFragmentShader(source: string): string {
  return source
    .replace(
      [
        'uniform float uTime;',
        '  uniform vec2 uResolution;',
        '  uniform float uIntensity;',
        '  uniform float uSpeed;',
        '  uniform vec2 uMouse;',
        '  uniform vec3 uSkyTop;',
        '  uniform vec3 uSkyMid;',
        '  uniform vec3 uSkyLow;',
        '  uniform vec3 uRibbonA;',
        '  uniform vec3 uRibbonB;',
        '  uniform vec3 uRibbonC;',
      ].join('\n'),
      [
        'layout(std140, set = 0, binding = 0) uniform AuroraUniforms {',
        '    float uTime;',
        '    vec2 uResolution;',
        '    float uIntensity;',
        '    float uSpeed;',
        '    vec2 uMouse;',
        '    vec3 uSkyTop;',
        '    vec3 uSkyMid;',
        '    vec3 uSkyLow;',
        '    vec3 uRibbonA;',
        '    vec3 uRibbonB;',
        '    vec3 uRibbonC;',
        '  };',
      ].join('\n'),
    )
    .replace('varying vec2 vUv;', 'layout(location = 0) in vec2 vUv;\n  layout(location = 0) out vec4 outColor;')
    .replaceAll('gl_FragColor =', 'outColor =')
    .replace(/^/, '#version 310 es\n');
}

function compileShader(label: string, source: string, stage: 'vertex' | 'fragment') {
  const compiled = glslang.compileGLSL(source, stage, false);
  expect(compiled.length, label).toBeGreaterThan(0);
}

describe('aurora shader GLSL compilation', () => {
  it('compiles the Three.js fullscreen vertex shader', () => {
    compileShader('aurora vertex shader', toSpirvVertexShader(auroraVertexShader), 'vertex');
  });

  it('compiles every theme-selectable fragment shader variant', () => {
    const variants = Object.entries(auroraFragmentShaders) as Array<[AuroraShaderVariant, string]>;
    for (const [variant, source] of variants) {
      compileShader(`${variant} fragment shader`, toSpirvFragmentShader(source), 'fragment');
    }
  });
});
