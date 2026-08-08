import { getUserId } from '@/shared/lib/session';

import { ApiError } from './ApiError';

interface ErrorBody {
  code?: string;
  message?: string;
}

export async function apiRequest<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const response = await fetch(path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      'X-User-Id': getUserId(),
      ...options.headers,
    },
  });

  if (!response.ok) {
    let code = 'UNKNOWN';
    let message = response.statusText;

    try {
      const body = (await response.json()) as ErrorBody;
      code = body.code ?? code;
      message = body.message ?? message;
    } catch (error) {
      void error;
    }

    throw new ApiError(message, code, response.status);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}
