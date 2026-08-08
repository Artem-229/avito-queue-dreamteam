import { apiRequest } from '@/shared/api';
import { isLiveApi } from '@/shared/config';

import { catalogItemToItem, type CatalogItemDto } from './dto';
import { type Item } from './model/types';

export async function fetchItems(): Promise<Item[]> {
  if (isLiveApi) {
    const dtos = await apiRequest<CatalogItemDto[]>('/api/v1/catalog');
    return dtos.map(catalogItemToItem);
  }
  return apiRequest<Item[]>('/api/items');
}

export async function fetchItem(id: string): Promise<Item> {
  if (isLiveApi) {
    const dto = await apiRequest<CatalogItemDto>(`/api/v1/catalog/${id}`);
    return catalogItemToItem(dto);
  }
  return apiRequest<Item>(`/api/items/${id}`);
}

export async function fetchSimilarItems(id: string): Promise<Item[]> {
  if (isLiveApi) {
    const dtos = await apiRequest<CatalogItemDto[]>(
      `/api/v1/catalog/${id}/similar`,
    );
    return dtos.slice(0, 4).map(catalogItemToItem);
  }
  return apiRequest<Item[]>(`/api/items/${id}/similar`);
}
