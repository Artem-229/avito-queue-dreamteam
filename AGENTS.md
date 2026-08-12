# AGENTS.md

Справочник по репозиторию для агентов и разработчиков. Держим коротким и нормативным: длинный
AGENTS.md увеличивает риск галлюцинаций и расхождений между агентами разных участников команды.

* **Инварианты и текущее состояние** — [CLAUDE.md](./CLAUDE.md).
* **Требования** — `docs/REQUIREMENTS.md` + `REQUIREMENTS-PATCH.md`.
* **Для человека: запуск, API, ключевые решения** — [README.md](./README.md).

## О проекте

MVP сервиса пользовательской очереди на дефицитные товары (Avito Start, кейс 2, «Dream Team»).
Пользователь встаёт в очередь перед чекаутом и получает временное персональное право на покупку.
Система гарантирует, что одна единица товара не уйдёт двум покупателям.

## Стек

* **Backend:** Go 1.25, gin, pgx/v5, goose (миграции — отдельный контейнер `migrator`), viper
* **Frontend:** React + TypeScript + Vite, feature-sliced (`frontend/`)
* **БД:** PostgreSQL 17 — единственное хранилище, без Redis и брокеров (NFR-03/NFR-04)
* **Запуск:** Docker Compose (postgres + migrator + app + frontend)

## Структура backend

```
/cmd                            — точка входа
/internal
  /app                          — сборка зависимостей, старт/остановка
  /config                       — configs/configuration.yaml через viper + env-переменные
  /domain                       — доменные типы и сентинел-ошибки, без внешних зависимостей
  /infra/http/rest              — gin: server.go, router.go, handlers/, middlewares/, session/
  /clients/gigachat             — внешний клиент эмбеддингов, вне горячего пути (INV-7)
  /infra/postgres/repository    — репозитории (pgx), транзакции, db(ctx, pool)
  /services                     — CatalogService, QueueEntryService (Reconcile/Buy),
                                  ChanceCalculator, DemoService, StatsService
                                  + интеграционные тесты
/migrations                     — goose, нумерация 00001, 00002, ...
/docs                           — REQUIREMENTS.md и производные
```

Правило зависимостей: `repository → services → handlers`, `domain` ни от чего не зависит.
Интерфейсы репозиториев объявлены на стороне потребителя (в `services`). Хендлеры обращаются
только к сервисам, никогда к репозиториям напрямую.

## Инварианты

Нормативный список — [CLAUDE.md](./CLAUDE.md) §1 (INV-1…INV-11). Краткая шпаргалка паттернов,
которые при ревью означают дефект:

| Что видно в коде | Нарушение |
|---|---|
| `r.pool.Exec(` / `r.pool.Query(` / `r.pool.QueryRow(` в `repository/` | INV-4 (ловит `forbidigo`) |
| Мутация прав или очереди без предшествующего `LockItem` в той же tx | INV-3 |
| `time.Now()` при вычислении `expires_at` или `created_at` очереди | INV-6 |
| TTL-литерал в Go-коде вместо `catalog_items.hold_ttl_seconds` | INV-6 |
| `ORDER BY created_at` без тайбрейкера `user_id` в выборках очереди | INV-10 |
| `total_stock ==` / `count ==` при проверке распроданности | INV-9 |
| `UPDATE queue_entries SET status` без обновления счётчиков тем же оператором | INV-5 |
| Арифметика `granted_count`/`used_count` отдельным вызовом из Go | INV-5 |
| `UPDATE ... SET status` без фильтра по исходному статусу | INV-11 |
| Новое состояние в ответе API без `message`/`next_step` | INV-8 |
| Вызов LLM, `http.Client` или `time.Sleep` между `InTx` и COMMIT | INV-7 |
| Импорт `pgx`, `gin` или `net/http` в `internal/domain` | правило зависимостей |
| Новая ручка покупки помимо `POST /catalog/:id/purchase` | INV-2 |
| Бизнес-логика состояний очереди на фронте | контракт: истина только на бэке |

## Домен

* **CatalogItem** — `id, name, price_kopecks, total_stock, granted_count, used_count,
  hold_ttl_seconds, category, seller_name, embedding, created_at, deleted_at`.
  `CHECK (granted_count + used_count <= total_stock)` — последняя линия защиты от оверселла.
* **QueueEntry** — `id, user_id, item_id,
  status(waiting|granted|purchased|expired|sold_out|cancelled),
  created_at, granted_at, expires_at, resolved_at, initial_position`.
  Партиальный `UNIQUE (user_id, item_id) WHERE status IN ('waiting','granted')`.

Одно участие — одна строка, от входа в очередь до финального исхода. Отдельной сущности
«право на покупку» в хранилище нет: право — это участие в статусе `granted` с непросроченным
`expires_at`. Раньше состояние жило в двух таблицах плюс счётчиках товара и синхронизировалось
вручную; таблицы слиты миграцией `00006`.

Счётчики товара — производные данные. Их обновляет тот же оператор, который меняет статус
участия (CTE `counters` в `queue_entries.go`), дельты приходят из `counterDeltas(from, to)` —
единственного места, где записано, что `granted` удерживает единицу, а `purchased` её
расходует. Отдельного вызова «поправить счётчик» в Go не существует и заводить его нельзя.

Записи не удаляются, меняется только `status` — история состояний остаётся. Повторный вход
(после `expired`/`cancelled`/`sold_out`/`purchased`) создаёт новую запись; актуальной считается
последняя по `created_at`.

## HTTP API (действующий контракт)

Все таймстемпы RFC3339, все id — UUID. Ошибки — единый формат
`{"code":"MACHINE_CODE","message":"текст для человека"}`. Полный список кодов — README «Ошибки»;
основные: `UNAUTHORIZED`, `INVALID_SESSION`, `INVALID_ITEM_ID`, `ITEM_NOT_FOUND`,
`NO_PURCHASE_RIGHT`, `ITEM_SOLD_OUT`, `ALREADY_IN_QUEUE`, `NOT_IN_QUEUE`, `CANNOT_LEAVE_QUEUE`,
`STATUS_CONFLICT`, `INTERNAL_SERVER_ERROR`.

Авторизация — подписанный токен из `POST /api/v1/demo/login`: httpOnly-кука `session` **или**
заголовок `Authorization: Bearer <token>` (заголовок приоритетнее — так работает переключатель
демо-аккаунтов). Неподписанной идентификации не существует.

| Метод | Путь | Назначение |
|---|---|---|
| `GET` | `/health` | без авторизации, пингует БД |
| `POST` | `/api/v1/demo/login` | выдать подписанную демо-сессию, без авторизации |
| `GET` | `/api/v1/catalog` | список товаров |
| `GET` | `/api/v1/catalog/:id` | товар |
| `GET` | `/api/v1/catalog/:id/similar` | похожие лоты по векторной близости, только нераспроданные |
| `POST` | `/api/v1/catalog/:id/queue` | встать в очередь, идемпотентно; возвращает конверт статуса |
| `GET` | `/api/v1/catalog/:id/queue/me` | статус и позиция; внутри `EnsureAdvanced`; поллится раз в секунду |
| `DELETE` | `/api/v1/catalog/:id/queue/me` | выйти из очереди; активное право отзывается |
| `POST` | `/api/v1/catalog/:id/purchase` | использовать право — единственный путь покупки |
| `GET` | `/api/v1/stats` | метрики: очереди по товарам, конверсия права в покупку, среднее ожидание |
| `POST` | `/api/v1/demo/items/:id/expire-now` | демо: сжечь выданные права (только при `DEMO_ENABLED`) |
| `POST` | `/api/v1/demo/simulate` | демо: N конкурентных входов, 1–100 (только при `DEMO_ENABLED`) |

Конверт статуса — единый для POST/GET/DELETE ручек очереди, поле `status` принимает
`not_in_queue | waiting | granted | purchased | expired | sold_out | cancelled`:

```json
{
  "status": "waiting", "position": 7, "initial_position": 12, "progress_percent": 45,
  "queue_size": 23, "eta_seconds": null, "chance": { "percent": 61, "basis": "item" },
  "expires_at": null, "server_time": "2026-08-09T12:32:56Z",
  "message": "Вы в очереди. Пожалуйста, подождите освобождения слота.",
  "next_step": { "kind": "wait", "label": "Ожидайте, статус обновится сам" },
  "alternatives": []
}
```

* `server_time` обязателен: таймер на фронте считается как `expires_at − server_time` с
  поправкой на часы браузера, а не по локальным часам.
* `eta_seconds` всегда `null` — осознанно (см. CLAUDE.md §3). Честный ответ на тот же вопрос —
  `chance`.
* `chance` приходит только для `waiting`. `percent` — вероятность того, что очередь дойдёт,
  `basis` — на чём посчитано: `item` (конверсия этого товара), `global` (общая по системе),
  `default` (данных ещё нет). Клиент обязан различать: выдавать дефолт за измеренное нельзя.
* `initial_position` и `progress_percent` — полоса продвижения очереди. Отсутствуют, когда
  рисовать её не на чем (вошёл первым, либо запись создана до появления колонки).
* `alternatives` (id похожих нераспроданных лотов) заполняется только для `sold_out` и `expired`.
* Имена состояний совпадают на бэкенде и фронте буква в букву.

## Как запустить

```bash
docker compose up -d --build   # postgres + migrator + app + frontend
curl http://localhost:8080/health
```

Запуск бэкенда с хоста без Docker и запуск тестов — README «Быстрый старт» и «Тесты»
(наружу Postgres проброшен на **5433**, только на `127.0.0.1`).

## Тесты и линтеры

```bash
go build ./cmd/... ./internal/... && go vet ./internal/...
TEST_DATABASE_DSN="postgres://postgres:postgres@localhost:5433/avito_queue?sslmode=disable" \
  go test -race -count=1 ./internal/...
golangci-lint run              # .golangci.yaml; forbidigo охраняет INV-4
cd frontend && npm run lint && npm run typecheck && npm run test
```

CI (`.github/workflows/lint.yml`): golangci-lint, бэкенд-тесты против Postgres из services
с накаткой goose-миграций, фронтовые lint/typecheck/test.

## Конвенции

* SQL — только через pgx, параметризованный, без ORM. Запросы живут в `repository` и больше нигде.
* Ошибки оборачиваются `fmt.Errorf("...: %w", err)` на каждом слое; на границе repository →
  domain `pgx.ErrNoRows` конвертируется в доменные сентинелы, которые дальше проверяются через
  `errors.Is` в хендлерах. Не превращать любую ошибку репозитория в 404/500 без разбора.
* Доменные статусы — типизированные строковые константы, не голые `string`.
* `userID` кладётся мидлварью в `gin.Context` как `uuid.UUID`; доставать через
  `handlers.userIDFromContext(c)`; `:id` товара — через `handlers.itemIDFromParam(c)`.
* TypeScript: `any` запрещён. Состояние сервера — только через слой API-клиента; бизнес-логики
  состояний на фронте нет, `message`/`next_step` приходят с бэкенда.
* Миграции вперёд-совместимые; применённую миграцию не редактируем, добавляем новую
  (исключение — сиды `00004` на этапе разработки, после правки нужен `docker compose down -v`).

## Definition of Done

1. `golangci-lint run` и `go test -race -count=1 ./internal/...` (с живой БД) зелёные.
2. Есть тест на новую логику; для правок в очереди или правах — тест на конкурентность.
3. Если изменился контракт — обновлены «HTTP API» здесь и в README.
4. Нет мёртвого кода, закомментированных блоков и `TODO` без номера задачи.
5. PR прочитал человек. Автомерж запрещён, финальная ответственность на команде.

## Чего не делать

* Не переписывать архитектуру: слои в порядке, правки локализуются в своём слое.
* Не добавлять зависимости без обоснования в PR. Никаких брокеров, Redis, ORM, k8s.
* Не создавать абстракции «на будущее»: интерфейс вводится, когда есть вторая реализация или
  его требует тест.
* Не расширять скоуп. Вне MVP: реальная оплата, регистрация, списание стоков.
* Не писать код сразу: сначала план (режим планирования), ревью плана человеком, потом код.
