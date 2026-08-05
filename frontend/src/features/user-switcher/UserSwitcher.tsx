import { useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';

import { DEMO_USERS, useSessionStore } from '@/shared/lib/session';

import styles from './UserSwitcher.module.css';

export function UserSwitcher() {
  const userId = useSessionStore((state) => state.userId);
  const setUserId = useSessionStore((state) => state.setUserId);
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  const handleSelect = (id: string) => {
    if (id === userId) return;
    setUserId(id);
    queryClient.clear();
    navigate('/');
  };

  return (
    <div className={styles.switcher}>
      <span className={styles.label}>Демо-пользователь</span>
      <div className={styles.segment} role="group">
        {DEMO_USERS.map((user) => (
          <button
            key={user.id}
            type="button"
            className={`${styles.option} ${
              user.id === userId ? styles.active : ''
            }`}
            onClick={() => {
              handleSelect(user.id);
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
