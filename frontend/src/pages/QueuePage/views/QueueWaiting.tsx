import { useRef, useState } from 'react';

import { useNavigate } from 'react-router-dom';

import { ItemCard, useSimilarItems } from '@/entities/item';
import { type QueueEntry, useLeaveQueue } from '@/entities/queue-entry';
import { chanceLabel, chanceTone } from '@/shared/lib/chance';
import { Button, Modal, ProgressBar, Skeleton, StatusPill } from '@/shared/ui';

import { QueueScreen } from '../QueueScreen';
import styles from '../QueuePage.module.css';

interface QueueWaitingProps {
  entry: QueueEntry;
}

export function QueueWaiting({ entry }: QueueWaitingProps) {
  const navigate = useNavigate();
  const leave = useLeaveQueue();
  const { data: similar, isLoading: similarLoading } = useSimilarItems(
    entry.itemId,
  );
  const [confirmOpen, setConfirmOpen] = useState(false);

  // position — место в очереди начиная с 1, значит перед пользователем стоит на
  // одного человека меньше.
  const ahead = Math.max(0, entry.position - 1);

  // Прогресс считает бэкенд (progress_percent). Клиентский расчёт остаётся
  // запасным для записей без этого поля.
  const clientInitialAhead = useRef(ahead);
  if (ahead > clientInitialAhead.current) {
    clientInitialAhead.current = ahead;
  }
  const clientProgress =
    clientInitialAhead.current > 0
      ? ((clientInitialAhead.current - ahead) / clientInitialAhead.current) * 100
      : 100;
  const progress = entry.progressPercent ?? clientProgress;

  const moved =
    entry.initialPosition !== null && entry.initialPosition > entry.position;

  const showSimilar = similarLoading || (similar ?? []).length > 0;

  const handleLeave = () => {
    leave.mutate(entry.itemId, {
      onSuccess: () => {
        navigate(`/item/${entry.itemId}`);
      },
    });
  };

  return (
    <QueueScreen
      itemId={entry.itemId}
      tone="purple"
      statusLabel="Вы в очереди"
      pulse
      wide={showSimilar}
      title={
        ahead > 0 ? `Перед вами ${String(ahead)} чел.` : 'Вы следующий в очереди'
      }
      description={entry.message}
      footer={
        <Button
          variant="ghost"
          onClick={() => {
            setConfirmOpen(true);
          }}
        >
          Выйти из очереди
        </Button>
      }
    >
      <div className={styles.progressWrap}>
        <ProgressBar value={progress} tone="queue" />
      </div>

      <span className={styles.eta}>
        Всего в очереди: {String(entry.queueSize)} чел. · ваше место{' '}
        {String(entry.position)}
        {moved && entry.initialPosition !== null
          ? ` (были ${String(entry.initialPosition)})`
          : ''}
      </span>

      {entry.chance && (
        <StatusPill
          tone={chanceTone(entry.chance.percent)}
          label={chanceLabel(entry.chance)}
        />
      )}

      {showSimilar && (
        <div className={styles.similarBlock}>
          <span className={styles.similarTitle}>
            Пока ждёте — присмотритесь к похожим
          </span>
          <div className={styles.similarGrid}>
            {similarLoading &&
              Array.from({ length: 3 }).map((_, index) => (
                <Skeleton key={index} height={240} radius="var(--radius-lg)" />
              ))}
            {similar?.map((candidate) => (
              <ItemCard key={candidate.id} item={candidate} />
            ))}
          </div>
        </div>
      )}

      <Modal
        open={confirmOpen}
        onClose={() => {
          setConfirmOpen(false);
        }}
        title="Выйти из очереди?"
        footer={
          <>
            <Button
              variant="ghost"
              onClick={() => {
                setConfirmOpen(false);
              }}
            >
              Остаться
            </Button>
            <Button
              variant="danger"
              loading={leave.isPending}
              onClick={handleLeave}
            >
              Выйти
            </Button>
          </>
        }
      >
        Вы потеряете текущую позицию. Чтобы купить товар снова, придётся встать в
        очередь заново.
      </Modal>
    </QueueScreen>
  );
}
