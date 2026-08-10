import { useCallback } from 'react';

import { useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';

import { useItem } from '@/entities/item';
import { queueKeys, type QueueEntry } from '@/entities/queue-entry';
import { getClockSkewMs } from '@/shared/lib/useCountdown';
import { Button, CountdownTimer } from '@/shared/ui';

import { QueueScreen } from '../QueueScreen';

interface QueueGrantedProps {
  entry: QueueEntry;
}

const FALLBACK_TTL_SECONDS = 120;

export function QueueGranted({ entry }: QueueGrantedProps) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { data: item } = useItem(entry.itemId);

  const expiresAtMs = entry.expiresAt
    ? Date.parse(entry.expiresAt)
    : Date.now();

  // Таймер считается от времени сервера, а не от часов браузера: бэкенд
  // присылает server_time тем же ответом, что и expires_at.
  const skewMs = getClockSkewMs(entry.serverTime);

  const handleExpire = useCallback(() => {
    void queryClient.invalidateQueries({
      queryKey: queueKeys.status(entry.itemId),
    });
  }, [queryClient, entry.itemId]);

  return (
    <QueueScreen
      itemId={entry.itemId}
      tone="green"
      statusLabel="Право на покупку — ваше"
      pulse
      title="Оформите покупку вовремя"
      description={entry.message}
      footer={
        <Button
          size="lg"
          onClick={() => {
            navigate(`/checkout/${entry.itemId}`);
          }}
        >
          {entry.nextStep.label}
        </Button>
      }
    >
      <CountdownTimer
        expiresAt={expiresAtMs}
        totalSeconds={item?.holdTtlSeconds ?? FALLBACK_TTL_SECONDS}
        onExpire={handleExpire}
        skewMs={skewMs}
      />
    </QueueScreen>
  );
}
