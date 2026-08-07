import { describe, expect, it } from 'vitest';

import { catalogItemToItem, type CatalogItemDto } from './dto';

const dto: CatalogItemDto = {
  id: '3f2504e0-4f89-41d3-9a0c-0305e82c3301',
  name: 'Кроссовки Limited',
  price: 24990,
  total_stock: 3,
  created_at: '2026-08-06T10:00:00Z',
};

describe('catalogItemToItem', () => {
  it('маппит snake_case DTO бэкенда в доменный Item', () => {
    const item = catalogItemToItem(dto);

    expect(item.id).toBe(dto.id);
    expect(item.title).toBe('Кроссовки Limited');
    expect(item.stock).toBe(3);
    expect(item.queueEnabled).toBe(true);
  });

  it('детерминированно подбирает emoji и accent по id', () => {
    expect(catalogItemToItem(dto)).toMatchObject({
      emoji: catalogItemToItem(dto).emoji,
      accent: catalogItemToItem(dto).accent,
    });
    expect(catalogItemToItem(dto).accent).toMatch(/^#[0-9A-F]{6}$/i);
  });
});
