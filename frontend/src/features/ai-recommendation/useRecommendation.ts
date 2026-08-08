import { useQuery } from '@tanstack/react-query';

import {
  fetchRecommendation,
  type RecommendationCandidate,
} from './api';

export function useRecommendation(
  soldOutTitle: string | undefined,
  candidates: RecommendationCandidate[],
) {
  const enabled = Boolean(soldOutTitle) && candidates.length > 0;

  return useQuery({
    queryKey: [
      'ai-recommendation',
      soldOutTitle,
      candidates.map((item) => item.title).join('|'),
    ],
    queryFn: () =>
      fetchRecommendation({
        soldOutTitle: soldOutTitle ?? '',
        candidates,
      }),
    enabled,
    retry: false,
    staleTime: Infinity,
  });
}
