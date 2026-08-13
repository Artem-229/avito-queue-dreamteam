import { describe, expect, it } from 'vitest';

import { catalogItemToItem, type CatalogItemDto } from './dto';

const dto: CatalogItemDto = {
  id: '3f2504e0-4f89-41d3-9a0c-0305e82c3301',
  name: 'Кроссовки Limited',
  price_kopecks: 2499000,
  hold_ttl_seconds: 90,
  total_stock: 3,
  granted_count: 1,
  used_count: 1,
  category: 'Обувь',
  seller_name: 'SneakerHub',
  created_at: '2026-08-06T10:00:00Z',
};

describe('catalogItemToItem', () => {
  it('маппит snake_case DTO бэкенда в доменный Item', () => {
    const item = catalogItemToItem(dto);

    expect(item.id).toBe(dto.id);
    expect(item.title).toBe('Кроссовки Limited');
    expect(item.totalStock).toBe(3);
    expect(item.sellerName).toBe('SneakerHub');
    expect(item.category).toBe('Обувь');
    expect(item.queueEnabled).toBe(true);
  });

  it('вычитает из тиража выкупленное и удержанные права', () => {
    expect(catalogItemToItem(dto).available).toBe(1);
    expect(catalogItemToItem({ ...dto, granted_count: 2 }).available).toBe(0);
  });

  it('считает распроданным только выкупленный тираж, а не удержанный', () => {
    expect(
      catalogItemToItem({ ...dto, granted_count: 2, used_count: 1 }).soldOut,
    ).toBe(false);
    expect(
      catalogItemToItem({ ...dto, granted_count: 0, used_count: 3 }).soldOut,
    ).toBe(true);
  });

  it('переносит шанс покупки с карточки товара, а в списке его нет', () => {
    expect(catalogItemToItem(dto).chance).toBeNull();
    expect(
      catalogItemToItem({ ...dto, chance_if_join: { percent: 72, basis: 'item' } })
        .chance,
    ).toEqual({ percent: 72, basis: 'item' });
  });

  it('без счётчиков считает весь тираж свободным', () => {
    const item = catalogItemToItem({
      ...dto,
      granted_count: undefined,
      used_count: undefined,
    });

    expect(item.available).toBe(3);
    expect(item.soldOut).toBe(false);
  });

  it('детерминированно подбирает emoji и accent по id', () => {
    expect(catalogItemToItem(dto)).toMatchObject({
      emoji: catalogItemToItem(dto).emoji,
      accent: catalogItemToItem(dto).accent,
    });
    expect(catalogItemToItem(dto).accent).toMatch(/^#[0-9A-F]{6}$/i);
  });
});
