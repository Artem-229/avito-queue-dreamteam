import styles from './Logo.module.css';

export function Logo() {
  return (
    <span className={styles.logo}>
      <span className={styles.mark} aria-hidden="true">
        <span className={styles.dot} />
      </span>
      <span className={styles.word}>
        Авито<span className={styles.accent}> Очередь</span>
      </span>
    </span>
  );
}
