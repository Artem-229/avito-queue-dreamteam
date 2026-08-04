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
  features/ # пользовательские сценарии (появятся на следующих этапах)
  entities/ # доменные сущности item / queue-entry
  shared/   # переиспользуемый UI и утилиты
  test/     # настройка тестового окружения
```

Направление импортов: `pages → widgets → features → entities → shared`.

## Прокси и API

Дев-сервер проксирует `/api` на `http://localhost:8080` (бэкенд Go). В проде тем же
занимается nginx (`nginx.conf`), там же отключена буферизация для SSE-потока очереди.

## Docker

Multi-stage сборка (`Dockerfile`): Node собирает статику, nginx её раздаёт и
проксирует `/api` на сервис `backend`. Поднимается из корневого `docker-compose`.
