import { apiRequest } from '@/shared/api';

import { type QueueMetrics } from './types';

/** Ответ бэкенда: GET /api/v1/stats. */
interface QueueStatsDto {
  total_waiting: number;
  total_granted: number;
  purchased: number;
  expired: number;
  /** null, пока ни одно право не завершилось — делить не на что. */
  conversion: number | null;
  avg_wait_seconds: number | null;
  items: ItemStatsDto[];
}

interface ItemStatsDto {
  item_id: string;
  title: string;
  waiting: number;
  granted: number;
  stock_left: number;
}

export async function fetchMetrics(): Promise<QueueMetrics> {
  const dto = await apiRequest<QueueStatsDto>('/api/v1/stats');

  return {
    totalWaiting: dto.total_waiting,
    totalGranted: dto.total_granted,
    purchased: dto.purchased,
    expired: dto.expired,
    conversion: dto.conversion,
    avgWaitSeconds: dto.avg_wait_seconds,
    items: dto.items.map((item) => ({
      itemId: item.item_id,
      title: item.title,
      waiting: item.waiting,
      granted: item.granted,
      stockLeft: item.stock_left,
    })),
  };
}
