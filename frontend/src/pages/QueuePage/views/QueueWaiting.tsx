import { useRef, useState } from 'react';

import { useNavigate } from 'react-router-dom';

import {
  type QueueEntry,
  useEta,
  useLeaveQueue,
} from '@/entities/queue-entry';
import { formatEta } from '@/shared/lib/formatTime';
import { Button, Modal, ProgressBar } from '@/shared/ui';

import { QueueScreen } from '../QueueScreen';
import styles from '../QueuePage.module.css';

interface QueueWaitingProps {
  entry: QueueEntry;
}

export function QueueWaiting({ entry }: QueueWaitingProps) {
  const navigate = useNavigate();
  const leave = useLeaveQueue();
  const eta = useEta(entry.entryId, true);
  const [confirmOpen, setConfirmOpen] = useState(false);

  const confidenceLabel: Record<string, string> = {
    high: 'высокая',
    medium: 'средняя',
    low: 'низкая',
  };

  const initialAhead = useRef(entry.totalAhead);
  if (entry.totalAhead > initialAhead.current) {
    initialAhead.current = entry.totalAhead;
  }

  const done = initialAhead.current - entry.totalAhead;
  const progress =
    initialAhead.current > 0 ? (done / initialAhead.current) * 100 : 100;

  const handleLeave = () => {
    leave.mutate(entry.entryId, {
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
      title={
        entry.totalAhead > 0
          ? `Перед вами ${String(entry.totalAhead)} чел.`
          : 'Вы следующий в очереди'
      }
      description="Не закрывайте страницу надолго: когда подойдёт очередь, вы получите право на покупку с ограниченным временем."
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

      {eta.data && (
        <span className={styles.eta}>
          Прогноз ИИ: {formatEta(eta.data.seconds)} ожидания · точность{' '}
          {confidenceLabel[eta.data.confidence] ?? eta.data.confidence}
        </span>
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
