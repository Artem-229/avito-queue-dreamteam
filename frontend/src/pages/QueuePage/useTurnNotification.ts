import { useEffect, useRef } from 'react';

import { type QueueStatus } from '@/entities/queue-entry';
import { showToast } from '@/shared/ui';

export function useTurnNotification(status: QueueStatus | undefined): void {
  const previous = useRef<QueueStatus | undefined>(undefined);

  useEffect(() => {
    if (
      status === 'waiting' &&
      typeof Notification !== 'undefined' &&
      Notification.permission === 'default'
    ) {
      void Notification.requestPermission();
    }

    if (
      status === 'granted' &&
      previous.current !== undefined &&
      previous.current !== 'granted'
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
