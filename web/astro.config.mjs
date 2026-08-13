// @ts-check
import { defineConfig, passthroughImageService } from 'astro/config';
import tailwindcss from '@tailwindcss/vite';

import react from '@astrojs/react';

import cloudflare from '@astrojs/cloudflare';

// https://astro.build/config
export default defineConfig({
  vite: {
      plugins: [tailwindcss()],
    },

  integrations: [react()],

  adapter: cloudflare({
    imageService: 'cloudflare-binding'
  }),
});