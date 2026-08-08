import { describe, expect, it } from 'vitest';

import { runSimulation } from './simulator';

describe('runSimulation', () => {
  it('никогда не продаёт больше единиц, чем есть в стоке', () => {
    const result = runSimulation({
      buyers: 50,
      stock: 3,
      buyProbability: 1,
      seed: 42,
    });

    expect(result.unitsSold).toBeLessThanOrEqual(3);
    expect(result.distribution.purchased).toBe(result.unitsSold);
  });

  it('исходы покрывают всех покупателей', () => {
    const result = runSimulation({
      buyers: 20,
      stock: 5,
      buyProbability: 0.6,
      seed: 7,
    });

    const { purchased, expired, soldOut } = result.distribution;
    expect(purchased + expired + soldOut).toBe(20);
    expect(result.timeline).toHaveLength(20);
  });

  it('детерминирован при одном seed', () => {
    const a = runSimulation({ buyers: 30, stock: 4, buyProbability: 0.5, seed: 1 });
    const b = runSimulation({ buyers: 30, stock: 4, buyProbability: 0.5, seed: 1 });

    expect(a.timeline).toEqual(b.timeline);
  });

  it('при вероятности покупки 100% раскупает весь сток', () => {
    const result = runSimulation({
      buyers: 10,
      stock: 4,
      buyProbability: 1,
      seed: 3,
    });

    expect(result.unitsSold).toBe(4);
    expect(result.distribution.soldOut).toBe(6);
  });
});
