import { apiRequest } from '@/shared/api';

import { catalogItemToItem, type CatalogItemDto } from './dto';
import { type Item } from './model/types';

export async function fetchItems(): Promise<Item[]> {
  const dtos = await apiRequest<CatalogItemDto[]>('/api/v1/catalog');
  return dtos.map(catalogItemToItem);
}

export async function fetchItem(id: string): Promise<Item> {
  const dto = await apiRequest<CatalogItemDto>(
    `/api/v1/catalog/${encodeURIComponent(id)}`,
  );
  return catalogItemToItem(dto);
}

export async function fetchSimilarItems(id: string): Promise<Item[]> {
  const dtos = await apiRequest<CatalogItemDto[]>(
    `/api/v1/catalog/${encodeURIComponent(id)}/similar`,
  );
  return dtos.slice(0, 4).map(catalogItemToItem);
}
