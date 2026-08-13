import { type FormEvent, useState } from 'react';

import { type Item } from '@/entities/item';
import { type AdminItemInput } from '@/features/admin';
import { Button } from '@/shared/ui';

import styles from './AdminPage.module.css';

interface ItemFormProps {
  initial?: Item | null;
  pending: boolean;
  onSubmit: (input: AdminItemInput) => void;
  onCancel: () => void;
}

export function ItemForm({ initial, pending, onSubmit, onCancel }: ItemFormProps) {
  const [name, setName] = useState(initial?.title ?? '');
  const [priceRub, setPriceRub] = useState(
    initial ? String(Math.round(initial.priceKopecks / 100)) : '',
  );
  const [totalStock, setTotalStock] = useState(
    initial ? String(initial.totalStock) : '',
  );
  const [holdTtl, setHoldTtl] = useState(
    initial ? String(initial.holdTtlSeconds) : '120',
  );
  const [category, setCategory] = useState(initial?.category ?? '');
  const [seller, setSeller] = useState(initial?.sellerName ?? '');

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    onSubmit({
      name: name.trim(),
      price_kopecks: Math.round(Number(priceRub) * 100),
      total_stock: Number(totalStock),
      hold_ttl_seconds: Number(holdTtl),
      category: category.trim(),
      seller_name: seller.trim(),
    });
  };

  return (
    <form className={styles.form} onSubmit={handleSubmit}>
      <label className={styles.field}>
        <span>Название</span>
        <input
          className={styles.input}
          value={name}
          required
          onChange={(event) => {
            setName(event.target.value);
          }}
        />
      </label>

      <label className={styles.field}>
        <span>Цена, ₽</span>
        <input
          className={styles.input}
          type="number"
          min={0}
          value={priceRub}
          required
          onChange={(event) => {
            setPriceRub(event.target.value);
          }}
        />
      </label>

      <div className={styles.formRow}>
        <label className={styles.field}>
          <span>Тираж, шт.</span>
          <input
            className={styles.input}
            type="number"
            min={1}
            value={totalStock}
            required
            onChange={(event) => {
              setTotalStock(event.target.value);
            }}
          />
        </label>

        <label className={styles.field}>
          <span>Время права, сек</span>
          <input
            className={styles.input}
            type="number"
            min={1}
            value={holdTtl}
            required
            onChange={(event) => {
              setHoldTtl(event.target.value);
            }}
          />
        </label>
      </div>

      <label className={styles.field}>
        <span>Категория</span>
        <input
          className={styles.input}
          value={category}
          required
          onChange={(event) => {
            setCategory(event.target.value);
          }}
        />
      </label>

      <label className={styles.field}>
        <span>Продавец</span>
        <input
          className={styles.input}
          value={seller}
          required
          onChange={(event) => {
            setSeller(event.target.value);
          }}
        />
      </label>

      <div className={styles.formActions}>
        <Button type="button" variant="ghost" onClick={onCancel}>
          Отмена
        </Button>
        <Button type="submit" loading={pending}>
          Сохранить
        </Button>
      </div>
    </form>
  );
}
