export const auroraVertexShader = /* glsl */ `
  varying vec2 vUv;

  void main() {
    vUv = uv;
    gl_Position = vec4(position.xy, 0.0, 1.0);
  }
`;

export const auroraFragmentShader = /* glsl */ `
  precision highp float;

  uniform float uTime;
  uniform vec2 uResolution;
  uniform float uIntensity;
  uniform float uSpeed;
  uniform vec2 uMouse;
  uniform vec3 uSkyTop;
  uniform vec3 uSkyMid;
  uniform vec3 uSkyLow;
  uniform vec3 uRibbonA;
  uniform vec3 uRibbonB;
  uniform vec3 uRibbonC;
  uniform float uLight; // 0 = dark (additive glow), 1 = light (blend ribbons onto a bright sky)

  varying vec2 vUv;

  float hash(vec2 p) {
    p = fract(p * vec2(123.34, 456.21));
    p += dot(p, p + 45.32);
    return fract(p.x * p.y);
  }

  float noise(vec2 p) {
    vec2 i = floor(p);
    vec2 f = fract(p);
    vec2 u = f * f * (3.0 - 2.0 * f);

    return mix(
      mix(hash(i + vec2(0.0, 0.0)), hash(i + vec2(1.0, 0.0)), u.x),
      mix(hash(i + vec2(0.0, 1.0)), hash(i + vec2(1.0, 1.0)), u.x),
      u.y
    );
  }

  float fbm(vec2 p) {
    float value = 0.0;
    float amplitude = 0.5;
    mat2 rotate = mat2(0.82, -0.57, 0.57, 0.82);

    for (int i = 0; i < 5; i++) {
      value += amplitude * noise(p);
      p = rotate * p * 2.03 + 17.17;
      amplitude *= 0.5;
    }

    return value;
  }

  float curtain(vec2 uv, float offset, float scale, float time) {
    float wave = sin((uv.x + offset) * 3.2 + time * 0.55) * 0.08;
    wave += sin((uv.x * 6.4 - offset) + time * 0.24) * 0.04;

    float flow = fbm(vec2(uv.x * scale + offset + time * 0.08, uv.y * 3.0 - time * 0.12));
    float center = 0.18 + wave + flow * 0.2;
    float veil = 1.0 - smoothstep(0.0, 0.34, abs(uv.y - center));
    float verticalFade = smoothstep(0.92, 0.12, uv.y) * smoothstep(-0.12, 0.42, uv.y);
    float strand = pow(smoothstep(0.2, 1.0, flow), 2.2);

    return veil * verticalFade * (0.45 + strand * 0.8);
  }

  void main() {
    vec2 uv = vUv;
    vec2 pixel = (gl_FragCoord.xy * 2.0 - uResolution.xy) / max(uResolution.x, uResolution.y);
    float time = uTime * uSpeed;

    vec3 color = mix(uSkyLow, uSkyMid, smoothstep(0.0, 0.72, uv.y));
    color = mix(color, uSkyTop, smoothstep(0.44, 1.0, uv.y));

    float vignette = smoothstep(1.08, 0.18, length(pixel * vec2(0.84, 1.12)));
    float horizon = smoothstep(0.0, 0.72, 1.0 - uv.y);

    float a = curtain(uv + vec2(uMouse.x * 0.018, 0.0), 0.10, 3.4, time);
    float b = curtain(uv + vec2(-0.05, 0.06), 1.45, 4.2, time * 0.82);
    float c = curtain(uv + vec2(0.11, -0.04), 2.65, 5.2, time * 0.62);

    float softMask = smoothstep(1.0, 0.05, uv.y) * smoothstep(-0.08, 0.38, uv.y);
    float shimmer = 0.82 + 0.18 * fbm(vec2(uv.x * 8.0 + time * 0.16, uv.y * 2.5 - time * 0.1));

    // Dark themes: additive glow (ribbons add light to a deep sky).
    vec3 darkAurora = uRibbonA * a + uRibbonB * b * 0.72 + uRibbonC * c * 0.48;
    vec3 darkColor = color + darkAurora * softMask * shimmer * uIntensity;
    darkColor += vec3(0.03, 0.11, 0.14) * horizon * vignette * 0.34;
    darkColor *= 0.56 + vignette * 0.64;

    // Light theme: keep the same soft curtain falloff as the dark path, but render the
    // aurora as a gentle tint that *darkens* the bright sky toward the ribbon colors
    // (adding light to a near-white sky would wash out; over-blending makes harsh blobs).
    // Each ribbon contributes a small, falloff-shaped amount so the bands stay wispy.
    float la = a * softMask * shimmer * uIntensity;
    float lb = b * softMask * shimmer * uIntensity;
    float lc = c * softMask * shimmer * uIntensity;
    // Weighted average hue of the aurora at this pixel, then a soft overall strength.
    float wsum = la + lb * 0.72 + lc * 0.48 + 1e-4;
    vec3 ribbonHue = (uRibbonA * la + uRibbonB * (lb * 0.72) + uRibbonC * (lc * 0.48)) / wsum;
    float strength = clamp((la + lb * 0.72 + lc * 0.48) * 0.5, 0.0, 0.55);
    // Multiply blend: tints and darkens the sky toward the ribbon color without saturating.
    vec3 lightColor = color * mix(vec3(1.0), ribbonHue, strength);
    lightColor *= 0.94 + vignette * 0.08;

    color = mix(darkColor, lightColor, uLight);

    gl_FragColor = vec4(color, 1.0);
  }
`;
