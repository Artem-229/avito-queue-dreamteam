import { create } from 'zustand';

import { ApiError } from '@/shared/api/ApiError';

/**
 * Личность пользователя целиком на стороне бэкенда: user_id генерирует и
 * подписывает сервер (POST /api/v1/demo/login), клиент его не выбирает.
 * Раньше здесь лежали захардкоженные UUID, уезжавшие в X-User-Id, — с таким
 * заголовком любой клиент назывался кем угодно и забирал чужое право на
 * покупку. Бэкенд этот путь закрыл, фронт следует за ним.
 */
const STORAGE_KEY = 'avito-queue-session';
const LOGIN_PATH = '/api/v1/demo/login';

export interface DemoUser {
  name: string;
  emoji: string;
}

/**
 * Демо-аккаунты различаются только именем: id выдаёт сервер при входе.
 * Имя — ключ, по которому мы помним выданный токен между перезагрузками,
 * чтобы «Аня» после F5 осталась той же Аней со своей позицией в очереди.
 */
export const DEMO_USERS: DemoUser[] = [
  { name: 'Аня', emoji: '🅰️' },
  { name: 'Борис', emoji: '🅱️' },
  { name: 'Вера', emoji: '🆅' },
];

const DEFAULT_NAME = DEMO_USERS[0].name;

/** Выданная сервером сессия: подписанный токен и id, зашитый в его подпись. */
export interface Account {
  userId: string;
  token: string;
}

interface PersistedSession {
  activeName: string;
  accounts: Record<string, Account>;
}

function isAccount(value: unknown): value is Account {
  if (typeof value !== 'object' || value === null) return false;
  const candidate = value as Partial<Account>;
  return (
    typeof candidate.userId === 'string' && typeof candidate.token === 'string'
  );
}

function readPersisted(): PersistedSession {
  const empty: PersistedSession = { activeName: DEFAULT_NAME, accounts: {} };

  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return empty;

    const parsed: unknown = JSON.parse(raw);
    if (typeof parsed !== 'object' || parsed === null) return empty;

    const candidate = parsed as Partial<PersistedSession>;
    const accounts: Record<string, Account> = {};

    for (const [name, account] of Object.entries(candidate.accounts ?? {})) {
      if (isAccount(account)) {
        accounts[name] = account;
      }
    }

    const activeName = DEMO_USERS.some(
      (user) => user.name === candidate.activeName,
    )
      ? (candidate.activeName ?? DEFAULT_NAME)
      : DEFAULT_NAME;

    return { activeName, accounts };
  } catch {
    return empty;
  }
}

function persist(state: PersistedSession): void {
  try {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({
        activeName: state.activeName,
        accounts: state.accounts,
      }),
    );
  } catch {
    // Приватный режим браузера: сессия проживёт до перезагрузки, и это
    // единственное последствие — падать из-за недоступного хранилища незачем.
  }
}

interface SessionState extends PersistedSession {
  setActiveName: (name: string) => void;
  setAccount: (name: string, account: Account) => void;
  dropAccount: (name: string) => void;
}

export const useSessionStore = create<SessionState>((set, get) => ({
  ...readPersisted(),

  setActiveName: (activeName) => {
    set({ activeName });
    persist({ activeName, accounts: get().accounts });
  },

  setAccount: (name, account) => {
    const accounts = { ...get().accounts, [name]: account };
    set({ accounts });
    persist({ activeName: get().activeName, accounts });
  },

  dropAccount: (name) => {
    const accounts = { ...get().accounts };
    delete accounts[name];
    set({ accounts });
    persist({ activeName: get().activeName, accounts });
  },
}));

interface LoginResponse {
  user_id: string;
  token: string;
}

/**
 * Вход намеренно ходит голым fetch, а не через apiRequest: apiRequest сам
 * требует сессию, и вызов друг друга замкнул бы модули в цикл.
 */
async function requestLogin(displayName: string): Promise<Account> {
  const response = await fetch(LOGIN_PATH, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ display_name: displayName }),
  });

  if (!response.ok) {
    throw new ApiError(
      'Не удалось создать сессию демо-пользователя',
      'LOGIN_FAILED',
      response.status,
    );
  }

  const body = (await response.json()) as LoginResponse;
  return { userId: body.user_id, token: body.token };
}

/**
 * Логины в полёте, по одному на имя: при первой загрузке страницы несколько
 * запросов стартуют одновременно, и без этой защёлки каждый завёл бы свою
 * сессию — пользователь получил бы три разных user_id и потерял очередь.
 */
const pendingLogins = new Map<string, Promise<Account>>();

export function getActiveName(): string {
  return useSessionStore.getState().activeName;
}

/** Токен активного аккаунта; при отсутствии — вход и запоминание выданного. */
export async function ensureSession(
  name: string = getActiveName(),
): Promise<Account> {
  const known = useSessionStore.getState().accounts[name];
  if (known) return known;

  const inFlight = pendingLogins.get(name);
  if (inFlight) return inFlight;

  const login = requestLogin(name)
    .then((account) => {
      useSessionStore.getState().setAccount(name, account);
      return account;
    })
    .finally(() => {
      pendingLogins.delete(name);
    });

  pendingLogins.set(name, login);
  return login;
}

/**
 * Перевыпуск сессии после 401: токен живёт сутки, а стенд переживает и больше.
 * Новый вход даёт новый user_id — прошлая очередь этого имени остаётся на
 * бэкенде ничьей, но альтернатива (молча показывать экран «требуется вход»
 * без формы входа) хуже.
 *
 * staleToken — тот, на котором прилетел 401. Страница очереди поллится раз в
 * секунду параллельно с другими запросами, и по истечении сессии 401 приходит
 * сразу на несколько: без этой проверки второй ответ выбросил бы токен,
 * который первый только что перевыпустил, и вход пошёл бы по кругу.
 */
export async function refreshSession(
  staleToken?: string,
  name: string = getActiveName(),
): Promise<Account> {
  const current = useSessionStore.getState().accounts[name];
  if (staleToken !== undefined && current && current.token !== staleToken) {
    return current;
  }

  pendingLogins.delete(name);
  useSessionStore.getState().dropAccount(name);
  return ensureSession(name);
}
