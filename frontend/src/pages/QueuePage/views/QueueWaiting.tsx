import { useRef, useState } from 'react';

import { useNavigate } from 'react-router-dom';

import { type QueueEntry, useLeaveQueue } from '@/entities/queue-entry';
import { Button, Modal, ProgressBar } from '@/shared/ui';

import { QueueScreen } from '../QueueScreen';
import styles from '../QueuePage.module.css';

interface QueueWaitingProps {
  entry: QueueEntry;
}

export function QueueWaiting({ entry }: QueueWaitingProps) {
  const navigate = useNavigate();
  const leave = useLeaveQueue();
  const [confirmOpen, setConfirmOpen] = useState(false);

  // position — место в очереди начиная с 1, значит перед пользователем
  // стоит на одного человека меньше.
  const ahead = Math.max(0, entry.position - 1);

  const initialAhead = useRef(ahead);
  if (ahead > initialAhead.current) {
    initialAhead.current = ahead;
  }

  const done = initialAhead.current - ahead;
  const progress =
    initialAhead.current > 0 ? (done / initialAhead.current) * 100 : 100;

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
      title={
        ahead > 0
          ? `Перед вами ${String(ahead)} чел.`
          : 'Вы следующий в очереди'
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
      </span>

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
