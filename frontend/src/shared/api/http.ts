import { ensureSession, refreshSession } from '@/shared/lib/session';

import { ApiError } from './ApiError';

interface ErrorBody {
  code?: string;
  message?: string;
}

/**
 * Токен идёт заголовком Authorization, а не кукой: куку бэкенд ставит
 * httpOnly, JS её не читает, и переключатель демо-аккаунтов не смог бы
 * держать в одном браузере больше одной сессии. Бэкенд проверяет заголовок
 * первым — явно выбранный аккаунт побеждает фоновую куку.
 */
function send(
  path: string,
  options: RequestInit,
  token: string,
): Promise<Response> {
  return fetch(path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
      ...options.headers,
    },
  });
}

export async function apiRequest<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const account = await ensureSession();
  let response = await send(path, options, account.token);

  // 401 означает истёкшую или недействительную сессию, и единственное
  // осмысленное лечение — перевыпустить её и повторить запрос ровно один раз.
  // Тело запроса всегда строка, поэтому повтор безопасен: перечитывать
  // израсходованный поток не приходится.
  if (response.status === 401) {
    const refreshed = await refreshSession(account.token);
    response = await send(path, options, refreshed.token);
  }

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
