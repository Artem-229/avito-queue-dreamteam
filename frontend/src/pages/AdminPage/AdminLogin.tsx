import { type FormEvent, useState } from 'react';

import { useAdminLogin } from '@/features/admin';
import { ApiError } from '@/shared/api';
import { Button, Card } from '@/shared/ui';

import styles from './AdminPage.module.css';

export function AdminLogin() {
  const [key, setKey] = useState('');
  const login = useAdminLogin();

  const errorMessage =
    login.error instanceof ApiError && login.error.status === 403
      ? 'Неверный ключ администратора'
      : login.error
        ? 'Не удалось выполнить вход'
        : null;

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    login.mutate(key);
  };

  return (
    <div className={styles.center}>
      <Card padding="lg" className={styles.loginCard}>
        <h1 className={styles.title}>Админ-панель</h1>
        <p className={styles.subtitle}>
          Введите ключ администратора, чтобы управлять товарами и смотреть
          метрики по очередям.
        </p>
        <form className={styles.loginForm} onSubmit={handleSubmit}>
          <input
            className={styles.input}
            type="password"
            placeholder="Ключ администратора"
            value={key}
            autoFocus
            onChange={(event) => {
              setKey(event.target.value);
            }}
          />
          {errorMessage && <span className={styles.error}>{errorMessage}</span>}
          <Button
            type="submit"
            size="lg"
            fullWidth
            loading={login.isPending}
            disabled={key.trim() === ''}
          >
            Войти
          </Button>
        </form>
      </Card>
    </div>
  );
}
