import { apiRequest } from '@/shared/api';

import { type QueueMetrics } from './types';

export function fetchMetrics(): Promise<QueueMetrics> {
  return apiRequest<QueueMetrics>('/api/metrics');
}
