# Landing Aurora Background

The landing hero uses a reusable Svelte 5 component at `web/src/lib/backgrounds/AuroraBackground.svelte`.
It renders one decorative Three.js WebGL canvas behind the hero content and keeps a CSS-only static fallback visible when WebGL is unavailable or before the renderer has initialized.

## Runtime behavior

- Three.js objects are created only inside synchronous `onMount`, so the component does not touch `window` or `document` during module evaluation.
- The renderer uses one `WebGLRenderer`, `Scene`, `OrthographicCamera`, fullscreen `PlaneGeometry(2, 2)`, and `ShaderMaterial`.
- `ResizeObserver` plus a window resize listener keep the renderer size and `uResolution` uniform synchronized with the hero.
- The animation loop skips rendering while `document.hidden` is true.
- Users with `prefers-reduced-motion: reduce` get a static frame and no animation loop.
- Teardown cancels animation frames, removes listeners, disconnects the observer, and disposes geometry, material, and renderer resources.

## Visual tuning knobs

The component exposes these props on the landing page:

- `intensity` controls aurora brightness and presence. Current landing value: `0.74`.
- `speed` controls shader time flow. Current landing value: `0.16` for slow, premium motion.

Shader-level tuning lives in `web/src/lib/backgrounds/auroraShaders.ts`:

- `emerald`, `cyan`, and `violet` set the restrained aurora palette.
- `curtain(...)` controls the height, softness, strand density, and drift of each aurora band.
- `skyTop`, `skyMid`, and `skyLow` define the deep navy night-sky gradient.
- `vignette`, `softMask`, and horizon glow keep the effect behind the form and away from loud neon/demo visuals.

Do not add images, videos, GIFs, particles, or texture dependencies for this background; it is intentionally procedural.
