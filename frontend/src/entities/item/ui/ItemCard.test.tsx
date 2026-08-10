import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';

import { type Item } from '../model/types';
import { ItemCard } from './ItemCard';

const item: Item = {
  id: 'sneakers-limited',
  title: 'Кроссовки Limited',
  priceKopecks: 2499000,
  holdTtlSeconds: 90,
  category: 'Обувь',
  sellerName: 'SneakerHub',
  totalStock: 5,
  available: 5,
  soldOut: false,
  queueEnabled: true,
  emoji: '👟',
  accent: '#965EEB',
};

function renderCard(override: Partial<Item> = {}) {
  return render(
    <MemoryRouter>
      <ItemCard item={{ ...item, ...override }} />
    </MemoryRouter>,
  );
}

describe('ItemCard', () => {
  it('показывает название, цену и ссылку на товар', () => {
    renderCard();

    expect(screen.getByText('Кроссовки Limited')).toBeInTheDocument();
    expect(screen.getByText(/24/)).toBeInTheDocument();
    expect(screen.getByRole('link')).toHaveAttribute(
      'href',
      '/item/sneakers-limited',
    );
  });

  it('показывает бейдж очереди для дефицитного товара', () => {
    renderCard({ queueEnabled: true });
    expect(screen.getByText('Ограниченный выпуск')).toBeInTheDocument();
  });

  it('скрывает бейдж очереди для обычного товара', () => {
    renderCard({ queueEnabled: false });
    expect(screen.queryByText('Ограниченный выпуск')).not.toBeInTheDocument();
  });

  it('различает выкупленный товар и временно занятые слоты', () => {
    const { unmount } = renderCard({ available: 0, soldOut: true });
    expect(screen.getByText('распродано')).toBeInTheDocument();
    unmount();

    renderCard({ available: 0, soldOut: false });
    expect(screen.getByText('сейчас разобран')).toBeInTheDocument();
  });
});
