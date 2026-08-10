import { useNavigate } from 'react-router-dom';

import { type QueueEntry, useJoinQueue } from '@/entities/queue-entry';
import { Button } from '@/shared/ui';

import { QueueScreen } from '../QueueScreen';

interface QueueNotInQueueProps {
  entry: QueueEntry;
}

/**
 * Состояние «в очереди не стоит» — полноценный экран, а не пустая страница:
 * у каждого состояния должны быть сообщение и следующий шаг.
 */
export function QueueNotInQueue({ entry }: QueueNotInQueueProps) {
  const navigate = useNavigate();
  const join = useJoinQueue();

  return (
    <QueueScreen
      itemId={entry.itemId}
      tone="blue"
      statusLabel="Вы не в очереди"
      title="Вы ещё не встали в очередь"
      description={entry.message}
      footer={
        <>
          <Button
            variant="secondary"
            onClick={() => {
              navigate(`/item/${entry.itemId}`);
            }}
          >
            К товару
          </Button>
          <Button
            loading={join.isPending}
            onClick={() => {
              join.mutate(entry.itemId);
            }}
          >
            {entry.nextStep.label}
          </Button>
        </>
      }
    />
  );
}
