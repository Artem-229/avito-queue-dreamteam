import { useQuery } from '@tanstack/react-query';

import { fetchMetrics } from './api';

export function useMetrics() {
  return useQuery({
    queryKey: ['queue-metrics'],
    queryFn: fetchMetrics,
    refetchInterval: 2500,
  });
}
