import { useNavigate } from 'react-router-dom';

import { ItemCard, useItem, useSimilarItems } from '@/entities/item';
import { type QueueEntry } from '@/entities/queue-entry';
import { useRecommendation } from '@/features/ai-recommendation';
import { Button, Skeleton } from '@/shared/ui';

import { QueueScreen } from '../QueueScreen';
import styles from '../QueuePage.module.css';

interface QueueSoldOutProps {
  entry: QueueEntry;
}

export function QueueSoldOut({ entry }: QueueSoldOutProps) {
  const navigate = useNavigate();
  const { data: item } = useItem(entry.itemId);
  const { data: similar, isLoading } = useSimilarItems(entry.itemId);

  const recommendation = useRecommendation(
    item?.title,
    (similar ?? []).map((candidate) => ({
      title: candidate.title,
      price: candidate.priceKopecks / 100,
    })),
  );

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
      {recommendation.data?.text && (
        <div className={styles.aiBlurb}>
          <span className={styles.aiBadge}>ИИ · GigaChat</span>
          <p className={styles.aiText}>{recommendation.data.text}</p>
        </div>
      )}

      <div className={styles.similarGrid}>
        {isLoading &&
          Array.from({ length: 3 }).map((_, index) => (
            <Skeleton key={index} height={240} radius="var(--radius-lg)" />
          ))}
        {similar?.map((candidate) => (
          <ItemCard key={candidate.id} item={candidate} />
        ))}
      </div>
    </QueueScreen>
  );
}
