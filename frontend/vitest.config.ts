import { defineConfig, mergeConfig } from 'vitest/config';

import viteConfig from './vite.config';

const baseConfig = viteConfig({ command: 'serve', mode: 'test' });

export default mergeConfig(
  baseConfig,
  defineConfig({
    test: {
      globals: true,
      environment: 'jsdom',
      setupFiles: './src/test/setup.ts',
      css: true,
    },
  }),
);
