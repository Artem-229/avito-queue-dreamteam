import { StrictMode } from 'react';

import { createRoot } from 'react-dom/client';

import { App } from '@/app/App';
import '@/app/styles/index.css';

async function enableMocking(): Promise<void> {
  if (!import.meta.env.DEV) return;
  if (import.meta.env.VITE_ENABLE_MOCKS === 'false') return;

  try {
    const { worker } = await import('@/mocks/browser');
    await worker.start({ onUnhandledRequest: 'bypass' });
  } catch (error) {
    console.error('MSW worker failed to start', error);
  }
}

function render(): void {
  const rootElement = document.getElementById('root');

  if (!rootElement) {
    throw new Error('Root element #root not found');
  }

  createRoot(rootElement).render(
    <StrictMode>
      <App />
    </StrictMode>,
  );
}

void enableMocking().then(render);
