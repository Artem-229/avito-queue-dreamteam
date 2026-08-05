export type QueueStatus =
  | 'QUEUED'
  | 'GRANTED'
  | 'EXPIRED'
  | 'PURCHASED'
  | 'SOLD_OUT'
  | 'LEFT';

export interface QueueEntry {
  entryId: string;
  itemId: string;
  userId: string;
  status: QueueStatus;
  position: number;
  totalAhead: number;
  grantedAt?: string;
  expiresAt?: string;
  etaSeconds?: number;
}

export interface Order {
  orderId: string;
  itemId: string;
  createdAt: string;
}

export interface EtaResult {
  seconds: number;
  confidence: 'low' | 'medium' | 'high';
}

const TERMINAL_STATUSES: readonly QueueStatus[] = [
  'EXPIRED',
  'PURCHASED',
  'SOLD_OUT',
  'LEFT',
];

export function isActiveRight(entry: QueueEntry): boolean {
  return entry.status === 'GRANTED';
}

export function isTerminalStatus(status: QueueStatus): boolean {
  return TERMINAL_STATUSES.includes(status);
}
