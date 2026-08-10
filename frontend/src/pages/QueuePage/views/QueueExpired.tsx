import { useNavigate } from 'react-router-dom';

import { ItemCard, useSimilarItems } from '@/entities/item';
import { type QueueEntry, useJoinQueue } from '@/entities/queue-entry';
import { Button, Skeleton } from '@/shared/ui';

import { QueueScreen } from '../QueueScreen';
import styles from '../QueuePage.module.css';

interface QueueExpiredProps {
  entry: QueueEntry;
  left?: boolean;
}

export function QueueExpired({ entry, left = false }: QueueExpiredProps) {
  const navigate = useNavigate();
  const join = useJoinQueue();

  // Похожие лоты показываются только при сгоревшем праве: пользователь всё ещё
  // хочет купить, и аналог у другого продавца — способ не потерять его.
  // После добровольного выхода навязывать альтернативы незачем.
  const { data: similar, isLoading } = useSimilarItems(entry.itemId, !left);

  // Маршрут очереди адресуется товаром, поэтому после повторного входа
  // никуда переходить не нужно — хук статуса обновится сам.
  const handleRejoin = () => {
    join.mutate(entry.itemId);
  };

  const showSimilar = !left && (isLoading || (similar ?? []).length > 0);

  return (
    <QueueScreen
      itemId={entry.itemId}
      tone="red"
      statusLabel={left ? 'Вы вышли из очереди' : 'Время истекло'}
      title={left ? 'Вы покинули очередь' : 'Право на покупку истекло'}
      description={
        left
          ? 'Вы вышли из очереди по своему решению. Можно встать заново — вы займёте место в конце очереди.'
          : 'Вы не успели оформить покупку за отведённое время, и право перешло следующему участнику. Можно встать в очередь заново или посмотреть похожие лоты.'
      }
      wide={showSimilar}
      footer={
        <>
          <Button
            variant="secondary"
            onClick={() => {
              navigate('/');
            }}
          >
            В каталог
          </Button>
          <Button loading={join.isPending} onClick={handleRejoin}>
            Встать в очередь заново
          </Button>
        </>
      }
    >
      {showSimilar && (
        <div className={styles.similarGrid}>
          {isLoading &&
            Array.from({ length: 3 }).map((_, index) => (
              <Skeleton key={index} height={240} radius="var(--radius-lg)" />
            ))}
          {similar?.map((candidate) => (
            <ItemCard key={candidate.id} item={candidate} />
          ))}
        </div>
      )}
    </QueueScreen>
  );
}
