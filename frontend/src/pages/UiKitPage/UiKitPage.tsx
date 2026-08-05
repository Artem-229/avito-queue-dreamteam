import { type ReactNode, useState } from 'react';

import {
  Badge,
  Button,
  Card,
  CountdownTimer,
  Modal,
  ProgressBar,
  Skeleton,
  StatusPill,
} from '@/shared/ui';

import styles from './UiKitPage.module.css';

function Section({
  title,
  children,
}: {
  title: string;
  children: ReactNode;
}) {
  return (
    <section className={styles.section}>
      <h2 className={styles.sectionTitle}>{title}</h2>
      <div className={styles.row}>{children}</div>
    </section>
  );
}

export function UiKitPage() {
  const [modalOpen, setModalOpen] = useState(false);
  const [expiresAt, setExpiresAt] = useState(() => Date.now() + 120_000);

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <h1 className={styles.title}>UI-kit</h1>
        <p className={styles.subtitle}>
          Дизайн-система «Авито Очередь». Витрина для самопроверки компонентов.
        </p>
      </header>

      <Section title="Кнопки">
        <Button variant="primary">Встать в очередь</Button>
        <Button variant="secondary">Подробнее</Button>
        <Button variant="ghost">Отмена</Button>
        <Button variant="danger">Выйти из очереди</Button>
        <Button loading>Загрузка</Button>
        <Button disabled>Недоступно</Button>
      </Section>

      <Section title="Размеры кнопок">
        <Button size="sm">Small</Button>
        <Button size="md">Medium</Button>
        <Button size="lg">Large</Button>
      </Section>

      <Section title="Бейджи">
        <Badge tone="purple">Ограниченный выпуск</Badge>
        <Badge tone="green">В наличии</Badge>
        <Badge tone="red">Распродано</Badge>
        <Badge tone="amber">Осталось мало</Badge>
        <Badge tone="blue">Новинка</Badge>
        <Badge>Neutral</Badge>
      </Section>

      <Section title="Статусы очереди">
        <StatusPill tone="purple" label="В очереди" pulse />
        <StatusPill tone="green" label="Право на покупку" pulse />
        <StatusPill tone="red" label="Время истекло" />
        <StatusPill tone="amber" label="Переподключение" pulse />
        <StatusPill tone="blue" label="Покупка оформлена" />
      </Section>

      <Section title="Карточки">
        <Card padding="md">Обычная карточка</Card>
        <Card padding="md" interactive>
          Интерактивная карточка (наведи)
        </Card>
      </Section>

      <Section title="Прогресс очереди">
        <div className={styles.stack}>
          <ProgressBar value={35} tone="queue" label="Позиция 13 из 20" />
          <ProgressBar value={80} tone="granted" label="Обслуживание" />
          <ProgressBar value={55} tone="blue" label="Синий" />
        </div>
      </Section>

      <Section title="Таймер права на покупку">
        <div className={styles.stack}>
          <CountdownTimer expiresAt={expiresAt} totalSeconds={120} />
          <Button
            variant="secondary"
            size="sm"
            onClick={() => {
              setExpiresAt(Date.now() + 15_000);
            }}
          >
            Запустить 15 c (проверить amber/red)
          </Button>
        </div>
      </Section>

      <Section title="Skeleton">
        <div className={styles.stack}>
          <Skeleton width={220} height={20} />
          <Skeleton width={160} height={20} />
          <Skeleton width={120} height={120} radius="var(--radius-lg)" />
        </div>
      </Section>

      <Section title="Модальное окно">
        <Button
          onClick={() => {
            setModalOpen(true);
          }}
        >
          Открыть модалку
        </Button>
        <Modal
          open={modalOpen}
          onClose={() => {
            setModalOpen(false);
          }}
          title="Выйти из очереди?"
          footer={
            <>
              <Button
                variant="ghost"
                onClick={() => {
                  setModalOpen(false);
                }}
              >
                Остаться
              </Button>
              <Button
                variant="danger"
                onClick={() => {
                  setModalOpen(false);
                }}
              >
                Выйти
              </Button>
            </>
          }
        >
          Если выйдете, потеряете текущую позицию и встанете в конец очереди.
        </Modal>
      </Section>
    </div>
  );
}
