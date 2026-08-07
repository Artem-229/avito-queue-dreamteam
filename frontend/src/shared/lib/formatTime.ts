export function formatMmSs(totalSeconds: number): string {
  const safe = Math.max(0, Math.floor(totalSeconds));
  const minutes = Math.floor(safe / 60);
  const seconds = safe % 60;

  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
}

export function formatEta(totalSeconds: number): string {
  const safe = Math.max(0, Math.round(totalSeconds));

  if (safe < 60) {
    return `≈ ${String(safe)} сек`;
  }

  const minutes = Math.round(safe / 60);
  return `≈ ${String(minutes)} мин`;
}
