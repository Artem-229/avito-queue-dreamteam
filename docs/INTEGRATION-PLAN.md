# План интеграции фронта и бэка + запуск одной командой

> **Статус: выполнен.** Сохранён как история решений: почему каждое расхождение
> контракта чинилось на той стороне, на которой чинилось. Действующий контракт —
> [../AGENTS.md](../AGENTS.md), актуальное состояние — [../CLAUDE.md](../CLAUDE.md).

Цель: `docker compose up` поднимает postgres + миграции + бэкенд + фронт, и основной сценарий
кейса проходится целиком в браузере против реального API, без моков.

---

## 1. Принцип: что чиним на фронте, что на бэке

| Расхождение | Чиним | Почему именно там |
|---|---|---|
| Имена статусов `QUEUED` vs `waiting` | **Фронт** | Имена бэка зашиты в `CHECK`-констрейнты БД, в контракт и в тексты кейса. На фронте это один `types.ts` и мапперы |
| Адресация по `entryId` vs по `itemId` | **Фронт** | Бэк спроектирован верно: агрегат — товар, запись очереди не адресуемая сущность. На фронте это ~50 строк `api.ts` |
| `snake_case` vs `camelCase` | **Фронт**, слоем DTO | Уже есть готовый паттерн — `catalogItemToItem`. JSON бэка остаётся идиоматичным |
| SSE (`EventSource`) | **Фронт**, удалить | Поллинг выбран осознанно: он же триггер ленивого `EnsureAdvanced`. На фронте fallback на поллинг уже написан |
| Отдельная ручка ETA | **Фронт**, удалить | `eta_seconds` едет в конверте статуса |
| `price` vs `price_kopecks` | **Фронт**, в DTO | Копейки — осознанное решение против float-денег |
| nginx: апстрим `backend`, срезанный префикс | **Фронт** (`nginx.conf`) | Одна строка |
| `VITE_API_MODE` по умолчанию `mock` | **Фронт** | |
| Формат ошибок `{error}` vs `{code,message}` | **Бэк** | Фронт **уже** читает `{code, message}` — это он написан правильно |
| Две ручки покупки | **Бэк** | Фронту нужен один путь; дубль всё равно удаляем (B-07) |
| Нет `/stats` для страницы метрик | **Бэк** | Дешёвые `COUNT`-запросы, плюс это отдельный балл за доп. функционал |
| Форма ответа `/demo/simulate` | **Фронт** упрощаем | Бэк отдаёт факт (`joined`/`failed`), выдуманная на моке «дистрибуция» не нужна |
| Фронта нет в compose | **Корневой `docker-compose.yaml`** | |

---

## 2. Этап 0. Блокеры бэка — предусловие

Без них интегрировать нечего: `waiting` отдаёт 500, повторный вход клинит товар.
Детали и SQL — CLAUDE.md §2. Порядок: **B-01 → B-02 → B-03**, около часа.

Проверка этапа (curl, без фронта):

```bash
ITEM=$(curl -s localhost:8080/api/v1/catalog | jq -r '.[0].id')
U1=11111111-1111-4111-8111-111111111111

curl -s -XPOST -H "X-User-Id: $U1" localhost:8080/api/v1/catalog/$ITEM/queue      | jq
curl -s        -H "X-User-Id: $U1" localhost:8080/api/v1/catalog/$ITEM/queue/me   | jq
# ожидаем 200 и status=granted|waiting, НЕ 500
```

---

## 3. Этап 1. Бэк: довести контракт под фронт

### 1.1 Единый формат ошибок (B-07) — жёсткая зависимость

`shared/api/http.ts` уже разбирает `{code, message}`. Пока бэк отдаёт `{error}`, пользователь
видит `response.statusText` вместо человеческого текста.

```go
// handlers/handler.go
type ErrorBody struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

func mapError(err error) (status int, body ErrorBody) {
    switch {
    case errors.Is(err, domain.ErrNoItemFound):
        return 404, ErrorBody{"ITEM_NOT_FOUND", "Товар не найден"}
    case errors.Is(err, domain.ErrUserAlreadyInQueue):
        return 409, ErrorBody{"ALREADY_IN_QUEUE", "Вы уже в очереди на этот товар"}
    case errors.Is(err, domain.ErrNoPurchaseRight):
        return 403, ErrorBody{"NO_ACTIVE_PURCHASE_RIGHT", "Нет активного права на покупку"}
    case errors.Is(err, domain.ErrItemSoldOut):
        return 409, ErrorBody{"ITEM_SOLD_OUT", "Товар полностью распродан"}
    case errors.Is(err, domain.ErrCannotLeaveQueue):
        return 409, ErrorBody{"CANNOT_LEAVE_QUEUE", "Из этого состояния выйти нельзя"}
    default:
        return 500, ErrorBody{"INTERNAL", "Внутренняя ошибка сервера"}
    }
}
```

`INVALID_ITEM_ID` формируется на месте `uuid.Parse` — это не доменный сентинел.

### 1.2 Один путь покупки

Удалить `POST /catalog/:id/buy` и `POST /checkout/:itemID/pay`, оставить
`POST /api/v1/catalog/:id/purchase`. Вместе с этим уезжает весь checkout-слой (CLAUDE.md §5.3 п.2)
и три красных теста.

### 1.3 `DELETE /queue/me` возвращает конверт

Сейчас `{"message": "left queue"}`. Пусть возвращает тот же `QueueStatusResponse`, что `GET` и
`POST` — фронт обновляет состояние из ответа, без второго запроса.

### 1.4 `GET /api/v1/stats` — для страницы метрик

Фронт ждёт `{totalWaiting, totalGranted, purchased, expired, conversion, avgWaitSeconds, items[]}`.
Одним запросом:

```sql
SELECT
  (SELECT count(*) FROM queue WHERE status='waiting')                        AS total_waiting,
  (SELECT count(*) FROM purchase_rights WHERE status='granted')              AS total_granted,
  (SELECT count(*) FROM purchase_rights WHERE status='used')                 AS purchased,
  (SELECT count(*) FROM purchase_rights WHERE status='expired')              AS expired;
```

`conversion` = `used / (used + expired)`, `null` при нулевом знаменателе. `avgWaitSeconds` можно
отдавать `null` — честнее, чем выдумывать (как и `eta_seconds`, см. EXTRA-FEATURES.md).
`items[]` — по товарам: `name`, `waiting`, `granted_count`, `total_stock - granted - used`.

Тут же уместен счётчик оверселлов (всегда 0) — это прямой ответ на вопрос кейса №1.

### 1.5 Решить судьбу `X-User-Id`

Фронт переключает пользователя через заголовок (`shared/lib/session.ts`, 3 фиксированных UUID).
Если приезжает подписанная сессия (B-06), нужно либо оставить `auth.allow_header = true`, либо
переводить переключатель на `POST /demo/login`. **Согласовать до мержа**, иначе переключатель
пользователей молча перестанет работать и демо в двух вкладках развалится.

---

## 4. Этап 2. Фронт: на живой бэкенд

### 2.1 Статусы и типы (`entities/queue-entry/model/types.ts`)

```ts
export type QueueStatus =
  | 'not_in_queue'
  | 'waiting'
  | 'granted'
  | 'purchased'
  | 'expired'
  | 'sold_out'
  | 'cancelled';

export interface NextStep {
  kind: 'join' | 'wait' | 'pay' | 'rejoin' | 'browse_similar' | 'done';
  label: string;
}

/** Конверт статуса — единственный источник состояния очереди. */
export interface QueueEntry {
  itemId: string;
  status: QueueStatus;
  message: string;
  nextStep: NextStep;
  position: number;
  queueSize: number;
  expiresAt: string | null;
  etaSeconds: number | null;
  serverTime: string;
  alternatives: string[];
}

const TERMINAL: readonly QueueStatus[] = ['purchased', 'sold_out', 'cancelled'];
export const isTerminalStatus = (s: QueueStatus) => TERMINAL.includes(s);
```

**`expired` больше не терминальный** — из него можно встать в очередь заново, поллинг должен
продолжаться. `LEFT` → `cancelled`, `QUEUED` → `waiting`.

### 2.2 DTO-маппер (новый `entities/queue-entry/dto.ts`)

```ts
export interface QueueStatusDto {
  status: string;
  message: string;
  position?: number;
  queue_size: number;
  expires_at: string | null;
  eta_seconds: number | null;
  server_time: string;
  next_step: { kind: string; label: string };
  alternatives: string[];
}

export function toQueueEntry(dto: QueueStatusDto, itemId: string): QueueEntry {
  return {
    itemId,
    status: dto.status as QueueStatus,
    message: dto.message,
    nextStep: dto.next_step as NextStep,
    position: dto.position ?? 0,
    queueSize: dto.queue_size,
    expiresAt: dto.expires_at,
    etaSeconds: dto.eta_seconds,
    serverTime: dto.server_time,
    alternatives: dto.alternatives ?? [],
  };
}
```

### 2.3 API очереди (`entities/queue-entry/api.ts`) — переписать целиком

```ts
const base = (itemId: string) => `/api/v1/catalog/${encodeURIComponent(itemId)}`;

export async function joinQueue(itemId: string): Promise<QueueEntry> {
  return toQueueEntry(await apiRequest<QueueStatusDto>(`${base(itemId)}/queue`, {
    method: 'POST',
  }), itemId);
}

export async function fetchStatus(itemId: string): Promise<QueueEntry> {
  return toQueueEntry(
    await apiRequest<QueueStatusDto>(`${base(itemId)}/queue/me`), itemId);
}

export async function leaveQueue(itemId: string): Promise<QueueEntry> {
  return toQueueEntry(await apiRequest<QueueStatusDto>(`${base(itemId)}/queue/me`, {
    method: 'DELETE',
  }), itemId);
}

export async function purchase(itemId: string): Promise<void> {
  await apiRequest<void>(`${base(itemId)}/purchase`, { method: 'POST' });
}
```

Удаляются: `fetchEntry`, `fetchEntryByItem`, `fetchEta`, `checkout`, `subscribeToEntry`.
Обработка 404 в `fetchEntryByItem` больше не нужна — бэк отдаёт `not_in_queue` конвертом (B-03).

### 2.4 Queries (`entities/queue-entry/queries.ts`)

Оставить один хук, ключ — `itemId`, транспорт — только поллинг:

```ts
export const queueKeys = { status: (itemId: string) => ['queue-status', itemId] as const };

export function useQueueStatus(itemId: string) {
  return useQuery({
    queryKey: queueKeys.status(itemId),
    queryFn: () => fetchStatus(itemId),
    refetchInterval: (q) =>
      q.state.data && isTerminalStatus(q.state.data.status) ? false : 1000,
  });
}
```

Удалить `useLiveEntry`, `EventSource`, переключатель `sse|polling`. Интервал — 1000 мс: он же
триггер `EnsureAdvanced` на бэке, а `NeedsReconcile` делает частый поллинг дешёвым.

### 2.5 Таймер от `server_time`, а не от часов браузера

`useCountdown` должен считать `expiresAt − serverTime` и дальше тикать локально от этой дельты.
Иначе у пользователя со сбитыми часами таймер врёт — ради этого `server_time` и добавлен.

### 2.6 Цена в копейках (`entities/item/dto.ts`, `shared/lib/formatPrice.ts`)

```ts
export interface CatalogItemDto {
  id: string; name: string;
  price_kopecks: number;
  total_stock: number; hold_ttl_seconds?: number;
  category?: string; seller_name?: string; created_at?: string;
}
// catalogItemToItem: price: dto.price_kopecks / 100
```

Либо честнее — хранить `priceKopecks` в модели и делить только в `formatPrice`.

### 2.7 Режим API по умолчанию — `live`

```ts
// shared/config.ts
export const API_MODE: ApiMode =
  import.meta.env.MODE === 'test' || import.meta.env.VITE_API_MODE === 'mock'
    ? 'mock'
    : 'live';
```

Тогда прод-сборка в Docker не требует build-аргументов (Vite запекает env на этапе сборки).

### 2.8 Удалить вторую реализацию бизнес-логики

`src/mocks/queue-engine.ts`, `queue-engine.test.ts`, `mocks/simulator.ts`, `simulator.test.ts` —
удалить целиком. MSW оставить только на статических фикстурах для юнит-тестов компонентов.
Держать две реализации правил очереди до сдачи невозможно, а расхождения вылезут на демо.

### 2.9 Симулятор — на реальные ручки

`features/simulator/api.ts` → `POST /api/v1/demo/simulate` с `{item_id, count}`, ответ
`{requested, joined, failed}`. Упростить `SimulationResult` под это. После запуска показать
`/api/v1/stats` — получается «стенд честности»: N параллельных входов, оверселлов ноль.

### 2.10 Ручка каталога уже корректна

`entities/item/api.ts` в live-режиме уже ходит в `/api/v1/catalog`, `/catalog/:id`,
`/catalog/:id/similar` — менять не нужно, только DTO цены.

### 2.11 ИИ-рекомендации — решить

`features/ai-recommendation` ходит в `/ai/recommendation`, который обслуживает `gigachat.plugin.ts`
— **это плагин dev-сервера Vite, в nginx-сборке его не существует**, в проде будет 404.
Варианты:

* **A.** Перенести прокси к GigaChat в Go (`POST /api/v1/ai/recommendation`), работает и в проде.
  Кейс отдельно отмечает интеграцию ИИ как плюс. Обязательно: без ключа — статический текст,
  без ошибок и пустых блоков; вызов только **после** COMMIT, никогда внутри транзакции (INV-7).
* **B.** Отключить фичу в прод-сборке.

Рекомендация — A, если остаётся время: это отдельные баллы, а деградация без ключа дешёвая.

---

## 5. Этап 3. Запуск одной командой

### 3.1 `frontend/nginx.conf` — две правки

```nginx
location /api/ {
    proxy_pass http://app:8080;      # без слэша в конце — префикс /api сохраняется
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}
```

* `backend` → `app`: так называется сервис в `docker-compose.yaml`.
* Убрать слэш после `:8080`: со слэшем nginx **срезает** `/api/`, и `/api/v1/catalog` уходит на
  бэк как `/v1/catalog` → 404 на всех запросах.
* Блок SSE-настроек (`proxy_buffering off`, `read_timeout 3600s`) удалить — SSE не используется.

### 3.2 Сервис фронта в `docker-compose.yaml`

```yaml
  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    container_name: avito-queue-frontend
    restart: unless-stopped
    ports:
      - "3000:80"
    depends_on:
      app:
        condition: service_started
```

Единая точка входа для жюри — `http://localhost:3000`. Бэкенд на `8080` можно оставить
опубликованным для curl-проверок из README.

### 3.3 `.gitignore` в корне

Сейчас пустой, при этом в дереве лежит `main.exe`. Минимально:

```
main.exe
server
/frontend/node_modules/
/frontend/dist/
*.tsbuildinfo
.env.local
```

### 3.4 Проверка

```bash
docker compose up --build -d
curl -s localhost:8080/health
curl -s localhost:3000/api/v1/catalog | jq '.[0]'   # проксирование живое
open http://localhost:3000
```

---

## 6. Порядок работ

```
Этап 0: B-01, B-02, B-03            (бэк, ~1 ч)   ← без этого фронт нечем кормить
   ↓
Этап 1: формат ошибок, один путь покупки, конверт в DELETE, /stats   (бэк, ~2,5 ч)
   ↓                                    ↘  можно параллельно
Этап 2: типы, DTO, api, queries, таймер, цена, удаление моков  (фронт, ~4 ч)
   ↓
Этап 3: nginx + compose + .gitignore   (~40 мин)
   ↓
ГЕЙТ: два браузера против одного docker compose up.
      Первый получает право, второй видит позицию,
      по истечении права слот уходит второму.
```

Этапы 1 и 2 частично параллелятся: фронт может писаться под согласованный контракт, пока бэк
его доводит. Не параллелится только этап 0 — он меняет то, что фронт будет дёргать каждую секунду.

---

## 7. Что нужно решить до старта

1. **`status` или `state`** в конверте. Код отдаёт `status`, AGENTS.md обещает `state`.
   Дешевле оставить `status` и поправить AGENTS.md.
2. **Судьба `X-User-Id`** при появлении подписанной сессии (см. 1.5) — иначе сломается
   переключатель пользователей.
3. **ИИ-рекомендации**: перенести на Go или отключить в проде (см. 2.11).
4. **Страница метрик**: делаем `/stats` или временно убираем страницу из роутера.
   Полумера «страница есть, ручки нет» хуже обоих вариантов.
