import { useCallback } from 'react';

import { useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';

import { queueKeys, type QueueEntry } from '@/entities/queue-entry';
import { Button, CountdownTimer } from '@/shared/ui';

import { QueueScreen } from '../QueueScreen';

interface QueueGrantedProps {
  entry: QueueEntry;
}

function computeTotalSeconds(entry: QueueEntry): number {
  if (entry.grantedAt && entry.expiresAt) {
    const diff = Date.parse(entry.expiresAt) - Date.parse(entry.grantedAt);
    if (diff > 0) return Math.round(diff / 1000);
  }
  return 120;
}

export function QueueGranted({ entry }: QueueGrantedProps) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const expiresAtMs = entry.expiresAt ? Date.parse(entry.expiresAt) : Date.now();

  const handleExpire = useCallback(() => {
    void queryClient.invalidateQueries({
      queryKey: queueKeys.entry(entry.entryId),
    });
  }, [queryClient, entry.entryId]);

  return (
    <QueueScreen
      itemId={entry.itemId}
      tone="green"
      statusLabel="Право на покупку — ваше"
      pulse
      title="Оформите покупку вовремя"
      description="Право персональное и действует ограниченное время. Успейте перейти к оплате, пока идёт таймер — иначе оно перейдёт следующему."
      footer={
        <Button
          size="lg"
          onClick={() => {
            navigate(`/checkout/${entry.itemId}`);
          }}
        >
          Перейти к оплате
        </Button>
      }
    >
      <CountdownTimer
        expiresAt={expiresAtMs}
        totalSeconds={computeTotalSeconds(entry)}
        onExpire={handleExpire}
      />
    </QueueScreen>
  );
}
