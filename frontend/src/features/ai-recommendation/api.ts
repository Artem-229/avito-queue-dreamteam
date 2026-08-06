import { apiRequest } from '@/shared/api';

export interface RecommendationCandidate {
  title: string;
  price: number;
}

export interface RecommendationPayload {
  soldOutTitle: string;
  candidates: RecommendationCandidate[];
}

export interface RecommendationResult {
  text: string | null;
}

export function fetchRecommendation(
  payload: RecommendationPayload,
): Promise<RecommendationResult> {
  return apiRequest<RecommendationResult>('/ai/recommendation', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}
