# AGENTS.md

Справочник по проекту для агентов и разработчиков, работающих в этом репозитории.

## О проекте

MVP сервиса пользовательской очереди на дефицитные товары (Avito Start, кейс 2, команда "Dream Team").
Пользователь встаёт в очередь перед чекаутом и получает временное персональное право на покупку —
система гарантирует, что одна единица товара не уйдёт двум покупателям одновременно.

## Стек

- **Backend:** Go 1.25, gin (HTTP-роутер), pgx/v5 (Postgres-драйвер), goose (миграции, применяются
  отдельным контейнером, не самим приложением), viper (конфиг)
- **Frontend:** React + TypeScript + Vite (директория `frontend/`, feature-sliced структура)
- **БД:** PostgreSQL 17
- **Запуск:** Docker Compose (postgres + goose-migrator + app)

## Структура backend

```
/cmd                            — точка входа (main.go)
/internal
  /app                          — сборка зависимостей (App, Repositories, Services), старт/остановка
  /config                       — чтение configs/configuration.yaml через viper
  /domain                       — доменные типы: CatalogItem, PurchaseRight, QueueService, сентинел-ошибки
  /infra/http/rest              — HTTP-слой на gin: server.go, router.go, handlers/, middlewares/
  /infra/postgres/repository    — репозитории для Catalog, QueueService, PurchaseRight (pgx)
  /services                     — сервисный слой: CatalogService, QueueService (Reconcile-логика), PurchaseRight
/migrations                     — goose-миграции (нумерация 00001, 00002, ...)
/configs                        — configuration.yaml
/docs                           — REQUIREMENTS.md
/frontend                       — отдельное SPA (Vite/React), общается с backend через /api/v1
```

Слои единообразны: `internal/infra/postgres/repository` → `internal/services` → `internal/infra/http/rest/handlers`.
Интерфейсы репозиториев объявлены на стороне потребителя (в `internal/services`), не в самом репозитории.

## Домен

- **CatalogItem** — товар: `id, name, price, total_stock, created_at, deleted_at`.
- **PurchaseRight** — персональное временное право на покупку одной единицы товара:
  `id, user_id, item_id, status(granted|used|cancelled|expired), created_at, expires_at`.
- **QueueService** (таблица `queue`) — запись в очереди пользователя на товар:
  `id, user_id, item_id, status(waiting|granted|sold_out|cancelled|expired|purchased), created_at, deleted_at`.

### Бизнес-логика очереди (`services.QueueService`)

- `Entry(userID, itemID)` — добавляет пользователя в очередь (status = waiting) и сразу вызывает `Reconcile`.
- `Reconcile(itemID)` — сдвиг очереди, единая точка изменения статусов, выполняется в транзакции:
  1. Просрочка старых `purchase_rights` (`ExpireOld`) → синхронный сброс статусов очереди (`MarkRecordsExpired`).
  2. Подсчёт активных прав (`CountActive`) и остатка стока (`CatalogRepo.GetItemByID`) → сколько прав ещё можно выдать.
  3. Выдача прав первым в очереди (`GetWaiting` + `PurchaseRightRepo.Create` + `QueueRepo.UpdateStatus`).

`services.PurchaseRight.Buy(userID, itemID)` — фиксирует покупку: находит `granted`-право пользователя
на товар и переводит его в `used`. Если права нет или оно не `granted` — `domain.ErrNoPurchaseRight`.

### HTTP-роуты

```
GET  /health
GET  /api/v1/catalog
GET  /api/v1/catalog/:id
POST /api/v1/catalog/:id/queue   — встать в очередь на товар
POST /api/v1/catalog/:id/buy     — купить (использовать granted-право)
```

Авторизация — `middlewares.TestAuthMiddleware`: читает заголовок `X-User-Id` (UUID), при отсутствии
подставляет нулевой UUID. Это временная заглушка для разработки, не боевая авторизация.

## Как запустить

```bash
docker compose up -d          # postgres + автоприменение миграций (сервис migrator) + сборка/запуск app
curl http://localhost:8080/health
```

Локальный `go run ./cmd` без Docker сейчас не заработает — `configs/configuration.yaml` хардкодит
`database.host: postgres` (имя сервиса в docker-сети), а `viper` не читает env-переменные из
`docker-compose.yaml` (`DB_HOST` и т.п. туда не пробрасываются). Известное ограничение, не чинить
между делом — если понадобится локальный запуск вне докера, нужно осознанно добавить
`viper.AutomaticEnv()`/`BindEnv` и поправить конфиг.

Миграции лежат в `/migrations`, применяются `goose`-контейнером при `docker compose up`
(см. сервис `migrator` в `docker-compose.yaml`), не самим приложением при старте.

## Тесты и линтеры

```bash
go build ./...
go vet ./...
golangci-lint run              # конфиг — .golangci.yaml (govet, staticcheck, unused, errcheck,
                                # rowserrcheck, sqlclosecheck, noctx, ineffassign; форматтер — goimports)
```

Frontend (`cd frontend`):

```bash
npm install
npm run dev
npm run test
npm run lint
```

## Конвенции

- Все SQL-запросы — через pgx, без ORM.
- Ошибки оборачиваются через `fmt.Errorf("...: %w", err)` на каждом слое; на границе repository → domain
  ошибки `pgx.ErrNoRows` конвертируются в доменные сентинелы (`domain.ErrNoItemFound`,
  `domain.ErrNoPurchaseRight`, ...), которые дальше проверяются через `errors.Is` в HTTP-хендлерах для
  выбора статус-кода — не превращайте любую ошибку репозитория в 404/500 без разбора.
- HTTP-хендлеры не должны напрямую обращаться к репозиториям — только через `internal/services`.
- Доменные статусы (`PurchaseRightStatus`, `QueueStatus` и т.п.) — типизированные строковые константы,
  не голые `string`.
- `userID` в `gin.Context` кладётся мидлварью уже как `uuid.UUID` (не строка) — доставать через
  `handlers.userIDFromContext(c)`, не парсить/приводить типы заново в каждом хендлере.
