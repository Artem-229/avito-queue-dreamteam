import { useNavigate } from 'react-router-dom';

import { type QueueEntry } from '@/entities/queue-entry';
import { Button } from '@/shared/ui';

import { QueueScreen } from '../QueueScreen';

interface QueuePurchasedProps {
  entry: QueueEntry;
}

export function QueuePurchased({ entry }: QueuePurchasedProps) {
  const navigate = useNavigate();

  return (
    <QueueScreen
      itemId={entry.itemId}
      tone="blue"
      statusLabel="Покупка оформлена"
      title="Товар уже куплен"
      description="По этому праву на покупку заказ уже оформлен. Спасибо!"
      footer={
        <Button
          onClick={() => {
            navigate('/');
          }}
        >
          В каталог
        </Button>
      }
    />
  );
}
