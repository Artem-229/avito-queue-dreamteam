import { type ReactNode } from 'react';

import styles from './PagePlaceholder.module.css';

interface PagePlaceholderProps {
  title: string;
  description: string;
  children?: ReactNode;
}

export function PagePlaceholder({
  title,
  description,
  children,
}: PagePlaceholderProps) {
  return (
    <section className={styles.root}>
      <span className={styles.tag}>Каркас</span>
      <h1 className={styles.title}>{title}</h1>
      <p className={styles.description}>{description}</p>
      {children}
    </section>
  );
}
