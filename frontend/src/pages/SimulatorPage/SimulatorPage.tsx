import { useState } from 'react';

import { useSimulation } from '@/features/simulator';
import { Button, Card } from '@/shared/ui';

import styles from './SimulatorPage.module.css';

const OUTCOME_LABEL: Record<string, string> = {
  PURCHASED: 'Купил',
  EXPIRED: 'Не успел',
  SOLD_OUT: 'Не досталось',
};

export function SimulatorPage() {
  const [buyers, setBuyers] = useState(20);
  const [stock, setStock] = useState(3);
  const [buyProbability, setBuyProbability] = useState(0.7);
  const simulation = useSimulation();

  const result = simulation.data;

  const handleRun = () => {
    simulation.mutate({
      buyers,
      stock,
      buyProbability,
      seed: Math.floor(Math.random() * 1_000_000),
    });
  };

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <h1 className={styles.title}>Симулятор очереди</h1>
        <p className={styles.subtitle}>
          Запускаем N виртуальных покупателей на ограниченный товар и смотрим, как
          распределяются права на покупку. Главное: право получают ровно столько
          человек, сколько единиц товара — двойных продаж быть не может.
        </p>
      </header>

      <Card padding="lg" className={styles.controls}>
        <label className={styles.field}>
          <span className={styles.label}>Покупателей: {buyers}</span>
          <input
            type="range"
            min={1}
            max={100}
            value={buyers}
            onChange={(event) => {
              setBuyers(Number(event.target.value));
            }}
          />
        </label>

        <label className={styles.field}>
          <span className={styles.label}>Единиц товара: {stock}</span>
          <input
            type="range"
            min={0}
            max={Math.max(1, buyers)}
            value={stock}
            onChange={(event) => {
              setStock(Number(event.target.value));
            }}
          />
        </label>

        <label className={styles.field}>
          <span className={styles.label}>
            Вероятность покупки при праве: {Math.round(buyProbability * 100)}%
          </span>
          <input
            type="range"
            min={0}
            max={100}
            value={Math.round(buyProbability * 100)}
            onChange={(event) => {
              setBuyProbability(Number(event.target.value) / 100);
            }}
          />
        </label>

        <Button
          size="lg"
          loading={simulation.isPending}
          onClick={handleRun}
        >
          Запустить симуляцию
        </Button>
      </Card>

      {result && (
        <div className={styles.results}>
          <p className={styles.headline}>
            Право на покупку реализовали <b>{result.unitsSold}</b> из{' '}
            {result.buyers} — ровно по числу товаров ({result.stock}). Двойных
            продаж нет.
          </p>

          <div className={styles.stats}>
            <div className={`${styles.stat} ${styles.green}`}>
              <span className={styles.statValue}>
                {result.distribution.purchased}
              </span>
              <span className={styles.statLabel}>Купили</span>
            </div>
            <div className={`${styles.stat} ${styles.amber}`}>
              <span className={styles.statValue}>
                {result.distribution.expired}
              </span>
              <span className={styles.statLabel}>Не успели (право истекло)</span>
            </div>
            <div className={`${styles.stat} ${styles.red}`}>
              <span className={styles.statValue}>
                {result.distribution.soldOut}
              </span>
              <span className={styles.statLabel}>Не досталось</span>
            </div>
          </div>

          <div className={styles.bar}>
            {result.distribution.purchased > 0 && (
              <div
                className={styles.segGreen}
                style={{
                  flexGrow: result.distribution.purchased,
                }}
              />
            )}
            {result.distribution.expired > 0 && (
              <div
                className={styles.segAmber}
                style={{ flexGrow: result.distribution.expired }}
              />
            )}
            {result.distribution.soldOut > 0 && (
              <div
                className={styles.segRed}
                style={{ flexGrow: result.distribution.soldOut }}
              />
            )}
          </div>

          <div className={styles.grid}>
            {result.timeline.map((buyer) => (
              <span
                key={buyer.order}
                className={`${styles.chip} ${styles[`chip-${buyer.outcome}`]}`}
                title={`#${String(buyer.order)} — ${OUTCOME_LABEL[buyer.outcome] ?? buyer.outcome}`}
              >
                {buyer.order}
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
