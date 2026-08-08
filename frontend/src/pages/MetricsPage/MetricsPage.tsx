import { useMetrics } from '@/features/metrics';
import { formatEta } from '@/shared/lib/formatTime';
import { Card, Skeleton } from '@/shared/ui';

import styles from './MetricsPage.module.css';

export function MetricsPage() {
  const { data, isLoading, isError } = useMetrics();

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <div>
          <h1 className={styles.title}>Метрики очереди</h1>
          <p className={styles.subtitle}>
            Живые показатели по всем товарам. Обновляются автоматически — встаньте
            в очередь в соседней вкладке и цифры оживут.
          </p>
        </div>
        <span className={styles.live}>
          <span className={styles.dot} /> live
        </span>
      </header>

      {isError && (
        <p className={styles.error}>Не удалось загрузить метрики.</p>
      )}

      {isLoading && !data ? (
        <Skeleton height={120} radius="var(--radius-lg)" />
      ) : (
        data && (
          <>
            <div className={styles.stats}>
              <Card padding="md" className={styles.stat}>
                <span className={styles.statValue}>{data.totalWaiting}</span>
                <span className={styles.statLabel}>Сейчас в очереди</span>
              </Card>
              <Card padding="md" className={styles.stat}>
                <span className={styles.statValue}>{data.totalGranted}</span>
                <span className={styles.statLabel}>С правом на покупку</span>
              </Card>
              <Card padding="md" className={styles.stat}>
                <span className={styles.statValue}>
                  {data.conversion === null
                    ? '—'
                    : `${String(Math.round(data.conversion * 100))}%`}
                </span>
                <span className={styles.statLabel}>Конверсия права → покупка</span>
              </Card>
              <Card padding="md" className={styles.stat}>
                <span className={styles.statValue}>
                  {data.avgWaitSeconds === null
                    ? '—'
                    : formatEta(data.avgWaitSeconds).replace('≈ ', '')}
                </span>
                <span className={styles.statLabel}>Среднее ожидание</span>
              </Card>
            </div>

            <Card padding="none" className={styles.tableCard}>
              <table className={styles.table}>
                <thead>
                  <tr>
                    <th>Товар</th>
                    <th>В очереди</th>
                    <th>С правом</th>
                    <th>Осталось</th>
                  </tr>
                </thead>
                <tbody>
                  {data.items.length === 0 && (
                    <tr>
                      <td colSpan={4} className={styles.empty}>
                        Пока нет активности. Встаньте в очередь на любой товар.
                      </td>
                    </tr>
                  )}
                  {data.items.map((item) => (
                    <tr key={item.itemId}>
                      <td>{item.title}</td>
                      <td>{item.waiting}</td>
                      <td>{item.granted}</td>
                      <td>{item.stockLeft}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </Card>
          </>
        )
      )}
    </div>
  );
}
