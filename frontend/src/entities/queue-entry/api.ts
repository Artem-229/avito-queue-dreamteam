import { apiRequest } from '@/shared/api';

import { toQueueEntry, type QueueStatusDto } from './dto';
import { type QueueEntry } from './model/types';

/**
 * Все операции очереди адресуются товаром, а не записью очереди: агрегат на
 * бэкенде — товар, запись очереди отдельной сущностью не адресуется.
 */
const base = (itemId: string) =>
  `/api/v1/catalog/${encodeURIComponent(itemId)}`;

export async function joinQueue(itemId: string): Promise<QueueEntry> {
  const dto = await apiRequest<QueueStatusDto>(`${base(itemId)}/queue`, {
    method: 'POST',
  });
  return toQueueEntry(dto, itemId);
}

/** Статус и позиция. Внутри бэкенд сам продвигает очередь (EnsureAdvanced). */
export async function fetchStatus(itemId: string): Promise<QueueEntry> {
  const dto = await apiRequest<QueueStatusDto>(`${base(itemId)}/queue/me`);
  return toQueueEntry(dto, itemId);
}

export async function leaveQueue(itemId: string): Promise<QueueEntry> {
  const dto = await apiRequest<QueueStatusDto>(`${base(itemId)}/queue/me`, {
    method: 'DELETE',
  });
  return toQueueEntry(dto, itemId);
}

/** Использовать право на покупку — единственный путь покупки. */
export async function purchase(itemId: string): Promise<void> {
  await apiRequest<void>(`${base(itemId)}/purchase`, { method: 'POST' });
}
