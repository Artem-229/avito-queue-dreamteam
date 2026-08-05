import { useNavigate, useParams } from 'react-router-dom';

import { useItem } from '@/entities/item';
import {
  isTerminalStatus,
  useEntryByItem,
  useJoinQueue,
} from '@/entities/queue-entry';
import { formatPrice } from '@/shared/lib/formatPrice';
import { Badge, Button, Card, Skeleton } from '@/shared/ui';

import styles from './ItemPage.module.css';

export function ItemPage() {
  const itemId = useParams().itemId ?? '';
  const navigate = useNavigate();

  const { data: item, isLoading, isError } = useItem(itemId);
  const { data: existingEntry } = useEntryByItem(itemId);
  const join = useJoinQueue();

  if (isError) {
    return <p className={styles.error}>Товар не найден.</p>;
  }

  if (isLoading || !item) {
    return (
      <div className={styles.layout}>
        <Skeleton height={360} radius="var(--radius-lg)" />
        <div className={styles.stack}>
          <Skeleton height={28} width="70%" />
          <Skeleton height={40} width="40%" />
          <Skeleton height={44} />
        </div>
      </div>
    );
  }

  const activeEntry =
    existingEntry && !isTerminalStatus(existingEntry.status)
      ? existingEntry
      : null;

  const handleJoin = () => {
    join.mutate(itemId, {
      onSuccess: (entry) => {
        navigate(`/queue/${entry.entryId}`);
      },
    });
  };

  return (
    <div className={styles.layout}>
      <div
        className={styles.cover}
        style={{ background: `${item.accent}22` }}
      >
        <span className={styles.emoji}>{item.emoji}</span>
      </div>

      <Card padding="lg" className={styles.panel}>
        <span className={styles.seller}>{item.sellerName}</span>
        <h1 className={styles.title}>{item.title}</h1>

        <div className={styles.badges}>
          {item.queueEnabled ? (
            <Badge tone="purple">Ограниченный выпуск</Badge>
          ) : (
            <Badge tone="green">Обычная покупка</Badge>
          )}
          <Badge>
            {item.stock > 0 ? `В наличии: ${String(item.stock)}` : 'Распродано'}
          </Badge>
        </div>

        <span className={styles.price}>{formatPrice(item.price)}</span>

        {item.queueEnabled ? (
          <div className={styles.actions}>
            <p className={styles.hint}>
              Этот товар продаётся через очередь. Вы встанете в очередь и
              получите ограниченное по времени право на покупку.
            </p>
            {activeEntry ? (
              <Button
                size="lg"
                fullWidth
                onClick={() => {
                  navigate(`/queue/${activeEntry.entryId}`);
                }}
              >
                Вернуться в очередь
              </Button>
            ) : (
              <Button
                size="lg"
                fullWidth
                loading={join.isPending}
                disabled={item.stock <= 0}
                onClick={handleJoin}
              >
                {item.stock > 0 ? 'Встать в очередь' : 'Товара нет в наличии'}
              </Button>
            )}
          </div>
        ) : (
          <Button size="lg" fullWidth variant="secondary" disabled>
            Оформление обычной покупки — вне рамок MVP
          </Button>
        )}
      </Card>
    </div>
  );
}
