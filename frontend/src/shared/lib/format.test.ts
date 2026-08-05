import { describe, expect, it } from 'vitest';

import { formatPrice } from './formatPrice';
import { formatEta, formatMmSs } from './formatTime';

describe('formatMmSs', () => {
  it('форматирует секунды в MM:SS', () => {
    expect(formatMmSs(0)).toBe('00:00');
    expect(formatMmSs(65)).toBe('01:05');
    expect(formatMmSs(-5)).toBe('00:00');
  });
});

describe('formatEta', () => {
  it('секунды до минуты', () => {
    expect(formatEta(45)).toBe('≈ 45 сек');
  });

  it('минуты для больших значений', () => {
    expect(formatEta(150)).toBe('≈ 3 мин');
  });
});

describe('formatPrice', () => {
  it('добавляет символ рубля', () => {
    const result = formatPrice(24990);
    expect(result).toContain('₽');
    expect(result).toMatch(/24/);
  });
});
