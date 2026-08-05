import { useNavigate } from 'react-router-dom';

import { type QueueEntry, useJoinQueue } from '@/entities/queue-entry';
import { Button } from '@/shared/ui';

import { QueueScreen } from '../QueueScreen';

interface QueueExpiredProps {
  entry: QueueEntry;
  left?: boolean;
}

export function QueueExpired({ entry, left = false }: QueueExpiredProps) {
  const navigate = useNavigate();
  const join = useJoinQueue();

  const handleRejoin = () => {
    join.mutate(entry.itemId, {
      onSuccess: (next) => {
        navigate(`/queue/${next.entryId}`);
      },
    });
  };

  return (
    <QueueScreen
      itemId={entry.itemId}
      tone="red"
      statusLabel={left ? 'Вы вышли из очереди' : 'Время истекло'}
      title={left ? 'Вы покинули очередь' : 'Право на покупку истекло'}
      description={
        left
          ? 'Вы вышли из очереди по своему решению. Можно встать заново — вы займёте место в конце очереди.'
          : 'Вы не успели оформить покупку за отведённое время, и право перешло следующему участнику. Можно встать в очередь заново.'
      }
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
    />
  );
}
