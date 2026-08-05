import { apiRequest } from '@/shared/api';

import { type Item } from './model/types';

export function fetchItems(): Promise<Item[]> {
  return apiRequest<Item[]>('/api/items');
}

export function fetchItem(id: string): Promise<Item> {
  return apiRequest<Item>(`/api/items/${id}`);
}

export function fetchSimilarItems(id: string): Promise<Item[]> {
  return apiRequest<Item[]>(`/api/items/${id}/similar`);
}
