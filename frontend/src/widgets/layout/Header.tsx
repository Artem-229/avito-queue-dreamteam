import { Link } from 'react-router-dom';

import { ROUTES } from '@/app/routes';
import { UserSwitcher } from '@/features/user-switcher';

import { Logo } from './Logo';
import styles from './Header.module.css';

export function Header() {
  return (
    <header className={styles.header}>
      <div className={`container ${styles.inner}`}>
        <Link to={ROUTES.catalog} className={styles.logoLink} aria-label="На главную">
          <Logo />
        </Link>

        <div className={styles.right}>
          <UserSwitcher />
          {import.meta.env.DEV && (
            <Link to="/ui-kit" className={styles.devLink}>
              UI-kit
            </Link>
          )}
          <span className={styles.badge}>MVP</span>
        </div>
      </div>
    </header>
  );
}
