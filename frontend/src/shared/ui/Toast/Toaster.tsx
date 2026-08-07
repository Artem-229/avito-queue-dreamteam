import { useEffect } from 'react';

import { createPortal } from 'react-dom';

import { type ToastItem, useToastStore } from './toastStore';
import styles from './Toaster.module.css';

function ToastCard({ toast }: { toast: ToastItem }) {
  const dismiss = useToastStore((state) => state.dismiss);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      dismiss(toast.id);
    }, 3500);

    return () => {
      window.clearTimeout(timer);
    };
  }, [toast.id, dismiss]);

  return (
    <div className={`${styles.toast} ${styles[toast.tone]}`} role="status">
      {toast.message}
    </div>
  );
}

export function Toaster() {
  const toasts = useToastStore((state) => state.toasts);

  if (toasts.length === 0) return null;

  return createPortal(
    <div className={styles.stack}>
      {toasts.map((toast) => (
        <ToastCard key={toast.id} toast={toast} />
      ))}
    </div>,
    document.body,
  );
}
