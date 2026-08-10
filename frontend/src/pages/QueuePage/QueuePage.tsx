import { useParams } from 'react-router-dom';

import { useQueueStatus } from '@/entities/queue-entry';
import { Skeleton } from '@/shared/ui';

import { useTurnNotification } from './useTurnNotification';
import { QueueExpired } from './views/QueueExpired';
import { QueueGranted } from './views/QueueGranted';
import { QueueNotInQueue } from './views/QueueNotInQueue';
import { QueuePurchased } from './views/QueuePurchased';
import { QueueSoldOut } from './views/QueueSoldOut';
import { QueueWaiting } from './views/QueueWaiting';
import styles from './QueuePage.module.css';

export function QueuePage() {
  const itemId = useParams().itemId ?? '';
  const { data: entry, isLoading, isError } = useQueueStatus(itemId);

  useTurnNotification(entry?.status);

  if (isError) {
    return <p className={styles.error}>Не удалось загрузить статус очереди.</p>;
  }

  if (isLoading || !entry) {
    return (
      <div className={styles.center}>
        <Skeleton height={200} width={360} radius="var(--radius-lg)" />
      </div>
    );
  }

  switch (entry.status) {
    case 'not_in_queue':
      return <QueueNotInQueue entry={entry} />;
    case 'waiting':
      return <QueueWaiting entry={entry} />;
    case 'granted':
      return <QueueGranted entry={entry} />;
    case 'expired':
      return <QueueExpired entry={entry} />;
    case 'sold_out':
      return <QueueSoldOut entry={entry} />;
    case 'purchased':
      return <QueuePurchased entry={entry} />;
    case 'cancelled':
      return <QueueExpired entry={entry} left />;
    default:
      return null;
  }
}
