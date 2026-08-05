import { type HTMLAttributes } from 'react';

import styles from './Badge.module.css';

type BadgeTone = 'neutral' | 'blue' | 'purple' | 'green' | 'red' | 'amber';

interface BadgeProps extends HTMLAttributes<HTMLSpanElement> {
  tone?: BadgeTone;
}

export function Badge({
  tone = 'neutral',
  className,
  children,
  ...rest
}: BadgeProps) {
  const classes = [styles.badge, styles[tone], className ?? '']
    .filter(Boolean)
    .join(' ');

  return (
    <span className={classes} {...rest}>
      {children}
    </span>
  );
}
