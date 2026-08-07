import { ItemCard, useItems } from '@/entities/item';
import { Skeleton } from '@/shared/ui';

import styles from './CatalogPage.module.css';

export function CatalogPage() {
  const { data: items, isLoading, isError } = useItems();

  return (
    <section className={styles.page}>
      <header className={styles.header}>
        <h1 className={styles.title}>Дефицитные товары</h1>
        <p className={styles.subtitle}>
          Товары с ограниченным выпуском продаются через очередь: встаньте в
          очередь, дождитесь права на покупку и оформите заказ вовремя.
        </p>
      </header>

      {isError && (
        <p className={styles.error}>Не удалось загрузить каталог. Обновите страницу.</p>
      )}

      <div className={styles.grid}>
        {isLoading &&
          Array.from({ length: 6 }).map((_, index) => (
            <Skeleton
              key={index}
              height={280}
              radius="var(--radius-lg)"
            />
          ))}

        {items?.map((item) => (
          <ItemCard key={item.id} item={item} />
        ))}
      </div>
    </section>
  );
}
