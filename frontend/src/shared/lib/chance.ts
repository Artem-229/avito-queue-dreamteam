export interface PurchaseChance {
  percent: number;
  basis: string;
}

export function chanceTone(percent: number): 'green' | 'amber' | 'red' {
  if (percent >= 66) return 'green';
  if (percent >= 40) return 'amber';
  return 'red';
}

export function chanceLabel(chance: PurchaseChance): string {
  const base = `Шанс купить ≈ ${String(chance.percent)}%`;
  return chance.basis === 'default' ? `${base} · оценка` : base;
}
