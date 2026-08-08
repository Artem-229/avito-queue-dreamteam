import { apiRequest } from '@/shared/api';

import { type SimulationParams, type SimulationResult } from './types';

export function runSimulationRequest(
  params: SimulationParams,
): Promise<SimulationResult> {
  return apiRequest<SimulationResult>('/api/sim/run', {
    method: 'POST',
    body: JSON.stringify(params),
  });
}
