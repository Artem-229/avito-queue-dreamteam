import { useState } from 'react';

import { type Item, useItems } from '@/entities/item';
import { type AdminItemInput, useAdminDeleteItem, useAdminSaveItem, useAdminStore } from '@/features/admin';
import { useMetrics } from '@/features/metrics';
import { formatPrice } from '@/shared/lib/formatPrice';
import { Button, Card, Modal, Skeleton, showToast } from '@/shared/ui';

import { AdminLogin } from './AdminLogin';
import { ItemForm } from './ItemForm';
import styles from './AdminPage.module.css';

export function AdminPage() {
  const key = useAdminStore((state) => state.key);
  const clear = useAdminStore((state) => state.clear);
  const { data: items, isLoading } = useItems();
  const metrics = useMetrics();
  const save = useAdminSaveItem();
  const remove = useAdminDeleteItem();

  // undefined — форма закрыта, null — создание, Item — редактирование.
  const [editing, setEditing] = useState<Item | null | undefined>(undefined);
  const [deleting, setDeleting] = useState<Item | null>(null);

  if (!key) {
    return <AdminLogin />;
  }

  const closeForm = () => {
    setEditing(undefined);
  };

  const handleSave = (input: AdminItemInput) => {
    save.mutate(
      { id: editing?.id ?? null, input },
      {
        onSuccess: () => {
          showToast(editing ? 'Товар обновлён' : 'Товар создан', 'green');
          closeForm();
        },
        onError: () => {
          showToast('Не удалось сохранить товар', 'red');
        },
      },
    );
  };

  const handleDelete = () => {
    if (!deleting) return;
    remove.mutate(deleting.id, {
      onSuccess: () => {
        showToast('Товар удалён', 'green');
        setDeleting(null);
      },
      onError: () => {
        showToast('Не удалось удалить товар', 'red');
      },
    });
  };

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <div>
          <h1 className={styles.title}>Админ-панель</h1>
          <p className={styles.subtitle}>
            Управление товарами и метрики по очередям.
          </p>
        </div>
        <Button variant="ghost" onClick={clear}>
          Выйти
        </Button>
      </header>

      <Card padding="lg" className={styles.section}>
        <div className={styles.sectionHead}>
          <h2 className={styles.sectionTitle}>Товары</h2>
          <Button
            onClick={() => {
              setEditing(null);
            }}
          >
            Создать товар
          </Button>
        </div>

        {isLoading ? (
          <Skeleton height={160} />
        ) : (
          <div className={styles.tableWrap}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>Название</th>
                  <th>Цена</th>
                  <th>Тираж</th>
                  <th>Свободно</th>
                  <th>Категория</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {items?.map((item) => (
                  <tr key={item.id}>
                    <td>{item.title}</td>
                    <td>{formatPrice(item.priceKopecks)}</td>
                    <td>{item.totalStock}</td>
                    <td>{item.available}</td>
                    <td>{item.category}</td>
                    <td className={styles.rowActions}>
                      <Button
                        size="sm"
                        variant="secondary"
                        onClick={() => {
                          setEditing(item);
                        }}
                      >
                        Изменить
                      </Button>
                      <Button
                        size="sm"
                        variant="danger"
                        onClick={() => {
                          setDeleting(item);
                        }}
                      >
                        Удалить
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>

      <Card padding="lg" className={styles.section}>
        <h2 className={styles.sectionTitle}>Метрики по очередям</h2>
        {metrics.isLoading ? (
          <Skeleton height={120} />
        ) : (
          <div className={styles.tableWrap}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>Товар</th>
                  <th>В очереди</th>
                  <th>С правом</th>
                  <th>Остаток</th>
                </tr>
              </thead>
              <tbody>
                {metrics.data && metrics.data.items.length > 0 ? (
                  metrics.data.items.map((row) => (
                    <tr key={row.itemId}>
                      <td>{row.title}</td>
                      <td>{row.waiting}</td>
                      <td>{row.granted}</td>
                      <td>{row.stockLeft}</td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={4} className={styles.empty}>
                      Пока нет активности в очередях.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        )}
      </Card>

      <Modal
        open={editing !== undefined}
        onClose={closeForm}
        title={editing ? 'Изменить товар' : 'Создать товар'}
      >
        <ItemForm
          key={editing?.id ?? 'new'}
          initial={editing}
          pending={save.isPending}
          onSubmit={handleSave}
          onCancel={closeForm}
        />
      </Modal>

      <Modal
        open={deleting !== null}
        onClose={() => {
          setDeleting(null);
        }}
        title="Удалить товар?"
        footer={
          <>
            <Button
              variant="ghost"
              onClick={() => {
                setDeleting(null);
              }}
            >
              Отмена
            </Button>
            <Button
              variant="danger"
              loading={remove.isPending}
              onClick={handleDelete}
            >
              Удалить
            </Button>
          </>
        }
      >
        Товар «{deleting?.title}» будет удалён из каталога.
      </Modal>
    </div>
  );
}
