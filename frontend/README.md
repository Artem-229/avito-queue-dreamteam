# Frontend — Авито Очередь

React 18 + TypeScript + Vite. UI на своих компонентах и дизайн-токенах в стиле Авито.

## Скрипты

```bash
npm install       # установка зависимостей
npm run dev       # дев-сервер на http://localhost:5173 (проксирует /api на :8080)
npm run build     # прод-сборка в dist/
npm run preview   # предпросмотр прод-сборки
npm run lint      # ESLint
npm run lint:css  # Stylelint
npm run format    # Prettier --write
npm run test      # Vitest
npm run typecheck # tsc без эмита
```

## Стек

- **Сборка:** Vite 6
- **Роутинг:** React Router v6
- **Серверный стейт:** TanStack Query v5
- **Клиентский стейт:** Zustand
- **Стили:** CSS Modules + дизайн-токены (`src/app/styles/tokens.css`)
- **Линтеры:** ESLint (flat config, type-checked) + Prettier + Stylelint
- **Тесты:** Vitest + React Testing Library

## Структура (Feature-Sliced, упрощённая)

```
src/
  app/      # провайдеры, роутер, глобальные стили и токены
  pages/    # страницы-экраны
  widgets/  # составные блоки (layout, header)
  features/ # пользовательские сценарии (симулятор, метрики, переключатель демо-аккаунтов, ИИ-рекомендации)
  entities/ # доменные сущности item / queue-entry
  shared/   # переиспользуемый UI и утилиты
  test/     # настройка тестового окружения
```

Направление импортов: `pages → widgets → features → entities → shared`.

## Прокси и API

Фронт всегда работает против реального бэкенда — моков и второй реализации
правил очереди на клиенте нет, единственный источник истины по состояниям —
конверт статуса из API. Дев-сервер проксирует `/api` на
`http://localhost:8080` (бэкенд Go), в проде тем же занимается nginx
(`nginx.conf`).

Статус очереди поллится раз в секунду: этот же запрос служит триггером
ленивого продвижения очереди на бэкенде.

## ИИ-рекомендации (GigaChat)

Прокси к GigaChat реализован плагином dev-сервера Vite
(`gigachat.plugin.ts`), ключ задаётся в `.env.local` — см. `.env.example`.
Фича работает только под `npm run dev`; в прод-сборке ручки
`/ai/recommendation` нет, блок рекомендаций молча не показывается.

## Docker

Multi-stage сборка (`Dockerfile`): Node собирает статику, nginx её раздаёт и
проксирует `/api` на сервис `app`. Поднимается из корневого `docker-compose`.
