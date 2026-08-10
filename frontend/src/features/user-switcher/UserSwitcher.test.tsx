import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, describe, expect, it } from 'vitest';

import { DEMO_USERS, useSessionStore } from '@/shared/lib/session';

import { UserSwitcher } from './UserSwitcher';

function renderSwitcher() {
  const client = new QueryClient();
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <UserSwitcher />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

// Демо-аккаунты различаются именем: user_id выдаёт сервер при входе, клиент
// его не выбирает. Токены засеяны заранее — иначе переключение полезло бы за
// сессией по сети, а проверяем мы здесь не вход, а выбор активного имени.
const [anya, boris] = DEMO_USERS;

function seedAccounts() {
  useSessionStore.setState({
    activeName: anya.name,
    accounts: {
      [anya.name]: { userId: 'user-anya', token: 'token-anya' },
      [boris.name]: { userId: 'user-boris', token: 'token-boris' },
    },
  });
}

afterEach(() => {
  seedAccounts();
});

describe('UserSwitcher', () => {
  it('меняет активного демо-пользователя при выборе', () => {
    seedAccounts();
    renderSwitcher();

    expect(useSessionStore.getState().activeName).toBe(anya.name);

    fireEvent.click(screen.getByRole('button', { name: /Борис/ }));

    expect(useSessionStore.getState().activeName).toBe(boris.name);
  });
});
