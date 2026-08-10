import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { fetchStatus, joinQueue, leaveQueue, purchase } from './api';
import { isTerminalStatus, type QueueEntry } from './model/types';

export const queueKeys = {
  status: (itemId: string) => ['queue-status', itemId] as const,
};

/**
 * Поллинг раз в секунду — осознанный выбор вместо SSE: этот же запрос служит
 * триггером ленивого продвижения очереди на бэкенде, а дешёвая проверка
 * NeedsReconcile не даёт частому опросу превратиться в горячую точку.
 */
const POLL_INTERVAL_MS = 1000;

export function useQueueStatus(itemId: string, enabled = true) {
  return useQuery({
    queryKey: queueKeys.status(itemId),
    queryFn: () => fetchStatus(itemId),
    enabled: enabled && itemId !== '',
    refetchInterval: (query) => {
      const data = query.state.data;
      if (data && isTerminalStatus(data.status)) return false;
      return POLL_INTERVAL_MS;
    },
  });
}

function useQueueMutation(
  mutationFn: (itemId: string) => Promise<QueueEntry>,
) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn,
    // Ответ — тот же конверт статуса, поэтому кладём его в кеш напрямую:
    // второй запрос сразу после действия не нужен.
    onSuccess: (entry) => {
      queryClient.setQueryData(queueKeys.status(entry.itemId), entry);
    },
  });
}

export function useJoinQueue() {
  return useQueueMutation(joinQueue);
}

export function useLeaveQueue() {
  return useQueueMutation(leaveQueue);
}

export function usePurchase() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (itemId: string) => purchase(itemId),
    onSuccess: (_result, itemId) => {
      void queryClient.invalidateQueries({
        queryKey: queueKeys.status(itemId),
      });
    },
  });
}
