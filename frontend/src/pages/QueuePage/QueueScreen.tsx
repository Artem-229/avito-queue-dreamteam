import { type ReactNode } from 'react';

import { useItem } from '@/entities/item';
import { StatusPill } from '@/shared/ui';

import styles from './QueuePage.module.css';

type Tone = 'purple' | 'green' | 'red' | 'blue' | 'amber';

interface QueueScreenProps {
  itemId: string;
  tone: Tone;
  statusLabel: string;
  pulse?: boolean;
  title: string;
  description: string;
  children?: ReactNode;
  footer?: ReactNode;
  wide?: boolean;
}

export function QueueScreen({
  itemId,
  tone,
  statusLabel,
  pulse = false,
  title,
  description,
  children,
  footer,
  wide = false,
}: QueueScreenProps) {
  const { data: item } = useItem(itemId);

  return (
    <div className={styles.center}>
      <div className={`${styles.panel} ${wide ? styles.panelWide : ''}`}>
        {item && (
          <div className={styles.itemLine}>
            <span
              className={styles.itemEmoji}
              style={{ background: `${item.accent}22` }}
            >
              {item.emoji}
            </span>
            <span className={styles.itemTitle}>{item.title}</span>
          </div>
        )}

        <StatusPill tone={tone} label={statusLabel} pulse={pulse} />

        <h1 className={styles.title}>{title}</h1>
        <p className={styles.description}>{description}</p>

        {children}

        {footer && <div className={styles.footer}>{footer}</div>}
      </div>
    </div>
  );
}
