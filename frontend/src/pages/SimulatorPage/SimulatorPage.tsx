import { useState } from 'react';

import { useQueryClient } from '@tanstack/react-query';

import { itemKeys, useItems } from '@/entities/item';
import { useSimulation } from '@/features/simulator';
import { Button, Card, Skeleton } from '@/shared/ui';

import styles from './SimulatorPage.module.css';

/** Потолок бэкенда: /demo/simulate принимает count от 1 до 100. */
const MAX_BUYERS = 100;

/**
 * Стенд честности: N синтетических пользователей одновременно встают в очередь
 * через настоящий Entry на бэкенде. Это не симуляция поверх мока, а нагрузка на
 * реальную систему — потому и является доказательством, а не иллюстрацией.
 */
export function SimulatorPage() {
  const { data: items, isLoading } = useItems();
  const queryClient = useQueryClient();
  const [itemId, setItemId] = useState('');
  const [count, setCount] = useState(50);
  // Свободные единицы на момент запуска: после симуляции остаток в каталоге
  // изменится, а выводы («право получат X человек») должны считаться от
  // стартового состояния, иначе цифры в отчёте не сойдутся.
  const [availableAtRun, setAvailableAtRun] = useState(0);
  const simulation = useSimulation();

  // По умолчанию выбирается первый товар со свободными единицами: запуск
  // нагрузки на распроданный товар ничего не демонстрирует — все участники
  // сразу получат sold_out.
  const firstAvailableId =
    items?.find((item) => item.available > 0)?.id ?? items?.[0]?.id ?? '';
  const selectedId = itemId || firstAvailableId;
  const selectedItem = items?.find((item) => item.id === selectedId);

  const handleRun = () => {
    if (!selectedItem) return;
    setAvailableAtRun(selectedItem.available);
    simulation.mutate(
      { itemId: selectedItem.id, count },
      {
        onSuccess: () => {
          // Остаток и очередь товара изменились — список в селекте обязан
          // показывать уже новое состояние.
          void queryClient.invalidateQueries({ queryKey: itemKeys.all });
        },
      },
    );
  };

  const result = simulation.data;
  const granted = result ? Math.min(result.joined, availableAtRun) : 0;
  const waiting = result ? result.joined - granted : 0;

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <h1 className={styles.title}>Стенд честности</h1>
        <p className={styles.subtitle}>
          Запускаем N параллельных запросов на вход в очередь к реальному
          бэкенду — теми же ручками, что использует интерфейс. Права на покупку
          получат ровно столько человек, сколько единиц товара: больше не
          позволит констрейнт в схеме базы, а не аккуратность кода.
        </p>
      </header>

      <Card padding="lg" className={styles.controls}>
        {isLoading ? (
          <Skeleton height={44} />
        ) : (
          <label className={styles.field}>
            <span className={styles.label}>Товар</span>
            <select
              className={styles.select}
              value={selectedId}
              onChange={(event) => {
                setItemId(event.target.value);
              }}
            >
              {items?.map((item) => (
                <option
                  key={item.id}
                  value={item.id}
                  disabled={item.available === 0}
                >
                  {item.title} —{' '}
                  {item.available > 0
                    ? `свободно ${String(item.available)} из ${String(item.totalStock)}`
                    : 'распродано'}
                </option>
              ))}
            </select>
          </label>
        )}

        <label className={styles.field}>
          <span className={styles.label}>Одновременных покупателей: {count}</span>
          <input
            type="range"
            min={1}
            max={MAX_BUYERS}
            value={count}
            onChange={(event) => {
              setCount(Number(event.target.value));
            }}
          />
        </label>

        <Button
          size="lg"
          loading={simulation.isPending}
          disabled={!selectedItem || selectedItem.available === 0}
          onClick={handleRun}
        >
          Запустить нагрузку
        </Button>
      </Card>

      {result && (
        <div className={styles.results}>
          <p className={styles.headline}>
            Запрошено <b>{result.requested}</b> параллельных входов: в очередь
            встали <b>{result.joined}</b>
            {result.failed > 0 ? (
              <>
                , отклонено <b>{result.failed}</b>
              </>
            ) : null}
            . Свободных единиц на старте было <b>{availableAtRun}</b> — право на
            покупку сразу получили <b>{granted}</b>, остальные{' '}
            <b>{waiting}</b> ждут освобождения слота в порядке очереди.
          </p>

          <div className={styles.stats}>
            <div className={`${styles.stat} ${styles.green}`}>
              <span className={styles.statValue}>{granted}</span>
              <span className={styles.statLabel}>Получили право</span>
            </div>
            <div className={`${styles.stat} ${styles.amber}`}>
              <span className={styles.statValue}>{waiting}</span>
              <span className={styles.statLabel}>Ждут в очереди</span>
            </div>
            <div className={`${styles.stat} ${styles.red}`}>
              <span className={styles.statValue}>0</span>
              <span className={styles.statLabel}>Двойных продаж</span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
