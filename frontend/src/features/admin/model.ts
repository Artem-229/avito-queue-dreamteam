import { create } from 'zustand';

interface AdminState {
  key: string | null;
  setKey: (key: string) => void;
  clear: () => void;
}

export const useAdminStore = create<AdminState>((set) => ({
  key: null,
  setKey: (key) => {
    set({ key });
  },
  clear: () => {
    set({ key: null });
  },
}));

export function getAdminKey(): string | null {
  return useAdminStore.getState().key;
}
