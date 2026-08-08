export type SimOutcome = 'PURCHASED' | 'EXPIRED' | 'SOLD_OUT';

export interface SimBuyer {
  order: number;
  outcome: SimOutcome;
}

export interface SimulationParams {
  buyers: number;
  stock: number;
  buyProbability: number;
  seed?: number;
}

export interface SimulationResult {
  buyers: number;
  stock: number;
  unitsSold: number;
  distribution: {
    purchased: number;
    expired: number;
    soldOut: number;
  };
  timeline: SimBuyer[];
}
