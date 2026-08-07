import { useNavigate } from 'react-router-dom';

import { ItemCard, useSimilarItems } from '@/entities/item';
import { type QueueEntry } from '@/entities/queue-entry';
import { Button, Skeleton } from '@/shared/ui';

import { QueueScreen } from '../QueueScreen';
import styles from '../QueuePage.module.css';

interface QueueSoldOutProps {
  entry: QueueEntry;
}

export function QueueSoldOut({ entry }: QueueSoldOutProps) {
  const navigate = useNavigate();
  const { data: similar, isLoading } = useSimilarItems(entry.itemId);

  return (
    <QueueScreen
      itemId={entry.itemId}
      tone="amber"
      statusLabel="Товар закончился"
      title="Товар разобрали"
      description="Пока вы стояли в очереди, товар успели купить. Чтобы не терять время, посмотрите похожие лоты — возможно, у другого продавца есть то, что нужно."
      wide
      footer={
        <Button
          variant="secondary"
          onClick={() => {
            navigate('/');
          }}
        >
          В каталог
        </Button>
      }
    >
      <div className={styles.similarGrid}>
        {isLoading &&
          Array.from({ length: 3 }).map((_, index) => (
            <Skeleton key={index} height={240} radius="var(--radius-lg)" />
          ))}
        {similar?.map((item) => (
          <ItemCard key={item.id} item={item} />
        ))}
      </div>
    </QueueScreen>
  );
}
