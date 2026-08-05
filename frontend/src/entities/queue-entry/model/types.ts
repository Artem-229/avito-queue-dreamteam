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

export function isActiveRight(entry: QueueEntry): boolean {
  return entry.status === 'GRANTED';
}
