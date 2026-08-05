import { create } from 'zustand';

export type ToastTone = 'blue' | 'green' | 'red' | 'amber';

export interface ToastItem {
  id: string;
  message: string;
  tone: ToastTone;
}

interface ToastState {
  toasts: ToastItem[];
  push: (message: string, tone?: ToastTone) => void;
  dismiss: (id: string) => void;
}

export const useToastStore = create<ToastState>((set) => ({
  toasts: [],
  push: (message, tone = 'blue') =>
    set((state) => ({
      toasts: [
        ...state.toasts,
        { id: `${String(Date.now())}-${String(Math.random())}`, message, tone },
      ],
    })),
  dismiss: (id) =>
    set((state) => ({
      toasts: state.toasts.filter((toast) => toast.id !== id),
    })),
}));

export function showToast(message: string, tone: ToastTone = 'blue'): void {
  useToastStore.getState().push(message, tone);
}
