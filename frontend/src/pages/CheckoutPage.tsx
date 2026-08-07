import { useEffect, useRef } from 'react';

import { useNavigate, useParams } from 'react-router-dom';

import { useItem } from '@/entities/item';
import { useCheckout, useEntryByItem } from '@/entities/queue-entry';
import { formatPrice } from '@/shared/lib/formatPrice';
import {
  Button,
  Card,
  CountdownTimer,
  Skeleton,
  showToast,
} from '@/shared/ui';

import styles from './CheckoutPage.module.css';

export function CheckoutPage() {
  const itemId = useParams().itemId ?? '';
  const navigate = useNavigate();

  const { data: entry, isLoading } = useEntryByItem(itemId);
  const { data: item } = useItem(itemId);
  const checkoutMutation = useCheckout();
  const submitted = useRef(false);

  const hasRight = entry?.status === 'GRANTED';

  useEffect(() => {
    if (isLoading || submitted.current) return;
    if (!hasRight) {
      showToast('Нет активного права на покупку этого товара', 'red');
      navigate(`/item/${itemId}`, { replace: true });
    }
  }, [hasRight, isLoading, itemId, navigate]);

  if (isLoading) {
    return (
      <div className={styles.center}>
        <Skeleton height={320} width={420} radius="var(--radius-lg)" />
      </div>
    );
  }

  if (!entry || !hasRight) {
    return null;
  }

  const handleConfirm = () => {
    submitted.current = true;
    checkoutMutation.mutate(entry.entryId, {
      onSuccess: (order) => {
        navigate(`/success/${order.orderId}`, { replace: true });
      },
      onError: () => {
        submitted.current = false;
        showToast('Не удалось оформить: право могло истечь', 'red');
      },
    });
  };

  return (
    <div className={styles.center}>
      <Card padding="lg" className={styles.panel}>
        <span className={styles.tag}>Оформление покупки</span>

        {item && (
          <div className={styles.item}>
            <span
              className={styles.emoji}
              style={{ background: `${item.accent}22` }}
            >
              {item.emoji}
            </span>
            <div className={styles.itemInfo}>
              <span className={styles.seller}>{item.sellerName}</span>
              <h1 className={styles.title}>{item.title}</h1>
              <span className={styles.price}>{formatPrice(item.price)}</span>
            </div>
          </div>
        )}

        <div className={styles.timerBlock}>
          <p className={styles.hint}>Право действует ещё:</p>
          {entry.expiresAt && (
            <CountdownTimer
              expiresAt={Date.parse(entry.expiresAt)}
              totalSeconds={120}
              onExpire={() => {
                showToast('Время на покупку истекло', 'red');
                navigate(`/item/${itemId}`, { replace: true });
              }}
            />
          )}
        </div>

        <Button
          size="lg"
          fullWidth
          loading={checkoutMutation.isPending}
          onClick={handleConfirm}
        >
          Подтвердить покупку
        </Button>
      </Card>
    </div>
  );
}
