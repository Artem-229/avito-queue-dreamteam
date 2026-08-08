import { useEffect, useRef } from 'react';

import { type QueueStatus } from '@/entities/queue-entry';
import { showToast } from '@/shared/ui';

export function useTurnNotification(status: QueueStatus | undefined): void {
  const previous = useRef<QueueStatus | undefined>(undefined);

  useEffect(() => {
    if (
      status === 'QUEUED' &&
      typeof Notification !== 'undefined' &&
      Notification.permission === 'default'
    ) {
      void Notification.requestPermission();
    }

    if (
      status === 'GRANTED' &&
      previous.current !== undefined &&
      previous.current !== 'GRANTED'
    ) {
      showToast('Ваша очередь подошла — у вас есть право на покупку!', 'green');

      if (
        typeof Notification !== 'undefined' &&
        Notification.permission === 'granted' &&
        document.hidden
      ) {
        new Notification('Авито Очередь', {
          body: 'Ваша очередь подошла! Успейте оформить покупку вовремя.',
        });
      }
    }

    previous.current = status;
  }, [status]);
}
