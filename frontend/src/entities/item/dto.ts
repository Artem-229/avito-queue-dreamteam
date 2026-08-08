import { type Item } from './model/types';

export interface CatalogItemDto {
  id: string;
  name: string;
  price: number;
  total_stock: number;
  category?: string;
  seller_name?: string;
  created_at?: string;
}

const ACCENTS = ['#965EEB', '#00AAFF', '#04E061', '#FFB020', '#FF6163'];
const EMOJIS = ['📦', '🛍️', '🎧', '👟', '🕹️', '💿', '📱', '⌚', '🎮', '💎'];

function pickBy<T>(list: T[], key: string): T {
  let hash = 0;
  for (const char of key) {
    hash = (hash * 31 + char.charCodeAt(0)) >>> 0;
  }
  return list[hash % list.length];
}

export function catalogItemToItem(dto: CatalogItemDto): Item {
  return {
    id: dto.id,
    title: dto.name,
    price: dto.price,
    stock: dto.total_stock,
    queueEnabled: true,
    sellerName: dto.seller_name ?? 'Продавец на Авито',
    category: dto.category ?? '',
    emoji: pickBy(EMOJIS, dto.id),
    accent: pickBy(ACCENTS, dto.id),
  };
}
