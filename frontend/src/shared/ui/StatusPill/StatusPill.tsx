import styles from './StatusPill.module.css';

type StatusTone = 'blue' | 'purple' | 'green' | 'red' | 'amber' | 'neutral';

interface StatusPillProps {
  tone?: StatusTone;
  label: string;
  pulse?: boolean;
}

export function StatusPill({
  tone = 'neutral',
  label,
  pulse = false,
}: StatusPillProps) {
  const classes = [styles.pill, styles[tone]].join(' ');

  return (
    <span className={classes}>
      <span
        className={`${styles.dot} ${pulse ? styles.pulse : ''}`}
        aria-hidden="true"
      />
      {label}
    </span>
  );
}
