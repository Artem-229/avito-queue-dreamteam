import {
  type SimBuyer,
  type SimulationParams,
  type SimulationResult,
} from '@/features/simulator/types';

function mulberry32(seed: number): () => number {
  let a = seed >>> 0;
  return () => {
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

export function runSimulation(params: SimulationParams): SimulationResult {
  const buyers = clamp(Math.floor(params.buyers), 1, 200);
  const stock = clamp(Math.floor(params.stock), 0, buyers);
  const buyProbability = clamp(params.buyProbability, 0, 1);
  const random = mulberry32(params.seed ?? 1);

  const timeline: SimBuyer[] = [];
  let remaining = stock;

  for (let index = 0; index < buyers; index += 1) {
    const order = index + 1;

    if (remaining <= 0) {
      timeline.push({ order, outcome: 'SOLD_OUT' });
      continue;
    }

    if (random() < buyProbability) {
      timeline.push({ order, outcome: 'PURCHASED' });
      remaining -= 1;
    } else {
      timeline.push({ order, outcome: 'EXPIRED' });
    }
  }

  const purchased = timeline.filter((b) => b.outcome === 'PURCHASED').length;
  const expired = timeline.filter((b) => b.outcome === 'EXPIRED').length;
  const soldOut = timeline.filter((b) => b.outcome === 'SOLD_OUT').length;

  return {
    buyers,
    stock,
    unitsSold: purchased,
    distribution: { purchased, expired, soldOut },
    timeline,
  };
}
