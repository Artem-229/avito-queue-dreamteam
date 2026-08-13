import { type Item } from './model/types';

export interface CatalogItemDto {
  id: string;
  name: string;
  price_kopecks: number;
  total_stock: number;
  /** Сколько прав выдано и держится прямо сейчас. */
  granted_count?: number;
  /** Сколько единиц уже выкуплено — обратно не возвращается. */
  used_count?: number;
  hold_ttl_seconds?: number;
  category?: string;
  seller_name?: string;
  created_at?: string;
  /** Есть только на карточке одного товара, в списке отсутствует. */
  chance_if_join?: { percent: number; basis: string } | null;
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
  const granted = dto.granted_count ?? 0;
  const used = dto.used_count ?? 0;

  return {
    id: dto.id,
    title: dto.name,
    priceKopecks: dto.price_kopecks,
    totalStock: dto.total_stock,
    // Свободные слоты — это весь тираж минус выкупленное и минус права,
    // которые кто-то держит прямо сейчас: показывать total_stock как
    // «в наличии» значит обещать товар, который уже кому-то обещан.
    available: Math.max(dto.total_stock - granted - used, 0),
    // Распродан — именно выкуплен: удержанное право может сгореть и слот
    // вернётся, поэтому granted в этот признак не входит.
    soldOut: used >= dto.total_stock,
    holdTtlSeconds: dto.hold_ttl_seconds ?? 120,
    queueEnabled: true,
    chance: dto.chance_if_join ?? null,
    sellerName: dto.seller_name ?? 'Продавец на Авито',
    category: dto.category ?? '',
    emoji: pickBy(EMOJIS, dto.id),
    accent: pickBy(ACCENTS, dto.id),
  };
}
