import { Outlet } from 'react-router-dom';

import { Toaster } from '@/shared/ui';

import { Header } from './Header';
import styles from './AppLayout.module.css';

export function AppLayout() {
  return (
    <div className={styles.root}>
      <Header />
      <main className={`container ${styles.main}`}>
        <Outlet />
      </main>
      <Toaster />
    </div>
  );
}
