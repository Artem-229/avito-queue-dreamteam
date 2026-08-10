import { formatMmSs } from '@/shared/lib/formatTime';
import { useCountdown } from '@/shared/lib/useCountdown';

import styles from './CountdownTimer.module.css';

interface CountdownTimerProps {
  expiresAt: number;
  totalSeconds: number;
  onExpire?: () => void;
  /** Сдвиг часов браузера относительно сервера, см. getClockSkewMs. */
  skewMs?: number;
}

const RADIUS = 52;
const CIRCUMFERENCE = 2 * Math.PI * RADIUS;

function toneFor(secondsLeft: number): 'green' | 'amber' | 'red' {
  if (secondsLeft <= 10) return 'red';
  if (secondsLeft <= 30) return 'amber';
  return 'green';
}

export function CountdownTimer({
  expiresAt,
  totalSeconds,
  onExpire,
  skewMs = 0,
}: CountdownTimerProps) {
  const { secondsLeft } = useCountdown(expiresAt, onExpire, skewMs);
  const tone = toneFor(secondsLeft);
  const ratio = totalSeconds > 0 ? secondsLeft / totalSeconds : 0;
  const dashOffset = CIRCUMFERENCE * (1 - Math.min(1, Math.max(0, ratio)));

  return (
    <div className={`${styles.timer} ${styles[tone]}`}>
      <svg className={styles.svg} viewBox="0 0 120 120" aria-hidden="true">
        <circle className={styles.trackRing} cx="60" cy="60" r={RADIUS} />
        <circle
          className={styles.progressRing}
          cx="60"
          cy="60"
          r={RADIUS}
          strokeDasharray={CIRCUMFERENCE}
          strokeDashoffset={dashOffset}
        />
      </svg>
      <span className={styles.value} role="timer" aria-live="polite">
        {formatMmSs(secondsLeft)}
      </span>
    </div>
  );
}
