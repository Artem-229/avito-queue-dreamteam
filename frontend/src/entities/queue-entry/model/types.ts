/**
 * Состояния очереди. Имена совпадают с бэкендом буква в букву — это значения
 * queue.status в Postgres, плюс виртуальное not_in_queue, которого в таблице нет:
 * бэкенд отдаёт его конвертом, чтобы у каждого состояния были message и next_step.
 */
import { type PurchaseChance } from '@/shared/lib/chance';

export type { PurchaseChance };

export type QueueStatus =
  | 'not_in_queue'
  | 'waiting'
  | 'granted'
  | 'purchased'
  | 'expired'
  | 'sold_out'
  | 'cancelled';

export type NextStepKind =
  | 'join'
  | 'wait'
  | 'pay'
  | 'rejoin'
  | 'browse_similar'
  | 'done';

export interface NextStep {
  kind: NextStepKind;
  label: string;
}

/**
 * Конверт статуса — единственный источник состояния очереди на клиенте.
 * Бизнес-логики состояний на фронте нет: message и nextStep приходят с бэкенда,
 * чтобы тексты не разъезжались между двумя реализациями.
 */
export interface QueueEntry {
  itemId: string;
  status: QueueStatus;
  message: string;
  nextStep: NextStep;
  /** Позиция в очереди, 1 — следующий на выдачу права. 0 вне статуса waiting. */
  position: number;
  /** Позиция на момент входа. null у записей до появления колонки. */
  initialPosition: number | null;
  /** Прогресс продвижения очереди 0..100, посчитан бэкендом. null, если недоступен. */
  progressPercent: number | null;
  /** Шанс, что очередь дойдёт до пользователя. null вне ожидания. */
  chance: PurchaseChance | null;
  queueSize: number;
  expiresAt: string | null;
  /** Всегда null: честная оценка требует данных о конверсии права в покупку. */
  etaSeconds: number | null;
  /** Время БД. Таймер считается от него, а не от часов браузера. */
  serverTime: string;
  alternatives: string[];
}

/**
 * Терминальные состояния — те, из которых опрос статуса больше ничего не узнает.
 * expired сюда НЕ входит: из него можно встать в очередь заново, и поллинг
 * должен продолжаться, чтобы показать результат повторного входа.
 */
const TERMINAL_STATUSES: readonly QueueStatus[] = [
  'purchased',
  'sold_out',
  'cancelled',
];

export function isActiveRight(entry: QueueEntry): boolean {
  return entry.status === 'granted';
}

export function isTerminalStatus(status: QueueStatus): boolean {
  return TERMINAL_STATUSES.includes(status);
}

/** Стоит ли пользователь в очереди прямо сейчас (ждёт или держит право). */
export function isInQueue(status: QueueStatus): boolean {
  return status === 'waiting' || status === 'granted';
}
