import { useNavigate, useParams } from 'react-router-dom';

import { Button, Card } from '@/shared/ui';

import styles from './SuccessPage.module.css';

export function SuccessPage() {
  const orderId = useParams().orderId ?? '';
  const navigate = useNavigate();

  return (
    <div className={styles.center}>
      <Card padding="lg" className={styles.panel}>
        <span className={styles.check} aria-hidden="true">
          ✓
        </span>
        <h1 className={styles.title}>Покупка оформлена</h1>
        <p className={styles.description}>
          Заказ успешно создан. Право на покупку использовано — товар
          зарезервирован за вами.
        </p>
        <div className={styles.orderRow}>
          <span className={styles.orderLabel}>Номер заказа</span>
          <span className={styles.orderId}>{orderId}</span>
        </div>
        <Button
          size="lg"
          fullWidth
          onClick={() => {
            navigate('/');
          }}
        >
          Вернуться в каталог
        </Button>
      </Card>
    </div>
  );
}
