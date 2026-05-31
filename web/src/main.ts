import './style.css';

import { mount } from 'svelte';
import App from './App.svelte';
import { applyTheme, prefersDarkScheme, readThemePreferences, resolveTheme } from './lib/theme';

// Apply the saved theme before mounting so the correct theme paints on the first frame.
applyTheme(resolveTheme(readThemePreferences(localStorage), prefersDarkScheme()));

const target = document.getElementById('app');

if (!target) {
  throw new Error('Could not find app mount point');
}

const app = mount(App, { target });

export default app;
