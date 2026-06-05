import type { AuroraShaderVariant } from '../theme';

export const auroraVertexShader = /* glsl */ `
  varying vec2 vUv;

  void main() {
    vUv = uv;
    gl_Position = vec4(position.xy, 0.0, 1.0);
  }
`;

const commonFragmentPrelude = /* glsl */ `
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

  vec3 skyColor(vec2 uv) {
    vec3 color = mix(uSkyLow, uSkyMid, smoothstep(0.0, 0.72, uv.y));
    return mix(color, uSkyTop, smoothstep(0.44, 1.0, uv.y));
  }

  float vignetteFor(vec2 pixel) {
    return smoothstep(1.08, 0.18, length(pixel * vec2(0.84, 1.12)));
  }
`;

export const auroraFragmentShader = /* glsl */ `
${commonFragmentPrelude}

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
    vec3 color = skyColor(uv);
    float vignette = vignetteFor(pixel);
    float horizon = smoothstep(0.0, 0.72, 1.0 - uv.y);

    float a = curtain(uv + vec2(uMouse.x * 0.018, 0.0), 0.10, 3.4, time);
    float b = curtain(uv + vec2(-0.05, 0.06), 1.45, 4.2, time * 0.82);
    float c = curtain(uv + vec2(0.11, -0.04), 2.65, 5.2, time * 0.62);

    float softMask = smoothstep(1.0, 0.05, uv.y) * smoothstep(-0.08, 0.38, uv.y);
    float shimmer = 0.82 + 0.18 * fbm(vec2(uv.x * 8.0 + time * 0.16, uv.y * 2.5 - time * 0.1));
    vec3 darkAurora = uRibbonA * a + uRibbonB * b * 0.72 + uRibbonC * c * 0.48;
    color += darkAurora * softMask * shimmer * uIntensity;
    color += vec3(0.03, 0.11, 0.14) * horizon * vignette * 0.34;
    color *= 0.56 + vignette * 0.64;

    gl_FragColor = vec4(color, 1.0);
  }
`;

export const cleanFireFragmentShader = /* glsl */ `
${commonFragmentPrelude}

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

  void main() {
    vec2 uv = vUv;
    vec2 pixel = (gl_FragCoord.xy * 2.0 - uResolution.xy) / max(uResolution.x, uResolution.y);
    float time = uTime * uSpeed;
    vec3 sky = skyColor(uv);
    float vignette = vignetteFor(pixel);

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
    vec3 color = sky;
    color = mix(color, ember, fire * (0.62 + heat * 0.24) * uIntensity);
    color += uRibbonA * fire * 0.28 * uIntensity;
    color += vec3(1.0, 0.92, 0.68) * whiteHotCore * 0.65 * uIntensity;
    color += uRibbonB * risingEmbers * uIntensity;
    color *= 0.97 + vignette * 0.05;

    gl_FragColor = vec4(clamp(color, 0.0, 1.0), 1.0);
  }
`;

export const campfireFragmentShader = /* glsl */ `
${commonFragmentPrelude}

  float campfireBody(vec2 uv, float center, float width, float height, float time) {
    float lift = uv.y / max(height, 0.001);
    float turbulence = fbm(vec2(uv.x * 4.0 + center * 3.0, uv.y * 3.2 - time * 0.62));
    float wind = sin((uv.x + center) * 5.6 + time * 1.25 + turbulence * 2.2) * 0.06;
    float bodyCenter = center + wind * smoothstep(0.12, 0.86, lift);
    float taper = width * (1.0 - smoothstep(0.04, 1.0, lift) * 0.72);
    float body = 1.0 - smoothstep(0.0, taper, abs(uv.x - bodyCenter));
    float bottom = smoothstep(-0.05, 0.14, uv.y);
    float top = 1.0 - smoothstep(height * (0.54 + turbulence * 0.22), height, uv.y);
    float brokenEdge = smoothstep(0.1, 0.92, turbulence + (1.0 - lift) * 0.28);

    return clamp(body * bottom * top * brokenEdge, 0.0, 1.0);
  }

  float flameLick(vec2 uv, float center, float width, float height, float time) {
    float lift = uv.y / max(height, 0.001);
    float turbulence = fbm(vec2(uv.x * 6.5 + center * 4.0 - time * 0.1, uv.y * 5.4 - time * 0.95));
    float sway = sin((uv.x + center) * 13.0 + time * 2.0 + turbulence * 3.0) * 0.05;
    float lickCenter = center + sway * (0.35 + lift);
    float taper = width * (1.0 - smoothstep(0.05, 0.94, lift));
    float tongue = 1.0 - smoothstep(0.0, max(taper, 0.018), abs(uv.x - lickCenter));
    float vertical = smoothstep(0.04, 0.22, uv.y) * (1.0 - smoothstep(height * 0.72, height, uv.y));
    float torn = smoothstep(0.28, 0.86, turbulence + (1.0 - lift) * 0.18);

    return clamp(tongue * vertical * torn, 0.0, 1.0);
  }

  float sparkField(vec2 uv, float time) {
    vec2 grid = vec2(uv.x * 34.0, uv.y * 18.0 - time * 2.2);
    vec2 cell = floor(grid);
    vec2 local = fract(grid) - 0.5;
    float seed = hash(cell);
    vec2 jitter = vec2(seed - 0.5, hash(cell + vec2(13.7, 13.7)) - 0.5) * 0.28;
    float spark = 1.0 - smoothstep(0.0, 0.09, length(local + jitter));
    float sparkActive = step(0.965, seed);
    float heightFade = smoothstep(0.22, 0.78, uv.y) * (1.0 - smoothstep(0.46, 1.0, uv.y));

    return spark * sparkActive * heightFade;
  }

  float smokeVeil(vec2 uv, float time) {
    float smoke = fbm(vec2(uv.x * 2.4 + time * 0.08, uv.y * 2.7 - time * 0.32));
    float plume = smoothstep(0.34, 0.82, uv.y) * (1.0 - smoothstep(0.46, 1.02, uv.y));
    float centerFade = 1.0 - smoothstep(0.12, 0.62, abs(uv.x - 0.54));

    return smoothstep(0.44, 0.82, smoke) * plume * centerFade;
  }

  void main() {
    vec2 uv = vUv;
    vec2 pixel = (gl_FragCoord.xy * 2.0 - uResolution.xy) / max(uResolution.x, uResolution.y);
    float time = uTime * uSpeed;
    vec3 sky = skyColor(uv);
    float vignette = vignetteFor(pixel);

    vec2 fireUv = vec2(uv.x, uv.y + 0.03);
    float leftBody = campfireBody(fireUv, 0.38, 0.28, 0.78, time);
    float centerBody = campfireBody(fireUv + vec2(0.0, -0.04), 0.52, 0.34, 0.96, time * 1.07);
    float rightBody = campfireBody(fireUv + vec2(0.0, 0.02), 0.66, 0.25, 0.82, time * 0.92);
    float bodies = clamp(leftBody * 0.72 + centerBody + rightBody * 0.68, 0.0, 1.0);

    float lickA = flameLick(fireUv, 0.43, 0.13, 1.08, time * 1.18);
    float lickB = flameLick(fireUv + vec2(0.04, -0.02), 0.58, 0.12, 0.98, time * 0.98);
    float lickC = flameLick(fireUv + vec2(-0.03, 0.04), 0.69, 0.09, 0.76, time * 1.42);
    float licks = clamp(lickA * 0.74 + lickB + lickC * 0.56, 0.0, 1.0);
    float flame = clamp(bodies + licks, 0.0, 1.0);
    float heat = fbm(vec2(uv.x * 7.0 + time * 0.2, uv.y * 6.8 - time * 0.86));
    float emberBase = bodies * (1.0 - smoothstep(0.02, 0.42, uv.y));
    float whiteHotCore = pow(clamp(flame * (1.0 - smoothstep(0.0, 0.34, abs(uv.x - 0.52))) * (1.0 - smoothstep(0.1, 0.76, uv.y)), 0.0, 1.0), 2.35);
    float sparks = sparkField(uv, time) * uIntensity;
    float smoke = smokeVeil(uv, time);

    vec3 emberRed = vec3(0.6, 0.09, 0.035);
    vec3 deepOrange = vec3(0.95, 0.28, 0.04);
    vec3 golden = vec3(1.0, 0.68, 0.16);
    vec3 hotCore = vec3(1.0, 0.94, 0.68);
    vec3 smokeTint = vec3(0.48, 0.42, 0.36);

    vec3 flameRamp = mix(deepOrange, golden, smoothstep(0.18, 0.82, heat + uv.y * 0.36));
    flameRamp = mix(emberRed, flameRamp, smoothstep(0.04, 0.38, uv.y));

    vec3 color = sky;
    color = mix(color, flameRamp, flame * (0.76 + heat * 0.18) * uIntensity);
    color += emberRed * emberBase * 0.46 * uIntensity;
    color += hotCore * whiteHotCore * 0.72 * uIntensity;
    color += golden * sparks * 0.55;
    color = mix(color, smokeTint, smoke * 0.1);
    color *= 0.95 + vignette * 0.07;

    gl_FragColor = vec4(clamp(color, 0.0, 1.0), 1.0);
  }
`;

export const draculaHomeFragmentShader = /* glsl */ `
${commonFragmentPrelude}

  float spectralRibbon(vec2 uv, float offset, float scale, float time) {
    float sweep = sin((uv.x + offset) * 2.8 + time * 0.42) * 0.1;
    sweep += sin((uv.x * 7.0 - offset) + time * 0.22) * 0.04;
    float mist = fbm(vec2(uv.x * scale + offset + time * 0.09, uv.y * 2.6 - time * 0.11));
    float center = 0.24 + sweep + mist * 0.22;
    float body = 1.0 - smoothstep(0.0, 0.36, abs(uv.y - center));
    float taper = smoothstep(0.94, 0.08, uv.y) * smoothstep(-0.1, 0.48, uv.y);
    float strands = pow(smoothstep(0.24, 1.0, mist), 2.4);

    return body * taper * (0.32 + strands * 0.92);
  }

  void main() {
    vec2 uv = vUv;
    vec2 pixel = (gl_FragCoord.xy * 2.0 - uResolution.xy) / max(uResolution.x, uResolution.y);
    float time = uTime * uSpeed;
    float vignette = vignetteFor(pixel);
    vec3 color = skyColor(uv);

    float veil = fbm(vec2(uv.x * 2.4 + time * 0.05, uv.y * 2.0 - time * 0.07));
    float a = spectralRibbon(uv + vec2(uMouse.x * 0.014, 0.0), 0.18, 3.2, time);
    float b = spectralRibbon(uv + vec2(-0.08, 0.08), 1.44, 4.6, time * 0.74);
    float c = spectralRibbon(uv + vec2(0.12, -0.03), 2.5, 5.6, time * 0.58);
    float horizon = smoothstep(0.0, 0.76, 1.0 - uv.y);
    vec3 ribbons = uRibbonA * a + uRibbonB * b * 0.72 + uRibbonC * c * 0.52;

    color += ribbons * (0.72 + veil * 0.22) * uIntensity;
    color += uRibbonB * pow(smoothstep(0.68, 1.0, veil), 4.0) * horizon * 0.22 * uIntensity;
    color += uRibbonC * horizon * vignette * 0.12;
    color *= 0.5 + vignette * 0.68;

    gl_FragColor = vec4(clamp(color, 0.0, 1.0), 1.0);
  }
`;

export const draculaBoardFragmentShader = /* glsl */ `
${commonFragmentPrelude}

  float lowFog(vec2 uv, float offset, float time) {
    float roll = fbm(vec2(uv.x * 2.8 + offset + time * 0.06, uv.y * 3.0 - time * 0.08));
    float band = 1.0 - smoothstep(0.0, 0.42, abs(uv.y - (0.34 + roll * 0.2)));
    float floorFade = smoothstep(-0.06, 0.54, uv.y) * smoothstep(1.04, 0.1, uv.y);
    return band * floorFade * smoothstep(0.22, 0.9, roll);
  }

  void main() {
    vec2 uv = vUv;
    vec2 pixel = (gl_FragCoord.xy * 2.0 - uResolution.xy) / max(uResolution.x, uResolution.y);
    float time = uTime * uSpeed;
    float vignette = vignetteFor(pixel);
    float gridPulse = fbm(vec2(uv.x * 7.0 + time * 0.08, uv.y * 4.8 - time * 0.12));

    vec3 color = mix(uSkyLow * 0.58, uSkyMid * 0.72, smoothstep(0.0, 1.0, uv.y));
    float purpleFog = lowFog(uv, 0.2, time);
    float pinkFog = lowFog(uv + vec2(0.08, 0.06), 1.7, time * 0.82);
    float cyanLine = pow(smoothstep(0.72, 1.0, gridPulse), 3.0) * smoothstep(0.05, 0.82, uv.y);

    color += uRibbonA * purpleFog * 0.62 * uIntensity;
    color += uRibbonB * pinkFog * 0.38 * uIntensity;
    color += uRibbonC * cyanLine * 0.16 * uIntensity;
    color *= 0.64 + vignette * 0.5;

    gl_FragColor = vec4(clamp(color, 0.0, 1.0), 1.0);
  }
`;

export const draculaCardFragmentShader = /* glsl */ `
${commonFragmentPrelude}

  float diagonalSheen(vec2 uv, float time) {
    float stripe = uv.x * 1.1 + uv.y * 0.72 + sin(uv.y * 5.0 + time * 0.6) * 0.05;
    float sweepA = 1.0 - smoothstep(0.0, 0.12, abs(fract(stripe - time * 0.09) - 0.5));
    float sweepB = 1.0 - smoothstep(0.0, 0.06, abs(fract(stripe * 1.7 + time * 0.07) - 0.5));
    return sweepA * 0.5 + sweepB * 0.24;
  }

  void main() {
    vec2 uv = vUv;
    vec2 pixel = (gl_FragCoord.xy * 2.0 - uResolution.xy) / max(uResolution.x, uResolution.y);
    float time = uTime * uSpeed;
    float vignette = vignetteFor(pixel);
    float noiseVeil = fbm(vec2(uv.x * 9.0 + time * 0.18, uv.y * 7.0 - time * 0.16));
    float sheen = diagonalSheen(uv, time) * smoothstep(0.2, 0.92, noiseVeil);
    float edge = 1.0 - smoothstep(0.2, 0.88, length(pixel * vec2(0.82, 1.08)));

    vec3 color = vec3(0.0);
    color += uRibbonA * sheen * 0.5 * uIntensity;
    color += uRibbonB * pow(sheen, 1.8) * 0.42 * uIntensity;
    color += uRibbonC * pow(smoothstep(0.72, 1.0, noiseVeil), 4.0) * 0.18 * uIntensity;
    color += uSkyMid * edge * 0.18;
    color *= 0.55 + vignette * 0.62;

    gl_FragColor = vec4(clamp(color, 0.0, 1.0), 1.0);
  }
`;

export const glitchHomeFragmentShader = /* glsl */ `
${commonFragmentPrelude}

  float scanline(vec2 uv) {
    return 0.78 + 0.22 * sin(uv.y * uResolution.y * 2.15);
  }

  float signalBand(vec2 uv, float time, float seed) {
    float drift = fract(time * (0.035 + seed * 0.014) + seed);
    float wobble = noise(vec2(floor(uv.y * 42.0), seed * 31.0 + floor(time * 6.0)));
    float center = fract(drift + wobble * 0.12);
    float band = 1.0 - smoothstep(0.0, 0.045 + seed * 0.018, abs(uv.y - center));
    float tear = step(0.64, noise(vec2(floor(uv.y * 18.0), floor(time * 9.0) + seed)));
    return band * (0.35 + tear * 0.9);
  }

  float blockNoise(vec2 uv, float time) {
    vec2 cell = floor(vec2(uv.x * 34.0, uv.y * 22.0));
    float n = hash(cell + floor(time * 12.0));
    return step(0.972, n) * (0.45 + 0.55 * hash(cell + 11.7));
  }

  void main() {
    vec2 uv = vUv;
    vec2 pixel = (gl_FragCoord.xy * 2.0 - uResolution.xy) / max(uResolution.x, uResolution.y);
    float time = uTime * uSpeed;
    float vignette = vignetteFor(pixel);

    float slice = floor(uv.y * 72.0);
    float tearNoise = noise(vec2(slice * 0.37, floor(time * 10.0)));
    float tearActive = step(0.77, tearNoise);
    float offset = (tearNoise - 0.5) * 0.07 * tearActive;
    vec2 signalUv = uv + vec2(offset + sin(uv.y * 38.0 + time * 2.4) * 0.005, 0.0);

    vec3 color = skyColor(signalUv);
    float lowFuzz = fbm(vec2(signalUv.x * 5.8 + time * 0.26, signalUv.y * 7.0 - time * 0.34));
    float bandA = signalBand(signalUv, time, 0.12);
    float bandB = signalBand(signalUv + vec2(0.03, 0.11), time * 1.27, 0.54);
    float chromaA = 1.0 - smoothstep(0.0, 0.28, abs(signalUv.y - (0.3 + sin(signalUv.x * 4.2 + time) * 0.08)));
    float chromaB = 1.0 - smoothstep(0.0, 0.22, abs(signalUv.y - (0.56 + sin(signalUv.x * 6.1 - time * 0.7) * 0.06)));
    float blocks = blockNoise(signalUv, time);

    color += uRibbonA * (bandA * 0.7 + chromaA * 0.26 + blocks * 0.36) * uIntensity;
    color += uRibbonB * (bandB * 0.52 + lowFuzz * 0.12) * uIntensity;
    color += uRibbonC * (chromaB * 0.34 + blocks * 0.42) * uIntensity;
    color += vec3(0.08, 0.18, 0.2) * pow(smoothstep(0.62, 1.0, lowFuzz), 3.0);
    color *= scanline(uv);
    color *= 0.48 + vignette * 0.72;

    gl_FragColor = vec4(clamp(color, 0.0, 1.0), 1.0);
  }
`;

export const glitchBoardFragmentShader = /* glsl */ `
${commonFragmentPrelude}

  float rollBand(vec2 uv, float time) {
    float center = fract(0.92 - time * 0.045);
    float wide = 1.0 - smoothstep(0.0, 0.18, abs(uv.y - center));
    float detail = fbm(vec2(uv.x * 8.0 + time * 0.12, uv.y * 16.0 - time * 0.2));
    return wide * smoothstep(0.28, 0.86, detail);
  }

  float brokenSlice(vec2 uv, float time) {
    float row = floor(uv.y * 38.0);
    float activeFlag = step(0.78, hash(vec2(row, floor(time * 5.0))));
    float edge = smoothstep(0.0, 0.08, fract(uv.y * 38.0)) * smoothstep(1.0, 0.86, fract(uv.y * 38.0));
    return activeFlag * edge;
  }

  void main() {
    vec2 uv = vUv;
    vec2 pixel = (gl_FragCoord.xy * 2.0 - uResolution.xy) / max(uResolution.x, uResolution.y);
    float time = uTime * uSpeed;
    float vignette = vignetteFor(pixel);
    float slice = brokenSlice(uv, time);
    float horizontalOffset = (noise(vec2(floor(uv.y * 38.0), floor(time * 5.0))) - 0.5) * 0.045 * slice;
    vec2 signalUv = uv + vec2(horizontalOffset, 0.0);
    float interference = rollBand(signalUv, time);
    float staticField = fbm(vec2(signalUv.x * 18.0 + time * 0.08, signalUv.y * 13.0 - time * 0.18));
    float thinLines = 0.85 + 0.15 * sin(signalUv.y * uResolution.y * 1.6);

    vec3 color = mix(uSkyLow * 0.44, uSkyMid * 0.58, smoothstep(0.0, 1.0, signalUv.y));
    color += uRibbonA * interference * 0.22 * uIntensity;
    color += uRibbonB * pow(smoothstep(0.66, 1.0, staticField), 3.0) * 0.16 * uIntensity;
    color += uRibbonC * slice * 0.18 * uIntensity;
    color *= thinLines;
    color *= 0.54 + vignette * 0.54;

    gl_FragColor = vec4(clamp(color, 0.0, 1.0), 1.0);
  }
`;

export const glitchCardFragmentShader = /* glsl */ `
${commonFragmentPrelude}

  float tinyBlocks(vec2 uv, float time) {
    vec2 cell = floor(vec2(uv.x * 56.0, uv.y * 34.0));
    float n = hash(cell + floor(time * 18.0));
    return step(0.982, n);
  }

  void main() {
    vec2 uv = vUv;
    vec2 pixel = (gl_FragCoord.xy * 2.0 - uResolution.xy) / max(uResolution.x, uResolution.y);
    float time = uTime * uSpeed;
    float vignette = vignetteFor(pixel);
    float jitter = (noise(vec2(floor(uv.y * 86.0), floor(time * 16.0))) - 0.5) * 0.026;
    vec2 signalUv = uv + vec2(jitter, 0.0);
    float scan = 0.72 + 0.28 * sin(signalUv.y * uResolution.y * 2.8);
    float line = pow(1.0 - smoothstep(0.0, 0.05, abs(fract(signalUv.y * 22.0 - time * 0.35) - 0.5)), 2.0);
    float blocks = tinyBlocks(signalUv, time);
    float edge = 1.0 - smoothstep(0.2, 0.88, length(pixel * vec2(0.82, 1.08)));

    vec3 color = vec3(0.0);
    color += uRibbonA * (line * 0.26 + blocks * 0.42) * uIntensity;
    color += uRibbonB * blocks * 0.32 * uIntensity;
    color += uRibbonC * (line * 0.22 + edge * 0.08) * uIntensity;
    color += uSkyMid * edge * 0.12;
    color *= scan;
    color *= 0.54 + vignette * 0.62;

    gl_FragColor = vec4(clamp(color, 0.0, 1.0), 1.0);
  }
`;

export const christmasCozyHomeFragmentShader = /* glsl */ `
${commonFragmentPrelude}

  float snowField(vec2 uv, float time, float scale, float speed) {
    vec2 grid = vec2(uv.x * scale, uv.y * scale * 0.62 + time * speed);
    vec2 cell = floor(grid);
    vec2 local = fract(grid) - 0.5;
    float seed = hash(cell);
    vec2 drift = vec2(sin(time * 0.42 + seed * 6.28), cos(time * 0.25 + seed * 4.12)) * 0.18;
    float flake = 1.0 - smoothstep(0.0, 0.075 + seed * 0.035, length(local + drift));
    return flake * step(0.78, seed);
  }

  float stringLight(vec2 uv, float time) {
    float wire = 1.0 - smoothstep(0.0, 0.035, abs(uv.y - (0.66 + sin(uv.x * 7.0 + time * 0.32) * 0.045)));
    float bulb = pow(smoothstep(0.82, 1.0, sin(uv.x * 38.0 - time * 1.6) * 0.5 + 0.5), 5.0);
    return wire * bulb;
  }

  void main() {
    vec2 uv = vUv;
    vec2 pixel = (gl_FragCoord.xy * 2.0 - uResolution.xy) / max(uResolution.x, uResolution.y);
    float time = uTime * uSpeed;
    float vignette = vignetteFor(pixel);
    float mist = fbm(vec2(uv.x * 2.2 + time * 0.05, uv.y * 2.8 - time * 0.06));
    float ribbon = 1.0 - smoothstep(0.0, 0.32, abs(uv.y - (0.28 + mist * 0.26 + sin(uv.x * 3.2 + time * 0.22) * 0.08)));
    float lights = stringLight(uv, time);
    float snow = snowField(uv, time, 34.0, -0.22) + snowField(uv + vec2(0.11, 0.04), time, 52.0, -0.34) * 0.55;

    vec3 color = skyColor(uv);
    color += uRibbonC * ribbon * 0.34 * uIntensity;
    color += uRibbonA * lights * 0.72 * uIntensity;
    color += uRibbonB * pow(smoothstep(0.7, 1.0, mist), 3.0) * 0.18 * uIntensity;
    color += vec3(0.92, 0.98, 1.0) * snow * 0.24;
    color *= 0.56 + vignette * 0.68;

    gl_FragColor = vec4(clamp(color, 0.0, 1.0), 1.0);
  }
`;

export const christmasCozyBoardFragmentShader = /* glsl */ `
${commonFragmentPrelude}

  float rollingGarland(vec2 uv, float time) {
    float wave = sin(uv.x * 4.2 + time * 0.3) * 0.06 + fbm(vec2(uv.x * 2.8, uv.y * 2.0 - time * 0.08)) * 0.1;
    float band = 1.0 - smoothstep(0.0, 0.22, abs(uv.y - (0.44 + wave)));
    return band * smoothstep(0.08, 0.92, uv.y);
  }

  float emberSpecks(vec2 uv, float time) {
    vec2 cell = floor(vec2(uv.x * 42.0, uv.y * 24.0));
    float seed = hash(cell);
    float pulse = 0.5 + 0.5 * sin(time * 2.2 + seed * 6.28);
    return step(0.965, seed) * pulse;
  }

  void main() {
    vec2 uv = vUv;
    vec2 pixel = (gl_FragCoord.xy * 2.0 - uResolution.xy) / max(uResolution.x, uResolution.y);
    float time = uTime * uSpeed;
    float vignette = vignetteFor(pixel);
    float garland = rollingGarland(uv, time);
    float specks = emberSpecks(uv, time);
    float velvet = fbm(vec2(uv.x * 8.0 + time * 0.05, uv.y * 6.0 - time * 0.08));

    vec3 color = mix(uSkyLow * 0.5, uSkyMid * 0.7, smoothstep(0.0, 1.0, uv.y));
    color += uRibbonC * garland * 0.22 * uIntensity;
    color += uRibbonA * specks * 0.26 * uIntensity;
    color += uRibbonB * pow(smoothstep(0.72, 1.0, velvet), 3.0) * 0.1 * uIntensity;
    color *= 0.56 + vignette * 0.56;

    gl_FragColor = vec4(clamp(color, 0.0, 1.0), 1.0);
  }
`;

export const christmasCozyCardFragmentShader = /* glsl */ `
${commonFragmentPrelude}

  float warmSheen(vec2 uv, float time) {
    float stripe = uv.x * 1.2 + uv.y * 0.72 + sin(uv.y * 5.0 + time * 0.45) * 0.04;
    return 1.0 - smoothstep(0.0, 0.11, abs(fract(stripe - time * 0.08) - 0.5));
  }

  void main() {
    vec2 uv = vUv;
    vec2 pixel = (gl_FragCoord.xy * 2.0 - uResolution.xy) / max(uResolution.x, uResolution.y);
    float time = uTime * uSpeed;
    float vignette = vignetteFor(pixel);
    float sheen = warmSheen(uv, time);
    float twinkle = step(0.972, hash(floor(vec2(uv.x * 48.0, uv.y * 28.0)) + floor(time * 5.0)));
    float edge = 1.0 - smoothstep(0.2, 0.88, length(pixel * vec2(0.82, 1.08)));

    vec3 color = vec3(0.0);
    color += uRibbonA * sheen * 0.34 * uIntensity;
    color += uRibbonC * edge * 0.12;
    color += uRibbonB * twinkle * 0.16 * uIntensity;
    color *= 0.58 + vignette * 0.62;

    gl_FragColor = vec4(clamp(color, 0.0, 1.0), 1.0);
  }
`;

export const christmasSnowHomeFragmentShader = /* glsl */ `
${commonFragmentPrelude}

  float snowField(vec2 uv, float time, float scale, float speed) {
    vec2 grid = vec2(uv.x * scale + sin(uv.y * 5.0 + time) * 0.18, uv.y * scale * 0.68 + time * speed);
    vec2 cell = floor(grid);
    vec2 local = fract(grid) - 0.5;
    float seed = hash(cell);
    float flake = 1.0 - smoothstep(0.0, 0.07 + seed * 0.03, length(local));
    return flake * step(0.72, seed);
  }

  float frostRibbon(vec2 uv, float time) {
    float flow = fbm(vec2(uv.x * 2.0 + time * 0.06, uv.y * 2.4 - time * 0.08));
    float band = 1.0 - smoothstep(0.0, 0.28, abs(uv.y - (0.34 + flow * 0.22)));
    return band * smoothstep(1.02, 0.08, uv.y);
  }

  void main() {
    vec2 uv = vUv;
    vec2 pixel = (gl_FragCoord.xy * 2.0 - uResolution.xy) / max(uResolution.x, uResolution.y);
    float time = uTime * uSpeed;
    float vignette = vignetteFor(pixel);
    float frost = frostRibbon(uv, time);
    float snow = snowField(uv, time, 36.0, -0.28) + snowField(uv + vec2(0.09, 0.03), time, 58.0, -0.44) * 0.58;
    float holly = pow(smoothstep(0.74, 1.0, fbm(vec2(uv.x * 5.0 - time * 0.08, uv.y * 4.0 + time * 0.04))), 3.0);

    vec3 color = skyColor(uv);
    color += uRibbonC * frost * 0.24 * uIntensity;
    color += uRibbonA * holly * 0.08 * uIntensity;
    color += uRibbonB * holly * 0.055 * uIntensity;
    color += vec3(1.0) * snow * 0.2;
    color *= 0.92 + vignette * 0.1;

    gl_FragColor = vec4(clamp(color, 0.0, 1.0), 1.0);
  }
`;

export const christmasSnowBoardFragmentShader = /* glsl */ `
${commonFragmentPrelude}

  float softDrift(vec2 uv, float time) {
    float flow = fbm(vec2(uv.x * 1.65 + time * 0.038, uv.y * 2.05 - time * 0.052));
    float lowerBand = 1.0 - smoothstep(0.0, 0.34, abs(uv.y - (0.38 + flow * 0.18)));
    float upperBand = 1.0 - smoothstep(0.0, 0.42, abs(uv.y - (0.68 - flow * 0.16)));
    float mask = smoothstep(-0.08, 0.28, uv.y) * smoothstep(1.08, 0.36, uv.y);
    return (lowerBand * 0.74 + upperBand * 0.56) * mask;
  }

  float driftingFlake(vec2 uv, float time, float scale, float speed) {
    vec2 grid = vec2(uv.x * scale + sin(uv.y * 4.2 + time * 0.7) * 0.22, uv.y * scale * 0.72 + time * speed);
    vec2 cell = floor(grid);
    vec2 local = fract(grid) - 0.5;
    float seed = hash(cell);
    float flake = 1.0 - smoothstep(0.0, 0.055 + seed * 0.03, length(local));
    float sparkle = 0.68 + 0.32 * sin(time * 1.35 + seed * 6.2831);
    return flake * step(0.86, seed) * sparkle;
  }

  float sparseCrystal(vec2 uv, float time) {
    vec2 grid = vec2(uv.x * 10.0, uv.y * 6.5);
    vec2 cell = floor(grid);
    vec2 local = fract(grid) - 0.5;
    float seed = hash(cell);
    float twinkle = 0.56 + 0.44 * sin(time * 0.78 + seed * 6.2831);
    float diamond = 1.0 - smoothstep(0.0, 0.06, abs(local.x) + abs(local.y));
    return diamond * step(0.91, seed) * twinkle;
  }

  void main() {
    vec2 uv = vUv;
    vec2 pixel = (gl_FragCoord.xy * 2.0 - uResolution.xy) / max(uResolution.x, uResolution.y);
    float time = uTime * uSpeed;
    float vignette = vignetteFor(pixel);
    float drift = softDrift(uv, time);
    float snow = driftingFlake(uv, time, 21.0, -0.24) + driftingFlake(uv + vec2(0.11, 0.04), time, 34.0, -0.38) * 0.52;
    float crystal = sparseCrystal(uv, time);
    float shade = fbm(vec2(uv.x * 3.2 - time * 0.028, uv.y * 2.7 + time * 0.024));

    vec3 frostBase = mix(vec3(0.965, 0.988, 1.0), vec3(0.78, 0.89, 1.0), smoothstep(0.0, 1.0, uv.y));
    vec3 auroraBlue = mix(vec3(0.42, 0.78, 1.0), uRibbonC, 0.48);
    vec3 hollyAccent = mix(uRibbonA, uRibbonB, smoothstep(0.52, 0.95, shade));
    vec3 color = frostBase;
    color = mix(color, auroraBlue, drift * 0.22 * uIntensity);
    color += auroraBlue * snow * 0.14 * uIntensity;
    color += hollyAccent * pow(smoothstep(0.7, 1.0, shade), 3.0) * 0.07 * uIntensity;
    color += vec3(1.0) * crystal * 0.16;
    color *= 0.94 + vignette * 0.08;

    gl_FragColor = vec4(clamp(color, 0.0, 1.0), 1.0);
  }
`;

export const christmasSnowCardFragmentShader = /* glsl */ `
${commonFragmentPrelude}

  float settledFrost(vec2 uv, float time) {
    float diagonal = uv.x * 0.72 + uv.y * 0.92 + sin(uv.x * 3.0 + time * 0.12) * 0.018;
    float sweep = 1.0 - smoothstep(0.0, 0.16, abs(fract(diagonal - time * 0.018) - 0.5));
    float crystal = pow(smoothstep(0.78, 1.0, fbm(vec2(uv.x * 5.0, uv.y * 5.0 - time * 0.025))), 3.0);
    return sweep * 0.16 + crystal * 0.08;
  }

  void main() {
    vec2 uv = vUv;
    vec2 pixel = (gl_FragCoord.xy * 2.0 - uResolution.xy) / max(uResolution.x, uResolution.y);
    float time = uTime * uSpeed;
    float vignette = vignetteFor(pixel);
    float frost = settledFrost(uv, time);
    float edge = 1.0 - smoothstep(0.25, 0.92, length(pixel * vec2(0.82, 1.08)));

    vec3 color = vec3(0.0);
    color += uRibbonC * frost * 0.08 * uIntensity;
    color += vec3(0.62, 0.74, 0.88) * edge * 0.025;
    color += vec3(1.0) * frost * 0.025;
    color *= 0.9 + vignette * 0.12;

    gl_FragColor = vec4(clamp(color, 0.0, 1.0), 1.0);
  }
`;

export const christmasCandyHomeFragmentShader = /* glsl */ `
${commonFragmentPrelude}

  float peppermintStripe(vec2 uv, float time, float width) {
    float diagonal = uv.x * 1.65 + uv.y * 1.05 - time * 0.12;
    return smoothstep(width, 0.0, abs(fract(diagonal * 5.0) - 0.5));
  }

  float sugarSparkle(vec2 uv, float time) {
    vec2 cell = floor(vec2(uv.x * 46.0, uv.y * 30.0));
    float seed = hash(cell);
    float pulse = 0.5 + 0.5 * sin(time * 3.0 + seed * 6.28);
    return step(0.966, seed) * pulse;
  }

  void main() {
    vec2 uv = vUv;
    vec2 pixel = (gl_FragCoord.xy * 2.0 - uResolution.xy) / max(uResolution.x, uResolution.y);
    float time = uTime * uSpeed;
    float vignette = vignetteFor(pixel);
    float stripeA = peppermintStripe(uv, time, 0.18);
    float stripeB = peppermintStripe(uv + vec2(0.16, -0.08), -time * 0.62, 0.12);
    float mintFlow = fbm(vec2(uv.x * 2.8 - time * 0.08, uv.y * 2.4 + time * 0.06));
    float sparkle = sugarSparkle(uv, time);

    vec3 color = skyColor(uv);
    color += uRibbonA * stripeA * 0.14 * uIntensity;
    color += uRibbonB * stripeB * 0.12 * uIntensity;
    color += uRibbonC * pow(smoothstep(0.7, 1.0, mintFlow), 2.6) * 0.11 * uIntensity;
    color += vec3(1.0) * sparkle * 0.12;
    color *= 0.9 + vignette * 0.12;

    gl_FragColor = vec4(clamp(color, 0.0, 1.0), 1.0);
  }
`;

export const christmasCandyBoardFragmentShader = /* glsl */ `
${commonFragmentPrelude}

  float peppermintWide(vec2 uv, float time) {
    float diagonal = uv.x * 1.42 + uv.y * 0.94 - time * 0.11;
    float wide = 1.0 - smoothstep(0.0, 0.13, abs(fract(diagonal * 3.25) - 0.5));
    float narrow = 1.0 - smoothstep(0.0, 0.055, abs(fract((diagonal + 0.17) * 3.25) - 0.5));
    float mask = smoothstep(0.02, 0.88, uv.y) * smoothstep(1.08, 0.16, uv.y);
    return (wide * 0.82 + narrow * 0.42) * mask;
  }

  float mintRibbon(vec2 uv, float time) {
    float flow = fbm(vec2(uv.x * 3.8 - time * 0.08, uv.y * 3.2 + time * 0.06));
    float ribbon = 1.0 - smoothstep(0.0, 0.28, abs(uv.y - (0.42 + flow * 0.24)));
    return ribbon * smoothstep(-0.08, 0.3, uv.y) * smoothstep(1.08, 0.38, uv.y);
  }

  float sugarSparkleBoard(vec2 uv, float time) {
    vec2 grid = vec2(uv.x * 42.0, uv.y * 26.0);
    vec2 cell = floor(grid);
    float seed = hash(cell);
    float pulse = 0.54 + 0.46 * sin(time * 2.7 + seed * 6.2831);
    return step(0.965, seed) * pulse;
  }

  void main() {
    vec2 uv = vUv;
    vec2 pixel = (gl_FragCoord.xy * 2.0 - uResolution.xy) / max(uResolution.x, uResolution.y);
    float time = uTime * uSpeed;
    float vignette = vignetteFor(pixel);
    float stripe = peppermintWide(uv, time);
    float mint = mintRibbon(uv, time);
    float sparkle = sugarSparkleBoard(uv, time);
    float sugar = fbm(vec2(uv.x * 7.0 + time * 0.055, uv.y * 5.0 - time * 0.07));

    vec3 creamBase = mix(vec3(1.0, 0.985, 0.965), vec3(0.88, 1.0, 0.94), smoothstep(0.0, 1.0, uv.y));
    vec3 candyRed = mix(uRibbonA, vec3(1.0, 0.16, 0.18), 0.42);
    vec3 candyGreen = mix(uRibbonB, vec3(0.18, 0.82, 0.44), 0.38);
    vec3 sugarPink = mix(uRibbonC, vec3(1.0, 0.54, 0.66), 0.46);
    vec3 color = creamBase;
    color = mix(color, sugarPink, pow(smoothstep(0.68, 1.0, sugar), 2.4) * 0.16 * uIntensity);
    color = mix(color, candyGreen, mint * 0.2 * uIntensity);
    color = mix(color, candyRed, stripe * 0.3 * uIntensity);
    color += vec3(1.0) * sparkle * 0.12;
    color *= 0.93 + vignette * 0.09;

    gl_FragColor = vec4(clamp(color, 0.0, 1.0), 1.0);
  }
`;

export const christmasCandyCardFragmentShader = /* glsl */ `
${commonFragmentPrelude}

  float diagonalCandy(vec2 uv, float time) {
    float diagonal = uv.x * 1.45 + uv.y * 0.98 - time * 0.1;
    float stripe = 1.0 - smoothstep(0.0, 0.08, abs(fract(diagonal * 6.0) - 0.5));
    float secondary = 1.0 - smoothstep(0.0, 0.045, abs(fract((diagonal + 0.18) * 6.0) - 0.5));
    return stripe * 0.42 + secondary * 0.22;
  }

  void main() {
    vec2 uv = vUv;
    vec2 pixel = (gl_FragCoord.xy * 2.0 - uResolution.xy) / max(uResolution.x, uResolution.y);
    float time = uTime * uSpeed;
    float vignette = vignetteFor(pixel);
    float stripes = diagonalCandy(uv, time);
    float edge = 1.0 - smoothstep(0.2, 0.88, length(pixel * vec2(0.82, 1.08)));
    float sparkle = step(0.982, hash(floor(vec2(uv.x * 58.0, uv.y * 34.0)) + floor(time * 7.0)));

    vec3 color = vec3(0.0);
    color += uRibbonA * stripes * 0.18 * uIntensity;
    color += uRibbonB * edge * 0.08;
    color += uRibbonC * sparkle * 0.12 * uIntensity;
    color *= 0.84 + vignette * 0.18;

    gl_FragColor = vec4(clamp(color, 0.0, 1.0), 1.0);
  }
`;


export const bloodHomeFragmentShader = /* glsl */ `
${commonFragmentPrelude}

  float bloodDrip(vec2 uv, float seed, float time) {
    float columns = 13.0;
    float cell = floor(uv.x * columns + seed);
    float localX = fract(uv.x * columns + seed) - 0.5;
    float r = hash(vec2(cell, seed * 17.0));
    float width = 0.06 + r * 0.08;
    float dripLength = 0.18 + r * 0.5 + 0.08 * sin(time * (0.24 + r * 0.2) + r * 6.2831);
    float top = 1.02 - r * 0.18;
    float y = top - uv.y;
    float stem = smoothstep(width, 0.0, abs(localX + sin(uv.y * 6.0 + time * 0.18 + r) * 0.035));
    float vertical = smoothstep(0.0, 0.08, y) * (1.0 - smoothstep(dripLength, dripLength + 0.08, y));
    float beadY = top - dripLength;
    float bead = 1.0 - smoothstep(0.0, width * 2.3, length(vec2(localX * 1.5, (uv.y - beadY) * 3.2)));
    return stem * vertical * (0.34 + r * 0.48) + bead * 0.62;
  }

  void main() {
    vec2 uv = vUv;
    vec2 pixel = (gl_FragCoord.xy * 2.0 - uResolution.xy) / max(uResolution.x, uResolution.y);
    float time = uTime * uSpeed;
    float vignette = vignetteFor(pixel);
    float velvet = fbm(vec2(uv.x * 2.2 + time * 0.025, uv.y * 2.6 - time * 0.045));
    float curtain = 1.0 - smoothstep(0.0, 0.34, abs(uv.y - (0.32 + velvet * 0.22 + sin(uv.x * 3.2 + time * 0.18) * 0.08)));
    float dripA = bloodDrip(uv, 0.0, time);
    float dripB = bloodDrip(uv + vec2(0.037, -0.05), 4.2, time * 0.74) * 0.58;
    float lowerPool = smoothstep(0.45, 0.0, uv.y) * smoothstep(0.42, 1.0, fbm(vec2(uv.x * 4.4 - time * 0.04, uv.y * 3.2 + time * 0.03)));

    vec3 color = skyColor(uv);
    vec3 oxblood = mix(uRibbonB, uRibbonA, smoothstep(0.2, 0.95, velvet));
    color += oxblood * curtain * 0.32 * uIntensity;
    color += uRibbonA * (dripA + dripB) * 0.46 * uIntensity;
    color += uRibbonC * pow(dripA + dripB, 2.2) * 0.18 * uIntensity;
    color += uRibbonB * lowerPool * 0.28 * uIntensity;
    color *= 0.48 + vignette * 0.72;

    gl_FragColor = vec4(clamp(color, 0.0, 1.0), 1.0);
  }
`;

export const bloodBoardFragmentShader = /* glsl */ `
${commonFragmentPrelude}

  float pooledBlood(vec2 uv, float time) {
    vec2 poolUv = vec2(uv.x * 2.6 + time * 0.026, uv.y * 3.4 - time * 0.018);
    float slow = fbm(poolUv);
    float ring = sin((uv.x * 3.1 + slow * 1.4 - time * 0.08) * 6.2831) * 0.5 + 0.5;
    float lower = smoothstep(0.92, 0.1, uv.y);
    return smoothstep(0.36, 0.82, slow) * lower + pow(ring, 5.0) * 0.16 * lower;
  }

  float bloodDrip(vec2 uv, float seed, float time) {
    float col = floor(uv.x * 17.0 + seed);
    float r = hash(vec2(col, seed));
    float x = fract(uv.x * 17.0 + seed) - 0.5;
    float fall = fract(time * (0.035 + r * 0.035) + r);
    float headY = 1.12 - fall * 1.34;
    float width = 0.045 + r * 0.055;
    float trail = smoothstep(width, 0.0, abs(x)) * smoothstep(headY - 0.36, headY, uv.y) * smoothstep(headY + 0.04, headY, uv.y);
    float bead = 1.0 - smoothstep(0.0, width * 2.1, length(vec2(x * 1.5, (uv.y - headY) * 3.0)));
    return trail * 0.28 + bead * 0.52;
  }

  void main() {
    vec2 uv = vUv;
    vec2 pixel = (gl_FragCoord.xy * 2.0 - uResolution.xy) / max(uResolution.x, uResolution.y);
    float time = uTime * uSpeed;
    float vignette = vignetteFor(pixel);
    float pool = pooledBlood(uv, time);
    float drip = bloodDrip(uv, 1.8, time) + bloodDrip(uv + vec2(0.021, 0.0), 8.4, time * 0.82) * 0.64;
    float satin = fbm(vec2(uv.x * 7.0 + time * 0.04, uv.y * 4.6 - time * 0.035));

    vec3 color = mix(uSkyLow * 0.5, uSkyMid * 0.68, smoothstep(0.0, 1.0, uv.y));
    color += uRibbonB * pool * 0.3 * uIntensity;
    color += uRibbonA * (pool * 0.24 + drip * 0.34) * uIntensity;
    color += uRibbonC * pow(smoothstep(0.74, 1.0, satin), 3.0) * 0.11 * uIntensity;
    color *= 0.54 + vignette * 0.58;

    gl_FragColor = vec4(clamp(color, 0.0, 1.0), 1.0);
  }
`;

export const bloodCardFragmentShader = /* glsl */ `
${commonFragmentPrelude}

  float rivulet(vec2 uv, float seed, float time) {
    float columns = 18.0;
    float col = floor(uv.x * columns + seed);
    float r = hash(vec2(col, seed * 11.0));
    float localX = fract(uv.x * columns + seed) - 0.5;
    float wobble = sin(uv.y * (8.0 + r * 5.0) + time * (0.18 + r * 0.18) + r * 6.2831) * (0.025 + r * 0.025);
    float width = 0.035 + r * 0.04;
    float topMask = smoothstep(1.04, 0.42 + r * 0.34, uv.y);
    float lowerFade = smoothstep(-0.06, 0.28, uv.y);
    return smoothstep(width, 0.0, abs(localX + wobble)) * topMask * lowerFade * (0.45 + r * 0.5);
  }

  float droplet(vec2 uv, float seed, float time) {
    vec2 grid = vec2(uv.x * 18.0 + seed, uv.y * 9.0 - time * (0.16 + seed * 0.01));
    vec2 cell = floor(grid);
    vec2 local = fract(grid) - 0.5;
    float r = hash(cell + seed);
    local.x += (r - 0.5) * 0.38;
    float drop = 1.0 - smoothstep(0.0, 0.12 + r * 0.05, length(local * vec2(0.82, 1.35)));
    return drop * step(0.9, r) * smoothstep(0.96, 0.34, uv.y);
  }

  void main() {
    vec2 uv = vUv;
    vec2 pixel = (gl_FragCoord.xy * 2.0 - uResolution.xy) / max(uResolution.x, uResolution.y);
    float time = uTime * uSpeed;
    float vignette = vignetteFor(pixel);
    float flow = rivulet(uv, 0.0, time) + rivulet(uv + vec2(0.024, -0.03), 6.7, time * 0.78) * 0.58;
    float beads = droplet(uv, 2.4, time) + droplet(uv + vec2(0.07, 0.04), 9.1, time * 0.7) * 0.55;
    float gloss = 1.0 - smoothstep(0.0, 0.1, abs(fract(uv.x * 1.12 + uv.y * 0.72 - time * 0.045) - 0.5));
    float edge = 1.0 - smoothstep(0.2, 0.88, length(pixel * vec2(0.82, 1.08)));

    vec3 color = vec3(0.0);
    color += uRibbonA * flow * 0.42 * uIntensity;
    color += uRibbonB * (flow + beads) * 0.24 * uIntensity;
    color += uRibbonC * (beads * 0.16 + gloss * edge * 0.08) * uIntensity;
    color += uSkyMid * edge * 0.08;
    color *= 0.58 + vignette * 0.62;

    gl_FragColor = vec4(clamp(color, 0.0, 1.0), 1.0);
  }
`;

export const auroraFragmentShaders: Record<AuroraShaderVariant, string> = {
  aurora: auroraFragmentShader,
  'clean-fire': cleanFireFragmentShader,
  campfire: campfireFragmentShader,
  'dracula-home': draculaHomeFragmentShader,
  'dracula-board': draculaBoardFragmentShader,
  'dracula-card': draculaCardFragmentShader,
  'glitch-home': glitchHomeFragmentShader,
  'glitch-board': glitchBoardFragmentShader,
  'glitch-card': glitchCardFragmentShader,
  'christmas-cozy-home': christmasCozyHomeFragmentShader,
  'christmas-cozy-board': christmasCozyBoardFragmentShader,
  'christmas-cozy-card': christmasCozyCardFragmentShader,
  'christmas-snow-home': christmasSnowHomeFragmentShader,
  'christmas-snow-board': christmasSnowBoardFragmentShader,
  'christmas-snow-card': christmasSnowCardFragmentShader,
  'christmas-candy-home': christmasCandyHomeFragmentShader,
  'christmas-candy-board': christmasCandyBoardFragmentShader,
  'christmas-candy-card': christmasCandyCardFragmentShader,
  'blood-home': bloodHomeFragmentShader,
  'blood-board': bloodBoardFragmentShader,
  'blood-card': bloodCardFragmentShader,
};

export function auroraFragmentShaderFor(variant: AuroraShaderVariant): string {
  return auroraFragmentShaders[variant] ?? auroraFragmentShader;
}
