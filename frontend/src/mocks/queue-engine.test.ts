import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { ApiFault, QueueEngine } from './queue-engine';

const SNEAKERS = 'sneakers-limited';
const CONSOLE = 'console-retro';
const COMMON = 'sneakers-common';

describe('QueueEngine', () => {
  let engine: QueueEngine;

  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(0);
    engine = new QueueEngine();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('ставит пользователя за ботов и считает позицию', () => {
    const entry = engine.join('user-1', SNEAKERS);

    expect(entry.status).toBe('QUEUED');
    expect(entry.totalAhead).toBe(3);
    expect(entry.etaSeconds).toBeGreaterThan(0);
  });

  it('выдаёт право на покупку после обслуживания ботов', () => {
    const joined = engine.join('user-1', SNEAKERS);

    vi.advanceTimersByTime(3 * 4000 + 100);
    const entry = engine.getEntry(joined.entryId);

    expect(entry.status).toBe('GRANTED');
    expect(entry.expiresAt).toBeDefined();
    expect(entry.totalAhead).toBe(0);
  });

  it('позволяет купить только при активном праве и списывает сток', () => {
    const joined = engine.join('user-1', SNEAKERS);
    vi.advanceTimersByTime(3 * 4000 + 100);
    engine.getEntry(joined.entryId);

    const before = engine.getItem(SNEAKERS).stock;
    const order = engine.checkout('user-1', joined.entryId);
    const after = engine.getItem(SNEAKERS).stock;

    expect(order.itemId).toBe(SNEAKERS);
    expect(after).toBe(before - 1);
    expect(engine.getEntry(joined.entryId).status).toBe('PURCHASED');
  });

  it('не даёт купить без активного права', () => {
    const joined = engine.join('user-1', SNEAKERS);

    expect(() => engine.checkout('user-1', joined.entryId)).toThrowError(
      ApiFault,
    );
  });

  it('переводит в SOLD_OUT, когда боты разобрали весь сток', () => {
    const joined = engine.join('user-1', CONSOLE);

    vi.advanceTimersByTime(3 * 3500 + 100);
    const entry = engine.getEntry(joined.entryId);

    expect(entry.status).toBe('SOLD_OUT');
    expect(() => engine.checkout('user-1', joined.entryId)).toThrowError(
      ApiFault,
    );
  });

  it('не пускает в очередь для товара без очереди', () => {
    expect(() => engine.join('user-1', COMMON)).toThrowError(ApiFault);
  });

  it('идемпотентно возвращает существующую запись при повторном join', () => {
    const first = engine.join('user-1', SNEAKERS);
    const second = engine.join('user-1', SNEAKERS);

    expect(second.entryId).toBe(first.entryId);
  });

  it('рекомендует товары той же категории первыми', () => {
    const similar = engine.getSimilarItems(SNEAKERS);

    expect(similar.length).toBeGreaterThan(0);
    expect(similar[0]?.category).toBe('Обувь');
    expect(similar.every((item) => item.id !== SNEAKERS)).toBe(true);
  });
});
