import { useMutation } from '@tanstack/react-query';

import { runSimulationRequest } from './api';
import { type SimulationParams } from './types';

export function useSimulation() {
  return useMutation({
    mutationFn: (params: SimulationParams) => runSimulationRequest(params),
  });
}
