import { useEffect, useState } from 'react';

export interface Countdown {
  secondsLeft: number;
  isExpired: boolean;
}

export function getSecondsLeft(expiresAt: number, now: number): number {
  return Math.max(0, Math.ceil((expiresAt - now) / 1000));
}

/**
 * Расхождение часов браузера и сервера. Бэкенд отдаёт server_time из Postgres
 * тем же ответом, что и expires_at, поэтому сдвиг считается один раз по разнице
 * и дальше применяется к локальным часам: иначе у пользователя со сбитым
 * временем таймер права врёт на величину этого сдвига.
 */
export function getClockSkewMs(serverTime: string | undefined): number {
  if (!serverTime) return 0;

  const parsed = Date.parse(serverTime);
  if (Number.isNaN(parsed)) return 0;

  return parsed - Date.now();
}

export function useCountdown(
  expiresAt: number,
  onExpire?: () => void,
  skewMs = 0,
): Countdown {
  const [secondsLeft, setSecondsLeft] = useState(() =>
    getSecondsLeft(expiresAt, Date.now() + skewMs),
  );

  useEffect(() => {
    setSecondsLeft(getSecondsLeft(expiresAt, Date.now() + skewMs));

    const id = window.setInterval(() => {
      const next = getSecondsLeft(expiresAt, Date.now() + skewMs);
      setSecondsLeft(next);

      if (next <= 0) {
        window.clearInterval(id);
        onExpire?.();
      }
    }, 250);

    return () => {
      window.clearInterval(id);
    };
  }, [expiresAt, onExpire, skewMs]);

  return { secondsLeft, isExpired: secondsLeft <= 0 };
}
