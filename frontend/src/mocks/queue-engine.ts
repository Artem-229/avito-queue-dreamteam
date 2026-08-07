import { type Item } from '@/entities/item';
import {
  type EtaResult,
  type Order,
  type QueueEntry,
} from '@/entities/queue-entry';

import { ITEM_SEEDS, type ItemSeed, toItem } from './data';

const USER_GRANT_TTL_MS = 120_000;
const ETA_SERVICE_SECONDS = 6;

type InternalStatus =
  | 'WAITING'
  | 'GRANTED'
  | 'EXPIRED'
  | 'PURCHASED'
  | 'SOLD_OUT'
  | 'LEFT';

interface InternalEntry {
  entryId: string;
  userId: string;
  itemId: string;
  isBot: boolean;
  status: InternalStatus;
  joinedAt: number;
  grantedAt?: number;
  expiresAt?: number;
}

interface ItemState {
  seed: ItemSeed;
  stock: number;
  entries: InternalEntry[];
  seeded: boolean;
  nextFreeAt: number;
}

export class ApiFault extends Error {
  readonly code: string;
  readonly status: number;

  constructor(code: string, message: string, status: number) {
    super(message);
    this.code = code;
    this.status = status;
  }
}

let counter = 0;
function nextId(prefix: string): string {
  counter += 1;
  return `${prefix}-${String(counter)}`;
}

function isLive(entry: InternalEntry): boolean {
  return entry.status === 'WAITING' || entry.status === 'GRANTED';
}

export class QueueEngine {
  private items = new Map<string, ItemState>();

  constructor() {
    this.reset();
  }

  reset(): void {
    counter = 0;
    this.items = new Map();
    for (const seed of ITEM_SEEDS) {
      this.items.set(seed.id, {
        seed,
        stock: seed.stock,
        entries: [],
        seeded: false,
        nextFreeAt: 0,
      });
    }
  }

  listItems(): Item[] {
    return ITEM_SEEDS.map((seed) => {
      const state = this.items.get(seed.id);
      return { ...toItem(seed), stock: state?.stock ?? seed.stock };
    });
  }

  getItem(id: string): Item {
    const state = this.requireItem(id);
    return { ...toItem(state.seed), stock: state.stock };
  }

  getSimilarItems(id: string): Item[] {
    const current = this.requireItem(id).seed;
    const items = this.listItems().filter((item) => item.id !== id);
    const sameCategory = items.filter(
      (item) => item.category === current.category,
    );
    const rest = items.filter((item) => item.category !== current.category);
    return [...sameCategory, ...rest].slice(0, 4);
  }

  join(userId: string, itemId: string): QueueEntry {
    const state = this.ensureItem(itemId);

    if (!state.seed.queueEnabled) {
      throw new ApiFault(
        'QUEUE_DISABLED',
        'Для этого товара очередь не действует',
        400,
      );
    }

    const existing = state.entries.find(
      (entry) => entry.userId === userId && isLive(entry),
    );
    if (existing) {
      return this.toView(state, existing);
    }

    if (!state.seeded) {
      state.seeded = true;
      for (let i = 0; i < state.seed.botsAhead; i += 1) {
        state.entries.push({
          entryId: nextId('bot'),
          userId: nextId('bot-user'),
          itemId,
          isBot: true,
          status: 'WAITING',
          joinedAt: Date.now(),
        });
      }
    }

    const entry: InternalEntry = {
      entryId: nextId('entry'),
      userId,
      itemId,
      isBot: false,
      status: 'WAITING',
      joinedAt: Date.now(),
    };
    state.entries.push(entry);

    this.tick(state);
    return this.toView(state, entry);
  }

  getEntry(entryId: string): QueueEntry {
    const found = this.findEntry(entryId);
    this.tick(found.state);
    return this.toView(found.state, found.entry);
  }

  getEta(entryId: string): EtaResult {
    const found = this.findEntry(entryId);
    this.tick(found.state);

    const live = found.state.entries.filter(isLive);
    const index = live.indexOf(found.entry);
    const ahead = index >= 0 ? index : 0;

    const perPerson =
      found.state.seed.botServiceMs > 0
        ? found.state.seed.botServiceMs / 1000
        : ETA_SERVICE_SECONDS;

    const seconds = Math.round(ahead * perPerson);
    const confidence: EtaResult['confidence'] =
      ahead <= 1 ? 'high' : ahead <= 4 ? 'medium' : 'low';

    return { seconds, confidence };
  }

  getEntryByItem(userId: string, itemId: string): QueueEntry | null {
    const state = this.requireItem(itemId);
    this.tick(state);
    const entry = [...state.entries]
      .reverse()
      .find((candidate) => candidate.userId === userId);
    return entry ? this.toView(state, entry) : null;
  }

  leave(entryId: string): void {
    const { state, entry } = this.findEntry(entryId);
    if (isLive(entry)) {
      entry.status = 'LEFT';
    }
    this.tick(state);
  }

  checkout(userId: string, entryId: string): Order {
    const { state, entry } = this.findEntry(entryId);
    this.tick(state);

    if (entry.userId !== userId) {
      throw new ApiFault(
        'NO_ACTIVE_RIGHT',
        'Право на покупку принадлежит другому пользователю',
        403,
      );
    }
    if (entry.status === 'EXPIRED') {
      throw new ApiFault('RIGHT_EXPIRED', 'Время на покупку истекло', 409);
    }
    if (entry.status === 'SOLD_OUT' || state.stock <= 0) {
      throw new ApiFault('ITEM_SOLD_OUT', 'Товар закончился', 409);
    }
    if (entry.status !== 'GRANTED') {
      throw new ApiFault(
        'NO_ACTIVE_RIGHT',
        'Нет активного права на покупку',
        409,
      );
    }

    state.stock -= 1;
    entry.status = 'PURCHASED';

    return {
      orderId: nextId('order'),
      itemId: state.seed.id,
      createdAt: new Date().toISOString(),
    };
  }

  private tick(state: ItemState): void {
    const now = Date.now();

    for (;;) {
      const head = state.entries.find(isLive);
      if (!head) break;

      if (state.stock <= 0) {
        for (const entry of state.entries) {
          if (entry.status === 'WAITING') entry.status = 'SOLD_OUT';
        }
        break;
      }

      if (head.status === 'WAITING') {
        const serviceMs = head.isBot
          ? state.seed.botServiceMs
          : USER_GRANT_TTL_MS;
        const begin = Math.max(state.nextFreeAt, head.joinedAt);
        head.status = 'GRANTED';
        head.grantedAt = begin;
        head.expiresAt = begin + serviceMs;
        state.nextFreeAt = head.expiresAt;
      }

      const expired = head.expiresAt !== undefined && now >= head.expiresAt;

      if (head.isBot) {
        if (expired) {
          head.status = 'PURCHASED';
          state.stock -= 1;
          continue;
        }
        break;
      }

      if (expired) {
        head.status = 'EXPIRED';
        continue;
      }
      break;
    }
  }

  private toView(state: ItemState, entry: InternalEntry): QueueEntry {
    const live = state.entries.filter(isLive);
    const index = live.indexOf(entry);
    const ahead = index >= 0 ? index : 0;

    const status =
      entry.status === 'WAITING' ? 'QUEUED' : (entry.status as QueueEntry['status']);

    const view: QueueEntry = {
      entryId: entry.entryId,
      itemId: entry.itemId,
      userId: entry.userId,
      status,
      position: ahead,
      totalAhead: ahead,
    };

    if (entry.grantedAt !== undefined) {
      view.grantedAt = new Date(entry.grantedAt).toISOString();
    }
    if (entry.expiresAt !== undefined && status === 'GRANTED') {
      view.expiresAt = new Date(entry.expiresAt).toISOString();
    }
    if (status === 'QUEUED') {
      view.etaSeconds = Math.max(1, ahead) * ETA_SERVICE_SECONDS;
    }

    return view;
  }

  private requireItem(id: string): ItemState {
    const state = this.items.get(id);
    if (!state) {
      throw new ApiFault('NOT_FOUND', 'Товар не найден', 404);
    }
    return state;
  }

  private ensureItem(id: string): ItemState {
    const existing = this.items.get(id);
    if (existing) return existing;

    const seed: ItemSeed = {
      id,
      title: 'Товар',
      price: 0,
      category: '',
      sellerName: '',
      stock: 5,
      queueEnabled: true,
      emoji: '📦',
      accent: '#00AAFF',
      botsAhead: 2,
      botServiceMs: 4000,
    };
    const state: ItemState = {
      seed,
      stock: seed.stock,
      entries: [],
      seeded: false,
      nextFreeAt: 0,
    };
    this.items.set(id, state);
    return state;
  }

  private findEntry(entryId: string): { state: ItemState; entry: InternalEntry } {
    for (const state of this.items.values()) {
      const entry = state.entries.find((item) => item.entryId === entryId);
      if (entry) return { state, entry };
    }
    throw new ApiFault('NOT_FOUND', 'Запись очереди не найдена', 404);
  }
}

export const queueEngine = new QueueEngine();
