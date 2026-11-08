import { mount } from 'svelte';
import ResponsiveFixture from './ResponsiveFixture.svelte';
import '../src/style.css';

mount(ResponsiveFixture, { target: document.getElementById('app')! });
