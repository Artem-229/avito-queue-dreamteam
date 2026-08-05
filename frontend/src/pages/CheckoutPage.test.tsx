import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';

import { CheckoutPage } from './CheckoutPage';

const mocks = vi.hoisted<{ entryStatus: string }>(() => ({
  entryStatus: 'QUEUED',
}));

vi.mock('@/entities/queue-entry', () => ({
  useEntryByItem: () => ({
    data: { status: mocks.entryStatus, entryId: 'e1', itemId: 'i1' },
    isLoading: false,
  }),
  useCheckout: () => ({ mutate: vi.fn(), isPending: false }),
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
    mocks.entryStatus = 'QUEUED';
    renderAt();

    expect(await screen.findByText('ITEM PAGE')).toBeInTheDocument();
  });
});
