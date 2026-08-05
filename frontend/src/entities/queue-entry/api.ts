import { ApiError, apiRequest } from '@/shared/api';

import {
  type EtaResult,
  type Order,
  type QueueEntry,
} from './model/types';

export function joinQueue(itemId: string): Promise<QueueEntry> {
  return apiRequest<QueueEntry>('/api/queue/join', {
    method: 'POST',
    body: JSON.stringify({ itemId }),
  });
}

export async function fetchEntryByItem(
  itemId: string,
): Promise<QueueEntry | null> {
  try {
    return await apiRequest<QueueEntry>(
      `/api/queue/entry?itemId=${encodeURIComponent(itemId)}`,
    );
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) {
      return null;
    }
    throw error;
  }
}

export function fetchEntry(entryId: string): Promise<QueueEntry> {
  return apiRequest<QueueEntry>(`/api/queue/${entryId}`);
}

export function leaveQueue(entryId: string): Promise<void> {
  return apiRequest<void>(`/api/queue/${entryId}/leave`, { method: 'POST' });
}

export function checkout(entryId: string): Promise<Order> {
  return apiRequest<Order>('/api/checkout', {
    method: 'POST',
    body: JSON.stringify({ entryId }),
  });
}

export function fetchEta(entryId: string): Promise<EtaResult> {
  return apiRequest<EtaResult>(`/api/queue/${entryId}/eta`);
}

export function subscribeToEntry(
  entryId: string,
  onMessage: (entry: QueueEntry) => void,
  onError?: () => void,
): () => void {
  const source = new EventSource(`/api/queue/${entryId}/events`);

  source.onmessage = (event: MessageEvent<string>) => {
    onMessage(JSON.parse(event.data) as QueueEntry);
  };

  source.onerror = () => {
    onError?.();
  };

  return () => {
    source.close();
  };
}
