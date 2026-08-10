import {
  type NextStep,
  type QueueEntry,
  type QueueStatus,
} from './model/types';

/** Ответ бэкенда: GET/POST/DELETE /api/v1/catalog/:id/queue[/me]. */
export interface QueueStatusDto {
  status: QueueStatus;
  message: string;
  position?: number;
  queue_size: number;
  expires_at: string | null;
  eta_seconds: number | null;
  server_time: string;
  next_step: NextStep;
  alternatives: string[] | null;
}

/**
 * Единственное место, где snake_case бэкенда превращается в camelCase модели.
 * itemId приходит параметром: в конверте его нет, он всегда известен вызывающему.
 */
export function toQueueEntry(dto: QueueStatusDto, itemId: string): QueueEntry {
  return {
    itemId,
    status: dto.status,
    message: dto.message,
    nextStep: dto.next_step,
    position: dto.position ?? 0,
    queueSize: dto.queue_size,
    expiresAt: dto.expires_at,
    etaSeconds: dto.eta_seconds,
    serverTime: dto.server_time,
    alternatives: dto.alternatives ?? [],
  };
}
