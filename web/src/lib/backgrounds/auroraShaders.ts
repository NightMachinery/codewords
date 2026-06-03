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
  uniform float uShaderVariant; // 0 = aurora, 1 = clean fire, 2 = campfire

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

  vec3 renderCleanFire(vec2 uv, vec3 sky, float time, float vignette) {
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
    float spark = 1.0 - smoothstep(0.0, 0.09, length(local + vec2(seed - 0.5, hash(cell + 13.7) - 0.5) * 0.28));
    float active = step(0.965, seed);
    float heightFade = smoothstep(0.22, 0.78, uv.y) * smoothstep(1.0, 0.46, uv.y);

    return spark * active * heightFade;
  }

  float smokeVeil(vec2 uv, float time) {
    float smoke = fbm(vec2(uv.x * 2.4 + time * 0.08, uv.y * 2.7 - time * 0.32));
    float plume = smoothstep(0.34, 0.82, uv.y) * smoothstep(1.02, 0.46, uv.y);
    float centerFade = smoothstep(0.62, 0.12, abs(uv.x - 0.54));

    return smoothstep(0.44, 0.82, smoke) * plume * centerFade;
  }

  vec3 renderCampfire(vec2 uv, vec3 sky, float time, float vignette) {
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
    float emberBase = bodies * smoothstep(0.42, 0.02, uv.y);
    float whiteHotCore = pow(clamp(flame * smoothstep(0.34, 0.0, abs(uv.x - 0.52)) * smoothstep(0.76, 0.1, uv.y), 0.0, 1.0), 2.35);
    float sparks = sparkField(uv, time) * uIntensity;
    float smoke = smokeVeil(uv, time);

    vec3 emberRed = vec3(0.6, 0.09, 0.035);
    vec3 deepOrange = vec3(0.95, 0.28, 0.04);
    vec3 golden = vec3(1.0, 0.68, 0.16);
    vec3 hotCore = vec3(1.0, 0.94, 0.68);
    vec3 smokeTint = vec3(0.48, 0.42, 0.36);

    vec3 flameRamp = mix(deepOrange, golden, smoothstep(0.18, 0.82, heat + uv.y * 0.36));
    flameRamp = mix(emberRed, flameRamp, smoothstep(0.04, 0.38, uv.y));

    vec3 fireColor = sky;
    fireColor = mix(fireColor, flameRamp, flame * (0.76 + heat * 0.18) * uIntensity);
    fireColor += emberRed * emberBase * 0.46 * uIntensity;
    fireColor += hotCore * whiteHotCore * 0.72 * uIntensity;
    fireColor += golden * sparks * 0.55;
    fireColor = mix(fireColor, smokeTint, smoke * 0.1);
    fireColor *= 0.95 + vignette * 0.07;

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

    if (uShaderVariant < 0.5) {
      color = darkColor;
    } else if (uShaderVariant < 1.5) {
      color = renderCleanFire(uv, color, time, vignette);
    } else {
      color = renderCampfire(uv, color, time, vignette);
    }

    gl_FragColor = vec4(color, 1.0);
  }
`;
