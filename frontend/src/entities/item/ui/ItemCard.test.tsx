import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';

import { type Item } from '../model/types';
import { ItemCard } from './ItemCard';

const item: Item = {
  id: 'sneakers-limited',
  title: 'Кроссовки Limited',
  price: 24990,
  category: 'Обувь',
  sellerName: 'SneakerHub',
  stock: 5,
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
});
