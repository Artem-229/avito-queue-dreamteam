import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';

import { CheckoutPage } from './CheckoutPage';

const mocks = vi.hoisted<{ entryStatus: string }>(() => ({
  entryStatus: 'waiting',
}));

vi.mock('@/entities/queue-entry', () => ({
  useQueueStatus: () => ({
    data: {
      itemId: 'i1',
      status: mocks.entryStatus,
      message: '',
      nextStep: { kind: 'pay', label: '' },
      position: 0,
      initialPosition: null,
      progressPercent: null,
      chance: null,
      queueSize: 0,
      expiresAt: null,
      etaSeconds: null,
      serverTime: '2026-08-10T00:00:00Z',
      alternatives: [],
    },
    isLoading: false,
  }),
  usePurchase: () => ({ mutate: vi.fn(), isPending: false }),
}));

vi.mock('@/entities/item', () => ({
  useItem: () => ({ data: undefined }),
}));

function renderAt() {
  return render(
    <MemoryRouter initialEntries={['/checkout/i1']}>
      <Routes>
        <Route path="/checkout/:itemId" element={<CheckoutPage />} />
        <Route path="/item/:itemId" element={<div>ITEM PAGE</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('CheckoutPage guard', () => {
  it('редиректит на страницу товара без активного права на покупку', async () => {
    mocks.entryStatus = 'waiting';
    renderAt();

    expect(await screen.findByText('ITEM PAGE')).toBeInTheDocument();
  });

  it('пускает на оформление, когда право выдано', () => {
    mocks.entryStatus = 'granted';
    renderAt();

    expect(screen.queryByText('ITEM PAGE')).not.toBeInTheDocument();
  });
});
