import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, describe, expect, it } from 'vitest';

import { useSessionStore } from '@/shared/lib/session';

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

afterEach(() => {
  useSessionStore.setState({ userId: 'user-1' });
});

describe('UserSwitcher', () => {
  it('меняет текущего пользователя при выборе', () => {
    renderSwitcher();

    expect(useSessionStore.getState().userId).toBe('user-1');

    fireEvent.click(screen.getByRole('button', { name: /Борис/ }));

    expect(useSessionStore.getState().userId).toBe('user-2');
  });
});
