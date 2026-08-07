import styles from './ProgressBar.module.css';

interface ProgressBarProps {
  value: number;
  max?: number;
  tone?: 'queue' | 'granted' | 'blue';
  label?: string;
}

export function ProgressBar({
  value,
  max = 100,
  tone = 'queue',
  label,
}: ProgressBarProps) {
  const percent = Math.min(100, Math.max(0, (value / max) * 100));

  return (
    <div className={styles.wrapper}>
      {label && <span className={styles.label}>{label}</span>}
      <div
        className={styles.track}
        role="progressbar"
        aria-valuenow={Math.round(percent)}
        aria-valuemin={0}
        aria-valuemax={100}
      >
        <div
          className={`${styles.fill} ${styles[tone]}`}
          style={{ width: `${percent}%` }}
        />
      </div>
    </div>
  );
}
