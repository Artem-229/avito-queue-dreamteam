import { useQuery } from '@tanstack/react-query';

import { fetchItem, fetchItems, fetchSimilarItems } from './api';

export const itemKeys = {
  all: ['items'] as const,
  one: (id: string) => ['item', id] as const,
  similar: (id: string) => ['item-similar', id] as const,
};

export function useItems() {
  return useQuery({
    queryKey: itemKeys.all,
    queryFn: fetchItems,
  });
}

export function useItem(id: string) {
  return useQuery({
    queryKey: itemKeys.one(id),
    queryFn: () => fetchItem(id),
  });
}

export function useSimilarItems(id: string, enabled = true) {
  return useQuery({
    queryKey: itemKeys.similar(id),
    queryFn: () => fetchSimilarItems(id),
    enabled,
  });
}
