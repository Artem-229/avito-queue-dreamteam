import { Link } from 'react-router-dom';

import { formatPrice } from '@/shared/lib/formatPrice';
import { Badge, Card } from '@/shared/ui';

import { type Item } from '../model/types';
import styles from './ItemCard.module.css';

interface ItemCardProps {
  item: Item;
}

/**
 * «Свободных нет» и «распродано» — разные вещи: в первом случае слоты держат
 * права, которые могут сгореть, и очередь имеет смысл; во втором товар
 * выкуплен и ждать нечего.
 */
function stockLabel(item: Item): string {
  if (item.soldOut) return 'распродано';
  if (item.available > 0) return `${String(item.available)} шт.`;
  return 'сейчас разобран';
}

export function ItemCard({ item }: ItemCardProps) {
  return (
    <Link to={`/item/${item.id}`} className={styles.link}>
      <Card padding="none" interactive className={styles.card}>
        <div
          className={styles.cover}
          style={{ background: `${item.accent}22` }}
        >
          <span className={styles.emoji}>{item.emoji}</span>
          {item.queueEnabled && (
            <span className={styles.badge}>
              <Badge tone="purple">Ограниченный выпуск</Badge>
            </span>
          )}
        </div>

        <div className={styles.body}>
          <span className={styles.seller}>{item.sellerName}</span>
          <h3 className={styles.title}>{item.title}</h3>
          <div className={styles.footer}>
            <span className={styles.price}>{formatPrice(item.priceKopecks)}</span>
            <span className={styles.stock}>{stockLabel(item)}</span>
          </div>
        </div>
      </Card>
    </Link>
  );
}
