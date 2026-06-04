# Landing Aurora Background

The landing hero uses a reusable Svelte 5 component at `web/src/lib/backgrounds/AuroraBackground.svelte`.
It renders one decorative Three.js WebGL canvas behind the hero content and keeps a CSS-only static fallback visible when WebGL is unavailable or before the renderer has initialized.

## Runtime behavior

- Three.js objects are created only inside synchronous `onMount`, so the component does not touch `window` or `document` during module evaluation.
- The renderer uses one `WebGLRenderer`, `Scene`, `OrthographicCamera`, fullscreen `PlaneGeometry(2, 2)`, and `ShaderMaterial`.
- `ResizeObserver` plus a window resize listener keep the renderer size and `uResolution` uniform synchronized with the hero.
- The animation loop skips rendering while `document.hidden` is true.
- The reduced-motion branch is still bypassed by the current force-motion override in the component,
  so shader animation remains enabled even when the OS prefers reduced motion.
- Teardown cancels animation frames, removes listeners, disconnects the observer, and disposes geometry, material, and renderer resources.

## Visual tuning knobs

The component exposes these props on the landing page:

- `intensity` controls aurora/fire brightness and presence. Current landing value: `1.0`.
- `speed` controls shader time flow. Current landing value: `1.0`.

Shader-level tuning lives in `web/src/lib/backgrounds/auroraShaders.ts`:

- Themes choose a procedural shader variant through `auroraPalettes` in `web/src/lib/theme.ts`:
  `aurora`, `clean-fire`, `campfire`, `dracula-home`, `dracula-board`, or `dracula-card`.
- Each variant is its own fragment shader source. `AuroraBackground` swaps the material's
  `fragmentShader` and sets `needsUpdate` when the active theme changes, so a compile/runtime issue
  in one optional variant cannot disable the other variants.
- `web/src/lib/backgrounds/auroraShaderCompile.test.ts` uses the lightweight
  `@webgpu/glslang` WASM compiler to compile the exported shader bodies in CI/test runs. The test
  adapts WebGL 1 declarations to GLSLang's SPIR-V profile only at test time; runtime shaders remain
  the WebGL sources consumed by Three.js.
- Dark themes use the `aurora` variant. `curtain(...)` controls the height, softness, strand density, and drift of each aurora band.
- Light uses the `campfire` variant. `campfireBody(...)`, `flameLick(...)`, `sparkField(...)`, and `smokeVeil(...)` combine uneven lower flame mass, torn rising licks, sparse sparks, and faint smoke/haze.
- Solarized Light uses the preserved `clean-fire` variant. `flameTongue(...)` shapes cleaner rising flame licks and composites warm orange/gold/coral color plus white-hot cores and small rising embers over the pale sky.
- Dracula uses three variants selected by the `surface` prop. `dracula-home` is the homepage hero's
  broad purple/pink/cyan atmosphere, `dracula-board` is slower low-contrast board fog behind the
  grid, and `dracula-card` is a tighter spectral sheen overlay for the Dracula board cards.
- `vignette`, `softMask`, and horizon glow keep the dark aurora restrained, while fire variants avoid dark multiply-style blending so they do not collapse into a shadow blob.

Do not add images, videos, GIFs, particles, or texture dependencies for this background; it is intentionally procedural.
