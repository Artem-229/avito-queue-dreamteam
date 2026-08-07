import { useEffect, useState } from 'react';

export interface Countdown {
  secondsLeft: number;
  isExpired: boolean;
}

export function getSecondsLeft(expiresAt: number, now: number): number {
  return Math.max(0, Math.ceil((expiresAt - now) / 1000));
}

export function useCountdown(
  expiresAt: number,
  onExpire?: () => void,
): Countdown {
  const [secondsLeft, setSecondsLeft] = useState(() =>
    getSecondsLeft(expiresAt, Date.now()),
  );

  useEffect(() => {
    setSecondsLeft(getSecondsLeft(expiresAt, Date.now()));

    const id = window.setInterval(() => {
      const next = getSecondsLeft(expiresAt, Date.now());
      setSecondsLeft(next);

      if (next <= 0) {
        window.clearInterval(id);
        onExpire?.();
      }
    }, 250);

    return () => {
      window.clearInterval(id);
    };
  }, [expiresAt, onExpire]);

  return { secondsLeft, isExpired: secondsLeft <= 0 };
}
