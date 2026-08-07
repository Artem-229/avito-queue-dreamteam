import { create } from 'zustand';

const STORAGE_KEY = 'avito-queue-user';
const DEFAULT_USER = 'user-1';

export interface DemoUser {
  id: string;
  name: string;
  emoji: string;
}

export const DEMO_USERS: DemoUser[] = [
  { id: 'user-1', name: 'Аня', emoji: '🅰️' },
  { id: 'user-2', name: 'Борис', emoji: '🅱️' },
  { id: 'user-3', name: 'Вера', emoji: '🆅' },
];

function readInitialUser(): string {
  try {
    return localStorage.getItem(STORAGE_KEY) ?? DEFAULT_USER;
  } catch {
    return DEFAULT_USER;
  }
}

function persistUser(id: string): void {
  try {
    localStorage.setItem(STORAGE_KEY, id);
  } catch {
    return;
  }
}

interface SessionState {
  userId: string;
  setUserId: (userId: string) => void;
}

export const useSessionStore = create<SessionState>((set) => ({
  userId: readInitialUser(),
  setUserId: (userId) => {
    persistUser(userId);
    set({ userId });
  },
}));

export function getUserId(): string {
  return useSessionStore.getState().userId;
}
