import { type CSSProperties } from 'react';

import styles from './Skeleton.module.css';

interface SkeletonProps {
  width?: string | number;
  height?: string | number;
  radius?: string | number;
  className?: string;
}

export function Skeleton({
  width = '100%',
  height = 16,
  radius = 'var(--radius-sm)',
  className,
}: SkeletonProps) {
  const style: CSSProperties = {
    width,
    height,
    borderRadius: radius,
  };

  return (
    <span
      className={`${styles.skeleton} ${className ?? ''}`.trim()}
      style={style}
      aria-hidden="true"
    />
  );
}
