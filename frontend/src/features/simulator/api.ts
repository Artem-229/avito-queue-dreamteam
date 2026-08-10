import { apiRequest } from '@/shared/api';

import { type SimulationParams, type SimulationResult } from './types';

interface SimulationResultDto {
  item_id: string;
  requested: number;
  joined: number;
  failed: number;
}

export async function runSimulationRequest(
  params: SimulationParams,
): Promise<SimulationResult> {
  const dto = await apiRequest<SimulationResultDto>('/api/v1/demo/simulate', {
    method: 'POST',
    body: JSON.stringify({ item_id: params.itemId, count: params.count }),
  });

  return {
    itemId: dto.item_id,
    requested: dto.requested,
    joined: dto.joined,
    failed: dto.failed,
  };
}
