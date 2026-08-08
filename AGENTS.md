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

### HTTP API-контракт

Авторизация — `middlewares.TestAuthMiddleware`: все `/api/v1/*`-ручки читают заголовок
`X-User-Id` (UUID). При отсутствии заголовка подставляется нулевой UUID
(`00000000-0000-0000-0000-000000000000`); если заголовок есть, но не парсится как UUID —
`400 {"error":"invalid X-User-Id header"}`. Это временная заглушка для разработки, не
боевая авторизация — таблицы пользователей нет, любой валидный UUID принимается как есть.

Все таймстемпы — RFC3339 (`time.Time` в JSON), все id — UUID-строки.

#### `GET /health`

`200 {"status":"ok"}`. Без авторизации.

#### `GET /api/v1/catalog`

Список товаров. `200`, тело — массив (`[]`, если товаров нет):

```json
[
  {
    "id": "b00c13dd-9320-49dd-995b-e5a757a0d439",
    "name": "Бюст Дзержинского",
    "price": 500,
    "total_stock": 150,
    "category": "Коллекционные фигуры",
    "seller_name": "Ретро Сувениры",
    "created_at": "2026-08-08T20:08:05.877491Z"
  }
]
```

#### `GET /api/v1/catalog/:id`

Один товар. `200` — объект как выше (без массива).
`400 {"error":"failed to parse id"}` — `:id` не UUID.
`404 {"error":"item not found"}` — валидный UUID, товара с таким id нет.

#### `GET /api/v1/catalog/:id/similar`

Похожие товары — те же поля, что и в списке, товары той же `category`, сам товар
исключён, лимит 20. `200`, тело — массив (`[]`, если похожих нет). Коды ошибок — как у
`GET /api/v1/catalog/:id` (сначала ищет исходный товар по `:id`, чтобы узнать его
category).

#### `POST /api/v1/catalog/:id/queue` — встать в очередь на товар

Идемпотентно: повторный вызов тем же `X-User-Id` для того же товара, пока участие ещё
активно (`waiting`/`granted`), не создаёт вторую запись.

`202 {"message":"joined queue","item_id":"<uuid>"}` — встал в очередь. Внутри синхронно
отрабатывает `Reconcile`, так что если слот свободен — участник может сразу оказаться в
`granted`, а не в `waiting` (проверить статус можно только через `GET /checkout/:itemID`,
см. ниже — отдельной ручки со статусом+позицией сейчас нет, см. «Известные ограничения»).
`400 {"error":"failed to parse id"}` — `:id` не UUID.
`404 {"error":"item not found"}` — товара нет.
`409 {"error":"already in queue"}` — уже стоит в очереди / уже есть granted-право на этот товар.

#### `POST /api/v1/catalog/:id/buy` — купить (использовать granted-право)

Работает только если у `X-User-Id` есть активное `granted`-право на этот товар — прямого
перехода к покупке без очереди нет. Атомарно помечает право `used` и переводит участие в
очереди в `purchased`.

`200 {"message":"purchase completed","item_id":"<uuid>"}`.
`400 {"error":"failed to parse id"}` — `:id` не UUID.
`403 {"error":"no granted purchase right for this item"}` — права нет вообще, оно не
`granted`, уже использовано, либо истекло.
`409 {"error":"item is sold out"}` — весь сток товара уже распродан (`used >= total_stock`) —
именно эта причина отказа, а не общая 403, если применимо.

#### `GET /api/v1/checkout/:itemID` — проверить право перед переходом к оплате

Гейт «перед покупкой» (CHK-01/CHK-02): не мутирует, только читает текущее состояние права.

**Важно:** эта ручка всегда отвечает `HTTP 200`, даже когда `allowed:false` — смотреть
нужно на поле `allowed`, а не на код ответа.

```json
{
  "purchase_id": "bd3cc16c-a07a-429c-ab39-51f87beffb1c",
  "allowed": true,
  "expires_at": "2026-08-08T20:16:12.701794Z",
  "reason": ""
}
```

Если `allowed:false` — `purchase_id` нулевой UUID, `expires_at` нулевое время
(`0001-01-01T00:00:00Z`), `reason` — одна из:
- `"Нет права на покупку этого товара"` — права на этот товар вообще нет (не вставал в
  очередь / не дошла очередь).
- `"Право на покупку товара неактивно"` — право есть, но не `granted` (истекло/использовано/
  отменено).

`400 {"error":"Некорректные данные товара"}` — `:itemID` не UUID.

#### `POST /api/v1/checkout/:itemID/pay` — оплатить

Тот же эффект, что и `POST /catalog/:id/buy` (внутри дёргает тот же `PurchaseRight.Buy`,
одна точка записи), просто как второй шаг после `GET /checkout/:itemID`.

`200 {"success":"Покупка завершена"}`.
`409 {"error":"<reason>"}` — `reason`: `"право на покупку недействительно, истекло или уже
использовано"` либо `"товар полностью распродан"`.
`400 {"error":"Некорректные данные товара"}` — `:itemID` не UUID.
`500 {"error":"Внутренняя ошибка сервера"}` — непредвиденная ошибка.

#### Статусы

- `queue.status`: `waiting | granted | sold_out | cancelled | expired | purchased`
  (`cancelled` зарезервирован под выход из очереди — ручки для этого пока нет).
- `purchase_rights.status`: `granted | used | cancelled | expired`.

#### Известные ограничения текущего контракта (важно для фронта)

- **Нет ручки «мой статус в очереди + позиция»**. Позиция в очереди считается на уровне
  `services.QueueService.GetStatus`/`GetPosition`, но не подключена к HTTP. Единственный
  способ фронту узнать, есть ли granted-право — `GET /checkout/:itemID`; отличить `waiting`
  от «никогда не вставал в очередь» или от `sold_out`/`expired` через API сейчас нельзя.
- **Нет ручки выхода из очереди** (`cancelled`) — S-06, не реализовано.
- **Нет SSE и ETA** — продвижение синхронное (по запросу), обновление статуса на фронте
  только через поллинг.
- `POST /api/v1/catalog/:id/queue` и `POST /api/v1/catalog/:id/buy` не возвращают текущий
  статус/позицию в теле ответа — только `message`/`item_id`.

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
