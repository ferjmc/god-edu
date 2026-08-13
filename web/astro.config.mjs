// @ts-check
import { defineConfig, passthroughImageService } from 'astro/config';
import tailwindcss from '@tailwindcss/vite';

import react from '@astrojs/react';

// https://astro.build/config
export default defineConfig({
  vite: {
      plugins: [tailwindcss()],
	},

  integrations: [react()],

  // El servicio de imágenes por default (basado en Sharp, un binario nativo)
  // genera un build local perfecto pero falla silenciosamente en el entorno
  // de build de Cloudflare Pages: el <Image> termina apuntando a /_image
  // (el endpoint de optimización EN RUNTIME), que no existe en un sitio 100%
  // estático — de ahí las imágenes rotas en producción aunque local andaba
  // bien. passthroughImageService copia el archivo tal cual, sin
  // procesarlo: nada de Sharp, mismo resultado en cualquier entorno.
  image: {
    service: passthroughImageService(),
  },
});