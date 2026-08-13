import { describe, expect, it } from 'vitest';

import { chanceLabel, chanceTone } from './chance';

describe('chanceTone', () => {
  it('высокий шанс — зелёный, средний — жёлтый, низкий — красный', () => {
    expect(chanceTone(80)).toBe('green');
    expect(chanceTone(50)).toBe('amber');
    expect(chanceTone(10)).toBe('red');
  });
});

describe('chanceLabel', () => {
  it('помечает дефолтную оценку, измеренную — нет', () => {
    expect(chanceLabel({ percent: 72, basis: 'item' })).toBe(
      'Шанс купить ≈ 72%',
    );
    expect(chanceLabel({ percent: 50, basis: 'default' })).toBe(
      'Шанс купить ≈ 50% · оценка',
    );
  });
});
