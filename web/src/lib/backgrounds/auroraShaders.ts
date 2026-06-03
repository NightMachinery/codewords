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
  uniform float uLight; // 0 = dark aurora, 1 = light-theme fire

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

  float flameTongue(vec2 uv, float offset, float width, float height, float time) {
    float lift = uv.y / max(height, 0.001);
    float gust = fbm(vec2(uv.x * 2.6 + offset, uv.y * 2.0 - time * 0.42));
    float lick = sin((uv.x + offset) * 9.0 + time * 1.7 + gust * 2.2) * 0.06;
    lick += sin((uv.x - offset) * 16.0 - time * 1.15) * 0.025;
    float center = 0.5 + offset * 0.18 + lick * (0.4 + lift);
    float taper = mix(width, width * 0.16, smoothstep(0.05, 1.0, lift));
    float body = 1.0 - smoothstep(0.0, taper, abs(uv.x - center));
    float vertical = smoothstep(0.0, 0.18, uv.y) * (1.0 - smoothstep(height * 0.58, height, uv.y));
    float raggedTop = smoothstep(0.24, 0.82, gust + (1.0 - lift) * 0.38);

    return clamp(body * vertical * raggedTop, 0.0, 1.0);
  }

  vec3 renderLightFire(vec2 uv, vec3 sky, float time, float vignette) {
    vec2 fireUv = vec2(uv.x, uv.y + 0.05);
    float left = flameTongue(fireUv, -1.15, 0.34, 0.92, time);
    float center = flameTongue(fireUv + vec2(0.0, -0.03), 0.0, 0.42, 1.05, time * 1.08);
    float right = flameTongue(fireUv + vec2(0.0, 0.02), 1.08, 0.32, 0.86, time * 0.92);
    float small = flameTongue(fireUv + vec2(0.08, -0.08), 0.62, 0.2, 0.66, time * 1.32);
    float fire = clamp(left * 0.76 + center + right * 0.72 + small * 0.62, 0.0, 1.0);
    float heat = fbm(vec2(uv.x * 4.8 + time * 0.18, uv.y * 5.6 - time * 0.68));
    float whiteHotCore = pow(clamp(fire * smoothstep(0.48, 0.0, abs(uv.x - 0.5)) * (1.08 - uv.y), 0.0, 1.0), 2.1);
    float risingEmbers = pow(smoothstep(0.58, 1.0, fbm(vec2(uv.x * 18.0 + time * 0.45, uv.y * 12.0 - time * 1.35))), 6.0);
    risingEmbers *= smoothstep(0.08, 0.64, uv.y) * smoothstep(0.98, 0.35, uv.y) * 0.38;

    vec3 ember = mix(uRibbonA, uRibbonB, smoothstep(0.16, 0.78, uv.y));
    ember = mix(ember, uRibbonC, smoothstep(0.46, 0.92, heat));
    vec3 fireColor = sky;
    fireColor = mix(fireColor, ember, fire * (0.62 + heat * 0.24) * uIntensity);
    fireColor += uRibbonA * fire * 0.28 * uIntensity;
    fireColor += vec3(1.0, 0.92, 0.68) * whiteHotCore * 0.65 * uIntensity;
    fireColor += uRibbonB * risingEmbers * uIntensity;
    fireColor *= 0.97 + vignette * 0.05;

    return clamp(fireColor, 0.0, 1.0);
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

    color = mix(darkColor, renderLightFire(uv, color, time, vignette), uLight);

    gl_FragColor = vec4(color, 1.0);
  }
`;
