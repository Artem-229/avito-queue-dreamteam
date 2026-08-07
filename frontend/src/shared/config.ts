export type ApiMode = 'mock' | 'live';

export const API_MODE: ApiMode =
  import.meta.env.VITE_API_MODE === 'live' ? 'live' : 'mock';

export const isLiveApi = API_MODE === 'live';
