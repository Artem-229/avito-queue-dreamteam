import { useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';

import { DEMO_USERS, ensureSession, useSessionStore } from '@/shared/lib/session';

import styles from './UserSwitcher.module.css';

export function UserSwitcher() {
  const activeName = useSessionStore((state) => state.activeName);
  const setActiveName = useSessionStore((state) => state.setActiveName);
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  const handleSelect = (name: string) => {
    if (name === activeName) return;

    setActiveName(name);
    // Сессию выданного имени греем заранее, чтобы первый же запрос после
    // переключения ушёл уже с нужным токеном, а не выяснял личность на лету.
    void ensureSession(name);
    queryClient.clear();
    navigate('/');
  };

  return (
    <div className={styles.switcher}>
      <span className={styles.label}>Демо-пользователь</span>
      <div className={styles.segment} role="group">
        {DEMO_USERS.map((user) => (
          <button
            key={user.name}
            type="button"
            className={`${styles.option} ${
              user.name === activeName ? styles.active : ''
            }`}
            onClick={() => {
              handleSelect(user.name);
            }}
            title={`Переключиться на: ${user.name}`}
          >
            <span aria-hidden="true">{user.emoji}</span>
            <span className={styles.name}>{user.name}</span>
          </button>
        ))}
      </div>
    </div>
  );
}
