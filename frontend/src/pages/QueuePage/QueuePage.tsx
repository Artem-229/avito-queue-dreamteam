import { useParams } from 'react-router-dom';

import { useLiveEntry } from '@/entities/queue-entry';
import { Skeleton } from '@/shared/ui';

import { QueueExpired } from './views/QueueExpired';
import { QueueGranted } from './views/QueueGranted';
import { QueuePurchased } from './views/QueuePurchased';
import { QueueSoldOut } from './views/QueueSoldOut';
import { QueueWaiting } from './views/QueueWaiting';
import styles from './QueuePage.module.css';

export function QueuePage() {
  const entryId = useParams().entryId ?? '';
  const { data: entry, isLoading, isError } = useLiveEntry(entryId);

  if (isError) {
    return <p className={styles.error}>Запись очереди не найдена.</p>;
  }

  if (isLoading || !entry) {
    return (
      <div className={styles.center}>
        <Skeleton height={200} width={360} radius="var(--radius-lg)" />
      </div>
    );
  }

  switch (entry.status) {
    case 'QUEUED':
      return <QueueWaiting entry={entry} />;
    case 'GRANTED':
      return <QueueGranted entry={entry} />;
    case 'EXPIRED':
      return <QueueExpired entry={entry} />;
    case 'SOLD_OUT':
      return <QueueSoldOut entry={entry} />;
    case 'PURCHASED':
      return <QueuePurchased entry={entry} />;
    case 'LEFT':
      return <QueueExpired entry={entry} left />;
    default:
      return null;
  }
}
