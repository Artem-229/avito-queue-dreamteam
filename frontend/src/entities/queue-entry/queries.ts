import { useEffect, useState } from 'react';

import {
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';

import {
  checkout,
  fetchEntry,
  fetchEntryByItem,
  fetchEta,
  joinQueue,
  leaveQueue,
} from './api';
import { isTerminalStatus, type QueueEntry } from './model/types';

export const queueKeys = {
  entry: (entryId: string) => ['queue-entry', entryId] as const,
  byItem: (itemId: string) => ['queue-entry-by-item', itemId] as const,
};

const POLL_INTERVAL_MS = 1500;

export function useEntry(entryId: string) {
  return useQuery({
    queryKey: queueKeys.entry(entryId),
    queryFn: () => fetchEntry(entryId),
    refetchInterval: (query) => {
      const data = query.state.data;
      if (data && isTerminalStatus(data.status)) return false;
      return POLL_INTERVAL_MS;
    },
  });
}

export function useLiveEntry(entryId: string) {
  const queryClient = useQueryClient();
  const [transport, setTransport] = useState<'sse' | 'polling'>('sse');

  const query = useQuery({
    queryKey: queueKeys.entry(entryId),
    queryFn: () => fetchEntry(entryId),
    refetchInterval: (currentQuery) => {
      const data = currentQuery.state.data;
      if (data && isTerminalStatus(data.status)) return false;
      return transport === 'polling' ? POLL_INTERVAL_MS : false;
    },
  });

  useEffect(() => {
    if (transport !== 'sse') return;

    const source = new EventSource(`/api/queue/${entryId}/events`);

    source.onmessage = (event: MessageEvent<string>) => {
      const entry = JSON.parse(event.data) as QueueEntry;
      queryClient.setQueryData(queueKeys.entry(entryId), entry);
      if (isTerminalStatus(entry.status)) source.close();
    };

    source.onerror = () => {
      source.close();
      setTransport('polling');
    };

    return () => {
      source.close();
    };
  }, [entryId, transport, queryClient]);

  return query;
}

export function useEntryByItem(itemId: string) {
  return useQuery({
    queryKey: queueKeys.byItem(itemId),
    queryFn: () => fetchEntryByItem(itemId),
  });
}

export function useEta(entryId: string, enabled: boolean) {
  return useQuery({
    queryKey: ['queue-eta', entryId],
    queryFn: () => fetchEta(entryId),
    enabled,
    refetchInterval: enabled ? 4000 : false,
  });
}

export function useJoinQueue() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (itemId: string) => joinQueue(itemId),
    onSuccess: (entry) => {
      queryClient.setQueryData(queueKeys.entry(entry.entryId), entry);
      void queryClient.invalidateQueries({
        queryKey: queueKeys.byItem(entry.itemId),
      });
    },
  });
}

export function useLeaveQueue() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (entryId: string) => leaveQueue(entryId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['queue-entry'] });
    },
  });
}

export function useCheckout() {
  return useMutation({
    mutationFn: (entryId: string) => checkout(entryId),
  });
}
