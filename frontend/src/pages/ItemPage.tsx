import { useNavigate, useParams } from 'react-router-dom';

import { useItem } from '@/entities/item';
import { isInQueue, useJoinQueue, useQueueStatus } from '@/entities/queue-entry';
import { formatPrice } from '@/shared/lib/formatPrice';
import { Badge, Button, Card, Skeleton } from '@/shared/ui';

import styles from './ItemPage.module.css';

export function ItemPage() {
  const itemId = useParams().itemId ?? '';
  const navigate = useNavigate();

  const { data: item, isLoading, isError } = useItem(itemId);
  const { data: entry } = useQueueStatus(itemId);
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

  const alreadyInQueue = entry ? isInQueue(entry.status) : false;

  const handleJoin = () => {
    join.mutate(itemId, {
      onSuccess: () => {
        navigate(`/queue/${itemId}`);
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
            {item.soldOut
              ? 'Распродано'
              : `Свободно: ${String(item.available)} из ${String(item.totalStock)}`}
          </Badge>
        </div>

        <span className={styles.price}>{formatPrice(item.priceKopecks)}</span>

        {item.queueEnabled ? (
          <div className={styles.actions}>
            <p className={styles.hint}>
              Этот товар продаётся через очередь. Вы встанете в очередь и
              получите ограниченное по времени право на покупку.
            </p>
            {alreadyInQueue ? (
              <Button
                size="lg"
                fullWidth
                onClick={() => {
                  navigate(`/queue/${itemId}`);
                }}
              >
                Вернуться в очередь
              </Button>
            ) : (
              <Button
                size="lg"
                fullWidth
                loading={join.isPending}
                // Отсутствие свободных слотов очередь не отменяет — она ровно
                // для этого и нужна: удержанное право может сгореть, и слот
                // уйдёт следующему. Кнопка гаснет, только когда товар выкуплен.
                disabled={item.soldOut}
                onClick={handleJoin}
              >
                {item.soldOut ? 'Товар распродан' : 'Встать в очередь'}
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
