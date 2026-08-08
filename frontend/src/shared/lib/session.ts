import { create } from 'zustand';

const STORAGE_KEY = 'avito-queue-user';

export interface DemoUser {
  id: string;
  name: string;
  emoji: string;
}

export const DEMO_USERS: DemoUser[] = [
  { id: '11111111-1111-4111-8111-111111111111', name: 'Аня', emoji: '🅰️' },
  { id: '22222222-2222-4222-8222-222222222222', name: 'Борис', emoji: '🅱️' },
  { id: '33333333-3333-4333-8333-333333333333', name: 'Вера', emoji: '🆅' },
];

const DEFAULT_USER = DEMO_USERS[0].id;

const UUID_RE =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

function readInitialUser(): string {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    return stored && UUID_RE.test(stored) ? stored : DEFAULT_USER;
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
